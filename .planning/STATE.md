---
gsd_state_version: "1.0"
milestone: v1.1
milestone_name: Spec Compliance and Resilience
current_phase: 2
current_phase_name: Stand and Simulator as Tests
status: executing
stopped_at: Completed 02-12-PLAN.md
last_updated: "2026-11-21T12:45:40.558Z"
last_activity: 2026-11-19
last_activity_desc: Phase 2 execution started
state_head: bf2b6b4d92564e84821fe7170f0fdc3f5b6dd986
progress:
  total_phases: 9
  completed_phases: 1
  total_plans: 21
  completed_plans: 18
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-11-19)

**Core value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.
**Current focus:** Phase 2 — Stand and Simulator as Tests

## Current Position

Phase: 2 (Stand and Simulator as Tests) — EXECUTING
Plan: 15 of 17
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
| Phase 02 P14 | 178s | 1 tasks | 1 files |
| Phase 02 P05 | 1044 | 2 tasks | 1 files |
| Phase 02 P06 | 574 | 2 tasks | 2 files |
| Phase 02 P07 | 350 | 2 tasks | 2 files |
| Phase 02 P08 | 983s | 2 tasks | 2 files |
| Phase 02 P10 | 269s | 2 tasks | 7 files |
| Phase 02 P15 | 10min | 1 tasks | 1 files |
| Phase 02 P16 | 169s | 1 tasks | 1 files |
| Phase 02 P09 | 900 | 3 tasks | 10 files |
| Phase 02 P12 | 720s | 2 tasks | 8 files |

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
- [Phase 02]: internal/stand nondarwin half: the darwin-untested leak assertion is a real runtime check — go/build MatchFile against an explicit darwin context over every sibling .go file (constraint-form-agnostic), so a leaked scenario file fails the darwin cell with a named error; the loud statement is a wrapped const keeping the plan's grep gate at exactly one phrase line
- [Phase 02]: R-03's stand closed window is shaped structurally (A seeded+stopped, replaced node alone with a 30 s backstop, then A restarts) rather than raced against a live multi-chunk listing; v1.0's Get answers NotFound, never Unavailable
- [Phase 02]: All four stand red tests assert the documented acceptance expressions (R-01 2×sync_interval + N/rate; R-02 N/rate + 10 s and ≤3N; burst ≥990/s and 5 s drain; R-03 five-point PROTO-01), so Phases 3-5 flip only the expectation-table rows
- [Phase 02]: Chaos corpus format is bare decimal seed lines (no header): head -1 of testdata/chaos-seeds.txt stays a valid SUPKV_SIM_SEED replay argument for the plan verify and CI scripts
- [Phase 02]: Deletion-heavy chaos scenarios use a deterministic closing drain sweep (each node's sorted live keys, no rng or time advance) so the corpus guarantees empty/tombstone-only final states without weakening model equality, the 10x hard cap, or the fault teeth
- [Phase 02]: Chaos deletes target the acting node's sorted live keys via a pre-drawn pick index: 'against live keys' holds at execution time while every random input stays fixed before any event runs
- [Phase 02]: Negative-table counter exactness rides an asymmetric adapter whose broadcast lands in a test-held sink socket, so own-echo can never contaminate the per-class DiscardStats deltas
- [Phase 02]: Wrong-proto_version is Decoded at decode() and BadProtoVersion at the core: the negative table asserts the codec verdict and the counter separately, pinning D-02's split
- [Phase 02]: Empty-key, tombstone-with-data and oversize rows pin CURRENT behavior with explicit XPORT-10/XPORT-03 forward references (processed evidence = the apply Ack where no counter exists) and flip in Phase 3
- [Phase 02]: DELETE-twice is codes.NotFound (the store no-op, no version burned) plus exactly one dump record line; node-down-during-delete bound derives from explicitly pinned flags, never production defaults
- [Phase 02]: Watch Send-error branch pinned twice (02-08): fake server streams assert the handler returns the transport error unchanged from both Send sites (snapshot loop and relayEvent) while a real bufconn transport pins the cleanup/isolation contract — a real client cancel races the handler's ctx.Done branch, so a fake is the only deterministic Send failure
- [Phase 02]: Conflict exactness rides ordered writes (02-08): B serves A's version before writing so its HLC has absorbed it and its write strictly dominates; both nodes run --sync-interval=10s so no stale proactive listing adds a rejected-older-version conflict on A
- [Phase 02]: sync-response-rate effect needs a restarted seeded node (02-08): handleHello's discovery flush broadcasts the sender's pending records immediately and bypasses the egress limiter; a dump-loaded restart leaves pending empty, so the empty node's paced DataRequests are the only recovery path
- [Phase 02]: Retransmit asserted as a named lower bound (02-08): >=1 plus a pre-kill baseline climb because cycle count depends on scheduler-dependent Ack timing on real sockets; conflict and discard are constructed single events and assert ==1
- [Phase 02]: Per-parameter effects are named subtests (02-08): one per stand-observable parameter (startup-gate, hello-interval, sync-response-rate, value-max-bytes, peer-timeout), each with its own topology/surface; remaining flags map to existing unit/exec tests in the SUMMARY parameter table
- [Phase 02]: Shared benchmark percentile reporter lives in the module-wide internal/testutil package: Go's internal rule blocks internal/udprepl and internal/grpcapi from internal/app/internal/testutil, and all three benchmarks share one implementation; the plan's app-internal path is a documented wrapper
- [Phase 02]: Benchmarks report p50-ms/p99-ms from their own measured per-op spans with no self-pacing; the -benchtime=1x smoke reports p50==p99==the single sample
- [Phase 02]: make race-soak is the single TEST-11 command definition (-race -count=20 over the affected set, COUNT override); T-11 dispositions: port window intact (mitigated), pre-arm SIGTERM carried per D-09 (TestShutdown_secondSignal remains the pin), README deadline fixed test-side per doc block
- [Phase 02]: Extension tests live in the external simnet_test package: the internal form would need a //nolint:testpackage suppression (skip-regexp covers only *_internal_test.go) while the external form imports replproto under the depguard $test exemption and reuses the existing simnet_test.go helpers
- [Phase 02]: The determinism replay gate asserts extension activation (OversizeRefused > 0, RecvBudgetRefused > 0, non-empty stores) before comparing same-seed digests, so a zero-policy replay cannot pass vacuously
- [Phase 02]: 02-16: the TEST-07 proof lives in package persistfile_test (external) composing only exported API — the internal package would need an unjustified //nolint:testpackage, and the external form reuses dumpfile_test.go's writeFile helper (same reasoning as 02-15's simnet_test)
- [Phase 02]: TEST-13 coverage meta-test lives in internal/app/repl: an 11-row index (decision, ADR file, anchor test) with word-bounded citation matching and self-exclusion; a decision counts when a test file carries ADR 000N or decision 14.N, so 14.10 cannot satisfy 14.1
- [Phase 02]: The printed property replay hint is package-scoped: testutil resolves the caller's import path from the internal/app anchor via runtime.Caller (os-free, depguard-safe) and ReplayCommand shares the one format helper; a test executes the printed string and asserts exit 0 plus the same test's re-run
- [Phase 02]: TEST-15 small-space records derive every field from the version so equal versions are byte-identical — the premise the equal-version no-op law needs; a separate subtest pins that an incoming equal-version record with different bytes is ignored
- [Phase 02]: P-01 engine test lives in package persistfile_test (external) not the plan's literal internal package: it composes only exported API and reuses open_seed_test.go's seedDumper/seedFixedNow helpers, while the internal form would need an unjustified //nolint:testpackage (02-15/02-16 precedent)
- [Phase 02]: scripts/red-runner serializes packages with -p 1: the stand reds assert wall-clock pacing and throughput bounds, so sibling packages' race builds and load would corrupt the verdict's measurement
- [Phase 02]: The P-01 process test asserts the dump still holds the record between Set (which confirms the covering dump, PERS-04) and SIGKILL/restart, so a red verdict names the loader's 61 440-byte decode bound rather than leaving the writer or the kill suspect

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

Last session: 2026-11-21T12:45:40.394Z
Stopped at: Completed 02-12-PLAN.md
Resume file: None
Next action: `/gsd:plan-phase 1`
