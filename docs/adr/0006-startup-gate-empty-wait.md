---
adr: 0006
review-decision: "14.6 (An empty-disk node waits longer for a peer)"
informs: [PROTO-02]
date: 2026-11-19
status: accepted
---

# 0006 — An empty-disk node waits longer for a peer (`startup_gate_empty`)

## Context

When a node starts and its neighbor is unreachable, the current uniform
`startup_gate` (1 s) treats an empty node and a node with data alike. An
empty node that opens immediately answers from an empty state before it has
learned the cluster's. Owner decision 14.6 settled the behavior on
2026-11-19 (`docs/reviews/2026-11-19-v1.0/REVIEW.md` §14 row 6).

## Decision

A node with data on disk opens via `startup_gate` (1 s), as now. A node with
an empty disk waits longer for a peer: a separate startup parameter, named
`startup_gate_empty` in the Spec, default 60 s.

## Consequences

A new startup parameter — configuration validation must know it (OBSV-02).
The first start of a pair is not slowed: two empty nodes exchange empty
lists immediately and open without waiting out the new parameter. The node
replacement guide (DOCS-03) documents what happens when the neighbor never
appears. PROTO-02 implements the parameter in Phase 4, alongside the
closed-interface startup of ADR 0005.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 6
- Related: ADR 0005 (the closed-interface startup this wait complements)
- Informs: PROTO-02 (the `startup_gate_empty` parameter, default 60 s)
