---
phase: 07-observability-and-configuration
plan: 04
subsystem: process-lifecycle
tags:
  - obsv-04
  - exit-codes
  - signal-before-wire
  - serve-error-dump
  - injectable-run
  - startup-error-record
requires:
  - phase: 07-observability-and-configuration
    provides: 07-02's config.Parse/config.Validate and the usage-only exit-2 path main still calls before run
  - phase: 05-persistence
    provides: persist.Engine.FlushTimeout's bounded final dump and the ErrDumpFailed/ErrUnconfirmed distinctions shutdown returns
provides:
  - the D-06 exit table (exitStartup 1, exitUsage 2, exitDumpFailed 3) with dump-failure precedence documented on the constants
  - run(cfg, rt) int as the injectable lifecycle seam and productionRuntime as a direct function-value assembly
  - signals armed before wire, so SIGTERM during dump load or bind takes the graceful path
  - a Serve error that still runs the full D-19 shutdown, bounded final dump included, before the code is chosen
  - startupError stage wrapping and one Startup-failed ERROR record that survives SUPKV_LOG_LEVEL=error
affects:
  - 07-10 (the OPERATIONS exit table and operations_test anchors move 1 → 3 for the failed final dump)
  - 07-11 (OBSV-04's phase checkbox closes with this plan's process evidence)
actuals:
  tokens: 6078
  tasks: 2
  commits: 2
  plan_head_before: f9180c576d336633f5036a7f7a15f0b2b8d28d53
  plan_head_after: be4a68fc3db77ea1d7231dff9bdb3ba32ad4960d
tech-stack:
  added: []
  patterns:
    - "Testable lifecycle: `run(cfg, rt) int` is a pure orchestration function over a `runtime` struct of function values; the production runtime is a direct assignment of armSignals/signal.Stop/wire/serve/shutdown (matching parameter directions), so no adapter closure can drift from the real path."
    - "Stage-carrying startup error: a typed `startupError{stage, err}` with Error/Unwrap lets `wire` keep its single error return while `run` emits one static-message ERROR record with `stage` and `detail`, surviving `SUPKV_LOG_LEVEL=error`."
    - "Root-proof failure construction: replacing the dump directory with a regular file makes os.CreateTemp fail with ENOTDIR for every uid, unlike chmod 0500 which a root-run CI bypasses."
key-files:
  created:
    - cmd/sup-kv/run_internal_test.go
  modified:
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
key-decisions:
  - "The stdlib runtime import is aliased goruntime: Go forbids a file-scope import and a package-block type sharing a name, and the plan's seam is named runtime."
  - "startupError wraps each wire failure (replication_bind, grpc_listen, metrics_listen); run renders stage and detail in one ERROR record, so the pre-fix log.Fatalf INFO-record disappearance is gone."
  - "Precedence is one switch in run: shutdownErr → 3, else serveErr → 1, else 0; a wire failure returns before serve/shutdown and still stops the signals. stopSignals runs on every path exactly once."
  - "The final-dump process test makes the engine dirty through a Set against the blocked path: a fresh node's Prompt-load dump succeeds inside the 50 ms debounce, so break-then-SIGTERM alone would find a clean engine (flushDump skips a clean engine) and exit 0; the failed Set leaves the pending dump the shutdown FlushTimeout retries."
  - "errors.As was replaced by errors.AsType[*startupError] to satisfy the modernize linter; the stage lookup behaviour is identical."
patterns-established:
  - "Pattern: an operator-facing lifecycle contract is tested twice — a fake-runtime unit test for the exact call order and code, and a real-binary process test for the observable exit status and log record."
  - "Pattern: a fatal startup path is a static-capitalized slog ERROR with stage/detail attributes, never a fatal logger call, so log-level filters cannot hide it."
requirements-completed: []
coverage:
  - id: D1
    description: "The 0/1/2/3 exit table with documented precedence, run's signal-before-wire seam, and one Startup-failed ERROR record with a stage at SUPKV_LOG_LEVEL=error"
    requirement: OBSV-04
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestRun_' -v"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_fatalBindLoggedAtError|TestMain_duplicateBind' -v"
        status: pass
    human_judgment: false
  - id: D2
    description: "A Serve error still runs the full shutdown (dump included), a failed final dump exits 3 on the signal and Serve-error paths, and the real binary over a blocked dump path exits 3 with the Final-dump-failed ERROR record"
    requirement: OBSV-04
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestRun_serveErrorStillShutsDown|TestRun_dumpFailureCode|TestRun_serveAndDumpFailure' -v"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_finalDumpFailureExitsThree|TestMain_lifecycle|TestMain_duplicateBind' -v"
        status: pass
      - kind: contract
        ref: "make lint && make test (whole repo green; 18 packages)"
        status: pass
    human_judgment: false
duration: 14min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 04: The Distinct Exit Contract and the Injectable run Orchestration Summary

**Made the process exit contract 0/1/2/3 with dump-failure precedence, armed signals before dump load and bind, kept the bounded final dump on every failure path, and turned the lifecycle into the testable `run` seam over a production runtime assembled by direct assignment.**

## Performance

- **Duration:** 14min
- **Started:** 2026-11-23T07:54:55Z
- **Completed:** 2026-11-23T08:08:34Z
- **Tasks:** 2
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- `cmd/sup-kv/main.go` now carries the D-06 table: `exitStartup = 1`, `exitUsage = 2`, `exitDumpFailed = 3`, with the precedence documented on the constants — a final-dump failure wins over a Serve error, and a startup failure never runs a final dump.
- `run(cfg config.Config, rt runtime) int` is the orchestration seam: `installSignals` runs before `wire`; a wire error logs one `Startup failed` ERROR record with `stage` and `detail`, stops the signals and returns 1 without shutdown; otherwise `serve` runs, a Serve error logs `Serve failed` and still falls through to `shutdown`, a shutdown error logs `Final dump failed`, and the code is 3 / 1 / 0 by precedence.
- `productionRuntime()` assigns `armSignals`, `signal.Stop`, `wire`, `serve` and `shutdown` directly — their directions match the seam exactly, so no adapter closure sits between the tested path and production.
- `main` is now parse → help/version → `setupLogging` → `warnOnNonWildcardBind` → `os.Exit(run(cfg, productionRuntime()))`; both `log.Fatalf` calls are gone, so no lifecycle fault renders as an INFO record that vanishes at `SUPKV_LOG_LEVEL=error`.
- `startupError{stage, err}` with `Error`/`Unwrap` wraps the three `wire` stages (`replication_bind`, `grpc_listen`, `metrics_listen`); `shutdown`'s doc comment now names the exit-code-3 path for a flush error.
- `cmd/sup-kv/run_internal_test.go` pins the call order and codes with fakes: signal before wire, clean 0, startup 1 with zero shutdown calls, Serve error with one shutdown call, dump failure 3, and Serve+dump → 3. `stopSignals` is asserted on every path.
- `cmd/sup-kv/main_test.go` adds `TestMain_fatalBindLoggedAtError` (occupied UDP port at `SUPKV_LOG_LEVEL=error` → exit 1 and exactly one parsed ERROR `Startup failed` record with a stage) and `TestMain_finalDumpFailureExitsThree` (dump directory replaced by a regular file, a write leaves the engine dirty, SIGTERM → exit 3 with the parsed `Final dump failed` ERROR record); the shared `logRecord`/`parseLogRecords` helper sits beside `parseSocketBuffersLine`.

## Task Commits

1. **Task 1: The exit table, the injectable run and the startup ERROR record** - `f343580` (feat)
2. **Task 2: Serve errors still dump, and a final-dump failure exits 3** - `be4a68f` (test)

**Plan metadata:** `docs(07-04): complete exit-contract plan` (this commit)

## Files Created/Modified

- `cmd/sup-kv/main.go` - exit table and stage constants, `startupError`, the `runtime` seam, `run`, `productionRuntime`, rewired `main`, stage-wrapped `wire` failures, corrected lifecycle doc comments
- `cmd/sup-kv/run_internal_test.go` - fake runtime with ordered call log and staged outcomes; the six `TestRun_*` rows for codes, ordering and precedence
- `cmd/sup-kv/main_test.go` - `exitStartupCode`/`exitDumpFailedCode`, the `logRecord`/`parseLogRecords` helper, and the fatal-bind and final-dump process tests

## Decisions Made

- **Directly assignable seam.** The `runtime` struct uses exactly the production signatures (`chan os.Signal` from `armSignals`, `chan<- os.Signal` for `signal.Stop`, `<-chan os.Signal` into `serve`/`shutdown`), so `productionRuntime` is a literal assignment rather than closures that could diverge.
- **Stage wrapper, not a second return.** `wire` keeps `(*wired, net.Listener, error)`; `startupError` carries the stage to `run`, which extracts it with `errors.AsType` and logs `stage` + `detail`.
- **Precedence in one switch.** The exit choice reads only `shutdownErr` and `serveErr`, so the two-failure path is structurally 3 and cannot regress to 1 by ordering.
- **Dirty-engine setup in the process test.** The plan's break-then-SIGTERM recipe alone exits 0: a fresh Prompt load marks the engine dirty, its initial dump succeeds inside the 50 ms debounce, and `flushDump` skips a clean engine. The test therefore writes against the blocked path (expected `Unavailable`, record installed, dump still pending) so shutdown's explicit `FlushTimeout` has a dump to retry and fails.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The stdlib `runtime` import is aliased `goruntime`**
- **Found during:** Task 1 (adding the planned `type runtime struct`)
- **Issue:** Go forbids a file-scope import name and a package-block type with the same name (`runtime already declared through import of package runtime`), and `printVersion` still needs `runtime.Version()`.
- **Fix:** Imported `goruntime "runtime"` and used `goruntime.Version()`; the seam type keeps the plan's name.
- **Files modified:** `cmd/sup-kv/main.go`
- **Commit:** `f343580`

**2. [Rule 3 - Blocking] The planned final-dump test sequence needed a dirty-engine step**
- **Found during:** Task 2 (designing `TestMain_finalDumpFailureExitsThree`)
- **Issue:** With the literal recipe (await serving, replace the directory, SIGTERM) the engine is clean: the fresh node's Prompt-load epoch dumps an empty generation inside the default 50 ms debounce, and `scheduler.flushDump` returns nil for a clean engine without touching the port — the process exits 0 and the test cannot prove code 3.
- **Fix:** After blocking the path, the test issues one `Set` and asserts its `Unavailable` answer; the failed covering dump leaves the engine dirty (with a pending rung), and shutdown's explicit `FlushTimeout` always attempts one dump, which fails → exit 3.
- **Files modified:** `cmd/sup-kv/main_test.go`
- **Commit:** `be4a68f`

**3. [Rule 3 - Blocking] Lint gates introduced by the new code**
- **Found during:** Task 1 and Task 2 lint runs
- **Issue:** `godoclint` required stdlib links in comments (`[errors.As]`, `[errors.Is]`); `err113` rejected `errors.New` inside test functions; `modernize` required `errors.AsType`; `noinlineerr` rejected `if err := ...; err != nil`; importing `status` shadowed existing `status` locals (revive import-shadowing).
- **Fix:** Static package-level sentinels (with a wrapped `fmt.Errorf`), `errors.AsType[*startupError]`, plain `err :=` assignments, and the import aliased `grpcstatus`.
- **Files modified:** `cmd/sup-kv/main.go`, `cmd/sup-kv/run_internal_test.go`, `cmd/sup-kv/main_test.go`
- **Commits:** `f343580`, `be4a68f`

**4. [Sequencing] Task 2's `Final dump failed` log line landed in Task 1**
- **Found during:** Task 1 (implementing `run`)
- **Issue:** `run` is the only place the shutdown error is observed; leaving its code-3 branch silent until Task 2 would have committed a half-implemented lifecycle.
- **Fix:** `run` logs `slog.Error("Final dump failed", "detail", ...)` from Task 1; Task 2 delivered the proof (three unit rows and the process test), so its commit is test-only.
- **Files modified:** `cmd/sup-kv/main.go` (Task 1), tests (Task 2)
- **Commits:** `f343580`, `be4a68f`

## Issues Encountered

- A stale `.git/gsd-plan-head-before-07-04` ledger from the v1.0 Phase 7 pointed at `80bef63` (not in this branch's history); it was reset to the plan-start HEAD `f9180c5` before measuring, following the 06-01/06-03, 07-01 and 07-02 precedent. `actuals.commits: 2` is measured from that ledger.
- `requirements.ready-ids` reports OBSV-04 blocked: the id is shared across sibling Phase-07 plans (#2388 shared-ID gate), so no requirement checkbox was marked and `requirements-completed` stays empty; 07-11 closes the phase-level checkboxes.
- No flaky or pre-existing failures: `make lint` and `make test` ran green on the committed tree (18 packages).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The exit contract is 0/1/2/3 with the dump failure winning, signals cover startup, and `run` is the testable seam, so 07-05 and the remaining plans can rely on a clean SIGTERM exiting 0 and a failed final dump exiting 3. Plan 07-10 still owes the OPERATIONS exit table and its `operations_test.go` anchors (the docs currently name exit 1 for a failed final dump). 7 plans remain in Phase 07 — ready for 07-05.

## Self-Check: PASSED

- Created file exists: `cmd/sup-kv/run_internal_test.go` — FOUND.
- Modified files exist: `cmd/sup-kv/main.go`, `cmd/sup-kv/main_test.go` — FOUND.
- Commits exist and are ancestors of HEAD: `f343580`, `be4a68f` — FOUND.
- Task verifies green: both `TestRun_*` runs and both process-test runs; `make lint` and `make test` exit 0 on the committed tree.
