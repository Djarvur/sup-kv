---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 02
subsystem: binary-ci-storage-hardening
tags: [signals, graceful-shutdown, ci-config, storage-docs, review-fixes, exec-testing]

requires:
  - phase: 05-grpc-api-and-binary
    provides: cmd/sup-kv serve/shutdown D-19 sequence, exit-code contract, exec smoke harness (startNode/reap/assertExitCode)
  - phase: 01-ci-bootstrap-and-repo-skeleton
    provides: ci.yml lint/gate jobs with the check-run contexts that are a ruleset contract
provides:
  - serve() holds SIGTERM/SIGINT ownership across the whole D-19 shutdown; a second signal during the GracefulStop drain takes the force path with the final Flush preserved (exit 0 + confirmed dump)
  - TestShutdown_secondSignal — the exec-level regression pinning the second-signal contract through a real process (the same contract exec scenario 3 in 06-07 asserts end-to-end)
  - ci.yml jobs bounded (timeout-minutes 15 lint / 20 gate) with zero check-run context renames
  - Corrected OnChange doc (tombstones fire the tap like any change; Get sees only live records) and the stated re-entrancy deadlock boundary
  - The duplicated store subtest removed; clean 02 review ledger (open: 0)
affects: [06-07 exec scenarios, TEST-04 suite, phase verification]

actuals:
  tokens: 4700    # chars/4 over the realized diff (18765 diff chars across 7 files)
  tasks: 2
  commits: 2      # MEASURED: git rev-list --count 03f2173..HEAD
plan_head_before: 03f2173577ccd715d4d219bdc17b28ecd528190f
plan_head_after: ba0d984fbbd69ee5080eca67a03f926c70e4699a

tech-stack:
  added: []
  patterns:
    - "signal ownership spanning serve AND shutdown via one armed buffered channel — the release (signal.Stop) runs after the final dump, never inside serve"
    - "the standard grpc health Watch stream as the exec-test drain blocker: it is not tied to apiCancel (unlike SupKV Watch), so GracefulStop observably waits on it; its NOT_SERVING update proves the D-19 sequence started"

key-files:
  created: []
  modified:
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - .github/workflows/ci.yml
    - internal/app/storage/store.go
    - internal/app/storage/store_test.go
    - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-REVIEW-DISPOSITION.md
    - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-REVIEW-DISPOSITION.md

key-decisions:
  - "P1-IN-01: one armed signal channel (armSignals, buffer 2) spans serving and shutdown — serve consumes the first signal, shutdown selects the second as its force branch, signal.Stop runs in main only after shutdown returns (Open Question 4 recommendation b: a second signal is an operator's hurry-up)"
  - "The final Flush sits after GracefulStop in the D-19 sequence and therefore survives the force stop — the confirmed dump still lands and the node exits 0"
  - "TestShutdown_secondSignal blocks the drain with a standard health Watch stream (not SupKV Watch, which apiCancel ends before GracefulStop by design) and observes NOT_SERVING on it as proof the drain is in progress when the second SIGTERM lands"
  - "Mutation check: re-moving signal.Stop to the pre-fix position reproduces the exact regression (signal kill, want exit 0) — the test binds on the fix, not on timing luck"
  - "P2-IN-03: the TestStore_Set copy of the duplicated subtest deleted, the TestStore_endToEnd copy kept byte-identical (the finding's own recommendation — endToEnd additionally asserts the unpacked (l, counter, nodeID))"
  - "cmpSame kept: it remains live in merge_test.go (package-shared constant), so the dedup removes no constant"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "Second-signal shutdown contract — signal ownership spans the whole D-19 sequence; a second SIGTERM during the blocked GracefulStop drain takes the force path, the final dump still contains the confirmed write, exit 0, no panic"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestShutdown_secondSignal (exec-level, real process; mutation-checked RED against the pre-fix signal release: 'process exit code = -1 (signal: terminated), want 0')"
        status: pass
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestMain_lifecycle green unmodified (single-signal semantics unchanged); full cmd suite under -race, 3x repeat"
        status: pass
    human_judgment: false
  - id: D2
    description: "CI jobs bounded — timeout-minutes 15 (lint) and 20 (gate), additive keys only, every check-run context byte-identical (ruleset required_status_checks contract)"
    verification:
      - kind: other
        ref: "structural gates: grep -c 'timeout-minutes:' ci.yml >= 2; fixed-string greps for 'name: gate (${{ matrix.name }})' and both matrix cell names; git hunk shows only the two added timeout lines (no name: diffs)"
        status: pass
    human_judgment: false
  - id: D3
    description: "OnChange doc accuracy — tombstone installs fire the tap exactly like any other change while a subsequent Get answers ErrNotFound (the record is a Records()-snapshot observable); re-entrancy boundary stated (synchronous tap under the store's locking discipline; even Get self-deadlocks against the held write lock)"
    verification:
      - kind: other
        ref: "git diff on store.go proven comment-only; phrase checks: 'Records() snapshot holds' and 'self-deadlocks' present; storage suite green under -race"
        status: pass
    human_judgment: false
  - id: D4
    description: "Duplicated store subtest removed — exactly one copy of 'a second set replaces the value under a strictly greater version' remains (TestStore_endToEnd), no assertion coverage lost"
    verification:
      - kind: unit
        ref: "grep count of the subtest name in store_test.go == 1; go test -race ./internal/app/storage/ green; full go test -race ./... green"
        status: pass
    human_judgment: false
  - id: D5
    description: "Disposition ledgers flipped in the same commits as their fixes (01: P1-IN-01 + P1-WR-01; 02: P2-IN-01/IN-02/IN-03), each row naming its fix and regression"
    verification:
      - kind: other
        ref: "git show --stat 69d1e47 ba0d984 (disposition files in each fix commit); grep '| fixed |' counts: 01 = 6, 02 = 3"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 02: Hardening Backlog Part 2 — Binary/CI/Storage Summary

**serve() now holds SIGTERM/SIGINT ownership across the whole D-19 shutdown — a second signal during the blocked GracefulStop drain takes the force path with the final confirmed dump still landing (exit 0), pinned end-to-end by the exec-level TestShutdown_secondSignal (mutation-checked RED against the pre-fix kill); ci.yml jobs bounded without renaming a single check-run context; OnChange doc corrected on tombstones and the re-entrancy boundary; the duplicated store subtest removed — 01/02 disposition rows flipped, whole-repo race+lint green.**

## Performance

- **Duration:** 9min (495s)
- **Started:** 2026-11-15T22:24:00Z
- **Completed:** 2026-11-15T22:32Z (UTC)
- **Tasks:** 2/2
- **Files modified:** 7

## Accomplishments

- **P1-IN-01 (Warning-class)**: `armSignals()` owns one buffered SIGTERM/SIGINT channel for serving AND the D-19 shutdown; `serve` consumes the first signal, `shutdown`'s GracefulStop select gains the second-signal case taking the existing force path (`gs.Stop()`), and `signal.Stop` runs in main only after shutdown completes — no repeated signal can kill the process mid-shutdown and forfeit the final dump (exit 143 eliminated).
- **TestShutdown_secondSignal** (exec harness, real binary): a standard health Watch stream — not tied to apiCancel, unlike SupKV Watch — holds the graceful drain observably open; NOT_SERVING observed on the stream proves the D-19 sequence started before the second SIGTERM lands; asserts exit 0 + final dump containment + no panic. Mutation check: moving `signal.Stop` back to the pre-fix position fails the test with the exact regression symptom (`signal: terminated, want 0`).
- **P1-WR-01**: `timeout-minutes: 15` (lint) / `20` (gate) — additive keys only; the hunk contains zero `name:` diffs, preserving the `lint` / `gate (<cell>)` check-run contexts the ruleset copies verbatim.
- **P2-IN-01/IN-02**: OnChange doc now names both surfaces (the `Records()` snapshot record; `Get` returns it only when live — a tombstone fires the tap like any other change while `Get` answers `ErrNotFound`) and states the re-entrancy boundary precisely (synchronous tap under the store's locking discipline; `sync.RWMutex` is not reentrant — even `Get` self-deadlocks).
- **P2-IN-03**: the `TestStore_Set` copy of the duplicated subtest deleted; the `TestStore_endToEnd` copy kept byte-identical per the finding's recommendation; `TestStore_Set`'s doc comment updated to point at the endToEnd pin.

## Task Commits

1. **Task 1 (tracer): second-signal shutdown fix** — `69d1e47` (fix) — tracer feedback gate re-ran verify end-to-end under auto mode (3x): PASS
2. **Task 2: CI bounds + OnChange docs + subtest dedup** — `ba0d984` (fix)

**Plan metadata:** (this commit) `docs(06-02): complete hardening backlog part 2 plan`

## Files Created/Modified

- `cmd/sup-kv/main.go` — armSignals/serve/shutdown restructure; signalBuffer const; second-signal force branch; updated serve/shutdown semantics docs
- `cmd/sup-kv/main_test.go` — TestShutdown_secondSignal (health-Watch drain blocker, status-observation helper, WaitForReady dial)
- `.github/workflows/ci.yml` — timeout-minutes on jobs lint (15) and gate (20)
- `internal/app/storage/store.go` — OnChange doc-comment corrections (comments only)
- `internal/app/storage/store_test.go` — duplicated subtest removed; TestStore_Set doc updated
- `01-REVIEW-DISPOSITION.md` — P1-IN-01, P1-WR-01 → fixed (open: 8 → 6)
- `02-REVIEW-DISPOSITION.md` — P2-IN-01/IN-02/IN-03 → fixed (open: 3 → 0)

## Decisions Made

See key-decisions frontmatter; all six are D-15 fix-shape decisions recorded per finding.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Lint] Lint shapes in the new code (mnd, revive, unparam)**
- **Found during:** Task 1 lint gate
- **Issue:** mnd flagged the literal channel buffer `2`; revive flagged a no-arg `log.Printf`; unparam fired on `awaitServing` only once my test added a 4th same-constant call site (the pristine 3-site state is silent — verified by linting against the pre-change main_test.go).
- **Fix:** Named `signalBuffer` const with doc; `log.Print` for the fixed string; dropped my test's `awaitServing` call because `waitForStatus(SERVING)` on the health Watch stream subsumes it (proves gate open AND stream liveness) — `awaitServing` returns to its pristine 3-caller shape.
- **Files modified:** cmd/sup-kv/main.go, cmd/sup-kv/main_test.go
- **Verification:** go tool golangci-lint run ./cmd/sup-kv/... exit 0; tests green 3x
- **Committed in:** 69d1e47

**2. [Rule 3 - Blocker] Health Watch dial raced node startup (connection refused)**
- **Found during:** Task 1 verification after dropping awaitServing
- **Issue:** Subscribing the health Watch immediately after startNode can precede the gRPC listener's readiness; the RPC failed fast with Unavailable (awaitServing previously covered this).
- **Fix:** `grpc.WaitForReady(true)` on the Watch call — the stream waits for the connection instead of failing fast; no polling loop, no error path.
- **Files modified:** cmd/sup-kv/main_test.go
- **Verification:** TestShutdown_secondSignal green 3x under -race (incl. the WaitForReady dial wait)
- **Committed in:** 69d1e47

**3. [Rule 1 - Bug] cmpSame deletion broke the storage package build**
- **Found during:** Task 2 storage suite run
- **Issue:** My first dedup also removed the `cmpSame` constant, assuming the deleted subtest was its last user — but merge_test.go (same test package) still uses it twice.
- **Fix:** Restored `cmpSame`; the dedup removes only the subtest.
- **Files modified:** internal/app/storage/store_test.go
- **Verification:** go test -race ./internal/app/storage/ green
- **Committed in:** ba0d984

---

**Total deviations:** 3 auto-fixed (2 Rule 1 lint/bug, 1 Rule 3 blocker; none architectural — no Rule 4 checkpoints)
**Impact on plan:** None on scope — all five findings landed as specified; the tracer's test needed two mechanical refinements (lint-shape and dial-readiness), both inside the task's own footprint.

## Issues Encountered

None beyond the deviations above. No auth gates (pure in-repo work).

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — the CI-runner behavioral half of P1-WR-01 is by design observable only on the runners (must_haves marks it verification: backstop); the local structural gates and the four matrix cells on this branch's PRs carry it.

## Threat Flags

None — no new security surface: the signal-path change narrows a Tampering window (T-6-03: exit-143 strandings eliminated), and the CI bounds are a low-severity DoS mitigation (T-6-04), both exactly as the plan's threat register prescribed.

## Next Phase Readiness

- The second-signal/durability contract 06-07's exec scenario 3 asserts end-to-end is now fixed and pinned by TestShutdown_secondSignal — scenario 3 extends this harness rather than rediscovering the bug.
- CI job bounds are in place before the new exec scenarios (06-07) run on the gate matrix — a hung scenario now fails its cell instead of holding a runner.
- 01-REVIEW-DISPOSITION open items are down to 6 (all info-level, none in D-15 scope); 02's ledger is fully closed (open: 0).

## Self-Check: PASSED

- key-files all exist on disk (main.go, main_test.go, ci.yml, store.go, store_test.go, both disposition files): FOUND
- Commits reachable from HEAD: 69d1e47, ba0d984 FOUND (merge-base --is-ancestor ok)
- evaluation-scope --plan 06-02 --commits-only: resolved (2 commits)
- All task acceptance criteria re-run: PASS (restructure in place, test asserts exit 0 + dump + no panic; TestMain_lifecycle unmodified and green; disposition rows fixed naming the test; lint scopes exit 0; ci.yml hunk shows only the two timeout additions; store.go diff comment-only; subtest count == 1; storage suite + full race suite + full lint exit 0)
- Plan-level verification re-run: go test -race ./cmd/sup-kv/ -run '^TestShutdown_secondSignal$|^TestMain_' -v green (4 tests); go test -race ./... green; go tool golangci-lint run ./... green
- Windows ledger: no stubs, skipped tests, or unrun verifies to record

---
*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
