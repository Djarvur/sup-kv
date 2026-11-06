---
phase: "4"
slug: "replication-and-deterministic-simulator"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-03"
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib, -race) — existing repo discipline |
| **Config file** | none — already installed (Makefile `test` target) |
| **Quick run command** | `go test -race ./internal/app/repl/... ./internal/udprepl/...` (phase packages; simulator in `./internal/app/internal/simnet/...`) |
| **Full suite command** | `make test` (go test -race -cover ./...) |
| **Estimated runtime** | ~10–15 seconds full suite (virtual-time simulator tests + one real-socket smoke round trip at ~0.02s) |

---

## Sampling Rate

- **After every task commit:** Run the task's scoped `go test -race ./<pkg>/... -run '<TestName>'`
- **After every plan wave:** Run `make test` (and `make lint` before push)
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

Task IDs are the executed PLAN.md tasks (04-01..04-06); every command below was re-run
green under `-race -count=1` during the validation audit (2026-11-06).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-T1 | 04-01 | 1 | REPL-03 | T-04-01 / T-04-SC | N/A (trusted segment); CRC-gated decode | integration (real proto round trip) | `go test -race ./internal/udprepl/ -run '^TestWire_twoNodeHandoff$' -v` | ✅ | ✅ green |
| 04-02-T1 | 04-02 | 2 | REPL-02 | T-04-05 / T-04-06 | N/A | unit | `go test -race ./internal/app/repl/ -run '^TestRetransmit_ackDrainSemantics$' -v` | ✅ | ✅ green |
| 04-02-T2 | 04-02 | 2 | REPL-02 | T-04-05 / T-04-06 | N/A | unit | `go test -race ./internal/app/repl/ -run '^TestRetransmit_backoffLadder$\|^TestRetransmit_suspendAndFlush$' -v` | ✅ | ✅ green |
| 04-02-T3 | 04-02 | 2 | REPL-06 | T-04-05 | N/A | unit | `go test -race ./internal/app/repl/ -run '^TestPeerTable_' -v` | ✅ | ✅ green |
| 04-03-T1 | 04-03 | 3 | REPL-04 | T-04-07 | replies unicast-only (no broadcast amplification) | unit | `go test -race ./internal/app/repl/ -run '^TestDataRequest_replyAndSilence$' -v` | ✅ | ✅ green |
| 04-03-T2 | 04-03 | 3 | REPL-05 | T-04-07 / T-04-08 | oversized key ships alone, never dropped | unit + cross-seam | `go test -race ./internal/app/repl/ -run '^TestSync_chunking$\|^TestSync_triggers$' -v && go test -race ./internal/udprepl/ -run '^TestSync_chunkWireBudget$' -v` | ✅ | ✅ green |
| 04-03-T3 | 04-03 | 3 | REPL-05 | T-04-07 | single rate limiter over all reactive unicasts | unit | `go test -race ./internal/app/repl/ -run '^TestSync_compare$\|^TestSync_rateLimit$' -v` | ✅ | ✅ green |
| 04-04-T1 | 04-04 | 4 | REPL-07 | T-04-09 | listing-only reaction (values never listed) | unit | `go test -race ./internal/app/repl/ -run '^TestRestart_runIDChangeTriggersFullSync$\|^TestRestart_newPeerNoFullSync$' -v` | ✅ | ✅ green |
| 04-04-T2 | 04-04 | 4 | REPL-07 | T-04-09 | N/A | unit | `go test -race ./internal/app/repl/ -run '^TestRestart_staleRunIdAndInFlight$' -v` | ✅ | ✅ green |
| 04-05-T1 | 04-05 | 5 | TEST-03 | T-04-10 / T-04-11 | N/A | unit (simulator self-tests) | `go test -race ./internal/app/internal/simnet/ -v` | ✅ | ✅ green |
| 04-05-T2 | 04-05 | 5 | REPL-10 | T-04-10 | N/A | integration (virtual net) | `go test -race ./internal/app/repl/ -run '^TestBound_' -v` | ✅ | ✅ green |
| 04-05-T3 | 04-05 | 5 | TEST-03 | T-04-10 / T-04-11 | N/A | integration (seeded chaos; `SUPKV_SIM_SEED` replay verified) | `go test -race ./internal/app/repl/ -run '^TestChaos_' -v` | ✅ | ✅ green |
| 04-06-T1 | 04-06 | 5 | REPL-08, REPL-09 (+ REPL-03 memory-apply seam) | T-04-02 / T-04-03 / T-04-04 | dup node_id logged as configuration error; five discard classes distinguishable and total; ApplyRemote errors surfaced, never panic | unit | `go test -race ./internal/app/repl/ -run '^TestDiscard_' -v && go test -race ./internal/app/persist/ -run '^TestEngine_applyRemoteTriggersDump$' -v` | ✅ | ✅ green |
| 04-06-T2 | 04-06 | 5 | REPL-01 | T-04-02 | zero-config discovery only (no static peer lists) | smoke (real sockets) | `go test -race ./internal/udprepl/ -run '^TestSmoke_twoNodesUDP$' -v` | ✅ | ✅ green (PASS, no SKIP, on Linux) |

Requirement coverage closed across the map: REPL-01..REPL-10 and TEST-03 each resolve to
at least one green automated command above. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky

---

## Wave 0 Requirements

- [x] Simulator scaffold (virtual clock + virtual network + seeded rng) exists before bound assertions land — simnet scaffold commit `4a6f777` precedes the bound-tests commit `ee83af5` (ancestry verified)
- [x] `go test -race` green on the phase packages from the first commit of each new package — `9690b4a` (repl + udprepl, 04-01 tracer green) and `4a6f777` (simnet, three-test suite green); full repo `make test` green at audit time

*Existing infrastructure (Makefile, CI matrix, property-test discipline in internal/app/internal/testutil) covers the rest.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS loopback broadcast smoke | REPL-01 (smoke variant) | macOS `lo0` has no BROADCAST flag — loopback broadcast is Linux-only; CI cell `macos-15` may loud-skip with a documented message after its pre-flight probe (probe → subnet-broadcast fallback → loud skip naming both addresses) | On Linux: run `TestSmoke_twoNodesUDP` (runs, asserts discovery + visibility + OwnEcho); on macOS: confirm the skip message names both attempted broadcast addresses and prints the documented rationale — record the first macOS CI outcome for Phase 6 |

*All other phase behaviors have automated verification (simulator + unit + one real-socket smoke test on Linux).*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s (per-task commands ~1s; full suite ~10–15s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated (audit 2026-11-06: every mapped command re-run green under `-race -count=1`; `make test` and `make lint` green on branch `gsd/phase-4-replication-and-deterministic-simulator`)

## Validation Audit 2026-11-06

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
