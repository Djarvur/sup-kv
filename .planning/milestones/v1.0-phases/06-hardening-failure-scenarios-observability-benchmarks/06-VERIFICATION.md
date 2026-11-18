---
phase: 06-hardening-failure-scenarios-observability-benchmarks
verified: 2026-11-18T11:22:28Z
status: passed
score: 16/16 must-haves verified
covered_files: [".github/workflows/ci.yml", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-01-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-01-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-02-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-02-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-03-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-03-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-04-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-04-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-05-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-05-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-06-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-06-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-07-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-07-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-08-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-08-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-09-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-09-SUMMARY.md", "Makefile", "cmd/sup-kv/main.go", "cmd/sup-kv/main_test.go", "cmd/sup-kv/scenario_test.go", "internal/app/internal/simnet/simnet.go", "internal/app/persist/bench_test.go", "internal/app/persist/engine.go", "internal/app/persist/engine_internal_test.go", "internal/app/persist/engine_test.go", "internal/app/repl/core.go", "internal/app/repl/discard_test.go", "internal/app/repl/merge_stats_test.go", "internal/app/repl/restart_test.go", "internal/app/repl/retransmit_test.go", "internal/app/repl/sim_bound_test.go", "internal/app/repl/sim_scenario_test.go", "internal/app/repl/sync_test.go", "internal/app/storage/store.go", "internal/app/storage/store_test.go", "internal/grpcapi/bench_test.go", "internal/grpcapi/server_test.go", "internal/grpcapi/wait_integration_test.go", "internal/metrics/metrics.go", "internal/metrics/metrics_test.go", "internal/persistfile/dumpfile.go", "internal/persistfile/fsync_darwin.go", "internal/persistfile/fsync_other.go", "internal/udprepl/bench_test.go", "internal/udprepl/repl.go", "internal/udprepl/repl_internal_test.go", "internal/udprepl/repl_test.go", "internal/udprepl/scenario_test.go", "internal/udprepl/smoke_test.go"]
covered_digest: "v4:sha256:bf1347188636a3f40e29115706f88134517afdee84b89143bd9aa67bf00814f6"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 15/16
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks Verification Report

**Phase Goal:** Every documented failure scenario is an automated passing test and the node is observable enough to operate — the evidence that the core value holds under chaos
**Verified:** 2026-11-18T11:22:28Z
**Status:** passed
**Re-verification:** Yes — staleness refresh after phase 7 (v1.0.0 at HEAD) changed three covered files

## Re-Verification Context

The prior report (2026-11-16T11:48:33Z, status passed, 15/16 with one abstained backstop truth) went stale because phase 7 changed covered inputs: `Makefile` (additive `release`/`release-snapshot` targets only — the `bench` target untouched), `cmd/sup-kv/main.go` (`reflection.Register(gs)` in `listenGRPC` — D-09, no shutdown/log/metrics wiring touched), and `cmd/sup-kv/scenario_test.go` (`scenarioPortLo` 21000→28100, reserving [27900, 28100) for the phase-7 doc tests' fixed ports). `git diff` from the prior report's commit (a9d5491) to HEAD confirms exactly these three covered files changed, all additively; every other covered file is byte-identical.

Both prior human-verification items were closed by the maintainer in the 06-UAT session (2026-11-16T20:36Z, status complete, 2/2 passed): the CI-runner observation and the go_*/process_* family enumeration acceptance. This refresh re-collected independent first-hand evidence for both (see truths 5 and 10).

## Goal Achievement

### Observable Truths

Merged from ROADMAP Success Criteria (4) + all 9 PLAN frontmatters (16 truths). All locally executed tests below ran in this session at HEAD (0de996a), `-race -count=1`, go1.27.4.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1/06-01: SyncResponseRate out-of-domain (<=0 or >1e9) clamps to 500/s; limiter never silently disabled | ✓ VERIFIED | core.go:59-66 `maxSyncResponseRate = 1e9` + clamp in normalized(); repl focused run (clamp + all 9 sim scenarios) ok this session |
| 2 | SC1/06-01: drained-row Acked never holds a peer twice; closed-loop NotifyLocalWrite returns deterministic error | ✓ VERIFIED | repl.go:645 sort+`slices.Compact`; `ErrRunLoopExited` (5 refs); udprepl focused run (scenarios + drainedAckedDedup + afterDone) ok |
| 3 | SC1/06-01: no-op merge / matching Restore trigger zero dumps; changed record still confirms (PERS-04) | ✓ VERIFIED | engine.go:507-508 `displaced` predicate + live-read probe; persist suite ok this session |
| 4 | SC1/06-02: second SIGTERM during GracefulStop takes force path; exit 0 only after final confirmed dump | ✓ VERIFIED | main.go:238-244 armSignals → serve → shutdown → signal.Stop; full `./cmd/sup-kv/` package ok this session (12.4s, incl. TestShutdown_secondSignal); phase-7 diff touched only listenGRPC reflection |
| 5 | SC1/06-02 (backstop): CI timeout-minutes + no context rename + darwin loud-skip on the runners | ✓ VERIFIED | Backstop DISCHARGED with directly observed runner behavior: master CI run 44137814529 (at HEAD's merge) — jobs exactly `lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)`, `gate (darwin-amd64-cross)`, all success, names byte-identical to the ruleset contract; macos-15 green on `go test -race ./...` (cmd/sup-kv 3.258s ok — the broadcast-dependent scenarios can only pass via the D-12 Skipf path on darwin lo0, proving the skip executed rather than failed); all jobs completed in 56s-4m14s, within the 15/20 timeout-minutes keys carried in ci.yml:21,57 at HEAD. Maintainer UAT pass recorded (06-UAT.md item 1, PR #32 run 43950202200). Nuance: the Skipf reason text itself prints only under `-v`, which CI does not use — the printed-reason mechanism is verified at the call site (scenario_test.go:128-132 names the probed address and the darwin lo0 D-12 reason) and its execution is proven by the green darwin cell |
| 6 | SC3/06-03: malformed SUPKV_LOG_LEVEL/FORMAT is a startup error exiting 2 | ✓ VERIFIED | main.go:117-118,319 resolveLogKnobs env-error path; full cmd/sup-kv suite ok this session |
| 7 | SC3/06-03: JSON default, one JSON object per stderr line, tap under detail attr, 5 discard classes distinguishable, zero discards = zero lines | ✓ VERIFIED | setupLogging before wire; OnDiscard tokens in core; full cmd suite ok (W-1 flake caveat below) |
| 8 | SC2/06-04: --metrics-addr unset = no listener; set = one listener /metrics + /debug/vars + /debug/pprof/* behind optional mTLS | ✓ VERIFIED | main.go:191,305-306 flag/env with no default; metrics suite + full cmd suite ok this session |
| 9 | SC2/06-04: incomplete cert triple exits 2 naming all three envs before bind; full triple requires+verifies client certs | ✓ VERIFIED | main.go:392-395 checkMetricsCerts before wire; suites ok |
| 10 | SC2/06-04: /metrics tracks all seams live + go_*/process_* families | ✓ VERIFIED | 11 supkv_ families + 5 runtime families in metrics.go; metrics suite ok this session. Enumeration judgment RESOLVED: maintainer accepted the five-family set as sufficient for D-04 (06-UAT.md item 2, 2026-11-16) |
| 11 | SC1/06-05: partition seconds/days converge to supremum within bound; deterministic winner; conflicts counter; no journal (D-10) | ✓ VERIFIED | 9 TestScenario funcs present; repl scenario run ok this session (3.7s); "conflict journal" grep hits only assert its absence |
| 12 | SC1/06-05: clock jumps live/across restart, far-ahead clock monotonic+convergent; drift warnings counted | ✓ VERIFIED | TestScenario_clockJumps/farAhead in the same green run |
| 13 | SC1/06-06: asymmetric-topology bidirectional convergence + adversarial datagrams classify exactly per class | ✓ VERIFIED | TestScenario_convergence/discardClasses; udprepl focused run ok this session |
| 14 | SC1/06-07: all four D-11 exec scenarios pass on real processes/sockets/gRPC | ✓ VERIFIED | 4 TestExec funcs; full cmd/sup-kv suite ok at HEAD on the lifted [28100, 32678) port band (the phase-7 change to this file is the band lift + comment only); CI master run green on both linux native cells running `go test -race ./...` — the real-socket two-process runs in CI clause observed first-hand |
| 15 | SC4/06-08: make bench runs exactly the three benchmarks, ReportMetric whitespace-free units, absent from CI | ✓ VERIFIED | `make bench` executed this session at HEAD: BenchmarkDump10kKeys (bytes/op + dump-bytes), BenchmarkReplicationBurst (writes/s + max-unacked), BenchmarkConfirmationLatency (latency-ns/op + max-latency-ns), ok ×3; Makefile:26-27 unchanged (`-run '^$$' -bench .`, exactly three packages); 0 bench references in ci.yml; phase-7 Makefile diff is additive release targets only |
| 16 | SC1/06-09: durability flush platform-uniform — darwin F_FULLFSYNC, others byte-identical | ✓ VERIFIED | fsync_darwin.go F_FULLFSYNC present (persistfile unchanged since prior verification); `GOOS=darwin GOARCH={arm64,amd64} go build ./...` (covers the new reflection import) + `go vet ./internal/persistfile/` green this session; persistfile suite ok (4.8s) |

**Score:** 16/16 truths verified (0 present-but-unverified)

### Advisory (New Scope, Unevidenced)

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | No new-scope findings; the three phase-7 changes to covered files were diffed and verified additive |

### Required Artifacts

All 34 covered implementation files across the 9 plans exist, are substantive and wired at HEAD (Levels 1-4 re-confirmed via the presence checks and test runs above; `gsd-tools verify.artifacts` cannot score these plans' string-form artifact entries — tool limitation, unchanged from prior report). The three phase-7-modified artifacts:

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| Makefile | bench target + phase-7 release targets | ✓ VERIFIED | bench target byte-unchanged; release/release-snapshot additive; make bench green first-hand |
| cmd/sup-kv/main.go | shutdown/log/metrics wiring intact under the phase-7 reflection addition | ✓ VERIFIED | diff = 1 import + 1 Register call in listenGRPC; armSignals/signal.Stop, resolveLogKnobs, metrics guard, checkMetricsCerts all present; full package green |
| cmd/sup-kv/scenario_test.go | 4 exec scenarios on a collision-free port window | ✓ VERIFIED | 4 TestExec funcs; band lifted above the reserved doc-test ports with the rationale comment; suite green locally and on CI |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| SUPKV_LOG_LEVEL/FORMAT | stderr JSON records | resolveLogKnobs → setupLogging → SetDefault → tap/discard bridge | ✓ WIRED | unchanged; full suite green |
| core seams | /metrics families | OnDiscard + MergeStats/DumpStats/UnackedStats/SentByType → metrics.Sources | ✓ WIRED | metrics suite green |
| armSignals | final Engine.Flush + exit 0 | serve → shutdown (D-19) → signal.Stop | ✓ WIRED | main.go:238-244 intact |
| checkMetricsCerts | exitUsage (exit 2) | finalizeConfig → fail before bind | ✓ WIRED | main.go:392-395 intact |
| dumpfile sync sites | syncDurable | platform helper both sites | ✓ WIRED | persistfile unchanged; suite + darwin builds green |
| ci.yml gate job | TestExec_* in CI | `go test -race ./...` on native cells | ✓ WIRED | observed green on master run 44137814529 (ubuntu-latest, ubuntu-24.04-arm, macos-15) |

### Data-Flow Trace (Level 4)

Unchanged from the prior report and re-confirmed green by this session's runs: metrics Sources closures read the real 06-03 accessors (live-scrape test in the green metrics suite), the log pipeline parses real process stderr (green), and all scenario suites assert against the independent pointwise-supremum model (green). All FLOWING.

### Behavioral Spot-Checks

All commands run this session at HEAD, load average ~2.1.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| full binary package (shutdown, log, metrics, TLS, debug, 4 exec scenarios on the new port band) | `go test -race -count=1 ./cmd/sup-kv/` | ok 12.402s | ✓ PASS |
| repl: clamp + 9 sim scenarios | `go test -race -count=1 ./internal/app/repl/ -run 'TestSync_responseRateClamp\|TestScenario_'` | ok 3.658s | ✓ PASS |
| udprepl: 2 scenarios + drained dedup + closed-loop notify | `go test -race -count=1 ./internal/udprepl/ -run 'TestScenario_\|TestMirror_drainedAckedDedup\|TestNotifyLocalWrite_afterDone'` | ok 1.327s | ✓ PASS |
| persist engine suite | `go test -race -count=1 ./internal/app/persist/ -run 'TestEngine_'` | ok 2.365s | ✓ PASS |
| metrics package | `go test -race -count=1 ./internal/metrics/` | ok 1.021s | ✓ PASS |
| persistfile + storage suites | `go test -race -count=1 ./internal/persistfile/ ./internal/app/storage/` | ok 4.848s / ok 2.219s | ✓ PASS |
| benchmarks (TEST-05) | `make bench` | 3/3 benchmark lines with derived metrics, ok ×3 | ✓ PASS |
| darwin cross-compile (both arches, incl. new reflection import) + persistfile vet | `GOOS=darwin GOARCH={arm64,amd64} go build ./...` + `go vet ./internal/persistfile/` | both OK | ✓ PASS |
| CI gate at HEAD (lint + 4 gate cells, contexts, darwin cell) | `gh run view 44137814529` (+ PR runs 44097253271, 43950202200) | all jobs success; five contexts byte-identical; macos-15 green on full race suite | ✓ PASS |

### Probe Execution

No probes declared by any plan and no `scripts/*/tests/probe-*.sh` exist — N/A.

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| OBS-01 | 06-03, 06-04 | Metrics endpoint: unacked count+age, dump duration/errors, merge conflicts, drift warnings | ✓ SATISFIED | Families + suites green this session; REQUIREMENTS.md Complete |
| OBS-02 | 06-04 | Discards by class | ✓ SATISFIED | 5-label class vocabulary; metrics suite green |
| OBS-03 | 06-03 | Structured logging, distinguishable drop classes | ✓ SATISFIED | cmd suite green this session |
| TEST-04 | 06-01, 06-02, 06-05, 06-06, 06-07, 06-09 | Every documented failure scenario an automated test | ✓ SATISFIED | All 16 scenario tests green this session (sim 9, real-socket 2, exec 4, exec-level regression 1); CI native cells green at HEAD — the in-CI clause now first-hand observed |
| TEST-05 | 06-08 | Benchmarks: dump 10k, burst throughput, bounded confirmation latency | ✓ SATISFIED | `make bench` first-hand at HEAD, all three derived metrics reported |

Orphaned requirements: none — all five Phase-6 IDs claimed by plans and complete in REQUIREMENTS.md traceability.

### Anti-Patterns Found

Debt-marker scan (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) over all 34 covered implementation files at HEAD: clean. No bare `t.Skip` in any scenario file (loud capability Skipf only). No logging inside internal/app non-test code (D-09 holds). client_golang appears only as an indirect tool-graph dep — direct requires remain exactly grpc + protobuf (the phase-7 goreleaser/grpcurl additions are `tool` directives, release/doc scope, not runtime requires). No conflict journal anywhere (D-10 holds). No bench references in ci.yml (D-13/D-14 hold).

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | None found | — | — |

### Warnings (non-blocking, carried)

**W-1 (carried): freeUDPPort pick-to-bind race in cmd/sup-kv/main_test.go:139.** Still pick-then-bind on the shared ephemeral pool; not addressed since the prior report and re-flagged as open by the phase-6 code review (06-REVIEW WR-01, disposition open, 2026-11-18). The full package passed cleanly this session and CI is green on all cells; the flake is load-dependent infrastructure, not an observability-behavior doubt. Phase 7's port-band reservation does not cover this helper.

**W-2 from the prior report (WINDOWS.md D-16 ledger row) is RESOLVED** — row 4 marked fixed 2026-11-16T12:55Z.

**Phase-6 code review record:** 06-REVIEW-DISPOSITION.md carries 6 open findings (WR-01 port race as above; WR-02 discard-class label re-stringifying; WR-03 NewMux comment mechanism; IN-01 SentByType counting point; IN-02 release-snapshot syft prerequisite — phase-7 scope; IN-03 empty-disk bound window). All warning/info severity, none block a phase-6 must-have; tracked in the disposition ledger for triage.

### Human Verification Required

None. Both items from the prior report were closed in the 06-UAT maintainer session (2026-11-16, 2/2 passed, recorded in 06-UAT.md):

1. CI-runner backstop observation — passed (PR #32 evidence) and independently re-confirmed by this verifier at HEAD (master run 44137814529: contexts unchanged, darwin cell green, jobs bounded).
2. go_*/process_* family enumeration — maintainer accepted the five-family set as sufficient for D-04.

### Gaps Summary

No failed must-haves, no abstentions, no regressions from the phase-7 changes. The one previously abstained backstop truth is discharged with first-hand runner evidence. All 16 truths verified, all artifacts present/substantive/wired/data-flowing at HEAD, all prohibitions held, requirements coverage complete, fingerprint refreshed over the unchanged covered set.

---

_Verified: 2026-11-18T11:22:28Z_
_Verifier: Claude (gsd-verifier)_
