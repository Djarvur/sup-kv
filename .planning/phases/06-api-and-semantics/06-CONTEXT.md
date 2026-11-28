# Phase 6: API and Semantics - Context

**Gathered:** 2026-11-22
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected; every choice logged below and reviewable in git)

<domain>
## Phase Boundary

Make the gRPC API tell the truth: `wait_for_replication` reports what actually happened, errors are total and path-free, Watch and health behave as documented, `List` stops copying the world, snapshot load is a full rollback, and the proto is public so consumers can build against it. Requirements APIS-01…07.

**Depends on:** Phase 5 — persistence outcomes and the confirmation seam (Watch-after-confirmation, snapshot rollback broadcast), on top of Phase 3's Outbox result seam and Phase 4's closed-window semantics.

</domain>

<decisions>
## Implementation Decisions

### Honest wait (APIS-01)
- **D-01:** `wait_for_replication` reports the write-time peer set: the set of peers known when the write was accepted is captured, zero known peers returns `(false, 0, 0)` immediately, and an `Ack` from a node that was not in that set never yields `true`. A displaced version resolves within 20 ms through the existing `Superseded` Outbox result (Phases 3/5), not by idling to the caller's timeout. The wait stays caller-bounded and its contract is documented. — [auto] [Honest wait] Q: what does honesty mean here? → Selected: write-time peer set + immediate zero-peer answer + Superseded fast path (A-01). — **Reversibility:** costly — public RPC semantics.

### Pagination (APIS-02)
- **D-02:** `List` becomes keyset-paginated over the sorted key→version view: an opaque page token (empty = first page), bounded default and maximum page sizes, and per-page cost independent of value sizes — the request never copies the whole store. Values ride only for the entries of the page. The semantics (ordering, token stability, end-of-list signal) are documented. — [auto] [List] Q: paginate or stream? → Selected: keyset pagination with bounded pages (A-02). — **Reversibility:** costly — public RPC shape.

### Errors, Watch, server settings (APIS-03/04/05)
- **D-03:** Error mapping is total and path-free: one mapping table covers every internal error; a missing snapshot maps to `NotFound`, a request needing a dump with no dump configured maps to `FailedPrecondition`, and no message carries a server filesystem path. Internal detail is logged, never returned. A test walks the RPC matrix and rejects path shapes in every status message. — [auto] [Errors] Q: how is totality proven? → Selected: exhaustive mapping table + path-free message test (A-03). — **Reversibility:** reversible.
- **D-04:** The Watch contract: local changes are announced only after the dump confirmation that made them durable; each confirmed write yields exactly one event; deletions are events; ordering follows versions; server shutdown ends streams with `Unavailable`, never `OK`. The contract text lands in the API docs (Phase 9 completes the surrounding document). — [auto] [Watch] Q: when is an event announced? → Selected: after dump confirmation, exactly once, Unavailable on shutdown (A-04). — **Reversibility:** costly — consumer-visible ordering.
- **D-05:** gRPC server settings are deliberate and documented: explicit keepalive parameters, explicit max receive/send message sizes tied to the protocol constants, and a health state machine that goes `NOT_SERVING` when shutdown begins and never returns to `SERVING` afterwards. A test pins the health transition and the settings. — [auto] [Server settings] Q: which settings are pinned? → Selected: keepalive + message-size ceilings + one-way health transition (A-05). — **Reversibility:** reversible.

### Public proto and rollback (APIS-06/07)
- **D-06:** The consumer-facing API proto and its generated Go code move outside `internal/` (a public package path), while the internal replication wire proto stays internal. A Go client example builds in CI as a separate module. The last-read-value file pattern is documented as consumer-library territory (14.7, accepted), not a server feature. — [auto] [Public proto] Q: what moves, what stays internal? → Selected: API proto public, replication proto internal, client example module in CI (A-06/14.7). — **Reversibility:** costly — import paths.
- **D-07:** Snapshot load is a full rollback (14.1): keys absent from the snapshot are deleted, the result broadcasts to peers (Phase 5 totality), and `TestEngine_snapshotCreateLoadRestore` is rewritten to prove the new behavior. The full-rollback semantics are documented. — [auto] [Rollback] Q: additive or rollback? → Selected: full rollback with broadcast (S-01/14.1). — **Reversibility:** costly — data semantics.

### Claude's Discretion
- Page-size defaults and maximum, token encoding, keepalive constants, the public package path, and the exact API-docs split between this phase and Phase 9.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Decisions and requirements
- `.planning/REQUIREMENTS.md` — APIS-01…07 verbatim acceptance
- `.planning/ROADMAP.md` — Phase 6 goal + 5 success criteria
- `docs/adr/` — the phase-1 register (14.1 full rollback, 14.7 consumer zone), ADR 0012
- `docs/SPEC-DEVIATIONS.md` — rows this phase closes (A-01…A-06, S-01 family)

### Review evidence
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — §7 A-01…A-06, §14.1/14.7
- `.planning/phases/04-protocol/04-REVIEW-DISPOSITION.md` — deferred items owned by this phase (IN-03 metrics surface is Phase 7, not here; WR-04 replay pacing was fixed in Phase 4)
- `.planning/phases/05-persistence/05-REVIEW-DISPOSITION.md` — Phase-5 deferrals (IN-04 is Phase 7)

### Code
- `internal/grpcapi/` — `server.go` (List/Watch/Set/Snapshot), `wait.go` (WaitReplicated seam), `watch.go` (hub), `gate.go` (health/gate), `apiproto/` (generated API code today)
- `internal/udprepl/repl.go` — `WaitReplicated` tracker, notification seam, `NotifyReplayedWrite`
- `internal/app/repl/core.go` — Outbox results (`Superseded`), notification guard
- `internal/app/persist/` — `LoadStats`, snapshot orchestration, full-rollback decision point
- `.planning/phases/05-persistence/05-03-SUMMARY.md` — tie-break law and test patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Outbox result seam (`Acked`/`Superseded`) from Phases 3/5; confirmation accounting from Phase 5
- Watch hub with sender-error handling; gate health state from Phase 4
- Strict lint, depguard, doc-anchor tests

### Established Patterns
- Consumer-side interfaces; adapters own I/O; core goroutine-free
- One mapping surface per concern; error classes named and counted

### Integration Points
- `internal/grpcapi/apiproto` → public path (import churn across grpcapi/main/tests)
- `wait_for_replication` RPC ↔ `Adapter.WaitReplicated` tracker
- `watch.go` ↔ persistence confirmation seam (Phase 5)
- `TestEngine_snapshotCreateLoadRestore` rewrite in `internal/app/persist`

</code_context>

<specifics>
## Specific Ideas

- Path-free errors need a message-shape test, not a review checklist: scan every mapped status message for absolute-path shapes.
- The public proto move is a compile-wide import churn; the generated code must stay byte-identical in content, only relocated.
- Watch-after-confirmation already has the Phase-5 notification path; the contract needs the test that an unconfirmed write is not announced.

</specifics>

<deferred>
## Deferred Ideas

- `docs/API.md` final wording and the 18-row failure table are Phase 9/DOCS-05; this phase authors the contract sections it owns.
- The metrics surface for Phase-4/5 counters and the value-limit library guard are Phase 7.

</deferred>

---

*Phase: 6-API and Semantics*
*Context gathered: 2026-11-22*
