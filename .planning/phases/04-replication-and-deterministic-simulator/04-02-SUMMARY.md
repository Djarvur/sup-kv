---
phase: 04-replication-and-deterministic-simulator
plan: 02
subsystem: replication
tags: [udp, replication, retransmission, exponential-backoff, peer-table, crdt, event-loop, go]

# Dependency graph
requires:
  - phase: 04-replication-and-deterministic-simulator/plan 01
    provides: the repl core skeleton — pending entry shell in NotifyLocalWrite, the accept-and-ignore Ack branch, PeerSnapshot and the Advance hello timer this plan expands
provides:
  - per-peer (key, version) pending tracking — wait-sets seeded from every known peer, drain on exact-version Ack from the last waiter (non-empty table required), local and remote supersession (REPL-02)
  - the retransmission backoff ladder — 200/400/800/1600/3200ms then 5000ms repeated, saturate-never-wrap, documented step-by-step at scanRetransmit; retransmit_interval as Advance's scan-granularity ceiling (A5 reading)
  - suspension (empty peer table freezes ladders and retains entries) and flush-on-discovery (the Hello ending an empty table broadcasts every pending entry, gains the waiter, restarts ladders — A6 reading)
  - peer-table lifecycle — at-or-after lastHello+peer_timeout eviction removing the evictee from every wait-set without draining entries, re-learning with no blacklist, N peers from day one (REPL-06, probe #18 both sides)
  - observability seam — PendingSnapshot (sorted keys, sorted WaitingOn, NextRetry) and UnackedStats (count + oldest age) for Phase 5 wait_for_replication and Phase 6 OBS-01
affects: [04-03 DataRequest replies on the same pending store, 04-04 restart sync, 04-05 simulator drives Advance's deadline contract and the ladder bounds, 04-06 classification totality, phase 05 gRPC wait_for_replication, phase 06 OBS-01 metrics]

# Actuals (#2632) — pairs with the plan's `estimate` (estimateTokens scale: chars/4 over the realized diff).
actuals:
  tokens: 12219
  tasks: 3
  commits: 5

plan_head_before: fc314ad408a52f36ff40c4f95cf615d94e6c74d0
plan_head_after: 81d460b4bc63d67eac3609a0a6c1f3aaebc0292f

# Tech tracking
tech-stack:
  added: []
  patterns: [per-peer wait-set membership on a per-key pending entry with drain = empty wait-set AND non-empty table, saturating exponential backoff documented in the firing scan's doc comment (clock.go precedent), eviction-before-scan ordering inside Advance so an expiring sole peer suppresses the same-tick send, deadline minimum over armed timers with a zero-time guard for absent contributions]

key-files:
  created:
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/peertable_test.go
  modified:
    - internal/app/repl/core.go

key-decisions:
  - "A5 scan-granularity reading pinned in code: retransmit_interval bounds Advance's returned sleep while any entry is pending; the per-key ladder doubles from ack_timeout and saturates at ack_timeout_max with every step spelled out at scanRetransmit's doc comment"
  - "Suspension is total — an empty peer table freezes nextRetry/backoff, never advancing them; during suspension the retransmit deadline keeps only the now+retransmit_interval ceiling so a stale past retry time never leaks into the deadline"
  - "Flush-on-discovery fires only on the empty-to-non-empty table transition; a Hello into a non-empty table (including a re-learned evicted peer) adds no waiters — re-learned peers regain coverage through the ladder's next fire"
  - "UnackedStats takes now as a parameter (the D-02 seam) rather than reading a clock — a pure read inside the core"
  - "Eviction runs before the retransmit scan inside Advance, so a sole peer expiring at now suppresses that same tick's send (suspension starts at the eviction event itself)"

patterns-established:
  - "Deadline contract: Advance returns the minimum over the armed timers (hello, earliest peer expiry, retransmit scan ceiling clamped by the earliest pending retry) and is always strictly future — due work fires inside the call"
  - "Sorted emission everywhere the core produces output: every multi-datagram path (retransmit scan, discovery flush) iterates keys via slices.Sorted(maps.Keys(...)) — probe #9 discipline"

requirements-completed: [REPL-02, REPL-06]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Pending-set semantics: wait-sets seeded from every known peer, exact-(key,version) Ack drain requiring an empty wait-set AND a non-empty table, local-write replacement and strictly-greater remote supersession, stranger/stale/out-of-order Ack immunity, PendingSnapshot/UnackedStats observability seam"
    requirement: REPL-02
    verification:
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestRetransmit_ackDrainSemantics (5 table rows, -race green)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Backoff ladder with saturation (exact cumulative steps 200/600/1400/3000/6200ms then every 5000ms, off-ladder probes silent), sorted multi-key emissions, suspension retaining entries and NextRetry over a 60-virtual-second empty-table window, flush-on-discovery restarting ladders at ack_timeout, strictly-future Advance deadlines"
    requirement: REPL-02
    verification:
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestRetransmit_backoffLadder + #TestRetransmit_suspendAndFlush (-race green)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Peer-table lifecycle: at-or-after lastHello+peer_timeout eviction asserted on both sides of the boundary (probe #18), eviction cleaning every wait-set without draining entries, late-Ack no-op, re-learning without blacklist, five-peer scrambled-order Ack tracking, never-known-peer and sole-peer-evicted-then-returning suspension paths"
    requirement: REPL-06
    verification:
      - kind: unit
        ref: "internal/app/repl/peertable_test.go#TestPeerTable_lifecycle + #TestPeerTable_nPeers + #TestPeerTable_solePeer (-race green)"
        status: pass
    human_judgment: false

# Metrics
duration: 19 min
completed: 2026-11-06
status: complete
---

# Phase 4 Plan 02: Replication Reliability Layer Summary

**Per-peer (key, version) ack tracking on a documented 200ms-to-5s saturation retransmission ladder, with suspension while no peer is known, flush-on-discovery, and the N-peer table's peer_timeout eviction (REPL-02, REPL-06).**

## Performance

- **Duration:** 19 min
- **Started:** 2026-11-06T18:03:11Z
- **Completed:** 2026-11-06T18:22:01Z
- **Tasks:** 3 (Tasks 1-2 TDD red-green, Task 3 auto)
- **Files modified:** 3 (1 modified, 2 created)

## Accomplishments
- Pending-set semantics (REPL-02): every local write tracked per known peer with the exact version; an Ack drains exactly its waiter on an exact match and the entry leaves only on full coverage or supersession — stale, stranger and out-of-order Acks change nothing (probes #2, #10)
- The retransmission ladder: exponential backoff 200/400/800/1600/3200ms saturating at 5000ms repeated forever, every step documented at the firing scan; retransmit_interval pinned as Advance's scan-granularity ceiling (research A5); emissions always sorted by key (probe #9)
- Suspension and flush-on-discovery: an empty peer table freezes every ladder and retains every entry; the first Hello that ends the empty table broadcasts the whole pending set in that same event, gains the waiter and restarts ladders (research A6)
- Peer-table lifecycle (REPL-06): eviction at the first Advance at or after lastHello+peer_timeout (probe #18 asserted one tick before AND exactly at the boundary), wait-set cleanup without draining, re-learning with no blacklist, five-peer independent tracking, and both single-node suspension paths
- The observability seam: PendingSnapshot (sorted keys, sorted WaitingOn, NextRetry) and UnackedStats (count, oldest age) — what Phase 5's wait_for_replication and Phase 6's OBS-01 read

## Task Commits

Each task was committed atomically (TDD: RED test commit then GREEN feat commit; Task 3 is a single feat commit per its type):

1. **Task 1 RED: failing TestRetransmit_ackDrainSemantics + compile stubs** - `1d704b2` (test)
2. **Task 1 GREEN: pending-set ack drain and supersession** - `64d1000` (feat)
3. **Task 2 RED: failing ladder and suspend/flush tests** - `27e70a1` (test)
4. **Task 2 GREEN: backoff ladder, suspension and flush-on-discovery** - `d3868f6` (feat)
5. **Task 3: peer table timeout eviction with wait-set cleanup** - `81d460b` (feat)

**Plan metadata:** (this SUMMARY commit)

Both RED runs were machine-classified `RED_EVIDENCE_OK` (go-test adapter: target test failed on the planned assertions — no build failures, no fixture faults); both GREEN commits descend from their RED commits (ancestry verified). No REFACTOR commits: the GREEN implementations needed no post-green cleanup.

## Files Created/Modified
- `internal/app/repl/core.go` - pendingEntry (version, waitSet, nextRetry, backoff, createdAt), handleAck drain, applyRemote supersession, scanRetransmit with the documented ladder, handleHello flush-on-discovery, evictPeers, nextDeadline/retransmitDeadline/earliestRetry/earliestPeerExpiry, PendingSnapshot/UnackedStats
- `internal/app/repl/retransmit_test.go` - TestRetransmit_ackDrainSemantics (5 rows), TestRetransmit_backoffLadder (2 subtests), TestRetransmit_suspendAndFlush, black-box harness (hand-built port datagrams, no adapter)
- `internal/app/repl/peertable_test.go` - TestPeerTable_lifecycle, TestPeerTable_nPeers, TestPeerTable_solePeer

## Decisions Made
- A5 reading pinned in code: scan-granularity ceiling on Advance's deadline, per-key ladder keyed on ack_timeout doubling to ack_timeout_max — documented with every step at scanRetransmit's doc comment
- Suspension freezes ladder state instead of advancing it; during suspension only the now+retransmit_interval ceiling contributes to the deadline (a stale past retry time must never leak into a strictly-future deadline)
- Flush-on-discovery only on the empty-to-non-empty transition; ordinary (non-discovery) Hellos — including a re-learned evicted peer's — modify no wait-set
- UnackedStats takes now as a parameter (D-02 seam) — the plan sketched a no-arg accessor, but the core reads no clock
- Eviction runs before the retransmit scan inside Advance so an expiring sole peer suppresses that same tick's send

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Zero-time peer expiry leaked into Advance's deadline**
- **Found during:** Task 3 (full-suite verification)
- **Issue:** earliestPeerExpiry returns the zero time for an empty table; `zero.Before(anything)` is true, so with no peers Advance returned a year-1 deadline — violating the strictly-future contract the Task 2 tests assert.
- **Fix:** Zero-guard in nextDeadline (`!expiry.IsZero() && expiry.Before(deadline)`).
- **Files modified:** internal/app/repl/core.go
- **Verification:** TestRetransmit_suspendAndFlush and TestRetransmit_backoffLadder futurity assertions green; full -race suite green.
- **Committed in:** 81d460b

**2. [Rule 1 - Test adjustment] Ladder test needed the peer's periodic re-Hellos**
- **Found during:** Task 3 (full-suite verification)
- **Issue:** The plan's Task 2 ladder test drives 11.2 virtual seconds against a peer Hello'd once at t0; once Task 3's peer-timeout scan existed, that peer is legitimately evicted at t0+5s and the ladder suspends mid-test.
- **Fix:** The test now delivers the peer's 1-second re-Hellos ahead of each probe — the wire behavior whose liveness the scan enforces.
- **Files modified:** internal/app/repl/retransmit_test.go
- **Verification:** TestRetransmit_backoffLadder green with all six ladder steps and both off-ladder probes intact.
- **Committed in:** 81d460b

---

**Total deviations:** 2 auto-fixed (2 Rule 1)
**Impact on plan:** Both fixes preserve the plan's semantics — one is a production regression lock on the deadline contract, the other keeps the plan's own ladder test valid once eviction exists. No scope creep.

## Issues Encountered
- Test-harness slips caught inside the TDD cycles, fixed before their GREEN commits: duplicate t.Parallel in table rows (panicked), and the out-of-order subtest re-asserting already-drained keys (now tracks the remaining set). Neither reached a commit's verify step un-green.
- Lint-direction adjustments to the RED-phase test (noinlineerr, prealloc, thelper, unparam on newTestCore) — code-direction fixes per the Phase 01 decision, never config weakening.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The pending set, ladder, suspension, flush and eviction semantics are in place for 04-03 (DataRequest replies consult the same store/pending state), 04-04 (restart sync), 04-05 (the simulator drives Advance's deadline contract directly — the strictly-future invariant is asserted) and 04-06 (classification totality, ApplyRemote seam, smoke test)
- Advance's returned deadline now aggregates hello + peer-expiry + retransmit contributions; later plans' timers (sync, rate-limited egress) join the same minimum
- D-04/D-05 intact: zero goroutines, zero locks, zero wall-clock reads inside internal/app/repl (grep-verified: sync./go func/time.Now/time.Sleep = 0)

## Self-Check: PASSED

Key files exist on disk (core.go, retransmit_test.go, peertable_test.go); the five task commits 1d704b2..81d460b are ancestors of HEAD on this branch (evaluation-scope resolved 5 commits for plan 04-02); all plan verification bullets re-run green under -race; full-repo `golangci-lint run ./...` and `go test -race -cover ./...` green (CI parity); purity greps clean; TDD gate commits present with RED-before-GREEN ancestry for both TDD tasks.

---
*Phase: 04-replication-and-deterministic-simulator*
*Completed: 2026-11-06*
