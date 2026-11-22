---
phase: 04-protocol
plan: 06
subsystem: replication-protocol
tags: [proto-08, totality, dump-failed-notify, snapshot-load-notify, restored-records, track-false, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 03
    provides: "the generation/install convergence point whose restore replay the engine reports"
  - phase: 04-protocol
    plan: 02
    provides: "the gate-first handler shape and the harness this plan's notify rows extend"
  - phase: 03-transport
    provides: "the LocalNotify seam, the paced egress queue and the pending ledger the notified records enter"
provides:
  - "persist.RestoredRecord and Engine.Restore returning the replayed records in sorted-key order, even when the final flush failed"
  - "replayRecord's explicit (version, replayed, error) signal, distinct from the tombstone flag"
  - "dump-failed Set/SetIfAbsent/Delete notifications with track=false before Unavailable"
  - "SnapshotLoad's replay notifications, one per replayed record, sorted and track=false, even on a failed final flush"
affects:
  - "Phase 5's wait_for_replication report and Phase 6's APIS-07 full-rollback load reuse the RestoredRecord return shape"
  - "04-07+ protocol plans inherit a totality-tested replication seam"

actuals:
  tokens: 8816
  tasks: 2
  commits: 4
  plan_head_before: 9ead5cea39cf27fd13625307979cbb143c2a309e
  plan_head_after: cacabdbbc4317f5b3616e10529b13f3bc386becd

tech-stack:
  added: []
  patterns:
    - "Report-what-installed: Restore returns exactly the records the replay installed, sorted, with skips absent, so the notify order is deterministic and never announces a version the store did not install."
    - "Error honesty with totality: the handler notifies what is in memory, then maps the error; ErrDumpFailed notifies, ErrClosed never does."
    - "A failed final dump surfaces as ErrDumpFailed wrapping its cause, so the RPC maps to Unavailable while errors.Is still reaches the underlying port error."

key-files:
  created:
    - internal/app/persist/engine_replay_internal_test.go
    - .planning/phases/04-protocol/red-evidence/04-06-task1.json
    - .planning/phases/04-protocol/red-evidence/04-06-task2.json
  modified:
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - internal/grpcapi/server.go
    - internal/grpcapi/server_test.go

key-decisions:
  - "Restore reports the replayed records even when the final flush failed, and converts a non-ErrClosed flush failure into ErrDumpFailed wrapping the cause: the behavior row and prohibition verification require ErrDumpFailed, while the cause stays inspectable with errors.Is; ErrClosed passes through unchanged so shutdown is never reported as a dump failure."
  - "replayRecord's bool is the explicit install/skip signal; the tombstone flag travels only in RestoredRecord.Deleted, so a live success (replayed=true, nonzero version) and either skip shape (replayed=false, zero version, nil error) can never read the same."
  - "A restore's replay skips — a tombstone with nothing to beat and a value over the shrunk store limit — are absent from the returned list and therefore never notified; the whole-restore no-op returns an empty list and dumps nothing."
  - "The handler notifies dump-failed writes before mapError answers Unavailable, with track=false even when the request asked for a wait report: the RPC errored, so no report exists. SetIfAbsent rides the same branch."
  - "SnapshotLoad notifies from Restore's returned list before mapping the error and skips the loop entirely on ErrClosed; the no-op load notifies nothing and returns OK."
  - "The notify order is the replay's sorted-key order, which the test asserts as an exact key sequence rather than a count, pinning determinism across runs."

patterns-established:
  - "Totality at the API seam: notify first, report the failure second — a write or load that exists in memory always reaches peers, while the RPC never claims success it cannot confirm."
  - "Signal separation: an install/skip bool and a tombstone flag travel separately, so no caller can mistake one for the other."

requirements-completed: [PROTO-08]

coverage:
  - id: D1
    description: "The engine's replay report: RestoredRecord, Restore returning sorted freshly stamped records (plus the error) even on a failed final flush, a no-op returning an empty list, and replayRecord's explicit install/skip signal with skips absent."
    requirement: PROTO-08
    verification:
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_restoreReturnsReplayed"
        status: pass
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_restoreNoOpReturnsEmpty"
        status: pass
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_restoreFailedFlushReturnsReplayed"
        status: pass
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_restoreSkipsReplay"
        status: pass
      - kind: unit
        ref: "internal/app/persist/engine_replay_internal_test.go#TestEngine_replayRecordSignal"
        status: pass
    human_judgment: false
  - id: D2
    description: "Dump-failed writes and deletes notify the stamped record with track=false before answering Unavailable; SetIfAbsent rides the same branch; ErrClosed notifies nothing."
    requirement: PROTO-08
    verification:
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_dumpFailedWriteNotifies"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_dumpFailedDeleteNotifies"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_dumpFailedCreateIfAbsentNotifies"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_closedWriteDoesNotNotify"
        status: pass
    human_judgment: false
  - id: D3
    description: "SnapshotLoad notifies every replayed record in sorted key order with track=false, including when the confirming dump failed; a no-op load notifies nothing and returns OK."
    requirement: PROTO-08
    verification:
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_snapshotLoadNotifies"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_snapshotLoadNoOpDoesNotNotify"
        status: pass
    human_judgment: false
  - id: D4
    description: "No snapshot semantics changed: existing restore, no-op, dump-failure, missing-snapshot and corrupt-snapshot suites stay green."
    requirement: PROTO-08
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ ./internal/grpcapi/"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 06: Total Replication of Unconfirmed Writes and Snapshot Loads Summary

**Replication is now total: a dump-failed write, its delete, and every record a snapshot load replays all reach the notify seam — with the RPC still reporting the dump failure honestly.**

## Performance

- **Duration:** 5min (294s)
- **Started:** 2026-11-22T04:37:35Z
- **Completed:** 2026-11-22T04:42:29Z
- **Tasks:** 2
- **Files modified:** 5 code/test files (+2 RED evidence records)

## Accomplishments

- `Engine.Restore` now returns `[]RestoredRecord` — key, fresh store-stamped version, data and tombstone flag — in deterministic sorted-key order, including when the confirming dump failed. A whole-restore no-op returns an empty list and performs no dump.
- `replayRecord` returns `(version.RecordVersion, replayed bool, error)`: an explicit install/skip signal separate from the tombstone flag. A tombstone with nothing to beat and a value over the shrunk store limit are skips with the zero version and nil error; a live success and a skip can never read the same.
- A failed final flush in `Restore` surfaces as `ErrDumpFailed` wrapping the port error, so the handler maps it to Unavailable while the cause stays reachable; `ErrClosed` passes through unchanged.
- `Set`, `SetIfAbsent` and `Delete` notify the stamped record before answering Unavailable when the dump failed (`track=false`); `ErrClosed` notifies nothing.
- `SnapshotLoad` consumes the replayed-record list and notifies each record in sorted order with `track=false`, including on a failed final flush; a no-op load notifies nothing and returns OK.
- Six handler rows assert the exact call sequence and flags — not merely counts — so a notification for the wrong record, order, or track flag fails.

## Task Commits

Each task followed TDD RED → GREEN and was committed atomically:

1. **Task 1 RED: restore-report tests and return-shape ripple** - `51ccde5` (test)
2. **Task 1 GREEN: return the replayed records from Restore** - `5397974` (feat)
3. **Task 2 RED: totality notify tests and harness capture** - `71b28bd` (test)
4. **Task 2 GREEN: notify unconfirmed writes and snapshot loads** - `cacabdb` (feat)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `internal/app/persist/engine.go` — `RestoredRecord`, `Restore`'s report and `ErrDumpFailed` conversion, `replayRecord`'s install/skip signal.
- `internal/app/persist/engine_test.go` — eight call sites moved to the two-value form; `snapshotFixture` and `installSnapshot` helpers; four restore-report rows.
- `internal/app/persist/engine_replay_internal_test.go` — white-box `TestEngine_replayRecordSignal` (the `_internal_test.go` testpackage exemption).
- `internal/grpcapi/server.go` — dump-failed notify branches in `Set`/`Delete`, the `SnapshotLoad` replay notify loop, and their contract docs.
- `internal/grpcapi/server_test.go` — `fakeDumpStore.snapshot`, `testServerConfig.notify`, the mutex-guarded `notifyCapture`, and six totality rows.
- `.planning/phases/04-protocol/red-evidence/04-06-task{1,2}.json` — persisted RED evidence, both `RED_EVIDENCE_OK`.

## Decisions Made

- **The failed final dump is reported as `ErrDumpFailed`.** The plan's action line says "plus the flush error" while its behavior row and prohibition verification require `ErrDumpFailed` (the handler must answer Unavailable). The implementation satisfies both: `fmt.Errorf("%w: %w", ErrDumpFailed, flushErr)` keeps the cause inspectable and makes `errors.Is` match the sentinel. `ErrClosed` is never wrapped, preserving the shutdown contract.
- **Skips are not errors and not notifications.** `ErrNotFound` on a tombstone replay and `ErrValueTooLarge` on a live replay return the zero version, `replayed=false` and nil; `Restore` omits them from the list, so the handler cannot announce a version the store never installed (T-4-19).
- **Notify before mapError.** Every totality path notifies first and maps the error second; the error path's track flag is always false because the RPC errors (no wait report exists). This is the same order the plan's prohibition demands.
- **Sorted order is the notify contract.** Tests compare the exact key sequence (`snap/alpha`, `snap/beta`, `snap/gamma`), so a map-order regression fails rather than passing on a count.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `replayRecord` is unexported, so its direct-signal test cannot live in `engine_test.go`**
- **Found during:** Task 1 RED
- **Issue:** The plan names `TestEngine_replayRecordSignal` in `engine_test.go`, which is `package persist_test` (external) and cannot call the unexported `replayRecord`.
- **Fix:** Added `internal/app/persist/engine_replay_internal_test.go` in `package persist` with the documented `_internal_test.go` testpackage exemption, matching the `fsops_test.go`/`egress_internal_test.go` precedent. `TestEngine_restoreSkipsReplay` stays external and proves the skip rows through `Restore`.
- **Files modified:** `internal/app/persist/engine_replay_internal_test.go`
- **Verification:** `go test -race ./internal/app/persist/ -run '^TestEngine_replayRecordSignal$'` green; lint clean.
- **Committed in:** `51ccde5`

**2. [Rule 3 - Blocking] `server.go`'s single `Restore` caller needed the two-value update in Task 1**
- **Found during:** Task 1 RED scaffold
- **Issue:** Changing `Restore`'s signature breaks `SnapshotLoad`, and Task 1's verify includes `go build ./...`.
- **Fix:** Updated the call site to `_, err :=` in the RED commit so the module compiles; Task 2 replaced it with the replay-notify loop.
- **Files modified:** `internal/grpcapi/server.go`
- **Verification:** `go build ./... && go vet ./...` green at Task 1 GREEN.
- **Committed in:** `51ccde5` (ripple), `cacabdb` (behavior)

**3. [Rule 1 - Bug] Plan wording conflict on the failed-flush error**
- **Found during:** Task 1 GREEN
- **Issue:** The action says "return the ... flush error" while the behavior row and prohibition verification require `ErrDumpFailed`; the raw port error would map to Internal, not Unavailable.
- **Fix:** Wrapped the flush error with `ErrDumpFailed` (cause preserved), `ErrClosed` passthrough. See Decisions Made.
- **Files modified:** `internal/app/persist/engine.go`
- **Verification:** `TestEngine_restoreFailedFlushReturnsReplayed` and `TestServer_snapshotLoadNotifies` green.
- **Committed in:** `5397974`

---

**Total deviations:** 3 auto-fixed (2 Rule 3, 1 Rule 1)
**Impact on plan:** No behavior was relaxed. The internal test file is a placement adaptation; the error-shape decision reconciles the plan's own contradictory lines in favor of its behavior/prohibition rows.

## TDD Gate Compliance

Both tasks followed RED → GREEN with persisted evidence; `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for each record:

- **Task 1:** RED `51ccde5` — `04-06-task1.json` records `TestEngine_restoreReturnsReplayed` failing on its planned assertion (empty list versus `[alpha beta delta]`); the same run showed `replayRecord` reporting `replayed=false`/zero version and the raw fake save error instead of `ErrDumpFailed`. The no-op and skip rows passed by construction as preserved-behavior guards. GREEN `5397974` made all five rows pass.
- **Task 2:** RED `71b28bd` — `04-06-task2.json` records `TestServer_dumpFailedWriteNotifies` failing with an empty notify capture; the delete, create-if-absent and snapshot-load rows failed the same way. The closed-store and no-op rows passed as required zero-notify guards. GREEN `cacabdb` made all six rows pass.
- **REFACTOR:** none separately — the implementation was already minimal and lint clean.

## Issues Encountered

- The plan's `read_first` names `TestServer_errorCodes`; the current harness test is `TestServer_errorMapping` (the `ts.cancelEngine()` path it cites). No action — the citation is stale, the path exists.
- None beyond the three documented deviations. Both packages are green under `-race`, `go build ./...` and `go vet ./...` pass, and `go tool golangci-lint run ./internal/app/persist/ ./internal/grpcapi/` is clean.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- PROTO-08's totality is implemented and covered on both halves: unconfirmed writes/deletes and snapshot-load replays reach the notify seam, while ErrClosed and no-op paths stay silent.
- No new dependencies; the threat register's T-4-18/T-4-19 mitigations are the tested notify paths, and T-4-20 rides the existing enqueue-and-return seam unchanged.
- 7 plans remain in Phase 4. Ready for `04-07-PLAN.md`.

## Self-Check: PASSED

- `internal/app/persist/engine_replay_internal_test.go` and both RED evidence records exist.
- All four task commits (`51ccde5`, `5397974`, `71b28bd`, `cacabdb`) are ancestors of HEAD.
- `persist.RestoredRecord`, the new `Restore`/`replayRecord` signatures and both handler notify paths are present and test-green.
