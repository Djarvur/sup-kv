---
phase: 08-performance
plan: 08
subsystem: benchmarks
tags: [performance, benchmarks, latency, perf-01, perf-06, profiles, make-targets, grpcapi]
requires:
  - phase: 08-performance
    provides: the one threshold table and hardware block (08-01, internal/testutil/perf.go) and the shared port allocator (08-06)
provides:
  - both documented dump profiles benchmarked through one shared confirmation-latency body with their conditions stated in code
  - TestPerfFastProfileLatency, the fast profile's executable mean/p99 bound under `make perf`
  - bench-target completeness, with the Phase-3/4 core benchmarks reachable from `make bench` and `make bench-smoke` while CI stays threshold-free
affects: [08-10, PERF-01, PERF-06, Makefile, internal/grpcapi]
actuals:
  tokens: 3482
  tasks: 3
  commits: 3
plan_head_before: 5af17441d7b243c307666992dc1df20012513a9e
plan_head_after: e1a056481c29f0f2b4456d8427d2462952f668e4
tech-stack:
  added: []
  patterns:
    - "Profile table over one measured body: each arm owns only its knobs and dump implementation, so the fast arm's cited conditions cannot drift from the shared measurement"
    - "A perf gate wraps the shared benchmark body (never the table) in testing.Benchmark so the reported Extra metrics land on the result the gate reads"
    - "Benchmark targets list every benchmark package once; the CI job invokes the target by name, so its command has one definition"
key-files:
  created: []
  modified:
    - internal/grpcapi/bench_test.go
    - Makefile
key-decisions:
  - "The fast arm is byte-for-byte the old body (1 ms debounce, 50 ms max-wait, fakeDumpStore); the default arm changes only the knobs and the dump store, so the two arms differ exactly in the conditions the doc must state."
  - "The gate calls benchConfirmationLatency directly rather than BenchmarkConfirmationLatency: sub-benchmark metrics land on the sub-B's Extra map, never the parent's, so wrapping the table would silently read zero samples."
  - "Every reported metric is required by name (latency-ns/op, p50-ms, p99-ms, max-latency-ns); a missing key fails loudly instead of logging a silent zero."
  - "The measured fast profile (2.64 ms mean / 4.82 ms p99 on this host) is far inside the 25 ms/100 ms bounds; the bounds stay the documented 2x-headroom values and are not retuned to the run, because a self-paced bound measures the harness."
patterns-established:
  - "A benchmark's profile is data, not a second function: name, knobs and dump implementation in one struct, with the condition prose attached to each arm."
  - "A perf gate logs the hardware block before measuring, requires each Extra key by name, and its failure message carries every condition and the constant."
requirements-completed: [PERF-01, PERF-06]
coverage:
  - id: D1
    description: "BenchmarkConfirmationLatency runs the fast (1 ms/50 ms, in-memory dump) and default (50 ms/1 s, persistfile dump) profiles through one shared body, each arm reporting latency-ns/op, max-latency-ns, p50-ms and p99-ms with its conditions in code"
    requirement: PERF-01
    verification:
      - kind: command
        ref: "go test -count=1 -run '^$' -bench '^BenchmarkConfirmationLatency$' -benchtime 1x ./internal/grpcapi/ (PASS: fast 1.443 ms, default 89.43 ms; two p99-ms lines)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/grpcapi/ -run 'TestWait|TestSet|TestRestore' (PASS in 2.058 s — no fixture or port leak from the refactor)"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestPerfFastProfileLatency asserts the fast profile's mean and p99 against PerfFastMean (25 ms) and PerfFastP99 (100 ms), fails by key name on a missing metric, logs the hardware block and every condition, and skips in make test and under -race"
    requirement: PERF-01
    verification:
      - kind: command
        ref: "go test -race -count=1 -run '^TestPerfFastProfileLatency$' -v ./internal/grpcapi/ prints '--- SKIP: TestPerfFastProfileLatency' and exits 0"
        status: pass
      - kind: command
        ref: "make perf (PASS: mean 2.636708 ms, p50 2.525972 ms, p99 4.81614 ms, max 5.746541 ms, 813 samples, bounds 25 ms/100 ms, hardware block logged)"
        status: pass
    human_judgment: false
  - id: D3
    description: "bench and bench-smoke both include ./internal/app/repl/ with extended prose, bench-smoke keeps -benchtime=1x, and .github/workflows is unchanged and still invokes make bench-smoke"
    requirement: PERF-06
    verification:
      - kind: command
        ref: "make bench-smoke (PASS: BenchmarkHandleDataRequest, BenchmarkHandleDataSync, BenchmarkCoreEventLoop, BenchmarkNotifyWriteCost and BenchmarkDump10kKeys all present; 0.93 s wall)"
        status: pass
      - kind: command
        ref: "grep -c 'bench .*./internal/app/repl/' Makefile returns 2; git diff --exit-code .github/workflows/ exits 0"
        status: pass
    human_judgment: false
duration: 6min
completed: 2026-11-24
status: complete
---

# Phase 08 Plan 08: Two-Profile Confirmation Benchmark and Bench-Target Completeness Summary

**Decision 14.3's benchmark surface is whole: `fast` and `default` share one confirmation-latency body with their conditions in code, the fast profile has an executable 25 ms/100 ms bound under `make perf` (measured 2.64 ms mean / 4.82 ms p99), and the Phase-3/4 core benchmarks are reachable from both bench targets with CI untouched.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-11-24T19:21:38Z
- **Completed:** 2026-11-24T19:27:45Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments

- `BenchmarkConfirmationLatency` is now a two-arm table over one measured body. `fast` keeps the exact old conditions — `Debounce 1 ms`, `MaxWait 50 ms`, the in-memory `fakeDumpStore` — so the "~10 ms" claim's cited knobs cannot drift; `default` runs the operator knobs (`50 ms`/`1 s`) against `persistfile.New(filepath.Join(b.TempDir(), "dump.json"))`, so its measured span includes encode + fsync + rotate. Both arms still report `latency-ns/op`, `max-latency-ns`, `p50-ms` and `p99-ms`, and an unacked report still fails the run.
- `TestPerfFastProfileLatency` wraps the shared body (not the table — sub-benchmark metrics never reach the parent `Extra` map) in `testing.Benchmark`, requires every metric by key name, and asserts the mean and p99 against `testutil.PerfFastMean`/`PerfFastP99`. Its failure messages carry the profile, both knobs, the in-memory dump, the two loopback UDP adapters, the bufconn client, the burst pace, the value size and `race=off`. It skips in `make test` and under `-race`.
- `make perf` passes end to end: all four stand rows (burst 15.67 s, default-profile latency 36.32 s, amplification 36.45 s, memory 27.60 s), the paired absent-peer row (ratio 0.95 against 2.0), and the new fast-profile gate (2.39 s).
- The Makefile's `bench` and `bench-smoke` target lists now include `./internal/app/repl/`, so `BenchmarkHandleDataRequest`, `BenchmarkHandleDataSync`, `BenchmarkCoreEventLoop` and `BenchmarkNotifyWriteCost` are no longer orphaned. `bench-smoke` keeps `-benchtime=1x` and the anchored-empty `-run '^$$'` exclusion; `.github/workflows/scheduled.yml` still invokes the target and was not edited.

## Task Commits

1. **Task 1: split BenchmarkConfirmationLatency into the two documented profiles** — `4ad3156` (perf)
2. **Task 2: TestPerfFastProfileLatency — the fast arm's bound** — `bbcdfd6` (perf)
3. **Task 3: wire the core benchmarks into bench and bench-smoke** — `e1a0564` (perf)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/grpcapi/bench_test.go` — `benchProfile` (name, debounce, maxWait, realDump), `benchFastProfile`/`benchDefaultProfile` with their full condition prose, the two-arm table, the shared `benchConfirmationLatency` body, and `TestPerfFastProfileLatency`.
- `Makefile` — `bench`/`bench-smoke` package lists gain `./internal/app/repl/`; both prose headers name the core benchmarks, keep the no-threshold/manual contract, and restate the one-definition CI rule.

## Measured Conditions and Numbers

- **Hardware block (gate):** CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4, rotational=unknown); go=linux/amd64 go1.27.4; race=off; GOMAXPROCS=16.
- **Load:** `make perf` started at 3.81 2.76 2.78 (1/5/15 min); the gate's own block recorded 2.43 2.57 2.71. Dump base `$(CURDIR)/.tmp/perf` for the stand rows; the fast arm's dump is in-memory and the default arm's is a temp directory.
- **Fast profile under `make perf`:** mean 2.636708 ms (bound 25 ms), p50 2.525972 ms, p99 4.81614 ms (bound 100 ms), max 5.746541 ms, 813 samples — debounce 1 ms, max-wait 50 ms, in-memory dump, two loopback UDP adapters, bufconn client, ~1 ms background burst, 16-byte values, race=off.
- **Fast profile under `make bench` (auto-scaled):** 823 samples, 2.637412 ms mean, p50 2.481 ms, p99 4.962 ms, max 5.508685 ms — consistent with the gate's run.
- **Default profile:** 89.43 / 89.82 / 89.61 ms at `-benchtime=1x` across three bench-smoke runs; auto-scaled `make bench` measured 13 samples at 90.616635 ms mean, p50 89.98 ms, p99 92.95 ms, max 92.95 ms. The gap against the fast arm is the real file store plus the larger debounce/max-wait — exactly the condition difference the doc must state.
- **Bench-smoke cost:** baseline before this plan 0.64 s wall (one arm), 0.76 s with both arms (before the Makefile change), 0.93 s after adding `./internal/app/repl/` — a +0.29 s delta. Package times: persist 0.073 s, repl 0.170 s, udprepl 0.026 s, grpcapi 0.163 s. `make bench` (auto-scaled, all packages) measured 18.90 s wall. No benchmark was trimmed.
- **`make perf` phases:** stand rows 116.741 s, repl perf row 2.904 s, grpcapi gate 2.420 s, threshold-free stand rerun 6.059 s.

## Decisions Made

- **The arms differ only in conditions.** The fast arm is the previous body unchanged; the default arm changes only the knobs and the dump store. Anything else (topology, burst, payload, wait seam) would make the two numbers incomparable.
- **The gate wraps the body, not the table.** `testing.Benchmark`'s result `Extra` map is per-`B`; metrics reported by a `b.Run` sub-benchmark never reach the parent, so the gate passes `benchFastProfile` to the shared body directly.
- **Missing metrics are failures.** All four reported keys are required by name, so a refactor that drops `p99-ms` or `max-latency-ns` fails the gate instead of logging zeros.
- **Bounds are not retuned.** This host measured the fast arm far inside 25 ms/100 ms; the constants stay the documented ~2x-headroom values from the research row, because tightening them to this run would make the gate self-pacing.
- **The CI contract is the target, not a copy.** `bench-smoke` keeps its one definition in the Makefile and `scheduled.yml` was not edited; the threshold-free smoke remains the PR/scheduled contract.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical validation] A missing p50/max metric would have been logged as zero**
- **Found during:** Task 2
- **Issue:** The plan required only `latency-ns/op` and `p99-ms` to be present; the plan also asks the gate to log p50 and max, which would silently print `0s` if those keys disappeared.
- **Fix:** One local `metric(key)` helper requires all four keys by name and fails with the key in the message; the assertions consume the same values.
- **Files modified:** `internal/grpcapi/bench_test.go`
- **Verification:** `make perf` PASS with all four values logged; the failure path is guarded by construction.
- **Committed in:** `bbcdfd6`

---

**Total deviations:** 1 auto-fixed (Rule 2). No architectural changes, no threshold relaxation, no production code touched.

## Issues Encountered

- `golines` rejected the two long gate failure messages; `go tool golangci-lint fmt` split them into one-argument-per-line calls, after which `make lint` is clean.
- The default arm needs a real dump path: `persistfile.New` takes the file path, so the arm uses `filepath.Join(b.TempDir(), benchDumpFile)`. `testing.B` re-creates its temp directory after each auto-scaled `runN` cleanup, so repeated `make bench` invocations never reuse a generation (confirmed by the 13-iteration `make bench` run).

## User Setup Required

None — the new gate is operator-invoked through `make perf` and skips in `make test`.

## Known Stubs

None. Both arms run, the gate has a passing and a skipping path, and both bench targets were executed with the core packages present.

## Threat Flags

None. The change is test-only plus one Makefile target list: no new endpoint, auth path, file-access pattern or package install (T-08-SC). T-08-20 is mitigated by the per-arm condition prose and T-08-21 by asserting `.github/workflows/` is unchanged and `bench-smoke` stays threshold-free.

## Next Phase Readiness

Ready for the rest of wave 3. 08-10 can publish the two profiles with their now-executable conditions: the fast arm's in-memory-dump result (2.64 ms mean / 4.82 ms p99 on this host, bound 25/100 ms) must be stated with the in-memory dump and the 1 ms/50 ms knobs, while the default profile's operator conditions remain the stand row (08-07: p50 584 ms / p99 1.074 s) and this benchmark's real-dump arm (90.6 ms at 1 ms burst pacing) is a separate, differently shaped measurement — do not merge them into one number. The Phase-3/4 core numbers are now reachable from `make bench`/`make bench-smoke` for the doc's comparison tables.

## Self-Check: PASSED

- Files: `internal/grpcapi/bench_test.go`, `Makefile`, `.planning/phases/08-performance/08-08-SUMMARY.md` all found.
- Commits: `4ad3156`, `bbcdfd6`, `e1a0564` all ancestors of HEAD.
