---
phase: 01-ci-bootstrap-and-repo-skeleton
plan: 02
subsystem: infra
tags: [github-actions, ci, gate-matrix, go, golangci-lint-action, race-detector, tidy-gate, cross-compile]

# Dependency graph
requires:
  - phase: 01-01
    provides: Go module skeleton with tool directives (golangci-lint v2.15.0), vendored strict .golangci.yml with sup-kv depguard rules, real race-test target
provides:
  - .github/workflows/ci.yml on master — lint job + 4-cell gate matrix (ubuntu-latest, ubuntu-24.04-arm, macos-15 native; darwin-amd64-cross) + tidy gate, with D-06 workflow hygiene
  - Fixed check-run context names for the plan 04 ruleset: "lint", "gate (ubuntu-latest)", "gate (ubuntu-24.04-arm)", "gate (macos-15)", "gate (darwin-amd64-cross)"
  - End-to-end pipeline proof — branch → PR → all five checks green → merge (PR #3)
  - Red-gate teeth proof — a violating PR demonstrably fails the lint check citing sup-kv-app-no-io (PR #4, closed unmerged, master untouched)
affects: [01-04 branch protection ruleset, 01-03 dependabot, 02-engine, all later feature phases]

# Actuals (#2632) — pairs with the plan's `estimate` (26000 tokens, 3 tasks, confidence low).
actuals:
  tokens: 5304
  tasks: 3
  commits: 3

# Measured commit provenance (#3968)
plan_head_before: d3f306f974ce2d7be42e26bdda48ff68dc367926
plan_head_after: f9110eef751038b66a89bd48a029144776659d06

# Tech tracking
tech-stack:
  added:
    - GitHub Actions workflow: actions/checkout@v7, actions/setup-go@v7 (go-version-file: go.mod), golangci-lint-action@v9 with version: v2.15.0
  patterns:
    - Job display name `gate (${{ matrix.name }})` → check-run context `gate (<cell>)`; any matrix change is a ruleset change
    - CI tidy gate: go mod tidy + git diff --exit-code go.mod go.sum on every native cell (D-06 locked form)
    - Concurrency group ci-${{ github.event_name }}-${{ github.ref }} — event name in the key keeps push and PR runs for the same ref from canceling each other
    - Tidy must be verified under GOTOOLCHAIN=<go.mod version> to match the CI condition exactly

key-files:
  created:
    - .github/workflows/ci.yml
  modified:
    - go.sum

key-decisions:
  - "Check-run contexts fixed verbatim as lint, gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15), gate (darwin-amd64-cross) — plan 04's ruleset required_status_checks must copy these exactly"
  - "go.sum gap fixed in the module-files direction (complete go.sum), never by weakening the tidy gate; tidy verified idempotent under GOTOOLCHAIN=go1.27.1"
  - "PR #3 merged with --merge (per-task commits preserved, matching the plan 01-01 pattern); probe PR #4 closed unmerged with branch deleted — red evidence only"
  - "Local parity checks (vet/test/tidy) run under GOTOOLCHAIN=go1.27.1 because local go is 1.27.3 while CI's go-version-file installs 1.27.1"

patterns-established:
  - "Green-before-merge discipline: a red cell is fixed on the branch and re-pushed; a red PR is never merged"
  - "Red-path probes are throwaway branches: push → PR → observe failure → close --delete-branch; the violating file never reaches master"

requirements-completed: [CI-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "ci.yml with lint job + 4-cell gate matrix + tidy gate + D-06 hygiene (permissions, event-name concurrency key, cancel-in-progress, go-version-file, full patch pin v2.15.0, fail-fast false)"
    requirement: CI-01
    verification:
      - kind: other
        ref: "grep counts per plan <verify> (hygiene/pin/matrix keys = 8; contents: read + 3 runner labels present); no pull_request_target and no github.event.* anywhere in the file"
        status: pass
    human_judgment: false
  - id: D2
    description: "Green pipeline proven end-to-end: PR #3 shows exactly lint + gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15), gate (darwin-amd64-cross), all pass; merged to master (f9110ee), branch deleted"
    requirement: CI-01
    verification:
      - kind: integration
        ref: "gh pr view 3 --jq '.state == \"MERGED\"' → true; gh api commits/f9110ee.../check-runs assertion (missing-names + non-success) = 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "Red-gate teeth: probe PR #4's lint check fails citing the depguard boundary; PR CLOSED unmerged, branch deleted, origin/master tree contains no probe file"
    requirement: CI-01
    verification:
      - kind: integration
        ref: "gh pr checks 4 → lint fail (depguard finding quoted verbatim below); gh pr view 4 → state CLOSED, mergedAt null; git ls-tree origin/master -- internal/app/probe_violation.go empty"
        status: pass
    human_judgment: false

# Metrics
duration: 13min
completed: 2026-10-31
status: complete
---

# Phase 1 Plan 02: CI Bootstrap and Repo Skeleton — CI Workflows Summary

**.github/workflows/ci.yml on master: lint + 4-cell OS/arch gate matrix proven end-to-end — a green PR (#3) merged with all five checks passing, and a violating PR (#4) demonstrably red on the depguard boundary**

## Performance

- **Duration:** 13 min
- **Started:** 2026-10-31T15:14:27Z
- **Completed:** 2026-10-31T15:27:58Z
- **Tasks:** 3
- **Files modified:** 2 (1 created, 1 amended)

## Accomplishments

- `ci.yml` on master (merge f9110ee): push (master only) + unfiltered pull_request triggers; `permissions: contents: read`; concurrency `ci-${{ github.event_name }}-${{ github.ref }}` with `cancel-in-progress: true`; lint job (ubuntu-latest) via golangci-lint-action@v9 pinned `version: v2.15.0` — the same build the go.mod tool directive pins; `setup-go` with `go-version-file: go.mod` on every job (single Go version source, D-06)
- Gate matrix with `fail-fast: false` and four named cells: `ubuntu-latest`, `ubuntu-24.04-arm`, `macos-15` run the full native gate (vet + `test -race` + tidy gate); `darwin-amd64-cross` runs `GOOS=darwin GOARCH=amd64` build + vet on Linux (D-05 amended)
- Pipeline proven green: PR #3's first run exposed a real module-hygiene gap (tidy gate red on all three native cells), fixed on the branch, then all five checks green, merged with commits preserved, branch deleted — plus the push-event run on master itself completed all five checks (push trigger verified live)
- Teeth proven red: probe PR #4 (net import in internal/app) failed the lint check citing `sup-kv-app-no-io` while all four gate cells stayed green (the file compiles, vets, and passes race tests — only the boundary is violated); closed unmerged, branch deleted, master tree clean

## Check-Run Names (verbatim, for the plan 04 ruleset)

`gh pr checks 3` on the merged PR (final output):

```
gate (darwin-amd64-cross)	pass	18s	https://github.com/Djarvur/sup-kv/actions/runs/41214308494/job/124575525920
gate (macos-15)	pass	40s	https://github.com/Djarvur/sup-kv/actions/runs/41214308494/job/124575525984
gate (ubuntu-24.04-arm)	pass	46s	https://github.com/Djarvur/sup-kv/actions/runs/41214308494/job/124575526015
gate (ubuntu-latest)	pass	55s	https://github.com/Djarvur/sup-kv/actions/runs/41214308494/job/124575525981
lint	pass	37s	https://github.com/Djarvur/sup-kv/actions/runs/41214308494/job/124575525749
```

The five context strings the plan 04 ruleset must require, exactly: `lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)`, `gate (darwin-amd64-cross)`.

## Red-Gate Evidence (Task 3, verbatim)

Failed check on probe PR #4: `lint` — fail (36s).

Finding line from the failing run's log (job 124576486415):

```
##[error]internal/app/probe_violation.go:5:8: import 'net' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no networking (depguard)
```

Supporting findings in the same log prove the wider strict config is live, not just depguard:

```
##[error]internal/app/probe_violation.go:9:6: Consider preallocating out with capacity 1 (prealloc)
##[error]internal/app/probe_violation.go:1:1: package-comments: package comment should be of the form "Package app ..." (revive)
```

Probe disposition: PR #4 state CLOSED (`mergedAt: null`), branch `probe/red-gate-teeth` deleted, `git ls-tree origin/master -- internal/app/probe_violation.go` empty — master never contained the probe.

## Must-Haves Truth Evidence

- **Full gate matrix green on the merged PR**: check-runs assertion on merge SHA f9110ee = 0 (all five names present, all conclusions success). Race gate not vacuous — ubuntu-latest cell log: `ok  github.com/Djarvur/sup-kv/internal/app 1.009s` under `go test -race ./...`.
- **Matrix cells independent**: PR #3's first run had three red native cells; the lint and darwin-amd64-cross cells ran to completion (pass) instead of being canceled — fail-fast disabled works.
- **Push/PR concurrency isolation**: the merge triggered a push-event run on `refs/heads/master` that completed all five checks while PR-event runs elsewhere completed too; the group key structurally embeds `github.event_name` (verified in-file).
- **Teeth**: red probe evidence above — the gate demonstrably fails bad PRs.

## Task Commits

Each task was committed atomically (on `feat/01-02-ci-workflow`, merged to master via PR #3 with commits preserved):

1. **Task 1: ci.yml — lint job, 4-cell gate matrix, tidy gate, workflow hygiene** - `1195848` (feat)
2. **Task 2: Prove the pipeline — PR green on all four cells, merge, record check names** - `c6f8eb5` (fix: go.sum tidy-gate completion, pushed during Task 2's fix-and-repush loop) + merge `f9110ee` (PR #3)
3. **Task 3: Red-gate teeth — violating PR demonstrably fails the lint check** - probe commit `ae1f5e2` on throwaway branch `probe/red-gate-teeth` (deleted; never merged, never on master — not part of master history)

**Plan metadata:** docs commit (this SUMMARY + state files) lands via a follow-up docs PR (CI-03 PR-only rule).

## Files Created/Modified

- `.github/workflows/ci.yml` - lint job + 4-cell gate matrix + tidy gate + D-06 hygiene (the entire plan deliverable)
- `go.sum` - +54 tidy-required entries (full-zip hashes for transitive graph modules: go-simpler.org/assert, go.uber.org/goleak, golang.org/x/net v0.61.0, x/tools test helpers, gopkg.in/check.v1, ...); go.mod unchanged

## Decisions Made

- Check-run contexts are now fixed contract strings (five names above); any future matrix cell rename is a ruleset change (recorded in ci.yml's header comment and STATE.md)
- The tidy-gate failure was fixed by completing go.sum, not by weakening the gate — consistent with the phase's gate-satisfaction direction (fix code/files, keep the gate strict)
- Local verification runs use `GOTOOLCHAIN=go1.27.1` to reproduce the exact CI condition (local go is 1.27.3; CI installs 1.27.1 from go-version-file)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] go.sum was not tidy-complete; native gate cells failed the tidy gate**
- **Found during:** Task 2 (pipeline proof)
- **Issue:** PR #3's first run: `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)` failed — `go mod tidy` under the CI toolchain added 54 go.sum entries (plan 01-01 had verified `git diff --exit-code` clean but never actually run tidy locally)
- **Fix:** ran `GOTOOLCHAIN=go1.27.1 go mod tidy` (exact CI condition), committed the completed go.sum, verified a second tidy is a byte-identical no-op, re-pushed; all five checks then green
- **Files modified:** go.sum
- **Verification:** tidy idempotent under go1.27.1; vet + race tests green under same pin; PR #3 all checks pass; check-runs assertion on merge SHA = 0
- **Committed in:** c6f8eb5 (part of Task 2's fix-and-repush loop, as the plan prescribes: "If any cell fails, fix the workflow or code on the branch and push again")

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The tidy gate did exactly what D-06 designed it to do — caught real module-file drift on its very first run. Fix stayed inside Task 2's prescribed loop; no scope creep.

## Issues Encountered

- Task 2's second verify (check-runs assertion on the merge SHA) initially returned 5 instead of 0: merging PR #3 triggered the push-event run on master, whose checks were still in-flight (`conclusion: null`) at assertion time. Once the push run completed (all five success), the assertion returned 0. Not a defect — it is live evidence the push trigger works; noted so plan 04 knows check-runs on a merge commit settle only after the push-event run finishes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ci.yml active on master from merge f9110ee; every future PR automatically runs strict lint + the 4-cell matrix, and every push to master re-runs it
- Plan 04's ruleset must copy the five context strings verbatim (section above) — any matrix rename between now and then invalidates them
- Plans 03 (Dependabot/vulncheck) and 04 (branch protection) can proceed; wave ordering per the phase plan
- CI-01 is fully evidenced now (matrix half + teeth half); merge-blocking enforcement itself still lands with the plan 04 ruleset (documented split)

## Self-Check: PASSED

`.github/workflows/ci.yml` exists on master (merge f9110ee); commits 1195848, c6f8eb5, f9110ee are ancestors of origin/master (`git merge-base --is-ancestor`); probe commit ae1f5e2 is NOT in master history; check-runs assertion on f9110ee = 0; all Task 1-3 acceptance criteria re-verified (five checks green on PR #3; PR #4 lint fail + CLOSED; master tree clean of probe).

---
*Phase: 01-ci-bootstrap-and-repo-skeleton*
*Completed: 2026-10-31*
