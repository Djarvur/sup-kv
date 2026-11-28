---
phase: 07-observability-and-configuration
plan: 06
subsystem: observability
tags:
  - obsv-03
  - adr-0004
  - conflict-counter
  - pending-displacement
  - apply-failure-warn
  - warn-limiter
  - stand-metrics
  - real-socket
requires:
  - phase: 07-observability-and-configuration
    provides: plan 07-05's localWrite-seeded conflict row and the received-lead counter, leaving only the conflict definition to change
  - phase: 04-protocol
    provides: the Outbox Superseded result seam and the single replicated-apply path
  - phase: 05-persistence
    provides: Merge/MergeChanged's per-outcome Conflict return, which this plan stops reading for the metric
provides:
  - supkv_merges_conflicts_total counting only pending (unacked) local writes displaced by an applied replicated record
  - one post-apply stored-record read serving both the reply verdict and the pending-displacement walk
  - a keyless apply-failure WARN rate-limited per (node_id, apply_failure) through the shared limiter
  - a real-socket stand proof that constructs exactly one honest conflict by stopping the peer and crafting a higher-version valid DataReplication
  - a corrected storage.Conflict doc that no longer claims to feed the conflict counter
affects:
  - 07-10 (the runbook's metric table can now describe the shipped definition)
  - 07-11 (the OBSV-03 phase checkbox closes once the sibling metric plans land)
  - Phase 8 (the chaos scenario no longer asserts an aggregate conflict count)
actuals:
  tokens: 12419
  tasks: 2
  commits: 3
  plan_head_before: 594f7790f8ca60cb369214b60fe86a8efcd0cce2
  plan_head_after: 355f8aab53a962e1183a5f2c404c928b47dfb0e7
tech-stack:
  added: []
  patterns:
    - "Metric definitions from producer state: the conflict count walks the core's pending set after a successful apply instead of trusting the store's per-outcome Conflict slice, so an ordinary overwrite can never inflate it."
    - "One post-apply read for two consumers: the stored record decides the Ack/answer verdict and whether an equal-version incoming displaced the pending write, so the reply and the count cannot disagree."
    - "Constructed exactness over aggregate evidence: the real-socket conflict proof stops the peer so the write stays unacked, then injects one valid higher-version datagram; a chaos total is logged, never asserted as a definition."
key-files:
  created:
    - .planning/phases/07-observability-and-configuration/red-07-06-task1-conflict.json
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/merge_stats_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/sim_scenario_test.go
    - internal/app/repl/warn_test.go
    - internal/app/storage/store.go
    - internal/stand/stand_metrics_test.go
key-decisions:
  - "Conflicts is incremented only inside the pending walk: a strictly greater incoming version, or an equal version the incoming record won on content, displaces the tracked write (count 1 plus Superseded); every other outcome counts 0."
  - "The equal-version walk is new: the old loop only handled strictly greater versions, so a tie-break win that displaced a pending write was silently uncounted and left the entry tracked."
  - "incomingWon is computed once from the stored record after the apply and reused by the reply branch — the Ack/answer verdict and the displacement count can never disagree."
  - "The apply-failure WARN keeps its rate-limit slot keyed by (node_id, apply_failure) through warnRateLimited, and the message carries the quoted handle only; the key name is gone from the text (T-07-17, T-07-18)."
  - "The stand conflict uses the real store's ValueMaxBytes path (newLimitedTestCore) rather than a hand-written fake returning ErrValueTooLarge: the production refusal path is exercised and the assertion set is unchanged."
  - "The chaos scenario logs the observed aggregate conflict count instead of asserting >= sharedKeys: a scheduler-dependent total cannot prove a definition, and the constructed unit/stand rows can."
patterns-established:
  - "Pattern: a counter definition is proven by a constructed row per outcome (pending displacement 1; confirmed-local, remote-to-remote, stale, identical and refused 0), not by an aggregate scenario total."
  - "Pattern: a log WARN that can be driven per datagram rides the shared per-(node_id, class) limiter and names the peer handle, never record data."
requirements-completed: []
coverage:
  - id: D1
    description: "supkv_merges_conflicts_total derives from c.pending: a pending write displaced by a strictly greater incoming version or by an equal-version tie-break win counts 1 and emits Superseded; confirmed-local/remote overwrites, stale rejections, identical redeliveries and refused applies count 0; the MergeStats doc states the definition"
    requirement: OBSV-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestMergeStats' -v"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ (whole package)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The apply-failure WARN is rate-limited per (node_id, apply_failure) through the shared limiter, names only the quoted source handle, and changes no Ack/supersession behavior; the pending entry survives repeated failures"
    requirement: OBSV-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestApplyFailure_warnLimiterKeyedByNodeAndClass|TestDiscard_warnLimiterKeyedByNodeAndClass' -v"
        status: pass
    human_judgment: false
  - id: D3
    description: "storage.Conflict's doc names MergeChanged/Merge's per-outcome return and no longer the Phase 4/6 journal; no store code line changes and the red-runner's out[key] = rec.Version mutation target is byte-identical"
    requirement: OBSV-03
    verification:
      - kind: contract
        ref: "git diff a98b541~1 a98b541 -- internal/app/storage/store.go is comment-only; the red-runner sed dry-run still rewrites out[key] = rec.Version and the periodic-sync core.go sed still applies"
        status: pass
    human_judgment: false
  - id: D4
    description: "TestStand_metricsByEffect constructs one pending-displacement conflict on real sockets — B stopped first, A's write left unacked, one crafted wire-valid higher-version DataReplication displacing it — and asserts conflicts == 1 exactly, with bad_crc == 1 unchanged and retransmit >= 1 reusing the single stopped peer"
    requirement: OBSV-03
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_metricsByEffect$' -v (logs conflicts=1 bad_crc=1 retransmitted=1)"
        status: pass
      - kind: contract
        ref: "make lint (whole repo)"
        status: pass
    human_judgment: false
duration: 27min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 06: Pending-Displacement Conflict Counter and Keyless Apply-Failure WARN Summary

**Made `supkv_merges_conflicts_total` count only the displacement of a pending (unacked) local write — computed from the core's pending set after the apply, never from the store's per-outcome `Conflict` slice — and made the apply-failure WARN keyless and rate-limited, with a rebuilt real-socket stand scenario that still asserts exactly one conflict.**

## Performance

- **Duration:** 27min
- **Started:** 2026-11-23T09:07:34Z
- **Completed:** 2026-11-23T09:34:28Z
- **Tasks:** 2 (Task 1 TDD tracer, Task 2 auto)
- **Files modified:** 8 (1 created, 7 modified)

## Accomplishments

- `applyRemote` discards `ApplyRemote`'s `Conflict` slice (`_, drift, err :=`) and derives the metric from `c.pending`: a strictly greater incoming version, or an equal version the incoming record won on content, increments `Conflicts`, emits `Superseded(tracked)`, and removes the entry. A pending version the incoming record did not beat stays tracked; refused applies return before both counters.
- One post-apply `Lookup` now feeds both decisions: `incomingWon` is read once and reused by the Ack/answer branch, so the reply verdict and the displacement count cannot disagree.
- `MergeStats`' doc defines `Conflicts` as pending (unacked) local writes displaced by an applied replicated record, explicitly excluding ordinary overwrites, stale rejections and identical redeliveries, and names the pending set as the source (ADR 0004).
- `storage.Conflict`'s doc now describes `MergeChanged`/`Merge`'s per-outcome return (tie-break included) and states plainly that it is not the conflict counter's source. No store code line changed — the red-runner's `out[key] = rec.Version` mutation target is byte-identical.
- `TestMergeStats_counts` was rebuilt as a fresh-core matrix: pending displacement 1 (with `Superseded` and the store's served version asserted), confirmed-local overwrite 0 (through a real Ack drain), remote-to-remote overwrite 0, equal-version tie-break win over pending 1 (with `Superseded`), tie-break loss 0 (pending entry still tracked), identical redelivery 0, stale incoming 0, refused apply 0. Plan 07-05's DriftWarnings rows (first far-ahead apply 1, behind-the-wall 0, beyond-the-lead bad_version 0) stay in the same test.
- `TestReplication_equalVersionContentLaw` and `TestReplication_newerApplied` were rebased to 0 conflicts with their docs rewritten: they seed the store directly, so no pending entry exists to displace.
- `sim_scenario_test.go` no longer asserts an aggregate `conflicts >= sharedKeys`; it logs the observed total, with a comment stating that the definition is proven by the constructed unit rows and `TestStand_metricsByEffect`.
- The apply-failure path now calls `warnRateLimited(now, nodeID, warnClassApplyFailure, ...)`; `applyRemote` takes `in.NodeID`, `handleMessage` passes it, and the message carries only the quoted source handle — one WARN per peer per `discardWarnInterval`, with counters and the unacked gauge still per datagram (T-07-17, T-07-18).
- `TestApplyFailure_warnLimiterKeyedByNodeAndClass` proves one WARN per peer/interval, no key text, no Ack, no supersession, the surviving pending entry, a second peer's own slot, and the boundary re-fire.
- `TestStand_metricsByEffect` now stops node B once before the conflict effect, waits for A's peer gauge to show B first, writes `standMetricsConflictKey` on A (unacked), and sends one crafted wire-valid `DataReplication` (16-byte `PackVersion(now+5 s, 0, 999)`, real record checksum, supported proto version, real CRC) to A's real UDP port. The final assertion stays an exact `conflicts == 1`; the bad_crc effect is unchanged and the retransmit effect reuses the already-stopped peer.

## Task Commits

1. **Task 1 RED: pin pending-displacement conflict counting** - `49befcf` (test)
2. **Task 1 GREEN: count only pending-write displacement as a conflict** - `a98b541` (feat)
3. **Task 2: rate-limit the apply-failure WARN and rebuild the stand conflict** - `355f8aa` (feat)

**Plan metadata:** `docs(07-06): complete pending-displacement conflict plan` (this commit)

## TDD Gate Compliance

- RED: `49befcf` (`test(07-06):`), validated with `gsd_run check tdd-red-evidence` → `RED_EVIDENCE_OK` (target `TestMergeStats_counts`, five failing rows, semantic assessment: four rows counted 1 under the old `len(conflicts)` definition and the tie-break win emitted no `Superseded`).
- GREEN: `a98b541` (`feat(07-06):`) descends from the RED commit; both plan `<verify>` commands pass.
- REFACTOR: not needed — the GREEN diff was the minimal change, and the only follow-up was a lint-driven style fix folded into Task 2.

## Files Created/Modified

- `internal/app/repl/core.go` - pending-displacement counting and the single post-apply read in `applyRemote`, the `apply_failure` limiter class and nodeID parameter, and the corrected `MergeStats`/`applyRemote` docs
- `internal/app/repl/merge_stats_test.go` - the six-outcome conflict matrix with the drift rows, the `Superseded`/served-version assertions, and a lint-clean beyond-lead probe
- `internal/app/repl/retransmit_test.go` - seed-only tie-break and newer-apply rows rebased to 0 conflicts, docs rewritten
- `internal/app/repl/sim_scenario_test.go` - aggregate conflict assertion replaced by a logged observation with a definition-provenance comment
- `internal/app/repl/warn_test.go` - `TestApplyFailure_warnLimiterKeyedByNodeAndClass`
- `internal/app/storage/store.go` - `Conflict` doc comment only; every code line untouched
- `internal/stand/stand_metrics_test.go` - rebuilt effect 1 (single stop, pending write, crafted valid higher-version datagram), effect 3 reuse of the stopped peer, and the new encoder helper
- `.planning/phases/07-observability-and-configuration/red-07-06-task1-conflict.json` - the validated RED evidence record

## Decisions Made

- **Count from producer state, not from the store's report**: `c.pending` is the only honest source of "unacked local write", and the store's `Conflict` slice reports every LWW outcome by contract (ADR 0004).
- **Equal-version wins count**: the tie-break win displaced a pending record, so it is a conflict; the loss leaves the entry tracked, because the tracked version still owns the stored record.
- **One post-apply read**: computing `incomingWon` once removed the second `Lookup` and guarantees the reply branch and the counter use the same verdict.
- **Apply-failure WARN stays a limiter citizen**: reuse `warnRateLimited` with a dedicated class token rather than a second throttle; the counters and the unacked gauge remain per datagram.
- **Stand exactness preserved**: the construction changed, the assertion did not — `conflicts == 1` remains an equality because exactly one pending displacement is constructed.
- **Real limited store instead of a fake**: the warn test drives the production `ErrValueTooLarge` path through `newLimitedTestCore`, which exercises the same refusal the stand value-limit scenario does.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Lint gate rejected an inline error check in the new test**
- **Found during:** Task 2 verification (`make lint`)
- **Issue:** `noinlineerr` rejected `if _, err := st.store.Get(keyBeta); err == nil` in the beyond-lead row added by Task 1's RED commit.
- **Fix:** Split into a plain `_, err := ...` assignment followed by the `err == nil` check; behavior unchanged.
- **Files modified:** `internal/app/repl/merge_stats_test.go`
- **Commit:** `355f8aa`

### Intentional plan deviations

- **Warn-limiter fixture:** the plan asked for a "fake store whose `ApplyRemote` returns `storage.ErrValueTooLarge`". The test instead uses the repository's existing `newLimitedTestCore(8, &logs)`, whose real store returns exactly `storage.ErrValueTooLarge` for the oversize record. The production refusal path is exercised and every asserted behavior (one WARN, no key, no Ack, no supersession, pending survives, second peer's slot) is unchanged.
- **Task 2 commit shape:** Task 2 is `type="auto"`, so its test and production changes landed in one `feat` commit, with the lint fix above folded in rather than amending the RED commit.

## Issues Encountered

- `TestCore_keyCountCostFlat` failed in the full `make test` run (DataSync 1000 keys 69 958 ns/op vs 50 000 keys 160 346 ns/op, ratio 2.29 > 2.0). This is the recorded Phase-8 wall-clock ratio flake (STATE.md Phase-04/05 notes; also seen by 07-05): a standalone rerun passed at ratio 0.99 (23 164 vs 22 935 ns/op), and no bound or test was modified. `make lint` is green on the whole repo.
- `requirements.ready-ids` reports OBSV-03 blocked: the id is shared across sibling Phase-07 plans (#2388 shared-ID gate), so no checkbox was marked and `requirements-completed` stays empty; 07-11 closes the phase-level checkboxes.
- The red-runner's two mutation seds were dry-run against the changed files: the `storage/store.go` `out[key] = rec.Version` substitution and the `core.go` `syncListing` timer substitution both still apply, and the frozen `stand_red_test.go`, `red_gate_test.go` and `red-expectations.tsv` are byte-identical.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The metric now means what the review's engine evidence says, so the runbook (07-10) can describe the shipped condition and the remaining OBSV metric plans can build on a truthful counter. The apply-failure path is flood-safe and keyless. The stand conflict proof matches the new definition while keeping its exact equality, and the chaos scenario no longer claims to prove a definition it cannot. No blockers; 5 plans remain in Phase 07 — ready for 07-07.

## Self-Check: PASSED

- Files verified: `internal/app/repl/core.go`, `internal/app/repl/merge_stats_test.go`, `internal/app/repl/warn_test.go`, `internal/stand/stand_metrics_test.go`, `internal/app/storage/store.go`, `07-06-SUMMARY.md`.
- Commits verified as ancestors of HEAD: `49befcf`, `a98b541`, `355f8aa`.

