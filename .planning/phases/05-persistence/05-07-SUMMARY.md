---
phase: 05-persistence
plan: 07
subsystem: persistence
tags: [injected-clock, timer-seam, retry-ladder, backoff, failure-classes, eacces, process-tests, metrics, prometheus, simulator, notify-installed, tie-break, determinism]
requires:
  - phase: 05-persistence
    provides: the repl core's time-as-argument seam precedent and the D-11 simulator outline
  - phase: 05-persistence
    provides: the adapter's save-error classification and port.FailureClass from 05-05
  - phase: 05-persistence
    provides: the equal-version content law and its deferred cross-node proof from 05-03
affects: [persistence, observability, failure-decay, metrics-surface, simulator, p08, p09, p12]
actuals:
  tokens: 19366
  tasks: 3
  commits: 6
plan_head_before: ad599ed5f84d877d008a9e5fbb0af434506a1c24
plan_head_after: 10e1eac718c8a9dc66426708e8da340fecda4404
tech-stack:
  added: []
  patterns:
    - "Channel-held timers: every timer is a <-chan time.Time in a loop variable, replaced on re-arm, never created inside the select"
    - "Ladder state fixed at the failure instant: retryDelay + retryAt, so intervening writes cannot postpone the next attempt"
    - "One callback per failed attempt: OnDumpFailure(err, class) on the dumper goroutine, wired by the binary to one static slog line"
    - "Class split beside the total: DumpStats.Errors stays the failed-attempt total, DiskFull/Permission are subsets; the exporter derives other"
    - "Simulator engine harness: Config.Now = Scheduler.Now and Config.After = ScheduleAfter, with an activity-counted settle before every virtual-time step"
    - "Notify-after-install: Cluster.NotifyInstalled runs the production notification without re-Setting the engine-stamped record"
key-files:
  created:
    - internal/app/persist/backoff_test.go
    - internal/app/repl/sim_dump_test.go
  modified:
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - internal/app/internal/simnet/simnet.go
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - internal/stand/persist_test.go
key-decisions:
  - "Config.Now/After default to time.Now/time.After only inside normalized(); every other engine read and every timer goes through the injected pair, so the simulator owns the dump loop's time"
  - "A failed dump doubles 100 ms to a 5 s cap and fixes the retry instant at the failure; a success resets the ladder, and a pending retry suppresses the debounce/deadline channels so a 1 s MaxWait cannot defeat the 5 s cap"
  - "The debounce channel is replaced on every re-arm and is therefore not passed into rearm; the retry channel keeps its already-armed instant while pending so a write during backoff registers a waiter the retry covers"
  - "DumpStats.Errors remains the failed-attempt total while DiskFull/Permission are class subsets; the exporter renders other as the clamped residual so every scrape carries the whole vocabulary"
  - "Health is untouched: the SERVING transition stays gate-only, and a persistent permission failure surfaces as Unavailable writes, one ERROR per attempt and the class counter"
  - "The simulator's Scheduler takes a lock (Schedule/ScheduleAfter/Now) because the engine arms timers from its own goroutine; RunUntil remains the single driver and releases the lock around callbacks"
  - "Cluster.NotifyInstalled is lookup + NotifyLocalWrite + route + drive + write-log capture, never Set, so the engine's stamp and the confirmed record are not replaced"
  - "Exact ladder instants stay in synctest; the simulator pins the interaction (confirmation before notification, failure still notifies, convergence) with a settle driver that never advances virtual time mid-step"
requirements-completed: [PERSIST-05, PERSIST-12]
coverage:
  - id: D1
    description: "The dumper's clock and timers are injectable: Config.Now/Config.After default to the wall clock only in normalized(), a supplied pair serves every read and timer, a Set confirms exactly at the Debounce virtual instant, and no production file in internal/app/persist calls time.Now/NewTimer/After/Until/Since outside those two defaults"
    requirement: "PERSIST-12"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_(injectedClock|confirmDebounceTiming)$'"
        status: pass
      - kind: build
        ref: "grep -rn 'time\\.Now\\|time\\.NewTimer\\|time\\.After\\|time\\.Until\\|time\\.Since' internal/app/persist/*.go (production hits = 0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A persistent dump failure walks the bounded ladder: 100 ms doubling to the 5 s cap (attempts at +100/+300/+700/+1500/+3100/+6300/+11300 ms after the first failure), a write registered during backoff is covered by the recovering attempt, and a success resets the ladder to its 100 ms base"
    requirement: "PERSIST-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_backoffLadder$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "Each failed attempt invokes OnDumpFailure exactly once with the underlying error and the adapter's port.FailureClass; DumpStats counts DiskFull/Permission beside the totals; supkv_dumps_failures_total{class} renders disk_full/permission/other; main logs one static ERROR with the class; and the EACCES process scenario asserts Unavailable, one permission ERROR, SERVING health, a bounded attempt rate and chmod-back recovery (root skips)"
    requirement: "PERSIST-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_dumpFailureClasses$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/metrics/ -run '^TestMetrics_'"
        status: pass
      - kind: process
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_persistDumpFailure$'"
        status: pass
      - kind: process
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMetrics_optIn$' (the binary scrape carries the family)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The simulator drives a real persist.Engine over a cluster node's store on virtual time: a confirmed write is announced after confirmation and converges the peer, a dump-failure window returns ErrDumpFailed yet still notifies while the fake's attempts follow the ladder, the shrunk-limit skip reports LoadStats.Decoded.ValueLimit == 1, and two same-seed runs produce identical attempt instants and final records"
    requirement: "PERSIST-12"
    verification:
      - kind: sim
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_dumpNotify$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "The cross-node equal-version half of the (version, content) law deferred by 05-03: two nodes seeded with the same key and version but different Data converge to the law's winner after Cluster.NotifyInstalled announces the record, in both directions, with a DataReplication on the wire and none from the merge-only seed"
    requirement: "PERSIST-12"
    verification:
      - kind: sim
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_tiebreakConverges$'"
        status: pass
    human_judgment: false
duration: 19min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 07: Injected Dump Time, Bounded Failure Decay and the Dump-to-Notify Simulator Summary

**The dumper now reads time only from `Config.Now`/`Config.After`, a persistent dump failure decays 100 ms → 5 s instead of busy-looping, every failed attempt logs once with its port class and publishes `supkv_dumps_failures_total`, and the simulator proves dump→confirmation→replication-notify — failure window and skip included — with the cross-node equal-version law converging through `Cluster.NotifyInstalled`.**

## Performance

- **Duration:** 19min
- **Started:** 2026-11-22T15:18:38Z
- **Completed:** 2026-11-22T15:37:12Z
- **Tasks:** 3
- **Files modified:** 11 (2 created, 9 modified)

## Accomplishments

- `persist.Config` gained `Now`, `After` and `OnDumpFailure`. `normalized()` fills the two clock funcs with the wall clock — now the package's only wall-clock assignments — and `Run` holds the debounce, deadline and retry channels in loop variables, replacing them on every re-arm. `time.NewTimer`, `time.Until`, `time.Since`, `timerChan` and `stopTimer` are gone; `runDump`, `failWaiters`, `await`, `markDirty` and the deadline arming all read `cfg.Now()`.
- The retry ladder is `nextRetry`: 100 ms doubling to a 5 s cap, fixed at the failure instant (`retryAt`) so a write during backoff registers a waiter instead of postponing the next attempt. While a retry is pending, `rearm` arms only the retry channel — the 1 s `MaxWait` deadline cannot fire earlier and defeat the cap. A success resets the ladder and covers the waiters.
- Each failed attempt runs `handleDumpFailure` once: the `port.FailureClassOf` class increments `DumpStats.DiskFull`/`Permission` (with `Errors` staying the total), advances the ladder, and invokes `OnDumpFailure(err, class)`. `main` wires that to one `slog.Error("Dump failed", "class", class, "detail", err)`.
- `supkv_dumps_failures_total{class}` publishes `disk_full`, `permission` and `other` (the clamped residual of the error total); the zero state and the by-effect values are both pinned. Health is untouched — the EACCES process scenario proves SERVING throughout, one permission ERROR, a bounded rate in a one-second read-only window (3 retries vs the old ~20), a non-zero class metric and chmod-back recovery.
- The simulator now hosts a real engine: `Scheduler` gained a lock plus `ScheduleAfter`, `Cluster.NotifyInstalled` is the additive notification seam (Lookup + NotifyLocalWrite + route + drain + drive + write log, never Set), and `sim_dump_test.go` drives confirmation at the exact debounce instant, the failure window's `ErrDumpFailed` + still-notify + ladder, the shrunk-limit report, same-seed replay, and both directions of the cross-node equal-version convergence.

## Task Commits

1. **Task 1 RED: injected clock and timer seam** - `1cc4169` (test)
2. **Task 1 GREEN: Config.Now/After and channel-held timers** - `6051876` (feat)
3. **Task 2 RED: retry ladder, failure classes and EACCES process proof** - `eadc703` (test)
4. **Task 2 GREEN: bounded ladder, class counters and the failure family** - `de50de1` (feat)
5. **Task 3 RED: dump-to-notify simulator scenario** - `8d8735e` (test)
6. **Task 3 GREEN: Cluster.NotifyInstalled** - `10e1eac` (feat)

**Plan metadata:** the `docs(05-07)` completion commit (this commit).

## Files Created/Modified

- `internal/app/persist/engine.go` — `Config.Now/After/OnDumpFailure`, the channel-held timer loop, `rearm`'s retry preference, `nextRetry`/`resetRetry`/`handleDumpFailure`, the `retryGrowth` constant and the `DumpStats` class fields.
- `internal/app/persist/engine_test.go` — `recordingSeam`, `TestEngine_injectedClock`, and the fake port's attempt-instant recording plus `failErr`.
- `internal/app/persist/backoff_test.go` — `TestEngine_backoffLadder` and `TestEngine_dumpFailureClasses` over the synctest bubble.
- `internal/app/internal/simnet/simnet.go` — the Scheduler lock and `ScheduleAfter`, the package doc update, and `Cluster.NotifyInstalled`.
- `internal/app/repl/sim_dump_test.go` — `virtualAfter`, the engine-over-cluster harness with the activity-counted settle driver, `TestScenario_dumpNotify` and `TestScenario_tiebreakConverges`.
- `internal/metrics/metrics.go`, `metrics_test.go` — the `supkv_dumps_failures_total` family, its zero and by-effect samples.
- `cmd/sup-kv/main.go`, `main_test.go` — the `OnDumpFailure` slog wiring and the binary-scrape family list.
- `internal/stand/persist_test.go` — `TestStand_persistDumpFailure` and the `persistSetErr` helper.

## Decisions Made

- **`retryAt` fixes the ladder instant.** A failure computes `delay` and `failureInstant+delay` under one mutex hold; `rearm` arms from that instant, so the `changed` nudges of writes during backoff cannot push the retry out.
- **A pending retry suppresses the debounce and deadline.** `MaxWait` (1 s) is below the 5 s cap, so arming the deadline during backoff would defeat the ladder — the ladder owns the next attempt while non-zero.
- **Class counters beside the total.** `Errors` stays the failed-attempt total (the existing family and tests stay meaningful); `DiskFull`/`Permission` are subsets and the exporter derives `other` as the clamped residual.
- **The scheduler takes a lock, RunUntil stays the driver.** The engine arms timers from its goroutine; `Schedule`/`ScheduleAfter`/`Now` are safe from any goroutine, and `RunUntil` releases the lock around each callback so handlers may schedule.
- **Exact instants stay in synctest; the simulator pins the interaction.** The D-11 fallback is explicit: the ladder's exact offsets are asserted in `backoff_test.go`, while `sim_dump_test.go` asserts confirmation-before-notification, failure-still-notifies, convergence and same-seed identity through a settle driver that never advances virtual time mid-step.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Compile-coupled RED] The simulator RED needed the Scheduler lock and a compiling `NotifyInstalled` stub**
- **Found during:** Task 3 RED
- **Issue:** The engine arms timers from its own goroutine, so the scheduler's `now`/`seq`/`pend` needed the lock and the one-call `ScheduleAfter` before a test could run without data races; and a test naming `Cluster.NotifyInstalled` cannot compile (and a panic is not a valid RED) without the method existing.
- **Fix:** The RED commit landed the Scheduler lock/ScheduleAfter (structural support, no tested behavior) and `NotifyInstalled` as a lookup-only stub with no announcement. Both target tests then failed on their planned assertions — the peer never received the confirmed or failed-window record, and the tie-break pair did not converge. GREEN restored the notification body.
- **Files modified:** `internal/app/internal/simnet/simnet.go`
- **Verification:** RED: `TestScenario_dumpNotify` failed with "node 1 holds ... ()"; GREEN: both scenarios pass under `-race`.
- **Commit:** `8d8735e` (RED), `10e1eac` (GREEN)

**2. [Rule 1 - Test bug] The ladder test's sleep target and the reset-rung advance**
- **Found during:** Task 2 GREEN
- **Issue:** The first draft slept the ladder's last offset (11.3 s) rather than the seventh-failure offset (6.3 s), so the "during backoff" write landed after the recovering attempt and its own debounce produced a ninth attempt; the reset assertion also read the attempt list before the 100 ms rung could fire.
- **Fix:** Sleep to the sixth ladder entry (6.3 s, the last failure) and advance 100 ms after the reset failure before reading the instants. The RED verdict was still the intended missing-ladder failure; the test corrections landed in the GREEN commit.
- **Files modified:** `internal/app/persist/backoff_test.go`
- **Verification:** `TestEngine_backoffLadder` passes with exactly 8 then 10 attempts at the expected offsets.
- **Commit:** `de50de1`

**3. [Rule 3 - Lint/test shape] Strict-gate shaping of the new engine code and tests**
- **Found during:** Task 2/3 lint
- **Issue:** `mnd` flagged the ladder's `d * 2`, `modernize` wanted `max(...)` for the negative-wait clamp, `staticcheck`/`ineffassign` flagged the always-replaced `debounce` parameter and its dead nil assignment, `revive` flagged an import-shadowing `port` local and `noinlineerr` an inline error check.
- **Fix:** Added `retryGrowth`, used `max(at.Sub(Now()), 0)`, dropped `debounce` from `rearm`'s signature and the dead assignment, renamed the local to `dumps`, and split the error check.
- **Files modified:** `internal/app/persist/engine.go`, `internal/app/repl/sim_dump_test.go`
- **Verification:** `make lint` clean (config verify + full run).
- **Commit:** `de50de1`, `10e1eac`

### Plan-artifact note

- `TestEngine_injectedClock` lives in `engine_test.go` per task 1's file list; the phase artifact list names `backoff_test.go` for it. `backoff_test.go` provides the ladder and class tests task 2 asks for.

## Issues Encountered

- None. `make lint`, `make test` (full race suite, 173 s on `internal/app/repl`) and the plan-level verifications are green; the EACCES stand scenario passes in 2.7 s.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The dumper's time, the failure ladder, the class surface and the simulator seam are complete; plan 05-08 can add the confirmation timeout on the same `Config.After` seam and `supkv_dumps_unconfirmed_total` beside `supkv_dumps_failures_total`.
- `Cluster.NotifyInstalled` is additive and available to later simulator scenarios (snapshot load notify, restore replay).
- Ready for 05-08.

## Self-Check: PASSED
