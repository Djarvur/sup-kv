---
phase: 04-protocol
plan: 03
subsystem: replication-protocol
tags: [proto-01, proto-02, proto-04, hlc-absorption, per-part-max, o1-key-count, content-generation, statestore-ripple, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 01
    provides: "the DataSync part handler, the completeness accumulation and the validation pipeline whose position the absorption respects"
  - phase: 03-transport
    provides: "the consumer-side StateStore seam, the store clock and the engine pass-through style"
provides:
  - "storage.Store.Observe: version-only HLC absorption with Merge's drift signal"
  - "storage.Store.Len and persist.Engine.KeyCount: an O(1), tombstone-inclusive, clone-free record count"
  - "storage.Store.Generation and persist.Engine.Generation: a content generation bumped only inside install"
  - "repl.StateStore.Observe/Generation and the per-part maximum-version absorption in handleDataSync"
affects:
  - "04-04 consumes KeyCount for the startup empty-vs-data gate"
  - "04-08 records Generation for the cached listing build's staleness check"
  - "04-06 wires the engine pass-throughs into the adapter; every StateStore implementer now carries the two new methods"

actuals:
  tokens: 12027
  tasks: 2
  commits: 5
  plan_head_before: 6877ac4830b68545e70b64d9597bafac423ad78f
  plan_head_after: 4e7061c5bab944643f789085280409c64f03fe11

tech-stack:
  added: []
  patterns:
    - "A version-only clock seam (Observe) takes no store lock: it touches no record, so the clock's own mutex is the only synchronization — the same seam Merge's trailing Update uses."
    - "A content generation bumps at the single convergence point (install), so local writes, merge winners and restore replay move it exactly once and every no-op or read path moves it not at all."
    - "A validated-datagram side effect (HLC absorption) sits in the handler after the validation gate and independent of the later state machine, so a discarded datagram and a still-partial round are both handled correctly."

key-files:
  created:
    - .planning/phases/04-protocol/red-evidence/04-03-task1.json
    - .planning/phases/04-protocol/red-evidence/04-03-task2.json
  modified:
    - internal/app/storage/store.go
    - internal/app/storage/store_test.go
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - internal/app/repl/core.go
    - internal/app/repl/listedkeys_internal_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/egress_internal_test.go
    - internal/app/internal/simnet/simnet.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/repl_internal_test.go
    - internal/udprepl/smoke_test.go
    - internal/udprepl/syncsize_internal_test.go
    - internal/grpcapi/wait_integration_test.go

key-decisions:
  - "Observe takes no store lock: it reads and writes no record, so the clock's own mutex is the only synchronization. Merge's trailing Update is the same seam; a store lock would serialize absorption against writers for no benefit."
  - "Generation bumps once inside install, the single convergence point of local writes, merge winners and restore replay. Bumping in each write path would need six places and would still miss a future change source; install cannot."
  - "The absorption call sits immediately after handleDataSync's header check and before accumulateSync. On the wired path the validation pipeline has already rejected bad headers and out-of-bound versions, so a discarded datagram never reaches the clock, while a still-partial round already advances the HLC (PROTO-01)."
  - "The fake that holds its own map (egressStore) returns a clock-less no-op Observe and a fixed 0 Generation: nothing caches a listing build over it. Every real-store wrapper delegates both, including the simulator's nodeStore."
  - "The drift return is unused in the core this phase (OBSV-05 owns drift semantics); the far-ahead drift row follows the existing absorb-then-warn clock contract pinned by TestMergeStats_counts, not a new threshold."
  - "Task 2's RED commit carries the interface and all implementer delegations: a Go package that does not compile is a build_failure, not valid RED evidence. The behavior-adding half — the per-part Observe call — was absent until GREEN."

patterns-established:
  - "Absorption at the pipeline position: the clock call belongs after validation and before the state machine, so hostile input cannot advance the HLC and partial rounds still contribute."
  - "Generation at the convergence point: a content counter lives inside install, making 'moves only on a real change' structural rather than a rule each write path must remember."

requirements-completed: [PROTO-01, PROTO-02]

coverage:
  - id: D1
    description: "Version-only absorption into the store clock: Observe feeds clock.Update, returns Merge's drift signal, and creates no record; Engine delegates it."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/storage/store_test.go#TestStore_observeAdvancesClock"
        status: pass
      - kind: unit
        ref: "internal/app/storage/store_test.go#TestStore_observeMonotone"
        status: pass
      - kind: unit
        ref: "internal/app/storage/store_test.go#TestStore_observeReturnsDrift"
        status: pass
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_keyCountAndObserveGeneration"
        status: pass
    human_judgment: false
  - id: D2
    description: "O(1) record count: Store.Len is one RLock plus len(records) — tombstones counted, no Data cloned — and Engine.KeyCount delegates without consulting dump state."
    requirement: PROTO-02
    verification:
      - kind: unit
        ref: "internal/app/storage/store_test.go#TestStore_len"
        status: pass
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_keyCountAndObserveGeneration"
        status: pass
    human_judgment: false
  - id: D3
    description: "Content generation: Store.Generation moves by exactly one per install and never for a rejected write, a no-op Delete, an equal-or-older Merge, Observe or any read; Engine and the core's StateStore expose it."
    requirement: PROTO-04
    verification:
      - kind: unit
        ref: "internal/app/storage/store_test.go#TestStore_generationContentOnly"
        status: pass
      - kind: unit
        ref: "internal/app/repl/listedkeys_internal_test.go#TestSync_generationPassThrough"
        status: pass
    human_judgment: false
  - id: D4
    description: "Per-part HLC absorption: every non-empty validated listing part absorbs its exact maximum entry version once; an empty part and a validation-discarded part absorb nothing."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/listedkeys_internal_test.go#TestSync_observePerPart"
        status: pass
      - kind: unit
        ref: "internal/app/repl/listedkeys_internal_test.go#TestSync_observeOncePerPart"
        status: pass
      - kind: unit
        ref: "internal/app/repl/listedkeys_internal_test.go#TestSync_observePerPartEmptyAndDiscarded"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every StateStore implementer satisfies the new methods: the engine, the simulator node store, the seven pre-existing fakes and 04-01's white-box fake; the module builds, vets, lints and the quick package set stays green."
    requirement: PROTO-01
    verification:
      - kind: other
        ref: "go build ./... && go vet ./..."
        status: pass
      - kind: other
        ref: "go test -race -count=1 ./internal/app/storage/ ./internal/app/persist/ ./internal/app/repl/ ./internal/udprepl/ ./internal/grpcapi/ ./internal/app/internal/simnet/"
        status: pass
      - kind: other
        ref: "make lint (config verify + golangci-lint run ./...)"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 03: Listing Parts Feed the HLC and the Store Exposes Count and Generation

**Every non-empty validated listing part now absorbs its exact maximum entry version into the store clock exactly once, while empty and discarded parts absorb nothing — and the store gained an O(1) tombstone-inclusive record count and a content generation that moves only when a stored record actually changed.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-11-22T03:45:43Z
- **Completed:** 2026-11-22T03:55:30Z
- **Tasks:** 2
- **Files modified:** 16 (14 declared code/test files plus 2 RED evidence records)

## Accomplishments

- PROTO-01's absorption half is in place: `handleDataSync` computes the exact maximum of a part's validated entry versions with the full 16-byte comparison and calls `StateStore.Observe` once, before accumulation and independent of stream completion. The next local write strictly beats every listed version, proven end to end through a real `storage.Store`.
- The pipeline position is the contract, and it is tested: a datagram discarded for an out-of-bound version performs zero clock calls, and an empty part performs zero. A hostile peer cannot advance the HLC before validation (T-4-10).
- `Store.Len` is one `RLock` plus `len(records)` — tombstones included, no `Data` cloned — and `Engine.KeyCount` delegates it without consulting dump state, so 04-04's startup empty-vs-data gate can decide without materializing the store (PROTO-02, T-4-11).
- `Store.Generation` bumps exactly once inside `install`: local `Set`/`SetIfAbsent`/`Delete`, winning merges and restore replay move it; a rejected `SetIfAbsent`, a missing-key `Delete`, an equal-or-older `Merge`, `Observe` and every read leave it untouched (T-4-43). A listing build can cache the value it was built at (04-08).
- The interface ripple is complete: the engine, the simulator's node store, the seven pre-existing test fakes and 04-01's white-box fake all satisfy `Observe` and `Generation`; `go build ./...`, `go vet ./...`, the six-package race set and `make lint` are green.

## Task Commits

Each task was committed atomically (TDD RED → GREEN where production changed):

1. **Task 1 RED: failing absorption/count/generation tests** - `453fa17` (test)
2. **Task 1 GREEN: store clock absorption, Len and Generation** - `e7350ce` (feat)
3. **Task 2 RED: failing per-part absorption tests + interface ripple** - `d0b1eae` (test)
4. **Task 2 GREEN: per-part max-version absorption** - `83c68ff` (feat)
5. **Lint cleanup in the new test rows** - `4e7061c` (refactor)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `internal/app/storage/store.go` — `Observe` (unpack → `clock.Update` → drift), `Len` (O(1) count), `Generation` and the `generation` field bumped in `install`
- `internal/app/storage/store_test.go` — five new rows: absorption advance, monotone older observation, drift return, O(1) tombstone-inclusive count, content-only generation
- `internal/app/persist/engine.go` — `Observe`, `KeyCount` and `Generation` pass-throughs with their caller contracts
- `internal/app/persist/engine_test.go` — `TestEngine_keyCountAndObserveGeneration`: empty-engine 0, confirmed `Set` 1, reads unchanged, `Observe` clock-only
- `internal/app/repl/core.go` — `StateStore.Observe`/`Generation` with contracts; per-part `Observe(maxEntryVersion(entries))` in `handleDataSync`; `maxEntryVersion` helper
- `internal/app/repl/listedkeys_internal_test.go` — the counting `observingStore` fake and the four new rows (per-part, empty/discarded, once-per-part max, generation pass-through)
- `internal/app/repl/retransmit_test.go`, `egress_internal_test.go` — `stateStore` delegates both; the map-holding `egressStore` returns a clock-less no-op and a fixed 0
- `internal/app/internal/simnet/simnet.go` — `nodeStore` delegates both, so every simulated node absorbs listing versions
- `internal/udprepl/repl_test.go`, `repl_internal_test.go`, `smoke_test.go`, `syncsize_internal_test.go`, `internal/grpcapi/wait_integration_test.go` — the five remaining fakes delegate both
- `.planning/phases/04-protocol/red-evidence/04-03-task{1,2}.json` — persisted RED evidence records

## Decisions Made

See frontmatter `key-decisions` — the substantive ones: no store lock in `Observe`; generation bumped only inside `install`; absorption placed after validation and before accumulation; map-holding fakes return a fixed generation; the drift return unused this phase; and the RED commits carrying the compile ripple because an uncompilable package is not valid RED.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The strict lint gate rejected the first spelling of the new test rows**
- **Found during:** Plan-level verification (`make lint`)
- **Issue:** The vendored strict config's `noinlineerr` rejects the inline `if _, _, err := ...; err != nil` guards I used in `store_test.go`, `engine_test.go` and `listedkeys_internal_test.go`, and `revive`'s `unused-receiver` rejects the map fake's `Observe`/`Generation` receivers that ignore their store.
- **Fix:** Replaced every inline error guard with a plain assignment plus check, and changed the `egressStore` receivers to `*egressStore` with no name (the receiver is deliberately unused).
- **Files modified:** `internal/app/storage/store_test.go`, `internal/app/persist/engine_test.go`, `internal/app/repl/listedkeys_internal_test.go`, `internal/app/repl/egress_internal_test.go`
- **Verification:** `make lint` (config verify + `golangci-lint run ./...`) green; the target tests still pass under `-race`.
- **Committed in:** `4e7061c`

---

**Total deviations:** 1 auto-fixed (Rule 3)
**Impact on plan:** Style-only; no assertion, fixture, bound or production behavior changed, and no scope was added.

## TDD Gate Compliance

Both tasks followed RED → GREEN with persisted evidence (`gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for each record):

- **Task 1:** RED `453fa17` — `04-03-task1.json` records six failing target tests led by `TestStore_observeAdvancesClock` (the next `Set` was older than the absorbed version); the only passing row was the preserved-behavior in-tolerance-zero drift subtest. GREEN `e7350ce` made all seven rows pass.
- **Task 2:** RED `d0b1eae` — `04-03-task2.json` records `TestSync_observePerPart` and `TestSync_observeOncePerPart` failing with zero `Observe` calls. The empty/discarded and generation pass-through rows passed in RED by construction: they pin the absence of a call and the compile ripple respectively. GREEN `83c68ff` added the per-part call and made all rows pass.
- **REFACTOR:** `4e7061c` is lint-style cleanup only; every test stayed green. No production refactor was warranted: the implementations are the minimal bodies the plan specifies.

## Issues Encountered

None beyond the lint deviation. The pre-existing `make test` cmd/sup-kv UDP pick-to-bind flake was not exercised: the plan's verification set and the lint gate ran green on the final tree, and the full-repo gate remains 04-13's.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. No placeholder values, TODOs or unwired data sources were introduced. The `Observe` drift return is deliberately unused in the core this phase (OBSV-05 owns drift semantics), which is the plan's scoping, not a stub.

## Threat Flags

None — no new trust-boundary surface. T-4-10 (validated-only clock absorption), T-4-11 (O(1) clone-free count) and T-4-43 (content-gated generation) are mitigated as planned.

## Next Phase Readiness

Ready for 04-04. `Engine.KeyCount` is the startup empty-vs-data selection's input, and the store's generation is available for 04-08's cached listing build; all implementers carry the two new interface methods. Ten plans remain in the phase.

---
*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Created files exist: this SUMMARY, both RED evidence records, and every claimed modified file.
- Commits exist and are ancestors of HEAD: `453fa17`, `e7350ce`, `d0b1eae`, `83c68ff`, `4e7061c`.
- The measured plan commit count is 5 from the ledger base `6877ac48`, matching the frontmatter `commits: 5`.
- No tracked file from the plan's diff was deleted.
