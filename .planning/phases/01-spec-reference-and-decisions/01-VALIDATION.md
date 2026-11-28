---
phase: "1"
slug: "spec-reference-and-decisions"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-19"
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | shell gate commands (git/grep/awk assertions) + `go build`/`make lint`/`make test` at the phase gate |
| **Config file** | Makefile (lint/test targets); no test framework install needed |
| **Quick run command** | `grep -cE '^\| S-' docs/SPEC-DEVIATIONS.md` (registry completeness) |
| **Full suite command** | `make lint && make test` |
| **Estimated runtime** | ~90 seconds (lint ~30s, race test suite ~60s) |

---

## Sampling Rate

- **After every task commit:** run the task's own `<automated>` gate commands (sub-second each)
- **After every plan wave:** build gate (`go build ./...`) + test gate (`make test`)
- **Before `/gsd:verify-work`:** full suite must be green
- **Max feedback latency:** 90 seconds (phase gate task only; per-task gates are sub-second)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-1 | 01 | 1 | SPEC-01 | T-01-01 | non-public spec material never staged/committed | shell gate | `git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` + tracked/porcelain count gates | ✅ | ✅ green |
| 01-01-2 | 01 | 1 | SPEC-01 | — | stale pre-move path references swept | shell gate | `grep -rn '2026-11-19-v1.0-review' .planning/{PROJECT,ROADMAP,REQUIREMENTS}.md \| awk 'END{print NR; exit (NR>0)}'` | ✅ | ✅ green |
| 01-01-3 | 01 | 1 | SPEC-01 | — | IDEA.md body byte-untouched (banner-only) | shell gate | `git diff --numstat -- docs/IDEA.md` (added>0, deleted==0) + banner greps | ✅ | ✅ green |
| 01-02-1 | 02 | 2 | SPEC-01 | T-01-02 | registry completeness and own-words discipline | shell gate | row count ==167; per-prefix contiguity awk; 13 section headings | ✅ | ✅ green |
| 01-02-2 | 02 | 2 | SPEC-01 | — | section-7 closures carry REQ tokens; enum-only statuses | shell gate | 7 unique closure REQ tokens; consumer-zone ≥4; deferred-v2(SNAP-01) ≥1 | ✅ | ✅ green |
| 01-02-3 | 02 | 2 | SPEC-01 | T-01-02 | no verbatim spec statements (D-06) | shell gate | empty-status awk; off-enum awk; verbatim-substring awk over the trace | ✅ | ✅ green |
| 01-03-1 | 03 | 2 | SPEC-02 | — | ADRs 0001–0006 with 14.x frontmatter mapping | shell gate | file count ==6; `review-decision:` present; D-11 loud line in 0005 | ✅ | ✅ green |
| 01-03-2 | 03 | 2 | SPEC-02 | — | ADRs 0007–0011; honest 14.11 gap | shell gate | file count ==5; D-11 loud line in 0007; gap note in 0011 | ✅ | ✅ green |
| 01-03-3 | 03 | 2 | SPEC-02 | — | ADR set mapping complete | shell gate | 11 files; 11 unique 14.x; exactly 2 loud lines; 11 unique `adr:` keys | ✅ | ✅ green |
| 01-04-1 | 04 | 3 | SPEC-01, SPEC-02 | T-01-01 | publication guard re-asserted; spec-identifying strings absent | shell gate | registry count guard; ADR-resolution loop; spec-string sweep; stale-path sweep | ✅ | ✅ green |
| 01-04-2 | 04 | 3 | SPEC-01, SPEC-02 | — | repo quality gates untouched | make | `make lint` + `make test` + guard re-assert | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements — no Wave-0 scaffolding needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none exist)
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-11-19 (validate-phase §6 — no gaps found; all 11 task gates executed green during waves 1–3 and re-proven by the 01-04 phase audit)

## Validation Audit 2026-11-19

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
