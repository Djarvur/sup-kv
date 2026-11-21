---
phase: 03-transport
plan: 05
subsystem: repl-core
tags: [unacked-ledger, per-peer-index, indexed-heap, o1-gauges, event-loop-flatness, xport-02]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-02's tombstone-visible Store.Version/Lookup/Versions view"
  - phase: 03-transport
    provides: "03-04's four-method repl.StateStore and the benchmark-ratio gate pattern"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the retransmit ladder, eviction, sync and chaos suites that pin the behavior this plan preserves"
provides:
  - "pending keyed by (key, version) and retaining no record Data; every emission re-reads the store through Lookup and skips a version mismatch"
  - "peerUnacked per-peer index: eviction is O(that peer's unacked), never a whole-pending scan"
  - "indexed (nextRetry, seq, pendingKey) due-heap with an external position map: due-only Advance, O(log n) re-arm/deletion, no stale items"
  - "O(1) UnackedStats backed by a createdAt-ordered list and O(1) PeerCount; the adapter mirror no longer allocates a peer map per event"
  - "BenchmarkCoreEventLoop and TestCore_unackedCostFlat: the 10-vs-10 000 2x acceptance as an executable gate"
affects: [03-06, 03-07, 03-08, 05-grpc-wait, 06-observability]

actuals:
  tokens: 9743
  tasks: 3
  commits: 3
plan_head_before: faef98dfbec1f757af38d46cd6251e3cecc15a2c
plan_head_after: 4ec662abf9f94eb982ff584f3f35bf877251a675

tech-stack:
  added: []
  patterns:
    - "Record-free pending tracking: the store is the single record source, and an emission that cannot resolve the tracked version is skipped."
    - "External position map over container/heap: map[pendingKey]int supports Fix/Remove in O(log n) without a heap pointer in the entry."
    - "Per-peer reverse index (peerUnacked) makes peer eviction O(peer unacked) instead of O(all pending)."
    - "O(1) gauges from map length plus an ordered list front, never a pending-set walk."
    - "Ratio gate on testing.Benchmark pairs: warm-up, both ns/op values named on failure, zero measurement fails by name."

key-files:
  created: []
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/bench_test.go
    - internal/udprepl/repl.go

key-decisions:
  - "The discovery flush's wait-set half is unconditional: the new peer joins every tracked entry's wait-set and ledger even when the store lookup cannot resolve the tracked version; only the stale broadcast is skipped, and the retransmit scan removes the stale entry at its next due time."
  - "Due retransmissions emit in (key, version) order to preserve the standing sorted-emission pin (probe #9), while the heap's (nextRetry, seq) order still makes the pop order a deterministic total order; the invariant test inserts ascending keys so insertion order and emission order coincide."
  - "Local supersession removes every strictly smaller tracked version under the key; an equal-version notification leaves tracking untouched and still broadcasts the (idempotent) record."
  - "Remote supersession generalizes today's single-key delete to every tracked version under the key that the strictly greater remote version displaced."
  - "PendingSnapshot keeps its keyed-by-key public contract and reports the greatest version under a key when two versions coexist transiently, matching the store's LWW choice."
  - "The adapter mirror now reads PeerCount() for the peer gauge; mirrorState is deliberately untouched here because 03-06 owns its removal."
  - "TestCore_unackedCostFlat stays sequential with a justified //nolint:paralleltest: it is a wall-clock ratio gate, and a parallel sibling benchmark would measure scheduler contention."

patterns-established:
  - "Index by identity, not by field: (key, version) in the map removes the stale-version ambiguity and needs no entry-held record."
  - "Structured pending state has one deletion helper (removePending) that clears map, list, heap and every waiter's ledger — no path can leak a stale index node (T-3-14)."
  - "Every emission resolves its record at send time; absence or a version mismatch is silence or deletion, never a stale datagram."
  - "Performance acceptance stays executable: fixed 2x bound, both measurements printed, named failure on a zero sample."

requirements-completed: [XPORT-02]

coverage:
  - id: D1
    description: "Pending tracking is keyed by (key, version) and retains no record Data; local and remote supersession remove every older version under the key"
    requirement: XPORT-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestRetransmit_|^TestPending_'"
        status: pass
      - kind: other
        ref: "sed -n '/^type pendingEntry struct/,/^}/p' internal/app/repl/core.go: no rec/Data field"
        status: pass
    human_judgment: false
  - id: D2
    description: "Per-peer ledger makes peer eviction O(that peer's unacked); UnackedStats and PeerCount are O(1) and covered by an empty-set gauge test"
    requirement: XPORT-02
    verification:
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestPending_gauges"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/udprepl/ (mirror and smoke suites)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Indexed due-heap scans only due entries, re-arms and deletes in O(log n) with no stale items, and keeps equal-deadline emission deterministic while the documented ladder timing stays unchanged"
    requirement: XPORT-02
    verification:
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestRetransmit_heapInvariant"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestRetransmit_|^TestBound_lossWithinAckTimeoutMax$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "The event-loop iteration cost at 10 and 10 000 unacked records differs by at most 2x, enforced by an automated benchmark-ratio gate"
    requirement: XPORT-02
    verification:
      - kind: unit
        ref: "internal/app/repl/bench_test.go#TestCore_unackedCostFlat"
        status: pass
      - kind: other
        ref: "go test -run '^$' -bench '^BenchmarkCoreEventLoop$' -benchtime=2x ./internal/app/repl/ renders both unacked arms"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 5: Record-Free Unacked Ledger and Indexed Due-Heap Summary

**Record-free `(key, version)` pending tracking with a per-peer ledger, an indexed due-heap and O(1) gauges makes one event-loop iteration flat at 10 000 unacked while the retransmit ladder's documented timing stays byte-identical.**

## Performance

- **Duration:** 15 min (880 s)
- **Started:** 2026-11-21T19:08:59Z
- **Completed:** 2026-11-21T19:23:39Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- The pending set is now `map[pendingKey]pendingEntry` with `pendingKey{key, version}` and no retained record: every discovery flush and retransmit resolves the record through `StateStore.Lookup`, so the store is the single source and a version mismatch can never send stale bytes.
- `peerUnacked map[uint16]map[pendingKey]struct{}` makes peer eviction touch only the expiring peer's unacked entries; the entry survives in suspension exactly as before, and `acked` stays untouched.
- `UnackedStats` reads the map length plus the front of a createdAt-ordered `container/list`, and `PeerCount` is the peer-table length; `mirrorStats` no longer calls the allocating `PeerSnapshot`.
- An indexed `(nextRetry, seq, pendingKey)` heap with a `map[pendingKey]int` position index makes `Advance` due-only and `nextDeadline` read the heap root; `earliestRetry` is gone, and every deletion path (drain, local replacement, remote supersession, stale-record skip) removes its heap item.
- `BenchmarkCoreEventLoop` and `TestCore_unackedCostFlat` turn XPORT-02(1) into an executable gate: measured 1042 ns/op at 10 unacked versus 1137 ns/op at 10 000, ratio 1.09, against the 2.0 bound.
- The retransmit ladder, suspension/flush, Ack-drain, eviction and sync suites are unchanged and green; `sync_test.go` needed no edits and `retransmit_test.go` only gained tests.

## Task Commits

Each task was committed atomically:

1. **Task 1: Record-free (key, version) ledger with per-peer unacked index** — `f4a166a` (feat)
2. **Task 2: Indexed due-heap replaces the whole-set retransmit scan** — `4f7014b` (feat)
3. **Task 3: Event-loop flatness gate at 10 vs 10 000 unacked** — `4ec662a` (test)

**Plan metadata:** see the `docs(03-05)` commit that follows this summary.

_Note: no TDD tasks in this plan; each task is one implementation commit._

## Files Created/Modified

- `internal/app/repl/core.go` — `pendingKey`, record-free `pendingEntry` with its list node, `peerUnacked`, the createdAt-ordered pending list, the indexed `retryHeap` and its `Core` helpers (`pushRetry`/`fixRetry`/`removeRetry`/`retryRoot`/`popDueRetries`), rewritten `handleHello` discovery flush, `applyRemote`/`handleAck`/`NotifyLocalWrite`/`evictPeers`/`scanRetransmit`/`nextDeadline`/`PendingSnapshot`/`UnackedStats`, and the new `PeerCount`.
- `internal/app/repl/retransmit_test.go` — new `TestPending_gauges` (empty gauge, out-of-order creation times, peer-table length) and `TestRetransmit_heapInvariant` (insert/ack/supersede/evict/re-arm churn, silence before the minimum `NextRetry`, exactly-due emission, equal-deadline order).
- `internal/app/repl/bench_test.go` — new `BenchmarkCoreEventLoop` (`unacked=10`, `unacked=10000`), `TestCore_unackedCostFlat` ratio gate, and the shared fixture.
- `internal/udprepl/repl.go` — `mirrorStats` reads `core.PeerCount()` instead of `len(core.PeerSnapshot())`.

## Decisions Made

- The discovery flush always adds the new peer to each tracked entry's wait-set and ledger; when `Lookup` cannot resolve the tracked version it skips only the broadcast. The entry stays tracked, and the retransmit scan removes it at its next due time — the wait-set half is not conditional on a record read.
- Due retransmissions emit by `(key, version)` to keep the repository's standing sorted-emission pin (probe #9) while the heap's `(nextRetry, seq)` order remains the deterministic pop order. `TestRetransmit_heapInvariant` inserts ascending keys, so its equal-deadline emissions are simultaneously insertion order and key order.
- Local supersession deletes every strictly smaller tracked version under the key; an equal-version notification is a tracking no-op but still broadcasts the idempotent record. A stale smaller local write does not displace a newer tracked entry.
- Remote supersession deletes every tracked version under the key strictly smaller than the applied remote version, replacing the old single-map-lookup delete.
- `PendingSnapshot` reports the greatest version under a key when two versions coexist transiently, keeping its keyed-by-key public contract and matching the store's LWW choice.
- `mirrorState` is untouched in this plan (03-06 owns its removal); only the gauge half of `mirrorStats` changed.
- The flatness gate stays sequential with a specific, justified `//nolint:paralleltest` — a wall-clock ratio gate must not measure a parallel sibling benchmark.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Plan/test conflict] Due-emission order is `(key, version)`, not raw heap-pop order**
- **Found during:** Task 2 (indexed due-heap)
- **Issue:** The plan's invariant-test sentence asked for equal-deadline emission in insertion order, but the pre-existing `TestRetransmit_backoffLadder` behaviorally pins multi-key emissions sorted ascending (writes gamma, alpha, beta must emit alpha, beta, gamma). Raw `(nextRetry, seq)` pop order would break that standing assertion, which the plan also requires to stay green.
- **Fix:** Kept the heap keyed by `(nextRetry, seq)` for a deterministic total order, then sorted the due set by `(key, version)` before emission. The new invariant test inserts ascending keys, so its equal-deadline emissions demonstrate insertion order and the sorted-key pin together.
- **Files modified:** `internal/app/repl/core.go`, `internal/app/repl/retransmit_test.go`
- **Verification:** `TestRetransmit_backoffLadder`, `TestRetransmit_heapInvariant` and the full retransmit set green; the ladder timing assertions are unchanged.
- **Committed in:** `4f7014b`

**2. [Rule 1 - Bug] Discovery flush initially dropped entries the store could not resolve**
- **Found during:** Task 1 (record-free ledger)
- **Issue:** The first flush implementation called `removePending` on a `Lookup` miss. Two existing `internal/udprepl` mirror tests legitimately notify a local write without a preceding store write (a synthetic path), so the discovery flush deleted their entry and the mirrored wait-set never gained the peer.
- **Fix:** The wait-set/ledger/ladder half is now unconditional; a failed lookup only skips the broadcast, and the entry remains tracked until the next retransmit scan (which owns stale-entry deletion). This also matches the plan's Task 1 step 5 wording.
- **Files modified:** `internal/app/repl/core.go`
- **Verification:** `go test -race -count=1 ./internal/udprepl/` green, including `TestMirror_pendingAckVisible` and `TestMirror_drainedAckedDedup`.
- **Committed in:** `f4a166a`

---

**Total deviations:** 2 auto-fixed (1 plan/test conflict resolution, 1 bug)
**Impact on plan:** No scope change. The emission order preserves an existing behavioral contract the plan also requires; the flush fix preserves the mirror contract while keeping the record-free guarantee.

## Issues Encountered

- The discovery-flush `Lookup` miss against the synthetic udprepl mirror tests was the only behavioral regression the full suite caught; it drove deviation 2.
- Lint iterations: `gci` formatting on the test insertion, `godoclint` requiring `[heap.Fix]`/`[heap.Remove]` doc links, and `gosec` G115 on `uint64(PeerCount())` — fixed with formatting, doc links, and a specific justified `//nolint:gosec` (the value is a map length).
- `TestCore_unackedCostFlat` takes about 30 s under `-race` because each arm runs a warm-up plus a `testing.Benchmark` measurement; the acceptance requires both, so the time is intentional.
- The benchmark's 2-iteration smoke run is noisy by nature (`unacked=10` measured slower than `unacked=10000` at `-benchtime=2x`); the gate's auto-scaled measurement reports the stable 1.09 ratio.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 03-06 can delete `mirrorState`/`retainDrained`'s per-event pending copy: `PeerCount` and the O(1) `UnackedStats` are already in place, and `PendingSnapshot` remains the on-demand seam.
- 03-07's paced record queue can index sends by the same `pendingKey` and re-arm the ladder from emission time, reusing `pushRetry`/`fixRetry`.
- The ladder timing (200/400/800/1600/3200/5000 ms; loss bounded by `ack_timeout_max`) is unchanged and still pinned by `TestRetransmit_backoffLadder`, `TestBound_lossWithinAckTimeoutMax` and the simulator suites.
- The full repository suite (`go test -race ./...`), `go build ./...` and the strict lint gate on both touched packages are green.

## Self-Check: PASSED

- Files verified present: `internal/app/repl/core.go`, `internal/app/repl/retransmit_test.go`, `internal/app/repl/bench_test.go`, `internal/udprepl/repl.go`.
- Commits verified in history: `f4a166a`, `4f7014b`, `4ec662a`; `check evaluation-scope --plan 03-05` resolves exactly these three commits.
- Plan verification re-run at close-out: `go test -race -count=1 ./internal/app/repl/`, `go test -race -count=1 ./internal/udprepl/`, both flatness gates (`TestCore_unackedCostFlat`, `TestCore_keyCountCostFlat`), `go test -race -count=1 ./internal/app/internal/simnet/`, `go tool golangci-lint run ./internal/app/repl/ ./internal/udprepl/` and `go build ./...` all green.
- Acceptance re-check: `pendingEntry` has no record/Data field; eviction walks only `peerUnacked[id]`; `UnackedStats`/`PeerCount` are O(1) and `TestPending_gauges` covers the empty set; local and remote supersession delete every older version; ladder/eviction/sync tests unchanged and green.
- Measured commits: `git rev-list --count faef98d..4ec662a` = 3.

---
*Phase: 03-transport*
*Completed: 2026-11-21*
