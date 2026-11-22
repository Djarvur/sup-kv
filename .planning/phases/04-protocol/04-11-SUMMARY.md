---
phase: 04-protocol
plan: 11
subsystem: replication-protocol
tags: [proto-01, proto-03, proto-04, proto-07, test-adaptation, merge-stats, warn-limiter, whole-suite, test-only, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 01
    provides: "the DataSync completeness header, the per-(node,class) WARN limiter and the bad_sync_header class every residual fixture and both new pins ride"
  - phase: 04-protocol
    plan: 03
    provides: "StateStore.Observe (the per-part clock absorption) and Generation, plus the smokeStore/peerStore Observe delegation that compiles the shared fakes"
  - phase: 04-protocol
    plan: 05
    provides: "the stale-answer branch (answer-not-Ack, no merge) the MergeStats pin drives"
  - phase: 04-protocol
    plan: 08
    provides: "the first-seen targeted listing whose extra traffic the residual fixtures and bounds had to tolerate"
  - phase: 04-protocol
    plan: 09
    provides: "the peer-identity listing queue, the seen-run ring and the duplicate-node_id WARN paths"
  - phase: 04-protocol
    plan: 10
    provides: "the wave-7 simulator scenarios (N=3, joiner window, F14) whose full suite this plan runs end-to-end"
provides:
  - "TestMergeStats_listingAndStaleExcluded: a complete listing part's Observe absorption leaves Conflicts/DriftWarnings at zero while the next local write strictly beats the listed version, and a stale DataReplication answers without merging"
  - "TestDiscard_warnLimiterKeyedByNodeAndClass: two classes from one peer each warn once, a second peer's first occurrence warns inside the same interval, repeats stay silent and the pair fires again exactly at the boundary"
  - "the recorded all-green whole-module run with the red env gate closed, plus a bench smoke run over every adapted bench fixture"
  - "the adaptation record mapping every declared file and every research Test Impact Map row to kind and outcome"
affects:
  - "04-12 cites the green whole-suite record as the phase artifact's evidence base"
  - "04-13's phase gate re-runs the same commands and flips the two frozen rows through scripts/red-expectations.tsv"
  - "the phase verifier reads the adaptation record as the completeness answer for the Test Impact Map"

actuals:
  tokens: 2224
  tasks: 2
  commits: 1
  plan_head_before: e7ce4d3c2672bdd8280d78c859926a186280ae43
  plan_head_after: 4a352e84990e1e7ea835dd98f1660c206c80d97a

tech-stack:
  added: []
  patterns:
    - "A deliberately silent counter path is pinned by a counter invariant plus a state relation: listing drift must leave MergeStats at zero AND the next local write must strictly beat the listed version, so an implementation that merely skips Observe fails."
    - "A rate limiter's key is tested by counting (class, node) substrings in the captured Log rather than the total, so dropping either key component fails on a distinct assertion."
    - "A whole-suite record cites command + exit status per package group, so the phase gate can reuse it without re-deriving expectations."

key-files:
  created: []
  modified:
    - internal/app/repl/merge_stats_test.go
    - internal/app/repl/warn_test.go

key-decisions:
  - "The listing-drift pin proves absorption through the PROTO-01 version relation, not a counter: MergeStats is intentionally silent for listings (Open Question 6), so the only honest evidence that Observe ran is that the next local write strictly beats the listed max version."
  - "The WARN pin asserts per-(class,node) counts, not a total: a total cannot distinguish 'the class is in the key' from 'the peer is in the key', and the plan requires both halves proven."
  - "No residual file needed an edit: every one of the 12 declared files either already carried its adaptation from the owning semantic plan or compiles unchanged through shared fakes (smokeStore/nodeStore), so task 2's deliverable is the recorded evidence rather than a diff."
  - "Benchmarks were smoke-run at -benchtime=1x on top of the plan's required checks: `go test` only compiles a benchmark, and task 1 explicitly required bench counting helpers to tolerate the first-seen listing."

patterns-established:
  - "Residual adaptation is closed by evidence, not by churn: a file that compiles and passes unchanged is recorded as compile-only/mechanical with the command that proves it."
  - "A test-only planning wave still records its realized diff honestly (two test files, 161 added lines) instead of inflating the estimate's 50k token scale."

requirements-completed: [PROTO-01, PROTO-03, PROTO-04, PROTO-07]

coverage:
  - id: D1
    description: "Listing drift is clock-only: a complete listing part's max version reaches Observe (the next local write strictly beats it) while Conflicts and DriftWarnings stay zero; a strictly older DataReplication is answered without merging and counts nothing."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/merge_stats_test.go#TestMergeStats_listingAndStaleExcluded"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestMergeStats_'"
        status: pass
    human_judgment: false
  - id: D2
    description: "The shared WARN limiter's key is the pair (node_id, class): two classes from one peer each warn once, repeats inside the interval stay silent, a second peer's first occurrence of the same class warns, a pair fires again exactly at the interval boundary, and the discard counters stay per datagram."
    requirement: PROTO-07
    verification:
      - kind: unit
        ref: "internal/app/repl/warn_test.go#TestDiscard_warnLimiterKeyedByNodeAndClass"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestDiscard_warnLimiter'"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every remaining fixture compiles and runs against the new wire header, the pre-body decode verdict, the Observe seam and the peer-identity listing queue: the whole module builds and vets, and the full race suite is green with SUPKV_RED_TESTS unset."
    requirement: PROTO-04
    verification:
      - kind: command
        ref: "go build ./... && go vet ./..."
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./..."
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_'"
        status: pass
    human_judgment: false
  - id: D4
    description: "The adapted benchmark fixtures execute, not merely compile: every benchmark in the four adapted packages runs one iteration without failing its own convergence or counting assertions."
    requirement: PROTO-03
    verification:
      - kind: command
        ref: "go test -run '^$' -bench . -benchtime=1x ./internal/app/repl/ ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/"
        status: pass
    human_judgment: false
  - id: D5
    description: "The two frozen files are byte-identical to the plan head; the red rows flip only through scripts/red-expectations.tsv in 04-13."
    requirement: PROTO-01
    verification:
      - kind: command
        ref: "git diff --exit-code -- internal/stand/stand_red_test.go internal/grpcapi/red_gate_test.go"
        status: pass
    human_judgment: false
  - id: D6
    description: "The adaptation record covers all 12 declared files and every research Test Impact Map row with kind and outcome, so the phase report can cite completeness; the touched package passes the strict lint gate and gofmt."
    requirement: PROTO-04
    verification:
      - kind: command
        ref: "go tool golangci-lint run ./internal/app/repl/"
        status: pass
      - kind: command
        ref: "gofmt -l internal/app/repl/merge_stats_test.go internal/app/repl/warn_test.go"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 11: Residual Fixture Adaptation and the Whole-Suite Green Record Summary

**The adaptation tail closes with two new pins — listing drift stays out of MergeStats while its Observe absorption remains provable, and the WARN limiter is proven keyed by (node_id, class) across two peers and two classes — and the whole module, benchmarks included, is green with the frozen red files untouched.**

## Performance

- **Duration:** 13min
- **Started:** 2026-11-22T06:17:04Z
- **Completed:** 2026-11-22T06:31:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `TestMergeStats_listingAndStaleExcluded` closes Open Question 6 at the counter layer. A complete one-part listing stamped one drift-warn window past the wall clock is fed to the core; `MergeStats()` stays `{Conflicts: 0, DriftWarnings: 0}`, yet the very next local write strictly beats the listed max version — so the part provably reached `Observe` and only the counter was deliberately silent. The second half feeds a strictly older `DataReplication` under a held key: the answer-not-Ack branch queues exactly one reply, never calls `ApplyRemote`, counts nothing, and leaves the held record serving. A counter that counted listing drift, or an implementation that skipped `Observe`, each fails a distinct assertion.
- `TestDiscard_warnLimiterKeyedByNodeAndClass` extends the limiter coverage beyond one class. Peer 1's `bad_proto_version` and `bad_sync_header` inside one interval each WARN once (proving the class is in the key); repeats of either pair inside the interval stay silent; peer 2's first occurrence of both classes inside peer 1's interval each WARN (proving the peer is in the key); peer 1's pair fires again exactly at `discardWarnInterval`; and `BadProtoVersion`/`BadSyncHeader` count every datagram, so the limiter is shown to suppress only the log line.
- The whole module was verified end-to-end: `go build ./...` and `go vet ./...` clean, `go test -race -count=1 ./...` green across all 16 packages (repl 136.0s, stand 6.8s inside the full run, every other package in seconds or less) with `SUPKV_RED_TESTS` unset, and the stand suite re-run alone in 6.12s wall.
- All four adapted benchmark packages were smoke-run at `-benchtime=1x`: `BenchmarkHandleDataRequest`/`BenchmarkHandleDataSync`/`BenchmarkCoreEventLoop`, `BenchmarkDump10kKeys`, `BenchmarkReplicationBurst` (which fails on any lost write) and `BenchmarkConfirmationLatency` all pass, so the bench fixtures also work at runtime, not just at compile time.
- The frozen files are byte-identical: `git diff --exit-code -- internal/stand/stand_red_test.go internal/grpcapi/red_gate_test.go` is empty.
- The strict lint gate passes on the touched package (`go tool golangci-lint run ./internal/app/repl/`, exit 0) and gofmt reports no file.

## Task Commits

1. **Task 1: Residual fixture adaptation — the drift and WARN pins** - `4a352e8` (test)
2. **Task 2: Whole-suite green and the adaptation record** - this SUMMARY/STATE/ROADMAP/REQUIREMENTS commit (docs, no code change)

**Plan metadata:** this SUMMARY/STATE/ROADMAP/REQUIREMENTS commit follows this file.

## Files Created/Modified

- `internal/app/repl/merge_stats_test.go` — added `TestMergeStats_listingAndStaleExcluded` (+62 lines): the listing-absorption/stale-answer counter pin, with the `version` and `storage` imports it needs.
- `internal/app/repl/warn_test.go` — added `TestDiscard_warnLimiterKeyedByNodeAndClass` (+99 lines): the multi-class/multi-peer limiter pin and the `fmt` import.

No other file was edited. Task 2's verification ran against the tree unchanged.

## Adaptation Record

Declared-file table (this plan's `files_modified`), kind per the research's Test Impact Map vocabulary:

| File | Kind | Outcome |
|------|------|---------|
| `internal/app/repl/merge_stats_test.go` | semantic (new pin) | Edited: `TestMergeStats_listingAndStaleExcluded` added and green. |
| `internal/app/repl/warn_test.go` | semantic (new pin) | Edited: `TestDiscard_warnLimiterKeyedByNodeAndClass` added and green. |
| `internal/app/persist/bench_test.go` | mechanical | Unchanged — no `DataSync`/`StateStore` fixture in the file; `BenchmarkDump10kKeys` smoke-ran green. |
| `internal/app/storage/view_test.go` | mechanical | Unchanged — the version/lookup/versions/hammer tests ride `Store` directly and are green. |
| `internal/app/internal/simnet/simnet_test.go` | mechanical | Unchanged — `nodeStore.Observe` already delegates in `simnet.go` (04-03); tie-break/determinism/faults tests green. |
| `internal/udprepl/bench_test.go` | mechanical | Unchanged — the benchmark measures convergence only (no core queue counting); smoke-ran green. |
| `internal/udprepl/scenario_test.go` | compile-only | Unchanged — constructs its nodes through `smokeStore`, which gained `Observe` in 04-03; `TestScenario_convergence` and `TestScenario_discardClasses` (including the pre-body `bad_proto_version` row) green. |
| `internal/udprepl/readloop_internal_test.go` | mechanical | Unchanged — the read-loop error-survival tests are green. |
| `internal/udprepl/socket_internal_test.go` | mechanical | Unchanged — the socket-buffer/boundary/concurrency tests are green. |
| `internal/grpcapi/bench_test.go` | mechanical | Unchanged — `BenchmarkConfirmationLatency` smoke-ran green. |
| `internal/stand/stand_test.go` | mechanical | Unchanged this plan — the derived empty-gate rows were adapted in 04-04; all eight `TestStand_*` scenarios green in 6.12s wall. |
| `internal/stand/stand_metrics_test.go` | mechanical | Unchanged this plan (04-04); `TestStand_metricsByEffect` green. |

Research Test Impact Map rows owned by earlier plans (cited for completeness):

| Row | Kind | Owned by |
|-----|------|----------|
| `internal/app/repl/sync_test.go` | semantic + mechanical | 04-01 |
| `internal/app/repl/restart_test.go` | semantic | 04-08, 04-09 |
| `internal/app/repl/discard_test.go` | mechanical + new | 04-01, 04-05, 04-08, 04-09 |
| `internal/app/repl/retransmit_test.go` | semantic + mechanical | 04-05, 04-08, 04-09 |
| `internal/app/repl/peertable_test.go` | minor | 04-09 |
| `internal/app/repl/bench_test.go` | mechanical | 04-01 (header fixture) |
| `internal/app/repl/sim_*.go` | semantic | 04-10 (plus 04-08) |
| `internal/app/repl/egress_internal_test.go` | mechanical | 04-05 |
| `internal/udprepl/repl_test.go` | semantic | 04-01, 04-03 |
| `internal/udprepl/smoke_test.go` | mechanical | 04-03 (`Observe` delegation) |
| `internal/udprepl/syncsize_internal_test.go` | mechanical | 04-01, 04-03 |
| `internal/udprepl/wait_integration_test.go` | mechanical | compiled unchanged (no `StateStore` fake in the file) |
| `internal/grpcapi/gate_test.go` | semantic | 04-02 |
| `internal/grpcapi/server_test.go` | mechanical | 04-02, 04-06 |
| `internal/grpcapi/red_gate_test.go` | none (frozen) | untouched — `git diff` empty |

## Decisions Made

- **Absorption is proven by the version relation, not a counter.** `MergeStats` deliberately ignores listing drift (OQ6), so a counter assertion alone could pass on an implementation that never called `Observe`. The test therefore also asserts the next local write strictly beats the listed max version — the PROTO-01 precision contract — making both halves load-bearing.
- **Per-pair WARN counts, not totals.** The test counts `class <c> from node <n>` occurrences in the captured Log. A total-only assertion would pass even if either key component were missing; the four distinct pair assertions cannot.
- **The stale half reuses the existing stale branch.** It drives `handleDataReplication` through `replFrom`, which the 04-05 tests already cover semantically; here the point is the counter invariant plus the queued-reply delta, so the new test stays inside `merge_stats_test.go` rather than duplicating `TestReplication_staleAnsweredNotAcked`.
- **Task 2 needed no edit.** Every declared file was already adapted or compiles unchanged; the honest deliverable is the recorded green run and the table above, not a manufactured diff.
- **Benches were executed, not just compiled.** `go test` compiles benchmarks without running them, and task 1 required bench helpers to tolerate first-seen listing traffic, so a one-iteration run was added as extra evidence and recorded as its own coverage row.

## Deviations from Plan

None — plan executed exactly as written. Task 2's action step 1 ("for each failure, decide…") found no failures; step 2 ("if the derived empty gate changed a timing expectation, re-derive") found no change (the stand suite is green unchanged); step 3's adaptation table is the section above.

## Issues Encountered

- **Both new pins were GREEN on first run.** The listing `Observe` path (04-03), the stale-answer branch (04-05) and the per-`(node_id, class)` WARN limiter (04-01) already existed; this plan's task is test evidence, so there is no implementation to write and no legitimate RED to record. Following the 04-07/04-10 precedent, no RED record was fabricated and the outcome is disclosed here rather than in a TDD-gate section (the plan's tasks carry no `tdd="true"`).
- **No fixture failures were found anywhere.** The full suite and the bench smoke run were green on the first attempt, so task 2 produced no edits; the recorded run is the result.

## Known Stubs

None. This plan is test-only; no placeholder values, skipped tests or unwired data sources were introduced.

## Threat Flags

None — no new trust-boundary surface. T-4-33 is mitigated as planned: both changed assertions are recorded above with their mechanisms, no assertion was deleted or weakened, and no test gained a skip. T-4-34 is mitigated: the frozen-file `git diff --exit-code` gate passed and neither frozen file appears in this plan's only commit. T-4-SC: no dependency was added or installed.

## Next Phase Readiness

- The whole-suite green record and the adaptation table are ready for 04-12 (the phase artifact) and 04-13 (the phase gate and the two TSV flips).
- Two plans remain in the phase; 04-12 is next.
