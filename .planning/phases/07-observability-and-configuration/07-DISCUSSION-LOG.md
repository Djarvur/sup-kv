# Phase 7: Observability and Configuration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-23
**Phase:** 7-Observability and Configuration
**Mode:** --auto (all areas auto-selected; recommended option chosen for every question)

---

## Auto-selected areas and choices

| Area | Question | Selected |
|------|----------|----------|
| Lifecycle logs | What does the catalog cover? | Startup+config, load generation, dump failure, gate, peers, drift at stable levels (O-01/S-07) |
| Drift | Does the runbook match the binary? | WARN + metric at `drift_warn`, runbook tied to the code path (S-06) |
| Validation | Where does validation live? | One config package + one function, exit 2, table-tested (O-02/Q-07) |
| Deferrals | Do the routed deferrals land here? | Yes: `sync_chunk_bytes` bounds, library value-limit guard, rate-limited apply WARN, unbound-listing class |
| Metrics | How are definitions confirmed? | One documented definition per metric + a test per definition (O-03/14.4), plus the four Phase-4 counters' surface |
| Exit codes | Which codes? | 2 usage / 1 startup / 3 final dump, ordered shutdown (O-05) |
| Boundaries | Which imports are ruled? | Direction rules + the observed cross-imports removed or justified (Q-05) |
| Decomposition | How far to split? | Four units in `udprepl` + wiring-only `main` (Q-07) |

**Notes:** The routed deferrals are binding inputs from the Phase-3/4/5 ledgers; the dead-code
deletion (Q-09) stays in Phase 9 and is not absorbed here.

## Claude's Discretion

- Catalog field names/levels, final-dump exit code (3 unless research grounds another), depguard
  rule names, split file names in `udrepl`.

## Deferred Ideas

- Dead code (Q-09): Phase 9.
- Performance thresholds and flakes: Phase 8.
