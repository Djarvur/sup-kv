---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 05
subsystem: testing
tags: [simulator, deterministic-testing, crdt-convergence, partition-scenarios, clock-skew, repl-core, simnet, virtual-time]

requires:
  - phase: 06-observability-logging-metrics
    provides: repl.Core MergeStats (Conflicts + DriftWarnings counters, OBS-01) — the D-10 winner+counter evidence and the drift-warning assertions read them
  - phase: 04-replication-and-deterministic-simulator
    provides: simnet scheduler/network/cluster, repl core restart+reconciliation contracts (REPL-07), the chaos skeleton and the documented bound helpers
provides:
  - simnet node-lifecycle seams — Cluster.SetClockOffset (per-node mutable clock offset), Cluster.ReplaceNode (in-place REPL-07 restart), Cluster.AddNode (empty-disk join), Network.Reregister (in-place registration swap)
  - The TEST-04 simulator scenario suite — nine deterministic virtual-time scenarios: clockJumps, farAhead, partitionSeconds, partitionDays, oneWay, loss, burst, emptyDisk, restart
  - "TEST-04's simulator half complete: partitions (seconds/days/one-way), datagram+Ack loss with duplicates/reordering, wall-clock jumps live and across restart, far-ahead clock, ~1000 writes/s burst, empty-disk recovery at sync_response_rate, restart-via-RunID"
affects: [TEST-04 verification, 06-08 benchmarks (burst correctness pinned separately from throughput), phase verification]

actuals:
  tokens: 14790    # chars/4 over the realized diff (59171 diff chars across 3 files)
  tasks: 3
  commits: 3       # MEASURED: git rev-list --count 7d7252f..HEAD
plan_head_before: 7d7252f8409adf2ca2f7a00906c33a94219ae5c7
plan_head_after: 816b45d3fc8f76fb410de95836c8808662534049

tech-stack:
  added: []        # zero new dependencies; stdlib + existing in-repo harness only
  patterns:
    - "convergence-driven drives, not quiescence-driven: bare three-seam quiescence also holds in the gap BEFORE a heal path starts (pre-fill, pre-periodic-round) — assert store-equality-with-model as the loop condition and quiescence only as a settling invariant"
    - "mutable fault windows behind a pointer the policy closure dereferences at send time — a plain slice variable is captured by its old header and later appends are invisible to the network"
    - "virtual-time economics: periodic cadence (hello/scan granularity) is linear in virtual time — scale ONLY the cadence knobs for days-scale runs; every merge/winner/bound semantic the scenario asserts is cadence-independent"
    - "egress-queue observability boundary: a burst-enqueued FIFO (two listings landing together) is pollable; a request spaced at exactly the limiter's spacing drains inside one delivery cascade and is structurally invisible to boundary polling"

key-files:
  created:
    - internal/app/repl/sim_scenario_test.go
  modified:
    - internal/app/internal/simnet/simnet.go
    - internal/app/repl/sim_bound_test.go

key-decisions:
  - "simnet gained ReplaceNode/AddNode/Network.Reregister beyond the planned SetClockOffset seam (Rule 3): Task 1's restart-time jump and Task 3's restart/empty-disk scenarios require in-place node replacement wired into the virtual network — Network.Register appends (stale registration would keep receiving broadcasts), so the harness needed an in-place swap; simnet is the test-only harness package, zero production code touched"
  - "partitionDays scales only the periodic cadence (hello_interval and retransmit scan granularity to one sync_interval, peer_timeout to three hellos): at the defaults a multi-day run costs ~20s of wall time in hello events plus one no-op retransmit scan per 100ms per retained suspended entry; every asserted semantic is cadence-independent and documented at the config"
  - "quiescence is not recovery: emptyDisk's bound is measured to converged-AND-quiescent (the cluster is also quiescent in the up-to-sync_interval gap before the peers' periodic listing starts the fill), and the restart stranded-key spread drives until store equality for the same reason"
  - "the loss cross-phase identical-supremum assertion compares winning DATA per key, not version bytes: a node's l watermark depends on which remote stamps it absorbed before a local write, so version bytes are delivery-order-dependent by design — the join law makes the fixpoint data order-independent (CORE-06)"
  - "boundPartitionHeal signature narrowed to boundPartitionHeal(keys) (unparam, lint gate): syncInterval and simDefaultResponseRate are the normalized defaults every caller means; same numbers, no re-derivation"
  - "farAhead pins the drift counter's asymmetry: the 24h-ahead node counts zero warnings (its wall matches its stamps) while both behind peers warn on post-absorption applies — the absorb-then-warn sequence from TestMergeStats_counts at cluster scale"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "simnet node-lifecycle seams: SetClockOffset, ReplaceNode, AddNode, Network.Reregister — deterministic, no rng draws, stale timers superseded"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "go test -race ./internal/app/internal/simnet/ (TestSimnet_determinism byte-identical trace across the seam rework)"
        status: pass
      - kind: unit
        ref: "go test -race ./internal/app/repl/ -run '^TestScenario_clock|^TestScenario_farAhead$' -v"
        status: pass
    human_judgment: false
  - id: D2
    description: "Wall-clock-jump and far-ahead scenarios: live forward/backward jumps keep strict version dominance and convergence; restart-time jump re-converges; 24h-ahead node wins concurrent keys by version order; behind peers count drift warnings, the ahead node none"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_clockJumps"
        status: pass
      - kind: integration
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_farAhead"
        status: pass
    human_judgment: false
  - id: D3
    description: "Partition scenarios: seconds (documented heal bound, divergence teeth), days (deterministic pointwise-supremum winner per key + MergeStats conflicts >= displaced versions, D-10 no journal), one-way connectivity (receiver holds union mid-block, supremum after lift)"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_partitionSeconds TestScenario_partitionDays TestScenario_oneWay"
        status: pass
    human_judgment: false
  - id: D4
    description: "Datagram/Ack-loss scenario: directed loss episodes (initial + 200/600ms ladder dropped, +1400ms delivered) with per-write absence teeth and the inclusive ack_timeout_max bound; duplicate/reorder phase replays the identical script converging to the identical winning data per key"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_loss"
        status: pass
    human_judgment: false
  - id: D5
    description: "~1000 writes/s burst: 3000 writes at 1ms virtual cadence over 3s (hot key + 300 spread keys), zero lost writes, bounded convergence to the supremum — correctness only, throughput is 06-08's"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_burst"
        status: pass
    human_judgment: false
  - id: D6
    description: "Empty-disk recovery bound: brand-new node recovers the full 50-key state within sync_interval + 2N*floor(1s/rate) + 2 spacings of its first Hello, arithmetic documented at the bound"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_emptyDisk"
        status: pass
    human_judgment: false
  - id: D7
    description: "Restart-via-RunID in cluster (REPL-07): in-place replacement re-converges via the targeted listing inside 2s (far under one sync_interval), peer row carries the new run id, solicited replay limiter-gated, stranded old-run write merges by version merit and spreads"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "internal/app/repl/sim_scenario_test.go#TestScenario_restart"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 05: Simulator Partition Scenarios Summary

The TEST-04 simulator half: nine deterministic virtual-time failure scenarios (clock jumps live and across restart, far-ahead clock, seconds/days/one-way partitions, datagram+Ack loss with duplicates and reordering, 1000 writes/s burst, empty-disk recovery at sync_response_rate, restart-via-RunID) over a simnet extended with per-node clock offsets and in-place node replacement.

## Performance

- **Duration:** 35min
- **Started:** 2026-11-16T00:22:59Z
- **Completed:** 2026-11-16T00:57:00Z
- **Tasks:** 3/3
- **Files modified:** 3

## Accomplishments
- One new simnet capability family, exactly as planned plus the replacement seams the scenarios required: SetClockOffset (the 06-RESEARCH OQ3 Cluster-method shape), ReplaceNode/AddNode/Reregister — all deterministic, all zero-rng
- Nine scenario tests, each seeded, replay-printing, quiesced on the three seams, and converged against the pointwise-supremum model — the whole suite runs in ~4s of wall time (virtual time only; days cost nothing)
- D-10 evidence shape held everywhere: deterministic winner asserted via model equality plus the MergeStats conflicts counter; no journal, no per-conflict events anywhere in the diff

## Task Commits

Each task was committed atomically:

1. **Task 1: simnet clock-offset seam + wall-clock-jump/far-ahead scenarios** - `f414832` (feat)
2. **Task 2: Partition scenarios — seconds, days, one-way, loss** - `641b92d` (test)
3. **Task 3: Burst, empty-disk recovery bound, restart resync** - `816b45d` (test)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified
- `internal/app/repl/sim_scenario_test.go` — the nine-scenario TEST-04 simulator suite (created)
- `internal/app/internal/simnet/simnet.go` — clock-offset seam, ReplaceNode/AddNode node lifecycle, Network.Reregister in-place swap
- `internal/app/repl/sim_bound_test.go` — boundPartitionHeal signature narrowed to its constant terms (unparam)

## Decisions Made
See key-decisions frontmatter; the load-bearing ones:

- **simnet replacement seams beyond the plan's one seam** (Rule 3): the plan's own Task 1 (restart-time jump) and Task 3 (in-place restart) require wiring a fresh core into the virtual network under an existing handle; Network.Register appends, leaving the stale registration receiving broadcasts — exactly the duplicate-node misconfiguration REPL-08 forbids. Reregister swaps in place; simnet is the test-only harness package (no production code touched).
- **Days-scale cadence economics**: multi-day virtual runs at default cadence cost ~20s wall (hello every 1s + one no-op retransmit scan per 100ms per suspended-retained entry). Only hello_interval, scan granularity and peer_timeout (proportionally) scale; every asserted semantic is cadence-independent, documented at the config site.
- **Convergence-driven, not quiescence-driven**: both emptyDisk's fill and the restart stranded-key spread sit in quiescent-but-diverged gaps by design; the drives loop on store equality with the model.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] simnet in-place node replacement capability**
- **Found during:** Task 1
- **Issue:** Task 1's restart-time clock jump and Task 3's in-place restart require replacing a node's core/store/runID under its existing handle; the harness had no such capability (Network.Register appends, so the old registration keeps receiving broadcasts and stamping the old run id)
- **Fix:** Cluster.ReplaceNode + Network.Reregister (in-place swap, generation-counter stale-timer supersession) and Cluster.AddNode for the empty-disk join; offsets/cfg remembered on Cluster, node construction unified in newClusterNode
- **Files modified:** internal/app/internal/simnet/simnet.go
- **Verification:** TestSimnet_determinism byte-identical across the rework; clockJumps/farAhead/emptyDisk/restart scenarios green
- **Committed in:** f414832

**2. [Rule 1 - Bug] loss windows invisible to the partition closure**
- **Found during:** Task 2 (development)
- **Issue:** the directed loss windows were appended to a slice captured by value in the policy closure — nothing was ever blocked and every "loss" write delivered immediately
- **Fix:** windows behind a pointer-dereferenced struct field the closure reads at send time
- **Files modified:** internal/app/repl/sim_scenario_test.go
- **Verification:** absence teeth and per-write ack_timeout_max bounds now assert and pass
- **Committed in:** 641b92d

**3. [Rule 1 - Bug] loss check timeline not monotonic**
- **Found during:** Task 2 (development)
- **Issue:** the per-write bound checks were appended per write (absent, served) so an early write's +5s served check leapt the virtual clock past a later write's still-open loss window before its absence check ran
- **Fix:** checks sorted by deadline and executed on one merged monotonic timeline with the writes
- **Files modified:** internal/app/repl/sim_scenario_test.go
- **Verification:** every check fires at its own virtual time; suite green
- **Committed in:** 641b92d

**4. [Rule 3 - Blocker] partitionDays wall-time cost at default cadence**
- **Found during:** Task 2 (development)
- **Issue:** a multi-day virtual partition at default cadence costs ~20-24s of wall time (hello events linear in virtual time; one no-op retransmit scan per 100ms per entry retained in suspension across the split) — against the seconds-of-wall-time suite budget
- **Fix:** scale only hello_interval/RetransmitInterval to one sync_interval and peer_timeout to three hellos for the days scenario, with the semantics-preservation argument documented at the config
- **Files modified:** internal/app/repl/sim_scenario_test.go
- **Verification:** partitionDays 2.6s wall; whole nine-scenario suite ~4s
- **Committed in:** 641b92d

**5. [Rule 3 - Lint gate] boundPartitionHeal unparam**
- **Found during:** Task 2
- **Issue:** the second scenario call site gave boundPartitionHeal a second caller, exposing that syncInterval/responseRate always carry the normalized defaults (unparam)
- **Fix:** signature narrowed to boundPartitionHeal(keys) with the constants inlined at the function — same numbers, no re-derivation, existing bound test updated mechanically
- **Files modified:** internal/app/repl/sim_bound_test.go
- **Verification:** go tool golangci-lint run ./... exit 0; TestBound_partitionHeal passes unchanged in semantics
- **Committed in:** 641b92d

---

**Total deviations:** 5 auto-fixed (1× Rule 1 bug ×2 sites, 3× Rule 3 blocker/lint)
**Impact on plan:** none on scope or assertions — every planned scenario and acceptance criterion landed; the deviations are harness capabilities the plan implicitly required and measurement-harness corrections found by the tests' own teeth.

## Issues Encountered
None beyond the deviations above. The tracer's verify gate ran end-to-end clean on the first Task-1 commit and again after Task 2/3 (auto mode active; no checkpoint required).

## Authentication Gates
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- TEST-04's simulator half is complete; the real-socket (06-06, done) and exec (06-07, done) halves already exist — TEST-04's suite is now three-thirds automated
- 06-08 (benchmarks) can build on the burst cadence precedent with performance measurement only, its correctness half pinned here
- The simnet seams (SetClockOffset, ReplaceNode, AddNode) are available for any later scenario work at zero determinism cost

## Verification Evidence
- `go test -race ./internal/app/repl/ -run '^TestScenario_' -v` — 9/9 PASS
- `go test -race ./internal/app/repl/` — whole package ok (sim_bound, sim_chaos, restart, discard, retransmit, sync, merge_stats suites unmodified and green)
- `go test -race ./internal/app/internal/simnet/` — ok, determinism byte-identical
- `go test -race ./...` — exit 0
- `go tool golangci-lint run ./...` — exit 0
- Replay spot-check: every run prints its SUPKV_SIM_SEED replay command; every failure observed during development printed it too (chaos skeleton), and fixed-seed replays (`SUPKV_SIM_SEED=987654321`, `-count=2`) reproduce deterministically

## Self-Check: PASSED

- All 3 key files exist on disk (sim_scenario_test.go, simnet.go, sim_bound_test.go)
- All 3 task commits (f414832, 641b92d, 816b45d) are ancestors of HEAD
- `check evaluation-scope --plan 06-05 --commits-only` resolved with 3 commits
- All task acceptance criteria re-run and passing (see Verification Evidence)

---
*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
