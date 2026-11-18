# Phase 2: Engine — HLC Versioning and LWW Store - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-02
**Phase:** 2-Engine — HLC Versioning and LWW Store
**Areas discussed:** reporting events from the pure core; HLC time source; interfaces in internal/port; property-test tooling

---

## Reporting events from the pure core

| Option | Description | Selected |
|--------|-------------|----------|
| A — return from method | `Merge` returns the list of losing records; clock methods return a drift signal. No hidden actions; later phases decide where to write them. | ✓ |
| B — callback handlers | Handler functions passed to the constructor; the library calls them itself on each event. | |
| C — accumulate inside | The library collects events in memory; the caller takes a snapshot on demand. | |

**User's choice:** A — return from method
**Notes:** Recommended by Claude; user agreed. Covers CORE-07 (conflict journal source) and VERS-05 (drift warning).

## HLC time source

| Option | Description | Selected |
|--------|-------------|----------|
| A — func in constructor | `now func() time.Time` given at construction; production passes `time.Now`, tests a controlled clock with jumps/stuck time. | ✓ |
| B — parameter of each call | Callers pass `now` into `Tick`/`Update` on every invocation. | |
| C — TimeSource interface | A formal time-source contract declared in `internal/port` now. | |

**User's choice:** A — func in constructor
**Notes:** C noted as contradicting the Phase 1 `port/doc.go` note ("interfaces appear with the phases that need them"); B pushes `now` through all calling code.

## Interfaces: now or later

| Option | Description | Selected |
|--------|-------------|----------|
| A — later, with consumers | Concrete types only in Phase 2; Phases 3–5 each declare the interface they need in `internal/port`. | ✓ |
| B — declare now | Fix the full Store/Clock contract in `internal/port` before consumers exist. | |

**User's choice:** A — later, with consumers
**Notes:** Matches Phase 1 decision and Go consumer-side interface practice; single author/implementation makes early abstraction low-value.

## Property-test tooling

| Option | Description | Selected |
|--------|-------------|----------|
| A — standard library | Hand-written deterministic generators; printed seed for exact failure replay; manual minimization; zero new dependencies. | ✓ |
| B — rapid library | `pgregory.net/rapid`: automatic counterexample minimization; a new test-only dependency in `go.mod`. | |

**User's choice:** A — standard library
**Notes:** Consistent with the project's zero-direct-requires principle; the property spaces here (small record sets, clock jumps) are tractable by hand.

---

## Claude's Discretion

API signatures, type placement inside `internal/app`, `Conflict` record fields, error representation, PRNG choice — planner decides per go-ultimate.

## Deferred Ideas

None.

## Session Notes

- The discussion was restarted once at the user's request: an earlier run used a non-existent Russian word («семы», a calque of "seam"); the user instructed never to use jargon or invented terms. Recorded as standing feedback.
- The TUI's question UI garbles long Cyrillic option descriptions (overlapping text); worked around by putting full explanations in chat text and keeping option labels short.
