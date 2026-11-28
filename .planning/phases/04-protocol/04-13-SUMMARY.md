---
phase: 04-protocol
plan: 13
subsystem: protocol-gate
tags: [proto-01, proto-02, proto-03, proto-04, proto-05, proto-06, proto-07, proto-08, proto-09, d-01-flip, red-gate, mutation-seeds, cr-01, cr-02, phase-gate, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 05
    provides: "the identity-only egress queue (commit e42a20a) that carries CR-02's fix"
  - phase: 04-protocol
    plan: 08
    provides: "the bounded, peer-keyed, generation-checked listing queue (commit 1e06c58) that carries CR-01's fix"
  - phase: 04-protocol
    plans: [01, 02, 03, 04, 06, 07, 09, 10, 11, 12]
    provides: "the complete protocol behavior on which the two R-03 rows turn green"
  - phase: 03-transport
    provides: "the Phase-3 disposition ledger whose CR rows this plan closes"
provides:
  - "the two R-03 expectation rows flipped green with the frozen tests byte-identical"
  - "a re-selected periodic-sync-disabled kill seed (848612892191557850), recorded with its exhaustive 16-seed scan in the runner"
  - "CR-01 and CR-02 closed in the Phase-3 disposition ledger with resolving fix commits and named regression tests"
  - "the executed Phase-4 gate record: tidy, lint, tests, red-runner with mutants, buf lint/breaking, determinism, bench cost probes, darwin build/vet, clean tree"
affects:
  - "Phase 5 owns the two P-01 rows, still red; no P-01 row or test was touched"
  - "the milestone exit now holds the recorded phase-gate evidence"

actuals:
  tokens: 1521
  tasks: 3
  commits: 5
  plan_head_before: 58ff7700b1fa288a5c55b7deaa7788283277cebc
  plan_head_after: 234cf6886ada4cf4765ac70428fc21652b4ac360

tech-stack:
  added: []
  patterns:
    - "An expectation-table flip is a pure data change: the frozen tests stay byte-identical, only the TSV row and (when behavior drift is measured) the mutant kill list may move."
    - "Mutant kill seeds are re-selected by exhaustive corpus scan when behavior changes underneath them, and the scan result is recorded beside the list — the control arm is never relaxed."
    - "A phase gate is a recorded table of commands and exit statuses on the committed tree, including the honest first-run failure and the evidence for why a rerun is a verdict rather than a retry."

key-files:
  created:
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T064704Z/ (drift run: rows OK, periodic mutant survived)
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T065007Z/ (green run after seed re-selection)
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T070216Z/ (phase-gate run on the final tree)
  modified:
    - scripts/red-expectations.tsv
    - scripts/red-runner
    - .planning/phases/03-transport/03-REVIEW-DISPOSITION.md
    - .planning/phases/04-protocol/deferred-items.md

key-decisions:
  - "The flip changes exactly the two R-03 TSV rows; both P-01 rows stay red and the frozen tests keep an empty diff (D-01/T-4-37)."
  - "The periodic-sync-disabled mutant's drift was resolved by re-selection, not acceptance: an exhaustive 16-seed scan found first-seen listings now heal 15 of 16 seeds, and 848612892191557850 remains a verified kill where the periodic broadcast is still load-bearing (RESEARCH Pitfall 7/T-4-38)."
  - "The control arm is unchanged: the same four corpus seeds must pass the un-mutated test, and they did on every run."
  - "CR-01's fixing commit is 1e06c58 (plan 04-08: bounded, shared, generation-checked listing queue) and CR-02's is e42a20a (plan 04-05: identity-only egress with drain-time construction); both were verified by git show and by the presence of the named regression tests."
  - "The one full-suite failure was the pre-existing IN-03 wall-clock ratio gate (t.Parallel), not this plan: standalone ratios were 0.96/1.00, the full rerun passed, and no bound was touched; it is logged in deferred-items for Phase 8."

requirements-completed: [PROTO-01, PROTO-02, PROTO-03, PROTO-04, PROTO-05, PROTO-06, PROTO-07, PROTO-08, PROTO-09]

coverage:
  - id: D1
    description: "The two R-03 expectation rows read green while both P-01 rows stay red, and the frozen R-03 tests pass unmodified under SUPKV_RED_TESTS=1."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "grep gate over scripts/red-expectations.tsv (2 R-03 green, 2 P-01 red)"
        status: pass
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$' (PASS)"
        status: pass
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 -timeout 10m ./internal/stand/ -run '^TestRed_R03_f14ReplacedNode$' (PASS, 3.24 s)"
        status: pass
      - kind: other
        ref: "git diff --name-only -- '*_test.go' is empty"
        status: pass
    human_judgment: false
  - id: D2
    description: "The full red runner is green on the final tree: 7 rows OK, control passes, and both mutants are killed on their recorded seeds."
    verification:
      - kind: integration
        ref: "bash scripts/red-runner (exit 0; evidence 20261122T070216Z: 7 rows, 0 errors; control 4/4 pass; sync-without-tombstones and periodic-sync-disabled killed on 848612892191557850)"
        status: pass
    human_judgment: false
  - id: D3
    description: "CR-01 is recorded fixed in the Phase-3 disposition ledger, naming plan 04-08/commit 1e06c58 and its regression tests, with the hash resolving to a git object."
    requirement: PROTO-04
    verification:
      - kind: unit
        ref: "grep '| CR-01 | critical | fixed |.*[0-9a-f]{7,}' + git cat-file -e 1e06c58"
        status: pass
    human_judgment: false
  - id: D4
    description: "CR-02 is recorded fixed in the Phase-3 disposition ledger, naming plan 04-05/commit e42a20a and its regression tests, with the hash resolving to a git object."
    requirement: PROTO-03
    verification:
      - kind: unit
        ref: "grep '| CR-02 | critical | fixed |.*[0-9a-f]{7,}' + git cat-file -e e42a20a"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Phase-4 gate ran in the plan's order on the committed tree and every result is recorded: tidy cleanliness, lint, tests, red runner, proto lint/breaking, determinism, bench cost probes, darwin build/vet, clean tree."
    verification:
      - kind: integration
        ref: "the nine gate rows below; all green with the recorded IN-03 flake and its standalone/rerun evidence"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 13: R-03 Flip and Phase Gate Summary

**The two remaining R-03 red rows are green by expectation alone — frozen tests byte-identical — the mutant kill seed was re-selected after first-seen listings changed its sensitivity, CR-01/CR-02 are closed in the ledger, and the full Phase-4 gate is recorded green on the committed tree.**

## Performance

- **Duration:** 19min 39s (1 178 s)
- **Started:** 2026-11-22T06:44:46Z
- **Completed:** 2026-11-22T07:04:24Z
- **Tasks:** 3
- **Files modified:** 4 tracked (TSV, runner, disposition ledger, deferred items) + 3 red-evidence directories (6 files)

## Accomplishments

- **D-01's flip is a data change.** `TestRed_R03_gateClosedUntilFullList` and `TestRed_R03_f14ReplacedNode` changed from `red` to `green` in `scripts/red-expectations.tsv`; both P-01 rows stayed red; no test file changed (`git diff --name-only -- '*_test.go'` empty). Both frozen tests pass unmodified under `SUPKV_RED_TESTS=1` — the grpcapi gate test in 0.03 s and the stand F14 scenario in 3.24 s.
- **The predicted mutant drift was measured, not accepted.** The first full red run failed exactly as RESEARCH Pitfall 7 warned: all 7 expectation rows were OK, but `periodic-sync-disabled` survived seed `8612631937004137123` because first-seen listings now give the cluster an independent reconciliation path. An exhaustive scan of all 16 corpus seeds against the mutant found it is killed by exactly one seed (`848612892191557850`); the kill list was re-selected to that seed and the scan recorded in `scripts/red-runner`. The control arm stayed at its four seeds and passed every run. The failed drift run and the green rerun are both committed as evidence.
- **Phase-3 criticals are closed with traceable commits.** CR-01 is `fixed` naming plan 04-08, commit `1e06c58`, and regression tests `TestListing_hostileBounded`/`TestListing_sharedBuild`; CR-02 is `fixed` naming plan 04-05, commit `e42a20a`, and `TestEgress_identityOnly`/`TestEgress_noValueRetention`. Both hashes were verified with `git show --stat` (they carry the named tests) and `git cat-file -e`.
- **The Phase-4 gate is green and recorded.** Every gate command ran on the committed tree; the one failure was the pre-existing IN-03 wall-clock ratio flake, proven standalone (0.96/1.00) and cleared by a full rerun, with no bound, test or expectation touched.

## Gate Results

| # | Command | Exit | Result |
|---|---------|------|--------|
| 1 | `go mod tidy && git diff --exit-code go.mod go.sum` | 0 | no dependency change (run at gate start and again at gate end) |
| 2 | `make lint` | 0 | config verify + strict run clean |
| 3 | `make test` | 2 → 0 | first run failed on `TestCore_keyCountCostFlat` (ratio 2.80 > 2.0, IN-03 under chaos load); standalone ratios 0.96/1.00 passed; full rerun `EXIT=0` with `internal/app/repl ok 186.037 s`, coverage 93.0% |
| 4 | `bash scripts/red-runner` | 0 | 7 expectation rows OK, 0 errors; control 4/4 pass; both mutants killed on `848612892191557850`; evidence `red-evidence/20261122T070216Z` |
| 5 | `make buf-lint && make buf-breaking` | 0 | lint clean; the real FILE-policy comparison ran against `origin/master` (`d6b2a75`, proto now present, D-08 guard no longer defers) |
| 6 | `go test -run '^$' -bench '^BenchmarkHandle(DataRequest\|DataSync)$' -benchtime=2x ./internal/app/repl/` | 0 | DataRequest 2 232 / 2 860 ns/op; DataSync 8 728 / 13 486 ns/op (1 000 / 50 000 keys) |
| 7 | `go test -race -count=1 ./internal/app/repl/ -run '^TestChaos_determinismReplay$\|^TestSim_determinism_withExtensions$'` | 0 | `TestChaos_determinismReplay` PASS; the second pattern lives in the simnet package, whose test was then run directly: `TestSim_determinism_withExtensions` PASS |
| 8 | `GOOS=darwin GOARCH=arm64 go build ./... && GOOS=darwin GOARCH=arm64 go vet ./...` | 0 | cross build and vet clean |
| 9 | `git status --porcelain` | 0 | empty after committing the gate's own evidence and the deferred-flake note |

The failed first test run is recorded verbatim rather than hidden: `HandleDatagram(DataRequest): 1000 keys = 13165.0 ns/op, 50000 keys = 36879.0 ns/op, ratio 2.80`. The test is the pre-existing `t.Parallel` wall-clock ratio gate flagged as IN-03 in `03-REVIEW.md`; this plan changed no Go source, the standalone measurement was 0.96, and the rerun was green. It is logged in `04-protocol/deferred-items.md` with Phase 8 as owner.

## Task Commits

| Task | Commit | Subject |
|------|--------|---------|
| 1 | `a44e661` | feat(04-13): flip the R-03 expectation rows and re-select the periodic-sync kill seed |
| 1 | `1399780` | test(04-13): record the red-gate evidence (drift run + green run) |
| 2 | `d12a52e` | docs(04-13): close CR-01 and CR-02 in the Phase-3 disposition ledger |
| 3 | `b650fde` | test(04-13): record the phase-gate red-runner evidence |
| 3 | `234cf68` | docs(04-13): defer the keyCountCostFlat ratio flake to Phase 8 |

**Plan metadata:** this SUMMARY/STATE/ROADMAP/REQUIREMENTS commit follows this file.

Metrics are measured from the ledger base `58ff7700`: `git rev-list --count 58ff7700..234cf68` = 5 commits.

## Files Created/Modified

- `scripts/red-expectations.tsv` — the two R-03 rows flipped to `green`; P-01 and the three transport rows unchanged.
- `scripts/red-runner` — `MUTANT_SEEDS_PERIODIC_SYNC_DISABLED` re-selected to `848612892191557850`; the comment records the exhaustive 16-seed scan, the reason (first-seen listings, RESEARCH Pitfall 7) and that the control arm is unchanged.
- `.planning/phases/03-transport/03-REVIEW-DISPOSITION.md` — CR-01/CR-02 marked `fixed` with their fix commits and regression tests; frontmatter `updated` refreshed; totals and all other rows unchanged.
- `.planning/phases/04-protocol/deferred-items.md` — the IN-03 `TestCore_keyCountCostFlat` flake entry (open, owner Phase 8).
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T064704Z/` — the drift run: rows OK, `periodic-sync-disabled` survived.
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T065007Z/` — the green run after re-selection.
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T070216Z/` — the phase-gate run on the final tree.

## Decisions Made

- **Flip only the TSV rows.** The two R-03 tests stay byte-identical; the flip is the expectation table's one-line change per D-01. P-01's two rows remain Phase 5's work.
- **Re-select the mutant kill seed; never relax the control.** First-seen listings are now a real reconciliation path, so `periodic-sync-disabled` survives 15 of 16 corpus seeds. `848612892191557850` still kills it and is the recorded kill; the four-seed control arm is unchanged and passed.
- **The fixing commits are the ones whose diffs carry the fix.** CR-01 → `1e06c58` (bounded/shared/generation-checked listing queue) rather than the follow-up first-seen Hello commit; CR-02 → `e42a20a` (identity-only egress) rather than the later stale-answer commit. Both were confirmed by `git show --stat` and by the presence of the named regression tests in those commits.
- **The IN-03 flake is documented, not fixed.** It is a pre-existing wall-clock ratio gate in a file this plan does not own; the plan's prohibition limits the diff to the TSV/runner/ledger, so the flake is deferred to Phase 8 with the standalone and rerun evidence recorded.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The periodic-sync-disabled mutant survived on its old kill seeds**
- **Found during:** Task 1, first full `bash scripts/red-runner` run
- **Issue:** All 7 expectation rows were OK, but the mutation stage failed: `mutant periodic-sync-disabled: PASSED on seed 8612631937004137123`. First-seen listings (plan 04-08) added an independent reconciliation path, exactly as RESEARCH Pitfall 7 predicted. The plan anticipated this and prescribed re-selecting the kill seeds.
- **Fix:** Scanned all 16 committed corpus seeds against the mutant built through the runner's own overlay/sed path: it is killed only by `848612892191557850` and survives the other 15. Updated `MUTANT_SEEDS_PERIODIC_SYNC_DISABLED` to that seed and recorded the scan plus the reason in the script comment. Control seeds and their verdicts are unchanged.
- **Files modified:** `scripts/red-runner`
- **Verification:** `bash scripts/red-runner` exit 0 — 7 rows OK, control 4/4 pass, both mutants killed; evidence `20261122T065007Z`.
- **Committed in:** `a44e661` (seed change) and `1399780` (drift + green evidence).

**2. [Rule 3 - Blocking] The determinism command's second test name lives in another package**
- **Found during:** Task 3, gate step 7
- **Issue:** The plan's command scopes `-run '^TestChaos_determinismReplay$|^TestSim_determinism_withExtensions$'` to `./internal/app/repl/`, but `TestSim_determinism_withExtensions` is defined in `internal/app/internal/simnet/simnet_ext_test.go`. Run verbatim, only the chaos replay executes; the second named test silently never matches.
- **Fix:** Ran the plan's command as written (chaos replay PASS), then ran the named simnet test directly in its own package to satisfy the gate's intent.
- **Files modified:** none (gate invocation only)
- **Verification:** `go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_determinism_withExtensions$'` → PASS.
- **Committed in:** no code change; recorded in the gate table above.

**3. [Rule 3 - Blocking] One full-suite run failed on the pre-existing IN-03 ratio gate**
- **Found during:** Task 3, gate step 3
- **Issue:** `make test` failed with `TestCore_keyCountCostFlat` ratio 2.80 > 2.0 (DataRequest, 50 000 vs 1 000 keys). The test is the `t.Parallel` wall-clock ratio gate flagged as IN-03 in `03-REVIEW.md`, contaminated by the concurrent chaos corpus. This plan changed no Go source.
- **Fix:** No bound or test was touched. Confirmed the test passes standalone (ratios 0.96 and 1.00) and re-ran the failing gate; the full suite returned `EXIT=0` (`internal/app/repl` 186.037 s). Logged the flake in `04-protocol/deferred-items.md` for Phase 8 (IN-03's own fix: drop `t.Parallel` or assert operation counts).
- **Files modified:** `.planning/phases/04-protocol/deferred-items.md`
- **Verification:** standalone PASS; `make test` rerun exit 0.
- **Committed in:** `234cf68`.

---

**Total deviations:** 3 auto-fixed (all Rule 3, all anticipated or environmental; no assertion, bound or expectation was relaxed).

## Issues Encountered

The only gate failure was the documented IN-03 ratio flake; it was resolved by proving the standalone measurement and rerunning the gate, with the failure recorded verbatim in the gate table and deferred-items.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 4 is complete: 13/13 plans, all nine PROTO requirements ready to close, red gate green with the two R-03 rows flipped, and the Phase-3 criticals closed in the ledger.
- The P-01 rows remain red by design for Phase 5. The two remaining known flakes (IN-03 ratio gate, WR-04 `freeUDPPort`) stay with Phase 8, recorded in deferred items.
- Ready for phase verification.

## TDD Gate Compliance

This is the phase-closing gate plan; its only behavior change is the expectation-table data flip, so no artificial RED/GREEN pair was manufactured (as the execution context states). The real red evidence is the runner's own recorded event stream: `20261122T064704Z` captures the drift failure and `20261122T065007Z`/`20261122T070216Z` capture the green verdicts with the mutation stage.

## Self-Check: PASSED

- Created evidence files exist at all three red-evidence directories.
- All five task commits (`a44e661`, `1399780`, `d12a52e`, `b650fde`, `234cf68`) are ancestors of HEAD.
- The measured commit count is 5 from ledger base `58ff7700`, matching the frontmatter `commits: 5`.
- No test file changed (`git diff --name-only -- '*_test.go'` empty); no expectation row other than the two R-03 flips changed.
