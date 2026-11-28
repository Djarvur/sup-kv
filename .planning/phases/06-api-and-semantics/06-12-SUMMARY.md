---
phase: 06-api-and-semantics
plan: 12
subsystem: api-watch-contract
tags:
  - apis-04
  - watch
  - startup-suppression
  - nil-tap
  - durable-change-feed
  - dump-confirmation
  - exactly-once
  - shutdown-unavailable
  - bounded-pending
  - deletions
  - version-order
  - tdd-red-evidence
  - status-complete
requires:
  - phase: 06-api-and-semantics (06-06)
    provides: the generation-tagged install tap, persist.Config.OnConfirmed, the pending queue with the confirmed watermark, and the bounded watchIngress shed policy
  - phase: 05-persistence
    provides: the dump cycle and confirmWaiters the durability watermark publishes after
provides:
  - "persist.Open suppresses the startup-load tap with a synchronous, nil-safe loading guard, so a loaded dump is never announced as a live change"
  - "WatchHub shutdown writes a fresh codes.Unavailable status with the message \"watch hub is shutting down\" before closing every subscriber channel; a late subscriber gets the same verdict"
  - "the startup-silence, nil-tap, dump-failed-then-confirmed, normal-write-exactly-once, shutdown-Unavailable, bounded-pending, deletion and per-key version-order test matrix"
affects:
  - 06-13-phase-gate
  - phase-9-api-docs
actuals:
  tokens: 7835
  tasks: 2
  commits: 4
  plan_head_before: e08c3792eb031d6764df663eeaa4ea41eec20e67
  plan_head_after: fa89e62eed4d06277c4848cc5bd42ec44649ef2a
tech-stack:
  added: []
  patterns:
    - "Startup merge is not a change feed: the caller's tap is captured once, a nil tap becomes a no-op, and a synchronous loading flag suppresses every install the merge produces."
    - "One shutdown-status helper for both the sweep and the late-subscribe refusal: a fresh status value per subscriber, so the code and message cannot drift and no single-use status is shared."
    - "Overflow policy pinned by observable behavior: a held-generation Ingest flood past watchIngress sheds with ResourceExhausted, and the resubscribe snapshot is the complete state."
key-files:
  created:
    - internal/app/persist/loader_internal_test.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-12-task1.json
    - .planning/phases/06-api-and-semantics/red-evidence/06-12-task2.json
  modified:
    - internal/app/persist/loader.go
    - internal/app/persist/engine_test.go
    - internal/grpcapi/watch.go
    - internal/grpcapi/watch_test.go
key-decisions:
  - "The startup guard is a plain bool, not an atomic: the merge runs on Open's goroutine and no tap consumer exists before Open returns, so after the merge the flag is only read — there is no shared write for the race detector to guard."
  - "A nil OnChange is substituted with a no-op inside Open, mirroring storage.Config.normalized, so every existing storage.Config{} caller stays a valid, panic-free configuration through the wrapper."
  - "The shutdown status lives in one helper (watchShuttingDown) used by both shutdownSubscribers and the late-subscribe refusal; every close writes a fresh value because status errors are single-use response payloads."
  - "The grpcapi tests that pass on the pre-change tree (normal-write exactly-once, bounded pending, deletions/order) are regression pins of the already-landed 06-06 gate; the RED targets are the two missing behaviors (startup suppression, shutdown status)."
  - "The nil-tap regression passes at RED by design: it protects behavior the wrapper must preserve, so its pre-change green is not an unexpected green."
requirements-completed: []
coverage:
  - id: D1
    description: "persist.Open over an encoded generation fires zero OnChange calls while loading, exposes every loaded record, and a nil tap stays writable."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestOpen_(loadDoesNotFireTap|nilTapStaysWritable)$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D2
    description: "A write whose dump failed is not announced; a later successful dump releases it exactly once, in install order, and a normal confirmed write is announced exactly once."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_(dumpFailedThenConfirmed|confirmedExactlyOnce)$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "A Watch stream open at hub shutdown ends with codes.Unavailable and the exact message \"watch hub is shutting down\", never OK."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_shutdownUnavailable$' -v -> PASS"
        status: pass
      - kind: process
        ref: "grep -q 'watch hub is shutting down' internal/grpcapi/watch.go -> exit 0; the same helper returns status.Error(codes.Unavailable, ...)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Pending is bounded by watchIngress: crossing it sheds every subscriber with ResourceExhausted, and a resubscription re-snapshots the complete current state."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWatchHub_pendingBounded$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D5
    description: "A Delete install is delivered as a deleted=true event, and two writes under one key arrive in strictly ascending version order with no duplicate."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_deletionsAndOrder$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D6
    description: "The changed packages stay green under the full suite and the strict lint gate."
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ ./internal/grpcapi/ -> PASS; go test -race -count=3 ./internal/grpcapi/ -run '^TestWatch' -> PASS"
        status: pass
      - kind: process
        ref: "make lint -> exit 0"
        status: pass
    human_judgment: false
duration: 6min 1s
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 12: Durable Watch Contract Expansion Summary

**The Watch feed now tells the whole truth at its edges: a startup dump is silent, a nil tap stays valid, a failed-then-confirmed write is announced exactly once when it becomes durable, and shutdown ends every stream with `Unavailable` instead of a clean-looking `OK`.**

## Performance

- **Duration:** 6min 1s (361 s)
- **Started:** 2026-11-23T00:46:33Z
- **Completed:** 2026-11-23T00:52:34Z
- **Tasks:** 2 (each TDD: RED + GREEN)
- **Files changed:** 7 (5 code/test, 2 RED-evidence records)
- **Commits:** 4 measured from ledger `e08c379..fa89e62`

## Accomplishments

- `persist.Open` no longer announces the startup load. It captures the caller's `storage.Config.OnChange`, substitutes a no-op when it is nil, and wraps it in a synchronous `loading` guard around `mergeRecords`: the wrapper returns early while loading and forwards `(key, rec, gen)` afterwards. The merge is single-goroutine and no tap consumer exists before `Open` returns, so a plain bool is race-free. A loaded dump can no longer fill the Watch hub's pending queue, and `storage.Config{}` callers stay panic-free.
- The dump-failure contract is now pinned through the API surface: a write whose dump fails answers `Unavailable` with the `persist.ErrDumpFailed` message and is not announced; the next successful dump — forced by a second write — releases both installs exactly once, in install order.
- The ordinary write path is pinned to exactly one event (`TestWatch_confirmedExactlyOnce`), and the deletion/order semantics are pinned end-to-end: a `Delete` arrives as a `deleted=true` event, and two writes under one key arrive in strictly ascending version order with no duplicate.
- Shutdown is honest: `shutdownSubscribers` writes a fresh `status.Error(codes.Unavailable, "watch hub is shutting down")` into every subscriber before closing its channel, and the late-subscribe refusal returns the same verdict through the same `watchShuttingDown` helper. A dying node never looks like a clean end of stream.
- The bounded pending policy is pinned by observable behavior: a held-generation `Ingest` flood past `watchIngress` sheds the subscriber with `ResourceExhausted`, and a resubscription re-snapshots the complete current state.

## Task Commits

1. **Task 1 RED: failing startup-load tap test and watch contract pins** — `4540fec` (test)
2. **Task 1 GREEN: suppress the startup-load tap with a nil-safe guard** — `ec37636` (feat)
3. **Task 2 RED: failing shutdown status test and overflow/order pins** — `24063ee` (test)
4. **Task 2 GREEN: end Watch streams with Unavailable on hub shutdown** — `fa89e62` (feat)

**Plan metadata:** see the final `docs(06-12)` commit.

## Files Created/Modified

- `internal/app/persist/loader.go` — `Open` captures the tap, substitutes a nil no-op and gates it with the synchronous `loading` flag around `mergeRecords`; the doc comment states the startup-is-not-a-change-feed contract.
- `internal/app/persist/engine_test.go` — `loadFixtureStore` (a `port.DumpStore` whose `Load` feeds the encoded fixture through the parse oracle) and `TestOpen_loadDoesNotFireTap`.
- `internal/app/persist/loader_internal_test.go` — the `nilTapDumpStore` fake and `TestOpen_nilTapStaysWritable` (package `persist`, the documented `_internal_test` exemption).
- `internal/grpcapi/watch.go` — `watchShuttingDown`; `shutdownSubscribers` writes the fresh status before each close; the late-subscribe path reuses the helper; the docs now say Unavailable, never nil.
- `internal/grpcapi/watch_test.go` — `TestWatch_confirmedExactlyOnce`, the API-driven rewrite of `TestWatch_dumpFailedThenConfirmed`, `TestWatch_shutdownUnavailable`, `TestWatchHub_pendingBounded`, `TestWatch_deletionsAndOrder`.
- `.planning/phases/06-api-and-semantics/red-evidence/06-12-task1.json`, `06-12-task2.json` — the `RED_EVIDENCE_OK` records for the two TDD targets.

## Decisions Made

- **Synchronous bool, not an atomic.** The startup merge is single-goroutine by construction and no tap consumer exists before `Open` returns; the flag is written only on that goroutine and afterwards only read, so an atomic would add cost and imply a concurrency the composition point does not have.
- **Nil tap becomes a no-op inside `Open`.** `storage.Config.normalized` already substitutes a no-op for direct `NewStore` callers; mirroring that in the wrapper keeps `storage.Config{}` a documented valid caller shape through `Open` and prevents a nil dereference if the tap ever fired before the store normalizes.
- **One shutdown-status helper.** `watchShuttingDown` is used by both the shutdown sweep and the late-subscribe refusal, so the code and message cannot drift; each subscriber gets a fresh status because status errors are single-use payloads.
- **Regression pins vs RED targets.** `TestWatch_confirmedExactlyOnce`, `TestWatchHub_pendingBounded` and `TestWatch_deletionsAndOrder` pass on the pre-change tree because plan 06-06's gate already implements them; they are pins, and the RED targets are the two genuinely missing behaviors. The nil-tap test is likewise a preserve-existing-behavior pin.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Encoded fixtures needed real record checksums**
- **Found during:** Task 1 RED
- **Issue:** the first RED run failed through the decode oracle instead of the tap assertion (`Open() discarded = 2`, zero loaded records): the hand-built `storage.Record` fixtures carried a zero checksum and `Decode` discarded them by class.
- **Fix:** compute `storage.RecordChecksum(key, version, data, false)` for each fixture record, then re-ran RED and captured the real tap assertion.
- **Files modified:** `internal/app/persist/engine_test.go`, `internal/app/persist/loader_internal_test.go` (both in the RED commit).
- **Verification:** RED re-run failed at `engine_test.go:303: Open fired OnChange 2 times, want 0`; GREEN passes.
- **Commit:** `4540fec`

### Plan-text clarifications

- **`server_test.go` and `watch_senderr_test.go` needed no edit.** The plan's `files_modified` listed them, but the harness already exposes `fakeDumpStore.failSave`/`holdDumps` and `cancelHub`, and the sender-error harness is unaffected by the shutdown status. The realized footprint is 5 code/test files (the two `files_modified` entries that needed changes plus `loader_internal_test.go`) plus the 2 RED-evidence records.
- **No post-load confirmed-watermark initialization was added.** The PATTERNS/Pitfall-8 note suggests seeding the hub watermark to the post-load store generation; the plan's Task 1 action and must_haves do not require it, and with the tap suppressed no loaded event is ever pending. The hub's watermark starting at zero is harmless: the first successful dump advances it past the loaded generation, and a clean `Main` load has nothing queued to release.
- **The pending-bound test asserts the observable contract, not the unexported slice.** `watch_test.go` is the external `grpcapi_test` package, so it cannot read `h.pending`. It asserts the two facts the acceptance names — `ResourceExhausted` on the shed stream and a complete resubscribe snapshot — and uses the snapshot's own key at the zero version for the synthetic flood, so any residue left in the pipeline is suppressed by the exact-cut rule rather than delivered.

## Issues Encountered

- The strict lint gate rejected the new tests with `noinlineerr` on four inline error checks; converted to plain assignments before the RED commit. `make lint` is green.
- The initial fixture checksum omission (see the Rule 3 entry) was fixed before the RED evidence was recorded, so the committed RED record is the real tap failure.
- `roadmap update-plan-progress 06` again matched the archived v1.0 Hardening row (the 06-05/06-06 precedent) and rewrote it; repaired by hand — v1.0 back to `9/9 Complete`, v1.1 forward to `8/13 In Progress`.
- `state update-progress` has declined on this unscoped-phase shape before; the position, plan counter, metric row, decisions and session fields are the authoritative updates this plan records.

## User Setup Required

None - no external service configuration required.

## TDD Gate Compliance

- RED before GREEN for both tasks: `test(06-12)` commits `4540fec` and `24063ee` precede their `feat(06-12)` commits `ec37636` and `fa89e62`.
- RED evidence is machine-validated: `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for `06-12-task1.json` (target `TestOpen_loadDoesNotFireTap`) and `06-12-task2.json` (target `TestWatch_shutdownUnavailable`); both targets executed and failed on their planned assertions, not on build/fixture faults.
- No REFACTOR commit: the GREEN changes were minimal and no cleanup was needed.

## Known Stubs

None in this slice. The startup-load suppression was the last Pitfall-8 deferral from 06-06.

## Next Phase Readiness

- APIS-04's REQUIREMENTS checkbox stays pending: `requirements ready-ids` reports `0/1` ready (the shared-ID gate among 06-06/06-09/06-13), so this plan marks no requirement complete.
- Plan 06-13's phase gate can run `make lint && make test`, the red runner and the frozen-file base diff on top of the complete Watch contract.
- Ready for 06-07; five plans remain in the phase (06-07, 06-08, 06-09, 06-10, 06-13).

## Self-Check: PASSED
