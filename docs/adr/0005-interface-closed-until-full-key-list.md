---
adr: 0005
review-decision: "14.5 (Interface stays closed until the full key list arrives)"
informs: [PROTO-01]
date: 2026-11-19
status: accepted
---

# 0005 — The interface stays closed until the full key list arrives

**accepted (by default; reopenable at Phase 4/6 discussion before implementation)**

## Context

During startup reconciliation a node knows only part of the cluster state,
yet v1.0 serves reads and writes while the full key list is still arriving
(review finding R-03; stand scenario F14). Rule (6) below — answering
`Unavailable` for a key that is listed but whose value has not arrived yet —
was added at the Spec amendment and awaited owner confirmation; SPEC-02
records it as accepted by default, reopenable at the Phase 4/6 discussion
before implementation. Owner decision 14.5 settled the whole behavior on
2026-11-19 (`docs/reviews/2026-11-19-v1.0/REVIEW.md` §14 row 5).

## Decision

The whole interface — reads and writes — stays closed until the node has
received the full key list with versions from a neighbor:

1. the neighbor sends the list immediately to a node it sees for the first
   time (R-05 → PROTO-04);
2. `DataSync` carries `SyncID`, `ChunkIndex` and `ChunkCount`;
3. the node remembers the list: `create_if_absent` answers `AlreadyExists`
   for any listed key, live or deleted, even before the value arrives;
4. the HLC is updated from the versions carried by the list;
5. health stays `NOT_SERVING` until the list is complete;
6. reading a key that is listed but whose value has not arrived yet answers
   `Unavailable` — the accepted-by-default rule this ADR carries the loud
   status line for.

If part of the list is lost, the node opens by timer; an explicit re-request
of the missing part remains an open question in the Spec.

## Consequences

PROTO-01 implements the closed-interface startup in Phase 4, together with
the sync header and immediate-first-sync mechanics it needs. The separate
startup wait for an empty-disk node is decided in ADR 0006. The Phase 4
agent must weigh this decision before implementation: rule (6) was accepted
by default and may be reopened at the Phase 4 discussion.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 5 (rule 6 accepted
  by default per SPEC-02; wording mirrors the SPEC-02 acceptance
  parenthetical, .planning/REQUIREMENTS.md)
- Registry rows (docs/SPEC-DEVIATIONS.md), the S-08 family closing as
  fixed-in-v1.1(PROTO-01) with this ADR in its link column: S-HLC-04,
  S-ALG-RC-05, S-ALG-ST-06, S-PROTO-18
- Informs: PROTO-01 (the interface stays closed until the full key list
  arrives; the 14.5(6) `Unavailable` rule is part of its acceptance)
