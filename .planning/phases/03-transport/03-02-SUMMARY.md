---
phase: 03-transport
plan: 02
subsystem: storage
tags: [store-read-view, tombstones, lww-store, simnet, wr-02, xport-11, xport-06, xport-03]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: "the LWW storage store, the persist engine composition path and the simnet cluster harness"
provides:
  - "Store.Version(key), Store.Lookup(key) and Store.Versions() — the cheap tombstone-visible read view"
  - "storage.MaxKeyBytes = 1024, the single documented key-length ceiling for XPORT-03"
  - "Engine.Lookup/Version/Versions forwarding so the production composition path serves the view"
  - "WR-02 fixed: simulated writes can no longer silently disappear and copy no whole store"
affects: [03-04-repl-core, 03-08, 03-09, 03-11, 06-listing]

actuals:
  tokens: 6438
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Tombstone-visible read view beside a live-only API read: Lookup/Versions see tombstones while Get stays ErrNotFound"
    - "Caller-owned store reads: Lookup clones Data and Versions builds a fresh map, never a view of store internals"
    - "Cheap per-entry version probe (Version) for sync comparisons, avoiding O(store) Records() snapshots per datagram"
    - "Test-harness writes fail loudly: an oversize simulated write panics with operation/key/node instead of vanishing"

key-files:
  created:
    - internal/app/storage/view_test.go
  modified:
    - internal/app/storage/store.go
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - internal/app/internal/simnet/simnet.go
    - internal/app/internal/simnet/simnet_ext_test.go

key-decisions:
  - "The view is tombstone-visible by construction: Version/Lookup/Versions include deleted records, while Get stays the live-only API read (D-08's store half, XPORT-11)."
  - "Every read hands out caller-owned state: Lookup clones Data, Versions builds a fresh map; no alias of store internals escapes (T-3-04)."
  - "MaxKeyBytes documents write-side validation only: pre-upgrade long keys stay readable, and enforcement lands on Set/Delete and the receive path (XPORT-03, 03-08/03-09)."
  - "Cluster.Write panics on a rejected store write — the simulator is test infrastructure, and a silently dropped write would invalidate every model comparison (WR-02, T-3-05)."

patterns-established:
  - "Read-view split: the API read (Get) and the replication read (Lookup) are separate methods with separate tombstone semantics."
  - "Store reads document their locking cost: one RLock, stated copy/no-copy behavior, so callers can price a listings path."
  - "Simulator author sequences read the just-written record with a point Lookup, never a whole-store Records() copy."

requirements-completed: [XPORT-11, XPORT-06]

coverage:
  - id: D1
    description: "Store exposes a tombstone-visible key→version view, point read and shallow snapshot with caller-owned results"
    requirement: XPORT-11
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestStore_version$|^TestStore_lookup_|^TestStore_versions_|^TestStore_view_concurrentHammer$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "storage.MaxKeyBytes = 1024 is the single documented key-length ceiling (datagram-ceiling rationale)"
    requirement: XPORT-03
    verification:
      - kind: unit
        ref: "go tool golangci-lint run ./internal/app/storage/"
        status: pass
    human_judgment: false
  - id: D3
    description: "persist.Engine forwards Lookup/Version/Versions while Get remains live-only"
    requirement: XPORT-11
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_lookup_tombstoneVisible$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "WR-02: an oversize simulated write panics loudly and the write log matches the store's point-read record"
    requirement: XPORT-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_write_refusesOversize$|^TestSim_writeLogRecordMatchesStore$'"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestChaos_convergence$|^TestScenario_'"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 02: Store read view and WR-02 Summary

**Tombstone-visible `Version`/`Lookup`/`Versions` store reads with caller-owned copies, `MaxKeyBytes = 1024`, Engine forwarding, and a simulator write path that panics instead of swallowing a rejected write**

## Performance

- **Duration:** 5 min
- **Started:** 2026-11-21T18:32:43Z
- **Completed:** 2026-11-21T18:37:40Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- `storage.Store` gained the three cheap reads XPORT-11 requires: `Version(key)` (per-entry version probe, no Data copy), `Lookup(key)` (tombstone-visible point read with a cloned `Data`) and `Versions()` (fresh shallow key→version map). `Get` is untouched and stays live-only.
- `storage.MaxKeyBytes = 1024` landed beside the value limit, with the 65 507-byte datagram-ceiling rationale recorded in its doc comment — the constant XPORT-03's API and receive-path checks will consume.
- `persist.Engine` forwards `Lookup`/`Version`/`Versions`, so the production composition path satisfies the new consumer-side interface without extra locking or copying.
- WR-02 is fixed and pinned: `Cluster.Write` panics naming the operation, key and node when the store rejects a write, and both `Write` and `Delete` read the written record through the point `Lookup` instead of copying the whole store. Chaos and scenario suites stay green.

## Task Commits

Each task was committed atomically:

1. **Task 1: Store key→version view, tombstone-visible point read, and the key-limit constant** — `cb4dd1f` (feat)
2. **Task 2: Engine forwards the view reads** — `447835e` (feat)
3. **Task 3: WR-02 — simnet Write/Delete stop swallowing store errors and use Lookup** — `b74ea67` (fix)

**Plan metadata:** see the `docs(03-02)` commit that follows this summary.

_Note: no TDD tasks in this plan; each task is one implementation commit._

## Files Created/Modified

- `internal/app/storage/store.go` — added `MaxKeyBytes`, `Version`, `Lookup`, `Versions` with RLock-only costs and copy contracts.
- `internal/app/storage/view_test.go` — new: version probe, tombstone visibility, snapshot isolation, and a concurrent read/write hammer asserting per-call record integrity.
- `internal/app/persist/engine.go` — added `Lookup`/`Version`/`Versions` pass-throughs; `Get`/`Records` unchanged.
- `internal/app/persist/engine_test.go` — new `TestEngine_lookup_tombstoneVisible` pinning the engine's view forwarding against its live-only `Get`.
- `internal/app/internal/simnet/simnet.go` — `Cluster.Write` error check and panic, `Lookup` reads in `Write`/`Delete`, `checkIndex` on both entry points.
- `internal/app/internal/simnet/simnet_ext_test.go` — new WR-02 regression pins: oversize write panic and write-log/store record agreement.

## Decisions Made

- Tombstones are visible through the view by construction; `Get` keeps its live-only `ErrNotFound` contract so the API read cannot regress.
- The view never leaks store internals: `Lookup` clones `Data` and `Versions` returns a fresh map (threat T-3-04).
- `MaxKeyBytes` is documented as write-side validation only — reads stay permissive for pre-upgrade long keys; enforcement belongs to 03-08/03-09.
- The simulator fails loudly on a rejected write (panic naming operation/key/node), because a silently dropped write would make every model comparison lie (WR-02, threat T-3-05).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Routed `Cluster.Write`/`Delete` through `checkIndex`**
- **Found during:** Task 3 (WR-02 fix)
- **Issue:** Both entry points indexed `c.nodes[i]` without a bounds check, so an out-of-range node panicked with a raw slice-bounds message instead of the harness's named panic — the second half of the WR-02 review finding.
- **Fix:** Added `c.checkIndex(i, "Write")` and `c.checkIndex(i, "Delete")` before touching the node, matching `ReplaceNodeWithState`/`SetRecvBudget`.
- **Files modified:** `internal/app/internal/simnet/simnet.go`
- **Verification:** `go test -race ./internal/app/internal/simnet/` green; the WR-02 regression tests pass.
- **Committed in:** `b74ea67` (part of Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** In-scope completion of the WR-02 review finding (the plan cited the `checkIndex`/harness convention). No scope creep.

## Issues Encountered

- The plan commit ledger file `.git/gsd-plan-head-before-03-02` was a stale artifact from Nov 3 (a previous milestone's `03-01`, base `d93e75a`); counting from it would have reported 128 commits. It was reset to this plan's true start HEAD `4321ad9` before the first task commit and the reset is reflected in the measured `commits: 3`.
- Two initial compile errors in `view_test.go` came from slicing map-index `RecordVersion` values (not addressable); fixed by binding each to a local. Lint then flagged shadowed `ok` names and inline error handling; both fixed before the task commit.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The store half of D-08 is in place: 03-04's listing, `DataRequest` and `DataSync` paths can consume `Versions`, `Lookup` and `Version` without a `Records()` deep copy per datagram.
- `storage.MaxKeyBytes` is ready for 03-08/03-09 to enforce on write and receive paths; reads remain permissive by design.
- WR-02's loud-failure contract is pinned, so simulator model comparisons can trust the write log; `nodeStore.Records` remains only until 03-04 shrinks `repl.StateStore`.

## Self-Check: PASSED

- Files verified present: `internal/app/storage/store.go`, `internal/app/storage/view_test.go`, `internal/app/persist/engine.go`, `internal/app/persist/engine_test.go`, `internal/app/internal/simnet/simnet.go`, `internal/app/internal/simnet/simnet_ext_test.go`.
- Commits verified in history: `cb4dd1f`, `447835e`, `b74ea67`.
- Plan verification re-run at close-out: storage/persist/simnet `-race` green, repl chaos+scenario green, lint on all three packages green, `go mod tidy` diff empty.
- Measured commits: `git rev-list --count 4321ad9..b74ea67` = 3.

---
*Phase: 03-transport*
*Completed: 2026-11-21*
