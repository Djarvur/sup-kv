---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 08
subsystem: testing
tags: [benchmarks, testing, makefile, udp-replication, grpc, persistence]

requires:
  - phase: 05-grpc-api-and-binary
    provides: gRPC Set with wait_for_replication seam (KnownPeers/PendingCopy ServerOptions)
  - phase: 06-hardening (plan 06-06)
    provides: asymmetric-port two-node topology helper + loud darwin skip (scenario_test.go)
provides:
  - BenchmarkDump10kKeys — dump at 10k keys through the real Encode+persistfile Save path (bytes/op, dump-bytes)
  - BenchmarkReplicationBurst — two-node ~1000 writes/s burst convergence rate on real sockets (writes/s, max-unacked)
  - BenchmarkConfirmationLatency — bounded wait_for_replication drain latency under a background burst (latency-ns/op, max-latency-ns)
  - make bench — the manual D-13 door (three packages, -run anchored empty, not in CI)
affects: [Makefile, internal/udprepl test helpers, internal/grpcapi test helpers]

actuals:
  tokens: 7900   # chars/4 over the realized diff (31604 chars)
  tasks: 3
  commits: 3
plan_head_before: aba7120144bbd6df52dd54a60a9a644079c897aa
plan_head_after: 8aea7fc385aa9539d55f97df24f6b71286e8fa2f

tech-stack:
  added: []   # stdlib testing.B + b.ReportMetric only — zero dependency changes (go mod tidy stable)
  patterns:
    - benchmark-through-real-path (no fakes of the measured seam: real Encode+Save, real UDP topology, real gRPC wait seam)
    - convergence-inside-the-benchmark (the run ends only when the peer serves every written key — a lost write fails, never skews)
    - helper generalization *testing.T -> testing.TB for benchmark reuse (single topology implementation)

key-files:
  created:
    - internal/app/persist/bench_test.go
    - internal/udprepl/bench_test.go
    - internal/grpcapi/bench_test.go
  modified:
    - Makefile
    - internal/udprepl/scenario_test.go
    - internal/udprepl/smoke_test.go
    - internal/grpcapi/wait_integration_test.go
    - internal/grpcapi/server_test.go

key-decisions:
  - "Reuse over duplication: the 06-06 asymmetric-topology helpers (and their loud darwin skip) were generalized from *testing.T to testing.TB so the benchmarks ride the exact same construction — no second topology implementation in the package (acceptance criterion)"
  - "newScenarioNode's cleanup now waits for Run's return after cancelling — the adapter closes its socket only in Run, so repeated runs (go test -count=2, verified) never leak bound ports"
  - "testing.TB carries neither Context() nor Deadline() (Go 1.24 added them to T and B only, never the interface): Deadline is read through a structural type assertion in waitFor; context.Background() sites carry the two new documented usetesting nolints (the linter's suggested fix does not compile)"
  - "BenchmarkReplicationBurst keys are namespaced per framework b.N-escalation run (atomic run counter), so 'node B serves all b.N keys' can never be satisfied by an earlier run's residue"
  - "BenchmarkConfirmationLatency's background burst writes through the author-notification path (store.Set + NotifyLocalWrite, the scenarioWrite precedent) so the ~1ms cadence never serializes on dump confirmation; measured writes go through the full gRPC Set+wait path"
  - "Unacked waits fail the latency benchmark (b.Fatalf naming the key) — bounded confirmation latency is the property under test; the burst benchmark likewise fails on any unobserved write"

requirements-completed: [TEST-05]

coverage:
  - id: D1
    description: "BenchmarkDump10kKeys + the make bench door (Makefile target, three-package scope, unit tests excluded, CI untouched)"
    requirement: TEST-05
    verification:
      - kind: other
        ref: "make bench — BenchmarkDump10kKeys line with bytes/op + dump-bytes metrics and ok line for internal/app/persist"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./internal/app/persist/... — exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "BenchmarkReplicationBurst — ~1000 writes/s two-node burst convergence rate on real UDP sockets (writes/s + max-unacked via ReportMetric)"
    requirement: TEST-05
    verification:
      - kind: other
        ref: "go test -run '^$' -bench '^BenchmarkReplicationBurst$' -benchtime=200x ./internal/udprepl/ — 200 iterations, 936 writes/s, ok line"
        status: pass
      - kind: other
        ref: "-count=2 leak spot-check (both runs green, cleanup waits for Run) and go test -race ./internal/udprepl/ green"
        status: pass
    human_judgment: false
  - id: D3
    description: "BenchmarkConfirmationLatency — bounded wait_for_replication appearance-to-drain latency under a background burst through the real gRPC wait seam (latency-ns/op + max-latency-ns; unacked fails the run)"
    requirement: TEST-05
    verification:
      - kind: other
        ref: "make bench (final run) — BenchmarkConfirmationLatency 132 iterations, ~10.3ms latency-ns/op, 17.2ms max-latency-ns, ok line for internal/grpcapi"
        status: pass
      - kind: other
        ref: "go test -race ./... and go tool golangci-lint run ./... — both exit 0"
        status: pass
    human_judgment: false

duration: 18min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 08: D-13 Benchmarks Summary

**Three stable-named benchmarks (dump at 10k keys, two-node burst convergence rate, bounded confirmation latency under burst) behind a manual `make bench` door — the repo's first benchmarks, reported via b.ReportMetric with zero CI coupling and zero dependency changes.**

## Performance

- **Duration:** ~18min
- **Started:** 2026-11-16T01:02:32Z
- **Completed:** 2026-11-16T01:20:41Z
- **Tasks:** 3/3
- **Files modified:** 8

## Accomplishments
- First benchmarks in the repo (baseline at plan time: zero `func Benchmark` — now three), each measuring through the REAL seam it names: persist.Encode + persistfile.Save for the dump, the 06-06 asymmetric UDP topology for the burst, the gRPC wait_for_replication seam over real adapters for the latency.
- Correctness inside the benchmarks: BenchmarkReplicationBurst ends only when node B serves every written key under its exact version; BenchmarkConfirmationLatency fails on any unacked report — a lost write or an unbounded drain is a bug, never a data point.
- Derived metrics with whitespace-free units via b.ReportMetric: bytes/op + dump-bytes; writes/s + max-unacked; latency-ns/op + max-latency-ns — ns/op stays visible everywhere; no thresholds anywhere (D-13), no profiles stored (D-14), `.github/` byte-untouched, `go mod tidy` stable.

## Task Commits

Each task was committed atomically:

1. **Task 1: BenchmarkDump10kKeys + the make bench door** - `9aa7bc0` (test) — tracer; feedback gate re-ran `make bench` end-to-end under auto mode (pass) before expansion
2. **Task 2: BenchmarkReplicationBurst** - `9997abc` (test)
3. **Task 3: BenchmarkConfirmationLatency** - `8aea7fc` (test)

## Files Created/Modified
- `internal/app/persist/bench_test.go` — BenchmarkDump10kKeys: 10,000-key store (deterministic size table, frozen HLC clock fake — Tick's counterMax rule keeps versions strictly monotonic), per-iteration real Encode + persistfile Save
- `internal/udprepl/bench_test.go` — BenchmarkReplicationBurst: ~1ms-paced writes on node A, measured window runs to full convergence on node B
- `internal/grpcapi/bench_test.go` — BenchmarkConfirmationLatency: bufconn gRPC client, background burst goroutine, per-write submission-to-acked-report timing
- `Makefile` — `bench` target (doc comment, .PHONY, three-package scope, `-run '^$'`)
- `internal/udprepl/scenario_test.go`, `internal/udprepl/smoke_test.go` — helpers generalized to testing.TB; newScenarioNode cleanup waits for Run
- `internal/grpcapi/wait_integration_test.go`, `internal/grpcapi/server_test.go` — waitFor/integrationUDPPort/probeLoopbackBroadcast generalized to testing.TB

## Decisions Made
See key-decisions in frontmatter — the load-bearing one: benchmarks reuse the exact 06-06 topology helpers via testing.TB generalization instead of any copy, per the plan's own acceptance criterion.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Test helpers took *testing.T; benchmarks hold *testing.B**
- **Found during:** Task 2 (and the same shape in Task 3)
- **Issue:** The plan mandates reusing 06-06's topology helper "verbatim" with "no second topology implementation", but the helpers' *testing.T signatures made them uncallable from a *testing.B benchmark goroutine.
- **Fix:** Generalized the six udprepl helpers (freeUDPPort, waitFor, scenarioProbeBroadcast, newScenarioNode, startScenarioPair, scenarioWrite) and three grpcapi helpers (waitFor, integrationUDPPort, probeLoopbackBroadcast) from *testing.T to testing.TB with thelper's `tb` naming. t.Context() sites became context.Background() (TB has no Context method); waitFor's hard-deadline clamp now reads Deadline() through a structural type assertion. newScenarioNode's cleanup additionally waits for Run's return (port-leak-free -count reruns). Behavior of every existing test is unchanged (full race suite green).
- **Files modified:** internal/udprepl/scenario_test.go, internal/udprepl/smoke_test.go, internal/grpcapi/wait_integration_test.go, internal/grpcapi/server_test.go
- **Verification:** go test -race ./... green; benchmarks ride the same helpers
- **Committed in:** 9997abc, 8aea7fc

**2. [Rule 1 - Lint] Vendored-strict-config findings on the new code**
- **Found during:** Tasks 1-2 (per-package lint runs)
- **Issue:** noinlineerr rejected inline `if err := ...; err != nil` in the persist benchmark; thelper required `tb` naming; usetesting suggested tb.Context() which does not compile on the TB interface; golines capped the first nolint comment length.
- **Fix:** Code direction everywhere possible (plain assignment error handling, tb renames, shortened comments); two documented `//nolint:usetesting` directives where no code-direction fix exists (the suggestion itself cannot compile — same sanctioned pattern as the existing gosec G115 nolints).
- **Files modified:** internal/app/persist/bench_test.go, internal/udprepl/scenario_test.go, internal/udprepl/smoke_test.go
- **Verification:** go tool golangci-lint run ./... exit 0
- **Committed in:** 9aa7bc0, 9997abc

---

**Total deviations:** 2 auto-fixed (1 Rule 3, 1 Rule 1)
**Impact on plan:** None on scope or contracts — helpers' observable test behavior unchanged; benchmarks gained exact reuse instead of duplication.

## Issues Encountered
None beyond the deviations above. The tracer feedback gate (auto mode active) re-ran `make bench` end-to-end after Task 1's commit and passed before expansion.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
TEST-05 delivered in full; `make bench` is the manual comparison door. 06-09 (final plan of the phase) can proceed; no shared requirements blocked by this plan.

---
*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*

## Self-Check: PASSED

- key-files created: all FOUND on disk (3 bench files + Makefile)
- commits 9aa7bc0, 9997abc, 8aea7fc: all ancestors of HEAD
- evaluation-scope --commits-only: resolved with 3 commits in scope
- Plan verification re-run: make bench exit 0 (all three benchmark lines + ok lines), go test -race ./... ok, golangci-lint ./... exit 0, git diff .github/ empty, go mod tidy stable
