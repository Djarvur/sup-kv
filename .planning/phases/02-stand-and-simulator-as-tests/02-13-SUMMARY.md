---
phase: 02-stand-and-simulator-as-tests
plan: 13
subsystem: testing
tags: [test-01, test-09, mutation-testing, go-overlay, chaos, convergence-bounds, long-partition, combined-failures, red-runner, d-02, d-06]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: "plan 02-06's stabilized chaos corpus/poll-to-supremum semantics (the mutant substrate) and plan 02-12's complete red-runner expectation table (the stage the mutation probes append to)"
provides:
  - "TestScenario_combinedFailures: three nodes under a directed loss window, duplication, a timed partition (past peer_timeout) and an in-partition ReplaceNode restart, asserted against boundPartitionHeal + a documented margin with the D-06 no-fresh-loss qualifier"
  - "TestScenario_longPartition: named peerTableSurvives / peerEvicted subtests straddling the REPL-06 at-or-after eviction boundary; evicted convergence is not before the first post-heal sync and not later than the documented bound"
  - "scnConvergeBy + scnAssertRecordsPresent helpers: the bound and the no-fresh-loss qualifier as executable, replayable assertions"
  - "scripts/red-runner D-02 mutation stage: two overlay mutants (sync without tombstones, periodic sync disabled) killed on verified corpus seeds with a passing control, compile/sed/timeout/clean-tree guards, and verdicts in the red evidence"
  - "recorded full red run red-evidence/20261121T134010Z: 7/7 expectation rows red, 4 controls pass, 4 mutant kills"
affects: [02-11 (the scheduled job runs the full script with the mutant stage), phase 3-5 (the stabilized chaos semantics are now proven non-diluted)]

# Actuals (#2632) — same estimateTokens scale as the plan estimate: chars/4 over the realized diff.
actuals:
  tokens: 43798
  tasks: 2
  commits: 2

plan_head_before: 2335cad5ec7ba7841a059ce7c1542d6280850966
plan_head_after: 893043ce032e2e1799ee4d7bb9050679ed8c2c2c

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "go test -overlay mutation probes: temp-dir sed copies + an absolute-path overlay JSON; the production tree is never written, and the runner proves it with git status"
    - "Mutant verdict inversion: nonzero go test under the mutant is the expected kill; the runner rejects a passing mutant, a failing control, a sed no-op, a mutated copy that does not compile, and a timeout"
    - "Two-sided convergence bound assertion: divergence teeth at the heal instant, a not-before-first-post-heal-sync lower bound, and a not-later-than-boundPartitionHeal upper bound"
    - "No-fresh-loss qualifier made structural: all fault windows and writes end at the heal, so every pre-heal record is asserted present byte-equal on every node"

key-files:
  created:
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T134010Z/go-test.json
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T134010Z/summary.txt
  modified:
    - internal/app/repl/sim_scenario_test.go
    - scripts/red-runner

key-decisions:
  - "The long-partition boundary pair is peer_timeout ± one hello round, not exactly vs +margin: the REPL-06 eviction rule is at-or-after lastHello+peer_timeout (TestPeerTable_lifecycle probe #18), so a partition of exactly peer_timeout evicts at its heal instant; the not-yet-evicted side must end one hello round before the deadline, the evicted side one hello round past it"
  - "The long-partition clock is pinned by advancing to the 2s Hello delivery (2s+one delay) before starting the split, so every pair's lastHello — and therefore the eviction deadline — is known exactly instead of inferred from the hello phase"
  - "The periodic-sync mutant advances nextSync and then returns on the timer path; a bare return leaves nextSync in the past, nextDeadline returns a past instant, and the simulation driver livelocks instead of failing the test"
  - "Mutant seeds are per-mutant corpus lists: an exhaustive 16-seed scan showed 'sync without tombstones' is killed by exactly one corpus seed (a node must never have held the key, otherwise the newer-flooding reply heals it) while 'periodic sync disabled' is killed by four; every listed seed is a verified kill, so the runner's 'every seed under the mutant must fail' inversion stays truthful"
  - "Combined-failure loss uses the TestScenario_loss deterministic directed time-window form rather than a static DropProb, so every fault window and every write ends at the heal — the D-06 no-fresh-loss qualifier is structural, and only duplication (which cannot lose a record) continues past it"
  - "Pre-fault writes are quiesced before the restart, so the restarted node's record must already live on both peers; the targeted listing then restores it, which is exactly the no-fresh-loss property under a wiped disk"
  - "The mutant stage adds a compile guard (go test -c with the overlay) and rejects timeouts: a sed edit that breaks compilation or livelocks would otherwise masquerade as a successful mutant kill"

patterns-established:
  - "Mutation sensitivity as a runner stage: overlay-only edits, verified corpus seeds, a control arm, and machine-readable verdicts appended to the run's evidence"
  - "Bound tests carry both sides: divergence teeth at the heal instant plus a lower bound (not before the first post-heal sync) and an upper bound (boundPartitionHeal)"
  - "Eviction-boundary probes are expressed in hello rounds because the returning peer's first unblocked Hello is the quantum of the liveness clock"

requirements-completed: [TEST-01, TEST-09]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "TestScenario_combinedFailures: loss + duplication + timed partition + ReplaceNode restart combined, converging within boundPartitionHeal(keys)+margin with divergence teeth and every pre-heal record present"
    requirement: "TEST-09"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_(combinedFailures|longPartition)$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestScenario_longPartition: peerTableSurvives and peerEvicted named subtests straddling the at-or-after eviction boundary, with the not-before-first-post-heal-sync lower bound and the boundPartitionHeal upper bound"
    requirement: "TEST-09"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_(combinedFailures|longPartition)$' (also -count=50 across fresh seeds)"
        status: pass
    human_judgment: false
  - id: D3
    description: "scripts/red-runner D-02 mutation stage: both overlay mutants killed on verified fixed-corpus seeds, control passes, sed/compile/timeout/clean-tree guards, --skip-mutants for local iteration only"
    requirement: "TEST-01"
    verification:
      - kind: integration
        ref: "bash scripts/red-runner (full run, exit 0)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Recorded red run evidence with the mutation verdicts appended: 7/7 expectation rows red, 4 control passes, 4 mutant kills"
    requirement: "TEST-01"
    verification:
      - kind: other
        ref: ".planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T134010Z/summary.txt"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 13: Mutation Sensitivity and TEST-09 Bounds Summary

**The stabilized chaos semantics are proven non-diluted: both overlay mutants kill `TestChaos_convergence` on verified fixed-corpus seeds while the control passes, and the combined-failure and long-partition scenarios pin both sides of the documented convergence bounds with the no-fresh-loss qualifier.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-11-21T13:03:00Z
- **Completed:** 2026-11-21T13:42:53Z
- **Tasks:** 2
- **Files modified:** 4 (2 source/script, 2 committed evidence files)

## Accomplishments

- **TEST-09 combined failures:** `TestScenario_combinedFailures` runs three nodes under a directed loss window, duplication, an 8 s partition (past peer_timeout) and a `ReplaceNode` restart inside the partition. Convergence is asserted no later than `boundPartitionHeal(8 keys) + 5 s`, every pre-heal record is asserted present byte-equal on every node (the D-06 no-fresh-loss qualifier), and the heal instant shows divergence.
- **TEST-09 long partition:** `TestScenario_longPartition` names two subtests on the REPL-06 eviction boundary. `peerEvicted` asserts convergence is not before the first post-heal sync (with a teeth probe one nanosecond before the 30 s tick) and not later than `boundPartitionHeal(6 keys)`; `peerTableSurvives` asserts the surviving peer tables heal through the ladder alone, before the sync round. Both assert divergence at the heal instant and every pre-partition/partition record present.
- **TEST-01 mutation sensitivity:** `scripts/red-runner` gained a D-02 stage that sed-edits temp-dir copies of `internal/app/repl/core.go`, wires them through `go test -overlay` keyed to the absolute repo path, and requires both mutants — "sync without tombstones" (listing drops `Deleted` records) and "periodic sync disabled" (timer path advances `nextSync` then returns before emitting) — to fail `TestChaos_convergence` on verified corpus seeds. The un-mutated control passes on the union.
- **Guards and evidence:** corpus-seed membership, sed no-op (`cmp`), mutated-copy compile check, timeout rejection, clean-tree check before/after, and a `--skip-mutants` local-only flag. Verdicts are appended to the run's `summary.txt`; the recorded run `red-evidence/20261121T134010Z` is green (7/7 expectation rows red, 4 control passes, 4 mutant kills).
- **Verification breadth:** the two new scenario tests passed 50 consecutive fresh-seed runs under `-race`, the full `./internal/app/repl/` suite is green, `make lint` is clean, and `make test` is green.

## Task Commits

Each task was committed atomically:

1. **Task 1: TEST-09 combined-failure and long-partition scenario tests** - `c18190e` (test)
2. **Task 2: red-runner D-02 overlay mutation stage (+ recorded red evidence)** - `893043c` (test)

**Plan metadata:** pending docs commit (SUMMARY/STATE/ROADMAP/REQUIREMENTS)

## Files Created/Modified

- `internal/app/repl/sim_scenario_test.go` — `TestScenario_combinedFailures`, `TestScenario_longPartition` (with `peerTableSurvives` / `peerEvicted`), and the `scnConvergeBy` / `scnAssertRecordsPresent` / `scnWritten` / `scnHelloInterval` helpers; package doc extended for plan 02-13
- `scripts/red-runner` — D-02 mutation stage: mutant definitions and sed expressions, per-mutant corpus seed lists, `control_probe`, `mutant_probe`, `run_mutation_stage`, `--skip-mutants`, guards, and evidence append
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T134010Z/go-test.json` — raw `go test -json` stream of the full recorded run
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T134010Z/summary.txt` — parsed expectation table plus the mutation verdicts

## Decisions Made

- **Eviction boundary pair in hello rounds:** the plan's flagged assumption expected a partition of exactly `peer_timeout` to leave the peer unevicted; the core's at-or-after rule (REPL-06, probe #18) evicts at the deadline itself. The subtests therefore straddle the boundary at `peer_timeout ± one hello round`, which keeps both sides of the boundary executable and honest.
- **Pin the last Hello explicitly:** the test advances to the 2 s Hello delivery (`2 s + one-way delay`) before starting the split, so `lastHello + peer_timeout` is a known instant rather than a phase guess.
- **Advance-then-return mutant:** "periodic sync disabled" advances `nextSync` in the guarded return; a bare return leaves the driver deadline in the past and livelocks the scheduler. The runner additionally rejects `panic: test timed out` as a non-kill.
- **Per-mutant seed lists:** the exhaustive scan is the source of truth — one corpus kill for "sync without tombstones", four for "periodic sync disabled" (three used). Listing a non-killing seed would make the runner's inversion fail a correct implementation.
- **Structural no-fresh-loss:** both scenarios finish all writes at the heal and use deterministic windows (loss/partition) so no post-heal loss can invalidate the bound; duplication is the only fault allowed to continue, and it cannot lose a record.
- **Committed evidence:** the recorded run is committed append-only, following the plan 02-12 precedent (T-2-24).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Mutant stage rejects build failures and timeouts**
- **Found during:** Task 2 (mutation stage)
- **Issue:** A sed expression that breaks compilation or makes the test hang would also produce a nonzero `go test` exit, which the naive inversion would accept as a successful mutant kill — a false proof of mutation sensitivity.
- **Fix:** The stage compiles the mutated test binary first (`go test -overlay=... -c -o ...`) and exits 1 with the mutant name on failure; it also treats `panic: test timed out` as an error rather than a kill.
- **Files modified:** `scripts/red-runner`
- **Verification:** the guard cases were exercised directly (no-op sed, syntax-broken sed, passing mutant, failing control) — each exits 1 with the mutant name; a normal run reports all kills.
- **Committed in:** `893043c` (Task 2 commit)

**2. [Rule 1 - Bug / plan assumption correction] Eviction boundary subtests use peer_timeout ± one hello round**
- **Found during:** Task 1 (long-partition scenario)
- **Issue:** The plan's flagged assumption says "a peer at exactly the timeout is not yet evicted; one step past it is". The core's documented and tested rule is at-or-after: the first `Advance` with `now >= lastHello+peer_timeout` evicts (`TestPeerTable_lifecycle`, probe #18). A partition of exactly `peer_timeout` therefore evicts at its heal instant, leaving no executable "not yet evicted" side at the exact boundary.
- **Fix:** The named subtests straddle the boundary at `peer_timeout − one hello round` (tables survive; ladder heals before the sync) and `peer_timeout + one hello round` (evicted; the first post-heal sync heals). The comment documents the at-or-after rule and why the exact instant is evicted.
- **Files modified:** `internal/app/repl/sim_scenario_test.go`
- **Verification:** both subtests pass 50 fresh-seed runs under `-race`; the evicted subtest's not-before-sync probe and lower bound both fire.
- **Committed in:** `c18190e` (Task 1 commit)

**3. [Rule 2 - Missing Critical / plan adaptation] Per-mutant corpus seed lists (1 + 3 instead of 2-3 each)**
- **Found during:** Task 2 (mutation stage)
- **Issue:** The action asks for 2-3 fixed corpus seeds per mutant with every seed failing under that mutant. An exhaustive scan of all 16 committed corpus seeds found "sync without tombstones" killed by exactly one seed (848612892191557850) and "periodic sync disabled" by four. Asserting a second seed for the tombstone mutant would fail on a correct implementation; asserting fewer seeds would weaken the proof.
- **Fix:** The runner carries verified per-mutant kill lists (one seed for the tombstone mutant, three for the periodic-sync mutant) and the control covers the union. The rationale is documented next to the lists.
- **Files modified:** `scripts/red-runner`
- **Verification:** the full recorded run kills all four probes and passes all four controls.
- **Committed in:** `893043c` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 missing critical, 1 plan-assumption correction)
**Impact on plan:** All are inside the plan's two file subjects plus committed run evidence. The mutation proof and both TEST-09 bounds are stronger than the literal plan text in the places where the code's documented behavior differs from the plan's assumed behavior; no production code changed.

## Issues Encountered

- **First "periodic sync disabled" mutation livelocked the simulator.** Rewriting the timer gate to `if !nextSync.IsZero() { return }` never advanced `nextSync`, so `nextDeadline` returned a past instant and `RunUntil` spun forever; the first probe hit the 10-minute `go test` timeout instead of failing. Fixed by advancing `nextSync` before returning and by making the runner treat timeouts as errors.
- **The tombstone mutant is inherently narrowly sensitive.** Retransmission plus `handleDataSync`'s newer-flooding reply heals most tombstone omissions; only a node that never held the key diverges. The exhaustive scan bounded this empirically instead of guessing.
- **A concurrent external test battery was running during the recorded red run.** The expectation stage still asserted 7/7 red rows with 0 errors; the stand red verdicts are failure-shape assertions, not measurements that a load spike could flip to green.

## User Setup Required

None — the mutation stage needs only the Go toolchain and `python3` already required by the runner.

## Known Stubs

None. The mutant stage operates on real source copies; the scenario tests run the production `repl.Core`/`storage.Store` through `simnet`.

## Threat Flags

None beyond the plan's register. The sed/overlay trust boundary (T-2-26) is mitigated exactly as planned: temp-dir copies only, mutated-copy-differs assertion, compile guard, control arm, and a post-stage `git status` proof that `internal/app/repl/core.go` is untouched.

## Next Phase Readiness

- **Plan 02-11 (scheduled machinery):** the runner's full path now includes the mutant stage; the scheduled job must keep running `bash scripts/red-runner` without `--skip-mutants` so the probes stay live.
- **Phases 3-5:** the expectation-table flips remain the only red→green change; this plan proves the stabilized chaos test still bites, so the tabulated fixes can proceed against a trusted gate.
- **Requirement bookkeeping:** TEST-09 is complete; TEST-01's mutation-sensitivity half lands here, while its 10 000-seed "0 fails" statement is the scheduled job's evidence surface (plan 02-11's D-09 machinery).

## Self-Check: PASSED

- [x] `internal/app/repl/sim_scenario_test.go` and `scripts/red-runner` exist
- [x] Task commits `c18190e` and `893043c` are ancestors of HEAD
- [x] Recorded red evidence `red-evidence/20261121T134010Z/summary.txt` exists and carries the mutant verdicts
- [x] `go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_(combinedFailures|longPartition)$'` passes
- [x] `st=$(git status --porcelain -- internal/app/repl/core.go)` is empty (tree-clean)
- [x] `check evaluation-scope --plan 02-13` resolves 2 commits and both plan files, no missing/outside files
