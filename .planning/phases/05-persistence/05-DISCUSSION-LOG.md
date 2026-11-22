# Phase 5: Persistence - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-22
**Phase:** 5-Persistence
**Mode:** --auto (all areas auto-selected; recommended option chosen for every question)

---

## Auto-selected areas and choices

| Area | Question | Selected |
|------|----------|----------|
| Value limit | How is the limit made single-source? | One exported constant, config-passed everywhere, codec reads config (P-01/Q-04) |
| Load outcomes | How are outcomes visible? | Explicit outcome enum through the port + one WARN/ERROR per class (P-02) |
| Generation integrity | How is truncation detected? | Record count/terminator in the format (P-03) |
| Dirty start | Post-backup/empty start state? | Dirty + first write re-establishes the good generation (P-04) |
| Failure decay | Persistent dump failure behavior? | Bounded backoff + ERROR/metric; health stays SERVING (P-05) |
| Memory | How is the two-state-size bound proven? | Streaming encode + documented model (P-06) |
| Tie-break | Random tail or deterministic? | Deterministic, ADR-recorded (P-07) |
| Deadlines | How is a hung fsync bounded? | Bounded confirmation wait + bounded shutdown (P-08) |
| Temp/darwin | Remove or repair the darwin helper? | Remove; one portable fsync path (P-09) |
| Skipped records | Oversize records on load? | Count + WARN + metric, evidence preserved by rotation (P-10) |
| Engine split | How far to split? | Five internal roles, public surface stable (Q-03/PERSIST-11) |
| Timer seam | Same seam as repl? | Yes, shared clock/timer contract (Q-06/PERSIST-12) |

**Notes:** P-01 is already pinned by frozen tests in `internal/stand` and
`internal/persistfile`; the phase's job is code until the rows flip, then the TSV change only.
The Phase-4 review's New-1 (`markDirty` nudge) is folded into PERSIST-04.

## Claude's Discretion

- Exact record-count/terminator encoding, corrupt-file naming, backoff constants, confirmation
  timeout default, orphan-temp age bound, internal split type names.

## Deferred Ideas

- `sync_chunk_bytes` bounds and the four counter surfaces: Phase 7.
- `TestCore_keyCountCostFlat` ratio flake: Phase 8.
