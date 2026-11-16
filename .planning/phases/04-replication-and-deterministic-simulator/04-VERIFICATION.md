---
phase: 04-replication-and-deterministic-simulator
verified: 2026-11-16T10:49:51Z
status: passed
score: 5/5 must-haves verified
covered_files:
  - .planning/phases/04-replication-and-deterministic-simulator/04-01-PLAN.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-01-SUMMARY.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-02-PLAN.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-02-SUMMARY.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-03-PLAN.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-03-SUMMARY.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-04-PLAN.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-04-SUMMARY.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-05-PLAN.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-05-SUMMARY.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-06-PLAN.md
  - .planning/phases/04-replication-and-deterministic-simulator/04-06-SUMMARY.md
  - Makefile
  - buf.yaml
  - go.mod
  - go.sum
  - internal/app/internal/simnet/simnet.go
  - internal/app/persist/bench_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_internal_test.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/core.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/merge_stats_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_bound_test.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/repl/sync_test.go
  - internal/udprepl/bench_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replproto/supkv/v1/supkv.pb.go
  - internal/udprepl/replproto/supkv/v1/supkv.proto
  - internal/udprepl/scenario_test.go
  - internal/udprepl/smoke_test.go
covered_digest: "v4:sha256:013066a84ce46205171c9e3d4c9058eaa244d29c3822e5ccf41f2360282f8c06"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 5/5
  gaps_closed: [] # staleness regeneration, not gap closure — the previous pass (verified 2026-11-14T22:35:00Z, digest b898d6b8…) went stale when Phase-6 hardening landed on phase-4 files: 06-01 fixed the four carried review findings (d7c5f4e WR-01+WR-02, 82180f7 IN-01, 0f5e03d IN-02), 06-03 added observability seams (34b089f OnDiscard, 3ff81db counters), 06-05 extended simnet (f414832 clock-offset/restart seams), 06-06/06-07/06-08 added scenario/bench tests
  gaps_remaining: []
  regressions: []
---

# Phase 4: Replication and Deterministic Simulator Verification Report

**Phase Goal:** Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment, with convergence under chaos proven by the deterministic simulator
**Verified:** 2026-11-16T10:49:51Z
**Status:** passed
**Re-verification:** Yes — staleness regeneration. The previous report (verified 2026-11-14T22:35:00Z, digest `b898d6b8…`) went stale when the Phase-6 hardening session committed twelve changes onto phase-4 files after that timestamp (`git log c194c6f..HEAD`): plan 06-01 landed the four carried review fixes (d7c5f4e — WR-02 rate-domain clamp + WR-01 drained-row Acked dedup; 82180f7 — IN-01 NotifyLocalWrite closed-loop; 0f5e03d — IN-02 conditional dirty + confirmWaiters invariant + dead Open fallback), 06-03 added observability seams into the same files (34b089f OnDiscard, 3ff81db MergeStats/retransmissions/DumpStats/mirrors), 06-05 extended simnet with clock-offset/restart seams (f414832), and 06-06/06-08 added scenario and benchmark test files plus mechanical refactors of four phase-4 test files. This pass re-verifies every changed file at full depth with fresh named tests and regression-checks the unchanged set. The four disposition rows the previous report carried as open warnings (WR-01/WR-02/IN-01/IN-02) are now disposition `fixed` in 04-REVIEW-DISPOSITION.md — verified fixed in the current tree below.

## Goal Achievement

### Observable Truths

The five ROADMAP success criteria are the contract; the 42 plan-level truths
from the six PLANs refine these five. All five were re-verified against the
CURRENT codebase (post-06-01/06-03/06-05 changes) with fresh behavioral
evidence — every behavior-dependent truth has a named passing test from this
pass (commands below).

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|-----------------------------------|--------|----------|
| 1 | Two (or N) nodes discover each other via broadcast Hello with no configuration beyond `broadcast_addr`, and a write on one node becomes visible on every other within one RTT in normal operation | ✓ VERIFIED | `TestSmoke_twoNodesUDP` fresh PASS, verbose-confirmed RUN then PASS (0.02s test body, real sockets); `TestBound_normalOneRTT` fresh PASS; adapter `New` signature re-read at repl.go:219 — `(bindAddr, broadcastAddr string, nodeID uint16, store repl.StateStore, cfg repl.Config)`, unchanged; `repl.Config` re-read at core.go:111-130 — timing knobs + `Log` + `OnDiscard func(class, detail string)` (a discard-event callback 06-03 added, NOT addressing); no peer lists or static addressing anywhere |
| 2 | Every local change is broadcast and tracked per-peer (key, version) until Ack or supersession, retransmitting with exponential backoff and suspending (not dropping) while no peer is known; receivers Ack after applying to memory; the unacked set drains after a partition heals | ✓ VERIFIED | core.go re-diffed since the previous full-depth pass: changes are the WR-02 clamp + counter seams only — tracking/ladder/suspension logic untouched; `TestRetransmit_ackDrainSemantics`, `TestRetransmit_suspendAndFlush`, `TestRetransmit_backoffLadder`, `TestPeerTable_lifecycle` fresh PASS; `TestBound_partitionHeal` fresh PASS; the IN-01 rework of `NotifyLocalWrite` (now one select over notify+done with a post-send re-check) preserves the enqueue-and-return contract (D-05) — `TestNotifyLocalWrite_afterDone` + the 1100-call `TestNotifyLocalWrite_afterRunReturns` both fresh PASS |
| 3 | Full reconciliation works in both directions: chunked DataSync on the sync timer, at startup and on neighbor restart; receivers answer with DataRequest or unicast DataReplication rate-limited by `sync_response_rate`; a DataRequest for an unknown key gets no reply; a Hello with a new `run_id` triggers full sync toward the restarted node | ✓ VERIFIED | `TestSync_rateLimit` + `TestSync_syncArrived` fresh PASS; `TestRestart_runIDChangeTriggersFullSync` (+ `staleRunIdAndInFlight`, `newPeerNoFullSync`) fresh PASS; `TestSync_chunkWireBudget` fresh PASS against the real codec; the WR-02 fix STRENGTHENS this truth — `normalized()` at core.go:158 now clamps `SyncResponseRate <= 0 || > maxSyncResponseRate (1e9)` to the 500/s default, so the floor(1s/rate) spacing is non-zero at every configuration (regression `TestSync_responseRateClamp` fresh PASS, pins 1ns at 1e9 and 2ms at 1e9+1); the egress-queue prohibition re-proven: exactly three producer call sites call `enqueueResponse` (core.go:610, 639, 641 — comment at 301 and the definition at 646/652 excluded) |
| 4 | Malformed traffic is dropped with distinguishable classes: duplicate `node_id` logged as a configuration error, plus own broadcast echo, bad CRC and bad proto version each identifiable | ✓ VERIFIED | `TestDiscard_classes` + `TestDiscard_totality` (500 adversarial datagrams) fresh PASS; own-echo counted in the fresh verbose smoke run; the 06-03 OnDiscard seam makes the classes MORE distinguishable (first-class tokens `DiscardBadCRC…` via `Config.OnDiscard`, core silent per D-09, duplicate-node_id keeps its Config.Log config-error line) — additive, no classification change |
| 5 | The deterministic simulator runs 2..N nodes over a seeded virtual network that loses, duplicates, reorders and partitions datagrams, and asserts convergence within the documented bounds — normal one RTT, datagram loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval` + response time — printing the seed on failure | ✓ VERIFIED | simnet.go CHANGED since the previous pass (06-05: `Reregister`/`ReplaceNode`/`AddNode`/`SetClockOffset` seams for wall-clock-jump and restart scenarios) — re-verified at full depth: `Reregister` swaps a registration IN PLACE keeping the registration-slice position, so broadcast fan-out order and rng draw order are unchanged (determinism preserved); offsets start at zero (clock = sched.Now until test code moves it); purity re-grepped on the current file: `go func`/`time.Now`/`time.Sleep`/`sync.` = 0 matches, zero stdlib-`net` usage (the `net.` grep hits are simnet's own Network methods); simnet still imported only from `_test.go` files (0 non-test importers — D-06 holds); fresh seeded replay `SUPKV_SIM_SEED=4242 TestChaos_convergence` PASS (1.21s); all three `TestBound_*` fresh PASS — `boundPartitionHeal` was mechanically refactored to close over the shared constants, formula `2*syncInterval + (keys-1)*(1s/rate)` unchanged |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Plan-Level Truth Detail

| Plan | Truths | Check this pass |
|------|--------|-----------------|
| 04-01 (REPL-03) | 6/6 | replproto tree UNCHANGED since the previous pass (absent from `git diff c194c6f..HEAD`); `TestWire_twoNodeHandoff` fresh PASS; core purity re-grepped: `go func`/`sync.`/`time.Now`/`time.Sleep` = 0, `"net"` import absent (D-03), protobuf in internal/app = comments only (core.go:11,83 — the doc comments SAY protobuf stays adapter-side) |
| 04-02 (REPL-02/06) | 8/8 | Tracking semantics untouched by 06-01/06-03 (diff read); 4 named tests + both NotifyLocalWrite regressions fresh PASS; the IN-01 rework keeps D-05's never-block guarantee — the select is over buffered send vs done, both immediate |
| 04-03 (REPL-04/05) | 8/8 | Backstop truth re-proven FRESH: `TestSync_rateLimit` PASS with the clamped normalization in force; `TestSync_responseRateClamp` NEW (06-01) pins both boundary sides — the floor(1s/rate) integer-division rule is now enforced over the whole domain; egress producers exactly 3 |
| 04-04 (REPL-07) | 8/8 | `TestRestart_runIDChangeTriggersFullSync` + both edge regressions fresh PASS; restart branch untouched by the phase-6 deltas (restart_test.go's only change is a `remoteRecord` helper signature simplification, assertions identical) |
| 04-05 (TEST-03/REPL-10) | 6/6 | simnet changed (06-05 seams) — full-depth re-verification above; determinism invariants re-proven by the fresh seeded replay + all three bounds |
| 04-06 (REPL-01/08/09) | 6/6 | `TestSmoke_twoNodesUDP` fresh verbose PASS; `TestDiscard_classes`/`TestDiscard_totality` fresh PASS; persist seam re-proven: ApplyRemote now marks dirty ONLY on actual state change (IN-02 fix) — the REPL-03 memory-apply contract (merge, no waiter, keep serving) is preserved and `TestEngine_applyRemoteTriggersDump` fresh PASS |

### Changed-Area Verification: the Phase-6 commits on Phase-4 files

Every covered file that changed since 2026-11-14T22:35:00Z, at full depth
(`git diff --stat c194c6f..HEAD`: 20 files, +3308/−131):

- **`internal/app/repl/core.go`** (122 lines delta) — three additive changes,
  each read in the current tree: (1) WR-02 fix — `maxSyncResponseRate = 1e9`
  constant (core.go:66) + clamp in `normalized()` (core.go:158) with the
  doc comments updated; (2) `Config.OnDiscard func(class, detail string)`
  (core.go:129) normalized to a no-op (core.go:164) — a callback, no
  addressing surface, REPL-01 prohibition intact; (3) `MergeStats` counters
  + `Retransmissions()` observer (core.go:213-220, 965-976) — read-only
  observability fed from existing handler paths. Purity greps on the current
  file: 0 goroutines, 0 wall-clock, 0 sync, 0 net imports. All
  classification/compare/retransmit/restart decision logic byte-stable
  outside these additions.
- **`internal/udprepl/repl.go`** (185 lines delta) — (1) WR-01 fix:
  `retainDrained` now `slices.Sort` + `slices.Compact` (repl.go:643-645) so a
  peer both pre-Acked and discovery-flushed publishes exactly once —
  mirror-observability only, the core pending set was already correct;
  (2) IN-01 fix: `NotifyLocalWrite` is one select over notify+done with a
  post-send re-check (repl.go:327-345) — the TOCTOU nil-return window is
  closed, the post-Run `ErrRunLoopExited` is deterministic, and the writer
  still never blocks (D-05); (3) `New` seeds `cfg.OnDiscard = logDiscard`
  when nil (repl.go:220-222) — the slog bridge making discards structured
  events by default (D-08); (4) counter mirrors (MergeStats/Retransmissions
  reflections) post-event. Signature of `New` unchanged; single event loop
  unchanged.
- **`internal/app/persist/engine.go`** (192 lines delta) — (1) IN-02 fix:
  `ApplyRemote` (engine.go:501-519) probes the key live before the merge and
  marks dirty only on actual displacement (`Winner == incoming`) or first
  install under a new key — rejected stale redeliveries and equal-version
  no-ops trigger no dump; `Restore` (engine.go:572-591) skips a snapshot
  equal to the store's current state as a whole-restore no-op (clean engine,
  final flush replies nil, no dump); (2) `confirmWaiters` gains its invariant
  assertion (a resolved waiter is never re-queued); (3) Open's dead
  Decode-error fallback removed (decoding rides Load's oracle argument);
  (4) `DumpStats` observability seam. The REPL-03 contract (merge + dirty +
  no waiter + keep serving) is preserved — `TestEngine_applyRemoteTriggersDump`
  fresh PASS.
- **`internal/app/internal/simnet/simnet.go`** (162 lines delta) — 06-05
  seams (`Reregister` in-place registration swap, `ReplaceNode`/`AddNode`
  rebuild-from-remembered-cfg, per-node mutable clock-offset slots). All
  additive to the Cluster harness; the scheduler, fault-draw discipline and
  drive loop are unchanged; determinism re-proven by fresh seeded replay.
  Still test-only linked (0 non-test importers).
- **Test files** — `retransmit_test.go`/`restart_test.go` (`remoteRecord`
  helper signature simplification — assertions identical),
  `sim_bound_test.go` (`boundPartitionHeal` closes over shared constants —
  same formula), `smoke_test.go` (`freeUDPPort`/`waitFor` generalized to
  `testing.TB` for benchmark reuse; one `//nolint:usetesting` with a written
  in-line justification — the suggested fix does not compile on the TB
  interface), `sync_test.go` (+`TestSync_responseRateClamp`),
  `repl_test.go` (+dedup/closed-loop regressions), `discard_test.go`
  (+OnDiscard assertions), `engine_test.go` (+redundant-dirty cases). New
  files: `engine_internal_test.go`, `merge_stats_test.go`,
  `sim_scenario_test.go`, `scenario_test.go`, `repl_internal_test.go`,
  `bench_test.go` ×2 — all green in the full-suite run. Zero phase-4
  assertions weakened (every diff read).
- **`Makefile`** (+12) — a manual-only `bench` target (D-13/D-14: no CI
  execution, no thresholds, no stored profiles); `test`/`lint` targets
  untouched. `buf.yaml`/`go.mod`/`go.sum`/replproto: ABSENT from the diff —
  unchanged since the previous verified pass.

### Required Artifacts

All artifacts across the six plans exist, are substantive, and are wired.
(`gsd-tools query verify.artifacts` returns 0/0 on every plan — the known
parser limitation with string-form artifact entries documented in every
previous pass; manual Level 1-4 verification stands.)

| Artifact | Lines | Status | Wiring |
|----------|-------|--------|--------|
| `internal/port/repl.go` | 140 | ✓ VERIFIED | Unchanged since 2026-11-08 — absent from c194c6f..HEAD; the seam both sides speak |
| `internal/app/repl/core.go` | 1105 | ✓ VERIFIED | CHANGED (06-01/06-03) — re-verified at full depth this pass; purity greps 0; single-event-loop discipline intact |
| `internal/udprepl/repl.go` | 760 | ✓ VERIFIED | CHANGED (06-01/06-03) — re-verified at full depth; `New` signature unchanged; single event loop; OnDiscard bridge |
| `internal/udprepl/replproto/supkv/v1/supkv.proto` | 74 | ✓ VERIFIED | Unchanged since the previous pass; buf zero-waiver workspace lint/format green then, tree untouched since |
| `internal/udprepl/replproto/supkv/v1/supkv.pb.go` | 660 | ✓ VERIFIED | Unchanged since the previous pass; consumed by repl.go (sole importer); `TestCodec_*` green in full suite |
| `internal/app/repl/{retransmit,peertable,sync,restart,discard,sim_bound,sim_chaos}_test.go` | all | ✓ VERIFIED | All green in the full race-suite run; retransmit/sync/restart/discard/bounds re-run fresh this pass |
| `internal/udprepl/syncsize_internal_test.go` | 149 | ✓ VERIFIED | Unchanged; fresh PASS (`TestSync_chunkWireBudget`) |
| `internal/app/internal/simnet/simnet.go` + `simnet_test.go` | 623 + 218 | ✓ VERIFIED | simnet.go CHANGED (06-05 seams) — full-depth re-verification above; determinism purity re-grepped 0; test-only linkage re-proven (0 non-test importers) |
| `internal/app/persist/engine.go` | 672 | ✓ VERIFIED | CHANGED (06-01/06-03/06-09 darwin flush) — ApplyRemote conditional dirty + Restore no-op verified in current tree; fresh seam tests PASS |
| `internal/app/persist/engine_test.go` | 1623 | ✓ VERIFIED | Extended (+150, redundant-dirty cases); green |
| `internal/udprepl/smoke_test.go` | 352 | ✓ VERIFIED | Refactored to testing.TB helpers only; fresh verbose PASS this pass (ran, not skipped) |

### Key Link Verification

All key links WIRED; links touching changed files re-verified by direct code
read this pass, the rest regression-checked with fresh tests:

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| port.Message structs | replproto generated types | `protoBody`/`portMessage` mapping | ✓ WIRED | Unchanged; `TestWire_twoNodeHandoff` + `TestSync_chunkWireBudget` fresh green |
| udprepl event loop | repl.Core handlers | Only `Run`'s loop calls HandleDatagram/NotifyLocalWrite/Advance | ✓ WIRED | Re-read in the changed repl.go — the 06-01/06-03 deltas add no second driver |
| NotifyLocalWrite | pending entry + broadcast | One select over notify+done, re-check after won send | ✓ WIRED | REWORKED by IN-01 fix — `TestNotifyLocalWrite_afterDone` + `afterRunReturns` fresh PASS |
| inbound Ack | wait-set drain | `handleACK` exact-version drain + sorted-idempotent `ackedInsert` | ✓ WIRED | `TestRetransmit_ackDrainSemantics` fresh PASS |
| Advance | retransmit + eviction scans | Single loop fires both, sorted by key | ✓ WIRED | `TestRetransmit_backoffLadder` + `TestPeerTable_lifecycle` fresh PASS |
| inbound DataRequest | rate-limited reply | Snapshot lookup → `enqueueResponse` → `drainEgress` | ✓ WIRED | `TestSync_rateLimit` fresh PASS; exactly 3 producer sites (core.go:610,639,641) |
| startup/timer/restart | chunked listing emission | One `listingChunks` build serves both triggers | ✓ WIRED | `TestRestart_runIDChangeTriggersFullSync` fresh PASS |
| simnet scheduler | Core handlers | `deliver`→HandleDatagram, `drive`→Advance | ✓ WIRED | Extended (06-05 restart seams drive the SAME handlers); seeded chaos replay fresh PASS |
| simnet network | Outbox drain | `route` sends every Outbound; Broadcast fans out incl. sender | ✓ WIRED | `Reregister` keeps slice position — fan-out order stable |
| per-node store | sim clock | `clock.NewClock(sched.Now + offset slot)` | ✓ WIRED | Offsets start at zero; SetClockOffset is test-driven only |
| observability seams | quiescence | `PendingDeliveries` + `UnackedStats().Count` + `QueuedResponses()` | ✓ WIRED | Exercised by the fresh seeded chaos run |
| persist.Engine.ApplyRemote | repl.StateStore | Signature match | ✓ WIRED | `TestEngine_applyRemoteTriggersDump` fresh PASS on the conditional-dirty version |
| adapter atomics | smoke test / Phase-5+ readers | `KnownPeers`/`OwnEcho`/`Discards()`/`MergeStats`/`Retransmissions` mirrored post-event | ✓ WIRED | `TestMirror_*` family fresh PASS |
| OnDiscard seam (NEW 06-03) | structured discard events | core invokes `cfg.OnDiscard(class, detail)` at every discard site; adapter defaults to slog bridge | ✓ WIRED | `TestDiscard_classes` fresh PASS; discard_test.go OnDiscard assertions green |

### Data-Flow Trace (Level 4)

No rendered/static-value surfaces in this backend phase; the data-flow
equivalents all trace to real sources:

| Path | Source | Status |
|------|--------|--------|
| Replicated records | `store.ApplyRemote` → real `storage.Store.Merge` (LWW join) | ✓ FLOWING |
| Dirty-triggered dumps | `ApplyRemote` marks dirty ONLY on `displaced \|\| first-install` (live probe + Merge Conflict contract) — a real state signal, not an unconditional flag | ✓ FLOWING |
| DataSync listings | `store.Records()` snapshot per build | ✓ FLOWING |
| Chaos model store | `WriteLog` merged by real `Merge` (simnet) | ✓ FLOWING |
| Discard/peer/fault/merge/retransmit counters | Core state mutated by real handler calls; mirrored to atomics post-event | ✓ FLOWING |
| Pending mirror + drained rows | `core.PendingSnapshot()` per iteration; drained rows from real Ack correlation, Acked union now deduplicated (WR-01) | ✓ FLOWING |
| Wire encode/decode | Real proto Marshal/Unmarshal — exercised live by fresh wire/chunk/smoke runs | ✓ FLOWING |

### Behavioral Spot-Checks

Single-test discipline; every command run fresh (`-count=1`, `-race`) this
pass. The full suite ran exactly once (`make test`, exit 0, all packages ok,
coverage recorded); `make lint` ran once (exit 0, zero findings).

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| WR-02 fix: rate-domain clamp both boundary sides | `go test -race -count=1 -run 'TestSync_responseRateClamp' ./internal/app/repl/` | PASS (1.03s) | ✓ PASS |
| WR-01 fix: drained-row Acked dedup through the real event loop | `go test -race -count=1 -run 'TestMirror_drainedAckedDedup' ./internal/udprepl/` (in mirror batch) | PASS (1.06s) | ✓ PASS |
| IN-01 fix: deterministic post-done error + 1100-call after-Run | `go test -race -count=1 -run 'TestNotifyLocalWrite_afterDone\|TestNotifyLocalWrite_afterRunReturns' ./internal/udprepl/` (in mirror batch) | PASS (1.06s) | ✓ PASS |
| IN-02 fix: redundant-dirty suppressed (ApplyRemote + Restore no-op) | `go test -race -count=1 -run 'TestEngine_redundantDirty' ./internal/app/persist/` (in persist batch) | PASS (1.02s) | ✓ PASS |
| Real-socket two-node discovery (REPL-01) | `go test -race -count=1 -run 'TestSmoke_twoNodesUDP' ./internal/udprepl/` | PASS — verbose `=== RUN` → `--- PASS` (0.02s), ran not skipped | ✓ PASS |
| Seeded determinism replay (D-06) | `SUPKV_SIM_SEED=4242 go test -race -count=1 -run 'TestChaos_convergence' ./internal/app/repl/` (in bounds batch) | PASS (1.21s) | ✓ PASS |
| Convergence bounds (REPL-10) | `go test -race -count=1 -run 'TestBound_normalOneRTT\|TestBound_partitionHeal\|TestBound_lossWithinAckTimeoutMax' ./internal/app/repl/` | PASS (1.21s) | ✓ PASS |
| Wire handoff + cross-seam chunk budget | `go test -race -count=1 -run 'TestWire_twoNodeHandoff\|TestSync_chunkWireBudget' ./internal/udprepl/` (in smoke batch) | PASS (1.03s) | ✓ PASS |
| Ack drain / suspension / ladder / peer lifecycle (REPL-02/06) | `go test -race -count=1 -run 'TestRetransmit_ackDrainSemantics\|TestRetransmit_suspendAndFlush\|TestRetransmit_backoffLadder\|TestPeerTable_lifecycle' ./internal/app/repl/` | PASS (1.04s) | ✓ PASS |
| Rate-limiter spacing + sync-arrived (REPL-05) | `go test -race -count=1 -run 'TestSync_rateLimit\|TestSync_syncArrived' ./internal/app/repl/` | PASS (1.03s) | ✓ PASS |
| Restart full sync + edges (REPL-07) | `go test -race -count=1 -run 'TestRestart_runIDChangeTriggersFullSync\|TestRestart_staleRunIdAndInFlight\|TestRestart_newPeerNoFullSync' ./internal/app/repl/` (in retransmit batch) | PASS (1.04s) | ✓ PASS |
| Discard classification + totality (REPL-08/09) | `go test -race -count=1 -run 'TestDiscard_classes\|TestDiscard_totality' ./internal/app/repl/` (in retransmit batch) | PASS (1.04s) | ✓ PASS |
| Adapter mirror seams | `go test -race -count=1 -run 'TestMirror_pendingAckVisible\|TestMirror_syncArrivedFlip\|TestMirror_drainedAckedDedup' ./internal/udprepl/` (in mirror batch) | PASS (1.06s) | ✓ PASS |
| Persist Flush + ApplyRemote seam | `go test -race -count=1 -run 'TestFlush_remoteDirtiness\|TestFlush_clean\|TestFlush_closed\|TestEngine_applyRemoteTriggersDump\|TestConfirmWaiters_aboveCoverInvariant' ./internal/app/persist/` | PASS (1.02s) | ✓ PASS |
| Full workspace, race detector | `make test` | exit 0, all packages ok | ✓ PASS |
| Strict lint gate | `make lint` | exit 0, zero findings | ✓ PASS |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` exist or are referenced by
any PLAN/SUMMARY (unchanged finding).

### Requirements Coverage

Plan frontmatter union: REPL-01,02,03,04,05,06,07,08,09,10 + TEST-03 —
exactly the 11 IDs REQUIREMENTS.md assigns to Phase 4 (all marked Complete
in its traceability table, lines 162-172). No orphaned requirements, no
unclaimed IDs.

| Requirement | Source Plan | Status | Evidence |
|-------------|------------|--------|----------|
| REPL-01 | 04-01, 04-06 | ✓ SATISFIED | `TestSmoke_twoNodesUDP` fresh verbose PASS; `New` + `Config` re-read on the CURRENT tree — `OnDiscard` is the only new field, a callback, no addressing surface |
| REPL-02 | 04-02 | ✓ SATISFIED | Tracking/backoff/suspension untouched by the phase-6 deltas (diff read); 4 named tests + NotifyLocalWrite regressions fresh PASS |
| REPL-03 | 04-01, 04-06 | ✓ SATISFIED | `TestWire_twoNodeHandoff` fresh PASS; Ack-after-apply unchanged; ApplyRemote conditional dirty preserves the memory-apply contract, `TestEngine_applyRemoteTriggersDump` fresh PASS |
| REPL-04 | 04-03 | ✓ SATISFIED | handleDataRequest unchanged; unknown-key silence covered by `TestDataRequest_replyAndSilence` (green in full suite) |
| REPL-05 | 04-03 | ✓ SATISFIED | Chunked listing + limiter unchanged; `TestSync_rateLimit` fresh PASS; WR-02 clamp closes the silent-disable edge — limiter provably never off |
| REPL-06 | 04-06→04-02 | ✓ SATISFIED | `TestPeerTable_lifecycle` fresh PASS; unbounded node_id-keyed map unchanged |
| REPL-07 | 04-04 | ✓ SATISFIED | `TestRestart_runIDChangeTriggersFullSync` + both edge regressions fresh PASS |
| REPL-08 | 04-06 | ✓ SATISFIED | Both duplicate variants drop + Log + count, now ALSO surfaced via OnDiscard tokens; `TestDiscard_classes` fresh PASS |
| REPL-09 | 04-06 | ✓ SATISFIED | `TestDiscard_totality` fresh PASS; own-echo counted in the fresh smoke run; CRC wording regression grep: every "tamper" mention is "never tamper protection" (repl.go:701, pb.go:33, store.go:130, doc.go:35) |
| REPL-10 | 04-05 | ✓ SATISFIED | All three `TestBound_*` fresh PASS on the seam-extended simnet |
| TEST-03 | 04-05 | ✓ SATISFIED | Chaos convergence + determinism — fresh seeded replay PASS after the 06-05 simnet extension |

### Anti-Patterns Found

Debt-marker gate: zero `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`
across all 19 changed/new covered implementation files (grep exit 1 — no
matches), re-run this pass. Empty-implementation scan on the four changed
production files: zero matches. The four review findings the previous report
carried as open warnings are now FIXED in code with dedicated regressions
(04-REVIEW-DISPOSITION.md: all five rows — incl. CR-01 — disposition
`fixed`, open: 0):

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/udprepl/repl.go | 643-645 | WR-01 duplicate node id in published drained row — **FIXED** (d7c5f4e): sort-then-`slices.Compact` after the Acked∪WaitingOn union; regression `TestMirror_drainedAckedDedup` fresh PASS | resolved | Mirror observability correctness restored |
| internal/app/repl/core.go | 158 | WR-02 `SyncResponseRate > 1e9` silently disabled the limiter — **FIXED** (d7c5f4e): `maxSyncResponseRate` clamp in `normalized()`; regression `TestSync_responseRateClamp` pins 1ns@1e9 / 2ms@1e9+1, fresh PASS | resolved | Limiter provably never silently off |
| internal/udprepl/repl.go | 327-345 | IN-01 NotifyLocalWrite pre-check TOCTOU nil-return window — **FIXED** (82180f7): one select over notify+done with post-send re-check; regressions `TestNotifyLocalWrite_afterDone` + `afterRunReturns` fresh PASS | resolved | Post-Run error deterministic; no drop reported as success |
| internal/app/persist/engine.go | 501-519, 572-591 | IN-02 unconditional dirty on ApplyRemote/Restore — **FIXED** (0f5e03d): conditional dirty coded to Merge's Conflict contract + whole-restore no-op; regression `TestEngine_redundantDirty` fresh PASS | resolved | No redundant dumps on no-op redeliveries |

No new anti-patterns introduced by the phase-6 changes.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Verdict |
|-----------|-----------|--------|---------|----------|---------|
| internal/udprepl/smoke_test.go | REPL-01, REPL-09 | yes (Linux verbose RUN+PASS this pass; macOS loud-skip by ratified D-08 design) | conditional-by-design | no | VALID |
| internal/udprepl/repl_test.go (+repl_internal_test.go) | REPL-03 wire + 06-01 regressions | yes | 0 | no | VALID |
| internal/app/repl/sim_chaos_test.go | TEST-03, REPL-10 | yes | 0 | no (expected values from the supremum model, not the SUT) | VALID |
| internal/app/repl/{retransmit,peertable,sync,restart,discard,sim_bound}_test.go | REPL-02..10 | yes | 0 | no | VALID |
| internal/app/repl/{sim_scenario,merge_stats}_test.go (new, phase-6) | TEST-03/OBS-01 adjuncts | yes | 0 | no | VALID |
| internal/udprepl/{scenario,bench}_test.go (new, phase-6) | scenario adjuncts | yes | 0 | no | VALID |
| internal/app/persist/engine_test.go (+engine_internal_test.go, bench_test.go) | REPL-03 persist seam + 06-01 regressions | yes | 0 | no | VALID |

**Disabled tests on requirements:** 0. **Circular patterns detected:** 0.

### Prohibition Disposition (ADR-550 D4)

| Prohibition | Tier | Verdict |
|-------------|------|---------|
| REPL-01 (04-01, 04-06): no config surface beyond broadcast_addr | test | ENFORCED — `New` (repl.go:219) and `Config` (core.go:111-130) re-read on the CURRENT tree: timing knobs + Log + OnDiscard callback only; no peer lists, no static addressing anywhere |
| REPL-08 (04-06): duplicate node_id never silently resolved | test | ENFORCED — both variants drop + Log + count, plus OnDiscard token; `TestDiscard_classes` fresh PASS |
| 04-02: pending never dropped on timeout/eviction/empty table | test | ENFORCED — only Ack/supersession delete (diff-confirmed untouched); `TestRetransmit_suspendAndFlush` fresh PASS |
| 04-02/D-05: retransmission never blocks a local write | test | ENFORCED — the IN-01-reworked NotifyLocalWrite is still enqueue-and-return (select of two immediate cases); `TestNotifyLocalWrite_afterRunReturns` fresh PASS |
| 04-02/REPL-06: no blacklist, no peer-count cap | test | ENFORCED — evicted peers re-learned; unbounded node_id-keyed map (diff-confirmed untouched) |
| 04-03: no bypass of the rate-limited egress queue | test | ENFORCED — exactly three producer sites call `enqueueResponse` (core.go:610, 639, 641; re-grepped on current tree) |
| 04-03/A4: chunks carry no continuation state | test | ENFORCED — self-sufficient slices (diff-confirmed untouched) |
| 04-03/REPL-04: replies unicast only | test | ENFORCED — unicast/queued unicast only (diff-confirmed untouched) |
| 04-04: no second peer row on run_id change | test | ENFORCED — upsert keyed by node_id; `TestRestart_staleRunIdAndInFlight` fresh PASS |
| 04-04: full-sync reaction only on run_id change | test | ENFORCED — inequality is the only trigger; `TestRestart_newPeerNoFullSync` fresh PASS |
| 04-05: no second protocol implementation in simnet | test | ENFORCED — the 06-05 seams (Reregister/ReplaceNode/AddNode/SetClockOffset) drive the SAME real Core handlers; 0 non-test importers (re-grepped) |
| 04-05: no wall-clock dependence | test | ENFORCED — `time.Sleep`/`time.Now`/`go func`/`sync.` in core.go and simnet.go: 0 matches (re-grepped on current tree) |
| 04-05/D-06: sim tooling not in production binary | test | ENFORCED — simnet imported only from `_test.go` files (re-grepped: 0 non-test importers) |
| REPL-09 CRC wording (04-01, 04-06) | judgment | **RATIFIED — resolved** (04-UAT.md item 1, human ratification 2026-11-06/07; UAT status complete). Regression grep this pass: every production mention still says corruption detection only, never tamper protection (repl.go:700-701, replproto/supkv/v1/supkv.pb.go:33, storage/store.go:130, storage/doc.go:35). No flagged prohibitions remain |

### Human Verification Required

None outstanding. Infrastructure/foundation phase — no user-facing
elements; all acceptance criteria verifiable programmatically. The previous
report's two items remain resolved with recorded evidence: the CRC-wording
judgment prohibition is ratified by human UAT (04-UAT.md, status complete,
2026-11-06/07) with the wording re-grepped unchanged on the current tree,
and D-08's macOS smoke outcome is empirically settled by CI (post-fix run
42514571899, `gate (macos-15)` success). No PRESENT_BEHAVIOR_UNVERIFIED
truths — every behavior-dependent truth has a fresh passing named test this
pass, including all four 06-01 fix regressions.

### Gaps Summary

None. The staleness trigger was the Phase-6 hardening session landing on
phase-4 files after the previous verified timestamp: plan 06-01 fixed the
four carried review findings (WR-01 drained-row Acked dedup, WR-02
rate-domain clamp, IN-01 NotifyLocalWrite closed-loop, IN-02 conditional
dirty — each verified fixed in the current tree with a fresh passing
regression), 06-03 added the OnDiscard and counter observability seams
(additive callbacks/mirrors — no decision logic touched), 06-05 extended
simnet with restart/clock-offset seams that preserve the determinism
discipline (in-place registration, zero-purity violations, test-only
linkage), and 06-06/06-08 added scenario/benchmark tests plus mechanical
refactors that weaken no phase-4 assertion. Every changed file was
re-verified at all four levels with fresh named tests; the full race suite
and the strict lint gate are green. The Phase-4 contract is intact on the
current tree — arguably stronger than at the last pass (the limiter can no
longer be silently disabled, and the mirror can no longer publish duplicate
Acked entries). The phase goal — convergence to the pointwise supremum over
a lossy UDP segment, proven under chaos by the deterministic simulator —
remains achieved in the codebase. Status is `passed`: no gaps, no
outstanding human-verification items, no flagged prohibitions.

---

_Verified: 2026-11-16T10:49:51Z_
_Verifier: Claude (gsd-verifier)_
