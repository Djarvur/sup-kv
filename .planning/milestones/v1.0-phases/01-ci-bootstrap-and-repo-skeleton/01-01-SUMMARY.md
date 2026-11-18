---
phase: 01-ci-bootstrap-and-repo-skeleton
plan: 01
subsystem: infra
tags: [go, golangci-lint, depguard, lint-gate, tool-directives, makefile, vendored-config]

# Dependency graph
requires: []
provides:
  - Go module github.com/Djarvur/sup-kv (go 1.27.1) with compileable skeleton packages cmd/sup-kv, internal/port, internal/app
  - Strict lint gate: vendored powerman/golangci-lint-strict 2.15.0 config with machine-enforced depguard architecture boundary (sup-kv-app-no-io, sup-kv-port-stdlib-only)
  - Pinned dev tools as go.mod tool directives (golangci-lint v2.15.0, govulncheck v1.9.0), zero direct runtime requires
  - Repo hygiene files: Makefile (lint/test/generate/build via go tool), AGENTS.md (conventions + 3 recorded deviations), merged .gitignore
  - Red/green proof that the depguard boundary demonstrably fails the gate on violations
affects: [01-02 CI workflows, 01-03 dependabot, 01-04 branch protection, 02-engine, all later feature phases]

# Actuals (#2632) — pairs with the plan's `estimate` (30000 tokens, 3 tasks).
actuals:
  tokens: 31861
  tasks: 3
  commits: 3

# Measured commit provenance (#3968)
plan_head_before: d82451f4fef6d387ec5a5c115e2b548bea57f091d
plan_head_after: 7137282090438b43effad715bc3b16df3a298222

# Tech tracking
tech-stack:
  added:
    - golangci-lint v2.15.0 (go.mod tool directive, invoked via go tool)
    - govulncheck v1.9.0 (go.mod tool directive, invoked via go tool)
  patterns:
    - Vendored strict lint config: upstream powerman content byte-unmodified behind its Origin header; project rules additive only inside linters.settings.depguard.rules (extends: disproven — silently no-ops the whole config)
    - Depguard architecture boundary: quoted '**/internal/<pkg>/**' globs (absolute-path match) + '!$test' exemption; plain-prefix deny entries (no regex anchors)
    - Dev tools pinned as Go 1.24+ tool directives, never tools.go, never bare go install
    - External _test package with t.Parallel() everywhere (strict testpackage/paralleltest)

key-files:
  created:
    - go.mod
    - go.sum
    - cmd/sup-kv/main.go
    - internal/port/doc.go
    - internal/app/doc.go
    - internal/app/app_test.go
    - .golangci.yml
    - Makefile
    - AGENTS.md
  modified:
    - .gitignore

key-decisions:
  - "Tool directives accepted as the D-03 reading: zero DIRECT runtime requires only; the ~213 indirect requires dragged in by golangci-lint/govulncheck are documented in AGENTS.md"
  - "Lint config vendored single-file (no extends: mechanism in v2.15.0); sup-kv depguard rules inserted as new keys inside the upstream depguard.rules mapping — additive-only diff proven against a pristine upstream fetch"
  - "sup-kv-port-stdlib-only allows $gostd plus the own module path github.com/Djarvur/sup-kv (resolves Pitfall 5/A8: port→app type references stay legal in later phases)"
  - "app_test.go lives in external package app_test with t.Parallel() — satisfying the strict gate's testpackage/paralleltest rather than weakening the config"
  - "Everything (code and planning docs) landed via pull requests per CI-03: code via PR #1, this SUMMARY via a docs PR; no direct commits to master"

patterns-established:
  - "Boundary enforcement: architecture rules are machine-enforced depguard rules in .golangci.yml; any change to the boundary is a visible config diff, never a code-side workaround"
  - "Gate satisfaction direction: when the strict gate fires on skeleton code, fix the code, never weaken the config"
  - "PR-only landing: feature branch cut from up-to-date master, PR, merge with per-task commits preserved (--merge, no squash)"

requirements-completed: [QUAL-01, CI-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Zero-direct-runtime-requires Go module skeleton (cmd/sup-kv, internal/port, internal/app) that builds, vets, and passes race tests with a real test target, plus pinned tool directives"
    requirement: QUAL-01
    verification:
      - kind: integration
        ref: "go build ./... && go vet ./... && go test -race ./... (ok github.com/Djarvur/sup-kv/internal/app)"
        status: pass
      - kind: other
        ref: "go tool golangci-lint version → 2.15.0; go.mod require scan: all 213 requires carry // indirect"
        status: pass
    human_judgment: false
  - id: D2
    description: "Vendored powerman/golangci-lint-strict 2.15.0 config (Origin header line 1, upstream byte-unmodified) with the two sup-kv depguard boundary rules added inside linters.settings.depguard.rules"
    requirement: QUAL-01
    verification:
      - kind: integration
        ref: "go tool golangci-lint config verify && go tool golangci-lint run ./... (exit 0, zero findings)"
        status: pass
      - kind: other
        ref: "diff vs pristine upstream fetch = single pure insertion (64a65,88); Origin header asserted verbatim; each rule key occurs exactly once"
        status: pass
    human_judgment: false
  - id: D3
    description: "Repo hygiene files: Makefile (.PHONY lint/test/generate/build via go tool), AGENTS.md (conventions + three recorded deviations + D-03 reading), .gitignore merged without losing original entries"
    requirement: QUAL-01
    verification:
      - kind: other
        ref: "make lint && make test green; grep -Fx '.gocache/' && '/sup-kv' && 'coverage.*' .gitignore all succeed; AGENTS.md deviation grep"
        status: pass
    human_judgment: false
  - id: D4
    description: "Depguard boundary proven live by red/green probe (verbatim findings captured), probes removed, gate green with no go.mod/go.sum churn, and everything landed on master via merged PR #1 with branch deleted"
    requirement: CI-01
    verification:
      - kind: integration
        ref: "red run exit 1 citing sup-kv-app-no-io + sup-kv-port-stdlib-only (quoted verbatim below); green run exit 0; git diff --exit-code go.mod go.sum clean; gh pr view 1 → state MERGED; git ls-files shows no probe paths"
        status: pass
    human_judgment: false

# Metrics
duration: 10min
completed: 2026-10-31
status: complete
---

# Phase 1 Plan 01: CI Bootstrap and Repo Skeleton Summary

**Zero-dependency Go module skeleton behind a strict golangci-lint 2.15.0 gate whose depguard rules machine-enforce the app/port architecture boundary, landed on master via PR #1**

## Performance

- **Duration:** 10 min
- **Started:** 2026-10-31T14:58:21Z
- **Completed:** 2026-10-31T15:08:34Z
- **Tasks:** 3
- **Files modified:** 10 (9 created, 1 merged)

## Accomplishments

- Module `github.com/Djarvur/sup-kv` (go 1.27.1) with the D-01 skeleton: `cmd/sup-kv` minimal main (command doc comment, no testable logic), `internal/port` + `internal/app` doc.go placeholders, and a trivial table-driven `TestAppSkeleton` giving `go test -race` a real target — build, vet, race tests green from the first commit
- golangci-lint v2.15.0 and govulncheck v1.9.0 pinned as Go 1.24+ tool directives (D-04); every one of the 213 require lines carries `// indirect` (D-03 zero direct runtime requires)
- Strict lint gate live and green on the skeleton itself (QUAL-01): powerman strict config vendored byte-unmodified behind its Origin header, with `sup-kv-app-no-io` (deny `net`, `os`, `syscall`, `google.golang.org/` prefixes in internal/app) and `sup-kv-port-stdlib-only` (list-mode strict, allow `$gostd` + own module path in internal/port) inserted as additive keys inside `linters.settings.depguard.rules`
- Boundary proven to have teeth (D-11): the red/green probe demonstrably failed the gate citing both sup-kv lists, then the tree cleaned back to green with zero go.mod/go.sum churn
- Repo hygiene: Makefile targets invoking the same pinned binaries as CI will, AGENTS.md recording the three project deviations, .gitignore merged (originals kept)

## Task Commits

Each task was committed atomically (on `feat/01-01-repo-skeleton`, merged to master via PR #1 with commits preserved):

1. **Task 1: Module skeleton compiles, vets, and passes race tests with pinned tools** - `415ff00` (feat)
2. **Task 2: Vendored strict lint config with sup-kv depguard boundary + repo hygiene files** - `e456764` (feat)
3. **Task 3: Depguard red/green probe + PR landing** - `7137282` (merge of PR #1; probe files themselves were never committed)

**Plan metadata:** docs commit (this SUMMARY + state files) lands via a follow-up docs PR.

## Depguard Red/Green Evidence (Task 3, verbatim)

Red run (`go tool golangci-lint run ./internal/...` with both probe files in place, exit 1):

```
internal/app/probe_violation.go:3:8: import 'net' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no networking (depguard)
import "net"
       ^
internal/port/probe_violation.go:3:8: import 'golang.org/x/mod/semver' is not allowed from list 'sup-kv-port-stdlib-only' (depguard)
import "golang.org/x/mod/semver"
       ^
internal/app/probe_violation.go:7:6: Consider preallocating out with capacity 1 (prealloc)
	var out []string
	    ^
```

The third finding (prealloc) additionally proves the wider strict config is live, not just depguard (Pitfall 1 counter-evidence). After deleting both probes: `go tool golangci-lint run ./...` exit 0, `make lint && make test` green, `git diff --exit-code go.mod go.sum` clean, and `git ls-files` on merged master shows no probe paths.

## Files Created/Modified

- `go.mod` / `go.sum` - module + pinned tool directives; ~213 indirect requires, zero direct
- `cmd/sup-kv/main.go` - command doc comment + empty main (no testable logic, D-01)
- `internal/port/doc.go` - package port placeholder (stdlib-only boundary target)
- `internal/app/doc.go` - package app placeholder (pure decision core target)
- `internal/app/app_test.go` - trivial table-driven TestAppSkeleton (external test package, t.Parallel)
- `.golangci.yml` - vendored powerman strict 2.15.0 + two sup-kv depguard rules (additive-only)
- `Makefile` - .PHONY lint/test/generate/build via go tool
- `AGENTS.md` - go-ultimate conventions + three recorded deviations + D-03 dependency-policy reading
- `.gitignore` - merged go-ultimate baseline entries; all original entries kept

## Decisions Made

- D-03 read as "zero direct runtime requires": the tool-directive graph's ~213 `// indirect` lines are the accepted, documented consequence of D-04 (recorded in AGENTS.md so the go.mod face is not misread)
- `extends:` config inheritance disproven and avoided (research Pitfall 1 — it silently disables the entire strict config); single-file vendoring with additive rule keys instead, proven byte-identical upstream via diff against a pristine fetch
- Port rule allows the own module path in addition to `$gostd` (research Pitfall 5 / assumption A8) so port↔app type references remain legal in later phases
- Test file placed in external `app_test` package with `t.Parallel()` — the strict gate's testpackage/paralleltest findings were fixed in the code direction, never by weakening the config
- PR-only process honored for docs as well as code (CI-03): the executor's guard refuses default-branch commits, and the project constraint says every change, code and docs, lands through a pull request

## Deviations from Plan

None - plan executed exactly as written.

The one iteration loop the plan prescribed (Task 2: "iterating on any strict findings in the skeleton files ... until the gate is green", anticipating research Pitfall 10) materialized as two findings — `testpackage` and `paralleltest` on `app_test.go` — and was resolved in the prescribed direction (fix the code, keep the config strict) within Task 2's commit.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Strict gate governs master from merge commit 7137282; plans 02 (CI workflows), 03 (Dependabot/govulncheck), 04 (branch protection) build on the skeleton and the Makefile targets, which CI must invoke verbatim
- CI-01 remains open until plan 01-02 (CI workflows) completes — the requirement is shared between the two plans and its REQUIREMENTS.md checkbox is gated on both summaries existing
- The golangci-lint-action in plan 02 must pin `version: v2.15.0` to match the go.mod tool directive exactly (research Pitfall 4)

## Self-Check: PASSED

All 10 created files exist on disk; task commits 415ff00, e456764 and merge 7137282 are ancestors of HEAD (verified via `git merge-base --is-ancestor`). Acceptance criteria for all three tasks re-verified on merged master: gate green (`config verify` + `run ./...` exit 0), no probe paths in `git ls-files`, go.mod/go.sum clean vs Task 2 commit.

---
*Phase: 01-ci-bootstrap-and-repo-skeleton*
*Completed: 2026-10-31*
