---
phase: 03-transport
plan: 13
subsystem: ci-scheduling-and-phase-gate
tags: [xport-01, xport-02, scheduled-ci, twenty-of-twenty, race-soak, wr-06, phase-gate, red-runner, darwin-cross, bench-smoke, complete]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-12's red-gate flip (R-01/R-02 rows green) and its committed full-run evidence, which make a passing 20x run the expected verdict"
  - phase: 03-transport
    provides: "03-01..03-11's paced sending, bounded queues, validation, read-loop and dead-code work that the gate certifies"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the D-01 red-expectations table, the D-02 mutation stage and the scheduled-workflow conventions (pinned actions, explicit timeouts)"
provides:
  - "a dedicated scheduled red-r01-x20 job: -count=20 at the documented 20 000-key scale, -timeout 110m, job timeout-minutes 120, SUPKV_STAND_NIGHTLY=1 and SUPKV_RED_TESTS=1, no skip flags"
  - "make race-soak now repeats ./internal/stand/ with measured 6.53 s single-run cost, ~2.2 min projected at COUNT=20, whole target ~27 min at COUNT=20"
  - "the executed phase gate record: tidy, lint, test, full red-runner, darwin cross build/vet, bench smoke, one documented-scale R-01 run, clean tree"
affects:
  - "Phase 4 (Protocol) is unblocked: Phase 3 is 13/13 with XPORT-01 and XPORT-02 flipped to complete"
  - "the nightly schedule now owns XPORT-01's 20/20 acceptance; the job's first scheduled run is the remaining observation (budget or throughput) rather than a code gate"

actuals:
  tokens: 1087
  tasks: 2
  commits: 3
  plan_head_before: 22f8caab3b21e4753b829fc80b6c5fb002245fa5
  plan_head_after: 8960b455f5e7bbf8df8281c22b50dcaa557dcf70

tech-stack:
  added: []
  patterns:
    - "A 20/20 acceptance gets its own scheduled job: the count, the scale and both timeouts (test binary and job) are explicit, and no skip/tolerance flag can turn a failing evidence run green."
    - "A soak set is extended only after the added package's single-run cost is measured; the measured number and its COUNT projection live in the Makefile comment that the reviewer can check."
    - "A phase gate is a recorded table of commands and exit statuses, not a summary assertion: every line was executed on the committed tree and its observed outcome is written down."

key-files:
  created:
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T225950Z/ (phase-gate red run, 7/7 OK + mutants killed)
  modified:
    - .github/workflows/scheduled.yml
    - Makefile

key-decisions:
  - "The 20x evidence is a dedicated red-r01-x20 job, not an extension of red-run: red-run's 60-minute budget cannot hold 20 documented-scale runs, and the 03-12 flip makes a failure a real transport verdict that must fail CI (RESEARCH Pitfall 2 / A5)."
  - "The job's evidence step uses compgen -G rather than red-run's `ls ... || true` idiom, because the plan's prohibition bans `|| true` anywhere in this job; the summary-absent branch states honestly that go test ran directly."
  - "The measured 19x cost input is the documented-scale single run (110.26 s), so 20 runs project to ~37 min inside the 120-minute job budget; the race-flake 120-minute timeout needs no raise after the stand addition (~27 min whole-target projection)."
  - "The gate recorded three red rows as designed (R-03 gateClosedUntilFullList, R-03 f14ReplacedNode, P-01 valueLimit/valueLimitEngine) and did not touch any bound, row or test to make the phase look greener."

patterns-established:
  - "CI evidence jobs that bypass the red-runner say so in an always() step instead of fabricating an evidence directory: the exit status is the verdict and the log states the contract."
  - "A measured-cost comment sits beside the target it justifies, so the budget review reads one artifact (Makefile) rather than a planning file."

requirements-completed: [XPORT-01, XPORT-02, XPORT-05, XPORT-08, XPORT-09]

coverage:
  - id: C1
    description: "The scheduled red-r01-x20 job runs the exact test with -count=20 at the documented scale under both env gates, with a 110-minute test timeout and a 120-minute job timeout and no skip/tolerance flags."
    requirement: XPORT-01
    verification:
      - kind: integration
        ref: "awk-extracted job block: run line carries -count=20 and TestRed_R01_pacedSync20k; timeout-minutes: 120; ruby YAML parse confirms job keys/env/steps"
        status: pass
    human_judgment: false
  - id: C2
    description: "One documented-scale R-01 run passes locally (20 000 keys x 1 KiB), the supporting evidence for the scheduled 20/20."
    requirement: XPORT-01
    verification:
      - kind: e2e
        ref: "SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'"
        status: pass
    human_judgment: false
  - id: C3
    description: "race-soak repeats internal/stand with the measured single-run cost documented, and the one-run sanity soak is green."
    requirement: XPORT-02
    verification:
      - kind: integration
        ref: "make race-soak COUNT=1 (7 packages ok, internal/stand 6.508 s; whole target 81.22 s); race-soak recipe grep carries ./internal/stand/"
        status: pass
    human_judgment: false
  - id: C4
    description: "The full phase gate ran in order and every result is recorded: tidy cleanliness, lint, tests, red-runner with mutants, darwin cross build/vet, bench smoke, one nightly-scale R-01 run, clean tree."
    requirement: XPORT-01
    verification:
      - kind: integration
        ref: "the eight gate rows in the gate table below; bash scripts/red-runner evidence 20261121T225950Z"
        status: pass
    human_judgment: false
  - id: C5
    description: "No bound, expectation row or test was weakened: the red rows that remain (R-03 x2, P-01 x2) are Phase 4/5 work and the three transport rows are green."
    requirement: XPORT-02
    verification:
      - kind: integration
        ref: "scripts/red-expectations.tsv unchanged by this plan; git diff of internal/stand/* and the TSV is empty across 03-13"
        status: pass
    human_judgment: false

duration: 1048s
completed: 2026-11-22
status: complete
---

# Phase 03 Plan 13: Scheduled 20x R-01 Evidence and the Phase Gate Summary

A dedicated nightly `red-r01-x20` job now carries XPORT-01's 20/20 acceptance at the documented 20 000-key scale with bounded test and job timeouts, `make race-soak` repeats `internal/stand` at a measured 6.53 s per run (WR-06), and the full phase gate ran green end to end on the committed tree.

## Performance

- **Duration:** 1048 s (17m 28s), measured from the persisted plan-start marker (start 2026-11-21T22:52:17Z, completed 2026-11-21T23:09:45Z UTC)
- **Tasks:** 2/2
- **Commits:** 3 measured from the plan ledger (`22f8caa..8960b45`), plus the metadata close-out commit
- **Files:** 2 source/config files + 10 evidence files (five run directories)
- **Estimate vs actual:** 45 000 estimated tokens vs 1 087 measured (chars/4 over the non-planning diff). The estimate was low-confidence and sized for the gate's wall-clock work, not its diff size; the realized diff is 48 added lines of CI/Makefile configuration.

## Accomplishments

**Task 1 — the 20/20 evidence duty is scheduled, and WR-06 is measured.**

`.github/workflows/scheduled.yml` gained a `red-r01-x20` job: `ubuntu-latest`, `timeout-minutes: 120`, `env: SUPKV_STAND_NIGHTLY: "1"` and `SUPKV_RED_TESTS: "1"`, the same pinned `checkout@v7`/`setup-go@v7` pair with `go-version-file: go.mod`, and one command:

```
go test -race -count=20 -timeout 110m ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'
```

No skip flag, no `continue-on-error`, no `|| true`: after the 03-12 flip a failure is a real XPORT-01 verdict and fails CI. The `if: always()` evidence step guards its glob with `compgen -G` and, when no red-runner evidence directory exists (the normal case — this job invokes `go test` directly), says so in the log instead of faking a summary. The workflow header's job inventory names the new job.

Cost was pinned by measurement, not assumption: the single documented-scale run measured **110.26 s**, so 20 runs project to ~37 minutes — inside the 120-minute job budget and consistent with RESEARCH A5's 25-40-minute estimate.

`Makefile`'s `race-soak` now includes `./internal/stand/`. The measured `go test -race -count=1 ./internal/stand/` wall time is **6.53 s** (package report 6.508 s), projecting ~2.2 minutes at `COUNT=20`; the red tests inside the package skip without `SUPKV_RED_TESTS`, so no red scenario inflates the soak. The whole target measured 81.22 s at `COUNT=1` (the chaos package is the bulk), projecting ~27 minutes at the default count — the 120-minute `race-flake` job timeout stands unchanged. `make race-soak COUNT=1` is green across all seven packages.

**Task 2 — the phase gate executed in order, each result recorded.**

| # | Command | Exit | Observed outcome |
|---|---------|------|------------------|
| 1 | `go mod tidy && git diff --exit-code go.mod go.sum` | 0 | no dependency change |
| 2 | `make lint` | 0 | config verify + strict run clean (1.18 s) |
| 3 | `make test` | 0 | every package ok (140.13 s); repl 97.3%, storage 100%, clock/version 100%, udprepl 86.3%, stand 82.1%, persistfile 89.6% |
| 4 | `bash scripts/red-runner` | 0 | 7 expectation rows OK, 0 errors; mutation stage green (control pass, both mutants killed on every listed seed); evidence `red-evidence/20261121T225950Z` (103.89 s) |
| 5 | `GOOS=darwin GOARCH=arm64 go build ./... && … go vet ./...` | 0 | cross build and vet clean |
| 6 | `make bench-smoke` | 0 | all three benchmarks printed p50-ms/p99-ms: dump 60.51/60.51, replication burst 0.0307/0.0307, confirmation 1.461/1.461 |
| 7 | `SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'` | 0 | `--- PASS: TestRed_R01_pacedSync20k (110.26s)`, 20 000 keys listed on B |
| 8 | `git status --porcelain` | 0 | empty after committing the gate's own evidence directory |

The red-runner's green set is R-01, R-02 drain and R-02 burst; R-03's two rows and P-01's two rows stay red for Phases 4/5. `scripts/red-expectations.tsv` was not edited by this plan.

## Task Commits

| Task | Commit | Subject |
|------|--------|---------|
| 1 | `f375357` | feat(03-13): schedule the 20x R-01 job and add the stand to race-soak |
| 2 | `8960b45` | test(03-13): record the phase-gate red-runner evidence |
| — | `d368add` | test(03-12): commit the halt-resolution red-gate evidence (precondition, see deviations) |

## Files Created/Modified

**Modified**
- `.github/workflows/scheduled.yml` — job-inventory comment line; the `red-r01-x20` job (timeouts, env, pinned actions, the exact 20x command, the `always()` evidence step).
- `Makefile` — `./internal/stand/` appended to `race-soak`; the measured-cost/projection comment (6.53 s single run, ~2.2 min stand at COUNT=20, 81 s / ~27 min whole target, why reds do not inflate it).

**Created**
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T225950Z/` — the gate run's `go-test.json` + `summary.txt` (7/7 OK, mutation stage green).
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T223807Z/`, `20261121T224014Z/`, `20261121T225048Z/` — the 03-12 halt-resolution runs, previously left untracked (see deviations).

## Decisions Made

- **Dedicated job, not a stretched red-run.** `red-run`'s 60-minute timeout cannot hold 20 documented-scale runs; the new job carries its own 120-minute budget while `-timeout 110m` bounds the test binary, so a wedge fails rather than holding a runner (T-3-44).
- **`compgen -G` instead of `|| true`.** The plan's prohibition forbids `|| true` anywhere in this job, so the evidence step tests the glob with a bash builtin; the else branch states the direct-`go test` contract honestly.
- **Measure before adding to the soak.** The stand's 6.53 s single-run measurement (and the 81.22 s whole-target run) justified the addition without touching the race-flake timeout.
- **No gate weakened.** Every command ran with its real exit status; the four non-transport red rows remain red and neither the TSV nor the frozen stand tests changed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Committed the 03-12 halt-resolution evidence left untracked**
- **Found during:** Task 1's pre-commit `git status` check
- **Issue:** Three red-evidence directories from 03-12's resolution (`20261121T223807Z`, `20261121T224014Z`, `20261121T225048Z` — the accepted full-speed run among them) were on disk but never committed, so Task 2's "`git status --porcelain` is empty after the gate" could not hold without either discarding evidence or committing it.
- **Fix:** Committed them unmodified as `d368add`, with the accepted run identified in the message.
- **Files modified:** 6 evidence files (2 per directory)
- **Verification:** `git status --porcelain` empty after the commit; the `225048Z` summary still shows 7/7 OK and the mutant kills.
- **Committed in:** `d368add`

### Plan-scope notes

- Task 2's commit is evidence-only: the gate executes commands rather than editing code, so its durable product is the recorded run (mirroring 01-04's zero-change gate precedent, which committed nothing because no evidence artifact was produced).

---

**Total deviations:** 1 auto-fixed (Rule 3)
**Impact on plan:** none on scope — the fix made the plan's own clean-tree criterion meaningful; no production file, bound, row or test was touched.

## Issues Encountered

- **The 20x job's hosted-runner cost is projected, not measured.** The local documented-scale run is 110.26 s on a 3.88 GHz performance-governed host; a slower hosted runner projects longer. The 120-minute budget absorbs it per A5, and the first scheduled run is the observation — if it exceeds the budget the finding is recorded and the budget raised rather than the count reduced (20/20 is the acceptance).
- **The 03-12 evidence gap** (untracked run directories) is recorded above; it was a close-out omission, not a runner defect.

## Threat Flags

None — no new endpoint, auth path or trust boundary. T-3-44 is implemented by the two explicit timeouts; T-3-45 by the skip-free run lines and the verbatim gate table; T-3-46 by the measured soak cost; T-3-SC by the `go mod tidy` gate.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 3 is complete: 13/13 plans, all thirteen XPORT requirements delivered. XPORT-01 and XPORT-02 flip to complete on this plan's evidence; XPORT-05/08/09 were already complete.
- Phase 4 (Protocol) is unblocked: the paced transport, bounded queues, validation classes, read loop and Outbox/view seams it depends on are in place and gated.
- The nightly 20/20 run is the standing acceptance for XPORT-01; the scheduled job's first execution is an observation, not a code gate.

---
*Phase: 03-transport*
*Completed: 2026-11-22*

## Self-Check: PASSED

- **Key files exist:** `.github/workflows/scheduled.yml`, `Makefile`, this SUMMARY, and the five red-evidence directories (`20261121T223807Z`, `224014Z`, `225048Z`, `225950Z`, `230732Z`).
- **Commits resolve:** measured `git rev-list --count 22f8caa..8960b45` = 3; all three hashes (`d368add`, `f375357`, `8960b45`) are ancestors of HEAD.
- **Acceptance criteria + plan verification re-run in full after the SUMMARY was written:**
  - `make lint` — exit 0 (1.20 s), config verify + strict run clean.
  - `make test` — exit 0 (128.88 s), every package ok.
  - `bash scripts/red-runner` — exit 0, 7 expectation rows OK, 0 errors, mutation stage green; evidence `red-evidence/20261121T230732Z` (110.47 s).
  - `GOOS=darwin GOARCH=arm64 go build ./... && … go vet ./... && go mod tidy && git diff --exit-code go.mod go.sum` — exit 0, no dependency diff.
  - `make bench-smoke` — exit 0, p50-ms/p99-ms on all three benchmarks (62.96, 0.01065, 1.484 ms).
  - `SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'` — exit 0 (109.25 s).
  - The two Task 1 greps (`red-r01-x20` block extraction; `race-soak` recipe) both PASS.
- **Tree state:** `git status --porcelain` after the re-run shows only this SUMMARY and the self-check's evidence directory — both committed by the close-out; no unexpected modification. Gate row 8's clean-tree check was taken before this file existed, as the plan prescribes.

