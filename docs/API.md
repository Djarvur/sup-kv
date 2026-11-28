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
whose decoded key exceeds the dump format's maximum record line — the largest
key a readable dump can carry — is rejected with `InvalidArgument`
(`invalid page token`). A key above the 1 024-byte write ceiling is a legal
position: the store keeps serving pre-upgrade records under longer keys, so
the token domain is the format's, not the write ceiling, and the walk resumes
past such a key. An **empty token** is the end signal: it means no matching
key follows the last examined one, and it is the only end-of-list signal (an
empty final page is legal). The empty `page_token` starts at the first key.

**The 6 MiB byte budget.** One page's summed serialized entries stay at or
below a 6 MiB budget, so a page's cost does not grow with the store's total
value bytes. The walk stops before an entry that would exceed the budget and
leaves it as the next page's first entry; the first entry is always included,
so a non-terminal page is never empty and the walk always progresses. The
budget is an upper bound on the serialized entries, which is what lets the
transport send ceiling cover any page the server produces.

## Errors

The table below covers every mapper code and the handler validation
constants; each section of the [RPC reference](#rpc-reference) states which
of these codes that RPC's inputs can produce. The mapper is
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
| `DataLoss` | `snapshot does not decode cleanly` | a `SnapshotLoad` of a snapshot readable at generation level but with at least one record line the decoder discarded (checksum, version, value-limit or malformed class); nothing is applied |
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
  normally, and an equal-version tie-break winner can install once — an
  equal-version content change is a real change, not a duplicate. An event
  already contained in the subscriber's snapshot — the same version and the
  same content — is suppressed; the same version with different content is
  delivered.
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
- **A partially readable snapshot is refused, not partially applied.** If
  decoding the snapshot discards any record line — a checksum mismatch, an
  unreadable version, a malformed line, or a value over the node's current
  value limit — `SnapshotLoad` fails with `DataLoss`
  (`snapshot does not decode cleanly`) before any change: no record is
  replayed, no live key is deleted, nothing is broadcast, and the in-memory
  state and on-disk generations stay untouched. Full-rollback deletion
  therefore runs only on a cleanly decoded snapshot; a partly damaged
  recovery artifact can never be turned into cluster-wide deletes.
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

The message-size ceilings and the keepalive cadence are pinned explicitly; the
remaining transport knobs (header-list size, connection timeout, stream and
window sizes, the concurrent-stream cap) keep gRPC's defaults.

| Setting | Value | Contract |
|---------|-------|----------|
| Server keepalive ping | 2 m idle | notice a dead consumer quickly on a LAN without background chatter |
| Keepalive answer timeout | 20 s | a silent peer is closed instead of holding a stream indefinitely |
| Minimum client ping interval | 30 s | a consumer library may run its own 30 s client keepalive without tripping enforcement |
| Permit without stream | `true` | a consumer may hold an idle connection between writes |
| `MaxRecvBytes` | the envelope ceiling (maximum page token plus field framing) + a 1 024-byte key + the effective value limit — 187 472 bytes at the default limit (190 440 at the 64 408-byte datagram ceiling) | covers the largest valid request of either shape: a maximum `SetRequest`, or a `ListRequest` carrying a maximum page token (the token domain is the dump format's line ceiling, so a pre-upgrade longer key stays walkable); a request above the ceiling is rejected at the transport with `ResourceExhausted`, before allocation-heavy handling |
| `MaxSendBytes` | 6 MiB page budget + 125 008-byte envelope | covers any `List` page the server can produce — maximum-length page token included — as well as every smaller unary response. A client that requests a maximum-size page must raise its own gRPC receive limit: the client default is 4 MiB, below a maximum page. |

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

## RPC reference

The sections below appear in the schema's own declaration order — the order
`supkvapi.v1.SupKVService` declares its rpcs in
`api/supkvapi/v1/supkvapi.proto` — and cover every RPC the schema declares,
so the reference and the proto cannot silently diverge (the order is
machine-checked). Each section states the
RPC's semantics and an error table of exactly the codes that RPC's inputs
can produce per the server's total mapping (see [Errors](#errors)). Three
facts hold for every RPC and are factored out here rather than repeated per
table:

- **The startup gate comes first.** While it is closed, every data RPC —
  reads and writes alike — answers `Unavailable`
  (`startup gate: waiting for first sync round or startup_gate`) before any
  validation, so even an empty key surfaces `Unavailable`, not
  `InvalidArgument`, until the gate opens; health reports `NOT_SERVING` for
  the same window. The tables below state what each RPC's inputs can produce
  once the gate is open.
- **A request over the transport receive ceiling is a transport verdict.**
  Any request larger than `MaxRecvBytes` is rejected with `ResourceExhausted`
  before the handler runs — a different answer from the handler's own
  `InvalidArgument` rejections, and the layer that spoke is visible in the
  code. The ceiling is documented once in
  [Transport and health](#transport-and-health).
- **Caller cancellation is transport behavior.** The caller's own context
  cancellation and deadline always end an RPC with `Canceled` or
  `DeadlineExceeded`, independent of the tables below.

### Get

`Get` returns the live record for one key — the full record (key, 16-byte
RFC 9562 `version`, `value` bytes, `deleted` flag) in one round trip. An
absent or tombstoned key is `NotFound`, never an empty success. `Get` does
not enforce the key-length ceiling: validation is write-side only, so
pre-upgrade records under longer keys stay readable.

A key a peer's complete listing named but whose value has not arrived
locally is its own answer: the key provably exists somewhere in the cluster,
so claiming `NotFound` would lie — see
[Consistency caveats](#consistency-caveats).

#### Errors

| Code | Message | Raised when |
|------|---------|-------------|
| `InvalidArgument` | `key must not be empty` | an empty key |
| `NotFound` | `key not found` | the key is absent or tombstoned locally |
| `Unavailable` | `startup gate: key is listed by a peer but its value has not arrived` | the key is listed by a peer but its value has not arrived (rule 14.5(6)) |
| `Internal` | `internal error` | the totality fallback: an unclassified engine error is logged once and answered with the constant — no internal detail crosses the wire |

### Set

`Set` writes `value` under `key` and returns only after the engine confirmed
the write on disk — a nil error means a dump containing the record
completed, so a confirmed write is never lost. The response carries the
stored record's 16-byte `version`. `create_if_absent` turns the write into
the create-if-absent variant: it refuses with `AlreadyExists` for any
existing record — tombstone included — and for a key a peer's complete
listing named, before any engine call (the peer already owns the key; a
local create would overwrite it).

`wait_for_replication` asks the server to wait after the confirmed write for
the replication Acks and to decorate the response with the aggregate report
(`acked`, `acked_count`, `total_peers`); it never changes the write outcome
and is bounded per request by `wait_timeout_ms` — the full contract is in
[wait_for_replication](#wait_for_replication), and the wait reports an
honest false rather than failing. A write the engine installed but whose
covering dump failed still replicates (the record is in memory and is
notified before the error) while the RPC answers `Unavailable`.

#### Errors

| Code | Message | Raised when |
|------|---------|-------------|
| `InvalidArgument` | `key must not be empty` | an empty key |
| `InvalidArgument` | `key length N exceeds the maximum of 1024 bytes (the limit is measured in bytes)` | a key over the 1 024-byte ceiling |
| `InvalidArgument` | `value exceeds the maximum allowed size` | a value over the effective `--value-max-bytes` limit |
| `AlreadyExists` | `key already exists` | `create_if_absent` for any existing record (tombstone included) or for a key a peer's complete listing named |
| `Unavailable` | `dump failed; write not confirmed` | the engine installed the write but its covering dump failed (the write stays in memory and replicates) |
| `Unavailable` | `write not confirmed before the confirmation timeout` | the covering dump outlived the confirmation timeout |
| `Unavailable` | `persist engine closed` | the engine is closed (process shutdown) |
| `Internal` | `internal error` | the totality fallback for an unclassified engine error |

### Delete

`Delete` writes a fresh-version tombstone for one key and returns only after
the engine confirmed the tombstone on disk — a delete is a write. The
response carries the tombstone's `version`. A missing or already-tombstoned
key surfaces `NotFound`: a delete of nothing happened, and no version is
burned. The wait fields carry the same meaning as on `Set`.

#### Errors

| Code | Message | Raised when |
|------|---------|-------------|
| `InvalidArgument` | `key must not be empty` | an empty key |
| `InvalidArgument` | `key length N exceeds the maximum of 1024 bytes (the limit is measured in bytes)` | a key over the 1 024-byte ceiling — a tombstone for an unwritable key would be its own unsendable record |
| `NotFound` | `key not found` | the key is absent or already tombstoned |
| `Unavailable` | `dump failed; write not confirmed` | the engine installed the tombstone but its covering dump failed (it stays in memory and replicates) |
| `Unavailable` | `write not confirmed before the confirmation timeout` | the covering dump outlived the confirmation timeout |
| `Unavailable` | `persist engine closed` | the engine is closed (process shutdown) |
| `Internal` | `internal error` | the totality fallback for an unclassified engine error |

### List

`List` returns the live (non-tombstoned) keys under `prefix` — the empty
prefix matches every key — in byte-wise lexicographic ascending order. The
two modes and the position token are specified in
[List pagination](#list-pagination); tombstones are never listed.

#### Errors

| Code | Message | Raised when |
|------|---------|-------------|
| `InvalidArgument` | `invalid page size` | a negative `page_size`, or a non-empty `page_token` without a positive `page_size` (the legacy listing has no page to continue) |
| `InvalidArgument` | `invalid page token` | an undecodable token, or one whose decoded key exceeds the dump format's record-line ceiling |

Past the gate, the only failures are the request's own: the listing has no
engine error path, so the table carries no `Internal` row — that absence is
deliberate, not an omission.

### Watch

`Watch` subscribes to changes under `prefix` (the empty prefix matches every
key) and is the only server-streaming RPC. The stream is the snapshot
followed by the live change feed; the durability mechanics, ordering and
buffer policy are specified in [Watch](#watch), and the API-level guarantee
in [Watch contract](#watch-contract).

#### Errors

A healthy stream does not fail: the only server-initiated way a live stream
ends is shutdown, with `Unavailable` — never `OK` — and there is no
`Internal` or other watch-failure path. A stream that ends without one of
the codes below ended because the client went away.

| Code | Message | Raised when |
|------|---------|-------------|
| `Unavailable` | `watch hub is shutting down` | the server is shutting down — every stream ends with this status, and a subscriber that arrives after shutdown began gets the same verdict |
| `ResourceExhausted` | `watch subscriber overwhelmed; resubscribe` | this subscriber's event buffer or the hub's pending bound overflowed; resubscribe and re-snapshot, losing nothing |
| `Unimplemented` | `watch requires a hub; construct the server through NewServer` | a server constructed without a Watch hub |

### SnapshotCreate

`SnapshotCreate` forces the current dump to complete — every confirmed write
is on disk — then byte-copies that finished dump under `name` (see
[Snapshots](#snapshots)). A create is a write: it is gated like one, and a
flush failure aborts with nothing copied.

#### Errors

| Code | Message | Raised when |
|------|---------|-------------|
| `InvalidArgument` | `invalid snapshot name` | a name outside the closed charset `[A-Za-z0-9._-]` or the 1..64 length bound — validated before any filesystem touch |
| `FailedPrecondition` | `no dump on disk to snapshot` | no main dump exists on disk to copy |
| `Unavailable` | `dump failed; write not confirmed` | the forced dump failed (nothing is copied) |
| `Unavailable` | `write not confirmed before the confirmation timeout` | the forced dump outlived the confirmation timeout |
| `Unavailable` | `persist engine closed` | the engine is closed (process shutdown) |
| `ResourceExhausted` | `disk full` | the copy's filesystem work was classified as disk-full |
| `PermissionDenied` | `permission denied` | the copy's filesystem work was classified as permission |
| `Internal` | `internal error` | the totality fallback for an unclassified filesystem fault |

### SnapshotLoad

`SnapshotLoad` replays the named snapshot as ordinary writes with fresh
versions — a full rollback, not a merge; the replay, the rollback walk and
the broadcast semantics are specified in [Snapshots](#snapshots). The call
returns only after one final dump covering the whole replay, and the result
replicates like any write burst.

#### Errors

| Code | Message | Raised when |
|------|---------|-------------|
| `InvalidArgument` | `invalid snapshot name` | a name outside the closed charset — validated before any filesystem touch |
| `NotFound` | `snapshot not found` | the named snapshot is absent from disk |
| `DataLoss` | `snapshot does not decode cleanly` | the snapshot is readable at generation level but at least one record line was discarded (checksum, version, value-limit or malformed class) — the load is refused before any change |
| `Unavailable` | `dump failed; write not confirmed` | the final confirming dump failed (the replayed state is in memory and still replicates) |
| `Unavailable` | `write not confirmed before the confirmation timeout` | the final dump outlived the confirmation timeout |
| `Unavailable` | `persist engine closed` | the engine is closed (process shutdown) |
| `PermissionDenied` | `permission denied` | the snapshot file is unreadable due to filesystem permissions |
| `Internal` | `internal error` | the totality fallback for an unclassified read fault |

## Watch contract

The [Watch](#watch) section specifies the mechanics; this is the API-level
guarantee an integrator builds on:

- **The cut is exact.** The stream is snapshot-then-live with one boundary:
  every key's state at subscribe time arrives in the snapshot, and every
  install after the cut arrives exactly once as a live event — nothing
  between the snapshot's keys and the first live event can be missed or
  doubled.
- **Local changes are announced only after they are durable.** A live event
  is released only once the dump confirmation that made the change durable
  has completed; a write whose dump failed stays pending until a covering
  dump succeeds. What the live feed announces is always on disk.
- **Pending delivery is bounded, and overload is loud.** The hub holds
  pending events within a fixed bound; past it — or past a subscriber's own
  event buffer — every affected stream ends with `ResourceExhausted`
  (`watch subscriber overwhelmed; resubscribe`) instead of a silent drop.
  The bounded pending queue is what keeps a slow subscriber from growing the
  hub's memory without limit, and a resubscribing client re-snapshots and
  loses nothing.
- **Shutdown ends every stream with `Unavailable`, never `OK`** — a dying
  node must not look like a clean end of the change feed — and a subscriber
  that arrives after shutdown began gets the same verdict.
- **Health never returns to SERVING.** Shutdown flips the health endpoint to
  `NOT_SERVING` one-way: a late gate resolver cannot resurrect a serving
  verdict, so an orchestrator acting on health never sends traffic to a
  dying node twice.

## Write-latency model

A client write returns only after a dump containing it completed, so write
latency is a property of the dump policy — the debounce window and the
full dump's duration — not of the RPC path alone. Under sustained write
flow the quiet window never arrives, so the dump fires at `dump_max_wait`
and each write waits for that dump, while a single sequential writer is a
different shape that pays the debounce on every write. The model, both
profiles' conditions and every measured figure live in
[docs/PERFORMANCE.md](PERFORMANCE.md) — that ledger is the one copy, and
this section deliberately carries no numbers.

## Key-prefix convention

The key-prefix convention: services that share a cluster keep their keys
under one '/'-prefixed namespace each — `myapp/config/...`,
`myapp/state/...` — so concurrent services never collide and `List`/`Watch`
prefixes line up with ownership. The convention is enforced by documentation,
not by the store: the store's map is deliberately
prefix-agnostic and treats a key as an opaque string up to the 1 024-byte
ceiling, because per-prefix policy (namespaced deletion, prefix-scoped
permissions) is a different product, and the convention costs the store
nothing while giving consumers a collision-free namespace. A consumer that
skips the convention still works — it just shares one flat namespace with
every other service on the cluster.

## Consistency caveats

- **Last-writer-wins is the whole conflict story.** Concurrent writes to one
  key resolve deterministically by record version — every replica converges
  to the same winner, never per-replica divergence — and a merge loser is an
  ordinary overwrite, not an error the API reports. Per-key atomicity is the
  only atomicity: there are no multi-key transactions.
- **Read-your-writes holds on the writing node.** Reads are local and
  uncoordinated: a write confirmed by node A is immediately visible to reads
  on A — read-your-writes is node-local, never cluster-wide. A client that
  must read what it just wrote keeps talking to the node it wrote to.
- **Cross-node visibility is bounded, not immediate.** A write reaches peers
  within the convergence bounds — one round trip normally, bounded by
  `ack_timeout_max` under loss and by the sync cadence after a partition
  heals. The bounds and the read semantics live in
  [docs/OPERATIONS.md](OPERATIONS.md).
- **A listed-but-unarrived key reads as Unavailable, not NotFound.** Between
  a peer's complete listing arriving and the listed key's value landing
  locally, a `Get` of that key answers `Unavailable`
  (`startup gate: key is listed by a peer but its value has not arrived`,
  rule 14.5(6)): the key provably exists somewhere in the cluster, so
  claiming absence would lie. Treat the window as retry-with-backoff, never
  as absence.

## Client patterns

Two integrator patterns close this reference: the cache a client keeps
itself, and obtaining the proto.

**The last-read-value cache is consumer-library territory.** A consumer that
must keep answering through a sup-kv outage keeps its own last-read-value
cache: the last value it read per key, served as a fallback while the node
is unreachable. sup-kv does not ship it — no node writes a client-side cache
and no RPC promises one (ADR 0007, decision 14.7, the accepted placement;
see [Consumer library territory](#consumer-library-territory)). The
consumer's library owns the pattern: what to keep, when to refresh it from a
`Get` or a `Watch` event, and when to serve the fallback.

**Obtaining the proto.** The schema is public and versioned in this
repository — [Public proto](#public-proto) carries the import table. In
short: the source is `api/supkvapi/v1/supkvapi.proto`, the generated Go
package `github.com/Djarvur/sup-kv/api/supkvapi/v1` is committed beside it
(so a Go client imports it directly, no code generation), and gRPC server
reflection is always on for non-Go clients. The runnable working example is
the `examples/client` module — a separate Go module
(`example.com/sup-kv-client`, deliberately outside the root module's prefix,
with a `replace` to this repository) that imports only the public package,
demonstrates `Set` with `wait_for_replication`, `Get`, paged `List` and
`Watch`, and builds in CI.
