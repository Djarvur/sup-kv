---
phase: 06-api-and-semantics
plan: 06
subsystem: api-watch-durability-gate
tags:
  - apis-04
  - watch
  - durability-watermark
  - generation-tap
  - onconfirmed
  - pending-queue
  - confirm-gate
  - exactly-once
  - dump-confirmation
  - tracer
  - tdd-red-evidence
  - status-complete
requires:
  - phase: 06-api-and-semantics (06-02)
    provides: the write-time capture and honest wait seam the same server harness exercises
  - phase: 06-api-and-semantics (06-05)
    provides: the harness shape (hub, engine, server) and the live-view/precedent for a test landing with its symbol
  - phase: 05-persistence
    provides: the dump cycle, confirmWaiters and the OnChange tap the watermark now extends
  - phase: 04-protocol
    provides: the store content generation (bumped once per install) and the exact-cut watch pipeline
provides:
  - "storage.Config.OnChange(key, rec, generation uint64): the tap hands the just-incremented content generation with every actual install"
  - "persist.Config.OnConfirmed(generation uint64): the generation captured immediately before Records() and published once per successful dump, after confirmWaiters; a failed dump publishes nothing"
  - "WatchHub pending queue + atomic confirmed watermark + Confirm(gen): events release in arrival order exactly when gen <= confirmed, once, and a repeated or lower confirmation delivers nothing"
  - "WatchHub.Ingest(key, rec, gen); the hub holds installs until the persist engine confirms a covering dump"
  - "production wiring OnChange -> hub.Ingest and OnConfirmed -> hub.Confirm in cmd/sup-kv/main.go, plus every Watch harness"
  - "the held-dump, dump-failed-then-confirmed and deterministic hub-gate tests"
affects:
  - 06-12-durability-gate-expansion
  - 06-13-phase-gate
  - phase-9-api-docs
actuals:
  tokens: 11368
  tasks: 1
  commits: 2
  plan_head_before: 2410b124e0a93265028601e1e113a71f4c590423
  plan_head_after: 4f4acba51c2275d6c77d714ff4dcea9aa3e8ca0e
tech-stack:
  added: []
  patterns:
    - "Install announcement vs durability claim: the stable tap carries the content generation, but only the persistence engine's confirmed watermark lets the hub deliver."
    - "Capture-before-snapshot watermark: runDump reads Store.Generation() immediately before Records(), so the published generation describes exactly the saved snapshot; an install between the reads publishes one dump later."
    - "Prefix drain under an atomic max: Confirmed is a CAS-maximum watermark with a buffered(1) wake; release dispatches only the confirmed prefix of the pending queue and drops it, which is exactly-once by construction."
    - "Bounded queue, shed and heal: both the ingress and the pending queue are capped at watchIngress; either overflow sheds every subscriber with ResourceExhausted and clears the queue, and resubscription re-snapshots."
key-files:
  created:
    - .planning/phases/06-api-and-semantics/red-evidence/06-06-task1.json
  modified:
    - internal/app/storage/store.go
    - internal/app/storage/store_test.go
    - internal/app/persist/engine.go
    - internal/app/persist/scheduler.go
    - internal/grpcapi/watch.go
    - internal/grpcapi/watch_test.go
    - internal/grpcapi/watch_senderr_test.go
    - internal/grpcapi/server_test.go
    - internal/grpcapi/bench_test.go
    - internal/grpcapi/wait_integration_test.go
    - cmd/sup-kv/main.go
key-decisions:
  - "The RED target is the two compile-clean integration tests (held dump; dump-failed-then-confirmed): they fail on the planned assertions against the old forwarding hub. The storage-tap-generation and hub-gate unit tests name symbols this plan introduces, so they cannot be RED — they landed with the GREEN implementation, the 06-05 store-view precedent."
  - "The watermark is read immediately before Records() and published only after confirmWaiters: a write racing the capture is in the saved snapshot but is confirmed by the next dump, so the watermark can never cover a generation the dump did not save."
  - "Confirm is an atomic maximum, not an assignment: a later lower confirmation cannot move the watermark backwards; the buffered(1) confirm channel coalesces wake-ups without ever blocking the dumper goroutine."
  - "The pending queue releases only a prefix and drops what it dispatches, so each generation is delivered at most once, in install order; a repeated Confirm(gen) finds nothing and delivers nothing."
  - "Both overflow flavors clear the pending queue as well as shedding subscribers: pending entries are install announcements, not durable state, and a resubscribing client's snapshot reconstructs exactly the same state (D-02)."
  - "The subscribe snapshot stays the current in-memory state, never a confirmed-only view: the asymmetry with the durable live feed is the documented D-04 contract, and filtering the snapshot would need per-subscribe pending state with an inconsistent overflow story."
  - "Startup-load suppression and the post-load watermark initialization (Pitfall 8) stay in plan 06-12, as the plan's objective assigns them; this plan's gate, bounded queue and wiring are complete without them."
patterns-established:
  - "A tap can be both the exact change observer and the durability-neutral signal: the generation travels with the record, and a single watermark decides visibility."
  - "Durability tests use the real engine timer/dump path with holdDumps and an injected save failure, so the gate is proven end-to-end rather than against a fake hub."
  - "A mid-test negative proof uses one pending Recv and a timed-out select, keeping the receiver for the later positive delivery instead of orphaning it (recvNone remains the final-assertion helper)."
requirements-completed: []
coverage:
  - id: D1
    description: "storage.Config.OnChange receives the just-incremented install generation, exactly once per actual install, and never for a rejected SetIfAbsent, a Delete of a missing key, an identical merge redelivery or an Observe."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestStore_changeTapGeneration$' -v -> PASS"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestStore_onChange$' -v -> PASS (existing tap contract preserved)"
        status: pass
    human_judgment: false
  - id: D2
    description: "persist.Config.OnConfirmed fires the generation captured immediately before Records(), once per successful dump and only after confirmWaiters; a failed dump publishes nothing, and the later successful dump covers the installed record."
    requirement: "APIS-04"
    verification:
      - kind: process
        ref: "gen := s.store.Generation() precedes records := s.store.Records() in internal/app/persist/scheduler.go (lines 232-233) -> exit 0"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_dumpFailedThenConfirmed$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "The hub gate is deterministic: Ingest(key, rec, 5) does not reach a subscriber while confirmed is below 5, Confirm(5) delivers it exactly once, and a second Confirm(5) delivers nothing."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWatchHub_confirmGate$' -v -> PASS (no event before Confirm(5), exactly one after, silence after the repeat)"
        status: pass
    human_judgment: false
  - id: D4
    description: "With holdDumps, a Set installs the record in memory but no Watch event arrives; the Set does not return; after releaseDumps the Set returns and exactly one event arrives."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_unconfirmedNotAnnounced$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D5
    description: "Production and test harness wiring is complete: storage.Config.OnChange -> hub.Ingest, persist.Config.OnConfirmed -> hub.Confirm in cmd/sup-kv/main.go, and the harnesses that expect Watch release."
    requirement: "APIS-04"
    verification:
      - kind: process
        ref: "go build ./... && go vet ./internal/grpcapi/ -> exit 0 (vet compiles the migrated Ingest call sites)"
        status: pass
      - kind: process
        ref: "grep -q 'OnConfirmed: hub.Confirm' cmd/sup-kv/main.go -> exit 0"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ ./internal/app/storage/ ./internal/app/persist/ -> PASS"
        status: pass
    human_judgment: false
duration: 9min 40s
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 06: Durable Watch Gate Summary

**Watch is now a durable change feed: every install carries its content generation, each successful dump publishes the generation its snapshot covered, and the hub holds events until that watermark releases them once, in install order.**

## Performance

- **Duration:** 9min 40s (580 s)
- **Started:** 2026-11-23T00:22:37Z
- **Completed:** 2026-11-23T00:32:17Z
- **Tasks:** 1 (TDD tracer: RED + GREEN)
- **Files changed:** 12 (11 code/test, 1 RED-evidence record)
- **Commits:** 2 measured from ledger `2410b12..4f4acba`

## Accomplishments

- `storage.Config.OnChange` now hands the just-incremented content generation with every actual install; `install` remains the single convergence point, so the tap's generation is monotone and fires exactly once per real change and never for a rejection or no-op.
- `persist.Config.OnConfirmed(generation uint64)` is the engine's durability publication. `runDump` reads `gen := s.store.Generation()` immediately before `records := s.store.Records()`, and on success calls `OnConfirmed(gen)` after `confirmWaiters`; a failed dump calls it never.
- `WatchHub` gained the pending gate: `watchEvent` carries `gen`, `Ingest(key, rec, gen)` enqueues, `pending []watchEvent` holds unconfirmed installs in arrival order, `confirmed atomic.Uint64` accepts only CAS-maximum advances, and `Confirm(gen)` wakes the hub through a buffered confirm channel. `Run` releases the leading `gen <= confirmed` prefix before and after each select iteration and drops what it dispatches.
- Both queues are bounded at `watchIngress`. A pending append that would exceed the bound, or a full ingress, sheds every subscriber with `ResourceExhausted` and clears the pending queue; the resubscription snapshot heals the loss.
- Production wiring is complete: `OnChange: hub.Ingest` and `OnConfirmed: hub.Confirm` in `cmd/sup-kv/main.go`, and the same pair in every harness that observes Watch delivery (server, sender-error, wait-integration, bench). The pre-existing synthetic `Ingest` flood sites pass generation 0, which is at or below the starting watermark, so they keep their immediate-release behavior.
- The test harness exposes `cancelHub` so the expansion plan's shutdown tests can end the hub deliberately.

## Task Commits

1. **Task 1 RED: failing durability-gate watch tests** — `ab4bc2a` (test)
2. **Task 1 GREEN: generation-tagged tap, confirmed watermark, pending/Confirm hub, wiring** — `4f4acba` (feat)

**Plan metadata:** see the final `docs(06-06)` commit.

## Files Created/Modified

- `internal/app/storage/store.go` — `OnChange` signature with the install generation; `install` passes `s.generation`.
- `internal/app/storage/store_test.go` — the tap recorder records generations; `TestStore_changeTapGeneration` pins the one-step-per-install and the no-event rejections/no-ops.
- `internal/app/persist/engine.go` — `Config.OnConfirmed` with the `OnDumpFailure` doc voice (once per successful dump, after waiters, must not block, nil valid).
- `internal/app/persist/scheduler.go` — generation capture immediately before `Records()`, success-only `OnConfirmed(gen)` after `confirmWaiters`.
- `internal/grpcapi/watch.go` — `watchEvent.gen`, pending queue, atomic-max `Confirm`, release/hold/shed helpers, the confirm wake channel, and the shed-clears-pending policy.
- `internal/grpcapi/watch_test.go` — `TestWatch_unconfirmedNotAnnounced`, `TestWatch_dumpFailedThenConfirmed`, `TestWatchHub_confirmGate`; the flood passes generation 0; `recvNone` keeps one negative-proof shape.
- `internal/grpcapi/watch_senderr_test.go`, `server_test.go`, `wait_integration_test.go`, `bench_test.go` — `OnConfirmed: hub.Confirm`, the `cancelHub` handle, and the migrated `Ingest` call sites.
- `cmd/sup-kv/main.go` — `OnConfirmed: hub.Confirm` and the updated `startEngine` doc.
- `.planning/phases/06-api-and-semantics/red-evidence/06-06-task1.json` — the compile-clean RED record.

## Decisions Made

- **RED target substitution.** The two integration tests (`TestWatch_unconfirmedNotAnnounced`, `TestWatch_dumpFailedThenConfirmed`) compile against the old tree and failed on their planned assertions, so they are the RED evidence. The storage-tap-generation and hub-gate tests name `generation uint64` and `WatchHub.Confirm`, which did not exist, so a RED run including them would have been a refused build failure; they landed with GREEN, as 06-05 did for its store-view test.
- **Capture-before-snapshot, publish-after-waiters.** The generation is read immediately before `Records()` and published only after `confirmWaiters`; an install racing the capture is in the saved snapshot but confirmed one dump later, so the watermark never over-claims.
- **Atomic maximum + prefix drain.** `Confirm` is a CAS maximum with a buffered(1) wake; `release` dispatches and drops only the confirmed prefix, making delivery exactly-once, in install order, and a repeated or lower confirmation a no-op.
- **Shed clears pending.** Pending entries are install announcements, not durable state; a shed client resubscribes and re-snapshots, so clearing can lose no information any subscriber needs (D-02).
- **Snapshot asymmetry preserved.** The subscribe snapshot remains the current in-memory state (what `Get` sees); the live feed is the durable change feed. This is the D-04 contract, not a defect.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The API-coupled tests could not precede their symbols**
- **Found during:** Task 1 RED
- **Issue:** `TestStore_changeTapGeneration` needs the three-argument `OnChange`, and `TestWatchHub_confirmGate` needs `WatchHub.Ingest(key, rec, gen)` and `Confirm`; none existed, so a RED run containing them would have been a `[build failed]` package and refused as evidence.
- **Fix:** The RED commit carries the two compile-clean integration tests and their machine-validated evidence (`RED_EVIDENCE_OK`, target `TestWatch_unconfirmedNotAnnounced`); the API-coupled tests shipped in the GREEN commit and pin the implementation.
- **Files modified:** `internal/app/storage/store_test.go`, `internal/grpcapi/watch_test.go`
- **Verification:** RED run exit 1 with both integration targets failing on their planned assertions; GREEN run green for all three tests.
- **Commit:** `4f4acba`

### Plan-text clarifications

- **`recvNone` lost its `window` parameter.** Every call site passed `watchQuietWindow`, and the three new call sites made `unparam` flag it; the helper now closes over the constant and its doc says so. No behavior change.
- **`wait_integration_test.go` and `bench_test.go` gained `OnConfirmed`.** They run the hub but watch nothing; wiring the same seam keeps their pending queue draining and matches the production shape the plan's `files_modified` anticipated.

## Issues Encountered

- The strict lint gate first rejected the new tests: `noinlineerr` on seven inline error checks and `unparam` on `recvNone`'s always-identical window. Fixed in the GREEN commit; `make lint` is green.
- The first full `go test -race ./...` hit a transient `TestLog_jsonDefault` UDP bind error (`address already in use` on port 50870), and a second full run hit `TestWaitIntegration_staleRecordNotAcked` under cross-package load. Both are the project's known load/port-sensitive flake classes (STATE records the Phase-8 flake owner); the cmd package and the grpcapi package each pass standalone, and the grpcapi suite passed three consecutive `-count=3` repetitions. No test or bound was changed for either.
- `roadmap update-plan-progress 06` again matched the archived v1.0 Hardening row (the 06-05 precedent) and rewrote it to `7/13 In Progress`; repaired by hand — v1.0 back to `9/9 Complete`, v1.1 forward to `7/13 In Progress`. The phase-section count and the 06-06 checkbox were correct.
- `state update-progress` declined with "phase scope is unscoped, not complete" (the 06-05 precedent); the phase-level progress bar stays at 33% until the phase closes. Position, plan counter, metric row, decisions and session fields all advanced.
- No authentication gates and no user setup were needed.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None in this slice. The startup-load suppression and post-load watermark initialization (Pitfall 8) are the explicitly deferred 06-12 slice, not a stub: with a loaded dump the startup installs queue as pending until the first successful dump, and the bounded queue sheds rather than growing without bound.

## Next Phase Readiness

- APIS-04's REQUIREMENTS checkbox stays pending: `requirements ready-ids` reports it blocked by the shared-ID gate among 06-06/06-09/06-13, so this plan marks no requirement complete. The shutdown-ends-Unavailable clause is plan 06-12's.
- Plan 06-12 can now build on `pending`, `confirmed`, `Confirm`, the `watchIngress` bound and the harness's `cancelHub`: it owns startup suppression, the overflow-policy pin, deletion/order and the Unavailable shutdown test.
- Plan 06-13's phase gate can exercise the production wiring: `main` threads both `OnChange` and `OnConfirmed`.
- Ready for 06-07; the durability-gate expansion slice is 06-12.

## Self-Check: PASSED
