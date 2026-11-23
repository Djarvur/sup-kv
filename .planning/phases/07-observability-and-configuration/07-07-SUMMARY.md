---
phase: 07-observability-and-configuration
plan: 07
subsystem: observability
tags:
  - obsv-01
  - obsv-05
  - obsv-06
  - lifecycle-catalog
  - typed-events
  - gate-open-reason
  - peer-events
  - drift-warn
  - startup-record
  - load-outcome
  - secrets-free
  - slog-attributes
requires:
  - phase: 07-observability-and-configuration
    provides: plan 07-04's run seam, setupLogging install and Startup-failed stage record that the startup record sits beside
  - phase: 07-observability-and-configuration
    provides: plan 07-05's received-lead DriftWarnings definition, which the drift event now narrates
  - phase: 07-observability-and-configuration
    provides: plan 07-06's shared per-(node_id, class) limiter, generalized into allowWarn and shared with clock_drift
  - phase: 06-api-and-semantics
    provides: grpcapi.ServerOptions, the gate resolver/health binding and RunGate's two opening paths
  - phase: 05-persistence
    provides: persist.LoadStats (Outcome/Generation/Decoded), DecodeStats.Total and Engine.KeyCount
provides:
  - grpcapi.ServerOptions.OnGateOpen(reason) fired once inside the gate CAS after the health flip, with reason sync/backstop
  - typed repl.PeerEvent (appear/restarted/evicted) and repl.DriftEvent plus nil-safe Config.OnPeerEvent/OnDrift seams
  - Core.allowWarn as the shared limiter slot logic, keyed by (node_id, clock_drift) for drift and unchanged for the discard/apply classes
  - one Startup gate opened INFO record, Peer appeared INFO, Peer restarted/evicted WARN, Clock drift WARN rendered once in cmd
  - one secrets-free Starting sup-kv INFO record with the build identity and every effective configuration knob
  - one load-outcome record per start: Dump load INFO for prompt/main, the existing WARN/ERROR extended with the same four attributes
affects:
  - 07-08 (the foreign proto_version branch and level matrix extend the same catalog)
  - 07-10 (the runbook can now name the emitted records and their fields)
  - 07-11 (OBSV-01/05/06 phase checkboxes close with this plan's unit and process evidence)
actuals:
  tokens: 13216
  tasks: 3
  commits: 4
  plan_head_before: 7222bc0c8fbce827d688323d1bbc80e002c04ac5
  plan_head_after: 0f68776e69da86175a2d7a220d5feb27be4a7591
tech-stack:
  added: []
  patterns:
    - "Typed transition, one renderer: the core emits a typed event (Kind + identities + measured age/lead) and cmd renders it exactly once, so the pure core stays slog-free and every transition is unit-testable without parsing strings."
    - "Reason inside the CAS: the gate callback fires in the same branch as the flag store and the health flip, making the event, the writable transition and SERVING one single-shot fact."
    - "One limiter slot function: allowWarn owns the (node_id, class) timestamp bookkeeping; warnRateLimited and emitDrift both go through it, so counters stay per event while the WARN is per interval."
    - "One record per start: the load outcome and its three companion attributes are built once and either carry the INFO record or join the pre-existing WARN/ERROR, never a second line."
key-files:
  created:
    - internal/app/repl/events.go
    - internal/app/repl/events_test.go
    - .planning/phases/07-observability-and-configuration/red-07-07-task2-events.json
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/warn_test.go
    - internal/grpcapi/gate.go
    - internal/grpcapi/gate_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
key-decisions:
  - "OnGateOpen is a typed string reason fired inside openGate's CAS branch after SetServingStatus, so a sync flip and the backstop can never produce two events."
  - "Peer events fire only on real transitions: first-seen (after peerGen++), an unseen run under a remembered id (after pushSeenRun, carrying the previous id), and the eviction loop before the delete; repeat/alternation/own echo emit nothing because HandleDatagram and handleHello return first."
  - "Drift rides allowWarn with the clock_drift token; the MergeStats counter increments before the limiter decision, so a suppressed event never suppresses the per-record truth."
  - "The drift log attribute is named handle rather than the catalog's source: sloglint forbids the source key (reserved by slog handlers); the event field remains Source and the peer rows already use handle."
  - "The load record's generation is derived from the outcome, not from port.Generation alone: the port's zero Generation is GenMain, so a prompt/unusable load would otherwise claim generation main."
  - "Startup secret discipline is structural: logStartup only reads MetricsAddr and MetricsCert != \"\"; the cert/key/CA paths are never referenced, and the process test asserts the three keys are absent (T-07-21)."
patterns-established:
  - "Pattern: a lifecycle seam is proven twice — a captured typed-event slice for the transition matrix and a real-binary JSON record for the rendered catalog entry."
  - "Pattern: one INFO record per start for the load outcome at every outcome, with fallback/unusable keeping their mandated level and gaining attributes rather than a second line."
requirements-completed: []
coverage:
  - id: D1
    description: "The startup gate opening is narrated once with its reason (sync/backstop) after the CAS and the health flip, including a real-binary INFO record and a clean SIGTERM exit"
    requirement: OBSV-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run 'TestGate_openReason|TestGate_opensOn|TestHealth_reflectsGate' -v"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMain_gateOpenLogged$' -v (one INFO record, reason=backstop, exit 0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Peer lifecycle events: appear (first-seen), restart (unseen run with previous and new ids), evict (silence age); repeat Hello, alternating run_id and own echo emit nothing; main renders each as one static-msg record with node_id/handle/run_id"
    requirement: OBSV-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestPeerEvents' -v"
        status: pass
      - kind: contract
        ref: "go build ./... and make lint green; OnPeerEvent is rendered only by logPeerEvent with capitalized static messages and sloglint-enforced snake_case keys"
        status: pass
    human_judgment: false
  - id: D3
    description: "Clock drift: one DriftEvent per (peer, 10 s) through the shared allowWarn slot while DriftWarnings counts every offending record; the DataSync part maximum reports too; nil OnDrift falls back to the prebuilt Log string"
    requirement: OBSV-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestDriftEvent|TestDiscard_warnLimiterKeyedByNodeAndClass' -v (incl. the clock_drift limiter row)"
        status: pass
    human_judgment: false
  - id: D4
    description: "One Starting sup-kv INFO record before wire carrying version, commit, go_version, node_id, all addresses, metrics_tls, every timing knob, log level/format, and no certificate/key/CA key"
    requirement: OBSV-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMain_startupLogRecord$' -v (version=dev, node_id=1, dump path, max_clock_lead=2m0s, metrics_tls=false, forbidden keys absent)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Exactly one load-outcome record per start: Dump load INFO with outcome/generation/records/discarded for prompt and main; the existing Backup WARN and Unusable ERROR gain the same attributes; a restart over the first run's dump reports main with records >= 1"
    requirement: OBSV-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_startupLogRecord|TestMain_startupLogLoadOutcome' -v"
        status: pass
      - kind: regression
        ref: "go test -race -count=1 ./cmd/sup-kv/ ./internal/app/persist/ (whole packages)"
        status: pass
    human_judgment: false
  - id: D6
    description: "OBSV-06 level coverage: the new catalog entries coexist with the existing dump-failure, unusable-dump, duplicate-node_id and foreign proto levels without regression; the core remains slog-free"
    requirement: OBSV-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ ./internal/grpcapi/ (whole packages)"
        status: pass
      - kind: contract
        ref: "grep -rn '\"log/slog\"' internal/app returns nothing; make lint green on the whole repo"
        status: pass
    human_judgment: false
duration: 24min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 07: Typed Lifecycle Events and One-Record-Per-Start Logging Summary

**Gave the operator the lifecycle catalog: the gate now narrates whether it opened on a listing or its backstop, the core emits typed peer appear/restart/evict and drift events rendered once in `cmd`, and every start logs one secrets-free startup record plus exactly one load-outcome record.**

## Performance

- **Duration:** 24min
- **Started:** 2026-11-23T09:53:06Z
- **Completed:** 2026-11-23T10:16:52Z
- **Tasks:** 3 (Task 1 tracer, Task 2 TDD, Task 3 auto)
- **Files modified:** 9 (3 created, 6 modified)

## Accomplishments

- `grpcapi.ServerOptions` gained nil-safe `OnGateOpen func(reason string)`. `openGate(reason)` now calls it inside the same CAS branch that stores the gate flag and flips health to SERVING, and `RunGate` passes `gateReasonSync`/`gateReasonBackstop` from its two opening paths. `TestGate_openReason` proves both reasons plus single-shot delivery over a buffered channel; `TestMain_gateOpenLogged` proves the real binary emits one INFO `Startup gate opened` with `reason=backstop` and exits 0 on SIGTERM.
- `internal/app/repl/events.go` defines `PeerEventKind` (`PeerAppeared`/`PeerRestarted`/`PeerEvicted`), `PeerEvent` and `DriftEvent`, plus the nil-safe `Config.OnPeerEvent`/`OnDrift` seams and the `emitPeer`/`emitDrift` helpers.
- `handleHello` emits `PeerAppeared` only in the `!remembered` branch after `peerGen++` and `PeerRestarted` after `pushSeenRun`, carrying the previous run captured before the push. `evictPeers` emits `PeerEvicted` with `now.Sub(lastHello)` before the delete. Repeat Hello, an alternating run id and the node's own echo emit nothing.
- The shared limiter was factored into `allowWarn(now, nodeID, class) bool`; `warnRateLimited` keeps identical behavior and `emitDrift` claims the new `clock_drift` slot. `applyRemote` and the `handleDataSync` part maximum now call `emitDrift` immediately after the per-record `DriftWarnings` increment; a nil `OnDrift` falls back to a prebuilt `cfg.Log` string.
- `cmd/sup-kv/main.go` renders the catalog once: `logGateOpen`, `logPeerEvent` (appear INFO; restart/evict WARN), `logDriftEvent` (WARN with node_id/handle/lead/drift_warn) and `logStartup` (build identity plus every effective knob, no cert/key/CA keys). Peer and drift seams are wired in `newReplicationAdapter`'s `repl.Config`.
- `logLoadStats(stats, records)` now emits exactly one record per start: an INFO `Dump load` for prompt/main with `outcome`/`generation`/`records`/`discarded`, and the pre-existing Backup WARN and Unusable ERROR with the same four attributes appended rather than a second line. `startEngine` passes `engine.KeyCount()`.
- `TestPeerEvents` and `TestDriftEvent` pin the transition matrix and limiter semantics; `TestMain_startupLogRecord` and `TestMain_startupLogLoadOutcome` pin the rendered records end-to-end, including a prompt-then-main restart over the first run's final dump.

## Task Commits

1. **Task 1: narrate the startup gate opening with its reason** - `ae90e00` (feat)
2. **Task 2 RED: pin peer and drift typed lifecycle events** - `957e3c5` (test)
3. **Task 2 GREEN: emit typed peer and drift events from the core** - `3a82bab` (feat)
4. **Task 3: log the startup configuration and one load outcome record** - `0f68776` (feat)

**Plan metadata:** `docs(07-07): complete lifecycle catalog plan` (this commit)

## TDD Gate Compliance

- RED: `957e3c5` (`test(07-07):`), validated with `gsd_run check tdd-red-evidence` → `RED_EVIDENCE_OK` (target `TestPeerEvents`, exit 1; 11 failing rows across `TestPeerEvents`, `TestDriftEvent` and the `clock_drift` limiter row; the two negative rows pass trivially as expected).
- GREEN: `3a82bab` (`feat(07-07):`) descends from the RED commit; both plan `<verify>` commands pass.
- REFACTOR: not needed as a separate commit — the GREEN diff was minimal and the only follow-up (extracting `logSocketBuffers` to keep `wire` under funlen) is behavior-preserving and folded into GREEN.

## Files Created/Modified

- `internal/app/repl/events.go` - `PeerEventKind`/`PeerEvent`/`DriftEvent`, the nil-safe Config seams' docs, `emitPeer`, `emitDrift` with the limiter and Log fallback
- `internal/app/repl/events_test.go` - `TestPeerEvents` (appear, repeat, restart, alternation, own echo, eviction) and `TestDriftEvent` (per-record counter vs per-interval event, listing-part maximum, wall-ahead, nil fallback)
- `internal/app/repl/core.go` - `warnClassClockDrift`, `OnPeerEvent`/`OnDrift` fields, `allowWarn`, the three peer emission sites, and `emitDrift` calls in both drift paths
- `internal/app/repl/warn_test.go` - the `clock_drift` limiter row proving its own slot and the per-record counter
- `internal/grpcapi/gate.go` - `OnGateOpen`, the reason constants, and `openGate(reason)` emitting inside the CAS after the health flip
- `internal/grpcapi/gate_test.go` - `TestGate_openReason` plus the single-shot reason helper
- `cmd/sup-kv/main.go` - `logStartup`, `logGateOpen`, `logPeerEvent`, `logDriftEvent`, `hexRunID`, `logSocketBuffers`, the one-record `logLoadStats` with its token helpers, and the three seam wirings
- `cmd/sup-kv/main_test.go` - `TestMain_startupLogRecord`, `TestMain_startupLogLoadOutcome`, `TestMain_gateOpenLogged`, the `logRecord.Reason` field and the map-based log parsing helpers
- `.planning/phases/07-observability-and-configuration/red-07-07-task2-events.json` - the validated RED evidence record

## Decisions Made

- **Gate reason is typed and emitted inside the CAS**: the same branch owns the flag store, the health flip and the callback, so the narration can never disagree with the writable transition or fire twice.
- **Peer events are state-transition values, not text**: the core carries Kind plus node id, handle, both run ids and the measured silence age; main owns the level and field names, which keeps the core slog-free and the tests non-brittle.
- **Drift event admitted by the shared slot, counter per record**: `emitDrift` increments nothing; the counter already moved, and `allowWarn` only gates the WARN/event.
- **Drift's peer identity logs as `handle`**: sloglint forbids `source` because slog handlers reserve it; the typed field stays `Source`, and `handle` matches the peer rows.
- **Generation "none" comes from the outcome**: the port's zero `Generation` is `GenMain`, so deriving the name from the enum alone would mislabel prompt/unusable loads.
- **Load attributes built once and shared**: the INFO/WARN/ERROR paths take the same four attributes, which is what makes "one record, no second line" a structural property.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] sloglint forbids the `source` log key**
- **Found during:** Task 2 verification (`make lint`)
- **Issue:** the catalog's drift attribute `source` is in sloglint's `forbidden-keys` list (reserved by slog's own source attribute); the build gate rejected it.
- **Fix:** `logDriftEvent` emits the peer identity as `handle` (the same attribute the peer rows use); `DriftEvent.Source` and its meaning are unchanged.
- **Files modified:** `cmd/sup-kv/main.go`
- **Verification:** `make lint` green; drift unit rows pass.
- **Commit:** `3a82bab`

**2. [Rule 3 - Blocking] `wire` exceeded funlen after adding the gate wiring**
- **Found during:** Task 2 verification (`make lint`)
- **Issue:** `wire` reached 61 lines against the 60-line funlen limit.
- **Fix:** extracted `logSocketBuffers(adapter)`, which performs the same XPORT-05 read-back and INFO record; `wire` is back under the limit.
- **Files modified:** `cmd/sup-kv/main.go`
- **Verification:** `make lint` green; socket-buffer process test still passes.
- **Commit:** `3a82bab`

**3. [Rule 3 - Blocking] goconst rejected the repeated `main` load token**
- **Found during:** Task 3 verification (`make lint`)
- **Issue:** the literal `"main"` appeared three times across the outcome, generation and candidate-prefix helpers.
- **Fix:** introduced the `loadToken*` constants and used them in all three places.
- **Files modified:** `cmd/sup-kv/main.go`
- **Verification:** `make lint` green; both load-outcome process rows pass.
- **Commit:** `0f68776`

**4. [Rule 3 - Blocking] unparam flagged the shared `datagramFromRun` test helper**
- **Found during:** Task 2 verification (`make lint`)
- **Issue:** every caller (including the three new ones) passed `peerID1`, so unparam reported the parameter as constant.
- **Fix:** the alternation subtest now drives `peerID2`, which is also a more diverse fixture; behavior under test is unchanged.
- **Files modified:** `internal/app/repl/events_test.go`
- **Verification:** `TestPeerEvents` green; `make lint` green.
- **Commit:** `3a82bab`

### Intentional plan deviations

- **TDD RED shape:** the RED commit includes the typed contract (`events.go` types and the two `Config` fields) with no emission behavior. Go's RED classifier refuses a non-compiling package (`build_failure`), so the API surface the tests name had to exist for the target tests to compile and fail on their planned assertions. The RED evidence, not the skeleton, authorizes GREEN; the emission and limiter logic landed only in the GREEN commit.
- **`source` → `handle`:** the plan's must-have names the drift field `source`; the lint gate forbids it, so the rendered key is `handle` (deviation 1 above). Nothing else in the catalog changed.

## Issues Encountered

- One full `internal/app/repl/` run failed `TestScenario_dumpNotify`'s failure-window row (`attempts by 300ms = 2, want 3 — the retry ladder is not signalling`). The row drives virtual time through the simulator and is untouched by this plan's changes (no dump-path code changed). It passed 5/5 in isolation (`-run TestScenario_dumpNotify -count=5`) and the whole package passed on the next full run, so it is recorded as a load-sensitive flake, not a regression; no bound or test was modified.
- `requirements.ready-ids` reports 0/3 ready (OBSV-01/OBSV-06/OBSV-05): the ids are shared with sibling Phase-07 plans (#2388 shared-ID gate), so no checkbox was marked and `requirements-completed` stays empty; 07-11 closes the phase-level checkboxes.
- The RED evidence record was first submitted with a `|`-joined target filter, which the classifier read as one unknown test name (`no_target_test_failure`); recording a single target (`TestPeerEvents`) with the real run output produced `RED_EVIDENCE_OK`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The catalog's five non-metric entries now exist at stable levels and fields, the core stays slog-free, and no certificate/key value can reach the log. 07-08 can extend the same catalog with the foreign proto_version branch and the level matrix; 07-10 can describe these exact records in the runbook. No blockers; 4 plans remain in Phase 07 — ready for 07-08.

## Self-Check: PASSED

- Files verified present: `internal/app/repl/events.go`, `internal/app/repl/events_test.go`, `internal/grpcapi/gate.go`, `cmd/sup-kv/main.go`, `cmd/sup-kv/main_test.go`, `red-07-07-task2-events.json`, `07-07-SUMMARY.md`.
- Commits verified as ancestors of HEAD: `ae90e00`, `957e3c5`, `3a82bab`, `0f68776`.
