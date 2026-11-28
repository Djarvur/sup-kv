---
phase: 06-api-and-semantics
plan: 08
subsystem: snapshot-rollback
tags:
  - apis-07
  - snapshot-load
  - full-rollback
  - fresh-tombstone
  - sorted-union
  - notify-totality
  - no-op-boundary
requires:
  - phase: 06-api-and-semantics (06-12)
    provides: the confirmed-watermark Watch gate and the SnapshotLoad notify path (PROTO-08)
  - phase: 06-api-and-semantics (06-04)
    provides: the total path-free error mapper SnapshotLoad answers through
  - phase: 04-protocol (04-06)
    provides: Restore's RestoredRecord report and the handler that notifies exactly that list
provides:
  - "internal/app/persist/snapshot.go: full-rollback Restore — restoreChanges counts an extra live key, replayKeys builds the sorted union, rollbackKey stamps the fresh tombstone"
  - "the rewritten TestEngine_snapshotCreateLoadRestore and TestEngine_restoreReturnsReplayed plus TestEngine_restoreExtraLiveKeyIsNotNoOp"
  - "internal/grpcapi/rollback_test.go: TestServer_snapshotLoadRollback (sorted union, failed-dump totality, no-op silence)"
affects:
  - 06-09-api-docs
  - 06-10-docs
  - 06-13-phase-gate
actuals:
  tokens: 8648
  tasks: 3
  commits: 5
  plan_head_before: f404573f8bf8fb6947b99380c5d045612b0c8672
  plan_head_after: 339b06ba9664bbb79c2d0cb88c93b49a0849cc99
tech-stack:
  added: []
  patterns:
    - "A full rollback is one deterministic sorted union of snapshot keys and current live keys absent from the snapshot, walked once: a snapshot key replays, an absent live key is deleted with a fresh tombstone and joins the returned report."
    - "The rollback result needs no handler wiring: SnapshotLoad already notifies exactly the record list Restore returns, so joining rollback tombstones to that list is the whole replication path (PROTO-08)."
    - "A no-op gate must count the work the pass would do, not only the snapshot's keys: an extra live key is a change, an extra tombstone is not."
key-files:
  created:
    - internal/grpcapi/rollback_test.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-08-task1.json
  modified:
    - internal/app/persist/snapshot.go
    - internal/app/persist/engine_test.go
key-decisions:
  - "restoreChanges returns true when any snapshot record differs OR any current record with Deleted==false is absent from the snapshot; an extra current tombstone is deliberately not a change, so an already-rolled-back key burns no version and produces no record (Pitfall 14)."
  - "replayKeys returns the sorted union of the snapshot's keys and the current live keys the snapshot does not name; replaySnapshot walks it once, so replay and rollback share one deterministic notify order."
  - "A rollback delete goes through storage.Store.Delete, whose ErrNotFound means a concurrent writer tombstoned the key first: the rollback outcome already holds, no version is burned, and no record is reported (T-06-27)."
  - "Rollback tombstones join RestoredRecord with nil Data and Deleted=true, so the existing SnapshotLoad notify loop broadcasts them with zero handler changes — including after a failed final dump, because the handler notifies before mapping the error (T-06-26)."
  - "The extra-live-key test asserts the rollback tombstone inside the reported sorted union (keep-a, keep-b, extra) rather than as the list's only element: D-07 step 3 replays every snapshot record on any change, so the plan's 'exactly that tombstone' wording is realized as 'exactly the extra key's tombstone in the union'."
  - "Tasks 2 and 3 were investigated unexpected GREENs: their asserted behaviour landed in Task 1's GREEN commit and the handler needed no change, so the two test commits are the deliverables and no RED was fabricated (04-07 / 06-04 precedent)."
  - "The single replay function tripped gocognit (25 > 20), nestif and revive's flag-parameter rule, so the honours-shaped refactor split it into replayKeys, replaySnapshotKey and rollbackKey with no behaviour change."
  - "APIS-07's REQUIREMENTS.md checkbox stays pending because requirements ready-ids reports it blocked (the #2388 shared-ID gate among 06-08/06-09/06-10/06-13)."
patterns-established:
  - "A rollback pass proves itself twice: an engine test pins the returned tombstones and the store outcome, and an RPC test pins the same records crossing the notify seam, so a handler regression cannot hide behind a correct engine."
  - "Boundary tests come in a control pair: the extra-live-key load must change the store and dump, while the equal-state load must return an empty list and write nothing."
requirements-completed: []
coverage:
  - id: D1
    description: "A snapshot load is a full rollback: every current live key absent from the snapshot is deleted with a fresh tombstone that dominates every pre-restore version, already-tombstoned absent keys keep their version and stay out of the report, and the pass is one deterministic sorted union with the partial-list-plus-error behavior on a mid-pass failure."
    requirement: "APIS-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_snapshotCreateLoadRestore$' -v -> PASS"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_restoreSkipsReplay$' -v -> PASS"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_restore' -v -> 7 PASS"
        status: pass
      - kind: process
        ref: "grep -q 'rec.Deleted' internal/app/persist/snapshot.go -> exit 0; restoreChanges' second loop returns true for an extra current live key"
        status: pass
    human_judgment: false
  - id: D2
    description: "The engine tests pin the rollback report: TestEngine_restoreReturnsReplayed expects alpha/beta/delta/gamma with gamma a fresh tombstone, TestEngine_restoreExtraLiveKeyIsNotNoOp proves an otherwise-matching snapshot still deletes the extra live key and performs the final dump, and TestEngine_restoreNoOpReturnsEmpty remains the empty-list no-dump control."
    requirement: "APIS-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_restore(ReturnsReplayed|ExtraLiveKeyIsNotNoOp|NoOpReturnsEmpty)$' -v -> 3 PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "The RPC test proves the rollback result broadcasts through the unchanged SnapshotLoad notify path: the extra key arrives as Deleted=true in the sorted union of replay and rollback keys, the same notifications survive a failed final dump that answers Unavailable, and a no-op load notifies nothing."
    requirement: "APIS-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_snapshotLoadRollback$' -v -> 3 subtests PASS"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_snapshotLoad' -v -> 5 tests PASS"
        status: pass
    human_judgment: false
  - id: D4
    description: "The repo-wide AGENTS.md gates stay green on the committed tree with the rollback semantics in place: no lint finding and no test regression in any package, including the stand and cmd suites that exercise snapshot loads."
    requirement: "APIS-07"
    verification:
      - kind: process
        ref: "make lint -> exit 0"
        status: pass
      - kind: process
        ref: "make test (go test -race -cover ./...) -> exit 0, all packages ok"
        status: pass
    human_judgment: false
duration: 12min
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 08: Snapshot Load Is a Full Rollback Summary

**A snapshot load now makes the snapshot the state: every live key absent from it is deleted with a fresh tombstone that joins the returned record list, so the unchanged `SnapshotLoad` notify path replicates the rollback — even when the final dump fails.**

## Performance

- **Duration:** 12 min (705 s)
- **Started:** 2026-11-23T01:15:07Z
- **Completed:** 2026-11-23T01:26:52Z
- **Tasks:** 3
- **Files modified:** 4 (2 code/test, 1 new RPC test, 1 RED record)
- **Commits:** 5 measured from ledger `f404573..339b06b`

## Accomplishments

- `internal/app/persist/snapshot.go`: `restoreChanges` now treats an extra current **live** key absent from the snapshot as a change (an extra tombstone stays a non-change), `replayKeys` builds the deterministic sorted union of snapshot keys and rollback keys, `replaySnapshotKey` replays a snapshot record through the existing Set/Delete rules, and `rollbackKey` deletes an absent live key with a fresh tombstone — `storage.ErrNotFound` there is a benign concurrent skip that burns no version.
- The rollback tombstones join `RestoredRecord` with nil `Data` and `Deleted=true`, so `SnapshotLoad` broadcasts them through the notify path it already had; no handler code changed, and the failed-final-dump path still announces everything in memory before answering `Unavailable`.
- `TestEngine_snapshotCreateLoadRestore` was rewritten for decision 14.1 / ADR 0001: `post-key` is deleted and returned as a dominating tombstone, `gone` keeps its pre-restore version and is absent from the report, `re-tomb`/`resurrect` keep their assertions, and the recovery-from-scratch half still ends with the three live snapshot keys.
- `TestEngine_restoreReturnsReplayed` now expects `alpha/beta/delta/gamma` with `gamma` a fresh tombstone; `TestEngine_restoreExtraLiveKeyIsNotNoOp` proves a snapshot that otherwise matches the store still rolls back the extra key and performs the final dump; `TestEngine_restoreNoOpReturnsEmpty` remains the empty-list, no-dump control.
- `internal/grpcapi/rollback_test.go` adds `TestServer_snapshotLoadRollback` with three subtests: the sorted union `snap/alpha, snap/mike (tombstone), snap/zulu` crosses the notify seam, the same notifications survive `failSave` plus `codes.Unavailable`, and an equal-state load notifies nothing.

## TDD Evidence

| Task | RED record | RED observation | Target |
|------|------------|-----------------|--------|
| 1 | `red-evidence/06-08-task1.json` | the additive implementation left `post-key` live (version equal to, not dominating, the pre-restore max) and returned no rollback tombstone | `TestEngine_snapshotCreateLoadRestore` |
| 2 | none — investigated unexpected GREEN | the rewritten report and boundary tests passed on the first run because Task 1's GREEN commit is the behaviour they pin | `TestEngine_restoreReturnsReplayed`, `TestEngine_restoreExtraLiveKeyIsNotNoOp` |
| 3 | none — investigated unexpected GREEN | the handler already notifies exactly the list `Restore` returns, so Task 1's rollback tombstones are the whole change | `TestServer_snapshotLoadRollback` |

Task 1's record passed `gsd_run check tdd-red-evidence` (`RED_EVIDENCE_OK`, reason `target_test_failed`). Tasks 2 and 3 are test-only deliverables whose production half is Task 1's; no RED record or artificial failure was fabricated (04-07 / 06-04 precedent).

## Task Commits

1. **Task 1 RED: pin snapshot-load full rollback** — `cbb58a5` (test)
2. **Task 1 GREEN: make a snapshot load a full rollback** — `05dff39` (feat)
3. **Task 2: pin the rollback reports and the extra-live-key boundary** — `8bff02a` (test)
4. **Refactor: split the replay pass into per-case helpers** — `7a20e04` (refactor)
5. **Task 3: prove the rollback broadcasts through SnapshotLoad** — `339b06b` (test)

**Plan metadata:** the final `docs(06-08)` commit.

## Files Created/Modified

- `internal/app/persist/snapshot.go` — full-rollback `restoreChanges`, the `replayKeys` union, `replaySnapshotKey`/`rollbackKey`, and the `Restore`/`RestoredRecord`/`replaySnapshot` doc rewrites.
- `internal/app/persist/engine_test.go` — the rewritten `TestEngine_snapshotCreateLoadRestore` and `TestEngine_restoreReturnsReplayed`, the new `TestEngine_restoreExtraLiveKeyIsNotNoOp`, and the no-op control.
- `internal/grpcapi/rollback_test.go` — `TestServer_snapshotLoadRollback` and its fixture/seeding helpers (new sibling; `server_test.go` untouched).
- `.planning/phases/06-api-and-semantics/red-evidence/06-08-task1.json` — the machine-validated RED record for Task 1.

## Decisions Made

- The rollback set is computed once from `e.store.Records()` and walked as the sorted union with the snapshot's keys, so the notify order is stable and the no-op gate and the pass agree on the same view.
- Only **live** current keys are deleted; an already-tombstoned absent key is left exactly as it is, so a load never burns a version or publishes a redundant tombstone.
- Concurrent writes after the rollback set is computed are ordinary LWW, not snapshot isolation; the `Restore` doc comment now states that explicitly.
- The extra-live-key test asserts the tombstone inside the reported union rather than as the only element: any change replays the snapshot records too, per D-07 step 3.
- The task-2/task-3 tests are regression pins committed as deliverables after investigating their first-run GREEN, matching the 04-07/06-04 handling.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The single replay function tripped three lint gates**
- **Found during:** the pre-commit lint run after Task 3's test was written
- **Issue:** `gocognit` reported cognitive complexity 25 (> 20) for `replaySnapshot`, `nestif` flagged the nested `if inSnapshot` block, and `revive`'s `flag-parameter` rejected the `inSnapshot bool` argument of the intermediate helper. The repo's strict lint config is a hard gate (`make lint`).
- **Fix:** split the pass into `replayKeys` (the sorted union), `replaySnapshotKey` (one snapshot install) and `rollbackKey` (one rollback delete), with the caller selecting by map membership instead of a flag. Behaviour is unchanged; both packages' suites and the full `make test` stayed green.
- **Files modified:** `internal/app/persist/snapshot.go`, `internal/app/persist/engine_test.go`
- **Commit:** `7a20e04`

**2. [Rule 3 - Blocking] `noinlineerr` rejected the inline error checks in the new tests**
- **Found during:** the same lint run
- **Issue:** the vendored strict config's `noinlineerr` rule rejects `if _, err := f(); err != nil` in the new test code.
- **Fix:** hoisted each call to a plain assignment followed by the `if`, in both `engine_test.go` and `rollback_test.go`.
- **Files modified:** `internal/app/persist/engine_test.go`, `internal/grpcapi/rollback_test.go`
- **Commit:** `7a20e04` (engine test), `339b06b` (RPC test)

### Documented Interpretation

**3. Task 2's "returns exactly that tombstone" versus D-07's full replay**
- **Found during:** Task 2
- **Issue:** the plan's `<behavior>` says the extra-live-key restore "returns exactly that tombstone", but the plan's own Task 1 action and D-07 step 3 replay **every** snapshot record whenever any change is detected, so the report necessarily also carries the re-stamped matching records.
- **Resolution:** the mandated implementation governs. `TestEngine_restoreExtraLiveKeyIsNotNoOp` asserts the full sorted union and pins the extra key's tombstone properties (Deleted, nil data, dominating version, `Get` NotFound) plus the final dump. No production behavior was bent to the prose.

## Issues Encountered

- **APIS-07's checkbox stays pending.** `gsd_run query requirements.ready-ids` reports APIS-07 as `blocked` because 06-08/06-09/06-10/06-13 all declare it (the #2388 shared-ID gate), so no requirement checkbox was flipped. The phase gate (06-13) closes it.
- **No other issues.** `make lint` and `make test` (the full `-race -cover` suite, including `internal/stand` and `cmd/sup-kv`) pass on the committed tree.

## User Setup Required

None - no external service configuration required.

## Self-Check: PASSED

- Created files exist: `internal/grpcapi/rollback_test.go`, `red-evidence/06-08-task1.json`.
- Commits exist: `cbb58a5`, `05dff39`, `8bff02a`, `7a20e04`, `339b06b`.
- Measured commits: 5 from ledger `f404573..339b06b`.

## Next Phase Readiness

- Full-rollback snapshot-load semantics are in place and replicated by the existing notify path; the engine and RPC proofs are committed.
- 3 plans remain in Phase 6 (06-09, 06-10, 06-13). Ready for 06-09.
