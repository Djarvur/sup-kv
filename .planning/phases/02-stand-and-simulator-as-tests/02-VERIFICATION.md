---
phase: 02-stand-and-simulator-as-tests
verified: 2026-11-21T15:00:25Z
status: passed
score: 37/37 must-haves verified
covered_files:
  - ".github/workflows/ci.yml"
  - ".github/workflows/scheduled.yml"
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
  - "internal/app/internal/simnet/simnet.go"
  - "internal/app/internal/simnet/simnet_ext_test.go"
  - "internal/app/internal/testutil/bench.go"
  - "internal/app/internal/testutil/rng.go"
  - "internal/app/persist/bench_test.go"
  - "internal/app/persist/codec_test.go"
  - "internal/app/persist/engine_blocking_test.go"
  - "internal/app/persist/engine_test.go"
  - "internal/app/persist/fuzz_dump_test.go"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_00_accepted_with_tombstone"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_01_accepted"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_02_generation_fault"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_03_generation_fault"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_04_per_record_discard"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_05_generation_fault"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_06_generation_fault"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_07_per_record_discard"
  - "internal/app/repl/decision_coverage_test.go"
  - "internal/app/repl/sim_chaos_test.go"
  - "internal/app/repl/sim_scenario_test.go"
  - "internal/app/repl/testdata/chaos-seeds.txt"
  - "internal/app/storage/prop_small_space_test.go"
  - "internal/grpcapi/bench_test.go"
  - "internal/grpcapi/gate_test.go"
  - "internal/grpcapi/red_gate_test.go"
  - "internal/grpcapi/server_test.go"
  - "internal/grpcapi/watch_senderr_test.go"
  - "internal/persistfile/dumpfile.go"
  - "internal/persistfile/fsops.go"
  - "internal/persistfile/fsops_test.go"
  - "internal/persistfile/open_seed_test.go"
  - "internal/persistfile/red_value_limit_test.go"
  - "internal/persistfile/snapshot.go"
  - "internal/stand/stand.go"
  - "internal/stand/stand_delete_test.go"
  - "internal/stand/stand_metrics_test.go"
  - "internal/stand/stand_nondarwin_test.go"
  - "internal/stand/stand_red_test.go"
  - "internal/stand/stand_test.go"
  - "internal/testutil/bench.go"
  - "internal/udprepl/bench_test.go"
  - "internal/udprepl/decode_negative_test.go"
  - "internal/udprepl/fuzz_decode_test.go"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_00_valid_hello"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_01_valid_data_replication"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_02_valid_tombstone"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_03_valid_ack"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_04_valid_data_request"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_05_valid_data_sync"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_06_corruption_undecodable"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_07_corruption_bad_crc"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_08_corruption_undecodable"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_09_corruption_undecodable"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_10_corruption_wrong_proto_version"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/seed_11_corruption_undecodable"
  - "scripts/red-expectations.tsv"
  - "scripts/red-runner"
covered_digest: "v4:sha256:70920e3a22335517fcb3103e93741a2fc598c057e87911e143421e04f475da46"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 02: Stand and Simulator as Tests — Verification Report

**Phase Goal:** The review's verdict is reproducible by machine — every claimed miss (recovery, burst) and every Critical finding is pinned by an automated test on real sockets or a hardened simulator, RED where the code is wrong. The tests are expected to fail against v1.0 code; that is the point — Critical findings get failing tests before fixes, and Phases 3-5 turn them green.

**Verified:** 2026-11-21T15:00:25Z
**Status:** passed
**Re-verification:** No — initial verification
**Verification basis:** live repository at `dd6eb2a` (branch `gsd/v1.1-milestone`), all 17 PLAN/SUMMARY files, REQUIREMENTS.md TEST-01…TEST-16, the committed red evidence, and fresh local executions of the red gate, the green suite, the lint gate, benchmarks, race soak, and the documented-scale stand scenario.

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A real-socket stand exists that spawns the real `cmd/sup-kv` binary over loopback UDP + gRPC on Linux, and CI runs it with `-race`. | ✓ VERIFIED | `internal/stand/stand.go` (`//go:build linux`, build-once binary, port window 28100–32678, probe, SIGKILL/SIGTERM lifecycle); `.github/workflows/ci.yml` has exactly one `go test -race ./internal/stand/` step; `make test` runs `go test -race -cover ./...`; `internal/stand` coverage 83.2%. |
| 2 | F9: an empty-disk node recovers 10 000 × 1 KiB keys within the derived bound at the documented nightly scale. | ✓ VERIFIED | `TestStand_emptyDiskRecovery` with `SUPKV_STAND_NIGHTLY=1` selects `emptyDiskKeysNightly=10000`, `emptyDiskValueBytes=1024`, documented flags; fresh run `SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestStand_emptyDiskRecovery$'` → `ok 143.835s` (executor recorded 55.6 s recovery inside the 85 s bound). |
| 3 | F9b: the pure-resync recovery scenario is pinned as a red test with the documented expression `2×sync_interval + N/sync_response_rate + margin`, and fails against v1.0. | ✓ VERIFIED | `TestRed_R01_pacedSync20k`: seeds then restarts A so only anti-entropy remains; asserts the documented bound. Fresh full red-run: timed out after 20.6 s with the 8 000-key bound (`2×200ms + 16s + 4.2s`) — R-01. `SUPKV_STAND_NIGHTLY=1` selects `redR01KeysNightly=20000` (+30 s sync, 1 s hello). |
| 4 | F14: the replaced node stays closed until the full multi-datagram listing, never overwrites the peer's user value, answers `AlreadyExists` after the listing, and takes a greater version on its next local write — and fails against v1.0. | ✓ VERIFIED | `TestRed_R03_f14ReplacedNode` asserts (a) `Unavailable` reads/writes before the listing, (b) A's value unchanged, (c) `NOT_SERVING`, (d) `AlreadyExists` after, (e) local version > listing max. Fresh run fails in 2.49 s: `want code Unavailable, got NotFound` (reads never consult the gate). |
| 5 | The 1000 writes/s × 15 s burst over 10 000 distinct keys is a red test asserting ≥990 writes/s and an unacked set empty 5 s after the send window. | ✓ VERIFIED | `TestRed_R02_burst` drives the open-loop schedule (1000/s × 15 s, 10 000 keys, 1024 workers/8 conns), asserts `achieved >= 990` and the 5 s drain bound from `/metrics`. Fresh red run: `25.1 writes/s` (3232/15000 OK, 11768 DeadlineExceeded) → FAIL. |
| 6 | F2c: a 70 000-character key is accepted locally, produces `sendto` failures, and never replicates to B. | ✓ VERIFIED | `TestStand_oversizeKeyF2c`: Set returns OK with the review's 1-byte value, polls A's stderr for the message-too-long line, holds the key absent from B for the derived window and absent from B's dump when it exists, and serves it on A; green in `make test`. |
| 7 | No socket test silently skips: a Linux capability-probe miss is a fatal, and the macOS cell reports "darwin replication is untested" through a named green test. | ✓ VERIFIED | Zero `Skipf` in `internal/stand`; `newStandPair` calls `t.Fatalf` on a probe miss and `standBinary` fatals on a failed build; the only `t.Skip` calls are the intentional `SUPKV_RED_TESTS` gates. `stand_nondarwin_test.go` (`!linux`) passes and logs the statement + build-tag boundary guard; `GOOS=darwin go vet ./internal/stand/` → exit 0. |
| 8 | R-01 fails against v1.0 code with `SUPKV_RED_TESTS=1`. | ✓ VERIFIED | Fresh full `bash scripts/red-runner` run: `TestRed_R01_pacedSync20k` observed `fail,run` — timed out at the documented bound with 8 000 keys. Nightly 20 000-key scale is wired and was recorded red by the executor (388 s). |
| 9 | R-02 unacked-set cost and burst drain fail against v1.0 code. | ✓ VERIFIED | Fresh red run: `TestRed_R02_unackedDrain` timed out after the 35 s bound (10 000 unacked precondition reached); `TestRed_R02_burst` achieved 25.1/s. Both `fail,run`. |
| 10 | R-03 closed-until-full-list fails against v1.0 at gate level and on the stand. | ✓ VERIFIED | `TestRed_R03_gateClosedUntilFullList` fails: `Get … want code Unavailable, got OK`; `TestRed_R03_f14ReplacedNode` fails: `want code Unavailable, got NotFound`. |
| 11 | P-01 single-value-limit fails against v1.0 at process and engine level. | ✓ VERIFIED | `TestRed_P01_valueLimit` fails for both 63 000 and exactly 64 000 bytes after SIGKILL/restart ("dump holds the record but the loader discards it"); `TestRed_P01_valueLimitEngine` fails with `Open discarded 1 record(s): key not found`. |
| 12 | Every red test skips without `SUPKV_RED_TESTS=1`, so `make test` stays green. | ✓ VERIFIED | `go test -race ... -run '^TestRed_'` (no env) → all seven SKIP/PASS; `make test` → exit 0, 12 packages ok, no FAIL; red tests gate on the env var as their first statement. |
| 13 | `scripts/red-runner` asserts the expectation table from `go test -json` events, exits 0 on the full run, and records evidence. | ✓ VERIFIED | Fresh `bash scripts/red-runner` → exit 0: `7 expectation row(s) checked, 0 error(s)`; all seven rows `OK`; evidence written to `red-evidence/20261121T145156Z/{go-test.json,summary.txt}` (plus five committed runs). A skip/absence for a red row is an error; a green-row fail and an unexpected failing test are errors. |
| 14 | `TestChaos_convergence` is mutation-sensitive: both named mutants kill it on fixed corpus seeds while the control passes, and the production tree stays clean. | ✓ VERIFIED | Fresh runner mutant stage: 4/4 controls pass; `sync-without-tombstones` killed on seed 848612892191557850; `periodic-sync-disabled` killed on three corpus seeds; stage asserts compile, sed-no-op, timeout, corpus membership and a clean `internal/app/repl/core.go`. |
| 15 | Simulator `Cluster.Delete` replicates a tombstone and propagates `ErrNotFound` without burning a version. | ✓ VERIFIED | `simnet.go` `Delete` calls `store.Delete` → `NotifyLocalWrite` → route/drive/write log; `TestSim_Delete_replicatesTombstone` green (tombstone at the peer, missing/already-tombstoned `ErrNotFound`, no version burn). |
| 16 | `Cluster.ReplaceNodeWithState` restarts a node with its state preserved. | ✓ VERIFIED | `simnet.go` builds the node, sorted-key `store.Merge` seed, then ReplaceNode's swap/Reregister/timer supersession; `TestSim_ReplaceNodeWithState_preservesState` green with a plain-`ReplaceNode` empty-boot control. |
| 17 | The simulator enforces a per-node incoming-datagram budget, refusing and counting past it. | ✓ VERIFIED | `Network.SetRecvBudget` + `FaultStats.RecvBudgetRefused`; `TestSim_recvBudget_refusesAndCounts` green: 1 delivered, 2 refused/counted, 2 unconfirmed. |
| 18 | The simulator refuses oversize datagrams measured by the encoder and counts the refusals. | ✓ VERIFIED | `Policy.SizeOf`/`MaxDatagramBytes` + `FaultStats.OversizeRefused`; `TestSim_sizeRefusal_byRealEncoder` green using the generated `replproto` messages through `proto.Marshal` (real encoder types/wire shape) and a measured limit; `simnet.go` imports no `replproto`. |
| 19 | Record announcements can be delayed by a virtual dump latency. | ✓ VERIFIED | `Policy.AnnounceDelay` consulted for `port.DataReplication`; `TestSim_announceDelay_virtualDumpLatency` green: absent at write+20 ms, present at write+21 ms, Ack waits only the drawn 1 ms. |
| 20 | Same-seed replay is byte-identical with all simulator extensions active. | ✓ VERIFIED | `TestSim_determinism_withExtensions` green: identical `TraceDigest`, `FaultStats` and per-node stores, with activation teeth (`OversizeRefused>0`, `RecvBudgetRefused>0`). |
| 21 | Chaos runs a fixed drift-checked corpus plus fresh seeds each run and converges. | ✓ VERIFIED | `testdata/chaos-seeds.txt` (16 seeds) + `TestChaos_seedCorpusGenerator` always regenerates twice and byte-compares (drift fails; write gated); `SUPKV_SIM_FRESH_SEEDS` parameterizes fresh runs (default 1); `TestChaos_convergence` iterates corpus subtests + fresh runs; fresh `SUPKV_SIM_FRESH_SEEDS=10` run → ok 1.552s; `TestChaos_corpusEmptyStateProbe` green. The 10 000-seed proof is the dispatchable scheduled duty (`scheduled.yml` seeds input, `gh workflow run scheduled.yml -f seeds=10000`). |
| 22 | Combined failures and partitions longer than `peer_timeout` converge not before the first post-heal sync and not later than the documented bound, with no fresh loss. | ✓ VERIFIED | `TestScenario_combinedFailures` and `TestScenario_longPartition` (peerTableSurvives / peerEvicted) assert `boundPartitionHeal(...)+margin`, divergence teeth at the heal instant, and every pre-heal record present; both green under `-race`. |
| 23 | The fsops seam records exactly write → fsync → rotate → rename → dir-fsync, and a failure at each step surfaces with hygiene intact. | ✓ VERIFIED | `internal/persistfile/fsops.go` five-method seam driven by `saveAtomically`; `TestFsops_saveOperationOrder` asserts the exact ordered slice and argument paths; rotation chain, failure-at-each-step matrix, and empty-dump round trip green. |
| 24 | A blocking dump Save prevents `Set` from returning. | ✓ VERIFIED | `TestEngine_blockingSaveHoldsSet` (synctest bubble) green: Set has not returned while Save stands; release → Set returns confirmed with one completed save. |
| 25 | `persist.Open` seeds the clock from the dump watermark on the real composition path. | ✓ VERIFIED | `TestOpen_clockSeededFromDumpWatermark` composes real `FileStore` + codec + `persist.Open`; five subtests green: watermark ahead dominates, boundary-equal still greater, empty start, equal-max-l counter pair, restart idempotence. |
| 26 | Corrupted datagrams have a verdict/counter table, and deletion survives the codecs. | ✓ VERIFIED | `TestDecode_negativeTable` has 8 rows (bad CRC, truncated envelope/body, wrong proto version, bad run_id, oversize, empty key and tombstone-with-data as XPORT-10 forward references) asserting verdicts and `DiscardStats` deltas; `TestWire_deleteRoundTrip` covers ASCII + UTF-8 tombstone wire and dump codec round trips. |
| 27 | `FuzzDecode` and `FuzzDumpDecode` exist with committed in-repo corpora and invariant-only assertions. | ✓ VERIFIED | Two fuzz targets, one per package; `internal/udprepl/testdata/fuzz/FuzzDecode/` 12 entries and `internal/app/persist/testdata/fuzz/FuzzDumpDecode/` 8 entries, each in `go test fuzz v1` format; plain `go test` seed runs green; deep runs are the scheduled `-fuzztime` jobs. |
| 28 | Deletion replication is proven at codec, simulator, two-process and dump levels, including a node that was down during the delete. | ✓ VERIFIED | Codec `TestWire_deleteRoundTrip`; simulator `TestSim_Delete_replicatesTombstone` + chaos deletions; two-process `TestStand_deleteReplication` (Delete→NotFound on B, `"deleted":true` in B's dump, delete-twice single tombstone) and `TestStand_deleteWhileNodeDown` (learns after start) — all green. |
| 29 | Watch cut/backpressure and Send-error branches are covered. | ✓ VERIFIED | `TestWatch_sendErrorDisconnectsSubscriber` covers snapshot and live-relay Send failures, real broken-transport isolation/liveness/fresh-subscribe, and re-asserts cut/shed; existing `TestWatch_snapshotThenLive`, `TestWatch_exactCut`, `TestWatch_slowConsumer` remain green. |
| 30 | Metrics are tested by observed effect: exact `/metrics` values after the constructed conflict + retransmit + discard scenario, and one observed-effect test per configuration parameter. | ✓ VERIFIED | `TestStand_metricsByEffect` asserts `supkv_merges_conflicts_total == 1`, `supkv_datagrams_discarded_total{class="bad_crc"} == 1`, retransmits `>= 1` (named lower bound) from live HTTP `/metrics`; `TestConfig_parametersByEffect` has five named parameter subtests and the SUMMARY maps every remaining flag to an existing observing test. |
| 31 | Every owner decision 14.1–14.11 is referenced by name in at least one test, and a meta-test fails naming any missing one. | ✓ VERIFIED | `decisionIndex` enumerates all 11 decisions/ADRs/anchors; independent grep across `cmd/` and `internal/` found every token in repo test files outside the module cache; `TestDecisions_referencedByTests` green. |
| 32 | Property tests draw half their versions from a small `(l, c, node)` space and the printed replay command works. | ✓ VERIFIED | `smallSpaceVersion` uses 3×3×3 space and `mixedVersion` draws it half the time; storage merge laws + persist codec/watermark properties use it; `TestProp_replayCommandExecutes` executes the exact printed package-scoped command and asserts the same test re-runs and passes; `TestProp_mergeSmallSpace` green. |
| 33 | A scheduled CI workflow exists with the D-09 six-job set, nightly cron + `workflow_dispatch`, explicit timeouts, pinned actions and least privilege. | ✓ VERIFIED | `.github/workflows/scheduled.yml`: cron `30 2 * * *`, dispatch `seeds`; jobs chaos-seeds, bench-smoke, red-run, race-flake, nightly-stand, deep-fuzz; 6 `timeout-minutes`, 2 `-fuzztime`, 2 `SUPKV_STAND_NIGHTLY`, `permissions: contents: read`, checkout@v7/setup-go@v7 with `go-version-file: go.mod`. |
| 34 | TEST-16's acceptance tokens are on schedule: 1000 seeds, `-shuffle=on -count=3`, `-bench . -benchtime=1x`. | ✓ VERIFIED | chaos-seeds defaults `SUPKV_SIM_FRESH_SEEDS` to 1000 and runs `go test -race ./internal/app/repl/ -run '^TestChaos' -shuffle=on -count=3`; bench-smoke runs `make bench-smoke` = `-bench . -benchtime=1x` (single definition); dispatch `seeds=10000` documented for the one-off proof. |
| 35 | The nightly real-socket run executes the review's section-15 set at 10 000 × 1 KiB: the green F9 half in nightly-stand and the red R-01/R-02/R-03 half in red-run. | ✓ VERIFIED | One `SUPKV_STAND_NIGHTLY=1` contract selects documented scale in both jobs; nightly-stand runs `go test -race -count=1 ./internal/stand/`; red-run runs `bash scripts/red-runner` (runner exports `SUPKV_RED_TESTS=1`); local nightly F9 run green (143.835 s), local red-run reproduced all Criticals red. |
| 36 | Benchmarks report p50/p99 without self-pacing on the real `persistfile`, and the smoke is CI-runnable. | ✓ VERIFIED | Fresh `make bench-smoke` → exit 0; all three benchmarks print `p50-ms`/`p99-ms` (Dump10kKeys 129.3/129.3, ReplicationBurst 0.02766/0.02766, ConfirmationLatency 6.980/6.980); measured loop contains no sleeps/rate limiters; Dump10kKeys uses real `persistfile.FileStore`. |
| 37 | TEST-11's command is defined as `make race-soak` (`-race -count=20` over the affected set); local `COUNT=1` is green, and the ×3-green bar plus the 10 000-seed proof are scheduled duties. | ✓ VERIFIED | `Makefile` `race-soak` with `COUNT ?= 20`; fresh `make race-soak COUNT=1` → exit 0 (cmd/sup-kv, udprepl, app/repl, app/persist, persistfile, grpcapi all ok); `scheduled.yml` race-flake runs it nightly with 120 min; the ×3 bar spans three nightly runs (RESEARCH A1); the 10k dispatch is documented. |

**Score:** 37/37 truths verified (0 present, behavior-unverified)

### Deferred Items

None. The only "future-run" items are the scheduled post-merge executions (nightly cron, `seeds=10000` dispatch, macOS cell) that the roadmap explicitly assigns to the scheduled/CI machinery rather than to local phase execution; they are treated as specified-by-schedule, not as unmet deliverables.

### Plan-Level Must-Have Coverage

| Plan | Requirement(s) | Outcome | Status |
|------|----------------|---------|--------|
| 02-01 | TEST-04 | env-gated R-03 gate red test; red-runner + expectation table + evidence; `make red-run` | ✓ VERIFIED |
| 02-02 | TEST-05, TEST-03 | `internal/stand` harness moved wholesale; four migrated scenarios; F2c pin; `cmd/sup-kv/scenario_test.go` deleted | ✓ VERIFIED |
| 02-03 | TEST-06, TEST-02 | simnet `Delete`, `ReplaceNodeWithState`, receive budget, size gate, announce delay; import block stdlib+app+port | ✓ VERIFIED |
| 02-04 | TEST-03 | fsops seam (five methods), recording/failing fakes, blocking-Save proof | ✓ VERIFIED |
| 02-05 | TEST-04 | four stand red tests with documented bounds; nightly scale selection | ✓ VERIFIED |
| 02-06 | TEST-01 | drift-checked corpus, corpus+fresh iteration, poll-to-model stabilization, deletions | ✓ VERIFIED |
| 02-07 | TEST-02, TEST-08 | 8-row negative table, deletion codec round trip, two-process deletion levels | ✓ VERIFIED |
| 02-08 | TEST-14, TEST-12 | Watch Send-error coverage; metrics-by-effect exact values; per-parameter effects | ✓ VERIFIED |
| 02-09 | TEST-13, TEST-15 | 11-decision coverage meta-test + citations; small-space draws; executable replay command | ✓ VERIFIED |
| 02-10 | TEST-10, TEST-11 | p50/p99 helper wired into three benchmarks; `bench-smoke`/`race-soak`; flake-class audit | ✓ VERIFIED |
| 02-11 | TEST-16, TEST-06, TEST-08, TEST-11 | nightly scale wiring; six-job `scheduled.yml`; ci.yml tightening | ✓ VERIFIED |
| 02-12 | TEST-04 | P-01 process + engine red tests; 7-row expectation table; full recorded red run | ✓ VERIFIED |
| 02-13 | TEST-01, TEST-09 | combined-failure/long-partition tests; red-runner overlay mutant stage | ✓ VERIFIED |
| 02-14 | TEST-05 | loud darwin report + build-tag boundary guard | ✓ VERIFIED |
| 02-15 | TEST-06, TEST-02 | six simulator extension tests incl. same-seed replay gate | ✓ VERIFIED |
| 02-16 | TEST-07 | five-row `persist.Open` clock-seeding proof on the real composition path | ✓ VERIFIED |
| 02-17 | TEST-08 | `FuzzDecode`/`FuzzDumpDecode` + 12/8 committed corpus entries | ✓ VERIFIED |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/stand/stand.go` | Linux real-socket harness | ✓ VERIFIED | Build-once binary, reserved port window + registry, fatal probe, lifecycle, deadline-clamped waits, dump/log seams |
| `internal/stand/stand_test.go` | Migrated scenarios + F2c + nightly scale | ✓ VERIFIED | `TestStand_killNineRestart`, `bothNodesDown`, `sigtermFinalDump`, `emptyDiskRecovery` (nightly 10k×1KiB), `oversizeKeyF2c` |
| `internal/stand/stand_red_test.go` | R-01/R-02/R-03/P-01 red tests | ✓ VERIFIED | Five env-gated tests, documented bounds, `standNightly` scale selection |
| `internal/stand/stand_delete_test.go` | Two-process deletion levels | ✓ VERIFIED | Delete→NotFound, tombstone in dump, delete-twice, node-down node learns after start |
| `internal/stand/stand_metrics_test.go` | Metrics by effect + parameter effects | ✓ VERIFIED | Exact rendered `/metrics` values; five parameter subtests |
| `internal/stand/stand_nondarwin_test.go` | Darwin report | ✓ VERIFIED | Named passing test + statement + build-tag guard; `GOOS=darwin go vet` green |
| `scripts/red-runner` | Expected-red gate + mutants | ✓ VERIFIED | `go test -json` parser, table assertion, evidence, overlay mutant stage |
| `scripts/red-expectations.tsv` | Complete 7-row table | ✓ VERIFIED | All rows expected `red`; flip contract documented |
| `red-evidence/*` | Recorded red runs | ✓ VERIFIED | 5 committed runs + fresh verifier run `20261121T145156Z`; all 7 rows red |
| `internal/app/internal/simnet/simnet.go` | Additive simulator API | ✓ VERIFIED | Delete, ReplaceNodeWithState, SetRecvBudget, SizeOf/MaxDatagramBytes, AnnounceDelay, two counters |
| `internal/app/internal/simnet/simnet_ext_test.go` | Six extension tests | ✓ VERIFIED | Delete, restart, budget, size gate, announce delay, determinism replay |
| `internal/app/repl/sim_chaos_test.go` | Stabilized chaos | ✓ VERIFIED | Corpus generator/drift, corpus+fresh iteration, poll-to-model, deletions |
| `internal/app/repl/testdata/chaos-seeds.txt` | Committed corpus | ✓ VERIFIED | 16 deterministic seeds |
| `internal/app/repl/sim_scenario_test.go` | TEST-09 scenarios | ✓ VERIFIED | combined failures + long partition bounds/teeth/qualifier |
| `internal/app/repl/decision_coverage_test.go` | TEST-13 meta-test | ✓ VERIFIED | 11-row index; word-bounded citations; ADR existence check |
| `internal/persistfile/fsops.go` | Five-op durability seam | ✓ VERIFIED | write/fsync/rotate/rename/syncDir + real default impl + shared hygiene |
| `internal/persistfile/dumpfile.go` | Seam-driven Save | ✓ VERIFIED | `saveAtomically` drives the five calls; exported constructors unchanged |
| `internal/persistfile/fsops_test.go` | Order + failure matrix | ✓ VERIFIED | Exact order, rotation chain, five failure rows, empty round trip |
| `internal/app/persist/engine_blocking_test.go` | Blocking-Save proof | ✓ VERIFIED | Set does not return while Save stands |
| `internal/persistfile/open_seed_test.go` | TEST-07 composition proof | ✓ VERIFIED | Five rows on real FileStore + codec + Open |
| `internal/persistfile/red_value_limit_test.go` | P-01 engine red test | ✓ VERIFIED | Env-gated, real composition; currently red |
| `internal/udprepl/decode_negative_test.go` | 8-row negative table + deletion codec | ✓ VERIFIED | Verdict + counter deltas; XPORT-10 forward references; tombstone round trip |
| `internal/udprepl/fuzz_decode_test.go` + corpus | FuzzDecode + 12 seeds | ✓ VERIFIED | Invariant-only body; corpora load under plain `go test` |
| `internal/app/persist/fuzz_dump_test.go` + corpus | FuzzDumpDecode + 8 seeds | ✓ VERIFIED | Contract-only body; echo/checksum accounting |
| `internal/grpcapi/red_gate_test.go` | R-03 gate-level red test | ✓ VERIFIED | Env-gated; full PROTO-01 acceptance; currently red |
| `internal/grpcapi/watch_senderr_test.go` | Watch Send-error branch | ✓ VERIFIED | Snapshot + relay Send failures, transport isolation, cut/shed |
| `internal/app/storage/prop_small_space_test.go` | TEST-15 property + replay executor | ✓ VERIFIED | 3×3×3 draws; printed command executed |
| `internal/testutil/bench.go` + wrapper | p50/p99 reporter | ✓ VERIFIED | Shared implementation; wrapper keeps the plan path; three call sites |
| `internal/app/internal/testutil/rng.go` | Working replay hint | ✓ VERIFIED | Package-scoped command; shared with replay executor |
| `.github/workflows/scheduled.yml` | Six-job nightly set | ✓ VERIFIED | cron + dispatch; timeouts; fuzztime; both stand jobs; no skip flags |
| `.github/workflows/ci.yml` | Named non-skippable stand step | ✓ VERIFIED | Exactly one `go test -race ./internal/stand/`; 4-cell matrix unchanged |
| `Makefile` | red-run / bench-smoke / race-soak | ✓ VERIFIED | Targets + `COUNT ?= 20` + flake-class dispositions |
| `cmd/sup-kv/scenario_test.go` | Deleted (moved) | ✓ DELETED | Intentional; zero `TestExec_` remains; single-process tests untouched |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `scripts/red-runner` | `internal/grpcapi/red_gate_test.go`, `stand_red_test.go`, `red_value_limit_test.go` | `go test -p 1 -race -json` events matched to the TSV | ✓ WIRED | Fresh run parsed 7 rows; skips/absences are errors; packages serialized for wall-clock integrity |
| `scripts/red-runner` | `internal/app/repl/core.go` | temp-copy sed + `go test -overlay` | ✓ WIRED | Two mutants killed; control passed; `git status` clean |
| `internal/grpcapi/red_gate_test.go` | `internal/grpcapi/gate.go` | gate behavior under partial vs full listing | ✓ WIRED | Fails on v1.0 with `want Unavailable, got OK` |
| `internal/stand/stand.go` | `cmd/sup-kv/main.go` | build-once real binary with node flags | ✓ WIRED | Real process lifecycle; SIGKILL −1 / SIGTERM 0 asserted |
| `internal/stand/stand_red_test.go` | `docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md` | documented bounds derived from flags, citations | ✓ WIRED | Bounds are expressions, reviewer seconds cited only in comments |
| `internal/app/repl/sim_chaos_test.go` | `internal/app/repl/testdata/chaos-seeds.txt` | corpus iteration + drift-checked generator | ✓ WIRED | Byte-exact regeneration; fresh seeds parameterized |
| `internal/app/internal/simnet/simnet_ext_test.go` | `internal/udprepl/replproto` | test-layer encoder injection into `Policy.SizeOf` | ✓ WIRED | Generated proto messages + real `proto.Marshal`; `simnet.go` imports no replproto |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `Cluster.Delete` → `store.Delete` → `NotifyLocalWrite` | ✓ WIRED | `ErrNotFound` propagated; tombstone in write log |
| `internal/persistfile/fsops.go` | `internal/persistfile/dumpfile.go`, `snapshot.go` | `saveAtomically` five-call sequence | ✓ WIRED | Order test proves the production Save path |
| `internal/persistfile/open_seed_test.go` | `internal/app/persist/engine.go` | `persist.Open` + real `FileStore`/codec | ✓ WIRED | Five rows assert through the returned engine |
| `internal/udprepl/decode_negative_test.go` | `internal/udprepl/repl.go` | decoder verdict machinery + `Adapter.Discards()` | ✓ WIRED | Sink-socket isolation makes counter deltas exact |
| `internal/stand/stand_delete_test.go` | `internal/stand/stand.go` | harness dump/Get/Delete seams | ✓ WIRED | NotFound, tombstone line, single-line tombstone contract |
| `internal/grpcapi/watch_senderr_test.go` | `internal/grpcapi/watch.go` | fake stream Send failures + real transport break | ✓ WIRED | Handler returns the transport error; hub stays healthy |
| `internal/stand/stand_metrics_test.go` | `internal/metrics/metrics.go` | live HTTP `/metrics` rendered samples | ✓ WIRED | Exact sample-line matching; no internal counter reads |
| `internal/app/repl/decision_coverage_test.go` | `docs/adr/0001…0011` | index + word-bounded citation regex | ✓ WIRED | All 11 ADRs exist and every decision token found in repo tests |
| `internal/app/storage/prop_small_space_test.go` | `internal/app/internal/testutil/rng.go` | `testutil.ReplayCommand` executed | ✓ WIRED | Same format helper prints and executes |
| `.github/workflows/scheduled.yml` | `scripts/red-runner`, `Makefile`, `internal/stand/*`, corpus | job commands | ✓ WIRED | Six jobs invoke the single-definition targets/scripts |
| `.github/workflows/ci.yml` | `internal/stand/` | named `go test -race ./internal/stand/` | ✓ WIRED | Step appears exactly once; matrix unchanged |
| `cmd/sup-kv/examples_test.go` | README/EXAMPLES blocks | `docBlockBudget` clamped in `t.Deadline()` | ✓ WIRED | Class-3 flake fixed test-side |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `scripts/red-runner` | parsed `Action`/`Test`/`Package` | live `go test -json` of the exact spawned command | Yes — 7 fail events + mutant verdicts | ✓ FLOWING |
| `stand_metrics_test.go` | metric samples | live node `/metrics` over HTTP | Yes — exact constructed values | ✓ FLOWING |
| `sim_chaos_test.go` | seeds / model / stores | committed corpus + fresh wall-clock seeds + production store | Yes — convergence to the pointwise-supremum model | ✓ FLOWING |
| `stand_delete_test.go` | tombstone lines | real node dump files on disk | Yes — `"deleted":true` record lines | ✓ FLOWING |
| `bench_test.go` (persist) | per-op durations, bytes | real `persist.Encode` + `persistfile.FileStore.Save` | Yes — 129.3 ms p50/p99, 5 147 550 dump bytes | ✓ FLOWING |
| `open_seed_test.go` | loaded records / clock versions | real codec decode of crafted dump files | Yes — full 16-byte version comparisons | ✓ FLOWING |
| `fuzz_decode_test.go` / `fuzz_dump_test.go` | datagram/dump bytes | committed `testdata/fuzz` corpora + fuzzer | Yes — production decoders | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Green suite without the red gate | `make test` | exit 0; 12 packages ok; no FAIL; stand 83.2% / persistfile 89.7% / repl 97.8% coverage | ✓ PASS |
| Strict lint gate | `make lint` | exit 0 (config verify + `run ./...`) | ✓ PASS |
| Red tests skip without env | `go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$'`; `./internal/stand/ -run '^TestRed_'` | all SKIP, PASS | ✓ PASS |
| R-03 gate red | `SUPKV_RED_TESTS=1 go test -race ... -run '^TestRed_R03_gateClosedUntilFullList$'` | FAIL: `want Unavailable, got OK` | ✓ PASS (red as required) |
| R-03 stand red | `SUPKV_RED_TESTS=1 go test -race ./internal/stand/ -run '^TestRed_R03_f14ReplacedNode$'` | FAIL: `want Unavailable, got NotFound` | ✓ PASS (red as required) |
| P-01 process red | `SUPKV_RED_TESTS=1 ... -run '^TestRed_P01_valueLimit$'` | FAIL both sizes after restart | ✓ PASS (red as required) |
| P-01 engine red | `SUPKV_RED_TESTS=1 ... ./internal/persistfile/ -run '^TestRed_P01_valueLimitEngine$'` | FAIL: `Open discarded 1 record(s)` | ✓ PASS (red as required) |
| Full expected-red gate + mutants | `bash scripts/red-runner` | exit 0; 7/7 rows red; 4 controls pass; 4 mutant kills; evidence written | ✓ PASS |
| Benchmark smoke | `make bench-smoke` | exit 0; 3 benchmark lines with p50-ms/p99-ms | ✓ PASS |
| Race soak (local single pass) | `make race-soak COUNT=1` | exit 0; six packages ok | ✓ PASS |
| Nightly documented-scale F9 | `SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestStand_emptyDiskRecovery$'` | ok 143.835 s | ✓ PASS |
| Chaos corpus + fresh seeds | `SUPKV_SIM_FRESH_SEEDS=10 go test -race ./internal/app/repl/ -run '^TestChaos_convergence$'` | ok 1.552 s | ✓ PASS |
| Simulator extension suite | `go test -race ./internal/app/internal/simnet/ -run '^TestSim_'` | ok 1.018 s | ✓ PASS |
| Decoder table + deletion codec + fuzz seeds | `go test -race ./internal/udprepl/ -run '^TestDecode_negativeTable$\|^TestWire_deleteRoundTrip$\|^FuzzDecode$'` | ok 1.082 s | ✓ PASS |
| Persistence seam + Open seeding | `go test -race ./internal/persistfile/ -run '^TestFsops_\|^TestOpen_clockSeeded'` | ok 1.215 s | ✓ PASS |
| Watch branches | `go test -race ./internal/grpcapi/ -run '^TestWatch'` | ok 2.204 s | ✓ PASS |
| Metrics/parameter effects | `go test -race ./internal/stand/ -run '^TestStand_metricsByEffect$\|^TestConfig_parametersByEffect$'` | ok 6.961 s | ✓ PASS |
| TEST-09 scenarios | `go test -race ./internal/app/repl/ -run '^TestScenario_(combinedFailures\|longPartition)$'` | ok 1.073 s | ✓ PASS |
| Decision coverage | `go test ./internal/app/repl/ -run '^TestDecisions_referencedByTests$'` (+ corpus meta-tests) | ok 4.516 s | ✓ PASS |
| Property suite + replay executor | `go test -race ./internal/app/storage/ -run '^TestProp_'` | ok 1.604 s | ✓ PASS |
| Darwin compile | `GOOS=darwin go vet ./internal/stand/` | exit 0 | ✓ PASS |

### Probe Execution

No probe scripts are declared by this phase or present in the repository (`find scripts -path '*/tests/probe-*.sh'` → none; the PLANs declare no `probe-*.sh`). This phase is a test-harness/CI phase, so the probe-execution step is not applicable; the equivalent behavioral evidence is the spot-check table above.

**Status:** SKIPPED (no probes exist)

### Test Quality Audit

| Test File / Group | Linked Req | Active | Skipped | Circular | Strongest Assertion | Verdict |
|-------------------|-----------|--------|---------|----------|---------------------|---------|
| `internal/grpcapi/red_gate_test.go` | TEST-04 | 1 (under red gate) | 1 env gate (by design) | No | Behavioral (gRPC codes + health) | ✓ |
| `internal/stand/stand_red_test.go` | TEST-04 | 5 (under red gate) | 5 env gates (by design) | No | Behavioral (real sockets, wall-clock bounds, metrics) | ✓ |
| `internal/persistfile/red_value_limit_test.go` | TEST-04 | 1 (under red gate) | 1 env gate (by design) | No | Behavioral (restart survival, real composition) | ✓ |
| `internal/stand/stand_test.go`, `stand_delete_test.go`, `stand_metrics_test.go` | TEST-02/03/05/06/12 | All | 0 | No | Behavioral (process lifecycle, dumps, exact metrics) | ✓ |
| `internal/app/internal/simnet/simnet_ext_test.go` | TEST-02/06 | 6 | 0 | No | Behavioral (state, counters, digests) | ✓ |
| `internal/app/repl/sim_chaos_test.go`, `sim_scenario_test.go` | TEST-01/09 | All | 0 | No | Behavioral (model equality, bounds, fault teeth) | ✓ |
| `internal/udprepl/decode_negative_test.go`, `fuzz_decode_test.go` | TEST-02/08 | All | 0 | No | Value/Behavioral (verdicts, counters, invariants) | ✓ |
| `internal/app/persist/fuzz_dump_test.go` | TEST-08 | 1 fuzz target | 0 | No | Behavioral (codec contract, checksums, echo) | ✓ |
| `internal/persistfile/fsops_test.go`, `open_seed_test.go` | TEST-03/07 | All | 0 | No | Behavioral (order, failure matrix, clock dominance) | ✓ |
| `internal/app/storage/prop_small_space_test.go` | TEST-15 | 2 | 0 | No | Behavioral (CRDT laws, executed replay) | ✓ |

**Disabled tests on requirements:** 0 blockers — the only skips are the seven intentional `SUPKV_RED_TESTS` gates, and the red-runner executes each of them in its red run (fail events observed). No requirement is proven only by a skipped test.
**Circular patterns detected:** 0 — expected values are the documented acceptance expressions, exact constructed events, or production-encoder fixtures; the chaos corpus generator proves format reproducibility, not correctness.
**Insufficient assertions:** 0 — requirement-linked tests assert values or multi-step behavior (codes, versions, counters, bounds, dumps), not existence/type only.
**Coverage quantities:** met — 8 corrupted-datagram rows, 12 + 8 fuzz corpus entries, 6 simulator extension tests, 5 clock-seeding rows, 7 expectation rows, 11 decision citations.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No `TBD`/`FIXME`/`XXX` in any phase-modified file | — | None |
| — | — | No placeholder/coming-soon/not-implemented text | — | None |
| — | — | No empty return stubs or console-only implementations | — | None |
| — | — | No hardcoded empty data at render/assertion sites | — | None |

The only `t.Skip` lines are the intentional red-gate skips listed above and one pre-existing root-permission skip in `internal/persistfile/dumpfile_test.go` (not linked to a phase requirement). No debt-marker blocker exists.

### Code Review (Advisory, Recorded)

`02-REVIEW.md` reports **0 critical / 6 warning / 8 info**; `02-REVIEW-DISPOSITION.md` records all 14 as `open` (not silently accepted). None is a phase-goal blocker, and the substantive acceptance was independently re-checked:

| Finding | Substance | Why advisory for this pass |
|---------|-----------|----------------------------|
| WR-01 | red-runner does not classify package-level `fail` events with no test name | The expected-red rows ARE asserted from per-test events; the fresh full run produced 7/7 meaningful failures with 0 errors, and a package crash would still be visible in the recorded raw JSON. Hardening candidate for the next phase. |
| WR-02 | `Cluster.Write` swallows `store.Set`'s error (pre-existing harness code) | Delete was implemented with the error path the plan requires; Write's pre-existing behavior is outside this phase's must-haves. |
| WR-03 | TEST-13 meta-test walks the module cache and could pass vacuously | Independently verified: every one of 14.1–14.11 is cited in a repo test file outside `.gomodcache` (1–2 files each). |
| WR-04 | Dispatch `seeds` validation echoes the untrusted value | The regex is anchored and the value is used only as a test count; no code execution. |
| WR-05 | Port-window mitigation is per-process; `freeTCPPort` uses the ephemeral pool | No flake observed in the fresh red run or the nightly stand run; documented as a Phase 3 flake-work target. |
| WR-06 | `race-soak` excludes `internal/stand` | TEST-11's command is defined and green at COUNT=1; the stand scenarios run under `-race` in CI and nightly; the ×20 scope refinement belongs to the scheduled flake work. |
| IN-01…IN-08 | bench smoke does not assert percentiles; scenario-ID collision; darwin tag naming; duplicated small-space constants; evidence-log selection; fresh-seed count logging; fuzz-corpus drift guard; hand-mirrored encoder | Informational hardening; none changes the verified acceptance. IN-08 is noted below. |

**Known limitation noted (IN-08 / TEST-06 size gate):** the simulator size-gate test mirrors the adapter's envelope assembly at the test layer while using the generated `replproto` messages and real `proto.Marshal`; it is not a call into the production encoder helper. The size refusal/accounting behavior is verified, but a production envelope-field change could drift the mirror until a parity test is added. This is a future hardening item, not a phase-goal failure.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|-------------|--------|----------|
| TEST-01 | 02-06, 02-13 | Chaos stable + mutation-sensitive | ✓ SATISFIED | Drift-checked 16-seed corpus + fresh seeds; poll-to-model semantics; both mutants kill; control passes; 10k dispatch scheduled |
| TEST-02 | 02-03, 02-07, 02-15 | Deletion replication at every level | ✓ SATISFIED | Codec round trip, simulator `Cluster.Delete` + chaos, two-process NotFound + dump tombstone, node-down learns after start |
| TEST-03 | 02-02, 02-04 | Confirmed write is on disk, pinned by tests | ✓ SATISFIED | fsops five-step order + failure matrix + blocking Save + `TestStand_killNineRestart` process case |
| TEST-04 | 02-01, 02-05, 02-12 | R-03 acceptance tests fail on current code | ✓ SATISFIED | All four Criticals (R-01, R-02 both parts, R-03 both levels, P-01 both levels) reproduced red in a fresh full run |
| TEST-05 | 02-02, 02-14 | Real-socket tests never silently skipped | ✓ SATISFIED | Linux probe/build failures are fatal; zero `Skipf`; named darwin report; CI runs `-race` |
| TEST-06 | 02-03, 02-11, 02-15 | Simulator blind spots closed + nightly 10k | ✓ SATISFIED | Budget, size gate, announce delay, restart-with-state, deletions; nightly 10k×1KiB wired and locally green (143.8 s) |
| TEST-07 | 02-16 | `persist.Open` clock seeding on real composition | ✓ SATISFIED | Five-row real FileStore + codec + Open test green |
| TEST-08 | 02-07, 02-11, 02-17 | Decoder negative tests + fuzzing | ✓ SATISFIED | 8-row table + 12/8 committed corpora + scheduled `-fuzztime` jobs |
| TEST-09 | 02-13 | Bounds under combined failures and long partitions | ✓ SATISFIED | `TestScenario_combinedFailures` / `longPartition` with lower/upper bounds, teeth and no-fresh-loss qualifier |
| TEST-10 | 02-10 | Benchmarks measure what they claim | ✓ SATISFIED | p50/p99 on all three real-path benchmarks; no self-pacing; `make bench-smoke` green |
| TEST-11 | 02-10, 02-11 | Known flake classes eliminated | ✓ SATISFIED | `race-soak` command + class dispositions; local COUNT=1 green; ×3-green scheduled (per phase contract) |
| TEST-12 | 02-08 | Metrics/configuration tested by observed effect | ✓ SATISFIED | Exact conflict/bad-CRC/retransmit values; five parameter-effect subtests + map for the rest |
| TEST-13 | 02-09 | Every section-14 decision has a named test | ✓ SATISFIED | All 11 decisions cited in repo tests; meta-test green |
| TEST-14 | 02-08 | Watch cut and backpressure branches covered | ✓ SATISFIED | Send-error, snapshot-then-live cut, slow-subscriber shed, broken-transport isolation |
| TEST-15 | 02-09 | Property tests shrink and explore equal-`l` versions | ✓ SATISFIED | Small 3×3×3 space in half the draws; executable package-scoped replay command |
| TEST-16 | 02-11 | Scheduled CI workflow exists | ✓ SATISFIED | `scheduled.yml` with 1000 seeds, `-shuffle=on -count=3`, `-bench . -benchtime=1x`, cron + dispatch |

**Requirement accounting:** all 16 TEST requirement IDs appear in PLAN frontmatter (02-01…02-17) and all 16 map to Phase 2 in REQUIREMENTS.md. No orphaned TEST requirement remains unclaimed.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts — `9/9`, `not_honored: []` (decision-coverage gate, non-blocking).

### Human Verification Required

N/A — Infrastructure/test-harness phase with no user-facing elements. All acceptance criteria are verifiable programmatically, and every programmatic check above was executed on the live tree. The remaining post-merge duties (nightly cron execution, `seeds=10000` dispatch, macOS cell execution) are the scheduled/CI evidence vehicles the roadmap explicitly assigns to the workflow, not human UAT items.

### Gaps Summary

No gaps. All 37 derived must-have truths are VERIFIED, all required artifacts exist and are substantive and wired, all key links resolve, all 16 TEST requirements are satisfied, `make test` and `make lint` are green on the untouched tree, the four Criticals reproduce RED against v1.0 through the machine gate, and the red runs are recorded as phase evidence. The intentionally-red tests are the phase's deliverable, not a gap: the expectation table and `scripts/red-runner` assert them, and the committed evidence records them.

---

_Verified: 2026-11-21T15:00:25Z_
_Verifier: ZCode (gsd-verifier)_
