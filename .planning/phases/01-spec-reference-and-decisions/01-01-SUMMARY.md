---
phase: 01-spec-reference-and-decisions
plan: 01
subsystem: docs
tags: [review-artifacts, gitignore-guard, relocation, reference-hygiene, historical-banner]
requires: []
provides:
  - "Committed review record: docs/reviews/2026-11-19-v1.0/REVIEW.md + evidence/ (46 files) — source of §7/§13/§14 for plans 02-04"
  - "Committed .gitignore guard making docs/reviews/2026-11-19-v1.0/internal/ uncommitable from any clone (D-04)"
  - "Local-only docs/reviews/2026-11-19-v1.0/internal/ (SPEC.md, SPEC-amended.md, SPEC-amendments.diff, spec-trace.md) — derivation input for plan 01-02, never committed"
  - "docs/IDEA.md loudly marked historical (banner, body byte-untouched) with as-built trio links"
  - "Post-move review path references in .planning/{PROJECT,ROADMAP,REQUIREMENTS}.md"
affects: [01-02-spec-deviation-registry, 01-03-owner-decision-adrs, 01-04-phase-verification, phase-2-red-tests, phase-9-docs]
actuals:
  tokens: 168000
  tasks: 3
  commits: 3
  note: "chars/4 over the full changeset diff (671,997 chars); ~97% of the volume is the relocated review payload (REVIEW.md ~104K + evidence/ ~760K) moved verbatim, not authored content — the estimate target (28,000) covered authored/planning text only"
plan_head_before: 378b27e390fcb3ca16f22d9329e0736c482b7dad
plan_head_after: 3b9569dd9e7f7d37f5df850961a7a9f2877dcf24
tech-stack:
  added: []
  patterns: ["committed .gitignore guard + explicit-path staging as publication control", "prepend-only historical banner (body byte-untouched)"]
key-files:
  created:
    - "docs/reviews/2026-11-19-v1.0/REVIEW.md"
    - "docs/reviews/2026-11-19-v1.0/evidence/ (46 files: docs-audit, review-engine, review-repl-api, test-audit, repro/, stand/)"
  modified:
    - ".gitignore"
    - "docs/IDEA.md"
    - ".planning/PROJECT.md"
    - ".planning/ROADMAP.md"
    - ".planning/REQUIREMENTS.md"
key-decisions:
  - "Negative internal/ verify gates scoped to the non-public directory prefix docs/reviews/2026-11-19-v1.0/internal/, not to any path containing the substring 'internal' — evidence/repro/**/internal/** repro-test files are legitimate D-01 payload"
patterns-established: ["guard block appended to .gitignore in the repo's comment-block convention", "loud blockquote+bold historical banner above the H1"]
requirements-completed: [SPEC-01]
coverage:
  - id: D1
    description: "Review changeset committed at docs/reviews/2026-11-19-v1.0/ (REVIEW.md + 46 evidence files); old nested dir and zip gone"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "git ls-files -- docs/reviews/2026-11-19-v1.0/{REVIEW.md,evidence/} -> 1 + 46"
        status: pass
      - kind: other
        ref: "git status --porcelain -- docs/ -> empty"
        status: pass
    human_judgment: false
  - id: D2
    description: "Publication guard: docs/reviews/2026-11-19-v1.0/internal/ uncommitable (committed .gitignore block)"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md -> .gitignore:35 match"
        status: pass
      - kind: other
        ref: "git ls-files / porcelain scoped to non-public prefix -> 0 hits; four non-public filenames -> 0 staged anywhere"
        status: pass
    human_judgment: false
  - id: D3
    description: "Stale pre-move path references swept from .planning/{PROJECT,ROADMAP,REQUIREMENTS}.md (4 hits)"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "grep -rn '2026-11-19-v1.0-review' .planning/{PROJECT,ROADMAP,REQUIREMENTS}.md -> 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "docs/IDEA.md historical banner (as-built trio links + Phase 9 note), body byte-untouched incl. five D-13 lines"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "git diff --numstat HEAD~1 HEAD -- docs/IDEA.md -> 6 added, 0 deleted"
        status: pass
      - kind: other
        ref: "head -20 docs/IDEA.md greps: historical=1, trio+Phase 9 anchors present"
        status: pass
    human_judgment: true
    rationale: "Banner wording/loudness is editorial judgment; automated greps verify structure (anchors, link style, pure addition) but not readability"
  - id: D5
    description: "PROJECT.md:66 parenthetical names IDEA.md the historical intent-at-start document and points at banner + as-built trio"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "grep -c 'historical intent-at-start' .planning/PROJECT.md -> 1; 'authoritative, self-contained design document' -> gone"
        status: pass
    human_judgment: false
duration: 6min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 1: Review Artifact Relocation, Publication Guard, Reference Sweep, Historical Banner Summary

**Committed the 2026-11-19 v1.0 review (REVIEW.md + 46 evidence files) at its flattened public home with a committed .gitignore guard keeping the non-public Spec material uncommitable, swept all stale pre-move references, and marked docs/IDEA.md historical with its body byte-untouched.**

## Performance

- **Duration:** 6min (349s)
- **Started/Completed:** 2026-11-19T14:39:58Z / 2026-11-19T14:45:47Z
- **Tasks:** 3
- **Files modified:** 51 (2 created-by-move top-level artifacts: REVIEW.md + evidence/ tree of 46 files; 5 edited: .gitignore, docs/IDEA.md, 3 planning docs)

## Accomplishments

- Relocated the untracked nested review directory to the flattened layout `docs/reviews/2026-11-19-v1.0/` (REVIEW.md, evidence/, internal/) and removed the old nested directories and the archive zip (D-02, D-03)
- Installed the committed publication guard (D-04): `.gitignore` block `# Non-public review material (upstream Spec; never committed)` + `docs/reviews/2026-11-19-v1.0/internal/`, proven matching via `git check-ignore -v` BEFORE any staging; staged explicit paths only; porcelain reviewed pre-commit
- Committed the review changeset: REVIEW.md + all 46 evidence files (c791181)
- Swept all 4 stale `docs/2026-11-19-v1.0-review/` references across the three planning docs; AGENTS.md and .claude/CLAUDE.md confirmed untouched (zero references, verified) (4cdf23a)
- Prepended the loud historical banner to docs/IDEA.md (blockquote+bold, as-built trio links in the repo's relative-link style, Phase 9 note for docs/ARCHITECTURE.md); body byte-untouched (6 added, 0 deleted; five D-13 conflict-journal lines intact) and adjusted the PROJECT.md:66 design parenthetical to the historical wording (3b9569d)

## Task Commits

1. **Task 1: Relocate review artifacts, install publication guard, commit the review changeset** - `c791181` (docs)
2. **Task 2: Sweep stale pre-move path references from planning docs** - `4cdf23a` (docs)
3. **Task 3: Mark docs/IDEA.md historical (banner) and adjust the PROJECT.md design parenthetical** - `3b9569d` (docs)

**Plan metadata:** committed with the plan close-out docs commit.

## Files Created/Modified

- `docs/reviews/2026-11-19-v1.0/REVIEW.md` - committed review record (moved from nested duplicate; source of §7/§13/§14 for plans 02-04)
- `docs/reviews/2026-11-19-v1.0/evidence/` - 46-file evidence archive (docs-audit.md, review-engine.md, review-repl-api.md, test-audit.md, repro/, stand/)
- `docs/reviews/2026-11-19-v1.0/internal/` - moved local-only, gitignored, never committed (SPEC.md, SPEC-amended.md, SPEC-amendments.diff, spec-trace.md)
- `.gitignore` - new final guard block (comment + `docs/reviews/2026-11-19-v1.0/internal/`)
- `docs/IDEA.md` - 6-line historical banner prepended above the H1; body byte-untouched
- `.planning/PROJECT.md` - :15 and :137 old-path refs updated; :66 parenthetical reworded historical
- `.planning/ROADMAP.md` - :5 old-path ref updated
- `.planning/REQUIREMENTS.md` - :6 old-path ref updated
- deleted: `docs/2026-11-19-v1.0-review.zip` (untracked archive, D-03), old nested directories `docs/2026-11-19-v1.0-review/`

## Decisions Made

- Verify-gate scoping: the plan's negative awk patterns matching any path containing `internal/` cannot coexist with acceptance criterion "46 evidence files committed" (22 evidence repro paths legitimately contain `internal/` in their module layout). Gates were scoped to the actual prohibition target — the directory prefix `docs/reviews/2026-11-19-v1.0/internal/` named by D-01/D-04, must_haves.key_links.pattern, and threat T-01-01 — and publication safety was additionally proven by asserting none of the four non-public filenames is staged anywhere and the spec-identifying-strings grep over the staged set is clean. No content change resulted; only verification methodology.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Task 1 negative-verify patterns false-positive on legitimate evidence payload**
- **Found during:** Task 1 (staging porcelain review)
- **Issue:** The plan's verify/acceptance awk patterns `/internal/` (over `git status --porcelain -- docs/` and `git ls-files -- docs/reviews/`) match 22 legitimate staged evidence paths under `evidence/repro/**/internal/**` (review repro-test module trees). Following them literally is impossible while also satisfying acceptance criterion 2 (`git ls-files ... evidence/ | wc -l` prints 46): the two plan assertions contradict each other. The plan's own prohibition names the narrow target: `docs/reviews/2026-11-19-v1.0/internal/` (SPEC.md, SPEC-amended.md, SPEC-amendments.diff, spec-trace.md).
- **Fix:** Scoped both negative gates to that prefix (`grep -c 'reviews/2026-11-19-v1\.0/internal/'` → 0 on both tracked listing and porcelain) and added three stronger publication asserts: none of the four non-public filenames staged anywhere (0), the ignored dir invisible as untracked in porcelain (0), spec-identifying strings (`confstor|0031|cyp-config|cyp-perf`) over the staged set (0). All clean.
- **Files modified:** none (verification methodology only)
- **Verification:** all six publication asserts pass; guard proven at `.gitignore:35`
- **Commit:** c791181 (changeset verified under the corrected gates)

**2. [Note - criterion calibration] Task 2 acceptance "grep -c totals 4" measures 5**
- **Found during:** Task 2 verification
- **Issue:** `.planning/ROADMAP.md:56` already carried the post-move path `docs/reviews/2026-11-19-v1.0/` at HEAD before this plan (planner-written phase-table line), so the new-path count after the sweep is 5, not the 4 the criterion assumed.
- **Fix:** none needed — the substantive gates all hold: exactly the 4 stale hits replaced (diff confined to those lines, 4 insertions/4 deletions), zero pre-move hits remain, AGENTS.md/.claude/CLAUDE.md untouched.
- **Files modified:** none (arithmetic note only)
- **Verification:** grep old path → 0; `git diff --stat` → 4 lines across 3 files
- **Commit:** 4cdf23a

**Total deviations:** 1 auto-fixed (Rule 1, verification scoping) + 1 calibration note.
**Impact on plan:** none on content or publication safety; the corrected gates are strictly more precise about the one-way risk.

## Issues Encountered

None beyond the documented deviations. The stale `.git/gsd-plan-head-before-01-01` ledger from a prior milestone's plan 01-01 was detected and refreshed to this plan's true base (378b27e) before any commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-02 (spec deviation registry) can now read its derivation sources at the settled paths: `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` (local-only, guarded) and `docs/reviews/2026-11-19-v1.0/REVIEW.md` §7/§13/§14 (committed)
- Plan 01-03 (owner-decision ADRs) sources REVIEW.md §14 at the committed path
- Plan 01-04 (phase verification) gates assert against the new paths and the guard
- The one-way publication decision (D-01) is executed; the guard follows every clone. Subsequent plans must keep staging explicit-path only (never `git add docs/` or `-A`)

## Self-Check: PASSED

- key-files exist on disk: REVIEW.md, evidence/ (46 files), internal/spec-trace.md (local-only), docs/IDEA.md — all FOUND
- evaluation-scope check `--plan 01-01 --commits-only --raw`: exit 0 (commits >= 1)
- All task acceptance criteria re-run post-commit: PASS (46 evidence files tracked; porcelain docs/ empty; guard matches at .gitignore:35; IDEA.md numstat 6 added/0 deleted; banner anchors present; PROJECT.md historical wording present, authoritative claim gone)
- Tracer feedback gate (Task 1): verify re-run end-to-end after commit — all four automated checks pass (auto mode active; expanded per gate chain)

---
*Phase: 01-spec-reference-and-decisions*
*Completed: 2026-11-19*
