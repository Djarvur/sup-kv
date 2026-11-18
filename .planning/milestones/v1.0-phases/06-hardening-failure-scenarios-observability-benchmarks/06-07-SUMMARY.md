---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 07
subsystem: binary-exec-testing
tags: [exec-scenarios, two-process, kill-9, corrupted-dump, sigterm, empty-disk-recovery, sync-response-rate, real-sockets, test-only]

requires:
  - phase: 05-grpc-api-and-binary
    provides: the shipped binary's D-19 shutdown sequence, exit-code contract, and the cmd/sup-kv exec harness (TestMain build-once, startNode/reap, signalAndWaitExit/assertExitCode, awaitServing/supKVConn)
  - phase: 06-hardening-failure-scenarios-observability-benchmarks
    provides: "06-02's TestShutdown_secondSignal signal-ownership fix and 06-06's asymmetric-topology pattern (Pattern 5) the scenarios ride"
provides:
  - cmd/sup-kv/scenario_test.go — the exec half of TEST-04 (D-11): TestExec_killNineRestart, TestExec_bothNodesDown (with the corrupted-dump-record case), TestExec_sigtermFinalDump, TestExec_emptyDiskRecovery, all on the shipped binary over real sockets and real gRPC
  - Shared exec scaffold: asymmetric two-node launcher with shrunken timers, loud D-12 capability probe (no build tags), real-gRPC Set/Get helpers, killAndReap pinning the SIGKILL exit shape
  - Zero production-code changes; the only harness-file touch is one documented nolint comment
affects: [TEST-04 verification, phase verification, milestone close, 06-08 benchmarks (topology pattern)]

actuals:
  tokens: 7455   # chars/4 over the realized diff (29822 diff chars across 2 files)
  tasks: 3
  commits: 3      # MEASURED: git rev-list --count cf1923b9..HEAD
plan_head_before: cf1923b98a6e9a766c139ef261fd2de7c4f3999b
plan_head_after: f9c1bc2c15735970362eb97e4f4e87424c3de6df

tech-stack:
  added: []
  patterns:
    - "scenario UDP ports picked from a private window below ip_local_port_range with a claim-then-release registry — no rival :0 bind can be issued there, structurally removing the pick-to-bind race the shared ephemeral pool suffers under parallel suites"
    - "proving on-disk discard vs peer-heal separately: restart the corrupted node ALONE first (absence is then only possible from its own load), then bring the peer back (heal is then only possible from replication)"
    - "in-test derived timing bounds: N * floor(1s/rate) from the passed flags plus a stated margin (gate window + timer rounds + CI slack) — never magic constants, never wall-clock precision (Pitfall 6)"

key-files:
  created:
    - cmd/sup-kv/scenario_test.go
  modified:
    - cmd/sup-kv/main_test.go   # one documented unparam nolint on awaitServing — comment-only

key-decisions:
  - "Scenario-local capability probe with per-suite skip text (the 06-06 precedent): the skip names the probed address, the wait, the darwin lo0 no-BROADCAST reason, and exactly what stays proven on the skipping platform (internal/udprepl/scenario_test.go, TestMain_lifecycle)"
  - "Surgical dump corruption = one checksum bit (value XOR 1) inside one record line: the JSONL stays structurally valid so the generation parses, the record's checksum no longer matches RecordChecksum, and PERS-02 discards exactly that record — proven by the restarted node's own 'discarded 1 undecodable dumped records at startup' stderr line (the per-record path, not the .prev fallback)"
  - "TestExec_bothNodesDown restarts A ALONE before B: with no peer, the corrupted key's absence and the other keys' presence are provably A's own disk load; B's restart then heals the discarded record — the plan's assertions made two-sided"
  - "TestExec_sigtermFinalDump kills B before the lone A restart, so the only possible source of the replicated key is the final dump itself; exit-code negatives (exitDumpFailed/exitUsage) deliberately not duplicated — 06-02/main tests own them"
  - "Empty-disk bound derived in-test: paced = 20 * (1s/50) = 400ms; margin = startup gate 1s + 2x hello + 2x sync + 2s CI slack; measured 116-195ms — the flagged assumption A5's timing assertion held, the documented fallback (convergence-within-deadline only) was NOT needed"
  - "killAndReap asserts exit code -1 (the harness's signal-kill Wait shape) — never confused with the SIGTERM contract's exit 0"
  - "execConvergeDeadline derived as 20x hello + 20x sync (6s) — generous-deadline discipline for loaded runners (a healthy pair converges in well under one second)"
  - "Node slots derived from the pair by node id (execPair.slots): a restart asks for its own slots and stays the SAME node by construction"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "Shared asymmetric two-node exec scaffold (private-range port picker, loud D-12 capability skip, shrunken timers, real-gRPC helpers, killAndReap SIGKILL exit shape)"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "all four TestExec_* scenarios run through newExecPair's probe on every execution — go test -race ./cmd/sup-kv/ -run '^TestExec_' -v PASS x4; 10/10 consecutive full-suite runs green"
        status: pass
      - kind: other
        ref: "grep: zero go:build lines in scenario_test.go; the one t.Skipf names the probed address, the wait, and the darwin lo0 reason"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestExec_killNineRestart — baseline bidirectional convergence, kill -9 node A (exit -1 asserted), B readable through the peer's death, A restarted on the same dump-path/ports converges both keys with no operator action"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "go test -race ./cmd/sup-kv/ -run '^TestExec_killNineRestart$' -v — PASS (0.36-0.66s), stable across -count=5 and 10 full-suite runs"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestExec_bothNodesDown — clean D-19 exits, one checksum bit corrupted in A's dump on disk, A restarted alone proves the surgical discard (stderr discard line + corrupted key gone + other key from disk), B's restart heals the record via replication"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "go test -race ./cmd/sup-kv/ -run '^TestExec_bothNodesDown$' -v — PASS (1.6-1.9s); zero time.Sleep in the file, every wait is a timer-derived waitFor"
        status: pass
    human_judgment: false
  - id: D4
    description: "TestExec_sigtermFinalDump — exit 0 after SIGTERM, the REPLICATED key present in the final dump on disk, and a lone restart (peer stopped first) serving it with no possible source but the final dump"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "go test -race ./cmd/sup-kv/ -run '^TestExec_sigtermFinalDump$' -v — PASS (1.3-1.6s)"
        status: pass
    human_judgment: false
  - id: D5
    description: "TestExec_emptyDiskRecovery — empty-disk node B against live seeded A recovers all 20 keys within the in-test derived bound N*floor(1s/rate)+margin, then participates fully (its write crosses to A)"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "go test -race ./cmd/sup-kv/ -run '^TestExec_emptyDiskRecovery$' -v — PASS (3.2s), measured recovery 116-195ms vs the 400ms paced bound; B started on a fresh t.TempDir dump dir"
        status: pass
      - kind: other
        ref: "go test -race ./... exit 0 and go tool golangci-lint run ./... exit 0 (plan-level verification)"
        status: pass
    human_judgment: false

duration: 26min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 07: Two-Process Exec Scenarios Summary

**TEST-04's exec half shipped: all four D-11 scenarios on the shipped binary over real sockets and real gRPC — kill -9 restart convergence, both-nodes-down restart healing a surgically corrupted dump record, SIGTERM's final confirmed dump proving replicated writes boot a lone node, and empty-disk recovery at the in-test derived sync_response_rate bound (measured 116-195ms against the 400ms paced bound) — plus a structural fix for the ephemeral-port race the stability loop exposed.**

## Performance

- **Duration:** 26min
- **Started:** 2026-11-15T23:26:42Z
- **Completed:** 2026-11-15T23:52:42Z (UTC)
- **Tasks:** 3/3
- **Files modified:** 2 (1 created, 1 comment-only touch)

## Accomplishments

- **TestExec_killNineRestart** (the tracer): two nodes converge a write each way on the asymmetric topology; node A dies to `Process.Kill` (exit -1 asserted — the harness's kill shape, never the SIGTERM contract); node B keeps serving both keys through the peer's death; A restarted with the same dump-path and ports converges both keys with no operator action beyond the restart. Tracer feedback gate re-ran the verify under auto mode: PASS before expansion.
- **TestExec_bothNodesDown**: both nodes exit cleanly (D-19), ONE record of A's dump gets a single checksum bit flipped on disk, and the restart proves both halves separately — A alone shows the discard line in its stderr, the corrupted key gone, every other key served from its own disk; B's restart heals the corrupted key back through replication alone.
- **TestExec_sigtermFinalDump**: a replicated write is provably in A's FINAL confirmed dump (exit 0 + on-disk grep), and a lone restart — B stopped first — serves it with no possible source but that dump.
- **TestExec_emptyDiskRecovery**: the bound is computed in-test from the flags (`20 * (1s/50) = 400ms` paced + derived margin), measured recovery ran 116-195ms, and B's own write replicates to A immediately after recovery. The flagged assumption A5 (timing assertion flakiness) did not materialize — the documented fallback was not needed.
- **Port-race hardening** (Rule 1, from the stability loop): scenario UDP ports now come from a private window below `ip_local_port_range` with a claim-then-release registry — 3 failures in 12 full-suite runs before, 10/10 green after.

## Task Commits

1. **Task 1 (tracer): kill -9 restart convergence + shared exec scaffold** — `0bb2984` (test)
2. **Task 2: both-nodes-down corrupted-record restart; SIGTERM final-dump** — `05958b0` (test)
3. **Task 3: empty-disk recovery bounded by sync_response_rate** — `f9c1bc2` (test)

**Plan metadata:** (this commit) `docs(06-07): complete two-process exec scenarios plan`

## Files Created/Modified

- `cmd/sup-kv/scenario_test.go` — NEW: private-range port picker + registry, capability probe (loud D-12 skip), execPair scaffold (slots by node id, shrunken timers hello 100ms / sync 200ms / gate 1s), execSet/execGet/execServing real-gRPC helpers, killAndReap, corruptDumpRecord, and the four TestExec_* scenarios
- `cmd/sup-kv/main_test.go` — one documented `//nolint:unparam` on `awaitServing` (comment-only; see deviations)

## Decisions Made

See key-decisions frontmatter. Highlights: the two-sided corrupted-record proof (alone-then-peer), the checksum-XOR-one-bit surgical corruption with the discard-line as its evidence, the lone-restart final-dump isolation, the in-test derived empty-disk bound, and the private-window port picker.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Lint blocker] unparam fired on the existing awaitServing helper once scenario call sites landed**
- **Found during:** Task 1, package lint gate
- **Issue:** Adding three `awaitServing` call sites (all passing `servingDeadline`, as every existing caller does) pushed the helper over unparam's reporting threshold: `timeout always receives servingDeadline`. The clean tree lints green — the finding is newly surfaced by this plan's callers, not pre-existing.
- **Fix:** The code-direction fix unparam suggests (remove the parameter) is barred by the plan's own rule (no changes to the existing harness helpers — it would rewrite every existing call site for zero behavior gain), so the sanctioned escape hatch applies: one documented nolint with the written justification inline. Every TestMain_*/TestLog_* test stays byte-unmodified.
- **Files modified:** cmd/sup-kv/main_test.go (one comment)
- **Verification:** go tool golangci-lint run ./cmd/sup-kv/... exit 0; all TestMain_* tests pass unmodified
- **Committed in:** 0bb2984

**2. [Rule 3 - Lint bridging] startExecNode's extra param temporarily nolinted**
- **Found during:** Task 2, package lint gate
- **Issue:** The scaffold's `extra ...string` (built for Task 3's `--sync-response-rate`) had no caller yet — unparam flagged it in Task 2's commit.
- **Fix:** The repo's 02-02 precedent: a written-reason bridging nolint, removed the moment the consuming call site landed (Task 3 passes the rate flag through it).
- **Files modified:** cmd/sup-kv/scenario_test.go
- **Verification:** lint green at both commits; the directive is gone from the final file
- **Committed in:** 05958b0 (added), f9c1bc2 (removed)

**3. [Rule 1 - Bug] Ephemeral-port pick-to-bind race made the parallel suite flaky**
- **Found during:** Task 2/3 stability loops (3 failures in 12 full-suite runs)
- **Issue:** `freeUDPPort`'s pick-close-rebind pattern races every other parallel test's `:0` binds: observed as the probe's rx bind failing EADDRINUSE, a scenario node dying at its UDP bind (10s serving timeout), and — via the added pool pressure — one existing test's node death and one existing test's probe bind failure (TestMain_precedence, TestLog_formatText: the same latent race the shared harness always had, materialized by the scenarios' added picks).
- **Fix:** Two layers. First (Task 2): the probe reports errors instead of failing and newExecPair retries on fresh ports. Then (Task 3, the structural fix): scenario UDP ports are picked from a private window [21000, 32678) strictly below the kernel's ephemeral range (32768 here) with a claim-then-release in-use registry held to test cleanup — no rival `:0` bind can ever be issued in the window, sibling scenarios cannot double-claim, and restarts keep their ports reserved. The scenarios leave the shared ephemeral pool entirely, returning the existing tests' exposure to its historical level (their own pickers are out of scope: harness files, plan-barred).
- **Files modified:** cmd/sup-kv/scenario_test.go
- **Verification:** 10/10 consecutive full-suite runs green (plus 2 earlier post-fix runs); 3 failures in the 12 pre-fix runs
- **Committed in:** 05958b0 (probe retry), f9c1bc2 (private-window picker)

---

**Total deviations:** 3 auto-fixed (1 Rule 1 bug, 2 Rule 3 lint blockers)
**Impact on plan:** None on scope — every must-have truth delivered; the scaffold is materially more reliable than the plan's minimum, and the only harness-file touch is one justified comment.

## Issues Encountered

The port-race investigation above (deviation 3). One earlier single suite failure occurred while lint compiled and 8 race-enabled processes ran concurrently on this heavily loaded machine — not reproduced in isolation, attributed to the same contention; the post-fix loops cover it. No auth gates (pure in-repo work).

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - every scenario asserts through real processes, real sockets, real gRPC, and real dump files; nothing is behind build tags or bare skips (the darwin skip is the one loud D-12 capability gate).

## Threat Flags

None. T-6-13 mitigated as planned: exit-code and dump assertions reuse assertExitCode and the lifecycle dump-grep precedent; TestExec_sigtermFinalDump pins exit 0 AND on-disk replicated-record containment together.

## Next Phase Readiness

- TEST-04's four exec scenarios join 06-06's real-socket in-process scenarios for phase verification; the plan-level suite (`go test -race ./...`), full-repo lint, and the whole cmd/sup-kv package are green together.
- The darwin CI cells (backstop, D-12): the four scenarios ride the capability probe — macos-15 expected green-with-loud-skip on the PR; observe there.
- 06-08's benchmarks can reuse the private-window port picker and the execPair topology pattern.

## Self-Check: PASSED

- key-files.created exists on disk: cmd/sup-kv/scenario_test.go FOUND
- Commits reachable from HEAD: 0bb2984, 05958b0, f9c1bc2 FOUND (merge-base --is-ancestor ok)
- evaluation-scope --plan 06-07 --commits-only: resolved, 3 commits
- All task acceptance criteria re-run: PASS (kill shape -1 asserted, not SIGTERM-confused; same dump-path/ports restart with timer-derived waitFor convergence; loud probe, zero build tags; TestMain_* unmodified and green; lint ./cmd/sup-kv/... 0 — surgical one-record corruption with loader-discard proof and peer-heal proof; exit 0 + replicated record in final dump + lone-restart service; zero fixed sleeps; in-test derived bound from N and rate; fresh empty dump dir; B's write crosses to A; four TestExec_* + all TestMain_* green together under -race)
- Plan-level verification: go test -race ./cmd/sup-kv/ -run '^TestExec_' -v all four PASS; whole cmd/sup-kv suite ok; go test -race ./... exit 0; go tool golangci-lint run ./... exit 0
- Windows ledger: no stubs, skipped tests, or unrun verifies to record

---
*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
