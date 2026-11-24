---
phase: "01"
slug: "spec-reference-and-decisions"
status: verified
threats_open: 0
asvs_level: 1
created: "2026-11-25"
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| local working tree → public git history | Non-public Spec-derived material sits beside committable review artifacts | Local-only trace, amended Spec and diff must never be staged |
| clone → commit | The committed `.gitignore` guard is the publication control that follows the repository | Paths under `docs/reviews/2026-11-19-v1.0/internal/` |
| local-only trace → public registry | Registry summaries are authored next to quoted normative statements | IDs, code references and statuses may cross; statement text may not |
| registry/ADR references → reader | Broken or ambiguous references can hide a deviation or its owner decision | Statement IDs, ADR numbers and requirement IDs |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-01 | Information Disclosure | git staging of `docs/reviews/2026-11-19-v1.0/internal/` | high | mitigate | Committed `.gitignore` guard; explicit-path staging; scoped tracked/status checks both report zero; `git check-ignore -v` matches | closed |
| T-01-02 | Information Disclosure | `docs/SPEC-DEVIATIONS.md` summary column | medium | mitigate | Own-words authoring rule plus the trace-statement substring audit; all 167 rows pass the non-empty and six-value enum gates | closed |
| T-01-03 | Information Disclosure / drift | ADR prose derived from committed REVIEW §14 | low | accept | Source is already public; ADR mapping gates keep decisions 14.1–14.11 one-to-one and preserve the two accepted-by-default markers | closed |
| T-01-04 | Information Disclosure | committed REVIEW.md and evidence archive | medium | mitigate | Four spec-identifying-string sweep reports zero; scoped publication prefix remains untracked and ignored; 46 evidence files remain committed | closed |
| T-01-SC | Tampering / supply chain | package installation during phase execution | low | accept | Docs-only phase installs no packages; no package-legitimacy trigger | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-01-03 | T-01-03 | REVIEW §14 is committed public source; translating its settled decisions adds no disclosure. Mapping gates control drift. | Phase 01 plan 01-03 | 2026-11-19 |
| R-01-SC | T-01-SC | The phase installs no third-party packages, so the supply-chain surface is not exercised. | Phase 01 plans 01-01 through 01-04 | 2026-11-19 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-25 | 5 | 5 | 0 | GSD execute-phase secure-phase hook (ASVS L1) |

Evidence rechecked on the current tree:

- `git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` matches `.gitignore:42`.
- Zero tracked paths and zero porcelain paths exist under the exact guarded prefix `docs/reviews/2026-11-19-v1.0/internal/`.
- Exactly 46 evidence files are tracked under `docs/reviews/2026-11-19-v1.0/evidence/`.
- The `confstor`, `0031`, `cyp-config` and `cyp-perf` sweep over committed review material reports zero hits.
- The stale pre-move review-path sweep reports zero tracked files outside phase-archival planning docs.
- The 167-row registry passes the trace-statement substring audit and the fixed status-enum gate.
- The eleven owner decisions 14.1–14.11 each map to exactly one ADR; ADR 0012 is later additive work and does not disturb that phase-1 mapping.

The scoped-prefix rule is deliberate: 22 committed evidence paths legitimately contain an `internal` path segment below `evidence/repro/`; they are not the guarded Spec-material directory.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-25 (secure-phase hook, ASVS L1; all mitigations rechecked)
