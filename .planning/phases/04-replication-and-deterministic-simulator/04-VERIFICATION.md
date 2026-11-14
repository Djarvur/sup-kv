---
phase: 04-replication-and-deterministic-simulator
verified: 2026-11-14T19:45:24Z
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
  - go.mod
  - go.sum
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/core.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sync_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replproto/supkv.pb.go
covered_digest: "v4:sha256:a14f92aadbb2ee97cbcb0f7517ace67869bf449fcd62baf95629ad8614a64f86"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 5/5
  gaps_closed: [] # staleness regeneration, not gap closure — phase 5 changed covered source files (repl.go, core.go, engine.go, supkv.pb.go, Makefile, go.mod/go.sum and four test files) after the last verified timestamp
  gaps_remaining: []
  regressions: []
---

# Phase 4: Replication and Deterministic Simulator Verification Report

**Phase Goal:** Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment, with convergence under chaos proven by the deterministic simulator
**Verified:** 2026-11-14T19:45:24Z
**Status:** passed
**Re-verification:** Yes — staleness regeneration. The previous report (verified 2026-11-08T21:09:31Z, digest `b9ffd02e…`) went stale when Phase 5's work landed on Phase-4 files: `internal/udprepl/repl.go` (commits d1026dc, 05f84de — copy-on-write pending mirror with drained-row retention), `internal/app/repl/core.go` (b06b54d sync-arrived state + `PendingInfo.Acked`; af3d5c4 counting-behavior doc), `internal/app/persist/engine.go` (86d33a6 exported `Flush`), the buf regeneration of `internal/udprepl/replproto/supkv.pb.go` (ed880c7), `Makefile`/`go.mod`/`go.sum` (buf tooling, grpc deps), and the test files `repl_test.go`, `retransmit_test.go`, `sync_test.go`, `engine_test.go`. `git log --since` confirms exactly these covered files changed; `smoke_test.go` and `supkv.proto` were NOT touched after the previous pass. This re-verification examines every changed file at full depth (all four levels plus fresh named tests) and regression-checks the unchanged set.

## Goal Achievement

### Observable Truths

The five ROADMAP success criteria are the contract; the 38 plan-level truths
from the six PLANs refine these five. All five were re-verified against the
CURRENT codebase with fresh behavioral evidence (every behavior-dependent
truth has a named passing test from this pass — commands below).

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|-----------------------------------|--------|----------|
| 1 | Two (or N) nodes discover each other via broadcast Hello with no configuration beyond `broadcast_addr`, and a write on one node becomes visible on every other within one RTT in normal operation | ✓ VERIFIED | `TestSmoke_twoNodesUDP` fresh PASS this pass (0.02s, real sockets, zero config — adapter `New` still takes only bind/broadcast/nodeID/store/cfg; `Config` holds only timing knobs + Log, no peer lists or static addressing); `TestBound_normalOneRTT` fresh PASS; `TestWire_twoNodeHandoff` fresh PASS |
| 2 | Every local change is broadcast and tracked per-peer (key, version) until Ack or supersession, retransmitting with exponential backoff and suspending (not dropping) while no peer is known; receivers Ack after applying to memory; the unacked set drains after a partition heals | ✓ VERIFIED | Changed file (core.go) re-verified at full depth: drain rule intact (`handleAck`: wait-set empty + `len(c.peers) > 0` guard), suspension/flush intact (`scanRetransmit` empty-table freeze), eviction never touches `acked`; `TestRetransmit_ackDrainSemantics`, `TestRetransmit_suspendAndFlush`, `TestRetransmit_backoffLadder`, `TestPeerTable_lifecycle` all fresh PASS; `TestBound_partitionHeal` fresh PASS |
| 3 | Full reconciliation works in both directions: chunked DataSync on the sync timer, at startup and on neighbor restart; receivers answer with DataRequest or unicast DataReplication rate-limited by `sync_response_rate`; a DataRequest for an unknown key gets no reply; a Hello with a new `run_id` triggers full sync toward the restarted node | ✓ VERIFIED | `handleDataSync`/`handleDataRequest`/`enqueueResponse`/`drainEgress`/`listingChunks`/restart branch all re-read intact; `TestSync_rateLimit` fresh PASS (floor(1s/rate) spacing pinned); `TestSync_syncArrived`, `TestRestart_runIDChangeTriggersFullSync`, `TestSync_chunkWireBudget`, `TestDataRequest_replyAndSilence` (full-suite) green |
| 4 | Malformed traffic is dropped with distinguishable classes: duplicate `node_id` logged as a configuration error, plus own broadcast echo, bad CRC and bad proto version each identifiable | ✓ VERIFIED | Classifier re-read (envelope-domain Undecodable before CRC gate; BadCRC carries envelope fields, body never unmarshaled); `TestDiscard_classes` + `TestDiscard_totality` fresh PASS |
| 5 | The deterministic simulator runs 2..N nodes over a seeded virtual network that loses, duplicates, reorders and partitions datagrams, and asserts convergence within the documented bounds — normal one RTT, datagram loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval` + response time — printing the seed on failure | ✓ VERIFIED | simnet byte-identical since previous pass (Scheduler/Network/FaultStats seams intact); fresh determinism replay `SUPKV_SIM_SEED=4242 TestChaos_convergence` PASS (0.05s); all three `TestBound_*` fresh PASS; full race suite green |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Plan-Level Truth Detail

| Plan | Truths | Check this pass |
|------|--------|-----------------|
| 04-01 (REPL-03) | 6/6 | core.go/port seam re-read at full depth (changed); pb.go provenance re-verified (buf-regenerated, marker intact); named tests fresh PASS |
| 04-02 (REPL-02/06) | 8/8 | retransmit/peertable machinery re-read in changed core.go; 4 named tests fresh PASS |
| 04-03 (REPL-04/05) | 8/8 | limiter arithmetic re-read (spacing `time.Second/rate` integer division intact); IDEA.md:147 echo confirmed verbatim; `TestSync_rateLimit` fresh PASS |
| 04-04 (REPL-07) | 8/8 | restart branch re-read in changed core.go; `TestRestart_runIDChangeTriggersFullSync` fresh PASS |
| 04-05 (TEST-03/REPL-10) | 6/6 | simnet + bound/chaos tests unchanged (byte-identical); seeded replay re-run fresh — PASS |
| 04-06 (REPL-01/08/09) | 6/6 | discard classifier re-read; smoke test unchanged since previous full-depth pass; fresh `TestSmoke_twoNodesUDP` + `TestDiscard_*` PASS |

The backstop-tagged truth of 04-03 (integer `floor(1s/rate)` spacing, first
response immediate, deterministic drain order, IDEA.md §Parameters echo) has
explicit behavioral evidence this pass: `TestSync_rateLimit` subtest "default
500/s: first immediate, then exactly one per 2ms in listing order" asserts
exactly one emission per 2 ms slot, zero emission at a half-spacing probe
Advance, and listing-order drain — fresh PASS; IDEA.md:147 carries the
verbatim spacing rule. No abstention required.

### Changed-Area Verification: Phase-5 commits on Phase-4 files

Every covered file that changed since 2026-11-08T21:09:31Z, at full depth:

- **`internal/udprepl/repl.go`** (d1026dc + 05f84de, +207 lines) — new
  copy-on-write pending mirror (`pendingCopy atomic.Pointer[map]`), the
  `SyncArrived` mirror, and drained-row retention (`retainDrained`).
  Level 1: present, 719 lines. Level 2: full read of the new code — the
  published map is rebuilt fresh per iteration, every `WaitingOn`/`Acked`
  slice cloned, pointer swapped wholesale, never mutated after publish; the
  `lastAck` correlation is cleared per iteration so a stale Ack cannot
  mislabel a later supersession; `prevPending` is the present→absent
  transition detector. Level 3: all core reads stay on the event-loop
  goroutine — only `Run`'s loop calls `HandleDatagram`/`NotifyLocalWrite`/
  `Advance` (re-grepped: lines 270/282/294/298), mirrors stored at the
  single post-event site; D-04 single-driver discipline preserved.
  Level 4: mirrored values flow from real core state
  (`PendingSnapshot()`, `SyncArrived()`), no static source. One defect
  found by the project's own review (WR-01, see Anti-Patterns) — Warning,
  not goal-blocking. Fresh tests: `TestMirror_pendingAckVisible`,
  `TestMirror_syncArrivedFlip`, `TestNotifyLocalWrite_afterRunReturns`,
  `TestSmoke_twoNodesUDP` all PASS.
- **`internal/app/repl/core.go`** (b06b54d + af3d5c4, +112/−33) — new
  `syncArrived` flag set on first received DataSync (any chunk, empty
  listing included), `PendingInfo.Acked` + `ackedInsert` (sorted-idempotent
  insert via BinarySearch), `handleAck` counting-behavior doc. All Phase-4
  seams re-read intact: applyRemote error path (CR-01 fix: no Ack, no
  supersession on store error), drain rule with the load-bearing
  empty-table guard, `scanRetransmit` suspension + 200/400/800/1600/3200ms
  → 5000ms saturating ladder documented in-code, `evictPeers`
  wait-set-only removal (never touches `acked`), sorted-by-key emission,
  `drainEgress` floor(1s/rate) spacing, `listingChunks` conservative-budget
  chunking. No `net` import (D-03), no `time.Sleep`/`time.Now` inside the
  core (D-02 seam — `now` is a parameter). Fresh tests: 10 named tests
  PASS (list in Behavioral Spot-Checks). WR-02 edge found by review
  (rate > 1e9 → zero interval) — Warning, see Anti-Patterns.
- **`internal/app/persist/engine.go`** (86d33a6, +15) — exported `Flush()`
  delegating to the existing `flush` (D-19 shutdown seam, Phase 5's
  consumer). Additive only; `ApplyRemote`/`Records` (the repl.StateStore
  seam) unchanged — signature match re-confirmed. Fresh tests:
  `TestFlush_remoteDirtiness`, `TestFlush_clean`, `TestFlush_closed`,
  `TestEngine_applyRemoteTriggersDump` PASS.
- **`internal/udprepl/replproto/supkv.pb.go`** (ed880c7, 4 lines) — buf
  regeneration of the identical schema (`supkv.proto` untouched since
  9690b4a, Nov 6). Provenance intact: `Code generated by protoc-gen-go. DO
  NOT EDIT.` at line 8, protoc-gen-go v1.36.12-devel, source path correct;
  660 lines, 8-message schema unchanged. `TestCodec_*` family green
  (round-trip property, byte stability, discard classes, truncated
  generation).
- **`Makefile` / `go.mod` / `go.sum`** (ed880c7, 46633f6, a09b8b0) — buf
  codegen targets (`proto`, `buf-lint`, `buf-format`, `buf-breaking` with
  the documented self-destructing guard), grpc/protobuf deps for Phase 5's
  API. No Phase-4 gate weakened: `make lint` and `make test` both exit 0 on
  this tree.
- **Test files** (`repl_test.go` +181, `retransmit_test.go` +101,
  `sync_test.go` +49, `engine_test.go` +97) — all additions are new
  subtests over the new seams (mirror visibility, acked-set honesty,
  sync-arrived flip, Flush contract); no existing Phase-4 assertion
  removed (test families enumerated via `-list`, all 52 named tests
  present; zero skips outside the ratified D-08 smoke classification).

### Required Artifacts

All artifacts across the six plans exist, are substantive, and are wired.
(`gsd-tools query verify.artifacts` returns 0/0 on these plans' string-form
artifact entries — the same parser limitation documented in every previous
pass; manual Level 1-4 verification stands, re-done this pass for every
changed file and regression-checked for the unchanged set.)

| Artifact | Lines | Status | Wiring |
|----------|-------|--------|--------|
| `internal/port/repl.go` | 140 | ✓ VERIFIED | Unchanged; the seam both sides speak (udprepl mention is doc-comment only — port stays stdlib-only per depguard) |
| `internal/app/repl/core.go` | 983 | ✓ VERIFIED | Changed — re-verified at full depth this pass; handler seams all present; no goroutines/locks/clock reads/net |
| `internal/udprepl/repl.go` | 719 | ✓ VERIFIED | Changed — re-verified at full depth this pass; single event loop; atomics mirror |
| `internal/udprepl/replproto/supkv.proto` + `supkv.pb.go` | 74 + 660 | ✓ VERIFIED | .proto unchanged; .pb.go buf-regenerated with provenance marker intact (line 8) |
| `internal/app/repl/{retransmit,peertable,sync,restart,sim_bound,sim_chaos,discard}_test.go` | 715+235+737+330+267+291+412 | ✓ VERIFIED | All green in the full race-suite run; retransmit/sync re-verified fresh (changed) |
| `internal/udprepl/syncsize_internal_test.go` | 149 | ✓ VERIFIED | Unchanged; green |
| `internal/app/internal/simnet/simnet.go` + `simnet_test.go` | 487 + 218 | ✓ VERIFIED | Unchanged (byte-identical); test-only linkage re-proven — imported only from `_test.go` files |
| `internal/app/persist/engine.go` | 587 | ✓ VERIFIED | Changed (additive `Flush`) — re-verified; `ApplyRemote` matches `repl.StateStore` |
| `internal/app/persist/engine_test.go` | 1473 | ✓ VERIFIED | Changed (new Flush tests) — green |
| `internal/udprepl/smoke_test.go` | 331 | ✓ VERIFIED | Unchanged since 507228a (pre-previous-pass); previous full-depth verification stands; fresh PASS this pass |

### Key Link Verification

All key links WIRED; links touching changed files re-verified by direct
code read this pass, the rest regression-checked (files byte-identical):

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| port.Message structs | replproto types | `protoBody`/`portMessage` mapping | ✓ WIRED | pb.go regenerated — schema identical, `TestWire_twoNodeHandoff` + `TestSync_chunkWireBudget` + `TestCodec_*` fresh green |
| udprepl event loop | repl.Core handlers | Only `Run`'s loop calls HandleDatagram/NotifyLocalWrite/Advance | ✓ WIRED | Re-grepped in changed repl.go: lines 270, 282, 294, 298 — the new mirror code adds no second driver |
| NotifyLocalWrite | pending entry + broadcast | Wait-set seeded from peer table, ladder armed, broadcast emitted | ✓ WIRED | Re-read in changed core.go (lines 616-636) |
| inbound Ack | wait-set drain | `handleAck` exact-version drain + sorted-idempotent `ackedInsert` | ✓ WIRED | Re-read in changed core.go (lines 494-518) |
| Advance | retransmit + eviction scans | Single loop fires both, sorted by key | ✓ WIRED | Re-read (scanRetransmit/evictPeers) |
| inbound DataRequest | rate-limited reply | Snapshot lookup → `enqueueResponse` → `drainEgress` | ✓ WIRED | Re-read (lines 532-538, 579-605) |
| startup/timer/restart | chunked listing emission | One `listingChunks` build serves both triggers | ✓ WIRED | Re-read |
| simnet scheduler | Core handlers | `deliver`→HandleDatagram, `drive`→Advance | ✓ WIRED | Unchanged files |
| simnet network | Outbox drain | `route` sends every Outbound; Broadcast fans out incl. sender | ✓ WIRED | Unchanged files |
| per-node store | sim clock | `clock.NewClock(sched.Now, ...)` | ✓ WIRED | Unchanged files |
| observability seams | quiescence | `PendingDeliveries` + `UnackedStats().Count` + `QueuedResponses()` | ✓ WIRED | Accessors re-read in changed core.go |
| persist.Engine.ApplyRemote | repl.StateStore | Signature match; `TestEngine_applyRemoteTriggersDump` fresh PASS | ✓ WIRED | Re-read in changed engine.go |
| adapter atomics | smoke test / Phase-5 readers | `KnownPeers`/`OwnEcho`/`Discards()` + new `SyncArrived`/`PendingCopy` mirrored at the single post-event site | ✓ WIRED | Re-read in changed repl.go; `TestMirror_*` fresh PASS |

### Data-Flow Trace (Level 4)

No rendered/static-value surfaces in this backend phase; the data-flow
equivalents all trace to real sources (changed paths re-traced fresh):

| Path | Source | Status |
|------|--------|--------|
| Replicated records | `store.ApplyRemote` → real `storage.Store.Merge` (LWW join) | ✓ FLOWING |
| DataSync listings | `store.Records()` snapshot per build | ✓ FLOWING |
| Chaos model store | `WriteLog` merged by real `Merge` (simnet unchanged) | ✓ FLOWING |
| Discard/peer/fault counters | Core state mutated by real handler calls; mirrored to atomics post-event | ✓ FLOWING |
| Pending mirror (new) | `core.PendingSnapshot()` cloned per iteration into the published copy-on-write map; drained rows from real Ack correlation | ✓ FLOWING |

### Behavioral Spot-Checks

Single-test discipline; every command run fresh (`-count=1`) this pass. The
full suite ran exactly once (`make test`, exit 0, race detector, all
packages ok).

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Real-socket two-node discovery (REPL-01) | `go test -race -count=1 -run 'TestSmoke_twoNodesUDP' ./internal/udprepl/` | PASS (0.02s) | ✓ PASS |
| Seeded determinism replay (D-06) | `SUPKV_SIM_SEED=4242 go test -race -count=1 -run 'TestChaos_convergence' ./internal/app/repl/` | PASS (0.05s) | ✓ PASS |
| New mirror seams (changed repl.go) | `go test -race -count=1 -run 'TestMirror_pendingAckVisible\|TestMirror_syncArrivedFlip\|TestNotifyLocalWrite_afterRunReturns' ./internal/udprepl/` | PASS (3/3) | ✓ PASS |
| Ack drain / CR-01 error path (changed core.go) | `go test -race -count=1 -run 'TestRetransmit_ackDrainSemantics\|TestRetransmit_suspendAndFlush\|TestRetransmit_backoffLadder' ./internal/app/repl/` | PASS (3/3) | ✓ PASS |
| Rate limiter spacing (backstop truth) | `go test -race -count=1 -run 'TestSync_rateLimit\|TestSync_syncArrived' ./internal/app/repl/` | PASS (2/2) | ✓ PASS |
| Convergence bounds (REPL-10) | `go test -race -count=1 -run 'TestBound_normalOneRTT\|TestBound_partitionHeal\|TestBound_lossWithinAckTimeoutMax' ./internal/app/repl/` | PASS (3/3) | ✓ PASS |
| Discard classification (REPL-08/09) | `go test -race -count=1 -run 'TestDiscard_classes\|TestDiscard_totality' ./internal/app/repl/` | PASS (2/2) | ✓ PASS |
| Restart full sync (REPL-07) | `go test -race -count=1 -run 'TestRestart_runIDChangeTriggersFullSync' ./internal/app/repl/` | PASS | ✓ PASS |
| Wire + cross-seam chunk budget | `go test -race -count=1 -run 'TestWire_twoNodeHandoff\|TestSync_chunkWireBudget' ./internal/udprepl/` | PASS (2/2) | ✓ PASS |
| Persist Flush seam (changed engine.go) | `go test -race -count=1 -run 'TestFlush_remoteDirtiness\|TestFlush_clean\|TestFlush_closed\|TestEngine_applyRemoteTriggersDump' ./internal/app/persist/` | PASS (4/4) | ✓ PASS |
| Full workspace, race detector | `make test` (`go test -race -cover ./...`) | exit 0, all packages ok | ✓ PASS |
| Strict lint gate | `make lint` (config verify + golangci-lint 2.15.0 strict) | exit 0, zero findings | ✓ PASS |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` exist or are referenced by
any PLAN/SUMMARY. The plans' numbered "probe #N" items are pinned research
edge cases discharged by the named Go tests, not executable scripts
(unchanged finding).

### Requirements Coverage

Plan frontmatter union: REPL-01,02,03,04,05,06,07,08,09,10 + TEST-03 —
exactly the 11 IDs REQUIREMENTS.md assigns to Phase 4 (all marked Complete
in its traceability table). No orphaned requirements, no unclaimed IDs.

| Requirement | Source Plan | Status | Evidence |
|-------------|------------|--------|----------|
| REPL-01 | 04-06 | ✓ SATISFIED | `TestSmoke_twoNodesUDP` fresh PASS (real sockets, zero config; no peer-list/static-address surface anywhere in adapter or Config) |
| REPL-02 | 04-02 | ✓ SATISFIED | Tracking/backoff/suspension/flush re-read in changed core.go; 3 named tests fresh PASS |
| REPL-03 | 04-01 | ✓ SATISFIED | `TestWire_twoNodeHandoff` fresh PASS; Ack-after-apply re-read (applyRemote) |
| REPL-04 | 04-04 | ✓ SATISFIED | Unknown key → zero outbound re-read (handleDataRequest); `TestDataRequest_replyAndSilence` green in full suite |
| REPL-05 | 04-03 | ✓ SATISFIED | Chunked listing + limiter re-read; `TestSync_rateLimit` fresh PASS; IDEA.md §Parameters echo intact (line 147) |
| REPL-06 | 04-02 | ✓ SATISFIED | Peer table lifecycle re-read (evictPeers at-or-after boundary, no blacklist, unbounded map); `TestPeerTable_lifecycle` fresh PASS |
| REPL-07 | 04-04 | ✓ SATISFIED | `TestRestart_runIDChangeTriggersFullSync` fresh PASS; upsert-by-node_id re-read |
| REPL-08 | 04-06 | ✓ SATISFIED | Both duplicate variants dropped+logged+counted (classifier re-read); `TestDiscard_classes` fresh PASS |
| REPL-09 | 04-06 | ✓ SATISFIED | Five distinguishable classes + totality; `TestDiscard_totality` (500 adversarial datagrams) fresh PASS; own-echo counted in fresh smoke run |
| REPL-10 | 04-05 | ✓ SATISFIED | All three `TestBound_*` fresh PASS |
| TEST-03 | 04-05 | ✓ SATISFIED | Chaos convergence + determinism — simnet unchanged; fresh seeded replay PASS |

### Anti-Patterns Found

Debt-marker gate: zero `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`
across all covered files (changed and unchanged), re-grepped this pass.
Zero skip patterns outside the ratified D-08 smoke loud-skip. The two
Warnings below are the fresh incremental review's (04-REVIEW.md,
2026-11-14) open findings, re-confirmed in code by this pass; neither
breaks any Phase-4 truth, artifact, or link — both live in Phase-5-facing
seams and are tracked open in 04-REVIEW-DISPOSITION.md.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/udprepl/repl.go | 513-514 | `retainDrained` union `clone(prev.Acked)+prev.WaitingOn` sorted without `slices.Compact` — duplicate node id in a published drained row after an Ack-in-suspension + discovery-flush sequence (review WR-01) | ⚠️ Warning | Mirror observability only — over-reports `len(Acked)` by the duplicate for a wait_for_replication consumer (bounded by the af3d5c4 clamp); the core's pending set still drains correctly; open in 04-REVIEW-DISPOSITION.md |
| internal/app/repl/core.go | 122-124, 285 | `normalized()` guards only `SyncResponseRate <= 0`; integer `time.Second/rate` → 0 for rate > 1e9, disabling limiter spacing (review WR-02) | ⚠️ Warning | Config-domain edge: requires an operator to set a nonsensical >1e9/s rate (default 500; documented envelope far below); `TestSync_rateLimit` proves the limiter in-domain; open in 04-REVIEW-DISPOSITION.md |
| internal/udprepl/repl.go | 327-338 | `NotifyLocalWrite` done-channel pre-check leaves a narrow TOCTOU window during Run's exit (review IN-01) | ℹ️ Info | Bounded (write persists locally, reconciles after restart); doc contract holds literally; open by triage |
| internal/app/persist/engine.go | 491-498 | `Restore` marks dirty unconditionally → one redundant covering dump on an all-skip replay (review IN-02) | ℹ️ Info | Redundant save only, no data effect; open by triage |
| internal/app/repl/core.go | 234 | `ownHandle` write-only field (prior review IN-01) | ℹ️ Info | Carried unchanged from previous pass; dead state, no behavioral effect |
| internal/app/internal/simnet/simnet.go | 459 | Cluster.Write discards the `Set` error (prior review IN-02) | ℹ️ Info | Carried unchanged; test-harness-only path |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/udprepl/smoke_test.go | REPL-01, REPL-09 | yes (Linux run; macOS loud-skip by ratified D-08 design) | conditional-by-design | no | Behavioral (mutual discovery, byte-equal visibility, OwnEcho counter) | VALID |
| internal/udprepl/repl_test.go | REPL-02 (mirror edges) | yes | 0 | no | Behavioral (held-snapshot stability, acked visibility, post-Run sentinel) | VALID |
| internal/app/repl/sim_chaos_test.go | TEST-03, REPL-10 | yes | 0 | no (expected values from the supremum model, not the SUT's store dump) | Behavioral (convergence + byte-identical replay) | VALID |
| internal/app/repl/{retransmit,peertable,sync,restart,discard}_test.go | REPL-02..09 | yes | 0 | no | Value/Behavioral | VALID |
| internal/app/persist/engine_test.go | REPL-03 (persist seam) | yes | 0 | no | Behavioral (dump timing through synctest; Flush contract) | VALID |

**Disabled tests on requirements:** 0. **Circular patterns detected:** 0.
**Insufficient assertions:** 0.

### Decision Coverage

8/8 trackable CONTEXT.md decisions honored by shipped artifacts
(`check.decision-coverage-verify`: `total: 8, honored: 8, not_honored: []` —
re-run this pass, matches both previous reports).

### Prohibition Disposition (ADR-550 D4)

| Prohibition | Tier | Verdict |
|-------------|------|---------|
| REPL-01 (04-01, 04-06): no config surface beyond broadcast_addr | test | ENFORCED — adapter `New` signature and `Config` struct re-read on the changed files: timing knobs + Log only, no peer lists/static addressing anywhere |
| REPL-08 (04-06): duplicate node_id never silently resolved | test | ENFORCED — both variants drop + log + count; `TestDiscard_classes` fresh PASS |
| 04-02: pending never dropped on timeout/eviction/empty table | test | ENFORCED — only Ack/supersession delete (re-read); `TestRetransmit_suspendAndFlush` fresh PASS |
| 04-02/D-05: retransmission never blocks a local write | test | ENFORCED — enqueue-and-return; `TestNotifyLocalWrite_afterRunReturns` fresh PASS |
| 04-02/REPL-06: no blacklist, no peer-count cap | test | ENFORCED — evicted peers re-learned; unbounded node_id-keyed map (re-read) |
| 04-03: no bypass of the rate-limited egress queue | test | ENFORCED — exactly three producer sites call `enqueueResponse` (re-read) |
| 04-03/A4: chunks carry no continuation state | test | ENFORCED — self-sufficient slices (re-read) |
| 04-03/REPL-04: replies unicast only | test | ENFORCED — unicast/queued unicast only (re-read) |
| 04-04: no second peer row on run_id change | test | ENFORCED — upsert keyed by node_id; `TestRestart_staleRunIdAndInFlight` green |
| 04-04: full-sync reaction only on run_id change | test | ENFORCED — inequality is the only trigger; `TestRestart_newPeerNoFullSync` green |
| 04-05: no second protocol implementation in simnet | test | ENFORCED — real Core handlers; no non-test importer (re-grepped: simnet_test.go, sim_bound_test.go, sim_chaos_test.go only) |
| 04-05: no wall-clock dependence | test | ENFORCED — no `time.Sleep`/`time.Now` in core.go or simnet.go (re-grepped) |
| 04-05/D-06: sim tooling not in production binary | test | ENFORCED — simnet imported only from `_test.go` files (re-grepped) |
| REPL-09 CRC wording (04-01, 04-06) | judgment | **RATIFIED — resolved** (04-UAT.md item 1, human ratification 2026-11-06/07). Regression grep this pass: every production mention still says corruption detection only, never tamper protection (supkv.pb.go:33, udprepl/repl.go:570, storage/store.go:122, storage/doc.go:35); test files use "tamper" only as the attack verb. No flagged prohibitions remain |

### Human Verification Required

None outstanding. Infrastructure/foundation phase — no user-facing
elements; all acceptance criteria verifiable programmatically. The previous
report's two items remain resolved with recorded evidence (reused per the
re-verification contract): the CRC-wording judgment prohibition is ratified
by human UAT (04-UAT.md, status complete) with the wording re-grepped
unchanged, and D-08's macOS smoke outcome is empirically settled by CI
(post-fix run 42514571899, `gate (macos-15)` success). No
PRESENT_BEHAVIOR_UNVERIFIED truths — every behavior-dependent truth has a
fresh passing named test this pass.

### Advisory (New Scope, Unevidenced)

Re-verification ran; no Step-7 finding was downgraded from blocker. The two
fresh-review Warnings (WR-01 drained-row duplicate, WR-02 rate > 1e9
limiter disable) are classified ⚠️ Warning — they do not fail any
must-have truth, artifact, or key link (both sit in Phase-5-facing seams,
are bounded, and are tracked open in 04-REVIEW-DISPOSITION.md with
concrete fixes proposed there).

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None downgraded | — | — |

### Gaps Summary

None. The staleness trigger was Phase 5's additive work on Phase-4 files:
the copy-on-write pending mirror and drained-row retention (repl.go), the
sync-arrived state and `PendingInfo.Acked` honesty seam (core.go), the
exported persist `Flush`, the buf regeneration of an identical proto
schema, and buf/grpc build-graph changes. Every changed file was
re-verified at all four levels with fresh named tests (22 individual test
runs plus one full race-suite run and the strict lint gate — all green);
every unchanged covered file is byte-identical to what previous passes
verified. The Phase-4 contract is intact on the current tree: the core's
protocol behavior (drain, suspension, ladder, reconciliation, discard
classification, convergence bounds) is unchanged and freshly proven, the
single-driver discipline survived the new mirror code, the regenerated
wire schema is byte-equivalent with provenance intact, and the new
`Flush`/mirror surfaces are additive. The two open review Warnings are
recorded above with their disposition; they do not block the goal. The
phase goal — convergence to the pointwise supremum over a lossy UDP
segment, proven under chaos by the deterministic simulator — remains
achieved in the codebase. Status is `passed`: no `gaps:` entries, no
outstanding human-verification items, no flagged prohibitions.

---

_Verified: 2026-11-14T19:45:24Z_
_Verifier: Claude (gsd-verifier)_
