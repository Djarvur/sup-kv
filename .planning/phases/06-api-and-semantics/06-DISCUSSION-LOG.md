# Phase 6: API and Semantics - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-22
**Phase:** 6-API and Semantics
**Mode:** --auto (all areas auto-selected; recommended option chosen for every question)

---

## Auto-selected areas and choices

| Area | Question | Selected |
|------|----------|----------|
| Honest wait | What does honesty mean here? | Write-time peer set + immediate zero-peer answer + Superseded fast path (A-01) |
| List | Paginate or stream? | Keyset pagination with bounded pages (A-02) |
| Errors | How is totality proven? | Exhaustive mapping table + path-free message test (A-03) |
| Watch | When is an event announced? | After dump confirmation, exactly once, `Unavailable` on shutdown (A-04) |
| Server settings | Which settings are pinned? | Keepalive + message-size ceilings + one-way health transition (A-05) |
| Public proto | What moves, what stays internal? | API proto public, replication proto internal, client example module in CI (A-06/14.7) |
| Rollback | Additive or rollback? | Full rollback with broadcast (S-01/14.1) |

**Notes:** 14.1 (full rollback) and 14.7 (consumer-library last-value pattern) were accepted by
default in Phase 1 and are recorded as accepted decisions; this phase implements them rather than
reopening them.

## Claude's Discretion

- Page-size defaults/maximum, token encoding, keepalive constants, the public package path, and the
  API-docs split between this phase and Phase 9.

## Deferred Ideas

- `docs/API.md` final wording and the failure table: Phase 9/DOCS-05.
- Metrics surface and value-limit library guard: Phase 7.
