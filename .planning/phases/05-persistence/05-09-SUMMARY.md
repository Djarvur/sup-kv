---
phase: 05-persistence
plan: 09
subsystem: persistence
tags: [engine-split, roles, confirmation, scheduler, facade, loader, snapshot, refactor, persistance-11, no-behavior-change]
requires:
  - phase: 05-persistence
    provides: the bounded confirmation, single-claim resolution and FlushTimeout seams from 05-08
  - phase: 05-persistence
    provides: the streaming encoder, load outcomes, preservation/sweep and the dump failure classes from 05-01..05-07
  - phase: 03-transport
    provides: the write-confirmation engine, the waiter slice and the capture-before-Records proof from Phase 3
  - phase: 04-protocol
    provides: the write-notify totality and the MergeChanged dirty decision
affects: [persistence, engine-structure, p10, p11, p12, phase-6, phase-7]
actuals:
  tokens: 20196
  tasks: 3
  commits: 2
plan_head_before: b2e7350e070220662de5c30d10b3501e16da6a3b
plan_head_after: 06704d63ab7bc30adf76434d22e295f39ab63877
tech-stack:
  added: []
  patterns:
    - "Composition-root embedding: Engine embeds the unexported *confirmer so the pre-split white-box tests (eng.mu, eng.waiters, eng.markDirty) compile unchanged while the role owns the state and its mutex"
    - "One published atomic holder: New constructs a single *dumpStats and hands the same pointer to the scheduler and the confirmer, so the timeout counter and the dump counters are one value"
    - "Role files as method groups over the one composition root, with cross-role calls going through one-lock methods and no nested role locks"
key-files:
  created:
    - internal/app/persist/confirm.go
    - internal/app/persist/scheduler.go
    - internal/app/persist/facade.go
    - internal/app/persist/loader.go
    - internal/app/persist/snapshot.go
  modified:
    - internal/app/persist/engine.go
    - internal/app/persist/doc.go
    - .planning/phases/05-persistence/deferred-items.md
key-decisions:
  - "Engine embeds *confirmer rather than holding a named conf field: the frozen white-box tests read Engine.mu and Engine.waiters and call Engine.markDirty, and 'no test edit' is a hard acceptance criterion; field promotion keeps those call sites byte-identical while the state still lives behind the role's own mutex"
  - "The retry ladder got the scheduler's own mutex: rearm reads the dirty epoch from the confirmer and the retry state from the scheduler under two sequential locks, never nested, preserving T-5-30's one-mutex-per-role rule"
  - "The scheduler owns the one shared *dumpStats holder: New constructs it once and hands the same pointer to newScheduler and newConfirmer, so await's timeout increment and the dump counters are one published value"
  - "Task 3 made no commit: the whole-tree gate ran on the tree tasks 1-2 committed, following the 01-04 precedent that committing a no-op fabricates history"
  - "The full make test run's only failure is the pre-existing TestCore_keyCountCostFlat wall-clock ratio flake in the untouched repl package; the standalone re-run passed and it is recorded in deferred-items.md for Phase 8"
patterns-established:
  - "Role files own declarations; engine.go owns the composition and the exported API"
  - "A state-owning role takes its own mutex once per entry point and never holds it across another role's lock or I/O"
  - "A frozen white-box test is preserved by embedding the role value, never by editing the test"
requirements-completed: [PERSIST-11]
coverage:
  - id: D1
    description: "The confirmation role lives in confirm.go: waiter and confirmer own the waiters, dirty/oldest, changed nudge, closed flag, runDone and flushCh behind one mutex; await, confirmWaiters, failWaiters, sweepClosed and exit carry the single-claim protocol verbatim, and the timeout increments unconfirmed on the shared holder"
    requirement: "PERSIST-11"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_(confirmationTimeout|timeoutThenDumpSuccess|waiterRemoved|markDirtyNudge|closeSurface|runExitClosesWaiters|flushRacesExitMidDump|flushRacesRunExit)$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "The scheduler role lives in scheduler.go: Run/runDump/flushDump/rearm/handleDumpFailure/nextRetry/resetRetry/recordDump and the one dumpStats holder; the retry ladder, debounce/deadline/retry channels and the 100 ms-to-5 s backoff are unchanged"
    requirement: "PERSIST-11"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_(backoffLadder|dumpFailureClasses|dumpFailureFansOut|confirmDebounceTiming|burstBoundedConfirmation|redundantDirty|injectedClock)$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestDumpStats_records$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The facade, loader and snapshot roles are separate files and engine.go is the composition root plus the public write/Flush API; Snapshot/Restore take exactly one copy and reuse the store write path"
    requirement: "PERSIST-11"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ ./internal/grpcapi/ ./internal/udprepl/ ./internal/persistfile/ ./cmd/sup-kv/"
        status: pass
    human_judgment: false
  - id: D4
    description: "The exported surface is unchanged and no interface was added: the five-role split is internal-only, and no consumer package was edited"
    requirement: "PERSIST-11"
    verification:
      - kind: other
        ref: "go doc -all ./internal/app/persist exported-declaration diff before/after: empty; grep 'type .* interface' internal/app/persist/*.go: none; git diff --name-only b2e7350..HEAD | grep -v '^internal/app/persist/': empty"
        status: pass
    human_judgment: false
  - id: D5
    description: "The lint gate and the darwin cross build/vet are green on the moved code"
    requirement: "PERSIST-11"
    verification:
      - kind: process
        ref: "make lint"
        status: pass
      - kind: process
        ref: "GOOS=darwin go build ./... && GOOS=darwin go vet ./..."
        status: pass
    human_judgment: false
  - id: D6
    description: "The whole pre-split suite is green on the moved code with no test file changed, proving the move behavior-preserving"
    requirement: "PERSIST-11"
    verification:
      - kind: unit
        ref: "go test -race -cover ./... (all packages green except TestCore_keyCountCostFlat, the documented wall-clock ratio flake in the untouched repl package; standalone re-run PASS at ratios 0.97/1.01)"
        status: pass
    human_judgment: true
    rationale: "The full make test run was green except one pre-existing, load-sensitive ratio flake in internal/app/repl, a package this plan's diff does not touch; the standalone re-run passed and the flake is deferred to Phase 8, so a verifier should classify the exception rather than the executor self-certifying it."
duration: 12min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 09: Five Internal Roles Summary

**`persist.Engine` is now a composition root over a `confirmer`, a `scheduler` and three method-group files — `loader.go`, `facade.go`, `snapshot.go` — with every exported signature, the capture-before-Records proof, the single-claim resolution and the retry ladder moved verbatim, and no test file touched.**

## Performance

- **Duration:** 12min
- **Started:** 2026-11-22T16:16:14Z
- **Completed:** 2026-11-22T16:28:26Z
- **Tasks:** 3
- **Files modified:** 7 code/docs files (5 created, 2 modified) plus the phase deferred-items log

## Accomplishments

- `confirm.go` is the confirmation-accounting role. `waiter` and the new unexported `confirmer` own the waiter slice, the dirty flag and oldest epoch, the non-blocking `changed` nudge, the closed flag, `runDone` and the flush request channel, behind one mutex. `await` (register + timeout claim), `confirmWaiters`, `failWaiters`, `sweepClosed` and the combined `exit` carry the single-claim protocol verbatim; the timeout branch removes its waiter with `slices.DeleteFunc` and increments `stats.unconfirmed` with an atomic Add on the one shared holder.
- `scheduler.go` is the dump-scheduling role. It owns `Run`'s loop (via `run`), `runDump`, `flushDump`, `rearm`, `handleDumpFailure`, `nextRetry`, `resetRetry`, `recordDump`, the timer channels and the unexported `dumpStats` holder behind `Engine.DumpStats`. The 100 ms-doubling-to-5 s ladder, the retry-instant pinning and the pending-retry channel suppression are unchanged. Its own mutex guards only the ladder state; `rearm` reads the dirty epoch from the confirmer and the ladder from the scheduler under two sequential locks, never nested.
- `facade.go` holds the thin store forwards (`Get`, `Lookup`, `Version`, `Versions`, `Observe`, `KeyCount`, `Generation`, `Records`) and `ApplyRemote`'s `MergeChanged`-dirty shape. `loader.go` owns `Open`, `mergeRecords` and `startLoadDirty`. `snapshot.go` owns `Snapshot`, `Restore`, `RestoredRecord`, `replaySnapshot`, `restoreChanges` and `replayRecord`.
- `engine.go` is now ~300 lines: `Config`/`normalized`, the `Engine` composition (embedded `*confirmer` plus store, port, config and `*scheduler`), the exported `DumpStats`/`LoadStats` types, `New`, `LoadStats`, the three write methods and `Flush`/`FlushTimeout`/`flush`.
- `doc.go` names the five roles and keeps every existing lifecycle, timing and snapshot contract sentence.
- The split is a pure move: the exported declaration set is byte-identical (`go doc -all` diff empty), the diff adds no `type ... interface`, no consumer package changed, and the pre-split suite compiles and passes with no test edit — including the frozen P-01 engine test and the two white-box tests.

## Task Commits

1. **Task 1: Extract the confirmation role** — `0370b08` (refactor)
2. **Task 2: Split the scheduler, facade, loader and snapshot roles** — `06704d6` (refactor)

**Plan metadata:** the `docs(05-09)` completion commit (this commit).

Task 3 is verification-only: it changed no file, so it produced no commit (01-04 precedent — a no-op commit would fabricate history).

## Files Created/Modified

- `internal/app/persist/confirm.go` (new) — `waiter`, `confirmer`, `await`, `confirmWaiters`, `failWaiters`, `sweepClosed`, `exit`, `markDirty`, `startDirty`, `epoch`, `isDirty`, `isClosed`, `capture`, `nudge`.
- `internal/app/persist/scheduler.go` (new) — `dumpStats`, `scheduler`, `newScheduler`, `Engine.Run`, `run`, `flushDump`, `rearm`, `runDump`, `handleDumpFailure`, `nextRetry`, `resetRetry`, `recordDump`, `Engine.DumpStats`, the retry constants.
- `internal/app/persist/facade.go` (new) — the store forwards plus `ApplyRemote`.
- `internal/app/persist/loader.go` (new) — `Open`, `startLoadDirty`, `mergeRecords`.
- `internal/app/persist/snapshot.go` (new) — `Snapshot`, `RestoredRecord`, `Restore`, `replaySnapshot`, `restoreChanges`, `replayRecord`.
- `internal/app/persist/engine.go` (modified, −829/+~45 from the original) — composition root, `New` wiring the one `dumpStats`, public write and Flush API.
- `internal/app/persist/doc.go` (modified) — the five-role paragraph; existing contracts preserved.
- `.planning/phases/05-persistence/deferred-items.md` (modified) — the two flakes observed this plan, with evidence and owners.

## Decisions Made

- **Embed the confirmer, do not name it `conf`.** The plan's action said `Engine` holds `conf *confirmer`; making that a named field would have forced an edit to `timeout_internal_test.go` (`eng.mu`, `eng.waiters`) and `load_stats_internal_test.go` (`eng.markDirty`), contradicting the plan's "no test edit" criterion and its frozen-suite prohibition. An embedded `*confirmer` is still a `*confirmer` field, promotes those three names to the same state, and adds no exported name. Recorded as the one Rule 3 deviation.
- **One published stats holder.** `New` constructs `stats := &dumpStats{}` once and hands the same pointer to `newConfirmer` and `newScheduler`, exactly as the plan's must-haves require; `Engine.DumpStats` is a value copy over that one holder.
- **Two role mutexes, never nested.** The old engine mutex became the confirmer's; the retry ladder moved to the scheduler's own mutex. `rearm` takes them sequentially. This keeps T-5-30's "each state-owning role keeps one mutex" and avoids a lock-order question entirely.
- **Task 3 documents rather than repairs the flake.** The only `make test` failure was the known `TestCore_keyCountCostFlat` ratio flake in the untouched `internal/app/repl`; it is out of the deviation scope boundary, passed standalone, and is recorded in `deferred-items.md` with Phase 8 as owner.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `Engine` embeds `*confirmer` instead of holding a named `conf` field**
- **Found during:** Task 1
- **Issue:** Two committed white-box tests reach into the engine's confirmation state directly — `timeout_internal_test.go` locks `eng.mu` and reads `eng.waiters`, and `load_stats_internal_test.go` calls `eng.markDirty()`. A named `conf *confirmer` field makes both fail to compile, and the plan's hard acceptance is "the suite green with no test edit".
- **Fix:** The confirmer is an embedded `*confirmer` field. Go promotes its fields and methods to `Engine`, so the three call sites resolve to the same state and the same mutex with zero test changes, while `confirm.go` still owns the state and no exported name is added. The composition-root comment states this reason.
- **Files modified:** `internal/app/persist/engine.go`, `internal/app/persist/confirm.go`
- **Verification:** `go test -race -count=1 ./internal/app/persist/` green; `git diff --stat b2e7350..HEAD -- 'internal/app/persist/*_test.go'` empty.
- **Commit:** `0370b08`

### Plan-artifact notes

- The plan's research table listed `Set`/`Delete`/`SetIfAbsent` under the facade. The PLAN's must-haves instead put "the public write/read/`Flush` methods" in `engine.go` and name only the read forwards plus `ApplyRemote` for `facade.go`; the executor followed the PLAN. The write methods still call the confirmer's `await`, so the role boundary holds.
- `DumpStats` and `LoadStats` types stay in `engine.go` (the must-have names `LoadStats` there); the unexported `dumpStats` holder lives in `scheduler.go` with its writer.
- The engine's exempted `retryMu` existed only inside task 1's tree; task 2 moved it into the scheduler as the role's `mu`, so the final tree has exactly two role mutexes.

## TDD Gate Compliance

The plan frontmatter is `type: execute`; tasks 1 and 2 carry `tdd="true"`, but both are behavior-preserving moves of already-tested code.

- **No `test(05-09)` commit exists, and none can legitimately.** The plan itself requires "the persist package suite is green with no test edit" and prohibits any behavior change. A target test naming `confirmer`/`scheduler` would not compile at the plan base (the `build_failure` / `INVALID_RED` class in `references/tdd.md`), and an assertion on existing behavior starts green (`unexpected_green`). There is no failing target assertion to write.
- **No `feat(05-09)` commit exists.** Both production commits are `refactor(05-09)`, the correct type for a diff with no behavior change.
- **RED-equivalent evidence:** task 1's planned structural probe (`grep -q 'type confirmer struct' internal/app/persist/confirm.go`, annotated `[HEAD=error:2 EXPECT>=1]`) exits 2 at the plan base and 0 after task 1; the pre-split suite is byte-identical and green before and after (no test file in the plan diff).
- **MVP+TDD behavior-adding gate:** task 1's `<behavior>` block names internal role ownership, not a user-visible outcome, and the task is a pure move, so it is not behavior-adding; the gate is not applicable and no halt was raised.

## Issues Encountered

- **Known flake, not move fallout:** the package-listed task-2 verify failed once on `TestLog_formatText` (`bind udp4 0.0.0.0:50584: address already in use`); the package re-run alone passed (6.2 s) and `make test` passed the package. Same pick-to-bind class as the 05-04/05-06 entries.
- **Known flake, not move fallout:** the full `make test` run failed only on `TestCore_keyCountCostFlat` (`internal/app/repl`, ratio 2.91 > 2.0 under full-suite load). The standalone re-run measured 0.97 and 1.01 and passed. The package is untouched by this plan's diff and the flake is already deferred to Phase 8 in STATE.md/05-CONTEXT.
- Both are recorded with evidence in `.planning/phases/05-persistence/deferred-items.md`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The five internal roles exist and `engine.go` is the composition root; the exported `persist` surface is unchanged, so 05-10 and Phase 6/7 can change one role without reopening the API.
- The confirmation role's `await`/claim protocol and the scheduler's ladder are now separable review units; the shared `*dumpStats` holder is the one place both write.
- Three plans remain in the phase (05-10, 05-11, 05-12). Ready for 05-10.

## Self-Check: PASSED

- Created files verified on disk: `internal/app/persist/confirm.go`, `scheduler.go`, `facade.go`, `loader.go`, `snapshot.go`, `.planning/phases/05-persistence/05-09-SUMMARY.md`.
- Task commits verified as ancestors of HEAD: `0370b08`, `06704d6`.
- Exported declaration set identical to the plan base (`go doc -all` diff empty); no test file changed by the plan diff.
