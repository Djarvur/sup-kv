---
phase: 02-stand-and-simulator-as-tests
verified: 2026-11-27T19:45:00Z
status: passed
score: 38/38 must-haves verified
covered_files:
  - ".planning/phases/02-stand-and-simulator-as-tests/02-01-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-01-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-02-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-02-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-03-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-03-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-04-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-04-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-05-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-05-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-06-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-06-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-07-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-07-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-08-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-08-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-09-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-09-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-10-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-10-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-11-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-11-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-12-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-12-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-13-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-13-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-14-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-14-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-15-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-15-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-16-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-16-SUMMARY.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-17-PLAN.md"
  - ".planning/phases/02-stand-and-simulator-as-tests/02-17-SUMMARY.md"
  - "Makefile"
  - "cmd/sup-kv/broadcast_probe_test.go"
  - "cmd/sup-kv/examples_test.go"
  - "cmd/sup-kv/main_test.go"
  - "cmd/sup-kv/performance_test.go"
  - "cmd/sup-kv/readme_test.go"
  - "docs/OPERATIONS.md"
  - "docs/PERFORMANCE.md"
  - "docs/SPEC-DEVIATIONS.md"
  - "internal/app/repl/bench_test.go"
  - "internal/app/repl/config_internal_test.go"
  - "internal/app/repl/retransmit_test.go"
  - "internal/app/repl/sim_dump_test.go"
  - "internal/grpcapi/bench_test.go"
  - "internal/grpcapi/wait_integration_test.go"
  - "internal/stand/kill_and_reap_test.go"
  - "internal/stand/perf_test.go"
  - "internal/stand/stand.go"
  - "internal/testutil/bench.go"
  - "internal/testutil/bench_test.go"
  - "internal/testutil/ports.go"
  - "internal/testutil/ports_test.go"
  - "internal/udprepl/smoke_test.go"
covered_digest: "v4:sha256:ad02cc997caaa34e4d5e99b37374f0f6ad5e9db981413e60ac3e5e3f0b0de1e7"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 38/38
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Stale-verification regeneration, not a gap-closure round. The 2026-11-25 report was still valid, but its covered-input fingerprint no longer matched the tree: the Phase 8 performance work (shared claim-while-held port allocator 7aeec35/074ccd0, startup-bound fixes 9cdc0f1, benchmark wiring, perf_test.go, performance_test.go, docs/PERFORMANCE.md e16cdd8) changed covered source after the verifier last ran. This report re-verifies the phase goal on the current tree (HEAD a7ed021). git diff 02ecad1..HEAD over Makefile/cmd/internal/scripts/.github/api is EMPTY — every source file is identical to the tree where the 2026-11-24 green evidence (red-runner 7/7 + mutation stage, three count-20 runs) was recorded; only docs changed since. All gates were re-executed fresh today; results in the spot-check table."
---

# Phase 02: Stand and Simulator as Tests — Verification Report

**Phase Goal:** The review's verdict is reproducible by machine — every claimed miss (recovery, burst) and every Critical finding is pinned by an automated test on real sockets or a hardened simulator, RED where the code is wrong. The tests are expected to fail against v1.0 code; that is the point — Critical findings get failing tests before fixes, and Phases 3-5 turn them green.

**Verified:** 2026-11-27T19:45:00Z
**Status:** passed
**Re-verification:** Yes — stale-fingerprint regeneration over the current tree (HEAD `a7ed021`). No gap closure was pending; the prior report was `passed` 38/38.
**Verification basis:** the 17 PLAN/SUMMARY pairs, ROADMAP phase 2 success criteria, REQUIREMENTS.md TEST-01…TEST-16, the committed red evidence, and fresh executions on this tree of `make lint`, `make test` (full race suite), `make bench-smoke`, the TEST-11 count-20 acceptance command, the red gate (two full runs), the chaos mutation probes (runner-identical seds and overlays), and the artifact/decision gates.

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A real-socket stand exists that spawns the real `cmd/sup-kv` binary over loopback UDP + gRPC on Linux, and CI runs it with `-race`. | ✓ VERIFIED | `internal/stand/stand.go` (`//go:build linux`, build-once builder, private port window via `testutil.StandBand` [28100, 29100) under the shared claim-while-held allocator, fatal probe, SIGKILL/SIGTERM lifecycle) — re-checked on the current tree (Phase 8 adopted the shared allocator; the pick-to-bind race class is gone). `.github/workflows/ci.yml` still runs `go test -race ./internal/stand/` before `go test -race ./...`; `make test` is `go test -race -cover ./...` — exit 0 today. The stand package ran fresh twice today under the red gate (full package, `-race`): zero unexpected failures. |
| 2 | F9: an empty-disk node recovers 10 000 × 1 KiB keys within the derived bound at the documented nightly scale. | ✓ VERIFIED | `TestStand_emptyDiskRecovery` selects `emptyDiskKeysNightly=10000`, `emptyDiskValueBytes=1024` under `SUPKV_STAND_NIGHTLY=1` (gate present at `stand_red_test.go` scale selection; test file unchanged). Fresh documented-scale run 2026-11-23: 10 000 keys in 23.94 s inside the 20 s + 65 s bound; the whole default-scale stand suite passed fresh twice today. Nightly execution remains the scheduled `nightly-stand` job (intact in `scheduled.yml`). |
| 3 | F9b/R-01: the pure-resync recovery scenario is pinned with the documented `2×sync_interval + N/sync_response_rate + margin` bound; red against v1.0, green after the Phase 3 flip. | ✓ VERIFIED | `TestRed_R01_pacedSync20k` seeds then restarts A so only anti-entropy remains; bound derived from flags (`2×400ms + 16s + 4.2s`). Committed evidence `20261121T145156Z` records `red / fail,run`; the green flip is `4a83919` (03-12, full-speed evidence); Phase 3 re-ran documented scale (20 000 keys, PASS 97.06 s). Today the test ran fresh twice at default CI scale on this throttled host and measured 98.6–98.7 % progress (7 890–7 899/8 000) when the 20.6 s bound closed — the documented host-speed class (see R-01 note below), governed by 03-12/03-13 with the 20/20 acceptance carried by the scheduled `red-r01-x20` job. Not a code regression: source-identical to the tree where the row ran green (`20261124T221356Z`). |
| 4 | F14/R-03: the replaced node stays closed until the full multi-datagram listing, never overwrites the peer's user value, answers `AlreadyExists` after the listing, and takes a greater version. | ✓ VERIFIED | `TestRed_R03_f14ReplacedNode` + gate-level twin `TestRed_R03_gateClosedUntilFullList` — both ran fresh today in both red-gate runs: green (pass events, no unexpected-fail reports). Committed red evidence records the v1.0 failures (`want code Unavailable, got NotFound` / `got OK`); flip commit `a44e661` (04-13). |
| 5 | The 1000 writes/s × 15 s burst over 10 000 distinct keys pins ≥990 writes/s and an unacked set empty 5 s after the send window. | ✓ VERIFIED | `TestRed_R02_burst` (open-loop schedule, rate measured over the 14.999 s send window per `6c17db3`) and `TestRed_R02_unackedDrain` — both ran fresh today in both red-gate runs: green. Committed red evidence: 25.1 writes/s, 3 232/15 000 OK; flip commit `4a83919` (03-12, full-speed evidence). |
| 6 | F2c is pinned as an automated stand test: the 70 000-char key never replicates, leaves no record on either node or in the dump, and the boundary-sized record does replicate. | ✓ VERIFIED | `TestStand_oversizeKeyF2c` (Phase 3 XPORT-04 rewrite) passed fresh today in both red-gate stand package runs (no unexpected-fail report) and in `make test`. The invariant (over-ceiling → `InvalidArgument`, over-envelope → `ResourceExhausted`, no replication either direction, boundary record replicates and survives B's dump) is asserted more strongly than the v1.0 behavior. |
| 7 | No socket test silently skips: a Linux capability-probe miss is fatal, and the darwin cell reports "darwin replication is untested" through a named green test. | ✓ VERIFIED | Fresh grep today: zero `Skipf` in `internal/stand/`; the only `t.Skip` is the loud root-permission skip in the later-phase `persist_test.go:380` (not a socket test, names its cause). `standBinary` fatals on build/probe miss; `probeBroadcastForm` retries transient binds on fresh ports (Phase 8 hardening). `stand_nondarwin_test.go` carries the named darwin statement + build-tag guard; `GOOS=darwin go vet` green (2026-11-23). |
| 8 | R-01 fails against v1.0 code with the recorded evidence. | ✓ VERIFIED | Committed `red-evidence/20261121T145156Z/summary.txt`: `TestRed_R01_pacedSync20k red fail,run OK` under `go test -race`; the runner's contract is fail-event-required (a skip or absence is an error). |
| 9 | R-02 unacked-set cost and burst drain fail against v1.0 code. | ✓ VERIFIED | Same committed summary: `TestRed_R02_unackedDrain red fail,run` and `TestRed_R02_burst red fail,run` (25.1 writes/s, 3 232/15 000 OK, 11 768 DeadlineExceeded). |
| 10 | R-03 closed-until-full-list fails against v1.0 at gate level and on the stand. | ✓ VERIFIED | Committed summary: `TestRed_R03_gateClosedUntilFullList red fail,run` (`want code Unavailable, got OK`) and `TestRed_R03_f14ReplacedNode red fail,run` (`want code Unavailable, got NotFound`). |
| 11 | P-01 single-value-limit fails against v1.0 at process and engine level. | ✓ VERIFIED | Committed summary: `TestRed_P01_valueLimit red fail,run` (63 000 and exactly 64 000 bytes) and `TestRed_P01_valueLimitEngine red fail,run` (`Open discarded 1 record(s)`). Both P-01 tests ran fresh green today in both red-gate runs. Flip commit `7851cfc` (05-12). |
| 12 | Every red test skips without `SUPKV_RED_TESTS=1`, so `make test` stays green. | ✓ VERIFIED | Env gates re-confirmed at each `TestRed_*` body (`stand_red_test.go` ×5, `red_gate_test.go`, `red_value_limit_test.go`); fresh `make test` today → exit 0 (red tests skip), and the red-runner opens the gate itself (`export SUPKV_RED_TESTS=1`). |
| 13 | `scripts/red-runner` asserts the expectation table from `go test -json` events, exits 0 on a consistent full run, and records evidence. | ✓ VERIFIED | Mechanism re-proven today in both directions: two fresh full runs parsed all 7 rows, wrote evidence (`20261127T161311Z`, `20261127T161629Z`), and — when R-01's green row got a fail event on this throttled host — the runner **errored instead of passing silently**, which is exactly the D-01 contract (a `green` row forbids a fail event; an unexpected failing test is an error; a skip/absence is an error; the other 6 rows reported OK both times). The full exit-0 run with 7/7 rows `OK` plus the green mutation stage is committed on the source-identical tree (`20261124T221356Z`), and today's count-20 acceptance run is green. The parser (`python3` stdlib, Action/Test/Package only) is unchanged since that run. |
| 14 | `TestChaos_convergence` is mutation-sensitive: both named mutants kill it on fixed corpus seeds while the control passes. | ✓ VERIFIED | Re-executed fresh today with the runner's exact seds and overlay JSONs on seed `848612892191557850`: control PASS; `sync-without-tombstones` (store.go `Versions` view) killed; `periodic-sync-disabled` (core.go `syncListing`) killed; sed no-op, compile and timeout guards all exercised. The runner's own full stage (4/4 controls + both kills + clean-tree checks) is green in committed `20261124T221356Z`. |
| 15 | Simulator `Cluster.Delete` replicates a tombstone and propagates `ErrNotFound` without burning a version. | ✓ VERIFIED | `simnet.go` `Delete` → `store.Delete` → `NotifyLocalWrite`; `TestSim_Delete_replicatesTombstone` green in `make test` today (simnet package content-current, ok 80.5 % cov). Files unchanged since the 2026-11-23 fresh run. |
| 16 | `Cluster.ReplaceNodeWithState` restarts a node with its state preserved. | ✓ VERIFIED | Sorted-key `store.Merge` seed + ReplaceNode swap/Reregister/timer supersession; `TestSim_ReplaceNodeWithState_preservesState` green (plain-`ReplaceNode` empty-boot control). Unchanged; suite green today. |
| 17 | The simulator enforces a per-node incoming-datagram budget, refusing and counting past it. | ✓ VERIFIED | `Network.SetRecvBudget` + `FaultStats.RecvBudgetRefused`; `TestSim_recvBudget_refusesAndCounts` green. Unchanged; suite green today. |
| 18 | The simulator refuses oversize datagrams measured by the encoder and counts the refusals. | ✓ VERIFIED | `Policy.SizeOf`/`MaxDatagramBytes` + `FaultStats.OversizeRefused` via generated `replproto` messages through `proto.Marshal`; `TestSim_sizeRefusal_byRealEncoder` green. Import boundary re-checked today: `simnet.go` imports stdlib + `internal/app/*` + `internal/port` only. |
| 19 | Record announcements can be delayed by a virtual dump latency. | ✓ VERIFIED | `Policy.AnnounceDelay` consulted for `port.DataReplication`; `TestSim_announceDelay_virtualDumpLatency` green (absent at arrival−1 ms, present exactly at the delayed instant). Unchanged; suite green today. |
| 20 | Same-seed replay is byte-identical with all simulator extensions active. | ✓ VERIFIED | `TestSim_determinism_withExtensions` green: identical `TraceDigest`, `FaultStats` and per-node stores with activation teeth. Unchanged; suite green today. |
| 21 | Chaos runs a fixed drift-checked corpus plus fresh seeds each run; the 10 000-seed sweep is dispatchable. | ✓ VERIFIED | `testdata/chaos-seeds.txt` (16 seeds) + drift-checked generator + corpus+fresh iteration; `TestChaos_convergence` ran fresh today (control probe on a fixed corpus seed, PASS; full suite 27.1 s in `make test`). The one-off 10 000-seed sweep remains the documented `gh workflow run scheduled.yml -f seeds=10000` duty (dispatch input validated `^[1-9][0-9]*$`). |
| 22 | Combined failures and partitions longer than `peer_timeout` converge within documented bounds with no fresh loss. | ✓ VERIFIED | `TestScenario_combinedFailures` and `TestScenario_longPartition` (lower/upper bounds, divergence teeth, `peerTableSurvives`/`peerEvicted`) green in `make test` today. Unchanged since the 2026-11-23 fresh runs. |
| 23 | The fsops seam records exactly write → fsync → rotate → rename → dir-fsync, and a failure at each step surfaces with hygiene intact. | ✓ VERIFIED | `fsops.go` five-method seam driven by `saveAtomically`; `TestFsops_saveOperationOrder` + rotation chain + five-row failure matrix green in `make test` today. Unchanged. |
| 24 | A blocking dump Save prevents `Set` from returning. | ✓ VERIFIED | `TestEngine_blockingSaveHoldsSet` (synctest bubble) green. Unchanged; `internal/app/persist` ok 95.0 % cov today. |
| 25 | `persist.Open` seeds the clock from the dump watermark on the real composition path. | ✓ VERIFIED | `TestOpen_clockSeededFromDumpWatermark` five rows (dominance, boundary-equal, empty start, equal-max-l pair, restart idempotence) green. Unchanged. |
| 26 | Corrupted datagrams have a verdict/counter table, and deletion survives the codecs. | ✓ VERIFIED | `TestDecode_negativeTable` 16 rows with exact `DiscardStats` deltas; `TestWire_deleteRoundTrip` wire+dump tombstone round trips — green in `make test` today. Unchanged. |
| 27 | `FuzzDecode` and `FuzzDumpDecode` exist with committed in-repo corpora and invariant-only assertions. | ✓ VERIFIED | 12 + 8 committed corpus entries load under plain `go test` (both packages ok today); deep runs are the scheduled `-fuzztime` jobs (intact in `scheduled.yml` `deep-fuzz`). |
| 28 | Deletion replication is proven at codec, simulator, two-process and dump levels, including a node down during the delete. | ✓ VERIFIED | Codec, simulator, `TestStand_deleteReplication`, `TestStand_deleteWhileNodeDown` — the stand package ran fresh twice today under the red gate with zero unexpected failures, plus `make test` ok. |
| 29 | Watch cut/backpressure and Send-error branches are covered. | ✓ VERIFIED | `TestWatch_sendErrorDisconnectsSubscriber` (fake-stream snapshot+relay failures, broken-transport isolation) + cut/shed tests green; `internal/grpcapi` ok 98.5 % cov, and the package ran fresh today under the red gate. |
| 30 | Metrics are tested by observed effect: exact `/metrics` values and one observed-effect test per configuration parameter. | ✓ VERIFIED | `TestStand_metricsByEffect` (exact `merges_conflicts_total == 1`, `discarded_total{class="bad_crc"} == 1`, retransmits > baseline) + five parameter subtests — stand package fresh-green twice today. |
| 31 | Every owner decision 14.1–14.11 is referenced by name in at least one test, and a meta-test fails naming any missing one. | ✓ VERIFIED | `decisionIndex` (11 rows) + `TestDecisions_referencedByTests` green in `make test` today; decision-coverage gate 9/9 (below). |
| 32 | Property tests draw half their versions from a small `(l, c, node)` space and the printed replay command works. | ✓ VERIFIED | `smallSpaceVersion` 3×3×3 draws + `TestProp_replayCommandExecutes` green; storage ok 100 % cov today. Unchanged. |
| 33 | A scheduled CI workflow exists with the D-09 job set, nightly cron + `workflow_dispatch`, explicit timeouts, pinned actions and least privilege. | ✓ VERIFIED | `scheduled.yml` re-read today: cron `30 2 * * *`, dispatch with validated `seeds`, seven jobs (`chaos-seeds`, `bench-smoke`, `red-run`, `race-flake`, `nightly-stand`, `red-r01-x20`, `deep-fuzz`), timeouts, pinned `checkout@v7`/`setup-go@v7` with `go-version-file: go.mod`. |
| 34 | TEST-16's acceptance tokens are on schedule: 1000 seeds, `-shuffle=on -count=3`, `-bench . -benchtime=1x`. | ✓ VERIFIED | chaos-seeds job defaults `SUPKV_SIM_FRESH_SEEDS` to 1000 and runs `-shuffle=on -count=3`; `make bench-smoke` = `-bench . -benchtime=1x` over four packages (Phase 8 wired the core benchmarks in — re-run today, exit 0, 12 benchmarks). Dispatch `seeds=10000` documented. |
| 35 | The nightly real-socket run executes the review's section-15 set at 10 000 × 1 KiB: green F9 in nightly-stand, red R-01/R-02/R-03 in red-run. | ✓ VERIFIED | One `SUPKV_STAND_NIGHTLY=1` contract selects documented scale in both jobs (re-read today); red-run invokes `bash scripts/red-runner`. Local documented-scale evidence on record (F9 23.94 s / R-01 20 000 keys 92.30 s on 2026-11-23; Phase 3 R-01 97.06 s). Cron execution remains the workflow's duty. |
| 36 | Benchmarks report p50/p99 without self-pacing on the real `persistfile`, and the smoke is CI-runnable. | ✓ VERIFIED | Fresh `make bench-smoke` today → exit 0: 12 benchmarks across `persist`, `repl`, `udprepl`, `grpcapi`, all with p50-ms/p99-ms (`BenchmarkDump10kKeys` 116.3 ms p50/p99 at 5 147 571 dump-bytes on real `persistfile.New`+`SaveStream`; `ConfirmationLatency` now fast/default profiles per Phase 8 08-08). The shared nearest-rank percentile lives in `internal/testutil/bench.go` (`LatencyPercentiles`) with the `internal/app/internal/testutil` wrapper keeping the TEST-10 call-site spelling — all three original call sites verified wired today. |
| 37 | TEST-11's command is defined as `make race-soak` with `COUNT ?= 20` over the affected set including `internal/stand`. | ✓ VERIFIED | Makefile re-read today: `COUNT ?= 20`; race-soak spans cmd/sup-kv, udprepl, app/repl, app/persist, persistfile, grpcapi, stand. |
| 38 | TEST-11's acceptance: `go test -race -count=20` over `cmd/sup-kv`, `udprepl` and affected packages green three times in a row (ROADMAP SC 5). | ✓ VERIFIED | Fresh run today on the current tree: `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/` → exit 0 (cmd/sup-kv 327 s, udprepl 47.8 s). Three consecutive green runs were recorded 2026-11-24 on this source-identical tree; the claim-while-held allocator (08-06 `7aeec35`/`074ccd0`) and the startup-bound fixes (08-09 `9cdc0f1`) are in place and re-verified (`main_test.go` uses `testutil.TCPPort`/`UDPPort` over `CmdBand()`; probe retries on fresh ports). |

**Score:** 38/38 truths verified (0 present, behavior-unverified)

### R-01 Local Bound Sensitivity (documented host-speed class — not a gap)

Today's two fresh red-gate runs both errored on exactly one row: `TestRed_R01_pacedSync20k` measured 7 899/8 000 and 7 890/8 000 keys (98.6–98.7 %) when its 20.6 s default-scale bound closed, on this host running the `powersave` governor (cpufreq samples ~900 MHz during the runs). This is the precise failure class the project already adjudicated:

- **03-12 halt record** (`03-12-SUMMARY.md`): "CPU governor `powersave` at ~1.64 GHz made the wall-clock reds miss (R-01 7 453–7 734/8 000 at the 20.6 s bound)… Pre-fix same-host evidence: 1 504/8 000." The halt was cleared by the orchestrator resolution in the same file: the flips were accepted on a `performance` governor (3.88 GHz) with a full red-runner at 7/7 rows and mutants killed (evidence `20261121T225048Z`), and rows `TestRed_R01_pacedSync20k`/`TestRed_R02_burst` flipped green in `4a83919`.
- **03-13 decision** (STATE.md decisions log): "The 20/20 acceptance lives in a dedicated `red-r01-x20` scheduled job — `-count=20` at the documented 20 000-key scale… a failure after the flip is a transport verdict and fails CI." That job is intact in `scheduled.yml` (lines 149–183): `-count=20 -timeout 110m`, job timeout 120 min, both env gates, no skip/tolerance flags.
- **Not a regression:** `git diff 02ecad1..HEAD` over all source directories is empty — the tree is byte-identical to the one where the row ran green (`red-evidence/20261124T221356Z`: 7/7 rows OK + full mutation stage) and where count-20 passed three times. The red tests' pinning mechanism, the runner, and the expectation table are unchanged; the v1.0 red evidence (19 % progress class) and the green margin differ from today's 98.7 % near-miss by an order of magnitude.
- **The harness behaved correctly today:** a green row receiving a fail event MUST error the run (D-01) — that is affirmative evidence for truth 13, not a defect in it.

Disposition: recorded as a host-environment condition with a standing governance decision; the machine-reproducible acceptance for R-01 is the scheduled `red-r01-x20` job, per the Phase 3 decisions. No must-have is failed or weakened by it.

### Deferred Items

None. No truth is addressed by a later milestone phase rather than this one.

### Advisory (New Scope, Unevidenced)

None. The re-verification scan produced no new-scope findings: no debt markers (`TBD`/`FIXME`/`XXX`), placeholders, stubs, or empty implementations in any of the 23 files changed since the previous verification (full scan recorded below).

### Plan-Level Must-Have Coverage

| Plan | Requirement(s) | Outcome | Status |
|------|----------------|---------|--------|
| 02-01 | TEST-04 | env-gated R-03 gate red test; red-runner + expectation table + evidence; `make red-run` | ✓ VERIFIED |
| 02-02 | TEST-05, TEST-03 | `internal/stand` harness moved wholesale; four migrated scenarios; F2c pin; `cmd/sup-kv/scenario_test.go` deleted | ✓ VERIFIED |
| 02-03 | TEST-06, TEST-02 | simnet `Delete`, `ReplaceNodeWithState`, receive budget, size gate, announce delay; import block stdlib+app+port | ✓ VERIFIED |
| 02-04 | TEST-03 | fsops seam (five methods), recording/failing fakes, blocking-Save proof | ✓ VERIFIED |
| 02-05 | TEST-04 | four stand red tests with documented bounds; nightly scale selection | ✓ VERIFIED |
| 02-06 | TEST-01 | drift-checked corpus, corpus+fresh iteration, poll-to-model stabilization, deletions | ✓ VERIFIED |
| 02-07 | TEST-02, TEST-08 | corrupted-datagram table (16 rows), deletion codec round trip, two-process deletion levels | ✓ VERIFIED |
| 02-08 | TEST-14, TEST-12 | Watch Send-error coverage; metrics-by-effect exact values; per-parameter effects | ✓ VERIFIED |
| 02-09 | TEST-13, TEST-15 | 11-decision coverage meta-test + citations; small-space draws; executable replay command | ✓ VERIFIED |
| 02-10 | TEST-10, TEST-11 | p50/p99 helper wired into benchmarks; `bench-smoke`/`race-soak`; flake audit closed by Phase 8 (allocator green, count-20 fresh-green today) | ✓ VERIFIED |
| 02-11 | TEST-16, TEST-06, TEST-08, TEST-11 | nightly scale wiring; scheduled workflow; ci.yml tightening | ✓ VERIFIED |
| 02-12 | TEST-04 | P-01 process + engine red tests; 7-row expectation table; full recorded red run | ✓ VERIFIED |
| 02-13 | TEST-01, TEST-09 | combined-failure/long-partition tests; red-runner overlay mutant stage | ✓ VERIFIED |
| 02-14 | TEST-05 | loud darwin report + build-tag boundary guard | ✓ VERIFIED |
| 02-15 | TEST-06, TEST-02 | six simulator extension tests incl. same-seed replay gate | ✓ VERIFIED |
| 02-16 | TEST-07 | five-row `persist.Open` clock-seeding proof on the real composition path | ✓ VERIFIED |
| 02-17 | TEST-08 | `FuzzDecode`/`FuzzDumpDecode` + 12/8 committed corpus entries | ✓ VERIFIED |

Artifact gate: `verify.artifacts` over all 17 plans → 31/31 artifacts exist and pass (re-run today).

### Phase Evidence: Red → Green

The phase's central deliverable is intact and the Phases 3–5 flips are visible end to end:

| Expectation row | Committed 2026-11-21 evidence | Current tree | Flip commit |
|-----------------|-------------------------------|--------------|-------------|
| `TestRed_R03_gateClosedUntilFullList` | red / fail,run | green — fresh pass today ×2 | `a44e661` (04-13) |
| `TestRed_R01_pacedSync20k` | red / fail,run | green `20261124T221356Z`; fresh runs today measured 98.6–98.7 % at default scale on a throttled host — governed host-speed class (see above) | `4a83919` (03-12) |
| `TestRed_R02_unackedDrain` | red / fail,run | green — fresh pass today ×2 | `276c0b3` (03-12) |
| `TestRed_R02_burst` | red / fail,run | green — fresh pass today ×2 | `4a83919` (03-12) |
| `TestRed_R03_f14ReplacedNode` | red / fail,run | green — fresh pass today ×2 | `a44e661` (04-13) |
| `TestRed_P01_valueLimit` | red / fail,run | green — fresh pass today ×2 | `7851cfc` (05-12) |
| `TestRed_P01_valueLimitEngine` | red / fail,run | green — fresh pass today ×2 | `7851cfc` (05-12) |

`scripts/red-expectations.tsv` carries exactly the seven rows, all `green`, one flip commit per fix phase. The red-gate mechanism is unchanged: a `red` row requires a fail event, a `green` row forbids one, a skip/absence/fail-for-green is an error, an unexpected failing test is an error.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/stand/stand.go` | Linux real-socket harness | ✓ VERIFIED | Build-once binary, shared claim-while-held port bands, fatal probe, lifecycle, deadline-clamped waits, dump/log seams (re-read today post-Phase 8) |
| `internal/stand/stand_test.go` | Migrated scenarios + F2c + nightly scale | ✓ VERIFIED | killNineRestart, bothNodesDown, sigtermFinalDump, emptyDiskRecovery, oversizeKeyF2c — fresh-green today ×2 |
| `internal/stand/stand_red_test.go` | R-01/R-02/R-03/P-01 red tests | ✓ VERIFIED | Five env-gated tests, documented bounds, nightly scale selection — 6/6 non-R-01 rows fresh-green today ×2 |
| `internal/stand/stand_delete_test.go` | Two-process deletion levels | ✓ VERIFIED | Fresh-green today ×2 (no unexpected fails in the stand package runs) |
| `internal/stand/stand_metrics_test.go` | Metrics by effect + parameter effects | ✓ VERIFIED | Fresh-green today ×2 |
| `internal/stand/stand_nondarwin_test.go` | Darwin report | ✓ VERIFIED | Named statement + build-tag guard intact |
| `scripts/red-runner` | Expected-red gate + mutants | ✓ VERIFIED | Parser, table assertion, evidence, mutant stage — re-executed today (expectation stage ×2, mutant stage ×1) |
| `scripts/red-expectations.tsv` | 7-row table, all flipped green | ✓ VERIFIED | All rows `green`; flip history one commit per fix |
| `red-evidence/*` | Recorded red and green runs | ✓ VERIFIED | Committed `20261121T145156Z` (all seven red), `20261124T221356Z` (all seven green + mutation stage); fresh verifier runs `20261127T161311Z`/`20261127T161629Z` |
| `internal/app/internal/simnet/simnet.go` | Additive simulator API | ✓ VERIFIED | Delete, ReplaceNodeWithState, budget, size gate, announce delay, counters; import boundary re-checked |
| `internal/app/internal/simnet/simnet_ext_test.go` | Six extension tests | ✓ VERIFIED | Green in suite today |
| `internal/app/repl/sim_chaos_test.go` | Stabilized chaos | ✓ VERIFIED | Corpus+fresh iteration, poll-to-model, deletions — fresh fixed-seed probe today |
| `internal/app/repl/testdata/chaos-seeds.txt` | Committed corpus | ✓ VERIFIED | 16 seeds; drift-checked |
| `internal/app/repl/sim_scenario_test.go` | TEST-09 scenarios | ✓ VERIFIED | Green in suite today (repl fresh 27.1 s) |
| `internal/app/repl/decision_coverage_test.go` | TEST-13 meta-test | ✓ VERIFIED | 11-row index; green in suite today |
| `internal/persistfile/fsops.go` + `dumpfile.go` + `fsops_test.go` | Five-op durability seam | ✓ VERIFIED | Order + failure matrix green; persistfile fresh-green under red gate today |
| `internal/app/persist/engine_blocking_test.go` | Blocking-Save proof | ✓ VERIFIED | Green in suite today |
| `internal/persistfile/open_seed_test.go` | TEST-07 composition proof | ✓ VERIFIED | Five rows green |
| `internal/persistfile/red_value_limit_test.go` | P-01 engine red test | ✓ VERIFIED | Fresh-green today ×2 |
| `internal/udprepl/decode_negative_test.go` | Negative table + deletion codec | ✓ VERIFIED | 16 rows; green in suite today |
| `internal/udprepl/fuzz_decode_test.go` + corpus | FuzzDecode + 12 seeds | ✓ VERIFIED | Seeds load under plain `go test` |
| `internal/app/persist/fuzz_dump_test.go` + corpus | FuzzDumpDecode + 8 seeds | ✓ VERIFIED | Seeds load under plain `go test` |
| `internal/grpcapi/red_gate_test.go` | R-03 gate-level red test | ✓ VERIFIED | Fresh-green today ×2 |
| `internal/grpcapi/watch_senderr_test.go` | Watch Send-error branch | ✓ VERIFIED | Green in suite today |
| `internal/app/storage/prop_small_space_test.go` | TEST-15 property + replay executor | ✓ VERIFIED | Green in suite today (storage 100 % cov) |
| `internal/testutil/bench.go` + wrapper | p50/p99 reporter | ✓ VERIFIED | Nearest-rank implementation; three+ call sites verified today post-Phase 8 refactor |
| `internal/app/internal/testutil/rng.go` | Working replay hint | ✓ VERIFIED | Package-scoped command; shared with replay executor |
| `.github/workflows/scheduled.yml` | Scheduled set | ✓ VERIFIED | Seven jobs incl. `red-r01-x20`; cron + dispatch; timeouts; pinned actions |
| `.github/workflows/ci.yml` | Named non-skippable stand step | ✓ VERIFIED | `go test -race ./internal/stand/` present; 4-cell matrix |
| `Makefile` | red-run / bench-smoke / race-soak | ✓ VERIFIED | `COUNT ?= 20`; race-soak includes stand; bench-smoke spans four packages |
| `cmd/sup-kv/scenario_test.go` | Deleted (moved) | ✓ DELETED | Intentional (D-03: moved, never duplicated); zero `TestExec_` remain |
| `internal/udprepl/repl.go` | (historical path) | ✓ SPLIT | Split into `codec.go`/`mirror.go` by later phases; both present and wired |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `scripts/red-runner` | red tests in grpcapi/stand/persistfile | `go test -p 1 -race -json` events matched to the TSV | ✓ WIRED | Fresh runs today parsed 7 rows and enforced the contract in both directions |
| `scripts/red-runner` | `internal/app/storage/store.go`, `internal/app/repl/core.go` | temp-copy sed + `go test -overlay` | ✓ WIRED | Mutant stage re-executed today: both kills, control pass, no-op/compile/timeout guards |
| `internal/grpcapi/red_gate_test.go` | `internal/grpcapi/gate.go` | gate behavior under partial vs full listing | ✓ WIRED | Fresh-green today ×2 |
| `internal/stand/stand.go` | `cmd/sup-kv/main.go` | build-once `go build` of the real binary | ✓ WIRED | Real process lifecycle exercised in today's stand runs |
| `internal/stand/stand_red_test.go` | documented review bounds | bounds derived from flags, citations | ✓ WIRED | Bounds are expressions (`2×400ms + 16s + 4.2s` observed in today's failure text) |
| `internal/app/repl/sim_chaos_test.go` | `testdata/chaos-seeds.txt` | corpus iteration + drift-checked generator | ✓ WIRED | Fresh fixed-seed probe today |
| `internal/app/internal/simnet/simnet_ext_test.go` | `internal/udprepl/replproto` | test-layer encoder injection into `Policy.SizeOf` | ✓ WIRED | `simnet.go` imports no replproto (re-checked) |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `Cluster.Delete` → `store.Delete` → `NotifyLocalWrite` | ✓ WIRED | Suite green today |
| `internal/persistfile/fsops.go` | `dumpfile.go`/`snapshot.go` | `saveAtomically` five-call sequence | ✓ WIRED | Order test proves the production Save path |
| `internal/persistfile/open_seed_test.go` | `internal/app/persist/engine.go` | `persist.Open` + real `FileStore`/codec | ✓ WIRED | Five rows green |
| `internal/udprepl/decode_negative_test.go` | `codec.go`/`mirror.go` | decoder verdicts + `Adapter.Discards()` | ✓ WIRED | Green in suite today |
| `internal/stand/stand_delete_test.go` | `internal/stand/stand.go` | harness dump/Get/Delete seams | ✓ WIRED | Fresh-green today ×2 |
| `internal/grpcapi/watch_senderr_test.go` | `internal/grpcapi/watch.go` | fake stream Send failures + real transport break | ✓ WIRED | Green in suite today |
| `internal/stand/stand_metrics_test.go` | `internal/metrics/metrics.go` | live HTTP `/metrics` rendered samples | ✓ WIRED | Fresh-green today ×2 |
| `internal/app/repl/decision_coverage_test.go` | `docs/adr/0001…0011` | index + word-bounded citation regex | ✓ WIRED | All 11 ADRs exist; gate 9/9 below |
| `internal/app/storage/prop_small_space_test.go` | `internal/app/internal/testutil/rng.go` | `testutil.ReplayCommand` executed | ✓ WIRED | Green in suite today |
| `.github/workflows/scheduled.yml` | `scripts/red-runner`, `Makefile`, `internal/stand/*`, corpus | job commands | ✓ WIRED | Re-read today; single-definition targets |
| `.github/workflows/ci.yml` | `internal/stand/` | `go test -race ./internal/stand/` | ✓ WIRED | Present before the full-suite step |
| `cmd/sup-kv/examples_test.go` | README/EXAMPLES blocks | doc tests with clamped deadlines | ✓ WIRED | Phase 8 serialized the grpcurl walkthroughs (`8ee2999`); cmd/sup-kv ok today incl. count-20 |
| `cmd/sup-kv/main_test.go` | `internal/testutil/ports.go` | shared claim-while-held allocator (`TCPPort`/`UDPPort`, `CmdBand`) | ✓ WIRED | Count-20 acceptance green today — the seam the fix landed through |
| `internal/stand/stand_test.go` | `cmd/sup-kv/scenario_test.go` | moved source | — HISTORICAL | Intentional deletion (D-03) |
| `internal/stand/stand_nondarwin_test.go` | `docs/adr/0010-darwin-development-only.md` | citation in the loud statement | ✓ WIRED | ADR exists |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `scripts/red-runner` | parsed `Action`/`Test`/`Package` | live `go test -json` of the spawned command | Yes — 6 pass events green today; the R-01 fail event errored the run as designed | ✓ FLOWING |
| `stand_metrics_test.go` | metric samples | live node `/metrics` over HTTP | Yes — exact constructed values (fresh ×2 today) | ✓ FLOWING |
| `sim_chaos_test.go` | seeds / model / stores | committed corpus + fresh wall-clock seeds + production store | Yes — fixed-seed probe fresh today | ✓ FLOWING |
| `stand_delete_test.go` | tombstone lines | real node dump files on disk | Yes — fresh ×2 today | ✓ FLOWING |
| `bench_test.go` (persist) | per-op durations, bytes | real `persist.EncodeTo` + `persistfile.FileStore.SaveStream` | Yes — 116.3 ms p50/p99, 5 147 571 dump-bytes today | ✓ FLOWING |
| `open_seed_test.go` | loaded records / clock versions | real codec decode of crafted dump files | Yes | ✓ FLOWING |
| `fuzz_decode_test.go` / `fuzz_dump_test.go` | datagram/dump bytes | committed `testdata/fuzz` corpora + fuzzer | Yes — both packages ok today | ✓ FLOWING |
| `main_test.go` port helpers | claimed ports | `internal/testutil/ports.go` registry (claim-while-held) | Yes — count-20 green today | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Strict lint gate | `make lint` | exit 0 (config verify + `run ./...`) | ✓ PASS |
| Full race suite | `make test` | exit 0; 15/15 packages ok (repl fresh 27.1 s; red tests skip) | ✓ PASS |
| Benchmark smoke | `make bench-smoke` | exit 0; 12 benchmarks, all with p50-ms/p99-ms | ✓ PASS |
| TEST-11 acceptance | `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/` | exit 0 (327.4 s + 47.8 s) — fresh today | ✓ PASS |
| Red gate, full run #1 | `bash scripts/red-runner` | exit 1: R-01 green row got fail (7 899/8 000 at the 20.6 s bound, throttled host); other 6 rows OK; evidence `20261127T161311Z` | ✓ PASS (contract enforced; governed host-speed class — see R-01 note) |
| Red gate, full run #2 | `bash scripts/red-runner` | exit 1: same single row, 7 890/8 000; evidence `20261127T161629Z` | ✓ PASS (same disposition) |
| Chaos mutation stage | runner-identical seds + overlays on seed 848612892191557850 | control PASS; both mutants killed; no-op/compile/timeout guards exercised | ✓ PASS |
| Red gate, full 7/7 + mutants (source-identical tree) | committed `20261124T221356Z` summary | 7/7 rows OK + 4/4 controls + both mutants killed | ✓ PASS |
| Count-20 ×3 (source-identical tree) | recorded 2026-11-24 21:09–21:23 UTC | three consecutive green runs | ✓ PASS |
| Darwin compile | `GOOS=darwin go vet ./internal/stand/` | exit 0 (2026-11-23; file unchanged) | ✓ PASS |
| Artifact declarations | `query verify.artifacts` over 17 plans | 31/31 pass (today) | ✓ PASS |
| Decision coverage | `query check.decision-coverage-verify` | 9/9 honored, `not_honored: []` (today) | ✓ PASS |
| Requirement accounting | plan frontmatter union vs REQUIREMENTS.md | 16/16 TEST IDs claimed, zero orphaned, zero extra | ✓ PASS |

### Probe Execution

No probe scripts are declared by this phase or present in the repository (`find scripts -path '*/tests/probe-*.sh'` → none). This phase is a test-harness/CI phase; the equivalent behavioral evidence is the spot-check table above.

**Status:** SKIPPED (no probes exist)

### Test Quality Audit

| Test File / Group | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-------------------|-----------|--------|---------|----------|-----------------|---------|
| `internal/grpcapi/red_gate_test.go` | TEST-04 | 1 (under red gate) | 1 env gate (by design) | No | Behavioral (gRPC codes + health) | ✓ |
| `internal/stand/stand_red_test.go` | TEST-04 | 5 (under red gate) | 5 env gates (by design) | No | Behavioral (real sockets, wall-clock bounds, metrics) | ✓ |
| `internal/persistfile/red_value_limit_test.go` | TEST-04 | 1 (under red gate) | 1 env gate (by design) | No | Behavioral (restart survival, real composition) | ✓ |
| `internal/stand/stand_test.go` + delete + metrics | TEST-02/03/05/06/12 | All | 0 | No | Behavioral | ✓ |
| `internal/app/internal/simnet/simnet_ext_test.go` | TEST-02/06 | 6 | 0 | No | Behavioral (state, counters, digests) | ✓ |
| `internal/app/repl/sim_chaos_test.go`, `sim_scenario_test.go` | TEST-01/09 | All | 0 | No | Behavioral (model equality, bounds, fault teeth) | ✓ |
| `internal/udprepl/decode_negative_test.go`, `fuzz_decode_test.go` | TEST-02/08 | All | 0 | No | Value/Behavioral | ✓ |
| `internal/app/persist/fuzz_dump_test.go` | TEST-08 | 1 fuzz target | 0 | No | Behavioral (codec contract) | ✓ |
| `internal/persistfile/fsops_test.go`, `open_seed_test.go` | TEST-03/07 | All | 0 | No | Behavioral | ✓ |
| `internal/app/storage/prop_small_space_test.go` | TEST-15 | 2 | 0 | No | Behavioral (CRDT laws, executed replay) | ✓ |
| Phase 8 additions on covered paths (`ports_test.go`, `bench_test.go`, `perf_test.go`, `performance_test.go`) | (later phase) | All | 0 | No | Behavioral/value (allocator bands, percentile gates, ledger anchors) | ✓ |

**Disabled tests on requirements:** 0 blockers — the only requirement-linked skips are the seven intentional `SUPKV_RED_TESTS` gates, and the red-runner executes each (pass events in committed green runs; fail/pass events observed in today's runs). No requirement is proven only by a skipped test.
**Circular patterns detected:** 0 — expected values are documented acceptance expressions, exact constructed events, or production-encoder fixtures.
**Insufficient assertions:** 0 for requirement-linked tests — value/behavioral level throughout.
**Coverage quantities:** met — 16 corrupted-datagram rows, 12 + 8 fuzz corpus entries, 6 simulator extension tests, 5 clock-seeding rows, 7 expectation rows, 11 decision citations.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No `TBD`/`FIXME`/`XXX` in any of the 23 files changed since the previous verification, nor in phase-modified files | — | None |
| — | — | No placeholder/coming-soon/not-implemented text | — | None |
| — | — | No empty-return stubs or console-only implementations | — | None |
| — | — | No hardcoded empty data at assertion sites | — | None |

The only non-red skip on the tree is the loud, environment-conditional root-permission skip in `internal/stand/persist_test.go:380` (later-phase test, names its cause). No debt-marker blocker exists.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts — **9/9, `not_honored: []`** (decision-coverage gate, re-run today; non-blocking).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|-------------|--------|----------|
| TEST-01 | 02-06, 02-13 | Chaos stable + mutation-sensitive | ✓ SATISFIED | Drift-checked corpus + fresh seeds; both mutants kill (fresh today); control passes; 10k dispatch scheduled |
| TEST-02 | 02-03, 02-07, 02-15 | Deletion replication at every level | ✓ SATISFIED | Codec, simulator, two-process, dump, node-down — all green today |
| TEST-03 | 02-02, 02-04 | Confirmed write is on disk, pinned by tests | ✓ SATISFIED | fsops order + failure matrix + blocking Save + process case — green today |
| TEST-04 | 02-01, 02-05, 02-12 | Criticals pinned red then flipped green | ✓ SATISFIED | Committed red evidence; all flips committed (`4a83919`/`a44e661`/`7851cfc`); 6/7 rows fresh-green today ×2; R-01 governed by the scheduled 20× job per 03-12/03-13 |
| TEST-05 | 02-02, 02-14 | Real-socket tests never silently skipped | ✓ SATISFIED | Zero `Skipf`; fatal probes; named darwin report; CI runs `-race` |
| TEST-06 | 02-03, 02-11, 02-15 | Simulator blind spots closed + nightly 10k | ✓ SATISFIED | All extensions green today; nightly wiring intact |
| TEST-07 | 02-16 | `persist.Open` clock seeding on real composition | ✓ SATISFIED | Five-row proof green today |
| TEST-08 | 02-07, 02-11, 02-17 | Decoder negative tests + fuzzing | ✓ SATISFIED | 16-row table + 12/8 corpora + scheduled `-fuzztime` jobs |
| TEST-09 | 02-13 | Bounds under combined failures and long partitions | ✓ SATISFIED | Both scenario tests green today |
| TEST-10 | 02-10 | Benchmarks measure what they claim | ✓ SATISFIED | Shared nearest-rank p50/p99 across all call sites; smoke green today |
| TEST-11 | 02-10, 02-11 | Known flakiness classes eliminated | ✓ COMPLETE | Claim-while-held allocator + startup-bound fixes in place; count-20 acceptance green fresh today (and 3× on 2026-11-24, source-identical tree) |
| TEST-12 | 02-08 | Metrics/configuration by observed effect | ✓ SATISFIED | Exact values + five parameter subtests green today |
| TEST-13 | 02-09 | Every section-14 decision has a named test | ✓ SATISFIED | Meta-test green; gate 9/9 |
| TEST-14 | 02-08 | Watch cut and backpressure branches covered | ✓ SATISFIED | Send-error + cut/shed green today |
| TEST-15 | 02-09 | Property shrinking + small-space exploration | ✓ SATISFIED | Green today; replay executor wired |
| TEST-16 | 02-11 | Scheduled CI workflow exists | ✓ SATISFIED | Seven-job schedule with the exact acceptance tokens, re-read today |

**Requirement accounting:** all 16 TEST IDs appear in PLAN frontmatter (02-01…02-17), all map to Phase 2 in REQUIREMENTS.md, none orphaned, none extra. 16/16 satisfied.

### Human Verification Required

N/A — infrastructure/foundation phase with no user-facing elements. All acceptance criteria are programmatically verifiable, and every programmatic check above was executed on the live tree today. No truth was left behavior-unverified (`behavior_unverified: 0`). The remaining post-merge duties (nightly cron execution, the `seeds=10000` dispatch, the `red-r01-x20` scheduled run, the macOS cell) are the scheduled/CI evidence vehicles the roadmap assigns to the workflow, not human UAT items.

### Gaps Summary

None. The phase's central deliverable is intact on the current tree: the stand and the hardened simulator exist, all 16 requirements have their tests, the seven red-gated Critical tests were recorded RED against v1.0 in committed evidence, the expectation table reflects the Phase 3–5 flips (one flip commit per fix), and the red-runner still enforces its contract — demonstrated today in the strongest possible way, by erroring on the one row whose wall-clock bound this throttled host (powersave, ~900 MHz) cannot currently meet at default CI scale. That R-01 sensitivity is the host-speed class adjudicated in Phase 3 (03-12 halt record + full-speed clearing evidence; 03-13's dedicated `red-r01-x20` scheduled job carrying the 20/20 documented-scale acceptance, where a failure is a transport verdict that fails CI), on a tree byte-identical in source to the one where the row ran green and count-20 passed three times. Today's fresh evidence — `make lint` green, `make test` green (15/15 packages), `make bench-smoke` green (12 benchmarks with p50/p99), count-20 acceptance green, both mutants killed with the control passing, 31/31 artifacts, 16/16 requirements, 9/9 decisions — confirms every must-have.

---

_Verified: 2026-11-27T19:45:00Z_
_Re-verified: 2026-11-27 — stale-fingerprint regeneration over the post-Phase-8 tree; no gaps existed or remain_
_Verifier: Claude (gsd-verifier)_
