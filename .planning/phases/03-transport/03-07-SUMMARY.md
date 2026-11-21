---
phase: 03-transport
plan: 07
subsystem: repl-transport-pacing
tags: [listing-pacer, chunk-rotation, record-send-queue, deterministic-jitter, bounded-egress, xport-01, xport-02, xport-07, xport-09]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-05's record-free (key, version) ledger, indexed due-heap, per-peer unacked index and O(1) gauges the record pacer drives"
  - phase: 03-transport
    provides: "03-06's Outbox result stream and adapter trackers, which the paced sends feed unchanged"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the red tests R-01/R-02 and the chaos/quiescence/bound suites this plan must keep green without editing the reds"
provides:
  - "one in-flight listing per destination: a FIFO keyed by destination (broadcast + per-peer handle), drained at floor(1s/SyncResponseRate) with a busy trigger dropped while nextSync still advances"
  - "broadcast listing rotation by a monotone round counter and targeted restart rotation by a per-peer counter kept in the peer row"
  - "a deduplicated (key, version) record-send queue fed by the discovery flush and the retransmit scan, with the ladder re-armed at emission only"
  - "deterministic jitter from (runID, counter, salt): positive-only on Hello/sync deadlines, +-10% on listing/record pacer spacing, never on ladder deadlines"
  - "a bounded (32 768) egress queue deduplicated by (handle, kind, key), drop-oldest with lockstep identity deletion and an EgressDropped counter"
  - "QueuedSends total-queue accessor and a chaos quiescence predicate that requires every queue empty"

actuals:
  tokens: 22192
  tasks: 3
  commits: 5
plan_head_before: 1a71117fd7fc7edcb8c5229c44218eb70a4d6388
plan_head_after: ce45f7f6187e2a74783d4cd09910ca1414208f06

tech-stack:
  added: []
  patterns:
    - "Three independent pacers at one normalized interval: listing, record and egress each keep their own next-slot and nextDeadline takes the minimum; the egress spacing stays exact while the two whole-set streams carry the +-10% jitter."
    - "Pacer drain clamp: a late Advance emits one datagram and measures the next slot from the actual emission time, so no backlog ever bursts."
    - "Destination-keyed listing FIFOs with an ordered destination list give a deterministic round-robin drain and an in-flight mark that clears exactly when the FIFO empties."
    - "One queue for the two whole-set emission paths (discovery flush and retransmits), deduplicated by pendingKey; the author's first broadcast stays immediate by design."
    - "armRetry chooses push-or-fix from the heap index, so a retry (popped by the scan) re-pushes at emission while a flush send (still indexed) is only re-timed."
    - "Deterministic jitter is FNV-1a over (runID, seq, salt) with per-site monotone counters: replay-identical, no PRNG state, and the ladder never sees it."

key-files:
  created:
    - internal/app/repl/egress_internal_test.go
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/sync_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/restart_test.go
    - internal/app/repl/discard_test.go
    - internal/app/repl/peertable_test.go
    - internal/app/repl/sim_chaos_test.go
    - internal/app/repl/sim_scenario_test.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/syncsize_internal_test.go

key-decisions:
  - "The broadcast rotation counter is core-wide and increments once per enqueued round; the targeted rotation lives in the peer row and survives a run_id change because handleHello updates the row in place instead of replacing it."
  - "A destination is in flight exactly while its FIFO map entry exists, so the one-in-flight rule needs no second mark and the drain deletes the entry when the FIFO empties."
  - "Record-send dedupe is the exact pendingKey (key, version); a flush send and a due retry for the same pair collapse into one datagram, and a skipped send consumes no pacer slot."
  - "The discovery flush enqueues instead of broadcasting; it resets backoff and sets a placeholder next retry so the deadline stays future, and the real ladder arming happens at emission (never enqueue)."
  - "A flush send whose stored version is missing or mismatched is skipped but leaves the entry tracked; only the retry path deletes it — preserving the 03-05/03-06 synthetic-notify mirror contract while stale cleanup still happens at the next due time."
  - "Jitter is positive-only on the Hello and sync deadlines (never earlier than the configured interval) and +-10% on listing/record spacing; the egress spacing stays exact and no ladder deadline is jittered (the loss <= ack_timeout_max bound depends on it)."
  - "The egress identity is (handle, kind, key) with the version deliberately excluded: a DataRequest carries none and a DataReplication answer re-reads the store at drain, so the queue pins no Data."
  - "QueuedResponses keeps its egress-only meaning; QueuedSends is the total-queue accessor quiescence now consults."
  - "The empty-disk scenario's bound includes the sync-jitter envelope because each node's periodic timer carries its own deterministic jitter."
  - "partitionDays asserts the conflicts counter covers every displaced version at least once rather than in both directions: the paced flush can deliver the winner before the loser leaves, and a superseded pending send is skipped by design."

patterns-established:
  - "Pacing is structural, not incidental: every whole-set emission path must pass a queue + pacer, and the tests measure the bound rather than a single tick's output."
  - "Jitter sites take their own monotone counter, so a site's stream is stable under changes elsewhere and same-seed replays stay byte-identical."
  - "Queue caps and their dedupe maps mutate in one function (dropOldestEgress) so eviction can never starve a later identical request."
  - "Test drains for paced queues advance to the core's returned deadline, so they measure the real spacing instead of guessing it."

requirements-completed: [XPORT-01, XPORT-02, XPORT-07, XPORT-09]

coverage:
  - id: D1
    description: "DataSync listings are enqueued and drained at one datagram per normalized interval; a busy trigger is dropped while nextSync still advances, and an empty store still emits one empty listing"
    requirement: XPORT-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestSync_'"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_pacing + TestSync_listingBusy + TestSync_emptyListing"
        status: pass
    human_judgment: false
  - id: D2
    description: "The broadcast round rotates its start chunk by a monotone counter and a targeted restart listing rotates a per-peer counter kept in the peer row; repeated rounds cover the same key multiset"
    requirement: XPORT-01
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_rotation"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestRestart_'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The discovery flush and retransmits share one (key, version)-deduplicated queue paced at the normalized interval; the ladder is armed from the emission time, and a paced 1 000-entry flush stays inside window/interval + 1"
    requirement: XPORT-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestRetransmit_|^TestBound_|^TestPending_'"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestRetransmit_ladderFromEmission + TestRetransmit_pacedStormBound + TestRetransmit_skipDrainedAndSuperseded"
        status: pass
    human_judgment: false
  - id: D4
    description: "Jitter is deterministic from (runID, counter, salt), positive-only on Hello/sync deadlines, +-10% on pacer spacing, and absent from every ladder deadline; same-seed replay stays byte-identical"
    requirement: XPORT-09
    verification:
      - kind: other
        ref: "grep call sites: jitteredSpacing only in drainListings/drainRecordSends; helloJitter/syncJitter only in Advance/syncListing; no ladder site"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestChaos_determinismReplay$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "The egress queue deduplicates by (handle, kind, key) — two listings of the same 1 000 missing keys leave 1 000 answers — is capped at 32 768 with drop-oldest plus identity deletion, and skips vanished records at drain"
    requirement: XPORT-07
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestEgress_'"
        status: pass
      - kind: unit
        ref: "internal/app/repl/egress_internal_test.go#TestEgress_boundedDeduped + TestEgress_identitySeparates + TestEgress_capDropsOldest + TestEgress_drainSkipsGoneRecords"
        status: pass
    human_judgment: false
  - id: D6
    description: "The simulator's quiescence predicate requires every core queue empty (listing, record, egress), so a scenario can never declare quiescence with an unsent datagram"
    requirement: XPORT-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/app/repl/ (chaos, scenario and bound suites)"
        status: pass
      - kind: other
        ref: "internal/app/repl/sim_chaos_test.go#chaosQuiescent checks UnackedStats, QueuedResponses and QueuedSends"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 7: Paced Listings, Record-Send Queue and Bounded Egress Summary

**Three queues sharing one normalized pacing interval — rotating listings, a deduplicated record-send queue armed at emission, and a bounded deduplicated answer queue — replace whole-set bursts, with deterministic jitter that never touches the retransmit ladder.**

## Performance

- **Duration:** 20 min (1212 s)
- **Started:** 2026-11-21T20:05:29Z
- **Completed:** 2026-11-21T20:25:41Z
- **Tasks:** 3
- **Files modified:** 11 (1 created, 10 modified)

## Accomplishments

- `syncListing` now builds chunks from one `Versions()` view, rotates the start by `syncRotation % len(chunks)`, enqueues them in a destination-keyed FIFO (broadcast = `""`) and lets `drainListings` emit one datagram per normalized interval. A busy trigger returns after advancing `nextSync`, so the stand's short interval can never livelock.
- The restart branch of `handleHello` enqueues the same rotated listing toward the restarted peer, rotating by a per-peer counter that survives the in-place row update; a handle with a listing in flight is left alone.
- `recordQueue []recordSend` with a `recordQueued map[pendingKey]struct{}` dedupe set is fed by the discovery flush and by `scanRetransmit`; `drainRecordSends` resolves each record with `Lookup` at emission, counts only retries, and re-arms the ladder from the emission time (`armRetry` pushes a popped retry item and fixes an indexed flush item).
- The discovery flush no longer broadcasts; the author's first `NotifyLocalWrite` broadcast stays immediate, as the XPORT-02(3) burst acceptance requires.
- Deterministic jitter: `jitterOffset` is FNV-1a over `(runID, seq, salt)`; `helloJitter`/`syncJitter` are positive-only tenths of their intervals, `jitteredSpacing` is ±10% with a 1 ns floor. The egress spacing is exact and no ladder deadline passes through the helper.
- The egress queue is capped at `maxEgressEntries = 32768` and deduplicated by `(handle, kind, key)`; overflow drops the oldest and deletes its identity in `dropOldestEgress`, counting `egressDropped`; `drainEgress` re-reads `DataReplication` records and skips vanished keys.
- `QueuedSends()` totals the listing, record and egress queues; `chaosQuiescent` now requires it to be zero alongside the unacked gauge and `QueuedResponses`.
- The plan's verification set is green: four-package `-race` run (repl, simnet, udprepl), the focused sync/retransmit/egress run, `go tool golangci-lint run ./...`, the red-test byte check, and the empty `math/rand` grep. The full repository `go test -race ./...` also passes.

## Task Commits

Each task was committed atomically, with one documented boundary move:

1. **Tasks 1+2: Paced rotating listings and the record-send queue with emission-time ladder arming** — `fd6114b` (feat)
2. **Task 3: Bounded, deduplicated sync-answer egress queue** — `4c6debb` (feat)
3. **udprepl wire-test adaptation** — `024012f` (test)
4. **Strict-lint compliance** — `698241f` (style)
5. **Jitter comment clarification** — `ce45f7f` (docs)

**Plan metadata:** see the `docs(03-07)` commit that follows this summary.

_Note: no TDD tasks in this plan; the boundary move and the extra adaptation commits are documented under Deviations._

## Files Created/Modified

- `internal/app/repl/core.go` — `listingFIFO`, `recordSend`, `egressKey`/`egressKind`, the `peer.rotation` field, the listing/record/egress queue fields and their jitter counters, `syncListing`/`enqueueRestartListing`/`enqueueListingChunk`/`drainListings`, `enqueueRecordSend`/`drainRecordSends`/`emitRecordSend`/`armRetry`, `jitterOffset`/`helloJitter`/`syncJitter`/`jitteredSpacing`, the bounded `enqueueResponse`/`dropOldestEgress`/`drainEgress`/`resolveQueuedResponse`, `QueuedSends`/`EgressDropped`, and the updated `Advance`/`nextDeadline`/`handleHello` wiring.
- `internal/app/repl/egress_internal_test.go` — new white-box suite with a mutable fake `StateStore`: 1 000 duplicate answers collapse, identities separate by handle and kind, the cap evicts and re-admits the same identity, and a vanished record drains silently.
- `internal/app/repl/sync_test.go` — `advanceDrain`/`oversizedKeyN`/`syncEntryKeySequence` helpers; `TestSync_chunking` and `TestSync_triggers` adapted to the paced drain and the jitter envelope; new `TestSync_pacing`, `TestSync_rotation`, `TestSync_listingBusy`, `TestSync_emptyListing`.
- `internal/app/repl/retransmit_test.go` — ladder tests drive the paced drain; new `TestRetransmit_ladderFromEmission`, `TestRetransmit_pacedStormBound`, `TestRetransmit_skipDrainedAndSuperseded`.
- `internal/app/repl/restart_test.go` — `freshListing` drains the startup round; `rotateChunks`; restart rounds assert the per-peer rotation.
- `internal/app/repl/discard_test.go` — the restart row now expects zero handler emissions and one queued listing chunk.
- `internal/app/repl/peertable_test.go` — the returning-Hello flush is drained by an Advance.
- `internal/app/repl/sim_chaos_test.go` — `chaosQuiescent` requires `QueuedSends() == 0`.
- `internal/app/repl/sim_scenario_test.go` — the empty-disk bound adds the sync-jitter envelope; the partition-days conflict assertion became the honest one-sided lower bound.
- `internal/udprepl/repl_test.go` — the startup Hello deadline asserts the jitter envelope.
- `internal/udprepl/syncsize_internal_test.go` — the three-chunk wire-budget round is driven across Advances.

## Decisions Made

- The broadcast rotation counter is core-wide (one increment per enqueued round); the targeted rotation lives in the peer row and is preserved across a run_id change because `handleHello` updates the row in place.
- A destination is in flight exactly while its FIFO map entry exists; no separate mark can drift from the queue.
- Record-send dedupe is the exact `pendingKey`; a skipped send consumes no pacer slot, so the drain stays truthful without spending rate on silence.
- The discovery flush sets a placeholder retry deadline (keeping `nextDeadline` future) and the real ladder arming happens at emission; `armRetry` distinguishes a popped retry (push) from an indexed flush send (fix).
- A flush send whose record is absent leaves the entry tracked (the 03-05/03-06 synthetic-notify contract); only the retry path deletes stale entries.
- Jitter is positive-only on Hello/sync deadlines, ±10% on listing/record spacing, exact on egress, and never on the ladder.
- The egress identity is `(handle, kind, key)`; versions stay out and `DataReplication` answers re-read the record at drain.
- `QueuedResponses` keeps its egress-only meaning; `QueuedSends` is the total-queue seam.
- The empty-disk scenario bound includes `sync_interval/10`; the partition-days conflict evidence asserts `>= sharedKeys` because the paced flush can supersede the loser before it crosses the wire.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Commit boundary] Tasks 1 and 2 landed as one commit**
- **Found during:** commit staging
- **Issue:** The three pacers share one `Core` struct, one `Advance` ordering, one `nextDeadline` minimum and one jitter-helper set; the listing half already touches the flush path and the tests' `advanceDrain` needs the total-queue accessor. Splitting would require hand-reconstructing an intermediate `core.go` that never existed and would leave `go build ./...` broken at the boundary.
- **Fix:** Committed the listing and record pacing together as `fd6114b`; Task 3's egress work is a separate commit.
- **Files modified:** the Tasks 1/2 file set
- **Verification:** each committed tree builds and passes its suites; the Task 1+2 tree was tested with the old egress implementation before the egress commit.
- **Committed in:** `fd6114b`

**2. [Rule 3 - Plan/test conflict] The queued restart/flush listings required more test adaptations than the plan listed**
- **Found during:** Task 1/2 test runs
- **Issue:** The plan's `files_modified` named `sync_test.go` and `retransmit_test.go`, but the restart listing and the discovery flush now enqueue instead of emitting inside the handler, so `restart_test.go`, `discard_test.go`, `peertable_test.go` and the udprepl wire tests asserted emissions that no longer happen there.
- **Fix:** Drained the paced queues in those tests with `advanceDrain`/explicit Advances, kept every semantic assertion (names unchanged), and added `wantQueued` to the discard table so the restart reaction remains observable.
- **Files modified:** `restart_test.go`, `discard_test.go`, `peertable_test.go`, `internal/udprepl/repl_test.go`
- **Verification:** the focused and full `-race` sets are green.
- **Committed in:** `fd6114b`, `024012f`

**3. [Rule 3 - Plan/test conflict] The Hello deadline assertion became an envelope assertion**
- **Found during:** udprepl wire tests
- **Issue:** `assertStartupHello` required the next Hello deadline to equal `now + hello_interval` exactly; the required positive-only jitter moves it into `[interval, interval + 10%)`.
- **Fix:** Asserted never-before and strictly-inside-the-envelope instead of equality.
- **Files modified:** `internal/udprepl/repl_test.go`
- **Verification:** `go test -race ./internal/udprepl/` green.
- **Committed in:** `024012f`

**4. [Rule 3 - Plan/test conflict] `TestScenario_emptyDisk`'s bound needed the jitter envelope**
- **Found during:** full repl suite
- **Issue:** The joiner recovers from the peers' next periodic listing, which now fires up to `sync_interval/10` later than the nominal tick; the old bound expired at exactly `sync_interval + 2N*pace`.
- **Fix:** Added `sync_interval/10` to the bound and documented the second jitter term (the two peers' timers may also fire apart).
- **Files modified:** `internal/app/repl/sim_scenario_test.go`
- **Verification:** the fixed seed and fresh seeds converge inside the new bound.
- **Committed in:** `fd6114b`

**5. [Rule 3 - Plan/test conflict] `TestScenario_partitionDays`' conflict lower bound became one-sided**
- **Found during:** full repl suite (seed 1795291793561864750)
- **Issue:** The paced discovery flush can deliver the winner before the loser's flush send leaves; the loser is then superseded out of tracking and skipped at drain, so the winner's node never observes the losing version. The old `>= 2*sharedKeys` assumed both directions always cross.
- **Fix:** Asserted `>= sharedKeys` — every displaced version is still counted at least once on the side that applied the winner — with the mechanism documented in the test; the deterministic-winner and convergence assertions are unchanged.
- **Files modified:** `internal/app/repl/sim_scenario_test.go`
- **Verification:** replay of the failing seed and fresh seeds pass.
- **Committed in:** `fd6114b`

**6. [Rule 2 - Lint compliance] Three bounded conversions need justified suppressions**
- **Found during:** `make lint`
- **Issue:** `gosec` G115 flags `int(uint64 % len(chunks))` and `time.Duration(uint64 % uint64(span))`, although the modulus bounds the value to a valid `int`/`Duration`.
- **Fix:** Three specific `//nolint:gosec` directives with written reasons (AGENTS.md requires linter name plus explanation); `mnd` was resolved by naming `jitterDivisor`.
- **Files modified:** `internal/app/repl/core.go`
- **Verification:** `go tool golangci-lint run ./...` green.
- **Committed in:** `698241f`

---

**Total deviations:** 6 auto-fixed (1 commit-boundary move, 4 test adaptations the plan's file list did not anticipate, 1 lint compliance)
**Impact on plan:** No scope change. Every must-have and acceptance criterion is met; the red tests stay byte-untouched and the TSV rows stay `red` for 03-12's flip.

## Issues Encountered

- `TestCore_keyCountCostFlat`'s fixtures leave the startup listing partially queued after their warm-up Advance; the measured Advance then drains at most one 64-byte-free chunk per iteration. Both arms amortize it and the 2× gate stays green, so the benchmark was left untouched.
- The partition-days conflict asymmetry was the only behavioral surprise: it is the direct consequence of the mandated flush pacing plus per-node Hello jitter, not a bug, and is now pinned honestly.
- The udprepl wire tests needed a paced drain before encoding their three chunks; the cross-seam size proof itself is unchanged.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 03-08 (wire validation) can build on the paced paths without touching them: the listing and record queues are internal to the core and the red tests are untouched.
- 03-10 owns XPORT-09's remaining halves (bind-address WARN, duplicate-instance failure); the timer-jitter half landed here.
- 03-12 can retarget the red-runner mutants: the periodic-sync sed still matches the `syncListing` timer gate, and the tombstone mutant belongs at `store.Versions` (RESEARCH Pitfall 1); the R-01/R-02 rows remain `red`.
- The phase acceptance mechanisms are in place: R-01's paced rotating listing and R-02's paced, deduplicated draining both exist and their bounds are unit-pinned; the stand red tests measure them at scale once the rows flip.

## Self-Check: PASSED

- All 11 plan files exist on disk; `egress_internal_test.go` is the only created file.
- Measured commits from the persisted plan ledger: `git rev-list --count 1a71117..ce45f7f` = 5; `plan_head_before` = `1a71117fd7fc7edcb8c5229c44218eb70a4d6388`, `plan_head_after` = `ce45f7f6187e2a74783d4cd09910ca1414208f06`.
- The `evaluation-scope` query verb is unavailable in this runtime (`gsd_run` is not installed), so the close-out uses the persisted plan ledger and `git rev-list --count` as the documented equivalent measurement.
- Plan verification re-run at close-out: `go test -race -count=1 ./internal/app/repl/ ./internal/app/internal/simnet/ ./internal/udprepl/` green; the focused `TestSync_|TestRetransmit_|TestEgress_` run green; `git diff --exit-code internal/stand/stand_red_test.go` clean; `go tool golangci-lint run ./internal/app/repl/` green and the full-repo lint green; the `math/rand` grep over `core.go` empty.
- Acceptance re-check: listings drain one datagram per interval inside the ±10% jitter envelope; broadcast and per-peer rotations are independent and deterministic; a busy trigger advances `nextSync` with a future deadline; the discovery flush and retransmits share one deduplicated queue armed at emission; a 1 000-entry flush stays within `window/interval + 1`; the egress queue leaves 1 000 answers for 1 000 duplicated keys, evicts and re-admits the same identity, and drains silently for a vanished record; quiescence requires every queue empty.
- Full repository `go test -race ./...` green, including the chaos determinism replay and both flatness gates.
- REQUIREMENTS.md boxes for XPORT-01/XPORT-02/XPORT-07/XPORT-09 are deliberately left for 03-12's XPORT-01..13 gate (03-10 still owns XPORT-09's bind/duplicate halves); the SUMMARY records the plan's four requirement IDs verbatim.

---
*Phase: 03-transport*
*Completed: 2026-11-21*
