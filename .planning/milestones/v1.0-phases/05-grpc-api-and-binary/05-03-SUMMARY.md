---
phase: 05-grpc-api-and-binary
plan: 03
subsystem: api
tags: [watch, grpc-streaming, change-events, snapshot-then-live, backpressure, hub-goroutine]

# Dependency graph
requires:
  - phase: 05-grpc-api-and-binary
    provides: 05-01's bufconn harness, committed SupKV wire contract (Watch = stream WatchEvent) and the grpcapi adapter package the hub and handler live in
  - phase: 04-replication-and-deterministic-simulator
    provides: persist.Engine (Set/ApplyRemote/Records) whose store Config threads the change tap; the strictly-greater install law that makes D-06 decidable
provides:
  - storage.Config.OnChange func(key string, rec Record) — the change tap fired from install under the store write lock, nil-normalized to a no-op; the record handed over is the stored copy (never mutated in place), safe to retain
  - grpcapi.WatchHub: NewWatchHub() (watchBuffer=256 per-subscriber buffer), NewWatchHubWithBuffer(n) test variant, non-blocking Ingest(key, rec) into the bounded ingress (4096) with overflow-flag disconnect-all, Run(ctx) hub goroutine
  - The snapshot-then-live exact cut: taken on the hub goroutine while servicing each subscribe; live events delivered only when strictly dominating the snapshot version for their key (RecordVersion.Compare); tombstones as ordinary deleted=true events (Q3 pin); arrival order, no origin field (D-05/D-07)
  - Server.Watch server-streaming handler; NewServer(engine, notify, hub) — the hub attaches the engine; nil hub keeps Watch answering Unimplemented (the embed path)
  - Test families TestStore_onChange* and TestWatch_snapshotThenLive/_exactCut/_replicatedChange/_prefix/_slowConsumer; harness extended with the watch wiring (hub-first construction, buffer parameter, static 64 KiB flow window, engine+hub exposure)
affects: [05-04, 05-05, 05-06, phase-6-hardening]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 17200    # chars/4 over the realized diff (~68.8k chars)
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns: [single-goroutine hub owning all subscriber state (udprepl event-loop discipline), non-blocking select-default ingress with overflow-flag shed, exact-cut suppression against the subscribe-time snapshot version map, disconnect-never-drop backpressure (etcd-style A3), err-set-before-channel-close termination status read via the close happens-before edge, snapshot streamed by the handler from the subscribe reply so a snapshot larger than the buffer cannot block the hub]

key-files:
  created:
    - internal/grpcapi/watch.go
    - internal/grpcapi/watch_test.go
  modified:
    - internal/app/storage/store.go
    - internal/app/storage/store_test.go
    - internal/grpcapi/server.go
    - internal/grpcapi/server_test.go

key-decisions:
  - "write() now routes through install() so the tap truly has one convergence point: the plan's mechanism assumed local writes reach install, but store.go's write stored the record directly — routing local Set/SetIfAbsent/Delete through install is what makes the tap fire for every change source (D-06 decidable at one point)"
  - "The snapshot is built by the hub goroutine during handleSubscribe and streamed by the handler from the subscribe reply (sorted-key order — arrival order governs live changes only), so a snapshot larger than the subscriber buffer can never block the hub or deadlock a slow reader; live events relay through the bounded channel with select-default disconnect"
  - "The disconnect status is written into sub.err before closing the events channel — the close happens-before the handler's zero-value receive, so the handler reads the termination status with no further synchronization"
  - "TestWatch_slowConsumer floods hub.Ingest (the production OnChange entry point) to overflow the buffer-2 channel deterministically: gRPC's dynamic flow control grants a stalled reader's stream far more buffering than any test-scale write round can exhaust (measured: a 1 MiB write round was absorbed; a pending Recv itself enlarges the window via requestRead/adjustWindow), so real writes alone cannot force the shed at test scale"
  - "The harness client advertises a static 64 KiB stream flow window (transport floor — smaller values are ignored) to bound reader-less buffering; all other tests read continuously and never notice it"

patterns-established:
  - "Change events cross goroutine boundaries through bounded non-blocking channel sends; overflow always sheds subscribers (ResourceExhausted + resubscribe), never drops events and never blocks the producer"
  - "Single-writer hub discipline: subscriber fields are hub-owned, cross-goroutine handoffs go through the subscribe/unsubscribe/done channels; one happens-before edge (err written before channel close) carries the termination status"

requirements-completed: [API-03]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "storage.Config.OnChange tap fires exactly once per actual install across every change source (fresh/overwrite Set, tombstone Delete, winning remote Merge, Merge-delivered tombstone) and stays silent on a strictly-older loser and an equal-version redelivery (D-06); the handed record equals what Get/Records returns"
    requirement: API-03
    verification:
      - kind: unit
        ref: "internal/app/storage/store_test.go#TestStore_onChange"
        status: pass
      - kind: other
        ref: "command:go test -race ./internal/app/storage/ -run '^TestStore_onChange' && go tool golangci-lint run ./internal/app/storage/... (both exit 0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Snapshot-then-live subscription with the exact cut: first events are the whole current state as ordinary WatchEvents (tombstone included, deleted=true), then every live change exactly once with no duplicates and no misses under concurrent writers (D-02, D-03, Q3, Pitfall 5)"
    requirement: API-03
    verification:
      - kind: integration
        ref: "internal/grpcapi/watch_test.go#TestWatch_snapshotThenLive"
        status: pass
      - kind: integration
        ref: "internal/grpcapi/watch_test.go#TestWatch_exactCut"
        status: pass
    human_judgment: false
  - id: D3
    description: "Replicated changes arrive indistinguishably from local ones (no origin field, full record shape) and a losing remote merge fires nothing end-to-end (D-07, D-06)"
    requirement: API-03
    verification:
      - kind: integration
        ref: "internal/grpcapi/watch_test.go#TestWatch_replicatedChange"
        status: pass
    human_judgment: false
  - id: D4
    description: "Prefix filter on both the snapshot and live phases; empty prefix sees all keys (D-04)"
    requirement: API-03
    verification:
      - kind: integration
        ref: "internal/grpcapi/watch_test.go#TestWatch_prefix"
        status: pass
    human_judgment: false
  - id: D5
    description: "Slow consumer is shed with ResourceExhausted while the store keeps accepting and confirming writes throughout — the tap never blocks the store write lock (T-05-05, A3)"
    requirement: API-03
    verification:
      - kind: integration
        ref: "internal/grpcapi/watch_test.go#TestWatch_slowConsumer"
        status: pass
    human_judgment: false
  - id: D6
    description: "The storage seam is additive with zero behavior change elsewhere: full make test (race+cover, all packages) and the full-repo strict lint gate green; internal/app/persist untouched (git diff shows no persist files)"
    requirement: API-03
    verification:
      - kind: other
        ref: "command:make test && go tool golangci-lint run ./... (both exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 60min
completed: 2026-11-09
status: complete
commits: 2
plan_head_before: e2dc8a3167b720c87318c5cbd1464b543cef3ab2
plan_head_after: fc8581bc327429d99148708b312b4401bc95943a
---

# Phase 05 Plan 03: Watch Stream Vertical Slice Summary

**Watch delivered as one vertical slice: the storage OnChange tap fired from the single install convergence point, a single-goroutine WatchHub with the exact snapshot-then-live cut and disconnect-never-drop backpressure, and the server-streaming Watch RPC over the 05-01 bufconn harness — all proven under -race**

## Performance

- **Duration:** 60 min
- **Started:** 2026-11-09T11:20:12Z
- **Completed:** 2026-11-09T12:20:18Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- `storage.Config.OnChange func(key string, rec Record)` — the change tap, documented with the under-write-lock/must-not-block constraint and the safe-retention contract, nil-normalized to a no-op exactly like `repl.Config.Log`; fired from `install` with the stored record (recomputed checksum included — the exact record a following Get returns)
- `write` now routes through `install` so every change source — local Set/SetIfAbsent/Delete, winning remote merges, merge-delivered tombstones, restore replay — converges on the single tap point; the strictly-greater join law keeps losers and equal-version redeliveries from ever reaching it (D-06), locked by regression tests
- `grpcapi.WatchHub` — one goroutine owning all subscriber state: non-blocking `Ingest` (the OnChange target) into the bounded 4096-event ingress so the store never waits; overflow sheds subscribers instead of dropping events; subscribe/unsubscribe via channels with a done-guarded shutdown sweep
- The exact cut, exact by construction: the hub goroutine takes `Records()` while servicing each subscribe, filters by prefix (D-04, empty = all), and delivers live events only when strictly dominating the snapshot version for their key; tombstones travel as ordinary deleted=true events (D-02, the Q3 pin — List excluding them is the deliberate asymmetry); arrival order preserved, no origin field (D-05/D-07)
- `Server.Watch` server-streaming handler: streams the snapshot, then relays until the stream context ends or the hub sheds the subscriber with ResourceExhausted ("watch subscriber overwhelmed; resubscribe"); `NewServer(engine, notify, hub)` grows the hub parameter and attaches the engine (hub-first construction works because `persist.Open` threads `storage.Config`)
- Five bufconn integration test families plus the storage tap tests, all green under -race; full `make test` and the full-repo strict lint gate green; `internal/app/persist` untouched

## Task Commits

Each task was committed atomically:

1. **Task 1: storage OnChange tap — fired from install, only on actual state change (D-06)** - `c2746cb` (feat)
2. **Task 2: WatchHub — snapshot-then-live exact cut, prefix filter, disconnect-on-full** - `fc8581b` (feat)

## Files Created/Modified
- `internal/app/storage/store.go` - OnChange in Config (nil-normalized), write routed through install, install fires the tap with the stored record
- `internal/app/storage/store_test.go` - TestStore_onChange (8 subtests: every source fires, D-06 silence locks, stored-record equality, nil OnChange no-op) + the changeTapRecorder helper
- `internal/grpcapi/watch.go` - WatchHub (Ingest/Run/subscribe/unsubscribe/disconnect paths), the exact-cut wants rule, Server.Watch + relayEvent, load-bearing watchBuffer/watchIngress constants
- `internal/grpcapi/watch_test.go` - TestWatch_snapshotThenLive/_exactCut/_replicatedChange/_prefix/_slowConsumer + recv helpers and the exact-cut assertion helper
- `internal/grpcapi/server.go` - Server gains the hub field; NewServer takes and attaches the hub; the "unimplemented RPCs" doc drops Watch
- `internal/grpcapi/server_test.go` - harness builds the hub first, threads OnChange: hub.Ingest into persist.Open, runs hub.Run under t.Cleanup, buffer parameter (NewWatchHubWithBuffer), static 64 KiB flow window, engine+hub exposed on testHarness

## Decisions Made
- write() routes through install() — the plan's "all converge on install" was an assumption about the code that did not hold; making it true is what lets the tap fire for local writes at all
- Snapshot events are streamed by the handler from the subscribe reply (hub-built, sorted-key order) rather than relayed through the bounded channel — a snapshot larger than watchBuffer can never block the hub or shed a fresh subscriber mid-snapshot
- The disconnect status rides the close happens-before edge (err written before channel close) — no locks between hub and handler goroutines
- TestWatch_slowConsumer uses an Ingest flood for determinism: gRPC's flow control grants a stalled reader's stream ~0.5-1 MiB of buffering (a pending Recv itself widens the window), so no test-scale write round can force the shed; the flood exercises the identical Ingest entry point the tap uses, and real Set liveness is asserted separately while the wedged subscriber is attached
- The harness client pins a static 64 KiB stream flow window (the transport's floor — smaller values are ignored) to bound reader-less buffering; every other test reads continuously and is unaffected

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Local writes did not converge on install — write() stored records directly**
- **Found during:** Task 1
- **Issue:** The plan fires the tap from install and states all change sources converge there, but store.go's write() wrote `s.records[key]` directly; firing only in install would leave Set/SetIfAbsent/Delete silent, failing the task's own test list
- **Fix:** write() now stamps the version via Tick and installs through install() (which clones the data and recomputes the checksum once) — the single convergence point the plan describes is now real; behavior otherwise byte-identical (the full storage suite stays green unmodified)
- **Files modified:** internal/app/storage/store.go
- **Verification:** TestStore_onChange fires on every source; the entire pre-existing storage suite passes without modification; persist/repl/udprepl suites green under -race
- **Committed in:** c2746cb

**2. [Rule 3 - Blocking] Slow-consumer shed untestable at test scale through real writes alone**
- **Found during:** Task 2
- **Issue:** The plan's TestWatch_slowConsumer spec (buffer-2 hub; subscriber stops reading; writes continue) is not reachable deterministically: gRPC v1.85's flow control grants a stalled reader's stream ~0.5-1 MiB of buffering (measured — a 1 MiB write round was absorbed; a pending Recv widens the window via requestRead/adjustWindow), and each test-scale Set is paced by dump confirmation, so the buffer-2 channel never overflows in test time
- **Fix:** Two harness/test adaptations, both documented in code: (a) the harness client advertises a static 64 KiB stream flow window (the transport floor — smaller values are ignored), bounding reader-less buffering; (b) the test floods hub.Ingest — the identical entry point the storage tap calls — whose dispatch cost per event is orders below the handler's relay cost, so the channel pins full and the shed fires deterministically; the real-write store-liveness assertion is kept verbatim
- **Files modified:** internal/grpcapi/server_test.go, internal/grpcapi/watch_test.go
- **Verification:** TestWatch_slowConsumer passes in ~0.5s under -race, asserting both the ResourceExhausted termination and that every Set completed
- **Committed in:** fc8581b

---

**Total deviations:** 2 auto-fixed (2 blocking-environment/test-infrastructure adaptations)
**Impact on plan:** Both deviations adapt test infrastructure to real library behavior; the plan's semantics, files, acceptance criteria and all D-02..D-07 guarantees are unchanged. No scope creep.

## Issues Encountered
- Nine strict-lint findings fixed in code direction across the two tasks (noinlineerr on the new storage tests; paralleltest defer, unused-receiver, flag-parameter, context-as-argument and unparam on the watch code) — no suppressions added; every gate re-run green after the fixes
- The slow-consumer test went through three designs while measuring gRPC's actual backpressure behavior (96-write burst, 1 MiB burst, 2 MiB snapshot park) before landing on the Ingest flood — the journey is recorded above because the measured behavior (a pending Recv enlarging the window) is what 05-06's shutdown and any future slow-consumer tuning will need to know

## Known Stubs
None — Watch is fully implemented end-to-end; the nil-hub Watch path answering Unimplemented is the documented embedding behavior (a server constructed without a hub is a programming error, and NewServer's doc says so), not a placeholder.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- API-03 delivered: 05-04's gate resolver and 05-05's snapshots wait on a server whose Watch surface is complete; the harness already wires hub-first construction, so main() (05-06) copies `NewWatchHub() → persist.Open(OnChange: hub.Ingest) → go hub.Run(ctx) → NewServer(eng, notify, hub)` verbatim
- WatchHub's shutdown path (ctx cancel closes every subscriber channel cleanly and guards late subscribe/unsubscribe sends) is ready for the D-19 shutdown ordering in 05-06
- No blockers; go.mod/go.sum untouched (T-05-SC holds — no package installs)

---
*Phase: 05-grpc-api-and-binary*
*Completed: 2026-11-09*

## Self-Check: PASSED

All six created/modified files exist on disk; commits c2746cb and fc8581b are ancestors of HEAD; `commits: 2` is measured from the persisted ledger (`git rev-list --count e2dc8a3..HEAD`); plan-level gates green at SUMMARY time: `make test` (full race+cover suite) and `go tool golangci-lint run ./...` both exit 0; internal/app/persist untouched (T-05-SC: no package installs, tidy-exact go.mod/go.sum unchanged).
