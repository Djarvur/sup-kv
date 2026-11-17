---
phase: "1"
slug: "ci-bootstrap-and-repo-skeleton"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-10-31"
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) + go tool golangci-lint 2.15.0 (strict gate) |
| **Config file** | go.mod (tool directives), .golangci.yml (vendored strict config + depguard boundary) |
| **Quick run command** | `go build ./... && go vet ./...` |
| **Full suite command** | `make test` (go test -race ./...) + `make lint` (go tool golangci-lint run ./...) |
| **Estimated runtime** | quick ~30 s · full ~180 s |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go vet ./...`
- **After every plan wave:** Run `make lint && make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 s (quick) / 180 s (full)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-T1 | 01 | 1 | QUAL-01, D-01/02/04 | — | N/A | build+test gate | `go build ./... && go vet ./... && go test -race ./...` | ✅ | ✅ green |
| 01-01-T2 | 01 | 1 | QUAL-01, D-10/11/12 | — | depguard boundary: app has no net/os/syscall/google.golang.org | lint-gate | `go tool golangci-lint run ./...` + `go tool golangci-lint config verify` | ✅ | ✅ green |
| 01-02-T1/2 | 02 | 2 | CI-01, D-05/06/07 | — | least-privilege workflow (contents: read), no workflow injection | ci-matrix assertions | `grep -c 'runs-on:\|golangci-lint-action\|go-version-file\|cancel-in-progress\|fail-fast' .github/workflows/ci.yml`; `grep -n 'contents: read' .github/workflows/ci.yml` | ✅ | ✅ green |
| 01-02-T3 | 02 | 2 | CI-01 | — | red gate is red: depguard violation blocks a PR | ci-live probe | `gh pr checks` on probe PR (executed 2026-10/11; failure cited sup-kv-app-no-io) | ✅ | ✅ green |
| 01-03-T1 | 03 | 3 | CI-02, D-04/06 | — | pinned govulncheck v1.9.0, never latest-tag | ci-assertion | `grep -n 'workflow_dispatch\|0 6 \* \* 1\|go tool govulncheck\|go-version-file' .github/workflows/vulncheck.yml` | ✅ | ✅ green |
| 01-03-T2 | 03 | 3 | CI-02 | — | dependabot grouping bounds PR noise | config-assertion | `grep -c 'package-ecosystem' .github/dependabot.yml` (= 2) | ✅ | ✅ green |
| 01-03-T3 | 03 | 3 | CI-02 | — | scan runs green against the live vulnerability DB | ci-dispatch | dispatch vulncheck on master → green (run 41216077968, 0 vulnerabilities) | ✅ | ✅ green |
| 01-04-T1 | 04 | 4 | CI-03, D-08 | — | ruleset payload valid; delete_branch_on_merge on | repo-api | `python3 -m json.tool docs/setup/ruleset.json`; `gh api repos/Djarvur/sup-kv --jq '.delete_branch_on_merge'` | ✅ | ✅ green |
| 01-04-T2 | 04 | 4 | CI-03 | — | 403 blocker surfaced, not hidden | checkpoint decision | `gh api repos/Djarvur/sup-kv/rulesets` probe; auto-selected defer (gate auto_select honored) | ✅ | ✅ green |
| 01-05-T1/2 | 05 | 5 | CI-02 | — | ratified overrides recorded where the verifier reads them | doc-assertion | grep assertions on `01-VERIFICATION.md` overrides frontmatter + `REQUIREMENTS.md` CI-02/CI-03 rows | ✅ | ✅ green |
| 01-06-T1/2 | 06 | 6 | CI-03 | — | DEFERRED markers replace PENDING; re-apply trail intact | doc-assertion | grep assertions on `branch-protection.md`, `PROJECT.md`, `.claude/CLAUDE.md`, `WINDOWS.md` ledger, `STATE.md` decisions | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements (go test + strict lint gate landed by plan 01-01 itself).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Red-gate probe (CI-01 teeth) | CI-01 | Re-running opens a live violating PR on the repo — social cost exceeds value; evidence recorded in 01-02-SUMMARY.md | Cut throwaway branch with a `net` import in internal/app, open PR, confirm lint check fails citing sup-kv-app-no-io, close PR |
| Ruleset enforcement | CI-03 | Plan-gated: rulesets API returns 403 on this private org repo; ratified accepted override 2026-10-31 | Re-apply per docs/setup/branch-protection.md after a plan upgrade; then attempt a direct push to master and confirm rejection |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 180 s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-11-11

## Validation Audit 2026-11-11

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
