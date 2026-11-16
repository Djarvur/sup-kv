---
phase: 06-hardening-failure-scenarios-observability-benchmarks
verified: 2026-11-16T11:48:33Z
status: human_needed
score: 15/16 must-haves verified
covered_files: [".github/workflows/ci.yml", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-01-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-01-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-02-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-02-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-03-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-03-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-04-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-04-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-05-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-05-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-06-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-06-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-07-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-07-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-08-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-08-SUMMARY.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-09-PLAN.md", ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-09-SUMMARY.md", "Makefile", "cmd/sup-kv/main.go", "cmd/sup-kv/main_test.go", "cmd/sup-kv/scenario_test.go", "internal/app/internal/simnet/simnet.go", "internal/app/persist/bench_test.go", "internal/app/persist/engine.go", "internal/app/persist/engine_internal_test.go", "internal/app/persist/engine_test.go", "internal/app/repl/core.go", "internal/app/repl/discard_test.go", "internal/app/repl/merge_stats_test.go", "internal/app/repl/restart_test.go", "internal/app/repl/retransmit_test.go", "internal/app/repl/sim_bound_test.go", "internal/app/repl/sim_scenario_test.go", "internal/app/repl/sync_test.go", "internal/app/storage/store.go", "internal/app/storage/store_test.go", "internal/grpcapi/bench_test.go", "internal/grpcapi/server_test.go", "internal/grpcapi/wait_integration_test.go", "internal/metrics/metrics.go", "internal/metrics/metrics_test.go", "internal/persistfile/dumpfile.go", "internal/persistfile/fsync_darwin.go", "internal/persistfile/fsync_other.go", "internal/udprepl/bench_test.go", "internal/udprepl/repl.go", "internal/udprepl/repl_internal_test.go", "internal/udprepl/repl_test.go", "internal/udprepl/scenario_test.go", "internal/udprepl/smoke_test.go"]
covered_digest: "v4:sha256:07e6d67a4ac0efc4d7ea9b3da176292dc10dee5d7a23a48b2c478450fefad998"
behavior_unverified: 1 # The abstained backstop truth (CI-runner behavior) — detailed below and in human_verification
overrides_applied: 0
behavior_unverified_items:
  - truth: "CI gate jobs carry timeout-minutes (15 lint / 20 gate) without any check-run context rename, and the darwin matrix cells loud-skip broadcast-dependent scenarios with a printed capability reason — observable only on the CI runners (06-02, verification: backstop)"
    test: "Push/PR the phase-6 branch and watch the CI gate run: confirm the gate job is bounded by timeout-minutes 20 (job settings / a wedged cell failing rather than hanging) and the macos-15 cell passes with the loud capability skip naming the darwin lo0 no-BROADCAST reason"
    expected: "All four gate cells green or loud-skipped (macos-15 scenario/exec tests skip via the capability probe with the D-12 reason printed); no check-run context renamed (ruleset required_status_checks still matches lint / gate (<cell>))"
    why_human: "Backstop truth — runner behavior is not locally observable; gh shows zero CI runs on gsd/v1.0-milestone (the branch is local, PR-only development), so no runner evidence exists yet. Local presence+wiring is verified (timeout-minutes keys present in ci.yml, name fields byte-identical, loud-skip code paths in scenario tests) but per the backstop contract presence+wiring never qualifies."
human_verification:
  - test: "On the phase-6 PR, observe the CI gate: timeout bounds in effect and darwin loud-skips (the abstained 06-02 backstop truth)"
    expected: "gate (<cell>) contexts unchanged and green on ubuntu-latest / ubuntu-24.04-arm / macos-15; macos-15 logs the D-12 capability skip (loopback broadcast fan-out unavailable on darwin lo0) for the socket/exec scenario suites instead of failing; jobs bounded at 15/20 minutes"
    why_human: "Observable only on GitHub runners; no CI run exists on this branch yet (gh run list for gsd/v1.0-milestone is empty)"
  - test: "Judge the minimal go_*/process_* runtime-family enumeration for D-04 adequacy (06-04 flagged assumption A2, coverage D2 human_judgment: true)"
    expected: "A human decision that go_goroutines, go_memstats_alloc_bytes, go_memstats_sys_bytes, process_cpu_seconds_total, process_resident_memory_bytes are sufficient for D-04's runtime-families requirement — or a named addition to ship in Phase 7"
    why_human: "Machine verification proves rendering, completeness and live tracking of every declared family (TestMetrics_families, TestMetrics_optIn green), but the adequacy of a five-family enumeration is a flagged judgment call, not a machine-decidable property"
---

# Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks Verification Report

**Phase Goal:** Every documented failure scenario is an automated passing test and the node is observable enough to operate — the evidence that the core value holds under chaos
**Verified:** 2026-11-16T11:48:33Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Merged from ROADMAP Success Criteria (4) + all 9 PLAN frontmatters (16 truths). All named tests below were executed by the verifier in this session (single bounded selections per package, `-race -count=1`), not taken from SUMMARY claims.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1/06-01: SyncResponseRate out-of-domain (<=0 or >1e9) clamps to 500/s; limiter never silently disabled | ✓ VERIFIED | core.go:158-159 clamp with `maxSyncResponseRate = 1e9`; `TestSync_responseRateClamp` 6-row table incl. 1e9=1ns / 1e9+1=2ms boundaries — PASS |
| 2 | SC1/06-01: drained-row Acked never holds a peer twice; closed-loop NotifyLocalWrite returns deterministic error | ✓ VERIFIED | repl.go:644-645 sort+`slices.Compact`; NotifyLocalWrite single select + post-send done re-check returning `ErrRunLoopExited`; `TestMirror_drainedAckedDedup`, `TestNotifyLocalWrite_afterDone` — PASS |
| 3 | SC1/06-01: no-op merge / matching Restore trigger zero dumps; changed record still confirms (PERS-04) | ✓ VERIFIED | engine.go:507 `displaced` predicate; whole-snapshot `restoreChanges` (engine.go:581,601); `TestEngine_redundantDirty` (a)/(b) — PASS |
| 4 | SC1/06-02: second SIGTERM during GracefulStop takes force path; exit 0 only after final confirmed dump | ✓ VERIFIED | main.go:237-246 `armSignals` → serve → shutdown → `signal.Stop` after shutdown; `TestShutdown_secondSignal` — PASS (behavioral: exec-level, real process, exit code + dump asserted) |
| 5 | SC1/06-02 (backstop): CI timeout-minutes + darwin loud-skip on the runners | ⚠️ ABSTAINED (insufficient_spec) | Local presence+wiring verified: ci.yml:21,57 `timeout-minutes: 15/20`, `name: gate (${{ matrix.name }})` and all four matrix names byte-identical; loud D-12 skip code in scenario tests. No CI run exists on this branch (gh run list empty) — runner behavior unobserved → human item 1 |
| 6 | SC3/06-03: malformed SUPKV_LOG_LEVEL/FORMAT is a startup error exiting 2 | ✓ VERIFIED | `resolveLogKnobs` joins collected env errors; `TestLog_envValidation` (both knobs) — PASS |
| 7 | SC3/06-03: JSON default, every stderr line one JSON object, tap under detail attr, 5 discard classes distinguishable, zero discards = zero lines | ✓ VERIFIED | `setupLogging` (main.go:258-264) before wire; `OnDiscard` fired at all six discard sites with stable tokens (core.go:379-407); `TestLog_jsonDefault`, `TestLog_formatText`, `TestDiscard_onDiscardSeam` (7 rows), `TestDiscardLog_classes` — PASS (flakiness warning W-1 below) |
| 8 | SC2/06-04: --metrics-addr unset = no listener/goroutine; set = one listener /metrics + /debug/vars + /debug/pprof/* behind optional mTLS | ✓ VERIFIED | wire() guard (main.go:470-472) — nil server when unset; `NewMux` mounts expvar + full pprof route set by name (metrics.go:392-406); `TestMetrics_optIn`, `TestMetrics_noDefaultListener` (two-node collision shape), `TestMetricsTLS_required`, `TestDebugSurface` — PASS |
| 9 | SC2/06-04: incomplete cert triple exits 2 naming all three envs before bind; full triple requires+verifies client certs | ✓ VERIFIED | `checkMetricsCerts` before wire (main.go:391-399); `NewTLSConfig` RequireAndVerifyClientCert + ClientCAs + TLS1.2 floor (metrics.go:415-432); `TestMetrics_incompleteCertSet`, `TestMetricsTLS_required` — PASS |
| 10 | SC2/06-04: /metrics tracks seams live (unacked count+age, dump duration/errors, merge conflicts, drift warnings, discards by class, peers, sent/retransmitted) + go_*/process_* | ✓ VERIFIED | All 11 supkv_ families over the 06-03 accessors + 5 runtime families (metrics.go:216-356, Getrusage normalizing); `TestMetrics_families` (every family once, TYPE lines, zero-state), `TestMetrics_optIn` (live binary scrape) — PASS. Enumeration adequacy → human item 2 |
| 11 | SC1/06-05: partition seconds/days converge to supremum within documented bound; deterministic winner asserted; merge-conflicts counter; no journal (D-10) | ✓ VERIFIED | `TestScenario_partitionSeconds/Days/oneWay` PASS; conflicts assertion `conflicts >= 2*sharedKeys` in-code (sim_scenario_test.go:438); bounds reuse documented constants (`boundPartitionHeal`, `ackTimeoutMax`); zero journal artifacts in diff (grep clean) |
| 12 | SC1/06-05: wall-clock jumps live + across restart, far-ahead clock stay monotonic and convergent; drift warnings counted | ✓ VERIFIED | `TestScenario_clockJumps` (strict-dominance assertions both jump directions + ReplaceNode restart-time jump + runID check), `TestScenario_farAhead` (DriftWarnings asymmetry asserted, sim_scenario_test.go:248-252) — PASS |
| 13 | SC1/06-06: asymmetric-topology bidirectional convergence + adversarial datagrams classify exactly per class | ✓ VERIFIED | `TestScenario_convergence` (stores byte-equal + PendingCopy drain), `TestScenario_discardClasses` 3/3 subtests (per-class exactness, dup-node_id log names node_id + both handles) — PASS; own-echo stays in the untouched smoke test (smoke_test.go:276) |
| 14 | SC1/06-07: all four D-11 exec scenarios pass on real processes/sockets/gRPC | ✓ VERIFIED | `TestExec_killNineRestart`, `TestExec_bothNodesDown` (surgical checksum-bit corruption, alone-then-peer two-sided proof), `TestExec_sigtermFinalDump` (exit 0 + replicated record in final dump + lone restart), `TestExec_emptyDiskRecovery` (in-test derived N/rate bound) — all PASS; zero `time.Sleep` in the file |
| 15 | SC4/06-08: make bench runs exactly the three benchmarks, ReportMetric whitespace-free units, absent from CI | ✓ VERIFIED | `make bench` executed: all three lines with bytes/op + dump-bytes, writes/s + max-unacked, latency-ns/op + max-latency-ns, ok for all three packages; Makefile:26-27 `-run '^$' -bench .` scoped; zero bench references in ci.yml |
| 16 | SC1/06-09: durability flush platform-uniform — darwin F_FULLFSYNC, others byte-identical | ✓ VERIFIED | fsync_darwin.go `syscall.Syscall(SYS_FCNTL, fd, F_FULLFSYNC, 0)` (darwin build tag), fsync_other.go `f.Sync()` pass-through, dumpfile.go routes both sites (143, 175); `GOOS=darwin` arm64+amd64 `go build ./...` + `go vet ./internal/persistfile/` green this session; persistfile suite green |

Supporting facts verified beyond the truths:
- **06-09 D-16 (Task 3 dependency): RESOLVED.** All five prior phases (1-5) carry VERIFICATION.md files verified 2026-11-16T10:42-11:12Z with fresh `covered_digest` fingerprints, all `passed`; commits e47733c/9b87620/5e62bc7/7d87620-class (72497d7, e47733c, 9b87620, 5e62bc7, 8c548d8) + STATE.md updated at 945f80f ("06-09 D-16 resolved"). The 06-09 prohibition (never mark D-16 done without the runs) held.
- **SC1 real-socket-in-CI clause:** the gate job's native cells run `go test -race ./...` (ci.yml:82-88), which includes the TestExec_* two-process scenarios — structurally in CI, not only simulation. Observation of the actual run is part of human item 1.

**Score:** 15/16 truths verified (1 abstained — the backstop truth, presence+wiring verified, runner behavior unobserved)

### Required Artifacts

All 29 artifacts across the 9 plans exist, are substantive (10KB-59KB each), and are wired. `gsd-tools verify.artifacts` could not score them (plans use string-form artifact entries, not `{path, ...}` objects — tool limitation), so Levels 1-4 were verified manually:

| Artifact | Expected | Status | Details |
|----------|----------|--------|--------|
| internal/app/repl/core.go | clamp, OnDiscard, MergeStats, tokens | ✓ VERIFIED | All present; wired into HandleDatagram/applyRemote/scanRetransmit |
| internal/app/repl/sim_scenario_test.go | 9 TEST-04 sim scenarios | ✓ VERIFIED | 9 Test funcs; all PASS under -race |
| internal/app/internal/simnet/simnet.go | SetClockOffset + node lifecycle seams | ✓ VERIFIED | SetClockOffset/ReplaceNode/AddNode/Reregister present, used by scenarios |
| internal/udprepl/scenario_test.go | asymmetric topology + adversarial classes | ✓ VERIFIED | TestScenario_convergence/discardClasses PASS |
| cmd/sup-kv/scenario_test.go | 4 exec scenarios | ✓ VERIFIED | 4 TestExec funcs; all PASS |
| internal/metrics/metrics.go + metrics_test.go | exporter, mux, TLS, families | ✓ VERIFIED | 4 tests PASS; wired via main.go listenMetrics |
| internal/persistfile/fsync_darwin.go / fsync_other.go / dumpfile.go | platform sync adapter | ✓ VERIFIED | Both build tags correct; both sync sites routed |
| cmd/sup-kv/main.go | slog pipeline, signals, metrics wiring | ✓ VERIFIED | All three wirings traced end-to-end |
| bench files ×3 + Makefile | TEST-05 benchmarks | ✓ VERIFIED | `make bench` runs all three with derived metrics |
| .github/workflows/ci.yml | timeout bounds, contexts intact | ✓ VERIFIED | Keys added; names byte-identical (runner half = human item 1) |
| go.mod | direct requires exactly grpc + protobuf | ✓ VERIFIED | D-04 prohibition held; client_golang appears only as indirect tool-graph deps |

### Key Link Verification

`gsd-tools verify.key-links` cannot evaluate these plans' free-form string links (one false "Source file not found" on 06-02 is the parser, not the code). Manual wiring verification:

| From | To | Via | Status |
|------|----|----|--------|
| SUPKV_LOG_LEVEL/FORMAT | stderr JSON records | resolveLogKnobs → setupLogging → SetDefault (before wire) → tap/discard bridge | ✓ WIRED |
| core applyRemote/discards | adapter mirrors → exporter | OnDiscard + MergeStats/DumpStats/UnackedStats/SentByType → metrics.Sources → /metrics | ✓ WIRED (metrics.go reads exactly these accessors) |
| armSignals | final Engine.Flush + exit 0 | serve → shutdown (D-19) → signal.Stop after | ✓ WIRED (main.go:237-246) |
| checkMetricsCerts | exitUsage (exit 2) | finalizeConfig → fail before wire/listenMetrics bind | ✓ WIRED |
| dumpfile writeSynced/syncDir | syncDurable | platform helper on both sites | ✓ WIRED |
| ci.yml gate job | TestExec_* in CI | `go test -race ./...` on native cells | ✓ WIRED (structural) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| internal/metrics/metrics.go | Sources closures | Adapter.Discards/KnownPeers/UnackedStats/MergeStats/Retransmissions/SentByType + Engine.DumpStats | Yes — TestMetrics_optIn scrapes a live binary and asserts the supkv_ families | ✓ FLOWING |
| cmd/sup-kv log pipeline | node stderr | slog handler over tap/OnDiscard events | Yes — TestLog_jsonDefault parses real process stderr | ✓ FLOWING |
| sim/exec scenarios | store state vs chaosModelStore | real cluster writes through the virtual/real network | Yes — pointwise-supremum model comparison | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| repl core: clamp + discard seam + merge stats + retransmit + 9 sim scenarios | `go test -race -count=1 ./internal/app/repl/ -run '<phase tests>'` | 13/13 PASS (partitionDays 1.53s, burst 2.71s — virtual time) | ✓ PASS |
| udprepl: dedup + closed-loop + discard log + mirrors + 2 scenarios | same shape `./internal/udprepl/` | 6/6 PASS (incl. 3 discardClasses subtests) | ✓ PASS |
| persist: redundant-dirty + above-cover invariant + DumpStats | same shape `./internal/app/persist/` | 3/3 PASS (5 subtests) | ✓ PASS |
| metrics package (render/sort/handler/families) | `go test -race -count=1 ./internal/metrics/` | 4/4 PASS | ✓ PASS |
| binary: shutdown, log, metrics, TLS, debug, 4 exec scenarios | same shape `./cmd/sup-kv/` | 13/13 PASS (failures seen only in partial-selection re-runs — W-1) | ✓ PASS |
| benchmarks | `make bench` | 3/3 benchmark lines + ok ×3 | ✓ PASS |
| darwin cross-compile | `GOOS=darwin GOARCH={arm64,amd64} go build ./...` + vet | both OK | ✓ PASS |
| full cmd/sup-kv package (gate shape) | `go test -race -count=1 ./cmd/sup-kv/` ×5 | 5/5 ok | ✓ PASS |

### Probe Execution

No probes declared by any plan and no `scripts/*/tests/probe-*.sh` exist — N/A.

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| OBS-01 | 06-03, 06-04 | Metrics endpoint: unacked count+age, dump duration/errors, merge conflicts, drift warnings | ✓ SATISFIED | Seams + families + optIn scrape green |
| OBS-02 | 06-04 | Discards by class | ✓ SATISFIED | `supkv_datagrams_discarded_total{class}` 5-label vocabulary; TestMetrics_families |
| OBS-03 | 06-03 | Structured logging, distinguishable drop classes | ✓ SATISFIED | TestLog_* + TestDiscardLog_classes green |
| TEST-04 | 06-01, 06-02, 06-05, 06-06, 06-07, 06-09 | Every documented failure scenario an automated test | ✓ SATISFIED | All 16 scenario tests across sim/real-socket/exec PASS this session (see truth table); REQUIREMENTS.md marks TEST-04 Complete — the shared-ID gate closed with the last declarator (06-09) |
| TEST-05 | 06-08 | Benchmarks: dump 10k, burst throughput, bounded confirmation latency | ✓ SATISFIED | `make bench` first-hand run, all three metrics reported |

Orphaned requirements: none — every Phase-6 requirement in REQUIREMENTS.md traceability is claimed by at least one plan. TEST-04's six declarators (01/02/05/06/07/09) and OBS-01's two (03/04) all completed.

Scenario-matrix spot-check against SC1's enumerated list: datagram loss ✓ (TestScenario_loss), Ack loss ✓ (same, ladder dropped + inclusive ack_timeout_max), duplicates/reordering ✓ (loss phase 2, identical-supremum), own-echo ✓ (untouched smoke test), partitions seconds/days ✓ (+conflicts counter), one-way ✓, restart via RunID ✓ (sim restart + exec killNine), empty-disk at sync_response_rate ✓ (sim emptyDisk + exec emptyDiskRecovery), both-nodes-down ✓, corrupted dump record ✓ (bothNodesDown), wall-clock jumps live + across restart ✓ (clockJumps both halves), far-ahead clock ✓, ~1000 writes/s burst ✓ (sim burst + BenchmarkReplicationBurst), duplicate node_id ✓ (discardClasses + log pair), proto mismatch ✓ (discardClasses + TestDiscardLog_classes), real-socket two-process in CI ✓ (structural; observation in human item 1).

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| repl/sim_scenario_test.go | TEST-04 | 9 | 0 | 0 | Behavioral (model equality, bound-based RunUntil, strict-dominance compares, counter >= displaced) | OK |
| udprepl/scenario_test.go | TEST-04 | 2 | 0 (loud D-12 probe only) | 0 | Behavioral (byte-equal stores, per-class exact counters, log content) | OK |
| cmd/sup-kv/scenario_test.go | TEST-04 | 4 | 0 (loud D-12 probe only) | 0 | Behavioral (exit codes, on-disk dump grep, gRPC reads, derived time bound) | OK |
| metrics/metrics_test.go | OBS-01/02 | 4 | 0 | 0 | Value (byte-exact rendering, family enumeration) | OK |
| persist/engine{,_internal}_test.go | TEST-04 | new 3 | 0 | 0 | Behavioral (dump-file absence/presence, waiter resolution) | OK |

Disabled tests on requirements: 0 (the only skips are the documented loud D-12 capability probes). Circular patterns: none — scenario expected values come from the independent pointwise-supremum model (join of the write log), not from the system under test. Expected-value provenance: bounds reuse documented constants, never re-derived. Coverage quantity: all nine planned sim scenarios + 2 real-socket + 4 exec exist and pass.

### Decision Coverage

16/16 trackable CONTEXT.md decisions honored by shipped artifacts (`check.decision-coverage-verify`: honored=16, total=16, not_honored=[]).

### Anti-Patterns Found

Debt-marker scan (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) over all 34 covered implementation files: clean. Stub/empty-implementation/hollow-prop patterns: none — every surface renders or asserts real data (Level 4 FLOWING). No warnings from this scan.

### Warnings (non-blocking)

**W-1: TestLog_jsonDefault / TestLog_formatText retain the ephemeral pick-to-bind race (flaky under adverse machine load).** In verifier-run partial selections (`-run '^TestShutdown_secondSignal$|^TestLog_|^TestMetrics|^TestDebugSurface$|^TestExec_'`) these two tests failed 4-6 times across ~14 runs on this heavily loaded machine, in two modes: (a) `probe rx bind: address already in use` (main_test.go:714 — `freeUDPPort` pick-then-bind on the shared ephemeral pool), and (b) 10s timeout waiting for the DuplicateNodeID event (downstream of the same race disturbing the pair). In isolation both pass; the full `./cmd/sup-kv/` package suite passed 5/5 consecutive runs, and the full-repo gate was green this session. This is the exact latent race 06-07 deviation 3 diagnosed and fixed only for the scenario tests (private-window picker, [21000, 32678)) — the harness-file pickers were plan-barred and left at "historical exposure". The OBS-03 behavior itself is not in doubt (failures are infrastructure, not logging). Recommended: port the private-window picker to `freeUDPPort`'s users (or the log tests) in a follow-up; CI runners are far less loaded, so gate risk is low.

**W-2: WINDOWS.md ledger row stale.** `.planning/WINDOWS.md` still lists the 06-09 D-16 `unrun-verify` entry as open although the work is done and verified (five fresh fingerprints + STATE.md 945f80f). The ledger should be updated to closed by the orchestrator; not a code gap.

### Human Verification Required

1. **CI-runner backstop observation (06-02 truth, abstained)** — On the phase-6 PR: confirm the gate jobs are bounded (timeout-minutes 15/20 in effect — a wedged cell fails, not hangs), the four check-run contexts are unchanged (`lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)`, `gate (darwin-amd64-cross)`), and the macos-15 cell passes with the loud capability skip naming the darwin lo0 no-BROADCAST reason for the socket/exec scenario suites.
   - Expected: all cells green or loud-skipped with the D-12 reason printed; no context renames.
   - Why human: no CI run exists on gsd/v1.0-milestone yet (branch is local); runner behavior is not locally observable.

2. **go_*/process_* family enumeration judgment (06-04 flagged assumption A2)** — Judge whether the five-family minimal runtime enumeration (go_goroutines, go_memstats_alloc_bytes, go_memstats_sys_bytes, process_cpu_seconds_total, process_resident_memory_bytes) satisfies D-04's runtime-families requirement.
   - Expected: an explicit human accept, or a named family addition scheduled (Phase 7).
   - Why human: rendering/completeness/liveness are machine-verified; enumeration adequacy is the flagged judgment call.

(The third SUMMARY-flagged human_judgment item — 06-09 coverage D3, the D-16 regeneration — is RESOLVED and machine-verified this session: all five prior phases regenerated with fresh fingerprints, STATE.md records closure. No human action remains for it.)

### Gaps Summary

No failed must-haves. All 16 observable truths are either verified with first-hand test evidence (15) or abstained per the backstop contract pending runner observation (1). All artifacts exist, are substantive, wired, and data-flowing. All prohibitions held (guard kept, contexts unrenamed, core silence, no client_golang, no journal, no build tags, no sleep-then-check, no CI benchmark coupling, D-16 done for real). Two non-blocking warnings (flaky log-test port race; stale WINDOWS.md row) and two human-verification items (CI observation; runtime-family enumeration judgment) remain.

---

_Verified: 2026-11-16T11:48:33Z_
_Verifier: Claude (gsd-verifier)_
