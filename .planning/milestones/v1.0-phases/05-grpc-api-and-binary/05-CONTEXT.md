# Phase 5: gRPC API and Binary - Context

**Gathered:** 2026-11-08
**Status:** Ready for planning

<domain>
## Phase Boundary

sup-kv becomes a usable product: a single static binary exposing the typed gRPC surface over the existing engine (persist.Engine + repl.Core) — Get/Set (create-if-absent)/Delete, List under prefix, Watch stream, snapshot create/load, optional `wait_for_replication` on writes, the gRPC standard health endpoint reflecting the startup gate, and the operational binary surface (flags/env config, `--version`, systemd-friendly lifecycle).

Scope anchor: requirements API-01..API-07 and TOOL-01. The engine, replication, persistence and their seams already exist (Phases 2–4); this phase adds the API layer and the binary, not new storage or replication behavior.

</domain>

<decisions>
## Implementation Decisions

### Watch stream (API-03)
- **D-01:** A Watch event carries the full record — key, version, value, deleted flag. The subscriber never needs a second Get to act on a change. — **Reversibility:** costly — this is the client-facing wire contract; changing the payload shape later breaks every subscriber.
- **D-02:** Tombstones are visible: deletes arrive as events with `deleted=true`. The stream has no holes — a subscriber sees every state change.
- **D-03:** Subscription starts with a snapshot of the current state followed by live changes. The snapshot is delivered as ordinary Watch events in the same single format — no special `initial_state` message type. — **Reversibility:** costly — client contract.
- **D-04:** Watch accepts a prefix filter, symmetric with List (API-02). `prefix=""` means all keys.
- **D-05:** Event order is arrival order at the node (local and replicated alike). No global re-sorting by version; state converges via CRDT merge, the stream is notifications, not a log.
- **D-06:** Events fire only on actual state changes; a merge loser is silently absorbed (no event for a version that did not become the stored record).
- **D-07:** No origin marking — local and replicated changes are indistinguishable in the stream (API-03: "local and replicated alike").

### wait_for_replication (API-05)
- **D-08:** The report is an aggregate: `acked` boolean plus counts (`acked_count` / `total_peers`). No per-peer breakdown — the replication topology is not exposed through the API.
- **D-09:** The wait timeout is a per-request RPC field with a config default (`wait_timeout`). A client may wait longer than the default on a cold start; the default is operator-tunable.
- **D-10:** The peer set is the snapshot of known peers at write time. Peers discovered (Hello) during the wait do not extend the awaited set — deterministic, easy to reason about.
- **D-11:** Zero known peers ⇒ `acked=true` immediately. A single-node deployment is a valid configuration with nothing to replicate.

### List under prefix (API-02)
- **D-12:** One RPC returns the full sorted key list for the prefix — no pagination. The scale envelope (thousands to tens of thousands of keys) fits a single response.
- **D-13:** List returns keys only; values are fetched via Get.
- **D-14:** Tombstones are not listed — List reflects live state, not history.
- **D-15:** Empty prefix means all keys (List without a filter enumerates the whole state).

### Binary and configuration (TOOL-01)
- **D-16:** Environment variables use the `SUPKV_` prefix: `SUPKV_NODE_ID`, `SUPKV_BIND_ADDR`, etc. — **Reversibility:** one-way — this is the published operational contract; systemd units and deployment scripts bind to it.
- **D-17:** Precedence: flag > env > default. The nearest instruction wins.
- **D-18:** `--version` prints version + commit + Go build version, injected via ldflags at build time — enough to identify a running build during incident triage.
- **D-19:** SIGTERM triggers a graceful shutdown: stop accepting gRPC requests, stop replication, then a final dump with write-confirmation. Committed data is never left only in memory across shutdown. — **Reversibility:** reversible.

### Claude's Discretion
- Codegen layout and tooling for the gRPC .proto (package placement, whether to follow the phase-4 `replproto` protoc+Makefile precedent or adopt buf) — planner/researcher decide; the Makefile already has a working `proto` target pattern from 04-01.
- Internal structure of the API adapter between the gRPC service and `persist.Engine`/`repl.Core` — follow the phase-4 D-01 decoded-structs port-seam precedent.
- gRPC server options (keepalive, max message sizes consistent with `value_max_bytes`, interceptors) — standard choices.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design (authoritative)
- `docs/IDEA.md` — the reviewed, self-contained design document; § "API (gRPC)" defines the RPC surface, error codes (OK/NotFound/AlreadyExists/InvalidArgument/Unavailable), `wait_for_replication` semantics, parameters and their defaults (incl. `startup_gate` 1s). Do not re-litigate; only implementation-level questions are open.

### Requirements
- `.planning/REQUIREMENTS.md` — API-01..API-07, TOOL-01 (the phase's traceability contract).

### Project conventions
- `AGENTS.md` — layout (`cmd/sup-kv/` single binary, main() holds no testable logic), dependency policy (gRPC + protobuf enter go.mod when first used), lint gate rules.

### Prior-phase context
- `.planning/phases/04-replication-and-deterministic-simulator/04-CONTEXT.md` — the seam decisions this phase consumes (decoded-structs port seam, single event-loop core, Outbox accumulator, peer table handles).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/persist` (`Engine`): the running engine — Run loop with flush channel, `ApplyRemote` (Merge + markDirty + nudge), write-confirmation waiters, snapshot create/load. The gRPC write/read paths are thin adapters over it.
- `internal/app/repl` (`Core`): `PeerSnapshot` (peer table), pending/unacked tracking (`UnackedStats`), `NotifyLocalWrite` (returns `ErrRunLoopExited` after Run — phase-4 WR-01 fix). These are the primitives for `wait_for_replication` (D-08..D-11) and for the startup-gate signal (first full sync round).
- `internal/port/` — the port-seam precedent: `dump.go` (whole-bytes seam, phase 3), `repl.go` (decoded-structs seam, phase 4). The gRPC layer's port seam follows `repl.go`'s shape.
- `internal/udprepl/replproto/` + Makefile `proto` target — the working protoc + protoc-gen-go v1.36.12 codegen pattern (04-01), committed generated output.

### Established Patterns
- Depguard architecture boundary: `internal/app` is pure (no net/os/syscall/google.golang.org/*); `internal/port` is stdlib+own-module only. The gRPC adapter lives outside `internal/app`; the engine surface crossing into it must go through a port seam.
- Consumer-side interfaces declared where consumed (storage.Clock precedent; port.StateStore in repl.go).
- Sequential event loops driven by handler methods with injected `now` — no goroutines in decision cores.

### Integration Points
- `cmd/sup-kv/` skeleton exists since Phase 1; main() receives flags/env/signals here (per AGENTS.md).
- Health endpoint: standard gRPC health checking; `NOT_SERVING` until the startup gate opens, `SERVING` after (API-06) — the gate signal comes from repl.Core's first full sync round or `startup_gate` elapsing.
- Startup gate rejects writes with `Unavailable`, serves reads immediately (API-04) — engine + repl already provide both halves.

</code_context>

<specifics>
## Specific Ideas

No specific references beyond IDEA.md — the design document is the authority; this discussion only pinned the open implementation choices recorded above.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 5-gRPC API and Binary*
*Context gathered: 2026-11-08*
