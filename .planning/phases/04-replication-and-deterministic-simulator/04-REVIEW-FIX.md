---
phase: 04-replication-and-deterministic-simulator
fixed_at: 2026-11-06T20:38:27Z
review_path: .planning/phases/04-replication-and-deterministic-simulator/04-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 4: Code Review Fix Report

**Fixed at:** 2026-11-06T20:38:27Z
**Source review:** .planning/phases/04-replication-and-deterministic-simulator/04-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (CR-01 Critical, WR-01 Warning; IN-01/IN-02 stay open per the fix scope)
- Fixed: 2
- Skipped: 0

## Fixed Issues

### CR-01: applyRemote Acks — and supersedes pending tracking with — a record the store rejected

**Files modified:** `internal/app/repl/core.go`, `internal/app/repl/retransmit_test.go`
**Commit:** 957fd0a
**Applied fix:** `applyRemote` now returns immediately after logging a failed
`store.ApplyRemote` — no Ack and no pending-supersession on the error path, so
the sender keeps retransmitting and the unacked gauge keeps climbing instead of
the protocol confirming a record that was never stored. The error-free path is
unchanged: a no-op merge (duplicate/equal-version redelivery) still returns nil
and still Acks (REPL-09's adjacency edge preserved). The doc comment states the
never-confirm-a-refused-record contract.

**Tests added** (subtests of `TestRetransmit_ackDrainSemantics`, plus helpers
`newLimitedTestCore` and `assertAck`):
- `a store-rejected apply is never Acked and never supersedes tracking —
  retransmission continues` — a strictly greater remote version whose Data
  exceeds the store's `ValueMaxBytes` (the heterogeneous-limit shape,
  `ErrValueTooLarge`) produces zero outbound messages, leaves the pending entry
  tracking the original version on both waiters, surfaces through `Config.Log`,
  and the retransmission ladder still fires the tracked record.
- `an error-free duplicate redelivery of an already-stored record still Acks` —
  the peer echo and its duplicate redelivery each emit exactly one unicast Ack
  for (key, version) without superseding the tracked entry.

This is a logic-condition change (the contract's human-verification class); it
is proven by the two regression subtests above and the full `make test` below,
not only by structural verification.

### WR-01: Adapter.NotifyLocalWrite blocks forever once Run has exited and the notify buffer fills

**Files modified:** `internal/udprepl/repl.go`, `internal/udprepl/repl_test.go`, `internal/udprepl/smoke_test.go`
**Commit:** 58cd6c7
**Applied fix:** `NotifyLocalWrite` now returns `error` (the seam allows it —
no interface constrains the adapter, and the only existing call site is
statement-shaped) and its send is guarded by `a.done` with a closed-loop
pre-check (the `readLoop` exit-guard precedent): while Run is active it
enqueues and returns nil; once Run has returned — `close(a.done)` has run —
every call deterministically drops the notification and returns the new
exported sentinel `ErrRunLoopExited` instead of blocking on a queue nobody
drains. A writer on the public write path can no longer wedge after the
replication context stops, and the drop is surfaced (Phase 5's gRPC handler
can fail the write visibly) rather than silently swallowing the first 1024
notifications. `smoke_test.go`'s call site now checks the returned error.

**Test added:** `TestNotifyLocalWrite_afterRunReturns` (in
`internal/udprepl/repl_test.go`) — drives a real adapter's `Run` to
`context.Canceled`, then makes 1100 post-Run `NotifyLocalWrite` calls (past
the 1024-entry notify buffer — exactly the WR-01 block-forever window), each
bounded by a deadline: every call returns `ErrRunLoopExited` promptly, none
blocks, none panics.

## Skipped Issues

None — IN-01 and IN-02 were outside this run's fix scope (Info tier) and stay
open in the source review.

## Verification

`make lint` (config verify + golangci-lint 2.15.0 strict gate) — exit 0, zero
findings, zero new `//nolint`. `make test` (`go test -race -cover ./...`) —
exit 0, all packages ok. Both gates ran in the main checkout on
`gsd/phase-4-replication-and-deterministic-simulator` (no isolated worktree;
the coordinator directed in-repo work), so the results are reproducible from
the tree as committed.

---

_Fixed: 2026-11-06T20:38:27Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
