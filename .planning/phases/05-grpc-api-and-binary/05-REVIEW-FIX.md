---
phase: 05-grpc-api-and-binary
fixed_at: 2026-11-09T14:48:48Z
review_path: .planning/phases/05-grpc-api-and-binary/05-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 5: Code Review Fix Report

**Fixed at:** 2026-11-09T14:48:48Z
**Source review:** .planning/phases/05-grpc-api-and-binary/05-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 3 (CR-01, WR-01, WR-02 — per triage; IN-01 out of scope)
- Fixed: 3
- Skipped: 0

**Gates:** `go test -race -count=1 ./...` exit 0 and `go tool golangci-lint run ./...`
exit 0, run in the **main checkout** on branch
`gsd/phase-4-replication-and-deterministic-simulator` (no worktree — the
orchestrator directed commits to extend the current branch directly). Zero new
`//nolint`; the pre-existing gosec directives on the touched lines are carried
over unchanged. `internal/app` received comment-only changes (verified: zero
non-comment diff lines), so the depguard boundary is untouched.

## Fixed Issues

### CR-01: wait_for_replication can never report acked=true against the real mirror

**Status:** fixed: requires human verification (the fix changes cross-module
mirror semantics; the human should confirm the shape matches intent — see the
direction note below)

**Files modified:** `internal/udprepl/repl.go`, `internal/grpcapi/wait.go`,
`internal/grpcapi/wait_integration_test.go` (new file — the review-required
integration test)
**Commit:** 05f84de (plus lint-gate touches in a61f357)

**Applied fix:** The review offered two shapes — (A) treat post-appearance
absence during drain as drained success, or (B) have the adapter mirror retain
recently drained entries for one bounded window as resolved rows — "or
additionally". **Shape B alone was implemented**, because shape A cannot fix
the finding's second named consequence (a cycle finishing inside one 5ms poll
window never appears to the poller at all, still burning the 500ms appearance
bound into an honest-looking false) and would leave the required integration
test failing on fast infrastructure. With B:

- The adapter retains a drained entry as a resolved row
  (`{Version, WaitingOn: nil, Acked: last-observed WaitingOn ∪ Acked}`) for
  `drainedRetention = 250ms` — comfortably above the 5ms API poll period.
  The union is exactly the core's acked set at deletion (the deletion fires
  inside the handleAck that empties the wait-set; evicted-never-acked peers
  live in neither observed set and stay excluded — T-05-04, Pitfall 3).
- Drain is distinguished from remote supersession by correlating the entry's
  disappearance with the Ack the event loop just processed (one event per
  iteration, D-04; the correlation flag clears per mirror refresh). A
  vanishing on any other event — a strictly greater remote version's
  supersession above all — retains nothing, so the mirror never claims Acked
  coverage for a version the peers moved past (the Q2 pin stays honest:
  superseded reports false through the unchanged absence branch).
- wait.go hands the appearance phase's observation straight into
  `waitForDrain` as its first evaluation — no re-poll gap in which a fast
  resolution could be missed (this also removes the same pre-existing race
  for live entries).

Eviction is not an Ack anywhere in the fix; the wait never changes the write
outcome (API-05, D-10) — untouched.

**Verification (fails before / passes after):** the new integration test
drives the production scenario end to end: two real `udprepl.Adapter`s on
asymmetric loopback ports (broadcast discovery crosses; every unicast — the
Ack above all — lands on a port exactly one socket holds, so the adapter's
shared-port caveat never applies), a real persist engine as the adapter's
store, `Set` with `wait_for_replication` through the real adapter mirror,
asserting `acked=true 1/1` well under the 400ms bound plus byte-equal
corroboration on the peer. Observed against the pre-fix code: **FAIL —
`acked=false 0/1` after the ~500ms appearance burn**; post-fix: **PASS**
(3 consecutive `-count=3` runs plus the full `-race` suite). A loud
capability probe skips honestly on platforms without loopback broadcast
(the smoke test's D-08 precedent).

### WR-01: handleAck doc contradicts the code — outside-wait-set Acks are counted

**Status:** fixed
**Files modified:** `internal/grpcapi/wait.go`, `internal/app/repl/core.go`
(comment-only), `internal/grpcapi/wait_test.go`
**Commit:** af3d5c4

**Applied fix:** The review offered either restricting `ackedInsert` to the
original write-time wait-set, or keeping the counting behavior and aligning
doc + report. **The counting-behavior direction was implemented** (with the
review's clamp), because the restriction shape entangles a design decision
beyond the stated direction: discovery adds post-write peers to live
wait-sets, so an origin-set gate would leave a discovery-added acker absent
from both sets — silently breaking the T-05-04 "absent from both sets was
evicted, never silently Acked" invariant the review elsewhere treats as
pinned. The implemented shape:

- `handleAck`'s doc now states what the code does: stale and unknown-key
  Acks stay silent no-ops (probe #2 holds); any other node_id's matching Ack
  joins the acked set, wait-set membership or not, so the acked length can
  exceed the write-time snapshot.
- D-10 is honored where the report is produced: `waitForDrain` clamps the
  observed Acked count to the write-time total, so both the reported
  `acked_count` and its `lastAcked == total` success criterion live inside
  the peer set fixed at write time — stranger noise can never inflate the
  report or flip the criterion.

**Verification (fails before / passes after):** `TestWait_strangerAcksClampedToTotal`
drives two stranger Acks (node ids 7, 9) against one write-time peer, then
resolves with the real peer's Ack. Observed pre-fix: **FAIL — `acked=false
3/1`** (inflated count breaking both the criterion and the total_peers
bound); post-fix: **PASS — `acked=true 1/1`**. Core behavior is unchanged
(zero non-comment diff in `internal/app`), so all pinned repl semantics
tests pass untouched.

### WR-02: wait_timeout_ms converted without range validation — int64 overflow

**Status:** fixed
**Files modified:** `internal/grpcapi/wait.go`, `internal/grpcapi/wait_test.go`
**Commit:** 6c171bf (plus lint-gate touches in a61f357)

**Applied fix:** Per the review's clamp direction (rejection is not an option
here beyond the review's own text: the write is already engine-confirmed when
the timeout resolves, and no path may turn a confirmed write into an error):
`waitTimeout` applies two bounds, one guard covering both Set and Delete —

1. values above `maxWaitTimeoutMs = math.MaxInt64 / time.Millisecond` clamp
   to it before the conversion (the raw product of a hostile value wraps to
   arbitrary — negative or huge-positive — durations);
2. the resolved duration caps at `waitTimeoutCeilingFactor` (10) times the
   operator default, so no client holds a write handler an order of
   magnitude beyond the operator's own configuration (the cap applies to the
   per-request field only; negative and zero keep the D-09 fallback, and an
   absurd operator default whose ×10 overflows is guarded by a `> 0` check).

**Verification (fails before / passes after):** `TestWait_overflowingTimeoutClamped`
uses arithmetic-verified hostile values: `18446744173910 ms` whose raw
conversion wraps to exactly `+100.2s`, and `9223372036855 ms` (conversion
ceiling + 1) which wraps to `−292y`. Observed pre-fix: **FAIL — rpc
DeadlineExceeded** (the server held the wrapped ~100s wait until the 3s
client deadline); post-fix: **PASS** — both values behave like ordinary
bounded timeouts (false 0/1, ~500ms under the 50ms operator default), the
negative wrap no longer returning on an already-expired deadline.

## Skipped Issues

None — every in-scope finding was fixed.

**Out of scope (left as-is, recorded open by triage):** IN-01 (mapError's
status-passthrough branch, `internal/grpcapi/server.go:306-308`) was not
touched, per the triage disposition.

## Notes for the human verifier

- **CR-01 shape choice:** shape B (adapter-side retention) rather than the
  review's first-listed shape A (drain-absence as success). Both are
  review-sanctioned ("Alternatively (or additionally)... Whichever shape is
  chosen"); B was chosen because it is the only shape that fixes both named
  consequences and it preserves Q2 honesty for remote supersession, where
  combining A with B would report a superseded version as acked. The
  Ack-vs-supersession correlation uses information the adapter mechanically
  possesses (which message it just fed the core) without re-deriving any
  protocol decision.
- The `style(05)` commit a61f357 carries the golangci-lint strict findings
  from the two fix commits (named ceiling factor, plain error assignments in
  the probe, skip message within the 200-col bound); no behavior change.
- The integration test's asymmetric-port topology is a test-side wiring
  choice (each node binds its own port and broadcasts toward the other's
  port) to make unicast Ack delivery deterministic on one host; production
  wiring in `cmd/sup-kv` is untouched.

---

_Fixed: 2026-11-09T14:48:48Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
