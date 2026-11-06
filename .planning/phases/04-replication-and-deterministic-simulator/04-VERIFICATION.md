---
phase: 04-replication-and-deterministic-simulator
verified: 2026-11-06T20:47:03Z
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
  - docs/IDEA.md
  - go.mod
  - go.sum
  - internal/app/internal/simnet/simnet.go
  - internal/app/internal/simnet/simnet_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/core.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/peertable_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_bound_test.go
  - internal/app/repl/sim_chaos_test.go
  - internal/app/repl/sync_test.go
  - internal/port/doc.go
  - internal/port/repl.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replproto/supkv.pb.go
  - internal/udprepl/replproto/supkv.proto
  - internal/udprepl/smoke_test.go
  - internal/udprepl/syncsize_internal_test.go
covered_digest: "v4:sha256:85c4b26f21d70102f5299d98482ce41c556d704cf84345e88a132fed6c67e236"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Ratify the judgment-tier prohibition (2 entries, plans 04-01 and 04-06): CRC32 is never described as tamper protection in code comments, logs, or docs — corruption detection only per the accepted trust model."
    expected: "A human confirms the natural-language framing is honest everywhere CRC32 is mentioned (grep evidence below covers all mentions; the LLM-judge verdict is non-authoritative in autonomous verification)."
    why_human: "Judgment-tier prohibition (ADR-550 D3): the check is a human judgment over natural-language framing, not a mechanical pattern. Autonomous verification records a NON-AUTHORITATIVE LLM-judge verdict (PASS) plus this flag — it must not be silently absorbed into a passed verdict."
  - test: "Record D-08's macOS smoke outcome when the first macOS CI gate runs (Phase 5+ CI contexts)."
    expected: "On macOS the smoke test loud-skips after its broadcast-reachability probe (loopback broadcast fan-out is Linux-only); the skip message names the platform limitation and points to the simulator for protocol correctness. First macOS CI run should show the documented skip, not a failure."
    why_human: "Empirical platform behavior — no macOS runner is reachable from this verification environment. Code inspection confirms the loud-skip path (smoke_test.go:219-239 probes first, then t.Skipf with the Pitfall 3 explanation)."
---

# Phase 4: Replication and Deterministic Simulator Verification Report

**Phase Goal:** Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment, with convergence under chaos proven by the deterministic simulator
**Verified:** 2026-11-06T20:47:03Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

The five ROADMAP success criteria are the contract; the 38 plan-level truths
from the six PLANs are refinements of these five (each plan truth maps to
exactly one SC). All plan-level truths were verified against code plus named
passing tests — the full detail is in the tables below.

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|-----------------------------------|--------|----------|
| 1 | Two (or N) nodes discover each other via broadcast Hello with no configuration beyond `broadcast_addr`, and a write on one node becomes visible on every other within one RTT in normal operation | ✓ VERIFIED | `TestSmoke_twoNodesUDP` PASS on real Linux UDP sockets (mutual discovery `KnownPeers()==1` both sides, OwnEcho counted — smoke_test.go:263,287); `TestBound_normalOneRTT` PASS with teeth probe at t0; `TestWire_twoNodeHandoff` PASS end-to-end through the port seam. Zero-config verified: no peer lists, no static addressing anywhere (adapter `New` takes only bind/broadcast/nodeID/store/cfg) |
| 2 | Every local change is broadcast and tracked per-peer (key, version) until Ack or supersession, retransmitting with exponential backoff and suspending (not dropping) while no peer is known; receivers Ack after applying to memory; the unacked set drains after a partition heals | ✓ VERIFIED | core.go: `NotifyLocalWrite` (wait-set, ladder at AckTimeout), `scanRetransmit` (ladder 200/400/800/1600/3200ms→5s cap, sorted-key emission, suspension on empty table), `handleAck` (exact-version drain), `handleHello` (flush-on-discovery). Tests PASS: `TestRetransmit_backoffLadder`, `TestRetransmit_suspendAndFlush`, `TestRetransmit_ackDrainSemantics` (7 subtests incl. both CR-01 regression subtests), `TestPeerTable_lifecycle/_nPeers/_solePeer`. Post-partition drain proven by `TestChaos_convergence` quiescence (UnackedStats().Count==0 on all cores with partitions injected, PASS) |
| 3 | Full reconciliation works in both directions: chunked DataSync on the sync timer, at startup and on neighbor restart; receivers answer with DataRequest or unicast DataReplication rate-limited by `sync_response_rate`; a DataRequest for an unknown key gets no reply; a Hello with a new `run_id` triggers full sync toward the restarted node | ✓ VERIFIED | core.go: `syncListing` (startup + timer triggers), `handleHello` restart branch (targeted unicast chunk sequence), `handleDataSync` (snapshot compare, request-newer/silence), `handleDataRequest` (reply-or-silence), `enqueueResponse`/`drainEgress` (FIFO limiter). Tests PASS: `TestSync_triggers/_chunking/_compare/_rateLimit`, `TestDataRequest_replyAndSilence`, `TestRestart_runIDChangeTriggersFullSync/_newPeerNoFullSync/_staleRunIdAndInFlight` |
| 4 | Malformed traffic is dropped with distinguishable classes: duplicate `node_id` logged as a configuration error, plus own broadcast echo, bad CRC and bad proto version each identifiable | ✓ VERIFIED | core.go `HandleDatagram` classification order pinned; both duplicate variants (own-id-foreign-run, known-id-second-handle-same-run) drop + log config error + count `DuplicateNodeID`; different run_id routes to restart, never duplicate. Tests PASS: `TestDiscard_classes` (8 subtests incl. both variants), `TestDiscard_totality` (500 seeded adversarial datagrams, counter conservation, zero panics) |
| 5 | The deterministic simulator runs 2..N nodes over a seeded virtual network that loses, duplicates, reorders and partitions datagrams, and asserts convergence within the documented bounds — normal one RTT, datagram loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval` + response time — printing the seed on failure | ✓ VERIFIED | simnet.go: discrete-event (Time,Seq) scheduler, send-time fate draws from one injected rng, fault stats with all four classes; drives the production `repl.Core` handlers (no second implementation — verified no simnet import outside _test.go). Tests PASS: `TestBound_normalOneRTT`, `TestBound_lossWithinAckTimeoutMax`, `TestBound_partitionHeal` (all inclusive with negative teeth probes), `TestChaos_convergence` (2..5 nodes, 40 writes, partitions; converges to pointwise-supremum model; all four fault classes observed non-zero; seed + replay command printed every run), `TestChaos_determinismReplay` (byte-identical traces). `SUPKV_SIM_SEED=4242` replay verified live |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Plan-Level Truth Detail (all 38 verified)

| Plan | Truths | Key evidence (all tests run and PASS by this verifier) |
|------|--------|--------------------------------------------------------|
| 04-01 (REPL-03) | 6/6 | `TestWire_twoNodeHandoff` (encode→CRC→decode→apply→Ack through the port seam); no goroutines/locks/wall-clock in `internal/app/repl` (grep clean); discard decisions in core, adapter CRC-before-body-decode (repl.go:415); peer table keyed by node_id with opaque string handles, no `net` import in app/port |
| 04-02 (REPL-02/06) | 8/8 | Ladder delays documented at `scanRetransmit` doc (200/400/800/1600/3200→5000ms cap, saturate-never-wrap); suspension retains entries; flush-on-discovery adds waiter + restarts ladder; eviction at-or-after boundary both sides asserted (`TestPeerTable_lifecycle`); `PendingSnapshot`/`UnackedStats` seams present |
| 04-03 (REPL-04/05) | 8/8 | Unknown-key DataRequest → zero outbound (verified code + `TestDataRequest_replyAndSilence`); tombstones replicate through listing/replies (tombstones included in `Records()` snapshot paths); deterministic sorted chunking (`slices.Sorted(maps.Keys(...))`, never map order); exact-budget chunk closure + oversized-single-entry (probe #15 subtests); empty-store listing still broadcast (probe #7); **backstop truth** (integer-division spacing) — explicit wired evidence: `TestSync_rateLimit` default 500/s row (first immediate, one per 2ms, listing order) AND 300/s row pinning floor division at 3333333ns (non-even divisor); echoed in docs/IDEA.md:147 |
| 04-04 (REPL-07) | 4/4 | Restart: in-place row update + unicast full listing through the shared `listingChunks` (never a second listing implementation); same-run_id Hello → lastHello refresh only; new node_id learned without listing; no re-add to wait-sets, no ladder touch. `TestRestart_*` ×3 PASS |
| 04-05 (TEST-03/REPL-10) | 6/6 | Simulator single-threaded deterministic (no `go func`/`sync.`/`time.Sleep` in core.go or simnet.go — grep clean); same handlers as production (simnet test-only, verified no non-test importer); same-seed byte-identical digests; bounds on virtual clock with response term from exact limiter arithmetic; chaos quiescence via the three seams + supremum-model deep-equality with fault-class teeth |
| 04-06 (REPL-01/08/09) | 6/6 | Five discard classes distinguishable and total (500 adversarial datagrams, conservation, zero panics); zero-length/garbage → Undecodable; both duplicate variants dropped AND surfaced (config-error line through `Config.Log` naming node_id and both handles + counter); real-socket discovery zero-config; smoke never asserts Ack/unicast between same-host REUSEADDR sockets (D-08 scope pinned); `Engine.ApplyRemote` merges + marks dirty with no waiter, oversize passes through (`TestEngine_applyRemoteTriggersDump` PASS, asserts return strictly before the debounced dump) |

### Verifier Ratifications (flagged assumptions — evidence-based, recorded)

- **Assumption #17 (crossing first Hellos):** RATIFIED. `handleHello`'s
  discovery/restart discrimination is order-independent by construction
  (upsert keyed by node_id; discovery = empty-table event, restart requires a
  remembered row). Simultaneous starts with crossing first Hellos are
  exercised live by the smoke test (both adapters race their startup Hellos)
  and by every seeded chaos run (all nodes' first timer pumps at t=0) —
  `TestChaos_convergence` and `TestChaos_determinismReplay` PASS.
- **Assumption #20 (duplicate-node_id split):** RATIFIED. Both variants land
  in `DuplicateNodeID` with a config-error log (core.go:315-334); the second
  variant requires run_id EQUALITY with the remembered row so a differing
  run_id always reaches the restart branch. Both variants are pinned by
  named subtests in `TestDiscard_classes` (PASS) and the restart-vs-duplicate
  discrimination by `TestRestart_runIDChangeTriggersFullSync` (PASS).

### Required Artifacts

All artifacts across the six plans exist, are substantive, and are wired.
`gsd-tools query verify.artifacts` could not parse these plans' string-form
artifact entries (returned 0/0), so Level 1-4 verification was performed
manually:

| Artifact | Lines | Status | Wiring |
|----------|-------|--------|--------|
| `internal/port/repl.go` | 140 | ✓ VERIFIED | Imported by core.go, udprepl/repl.go, simnet — the seam both sides speak |
| `internal/app/repl/core.go` | 911 | ✓ VERIFIED | Driven by udprepl event loop (production) and simnet (tests); no goroutines/locks/clock reads |
| `internal/udprepl/repl.go` | 520 | ✓ VERIFIED | Only net/protobuf replication code; single event loop; atomics mirror |
| `internal/udprepl/replproto/supkv.proto` + `supkv.pb.go` | 74 + 660 | ✓ VERIFIED | Envelope + Body oneof, all five messages; committed protoc-gen-go v1.36.12 output (== go.mod tool directive) |
| `internal/app/repl/retransmit_test.go` | 616 | ✓ VERIFIED | 7 ack-drain subtests incl. both CR-01 regressions — PASS |
| `internal/app/repl/peertable_test.go` | 235 | ✓ VERIFIED | Lifecycle/n-peers/sole-peer — PASS |
| `internal/app/repl/sync_test.go` | 688 | ✓ VERIFIED | Triggers/chunking/compare/rateLimit — PASS |
| `internal/udprepl/syncsize_internal_test.go` | 149 | ✓ VERIFIED | Cross-seam wire-budget proof (`TestSync_chunkWireBudget` PASS) |
| `internal/app/repl/restart_test.go` | 330 | ✓ VERIFIED | 3 restart tests — PASS |
| `internal/app/internal/simnet/simnet.go` + `simnet_test.go` | 487 + 218 | ✓ VERIFIED | Test-only (verified: no importer outside `_test.go` — D-06 prohibition holds); drives real Core handlers |
| `internal/app/repl/sim_bound_test.go` | 267 | ✓ VERIFIED | 3 bound tests — PASS |
| `internal/app/repl/sim_chaos_test.go` | 291 | ✓ VERIFIED | Chaos + determinism — PASS |
| `internal/app/repl/discard_test.go` | 412 | ✓ VERIFIED | Classes + 500-datagram totality — PASS |
| `internal/app/persist/engine.go` | 572 | ✓ VERIFIED | `ApplyRemote` (line 424) + `Records` (line 443) match `repl.StateStore` signatures exactly — Engine satisfies the interface structurally |
| `internal/app/persist/engine_test.go` | 1376 | ✓ VERIFIED | `TestEngine_applyRemoteTriggersDump` — PASS |
| `internal/udprepl/smoke_test.go` | 290 | ✓ VERIFIED | Real-socket two-node discovery — PASS on this Linux host |

### Key Link Verification

All key links verified manually (gsd-tools returned 0/0 for these plans'
string-form entries):

| From | To | Via | Status |
|------|----|-----|--------|
| port.Message structs | replproto types | `protoBody`/`portMessage` map every field of every message; cross-seam proven by `TestWire_twoNodeHandoff` + `TestSync_chunkWireBudget` (estimate constants pinned against the real codec) | ✓ WIRED |
| udprepl event loop | repl.Core handlers | Only `Run`'s loop calls HandleDatagram/NotifyLocalWrite/Advance; readLoop and authors feed queues | ✓ WIRED |
| NotifyLocalWrite | pending entry + broadcast | Wait-set seeded from peer table, ladder armed, broadcast emitted | ✓ WIRED |
| inbound Ack | wait-set drain | `handleAck` exact-version drain with empty-table guard | ✓ WIRED |
| Advance | retransmit + eviction scans | Single loop fires both, sorted-key emission, at-or-after eviction | ✓ WIRED |
| inbound DataRequest | rate-limited reply | Snapshot lookup → `enqueueResponse` → `drainEgress` in Advance | ✓ WIRED |
| startup/timer/restart | chunked listing emission | One `listingChunks` build serves broadcast timer and restart unicast | ✓ WIRED |
| simnet scheduler | Core handlers | `deliver`→HandleDatagram, `drive`→Advance loop with generation-countered timers | ✓ WIRED |
| simnet network | Outbox drain | `route` sends every Outbound; Broadcast fans out including sender (own echo real) | ✓ WIRED |
| per-node store | sim clock | `clock.NewClock(sched.Now, ...)` — virtual time through the D-02 seam | ✓ WIRED |
| observability seams | quiescence | `PendingDeliveries` + `UnackedStats().Count` + `QueuedResponses` — the chaos termination signals | ✓ WIRED |
| persist.Engine.ApplyRemote | repl.StateStore | Signatures match exactly; exercised by `TestEngine_applyRemoteTriggersDump`; gRPC handler consumption is Phase 5 scope | ✓ WIRED |
| adapter atomics | smoke test | `mirrorStats` after each iteration; smoke asserts `KnownPeers`/`OwnEcho` through them | ✓ WIRED |

### Data-Flow Trace (Level 4)

No rendered/static-value surfaces in this backend phase; the data-flow
equivalents all trace to real sources:

| Path | Source | Status |
|------|--------|--------|
| Replicated records | `store.ApplyRemote` → real `storage.Store.Merge` (LWW join) | ✓ FLOWING |
| DataSync listings | `store.Records()` snapshot per build | ✓ FLOWING |
| Chaos model store | `WriteLog` (captured production author sequence) merged by real `Merge` | ✓ FLOWING |
| Discard/peer/fault counters | Core/simnet state mutated by real handler calls; mirrored to atomics | ✓ FLOWING |

### Behavioral Spot-Checks

Every behavior-dependent truth has a passing named test (single-test
discipline; full suite run exactly once at the end):

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Chaos convergence to pointwise supremum | `go test ./internal/app/repl/ -run '^TestChaos_convergence$'` | PASS (0.06s; seed + replay printed; all 4 fault classes non-zero) | ✓ PASS |
| Same-seed determinism | `go test ./internal/app/repl/ -run '^TestChaos_determinismReplay$'` | PASS (byte-identical digests) | ✓ PASS |
| Three convergence bounds | `go test ./internal/app/repl/ -run '^TestBound_'` | 3/3 PASS with negative teeth probes | ✓ PASS |
| CR-01 regression (no false Ack on rejected apply) | `go test ./internal/app/repl/ -run '^TestRetransmit_ackDrainSemantics$'` | 7/7 subtests PASS incl. store-rejected and duplicate-redelivery cases | ✓ PASS |
| Discard totality (500 adversarial datagrams) | `go test ./internal/app/repl/ -run '^TestDiscard_totality$'` | PASS | ✓ PASS |
| Real-socket two-node discovery (REPL-01) | `go test ./internal/udprepl/ -run '^TestSmoke_twoNodesUDP$'` | PASS on this Linux host (0.02s) | ✓ PASS |
| WR-01 fix (post-Run NotifyLocalWrite) | `go test ./internal/udprepl/ -run '^TestNotifyLocalWrite_afterRunReturns$'` | PASS | ✓ PASS |
| Seed replay hook (D-06) | `SUPKV_SIM_SEED=4242 go test ./internal/app/repl/ -run '^TestChaos_convergence$'` | ok | ✓ PASS |
| Persist remote-apply seam | `go test ./internal/app/persist/ -run '^TestEngine_applyRemoteTriggersDump$'` | PASS | ✓ PASS |
| Cross-seam wire budget | `go test ./internal/udprepl/ -run '^TestSync_chunkWireBudget$'` | PASS | ✓ PASS |
| Full workspace, race detector | `go test -race ./...` | all packages ok | ✓ PASS |
| Strict lint gate | `make lint` | exit 0, zero findings | ✓ PASS |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` exist or are referenced by
any PLAN/SUMMARY. The plans' numbered "probe #N" items are pinned research
edge cases discharged by the named Go tests above, not executable scripts.

### Requirements Coverage

Plan frontmatter union: REPL-01,02,03,04,05,06,07,08,09,10 + TEST-03 —
exactly the 11 IDs the ROADMAP assigns to Phase 4. No orphaned requirements,
no unclaimed IDs.

| Requirement | Source Plan | Status | Evidence |
|-------------|------------|--------|----------|
| REPL-01 | 04-06 | ✓ SATISFIED | `TestSmoke_twoNodesUDP` PASS (real sockets, zero config); no peer lists/static addressing anywhere |
| REPL-02 | 04-02 | ✓ SATISFIED | Tracking/backoff/suspension/flush verified in code + 3 passing test families |
| REPL-03 | 04-01 | ✓ SATISFIED | `TestWire_twoNodeHandoff` + Ack-after-apply in `applyRemote`; CR-01 fix gates Ack on successful apply |
| REPL-04 | 04-03 | ✓ SATISFIED | `TestDataRequest_replyAndSilence` PASS (unknown key → zero outbound) |
| REPL-05 | 04-03 | ✓ SATISFIED | `TestSync_*` ×4 PASS; limiter scope pinned (three producer sites only); IDEA.md:147 echo |
| REPL-06 | 04-06→04-02 | ✓ SATISFIED | `TestPeerTable_*` PASS; N-peers map, at-or-after eviction, no blacklist/cap |
| REPL-07 | 04-04 | ✓ SATISFIED | `TestRestart_*` ×3 PASS; in-place row update, targeted unicast listing |
| REPL-08 | 04-06 | ✓ SATISFIED | Both variants dropped + logged + counted (`TestDiscard_classes` PASS) |
| REPL-09 | 04-01, 04-06 | ✓ SATISFIED | Five distinguishable classes; totality over 500 adversarial datagrams (`TestDiscard_totality` PASS) |
| REPL-10 | 04-05 | ✓ SATISFIED | Three bound tests PASS on the virtual clock, inclusive with teeth probes |
| TEST-03 | 04-05 | ✓ SATISFIED | Seeded 2..5-node sim, drop/dup/reorder/partition observed non-zero, chaos convergence + determinism PASS |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/app/internal/simnet/simnet.go | 459 | `_, _, _ = store.Set(...)` — Write discards the Set error (review IN-02) | ℹ️ Info | Open by documented triage (04-REVIEW-FIX.md); test-harness-only path, unreachable with current scenario payloads (≤ 24-byte keys, small payloads far under the 61440 limit) |
| internal/app/repl/core.go | 229 | `ownHandle` write-only field (review IN-01) | ℹ️ Info | Open by documented triage; dead state, no behavioral effect |

Debt-marker gate: zero `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` in all
phase files (production and test). No placeholder/empty-impl strings, no
stray prints in production code.

### Decision Coverage

8/8 trackable CONTEXT.md decisions honored by shipped artifacts
(`check.decision-coverage-verify`: `total: 8, honored: 8, not_honored: []`).

### Prohibition Disposition (ADR-550 D4)

| Prohibition | Tier | Verdict |
|-------------|------|---------|
| REPL-01 (04-01, 04-06): no config surface beyond broadcast_addr | test | ENFORCED — no peer lists/static addressing exist; smoke test proves zero-config discovery on real sockets |
| REPL-08 (04-06): duplicate node_id never silently resolved | test | ENFORCED — both variants drop + log + count; wired enforcement in `TestDiscard_classes` |
| 04-02: pending never dropped on timeout/eviction/empty table | test | ENFORCED — code retains entries (only Ack/supersession delete); wired enforcement in `TestRetransmit_suspendAndFlush` |
| 04-02/D-05: retransmission never blocks a local write | test | ENFORCED — `NotifyLocalWrite` enqueues-and-returns; adapter returns `error` post-Run instead of blocking (`TestNotifyLocalWrite_afterRunReturns`) |
| 04-02/REPL-06: no blacklist, no peer-count cap | test | ENFORCED — evicted peers re-learned by next Hello; map keyed by node_id unbounded; `TestPeerTable_nPeers` |
| 04-03: no bypass of the rate-limited egress queue | test | ENFORCED — exactly three producer sites call `enqueueResponse`; restart/timer listings are proactive by the pinned scope |
| 04-03/A4: chunks carry no continuation state | test | ENFORCED — `chunkSyncEntries` emits self-sufficient slices; any chunk comparable alone |
| 04-03/REPL-04: replies unicast only | test | ENFORCED — `out.Unicast`/queued unicast only |
| 04-04: no second peer row on run_id change | test | ENFORCED — upsert keyed by node_id; `TestRestart_staleRunIdAndInFlight` |
| 04-04: full-sync reaction only on run_id change | test | ENFORCED — inequality is the only trigger; `TestRestart_newPeerNoFullSync` |
| 04-05: no second protocol implementation in simnet | test | ENFORCED — simnet routes to real Core handlers; no non-test importer |
| 04-05: no wall-clock dependence | test | ENFORCED — no `time.Sleep` in decision path; tests run in milliseconds of wall time |
| 04-05/D-06: sim tooling not in production binary | test | ENFORCED — simnet imported only from `_test.go` files (grep-verified) |
| REPL-09 CRC wording (04-01, 04-06) | judgment | **unverified-prohibition — human review recommended.** LLM-judge verdict PASS (non-authoritative): all 6 mention sites say "corruption detection only, never tamper protection" (supkv.proto:15, supkv.pb.go:33, udprepl/repl.go:371, storage/store.go:103, storage/doc.go:35; codec_test.go uses "tamper" as an attack verb in a test name, not a CRC claim). Routed to Human Verification below |

### Human Verification Required

1. **Ratify the judgment-tier CRC-wording prohibition (plans 04-01 and 04-06)**
   - **Test:** Confirm CRC32 is never framed as tamper protection in comments, logs, or docs.
   - **Expected:** All mention sites describe corruption detection only (evidence above lists every site; HMAC documented as future work per the trust model).
   - **Why human:** Judgment-tier prohibition — the ADR-550 autonomous-mode verdict is non-authoritative by design; a human owns the natural-language framing call.

2. **D-08 macOS smoke outcome (empirical)**
   - **Test:** Observe `TestSmoke_twoNodesUDP` on the first macOS CI run (Phase 5+ CI contexts).
   - **Expected:** The documented loud skip after the broadcast-reachability probe (loopback broadcast fan-out is Linux-only), not a failure.
   - **Why human:** Platform behavior requires a macOS runner; this environment is Linux-only.

### Gaps Summary

None. All five success criteria, all 38 plan-level truths, all artifacts
(Levels 1-4), and all key links are verified with passing behavioral tests —
the phase goal (convergence to the pointwise supremum over a lossy UDP
segment, proven under chaos by the deterministic simulator) is achieved in
the codebase, not merely claimed. The two code-review fixes (CR-01 commit
957fd0a, WR-01 commit 58cd6c7) are present in code with regression tests
that pass. Full `go test -race ./...` and `make lint` are green as re-run by
this verifier. The status is `human_needed` solely for the two ratification
items above (judgment-tier prohibition + empirical macOS outcome); no
`gaps:` entries exist and no gap-closure planning is required.

---

_Verified: 2026-11-06T20:47:03Z_
_Verifier: Claude (gsd-verifier)_
