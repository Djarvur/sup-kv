# Phase 4: Protocol - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-22
**Phase:** 4-Protocol
**Mode:** --auto (all areas auto-selected; recommended option chosen for every question)

---

## Auto-selected areas and choices

| Area | Question | Selected |
|------|----------|----------|
| Closed interface | How is "complete" recognized? | SyncID/ChunkIndex/ChunkCount accumulation; gate backstop only on loss (Spec mechanism, R-03) |
| Empty-disk start | How long does an empty node wait? | `startup_gate_empty` 60 s (14.6/ADR 0006) |
| Stale records | Ack or answer? | Answer with the newer record, no Ack (R-04) |
| New peer sync | When is a full listing sent? | First-seen Hello, exactly once (R-05) |
| Notification order | Accept regressive notifications? | Ignore with a counter (R-10) |
| Duplicate identity | How is a duplicate node_id handled? | Named config-error class + no listing (R-12) |
| Version mismatch | Where is the version checked? | Pre-body-decode, own class, rate-limited WARN (R-13) |
| Totality / N | Scope of totality + N≥3? | Full totality (dump-failed writes, snapshot loads) + documented/tested N≥3 (S-02/PROTO-09) |
| Review criticals | Where do CR-01/CR-02 land? | This phase (D-09): bound the restart-listing queue + (node_id, run_id) dedupe; egress carries identity only |

## ADR 0005 reopening window

Owner decision 14.5(6) was accepted by default and reopenable at this discussion per ADR 0005.
**Not reopened** — kept exactly as accepted (a read of a listed-but-not-yet-arrived key returns
`Unavailable`). Any research conflict must surface as REVISION_CONFLICT.

## Claude's Discretion

- SyncID allocation scheme, chunk arithmetic, doc split (this phase vs Phase 9), unconfirmed-write counter name, restart-pending state shape.

## Deferred Ideas

None — discussion stayed within phase scope.
