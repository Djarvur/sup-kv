# Phase 8: Performance - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-23
**Phase:** 8-Performance
**Mode:** --auto (all areas auto-selected; recommended option chosen for every question)

---

## Auto-selected areas and choices

| Area | Question | Selected |
|------|----------|----------|
| Latency model | Where do the two profiles live? | One model section, both profiles, each number with its conditions (S-FAIL-15, 14.3) |
| Burst bound | Stand the burst up or document it? | Standing test + explicit guarantee/non-guarantee text (R-02, PERF-03) |
| Amplification | Measure or record the gap? | Measure on target-grade media where available, else record the gap with the method and the missing hardware |
| Memory | Formula or points? | Formula + measured points with hardware conditions, tied to PERSIST-06 |
| Absent peer | Benchmark shape? | Paired benchmark, absent vs live peer, ratio bound; fix 04 New-3 while there |
| Perf target | One target or profiles? | One `make perf` with central thresholds and hardware blocks; CI keeps the threshold-free smoke target |
| Flakes | Fix or record? | Fix the classes with deterministic assertions; never relax a bound to green |

**Notes:** the routed flake classes are binding inputs from the Phase-3/4/5/6/7 ledgers. Decision
14.3 (dump policy) stands — no design change in this phase; only documentation and measurement.

## Claude's Discretion

- Location of the performance model document, threshold values and their basis, benchmark names,
  hardware-block shape, and the mechanism for deterministic ratio gates (operation counters vs
  serialized measurement).

## Deferred Ideas

- Dead code and the registry resweep: Phase 9.
- Dump-policy redesign: out of scope (ADR 0003).
