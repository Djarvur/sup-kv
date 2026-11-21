---
phase: 03-transport
plan: 06
subsystem: replication-wait-seam
tags: [outbox-results, waiter-tracker, wait-for-replication, mirror-removal, xport-12]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-05's record-free (key, version) ledger, indexed due-heap and O(1) unacked/peer gauges this plan's tracker tests observe"
  - phase: 03-transport
    provides: "03-04's four-method repl.StateStore and the result-free Outbox the stream extends"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the wait_for_replication behavior contract, the mode/churn suites and the determinism gate the rewrite must preserve"
provides:
  - "port.Result/ResultKind (Ack, Drained, Superseded, WaitSetEmpty) and Outbox.Ack/Drained/Superseded/WaitSetEmpty/DrainResults as an independent result stream"
  - "core emissions at the exact resolution sites with the TRACKED version; no callback enters the core"
  - "the adapter's mutex-guarded tracker map, created synchronously in NotifyLocalWrite before the notify enqueue (the sole creation site) and deleted on every WaitReplicated return path"
  - "Adapter.WaitReplicated: full write-time coverage is the only success; eviction/supersession are honest false; the count is clamped to the write-time total"
  - "ServerOptions.WaitReplicated replacing PendingCopy end to end; wait.go is a thin projection with the total==0 tracker-release path"
  - "mirrorState, retainDrained, drainedRetention, prevPending/lastAck/drained and the 5 ms poll / 500 ms appearance machinery deleted"

actuals:
  tokens: 33307
  tasks: 3
  commits: 2
  plan_head_before: f831687e22cfbc3d35bfd7a794d6b9f451b115f9
  plan_head_after: c1da11efac37e6a4095c226a53b2aef09e2f7c87

tech-stack:
  added: []
  patterns:
    - "Results travel out of the Outbox as plain data; the driver owns every waiter, and the core stays goroutine- and channel-free (D-04)."
    - "Track at notify, before the enqueue: the tracker exists before any handler can observe the write, so the appearance race is structural, not polled away."
    - "A tracker's done channel closes exactly once under its own mutex; map lookup and deletion are separately mutex-guarded, and a lookup miss is always a drop, never an insert."
    - "Honest-false resolutions: eviction and supersession resolve a waiter without claiming Ack coverage no peer granted."
    - "Per-write tracker tests write the record through the store before notifying (the production author sequence), so the retransmit scan cannot remove the entry mid-test."

key-files:
  created: []
  modified:
    - internal/port/repl.go
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/internal/simnet/simnet.go
    - internal/udprepl/repl.go
    - internal/udprepl/repl_internal_test.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/scenario_test.go
    - internal/udprepl/smoke_test.go
    - internal/grpcapi/wait.go
    - internal/grpcapi/wait_test.go
    - internal/grpcapi/wait_integration_test.go
    - internal/grpcapi/gate.go
    - internal/grpcapi/server.go
    - internal/grpcapi/server_test.go
    - internal/grpcapi/bench_test.go
    - cmd/sup-kv/main.go

key-decisions:
  - "The Ack result carries the peer for the WR-01 count; Drained/Superseded/WaitSetEmpty carry the tracked (older) version, never the displacing one."
  - "WaitSetEmpty is emitted only on the waiter-to-empty transition inside eviction (a pre-existing empty wait-set is not re-announced)."
  - "WaitReplicated success is resolved && ok && clamped == total: a stranger-only acked set cannot report success before the Drained event."
  - "The zero-peer path calls WaitReplicated(total 0) purely as the tracker release point; its report is discarded because D-11 fixes (true, 0, 0)."
  - "The full-coverage wait tests write the record through the store before notifying: an unstored tracked record is removed by the retransmit scan, which would make the eviction test time out."
  - "Tasks 2 and 3 share one commit: the LocalNotify/WaitReplicated signature change and PendingCopy removal are one compile unit, so splitting would leave go build ./... broken between them (the 02-03 commit-boundary precedent)."
  - "The 17-file footprint is one mechanical seam propagation: producer (server.go/main.go), adapter (udprepl), consumers (wait.go/gate.go) and every test constructor; no new files and no extra packages."

patterns-established:
  - "Result vocabulary is four kinds; no fifth was needed for any resolution path (the plan's assumption held)."
  - "Adapter-level tracker tests pin creation-before-enqueue, zero-peer release, timeout release and ctx-cancel release directly against the map."
  - "gRPC wait tests script the tracker seam and keep every pre-existing test name and semantic assertion; the appearance-race test became the immediate-false pin."

requirements-completed: [XPORT-12]

coverage:
  - id: D1
    description: "The Outbox result stream carries Ack/Drained/Superseded/WaitSetEmpty with DrainResults resetting independently of Drain, and the core emits each at its exact resolution site with the tracked version"
    requirement: XPORT-12
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestOutboxResults'"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestOutboxResults_resolutionEvents (ack+stranger, drained, remote/local supersession, eviction)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The adapter creates the tracker synchronously at notify time (before the enqueue), resolves it from the result stream, drops results for absent trackers, and releases every tracker on resolution, zero-peer, timeout, ctx-cancel and failed-notify paths"
    requirement: XPORT-12
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated|^TestNotify_'"
        status: pass
      - kind: unit
        ref: "internal/udprepl/repl_internal_test.go#TestNotify_createsTrackerBeforeCore + TestWaitReplicated_{zeroPeers,timeout,ctxCancel}Releases"
        status: pass
    human_judgment: false
  - id: D3
    description: "The 5 ms poll, 500 ms appearance phase, mirrorState, retainDrained, drainedRetention and PendingCopy are deleted; wait.go reports only the tracker's observed result clamped to the write-time total, and every pre-existing wait behavior is re-asserted"
    requirement: XPORT-12
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWait' (all preserved names plus fullCoverageOnDrained, trackerNilIsSingleNode, noTrackerReportsFalse)"
        status: pass
      - kind: other
        ref: "! grep -rn 'waitPoll|waitAppear|PendingCopy|drainedRetention|mirrorState' internal/grpcapi internal/udprepl cmd/sup-kv"
        status: pass
    human_judgment: false
  - id: D4
    description: "The signature change is wired through producer, adapter, consumers and tests (Set/Delete pass the wait flag; main injects adapter.WaitReplicated), and the real two-adapter loop reports acked=true 1/1"
    requirement: XPORT-12
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_integration_test.go#TestWaitIntegration_ackedThroughRealAdapterTracker"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ ./cmd/sup-kv/"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./... (green) and go build ./..."
        status: pass
    human_judgment: false

duration: 21min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 6: Result-Fed Waiter Trackers and Polling-Mirror Removal Summary

**Per-write waiter trackers created before each notification and fed by a four-event Outbox result stream replace the 5 ms polling mirror, making full write-time coverage the only success and releasing a tracker on every resolution and abandonment path.**

## Performance

- **Duration:** 21 min (1252 s)
- **Started:** 2026-11-21T19:32:43Z
- **Completed:** 2026-11-21T19:53:35Z
- **Tasks:** 3
- **Files modified:** 17 (the plan's accepted seam-propagation footprint; no new files)

## Accomplishments

- `port.Outbox` now carries an independent result stream — `ResultAck`, `ResultDrained`, `ResultSuperseded`, `ResultWaitSetEmpty` — drained by `DrainResults`, with `Drain` untouched so every existing message assertion kept its meaning.
- The core emits at the exact sites: `handleAck` emits an Ack for every matching Ack (strangers included — the WR-01 count) and a Drained at the deletion point; `applyRemote` and `NotifyLocalWrite` emit Superseded for each displaced tracked version; `evictPeers` emits WaitSetEmpty only on the waiter-to-empty transition. No callback crossed into the core.
- The adapter creates a tracker synchronously in `NotifyLocalWrite` when the write requested `wait_for_replication` — before the notify enqueue and therefore before the core or the RPC handler can observe the write. The Run loop only looks trackers up; results for absent identities are dropped and never insert.
- `Adapter.WaitReplicated` waits on the tracker's close-once channel and reports `resolved && ok && clamped == total`; it deletes the tracker on resolution, zero-peer release, ctx cancellation, timeout and failed notify, so the map is bounded by concurrent waits (T-3-16).
- `ServerOptions.WaitReplicated` replaced `PendingCopy` end to end (production wiring, integration test, benchmark), and `wait.go` shrank to a projection with the preserved D-09/D-10/D-11/WR-01/WR-02 semantics and the `total == 0` release call.
- `mirrorState`, `retainDrained`, `drainedRetention`, `prevPending`/`lastAck`/`drained` and the 5 ms poll / 500 ms appearance machinery are gone; the grep gate confirms no name survives in the three touched trees.
- Simnet drains the result stream after every handler alongside the message stream, so long simulations cannot accumulate results even though they ignore them.

## Task Commits

Each task was committed atomically; tasks 2 and 3 share one commit because the signature change and `PendingCopy` removal are one compile unit:

1. **Task 1: Outbox result events and core emissions** — `a848ef6` (feat)
2. **Tasks 2+3: Adapter tracker, WaitReplicated, thin wait.go and the signature propagation** — `c1da11e` (feat)

**Plan metadata:** see the `docs(03-06)` commit that follows this summary.

_Note: no TDD tasks in this plan; each task is one implementation commit except the documented 2+3 boundary move._

## Files Created/Modified

All 17 files are the plan's accepted footprint: the seam's producer (`internal/grpcapi/server.go`, `cmd/sup-kv/main.go`), the port contract (`internal/port/repl.go`), the core emissions (`internal/app/repl/core.go`), the adapter (`internal/udprepl/repl.go`), the simulator drain (`internal/app/internal/simnet/simnet.go`), and every test that constructs or drives the seam.

- `internal/port/repl.go` — `ResultKind`/`Result`, the four Outbox emitters, `DrainResults`; `Drain`/`DrainResults` documented as independent streams.
- `internal/app/repl/core.go` — `handleAck` takes the Outbox and emits Ack+Drained, `applyRemote`/`removeSupersededLocal` emit Superseded, `evictPeers` takes the Outbox and emits WaitSetEmpty.
- `internal/app/repl/retransmit_test.go` — `TestOutboxResults_drainResets` (stream independence and reset) and `TestOutboxResults_resolutionEvents` (all four sites, stranger Ack, local vs remote supersession).
- `internal/app/internal/simnet/simnet.go` — `DrainResults` after `deliver`, the `drive` Advance, and both author sequences.
- `internal/udprepl/repl.go` — `trackerKey`/`tracker` with close-once `done`, `trackersMu`/`trackers`, `NotifyLocalWrite(track bool)` creating before enqueueing and deleting on failure, `applyResults`, `WaitReplicated`, and the deletion of `mirrorState`/`retainDrained`/`PendingCopy`/`drainedRetention` and their fields.
- `internal/udprepl/repl_internal_test.go` — white-box `TestNotify_createsTrackerBeforeCore`, `TestWaitReplicated_zeroPeersReleases`, `TestWaitReplicated_timeoutReleases`, `TestWaitReplicated_ctxCancelReleases`.
- `internal/udprepl/repl_test.go` — the mirror tests became `TestWaitReplicated_{ackCoverage,strangerAckCounts,evictionNotAck,superseded}`; the sync mirror test observes the unacked gauge instead of the removed snapshot; all notify call sites pass the flag.
- `internal/udprepl/scenario_test.go` — convergence asserts both O(1) unacked gauges reach zero.
- `internal/udprepl/smoke_test.go` — notify call passes `track=false`.
- `internal/grpcapi/wait.go` — polling and `waitForDrain` deleted; the thin `waitForReplication` calls the seam, releases on total==0, and clamps the count.
- `internal/grpcapi/wait_test.go` — `fakeTracker` scripts the seam; all existing test names preserved, `TestWait_appearanceRace` replaced by `TestWait_noTrackerReportsFalse`, plus `TestWait_fullCoverageOnDrained` and `TestWait_trackerNilIsSingleNode`.
- `internal/grpcapi/wait_integration_test.go`, `bench_test.go`, `server_test.go` — `WaitReplicated` wiring and the `track` call-site updates.
- `internal/grpcapi/gate.go`, `server.go` — the `WaitReplicated` option and the `LocalNotify`/`notifyLocalWrite` `track` propagation (Set and Delete pass the request flag).
- `cmd/sup-kv/main.go` — `WaitReplicated: adapter.WaitReplicated` in the production wiring.

## Decisions Made

- Result identity is always the tracked write: Ack carries the peer; the other three carry the tracked version, never a displacing one.
- The Ack result is informational only; a waiter resolves success solely on Drained (so stranger-only Acks can never fabricate success).
- WaitSetEmpty is transition-based: eviction announces only when the last waiter leaves, so an already-suspended entry is not re-announced.
- The tracker's `done` closes exactly once under the tracker mutex; the map's mutex is separate, and every lookup miss is a drop.
- `total == 0` in `WaitReplicated` deletes any tracker and returns `(true, 0)` — the D-11 release point.
- Task/adapter tests follow the production author sequence (store write, then notify): the retransmit scan removes a tracked version the store cannot resolve, which would make timestamp-bound waits flaky.
- The 17-file spread is one seam propagation; the commit split moved only where compilation required (tasks 2+3 together).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Tasks 2 and 3 landed as one commit**
- **Found during:** Task 2/3 commit boundary
- **Issue:** `NotifyLocalWrite` gaining `track bool` and `ServerOptions.PendingCopy` becoming `WaitReplicated` are one compile unit: committing the adapter alone would leave `cmd/sup-kv/main.go` and `internal/grpcapi` unable to build at the intermediate commit.
- **Fix:** Committed the adapter, the gRPC/main propagation and all consumer tests together in `c1da11e`; Task 1 remains its own commit `a848ef6`.
- **Files modified:** the 13 Task-2/3 files
- **Verification:** `go build ./...`, the full verification set and the full-repo lint are green at both commits' tree states (the second is HEAD).
- **Committed in:** `c1da11e`

**2. [Rule 3 - Test correctness] Tracker tests write the store record before notifying**
- **Found during:** Task 2 eviction test
- **Issue:** A notification whose record is absent from the store is removed by the retransmit scan at its first due time; with a short peer timeout the eviction test could lose its entry and time out instead of observing WaitSetEmpty.
- **Fix:** Added the store-write-then-notify author sequence (`writeStoredRecord`) to the four tracker tests.
- **Files modified:** `internal/udprepl/repl_test.go`
- **Verification:** `go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated'` green, including the 300 ms eviction case.
- **Committed in:** `c1da11e`

**3. [Rule 1 - Test harness] One persistent sender socket in the white-box tracker test**
- **Found during:** Task 2 `TestNotify_createsTrackerBeforeCore`
- **Issue:** Sending the Hello and the Ack from two freshly dialed sockets gave node B two source handles; the core's duplicate-node_id classifier drops a remembered run_id announcing from a new handle, so the Ack never drained the entry.
- **Fix:** The test dials once and sends both datagrams over that socket.
- **Files modified:** `internal/udprepl/repl_internal_test.go`
- **Verification:** the full tracker-test set green.
- **Committed in:** `c1da11e`

**4. [Rule 2 - Lint compliance] One justified suppression for the mandated flag parameter**
- **Found during:** Task 3 lint gate
- **Issue:** `revive` flags `NotifyLocalWrite(..., track bool)` as a control flag, but the plan mandates that exact signature and splitting it into two methods would split the D-05 producer contract.
- **Fix:** `//nolint:revive // flag-parameter: ...` naming the linter and the reason (AGENTS.md requires both).
- **Files modified:** `internal/udprepl/repl.go`
- **Verification:** `go tool golangci-lint run ./...` green.
- **Committed in:** `c1da11e`

---

**Total deviations:** 4 auto-fixed (1 commit-boundary blocking issue, 2 test-correctness fixes, 1 justified lint suppression)
**Impact on plan:** No scope change. Every plan acceptance criterion and verification command passed; the only structural movement is the task 2+3 commit boundary, forced by compilation.

## Issues Encountered

- The first tracker tests raced the notification against the Ack/DataReplication events; they now wait for the O(1) unacked gauge (and KnownPeers) to confirm the event was processed before sending the resolving datagram.
- Lint iterations: `noinlineerr`, `paralleltest`, `noctx`, `gci`, `unparam` (three helper signatures), `modernize` (`min`), and `revive` (flag parameter plus an import-shadowing local named `port`) were all fixed or justified.
- `TestWaitReplicated_evictionNotAck` is inherently real-time (~300 ms) because it must let the core's peer timeout fire; it sits well inside the 2 s wait deadline and passed under the full parallel `-race` suite.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 03-07's paced record queue can index sends by the same `pendingKey`; the tracker map's identity discipline and the result stream are independent of it.
- Phase 5's gRPC wait path is fully wired: Set and Delete pass the request's wait flag through `LocalNotify`, and `Adapter.WaitReplicated` is the injected `ServerOptions.WaitReplicated` method value.
- Phase 6 observability can read the tracker seam without touching the core; the old pending snapshot is gone, so no per-event copy remains to optimize or observe.
- The four-event vocabulary covered every resolution path; no fifth kind was needed (the plan's probe assumption held).

## Self-Check: PASSED

- All 17 plan files exist on disk; no extra files were created.
- Commits verified in history: `a848ef6`, `c1da11e`; measured count from the plan ledger `f831687..c1da11e` is 2.
- The `evaluation-scope` query verb is unavailable in this runtime (the CLI reports `Unknown command`), so the self-check uses the persisted plan ledger and `git rev-list --count` as the equivalent measurement (the documented stale-ledger fallback).
- Acceptance re-check at close-out: `go build ./...` green; the four-package `-race` set and the `TestChaos|TestScenario` determinism run green; the polling/mirror grep gate exits zero with no match; `go tool golangci-lint run ./...` green.
- Acceptance criteria re-verified: DrainResults resets and the streams are independent; the four events emit at the named sites with the tracked version; trackers are created before the enqueue and the map is empty after every wait/release path; `wait.go` has no timer poll; `WaitReplicated` replaced `PendingCopy` end to end; full coverage reports true and partial coverage reports false with the observed count.

---
*Phase: 03-transport*
*Completed: 2026-11-21*
