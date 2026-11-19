---
adr: 0008
review-decision: "14.8 (Metrics stay off by default)"
informs: [DOCS-10]
date: 2026-11-19
status: accepted
---

# 0008 — Metrics off by default

## Context

The metrics endpoint is disabled unless enabled, so the Spec-required metrics
are not available out of the box (review item O-04,
docs/reviews/2026-11-19-v1.0/REVIEW.md:271). Owner decision 14.8 settled the
question on 2026-11-19 (REVIEW.md §14 row 8): the endpoint stays off by
default. This decision also resolves review item O-04.

## Decision

Metrics remain off by default. `--metrics-addr` is an operations
requirement: the deployment guide documents enabling metrics, with a working
example.

## Consequences

The deployment guide gains the metrics setup section (DOCS-10, alongside the
dump-policy profile documentation of ADR 0003). The lifecycle/event-log work
(review O-01 → OBSV-01) proceeds independently of this decision. Honest
counter definitions — the conflict and drift counters still measure something
other than the Spec's definitions — are a separate fix under OBSV-03/OBSV-05.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 8; review item O-04
  at REVIEW.md:271, resolved by 14.8
- Registry row (docs/SPEC-DEVIATIONS.md): S-IMP-08
  (accepted-deviation(ADR 0008) — all seven required metric families exist
  while the endpoint stays off unless enabled)
- Informs: DOCS-10 (dump and metrics setup in the deployment guide, 14.3 and
  14.8)
