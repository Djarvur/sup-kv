---
phase: 08-performance
plan: 11
subsystem: performance
tags: [phase-gate, frozen-red-evidence, mutation-testing, lint, buf-breaking, darwin-cross, client-example, make-perf, measurement-ledger, perf-01, perf-02, perf-03, perf-04, perf-05, perf-06]
requires:
  - phase: 08-performance
    provides: plans 08-01…08-10 — the threshold table, deterministic store-call and pending-visit gates, port allocator, version-only replay seam, harness diagnostics and the published measurement ledger this gate verifies
  - phase: 07-observability-and-configuration
    provides: the pinned-hash phase-gate shape this plan repeats, and the stabilized stand harness `make perf` builds on
provides:
  - a recorded Phase-8 gate result on the committed tree — three frozen content hashes, both mutation sed targets, red-runner 7/7 with both mutants killed, make lint/test/buf-lint/buf-breaking/client-example green, darwin build/vet green, the vendored lint body and direct runtime require block unchanged, the CI workflows untouched, and a recorded make perf run with its hardware block, load and per-row results
affects: [phase-completion, 09-documentation-and-release, PERF-01, PERF-02, PERF-03, PERF-04, PERF-05, PERF-06]
actuals:
  tokens: 5172
  tasks: 2
  commits: 2
plan_head_before: d64d189c08d0c08dc6731f4a8dc613c6d123f289
plan_head_after: a27d62b300bc5e6704b766becececbfde1b913e2
tech-stack:
  added: []
  patterns:
    - "Phase gate: content hashes (never refs) for frozen evidence, literal shape greps for the mutation seds, then the unmodified standing gates once on the committed tree"
    - "Record a perf run's whole hardware block, the host load at start and the per-row measured figures, never a bare number"
key-files:
  created:
    - .planning/phases/08-performance/08-11-SUMMARY.md
  modified: []
key-decisions:
  - "This gate plan makes no task commit: `files_modified: []`, no source changed and every gate passed on the committed tree, so committing a no-op would fabricate history (the 01-04/07-11 precedent); the record lives in this SUMMARY."
  - "The plan's explicit deterministic-gate package list omitted `./internal/stand/`, where `TestPerfBurstBoundsMatchRedContract` lives (`internal/stand/perf_test.go:247`); the verbatim command was run as written and the drift guard was then run explicitly in its own package (PASS 0.00 s). That is strictly more coverage, never a narrowed run, and `make test` exercised it as well."
  - "The measured commit range contains two external planning-only commits (`c0a5266`, `a27d62b`) authored by a concurrent agent while the gate ran; the source tree stays byte-identical to the dispatched HEAD `d64d189` (`git diff --stat -- . ':(exclude).planning'` is empty), so the gate results stand for the committed source tree and this plan's own commits are zero."
  - "`make buf-breaking` ran the real FILE-policy comparison: `origin/master` (`d6b2a75`) carries both protos, so the D-08 deferral guard took the comparison branch, which passed with no output."
  - "The `make perf` run was the first and only attempt: the host load was 2.31 at start and rose to 4.53 during the late rows, and all seven rows passed — no rerun and no relaxed bound."
patterns-established:
  - "A phase gate records the load at its own start beside the hardware block, so a later reader can tell a calm-host run from a lucky one."
requirements-completed: [PERF-01, PERF-02, PERF-03, PERF-04, PERF-05, PERF-06]
coverage:
  - id: D1
    description: "The three frozen red-evidence paths hash to their planning-time literals"
    requirement: "PERF-01…PERF-06"
    verification:
      - kind: contract
        ref: "git hash-object (no ref): red_gate=519789bd1f7abb2af09053b81c26e0ce45d9bdaf, stand_red=573fed68d2f1a039b8ac023a342446103064273b, tsv=add45b810d36d69fd574ca83cb575dab5722a55b"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both mutation sed targets still exist and the full red-runner is 7/7 with 0 errors and both mutants killed"
    requirement: "PERF-01…PERF-06"
    verification:
      - kind: contract
        ref: "grep -F targets present: internal/app/repl/core.go:2535 'if !c.nextSync.IsZero() && now.Before(c.nextSync) {'; internal/app/storage/store.go:253,274 'out[key] = rec.Version'"
        status: pass
      - kind: integration
        ref: "bash scripts/red-runner (no flags) -> exit 0; 'red-runner: 7 expectation row(s) checked, 0 error(s)'; control 4/4 pass; sync-without-tombstones and periodic-sync-disabled both killed on seed 848612892191557850; 'mutation stage green'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The standing gates are green on the final tree, and the darwin cross build and vet pass"
    requirement: "PERF-01…PERF-06"
    verification:
      - kind: contract
        ref: "make lint -> config verify + golangci-lint run ./... exit 0; make test -> go test -race -cover ./... exit 0 (15 packages ok, no FAIL; internal/app/repl 49.658 s uncached, the rest cache hits from the red-runner build); make buf-lint exit 0; make buf-breaking exit 0 (real comparison vs refs/remotes/origin/master d6b2a75); make client-example exit 0"
        status: pass
      - kind: contract
        ref: "GOOS=darwin go build ./... && GOOS=darwin go vet ./... exit 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "The vendored lint body, the direct runtime require set and the CI workflows are unchanged"
    requirement: PERF-06
    verification:
      - kind: contract
        ref: "sed -n '1,64p' .golangci.yml | git hash-object --stdin = a121e0466bc211e48ca898f6c07434dc50eba07c; awk first require block | git hash-object --stdin = 9679f1431d1815116871a4f7af79e727283a2875; git diff --exit-code .github/workflows/ clean; Makefile bench-smoke target unchanged (no source diff anywhere)"
        status: pass
    human_judgment: false
  - id: D5
    description: "`make perf` runs on this host and its hardware block, load and every row result are recorded"
    requirement: PERF-06
    verification:
      - kind: integration
        ref: "make perf -> exit 0; 7 PASS rows (>=5), no FAIL TestPerf/TestStand_; hardware block present; load at start 2.31 2.46 3.32 on 16 logical CPUs; burst 1000.1 writes/s (min 990) drain 129.0 ms (bound 5 s); default profile p50 724.4 ms p99 1.484 s at 1000.1 writes/s; amplification 14.36x with bytes-per-dump 14,979,072 <= 16,475,180; three memory points; fast profile mean 2.795 ms (bound 25 ms) p99 5.782 ms (bound 100 ms); absent/live 2381/2289 ns/op ratio 1.04 (bound 2.0); final threshold-free stand suite ok 7.416 s"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every deterministic gate named in the plan and the 100-run dumpNotify scenario pass"
    requirement: "PERF-01…PERF-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ ./internal/grpcapi/ ./internal/udprepl/ ./internal/testutil/ -> ok 2.074/1.045/2.734/1.023 s; names: TestCore_keyCountCostFlat, TestCore_unackedCostFlat, TestCore_notifyWriteCostFlat, TestCore_applyRemoteCostFlat, TestCore_pendingIndexInvariant/Differential/ProductionPaths, TestChunkSyncEntries_capacityFollowsEntries, TestReplaySeamCarriesVersionOnlyRecord, TestNotifyReplayedWriteCarriesVersionOnly/AllocationBound, TestBandsPartitionThePrivateWindow, TestUDPPortPickIsInBandAndDistinct/IsImmediatelyBindable/BandExhaustionNeverReturnsZero"
        status: pass
      - kind: unit
        ref: "go test -race -count=100 -run '^TestScenario_dumpNotify$' ./internal/app/repl/ -> ok 1.370 s (100 runs)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 -v -run '^TestPerfBurstBoundsMatchRedContract$' ./internal/stand/ -> PASS 0.00 s (the drift guard lives in internal/stand, absent from the plan's package list)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The doc anchors are green and every TestPerf name the ledger cites exists in the tree"
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestOperationsDoc|TestPerformanceDoc|TestAPIDoc|TestExamplesDoc|TestReadmeQuickstart' -> ok 7.822 s"
        status: pass
      - kind: contract
        ref: "for each of TestPerfAbsentPeerWriteCost, TestPerfAmplification, TestPerfBurstDistinctKeys, TestPerfDefaultProfileLatency, TestPerfFastProfileLatency, TestPerfMemoryPoints: go test -list '^<name>$' over ./internal/stand/ ./internal/app/repl/ ./internal/grpcapi/ | grep -qx -> all six listed"
        status: pass
    human_judgment: false
duration: 20min
completed: 2026-11-24
status: complete
---

# Phase 08 Plan 11: Phase Gate Summary

The Phase-8 gate is green on the committed tree: the frozen red evidence still hashes to its planning-time bytes and both mutation seds still bite (7/7 rows, both mutants killed), every standing gate plus the darwin cross cell is green with the vendored lint body, the direct runtime requires and the CI workflows unchanged, and the recorded `make perf` run passed all seven rows at load 2.3-4.5 with its hardware block captured.

## Performance

- **Duration:** 20min (1187 s)
- **Started:** 2026-11-24T21:37:41Z
- **Completed:** 2026-11-24T21:57:28Z
- **Tasks:** 2
- **Files modified:** 0 source files (this gate plan changes no source; only `.planning/` artifacts)

## Accomplishments

- **Frozen content pins match.** `git hash-object` (no ref, no merge-base) returns `red_gate=519789bd1f7abb2af09053b81c26e0ce45d9bdaf`, `stand_red=573fed68d2f1a039b8ac023a342446103064273b` and `tsv=add45b810d36d69fd574ca83cb575dab5722a55b` — byte-identical to the planning-time literals. No frozen test and no expectation row was edited.
- **Both mutation sed targets still exist, un-re-anchored.** The periodic-sync gate line is at `internal/app/repl/core.go:2535` and the tombstone write at `internal/app/storage/store.go:253,274`; `scripts/red-runner` has no diff, so the Phase-8 core edits (pending index, rewritten loops, counters) preserved both shapes.
- **Full red-runner 7/7 with mutants.** `bash scripts/red-runner` (no flags, no `--skip-mutants`) exits 0 twice (the substantive run and the verbatim verify): every one of the seven green rows reports pass, the control arm passes all four corpus seeds, and both mutants are killed on seed 848612892191557850 (`sync-without-tombstones`, `periodic-sync-disabled`), ending with `mutation stage green`. Evidence: `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261124T213957Z/` and `20261124T214214Z/` (untracked, as designed).
- **Standing gates green.** `make lint` (config verify + `go tool golangci-lint run ./...`) exit 0; `make test` (`go test -race -cover ./...`) exit 0 with 15 packages ok and no FAIL (`internal/app/repl` 49.658 s uncached, the rest cache hits from the immediately preceding red-runner build, which keys on the same unchanged source); `make buf-lint` exit 0; `make buf-breaking` exit 0 — it ran the **real** FILE-policy comparison because `refs/remotes/origin/master` (`d6b2a75`) carries both protos, so the D-08 deferral guard took the comparison branch; `make client-example` exit 0 (build, vet, `go mod tidy`, `git diff --exit-code` on the nested module).
- **Darwin cross gate green.** `GOOS=darwin go build ./...` and `GOOS=darwin go vet ./...` both succeeded.
- **Frozen configuration pins hold.** `.golangci.yml` lines 1-64 hash to `a121e0466bc211e48ca898f6c07434dc50eba07c`; the first `require ( … )` block of `go.mod` hashes to `9679f1431d1815116871a4f7af79e727283a2875` (still exactly grpc v1.85.0 and protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e) — no direct runtime dependency was added in Phase 8. `git diff --exit-code .github/workflows/` is clean, and `make bench-smoke` remains the threshold-free smoke target (unchanged `Makefile`).
- **Every deterministic gate ran and passed.** The explicit list is green: the four cost-flat gates, the three pending-index tests, the chunker capacity test, the replay version-only seam, both replayed-write tests, the port band/allocator tests, and the 100-run `TestScenario_dumpNotify` scenario (all names in the coverage table). The burst-constant drift guard passed too — see Deviation 1 for where it lives.
- **Doc anchors and doc/test linkage green.** `TestOperationsDoc`, `TestOperationsDocFamilies`, `TestPerformanceDoc`, `TestPerformanceDocFamilies`, `TestAPIDoc`, `TestExamplesDoc` and `TestReadmeQuickstart` pass (7.822 s). Every `TestPerf*` name `docs/PERFORMANCE.md` cites — `TestPerfAbsentPeerWriteCost`, `TestPerfAmplification`, `TestPerfBurstDistinctKeys`, `TestPerfDefaultProfileLatency`, `TestPerfFastProfileLatency`, `TestPerfMemoryPoints` — is listed by `go test -list` in the tree, so no cited test is missing.
- **The measurement run is recorded with its block.** `make perf` exits 0 with 7 passing rows and no failing `TestPerf`/`TestStand_` line:

  | Row | Result |
  |-----|--------|
  | `TestPerfBurstDistinctKeys` | PASS 15.70 s — achieved 1000.1 writes/s (min 990), drain 129.040 ms (bound 5 s), 10 000 keys x 1 KiB, ext4 |
  | `TestPerfDefaultProfileLatency` | PASS 39.28 s — p50 724.360 ms, p99 1.484 s, rate 1000.1 writes/s, 100 hot keys, state 10 000 keys |
  | `TestPerfAmplification` | PASS 37.73 s — amplification 14.36x, bytes-per-dump 14,979,072 (bound 16,475,180), 15 000 writes, logical 15,645,000 B |
  | `TestPerfMemoryPoints` | PASS 31.81 s — three points (1 000/10 000 x 1 KiB, 1 000 x 60 KiB), logical/dump/RSS ratios logged |
  | `TestPerfBurstBoundsMatchRedContract` | PASS 0.00 s — perf threshold and frozen red contract name identical bounds |
  | `TestPerfAbsentPeerWriteCost` | PASS 2.76 s — absent 2381.0 ns/op vs live 2289.0 ns/op, ratio 1.04 (bound 2.0), 10 000 unacked |
  | `TestPerfFastProfileLatency` | PASS 2.45 s — mean 2.795 ms (bound 25 ms), p99 5.782 ms (bound 100 ms), 784 samples |

  Hardware block (the first row's, identical on every row): `CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=2.45 2.49 3.33; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16`. Host load was `2.31 2.46 3.32` at the gate's start (`/tmp/sup-kv-perf-load-08.txt`), rising to `4.53 3.31 3.53` on the last row; the final threshold-free stand suite passed in 7.416 s.
- **The production tree is untouched by this plan.** `git diff --stat -- . ':(exclude).planning'` is empty and every guarded path (`.github`, `Makefile`, `go.mod`, `go.sum`, `.golangci.yml`, `scripts`, `internal`, `api`, `cmd`, `docs`, `examples`) is clean; the only new paths are the runner evidence directories and the gitignored `.tmp/perf`.

## Task Commits

This gate plan produced **no task commits**: `files_modified: []`, no source edit, and every gate passed on the committed tree. Committing a no-op would fabricate history (the Phase-01 plan 01-04 and 07-11 precedent); the green gate record is this SUMMARY.

- Plan head before: `d64d189c08d0c08dc6731f4a8dc613c6d123f289` (the dispatched HEAD)
- `git rev-list --count d64d189..HEAD` at SUMMARY time: **2** — both are external planning-only commits (`c0a5266 docs(phase-01): add security threat verification`, `a27d62b docs(01): record code review disposition`) authored by a concurrent agent while this gate ran; neither touches source and neither is this plan's work (see Issues).
- This plan's own source commits: **0**. Plan head at SUMMARY time: `a27d62b300bc5e6704b766becececbfde1b913e2`.
- The single metadata commit (`docs(08-11): complete phase-gate plan`) follows this file and carries SUMMARY.md, STATE.md, ROADMAP.md and REQUIREMENTS.md only.

## Files Created/Modified

- `.planning/phases/08-performance/08-11-SUMMARY.md` — this gate record.

No source, test, proto, lint-config, Makefile, workflow or `go.mod`/`go.sum` file was modified.

## Decisions Made

- Zero task commits, because a gate plan that changes no file and passes every gate has nothing to commit; the evidence and timings live in this SUMMARY (and the untracked runner evidence).
- The recorded `make perf` run is the first and only attempt. The plan allows one rerun for a loaded-host miss; there was no miss, so no rerun and no threshold was touched.
- `buf-breaking` was not deferred: both protos exist on `origin/master`, so the guard executed the real wire comparison — a stronger result than the deferral the wording anticipated, and it was left to run as written.
- The doc's measured figures were deliberately **not** compared with this run's figures: two legitimate runs differ, which is why the published table names its own commit and block. Only the doc/test name linkage and the compiled-threshold drift guard were gated, exactly as the plan specifies.

## Deviations from Plan

**1. [Rule 3 - Plan-text gap] Ran the burst drift guard in its own package, in addition to the verbatim gate command**
- **Found during:** Task 2, the explicit deterministic-gate list
- **Issue:** The plan's command names `TestPerfBurstBoundsMatchRedContract` but its package list is `./internal/app/repl/ ./internal/grpcapi/ ./internal/udprepl/ ./internal/testutil/`; the test actually lives at `internal/stand/perf_test.go:247`, so the command as written cannot exercise the name it lists. `make test` does run it (the `must_haves` truth says so), but the gate's own list could not name its result.
- **Fix:** Ran the verify block verbatim (green), then ran `go test -race -count=1 -v -run '^TestPerfBurstBoundsMatchRedContract$' ./internal/stand/` — PASS 0.00 s. No source, test, threshold or command was renamed; this adds coverage rather than narrowing any run.
- **Files modified:** none
- **Verification:** `/tmp` gate logs; the result is recorded in coverage D6.
- **Commit:** none (gate plan)

No plan prohibition was crossed: no frozen file, expectation row or pinned literal was edited, no `//nolint` was added, no perf threshold was relaxed, no test run was narrowed, `-short` was never passed, and the runner ran without `--skip-mutants`. The evidence directory stays untracked.

## Issues Encountered

- **Concurrent external work in the same working tree.** Two planning-only commits landed mid-gate (`c0a5266` at 00:53:48+03:00 and `a27d62b` at 00:55:23+03:00 local) and `.planning/config.json` was modified externally (workflow `_auto_chain_active` true→false). None of these were staged, modified or reverted by this plan; they change no source, so every gate result stands for the byte-identical committed source tree. The commit-range measurement in "Task Commits" reflects them honestly rather than narrating a zero.
- **`make test` cache hits.** Fourteen of the fifteen packages reported cached results (the immediately preceding red-runner built the same unchanged source); `internal/app/repl` re-ran uncached in 49.658 s. No source changed between the two runs, which is what makes the cache valid; nothing was narrowed.
- **No load-driven failure occurred.** Unlike the Phase-7 gate, the host was calm (load 2.31 at start, 4.53 at the end) and the two wall-clock ratio gates passed inside the first `make test`; the Phase-8 deterministic cost gates were green with no retry.

## User Setup Required

None - no external service configuration required.

## Threat Flags

None — this plan adds no source, endpoint, auth path or schema change; it only records results of existing gates.

## Next Phase Readiness

Phase 8 is complete: all eleven plans have summaries, the phase gate is green on the committed tree, and PERF-01…PERF-06 are ready for requirement close-out. The published ledger (`docs/PERFORMANCE.md`) carries its own measured commit (`9cdc0f1`) and hardware block; this gate's block is recorded above as the second, independent confirmation that the thresholds hold on the committed tree. Ready for phase completion (review/verification) and Phase 9 (documentation and release).

## Self-Check: PASSED

- Every gate was executed as written and its observed output is recorded above (hashes, greps, runner line, make targets, darwin, deterministic list, 100-run scenario, doc anchors, doc linkage, perf block and rows).
- No source file changed: `git diff --stat -- . ':(exclude).planning'` is empty; the tree carries only untracked red-evidence directories and the externally modified `.planning` files.
- `08-11-SUMMARY.md` exists at the expected path; the plan's own source commits are 0 before this file, as a no-source gate plan must be, and the two commits measured in the range are documented external planning-only commits.
