---
phase: 04-protocol
plan: 10
subsystem: replication-protocol
tags: [proto-09, proto-01, three-node, simulator, complete-listing, f14, decision-14.9, determinism, test-only, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 08
    provides: "the first-seen targeted listing (no sync_interval term) and the generation-fresh shared listing build the joiner and F14 scenarios exercise"
  - phase: 04-protocol
    plan: 09
    provides: "the peer identity ring and the Hello classification whose restart branch the F14 replacement rides"
  - phase: 04-protocol
    plan: 07
    provides: "the real-socket three-node wiring proof the 14.9 anchor names alongside the simulator evidence"
  - phase: 04-protocol
    plan: 05
    provides: "the identity-only egress queue and paced replay the join and heal paths consume"
  - phase: 04-protocol
    plan: 03
    provides: "StateStore.Observe/Generation and the per-part DataSync handler the completeness window observes"
  - phase: 04-protocol
    plan: 01
    provides: "the DataSync completeness header, per-(node,run,sync_id) accumulation and SyncArrived/ListingComplete contract"
  - phase: 03-transport
    provides: "the simulator harness (Scheduler, Network, Cluster, ReplaceNodeWithState, AddNode) and its TraceDigest observation seam"
provides:
  - "TestScenario_threeNode plus runThreeNodeScenario: N=3 origin convergence, a directed block applied/healed through a mutable partition predicate inside boundPartitionHeal, and same-seed trace-digest equality"
  - "TestScenario_joinerWaitsForCompleteListing: the partial-listing window observed one pacing slot at a time — SyncArrived/ListingComplete stay false across intermediate parts and flip only on the completing peer's final part"
  - "TestScenario_f14ReplacedNode: ReplaceNodeWithState onto a stale snapshot opens no gate on the seed, flips both completeness reads only after the complete listing, and preserves node A's newer value"
  - "the 14.9 decision-coverage anchor repointed to the N=3 simulator scenario, with the real-socket three-node test named as the wiring proof"
  - "a recorded audit that first-seen listing traffic moves no simnet extension bound or determinism expectation"
affects:
  - "04-11 and 04-12 cite the N=3 simulator scenario as PROTO-09's semantic evidence alongside the real-socket wiring proof"
  - "04-13's phase gate re-runs the scenario, decision and simnet suites this plan leaves green"
  - "the phase verifier reads TestScenario_threeNode's digest-equality assertion as the scenario-local determinism pin"

actuals:
  tokens: 5112
  tasks: 2
  commits: 2
  plan_head_before: cbb715e677bc7f52aa99ab6840ed95d3d10ca20e
  plan_head_after: 14b69580caade663982db7da5d535a57969a097b

tech-stack:
  added: []
  patterns:
    - "Scenario-local determinism: the same seeded scenario is factored into a runner returning (cluster, TraceDigest) and invoked twice; equality of the two digests is asserted in addition to the suite's replay pins, so extra listing traffic can never quietly become nondeterminism."
    - "Partial-listing observation through exported seams: listing parts are counted from Network.TraceDigest's DataSync deliveries to the joiner, and the completeness reads come from Core.SyncArrived/ListingComplete — no core-internal stream state is inspected."
    - "A mutable partition predicate captured by simFixedDelay is the only way to toggle a Network policy (Policy is fixed at construction); the scenario applies, observes and heals the block through that one closure."
    - "One-entry-per-chunk chunking is forced by a SyncChunkBytes budget below the per-entry estimate, making the multi-part listing round deterministic without touching production defaults."

key-files:
  created: []
  modified:
    - internal/app/repl/sim_scenario_test.go
    - internal/app/repl/decision_coverage_test.go
    - internal/app/internal/simnet/simnet_ext_test.go

key-decisions:
  - "runThreeNodeScenario threads *testing.T through the factored runner so the mid-partition teeth — node 3 must NOT have converged — are asserted where that state exists; the returned (*simnet.Cluster, string) contract is exactly as planned, and both same-seed invocations assert."
  - "The N=3 block is a bidirectional isolation of n3 (the mutable predicate's (from==n3) != (to==n3) form), held for one virtual second: long enough to observe divergence, far below peer_timeout so the peer table survives and the retransmission ladder heals it."
  - "The joiner scenario forces one entry per listing part with SyncChunkBytes=100 (below len(key)+24 plus the 96-byte chunk overhead), yielding four parts per peer; the flip step must show all four parts of the completing peer's round delivered, which is what proves the signal waited for the final part."
  - "Listing parts are counted from the delivery trace, not from a new exported counter: TraceDigest is the existing observation seam, and adding a test-only production counter would have been a production shortcut the plan's prohibitions forbid."
  - "The F14 listing bound reuses the empty-disk bound's arithmetic shape for a one-key store: one Hello one-way, one immediate listing chunk, one delivery one-way, plus four pacing slots of slack — an upper bound derived from the mechanism, not a widened range."
  - "The simnet extension audit changed comments only: the determinism gate compares two same-seed runs (there is no golden digest to move), and the receive budget is armed after discovery listings have drained, so no counted bound moved."
  - "Task 1 is test-only and its asserted behavior shipped in plans 04-01/04-03/04-08, so the target tests passed on their first run; no RED record was fabricated (the 04-07 precedent) and the unexpected GREEN is disclosed below."

patterns-established:
  - "Completeness-window scenarios step the scheduler at the listing-pacer cadence and read parts from the delivery trace, so every intermediate part is observable without racing wall clocks."
  - "Same-seed scenario determinism is asserted by re-running the factored body and comparing TraceDigest values, not by comparing only final stores."
  - "A stale preserved snapshot is proven incapable of opening the gate: the F14 scenario checks both completeness reads at the restart instant before any event runs."

requirements-completed: [PROTO-09, PROTO-01]

coverage:
  - id: D1
    description: "N=3 simulator convergence — a write originated on every node is visible on all three, and a directed block of node 3 applied and healed through a mutable predicate converges inside the documented partition-heal bound with no fresh loss; the same seeded scenario run twice yields identical trace digests."
    requirement: PROTO-09
    verification:
      - kind: unit
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_threeNode"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_threeNode$'"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_|^TestDecisions_'"
        status: pass
    human_judgment: false
  - id: D2
    description: "PROTO-01's completeness window: with one entry per listing part, an intermediate part arrives while the joiner's SyncArrived() is false and its store is incomplete, and the signal flips only when the completing peer's final part lands; ListingComplete moves with it."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_joinerWaitsForCompleteListing"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_joinerWaitsForCompleteListing$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "PROTO-01's F14 value-preservation consequence at simulator level: ReplaceNodeWithState onto a stale preserved snapshot leaves SyncArrived()/ListingComplete() false at the restart instant and flips them only after the complete listing, while both stores serve node A's newer value byte-equal — the stale seed never displaces it."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_f14ReplacedNode"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_f14ReplacedNode$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "Decision 14.9's evidence anchor names the N=3 simulator scenario, no longer the two-node chaos stand-in, and the index comment names the real-socket three-node test as the wiring proof; every decision row still resolves to an existing evidence test."
    requirement: PROTO-09
    verification:
      - kind: unit
        ref: "internal/app/repl/decision_coverage_test.go#TestDecisions_referencedByTests"
        status: pass
      - kind: command
        ref: "grep -c '\"14\\.9\".*\"TestScenario_threeNode\"' internal/app/repl/decision_coverage_test.go && ! grep -q '\"14\\.9\".*\"TestChaos_convergence\"' internal/app/repl/decision_coverage_test.go"
        status: pass
    human_judgment: false
  - id: D5
    description: "The simulator extension suite is audited and green: first-seen listing traffic moves no counted bound, and the determinism gate's same-seed equality contract is documented as the comparison (no golden digest exists to update)."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/internal/simnet/simnet_ext_test.go#TestSim_determinism_withExtensions"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/internal/simnet/"
        status: pass
    human_judgment: false
  - id: D6
    description: "No regression in the touched packages: the whole internal/app/repl package (all scenarios, chaos determinism replay, bound suite, decision meta-test) and the simnet harness stay green under the race detector, and the strict lint config passes for both packages."
    requirement: PROTO-09
    verification:
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestChaos_determinismReplay$'"
        status: pass
      - kind: command
        ref: "go tool golangci-lint run ./internal/app/repl/ ./internal/app/internal/simnet/"
        status: pass
    human_judgment: false

duration: 397s
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 10: N=3 Simulator Evidence, the Complete-Listing Window and the F14 Replacement Summary

**The deterministic simulator now proves N=3 convergence with a real applied-and-healed partition, watches a joiner stay closed across intermediate listing parts until the final part lands, and shows a stale preserved snapshot can never displace a peer's newer value or open a gate.**

## Performance

- **Duration:** 397s (~7 min)
- **Started:** 2026-11-22T06:04:56Z
- **Completed:** 2026-11-22T06:11:33Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- `TestScenario_threeNode` (with the `runThreeNodeScenario` body) proves the phase's N=3 semantics in virtual time: one distinct write per origin converges on all three nodes; a bidirectional block of n3 toggled through a mutable `simFixedDelay` predicate is applied, observed (the blocked node lacks the split write and its store differs from the kept-write model while nodes 1 and 2 hold it), healed, and converges inside `boundPartitionHeal` with every pre-heal record present byte-equal. The scenario runs twice on the same seed and asserts identical `TraceDigest()` values — scenario-local determinism on top of the suite replay pins.
- `TestScenario_joinerWaitsForCompleteListing` turns `SyncChunkBytes` below the per-entry estimate to force one entry per part (four parts per peer). Stepping the scheduler one listing-pacing slot at a time, it records a part arriving while `SyncArrived()` is false and the store is incomplete, then asserts the flip happens only when the completing peer's fourth part has landed, with `ListingComplete()` moving in the same event. A flip at the first observed part, a premature flip, or no flip all fail.
- `TestScenario_f14ReplacedNode` proves PROTO-01's value-preservation consequence at simulator level: node B restarts onto a stale preserved snapshot via `ReplaceNodeWithState`; both completeness reads are false at the restart instant (a seeded store is not a listing), they flip only after the peer's complete listing arrives, and both stores end on node A's newer value byte-equal — the stale seed never displaces it.
- The 14.9 decision-coverage row now names `TestScenario_threeNode` instead of the two-node chaos stand-in, and the index comment records that the real-socket three-node test is the wiring proof over the same contract. `TestDecisions_referencedByTests` stays green.
- The simnet extension suite was audited: the determinism gate compares two same-seed runs (no golden digest exists to update), the receive budget is armed after discovery listings drain, and no counted bound moved. The audit is recorded in the two affected comments.

## Task Commits

1. **Task 1: N=3, complete-listing and F14 simulator scenarios** - `6cd08ac` (test)
2. **Task 2: 14.9 anchor and simnet extension audit** - `14b6958` (test)

**Plan metadata:** this SUMMARY/STATE/ROADMAP/REQUIREMENTS commit follows this file.

## Files Created/Modified

- `internal/app/repl/sim_scenario_test.go` — added `scnDataSyncPartsBySource`, `runThreeNodeScenario`, `TestScenario_threeNode`, `TestScenario_joinerWaitsForCompleteListing` and `TestScenario_f14ReplacedNode` (~351 lines). `sim_bound_test.go` was audited and needed no edit: the bound suite is green unchanged because first-seen listings drain before every measurement starts.
- `internal/app/repl/decision_coverage_test.go` — 14.9 repointed to `TestScenario_threeNode`; the adjacent-evidence comment now covers 14.7 only and names the real-socket three-node test as 14.9's wiring proof.
- `internal/app/internal/simnet/simnet_ext_test.go` — comment-only audit records at `TestSim_recvBudget_refusesAndCounts` and `TestSim_determinism_withExtensions`; no counted bound or digest expectation moved.

## Decisions Made

- **The runner takes `t`.** `runThreeNodeScenario(seed)` as literally written could not assert the mid-partition teeth (the blocked node must not have converged) without either fabricating a return slot or panicking. Threading `*testing.T` keeps the planned `(*simnet.Cluster, string)` return contract and puts the assertion where the state exists; both same-seed invocations run the same assertions.
- **The block is bidirectional n3 isolation for one virtual second.** It is long enough for the split write to reach both live nodes and demonstrably not n3, and short enough that peer_timeout never evicts, so the heal rides the retransmission ladder inside the documented bound.
- **Parts are counted from `TraceDigest`, not a new counter.** The trace is the harness's existing exported observation seam; a test-only production counter would have been exactly the "production-only shortcut" the plan forbids.
- **The joiner round has four parts.** `SyncChunkBytes=100` is below `len(key)+24` plus the 96-byte chunk overhead, so every entry closes its chunk; four keys therefore mean four parts, and the flip-step assertion requires that exact count from the completing peer.
- **No golden digest exists in the simnet extension gate.** The gate compares same-seed runs; the added comment states the extra listing traffic changes what is compared, never equality.
- **The F14 bound is the empty-disk shape for one key.** One Hello one-way + one immediate listing chunk + one delivery one-way + four pacing slots of slack, all stated terms.

## Deviations from Plan

### Documented Adaptations

**1. `runThreeNodeScenario` threads `*testing.T`**
- **Found during:** Task 1 authoring
- **Issue:** The plan's stated signature `runThreeNodeScenario(seed int64) (*simnet.Cluster, string)` gives the factored body no way to assert the plan's own mid-partition requirement ("assert node 3 has not converged") at the instant that state exists.
- **Fix:** Added `t *testing.T` as the first parameter and `t.Helper()`; the returned `(*simnet.Cluster, string)` contract, the two same-seed invocations and the digest-equality assertion are exactly as planned.
- **Files modified:** `internal/app/repl/sim_scenario_test.go`
- **Verification:** `TestScenario_threeNode` passes; the mid-partition absence/model-divergence assertions and `FaultStats().Partitioned > 0` all fire.
- **Committed in:** `6cd08ac`

**2. `sim_bound_test.go` required no re-derivation**
- **Found during:** Task 1 action step 4
- **Issue:** The plan expected the extra first-seen/three-node listing traffic might move a budget.
- **Fix:** None needed — the bound suite's measurements start after `quiesceDiscovery`, whose first-seen listings (one empty chunk per pair) deliver within two virtual milliseconds, and the three-node scenario runs outside the bound suite. All bounds stayed upper bounds with their existing mechanisms, so nothing was widened or rewritten.
- **Files modified:** none (`TestBound_*` green unchanged)
- **Verification:** `go test -race -count=1 ./internal/app/repl/ -run '^TestBound_'` passes.
- **Committed in:** `6cd08ac` (audit note in the commit message)

## Issues Encountered

- **Unexpected GREEN in Task 1's TDD phase.** All three target scenarios passed on their first run. Investigation confirmed the asserted production behavior already exists (the DataSync completeness contract from 04-01, the per-part observe path from 04-03, the first-seen listing path from 04-08 and the peer identity/classification work from 04-09); the task's only deliverable is test evidence, so there is no implementation to write and no legitimate RED to record. Following the 04-07 precedent, no RED record was fabricated and the outcome is disclosed here.
- **Two lint findings in the new test code** (`stringsseq` on `strings.Split`, and `noinlineerr` on two inline error checks) were fixed via `strings.SplitSeq` and plain assignments inside the task-1 commit. No suppressions were added.

## TDD Gate Compliance

- **Task 1 (`tdd="true"`):** unexpected GREEN — the three target tests passed before any implementation change, because the behavior they assert shipped in earlier phase-4 plans. No RED evidence record exists and none was manufactured; the test commit `6cd08ac` is the deliverable, and this disclosure is the required semantic assessment.
- **Task 2 (non-TDD):** committed as `test(04-10)`; test/comment-only, no RED expected.
- **REFACTOR:** none separately — the two lint-shaped fixes were folded into `6cd08ac`; no production simplification applied.

## Known Stubs

None. The plan is test-only; no placeholder values, skipped tests or unwired data sources were introduced.

## Threat Flags

None — no new trust-boundary surface (test code only, no new dependency). T-4-31 is mitigated as planned: the scenarios assert exported contracts (`SyncArrived`, `ListingComplete`, store convergence, trace digest) and never internal counters where the contract exists. T-4-32 is mitigated: the only bound touched is `boundPartitionHeal`, used unchanged, and the audit records that no assertion was deleted or widened.

## Next Phase Readiness

- PROTO-09's simulator evidence and PROTO-01's completeness/value-preservation evidence are in place alongside the existing real-socket wiring proof.
- `go test -race -count=1 ./internal/app/repl/` (full package, including chaos determinism replay and the bound suite), `go test -race -count=1 ./internal/app/internal/simnet/` and the strict lint gate are green.
- PROTO-09 and PROTO-01 remain unchecked in REQUIREMENTS.md: `requirements.ready-ids` blocks both because 04-12 also owns PROTO-09, 04-11 also owns PROTO-01, and the 04-13 phase gate owns both. 04-13 closes them.
- 3 plans remain in Phase 4. Ready for `04-11-PLAN.md`.

---

*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Files exist: `internal/app/repl/sim_scenario_test.go`, `internal/app/repl/decision_coverage_test.go`, `internal/app/internal/simnet/simnet_ext_test.go`, this SUMMARY.
- Commits exist and are ancestors of HEAD: `6cd08ac`, `14b6958`.
- Measured plan commits: 2 from ledger base `cbb715e6`, matching the frontmatter `commits: 2`.
- No tracked file was deleted by either commit.

