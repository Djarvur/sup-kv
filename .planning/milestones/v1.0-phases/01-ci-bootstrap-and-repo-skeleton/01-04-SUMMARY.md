---
phase: 01-ci-bootstrap-and-repo-skeleton
plan: 04
subsystem: infra
tags: [github-rulesets, branch-protection, gh-cli, ci-enforcement, repository-settings, delete-branch-on-merge]

# Dependency graph
requires:
  - phase: 01-02
    provides: ci.yml on master with the five fixed check-run context names (lint + four gate cells) the ruleset contexts copy verbatim; the green-matrix gate every PR in this plan passed through
  - phase: 01-03
    provides: completed dependency-currency half of the phase (wave ordering); this plan depends on it per depends_on
provides:
  - docs/setup/ruleset.json — the complete, field-verified "protect master" ruleset payload (pull_request 0-approval, strict required_status_checks with the five verbatim contexts, non_fast_forward, deletion), ready to POST the instant the org plan allows rulesets on private repos
  - docs/setup/branch-protection.md — the D-08 audit trail: exact gh api apply / read-side verify / update / re-apply commands, the zero-approval rationale, the context-coupling rule (any matrix change is a ruleset change), and the PENDING marker with the recorded defer decision
  - Repo setting delete_branch_on_merge=true — applied via PATCH and verified live (unconditional part of CI-03's ergonomics)
  - The explicit deferral record: CI-03's protection sub-item is plan-gated (org plan 403), decision `defer` taken at the Task 2 checkpoint, re-apply is one command
affects: [all later phases (PR-only discipline, ruleset contexts coupling), verify-work UAT (CI-03 protection sub-item review), ship gate (open windows entry until ruleset applied), complete-milestone]

# Actuals (#2632) — pairs with the plan's `estimate` (22000 tokens, 3 tasks, confidence low).
actuals:
  tokens: 1312
  tasks: 3
  commits: 4

# Measured commit provenance (#3968)
plan_head_before: 5e587c1b02cf302b58017ee60e5cb0347db44514
plan_head_after: edc5d3f79be1d5b7ef409bb16a5ec8548030edd2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Protection is scripted, never UI-clicked — gh api POST/PATCH/GET with the payload in git as source of truth (D-08)
    - Context-coupling rule — required_status_checks contexts must equal the ci.yml check-run display names; every matrix change is a ruleset change in the same commit
    - Zero-approving-reviews PR gate — required_approving_review_count 0 for a solo maintainer (GitHub forbids self-approval, so any count >= 1 deadlocks every PR); the PR-only contract itself still enforced by the pull_request rule

key-files:
  created:
    - docs/setup/ruleset.json
    - docs/setup/branch-protection.md
  modified: []

key-decisions:
  - "Task 2 decision checkpoint auto-selected `defer` (gate=blocking, auto_select honored under active auto mode): no org plan upgrade now; ruleset payload recorded, re-apply one command; CI-03's protection sub-item flagged plan-gated, not hidden"
  - "required_approving_review_count fixed at 0 (A5 resolved): solo maintainer cannot approve own PRs, count >= 1 would deadlock; require_last_push_approval and required_review_thread_resolution also false for the same reason — no approval requirement nobody else can satisfy"
  - "delete_branch_on_merge enabled now via PATCH (ordinary repo setting, not plan-gated) — verified true read-side"
  - "CI-03 deliberately NOT marked complete in REQUIREMENTS.md: PR-only development is practiced (every phase-01 change landed via PR #1-#10), but server-side branch protection is deferred — traceability row annotated as Partial with the deferral pointer instead"

patterns-established:
  - "Scripted-protection audit pattern: payload + commands in git, GitHub state follows git; bump = edit payload via PR, re-run the doc's apply/verify"
  - "Honest-flagging pattern for plan-gated items: marker in the setup doc + SUMMARY coverage routed to human judgment + windows ledger entry — a deferred requirement stays visible at ship time"

requirements-completed: []

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "docs/setup/ruleset.json on master: valid JSON, name 'protect master', target branch, enforcement active, ref_name include exactly refs/heads/master, all four rule types (pull_request 0-approval, required_status_checks strict + 5 contexts, non_fast_forward, deletion); the five context strings match the 01-02-SUMMARY check-run names character-for-character"
    requirement: CI-03
    verification:
      - kind: other
        ref: "python3 json parse + context-equality check on origin/master payload (match: True; rule types [pull_request, required_status_checks, non_fast_forward, deletion])"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/setup/branch-protection.md on master: POST apply command, read-side verify command, delete_branch_on_merge PATCH, re-application/audit note with the context-coupling rule, zero-approval rationale, prominent PENDING marker carrying the 2026-10-31 defer decision"
    requirement: CI-03
    verification:
      - kind: other
        ref: "grep on origin/master: POST/verify/PATCH/PENDING/rationale all present; 'Deferral decision 2026-10-31: defer' at line 24"
        status: pass
    human_judgment: false
  - id: D3
    description: "Repository setting delete_branch_on_merge=true applied via gh api PATCH and verified by independent read"
    verification:
      - kind: integration
        ref: "gh api repos/Djarvur/sup-kv --jq '.delete_branch_on_merge' -> true (read back after PATCH)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every file change in this plan landed through a green PR: PR #9 (ruleset.json + branch-protection.md, 5/5 checks, merged 2a77712) and PR #10 (PENDING-marker status update, 5/5 checks, merged edc5d3f); branches deleted after merge"
    verification:
      - kind: integration
        ref: "gh pr view 9/10 --jq '.state == \"MERGED\"' -> true; gh pr checks 9/10 -> all five pass each"
        status: pass
    human_judgment: false
  - id: D5
    description: "The deferral record itself: Task 2 decision taken (auto-select defer under active auto mode, gate=blocking with auto_select honored), read-side probe re-verified the live 403 ('Upgrade to GitHub Pro or make this repository public'), NO ruleset POST attempted, doc marker updated with decision date"
    verification:
      - kind: integration
        ref: "probe output (403 verbatim) recorded above; git show origin/master:docs/setup/branch-protection.md carries '## Status: PENDING' + decision line"
        status: pass
    human_judgment: false
  - id: D6
    description: "CI-03 protection sub-item: direct pushes to master rejected server-side (ruleset active, force pushes and deletion blocked) — DEFERRED, not delivered; master has no server-side push protection until the org plan changes"
    requirement: CI-03
    verification: []
    human_judgment: true
    rationale: "Deferred by explicit decision at the Task 2 checkpoint (auto-selected 'defer'): the rulesets API returns 403 on this private org repo (plan-gated). Unprovable while the ruleset is pending — the plan's backstop truth (direct pushes rejected once applied) abstains to human review; flagged plan-gated here and in the windows ledger rather than silently passed."

# Metrics
duration: 6min
completed: 2026-10-31
status: complete
---

# Phase 1 Plan 04: CI Bootstrap and Repo Skeleton — Branch Protection Summary

**"protect master" ruleset payload + scripted apply/verify doc landed on master via green PRs, delete_branch_on_merge enabled live; the ruleset itself explicitly DEFERRED (org-plan 403, checkpoint decision) with the re-apply path one command away — CI-03 honest-split, protection flagged plan-gated**

## Performance

- **Duration:** 6 min
- **Started:** 2026-10-31T15:50:58Z
- **Completed:** 2026-10-31T15:57:06Z (task merges; SUMMARY thereafter)
- **Tasks:** 3 (2 auto + 1 decision checkpoint auto-selected)
- **Files created:** 2 (both documentation/config; no code symbols)

## Accomplishments

- `docs/setup/ruleset.json` on master (merge `2a77712`, PR #9): the field-verified payload — name `protect master`, target `branch`, `enforcement: active`, `conditions.ref_name.include` exactly `refs/heads/master`; rules `pull_request` (`required_approving_review_count: 0`, `dismiss_stale_reviews_on_push: true`, `require_code_owner_review/require_last_push_approval/required_review_thread_resolution: false`), `required_status_checks` (`strict_required_status_checks_policy: true`; five contexts copied verbatim: `lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)`, `gate (darwin-amd64-cross)`), `non_fast_forward`, `deletion`. Context equality with the 01-02-SUMMARY names proven programmatically (character-for-character, `match: True`).
- `docs/setup/branch-protection.md` on master: the D-08 audit trail — exact `gh api` apply (`POST repos/Djarvur/sup-kv/rulesets --input docs/setup/ruleset.json`, 201 created / 422 payload-error semantics), read-side verification asserting name + active enforcement + the four rule types, the update (`PUT .../rulesets/<id>`) and from-scratch re-apply paths, the delete-branch-on-merge PATCH, the context-coupling rule (any gate-matrix change is a ruleset change in the same commit — T-01-09 mitigation), the zero-approval rationale, the no-path-exemptions note (CI-03 prohibition), and the prominent PENDING marker.
- `delete_branch_on_merge=true` applied NOW (ordinary repo setting, not plan-gated): PATCH executed, independent read-back prints `true`.
- Task 3 (defer path): read-side probe re-verified the live blocker verbatim — `403 ... Upgrade to GitHub Pro or make this repository public to enable this feature` — no POST attempted; PENDING marker updated with "**Deferral decision 2026-10-31: `defer`**" and landed through green PR #10 (`edc5d3f`).

## Checkpoint Decision Record (Task 2)

`checkpoint:decision gate="blocking" auto_select="defer"` — auto mode active (`workflow.auto_advance: true`, `_auto_chain_active: true`), gate is `blocking` (not `blocking-human`) and `auto_select` names a valid option → **auto-selected: defer** per the auto-mode checkpoint protocol. The checkpoint's A5 confirmation stands as written: `required_approving_review_count: 0` (a different count was not requested). Consequence recorded everywhere required: CI-03's protection sub-item is plan-gated — flagged, not hidden.

## Task Commits

Each task was committed atomically (PR-only contract honored by this plan's own merges; per-task commits preserved via `--merge`):

1. **Task 1: Ruleset payload + setup doc + delete_branch_on_merge** - `6001b77` (feat) + merge `2a77712` (PR #9, all five checks green, branch deleted)
2. **Task 2: Decision — apply now or defer** - no commit (auto-selected `defer`; recorded in Task 3's doc update)
3. **Task 3: Confirm the recorded deferral** - `17d89c4` (docs) + merge `edc5d3f` (PR #10, all five checks green, branch deleted)

**Plan metadata:** this SUMMARY + state files land via the follow-up docs PR (CI-03 PR-only rule, plans 01-02/01-03 pattern).

## Files Created/Modified

- `docs/setup/ruleset.json` - repository ruleset payload for refs/heads/master (pull_request 0-approval, strict required status checks with five verbatim contexts, non_fast_forward, deletion)
- `docs/setup/branch-protection.md` - apply/verify/re-apply command reference, zero-approval rationale, context-coupling rule, PENDING marker with defer decision
- Repository setting (GitHub-side): `delete_branch_on_merge=true`

## Decisions Made

- Defer, not apply-now: the org plan does not support rulesets on private repos (live 403, re-verified by probe); upgrading is a human billing action and D-09 locks the repo private — the scripted payload + commands make re-application a one-command operation the moment the plan changes (option `defer` pros/cons as presented at the checkpoint)
- Zero approving reviews with the PR requirement retained: GitHub forbids authors approving their own PRs, so any positive count deadlocks the solo maintainer; `require_last_push_approval` and `required_review_thread_resolution` set false for the same reason (A5 resolution, recorded in the setup doc for future maintainers)
- CI-03 stays Pending in REQUIREMENTS.md with a "Partial" traceability annotation pointing at the deferral — marking it Complete would hide a plan-gated gap the plan explicitly requires to stay visible (hence `requirements-completed: []` in this SUMMARY's frontmatter)
- Unrelated pre-existing working-tree files (`.planning/config.json`, `docs/IDEA.md`, untracked `01-PATTERNS.md`, `.planning/state.json`) left untouched, staged nothing beyond plan files — carried over from plan 01-03's surface, still awaiting orchestrator/next-planner disposition

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. (Applying the ruleset later requires a one-time human billing decision — org plan upgrade — after which the executor-side apply is one command from `docs/setup/branch-protection.md`.)

## Known Stubs

None - documentation and configuration only; the deferred ruleset is an explicitly decision-gated omission, not a stub (see coverage D6).

## Threat Surface

No new threat surface beyond the plan's register: T-01-09 mitigated as scripted (context-coupling rule + read-side verification commands in the doc); T-01-10 mitigated as designed (blocking user decision taken and recorded — the exposure window is a conscious, documented choice).

## Next Phase Readiness

- Phase 1 is now complete on its own terms: all four plans summarized; every landed change went through a green PR (PRs #1-#10); lint boundary, gate matrix, tidy gate, vulncheck, Dependabot all live on master
- Standing gap until the org plan changes: master has no server-side push protection (CI-03 protection sub-item plan-gated; windows ledger entry open; re-apply = one command in `docs/setup/branch-protection.md`)
- The five ruleset contexts are coupled to ci.yml — any matrix rename between now and application must update `docs/setup/ruleset.json` in the same commit
- Ready for `/gsd:verify-work` (expect CI-03 routed to human review via coverage D6) and `/gsd:plan-phase 02` (engine)

## Self-Check: PASSED

All three created files exist on disk; task commits 6001b77, 17d89c4 and merges 2a77712 (PR #9), edc5d3f (PR #10) are ancestors of origin/master; both PRs verified MERGED with 5/5 checks; contexts-equality, doc-content greps, and delete_branch_on_merge=true re-verified against master; the rulesets endpoint's 403 re-confirmed as the recorded defer state. All Task 1-3 acceptance criteria re-verified (defer path).

---
*Phase: 01-ci-bootstrap-and-repo-skeleton*
*Completed: 2026-10-31*
