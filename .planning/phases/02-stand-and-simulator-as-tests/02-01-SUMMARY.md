---
phase: 02-stand-and-simulator-as-tests
plan: 01
subsystem: testing
tags: [red-tests, ci-gating, go-test-json, startup-gate, grpc, d01, tracer]

# Dependency graph
requires:
  - phase: 01-spec-reference-and-decisions
    provides: ADR 0005 (decision 14.5 — interface closed until the full key list), SPEC-DEVIATIONS registry, the committed review evidence the red tests pin
provides:
  - The D-01 red-gating mechanism end to end: scripts/red-runner (go test -json event parser + expectation-table assertion + evidence capture), scripts/red-expectations.tsv, make red-run
  - The first Critical red test: TestRed_R03_gateClosedUntilFullList (gate-level PROTO-01 acceptance, env-gated on SUPKV_RED_TESTS=1)
  - First red-run evidence under .planning/phases/02-stand-and-simulator-as-tests/red-evidence/
  - The flip contract for Phases 3-5: a one-line expectation-table row change (red -> green)
affects: [02-02..02-17 (more red tests extend the runner's package list and the table), phases 3-5 (flip rows as fixes land), 02-11 (scheduled.yml invokes the red-run job)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 26086    # chars/4 over all changed files (source deliverables ~5k, committed evidence JSON ~21k)
  tasks: 3
  commits: 3      # MEASURED from the plan-start ledger at SUMMARY write (#3968); the plan metadata docs commit adds one more
  plan_head_before: 24589c5301803ebbfd81040142d2989bc92a1e6a
  plan_head_after: ba6c5c8632e651e6e06b78b925d0c4d4ed671040

# Tech tracking
tech-stack:
  added: none — bash + python3 stdlib json + `go test -json` events; zero new module dependencies (zero-install phase per the Package Legitimacy Audit)
  patterns:
    - env-gated expected-red tests (D-01): body's first statement gates on SUPKV_RED_TESTS=1; normal gates stay green by construction
    - expectation-table contract: runner asserts every row from parsed -json events; the Phase 3-5 flip is a one-line table change
    - append-only evidence capture: red-evidence/<UTC ts>/ with raw go-test.json + parsed name->action summary, never overwritten

key-files:
  created:
    - internal/grpcapi/red_gate_test.go
    - scripts/red-runner
    - scripts/red-expectations.tsv
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261119T192146Z/
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261119T192248Z/
  modified:
    - Makefile

key-decisions:
  - "The SyncArrived fake bakes the v1.0 mirror semantics (any received DataSync datagram counts, gate.go A1) so the red IS the R-03 defect itself; the closed-side assertions run while the resolver's first poll is parked on a release channel (the TestGate_opensOnTimeout determinism pattern), which makes the Phase 4 flip a zero-test-edit event — phase 1 asserts the closed interface (ADR 0005: reads and writes alike) deterministically, phase 2 flips to the full listing and asserts the open half"
  - "Runner package list is hardcoded with a comment to extend (./internal/grpcapi/ first): the TSV format carries no package column, so deriving packages from table rows would need a second mapping file"
  - "An unexpected failing test (name not an expected-red row) is a runner error: a broken normal test can never hide behind the expected reds (T-2-01 'unknown/missing names are errors' applied to the event stream, subtest names of table rows counted as their parent's)"
  - "Runner never trusts the go test exit code and never greps human output: verdicts come only from Action/Test fields of the JSON events of the exact command the script spawned (Pitfall 6 + T-2-01/T-2-02 mitigations)"

patterns-established:
  - "Red-test gating (D-01): every TestRed_* body begins with the env gate check; scripts/red-runner asserts expected-red names FAIL via go test -json events; a skip for an expected-red name is an error (a closed env gate is never a pass)"
  - "Evidence capture: every red run writes .planning/phases/02-stand-and-simulator-as-tests/red-evidence/<UTC ts>/{go-test.json,summary.txt}; never overwrites an existing directory"
  - "Flip contract: Phases 3-5 change only scripts/red-expectations.tsv rows (red -> green), never the runner script, never the Makefile, never the test bodies"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "TestRed_R03_gateClosedUntilFullList — gate-level PROTO-01 acceptance (ADR 0005 / decision 14.5): interface closed while only a partial DataSync listing has arrived (Get/Set/Delete Unavailable + health NOT_SERVING), open after the full listing (Get serves the record, health SERVING); FAILS against v1.0 code with SUPKV_RED_TESTS=1, SKIPs without it"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "internal/grpcapi/red_gate_test.go#TestRed_R03_gateClosedUntilFullList (SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$' → exit 1 with 'want code Unavailable, got OK' — the red verdict is the deliverable)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$' (no env var) → SKIP, exit 0 — make test stays green"
        status: pass
    human_judgment: false
  - id: D2
    description: "scripts/red-runner + scripts/red-expectations.tsv — the D-01 expected-red gate: parses go test -json events (Action/Test only), asserts every table row (red rows must FAIL; skip/absence/green-row fail/unexpected failing test/malformed row are errors), records evidence per run"
    requirement: TEST-04
    verification:
      - kind: other
        ref: "bash scripts/red-runner → exit 0; evidence <ts>/go-test.json carries Action=fail events for TestRed_R03_gateClosedUntilFullList; summary.txt row 'TestRed_R03_gateClosedUntilFullList red fail,run ... OK'"
        status: pass
      - kind: other
        ref: "negative paths via scratch copies (real script untouched): closed env gate → 'expected red, got skip' exit 1; row flipped red->green while the test still fails → exit 1"
        status: pass
      - kind: other
        ref: "grep -v '^#' scripts/red-expectations.tsv | grep -c 'TestRed_R03_gateClosedUntilFullList<TAB>red' → 1"
        status: pass
    human_judgment: false
  - id: D3
    description: "make red-run — the Makefile wiring of the expected-red gate (.PHONY + target next to bench); the flip-to-green never touches the Makefile"
    requirement: TEST-04
    verification:
      - kind: other
        ref: "make red-run → exit 0, FAIL record for TestRed_R03_gateClosedUntilFullList in the evidence summary (red-evidence/20261119T192248Z)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Normal gates stay green by construction on the committed tree — red tests skip without the env var, so lint and the race suite never run their bodies"
    requirement: TEST-04
    verification:
      - kind: other
        ref: "make lint → exit 0 (config verify + run ./...); make test (go test -race -cover ./...) → exit 0, all packages ok"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 1: First Red Test + Red-Gating Mechanism Summary

**The D-01 red-gating mechanism proven end to end: TestRed_R03_gateClosedUntilFullList fails against v1.0 code under SUPKV_RED_TESTS=1, scripts/red-runner asserts it from go test -json events against the expectation table and captures evidence, and make test/lint stay green because the test skips without the gate.**

## Performance

- **Duration:** 9 min (520 s)
- **Started:** 2026-11-19T19:15:49Z
- **Completed:** 2026-11-19T19:26:09Z (self-check re-run extended to ~19:26)
- **Tasks:** 3
- **Files modified:** 8 (4 deliverables + 4 committed evidence files)

## Accomplishments

- `internal/grpcapi/red_gate_test.go` — `TestRed_R03_gateClosedUntilFullList`, the phase's first Critical red test: env-gated on `SUPKV_RED_TESTS=1` as its first statement (D-01), in-process through the existing gate harness (`newGateServer`, `assertCode`, `awaitGateOpen`, `gateTestNeverBackstop`). It asserts the full observable gate-level PROTO-01 acceptance (ADR 0005 / decision 14.5): closed while only a partial DataSync listing has arrived — `Get`/`Set`/`Delete` all `Unavailable`, health `NOT_SERVING` — and open after the full listing arrives — `Get` serves the record, health `SERVING`. Against v1.0 code it fails exactly on the R-03 defect (the resolver opens on any first chunk; `Get` serves reads while the interface must be closed); without the env var it skips, so `make test` stays green.
- `scripts/red-runner` — the expected-red gate: exports `SUPKV_RED_TESTS=1` itself (a closed env gate can never masquerade as a pass — a skip for an expected-red name is an error), runs the red-test packages under `go test -race -count=1 -json`, does not trust the exit code, parses only `Action`/`Test`/`Package` from the events with python3 stdlib json (no text greps, no eval — Pitfall 6 and T-2-01/T-2-02 mitigations), and asserts every `scripts/red-expectations.tsv` row: red rows must have a fail event; skip, absence, a green-row fail, an unexpected failing test, or a malformed row are errors. Every run records `go-test.json` + a parsed summary under `red-evidence/<UTC ts>/`, never overwriting a directory.
- `scripts/red-expectations.tsv` — the expectation table: header comments carrying the provenance (REVIEW.md §5 R-03, §15 composition) and the flip contract, plus the single data row `TestRed_R03_gateClosedUntilFullList` → `red`. The Phase 4 flip is a one-line change to this file.
- `Makefile` — `red-run` target (+ `.PHONY`) invoking `bash scripts/red-runner`, placed next to `bench`; the `test` target untouched.
- Negative paths verified via scratch copies only (the real script untouched): closed env gate → exit 1 on the skip error; row flipped to `green` while the test still fails → exit 1 (the flip semantics work as D-01 specifies).
- Two red-run evidence captures committed under `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/` (`20261119T192146Z` from the runner verify, `20261119T192248Z` from `make red-run`), each carrying the raw `go test -json` stream with `Action:"fail"` events for the R-03 test and the parsed name→action summary.

## Task Commits

Each task was committed atomically:

1. **Task 1: First red test — R-03 gate stays closed until the full key list** - `3463631` (test)
2. **Task 2: red-runner script + expectation table + evidence capture** - `bee7369` (feat)
3. **Task 3: Makefile red-run target + wire-up** - `ba6c5c8` (chore)

**Plan metadata:** the `docs(02-01)` commit following this SUMMARY (carries SUMMARY + STATE/ROADMAP/REQUIREMENTS updates).

_Measured from the plan-start ledger `24589c5` (`git rev-list --count 24589c5..HEAD` at SUMMARY write): 3 task commits._

## Files Created/Modified

- `internal/grpcapi/red_gate_test.go` — the R-03 gate-level red test (env-gated, full PROTO-01 acceptance, reuses the existing gate-test harness)
- `scripts/red-runner` — the D-01 expected-red gate (executable in git, 100755): `-json` event parsing, expectation-table assertion, evidence capture
- `scripts/red-expectations.tsv` — the expectation table (one data row: R-03 = red; the Phase 4 flip target)
- `Makefile` — `red-run` target + `.PHONY` entry, next to `bench`
- `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/{20261119T192146Z,20261119T192248Z}/` — committed red-run evidence (raw JSON + parsed summary)

## Decisions Made

- The `SyncArrived` fake models the v1.0 mirror semantics ("any received DataSync datagram counts", gate.go A1) so the red verdict demonstrates the actual early-open defect; the closed-side assertions run while the resolver's first poll is parked on a release channel (the `TestGate_opensOnTimeout` determinism pattern), making them race-free under parallel load AND making the Phase 4 flip a zero-test-edit event: once Phase 4 gates reads (ADR 0005 mandates the whole interface closed), the same fake sequence yields closed-while-partial → open-after-complete with every assertion unchanged.
- Runner package list hardcoded with an extend comment (the TSV has no package column; a derived list would need a second mapping file).
- Unexpected failing tests are runner errors (subtest names of table rows counted as their parent's) — hardening beyond the plan's letter, implementing the threat register's "unknown/missing names are errors, not silent passes" (T-2-01) on the event stream.
- The runner's `ROOT` resolves via `git rev-parse --show-toplevel` with a script-relative fallback, so the Makefile invocation and scratch-copy invocations both bind to the repo.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first `scripts/red-runner` draft derived `ROOT` with `git rev-parse ... || cd ... && pwd` — shell's left-associative `||`/`&&` made `pwd` run even when git succeeded, producing a two-line `ROOT` and a "table not found" failure. Fixed before the Task 2 commit with an explicit `{ ... }` group around the fallback. (Self-inflicted, caught on first run, fixed within the task.)
- The strict lint gate's `noinlineerr` flagged the test's inline `if _, _, err := ...; err != nil` form; restructured to plain assignment before the Task 1 commit. No suppressions needed.
- Both issues were resolved inside their tasks' commits; no plan-level scope change.

## Known Stubs

None. The intentionally-failing red test is not a stub: it is the plan's deliverable (TEST-04's acceptance is that the R-03 acceptance tests fail on the current code), its failure is machine-asserted by the runner against the expectation table on every red-run, and its resolution path is the Phase 4 flip (`red` → `green` in `scripts/red-expectations.tsv`). The expectation table itself is the cross-run tracking ledger for red rows, so no entries are added to WINDOWS.md for them.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The red-gating mechanism (D-01) is proven end to end and is the precondition every other red deliverable in this phase builds on: later plans append rows to `scripts/red-expectations.tsv`, uncomment/extend the runner's `PACKAGES` list (the placeholders for `./internal/stand/`, `./internal/persistfile/`, `./internal/app/persist/` are already in the script), and land `TestRed_*` tests gated on `SUPKV_RED_TESTS=1`.
- `make red-run` is the single command the scheduled workflow's red-run job (02-11, D-09) will invoke.
- No blockers. The R-03 defect remains unfixed by design (phase boundary: tests only); Phases 4 (PROTO-01) flips the row when the fix lands.

## Self-Check: PASSED

Re-run on the committed tree (`ba6c5c8`):

- Key files exist: all 4 deliverables + both evidence directories — FOUND.
- `gsd-tools check evaluation-scope --plan 02-01 --commits-only --raw` → resolved, 3 commits, exit 0 (not 69).
- `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$'` → exit 1, FAIL ("Get while only a partial DataSync listing has arrived: want code Unavailable, got OK") — the red verdict.
- `go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$'` (no env) → exit 0, SKIP.
- `grep -v '^#' scripts/red-expectations.tsv | grep -c 'TestRed_R03_gateClosedUntilFullList<TAB>red'` → 1.
- Committed evidence `20261119T192248Z/go-test.json` contains `Action:"fail"` events for the R-03 test; `summary.txt` row reads `OK`.
- `make red-run` → exit 0 with the FAIL record in evidence (run on the committed Makefile content).
- `make lint` (config verify + `run ./...`) → exit 0. `make test` (`go test -race -cover ./...`) → exit 0, all packages ok — on the same tree content that is now HEAD.

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-19*
