---
phase: 08-performance
plan: 05
subsystem: replication-cost-gates
tags: [performance, determinism, pending-visits, counting-double, perf-05, tdd]
requires:
  - phase: 08-performance
    provides: the single threshold table (08-01, internal/testutil/perf.go) and the per-key pending index plus PendingVisits seam (08-02)
provides:
  - a counting consumer-seam StateStore double and exact per-datagram store-call vectors at 1 000 and 50 000 stored keys
  - exact-zero PendingVisits gates for an idle Advance, a fresh-key write and a fresh-key apply, plus the repeated-key wiring proof
  - the paired absent/live write-cost benchmark and the perf-gated D-05 ratio
affects: [08-06, 08-07, 08-11, PERF-05, internal/app/repl]
actuals:
  tokens: 9646
  tasks: 3
  commits: 4
plan_head_before: b19a46cfcc0b058eb9b73dcf03552fd4a225168d
plan_head_after: 147a154002c0e353e16daea624b80af4c49542ad
tech-stack:
  added: []
  patterns:
    - "Operation-count gate: a counting consumer-seam double plus exact struct-literal vectors replaces a wall-clock ratio"
    - "PendingVisits delta over a fixed iteration count as the deterministic pending-set cost seam"
    - "Paired absent/live arms in one benchmark and one perf-gated test, with the bound from the single threshold table"
key-files:
  created: []
  modified:
    - internal/app/repl/bench_test.go
    - internal/app/repl/retransmit_test.go
key-decisions:
  - "The key-count gate asserts exact consumer-seam call vectors at both sizes: DataRequest = 1 Version + 1 Lookup, DataSync = 1 Observe + 16 Version, with Versions/Records/ApplyRemote at zero."
  - "The counting double embeds the external test package's delegating stateStore and adds no production surface; a method added to repl.StateStore becomes a compile error in the double."
  - "Task 2's three PendingVisits gates are tripwires that pass on arrival; the phase's RED evidence for the write/apply flatness stays with 08-02's Tests 1-2, exactly as the plan states."
  - "The paired PERF-05 test SKIPs outside make perf, logs the hardware block and both arms, and reads its bound from testutil.PerfAbsentPeerRatio rather than retyping 2.0."
  - "The store-keys and unacked metrics are reported after the measured loop: ResetTimer clears b.extra, so the pre-existing before-ResetTimer reports never reached the result line."
patterns-established:
  - "A deterministic cost claim names the seam it counts and asserts an exact vector per operation, so a reintroduced per-key loop fails by method name instead of by host load."
requirements-completed: [PERF-05]
coverage:
  - id: D1
    description: "The key-count gate is a deterministic seam-call count: DataRequest = 1 Version + 1 Lookup, DataSync = 1 Observe + 16 Version per iteration, identical at 1 000 and 50 000 stored keys with Versions/Records/ApplyRemote at zero"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestCore_keyCountCostFlat$' (PASS in 0.81s under -race)"
        status: pass
    human_judgment: false
  - id: D2
    description: "An idle event-loop iteration inspects zero pending entries at 10 and at 10 000 pending records, with no testing.Benchmark and no wall clock"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestCore_unackedCostFlat$' (PASS)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Fresh-key write and fresh-key inbound replication inspect zero pending entries at both set sizes; the write gate also proves the pending count grows by exactly K and the apply gate proves each record installed"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestCore_notifyWriteCostFlat|TestCore_applyRemoteCostFlat' (PASS)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The visit counter is wired to the real write loops: a newer-version repeated write inspects exactly 2 pending entries"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestCore_pendingVisitsRepeatedKey$' (PASS)"
        status: pass
    human_judgment: false
  - id: D5
    description: "PERF-05's paired ratio: BenchmarkNotifyWriteCost/{absent,live} reports ns/op and pending-visits/op at 10 000 pending 1 KiB records; TestPerfAbsentPeerWriteCost measures both arms back to back, SKIPs in make test and passes under make perf with ratio 0.92 against the 2.0 bound"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 -run '^TestPerfAbsentPeerWriteCost$' ./internal/app/repl/ prints '--- SKIP: TestPerfAbsentPeerWriteCost'"
        status: pass
      - kind: command
        ref: "make perf (SUPKV_PERF=1): '--- PASS: TestPerfAbsentPeerWriteCost', absent 2065 ns/op, live 2233 ns/op, ratio 0.92, 0.00 pending-visits/op in both arms"
        status: pass
    human_judgment: false
duration: 15min
completed: 2026-11-24
status: complete
---

# Phase 08 Plan 05: Deterministic Cost Gates Summary

**The two load-flaky wall-clock ratio gates are now operation counts — a constant set of store calls per datagram and zero pending-entry inspections per hot-path operation — and D-05's absent-peer claim is a paired benchmark that measured 0.92 against the 2.0 bound under `make perf`.**

## Performance
- **Duration:** 15min (approx.)
- **Started:** 2026-11-24T18:17:00Z (approx.; the RED run started at 18:24:31Z)
- **Completed:** 2026-11-24T18:31:52Z
- **Tasks:** 3
- **Files modified:** 2 (0 created, 2 modified)

## Accomplishments

- `retransmit_test.go` now carries `countingStore`, which embeds the delegating `stateStore` and counts every `repl.StateStore` call (`ApplyRemote`, `Version`, `Lookup`, `Versions`, `Observe`, `Generation`) plus the test-only `Records`. A `var _ repl.StateStore = (*countingStore)(nil)` assertion makes the next interface method a compile error, so the count cannot silently become incomplete. The double lives in the external test package; no production surface was added.
- `TestCore_keyCountCostFlat` is an operation-count gate: at 1 000 and 50 000 stored keys it runs 1 000 iterations of the shared benchmark body and asserts exact struct-literal vectors — DataRequest `{Version: 1000, Lookup: 1000}`, DataSync `{Observe: 1000, Version: 16000}` — with `Versions`, `Records` and `ApplyRemote` at zero, plus cross-size equality. Counters are snapshotted after fixture construction, so the fixture's own listing calls are excluded. The gate passes in 0.81 s under `-race` on a host at load ~3.
- The fixture seeds legal versions (`l = testNowMs+i`, counter 0) instead of `uint16(i)` into the 12-bit counter field (03-REVIEW IN-04); the 50 000-key seed spans 49 999 ms, inside the 60 s drift-warn window. `assertSeededVersions` fails by name when a seeded head entry leaves that domain.
- `TestCore_unackedCostFlat` is now an exact-zero `PendingVisits` gate over 1 000 idle `Advance`+`Drain` iterations at 10 and 10 000 pending entries; no `testing.Benchmark`, no wall clock, `t.Parallel()` restored.
- Two new always-on tripwires keep the write and apply paths key-local: `TestCore_notifyWriteCostFlat` (1 000 fresh-key writes, zero visits, pending count +1 000) and `TestCore_applyRemoteCostFlat` (1 000 wire-valid fresh-key replications, zero visits, every apply installed). Both pass on arrival; their comments name 08-02's Tests 1-2 as the RED evidence rather than claiming a transition.
- `TestCore_pendingVisitsRepeatedKey` proves the counter reaches the real write loops: one newer-version repeated write inspects exactly 2 pending entries (guard read plus supersession read).
- `BenchmarkNotifyWriteCost/{absent,live}` reports ns/op, `unacked` and `pending-visits/op` (0.00 in both arms) at 10 000 pending 1 KiB records; the absent arm grows its set each iteration, the live arm drains each measured entry with an Ack from the known peer and asserts the seeded set stayed fixed.
- `TestPerfAbsentPeerWriteCost` runs both arms back to back via `testing.Benchmark`, logs `testutil.HardwareBlock()`, both ns/op values, both visit counts and the ratio, and fails by name above `testutil.PerfAbsentPeerRatio`. It SKIPs in `make test` and PASSed under `make perf` at 0.92 (absent 2065 ns/op, live 2233 ns/op).

## Task Commits

1. **Task 1 RED: failing fixture version-domain gate** — `6acb2a6` (test)
2. **Task 1 GREEN: legal fixture versions** — `4a04ab1` (test)
3. **Task 2: PendingVisits gates** — `d9e9d44` (test)
4. **Task 3: paired benchmark and perf gate** — `147a154` (test)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/app/repl/bench_test.go` — four deterministic gates plus the repeated-key proof, `benchFixtureWith` and the single shared handler bodies, the paired `BenchmarkNotifyWriteCost`, and the perf-gated `TestPerfAbsentPeerWriteCost`.
- `internal/app/repl/retransmit_test.go` — `storeCounts`, `countingStore` (embedded `*stateStore`, seven counters) and its compile-time interface assertion.

## Decisions Made

- **Count the seam, not the clock.** Every cost claim names the seam it measures and asserts an exact count, so a per-key loop or a whole-store read fails by method name and both key counts instead of by host load.
- **One shared body for benchmark and gate.** `benchHandleDataRequestBody`/`benchHandleDataSyncBody` drive both paths, so the gate cannot pass on a path the benchmark does not exercise.
- **The double is test-only and compile-complete.** No production method was added; a future `StateStore` addition fails the double's compile, which is the intended forcing function.
- **Tripwires say they are tripwires.** The write/apply gates' comments attribute the RED evidence to 08-02 and describe themselves as always-on regression guards, per the plan's explicit instruction.
- **ns/op stays for measurement.** Both benchmarks remain for `make bench`/`make perf` reporting; no always-on test calls `testing.Benchmark` for an assertion except the perf-gated paired test, which SKIPs in `make test`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Benchmark metrics were silently cleared by `ResetTimer`**
- **Found during:** Task 1 verification (`store-keys` missing from the benchmark output)
- **Issue:** `b.ReportMetric(...)` calls sat before `b.ResetTimer()`, and `testing.B.ResetTimer` clears `b.extra`; the `store-keys` (and later `unacked`) metric the plan's verify predicates look for never reached the result line. This was pre-existing for `store-keys`; the plan's verify would have failed on any tree.
- **Fix:** Moved each `ReportMetric` call after the measured body (after `ResetTimer`), leaving `ReportAllocs` before it. The metrics now print (`1000 store-keys`, `10/10000 unacked`).
- **Files modified:** `internal/app/repl/bench_test.go`
- **Commit:** `4a04ab1` (store-keys), `d9e9d44` (unacked)

**2. TDD RED shape: assertion failure instead of the plan's "does not exist" build failure**
- **Found during:** Task 1 RED
- **Issue:** The plan's Task 1 behavior says the tests fail "because the counting double and the gate do not exist" — in Go that is a package build failure, which the project's `gsd_run check tdd-red-evidence` classifier refuses as `build_failure`/INVALID_RED.
- **Fix:** The RED commit compiles and fails on the one real defect the task fixes: `TestCore_keyCountCostFlat`'s fixture-domain assertion (`counter=1, l=testNowMs` for the second seed, 03-REVIEW IN-04). The counting double and exact vectors are green on arrival because the handlers are already O(1) — the gate makes that a counted claim. GREEN fixes the seeding. RED validated as `RED_EVIDENCE_OK` with reason `target_test_failed`.
- **Files modified:** `internal/app/repl/bench_test.go`, `internal/app/repl/retransmit_test.go`
- **Commit:** `6acb2a6` (RED), `4a04ab1` (GREEN)

**3. Task 2 carries no RED transition, by the plan's own text**
- **Found during:** Task 2 planning
- **Issue:** Task 2 is marked `tdd="true"`, but its behavior section states that all three gates pass on arrival (08-02 is their dependency) and that the phase's RED evidence for the write/apply flatness belongs to 08-02's Tests 1-2.
- **Fix:** Delivered as one test commit with comments naming the gates as tripwires and pointing at 08-02, rather than fabricating a failing intermediate state.
- **Files modified:** `internal/app/repl/bench_test.go`
- **Commit:** `d9e9d44`

## TDD Gate Compliance

| Gate | Commit | Result |
|------|--------|--------|
| RED (Task 1) | `6acb2a6` test(08-05): add failing fixture version-domain gate | `RED_EVIDENCE_OK` (`target_test_failed`): `TestCore_keyCountCostFlat` failed on `seeded "bench/key-000001" as (l=1730000000000 counter=1 node=1)` |
| GREEN (Task 1) | `4a04ab1` test(08-05): seed legal fixture versions for the count gate | Gate passes in 0.81 s under `-race`; vectors exact at both sizes |
| Task 2 | `d9e9d44` test(08-05): gate pending visits on the idle, write and apply paths | No RED transition by plan design (tripwires, RED evidence in 08-02) |
| Task 3 | `147a154` test(08-05): add paired absent/live write-cost benchmark and perf gate | Perf-gated measurement, no TDD cycle |

The RED evidence record, console output and exit file are under `.planning/phases/08-performance/red-evidence/20261124T182431Z/` (untracked, matching the phase's red-evidence convention).

## Issues Encountered

- The first `make lint` run flagged `golines` (four over-200-column `Fatalf` lines), `thelper` (the `testing.Benchmark` closures) and `unparam` (`assertZeroVisits` always received `benchIterations`). Fixed by splitting the format strings, adding `b.Helper()`, and dropping the constant parameter from the helper signature. `make lint` exits 0 on the whole repo.
- `make perf` skipped `internal/grpcapi` ("no tests to run") and passed the stand and repl rows; no `TestStand_` failure surfaced, so the known F-6 class did not appear.
- Full gates after the work: `make lint` exit 0; `make test` (`go test -race -cover ./...`) exit 0, repl at 93.2 % coverage (57.7 s); `make perf` exit 0 with PERF-05 ratio 0.92.
- Prohibited files untouched: `git diff --name-only b19a46c..HEAD` lists exactly `internal/app/repl/bench_test.go` and `internal/app/repl/retransmit_test.go` — `internal/stand/stand_red_test.go` and `internal/grpcapi/red_gate_test.go` are byte-identical.

## User Setup Required

None - no external service configuration required (the perf row is operator-invoked via `make perf`).

## Known Stubs

None. Every gate asserts its claim on a real core/store path; no placeholder, skipped assertion or unrun verify was left behind.

## Next Phase Readiness

Ready for the rest of wave 2. The repl cost claims are now deterministic and load-independent, and the absent/live ratio has a named bound and a reproducible measurement. PERF-05 is complete in `REQUIREMENTS.md`; the remaining PERF rows (01-04, 06) stay with their owning plans.

## Self-Check: PASSED

- Files: `internal/app/repl/bench_test.go`, `internal/app/repl/retransmit_test.go`, `.planning/phases/08-performance/08-05-SUMMARY.md` all found.
- Commits: `6acb2a6`, `4a04ab1`, `d9e9d44`, `147a154` all ancestors of HEAD.
