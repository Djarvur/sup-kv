---
phase: 08-performance
plan: 03
subsystem: replication-determinism
tags: [performance, determinism, synctest, chunking, perf-01, perf-06]
requires:
  - phase: 08-performance
    provides: the wave-1 threshold table (08-01) and the per-key pending index (08-02) landed before this plan
provides:
  - a sleep-free dumpNotify scenario harness sequenced by testing/synctest, one bubble per scenario run
  - chunker capacity and no-alias assertions that fail against both WR-06 regressions (budget-sized hint, cur[:0] reuse)
affects: [PERF-01, PERF-06, internal/app/repl, 08-11]
actuals:
  tokens: 4244
  tasks: 2
  commits: 2
plan_head_before: 6d5eed23b0f03122986f199d2bdaf55078ef245c
plan_head_after: 5513fcd0f4db09aade0c15a48778e280fc589670
tech-stack:
  added: []
  patterns:
    - "Scenario sequencing: one testing/synctest bubble per scenario run created in the subtest body, synctest.Wait as the only idle barrier over the virtual scheduler"
    - "Slice-sizing teeth: capacity bounded by the entries remaining at chunk creation plus a backing-array identity check, with the teeth proved by temporary mutation"
key-files:
  created: []
  modified:
    - internal/app/repl/sim_dump_test.go
    - internal/app/repl/config_internal_test.go
key-decisions:
  - "Each scenario-run subtest owns its bubble and asserts on the captured simDumpRun after the bubble returns, so every assertion runs with the engine goroutine already stopped by the bubble's cleanup."
  - "The same-seed subtest runs runDumpConfirm in two separate bubbles: one bubble per run, because the first engine's cleanup happens at bubble exit."
  - "The load-only shrunk-limit subtest stays outside a bubble: it starts no goroutine and is not a scenario run, so the must-have's one-bubble-per-scenario-run rule does not apply to it."
  - "writeAndConfirm and notify use non-blocking reads after synctest.Wait instead of wall-clock timeouts; a missing result fails by name against the engine, never by a silence."
  - "Task 2's test passes on arrival because the WR-06 production fix predates this plan (commit 7ba917e); its teeth are proved by two temporary mutations of core.go reverted byte-identically (md5 unchanged), not by a committed failing product state."
patterns-established:
  - "A real engine goroutine over the simulator's virtual scheduler is sequenced by synctest.Wait, never by sleeps, quiet-round counters or activity counters."
  - "A capacity contract is asserted through cap() and pointer identity from the in-package test, so a budget-sized pre-allocation or slice reuse fails by name."
requirements-completed: [PERF-01, PERF-06]
coverage:
  - id: D1
    description: "The dumpNotify scenario harness contains no wall-clock synchronization: one testing/synctest bubble per scenario run, synctest.Wait as the only idle barrier, and 100 repetitions under -race pass without a stall"
    requirement: PERF-01
    verification:
      - kind: unit
        ref: "go test -race -count=100 -run '^TestScenario_dumpNotify$' ./internal/app/repl/ (PASS in 1.416s)"
        status: pass
      - kind: command
        ref: "grep gate over sim_dump_test.go for time.Sleep/runtime.Gosched/simSettlePause/simQuietRounds/simSettleWallCap/simActivity prints count=0"
        status: pass
    human_judgment: false
  - id: D2
    description: "The chunker test pins cap(chunks[0]) == len(entries) for the 3-entry maximal-budget case, bounds every multi-chunk capacity by the entries remaining, and rejects adjacent chunks sharing a backing array; the existing shape/order assertions are unchanged"
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestChunkSyncEntries_capacityFollowsEntries$' -v (PASS, three subtests)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestChunkSyncEntries|TestSync_' (PASS in 2.262s)"
        status: pass
      - kind: command
        ref: "mutation proof: budget-only hint reports cap 2729 vs 3; cur[:0] reuse reports chunks 0 and 1 sharing a backing array; both RED evidence records RED_EVIDENCE_OK, core.go restored to md5 fdcdfbabf2d14b4b131cbbea44d0c086"
        status: pass
    human_judgment: false
duration: 7min
completed: 2026-11-24
status: complete
---

# Phase 08 Plan 03: Deterministic dumpNotify Harness and Chunker Teeth Summary

**The dumpNotify scenario is now sequenced by `testing/synctest`'s idle barrier instead of 1.2 ms sleep-and-count windows, and the chunker's WR-06 sizing claim is executable down to `cap()` and backing-array identity.**

## Performance

- **Duration:** 7min
- **Started:** 2026-11-24T18:53:06Z
- **Completed:** 2026-11-24T19:00:18Z
- **Tasks:** 2
- **Files modified:** 2 (0 created, 2 modified)

## Accomplishments

- `sim_dump_test.go` lost every wall-clock synchronization mechanism: `simSettlePause`, `simQuietRounds`, `simSettleWallCap`, the `simActivity` counter, `settle`, `awaitActivity` and the `virtualAfter` activity bump are gone. `virtualAfter` is now a plain `sched.ScheduleAfter` adapter.
- `synctest.Wait()` is the only idle barrier. `advanceTo` is `sched.RunUntil(target)` followed by `Wait`; `writeAndConfirm` waits for the engine's initial idle block, starts `Set`, waits for both the Set goroutine and the engine to block, fires the debounce instant, waits again, then requires the Set result in a non-blocking read that names the engine on failure. `stop` cancels, waits, then non-blocking reads the buffered `done` and asserts `context.Canceled`.
- Each scenario run creates exactly one bubble in its subtest body, with the assertions executed on the captured `simDumpRun` after the bubble returns. The same-seed subtest runs the confirm scenario in two separate bubbles, one per engine goroutine. All five `t.Parallel()` calls in `TestScenario_dumpNotify` and its subtests are gone; `TestScenario_tiebreakConverges` is untouched.
- Every original claim is still asserted: the first attempt at the debounce instant, the confirmation instant, announcement-after-confirmation ordering, the `ErrDumpFailed` outcome, the 0/100/300/700 ms ladder with the recovering fourth attempt, peer convergence on both nodes, the shrunk-limit skip report, and byte-identical same-seed runs.
- `-count=100` under `-race` passes (`ok ... 1.416s`, wall time 1.7 s); the no-sleep grep gate reports 0.
- `config_internal_test.go`'s `TestChunkSyncEntries_capacityFollowsEntries` now asserts `cap(chunks[0]) == len(entries)` for the 3-entry maximal-budget case, bounds every chunk's capacity by the entries remaining at its creation in a two-entry-budget six-entry listing, rejects adjacent chunks sharing a backing array, and keeps the existing shape/order and empty-listing claims.
- The chunker test's teeth were demonstrated: a temporary budget-only `hint` reports `cap(chunks[0]) = 2729, want 3`; a temporary `cur[:0]` reuse reports `chunks 0 and 1 share a backing array`. Both mutation runs validated as `RED_EVIDENCE_OK` (`target_test_failed`).
- `make lint` and `make test` (`go test -race -cover ./...`) are green across the repository; `internal/app/repl` coverage is 93.2%.

## Task Commits

1. **Task 1: Put the dumpNotify harness on testing/synctest** - `95788e3` (test)
2. **Task 2: Give the chunk capacity claim teeth** - `5513fcd` (test)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/app/repl/sim_dump_test.go` — synctest conversion: bubble creation in each scenario-run subtest, `advanceTo`/`writeAndConfirm`/`notify`/`stop` on `synctest.Wait`, the plain `virtualAfter` adapter, and removal of the sleep/quiet/activity harness (56 insertions, 98 deletions).
- `internal/app/repl/config_internal_test.go` — capacity equality, per-chunk capacity bounds and backing-array identity assertions, plus the empty-listing case; the existing shape/order assertions stay (86 insertions, 17 deletions).

## Decisions Made

- **One bubble per scenario run, assertions after it.** Each `runDumpConfirm`/`runDumpFailure` call happens inside its own `synctest.Test` bubble in the subtest body; the `simDumpRun` value is asserted outside, where cleanup has already stopped the engine.
- **Two bubbles for the same-seed proof.** The engine goroutine's cleanup runs at bubble exit, so two runs in one bubble would leave the first engine alive.
- **The load-only subtest is not a scenario run.** The shrunk-limit subtest uses `persist.Open` synchronously and starts no goroutine; the must-have's "each scenario run gets its own bubble" rule does not cover it.
- **No wall-clock fallback survives.** The post-barrier receives are non-blocking and fail by name; the only timeouts left are the engine's virtual-time timers on the simulator scheduler.
- **The chunker budget is derived, not retyped.** The multi-chunk budget is `chunkOverheadBytes + entrySize + entrySize`, and the per-chunk count is derived from the budget (`(budget - chunkOverheadBytes) / entrySize`), so the test states the counting rule rather than magic numbers.

## Deviations from Plan

### Auto-fixed Issues

None. No Rule 1-4 deviation was needed: no product bug surfaced, `core.go` stayed untouched, and every plan verification passed on the first green run.

### Documented TDD adaptation

**1. Task 2's RED is a mutation run, not a committed failing test**
- **Found during:** Task 2 RED.
- **Issue:** Task 2 is `tdd="true"`, but its only deliverable file is the test, and the product implementation it pins is already correct (WR-06 was fixed in `7ba917e`). The plan forbids changing `core.go`, so no committed RED/GREEN transition exists.
- **Fix:** Proved the teeth with two temporary working-tree mutations of `core.go` — the budget-only `hint` and `cur[:0]` reuse — each run captured as RED evidence and each reverted with `git checkout -- internal/app/repl/core.go`; the file's md5 returned to the pre-plan `fdcdfbabf2d14b4b131cbbea44d0c086` and `git status` shows it unmodified. Both records are `RED_EVIDENCE_OK` (`target_test_failed`). The GREEN state is the committed test at `5513fcd` passing against the real chunker.
- **Files modified:** none beyond `config_internal_test.go`; `core.go` byte-identical.
- **Verification:** `.planning/phases/08-performance/red-evidence/20261124T185646Z/` holds both records, reports and exit files (untracked, matching the phase's convention).

**Total deviations:** 0 auto-fixed; one documented TDD adaptation (mutation-based RED, no product change).
**Impact on plan:** No scope change and no bound relaxed; the mutation runs are exactly the plan's stated teeth proof ("reverting the hint ... makes the test fail").

## TDD Gate Compliance

| Gate | Commit | Result |
|------|--------|--------|
| RED (Task 2) | mutation run (working tree only, not committed) | `RED_EVIDENCE_OK` on both records (`target_test_failed`): `cap(chunks[0]) = 2729, want 3`; `chunks 0 and 1 share a backing array` |
| GREEN (Task 2) | `5513fcd` test(08-03): pin the chunker slice sizing and no-alias rule | Capacity, remaining-entries bounds and no-alias assertions pass; sibling chunker suite green |

Task 1 is not a TDD task (harness conversion): its gate is the `-count=100 -race` run plus the no-sleep grep, both green at `95788e3`.

## Issues Encountered

- The first mutation attempt changed both the initial `hint` and the per-chunk `make`, which left the loop index `i` unused and produced a build failure instead of an assertion failure. The evidence run was redone with the single-line initial-hint mutation so the target test failed on its capacity assertion (`2729`), and a separate `cur[:0]` mutation exercised the alias assertion.
- The plan's assumption held: the defect was in the harness, not the engine. `-count=100` under `-race` shows no stall, so there is no product finding to report.

No blocking issues.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. The converted harness has no placeholder or skipped scenario; every original assertion is executed inside the new bubbles and the chunker cases are real assertions against the production chunker.

## Next Phase Readiness

Ready for 08-07 and later. The dumpNotify scenario can no longer flake on host load, and the chunker's capacity and no-alias contract is pinned. PERF-01's executable half and PERF-06's deterministic-suite contribution from this plan are in place; the remaining PERF-01 documentation work stays with the plans that own it.

## Self-Check: PASSED

- Files: `internal/app/repl/sim_dump_test.go`, `internal/app/repl/config_internal_test.go`, `.planning/phases/08-performance/08-03-SUMMARY.md` all found.
- Commits: `95788e3` and `5513fcd` both ancestors of HEAD.
- `core.go` md5 `fdcdfbabf2d14b4b131cbbea44d0c086` matches the pre-plan value and `git status` shows no change.
