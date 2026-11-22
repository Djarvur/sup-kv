---
phase: 05-persistence
plan: 05
subsystem: persistence
tags: [load-outcome, evidence-preservation, corrupt-rename, rotation-survival, orphan-sweep, temp-marker, parent-sync, dir-fsync-tolerance, failure-class, errno-classification]
requires:
  - phase: 05-persistence
    provides: the port LoadResult contract and streamed SaveStream from 05-04
  - phase: 03-transport
    provides: the fsops five-step atomic save seam and the targetExists rotation guard
affects: [persistence, load-outcome, dump-path, evidence-preservation, temp-hygiene, p09, p12]
actuals:
  tokens: 12676
  tasks: 3
  commits: 6
plan_head_before: b5cd6eae3b346e336a7ec643e500ad2bef089868
plan_head_after: f5e954ab131bd3346ca207e1a6f9c5d207142635
tech-stack:
  added: []
  patterns:
    - "Preservation at load: every rejected regular candidate is renamed to .corrupt-<UTC>[-N] before any save, so the rotation path never holds a corrupt generation"
    - "Rotation survival: after a Backup start the corrupt main is gone, the first rotate is a no-op, and .prev stays byte-identical"
    - "Marker-based sweep: temps carry .tmp~, a character outside the snapshot charset, so selection never keys on a data-file prefix"
    - "ensureDir: a created directory syncs its parent before the temp write; an existing directory adds no seam step"
    - "Classed errors: errno knowledge stays in the adapter and crosses the port as port.FailureClass while errors.Is stays transparent"
key-files:
  created:
    - internal/persistfile/load_outcome_test.go
    - internal/persistfile/sweep_test.go
    - internal/persistfile/saveerror.go
    - internal/persistfile/saveerror_test.go
  modified:
    - internal/persistfile/dumpfile.go
    - internal/persistfile/dumpfile_test.go
    - internal/persistfile/fsops.go
    - internal/persistfile/fsops_test.go
    - internal/persistfile/snapshot.go
    - internal/port/dump.go
key-decisions:
  - "Preservation is load-time and direct: Lstat decides regularity, Rename replaces the entry itself, and a non-regular path is a failure with a cause and no rename (T-5-13)"
  - "The same-second collision probe uses base, base-1, base-2 … with Lstat, so two preserves never overwrite each other (Pitfall 15)"
  - "An absent main is recorded as the one main failure (fs.ErrNotExist, no preserved path) whenever .prev is accepted, so the engine's single Backup WARN always names why the main was not used"
  - "The temp marker is .tmp~ — '~' is outside [A-Za-z0-9._-] — superseding the research dump-* prefix, because --dump-path and snapshot names are unrestricted and a dump- prefix matches live files (Pitfall 16)"
  - "The sweep selects by marker containment on regular files within the one-hour bound, scans both the dump and snapshots directories, and excludes the configured main, its .prev and every .corrupt-* exactly in the dump directory"
  - "ensureDir parent-syncs only when it created the directory; existing directories keep the exact five-step TEST-03 order, and CreateSnapshot uses the same helper"
  - "Only the directory fsync tolerates ENOTSUP/EINVAL; the file-data fsync keeps every error, and classifySaveError wraps every SaveStream error once at the adapter boundary"
patterns-established:
  - "Evidence-preserving load: failure record + collision-free rename + best-effort parent sync, with the accepted generation untouched"
  - "Lint-clean sweep factoring: protectedDataName and sweepEntry keep the flag-parameter and cognitive-complexity gates green without changing the selection rules"
  - "RED stubs for compile-coupled contract tasks: the new port vocabulary and declaration-only helpers land with the failing tests, then GREEN replaces the stubs"
requirements-completed: [PERSIST-02, PERSIST-09]
coverage:
  - id: D1
    description: "Load reports Prompt/Main/Backup/Unusable with per-candidate causes, preserves every rejected regular candidate under a collision-free .corrupt-<UTC>[-N] name, never renames a non-regular file, and the first save after a backup start leaves .prev byte-identical"
    requirement: "PERSIST-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFileStore_load'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFileStore_preserveCollision$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "The startup sweep removes only regular .tmp~-marked files older than one hour in the dump and snapshots directories, protects the configured main/.prev/.corrupt-* for any name (including a marker-bearing main) and every legal snapshot, and reports Swept/SweepErr without aborting the load"
    requirement: "PERSIST-09"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestSweep_'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestSweep_protectsDataNames$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "A newly created dump or snapshots directory syncs its parent before the temp write; an existing directory adds no step; only the directory fsync tolerates ENOTSUP/EINVAL; save errors carry port.FailureClass and stay errors.Is-transparent; the darwin build/vet stay green"
    requirement: "PERSIST-09"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_saveOperationOrder$|^TestEnsureDir_'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestSyncDir|^TestClassifySaveError'"
        status: pass
      - kind: build
        ref: "GOOS=darwin go build ./... && GOOS=darwin go vet ./..."
        status: pass
    human_judgment: false
duration: 14min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 05: Load Outcomes, Evidence Preservation and Temp Hygiene Summary

**Load now reports all four outcomes with causes, renames every rejected generation to `.corrupt-<UTC>[-N]` before any save so the first rotation leaves `.prev` byte-identical, and sweeps only hour-old `.tmp~` temps; new directories parent-sync before the temp write and save errors cross the port as `FailureClass`.**

## Performance

- **Duration:** 14min
- **Started:** 2026-11-22T14:22:31Z
- **Completed:** 2026-11-22T14:36:20Z
- **Tasks:** 3
- **Files modified:** 10 (4 created, 6 modified)

## Accomplishments

- `Load(now, parse)` now yields `Prompt`/`Main`/`Backup`/`Unusable` with one `GenerationFailure` per candidate that existed and failed: a corrupt main is preserved and named, an absent main is recorded with `fs.ErrNotExist` when `.prev` is accepted, and an absent candidate adds no failure in the unusable case.
- Preservation is load-time evidence handling: `Lstat` decides regularity (a directory/symlink/device is a failure with no rename), `Rename` moves the entry to `<path>.corrupt-<UTC>`, and same-second preserves probe `-1`, `-2` so both survive.
- Rotation survival is proven: after a `Backup` start the corrupt main has left the main path, the first save's rotate is a no-op, and `.prev` bytes are identical before and after.
- Temp files are named `.tmp~<random>`; the marker's `~` is outside the snapshot charset, so the startup sweep selects by marker only and never by a data-file prefix.
- `sweepOrphans` removes only regular marker files older than one hour, in both the dump and `.snapshots` directories; the configured main, its `.prev` and every `.corrupt-*` are excluded exactly (including a main whose own name carries the marker), every legal snapshot survives by construction, and `Swept`/`SweepErr` report the result without aborting the load.
- `ensureDir` syncs a created directory's parent before the write; an existing directory adds no step, so the five-step TEST-03 order is unchanged. `CreateSnapshot` uses the same helper for its directory.
- `syncDir` tolerates `ENOTSUP`/`EINVAL` only; the file-data fsync path is untouched. `port.FailureClass`/`ClassedError`/`FailureClassOf` carry the vocabulary, and `classifySaveError` wraps every `SaveStream` error (`ENOSPC` → `disk_full`, `EACCES`/`fs.ErrPermission` → `permission`, else `other`) while keeping `errors.Is`.
- Full gates green: `make lint`, `make test` (race, all packages), `GOOS=darwin go build ./... && GOOS=darwin go vet ./...`.

## Task Commits

1. **Task 1 RED: preservation and collision tests** - `0e5b22b` (test)
2. **Task 1 GREEN: preserve the corrupt main, keep `.prev` across the first save** - `9d940d4` (feat)
3. **Task 2 RED: outcome table and sweep tests** - `c43427b` (test)
4. **Task 2 GREEN: marker-named temps and the age-bounded orphan sweep** - `7d60d3d` (feat)
5. **Task 3 RED: parent-sync, tolerance and errno-class tests** - `84d3600` (test)
6. **Task 3 GREEN: ensureDir, directory-fsync tolerance, save-error classification** - `f5e954a` (feat)

**Plan metadata:** the `docs(05-05)` completion commit (this commit).

## Files Created/Modified

- `internal/persistfile/dumpfile.go` — `corruptSuffix`/`stampLayout`, `preserve`, `failedCandidate`, the sweep (`orphanTempMarker`, `defaultOrphanTempAge`, `sweepOrphans`, `protectedDataName`, `sweepEntry`), `ensureDir`, and `classifySaveError` at the `SaveStream` boundary.
- `internal/persistfile/saveerror.go` — `classedSaveError` and the errno-to-`port.FailureClass` classification.
- `internal/persistfile/fsops.go` — the `.tmp~` marker-aware temp hygiene comment, `tolerateDirSyncErr`, and the tolerant directory step.
- `internal/persistfile/snapshot.go` — the snapshot directory is created through `ensureDir`.
- `internal/port/dump.go` — `FailureClass`, `FailureDiskFull`, `FailurePermission`, `String`, `ClassedError`, `FailureClassOf`.
- `internal/persistfile/load_outcome_test.go` — preservation, collision, four-outcome, non-regular and unreadable candidate tests.
- `internal/persistfile/sweep_test.go` — age bound in both directories, data-name protection for dump-prefixed and marker-bearing mains, and the unreadable-directory report.
- `internal/persistfile/saveerror_test.go` — the errno classification table and class label vocabulary.
- `internal/persistfile/fsops_test.go` — `TestEnsureDir_*`, `TestSyncDirTolerant`, and the parent-sync order proof.
- `internal/persistfile/dumpfile_test.go` — the absent-main backup row now expects its one `fs.ErrNotExist` failure; the temp-litter comment names the marker scheme.

## Decisions Made

- **Preservation before saves, not after.** The corrupt generation leaves `main` at load; otherwise marking the engine dirty would accelerate the loss on the first rotation.
- **Exact exclusions plus marker selection.** A data prefix cannot be the sweep key because `--dump-path` and snapshot names are unrestricted; `~` in the marker makes every legal snapshot marker-free.
- **One main failure for every Backup.** The absent-main case is a recorded failure (`fs.ErrNotExist`), not silence, so the engine's WARN needs no synthesis.
- **Directory tolerance is directory-only.** `tolerateDirSyncErr` is called from `syncDir` alone; the file-data `fsync` keeps every error.
- **Classification at the adapter edge.** `classifySaveError` wraps `SaveStream`'s error once; `port.FailureClassOf` is the core's only view, and `errors.Is` still reaches the errno.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test expectation ordering] The absent-main backup row moved with the behavior change**
- **Found during:** Task 1
- **Issue:** Task 1's action adds the absent-main `GenerationFailure` when `.prev` is accepted, but the plan listed `dumpfile_test.go` under Task 2; the existing "main removed serves prev" row expected zero failures and would have failed in the Task 1 commit.
- **Fix:** Updated that row's `wantFails` to 1 and its doc comment in the Task 1 GREEN commit, so each task commit leaves the package suite green.
- **Files modified:** `internal/persistfile/dumpfile_test.go`
- **Verification:** `go test -race -count=1 ./internal/persistfile/` green at `9d940d4`.
- **Commit:** `9d940d4`

**2. [Rule 1 - TDD sequencing] Task 3 RED carries the port vocabulary and declaration stubs**
- **Found during:** Task 3
- **Issue:** The new tests reference `port.FailureClass`, `tolerateDirSyncErr` and `classifySaveError`, which do not exist on the pre-task tree; a build failure is not valid RED evidence (the same class of issue 05-04 recorded for compile-coupled contract work).
- **Fix:** The RED commit added the real stdlib-only port vocabulary and declaration stubs — `tolerateDirSyncErr` and `classifySaveError` returned their input — so the target tests failed on assertions (ENOTSUP/EINVAL not skipped, ENOSPC/EACCES classified `other`). The GREEN commit replaced the stubs.
- **Files modified:** `internal/port/dump.go`, `internal/persistfile/fsops.go`, `internal/persistfile/saveerror.go`
- **Verification:** RED run showed the three expected assertion failures; GREEN run is green.
- **Commit:** `84d3600` (RED), `f5e954a` (GREEN)

**3. [Rule 3 - Lint/toolchain] `FailureClassOf` uses `errors.AsType`**
- **Found during:** Task 3
- **Issue:** The vendored strict config's `modernize` linter requires `errors.AsType[ClassedError](err)` over the `errors.As(err, &classed)` form; the module is pinned at Go 1.27, which provides it.
- **Fix:** Used `errors.AsType`; behavior is unchanged (nil/non-matching error returns `FailureOther`).
- **Files modified:** `internal/port/dump.go`
- **Verification:** `go tool golangci-lint run ./internal/persistfile/ ./internal/port/` clean.
- **Commit:** `84d3600`

## Issues Encountered

- The strict lint gate required two small test-shape fixes (`noinlineerr` and `govet` shadow) and a sweep factoring split (`protectedDataName` + `sweepEntry`) to avoid `revive`'s flag-parameter warning; no selection semantics changed. All gates green on the final tree.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The adapter now carries the full D-02/D-08 contract: outcomes, per-candidate causes, preservation, rotation survival, sweep, parent sync and classified save errors. Plan 05-06 can map `LoadResult` into the engine's `LoadStats` and own the WARN/ERROR logging without another adapter change.
- `port.FailureClassOf` is ready for the engine's failure counters and metrics wiring.
- Ready for 05-06.

## Self-Check: PASSED
