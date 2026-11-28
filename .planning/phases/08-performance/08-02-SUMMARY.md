---
phase: 08-performance
plan: 02
subsystem: decision-core
tags: [performance, pending-index, xport-02, perf-05, tdd, invariants, differential]
requires:
  - phase: 03-transport
    provides: the pending set, the per-peer ledger, the retry heap and the XPORT-12 results stream
  - phase: 02-stand-and-simulator-as-tests
    provides: the deterministic simulator and the frozen red-seed harness
  - phase: 07-observability-and-configuration
    provides: the stabilized observability seams that consume the unchanged results
provides:
  - the per-key pending index (pendingByKey) with exactly two maintenance sites
  - key-local trackedVersion, removeSupersededLocal and applyRemote displacement
  - the deterministic PendingVisits counter seam
  - the visit gates, the index invariant, the fixed-seed differential and the production-path proof
affects: [08-04, 08-05, 08-06, 08-10]
actuals:
  tokens: 10543
  tasks: 3
  commits: 5
plan_head_before: 1c3fdff664c4b7d42629337cbb13830498284be2
plan_head_after: 7609b93ed64d490bd28f504263367a1b952cdd07
tech-stack:
  added: []
  patterns:
    - single-slot per-key index over a structurally single-valued map
    - deterministic entry-inspection counter as the load-independent flatness seam
    - index-vs-naive-scan differential over a fixed-seed operation script
key-files:
  created:
    - internal/app/repl/pending_index_internal_test.go
  modified:
    - internal/app/repl/core.go
key-decisions:
  - "pendingByKey is a single-slot map[string]pendingKey, not a set-valued map: the one-pair-per-key invariant is total, so a second pair under a key is a bug the differential names rather than a state the index accommodates."
  - "PendingVisits increments once per pending entry INSPECTED, including the key-mismatch scans of the pre-index code, which is what makes the 2N/N RED counts meaningful; after the index a miss inspects zero and a hit one."
  - "RED evidence is the plan's own behavior-5 split (write 2N and apply N red, idle-Advance tripwire green): the invariant/differential tests were authored in GREEN because referencing pendingByKey in RED would have made the package a build_failure, which #3770 refuses as INVALID_RED."
  - "PERF-05 is advanced but not closed: this plan lands the core fix and the deterministic proofs; 08-04/08-05 still own the docs and the benchmark gates, so REQUIREMENTS.md is deliberately not marked and requirements ready-ids still reports PERF-05 blocked."
patterns-established:
  - "Index maintenance contracts live on the field (invariant, exactly-two sites, warning that new insert/delete paths must use them) and the counter contract lives on the counter (five inspected loops, new loops must increment it)."
  - "A structural index is proven three ways: exact visit counts at two set sizes, a differential against the naive scan after every scripted operation, and the exact results sequence on the production driver paths."
requirements-completed: [PERF-05]
coverage:
  - id: D1
    description: "pendingByKey plus the key-local trackedVersion, removeSupersededLocal and applyRemote displacement, with PendingVisits and its accessor"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestCore_pendingVisits' (fresh-key write and apply report exactly 0 visits at 10 and 10 000 pending; the idle-Advance tripwire reports 0; the structure pin reports 2/3/4)"
        status: pass
      - kind: command
        ref: "grep -nE 'pendingByKey\\[|delete\\(c\\.pendingByKey|pendingByKey:' internal/app/repl/core.go — assignments only in New, trackLocalWrite's insert and removePending's guarded delete"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ (whole package, unchanged supersession/ledger/Ack/eviction/snapshot/scenario tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The index invariant and the fixed-seed differential (index vs naive scan, ledger vs wait-sets, unacked gauge) after every operation"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestCore_pendingIndexInvariant|TestCore_pendingIndexDifferential' (2 000 scripted ops, T-08-04 stale-delete guard exercised)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The production-path proof: exact Superseded/Drained/WaitSetEmpty/Ack sequence through HandleDatagram/Advance, invariant after every step"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestCore_pendingIndexProductionPaths'"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./internal/grpcapi/ (T-08-06: the adapter seams observe no change)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The maintenance-contract comments and a confined change surface (index/counter/docs only) with the benchmark fixture no longer O(n^2)"
    requirement: PERF-05
    verification:
      - kind: command
        ref: "cumulative non-comment diff from 1c3fdff: two field declarations, the New initializer, the three rewritten loops, the two remaining-loop increments, the accessor and the guarded delete — nothing else"
        status: pass
      - kind: command
        ref: "make lint (whole repo) and make test (whole repo, -race -cover)"
        status: pass
      - kind: command
        ref: "go test -run '^$' -bench '^BenchmarkCoreEventLoop$' -benchtime 10x ./internal/app/repl/ — 3.24s before, 0.86s after (the 10 000-entry fixture seeding is no longer O(n^2)); ns/op and unacked still reported"
        status: pass
    human_judgment: false
duration: 16min
completed: 2026-11-23
status: complete
---

# Phase 08 Plan 02: Per-Key Pending Index Summary

**The pending set now carries a per-key index: a fresh-key local write or inbound replication inspects zero pending entries at any set size, with the one-pair-per-key invariant, the ledger and every XPORT-12 emission proven unchanged.**

## Performance

- **Duration:** 16min (954s)
- **Started:** 2026-11-23T18:01:41Z
- **Completed:** 2026-11-23T18:17:35Z
- **Tasks:** 3 (Task 1 TDD: RED + GREEN; Tasks 2-3 standard)
- **Files modified:** 2 (1 created, 1 modified)

The deterministic evidence at both acceptance sizes:

| Operation (over N pending entries) | N=10, pre-index | N=10 000, pre-index | N=10 and N=10 000, post-index |
|---|---|---|---|
| fresh-key local write | 20 (2N) | 20 000 (2N) | **0** |
| fresh-key inbound replication | 10 (N) | 10 000 (N) | **0** |
| idle `Advance` (tripwire) | 0 | 0 | **0** |
| repeated-key local write | 2 | 2 | 2 |
| due retransmission (3 due items) | 3 | 3 | 3 |
| discovery flush (4 pending pairs) | 4 | 4 | 4 |

The fixture seeding itself was the hidden quadratic: the RED run needed 1.67 s per 10 000-entry subtest; after the index the same seed costs 0.02 s. `BenchmarkCoreEventLoop` wall time fell from 3.24 s to 0.86 s at `-benchtime 10x` (ns/op at that benchtime is noise-dominated on this host; repeated 100x runs put both sizes in the same 240-820 ns/op band).

## Accomplishments

- `Core.pendingByKey map[string]pendingKey` is the per-key index into `pending`. It is maintained in **exactly two places**: `trackLocalWrite` inserts beside `c.pending[pk] = entry`, and `removePending` deletes behind `if cur, indexed := ...; indexed && cur == pk`, so a stale call can never evict a newer pair (T-08-04). `New` initializes it with the map literal.
- `trackedVersion` and `removeSupersededLocal` read only the key's index slot: a miss inspects zero entries, a hit one. `applyRemote`'s displacement block resolves `msg.Key`'s indexed pair and keeps the identical `cmp > 0 || (cmp == 0 && incomingWon)` rule, the `Conflicts` increment, the `Superseded` emission with the TRACKED version and `removePending`.
- `Core.PendingVisits() uint64` counts pending-entry inspections at the five whole-set loops — `trackedVersion`, `removeSupersededLocal`, the `applyRemote` displacement, `scanRetransmit`'s due loop and `handleHello`'s discovery flush. It is driver-goroutine-only, has no adapter mirror and is deliberately not a metrics family; the field comment makes a new pending-set loop's obligation explicit so the flatness gate stays a complete measure.
- `pending_index_internal_test.go` proves the change four independent ways: the five exact visit gates (`10` vs `10 000`), the structural pin, the full invariant helper (index size/correspondence, ledger ↔ wait-set equality in both directions, `UnackedStats` count), a 2 000-operation fixed-seed differential comparing `trackedVersion` against a naive scan after every op, and the production-path script driving write, discovery, supersession, duplicate/older notifications, Ack drain, remote displacement, peer-timeout eviction, suspension, re-learn and the stale-record removal through `HandleDatagram`/`Advance` with the exact result sequence asserted per step.
- `handleHello`'s discovery flush and `sortedPendingKeys` are untouched in order and behavior: the flush still emits every pending pair once to a newly seen peer, re-arms its ladder and never extends the wait-set or the ledger; it now prices itself in `PendingVisits`.
- The idle path was already flat; `TestCore_pendingVisitsIdleAdvance` is documented in place as a **tripwire**, never RED evidence.

## Task Commits

1. **Task 1 RED: failing pending-visit gates** — `d9a2bdd` (test)
2. **Task 1 GREEN: per-key pending index** — `f162bd5` (feat)
3. **Task 2: production-path + exact-sequence proof** — `f914bba` (test)
4. **Task 3: maintenance-contract documentation** — `462a8d6` (docs)
5. **Lint-clean refactor of the proofs** — `7609b93` (refactor)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/app/repl/core.go` — `pendingByKey` and `pendingVisits` fields, the `New` initializer, the insert/delete sites, the three key-local rewrites, the two remaining-loop increments, the `PendingVisits` accessor and the field/accessor/`PendingSnapshot`/flush contract comments.
- `internal/app/repl/pending_index_internal_test.go` — the in-package fixture, the five visit gates, the invariant helper, the naive-scan differential and the production-path proof.

## Decisions Made

- The index is a single-slot map keyed by string. The one-pair-per-key invariant is already total (guard + supersession-before-insert), so a set-valued index would pay allocation to tolerate a state that is a bug; the differential turns any hidden second pair into a named failure.
- `PendingVisits` counts **inspected entries**, including the pre-index loops' key-mismatch iterations. That is the only reading under which the RED evidence (2N/N) and the post-fix zero are the same measurement.
- PERF-05 is recorded on this plan's coverage but **not** marked in `REQUIREMENTS.md`: `requirements ready-ids` reports it blocked, because 08-04 (docs) and 08-05 (the paired wall-clock gates) still cover it.
- The plan's RED shape (invariant/differential tests authored alongside the visit gates) was split: those tests reference `pendingByKey`, so including them in RED would have been a package build failure. They landed in GREEN; the RED record is the plan's own behavior-5 split.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Process] RED evidence is the counter-instrumented split, not the plan's compile failure**
- **Found during:** Task 1 RED
- **Issue:** The task text says the test file fails to compile because `PendingVisits` does not exist. Under `tdd_mode`, `go test` reports that as `build_failure` and the classifier refuses it as INVALID_RED (#3770); a build failure cannot authorize GREEN. The plan's own behavior 5 already names the acceptable split: tests 1-2 red at 2N/N, test 3 green as the tripwire.
- **Fix:** The RED commit carries the compiling counter skeleton (field, accessor, five increments in the existing loops) plus tests 1-4. Tests 1-2 failed on their assertions with 20/20 000 and 10/10 000; test 3 passed as the documented tripwire; test 4 passed. The index and the invariant/differential tests landed in the GREEN commit.
- **Files modified:** `internal/app/repl/core.go`, `internal/app/repl/pending_index_internal_test.go`
- **Verification:** `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` (`target_test_failed`) for `TestCore_pendingVisitsFreshKeyWrite`.
- **Commit:** `d9a2bdd`

**2. [Rule 2 - Lint gate] The proof helpers were restructured to keep `make lint` green**
- **Found during:** Task 3 (lint gate)
- **Issue:** `unparam` flagged the always-constant peer parameter, `govet` flagged the shadowed `ok` in `removePending`, `noinlineerr` flagged three inline Merge error checks and `golines` flagged one long failure line.
- **Fix:** The fixture owns its peer identity (helpers take none), the index probe was renamed `indexed`, the Merge checks use plain assignments and the message was wrapped. No behavior changed.
- **Files modified:** `internal/app/repl/core.go`, `internal/app/repl/pending_index_internal_test.go`
- **Verification:** `make lint` exit 0 on the whole repo; the full `-race` suite still green.
- **Commit:** `462a8d6` (core rename), `7609b93` (test refactor)

## TDD Gate Compliance

| Gate | Commit | Result |
|------|--------|--------|
| RED (Task 1) | `d9a2bdd` test(08-02): add failing pending-visit gates | `RED_EVIDENCE_OK`; write 2N (20/20 000), apply N (10/10 000), idle tripwire 0 |
| GREEN (Task 1) | `f162bd5` feat(08-02): add the per-key pending index | All visit gates 0 at both sizes; invariant and differential green |
| Task 2 (not TDD) | `f914bba` test(08-02): prove the pending index on the production driver paths | Exact result sequence + invariant after all ten steps |
| Task 3 (not TDD) | `462a8d6` docs(08-02): state the pending index maintenance contract | `make lint`, `make test` green |

The RED evidence record and console report are under `.planning/phases/08-performance/red-evidence/20261123T180426Z/` (untracked, matching the phase's red-evidence convention).

## Issues Encountered

- None blocking. The two wall-clock ratio gates in `bench_test.go` were not edited and passed in this run's `make test`; they remain 08-05's to replace.
- `BenchmarkCoreEventLoop` at `-benchtime 10x` measures only ten iterations, so its ns/op swings between runs (before 653.5 → after 1467 on one sample; 100x repeats put both sizes in one band). The stable, meaningful number is the fixture wall time: 3.24 s → 0.86 s.
- The RED run's 10 000-entry seeding was itself O(n²) (1.67 s per failing subtest), which is the defect being fixed; the post-fix seed is 0.02 s.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. The index, the counter and all five proof gates are wired to production paths; no placeholder or TODO remains.

## Threat Flags

None new. The plan's register is mitigated as designed: T-08-04 by the guarded delete plus the differential's stale-delete coverage; T-08-05 by removing the per-write whole-set scan while leaving the retransmission ladder and dump confirmation untouched; T-08-06 by the exact result-sequence assertions and the repl/udprepl/grpcapi `-race` suites; T-08-SC by installing nothing (`internal/app` stays stdlib-only).

## Next Phase Readiness

- 09 Phase 8 plans remain (08-03 … 08-11); this plan's artifacts are ready for 08-04 (docs/measurement) and 08-05 (the `PendingVisits` gates replacing the two ratio gates in `bench_test.go`).
- `make lint` and `make test` are green on this tree, and the full `repl`, `udprepl` and `grpcapi` suites pass under `-race`.
- The maintenance contract is stated where a future editor will read it: adding a pending mutation without the two sites, or a pending-set loop without the counter, is now a contract violation flagged in the field comments.

## Self-Check: PASSED

- `internal/app/repl/core.go` — FOUND
- `internal/app/repl/pending_index_internal_test.go` — FOUND
- `.planning/phases/08-performance/08-02-SUMMARY.md` — FOUND
- Commits `d9a2bdd`, `f162bd5`, `f914bba`, `462a8d6`, `7609b93` — all ancestors of HEAD
