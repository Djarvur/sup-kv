---
phase: 02-stand-and-simulator-as-tests
verified: 2026-11-23T19:56:26Z
status: passed
score: 38/38 must-haves verified
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
  - "docs/adr/0010-darwin-development-only.md"
  - "internal/app/internal/simnet/simnet.go"
  - "internal/app/internal/simnet/simnet_ext_test.go"
  - "internal/app/internal/testutil/bench.go"
  - "internal/app/internal/testutil/rng.go"
  - "internal/app/persist/bench_test.go"
  - "internal/app/persist/codec.go"
  - "internal/app/persist/codec_test.go"
  - "internal/app/persist/engine.go"
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
  - "internal/app/repl/core.go"
  - "internal/app/repl/decision_coverage_test.go"
  - "internal/app/repl/sim_bound_test.go"
  - "internal/app/repl/sim_chaos_test.go"
  - "internal/app/repl/sim_scenario_test.go"
  - "internal/app/repl/testdata/chaos-seeds.txt"
  - "internal/app/storage/prop_small_space_test.go"
  - "internal/app/storage/store.go"
  - "internal/grpcapi/bench_test.go"
  - "internal/grpcapi/gate.go"
  - "internal/grpcapi/gate_test.go"
  - "internal/grpcapi/red_gate_test.go"
  - "internal/grpcapi/server_test.go"
  - "internal/grpcapi/watch.go"
  - "internal/grpcapi/watch_senderr_test.go"
  - "internal/metrics/metrics.go"
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
  - "internal/udprepl/codec.go"
  - "internal/udprepl/decode_negative_test.go"
  - "internal/udprepl/fuzz_decode_test.go"
  - "internal/udprepl/mirror.go"
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
covered_digest: "v4:sha256:4bf1ece2ce96e4d15e0ec8f3ecf4e3c552d5b74c13ab73d7a554ac9590cdc3c5"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 37/38
  gaps_closed:
    - "TEST-11: the count-20 race-soak acceptance command (`go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/`) is green three times in a row"
  gaps_remaining: []
  regressions: []
  note: "Closed after the 2026-11-23 report. 08-06 replaced every ephemeral picker with the shared claim-while-held allocator over disjoint private bands; follow-up 7aeec35 claims a candidate before its bind probe. 08-09 follow-up 9cdc0f1 measures startup bounds from process start and retries a transient loopback-broadcast bind on a fresh port (the probeBroadcastForm policy). The acceptance command passed three consecutive runs at 9cdc0f1 (2026-11-24 21:09-21:23 UTC) and again at the phase-8 gate tree; make lint, make test and the phase-8 gate are green."
---

# Phase 02: Stand and Simulator as Tests — Verification Report

**Phase Goal:** The review's verdict is reproducible by machine — every claimed miss (recovery, burst) and every Critical finding is pinned by an automated test on real sockets or a hardened simulator, RED where the code is wrong. The tests are expected to fail against v1.0 code; that is the point — Critical findings get failing tests before fixes, and Phases 3-5 turn them green.

**Verified:** 2026-11-23T19:56:26Z
**Status:** passed
**Re-verification:** Yes — the TEST-11 gap recorded on 2026-11-23 is closed. The shared claim-while-held allocator (08-06, follow-up `7aeec35`), the process-start measurement and the fresh-port probe retry (08-09 follow-up `9cdc0f1`) make the count-20 acceptance command green three runs in a row (2026-11-24 21:09-21:23 UTC), repeated at the phase-8 gate tree; `make lint`, `make test` and the phase-8 gate are green.
**Verification basis:** the 17 PLAN/SUMMARY pairs, ROADMAP phase 2 success criteria, REQUIREMENTS.md TEST-01…TEST-16, the committed red evidence, and fresh executions of the red gate, the full race suite, the documented race soak, the count-20 acceptance command, lint, benchmark smoke, the nightly F9/R-01 documented-scale runs, chaos with fresh seeds, and the artifact/decision gates.

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A real-socket stand exists that spawns the real `cmd/sup-kv` binary over loopback UDP + gRPC on Linux, and CI runs it with `-race`. | ✓ VERIFIED | `internal/stand/stand.go` (`//go:build linux`, build-once builder of `github.com/Djarvur/sup-kv/cmd/sup-kv`, private port window 28100–32678 + claim registry, fatal probe, SIGKILL/SIGTERM lifecycle); `.github/workflows/ci.yml` runs `go test -race ./internal/stand/` then `go test -race ./...` in every non-cross cell; `make test` is `go test -race -cover ./...`; stand coverage 83.6%. |
| 2 | F9: an empty-disk node recovers 10 000 × 1 KiB keys within the derived bound at the documented nightly scale. | ✓ VERIFIED | `TestStand_emptyDiskRecovery` selects `emptyDiskKeysNightly=10000`, `emptyDiskValueBytes=1024`, documented default flags under `SUPKV_STAND_NIGHTLY=1`; fresh run `SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m -v ./internal/stand/ -run '^TestStand_emptyDiskRecovery$'` → `nightly scale selected`, `empty-disk recovery: 10000 keys in 23.94s (paced bound 20s, margin 1m5s)`, PASS 51.73s. |
| 3 | F9b/R-01: the pure-resync recovery scenario is pinned with the documented `2×sync_interval + N/sync_response_rate + margin` bound; it was red against v1.0 and is green now. | ✓ VERIFIED | `TestRed_R01_pacedSync20k` seeds then restarts A so only anti-entropy remains. Committed evidence `20261121T145156Z` records `red / fail,run`; fresh documented-scale run `SUPKV_STAND_NIGHTLY=1 SUPKV_RED_TESTS=1 ... -run '^TestRed_R01_pacedSync20k$'` → PASS 92.30s at 20 000 keys (progress 9855→19261→20000). |
| 4 | F14/R-03: the replaced node stays closed until the full multi-datagram listing, never overwrites the peer's user value, answers `AlreadyExists` after the listing, and takes a greater version; red against v1.0, green now. | ✓ VERIFIED | `TestRed_R03_f14ReplacedNode` asserts `Unavailable` reads/writes before the listing, A's value unchanged, `NOT_SERVING`, `AlreadyExists` after, local version > listing max. Committed evidence records red (`want code Unavailable, got NotFound`); fresh red-gate run: pass. Gate-level twin `TestRed_R03_gateClosedUntilFullList` likewise red→green. |
| 5 | The 1000 writes/s × 15 s burst over 10 000 distinct keys pins ≥990 writes/s and an unacked set empty 5 s after the send window; red against v1.0, green now. | ✓ VERIFIED | `TestRed_R02_burst` drives the open-loop schedule (1000/s × 15 s, 10 000 keys, 1024 workers/8 conns) and asserts `achieved >= 990` plus the separate ≤5 s drain. Committed evidence records red (`25.1 writes/s`, 3232/15000 OK); fresh red-gate run: pass. Also `TestRed_R02_unackedDrain` (N/rate + 10 s, ≤3N retransmits) red→green. |
| 6 | F2c is pinned as an automated stand test: the 70 000-char key never replicates, leaves no record on either node or in the dump, and the boundary-sized record does replicate. | ✓ VERIFIED | `TestStand_oversizeKeyF2c` now pins the corrected verdicts (Phase 3 XPORT-04 rewrote it in `03-09`): over-ceiling key → `InvalidArgument` from the key bound, over-envelope key → `ResourceExhausted`, no live key on A, absent from B for the full window, absent from B's dump; the 1 024-byte-key / 61 440-byte-value boundary record replicates and survives B's dump. The v1.0 acceptance-with-sendto-errors behavior is recorded in the review evidence; the F2c invariant (never replicates) is asserted more strongly now. Green in `make test` and the red-runner's stand package run. |
| 7 | No socket test silently skips: a Linux capability-probe miss is a fatal, and the darwin cell reports "darwin replication is untested" through a named green test. | ✓ VERIFIED | `grep -rn Skipf internal/stand/` → zero. `standBinary` fatals on a build/probe miss; `newStandPair`/`probeLoopbackSend` treat a probe miss as a fatal on Linux. `stand_nondarwin_test.go` (`!linux`) passes and logs `darwin replication is untested: …`; it also asserts the build-tag boundary. `GOOS=darwin go vet ./internal/stand/` → exit 0. The only stand skips are the five intentional `SUPKV_RED_TESTS` gates plus one loud root-permission skip in the (later-phase) `persist_test.go`, which is not a socket test. |
| 8 | R-01 fails against v1.0 code with the recorded evidence. | ✓ VERIFIED | Committed `red-evidence/20261121T145156Z/summary.txt`: `TestRed_R01_pacedSync20k red fail,run OK` under `go test -race`. The runner's contract is fail-event-required (a skip or absence is an error). |
| 9 | R-02 unacked-set cost and burst drain fail against v1.0 code. | ✓ VERIFIED | Same committed summary: `TestRed_R02_unackedDrain red fail,run` and `TestRed_R02_burst red fail,run`. The burst failure text in the phase record: 25.1 writes/s, 3232/15000 OK, 11768 DeadlineExceeded. |
| 10 | R-03 closed-until-full-list fails against v1.0 at gate level and on the stand. | ✓ VERIFIED | Committed summary: `TestRed_R03_gateClosedUntilFullList red fail,run` (`want code Unavailable, got OK`) and `TestRed_R03_f14ReplacedNode red fail,run` (`want code Unavailable, got NotFound`). |
| 11 | P-01 single-value-limit fails against v1.0 at process and engine level. | ✓ VERIFIED | Committed summary: `TestRed_P01_valueLimit red fail,run` (both 63 000 and exactly 64 000 bytes) and `TestRed_P01_valueLimitEngine red fail,run` (`Open discarded 1 record(s): key not found`). |
| 12 | Every red test skips without `SUPKV_RED_TESTS=1`, so `make test` stays green. | ✓ VERIFIED | Each of the seven `TestRed_*` bodies begins with the env gate; fresh `go test -race -cover ./...` → exit 0 (red tests skip), and the red-runner opens the gate itself. |
| 13 | `scripts/red-runner` asserts the expectation table from `go test -json` events, exits 0 on the full run, and records evidence. | ✓ VERIFIED | Fresh `bash scripts/red-runner` → exit 0: `7 expectation row(s) checked, 0 error(s)`; all 7 rows `OK` (green); mutation stage `control` 4/4 pass, both mutants killed; evidence written to `red-evidence/20261123T193404Z/{go-test.json,summary.txt}`. A skip/absence for a table row remains an error; an unexpected failing test is an error. |
| 14 | `TestChaos_convergence` is mutation-sensitive: both named mutants kill it on fixed corpus seeds while the control passes, and the production tree stays clean. | ✓ VERIFIED | Fresh runner mutant stage: 4/4 controls pass; `sync-without-tombstones` killed seed 848612892191557850; `periodic-sync-disabled` killed seed 848612892191557850; the stage asserts compile, sed-no-op, timeout, corpus membership and a clean `store.go`/`core.go` via `git status`. |
| 15 | Simulator `Cluster.Delete` replicates a tombstone and propagates `ErrNotFound` without burning a version. | ✓ VERIFIED | `simnet.go` `Delete` (line 816) calls `store.Delete` → `NotifyLocalWrite` → route/drive/write log; `TestSim_Delete_replicatesTombstone` green: tombstone dominates the live write, peer receives it, missing/already-tombstoned keys return `ErrNotFound`, the write log does not grow and the clock counter advances by exactly one across interleaved failed deletes. |
| 16 | `Cluster.ReplaceNodeWithState` restarts a node with its state preserved. | ✓ VERIFIED | `simnet.go` builds the node, sorted-key `store.Merge` seed, then the ReplaceNode swap/Reregister/timer supersession; `TestSim_ReplaceNodeWithState_preservesState` green with a plain-`ReplaceNode` empty-boot control. |
| 17 | The simulator enforces a per-node incoming-datagram budget, refusing and counting past it. | ✓ VERIFIED | `Network.SetRecvBudget` + `FaultStats.RecvBudgetRefused` in `scheduleDelivery`; `TestSim_recvBudget_refusesAndCounts` green: 1 delivered, 2 refused/counted, sender's unacked count 2. |
| 18 | The simulator refuses oversize datagrams measured by the encoder and counts the refusals. | ✓ VERIFIED | `Policy.SizeOf`/`MaxDatagramBytes` + `FaultStats.OversizeRefused`; `TestSim_sizeRefusal_byRealEncoder` green with generated `replproto` messages through `proto.Marshal` and a measured midpoint limit; `simnet.go` still imports stdlib + `internal/app` + `internal/port` only. |
| 19 | Record announcements can be delayed by a virtual dump latency. | ✓ VERIFIED | `Policy.AnnounceDelay` consulted for `port.DataReplication`; `TestSim_announceDelay_virtualDumpLatency` green: absent at arrival−1 ms, present exactly at the delayed instant, trace line at the exact UnixMilli, Ack unaffected. |
| 20 | Same-seed replay is byte-identical with all simulator extensions active. | ✓ VERIFIED | `TestSim_determinism_withExtensions` green: identical `TraceDigest`, `FaultStats` and per-node stores, with activation teeth (`OversizeRefused>0`, `RecvBudgetRefused>0`). |
| 21 | Chaos runs a fixed drift-checked corpus plus fresh seeds each run and converges; the 10 000-seed sweep is dispatchable. | ✓ VERIFIED | `testdata/chaos-seeds.txt` (16 seeds) + `TestChaos_seedCorpusGenerator` regenerates twice and byte-compares (drift fails; write gated); `TestChaos_convergence` iterates corpus + `SUPKV_SIM_FRESH_SEEDS` fresh runs; fresh `SUPKV_SIM_FRESH_SEEDS=10` run → ok 1.634s; `TestChaos_corpusEmptyStateProbe` green. The one-off 10 000-seed execution is the documented `gh workflow run scheduled.yml -f seeds=10000` duty (not run here; 1–3 h budget). |
| 22 | Combined failures and partitions longer than `peer_timeout` converge not before the first post-heal sync and not later than the documented bound, with no fresh loss. | ✓ VERIFIED | `TestScenario_combinedFailures` (3 nodes, loss + duplication + timed partition + restart; divergence teeth at heal; bound `boundPartitionHeal + margin`; no-fresh-loss) and `TestScenario_longPartition` (`peerTableSurvives`, `peerEvicted` past `peer_timeout`; divergence teeth; lower/upper bounds) — both green in `make test`/race-soak. |
| 23 | The fsops seam records exactly write → fsync → rotate → rename → dir-fsync, and a failure at each step surfaces with hygiene intact. | ✓ VERIFIED | `internal/persistfile/fsops.go` five-method seam; `saveAtomically` drives the exact order; `TestFsops_saveOperationOrder` asserts the ordered slice and argument paths; rotation chain, five-row failure matrix, fill-failure hygiene and empty-dump round trip green. |
| 24 | A blocking dump Save prevents `Set` from returning. | ✓ VERIFIED | `TestEngine_blockingSaveHoldsSet` (synctest bubble) green: Set has not returned while Save stands; release → Set returns nil with exactly one completed save. |
| 25 | `persist.Open` seeds the clock from the dump watermark on the real composition path. | ✓ VERIFIED | `TestOpen_clockSeededFromDumpWatermark` composes real `FileStore` + codec + `persist.Open`; five rows green (watermark ahead dominates, boundary-equal still greater, empty start, equal-max-l counter pair, restart idempotence). |
| 26 | Corrupted datagrams have a verdict/counter table, and deletion survives the codecs. | ✓ VERIFIED | `TestDecode_negativeTable` now carries 16 rows (bad CRC, truncated envelope/body, proto-version variants, bad run_id, oversize key, key-too-long, empty key, tombstone-with-data, bad checksum, far-future version and sync entry), each asserting its decode verdict and, when fed, the exact `DiscardStats` delta; `TestWire_deleteRoundTrip` covers ASCII + UTF-8 tombstone wire and dump codec round trips (including tombstone-with-data fidelity). |
| 27 | `FuzzDecode` and `FuzzDumpDecode` exist with committed in-repo corpora and invariant-only assertions. | ✓ VERIFIED | Two fuzz targets, one per package; `internal/udprepl/testdata/fuzz/FuzzDecode/` 12 entries and `internal/app/persist/testdata/fuzz/FuzzDumpDecode/` 8 entries, each in `go test fuzz v1` format; plain `go test` seed runs green; deep runs are the scheduled `-fuzztime=10m` jobs. |
| 28 | Deletion replication is proven at codec, simulator, two-process and dump levels, including a node that was down during the delete. | ✓ VERIFIED | Codec `TestWire_deleteRoundTrip`; simulator `TestSim_Delete_replicatesTombstone` + chaos deletions; two-process `TestStand_deleteReplication` (Delete→NotFound on B, `"deleted":true` in B's dump, delete-twice single tombstone) and `TestStand_deleteWhileNodeDown` (learns after start) — all green. |
| 29 | Watch cut/backpressure and Send-error branches are covered. | ✓ VERIFIED | `TestWatch_sendErrorDisconnectsSubscriber` covers snapshot and live-relay Send failures with a fake stream (error identity, call counts), real broken-transport isolation/liveness/fresh-subscribe, and re-asserts cut/shed; the existing `TestWatch_snapshotThenLive`, `TestWatch_exactCut`, `TestWatch_slowConsumer` remain green. |
| 30 | Metrics are tested by observed effect: exact `/metrics` values after the constructed conflict + retransmit + discard scenario, and one observed-effect test per configuration parameter. | ✓ VERIFIED | `TestStand_metricsByEffect` asserts `supkv_merges_conflicts_total == 1`, `supkv_datagrams_discarded_total{class="bad_crc"} == 1`, retransmits `> baseline` and the final `>= 1` lower bound from live HTTP `/metrics`; `TestConfig_parametersByEffect` has five named parameter subtests (startup gate, hello interval, sync response rate, value max, peer timeout) and the remaining flags map to existing observing tests in `02-08-SUMMARY.md`. |
| 31 | Every owner decision 14.1–14.11 is referenced by name in at least one test, and a meta-test fails naming any missing one. | ✓ VERIFIED | `decisionIndex` enumerates all 11 decisions/ADRs; `TestDecisions_referencedByTests` walks the module's own test sources (skipping `docs/` and `testdata/`; the meta-test excludes itself) with word-bounded citation regexes and asserts each ADR file exists. Green. |
| 32 | Property tests draw half their versions from a small `(l, c, node)` space and the printed replay command works. | ✓ VERIFIED | `smallSpaceVersion` uses a 3×3×3 space and `mixedVersion` draws it half the time; storage merge laws + collision/content-law properties use it; `TestProp_replayCommandExecutes` executes the exact printed package-scoped command and asserts the same test re-runs and passes; `TestProp_mergeSmallSpace` green. |
| 33 | A scheduled CI workflow exists with the D-09 job set, nightly cron + `workflow_dispatch`, explicit timeouts, pinned actions and least privilege. | ✓ VERIFIED | `.github/workflows/scheduled.yml`: cron `30 2 * * *`, dispatch `seeds`; jobs chaos-seeds, bench-smoke, red-run, race-flake, nightly-stand, deep-fuzz (the D-09 six) plus the additive `red-r01-x20` (03-13); 7 `timeout-minutes`, 2 `-fuzztime`, 2 `SUPKV_STAND_NIGHTLY`, `permissions: contents: read`, checkout@v7/setup-go@v7 with `go-version-file: go.mod`. |
| 34 | TEST-16's acceptance tokens are on schedule: 1000 seeds, `-shuffle=on -count=3`, `-bench . -benchtime=1x`. | ✓ VERIFIED | chaos-seeds defaults `SUPKV_SIM_FRESH_SEEDS` to 1000 (validated dispatch input) and runs `go test -race ./internal/app/repl/ -run '^TestChaos' -shuffle=on -count=3`; bench-smoke runs `make bench-smoke` = `-bench . -benchtime=1x` (single definition); dispatch `seeds=10000` documented. |
| 35 | The nightly real-socket run executes the review's section-15 set at 10 000 × 1 KiB: the green F9 half in nightly-stand and the red R-01/R-02/R-03 half in red-run. | ✓ VERIFIED | One `SUPKV_STAND_NIGHTLY=1` contract selects documented scale in both jobs; nightly-stand runs `go test -race -count=1 ./internal/stand/`; red-run runs `bash scripts/red-runner` (exports `SUPKV_RED_TESTS=1`). Locally: F9 10 000 keys in 23.94 s inside the 20 s + 65 s bound, and R-01 at 20 000 keys PASS 92.30 s. R-02's documented drain scale coincides with the default, the burst is scale-independent (1000/s × 15 s × 10 000 keys), R-03's nightly listing is 4096 keys. |
| 36 | Benchmarks report p50/p99 without self-pacing on the real `persistfile`, and the smoke is CI-runnable. | ✓ VERIFIED | Fresh `make bench-smoke` → exit 0; Dump10kKeys `57.13 p50-ms / 57.13 p99-ms`, ReplicationBurst `0.008972/0.008972`, ConfirmationLatency `1.585/1.585` at `-benchtime=1x`; the measured per-op spans are taken inside the loops (the burst benchmarks pace their submission workload with 1 ms tickers — the workload shape, not a measurement throttle); Dump10kKeys uses real `persistfile.New` + `SaveStream`. |
| 37 | TEST-11's command is defined as `make race-soak` with `COUNT ?= 20` over the affected set including `internal/stand`; `COUNT=1` is locally green. | ✓ VERIFIED | `Makefile` `race-soak` runs `go test -race -count=$(COUNT)` over cmd/sup-kv, udprepl, app/repl, app/persist, persistfile, grpcapi and stand; fresh `make race-soak COUNT=1` → all 7 packages ok. (WR-06's stand exclusion is closed.) |
| 38 | TEST-11's acceptance: `go test -race -count=20` over `cmd/sup-kv`, `udprepl` and the affected packages is green three times in a row (ROADMAP success criterion 5). | ✓ VERIFIED | `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/` passed three consecutive runs (2026-11-24 21:09-21:23 UTC) and again at the phase-8 gate tree. The pick-release-then-bind class was removed by the shared claim-while-held allocator (08-06, claim-before-probe follow-up `7aeec35`) and the process-start measurement plus fresh-port probe retry (08-09 follow-up `9cdc0f1`); `make lint`/`make test` and the phase-8 gate are green. |

**Score:** 38/38 truths verified (0 present, behavior-unverified; TEST-11 re-verified green)

### Deferred Items

None that are addressed by a later milestone phase. The TEST-11 flake class is not scheduled in any later phase's goal or success criteria (phases 3–9 contain no port-window/flake work; the only later mention is the WR-06 addition that put `internal/stand` into `race-soak`). It is a real, actionable gap.

### Advisory (New Scope, Unevidenced)

None. The only failing item is backed by a reproducible command (the count-20 acceptance run), not by opinion.

### Plan-Level Must-Have Coverage

| Plan | Requirement(s) | Outcome | Status |
|------|----------------|---------|--------|
| 02-01 | TEST-04 | env-gated R-03 gate red test; red-runner + expectation table + evidence; `make red-run` | ✓ VERIFIED |
| 02-02 | TEST-05, TEST-03 | `internal/stand` harness moved wholesale; four migrated scenarios; F2c pin; `cmd/sup-kv/scenario_test.go` deleted | ✓ VERIFIED |
| 02-03 | TEST-06, TEST-02 | simnet `Delete`, `ReplaceNodeWithState`, receive budget, size gate, announce delay; import block stdlib+app+port | ✓ VERIFIED |
| 02-04 | TEST-03 | fsops seam (five methods), recording/failing fakes, blocking-Save proof | ✓ VERIFIED |
| 02-05 | TEST-04 | four stand red tests with documented bounds; nightly scale selection | ✓ VERIFIED |
| 02-06 | TEST-01 | drift-checked corpus, corpus+fresh iteration, poll-to-model stabilization, deletions | ✓ VERIFIED |
| 02-07 | TEST-02, TEST-08 | corrupted-datagram table (now 16 rows), deletion codec round trip, two-process deletion levels | ✓ VERIFIED |
| 02-08 | TEST-14, TEST-12 | Watch Send-error coverage; metrics-by-effect exact values; per-parameter effects | ✓ VERIFIED |
| 02-09 | TEST-13, TEST-15 | 11-decision coverage meta-test + citations; small-space draws; executable replay command | ✓ VERIFIED |
| 02-10 | TEST-10, TEST-11 | p50/p99 helper wired into three benchmarks; `bench-smoke`/`race-soak`; flake audit **closed by Phase 8** (the audited port class is green) | ✓ RE-VERIFIED (TEST-11) |
| 02-11 | TEST-16, TEST-06, TEST-08, TEST-11 | nightly scale wiring; scheduled workflow; ci.yml tightening | ✓ VERIFIED |
| 02-12 | TEST-04 | P-01 process + engine red tests; 7-row expectation table; full recorded red run | ✓ VERIFIED |
| 02-13 | TEST-01, TEST-09 | combined-failure/long-partition tests; red-runner overlay mutant stage | ✓ VERIFIED |
| 02-14 | TEST-05 | loud darwin report + build-tag boundary guard | ✓ VERIFIED |
| 02-15 | TEST-06, TEST-02 | six simulator extension tests incl. same-seed replay gate | ✓ VERIFIED |
| 02-16 | TEST-07 | five-row `persist.Open` clock-seeding proof on the real composition path | ✓ VERIFIED |
| 02-17 | TEST-08 | `FuzzDecode`/`FuzzDumpDecode` + 12/8 committed corpus entries | ✓ VERIFIED |

### Phase Evidence: Red → Green

The phase's central deliverable is intact and the Phases 3–5 flips are visible end to end:

| Expectation row | Committed 2026-11-21 evidence | Current tree (fresh 2026-11-23 run) | Flip commit |
|-----------------|-------------------------------|------------------------------------|-------------|
| `TestRed_R03_gateClosedUntilFullList` | red / fail,run | green / pass,run | `a44e661` (04-13) |
| `TestRed_R01_pacedSync20k` | red / fail,run | green / pass,run | `4a83919` (03-12) |
| `TestRed_R02_unackedDrain` | red / fail,run | green / pass,run | `276c0b3` (03-12) |
| `TestRed_R02_burst` | red / fail,run | green / pass,run | `4a83919` (03-12) |
| `TestRed_R03_f14ReplacedNode` | red / fail,run | green / pass,run | `a44e661` (04-13) |
| `TestRed_P01_valueLimit` | red / fail,run | green / pass,run | `7851cfc` (05-12) |
| `TestRed_P01_valueLimitEngine` | red / fail,run | green / pass,run | `7851cfc` (05-12) |

The red-gate mechanism itself is unchanged: a `red` row requires a fail event, a `green` row forbids one, a skip/absence/fail-for-green is an error. The only test edit between the red evidence and the flips is `03-12`'s burst-rate window alignment (rate over the 14.999 s send window, the separate ≤5 s drain bound unchanged), documented in the test's doc comment and the SUMMARY — it aligns the measurement with the XPORT-02(3) acceptance rather than weakening it.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/stand/stand.go` | Linux real-socket harness | ✓ VERIFIED | Build-once binary, reserved port window + registry, fatal probe, lifecycle, deadline-clamped waits, dump/log seams |
| `internal/stand/stand_test.go` | Migrated scenarios + F2c + nightly scale | ✓ VERIFIED | `TestStand_killNineRestart`, `bothNodesDown`, `sigtermFinalDump`, `emptyDiskRecovery` (nightly 10k×1KiB, fresh green), `oversizeKeyF2c` (Phase 3 rewrite) |
| `internal/stand/stand_red_test.go` | R-01/R-02/R-03/P-01 red tests | ✓ VERIFIED | Five env-gated tests, documented bounds, `standNightly` scale selection; green under the red gate |
| `internal/stand/stand_delete_test.go` | Two-process deletion levels | ✓ VERIFIED | Delete→NotFound, tombstone in dump, delete-twice, node-down node learns after start |
| `internal/stand/stand_metrics_test.go` | Metrics by effect + parameter effects | ✓ VERIFIED | Exact rendered `/metrics` values; five parameter subtests |
| `internal/stand/stand_nondarwin_test.go` | Darwin report | ✓ VERIFIED | Named passing test + statement + build-tag guard; `GOOS=darwin go vet` green |
| `scripts/red-runner` | Expected-red gate + mutants | ✓ VERIFIED | `go test -json` parser, table assertion, evidence, overlay mutant stage; fresh exit 0 |
| `scripts/red-expectations.tsv` | 7-row table, all flipped green | ✓ VERIFIED | All rows `green`; flip history is one row per fix |
| `red-evidence/*` | Recorded red and green runs | ✓ VERIFIED | 35 committed runs; `20261121T145156Z` records all seven red; fresh `20261123T193404Z` (untracked verifier run) records all seven green + mutant kills |
| `internal/app/internal/simnet/simnet.go` | Additive simulator API | ✓ VERIFIED | Delete, ReplaceNodeWithState, SetRecvBudget, SizeOf/MaxDatagramBytes, AnnounceDelay, two counters; import block stdlib+app+port |
| `internal/app/internal/simnet/simnet_ext_test.go` | Six extension tests | ✓ VERIFIED | Delete, restart, budget, size gate, announce delay, determinism replay |
| `internal/app/repl/sim_chaos_test.go` | Stabilized chaos | ✓ VERIFIED | Corpus generator/drift, corpus+fresh iteration, poll-to-model, deletions |
| `internal/app/repl/testdata/chaos-seeds.txt` | Committed corpus | ✓ VERIFIED | 16 deterministic seeds, byte-compared each run |
| `internal/app/repl/sim_scenario_test.go` | TEST-09 scenarios | ✓ VERIFIED | combined failures + long partition bounds/teeth/qualifier |
| `internal/app/repl/decision_coverage_test.go` | TEST-13 meta-test | ✓ VERIFIED | 11-row index; word-bounded citations; ADR existence check |
| `internal/persistfile/fsops.go` | Five-op durability seam | ✓ VERIFIED | write/fsync/rotate/rename/syncDir + real default impl + shared hygiene |
| `internal/persistfile/dumpfile.go` | Seam-driven Save | ✓ VERIFIED | `saveAtomically` drives the five calls; exported constructors unchanged |
| `internal/persistfile/fsops_test.go` | Order + failure matrix | ✓ VERIFIED | Exact order, rotation chain, five failure rows, empty round trip |
| `internal/app/persist/engine_blocking_test.go` | Blocking-Save proof | ✓ VERIFIED | Set does not return while Save stands |
| `internal/persistfile/open_seed_test.go` | TEST-07 composition proof | ✓ VERIFIED | Five rows on real FileStore + codec + Open |
| `internal/persistfile/red_value_limit_test.go` | P-01 engine red test | ✓ VERIFIED | Env-gated, real composition; green now |
| `internal/udprepl/decode_negative_test.go` | Negative table + deletion codec | ✓ VERIFIED | 16 rows with verdict + counter deltas; tombstone round trips |
| `internal/udprepl/fuzz_decode_test.go` + corpus | FuzzDecode + 12 seeds | ✓ VERIFIED | Invariant-only body; corpora load under plain `go test` |
| `internal/app/persist/fuzz_dump_test.go` + corpus | FuzzDumpDecode + 8 seeds | ✓ VERIFIED | Contract-only body; echo/checksum accounting |
| `internal/grpcapi/red_gate_test.go` | R-03 gate-level red test | ✓ VERIFIED | Env-gated; full PROTO-01 acceptance; green now |
| `internal/grpcapi/watch_senderr_test.go` | Watch Send-error branch | ✓ VERIFIED | Snapshot + relay Send failures, transport isolation, cut/shed |
| `internal/app/storage/prop_small_space_test.go` | TEST-15 property + replay executor | ✓ VERIFIED | 3×3×3 draws; printed command executed |
| `internal/testutil/bench.go` + wrapper | p50/p99 reporter | ✓ VERIFIED | Shared implementation; wrapper keeps the plan path; three call sites |
| `internal/app/internal/testutil/rng.go` | Working replay hint | ✓ VERIFIED | Package-scoped command; shared with replay executor |
| `.github/workflows/scheduled.yml` | Scheduled set | ✓ VERIFIED | Six D-09 jobs + additive `red-r01-x20`; cron + dispatch; timeouts; fuzztime; both stand jobs |
| `.github/workflows/ci.yml` | Named non-skippable stand step | ✓ VERIFIED | `go test -race ./internal/stand/` appears once before `go test -race ./...`; 4-cell matrix unchanged |
| `Makefile` | red-run / bench-smoke / race-soak | ✓ VERIFIED | Targets + `COUNT ?= 20`; race-soak now includes `internal/stand`; flake-class comment (correct after Phase 8; see Gaps Summary) |
| `cmd/sup-kv/scenario_test.go` | Deleted (moved) | ✓ DELETED | Intentional; zero `TestExec_` remains; single-process tests untouched |

### Key Link Verification

The CLI `verify.key-links` verb reports `Target not referenced in source` for 15 of 18 declared links because it looks for a literal file-path reference (same-package targets and package-path builds never match). Each link was therefore verified manually; all are behaviorally wired.

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `scripts/red-runner` | `internal/grpcapi/red_gate_test.go`, `stand_red_test.go`, `red_value_limit_test.go` | `go test -p 1 -race -json` events matched to the TSV | ✓ WIRED | Fresh run parsed 7 rows; skips/absences are errors; packages serialized for wall-clock integrity |
| `scripts/red-runner` | `internal/app/storage/store.go`, `internal/app/repl/core.go` | temp-copy sed + `go test -overlay` | ✓ WIRED | Both mutants killed; 4/4 controls pass; `git status` clean for both sources |
| `internal/grpcapi/red_gate_test.go` | `internal/grpcapi/gate.go` | gate behavior under partial vs full listing via `ServerOptions.SyncArrived` | ✓ WIRED | Green after the Phase 4 flip; the test asserts closed reads/writes + `NOT_SERVING` then open reads + `SERVING` |
| `internal/stand/stand.go` | `cmd/sup-kv/main.go` | build-once `go build` of `github.com/Djarvur/sup-kv/cmd/sup-kv` | ✓ WIRED | `standBuildTarget` (line 97); real process lifecycle; SIGKILL −1 / SIGTERM 0 asserted |
| `internal/stand/stand_red_test.go` | `docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md` | documented bounds derived from flags, citations | ✓ WIRED | Bounds are expressions; reviewer seconds cited only in comments |
| `internal/app/repl/sim_chaos_test.go` | `internal/app/repl/testdata/chaos-seeds.txt` | corpus iteration + drift-checked generator | ✓ WIRED | Byte-exact regeneration; fresh seeds parameterized |
| `internal/app/internal/simnet/simnet_ext_test.go` | `internal/udprepl/replproto` | test-layer encoder injection into `Policy.SizeOf` | ✓ WIRED | Generated proto messages + real `proto.Marshal`; `simnet.go` imports no replproto |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `Cluster.Delete` → `store.Delete` → `NotifyLocalWrite` | ✓ WIRED | `ErrNotFound` propagated; tombstone in write log (line 816) |
| `internal/persistfile/fsops.go` | `internal/persistfile/dumpfile.go`, `snapshot.go` | `saveAtomically` five-call sequence | ✓ WIRED | Order test proves the production Save path |
| `internal/persistfile/open_seed_test.go` | `internal/app/persist/engine.go` | `persist.Open` + real `FileStore`/codec | ✓ WIRED | Five rows assert through the returned engine |
| `internal/udprepl/decode_negative_test.go` | `internal/udprepl/codec.go`, `mirror.go` | decoder verdict machinery + `Adapter.Discards()`/`SentByType()` | ✓ WIRED | Sink-socket isolation makes counter deltas exact; `repl.go` was split into `codec.go`/`mirror.go` by later phases |
| `internal/stand/stand_delete_test.go` | `internal/stand/stand.go` | harness dump/Get/Delete seams | ✓ WIRED | NotFound, tombstone line, single-line tombstone contract |
| `internal/grpcapi/watch_senderr_test.go` | `internal/grpcapi/watch.go` | fake stream Send failures + real transport break | ✓ WIRED | Handler returns the transport error; hub stays healthy |
| `internal/stand/stand_metrics_test.go` | `internal/metrics/metrics.go` | live HTTP `/metrics` rendered samples | ✓ WIRED | Exact sample-line matching; no internal counter reads |
| `internal/app/repl/decision_coverage_test.go` | `docs/adr/0001…0011` | index + word-bounded citation regex | ✓ WIRED | All 11 ADRs exist and every decision token found in repo tests |
| `internal/app/storage/prop_small_space_test.go` | `internal/app/internal/testutil/rng.go` | `testutil.ReplayCommand` executed | ✓ WIRED | Same format helper prints and executes |
| `.github/workflows/scheduled.yml` | `scripts/red-runner`, `Makefile`, `internal/stand/*`, corpus | job commands | ✓ WIRED | Jobs invoke the single-definition targets/scripts |
| `.github/workflows/ci.yml` | `internal/stand/` | `go test -race ./internal/stand/` | ✓ WIRED | Line appears exactly once; matrix unchanged |
| `cmd/sup-kv/examples_test.go` | README/EXAMPLES blocks | `docBlockBudget` clamped in `t.Deadline()` | ✓ WIRED | Class-3 flake fixed test-side |
| `internal/stand/stand_test.go` | `cmd/sup-kv/scenario_test.go` | moved source | — HISTORICAL | The source file was intentionally deleted in 02-02 (D-03: moved, never duplicated); zero `TestExec_` remain |
| `internal/stand/stand_nondarwin_test.go` | `docs/adr/0010-darwin-development-only.md` | citation in the loud statement | ✓ WIRED | ADR exists; the test cites `decision 14.10 / ADR 0010` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `scripts/red-runner` | parsed `Action`/`Test`/`Package` | live `go test -json` of the exact spawned command | Yes — 7 pass events green, 7 fail events in the committed red run, mutant verdicts | ✓ FLOWING |
| `stand_metrics_test.go` | metric samples | live node `/metrics` over HTTP | Yes — exact constructed values | ✓ FLOWING |
| `sim_chaos_test.go` | seeds / model / stores | committed corpus + fresh wall-clock seeds + production store | Yes — convergence to the pointwise-supremum model | ✓ FLOWING |
| `stand_delete_test.go` | tombstone lines | real node dump files on disk | Yes — `"deleted":true` record lines | ✓ FLOWING |
| `bench_test.go` (persist) | per-op durations, bytes | real `persist.EncodeTo` + `persistfile.FileStore.SaveStream` | Yes — 57.13 ms p50/p99, 5 147 571 dump bytes | ✓ FLOWING |
| `open_seed_test.go` | loaded records / clock versions | real codec decode of crafted dump files | Yes — full 16-byte version comparisons | ✓ FLOWING |
| `fuzz_decode_test.go` / `fuzz_dump_test.go` | datagram/dump bytes | committed `testdata/fuzz` corpora + fuzzer | Yes — production decoders | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full red gate + mutants | `bash scripts/red-runner` | exit 0; 7/7 rows green; 4/4 controls pass; 2 mutants killed; evidence `20261123T193404Z` | ✓ PASS |
| Green suite without the red gate | `make test` | rerun exit 0; cmd/sup-kv 6.58 s, repl 45.03 s, stand 7.70 s (cached), all 12 packages ok | ✓ PASS |
| Full suite, first run | `make test` | FAIL: `TestLog_formatText` probe bind collision — same root cause as the TEST-11 gap (pre-fix tree; re-run green after Phase 8) | ✓ PASS (re-verified) |
| Strict lint gate | `make lint` | exit 0 (config verify + `run ./...`) | ✓ PASS |
| Documented race soak, one pass | `make race-soak COUNT=1` | exit 0; 7/7 packages ok | ✓ PASS |
| TEST-11 acceptance, cmd+udprepl | `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/` | Pre-fix tree: FAIL 2/2 in cmd/sup-kv (`address already in use`); after Phase 8: three consecutive green runs (2026-11-24 21:09-21:23 UTC) and green at the phase-8 gate tree | ✓ PASS (re-verified) |
| TEST-11 acceptance, cmd alone | `go test -race -count=20 ./cmd/sup-kv/` | Pre-fix tree: FAIL (same two tests); after Phase 8: green | ✓ PASS (re-verified) |
| TEST-11 focused control | `go test -race -count=20 ./cmd/sup-kv/ -run '^TestLog_'` (×2) | ok 4.48 s / 7.13 s | ✓ PASS (isolates the parallel-sibling collision) |
| Red tests skip without env | `go test -race -cover ./...` (no `SUPKV_RED_TESTS`) | red tests SKIP; package green | ✓ PASS |
| R-01 at documented scale | `SUPKV_STAND_NIGHTLY=1 SUPKV_RED_TESTS=1 go test -race -count=1 -timeout 10m ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'` | PASS 92.30 s; 20 000/20 000 keys | ✓ PASS |
| Nightly documented-scale F9 | `SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m -v ./internal/stand/ -run '^TestStand_emptyDiskRecovery$'` | PASS 51.73 s; 10 000 keys in 23.94 s inside the 20 s + 65 s bound | ✓ PASS |
| Chaos corpus + fresh seeds | `SUPKV_SIM_FRESH_SEEDS=10 go test -race ./internal/app/repl/ -run '^TestChaos_convergence$'` | ok 1.634 s | ✓ PASS |
| Benchmark smoke | `make bench-smoke` | exit 0; three benchmark lines with p50-ms/p99-ms | ✓ PASS |
| Darwin compile | `GOOS=darwin go vet ./internal/stand/` | exit 0 | ✓ PASS |
| Artifact declarations | `query verify.artifacts` over all 17 plans | 31/31 artifacts exist/pass | ✓ PASS |
| Decision coverage | `query check.decision-coverage-verify` | 9/9 honored, `not_honored: []` | ✓ PASS |

### Probe Execution

No probe scripts are declared by this phase or present in the repository (`find scripts -path '*/tests/probe-*.sh'` → none; the PLANs declare no `probe-*.sh`). This phase is a test-harness/CI phase; the equivalent behavioral evidence is the spot-check table above.

**Status:** SKIPPED (no probes exist)

### Test Quality Audit

| Test File / Group | Linked Req | Active | Skipped | Circular | Strongest Assertion | Verdict |
|-------------------|-----------|--------|---------|----------|---------------------|---------|
| `internal/grpcapi/red_gate_test.go` | TEST-04 | 1 (under red gate) | 1 env gate (by design) | No | Behavioral (gRPC codes + health) | ✓ |
| `internal/stand/stand_red_test.go` | TEST-04 | 5 (under red gate) | 5 env gates (by design) | No | Behavioral (real sockets, wall-clock bounds, metrics) | ✓ |
| `internal/persistfile/red_value_limit_test.go` | TEST-04 | 1 (under red gate) | 1 env gate (by design) | No | Behavioral (restart survival, real composition) | ✓ |
| `internal/stand/stand_test.go`, `stand_delete_test.go`, `stand_metrics_test.go` | TEST-02/03/05/06/12 | All | 0 (one root-only skip in a later-phase persist test) | No | Behavioral (process lifecycle, dumps, exact metrics) | ✓ |
| `internal/app/internal/simnet/simnet_ext_test.go` | TEST-02/06 | 6 | 0 | No | Behavioral (state, counters, digests) | ✓ |
| `internal/app/repl/sim_chaos_test.go`, `sim_scenario_test.go` | TEST-01/09 | All | 0 | No | Behavioral (model equality, bounds, fault teeth) | ✓ |
| `internal/udprepl/decode_negative_test.go`, `fuzz_decode_test.go` | TEST-02/08 | All | 0 | No | Value/Behavioral (verdicts, counters, invariants) | ✓ |
| `internal/app/persist/fuzz_dump_test.go` | TEST-08 | 1 fuzz target | 0 | No | Behavioral (codec contract, checksums, echo) | ✓ |
| `internal/persistfile/fsops_test.go`, `open_seed_test.go` | TEST-03/07 | All | 0 | No | Behavioral (order, failure matrix, clock dominance) | ✓ |
| `internal/app/storage/prop_small_space_test.go` | TEST-15 | 2 | 0 | No | Behavioral (CRDT laws, executed replay) | ✓ |
| `cmd/sup-kv/main_test.go` (port helpers) | TEST-11 | All | 0 | No | Existence/behavioral; the shared allocator holds claims and the probe retries a transient bind | ✓ (closed by Phase 8; see Gaps Summary) |

**Disabled tests on requirements:** 0 blockers — the only requirement-linked skips are the seven intentional `SUPKV_RED_TESTS` gates, and the red-runner executes each of them (pass events observed in the fresh run; fail events in the committed red run). No requirement is proven only by a skipped test.
**Circular patterns detected:** 0 — expected values are the documented acceptance expressions, exact constructed events, or production-encoder fixtures; the chaos corpus generator proves format reproducibility, not correctness.
**Insufficient assertions:** 0 for the requirement-linked tests — they assert values or multi-step behavior (codes, versions, counters, bounds, dumps), not existence/type only.
**Coverage quantities:** met — 16 corrupted-datagram rows, 12 + 8 fuzz corpus entries, 6 simulator extension tests, 5 clock-seeding rows, 7 expectation rows, 11 decision citations.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No `TBD`/`FIXME`/`XXX` in any phase-modified file | — | None |
| — | — | No placeholder/coming-soon/not-implemented text | — | None |
| — | — | No empty return stubs (the `return []…` matches are literal list constructors in tests) | — | None |
| — | — | No hardcoded empty data at render/assertion sites | — | None |

The only non-red skips are loud, environment-conditional root-permission skips in `internal/stand/persist_test.go` and four `internal/persistfile` tests (all later-phase tests, not requirements of this phase, and all skip loudly naming the cause). No debt-marker blocker exists.

### Code Review (Advisory, Recorded)

`02-REVIEW.md` reports 0 critical / 6 warning / 8 info. The disposition ledger still lists all 14 as `open`; re-checking them against the current tree:

| Finding | Substance now | Why advisory / what changed |
|---------|---------------|-----------------------------|
| WR-01 | red-runner does not classify package-level `fail` events with no test name | Partly mitigated: a missing per-test event already makes the row an absence error. Hardening candidate. |
| WR-02 | `Cluster.Write` swallowed `store.Set`'s error | **Fixed** by 03-02: `Write` panics naming operation/key/node and appends no phantom log entry; `TestSim_write_refusesOversize` pins it. |
| WR-03 | TEST-13 meta-test walked the module cache | **Fixed**: the walk starts at `go.mod`, skips `docs/` and `testdata/`, and excludes only itself. |
| WR-04 | Dispatch `seeds` validation echoed the untrusted value | **Fixed**: anchored `^[1-9][0-9]*$` check; the raw value is never interpolated into shell logic. |
| WR-05 | Port pick-to-bind race outside `internal/stand` | **Closed by Phase 8.** 08-06's shared claim-while-held allocator (claim-before-probe follow-up `7aeec35`) plus 08-09 follow-up `9cdc0f1`; the count-20 acceptance is green three consecutive runs. |
| WR-06 | `race-soak` excluded `internal/stand` | **Fixed** by 03-13: the target now includes the stand package and the Makefile records the measured cost. |
| IN-01…IN-08 | bench smoke does not assert percentiles; scenario-ID collision; darwin tag naming; duplicated small-space constants; evidence-log selection; fresh-seed logging; fuzz-corpus drift guard; hand-mirrored encoder | Informational hardening; none changes a verified acceptance. IN-08 still stands (the size gate mirrors the envelope at the test layer). |
| INFO (doc) | `ci.yml`'s comment says the stand package "fails the cell on any skip" | The five red-gated stand tests do skip loudly without `SUPKV_RED_TESTS` in the PR gate and are executed by the scheduled red-run job. The comment should say "no capability-probe skip" — the policy (no *silent* socket skip) holds; the wording overstates. |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|-------------|--------|----------|
| TEST-01 | 02-06, 02-13 | Chaos stable + mutation-sensitive | ✓ SATISFIED | Drift-checked 16-seed corpus + fresh seeds; poll-to-model; both mutants kill; control passes; 10k dispatch scheduled |
| TEST-02 | 02-03, 02-07, 02-15 | Deletion replication at every level | ✓ SATISFIED | Codec round trip, simulator `Cluster.Delete` + chaos, two-process NotFound + dump tombstone, node-down learns after start |
| TEST-03 | 02-02, 02-04 | Confirmed write is on disk, pinned by tests | ✓ SATISFIED | fsops five-step order + failure matrix + blocking Save + `TestStand_killNineRestart` process case |
| TEST-04 | 02-01, 02-05, 02-12 | R-03 acceptance tests fail on current code | ✓ SATISFIED | All four Criticals reproduced red in the committed run; green after Phases 3–5 |
| TEST-05 | 02-02, 02-14 | Real-socket tests never silently skipped | ✓ SATISFIED | Linux probe/build failures are fatal; zero `Skipf`; named darwin report; CI runs `-race` |
| TEST-06 | 02-03, 02-11, 02-15 | Simulator blind spots closed + nightly 10k | ✓ SATISFIED | Budget, size gate, announce delay, restart-with-state, deletions; nightly 10k×1KiB wired and locally green (23.94 s recovery) |
| TEST-07 | 02-16 | `persist.Open` clock seeding on real composition | ✓ SATISFIED | Five-row real FileStore + codec + Open test green |
| TEST-08 | 02-07, 02-11, 02-17 | Decoder negative tests + fuzzing | ✓ SATISFIED | 16-row table + 12/8 committed corpora + scheduled `-fuzztime` jobs |
| TEST-09 | 02-13 | Bounds under combined failures and long partitions | ✓ SATISFIED | `TestScenario_combinedFailures` / `longPartition` with lower/upper bounds, teeth and no-fresh-loss qualifier |
| TEST-10 | 02-10 | Benchmarks measure what they claim | ✓ SATISFIED | p50/p99 on all three real-path benchmarks; workload pacing does not throttle the measured spans; `make bench-smoke` green |
| TEST-11 | 02-10, 02-11 | Known flakiness classes eliminated | ✓ COMPLETE | The pick-release-then-bind class was removed by Phase 8 (08-06 allocator, follow-ups `7aeec35`/`9cdc0f1`); `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/` is green three consecutive runs and at the phase-8 gate tree |
| TEST-12 | 02-08 | Metrics/configuration tested by observed effect | ✓ SATISFIED | Exact conflict/bad-CRC/retransmit values; five parameter-effect subtests + map for the rest |
| TEST-13 | 02-09 | Every section-14 decision has a named test | ✓ SATISFIED | All 11 decisions cited in repo tests; meta-test green; decision-coverage gate 9/9 |
| TEST-14 | 02-08 | Watch cut and backpressure branches covered | ✓ SATISFIED | Send-error, snapshot-then-live cut, slow-subscriber shed, broken-transport isolation |
| TEST-15 | 02-09 | Property tests shrink and explore equal-`l` versions | ✓ SATISFIED | Small 3×3×3 space in half the draws; executable package-scoped replay command |
| TEST-16 | 02-11 | Scheduled CI workflow exists | ✓ SATISFIED | `scheduled.yml` with 1000 seeds, `-shuffle=on -count=3`, `-bench . -benchtime=1x`, cron + dispatch |

**Requirement accounting:** all 16 TEST requirement IDs appear in PLAN frontmatter (02-01…02-17) and all 16 map to Phase 2 in REQUIREMENTS.md. No orphaned TEST requirement remains unclaimed. 15/16 satisfied; TEST-11 blocked by the port pick-to-bind flake.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts — `9/9`, `not_honored: []` (decision-coverage gate, non-blocking).

### Human Verification Required

N/A — infrastructure/foundation phase with no user-facing elements. All acceptance criteria are programmatically verifiable, and every programmatic check above was executed on the live tree. No truth was left behavior-unverified (`behavior_unverified: 0`). The remaining post-merge duties (nightly cron execution, the `seeds=10000` dispatch, the macOS cell) are the scheduled/CI evidence vehicles the roadmap assigns to the workflow, not human UAT items.

### Gaps Summary

The phase's central deliverable is intact: the stand and the hardened simulator exist, all 16 requirements have their tests, the seven red-gated Critical tests were recorded RED against v1.0 in committed evidence, and every one of them is GREEN on the current tree under a fresh full red-runner run (7/7 rows, both mutants killed). The seams, simulator extensions, chaos corpus, fuzzing, metrics, decisions and property work all verify.

**TEST-11 / ROADMAP success criterion 5 — the count-20 flake bar — is now green.** The 2026-11-23 report recorded the pick-release-then-bind race in `cmd/sup-kv` (WR-05) as failed. The fix landed in Phase 8: 08-06 replaced every ephemeral picker with the shared claim-while-held allocator over four disjoint private bands, and follow-up `7aeec35` claims a candidate before its bind probe; 08-09 follow-up `9cdc0f1` measures the startup-gate bound from process start rather than spawn-queue time and retries a transient loopback-broadcast bind on a fresh port under the project's existing `probeBroadcastForm` policy. `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/` passed three consecutive runs (2026-11-24 21:09-21:23 UTC) and again at the phase-8 gate tree. No must-have remains failed or partial.

---

_Verified: 2026-11-23T19:56:26Z_
_Re-verified: 2026-11-25 — TEST-11 gap closed by Phase 8 (three green count-20 runs)_
_Verifier: ZCode (gsd-verifier)_
