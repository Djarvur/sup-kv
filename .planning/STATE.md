---
gsd_state_version: "1.0"
milestone: v1.1
milestone_name: Spec Compliance and Resilience
current_phase: 2
current_phase_name: Stand and Simulator as Tests
status: executing
stopped_at: Completed 02-04-PLAN.md
last_updated: "2026-11-19T20:46:43.629Z"
last_activity: 2026-11-19
last_activity_desc: Phase 2 execution started
state_head: 26e935ee1e9b8eed36ec64bce20c34f22c03eb6f
progress:
  total_phases: 9
  completed_phases: 1
  total_plans: 21
  completed_plans: 8
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-11-19)

**Core value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.
**Current focus:** Phase 2 — Stand and Simulator as Tests

## Current Position

Phase: 2 (Stand and Simulator as Tests) — EXECUTING
Plan: 5 of 17
Status: Ready to execute
Last activity: 2026-11-19 — Phase 2 execution started

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**
- v1.0 (archived): 38 plans, 7 phases — avg ~10 min/plan; full per-plan history in [milestones/v1.0-ROADMAP.md](milestones/v1.0-ROADMAP.md)
- v1.1: no plans executed yet

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| v1.1-1 | 0 | - | - |
| 1 | 4 | - | - |

*Updated after each plan completion*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P01 | 349s | 3 tasks | 51 files |
| Phase 01 P02 | 787s | 3 tasks | 1 files |
| Phase 01 P03 | 524s | 3 tasks | 11 files |
| Phase 01 P04 | 299 | 2 tasks | 0 files |
| Phase 02 P01 | 520s | 3 tasks | 8 files |
| Phase 02 P02 | 2040s | 2 tasks | 4 files |
| Phase 02 P03 | 441 | 2 tasks | 1 files |
| Phase 02 P04 | 1191s | 2 tasks | 5 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: v1.1 phase structure follows v1.0 review section 15, maintainer-confirmed 2026-11-19 — red tests (Phase 2) precede fixes (Phases 3-5); no deviation from the confirmed composition
- Roadmap: phase numbers reset to 1 for v1.1 (`--reset-phase-numbers`); v1.0 phases 1-7 are archived history under milestones/v1.0-phases/
- Roadmap: owner decisions 14.1-14.11 are settled (2026-11-19) and are not re-litigated; 14.5(6) and 14.7 accepted by default and recorded in SPEC-02
- Roadmap: Phase 2's Critical-scenario tests (R-01, R-02, R-03, P-01) are expected RED against v1.0 code by design — planning must choose the gating mechanism so red tests can merge without breaking CI (they turn green in Phases 3-5)
- [Phase 01]: Publication-guard verify gates scoped to the non-public prefix docs/reviews/2026-11-19-v1.0/internal/; evidence/repro/**/internal/** repro paths are legitimate D-01 payload (Rule 1 fix during 01-01 Task 1)
- [Phase 01]: Registry Links cells expand the trace's shorthand code refs to full repo paths per the trace's own file legend, so public readers resolve links without the local-only legend
- [Phase 01]: S-PROTO-08 closed as implemented with recorded-intent nuance in the summary cell (04-CONTEXT D-02); accepted-deviation(14.x) would have been an invented reference
- [Phase 01]: S-HLC-07 accepted-deviation references recorded v1.0 research assumption 02-RESEARCH A2 — no ADR/14.x covers the zero filler; amend-check confirmed the amendment left the random-filler requirement standing
- [Phase 01]: Unnamed rows resolved by the priority rule with REQ IDs verified in REQUIREMENTS.md: S-PROTO-02/03 to XPORT-03, S-PAR-05 to PERSIST-01, S-ALG-RC-09 to XPORT-07, S-PAR-16 to OBSV-05, S-FAIL-17 to OBSV-06, S-FAIL-03 to PROTO-03, S-HLC-10 to PROTO-01, S-IMP-08 accepted-deviation(ADR 0008)
- [Phase 01]: ADR set authored slim-MADR with rule-level detail for 14.5 (six numbered sub-rules) so Phase 4 can address accepted-by-default rule (6) individually; honest gaps kept (0002 informs [], 0011 loss-share has no dedicated v1.1 REQ) instead of invented mappings
- [Phase 01]: Phase gate closed with zero task commits: every gate passed on the committed tree as plans 02/03 left it; fixups were gated on gate failure and none failed — Plan 01-04's <output> creates no new files and commits fixups only when a gate fails; committing a no-op would fabricate history - the green gate record lives in 01-04-SUMMARY.md
- [Phase 01]: Scoped-prefix publication guard (docs/reviews/2026-11-19-v1.0/internal/) is the standing phase-gate assertion, never any-path-containing-internal substring matching — 22 legitimate evidence/repro/**/internal/** repro-test module paths are committed D-01 payload; literal substring matching contradicts the 46-tracked-evidence-files acceptance (wave-1 Rule 1 scoping, re-affirmed at the gate)
- [Phase 02]: R-03 red test's SyncArrived fake bakes v1.0 mirror semantics (any-chunk->true) with closed-side assertions parked behind a release channel (TestGate_opensOnTimeout pattern) — the Phase 4 flip needs zero test edits, only the expectation-table row
- [Phase 02]: red-runner errors on unexpected failing tests (names not in the expectation table; subtests count as their parent's) — T-2-01 'unknown names are errors' applied to the event stream
- [Phase 02]: red-runner package list hardcoded with an extend comment (./internal/grpcapi/ first) — the TSV carries no package column
- [Phase 02]: internal/stand harness + scenarios landed as one commit (91765fd): the strict lint gate's transitive unused analysis cannot reach the t-coupled harness helpers without their callers — stand.go alone is un-committable; the plan's Task 1/2 file split survives in content, only the commit boundary moved
- [Phase 02]: internal/stand keeps the plan's file names AND internal tests via justified //nolint:testpackage (stand_test.go package clause) and //nolint:depguard (testing import in stand.go) — both with written reasons per AGENTS.md; the package doc states the test-infrastructure status the path-based rules cannot see; sibling plans 02-05/02-14 follow the same pattern
- [Phase 02]: The cmd/sup-kv port registry was NOT dead after the scenario move — broadcast_probe_test.go (Phase 7 DOCS-01 probe) still uses scenarioUDPPort, so it moved to its remaining consumer and main_test.go needed zero removals
- [Phase 02]: F2c pin rides standAbsent (NotFound-based absence) + standSetWith (review's 1-byte value): a value-mismatched record must fail the never-replicated window, and the derived 70KB scenario value would trip the store's value limit before the datagram ceiling matters
- [Phase 02]: Cluster.Delete returns (storage.Record, error): the plan's must_haves ErrNotFound propagation beats the prose's bare-Record signature
- [Phase 02]: ReplaceNodeWithState seeds via store.Merge in sorted-key order: map iteration would randomize the HLC watermark the merges leave behind and break same-seed replay; the VERS-04 property rides the merge path, no Watermark pre-seed
- [Phase 02]: simnet size gate refuses at schedule time (before booking — PendingDeliveries stays truthful); receive budget consults at fire time against the live map, keyed by handle so it survives ReplaceNode
- [Phase 02]: AnnounceDelay rides port.DataReplication only (write broadcasts + sync answers); over-limit seed records panic per the checkIndex/Reregister harness convention
- [Phase 02]: fsops seam: failure hygiene is impl-side in shared production helpers (closeRemoveTemp/removeTemp) — natural failure paths and the injection fake run the same hygiene code, so the failure matrix observes real dump hygiene; saveAtomically is a pure MkdirAll + five-call sequence, behavior bit-identical (existing suites green)
- [Phase 02]: fsops fakes ride the embedded real fileFSOps: the order assertion (write→fsync→rotate→rename→dir-fsync) is taken over the production Save path's actual operations and the rotation chain over real files; the failing fake short-circuits before the real op and performs the step's own hygiene via the production helpers
- [Phase 02]: fsops internal-test placement: justified //nolint:testpackage keeps the plan's artifact name fsops_test.go with package persistfile (the internal/stand precedent) so the fakes reach the unexported FileStore.ops injection point; exported constructors unchanged, nothing added to internal/port
- [Phase 02]: D-07 blocking proof rides a synctest bubble with an in-memory blockingDumpPort (startDumper pattern): deterministic not-returned window, no real-time flake surface for the nightly -race -count=20 soak

### Pending Todos

None yet.

### Blockers/Concerns

- Carried from v1.0: server-side branch-protection ruleset still deferred (org-plan 403); re-apply path recorded in docs/setup/branch-protection.md — revisit at the v1.0.1+/ruleset window noted in PROJECT.md
- Carried from v1.0: macOS quickstart is option-b (Linux CI-verified + marked darwin loud-skip); v1.1 DOCS-09 must keep the pair working and DOCS-10 demotes darwin to development-only (14.10)

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| Feature | AUTH-01 HMAC datagram auth; GARB-01 tombstone GC; PFXD-01 per-prefix delete-wins; PEER-01 explicit peer lists; SNAP-01 consumer proto mapping | Tracked in REQUIREMENTS.md v2 section | 2026-11-19 | v1.1 |

## Session Continuity

Last session: 2026-11-19T20:46:30.400Z
Stopped at: Completed 02-04-PLAN.md
Resume file: None
Next action: `/gsd:plan-phase 1`
