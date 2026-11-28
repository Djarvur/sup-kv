---
phase: 05-persistence
plan: 08
subsystem: persistence
tags: [confirmation-timeout, err-unconfirmed, single-claim, waiter-removal, bounded-flush, bounded-shutdown, rpc-totality, metrics, prometheus, synctest]
requires:
  - phase: 05-persistence
    provides: the injected Config.Now/After timer seam and the bounded dump-failure ladder from 05-07
  - phase: 05-persistence
    provides: the write-confirmation engine, its waiter slice and the flush seam from Phase 3/4
  - phase: 04-protocol
    provides: the PROTO-08 write-notify totality and the RPC error-mapping surface
affects: [persistence, durability, rpc-contract, shutdown, observability, p09, p10, p12]
actuals:
  tokens: 14191
  tasks: 3
  commits: 6
plan_head_before: 2107ac377f82c8a9c53ec15cac780039389ffd69
plan_head_after: 8da1a1957d7d19c922f111582cd0762ea5f7c350
tech-stack:
  added: []
  patterns:
    - "Single-claim waiter resolution: every resolver sets claimed under the engine mutex before writing err and closing; capture alone never claims, so a hung dump cannot block the timeout"
    - "Bounded blocking seams: every wait selects on done/runDone and a deadline armed through Config.After"
    - "Unconfirmed is a durability answer, not a failure: a distinct sentinel, its own counter, an Unavailable mapping and the same replication notify as a dump failure"
key-files:
  created:
    - internal/app/persist/timeout_test.go
    - internal/app/persist/timeout_internal_test.go
  modified:
    - internal/app/persist/engine.go
    - internal/app/persist/errors.go
    - internal/grpcapi/server.go
    - internal/grpcapi/server_test.go
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
key-decisions:
  - "A waiter is claimed at resolution, never at capture: the dump's captured slice does not own it, so a dump hanging after capture cannot keep the caller's timeout from resolving the write"
  - "The timeout branch claims under the mutex, stores ErrUnconfirmed, removes its waiter with slices.DeleteFunc and increments an atomic counter; the record's durability rests on runDump's capture-before-snapshot proof, so no re-dirty is needed — a late success saves it, a late failure re-dirties through failWaiters, and a permanently hung dumper blocks any dump anyway"
  - "Flush() is FlushTimeout(0) resolved to ConfirmTimeout; one cfg.After channel bounds both the flush-channel send and the reply, and expiry returns the wrapped ErrUnconfirmed"
  - "Restore returns its partial replay list under a timeout and keeps ErrUnconfirmed distinguishable from ErrDumpFailed: only true dump failures get the ErrDumpFailed wrap"
  - "The binary passes shutdownGrace to FlushTimeout and bounds the engineDone wait with time.After(shutdownGrace), logging one static WARN and abandoning the goroutine; a flush error keeps exit code 1"
  - "ErrUnconfirmed is total on the RPC path: both write branches notify an unconfirmed-installed record and mapError answers Unavailable with the sentinel's own message"
  - "supkv_dumps_unconfirmed_total is independent of the failure counters: a confirmation timeout says nothing about the dump's outcome"
patterns-established:
  - "Single-claim channel resolution behind a mutex with a claimed flag; every late resolver skips the already-claimed waiter"
  - "Deadline-bounded engine calls: a write, a flush, a snapshot and a restore each carry an explicit timeout outcome"
  - "Virtual-time proof of exact deadlines with a hanging fake port; no real I/O inside a synctest bubble"
requirements-completed: [PERSIST-08]
coverage:
  - id: D1
    description: "Config.ConfirmTimeout defaults to 10 s; a hung dump resolves Set with ErrUnconfirmed at exactly the deadline, the record stays installed, the waiter is removed (memory bounded) and the counter increments; a late successful dump finds no waiter, double-closes nothing and still saves the record"
    requirement: "PERSIST-08"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_(confirmationTimeout|timeoutThenDumpSuccess|waiterRemoved)$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Flush and FlushTimeout bound both flush waits and return ErrUnconfirmed on expiry; Snapshot copies nothing when the forced flush expires; Restore returns its partial replay list with an error that matches ErrUnconfirmed and never ErrDumpFailed"
    requirement: "PERSIST-08"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^Test(Flush|Snapshot|Restore)_timeout'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The binary's shutdown calls FlushTimeout(shutdownGrace), then waits at most shutdownGrace for engineDone and logs one WARN instead of hanging when the dumper outlives it; a flush error keeps exit code 1"
    requirement: "PERSIST-08"
    verification:
      - kind: process
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestShutdown_'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestFlush_timeout$' (the deadline mechanism the binary calls)"
        status: pass
    human_judgment: true
    rationale: "The engineDone-expiry WARN/abandon branch cannot be fault-injected through the real persistfile adapter (no stuck-fsync seam); the normal bounded path is process-tested and the deadline mechanism is unit-proven, so a verifier should classify the abandon branch by review."
  - id: D4
    description: "A timed-out Set and Delete notify the installed record with track=false before answering Unavailable, mapError maps ErrUnconfirmed to Unavailable with the sentinel message, and a closed engine still notifies nothing"
    requirement: "PERSIST-08"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_unconfirmed'"
        status: pass
    human_judgment: false
  - id: D5
    description: "supkv_dumps_unconfirmed_total renders at zero on an idle engine, carries DumpStats.Unconfirmed by effect, and appears in the running binary's scrape"
    requirement: "PERSIST-08"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/metrics/ -run '^TestMetrics_'"
        status: pass
      - kind: process
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMetrics_optIn$'"
        status: pass
    human_judgment: false
duration: 15min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 08: Bounded Confirmation, Flush and Shutdown Summary

**A hung fsync can no longer park a write, a snapshot, a restore or the process: every confirmation wait carries a 10 s default deadline and an explicit `ErrUnconfirmed` outcome decided by a single claim, `FlushTimeout` bounds shutdown, a timed-out write still replicates, and `supkv_dumps_unconfirmed_total` makes the class observable.**

## Performance

- **Duration:** 15min
- **Started:** 2026-11-22T15:48:56Z
- **Completed:** 2026-11-22T16:04:20Z
- **Tasks:** 3
- **Files modified:** 10 (2 created, 8 modified)

## Accomplishments

- `persist.Config.ConfirmTimeout` defaults to 10 s in `normalized()`, and `ErrUnconfirmed` joins the sentinel block with its "installed, not yet durable, replicate like a dump failure" contract. `await` now selects between the waiter and a `cfg.After(ConfirmTimeout)` channel instead of blocking on `<-w.done` forever.
- `waiter.claimed` carries the single-claim protocol. `confirmWaiters`, `failWaiters`, `sweepClosed` and the timeout branch all set it under the engine mutex before writing `err` and closing; a resolver finding it set skips the waiter untouched. Capture alone never claims, so a dump that hangs after capture cannot block the caller's timeout. The timeout branch removes its entry with `slices.DeleteFunc`, counts the outcome, and returns `ErrUnconfirmed`; a later successful dump finds nothing to double-close and still saves the record.
- `DumpStats.Unconfirmed` is backed by an `atomic.Uint64` written by the awaiting goroutines and read lock-free by the exporter.
- `Flush()` is now `FlushTimeout(0)`; both bound the flush-channel send and the reply with one deadline and return `fmt.Errorf("%w: flush", ErrUnconfirmed)` on expiry. `Snapshot` flushes under the confirmation timeout and copies nothing on expiry. `Restore` keeps its partial replay list and returns `ErrUnconfirmed` distinguishable from `ErrDumpFailed`.
- The binary's shutdown calls `FlushTimeout(shutdownGrace)` and then rides a `select` between `engineDone` and `time.After(shutdownGrace)`, logging one static WARN and abandoning a dumper still stuck in the kernel. A flush error still exits 1.
- The RPC path is total over the new class: `Set` and `Delete` notify the installed record for `ErrDumpFailed` or `ErrUnconfirmed` (track=false), and `mapError` maps `ErrUnconfirmed` to `Unavailable` with the sentinel message. The empty-capture closed-engine test stays green.
- `supkv_dumps_unconfirmed_total` joins `dumpFamilies`, renders at zero, carries `Unconfirmed` by effect, and appears in the binary scrape list.

## Task Commits

1. **Task 1 RED: confirmation-timeout engine tests** — `a034ad9` (test)
2. **Task 1 GREEN: single-claim confirmation timeout** — `5fd09c6` (feat)
3. **Task 2 RED: flush/snapshot/restore timeout tests** — `999e69d` (test)
4. **Task 2 GREEN: bounded flush and shutdown** — `ed4ce4f` (feat)
5. **Task 3 RED: unconfirmed RPC and metric tests** — `62e7c8b` (test)
6. **Task 3 GREEN: RPC totality and the unconfirmed family** — `8da1a19` (feat)

**Plan metadata:** the `docs(05-08)` completion commit (this commit).

## Files Created/Modified

- `internal/app/persist/engine.go` — `defaultConfirmTimeout`, `Config.ConfirmTimeout`, `waiter.claimed`, the claim checks in all four resolvers, `await`'s timeout branch, `FlushTimeout`, the duration-parameterized `flush`, `DumpStats.Unconfirmed`, and the bounded `Snapshot`/`Restore` calls.
- `internal/app/persist/errors.go` — the `ErrUnconfirmed` sentinel.
- `internal/app/persist/timeout_test.go` — the hanging port and the write/flush/snapshot/restore deadline tests.
- `internal/app/persist/timeout_internal_test.go` — the white-box waiter-removal proof (the slice is not public API).
- `internal/grpcapi/server.go` — the `ErrUnconfirmed` notify branch in `Set`/`Delete` and the `Unavailable` mapping plus doc table.
- `internal/grpcapi/server_test.go` — the dump hold (hung-fsync stand-in), the confirmation-timeout harness knob, and the two unconfirmed RPC tests.
- `internal/metrics/metrics.go`, `metrics_test.go` — `supkv_dumps_unconfirmed_total`, its zero and by-effect samples.
- `cmd/sup-kv/main.go`, `main_test.go` — the bounded shutdown (`FlushTimeout` + `engineDone` grace WARN) and the binary scrape family list.

## Decisions Made

- **Claim at resolution, not at capture.** The dump's captured slice is not ownership; that is what lets the timeout branch win even while the dump hangs after capture.
- **No re-dirty on timeout.** The record is installed before the waiter is registered and `runDump` captures before taking `Records()`, so the in-flight dump's snapshot contains it. A late success persists it; a late failure re-dirties through `failWaiters`; a permanently hung dumper could not dump again regardless. The timeout's job is to bound the caller, and its removal keeps memory bounded.
- **One deadline for both flush waits.** `flush(d)` arms a single `cfg.After(d)` channel and races it against both the `flushCh` send and the reply, so the bound covers the whole operation rather than each half separately.
- **`ErrUnconfirmed` is never rewritten as `ErrDumpFailed`.** `Restore` wraps only true dump failures; the new class passes through so callers and `mapError` can tell an unknown outcome from a failed one.
- **The unconfirmed counter is independent.** It is not a subset of `dumps_errors_total`: the dump may still succeed after the client was answered.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Compile-coupled RED] Structural vocabulary landed with the RED commits**
- **Found during:** Tasks 1-3 RED
- **Issue:** A target test naming `ErrUnconfirmed`, `Config.ConfirmTimeout`, `DumpStats.Unconfirmed`, `FlushTimeout`, the fake's hold channel or the harness timeout knob cannot compile without those declarations, and a build failure is not valid RED evidence.
- **Fix:** The RED commits carried only the structural minimum (sentinel, config/stat fields, a `FlushTimeout` stub delegating to the unbounded `flush`, and test-only fake/harness fields). Every target test then compiled and failed on its planned assertion — the unbounded wait, the Internal mapping, the missing family.
- **Files modified:** `internal/app/persist/errors.go`, `engine.go`, `internal/grpcapi/server_test.go`
- **Verification:** `RED_EVIDENCE_OK` for all three records (`.planning/phases/05-persistence/red-05-08-task{1,2,3}.json`).
- **Commit:** `a034ad9`, `999e69d`, `62e7c8b`

**2. [Rule 3 - Lint/test shape] White-box waiter test needs the `_internal_test.go` suffix**
- **Found during:** Task 1 RED
- **Issue:** The plan said "expose an unexported accessor or use an internal test file". The unexported `waiters` slice is only reachable from `package persist`, and the strict `testpackage` lint only exempts files named `*_internal_test.go`.
- **Fix:** `TestEngine_waiterRemoved` lives in `timeout_internal_test.go` (the plan-sanctioned internal file); the behavioral deadline tests stay in the external `timeout_test.go`. The artifact row naming `timeout_test.go` covers the suite; the white-box file is its documented companion.
- **Files modified:** `internal/app/persist/timeout_internal_test.go`
- **Verification:** `make lint` clean; the test passes.
- **Commit:** `a034ad9` (RED), `5fd09c6` (GREEN)

### Plan-artifact notes

- `internal/app/persist/engine_blocking_test.go` is listed in `files_modified` but was deliberately not edited: its 200 ms/1 s windows sit far below the 10 s default and the prohibition keeps them byte-identical. `git diff plan_head_before..HEAD -- internal/app/persist/engine_blocking_test.go` is empty and the test is green.
- `cmd/sup-kv/main_test.go` gained only the scrape-list row for the new family. The engineDone-expiry WARN branch has no process-level fault injection (the real adapter exposes no stuck-fsync seam); its deadline mechanism is unit-proven and the normal shutdown path is process-tested.

## TDD Gate Compliance

Each task ran the full RED → GREEN cycle; no refactor commit was needed.

| Task | RED evidence | RED failure | GREEN |
|------|--------------|-------------|-------|
| 1 | `red-05-08-task1.json` (`RED_EVIDENCE_OK`) | Set never returned at the deadline — the wait was unbounded | `5fd09c6` |
| 2 | `red-05-08-task2.json` (`RED_EVIDENCE_OK`) | Flush, Snapshot and Restore blocked past their guard — the flush waits were unbounded | `ed4ce4f` |
| 3 | `red-05-08-task3.json` (`RED_EVIDENCE_OK`) | The RPC answered Internal and notified nothing — the class was not total | `8da1a19` |

Every RED target compiled and failed on its planned assertion; no build, fixture or zero-test failure was accepted.

## Issues Encountered

None. `make lint` is clean and `go test -race -cover ./...` is green, including the 218 s `internal/app/repl` suite and the process-level stand tests. The frozen `TestEngine_blockingSaveHoldsSet` stayed green and unedited.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The confirmation deadline, the single-claim protocol, the bounded flush and the bounded shutdown are complete; plan 05-09 can split the engine into its five internal roles knowing `await`, the claim protocol and the timeout's counter are pinned by tests.
- `FlushTimeout(d)` is the seam 05-10's OPERATIONS text and any later configuration knob (`--confirm-timeout`) can build on.
- Ready for 05-09.

## Self-Check: PASSED

- Created files verified on disk: `internal/app/persist/timeout_test.go`, `internal/app/persist/timeout_internal_test.go`, `.planning/phases/05-persistence/05-08-SUMMARY.md`.
- Task commits verified as ancestors of HEAD: `a034ad9`, `5fd09c6`, `999e69d`, `ed4ce4f`, `62e7c8b`, `8da1a19`.

