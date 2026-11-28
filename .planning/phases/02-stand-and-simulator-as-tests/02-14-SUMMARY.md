---
phase: 02-stand-and-simulator-as-tests
plan: 14
subsystem: testing
tags: [stand, darwin, test-05, build-tags, loud-report, d03, adr-0010]

# Dependency graph
requires:
  - phase: 02-02 (same wave, sibling)
    provides: the internal/stand package (linux-tagged harness stand.go + scenarios stand_test.go) this file complements build-tag-independently, and TEST-05's Linux branch (skip = error) whose second branch this plan is
  - phase: 01-spec-reference-and-decisions
    provides: ADR 0010 (owner decision 14.10, darwin development-only) — the decision the loud statement cites
provides:
  - internal/stand/stand_nondarwin_test.go: the non-Linux half of the stand package — TestPlatform_darwinReplicationUntested, a named PASSING test the darwin CI cell executes to print the darwin-untested statement (decision 14.10 / ADR 0010, TEST-05), plus a build-tag boundary guard requiring every sibling .go file to be excluded for darwin
  - TEST-05's second branch: the macOS cell states the limitation through a green named result instead of a silent skip — with 02-02's Linux branch, the no-silent-skips policy now holds on every platform
affects: [02-11 (CI/scheduled wiring of the darwin cell), phases 3-5 (TEST-05 stays the standing policy), DOCS-10 darwin demotion consumers]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 671    # chars/4 over the realized diff (2684-byte file, 69 insertions)
  tasks: 1
  commits: 1     # MEASURED: git rev-list --count b44b8db..HEAD (#3968)
  plan_head_before: b44b8dbb60facd6227a98cdde16d9bbc59b93156
  plan_head_after: 51d7c92446aec9e69f40f5ba5eb79ae986e12fbe

# Tech tracking
tech-stack:
  added: none — stdlib only (go/build, os, path/filepath, runtime, testing); zero new module dependencies
  patterns:
    - platform-gated package halves: the linux-tagged harness/scenarios and the !linux loud report exclude each other from their builds by construction
    - build-tag boundary guard via go/build.MatchFile against an explicit darwin context — constraint-form-agnostic "excluded for darwin" semantics
    - the loud statement as a wrapped const so the plan's grep gate counts the phrase on exactly one source line

key-files:
  created:
    - internal/stand/stand_nondarwin_test.go
  modified: []

key-decisions:
  - "The leak assertion is a real runtime check, not only the compile-proof the plan's parenthetical names: go/build MatchFile with an explicit darwin context (GOOS=darwin, GOARCH=amd64) over every sibling .go file gives exact 'excluded for darwin' semantics regardless of constraint form (linux, linux && amd64, !darwin) and fails on the darwin cell with the leaking file's name"
  - "The darwin-untested statement is a wrapped string const so the plan's grep gate counts the phrase exactly once; the package/test doc comments deliberately avoid the phrase"

patterns-established:
  - "Non-Linux half of a platform-gated test package: ONE named PASSING test carrying the limitation statement (t.Logf only; a skip directive is forbidden — the green named result IS the report)"
  - "Build-tag boundary assertion: the platform report test scans the package dir and requires every sibling file to be excluded in an explicit foreign-GOOS go/build context, so a leaked scenario file fails with a named error instead of compiling untested socket code"

requirements-completed: [TEST-05]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "internal/stand/stand_nondarwin_test.go (//go:build !linux, package stand): TestPlatform_darwinReplicationUntested passes while printing the darwin-untested statement (decision 14.10 / ADR 0010, TEST-05) — the darwin CI cell reports through a green named result, never a silent skip; the same test asserts the build-tag boundary (every sibling .go file excluded for darwin); zero skip directives in the package on any platform. The compiled test's execution happens on the darwin CI cell post-merge (a Linux host cannot run the Mach-O binary); the plan's automated gate is the darwin compile."
    requirement: TEST-05
    verification:
      - kind: other
        ref: "GOOS=darwin go vet ./internal/stand/ → exit 0 (baseline before this plan: exit 1 'build constraints exclude all Go files')"
        status: pass
      - kind: other
        ref: "GOOS=darwin CGO_ENABLED=0 go test -c ./internal/stand/ → darwin test binary compiles (not executed locally, per plan)"
        status: pass
      - kind: unit
        ref: "grep -c 'darwin replication is untested' internal/stand/stand_nondarwin_test.go → 1"
        status: pass
      - kind: other
        ref: "grep -rc 'Skipf' internal/stand/ | grep -v ':0' | wc -l → 0"
        status: pass
    human_judgment: false

# Metrics
duration: 3min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 14: Loud Non-Linux Darwin Report Summary

**internal/stand's !linux half: `TestPlatform_darwinReplicationUntested` — a named PASSING test the darwin CI cell executes to print "darwin replication is untested" (decision 14.10 / ADR 0010, TEST-05), with a go/build boundary guard keeping every scenario file linux-tagged and zero skip directives in the package on any platform**

## Performance

- **Duration:** 3 min (178 s)
- **Started:** 2026-11-19T20:49:08Z
- **Completed:** 2026-11-19T20:52:06Z
- **Tasks:** 1
- **Files modified:** 1 (created; +69 lines)

## Accomplishments

- `internal/stand/stand_nondarwin_test.go` (`//go:build !linux`, package stand) — TEST-05's second branch: `TestPlatform_darwinReplicationUntested` asserts nothing about sockets and always passes, logging the plan's statement verbatim ("darwin replication is untested: the two-node real-socket scenarios in this package are linux-only (decision 14.10 / ADR 0010, TEST-05) — CI's darwin cell reports here instead of silently skipping socket tests") via `t.Logf` only — a skip directive is forbidden, so the darwin cell shows a green named result carrying the limitation.
- Build-tag boundary assertion in the same test: after locating itself via `runtime.Caller`, it scans the package directory and requires every sibling `.go` file to be excluded for darwin (`go/build.MatchFile` against an explicit darwin context: GOOS=darwin, GOARCH=amd64) — a future scenario file that leaks across the tag fails on the darwin cell with a named error instead of compiling untested socket code.
- The package now compiles for darwin (`GOOS=darwin go vet` exit 0 vs baseline exit 1), completing the stand package's platform halves: 02-02's linux-tagged harness + scenarios and this !linux loud report are excluded from each other's builds by construction (order-independent sibling plans, per the plan's assumption).

## Task Commits

Each task was committed atomically:

1. **Task 1: loud non-Linux report (TEST-05 second branch)** - `51d7c92` (test)

_Measured from the plan-start ledger `b44b8db` (`git rev-list --count b44b8db..HEAD` at SUMMARY write): 1 task commit._

## Files Created/Modified

- `internal/stand/stand_nondarwin_test.go` - created (69 lines): the non-Linux half of the stand package — the loud darwin report plus the build-tag boundary guard

## Decisions Made

- **The leak assertion is a real runtime check, not just the compile-proof.** The plan's action asks the test to "assert ... that no scenario file leaked across the build tag (the package compiles here without them by construction)". Compilation alone yields no named runtime error and no per-file diagnosis, so the test scans the package dir: `go/build.MatchFile` with an explicit darwin context gives the exact "excluded for darwin" semantics regardless of constraint form (`linux`, `linux && amd64`, `!darwin` — the acceptance must hold for any future tag wording), and the guard fails naming the leaking file.
- **The statement is a wrapped string const.** The plan's verify gates the phrase `darwin replication is untested` at exactly 1 line; a wrapped const keeps the phrase on one source line under formatter pressure, and the package/test doc comments deliberately avoid the phrase.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None — all verifies passed on the first run (darwin vet flipped baseline exit 1 → 0, statement count 1, Skipf 0, gofmt clean; the stronger `GOOS=darwin go test -c` compile proof also passed).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TEST-05 now has both branches implemented: 02-02's Linux branch (a capability-probe miss is a fatal error; zero skips in the package) and this plan's darwin branch (the green named report). With both declaring plans' SUMMARYs on disk, TEST-05 should now be ready to mark complete.
- The darwin CI cell (macos-15 in ci.yml) executes the named test on its next run post-merge — its green result carrying the statement is the TEST-05 macOS-cell acceptance going live; no further wiring is needed (the cell already runs `go test ./...`).
- No blockers. The stand package is complete for the phase's platform story: harness (02-02), scenarios (02-02), red tests (02-05, same wave), and the platform report (this plan).

## Self-Check: PASSED

Re-run on the committed tree (`51d7c92`):

- Key file exists: `internal/stand/stand_nondarwin_test.go` — FOUND (2684 bytes).
- Commit exists: `51d7c92` — ancestor of HEAD (`git merge-base --is-ancestor` → 0).
- `node $HOME/.zcode/gsd-core/bin/gsd-tools.cjs check evaluation-scope --plan "02-14" --commits-only --raw` → status resolved, 1 commit, exit 0 (not 69).
- `GOOS=darwin go vet ./internal/stand/` → exit 0 (baseline before this plan: exit 1).
- `grep -c 'darwin replication is untested' internal/stand/stand_nondarwin_test.go` → 1.
- `grep -rc 'Skipf' internal/stand/ | grep -v ':0' | wc -l` → 0.
- `gofmt -l internal/stand/` → clean.
- `GOOS=darwin CGO_ENABLED=0 go test -c ./internal/stand/` → darwin test binary compiles (execution reserved for the darwin CI cell per the plan's fails_when note).
- `go vet ./internal/stand/` (Linux) → exit 0 — the linux half is untouched by the !linux file (build-tag excluded).

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-19*
