---
phase: 01-ci-bootstrap-and-repo-skeleton
plan: 03
subsystem: infra
tags: [govulncheck, dependabot, github-actions, vulnerability-scanning, dependency-management, go]

# Dependency graph
requires:
  - phase: 01-01
    provides: Go module skeleton with the govulncheck v1.9.0 tool directive resolvable via go tool on go.mod
  - phase: 01-02
    provides: ci.yml on master — lint + 4-cell gate matrix that gates every PR including this plan's
provides:
  - .github/workflows/vulncheck.yml on master — weekly cron (Mon 06:00 UTC) + dependency-PR paths trigger (go.mod/go.sum) + workflow_dispatch, running the pinned go tool govulncheck
  - .github/dependabot.yml on master — gomod + github-actions version updates, weekly, minor+patch grouped (gomod-safe-updates, actions-safe-updates)
  - Live proof of CI-02's assertable half — dispatched vulncheck run green on master ("Your code is affected by 0 vulnerabilities")
  - Documented caveat that Dependabot ALERTS are GHAS-gated on this private repo until the deferred public flip
affects: [01-04 branch protection ruleset, all later phases (dependency currency), verify-work UAT (first Dependabot PR observation)]

# Actuals (#2632) — pairs with the plan's `estimate` (18000 tokens, 3 tasks, confidence low).
actuals:
  tokens: 509
  tasks: 3
  commits: 3

# Measured commit provenance (#3968)
plan_head_before: b0904f270f0aa0e2f61460b148121c7db9431946
plan_head_after: a893d9c6f15f116fb6bc83616d6248bc7b859ded

# Tech tracking
tech-stack:
  added:
    - Scheduled GitHub Actions workflow vulncheck.yml (checkout@v7, setup-go@v7 with go-version-file: go.mod, go tool govulncheck ./...)
    - Dependabot version updates for gomod + github-actions ecosystems with grouped minor+patch updates
  patterns:
    - Tool-directive invocation in CI — go tool govulncheck, never a go run of a latest tag (local/CI identical pinned build, D-04/D-06)
    - paths-filtered pull_request trigger (go.mod/go.sum) so dependency PRs are scanned before merge; schedule catches CVEs that land without code changes
    - Dependabot grouping (one minor+patch PR per ecosystem per week) sized to the small tool graph

key-files:
  created:
    - .github/workflows/vulncheck.yml
    - .github/dependabot.yml
  modified: []

key-decisions:
  - "vulncheck cron fixed at 0 6 * * 1 (Monday 06:00 UTC) — discretion area picked and recorded per plan"
  - "Dependabot grouped minor+patch weekly is deliberate: the tool graph is small and one PR per week beats five; majors would arrive ungrouped for manual review"
  - "Dependabot ALERTS documented as GHAS-gated on this private repo — updates active, alerts off until the deferred public flip (plan-gated, not silently dropped)"
  - "workflow_dispatch is the testability trigger: the scan is provable on demand instead of waiting for the weekly cron (research-flagged)"

patterns-established:
  - "Dependency currency loop: Dependabot proposes go.mod/action updates as PRs -> vulncheck paths-trigger scans those PRs before merge -> weekly cron rescans master for CVEs that land without code changes"

requirements-completed: [CI-02]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "vulncheck.yml on master: schedule cron 0 6 * * 1, pull_request paths go.mod/go.sum, workflow_dispatch, contents:read, concurrency vulncheck-ref with cancel-in-progress, checkout@v7 + setup-go@v7 with go-version-file: go.mod, scan step go tool govulncheck ./... (no latest-tag run)"
    requirement: CI-02
    verification:
      - kind: other
        ref: "plan <verify> grep (workflow_dispatch / 0 6 * * 1 / go tool govulncheck / go-version-file all present; no go run anywhere in the file)"
        status: pass
    human_judgment: false
  - id: D2
    description: "dependabot.yml on master: version 2, gomod + github-actions entries, both directory /, both weekly, both grouped minor+patch (gomod-safe-updates, actions-safe-updates), commit-message prefixes go-deps / ci with scope included, open-pull-requests-limit 10"
    requirement: CI-02
    verification:
      - kind: other
        ref: "grep -c 'package-ecosystem' .github/dependabot.yml = 2; gh api repos/Djarvur/sup-kv/contents/.github/dependabot.yml --jq '.size > 0' = true (default branch)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both config files landed on master through one green PR (#7) gated by the full plan-02 matrix: lint, gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15), gate (darwin-amd64-cross) — all pass; merged a893d9c, branch deleted"
    requirement: CI-02
    verification:
      - kind: integration
        ref: "gh pr checks 7 → 5/5 pass; gh pr view 7 --jq '.state == \"MERGED\"' → true"
        status: pass
    human_judgment: false
  - id: D4
    description: "Dispatched vulncheck run green on master: run 41216077968 (event workflow_dispatch, head a893d9c), conclusion success, step 'Run go tool govulncheck ./...' success, log 'Your code is affected by 0 vulnerabilities'"
    requirement: CI-02
    verification:
      - kind: integration
        ref: "gh run list --workflow=vulncheck.yml --limit 1 --json conclusion → success; run URL https://github.com/Djarvur/sup-kv/actions/runs/41216077968"
        status: pass
    human_judgment: false
  - id: D5
    description: "Backstop truth: Dependabot opens its first version-update PR within the weekly schedule window (observed only after the schedule elapses — not assertable at merge time)"
    verification: []
    human_judgment: true
    rationale: "Weekly window has not elapsed at execution time; the plan itself marks this truth verification: backstop — it abstains to human review at verify time rather than silently passing"

# Metrics
duration: 7min
completed: 2026-10-31
status: complete
---

# Phase 1 Plan 03: CI Bootstrap and Repo Skeleton — Vulnerability Scanning + Dependabot Summary

**vulncheck.yml (weekly cron Mon 06:00 UTC + go.mod/go.sum paths trigger + dispatch, pinned `go tool govulncheck` v1.9.0) and dependabot.yml (gomod + github-actions, weekly, minor+patch grouped) on master via green PR #7 — dispatched scan green: "Your code is affected by 0 vulnerabilities"**

## Performance

- **Duration:** 7 min
- **Started:** 2026-10-31T15:37:32Z
- **Completed:** 2026-10-31T15:44:41Z
- **Tasks:** 3
- **Files created:** 2 (both configuration; no code symbols)

## Accomplishments

- `.github/workflows/vulncheck.yml` on master (merge a893d9c): three triggers — weekly `schedule` cron `0 6 * * 1`, `pull_request` filtered to `go.mod`/`go.sum` paths (dependency PRs scanned before merge), and `workflow_dispatch` (scan provable on demand). Hygiene per D-06: `permissions: contents: read`, concurrency `vulncheck-${{ github.ref }}` with `cancel-in-progress: true`. Single ubuntu-latest job: `actions/checkout@v7`, `actions/setup-go@v7` with `go-version-file: go.mod`, then `go tool govulncheck ./...` — the tool-directive invocation pinned at v1.9.0, never a latest-tag run.
- `.github/dependabot.yml` on master: version 2, `gomod` + `github-actions` ecosystems at directory `/`, weekly interval, `open-pull-requests-limit: 10`, commit-message prefixes `go-deps`/`ci` with scope included, groups `gomod-safe-updates`/`actions-safe-updates` bundling minor+patch.
- Both files landed through one green PR (#7) gated by the full plan-02 matrix — lint + all four gate cells pass; merged, branch deleted.
- CI-02's assertable half proven live: dispatched run 41216077968 on master completed `success`; the scan log records "Your code is affected by 0 vulnerabilities" (plus 0 in imported packages and 14 in required-but-not-called modules — the reachability model working as designed, findings would fail the scan visibly with no exclusion list).
- The GHAS-gated alerts caveat is documented (below), not hidden — Dependabot ALERTS require a GHAS license unavailable on this private org tier; version updates are active, alerts stay off until the deferred public flip.

## Vulncheck Run Evidence (Task 3, verbatim)

- Run: https://github.com/Djarvur/sup-kv/actions/runs/41216077968 — `event: workflow_dispatch`, `headBranch: master`, `headSha: a893d9c`, `conclusion: success` (2026-10-31T15:41:50Z → 15:42:35Z)
- Steps: `Run actions/checkout@v7: success` · `Run actions/setup-go@v7: success` · `Run go tool govulncheck ./...: success`
- Scan log: "Your code is affected by 0 vulnerabilities." / "This scan also found 0 vulnerabilities in packages you import and 14 vulnerabilities in modules you require, but your code doesn't appear to call these vulnerabilities."
- Config liveness: `gh api repos/Djarvur/sup-kv/contents/.github/dependabot.yml --jq '.size > 0'` → `true`; `gh workflow list` shows `vulncheck`, `Dependabot Updates`, and `Dependency Graph` all `active`.

## GHAS-Gated Alerts Caveat (accepted, plan-gated)

Dependabot ALERTS require a GHAS/Code Security license unavailable on this private org tier (research-verified Pitfall 9). What is live now: version updates (both ecosystems) and the govulncheck scan. What stays off: Dependabot alerts — unlocked by the deferred public flip (a Deferred item, not silently dropped). The first version-update PR is expected within the weekly schedule window and is a verify-time observation, not a merge-time assertion (backstop truth D5).

## Task Commits

Each task was committed atomically (on `feat/01-03-vulncheck-dependabot`, merged to master via PR #7 with commits preserved):

1. **Task 1: vulncheck.yml — scheduled, dependency-PR-triggered, dispatchable scan** - `e9cefcd` (feat)
2. **Task 2: dependabot.yml — gomod + github-actions, weekly, grouped minor+patch** - `fa78082` (feat) + merge `a893d9c` (PR #7, all five checks green, branch deleted)
3. **Task 3: Dispatch the vulncheck run on master and assert green** - no new commit (executes the merged workflow; evidence above)

**Plan metadata:** docs commit (this SUMMARY + state files) lands via a follow-up docs PR (CI-03 PR-only rule, plan 01-02 pattern).

## Files Created/Modified

- `.github/workflows/vulncheck.yml` - scheduled + dependency-PR-triggered + dispatchable govulncheck job running the tool-directive-pinned build
- `.github/dependabot.yml` - version-update config for gomod and github-actions ecosystems, weekly, minor+patch grouped

## Decisions Made

- Cron fixed at Monday 06:00 UTC (`0 6 * * 1`) — the plan's recorded discretion choice
- Grouped minor+patch weekly updates: the tool graph is small, one PR per week beats five; majors remain ungrouped manual review
- Dispatch chosen as the merge-time proof trigger (testability), schedule as the continuous-coverage trigger (CVEs land without code changes), paths-filtered PR trigger as the pre-merge dependency gate

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The plan's Task 2 verify `gh pr view --json state,mergedAt --jq ...` needs an explicit PR argument when `--repo` is used (no repo-context resolution); re-run as `gh pr view 7 --repo Djarvur/sup-kv ...` → `true`. Same assertion, no impact.
- Unrelated working-tree files were present throughout and left untouched (staged nothing beyond plan files): `.planning/config.json` modified (`_auto_chain_active` flag, orchestrator-set), `docs/IDEA.md` modified (QUALITY/CI bullets), untracked `.planning/phases/01-ci-bootstrap-and-repo-skeleton/01-PATTERNS.md` and `.planning/state.json`. Surfaced for the orchestrator/next planner to disposition.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - configuration only, no stubs.

## Next Phase Readiness

- Vulnerability scanning runs continuously (weekly cron) and provably (dispatch green) with the pinned govulncheck v1.9.0; Dependabot keeps gomod + github-actions current as grouped weekly PRs — CI-02's assertable half is done, alerts documented as GHAS-gated
- CI-02 marked complete in REQUIREMENTS.md (single-plan requirement, no shared-ID gate)
- Plan 04 (branch protection ruleset) can proceed; its ruleset needs only the five ci.yml check-run contexts — vulncheck adds no required checks (by design: it triggers on dependency paths, not every PR)
- Verify-time observation pending: first Dependabot version-update PR expected within the weekly window (D5, backstop)

## Self-Check: PASSED

- `.github/workflows/vulncheck.yml` and `.github/dependabot.yml` exist on master (git ls-tree origin/master; merge a893d9c)
- Commits e9cefcd, fa78082, a893d9c are ancestors of origin/master (`git merge-base --is-ancestor`)
- All Task 1-3 acceptance criteria re-verified (trigger/hygiene/pin greps; ecosystem count 2; PR #7 MERGED with 5/5 checks; run 41216077968 conclusion success; contents API size > 0)

---
*Phase: 01-ci-bootstrap-and-repo-skeleton*
*Completed: 2026-10-31*
