---
phase: 08-performance
researched: 2026-11-23
head: bf6966f6eb80ecd984aad82a3e3c70c4c5cae289
branch: gsd/v1.1-milestone
confidence: >-
  HIGH for the as-built measurement map and every mechanism recommendation
  (each claim read from the named file this session or produced by a command
  run this session); MEDIUM for threshold starting values and the
  target-media evidence (the target media is not present on this host).
---

# Phase 8: Performance - Research

**Researched:** 2026-11-23
**Domain:** Go 1.27 measurement design — deterministic cost gates, real-binary stand perf tests, dump latency/amplification/memory models, absent-peer cost, flake-class removal
**Confidence:** HIGH for the current-state map and mechanisms; MEDIUM for threshold starting values (marked as starting values, not locked numbers)

**Provenance:** every `path:line` reference below is a `[VERIFIED: path:line]` read performed in this session, and the discrete values quoted beside a claim are verbatim from that file; `[CITED: …]` marks an external document fetched this session; starting values and unverified environment assumptions are collected in the Assumptions Log as `[ASSUMED]`. In-repo review artifacts (`docs/reviews/…`) are secondary evidence and are labelled as such.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** One documented latency model covers both profiles with their measurement conditions. The default profile: under sustained write flow the quiet window never arrives, so the dump runs on `dump_max_wait` and write latency reaches `dump_max_wait` + dump time by design; a single sequential client does at most ~14 writes/s because `dump_debounce` waits for quiet. The fast-response profile (`Debounce` 1 ms / `MaxWait` 50 ms) is documented with its disk cost and carries the "~10 ms under 1000 writes/s" claim, tied to the exact benchmark conditions it was measured under (`internal/grpcapi/bench_test.go`'s parameters), never stated standalone. — [auto] [Model] Q: where do the two profiles live? → Selected: one model section, both profiles, each number with its conditions (S-FAIL-15, 14.3). — **Reversibility:** reversible.
- **D-02:** The 1000 writes/s burst over distinct keys becomes a standing test with a bound, not a folklore number: the XPORT-02(3) scenario runs in the standing suite, the docs state what is guaranteed (dump cadence, one datagram per write, no loss of confirmed writes) and what is not (distinct-key bursts can storm retransmissions; only hot-key bursts hold the rate today), and the `S-FAIL-15` registry row is closed against the shipped test. — [auto] [Burst] Q: stand the burst up or document it? → Selected: standing test + explicit guarantee/non-guarantee text (R-02, PERF-03). — **Reversibility:** reversible.
- **D-03:** Disk-write amplification is measured with a documented method (bytes written by the dumper per logical byte written by clients, on the available media, with the state size and cadence named) and published with a hardware description, or the gap is recorded honestly against Spec open question #2 — the design is not changed: decision 14.3 (full-dump policy) stands, recorded in ADR 0003. — [auto] [Amplification] Q: measure or record the gap? → Selected: measure on target-grade media where available, else record the gap with the method and the missing hardware (PERF-04). — **Reversibility:** reversible.
- **D-04:** The memory model is documented from stand measurements: one dump cycle allocates a small multiple of the state size, RSS tracks the state plus that transient, and the measured points (state size → RSS, one dump cycle) are published with their hardware description and tied to the streaming-encode semantics delivered by PERSIST-06 (Phase 5). — [auto] [Memory] Q: formula or points? → Selected: formula + measured points with conditions (P-06, PERF-05). — **Reversibility:** reversible.
- **D-05:** Write latency with an absent peer and a growing unconfirmed set is proven near live-peer latency by a benchmark on the same fix as XPORT-02; the benchmark states the unacked set size, the peer state and the ratio bound, and the paired measurement at a live peer is stored in the same test so a regression fails by name. While here, 04 New-3 is fixed: the paced replay path must not copy `rec.Data` twice before discarding it. — [auto] [Absent peer] Q: benchmark shape? → Selected: paired benchmark, absent vs live peer, ratio bound (PERF-06, 04-New-3). — **Reversibility:** reversible.
- **D-06:** `make perf` runs the stand suite with thresholds defined in one place, and every published number carries a hardware description block (CPU, RAM, disk, load). Thresholds are documented bounds (not stored profiles, not self-pacing), and CI keeps running the threshold-free smoke target as today; `make perf` is the named operator-facing target. — [auto] [Perf target] Q: one target or profiles? → Selected: one `make perf` with central thresholds and hardware blocks (D-13/D-14 stay for CI smoke). — **Reversibility:** reversible.
- **D-07:** The routed flake classes are closed, not muted: the two wall-clock ratio gates (`TestCore_keyCountCostFlat`, `TestCore_unackedCostFlat`) lose their load sensitivity (drop `t.Parallel` where it contaminates, or replace the wall-clock ratio with a deterministic operation-count assertion), `TestScenario_dumpNotify` gets deterministic sequencing, the `freeUDPPort` pick-then-bind class is structurally removed, the port-contention/lifecycle flakes get a serialized or ephemeral-port cure, and `killAndReap` reports an already-exited child with its captured stderr/exit status. 04 New-5 strengthens `TestChunkSyncEntries_capacityFollowsEntries` to pin the slice-sizing claim. — [auto] [Flakes] Q: fix or record? → Selected: fix the classes with deterministic assertions; a bound is never relaxed to green (04 New-3/New-5, 05/06/07 deferred-items). — **Reversibility:** reversible.

### Claude's Discretion

- Where the performance model lives (new `docs/PERFORMANCE.md` vs a section of OPERATIONS), the exact threshold values and their basis, benchmark names and the shape of the hardware block, and the mechanism for deterministic ratio gates (operation counters vs serialized measurement).

### Deferred Ideas (OUT OF SCOPE)

- Dead-code deletion and the registry resweep (`New-3` from the Phase-7 re-verification) stay in Phase 9.
- Any dump-policy redesign is out of scope: decision 14.3 stands (ADR 0003); only documentation and measurement change here.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description (from REQUIREMENTS.md) | Research Support |
|----|-----------------------------------|------------------|
| PERF-01 | The write-latency model is documented for both profiles (PERF-01, PERF-02, 14.3) | Exact defaults, cadence formulas and the fast-profile benchmark conditions mapped in "PERF-01" below; model doc recommended at `docs/PERFORMANCE.md`; deterministic cadence proof already exists at `TestEngine_burstBoundedConfirmation` |
| PERF-02 | The 1000 writes/s burst over distinct keys is a tested bound (PERF-03) | The XPORT-02(3) scenario already exists as `TestRed_R02_burst` and passes; the fix is to expose it to `make perf` as a perf row without weakening the red contract, plus the guarantee/non-guarantee doc text and the S-FAIL-15 row update |
| PERF-03 | Disk-write amplification is measured on target-grade media or the gap is recorded (PERF-04, Spec open question #2) | Method: dump size × dump count from `supkv_dumps_total` and `/proc/<pid>/io` `write_bytes`; target media absent on this host, so the gap+method is the honest outcome (D-03 permits it) |
| PERF-04 | The memory model is documented from stand measurements (PERF-05, with PERSIST-06) | One-copy streaming path quoted (`scheduler.runDump`, `EncodeTo`, 64 KiB bufio); formula + measured points; RSS source is the phase-7 current-RSS metric; transient source is the dump benchmark's `peak-live-B` |
| PERF-05 | Write latency with an absent peer and a growing unconfirmed set stays near the live-peer latency (PERF-06; same fix as XPORT-02, proven by a benchmark) | Fundamental gap found: `trackedVersion`/`removeSupersededLocal`/`applyRemote` still scan the whole pending set per write/datagram (IN-02 open); per-key index + paired live/absent benchmark + `PendingVisits` gate specified |
| PERF-06 | `make perf` runs the stand suite with thresholds; published numbers carry a hardware description | `internal/stand` harness, load generator and metrics scrape are reusable; one const threshold file in `internal/testutil`, `SUPKV_PERF=1` gate, `-p 1`, no `-race`, hardware block helper, CI unchanged |
</phase_requirements>

## Summary

Phase 8 is a measurement-and-flake phase, not a design phase. The runtime require
set stays pinned (gRPC + protobuf only), the dump policy stays untouched
(ADR 0003 / decision 14.3), and no new harness is needed: `internal/stand` already
builds the real binary once, launches real nodes, scrapes `/metrics`, serves a
gRPC client pool and contains the reviewer's open-loop load generator
(`redRunBurst`). The three benchmark packages behind `make bench` already report
the right shape (ns/op, p50/p99, bytes/op, peak-live-B, writes/s), and the
Phase-3/4 core benchmarks report the handler/event-loop shape (they are just not
wired into the target yet). What is missing is a single perf entry point,
central thresholds, a hardware block, and gates that do not depend on wall
clock.

The central finding is that the two routed wall-clock ratio gates can be made
**deterministic** with an operation-count assertion, not a serialized timing
run. The `StateStore` interface already makes an O(store)-per-datagram read
compile-impossible (its whole-store method was removed for XPORT-06), so the
key-count claim is exactly "a constant number of store calls per datagram"; a
counting test double proves that in milliseconds. The unacked claim is exactly
"an idle `Advance` inspects zero pending entries"; a `PendingVisits` counter in
the core's few pending loops proves that deterministically. `testing.Benchmark`
is then only the manual measurement tool (`make bench` / `make perf`), never the
assertion.

A second finding is that XPORT-02 did **not** finish the absent-peer cost fix.
The review's `mirrorState`/`PendingSnapshot` whole-set copies are gone, but every
local write still scans the whole pending map twice
(`trackedVersion`, `removeSupersededLocal`: `internal/app/repl/core.go:3429-3452`)
and every inbound `DataReplication` scans it once (`applyRemote`:
`internal/app/repl/core.go:1659`). That is 03-REVIEW IN-02, still marked `open`
with "Note for PERF/Phase 8" (`.planning/phases/03-transport/03-REVIEW-DISPOSITION.md:22`).
PERF-05 therefore needs the per-key pending index plus a paired absent/live
benchmark — and that benchmark is the honest home for the proof, because the
existing `BenchmarkCoreEventLoop` deliberately measures only an idle `Advance`
and cannot see these scans.

The third cluster is the flake classes. The root causes are concrete: the
key-count gate carries `t.Parallel` while the sibling gate is deliberately
sequential (`internal/app/repl/bench_test.go:172` vs `:279`); the dumpNotify
harness settles goroutines with 200 µs sleeps and a "any activity" barrier
(`internal/app/repl/sim_dump_test.go:248-312`); three packages still pick UDP
ports from the kernel ephemeral pool and then bind later
(`cmd/sup-kv/main_test.go:141-149`, `internal/udprepl/smoke_test.go:47-60`,
`internal/grpcapi/wait_integration_test.go:61-75`), while only `internal/stand`
and the cmd broadcast probe use the private window; and `killAndReap` turns
`os.ErrProcessDone` into a bare fatal without the child's status or captured
stderr (`internal/stand/stand.go:423-439`). Each has a deterministic cure
documented below.

**Primary recommendation:** Do the phase as five workstreams, in this order:
(1) replace the two wall-clock gates with operation-count gates and add the
per-key pending index (the code fix PERF-05 needs, closing IN-02);
(2) fix the dumpNotify harness on `testing/synctest` and the port allocator /
`killAndReap` diagnostics; (3) add the core/API perf benchmarks with a paired
absent/live arm and remove the 04 New-3 double copy; (4) add the stand perf
tests (default profile, fast profile, burst, amplification, memory) gated by
`SUPKV_PERF=1`; (5) add `make perf`, one threshold file, the hardware block and
`docs/PERFORMANCE.md`, and update S-FAIL-15 plus the `.planning/PROJECT.md`
"~10 ms" claim with its conditions.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Deterministic cost assertions (store calls, pending visits) | Application core tests (`internal/app/repl`) | — | The claim is about the core's work per event; only the core's own counters and the consumer-side `StateStore` seam can see it |
| Wall-clock benchmark measurement (ns/op, p50/p99) | Benchmark test tier (`internal/app/{persist,repl}`, `internal/grpcapi`, `internal/udprepl`) | — | `testing.B` + `ReportMetric` is the calibrated measurement surface; never a production path |
| Write-latency / amplification / memory measurement at the real binary | Test harness (`internal/stand`) | OS (`/proc`, `statfs`, mountinfo) | The stand owns the real process, real dump path, `/metrics` scrape and the load generator |
| Threshold storage and hardware description | Test utility (`internal/testutil`) | Docs (`docs/PERFORMANCE.md`) | One const table consumed by every perf test; the doc carries the human-readable block and commands |
| Dump cadence and byte accounting | Application core (`internal/app/persist`) | Adapter (`internal/persistfile`) | `scheduler.runDump` owns the cycle; the adapter owns the 64 KiB buffered write and fsync sequence |
| Port allocation for socket tests | Test utility (`internal/testutil`) | Every socket-owning test package | One allocator with disjoint package bands and a per-process claim registry |
| Virtual-time determinism of the engine simulator | Simulator test tier (`internal/app/repl`) | `testing/synctest` | The harness starts a real `persist.Engine` goroutine over the virtual scheduler; synctest supplies the idle barrier |
| `make perf` entry point | Build tooling (`Makefile`) | — | One operator-facing command; CI keeps the threshold-free smoke target |
| Published-number drift guard | Documentation test (`cmd/sup-kv`) | `internal/testutil` thresholds | The existing `TestOperationsDoc` pattern: doc anchors cross-checked against code |


## Measurement Infrastructure as Built

Everything below exists today and is reused; the phase adds test files and a
Makefile target, not a second harness.

| Asset | Location | What it measures / provides | Gate or report |
|-------|----------|-----------------------------|----------------|
| `BenchmarkHandleDataRequest`, `BenchmarkHandleDataSync` | `internal/app/repl/bench_test.go:107-164` | One inbound handler call at 1 000 and 50 000 stored keys; fixtures seeded once outside the loop; virtual time stepped one limiter slot per iteration | Reports ns/op and `store-keys`; no threshold |
| `TestCore_keyCountCostFlat` | `internal/app/repl/bench_test.go:171-205` (constants `:25-47`) | XPORT-06 2× acceptance; `testing.Benchmark` warm-up + measurement per size | **Wall-clock gate**, `t.Parallel` at `:172`, bound `benchMaxRatio = 2.0` at `:35` |
| `BenchmarkCoreEventLoop` | `internal/app/repl/bench_test.go:245-269` (fixture `:223-238`) | Idle `Advance` + `UnackedStats` + `PeerCount` at 10 and 10 000 pending; all retry deadlines future; **deliberately excludes** write/apply paths | Reports ns/op and `unacked`; no threshold |
| `TestCore_unackedCostFlat` | `internal/app/repl/bench_test.go:280-309` (constants `:210-217`) | XPORT-02(1) 2× acceptance on the idle iteration | **Wall-clock gate**, sequential with `//nolint:paralleltest` at `:279`, bound `benchUnackedMaxRatio = 2.0` at `:216` |
| `BenchmarkDump10kKeys` | `internal/app/persist/bench_test.go:107-141` (fixtures `:45-61`, `countingWriter` `:65-75`, `liveHeapBytes` `:80-84`) | One real streamed dump cycle at 10 000 keys through `Records()` + `EncodeTo` + `persistfile.FileStore.SaveStream` | Reports ns/op, `p50-ms`, `p99-ms`, `bytes/op`, `dump-bytes`, `peak-live-B`; no threshold |
| `BenchmarkConfirmationLatency` | `internal/grpcapi/bench_test.go:68-244` | One `Set(WaitForReplication)` through plain gRPC over bufconn, two real loopback UDP adapters, a ~1 ms background burst, production wait seam | Reports `latency-ns/op`, `max-latency-ns`, `p50-ms`, `p99-ms`; **fast profile only**: `Debounce: time.Millisecond`, `MaxWait: 50 * time.Millisecond` at `:85-86`; in-memory dump (`fakeDumpStore`, `internal/grpcapi/server_test.go:80-110`) |
| `BenchmarkReplicationBurst` | `internal/udprepl/bench_test.go:56-99` | Two-node real-UDP burst at ~1 ms cadence, measured to full convergence | Reports `writes/s`, `max-unacked`, `p50-ms`, `p99-ms`; no threshold; convergence-inclusive rate reads ~420-440/s in the review evidence, so it is not the 1 000/s proof |
| Percentile reporter | `internal/testutil/bench.go:27-44`; wrapper `internal/app/internal/testutil/bench.go:11-16` | Nearest-rank p50/p99 in ms under whitespace-free metric names | Shared by all benchmark packages |
| Persist retry-ladder synctest suite | `internal/app/persist/backoff_test.go:1-14,114-250`; `timeout_test.go:179-300`; `engine_blocking_test.go:101-132` | Exact virtual instants + `synctest.Wait()` idle barrier over in-memory ports | Precedent for the dumpNotify cure |
| Deterministic burst-cadence proof | `internal/app/persist/engine_test.go:644-693` (constants `:322-324`) | 1 000 writes at 1 ms virtual cadence (default `50 ms/1 s`) → first dump at exactly 1 s, ≤3 dumps, all confirmed | Green test; the model's executable half |
| Stand harness | `internal/stand/stand.go:59-150,320-525,527-700` | Build-once real binary, launch/SIGKILL/SIGTERM, wait helpers, real gRPC clients, log capture | Test infrastructure |
| Stand port allocator | `internal/stand/stand.go:175-228` (constants `:88-92`) | Private window `[28100, 32678)` below the kernel ephemeral range + per-process claim registry; probe-retry at `newStandPair` `:548-583` | Only `internal/stand` and `cmd/sup-kv/broadcast_probe_test.go` use it |
| Stand metrics scrape | `internal/stand/stand_red_test.go:194-231`; `internal/stand/stand_metrics_test.go:172-260` | Exact-value `/metrics` scrape + per-family extraction | Reusable by perf tests |
| Stand open-loop load generator | `internal/stand/stand_red_test.go:470-515` | 1 024 workers over 8 connections, 10 s RPC deadline, schedule-based open loop, errors by gRPC code | Reusable; extend with per-op latency |
| Burst scenario | `internal/stand/stand_red_test.go:537-600` (constants `:91-116`) | 1 000 writes/s × 15 s over 10 000 distinct keys; ≥990/s achieved; unacked empty ≤5 s after the send window | Env-gated by `SUPKV_RED_TESTS=1`; already green in `scripts/red-expectations.tsv` |
| Unacked-drain scenario | `internal/stand/stand_red_test.go:381-455` | 10 000 unacked 1 KiB records, peer returns; drain ≤ N/rate + 10 s; retransmits ≤3N | Env-gated; green |
| 10 k × 1 KiB nightly stand | `internal/stand/stand_test.go:100-145,351-455` | Real binary, default documented flags, empty-disk recovery | Nightly scale selected by `SUPKV_STAND_NIGHTLY=1` |
| Makefile measurement targets | `Makefile:32-38` (`bench`), `:40-41` (`bench-smoke`), `:66-67` (`race-soak`), `:77-78` (`red-run`) | Manual benchmark, CI 1x smoke, 20× race soak, assert-red runner | `make perf` is new; note `bench`/`bench-smoke` currently list only persist, udprepl and grpcapi — the Phase-3/4 core benchmarks in `internal/app/repl` are **not** wired in (see PERF-06) |
| CI | `.github/workflows/ci.yml:88-93` (lint/gate); `.github/workflows/scheduled.yml:64,78,98,127,147,168` | PR gate `go test -race ./...`; nightly bench-smoke, red-run, race-soak, nightly stand, R-01 ×20 | **No thresholds in CI by design** (D-13/D-14) |
| Docs anchors | `cmd/sup-kv/operations_test.go:48-171` | Literal-anchor drift guard for `docs/OPERATIONS.md` | Pattern to copy for `docs/PERFORMANCE.md` |

The stand's non-perf scenario tests — `stand_test.go` (lifecycle, kill/restart,
both-down, SIGTERM, empty-disk, oversize key, startup-gate), `stand_delete_test.go`,
`stand_buffers_test.go`, `stand_metrics_test.go`, `stand_form_test.go`,
`persist_test.go` — carry derived timeouts/convergence bounds, not performance
thresholds. They are the functional suite; their shared deadline constants live
at `internal/stand/stand.go:44-51` and the scenario-derived bounds at
`internal/stand/stand_test.go:77-85`. `make test` and the nightly stand job run
them; the new `make perf` rows do not re-run them (see PERF-06 for the literal
reading option).

Baseline measurements run this session on the research host (not published
claims; starting-value evidence only). Host: Intel i9-9980HK, 16 logical CPUs,
64 GiB RAM, NVMe `APPLE SSD AP1024N` on ext4/LVM, load average ~8.07, Go
1.27.4, no `-race`.

| Command | Result |
|---------|--------|
| `make bench-smoke` (1x) | `BenchmarkDump10kKeys`: 63.3 ms, `dump-bytes` 5 147 571, `peak-live-B` 10 436 384; `BenchmarkConfirmationLatency` (fast profile): 1.87 ms single sample; `BenchmarkReplicationBurst`: 1 sample |
| `go test -run '^$' -bench '^BenchmarkDump10kKeys$' -benchtime 2s ./internal/app/persist/` | 113.8 ms/op, p50 113.0 ms, p99 182.6 ms, `bytes/op` 19.0 MB, `dump-bytes` 5 147 571, `peak-live-B` 17 500 160 |
| `go test -run '^$' -bench '^BenchmarkConfirmationLatency$' -benchtime 2s ./internal/grpcapi/` | 11.53 ms mean, p50 6.10 ms, p99 44.93 ms, max 68.0 ms |
| `go test -race -count=1 -run '^(TestCore_keyCountCostFlat\|TestCore_unackedCostFlat)$' ./internal/app/repl/` | keyCount 0.87 / 0.91, unacked 1.05 — **passes, 79 s wall**; the two gates are the slowest tests in view |
| `SUPKV_RED_TESTS=1 go test -count=1 -run '^TestRed_R02_burst$' ./internal/stand/` | PASS, 15.79 s (its own ≥990/s and ≤5 s drain assertions) |
| `SUPKV_RED_TESTS=1 go test -count=1 -run '^TestRed_R02_unackedDrain$' ./internal/stand/` | PASS, 37.80 s; precondition log `9985/10000 unacked` confirms the event loop trails the seed |

## Standard Stack

No new module dependency is proposed. Everything below is the Go standard
library, the existing in-repo test infrastructure, or the OS accounting files.
The `go.mod` runtime requires stay exactly gRPC + protobuf; the `tool`
directives are unchanged.

### Core

| Facility | Version | Purpose | Why standard here |
|----------|---------|---------|-------------------|
| `testing` (`B`, `ReportMetric`, `Benchmark`, `BenchmarkResult.Extra`, `AllocsPerRun`) | Go 1.27.4 | Benchmarks, per-metric reporting, per-op counters | Already the repo's measurement surface: `b.ReportMetric(float64(keys), "store-keys")` at `internal/app/repl/bench_test.go:123,156`; `ReportLatencyPercentiles` at `internal/testutil/bench.go:42-43` |
| `testing` fixed-iteration loop | Go 1.27.4 | Deterministic count gates (no timing) | A plain `for range K` plus counter deltas is load-independent; avoids `testing.Benchmark`'s ≥1 s auto-scaling |
| `testing/synctest` | stdlib (stable since Go 1.25) | Deterministic goroutine-idle barrier for the dumpNotify harness | `synctest.Test(t, func(t *testing.T){...})` + `synctest.Wait()`; already used at `internal/app/persist/engine_test.go:650-654`, `backoff_test.go:114,134`; verified constraints: inside the bubble `T.Run`, `T.Parallel`, `T.Deadline` must not be called (`go doc testing/synctest.Test`) |
| `runtime.ReadMemStats` | Go 1.27.4 | Live heap sampling in the dump benchmark | Already used at `internal/app/persist/bench_test.go:80-84` (`HeapAlloc`) |
| `runtime/debug.ReadBuildInfo` | Go 1.27.4 | Hardware block: Go version, module version, VCS revision/time | Stdlib; no new dependency |
| `runtime`, `os`, `syscall` (tests only) | Go 1.27.4 | CPU/RAM/load/disk facts, current RSS, CPU time if needed | `internal/app` test files are exempt from the no-I/O depguard rule (`!$test` at `.golangci.yml:68-79`); `internal/testutil` is a test-only-imported normal package |
| `/proc/self/io` (`write_bytes`), `/proc/self/statm`, `/proc/self/status`, `/proc/loadavg`, `/proc/meminfo`, `/proc/cpuinfo`, `/proc/self/mountinfo`, `/proc/sys/net/ipv4/ip_local_port_range` | Linux kernel | Amplification, RSS, load, hardware block, port-window assumption | The stand is already `//go:build linux` (`internal/stand/stand.go:1`); `write_bytes` is "the number of bytes really sent to the storage layer" `[CITED: man7.org/linux/man-pages/man5/proc_pid_io.5.html]` |
| `os.Stat` / `FileInfo.Size` | Go 1.27.4 | Dump file size per generation | No parsing; the dump file is the artifact |
| `os.ErrProcessDone` | Go 1.27.4 | `killAndReap` already-exited diagnosis | `var ErrProcessDone = errors.New("os: process already finished")` (`go doc os.ErrProcessDone`) |
| `syscall.Getrusage` | Go 1.27.4 (Unix) | CPU-time measurement alternative if a paired ratio must remain | `Rusage.Utime/Stime`; note `Maxrss` is peak RSS, not a current-RSS source (Phase 7 replaced the metric's peak use) |

### Supporting (existing in-repo)

| Asset | Location | Purpose |
|-------|----------|---------|
| Stand harness | `internal/stand/stand.go` | Real binary lifecycle, ports, waits, clients, logs |
| Stand metrics helpers | `internal/stand/stand_red_test.go:194-246` | Exact-value `/metrics` reads with not-yet semantics |
| Open-loop load generator | `internal/stand/stand_red_test.go:470-515` | The reviewer-shaped 1 024-worker/8-connection schedule |
| Percentile reporter | `internal/testutil/bench.go:27-44` | p50/p99 under `p50-ms`/`p99-ms` |
| Dump-adapter fs seam | `internal/persistfile/fsops.go:101-140,346-369`; `internal/persistfile/dumpfile.go:107-111` | Real 64 KiB buffered write, fsync, rotate, rename, dir fsync |
| Metrics exporter | `internal/metrics/metrics.go:446-500,502-560` | `supkv_dumps_*`, counters, `process_resident_memory_bytes` |

### Alternatives Considered

| Instead of | Could use | Tradeoff / decision |
|------------|-----------|---------------------|
| Operation-count gates | Wall-clock ratio with `-p 1` / a serialized job | Wall clock is exactly the flake the phase must remove; `-p 1` cannot serialize packages inside the required `make test` target without slowing every run, and it still measures a shared host |
| Operation-count gates | CPU-time ratio via `syscall.Getrusage` | CPU time is immune to other processes' load but still not deterministic and adds platform code; keep as a fallback only for the wall-clock paired benchmark in `make perf` |
| Go const threshold table in `internal/testutil` | Env vars set by the Makefile | Two sources of truth (Makefile + test defaults) and a parsing layer; consts are compiled, reviewable and lint-clean |
| Go const threshold table | A YAML/JSON thresholds file | Adds a parser and a new config format for 10 numbers; D-06 says bounds, not stored profiles |
| `benchstat` | `go tool`/module dependency | New tool directive drags a dependency graph; the repo deliberately keeps tools minimal |
| `gopsutil`/`prometheus/client_golang` | `/proc` + existing exporter | Breaks the pinned runtime graph and the metrics decision (in-repo text exporter) |
| External load generator (`hey`, custom binary) | Reuse `redRunBurst` | The stand already has the exact reviewer-shaped generator; a second one is the "second harness" the phase forbids |
| `fio`/`iostat` | `/proc/<pid>/io` + `os.Stat` + `supkv_dumps_total` | Process-submitted bytes answer the amplification question; device-level journal/COW effects are a documented limitation, not a reason for a new tool |

**Installation:** none. No `go get`, no `go.mod` change.

## Package Legitimacy Audit

**Not applicable — this phase installs no external package.** Every
recommendation is Go standard library or in-repo code. The runtime require set
stays `google.golang.org/grpc` + `google.golang.org/protobuf`
(`go.mod:15-18`), and the tool directives are unchanged (`go.mod:5-13`). No
registry lookup or slopsquat check is required.

## Don't Hand-Roll

| Problem | Don't build | Use instead | Why |
|---------|-------------|-------------|-----|
| p50/p99 aggregation | A second percentile implementation | `internal/testutil.ReportLatencyPercentiles` (`internal/testutil/bench.go:27-44`) | Shared by all benchmark packages; nearest-rank semantics already pinned |
| Real-binary scenario driving | A new process/socket harness | `internal/stand` (`stand.go:381-525,548-700`) | Build-once, reap, signal, log capture, metrics scrape, wait clamps already hardened across phases 2-7 |
| Open-loop write load | A new load generator | `redRunBurst` (`internal/stand/stand_red_test.go:470-515`) | Reviewer-shaped 1 024 workers × 8 connections, schedule-based, error census; extend it with latency samples |
| Virtual-time goroutine synchronization | Sleep/settle loops | `testing/synctest` (`synctest.Test`/`Wait`) | Already the repo's deterministic idiom; deadlock detection replaces a test-binary timeout |
| Port allocation | A per-test `:0` picker | One private-window allocator in `internal/testutil` | Removes the pick-then-bind class structurally; one registry, disjoint package bands |
| Hardware facts | A new metrics/agent library | `runtime` + `debug.ReadBuildInfo` + `/proc` reads | Stdlib; no dependency; Linux is the only perf target |
| Amplification accounting | A block-layer tracer | `supkv_dumps_total` delta × dump file size, cross-checked with `/proc/<pid>/io` `write_bytes` | The binary already counts dumps; the dump file is the payload |
| Memory measurement | A heap profiler parser | `runtime.ReadMemStats` + `process_resident_memory_bytes`/`/proc/<pid>/status` | The phase-7 metric is already the honest current-RSS probe |
| Threshold storage | A config-file format/flag surface | A const table in `internal/testutil` | One place, compiled, lint-visible; D-06 says bounds, not profiles |

**Key insight:** the repo already contains every measurement primitive Phase 8
needs. The work is wiring, one code fix (the per-key pending index), two code
cleanups (dumpNotify harness, replay double copy), and one structural test-infra
helper (ports). Building a second harness or a second metrics stack would be the
main way to make this phase large and fragile.

## Recommended Design per Requirement

### PERF-01 — The write-latency model for both profiles

**The model, with exact knobs.**

| Quantity | Default profile | Fast-response profile | Source |
|----------|-----------------|-----------------------|--------|
| `dump_debounce` | `50 * time.Millisecond` | 1 ms (benchmark/operator setting) | `internal/config/config.go:107`; fast setting `internal/grpcapi/bench_test.go:85` |
| `dump_max_wait` | `time.Second` | 50 ms | `internal/config/config.go:108`; `internal/grpcapi/bench_test.go:86` |
| Sustained-flow dump start | `oldest_unconfirmed + MaxWait` | `oldest_unconfirmed + MaxWait` | `internal/app/persist/scheduler.go:191-198` (`nextDebounce`/`deadline` arming) |
| Latency under sustained flow | `U(0, MaxWait) + dump_time`; p50 ≈ MaxWait/2 + dump, p99 ≈ MaxWait + dump | same shape with 50 ms | Model + review measurements |
| Sequential single client | `Debounce + dump_time` per write; ceiling `≈ 1/(Debounce+dump_time)` ≈ 14 writes/s | `1 ms + dump_time` | `internal/app/persist/scheduler.go:194-198`; review 58-70 ms floor |
| Disk cost | dumps at `≈ 1/MaxWait` under continuous flow × full dump bytes | dumps at `≈ 1/MaxWait` = 20/s × full dump bytes | `runDump` always rewrites the whole state (`scheduler.go:228-249`) |

The default-profile cadence is already proven deterministically:
`TestEngine_burstBoundedConfirmation` writes 1 000 records at 1 ms virtual
cadence (`burstWrites = 1000`), and asserts the first dump is at exactly 1 s,
at most 3 dumps total, and every write confirmed by then
(`internal/app/persist/engine_test.go:644-693`, constants at `:322-324`). That
test is the model's executable half; Phase 8 adds the *measured* half.

**Doc conditions for the "~10 ms under 1000 writes/s" claim.** The number comes
from a fast-profile benchmark whose dump is an in-memory fake, so it excludes
real disk/dump time: `Debounce: 1 ms`, `MaxWait: 50 ms`
(`internal/grpcapi/bench_test.go:84-88`), two real loopback UDP adapters, a
bufconn gRPC client, a ~1 ms background burst (`benchBurstPace`, `:45`), one
`Set(WaitForReplication: true)` per iteration, and an in-memory dump
(`fakeDumpStore.SaveStream`, `internal/grpcapi/server_test.go:92-110`). This
session measured on the research host: 11.53 ms mean, p50 6.10 ms, p99 44.93 ms,
max 68.0 ms. The doc must state exactly these conditions and must not state
"~10 ms" standalone. On a **real** dump at 10 000 × 1 KiB the encode+save is
40-70 ms in the review (113 ms mean on this loaded host in the 2 s run), so the
fast profile's 50 ms `MaxWait` is a floor, not a ceiling: real fast-profile
latency is roughly `max(MaxWait, dump_time)` plus wait queueing. The ~10 ms
number is an in-memory-dump benchmark result and must be labelled as such.
The default profile's published conditions: 10 000 keys × 1 KiB, open-loop
1 000 writes/s, default dump knobs, real `persistfile` dump, race off; review
measured p50 0.528-0.560 s / p99 1.019-1.054 s
(`docs/reviews/2026-11-19-v1.0/REVIEW.md:121`).

**Implementation.**
- Extend `BenchmarkConfirmationLatency` into a profile table
  (`default`: 50 ms/1 s; `fast`: 1 ms/50 ms) so decision 14.3's "benchmarks for
  both profiles" is satisfied; the fast arm keeps the current body, the default
  arm uses a real-disk stand test instead of the in-memory fake where the dump
  time matters.
- Add `TestPerfDefaultProfileLatency` in `internal/stand/perf_test.go`: start
  both nodes first (so the first-seen listing is the empty startup one, not a
  10 000-key paced round inside the window), pin `--sync-interval` long (e.g.
  `10m`; state the condition), preload 10 000 × 1 KiB, wait until no dump is in
  flight and the unacked gauge is 0, then run an open-loop 1 000 writes/s × 15 s
  over the hot-key set; measure scheduled-to-completion latency
  (coordinated-omission-free), assert p50/p99 starting bounds, log the profile.
  Set `p.dumpA`/`p.dumpB` under `SUPKV_PERF_DUMP_DIR` (the harness default is
  `t.TempDir()`), so the dump time is measured on a real filesystem.
- Add `TestPerfFastProfileLatency` in `internal/grpcapi` wrapping the fast
  benchmark via `testing.Benchmark` and asserting `Extra["p50-ms"]`,
  `Extra["p99-ms"]` and `Extra["latency-ns/op"]`.
- Document the sequential-client ceiling from the same knobs; optionally add a
  single-client perf measurement that asserts the achieved rate is in the
  documented band (≥8/s and ≤20/s at 1 KiB/10 k state), which is a policy
  characteristic, not a performance regression gate.

**Acceptance probe:** `make perf` runs both profile tests; the doc carries a
table where each number has profile + keys + value size + rate + media + race
state; `TestPerformanceDoc` anchors the threshold strings.

### PERF-02 — The 1 000 writes/s burst over distinct keys

The scenario already exists and is green: `TestRed_R02_burst`
(`internal/stand/stand_red_test.go:537-600`) drives `redRunBurst` (open-loop
schedule, 1 024 workers, 8 connections, 10 s RPC deadline, 1 000/s × 15 s over
10 000 distinct keys), asserts `achieved ≥ redBurstMinAchieved = 990` and the
unacked gauge reaches 0 within `redBurstDrainBound = 5 s` after the send window
(constants `:91-116`). `scripts/red-expectations.tsv` already lists it `green`.
It is gated by `SUPKV_RED_TESTS=1`, so it runs nightly in the red-run job, not
in the PR gate.

**No-duplication design:** extract the scenario body (from `p := newStandPair(t)`
to the final assertions) into a shared helper in the stand package, then:
- keep `TestRed_R02_burst` as the red-contract row (no change to the table);
- add `TestPerfBurstDistinctKeys` (env `SUPKV_PERF=1`) calling the same helper,
  so `make perf` runs the identical scenario with the same bounds and reports
  the achieved rate. The two names serve two contracts (nightly red evidence vs
  operator perf target) with one implementation.
- Docs: the guarantee/non-guarantee text (below).
- `docs/SPEC-DEVIATIONS.md:210` S-FAIL-15: update the text to drop "holds only
  on hot keys today" and cite the shipped test and the documented bounds. Phase
  9's registry resweep owns unrelated rows; this row is this phase's closure.

Guarantee at the measured rate (1 KiB values, two healthy nodes, default
buffers, distinct keys, open loop): ≥990 accepted writes/s over 15 s; each
accepted write is replicated (one immediate first datagram, paced retransmits);
the unacked set empties within 5 s of the send window — and because an `Ack` is
emitted only after the receiver applied the record, an empty unacked set also
proves peer receipt, so "no loss of confirmed writes" is observable from the
same gauge; the dump cadence stays at most one dump per `dump_max_wait` under
sustained flow (one datagram per write, not one dump per write). Not guaranteed: 60 KiB values (IP fragmentation),
rates above 1 000/s, an absent/unresponsive peer (the pending set grows; the
write stays dump-confirmed but the wait report degrades), lossy/WAN segments, or
client latency below `dump_max_wait` under sustained flow.

**Interpretation note for the verifier:** "standing suite" is read here as the
stand package's routine coverage — the scenario already runs nightly in the
red-run job (`scheduled.yml:80-110`, table row `green`) and the new perf row
makes it operator-runnable through `make perf`. If the verifier requires the PR
gate itself to run it, drop the env gate for this one test; that puts a
wall-clock 990/s achieved-rate bound into every `go test -race ./...` run
(~16-20 s plus setup) and reintroduces exactly the shared-runner class this
phase removes. The plan should state the chosen reading explicitly.

**Acceptance probe:** the perf row passes; the doc section carries both lists;
S-FAIL-15 points at the test.

### PERF-03 — Disk-write amplification, measured or the gap recorded

**Method (process-submitted bytes, target media when available):**
1. Run a node with default dump knobs and metrics enabled, with its dump
   directory on a **real filesystem** — `SUPKV_PERF_DUMP_DIR`, defaulting to
   `$(CURDIR)/.tmp/perf` (gitignored) under `make perf`; `/tmp` is frequently
   tmpfs and would measure memory, not disk. Verify the fs type in the test and
   name it in the block. Preload a fixed state (e.g. 10 000 × 1 KiB) so the
   state size is known and constant. Start the peer first (empty first-seen
   listing), pin `--sync-interval` long, and wait for the preload's last dump
   to complete before taking the baseline, so the window contains only the
   steady-state dumps.
2. Record a baseline over a window: `supkv_dumps_total`, the main dump file
   size (`os.Stat`), `/proc/<pid>/io` `write_bytes`, and the logical bytes the
   clients wrote (`writes × value bytes`). Use the `dumps_total` delta only
   while `supkv_dumps_errors_total` is unchanged (a failed attempt writes no
   generation).
3. Drive hot-key writes for a fixed window (e.g. 1 000/s × 15 s). Record the
   same values.
4. Compute `amplification = (dumpsΔ × dumpBytes) / logicalBytes`, and compare
   `write_bytesΔ` with `dumpsΔ × dumpBytes`. The expected relation from the
   code: each `runDump` takes one `store.Records()` copy and streams one new
   generation through `SaveStream` (`internal/app/persist/scheduler.go:228-249`;
   `internal/persistfile/dumpfile.go:107-111`; `internal/persistfile/fsops.go:101-140`),
   so bytes submitted per dump ≈ the dump file size (rotation renames, it does
   not rewrite the old generation). Both nodes dump — B applies replicated
   records and marks dirty too — so measure the named node's `/proc/<pid>/io`
   and state whether the pair shares a filesystem; the media total is the sum
   of both processes' submissions.
5. Record media facts: `findmnt`/`/proc/self/mountinfo` fs type, device,
   rotational flag, and whether the media is target-grade (the two-board
   appliance's eMMC/SD is the target; a workstation NVMe is not).

Basis from the review: 10 writes/s closed loop rewrote a 15 MB state at
~155 MiB/s with 10.9 dumps/s → ~14.2 MB per dump (0.95× the 15 MB file); at
`dump_max_wait` the same state costs ~14 MiB/s
(`docs/reviews/2026-11-19-v1.0/REVIEW.md:97,123`). This is a policy
consequence, not a defect; the phase publishes the number and the method. On
this host the media is an NVMe SSD (`APPLE SSD AP1024N`, ext4/LVM), which is
**not** the target-grade media; D-03 therefore says publish the method and
record the gap against Spec open question #2 rather than claim target numbers.
If a target device is ever attached, rerun the same test and replace the gap
with the measured row (same hardware-block format).

**Implementation:** `TestPerfAmplification` in `internal/stand/perf_test.go`,
reusing the preload/loadgen machinery and `redMetricsBody`/`redMetricValue`.
**Acceptance probe:** the test logs dumps, dump bytes, logical bytes,
`write_bytes`, amplification and media; the doc has either the measured row
with hardware or the explicit gap + method + missing hardware. A sanity gate
`write_bytesΔ/dumpBytesΔ ≤ 1.10` (starting value; 10 % headroom for stat
granularity and filesystem bookkeeping) catches an accidental double-write of
the generation.

### PERF-04 — The memory model from stand measurements

**The as-built path (what the model must describe):** one `store.Records()`
deep copy, then streamed JSONL encoding through the adapter's 64 KiB
`bufio.Writer` — the encoded generation is never materialized as a second
buffer (`internal/app/persist/scheduler.go:228-249`;
`internal/app/persist/codec.go:138-260`; `internal/persistfile/fsops.go:11-16,101-140`).
`docs/OPERATIONS.md:190-215` already states this model qualitatively ("one state
copy plus one record line"; "roughly 1.4 times the state size in total
allocations" — value-dominated states; see the pitfall below about
key-dominated states).

**Formula + measured points to publish:**
```
logical_state   = Σ (len(key) + len(value)) over live records
dump_bytes      ≈ 1.37 × Σ len(value) + JSON/key overhead      (base64 4/3 + framing)
live_heap_peak  ≈ logical_state + N×record_overhead + 64 KiB + max_line(~1.34×value_max)
alloc_churn/cycle ≈ Records copy + per-record line garbage (value-dominated ≈ 1.4× state;
                    key-dominated is higher because per-record map/key overhead dominates)
RSS             ≈ baseline + live heap + GC headroom (GOGC=100 by default)
```
Points to measure (keep the run under ~1 min): (1 000 keys × 1 KiB),
(10 000 × 1 KiB), (1 000 × 60 KiB). For each: preload, wait for a covering dump,
read `process_resident_memory_bytes` (current RSS on Linux,
`internal/metrics/metrics.go:539-540`; `internal/metrics/rss_linux.go:21-39`) and
`/proc/<pid>/status` `VmRSS`/`VmHWM`; sample VmRSS at ~2 ms around one dump for
the transient. The precise transient is the dump benchmark's `peak-live-B`
(`internal/app/persist/bench_test.go:92-100,117-134,140`); this session measured
10.4-17.5 MB at 10 000 mixed small keys with a 5.1 MB dump, i.e. key-dominated
overhead, which is a useful contrast point for the doc.

**Implementation:** `TestPerfMemoryPoints` in `internal/stand/perf_test.go`
(table-driven, one fresh node per point or one node with restarts between
points to avoid state bleed). Optionally add
`BenchmarkDumpEnvelopeValues` (1 000 × 60 KiB) and a deterministic gate
`peak-live ≤ 2 × logical_state` (PERSIST-06 acceptance was ≤1.3× live at
10 k × 60 KiB; 2× leaves headroom) — this is the only memory gate worth having,
because it catches a return to encode-into-bytes (which would be ≈4-5×).
**Acceptance probe:** the doc table lists state → dump bytes → RSS → peak, with
the hardware block; the test logs the same points.

### PERF-05 — Absent peer, growing unconfirmed set, and 04 New-3

**Does write latency still degrade? Yes, structurally, on the write path.**
XPORT-02 removed the adapter's per-event `mirrorState`/`PendingSnapshot`
whole-set copies (03-06 deleted them; `internal/udprepl/mirror.go:212-267` is
now O(1) per event). But the core still does a whole-pending scan on every local
write:

```
trackLocalWrite
  ├─ trackedVersion(key)          for pk := range c.pending { ... }   core.go:3429-3437
  └─ removeSupersededLocal(...)   for pk := range c.pending { ... }   core.go:3444-3452
applyRemote (per inbound DataReplication)
  └─ for pk := range c.pending { if pk.key != msg.Key { continue } }  core.go:1659
```

At 10 000 pending records that is two full map scans per local write (plus one
per *applied* inbound replication; the stale/strictly-older early returns skip
the loop). The adapter's `notify` queue is 1 024 deep
(`internal/udprepl/loop.go:24-29`), and the gRPC handler calls
`notifyLocalWrite` synchronously after the engine write
(`internal/grpcapi/server.go:234`), so a saturated event loop back-pressures the
write RPC path — the same back-pressure mechanism that produced the review's
absent-peer 460 ms vs live 76 ms measurement
(`docs/reviews/2026-11-19-v1.0/REVIEW.md:125`); the remaining scans are smaller
than the deleted `mirrorState` but still scale with the pending set. This is
03-REVIEW IN-02, still `open` with "Note for PERF/Phase 8"
(`.planning/phases/03-transport/03-REVIEW-DISPOSITION.md:22`); the existing
`BenchmarkCoreEventLoop` cannot see it because it measures only an idle
`Advance` at 10 vs 10 000 (`internal/app/repl/bench_test.go:240-269`).

**Fix (the same fix XPORT-02 needs finished): a per-key pending index.**
- Add `keyPending map[string]pendingKey` beside `pending`/`peerUnacked`, or the
  review's suggested `map[string]map[pendingKey]struct{}` if the planner prefers
  resilience over the one-pair-per-key invariant. The invariant is real: the
  guard rejects not-greater versions and `removeSupersededLocal` removes every
  smaller pair before insert, so at most one pair per key exists.
- Maintain it in `trackLocalWrite` (insert) and `removePending` (delete); touch
  nothing else. `evictPeers` intentionally does not remove pending entries
  (suspension), so no index maintenance there.
- Rewrite `trackedVersion`, `removeSupersededLocal` and `applyRemote`'s
  displacement loop to read only that key's index entry. The discovery flush in
  `handleHello` (`core.go:1503-1516`) stays O(n): it must emit every pending
  record once to a newly seen peer, and that is a one-time event.
- Add `PendingVisits() uint64` (a plain counter incremented in every loop over
  `c.pending`/the index; no atomics — driver-goroutine-only like the existing
  counters). This is the deterministic seam the gates read.

**Benchmark and gate.**
- `BenchmarkNotifyWriteCost/{live,absent}` in `internal/app/repl/bench_test.go`:
  - absent arm: empty peer table, 10 000 pending 1 KiB records pre-seeded,
    then one `NotifyLocalWrite` per iteration with a fresh key (the pending set
    grows during the run);
  - live arm: one known peer that Acks each write immediately, pending stays
    small;
  - report `pending-visits/op` (via `b.ReportMetric`) and ns/op; state the
    unacked size, peer state and value size in the benchmark comment.
- `TestCore_notifyWriteCostFlat` / `TestCore_applyRemoteCostFlat`: run a fixed
  K (e.g. 1 000) of the same operations at 10 vs 10 000 pending, always with a
  fresh key (so the index has nothing to inspect), and assert `PendingVisits`
  per op is exactly 0 at both sizes (a repeated-key write may inspect ≤2; the
  gate's fixture uses fresh keys). Today the same loop visits 2×N and N
  respectively — the gate is RED before the index. No timing.
- `TestPerfAbsentPeerWriteCost` (in `internal/app/repl`, beside the benchmark):
  the paired wall-clock ratio ≤ 2.0, run only by `make perf` on a calm host,
  **reporting** both arms and failing by name. This is the D-05 “proven by a
  benchmark” half; the deterministic visits gate above is the always-on half.
- Docs: state the conditions and the post-fix expectation (event-loop cost
  independent of the pending set; a write’s waiting behavior when the peer is
  absent is the A-01 immediate `(false,0)` report, not a stuck RPC — but the
  notification path itself must stay O(1)).

**04 New-3 — the double copy (exact path).**
`persist.replaySnapshotKey` puts a cloned `Data` in `RestoredRecord`
(`internal/app/persist/snapshot.go:216-227`), the gRPC layer clones it again
when handing it to the replay seam (`internal/grpcapi/server.go:412-421`), and
the adapter clones it a third time via `cloneLocalRecord`
(`internal/udprepl/loop.go:254-267`). The core ignores `Data` entirely:
`Core.NotifyReplayedWrite` only passes `rec` to `trackLocalWrite`, which reads
`rec.Version` and nothing else (`internal/app/repl/core.go:2350-2361,2370-2442`);
emission re-reads the record with `Lookup` at drain time
(`core.go:3032-3064`). The minimal fix is to stop copying on the two hops after
the report:
- `server.notifyReplayedWrite`: pass `storage.Record{Version: rec.Version}`
  (no `bytes.Clone`, no checksum recomputation — both are unused on this path);
- `adapter.NotifyReplayedWrite`: enqueue
  `localWrite{key: key, rec: storage.Record{Version: rec.Version}, replay: true}`
  — no `cloneLocalRecord`;
- update the `LocalReplayNotify` doc (`internal/grpcapi/server.go:53-60`) to say
  the record carries only the version, and the adapter seam doc
  (`internal/udprepl/loop.go:248-259`).
- Add an allocation gate: `BenchmarkReplayNotify` with a 60 KiB record reports
  bytes/op well below 60 KiB (the two clones cost ~2.4 GB transient on the
  review's 20 000 × 61 440-byte load); a `TestReplayNotify_noDataCopy` can
  assert `testing.Benchmark(...).AllocedBytesPerOp() < 1024` for one 60 KiB
  record. The remaining `RestoredRecord.Data` clone is the report's own payload;
  New-3 counts the two discard-path copies.

**Acceptance probe:** the four count gates are green in `make test`; the paired
benchmark prints both arms and a ratio; the replay allocation gate passes; the
docs state the measured result and the 2.4 GB transient is gone.

### PERF-06 — `make perf`, thresholds in one place, hardware block

**Reading of D-06.** "Runs the stand suite with thresholds" is implemented as
the perf rows *over the stand harness* (the thresholds run on the real-binary
suite, not a new harness); the functional stand suite keeps running in
`make test` and the nightly job. If the verifier reads it literally as the whole
functional suite too, add one `go test -count=1 ./internal/stand/` line to
`perf`; it costs ~7 s at the default scale (the Makefile's measured 6.53 s
baseline, `Makefile:57-60`) and is harmless.

**Target shape (one new Makefile target, CI unchanged):**

```make
## perf: operator-facing threshold run over the stand harness and the
## benchmark wrappers (the perf ROWS, not the whole functional stand suite).
## Linux only (the stand is linux-only); no -race (the numbers are the
## product's, not the race build's); packages serialized so one measurement
## cannot contaminate the next. Thresholds live in internal/testutil/perf.go;
## the hardware block is logged by the tests. CI deliberately keeps the
## threshold-free `make bench-smoke` (D-13/D-14).
perf:
	mkdir -p .tmp/perf
	SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(CURDIR)/.tmp/perf \
	  go test -count=1 -p 1 -timeout 30m -v -run '^TestPerf' \
	  ./internal/stand/ ./internal/app/repl/ ./internal/grpcapi/
```

- Gate: each `TestPerf*` starts with `if !testutil.PerfEnabled() { t.Skip("perf: run make perf") }`
  and `if testutil.RaceEnabled() { t.Skip("perf: numbers are measured without -race; run make perf") }`.
  This mirrors the established `SUPKV_RED_TESTS` pattern (red tests skip without
  the env; `Makefile:69-78`), so `make test` stays threshold-free and green.
- Dump directory: `testutil.PerfDumpDir(t)` returns a fresh per-test subdirectory
  under `$SUPKV_PERF_DUMP_DIR` when set (cleaned via `t.Cleanup`), else
  `t.TempDir()`, and always records the fs type in the block.
  `TestPerfAmplification` fails if the resolved fs is tmpfs/ramfs, naming
  `SUPKV_PERF_DUMP_DIR` — a memory-backed dump is a memory measurement, not a
  disk one. Latency/memory tests may run on tmpfs but must log it.
- Thresholds: one const block in `internal/testutil/perf.go` (or a function
  returning a table if a `gochecknoglobals`-clean slice is needed), e.g.
  `PerfDefaultP50`, `PerfDefaultP99`, `PerfFastMean`, `PerfFastP99`,
  `PerfBurstMinRate`, `PerfBurstDrain`, `PerfBytesPerDumpRatioMax`,
  `PerfAbsentPeerRatioMax`. No env parsing, no second defaults. Tests fail with
  the PERF ID and the threshold value in the message.
- Hardware block: `testutil.HardwareBlock()` logs CPU model (`/proc/cpuinfo`),
  logical CPUs (`runtime.NumCPU`), RAM (`/proc/meminfo MemTotal`), load
  (`/proc/loadavg`), GOOS/GOARCH, Go version, module/VCS revision
  (`debug.ReadBuildInfo`), race on/off (build-tag pair, verified: `-race` sets
  the `race` build tag by local probe), GOMAXPROCS, dump path + fs type
  (`/proc/self/mountinfo`). `make perf` runs `-v`, so the block is printed;
  each published doc table carries a copy plus the command.
- Docs: new `docs/PERFORMANCE.md` (recommended over a section of OPERATIONS,
  because it is a model + measurement ledger with multiple tables and
  commands). `docs/OPERATIONS.md:190-215` keeps a short memory-model summary and
  links to it. `docs/API.md:315-330` says the write-latency model is Phase 9's;
  Phase 9's DOCS-05 should link to `docs/PERFORMANCE.md` rather than duplicate
  it — record that in the phase summary for Phase 9.
- Drift guard: a `TestPerformanceDoc` in `cmd/sup-kv` (the `TestOperationsDoc`
  shape, `cmd/sup-kv/operations_test.go:48-171`) asserts the doc exists, carries
  the threshold strings, the profile conditions, and the hardware-block
  headings. This keeps "one place" honest when a threshold changes.
- Darwin: `make perf` fails fast with a named message (`internal/stand` is
  Linux-only; darwin is development-only per 14.10). Document that the perf
  target is Linux.
- CI: no change. Optionally mention `make perf` in the release procedure later
  (Phase 9); D-06 says the threshold-free smoke stays the CI contract.
- Bench-target completeness: add `./internal/app/repl/` to `make bench` and
  `make bench-smoke` (or add a `bench-core` target) so the Phase-3/4 core
  benchmarks are not orphaned; they are currently absent from both targets
  (`Makefile:32-41`). With the per-key index the 10 000-entry fixture seeding
  stops being O(n²), so the smoke run stays cheap. Update the Makefile comment
  and the scheduled bench-smoke job stays target-driven (`scheduled.yml:66-78`).

**Threshold starting values (planner picks the final values; all are starting
values with a stated basis, not locked decisions):**

| ID | Metric | Starting bound | Basis |
|----|--------|----------------|-------|
| PERF-01 | default-profile p50 / p99 at 10 k × 1 KiB, 1 000/s, 15 s | ≤ 750 ms / ≤ 1 500 ms | review 0.528-0.560 / 1.019-1.054 s; +40 % headroom; model mean MaxWait/2 + dump |
| PERF-01 | fast-profile mean / p99 (the benchmark's conditions) | ≤ 25 ms / ≤ 100 ms | measured this session 11.53 / 44.93 ms; 2.2× / 2× headroom; p99 also ≈ 2 × MaxWait |
| PERF-01 | sequential-client rate at 10 k × 1 KiB | 8-20 writes/s (documented, not a failure gate) | 1/(50 ms + 40-70 ms) ≈ 10-14/s; review 14/s |
| PERF-02 | achieved rate / drain | ≥ 990 writes/s; unacked = 0 ≤ 5 s after send window | REQUIREMENTS PERF-02; existing test constants `stand_red_test.go:91-116` |
| PERF-03 | bytes submitted per dump | ≤ 1.10 × dump file size; report amplification | each dump writes one generation; review 14.2 MB/dump vs 15 MB file |
| PERF-04 | RSS at 1 k × 60 KiB (tripwire) | ≤ 12 × logical_state + 256 MiB | review 5.7-9× at 60 MB state (old design); loose tripwire only |
| PERF-04 | envelope-value dump peak-live | ≤ 2 × logical_state (optional benchmark gate) | PERSIST-06 acceptance ≤1.3× live; 2× headroom |
| PERF-05 | absent/live write-notification ns/op | ratio ≤ 2.0 | D-05 / XPORT-02 acceptance; deterministic visits gate is the always-on proof |
| PERF-06 | all threshold values | one const block in `internal/testutil/perf.go` | D-06 “defined in one place” |

### Flake classes routed to this phase

Each row below names the mechanism found in the code and the deterministic
cure; none relaxes a bound.

#### F-1. `TestCore_keyCountCostFlat` — wall-clock ratio plus `t.Parallel`

**Mechanism.** The gate measures ns/op at 1 000 and 50 000 keys with
`testing.Benchmark` (`internal/app/repl/bench_test.go:171-205`) and carries
`t.Parallel()` at `:172`; the sibling gate is deliberately sequential because
"a parallel sibling benchmark would contaminate the measurement"
(`:271-279`). Full-suite runs on this 16-core host have been recorded at load
9-13, with ratios 2.80/2.91/3.54/5.65/6.0 against the 2.0 bound; the same tree
passes standalone at 0.87-1.05 (`.planning/phases/04-protocol/deferred-items.md`,
`05-persistence/deferred-items.md`, `07-observability-and-configuration/deferred-items.md`).
The counters are otherwise genuinely O(1): the `StateStore` interface removed
the whole-store read precisely so "the O(store)-per-datagram cost
[is] compile-impossible (XPORT-06)" (`internal/app/repl/core.go:393-400`).

**Cure (deterministic operation count).** Wrap the fixture's `stateStore` in a
counting double, run a fixed K-iteration loop of the same handler body, and
assert the per-datagram store-call counts are identical at both sizes and match
the expected shape:
- `DataRequest`: `Version` 1, `Lookup` 1, `Versions` 0, `Records` 0, `Observe` 0;
- `DataSync`: `Observe` 1, `Version` len(entries)=16, `Lookup` 0, `Versions` 0.

Snapshot the counters **after** fixture construction and measure deltas: the
fixture's startup Hello triggers the one targeted listing build, which calls
`Versions()`/`Generation()` legitimately (`benchFixture`, `internal/app/repl/bench_test.go:78-85`),
and those calls must not be attributed to the measured datagram. The expected
exact counts are what catch a per-datagram `Versions()` call (which is O(store)
but constant); the cross-size equality catches a per-key loop. No timing, no
`t.Parallel` hazard; keep `t.Parallel` for suite speed.
Also fix the fixture's version domain while touching it: the current
`version.PackVersion(testNowMs, uint16(i), localID)` packs counters up to 49 999,
outside `PackVersion`'s documented 12-bit counter domain (IN-04,
`.planning/phases/03-transport/03-REVIEW.md:211-215`); use
`testNowMs+int64(i)` with counter 0, as `benchUnackedFixture` already does.

**Files:** `internal/app/repl/bench_test.go` (fixture + gate),
`internal/app/repl/retransmit_test.go:69-104` (counting wrapper),
`internal/app/repl/config_internal_test.go` only if the fixture moves.
**Acceptance probe:** the gate runs in <1 s, is green under any host load, and
fails by name when a handler starts reading the whole store.

#### F-2. `TestCore_unackedCostFlat` — wall-clock ratio on the idle loop

**Mechanism.** `testing.Benchmark` ns/op at 10 vs 10 000 pending
(`internal/app/repl/bench_test.go:280-309`); full-suite ratios 1.14 standalone
vs 2.86/3.45/3.60/4.30 under recorded load (`07-observability-and-configuration/deferred-items.md`).
The measured path is `Advance` + `UnackedStats` + `PeerCount`; all are O(1) by
construction (`scanRetransmit` pops only due heap items: `core.go:2964-2975`;
`popDueRetries` starts at the heap root: `core.go:3524-3532`; `UnackedStats`
reads `len(c.pending)` and the list front: `core.go:3346-3362`).

**Cure.** No existing counter measures this work: `Retransmissions()` counts
retry emissions, `PacedSends()` counts datagram handoffs, and `UnackedStats()`
counts entries — all emissions or lengths, never set scans. Add `PendingVisits()`
to `Core` (plain `uint64`, incremented in every loop over the pending set —
`scanRetransmit`'s due loop, `trackedVersion`, `removeSupersededLocal`,
`applyRemote`'s displacement loop, and the discovery flush). `TestCore_unackedCostFlat` runs K=1 000 idle iterations at 10 and
10 000 and asserts `PendingVisits/op == 0` at both sizes (and store calls 0).
The claim is now the literal one: an idle event-loop iteration inspects zero
pending entries. The ns/op benchmark stays for `make bench`; it is no longer a
gate. This makes the test fast and load-independent.

**Files:** `internal/app/repl/core.go` (counter increments + accessor),
`internal/app/repl/bench_test.go`.
**Acceptance probe:** gate <1 s, green at load 12, red if any idle-path loop over
the pending set is reintroduced and instrumented.

#### F-3. `TestScenario_dumpNotify` — wall-clock settling in the virtual-time harness

**Mechanism.** `simEngine.settle` waits for six 200 µs quiet rounds and
`awaitActivity` waits for "any seam event beyond base"
(`internal/app/repl/sim_dump_test.go:248-284`); `writeAndConfirm` then advances
virtual time to `start+Debounce` (`:296-312`). `virtualAfter` bumps activity for
*every* armed timer, including the confirmer's `ConfirmTimeout` arm
(`:96-104`), so `awaitActivity` can return before the scheduler has consumed the
dirty event and armed the debounce; under load the 1.2 ms quiet window can also
expire before the engine goroutine is scheduled. That is the recorded *analysis*
(not a verified root cause) in
`.planning/phases/05-persistence/deferred-items.md`; the observed failure mode
is a blocked `<-setRet` with the scheduler select idle, which is enough to make
the synctest cure correct regardless of which of the two races fired.

**Cure (deterministic sequencing).** Convert the harness to a `testing/synctest`
bubble: start the engine goroutine inside the bubble, replace `settle`,
`awaitActivity` and the quiet-round constants with `synctest.Wait()`, and drive
virtual time with the existing `sched.RunUntil`. `synctest.Wait` blocks until
every other goroutine in the bubble is durably blocked — the exact "engine has
consumed the nudge and re-armed" proof the sleeps approximate. Remove the
subtests' `t.Parallel` calls (inside a bubble `T.Parallel`/`T.Run` are
forbidden; call `synctest.Test` from the subtest body, not inside another
bubble). Precedent: `internal/app/persist/engine_test.go:650-654,684-691`,
`backoff_test.go:114,134`, `timeout_test.go:277`.
**Files:** `internal/app/repl/sim_dump_test.go` only.
**Acceptance probe:** no `time.Sleep`/wall-clock settle in the file; the test
runs 100 iterations (`-count=100`) under load without a hang; the same-seed
subtest still compares two independent bubbles. Run each `runDumpConfirm` in
its own `synctest.Test` bubble: the engine goroutine's cleanup happens at
bubble exit, so two runs in one bubble would leave the first engine alive.

#### F-4. `freeUDPPort` pick-then-bind — structurally remove the class

**Mechanism.** Three packages pick a port from the kernel ephemeral pool with a
throwaway `:0` bind and bind the real socket later:
`cmd/sup-kv/main_test.go:141-149` (`freeUDPPort`),
`internal/udprepl/smoke_test.go:47-60` (`freeUDPPort`),
`internal/grpcapi/wait_integration_test.go:61-75` (`integrationUDPPort`).
`internal/stand` already uses the private window below the ephemeral range plus
a per-process claim registry (`internal/stand/stand.go:88-92,175-228`), and
`cmd/sup-kv/broadcast_probe_test.go:50-100` duplicates the same idea — with the
**same** window, so the two registries cannot coordinate across processes. The
Makefile's claim that class 1 is "structurally mitigated"
(`Makefile:48-50`) therefore covers only the stand, not the packages that
actually flaked (`TestLog_formatText`, the lifecycle tests, the `cmd/sup-kv`
probe). The kernel's ephemeral range on this host is `32768 60999`
(`/proc/sys/net/ipv4/ip_local_port_range`).

**Cure (one allocator, disjoint bands, claim-while-held, bounded retry).**
Move the picker into `internal/testutil` (it may import `net`/`testing`; the
package doc already declares it the module-wide test-helper home and the
`no-test-libs-in-prod` rule exempts `**/test*/**`, `.golangci.yml:52-64`):
- private window below the ephemeral range: `[28100, 32678)`; keep
  `[27900, 28100)` reserved for the fixed doc ports (README 27901/27902,
  EXAMPLES 28001/28002 — already disjoint);
- partition the window into per-package bands so separate test processes cannot
  collide: stand `[28100, 29100)`, `cmd/sup-kv` `[29100, 30100)`, `udprepl`
  `[30100, 31100)`, `grpcapi` `[31100, 32100)`; keep the rest spare;
- one `sync.Mutex` + `map[string]bool` registry keyed by `network:port`; the
  picker binds the candidate, claims it **while the bind is held**, then
  releases it to the test (the stand's existing claim-then-release order);
- bounded retry: 32 candidates, then `t.Fatalf` naming the band; at the
  consumer, a bind failure (`udprepl.New` returns it; the spawned binary exits
  with `address already in use`) retries with a fresh candidate up to 5 times;
- assert the assumption: read `/proc/sys/net/ipv4/ip_local_port_range` and fail
  loudly if its low bound is inside the window (the structural guarantee is
  broken, e.g. a non-default sysctl). On non-Linux, skip the check with a
  comment (darwin is development-only).
- Cover TCP with the same allocator (`freeTCPPort` in `cmd/sup-kv/main_test.go:129-137`
  and `internal/stand/stand.go:223-231`): the gRPC/metrics listener has the
  same pick-then-bind window. TCP and UDP cannot collide on the same port
  number, and the registry key includes the network, so the bands can be shared
  between protocols.
- Replace every `freeUDPPort`/`freeTCPPort`/`integrationUDPPort`/`scenarioUDPPort`
  call site; keep thin package-local wrappers with the package band so call
  sites read the same. Delete the duplicate registry in `broadcast_probe_test.go`.

**Files:** new `internal/testutil/ports.go`; `cmd/sup-kv/main_test.go`,
`cmd/sup-kv/broadcast_probe_test.go`, `internal/udprepl/smoke_test.go`,
`internal/grpcapi/wait_integration_test.go`, `internal/stand/stand.go`; update
the flake-class note at `Makefile:48-50` so it describes the whole-suite
mitigation rather than only the stand.
**Acceptance probe:** `grep -rn 'ListenPacket(.*":0"'` is clean in socket tests
(the remaining `:0` binds are throwaway tx sockets, not advertised ports); the
suite passes at `-count=20` under load; a deliberately occupied candidate in
the band causes a re-pick, not a failure.

#### F-5. `TestLog_formatText` / `TestMain_lifecycle` — port contention and readiness under load

**Mechanism.** `TestLog_formatText` picks `portA, portB` with `freeUDPPort`,
then `probeLoopbackBroadcast` binds `portB` and fails fatally on
`address already in use` (`cmd/sup-kv/main_test.go:1842-1852,1715-1750`); the
same test then waits for a duplicate-node log event with the 10 s
`servingDeadline`. `TestMain_lifecycle` waits for health `SERVING` with the same
10 s (`cmd/sup-kv/main_test.go:1081-1099`); under full-suite `-race` load plus
other packages' processes, a 10 s process-start+gate budget is too tight
(recorded twice in `.planning/phases/06-api-and-semantics/deferred-items.md`).
These are two different classes: the bind failure is F-4; the readiness timeout
is CPU starvation.

**Cure.** (a) F-4 removes the bind class. (b) Add a distinct
`processServingDeadline = 30 * time.Second` used for the first readiness wait of
a freshly spawned child, keeping `servingDeadline = 10 s` for polls of a
already-running node; make `awaitServing`/`startNode` able to check
`node.exited` and fail immediately with the captured stderr on an early child
exit (a crashed child must not look like a slow one). This is a harness
readiness budget, not a product bound, so it does not relax any claim. (c) Give
the duplicate-node log wait its own `logEventDeadline = 30 * time.Second` (it is
a log-delivery wait, not a serving wait). (d) Optionally cap concurrent
spawned binaries in `cmd/sup-kv` with a small semaphore (e.g. 4) if the package
still shows self-contention; do not remove `t.Parallel` wholesale.
**Files:** `cmd/sup-kv/main_test.go` (constants + `awaitServing`),
`internal/stand/stand.go` (`awaitServing` twin if the perf tests use it).
**Acceptance probe:** the two named tests pass repeatedly under a full
`go test -race ./...` at load ~12; a killed child yields a diagnostic naming the
exit status and stderr, not "timed out waiting for health".

#### F-6. `killAndReap` — already-exited child loses its diagnosis

**Mechanism.** `killAndReap` calls `node.cmd.Process.Kill()` and turns any error
into a bare fatal (`internal/stand/stand.go:423-439`). When the child exited on
its own between the confirmed `Set` and the kill, `Kill` returns
`os.ErrProcessDone`; the test says `SIGKILL <pid>: os: process already
finished` and discards the exit status and the captured stderr — exactly the
07-11 phase-gate row (`07-observability-and-configuration/deferred-items.md`;
call site `internal/stand/persist_test.go:76-86,294-300`).
**Cure.** Treat `errors.Is(err, os.ErrProcessDone)` as a closed-but-unexpected
outcome: wait for `node.exited`, read the already-buffered `node.exit` status,
and fail with a message carrying pid, Wait status/exit code, and the captured
stderr (tail) plus stdout. Keep a genuine kill error fatal, also with the
captured output. Optionally store the Wait result in a mutex-guarded field when
the reaper goroutine finishes, so diagnostics can be read without consuming the
1-buffered channel twice.
**Files:** `internal/stand/stand.go` (killAndReap + standNode),
`internal/stand/persist_test.go` (assertion message expectation if any).
**Acceptance probe:** a test (or a controlled child that exits immediately)
shows the diagnostic; the 07-11 scenario under load names the exit status and
stderr.

#### F-7. `TestExamplesDoc` / `TestReadmeQuickstart` — grpcurl walkthrough stall under compile load

**Mechanism.** Both doc tests are `t.Parallel()` (`cmd/sup-kv/examples_test.go:336-337`,
`readme_test.go:75-76`) and execute many blocks through `bash -e` + `go tool
grpcurl` shims; the recorded failure is an empty grpcurl capture (exit 1, no
`NotFound`) under full-suite compile load (`.planning/phases/04-protocol/deferred-items.md`).
Ports are in the reserved band and already disjoint (README 279xx, EXAMPLES
280xx), so this is not F-4; it is CPU/tool-start contention. Each block is
already bounded by `docBlockBudget = time.Minute` (`examples_test.go:123-131`).
**Cure.** Serialize the two heavyweight doc tests (drop `t.Parallel`; they do
not need to run beside the package's ~20 process-spawning tests). Keep the block
budget and the process-group reap. If it still recurs, a dedicated serialized
docs job is the escalation, but the context asks for the cheap structural cure
first.
**Files:** `cmd/sup-kv/examples_test.go:336-337`,
`cmd/sup-kv/readme_test.go:75-76`.
**Acceptance probe:** full `go test -race ./cmd/sup-kv/` at load ~12 passes
repeatedly; a wedged block still fails its own block deadline.

#### F-8. 04 New-5 — `TestChunkSyncEntries_capacityFollowsEntries` has no teeth

**Mechanism.** The test asserts only the chunk shape and order
(`internal/app/repl/config_internal_test.go:62-81`); the implementation sizes
`cur` from `hint := min(len(entries), budget/entryOverheadBytes)`
(`internal/app/repl/core.go:2828-2846`). A regression to
`make(..., 0, budget/entryOverheadBytes)` (the WR-06 bug: tens of millions of
entries at a hostile budget) would still pass every current assertion.
**Cure.** Assert the capacity directly (the test is in `package repl`, so `cap`
is visible): for the 3-entry/max-budget case, `cap(chunks[0]) == len(entries)`
(and never ~2 729); add a multi-chunk case asserting each chunk's cap ≤ the
entries remaining at its creation and that adjacent chunks do not share a
backing array (`&chunks[0][0] != &chunks[1][0]`), pinning the "cur resets to
nil, not cur[:0]" rule. Keep the existing shape/order assertions.
**Files:** `internal/app/repl/config_internal_test.go`.
**Acceptance probe:** the strengthened test fails against the old
`budget/entryOverheadBytes` sizing and passes now.

#### F-9. 04 New-3 — paced-replay double copy

Covered in PERF-05 above; one mechanism (drop the two post-report `Data`
clones), one allocation gate, and the doc/comment updates.

## Architecture Patterns

### Pattern 1: Deterministic work-count gate

**What:** a fixed-iteration loop measures *operations*, not time. Counters live
at the two seams the claim crosses: the consumer-side `StateStore` double for
store reads, and a plain counter in `Core` for pending-entry visits. The gate
asserts exact per-op counts (and cross-size equality), so it is independent of
CPU frequency, host load and the race detector.
**When to use:** every claim of the form "cost is independent of N" where N is
a data-set size.
**Why it is sufficient here:** the key-count claim is structurally a "number of
store calls per datagram" statement (the interface removed the whole-store read);
the unacked claim is structurally "no pending entry is inspected on the idle
path". Both are countable exactly.
**Example:** see Code Examples §1.

### Pattern 2: `testing/synctest` bubble for goroutine-driven virtual time

**What:** run the real `persist.Engine` goroutine inside a synctest bubble over
the in-process virtual scheduler; replace wall-clock settling with
`synctest.Wait()`.
**When to use:** any harness with a real goroutine that consumes virtual-time
channels and a test that must advance virtual time only after it has re-armed.
**Constraints:** `T.Run`, `T.Parallel`, `T.Deadline` must not be called inside
the bubble; create the bubble in the subtest body, not inside another bubble.
**Example:** `internal/app/persist/engine_test.go:650-654` (verified precedent).

### Pattern 3: One port allocator with package bands

**What:** `testutil.UDPPort(t, band)` / `TCPPort(t, band)`: random candidate in
a package-private band below the kernel ephemeral range, bind-probe, claim in a
process-global registry while holding the bind, release to the test, bounded
retry.
**When to use:** every test that advertises a port to a child process or an
adapter.
**Why bands:** test binaries are separate processes, so an in-process registry
cannot coordinate them; disjoint bands make cross-process collisions impossible
by construction (the same reasoning the stand already documents at
`internal/stand/stand.go:78-92`).

### Pattern 4: Perf target with an env gate, central thresholds, hardware block

**What:** `make perf` sets `SUPKV_PERF=1`, runs only `^TestPerf` tests with
`-p 1` and no `-race`, and the tests read one const threshold table and log a
hardware block.
**When to use:** operator-facing wall-clock thresholds that must not run in the
shared-runner PR gate.
**Why an env gate, not a build tag:** the repo's established pattern is
`SUPKV_RED_TESTS` (`Makefile:69-78`); a build tag would hide the files from
`go vet`/lint unless every tool is taught the tag, and a silent compile rot is
worse than a named skip.

### Anti-patterns to avoid

- **Wall-clock ratio gates in `make test`:** the exact class this phase removes
  (load 9-13 on 16 cores is normal on this host).
- **A second harness/load generator:** the stand + `redRunBurst` already exist.
- **`testing.AllocsPerRun` inside a `t.Parallel` test or beside goroutines:**
  it sets `runtime.GOMAXPROCS(1)` process-wide during the measurement
  (`go doc testing.AllocsPerRun`).
- **`runtime.GC()` inside a benchmark loop:** distorts the measurement and the
  reported allocations; report `peak-live-B` and let the runtime behave
  normally.
- **Measuring the adapter when the claim is about the core** (or vice versa):
  state the layer; e.g. `BenchmarkReplicationBurst` measures two real adapters
  and convergence-inclusive throughput, so its ~420-440 writes/s is not the
  1 000/s burst bound.
- **Publishing a benchmark number without the profile/flags, state size, media
  and race state:** every table in `docs/PERFORMANCE.md` carries the block.
- **Perf tests running under `-race`:** the race build measures the detector,
  not the product; skip with a named reason.

## Code Examples

### 1. Deterministic count gate (pattern to implement)

Existing verified reporting idiom (`internal/app/repl/bench_test.go:122-124`):

```go
b.ReportAllocs()
b.ReportMetric(float64(keys), "store-keys")
b.ResetTimer()
```

`testing.BenchmarkResult` exposes reported metrics (`go doc testing.BenchmarkResult`):

```go
// Extra records additional metrics reported by ReportMetric.
Extra map[string]float64
```

Recommended shape (proposed; stdlib only):

```go
// countingStore delegates to the real stateStore and counts the seam calls
// the inbound handlers make. Single-goroutine fixture: plain ints, no atomics.
type countingStore struct {
    inner                    *stateStore
    version, lookup, versions int
    records, observes, generations int
}

func (c *countingStore) Version(k string) (version.RecordVersion, bool) {
    c.version++
    return c.inner.Version(k)
}
// ... Lookup, Versions, ApplyRemote, Observe, Generation likewise.

func assertStoreCounts(t *testing.T, name string, got, want countingStore) {
    t.Helper()
    if got.version != want.version || got.lookup != want.lookup || got.versions != want.versions {
        t.Fatalf("%s: store calls = version %d lookup %d versions %d, want %d/%d/%d",
            name, got.version, got.lookup, got.versions, want.version, want.lookup, want.versions)
    }
}
```

The gate runs the same `benchHandleDataRequest`/`benchHandleDataSync` body for a
fixed K (e.g. 1 000) iterations at 1 000 and 50 000 keys and compares
`(after-before)/K`. No `testing.Benchmark` is involved in the assertion, which
is what removes the load sensitivity and the 79 s runtime.

### 2. `PendingVisits` seam (pattern to implement)

```go
// PendingVisits returns how many pending entries the core's hot paths have
// inspected since construction: the deterministic cost seam for the
// XPORT-02/PERF-05 flatness gates. Driver-goroutine-only like every other
// core counter; it is not exported as a metric.
func (c *Core) PendingVisits() uint64 { return c.pendingVisits }
```

Increment exactly at the loops that should be key-local (`trackedVersion`,
`removeSupersededLocal`, `applyRemote`'s displacement loop, `scanRetransmit`'s
due loop) and at the deliberately O(n) discovery flush, so the count is a
complete measure. Gate: `visits/op == 0` for an idle `Advance` at both sizes
and `visits/op == 0` for a fresh-key write at both sizes (a repeated-key write
may visit ≤2; the gate's fixture uses fresh keys).

### 3. Minimal 04 New-3 patch (verified call graph)

```go
// internal/grpcapi/server.go (current, :412-421) — remove the clone:
_ = s.opts.ReplayNotify(rec.Key, storage.Record{Version: rec.Version})

// internal/udprepl/loop.go (current, :254-267) — remove cloneLocalRecord:
return a.enqueueLocalWrite(localWrite{
    key:    key,
    rec:    storage.Record{Version: rec.Version},
    replay: true,
})
```

`Core.NotifyReplayedWrite` reads only `rec.Version` through `trackLocalWrite`
(`internal/app/repl/core.go:2350-2361,2370-2442`), and the paced drain re-reads
the record through `Lookup` (`core.go:3032-3064`), so `Data`, `Deleted` and
`Checksum` are dead on this path. Update the seam docs; the `RestoredRecord.Data`
clone in `persist` (`internal/app/persist/snapshot.go:216-227`) remains the
report's own payload.

### 4. `make perf` and the threshold table (proposed)

```make
.PHONY: ... perf
perf:
	mkdir -p .tmp/perf
	SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(CURDIR)/.tmp/perf \
	  go test -count=1 -p 1 -timeout 30m -v -run '^TestPerf' \
	  ./internal/stand/ ./internal/app/repl/ ./internal/grpcapi/
```

```go
// internal/testutil/perf.go — the one threshold place (D-06). Consts, not
// globals, so gochecknoglobals has nothing to flag.
const (
    PerfDefaultP50        = 750 * time.Millisecond
    PerfDefaultP99        = 1500 * time.Millisecond
    PerfFastMean          = 25 * time.Millisecond
    PerfFastP99           = 100 * time.Millisecond
    PerfBurstMinRate      = 990.0
    PerfBurstDrain        = 5 * time.Second
    PerfBytesPerDumpRatio = 1.10
    PerfAbsentPeerRatio   = 2.0
)
```

Each test fails with the PERF ID and the constant value, e.g.
`"PERF-01 default-profile p99 = 1.62s > PerfDefaultP99 (1.5s) [10k x 1KiB, 1000 writes/s, NVMe ext4, race off]"`.

### 5. Hardware block (proposed helper)

Log once per perf test (or in `TestPerfHardware`):

```
hardware: CPU=Intel(R) Core(TM) i9-9980HK @ 2.40GHz; logical=16; RAM=64.0 GiB;
load=8.07 7.93 8.88; disk=/dev/mapper/... ext4 (nvme0n1, rotational=0);
go=go1.27.4; module=github.com/Djarvur/sup-kv (vcs bf6966f, 2026-11-23);
race=off; GOMAXPROCS=16; dump=/tmp/.../dump.json
```

Sources: `/proc/cpuinfo` `model name`, `runtime.NumCPU`, `/proc/meminfo`
`MemTotal`, `/proc/loadavg`, `/proc/self/mountinfo` + `/sys/block/*/queue/rotational`,
`runtime.Version`, `debug.ReadBuildInfo`, the race build-tag pair, `runtime.GOMAXPROCS(0)`.

## Common Pitfalls

### Pitfall 1: Wall-clock thresholds on a shared runner

**What goes wrong:** a 2× ratio gate measures host load, not code. The recorded
history is unambiguous: the same tree passes at 0.87-1.05 standalone and fails
at 2.80-6.0 during full-suite runs while load is 9-13 on 16 cores
(`03/04/05/07 deferred-items`).
**Why it happens:** `go test ./...` runs packages in parallel processes; `-race`
multiplies CPU; a busy developer machine or a hosted runner adds unrelated load.
**How to avoid:** operation-count gates in `make test`; wall-clock thresholds
only in the env-gated `make perf`, run serially (`-p 1`), without `-race`, on a
documented host. Never "fix" a flake by relaxing the claim.
**Warning signs:** a gate that passes standalone and fails only in `make test`;
a ratio that moves with `uptime`.

### Pitfall 2: Self-pacing benchmarks

**What goes wrong:** `testing.B` auto-scales `b.N`; a benchmark with a 1 ms
ticker "paces the workload" but the measured span can converge at a different
rate than production; a throughput number divided by send-window + drain
understates the achieved rate.
**Why it happens:** Go benchmarks are designed to run as fast as possible; the
repo's existing benchmarks deliberately use a ticker as the *workload shape*
(`internal/grpcapi/bench_test.go:45`, `internal/udprepl/bench_test.go:26`) and
report convergence-inclusive `writes/s`.
**How to avoid:** state the workload shape in the comment; report per-op
latency from the measured span; don't gate throughput on a benchmark whose rate
includes convergence (the burst proof is the stand test, not
`BenchmarkReplicationBurst`). Use a fixed `-benchtime=30x` for the
default-profile benchmark where `b.N` would otherwise be 1-2 samples.

### Pitfall 3: Measuring with `-race`

**What goes wrong:** the race detector adds 2-20× CPU and allocation overhead;
a threshold calibrated without it fails every time under `make test`'s `-race`.
**Why it happens:** the PR gate runs `go test -race ./...`; perf tests are
skipped there but a developer may run them with `-race` by habit.
**How to avoid:** `make perf` runs without `-race`; every `TestPerf*` skips with
a named reason when the race build tag is set; the hardware block prints
`race=off`.

### Pitfall 4: `runtime.GC()` inside a benchmark

**What goes wrong:** a forced GC per iteration removes the allocator/GC behavior
production actually has, collapses `peak-live-B`, and can make a streaming
design look better or worse than it is.
**Why it happens:** "to make memory numbers stable".
**How to avoid:** report `peak-live-B` (already sampled after the fill,
`internal/app/persist/bench_test.go:92-100`), use `-benchmem`, and let GC run
normally; compare points across runs, not GC-forced snapshots.

### Pitfall 5: Hardware-dependent numbers without a block

**What goes wrong:** the review's numbers came from an Apple-silicon Docker VM
with a virtual disk (`docs/reviews/.../evidence/stand/RESULTS.md:3-8`); the
"~10 ms" claim lost its conditions and became folklore. A dump on NVMe vs eMMC
differ by an order of magnitude.
**How to avoid:** every published row carries CPU, RAM, disk device/fs, load,
Go version, commit and race state; `make perf` logs the block; docs copy it and
the command. If target media is absent, say so and record the gap (D-03).

### Pitfall 6: Measuring the wrong layer

**What goes wrong:** `BenchmarkReplicationBurst` measures two real adapters plus
convergence (~420-440 writes/s), not the 1 000/s burst bound; the fast-profile
benchmark uses an in-memory dump, so its ~10 ms excludes real disk time;
`BenchmarkCoreEventLoop` measures only an idle `Advance`, not the write path.
**How to avoid:** state the layer and the excluded components for every number.
PERF-05's proof lives at the core notification path (where the scan is) plus a
paired live/absent arm; PERF-01's default numbers live at the stand (real disk);
the fast-profile claim explicitly says "in-memory dump".

### Pitfall 7: `testing.AllocsPerRun` changes `GOMAXPROCS`

**What goes wrong:** `AllocsPerRun` sets `runtime.GOMAXPROCS(1)` process-wide
during the measurement (`go doc testing.AllocsPerRun`); using it in a
`t.Parallel` test or beside running goroutines distorts both.
**How to avoid:** use it only in a sequential, single-goroutine test, or avoid
it entirely: the count gates count calls, and `testing.BenchmarkResult` already
reports `AllocsPerOp`/`AllocedBytesPerOp`.

### Pitfall 8: "State size" means two different things

**What goes wrong:** the OPERATIONS memory model says allocation churn is
"roughly 1.4 times the state size" (`docs/OPERATIONS.md:209-214`), which is true
when values dominate; with 10 000 small keys this session measured `bytes/op` ≈
19 MB against a 5.1 MB dump, because per-key map/record/line overhead dominates.
A gate that assumes 1.4× will fail on a key-dominated fixture.
**How to avoid:** define `logical_state = Σ(len(k)+len(v))` and report both
state and dump bytes; publish points for a value-dominated state (60 KiB) and
a key-dominated state (10 k × 1 KiB), not one ratio.

### Pitfall 9: Page cache is not disk, and `/proc/io` is not the device

**What goes wrong:** the dump benchmark writes to `b.TempDir()` through the page
cache; `/proc/<pid>/io` `write_bytes` counts bytes submitted to the storage
layer, not journal/COW amplification.
**How to avoid:** call the benchmark number "bytes/op (page cache)", call the
stand number "process-submitted bytes (`write_bytes`)", and record the fs/device
with the media facts. Device-level amplification is out of scope and named as a
limitation.

### Pitfall 10: `synctest` forbids the usual subtest tools

**What goes wrong:** calling `t.Run`, `t.Parallel` or `t.Deadline` inside a
bubble panics/fails (`go doc testing/synctest.Test`).
**How to avoid:** create the bubble in the subtest body and keep all table/
subtest structure outside it; remove the dumpNotify subtests' `t.Parallel`.

### Pitfall 11: A perf gate that silently never runs

**What goes wrong:** an env-gated test that nobody sets, in a target CI never
invokes, rots unnoticed.
**How to avoid:** compile it in the normal suite (no build tag), name the gate
in the doc and the Makefile, and add `TestPerformanceDoc` so a threshold or
command drift breaks `make test`. Note the residual in the assumptions log:
`make perf` is operator-facing and not in CI (D-06).

### Pitfall 12: Lint/architecture side effects of test helpers

**What goes wrong:** a global registry trips `gochecknoglobals`; a helper in
`internal/app` importing `net`/`os` is fine only for test files (`!$test` at
`.golangci.yml:68-79`); a helper imported by adapters from the core path would
trip depguard.
**How to avoid:** keep the port allocator and hardware block in
`internal/testutil` (the declared module-wide test-helper home), use `const`
tables/functions instead of global vars, and use `//nolint:gochecknoglobals`
only for the one shared claim map with a written reason (the stand's existing
pattern, `internal/stand/stand.go:185`).

### Pitfall 13: Container CPU quotas vs `runtime.NumCPU`

**What goes wrong:** a 16-CPU host with a 2-CPU cgroup quota reports 16; a
threshold calibrated on 16 logical CPUs fails under the quota.
**How to avoid:** log `runtime.NumCPU`, `runtime.GOMAXPROCS(0)` and the cgroup
quota if readable; document the runner class; keep headroom in starting values
and treat `make perf` as a dedicated-host target, not a CI target.

### Pitfall 14: Comparing numbers from different runs/hardware

**What goes wrong:** a ratio computed across two different benchmark runs (or
two hosts) mixes machine noise into the claim.
**How to avoid:** the paired arms live in the same test/process; a ratio is
computed only from arms measured back-to-back in one invocation; docs publish
each table with one hardware block.

## State of the Art

| Old approach | Current approach | When it changed | Impact here |
|--------------|------------------|-----------------|-------------|
| Wall-clock ratio gates in the always-on suite | Deterministic operation counts | This phase (2026-11) | Gates run in <1 s, load-independent; ns/op stays a manual metric |
| Hand-rolled sleep/settle loops for virtual-time harnesses | `testing/synctest` (`Test`/`Wait`) | Stable Go 1.25; used in this repo since Phase 5 | Removes the dumpNotify flake class and the test-binary timeout |
| `:0` pick-then-bind for test ports | Private window below the ephemeral range + per-process claim registry | Phase 2 for the stand; this phase extends it to every socket package | Removes the pick-to-bind class structurally |
| Encode-into-`bytes.Buffer` dump (3 materializations) | One `Records()` copy + streamed JSONL through a 64 KiB writer | Phase 5 / PERSIST-06 | The memory model and `peak-live-B` now measure the streaming design |
| `ru_maxrss` peak RSS metric | Current RSS from `/proc/self/statm` (Linux) | Phase 7 / OBSV-03 | The perf memory points use an honest current gauge |
| Reviewer's external loadgen / ad-hoc benchmarks | The in-repo `redRunBurst` + stand perf tests behind one target | This phase | One reproducible command per published number |
| `testing.B` auto-scaling as a threshold oracle | Fixed-workload, fixed-iteration measurement; `ReportMetric` for shape | This phase | No self-pacing, no coordinated omission in the default-profile latency |

**Deprecated/outdated in this phase's scope:**
- `t.Parallel` on wall-clock gates: no longer meaningful once the gate counts
  operations; keep parallelism for speed.
- The Makefile's claim that flake class 1 is "structurally mitigated" for the
  whole suite (`Makefile:48-50`): true only for `internal/stand` today; this
  phase makes it true for `cmd/sup-kv`, `udprepl` and `grpcapi` too.
- The PROJECT.md "~10 ms under 1000 w/s" bullet without conditions
  (`.planning/PROJECT.md:43`): replaced by a pointer to the measured profile.

## Assumptions Log

Every row here is an `[ASSUMED]` claim or a starting value derived from a
measurement, not a locked decision. A1, A2 and A4 are the three the planner
should confirm with the target host/user before locking thresholds.

| # | Claim | Section | Risk if wrong |
|---|-------|---------|---------------|
| A1 | The threshold values in this report are **starting values**, not measured target-hardware decisions | PERF-06 table | A threshold could be too tight for the target board or too loose to catch a regression; the executor must run `make perf` once and record the measured baseline before locking values |
| A2 | Target-grade media (eMMC/SD on the two-board appliance) is not available on this host; the available media is an NVMe SSD | PERF-03 | The published amplification number would be mislabeled if the doc did not record the gap; D-03 explicitly permits recording it |
| A3 | `/proc/<pid>/io` `write_bytes` meaning and attribution: "the number of bytes really sent to the storage layer" | PERF-03 | If the kernel attributes buffered writes late or coalesces them, the cross-check is approximate; the dump-size × count formula is the primary number |
| A4 | `make perf` runs on a calm, dedicated host; the recorded load is advisory, not a gate | PERF-06 | A loaded host fails thresholds that would pass clean; document and rerun rather than relax |
| A5 | The per-key index does not change observable behavior (same pending set, same results, only key-local iteration) | PERF-05 | If a hidden path inserts a second pair per key, a single-value index would overwrite; the invariant test and the review's set alternative mitigate |
| A6 | `Core.NotifyReplayedWrite` and `trackLocalWrite` read only `rec.Version` | PERF-05 / New-3 | Verified by reading `core.go:2350-2442`; if a future change reads `Data`/`Checksum`, the version-only replay record would break it and the allocation gate would pass while behavior would not; add a comment + test |
| A7 | The fast-profile "~10 ms" claim is only valid under the benchmark's conditions (in-memory dump, 1 ms burst, two loopback adapters) | PERF-01 | Stating it standalone would be false; the doc anchor must pin the conditions |
| A8 | The ephemeral port range starts at 32768 on the target Linux hosts | F-4 | If a host narrows/lowers it, the private band could collide; the allocator's sysctl assertion makes this a loud failure, not a flake |
| A9 | Go 1.27.4 with `testing/synctest` stable and the `race` build tag set by `-race` | F-3, PERF-06 | Verified locally (`go doc`, a temp `-race` build-tag probe); a Go bump keeps both |
| A10 | `SUPKV_PERF=1` gating is acceptable for an operator-facing target (the established `SUPKV_RED_TESTS` pattern) | PERF-06 | If a verifier treats the skip as a silent miss, the skip message names `make perf`; the tests still compile in every suite |
| A11 | The two ratio-gate claims are exactly "constant store calls per datagram" and "zero pending visits on the idle path" | F-1/F-2 | A future non-instrumented scan would evade the visits gate; the paired ns/op benchmark in `make perf` is the backstop |
| A12 | `dumps_total` (`supkv_dumps_total`) is monotone and counts every attempt (success and failure) | PERF-03 | Verified in `internal/metrics/metrics.go:446-500` and `scheduler.recordDump`; a failed dump also writes no generation, so the size×count formula must use successful dumps only — the test must use a healthy window |
| A13 | The stand's shrunken timers can be overridden per test by later flags | PERF-01/02 | `nodeArgsWithExtras` appends extras after the defaults (`internal/stand/stand.go:676-684`); later flags win in `main.go` parsing |
| A14 | The published numbers will be rerun by the executor on its own hardware; this report's host numbers are baselines only | Summary, PERF-06 | Publishing this host's NVMe numbers as target numbers would violate PERF-01/03 conditions |
| A15 | PERF-01/03/04 tests write dumps to a real filesystem (`SUPKV_PERF_DUMP_DIR`, default `.tmp/perf`), never blindly to `/tmp` | PERF-03, PERF-06, Environment | A tmpfs dump would measure memory and make the amplification/dump-time numbers meaningless; the test must assert and log the fs type |

## Open Questions (RESOLVED)

1. **Can the two wall-clock ratio gates be made deterministic without weakening
   the claim?**
   - Resolution: Yes. Replace the ns/op ratio with operation counts at the two
     seams the claims cross: a counting `StateStore` double for inbound
     handlers, and a `PendingVisits` counter in the core for the idle/write/
     apply paths. Exact expected counts make the claim stronger than a 2×
     wall-clock bound. Evidence: `StateStore`'s doc says the whole-store read
     was removed so O(store)-per-datagram is compile-impossible
     (`internal/app/repl/core.go:393-400`); handlers call `Version` per entry
     and `Observe` once (`core.go:1794-1834`); `UnackedStats` and
     `scanRetransmit` are O(1) by construction (`core.go:3346-3362,2964-2975`).
2. **Do the benchmarks need `testing.Benchmark` for the gate?**
   - Resolution: No. A fixed-iteration loop plus counter deltas is sufficient
     and runs in milliseconds; `testing.Benchmark` stays for `make bench` /
     `make perf` reporting. `testing.Benchmark` auto-scales each arm for ≥1 s,
     and the 10 000-entry fixture is seeded through an O(n²) path today; the two
     gates took 79 s under `-race` this session, most of it setup/scaling.
3. **Does write latency with an absent peer still degrade, and why?**
   - Resolution: Yes, structurally: every local write scans the whole pending
     map twice (`trackedVersion`, `removeSupersededLocal`,
     `core.go:3429-3452`), and every inbound `DataReplication` scans it once
     (`applyRemote`, `core.go:1659`). The adapter's `mirrorState` copy is gone
     (`internal/udprepl/mirror.go:212-267` is O(1)), but the core scans remain;
     this is 03-REVIEW IN-02, still open with "Note for PERF/Phase 8"
     (`03-REVIEW-DISPOSITION.md:22`). Fix: per-key pending index + the gates
     above + a paired benchmark.
4. **What is the minimal 04 New-3 fix?**
   - Resolution: Drop the two `Data` clones after the `RestoredRecord` is built:
     `server.notifyReplayedWrite` (`internal/grpcapi/server.go:412-421`) and
     `adapter.NotifyReplayedWrite` (`internal/udprepl/loop.go:254-267`). The core
     reads only `rec.Version` (`core.go:2350-2361,2370-2442`) and the drain
     re-reads via `Lookup` (`core.go:3032-3064`). No signature change required;
     update the seam docs and add an allocation gate.
5. **Can `make perf` wrap the stand suite without a second harness?**
   - Resolution: Yes. `internal/stand` already has build-once, launch, reap,
     real gRPC clients, `/metrics` scrapes, the reviewer-shaped load generator
     and a private port allocator. Add `internal/stand/perf_test.go` with
     `SUPKV_PERF=1` rows; `-p 1` and no `-race`; CI keeps `make bench-smoke`
     (D-06/D-13/D-14).
6. **Where do thresholds live?**
   - Resolution: One const block in `internal/testutil` (the declared
     module-wide test-helper home, imported by every test package, exempt from
     the no-test-libs lint rule). No env parsing, no threshold file, no stored
     profiles; `TestPerformanceDoc` anchors the doc to the consts.
7. **Can the dumpNotify harness be made deterministic?**
   - Resolution: Yes, on `testing/synctest`: the engine goroutine runs in a
     bubble, `synctest.Wait()` supplies the idle barrier, and `settle`/
     `awaitActivity`/the 200 µs quiet rounds are deleted. Constraints and
     precedent: `go doc testing/synctest.Test`; `internal/app/persist/engine_test.go:650-654,684-691`.
8. **How is disk amplification measured from the shipped binary?**
   - Resolution: Preload a fixed state; record `supkv_dumps_total`, the dump
     file size and `/proc/<pid>/io` `write_bytes` over a window; drive a known
     logical byte count; compute `(Δdumps × dumpBytes)/logicalBytes` and
     cross-check `Δwrite_bytes` against `Δdumps × dumpBytes`. Each `runDump`
     writes one full generation through `SaveStream` (`scheduler.go:228-249`);
     rotation renames, it does not rewrite.
9. **How is RSS / dump-transient memory sampled?**
   - Resolution: Current RSS from the phase-7 `process_resident_memory_bytes`
     metric (`internal/metrics/metrics.go:539-540`, Linux `/proc/self/statm`),
     plus `/proc/<pid>/status` `VmRSS`/`VmHWM` sampling around a dump; the
     precise transient is the dump benchmark's `peak-live-B`
     (`internal/app/persist/bench_test.go:92-100,117-134`).
10. **What are the exact latency formulas?**
    - Resolution: Default under sustained flow: `latency = U(0, MaxWait) +
      dump_time`; p50 ≈ MaxWait/2 + dump, p99 ≈ MaxWait + dump. Sequential
      single client: `Debounce + dump_time`, ceiling ≈1/(Debounce+dump) ≈14/s.
      Fast profile with the in-memory benchmark dump: measured mean 11.53 ms,
      p50 6.10, p99 44.93 under the benchmark's conditions. Source:
      `scheduler.rearm` (`scheduler.go:176-199`) and this session's runs.
11. **Does PERF-02 need the red test ungated?**
    - Resolution: No. Keep the red row (nightly evidence) and add a perf row
      that calls the same extracted scenario under `SUPKV_PERF=1`; both use the
      same thresholds (990/s, 5 s drain). The docs describe the
      guarantee/non-guarantee split; S-FAIL-15 is updated to point at the test.
12. **Where does the write-latency model live?**
    - Resolution: New `docs/PERFORMANCE.md` (model + measured tables +
      thresholds + commands). OPERATIONS keeps a short memory summary and links
      to it; API.md's Phase-9 section links to it instead of duplicating.
13. **How are the port flakes structurally removed?**
    - Resolution: One allocator in `internal/testutil`: private window below
      the kernel ephemeral range, disjoint per-package bands, in-process claim
      registry held across the pick, bounded retry, and a sysctl assertion.
      The stand's existing window/registry is the pattern; the missing packages
      are `cmd/sup-kv`, `internal/udprepl` and `internal/grpcapi`.
14. **What about `TestMain_lifecycle`'s health timeout?**
    - Resolution: It is a readiness budget, not a product bound. Add a distinct
      process-start deadline (30 s) for a freshly spawned child, check the
      child's exit channel in the readiness wait and report its stderr; keep
      the 10 s poll for already-running nodes.
15. **How does `killAndReap` diagnose an already-exited child?**
    - Resolution: Treat `os.ErrProcessDone` as an unexpected closed outcome,
      wait for `node.exited`, read the buffered `node.exit` status, and fail
      with pid + Wait status + captured stderr/stdout. Genuine kill errors stay
      fatal with the same diagnostics.
16. **What are the deterministic cures for the load-sensitive doc walkthrough?**
    - Resolution: Drop `t.Parallel` from `TestExamplesDoc` and
      `TestReadmeQuickstart`; they compete with ~20 process-spawning tests in the
      same package, and their ports are already in the disjoint reserved band.
      Keep `docBlockBudget` and the process-group reap.
17. **How is 04 New-5 given teeth?**
    - Resolution: Assert `cap` directly in the internal test: one chunk at a
      max budget has `cap == len(entries)` (not `budget/entryOverheadBytes`),
      multi-chunk caps fit the remaining entries, and adjacent chunks do not
      alias. The implementation's hint is
      `min(len(entries), budget/entryOverheadBytes)` (`core.go:2828-2846`).
18. **Are new dependencies needed?**
    - Resolution: No. Every mechanism is stdlib (`testing`, `synctest`,
      `runtime`, `runtime/debug`, `os`, `syscall`, `/proc`) or existing in-repo
      test infrastructure. The runtime require set stays pinned.

## Environment Availability

| Dependency | Required by | Available | Version / evidence | Fallback |
|------------|-------------|-----------|--------------------|----------|
| Go toolchain | everything | ✓ | `go1.27.4 linux/amd64` | — |
| `make` | `make test`, `make perf` | ✓ | GNU make in the repo workflow | run the `go test` line directly |
| Linux `/proc` (`io`, `statm`, `status`, `loadavg`, `meminfo`, `cpuinfo`, `mountinfo`, `ip_local_port_range`) | amplification, RSS, hardware block, port window assertion | ✓ | `/proc/sys/net/ipv4/ip_local_port_range` = `32768 60999` | `runtime`-only hardware block; amplification method becomes stat+count only |
| Real Linux sockets / loopback broadcast | stand and perf tests | ✓ | stand suite runs green on this host | darwin is development-only; perf target is Linux |
| 16 logical CPUs, 64 GiB RAM | perf host baseline | ✓ | i9-9980HK, 64 GiB | thresholds carry headroom; smaller hosts may need re-baselining |
| NVMe SSD, ext4/LVM | available dump media | ✓ | `APPLE SSD AP1024N`, rotational=0, ext4 | **not target-grade**; record the gap for eMMC/SD (D-03) |
| Non-tmpfs dump directory | PERF-01/03/04 real-disk measurements | ✓ on this host (`/tmp` and the repo both resolve to the ext4 root) | `SUPKV_PERF_DUMP_DIR=$(CURDIR)/.tmp/perf` (gitignored, `.gitignore:15`) | if only tmpfs exists, tmpfs results are memory measurements; record that as part of the PERF-03 gap |
| Target-grade eMMC/SD media | PERF-03 target measurement | ✗ | not attached | record method + gap against Spec open question #2 |
| Calm host (load < NumCPU) | meaningful perf numbers | ✗ at research time | load 8.07 on 16 CPUs | run `make perf` on a quiet host; log load; do not relax thresholds |
| `grpcurl` (go tool) | doc walkthroughs | ✓ | `go tool grpcurl` per `go.mod:5-13` | — |
| `golangci-lint` (go tool) | lint gate | ✓ | `go tool golangci-lint` v2.15.0 | — |
| Docker / external loadgen / `fio` / `iostat` / `benchstat` | — | not used | — | not needed; the stand + `/proc` cover the measurements |

**Missing dependencies with no fallback:**
- Target-grade eMMC/SD media for PERF-03. D-03 permits recording the gap
  honestly; do not relabel NVMe numbers as target numbers.

**Missing dependencies with fallback:**
- A calm perf host: rerun on a quiet host; thresholds are operator-facing.
- Darwin perf measurement: Linux-only by design (14.10); darwin remains a
  development platform.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (Go 1.27.4) + `go tool golangci-lint` v2.15.0 |
| Config file | `go.mod` (go directive), `.golangci.yml`, `Makefile`; no extra test config |
| Quick run command | `go test -race -count=1 ./internal/app/repl/ ./internal/stand/ ./cmd/sup-kv/` |
| Full suite command | `make test` (`go test -race -cover ./...`) |
| Perf run command | `make perf` (`-p 1`, no `-race`, `SUPKV_PERF=1`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test type | Automated command | File exists? |
|--------|----------|-----------|-------------------|--------------|
| PERF-01 | Default-profile model + measured p50/p99; fast-profile measured mean/p99; sequential ceiling documented | Perf (stand + benchmark wrapper) + doc anchor | `make perf`; `go test -run '^TestPerformanceDoc$' ./cmd/sup-kv/` | ❌ Wave 0 (`internal/stand/perf_test.go`, `internal/grpcapi` perf wrapper, `docs/PERFORMANCE.md`) |
| PERF-02 | 1 000/s distinct-key burst ≥990/s and ≤5 s drain; guarantee/non-guarantee text | Perf (stand) + doc anchor + existing red row | `make perf`; `SUPKV_RED_TESTS=1 go test -run '^TestRed_R02_burst$' ./internal/stand/` | Partial (`TestRed_R02_burst` exists; perf row + doc text are Wave 0) |
| PERF-03 | Amplification measured (or gap recorded) with method + media | Perf (stand) + doc | `make perf` | ❌ Wave 0 |
| PERF-04 | Memory model formula + points with hardware | Perf (stand) + doc | `make perf` | ❌ Wave 0 (OPERATIONS has the qualitative model) |
| PERF-05 | Absent/live paired benchmark ratio + deterministic visits gate; New-3 allocation gate | Unit/deterministic + perf | `go test -race ./internal/app/repl/`; `make perf` | ❌ Wave 0 (index + gates + benchmark) |
| PERF-06 | `make perf` with central thresholds + hardware block; doc anchor | Build target + doc anchor | `make perf`; `make lint`; `make test` | ❌ Wave 0 (`Makefile` target, `internal/testutil/perf.go`, `docs/PERFORMANCE.md`) |

### Deterministic gates (always in `make test`)

| Gate | Assertion | Command |
|------|-----------|---------|
| `TestCore_keyCountCostFlat` | Per-datagram store calls identical at 1 000/50 000 and equal to the expected shape; `Versions`/`Records` zero | `go test -race -run '^TestCore_keyCountCostFlat$' ./internal/app/repl/` |
| `TestCore_unackedCostFlat` | `PendingVisits/op == 0` on an idle `Advance` at 10/10 000 | `go test -race -run '^TestCore_unackedCostFlat$' ./internal/app/repl/` |
| `TestCore_notifyWriteCostFlat` | `PendingVisits/op` for a fresh-key write is 0 at both 10 and 10 000 pending (a repeated-key write may be ≤2, but the fixture uses fresh keys) | `go test -race -run '^TestCore_notifyWriteCostFlat$' ./internal/app/repl/` |
| `TestCore_applyRemoteCostFlat` | `PendingVisits/op` for an inbound replication for a fresh key is 0 at both 10 and 10 000 pending | `go test -race -run '^TestCore_applyRemoteCostFlat$' ./internal/app/repl/` |
| `TestChunkSyncEntries_capacityFollowsEntries` | `cap` follows entries; adjacent chunks do not alias | `go test -run '^TestChunkSyncEntries_capacityFollowsEntries$' ./internal/app/repl/` |
| `TestScenario_dumpNotify` | No wall-clock settle; `-count=100` under load | `go test -race -count=100 -run '^TestScenario_dumpNotify$' ./internal/app/repl/` |
| Replay allocation gate | One 60 KiB replay notify allocates far below the record size | `go test -run '^TestReplayNotify_noDataCopy$' ./internal/udprepl/` (or the benchmark gate) |
| Port allocator | No `:0` advertised port in socket tests; bands disjoint | `grep -rn 'ListenPacket' --include='*_test.go' internal cmd` |

### Sampling Rate

- **Per task commit:** the package-scoped quick command above; lint stays clean.
- **Per wave merge:** `make test` (race, full suite).
- **Phase gate:** `make test` + `make lint` + `make perf` on a calm host (record
  the hardware block) + `make red-run` remains green (the perf changes touch
  no red row, but the stand package is in the red-runner's package list).
- **Before publishing numbers:** rerun `make perf` twice on the target-class
  host; publish the second run with its block.

### Wave 0 Gaps

- [ ] `internal/testutil/perf.go` — threshold consts, `PerfEnabled`,
      `RaceEnabled`, `HardwareBlock`.
- [ ] `internal/testutil/ports.go` — shared private-band allocator + registry.
- [ ] `internal/app/repl/bench_test.go` — counting store, count gates, notify/
      apply benchmarks and gates, IN-04 fixture fix.
- [ ] `internal/app/repl/core.go` — per-key pending index + `PendingVisits`.
- [ ] `internal/app/repl/sim_dump_test.go` — synctest conversion.
- [ ] `internal/stand/perf_test.go` — default/fast/burst/amplification/memory
      perf rows; hardware log; `SUPKV_PERF` gate; carry `//go:build linux` like
      the rest of the stand package.
- [ ] `internal/grpcapi/bench_test.go` — profile table; perf wrapper test.
- [ ] `internal/stand/stand.go` — allocator adoption; `killAndReap` diagnostic.
- [ ] `cmd/sup-kv/main_test.go`, `internal/udprepl/smoke_test.go`,
      `internal/grpcapi/wait_integration_test.go` — allocator adoption;
      readiness deadline; doc-test serialization.
- [ ] `internal/grpcapi/server.go`, `internal/udprepl/loop.go` — New-3 clones.
- [ ] `Makefile` — `perf` target + `.PHONY`.
- [ ] `docs/PERFORMANCE.md`, `docs/OPERATIONS.md` link, `docs/SPEC-DEVIATIONS.md`
      S-FAIL-15, `.planning/PROJECT.md:43` condition pointer.
- [ ] `cmd/sup-kv` `TestPerformanceDoc` (or place it with the existing doc
      anchors) — doc/threshold drift guard.

## Security Domain

`security_enforcement` is enabled in `.planning/config.json`; ASVS level 1.
This phase adds no authentication, session, cryptographic or network-exposure
surface; it adds test-only measurement code and one Makefile target.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard control |
|---------------|---------|------------------|
| V2 Authentication | no | No auth surface changed (replication trust model unchanged; ADR/README trust section untouched) |
| V3 Session Management | no | No sessions |
| V4 Access Control | no | No new endpoints or RPCs; the metrics endpoint remains opt-in (`--metrics-addr`) |
| V5 Input Validation | yes (existing, unchanged) | `internal/config` validation is untouched; perf tests read only `SUPKV_PERF=1` and never parse user input; the doc-anchor test reads repo files only |
| V6 Cryptography | no | No crypto; CRC32 semantics unchanged |
| V7 Error Handling / Logging | yes (diagnostics only) | `killAndReap` and readiness failures must print captured child stderr; tests only, no production logging change; no secrets are introduced (tests use fixed keys/values and temp paths) |
| V12 Files / Resources | yes (test-only) | Dump files in `t.TempDir()`; 0600 fixture modes; `/proc/<pid>` reads are local to the test host; no path from untrusted input |

### Known Threat Patterns for this Stack

| Pattern | STRIDE | Standard mitigation |
|---------|--------|---------------------|
| Test process reads another process's `/proc` | Information disclosure (test-only) | Tests read their own children by pid; no secrets in perf fixtures; the hardware block contains no credentials |
| Port-band squatter/DoS | Denial of service (test-only) | Bind-probe + claim registry + bounded retry; a foreign holder causes a re-pick, never a false pass |
| Perf fixture with huge values exhausts test memory | Resource exhaustion (test-only) | Scales capped (10 k × 1 KiB, 1 k × 60 KiB); temp dirs cleaned by `t.TempDir` |
| Threshold/doc drift misleads operators | Repudiation/integrity (docs) | `TestPerformanceDoc` anchors the doc to the const threshold table and the commands |
| `make perf` run on a loaded host produces misleading numbers | Measurement integrity | Hardware block logs load; thresholds have documented headroom; rerun on a calm host; never relax a bound |

## Sources

### Primary (HIGH confidence)

- `internal/app/repl/bench_test.go` — two gates, fixtures, constants, `t.Parallel`/`nolint`, `ReportMetric` idiom.
- `internal/app/repl/core.go` — `StateStore` contract (`:393-436`), handlers (`:1749-1834`), `Advance`/`scanRetransmit` (`:2444-2460,2964-2975`), retry heap (`:3474-3532`), `NotifyLocalWrite`/`NotifyReplayedWrite`/`trackLocalWrite` (`:2325-2442`), `UnackedStats` (`:3346-3362`), pending scans (`:1659,3429-3452`).
- `internal/app/persist/engine.go`, `scheduler.go`, `codec.go`, `snapshot.go` — defaults/normalization (`engine.go:13-83`), debounce/deadline arming (`scheduler.go:176-199`), `runDump` (`scheduler.go:228-249`), `recordDump`/`DumpStats` (`scheduler.go:304-328`), retry ladder (`scheduler.go:15-25,279-302`), streaming encode (`codec.go:138-260`), replay clone (`snapshot.go:216-227`).
- `internal/app/persist/bench_test.go`, `internal/grpcapi/bench_test.go`, `internal/udprepl/bench_test.go`, `internal/testutil/bench.go` — benchmark shapes and metric names.
- `internal/app/persist/engine_test.go:644-693` and `backoff_test.go`/`timeout_test.go`/`engine_blocking_test.go` — synctest precedent and the deterministic burst cadence.
- `internal/config/config.go:76-112,233-272` — every default used in the model.
- `internal/app/storage/store.go:21,33,218-256,283-304` — value/key limits, `Version`/`Versions`/`Lookup`, install clone.
- `internal/persistfile/fsops.go:11-16,101-140`; `internal/persistfile/dumpfile.go:107-111,346-369` — 64 KiB buffered stream, fsync/rotate/rename/dir-fsync.
- `internal/metrics/metrics.go:446-560`; `internal/metrics/rss_linux.go:21-39` — dump families and current RSS.
- `internal/udprepl/loop.go` — notify queue (`:24-29`), replay seam (`:232-267,281-290`), `applyResults` (`:348-367`), `Run` (`:150-190`).
- `internal/grpcapi/server.go` — write handler/notify (`:193-241`), `LocalReplayNotify` (`:53-60`), `notifyReplayedWrite` (`:412-421`).
- `internal/stand/stand.go`, `stand_test.go`, `stand_red_test.go`, `persist_test.go`, `stand_metrics_test.go` — harness, ports, loadgen, metrics, `killAndReap`.
- `cmd/sup-kv/main_test.go`, `broadcast_probe_test.go`, `examples_test.go`, `readme_test.go`, `operations_test.go` — port pickers, readiness waits, doc tests, anchor pattern.
- `Makefile:1-78`; `.github/workflows/ci.yml:61-93`; `.github/workflows/scheduled.yml:64-198` — targets and CI contract.
- `go doc testing.B.ReportMetric`, `go doc testing.BenchmarkResult`, `go doc testing/synctest.Test`, `go doc testing/synctest.Wait`, `go doc testing.AllocsPerRun`, `go doc os.ErrProcessDone`, `go doc runtime/metrics` — stdlib contracts (tool-verified this session).
- Local probe: `go test -race` sets the `race` build tag (temp module probe, 2026-11-23).
- Local measurements: `make bench-smoke`, the 2 s dump and confirmation benchmarks, the two gates, the burst and unacked-drain stand scenarios (2026-11-23, host block in the Summary).
- `[CITED: https://man7.org/linux/man-pages/man5/proc_pid_io.5.html]` — `write_bytes` is "the number of bytes really sent to the storage layer"; `rchar`/`wchar` are syscall byte counts.

### Secondary (MEDIUM confidence)

- `docs/reviews/2026-11-19-v1.0/REVIEW.md` §4 (`:85-126`) and §5 (`R-01/R-02` at `:130-143`) — the review's stand measurements and severity/context.
- `docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md:1-80` — the reviewer's host, disk caveat and tables.
- `docs/reviews/2026-11-19-v1.0/evidence/review-engine.md:546-580` — P-06/ENG-12 memory measurements and the streaming fix rationale.
- `.planning/phases/03-transport/03-REVIEW.md:199-215` — IN-02 (uncovered whole-set scans) and IN-03/IN-04 (gate parallelism; fixture version domain).
- `.planning/phases/{04,05,06,07}/deferred-items.md` — the flake history and the routed owners.
- `docs/OPERATIONS.md:190-215` — the existing qualitative memory model.

### Tertiary (LOW confidence)

- None. No claim in this report rests on an unverified web source or on
  training memory alone.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all facilities are stdlib or already in the repo; no
  dependency decision is open.
- Architecture / measurement map: HIGH — every cited mechanism was read in the
  named file this session and, where relevant, exercised by a command this
  session.
- Deterministic gate design: HIGH for the mechanism (counts are exact and the
  seams exist); MEDIUM for the completeness of the `PendingVisits`
  instrumentation (a future un-instrumented scan would evade it; the paired
  `make perf` benchmark is the backstop).
- Threshold values: MEDIUM — starting values with a stated basis, not measured
  on target hardware; A1/A2/A4 flag the confirmation work.
- Flake cures: HIGH for the mechanisms; MEDIUM for the readiness-deadline
  value (30 s) and the port-band partition, which should be re-checked on the
  CI runner class.

**Research date:** 2026-11-23
**Valid until:** ~2026-12-23 for thresholds and host-specific numbers (fast
moving); ~2027-02-23 for the model, mechanisms and flake cures (stable until
the dump policy, the core's pending structures or the Go toolchain change).

