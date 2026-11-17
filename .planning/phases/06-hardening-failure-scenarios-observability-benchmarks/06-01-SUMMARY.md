---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 01
subsystem: replication-persistence-hardening
tags: [crdt, replication, udp, persist-engine, review-fixes, regression-tests]

requires:
  - phase: 04-replication-and-deterministic-simulator
    provides: repl core + udprepl adapter mirror seams (normalized config, retainDrained, NotifyLocalWrite)
  - phase: 03-persistence-dump-and-write-confirmation
    provides: persist Engine (ApplyRemote/Restore/confirmWaiters/Open) over storage.Merge
provides:
  - Six D-15 review fixes with mutation-checked regression tests (rate clamp, Acked dedup, closed-loop NotifyLocalWrite, conditional dirty, confirmWaiters invariant assertion, dead Open fallback removed)
  - Clean 03/04 review-disposition ledgers (all D-15 rows fixed; 03 open reduced to IN-03/IN-04, 04 open to zero)
  - The exact egress-pacing / drained-mirror / dump-confirmation contracts every later TEST-04 failure scenario asserts through
affects: [06-02 failure scenarios, 06-04 sim scenarios, TEST-04 suite, observability plans]

actuals:
  tokens: 11000   # chars/4 over the realized diff (44191 diff chars across 9 files)
  tasks: 3
  commits: 3      # MEASURED: git rev-list --count 45208b8..HEAD

tech-stack:
  added: []
  patterns:
    - "conditional dirty coded to the store's documented Conflict contract (Winner=incoming ⇔ displacement) plus a live pre-probe for the conflicts-nil ambiguity"
    - "whole-snapshot restore no-op predicate — per-record skipping is forbidden by the pinned fresh-version dominance contract"
    - "single select + post-send liveness re-check for enqueue-vs-shutdown races (the flush() reply re-check's sibling shape)"
    - "decode-through-the-oracle: when an oracle function IS another function's error channel, no error branch remains at the call site"

key-files:
  created:
    - internal/app/persist/engine_internal_test.go
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/sync_test.go
    - internal/udprepl/repl.go
    - internal/udprepl/repl_test.go
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - .planning/phases/03-persistence-dump-and-write-confirmation/03-REVIEW-DISPOSITION.md
    - .planning/phases/04-replication-and-deterministic-simulator/04-REVIEW-DISPOSITION.md

key-decisions:
  - "P4-WR-02: SyncResponseRate clamps to 500/s outside (0, 1e9] via maxSyncResponseRate — floor(1s/rate) stays non-zero, the limiter is never silently off"
  - "P4-WR-01: the drained-row Acked union is sort-then-Compact — a stranger's Ack (counting behavior) plus a discovery-flush re-add legitimately puts one peer in both sets"
  - "P4-IN-01: NotifyLocalWrite is one select over notify+done WITH a post-send re-check — a bare single select loses ~50% of post-Run calls to the uniform choice against the closed done and breaks the pinned 1100-call regression"
  - "P3/P4-IN-02: conditional dirty coded to Merge's Conflict contract — Winner=incoming is a displacement, Winner=stored a rejection; the conflicts-nil ambiguity (new key vs equal redelivery) resolved by a Get pre-probe; an equal-tombstone redelivery arms one redundant dump, the safe direction only"
  - "Restore no-op is decided WHOLE-snapshot, never per-record: the per-record content-skip broke the pinned fresh-version dominance contract (TestEngine_snapshotCreateLoadRestore) and was reworked before commit"
  - "P3-WR-01: the above-cover guard is kept verbatim in intent with an ADDED invariant assertion (a kept waiter must be unresolved; a resolved one is never re-queued) — belt documented AND asserted"
  - "P3-IN-01: Open decodes through Load's own oracle argument — Parse IS Decode's error channel, so the accepted generation is exactly the decoded one and the dead branch is gone (removal, not the comment-tightening the finding offered)"
  - "White-box tests ride a *_internal_test.go file (the repl_internal_test.go precedent) — the above-cover state is unreachable through the public API"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "SyncResponseRate domain clamp — out-of-domain rates (<=0, >1e9) normalize to 500/s; in-domain floor(1s/rate) arithmetic untouched"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_responseRateClamp (6-row table incl. 1e9=1ns and 1e9+1=2ms boundary); full ^TestSync_ suite under -race"
        status: pass
    human_judgment: false
  - id: D2
    description: "Drained-row Acked dedup — the mirror never publishes a peer twice in a drained row's Acked union"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "internal/udprepl/repl_test.go#TestMirror_drainedAckedDedup (mutation-checked: RED without slices.Compact); full ^TestMirror_ suite under -race"
        status: pass
    human_judgment: false
  - id: D3
    description: "Closed-loop NotifyLocalWrite — deterministic ErrRunLoopExited after Run's exit, no hang, no silent nil, no panic"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "internal/udprepl/repl_test.go#TestNotifyLocalWrite_afterDone + pinned TestNotifyLocalWrite_afterRunReturns (1100 calls) green unmodified; full udprepl suite under -race"
        status: pass
    human_judgment: false
  - id: D4
    description: "Conditional dirty — equal/stale remote redeliveries and a matching-snapshot Restore trigger zero dumps; a changed record still arms a dump that covers waiters (PERS-04 intact)"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_redundantDirty cases (a)/(b) in synctest (mutation-checked both directions); full persist suite incl. pinned snapshot/restore dominance test unmodified"
        status: pass
    human_judgment: false
  - id: D5
    description: "confirmWaiters keeps the above-cover guard AND carries the invariant assertion (kept waiters provably later resolved or failed; resolved waiters never re-queued)"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "internal/app/persist/engine_internal_test.go#TestConfirmWaiters_aboveCoverInvariant (3 subtests; both mutations RED — guard removal and assertion removal)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Open's dead Decode-error fallback removed — decoding rides Load's oracle argument; fallback chain (main -> .prev -> empty) unchanged"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "TestEngine_endToEndRestart + TestEngine_openEmptyStart pass unmodified; go build/vet clean"
        status: pass
    human_judgment: false
  - id: D7
    description: "Disposition ledgers flipped in the same commits as their fixes (03: WR-01/IN-01/IN-02; 04: WR-01/IN-01/WR-02/IN-02), each row naming its regression test"
    verification:
      - kind: other
        ref: "git show --stat d7c5f4e / 82180f7 / 0f5e03d (disposition files in each fix commit); grep '| fixed |' counts 03=4, 04=5"
        status: pass
    human_judgment: false

duration: 22min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 01: Hardening Backlog Part 1 (D-15) Summary

**All six D-15 replication/persistence review fixes landed with mutation-checked regressions: egress-rate domain clamp, drained-row Acked dedup, atomic closed-loop NotifyLocalWrite, contract-coded conditional dirty (ApplyRemote + whole-snapshot Restore), the confirmWaiters invariant assertion beside the kept guard, and Open's dead Decode fallback removed — 03/04 disposition ledgers flipped, whole-repo race+lint green.**

## Performance

- **Duration:** 22min
- **Started:** 2026-11-15T21:52:05Z
- **Completed:** 2026-11-15T22:14Z (UTC)
- **Tasks:** 3/3
- **Files modified:** 9 (1 created)

## Accomplishments

- **P4-WR-02 rate clamp**: `normalized()` clamps `SyncResponseRate` outside (0, 1e9] to the 500/s default (`maxSyncResponseRate`); `TestSync_responseRateClamp` pins exact spacing nanoseconds on both boundary sides (1ns at 1e9, 2ms at 1e9+1) behaviorally through the egress queue.
- **P4-WR-01 Acked dedup**: `retainDrained`'s union is sort-then-Compact; `TestMirror_drainedAckedDedup` drives the stranger-Ack + discovery-flush re-add shape through the real socket/event loop and asserts each peer exactly once in the published drained row (mutation-checked RED without the Compact).
- **P4-IN-01 NotifyLocalWrite**: one select over notify+done with a post-send re-check; `TestNotifyLocalWrite_afterDone` pins the prompt error return and the pinned 1100-call `TestNotifyLocalWrite_afterRunReturns` stays green unmodified.
- **P3-IN-02 + P4-IN-02 conditional dirty**: ApplyRemote marks dirty only on actual state change (Merge's Conflict contract + live pre-probe); Restore skips only a WHOLE-snapshot match; `TestEngine_redundantDirty` proves both directions (no-op → zero dumps; greater version → one covering dump resolving the local waiter — PERS-04 intact).
- **P3-WR-01 belt + assertion**: the above-cover guard is kept verbatim in intent and now carries its invariant assertion; white-box `TestConfirmWaiters_aboveCoverInvariant` (3 subtests, both mutations RED).
- **P3-IN-01 dead fallback removed**: Open decodes through Load's own oracle argument — no unreachable branch remains; existing Open tests pass unmodified.

## Task Commits

1. **Task 1 (tracer): rate clamp + Acked dedup** — `d7c5f4e` (fix) — tracer feedback gate re-ran verify end-to-end under auto mode: PASS
2. **Task 2: NotifyLocalWrite TOCTOU** — `82180f7` (fix)
3. **Task 3: persist engine fixes** — `0f5e03d` (fix)

**Plan metadata:** (this commit) `docs(06-01): complete hardening backlog part 1 plan`

## Files Created/Modified

- `internal/app/repl/core.go` — maxSyncResponseRate const + normalized() clamp + Config/normalized docs
- `internal/app/repl/sync_test.go` — TestSync_responseRateClamp (6-row table)
- `internal/udprepl/repl.go` — drained-row Compact + retainDrained doc; NotifyLocalWrite single-select+recheck + doc
- `internal/udprepl/repl_test.go` — TestMirror_drainedAckedDedup, TestNotifyLocalWrite_afterDone
- `internal/app/persist/engine.go` — Open oracle decode, confirmWaiters assertion, conditional ApplyRemote/Restore dirty, restoreChanges
- `internal/app/persist/engine_test.go` — TestEngine_redundantDirty (synctest, two cases)
- `internal/app/persist/engine_internal_test.go` — NEW: white-box TestConfirmWaiters_aboveCoverInvariant
- `03/04-REVIEW-DISPOSITION.md` — six D-15 rows flipped to fixed, each naming its regression test, in the same commits as the fixes

## Decisions Made

See key-decisions frontmatter; all seven are D-15 fix-shape decisions recorded per finding.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] NotifyLocalWrite "single select" needed the post-send re-check**
- **Found during:** Task 2
- **Issue:** The plan's literal "single select over the notify channel and the done channel" is insufficient: once Run has returned (done closed) and the 1024-buffer has room, BOTH select cases are permanently ready and Go's uniform choice returns nil for ~half the calls — silently reporting success for notifications nobody drains, and failing the pinned 1100-call TestNotifyLocalWrite_afterRunReturns regression.
- **Fix:** Single select + non-blocking done re-check after a won send — exactly the review's own alternative fix shape ("a loop-liveness flag checked after a successful send"). Post-Run behavior is deterministic; no pre-check remains.
- **Files modified:** internal/udprepl/repl.go
- **Verification:** TestNotifyLocalWrite_afterDone + full udprepl suite under -race green
- **Committed in:** 82180f7

**2. [Rule 1 - Bug] Restore per-record content-skip broke the pinned dominance contract**
- **Found during:** Task 3 verification (full persist suite)
- **Issue:** The first Restore implementation skipped replay writes per-record on matching content; TestEngine_snapshotCreateLoadRestore failed — restore's fresh-version dominance is load-bearing (a skipped live record keeps an old version that in-flight peer state can beat), so per-record skipping is semantically wrong, not just an optimization.
- **Fix:** Reworked before commit: the no-op is decided at the whole-snapshot level (restoreChanges compares every snapshot record — version, data, tombstone — against the store); any difference replays in full with every record re-stamped fresh. Plan's case (a) still holds (a snapshot equal to the current state replays as a no-op, zero dumps).
- **Files modified:** internal/app/persist/engine.go
- **Verification:** Full persist suite green including the unmodified pinned snapshot/restore test; TestEngine_redundantDirty mutation-check (unconditional-dirty mutant RED)
- **Committed in:** 0f5e03d

**3. [Rule 3 - Blocker] White-box test file required (new file not in plan's files list)**
- **Found during:** Task 3
- **Issue:** TestConfirmWaiters_aboveCoverInvariant must construct the above-cover waiter state, which is provably unreachable through the public API (await bumps lastLocal before registering under the same lock).
- **Fix:** Created engine_internal_test.go (package persist) following the udprepl repl_internal_test.go precedent.
- **Files modified:** internal/app/persist/engine_internal_test.go (new)
- **Verification:** go test -race ./internal/app/persist/ green; lint clean
- **Committed in:** 0f5e03d

**4. [Rule 1 - Lint] Doc-link and inline-err lint shapes in new code**
- **Found during:** Tasks 1 and 3 lint gates
- **Issue:** godoclint required link-form stdlib references in the new constant's doc; noinlineerr flagged inline error handling in the new tests.
- **Fix:** Rephrased the constant doc (no raw conversion expression); converted flagged test sites to plain assignments.
- **Files modified:** internal/app/repl/core.go, internal/app/persist/engine_test.go, internal/app/persist/engine_internal_test.go
- **Verification:** go tool golangci-lint run ./... exit 0
- **Committed in:** d7c5f4e, 0f5e03d

---

**Total deviations:** 4 auto-fixed (1 Rule 1 bug, 3 Rule 3 blockers/lint-shape; none architectural — no Rule 4 checkpoints)
**Impact on plan:** None on scope — all six fixes landed as specified; two fix MECHANISMS were refined (both documented above and in disposition rows), neither changed a finding's semantics.

## Issues Encountered

None beyond the deviations above. No auth gates (pure in-repo work).

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - every fix ships with a binding regression test; no pinned suite was modified.

## Next Phase Readiness

- The egress-pacing, drained-row mirror, and dump-confirmation contracts that every TEST-04 failure scenario (06-02/06-04) asserts through are now bug-free and pinned — the scenarios will test the real contract.
- 03-REVIEW-DISPOSITION open items are down to IN-03 (checksum collapse) and IN-04 (snapshots dir on ErrNoDump), both info-level and untouched by this plan; 04's ledger is fully closed.
- Tracer feedback gate (Task 1) passed under auto mode: verify re-ran end-to-end green before expansion tasks.

## Self-Check: PASSED

- key-files.created exists on disk: internal/app/persist/engine_internal_test.go FOUND
- Commits reachable from HEAD: d7c5f4e, 82180f7, 0f5e03d FOUND (merge-base --is-ancestor ok)
- evaluation-scope --plan 06-01 --commits-only: resolved, 3 commits
- All task acceptance criteria re-run: PASS (clamp code+boundaries, sort-then-Compact, disposition rows 03=4/04=5 fixed, lint scopes exit 0, no pre-check select shape, Open oracle decode)
- Plan-level verification: go test -race ./... exit 0; go tool golangci-lint run ./... exit 0; disposition diffs in same commits as fixes (git show --stat)
- Windows ledger: no stubs, skipped tests, or unrun verifies to record

---
*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
