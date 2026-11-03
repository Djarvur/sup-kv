---
phase: "4"
slug: "replication-and-deterministic-simulator"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
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
| **Quick run command** | `go test -race ./internal/app/<pkg>/...` (phase packages) |
| **Full suite command** | `make test` (go test -race -cover ./...) |
| **Estimated runtime** | ~30–60 seconds (simulator adds virtual-time tests; smoke test adds a real-socket round trip) |

---

## Sampling Rate

- **After every task commit:** Run the task's scoped `go test -race ./<pkg>/... -run '<TestName>'`
- **After every plan wave:** Run `make test` (and `make lint` before push)
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | REPL-01 | T-04-01 / — | N/A (trusted segment) | integration (virtual net) | `go test -race ./internal/app/<repl>/ -run 'TestHello'` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | REPL-02 | T-04-02 / — | N/A | unit + sim | `go test -race ./internal/app/<repl>/ -run 'TestRetransmit\|TestUnacked'` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | REPL-03, REPL-06 | T-04-03 / — | N/A | unit + sim | `go test -race ./internal/app/<repl>/ -run 'TestAck\|TestPeerTable'` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 2 | REPL-05, REPL-07 | T-04-04 / — | N/A | sim | `go test -race ./internal/app/<repl>/ -run 'TestSync'` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 2 | REPL-04, REPL-08, REPL-09 | T-04-05 / — | dup node_id logged as config error; discard classes distinguishable | unit | `go test -race ./internal/app/<repl>/ -run 'TestDiscard\|TestDataRequest'` | ❌ W0 | ⬜ pending |
| 04-0N-xx | NN | N | TEST-03 | — / — | N/A | sim (bounds + chaos) | `go test -race ./internal/app/<sim>/ -run 'TestSim' -v` | ❌ W0 | ⬜ pending |
| 04-0N-xx | NN | N | TEST-03 (smoke) | — / — | N/A | real-socket smoke | `go test -race ./internal/<adapter>/ -run 'TestSmoke' -v` | ❌ W0 | ⬜ pending |

*Task IDs and package names are placeholders — the planner's PLAN.md files are authoritative; this map is the sampling contract they must satisfy. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Simulator scaffold (virtual clock + virtual network + seeded rng) exists before bound assertions land
- [ ] `go test -race` green on the empty phase packages from the first commit of each new package

*Existing infrastructure (Makefile, CI matrix, property-test discipline in internal/app/internal/testutil) covers the rest.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS loopback broadcast smoke | REPL-01 (smoke variant) | macOS `lo0` has no BROADCAST flag — loopback broadcast is Linux-only; CI cell `macos-15` may skip with a loud documented message after its pre-flight probe | On Linux: run the smoke test; on macOS: confirm the skip message names the fallback tried and prints the documented rationale |

*All other phase behaviors have automated verification (simulator + unit + one real-socket smoke test on Linux).*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
