---
phase: 06-api-and-semantics
plan: 11
subsystem: api-and-replication
tags: [apis-01, captured-set-promise, tracker-wants, wait-set-empty, discovery-flush-split, in-05, superseded-fast-path, unknown-ack-inertness, zero-peer-release, two-node-integration, tdd-red-evidence, status-complete]
requires:
  - phase: 06-api-and-semantics
    provides: 06-02's write-time peer capture, the peer-carrying LocalNotify, the seeded wait-set and the WaitSetEmpty emission this plan makes authoritative
  - phase: 05-persistence
    provides: the Outbox result stream (Ack/Superseded/WaitSetEmpty/Drained) the tracker consumes
  - phase: 04-protocol
    provides: the core peer table, the discovery flush and the retransmission ladder the tracked entry rides
provides:
  - "the adapter tracker scoped to its captured write-time peers (total + wants): outside Acks are inert and a partial captured set can never report full coverage"
  - "the discovery flush split: it still broadcasts the record and re-arms the ladder, but never inserts the new peer into a wait-set or the per-peer ledger (IN-05 closes)"
  - "the honest zero-peer release verdict (false, 0), with the notify-time tracker still deleted"
  - "the Superseded fast path pinned deterministically (one Outbox result in the displacing call) and on real sockets under a generous 250 ms bound"
  - "the regression matrix: unknown-Ack inertness, captured-peer-only completion, zero-peer release and a two-node post-write-discovery integration test"
affects:
  - 06-13-phase-gate
  - 06-03
  - phase-7-metrics
actuals:
  tokens: 16216
  tasks: 2
  commits: 4
  plan_head_before: ef7bf467daa9c3ce283df18cedd5577baffb82be
  plan_head_after: 034814e6f27e813fa65125a150eef11c93bab6bb
tech-stack:
  added: []
  patterns:
    - "Promise ownership: the tracker stores both the captured total and membership, and the report is the intersection of the acked set with that membership — the caller's total is only the release signal, never the coverage denominator."
    - "Broadcast-without-promise: a discovery flush re-arms the ladder and queues the record send while leaving the wait-set untouched, separating 'who receives the record' from 'who the write promised'."
    - "Structural fast-path proof: the 20 ms displacement target is proven by the displacing handler call carrying ResultSuperseded in its Outbox, while the real-socket regression keeps a generous CI-safe bound."
key-files:
  created:
    - internal/app/repl/wait_internal_test.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-11-task1-core.json
    - .planning/phases/06-api-and-semantics/red-evidence/06-11-task1-tracker.json
  modified:
    - internal/udprepl/repl.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/repl_internal_test.go
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/peertable_test.go
    - internal/app/repl/restart_test.go
    - internal/grpcapi/wait_test.go
key-decisions:
  - "The tracker's captured total is authoritative over the caller's total: WaitReplicated counts the acked-set ∩ wants, and a tracker whose captured total is zero releases the same way as a total==0 call. The caller's total remains the release signal and must equal the captured length in production."
  - "The zero-peer release now answers (false, 0) at the adapter seam as the plan directs (the 06-02 (true, 0) 'benign release' verdict is retired). The handler still owns the public report and discards the release verdict; both now agree on the honest false."
  - "The discovery flush keeps the ladder reset, the retry re-arm and enqueueRecordSend, and drops only waitSet[in.NodeID] and ledgerAdd: the record still reaches the newly discovered peer, the promise does not widen (T-06-07)."
  - "Every test that had pinned the flush's wait-set growth was updated deliberately: TestRetransmit_suspendAndFlush, TestPeerTable_solePeer, TestRestart_firstHelloStillFlushesPending, TestWaitReplicated_ackCoverage and TestNotify_createsTrackerBeforeCore."
  - "The two-node integration test parks the captured peer's ApplyRemote until the third node's Hello has been processed, so B's Ack alone completes the wait while the discovered peer is demonstrably known — no scheduler race decides the outcome."
  - "Task 2 is a regression-pinning task: its tests pass on the Task 1 tree, so no second RED commit exists (tdd.md: a test that does not fail means the feature exists). The behavior change and its real RED live in Task 1."
patterns-established:
  - "A promise-set report is computed as an intersection, never a clamped raw count: the membership set is the only source of truth for both the numerator and the denominator."
  - "A discovery path that must not widen an existing promise is tested white-box for the ledger (peerUnacked) and black-box for WaitingOn, because the two structures can drift independently."
requirements-completed: [APIS-01]
coverage:
  - id: D1
    description: "The adapter tracker counts only Acks from the captured write-time peers; an Ack from a peer outside the capture neither resolves the wait nor increments the count, and a partial captured set can never report full coverage."
    requirement: "APIS-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated_(trackerCapturedPeersOnly|strangerAckDoesNotResolve|capturedPeerOnly)$' -> PASS (both tracker subtests plus the two stranger-Ack subtests and the two-peer captured-only path)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A captured-but-gone peer resolves ResultWaitSetEmpty and reports (false, 0) immediately, without waiting out the caller timeout."
    requirement: "APIS-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated_waitSetEmptyImmediate$' -> PASS (0.31s: eviction then notify with a stale capture, report false 0 under 250 ms against a 2 s timeout)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The discovery flush still broadcasts/re-arms the record but never inserts the new peer into a wait-set or the per-peer ledger (IN-05 closes)."
    requirement: "APIS-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^(TestPeerTable_discoveryFlushKeepsPromise|TestRetransmit_suspendAndFlush|TestPeerTable_solePeer|TestRestart_firstHelloStillFlushesPending)$' -> PASS"
        status: pass
    human_judgment: false
  - id: D4
    description: "A displaced version resolves through ResultSuperseded in the same displacing handler call (the structural 20 ms proof), and the real-adapter regression returns in well under 250 ms against a 5 s caller timeout."
    requirement: "APIS-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestNotify_supersededResult$' -> PASS"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated_supersededFast$' -> PASS (0.02s)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The zero-peer path releases the notify-time tracker and reports (false, 0), and a two-node real-adapter wait with a post-write discovery completes 1/1 from the captured peer alone."
    requirement: "APIS-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated_zeroPeersReleases$' -> PASS (tracker present after notify, absent after the single total==0 wait)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWait_zeroPeers$' -> PASS (exactly one total==0 release call)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWait_integrationCapturedPeerOnly$' -> PASS (post-write third-node discovery, report acked=true 1/1)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The changed tree passes the repo-wide strict gates without regressing any package."
    requirement: "APIS-01"
    verification:
      - kind: process
        ref: "go build ./... && go vet ./... -> exit 0"
        status: pass
      - kind: process
        ref: "go tool golangci-lint run ./... -> exit 0"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./... -> every package ok (app/repl 107.3s, grpcapi 2.2s, udprepl 1.4s, cmd/sup-kv 7.5s, stand 8.1s)"
        status: pass
    human_judgment: false
duration: 30min
completed: 2026-11-22
status: complete
---

# Phase 06 Plan 11: Captured-Set Promise Enforcement Summary

**`wait_for_replication` now counts only the peers the write actually promised: the tracker is scoped to its captured write-time set, a captured-but-gone peer answers an immediate honest false, and the discovery flush broadcasts the record without ever widening the promise (IN-05).**

## Performance

- **Duration:** ~30min (started ~2026-11-22T23:22:00Z, completed 2026-11-22T23:52:00Z)
- **Tasks:** 2 (Task 1 tracer with a real RED/GREEN cycle; Task 2 regression pinning)
- **Commits:** 4 — measured `git rev-list --count ef7bf46..HEAD` (`0c3a37c` test, `e15f72b` feat, `6e7784e` refactor, `034814e` test)
- **Files changed:** 11 paths (949 insertions, 103 deletions; 1 new test file and 2 RED records created)

## Accomplishments

- **The tracker owns exactly the promise it made.** `tracker` gained `total` and `wants`, both copied from the captured write-time slice when `NotifyLocalWrite` creates it. `applyResults` inserts a `ResultAck` only for a member of `wants`, and `WaitReplicated` computes its count as the intersection of the acked set with that membership and requires `count == tr.total` for success. A stranger's matching Ack still reaches the core's result stream (WR-01) but cannot resolve the wait, cannot increment the count, and cannot help a partial captured set report full coverage (T-06-04).
- **A captured peer that is already gone answers immediately.** The 06-02 `WaitSetEmpty` emission now flows through a tracker scoped to the same capture: the real-socket regression evicts the peer, notifies with the stale capture, and observes `(false, 0)` in 0.31 s against a 2 s caller timeout.
- **IN-05 closes: the discovery flush stops growing the promise.** The flush still resets the ladder, re-arms the retry and enqueues the record send — the newly discovered peer receives the pending set — but the `waitSet[in.NodeID]` insert and `ledgerAdd` are gone. The white-box test holds the invariant's two halves apart (a pending entry with a captured waiter while the table is empty) and asserts the wait-set/ledger stay exactly as captured.
- **The zero-peer release is honest at both layers.** The adapter now returns `(false, 0)` for `total == 0` while still deleting the tracker the notify created; the gRPC handler's public `(false, 0, 0)` is unchanged. A tracker whose captured total is zero is released the same way even if its caller passes a non-zero total.
- **The Superseded fast path is pinned twice.** `TestNotify_supersededResult` proves structurally that the displacing handler call carries `ResultSuperseded` for the older version in its Outbox (no timer, no poll — the 20 ms design target by control flow). `TestWaitReplicated_supersededFast` proves the same path on real sockets in 0.02 s against a 5 s caller timeout, asserting only the generous 250 ms CI bound (RESEARCH Pitfall 2).
- **The integration wiring is proven end to end.** A two-node real-adapter pair with `WritePeers = adapterA.KnownPeerIDs` parks the captured peer's `ApplyRemote` until a third node's Hello has been processed, then releases the Ack: the report is `acked=true 1/1`, so the post-write discovery neither extends the total nor becomes required.
- **All gates green on the committed tree.** `go build ./... && go vet ./...`, repo-wide `go tool golangci-lint run ./...`, and `go test -race -count=1 ./...` (every package) all exit 0.

## Task Commits

1. **Task 1 (RED): the failing captured-set promise tests** — `0c3a37c` (test). Adds `wait_internal_test.go` (ledger/wait-set invariant), the tracker membership/total tests, the captured-but-gone adapter test, and the deliberate updates to every assertion that had pinned the flush's wait-set growth. Both RED runs are recorded and classified `RED_EVIDENCE_OK`.
2. **Task 1 (GREEN): the captured-set promise, the flush split and the honest zero-peer release** — `e15f72b` (feat). `tracker.total`/`wants`, the filtered `applyResults`, the intersection-based `WaitReplicated`, the `(false, 0)` release, the discovery-flush split, and the doc sweep that removes every "discovery adds waiters" claim.
3. **Tracker field documentation** — `6e7784e` (refactor). Groups `wants` with its promise-set contract; no behavior change.
4. **Task 2: the regression matrix** — `034814e` (test). Deterministic Superseded proof, real-socket superseded bound, unknown-Ack inertness, two-peer captured-only completion, strengthened zero-peer release, and the two-node post-write-discovery integration test.

**Plan metadata:** the `docs(06-11)` completion commit (this commit).

## Files Created/Modified

- `internal/udprepl/repl.go` — `tracker.total`/`wants`; `newTracker(peers)`; `ensureTracker(k, peers)`; Ack filtering in `applyResults`; intersection-based count and captured-total success in `WaitReplicated`; `(false, 0)` zero-peer release; doc corrections.
- `internal/app/repl/core.go` — the discovery-flush split (no wait-set insert, no ledger add) and the doc sweep: `pendingEntry`, `NotifyLocalWrite`, `handleHello`, `handleAck`, `evictPeers` and `emitRecordSend` now state that the promise is exactly the write-time capture.
- `internal/app/repl/wait_internal_test.go` — new white-box test proving the flush leaves the wait-set and `peerUnacked` untouched while still queueing the record send and resetting the ladder.
- `internal/app/repl/retransmit_test.go` — `TestNotify_supersededResult` (structural 20 ms proof) and the no-growth `TestRetransmit_suspendAndFlush` expectation.
- `internal/app/repl/peertable_test.go`, `internal/app/repl/restart_test.go` — the return-Hello/discovery wait-set expectations updated to the no-growth promise.
- `internal/udprepl/repl_test.go` — `TestWaitReplicated_{capturedPeerOnly,strangerAckDoesNotResolve,supersededFast,waitSetEmptyImmediate}`; `ackCoverage` and `superseded` now establish discovery before the write so their promise is the captured peer.
- `internal/udprepl/repl_internal_test.go` — `TestWaitReplicated_trackerCapturedPeersOnly` (two subtests) and the strengthened zero-peer release test; `TestNotify_createsTrackerBeforeCore` now proves the pre-Run zero-capture release and the post-discovery successful wait.
- `internal/grpcapi/wait_test.go` — `blockingPeerStore`, `encodeHelloDatagram`, and `TestWait_integrationCapturedPeerOnly`.
- `.planning/phases/06-api-and-semantics/red-evidence/06-11-task1-core.json`, `06-11-task1-tracker.json` — the immutable RED records (commands, exit 1, full console output, targets, expected/actual, semantic assessment).

## Decisions Made

- **The captured total is authoritative, not the caller's.** `WaitReplicated` keeps the `total` parameter for the release short-circuit, but the tracker's copied total governs both the count and the success rule. This is what makes a stranger Ack inert even on a `ResultDrained` and keeps the report honest if a caller ever passes a stale total.
- **The adapter's zero-peer verdict changed from `(true, 0)` to `(false, 0)`** as the plan directs. 06-02 had documented `(true, 0)` as a benign release the handler discards; this plan makes both layers agree on the honest false while preserving the delete.
- **The discovery flush is a broadcast-only operation.** Keeping the ladder reset and `enqueueRecordSend` is what lets a brand-new peer receive the pending set promptly; dropping the wait-set/ledger half is what makes the promise exact. The two responsibilities were previously conflated in `waitSet` (RESEARCH Pitfall 4).
- **The 20 ms target is proven structurally, never measured.** The deterministic Outbox assertion is the proof; the real-adapter test names the target in its comment and asserts a generous 250 ms bound so `-race` scheduler latency cannot flake it.
- **Task 2's tests are regression pins, not a new behavior.** They pass on the Task 1 tree; no second RED commit exists. The plan frontmatter is `type: execute`, and tdd.md's own error handling says a non-failing RED means the feature exists — investigated and recorded here rather than fabricating a RED.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Three more tests pinned the old flush growth**
- **Found during:** the full `internal/app/repl` package run after GREEN
- **Issue:** `TestRestart_firstHelloStillFlushesPending` (not named in the plan's update list) and two internal adapter fixtures still asserted that discovery adds the announcing peer to the wait-set.
- **Fix:** `TestRestart_firstHelloStillFlushesPending` now asserts the flush still queues the record and listing while `WaitingOn` stays empty; `TestWaitReplicated_ackCoverage` and `TestNotify_createsTrackerBeforeCore` now establish discovery before the write so their promise is captured, and the pre-Run write in the latter is exercised as the honest zero-peer release.
- **Files modified:** `internal/app/repl/restart_test.go`, `internal/udprepl/repl_test.go`, `internal/udprepl/repl_internal_test.go`
- **Verification:** `go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./internal/grpcapi/` -> all ok.
- **Commit:** `e15f72b`

**2. [Rule 1 - Bug] A shadowed `acked`/`count` pair tripped the strict lint gate**
- **Found during:** the repo-wide lint run after the Task 1 GREEN edits
- **Issue:** the post-discovery wait goroutine redeclared the outer zero-peer `acked, count` names, failing `govet`'s shadow check.
- **Fix:** renamed the goroutine locals to `waitAcked`/`waitCount`.
- **Files modified:** `internal/udprepl/repl_internal_test.go`
- **Verification:** `go tool golangci-lint run ./...` exit 0.
- **Commit:** `e15f72b`

**3. [Rule 3 - Blocking] The cross-package discovery fixture could not use `EncodeDatagramForTest`**
- **Found during:** writing the grpcapi integration test
- **Issue:** `udprepl.EncodeDatagramForTest` is exported but lives in that package's `_test.go` binary, so it is invisible to `grpcapi_test`; the test did not compile.
- **Fix:** added a small `encodeHelloDatagram` fixture in `wait_test.go` that frames the same production envelope (proto_version 2, node/run ids, CRC32 over the marshaled Body) from the generated schema.
- **Files modified:** `internal/grpcapi/wait_test.go`
- **Verification:** `go test -race -count=1 ./internal/grpcapi/ -run '^TestWait_integration'` -> PASS.
- **Commit:** `034814e`

## TDD Gate Compliance

- **Task 1 RED:** `0c3a37c` — `test(06-11)`. Evidence `red-evidence/06-11-task1-core.json` and `06-11-task1-tracker.json`, both `RED_EVIDENCE_OK` (`target_test_failed`: 4 failing tests each; targets `TestPeerTable_discoveryFlushKeepsPromise` and `TestWaitReplicated_trackerCapturedPeersOnly`). The failures were planned assertion failures — wait-set growth, inflated counts, partial-set success and the `(true, 0)` release — not build or fixture faults.
- **Task 1 GREEN:** `e15f72b` — `feat(06-11)`. Both verify commands exit 0.
- **Task 1 REFACTOR:** `6e7784e` — `refactor(06-11)`, documentation-only, tests unchanged and green.
- **Task 2:** the plan marks the task `tdd="true"`, but it is explicitly the regression-pinning half ("The existing Superseded path is pinned"). Its tests pass unchanged on the Task 1 tree, so there is no RED commit; tdd.md's "test doesn't fail in RED phase: the feature may already exist — investigate" is satisfied by inspection and recorded here. The plan's own frontmatter is `type: execute`, not `type: tdd`.

## Issues Encountered

- **The plan's `must_haves` describe a discovery flush that cannot fire while a captured promise is live.** The flush triggers only when a Hello ends an *empty* peer table; a non-empty captured wait-set implies its captured peer is still in the table, and evicting it empties the wait-set and emits `WaitSetEmpty`. The white-box test therefore installs the split-apart state directly (captured waiter + empty table) to test the invariant, and the end-to-end integration test proves the complementary property: a post-write discovery cannot move the reported total or become required. Both halves are covered; the literal "wait-set stays `{1}`" scenario is unreachable through the public seams and is documented instead of faked.
- **The acceptance grep for `wants map[uint16]struct{}` is sensitive to gofmt alignment.** The field was grouped on its own line (with the promise-set contract beside it) so the literal text exists while the struct stays gofmt-clean; this is why the small `refactor` commit exists.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 06-05.** A-01 is now complete end to end: the captured set governs both the report and the flush, every return path releases its tracker, and the regression matrix is green under `-race`.
- **Plan 06-13 owns the phase gate.** It will re-run `make lint && make test`, `bash scripts/red-runner`, and the frozen-file base diff; this plan verified the full `go test -race ./...` suite and repo-wide lint on its committed tree.
- **Defects recorded:** none new. The earlier `roadmap update-plan-progress` v1.0-row defect remains in the phase's `deferred-items.md`; it is a tooling issue outside this plan's scope.

## Self-Check: PASSED

- Created files exist: `.planning/phases/06-api-and-semantics/06-11-SUMMARY.md`, `internal/app/repl/wait_internal_test.go`, `red-evidence/06-11-task1-core.json`, `red-evidence/06-11-task1-tracker.json` — all FOUND.
- Task commits are ancestors of HEAD: `0c3a37c`, `e15f72b`, `6e7784e`, `034814e` — all FOUND.
