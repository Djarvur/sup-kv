# Phase 2: Stand and Simulator as Tests - Research

**Researched:** 2026-11-19
**Domain:** Go test-harness engineering — real-socket two-process stands, deterministic discrete-event simulation, red-test gating, native fuzzing, mutation probes via `go build -overlay`, scheduled GitHub Actions
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01 (Red-test gating):** Critical-scenario red tests (R-01 paced sync over 20 000 keys, R-02 unacked-set cost and burst drain, R-03 closed-until-list interface, P-01 single value limit) live in the normal test packages but **skip unless `SUPKV_RED_TESTS=1`** is set. The scheduled CI workflow runs them with the variable set and **asserts the named tests FAIL** (a small runner script greps the test output for each expected-red test name in the FAIL section; absence of an expected red is an error). Red runs are recorded as phase evidence (test output saved under `.planning/phases/02-…/red-evidence/`). Phases 3–5 flip the assertion to "must pass" when the fix lands (the flip is a one-line change in the runner script's expectation table). Reversibility: reversible.
- **D-02 (Mutation sensitivity):** two named mutants ("sync without tombstones", "periodic sync disabled") are implemented as **code-level mutation probes in the runner script** (sed-edited copies of the target functions built in a temp overlay via `go build -overlay`), asserted to make the test fail on the fixed seed corpus; the production tree is never mutated in place.
- **D-03 (Stand placement):** the stand lives in a new package `internal/stand` (harness helpers: node spawn via the real `cmd/sup-kv` binary, temp dirs, broadcast socket setup, log capture, convergence polling) with scenario tests in `internal/stand/stand_*_test.go` guarded by `//go:build linux`. On non-Linux the package carries a single test that reports darwin replication untested loudly (TEST-05: no silent skips — a Linux CI skip is an error; the macOS cell states the limitation). Existing two-process exec patterns from v1.0's failure-scenario suite are the starting point, moved/extended rather than duplicated.
- **D-04 (Scenario set):** F9 (empty-disk peer, 10 000 × 1 KiB written offline), F9b (pure resync, 11 740 keys — scaled variant at 10 000), F14 (replaced node + `create_if_absent` seeding), burst 1000 writes/s × 15 s over 10 000 distinct keys (≥990 writes/s, unacked set empty 5 s after end), F2c (70 000-char key → `OK` then `sendto: message too long`, never replicated). Timing assertions use the review's stated bounds (F9b ≤ 2 × `sync_interval` + N/`sync_response_rate` + margin; burst ≥990/s and set-drain ≤5 s).
- **D-05 (Simulator shape):** simnet extensions land in `internal/app/internal/simnet` as additive API: per-node incoming-datagram budget (delivery refused + counted past budget), datagram size checks by the real encoder (`replproto` encode → length), record announcement delayed by a virtual dump latency, restart-with-preserved-state, and `Delete` in the cluster API. Chaos and scenario tests gain deletions in the mix.
- **D-06 (Chaos corpus):** `TestChaos_convergence` runs a **fixed seed corpus checked into `testdata/chaos-seeds.txt`** (regenerable by a generator test) plus fresh seeds each run; nightly-scale runs use the scheduled workflow. Combined-failure and long-partition (≥ `peer_timeout` + margin) cases become scenario tests with the documented bound (`not before first post-heal sync, not later than the bound`, with the no-fresh-loss qualifier).
- **D-07 (Seam shape):** filesystem seam under `persistfile`: a small `fsops` interface (write/fsync/rotate/rename/dir-fsync) with recording and failing fakes; a test asserts the exact operation ORDER and injects a failure at each step; a blocking `Save` fake proves `Set` does not return while the dump stands. `persist.Open` clock seeding is proven on the real composition path (`Open` with injected time earlier than dump max → first local write dominates).
- **D-08 (Pinning set):** negative decoder tests: corrupted-datagram table (bad CRC, truncated, wrong proto_version, bad run_id, oversize, empty key, tombstone-with-data) each asserting its discard class + counter; `FuzzDecode` and `FuzzDumpDecode` with in-repo `testdata/` corpora. Deletion replication pinned at four levels: codec round-trip (`Deleted: true` per message), simulator `Cluster.Delete`, two-process (Delete on A → `NotFound` on B, `"deleted":true` in B's dump), node-down-during-delete. Watch cut/backpressure: deterministic snapshot-then-live cut + slow-subscriber disconnect + `Send` error branch. Metrics by observed effect: `/metrics` after a conflict+retransmit+discard scenario asserts exact values. Per-decision tests reference 14.x by name in their comments/doc (TEST-13). Property shrinking: generators draw half the versions from a small `(l, c, node)` space; the printed replay command is executed by a test.
- **D-09 (CI topology):** new `.github/workflows/scheduled.yml` (cron nightly): 1000-seed chaos (`-shuffle=on -count=3`), bench smoke (`-bench . -benchtime=1x`), red-expected Critical run (D-01), `go test -race -count=20` over `cmd/sup-kv`, `udprepl` and affected packages (green ×3 across runs is the flakiness bar — a failing flake class that needs a Phase 3–5 fix carries its red test forward), and the nightly real-socket run at 10 000 keys × 1 KiB with the section-15 scenarios. PR CI keeps: lint, unit, race unit, Linux socket tests (mandatory, non-skippable). Benchmarks report p50/p99 without self-pacing on the real `persistfile` (no fake disk, default params).

### Claude's Discretion
- Exact helper APIs, test names (dot-consistent with existing suite), seed-corpus file format, runner-script implementation details, how the red-evidence directory is structured.
- Which existing v1.0 two-process tests move into `internal/stand` vs stay.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | `TestChaos_convergence` stable (0 fails/10 000 seeds) + mutation-sensitive | Seed corpus mechanics exist (`SUPKV_SIM_SEED`, `sim_chaos_test.go:29-56`); stabilization is a TEST-semantics change (see Pitfall 1); mutants realized via `go build -overlay` — empirically verified working under `go test -overlay` (go 1.27.4 probe, this session) |
| TEST-02 | Deletion replication at every level | `storage.Store.Delete` exists (store.go:203); simnet has NO `Cluster.Delete` today (simnet.go read in full); codec round-trip + two-process shapes from existing exec harness; dump is JSONL — `"deleted":true` greppable |
| TEST-03 | "Confirmed write is on disk" pinned | fsops seam target read in full (`persistfile/dumpfile.go`): current order MkdirAll→CreateTemp→Write→fsync(file)→rotate→rename→dir-fsync; blocking-`Save` proof via engine's synchronous dump path |
| TEST-04 | Startup gate tested as property; R-03 tests fail today | Gate opens on ANY first DataSync chunk or 1 s backstop — `gate.go:122 RunGate` / `core.go:631 handleDataSync`; no SyncID/ChunkIndex/ChunkCount in proto; red shape from reviewer's `TestReview_gateOpensBeforeSyncIsProcessed` + F14 stand scenario |
| TEST-05 | No silent skips; -race test binary | Current skip mechanism is capability-probe `t.Skipf` (`smoke_test.go:226-262`, `scenario_test.go:103-141`, `repl_test.go:674`) — the exact pattern to replace with errors on Linux; darwin named-test loud report per D-03 |
| TEST-06 | Simulator blind spots closed + nightly 10k real-socket run | simnet extension points mapped (no budget/size/dump-delay/restart-with-state/Delete today); dump-before-broadcast is real in production (RESULTS.md §4 note: "the DataReplication is broadcast after the dump is confirmed") but absent in sim |
| TEST-07 | `persist.Open` clock seeding proven | `Open(nodeID, now func() time.Time, driftWarn, dumps, storeCfg, engineCfg)` — `now` already injectable; watermark via `Watermark(records)` then `clock.NewClock(now, initialL, initialC, driftWarn)` (engine.go:124-150) |
| TEST-08 | Decoder negative table + fuzzing with in-repo corpora | Decode verdicts are `Decoded`/`BadCRC`/`Undecodable` (port/repl.go:16-26); core `DiscardStats` = BadCRC, Undecodable, BadProtoVersion, OwnEcho, DuplicateNodeID (core.go:205-211); `decode` at udprepl/repl.go:732; fuzz shapes exist in reviewer's probes (`FuzzZZDecode`, `FuzzZZDumpDecode`); corpus format empirically verified |
| TEST-09 | Bounds under combined failures + long partitions | Long-partition mechanics proven by existing `TestScenario_partitionDays` (cadence-scaling trick); combined-failure bound arithmetic from `boundPartitionHeal` (sim_bound_test.go:118-120); T-09 evidence: 8/10 000 seeds breach 2×sync_interval today, worst 88.8 s |
| TEST-10 | Benchmarks honest; p50/p99; smoke in CI | Three benchmarks exist and are real-path already (`BenchmarkDump10kKeys`, `BenchmarkReplicationBurst`, `BenchmarkConfirmationLatency`); deltas: p50/p99 reporting (currently ns/op + ReportMetric only) and smoke `-benchtime=1x` in CI (today "CI never runs it") |
| TEST-11 | Flake classes eliminated; `-race -count=20` green ×3 | Three T-11 flake classes: port pick-to-bind race (already mitigated by reserved window, cmd/sup-kv/scenario_test.go:93-117), SIGTERM-before-handler (binary window), README test deadline; baseline timings measured this session |
| TEST-12 | Metrics/config by observed effect | `/metrics` opt-in via `--metrics-addr`; `metrics.Snapshot`+`Render` render exact families — exact-value assertion feasible on the stand; per-parameter effect tests |
| TEST-13 | Every 14.x decision has a test referencing it by name | 11 ADRs exist: docs/adr/0001…0011 map 14.1…14.11 (verified by directory listing); SPEC-DEVIATIONS.md carries the registry to cross-link |
| TEST-14 | Watch cut + backpressure branches | Existing: `TestWatch_snapshotThenLive`, `TestWatch_exactCut`, `TestWatch_slowConsumer` (ResourceExhausted shed verified); missing: `Send` error branch coverage |
| TEST-15 | Property shrinking + working replay command | `rapid` NOT in go.mod and NOT imported anywhere (verified by grep this session); D-08's mechanism is small-space draws — dependency-free; current replay hint `go test ./... -run '<Test>' -args -seed=N` (testutil/rng.go:48) is broken for packages without the flag — fix the printed command and execute it from a test |
| TEST-16 | Scheduled workflow exists | `vulncheck.yml` is the in-repo cron precedent (`0 6 * * 1`, workflow_dispatch, paths-filtered PR trigger); `ci.yml` gate matrix documented below |
</phase_requirements>

## Project Constraints (from CLAUDE.md / AGENTS.md)

From `.claude/CLAUDE.md` and root `AGENTS.md` (both read this session):

- **Go version:** as pinned in `go.mod` — the directive is `go 1.27.4` [VERIFIED: go.mod:3]. Do not raise without an ADR.
- **Lint/test gates before push:** `make lint` (`go tool golangci-lint config verify` + `run ./...`) and `make test` (`go test -race -cover ./...`); CI runs the same [VERIFIED: Makefile:645-654, ci.yml:80-85].
- **Depguard architecture boundary:** `internal/app` may not import `net`, `os`, `syscall`, `google.golang.org/*` (rule `sup-kv-app-no-io`); `internal/port` stdlib+own-module only (rule `sup-kv-port-stdlib-only`). Any boundary change is a visible `.golangci.yml` diff — the fsops seam (D-07) lives under `persistfile` (an adapter), NOT under `internal/app`, precisely because of this rule.
- **Vendored lint config:** `.golangci.yml` is powerman/golangci-lint-strict for exactly golangci-lint 2.15.0, byte-unmodified behind its Origin header; project rules only inside `linters.settings.depguard.rules`.
- **No lint suppressions without justification:** every `//nolint` names the linter and carries a reason. Test-heavy harness code (globals, G115/G404 style conversions) has established in-repo precedent patterns to follow.
- **Zero direct runtime deps:** Phase 1 ships zero direct runtime requires; this phase adds **no new module dependencies at all** (test-only too — see Package Legitimacy Audit).
- **Tools via `go tool`:** never `tools.go`, never bare `go install`.
- **GSD workflow enforcement** (.claude/CLAUDE.md): repo edits only through GSD workflow — satisfied by this phase pipeline.
- **PR-only development, branch protection on master** (ruleset ENFORCED); every change lands via PR from a feature branch.

## Summary

Phase 2 converts the v1.0 review's verdict into machine-reproducible evidence, writing tests and harnesses only — red stays red by design. The research grounded every deliverable in the existing code: the deterministic simulator (`internal/app/internal/simnet`, read in full — its API surface has `Scheduler`/`Policy`/`Network`/`Cluster` with `Write`, `ReplaceNode` (empty store), `AddNode`, `SetClockOffset`, but NO budget, NO size check, NO dump delay, NO restart-with-preserved-state, NO `Delete`), the chaos/scenario/bound suites (`SUPKV_SIM_SEED` replay contract, three-seam quiescence, pointwise-supremum model assertion), the two-process exec harness (`cmd/sup-kv/scenario_test.go` — asymmetric-port topology, reserved port window, build-once binary, kill/reap plumbing), the persistfile write path (the fsops seam target whose real order is already the TEST-03 order), the persist engine's injectable-clock `Open`, the gRPC gate that opens on any first DataSync chunk (the R-03 defect), and the CI topology (`ci.yml` 4-cell gate matrix running `go test -race ./...` on two Linux cells + macOS; `vulncheck.yml` as the cron precedent).

Two findings matter most for planning. First, the Go-mechanics questions were answered **empirically on go 1.27.4** this session: `go test -overlay=cfg.json` DOES apply a mutated file at compile time despite the `go help build` caveat (probe built, mutant applied, test failed under the mutant — D-02's mechanism works as a plain `go test` invocation); Go native fuzzing runs the in-repo seed corpus from `testdata/fuzz/<FuzzName>/` in plain `go test` (fast CI smoke) and only mutates under `-fuzz` with `-fuzztime` (nightly deep mode); `-shuffle=on` reports its seed for reproducibility. Second, the hardest planning question is TEST-01's "0 failures on 10 000 seeds" against a test the review measured failing on 3.74% of seeds — the phase boundary forbids product fixes, so stabilization must come from test-side semantics (convergence-polling instead of first-quiet-moment checking; the quiescence condition is necessary but not sufficient under the R-05 delayed-heal behavior). That reconciliation is flagged as a checkpoint for the planner.

**Primary recommendation:** Build in this order — (1) stand package + harness move, (2) red tests + runner script (D-01) since every later deliverable asserts against them, (3) simnet extensions (D-05) then chaos corpus/stability (D-06, D-02 mutants), (4) seams/fuzz/deletion/metrics/watch pins (D-07, D-08), (5) scheduled.yml last (D-09) since its jobs invoke everything else.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Real-socket scenario execution (F9/F9b/F14/burst/F2c, deletions) | `internal/stand` (test tier; spawns real `cmd/sup-kv` processes) | `cmd/sup-kv` binary (unmodified) | Process lifecycle, real signals, real dumps are only provable against the shipped binary; the binary itself is never changed by this phase |
| Deterministic simulation extensions (budget/size/dump-delay/restart-state/Delete) | `internal/app/internal/simnet` (harness lib, non-test file) | `internal/app/repl/*_test.go` consumers | simnet is the one place the production core can be driven deterministically; extensions must stay goroutine-free, wall-clock-free, single-rng |
| Red-gating + expected-fail assertion | `.github/workflows/scheduled.yml` + runner script (scripts/, in-repo) | test files gated by `SUPKV_RED_TESTS` env | CI owns the assert-red contract; the tests themselves only gate on the env var |
| Mutation probes | runner script (temp overlay JSON + sed) | `go build -overlay` toolchain | No repo mutation; empirically verified compile-time application |
| fsops seam | `internal/persistfile` (adapter layer) | `internal/app/persist` tests via `port.DumpStore` | The depguard rule puts all `os` I/O in the adapter; fakes implement `port.DumpStore` for engine-level proofs |
| Decoder negative table + fuzzing | `internal/udprepl` tests (`decode`), `internal/app/persist` tests (`Decode`) | — | Decode verdicts originate in the adapter; dump decode in persist codec |
| Scheduled heavy machinery | GitHub Actions cron workflow | `make` targets it invokes | Nightly cost off the PR path; vulncheck.yml is the established pattern |
| Metrics-by-effect | `internal/stand` scenario + `internal/metrics` snapshot/render | — | Exact-value `/metrics` assertion needs a live node exposing `--metrics-addr` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go testing (vanilla) | go 1.27.4 (`go.mod:3`) | All tests, table-driven, `-race` | Project mandate; no assertion libraries [VERIFIED: AGENTS.md conventions, .claude/CLAUDE.md stack] |
| `go test -overlay` | go 1.27.4 | D-02 mutation probes: JSON `{"Replace": {diskPath: mutatedCopyPath}}` | Empirically verified this session: mutant applied under plain `go test -overlay`, production tree untouched [VERIFIED: empirical probe, go 1.27.4] |
| Go native fuzzing (`testing.F`) | go 1.27.4 | `FuzzDecode` / `FuzzDumpDecode`; PR CI = seed corpus only; scheduled = `-fuzz -fuzztime` | Stdlib; corpus format `go test fuzz v1` files under `testdata/fuzz/<FuzzName>/` verified empirically [VERIFIED: empirical probe + `go doc testing.F`] |
| `go test -shuffle` | go 1.27.4 | `-shuffle=on -count=3` in scheduled chaos job | `go help testflag` verified: on seeds from clock, integer pins seed, seed is reported [VERIFIED: `go help testflag`, go 1.27.4] |
| `os/exec` + existing harness patterns | stdlib | Stand node spawn/kill/restart | The `cmd/sup-kv` harness (startNode/launchNode/killAndReap/signalAndWaitExit) is the proven pattern to move [VERIFIED: cmd/sup-kv/scenario_test.go:365-385, main_test.go:214-350] |
| GitHub Actions `schedule` + `workflow_dispatch` | current | scheduled.yml | vulncheck.yml precedent: cron `"0 6 * * 1"`, dispatchable, paths-filtered PR trigger [VERIFIED: vulncheck.yml:7-13] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `math/rand/v2` (PCG split-seed) | stdlib | simnet fault draws; `simRNG` seed-split convention | All simulator randomness — same-seed byte-identical replay [VERIFIED: sim_bound_test.go:31-35] |
| `go test -json` | go 1.27.4 | Machine-readable FAIL events for the red-runner expectation table | Empirically verified: per-test events carry `Action:"fail"`, `Test:"<name>"` [VERIFIED: empirical probe this session] |
| `container/heap` | stdlib | simnet scheduler (exists) | Don't-hand-roll pick already embodied [VERIFIED: simnet.go:46-76] |
| `pgregory.net/rapid` | **NOT PRESENT** | — | Deliberately not added: D-08's shrinking is small-space draws + executable replay, dependency-free. Absence verified: not in go.mod, no import anywhere [VERIFIED: go.mod read; grep over internal/ and cmd/ returned nothing] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `go build -overlay` mutants (D-02, locked) | git worktree/stash mutation; separate mutant branch | Overlay is locked; worktree juggling mutates repo state and races CI checkouts — rejected by D-02 |
| `go test -json` parsing in runner | plain-text grep for `--- FAIL: <name>` | Text grep is simpler but risks name collisions in subtest output; `-json` events are unambiguous. Both viable; `-json` recommended (discretion area) |
| Single `scheduled.yml` (D-09, locked) | multiple workflows per job | Locked: one place to look |

**Installation:**
```bash
# Nothing to install. This phase adds zero module dependencies.
# All new packages are in-repo (internal/stand); all mechanisms are Go stdlib/toolchain.
go version  # go1.27.4 — matches go.mod pin
```

**Version verification:** `go.mod` read this session — `go 1.27.4` directive, zero direct runtime requires beyond grpc v1.85.0 + protobuf (unchanged by this phase). No ecosystem-registry lookups needed (no new packages).

## Package Legitimacy Audit

> This phase installs **no external packages** — not runtime, not test-only, not tools. All deliverables are in-repo packages plus Go stdlib/toolchain features verified against the local go 1.27.4 toolchain.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| (none — zero-install phase) | — | — | — | — | — | none |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

Note on `pgregory.net/rapid`: the v1.0 stack research listed it as "the one justified test-only dependency", but it was never added to go.mod and T-15 flags that mismatch. The locked D-08 decision resolves TEST-15 **without** it (small-space draws + executable replay command). Do not add rapid in this phase; adding it would contradict the locked decision's dependency-free mechanism.

## Architecture Patterns

### System Architecture Diagram

```
                        ┌─────────────────────────────────────────────────┐
                        │   .github/workflows/scheduled.yml (cron nightly) │
                        └──┬──────────┬──────────┬──────────┬──────────────┘
                           │          │          │          │
             1000-seed chaos│   bench smoke   red-run   race -count=20   nightly stand 10k
                           ▼          ▼          ▼          ▼              ▼
                  ┌──────────────────────────────────────────────────────────┐
                  │              scripts/red-runner.sh (expectation table)     │
                  │   SUPKV_RED_TESTS=1 go test -json → parse fail events →   │
                  │   assert each expected-red name FAILED → save evidence     │
                  │   go build -overlay mutants → assert TestChaos fails      │
                  └───────┬──────────────────────────────────────────────────┘
                          │ invokes
        ┌─────────────────┼─────────────────────────────────────┐
        ▼                 ▼                                     ▼
┌────────────────┐ ┌──────────────────┐ ┌──────────────────────────────┐
│ internal/stand │ │ internal/app/repl│ │ normal packages (persistfile │
│ //go:build     │ │ sim suites       │ │ udprepl grpcapi app/persist) │
│ linux          │ │ (chaos/bound/    │ │ red tests gated by env       │
│ spawns real    │ │  scenario)       │ │ SUPKV_RED_TESTS=1            │
│ cmd/sup-kv     │ └────────┬─────────┘ └──────────────────────────────┘
│ binaries       │          │ drives
│ real UDP       │          ▼
│ + gRPC         │ ┌──────────────────────────────────┐
└────────────────┘ │ internal/app/internal/simnet     │  D-05 extensions:
                   │ Scheduler→Policy→Network→Cluster │  budget, size-by-real-
                   │ (discrete-event, single rng)     │  encoder, dump delay,
                   └──────────────────────────────────┘  restart-with-state, Delete

 PR CI (ci.yml, tightened): lint + gate matrix (go test -race ./...) —
 Linux socket tests mandatory/non-skippable; macOS cell reports darwin
 untested via the named stand test.
```

Primary use case trace: a nightly cron fires → scheduled.yml jobs invoke the stand package (spawning real binaries over loopback broadcast), the sim suites (fixed `testdata/chaos-seeds.txt` + fresh seeds), the red-runner (env-gated red tests + overlay mutants), and the bench smoke — each job's failure is a machine-reproducible verdict on the review's findings.

### Recommended Project Structure
```
internal/stand/                 # NEW (D-03) — //go:build linux on scenario+helpers
├── stand.go                    # harness helpers (moved/extended from cmd/sup-kv): binary build,
│                               #   node spawn/kill/reap, port registry, asymmetric topology,
│                               #   log capture, convergence polling   [linux tag]
├── stand_test.go               # scenarios: F9, F9b-scaled, F14, burst, F2c, deletions,
│                               #   single-node SIGKILL, metrics-by-effect   [linux tag]
├── stand_red_test.go           # R-01/R-02/R-03 red tests (SUPKV_RED_TESTS=1 gate)  [linux tag]
└── stand_nondarwin_test.go     # //go:build !linux — the loud darwin-untested named test

internal/app/internal/simnet/
└── simnet.go                   # EXTEND additively (D-05): Cluster.Delete,
                                #   ReplaceNodeWithState (restart-with-preserved-state),
                                #   per-node datagram budget, size check via real encoder,
                                #   virtual dump delay on record announcement

internal/app/repl/
├── sim_chaos_test.go           # EXTEND: fixed-seed corpus from testdata/chaos-seeds.txt,
│                               #   deletions in the mix, convergence-poll semantics
├── sim_scenario_test.go        # EXTEND: combined-failure + long-partition (≥peer_timeout) tests
└── testdata/
    └── chaos-seeds.txt         # NEW (D-06) fixed seed corpus, generator-test regenerable

internal/app/repl/testdata/fuzz/      # (if FuzzDecode lands here) or under udprepl:
internal/udprepl/testdata/fuzz/FuzzDecode/          # NEW in-repo seed corpora
internal/app/persist/testdata/fuzz/FuzzDumpDecode/  # NEW
    ...

internal/persistfile/
└── fsops.go                    # NEW (D-07): small fsops interface; FileStore calls it
    (recording/failing fakes live in _test.go files)

scripts/
└── red-runner.sh (or .go)      # NEW (D-01/D-02): expectation table, -json parsing,
                                #   overlay mutant builds, evidence capture into
                                #   .planning/phases/02-…/red-evidence/

.github/workflows/
├── ci.yml                      # TIGHTEN (D-09): Linux socket tests non-skippable
└── scheduled.yml               # NEW (D-09): nightly jobs per the decision
```

### Pattern 1: Red-test gating (D-01) — env-gated skip + scheduled assert-red
**What:** Red tests live in normal packages; they `t.Skip` unless `SUPKV_RED_TESTS=1`; the scheduled runner sets the var, runs the packages, and asserts each expected-red name appears among FAIL events.
**When to use:** every Critical red test (R-01, R-02, R-03, P-01).
**Example:**
```go
// Source: in-repo conventions + D-01; go test -json verified this session
func TestRed_R01_pacedSync20k(t *testing.T) {
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	// ... scenario asserting the DOCUMENTED bound (which v1.0 code breaches) ...
}
```
```bash
# runner-side, machine-readable verdict (verified format this session):
SUPKV_RED_TESTS=1 go test -race -count=1 -json ./internal/stand/ ./internal/persistfile/ > out.json
# parse events: Action=="fail" && Test=="TestRed_R01_pacedSync20k" → expected; missing → error
```
The expectation table maps test name → `red` (today) and flips to `green` when the Phase 3–5 fix lands — a one-line change.

### Pattern 2: Mutation probes via `go build -overlay` (D-02) — EMPIRICALLY VERIFIED
**What:** sed-edit a copy of the target function's file into a temp dir; point an overlay JSON at it; run the chaos test under the overlay; assert it FAILS on the fixed seed corpus.
**Empirical proof (this session, go 1.27.4):** a probe module's `Add(a,b)` replaced by `a*b` through the overlay made `go test -overlay=overlay.json ./pkg/` fail while the un-overlaid run passed. The `go help build` caveat ("overlays will not appear … through go test") concerns runtime file reads, not compilation.
**Example:**
```bash
# Source: empirical probe /tmp/ovprobe (this session)
cp internal/app/repl/core.go "$TMP/core.go.mut"
sed 's/entries := c.listingChunks()/entries := withoutTombstones(c.listingChunks())/' \
    "$TMP/core.go.mut" > "$TMP/core.go"   # mutant "sync without tombstones"
printf '{"Replace": {"%s/internal/app/repl/core.go": "%s/core.go"}}' "$PWD" "$TMP" > "$TMP/ov.json"
SUPKV_SIM_SEED=$seed go test -overlay="$TMP/ov.json" ./internal/app/repl/ \
    -run '^TestChaos_convergence$' -count=1   # must FAIL under the mutant
```
Mutant targets in current code: "sync without tombstones" → the listing build (`core.go:751 syncListing` / `:771 listingChunks`, tombstones included today); "periodic sync disabled" → the periodic sync emission path (`Advance`'s sync timer, `core.go:724-750`).

### Pattern 3: simnet additive extensions (D-05) — keep the determinism contract
**What:** every extension is plain harness state or a policy field; no wall clock, no goroutines, all randomness through the one injected rng; new API follows `Cluster.Write`'s shape.
**Key anchors from the full read of simnet.go:**
- `Cluster.Write(i, key, data)` (simnet.go:569) = store.Set + core.NotifyLocalWrite + route + drive. `Cluster.Delete(i, key)` mirrors it with `store.Delete` (storage/store.go:203) and the tombstone record through the same notification path.
- `ReplaceNode(i, runID, offset)` (simnet.go:457) boots a FRESH EMPTY store — restart-with-preserved-state needs a sibling (e.g. `ReplaceNodeWithState(i, runID, offset, records)`) that seeds the replacement's store before wiring, keeping the in-place `Reregister` and timer-generation supersession.
- Budget/size/dump-delay join `Policy`/`Network.Send`/`scheduleDelivery`: budget = per-`netNode` counter consulted in `scheduleDelivery` (refused past budget, counted); size = encode the message with the real `replproto` encoder to get its length (simnet imports `internal/udprepl/replproto` for measurement — check the import direction is harness→adapter, which the depguard rules permit for a non-test harness file under `internal/app/internal/`, or precompute lengths at the test layer if the lint gate objects — this is a planner decision point); dump delay = an optional per-cluster latency applied to record announcements (`DataReplication`/sync answers) mirroring production's dump-then-broadcast order (RESULTS.md §4: "the DataReplication is broadcast after the dump is confirmed").
- **Existing determinism contracts to preserve:** `SetClockOffset`/`ReplaceNode`/`AddNode` explicitly "consume no rng draws and schedule no events" (simnet.go:439-441, 456, 479) — new API must document the same contract.

### Pattern 4: chaos seed corpus + fresh seeds (D-06)
**What:** `chaosSeed` (sim_chaos_test.go:46-56) currently returns `SUPKV_SIM_SEED` or `time.Now().UnixNano()`. The extension: when no env seed, ALSO draw the fixed corpus — e.g. the test iterates `testdata/chaos-seeds.txt` (one seed per line) in a subtest plus one fresh-seed run; a sibling generator test regenerates the file. The replay contract stays: `chaosReplayFmt = "replay: SUPKV_SIM_SEED=%d go test ./internal/app/repl/ -run '^TestChaos_convergence$'"` [VERIFIED: sim_chaos_test.go:30-31].
**Stabilization semantics (see Pitfall 1):** poll until stores equal the pointwise-supremum model (convergence), not merely until the three-seam quiescence (`chaosQuiescent`, sim_chaos_test.go:206-216) — quiescence is necessary but not sufficient while R-05 delays a returning peer's knowledge to its next periodic sync.

### Pattern 5: fsops seam (D-07) — the real order already matches TEST-03
**What:** `saveAtomically` (persistfile/dumpfile.go:97-113) performs: `MkdirAll` → `syncedTemp` (CreateTemp → Write → fsync → Close) → `placeFile` (rotate main→.prev via Rename, then rename tmp→main) → `syncDir` (open dir → fsync). TEST-03's asserted order "write → fsync → rotate → rename → dir fsync" is exactly this. The seam extracts those five operations behind a small interface so recording fakes assert the order and failing fakes inject at each step; the blocking-`Save` fake proves `Set` does not return while the dump stands (the engine's dump path is synchronous — `Run` serializes every Save; waiters block on the dump, engine.go:180-219).
**`persist.Open` clock seeding (TEST-07):** `Open(nodeID, now, driftWarn, dumps, storeCfg, engineCfg)` takes the `now` func directly (engine.go:124-150): seed a dump whose max version `l` exceeds the injected now, write locally, assert the first local write dominates all loaded versions. The real composition path — no seam changes needed.

### Pattern 6: decoder negative table + fuzzing (D-08/TEST-08)
**What:** the corrupted-datagram table drives `decode(datagram []byte) port.InboundDatagram` (udprepl/repl.go:732) and asserts the mechanical verdict + the core's discard counter. The verdict vocabulary is fixed: `Decoded`/`BadCRC`/`Undecodable` [VERIFIED: port/repl.go:16-26, quoted in Common Pitfalls]. Class + counter assertions ride `Adapter.Discards()` → `repl.DiscardStats` [VERIFIED: core.go:205-211, quoted below]. The reviewer's probes (`FuzzZZDecode`, `FuzzZZDumpDecode` in evidence/repro/tests/) are the worked shapes: seed with each valid message type encoded + empty bytes; the fuzz body asserts invariants (decoded verdict never carries nil Msg; core never emits an unencodable message; accepted dump records round-trip with verified checksums).
**Split:** PR CI runs `go test` (seed corpus only — fast, empirically <1 s in the probe); scheduled runs `-fuzz=FuzzDecode -fuzztime=...`. `-fuzz` requires the regexp to match exactly one fuzz test in exactly one package [VERIFIED: `go help testflag`].

### Pattern 7: stand harness move (D-03) — extend, don't duplicate
**What:** the exec harness in `cmd/sup-kv` is external test code (`package main_test`) — `internal/stand` cannot import it; the helpers move wholesale into `internal/stand` (linux-tagged) and `cmd/sup-kv` keeps (or sheds) its own suite per the discretion area. The load-bearing assets to carry: build-once binary (`smokeBinary`), `startNode`/`launchNode`/`smokeNode` (cmd/exit-channel/stderr buffer), `killAndReap` (SIGKILL exit shape `-1`), `signalAndWaitExit` (exit 0), `awaitServing`/`supKVConn`, the reserved port window + registry, the asymmetric-port topology (A binds P1 → broadcasts 127.255.255.255:P2; B mirrors), and `probeLoopbackSend`. Node knobs are flags with `SUPKV_` env twins [VERIFIED: main.go:94-126, 292-301] — the stand passes shrunken timers as flags exactly as the exec suite does today.

### Pattern 8: scheduled.yml (D-09) — one cron workflow, vulncheck.yml as template
**What:** `schedule: - cron: <nightly>` + `workflow_dispatch:` (the vulncheck.yml triad, minus the paths filter) [VERIFIED: vulncheck.yml:6-13]. Jobs: chaos-1000seeds (`-shuffle=on -count=3`), bench-smoke (`-bench . -benchtime=1x` over persist/udprepl/grpcapi — `make bench`'s package set minus thresholds), red-run (runner script; saves evidence), race-flake (`-race -count=20` over cmd/sup-kv + udprepl + affected), nightly-stand (10 000 × 1 KiB section-15 scenarios). Timing budget: local `-race` baselines measured this session — udprepl 1.3 s, cmd/sup-kv exec subset 6.2 s for count=1; ×20 on a 2-core hosted runner lands in the tens-of-minutes range for cmd/sup-kv — set the job `timeout-minutes` generously (the ci.yml precedent uses 15/20) [ASSUMED: CI-runner slowdown factor].

### Anti-Patterns to Avoid
- **Fixing the product while writing red tests:** the phase boundary forbids it; a red test that accidentally passes means the test asserts the wrong thing (or a fix leaked in) — the assert-red runner catches the first case, code review must catch the second.
- **Asserting wall-clock precision on the stand:** every timing assertion derives from the configured timers plus margin (the `TestExec_emptyDiskRecovery` pattern: `paced = N*(1s/rate)`, margin = gate + intervals + slack) — never a magic constant.
- **Silent capability skips on Linux:** any `t.Skipf` in the stand on a Linux CI cell is a TEST-05 violation. Skip-vs-error must branch on the platform, not on the probe outcome.
- **Mutating the production tree for mutation probes:** D-02 locks the overlay mechanism; sed-in-place would race the parallel gate cells.
- **Simulator features that break determinism:** any wall-clock read, goroutine, or map-iteration-order dependence in simnet extensions invalidates same-seed replay — the existing package doc forbids all three (simnet.go:1-18).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Mutant builds without repo mutation | git stash/worktree juggling; fork bombs of copies | `go build -overlay` with a temp JSON | One flag; compile-time application empirically verified; no git state touched |
| Fuzzing engine + corpus management | custom random-input loops | Go native fuzzing (`testing.F`) + `testdata/fuzz/` corpora | Coverage-guided mutation, crasher files written to the in-repo corpus automatically |
| Deterministic priority queue | hand-rolled timer list | `container/heap` (already in simnet) | Total (Time, Seq) order is subtle; already proven |
| Test-order randomization + reproducibility | shuffle inside Makefile scripts | `go test -shuffle=on` (seed reported) | Toolchain feature, seed printed for replay |
| Machine-readable pass/fail extraction | regex over human output | `go test -json` events (`Action`/`Test` fields) | Unambiguous per-test verdicts; verified this session |
| UDP port allocation under parallelism | hope + ephemeral ports | the reserved sub-ephemeral window + registry (already built) | Removes the pick-to-bind race structurally (T-11 class 1) |
| Process lifecycle (kill/reap/signal) | ad-hoc exec in each test | the moved `internal/stand` harness | Exit-code contracts (kill = -1, SIGTERM = 0) already pinned |
| Per-parameter effect matrix | one mega-test | table-driven one-test-per-parameter | TEST-12's acceptance wording maps 1:1 |

**Key insight:** every mechanism this phase needs already exists either in the Go toolchain (overlay, fuzzing, shuffle, -json) or in this repo's test suites (simnet, exec harness, probe patterns). The phase is composition and extension, not invention — the only genuinely new abstractions are the fsops interface and the runner script's expectation table, both small by construction.

## Runtime State Inventory

> Included because D-03/D-04 move existing two-process tests between packages (a light test-code migration). The shipped product's runtime state is untouched — this phase writes tests and harnesses only.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no product data formats, keys, or IDs change; test fixtures are temp-dir only [verified by phase scope: tests-only] | None |
| Live service config | None — no external services; CI workflows are files in git (ci.yml, release.yml, vulncheck.yml read this session) | Add scheduled.yml; tighten ci.yml |
| OS-registered state | None — no Task Scheduler/pm2/launchd/systemd registrations; the stand spawns short-lived processes via `os/exec` | None |
| Secrets/env vars | None added; new env var `SUPKV_RED_TESTS` is a test gate read by `_test.go` files only (no product code reads it) | None |
| Build artifacts | `bin/sup-kv` is gitignored and built fresh by the stand's build-once helper each run (the `smokeBinary` pattern builds into a temp dir); no stale artifact risk. The reviewer's repro subtree (`docs/reviews/2026-11-19-v1.0/evidence/repro/`) is a FOREIGN Go module — never import into the main graph [CITED: repro/README.md] | None |
| Test-code migration residue | Moving exec harness/tests out of `cmd/sup-kv` must not leave dead duplicates: `cmd/sup-kv/scenario_test.go` (four `TestExec_*`), the harness half of `main_test.go`, and possibly `internal/udprepl`'s socket scenarios are the candidates named by D-03/D-04 | Decide move-vs-stay per test (discretion area); delete or keep each deliberately; `go vet`/lint catch unused helpers |

## Common Pitfalls

### Pitfall 1: Chaos "stability" vs the R-05 delayed heal (the phase's hardest call)
**What goes wrong:** TEST-01 demands 0 failures on 10 000 seeds, but the review measured the current test failing on 3.74% of seeds (374/10 000; worst example seed 1795020540436451818), because quiescence is observed while a returning neighbor still lacks records only its next periodic sync will bring [CITED: REVIEW.md:316 (T-01)].
**Why it happens:** the test asserts convergence at the first quiet moment; the three-seam quiescence condition (no pending deliveries, no unacked, no queued responses) can hold before the R-05-shaped heal completes [VERIFIED: sim_chaos_test.go:206-216 quiescence reads only those three seams].
**How to avoid:** strengthen the test's semantics, not the product: treat quiescence as necessary-not-sufficient — after quiescence, poll (bounded by the hard cap, 10 × sync_interval past the last write) until every store equals the pointwise-supremum model; only then assert. Whether that alone reaches 0/10 000 is exactly what the fixed corpus + nightly scale must measure — plan a checkpoint: if genuine product-shape failures remain, they are R-05 red evidence and TEST-01 stability needs the planner's explicit qualifier until Phase 4.
**Warning signs:** failing seeds whose stores differ only on keys whose writers' peers were evicted (peer_timeout = 5 s default) mid-run.

### Pitfall 2: Silent skips on Linux CI (the T-05 defect this phase fixes)
**What goes wrong:** the existing socket tests skip via capability probes with `t.Skipf` when the 300 ms broadcast probe misses — 11 Linux socket tests could silently skip in CI [CITED: REVIEW.md:320 (T-05); VERIFIED skip sites: udprepl/smoke_test.go:226-262, scenario_test.go:103-141, repl_test.go:674-712, cmd/sup-kv/scenario_test.go:222-241].
**How to avoid:** in `internal/stand`, a probe failure on Linux is `t.Fatalf` (error), while non-Linux never runs the scenarios at all (build tag) and is covered by the loud darwin named test. Keep the probe's three-outcome shape (reached / not-reached / probe-error-with-retry) — only the not-reached-on-Linux outcome changes severity.

### Pitfall 3: `go help build`'s overlay caveat misread
**What goes wrong:** the docs say overlays "will not appear when binaries and tests are run through go run and go test" — a planner reading only that would reject D-02's mechanism.
**How to avoid:** the caveat concerns runtime reads by the executed binary, not compilation: empirically verified this session that `go test -overlay=…` applies the mutant at compile time (probe: un-overlaid pass, overlaid FAIL, pasted output). If a future Go release changes this, `go test -c -overlay` + run-the-binary is the verified fallback (also probed green this session).

### Pitfall 4: `-fuzz` in CI hangs the job
**What goes wrong:** `go test -fuzz=FuzzX` without `-fuzztime` runs until killed; also `-fuzz` errors out unless the regexp matches exactly one fuzz target in exactly one package [VERIFIED: `go help testflag`].
**How to avoid:** PR CI runs plain `go test` (seed corpus only); scheduled deep-fuzz always passes `-fuzztime`; one fuzz target per package (FuzzDecode in udprepl, FuzzDumpDecode in app/persist keeps them in separate packages — no regexp collision).

### Pitfall 5: `-race -count=20` timing and known flake classes
**What goes wrong:** ×20 multiplies every test's wall time; the three T-11 flake classes (port pick-to-bind race — already structurally mitigated by the reserved window; SIGTERM before the handler is installed — a real binary-side window; README/doc test without its own deadline) will surface at ×20 and "a failing flake class that needs a Phase 3–5 fix carries its red test forward" (D-09's own bar).
**How to avoid:** budget generous `timeout-minutes`; keep the port-window mechanism intact during the harness move; the ×3-green-across-runs bar spans scheduled runs, not one run.
**Measured baselines (local, linux/amd64, -race, count=1):** udprepl 1.3 s; cmd/sup-kv exec subset 6.2 s [VERIFIED: measured this session].

### Pitfall 6: Red-runner grep false positives
**What goes wrong:** grepping plain output for a test name matches `=== RUN` lines, t.Logf replay hints, and package lines — a test that PANICKED or was SKIPPED could look "found".
**How to avoid:** parse `go test -json` events: expected-red means an event with `Action:"fail"` AND `Test:"<name>"` (verified event shape this session). Treat `Action:"skip"` for an expected-red name as an error (the env gate failed to open).

### Pitfall 7: simnet size-check import direction
**What goes wrong:** D-05's "size checks by the real encoder" pulls `internal/udprepl/replproto` into a file under `internal/app/internal/simnet` — the app side of the depguard boundary must not import adapters beyond what the existing simnet already does (it imports `internal/app/*` and `internal/port` only today [VERIFIED: simnet.go:20-32]).
**How to avoid:** either (a) precompute/measure lengths at the test layer and configure the budget test with real encoder measurements, or (b) have simnet accept a `SizeOf func(port.Message) int` injected by the test file (which may import replproto freely) — harness stays lint-clean either way. Planner picks; (b) preserves "same handlers, measured by the real encoder" while keeping the non-test file's imports stdlib+app.

### Pitfall 8: F9b scale vs CI wall-clock
**What goes wrong:** F9b's documented bound (~83 s at 11 740 keys) breached to ~1000 s on the stand — the red R-01 evidence. A 10 000-key stand scenario with default timers costs minutes; the burst scenario (1000 w/s × 15 s) plus drain adds more. PR CI runs socket tests on every push — an unbounded suite breaks the 20-minute gate timeout.
**How to avoid:** D-04/D-09 split: PR CI runs the stand scenarios at CI-viable scales (the existing exec suite's shrunken-timer pattern); the full 10 000 × 1 KiB scale runs nightly in scheduled.yml. The reviewer's measured bounds are asserted where the scale supports them.

## Code Examples

### The exact discrete values tests will assert against (verbatim quotes)

```go
// Source: internal/app/repl/core.go:205-211 (read this session)
type DiscardStats struct {
	BadCRC          uint64
	Undecodable     uint64
	BadProtoVersion uint64
	OwnEcho         uint64
	DuplicateNodeID uint64
}
```

```go
// Source: internal/port/repl.go:16-26 (read this session) — the three decode outcomes
const (
	// Decoded means the envelope and body both unmarshaled and passed the
	// CRC gate; Msg carries the decoded message.
	Decoded DecodeStatus = iota
	// BadCRC means the envelope unmarshaled but its CRC32 does not cover
	// the body bytes; the adapter did NOT decode the body (D-02).
	BadCRC
	// Undecodable means the datagram is not a well-formed envelope (or
	// body) at all — including schema-domain violations such as a run_id
	// that is not 16 bytes.
	Undecodable
)
```

```go
// Source: internal/app/persist/codec.go:25 (read this session) — P-01's single constant
const maxRecordBytes = 61440
```

```go
// Source: internal/app/repl/retransmit_test.go:38-40, sync_test.go:27-28 (read this session)
ackTimeout    = 200 * time.Millisecond
ackTimeoutMax = 5 * time.Second
suspendWindow = 60 * time.Second
syncInterval  = 30 * time.Second
```

```go
// Source: internal/app/repl/sim_chaos_test.go:30-31 (read this session)
simSeedEnv     = "SUPKV_SIM_SEED"
chaosReplayFmt = "replay: SUPKV_SIM_SEED=%d go test ./internal/app/repl/ -run '^TestChaos_convergence$'"
```

```go
// Source: cmd/sup-kv/scenario_test.go:105-107 (read this session) — reserved port window
scenarioPortLo    = 28100
scenarioPortHi    = 32678
scenarioPortTries = 32
```

```yaml
# Source: .github/workflows/vulncheck.yml:7-13 (read this session) — the cron precedent
on:
  schedule:
    - cron: "0 6 * * 1" # weekly, Monday 06:00 UTC
  pull_request:
    paths:
      - go.mod
      - go.sum
  workflow_dispatch:
```

### The review's measured numbers the stand tests encode

| Scenario | Reviewer's measurement | The bound the test encodes |
|----------|------------------------|-----------------------------|
| F9 (empty-disk + offline writes, 10k×1KiB) | B has 10 000 keys after 14.2 s; then retransmit storm: unacked 8909→0 took ~14.5 min; 949 552 retransmits; 1 044 161 socket drops by ~5.5 min [VERIFIED: evidence/stand/RESULTS.md:29] | R-02 red: set must empty within N/`sync_response_rate` + 10 s, ≤3N retransmits (REVIEW.md:142) |
| F9b (pure resync, 11 740 keys) | 9908 keys at 160 s stuck through 271 s (4 rounds no progress; missing keys in deciles 6–10), all 11 740 at ~1015 s; documented bound ≈ 83 s [VERIFIED: RESULTS.md:30; REVIEW.md:133] | R-01 red: full convergence ≤ 2 × sync_interval + N/`sync_response_rate`, 20/20 runs (REVIEW.md:135) |
| F14 (replaced node seeds default) | B's `create_if_absent` succeeds right after gate opens; 3 s later BOTH nodes hold the default [VERIFIED: RESULTS.md:37] | R-03 red: interface closed until the full key list; before it B answers `Unavailable` read+write; after, `AlreadyExists` (REVIEW.md:149) |
| Burst 1000 w/s over NEW keys | achieved 465 w/s, p99 10 942 ms; 42 s later 4327 unacked, 102 835 retransmits [VERIFIED: RESULTS.md:71] | ≥990 w/s achieved; unacked empty 5 s after end (D-04) |
| F2c (70 000-char key) | Set → OK; `sendto: message too long` on every send; never on B after 2 × sync_interval [VERIFIED: RESULTS.md:20] | the red behavior today; XPORT-03 fixes it in Phase 3 |
| F15/P-01 (value_max 64000, 63 000-byte value) | Set OK; after restart the key is gone (`discarded 1 undecodable dumped records at startup`) [VERIFIED: RESULTS.md:38] | P-01 red: record of exactly `value_max_bytes` survives restart at any accepted flag value (REVIEW.md:201) |
| Chaos instability | 374/10 000 seeds fail; seed 1795020540436451818 reproduced master's red [CITED: REVIEW.md:316] | TEST-01: 0/10 000 after test-side stabilization |
| Combined failures | 8/10 000 seeds exceed 2 × sync_interval, worst 88.8 s [CITED: REVIEW.md:324 (T-09)] | TEST-09 bound: not before first post-heal sync, not later than the bound, no-fresh-loss qualifier |

### The gate that must stay red (R-03's attach point)

```go
// Source: internal/grpcapi/gate.go:122-150 (read this session) — opens on ANY sync, or 1 s backstop
func (s *Server) RunGate(ctx context.Context) {
	deadline := s.opts.StartupGate
	if deadline <= 0 {
		deadline = defaultStartupGate
	}
	backstop := time.NewTimer(deadline)
	defer backstop.Stop()
	poll := time.NewTicker(gatePollInterval)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-backstop.C:
			s.openGate()
			return
		case <-poll.C:
			if s.opts.SyncArrived != nil && s.opts.SyncArrived() {
				s.openGate()
				return
			}
		}
	}
}
```
The SyncArrived mirror is "A1 semantics: any received DataSync datagram counts" [VERIFIED: gate.go:39-46] — one chunk (even empty) opens the gate. The red test (F14 shape) asserts the interface stays closed until the full list; current code fails that assertion by construction.

### Fuzz harness seed-corpus seeding (verified shapes from the reviewer's probes + this session's probe)

```go
// Source: shape verified from evidence/repro/tests/probe/.../zz_fuzz_internal_test.go:25-44
// (consult-only reference) + empirical corpus-format probe this session
func FuzzDecode(f *testing.F) {
	run := [16]byte{9, 9, 9}
	ver := version.PackVersion(1_700_000_000_000, 7, 3)
	for _, msg := range []port.Message{
		port.Hello{},
		port.DataReplication{Key: "k", Record: storage.Record{Version: ver, Data: []byte("v")}},
		port.DataReplication{Key: "k", Record: storage.Record{Version: ver, Deleted: true}},
		port.Ack{Key: "k", Version: ver},
		port.DataRequest{Key: "k"},
		port.DataSync{Entries: []port.SyncEntry{{Key: "a", Version: ver}, {Key: "b", Version: ver}}},
	} {
		d, err := encodeDatagram(3, run, msg)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(d)
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) { /* verdict + core-safety invariants */ })
}
```
In-repo corpus files land at `testdata/fuzz/FuzzDecode/<entry>` with the two-line format `go test fuzz v1` + `[]byte("...")` — empirically verified loaded and executed this session.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `tools.go` blank imports | `tool` directives in go.mod | Go 1.24 | Repo already migrated; no action |
| `testing/synctest` experimental | stable since Go 1.25 | Go 1.25 | Available but the repo's determinism rides simnet's own scheduler — no change needed |
| External fuzzing tools | native `testing.F` | Go 1.18+ | Stdlib path chosen; no deps |
| `go test` text output parsing | `-json` events | long-standing | Runner script uses it |

**Deprecated/outdated:** nothing in this phase's domain is deprecated; `pgregory.net/rapid` in the stack doc is aspirational history — the codebase never adopted it and D-08 supersedes it with the dependency-free shrinking mechanism.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Hosted CI runners run the race suite ~2–4× slower than this machine (local baselines verified; CI factor estimated) | Pitfall 5, scheduled.yml budget | Job timeout too tight → scheduled job fails spuriously; pick generous timeouts with explicit headroom |
| A2 | Chaos instability is fully test-side: polling-until-convergence semantics reach 0/10 000 seeds without product fixes | Pitfall 1, TEST-01 | If some failures are genuine R-05-shaped product behavior, TEST-01 needs an explicit qualifier/red-forward until Phase 4 — planner checkpoint required |
| A3 | macOS lo0 lacks the BROADCAST flag so the darwin leg cannot run two-node stand tests (in-repo probe messages document this) | Pattern 7, TEST-05 | Low — worst case the darwin cell runs the probe and reports per its own verdict; the named loud test covers either way |
| A4 | A 10 000-seed chaos proof run costs on the order of 1–3 h wall (per-seed cost extrapolated from suite shape; not measured) | TEST-01 acceptance, D-06 | The one-off 10k proof needs a scheduled/manual-dispatch slot or a phase-verification run with a long timeout |
| A5 | GitHub Actions cron granularity supports nightly schedules reliably (standard behavior; vulncheck.yml proves the mechanism exists but its own schedule cadence is weekly) | Pattern 8 | Low — nightly cron is a first-class Actions feature |
| A6 | `cmd/sup-kv`'s remaining tests after the harness move can still build the binary independently (or the stand's builder is reused via a small shared helper) — the exact split is the discretion area | Runtime State, Pattern 7 | Low — worst case the builder helper is duplicated then deduplicated in review |

## Open Questions

1. **How is the 10 000-seed TEST-01 acceptance proven, given D-09 schedules 1000 seeds nightly?**
   - What we know: D-06 fixes a corpus + fresh seeds per run; D-09 schedules 1000 seeds `-shuffle=on -count=3`; TEST-01's acceptance says 0 failures on 10 000 seeds; a full run costs hours (A4).
   - What's unclear: whether the 10k proof is a one-off phase-gate run (local or a scheduled workflow's manual dispatch with a bigger seed count), or an accumulating nightly statistic.
   - Recommendation: planner should plan a one-off phase-verification run (dispatchable job with a seed-count parameter, default 1000 nightly / 10 000 on demand) — the workflow parameterization is trivial.
2. **Where do the simnet size measurements come from — test-layer injection (a `SizeOf` func) vs simnet importing replproto?**
   - What we know: simnet's non-test file imports only stdlib + internal/app + internal/port today; D-05 says "size checks by the real encoder (`replproto` encode → length)".
   - What's unclear: whether the lint gate (depguard, strict config) tolerates a harness file importing the adapter's proto package.
   - Recommendation: prefer the injected-func shape (Pitfall 7b); a quick `go tool golangci-lint run` probe during planning settles it definitively.
3. **Which cmd/sup-kv tests move to internal/stand vs stay?**
   - What we know: D-03 says moved/extended rather than duplicated; the four `TestExec_*` scenarios are the natural move candidates; `TestExec_emptyDiskRecovery` is the seed of the F9 stand test; discretion is explicitly Claude's.
   - What's unclear: whether `cmd/sup-kv` keeps any socket tests (its doc tests and lifecycle tests are not socket tests and stay by construction).
   - Recommendation: move all four `TestExec_*` scenarios; keep main_test.go's single-process lifecycle/log/metrics tests in place (they test one process, not replication).
4. **What exactly must the R-03 red tests assert to be genuinely red TODAY (and not accidentally green)?**
   - What we know: gate opens on any first DataSync chunk or 1 s backstop; F14's acceptance has five enumerable assertions (value on A unchanged; `Unavailable` both directions pre-list; `AlreadyExists` post-list for peer-known keys; three-part list keeps closed; next local write after the max-version part gets a greater version).
   - What's unclear: how many of the five the current code fails instantly (all five presumably fail; the runner's assert-red only needs the test to fail, but the test should assert the full acceptance so it becomes the Phase 4 acceptance test on flip).
   - Recommendation: write the red tests as the FULL PROTO-01 acceptance assertions (the flip-to-green in Phase 4 then requires zero test edits beyond the expectation table) — consistent with D-01's reversibility contract.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ (local + CI via `go-version-file: go.mod`) | go1.27.4 | — |
| Linux real sockets (loopback broadcast to 127.255.255.255) | internal/stand, udprepl scenarios | ✓ (local linux/amd64; CI ubuntu cells) — the capability the existing probes verify at runtime | — | none needed on Linux; darwin excluded by build tag (D-03) |
| GitHub Actions Linux runners | stand in CI, scheduled.yml | ✓ (ubuntu-latest, ubuntu-24.04-arm already in the matrix) | — | — |
| cron workflow mechanism | scheduled.yml | ✓ (vulncheck.yml precedent) | — | workflow_dispatch for manual proof |
| `go test -overlay` / `-fuzz` / `-shuffle` / `-json` | D-02 / TEST-08 / D-09 / D-01 runner | ✓ all verified on go1.27.4 this session | 1.27.4 | `-c`+run-binary for overlay (also verified) |
| golangci-lint 2.15.0 (go tool directive) | lint gate over new test files | ✓ (Makefile + CI, pinned) | 2.15.0 | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go vanilla `testing`, go 1.27.4; `-race` mandatory; table-driven with named subtests |
| Config file | none — flags on the command line / Makefile targets (`make test` = `go test -race -cover ./...`) |
| Quick run command | `go test -race -count=1 ./internal/stand/ ./internal/app/repl/ -run 'Test(Stand|Chaos|Bound|Scenario)'` |
| Full suite command | `make test` (repo-wide `-race -cover`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | chaos stable + mutant-sensitive | simulation (deterministic) + runner-script mutants | `go test -race ./internal/app/repl/ -run '^TestChaos' -count=<seeds>`; runner overlay step | partially — chaos exists, corpus + mutants are Wave 0 |
| TEST-02 | deletion replication ×4 levels | unit (codec) + simulation + exec | `go test -race ./internal/udprepl/ ./internal/app/internal/simnet/... ./internal/stand/` | ❌ Wave 0 |
| TEST-03 | fsops order + blocking Save + SIGKILL process | unit (fakes) + exec | `go test -race ./internal/persistfile/ ./internal/stand/ -run 'Test(Fsops|Persist)'` | ❌ Wave 0 |
| TEST-04 | R-03 acceptance fails on current code | simulation + exec (red-gated) | `SUPKV_RED_TESTS=1 go test -json ... -run '^TestRed_R03'` | ❌ Wave 0 |
| TEST-05 | no silent skips; -race binary | structural (skip-vs-error policy) + darwin named test | `go test -race ./internal/stand/` on both platforms | ❌ Wave 0 |
| TEST-06 | simnet extensions + nightly 10k | simulation + scheduled exec | simnet tests + scheduled.yml nightly job | ❌ Wave 0 |
| TEST-07 | Open clock seeding | unit | `go test -race ./internal/app/persist/ -run 'TestOpen'` | ❌ Wave 0 |
| TEST-08 | negative table + fuzz corpora | unit table + fuzz seeds (PR) / -fuzz (nightly) | `go test -race ./internal/udprepl/ ./internal/app/persist/ -run 'TestDecode|Fuzz'` | ❌ Wave 0 |
| TEST-09 | combined-failure + long-partition bounds | simulation | `go test -race ./internal/app/repl/ -run '^TestScenario_(Combined|LongPartition)'` | ❌ Wave 0 |
| TEST-10 | benchmarks honest, p50/p99, smoke | benchmark smoke in CI | `go test -run '^$' -bench . -benchtime=1x ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/` | partially — benchmarks exist; percentile reporting is Wave 0 |
| TEST-11 | `-race -count=20` green ×3 | scheduled race soak | scheduled.yml job over cmd/sup-kv + udprepl + affected | ❌ Wave 0 (workflow) |
| TEST-12 | metrics/config by observed effect | exec + unit tables | `go test -race ./internal/stand/ -run 'TestMetrics'` + per-parameter tables | ❌ Wave 0 |
| TEST-13 | per-decision 14.x named tests | unit/scenario referencing ADR names | repo-wide grep-checkable: test files cite `ADR 0001…0011` / `14.1…14.11` | partially — many decision behaviors tested; name-referencing pass is Wave 0 |
| TEST-14 | Watch cut/backpressure branches | unit (grpcapi) | `go test -race ./internal/grpcapi/ -run '^TestWatch'` | partially — cut/slow-consumer exist; `Send` error branch is Wave 0 |
| TEST-15 | small-space draws + executable replay | property | `go test -race ./internal/app/... -run 'Prop'` + the replay-executor test | ❌ Wave 0 |
| TEST-16 | scheduled workflow exists | workflow file + dispatchable jobs | `gh workflow run scheduled.yml` / cron | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** the quick run command (target < 60 s; stand scenarios excluded on non-Linux by build tag)
- **Per wave merge:** `make test` (full `-race -cover` suite)
- **Phase gate:** full suite green; PLUS the phase-specific gate: the red-runner asserting every expected-red test FAILED with evidence captured under `.planning/phases/02-…/red-evidence/`; the 10 000-seed chaos proof run (Open Question 1)

### Wave 0 Gaps
- [ ] `internal/stand` package scaffolding (linux-tagged helpers + non-linux loud test) — carries TEST-02/03/04/05/06/12 exec halves
- [ ] `scripts/red-runner` (expectation table, `-json` parsing, overlay mutants, evidence dir) — carries D-01/D-02
- [ ] `internal/app/repl/testdata/chaos-seeds.txt` + generator test — carries D-06
- [ ] `testdata/fuzz/FuzzDecode/` + `testdata/fuzz/FuzzDumpDecode/` seed corpora — carries TEST-08
- [ ] `.github/workflows/scheduled.yml` — carries D-09/TEST-16 (last wave: its jobs invoke everything above)

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1` (config read this session). This is a test-only phase: no product auth/session/crypto code changes. The security-relevant surface is the untrusted-datagram handling that the corrupted-datagram table and fuzz targets pin.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | AUTH-01 (datagram HMAC) explicitly deferred to v2 [VERIFIED: REQUIREMENTS.md v2 section] |
| V3 Session Management | no | no sessions in this phase |
| V4 Access Control | no | trusted-LAN model; no ACL changes |
| V5 Input Validation | **yes** | the corrupted-datagram table (bad CRC, truncated, wrong proto_version, bad run_id, oversize, empty key, tombstone-with-data) + `FuzzDecode`/`FuzzDumpDecode` pin the decoder's reject behavior; discard classes + counters are the observability control |
| V6 Cryptography | no | CRC32 integrity only (existing); no crypto introduced or changed |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed/corrupted datagram flood on the broadcast segment | Tampering | CRC gate + discard classes (`BadCRC`, `Undecodable`) counted and logged; pinned by TEST-08's table + fuzzing |
| Oversize datagram (≥ 65 507 UDP ceiling; EMSGSIZE) | Tampering | kernel rejects `sendto` (F2c's observed `message too long`); receiver-side oversize class in the table |
| Wrong proto_version / schema-domain violations (run_id ≠ 16 bytes) | Tampering | `Undecodable` verdict with distinct class + counter; PROTO-07 (Phase 4) adds pre-decode version checks |
| Duplicate node_id impersonation | Spoofing | `DuplicateNodeID` counter; PROTO-06 names the config error (Phase 4) — this phase pins the counter's existence via metrics-by-effect |

## Sources

### Primary (HIGH confidence)
- In-repo code read this session (all claims tagged `[VERIFIED: <path:lines>]`): `internal/app/internal/simnet/simnet.go` (full), `internal/app/repl/sim_chaos_test.go`, `sim_bound_test.go`, `sim_scenario_test.go`, `retransmit_test.go`/`sync_test.go` constants, `internal/app/repl/core.go` structure + DiscardStats, `internal/port/repl.go`, `internal/persistfile/dumpfile.go` (full), `internal/app/persist/engine.go` (Open path), `internal/app/persist/codec.go:25`, `internal/grpcapi/gate.go` (RunGate), `internal/grpcapi/watch_test.go`/`wait_test.go`/`bench_test.go`, `internal/udprepl/repl.go` decode area, `cmd/sup-kv/scenario_test.go` (full), `cmd/sup-kv/main_test.go` harness inventory, `cmd/sup-kv/main.go` env/flag surface, `internal/app/internal/testutil/rng.go` + `gen.go`, `internal/metrics/metrics.go`, `internal/app/storage/store.go` method list, `go.mod`, `Makefile`, `.github/workflows/ci.yml`, `.github/workflows/vulncheck.yml`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `.planning/config.json`, `docs/adr/` listing, `docs/SPEC-DEVIATIONS.md` head
- Empirical toolchain probes this session (go 1.27.4, outputs pasted in the conversation): `go test -overlay` mutant application (plain and `-c` forms), native-fuzz seed-corpus loading + `-fuzztime` run + corpus file format, `go test -json` event shape, `-shuffle`/`-fuzz` flag semantics from `go help build` / `go help testflag` / `go doc testing.F`, local `-race` timing baselines
- Committed review evidence (public docs, read this session): `docs/reviews/2026-11-19-v1.0/REVIEW.md` §3/§4/§5/§6/§11/§14/§15, `evidence/stand/RESULTS.md` (full), `evidence/repro/README.md`, repro test-file shapes (`synctail_probe_test.go`, `zz_fuzz_internal_test.go`, `zz_fuzz_test.go`, persist review repro test names)

### Secondary (MEDIUM confidence)
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` §11 figures not independently reproduced here (chaos 3.74% failure rate, mutation-census survival list, T-11 flake-class inventory) — cited, treated as the review's measurements the phase encodes

### Tertiary (LOW confidence)
- CI-hosted runner slowdown factor for `-race -count=20` (A1) — local baselines verified, CI factor estimated
- Per-seed chaos wall-clock cost at 10 000-seed scale (A4) — extrapolated, not measured

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies; every mechanism verified against the local go 1.27.4 toolchain or existing in-repo code
- Architecture: HIGH — every pattern anchored to a file read in full this session; the one genuinely new mechanism (overlay mutants) empirically probed
- Pitfalls: HIGH for in-repo/review-documented traps (skip sites, flake classes, chaos failure mechanism), MEDIUM where CI-runner behavior is estimated

**Research date:** 2026-11-19
**Valid until:** 2026-12-19 (stable: the domain is this repo's own test infrastructure; only the Go toolchain's minor revs could move the overlay/fuzz specifics)
