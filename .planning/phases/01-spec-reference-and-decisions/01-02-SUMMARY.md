---
phase: 01-spec-reference-and-decisions
plan: 02
subsystem: docs
tags: [spec-deviation-registry, d08-enum, section7-closures, amendment-adjudication, own-words-audit]
requires:
  - "01-01 review relocation: derivation sources at settled paths — local-only docs/reviews/2026-11-19-v1.0/internal/spec-trace.md (167 rows) + SPEC-amended.md; committed docs/reviews/2026-11-19-v1.0/REVIEW.md §7/§13/§14"
provides:
  - "docs/SPEC-DEVIATIONS.md — the complete deviation registry: preamble (derivation source, D-08 status enum, S-12 closure note) + 13 sections + 167 rows in the D-06 schema ID | Summary | Status | Links"
  - "Every review §7 deviation (S-01, S-02, S-06, S-07, S-08, S-11) closed per the D-08 mapping; S-12 closed in the preamble"
  - "Machine-checkable registry vocabulary: Phase 9's unclosed-row grep (DOCS-08) can run against this file as-is"
affects: [01-03-owner-decision-adrs, 01-04-phase-verification, phase-3-transport-fixes, phase-4-protocol-fixes, phase-5-persistence-fixes, phase-9-docs]
actuals:
  tokens: 11000
  tasks: 3
  commits: 1
  note: "chars/4 over the authored changeset (registry 38.1K chars + planning docs); estimate (50000) covered authoring 167 adjudicated rows — landed under because scripted derivation did the mechanical half"
plan_head_before: 3aba32d391c18a73184265c361be6e876df264b8
plan_head_after: ffbe162301c2737b93213ffa90c171e9253635df
tech-stack:
  added: []
  patterns: ["scripted derivation with mechanical gates (count/contiguity/enum/substring) as the authoring safety net for a 167-row registry", "one-way reference expansion of local-only shorthand code refs to full repo paths via the trace's own file legend"]
key-files:
  created:
    - "docs/SPEC-DEVIATIONS.md"
  modified: []
key-decisions:
  - "Registry Links cells expand the trace's shorthand code refs (core:/store:/eng:) to full repo paths per the trace's own file legend, so public readers resolve links without the local-only legend"
  - "S-PROTO-08 (trace Implemented-with-deviation) closed as implemented with the recorded-intent nuance in the summary cell (04-CONTEXT D-02) — no owner decision covers it, so accepted-deviation(14.x) would have been an invented reference"
  - "S-HLC-07 accepted-deviation references the recorded v1.0 research assumption (02-RESEARCH A2) instead of an ADR/14.x — none of ADR 0001..0011 (14.1..14.11) covers the zero filler; amend-check confirmed the amendment left the random-filler requirement standing (SPEC-amended.md L245)"
  - "Rows not named by the plan/research resolved by the priority rule against REQ IDs verified in REQUIREMENTS.md: S-PROTO-02/03 → XPORT-03, S-PAR-05 → PERSIST-01 (datagram half: XPORT-03 in Links), S-ALG-RC-09 → XPORT-07, S-PAR-16 → OBSV-05, S-FAIL-17 → OBSV-06, S-FAIL-03 → PROTO-03, S-HLC-10 → PROTO-01, S-IMP-08 → accepted-deviation(ADR 0008) with counter-honesty fixes (OBSV-03/OBSV-05) named in Links"
patterns-established: ["registry status flip workflow for later phases: one cell edit per row, gates re-run", "preamble-not-row placement for findings without a statement ID (S-12)"]
requirements-completed: [SPEC-01]
coverage:
  - id: D1
    description: "Registry committed: preamble + 13 sections + 167 rows in the D-06 schema; per-prefix contiguity proves no row lost or duplicated"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "grep -c '^| S-' docs/SPEC-DEVIATIONS.md -> 167; awk contiguity gate exit 0; awk '^### ' count -> 13"
        status: pass
      - kind: other
        ref: "commit ffbe162: exactly one file, 259 insertions, 0 deletions; git ls-files -> 1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every §7 deviation closed per the D-08 mapping: S-PERS-12 → APIS-07 (+ADR 0001); S-02 rows → PROTO-08; S-06 rows → OBSV-05; S-07 rows → OBSV-06; S-08 rows → PROTO-01/03/04; S-11 rows → consumer-zone/deferred-v2(SNAP-01); S-12 in the preamble"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "grep -oE closure tokens -> 7 unique; consumer-zone lines -> 5; deferred-v2(SNAP-01) -> 1; XPORT-01/XPORT-02/PERF-02 rows -> 3"
        status: pass
    human_judgment: false
  - id: D3
    description: "All 167 status cells non-empty and inside the six-value D-08 enum; no TBD anywhere"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "awk empty-status gate exit 0; awk enum-membership gate exit 0; grep TBD -> 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "Own-words rule: no registry summary cell is a copy/substring of its trace statement cell; summaries authored as one-line English glosses"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "awk substring audit (trace statement cells vs registry summary cells) exit 0"
        status: pass
    human_judgment: true
    rationale: "The audit catches exact copies only; near-verbatim translation avoidance is the D-06 authoring discipline itself — glosses were written fresh from a full read of the trace, with intentional rephrasing away from the normative sentence structure"
  - id: D5
    description: "Status adjudication: 12 amendment-removed rows + S-SEM-07 superseded-by-amendment; §7 closures per D-08; flagged rows (S-HLC-07, stand-downgraded trio, S-ALG-ST-06) per the research resolutions; amend-check run against SPEC-amended.md (hunk inventory confirms the amendment surface)"
    requirement: SPEC-01
    verification:
      - kind: other
        ref: "bidirectional trace-vs-registry status join: 116 Implemented -> 114 implemented + 2 locked flips; census 115 implemented / 12 superseded / 4 consumer-zone / 1 deferred / 2 accepted / 33 fixed-in-v1.1 = 167"
        status: pass
    human_judgment: true
    rationale: "Per-row status choice for rows not named by the plan applied the locked priority rule (§14 > §7 > trace note, amend-check first) with REQ mappings verified against REQUIREMENTS.md; the machine gates prove enum membership, the judgment is in the mapping choices documented under key-decisions"
duration: 13min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 2: Spec Deviation Registry Summary

**Seeded docs/SPEC-DEVIATIONS.md in one pass: all 167 statement IDs grouped under 13 sections, every status drawn from the fixed six-value D-08 enum, every review §7 deviation closed per the locked mapping, and all completeness gates green — Phase 9's unclosed-row grep can run against this file today.**

## Performance

- **Duration:** 13min (787s)
- **Started/Completed:** 2026-11-19T15:06:52Z / 2026-11-19T15:19:59Z
- **Tasks:** 3
- **Files created/modified:** 1 (docs/SPEC-DEVIATIONS.md, 259 lines, 38.1K chars)

## Accomplishments

- Authored the complete registry (167 rows, D-06 schema `ID | Summary | Status | Links`) with own-words English summaries written fresh from a full read of the 167-row local-only trace — never copied, never hand-transcribed from the statement column
- Grouped rows under exactly 13 English section headings mirroring the trace's subsections, preserving the trace's row membership 1:1 (per-prefix count == max suffix, mechanically proven)
- Adjudicated all statuses: the 12 amendment-removed rows plus S-SEM-07 as superseded-by-amendment; §7 closures exactly per D-08 (S-PERS-12 → APIS-07 + ADR 0001; S-PROTO-14/S-ALG-CU-09/S-PERS-13/S-SEM-08 → PROTO-08; S-HLC-13/S-FAIL-14 → OBSV-05; S-ALG-CU-08/S-PERS-09/S-PROTO-04 → OBSV-06; the S-08 families → PROTO-01/03/04; four consumer-zone rows + ADR 0007; S-IMP-04 → deferred-v2(SNAP-01)); flagged rows per the research resolutions (S-HLC-07 accepted-deviation, stand-downgraded trio → XPORT-01/XPORT-02/PERF-02, S-ALG-ST-06 → PROTO-01)
- Ran the amend-check against SPEC-amended.md (hunk inventory) before adjudicating non-implemented rows; confirmed the amendment left the random-filler requirement standing (S-HLC-07 stays a live accepted deviation) and adopted RunID-as-random-bytes (S-PROTO-12)
- Closed S-12 in the preamble (registry exists + IDEA.md historical banner), never as a table row — it has no Spec statement ID
- Passed the full gate block: 167 rows, per-prefix contiguity, empty-status, enum membership, own-words substring audit, §7 closure tokens (7 unique REQ tokens, 5 consumer-zone lines, 1 deferral), no TBD, no empty cells
- Publication safety held: explicit-path staging only, zero `internal/` paths in the commit, zero deletions

## Task Commits

1. **Task 1: Scaffold registry and seed all 167 rows by scripted extraction** - verified in place (gates: 167 rows, contiguity, 13 headings, preamble anchors)
2. **Task 2: Adjudicate statuses — amend-check, section-7 closures, flagged rows** - verified in place (7 closure tokens, consumer-zone >= 4, deferral >= 1, per-row spot checks)
3. **Task 3: Run completeness gates, own-words audit, commit the registry** - `ffbe162` (docs)

Tasks 1-2 verified without intermediate commits: the plan defines a single registry commit at Task 3, after the full gate block — committing half-adjudicated intermediate states would put a broken registry in public history.

**Plan metadata:** committed with the plan close-out docs commit.

## Files Created/Modified

- `docs/SPEC-DEVIATIONS.md` - the deviation registry: preamble (derivation source, D-08 enum, S-12 note, ADR/REVIEW/G-xx/L-ref conventions) + 13 sections + 167 rows; status census 115 implemented / 12 superseded-by-amendment / 33 fixed-in-v1.1 / 4 consumer-zone / 2 accepted-deviation / 1 deferred-v2

## Decisions Made

- **Links cells carry full repo paths.** The trace's Code evidence column uses local shorthand (`core:631-644`, `eng:552-558`) defined in the trace's own file legend — which is local-only (D-01). The registry expands every shorthand to the full repo path (`internal/app/repl/core.go:631-644`) so public readers can resolve links; line ranges carried verbatim (REVIEW.md precedent).
- **S-PROTO-08 → implemented, nuance in the summary cell.** Trace marks it Implemented-with-deviation (duplicate-id warning repeats per datagram; same-run-id echo dropped silently). The spec requirement (log a config error and ignore) is satisfied and the behavior is recorded intent (04-CONTEXT D-02); no owner decision covers it, so `accepted-deviation(14.x)` would have invented a reference. The nuance is stated verbatim in the summary.
- **S-HLC-07 → accepted-deviation(02-RESEARCH A2).** D-08's grammar says `<ADR|14.x>`, but no ADR (0001..0011 map 14.1..14.11) or §14 decision covers the zero version filler; the acceptance lives in the recorded v1.0 research assumption. Amend-check first: SPEC-amended.md L245 still requires random filler, so the deviation is live and honestly referenced. Machine gate unaffected (non-empty parens).
- **Unnamed rows resolved by the priority rule with verified REQ mappings.** S-PROTO-02/03 → XPORT-03 (key length + ceiling validated), S-PAR-05 → PERSIST-01 (single enforced limit source; datagram half named XPORT-03 in Links), S-ALG-RC-09 → XPORT-07 (bounded, deduplicated queue), S-PAR-16 → OBSV-05 (parameter semantics change with the drift fix), S-FAIL-17 → OBSV-06 (same §7 S-07/G-10 logging gap as S-PROTO-04), S-FAIL-03 → PROTO-03 (stale-payload reply, G-04), S-HLC-10 → PROTO-01 (same omission as S-HLC-04), S-IMP-08 → accepted-deviation(ADR 0008) with the counter-honesty fixes (OBSV-03, OBSV-05) named in Links.

## Deviations from Plan

None - plan executed exactly as written. (The single-commit consolidation for Tasks 1-2 follows the plan's own Task 3 commit instruction; the stale commit ledger from the prior milestone's plan 01-02 was refreshed to the true base 3aba32d before SUMMARY measurement — same pre-existing condition plan 01-01 documented for its ledger.)

## Issues Encountered

- The per-plan commit ledger `.git/gsd-plan-head-before-01-02` already existed from the v1.0 milestone's plan 01-02 (base d3f306f, a v1.0 merge commit 94 commits back). Detected at SUMMARY time via the measured commit count; refreshed to the true base (3aba32d, HEAD at plan start) before recording `actuals`.
- The `check evaluation-scope --plan 01-02 --commits-only` self-check resolves plan IDs across all history and returned the prior milestone's 01-02 commits (CI workflows). Exit 0 as required; the authoritative commit evidence for this plan is the ledger measurement above plus commit ffbe162.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-03 (owner-decision ADRs) can cite registry rows: ADR numbers 0001-0011 are already referenced from Links cells, so each ADR's back-references resolve on arrival
- Plan 01-04 (phase verification) gates against a registry that already passes the D-08 enum and closure greps it will re-run
- Later phases flip statuses in place: one status-cell edit per row, gates re-run — no rows may be added or removed (D-07: registry complete on day one)
- The trace, SPEC-amended.md and the diff remain local-only under the committed .gitignore guard; every registry commit must keep staging explicit-path

## Self-Check: PASSED

- key-files exist on disk: docs/SPEC-DEVIATIONS.md — FOUND
- commits measured from ledger: 1 (ffbe162, plan_head_before 3aba32d..plan_head_after ffbe162) — >= 1
- `check evaluation-scope --plan 01-02 --commits-only --raw`: exit 0 (resolved against stale prior-milestone history — see Issues Encountered; ledger measurement is the authoritative count)
- All task acceptance criteria re-run post-commit: PASS (167 rows; contiguity exit 0; 13 headings; preamble anchors present; own-words audit exit 0; `git ls-files` -> 1)
- Plan <verification> re-run: all three blocks green

---
*Phase: 01-spec-reference-and-decisions*
*Completed: 2026-11-19*
