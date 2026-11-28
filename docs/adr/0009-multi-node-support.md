---
adr: 0009
review-decision: "14.9 (Three-or-more-node support is completed)"
informs: [PROTO-09]
date: 2026-11-19
status: accepted
---

# 0009 — Three-or-more-node support

## Context

The design is N nodes from day one, but v1.0's testing and documentation
center on the pair (§13 open questions row 8: replication between pairs — N
nodes in one broadcast segment, no explicit address list). Owner decision
14.9 settled the scope on 2026-11-19
(`docs/reviews/2026-11-19-v1.0/REVIEW.md` §14 row 9): support for several
nodes is brought to completion.

## Decision

Multi-node support is completed: behavior and timing are described for three
or more nodes and covered by tests.

## Consequences

- PROTO-09 documents 3+-node behavior and timing, and the
  `wait_for_replication` semantics at N > 2 go into the API docs;
- real-socket tests with three nodes are added;
- `TestChaos_convergence` is stabilized (T-01 → TEST-01) as the multi-node
  correctness vehicle.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 9; §13 open
  questions row 8
- Informs: PROTO-09 (three-or-more-node behavior described and tested);
  TEST-01 (chaos stabilization) provides the evidence base
