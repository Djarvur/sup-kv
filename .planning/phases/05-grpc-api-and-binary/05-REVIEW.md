---
phase: 05-grpc-api-and-binary
reviewed: 2026-11-09T14:11:17Z
depth: deep
files_reviewed: 27
files_reviewed_list:
  - internal/grpcapi/server.go
  - internal/grpcapi/watch.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/wait.go
  - internal/grpcapi/apiproto/supkvapi.proto
  - internal/grpcapi/server_test.go
  - internal/grpcapi/watch_test.go
  - internal/grpcapi/gate_test.go
  - internal/grpcapi/wait_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/app/repl/core.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sync_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_test.go
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - Makefile
  - buf.yaml
  - buf.gen.yaml
  - go.mod
  - go.sum
  - internal/grpcapi/apiproto/supkvapi.pb.go
  - internal/grpcapi/apiproto/supkvapi_grpc.pb.go
findings:
  critical: 1
  warning: 2
  info: 1
  total: 4
  necessity: 1
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-11-09T14:11:17Z
**Depth:** deep (cross-file call chains traced: grpcapi wait ↔ repl core ↔ udprepl mirror)
**Files Reviewed:** 27 (source + tests + toolchain pins + generated-code provenance)
**Status:** issues_found

## Summary

Phase 5 adds the gRPC surface (`internal/grpcapi`: unary handlers, WatchHub, startup gate,
wait_for_replication), the storage OnChange tap, the repl/udprepl read seams (`syncArrived`,
`PendingInfo.Acked`, adapter atomics), the exported `persist.Engine.Flush` shutdown seam, and
the `cmd/sup-kv` binary (flags/env, D-19 ordered shutdown). Verified during review:
`go build ./...` and `go vet ./...` clean; `go test -race` over all touched packages passes;
`golangci-lint config verify` + `go tool golangci-lint run ./...` clean; both generated files
(`supkvapi.pb.go`, `supkvapi_grpc.pb.go`) are byte-identical to a fresh `go tool buf generate`
run into a scratch directory — no hand edits; go.mod/grpc v1.85.0/protobuf/buf v1.75.0/
golangci-lint v2.15.0 pins are consistent with AGENTS.md policy; depguard boundaries hold;
every `//nolint` names its linter with a reason; the snapshot name path-traversal control is
exercised through the real persistfile adapter.

The Watch surface (hub goroutine, exact cut, slow-consumer shed), the gate (resolver, health
binding, write gating) and the binary shutdown order check out under adversarial reading —
including the two-phase subscribe, the overflow-flag wake guarantee (a full ingress always has
queued events that wake the loop before the flag is consulted), and the flush/runDone exit race
in `persist`.

The one Critical finding is a cross-module semantic gap: **`wait_for_replication` cannot report
`acked=true` on the healthy replication path.** Its success criterion requires observing an
Acked-coverage state that the real repl core never publishes — the core deletes the pending
entry in the same handler call that completes coverage, so the adapter mirror jumps straight
from "partial Acked" to "absent". The unit tests pass only because the fake mirror publishes a
state the real core never produces. Two Warnings cover a doc/contract contradiction in
`handleAck` (outside-wait-set Acks are counted, contradicting both the doc comment and the
D-08/D-10 report semantics) and an unvalidated `wait_timeout_ms` whose ns conversion can
overflow. Known phase-4 info findings (IN-01 ownHandle dead state, IN-02 simnet Write error
ignored) are not re-reported.

## Critical Issues

### CR-01: wait_for_replication can never report acked=true against the real mirror — success criterion is observationally unsatisfiable, and fast replication burns the 500ms appearance bound

**File:** `internal/grpcapi/wait.go:113-156` (with `internal/app/repl/core.go:488-500`, `internal/udprepl/repl.go:200-228`, `internal/grpcapi/wait_test.go:153,258`)
**Issue:** Both success branches of the drain phase require *observing* `len(Acked) == total`
on a pending entry that is still present (`wait.go:123-127` on absence, `wait.go:134-141` on
empty WaitingOn). But the real core deletes the entry in the same `handleAck` call that empties
the wait-set (`core.go:495-497`: `if len(entry.waitSet) == 0 && len(c.peers) > 0 { delete(c.pending, ack.Key) }`),
and the adapter mirror is refreshed only after full event-loop iterations, one datagram per
iteration (`repl.go:208-226`). So the published mirror for a healthy replication transitions
`{Acked: ≤ total-1, WaitingOn: non-empty}` → `absent` — **no mirror state ever exists with the
entry present and full Acked coverage**. Every poll therefore observes either a partial count
(drain → `lastAcked < total` → acked=false) or absence with `lastAcked` below total → acked=false.
The only present-with-empty-WaitingOn state the core publishes is eviction suspension
(all peers evicted, table empty), where acked=true additionally requires every peer to have
delivered an Ack *after* eviction — the degenerate path `TestMirror_pendingAckVisible`
(`repl_test.go:315-341`) itself has to resort to ("zero known peers — the suspension rule").

Observable consequences on the production wiring (`main.go:324-330`):
1. Every fully-successful replication reports `acked=false` with partial or zero counts —
   API-05's D-08 report fails its purpose exactly when replication works.
2. When the whole notify→broadcast→ack cycle completes inside one 5ms poll window (the normal
   healthy-LAN case), the appearance phase sees only absence and polls until `waitAppear`
   (500ms, `wait.go:32,79-96`) before returning `(false, 0, total)` — half a second of added
   latency on the happy path, plus the false report.

The unit tests mask this: `TestWait_acksArrive`/`TestWait_appearanceRace` publish
`repl.PendingInfo{Version: ver, Acked: []uint16{1}}` with the entry *present* and WaitingOn
*empty* (`wait_test.go:153,258`) — a state `TestRetransmit_ackDrainSemantics`
(`retransmit_test.go:414-418`) proves the real core never produces ("pending entry retained
after an Ack from every waited-on peer" fails if the entry survives). No test drives
`waitForReplication` against the real adapter mirror with ≥1 known peer. Secondary note: the
write-time `total` comes from the stale-by-one-iteration `KnownPeers` mirror while the wait-set
is seeded later on the driver goroutine, so `total` can also disagree with the entry's actual
wait-set across a discovery/eviction race.

This does not lose writes (the wait never changes the write outcome — that contract holds), but
it makes the shipped wait_for_replication feature functionally broken in its primary scenario.

**Fix:** Treat post-appearance absence as drained success: the core's own contract is that an
entry is deleted exactly when every write-time waiter Acked, so after the appearance phase has
confirmed the entry existed with the tracked version, absence during drain should resolve
`acked=true` (e.g. `return true, lastAcked, total` or `true, total, total`). The residual
ambiguity — supersession (remote or local) deleting and draining the replacement within one
poll window — is rare and strictly less dishonest than today's always-false. Alternatively (or
additionally), have the adapter mirror retain recently drained entries for one bounded
generation as resolved `{Version, Acked: full, WaitingOn: nil}` rows, which removes the
ambiguity and yields exact counts without touching REPL-02 core semantics. Whichever shape is
chosen, add an integration test that drives a Set with `wait_for_replication` through the real
`udprepl.Adapter` mirror with at least one acking peer and asserts `acked=true`.

## Warnings

### WR-01: handleAck doc contradicts the code — Acks from peers outside the wait-set are counted, inflating acked_count beyond total_peers

**File:** `internal/app/repl/core.go:482-494`
**Issue:** The doc comment pins "an Ack from a peer outside the wait-set [is one of the]
silent no-ops — no counter, no log (probe #2)" (`core.go:482-487`). The code does the
opposite: after the stale/unknown-key guard, it runs `delete(entry.waitSet, peerID)` (a no-op
for a foreign peer) and then `entry.acked = ackedInsert(entry.acked, peerID)` (`core.go:493-494`)
— any node_id sending a matching (key, version) Ack joins the Acked set, wait-set membership
or not. This doc/code contradiction was introduced when 05-02 added the Acked seam without
revisiting the probe-#2 pin. Consequences: (a) `PendingInfo.Acked` — the T-05-04 honesty seam
feeding the D-08 report — counts Acks from nodes outside the write-time peer snapshot,
contradicting D-10's "peer set fixed at write time"; (b) `acked_count` can exceed
`total_peers` in the report (e.g. total=1, two duplicate-Acking strangers → Acked=[7,9] →
acked_count=2 > total_peers=1), and the inflated count interacts with the success criterion
(`lastAcked == total`) in unpredictable ways.
**Fix:** Pick one semantics and align code, doc and report: either restrict `ackedInsert` to
peers that were in the entry's original write-time wait-set (requires keeping the original
set on the entry), or update the doc comment to the counting behavior and clamp the report's
`ackedCount` to `min(ackedLen, total)` in `waitForDrain`. The doc comment must stop asserting
a no-op the code does not perform.

### WR-02: wait_timeout_ms is converted to a duration without range validation — int64 overflow yields arbitrary (possibly unbounded) server-side waits

**File:** `internal/grpcapi/wait.go:161-166` (fields at `internal/grpcapi/apiproto/supkvapi.proto:69,88`)
**Issue:** `time.Duration(requestMs) * time.Millisecond` silently wraps for positive
`wait_timeout_ms` above ~9.2×10¹² (int64-ns overflow). A hostile or buggy client can wrap the
deadline negative (an immediate, wrong report) or to a huge positive value — the wait loop then
polls every 5ms with an effectively unbounded deadline, holding the handler goroutine and timer
until the client disconnects (server-side there is no per-RPC deadline; `ctx.Done()` is the only
exit). Negative and zero values already fall back to the operator default, so only the
positive-overflow path is unvalidated remote input.
**Fix:** Validate/clamp at the handler: reject `wait_timeout_ms` above a sane ceiling (e.g.
`math.MaxInt64/time.Millisecond` or `10*defaultWaitTimeout`) with InvalidArgument, or clamp it
in `waitTimeout` to the ceiling. One guard covers both Set and Delete since they share
`waitTimeout`.

## Info

### IN-01: Generality without instantiation — mapError's status-passthrough branch is unreachable in this stack

**File:** `internal/grpcapi/server.go:306-308`
**Category:** necessity
**Issue:** `if _, ok := status.FromError(err); ok { return err }` defends against double-mapping
an already-status error into Internal. No caller can produce one: mapError's only call sites
pass `persist.Engine`/persistfile errors, and the depguard rule `sup-kv-app-no-io` forbids
`internal/app` from importing `google.golang.org/*`, so no engine error can implement
`GRPCStatus()`. The branch cannot fire today (provable from `.golangci.yml` + the call sites);
every engine error falls through the six mapped sentinels to the Internal branch as designed.
**Fix:** Removal is a scope decision, not a defect fix — the branch is harmless forward
compatibility for future wire additions (e.g. interceptors). If kept, it is adequately covered
by the doc comment; if the phase wants a strict minimalism pass, delete it.

## Verified-clean areas (adversarial reading, no findings)

- **Watch exact cut:** snapshot is taken on the hub goroutine at registration; any event
  dispatched after either predates `Records()` (suppressed by the equal-version rule) or
  strictly dominates the snapshot (delivered) — no duplicate/hole. Overflow flag always wakes
  (a full ingress guarantees queued events drain before the flag is consulted). Slow-consumer
  shed mid-snapshot is wasteful but contract-correct (resubscribe re-snapshots).
- **Gate:** resolver is the single opener with CAS-bound health flip; writes gated before
  validation per A7/API-04; reads deliberately ungated per API-04.
- **Binary D-19 order** (`main.go:397-428`): health NOT_SERVING → apiCancel (ends Watch, hub)
  → bounded GracefulStop with Stop fallback → replCancel + join → `engine.Flush()` → engineCancel
  → join. Matches the contract; flush/runDone exit races are regression-tested
  (`TestEngine_flushRacesRunExit`, `TestEngine_flushRacesExitMidDump`).
- **OnChange tap** (`store.go:251-261`): fires only from `install` on actual state change
  (D-06); non-blocking Ingest under the write lock (T-05-05); checksum recompute in
  `notifyLocalWrite` matches the store's own computation.
- **Snapshot RPC security:** names cross to persistfile's `validateName` unchanged; hostile
  names tested end-to-end through the real adapter (`gate_test.go:311-322`).
- **Toolchain:** generated code provenance verified by regeneration diff; lint gate clean;
  no debug artifacts; all `//nolint` carry linter name + reason.

---

_Reviewed: 2026-11-09T14:11:17Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
