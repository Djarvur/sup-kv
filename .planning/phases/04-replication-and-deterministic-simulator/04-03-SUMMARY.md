---
phase: 04-replication-and-deterministic-simulator
plan: "03"
subsystem: replication
tags: [udp, replication, anti-entropy, datasync, chunking, rate-limiting, datarequest, deterministic, event-loop, go]

# Dependency graph
requires:
  - phase: 04-replication-and-deterministic-simulator/plan 01
    provides: the port message structs (DataRequest/DataSync/SyncEntry), the Outbox seam, HandleDatagram dispatch and the startup Hello this plan's handlers and triggers extend
  - phase: 04-replication-and-deterministic-simulator/plan 02
    provides: the pending machinery and Advance's strictly-future minimum-deadline contract the egress slot and sync timer join
provides:
  - REPL-04: unicast exact-record DataRequest replies (tombstones included — deletes heal through reconciliation) and provable silence for unknown keys
  - REPL-05 emission: the deterministic chunked DataSync listing — one Records() snapshot, every record sorted by key, chunks closed at exactly sync_chunk_bytes by the conservative estimate, oversized single entries shipped alone, empty store still announcing one zero-entry listing, startup (Hello-then-DataSync) and sync_interval timer triggers
  - REPL-05 receive: the per-key four-case compare against one snapshot — missing/stale queue a DataRequest, strictly-newer local record queues the exact-record unicast reply, equal stays silent, tombstone-newer travels
  - the single rate-limited egress queue: exactly three enqueue sites (all reactive unicasts), floor(1s/rate) integer-nanosecond spacing, first response immediate, drain order equals enqueue order, QueuedResponses() seam for 04-05's quiescence check
  - the cross-seam wire-size proof (udprepl TestSync_chunkWireBudget) that the estimate constants stay conservative against the real protobuf codec, and the IDEA.md §Parameters echo of the rounding rule
affects: [04-04 restart-targeted sync reuses the listing/compare machinery, 04-05 simulator drives the egress deadline and quiescence seam, 04-06 classification totality over the new datagram types, phase 05 gRPC, phase 06 OBS metrics]

# Actuals (#2632) — pairs with the plan's `estimate` (estimateTokens scale: chars/4 over the realized diff).
actuals:
  tokens: 14293
  tasks: 3
  commits: 6

plan_head_before: 784b6b1bd2f357ce742492fb4740b8a68e8d541c
plan_head_after: 32b22ce6f26c6d7472f664daf1e1fe6707c25a7e

# Tech tracking
tech-stack:
  added: []
  patterns: [conservative core-side size estimates with a cross-seam adapter test as the honesty proof (D-01 consequence), one rate-limited FIFO for every reactive unicast with enqueue-order drain, at-exactly-the-budget chunk closure tested by construction (graduated key lengths whose estimates sum to budget minus overhead), prior-plan test assertions updated in code direction when a later plan's designed behavior supersedes them]

key-files:
  created:
    - internal/app/repl/sync_test.go
    - internal/udprepl/syncsize_internal_test.go
  modified:
    - internal/app/repl/core.go
    - internal/udprepl/repl_test.go
    - docs/IDEA.md

key-decisions:
  - "Egress-limiter scope pin: ONE FIFO over all reactive unicasts (exactly three enqueue sites — DataRequest emission, DataRequest reply, DataSync-newer response; Hello/retransmits/DataSync never queue), spacing floor(1s/rate) integer nanoseconds, first response immediate"
  - "Chunk budget by conservative estimate constants (entryOverheadBytes 24 / chunkOverheadBytes 64) because protobuf is adapter-side (D-01); the real-codec guarantee is proven cross-seam in udprepl — a codec change that shrinks the slack breaks the test, not production"
  - "Startup emission order Hello-then-DataSync pinned by TestSync_triggers (IDEA §Node lifecycle); REPL-07 neighbor-restart targeting left to plan 04-04 by doc pointer at syncListing"
  - "The egress queue machinery landed in Task 1's GREEN (the plan sketched it under Task 3) because Task 1's reply-drain assertions require enqueue+drain; Task 3 added the two compare enqueue sites and pinned the spacing arithmetic"

patterns-established:
  - "Snapshot discipline: exactly one Records() per handler call shared by everything the event emits (handleDataRequest, handleDataSync, syncListing)"
  - "Graduated-length key construction for at-boundary chunk tests: estimates summing to exactly budget minus overhead close the chunk at exactly the budget (probe #15 made constructible)"
  - "floor(1s/rate) integer-nanosecond division is the only clock arithmetic in the limiter — deterministic under the virtual clock, echoed in IDEA.md §Parameters"

requirements-completed: [REPL-04, REPL-05]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "DataRequest reply and silence contracts: held keys (live and tombstoned) answered with one unicast exact-record DataReplication (Version, Data, Deleted, Checksum), unknown keys enqueue and emit nothing across Advances past the rate window, multiple requests share one queue in request order"
    requirement: REPL-04
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestDataRequest_replyAndSilence (4 rows, -race green)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Deterministic chunked listing: chunk closed at exactly the 1400-byte budget by estimate, boundary-crossing entry opens the next chunk, oversized single entry ships alone, every key in exactly one chunk, two fresh-snapshot builds identical, empty store emits one zero-entry listing"
    requirement: REPL-05
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_chunking (3 subtests, -race green)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Cross-seam wire budget: every budgeted chunk's real proto-encoded Envelope stays within sync_chunk_bytes against the plan-01 adapter codec; the oversized chunk equals its own unavoidable minimum (one datagram, never split)"
    requirement: REPL-05
    verification:
      - kind: unit
        ref: "internal/udprepl/syncsize_internal_test.go#TestSync_chunkWireBudget (white-box, -race green)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Sync triggers: startup fires Hello-then-DataSync on the first Advance (order pinned), silence across off-tick Advances, re-fire exactly at lastSync+sync_interval"
    requirement: REPL-05
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_triggers (-race green)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Per-key compare: missing and stale queue DataRequests, strictly-newer local record queues the exact-record unicast reply, equal versions silent, local tombstone newer than the listing's live entry travels as the reply"
    requirement: REPL-05
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_compare (-race green)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Rate limiter: first response immediate then exactly one per 2ms slot in listing order with zero-emission probes between slots, queue empty by enqueue+(K-1)*spacing, QueuedResponses seam, 300/s floor division pinned at exactly 3333333ns"
    requirement: REPL-05
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_rateLimit (2 subtests, -race green)"
        status: pass
    human_judgment: false

# Metrics
duration: 19 min
completed: 2026-11-06
status: complete
---

# Phase 4 Plan 03: Reconciliation — DataRequest Replies and Chunked DataSync Summary

**Reconciliation in both directions: unicast exact-record DataRequest replies (tombstones travel, unknown keys stay provably silent) plus deterministic chunked DataSync — sorted listings estimate-budgeted to sync_chunk_bytes, startup and timer triggers, per-key four-case compare, and one floor(⌊1s/rate⌋)-spaced egress queue carrying every reactive unicast.**

## Performance

- **Duration:** 19 min
- **Started:** 2026-11-06T18:33:24Z
- **Completed:** 2026-11-06T18:53:04Z
- **Tasks:** 3 (all TDD red-green)
- **Files modified:** 5 (1 core, 2 test files created, 1 prior-plan test updated, 1 doc line)

## Accomplishments
- REPL-04 proven: a DataRequest for a held key — live or tombstoned — is answered through the egress queue with one unicast DataReplication carrying the byte-equal snapshot record (Checksum included); an unknown key enqueues and emits nothing, ever, asserted across five Advances past the rate window
- REPL-05 emission proven: one sorted-by-key listing per Records() snapshot, chunked so the chunk filled to exactly 1400 estimate-bytes closes (graduated key lengths constructed to hit the boundary), the crossing entry opens the next chunk, an oversized single entry ships alone — self-sufficient chunks, no continuation state (A4) — and an empty store still broadcasts one zero-entry listing (probe #7)
- Startup and timer triggers: the first Advance fires Hello then the DataSync chunk sequence (IDEA §Node lifecycle order pinned); the timer re-fires exactly at lastSync+sync_interval, never between ticks; REPL-07 neighbor-restart targeting documented as plan 04-04's
- Cross-seam honesty proof: udprepl's white-box TestSync_chunkWireBudget encodes every emitted chunk through the real adapter codec — all budgeted chunks ≤ 1400 real bytes, the oversized chunk exactly its own unavoidable minimum
- REPL-05 receive proven: per-key compare against one snapshot — missing/stale queue a DataRequest, strictly-newer local record queues the exact-record reply, equal is silence, and a local tombstone newer than the listing's live entry heals the sender's delete
- One rate limiter over ALL reactive unicasts (exactly three enqueue sites, grep-verified): first response immediate, then exactly floor(1s/rate) integer nanoseconds apart (2ms at 500/s; 3333333ns pinned at 300/s), enqueue order IS drain order, QueuedResponses() exposed as the 04-05 quiescence seam, and the rounding rule echoed in docs/IDEA.md §Parameters (probe #16)

## Task Commits

Each task was committed atomically (TDD: RED test commit then GREEN feat commit per task):

1. **Task 1 RED: failing TestDataRequest_replyAndSilence + QueuedResponses compile stub** - `fc13369` (test)
2. **Task 1 GREEN: DataRequest reply path over the egress queue** - `6ce1697` (feat)
3. **Task 2 RED: failing chunking/trigger tests + the udprepl wire-budget test** - `10e313c` (test)
4. **Task 2 GREEN: deterministic listing, chunking, triggers + IDEA.md echo** - `ab88f34` (feat)
5. **Task 3 RED: failing compare and rate-limit tests** - `1150260` (test)
6. **Task 3 GREEN: DataSync compare enqueue sites** - `32b22ce` (feat)

**Plan metadata:** (this SUMMARY commit)

All three RED runs were machine-classified `RED_EVIDENCE_OK` (go-test adapter: each target failed on its planned assertions — zero replies queued, zero chunks emitted, zero compare responses queued; no build failures, no fixture faults). Each GREEN commit directly descends from its RED commit (merge-base verified). No REFACTOR commits: neither GREEN needed post-green cleanup.

## Files Created/Modified
- `internal/app/repl/core.go` - queuedUnicast egress queue + nextSendAt + floor(1s/rate) interval derivation, enqueueResponse/drainEgress, handleDataRequest, syncListing (startup + timer triggers), chunkSyncEntries with entryOverheadBytes/chunkOverheadBytes, handleDataSync compare, QueuedResponses(), nextSync and nextSendAt deadline contributions
- `internal/app/repl/sync_test.go` - TestDataRequest_replyAndSilence, TestSync_chunking, TestSync_triggers, TestSync_compare, TestSync_rateLimit (+ graduated-key and emission-filter helpers)
- `internal/udprepl/syncsize_internal_test.go` - TestSync_chunkWireBudget, white-box on purpose (the configured *_internal_test.go exemption, dumpfile_internal_test.go precedent)
- `internal/udprepl/repl_test.go` - startup assertion updated: one Hello followed by the startup DataSync (Rule 1 adjustment, see Deviations)
- `docs/IDEA.md` - §Parameters sync_response_rate item extended with the spacing/rounding rule (exactly one line)

## Decisions Made
- Egress-limiter scope pin implemented literally: one FIFO, exactly three enqueue sites (DataRequest emission, DataRequest reply, DataSync-newer response); proactive traffic (Hello, retransmits, timer/startup DataSync) never queues — the amplification bound T-04-07 depends on
- Chunk budget by conservative core-side estimate (len(key)+24 per entry, 64 per chunk) with the derivation and the real-datagram guarantee in the constants' doc comments; the udprepl cross-seam test is the honesty proof against the real codec
- The egress queue machinery landed in Task 1's GREEN (the plan sketched the queue under Task 3) because Task 1's reply-drain assertions already require enqueue+drain; Task 3 added the two compare enqueue sites and pinned the spacing arithmetic
- REPL-07 neighbor-restart targeting left to plan 04-04 via a doc pointer at syncListing (per the plan's explicit boundary)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Prior-plan test adjustment] TestWire_twoNodeHandoff startup assertion vs the designed startup DataSync**
- **Found during:** Task 2 (full-suite verification)
- **Issue:** Plan 04-01's tracer test asserted the first Advance drains exactly one outbound message (the Hello); Task 2's designed startup trigger now emits Hello plus the startup DataSync listing, so the tracer assertion failed against the new designed behavior.
- **Fix:** The startup assertion now pins exactly one broadcast Hello followed by the startup DataSync — the updated designed contract (the 04-02 eviction/ladder precedent: a prior plan's test yields to a later plan's designed behavior).
- **Files modified:** internal/udprepl/repl_test.go
- **Verification:** go test -race ./internal/udprepl/... green; make test green.
- **Committed in:** ab88f34

---

**Total deviations:** 1 auto-fixed (1 Rule 1)
**Impact on plan:** The fix updates a prior plan's test assertion to the startup contract this plan was mandated to introduce. No scope creep.

## Issues Encountered
- Lint-direction adjustments inside the TDD cycles (never config weakening): noinlineerr/unnecessary-format/unparam in the Task 1 RED test (a second requester peer now exercises requestFrom both ways in the silence row), and prealloc on chunkSyncEntries' current-chunk slice — solved with the tight budget/entryOverheadBytes capacity bound and a doc comment on why closed chunks never reuse a backing array.
- The plan's Task 2 "entry that would go one byte over" boundary is exercised by construction: the first chunk's estimates sum to exactly 1400 (8 graduated lengths, 64 + 1336), so any following entry crosses — the at-budget and crossing cases are both pinned.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Ready for 04-04 (restart-targeted sync): the listing/compare machinery and the doc pointer at syncListing mark the seam; a Hello with a new run_id adds one targeted-listing trigger
- Ready for 04-05 (simulator): Advance's deadline now aggregates hello + sync + peer-expiry + retransmit + egress-slot contributions (all strictly future), and QueuedResponses() is the quiescence check the heal-bound tests read
- D-01/D-02/D-04/D-05 intact: protobuf stays adapter-side (estimate constants instead), every discard/compare/scheduling decision lives in the core, zero goroutines, zero locks, zero wall-clock reads inside internal/app/repl (grep-verified)
- go.mod/go.sum untouched (T-04-SC); full make lint + make test green (CI parity)

## Self-Check: PASSED

Key files exist on disk (core.go, sync_test.go, syncsize_internal_test.go, repl_test.go, IDEA.md); the six task commits fc13369..32b22ce are ancestors of HEAD on this branch (evaluation-scope resolved 6 commits for plan 04-03); all plan verification bullets re-run green under -race (both trees); full-repo make lint and make test green (CI parity); purity greps clean (no sync./go func/time.Now/time.Sleep in internal/app/repl); docs/IDEA.md grep gate = 1; TDD gate commits present with RED-before-GREEN ancestry for all three tasks.

---
*Phase: 04-replication-and-deterministic-simulator*
*Completed: 2026-11-06*
