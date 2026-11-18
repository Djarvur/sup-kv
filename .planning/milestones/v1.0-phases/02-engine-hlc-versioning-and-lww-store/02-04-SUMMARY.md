---
phase: 02-engine-hlc-versioning-and-lww-store
plan: 04
subsystem: engine
tags: [go, hlc, uuidv7, rfc-9562, lww-crdt, restart-seam, counter-seed, tdd, race-detector, stdlib-only]

requires:
  - phase: 02-engine-hlc-versioning-and-lww-store
    plan: "01"
    provides: Clock (Kulkarni Tick/Update, counterMax overflow, InitialL seam, signed drift), RecordVersion pack/unpack/Compare, Store with Merge and NewStore composition point
  - phase: 02-engine-hlc-versioning-and-lww-store
    plan: "02"
    provides: the six-test TEST-02 property suite (restart property, adversarial monotonicity, uniqueness) and the shared seeded generators (newRNG, nextWallTime)
provides:
  - Counter-seeded restart seam — NewClock's initialC uint16 parameter (masked to 12 bits) and Config.InitialC propagated by NewStore — so a restart seeds the clock from the dump's FULL watermark and every post-restart version strictly dominates every version in the dump (VERS-04 strict dominance, G-02-VERS04-DOMINANCE / WR-01 closed)
  - WR-01 regression locks at store level — TestStore_restartDominance variant A (a peer holding the dump watermark accepts the first post-restart write, no silent revert) and variant B (the stale watermark record displaces with exactly one Conflict, no equal-version silent drop)
  - Full-watermark restart property — TestClock_restartProperty over a (dumpMax, counter >= 1, ownNodeID) watermark across a 100-tick adversarial walk
affects: [03-persistence, 04-replication, 06-hardening]

actuals:
  tokens: 5035    # chars/4 over the realized diff (estimate was 20000, confidence low)
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - restart seams carry the dump's FULL watermark — maximum l plus maximum counter at that l — as constructor state, never l alone
    - mutation-checked regression tests: removing the seed reproduces the verifier's probe failures, proving the tests bind on the seam

key-files:
  created: []
  modified:
    - internal/app/clock.go
    - internal/app/store.go
    - internal/app/clock_test.go
    - internal/app/store_test.go
    - internal/app/hlc_prop_test.go

key-decisions:
  - "initialC parameter over a dumpMax+1 convention: the seam expresses an exact restart (constructor state equals the dump watermark); dumpMax+1 stays unenforceable at the NewClock level and discards the counter — the WR-01 mechanism choice the plan locked"
  - "Tick and Update stayed byte-identical: the seeded state routes the first Tick through the existing increment, counterMax-overflow and wall-ahead paths, so the 02-02 Kulkarni proofs remain valid"
  - "Task 2 is test-only: the behavior shipped with Task 1's seam, so the dominance proofs are regression locks whose failing mode was demonstrated by mutation (seed removed), mirroring the 02-02 TDD note"

patterns-established:
  - "Restart-dominance rows park the wall clock exactly at the watermark millisecond via fixedNow and carry the store's own NodeID in the watermark version — the adversarial same-ms restart"

requirements-completed: [VERS-02, VERS-04, TEST-02]

coverage:
  - id: D1
    description: "Counter-seeded restart seam: NewClock initialC masked to the 12-bit domain and seeded into the clock state; Config.InitialC placed after InitialL and propagated by NewStore at the single composition point; NewClock and Config doc contracts rewritten to demand the dump's full watermark and state that initialL alone is insufficient at counter >= 1 on the watermark millisecond"
    requirement: VERS-04
    verification:
      - kind: unit
        ref: "internal/app/clock_test.go#TestClock_restartFromDumpMax"
        status: pass
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_restartDominance"
        status: pass
    human_judgment: false
  - id: D2
    description: "Strict dominance over a full-watermark dump under adversarial clock input: 100-tick walk where every packed emission Compare-returns newer against every dump record including the (dumpMax, watermarkC 1..4095, testNodeID) watermark, l never drops below dumpMax, readings strictly increase"
    requirement: TEST-02
    verification:
      - kind: unit
        ref: "internal/app/hlc_prop_test.go#TestClock_restartProperty"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both WR-01 consequences dead at store level: variant A — a peer holding the dump watermark accepts the first post-restart write (exactly one Conflict with Winner the fresh version, Get returns fresh data); variant B — the stale watermark record displaces with exactly one Conflict and Get keeps the fresh data (no equal-version no-op)"
    requirement: VERS-04
    verification:
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_restartDominance"
        status: pass
    human_judgment: false
  - id: D4
    description: "Quality gates green at plan close with the shipped graph untouched: full race suite, strict lint with zero suppressions, byte-identical go.mod/go.sum, internal/port and the vendored lint config unmodified"
    verification:
      - kind: other
        ref: "go test -race -count=1 ./... and go tool golangci-lint run ./... — both exit 0"
        status: pass
      - kind: other
        ref: "git diff --quiet go.mod go.sum — exit 0"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-11-02
status: complete
commits: 3
plan_head_before: bedbd97ccdfdc5e510ce834ded83de1c74de241e
plan_head_after: b4b5397daa492e04977c81c33831ef1113bd2ea8
---

# Phase 02 Plan 04: Restart-Seam Dominance (Gap Closure G-02-VERS04-DOMINANCE) Summary

**Counter-seeded restart seam (NewClock initialC + Config.InitialC) making every post-restart version strictly dominate the dump's full watermark — the two falsified 02-02 truths now property-proven over a (dumpMax, counter >= 1, ownNodeID) watermark, with both WR-01 probe failures (silent revert, silent drop) encoded as deterministic regression rows.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-11-02T02:29:21Z
- **Completed:** 2026-11-02T02:38:03Z
- **Tasks:** 2
- **Files modified:** 5 (2 production, 3 test; no new files)

## Accomplishments

- NewClock extended to `NewClock(now func() time.Time, initialL int64, initialC uint16, driftWarn time.Duration) *Clock` — the constructor masks initialC with counterMask (total over any uint16 input, mirroring Update's remote-counter masking) and seeds the clock state; Tick and Update bodies are byte-identical, so the seeded first Tick routes through the existing `(initialL, initialC+1)` / counterMax-overflow / wall-ahead paths
- Config gained `InitialC uint16` directly after InitialL; NewStore propagates it at the single composition point (store.go's NewClock call). Both doc contracts now demand the dump's FULL watermark and state explicitly that seeding initialL alone is insufficient whenever any dump version carries counter >= 1 at the watermark millisecond
- Six new unit rows: (T, initialC+1) first and second tick, the counterMax 4095 bump to (T+1, 0), wall-ahead adoption of (pt, 0), out-of-range 4101 masked to 5, and the two store rows stamping (T, 6, N) / (T+1, 0, N) strictly past a seeded watermark
- TestClock_restartProperty supersedes the counter-0 narrowing: the dump's watermark record is now (dumpMax, watermarkC 1..4095 from the seeded rng, testNodeID) — counter >= 1 AND the store's own NodeID, the exact collision hazard — and every packed emission across the 100-tick adversarial walk beats every dump record
- Both WR-01 consequences are dead and regression-locked: a peer holding the dump watermark accepts the first post-restart write (variant A), and merging the stale watermark record back displaces with exactly one Conflict while Get keeps the fresh data (variant B)
- Binding proven by mutation: with the seed stripped, the property fails at tick 1 (`(dumpMax, 1)` vs watermark `(dumpMax, 257)` in the demo run) and both regressions reproduce the verifier's probe failures verbatim

## Task Commits

Each task was committed atomically (TDD: RED then GREEN per task):

1. **Task 1 RED: failing restart-dominance rows** — `a5a78a5` (test)
2. **Task 1 GREEN: counter-seeded seam** — `68e4b62` (feat)
3. **Task 2: full-watermark property + WR-01 regressions** — `b4b5397` (test)

**Plan metadata:** (final docs commit below)

Note on TDD: Task 1 followed the full RED/GREEN cycle — the RED commit is the six behavior rows, which cannot compile against the three-parameter NewClock and the InitialC-less Config (the plan's explicitly defined failing state; a signature change admits no compiling RED in Go without pre-implementing). Task 2 changed no production code — the behavior shipped with Task 1's seam — so it is a single test commit whose failing mode was demonstrated by the mutation run, mirroring the 02-02 TDD note.

## Files Created/Modified

- `internal/app/clock.go` — NewClock gains initialC (masked, seeded); doc contract rewritten to the full-watermark restart semantics; Tick/Update untouched
- `internal/app/store.go` — Config.InitialC field + full-watermark doc; NewStore passes cfg.InitialC to NewClock at the single composition point
- `internal/app/clock_test.go` — TestClock_restartFromDumpMax extended with the four seed subtests; 6 existing NewClock call sites moved to the new arity
- `internal/app/store_test.go` — new TestStore_restartDominance (Config propagation rows + variant A/B regressions); watermarkC fixture constant
- `internal/app/hlc_prop_test.go` — TestClock_restartProperty rewritten to the (dumpMax, watermarkC, testNodeID) watermark model; 7 NewClock call sites moved to the new arity

## Decisions Made

- Mechanism: initialC, not dumpMax+1 — recorded as the plan's locked choice; the seam now expresses an exact restart, enforceable at the NewClock level, and Phase 3 can wire dump loading onto it directly
- The seeded constructor state makes the first Tick's dominance a theorem of the unchanged Tick algorithm — no algorithm change, so the 02-02 Kulkarni property proofs carry over unmodified
- watermarkC drawn from 1..4095 in the property (never 0): counter 0 at the watermark is the already-covered cold case; >= 1 is the hazardous family the verifier falsified

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- One strict-gate lint finding on the new code (noinlineerr on the seed-Merge inline error check in variant A) — fixed in the code direction with a plain assignment before commit; zero nolint introduced
- The plan's read_first cites TestMerge_losers in merge_test.go; it lives in store_test.go with the conflictsEqual helper — both files were read; the advisory had already flagged this

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The restart seam Phase 3 wires dump loading onto (NewStore's Config{InitialL, InitialC}) now preserves VERS-04 strict dominance by construction: pass the dump's max l and the max counter at that l, and no post-restart write can collide with or subordinate to any dumped version
- The two 02-VERIFICATION failed truths are now backed by named tests (TestClock_restartProperty watermark case; TestStore_restartDominance variants B and A) — the phase's gap G-02-VERS04-DOMINANCE is closed and 02-VERIFICATION can be re-run
- go.mod/go.sum byte-identical, zero lint suppressions in internal/app, internal/port untouched (D-03/D-04 hold)

## Self-Check: PASSED

All five modified files plus this SUMMARY verified on disk; all three task commits (a5a78a5, 68e4b62, b4b5397) verified as ancestors of HEAD; full gates re-run green at plan close (race suite `-count=1`, strict lint, go.mod/go.sum byte-identical); seam grep gates re-confirmed (initialC uint16 = 1, counterMask = 3, InitialC in store.go = 5, watermarkC in hlc_prop_test.go = 5, TestStore_restartDominance = 1).

---
*Phase: 02-engine-hlc-versioning-and-lww-store*
*Completed: 2026-11-02*
