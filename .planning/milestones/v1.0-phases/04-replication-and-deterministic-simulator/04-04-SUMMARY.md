---
phase: 04-replication-and-deterministic-simulator
plan: "04"
subsystem: replication
tags: [udp, replication, restart, run-id, datasync, anti-entropy, repl-07, deterministic, event-loop, go]

# Dependency graph
requires:
  - phase: 04-replication-and-deterministic-simulator/plan 01
    provides: the Hello upsert in handleHello, the peer table keyed by node_id and the InboundDatagram run_id field this plan's discrimination branches on
  - phase: 04-replication-and-deterministic-simulator/plan 03
    provides: the deterministic listing/chunking machinery (listingChunks, formerly inline in syncListing) the targeted restart listing reuses, and the egress-queue scope pin it bypasses
provides:
  - REPL-07: a Hello from a known node_id whose run_id differs from the remembered one updates the entry in place (handle, run_id, lastHello — one table row) and fires exactly one full current DataSync listing unicast toward the restarted peer, every chunk in chunk order, once per detected change
  - the discrimination contract: unknown node_id learns without a targeted listing (startup DataSync + periodic rounds reconcile it); same-run_id repeat refreshes lastHello only (idempotence bounds the flap case, not rate limiting)
  - the #19 edge pins, doc-pinned in the Hello handler and test-pinned: inequality with the remembered run_id is the sole trigger (random run_ids are unordered — a stale return is just another change, idempotent re-sync); in-flight Ack/DataReplication from the old run process on (key, version) merits; rapid double restart updates in place, one listing per change
  - the no-re-tracking pin: a detected restart never re-adds wait-sets and never touches the backoff ladder — reconciliation heals a restarted peer, retransmission does not
affects: [04-05 simulator drives restart scenarios on the same reaction, 04-06 duplicate-node_id/second-handle classification and the restart-vs-duplicate doc pin, phase 05 startup gate reads the full-sync reaction, phase 06 OBS metrics and restart failure scenarios]

# Actuals (#2632) — pairs with the plan's `estimate` (estimateTokens scale: chars/4 over the realized diff).
actuals:
  tokens: 5423
  tasks: 2
  commits: 3

plan_head_before: 8804c8b41d78080ac189ffd959d9c64e7cb6643c
plan_head_after: b3ee2ceaa689df4937cd003edbfa1e1df33bac9f

# Tech tracking
tech-stack:
  added: []
  patterns: [doc-comment pinning of an unresolved probe reading at its single decision point (the Hello handler carries the whole #19 discrimination contract), reaction reuses prior-plan machinery via an extracted shared build (listingChunks) proven equal to an independent fresh build by a second core over the same store, discovery/restart mutual exclusivity by construction (a restart needs a remembered row and an empty table remembers nothing)]

key-files:
  created:
    - internal/app/repl/restart_test.go
  modified:
    - internal/app/repl/core.go

key-decisions:
  - "Inequality with the remembered run_id is the sole restart trigger: run_ids are 16 random bytes per process start with no temporal order, so a stale return is indistinguishable from a fresh restart and is handled as just another change — entry flip plus one idempotent re-sync listing, never a second table row"
  - "The restart listing bypasses the egress limiter (proactive, once per detected change — the plan-04-03 egress-scope pin) while the replay traffic it solicits stays limiter-gated; no wait-set re-add, no ladder touch — the restarted node heals through reconciliation, not retransmission"
  - "The reaction reuses the plan-04-03 listing verbatim through listingChunks, extracted from syncListing and shared by both triggers — one chunking implementation, no new exported symbols"

patterns-established:
  - "Independent-fresh-build equality: the expected chunk sequence comes from a second core's startup listing over the same store, proving machinery reuse instead of a parallel test-side implementation"
  - "Generation-tagged run ids (runIDFor) let one synthetic node carry R1/R2/R3 across stale-return and double-restart sequences deterministically"

requirements-completed: [REPL-07]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "REPL-07 reaction: a Hello with a changed run_id from a known node updates the entry in place (one PeerSnapshot row, new run_id and handle) and emits exactly the fresh plan-04-03 two-chunk listing unicast to the restarted peer's handle with nothing else in the outbox, bypassing the egress queue (QueuedResponses stays 0); a same-run_id repeat emits nothing (idempotence)"
    requirement: REPL-07
    verification:
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_runIDChangeTriggersFullSync (-race green)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Discrimination: a brand-new node_id is learned into PeerSnapshot with zero unicast DataSync toward it (its own startup DataSync and the periodic rounds reconcile it) and zero messages of any kind from the learn event"
    requirement: REPL-07
    verification:
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_newPeerNoFullSync (-race green)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Stale run_id returning (R1→R2 seen, delayed R1 arrives: entry flips back to R1, second idempotent full listing, exactly two listings total, never two rows) and rapid double restart (R1→R2→R3 across consecutive Advances: one full listing per change, two total, single row ending at R3)"
    requirement: REPL-07
    verification:
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_staleRunIdAndInFlight (subtests 1 and 3, -race green)"
        status: pass
    human_judgment: false
  - id: D4
    description: "In-flight old-run traffic processes run_id-agnostically on (key, version) merits: an old-run Ack drains the pending waiter, an old-run DataReplication with a strictly greater version supersedes the pending entry, installs in the store and is Acked"
    requirement: REPL-07
    verification:
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_staleRunIdAndInFlight (subtest 2, -race green)"
        status: pass
    human_judgment: false

# Metrics
duration: 6 min
completed: 2026-11-06
status: complete
---

# Phase 4 Plan 04: Restart Semantics — Targeted Full Sync on run_id Change Summary

**REPL-07 shipped: a detected restart (known node_id, changed run_id) updates the peer row in place and fires exactly one full plan-04-03 chunked listing unicast toward the restarted peer — limiter-bypassing, idempotent, no re-tracking — with the stale-return, in-flight old-run traffic and double-restart edges pinned.**

## Performance

- **Duration:** 6 min (19:00–19:06 UTC)
- **Started:** 2026-11-06T19:00:47Z
- **Completed:** 2026-11-06T19:06:37Z
- **Tasks:** 2 (Task 1 TDD red-green; Task 2 adversarial-edge tests)
- **Files modified:** 2 (1 core, 1 test file created)

## Accomplishments
- REPL-07 proven: the known-node_id/changed-run_id branch of the Hello handler updates the entry (handle, runID, lastHello — one row per node_id, never a fork) and emits the full current DataSync listing UNICAST to the restarted peer's handle, every chunk in chunk order from one Records() snapshot through the plan-04-03 machinery (listingChunks extracted from syncListing and shared by both triggers — grep-verified single chunkSyncEntries call site)
- Discrimination exact: unknown node_id learns with no targeted listing; same-run_id repeat refreshes lastHello only — the flap case is bounded by idempotence, not rate limiting; the listing bypasses the egress queue (QueuedResponses() == 0 while it emits) per the plan-04-03 egress-scope pin, while the replay traffic it solicits stays limiter-gated
- Probe #19 edges pinned in the Hello handler's doc comment and in tests: inequality with the remembered run_id is the only trigger (random 16-byte run_ids carry no order — a stale return is just another change: entry flip plus one idempotent re-sync, exactly two listings total, never two rows); in-flight Ack/DataReplication from the old run process normally on (key, version) merits (drain, supersession and re-Ack proven); rapid double restart R1→R2→R3 updates in place with one listing per change
- No-re-tracking pin (REPL-02 interplay): a detected restart adds no wait-set members and touches no ladder — a restarted node heals through reconciliation via the listing/compare loop; full internal/app/repl sweep (plans 04-01..04-04, 14 top-level tests) green under -race with -v; package, full-repo lint and full make-test parity green

## Task Commits

Each task was committed atomically (TDD: RED test commit then GREEN feat commit; Task 2 is a pure test-hardening task):

1. **Task 1 RED: failing TestRestart_runIDChangeTriggersFullSync + TestRestart_newPeerNoFullSync + helpers** - `595a625` (test)
2. **Task 1 GREEN: run_id-change targeted full listing in the Hello handler** - `908ee0a` (feat)
3. **Task 2: TestRestart_staleRunIdAndInFlight (stale return, in-flight traffic, double restart)** - `b3ee2ce` (test)

**Plan metadata:** (this SUMMARY commit)

The single RED run was machine-classified `RED_EVIDENCE_OK` (go-test adapter: TestRestart_runIDChangeTriggersFullSync failed on its planned listing-equality assertion — empty listing vs the fresh two-chunk sequence; TestRestart_newPeerNoFullSync passed pre-implementation as expected, being the discrimination regression lock). The GREEN commit directly descends from the RED commit (merge-base verified). No REFACTOR commit: the GREEN landed with the listingChunks extraction already in its minimal shape.

## Files Created/Modified
- `internal/app/repl/core.go` - handleHello restart branch (detection before the upsert, targeted unicast listing emission, the full #19 discrimination doc-comment pin), listingChunks extracted from syncListing and shared by the sync timer and the restart reaction, syncListing/HandleDatagram doc pointers updated to the now-shipped REPL-07 reaction
- `internal/app/repl/restart_test.go` - TestRestart_runIDChangeTriggersFullSync, TestRestart_newPeerNoFullSync, TestRestart_staleRunIdAndInFlight (3 subtests) + helpers (runIDFor, helloFromRun, targetedSyncs, seedGraduatedKeys, freshListing)

## Decisions Made
- Inequality-trigger pin: the remembered-vs-announced run_id comparison is the whole restart test — no ordering, no history, no dedup state; this makes a stale return and a fresh restart the same event and keeps the table at one row per node_id
- Discovery/restart mutual exclusivity by construction: a restart requires a remembered row; an empty table (post-eviction) remembers nothing, so an evicted peer's return Hello — whatever its run_id — is the unknown-node case (learn, no listing)
- The restart listing is proactive traffic outside the egress limiter, exactly like the timer/startup DataSync; the amplification bound is the once-per-change idempotence rule plus chunk framing
- Test-side expected sequences come from an independent fresh core's startup listing over the same store (the plan-04-03 machinery itself), never a test-side reimplementation of chunking

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- unparam flagged runIDFor/helloFromRun after Task 2 (nodeID always peerID1): fixed in code direction by dropping the always-constant parameter — every restart in the suite is peer B; lint green, no suppression.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Ready for 04-05 (simulator): restart scenarios drive the same handleHello reaction; the doc-pin and listing counts give the simulator exact invariants to assert on the virtual clock
- Ready for 04-06: the duplicate-node_id second-handle classification (T-04-09's concurrent two-sources case) and the restart-vs-duplicate discrimination doc pin land there against this plan's handler shape
- D-01..D-05 intact: no new exported symbols, zero goroutines, zero locks, zero wall-clock reads inside internal/app/repl (grep-verified); go.mod/go.sum untouched (T-04-SC)
- Full make lint + make test green (CI parity)

## Self-Check: PASSED

Key files exist on disk (core.go, restart_test.go); the three task commits 595a625..b3ee2ce are ancestors of HEAD on this branch (evaluation-scope resolved 3 commits for plan 04-04); all plan verification bullets re-run green under -race (three TestRestart tests plus the full 14-test package sweep); go tool golangci-lint run ./internal/app/repl/... and full-repo lint exit 0; purity greps clean (no time.Now/time.Sleep/go func in non-test repl sources); the pinned reading greps in the Hello handler doc comment (inequality trigger, no temporal order, run_id-agnostic); chunkSyncEntries has exactly one call site (inside listingChunks — no second listing implementation); TDD gates present with RED-before-GREEN ancestry.

---
*Phase: 04-replication-and-deterministic-simulator*
*Completed: 2026-11-06*
