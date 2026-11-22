# Phase 4: Protocol - Context

**Gathered:** 2026-11-22
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected; every choice logged below and reviewable in git)

<domain>
## Phase Boundary

Implement the Spec's reconciliation semantics: a joining/restarting node's interface stays closed until the complete key list arrives; empty-disk nodes wait longer than data nodes (`startup_gate_empty`); a stale incoming record is answered with the newer one (never acked); new/returning peers get an immediate sync; notifications never regress; duplicate node_id is a named config error; proto_version is checked before body decode; replication is total (dump-failed writes, snapshot loads); N≥3 behavior is specified and tested. Requirements PROTO-01…09. Turns the remaining two red rows green: R-03 gate (2 tests) — P-01 stays red for Phase 5.

**ADR 0005 reopening window (D-11 of Phase 1):** owner decision 14.5(6) — a read of a listed-but-not-yet-arrived key returns `Unavailable` — was accepted by default and marked reopenable at this discussion. **Decision: NOT reopened** (kept exactly as accepted). If research surfaces a conflict, it must be raised as a REVISION_CONFLICT, not silently changed.

</domain>

<decisions>
## Implementation Decisions

### Closed interface until the complete listing (PROTO-01 / R-03 / 14.5)
- **D-01:** The startup gate opens only when the COMPLETE listing has arrived and been processed: `DataSync` gains `SyncID`/`ChunkIndex`/`ChunkCount`; the receiver accumulates parts of a SyncID and marks the listing complete only when all chunks 1..Count processed; `startup_gate` (1 s) remains only as the documented loss-backstop (a node that never receives a complete listing opens on it, per the Spec's open question on explicit re-request). While closed: reads and writes answer `Unavailable`; health stays `NOT_SERVING`; a read of a key known from the listing but whose value has not arrived returns `Unavailable` (14.5(6), accepted); `create_if_absent` for any listed key (live or tombstoned) returns `AlreadyExists`; the HLC updates from each part's max version. — [auto] [Closed interface] Q: how is "complete" recognized? → Selected: SyncID/ChunkIndex/ChunkCount accumulation, gate backstop only on loss (recommended default: the Spec's stated mechanism, review R-03/§14.5). — **Reversibility:** costly — gate semantics ride the RPC contract.
- **D-02:** An **empty-disk node waits for a peer**: new `startup_gate_empty` parameter (default 60 s, flag + `SUPKV_` env twin, validated in the same one-place surface Phase 7 will formalize). A node with data on disk opens via `startup_gate` as today; two empty nodes exchange empty listings and open without waiting for the new parameter; a node whose peer never appears stays closed until the parameter elapses (documented in the node-replacement runbook text, Phase 9's home). — [auto] [Empty-disk start] Q: how long does an empty node wait? → Selected: startup_gate_empty 60 s (recommended default: 14.6/ADR 0006). — **Reversibility:** costly — new public parameter.

### Peer interactions (PROTO-03/04/05/06/07)
- **D-03:** A stale `DataReplication` (received version ≤ held version) is answered with **exactly one unicast `DataReplication` carrying the newer record and NO `Ack`**; the sender's `wait_for_replication` for the stale version reports `acked: false` (Outbox result semantics from Phase 3). — [auto] [Stale records] Q: ack or answer? → Selected: answer with the newer record, no Ack (recommended default: R-04/third Replication outcome). — **Reversibility:** reversible.
- **D-04:** First `Hello` from an unknown `node_id` triggers **exactly one full listing toward it** (a repeat produces nothing); a peer returning after `peer_timeout` + 1 s receives the neighbor's state within N / `sync_response_rate` plus margin **without** the `sync_interval` term; `TestRestart_newPeerNoFullSync` is rewritten to the new contract. — [auto] [New peer sync] Q: when is a full listing sent? → Selected: first-seen Hello, exactly once (recommended default: R-05/PROTO-04). — **Reversibility:** reversible.
- **D-05:** Local-write notifications with a version not greater than the tracked one are **ignored** (never regress the mirror). — [auto] [Notification order] Q: accept regressive notifications? → Selected: ignore with a counter (recommended default: R-10). — **Reversibility:** reversible.
- **D-06:** A third process under a known `node_id` with **alternating `run_id`s is a named configuration error**: counted, logged at WARN on the Spec-mandated entry, receives no full listing. — [auto] [Duplicate identity] Q: how is a duplicate node_id handled? → Selected: config-error class + no listing (recommended default: R-12). — **Reversibility:** reversible.
- **D-07:** `proto_version` is checked **before body decode**; incompatible → class `bad_proto_version`, WARN rate-limited per peer per interval (the rate-limited WARN helper from 03-08's discard classes). — [auto] [Version mismatch] Q: where is the version checked? → Selected: pre-decode with its own class (recommended default: R-13). — **Reversibility:** reversible.

### Totality and N nodes (PROTO-08/09)
- **D-08:** Replication is **total**: a write whose dump failed still broadcasts (counted as unconfirmed) and a snapshot load broadcasts its result (with APIS-07's full-rollback semantics, Phase 6). 3+-node behavior and timing documented in the phase artifact; `wait_for_replication` semantics at N > 2 documented there too (Phase 6 folds both into docs/API.md; DOCS-05/APIS-05 own the final wording — handoff recorded in the phase report). Real-socket three-node tests land here. — [auto] [Totality/N] Q: scope of totality + N≥3? → Selected: full totality + documented/tested N≥3 (recommended default: S-02/PROTO-09). — **Reversibility:** reversible.

### Phase-3 review criticals in this domain (CR-01/CR-02)
- **D-09:** Fix both Phase-3 review criticals here, as explicit tasks: **CR-01** — bound the restart-listing queue and key dedupe by `(node_id, run_id)` instead of the datagram source handle (the D-04 first-Hello path is the same code; a hostile segment host must not accumulate FIFOs/queued chunks, and the per-trigger `Versions()` copy must not scale with store size); **CR-02** — the egress queue carries identity only and builds the message at drain (no pinned record copies, no Lookup before dedupe). Both get regression tests (overlay-style probes converted to ordinary tests where feasible). — [auto] [Review criticals] Q: where do CR-01/CR-02 land? → Selected: this phase (same domain, D-04/D-08 code paths). — **Reversibility:** reversible.

### Claude's Discretion
- Exact SyncID allocation (random vs HLC-seeded), chunk-count arithmetic, which node-replacement doc text lands here vs Phase 9, the unconfirmed-write counter name, the restart-pending state shape.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Decisions and requirements
- `docs/adr/0005-interface-closed-until-full-key-list.md` (14.5, the D-01 contract; 14.5(6) accepted — not reopened), `docs/adr/0006-startup-gate-empty-wait.md` (14.6), `docs/adr/0009-multi-node-support.md` (14.9)
- `.planning/REQUIREMENTS.md` — PROTO-01…09 verbatim acceptance
- `.planning/ROADMAP.md` — Phase 4 goal + 5 success criteria
- `docs/SPEC-DEVIATIONS.md` — rows this phase closes (S-08 family: S-HLC-04/S-ALG-RC-05/S-ALG-ST-06/S-PROTO-18)

### Review evidence
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — §5 R-03/R-04/R-05/R-10/R-12/R-13, S-02/S-08, §14.5/14.6/14.9, §15 phase composition
- `docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md` — F14 measured evidence (the 3 s default-seed displacement)

### Code
- `internal/app/repl/core.go` — handleHello/handleDataSync/handleDataReplication/notification seams (Post-Phase-3 shapes: Outbox results, unacked ledger, listing queues)
- `internal/udprepl/repl.go` — decode path (pre-body proto_version check slots beside the 03-08 discard classes), run_id tracking
- `internal/grpcapi/gate.go` + `server.go` — gate health/startup semantics (14.5(6) read behavior, AlreadyExists-on-listed)
- `.planning/phases/03-transport/03-REVIEW.md` + `03-REVIEW-DISPOSITION.md` — CR-01/CR-02 details for D-09
- `.planning/phases/02-stand-and-simulator-as-tests/02-05-SUMMARY.md` — the R-03 red tests this phase turns green (frozen, flip via TSV)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 3's Outbox result seam, unacked ledger, pacers, listing/egress queues, discard classes — all extend; nothing re-invented
- simnet (deletions/restart/budgets) + stand harness + red runner — the verification loop; R-03 tests already assert the full acceptance
- `version.Validate` + MaxClockLead from Phase 3 for any version received in listings

### Established Patterns
- Core goroutine-free, results via Outbox; adapter owns sockets/timers; depguard-enforced
- Deterministic simulator replay must stay byte-identical
- Strict lint; table-driven tests; red-gate contract (D-01: only the TSV flips)

### Integration Points
- `internal/stand/stand_red_test.go` — R-03 gate + F14 tests flip green (expectation rows)
- `scripts/red-expectations.tsv` — two R-03 rows flip `red → green`
- `gate.go`/`server.go` — the closed-window read/write/health contract (14.5(6))

</code_context>

<specifics>
## Specific Ideas

- The Spec-mandated WARN entries (duplicate node_id, foreign proto_version) must land at the right levels now — OBSV-06 (Phase 7) formalizes the catalog, but the log lines exist from this phase.
- The R-03 acceptance names exact behaviors (value on A unchanged; three-part listing keeps the interface closed for the first two; after the max-version part the next local write gets a greater version) — the frozen Phase-2 tests already assert them all; zero test edits needed.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 4-Protocol*
*Context gathered: 2026-11-22*

## Research resolutions (2026-11-22, phase-planning addendum)

Applied to the decisions above, with the research's evidence:
- **D-03 refined:** a **strictly older** `DataReplication` is answered with exactly one unicast newer record and NO Ack; an **equal-version** record is still Acked (the retransmission drain depends on it — R-02 would regress otherwise). S-PROTO-16's equal-Ack reading wins over the first "≤" draft.
- Delete on a listed-but-not-arrived key keeps local `NotFound` (14.5(6) names reads; no acceptance requires Unavailable for Delete).
- Three-node real-socket tests use in-process shared-port adapters (recommended).
- Listing drift does not feed `MergeStats` (OBSV-05 owns drift).
- `startup_gate_empty` gets basic relation validation now; Phase 7's one-point validation formalizes the catalog.
- The mixed 3-node window where the gate opens on an empty peer's complete listing is documented, not changed (D-01 stands).
- Wire constants: proto version 1→2; `sync_id` uint64 + `chunk_index`/`chunk_count` uint32; `chunkOverheadBytes` 64→96 (boundary fixtures re-derived).
- The frozen tests survive via one harness rule: the harness derives `--startup-gate-empty` from an explicit `--startup-gate` in extras (otherwise the new empty-gate default opens the replaced node after 1 s and the closed-side assertions race).
