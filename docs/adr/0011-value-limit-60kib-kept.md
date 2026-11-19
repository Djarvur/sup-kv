---
adr: 0011
review-decision: "14.11 (The ~60 KiB single-value limit stays)"
informs: [PERSIST-01]
date: 2026-11-19
status: accepted
---

# 0011 — The ~60 KiB single-value limit stays

## Context

The Spec left open the question of the ~60 KiB value limit and fragment
losses (§13 open questions row 3: no position taken; stand measurements in
review section 4; on darwin, records above ~9 KiB do not leave the machine —
R-08). Owner decision 14.11 settled it on 2026-11-19
(`docs/reviews/2026-11-19-v1.0/REVIEW.md` §14 row 11): the limit stays.

## Decision

The ~60 KiB single-value limit is kept: `value_max_bytes` stays at its
61 440-byte default, and larger blobs remain out of scope — they are a
different product.

## Consequences

- The share of lost datagrams is measured on a real link between boards, and
  the documentation gains the dependence of large-value delivery time on
  losses (stand figures in review section 4). **Honest gap:** this loss-share
  measurement has no dedicated v1.1 requirement — it is recorded here
  plainly rather than invented into a mapping; the delivery-time
  documentation lands adjacent to the PERF phase's latency-model work.
- The limit itself is defined in one place and enforced: PERSIST-01 keeps
  the 61 440 constant single-sourced and passed by configuration (with
  XPORT-03 validating the `value_max_bytes` ceiling at startup).

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 11; §13 open
  questions row 3
- Informs: PERSIST-01 (the value limit defined in one place and enforced)
- Out of Scope: "Values > 60 KiB — larger blobs are a different product" —
  .planning/PROJECT.md
