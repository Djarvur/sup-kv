# sup-kv API

sup-kv exposes one gRPC service, `supkvapi.v1.SupKVService`, with seven RPCs:
`Get`, `Set`, `Delete`, `List`, `Watch` (the only server-streaming method),
`SnapshotCreate` and `SnapshotLoad`. Every state-carrying message travels as a
full record — key, 16-byte RFC 9562 record version, value bytes and the
`deleted` tombstone flag — so a client never needs a second round trip to act
on a change.

The client surface carries the same trust model as the replication wire: no
authentication and no authorization. Any client that reaches a node's gRPC
address can read and write everything, and the default `--grpc-addr` binds
loopback only. Bind it wider only on a segment where every reachable client is
trusted; see [docs/OPERATIONS.md](OPERATIONS.md) for the full trust model,
scale envelope and operational contract. This document is the API contract
itself: what each RPC promises, what it deliberately does not promise, and
which errors a client must expect.

## Public proto

The client-facing schema is public and versioned in this repository:

| What | Where |
|------|-------|
| Proto source | `api/supkvapi/v1/supkvapi.proto` |
| Proto package | `supkvapi.v1` |
| Go import path | `github.com/Djarvur/sup-kv/api/supkvapi/v1` |
| Go package name | `supkvapi` |
| Service | `SupKVService` |

```go
import supkvapi "github.com/Djarvur/sup-kv/api/supkvapi/v1"
```

The generated Go files (`supkvapi.pb.go`, `supkvapi_grpc.pb.go`) are committed
beside the proto, so consuming the API needs no code generation. When the
schema changes, regenerate with `make proto` — one `buf generate` pass over
`api/` with the go.mod-pinned buf tool — and commit the result; never hand-edit
generated code. `supkvapi.proto` is the source of truth: field names and
numbers are pinned once clients exist, and changes are deliberate (the buf
lint/breaking gate runs in CI).

The service itself is discoverable at runtime: gRPC server reflection is
always on, so `grpcurl` and other reflection clients need no proto files —
`grpcurl -plaintext <addr> describe supkvapi.v1.SupKVService` prints the whole
surface. The runnable Go client example at `examples/client` is a separate
module built in CI, and is the reference for a consumer's own client.

The seven RPCs are `Get`, `Set`, `Delete`, `List`, `Watch`, `SnapshotCreate`
and `SnapshotLoad`; each is documented in its own section below.

## wait_for_replication

`Set` and `Delete` accept `wait_for_replication` and `wait_timeout_ms`. The
flag asks the server to wait after the confirmed write for replication Acks
and to decorate the response with the aggregate report; it never changes the
write outcome. The semantics are pinned to what the server can actually
observe:

- **The write-time peer set is the promise set.** The peer ids known at the
  moment the write is accepted are captured once, immediately before the
  engine write. The same captured set feeds the replication notification and
  the wait report, so a peer discovered during the wait never joins the
  awaited set, never moves `total_peers`, and an Ack from a node outside the
  set can never make `acked` true.
- **Zero known peers answers immediately.** With zero known peers captured at write
  acceptance — a single node, or a node that knows no peers yet — the report
  is `acked=false`, `acked_count=0`, `total_peers=0`, returned without
  waiting. The write is still engine-confirmed on disk; nothing was awaited
  and nothing was acknowledged, so claiming success would be a lie. The same
  answer is returned when the node has no wait seam configured.
- **A displaced version resolves promptly through `Superseded`.** If a newer
  write supersedes the version before its Acks arrive, the existing
  `Superseded` result resolves the wait as an honest false instead of idling
  to the caller's timeout; the design target for that fast path is 20 ms, and
  the wait is allowed to answer earlier than its deadline.
- **The wait is caller-bounded.** A positive `wait_timeout_ms` wins over the
  operator-tunable default (1 s in the shipped binary). A request value above
  ten times that default is clamped to the ceiling, and a value large enough
  to overflow the millisecond-to-nanosecond conversion is clamped before the
  conversion. `wait_timeout_ms` of zero or below means "use the operator
  default", never "wait forever".
- **The report is aggregate.** `acked_count` counts Acks from peers in the
  write-time set and is clamped to `total_peers`; `total_peers` is the
  captured set's size, never a live peer count. There is no per-peer
  breakdown. A version evicted by a peer timeout, a context cancellation and
  an expired deadline all resolve as `acked=false` with the count observed so
  far.
- **`acked=false` is honest, not a failure.** The peers that did not Ack
  within the bound did not confirm; the confirmed write still stands on the
  writer and keeps replicating. Without the flag, every report field keeps
  its zero value.

## List pagination

`List` returns the live (non-tombstoned) keys under a prefix from one live
key-to-version view. Both modes order keys byte-wise lexicographic ascending,
and tombstones are never listed: `List` reflects live state, not history.
`page_size` selects the mode.

- **`page_size` unset or `0` — the legacy listing.** Every matching live key
  in one response, keys only: `records` and `next_page_token` stay empty.
  This is the frozen single-call contract existing clients and the R-01 stand
  poll depend on. The mode copies no value bytes, so its response cost is
  independent of value sizes — but it is sized by key count. A key space whose
  encoded listing would exceed the transport send ceiling must use
  `page_size > 0` rather than one oversized response. A non-empty `page_token`
  in this mode is rejected with `InvalidArgument` (`invalid page size`): the
  legacy listing has no page to continue.
- **`page_size > 0` — a bounded keyset page.** The response carries the page's
  records with their values, with `keys` aligned to `records` entry by entry.
  Values ride only for the entries of the page, so a page's cost is
  independent of the store's total value bytes. `page_size` above the `256`
  maximum clamps to `256`; a negative `page_size` is rejected with
  `InvalidArgument` (`invalid page size`). `page_size` is an upper bound, not
  a promise: a page may come back shorter because the listing ended, because
  the 6 MiB serialized-byte budget was reached, or because keys raced away
  between the view and their lookup.

**The position token.** `next_page_token` is the opaque raw-URL-safe base64 of
the last examined key; send it back as `page_token` to resume strictly after
that position. It is a position, not a snapshot: keys added or deleted between
pages may appear or disappear, no page is an isolated view, and a resumed walk
neither replays nor skips a key by construction. An undecodable token, or one
whose decoded key exceeds the 1 024-byte key ceiling, is rejected with
`InvalidArgument` (`invalid page token`). An **empty token** is the end signal:
it means no matching key follows the last examined one, and it is the only
end-of-list signal (an empty final page is legal). The empty `page_token`
starts at the first key.

**The 6 MiB byte budget.** One page's summed serialized entries stay at or
below a 6 MiB budget, so a page's cost does not grow with the store's total
value bytes. The walk stops before an entry that would exceed the budget and
leaves it as the next page's first entry; the first entry is always included,
so a non-terminal page is never empty and the walk always progresses. The
budget is an upper bound on the serialized entries, which is what lets the
transport send ceiling cover any page the server produces.

## Errors

The table below covers every error the RPC handlers produce. The mapper is
total: an error that matches no row still answers a fixed status, so no
internal detail can escape through the response path. The table is the
contract, and the row order is part of it — the first match wins.

| Code | Message | Raised when |
|------|---------|-------------|
| `NotFound` | `key not found` | a `Get` of an absent or tombstoned key, or a `Delete` with no live record |
| `AlreadyExists` | `key already exists` | `create_if_absent` for any existing record, tombstone included, or for a key a peer's complete listing named |
| `InvalidArgument` | `value exceeds the maximum allowed size` | a write value over the effective `--value-max-bytes` limit |
| `Unavailable` | `dump failed; write not confirmed` | the engine installed the write but its covering dump failed |
| `ResourceExhausted` | `disk full` | a dump failure classified as disk-full (a snapshot create surfaces the class directly) |
| `PermissionDenied` | `permission denied` | a dump failure classified as permission, or `fs.ErrPermission` from the filesystem |
| `Unavailable` | `write not confirmed before the confirmation timeout` | the covering dump outlived the confirmation timeout |
| `Unavailable` | `persist engine closed` | the engine is closed (process shutdown) |
| `InvalidArgument` | `invalid snapshot name` | a snapshot name outside the adapter's closed charset |
| `FailedPrecondition` | `no dump on disk to snapshot` | a snapshot operation needs a dump that does not exist |
| `NotFound` | `snapshot not found` | the named snapshot is absent from disk |
| `Canceled` | `context canceled` | the caller cancelled the RPC |
| `DeadlineExceeded` | `context deadline exceeded` | the caller's deadline expired |
| `Internal` | `internal error` | the totality fallback: anything unclassified |

The ordering rules that make the table total and honest:

- a pre-built status error passes through unchanged, so a handler's own
  validation status is never re-mapped;
- the dump-failure row precedes the port failure classes: a write whose dump
  failed answers `Unavailable` with that constant, while `SnapshotCreate`
  surfaces the adapter's own class as `ResourceExhausted` or `PermissionDenied`;
- the no-dump row precedes the filesystem rows, so the adapter's sentinel
  survives instead of collapsing into `fs.ErrNotExist`;
- `context.Canceled` and `context.DeadlineExceeded` are matched through their
  sentinels and answered via `status.FromContextError`, so a wrapped
  cancellation still reports `Canceled`, never `Unknown`;
- every other error takes the fallback: the internal detail is logged exactly
  once as a static error line with the cause as an attribute, and the response
  carries the constant `internal error`. Internal detail is logged and never
  returned; no server path, errno or configured filename crosses the wire.

Handler validation adds these constant messages to the same codes (they are
not mapper rows, but they are part of the client-visible contract):

| Code | Message | Raised when |
|------|---------|-------------|
| `InvalidArgument` | `key must not be empty` | `Get`, `Set` or `Delete` with an empty key |
| `InvalidArgument` | `key length N exceeds the maximum of 1024 bytes (the limit is measured in bytes)` | a write key over the 1 024-byte ceiling |
| `Unavailable` | `startup gate: waiting for first sync round or startup_gate` | any data RPC while the startup gate is closed |
| `Unavailable` | `startup gate: key is listed by a peer but its value has not arrived` | a `Get` of a listed-but-unarrived key (rule 14.5(6)) |
| `ResourceExhausted` | `watch subscriber overwhelmed; resubscribe` | a slow Watch subscriber is shed |
| `Unimplemented` | `watch requires a hub; construct the server through NewServer` | a server constructed without a Watch hub |

## Watch

`Watch` subscribes to changes under a prefix (the empty prefix matches every
key) and is the only server-streaming RPC. The stream is the snapshot followed
by the live change feed, with these pinned semantics:

- **The snapshot is the current in-memory state** at subscribe time — what
  `Get` sees — delivered in sorted key order with tombstones included as
  `deleted=true`.
- **The live feed is the durable change feed.** A change is released to
  subscribers only after the dump confirmation that made it durable. A write
  whose covering dump failed stays pending; the next successful dump covering
  its install generation releases it.
- **Delivery is exactly once.** A released event leaves the pending queue, and
  a repeated or lower confirmation delivers nothing, so one confirmed write
  yields one event.
- **Deletions are events.** A tombstone arrives as a record with
  `deleted=true`, in the same shape as any other change.
- **Ordering is install order, which is version order per key.** A key's events
  arrive in non-decreasing version order: a strictly greater version installs
  normally, and an equal-version tie-break winner can install once. An event
  already contained in the subscriber's snapshot is suppressed.
- **The snapshot and the live feed have deliberately different durability.**
  The snapshot is current memory and may include a write whose dump has not
  confirmed yet; the live feed never does. That asymmetry is the contract, not
  a defect: it is what makes the snapshot cheap and the change feed honest.
- **The stream carries no origin field.** Local and replicated changes are
  indistinguishable.
- **A slow subscriber is disconnected, never silently dropped.** Each
  subscriber buffers 256 undelivered events; a full buffer ends that stream
  with `ResourceExhausted` (`watch subscriber overwhelmed; resubscribe`). A hub
  queue at its 4 096-event bound sheds every subscriber the same way. In both
  cases the client resubscribes and re-snapshots, losing nothing.
- **Shutdown ends every stream with `Unavailable`.** The status message is
  `watch hub is shutting down` — never `OK` — and a subscriber that arrives
  after shutdown began gets the same status.
- **Watch is gated like every read.** While the startup gate is closed, the
  stream is refused with `Unavailable`.

## Snapshots

`SnapshotCreate` forces the current dump to complete — every confirmed write is
on disk — then byte-copies that finished dump under the requested name. The
copy is byte-identical to the main dump at creation time and lands in the
snapshot directory beside the dump. A flush failure aborts with nothing
copied: state that is not confirmed on disk must never become a recovery point.
The name is validated against a closed charset before any filesystem touch; an
invalid name is `InvalidArgument`. A create is a write and is gated like one.

`SnapshotLoad` is a full rollback (decision 14.1, ADR 0001), not a merge:

- The snapshot's keys replay as ordinary writes with a **fresh version**: a
  live snapshot record becomes a fresh `Set` that beats any current tombstone,
  and a snapshot tombstone becomes a fresh `Delete` that beats any current live
  value. The snapshot's own versions are never re-installed; fresh versions are
  what make the load replicate and win by normal last-write-wins.
- Every current live key **absent from the snapshot** is deleted with a fresh
  tombstone, so the snapshot's key set becomes the store's key set and the
  rollback tombstones join the result.
- A key already tombstoned locally and absent from the snapshot is already
  rolled back: **no version is burned** on it and no record is produced.
- The replay and the rollback walk one deterministic sorted-key order, and the
  call returns only after one final dump covering the whole replay.
- **The result is broadcast.** Every record the replay installed — rollback
  tombstones included — reaches peers through the ordinary paced replication
  path, and Watch subscribers see the same installs as a write burst. The
  broadcast happens even when the final confirming dump failed (the replayed
  state is in memory), but never on a closed engine.
- **The rollback is not linearizable against concurrent writes.** A write that
  lands after the rollback set is computed is ordinary last-write-wins: it is
  neither suppressed nor rolled back. `SnapshotLoad` is a full rollback, not
  snapshot isolation.

## Transport and health

Every gRPC server setting is explicit; no message size or keepalive cadence is
inherited from a gRPC default.

| Setting | Value | Contract |
|---------|-------|----------|
| Server keepalive ping | 2 m idle | notice a dead consumer quickly on a LAN without background chatter |
| Keepalive answer timeout | 20 s | a silent peer is closed instead of holding a stream indefinitely |
| Minimum client ping interval | 30 s | a consumer library may run its own 30 s client keepalive without tripping enforcement |
| Permit without stream | `true` | a consumer may hold an idle connection between writes |
| `MaxRecvBytes` | 1 024-byte key ceiling + the effective value limit + 1 382-byte envelope — 63 846 bytes at the default limit (66 814 at the 64 408-byte datagram ceiling) | covers the largest valid `SetRequest`; an over-limit request is rejected at the transport with `ResourceExhausted`, before allocation-heavy handling |
| `MaxSendBytes` | 6 MiB page budget + 1 382-byte envelope | covers any `List` page the server can produce, as well as every smaller unary response |

An over-limit request is a transport verdict (`ResourceExhausted`), a different
answer from the handler's `InvalidArgument` value-limit rejection: the layer
that spoke is visible in the code.

Health uses the standard `grpc.health.v1.Health` endpoint and is bound 1:1 to
the startup gate. It reports `NOT_SERVING` for the empty (whole-server) service
name from construction until the startup gate opens, then `SERVING`. Every data
RPC answers `Unavailable` while the gate is closed. When shutdown begins the
status flips to `NOT_SERVING` and never returns to SERVING: the transition is
one-way (the standard health server's `Shutdown` is called; `Resume` never is),
so a late gate resolver cannot resurrect a serving verdict.

## Consumer library territory: last-read-value

The "last read value" file/cache pattern — a consumer keeps a local fallback
copy of the last value it read, so its own service keeps answering through a
sup-kv outage — is consumer-library territory. It is not a server feature and
this binary does not ship it: no node writes a client-side cache and no RPC
promises one. This is the accepted placement of ADR 0007 (review decision
14.7); the pattern is documented here and implemented with the consumer's
library, not in sup-kv.

## What Phase 9 adds

This page is the phase-owned half of the API contract. The rest belongs to
Phase 9's DOCS-05 work and is deliberately not claimed complete here: the full
per-RPC failure table (every RPC against every error, rather than this page's
mapper-level table), the write-latency model, and the key-prefix convention.
Treat those sections as pending until Phase 9 lands them; this document claims
only the sections it states.
