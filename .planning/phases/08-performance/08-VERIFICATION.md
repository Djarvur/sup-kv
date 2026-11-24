---
phase: 08-performance
verified: 2026-11-24T22:15:38Z
status: passed
score: 20/20 must-haves verified
covered_files:
  - .planning/phases/08-performance/08-01-PLAN.md
  - .planning/phases/08-performance/08-01-SUMMARY.md
  - .planning/phases/08-performance/08-02-PLAN.md
  - .planning/phases/08-performance/08-02-SUMMARY.md
  - .planning/phases/08-performance/08-03-PLAN.md
  - .planning/phases/08-performance/08-03-SUMMARY.md
  - .planning/phases/08-performance/08-04-PLAN.md
  - .planning/phases/08-performance/08-04-SUMMARY.md
  - .planning/phases/08-performance/08-05-PLAN.md
  - .planning/phases/08-performance/08-05-SUMMARY.md
  - .planning/phases/08-performance/08-06-PLAN.md
  - .planning/phases/08-performance/08-06-SUMMARY.md
  - .planning/phases/08-performance/08-07-PLAN.md
  - .planning/phases/08-performance/08-07-SUMMARY.md
  - .planning/phases/08-performance/08-08-PLAN.md
  - .planning/phases/08-performance/08-08-SUMMARY.md
  - .planning/phases/08-performance/08-09-PLAN.md
  - .planning/phases/08-performance/08-09-SUMMARY.md
  - .planning/phases/08-performance/08-10-PLAN.md
  - .planning/phases/08-performance/08-10-SUMMARY.md
  - .planning/phases/08-performance/08-11-PLAN.md
  - .planning/phases/08-performance/08-11-SUMMARY.md
  - Makefile
  - cmd/sup-kv/broadcast_probe_test.go
  - cmd/sup-kv/examples_test.go
  - cmd/sup-kv/main_test.go
  - cmd/sup-kv/performance_test.go
  - cmd/sup-kv/readme_test.go
  - docs/OPERATIONS.md
  - docs/PERFORMANCE.md
  - docs/SPEC-DEVIATIONS.md
  - internal/app/repl/bench_test.go
  - internal/app/repl/config_internal_test.go
  - internal/app/repl/core.go
  - internal/app/repl/pending_index_internal_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_dump_test.go
  - internal/grpcapi/bench_test.go
  - internal/grpcapi/replay_seam_internal_test.go
  - internal/grpcapi/server.go
  - internal/grpcapi/wait_integration_test.go
  - internal/stand/kill_and_reap_test.go
  - internal/stand/perf_test.go
  - internal/stand/stand.go
  - internal/testutil/bench.go
  - internal/testutil/bench_test.go
  - internal/testutil/perf.go
  - internal/testutil/perf_linux.go
  - internal/testutil/perf_linux_test.go
  - internal/testutil/perf_other.go
  - internal/testutil/perf_test.go
  - internal/testutil/ports.go
  - internal/testutil/ports_test.go
  - internal/testutil/race_off.go
  - internal/testutil/race_on.go
  - internal/udprepl/loop.go
  - internal/udprepl/replay_notify_internal_test.go
  - internal/udprepl/smoke_test.go
covered_digest: "v4:sha256:8d444915eb4eb58368364de1b10b6d476a32bea1457210009a038152f88e8013"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 8: Performance Verification Report

**Phase Goal:** Performance claims are measurements with documented conditions — the write-latency model, the burst bound, disk amplification, the memory model and absent-peer latency are all documented and standing-tested, not folklore.
**Verified:** 2026-11-24T22:15:38Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | SC1 — write-latency model documented for both profiles: sustained flow reaches `dump_max_wait` + dump time by design; fast profile (Debounce/MaxWait) has its disk cost; the "~10 ms" claim is tied to measurement conditions | ✓ VERIFIED | `docs/PERFORMANCE.md:23-93` states the sustained-flow model (`latency = U(0,MaxWait)+dump_time`, "by design"), the default/fast knob table, the fast profile's measured 2.82 ms mean/5.55 ms p99 under named in-memory/two-loopback/race-off conditions, and the ~1 dump/s vs ~20 dumps/s disk-cost paragraph linked to amplification. The standalone phrase is absent (grep), and `TestPerformanceDoc`=PASS guards every anchor and forbids the old wording. |
| 2 | SC2 — 1000 writes/s distinct-key burst is a standing tested bound; docs state what is and is not guaranteed | ✓ VERIFIED | `internal/stand/perf_test.go:174-242` (`TestPerfBurstDistinctKeys`) ran green in this verification: 1000.1 writes/s (min 990), drain 173.7 ms (bound 5 s), 10 000 distinct keys, 1 KiB. Frozen `TestRed_R02_burst` remains in `internal/stand/stand_red_test.go`; `TestEngine_burstBoundedConfirmation` exists (`internal/app/persist/engine_test.go:650`); doc guarantee/non-guarantee lists at `docs/PERFORMANCE.md:95-125`. |
| 3 | SC3 — amplification and memory measured, or the gap recorded: target-grade media (Spec open question #2) and memory model from stand measurements (PERSIST-06) | ✓ VERIFIED | `TestPerfAmplification` ran green: 15 dumps × 14 977 493 B = 224 686 080 B `write_bytes` Δ, bytes-per-dump 14 979 072 ≤ 16 475 242, amplification 14.36×, failures 0, real ext4 and tmpfs refusal in code. `TestPerfMemoryPoints` ran green for 1k×1 KiB, 10k×1 KiB, 1k×60 KiB (RSS 3.81× logical, dump peak 0.46×). The workstation-NVMe-vs-eMMC target-media gap is recorded in `docs/PERFORMANCE.md:147-171`. |
| 4 | SC4 — write latency with an absent peer and a growing unconfirmed set stays near live-peer latency, proven by a benchmark (same fix as XPORT-02) | ✓ VERIFIED | Per-thread index in `internal/app/repl/core.go` (`pendingByKey`, `PendingVisits`). `BenchmarkNotifyWriteCost/{absent,live}` and `TestPerfAbsentPeerWriteCost` exist; the perf gate ran green this pass: absent 2310.0 ns/op vs live 2361.0 ns/op, ratio 0.98 (bound 2.0), both 0.00 pending-visits/op at 10 000 unacked. Always-on zero-visit gates (`TestCore_notifyWriteCostFlat`, `TestCore_applyRemoteCostFlat`, `TestCore_unackedCostFlat`) pass in `make test`. |
| 5 | SC5 — `make perf` runs the stand suite with thresholds, and every published number carries a hardware description | ✓ VERIFIED | `make perf` executed independently on this host (2026-11-24T22:07:42Z, load 1.56 3.04 3.46): exit 0, 7/7 perf rows PASS, final threshold-free stand suite `ok 5.675 s`; hardware block logged by every row. Each of the five measured doc sections carries exactly one `hardware:` block and a producing command (`TestPerformanceDoc` enforces one per section, PASS). |
| 6 | One compiled threshold block; perf rows run only under `make perf`; no threshold literal duplicated in a consumer | ✓ VERIFIED | `internal/testutil/perf.go:44-82` is the only table; `grep` finds no threshold literals in the consumers and no threshold file. Without `SUPKV_PERF=1` the burst row SKIPs naming `make perf`; under `-race` the fast row SKIPs naming the race build (both observed). `TestPerfEnabled`/`TestPerfThresholdRelations` pass. |
| 7 | `make perf` shape: Linux-only guard, `.tmp/perf` dump base, `-p 1`, no `-race`, perf rows then the stand suite, CI untouched | ✓ VERIFIED | `Makefile:50-68` matches exactly; CI diff since the phase base is empty; `.github/workflows/ci.yml` still runs lint/test/build/vet/client-example only and `scheduled.yml:78` invokes `make bench-smoke`; no `perf` in CI. |
| 8 | Per-key pending index: fresh-key write/apply inspect 0 entries at any size; one-pair-per-key invariant; same observables; `PendingVisits` test-only counter; differential agreement | ✓ VERIFIED | Insert/delete sites `core.go:2441` and `core.go:3459-3463` (equality-guarded); `TestCore_pendingVisitsFreshKeyWrite/Apply/IdleAdvance/Structure`, `TestCore_pendingIndexInvariant/Differential/ProductionPaths` all pass under `-race`; `PendingVisits` has no metrics/adapter consumer (grep). |
| 9 | Version-only replay seam: gRPC and UDP adapter carry only `Version`; allocation gate; live `NotifyLocalWrite` still clones; nil-`ReplayNotify` fallback intact | ✓ VERIFIED | `internal/grpcapi/server.go:412-426` builds `storage.Record{Version: rec.Version}`; `internal/udprepl/loop.go:255-266` enqueues version-only while `NotifyLocalWrite` at :239 still uses `cloneLocalRecord`; `TestReplaySeamCarriesVersionOnlyRecord`, `TestNotifyReplayedWriteCarriesVersionOnly/AllocationBound` pass. |
| 10 | Deterministic cost gates and paired absent/live benchmark with exact vectors | ✓ VERIFIED | `internal/app/repl/bench_test.go:219-260` pins DataRequest = 1 `Version` + 1 `Lookup`, DataSync = 1 `Observe` + 16 `Version` at 1 000 and 50 000 keys; idle/fresh-key paths exactly 0 visits; repeated-key write exactly 2; `TestPerfAbsentPeerWriteCost` reads `PerfAbsentPeerRatio`. All pass. |
| 11 | Shared port allocator: one registry, claim-while-held, `network:port` keying, four disjoint bands, sysctl assertion, bounded retry, all four packages adopted | ✓ VERIFIED | `internal/testutil/ports.go:35-56` bands `[28100,29100) [29100,30100) [30100,31100) [31100,32100)`, 32 candidates, `ip_local_port_range` check with non-Linux skip; adopters in `stand.go`, `cmd/sup-kv` (both tests), `udprepl/smoke_test.go`, `grpcapi/wait_integration_test.go`; the private registries are gone. Band/sysctl/claim tests pass. |
| 12 | synctest dump↔replication harness with no wall-clock settling; one bubble per run; unchanged scenario claims; chunker capacity teeth | ✓ VERIFIED | `internal/app/repl/sim_dump_test.go` imports `testing/synctest`, has no `time.Sleep`/wall-clock cap, runs each scenario in its own subtest-body bubble, asserts debounce instant, failure notify, ladder, shrunk-limit skip, same-seed equality; `TestScenario_dumpNotify -count=100` passed (1.384 s). `TestChunkSyncEntries_capacityFollowsEntries` passes. |
| 13 | Two-profile confirmation benchmark through one body; fast-profile executable bound; bench/bench-smoke include core benchmarks | ✓ VERIFIED | `internal/grpcapi/bench_test.go:88-132` defines fast (1 ms/50 ms, in-memory) and default (50 ms/1 s, real `persistfile`) arms; `TestPerfFastProfileLatency` ran green: mean 2.762 ms (bound 25 ms), p99 5.682 ms (bound 100 ms), 825 samples. `Makefile` bench and bench-smoke both list `./internal/app/repl/`; `bench-smoke` keeps `-benchtime=1x`. |
| 14 | Harness diagnostics: exited child diagnosed, classified readiness budgets, spawn semaphore, serialized doc walkthroughs | ✓ VERIFIED | `internal/stand/stand.go:469-500` handles `os.ErrProcessDone` with exit diagnosis; `cmd/sup-kv/main_test.go:78-85` adds `processServingDeadline` 30 s / `logEventDeadline` 30 s and `spawnSemSlots = 8` release-after-reap; `TestExamplesDoc`/`TestReadmeQuickstart` have no `t.Parallel`. `kill_and_reap_test.go` and the readiness tests pass. |
| 15 | Measurement ledger: per-section hardware block + command, drift guard, S-FAIL-15 closure, OPERATIONS summary/link, PROJECT claim conditioned | ✓ VERIFIED | `docs/PERFORMANCE.md` has six mandated sections, five `hardware:` blocks, five commands, and threshold literals derived from Go constants; `TestPerformanceDoc`, `TestPerformanceDocFamilies`, `TestSpecDeviationsBurstRow` pass uncached. S-FAIL-15 says `fixed-in-v1.1(PERF-02)`, cites the three tests and 990/5 s. `docs/OPERATIONS.md:190-200` links the ledger; `.planning/PROJECT.md:43` carries the conditions and no "~10 ms" phrase. |
| 16 | Phase gate: frozen hashes, mutation seds, red-runner 7/7 with both mutants killed, standing gates green, config pins, CI unchanged, deterministic gates in `make test` | ✓ VERIFIED | `git hash-object` reproduces 519789bd…, 573fed68…, add45b81…; sed targets exist at `core.go:2535` and `store.go:253,274`; `bash scripts/red-runner` re-run this pass: exit 0, "7 expectation row(s) checked, 0 error(s)", both mutants killed. `make lint`, `make test` (15 packages ok), `make buf-lint`, `make buf-breaking`, `make client-example`, `GOOS=darwin go build/vet` all exit 0. `.golangci.yml` 1-64 = a121e046…, first `go.mod` require block = 9679f143…, workflows diff empty. |
| 17 | Prohibition — frozen files must not be edited | ✓ VERIFIED | All three content hashes are byte-identical to the planning-time literals (row 16). |
| 18 | Prohibition — no `perf` or threshold in CI; `bench-smoke` stays threshold-free | ✓ VERIFIED | No `perf` in `.github/workflows/`; workflows unchanged since phase base; `bench-smoke` has no thresholds. |
| 19 | Prohibition — thresholds not parsed from env, no threshold file, no `var` table, helper stdlib-only | ✓ VERIFIED | Only env reads are the `SUPKV_PERF` gate and `SUPKV_PERF_DUMP_DIR`; no threshold file exists; thresholds are `const`; `go list` imports show only stdlib/own module (depguard `sup-kv-app-no-io` lint passes). |
| 20 | Prohibition — no new direct runtime dependency | ✓ VERIFIED | First `go.mod` require block hashes to 9679f1431d1815116871a4f7af79e727283a2875 exactly as planned; only grpc and protobuf remain. |

**Score:** 20/20 truths verified (0 present, behavior-unverified)

### Required Artifacts

All artifacts declared across the 11 plans exist and are substantive and wired. Representative rows:

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/testutil/perf.go` | one threshold table, gate, hardware block rule | ✓ VERIFIED | 137 lines; consts + `PerfEnabled`/`RequirePerf`/`perfDumpDir`; consumed by all four perf test packages. |
| `internal/testutil/perf_linux.go`, `perf_other.go`, `race_on.go`, `race_off.go` | hardware block, fs/device probes, build-tag race label | ✓ VERIFIED | 298/41/8/8 lines; `/proc` + `Statfs` + mountinfo + rotational probes, tmpfs/ramfs detection. |
| `internal/stand/perf_test.go` | burst, default latency, amplification, memory rows, drift guard | ✓ VERIFIED | 823 lines; five tests; all four perf rows ran green this pass. |
| `internal/app/repl/bench_test.go`, `retransmit_test.go` | cost gates, counting double, paired benchmark | ✓ VERIFIED | 30 KB; exact vectors, 0/2 visit counts, `BenchmarkNotifyWriteCost/{absent,live}`. |
| `internal/app/repl/core.go` | `pendingByKey`, `PendingVisits` | ✓ VERIFIED | index insert beside `c.pending[pk]`, equality-guarded delete; counter not exported as a metric. |
| `internal/app/repl/pending_index_internal_test.go` | RED-era gates + invariants | ✓ VERIFIED | 28 KB; fresh-write/apply/idle zero-visit gates, invariant/differential/production-path proofs. |
| `internal/app/repl/sim_dump_test.go`, `config_internal_test.go` | synctest harness, chunker teeth | ✓ VERIFIED | synctest bubbles; cap() assertions; no wall-clock settling. |
| `internal/grpcapi/server.go`, `internal/udprepl/loop.go`, replay tests | version-only replay seam | ✓ VERIFIED | version-only construction and enqueue; allocation ceiling <1 KiB for 60 KiB records. |
| `internal/testutil/ports.go`, `ports_test.go`, adopters | shared allocator | ✓ VERIFIED | bands, claim-while-held, sysctl assertion; four packages delegate. |
| `internal/grpcapi/bench_test.go` | two-profile benchmark + fast gate | ✓ VERIFIED | shared body, both profiles, per-arm conditions; fast gate green. |
| `internal/stand/stand.go`, `kill_and_reap_test.go`, `cmd/sup-kv/main_test.go` | diagnostics/budgets/serialization | ✓ VERIFIED | exited-child diagnosis, semaphore, serialized doc walkthroughs. |
| `docs/PERFORMANCE.md`, `cmd/sup-kv/performance_test.go` | ledger + drift guard | ✓ VERIFIED | 269-line ledger with per-section block/command; three guards pass. |
| `docs/OPERATIONS.md`, `docs/SPEC-DEVIATIONS.md`, `.planning/PROJECT.md` | ripple + registry closure | ✓ VERIFIED | memory summary/link; S-FAIL-15 row closed; conditioned PROJECT claim. |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `Makefile` perf target | `internal/testutil/perf.go` | only place `SUPKV_PERF=1`/`SUPKV_PERF_DUMP_DIR` are set | ✓ WIRED | `Makefile:61-68`; thresholds are the compiled consts. |
| `internal/stand/perf_test.go` | `internal/stand/stand_red_test.go` | `redRunBurst`, constants read-only | ✓ WIRED | Called at perf_test.go:205; frozen file hash unchanged. |
| `internal/stand/perf_test.go` | `/proc/<pid>/io`, `/proc/<pid>/status` | amplification write_bytes and RSS | ✓ WIRED | `perfProcIO`/`perfProcStatus`; used in the measured rows (run logged real values). |
| `internal/app/repl/core.go trackLocalWrite` | `pendingByKey` | single insert site | ✓ WIRED | `core.go:2440-2441`. |
| `internal/app/repl/core.go removePending` | `pendingByKey` | equality-guarded chokepoint | ✓ WIRED | `core.go:3459-3463`. |
| `internal/app/repl/pending_index_internal_test.go` | `PendingVisits` | delta assertions | ✓ WIRED | Tests pass and the counter has no production consumer. |
| `internal/app/repl/sim_dump_test.go` | `testing/synctest` | bubble per scenario, `synctest.Wait` barrier | ✓ WIRED | Import + `synctest.Test` in each scenario body. |
| `internal/app/repl/config_internal_test.go` | `core.chunkSyncEntries` | cap() of returned chunks | ✓ WIRED | Test in package `repl`; passes. |
| `internal/grpcapi/server.go notifyReplayedWrite` | `core.trackLocalWrite` | version is the only consumed field | ✓ WIRED | Version-only record constructed at server.go:425. |
| `internal/udprepl/loop.go NotifyReplayedWrite` | adapter notify channel | enqueued version-only `localWrite` | ✓ WIRED | loop.go:266; direct channel read asserted by test. |
| `internal/app/repl/bench_test.go` | `retransmit_test.go` `stateStore` | counting double wraps real seam | ✓ WIRED | `countingStore`/`stateStore` definitions; exact vectors pass. |
| `internal/grpcapi/bench_test.go` default arm | `internal/persistfile` | real dump store in temp dir | ✓ WIRED | `dumps = persistfile.New(...)` at bench_test.go:164. |
| `internal/stand/stand.go scenarioUDPPort` | `internal/testutil/ports.go` | `UDPPort(StandBand())` | ✓ WIRED | stand.go:169-180. |
| `cmd/sup-kv` tests | `internal/testutil/ports.go` | `CmdBand` wrappers | ✓ WIRED | main_test.go:169,176; broadcast_probe_test.go:46. |
| `internal/udprepl/smoke_test.go`, `internal/grpcapi/wait_integration_test.go` | `internal/testutil/ports.go` | `testing.TB` allocator | ✓ WIRED | smoke_test.go:53; wait_integration_test.go:68. |
| `internal/stand/stand.go reapAfterKill` | stored Wait result | reaper stores before closing `exited` | ✓ WIRED | stand.go:302,311; diagnosis read once. |
| `cmd/sup-kv/main_test.go awaitNodeServing` | `smokeNode.exited` | closed channel ⇒ immediate diagnosis | ✓ WIRED | main_test.go:436-465; tests assert first-poll diagnosis. |
| `cmd/sup-kv/performance_test.go` | `internal/testutil/perf.go` | threshold literals formatted from consts | ✓ WIRED | `thresholdAnchors()`; guard passes. |
| `docs/PERFORMANCE.md` | perf rows in the tree | every cited test/benchmark exists | ✓ WIRED | `go test -list '^TestPerf'` over three packages lists all six rows + drift guard. |
| `docs/OPERATIONS.md` | `docs/PERFORMANCE.md` | one memory-section link, no duplicated figures | ✓ WIRED | OPERATIONS.md:200. |
| `Makefile bench-smoke` | `.github/workflows/scheduled.yml` | workflow invokes target by name | ✓ WIRED | scheduled.yml:78. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `TestPerfDefaultProfileLatency` | p50/p99 | real binary over real gRPC + real `persistfile` dump, open-loop 1000 w/s × 15 s | Yes — this run: 583.0 ms / 1.0738 s at 1000.1 w/s | ✓ FLOWING |
| `TestPerfBurstDistinctKeys` | achieved rate / drain | frozen `redRunBurst` over the real binary; `supkv_replication_unacked_records` from `/metrics` | Yes — 1000.1 w/s, drain 173.7 ms, 10 000 keys | ✓ FLOWING |
| `TestPerfAmplification` | bytes-per-dump / amplification | `supkv_dumps_total` + real dump file stat + `/proc/<pid>/io` write_bytes | Yes — 15 dumps, 14 979 072 B/dump, 14.36× | ✓ FLOWING |
| `TestPerfMemoryPoints` | RSS/VmHWM/peak | `/metrics` `process_resident_memory_bytes` + `/proc/<pid>/status` on the named node's pid | Yes — 3 points, 60 KiB row 3.81×/0.46× | ✓ FLOWING |
| `TestPerfAbsentPeerWriteCost` | ns/op, visits/op | `testing.Benchmark` over the real core with 10 000 pending 1 KiB records | Yes — 2310 vs 2361 ns/op, 0.98 ratio | ✓ FLOWING |
| `TestPerfFastProfileLatency` | mean/p99 | `BenchmarkConfirmationLatency` fast arm, in-memory dump, two loopback adapters | Yes — 2.762 ms / 5.682 ms, 825 samples | ✓ FLOWING |
| `internal/testutil/perf_linux.go HardwareBlock` | CPU/RAM/load/disk | `/proc/cpuinfo`, `/proc/meminfo`, `/proc/loadavg`, `/proc/self/mountinfo`, `/sys/block/*/queue/rotational` | Yes — real host block logged on every row | ✓ FLOWING |
| `docs/PERFORMANCE.md` rows | published figures | transcribed from two green `make perf` runs at commit `9cdc0f1`, each with block and command | Yes — independently corroborated by this pass's run (same amplification 14.36×, same bytes-per-dump, matching p50/p99/memory within run-to-run noise) | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Operator target runs the perf rows and the stand suite with thresholds | `make perf` (load 1.56→2.43, performance governor, 16 CPU) | exit 0; 7 PASS perf rows; `ok internal/stand 117.443 s`; second threshold-free `ok internal/stand 5.675 s` | ✓ PASS |
| Burst bound | row output | achieved 1000.1 w/s (min 990), drain 173.746 ms (bound 5 s) | ✓ PASS |
| Default-profile latency | row output | p50 583.025 ms (bound 750 ms), p99 1.073769 s (bound 1.5 s) | ✓ PASS |
| Amplification | row output | 14.36×; bytes-per-dump 14 979 072 ≤ 16 475 242; failures 0 | ✓ PASS |
| Memory points | row output | 1k/10k×1 KiB and 1k×60 KiB; 60 KiB RSS 3.81× (bound 12×+256 MiB), peak 0.46× (bound 2×) | ✓ PASS |
| Absent vs live peer | row output | 2310.0 vs 2361.0 ns/op, ratio 0.98 (bound 2.0) | ✓ PASS |
| Fast profile | row output | mean 2.762 ms (bound 25 ms), p99 5.682 ms (bound 100 ms) | ✓ PASS |
| Perf rows skip outside the operator target | `env -u SUPKV_PERF go test -run '^TestPerfBurstDistinctKeys$'` | SKIP naming `make perf` | ✓ PASS |
| Perf rows skip under race | `SUPKV_PERF=1 go test -race -run '^TestPerfFastProfileLatency$'` | SKIP naming the race build | ✓ PASS |
| Deterministic dumpNotify harness ×100 | `go test -race -count=100 -run '^TestScenario_dumpNotify$' ./internal/app/repl/` | ok 1.384 s | ✓ PASS |
| Lint gate | `make lint` | config verify + `golangci-lint run ./...` exit 0 | ✓ PASS |
| Standing test gate | `make test` | exit 0, 15 packages ok, no FAIL (repl 60.688 s) | ✓ PASS |
| Proto gates | `make buf-lint`, `make buf-breaking` | exit 0 each (breaking ran the real comparison) | ✓ PASS |
| Public-client gate | `make client-example` | exit 0 (build, vet, tidy clean) | ✓ PASS |
| Darwin cross cell | `GOOS=darwin go build ./...`, `go vet ./...` | exit 0 each | ✓ PASS |
| Expected-red/mutation gate | `bash scripts/red-runner` | exit 0; "7 expectation row(s) checked, 0 error(s)"; both mutants killed; control passed | ✓ PASS |
| Doc drift guards | `go test -count=1 -run 'TestPerformanceDoc|TestPerformanceDocFamilies|TestSpecDeviationsBurstRow' ./cmd/sup-kv/` | 3/3 PASS | ✓ PASS |

### Probe Execution

No probe scripts are declared by the phase plans and no conventional `scripts/*/tests/probe-*.sh` exists in this tree; Step 7c is not applicable. The phase's own runnable gate (`scripts/red-runner`) was executed and is recorded above.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| PERF-01 | 08-01, 08-03, 08-07, 08-08, 08-10, 08-11 | Write-latency model documented for both profiles | ✓ SATISFIED | `docs/PERFORMANCE.md:23-93`; `TestPerfDefaultProfileLatency`, `TestPerfFastProfileLatency`; `TestPerformanceDoc`; model `by design` and conditioned fast numbers verified. |
| PERF-02 | 08-01, 08-06, 08-09, 08-10, 08-11 | 1000 writes/s distinct-key burst is a tested bound | ✓ SATISFIED | `TestRed_R02_burst` (frozen) + `TestPerfBurstDistinctKeys` (green this pass) + `TestEngine_burstBoundedConfirmation`; guarantee/non-guarantee doc text. |
| PERF-03 | 08-07, 08-10, 08-11 | Amplification measured on target-grade media or gap recorded | ✓ SATISFIED | `TestPerfAmplification` measured on NVMe/ext4 (14.36×, per-dump bound green); target-media gap recorded in `docs/PERFORMANCE.md:165-171`. (One inaccurate pointer — see Warnings.) |
| PERF-04 | 08-07, 08-10, 08-11 | Memory model documented from stand measurements | ✓ SATISFIED | Formula + three measured points at `docs/PERFORMANCE.md:179-222`; `TestPerfMemoryPoints` green; OPERATIONS links without duplicating. |
| PERF-05 | 08-02, 08-04, 08-05, 08-10, 08-11 | Absent-peer latency near live-peer, proven by a benchmark | ✓ SATISFIED | `BenchmarkNotifyWriteCost/{absent,live}` + `TestPerfAbsentPeerWriteCost` (0.98 ratio vs 2.0) plus always-on zero-visit gates. |
| PERF-06 | 08-01, 08-03, 08-06, 08-08, 08-09, 08-10, 08-11 | `make perf` with thresholds; published numbers carry hardware description | ✓ SATISFIED | `make perf` green 7/7 this pass; five sections each with exactly one hardware block and command; Linux guard, `.tmp/perf`, `-p 1`, no race, stand suite. |

No ORPHANED requirements: REQUIREMENTS.md maps exactly PERF-01..PERF-06 to Phase 8, and every one is claimed by at least one plan.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts (7/7; `check.decision-coverage-verify` output: `total: 7, honored: 7, not_honored: []`).

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
| --------- | ---------- | ------ | ------- | -------- | --------------- | ------- |
| `internal/stand/perf_test.go` | PERF-01..04, 06 | 5 | perf-gated by design (all ran under `make perf`) | No | Value (p50/p99 vs bounds, amplification, ratio, RSS) | PASS |
| `internal/app/repl/bench_test.go` | PERF-05 | 6+ | perf-gated row; gates always on | No | Exact counts + ratio bound | PASS |
| `internal/grpcapi/bench_test.go` | PERF-01, 06 | 2 | perf-gated row by design | No | Mean/p99 vs bounds | PASS |
| `internal/app/repl/pending_index_internal_test.go` | PERF-05 | 7 | 0 | No | Invariant/differential vs naive scan | PASS |
| `internal/grpcapi/replay_seam_internal_test.go`, `internal/udprepl/replay_notify_internal_test.go` | PERF-05 | 3 | 0 | No | Record shape + allocation bytes/op | PASS |
| `cmd/sup-kv/performance_test.go` | PERF-01..06 | 3 | 0 | No (doc guard reads authored docs/constants; numbers came from `make perf`, not the test) | Threshold/statement presence | PASS |
| `internal/testutil/perf*_test.go`, `ports_test.go`, `bench_test.go` | PERF-02, 06 | 12+ | 0 (race branch is conditional) | No | Gate/band/percentile behavior | PASS |

**Disabled tests on requirements:** 0 unconditional skips — the perf rows skip only outside the operator target and all were executed there. **Circular patterns detected:** 0. **Insufficient assertions:** 0.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers `TBD`/`FIXME`/`XXX` across all 36 phase-changed files | None | `grep` finds none. |
| — | — | Stub patterns (`return nil`/empty data) in production artifacts | None | Hits are error paths, not stubs; no user-visible value is hardcoded. |
| `docs/PERFORMANCE.md` | 169 | Claims `docs/SPEC-DEVIATIONS.md` records the amplification target-media gap, but the registry has no such row (grep for media/amplification/eMMC/open question finds none) | ⚠️ WARNING | Documentation cross-reference accuracy only. The gap itself is recorded in `docs/PERFORMANCE.md:147-171`, so SC3/PERF-03 are satisfied; a reader following the pointer lands on a doc without it. Recommend either adding the gap to the registry or deleting the pointer (one-line doc fix, not a phase blocker). |

No 🛑 BLOCKER anti-patterns.

### Human Verification Required

N/A — this is an infrastructure/tooling phase with no user-facing elements; every behavior-dependent truth was exercised by `make perf`, `make test`, the red-runner, or the doc guards. No ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths and no abstained non-inferable checks.

### Deferred Items

None. No identified gap is deferred to a later phase; Phase 9's documentation scope does not absorb any Phase 8 performance deliverable.

### Gaps Summary

No gaps. All five roadmap success criteria and all six PERF requirements are observably true in the committed tree, and the phase's headline behavior — `make perf` with thresholds and hardware blocks — was reproduced in this verification run rather than accepted from the summaries.

One non-blocking warning remains: the ledger's target-media-gap paragraph points at `docs/SPEC-DEVIATIONS.md`, which does not in fact carry that record. The record lives in `docs/PERFORMANCE.md` itself, so the phase goal is achieved; the pointer should be corrected or the registry entry added.

---

_Verified: 2026-11-24T22:15:38Z_
_Verifier: Claude (gsd-verifier)_
