---
phase: 06-api-and-semantics
plan: 04
subsystem: api-and-semantics
tags: [apis-03, error-mapping, totality, path-free-status, constant-fallback, slog-detail, failedprecondition, resourc-exhausted, permission-denied, snapshot-errors, tdd-red-evidence, status-complete]
requires:
  - phase: 05-persistence
    provides: "the port.DumpStore snapshot seams (CreateSnapshot/ReadSnapshot), the classifySaveError reduction and the port.FailureClass vocabulary the mapper consumes"
  - phase: 06-api-and-semantics
    plan: 01
    provides: "the public api/supkvapi/v1 wire package every touched test drives"
provides:
  - "one ordered, total mapError table: every storage/persist/persistfile sentinel and both port failure classes pinned to a code plus constant or sentinel text"
  - "persistfile.ErrNoDump -> FailedPrecondition and fs.ErrNotExist -> NotFound (\"snapshot not found\"), so a missing snapshot is never Internal and never carries the configured path"
  - "a constant Internal fallback (exactly \"internal error\") that logs the detail with slog instead of formatting it into the response"
  - "classifySaveError around CreateSnapshot's directory-setup, main-dump-read and atomic-save failures, so ENOSPC -> FailureDiskFull -> ResourceExhausted and EACCES -> FailurePermission -> PermissionDenied"
  - "three proof layers: the total-table test, the non-parallel log-capture test, and the six-RPC path-shape matrix with its predicate self-check"
affects: [06-05, 06-09, 06-10, 06-13-phase-gate, phase-9-api-docs, apis-03-acceptance]
actuals:
  tokens: 8453
  tasks: 3
  commits: 4
  plan_head_before: dabd55ef13ed1d75c509e08c79e21a6920d649c1
  plan_head_after: e93bc0427d9ef77a49e47df2494452f9e395b72c
tech-stack:
  added: []
  patterns:
    - "Totality by constant: one ordered switch over sentinels and failure classes with a single constant fallback that logs the detail through slog, so no error text is ever formatted into a client-visible message."
    - "Class-at-the-adapter: the persistfile adapter reduces errno to port.FailureClass and the API maps only the class, keeping path and errno text behind the port (T-06-13)."
    - "Path-shape predicate: a table-driven RPC matrix rejects the injected path, absolute-path shapes and backslash separators in every status message, and a self-check on the predicate keeps the filter from passing vacuously."
key-files:
  created:
    - internal/grpcapi/errors_test.go
    - internal/grpcapi/errors_internal_test.go
    - internal/persistfile/snapshot_class_internal_test.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-04-task1.json
  modified:
    - internal/grpcapi/server.go
    - internal/persistfile/snapshot.go
key-decisions:
  - "The two port-class rows sit after persist.ErrDumpFailed: the engine wraps a failed save as ErrDumpFailed, so a write answers Unavailable with the pinned sentinel, while CreateSnapshot surfaces the adapter's class directly and reaches ResourceExhausted/PermissionDenied (T-06-13). Placing the class rows first would change the dump-failure message."
  - "persistfile.ErrNoDump precedes the fs rows so its sentinel text survives; \"no dump configured\" is realized as ErrNoDump because --dump-path is mandatory, which is the research reading of D-03."
  - "The context rows pass the matched sentinel to status.FromContextError, never the arriving error: a wrapped cancellation would make FromContextError answer Unknown, which the plan explicitly forbids mapping to Internal."
  - "The fallback logs slog.Error(\"Unmapped RPC error\", \"detail\", err.Error()) and returns exactly the constant; the log test is the only non-parallel test in the plan because it swaps the process-global slog handler, with a named //nolint:paralleltest reason."
  - "CreateSnapshot classifies all three failure steps (ensureDir, the non-ErrNoDump readMainDump error, saveAtomically) and passes ErrNoDump through untouched; the adapter test pins FailureClassOf(ErrNoDump) == FailureOther and the absence of port.ClassedError."
  - "The RPC matrix uses one DumpStore fake per errno variant whose save, create and read seams all return the same *fs.PathError carrying the test's scratch path. SnapshotCreate's forced flush fails on the save seam before CreateSnapshot runs, so that row proves the ErrDumpFailed constant; the CreateSnapshot class mapping is proven by the adapter test and the total table instead."
  - "Task 2's tdd=true produced an investigated unexpected GREEN: the plan assigns the mapper rows and the adapter classification to Task 1's files, so Task 1's GREEN commit implemented exactly the behaviour Task 2 pins. The test commit is the deliverable and no RED was fabricated and no unnecessary production change invented (04-07 precedent)."
  - "The stale plan-head ledger .git/gsd-plan-head-before-06-04 (dated 2026-11-16, a v1.0 commit) was reset to this plan's true pre-plan HEAD dabd55e before measuring, the 06-01/06-02/06-03 precedent, so actuals.commits: 4 is the measured count."
patterns-established:
  - "A status-message constant block beside the mapper: exported-to-clients text lives in named constants that tests assert by literal, so a sentinel rename fails a test instead of silently moving the wire text."
  - "The last-resort error path logs and answers a constant; the log carries the only copy of the detail, which makes the fallback both total and path-free by construction."
requirements-completed: [APIS-03]
coverage:
  - id: D1
    description: "One ordered mapError table covers every storage/persist/persistfile sentinel and both port failure classes; ErrNoDump maps to FailedPrecondition, fs.ErrNotExist to NotFound with \"snapshot not found\", fs.ErrPermission and FailurePermission to PermissionDenied, FailureDiskFull to ResourceExhausted with \"disk full\", context cancellation/deadline to Canceled/DeadlineExceeded, a pre-built status passes through, and every other error logs its detail and answers exactly \"internal error\"."
    requirement: "APIS-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_' -> PASS (16 table rows + the 3 external snapshot rows; the 16 rows are the 14 mapper cases plus the pass-through and the fallback)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_unmappedIsConstantAndLogged$' -> PASS (message is exactly \"internal error\", the path-bearing detail appears only in the captured slog output)"
        status: pass
      - kind: process
        ref: "grep -q 'internalErrorMessage = \"internal error\"' internal/grpcapi/server.go -> exit 0; grep -q 'persistfile.ErrNoDump' internal/grpcapi/server.go -> exit 0 with the case in the FailedPrecondition position"
        status: pass
    human_judgment: false
  - id: D2
    description: "CreateSnapshot classifies its ensureDir, main-dump read and atomic save failures with classifySaveError while returning ErrNoDump unchanged, so ENOSPC/EACCES reach the mapper as port classes rather than path-bearing error text."
    requirement: "APIS-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestCreateSnapshot_failureClasses$' -> PASS (ENOSPC is FailureDiskFull, fs.ErrPermission is FailurePermission, a dump-less store keeps ErrNoDump with FailureOther and no port.ClassedError)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_noDumpIsFailedPrecondition$' -> PASS (code FailedPrecondition, message exactly \"no dump on disk to snapshot\")"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFileStore_snapshotErrorClasses$' -> PASS (the pre-existing error-class split still holds after classification)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The RPC path-shape matrix drives Get, Set, Delete, List, SnapshotCreate and SnapshotLoad against a DumpStore whose save/create/read seams return a path-bearing *fs.PathError in the fs.ErrNotExist and fs.ErrPermission variants, rejecting the injected path, absolute-path shapes and backslash separators in every returned message, pinning each row's code and self-checking the predicate."
    requirement: "APIS-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestRPC_statusMessagesPathFree$' -> PASS (both errno variants, all six rows; every error carries its pinned non-Unknown code)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -> PASS (whole package, 2.1 s under -race; no pre-existing test regressed)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The real persistfile adapter's missing-snapshot PathError — which carries the configured dump directory — becomes NotFound with the constant message through the production mapper, proving the concrete A-03 leak is closed end to end."
    requirement: "APIS-03"
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_missingSnapshotRealAdapter$' -> PASS (real persistfile FileStore over t.TempDir; code NotFound, message exactly \"snapshot not found\", no slash or backslash)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The touched tree is gate-clean: the API and adapter packages build and vet with every test package compiled, and the repo-wide strict lint passes."
    requirement: "APIS-03"
    verification:
      - kind: process
        ref: "go build ./... && go vet ./... -> exit 0 (vet type-checks every test package)"
        status: pass
      - kind: process
        ref: "make lint -> exit 0 (config verify + `go tool golangci-lint run ./...` over the whole repo)"
        status: pass
    human_judgment: false
duration: 7min
completed: 2026-11-22
status: complete
---

# Phase 06 Plan 04: Total, Path-Free Error Mapping Summary

**Every engine and adapter error now crosses the wire through one ordered table whose answer is a constant or a package sentinel — a missing snapshot is `NotFound` with "snapshot not found", a dump-less create is `FailedPrecondition`, and anything unforeseen is exactly "internal error" with the path-bearing detail logged, never returned.**

## Performance

- **Duration:** 7min (started 2026-11-22T23:15:52Z, completed 2026-11-22T23:22:49Z)
- **Tasks:** 3 (1 tracer with TDD, 1 proof-test task, 1 matrix test)
- **Commits:** 4 (`ccf6760` test RED, `3d3f24c` feat GREEN, `f298532` test proof, `e93bc04` test matrix) — measured `git rev-list --count dabd55e..HEAD`
- **Files:** 6 changed (3 created test files, 1 created RED record, 2 production files modified); 684 insertions, 12 deletions

## Accomplishments

- **The A-03 path leak is closed at its source.** `ReadSnapshot` still returns the adapter's raw `*fs.PathError` — the path-bearing text stays where it is useful — but the mapper now reduces `fs.ErrNotExist` to `NotFound` with the constant `snapshot not found`. The real-adapter test proves the configured dump directory never appears in a status message.
- **`ErrNoDump` finally has a case.** A `SnapshotCreate` with no main dump on disk answers `FailedPrecondition` with the adapter's sentinel text ("no dump on disk to snapshot"), the pinned reading of "no dump configured" for a mandatory `--dump-path` build. The case precedes the `fs` rows so its own text survives.
- **Totality is structural, not enumerated.** The seven pre-existing sentinel cases are unchanged; the new rows add `ErrNoDump`, `fs.ErrNotExist`, `fs.ErrPermission`, `FailureDiskFull`, `FailurePermission`, `context.Canceled` and `context.DeadlineExceeded`. Everything else reaches one fallback that logs `slog.Error("Unmapped RPC error", "detail", err.Error())` and answers exactly `internal error` — the `%v` formatting that leaked paths is gone (T-06-11, T-06-12).
- **Snapshot create failures are classified adapter-side.** `CreateSnapshot` now wraps its `ensureDir`, non-`ErrNoDump` read and `saveAtomically` failures with `classifySaveError`, so ENOSPC crosses the port as `FailureDiskFull` (→ `ResourceExhausted` "disk full") and EACCES as `FailurePermission` (→ `PermissionDenied`), while `ErrNoDump` stays an unclassified sentinel the mapper can distinguish (T-06-13).
- **Three independent proof layers.** The total-table test names every switch case plus the status pass-through and the fallback; the non-parallel log-capture test proves the detail is logged while the response is the constant; the six-RPC matrix drives Get, Set, Delete, List, SnapshotCreate and SnapshotLoad against path-bearing save/create/read errors in both errno variants and rejects every path shape, with a self-check that the predicate really rejects a path-bearing text.
- **Gate-clean on the committed tree:** repo-wide strict `golangci-lint` (config verify included) exits 0, `go build ./...` and `go vet ./...` (which compiles every test package) exit 0, and both touched packages pass under `-race` in full (grpcapi 2.1 s, persistfile 5.0 s).

## Task Commits

1. **Task 1 (RED): failing snapshot error-mapping tests** — `ccf6760` (`test`). `errors_test.go` plus the RED record: a `DumpStore` returning `&fs.PathError{Path: <temp>/secret, Err: fs.ErrNotExist}` from `ReadSnapshot` must answer `NotFound`/"snapshot not found" with no temp path, a `CreateSnapshot` returning `ErrNoDump` must answer `FailedPrecondition` with the sentinel text, and the real `newFileServer` adapter must answer `NotFound` path-free. Both RPC tests failed against the `%v` fallback (`want code NotFound, got Internal` with the temp path in the message; `want code FailedPrecondition, got Internal`).
2. **Task 1 (GREEN): the total path-free table and the classified snapshot create** — `3d3f24c` (`feat`). The ordered `mapError` rewrite with the named message constants, the slog fallback, and `CreateSnapshot`'s `classifySaveError` wrappers.
3. **Task 2: totality and failure-class proofs** — `f298532` (`test`). `errors_internal_test.go` (16-row total table + non-parallel slog capture) and `snapshot_class_internal_test.go` (ENOSPC/EACCES/no-dump rows through the `failingFSOps` seam).
4. **Task 3: the RPC path-shape matrix** — `e93bc04` (`test`). `TestRPC_statusMessagesPathFree` over both errno variants, the injectable `SaveStream` error on the fake, the path-shape predicate helper and its self-check.

**Plan metadata:** the `docs(06-04)` completion commit (this commit).

## Files Created/Modified

- `internal/grpcapi/server.go` — `mapError` rewritten as the ordered total table with the message-constant block (`internalErrorMessage`, `snapshotNotFoundMessage`, `permissionDeniedMessage`, `diskFullMessage`); the class rows after `ErrDumpFailed`; the `fs` rows after `ErrNoDump`; context rows through `status.FromContextError`; the fallback logs and returns the constant. Imports `io/fs`, `log/slog` and `internal/port`.
- `internal/persistfile/snapshot.go` — `CreateSnapshot` classifies its three failure steps with `classifySaveError` and returns `ErrNoDump` unchanged; the doc states the D-03/T-06-13 rationale.
- `internal/grpcapi/errors_test.go` (new) — the `snapshotErrorDumpStore` fake (injectable save/create/read errors), the three D-03 snapshot tests, `pathShapePattern`, `assertStatusMessagePathFree` and `TestRPC_statusMessagesPathFree`.
- `internal/grpcapi/errors_internal_test.go` (new) — the 16-row total table, the test-local `classedError`/`unmappedError`, and the non-parallel log-capture test.
- `internal/persistfile/snapshot_class_internal_test.go` (new) — the three `CreateSnapshot` classification rows through the `failingFSOps` injection point.
- `.planning/phases/06-api-and-semantics/red-evidence/06-04-task1.json` — the immutable RED record (command, exit 1, full console output, target test, expected/actual, semantic assessment, `RED_EVIDENCE_OK`).

## Decisions Made

- **Class rows after `ErrDumpFailed`, not before.** The engine wraps a failed save as `ErrDumpFailed` (the write answers `Unavailable` with the pinned sentinel), while `CreateSnapshot` surfaces the adapter's class directly. Any other order would change the dump-failure message or hide the class.
- **`ErrNoDump` before the `fs` rows.** Its sentinel text is the pinned "no dump configured" answer, and the adapter deliberately returns it instead of the underlying `fs.ErrNotExist`.
- **The context rows pass the matched sentinel, not the arriving error.** `status.FromContextError(err)` on a *wrapped* cancellation answers `Unknown`; passing `context.Canceled`/`DeadlineExceeded` keeps the mapping exact, which the plan's prohibition requires.
- **The fallback logs with a static message and a snake_case attribute.** `slog.Error("Unmapped RPC error", "detail", ...)` is the operator's only view of an unforeseen failure; the response carries exactly `internal error`.
- **`CreateSnapshot` classifies all three steps.** `ensureDir` was the trap: an ENOSPC during snapshot-directory creation would otherwise reach the mapper as an unclassified error and lose the class counter.
- **One DumpStore fake per errno variant in the matrix.** With the save seam failing, `SnapshotCreate`'s forced flush fails before `CreateSnapshot` runs, so that row proves the `ErrDumpFailed` constant; the class mapping is proven by the adapter test and the total table (the plan's own split).
- **Task 2's test-only TDD task was an investigated unexpected GREEN.** The plan's file assignment puts the mapper rows and the adapter classification in Task 1, so Task 1's GREEN already satisfied Task 2's assertions. The tests are committed as the deliverable; no RED was fabricated (04-07 precedent).
- **The stale ledger was reset before measuring.** `.git/gsd-plan-head-before-06-04` pointed at a 2026-11-16 v1.0 commit; it was reset to the true pre-plan HEAD `dabd55e`, matching 06-01/06-02/06-03.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Two doc comments tripped `godoclint`'s stdlib link rule**
- **Found during:** the pre-commit lint run after Task 1's GREEN edits
- **Issue:** the new message-constant comments mentioned `*fs.PathError` and `fs.ErrPermission` as bare text; the vendored strict config requires `[*fs.PathError]` / `[fs.ErrPermission]` link syntax.
- **Fix:** the two references were bracketed. The fix landed inside the GREEN commit, so no commit in the plan's history fails `make lint`.
- **Files modified:** `internal/grpcapi/server.go`
- **Verification:** `go tool golangci-lint run ./internal/grpcapi/... ./internal/persistfile/...` exit 0; later `make lint` exit 0.

**2. [Rule 2 - Missing critical functionality] The matrix's path-shape predicate could pass vacuously**
- **Found during:** Task 3's design
- **Issue:** a regex that never matched would make `TestRPC_statusMessagesPathFree` pass without proving anything.
- **Fix:** the test asserts the predicate rejects a synthetic `internal error: open <path>: ...` text and accepts the mapper's constant before driving the RPC rows — the filter's teeth are part of the test.
- **Files modified:** `internal/grpcapi/errors_test.go`
- **Verification:** the matrix passes with the self-check in place (`go test -race -count=1 ./internal/grpcapi/ -run '^TestRPC_statusMessagesPathFree$'`).

Otherwise none — the plan executed as written.

## TDD Gate Compliance

- **Task 1 RED:** `ccf6760` — `test(06-04): add failing snapshot error-mapping tests`. Evidence `.planning/phases/06-api-and-semantics/red-evidence/06-04-task1.json`, verdict `RED_EVIDENCE_OK` (target `TestMapError_missingSnapshotIsNotFound`, 2 tests, 2 failing, reason `target_test_failed`). The failures are assertion-level: the observed message contained the injected temp path, which is exactly the leak under repair.
- **Task 1 GREEN:** `3d3f24c` — `feat(06-04): map every error through a total path-free table`. Both target tests pass, the third real-adapter test passes, and both packages stay green.
- **Task 1 REFACTOR:** none needed — no cleanup commit was made and the GREEN tree is the committed tree.
- **Task 2 (`tdd="true"`, test-only files):** investigated **unexpected GREEN**. All three verify commands pass on the first run because the asserted behaviour landed in Task 1's GREEN commit by the plan's own file assignment (`server.go` and `snapshot.go` are Task 1 files; Task 2's files are the two test files). This is the 04-07 precedent: the test commit `f298532` is the deliverable, and no RED record, fake failure or unnecessary production change was invented. The investigation result is recorded here rather than suppressed.

## Issues Encountered

- **APIS-03's checkbox stays pending.** `gsd_run query requirements.ready-ids` reports APIS-03 as `blocked` (sibling plans also declare it, the #2388 shared-ID gate), so no requirement checkbox was flipped. That matches 06-03's APIS-06 handling.
- **No other issues.** The full package suites, build, vet and repo-wide lint passed on the committed tree; the known `TestScenario_dumpNotify` / ratio flakes were not touched by this plan's focused scope.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 06-11 (wave 3, depends on 06-02).** The stacked uncommitted-summary state is clean: 9 plans remain in the phase.
- **Ready for 06-05 (wave 4, depends on 06-01 + this plan).** `List` pagination can rely on the mapper's totality: any new engine error it surfaces either has a row or lands on the logged constant, and it cannot leak a path.
- **Phase 9's API error table** can be written from the mapper's doc comment and the total-table test, which now enumerate the same rows in the same order.
- **No deferred work from this plan.** Every must-have truth is pinned by a test, including the "no backslash separator" and "non-Unknown code" clauses.

## Self-Check: PASSED

- Created files exist: `internal/grpcapi/errors_test.go`, `internal/grpcapi/errors_internal_test.go`, `internal/persistfile/snapshot_class_internal_test.go`, `.planning/phases/06-api-and-semantics/red-evidence/06-04-task1.json` — all FOUND.
- Task commits are ancestors of HEAD: `ccf6760`, `3d3f24c`, `f298532`, `e93bc04` — all FOUND.
- Measured commit count from the plan-head ledger (`dabd55e..HEAD`): 4, matching `actuals.commits`.
