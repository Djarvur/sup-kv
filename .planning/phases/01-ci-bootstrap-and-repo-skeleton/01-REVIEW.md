---
phase: 01-ci-bootstrap-and-repo-skeleton
reviewed: 2026-10-31T16:16:54Z
depth: standard
files_reviewed: 16
files_reviewed_list:
  - AGENTS.md
  - cmd/sup-kv/main.go
  - docs/setup/branch-protection.md
  - docs/setup/ruleset.json
  - .github/dependabot.yml
  - .github/workflows/ci.yml
  - .github/workflows/vulncheck.yml
  - .gitignore
  - .golangci.yml
  - go.mod
  - go.sum
  - internal/app/app_test.go
  - internal/app/doc.go
  - internal/port/doc.go
  - Makefile
  - .claude/CLAUDE.md
findings:
  critical: 0
  warning: 1
  info: 4
  total: 5
status: issues_found
---

# Phase 1: Code Review Report

**Reviewed:** 2026-10-31T16:16:54Z
**Depth:** standard
**Files Reviewed:** 16
**Status:** issues_found

## Summary

Phase 1 skeleton for sup-kv: CI quality gates (strict golangci-lint + 4-cell OS/arch
matrix), Dependabot + govulncheck automation, and a recorded-but-deferred branch
protection ruleset. This review went beyond static reading: every YAML/JSON file was
machine-parsed, the Go code was built/vetted/linted with the project's own pinned
toolchain, `go mod tidy` sync was verified in a scratch copy, and — decisively — the
depguard architecture-boundary rules were verified against the actual depguard v2.2.1
source in the module cache and then empirically probed with deliberately violating
scratch files (in `/tmp`, never in the repo).

**Verified correct (adversarially confirmed, not assumed):**

- **Depguard rules work as commented.** depguard v2.2.1 `strInPrefixList`
  (`settings.go:224-248`) confirms: plain `pkg` strings are prefix matches (`net`
  catches `net/http`), trailing `$` means exact match, and bare dot-free words never
  match dotted third-party paths (so `pkg: os` cannot false-positive on e.g.
  `gosimple/...`). Empirical probe: `internal/app` file importing `net/http` +
  `os/exec` → denied by `sup-kv-app-no-io`; `internal/port` file importing
  `github.com/stretchr/testify` → denied by `sup-kv-port-stdlib-only` (strict mode);
  port file importing stdlib + `github.com/Djarvur/sup-kv/internal/app` → allowed.
  `!$test` expands to `!**/*_test.go` and rules without `files:` (upstream `main`)
  apply to all files — all exactly as the config comments claim.
- **Workflows are injection- and privilege-hardened.** Every `${{ }}` interpolation in
  `ci.yml` draws from the statically-defined matrix or config, never from untrusted
  PR data; there is no `pull_request_target`; both workflows pin
  `permissions: contents: read`; no secrets are referenced.
- **Ruleset contexts match CI check names exactly**: `lint` (job without `name:`) and
  `gate (${{ matrix.name }})` produce exactly the five contexts in `ruleset.json`
  (`lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)`,
  `gate (darwin-amd64-cross)`). `ruleset.json` is valid JSON with correct ruleset-API
  field names; the `gh api` commands in `branch-protection.md` target the actual
  remote (`git remote -v` → `Djarvur/sup-kv`).
- **Toolchain state is consistent**: `go build`/`go vet`/gofmt clean; full strict
  lint gate passes on the skeleton (`go tool golangci-lint run ./...` exit 0, so the
  project-added depguard keys parse and compile inside the vendored config);
  `go mod tidy && git diff --exit-code go.mod go.sum` is clean in a scratch copy, so
  the CI tidy gate will not fail spuriously; `vulncheck.yml` cron `0 6 * * * 1`
  matches its "weekly, Monday 06:00 UTC" comment and uses the go.mod tool pin as
  claimed.

No Critical findings. One Warning (an automation design flaw that will silently break
a documented invariant), plus four Info items.

## Critical Issues

None.

## Warnings

### WR-01: Dependabot will silently break the golangci-lint local/CI version-parity invariant

**File:** `.github/dependabot.yml:7-19` (interacting with `.github/workflows/ci.yml:28`, `.golangci.yml:1`, `go.mod:94`)
**Issue:** The `tool` directive pulls `github.com/golangci/golangci-lint/v2 v2.15.0` into go.mod as a require line (line 94, `// indirect`). Dependabot's `gomod` ecosystem opens PRs for outdated requires — including indirect ones — so the weekly grouped "minor+patch" PR will bump the golangci-lint module version. But nothing updates the two other pins that the design requires to stay in lockstep:

1. the action input `version: v2.15.0` in `ci.yml` (Dependabot's `github-actions` ecosystem only bumps `uses:` refs, never `version:` inputs), and
2. the vendored powerman strict config, which per AGENTS.md deviation #2 must match the lint version **exactly** ("mismatch fails to parse or silently drops sections" per the project's own research).

The trap fires silently: the Dependabot PR passes CI (the lint job still runs v2.15.0 from the action pin), so merging it looks green — after which local `make lint` runs a different golangci-lint against the 2.15.0-exact config, violating the "full patch pin == go.mod tool directive (local/CI parity)" comment in `ci.yml:28` and potentially failing to parse or silently dropping lint sections. All three pins must move together, and only one of them is automated.

**Fix:** exclude the linter module from automated updates so bumps go through the manual re-vendor protocol that AGENTS.md deviation #2 already mandates:

```yaml
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
    ignore:
      # Must move in lockstep with .golangci.yml (vendored 2.15.0 strict config)
      # and the `version:` pin in .github/workflows/ci.yml — re-vendor by hand
      # per AGENTS.md deviation #2. See WR-01 in 01-REVIEW.md.
      - dependency-name: github.com/golangci/golangci-lint/v2
```

(`golang.org/x/vuln` needs no exclusion: `vulncheck.yml` resolves it via `go tool`, so the Dependabot bump and the workflow stay in sync by construction.)

## Info

### IN-01: AGENTS.md quotes an outdated go directive value

**File:** `AGENTS.md:5`
**Issue:** "Go version: as pinned in `go.mod` (`go 1.27`)" — go.mod actually pins `go 1.27.1`. The self-referential phrasing ("as pinned in go.mod") keeps this harmless, but the parenthetical value is stale.
**Fix:** Update the parenthetical to `go 1.27.1`, or drop the value and rely on the pointer to go.mod.

### IN-02: .claude/CLAUDE.md asserts branch protection that is not yet applied

**File:** `.claude/CLAUDE.md:17`
**Issue:** The Process constraint states "branch protection on `master`, direct pushes forbidden" as if enforced. Per `docs/setup/branch-protection.md` the ruleset is **NOT applied** (403 rulesets-API error on the private org repo; plan-gated, deferred 2026-10-31). An agent trusting the generated context would assume direct pushes to master are blocked when they are not — the master branch is currently protected only by convention. This is the one actively-misleading staleness found in the generated file (reviewed for stale content only, per scope).
**Fix:** Regenerate the context after the ruleset is applied, or add a caveat that CI-03's protection sub-item is plan-gated/deferred (see docs/setup/branch-protection.md).

### IN-03: Third-party actions pinned by mutable major tags, not commit SHAs

**File:** `.github/workflows/ci.yml:22-26,46`, `.github/workflows/vulncheck.yml:26`
**Issue:** `actions/checkout@v7`, `actions/setup-go@v7`, and third-party `golangci/golangci-lint-action@v9` are tag refs, which are mutable. A hijacked tag executes attacker code in CI. Impact here is bounded (token is `contents: read`, no secrets in either workflow, no `pull_request_target`, PR code already runs in `go test`), so this is hardening advice, not a vulnerability.
**Fix:** Pin third-party actions (at minimum `golangci/golangci-lint-action`) to full commit SHAs; Dependabot updates SHA pins for `github-actions` natively.

### IN-04: The no-I/O boundary exempts test files and does not cover x/sys

**File:** `.golangci.yml:68-80`
**Issue:** Two intentional-but-worth-knowing gaps in `sup-kv-app-no-io`: (a) `!$test` exempts `*_test.go` in `internal/app`, so the machine-enforced purity claim covers production files only — app tests could open sockets/files without lint complaint (the exemption is deliberate per the inline comment, and is what lets port/app tests use the planned `pgregory.net/rapid` test-only dep); (b) the deny entries `syscall`/`google.golang.org/` do not cover `golang.org/x/sys/unix` (dotted path, separate module), so raw-syscall access would only be caught by the dependency policy (a visible go.mod change), not by the lint gate.
**Fix:** Accept as documented; if the boundary should bind tests too, drop `!$test` from this rule only (keep it in the port rule) and add `golang.org/x/sys/` to the deny list in the same `.golangci.yml` commit.

---

_Reviewed: 2026-10-31T16:16:54Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
