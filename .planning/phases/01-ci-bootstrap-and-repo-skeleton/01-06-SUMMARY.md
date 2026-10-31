---
phase: 01-ci-bootstrap-and-repo-skeleton
plan: 06
subsystem: infra
tags: [gap-closure, ratified-override, branch-protection, broken-windows, state-markers, traceability]

# Dependency graph
requires:
  - phase: 01-ci-bootstrap-and-repo-skeleton (plans 01-04 + 01-05)
    provides: the deferral decision (D-08 defer, org-plan 403) and the recorded accepted overrides in 01-VERIFICATION.md this plan's markers cite
provides:
  - "Ratified-deferral state in every marker beyond VERIFICATION.md/REQUIREMENTS.md: branch-protection.md DEFERRED (accepted override) status with verbatim re-apply trail; PROJECT.md/.claude/CLAUDE.md Process constraint honest about the deferral (IN-02 closed at source and mirror)"
  - "Broken-windows ledger item 2 waived with the override reason (open_count 1 / waived_count 1); Dependabot first-PR backstop (item 1) still open for its 2026-11-02 06:00 UTC window"
  - "STATE.md decision log carrying both maintainer ratifications (gap 1 CI-03 deferral, gap 2 CI-02 alerts) with 2026-10-31 decision date and gap-plan provenance"
affects: [phase-01 verifier re-run, gsd:verify-work, gsd:ship (windows gate reads open_count), all downstream agents reading .claude/CLAUDE.md]

# Actuals (#2632) — same estimateTokens scale (chars/4 over the realized diff)
actuals:
  tokens: 2263
  tasks: 2
  commits: 4  # MEASURED: git rev-list --count gsd-plan-head-before-01-06..HEAD (2 task commits + PR merges #20/#21; metadata PR lands after)

tech-stack:
  added: []
  patterns:
    - "Ratified-marker pattern: a deferred capability is recorded as DEFERRED (accepted override) with attribution date + ratification paragraph, while the entire re-apply command trail stays byte-verbatim — the record IS the revert path"

key-files:
  created:
    - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-06-SUMMARY.md
  modified:
    - docs/setup/branch-protection.md
    - .planning/PROJECT.md
    - .claude/CLAUDE.md
    - .planning/WINDOWS.md
    - .planning/STATE.md

key-decisions:
  - "Status marker flipped from unresolved PENDING to DEFERRED (accepted override) with a ratification paragraph, while every apply/verify/re-apply command and the Deferral decision 2026-10-31 line stay verbatim — the doc records the ratification without touching the one-command re-apply path"
  - "Identical deferral parenthetical inserted at the identical position in PROJECT.md (source) and .claude/CLAUDE.md (generated mirror) so regeneration cannot revert IN-02's fix; verified textually identical"
  - "WINDOWS.md item 2 waived via gsd_run windows waive with the override reason; item 1 (Dependabot first-PR backstop) deliberately left open — its 2026-11-02 window has not elapsed and the BACKSTOP-OPEN gate asserts it"
  - "CI-02/CI-03 deliberately NOT marked complete (requirements-completed empty): this plan records ratified deferral markers, it does not deliver the deferred halves — REQUIREMENTS.md's honest Partial state stands"

patterns-established:
  - "Deferral-state marker trio: repo doc (status heading + ratification paragraph) + constraint source/mirror parenthetical + ledger waiver — all citing the same decision date and provenance so every reader sees one story"

requirements-completed: []  # DELIBERATE: this plan records ratified deferral markers for CI-02/CI-03; the deferred halves (ruleset application, GHAS alerts) remain undelivered — REQUIREMENTS.md Partial/deferral annotations stand

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Repo-doc ratified markers: branch-protection.md Status DEFERRED (accepted override) + ratification paragraph with the entire command trail verbatim; PROJECT.md and .claude/CLAUDE.md Process constraints carry the identical deferral parenthetical (IN-02 closed at source and mirror)"
    requirement: CI-03
    verification:
      - kind: other
        ref: "plan 01-06 Task 1 <verify> gates (HEADING-OK, NO-PENDING, TRAIL-INTACT, CONSTRAINT-SYNC) — all PASS, re-run post-merge on master; PR #20 5/5 ci.yml checks green, merged f4d088b"
        status: pass
    human_judgment: false
  - id: D2
    description: "Broken-windows ledger: item 2 (CI-03 protection deferral) waived with reason naming the ratified accepted override; item 1 (Dependabot first-PR backstop) still open; counters open_count 1 / waived_count 1"
    verification:
      - kind: other
        ref: "plan 01-06 Task 2 <verify> gates (LEDGER-OK, BACKSTOP-OPEN via gsd_run windows status) — all PASS, re-run post-merge on master; PR #21 5/5 ci.yml checks green, merged bf82f30"
        status: pass
    human_judgment: false
  - id: D3
    description: "STATE.md decision log: exactly two new entries (Verification gap 1 and gap 2, both ratified 2026-10-31 maintainer accepted override, gap-plan provenance 01-06/01-05) appended to the Decisions section; frontmatter untouched"
    verification:
      - kind: other
        ref: "plan 01-06 Task 2 <verify> gate DECISIONS-OK (exactly-once grep for both entries) — PASS, re-run post-merge on master"
        status: pass
    human_judgment: false

# Metrics
duration: 8min
completed: 2026-10-31
status: complete
plan_head_before: 6adb3156aa3c570c9b093981957293fd37b8bd5b
plan_head_after: bf82f301639684eec6ba2db89bb372a1ff4eba30
---

# Phase 1 Plan 6: Gap-Closure Deferral-Marker Recording Summary

**Ratified 2026-10-31 deferral recorded in every marker beyond the verification artifacts: branch-protection.md flipped to DEFERRED (accepted override) with a verbatim re-apply trail, PROJECT.md/CLAUDE.md Process constraints honest (IN-02 closed), WINDOWS.md item 2 waived with open_count 1, and both ratifications logged in STATE.md — all via green PRs #20/#21**

## Performance

- **Duration:** 8 min
- **Started:** 2026-10-31T18:55:17Z
- **Completed:** 2026-10-31T19:02:49Z
- **Tasks:** 2
- **Files modified:** 5 (all tracked artifacts; no new source files)

## Accomplishments
- `docs/setup/branch-protection.md` Status heading `## Status: PENDING` → `## Status: DEFERRED (accepted override)`; new ratification paragraph (**Ratified 2026-10-31: accepted override.** …) appended after the Deferral decision 2026-10-31 line. The 403 explanation, the plan-gated flagging, and every command (Apply POST, read-side Verify, PATCH delete_branch_on_merge, PUT/DELETE re-apply) are byte-identical — the only removed line in the whole file is the old heading.
- `.planning/PROJECT.md` and `.claude/CLAUDE.md` Process bullets each gained the identical parenthetical `(server-side ruleset deferred 2026-10-31 — ratified accepted override, org-plan gated; one-command re-apply in docs/setup/branch-protection.md)` immediately after "branch protection on `master`" — the constraint no longer asserts server-side enforcement, and the generated mirror equals its source (verified textually identical), so the verifier's IN-02 "actively misleading" finding is closed at its source and cannot resurface on regeneration.
- `.planning/WINDOWS.md` item 2 (CI-03 protection deferral) waived via `gsd_run windows waive 2 "Ratified accepted override 2026-10-31 (maintainer): …"`; counters open_count 1 / waived_count 1. Item 1 (Dependabot first-PR backstop, window Mon 2026-11-02 06:00 UTC) intentionally stays open — the BACKSTOP-OPEN gate asserts it.
- `.planning/STATE.md` Decisions section gained exactly two entries — Verification gap 1 (CI-03 branch protection, gap plan 01-06) and Verification gap 2 (CI-02 alerts GHAS-gated, gap plan 01-05), both ratified 2026-10-31 by the maintainer as accepted overrides — appended in the existing `- [Phase 01]: …` style; frontmatter untouched.
- Both tasks landed through the PR-only flow (Task 1: PR #20, merge `f4d088b`; Task 2: PR #21, merge `bf82f30`), each with all 5 ci.yml checks green (lint, gate ubuntu-latest, ubuntu-24.04-arm, macos-15, darwin-amd64-cross). No ruleset application, no org-plan change, no public flip, no GHAS enablement.

## Task Commits

Each task was committed atomically on its own feature branch and merged via a green PR:

1. **Task 1: Ratified-deferral markers in repo docs — branch-protection.md status, PROJECT.md Process constraint, .claude/CLAUDE.md mirror (IN-02)** - `0a4ca44` (docs, merged via PR #20, merge `f4d088b`)
2. **Task 2: Planning-state markers — waive WINDOWS.md item 2 with the override reason; log both ratified overrides in STATE.md** - `fe460f0` (docs, merged via PR #21, merge `bf82f30`)

**Plan metadata:** this SUMMARY commit (docs, via docs PR)

## Files Created/Modified
- `docs/setup/branch-protection.md` - Status heading → DEFERRED (accepted override); ratification paragraph; command trail verbatim
- `.planning/PROJECT.md` - Process constraint gains the deferral parenthetical (source of the generated mirror)
- `.claude/CLAUDE.md` - identical parenthetical at the identical position in the generated mirror (IN-02 closed)
- `.planning/WINDOWS.md` - item 2 waived with the override reason; item 1 open; counters 1/1
- `.planning/STATE.md` - two ratified-override decision entries; frontmatter untouched
- `.planning/phases/01-ci-bootstrap-and-repo-skeleton/01-06-SUMMARY.md` - this summary

## Decisions Made
- Flipped the status marker rather than deleting or rewriting it: the heading now states the ratified outcome, while the Deferral decision line and all re-apply commands stay verbatim — the audit trail doubles as the revert path the maintainer chose to keep.
- Waived only ledger item 2 and left item 1 open: the Dependabot first-PR backstop is a future observation (window Mon 2026-11-02 06:00 UTC), not a defect to waive — the verify gate asserts `open_count: 1` with item 1 still open so an over-broad waiver cannot pass silently.
- `requirements-completed: []` — mirroring plan 01-05's precedent: this plan records accepted deviations; CI-02/CI-03 stay Partial/deferred in REQUIREMENTS.md until the deferred halves are actually delivered (ruleset applied / GHAS alerts on).
- Task 2 run on its own fresh branch (`docs/01-gap-state-markers`) after PR #20 merged, per the plan's explicit "or a fresh branch if run separately" option — keeps each PR diff scoped to its task's files.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Plan-text staleness (no action needed): Task 2's read_first described "the two plan 01-04 rows" as the last Decisions entries — plan 01-05 had since appended three of its own rows. The two new entries were appended at the true end of the list (chronological, after the 01-05 rows), which is the section's evident intent; the DECISIONS-OK exactly-once gates confirm the outcome.

## Authentication Gates
None - gh was already authenticated as onokonem; the environment's dead http(s)_proxy was cleared for gh calls per dispatch notes.

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - documentation-state markers only; no code, no stubs.

## Next Phase Readiness
- Phase 1 (CI Bootstrap and Repo Skeleton) is complete: all 6 of 6 plans have summaries. The verifier re-run should now find gap 1 recorded as ratified in every marker it named (VERIFICATION.md overrides from 01-05, plus the repo/state markers from this plan), with the Dependabot backstop the only open ledger item — tracked, not hidden.
- `/gsd:ship` (windows gate) still sees open_count 1 by design until the Dependabot first-PR backstop is observed after Mon 2026-11-02 06:00 UTC (`gsd_run windows fixed 1` once observed).
- Ready for `/gsd:plan-phase 2` (Engine: RecordVersion/HLC + UUIDv7 + LWW map + property tests) — the enforced repo skeleton and quality gates it builds on are in place.

## Self-Check: PASSED

- Files exist: 01-06-SUMMARY.md, branch-protection.md, PROJECT.md, .claude/CLAUDE.md, WINDOWS.md, STATE.md — all FOUND
- Commits are ancestors of HEAD: 0a4ca44, f4d088b (PR #20), fe460f0, bf82f30 (PR #21) — all FOUND
- `gsd_run check evaluation-scope --plan 01-06 --commits-only` → resolved, 2 task commits on this branch (0a4ca44, fe460f0)
- Task 1 verify gates re-run post-merge on master: HEADING-OK, NO-PENDING, TRAIL-INTACT, CONSTRAINT-SYNC — all PASS
- Task 2 verify gates re-run post-merge on master: LEDGER-OK, DECISIONS-OK, BACKSTOP-OPEN — all PASS
- Plan-level cross-checks: windows status JSON consistent with the ledger table (1 open / 1 waived, item 1 open); branch-protection.md diff shows only the PENDING heading removed (command trail diff-clean); PROJECT.md and .claude/CLAUDE.md Process bullets textually identical; plan diff touches exactly the five named files

---

*Phase: 01-ci-bootstrap-and-repo-skeleton*
*Completed: 2026-10-31*
