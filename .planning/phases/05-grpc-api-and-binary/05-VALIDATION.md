---
phase: "5"
slug: "grpc-api-and-binary"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-08"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from 05-RESEARCH.md `## Validation Architecture`; the planner's PLAN.md files are authoritative — replace placeholder rows with real executed tasks during validate-phase.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib, -race) — existing repo discipline |
| **Config file** | none — already installed (Makefile `test` target) |
| **Quick run command** | `go test -race ./internal/grpcapi/... ./cmd/sup-kv/...` (phase packages) |
| **Full suite command** | `make test` (go test -race -cover ./...) |
| **gRPC harness** | bufconn in-memory listener (research-verified pattern) for API integration tests |
| **Estimated runtime** | ~30–60 seconds (bufconn suites add stream tests; binary smoke adds one real-socket round trip) |

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
| 05-0N-xx | NN | N | API-01..API-05 | T-05-xx (plan-time) | TBD by planner | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestKV_'` | ❌ W0 | ⬜ pending |
| 05-0N-xx | NN | N | API-03 (Watch cut) | T-05-xx | slow-consumer disconnect, never block store | unit + integration | `go test -race ./internal/grpcapi/ -run 'TestWatch_'` | ❌ W0 | ⬜ pending |
| 05-0N-xx | NN | N | API-04/API-06 (gate+health) | T-05-xx | gate rejects writes; health NOT_SERVING | bufconn | `go test -race ./internal/grpcapi/ -run 'TestGate_|TestHealth_'` | ❌ W0 | ⬜ pending |
| 05-0N-xx | NN | N | API-07 (snapshots via API) | T-05-xx | path traversal closed at persistfile | bufconn | `go test -race ./internal/grpcapi/ -run 'TestSnapshot_'` | ❌ W0 | ⬜ pending |
| 05-0N-xx | NN | N | TOOL-01 | T-05-xx | — | binary smoke (real socket) | `go test -race ./cmd/sup-kv/ -run 'TestBinary_'` | ❌ W0 | ⬜ pending |

*Task IDs are placeholders — the planner's PLAN.md files are authoritative; this map is the sampling contract they must satisfy. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] bufconn test harness exists before API integration tests land
- [ ] `go test -race` green on the empty phase packages from the first commit of each new package

*Existing infrastructure (Makefile, CI matrix, property-test discipline) covers the rest.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| systemd unit behavior end-to-end | TOOL-01 | Type=simple unit + SIGTERM lifecycle on a real host is deployment-environment state; automated coverage stops at signal-handling tests | Run a unit file with Type=simple; send SIGTERM; observe graceful stop (health NOT_SERVING, final confirmed dump) |

*All other phase behaviors have automated verification (bufconn integration + binary smoke).*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
