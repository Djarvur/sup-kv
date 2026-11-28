---
phase: 01-spec-reference-and-decisions
plan: 03
subsystem: docs
tags: [adr, slim-madr, owner-decisions, review-section14, d11-loud-lines, spec-compliance]
requires:
  - "01-01 review relocation: docs/reviews/2026-11-19-v1.0/REVIEW.md committed — §14 (rows at :396-406, dating note :408) is the source of every ADR"
  - "01-02 registry: docs/SPEC-DEVIATIONS.md Links cells cite ADR numbers, so ADR back-references resolve on arrival"
provides:
  - "Eleven slim-MADR ADRs docs/adr/0001..0011-<slug>.md covering owner decisions 14.1-14.11 exactly once (SPEC-02's file deliverable)"
  - "Explicit 14.x <-> ADR <-> informed v1.1-REQ mapping in every ADR's lowercase-key frontmatter (adr, review-decision, informs, date, status)"
  - "The two loud accepted-by-default status lines (D-11), verbatim and bold under the H1: ADR 0005 (14.5(6) Unavailable rule) and ADR 0007 (14.7 last-read-value placement)"
  - "Honest-gap records: ADR 0002 (no v1.1 REQ — behavior already correct) and ADR 0011 (loss-share measurement has no dedicated v1.1 requirement)"
affects: [01-04-phase-verification, phase-4-protocol-fixes, phase-6-api-fixes, phase-9-docs]
actuals:
  tokens: 4745
  tasks: 3
  commits: 2
  note: "chars/4 over the 11 authored ADR files (18,982 chars); the 38000 estimate assumed heavier per-ADR authoring — slim MADR with §14 as the already-argued source lands far under"
plan_head_before: c32467c2b7b2593692e3ff82feaac238e9dee210
plan_head_after: 9f56500c41b8fd005654d6f344aaf762e07aa054
tech-stack:
  added: []
  patterns: ["slim-MADR ADR (Context/Decision/Consequences/Links) with lowercase-key frontmatter as the machine-readable 14.x <-> REQ mapping (D-09/D-10)", "loud bold status line directly under the H1 for accepted-by-default decisions (D-11)"]
key-files:
  created:
    - "docs/adr/0001-snapshot-load-full-rollback.md"
    - "docs/adr/0002-create-if-absent-blocked-by-tombstone.md"
    - "docs/adr/0003-dump-policy-unchanged.md"
    - "docs/adr/0004-no-conflict-journal.md"
    - "docs/adr/0005-interface-closed-until-full-key-list.md"
    - "docs/adr/0006-startup-gate-empty-wait.md"
    - "docs/adr/0007-public-proto-consumer-zone.md"
    - "docs/adr/0008-metrics-off-by-default.md"
    - "docs/adr/0009-multi-node-support.md"
    - "docs/adr/0010-darwin-development-only.md"
    - "docs/adr/0011-value-limit-60kib-kept.md"
  modified: []
key-decisions:
  - "ADR bodies translate §14's substance into English without re-arguing it (D-09); 14.5's sub-rules carried as a numbered decision list so the Phase 4 agent can weigh rule (6) — the accepted-by-default item — specifically"
  - "Honest gaps recorded instead of invented mappings: ADR 0002 informs [] (behavior already correct; Out of Scope rationale in Links) and ADR 0011 states the 14.11 loss-share measurement has no dedicated v1.1 requirement (RESEARCH Pattern 3)"
  - "ADR 0008's Links cite registry row S-IMP-08, which plan 02 closed as accepted-deviation(ADR 0008) — the back-reference resolves in both directions; 0001/0007 likewise cite the exact registry rows the plan names"
patterns-established: ["slim-MADR ADR set 0001..0011 with frontmatter mapping — the repo's first ADR directory", "loud-line placement directly under the H1, before ## Context"]
requirements-completed: [SPEC-02]
coverage:
  - id: D1
    description: "ADRs 0001-0006 exist with the locked slugs, five-key lowercase frontmatter, review-decision citing 14.1-14.6, no options-analysis sections; 0005 carries the verbatim D-11 line under the H1"
    requirement: SPEC-02
    verification:
      - kind: other
        ref: "ls docs/adr/ | grep -cE '^000[1-6]-' -> 6; grep -L '^review-decision:' docs/adr/000[1-6]-*.md -> empty; five-key check -> 6 files x 5 keys"
        status: pass
      - kind: other
        ref: "grep -c 'reopenable at Phase 4/6 discussion before implementation' docs/adr/0005-*.md -> 1; options-analysis heading grep -> no matches"
        status: pass
    human_judgment: false
  - id: D2
    description: "ADRs 0007-0011 exist; 0007 carries the D-11 line; 0008 Context cites the O-04 resolution via 14.8 (REVIEW.md:271); 0011 records the honest 14.11 gap"
    requirement: SPEC-02
    verification:
      - kind: other
        ref: "ls docs/adr/ | grep -cE '^00(07|08|09|10|11)-' -> 5; grep -c 'no dedicated v1.1 requirement' docs/adr/0011-*.md -> 1; grep -c 'O-04' docs/adr/0008-*.md -> 3"
        status: pass
    human_judgment: false
  - id: D3
    description: "Full-set gates: exactly 11 files numbered 0001..0011; the eleven review-decision values cover 14.1-14.11 each exactly once; the D-11 line appears in exactly 0005 and 0007; all 11 files git-tracked"
    requirement: SPEC-02
    verification:
      - kind: other
        ref: "ls docs/adr/*.md | wc -l -> 11; review-decision 14.x census -> 11 values x 1 each; grep -l D-11 line -> 0005 + 0007 only; git ls-files -- docs/adr/ | wc -l -> 11"
        status: pass
    human_judgment: false
  - id: D4
    description: "ADR prose carries §14's substance in English without re-arguing it; every Links section cites its §14 row and the named registry rows/REQ IDs resolve against docs/SPEC-DEVIATIONS.md and .planning/REQUIREMENTS.md"
    requirement: SPEC-02
    verification:
      - kind: other
        ref: "grep -L '§14 row' docs/adr/*.md -> empty; 0001 cites S-PERS-12/S-PERS-13/S-SEM-08/S-ALG-CU-09 (4/4); 0007 cites S-AVL-03/S-IMP-04/S-IMP-06/S-IMP-10/S-IMP-13 (5/5)"
        status: pass
    human_judgment: true
    rationale: "Fidelity of the Russian-to-English substance translation is the D-09 authoring discipline itself; the automated gates verify mapping and structure, not prose fidelity — every ADR was written fresh from a full read of its §14 row"
duration: 9min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 3: Owner Decision ADRs (14.1-14.11) Summary

**Eleven slim-MADR ADRs at docs/adr/0001..0011 record owner decisions 14.1-14.11 exactly once each, with the explicit 14.x ↔ ADR ↔ v1.1-REQ frontmatter mapping and the two loud accepted-by-default status lines (0005, 0007) — SPEC-02's deliverable is in place.**

## Performance

- **Duration:** 9min (524s)
- **Started/Completed:** 2026-11-19T15:23:49Z / 2026-11-19T15:32:33Z
- **Tasks:** 3
- **Files created:** 11 (docs/adr/0001..0011-*.md, 18,982 chars total)

## Accomplishments

- Authored the eleven ADRs in slim MADR shape (Context / Decision / Consequences / Links — no options-analysis ceremony), English, each translating its REVIEW.md §14 row in substance and citing the row rather than re-arguing it (D-09)
- Gave every ADR the explicit mapping via lowercase-key frontmatter: `adr: 000N`, `review-decision: "14.N (title)"`, `informs: [REQ IDs]`, `date: 2026-11-19`, `status: accepted` (D-10); Links sections carry the §14 row reference plus the affected registry rows and informed REQ IDs
- Placed the verbatim loud line `**accepted (by default; reopenable at Phase 4/6 discussion before implementation)**` directly under the H1 of ADR 0005 (14.5(6): read of a listed-but-not-yet-arrived key → `Unavailable`) and ADR 0007 (14.7: last-read-value file outside sup-kv), wording mirrored from the SPEC-02 acceptance parenthetical (D-11)
- Recorded the honest gaps without inventing mappings: ADR 0002 informs [] with the Out-of-Scope rationale in Links; ADR 0011 states plainly in Consequences that the 14.11 loss-share measurement has no dedicated v1.1 requirement
- Made the registry back-references resolve bidirectionally where plan 02 created them: ADR 0001 ↔ S-PERS-12, ADR 0004 ↔ S-SEM-07/S-FAIL-05, ADR 0007 ↔ S-AVL-03/S-IMP-04/S-IMP-06/S-IMP-10/S-IMP-13, ADR 0008 ↔ S-IMP-08 (accepted-deviation(ADR 0008))
- Passed the full gate block: 11 files; 11 unique review-decision values covering 14.1-14.11 each exactly once; D-11 line in exactly 0005 and 0007; 11 unique `adr:` values; five frontmatter keys in every file; every Links section cites its §14 row; no options-analysis headings; all 11 files git-tracked

## Task Commits

1. **Task 1: Author ADRs 0001-0006 (decisions 14.1-14.6)** - `d31803b` (docs)
2. **Task 2: Author ADRs 0007-0011 (decisions 14.7-14.11)** - `9f56500` (docs)
3. **Task 3: Verify the ADR set mapping and commit** - verification-only; no commit needed (all 11 files were already tracked by the Task 1-2 commits; the plan's Task 3 commit applies only to remaining unstaged files, and none remained)

**Plan metadata:** committed with the plan close-out docs commit.

## Files Created/Modified

- `docs/adr/0001-snapshot-load-full-rollback.md` - 14.1; informs APIS-07; cites S-PERS-12/S-PERS-13/S-SEM-08/S-ALG-CU-09; names the TestEngine_snapshotCreateLoadRestore rewrite
- `docs/adr/0002-create-if-absent-blocked-by-tombstone.md` - 14.2; informs []; Out-of-Scope rationale in Links
- `docs/adr/0003-dump-policy-unchanged.md` - 14.3; informs PERF-01; latency-model/fast-response-profile consequences
- `docs/adr/0004-no-conflict-journal.md` - 14.4; informs OBSV-03; Phase 9 (IDEA.md scrub, M-01..M-52) and Phase 7 (store.go comment) timing per D-13
- `docs/adr/0005-interface-closed-until-full-key-list.md` - 14.5; informs PROTO-01; **D-11 loud line**; six numbered sub-rules incl. the 14.5(6) `Unavailable` rule
- `docs/adr/0006-startup-gate-empty-wait.md` - 14.6; informs PROTO-02; `startup_gate_empty` default 60 s
- `docs/adr/0007-public-proto-consumer-zone.md` - 14.7; informs APIS-06; **D-11 loud line**; cites S-AVL-03/S-IMP-04/S-IMP-06/S-IMP-10/S-IMP-13
- `docs/adr/0008-metrics-off-by-default.md` - 14.8; informs DOCS-10; Context resolves O-04 (REVIEW.md:271); Links cite S-IMP-08
- `docs/adr/0009-multi-node-support.md` - 14.9; informs PROTO-09; 3+-node behavior/timing, TEST-01 evidence base
- `docs/adr/0010-darwin-development-only.md` - 14.10; informs DOCS-10, XPORT-05; ~9 KiB limit documented, not fixed
- `docs/adr/0011-value-limit-60kib-kept.md` - 14.11; informs PERSIST-01; honest gap: loss-share measurement has no dedicated v1.1 requirement

## Decisions Made

- **Rule-level detail for 14.5.** The §14 row packs six sub-rules into one cell; the ADR carries them as a numbered decision list so the reopened item (rule 6) is individually addressable at the Phase 4 discussion, and Context identifies rule (6) as the accepted-by-default item the loud line covers.
- **Honest gaps, no invented mappings.** ADR 0002's `informs: []` is intentional (the behavior is already correct — there is nothing to inform), and ADR 0011 names the missing requirement instead of stretching PERSIST-01 to cover the loss-share measurement.
- **Bidirectional registry linkage where it exists.** ADR Links cite exactly the registry rows that reference them back (0001/0004/0007/0008), keeping the citation graph consistent without retrofitting plan 02's committed cells.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The per-plan commit ledger `.git/gsd-plan-head-before-01-03` already existed from the v1.0 milestone's plan 01-03 (base b0904f2, dated Oct 31). Same pre-existing condition plans 01-01 and 01-02 documented; refreshed to the true base (c32467c, HEAD at plan start) before any commit, so the SUMMARY's `actuals.commits` (2) is measured from the correct range.
- `check evaluation-scope --plan 01-03 --commits-only` resolves plan IDs across all history and includes the prior milestone's 01-03 commits (dependency currency work) in its list. Exit 0 as required; the authoritative commit evidence for this plan is the ledger measurement (2: d31803b, 9f56500).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-04 (phase verification) can gate SPEC-02 against the committed set: the exact greps it needs (11 files, 11 unique 14.x decisions, D-11 line in 0005+0007) all pass today
- Phase 4/6 agents hold the reopen decision: ADRs 0005 and 0007 put the loud accepted-by-default line and its scope in the artifact they will open first
- Later phases reference ADRs by number (REQUIREMENTS.md already carries the 14.x ↔ REQ parentheticals; TEST-13 requires every section-14 decision referenced by name in tests)
- Phase 9's ADR index (DOCS-10) can be generated from the frontmatter — every file carries machine-readable adr/review-decision/informs keys

## Self-Check: PASSED

- key-files exist on disk: all 11 docs/adr/0001..0011-*.md — FOUND
- commits measured from ledger: 2 (d31803b, 9f56500; plan_head_before c32467c → plan_head_after 9f56500) — >= 1
- `check evaluation-scope --plan 01-03 --commits-only --raw`: exit 0 (includes stale prior-milestone 01-03 history — see Issues Encountered; the ledger measurement is the authoritative count)
- All task acceptance criteria re-run post-commit: PASS (6+5 files with locked slugs; 5 frontmatter keys each; 14.1-14.11 each exactly once; D-11 line in exactly 0005 and 0007; no options-analysis headings; 0001 cites its four registry rows; 0011 records the honest gap; `git ls-files -- docs/adr/ | wc -l` -> 11)
- Plan <verification> re-run: all three blocks green (Task 1, Task 2, Task 3 gates)

---
*Phase: 01-spec-reference-and-decisions*
*Completed: 2026-11-19*
