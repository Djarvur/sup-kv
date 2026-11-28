---
phase: 08-performance
plan: 09
subsystem: test-harness-diagnostics
tags: [performance, flake-class, harness, readiness, diagnosis, spawn-serializer, doc-walkthrough, perf-02, perf-06]
requires:
  - phase: 08-performance
    provides: the shared claim-while-held port allocator (08-06) and the perf rows proven on the stand harness before the diagnostic refactor touches the same helpers (08-07)
provides:
  - a standNode that stores its Wait result and explains an already-exited child (pid, exit code/Wait text, stdout and stderr tails)
  - reapAfterKill with three documented outcomes, keeping the genuine SIGKILL -1 assertion in killAndReap
  - awaitNodeServing/requireNodeServing that fail a dead child on the first poll instead of waiting it out
  - classified readiness budgets (processServingDeadline 30 s harness start budget, logEventDeadline 30 s) with servingDeadline, exitDeadline and pollEvery untouched
  - a release-after-reap spawnSem serializer capping the cmd/sup-kv test binary at eight live spawned children
  - the two grpcurl doc walkthroughs serialized with their per-block budget and process-group reap intact
affects: [08-10, PERF-02, PERF-06, internal/stand, cmd/sup-kv]
actuals:
  tokens: 9389
  tasks: 3
  commits: 3
plan_head_before: 39ca6d580f49e4f911baceaa1ac061a27ba05063
plan_head_after: 8ee2999206c31345d6d48645cd61a7ebbf908052
tech-stack:
  added: []
  patterns:
    - "Store the Wait result before closing the exited channel: the closure is the happens-before fence, so the diagnosis is readable repeatedly without consuming the one-buffered exit channel"
    - "Classify a budget by kind instead of raising a shared one: a freshly spawned child's first wait gets a documented harness start budget, a poll of an already-running node keeps servingDeadline, and a log-delivery wait gets its own budget"
    - "A release-after-reap semaphore is the structural cure for process-start starvation; the cleanup registered before node.reap frees the slot only after the child is gone"
    - "A dead child is diagnosed through an error-returning wait core (awaitNodeServing) with a thin fatal wrapper at the call sites, so the failure message carries the evidence"
key-files:
  created:
    - internal/stand/kill_and_reap_test.go
  modified:
    - internal/stand/stand.go
    - cmd/sup-kv/main_test.go
    - cmd/sup-kv/examples_test.go
    - cmd/sup-kv/readme_test.go
key-decisions:
  - "The 30 s first-wait budget is recorded as a harness start budget, not a product bound: the structural cure (at most eight live spawned children) lands first, servingDeadline still governs every poll of a running node, and exceeding the bound is a finding to report rather than a reason to raise it."
  - "exitStatus returns (bool, error) instead of the plan's (error, bool): revive's error-return rule is a hard lint gate; the stored result may legitimately be nil, so the bool is the exited marker."
  - "The cmd/sup-kv first-wait call sites use requireNodeServing, a fatal wrapper over awaitNodeServing: it keeps the error-returning core testable and avoids seventeen inline error-handling blocks (noinlineerr)."
  - "TestMain_readinessDiagnosis was added beyond the three named verify tests: it drives the error-returning core on a dead child directly and gives awaitNodeServing's timeout parameter a second value, which unparam requires."
  - "The stand package's launchNode lost its env parameter: unparam proved every caller passed nil, and the inherited environment is unchanged (no behavior change)."
  - "Only the two doc walkthroughs were serialized; every other test keeps its parallel marker (43 t.Parallel calls remain in main_test.go)."
patterns-established:
  - "Diagnosis-first kill paths: a kill error is never the whole message — the pid, the Wait status and the captured output tails travel with it."
  - "Budget classes in the harness: start (processServingDeadline), running-node readiness (servingDeadline), log delivery (logEventDeadline) and exit (exitDeadline), each with its basis in code."
requirements-completed: [PERF-02, PERF-06]
coverage:
  - id: D1
    description: "An already-exited child is diagnosed by reapAfterKill and by the readiness wait: pid, exit code/Wait status, and the captured stderr and stdout tails; the genuine SIGKILL path still returns nil and killAndReap still proves the -1 shape."
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestKillAndReap|^TestExitDiagnosis|^TestAwaitNodeServing' -v (PASS 2.65 s: already-exited stderr half, --version stdout half, genuine -1 path, repeated exitDiagnosis, dead-child first-poll bound)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/stand/ -run 'TestStand_persist|TestStand_killNineRestart' -v (PASS 4.30 s: the refactored kill path keeps every existing scenario green)"
        status: pass
    human_judgment: false
  - id: D2
    description: "cmd/sup-kv readiness budgets are classified (processServingDeadline 30 s, logEventDeadline 30 s, unchanged servingDeadline/exitDeadline/pollEvery), a release-after-reap serializer caps live spawned children at eight, and a dead child fails the first readiness wait immediately with its diagnosis."
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_lifecycle|TestLog_formatText|TestLog_jsonDefault' -v (PASS 1.88 s)"
        status: pass
      - kind: unit
        ref: "go test -race -count=5 ./cmd/sup-kv/ -run 'TestMain_lifecycle|TestLog_formatText|TestLog_jsonDefault' (PASS 4.63 s at load 2.78 1.56 1.88)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_readinessDiagnosis' -v (PASS 0.01 s: exited child diagnosed on the first poll, exit code 2 and stderr tail asserted)"
        status: pass
      - kind: unit
        ref: "grep gate: processServingDeadline present, logEventDeadline count 4, spawnSem present, t.Parallel() count 43 (>= 20)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestExamplesDoc and TestReadmeQuickstart are serialized (no explicit parallel marker) while keeping their per-block budget and process-group reap; no other test's parallelism changed."
    requirement: PERF-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestExamplesDoc$|^TestReadmeQuickstart$' -v (PASS 8.14 s)"
        status: pass
      - kind: unit
        ref: "go test -race -count=3 ./cmd/sup-kv/ -run 'TestExamplesDoc|TestReadmeQuickstart|TestMain_lifecycle|TestLog_formatText' (PASS 20.75 s beside the process-spawning tests at load 2.69 1.80 1.94)"
        status: pass
      - kind: command
        ref: "! grep -qE '^[[:space:]]*t\\.Parallel\\(\\)' on examples_test.go and readme_test.go (no marker survived); full packages go test -race -count=1 ./cmd/sup-kv/ ./internal/stand/ PASS 10.75 s / 7.20 s; make lint PASS"
        status: pass
    human_judgment: false
duration: 13min
completed: 2026-11-24
status: complete
---

# Phase 08 Plan 09: Dead-Child Diagnosis, Classified Readiness Budgets and Serialized Doc Walkthroughs Summary

**A killed or crashed child now explains itself: `reapAfterKill` returns the pid, exit status and captured output tails instead of a bare `os: process already finished`, readiness budgets are classified by kind with the structural spawn cap landing first, and the two grpcurl walkthroughs run serialized beside their process-spawning siblings.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-11-24T19:32:19Z
- **Completed:** 2026-11-24T19:45:00Z
- **Tasks:** 3
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- `standNode` now stores its `cmd.Wait()` result under `exitMu` before closing `exited`, so `exitDiagnosis()` is callable repeatedly without consuming the one-buffered `exit` channel. It renders the pid, the `*exec.ExitError` code and `String()` (or the raw Wait error) and 2048-byte tails of both captured streams.
- `killAndReap` became the fatal wrapper over `reapAfterKill`, which has three documented outcomes: a genuine SIGKILL still returns nil after confirming the -1 Wait shape; `os.ErrProcessDone` waits for the reaper and returns `process exited before SIGKILL: child exited unexpectedly: process N exited: exit code 2 (...); stderr tail: ...; stdout tail: ...`; any other kill error carries the diagnosis when the child is gone.
- `awaitNodeServing` (and its `requireNodeServing` wrapper) checks the exit diagnosis on every poll, so a dead child fails on the first iteration with the same evidence — the F-6/07-11 gate row can now name the cause instead of the timing.
- `cmd/sup-kv` gained `processServingDeadline` (30 s harness start budget) for a freshly spawned child's first SERVING wait and `logEventDeadline` (30 s) for both duplicate-node log waits; `servingDeadline`, `exitDeadline` and `pollEvery` are byte-unchanged. The structural cure — a release-after-reap `spawnSem` cap of eight live spawned children with a named two-minute acquire failure — landed alongside, not instead of, the budget.
- `TestExamplesDoc` and `TestReadmeQuickstart` lost their explicit parallel marker only; both keep `docBlockBudget`, the process-group reap and every assertion, and their doc comments state why the serialization is deliberate. All other tests keep their parallel markers (43 remain).

## Task Commits

Each task was committed atomically:

1. **Task 1: Remember the child's exit status and diagnose a dead child** — `82df6f1` (fix)
2. **Task 2: Classify the readiness budgets and fail fast on an early exit** — `30f291b` (test)
3. **Task 3: Serialize the two grpcurl doc walkthroughs** — `8ee2999` (test)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/stand/kill_and_reap_test.go` — new (`//go:build linux`, internal package): `TestKillAndReap` (already-exited stderr half with `--node-id=0`, `--version` stdout half, genuine SIGKILL nil), `TestExitDiagnosis` (nil while running, repeated diagnosis after a clean exit 0) and `TestAwaitNodeServing` (healthy wrapper, dead-child first-poll diagnosis).
- `internal/stand/stand.go` — `standNode.exitMu/exitResult`, the store-before-close reaper, `exitStatus`, `outputTail`, `waitStatus`, `exitDiagnosis`, `reapAfterKill` + `killAndReap`, `awaitNodeServing` + `requireNodeServing`; the static sentinels `errChildExited`/`errKillShape`; `launchNode`'s unused `env` parameter removed.
- `cmd/sup-kv/main_test.go` — the two new budget constants and their basis, `spawnSem`/`spawnSemSlots`/`spawnAcquireDeadline` and the acquire/release ordering, `smokeNode.exitMu/exitResult`, `exitDiagnosis`/`awaitNodeServing`/`requireNodeServing`, `TestMain_readinessDiagnosis`, seventeen first-wait conversions, and both duplicate-node waits on `logEventDeadline`.
- `cmd/sup-kv/examples_test.go`, `cmd/sup-kv/readme_test.go` — parallel marker removed, serialization paragraph added to each doc comment, `//nolint:paralleltest` with the reason.

## Budget Classification and Load Evidence

The one number that moved is a harness start budget, recorded here with its evidence trail (D-07 fidelity; T-08-24):

- **F-5 / 06-02 deferred:** `TestMain_lifecycle` failed once with `timed out after 10s waiting for health SERVING` while passing standalone in 0.15 s — process-start starvation under the full suite, not a product deadline.
- **F-6 / 07-11 gate row:** `TestStand_persistLoadUnusableSingle: SIGKILL 2116885: os: process already finished` at load 11-12 on 16 cores — the child exited between the confirmed `Set` and the kill, and the bare fatal discarded its status and stderr.
- **F-7 / 04-protocol deferred:** the "Read a missing key" block produced an empty `grpcurl` capture under full-suite compile load; the same package passed 3/3 in isolation.
- **Structural cure:** `spawnSem` holds at most `spawnSemSlots = 8` live spawned binaries in the `cmd/sup-kv` test binary; `spawnAcquireDeadline = 2 min` makes a saturated serializer a named failure.
- **Fixed upper bound:** `processServingDeadline = 30 s`, and `servingDeadline = 10 s` still governs every poll of an already-running node. No claim, threshold or product deadline changed. If the 30 s bound is ever exceeded, that is a finding to report, never a reason to raise it.
- **Measured loads during this plan's runs:** 2.78 1.56 1.88 (five repetitions), 2.69 1.80 1.94 (three walkthrough repetitions), 3.92 2.29 2.10 (full affected packages). Every named test passed; the semaphore was never saturated.

## Decisions Made

- **Diagnosis-first, then budgets, then serialization.** The F-6 evidence path (pid/status/tails) landed before the budget work, exactly as D-07 routes it: a failure that discards its evidence is worse than the flake.
- **The started channel closure is the fence.** The reaper writes the Wait result under `exitMu` before `close(exited)`; readers wait on the channel and then read the field. This keeps every existing `<-node.exit` consumer untouched and makes `exitDiagnosis` repeatable.
- **The genuine path keeps its own assertion.** `reapAfterKill` returns an error naming an unexpected status instead of calling `assertExitCode`; `killAndReap` still fatals, and `TestKillAndReap` proves a live SIGKILL returns nil. `assertExitCode`'s -1 assertion was not weakened.
- **Budgets are classified, never raised.** `processServingDeadline` is only for a freshly spawned child's first wait; the single later poll of an already-answering node (the empty-gate subtest) stays `awaitServing(t, addr, 5*time.Second)`.
- **Both duplicate-node waits changed.** `TestLog_jsonDefault` and `TestLog_formatText` carried the identical `servingDeadline` log-delivery wait; both now use `logEventDeadline`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking lint gate] `exitStatus` returns `(bool, error)` instead of `(error, bool)`**
- **Found during:** Task 1 (first lint run)
- **Issue:** `revive`'s error-return rule rejects an `error` that is not the last return value; the plan named `(error, bool)`.
- **Fix:** Return `(bool, error)` and document the bool as the exited marker (the result may legitimately be nil for exit 0). No behavior change.
- **Files modified:** `internal/stand/stand.go`
- **Verification:** `go tool golangci-lint run ./internal/stand/...` clean; all Task 1 tests pass.
- **Committed in:** `82df6f1`

**2. [Rule 3 - Blocking lint gate] Static sentinels and `errors.AsType` in the kill path**
- **Found during:** Task 1 (lint run)
- **Issue:** `err113` rejects the dynamic errors `exitDiagnosis` and `reapAfterKill` returned; `modernize` requires `errors.AsType`; `godoclint` requires the `[os.ErrProcessDone]` link form; `errorlint` requires `%w` for an error value.
- **Fix:** Added `errChildExited` and `errKillShape` sentinels wrapped with context, switched both `errors.As` uses to `errors.AsType[*exec.ExitError]`, linked `[os.ErrProcessDone]`, and split the nil/non-nil Wait-error branches so the non-nil one wraps with `%w`.
- **Files modified:** `internal/stand/stand.go`
- **Verification:** `make lint` PASS; the diagnosis messages still carry every asserted substring.
- **Committed in:** `82df6f1`

**3. [Rule 3 - Blocking lint gate] `launchNode`'s `env` parameter removed in `internal/stand`**
- **Found during:** Task 1 (lint run)
- **Issue:** `unparam` reported "env always receives nil" once the new tests added call sites; the plan's launcher thread had no non-nil caller.
- **Fix:** Dropped the parameter and the now-dead `cmd.Env` branch; the child still inherits the process environment exactly as before.
- **Files modified:** `internal/stand/stand.go`
- **Verification:** `go tool golangci-lint run ./internal/stand/...` clean; the full stand package passes under `-race`.
- **Committed in:** `82df6f1`

**4. [Rule 2 - Missing critical coverage] `TestMain_readinessDiagnosis` added and `requireNodeServing` carries the start budget**
- **Found during:** Task 2 (lint run)
- **Issue:** `unparam` flagged `requireNodeServing`'s timeout (always `processServingDeadline`), and then `awaitNodeServing`'s timeout once the wrapper hardcoded it. The plan also expects the dead-child path asserted through the error-returning core, not only through a fatal wrapper.
- **Fix:** `requireNodeServing` takes no timeout and passes `processServingDeadline`; the new `TestMain_readinessDiagnosis` drives `awaitNodeServing` directly with `servingDeadline` on a `--node-id=0` child and asserts the first-poll bound, pid, exit code 2 and stderr tail. Two distinct timeout values keep `unparam` satisfied without a suppression.
- **Files modified:** `cmd/sup-kv/main_test.go`
- **Verification:** `go tool golangci-lint run ./cmd/sup-kv/...` clean; `TestMain_readinessDiagnosis` passes in 0.01 s; the five-repetition run is green.
- **Committed in:** `30f291b`

**5. [Rule 3 - Blocking lint gate] `paralleltest` directive placement**
- **Found during:** Task 3
- **Issue:** Removing the markers made `paralleltest` require a suppression; an inline directive failed `golines`, and the standalone directive failed `gci` until the formatter inserted its `//` separator.
- **Fix:** `go tool golangci-lint fmt` placed the `//nolint:paralleltest // serialized on purpose: ...` directive above each function with the formatter's separator, keeping the reason human-readable.
- **Files modified:** `cmd/sup-kv/examples_test.go`, `cmd/sup-kv/readme_test.go`
- **Verification:** `make lint` PASS; neither file carries a `t.Parallel()` call.
- **Committed in:** `8ee2999`

---

**Total deviations:** 5 auto-fixed (4 blocking lint-gate/Rule 3, 1 missing critical coverage/Rule 2). No architectural changes, no product bound or claim relaxed, no frozen file touched.

## Issues Encountered

- The 2048-byte stderr tail of a `--node-id=0` child begins inside the usage dump, so `--node-id` (the first error line) falls outside it. The tests assert `required unique 16-bit node ID` — the flag's stable help line inside the captured tail — which proves the tail is present without enlarging the budget.
- The first `TestExitDiagnosis` draft SIGTERMed immediately after launch and got `signal: terminated` instead of the clean exit 0: the signal handler was not yet armed. The test now waits for SERVING before SIGTERM, matching every other scenario.
- The RED evidence for Task 1 is captured at `.planning/phases/08-performance/red-evidence/20261124T193304Z/` (untracked, like the other red-evidence directories).

## User Setup Required

None — test-harness only; no external service, environment variable or package install (T-08-SC).

## Known Stubs

None. Every new helper is exercised by a passing test; no test was skipped and no `<verify>` was left unrun.

## Threat Flags

None. The change is test-only: no new endpoint, auth path, file-access pattern or dependency. T-08-22 is mitigated by the stored Wait result and the diagnosis on every kill/readiness failure; T-08-23 by serializing only the two heavyweight walkthroughs; T-08-24 by distinct budgets with their basis and untouched product bounds.

## Next Phase Readiness

Ready for wave 4/08-10. 08-10 can document the F-5/F-6/F-7 cures as shipped: the harness now names a dead child's pid, status and captured tails, the `cmd/sup-kv` test binary caps live spawned children at eight, and the two grpcurl walkthroughs run serialized while remaining block-budgeted. The 30 s start budget is a harness figure only — it must not be presented as a product deadline, and the measured loads (2.69-3.92 on 16 cores) belong beside it as evidence.

## Self-Check: PASSED

- Files: `internal/stand/stand.go`, `internal/stand/kill_and_reap_test.go`, `cmd/sup-kv/main_test.go`, `cmd/sup-kv/examples_test.go`, `cmd/sup-kv/readme_test.go` all found.
- Commits: `82df6f1`, `30f291b`, `8ee2999` all ancestors of HEAD.
