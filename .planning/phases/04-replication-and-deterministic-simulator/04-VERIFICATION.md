---
phase: 04-replication-and-deterministic-simulator
verified: 2026-11-14T22:35:00Z
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
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/core.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sync_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replproto/supkv/v1/supkv.pb.go
  - internal/udprepl/replproto/supkv/v1/supkv.proto
covered_digest: "v4:sha256:b898d6b8a0cd343a166042701ec06bf6959a3bcbab5998a0bc385c7b82237bf3"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 5/5
  gaps_closed: [] # staleness regeneration, not gap closure — the buf zero-waiver refactor 7b55ad1 (import path in repl.go; replproto tree moved to supkv/v1 with a regenerated pb.go) and the toolchain bump c194c6f (go1.27.4, GO-2026-6859) changed covered files after the last verified timestamp 2026-11-14T19:45:24Z
  gaps_remaining: []
  regressions: []
---

# Phase 4: Replication and Deterministic Simulator Verification Report

**Phase Goal:** Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment, with convergence under chaos proven by the deterministic simulator
**Verified:** 2026-11-14T22:35:00Z
**Status:** passed
**Re-verification:** Yes — staleness regeneration (06-CONTEXT.md D-16 folds this re-run into the phase-6 execution flow). The previous report (verified 2026-11-14T19:45:24Z, digest `a14f92aa…`) went stale when two post-timestamp commits touched covered files: `7b55ad1` (2026-11-14T20:07:49Z — buf zero-waiver refactor: the replproto tree moved to `internal/udprepl/replproto/supkv/v1/`, repl.go's import updated, Makefile/buf.yaml restructured) and `c194c6f` (2026-11-14T22:22:42Z — toolchain go1.27.4 closing GO-2026-6859/CVE-2026-97054; the `go 1.27.1` language pin unchanged). `git log --since` confirms these are the ONLY covered-file changes since the previous pass. This re-verification examines every changed file at full depth and regression-checks the unchanged set with fresh named tests.

## Goal Achievement

### Observable Truths

The five ROADMAP success criteria are the contract; the 42 plan-level truths
from the six PLANs refine these five. All five were re-verified against the
CURRENT codebase with fresh behavioral evidence — every behavior-dependent
truth has a named passing test from this pass (commands below).

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|-----------------------------------|--------|----------|
| 1 | Two (or N) nodes discover each other via broadcast Hello with no configuration beyond `broadcast_addr`, and a write on one node becomes visible on every other within one RTT in normal operation | ✓ VERIFIED | `TestSmoke_twoNodesUDP` fresh PASS this pass (1.03s, real sockets, zero config — adapter `New` signature re-read at repl.go:198: `(bindAddr, broadcastAddr string, nodeID uint16, store repl.StateStore, cfg repl.Config)`, `Config` re-read: timing knobs + Log only, no peer lists or static addressing anywhere); `TestBound_normalOneRTT` fresh PASS; `TestWire_twoNodeHandoff` fresh PASS through the MOVED proto tree |
| 2 | Every local change is broadcast and tracked per-peer (key, version) until Ack or supersession, retransmitting with exponential backoff and suspending (not dropping) while no peer is known; receivers Ack after applying to memory; the unacked set drains after a partition heals | ✓ VERIFIED | core.go unchanged since the previous full-depth pass (last commit af3d5c4, pre-timestamp) — regression-checked with FRESH runs: `TestRetransmit_ackDrainSemantics`, `TestRetransmit_suspendAndFlush`, `TestRetransmit_backoffLadder`, `TestPeerTable_lifecycle` all PASS; `TestBound_partitionHeal` fresh PASS |
| 3 | Full reconciliation works in both directions: chunked DataSync on the sync timer, at startup and on neighbor restart; receivers answer with DataRequest or unicast DataReplication rate-limited by `sync_response_rate`; a DataRequest for an unknown key gets no reply; a Hello with a new `run_id` triggers full sync toward the restarted node | ✓ VERIFIED | `TestSync_rateLimit` + `TestSync_syncArrived` fresh PASS (floor(1s/rate) spacing pinned); `TestRestart_runIDChangeTriggersFullSync` fresh PASS; `TestSync_chunkWireBudget` fresh PASS against the real codec over the moved schema — the wire path through `replproto/supkv/v1` is intact end-to-end |
| 4 | Malformed traffic is dropped with distinguishable classes: duplicate `node_id` logged as a configuration error, plus own broadcast echo, bad CRC and bad proto version each identifiable | ✓ VERIFIED | Classifier unchanged (core.go); `TestDiscard_classes` + `TestDiscard_totality` (500 adversarial datagrams) fresh PASS; own-echo counted in the fresh smoke run (OwnEcho ≥ 1 asserted on a real Linux socket) |
| 5 | The deterministic simulator runs 2..N nodes over a seeded virtual network that loses, duplicates, reorders and partitions datagrams, and asserts convergence within the documented bounds — normal one RTT, datagram loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval` + response time — printing the seed on failure | ✓ VERIFIED | simnet byte-identical since 2026-11-08 (zero commits since — `git log` empty); fresh seeded replay `SUPKV_SIM_SEED=4242 TestChaos_convergence` PASS (1.19s); all three `TestBound_*` fresh PASS; full race suite green |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Plan-Level Truth Detail

| Plan | Truths | Check this pass |
|------|--------|-----------------|
| 04-01 (REPL-03) | 6/6 | Wire schema verified at its NEW path (`replproto/supkv/v1/`): proto diff vs pre-move shows exactly ONE changed line (go_package gains `/supkv/v1`, `;replproto` alias preserved — field names/numbers untouched, wire format unchanged); pb.go is a clean regen (98 changed lines, ALL mechanical descriptor-path renames; DO-NOT-EDIT marker at the re-generated header, protoc-gen-go v1.36.12-devel, source `supkv/v1/supkv.proto`); `TestWire_twoNodeHandoff` fresh PASS |
| 04-02 (REPL-02/06) | 8/8 | core.go unchanged since previous full-depth pass; 4 named tests fresh PASS |
| 04-03 (REPL-04/05) | 8/8 | Backstop truth re-proven with fresh behavioral evidence: `TestSync_rateLimit` PASS; IDEA.md §Parameters echo re-confirmed ("sync responses spaced ⌊1s/rate⌋", grep = 1) |
| 04-04 (REPL-07) | 8/8 | `TestRestart_runIDChangeTriggersFullSync` fresh PASS; restart branch unchanged in core.go |
| 04-05 (TEST-03/REPL-10) | 6/6 | simnet + bound/chaos tests unchanged (no commits since 2026-11-08); seeded replay re-run fresh — PASS |
| 04-06 (REPL-01/08/09) | 6/6 | Discard classifier unchanged; fresh `TestSmoke_twoNodesUDP` + `TestDiscard_*` PASS; persist seam fresh PASS (`TestEngine_applyRemoteTriggersDump`, `TestFlush_*`) |

### Changed-Area Verification: the staleness commits on Phase-4 files

Every covered file that changed since 2026-11-14T19:45:24Z, at full depth:

- **`internal/udprepl/repl.go`** (7b55ad1) — the whole-file diff against the
  pre-commit state is exactly ONE line: the import
  `internal/udprepl/replproto` → `replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"`
  (explicit alias, so every `replproto.Xxx` reference compiles unchanged —
  zero churn beyond the import). Level 3 wiring re-grepped: the single event
  loop still calls `Advance`/`HandleDatagram`/`NotifyLocalWrite` at lines
  270/282/294/298 only; no second driver added. Level 4: encode/decode flow
  through the moved generated types — proven live by fresh
  `TestWire_twoNodeHandoff` + `TestSync_chunkWireBudget` + `TestSmoke_twoNodesUDP`
  + `TestMirror_*` PASSes. No handler, mirror, codec, timer or lifecycle
  logic touched.
- **`internal/udprepl/replproto/supkv/v1/supkv.proto`** (renamed from
  `internal/udprepl/replproto/supkv.proto`, R098) — byte-identical to the
  pre-move file apart from exactly one line: go_package gains the `/supkv/v1`
  tree segment while preserving the `;replproto` package alias. Message,
  field, and number inventory untouched — the UDP wire format is unchanged
  (same bytes, same CRC domain). 74 lines.
- **`internal/udprepl/replproto/supkv/v1/supkv.pb.go`** (renamed +
  regenerated, R081) — clean mechanical regen verified by diff against the
  pre-move file: 98 changed lines, every one a path/descriptor rename
  (`file_internal_udprepl_replproto_supkv_proto*` → `file_supkv_v1_supkv_proto*`,
  the source-comment line, the embedded raw descriptor's path and
  go_package strings). Provenance intact: `Code generated by
  protoc-gen-go. DO NOT EDIT.` header, same generator stamp
  (v1.36.12-devel), `package replproto` preserved. 660 lines, 8-message
  schema unchanged. `TestCodec_*` family green in the full-suite run.
- **`Makefile` / `buf.yaml`** (7b55ad1) — buf codegen restructured per tree:
  `make proto` runs `go tool buf generate internal/udprepl/replproto
  --template buf.gen.repl.yaml` (plus the api tree); buf.yaml now carries
  two module roots (one per proto tree) with `use: [STANDARD]` and ZERO
  waivers — the former ignore_only keys are gone (remaining "waiver"
  mentions in the file are explanatory comments only). `go tool buf lint`
  exit 0, `go tool buf format --diff --exit-code` exit 0 this pass.
- **`go.mod`** (c194c6f) — `toolchain go1.27.4` added, closing
  GO-2026-6859/CVE-2026-97054 (security benefit); the `go 1.27.1` language
  pin unchanged (AGENTS.md constraint respected — no ADR needed for a
  toolchain-only bump). `make test` and `make lint` both exit 0 on this
  toolchain.
- **Stale-path check** — zero references to the old import path
  `internal/udprepl/replproto"` remain anywhere in the tree (grep = 0); the
  only consumer (udprepl/repl.go:32) points at the new path. Nothing
  orphaned at the old location.

### Required Artifacts

All artifacts across the six plans exist, are substantive, and are wired.
(`gsd-tools query verify.artifacts` returns 0/0 on every plan — the known
parser limitation with string-form artifact entries documented in every
previous pass; manual Level 1-4 verification stands.)

| Artifact | Lines | Status | Wiring |
|----------|-------|--------|--------|
| `internal/port/repl.go` | 140 | ✓ VERIFIED | Unchanged (no commits since 2026-11-08); the seam both sides speak |
| `internal/app/repl/core.go` | 983 | ✓ VERIFIED | Unchanged since previous full-depth pass (af3d5c4); purity greps re-run: `go func`/`sync.`/`time.Now`/`time.Sleep` = 0; protobuf in internal/app = 0 files |
| `internal/udprepl/repl.go` | 719 | ✓ VERIFIED | Changed (import-only) — re-verified at full depth this pass; single event loop; import points at the moved tree |
| `internal/udprepl/replproto/supkv/v1/supkv.proto` | 74 | ✓ VERIFIED | MOVED (was `internal/udprepl/replproto/supkv.proto`) — byte-identical apart from go_package; buf lint/format green in the zero-waiver workspace |
| `internal/udprepl/replproto/supkv/v1/supkv.pb.go` | 660 | ✓ VERIFIED | MOVED + regenerated — clean mechanical regen with provenance marker intact; consumed by repl.go (the sole importer) |
| `internal/app/repl/{retransmit,peertable,sync,restart,sim_bound,sim_chaos,discard}_test.go` | 715+235+737+330+267+291+412 | ✓ VERIFIED | All green in the full race-suite run; retransmit/sync re-run fresh |
| `internal/udprepl/syncsize_internal_test.go` | 149 | ✓ VERIFIED | Unchanged; fresh PASS (`TestSync_chunkWireBudget`) |
| `internal/app/internal/simnet/simnet.go` + `simnet_test.go` | 487 + 218 | ✓ VERIFIED | Unchanged (zero commits since 2026-11-08); test-only linkage re-proven — 0 non-test importers; purity greps 0 |
| `internal/app/persist/engine.go` | 587 | ✓ VERIFIED | Unchanged since previous pass (86d33a6); `ApplyRemote` matches `repl.StateStore`; fresh `TestEngine_applyRemoteTriggersDump` + `TestFlush_*` PASS |
| `internal/app/persist/engine_test.go` | 1473 | ✓ VERIFIED | Unchanged; green |
| `internal/udprepl/smoke_test.go` | 331 | ✓ VERIFIED | Unchanged; fresh PASS this pass (ran, not skipped — 1.03s) |

### Key Link Verification

All key links WIRED; links touching changed files re-verified by direct code
read this pass, the rest regression-checked with fresh tests:

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| port.Message structs | replproto generated types (moved tree) | `protoBody`/`portMessage` mapping | ✓ WIRED | Import path updated (repl.go:32); `TestWire_twoNodeHandoff` + `TestSync_chunkWireBudget` fresh green through the moved schema |
| udprepl event loop | repl.Core handlers | Only `Run`'s loop calls HandleDatagram/NotifyLocalWrite/Advance | ✓ WIRED | Re-grepped in the changed file: lines 270, 282, 294, 298 — the import-only delta adds no second driver |
| NotifyLocalWrite | pending entry + broadcast | Wait-set seeded from peer table, ladder armed, broadcast emitted | ✓ WIRED | core.go unchanged; `TestNotifyLocalWrite_afterRunReturns` fresh PASS |
| inbound Ack | wait-set drain | `handleAck` exact-version drain + sorted-idempotent `ackedInsert` | ✓ WIRED | `TestRetransmit_ackDrainSemantics` fresh PASS |
| Advance | retransmit + eviction scans | Single loop fires both, sorted by key | ✓ WIRED | `TestRetransmit_backoffLadder` + `TestPeerTable_lifecycle` fresh PASS |
| inbound DataRequest | rate-limited reply | Snapshot lookup → `enqueueResponse` → `drainEgress` | ✓ WIRED | `TestSync_rateLimit` fresh PASS |
| startup/timer/restart | chunked listing emission | One `listingChunks` build serves both triggers | ✓ WIRED | `TestRestart_runIDChangeTriggersFullSync` fresh PASS |
| simnet scheduler | Core handlers | `deliver`→HandleDatagram, `drive`→Advance | ✓ WIRED | simnet unchanged; seeded chaos replay fresh PASS |
| simnet network | Outbox drain | `route` sends every Outbound; Broadcast fans out incl. sender | ✓ WIRED | simnet unchanged |
| per-node store | sim clock | `clock.NewClock(sched.Now, ...)` | ✓ WIRED | simnet unchanged |
| observability seams | quiescence | `PendingDeliveries` + `UnackedStats().Count` + `QueuedResponses()` | ✓ WIRED | Exercised by the fresh seeded chaos run |
| persist.Engine.ApplyRemote | repl.StateStore | Signature match | ✓ WIRED | `TestEngine_applyRemoteTriggersDump` fresh PASS |
| adapter atomics | smoke test / Phase-5 readers | `KnownPeers`/`OwnEcho`/`Discards()` + `SyncArrived`/`PendingCopy` mirrored post-event | ✓ WIRED | `TestMirror_*` fresh PASS |

### Data-Flow Trace (Level 4)

No rendered/static-value surfaces in this backend phase; the data-flow
equivalents all trace to real sources:

| Path | Source | Status |
|------|--------|--------|
| Replicated records | `store.ApplyRemote` → real `storage.Store.Merge` (LWW join) | ✓ FLOWING |
| DataSync listings | `store.Records()` snapshot per build | ✓ FLOWING |
| Chaos model store | `WriteLog` merged by real `Merge` (simnet unchanged) | ✓ FLOWING |
| Discard/peer/fault counters | Core state mutated by real handler calls; mirrored to atomics post-event | ✓ FLOWING |
| Pending mirror | `core.PendingSnapshot()` cloned per iteration into the copy-on-write map; drained rows from real Ack correlation | ✓ FLOWING |
| Wire encode/decode (moved tree) | Real proto Marshal/Unmarshal over the moved generated types — exercised live by the fresh wire, chunk-budget and smoke runs | ✓ FLOWING |

### Behavioral Spot-Checks

Single-test discipline; every command run fresh (`-count=1`) this pass. The
full suite ran exactly once (`make test`, exit 0, race detector, coverage,
all packages ok); `make lint` ran once (exit 0); `buf lint` + `buf format
--diff --exit-code` ran once (both exit 0, zero waivers).

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Real-socket two-node discovery (REPL-01) | `go test -race -count=1 -run 'TestSmoke_twoNodesUDP' ./internal/udprepl/` | PASS (1.03s) | ✓ PASS |
| Seeded determinism replay (D-06) | `SUPKV_SIM_SEED=4242 go test -race -count=1 -run 'TestChaos_convergence' ./internal/app/repl/` | PASS (1.19s) | ✓ PASS |
| Wire handoff + cross-seam chunk budget over the MOVED proto | `go test -race -count=1 -run 'TestWire_twoNodeHandoff\|TestSync_chunkWireBudget' ./internal/udprepl/` | PASS (1.02s) | ✓ PASS |
| Ack drain / suspension / ladder / peer lifecycle (REPL-02/06) | `go test -race -count=1 -run 'TestRetransmit_ackDrainSemantics\|TestRetransmit_suspendAndFlush\|TestRetransmit_backoffLadder\|TestPeerTable_lifecycle' ./internal/app/repl/` | PASS (1.02s) | ✓ PASS |
| Rate limiter spacing (backstop truth) + sync-arrived | `go test -race -count=1 -run 'TestSync_rateLimit\|TestSync_syncArrived' ./internal/app/repl/` | PASS (1.01s) | ✓ PASS |
| Convergence bounds (REPL-10) | `go test -race -count=1 -run 'TestBound_normalOneRTT\|TestBound_partitionHeal\|TestBound_lossWithinAckTimeoutMax' ./internal/app/repl/` | PASS (1.03s) | ✓ PASS |
| Discard classification + totality (REPL-08/09) | `go test -race -count=1 -run 'TestDiscard_classes\|TestDiscard_totality' ./internal/app/repl/` | PASS (1.05s) | ✓ PASS |
| Restart full sync (REPL-07) | `go test -race -count=1 -run 'TestRestart_runIDChangeTriggersFullSync' ./internal/app/repl/` | PASS (1.02s) | ✓ PASS |
| Adapter mirror seams | `go test -race -count=1 -run 'TestMirror_pendingAckVisible\|TestMirror_syncArrivedFlip\|TestNotifyLocalWrite_afterRunReturns' ./internal/udprepl/` | PASS (1.05s) | ✓ PASS |
| Persist Flush + ApplyRemote seam | `go test -race -count=1 -run 'TestFlush_remoteDirtiness\|TestFlush_clean\|TestFlush_closed\|TestEngine_applyRemoteTriggersDump' ./internal/app/persist/` | PASS (1.02s) | ✓ PASS |
| Full workspace, race detector | `make test` (`go test -race -cover ./...`) | exit 0, all packages ok | ✓ PASS |
| Strict lint gate | `make lint` (config verify + golangci-lint strict) | exit 0, zero findings | ✓ PASS |
| Buf zero-waiver workspace (staleness trigger re-proven green) | `go tool buf lint` + `go tool buf format --diff --exit-code` | both exit 0; buf.yaml `use: [STANDARD]`, no ignore_only keys | ✓ PASS |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` exist or are referenced by
any PLAN/SUMMARY (unchanged finding).

### Requirements Coverage

Plan frontmatter union: REPL-01,02,03,04,05,06,07,08,09,10 + TEST-03 —
exactly the 11 IDs REQUIREMENTS.md assigns to Phase 4 (all marked Complete
in its traceability table). No orphaned requirements, no unclaimed IDs.

| Requirement | Source Plan | Status | Evidence |
|-------------|------------|--------|----------|
| REPL-01 | 04-06 | ✓ SATISFIED | `TestSmoke_twoNodesUDP` fresh PASS (real sockets, zero config; adapter `New` + `Config` re-read — no peer-list/static-address surface) |
| REPL-02 | 04-02 | ✓ SATISFIED | Tracking/backoff/suspension/flush unchanged in core.go; 4 named tests fresh PASS |
| REPL-03 | 04-01 | ✓ SATISFIED | `TestWire_twoNodeHandoff` fresh PASS through the moved proto tree; Ack-after-apply unchanged |
| REPL-04 | 04-03 | ✓ SATISFIED | handleDataRequest unchanged; `TestDataRequest_replyAndSilence` green in the full suite |
| REPL-05 | 04-03 | ✓ SATISFIED | Chunked listing + limiter unchanged; `TestSync_rateLimit` fresh PASS; IDEA.md §Parameters echo intact |
| REPL-06 | 04-02 | ✓ SATISFIED | `TestPeerTable_lifecycle` fresh PASS |
| REPL-07 | 04-04 | ✓ SATISFIED | `TestRestart_runIDChangeTriggersFullSync` fresh PASS |
| REPL-08 | 04-06 | ✓ SATISFIED | Both duplicate variants drop+log+count (classifier unchanged); `TestDiscard_classes` fresh PASS |
| REPL-09 | 04-06 | ✓ SATISFIED | `TestDiscard_totality` (500 adversarial datagrams) fresh PASS; own-echo counted in the fresh smoke run |
| REPL-10 | 04-05 | ✓ SATISFIED | All three `TestBound_*` fresh PASS |
| TEST-03 | 04-05 | ✓ SATISFIED | Chaos convergence + determinism — simnet unchanged; fresh seeded replay PASS |

### Anti-Patterns Found

Debt-marker gate: zero `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`
across all covered files (changed and unchanged), re-grepped this pass.
The four open review findings (04-REVIEW-DISPOSITION.md) are re-confirmed
present in code at the cited sites this pass; NONE breaks a Phase-4 truth,
artifact, or key link — all four are folded into the Phase-6 plans by the
maintainer's D-15 decision ("Влить всё", 06-CONTEXT.md:47), which names
each finding and its planned fix.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/udprepl/repl.go | 513-516 | `retainDrained` union `clone(prev.Acked)+prev.WaitingOn` sorted without `slices.Compact` — duplicate node id in a published drained row (WR-01; unchanged by the import-only delta, re-read this pass) | ⚠️ Warning | Mirror observability only; core pending set drains correctly; Phase-6 fix planned (D-15: `slices.Compact` after sort) |
| internal/app/repl/core.go | 122-124, 285 | `normalized()` guards only `SyncResponseRate <= 0`; integer `time.Second/rate` → 0 for rate > 1e9 (WR-02) | ⚠️ Warning | Config-domain edge (default 500); Phase-6 fix planned (D-15: clamp out-of-domain rate) |
| internal/udprepl/repl.go | 314-339 | `NotifyLocalWrite` done-channel pre-check TOCTOU window during Run's exit (IN-01; re-read this pass) | ℹ️ Info | Bounded; Phase-6 fix planned (D-15) |
| internal/app/persist/engine.go | 491-498 | `Restore` marks dirty unconditionally → one redundant covering dump (IN-02) | ℹ️ Info | Redundant save only; Phase-6 fix planned (D-15, merged with P3-IN-02) |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Verdict |
|-----------|-----------|--------|---------|----------|---------|
| internal/udprepl/smoke_test.go | REPL-01, REPL-09 | yes (Linux run this pass; macOS loud-skip by ratified D-08 design) | conditional-by-design | no | VALID |
| internal/udprepl/repl_test.go | REPL-03 (wire) | yes | 0 | no | VALID |
| internal/app/repl/sim_chaos_test.go | TEST-03, REPL-10 | yes | 0 | no (expected values from the supremum model, not the SUT) | VALID |
| internal/app/repl/{retransmit,peertable,sync,restart,discard,sim_bound}_test.go | REPL-02..10 | yes | 0 | no | VALID |
| internal/app/persist/engine_test.go | REPL-03 (persist seam) | yes | 0 | no | VALID |

**Disabled tests on requirements:** 0. **Circular patterns detected:** 0.

### Prohibition Disposition (ADR-550 D4)

| Prohibition | Tier | Verdict |
|-------------|------|---------|
| REPL-01 (04-01, 04-06): no config surface beyond broadcast_addr | test | ENFORCED — adapter `New` (repl.go:198) and `Config` re-read on the changed file: timing knobs + Log only, no peer lists/static addressing anywhere |
| REPL-08 (04-06): duplicate node_id never silently resolved | test | ENFORCED — both variants drop + log + count; `TestDiscard_classes` fresh PASS |
| 04-02: pending never dropped on timeout/eviction/empty table | test | ENFORCED — only Ack/supersession delete (unchanged core.go); `TestRetransmit_suspendAndFlush` fresh PASS |
| 04-02/D-05: retransmission never blocks a local write | test | ENFORCED — enqueue-and-return; `TestNotifyLocalWrite_afterRunReturns` fresh PASS |
| 04-02/REPL-06: no blacklist, no peer-count cap | test | ENFORCED — evicted peers re-learned; unbounded node_id-keyed map (unchanged) |
| 04-03: no bypass of the rate-limited egress queue | test | ENFORCED — exactly three producer sites call `enqueueResponse` (unchanged core.go) |
| 04-03/A4: chunks carry no continuation state | test | ENFORCED — self-sufficient slices (unchanged) |
| 04-03/REPL-04: replies unicast only | test | ENFORCED — unicast/queued unicast only (unchanged) |
| 04-04: no second peer row on run_id change | test | ENFORCED — upsert keyed by node_id; `TestRestart_staleRunIdAndInFlight` green in full suite |
| 04-04: full-sync reaction only on run_id change | test | ENFORCED — inequality is the only trigger; `TestRestart_newPeerNoFullSync` green in full suite |
| 04-05: no second protocol implementation in simnet | test | ENFORCED — real Core handlers; 0 non-test importers (re-grepped) |
| 04-05: no wall-clock dependence | test | ENFORCED — `time.Sleep`/`time.Now` in core.go and simnet.go: 0 matches (re-grepped) |
| 04-05/D-06: sim tooling not in production binary | test | ENFORCED — simnet imported only from `_test.go` files (re-grepped: 0 non-test importers) |
| REPL-09 CRC wording (04-01, 04-06) | judgment | **RATIFIED — resolved** (04-UAT.md item 1, human ratification 2026-11-06/07; UAT status complete). Regression grep this pass: every production mention still says corruption detection only, never tamper protection (repl.go:570, replproto/supkv/v1/supkv.pb.go:33, storage/store.go:122, storage/doc.go:35). No flagged prohibitions remain |

### Human Verification Required

None outstanding. Infrastructure/foundation phase — no user-facing
elements; all acceptance criteria verifiable programmatically. The previous
report's two items remain resolved with recorded evidence: the CRC-wording
judgment prohibition is ratified by human UAT (04-UAT.md, status complete,
2026-11-06/07) with the wording re-grepped unchanged on the current tree
(including the moved pb.go), and D-08's macOS smoke outcome is empirically
settled by CI (post-fix run 42514571899, `gate (macos-15)` success). No
PRESENT_BEHAVIOR_UNVERIFIED truths — every behavior-dependent truth has a
fresh passing named test this pass.

### Gaps Summary

None. The staleness trigger was purely mechanical: the buf zero-waiver
refactor (7b55ad1) moved the replproto tree into its package-derived
directory `internal/udprepl/replproto/supkv/v1/` (proto byte-identical
apart from go_package; pb.go a clean regen with provenance intact) and
updated exactly one import line in repl.go, and the toolchain bump
(c194c6f, go1.27.4) closed CVE-2026-97054 without touching the language
pin. Every changed file was re-verified at all four levels with fresh named
tests; every unchanged covered file is identical to what previous passes
verified and was regression-checked with fresh test runs (11 named-test
commands plus one full race-suite run, the strict lint gate, and the buf
lint/format zero-waiver gates — all green). The Phase-4 contract is intact
on the current tree: the wire format through the moved schema is unchanged
and freshly proven end-to-end (real sockets, real codec, seeded chaos
replay), the protocol core is byte-identical to the previously verified
state, and the four open review findings are unchanged, bounded, and
scheduled into Phase 6 by the maintainer's D-15 decision. The phase goal —
convergence to the pointwise supremum over a lossy UDP segment, proven
under chaos by the deterministic simulator — remains achieved in the
codebase. Status is `passed`: no gaps, no outstanding human-verification
items, no flagged prohibitions.

---

_Verified: 2026-11-14T22:35:00Z_
_Verifier: Claude (gsd-verifier)_
