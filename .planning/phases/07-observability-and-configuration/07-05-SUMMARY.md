---
phase: 07-observability-and-configuration
plan: 05
subsystem: observability
tags:
  - obsv-05
  - obsv-03
  - received-lead-drift
  - drift-counter
  - listing-part-maxima
  - boundary-table
  - two-minute-clock-lead
requires:
  - phase: 07-observability-and-configuration
    provides: plan 07-02's 2 m max-clock-lead default and the millisecond-resolution drift_vs_lead rule that make the warning reachable
  - phase: 04-protocol
    provides: the Observe seam and the DataSync part maximum the listing branch counts
  - phase: 03-transport
    provides: the validation pipeline position that keeps a beyond-bound version a bad_version discard
provides:
  - Clock.Update reporting the received reading's lead against the wall clock (Tick keeps the local-vs-wall diagnostic)
  - positive-lead-only drift counting in applyRemote, including the equal-version tie-break path
  - one DriftWarnings count per offending DataSync part maximum
  - boundary, saturation and per-op property models for the two drift conventions
  - 2 m-aligned version property window, near-version generator and far-ahead scenario comments
affects:
  - 07-07 (the OnDrift WARN and limiter consume the now-truthful counter definition)
  - 07-06 (the conflict rows are seeded through localWrite for the pending-only definition)
  - 07-10 (the runbook edit can now describe the shipped condition, not a fictional one)
actuals:
  tokens: 9744
  tasks: 2
  commits: 4
  plan_head_before: b41d6b58743bc604d7c0216d0587a95484ef3380
  plan_head_after: 4a6ff542590af81bec2639490d409e67972619f1
tech-stack:
  added: []
  patterns:
    - "Per-op drift models: Tick's operand is the clock's own l, Update's is the received reading, so one shared expectedDrift would silently bless either convention — the property walks select the model by the branch that produced the reading."
    - "Boundary rows at millisecond resolution: exactly drift_warn is zero and drift_warn+1 ms is positive, pinning the strict \"more than\" relation the Spec states."
    - "The operator signal stays a counter in the core; the WARN/event seam is a separate plan, so the counter can be made truthful before the limiter lands."
key-files:
  created: []
  modified:
    - internal/app/clock/clock.go
    - internal/app/clock/clock_test.go
    - internal/app/clock/hlc_prop_test.go
    - internal/app/storage/store_test.go
    - internal/app/repl/core.go
    - internal/app/repl/merge_stats_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/sim_scenario_test.go
    - internal/app/internal/testutil/gen.go
    - internal/app/version/validate_test.go
key-decisions:
  - "Update compares remoteL with the wall clock and nothing else: a row with a local l far ahead and an in-tolerance received reading asserts zero, so a regression back to the local operand fails the table."
  - "Only a positive drift is counted in applyRemote; the negative return stays observable in the clock (a wall that jumped past the received reading) but is never a warning, matching the Spec's one-direction condition."
  - "handleDataSync counts from maxEntryVersion before accumulation: one part maximum, never per entry, so a listing of N offending entries cannot inflate the counter (T-07-16)."
  - "Behavior test 6 (an equal-version tie-break loss with a leading stamp) lives in its own TestMergeStats_equalVersionTieBreakCountsDrift: a leading remote reading is absorbed first so the real localWrite helper stamps a leading version, keeping the row on the production author path."
  - "The tie-break test asserts only the drift counter and the surviving stored record, not the conflict count: plan 07-06's pending-only definition decides whether a lost tie-break conflicts, and this plan must not pre-empt it."
patterns-established:
  - "Pattern: a drift boundary test names the operand per op in the table struct (initialL vs remoteL) instead of sharing one field, so a row cannot accidentally test the other operation."
requirements-completed: []
coverage:
  - id: D1
    description: "Clock.Update reports the received version's lead: exactly drift_warn is zero, drift_warn+1 ms is positive, a behind-the-wall reading is zero even with a local l far ahead, a forward wall jump is negative, and a disabled warn is silent; Tick's local-vs-wall signal is unchanged"
    requirement: OBSV-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/clock/ -run 'TestClock_driftSignal|TestClock_driftSignalSaturates' -v"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/clock/ -run 'TestClock_monotonic|TestClock_driftUnderChaos' -v"
        status: pass
    human_judgment: false
  - id: D2
    description: "applyRemote counts a drift warning only for a positive received lead, including a record that loses the equal-version tie-break; a behind-the-wall record and a bad_version record count zero"
    requirement: OBSV-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestMergeStats_counts|TestMergeStats_equalVersionTieBreakCountsDrift' -v"
        status: pass
    human_judgment: false
  - id: D3
    description: "A DataSync part's maximum entry version counts exactly one warning when it leads beyond drift_warn, even with two offending entries in the part; a stale DataReplication still counts nothing"
    requirement: OBSV-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestMergeStats_listingAndStaleExcluded' -v"
        status: pass
    human_judgment: false
  - id: D4
    description: "The warning is reachable at the shipped defaults: a +66 s record inside the 2 m max_clock_lead is accepted and counted, a +2 m 1 s record is discarded as bad_version, and the storage Observe return is the received stamp's lead on its first observation"
    requirement: OBSV-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestMergeStats' -v"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run 'TestStore_observeReturnsDrift' -v"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every test-model artifact that must straddle the inbound bound describes 2 m: validateLead and the property window, nearFutureMs, the retransmit seam comment and the far-ahead scenario comments; the version property still draws both accepted and future readings"
    requirement: OBSV-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/version/ -run 'TestValidate' -v"
        status: pass
      - kind: contract
        ref: "grep -rn '60 s production default|60 s default MaxClockLead|absorb-then-warn|post-absorption apply' internal/ (no max-clock-lead claim remains; the stand startup-gate 60 s is unrelated)"
        status: pass
    human_judgment: false
  - id: D6
    description: "MergeStats documents the received-record and listing-part-maxima definition, and no code path counts a negative drift"
    requirement: OBSV-03
    verification:
      - kind: contract
        ref: "grep -n 'received record and DataSync part maximum' internal/app/repl/core.go; grep -n 'if drift > 0' internal/app/repl/core.go (two sites)"
        status: pass
      - kind: integration
        ref: "make lint && make test (whole repo; the known Phase-8 ratio flake passed standalone)"
        status: pass
    human_judgment: false
duration: 19min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 05: Received-Lead Clock-Drift Signal Summary

**Made the clock-drift warning measure the received version's lead against local wall time — the exact Spec condition — and counted it once per received record and once per offending DataSync part, so the metric is reachable at the shipped 60 s/2 m defaults instead of being an unreachable local-versus-wall diagnostic.**

## Performance

- **Duration:** 19min
- **Started:** 2026-11-23T08:15:47Z
- **Completed:** 2026-11-23T08:35:01Z
- **Tasks:** 2 (both TDD)
- **Files modified:** 10 (0 created, 10 modified)

## Accomplishments

- `Clock.Update` now evaluates `driftBetween(remoteL, ptMs, c.driftWarn)` — the received reading against the wall clock. `Tick` keeps its local-versus-wall diagnostic and the Kulkarni absorb, counter logic, saturation band and the `driftBetween`/`clampDriftRange` bodies are byte-identical. The `NewClock` and `Update` docs state which operand each operation measures.
- `TestClock_driftSignal` carries a `remoteL` column and thirteen rows: the six Tick rows unchanged plus exactly-warn (0), warn+1 ms (positive), behind-the-wall (0), local-l-far-ahead (0, the regression guard against the old operand), forward wall jump (negative), and zero/negative warn disabling the signal. The saturation Update subtest now passes `remoteL = maxL` with a normal wall; the Tick saturation cases are untouched.
- The two adversarial walks (`TestClock_monotonic`, `TestClock_driftUnderChaos`) split the single model into `expectedTickDrift` and `expectedUpdateDrift`, selected by the branch that produced the reading; both signs still fire (backward wall jump → positive received lead, forward jump → negative).
- `TestStore_observeReturnsDrift` asserts the received-lead truth: the first far-ahead `Observe` returns a positive drift and a repeat stays positive (the signal depends on the stamp, not on prior absorption); the in-tolerance row keeps zero.
- `applyRemote` counts only `drift > 0`; `handleDataSync` captures `Observe(maxEntryVersion(msg.Entries))` and counts one warning per offending part maximum, before accumulation, with no per-entry inflation.
- `MergeStats`' doc defines `DriftWarnings` as received records and DataSync part maxima whose version leads local wall time by more than `drift_warn`, regardless of the merge outcome. No WARN/event seam was added — plan 07-07 owns `OnDrift` and the limiter.
- `TestMergeStats_counts` now expects the far-ahead warning on the FIRST apply, adds the behind-the-wall and bad_version rows, and seeds the displaced local record through the real `localWrite` helper so plan 07-06's pending-only conflict definition inherits a valid row. `TestMergeStats_equalVersionTieBreakCountsDrift` covers the tie-break case, and `TestMergeStats_listingAndStaleExcluded` counts exactly one for a two-entry offending part.
- All inbound-bound test models describe 2 m: `validateLead`/`windowMs` (2 m / 121 s), `nearFutureMs` (150 s), the `newTestCoreWithConfig` seam comment and the two far-ahead scenario comments; no sentence still calls 60 s the max-clock-lead default (`drift_warn`'s own 60 s is unchanged).

## Task Commits

1. **Task 1 RED: pin the received-lead drift contract at its boundaries** - `b918b98` (test)
2. **Task 1 GREEN: measure the received reading's lead in Clock.Update** - `53413e4` (feat)
3. **Task 2 RED: pin positive-lead counting and listing maxima** - `d310439` (test)
4. **Task 2 GREEN: count only positive received leads and listing maxima** - `4a6ff54` (feat)

**Plan metadata:** `docs(07-05): complete received-lead drift plan` (this commit)

## Files Created/Modified

- `internal/app/clock/clock.go` - `Update`'s drift operand is `remoteL`; the `Update` and `NewClock` docs name the per-op convention; nothing else touched
- `internal/app/clock/clock_test.go` - the `remoteL`-bearing boundary table and the rebuilt saturation Update case
- `internal/app/clock/hlc_prop_test.go` - `expectedTickDrift`/`expectedUpdateDrift` and their selection in both adversarial walks
- `internal/app/storage/store_test.go` - `TestStore_observeReturnsDrift` rewritten to the received-lead first-observation truth
- `internal/app/repl/core.go` - `drift > 0` counting in `applyRemote`, one-per-part counting in `handleDataSync`, and the `MergeStats` definition
- `internal/app/repl/merge_stats_test.go` - the rewritten counts/listing tests, the new tie-break test and the `driftLead` comment
- `internal/app/repl/retransmit_test.go` - the `newTestCoreWithConfig` seam comment names the 2 m production default
- `internal/app/repl/sim_scenario_test.go` - the two "production 60 s default" comments moved to 2 m and the far-ahead sequence comment moved to received-lead
- `internal/app/internal/testutil/gen.go` - `nearFutureMs` 150 s and both straddle comments
- `internal/app/version/validate_test.go` - `validateLead` 2 m and the property `windowMs` 121 s

## Decisions Made

- **`remoteL`, never `c.l`, in Update**: the "local l far ahead" table row makes a revert to the old operand fail immediately, which is the defect this plan exists to close.
- **Negative drifts stay observable but uncounted**: `driftBetween` remains signed and saturation-safe; the core's `> 0` gate implements the Spec's one-direction warning without weakening the clock's diagnostic.
- **One count per DataSync part maximum**: counting happens before accumulation from the already-existing `maxEntryVersion`, so the per-part bound (T-07-16) holds and no per-entry loop touches the counter.
- **Tie-break row on the real author path**: a leading remote reading is absorbed under another key first so `localWrite` stamps a leading version; this keeps the row honest without hand-packing a local version.
- **No conflict assertion in the tie-break row**: plan 07-06 owns the pending-only conflict definition, so this plan asserts the drift counter and the surviving record only.
- **Behavior test 6 in its own function**: the equal-version tie-break drift case is clearer as `TestMergeStats_equalVersionTieBreakCountsDrift` than as one more step in the cumulative counter sequence.

## Deviations from Plan

None - plan executed exactly as written. The two TDD tasks landed as separate test/feat commits each, with the 2 m test-model alignment in Task 2's test commit as the plan directs.

## Issues Encountered

- `TestCore_keyCountCostFlat` failed in the full `make test` run (DataSync 1000 keys 68 170 ns/op vs 50 000 keys 247 809 ns/op, ratio 3.64 > 2.0). It is the recorded Phase-8 wall-clock ratio flake (STATE.md, Phase-04/05 notes): a standalone rerun passed at ratio 0.61 (37 523 vs 22 806 ns/op), and no bound or test was modified. `make lint` was green on the whole repo.
- `requirements.ready-ids` reports OBSV-05 and OBSV-03 blocked: both ids are shared across sibling Phase-07 plans (#2388 shared-ID gate), so no checkbox was marked and `requirements-completed` stays empty; 07-11 closes the phase-level checkboxes.
- The stale `gsd-plan-head-before-07-05` ledger pointed at `07f24a3` (not in this branch's history); it was reset to the plan-start HEAD `b41d6b5` before measuring, following the 07-02 precedent.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The drift counter and the storage Observe return now describe exactly the Spec condition, so plan 07-07 can attach `OnDrift` and its per-peer limiter to a truthful counter without redefining it, and plan 07-10 can describe the shipped condition in the runbook. The conflict rows are seeded through `localWrite`, leaving plan 07-06's pending-only definition a clean change. No blockers; 6 plans remain in Phase 07 — ready for 07-06.

## Self-Check: PASSED
