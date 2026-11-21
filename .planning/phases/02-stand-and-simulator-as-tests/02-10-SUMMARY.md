---
phase: 02-stand-and-simulator-as-tests
plan: 10
subsystem: testing
tags: [benchmarks, p50, p99, percentile, race-soak, bench-smoke, flake-audit, makefile, test-10, test-11]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: the three real-path benchmarks (06-08) and the internal/stand harness move with its reserved port window (02-02)
provides:
  - Shared benchmark percentile reporter (p50-ms/p99-ms) wired into all three real-path benchmarks
  - "make bench-smoke: TEST-10's single -benchtime=1x CI smoke command"
  - "make race-soak: TEST-11's -race -count=20 command over the affected set, COUNT override for cheap local runs"
  - Per-class dispositions for the three T-11 flake classes after the harness move
  - Per-block doc-harness deadline for the README/EXAMPLES verbatim runner
affects: [phase-8-performance, plan-02-11-scheduled-workflow, verify-work, TEST-10, TEST-11]

# Actuals (#2632) — chars/4 over the realized diff; pairs with the plan's estimate.
actuals:
  tokens: 4389
  tasks: 2
  commits: 2
plan_head_before: 03bee19eccb19a4e5e3ea321ca475e6521cc05cd
plan_head_after: d78eb844e5154e1261d97346d77ec22484cdc2f7

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-op duration collection in the measured loop feeding one shared ReportLatencyPercentiles helper (p50-ms/p99-ms via b.ReportMetric)"
    - "Flake-class dispositions recorded as Makefile comments beside the soak command that runs them"
    - "Verbatim doc-block runner bounded by a per-block budget clamped inside t.Deadline() (the waitFor hard-deadline pattern for subprocesses)"

key-files:
  created:
    - internal/testutil/bench.go
    - internal/app/internal/testutil/bench.go
  modified:
    - internal/app/persist/bench_test.go
    - internal/udprepl/bench_test.go
    - internal/grpcapi/bench_test.go
    - cmd/sup-kv/examples_test.go
    - Makefile

key-decisions:
  - "The shared percentile implementation lives in the new module-wide internal/testutil package: Go's internal-package rule forbids internal/udprepl and internal/grpcapi from importing internal/app/internal/testutil (verified by probe), and all three benchmarks must share one implementation; the plan's path keeps a documented wrapper"
  - "Metric names are p50-ms/p99-ms — whitespace-free unit strings in the repo's ReportMetric convention (latency-ns/op, dump-bytes); n=1 reports p50 == p99 == the sample, so -benchtime=1x is the CI smoke"
  - "Per-op duration definitions stay honest to each benchmark: dump = Encode+Save iteration span; burst = the author-side write span after the pacing tick (the tick is workload shape, not measured work); confirmation = submission-to-acked report"
  - "Measured workloads are unchanged: no sleeps, rate limiters or adaptive load were added inside any measured section; only time.Now collection and post-loop reporting are new"
  - "Class 2 (SIGTERM before the handler is installed) is carried per D-09's bar as a product-side window: no cmd/sup-kv/main.go change this phase; TestShutdown_secondSignal remains the post-ownership pin"
  - "Class 3 was fixed test-side with a one-minute docBlockBudget clamped inside t.Deadline() in the shared verbatim runner, so a wedged README/EXAMPLES block fails its own run instead of the package timeout"

patterns-established:
  - "Benchmark honesty: every published distribution comes from the measured loop's own durations, and the workload path (real persistfile, real sockets, default params) is untouched"
  - "Flake elimination: structurally mitigated classes are re-verified after harness moves; product-side classes carry a named disposition and their existing pin; test-side classes get a deterministic deadline"
  - "A soak command's single definition lives in the Makefile, so the scheduled job and local runs execute the identical package set and flags"

requirements-completed: [TEST-10, TEST-11]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "All three real-path benchmarks report p50 and p99 in milliseconds from their own per-op durations, with no self-pacing and unchanged measured workloads"
    requirement: TEST-10
    verification:
      - kind: other
        ref: "go test -run '^$' -bench . -benchtime=1x ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/ (3 benchmark result lines, 6 p50-ms/p99-ms metric occurrences)"
        status: pass
    human_judgment: false
  - id: D2
    description: "make bench-smoke is TEST-10's CI smoke command — every benchmark once at -benchtime=1x with no thresholds"
    requirement: TEST-10
    verification:
      - kind: other
        ref: "make bench-smoke (exit 0; 3 benchmarks, 6 percentile occurrences)"
        status: pass
    human_judgment: false
  - id: D3
    description: "make race-soak is TEST-11's -race -count=20 command over cmd/sup-kv, udprepl and the affected packages, with COUNT as the local-pass override"
    requirement: TEST-11
    verification:
      - kind: other
        ref: "make race-soak COUNT=1 (six packages green under -race; the ×3-green-across-runs bar is plan 11's scheduled job)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Three T-11 flake classes audited after the harness move with dispositions: port pick-to-bind intact (structurally mitigated), SIGTERM-before-handler carried per D-09, README deadline fixed test-side"
    requirement: TEST-11
    verification:
      - kind: other
        ref: "internal/stand/stand.go scenarioPortLo/Hi+registry intact; Makefile race-soak disposition comments; cmd/sup-kv/examples_test.go docBlockBudget"
        status: pass
    human_judgment: false

# Metrics
duration: 4min
completed: 2026-11-21
status: complete
---

# Phase 02 Plan 10: Benchmarks Honest and Race-Soak Prepared Summary

**All three real-path benchmarks now publish p50/p99 latency from their own measured per-op durations with no self-pacing, and TEST-11's flake bar has its single command definition (`make race-soak`) plus dispositions for all three T-11 classes.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-11-21T11:41:38Z
- **Completed:** 2026-11-21T11:46:07Z
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `ReportLatencyPercentiles` (shared, stdlib-only) sorts the per-op durations and reports `p50-ms`/`p99-ms` through `b.ReportMetric`; `BenchmarkDump10kKeys`, `BenchmarkReplicationBurst` and `BenchmarkConfirmationLatency` all feed it from their own measured loops.
- The benchmark smoke is green at `-benchtime=1x`: three benchmark lines, six percentile metric occurrences, zero on the baseline commit. The single-sample case reports p50 == p99 == the sample by construction.
- `make bench-smoke` is TEST-10's CI command exactly once (plan 11's bench job can invoke it directly); `make race-soak` is TEST-11's `-race -count=20` command over `cmd/sup-kv`, `udprepl` and the affected packages, with `COUNT` as the documented local-pass override.
- The T-11 audit found the class-1 mitigation intact after the harness move (private port window `[28100, 32678)` + claim registry in `internal/stand/stand.go`), recorded class 2 as carried per D-09 (product-side SIGTERM window; `TestShutdown_secondSignal` remains the pin), and fixed class 3 test-side with a per-block doc deadline clamped inside `t.Deadline()`.
- `make lint` and `make test` (full `-race -cover ./...`) are green on the committed tree in addition to the plan's own verifies.

## Task Commits

Each task was committed atomically:

1. **Task 1: p50/p99 reporting on the three benchmarks** — `72a4f1c` (test)
2. **Task 2: flake-class audit + race-soak and bench-smoke targets** — `d78eb84` (test)

**Plan metadata:** docs commit created after this SUMMARY (Step: final metadata commit).

_No production code changed — both deliverables are test/benchmark infrastructure and CI commands._

## Measured Benchmark Output (`make bench-smoke`)

```
BenchmarkDump10kKeys-16         1  128589723 ns/op  5147550 bytes/op  5147550 dump-bytes  128.6 p50-ms  128.6 p99-ms
BenchmarkReplicationBurst-16    1   12164961 ns/op        0 max-unacked                0.02412 p50-ms  0.02412 p99-ms  82.24 writes/s
BenchmarkConfirmationLatency-16 1    7479325 ns/op  7470511 latency-ns/op  7470511 max-latency-ns  7.471 p50-ms  7.471 p99-ms
```

These are `-benchtime=1x` smoke values (one sample each), not performance claims; the scheduled bench job (plan 11) runs the same target with default benchtime for real distributions.

## Files Created/Modified

- `internal/testutil/bench.go` — module-wide shared percentile reporter (`ReportLatencyPercentiles`, nearest-rank p50/p99, empty/n=1 guards); home required because of Go's internal-package rule.
- `internal/app/internal/testutil/bench.go` — the plan's artifact path: a documented wrapper re-exporting the reporter for the `internal/app` call site (`internal/app/persist/bench_test.go`).
- `internal/app/persist/bench_test.go` — collects the real Encode+Save iteration spans; reports p50/p99 in addition to bytes/op and dump-bytes.
- `internal/udprepl/bench_test.go` — collects each burst write's author-side span (after the pacing tick); reports p50/p99 in addition to writes/s and max-unacked.
- `internal/grpcapi/bench_test.go` — collects each Set-with-wait submission-to-acked span; reports p50/p99 in addition to latency-ns/op and max-latency-ns.
- `cmd/sup-kv/examples_test.go` — `runBashVerbatim` now bounds each verbatim doc block by `docBlockBudget` (1 min) clamped 250ms inside `t.Deadline()`; the deadline failure names the block and carries its captured output.
- `Makefile` — adds `bench-smoke` and `race-soak` targets, `COUNT ?= 20`, and the T-11 flake-class disposition record beside `race-soak`.

## T-11 Flake-Class Audit

| Class | Finding after the `internal/stand` move | Disposition |
|-------|-----------------------------------------|-------------|
| 1. Port pick-to-bind | `internal/stand/stand.go` still carries `scenarioPortLo = 28100`, `scenarioPortHi = 32678`, `scenarioPortTries = 32` and the claim-while-bound registry; the private window sits below the kernel ephemeral range, so sibling `:0` binds cannot collide | Structurally mitigated — intact, no change |
| 2. SIGTERM before the handler | `armSignals()` (`cmd/sup-kv/main.go`) is called only after config parsing, logging setup and `wire` (bind + dump load); a SIGTERM landing in that pre-arm window still takes the default disposition (no final dump). `TestShutdown_secondSignal` pins the post-ownership second-signal force path, not the pre-arm window | Carried per D-09's bar — product-side window needs a Phase 3-5 fix; no `cmd/sup-kv/main.go` change this phase |
| 3. README/doc-test deadline | `TestReadmeQuickstart` had no deadline of its own; `runBashVerbatim` used only `t.Context()`, so a wedged block ran until the package `go test` timeout (local README run: 4.65s) | Fixed test-side — per-block `docBlockBudget` (1 min) clamped inside `t.Deadline()`; a wedged block now fails its own run with the block text and captured output |

## Decisions Made

- **Shared helper home moved one level out.** A compile probe confirmed `internal/udprepl`/`internal/grpcapi` cannot import `internal/app/internal/testutil` (Go internal rule: `use of internal package ... not allowed`). The implementation therefore lives in the new module-wide `internal/testutil` (path exempt from `no-test-libs-in-prod` via `**/test*/**`), and the plan's artifact path keeps a thin documented wrapper so the `internal/app` call site stays as planned.
- **Metric naming:** `p50-ms`/`p99-ms`, whitespace-free unit strings consistent with the existing `latency-ns/op`, `max-latency-ns` and `dump-bytes` metrics.
- **Per-op span definitions:** dump benchmarks measure the Encode+Save iteration; the burst measures the write call after the pacing tick (the ~1ms ticker is the burst's workload shape, not measured work — excluding it keeps p50/p99 about the write itself); confirmation measures submission to acked report (the existing span).
- **No self-pacing:** no sleeps, throttle or adaptive load were added in any measured section; the only new work is duration collection and post-loop reporting (with `b.StopTimer()` before the sort so reporting does not inflate `ns/op`).
- **Class 2 carried, not fixed:** D-09 makes the pre-arm SIGTERM window a Phase 3-5 product fix; this phase records the disposition and leaves the existing pin in place.
- **Verification counting corrected:** the plan's Task 1 verify uses `grep -c` (matching lines) with a threshold of 6, but Go prints one result line per benchmark carrying both metrics, so the maximum is 3 lines. Measured the stated intent instead: `grep -o 'p50-ms\|p99-ms' | wc -l` = 6 occurrences, and `grep -c 'p50-ms.*p99-ms'` = 3 lines (baseline 0).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Shared helper could not live only at `internal/app/internal/testutil/bench.go`**
- **Found during:** Task 1 (p50/p99 reporting)
- **Issue:** Go's internal-package rule forbids `internal/udprepl` and `internal/grpcapi` from importing `internal/app/internal/testutil`; a probe compile returned `use of internal package github.com/Djarvur/sup-kv/internal/app/internal/testutil not allowed`. The plan's "shared helper, three call sites" is impossible with a single file at that path.
- **Fix:** Implementation moved to the new module-wide `internal/testutil/bench.go`; `internal/app/internal/testutil/bench.go` (the plan's artifact) is a documented wrapper re-exporting it for the `internal/app` call site. One implementation, three call sites.
- **Files modified:** `internal/testutil/bench.go` (created), `internal/app/internal/testutil/bench.go` (created)
- **Verification:** `go vet` + `go tool golangci-lint run` clean on both packages; the three benchmarks report 6 percentile occurrences.
- **Committed in:** `72a4f1c` (Task 1 commit)

**2. [Rule 2 - Missing critical functionality] Class-3 deadline clamp landed in the shared doc runner**
- **Found during:** Task 2 (flake-class audit)
- **Issue:** The README test's deadline class can only be fixed where subprocesses are launched — `runBashVerbatim` in `cmd/sup-kv/examples_test.go` — which the plan's `files_modified` list omitted (it named `readme_test.go`). The plan's Task 2 Class 3 instruction explicitly authorizes the test-only clamp.
- **Fix:** Added `docBlockBudget` (1 minute, clamped 250ms inside `t.Deadline()`) and a deadline-specific fatal message; the same bound covers the EXAMPLES walkthrough because they share the runner.
- **Files modified:** `cmd/sup-kv/examples_test.go`
- **Verification:** `go test -race ./cmd/sup-kv/` green (README quickstart PASS in 4.65s baseline, well inside the budget); strict lint clean.
- **Committed in:** `d78eb84` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical functionality)
**Impact on plan:** Both are structural necessities, not scope creep: the helper move is forced by the language's internal-package rule (and preserves the plan's artifact path), and the deadline fix is the test-only change the plan's Class 3 instruction calls for.

## Issues Encountered

- **Task 1 verify's `grep -c` arithmetic (plan defect).** `grep -c` counts matching lines, and Go emits one benchmark result line per benchmark (each carrying both metrics), so a correct implementation yields 3 matching lines, never the stated 6. Verified the intent instead: 6 percentile occurrences (`grep -o ... | wc -l`), 3 lines carrying both metrics, baseline HEAD = 0. No implementation change was needed; the measurement is documented above.
- **Full 20× soak not run locally** — per the plan and RESEARCH A1, `make race-soak COUNT=1` was the local sanity pass (six packages green, 11.8s); the `-count=20` default and the ×3-green-across-runs bar belong to plan 11's scheduled job.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11's `scheduled.yml` can invoke `make bench-smoke` and `make race-soak` directly: the targets are the single definitions of TEST-10's smoke and TEST-11's `-count=20` command, so the workflow carries no duplicated flags.
- TEST-10 is fully covered by this plan (measured p50/p99, real persistfile, no self-pacing, CI-runnable smoke).
- TEST-11's command and class dispositions are ready; its ×3-green bar is realized by the scheduled job, and the carried SIGTERM-before-handler class remains a Phase 3-5 fix with its existing pin.
- Full gates green on the committed tree: `make lint`, `make test`, `make bench-smoke`, `make race-soak COUNT=1`.

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*

## Self-Check: PASSED

- `internal/testutil/bench.go`: FOUND
- `internal/app/internal/testutil/bench.go`: FOUND
- `internal/app/persist/bench_test.go`: FOUND
- `internal/udprepl/bench_test.go`: FOUND
- `internal/grpcapi/bench_test.go`: FOUND
- `cmd/sup-kv/examples_test.go`: FOUND
- `Makefile`: FOUND
- Commit `72a4f1c` (Task 1): FOUND (ancestor of HEAD)
- Commit `d78eb84` (Task 2): FOUND (ancestor of HEAD)
- Task 1 acceptance re-run on the committed tree: benchmark smoke exit 0; 3 benchmark lines; 6 `p50-ms`/`p99-ms` occurrences (the plan's stated intent); 3 lines carry both metrics; baseline HEAD had 0.
- Task 2 acceptance re-run on the committed tree: `grep -c '^race-soak:\|^bench-smoke:' Makefile` = 2; `make race-soak COUNT=1` exit 0 over all six packages.
- Plan verification re-run: `make bench-smoke` exit 0; `make lint` PASS; `make test` PASS.
- Measured plan commits: 2 (`03bee19..d78eb84`).
