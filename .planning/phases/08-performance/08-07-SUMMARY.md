---
phase: 08-performance
plan: 07
subsystem: perf-measurements
tags: [performance, latency, amplification, memory-model, perf-01, perf-03, perf-04, stand, percentiles]
requires:
  - phase: 08-performance
    provides: the single threshold table and hardware block (08-01, internal/testutil/perf.go) and the shared port allocator (08-06)
provides:
  - one exported nearest-rank LatencyPercentiles primitive shared by every benchmark and the stand's latency row
  - the measured default-profile write-latency distribution at 10k x 1 KiB under an open-loop 1000 writes/s x 15 s schedule on ext4
  - the measured disk-write amplification figure with its five-step method, media facts and tmpfs/ramfs refusal
  - the measured memory-model points (1k x 1 KiB, 10k x 1 KiB, 1k x 60 KiB) with both 60 KiB gates, ready for 08-10 to publish
affects: [08-10, PERF-01, PERF-03, PERF-04, internal/stand, internal/testutil]
actuals:
  tokens: 8957
  tasks: 3
  commits: 3
plan_head_before: dd842d618a08928868dd2a2f39fa8feacd8dfdbf
plan_head_after: 2131033
tech-stack:
  added: []
  patterns:
    - "Open-loop latency sampling from each op's scheduled instant to its completion, so queueing delay is in the sample (no coordinated omission)"
    - "Amplification as (Δ dumps × dump size)/logical bytes, cross-checked against /proc/<pid>/io write_bytes per dump on a refused-if-memory-backed filesystem"
    - "State→RSS points with a 2 ms VmRSS sampler across one forced dump, gated only on the value-dominated row"
key-files:
  created:
    - internal/testutil/bench_test.go
  modified:
    - internal/testutil/bench.go
    - internal/stand/perf_test.go
key-decisions:
  - "LatencyPercentiles is the one nearest-rank implementation (sorting in place, zero for empty); ReportLatencyPercentiles is a thin caller, and a known 1..100 ms sample pins p50=50 ms / p99=99 ms."
  - "The default-profile row reuses the frozen burst row's rate, window, worker, connection and RPC constants read-only and pins --sync-interval=10m so no listing round enters the window; the send-window end is the schedule's last slot, not the drain's end."
  - "Logical bytes for amplification are the client-submitted key+value bytes (key width 19, value 1024), stated in the log; the figure is published, never asserted, while bytes-per-dump is gated by PerfBytesPerDumpRatio."
  - "A zero-success open loop fails the measurement explicitly, so an all-error run cannot pass as a zero-sample distribution."
  - "The memory row logs all five figures and both ratios for every point and gates only the 60 KiB value-dominated point, where PerfDumpPeakLiveRatio gets its only consumer."
patterns-established:
  - "A perf row logs every raw input (counters on both sides, media facts, pids) so a failure is diagnosable without a rerun."
  - "A measurement precondition that cannot be satisfied (memory-backed dump, zero dumps, failed dump in-window, zero successful writes) fails loudly by name rather than publishing a number."
requirements-completed: [PERF-01, PERF-03, PERF-04]
coverage:
  - id: D1
    description: "One nearest-rank percentile implementation serves benchmarks and the stand; the default profile's p50/p99 are measured at 10k x 1 KiB under an open-loop 1000 writes/s x 15 s schedule on ext4 and asserted against PerfDefaultP50/PerfDefaultP99 with every condition in the failure message"
    requirement: PERF-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/testutil/ -run 'TestLatencyPercentiles' -v (PASS); BenchmarkDump10kKeys still reports p50-ms=76.03 / p99-ms=76.03"
        status: pass
      - kind: command
        ref: "SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$PWD/.tmp/perf go test -run '^TestPerfDefaultProfileLatency$' ./internal/stand/ (PASS: p50 584.34 ms, p99 1.0737 s, rate 1000.1/s, drain 654.8 ms; bounds 750 ms / 1500 ms)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Disk-write amplification is measured on a real filesystem with the documented method (Δ dumps × dump size vs logical bytes, cross-checked by write_bytes per dump), every raw input and media fact logged, and a tmpfs/ramfs dump base refused by name before any measurement"
    requirement: PERF-03
    verification:
      - kind: command
        ref: "SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$PWD/.tmp/perf go test -run '^TestPerfAmplification$' ./internal/stand/ (PASS: 15 dumps, 14.98 MB dump, 14,979,072 bytes/dump against the 16,475,181 bound, amplification 14.36x, failures 0)"
        status: pass
      - kind: command
        ref: "SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=/dev/shm go test -run '^TestPerfAmplification$' ./internal/stand/ (FAIL by name: 'SUPKV_PERF_DUMP_DIR=\"/dev/shm\" lies on tmpfs ...')"
        status: pass
    human_judgment: false
  - id: D3
    description: "The memory model has three measured state→RSS points with logical size, dump bytes, current RSS, VmRSS, VmHWM, pre-dump RSS and dump-cycle peak, and two gates on the value-dominated 60 KiB point: the loose RSS tripwire and the PerfDumpPeakLiveRatio dump transient"
    requirement: PERF-04
    verification:
      - kind: command
        ref: "SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$PWD/.tmp/perf go test -run '^TestPerfMemoryPoints$' ./internal/stand/ (PASS: 3 points; 60 KiB row rss 207.9-280.1 MB, peak growth 19.6-52.5 MB, both gates clear)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 -v -run '^TestPerfMemoryPoints$' ./internal/stand/ prints '--- SKIP: TestPerfMemoryPoints' and exits 0"
        status: pass
    human_judgment: false
duration: 15min
completed: 2026-11-24
status: complete
---

# Phase 08 Plan 07: Measured Latency, Amplification and Memory Points Summary

**The three numbers the phase publishes beyond the burst are now measurements with their conditions attached: default-profile p50 584 ms / p99 1.074 s at 1000.1 writes/s on ext4, 14.36× disk-write amplification with 1.0001× the dump file submitted per dump, and three state→RSS points whose 60 KiB row clears both memory gates.**

## Performance

- **Duration:** 15 min (approx.)
- **Started:** 2026-11-24T22:02:00+03:00 (approx.)
- **Completed:** 2026-11-24T22:14:00+03:00
- **Tasks:** 3
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- `internal/testutil/bench.go` now exposes `LatencyPercentiles(durations) (p50, p99)` — nearest-rank, sorting in place, zeroes for an empty slice — and `ReportLatencyPercentiles` is a thin caller. `internal/testutil/bench_test.go` pins the rank rule on a known sample set (1..100 ms → 50 ms / 99 ms, a single sample, an empty slice). The benchmark smoke still prints `p50-ms`/`p99-ms` (`BenchmarkDump10kKeys`: 76.03 / 76.03).
- `TestPerfDefaultProfileLatency` preloads 10 000 × 1 KiB through the frozen seeding helper, waits for the covering dump and an empty unacked set, pins `--sync-interval=10m`, and drives an open-loop 1 000 writes/s × 15 s schedule over 100 hot keys with 1 024 workers over 8 connections. Each sample is completion minus its scheduled instant, so queueing delay cannot hide. The verdict line names p50/p99, the achieved rate, both key counts, the value size, the filesystem and `race=off`. Measured standalone: **p50 584.34 ms, p99 1.0737 s, 1000.1 writes/s, drain 654.8 ms** against the 750 ms / 1 500 ms bounds; the `make perf` rerun measured p50 581.55 ms / p99 1.0719 s / drain 642.8 ms.
- `TestPerfAmplification` implements PERF-03's five-step method: a 10 000 × 1 KiB preload and its covering dump as the baseline, then an open-loop 1 000/s × 15 s hot-key window, with the tuple `{supkv_dumps_total, dump file size, /proc/<pid>/io write_bytes, summed failure classes}` taken on both sides. It refuses a tmpfs/ramfs dump base by name, refuses a zero-dump window and a moved failure counter, and fails when bytes submitted per dump exceed `testutil.PerfBytesPerDumpRatio × dump size`. Measured: **dumps Δ 15, dump 14,977,437 bytes, write_bytes Δ 224,686,080 → 14,979,072 bytes/dump (1.0001× the file, bound 16,475,181), amplification 14.36×, failures 0** on ext4 (`/dev/mapper/ubuntu--vg-ubuntu--lv`, rotational=unknown).
- The `/dev/shm` refusal was exercised: the row fails in 0.00 s with `SUPKV_PERF_DUMP_DIR="/dev/shm" lies on tmpfs: a memory-backed dump measures memory, not disk`.
- `TestPerfMemoryPoints` measures three fresh-pair points and logs logical size (Σ(len(key)+len(value)), key width 14), dump bytes, current RSS, VmRSS, VmHWM, pre-dump RSS, the 2 ms-sampled dump-cycle peak and both ratios, plus the filesystem. Only the value-dominated 60 KiB point gates. `make perf` measured:
  - 1 000 × 1 KiB: logical 1,038,000; dump 1,492,779; rss 28.8 MB (27.73×, no gate); peak growth 0.00×.
  - 10 000 × 1 KiB: logical 10,380,000; dump 14,927,532; rss 66.9 MB (6.45×, no gate); peak growth 0.51×.
  - 1 000 × 60 KiB: logical 61,454,000; dump 82,044,779; rss 280.1 MB (4.56× against the 1,005.9 MB tripwire); peak growth 19.56 MB (0.32× against the 122.9 MB `PerfDumpPeakLiveRatio` bound). The standalone run measured rss 207.9 MB (3.38×) and peak growth 52.5 MB (0.85×).
- All three rows reuse the frozen file's seeding helper, metrics parser and burst constants read-only; no second percentile, seeding or parsing implementation exists.

## Task Commits

1. **Task 1: shared percentile primitive and default-profile latency row** — `a932fe2` (perf)
2. **Task 2: disk-write amplification with method, media and refusal** — `2dc2da1` (perf)
3. **Task 3: memory-model points and dump-transient gate** — `2131033` (perf)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/testutil/bench.go` — `LatencyPercentiles` exported; `ReportLatencyPercentiles` delegates; rank rule unchanged.
- `internal/testutil/bench_test.go` — `TestLatencyPercentiles` pins nearest-rank on 1..100 ms, a single sample and an empty slice.
- `internal/stand/perf_test.go` — the three perf rows, the open-loop helper, the dump-cycle RSS sampler, the `/proc` readers and the snapshot tuple; one measurement-shape const block.

## Measured Conditions

- **Hardware block:** CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4, rotational=unknown); go=linux/amd64 go1.27.4; race=off; GOMAXPROCS=16.
- **Load:** standalone runs at 1.70 / 2.93 / 4.57 (latency / amplification / memory), the `make perf` sweep at 7.91 → 3.70 (1-minute averages, reported per test in each hardware block).
- **Media:** ext4 on `/dev/mapper/ubuntu--vg-ubuntu--lv`. This is workstation NVMe, **not** the two-board appliance's target-grade eMMC/SD — D-03's honest outcome is method + measured row + a recorded gap, which 08-10 publishes against Spec open question #2.
- **Dump base:** `$(CURDIR)/.tmp/perf` (gitignored), per-test subdirectory, filesystem type logged by `PerfDumpDir`.

## Decisions Made

- **One rank rule, one implementation.** `LatencyPercentiles` owns the sort and the nearest-rank selection; the reporter and the stand both call it, and the known sample set is the drift barrier.
- **Open loop measures what the client felt.** A job's sample starts at its scheduled instant, so a backlog lengthens the sample instead of disappearing (PERF-01's coordinated-omission requirement, T-08-19).
- **The drain bound is anchored to the send window's end.** The window end is the schedule's last slot, exactly as the frozen burst row computes it; the drain check exists as a precondition, not as a second PERF-02 bound.
- **Amplification is published, bytes-per-dump is gated.** The ratio is a policy consequence (full-dump, ADR 0003) and is logged raw; `PerfBytesPerDumpRatio` only catches an accidental multi-copy generation.
- **Logical-byte accounting is explicit.** For the amplification denominator, each successful hot-key write contributes its key width (19 bytes) plus its 1 024-byte value; the row logs both terms and the write count so the figure is reproducible.
- **The 60 KiB row carries both memory gates.** The key-dominated points are logged, not asserted, because their overhead is the model's contrast point (research Pitfall 8), not a regression.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical validation] Zero-success open loops would have passed vacuously**
- **Found during:** Task 1 / Task 2 implementation
- **Issue:** If every scheduled write failed, `LatencyPercentiles` would return `(0, 0)` and both latency assertions would pass; the amplification denominator would also be zero. The plan's action text did not name this case.
- **Fix:** Both open-loop rows fail explicitly when `result.ok == 0`, naming the error counts by gRPC code; the amplification row's message states the denominator would be zero.
- **Files modified:** `internal/stand/perf_test.go`
- **Verification:** `make perf` passes with 15 000/15 000 successful writes in both rows; the guard is exercised only by construction (no error run was produced).
- **Committed in:** `a932fe2` (latency), `2dc2da1` (amplification)

**2. [Rule 3 - Blocking issue] Task 3's skip verify omitted `-v`, so a correct skip could never satisfy it**
- **Found during:** Task 3 verification
- **Issue:** The plan's `<automated>` runs `go test ... ./internal/stand/ | grep -q -- '--- SKIP: TestPerfMemoryPoints'` without `-v`. Go prints `--- SKIP:` lines only in verbose mode, so the pipeline exits 1 for any correctly skipping test; the command as written is unsatisfiable.
- **Fix:** Ran the same command with `-v`; it printed `--- SKIP: TestPerfMemoryPoints (0.00s)` and exited 0. No test change was needed — the row already skips through `testutil.RequirePerf`. The `make test` gate likewise runs the row as a skip.
- **Files modified:** none (verification-command correction)
- **Verification:** `SUPKV_PERF= go test -race -count=1 -v -run '^TestPerfMemoryPoints$' ./internal/stand/` → `--- SKIP: TestPerfMemoryPoints`; `make test` exits 0 with the row skipped.
- **Committed in:** `2131033` (the row's implementation)

---

**Total deviations:** 2 (1 auto-fixed by Rule 2, 1 verification-command correction under Rule 3). No scope creep; no production code changed.

## Issues Encountered

- The first `make lint`-equivalent pass flagged `golines` on three over-200-column failure messages, and `gci` on the `+` spacing around a multi-line `Logf`. Split the messages and ran `go tool golangci-lint fmt` on the file.
- `unparam` flagged `perfMetric`'s metric-name parameter (always `supkv_dumps_total`); renamed to `perfDumps` with no name parameter. `nolintlint` flagged a `paralleltest` nolint on the wrong line; moved it to the range statement.
- The plan's `rss` gate is only meaningful for the value-dominated row; the 1 000 × 1 KiB row's ratio is 27.7× because the process baseline dominates a 1 MB state. Both ratios are logged for all points as the plan requires, and no gate fires there.

## User Setup Required

None — the rows are operator-invoked via `make perf` and skip in `make test`.

## Known Stubs

None. Every published figure comes from a run whose command, hardware block and media are recorded; no placeholder, skipped assertion or unrun verify was left behind.

## Threat Flags

None. The change is test-only: `/proc/<pid>` reads are limited to the test's own child pids (T-08-18, accepted), no new endpoint or auth surface exists, and no package was installed (T-08-SC).

## Next Phase Readiness

Ready for the rest of wave 3. PERF-01, PERF-03 and PERF-04 now have reproducible measurements with conditions; 08-10 can copy the rows and the hardware block into the performance doc, and D-03's media gap is explicit (workstation NVMe, not target-grade eMMC/SD). The two 60 KiB memory gates clear with wide margins in both runs, so the streaming path's one-copy claim has a measured contrast to the old design.

## Self-Check: PASSED

- Files: `internal/testutil/bench.go`, `internal/testutil/bench_test.go`, `internal/stand/perf_test.go`, `.planning/phases/08-performance/08-07-SUMMARY.md` all found.
- Commits: `a932fe2`, `2dc2da1`, `2131033` all ancestors of HEAD.
