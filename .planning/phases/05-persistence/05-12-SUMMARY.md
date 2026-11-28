---
phase: 05-persistence
plan: 12
subsystem: persistence-gate
tags: [persist-01, persist-02, persist-03, persist-04, persist-05, persist-06, persist-07, persist-08, persist-09, persist-10, persist-11, persist-12, d-01-flip, red-gate, mutation-kill, phase-gate, p-01, status-complete]
requires:
  - phase: 05-persistence
    provides: the single-source value limit, the configured-limit decoder and the census codec from 05-01/05-02
  - phase: 05-persistence
    provides: the load outcomes, evidence preservation, dirty start and skip accounting from 05-04..05-06
  - phase: 05-persistence
    provides: the injected clock/timer seam, bounded retry ladder and the simulator dump-notify proof from 05-07
  - phase: 05-persistence
    provides: the confirmation/flush/shutdown deadlines and the five-role engine split from 05-08/05-09
provides:
  - "the seven-green expectation table: both P-01 rows flipped, the other five rows and the header untouched"
  - "two committed red-runner evidence runs (the flip run and the final-tree gate run), each 7/7 rows OK with both mutants killed"
  - "the Phase-5 gate record on the committed tree: lint, race tests, darwin build/vet and bench-smoke green"
affects: [phase-6, phase-8-flake-owner, phase-9]
actuals:
  tokens: 5453
  tasks: 2
  commits: 2
  plan_head_before: 21bdb24a8eb6d0340b6662d6e220e61c28049136
  plan_head_after: 9b8008a33950b3048d7e466f6de5756fb0ceed69
tech-stack:
  added: []
  patterns:
    - "A phase-closing flip is proven both ways: the frozen tests run directly before the flip, and the runner's event stream after it records observed=pass,run — never merely the absence of a fail."
    - "Gate evidence keeps the honest first-run failure and its standalone/rerun proof beside the green rerun; an out-of-scope flake earns a deferred-items row, never a weakened bound."
key-files:
  created:
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T170042Z/ (go-test.json, summary.txt — the flip run)
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T172719Z/ (go-test.json, summary.txt — the final-tree gate run)
    - .planning/phases/05-persistence/05-12-SUMMARY.md
  modified:
    - scripts/red-expectations.tsv
    - .planning/phases/05-persistence/deferred-items.md
key-decisions:
  - "Both frozen P-01 tests were executed directly before the flip — the engine test PASS in 1.06 s, the stand test PASS in 4.18 s (63 000- and 64 000-byte subtests both PASS) — so the green rows rest on a genuine pass, not on the parser's green tolerance; the runner then recorded observed=pass,run for both."
  - "The flip is exactly the two data fields (red -> green) in scripts/red-expectations.tsv; the header, the other five rows and the file's whitespace are untouched, and the frozen runner and tests are byte-identical against the phase-start base 4f81fd7."
  - "The full gate ran in the plan's order on the committed flip; the first make test hung in TestScenario_dumpNotify and the package-alone run hit the known ratio flake, so the record keeps both failures visible and the green full-suite rerun, with no test, bound or runner line changed (the 04-13 phase-gate precedent)."
  - "The red-runner evidence from the flip run (20261122T170042Z) and the final-tree run (20261122T172719Z) is committed, matching the phase-3/4 evidence convention, so the phase verifier can replay the exact event stream."
  - "Both flake classes are deferred to the Phase-8 flake owner in deferred-items.md with their standalone and rerun evidence; 05-12's TSV-only diff cannot have caused either (the 05-09 precedent for the ratio flake)."
patterns-established:
  - "A flip's green row is evidence-backed by the runner's observed action set (pass,run), so a skip or an absence can never masquerade as the flip's success."
  - "A phase gate is recorded as a command/exit-status table on the committed tree, including the honest first-run failure and the evidence that the rerun is a verdict rather than a disabled assertion."
requirements-completed: [PERSIST-01, PERSIST-02, PERSIST-03, PERSIST-04, PERSIST-05, PERSIST-06, PERSIST-07, PERSIST-08, PERSIST-09, PERSIST-10, PERSIST-11, PERSIST-12]
coverage:
  - id: D1
    description: "Both P-01 rows read green and the other five rows stay green (0 red, 7 green); the frozen runner and both frozen tests are byte-identical to the phase-start base."
    requirement: "PERSIST-01"
    verification:
      - kind: process
        ref: "grep -v '^#' scripts/red-expectations.tsv | grep -c 'red$' -> 0; ... grep -c 'green$' -> 7"
        status: pass
      - kind: process
        ref: "git diff --quiet 4f81fd7a04ea6f6f137c0908538d0f13aa26fc32 -- scripts/red-runner internal/stand/stand_red_test.go internal/persistfile/red_value_limit_test.go -> exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both frozen P-01 tests genuinely pass before the flip, and the full red runner is 7/7 OK with the control arm passing and both mutants killed after it."
    requirement: "PERSIST-01"
    verification:
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/persistfile/ -run '^TestRed_P01_valueLimitEngine$' -> PASS (1.06 s)"
        status: pass
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_P01_valueLimit$' -> PASS (4.18 s; 63000-bytes PASS, 64000-bytes PASS)"
        status: pass
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 bash scripts/red-runner (flip run) -> exit 0; evidence 20261122T170042Z: 7 rows, 0 errors; both P-01 rows observed pass,run; control 4/4; both mutants killed on 848612892191557850"
        status: pass
    human_judgment: false
  - id: D3
    description: "The full red runner is green again on the committed tree, so the flip's proof describes what is committed."
    verification:
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 bash scripts/red-runner (final tree) -> exit 0; evidence 20261122T172719Z: 7 rows, 0 errors; both mutants killed"
        status: pass
    human_judgment: false
  - id: D4
    description: "make lint is green with the vendored strict config verified."
    verification:
      - kind: process
        ref: "make lint -> exit 0 (go tool golangci-lint config verify; go tool golangci-lint run ./...)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The full race suite is green on the final tree. The first run failed on a load-sensitive harness hang in the untouched repl package and a package-alone run hit the known ratio flake; both passed standalone and the full rerun was green without any test, bound or assertion change."
    verification:
      - kind: integration
        ref: "make test (first run) -> FAIL: internal/app/repl test binary timed out at 600.054 s in TestScenario_dumpNotify/a_confirmed_write_is_announced_and_converges_the_peer (writeAndConfirm, sim_dump_test.go:309); every other package ok"
        status: fail
      - kind: integration
        ref: "SUPKV_RED_TESTS unset, go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_dumpNotify$' -> PASS (1.1 s)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestCore_keyCountCostFlat$' -> PASS (ratios 0.50 and 0.20 against the 2.0 bound)"
        status: pass
      - kind: integration
        ref: "make test (rerun) -> exit 0: internal/app/repl ok 276.003 s coverage 93.1%; every package ok"
        status: pass
    human_judgment: true
    rationale: "The first full-suite run failed on a load-sensitive virtual-time harness hang and the known ratio flake in a package 05-12's TSV-only diff does not touch; the rerun was green. The verifier should classify the flake exception rather than the executor self-certifying it (the 05-09 precedent)."
  - id: D6
    description: "The darwin cross build and vet are green."
    verification:
      - kind: process
        ref: "GOOS=darwin go build ./... && GOOS=darwin go vet ./... -> exit 0"
        status: pass
    human_judgment: false
  - id: D7
    description: "The benchmark smoke runs every benchmark once on the real paths and reports its p50/p99 lines."
    verification:
      - kind: process
        ref: "make bench-smoke -> exit 0; BenchmarkDump10kKeys 61.18 p50-ms/61.18 p99-ms, BenchmarkReplicationBurst 0.03727 p50/p99, BenchmarkConfirmationLatency 1.628 p50/p99"
        status: pass
    human_judgment: false
  - id: D8
    description: "No unexpected tracked file changed outside the plan's reviewed set, and the committed tree is clean."
    verification:
      - kind: process
        ref: "changed=$(git diff --name-only HEAD) -> empty; git status --porcelain -> empty after the task commits"
        status: pass
    human_judgment: false
duration: 30min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 12: P-01 Flip and Phase Gate Summary

**Both P-01 expectation rows are green by a two-line data flip with the runner and both frozen tests byte-identical, the full red gate is 7/7 OK with both mutants killed on the committed tree, and the Phase-5 gate is recorded green — lint, race tests (after a documented load-sensitive hang rerun), darwin build/vet and bench-smoke — with no test or bound weakened.**

## Performance

- **Duration:** 30min
- **Started:** 2026-11-22T16:58:20Z
- **Completed:** 2026-11-22T17:28:00Z
- **Tasks:** 2
- **Files modified:** 2 authored (`scripts/red-expectations.tsv`, `deferred-items.md`) plus two committed red-runner evidence directories and this summary.

## Accomplishments

- `TestRed_P01_valueLimit` and `TestRed_P01_valueLimitEngine` flipped from `red` to `green`; the header and the other five rows are untouched. The file now carries **0 red and 7 green data rows**.
- Before the flip, both frozen tests were run directly: the engine twin passed in 1.06 s, and the process twin passed in 4.18 s with both the 63 000-byte and the 64 000-byte subtests green — so the flip records a genuine demonstrable pass, not merely the runner's tolerance of a non-fail action.
- The full red gate passed twice with an identical verdict: **7 expectation rows checked, 0 errors**, both P-01 rows observed `pass,run`, the control arm passing all four corpus seeds, and both mutants (`sync-without-tombstones`, `periodic-sync-disabled`) killed on seed `848612892191557850` (evidence `20261122T170042Z` and `20261122T172719Z`).
- The frozen contract held: `git diff --quiet 4f81fd7 -- scripts/red-runner internal/stand/stand_red_test.go internal/persistfile/red_value_limit_test.go` exits 0 against the phase-start base, so no edit committed by an earlier plan slipped in.
- `make lint` (config verify plus the strict run), `GOOS=darwin go build ./... && GOOS=darwin go vet ./...`, and `make bench-smoke` all exited 0 on the committed tree. The bench smoke reported all three p50/p99 lines on the real paths.
- `make test` is green on the final tree after an honest rerun: `internal/app/repl` passed in 276.003 s at 93.1 % coverage and every other package reported ok. The first run's hang and the package-alone ratio flake are documented below and deferred with their standalone evidence; no test, bound or assertion was altered.

## Task Commits

1. **Task 1: Flip the two P-01 rows and run the full red gate** — `7851cfc` (feat), including the flip run's evidence.
2. **Task 2: Full project gate on the final tree** — `9b8008a` (test), including the final-tree red-runner evidence and the two deferred-items rows.

**Plan metadata:** the `docs(05-12)` completion commit (this commit).

## Files Created/Modified

- `scripts/red-expectations.tsv` (modified) — the two P-01 data rows flipped `red` -> `green`; nothing else in the file changed.
- `.planning/phases/05-persistence/deferred-items.md` (modified) — two rows: the `TestScenario_dumpNotify` harness hang and the repeat `TestCore_keyCountCostFlat` ratio flake, each with its standalone and rerun evidence.
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T170042Z/` (created) — the flip run's `go-test.json` and `summary.txt`.
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T172719Z/` (created) — the final-tree gate run's `go-test.json` and `summary.txt`.
- `.planning/phases/05-persistence/05-12-SUMMARY.md` (created) — this record.

## Decisions Made

- **The green rows rest on a direct run, not on parser tolerance.** A `green` row only forbids a fail event, so a skip would also pass the table check. Running both frozen tests with `SUPKV_RED_TESTS=1` before the flip (engine 1.06 s; stand 4.18 s, both sizes) proves the behavior shipped; the runner's `observed=pass,run` then pins that the same genuine pass is what the committed table records.
- **The flip stayed a pure data change.** No runner flag, no test edit, no bound change: only the second field of the two P-01 rows moved. The phase-start-base diff gate (not the working-tree diff) proves this against edits that earlier plans may have committed.
- **The gate record keeps its failures.** The first `make test` timed out the `internal/app/repl` binary at 600 s in `TestScenario_dumpNotify`, and the package-alone run failed the known `TestCore_keyCountCostFlat` ratio gate. Both are load-sensitive and outside this plan's TSV-only diff; both passed standalone (1.1 s and 0.50/0.20 ratios) and the full rerun was green. Reporting only the rerun would hide a real observation, so both are in the coverage block and in `deferred-items.md`.
- **The runner evidence is committed, not left untracked.** Two evidence directories join the phase-3/4 convention so the verifier can replay the exact `go test -json` event stream that produced each verdict.

## Deviations from Plan

None - plan executed exactly as written.

The only out-of-plan write is `deferred-items.md`, which is the plan's own reporting mechanism: task 2's action says a failure is reported and never worked around, and the phase's deferred-items ledger is where prior plans recorded out-of-scope flakes. No source, test, runner or expectation beyond the two-row flip was touched.

## Issues Encountered

1. **First full-suite run: `internal/app/repl` binary timeout (600.054 s) in `TestScenario_dumpNotify`.** The goroutine dump showed `writeAndConfirm` blocked at `sim_dump_test.go:309` (`<-setRet`) while the scheduler select sat idle. Standalone, the test passes in 1.1 s; the full-suite rerun passed the package in 276.003 s. The likeliest sequencing hole (analysis, not a verified root cause): `awaitActivity` counts the Set goroutine's own `Config.After(ConfirmTimeout)` arm as new activity, so under load it can return before the scheduler's `rearm` books the debounce; the test then advances virtual time past that arming and stops advancing, leaving the confirming dump unable to fire. Recorded in `deferred-items.md`; no test was weakened.
2. **Known ratio flake, repeat: `TestCore_keyCountCostFlat`.** The package-alone race run measured 4 684 ns/op at 1 000 keys vs 16 566 ns/op at 50 000 keys (ratio 3.54 > 2.0); the standalone re-run measured 0.50 and 0.20 and passed, and the full-suite rerun passed. This is the 05-09 deferred row (Phase 8 owner), re-recorded with this run's evidence.
3. **No other code issues.** Lint, darwin build/vet, bench-smoke and the red runner were green on their first gate invocation, and `git status --porcelain` was empty after the task commits.
4. **`roadmap.update-plan-progress 05` mis-targets the Progress table.** The phase-detail section updated correctly (`11/12` -> `12/12`, 05-12 checked), but the table match hit the first `5.` row — the **v1.0** "gRPC API and Binary" row (`11/12` -> `12/12`) — while the v1.1 "5. Persistence" row still reads `0/?`. The same mis-targeting is pre-existing on the v1.1 phases 3–4 rows (their v1.0 rows 3–4 already carry the v1.1 counts), so the table defect is left untouched for the phase verifier/orchestrator rather than hand-edited here; no file other than the phase-detail section was intended to change.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 5 is complete: all 12 plans have summaries, both P-01 rows are green, and the seven-row expectation table no longer carries a red row.
- Ready for phase verification. The verifier should classify D5's flake exception (first-run hang plus the known ratio flake, both standalone-proven and deferred to Phase 8) rather than treat the green rerun as a bare pass.
- Phase 6 (API and Semantics) is next; its `wait_for_replication` work rests on the Phase-5 confirmation outcomes (`ErrUnconfirmed`, `ErrDumpFailed`) and the frozen `persist.Engine` API delivered by 05-08/05-09.

## Self-Check: PASSED

- Files verified on disk: `scripts/red-expectations.tsv`, `.planning/phases/05-persistence/deferred-items.md`, and both evidence `summary.txt` files.
- Task commits verified as ancestors of HEAD: `7851cfc`, `9b8008a`.
- Table state re-checked after the commits: 0 red rows, 7 green rows; frozen runner and tests identical to `4f81fd7`.
