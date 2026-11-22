---
phase: "5"
slug: "persistence"
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-22"
validated: "2026-11-22"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract. Audited after execution: every task in the 12 plans
> carries an automated verification that ran green on the committed tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` + `go test -race` |
| **Config file** | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `race-soak`) |
| **Quick run command** | `go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./internal/grpcapi/` |
| **Full suite command** | `make lint && make test` (+ `SUPKV_RED_TESTS=1 bash scripts/red-runner`) |
| **Estimated runtime** | ~120–200 s full suite; ~5–30 s per task command |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (the map below)
- **After every plan wave:** `go build ./... && make test`
- **Before phase verification:** full suite green on the committed tree
- **Max feedback latency:** ~200 s (full suite); per-task commands ≤ 30 s

---

## Per-Task Verification Map

> **Population:** the rows below stay the template until execution. The map is filled by the
> validate-phase workflow after the 12 plans have run, from each task's `<verify><automated>`
> block and its recorded status — planning has not run those commands, so a pre-filled row would
> invent a result. The plans' verify blocks are the authoritative per-task command list until
> then.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 1 | PERSIST-{XX} | T-{N}-01 / — | {expected secure behavior or "N/A"} | unit | `{command}` | ✅ / ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `{tests/test_file.py}` — stubs for PERSIST-{XX}
- [ ] `{tests/conftest.py}` — shared fixtures
- [ ] `{framework install}` — if no framework detected

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| {behavior} | PERSIST-{XX} | {reason} | {steps} |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < {N}s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** {pending / approved YYYY-MM-DD}
