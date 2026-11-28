---
phase: 04-protocol
plan: 08
subsystem: replication-protocol
tags: [proto-04, cr-01, listing-queue, shared-build, generation-freshness, first-seen-listing, tdd, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 03
    provides: "StateStore.Generation (the content generation the cached listing build records) and the per-part DataSync handler"
  - phase: 04-protocol
    plan: 05
    provides: "the identity-only egress queue and the paced unicast answer path the listing drain sits beside"
  - phase: 04-protocol
    plan: 01
    provides: "the completeness header, the listing accumulator and the plan-04-03 listing machinery this plan rewrites"
  - phase: 03-transport
    provides: "the restart-listing queue CR-01 found unbounded and handle-keyed, the shared listing pacer and the bounded-queue idioms"
provides:
  - "listingDst (node_id, run_id) destinations with the zero identity as the broadcast round — a source handle is never a queue identity"
  - "listingBuild{chunks, at, syncID, generation} shared by every FIFO while the age and the store generation still hold; a mutation forces exactly one rebuild"
  - "maxListingDestinations 32 / maxListingChunks 65536 caps with a counted drop and the ListingDropped() accessor"
  - "listingFIFO{build, dst, startIndex, emitted}: O(1) per destination, rotation by original chunk index"
  - "restart/eviction FIFO cleanup so queued listing work cannot outlive the peer row"
  - "first-seen Hello semantics: exactly one immediate targeted listing, plus the discovery flush in the same event"
affects:
  - "04-09 classified its Hello outcomes on this first-seen path; the stale-return alternation classification still treats a new run_id as a restart here"
  - "04-12's multi-node documentation describes the first-seen listing as the no-sync-interval-term heal"
  - "Phase 7 hardening inherits the still-unbounded peer table (Open Question 5), now decoupled from the bounded listing state"

actuals:
  tokens: 18000
  tasks: 3
  commits: 5
  plan_head_before: c6355093a3d1ecee34b3c470feb4d81745dbf68b
  plan_head_after: 70ff0dd648235fbc4f34e798f5644bc98bdd1514

tech-stack:
  added: []
  patterns:
    - "Shared immutable build with a content-generation freshness check: one O(store) copy per (age, generation), reused by every destination, rebuilt exactly once on any mutation — a cache that cannot serve stale state because staleness is a recorded generation mismatch, not a heuristic."
    - "Identity-keyed queues: a queue destination is a protocol identity (node_id, run_id), never a transport attribute (source handle); a spoofed port buys no queue entry."
    - "One enqueue path for every listing trigger (periodic, first-seen, restart), with the caller supplying the rotation; no second implementation can drift."
    - "Cleanup co-located with the state transition: the run change and the peer eviction drop the FIFO in the same event that drops the row/keys, so no queued work outlives its owner."

key-files:
  created:
    - internal/app/repl/listing_internal_test.go
    - .planning/phases/04-protocol/red-evidence/04-08-task1.json
    - .planning/phases/04-protocol/red-evidence/04-08-task2.json
  modified:
    - internal/app/repl/core.go
    - internal/app/internal/simnet/simnet.go
    - internal/app/repl/restart_test.go
    - internal/app/repl/discard_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/sim_scenario_test.go

key-decisions:
  - "A listing destination is listingDst{nodeID, runID} with the zero identity reserved for the broadcast round; the drain resolves the current handle from the peer row, so an attacker-chosen source port can never mint a FIFO (CR-01/T-4-25)."
  - "listingBuild carries the round's sync_id: one id per build, not per trigger. Two destinations reusing one build share the round identity, which is safe because the chunks are byte-identical — and it keeps the trigger path free of counter work."
  - "Only a restart advances the per-peer rotation counter; the first-seen round starts at chunk 0 and leaves the counter alone. This preserves the pinned rotation contracts (runIDChangeTriggersFullSync expects the first restart unrotated, the stale-return row the next rotation, the double restart one rotation per change) while still covering the key space under repeated restarts."
  - "The generation is read through StateStore.Generation() on every trigger and recorded on the build; reuse requires BOTH now-build.at < SyncInterval AND an unchanged generation, so a mutation always forces exactly one rebuild before the next listing (T-4-24)."
  - "broadcastDst is a function returning the zero listingDst rather than a package-level var, keeping the core's state model free of mutable package state under the strict lint gate."
  - "The cap order is destination-first (cheap refusal), then the cached build, then the queued-chunk bound; a refused trigger is counted once and leaves no partial FIFO behind (T-4-23)."
  - "The RED commits carry only compile scaffolding — the two bound constants, the listingDropped counter and its copying accessor — so the white-box tests build and fail on their planned assertions instead of failing as a build_failure (the 04-03 precedent)."
  - "The bench DataSync fixture was already a complete, header-valid 1-of-1 listing from 04-01 and its Versions/Lookup accounting is untouched by this plan, so bench_test.go needed no edit; sim_bound/sim_chaos budgets also held unchanged."
  - "TestScenario_emptyDisk's bound drops the sync_interval term entirely (simOneWay + 2N*spacing + 4*spacing) because the joiner's own first Hello now triggers the peers' first-seen listings; the new bound is ~200ms instead of ~33s, a strictly tighter gate."

patterns-established:
  - "Cache honesty by recorded generation: a snapshot cache stores the content generation it was copied at and reuses only while it matches, so a mutation costs one rebuild and stale service is structurally impossible."
  - "The queue key is the protocol identity; the transport handle is resolved at emission time from the authoritative row."
  - "First-seen and restart share one enqueue function, with only the rotation source differing."

requirements-completed: [PROTO-04]

coverage:
  - id: D1
    description: "CR-01 bounded, peer-identity-keyed listing queue: at most maxListingDestinations in-flight FIFOs and maxListingChunks queued chunks, every refused trigger counted, no source handle as a destination."
    requirement: PROTO-04
    verification:
      - kind: unit
        ref: "internal/app/repl/listing_internal_test.go#TestListing_hostileBounded"
        status: pass
      - kind: unit
        ref: "internal/app/repl/listing_internal_test.go#TestListing_restartDropsOldRunFIFO"
        status: pass
    human_judgment: false
  - id: D2
    description: "One shared immutable listing build per (sync_interval age, store generation): two triggers inside one age and generation copy Versions() once; a store mutation forces exactly one rebuild and the emitted chunks carry the mutated store."
    requirement: PROTO-04
    verification:
      - kind: unit
        ref: "internal/app/repl/listing_internal_test.go#TestListing_sharedBuild"
        status: pass
      - kind: unit
        ref: "internal/app/repl/listing_internal_test.go#TestListing_generationRebuild"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_generationInvalidatesCachedListing"
        status: pass
    human_judgment: false
  - id: D3
    description: "Rotation reorders emission while every chunk keeps its original 1-based index and the round's chunk_count, so completeness stays recognizable."
    requirement: PROTO-04
    verification:
      - kind: unit
        ref: "internal/app/repl/listing_internal_test.go#TestListing_rotationPreservesIndices"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_rotation"
        status: pass
    human_judgment: false
  - id: D4
    description: "First-seen/repeat/restart/returning Hello outcomes: an unknown peer gets exactly one complete targeted listing and is learned; a repeat emits nothing; a new run_id emits one listing and drops the old run's FIFO; a returning-after-eviction peer takes the first-seen path; a first Hello that ends an empty table also flushes every pending write."
    requirement: PROTO-04
    verification:
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_firstHelloExactlyOneListing"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_repeatHelloNoListing"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_runIDChangeTriggersFullSync"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_returningAfterEvictionGetsListing"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_firstHelloStillFlushesPending"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_helloDoesNotLeakListingsToStranger"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestRetransmit_suspendAndFlush"
        status: pass
    human_judgment: false
  - id: D5
    description: "The whole repl package — simulator scenarios and invariant bounds included — is green with the new discovery traffic: the evicted-partition subtest heals before the first post-heal sync tick, the empty-disk bound loses its sync_interval term, and both determinism replays stay identical."
    requirement: PROTO-04
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_longPartition"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_emptyDisk"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sim_chaos_test.go#TestChaos_determinismReplay"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sim_scenario_test.go#TestSim_determinism_withExtensions"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./internal/app/repl/"
        status: pass
    human_judgment: false
  - id: D6
    description: "simnet.Cluster.AddNode documents the first-Hello targeted listing plus the periodic rounds instead of the stale 'never a targeted listing' contract."
    requirement: PROTO-04
    verification:
      - kind: other
        ref: "go vet ./... (doc comment compiles with the harness) and TestScenario_emptyDisk exercises AddNode on the new path"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 08: Bounded First-Seen Listings Summary

**A first or returning peer now receives one immediate, generation-fresh targeted listing — while the same machinery bounds the CR-01 queue by peer identity with a counted drop and one shared store copy per (age, generation).**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-11-22T05:00:00Z (approx.)
- **Completed:** 2026-11-22T05:30:00Z
- **Tasks:** 3
- **Files modified:** 6 code/test files (+1 created test file, 2 RED evidence records)

## Accomplishments

- CR-01 is fixed structurally. `listingDst{nodeID, runID}` is the queue identity — the zero identity is the broadcast round — so an attacker-chosen source port can never mint a FIFO (T-4-25). `maxListingDestinations = 32` and `maxListingChunks = 65536` bound the queue, every refusal increments the new `ListingDropped()` counter, and a restart or eviction drops the destination's FIFO and releases its remaining chunk work (T-4-23).
- The listing is built once per `(sync_interval age, store generation)` and shared by every FIFO that holds a pointer to it. `Versions()` runs once across two triggers inside one age, and any store mutation forces exactly one rebuild before the next listing — the regression `TestRestart_generationInvalidatesCachedListing` proves a cached pre-mutation build is never served (T-4-24). CR-01's O(store)-per-attacker-trigger cost is gone.
- Rotation is preserved exactly: the shared build's chunks are emitted from `startIndex` onward, each stamping its ORIGINAL 1-based `chunk_index` and the build's `chunk_count`, so a receiver's completeness accumulation is unaffected by order.
- PROTO-04's Hello semantics land in `handleHello`: an unknown `node_id` is learned and gets exactly one complete targeted listing, and the branch FALLS THROUGH to the discovery flush instead of returning early (Pitfall 5), so a first Hello that ends an empty table both lists and flushes (pinned by `TestRestart_firstHelloStillFlushesPending` and `TestRetransmit_suspendAndFlush`). A repeat Hello refreshes liveness only; a new `run_id` drops the old run's listing and enqueues one rotated round; a returning-after-eviction peer is simply unknown again and gets the first-seen listing.
- The invariant wave follows the new traffic: the evicted-partition subtest now requires convergence BEFORE the first post-heal sync tick (the first-seen listing heals it), and `TestScenario_emptyDisk`'s bound drops the `sync_interval` term entirely — `simOneWay + 2N*spacing + 4*spacing`, ~200ms instead of ~33s. Both determinism replays stay identical, and the full `internal/app/repl` package, `go vet ./...` and the package lint gate are green.

## Task Commits

Each TDD task produced its RED and GREEN commits; the invariant wave one adaptation commit:

1. **Task 1 RED: failing listing-queue tests + compile scaffolding** - `a0267a5` (test)
2. **Task 1 GREEN: bounded, shared, generation-checked listing queue (CR-01)** - `1e06c58` (feat)
3. **Task 2 RED: failing first-seen Hello tests** - `375da17` (test)
4. **Task 2 GREEN: list on first Hello without losing the discovery flush** - `20f7b50` (feat)
5. **Task 3: adapt the invariant wave to first-seen listings** - `70ff0dd` (test)

**Plan metadata:** this SUMMARY/STATE/ROADMAP/REQUIREMENTS commit follows this file.

## Files Created/Modified

- `internal/app/repl/core.go` — `listingDst`, `listingBuild`, the rewritten `listingFIFO`, the two caps, `listingDropped`/`ListingDropped()`, `enqueueListing`, `listingBuild`, `dropListing`, the rewritten `drainListings`, `syncListing`, the three-way `handleHello` with first-seen/restart/drop, and the eviction cleanup.
- `internal/app/repl/listing_internal_test.go` — the white-box fixtures (counting `listingStore`, drain/hello helpers) and the five `TestListing_*` rows: hostile bound, shared build, generation rebuild, rotation indices, restart FIFO release.
- `internal/app/repl/restart_test.go` — `TestRestart_firstHelloExactlyOneListing` (replacing `newPeerNoFullSync`), `repeatHelloNoListing`, `returningAfterEvictionGetsListing`, `firstHelloStillFlushesPending`, `helloDoesNotLeakListingsToStranger`, `generationInvalidatesCachedListing`; the stale-run subtest drains its learn listing before the "advance emits nothing" assertion.
- `internal/app/internal/simnet/simnet.go` — `AddNode`'s doc comment now names the first-Hello targeted listing plus the periodic rounds.
- `internal/app/repl/discard_test.go` — the seed Hello drains its own first-seen listing; the stranger row expects its one queued chunk.
- `internal/app/repl/retransmit_test.go` — the stale-record row drains the first-seen listing; the paced-storm row pins the exact `stormPending+1` count with a one-chunk listing.
- `internal/app/repl/sim_scenario_test.go` — the long-partition evicted lower bound follows D-04, the empty-disk bound and comment are re-derived, and the restart scenario's stale "waits for periodic rounds" parenthetical is corrected.
- `.planning/phases/04-protocol/red-evidence/04-08-task{1,2}.json` — both `RED_EVIDENCE_OK`.

## Decisions Made

- **The FIFO key is the peer identity, the handle is resolved at drain time.** Eviction and a run change drop the FIFO in the same event that drops the row/keys, so a stale destination cannot outlive its owner.
- **One `sync_id` per build.** Destinations reusing a build share the round identity; identical chunks make that semantically the same round, and the trigger path allocates nothing.
- **Only restarts advance the rotation counter.** The first-seen round starts at chunk 0; this keeps the pinned rotation contracts green (run-id change → unrotated, stale return → one rotation, double restart → one rotation per change).
- **`broadcastDst` is a function.** The zero identity needs a name at two sites; a package-level var would be the only mutable package state in the core and would need a lint suppression.
- **Cap order: destination count → build → chunk count.** The cheap refusal runs first; the build is cached; a refused trigger is counted once and leaves no partial FIFO.
- **RED scaffolding is declarations only.** The two constants, the counter and its accessor let the white-box tests compile against the old machinery and fail on their assertions (04-03's precedent); no behavior shipped in a RED commit.
- **The empty-disk bound is tighter, not looser.** Removing the `sync_interval` term reduces the allowed recovery from ~33s to ~200ms.

## Deviations from Plan

None - plan executed exactly as written.

The bench fixture and the sim-bound/chaos budgets were audited and needed no edit (see Decisions): the DataSync benchmark already feeds a complete, header-valid 1-of-1 listing from 04-01, and no budget assertion moved.

## Issues Encountered

None. The task-1 test run needed one correction during RED authoring: the hostile probe initially used node id 1, which collides with the local node id and is discarded as `duplicate_node_id`; node ids now start at 2 so all 200 hostile triggers reach the Hello handler deterministically.

## User Setup Required

None - no external service configuration required.

## TDD Gate Compliance

Both TDD tasks followed RED → GREEN with persisted, machine-validated evidence:

- **Task 1:** RED `a0267a5` — `04-08-task1.json` records four target tests failing on their planned assertions (hostileBounded's two probes: 200 destinations vs the 32 cap; sharedBuild: 3 `Versions()` copies vs 1; generationRebuild: 2 copies vs 1 before the trigger; restartDropsOldRunFIFO: 7 chunks vs 4). `rotationPreservesIndices` passed, as designed: it pins preserved rotation behavior and cannot fail pre-implementation. `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK`.
- **Task 1 GREEN:** `1e06c58` made all five rows pass (7 PASS lines with subtests) while the old Hello contract — and therefore every pre-existing test — stayed green.
- **Task 2:** RED `375da17` — `04-08-task2.json` records four first-seen rows failing on their planned assertions plus the first-seen subtest of `generationInvalidatesCachedListing`; the repeat, run-id-change, restart-freshness and stale-run rows passed as preserved-contract guards. `RED_EVIDENCE_OK`.
- **Task 2 GREEN:** `20f7b50` made all seven rows pass (9 PASS lines with subtests) and `TestRestart_|TestRetransmit_suspendAndFlush|TestPeer` green.
- **Task 3** is a non-TDD adaptation task, committed as `test(04-08): adapt the invariant wave to first-seen listings`; no queue assertion was deleted, and the empty-disk bound was tightened rather than widened.

## Known Stubs

None. No placeholder values, TODOs or unwired data sources were introduced.

## Threat Flags

None — no new trust-boundary surface. T-4-23 (bounded destinations/chunks with counted drops), T-4-24 (one build per generation, stale-proof reuse) and T-4-25 (no handle-keyed queue identity) are mitigated as planned; T-4-26 (unbounded peer table) remains the recorded Phase 7 residual and the new per-source state is bounded. No new dependency (T-4-SC).

## Next Phase Readiness

- CR-01 and PROTO-04 are complete and covered; the listing machinery is bounded, peer-keyed, build-shared and generation-fresh.
- 5 plans remain in Phase 4. Ready for `04-09-PLAN.md`, which builds the alternating-run_id classification on this first-seen path.

---
*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Created files exist: this SUMMARY, `internal/app/repl/listing_internal_test.go`, both RED evidence records.
- Commits exist and are ancestors of HEAD: `a0267a5`, `1e06c58`, `375da17`, `20f7b50`, `70ff0dd`.
- The measured plan commit count is 5 from the ledger base `c6355093`, matching the frontmatter `commits: 5`.
- No tracked file from the plan's diff was deleted.
