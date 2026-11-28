---
phase: 03-transport
plan: 04
subsystem: repl-core
tags: [state-store-view, tombstone-visibility, xport-06, xport-11, key-count-flatness, benchmarks, egress-limiter]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-02's Store.Version/Lookup/Versions tombstone-visible view and persist.Engine forwarding"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the deterministic simnet harness and the repl core's rate-limited egress queue"
provides:
  - "repl.StateStore shrunk to ApplyRemote/Version/Lookup/Versions — the whole-store snapshot path is compile-impossible"
  - "listingChunks built from Versions(), still tombstone-inclusive and byte-order sorted"
  - "handleDataRequest through one Lookup; handleDataSync through one Version per entry plus Lookup only for a strictly-newer reply"
  - "BenchmarkHandleDataRequest and BenchmarkHandleDataSync at 1 000 and 50 000 stored keys"
  - "TestCore_keyCountCostFlat: XPORT-06's 2x acceptance as an executable gate"
affects: [03-05, 03-08, 03-09, 03-11, 04-listing, 06-observability]

actuals:
  tokens: 7884
  tasks: 2
  commits: 2
plan_head_before: 5ac646691bfd4858007ae1030d2a22e64e2df044
plan_head_after: b71a72e166c18bfd209acc0e5f7a9da82f67da68

tech-stack:
  added: []
  patterns:
    - "Consumer-side interface shrink as the regression lock: removing Records from repl.StateStore makes the O(store) path fail to compile (T-3-12)."
    - "Per-entry version probe for reconciliation: DataSync compares through Version(key) and reads a record only when the local copy is strictly newer."
    - "Benchmark fixtures build outside the measured loop; one limiter slot of virtual time per iteration drains the egress queue the handler fills."
    - "Ratio gate as acceptance: testing.Benchmark pairs with a fixed 2x bound, warm-ups first, and a failure that names both ns/op values."

key-files:
  created:
    - internal/app/repl/bench_test.go
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/sync_test.go
    - internal/app/internal/simnet/simnet.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/smoke_test.go
    - internal/udprepl/syncsize_internal_test.go
    - internal/grpcapi/wait_integration_test.go

key-decisions:
  - "StateStore's view is the interface contract: Records is gone, so no adapter or handler can silently re-introduce a whole-store deep copy (XPORT-06)."
  - "handleDataSync never calls Versions(): one tombstone-visible Version probe per entry, and Lookup only for a strictly-newer local reply — the O(chunk) shape XPORT-06 measures."
  - "A store read missing between the probe and the reply Lookup degrades to silence; under the single-driver invariant it cannot happen, and silence is the only truthful reply."
  - "Benchmarks pin SyncResponseRate at 1e9/s, so one nanosecond of virtual time is one limiter slot: each iteration drains its own queued reply and the measured loop allocates nothing per response."
  - "The repl_test adapter keeps a test-only Records helper for expected-value construction; the core interface no longer declares it."

patterns-established:
  - "Reconciliation reads scale with the chunk, never the store: probe Version per entry and point-read Lookup only when a reply is due."
  - "The interface shrink is the architecture gate: a stale adapter is a compile error, not a runtime surprise."
  - "Performance acceptance is executable: a benchmark-ratio test with a documented 2x bound and named failure output."

requirements-completed: [XPORT-06, XPORT-11]

coverage:
  - id: D1
    description: "The replication core consumes the tombstone-visible view through a four-method StateStore; the whole-store snapshot path is compile-impossible"
    requirement: XPORT-06
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... (every adapter and call site compiles)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./internal/app/internal/simnet/"
        status: pass
      - kind: other
        ref: "grep -n 'Records()' internal/app/repl/core.go returns no StateStore use; go tool golangci-lint run ./internal/app/repl/ ./internal/udprepl/"
        status: pass
    human_judgment: false
  - id: D2
    description: "Inbound DataRequest/DataSync handling is cost-flat between 1 000 and 50 000 stored keys, proven by the benchmark-ratio gate"
    requirement: XPORT-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestCore_keyCountCostFlat$'"
        status: pass
      - kind: other
        ref: "go test -run '^$' -bench '^BenchmarkHandle(DataRequest|DataSync)$' -benchtime=2x ./internal/app/repl/ renders four sub-benchmark lines"
        status: pass
    human_judgment: false
  - id: D3
    description: "Sync listings stay tombstone-inclusive and byte-order sorted while built from the shallow Versions view"
    requirement: XPORT-11
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/app/repl/ (TestSync_chunking, TestSync_compare, restart and chaos suites)"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 04: Replication core on the cheap store view Summary

**StateStore shrunk to the tombstone-visible view, every sync path re-pointed at it, and a benchmark-ratio gate proving inbound handling flat from 1 000 to 50 000 stored keys**

## Performance

- **Duration:** 10 min
- **Started:** 2026-11-21T18:51:21Z
- **Completed:** 2026-11-21T19:01:21Z
- **Tasks:** 2
- **Files modified:** 9 (1 created, 8 modified)

## Accomplishments

- `repl.StateStore` now declares only `ApplyRemote`, `Version`, `Lookup` and `Versions`; `Records` is gone from the interface, so the per-datagram whole-store deep copy is a compile error rather than a code-review hope (the T-3-12 build gate).
- `listingChunks` builds the sorted, tombstone-inclusive entry sequence from the shallow `Versions()` map, preserving the existing chunking, empty-listing and determinism contracts.
- `handleDataRequest` answers through one point `Lookup`; `handleDataSync` probes `Version(entry.Key)` per listing entry and takes a `Lookup` only when the local record is strictly newer — the exact O(chunk) shape XPORT-06 requires, with `Versions()` never called on the inbound path.
- Every simulator and test adapter (`simnet.nodeStore`, both `stateStore`s, `smokeStore`, `wireStateStore`, `peerStore`) implements the four-method view; a missed adapter would not compile.
- `BenchmarkHandleDataRequest` and `BenchmarkHandleDataSync` measure 1 000- and 50 000-key stores, and `TestCore_keyCountCostFlat` turns XPORT-06's 2x acceptance into an executable gate. Measured under `-race`: DataRequest ratio 1.02 (5 200 vs 5 327 ns/op), DataSync ratio 1.10 (6 738 vs 7 439 ns/op).

## Task Commits

Each task was committed atomically:

1. **Task 1: Shrink StateStore and move every sync path onto the view** — `3727d12` (feat)
2. **Task 2: Key-count flatness gate: DataRequest/DataSync at 1 000 vs 50 000 keys** — `b71a72e` (test)

**Plan metadata:** see the `docs(03-04)` commit that follows this summary.

_Note: no TDD tasks in this plan; each task is one implementation commit._

## Files Created/Modified

- `internal/app/repl/core.go` — four-method `StateStore`; `listingChunks` from `Versions()`; `handleDataRequest` via `Lookup`; `handleDataSync` via per-entry `Version` + conditional `Lookup`; doc comments re-pointed away from the removed snapshot.
- `internal/app/repl/bench_test.go` — new: both handlers benchmarked at 1 000/50 000 keys plus the `TestCore_keyCountCostFlat` ratio gate.
- `internal/app/repl/retransmit_test.go` — `stateStore` gains the three view delegations; test-only `Records` helper retained.
- `internal/app/repl/sync_test.go` — stale whole-store-snapshot doc comments corrected (deviation 1).
- `internal/app/internal/simnet/simnet.go` — `nodeStore` implements the four-method view for the simulator.
- `internal/udprepl/repl_test.go`, `internal/udprepl/smoke_test.go`, `internal/udprepl/syncsize_internal_test.go` — adapters updated to the view.
- `internal/grpcapi/wait_integration_test.go` — `peerStore` updated to the view.

## Decisions Made

- The interface shrink is the regression lock: `Records` left `repl.StateStore` entirely, so the O(store) trap cannot return without a visible interface change.
- `handleDataSync` compares through `Version` per entry and never `Versions()`; a strictly-newer local record is read with a point `Lookup`, so the reply path pays O(record), not O(store).
- A `Lookup` miss after a positive `Version` probe is treated as silence: impossible under the single-driver store invariant, and a truthful no-reply if it ever happened.
- The benchmarks pin `SyncResponseRate` at its documented maximum (1e9/s = 1 ns spacing), letting each iteration's `Advance` drain the reply that iteration queued — no unbounded egress growth distorts the measurement.
- The shared `repl_test` adapter keeps `Records()` as a test-only expected-value helper; the core interface no longer declares it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Stale documentation] Corrected two comments that still described the removed snapshot discipline**
- **Found during:** Task 1 (StateStore shrink)
- **Issue:** `internal/app/repl/sync_test.go` still claimed "One Records() snapshot serves each handler call" and "against ONE store snapshot" — descriptions of behavior this task removed.
- **Fix:** Reworded both comments to the point-`Lookup` / per-entry `Version` discipline.
- **Files modified:** `internal/app/repl/sync_test.go`
- **Verification:** `go vet ./...`, race tests and lint on the touched packages all green.
- **Committed in:** `3727d12` (part of Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 stale documentation)
**Impact on plan:** Comment-only accuracy fix directly caused by the interface change; no behavior, scope or dependency change.

## Issues Encountered

- The first `bench_test.go` draft tripped two lint rules before commit: `noinlineerr` on the seed `Merge` error check and `thelper` on the `testing.Benchmark` closures. Both were fixed by a plain error assignment and by starting each benchmark closure with `b.Helper()`; lint then reported nothing.
- `testing.Benchmark` respects the package's benchmark time (1 s per run), so the flatness test spends roughly 30 s under `-race`; the acceptance criterion mandates warm-ups and a real measurement, so the time is intentional rather than trimmed.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The view consumption half of D-08 is complete: 03-05 and later sync work can call `Version`/`Lookup`/`Versions` on the production `persist.Engine` and every test adapter.
- `TestCore_keyCountCostFlat` is the standing XPORT-06 gate; any future per-chunk `Versions()` call or whole-store snapshot on the inbound path will show up as a failing ratio.
- The tombstone-inclusive listing contract is unchanged and still pinned by `TestSync_chunking`, `TestSync_compare`, restart and chaos suites.

## Self-Check: PASSED

- Files verified present: `internal/app/repl/core.go`, `internal/app/repl/bench_test.go`, `internal/app/repl/retransmit_test.go`, `internal/app/repl/sync_test.go`, `internal/app/internal/simnet/simnet.go`, `internal/udprepl/repl_test.go`, `internal/udprepl/smoke_test.go`, `internal/udprepl/syncsize_internal_test.go`, `internal/grpcapi/wait_integration_test.go`.
- Commits verified in history: `3727d12`, `b71a72e`; `check evaluation-scope --plan 03-04` resolves exactly these two commits.
- Plan verification re-run at close-out: `go build ./...` and `go vet ./...` green; race tests green for `internal/app/repl/`, `internal/udprepl/` and `internal/app/internal/simnet/`; flatness gate green with both ratios logged; lint green on both touched packages; `grep -n "Records()" internal/app/repl/core.go` returns nothing.
- Measured commits: `git rev-list --count 5ac6466..b71a72e` = 2.

---
*Phase: 03-transport*
*Completed: 2026-11-21*
