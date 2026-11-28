---
adr: 0007
review-decision: "14.7 (Public proto package; the last-read-value file stays outside sup-kv)"
informs: [APIS-06]
date: 2026-11-19
status: accepted
---

# 0007 — Public proto and consumer zone

**accepted (by default; reopenable at Phase 4/6 discussion before implementation)**

## Context

How do consumers integrate sup-kv (review S-11, A-06)? sup-kv serves as the
storage for consumer services, and the client-side "last read value" file
pattern — a local fallback copy of the last read value for outage tolerance —
was open: the review's reading placed it in the consumer's client library,
outside sup-kv, and that interpretation awaited owner confirmation. SPEC-02
records it as accepted by default, reopenable at the Phase 4/6 discussion
before implementation. Owner decision 14.7 settled the integration shape on
2026-11-19 (`docs/reviews/2026-11-19-v1.0/REVIEW.md` §14 row 7).

## Decision

sup-kv stays a separate program. The proto file and the generated gRPC code
move out of `internal/` into a public package, with buf set up for it; a Go
client example builds in CI as a separate module. The last-read-value file
belongs to the consumer's client library: the pattern is documented, and the
code lives with the consumer, not in sup-kv.

## Consequences

APIS-06 lands the public package, the buf setup and the CI-built client
example (Phase 6); docs/API.md documents the last-read-value cache pattern as
consumer-library territory. Client caching libraries stay out of sup-kv
(Out of Scope). Mapping snapshot operations onto the consumer's proto remains
a v2 question (SNAP-01, review open question 7). The Phase 6 agent must
weigh this decision before implementation: the last-read-value placement was
accepted by default and may be reopened at the Phase 6 discussion.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 7 (accepted by
  default per SPEC-02; wording mirrors the SPEC-02 acceptance parenthetical,
  .planning/REQUIREMENTS.md)
- Registry rows (docs/SPEC-DEVIATIONS.md): S-AVL-03, S-IMP-06, S-IMP-10,
  S-IMP-13 (consumer-zone), S-IMP-04 (deferred-v2(SNAP-01))
- Informs: APIS-06 (proto and generated code public; the last-read-value
  file documented as consumer-library territory, 14.7 accepted)
- Out of Scope: "Client caching libraries shipped in sup-kv" —
  .planning/REQUIREMENTS.md
