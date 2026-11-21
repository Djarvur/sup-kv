---
phase: 02-stand-and-simulator-as-tests
plan: 11
subsystem: ci
tags: [scheduled-ci, nightly, cron, workflow-dispatch, documented-scale, f9, test-16, test-06, test-08, test-11, d-09]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: "plans 02-02/02-05 — the internal/stand harness (build-once binary, asymmetric-port pair, SIGKILL/SIGTERM lifecycle) and the standNightly hook with the red scenarios' documented-scale selection; plan 02-10 — the bench-smoke and race-soak Makefile targets (single command definitions); plan 02-13 — the red-runner's full assert-red + D-02 mutation stage; plan 02-17 — FuzzDecode/FuzzDumpDecode with committed corpora"
provides:
  - "internal/stand/stand_test.go — TestStand_emptyDiskRecovery's nightly selection: SUPKV_STAND_NIGHTLY=1 runs the review's F9 documented shape (10 000 keys × 1 KiB, reviewer's default flags) with the bound derived from the selected flags; the default 20-key shrunken-timer scale is byte-identical to before"
  - ".github/workflows/scheduled.yml — the six-job D-09 nightly set: chaos-seeds (validated workflow_dispatch seeds → SUPKV_SIM_FRESH_SEEDS, -shuffle=on -count=3), bench-smoke (make bench-smoke), red-run (SUPKV_STAND_NIGHTLY=1, full scripts/red-runner with no skip flags), race-flake (make race-soak), nightly-stand (SUPKV_STAND_NIGHTLY=1), deep-fuzz (both targets with -fuzztime); nightly cron + workflow_dispatch"
  - ".github/workflows/ci.yml — the TEST-05 policy header and a named, greppable non-skippable stand step (`go test -race ./internal/stand/`) in the non-cross gate; matrix and check-run names unchanged"
affects: [phases 3-5 (the red-run job keeps the expected-red floors asserted nightly; the flip changes only the expectation-table rows), Phase 7/8 (the scheduled set is the heavy-machinery home future performance jobs extend)]

# Actuals (#2632) — same estimateTokens scale as the plan estimate: chars/4 over the realized diff.
actuals:
  tokens: 5226
  tasks: 3
  commits: 2

plan_head_before: f31c2ea81360603d1ba91a64863a2799f8150ced
plan_head_after: 5263188b9b7501a30f479160d758e6c49f0807a8

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Env-var scale selection: SUPKV_STAND_NIGHTLY=1 switches one scenario between the CI-viable shrunken-timer scale and the review's documented scale; the default behavior is unchanged and the selection lives in the scenario, not the workflow"
    - "Derived bounds name their flags: paced = N × floor(1s/rate), margin = gate + 2×hello + 2×sync + slack; the nightly selection swaps only the flag values, so the expression stays honest at both scales"
    - "Nightly seeding uses the review's preload profile (64 workers round-robin over 8 connections); recovery is polled with one throttled prefix List, then every 1 KiB value is verified once"
    - "Scheduled workflow follows vulncheck.yml's schedule + workflow_dispatch triad: one cron file, contents: read, per-job timeout-minutes, and only the ci.yml action pins (checkout@v7, setup-go@v7)"
    - "Untrusted dispatch input is validated in the first job step (positive-integer regex) with a 1000 fallback written via $GITHUB_ENV (T-2-21)"

key-files:
  created:
    - .github/workflows/scheduled.yml
  modified:
    - internal/stand/stand_test.go
    - .github/workflows/ci.yml

key-decisions:
  - "The nightly F9 scale runs the reviewer's documented default flags (hello 1 s, sync 30 s, sync_response_rate 500; gate 1 s is the harness constant) — the exact stand configuration the F9 evidence row measured, not the shrunken harness timers"
  - "The recovery bound's margin is derived from the SELECTED flags at both scales (nightly: 1 s gate + 2 s hello + 60 s sync + 2 s slack = 65 s on top of the 20 s paced term); no magic wall-clock constant entered the expression"
  - "The post-recovery write convergence wait is derived from the selected flags too (nightly: 2×sync + 2×hello + 3 s) — the shrunken-timer expression (6 s) timed out at the documented 30 s cadence during local verification"
  - "Nightly values are a deterministic 1 KiB per key (the key padded with '.'): the List poll checks presence only, then a full value pass asserts every 1 KiB record's bytes"
  - "No artifact-upload action is introduced (release.yml uses none, and the plan forbids new action introductions): the red-evidence summary is printed into the workflow log and a deep-fuzz crasher is printed by go test into the job log for the human follow-up commit"
  - "The seeds dispatch input is validated in its own first step (grep -Eq '^[1-9][0-9]*$'), falls back to 1000 on any invalid/zero value, and is consumed only as a test-loop count env var"
  - "Job budgets: chaos 240 min (the one-off 10 000-seed proof costs 1-3 h per RESEARCH A4), bench 20, red-run 60, race-flake 120 (≥ 90 required), nightly-stand 60, deep-fuzz 40"
  - "Both stand jobs export SUPKV_STAND_NIGHTLY=1; nightly-stand deliberately does NOT set SUPKV_RED_TESTS (red tests skip there) — assert-red belongs to red-run, whose runner exports SUPKV_RED_TESTS itself"

patterns-established:
  - "Two stand jobs, one scale contract: the green nightly-stand and the red red-run select the same documented scale through SUPKV_STAND_NIGHTLY=1, while the red gate stays the runner's own SUPKV_RED_TESTS export"
  - "Workflow dispatch input gate: validate in a first step, fall back via $GITHUB_ENV; never interpolate the raw input into shell logic beyond the validation message"
  - "Every scheduled job carries an explicit timeout-minutes and every fuzz invocation a -fuzztime — the T-2-20 bound is structural, not a convention"

requirements-completed: [TEST-16, TEST-06, TEST-08, TEST-11]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "TestStand_emptyDiskRecovery selects the review's F9 documented scale under SUPKV_STAND_NIGHTLY=1 (10 000 keys × 1 KiB, documented default flags) while the default scale stays green and unchanged"
    requirement: "TEST-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_emptyDiskRecovery$' (default scale green) and SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestStand_emptyDiskRecovery$' (10 000 keys in 55.6 s within the 85 s derived bound)"
        status: pass
    human_judgment: false
  - id: D2
    description: "scheduled.yml carries the six-job D-09 set (chaos-seeds, bench-smoke, red-run, race-flake, nightly-stand, deep-fuzz) on a nightly cron plus workflow_dispatch, with contents: read, explicit timeouts and ci.yml action pins only"
    requirement: "TEST-16"
    verification:
      - kind: integration
        ref: "grep -c 'timeout-minutes:' = 6; grep -c 'workflow_dispatch:' = 1; Ruby YAML parse enumerates all six jobs; grep shows checkout@v7/setup-go@v7 with go-version-file: go.mod and zero secrets/upload actions"
        status: pass
    human_judgment: false
  - id: D3
    description: "TEST-16 acceptance tokens verbatim on schedule: 1000-seed chaos with -shuffle=on -count=3, and -bench . -benchtime=1x via make bench-smoke"
    requirement: "TEST-16"
    verification:
      - kind: integration
        ref: "grep -c 'shuffle=on\\|count=3\\|benchtime=1x\\|SUPKV_STAND_NIGHTLY\\|SUPKV_SIM_FRESH_SEEDS\\|red-runner' .github/workflows/scheduled.yml = 14"
        status: pass
    human_judgment: false
  - id: D4
    description: "Q1 parameterization: the workflow_dispatch seeds input flows into SUPKV_SIM_FRESH_SEEDS through a positive-integer validation step with a 1000 fallback; the one-off 10 000-seed proof command documented"
    requirement: "TEST-16"
    verification:
      - kind: other
        ref: "gh workflow run scheduled.yml -f seeds=10000 (documented; execution is post-merge GitHub Actions verification, not run from planning)"
        status: pass
    human_judgment: true
  - id: D5
    description: "Red-run job exports SUPKV_STAND_NIGHTLY=1 and invokes scripts/red-runner with no skip flags (assert-red + the D-02 mutation stage in full); nightly-stand exports SUPKV_STAND_NIGHTLY=1 only, so the review's section-15 set runs nightly across the two jobs"
    requirement: "TEST-06"
    verification:
      - kind: integration
        ref: "grep -c SUPKV_STAND_NIGHTLY = 2; the red-run step is exactly 'bash scripts/red-runner'; no --skip flag appears anywhere in the workflow"
        status: pass
    human_judgment: false
  - id: D6
    description: "Deep-fuzz job runs FuzzDecode and FuzzDumpDecode with -fuzztime on the schedule, one target per package, and never auto-commits crashers"
    requirement: "TEST-08"
    verification:
      - kind: integration
        ref: "grep -c 'fuzztime=' = 2; both steps name their single package (./internal/udprepl/, ./internal/app/persist/)"
        status: pass
    human_judgment: false
  - id: D7
    description: "PR CI tightened without a matrix change: named non-skippable stand step plus the TEST-05 policy header; 4-cell matrix and check-run names untouched"
    requirement: "TEST-16"
    verification:
      - kind: integration
        ref: "grep -c 'go test -race ./internal/stand/' .github/workflows/ci.yml = 1; git diff shows no added 'matrix:' line; Ruby YAML parse enumerates the same four cells"
        status: pass
    human_judgment: false
  - id: D8
    description: "TEST-11's ×3-green bar has its scheduled vehicle: make race-soak runs nightly with a 120-minute budget and the bar's comment"
    requirement: "TEST-11"
    verification:
      - kind: integration
        ref: "scheduled.yml race-flake job: timeout-minutes 120, run 'make race-soak' (COUNT defaults to 20); local sanity 'make race-soak COUNT=1' was plan 02-10's evidence; the full ×20 run belongs to the scheduled job"
        status: pass
    human_judgment: true

duration: 12min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 11: Nightly Scheduled Machinery Summary

**The heavy machinery now runs nightly: `scheduled.yml` carries the six-job D-09 set with a validated 10 000-seed dispatch path, the red and green halves of the review's section-15 scenarios execute at the documented 10 000 × 1 KiB scale through one `SUPKV_STAND_NIGHTLY` contract, deep fuzzing is bounded by `-fuzztime`, and PR CI gained a named non-skippable stand step with its matrix untouched — verified locally, where the nightly F9 run recovered 10 000 keys in 55.6 s inside its 85 s derived bound.**

## Performance

- **Duration:** 12 min (694 s)
- **Started:** 2026-11-21T13:53:24Z
- **Completed:** 2026-11-21T14:04:58Z
- **Tasks:** 3 (Task 1 code/config, Task 2 config, Task 3 structural audit — no separate commit)
- **Files created/modified:** 3 (1 new workflow, 2 modified)

## Accomplishments

- **Nightly scale selection wired (TEST-06, D-09):** `TestStand_emptyDiskRecovery` reads the existing `standNightly` hook and, when `SUPKV_STAND_NIGHTLY=1`, runs the review's F9 documented shape — 10 000 keys × 1 KiB with the reviewer's default flags (hello 1 s, sync 30 s, `sync_response_rate` 500; the 1 s startup gate is the harness constant). The recovery bound stays the derived-from-flags expression (`N × floor(1s/rate)` + gate/hello/sync/slack), scaling with N; the default (env unset) path is the same shrunken-timer 20-key scenario as before, verified green unchanged.
- **The nightly run is real, not nominal:** a local `SUPKV_STAND_NIGHTLY=1` run seeded 10 000 × 1 KiB records through the review's preload profile (64 workers / 8 connections), recovered all 10 000 keys in **55.6 s inside the 85 s bound** (20 s paced + 65 s margin), passed a full 1 KiB value pass over every recovered record, and converged the recovered node's own write back to the seeded peer. No bound was weakened and no red evidence was produced (the honesty note's breach condition did not fire).
- **`scheduled.yml` (TEST-16, TEST-08, TEST-11):** nightly cron `30 2 * * *` plus `workflow_dispatch` with a `seeds` input (default "1000", documented for the one-off 10 000-seed TEST-01 proof). Six jobs, each ubuntu-latest with an explicit timeout: `chaos-seeds` (validated `SUPKV_SIM_FRESH_SEEDS` → `go test -race ./internal/app/repl/ -run '^TestChaos' -shuffle=on -count=3`), `bench-smoke` (`make bench-smoke` = `-bench . -benchtime=1x`), `red-run` (`SUPKV_STAND_NIGHTLY=1`, plain `bash scripts/red-runner` — assert-red and the D-02 mutant stage in full), `race-flake` (`make race-soak`, 120 min), `nightly-stand` (`SUPKV_STAND_NIGHTLY=1`, `go test -race -count=1 ./internal/stand/`), `deep-fuzz` (`-fuzztime=10m` on FuzzDecode then FuzzDumpDecode, one target per package).
- **Documented-scale mapping across the two stand jobs:** `nightly-stand` runs the green half (F9 at 10 000 × 1 KiB via Task 1's selection plus scale-independent F2c, red tests skipped because `SUPKV_RED_TESTS` is absent); `red-run` runs the red half (R-01's 20 000-key paced listing and the R-02/R-03 documented-scale variants via plan 02-05's selection, with the runner exporting `SUPKV_RED_TESTS` itself). Together the review's section-15 set executes at documented scale every night.
- **Q1 parameterization and input gate (T-2-21):** the dispatch input is validated in a dedicated first step (positive-integer regex, digits only); any other value produces a warning and falls back to 1000 via `$GITHUB_ENV`. The value is consumed only as a test-loop count.
- **PR CI tightened, matrix frozen (TEST-05 policy):** the ci.yml header documents that internal/stand socket tests are non-skippable on Linux (a probe miss is a fatal), that the darwin cell reports via the named `TestPlatform_darwinReplicationUntested` result, and that the cross cell builds only. The non-cross gate step now runs `go test -race ./internal/stand/` before the repo-wide run — a named, greppable check; the 4-cell matrix and all five check-run names are unchanged (verified by a diff grep for added `matrix:` lines).

## Task Commits

Each task was committed atomically:

1. **Task 1: nightly documented-scale stand run — scale wiring + scheduled.yml** — `7f0d2d5` (feat)
2. **Task 2: tighten ci.yml — Linux socket tests non-skippable** — `5263188` (ci)
3. **Task 3: structural verification of the scheduled machinery** — no code; its audit is this summary (D-09 contract table, dispatch command)

**Plan metadata:** this docs commit (SUMMARY/STATE/ROADMAP/REQUIREMENTS)

## Files Created/Modified

- `.github/workflows/scheduled.yml` — new six-job nightly workflow (cron + dispatch, validated seeds, per-job timeouts, pinned actions, contents: read)
- `internal/stand/stand_test.go` — nightly scale constants, `emptyDiskValue` (1 KiB deterministic values), `seedEmptyDiskKeys` (review preload profile), `emptyDiskRecovered` (throttled prefix-List presence poll), and the scale-derived recovery/write convergence bounds
- `.github/workflows/ci.yml` — TEST-05 policy header plus the named non-skippable stand step

## D-09 Contract Audit (Task 3)

| Contract line | Evidence in `scheduled.yml` | Verdict |
|---|---|---|
| TEST-16 tokens verbatim | `-shuffle=on`, `-count=3` in chaos-seeds; `-bench . -benchtime=1x` documented on `make bench-smoke`; nightly cron present | PASS |
| Job set complete | chaos-seeds, bench-smoke, red-run, race-flake, nightly-stand, deep-fuzz (six) | PASS |
| red-run: no skip flags, assert-red full | exactly `bash scripts/red-runner`; no `--skip` anywhere; `SUPKV_STAND_NIGHTLY: "1"` | PASS |
| nightly-stand: green scale only | `SUPKV_STAND_NIGHTLY: "1"` and no `SUPKV_RED_TESTS` — red tests skip by construction | PASS |
| Documented-scale mapping (TEST-06) | nightly-stand: F9 10 000 × 1 KiB (Task 1 selection) + F2c; red-run: R-01 20k, R-02 10k drain/burst, R-03 F14 via plan 02-05's selection | PASS |
| Q1 parameterization | `workflow_dispatch.inputs.seeds` → validated → `SUPKV_SIM_FRESH_SEEDS`; fallback 1000 | PASS |
| PR-CI contract | lint + 4-cell matrix unchanged; named stand step added; matrix diff empty; macOS cell reports the limitation via the named test | PASS |
| Environment honesty | `permissions: contents: read`; no secrets; no new actions; every job has `timeout-minutes`; both fuzz steps have `-fuzztime` | PASS |
| No workflow executed from planning | this plan only wrote files; GitHub Actions execution is post-merge verification | PASS |

**One-off 10 000-seed TEST-01 proof (manual dispatch):**

```bash
gh workflow run scheduled.yml -f seeds=10000
gh run watch            # the chaos-seeds job budget is 4 h (RESEARCH A4: 1-3 h)
```

The nightly cron cannot be triggered locally; the first real scheduled evidence lands after merge.

## Decisions Made

- **Documented defaults, not shrunken timers, at nightly:** the F9 scenario runs with the reviewer's exact stand flags so the nightly assertion measures the configuration the evidence row describes.
- **Bounds derive from the selected flags:** the margin swaps from the shrunken-timer constants to `gate + 2×hello + 2×sync + slack` when nightly, and the post-recovery write wait does the same — the 6 s shrunken expression was observed timing out at the 30 s documented cadence, so it was replaced rather than left as a hidden nightly failure.
- **Presence by List, integrity by full value pass:** 10 000 individual Gets per poll would drown the recovering node; the throttled prefix List is the progress probe, and every 1 KiB value is then verified once.
- **No artifact-upload action:** `release.yml` uses none, and the plan forbids new action introductions. The red-evidence summary is printed into the workflow log; a deep-fuzz crasher is printed by `go test` into the job log and committed by a human under plan 02-17's prohibition.
- **Timeouts are part of the contract:** every job is bounded (240/20/60/120/60/40 min), and both fuzz steps carry `-fuzztime` — a wedged job fails instead of holding a runner.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Nightly post-recovery write wait used the shrunken-timer bound**
- **Found during:** Task 1 verification (local `SUPKV_STAND_NIGHTLY=1` run)
- **Issue:** after the 10 000-key recovery passed, the "recovered node's write visible on node A" wait still used `standConvergeDeadline` (6 s, derived from the shrunken 100 ms/200 ms timers). Under the documented 30 s sync cadence the write announce round needs up to ~30 s, so the test failed at 6 s.
- **Fix:** derive the wait from the selected flags — nightly `2×sync + 2×hello + 3 s` (65 s) — while leaving the default expression untouched. Re-ran the full nightly scenario: PASS (10 000 keys in 55.6 s, value pass green, write convergence green).
- **Files modified:** `internal/stand/stand_test.go`
- **Verification:** `SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestStand_emptyDiskRecovery$'` → PASS (139.97 s)
- **Commit:** `7f0d2d5`

**Total deviations:** 1 (Rule 1 auto-fixed; no architectural change, no scope change)

**Impact:** none on the plan contract — the plan's "bound stays derived from flags, scales with N" requirement is strengthened (the write wait now follows the same discipline); the default CI path is byte-identical.

## Issues Encountered

- **YAML plain-scalar colon trap:** three step names containing `TEST-16: ...`-style text were invalid YAML plain scalars (a colon followed by a space). Ruby Psych caught this while PyYAML is absent on this host; the names were quoted and the file re-parsed. The plan's YAML verify step honestly skips on a PyYAML-less host, so Ruby was used as the independent parse check (it enumerated all six jobs).
- **PyYAML absent:** `python3 -I -c "import yaml"` fails on this host; the plan's verify exits 0 (honest skip). The structural greps plus the Ruby parse are the primary check here, and GitHub's own parser will be the final validator post-merge.
- **Local nightly runtime:** the documented-scale scenario takes ~140 s under `-race` locally (seeding + 55.6 s recovery + 10 000 value Gets), which fits comfortably inside the job's 60-minute budget and the CI-runner slowdown headroom.

## User Setup Required

None - no external service, secret or environment configuration. The workflow uses `GITHUB_TOKEN`-less least privilege (`contents: read`) and runs only repo-owned scripts.

## Known Stubs

None. The workflow invokes real targets/scripts (`make bench-smoke`, `make race-soak`, `bash scripts/red-runner`, `go test -fuzz`), and the nightly scale selection drives the real stand scenario; no placeholder step or mocked command exists.

## Threat Flags

None beyond the plan's registered boundary. The workflow is exactly the T-2-20…T-2-23 surface the plan threat-modelled, and the mitigations are implemented: explicit `timeout-minutes` on every job plus `-fuzztime` on both fuzz steps (T-2-20), positive-integer validation with fallback for the dispatch input (T-2-21), `permissions: contents: read` with no secrets (T-2-22), and the plain no-flag red-runner invocation (T-2-23). No install steps exist (T-2-SC).

## Next Phase Readiness

- **Phases 3-5 (the flips):** the red-run job now asserts the expected-red floors nightly at documented scale; each fix flips only its `scripts/red-expectations.tsv` row, and the mutation stage keeps the chaos test honest.
- **TEST-06 stays open in REQUIREMENTS.md until this plan's metadata commit** — this is the last declaring plan (with 02-03 and 02-15); `requirements.ready-ids` decides, and `mark-complete` fires in this plan's close-out.
- **TEST-08/TEST-11/TEST-16 are completed by this plan** (with 02-07/02-17, 02-10 and this plan respectively): negative table + fuzz corpora + scheduled deep runs; race-soak target + nightly ×3-green vehicle; scheduled workflow with the acceptance tokens.
- **Post-merge verification:** the nightly cron and the `gh workflow run scheduled.yml -f seeds=10000` proof execute on GitHub only — both are documented here and are the Phase-2 verification's external evidence.

## Self-Check

- [x] `.github/workflows/scheduled.yml` exists and parses (Ruby Psych; PyYAML honestly absent) with six jobs
- [x] `internal/stand/stand_test.go` exists; `TestStand_emptyDiskRecovery` references `standNightly` 3 times
- [x] `.github/workflows/ci.yml` exists; `go test -race ./internal/stand/` appears exactly once; no added `matrix:` line
- [x] Task commits `7f0d2d5` and `5263188` are ancestors of HEAD; no deletions in either
- [x] Task 1 verify: `timeout-minutes` 6, `fuzztime=` 2, `workflow_dispatch:` 1, `SUPKV_STAND_NIGHTLY` 2
- [x] Task 2 verify: stand step count 1; `git diff` matrix check passes
- [x] Task 3 verify: token count 14 (≥ 6); no skip flags; no secrets; cron present
- [x] Default-scale stand test green (`go test -race -count=1 ./internal/stand/ -run '^TestStand_emptyDiskRecovery$'`)
- [x] Nightly-scale stand test green locally (`SUPKV_STAND_NIGHTLY=1`, 10 000 keys in 55.6 s within 85 s bound)
- [x] `make lint` and `make test` pass repo-wide
- [x] `check evaluation-scope --plan 02-11`: 2 commits, 3 changed files, `missingOnDisk: []`, `outsideUnion: []`, `unreachable: []`

**Self-Check: PASSED**

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
