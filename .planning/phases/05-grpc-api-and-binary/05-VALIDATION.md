---
phase: "5"
slug: "grpc-api-and-binary"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: false
created: "2026-11-08"
updated: "2026-11-09"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from 05-RESEARCH.md `## Validation Architecture`; map filled from the executed 05-01..05-06 plans and their green test suites (validate-phase §6, 2026-11-09).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib, -race) — existing repo discipline |
| **Config file** | none — already installed (Makefile `test` target) |
| **Quick run command** | `go test -race ./internal/grpcapi/... ./cmd/sup-kv/...` (phase packages) |
| **Full suite command** | `make test` (go test -race -cover ./...) |
| **gRPC harness** | bufconn in-memory listener (05-01 landed it; all API integration tests ride it) |
| **Estimated runtime** | ~90 seconds (bufconn suites + binary smoke with a real socket round trip) |

---

## Sampling Rate

- **After every task commit:** Run the task's scoped `go test -race ./<pkg>/... -run '<TestName>'`
- **After every plan wave:** Run `make test` (and `make lint` before push)
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-t1 | 05-01 | 1 | API-01 | — | full error-code mapping (OK/NotFound/AlreadyExists/InvalidArgument/Unavailable); memory retention under dump failure (PERS-04); ErrClosed | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestServer_'` | ✅ | ✅ green |
| 05-01-t2 | 05-01 | 1 | API-02 | — | List sorted, tombstone-free, prefix-bounded (D-12..D-15) | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestServer_list'` | ✅ | ✅ green |
| 05-02-t1 | 05-02 | 1 | API-04 (sync half) | — | syncArrived set on first DataSync (A1 accessor); adapter atomic mirror visible over real loopback loop | unit + integration | `go test -race ./internal/app/repl/ -run 'TestSync_syncArrived' && go test -race ./internal/udprepl/ -run 'TestMirror_'` | ✅ | ✅ green |
| 05-02-t2 | 05-02 | 1 | API-05 (Acked seam) | — | Acked sorted/idempotent; eviction never touches acked; PendingSnapshot clones | unit | `go test -race ./internal/app/repl/ -run 'TestRetransmit_ackDrainSemantics'` | ✅ | ✅ green |
| 05-03-t1 | 05-03 | 2 | API-03 (tap) | — | OnChange fires only on actual state change from install; loser absorption stays silent (D-06) | unit | `go test -race ./internal/app/storage/ -run 'TestInstall'` | ✅ | ✅ green |
| 05-03-t2 | 05-03 | 2 | API-03 (Watch) | — | exact snapshot-then-live cut; prefix filter; slow-consumer shed (never blocks store); replicated changes | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestWatch_'` | ✅ | ✅ green |
| 05-04-t1 | 05-04 | 3 | API-04 | — | writes Unavailable during gate, reads served; opens on sync or timeout backstop | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestGate_'` | ✅ | ✅ green |
| 05-04-t2 | 05-04 | 3 | API-06 | — | health mirrors gate (NOT_SERVING until gated open) | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestHealth_'` | ✅ | ✅ green |
| 05-04-t3 | 05-04 | 3 | API-07 | — | snapshots gated as writes; name validation via persistfile (path traversal rejected) | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestSnapshot_'` | ✅ | ✅ green |
| 05-05-t1 | 05-05 | 4 | API-05 | — | aggregate Ack report; zero-peer short-circuit; per-request timeout; never changes write outcome | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestWait_'` | ✅ | ✅ green |
| 05-05-t2 | 05-05 | 4 | API-05 (edges) | — | appearance-before-drain race; eviction-not-Ack; supersession | bufconn integration | `go test -race ./internal/grpcapi/ -run 'TestWait_appearanceRace|TestWait_evictionNotAck|TestWait_superseded'` | ✅ | ✅ green |
| 05-06-t1 | 05-06 | 5 | TOOL-01 | — | `--version` reports injected ldflags version | process smoke | `go test -race ./cmd/sup-kv/ -run 'TestMain_version'` | ✅ | ✅ green |
| 05-06-t2 | 05-06 | 5 | TOOL-01 | — | flag > env > default precedence; malformed SUPKV_ env fatal exit 2 | process smoke | `go test -race ./cmd/sup-kv/ -run 'TestMain_precedence'` | ✅ | ✅ green |
| 05-06-t3 | 05-06 | 5 | TOOL-01 | — | real binary: gate closes, write confirmed, SIGTERM → D-19 shutdown → exit 0 → dump contains key | process smoke | `go test -race ./cmd/sup-kv/ -run 'TestMain_lifecycle'` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] bufconn test harness exists before API integration tests land (landed in 05-01)
- [x] `go test -race` green on the phase packages from the first commit of each new package

*Existing infrastructure (Makefile, CI matrix, buf gates, property-test discipline) covers the rest.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| systemd unit behavior end-to-end | TOOL-01 | Type=simple unit + SIGTERM lifecycle on a real host is deployment-environment state; automated coverage stops at signal-handling tests (TestMain_lifecycle) | Run a unit file with Type=simple; send SIGTERM; observe graceful stop (health NOT_SERVING, final confirmed dump) |

*All other phase behaviors have automated verification (bufconn integration + binary smoke).*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-11-09
