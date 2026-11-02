---
phase: 03-persistence-dump-and-write-confirmation
plan: 02
subsystem: persistence
tags: [go, snapshots, restore-as-writes, lww, path-traversal, name-validation, write-confirmation, synctest, tdd]

requires:
  - phase: 03-persistence-dump-and-write-confirmation
    provides: port.DumpStore snapshot seam (CreateSnapshot/ReadSnapshot, validated names), persist engine flushCh wiring, persistfile adapter
provides:
  - Engine.Snapshot (forced flush then adapter byte-copy, D-10) and Engine.Restore (replay-as-ordinary-writes with fresh versions, D-09) — PERS-05 complete
  - Snapshot name-validation security table, atomic overwrite and error-class pins in internal/persistfile
  - synctest-pinned flush-forces-dump semantics against the real dumper loop
affects: [05-grpc-api, 06-hardening]

actuals:
  tokens: 7367   # chars/4 over the realized diff (615 added / 12 removed lines); plan estimate 32000 at confidence:med
  tasks: 2
  commits: 3      # measured: git rev-list --count d93e75a..HEAD

plan_head_before: d93e75afa77f7d1f08ee98128207f2a4c8062f94
plan_head_after: c34d8c5677bcd80de8ad153ec5e2d568e75914cc

tech-stack:
  added: []        # zero new dependencies — stdlib + in-repo only (T-3-SC)
  patterns:
    - forced flush as a reply-channel handshake over flushCh with a clean-engine no-dump fast path
    - restore-as-writes: replay through the store's ordinary write path, one batched confirming flush
    - validation-before-filesystem-touch proven by error-class ordering probes (sentinel, never ErrNoDump)

key-files:
  created:
    - internal/persistfile/snapshot_test.go
  modified:
    - internal/app/persist/engine.go
    - internal/app/persist/doc.go
    - internal/app/persist/engine_test.go

key-decisions:
  - "Flush semantics: Run services a flush request with an immediate dump only when dirty — a clean engine replies nil without a redundant save (D-10's literal reading, pinned by TestEngine_snapshotFlushForcesDump)"
  - "Restore replays via store.Set/store.Delete only (never Merge), in sorted-key order, with ErrNotFound and ErrValueTooLarge skips mirroring Open; the batched final flush carries flagged assumption 1 to phase verification"
  - "The exit sweep resolves flush requests already handed to the exiting Run with ErrClosed — no stranded Snapshot caller"
  - "Task 2 landed as one test commit: the behavior under test was already implemented (adapter in 03-01, flush in Task 1), so the suite is a regression lock — 03-01 Task 3 precedent"

patterns-established:
  - "Clean-engine fast paths are part of the flush seam: forcing a dump means forcing it when state is unconfirmed, never a redundant save"
  - "Security tables prove ordering, not just rejection: the sentinel must appear even when a later filesystem error (ErrNoDump, fs.ErrNotExist) would otherwise mask it"

requirements-completed: [PERS-05]

coverage:
  - id: D1
    description: "A named snapshot is a byte-exact copy of the confirmed on-disk main dump, created only after a covering dump completes (dirty state forced, clean state straight-copied)"
    requirement: PERS-05
    verification:
      - kind: integration
        ref: "internal/app/persist/engine_test.go#TestEngine_snapshotCreateLoadRestore"
        status: pass
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_snapshotFlushForcesDump"
        status: pass
    human_judgment: false
  - id: D2
    description: "Restore replays exactly the snapshot's keys with fresh dominating versions (snapshot-live beats a current tombstone, snapshot tombstone beats a current live value, tombstone no-ops burn no version); keys absent from the snapshot keep their state"
    requirement: PERS-05
    verification:
      - kind: integration
        ref: "internal/app/persist/engine_test.go#TestEngine_snapshotCreateLoadRestore"
        status: pass
    human_judgment: false
  - id: D3
    description: "A fresh engine with only the carried-over snapshot directory restores every live snapshot key (recovery from scratch); snapshot tombstones are ErrNotFound no-ops there"
    requirement: PERS-05
    verification:
      - kind: integration
        ref: "internal/app/persist/engine_test.go#TestEngine_snapshotCreateLoadRestore"
        status: pass
    human_judgment: false
  - id: D4
    description: "No snapshot name escapes .snapshots: both operations reject every traversal, separator, dot-special, out-of-charset and 65-char shape with ErrInvalidSnapshotName before any filesystem touch (64 passes the boundary)"
    requirement: PERS-05
    verification:
      - kind: integration
        ref: "internal/persistfile/snapshot_test.go#TestFileStore_nameValidation"
        status: pass
    human_judgment: false
  - id: D5
    description: "Re-creating an existing snapshot name atomically overwrites in place with the later dump's bytes — no .prev sibling accumulates"
    verification:
      - kind: integration
        ref: "internal/persistfile/snapshot_test.go#TestFileStore_snapshotOverwriteAtomic"
        status: pass
    human_judgment: false
  - id: D6
    description: "Error classes stay distinguishable: snapshotting with no dump is ErrNoDump, a missing snapshot read wraps fs.ErrNotExist, and a failed Restore changes nothing"
    verification:
      - kind: integration
        ref: "internal/persistfile/snapshot_test.go#TestFileStore_snapshotErrorClasses"
        status: pass
      - kind: integration
        ref: "internal/app/persist/engine_test.go#TestEngine_restoreMissingSnapshot"
        status: pass
    human_judgment: false
  - id: D7
    description: "Phase 3 exit state: make lint and make test green repo-wide, go.mod/go.sum byte-identical, internal/port exactly doc.go and dump.go — PERS-01..PERS-05 all covered by automated tests"
    verification:
      - kind: other
        ref: "make lint && make test (full gate, same targets as CI)"
        status: pass
    human_judgment: false

duration: 16min
completed: 2026-11-02
status: complete
---

# Phase 3 Plan 2: Dump and Write Confirmation Summary

**Engine.Snapshot (forced flush through the dumper, then the adapter's byte-copy of the confirmed main dump) and Engine.Restore (same-codec decode, sorted-key replay as fresh-version ordinary writes with one batched confirming flush), plus the adapter's name-validation security table, error classes and synctest-pinned flush semantics — closing PERS-05 and with it Phase 3.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-11-02T23:20:22Z
- **Completed:** 2026-11-02T23:36:25Z
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- Snapshot works end-to-end against the real filesystem: `Snapshot(name)` forces the current dump to complete (a dirty engine is dumped immediately through the flushCh seam; a clean engine skips the redundant save) and the produced file is byte-identical to the main dump at creation (PERS-05, D-10)
- Restore replays exactly the snapshot's keys through the store's ordinary write path with fresh versions — a snapshot-live value beats a current tombstone, a snapshot tombstone beats a current live value, tombstone no-ops burn no version, and keys absent from the snapshot keep their state (D-09); a fresh engine restores the whole snapshot from the carried-over directory alone
- The security boundary is proven, not assumed: eight invalid name shapes rejected by both operations with `ErrInvalidSnapshotName` before any filesystem touch (the sentinel wins even where `ErrNoDump`/`fs.ErrNotExist` would mask the ordering), the 64/65 length boundary pinned, and directory scans prove nothing escaped `.snapshots` (T-3-04)
- Phase 3's executable scope is closed: `make lint` and `make test` green repo-wide under `-race`, go.mod/go.sum byte-identical, internal/port unchanged at doc.go + dump.go — PERS-01..PERS-05 covered by automated tests

## Task Commits

Each task was committed atomically (Task 1 as a TDD RED/GREEN pair; Task 2 as a single regression-lock commit over already-implemented behavior):

1. **Task 1 (tracer, RED):** `585d619` (test) — failing snapshot/restore tracer + compile-enabling stubs; failed on the planned assertion (snapshot file absent after `Snapshot("pre")`)
2. **Task 1 (tracer, GREEN):** `175ffb1` (feat) — Engine.Snapshot/Restore, flushCh dirty-check servicing, exit-sweep flush resolution, doc.go contracts
3. **Task 2:** `c34d8c5` (test) — name-validation table, overwrite/error-class pins, synctest flush-forces-dump, restore-missing; full-phase gate

**Plan metadata:** recorded by the docs commit that carries this SUMMARY.

## Files Created/Modified

- `internal/app/persist/engine.go` - Engine.Snapshot (force-then-copy), Engine.Restore (decode, sorted-key replay via store.Set/Delete, one confirming flush), flushDump's clean-engine fast path, markDirty, replayRecord, exit-sweep flush resolution
- `internal/app/persist/doc.go` - the two operations' contracts: D-10 force-then-copy, D-09 replay-only-snapshot-keys with fresh-version dominance, skip rules, fail-loudly
- `internal/app/persist/engine_test.go` - the tracer TestEngine_snapshotCreateLoadRestore, TestEngine_snapshotFlushForcesDump (synctest), TestEngine_restoreMissingSnapshot; fake-port snapshot accessor
- `internal/persistfile/snapshot_test.go` - TestFileStore_nameValidation (security table + directory scans + ordering probes), TestFileStore_snapshotOverwriteAtomic, TestFileStore_snapshotErrorClasses

## Decisions Made

- Flush semantics per D-10's literal wording: Run services a flush request with an immediate dump only when dirty; a clean engine has nothing unconfirmed on disk and replies nil without a redundant save — pinned exactly (attempt count == 2) under synctest
- Restore confirmation is semantics-level (research Open Question 4's resolved reading): each key replays through the store's ordinary write path, and the whole batch is confirmed by ONE final flush — flagged assumption 1, carried to phase verification alongside 03-01's PERS-04 assumption
- Snapshot overwrite is atomic-in-place with no .prev (research Open Question 2's resolved reading) — flagged assumption 2, carried to phase verification
- Plan-internal naming inconsistency resolved in favor of the task blocks: the frontmatter artifact field says `contains: TestEngine_restoreReplaysSnapshot`, but the tasks and the plan's own artifacts list name `TestEngine_snapshotCreateLoadRestore` (the name the `<verify>` command greps for); the test carries the tasks' name

## TDD Gate Compliance

Plan type is `execute` with `tdd="true"` tasks (repo-level `tdd_mode` is off, so the machine classifier gate does not apply; the 03-01 RED/GREEN precedent does):

- Task 1: genuine RED — `585d619` failed on the planned behavior assertion (`read the snapshot: ... no such file or directory`), the package compiled and only the target test failed; GREEN — `175ffb1` makes it pass with lint/build/vet/deps all clean
- Task 2: single `test` commit — the suite pins behavior that already existed when the tests were written (the adapter shipped in 03-01, the flush semantics in Task 1's GREEN), so every test was green on arrival by design; this mirrors 03-01's Task 3 (codec properties over shipped code). The tests are the deliverable, exactly as the plan's artifact block specifies
- Tracer feedback gate: auto mode active, no `blocking-human` gate — the tracer's full verify chain re-ran end-to-end after GREEN and passed, so execution expanded to Task 2

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fresh-engine record-count assertion corrected from 5 to 3**
- **Found during:** Task 1 (first GREEN run)
- **Issue:** The tracer asserted a fresh engine restoring the snapshot holds all 5 snapshot keys, but the two snapshot tombstones replay as `store.Delete` ErrNotFound no-ops on a store that never held those keys — Delete writes nothing there (store.go:179-183), so only the 3 live keys exist. The test expectation contradicted the plan's own Task 1 behavior spec ("a key the snapshot holds deleted that no longer exists makes Restore's Delete a skip ... no version burned")
- **Fix:** Assert 3 records and document the no-op semantics in the assertion message; the Get(ErrNotFound) assertions for the tombstoned keys already covered their absence
- **Files modified:** internal/app/persist/engine_test.go
- **Verification:** TestEngine_snapshotCreateLoadRestore green under -race
- **Committed in:** 175ffb1

**2. [Rule 1 - Bug] Two strict-lint findings in the new tracer test**
- **Found during:** Task 1 (first GREEN run)
- **Issue:** govet shadow flagged `rec` redeclared inside the fresh-engine loop; noinlineerr flagged an inline `if _, err := eng.Get(...)` handler
- **Fix:** Renamed the loop variables and split the inline handler into a plain assignment + if
- **Files modified:** internal/app/persist/engine_test.go
- **Verification:** `go tool golangci-lint run ./internal/app/persist/...` exits 0
- **Committed in:** 175ffb1

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs, both in the new test code — zero production-code fixes needed; the 03-01 implementation held)
**Impact on plan:** None on scope; both fixes corrected the test to assert the plan's specified semantics.

## Issues Encountered

None beyond the deviations above — the adapter seam from 03-01 (CreateSnapshot/ReadSnapshot, validateName, ErrNoDump/ErrInvalidSnapshotName) needed no changes under the new engine orchestration.

## Authentication Gates

None — no external services.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — every snapshot path confirms through a real or faked DumpStore; no placeholder data flows anywhere.

## Next Phase Readiness

- PERS-01..PERS-05 delivered and machine-verified across plans 03-01 + 03-02; Phase 3's executable scope is complete
- Ready for phase verification (`/gsd:verify-work 3`), then Phase 4 planning (replication — flagged in STATE.md as needing `--research-phase`)
- Flagged for phase verification: restore's batched confirmation (assumption 1), snapshot overwrite-without-.prev (assumption 2), and 03-01's carried file-mode assumption

## Self-Check: PASSED

- All 4 created/modified files exist on disk (verified with [ -f ])
- All 3 commits (585d619, 175ffb1, c34d8c5) are ancestors of HEAD (verified with git merge-base --is-ancestor)
- Full-phase gate green at the final commit: `make lint` (config verify + run ./...) and `make test` (`go test -race -cover ./...`) exit 0; go.mod/go.sum byte-identical; internal/port holds exactly doc.go + dump.go; no file deletions in any task commit

---
*Phase: 03-persistence-dump-and-write-confirmation*
*Completed: 2026-11-02*
