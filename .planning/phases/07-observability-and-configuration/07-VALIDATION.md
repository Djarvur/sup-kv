---
phase: "7"
slug: "observability-and-configuration"
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-22"
---

# Phase 7 — Validation Strategy

> Per-phase validation contract. Audited after execution: every task in the phase plans
> carries an automated verification that ran green on the committed tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` + `go test -race` |
| **Config file** | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `buf-lint`, `buf-breaking`, `client-example`) |
| **Quick run command** | `go test -race -count=1 ./internal/grpcapi/ ./cmd/sup-kv/ ./api/supkvapi/v1/` |
| **Full suite command** | `make lint && make test && make buf-lint && make buf-breaking && make client-example` (+ `SUPKV_RED_TESTS=1 bash scripts/red-runner`) |
| **Estimated runtime** | ~200–300 s full suite; per-task commands ≤ 60 s |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (the map below)
- **After every plan wave:** `go build ./... && make test`
- **Before phase verification:** full suite green on the committed tree
- **Max feedback latency:** ~300 s (full suite); per-task commands ≤ 60 s

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 1 | OBSV-{XX} | T-{N}-01 / — | per plan threat model | automated | `{command}` | ✅ / ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < {N}s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** {pending / approved YYYY-MM-DD}
