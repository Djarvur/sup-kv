---
phase: 02-stand-and-simulator-as-tests
plan: 02
subsystem: testing
tags: [stand, real-socket, exec-harness, two-process, f2c, test-05, linux-only, d03]

# Dependency graph
requires:
  - phase: 04-replication (v1.0 shipped)
    provides: the real cmd/sup-kv binary the stand builds once and drives (flags per main.go, SIGKILL/-1 and SIGTERM/0 exit contracts)
  - phase: v1.0 failure-scenario suite
    provides: the exec harness source this plan moved wholesale (main_test.go helpers, scenario_test.go scenarios)
provides:
  - internal/stand package: the two-process real-socket test harness (build-once binary, reserved port window + claim registry, three-outcome probe with not-reached = fatal on Linux, node lifecycle with the SIGKILL/SIGTERM exit contracts, deadline-clamped waits, asymmetric-port topology, serving/dump/log seams)
  - The four migrated TestStand_* scenarios (kill-nine-restart, both-nodes-down, sigterm-final-dump, empty-disk-recovery) plus the new TestStand_oversizeKeyF2c green pin, all green under -race on Linux with zero skips
  - The standNightly (SUPKV_STAND_NIGHTLY) scale-selection hook, declared for plan 11's nightly wiring
  - The platform every stand-level deliverable of the phase builds on (red tests 02-05, deletion levels, metrics-by-effect, nightly 10k)
affects: [02-03..02-10, 02-12 (stand-level deliverables consume the harness), 02-14 (darwin report completes TEST-05's second branch), 02-11 (scheduled.yml invokes the stand), phases 3-5 (the F2c pin flips with XPORT-03)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 21678    # chars/4 over the changed files (29246 stand.go + 20718 stand_test.go + 10701 broadcast_probe_test.go + 26049 deleted scenario_test.go = 86714)
  tasks: 2
  commits: 2      # MEASURED from the plan-start ledger at SUMMARY write (#3968)
  plan_head_before: 18d8a142db61b287e448d6899991b75567292935
  plan_head_after: 5f8629f66d63534fbd34e791b000004ac246a323

# Tech tracking
tech-stack:
  added: none — stdlib only (os/exec, net, math/rand/v2, grpc/apiproto already in go.mod); zero new module dependencies
  patterns:
    - two-process exec harness in a non-test package file (//go:build linux): build-once binary via sync.Once state, loud fatal on failed build (TEST-05)
    - reserved sub-ephemeral port window + mutex claim registry surviving the harness move (T-11 flake-class-1)
    - negative timed assertion through waitFor: the condition is "window elapsed AND still absent" so an early appearance fails loudly at the deadline instead of passing vacuously
    - justified nolints on test-infrastructure paths the machine rules cannot classify (depguard no-test-libs-in-prod, testpackage internal-tests pairing)

key-files:
  created:
    - internal/stand/stand.go
    - internal/stand/stand_test.go
  modified:
    - cmd/sup-kv/broadcast_probe_test.go
  deleted:
    - cmd/sup-kv/scenario_test.go

key-decisions:
  - "Harness + scenarios landed as ONE commit (91765fd): the strict lint gate's transitive unused analysis cannot reach the t-coupled harness helpers without their callers, so stand.go alone is un-committable — verified empirically before committing; the plan's Task 1/2 file split survives in content, only the commit boundary moved"
  - "internal/stand keeps the plan's file names AND its internal-test requirement (unexported helpers shared by scenarios) via a justified //nolint:testpackage on stand_test.go's package clause and //nolint:depguard on stand.go's testing import — both carry written reasons per AGENTS.md; the package doc states the test-infrastructure status the path-based rules cannot see (the testutil package documents its test* directory exemption the same way)"
  - "The port registry moved to broadcast_probe_test.go (its sole remaining cmd/sup-kv consumer) instead of being deleted: the Phase 7 quickstart-broadcast probe (DOCS-01) still uses scenarioUDPPort, so the plan's expected dead-helper removal set was empty and main_test.go needed zero removals"
  - "F2c assertions ride standAbsent (NotFound-based) rather than standServing: a value-mismatched record must fail the never-replicated window, not pass it vacuously; the oversize write uses standSetWith with the review's 1-byte value because the derived scenario value trips the store's 60 KiB limit before the datagram ceiling matters"
  - "Moved identifiers renamed to the stand namespace (execPair→standPair, execValue→standValue, keys exec/→stand/...) while bodies and timer-derived bounds stayed verbatim; scenarioUDPPort keeps its source name in both homes"

patterns-established:
  - "internal/stand file naming: non-test harness in stand.go; internal test files use the plain *_test.go names with //nolint:testpackage on the package clause (siblings 02-05 stand_red_test.go and 02-14 stand_nondarwin_test.go should follow this pattern)"
  - "Negative timed assertions: waitFor(t, window+slack, ..., cond = elapsed>=window && absent(...)) — the window must be part of the condition, never a bare sleep"
  - "Capability probes: three-outcome shape preserved (reached / not-reached / probe-error-with-retry); the not-reached outcome is t.Fatalf on Linux (TEST-05) — only outcome handling may differ from the source's skip"

patterns-established-note: See patterns-established above.

requirements-completed: [TEST-05, TEST-03]

coverage:
  - id: D1
    description: "internal/stand harness (stand.go): build-once binary (fatal not skip on failed build), reserved port window + claim registry, three-outcome probeLoopbackSend (not-reached = fatal on Linux), node lifecycle (launch/reap, SIGKILL -1, SIGTERM 0, assertExitCode), deadline-clamped waitFor, asymmetric-port topology (standPair/nodeArgs/slots), awaitServing/standClient/standSetWith/standGet/standServing/standAbsent seams, readFinalDump, corruptDumpRecord"
    requirement: TEST-05
    verification:
      - kind: unit
        ref: "go build ./internal/stand/ && go vet ./internal/stand/ → exit 0"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./internal/stand/... (and make lint repo-wide) → exit 0, zero issues"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ → ok (coverage 81.8% of statements) — every harness seam exercised by the five scenarios"
        status: pass
    human_judgment: false
  - id: D2
    description: "The four TestExec_* scenarios migrated as TestStand_killNineRestart, TestStand_bothNodesDown, TestStand_sigtermFinalDump, TestStand_emptyDiskRecovery — bodies and timer-derived bounds verbatim under the stand namespace; cmd/sup-kv/scenario_test.go deleted (zero duplication)"
    requirement: TEST-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_' → all five PASS, none skipped (killNineRestart 0.49s, bothNodesDown 1.77s, sigtermFinalDump 1.54s, emptyDiskRecovery 3.24s)"
        status: pass
      - kind: other
        ref: "test ! -f cmd/sup-kv/scenario_test.go → true; grep -rc 'func TestExec_' cmd/sup-kv/ → 0 remaining"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestStand_oversizeKeyF2c — the F2c green pin: 70 000-char key Set returns OK (1-byte value per the review's shape), sendto message-too-long failures polled live on A's stderr, the key stays absent from B over the derived window (2×sync_interval + gate + discovery + slack) via standAbsent, B's final dump (when it exists) lacks the key, A serves the key throughout"
    requirement: TEST-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_oversizeKeyF2c$' → PASS (3.81s; log line 'oversize key stayed unreplicated for the full 3.6s window')"
        status: pass
    human_judgment: false
  - id: D4
    description: "cmd/sup-kv shrink: scenario_test.go deleted; the single-process suite untouched and green; the port registry hosted by its remaining consumer (broadcast_probe_test.go) with updated cross-references"
    requirement: TEST-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ → ok 7.9s (the full single-process suite: TestMain_*, TestShutdown_*, TestLog_*, TestMetrics_*, TestMetricsTLS_*, TestDebugSurface, TestProbeQuickstartBroadcast)"
        status: pass
      - kind: other
        ref: "make lint → exit 0 (config verify + run ./...); make test (go test -race -cover ./...) → exit 0, all 12 packages ok"
        status: pass
    human_judgment: false
  - id: D5
    description: "TEST-05 Linux branch enforced: zero silent skips in the stand package — a capability-probe not-reached outcome is t.Fatalf at the newStandPair call site, a failed binary build is t.Fatalf in standBinary"
    requirement: TEST-05
    verification:
      - kind: other
        ref: "grep -rc 'Skipf' internal/stand/ → 0 matches; grep -c 't.Skip' internal/stand/ → 0 matches (the darwin branch is same-wave plan 02-14's named test)"
        status: pass
    human_judgment: false

# Metrics
duration: 34min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 2: internal/stand Harness + Migrated Scenarios Summary

**The two-process real-socket stand is live: `internal/stand` carries the harness moved wholesale from cmd/sup-kv (build-once binary, reserved port registry, fatal-on-Linux probe, SIGKILL/SIGTERM exit contracts, deadline-clamped waits, asymmetric-port topology) and runs the four migrated TestStand_* scenarios plus the new TestStand_oversizeKeyF2c pin green under `-race` with zero silent skips — TEST-05's Linux branch and TEST-03's process-level half now have their platform.**

## Performance

- **Duration:** 34 min (2040 s)
- **Started:** 2026-11-19T19:33:36Z
- **Completed:** 2026-11-19T20:07:36Z
- **Tasks:** 2 (landed as 2 commits; the harness and its scenarios share one commit — see Deviations)
- **Files changed:** 4 (2 created, 1 modified, 1 deleted; +1298/−686 lines)

## Accomplishments

- `internal/stand/stand.go` (`//go:build linux`, non-test file) — the two-process real-socket harness, moved wholesale from cmd/sup-kv's exec suite (D-03): the sync.Once-guarded build-once cmd/sup-kv builder (a failed build is a loud `t.Fatalf`, never a silent pass — TEST-05); the reserved sub-ephemeral port window (28100–32678) with the mutex claim registry, claim-while-bound then release with t.Cleanup (the T-11 flake-class-1 mitigation survives the move intact); the three-outcome `probeLoopbackSend` with its contract documented ("not-reached on Linux is a fatal at the call site — TEST-05; on non-Linux this file does not compile — stand_nondarwin_test.go, plan 02-14"); the node lifecycle (`launchNode`/`reap`, `killAndReap` asserting exit −1, `signalAndWaitExit` asserting exit 0, `assertExitCode`); `waitFor` with the test-deadline clamp (pollEvery 10 ms, hardDeadlineMargin) — every stand wait goes through it; the asymmetric-port topology (`standPair`, `newStandPair`, `nodeArgs` per main.go's flag names, `slots` mirror); the serving/dump/log seams (`awaitServing` fresh-client-per-poll, `standClient`/`standSetWith`/`standGet`/`standServing`/`standAbsent`, `readFinalDump`, `corruptDumpRecord`, `startStandLoggedNode` + `logBuffer`).
- `internal/stand/stand_test.go` — the four `TestExec_*` scenarios migrated as `TestStand_killNineRestart` (TEST-03's process-level acceptance: Set → OK, SIGKILL exit −1, restart, Get succeeds, dump reload path), `TestStand_bothNodesDown` (clean SIGTERM exits, one corrupted dump record, the discard line on A's live stderr, replication heals), `TestStand_sigtermFinalDump` (clean exit 0 with the final dump covering the replicated write, lone restart serves from disk), `TestStand_emptyDiskRecovery` (the F9 seed: 20 keys, paced bound N·(1s/rate) + gate + intervals + margin derived from the flags, never magic constants) — bodies and bounds verbatim under the stand namespace; plus the NEW `TestStand_oversizeKeyF2c` green pin: a 70 000-character key with the review's 1-byte value Sets OK, the sendto message-too-long failures appear on A's live stderr, and the key never reaches B (absent over the derived 2×sync_interval + margin window, absent from B's final dump, served on A throughout).
- `TestMain` wires the build-once binary eagerly (a failed build is reported loudly before any test runs), announces the `SUPKV_STAND_NIGHTLY=1` scale mode, and removes the build temp dir after the suite; `standNightly` is the declared scale-selection hook for plan 11.
- `cmd/sup-kv/scenario_test.go` deleted (git rm) — zero duplication left; the four scenarios exist only in `internal/stand`.
- `cmd/sup-kv/broadcast_probe_test.go` hosts the port-window constants, claim registry and `scenarioUDPPort` its Phase 7 quickstart-broadcast probe still uses, with the cross-reference comment updated to point at internal/stand's `probeLoopbackSend`.
- `cmd/sup-kv/main_test.go` untouched — the audit showed every harness helper it carries (buildSmokeBinary, launchNode, waitFor, awaitServing, …) is still used by the single-process tests, so the plan's "remove the dead helpers" set was empty.

## Task Commits

1. **Task 1 + the scenario file of Task 2: internal/stand harness + stand scenarios (moved not duplicated)** - `91765fd` (feat) — the harness and its callers land as one commit (lint-gate atomicity, see Deviations 1)
2. **Task 2: delete the cmd/sup-kv exec scenario suite; move the port registry to its remaining consumer** - `5f8629f` (test) — carries the intentional deletion of `cmd/sup-kv/scenario_test.go`

_Measured from the plan-start ledger `18d8a14` (`git rev-list --count 18d8a14..HEAD` at SUMMARY write): 2 task commits._

## Files Created/Modified

- `internal/stand/stand.go` — created (783 lines): the linux-tagged harness (non-test file; compiles under `go build ./internal/stand/`)
- `internal/stand/stand_test.go` — created (446 lines): TestMain, the standNightly hook, scenario constants and the five TestStand_* tests
- `cmd/sup-kv/broadcast_probe_test.go` — modified (+69/−1): hosts the port window + registry + scenarioUDPPort its probe still uses
- `cmd/sup-kv/scenario_test.go` — DELETED (684 lines): the wholesale-move source; intentional deletion documented here

## Decisions Made

- **One commit for harness + scenarios.** The strict lint gate's transitive `unused` analysis runs over the package with its test files and flags every t-coupled harness helper that has no caller — stand.go alone is un-committable (verified empirically with a scratch probe before writing the real files). The plan's Task 1/Task 2 file split survives in content; only the commit boundary moved.
- **Justified nolints instead of renames or exports.** The vendored `testpackage` linter requires external `foo_test` packages for `*_test.go` files (the repo's internal-test convention is the `*_internal_test.go` name), and the upstream `no-test-libs-in-prod` depguard rule denies `testing` imports by path — its test* directory exemptions cannot see a harness in `internal/stand`. Both carry written reasons per AGENTS.md's suppression rule, and the package doc states the test-infrastructure status (mirroring how `internal/app/internal/testutil` documents its directory exemption). Sibling plans 02-05/02-14 create files in this package — they should follow the same pattern (recorded in patterns-established).
- **The port registry moved, not deleted.** The plan expected it dead after the scenario deletion; `broadcast_probe_test.go` (Phase 7's DOCS-01 probe, out of this plan's scope to alter) still calls `scenarioUDPPort`. It now lives next to its remaining consumer — plan A6's "residual overlap is deduplicated in review" accepted: the registry exists in internal/stand (moved) and cmd/sup-kv (still used).
- **F2c assertions use absence and raw values.** `standAbsent` (NotFound-based) replaces `standServing` in the never-replicated window so a value-mismatched record fails the assertion rather than passing it vacuously; `standSetWith` carries the review's 1-byte value because the derived scenario value (70 KB) trips the store's value limit before the datagram ceiling is ever reached.
- **stand namespace rename.** Moved identifiers renamed (execPair→standPair, execValue→standValue, startExecNode→startStandNode, keys `exec/…`→`stand/…`) while bodies and timer-derived bounds stayed verbatim; `scenarioUDPPort` keeps its source name in both homes.

## Deviations from Plan

**1. [Rule 3 - Blocker] Task 1 and Task 2's stand_test.go share one commit**
- **Found during:** Task 1 verification
- **Issue:** `go tool golangci-lint run ./internal/stand/...` fails on a standalone stand.go — the `unused` linter transitively flags every t-coupled harness helper (no callers exist before stand_test.go lands), so the plan's Task 1 commit cannot pass its own lint verify
- **Fix:** committed stand.go + stand_test.go together as `91765fd`; Task 2's commit `5f8629f` carries the cmd/sup-kv shrink
- **Files modified:** commit boundary only (content per plan)
- **Verification:** both commits lint-clean; whole-repo `make lint` + `make test` green at HEAD
- **Commit:** `91765fd`

**2. [Rule 3 - Blocker] testpackage lint rule vs the plan's internal-test package clause**
- **Found during:** Task 1 verification (empirical probe)
- **Issue:** the vendored strict config's `testpackage` linter requires `stand_test.go` to declare `package stand_test`; the plan requires `package stand` (internal tests sharing the harness's unexported helpers) — the plan's filename + package combination is un-lintable without a suppression
- **Fix:** `package stand //nolint:testpackage // …` with the written reason (the D-03 internal-test pairing the rule cannot express); the repo's alternative convention (`*_internal_test.go` naming) was rejected to keep the plan's mandated file names (must_haves artifacts, sibling-plan references to stand_test.go)
- **Files modified:** internal/stand/stand_test.go (package clause)
- **Verification:** lint clean; the nolint is used (testpackage fires without it — probe-verified)
- **Commit:** `91765fd`

**3. [Rule 3 - Blocker] depguard no-test-libs-in-prod vs the testing import in stand.go**
- **Found during:** Task 1 verification (empirical probe)
- **Issue:** the upstream depguard rule denies `testing` in non-test files outside test* directories; `internal/stand/stand.go` is a non-test file by plan (so `go build ./internal/stand/` compiles the harness) whose helpers take `*testing.T` by design
- **Fix:** `//nolint:depguard // …` on the import with the written reason; the package doc explains the situation the same way the testutil package doc explains its directory exemption
- **Files modified:** internal/stand/stand.go (import line, package doc)
- **Verification:** lint clean
- **Commit:** `91765fd`

**4. [Rule 3 - Blocker] The port registry was not dead — moved to its remaining consumer instead of deleted**
- **Found during:** Task 2 (pre-deletion usage audit)
- **Issue:** the plan's "remove the harness helpers that became dead (execPair/port-registry/probe)" assumed the port registry dies with scenario_test.go; `cmd/sup-kv/broadcast_probe_test.go` (Phase 7's DOCS-01 quickstart-broadcast probe) still calls `scenarioUDPPort`, and main_test.go's helpers are all still used — the dead set was empty
- **Fix:** scenarioUDPPort + the window constants + the claim registry moved into broadcast_probe_test.go (sole remaining consumer); main_test.go left untouched; stale cross-reference comment updated to internal/stand's probeLoopbackSend
- **Files modified:** cmd/sup-kv/broadcast_probe_test.go; cmd/sup-kv/main_test.go unchanged (deviation from the plan's letter that anticipated touching it)
- **Verification:** `go test -race -count=1 ./cmd/sup-kv/` green including TestProbeQuickstartBroadcast
- **Commit:** `5f8629f`

**Total deviations:** 4 (all Rule 3 blockers auto-fixed without scope change)
**Impact:** content identical to the plan's intent; commit boundary, package-clause suppressions, and the registry's new home are the deltas. The five success criteria all hold.

## Issues Encountered

- **F2c's Set was rejected with InvalidArgument (value exceeds the maximum allowed size)** — the first draft reused the derived scenario value, making the 70 000-char key's value 70 021 bytes and tripping the store's value limit before the datagram ceiling mattered. Fixed with `standSetWith` carrying the review's 1-byte value; the pin now matches the review's measured shape. (Self-caught on first run, fixed within Task 1's commit.)
- **Nil-map panic in the port registry** — the struct-wrapping refactor of the registry (for the gochecknoglobals suppression) dropped the `make(map[int]bool)` initialization; fixed with a composite-literal initializer. (Self-caught on first run.)
- **Data race in F2c's stderr poll** — node A was started with a plain bytes.Buffer while the test polled its live stderr; fixed by using `startStandLoggedNode` (the logBuffer seam the harness already provides). The race detector is now clean across the suite. (Self-caught on first run.)
- **B's final dump may not exist** — an empty store never persists anything, so F2c's disk-level assertion applies only when the final dump exists (no-dump is consistent with B never having applied anything); the Get-based window remains the primary never-replicated assertion.
- **Formatter-driven nolint placement churn** — golines wrapped two long nolint-carrying lines, detaching the directives from their issue lines; fixed by shortening the reasons (nolintlint then caught one genuinely unneeded gochecknoglobals directive in the test-only cmd/sup-kv package — removed).

## Known Stubs

- `standNightly` (`internal/stand/stand_test.go`) is a declared-but-unwired scale-selection hook: `SUPKV_STAND_NIGHTLY=1` selects nothing yet beyond the TestMain mode announcement. Intentional per the plan ("standNightly selects nothing yet (full-scale selection is wired by plan 11)"); plan 02-11 wires the full documented scale for the nightly run. Not recorded in WINDOWS.md — the phase's own plan set (02-11) is the tracking ledger for this wiring.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The stand platform is ready for its consumers: plan 02-05's stand-level red tests (`stand_red_test.go`, D-01 gate on `SUPKV_RED_TESTS=1`) extend this package and use the harness helpers; the deletion-level and metrics-by-effect deliverables (D-08, TEST-12) ride the same seams; plan 02-11's scheduled.yml invokes `go test ./internal/stand/` for the nightly 10k run with `SUPKV_STAND_NIGHTLY=1`.
- The red-runner's package list already carries the `./internal/stand/` placeholder (02-01) — uncomment it when the first stand-level red test lands (02-05).
- TEST-05's second branch (the loud darwin report, `stand_nondarwin_test.go` under `//go:build !linux`) is same-wave sibling plan 02-14; the package doc and probe contract already reference it.
- No blockers. The F2c defect remains unfixed by design (phase boundary: tests only); XPORT-03 fixes it in Phase 3 and the pin flips with the fix.

## Self-Check: PASSED

Re-run on the committed tree (`5f8629f`):

- Key files exist: `internal/stand/stand.go`, `internal/stand/stand_test.go` — FOUND; `cmd/sup-kv/scenario_test.go` — CONFIRMED DELETED.
- Commits exist: `91765fd`, `5f8629f` — both ancestors of HEAD.
- `node $HOME/.zcode/gsd-core/bin/gsd-tools.cjs check evaluation-scope --plan 02-02 --commits-only --raw` → resolved, 2 commits, exit 0 (not 69).
- `go test -race -count=1 ./internal/stand/ -run '^TestStand_'` → ok 6.6s; all five scenarios PASS, none skipped.
- `go test -race -count=1 ./cmd/sup-kv/` → ok 7.9s.
- `grep -rc 'Skipf' internal/stand/` → 0 matches; `grep -rc 't.Skip' internal/stand/` → 0 matches (TEST-05 Linux branch).
- `make lint` (config verify + `run ./...`) → exit 0. `make test` (`go test -race -cover ./...`) → exit 0, all 12 packages ok (internal/stand coverage 81.8%).
- `test ! -f cmd/sup-kv/scenario_test.go` → true; `grep -rc 'func TestExec_' cmd/sup-kv/ | grep -v ':0' | wc -l` → 0.

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-19*
