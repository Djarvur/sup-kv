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
