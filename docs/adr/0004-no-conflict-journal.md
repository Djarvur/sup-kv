---
adr: 0004
review-decision: "14.4 (No conflict journal)"
informs: [OBSV-03]
date: 2026-11-19
status: accepted
---

# 0004 — No conflict journal

## Context

The pre-amendment Spec required a conflict journal recording lost versions;
docs/IDEA.md promised one and a comment in the store names one. The review
took the question to the owner (§13 open questions row 6). Owner decision
14.4 settled it on 2026-11-19 (`docs/reviews/2026-11-19-v1.0/REVIEW.md`
§14 row 4): no journal. The Spec amendment removed the requirement the same
day.

## Decision

The conflict journal is not built. The conflict counter is fixed instead: it
counts only the displacement of a local unconfirmed record by another value —
not ordinary overwrites.

## Consequences

Three promised-journal surfaces are corrected in later phases, deliberately
not in this phase (consistent with the phase's no-double-editing decision
D-13):

- the conflict-counter fix lands with OBSV-03 (Phase 7);
- the journal promises in docs/IDEA.md (five places) and .claude/CLAUDE.md
  are scrubbed in Phase 9 together with the docs audit (M-01..M-52);
- the store.go comment fix is Phase 7 territory (OBSV-03 / 14.4).

Until those phases land, the stale promises remain visibly historical under
the docs/IDEA.md banner.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 4
- Registry rows (docs/SPEC-DEVIATIONS.md): S-SEM-07, S-FAIL-05
  (superseded-by-amendment — the journal requirement was removed by decision
  14.4)
- Informs: OBSV-03 (honest metric definitions, including the conflict
  counter)
- Out of Scope: "Conflict journal — owner decision 14.4: not built; the
  counter is fixed instead (OBSV-03)" — .planning/REQUIREMENTS.md
