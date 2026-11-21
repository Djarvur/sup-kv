---
phase: 02-stand-and-simulator-as-tests
plan: 12
subsystem: testing
tags: [red-tests, p-01, value-limit, persistfile, stand, expectation-table, red-runner, red-evidence, d01, adr-0011]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: 02-01's D-01 mechanism (SUPKV_RED_TESTS gate, scripts/red-runner, the expectation table, append-only evidence) and 02-05's four stand red tests (R-01, R-02 drain/burst, R-03 F14) that the full recorded run consumes
  - phase: 04-replication (v1.0 shipped)
    provides: the defect-bearing codec bound (maxRecordBytes = 61440) and main.go's unchecked --value-max-bytes flag that P-01 pins
provides:
  - TestRed_P01_valueLimit — process-level P-01 with the reviewer's 63 000-byte case and the exactly-value_max_bytes (64 000) boundary subtest, both SIGKILL/restart at --value-max-bytes=64000
  - TestRed_P01_valueLimitEngine — engine-level P-01 on the real persist.Open + persistfile.FileStore composition, no fake dump port
  - The complete seven-row expectation table: R-01, R-02 drain, R-02 burst, R-03 gate-level, R-03 stand F14, P-01 process, P-01 engine
  - scripts/red-runner over ./internal/grpcapi/, ./internal/stand/ and ./internal/persistfile/, serialized with -p 1 so the wall-clock stand reds own the machine
  - The full recorded red run red-evidence/20261121T124106Z (7/7 rows asserted FAIL, 0 errors) plus the previously untracked 2026-11-19T19:32:27Z run preserved append-only
affects: [02-11 (the scheduled red-run job invokes the completed runner/table), phases 3-5 (each fix flips exactly one table row red -> green, no test edits)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 51472    # chars/4 over the realized diff; 4/5 of it is the two committed go-test.json evidence streams
  tasks: 2
  commits: 3       # MEASURED from the plan-start ledger at SUMMARY write (#3968): git rev-list --count 5402b6c..HEAD
  plan_head_before: 5402b6cbd1b45336ab6a4da13e917863a7eccd6a
  plan_head_after: bf2b6b4d92564e84821fe7170f0fdc3f5b6dd986

# Tech tracking
tech-stack:
  added: none — stdlib testing plus the existing persist/persistfile/storage packages; the red-runner stays bash + python3 stdlib json
  patterns:
    - env-gated expected-red engine test on the REAL composition path (persist.Open + FileStore + real codec), skipping without SUPKV_RED_TESTS=1
    - process-level red test that asserts the dump holds the record between SIGKILL and restart, separating writer persistence from loader loss
    - red-runner package serialization (-p 1) so wall-clock pacing assertions are not corrupted by sibling packages' race-instrumented load
    - append-only red evidence: one timestamped directory per run, never overwritten, committed as phase evidence (T-2-24)

key-files:
  created:
    - internal/persistfile/red_value_limit_test.go
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T124106Z/
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261119T193227Z/
  modified:
    - internal/stand/stand_red_test.go
    - scripts/red-expectations.tsv
    - scripts/red-runner

key-decisions:
  - "The engine-level P-01 test lives in package persistfile_test (external), not the plan's literal `package persistfile`: it composes only exported API (persist.Open, persistfile.New, Engine.Set/Get) and reuses open_seed_test.go's seedDumper/seedFixedNow/seedNowMs helpers, while the internal form would need an unjustified //nolint:testpackage — the same reasoning 02-15/02-16 recorded"
  - "The runner now passes -p 1: the stand red scenarios assert wall-clock pacing and throughput bounds, so a sibling package's race build and test load would corrupt the very measurement the red verdict reads"
  - "The process-level test asserts the dump still holds the record after Set returned and before the SIGKILL — Set blocks until the covering dump completes (PERS-04), so the loss is provably the loader's (decoder's 61 440-byte bound), not the writer's or the kill's"
  - "Both P-01 sizes run at --value-max-bytes=64000 (an accepted v1.0 flag value): 63 000 bytes is the reviewer's F15 case and 64 000 is the acceptance's exact-value_max_bytes boundary; both are discarded at decode today"
  - "The earlier 2026-11-19T19:32:27Z evidence run (captured during 02-01 verification, never committed) was preserved in a separate docs commit rather than deleted: append-only evidence is the T-2-24 contract"

patterns-established:
  - "An expected-red test at the storage-engine layer composes the real Open/FileStore codec path and asserts survival of a confirmed record — never a fake dump port that could mask the decode bound"
  - "A red verdict may be separated into writer evidence (dump contains the record) and loader evidence (reopen loses it) so the failure names the defect, not an infrastructure suspicion"
  - "Every red-runner package addition keeps the hardcoded list with its extend comment; the table stays package-free and the flip remains a one-row TSV change"

requirements-completed: [TEST-04]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "TestRed_P01_valueLimit — process-level P-01 (decision 14.11 / ADR 0011): a node with the accepted --value-max-bytes=64000 confirms a 63 000-byte value (reviewer's F15) and one of exactly 64 000 bytes (the acceptance boundary), takes SIGKILL, restarts over the same dump, and both values are gone today because the loader discards records over 61 440 bytes; the test FAILs under SUPKV_RED_TESTS=1 and SKIPs without it"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_P01_valueLimit$' → exit 1, both subtests fail with 'the N-byte value confirmed with --value-max-bytes=64000 is gone after restart: the dump holds the record but the loader discards it' (4.3 s)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestRed_P01' (no env var) → SKIP, exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestRed_P01_valueLimitEngine — engine-level P-01 on the real persist.Open + persistfile.FileStore + codec composition: a 63 000-byte value is accepted under storage.Config.ValueMaxBytes=64000, confirmed by Set, present in the dump, and discarded on reopen (Open discarded 1); FAILs under the env gate and SKIPs without it"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/persistfile/ -run '^TestRed_P01_valueLimitEngine$' → exit 1: 'the confirmed 63000-byte record is gone after reopening the dump (Open discarded 1 record(s)): key not found' (0.1 s)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestRed_P01' (no env var) → SKIP, exit 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "scripts/red-expectations.tsv carries exactly seven data rows — TestRed_R03_gateClosedUntilFullList, TestRed_R01_pacedSync20k, TestRed_R02_unackedDrain, TestRed_R02_burst, TestRed_R03_f14ReplacedNode, TestRed_P01_valueLimit, TestRed_P01_valueLimitEngine — all expected red; the flip-to-green contract stays one TSV row per fix"
    requirement: TEST-04
    verification:
      - kind: other
        ref: "grep -v '^#' scripts/red-expectations.tsv | wc -l → 7; every data row is '<name><TAB>red' with no trailing whitespace (cat -A checked)"
        status: pass
    human_judgment: false
  - id: D4
    description: "scripts/red-runner covers ./internal/grpcapi/, ./internal/stand/ and ./internal/persistfile/ and serializes packages with -p 1; one full run asserts all seven rows FAIL with zero errors and commits the raw go-test.json + parsed summary under red-evidence/20261121T124106Z"
    requirement: TEST-04
    verification:
      - kind: other
        ref: "bash scripts/red-runner → exit 0: '7 expectation row(s) checked, 0 error(s)'; summary.txt shows OK for all 7 rows (stand 360.1 s, persistfile 4.4 s, grpcapi 2.0 s)"
        status: pass
      - kind: other
        ref: "evidence failure detail: R-01 timed out at 20.6 s with 1 546/8 000 keys on B; R-02 drain never emptied within 35 s; burst achieved 27.8 writes/s with 11 487 DeadlineExceeded; R-03 answered NotFound where Unavailable is required; both P-01 sizes lost after restart"
        status: pass
    human_judgment: false
  - id: D5
    description: "Normal gates stay green and the phase boundary holds: no production non-test file changed; make lint and make test pass with the red tests skipping"
    requirement: TEST-04
    verification:
      - kind: other
        ref: "make lint → exit 0 (config verify + go tool golangci-lint run ./...)"
        status: pass
      - kind: other
        ref: "make test (go test -race -cover ./...) → exit 0; internal/stand ok 6.96 s coverage 82.1%, internal/persistfile ok 7.19 s coverage 89.7%"
        status: pass
      - kind: other
        ref: "git diff --name-only 5402b6c..HEAD -- internal/app internal/udprepl internal/grpcapi internal/persistfile | grep -v '_test\\.go$' → empty"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 12: P-01 Red Tests and the Complete Seven-Row Red Run Summary

**P-01's value limit is pinned at process and engine level — a 63 000-byte value and one of exactly value_max_bytes both vanish across a restart at --value-max-bytes=64000 — completing the seven-row expectation table whose single recorded full run shows all four Criticals failing against v1.0.**

## Performance

- **Duration:** 12 min (720 s)
- **Started:** 2026-11-21T12:31:16Z
- **Completed:** 2026-11-21T12:43:16Z
- **Tasks:** 2
- **Files modified:** 8 (2 test files, 2 scripts, 4 committed evidence files)

## Accomplishments

- `internal/stand/stand_red_test.go` gained `TestRed_P01_valueLimit` (process level, appended after R-03): a node run with the review's accepted `--value-max-bytes=64000` confirms a 63 000-byte value and a value of exactly 64 000 bytes, takes SIGKILL, and restarts over the same dump. Today both are gone — the dump still carries the record after `Set` returned (asserted explicitly), but the loader's 61 440-byte decode bound discards it. Both subtests fail under `SUPKV_RED_TESTS=1` in 4.3 s and skip without it.
- `internal/persistfile/red_value_limit_test.go` (new) holds `TestRed_P01_valueLimitEngine`: the engine-level core on the real composition path — `persist.Open` over `persistfile.New` with the real codec, no fake dump port. With `storage.Config.ValueMaxBytes=64000` the store accepts and confirms a 63 000-byte record; the dump provably contains it; reopening discards exactly one record and `Get` answers not-found. Red in 0.1 s.
- `scripts/red-expectations.tsv` now carries exactly seven data rows: the gate-level R-03 row plus R-01 paced sync, R-02 unacked drain, R-02 burst, R-03 stand F14, P-01 process and P-01 engine — every row expected red, so Phases 3-5 flip one line each with zero test edits.
- `scripts/red-runner` now runs `./internal/grpcapi/`, `./internal/stand/` and `./internal/persistfile/` and serializes packages with `-p 1`, so the stand's wall-clock pacing assertions are not corrupted by sibling packages' race-instrumented builds and tests.
- One full recorded run (`red-evidence/20261121T124106Z`) asserts 7/7 rows FAIL with 0 errors: R-01 stalled at 1 546/8 000 keys after its 20.6 s bound; R-02's unacked set never drained within 35 s; the burst achieved 27.8 writes/s with 11 487 `DeadlineExceeded`; R-03 answered `NotFound` where `Unavailable` is required; both P-01 sizes were lost. This is the phase's machine-reproducible verdict in a single evidence directory.
- The pre-existing untracked `red-evidence/20261119T193227Z` run (captured during 02-01's verification) was committed in its own docs commit rather than deleted, keeping the evidence ledger append-only.

## Task Commits

Each task was committed atomically:

1. **Task 1: P-01 value-limit red tests (process + engine level)** - `bcd31e3` (test)
2. **Task 2: complete the expectation table + runner package list + full red-run evidence** - `1a15f02` (test)
3. **Evidence preservation (earlier untracked run)** - `bf2b6b4` (docs)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit that follows this file (docs).

## Files Created/Modified

- `internal/stand/stand_red_test.go` — modified: added the P-01 constants, `TestRed_P01_valueLimit` (env-gated, two subtests) and `redAssertP01SurvivesRestart`
- `internal/persistfile/red_value_limit_test.go` — created: `TestRed_P01_valueLimitEngine` on the real Open/FileStore composition, with the dump-contains-record assertion between write and reopen
- `scripts/red-expectations.tsv` — modified: six appended red rows (seven data rows total)
- `scripts/red-runner` — modified: package list extended to stand + persistfile; `-p 1` added with its measurement rationale
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T124106Z/{go-test.json,summary.txt}` — created: the full seven-row red run
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261119T193227Z/{go-test.json,summary.txt}` — created (preserved from untracked): the earlier 02-01-era run

## Decisions Made

- **Engine-level test package:** `persistfile_test` (external), not the plan's literal `package persistfile`. The test needs only exported API and reuses `open_seed_test.go`'s helpers; the internal form would require an unjustified `//nolint:testpackage`, which AGENTS.md forbids. Same reasoning as 02-15/02-16.
- **Runner serialization:** `-p 1` is now part of the runner. The stand red tests assert wall-clock bounds, and the file's own contract says competing scenarios corrupt each other's measurements; running three packages in parallel under `-race` would do exactly that.
- **Writer-vs-loader separation:** the process test asserts the record is in the dump after `Set` returns and before the kill, so a red failure names the decoder bound rather than leaving open a "the write never reached disk" suspicion.
- **Evidence preservation:** the stray 2026-11-19T19:32:27Z evidence directory was committed (not deleted) in a separate docs commit, preserving the append-only red-run record without mixing it into either task commit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Engine test package clause adapted to the strict lint gate**
- **Found during:** Task 1 (P-01 red tests)
- **Issue:** the plan names `package persistfile` for `red_value_limit_test.go`, but the file composes only exported API; the internal form would trip `testpackage` (its `skip-regexp` covers only `*_internal_test.go`) and need a `//nolint:testpackage` with no honest justification.
- **Fix:** wrote `package persistfile_test`, matching the package's existing external tests and reusing their `seedDumper`/`seedFixedNow`/`seedNowMs` helpers. Artifact path, test name and coverage are unchanged.
- **Files modified:** `internal/persistfile/red_value_limit_test.go`
- **Verification:** `make lint` exit 0; both red/skip verifies pass.
- **Committed in:** `bcd31e3` (Task 1 commit)

**2. [Rule 2 - Missing Critical] Runner serializes packages so the wall-clock red verdicts are trustworthy**
- **Found during:** Task 2 (runner package-list extension)
- **Issue:** adding two more packages made `go test` run up to three race-instrumented test binaries concurrently; the stand red scenarios assert pacing/throughput against the wall clock, which the stand file's own contract forbids sharing.
- **Fix:** added `-p 1` to the runner's `go test` invocation with a comment naming the measurement-integrity reason.
- **Files modified:** `scripts/red-runner`
- **Verification:** the full run recorded all seven rows FAIL with 0 errors, stand package 360 s.
- **Committed in:** `1a15f02` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 blocking)
**Impact on plan:** both were required for the lint gate and for the integrity of the red verdict; no production code, no scope creep, and the flip contract is untouched.

## Issues Encountered

- A third evidence directory (`20261119T193227Z`) predated this plan and was untracked: a separate 02-01-era run with a distinct raw JSON stream but the same one-row summary. It was preserved in `bf2b6b4` rather than deleted, per the append-only evidence contract. The working tree is clean.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The D-01 set is complete: all four Criticals are pinned across the phase (R-01, R-02 drain + burst, R-03 gate + stand F14, P-01 process + engine), and the single recorded run demonstrates them red against v1.0.
- Phases 3-5 need only flip the corresponding `scripts/red-expectations.tsv` row from `red` to `green` when a fix lands; no test, runner or Makefile edit is expected.
- Plan 02-11's scheduled red-run job can invoke `make red-run` as-is; its documented-scale stand execution is selected by `SUPKV_STAND_NIGHTLY=1` (this plan's evidence run intentionally used the default local scales).

## Self-Check: PASSED

- Files verified present: `internal/persistfile/red_value_limit_test.go`, `internal/stand/stand_red_test.go`, `scripts/red-runner`, `scripts/red-expectations.tsv`, both evidence directories' `go-test.json` + `summary.txt`.
- Commits verified ancestors of HEAD: `bcd31e3`, `1a15f02`, `bf2b6b4`.
- Acceptance re-run on the committed tree: both P-01 tests FAIL under `SUPKV_RED_TESTS=1`, both SKIP without it; `bash scripts/red-runner` exits 0 with 7 expectation rows and 0 errors; `make lint` and `make test` exit 0.

---

*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
