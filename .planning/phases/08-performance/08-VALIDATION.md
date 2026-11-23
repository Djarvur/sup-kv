---
phase: "8"
slug: "performance"
status: draft
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-23"
---

# Phase 8 — Validation Strategy

> Seed contract for the performance phase. Every task in `08-01-PLAN.md` …
> `08-11-PLAN.md` carries an automated verification with a `<fails_when>`; this
> file maps them one row per task so the executor and the Nyquist audit read the
> same table. Statuses stay `pending` until the plan's command has run on the
> committed tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` (incl. `testing/synctest`) + `go tool golangci-lint` v2.15.0 |
| **Config file** | `go.mod` (go directive), `.golangci.yml`, `Makefile`; no extra test config |
| **Quick run command** | `go test -race -count=1 ./internal/app/repl/ ./internal/testutil/ ./internal/stand/ ./cmd/sup-kv/` |
| **Full suite command** | `make lint && make test && make buf-lint && make buf-breaking && make client-example` |
| **Perf command** | `make perf` (`uname` Linux guard; `SUPKV_PERF=1`, `SUPKV_PERF_DUMP_DIR=$(CURDIR)/.tmp/perf`, `-p 1`, no `-race`; then the functional stand suite once with no thresholds) |
| **Frozen contract** | `internal/grpcapi/red_gate_test.go`, `internal/stand/stand_red_test.go`, `scripts/red-expectations.tsv` (byte-pinned; `bash scripts/red-runner` = 7/7 with both mutants) |
| **Estimated runtime** | per-task commands ≤ 120 s; `make test` ~6 min under `-race`; `make perf` ~4-6 min on a calm host |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (the map below).
- **After every plan wave:** `go build ./... && make test` (race, full suite).
- **Before the phase gate:** the 08-11 gate list — frozen hashes, sed targets, full red-runner, `make lint/test/buf-lint/buf-breaking/client-example`, darwin build/vet, pinned `.golangci.yml`/`go.mod` hashes, the explicit deterministic-gate list, the 100-run dumpNotify scenario, the doc anchors.
- **Before publishing a number:** `make perf` twice on the target-class host; `docs/PERFORMANCE.md` carries the second run's block.
- **Max feedback latency:** ~360 s (full suite); every per-task command ≤ 120 s; the perf rows are the only multi-minute commands and are operator-triggered.

---

## Wave Graph

Waves serialize at the Go-package level as well as the file level: no two plans in a wave
modify the same file, and no two plans in a wave modify the same package (a parallel sibling
writing another file in the same package can break a package-scoped verify). Every declared
`depends_on` is therefore a real ordering, not a formality.

| Wave | Plans | Packages touched in that wave |
|------|-------|-------------------------------|
| 1 | 08-01, 08-02, 08-04 | `internal/testutil`+`internal/stand`+`Makefile`; `internal/app/repl`; `internal/grpcapi`+`internal/udprepl` |
| 2 | 08-05, 08-06 | `internal/app/repl`; `internal/testutil`+`internal/stand`+`cmd/sup-kv`+`internal/udprepl`+`internal/grpcapi`+`Makefile` |
| 3 | 08-03, 08-07, 08-08 | `internal/app/repl`; `internal/testutil`+`internal/stand`; `internal/grpcapi`+`Makefile` |
| 4 | 08-09 | `internal/stand`+`cmd/sup-kv` |
| 5 | 08-10 | `cmd/sup-kv`+`docs` |
| 6 | 08-11 | none (verification only) |

Reasoning per edge: the core index (08-02) precedes the gates that count on it (08-05); the port
allocator (08-06) precedes the stand perf rows that call its wrapper (08-07); the replay cleanup
(08-04) is independent; the docs (08-10) follow every measurement they cite and the last
`cmd/sup-kv` harness change; the gate (08-11) is last.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 08-01 | 1 | PERF-06 | T-08-01 | One compiled threshold table; a perf test cannot run without the operator gate | unit | `go test -race -count=1 ./internal/testutil/ -run 'TestPerfEnabled\|TestRequirePerfSkips\|TestPerfThresholdRelations\|TestRaceEnabledStable' -v` | ✅ (task) | ⬜ pending |
| 08-01-02 | 08-01 | 1 | PERF-06 | T-08-03 | Hardware block and dump-filesystem guard degrade to `unknown`, never fail; per-test dump dirs | unit | `go test -race -count=1 ./internal/testutil/ -run 'TestHardwareBlock\|TestPerfDumpDirResolvesUnderBase\|TestPerfDumpFSType\|TestPerfDumpDevice' -v` + `GOOS=darwin go vet ./internal/testutil/` | ✅ (task) | ⬜ pending |
| 08-01-03 | 08-01 | 1 | PERF-02 | T-08-02 | Burst bound runs only under `make perf`; frozen red file untouched; constants cannot drift | perf + drift guard | `make perf > log 2>&1; test $? -eq 0` (exit code captured before any grep) and `go test -race -count=1 -run '^TestPerfBurst' -v ./internal/stand/` | ✅ (task) | ⬜ pending |
| 08-02-01 | 08-02 | 1 | PERF-05 | T-08-04 | Key-local pending index; stale delete cannot evict a newer pair; counter never publishes. Tests 1-2 are the phase's RED evidence (2N/N pre-index); the idle-Advance test is a tripwire, green before the fix | unit | `go test -race -count=1 ./internal/app/repl/ -run 'TestCore_pendingVisits\|TestCore_pendingIndex' -v` | ✅ (task) | ⬜ pending |
| 08-02-02 | 08-02 | 1 | PERF-05 | T-08-06 | XPORT-12 emissions, ledger, suspension and stale-record removal proven unchanged | unit | `go test -race -count=1 ./internal/app/repl/ -run 'TestCore_pendingIndexProductionPaths\|TestCore_pendingIndexDifferential\|TestCore_pendingIndexInvariant' -v` | ✅ (task) | ⬜ pending |
| 08-02-03 | 08-02 | 1 | PERF-05 | T-08-05 | Change surface confined to the index/counter/doc; absent-peer growth no longer multiplies write cost | unit + bench | `go test -race -count=1 ./internal/app/repl/` + `BenchmarkCoreEventLoop` 10x | ✅ | ⬜ pending |
| 08-03-01 | 08-03 | 3 | PERF-01 | T-08-08 | No wall-clock settle in the dumpNotify harness; runtime idle barrier instead | unit ×100 | `go test -race -count=100 -run '^TestScenario_dumpNotify$' ./internal/app/repl/` | ✅ | ⬜ pending |
| 08-03-02 | 08-03 | 3 | PERF-06 | T-08-07 | Chunk capacity bounded by entries, not the byte budget; adjacent chunks never alias | unit | `go test -race -count=1 ./internal/app/repl/ -run '^TestChunkSyncEntries_capacityFollowsEntries$' -v` | ✅ | ⬜ pending |
| 08-04-01 | 08-04 | 1 | PERF-05 | T-08-10 | Version-only replay record pinned by field and by bytes/op (RED first) | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestReplaySeam' -v`; `./internal/udprepl/ -run '^TestNotifyReplayedWrite' -v` | ✅ (task) | ⬜ pending |
| 08-04-02 | 08-04 | 1 | PERF-05 | T-08-09 | Two discard-path clones deleted; live path keeps its defensive copy | unit | `go test -race -count=1 ./internal/grpcapi/ ./internal/udprepl/` + seam-doc greps | ✅ (task) | ⬜ pending |
| 08-05-01 | 08-05 | 2 | PERF-05 | T-08-11 | Per-datagram store calls are exact and size-independent; whole-store reads impossible to add silently | unit | `go test -race -count=1 ./internal/app/repl/ -run '^TestCore_keyCountCostFlat$' -v` | ✅ | ⬜ pending |
| 08-05-02 | 08-05 | 2 | PERF-05 | T-08-12 | Idle/write/apply paths inspect zero pending entries; counter proven wired. All three gates are tripwires (they land after 08-02, so they pass on arrival) | unit | `go test -race -count=1 ./internal/app/repl/ -run 'TestCore_unackedCostFlat\|TestCore_notifyWriteCostFlat\|TestCore_applyRemoteCostFlat\|TestCore_pendingVisitsRepeatedKey' -v` | ✅ | ⬜ pending |
| 08-05-03 | 08-05 | 2 | PERF-05 | T-08-13 | Paired absent/live ratio measured in one process, `make perf` only, fails by name | perf | `make perf` + `go test -race -run '^TestPerfAbsentPeerWriteCost$' -v ./internal/app/repl/` | ✅ (task) | ⬜ pending |
| 08-06-01 | 08-06 | 2 | PERF-06 | T-08-16 | Disjoint bands, network-keyed registry, claim-while-held, sysctl assertion | unit | `go test -race -count=1 ./internal/testutil/ -run 'TestBands\|TestUDPPort\|TestTCPPort\|TestRegistry\|TestEphemeral' -v` | ✅ (task) | ⬜ pending |
| 08-06-02 | 08-06 | 2 | PERF-06 | T-08-14 | Stand and cmd adopt the allocator; both private registries deleted | unit | `go test -race -count=1 ./cmd/sup-kv/ -run 'TestLog_formatText\|TestMain_lifecycle\|TestBroadcastProbe' -v` + registry grep | ✅ | ⬜ pending |
| 08-06-03 | 08-06 | 2 | PERF-06 | T-08-15 | All four socket packages share the allocator; no advertised `:0` bind in the five former picker files (throwaway/sink sockets named as accepted residue) | socket tests ×20 | `go test -race -count=20 -run 'TestProbeQuickstartBroadcast\|TestSmoke_twoNodesUDP\|TestWaitIntegration_' ./cmd/sup-kv/ ./internal/udprepl/ ./internal/grpcapi/` + `-count=5` stand scenarios + per-file no-`:0` greps | ✅ | ⬜ pending |
| 08-07-01 | 08-07 | 3 | PERF-01 | T-08-19 | Default-profile latency sampled scheduled→completion; one percentile implementation | perf + unit | `mkdir -p .tmp/perf && SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$PWD/.tmp/perf go test -count=1 -run '^TestPerfDefaultProfileLatency$' -v ./internal/stand/` | ✅ (task) | ⬜ pending |
| 08-07-02 | 08-07 | 3 | PERF-03 | T-08-17 | Amplification refuses tmpfs and an unhealthy dump window; raw inputs logged | perf | `mkdir -p .tmp/perf && SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$PWD/.tmp/perf go test -count=1 -run '^TestPerfAmplification$' -v ./internal/stand/` (+ the `/dev/shm` refusal) | ✅ (task) | ⬜ pending |
| 08-07-03 | 08-07 | 3 | PERF-04 | T-08-17 | Three measured points with logical size, dump bytes, RSS, VmHWM, pre-dump RSS and peak; two 60 KiB gates (RSS tripwire + dump-transient ratio) | perf | `mkdir -p .tmp/perf && SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$PWD/.tmp/perf go test -count=1 -run '^TestPerfMemoryPoints$' -v ./internal/stand/` | ✅ (task) | ⬜ pending |
| 08-08-01 | 08-08 | 3 | PERF-01 | T-08-20 | Both profiles benchmarked; fast arm's in-memory conditions preserved | bench | `go test -count=1 -run '^$' -bench '^BenchmarkConfirmationLatency$' -benchtime 1x ./internal/grpcapi/` | ✅ | ⬜ pending |
| 08-08-02 | 08-08 | 3 | PERF-01 | T-08-20 | Fast-arm bound measured only under `make perf`, missing metric fails by key | perf | `go test -race -run '^TestPerfFastProfileLatency$' -v ./internal/grpcapi/` (SKIP) + `make perf` | ✅ (task) | ⬜ pending |
| 08-08-03 | 08-08 | 3 | PERF-06 | T-08-21 | Core benchmarks wired into both bench targets; CI smoke stays threshold-free | build target | `make bench-smoke` + workflow diff | ✅ | ⬜ pending |
| 08-09-01 | 08-09 | 4 | PERF-06 | T-08-22 | Dead child diagnosed with pid, exit status and output tails; SIGKILL path unchanged | unit | `go test -race -count=1 ./internal/stand/ -run '^TestKillAndReap\|^TestExitDiagnosis\|^TestAwaitNodeServing' -v` | ✅ (task) | ⬜ pending |
| 08-09-02 | 08-09 | 4 | PERF-06 | T-08-22 | Spawn semaphore (release after reap) plus classified readiness budgets; both duplicate-node log waits converted; crashed child never looks slow | unit ×5 | `go test -race -count=5 ./cmd/sup-kv/ -run 'TestMain_lifecycle\|TestLog_formatText\|TestLog_jsonDefault'` | ✅ | ⬜ pending |
| 08-09-03 | 08-09 | 4 | PERF-02 | T-08-23 | Heavyweight doc walkthroughs serialized with their own block budget | doc test ×3 | `go test -race -count=3 ./cmd/sup-kv/ -run 'TestExamplesDoc\|TestReadmeQuickstart\|TestMain_lifecycle\|TestLog_formatText'` | ✅ | ⬜ pending |
| 08-10-01 | 08-10 | 5 | PERF-01, PERF-02, PERF-03, PERF-04, PERF-05, PERF-06 | T-08-26 | Thresholds anchored to Go constants; per-section block+command; S-FAIL-15 row anchored; standalone claims forbidden (RED-first) | doc anchor | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestPerformanceDoc$' -v` (RED: doc absent) | ✅ (task) | ⬜ pending |
| 08-10-02 | 08-10 | 5 | PERF-01, PERF-02, PERF-03, PERF-04, PERF-05, PERF-06 | T-08-25 | Every measured section carries exactly one block plus its command and conditions; disk cost documented; target-media gap recorded | doc anchor | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestPerformanceDoc$' -v` + disk-cost/condition/gap greps | ✅ (task) | ⬜ pending |
| 08-10-03 | 08-10 | 5 | PERF-01, PERF-02 | T-08-25 | S-FAIL-15 cites the three shipped tests and the two bounds (hot-keys-only wording gone); no duplicated memory numbers; PROJECT.md's old standalone claim gone | doc anchor | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestSpecDeviationsBurstRow$' -v` + `TestOperationsDoc\|TestPerformanceDoc\|TestAPIDoc\|TestExamplesDoc\|TestReadme` + greps | ✅ | ⬜ pending |
| 08-11-01 | 08-11 | 6 | PERF-01…06 | T-08-28 | Frozen red evidence byte-identical; sed targets present; 7/7 with mutants | gate | frozen `git hash-object` pins + `bash scripts/red-runner` | ✅ | ⬜ pending |
| 08-11-02 | 08-11 | 6 | PERF-01…06 | T-08-29, T-08-30 | Full standing gate, darwin, pinned hashes, workflow diff, recorded `make perf`, and every doc-cited `TestPerf*` name listed in the tree | gate | `make lint && make test && make buf-lint && make buf-breaking && make client-example`; darwin; hash pins; `make perf` with its exit code captured before grepping; `go test -list` name check | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*File Exists: `✅` = the referenced test/soak infrastructure is on the committed tree; `✅ (task)` = the task creates the named file itself, so no separate Wave 0 scaffold is required.*

---

## Wave 0 Requirements

No separate Wave 0 plan is needed: every helper a later task depends on is itself the first task of a plan, so the dependency is visible in the wave graph rather than in an unplanned scaffold.

| Helper | Owning task | Consumed by |
|--------|-------------|-------------|
| `internal/testutil/perf.go` (consts, `PerfEnabled`, `RequirePerf`) | 08-01-01 | 08-05-03, 08-07-*, 08-08-02 |
| `internal/testutil/perf_linux.go` / `perf_other.go` (`HardwareBlock`, `PerfDumpDir`, `PerfDumpFSType`, `PerfDumpDevice`) | 08-01-02 | 08-01-03, 08-07-*, 08-08-02 |
| `internal/testutil/ports.go` (bands, UDP/TCP pickers) | 08-06-01 | 08-06-02, 08-06-03 |
| `internal/testutil/bench.go` (`LatencyPercentiles`) | 08-07-01 | 08-07-01, 08-07-03 |
| `make perf` target | 08-01-03 | 08-05-03, 08-07-*, 08-08-02, 08-11-02 |
| `internal/app/repl/core.go` `PendingVisits` | 08-02-01 | 08-05-02, 08-05-03 |

`wave_0_complete: true` records that no plan references a file no task creates; it does not claim the helpers exist before execution.

---

## Manual-Only Verifications

| Behavior | Requirement | Why manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The published latency/amplification/memory figures are true on the reader's hardware | PERF-01, PERF-03, PERF-04 | A number measured on one host cannot be asserted on another; the doc-anchor test proves presence and drift, never truth | Run `make perf` on the reader's host, compare the block and the rows with `docs/PERFORMANCE.md`, and rerun a diverging row rather than editing the threshold |
| The target-grade media gap closes | PERF-03 | No eMMC/SD is attached to the development host (D-03 permits the recorded gap) | Attach the target device, set `SUPKV_PERF_DUMP_DIR` on it, run `TestPerfAmplification`, replace the gap row with the measured one |
| A calm host was used | PERF-06 | Load is environmental, not a code property (A4) | Check the block's `load=` line against `runtime.NumCPU`; rerun on a quieter window if it is saturated |

---

## Validation Sign-Off

- [x] All 30 tasks carry an `<automated>` verify with a `<fails_when>` (or a Wave 0 dependency named above)
- [x] Sampling continuity: no three consecutive tasks without automated verify
- [x] Wave 0 covers every reference a plan depends on (the table above; each row is a task, not an assumption)
- [x] No watch-mode flags; the only env gates are `SUPKV_PERF` and the pre-existing `SUPKV_RED_TESTS`
- [x] Feedback latency: per-task ≤ 120 s, full suite ~360 s
- [x] Frozen-contract checks are in the gate plan (hashes, sed shapes, 7/7 with mutants)
- [x] `nyquist_compliant: true` set in frontmatter
- [ ] Phase gate executed (08-11) and every row above flipped to ✅ green

**Approval:** pending — seed written 2026-11-23 with the plan set. Flip `status` to `validated` only after 08-11 runs its full list on the committed tree and every row's command has been re-run.
