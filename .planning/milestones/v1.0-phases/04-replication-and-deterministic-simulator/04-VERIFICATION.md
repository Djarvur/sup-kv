---
phase: 04-replication-and-deterministic-simulator
verified: 2026-11-18T10:47:44Z
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
covered_digest: "v4:sha256:d54e11b10724765f7656f8f98e6acf27c3d0e2b1aaae431c9af8ca54612df3d7"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 5/5
  gaps_closed: [] # staleness regeneration, not gap closure — the previous pass (verified 2026-11-16T10:49:51Z, digest 013066a8…) went stale when the Phase-7 release session committed onto three covered files after that timestamp: Makefile (+15, additive `release`/`release-snapshot` goreleaser targets, ac14f92), go.mod (tool directives +grpcurl +goreleaser, go directive 1.27.1→1.27.4), go.sum (graph growth from those tools). The replication/simulator Go source is byte-identical since that pass (git diff @{2026-11-16T10:49:51}..HEAD over covered paths: exactly those 3 files)
  gaps_remaining: []
  regressions: []
advisory:
  - finding: "WR-03 (04-REVIEW-DISPOSITION.md, open): the `release-snapshot` Makefile target — phase-7's mandatory pre-tag gate — fails on machines without the external `syft` binary"
    category: other
    reason: "New-scope since the prior verified pass (raised by the 2026-11-18 cross-phase code review). Concerns the phase-7 release pipeline added to the shared Makefile AFTER this phase's scope; no phase-4 must-have maps to release targets (phase-4's Makefile concern is the `test`/`lint`/`bench` targets, all re-proven green this pass). No deterministic evidence of phase-4 goal impact; already tracked `open` in 04-REVIEW-DISPOSITION.md. Would be resolved by a go.mod-pinned syft tool directive or a help-text fix — phase-7 scope."
    evidence_status: "none provided"
  - finding: "WR-04 (04-REVIEW-DISPOSITION.md, open): the grpcurl/goreleaser go.mod tool-directive additions sit outside govulncheck's import-graph reach (253 new indirect requires invisible to the vulnerability gate)"
    category: security
    reason: "New-scope since the prior verified pass (2026-11-18 review). A supply-chain observation about phase-7 dev tooling; the shipped runtime graph this phase depends on is unchanged (direct requires exactly grpc v1.85.0 + protobuf v1.36.13-…, re-read this pass) and the phase-4 code compiles and passes the full race suite on the new module set. No phase-4 must-have maps to vulnerability-gate coverage of tool directives; tracked `open` in 04-REVIEW-DISPOSITION.md. Would be resolved by a tool-graph scanning step in vulncheck.yml — phase-7/release scope."
    evidence_status: "none provided"
---

# Phase 4: Replication and Deterministic Simulator Verification Report

**Phase Goal:** Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment, with convergence under chaos proven by the deterministic simulator
**Verified:** 2026-11-18T10:47:44Z
**Status:** passed
**Re-verification:** Yes — staleness regeneration. The previous report (verified 2026-11-16T10:49:51Z, digest `013066a8…`) went stale when the Phase-7 release session (through HEAD = v1.0.0, 5533509) committed onto three covered files after that timestamp: `Makefile` (additive `release`/`release-snapshot` goreleaser targets), `go.mod` (tool directives +grpcurl +goreleaser; `go 1.27.1`+toolchain → `go 1.27.4`), `go.sum` (graph growth from those tools — 285→538 indirect requires). `git diff --name-only @{2026-11-16T10:49:51}..HEAD` over every covered path shows exactly those three files: the replication/simulator Go source, `internal/port`, `buf.yaml`, and all PLAN/SUMMARY docs are byte-identical to what the previous pass verified at full depth. This pass therefore: full-depth-verifies the three changed files (whose deltas are phase-7 release machinery, not replication logic), regression-checks the unchanged set (existence + purity/wiring greps), and re-proves every behavior-dependent truth with fresh named tests on the CURRENT tree — necessary because the go directive bump and module-graph growth could in principle have changed build or test behavior; they did not (all evidence below).

## Goal Achievement

### Observable Truths

The five ROADMAP success criteria are the contract; the 42 plan-level truths
from the six PLANs refine these five. All five re-verified against the
CURRENT codebase (post-phase-7 Makefile/go.mod/go.sum changes) with fresh
behavioral evidence — every behavior-dependent truth has a named passing
test from this pass (commands below).

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|-----------------------------------|--------|----------|
| 1 | Two (or N) nodes discover each other via broadcast Hello with no configuration beyond `broadcast_addr`, and a write on one node becomes visible on every other within one RTT in normal operation | ✓ VERIFIED | `TestSmoke_twoNodesUDP` fresh verbose PASS (RUN then PASS, 0.02s, real sockets); `TestBound_normalOneRTT` fresh PASS; smoke_test.go byte-identical since the prior full-depth pass (absent from the staleness diff); `repl.Config` re-read on the current tree — timing knobs + `Log` + `OnDiscard` callback only, no peer lists, no static addressing; adapter `New(bindAddr, broadcastAddr, nodeID, store, cfg)` unchanged (repl.go:219) |
| 2 | Every local change is broadcast and tracked per-peer (key, version) until Ack or supersession, retransmitting with exponential backoff and suspending (not dropping) while no peer is known; receivers Ack after applying to memory; the unacked set drains after a partition heals | ✓ VERIFIED | core.go byte-identical since the prior pass (purity grep re-run: `go func`/`time.Now`/`time.Sleep`/`sync.`/`"net"` = 0 matches); `TestRetransmit_ackDrainSemantics`, `TestRetransmit_suspendAndFlush`, `TestRetransmit_backoffLadder`, `TestPeerTable_lifecycle`, `TestNotifyLocalWrite_afterDone`, `TestNotifyLocalWrite_afterRunReturns` all fresh PASS |
| 3 | Full reconciliation works in both directions: chunked DataSync on the sync timer, at startup and on neighbor restart; receivers answer with DataRequest or unicast DataReplication rate-limited by `sync_response_rate`; a DataRequest for an unknown key gets no reply; a Hello with a new `run_id` triggers full sync toward the restarted node | ✓ VERIFIED | `TestSync_rateLimit`, `TestSync_syncArrived`, `TestSync_responseRateClamp`, `TestRestart_runIDChangeTriggersFullSync` (+`staleRunIdAndInFlight`, `newPeerNoFullSync`), `TestSync_chunkWireBudget` all fresh PASS; egress-queue prohibition re-proven: exactly three producer call sites call `enqueueResponse` (core.go:610, 639, 641 — definition at 652 excluded) |
| 4 | Malformed traffic is dropped with distinguishable classes: duplicate `node_id` logged as a configuration error, plus own broadcast echo, bad CRC and bad proto version each identifiable | ✓ VERIFIED | `TestDiscard_classes` + `TestDiscard_totality` (500 adversarial datagrams) fresh PASS; discard logic byte-identical since the prior full-depth pass (discard_test.go, core.go unchanged) |
| 5 | The deterministic simulator runs 2..N nodes over a seeded virtual network that loses, duplicates, reorders and partitions datagrams, and asserts convergence within the documented bounds — normal one RTT, datagram loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval` + response time — printing the seed on failure | ✓ VERIFIED | Fresh seeded replay `SUPKV_SIM_SEED=4242 TestChaos_convergence` PASS (0.21s) + all three `TestBound_*` fresh PASS on the current tree; simnet.go byte-identical since the prior full-depth pass; purity grep re-run on current file: `go func`/`time.Now`/`time.Sleep`/`sync.` = 0; simnet still imported only from `_test.go` files (0 non-test importers re-grepped — D-06 holds) |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Plan-Level Truth Detail

Unchanged source since the prior full-depth pass — regression-checked with
fresh named tests per plan:

| Plan | Truths | Check this pass |
|------|--------|-----------------|
| 04-01 (REPL-03) | 6/6 | replproto tree + core.go + repl.go byte-identical since the prior pass; `TestWire_twoNodeHandoff` fresh PASS; purity grep 0 on core.go |
| 04-02 (REPL-02/06) | 8/8 | 4 named retransmit/peertable tests + both NotifyLocalWrite regressions fresh PASS |
| 04-03 (REPL-04/05) | 8/8 | `TestSync_rateLimit` + `TestSync_responseRateClamp` fresh PASS; egress producers exactly 3 (re-grepped) |
| 04-04 (REPL-07) | 8/8 | `TestRestart_runIDChangeTriggersFullSync` + both edge regressions fresh PASS |
| 04-05 (TEST-03/REPL-10) | 6/6 | Fresh seeded chaos replay + all three bounds PASS; determinism purity + test-only linkage re-proven |
| 04-06 (REPL-01/08/09) | 6/6 | `TestSmoke_twoNodesUDP` fresh verbose PASS; `TestDiscard_classes`/`TestDiscard_totality` fresh PASS; persist seam `TestEngine_applyRemoteTriggersDump` + `TestEngine_redundantDirty` fresh PASS |

### Changed-Area Verification: the Phase-7 commits on Phase-4 covered files

Exactly three covered files changed since 2026-11-16T10:49:51Z
(`git diff --stat @{2026-11-16T10:49:51}..HEAD -- <covered paths>`:
Makefile +15, go.mod +295/−13, go.sum — 3 files). All additive
phase-7 release machinery; none touches replication logic:

- **`Makefile`** (+15) — two new targets appended (`release`, `release-snapshot`,
  both `go tool goreleaser …`) plus their `.PHONY` entries. The phase-4-relevant
  targets are untouched and re-proven green this pass: `make test` (exit 0,
  full race suite, all packages ok) and `make lint` (exit 0, config verify +
  run, zero findings). `test`/`lint`/`bench` bodies byte-identical (diff shows
  only the appended block and the one `.PHONY` line).
- **`go.mod`** — `go 1.27.1`+`toolchain go1.27.4` collapsed to `go 1.27.4`
  (local toolchain is exactly go1.27.4 — build proven); tool directives
  +`github.com/fullstorydev/grpcurl/cmd/grpcurl`,
  +`github.com/goreleaser/goreleaser/v2`. The DIRECT require block is
  unchanged: exactly `google.golang.org/grpc v1.85.0` +
  `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e` —
  the phase-4 wire/runtime contract holds. Everything else is `// indirect`
  graph growth pulled by the two tools. Phase-4 code compiles and passes the
  full race suite on this module set (evidence below).
- **`go.sum`** — mechanical graph growth for the tool directives; no direct
  dependency version changed (verified: the direct-require lines are
  byte-identical between the diff sides).
- **Everything else covered** — `internal/app/repl/**`, `internal/udprepl/**`
  (incl. replproto), `internal/app/persist/engine*.go`, `internal/app/internal/simnet/**`,
  `internal/port/repl.go`, `buf.yaml`, all PLAN/SUMMARY docs: ABSENT from
  the staleness diff, byte-identical to the prior verified pass.

### Required Artifacts

All artifacts across the six plans exist, are substantive, and are wired.
(`gsd-tools query verify.artifacts` returns 0/0 on every plan — the known
parser limitation with string-form artifact entries, documented in every
previous pass; manual Level 1-4 verification stands.)

| Artifact | Lines (current) | Status | Wiring |
|----------|-------|--------|--------|
| `internal/port/repl.go` | 140 | ✓ VERIFIED | Unchanged since 2026-11-08; the seam both sides speak |
| `internal/app/repl/core.go` | 1073 | ✓ VERIFIED | Unchanged this window; purity greps 0; single-event-loop discipline intact; green in full suite |
| `internal/udprepl/repl.go` | 850 | ✓ VERIFIED | Unchanged this window; `New` signature re-read; smoke/mirror/notify tests fresh PASS |
| `internal/udprepl/replproto/supkv/v1/supkv.proto` | 74 | ✓ VERIFIED | Unchanged; wire tests fresh PASS |
| `internal/udprepl/replproto/supkv/v1/supkv.pb.go` | 660 | ✓ VERIFIED | Unchanged; sole consumer repl.go; `TestWire_twoNodeHandoff` fresh PASS |
| `internal/app/repl/{retransmit,peertable,sync,restart,discard,sim_bound,sim_chaos,sim_scenario,merge_stats}_test.go` | 715/235/798/330/565/269/291/—/— | ✓ VERIFIED | All green in the full race suite; headline tests re-run fresh this pass |
| `internal/udprepl/syncsize_internal_test.go` | 149 | ✓ VERIFIED | Unchanged; fresh PASS (`TestSync_chunkWireBudget`) |
| `internal/app/internal/simnet/simnet.go` + `simnet_test.go` | 599 + 218 | ✓ VERIFIED | Unchanged this window; determinism purity 0; 0 non-test importers; fresh seeded replay PASS |
| `internal/app/persist/engine.go` | 697 | ✓ VERIFIED | Unchanged this window; seam tests fresh PASS |
| `internal/app/persist/engine_test.go` (+internal, bench) | 1623 | ✓ VERIFIED | Unchanged; green in full suite; seam cases re-run fresh |
| `internal/udprepl/smoke_test.go` | 344 | ✓ VERIFIED | Unchanged; fresh verbose PASS this pass (ran, not skipped) |
| `Makefile` | 92 | ✓ VERIFIED | CHANGED (phase-7 release targets, additive) — `test`/`lint` bodies untouched and both re-proven exit 0 this pass |
| `go.mod` / `go.sum` | — | ✓ VERIFIED | CHANGED (phase-7 tool directives + go 1.27.4) — direct requires unchanged (grpc v1.85.0, protobuf v1.36.13-…); build + full race suite green on the new set |
| `buf.yaml` | — | ✓ VERIFIED | Unchanged this window |

### Key Link Verification

All key links WIRED; the Go source behind every link is byte-identical to
the prior full-depth pass, and each link's proof test re-ran fresh this pass:

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| port.Message structs | replproto generated types | `protoBody`/`portMessage` mapping | ✓ WIRED | `TestWire_twoNodeHandoff` + `TestSync_chunkWireBudget` fresh green |
| udprepl event loop | repl.Core handlers | Only `Run`'s loop calls HandleDatagram/NotifyLocalWrite/Advance | ✓ WIRED | Source unchanged; no second driver (prior full-depth read stands) |
| NotifyLocalWrite | pending entry + broadcast | One select over notify+done, re-check after won send | ✓ WIRED | `TestNotifyLocalWrite_afterDone` + `afterRunReturns` fresh PASS |
| inbound Ack | wait-set drain | `handleACK` exact-version drain | ✓ WIRED | `TestRetransmit_ackDrainSemantics` fresh PASS |
| Advance | retransmit + eviction scans | Single loop fires both, sorted by key | ✓ WIRED | `TestRetransmit_backoffLadder` + `TestPeerTable_lifecycle` fresh PASS |
| inbound DataRequest | rate-limited reply | Snapshot lookup → `enqueueResponse` → `drainEgress` | ✓ WIRED | `TestSync_rateLimit` fresh PASS; exactly 3 producer sites (core.go:610,639,641) |
| startup/timer/restart | chunked listing emission | One `listingChunks` build serves both triggers | ✓ WIRED | `TestRestart_runIDChangeTriggersFullSync` fresh PASS |
| simnet scheduler | Core handlers | `deliver`→HandleDatagram, `drive`→Advance | ✓ WIRED | Fresh seeded chaos replay PASS |
| simnet network | Outbox drain | `route` sends every Outbound; Broadcast fans out incl. sender | ✓ WIRED | Exercised by the fresh chaos run |
| per-node store | sim clock | `clock.NewClock(sched.Now + offset slot)` | ✓ WIRED | Offsets start zero; exercised by fresh bounds run |
| observability seams | quiescence | `PendingDeliveries` + `UnackedStats().Count` + `QueuedResponses()` | ✓ WIRED | Exercised by the fresh seeded chaos run |
| persist.Engine.ApplyRemote | repl.StateStore | Signature match; conditional dirty | ✓ WIRED | `TestEngine_applyRemoteTriggersDump` + `TestEngine_redundantDirty` fresh PASS |
| adapter atomics | smoke test / Phase-5+ readers | KnownPeers/OwnEcho/Discards()/MergeStats/Retransmissions mirrored post-event | ✓ WIRED | `TestMirror_drainedAckedDedup` fresh PASS |
| OnDiscard seam | structured discard events | core invokes `cfg.OnDiscard(class, detail)`; adapter defaults to slog bridge | ✓ WIRED | `TestDiscard_classes` fresh PASS |

### Data-Flow Trace (Level 4)

No rendered/static-value surfaces in this backend phase; the data-flow
equivalents all trace to real sources (source byte-identical to the prior
full-depth pass; consumers re-exercised fresh):

| Path | Source | Status |
|------|--------|--------|
| Replicated records | `store.ApplyRemote` → real `storage.Store.Merge` (LWW join) | ✓ FLOWING |
| Dirty-triggered dumps | `ApplyRemote` conditional dirty (displaced \|\| first-install) | ✓ FLOWING |
| DataSync listings | `store.Records()` snapshot per build | ✓ FLOWING |
| Chaos model store | `WriteLog` merged by real `Merge` (simnet) | ✓ FLOWING |
| Discard/peer/fault/merge/retransmit counters | Core state mutated by real handler calls; mirrored post-event | ✓ FLOWING |
| Wire encode/decode | Real proto Marshal/Unmarshal — exercised live by fresh wire/chunk/smoke runs | ✓ FLOWING |

### Behavioral Spot-Checks

Single-test discipline; every command run fresh (`-count=1`, `-race`) this
pass on the current tree (go 1.27.4, post-phase-7 module set). The full
suite ran exactly once (`make test`, exit 0, all packages ok); `make lint`
ran once (exit 0, zero findings) — both needed re-proving because go.mod
changed since the prior verified pass.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Real-socket two-node discovery (REPL-01) | `go test -race -count=1 -run 'TestSmoke_twoNodesUDP' ./internal/udprepl/` | verbose `=== RUN` → `--- PASS` (0.02s), ran not skipped | ✓ PASS |
| Seeded determinism replay (TEST-03) | `SUPKV_SIM_SEED=4242 go test -race -count=1 -run 'TestChaos_convergence' ./internal/app/repl/` | PASS (0.21s) | ✓ PASS |
| Convergence bounds (REPL-10) | `go test -race -count=1 -run 'TestBound_normalOneRTT\|TestBound_partitionHeal\|TestBound_lossWithinAckTimeoutMax' ./internal/app/repl/` | all 3 PASS | ✓ PASS |
| Wire handoff + cross-seam chunk budget | `go test -race -count=1 -run 'TestWire_twoNodeHandoff\|TestSync_chunkWireBudget' ./internal/udprepl/` | PASS | ✓ PASS |
| Ack drain / suspension / ladder / peer lifecycle (REPL-02/06) | `go test -race -count=1 -run 'TestRetransmit_ackDrainSemantics\|TestRetransmit_suspendAndFlush\|TestRetransmit_backoffLadder\|TestPeerTable_lifecycle' ./internal/app/repl/` | all 4 PASS | ✓ PASS |
| Rate limiter + clamp + sync-arrived (REPL-05) | `go test -race -count=1 -run 'TestSync_rateLimit\|TestSync_responseRateClamp\|TestSync_syncArrived' ./internal/app/repl/` | all 3 PASS | ✓ PASS |
| Restart full sync + edges (REPL-07) | `go test -race -count=1 -run 'TestRestart_runIDChangeTriggersFullSync\|TestRestart_staleRunIdAndInFlight\|TestRestart_newPeerNoFullSync' ./internal/app/repl/` | all 3 PASS | ✓ PASS |
| Discard classification + totality (REPL-08/09) | `go test -race -count=1 -run 'TestDiscard_classes\|TestDiscard_totality' ./internal/app/repl/` | PASS | ✓ PASS |
| NotifyLocalWrite closed loop (IN-01 regression) | `go test -race -count=1 -run 'TestNotifyLocalWrite_afterDone\|TestNotifyLocalWrite_afterRunReturns' ./internal/udprepl/` | PASS | ✓ PASS |
| Mirror dedup (WR-01 regression) | `go test -race -count=1 -run 'TestMirror_drainedAckedDedup' ./internal/udprepl/` | PASS | ✓ PASS |
| Persist seam + redundant-dirty (REPL-03/IN-02) | `go test -race -count=1 -run 'TestEngine_applyRemoteTriggersDump\|TestEngine_redundantDirty\|TestFlush_remoteDirtiness\|TestFlush_clean' ./internal/app/persist/` | all 4 PASS | ✓ PASS |
| Full workspace, race detector, current go.mod | `make test` | exit 0, all packages ok (persist 97.7%, repl 97.8%, udprepl 86.2% coverage) | ✓ PASS |
| Strict lint gate on current go.mod | `make lint` | exit 0, zero findings | ✓ PASS |

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
| REPL-01 | 04-01, 04-06 | ✓ SATISFIED | `TestSmoke_twoNodesUDP` fresh verbose PASS; `Config`/`New` re-read — no addressing surface |
| REPL-02 | 04-02 | ✓ SATISFIED | 4 named retransmit/peertable tests fresh PASS; source unchanged |
| REPL-03 | 04-01, 04-06 | ✓ SATISFIED | `TestWire_twoNodeHandoff` + `TestEngine_applyRemoteTriggersDump` fresh PASS |
| REPL-04 | 04-03 | ✓ SATISFIED | handleDataRequest unchanged; silence case covered in full suite (green) |
| REPL-05 | 04-03 | ✓ SATISFIED | `TestSync_rateLimit` + `TestSync_responseRateClamp` + `TestSync_chunkWireBudget` fresh PASS |
| REPL-06 | 04-02 | ✓ SATISFIED | `TestPeerTable_lifecycle` fresh PASS; unbounded node_id-keyed map unchanged |
| REPL-07 | 04-04 | ✓ SATISFIED | 3 named restart tests fresh PASS |
| REPL-08 | 04-06 | ✓ SATISFIED | `TestDiscard_classes` fresh PASS; drop + Log + count + OnDiscard token |
| REPL-09 | 04-06 | ✓ SATISFIED | `TestDiscard_totality` fresh PASS; CRC wording regression grep re-run: every production mention says corruption detection only, never tamper protection (repl.go:701, supkv.pb.go:33, store.go:130, doc.go:35) |
| REPL-10 | 04-05 | ✓ SATISFIED | All three `TestBound_*` fresh PASS |
| TEST-03 | 04-05 | ✓ SATISFIED | Fresh seeded replay PASS; purity + test-only linkage re-proven |

### Anti-Patterns Found

Debt-marker gate re-run on the changed covered files (Makefile, go.mod,
go.sum): zero `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` matches; and
on the unchanged covered implementation files: zero matches (regression
grep). Empty-implementation scan: zero matches. The smoke test's
`t.Skipf` calls are the documented platform-capability probes (D-08
ratified: loud skip naming both addresses and the platform reason, only
when same-host broadcast is genuinely unavailable) — on Linux the test RUNS
and PASSED fresh this pass; not a disabled test.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | no debt markers, stubs, or empty implementations in any covered file | — | — |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Verdict |
|-----------|-----------|--------|---------|----------|---------|
| internal/udprepl/smoke_test.go | REPL-01, REPL-09 | yes (Linux verbose RUN+PASS this pass; macOS loud-skip by ratified D-08 design) | conditional-by-design | no | VALID |
| internal/udprepl/repl_test.go (+repl_internal_test.go) | REPL-03 wire + regressions | yes | 0 | no | VALID |
| internal/app/repl/sim_chaos_test.go | TEST-03, REPL-10 | yes | 0 | no (expected values from the supremum model, not the SUT) | VALID |
| internal/app/repl/{retransmit,peertable,sync,restart,discard,sim_bound,sim_scenario,merge_stats}_test.go | REPL-02..10, TEST-03 | yes | 0 | no | VALID |
| internal/udprepl/{scenario,bench}_test.go | scenario adjuncts | yes | 0 | no | VALID |
| internal/app/persist/{engine_test.go,engine_internal_test.go,bench_test.go} | REPL-03 persist seam + regressions | yes | 0 | no | VALID |

**Disabled tests on requirements:** 0. **Circular patterns detected:** 0.

### Prohibition Disposition (ADR-550 D4)

| Prohibition | Tier | Verdict |
|-------------|------|---------|
| REPL-01 (04-01, 04-06): no config surface beyond broadcast_addr | test | ENFORCED — `New` (repl.go:219) and `Config` re-read on the current tree: timing knobs + Log + OnDiscard callback only; no peer lists, no static addressing anywhere |
| REPL-08 (04-06): duplicate node_id never silently resolved | test | ENFORCED — both variants drop + Log + count + OnDiscard token; `TestDiscard_classes` fresh PASS |
| 04-02: pending never dropped on timeout/eviction/empty table | test | ENFORCED — source unchanged since prior diff-verified pass; `TestRetransmit_suspendAndFlush` fresh PASS |
| 04-02/D-05: retransmission never blocks a local write | test | ENFORCED — NotifyLocalWrite still enqueue-and-return; `TestNotifyLocalWrite_afterRunReturns` fresh PASS |
| 04-02/REPL-06: no blacklist, no peer-count cap | test | ENFORCED — evicted peers re-learned; unbounded node_id-keyed map unchanged |
| 04-03: no bypass of the rate-limited egress queue | test | ENFORCED — exactly three producer sites (core.go:610, 639, 641; re-grepped this pass) |
| 04-03/A4: chunks carry no continuation state | test | ENFORCED — self-sufficient slices (source unchanged) |
| 04-03/REPL-04: replies unicast only | test | ENFORCED — unicast/queued unicast only (source unchanged) |
| 04-04: no second peer row on run_id change | test | ENFORCED — upsert keyed by node_id; `TestRestart_staleRunIdAndInFlight` fresh PASS |
| 04-04: full-sync reaction only on run_id change | test | ENFORCED — inequality is the only trigger; `TestRestart_newPeerNoFullSync` fresh PASS |
| 04-05: no second protocol implementation in simnet | test | ENFORCED — simnet drives the SAME real Core handlers; 0 non-test importers (re-grepped) |
| 04-05: no wall-clock dependence | test | ENFORCED — `time.Sleep`/`time.Now`/`go func`/`sync.` in core.go and simnet.go: 0 matches (re-grepped) |
| 04-05/D-06: sim tooling not in production binary | test | ENFORCED — simnet imported only from `_test.go` files (re-grepped: 0 non-test importers) |
| REPL-09 CRC wording (04-01, 04-06) | judgment | **RATIFIED — resolved** (04-UAT.md item 1, human ratification 2026-11-06/07; UAT status complete). Regression grep this pass: every production mention still says corruption detection only, never tamper protection (repl.go:701, supkv.pb.go:33, store.go:130, doc.go:35). No flagged prohibitions remain |

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts:
8/8 honored, 0 not honored (`gsd-tools query check.decision-coverage-verify`).

### Advisory (New Scope, Unevidenced)

New-scope findings from the 2026-11-18 cross-phase code review (04-REVIEW.md /
04-REVIEW-DISPOSITION.md — recorded AFTER the previous verified pass) that
touch files in this phase's covered set. Both concern phase-7 release
machinery added to shared files after this phase's scope closed; neither maps
to any phase-4 must-have, and no deterministic evidence ties either to
phase-4 goal failure (the contrary holds: full race suite + lint green on
the current tree). Reported, not blocking; both are already tracked
`open` (recorded, not yet triaged) in 04-REVIEW-DISPOSITION.md.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | WR-03: `release-snapshot` Makefile target fails without the external `syft` binary (empirically reproduced during the review) | other | phase-7 release pipeline on the shared Makefile; phase-4's Makefile surface (`test`/`lint`/`bench`) re-proven green; new-scope, no phase-4 goal impact evidenced |
| 2 | WR-04: grpcurl/goreleaser tool-directive graphs (253 new indirect requires) invisible to `govulncheck ./...` | security | phase-7 dev tooling supply-chain observation; shipped runtime graph this phase depends on unchanged (direct requires grpc+protobuf only, re-read); new-scope, no phase-4 goal impact evidenced |

### Human Verification Required

None outstanding. Infrastructure/foundation phase — no user-facing
elements; all acceptance criteria verifiable programmatically. The previous
report's two items remain resolved with recorded evidence: the CRC-wording
judgment prohibition is ratified by human UAT (04-UAT.md, status complete,
2026-11-06/07) with the wording re-grepped unchanged on the current tree,
and D-08's macOS smoke outcome is empirically settled by CI. No
PRESENT_BEHAVIOR_UNVERIFIED truths — every behavior-dependent truth has a
fresh passing named test this pass.

### Gaps Summary

None. The staleness trigger was the Phase-7 release session committing onto
three covered files after the previous verified timestamp: additive
`release`/`release-snapshot` Makefile targets, grpcurl/goreleaser tool
directives, and the go 1.27.4 directive — none touching replication or
simulator logic (the Go source is byte-identical, proven by
`git diff @{2026-11-16T10:49:51}..HEAD` over the covered set). The changed
files were verified at full depth: the phase-4-relevant Makefile targets
(`test`, `lint`) are untouched and both re-proven exit 0; the direct
dependency contract (grpc v1.85.0 + protobuf) is unchanged and the whole
phase-4 test surface was re-run fresh on the new module set — every named
behavioral test PASS, the full race suite PASS, the strict lint gate PASS.
The two open warnings from the 2026-11-18 cross-phase code review (WR-03
syft dependency in the release gate, WR-04 tool-graph visibility to
govulncheck) are phase-7 release-machinery concerns recorded as advisory
here — tracked open in 04-REVIEW-DISPOSITION.md. The phase goal —
convergence to the pointwise supremum over a lossy UDP segment, proven
under chaos by the deterministic simulator — remains achieved in the
codebase at HEAD (v1.0.0). Status is `passed`: no gaps, no outstanding
human-verification items, no flagged prohibitions.

---

_Verified: 2026-11-18T10:47:44Z_
_Verifier: Claude (gsd-verifier)_
