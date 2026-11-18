---
phase: 05-grpc-api-and-binary
plan: 02
subsystem: api
tags: [replication, seams, atomics, wait-for-replication, startup-gate, copy-on-write]

# Dependency graph
requires:
  - phase: 04-replication-and-deterministic-simulator
    provides: repl.Core single-driver state machine (pending tracking, handleAck, evictPeers, handleDataSync), udprepl Adapter post-iteration mirrorStats site and its atomic-mirror discipline
  - phase: 05-grpc-api-and-binary
    provides: 05-01's grpcapi adapter — the consumer that reads these seams in 05-04 (gate resolver) and 05-05 (wait_for_replication loop)
provides:
  - repl.Core.SyncArrived() bool — first-DataSync-received flag, A1 semantics documented on the accessor (API-04 sync half for the startup gate)
  - repl.PendingInfo.Acked []uint16 — sorted, idempotent (re-Ack adds nothing), never touched by eviction, so Ack and peer_timeout eviction are distinguishable by construction (API-05 honesty half, T-05-04)
  - udprepl.Adapter.SyncArrived() bool and Adapter.PendingCopy() map[string]repl.PendingInfo — copy-on-write atomic mirrors refreshed after every loop event, readable from any goroutine
affects: [05-04, 05-05, 05-06, phase-6-hardening]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 8231    # chars/4 over the realized diff (32925 chars)
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns: [copy-on-write atomic mirror (pointer replaced wholesale, every slice cloned — published snapshots immutable and stable across events), sibling mirrorState called at the existing post-iteration mirrorStats site so the state pair is never torn, loopback-unicast driving of the real event loop with the fake peer holding ONE socket]

key-files:
  created: []
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/sync_test.go
    - internal/udprepl/repl.go
    - internal/udprepl/repl_test.go

key-decisions:
  - "mirrorState is one sibling call at the existing post-iteration stats site — the sync flag and pending copy are refreshed together after every loop event, so a reader never sees the pair torn"
  - "handleAck writes the pending entry back to the map after appending to acked (the entry is a value copy; waitSet was always mutated through its map reference) — the drain path still deletes, the retained path now persists the acked update"
  - "evictPeers deliberately never touches acked — eviction stays distinguishable from an Ack by construction (T-05-04, RESEARCH Pitfall 3), locked by the eviction-shrinks-WaitingOn-while-Acked-stays-empty regression test"
  - "SyncArrived A1 reading pinned on the accessor doc: the first received DataSync datagram counts as the first sync round (chunks are self-sufficient, no completeness state), the startup_gate timeout is the backstop"
  - "Mirror tests drive the real event loop over loopback unicast with the fake peer holding one socket across sends — a per-datagram throwaway socket makes the core's REPL-08 duplicate-node_id classifier discard the peer's later datagrams (same identity, second handle)"

patterns-established:
  - "Cross-goroutine replication reads: core state changes only on the driver goroutine; anything a handler goroutine needs is mirrored into adapter atomics at the post-iteration site (counters, peer count, sync flag, pending copy)"
  - "Copy-on-write snapshots for cross-goroutine reads: rebuild + wholesale pointer store, never mutate a published map; callers hold stable snapshots across events"

requirements-completed: [API-04, API-05]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Core.SyncArrived() bool — false until the first received DataSync datagram (own emissions and Hello receipts do not count), true forever after (A1: any received chunk counts)"
    requirement: API-04
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_syncArrived"
        status: pass
    human_judgment: false
  - id: D2
    description: "PendingInfo.Acked []uint16 — an Acked peer reads from Acked and vanishes from WaitingOn; peer_timeout eviction shrinks WaitingOn while Acked stays empty; a duplicate Ack adds no second entry; multi-peer Acked stays sorted despite scrambled arrival order"
    requirement: API-05
    verification:
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestRetransmit_ackDrainSemantics (the Acked rows: sorted-insertion, eviction-regression-lock, duplicate-idempotence)"
        status: pass
    human_judgment: false
  - id: D3
    description: "udprepl.Adapter.SyncArrived()/PendingCopy() — both replication states readable from any goroutine through copy-on-write atomics refreshed after every loop iteration; a snapshot held across a later event stays stable while the fresh copy shows the post-event state"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestMirror_pendingAckVisible"
        status: pass
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestMirror_syncArrivedFlip"
        status: pass
    human_judgment: false
  - id: D4
    description: "Zero behavior change to replication itself: no timer, message or state-machine logic touched; both packages stay green under race and the strict lint gate"
    requirement: API-04
    verification:
      - kind: other
        ref: "command:go test -race ./internal/app/repl/... ./internal/udprepl/... && go tool golangci-lint run ./internal/app/repl/... ./internal/udprepl/... (both exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 16min
completed: 2026-11-09
status: complete
commits: 2
plan_head_before: ce4a28edcfdb5aa4109f88bf0d065add570e85bd
plan_head_after: d1026dcff8a90bac08160e37ccf0c60ec6cfc8d2
---

# Phase 05 Plan 02: Replication Read Seams Summary

**Core.SyncArrived (first-DataSync, A1) and PendingInfo.Acked (Ack-vs-eviction honesty) added to the pure repl core, mirrored cross-goroutine by udprepl copy-on-write atomics — the API-04 gate's sync half and the API-05 wait report's honest aggregate, proven under -race**

## Performance

- **Duration:** 16 min
- **Started:** 2026-11-09T10:57:10Z
- **Completed:** 2026-11-09T11:13:22Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- `repl.Core` gains the unexported `syncArrived` flag — set by `handleDataSync` on any received DataSync datagram (empty listings included, the A1 reading) and never reset — exposed through the copying `SyncArrived()` accessor whose doc pins the A1 semantics and the startup_gate backstop (API-04 sync half)
- `pendingEntry` gains the sorted, idempotent `acked` slice: `handleAck` inserts the acking peer (with the map write-back the value-copy semantics require); `evictPeers` deliberately never touches it, so `PendingSnapshot` distinguishes "peer Acked" (`Acked` contains it) from "peer evicted by peer_timeout" (absent from both `Acked` and `WaitingOn`) — the T-05-04 mitigation for the D-08 aggregate (API-05 honesty half)
- `udprepl.Adapter` gains `syncArrived atomic.Bool` and `pendingCopy atomic.Pointer[map[string]repl.PendingInfo]`, refreshed by a sibling `mirrorState` at the exact post-iteration site `mirrorStats` runs at; the pending mirror is copy-on-write (fresh map, every WaitingOn/Acked slice cloned) so published snapshots are immutable and stable across events; `SyncArrived()`/`PendingCopy()` accessors follow the `KnownPeers()` style with the single-driver discipline documented
- Tests: three new Acked rows in `TestRetransmit_ackDrainSemantics` (sorted insertion under scrambled arrival, the eviction regression lock, duplicate-Ack idempotence), `TestSync_syncArrived` (false before any receipt including after own emissions and a Hello, true after one DataSync datagram, sticky), and `TestMirror_pendingAckVisible`/`TestMirror_syncArrivedFlip` driving the real event loop over loopback unicast — everything green under `-race`, strict lint clean

## Task Commits

Each task was committed atomically:

1. **Task 1: repl.Core — sync-arrived state and PendingInfo.Acked** - `b06b54d` (feat)
2. **Task 2: udprepl mirrors — SyncArrived and PendingCopy atomics** - `d1026dc` (feat)

## Files Created/Modified
- `internal/app/repl/core.go` - syncArrived flag + SyncArrived accessor; pendingEntry.acked + ackedInsert; handleAck insertion with write-back; evictPeers acked-untouched pin; PendingInfo.Acked cloned by PendingSnapshot
- `internal/app/repl/retransmit_test.go` - three Acked rows in TestRetransmit_ackDrainSemantics (sorted insertion, eviction regression lock, duplicate idempotence)
- `internal/app/repl/sync_test.go` - TestSync_syncArrived (the A1 flip and stickiness)
- `internal/udprepl/repl.go` - syncArrived/pendingCopy atomics; mirrorState at the post-iteration site; SyncArrived/PendingCopy accessors with the immutable-snapshot contract
- `internal/udprepl/repl_test.go` - TestMirror_pendingAckVisible / TestMirror_syncArrivedFlip over the real loop; peerSocket/sendDatagram/newMirrorAdapter helpers (fake peer holds one socket)

## Decisions Made
- The mirror runs as one sibling `mirrorState` call beside `mirrorStats` — the sync flag and pending copy refresh together after every loop event, never torn
- `handleAck` writes the entry back to the pending map: `entry` is a value copy, so the new `acked` slice header must be stored (the drain path still deletes unchanged)
- The A1 reading ("first received DataSync datagram counts") is documented on the accessor itself — the single home for the semantics the 05-04 gate resolver consumes
- Mirror tests drive the real event loop deterministically: loopback unicast to the adapter's exclusively-held port, with the fake peer holding ONE socket so the REPL-08 duplicate-node_id classifier never discards its traffic

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] TestSync_syncArrived drives the flip receipt from peerID2 (unparam)**
- **Found during:** Task 1
- **Issue:** The new `syncFrom(peerID1)` call made `syncFrom`'s peerID parameter receive peerID1 at every call site; the strict gate's `unparam` failed the plan's own lint `<verify>` (empirically green at HEAD, red with the addition)
- **Fix:** The flip receipt is driven from peerID2 in code direction — no suppression added; the parameter genuinely varies and the test additionally proves the flip works from any sender
- **Files modified:** internal/app/repl/sync_test.go
- **Verification:** `go tool golangci-lint run ./internal/app/repl/...` exits 0; focused tests still pass
- **Committed in:** b06b54d (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Test-construction detail only; the plan's semantics, files and acceptance criteria are unchanged.

## Issues Encountered
- `TestMirror_syncArrivedFlip` initially timed out: the fake peer sent its Hello and DataSync from two different ephemeral sockets, so the core's second-variant duplicate-node_id classifier (REPL-08: one identity claimed from two handles) correctly discarded the DataSync. Fixed by holding one persistent peer socket across sends (`peerSocket`) — the test now models the real one-node-one-socket topology and doubles as a classifier-adjacency check.

## Known Stubs
None — no placeholder, TODO or unwired surface was introduced; both seams are consumed by tests and land ready for the 05-04 gate resolver and the 05-05 wait loop.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- API-04 sync half ready: 05-04's gate resolver reads `Adapter.SyncArrived()` (OR the startup_gate deadline) — never the core
- API-05 honesty half ready: 05-05's wait loop polls `Adapter.PendingCopy()`, counts from `Acked`, and `total` stays the write-time peer snapshot (D-10/D-11 unchanged)
- No blockers; go.mod/go.sum untouched (T-05-SC holds — no package installs)

---
*Phase: 05-grpc-api-and-binary*
*Completed: 2026-11-09*

## Self-Check: PASSED

All five modified files exist on disk; commits b06b54d and d1026dc are ancestors of HEAD; `commits: 2` is measured from the persisted ledger (`git rev-list --count ce4a28e..HEAD`); plan-level gates green at SUMMARY time: `go test -race ./internal/app/repl/... ./internal/udprepl/...` and `go tool golangci-lint run ./internal/app/repl/... ./internal/udprepl/...` both exit 0; go.mod/go.sum untouched (T-05-SC).
