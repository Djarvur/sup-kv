---
adr: 0002
review-decision: "14.2 (Create-if-absent on a deleted key stays blocked by the tombstone)"
informs: []
date: 2026-11-19
status: accepted
---

# 0002 — A deleted key blocks create-if-absent with AlreadyExists

## Context

What does `create_if_absent` answer when the key exists only as a tombstone
(was deleted)? The review posed the question to the owner (§13 open
questions row 5 records the related tombstone-cleanup position). Owner
decision 14.2 settled it on 2026-11-19 (`docs/reviews/2026-11-19-v1.0/REVIEW.md`
§14 row 2), and the Spec amendment adopted the behavior the same day.

## Decision

A deleted key blocks the operation: `create_if_absent` answers
`AlreadyExists`, and the key stays deleted until an ordinary write recreates
it. The code already behaves this way; nothing changes.

## Consequences

No code change. Together with keeping tombstones forever (garbage collection
is deferred to v2 as GARB-01), a deleted key returns to the map only through
an ordinary write — no path resurrects it implicitly. Later phases must not
"fix" this behavior. (For a key known from a startup sync list but whose
value has not arrived yet, ADR 0005 gives `create_if_absent` the same
`AlreadyExists` answer.)

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 2
- Registry row (docs/SPEC-DEVIATIONS.md): S-ALG-CU-10
  (superseded-by-amendment — the amendment adopted this behavior as Spec)
- No v1.1 requirement: intentionally none — the behavior is already correct;
  see the Out of Scope table in .planning/REQUIREMENTS.md
  ("`create_if_absent` unblocking a deleted key")
