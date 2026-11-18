---
phase: 02-engine-hlc-versioning-and-lww-store
plan: 03
subsystem: testing
tags: [go, crdt, lww-element-map, property-testing, math-rand-v2-pcg, merge-laws, tombstone, race-detector, concurrency-testing, stdlib-only, replayable-seeds]

requires:
  - phase: 02-engine-hlc-versioning-and-lww-store
    plan: "01"
    provides: Store with Merge-installs-externally-versioned-records, Conflict return contract (D-01), tombstones kept in the map
  - phase: 02-engine-hlc-versioning-and-lww-store
    plan: "02"
    provides: seeded generators newRNG/randRecord/randStore/randKey/randVersion with the printed-seed and -seed replay contract (D-04)
provides:
  - TEST-01 merge-law proof suite — commutativity (state plus key-sorted conflict sets), associativity, idempotence over 500 seeded triples each for mixed and tombstone-heavy stores
  - Exact loser-accounting and duplicate-delivery no-op properties (CORE-06, CORE-07)
  - Race-detector concurrency proofs — full-API hammer, exactly-one-winner SetIfAbsent, greater-version-wins Delete-vs-Set over 50 duels
  - Full phase gate evidence — build, vet, race suite at 100 percent statement coverage, strict lint, go.mod/go.sum and internal/port unchanged
affects: [03-persistence, 04-replication, 06-hardening]

actuals:
  tokens: 4138
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - order-insensitive conflict-set comparison (sort by key, compare element-wise) so map-iteration order cannot flake law tests
    - goroutine-safe fake now via atomic.Int64 millisecond counter (atomicNow) — the D-02 seam under concurrency
    - per-goroutine deterministic generators rand.New(rand.NewPCG(seed, workerIndex)) inside one hammer

key-files:
  created:
    - internal/app/merge_test.go
  modified:
    - internal/app/store_test.go

key-decisions:
  - "Zero store.go fixes: the plan 02-01 Merge held every law — commutativity, associativity, idempotence over tombstones, exact loser pairing — the fix site was never triggered"
  - "atomicNow builds time values with time.UnixMilli, applying the locked plan 02-02 correction to this plan's own time.Unix(ms, 0) prose (Pitfall 8: time.Unix reads its first argument as seconds)"
  - "The hammer pre-seeds every key with Set before launching goroutines so the every-key-present coherence assertion is guaranteed reachable — Delete and SetIfAbsent never create, so a Get/Delete-only key could otherwise be legitimately absent"
  - "Duplicate delivery is tested as install-then-retransmit on an empty key: a re-delivered LOSER record legitimately re-reports its Conflict (CORE-07 counts rejected versions), so Pitfall 3's strict no-op holds exactly for the equal-version case"
  - "noinlineerr findings (5 sites) fixed in the code direction with plain err assignments — zero nolint in internal/app maintained"

patterns-established:
  - "Law helper assertMergeLaws(t, step, a, b, c): shared by TestMerge_laws and TestMerge_lawsTombstones so both mixes prove identical laws"
  - "Duel tests capture both writers' returned versions and assert the survivor equals the greater — outcome-independent assertions over racy interleavings"

requirements-completed: [TEST-01, CORE-06, CORE-07]

coverage:
  - id: D1
    description: "Merge laws property-proven over 500 seeded triples (mixed and tombstone-heavy stores): commutativity with identical state and key-sorted conflict sets in both directions, associativity either grouping, idempotence with zero conflicts"
    requirement: TEST-01
    verification:
      - kind: unit
        ref: "internal/app/merge_test.go#TestMerge_laws"
        status: pass
      - kind: unit
        ref: "internal/app/merge_test.go#TestMerge_lawsTombstones"
        status: pass
    human_judgment: false
  - id: D2
    description: "Loser accounting exact and duplicates silent: differing versions yield exactly one Conflict (winner = greater full 16-byte version, loser = lesser, never equal-on-equal); a retransmitted record returns nil conflicts and byte-identical Records()"
    requirement: CORE-07
    verification:
      - kind: unit
        ref: "internal/app/merge_test.go#TestMerge_losersProperty"
        status: pass
      - kind: unit
        ref: "internal/app/merge_test.go#TestMerge_duplicateDelivery"
        status: pass
    human_judgment: false
  - id: D3
    description: "Concurrency proofs under the race detector: 8x200-op full-API hammer with atomic fake now and per-goroutine PCG (zero findings, final Get/Records coherence), exactly one SetIfAbsent winner among 16 racers, Delete-vs-Set survivor equals the greater captured version over 50 duels"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_concurrentHammer"
        status: pass
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_concurrentSetIfAbsent"
        status: pass
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_concurrentDeleteVsSet"
        status: pass
    human_judgment: false
  - id: D4
    description: "Full phase gate green with the structural boundaries intact: build, vet, race suite at 100% statement coverage of internal/app, lint config verify plus strict run, go.mod/go.sum unchanged (D-04), internal/port doc.go only (D-03)"
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... && go test -race ./... — exit 0"
        status: pass
      - kind: other
        ref: "go tool golangci-lint config verify && go tool golangci-lint run ./... — exit 0, zero issues"
        status: pass
      - kind: other
        ref: "git diff --quiet go.mod go.sum && test \"$(ls internal/port)\" = doc.go — exit 0"
        status: pass
    human_judgment: false

duration: 14min
completed: 2026-11-02
status: complete
commits: 2
plan_head_before: 573abd9c0c794793da7e34c4d848656b11dfa900
plan_head_after: 8f3b8c2c4b560470ddde445853146b49733e8022
---

# Phase 02 Plan 03: HLC Versioning and LWW Store Summary

**Clock-free property suite proving the CRDT join laws (commutativity with identical conflict sets, associativity, idempotence, tombstone-inclusive, exact loser accounting, silent duplicate delivery) plus a zero-finding race-detector hammer over the whole exported API — the plan 02-01 Merge passed every law with zero production fixes, closing the phase at 100 percent statement coverage with go.mod/go.sum and internal/port untouched.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-11-02T01:19:36Z
- **Completed:** 2026-11-02T01:33:11Z
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 appended)

## Accomplishments

- The merge laws hold over 500 seeded random store triples for both record mixes — live/tombstoned (TestMerge_laws) and tombstone-heavy (TestMerge_lawsTombstones): merge(a,b) and merge(b,a) reach identical per-key state AND identical key-sorted conflict sets, merge(merge(a,b),c) equals merge(a,merge(b,c)), and merge(a,a) is a strict no-op — delivery order, duplicates and retransmissions cannot affect the outcome (TEST-01, CORE-06)
- The no-resurrection rule proven property-style: an older live record merged after a newer tombstone keeps the tombstone and reports the live record as the Conflict loser (CORE-06)
- Exact loser accounting in both directions over 500 seeded pairs — exactly one Conflict with Winner the greater full 16-byte version and Loser the lesser, zero conflicts on equal versions, never a Conflict pairing equal versions; a retransmitted record is a strict no-op with byte-identical Records() (CORE-07, Pitfall 3)
- Race-detector proofs: an 8-goroutine x 200-op hammer over Set/SetIfAbsent/Delete/Merge/Get with an atomic fake now and per-goroutine deterministic PCG generators completes with zero findings and final Get/Records() coherence; 16 racing SetIfAbsent callers yield exactly one winner whose payload is stored; 50 Delete-vs-Set duels leave exactly the greater captured version
- Full phase gate green exactly as CI runs it: go build, go vet, go test -race (100 percent statement coverage of internal/app), golangci-lint config verify plus strict run with zero issues and zero nolint — go.mod/go.sum byte-identical (D-04) and internal/port still holding only doc.go (D-03)

## Task Commits

Each task was committed atomically:

1. **Task 1: TEST-01 merge-law property suite** — `37e88c1` (test)
2. **Task 2: Concurrency hammer + full phase gate** — `8f3b8c2` (test)

**Plan metadata:** (final docs commit below)

Note on TDD: Task 1 carries tdd="true" but the behavior under test was
implemented by plan 02-01; the tests are the deliverable. Every law passed on
the first full run — the plan's "fix belongs in store.go's Merge" site was
never triggered, so no GREEN/feat commit exists, matching the plan's stated
expectation and the plan 02-02 precedent.

## Files Created/Modified

- `internal/app/merge_test.go` — TestMerge_laws, TestMerge_lawsTombstones, TestMerge_duplicateDelivery, TestMerge_losersProperty; local helpers cloneStore, mergeAll, recordsEqual, sortedConflicts, assertMergeLaws, randTombStore
- `internal/app/store_test.go` — appended TestStore_concurrentHammer, TestStore_concurrentSetIfAbsent, TestStore_concurrentDeleteVsSet and the atomicNow helper (imports grew by fmt, math/rand/v2, sync, sync/atomic)

## Decisions Made

- Zero store.go fixes — the 02-01 join implementation satisfied every law and the exact loser-pairing contract on all 1000 seeded triples plus 500 loser pairs; the plan's code-direction fix site stayed unused
- atomicNow uses time.UnixMilli(ms.Add(1)): this plan's prose repeated the "time.Unix(ms, 0)" seconds/ms trap that plan 02-02 already diagnosed (Pitfall 8), so the locked correction was applied rather than the literal prose
- The hammer pre-seeds every key with a Set so the "every key present in Records()" coherence assertion is meaningful: Delete and SetIfAbsent never create records, so a key touched only by Get/Delete could otherwise be legitimately absent — presence then proves no concurrent path drops a key
- TestMerge_duplicateDelivery delivers into an empty key (install, then retransmit): a re-delivered LOSER record legitimately re-reports its Conflict — CORE-07 counts every rejected version — so the strict no-op contract holds exactly for the equal-version case Pitfall 3 describes
- noinlineerr findings (5 inline error checks) fixed with plain err assignments in the code direction; zero nolint directives remain in internal/app

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Five noinlineerr lint findings on first lint run (inline `if _, _, err := ...; err != nil` checks) — fixed in the code direction with plain assignments, same strict-gate class plan 02-02 met
- An initial replay-check command failed with "flag provided but not defined: -v" — my own flag ordering (flags after -args reach the test binary); re-running with the correct order replayed both law suites deterministically at -seed=12345

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 2 is complete: all three plans landed on feat/02-engine-hlc-lww-store with the engine's ROADMAP success criteria demonstrably true — single-node KV semantics (02-01 unit tables), merge laws property-proven (this plan), HLC monotonic + byte-order + uniqueness property-proven (02-02)
- Phase 3 can serialize Record and rely on Records() as the dump source; Phase 4 can journal Merge's returned Conflicts as the conflict-journal source knowing every displaced or rejected version appears exactly once and duplicates are silent
- go.mod/go.sum byte-identical to the phase start; internal/port untouched (D-03, D-04 hold)

## Self-Check: PASSED

Both task files verified on disk; both task commits (37e88c1, 8f3b8c2) verified as ancestors of HEAD; full gates (build, vet, race suite with 100.0% coverage of internal/app, lint config verify + run, go.mod/go.sum diff, internal/port listing) re-run green at plan close.

---
*Phase: 02-engine-hlc-versioning-and-lww-store*
*Completed: 2026-11-02*
