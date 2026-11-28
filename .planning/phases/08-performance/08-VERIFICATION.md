---
phase: 08-performance
verified: 2026-11-27T17:52:01Z
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
covered_digest: "v4:sha256:0663f861b87a09504fd4d6ffac7e62fea63d4cdc89f84c7afce3354071f1348b"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 20/20
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 8: Performance Verification Report

**Phase Goal:** Performance claims are measurements with documented conditions — the write-latency model, the burst bound, disk amplification, the memory model and absent-peer latency are all documented and standing-tested, not folklore.
**Verified:** 2026-11-27T17:52:01Z
**Status:** passed
**Re-verification:** Yes — stale-fingerprint regeneration. The 2026-11-25 report (passed, 20/20) went stale because its covered file `docs/PERFORMANCE.md` was afterwards changed by commit `e16cdd8` (docs(08-10): point the amplification gap at the ledger, not a missing registry row). Not a gap-closure round: the previous status was passed with no gaps; its one non-blocking warning (the ledger pointed at a SPEC-DEVIATIONS row that does not exist) is exactly what `e16cdd8` fixed, and this run confirms the fix.

**Evidence scope notes (this host):**
1. The host runs powersave-throttled today. `make perf` was not re-run this pass; its wall-clock figures are not re-derived here. Instead: git shows the production sources and the perf-row test sources are byte-identical since commit `9cdc0f1`, where two consecutive green `make perf` runs produced the ledger (only `docs/PERFORMANCE.md` itself and the doc-guard test `cmd/sup-kv/performance_test.go` changed afterwards). The recorded green runs therefore remain valid evidence for this tree, and every deterministic gate that stands in `make test` was re-run green first-hand today.
2. The red gate's R-01 row misses locally — the adjudicated Phase-3 wall-clock class (03-12 halt record, 03-13 resolution, `4a83919` full-speed flip evidence, scheduled `red-r01-x20` CI acceptance). Observed first-hand this pass; treated as a host environment artifact, not a Phase-8 regression.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | SC1 — write-latency model documented for both profiles: sustained flow reaches `dump_max_wait` + dump time by design; fast profile (Debounce/MaxWait) has its disk cost; the "~10 ms" claim is tied to measurement conditions | ✓ VERIFIED | Read in full this pass: `docs/PERFORMANCE.md:23-93` states the sustained-flow model (`latency = U(0,MaxWait)+dump_time`, "by design"), the default/fast knob table, the fast profile's measured 2.82 ms mean/5.55 ms p99 under its named conditions (in-memory dump, two loopback adapters, race off, load logged), the single-writer ceiling ≈14 writes/s, and the disk-cost paragraph (≈1 dump/s vs ≈20 dumps/s) cross-linked to amplification. `grep "10 ms"` over the ledger and PROJECT.md: 0 hits — no standalone "~10 ms" claim. `TestPerformanceDoc` PASS (2.244 s, this pass). |
| 2 | SC2 — 1000 writes/s distinct-key burst is a standing tested bound; docs state what is and is not guaranteed | ✓ VERIFIED | `internal/stand/perf_test.go` has `TestPerfBurstDistinctKeys`; frozen `TestRed_R02_burst` hash reproduces (`573fed68…`); `TestEngine_burstBoundedConfirmation` exists. Today's red-runner ran the R-02 burst row green under `SUPKV_RED_TESTS=1` (7 rows checked, only R-01 errored). Doc guarantee/non-guarantee lists read at `docs/PERFORMANCE.md:104-115` (990 writes/s, 5 s drain, and the explicit not-guaranteed list). Recorded `make perf` figures: 1000.1 w/s achieved, drain 305.13 ms, zero failed writes. |
| 3 | SC3 — amplification and memory measured, or the gap recorded: target-grade media (Spec open question #2) and memory model from stand measurements (PERSIST-06) | ✓ VERIFIED | `TestPerfAmplification`/`TestPerfMemoryPoints` exist in `internal/stand/perf_test.go`; recorded green rows (14.36× amplification, bytes-per-dump within the 1.1× bound, 3 memory points, 60 KiB row 3.73× RSS / 0.54× peak inside both tripwires). The target-media gap is recorded in the ledger itself at `docs/PERFORMANCE.md:165-172` with the corrected sentence: the registry "indexes Spec statements, so it carries no separate row for an open question" — verified TRUE (grep for eMMC/amplification/open-question/target-media in `docs/SPEC-DEVIATIONS.md`: 0 hits). The previous report's pointer warning is resolved by `e16cdd8`. |
| 4 | SC4 — write latency with an absent peer and a growing unconfirmed set stays near live-peer latency, proven by a benchmark (same fix as XPORT-02) | ✓ VERIFIED | Per-thread index in `internal/app/repl/core.go` (`pendingByKey` insert `:2441`, equality-guarded delete `:3461`, `PendingVisits()` `:3362`). `BenchmarkNotifyWriteCost` (`bench_test.go:553`) and `TestPerfAbsentPeerWriteCost` (`:655`) exist; always-on zero-visit gates (`TestCore_notifyWriteCostFlat:433`, `TestCore_applyRemoteCostFlat:473`, `TestCore_unackedCostFlat:398`) ran green in today's `make test`. Recorded perf row: absent 2249 ns/op vs live 8598 ns/op, ratio 0.26 ≤ 2.0, both arms 0.00 pending-visits/op. |
| 5 | SC5 — `make perf` runs the stand suite with thresholds, and every published number carries a hardware description | ✓ VERIFIED | `Makefile` perf target read this pass (`perf:` at :60-68): Linux guard, `SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(CURDIR)/.tmp/perf`, `-count=1 -p 1 -timeout 30m -v` perf rows, then the threshold-free stand suite. The ledger carries five `hardware:` blocks and five producing commands, one per measured section (`docs/PERFORMANCE.md:73-77, 121-125, 174-178, 219-223, 245-249`), plus the fast arm's differing load logged inline (:64-65); `TestPerformanceDocFamilies` asserts this per section and PASSes. Two green `make perf` runs recorded at `9cdc0f1` (ledger publication rule, :8-15). (Not re-run — evidence-scope note 1.) |
| 6 | One compiled threshold block; perf rows run only under `make perf`; no threshold literal duplicated in a consumer | ✓ VERIFIED | `internal/testutil/perf.go` is the only table (25 `Perf*` symbols; consts). `grep` for threshold constants outside `perf.go`/tests: 0 hits. This pass observed both skips first-hand: without `SUPKV_PERF` the burst row SKIPs with "perf: operator-facing threshold run — invoke `make perf` (SUPKV_PERF=1, no -race)"; under `-race` with `SUPKV_PERF=1` the fast row SKIPs. `TestPerformanceDoc` formats doc literals from the Go consts, so drift fails `make test` (PASS today). |
| 7 | `make perf` shape: Linux-only guard, `.tmp/perf` dump base, `-p 1`, no `-race`, perf rows then the stand suite, CI untouched | ✓ VERIFIED | `Makefile:60-68` matches exactly (read this pass); `.github/workflows/` diff since the gate commit `dc51e8c` is empty; `grep "make perf\|SUPKV_PERF"` over workflows: 0 hits; `scheduled.yml` still invokes `make bench-smoke` (3 hits). |
| 8 | Per-key pending index: fresh-key write/apply inspect 0 entries at any size; one-pair-per-key invariant; same observables; `PendingVisits` test-only counter; differential agreement | ✓ VERIFIED | Insert site `core.go:2441`, equality-guarded removal chokepoint `core.go:3461`, `PendingVisits()` `core.go:3362`. The zero-visit/invariant/differential/production-path gates ran green in today's `make test` (repl package 54.367 s fresh, exit 0). `PendingVisits` has no metrics/adapter consumer (grep). |
| 9 | Version-only replay seam: gRPC and UDP adapter carry only `Version`; allocation gate; live `NotifyLocalWrite` still clones; nil-`ReplayNotify` fallback intact | ✓ VERIFIED | `internal/grpcapi/server.go:425` builds `storage.Record{Version: rec.Version}`; `internal/udprepl/loop.go:261` enqueues version-only while `:239` keeps `cloneLocalRecord` for the live write path (read this pass). Shape and allocation gates ran in today's `make test` (grpcapi/udprepl ok). |
| 10 | Deterministic cost gates and paired absent/live benchmark with exact vectors | ✓ VERIFIED | `internal/app/repl/bench_test.go` carries the counting-double vectors (`TestCore_keyCountCostFlat` family at :398/:433/:473), the paired benchmark at :553 and the perf-gated ratio test at :655. All always-on gates green in today's `make test`. |
| 11 | Shared port allocator: one registry, claim-while-held, `network:port` keying, four disjoint bands, sysctl assertion, bounded retry, all four packages adopted | ✓ VERIFIED | `internal/testutil/ports.go:36-40` bands `[28100,29100) [29100,30100) [30100,31100) [31100,32100)` above the `[27900,28100)` doc-port reserve; adopters confirmed present: `internal/stand/stand.go`, `cmd/sup-kv/main_test.go`, `cmd/sup-kv/broadcast_probe_test.go`, `internal/udprepl/smoke_test.go`, `internal/grpcapi/wait_integration_test.go`. Band/sysctl tests ran in today's `make test` (testutil ok). |
| 12 | synctest dump↔replication harness with no wall-clock settling; one bubble per run; unchanged scenario claims; chunker capacity teeth | ✓ VERIFIED | `internal/app/repl/sim_dump_test.go`: 16 `synctest` references, 0 `time.Sleep` (grep this pass); each scenario runs in its own subtest-body bubble. Chunker teeth in `config_internal_test.go`. Both ran green in today's `make test`. |
| 13 | Two-profile confirmation benchmark through one body; fast-profile executable bound; bench/bench-smoke include core benchmarks | ✓ VERIFIED | `internal/grpcapi/bench_test.go` defines the `fast` arm (1 ms/50 ms, in-memory) and the default arm (production `persistfile`), conditions stated in code (:10-14, :81-102); `TestPerfFastProfileLatency` exists and SKIPs outside `make perf` (observed this pass). Recorded green row: mean 2.82 ms / p99 5.55 ms under its named conditions. `Makefile` bench and bench-smoke both include `./internal/app/repl/`; bench-smoke runs `-benchtime=1x`. |
| 14 | Harness diagnostics: exited child diagnosed, classified readiness budgets, spawn semaphore, serialized doc walkthroughs | ✓ VERIFIED | `internal/stand/stand.go:477` handles `os.ErrProcessDone` with exit diagnosis; `cmd/sup-kv/main_test.go:78` `processServingDeadline = 30s` and the `spawnSemSlots` release-after-reap semaphore (:84) are present. All green in today's `make test` (cmd/sup-kv ok). |
| 15 | Measurement ledger: per-section hardware block + command, drift guard, S-FAIL-15 closure, OPERATIONS summary/link, PROJECT claim conditioned | ✓ VERIFIED | `docs/PERFORMANCE.md` read in full this pass: six mandated sections, five hardware blocks + five commands, thresholds named as constants; `TestPerformanceDoc`, `TestPerformanceDocFamilies`, `TestSpecDeviationsBurstRow` PASS uncached (2.244 s today). `docs/SPEC-DEVIATIONS.md:210` S-FAIL-15 = `fixed-in-v1.1(PERF-02)` citing the three tests and the 990 w/s / 5 s bounds. `docs/OPERATIONS.md:200` links `PERFORMANCE.md#memory-model`. `.planning/PROJECT.md:43` carries the fast-profile conditions (1 ms debounce / 50 ms max-wait, in-memory dump, two loopback adapters, race off) and no "~10 ms" phrase. |
| 16 | Phase gate: frozen hashes, mutation seds, red-runner with both mutants killed, standing gates green, config pins, CI unchanged, deterministic gates in `make test` | ✓ VERIFIED | `git hash-object` reproduces all three frozen hashes exactly this pass (519789bd…, 573fed68…, add45b81…). Both mutation sed targets exist (`core.go` 1 hit; `store.go` 2 hits). `make lint`, `make test` (19 package rows ok, exit 0), `make buf-lint`, `make buf-breaking`, `make client-example`, `GOOS=darwin go build`/`go vet` all exit 0 this pass. `.golangci.yml` lines 1-64 hash reproduces (`a121e046…`). Red-runner this pass: 7 rows checked, 6 green, R-01 failed locally — the adjudicated throttled-host class (evidence-scope note 2); the runner halted at the expectation stage, so today's mutation-stage evidence is the same-day standalone reproduction recorded in the Phase-3 regeneration (`03-VERIFICATION.md`: control passes, both mutants killed) on mutation-target files git-identical since this phase's gate. Gate-time record (7/7, both mutants killed, green `make perf`) stands at `dc51e8c`. |
| 17 | Prohibition — frozen files must not be edited | ✓ VERIFIED | All three content hashes byte-identical to the planning-time literals (row 16, this pass). |
| 18 | Prohibition — no `perf` or threshold in CI; `bench-smoke` stays threshold-free | ✓ VERIFIED | Workflows diff since gate commit empty; no `perf`/`SUPKV_PERF` in `.github/workflows/` (grep this pass). |
| 19 | Prohibition — thresholds not parsed from env, no threshold file, no `var` table, helper stdlib-only | ✓ VERIFIED | The only env reads in `internal/testutil` are the `SUPKV_PERF` gate and `SUPKV_PERF_DUMP_DIR` (grep this pass); thresholds are `const`; `make lint` (depguard rules incl. `sup-kv-app-no-io`) exit 0 today. |
| 20 | Prohibition — no new direct runtime dependency | ✓ VERIFIED | `git diff dc51e8c..HEAD -- go.mod` is empty — the file is byte-identical to the phase-gate commit (and unchanged since `0ead1e8`, pre-phase). First runtime `require` block contains exactly grpc + protobuf. Note: the plan's pinned block-hash literal (9679f143…) was not reproduced by this run's extraction variants (81e1a957… for `sed '/^require (/,/^)/p'`); the git identity check above is the evidence that nothing drifted — the mismatch is extraction-method, not content. |

**Score:** 20/20 truths verified (0 present, behavior-unverified)

### Required Artifacts

All artifacts declared across the 11 plans exist, are substantive, and are wired; every one is inside the recomputed fingerprint's covered set (58 files). Representative rows (re-confirmed this pass):

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/testutil/perf.go` | one threshold table, gate, hardware block rule | ✓ VERIFIED | Consts + `PerfEnabled`/`RequirePerf`/dump-dir env; only env reads are the two `SUPKV_PERF` vars; consumed by all perf test packages. |
| `internal/testutil/perf_linux.go`, `perf_other.go`, `race_on.go`, `race_off.go` | hardware block, fs/device probes, race label | ✓ VERIFIED | Present, unchanged since gate; probes feed the ledger's blocks. |
| `internal/stand/perf_test.go` | burst, default latency, amplification, memory rows, drift guard | ✓ VERIFIED | Five perf rows present; burst row SKIP message names `make perf` (observed); recorded green runs. |
| `internal/app/repl/bench_test.go`, `retransmit_test.go` | cost gates, counting double, paired benchmark | ✓ VERIFIED | Exact-vector gates and `BenchmarkNotifyWriteCost` present; always-on gates green in `make test`. |
| `internal/app/repl/core.go` | `pendingByKey`, `PendingVisits` | ✓ VERIFIED | Insert :2441, guarded delete :3461, counter :3362; no production metric consumer. |
| `internal/app/repl/sim_dump_test.go`, `config_internal_test.go` | synctest harness, chunker teeth | ✓ VERIFIED | 16 synctest refs, 0 sleeps; teeth tests green in `make test`. |
| `internal/grpcapi/server.go`, `internal/udprepl/loop.go`, replay tests | version-only replay seam | ✓ VERIFIED | Version-only construction :425 / enqueue :261; live path still clones :239. |
| `internal/testutil/ports.go` + adopters | shared allocator | ✓ VERIFIED | Four bands + doc-port reserve; five adopter files confirmed. |
| `internal/grpcapi/bench_test.go` | two-profile benchmark + fast gate | ✓ VERIFIED | Shared body, per-arm conditions, fast gate SKIP outside `make perf`. |
| `internal/stand/stand.go`, `cmd/sup-kv/main_test.go`, `kill_and_reap_test.go` | diagnostics/budgets/serialization | ✓ VERIFIED | `ErrProcessDone` diagnosis, 30 s budgets, 8-slot semaphore. |
| `docs/PERFORMANCE.md`, `cmd/sup-kv/performance_test.go` | ledger + drift guard | ✓ VERIFIED | 270-line ledger read in full; three guards PASS uncached. |
| `docs/OPERATIONS.md`, `docs/SPEC-DEVIATIONS.md`, `.planning/PROJECT.md` | ripple + registry closure | ✓ VERIFIED | Ledger link at OPERATIONS:200; S-FAIL-15 closed; PROJECT claim conditioned. |

### Key Link Verification

Spot re-confirmed this pass (all previously WIRED in the initial verification; the changed file is doc-only):

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `Makefile` perf target | `internal/testutil/perf.go` | `SUPKV_PERF=1`/`SUPKV_PERF_DUMP_DIR` set only here | ✓ WIRED | `Makefile:60-68` read; thresholds are the compiled consts. |
| `internal/stand/perf_test.go` | `internal/stand/stand_red_test.go` | `redRunBurst` over the frozen contract | ✓ WIRED | Frozen file hash unchanged (`573fed68…`). |
| `internal/stand/perf_test.go` | `/proc/<pid>/io`, `/proc/<pid>/status` | amplification write_bytes and RSS | ✓ WIRED | Present; recorded rows logged real values. |
| `internal/app/repl/core.go` insert/remove | `pendingByKey` | single insert + equality-guarded chokepoint | ✓ WIRED | `core.go:2441`, `core.go:3461`. |
| `internal/grpcapi/server.go notifyReplayedWrite` | `core.trackLocalWrite` | version is the only consumed field | ✓ WIRED | `server.go:425`. |
| `internal/udprepl/loop.go NotifyReplayedWrite` | adapter notify channel | version-only enqueue, live path unchanged | ✓ WIRED | `loop.go:261` vs `:239`. |
| `internal/grpcapi/bench_test.go` default arm | `internal/persistfile` | real dump store | ✓ WIRED | Present; green in `make test`. |
| `cmd/sup-kv/performance_test.go` | `internal/testutil/perf.go` | threshold literals formatted from consts | ✓ WIRED | Guard PASS today after the doc edit. |
| `docs/PERFORMANCE.md` | perf rows in the tree | every cited test/benchmark exists | ✓ WIRED | Cited tests enumerated in truths 2-5; `TestPerformanceDoc` guards anchors. |
| `docs/PERFORMANCE.md` target-media gap | the ledger itself (corrected) | self-recorded, registry claim true | ✓ WIRED | `docs/PERFORMANCE.md:169-170`; registry grep confirms 0 such rows. |
| `docs/OPERATIONS.md` | `docs/PERFORMANCE.md` | one memory-section link | ✓ WIRED | OPERATIONS.md:200. |
| `Makefile bench-smoke` | `.github/workflows/scheduled.yml` | workflow invokes target by name | ✓ WIRED | 3 hits in scheduled.yml; CI diff empty. |

### Data-Flow Trace (Level 4)

The ledger's figures trace to the recorded green `make perf` runs at commit `9cdc0f1` on sources git-identical to this tree (evidence-scope note 1); the deterministic half of each chain re-ran green today.

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `TestPerfBurstDistinctKeys` figures | achieved rate / drain | frozen `redRunBurst` over the real binary; unacked gauge from `/metrics` | Yes — recorded 1000.1 w/s / 305.13 ms; R-02 burst row green in today's red-runner | ✓ FLOWING |
| `TestPerfAmplification` figures | bytes-per-dump / amplification | `supkv_dumps_total` + dump file stat + `/proc/<pid>/io` write_bytes | Yes — recorded 15 dumps, 14 979 072 B/dump, 14.36× | ✓ FLOWING |
| `TestPerfMemoryPoints` figures | RSS/VmHWM/peak | `/metrics` + `/proc/<pid>/status` | Yes — recorded 3 points, 60 KiB row 3.73×/0.54× | ✓ FLOWING |
| `TestPerfAbsentPeerWriteCost` figures | ns/op, visits/op | `testing.Benchmark` over the real core at 10 000 pending | Yes — recorded 2249 vs 8598 ns/op, 0.26 ratio; deterministic half green today | ✓ FLOWING |
| `TestPerfFastProfileLatency` figures | mean/p99 | `BenchmarkConfirmationLatency` fast arm | Yes — recorded 2.82 ms / 5.55 ms, 817 samples | ✓ FLOWING |
| `internal/testutil/perf_linux.go HardwareBlock` | CPU/RAM/load/disk | `/proc` probes + mountinfo + rotational | Yes — five real blocks in the ledger | ✓ FLOWING |
| `docs/PERFORMANCE.md` rows | published figures | transcribed from the two green `make perf` runs at `9cdc0f1`, each with block and command | Yes — corroborated first-hand in the 2026-11-25 verification on identical code; deterministic gates re-green today | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Standing test gate | `make test` | exit 0; 19 package rows ok (repl 54.367 s fresh, race on) | ✓ PASS |
| Lint gate | `make lint` | exit 0 (config verify + golangci-lint) | ✓ PASS |
| Doc drift guards on the changed ledger | `go test -count=1 -run 'TestPerformanceDoc\|TestPerformanceDocFamilies\|TestSpecDeviationsBurstRow' ./cmd/sup-kv/` | ok 2.244 s — the `e16cdd8` doc edit passes its own guard | ✓ PASS |
| Perf row skips outside operator target | `env -u SUPKV_PERF go test -v -run '^TestPerfBurstDistinctKeys$' ./internal/stand/` | SKIP with "invoke `make perf` (SUPKV_PERF=1, no -race)" | ✓ PASS |
| Perf row skips under race | `SUPKV_PERF=1 go test -race -v -run '^TestPerfFastProfileLatency$' ./internal/grpcapi/` | SKIP | ✓ PASS |
| Red-gate expectation rows | `bash scripts/red-runner` | 7 rows checked, 6 green; R-01 failed locally at the throttled-host scale — adjudicated Phase-3 class (evidence-scope note 2); exit 1 | ✓ PASS (with note 2) |
| Proto gates | `make buf-lint`, `make buf-breaking` | exit 0 each | ✓ PASS |
| Public-client gate | `make client-example` | exit 0 | ✓ PASS |
| Darwin cross cell | `GOOS=darwin go build ./...`, `GOOS=darwin go vet ./...` | exit 0 each | ✓ PASS |
| Whole `make perf` on this host | not re-run — powersave-throttled host; sources byte-identical to the recorded green runs at `9cdc0f1` | recorded runs stand; deterministic gates re-run green above | ? SKIP (evidence-scope note 1) |

### Probe Execution

No probe scripts are declared by the phase plans and no conventional `scripts/*/tests/probe-*.sh` exists in this tree (find: 0); Step 7c is not applicable. The phase's own runnable gate (`scripts/red-runner`) was executed and is recorded above.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| PERF-01 | 08-01, 08-03, 08-07, 08-08, 08-10, 08-11 | Write-latency model documented for both profiles | ✓ SATISFIED | Ledger read in full (`docs/PERFORMANCE.md:23-93`); guards PASS on the current doc; recorded green rows for both profiles. |
| PERF-02 | 08-01, 08-06, 08-09, 08-10, 08-11 | 1000 writes/s distinct-key burst is a tested bound | ✓ SATISFIED | `TestRed_R02_burst` (frozen, hash reproduces) + `TestPerfBurstDistinctKeys` + `TestEngine_burstBoundedConfirmation`; R-02 burst row green in today's red-runner; guarantee/non-guarantee text read. |
| PERF-03 | 08-07, 08-10, 08-11 | Amplification measured on target-grade media or gap recorded | ✓ SATISFIED | Measured on workstation NVMe/ext4 (recorded 14.36×, per-dump bound inside 1.1×); target-media gap recorded in the ledger itself and the corrected registry statement verified true. |
| PERF-04 | 08-07, 08-10, 08-11 | Memory model documented from stand measurements | ✓ SATISFIED | Formula + three measured points at `docs/PERFORMANCE.md:184-217`; both 60 KiB tripwires inside bounds; OPERATIONS links without duplicating. |
| PERF-05 | 08-02, 08-04, 08-05, 08-10, 08-11 | Absent-peer latency near live-peer, proven by a benchmark | ✓ SATISFIED | Paired benchmark + perf-gated ratio (recorded 0.26 ≤ 2.0) plus always-on zero-visit gates green in today's `make test`. |
| PERF-06 | 08-01, 08-03, 08-06, 08-08, 08-09, 08-10, 08-11 | `make perf` with thresholds; published numbers carry hardware description | ✓ SATISFIED | `make perf` target shape verified; five per-section hardware blocks + commands; recorded green runs; CI threshold-free. |

No ORPHANED requirements: REQUIREMENTS.md maps exactly PERF-01..PERF-06 to Phase 8, and every plan's `requirements:` field accounts for all six across the phase (08-01 [PERF-02, PERF-06], 08-02 [PERF-05], 08-03 [PERF-01, PERF-06], 08-04 [PERF-05], 08-05 [PERF-05], 08-06 [PERF-02, PERF-06], 08-07 [PERF-01, PERF-03, PERF-04], 08-08 [PERF-01, PERF-06], 08-09 [PERF-02, PERF-06], 08-10/08-11 [PERF-01..06]).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers `TBD`/`FIXME`/`XXX` across all 36 phase-changed implementation files | None | grep this pass: 0 hits. |
| — | — | Stub patterns / placeholder text in production artifacts | None | 0 hits in the ledger and guard; production code unchanged since the verified gate. |
| `docs/PERFORMANCE.md` | 169 | (Previous report's warning) ledger pointed at a `docs/SPEC-DEVIATIONS.md` row that does not exist | ⚠️ RESOLVED | Commit `e16cdd8` re-pointed the gap at the ledger itself; this pass verified the corrected statement is factually true (registry carries no such row, grep 0 hits) and the doc guard still PASSes. |

No 🛑 BLOCKER anti-patterns.

### Human Verification Required

N/A — infrastructure/documentation phase with no user-facing elements. Every behavior-dependent truth is covered by a deterministic gate inside `make test` (re-run green today) or by a recorded green `make perf` run on sources git-identical to this tree. No ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths; no harvested `<human-check>` items in the plans.

### Deferred Items

None. No identified gap is deferred; Phase 9's documentation scope does not absorb any Phase 8 performance deliverable.

### Gaps Summary

No gaps. All five roadmap success criteria and all six PERF requirements remain observably true on the current tree. The stale trigger — commit `e16cdd8`'s edit to the covered `docs/PERFORMANCE.md` — was itself the fix for the previous report's single warning, and the corrected ledger passes its own drift guards. The two evidence-scope notes above (throttled-host `make perf` not re-run; adjudicated R-01 local red-row miss) are recorded host-environment conditions, not phase gaps.

---

_Verified: 2026-11-27T17:52:01Z_
_Verifier: Claude (gsd-verifier)_
