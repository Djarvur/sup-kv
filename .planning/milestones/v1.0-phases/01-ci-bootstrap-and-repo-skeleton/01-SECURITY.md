---
phase: "01"
slug: "ci-bootstrap-and-repo-skeleton"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-01"
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| network → build graph | tool-directive fetches cross from proxy.golang.org into the module graph | Go module code (pinned tool versions) |
| raw.githubusercontent.com → repo | the vendored lint config enters the repo as data from an external host | lint config text (vendored strict config) |
| PR author → CI | untrusted pull-request content crosses into workflow execution | arbitrary PR branch code/config |
| GitHub → repo | the default workflow token crosses into the checkout during runs | GITHUB_TOKEN (contents: read only) |
| Go vuln DB → CI | external advisory data drives a gating decision | vulnerability advisories (OSV) |
| Dependabot service → repo | an automated actor opens PRs against master | dependency-update PRs |
| git client → master | direct pushes, force pushes, and deletions cross into the protected default branch | git refs on master |
| ruleset config → GitHub enforcement | the scripted payload becomes server-side access control | ruleset JSON |
| maintainer decision → verification record | human ratification crosses into git-tracked verifier artifacts | override entries, waivers, annotations |
| agent context → future agents | .claude/CLAUDE.md is trusted by every downstream agent | project instructions |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-SC | Tampering | go get -tool installs (golangci-lint v2.15.0, govulncheck v1.9.0) | high | mitigate | Package Legitimacy Audit in 01-RESEARCH.md; exact-version pins in go.mod (`github.com/golangci/golangci-lint/v2 v2.15.0`, `golang.org/x/vuln v1.9.0`) | closed |
| T-01-01 | Tampering | .golangci.yml loading (unknown key silently no-ops the strict config) | high | mitigate | `go tool golangci-lint config verify` is a standing gate: Makefile `lint` target + ci.yml lint job step (PR #26); empirical probe recorded: `run` with bogus key exits 0, `config verify` exits 3 | closed |
| T-01-02 | Tampering | go.mod dependency graph | medium | mitigate | D-03 zero direct runtime requires; go.mod require block is `// indirect`-only; tool graph pinned to canonical upstream paths | closed |
| T-01-03 | Tampering | vendored config fetch | medium | mitigate | Origin header asserted verbatim as first line of .golangci.yml; diff-vs-pristine upstream fetch = single pure insertion (01-01 SUMMARY verification) | closed |
| T-01-04 | Elevation | GITHUB_TOKEN permissions | medium | mitigate | `permissions: contents: read` at workflow top in ci.yml and vulncheck.yml; no pull_request_target trigger | closed |
| T-01-05 | Tampering | workflow run steps | medium | mitigate | no `github.event.*` untrusted-field interpolation anywhere in workflows (grep-verified); static matrix literals | closed |
| T-01-06 | Tampering | third-party action tags | medium | mitigate | actions major-pinned (checkout@v7, setup-go@v7, golangci-lint-action@v9 — deliberate, recorded in 01-02 SUMMARY), linter itself patch-pinned `version: v2.15.0` in CI == go.mod tool directive; Dependabot github-actions surfaces tag moves as reviewable PRs | closed |
| T-01-07 | Tampering | .github/dependabot.yml | medium | mitigate | presence and shape asserted via contents API at plan time (01-03); gomod + github-actions weekly grouped minor+patch verified on master; first-PR backstop truth tracked in WINDOWS.md ledger (item 1) | closed |
| T-01-08 | Tampering | vulncheck workflow | medium | mitigate | three triggers verified: weekly cron (Mon 06:00 UTC), pull_request paths go.mod/go.sum, workflow_dispatch; no exclusion list for findings; removal is a reviewable master diff | closed |
| T-01-09 | Elevation | ruleset drift after matrix change | medium | mitigate | context-coupling rule ("every gate-matrix change is a ruleset change") + gh api read-side verify + auditable re-apply commands in docs/setup/branch-protection.md | closed |
| T-01-10 | Elevation | master unprotected while deferred | medium | mitigate | surfaced as blocking user decision with explicit recorded deferral (PENDING marker in branch-protection.md); exposure window is a conscious documented choice; private repo, single maintainer | closed |
| T-01-11 | Elevation | Dependabot-opened PRs | low | accept | all PRs (including Dependabot's) pass the same ci.yml matrix; runtime-graph prohibition + indirect-only requires make a smuggled runtime dep reviewable in the PR diff | closed |
| T-01-12 | Tampering | 01-VERIFICATION.md override mechanism | medium | mitigate | additive-only frontmatter edit verified by the verifier: exactly two accepted entries, must_have/reason verbatim from the report's suggestion block, attributed accepted_by/accepted_at, 9 insertions 0 deletions | closed |
| T-01-13 | Tampering | .planning/REQUIREMENTS.md traceability | medium | mitigate | CI-02/CI-03 rows annotated Partial with the ratified deferral recorded — status moved toward honesty, never erasing the deferral | closed |
| T-01-14 | Elevation | refs/heads/master (residual exposure of the ratified deferral) | medium | accept | accepted by the maintainer 2026-10-31 (only party with authority over the org billing decision); bounded: private repo, single maintainer, PR-only practiced, one-command re-apply kept, ledger tracks the deviation | closed |
| T-01-15 | Tampering | WINDOWS.md ledger waiver scope | medium | mitigate | waiver targets item 2 only with mandatory ratified reason; ledger frontmatter asserts open_count: 1 with item 1 still open (first-PR observation stays demanded) | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-01-11 | Dependabot PRs run through the same ci.yml matrix as any PR; the runtime-graph prohibition makes a smuggled runtime dep inspectable in the PR diff | onokonem (plan-time disposition) | 2026-10-31 |
| AR-02 | T-01-14 | Direct-push rejection deferred (org plan gates rulesets on private repos, 403); private repo, single authenticated maintainer, PR-only practiced; one-command re-apply kept in docs/setup/branch-protection.md | onokonem | 2026-10-31 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-01 | 15 | 15 | 0 | gsd-secure-phase (orchestrator L1 + maintainer) |

Audit notes: initial L1 classification found T-01-01 OPEN (config verify was a one-time proof, not a standing gate; empirical probe — `run` with an unknown top-level key exits 0, `config verify` exits 3). Maintainer chose to fix rather than accept; PR #26 added `config verify` to the Makefile `lint` target and the ci.yml lint job (all 5 checks green, merged 2026-11-01). Re-classified CLOSED against merged master.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-01
