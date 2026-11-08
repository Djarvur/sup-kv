---
phase: 04-replication-and-deterministic-simulator
verified: 2026-11-08T21:09:31Z
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
covered_digest: "v4:sha256:b9ffd02ef7e6849925d3e0d2793a3deb8d3bf9b422e793e10542787ce8e1e22c"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 5/5
  gaps_closed: [] # staleness regeneration, not gap closure — covered source (smoke_test.go) changed after the last verified timestamp
  gaps_remaining: []
  regressions: []
---

# Phase 4: Replication and Deterministic Simulator Verification Report

**Phase Goal:** Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment, with convergence under chaos proven by the deterministic simulator
**Verified:** 2026-11-08T21:09:31Z
**Status:** passed
**Re-verification:** Yes — staleness regeneration. The previous report (verified 2026-11-06T20:47:03Z, digest `85c4b26f…`) went stale when `internal/udprepl/smoke_test.go` changed after it was written: commit 507228a (2026-11-08, D-08 macOS smoke fix) plus planning-doc commits. `git log --since` confirms **smoke_test.go is the only implementation file changed** since the previous pass; this re-verification therefore re-verifies that file at full depth and regression-checks everything else.

## Goal Achievement

### Observable Truths

The five ROADMAP success criteria are the contract; the 38 plan-level truths
from the six PLANs are refinements of these five (each plan truth maps to
exactly one SC). All plan-level truths were verified against code plus named
passing tests in the previous pass; this pass regression-checked them (the
underlying files are byte-identical — only smoke_test.go changed) and
re-ran the full suite plus the changed-area and determinism spot-checks.

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|-----------------------------------|--------|----------|
| 1 | Two (or N) nodes discover each other via broadcast Hello with no configuration beyond `broadcast_addr`, and a write on one node becomes visible on every other within one RTT in normal operation | ✓ VERIFIED | Re-verified at full depth this pass (the changed file's truth). `TestSmoke_twoNodesUDP` fresh PASS on this Linux host (0.02s): mutual discovery `KnownPeers()==1` both sides, byte-equal write visibility, own echo counted in OwnEcho. The D-08 loud-skip path is now also empirically proven: pre-fix macos-15 CI run failed (the observed EADDRINUSE answer), post-fix run 42514571899 has `gate (macos-15)` success — the ratified skip contract holds on macOS too. `TestBound_normalOneRTT` and `TestWire_twoNodeHandoff` green in the full-suite run; zero-config still holds (adapter `New` takes only bind/broadcast/nodeID/store/cfg) |
| 2 | Every local change is broadcast and tracked per-peer (key, version) until Ack or supersession, retransmitting with exponential backoff and suspending (not dropping) while no peer is known; receivers Ack after applying to memory; the unacked set drains after a partition heals | ✓ VERIFIED | Regression: core.go unchanged since previous verification (key seams re-grepped: `NotifyLocalWrite`, `scanRetransmit`, `handleAck`, `handleHello` all present); retransmit/peertable/sync test families green in the full `go test -race ./...` run |
| 3 | Full reconciliation works in both directions: chunked DataSync on the sync timer, at startup and on neighbor restart; receivers answer with DataRequest or unicast DataReplication rate-limited by `sync_response_rate`; a DataRequest for an unknown key gets no reply; a Hello with a new `run_id` triggers full sync toward the restarted node | ✓ VERIFIED | Regression: sync/restart machinery unchanged (`syncListing`, `handleDataSync`, `handleDataRequest`, `enqueueResponse`, `listingChunks` all present); `TestSync_*`, `TestDataRequest_replyAndSilence`, `TestRestart_*` green in the full-suite run |
| 4 | Malformed traffic is dropped with distinguishable classes: duplicate `node_id` logged as a configuration error, plus own broadcast echo, bad CRC and bad proto version each identifiable | ✓ VERIFIED | Regression: classifier unchanged; `TestDiscard_classes` (8 subtests, both duplicate variants) and `TestDiscard_totality` (500 adversarial datagrams) green in the full-suite run |
| 5 | The deterministic simulator runs 2..N nodes over a seeded virtual network that loses, duplicates, reorders and partitions datagrams, and asserts convergence within the documented bounds — normal one RTT, datagram loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval` + response time — printing the seed on failure | ✓ VERIFIED | Regression + fresh determinism spot-check: `SUPKV_SIM_SEED=4242 TestChaos_convergence` fresh PASS this pass; `TestBound_*` / `TestChaos_*` green in the full-suite run; simnet unchanged (Scheduler/Network/FaultStats seams re-grepped) |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Plan-Level Truth Detail (carried from the previous pass — files unchanged)

| Plan | Truths | Regression check this pass |
|------|--------|----------------------------|
| 04-01 (REPL-03) | 6/6 | port seam / core / adapter / replproto all unchanged; generated-code marker + protoc-gen-go provenance intact; full-suite green |
| 04-02 (REPL-02/06) | 8/8 | retransmit/peertable machinery unchanged; full-suite green |
| 04-03 (REPL-04/05) | 8/8 | sync machinery + limiter arithmetic unchanged; docs/IDEA.md echo unchanged (§Parameters); full-suite green |
| 04-04 (REPL-07) | 4/4 | restart machinery unchanged; full-suite green |
| 04-05 (TEST-03/REPL-10) | 6/6 | simnet + bound/chaos tests unchanged; seeded replay re-run fresh this pass — PASS |
| 04-06 (REPL-01/08/09) | 6/6 | discard machinery + persist seam unchanged; **smoke_test.go re-verified at full depth (see below)**; full-suite green |

### Changed-Area Verification: internal/udprepl/smoke_test.go (commit 507228a)

The change (test-only, 68+/27−) implements the D-08 loud-skip contract that
04-UAT item 2 had ratified, after the first macos-15 CI run (PR #31)
answered D-08 empirically: darwin refuses the probe's second wildcard UDP
bind to the same port (EADDRINUSE — SO_REUSEADDR does not share a wildcard
port there), which pre-fix made the probe fail the test instead of
classifying the platform.

- **Level 1 (exists):** file present, 331 lines — ✓
- **Level 2 (substantive):** full read by this verifier. New `errSharedWildcardBind` sentinel wraps exactly `syscall.EADDRINUSE` at the probe's rx bind; probe sequence extracted into `probeBroadcastAddr` (loopback first, then subnet broadcast) whose every unavailable path is a loud `t.Skipf` naming the probed addresses, the platform reason and the simulator pointer; `t.Fatalf` reserved for genuine setup breakage. The smoke test itself still asserts discovery (KnownPeers==1 both sides), byte-equal write visibility, and OwnEcho counting — value-level and behavioral assertions, not existence-level. — ✓
- **Level 3 (wired):** external test package (`udprepl_test`) driving the real adapter, real `repl.Core`, real `storage.Store` through the `smokeStore` `StateStore` adapter — the production seam. — ✓
- **Level 4 (data flows):** all asserted values flow from real UDP datagrams between two real sockets and real store merges; no static or mocked source. — ✓ FLOWING
- **Linux behavior unchanged:** fresh `TestSmoke_twoNodesUDP` PASS on this Linux host (0.02s).
- **macOS behavior now empirically proven:** pre-fix run 42513202587 `gate (macos-15)` = failure (the observed D-08 answer); post-fix run 42514571899 `gate (macos-15)` = success (loud-skip path taken, gate green). Recorded in STATE.md (commit d647af9).
- **Quality gates on the changed file:** `make lint` exit 0 (strict gate, incl. the nestif-motivated `probeBroadcastAddr` flattening) and the full race suite green.

### Required Artifacts

All artifacts across the six plans exist, are substantive, and are wired —
unchanged from the previous pass's manual Level 1-4 verification
(`gsd-tools query verify.artifacts` still returns 0/0 on these plans'
string-form artifact entries — same parser limitation as documented
previously; the previous manual verification stands and this pass confirmed
file-set composition is identical and only smoke_test.go changed):

| Artifact | Lines | Status | Wiring |
|----------|-------|--------|--------|
| `internal/port/repl.go` | 140 | ✓ VERIFIED | The seam both sides speak (core, adapter, simnet) |
| `internal/app/repl/core.go` | 911 | ✓ VERIFIED | Handler seams re-grepped present; no goroutines/locks/clock reads |
| `internal/udprepl/repl.go` | 520 | ✓ VERIFIED | Only net/protobuf replication code; single event loop; atomics mirror |
| `internal/udprepl/replproto/supkv.proto` + `supkv.pb.go` | 74 + 660 | ✓ VERIFIED | protoc-gen-go generated marker intact (line 8) |
| `internal/app/repl/{retransmit,peertable,sync,restart,sim_bound,sim_chaos,discard}_test.go` | — | ✓ VERIFIED | All green in the full race-suite run |
| `internal/udprepl/syncsize_internal_test.go` | 149 | ✓ VERIFIED | Green in full-suite run |
| `internal/app/internal/simnet/simnet.go` + `simnet_test.go` | 487 + 218 | ✓ VERIFIED | Test-only (no non-test importer); Scheduler/Network/FaultStats/PendingDeliveries/TraceDigest seams re-grepped present; IN-02's `c.nodes[i].store.Set` discard still at line 459 as documented |
| `internal/app/persist/engine.go` | 572 | ✓ VERIFIED | `ApplyRemote` matches `repl.StateStore` structurally; unchanged |
| `internal/app/persist/engine_test.go` | 1376 | ✓ VERIFIED | Green in full-suite run |
| `internal/udprepl/smoke_test.go` | 331 | ✓ VERIFIED | **Re-verified at full depth this pass** — see Changed-Area section |

### Key Link Verification

All key links verified WIRED in the previous pass; this pass re-confirmed
the anchor symbols are present and the wiring-relevant files are unchanged:

| From | To | Via | Status |
|------|----|-----|--------|
| port.Message structs | replproto types | `protoBody`/`portMessage` mapping; `TestWire_twoNodeHandoff` + `TestSync_chunkWireBudget` | ✓ WIRED |
| udprepl event loop | repl.Core handlers | Only `Run`'s loop calls HandleDatagram/NotifyLocalWrite/Advance | ✓ WIRED |
| NotifyLocalWrite | pending entry + broadcast | Wait-set seeded from peer table, ladder armed, broadcast emitted | ✓ WIRED |
| inbound Ack | wait-set drain | `handleAck` exact-version drain | ✓ WIRED |
| Advance | retransmit + eviction scans | Single loop fires both | ✓ WIRED |
| inbound DataRequest | rate-limited reply | Snapshot lookup → `enqueueResponse` → `drainEgress` | ✓ WIRED |
| startup/timer/restart | chunked listing emission | One `listingChunks` build serves both triggers | ✓ WIRED |
| simnet scheduler | Core handlers | `deliver`→HandleDatagram, `drive`→Advance | ✓ WIRED |
| simnet network | Outbox drain | `route` sends every Outbound; Broadcast fans out incl. sender | ✓ WIRED |
| per-node store | sim clock | `clock.NewClock(sched.Now, ...)` | ✓ WIRED |
| observability seams | quiescence | `PendingDeliveries` + `UnackedStats().Count` + `QueuedResponses` | ✓ WIRED |
| persist.Engine.ApplyRemote | repl.StateStore | Signature match; `TestEngine_applyRemoteTriggersDump` | ✓ WIRED |
| adapter atomics | smoke test | `KnownPeers`/`OwnEcho`/`Discards()` asserted through them (re-verified in the fresh smoke run) | ✓ WIRED |

### Data-Flow Trace (Level 4)

No rendered/static-value surfaces in this backend phase; the data-flow
equivalents all trace to real sources (unchanged this pass; the changed
smoke test's values were traced fresh — real UDP datagrams, real store merges):

| Path | Source | Status |
|------|--------|--------|
| Replicated records | `store.ApplyRemote` → real `storage.Store.Merge` (LWW join) | ✓ FLOWING |
| DataSync listings | `store.Records()` snapshot per build | ✓ FLOWING |
| Chaos model store | `WriteLog` merged by real `Merge` | ✓ FLOWING |
| Discard/peer/fault counters | Core/simnet state mutated by real handler calls; mirrored to atomics | ✓ FLOWING |

### Behavioral Spot-Checks

Single-test discipline where a named test adds signal over the single
full-suite run (which was run exactly once, exit 0):

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Changed-area: real-socket two-node discovery (REPL-01) | `go test ./internal/udprepl/ -run '^TestSmoke_twoNodesUDP$' -count=1` | fresh PASS (0.02s) on this Linux host | ✓ PASS |
| Seeded determinism replay (D-06 hook) | `SUPKV_SIM_SEED=4242 go test ./internal/app/repl/ -run '^TestChaos_convergence$' -count=1` | fresh PASS (0.008s) | ✓ PASS |
| Full workspace, race detector | `make test` (`go test -race -cover ./...`) | exit 0, all packages ok — covers every truth's named test family on the unchanged files | ✓ PASS |
| Strict lint gate (incl. changed file) | `make lint` | exit 0, zero findings | ✓ PASS |
| macOS loud-skip contract (empirical) | `gh run view 42514571899` (post-fix CI) | `gate (macos-15)` success; pre-fix run 42513202587 failure documented the platform answer | ✓ PASS |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` exist or are referenced by
any PLAN/SUMMARY. The plans' numbered "probe #N" items are pinned research
edge cases discharged by the named Go tests, not executable scripts.

### Requirements Coverage

Plan frontmatter union: REPL-01,02,03,04,05,06,07,08,09,10 + TEST-03 —
exactly the 11 IDs REQUIREMENTS.md assigns to Phase 4 (all marked Complete
in its traceability table). No orphaned requirements, no unclaimed IDs.

| Requirement | Source Plan | Status | Evidence |
|-------------|------------|--------|----------|
| REPL-01 | 04-06 | ✓ SATISFIED | `TestSmoke_twoNodesUDP` fresh PASS (real sockets, zero config); loud-skip path now empirically proven on macOS CI too |
| REPL-02 | 04-02 | ✓ SATISFIED | Tracking/backoff/suspension/flush — unchanged files, full-suite green |
| REPL-03 | 04-01 | ✓ SATISFIED | `TestWire_twoNodeHandoff` + Ack-after-apply — unchanged, full-suite green |
| REPL-04 | 04-03 | ✓ SATISFIED | Unknown key → zero outbound — unchanged, full-suite green |
| REPL-05 | 04-03 | ✓ SATISFIED | Chunked listing + limiter — unchanged, full-suite green; IDEA.md echo intact |
| REPL-06 | 04-02 | ✓ SATISFIED | Peer table lifecycle — unchanged, full-suite green |
| REPL-07 | 04-04 | ✓ SATISFIED | Restart full sync — unchanged, full-suite green |
| REPL-08 | 04-06 | ✓ SATISFIED | Both duplicate variants dropped+logged+counted — unchanged, full-suite green |
| REPL-09 | 04-01, 04-06 | ✓ SATISFIED | Five distinguishable classes, totality — unchanged, full-suite green; own-echo counted in fresh smoke run |
| REPL-10 | 04-05 | ✓ SATISFIED | Three bound tests — unchanged, full-suite green |
| TEST-03 | 04-05 | ✓ SATISFIED | Chaos convergence + determinism — unchanged files + fresh seeded replay PASS |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/app/repl/core.go | 229 | `ownHandle` write-only field (review IN-01) | ℹ️ Info | Open by documented triage (04-REVIEW-DISPOSITION.md); dead state, no behavioral effect; carried unchanged from previous pass |
| internal/app/internal/simnet/simnet.go | 459 | `_, _, _ = c.nodes[i].store.Set(...)` — Write discards the Set error (review IN-02) | ℹ️ Info | Open by documented triage; test-harness-only path, unreachable with current scenario payloads; carried unchanged from previous pass |

Debt-marker gate: zero `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`
across all covered files (production and test), re-grepped this pass. No
placeholder/empty-impl patterns in the changed file; its `t.Skip` paths are
documented platform-capability classifications (D-08 design, ratified), not
requirement skips — on Linux (and now macOS) the test runs or loud-skips
with the classification, never silently.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/udprepl/smoke_test.go | REPL-01, REPL-09 | yes (Linux run; macOS loud-skip by ratified D-08 design — capability classification, not a requirement skip) | conditional-by-design | no | Behavioral (mutual discovery, byte-equal visibility, OwnEcho counter) | VALID |
| internal/app/repl/sim_chaos_test.go | TEST-03, REPL-10 | yes | 0 | no (expected values from the supremum model, not the SUT's store dump) | Behavioral (convergence + byte-identical replay) | VALID |
| internal/app/repl/{retransmit,peertable,sync,restart,discard}_test.go | REPL-02..09 | yes | 0 | no | Value/Behavioral | VALID |
| internal/app/persist/engine_test.go | REPL-03 | yes | 0 | no | Behavioral (dump timing through synctest) | VALID |

**Disabled tests on requirements:** 0 (the smoke test's platform skip is a
loud capability classification with simulator coverage pointed to, ratified
by UAT — not a disabled requirement test)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Decision Coverage

8/8 trackable CONTEXT.md decisions honored by shipped artifacts
(`check.decision-coverage-verify`: `total: 8, honored: 8, not_honored: []` —
re-run this pass, matches the previous report).

### Prohibition Disposition (ADR-550 D4)

| Prohibition | Tier | Verdict |
|-------------|------|---------|
| REPL-01 (04-01, 04-06): no config surface beyond broadcast_addr | test | ENFORCED — no peer lists/static addressing exist; smoke test proves zero-config discovery on real sockets (fresh PASS) |
| REPL-08 (04-06): duplicate node_id never silently resolved | test | ENFORCED — both variants drop + log + count; `TestDiscard_classes` |
| 04-02: pending never dropped on timeout/eviction/empty table | test | ENFORCED — only Ack/supersession delete; `TestRetransmit_suspendAndFlush` |
| 04-02/D-05: retransmission never blocks a local write | test | ENFORCED — enqueue-and-return; `TestNotifyLocalWrite_afterRunReturns` |
| 04-02/REPL-06: no blacklist, no peer-count cap | test | ENFORCED — evicted peers re-learned; unbounded node_id-keyed map |
| 04-03: no bypass of the rate-limited egress queue | test | ENFORCED — exactly three producer sites call `enqueueResponse` |
| 04-03/A4: chunks carry no continuation state | test | ENFORCED — self-sufficient slices |
| 04-03/REPL-04: replies unicast only | test | ENFORCED — unicast/queued unicast only |
| 04-04: no second peer row on run_id change | test | ENFORCED — upsert keyed by node_id; `TestRestart_staleRunIdAndInFlight` |
| 04-04: full-sync reaction only on run_id change | test | ENFORCED — inequality is the only trigger; `TestRestart_newPeerNoFullSync` |
| 04-05: no second protocol implementation in simnet | test | ENFORCED — real Core handlers; no non-test importer |
| 04-05: no wall-clock dependence | test | ENFORCED — no `time.Sleep` in decision path |
| 04-05/D-06: sim tooling not in production binary | test | ENFORCED — simnet imported only from `_test.go` files |
| REPL-09 CRC wording (04-01, 04-06) | judgment | **RATIFIED — resolved.** 04-UAT.md item 1: result pass (human ratification 2026-11-06/07, "total 2, passed 2, issues 0"). Regression grep this pass confirms all mention sites still say corruption detection only, never tamper protection (supkv.proto:15, supkv.pb.go:33, udprepl/repl.go:371, storage/store.go:103, storage/doc.go:35; codec_test.go uses "tamper" only as an attack verb in test names). No flagged prohibitions remain |

### Human Verification Required

None outstanding. The previous report's two items are resolved with recorded
evidence (reused per the re-verification contract, not re-escalated):

1. **Judgment-tier CRC-wording prohibition (plans 04-01/04-06)** — RESOLVED.
   Ratified by the human UAT run: 04-UAT.md item 1, result pass
   (2026-11-06/07). Regression grep this pass confirms the wording is
   unchanged in every mention site.
2. **D-08 macOS smoke outcome** — RESOLVED empirically. The first macos-15
   CI run (PR #31, 2026-11-08) answered the open question: darwin refuses
   the shared wildcard bind (EADDRINUSE), which the pre-fix probe turned
   into a failure (run 42513202587, gate macos-15 = failure). Commit 507228a
   maps exactly that errno to the ratified loud-skip contract, and the
   post-fix run 42514571899 shows `gate (macos-15)` = success. Recorded in
   STATE.md (commit d647af9). No macOS runner is needed for this phase's
   goal anymore; future platform behavior is covered by the ongoing CI gate.

### Advisory (New Scope, Unevidenced)

Re-verification ran; Step 7 surfaced no new-scope findings on the changed
file or elsewhere.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Gaps Summary

None. The staleness trigger was commit 507228a's test-only change to
`internal/udprepl/smoke_test.go`; that file re-verified at full depth (all
four levels plus fresh Linux PASS and empirical macOS CI evidence), every
other covered file is byte-identical to what the previous pass verified,
the full race suite and the strict lint gate are green on this tree, the
seeded determinism replay passes fresh, and both previously-flagged human
verification items are resolved with recorded evidence (UAT ratification +
macOS CI outcome). The phase goal — convergence to the pointwise supremum
over a lossy UDP segment, proven under chaos by the deterministic simulator
— remains achieved in the codebase. Status is `passed`: no `gaps:` entries,
no outstanding human-verification items, no flagged prohibitions.

---

_Verified: 2026-11-08T21:09:31Z_
_Verifier: Claude (gsd-verifier)_
