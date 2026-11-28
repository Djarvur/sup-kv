---
phase: 07-observability-and-configuration
plan: 10
subsystem: observability
tags: [operations-runbook, lifecycle-catalog, exit-codes, clock-drift, metrics-definitions, spec-registry, architecture-note]
requires:
  - phase: 07-observability-and-configuration
    provides: plans 07-07/07-08's lifecycle records and level matrix, and plans 07-05/07-09's honest drift/sent/conflict definitions
  - phase: 05-persistence
    provides: the one-record-per-class load report the runbook now documents
provides:
  - docs/OPERATIONS.md as the as-built operator contract — lifecycle catalog, 0/1/2/3 exit table, signal ordering, five env-only knobs
  - the reachable drift condition and the 2 m lead default with the drift_warn < max_clock_lead startup rule
  - the four routed core counters, the unbound_listing label and the honest metric definitions in the monitoring list
  - five SPEC-DEVIATIONS rows reflecting the shipped code paths
  - AGENTS.md naming internal/config and the three additive depguard rules plus the written grpcapi exception
affects: [07-11, 09-registry-audit]
actuals:
  tokens: 8096
  tasks: 2
  commits: 2
plan_head_before: 14783d1b2c4e3945b1b43354360e8c3b7e534db1
plan_head_after: 032959b46e56d05a0bef86e03ed95ab02657095b
tech-stack:
  added: []
  patterns:
    - "Doc guard: one anchor literal per mandated statement plus a whitespace-normalized forbidden-phrase table for claims the phase corrected"
    - "Registry flip: the status becomes implemented only when the row's links name the shipped code path and a named test"
key-files:
  created: []
  modified:
    - docs/OPERATIONS.md
    - cmd/sup-kv/operations_test.go
    - docs/SPEC-DEVIATIONS.md
    - AGENTS.md
key-decisions:
  - "The drift runbook states only the shipped direction — a received record or DataSync part maximum leading local wall time — and the anchor 'received version leads' keeps every other direction out"
  - "operationsForbidden compares whitespace-normalized text on both sides, because the stale load-outcome sentence is line-wrapped and a raw substring search would have been vacuous"
  - "The monitoring list is grouped by concern (reach/backlog, traffic, routed counters, discards, conflicts/drift, dumps, process) so each definition sits beside its family; TestOperationsDocFamilies still cross-checks every name against the exporter"
  - "S-SEM-07 keeps superseded-by-amendment — decision 14.4 removed the journal — and only the counter's meaning changes to pending-local displacement"
  - "S-IMP-08 keeps accepted-deviation(ADR 0008) because the endpoint stays opt-in, while its evidence now records that the conflict and drift counters are honest"
  - "AGENTS deviation 1 names the three additive rules and the grpcapi composition exception; the Go-version policy, the vendored-config provenance and the nolint rule are byte-identical"
patterns-established:
  - "Pattern: doc claims that a phase corrects go into a normalized forbidden table, so the old sentence cannot survive a reflow"
requirements-completed: [OBSV-01, OBSV-02, OBSV-03, OBSV-04, OBSV-05, OBSV-06, OBSV-07]
coverage:
  - id: D1
    description: "OPERATIONS carries one bullet per lifecycle-catalog entry with its level and stable fields, anchored message by message"
    requirement: OBSV-01
    verification:
      - kind: unit
        ref: "cmd/sup-kv/operations_test.go#TestOperationsDoc (## Lifecycle logs, Starting sup-kv, Dump load, Startup gate opened, Peer restarted, Clock drift)"
        status: pass
      - kind: unit
        ref: "cmd/sup-kv/catalog_internal_test.go#TestCatalogLevels (the level truth each bullet restates)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The exit table is 0 clean / 1 startup-or-Serve / 2 usage / 3 final dump, with the signal-before-load ordering and the five env-only knobs"
    requirement: OBSV-04
    verification:
      - kind: unit
        ref: "cmd/sup-kv/run_internal_test.go#TestRun_signalInstalledBeforeWire, #TestRun_startupFailureExitsOneNoDump, #TestRun_serveErrorStillShutsDown, #TestRun_dumpFailureCode, #TestRun_serveAndDumpFailure"
        status: pass
      - kind: contract
        ref: "grep 'exit code 3' docs/OPERATIONS.md and TestOperationsDoc's exit-code anchors (0/1/2/3)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The clock-drift runbook describes only the shipped received-lead condition, the one-WARN-per-peer rate limit and the 2 m lead default with the drift_warn < max_clock_lead rule"
    requirement: OBSV-05
    verification:
      - kind: unit
        ref: "internal/app/clock/clock_test.go#TestClock_driftSignal; internal/app/repl/merge_stats_test.go#TestMergeStats_equalVersionTieBreakCountsDrift; internal/app/repl/events_test.go#TestDriftEvent"
        status: pass
      - kind: contract
        ref: "grep 'received version leads' and 'drift_warn < max_clock_lead' docs/OPERATIONS.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "The monitoring list names the four routed counters, the unbound_listing label and the conflict/sent/duration/drift/RSS definitions, and every named family exists in the exporter"
    requirement: OBSV-03
    verification:
      - kind: unit
        ref: "cmd/sup-kv/operations_test.go#TestOperationsDocFamilies (every supkv_ name in the doc must appear in internal/metrics/metrics.go)"
        status: pass
      - kind: unit
        ref: "internal/metrics/metrics_definitions_test.go#TestMetrics_definitions (the Help phrases the doc restates)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Five registry rows reflect the shipped code: S-HLC-13, S-PAR-16 and S-FAIL-17 implemented; S-SEM-07 and S-IMP-08 notes updated; S-HLC-07 and the v2 rows untouched"
    requirement: OBSV-05
    verification:
      - kind: contract
        ref: "grep -n 'S-HLC-13|S-PAR-16|S-FAIL-17' docs/SPEC-DEVIATIONS.md -> each row reads implemented; grep 'pending-local displacement' -> present; git diff shows exactly five rows changed"
        status: pass
      - kind: contract
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestOperationsDoc' -> ok (no doc regression)"
        status: pass
    human_judgment: true
    rationale: "the row-to-code correspondence is the T-07-31 review check — the greps prove the status token, not that each citation names the shipped path"
  - id: D6
    description: "AGENTS.md names internal/config in the Layout and the three additive depguard rules plus the written grpcapi exception, with the rest of the file unchanged"
    requirement: OBSV-07
    verification:
      - kind: contract
        ref: "grep -c 'sup-kv-adapters-not-in-core' AGENTS.md >= 1; grep 'internal/config/' AGENTS.md matches; git diff HEAD~2..HEAD -- AGENTS.md touches only Project deviations item 1 and Layout"
        status: pass
    human_judgment: false
duration: 5min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 10: Operator Documentation and Registry Close-Out Summary

OPERATIONS.md now documents the binary as shipped — the lifecycle catalog with a level per entry, the 0/1/2/3 exit table, the reachable received-lead drift condition, the four routed counters and the honest metric definitions — with anchor tests and a forbidden-phrase table holding the corrected sentences out, while five SPEC-DEVIATIONS rows and the AGENTS architecture note were brought up to the as-built code.

## Performance

- **Duration:** 5min
- **Started:** 2026-11-23T11:26:40Z
- **Completed:** 2026-11-23T11:31:21Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- `## Lifecycle logs` lists every catalog entry with its level and stable fields, cross-anchored to `TestCatalogLevels`; the exit table now distinguishes 0 clean, 1 startup-or-Serve, 2 usage and 3 final dump, states that a Serve error still runs the bounded final dump, that signals are armed before load and bind, that a second signal takes the hurry-up path without skipping the flush, and that `--help` names the five env-only knobs.
- The clock-drift runbook describes only the condition the binary can produce: a received record or DataSync part maximum whose version leads local wall time by more than `--drift-warn`, counted per record and WARNed once per peer per interval. The `--max-clock-lead` default is 2m and the startup relation `drift_warn < max_clock_lead` is stated with the one-point validation catalog.
- The monitoring list carries the four routed Phase-4 counters, the `unbound_listing` discard label and one-line definitions for conflicts (pending displacement), sent (socket hand-off, not delivery), duration (last dump attempt), drift (received lead) and `process_resident_memory_bytes` (current RSS on Linux, peak on development darwin builds).
- Two stale claims were removed rather than edited around: the Main load's "nothing is logged" sentence and the failed-final-dump exit-1 parenthetical. `operationsForbidden` now fails the test if either returns in any line wrapping.
- S-HLC-13, S-PAR-16 and S-FAIL-17 read `implemented` with links to the shipped clock/core/config paths, S-SEM-07's counter is described as pending-local displacement and S-IMP-08 records honest counters under the unchanged opt-in decision.
- AGENTS.md names `internal/config/` in the Layout (composition-level, outside the app no-I/O rule) and its deviation 1 lists the three additive depguard rules plus the written `grpcapi` composition exception.

## Task Commits

1. **Task 1: OPERATIONS lifecycle, exit and drift contract with anchors** — `3c8d3f4` (docs)
2. **Task 2: Registry rows and the AGENTS architecture note** — `032959b` (docs)

**Plan metadata:** committed as `docs(07-10): complete operator documentation plan`.

## Files Created/Modified

- `docs/OPERATIONS.md` — lifecycle catalog, configuration-validation catalog, 0/1/2/3 exit table, rewritten drift runbook and monitoring list.
- `cmd/sup-kv/operations_test.go` — new anchors, the exit-1 description corrected, `exit code 3` and the Phase-7 literals added, `operationsForbidden` normalized check.
- `docs/SPEC-DEVIATIONS.md` — five rows updated (S-HLC-13, S-SEM-07, S-FAIL-17, S-PAR-16, S-IMP-08).
- `AGENTS.md` — Layout gains `internal/config/`; Project deviations item 1 names the three additive rules and the grpcapi exception.

## Decisions Made

- The drift runbook states one direction only (received version ahead of local time) and the anchor `received version leads` holds it there; a node whose clock is ahead does not warn itself, which the text now says explicitly.
- `operationsForbidden` normalizes whitespace on both sides because the removed load-outcome sentence was line-wrapped; a raw substring check over the authored text would have passed vacuously.
- The monitoring list is grouped by concern instead of one dense paragraph, so each definition sits beside its family; `TestOperationsDocFamilies` still validates every named family against the exporter.
- Registry statuses flip only with named code paths and tests; S-SEM-07 stays `superseded-by-amendment` (the journal removal is still the decision) and S-IMP-08 stays `accepted-deviation(ADR 0008)` (the endpoint is still opt-in).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 07-11 (the phase gate and OBSV checkbox close-out) is ready: the docs now match the shipped surface, all four changed files are committed, and `TestOperationsDoc`, `TestOperationsDocFamilies`, the full `./cmd/sup-kv/` suite and the plan's greps are green. Phase 9 still owns the full registry audit and the CONFIGURATION/DEPLOYMENT reference docs.

## Self-Check: PASSED

- `docs/OPERATIONS.md`, `cmd/sup-kv/operations_test.go`, `docs/SPEC-DEVIATIONS.md`, `AGENTS.md` exist and are committed at `032959b`.
- Commits `3c8d3f4` and `032959b` are ancestors of HEAD.
