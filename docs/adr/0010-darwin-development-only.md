---
adr: 0010
review-decision: "14.10 (darwin is development-only)"
informs: [DOCS-10, XPORT-05]
date: 2026-11-19
status: accepted
---

# 0010 — darwin is development-only

## Context

On darwin, datagrams larger than ~9 KiB do not leave the machine (review
finding R-08; §13 open questions row 3 notes the same limit), which silently
breaks replication for realistic record sizes. Owner decision 14.10 settled
the platform status on 2026-11-19 (`docs/reviews/2026-11-19-v1.0/REVIEW.md`
§14 row 10): macOS is for development only.

## Decision

darwin is demoted to development-only — not a production platform. The
~9 KiB send limit is documented, not fixed.

## Consequences

- darwin leaves the release archives, or its archives are marked unsuitable
  for production (DOCS-10);
- the README's "Quickstart on macOS" section is aligned with the
  development-only status; the v1.0 macOS quickstart remains the
  Linux-CI-verified + darwin loud-skip pair until revisited;
- XPORT-05 documents the darwin ~9 KiB send limit instead of fixing it;
- building and unit tests on macOS remain supported.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 10
- Out of Scope: "darwin as a production platform — owner decision 14.10:
  development-only; limits documented, not fixed" — .planning/REQUIREMENTS.md
- Informs: DOCS-10 (darwin demoted in README and release archives), XPORT-05
  (the darwin ~9 KiB send limit documented, not fixed)
