---
adr: 0001
review-decision: "14.1 (Snapshot load is a full rollback as per Spec)"
informs: [APIS-07]
date: 2026-11-19
status: accepted
---

# 0001 — Snapshot load is a full rollback

## Context

The v1.0 snapshot load merges the snapshot into the live map instead of
restoring it: keys created after the snapshot was taken survive the load
(review stand finding F13; recorded as deviation S-01). Owner decision 14.1
settled the question on 2026-11-19 (`docs/reviews/2026-11-19-v1.0/REVIEW.md`
§14 row 1); the Spec was amended the same day to match the decision.

## Decision

Snapshot load is a full rollback, as the Spec requires: keys absent from the
snapshot are deleted on load. The result of the load is broadcast to peers so
every replica converges to the restored state (the S-02 closure, delivered
with PROTO-08).

## Consequences

`TestEngine_snapshotCreateLoadRestore`
(internal/app/persist/engine_test.go) pins the merge behavior today and is
rewritten for the rollback behavior. APIS-07 implements the rollback in
Phase 6; PROTO-08 carries the load result to peers.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 1 (decision dated
  2026-11-19; the Spec was brought in line the same day)
- Registry rows (docs/SPEC-DEVIATIONS.md): S-PERS-12 (the
  merge-instead-of-rollback deviation itself), S-PERS-13, S-SEM-08,
  S-ALG-CU-09 (every change replicates, including the snapshot-load result)
- Informs: APIS-07 (snapshot load is a full rollback); companion: PROTO-08
  (every local change replicates, including snapshot loads)
