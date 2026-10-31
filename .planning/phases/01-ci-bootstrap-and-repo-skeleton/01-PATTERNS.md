# Phase 1: CI Bootstrap and Repo Skeleton - Pattern Map

**Mapped:** 2026-10-31
**Files analyzed:** 13 (12 create, 1 modify) + 1 verification-only artifact
**Analogs found:** 0 / 13 — **the repository is greenfield**

> **Greenfield finding (explicit):** `git ls-files` shows only planning docs (`.planning/**`),
> `.claude/CLAUDE.md`, `README.md`, `LICENSE`, `.gitignore`, `docs/IDEA.md`. There is no Go
> source, no `go.mod`, no `.github/`, no `.golangci.yml`, no `Makefile`, no `AGENTS.md`. Every
> pattern below therefore comes from two clearly-labeled non-repo sources, per the phase brief:
>
> 1. **[R]** `01-RESEARCH.md` Code Examples / Patterns — all executed or live-verified today
>    against go 1.27.1 + golangci-lint v2.15.0 (highest confidence; prefer these).
> 2. **[G]** go-ultimate skill v0.15.0 (mandated conventions per CONTEXT.md canonical_refs;
>    external plugin cache, NOT repo analogs):
>    `/home/nil/.zcode/cli/plugins/cache/djarvur-plugin-marketplace/go-ultimate/0.15.0/skills/go-ultimate/`
>    — key files: `references/repo-and-ci.md`, `references/engineering-policy.md`,
>    `references/project-layouts.md`, `assets/ci/test.yml`, `assets/ci/lint.yml`,
>    `assets/cli-simple/main.go`.
>
> No excerpt below is invented; each cites its source file and line range.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `go.mod` | config | declarative (tool pinning) | none — greenfield | none |
| `cmd/sup-kv/main.go` | component (entrypoint) | n/a (placeholder per D-01) | none — greenfield ([G] policy + template) | none |
| `internal/port/doc.go` | component (package placeholder) | n/a | none — greenfield | none |
| `internal/app/doc.go` | component (package placeholder) | n/a | none — greenfield | none |
| `internal/app/app_test.go` | test | batch (test run) | none — greenfield | none |
| `.golangci.yml` | config (lint gate) | batch (lint over files) | none — greenfield ([R] verified config) | none |
| `.github/workflows/ci.yml` | config (CI workflow) | event-driven (push/PR) | none — greenfield ([R]+[G] workflow) | none |
| `.github/workflows/vulncheck.yml` | config (CI workflow) | event-driven (cron + dep PRs) | none — greenfield ([R]+[G] workflow) | none |
| `.github/dependabot.yml` | config | event-driven (scheduled updates) | none — greenfield ([R] verified keys) | none |
| `Makefile` | config (task runner) | request-response (target invocation) | none — greenfield ([G] task runner) | none |
| `AGENTS.md` | config (agent conventions) | n/a | none — greenfield ([G] template) | none |
| `docs/setup/` ruleset script + doc (names = discretion) | utility + docs | request-response (`gh api` calls) | none — greenfield ([R] verified payload) | none |
| `.gitignore` (modify) | config | n/a | none — closest in-repo file is itself | self |

Verification-only artifact (NOT committed to master): `internal/app/probe_violation.go` — the
depguard red/green probe ([R] lines 543-561).

## Pattern Assignments

### `go.mod` (config, declarative)

**No repo analog — greenfield.** Reference: [R] "Installation" lines 108-135 (executed today on go 1.27.1).

Commands to run (D-04; never `tools.go`, never bare `go install` — [G] repo-and-ci.md lines 138-149):

```bash
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
go get -tool golang.org/x/vuln/cmd/govulncheck@v1.9.0
```

Resulting shape (verbatim from the executed scratch module, [R] lines 117-131):

```
module github.com/Djarvur/sup-kv

go 1.27.1

tool (
	golang.com/golangci/golangci-lint/v2/cmd/golangci-lint
	golang.org/x/vuln/cmd/govulncheck
)

require (
	... ~200 lines: golangci/golangci-lint/v2 v2.15.0 // indirect
	... golang.org/x/vuln v1.9.0 // indirect
)
```

Planner must accept: `tool` directives drag ~200 `// indirect` requires into `go.mod`/`go.sum`;
"zero-dependency" (D-03) means *zero direct runtime requires* only ([R] lines 133). Go version
policy: applications pin latest stable → `go 1.27` ([G] engineering-policy.md lines 30-40).

---

### `cmd/sup-kv/main.go` (component/entrypoint, placeholder)

**No repo analog — greenfield.** Shape mandate: [G] engineering-policy.md §package main (lines 42-54):

- `cmd/<app-name>/main.go` for the single app (sup-kv uses `cmd/sup-kv/` per D-01 — matches).
- `main()` manages only: flags, env, signal handling, graceful shutdown — **and must contain no
  code that should be tested.**

Closest concrete template ([G] `assets/cli-simple/main.go` lines 1-5 — note: flat-layout variant;
sup-kv keeps the same `main`-does-nothing-testable shape under `cmd/sup-kv/`):

```go
// Command my-tool is a flat single-file CLI.
//
// Template: cli-simple. Layout: flat at module root, no internal/, no cmd/.
// See go-ultimate/references/project-layouts.md § Script and § CLI.
package main
```

Phase 1 scope is a *minimal placeholder main* (D-01) — a package comment plus an empty or
near-empty `main()` is enough; do NOT port the flag/signal logic from the template yet (that is
Phase 5's binary). The package comment is mandatory even for the placeholder: the strict config's
revive `package-comments` fires on trivial two-line files ([R] Pitfall 10, lines 446-447).

---

### `internal/port/doc.go` and `internal/app/doc.go` (component, package placeholders)

**No repo analog — greenfield.** The `internal/port` + `internal/app` naming is not invented — it
is go-ultimate's service layout verbatim ([G] engineering-policy.md §internal/ discipline,
lines 644-653):

> For services: `internal/port`, `internal/app`, `internal/in`, `internal/dal`, `internal/out`
> (see architecture.md).

Placeholder content requirements (all from [R] Pitfall 10 — the strict gate fires on style from
commit #1):
- Every package needs a doc comment (`// Package port ...` / `// Package app ...`) — verified:
  revive `package-comments` fails trivial placeholders without one.
- Adapter dirs are NOT created now (D-01); only `port` and `app` exist after this phase.

---

### `internal/app/app_test.go` (test, batch)

**No repo analog — greenfield.** Requirements assembled from [R] + [G]:
- One trivial test so `go test -race ./...` has a real target (D-01; [R] Validation Architecture
  line 647: "Wave 0 Gaps").
- Framework: stdlib `testing` only, table-driven, **no assertion library** ([G] mandate; echoed in
  `.claude/CLAUDE.md` line 50 and [R] line 626).
- Write it in table-test shape immediately — the strict config (prealloc etc.) fires on naive
  loops even in a trivial test ([R] Pitfall 10).

Skeleton shape to plan for (structure per go-ultimate testing conventions; content is discretion):

```go
func TestPlaceholder(t *testing.T) {
	tests := []struct {
		name string
	}{ /* one trivial case */ }
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { /* trivial assert via t.Error */ })
	}
}
```

---

### `.golangci.yml` (config, lint gate) — the highest-risk file in the phase

**No repo analog — greenfield.** Reference pattern is [R] Pattern 1 (lines 227-281), executed
end-to-end today. Two load-bearing facts:

1. **`extends:` DOES NOT EXIST in v2.15.0 and silently disables the entire config** ([R] Pitfall 1;
   D-10's "`.golangci.yml` extends: it" is disproven — planner must ship the single-file vendored
   form, still within D-10's intent of byte-unmodified upstream + project additions).
2. Project rules go **inside the existing `linters.settings.depguard.rules` mapping as new keys** —
   a second top-level `linters:` key breaks YAML parsing ([R] Pitfall 3).

Verified config shape ([R] lines 235-272, abridged; upstream sections stay byte-unmodified with
the `# Origin:` provenance header intact):

```yaml
# Origin: https://github.com/powerman/golangci-lint-strict version 2.15.0
version: "2"
linters:
  default: all
  disable:
    # ... upstream disable list, byte-unmodified ...
  settings:
    # ... upstream settings ...
    depguard:
      rules:
        main:                          # upstream rule, byte-unmodified
          deny:
            - pkg: math/rand$
              desc: use math/rand/v2
        # ---- sup-kv architecture boundary (project additions; upstream above is unmodified) ----
        sup-kv-app-no-io:
          files:
            - '**/internal/app/**'     # leading **/ REQUIRED (absolute path match); quotes REQUIRED
            - '!$test'                 # exempts *_test.go from the boundary
          deny:
            - pkg: net                 # plain string = PREFIX match (catches net, net/http, net/url)
              desc: internal/app is the pure decision core - no networking
            - pkg: os
              desc: internal/app is the pure decision core - no filesystem
            - pkg: syscall
              desc: internal/app is the pure decision core - no syscalls
            - pkg: google.golang.org/
              desc: internal/app is the pure decision core - no gRPC/protobuf
        sup-kv-port-stdlib-only:
          files:
            - '**/internal/port/**'
            - '!$test'
          list-mode: strict
          allow:
            - $gostd                   # NOTE: also denies own module path — see Pitfall 5
```

Depguard semantics the planner must respect ([R] Pattern 2, lines 283-290): globs match absolute
paths (leading `**/` mandatory, quoted); deny entries are prefix matches, `suffix$` is exact,
`^net$`-style regex matches nothing; `$gostd` = GOROOT/src top-level dirs.

Open finalization point for the planner: whether `sup-kv-port-stdlib-only` also allows
`github.com/Djarvur/sup-kv` (own module) — [R] Pitfall 5 / assumption A8; wrong choice blocks
legitimate port→app type references in later phases.

nolint policy (D-12) is already encoded upstream: `nolintlint.require-explanation/require-specific:
true` verified in the vendored config ([R] line 60).

---

### `.github/workflows/ci.yml` (config, event-driven CI)

**No repo analog — greenfield.** Two-layer reference:

- **Structure/hygiene layer** [G] repo-and-ci.md "Minimum workflow" (lines 214-265) and
  `assets/ci/test.yml` (lines 1-75) — the `permissions`/`concurrency`/`go-version-file`/tidy-gate
  skeleton. [G] `assets/ci/lint.yml` adds a `gofmt -l` guard step (lines 43-50).
- **Adapted, version-correct layer** [R] Code Examples (lines 453-513) — the go-ultimate minimum
  re-pinned to the phase's locked versions and 4-cell matrix. **Prefer [R]; see version warning
  below.**

Verified workflow ([R] lines 456-513, the load-bearing parts):

```yaml
name: CI
on:
  push:
    branches: [master]        # sup-kv default branch is master — NOT [main] as in [G] examples
  pull_request:

permissions:
  contents: read

concurrency:
  group: ci-${{ github.event_name }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v9
        with:
          version: v2.15.0        # full patch pin == go.mod tool directive (Pitfall 4)

  gate:                            # required checks are named "gate (<cell>)"
    strategy:
      fail-fast: false             # a red cell must not cancel the others' evidence
      matrix:
        include:
          - name: ubuntu-latest          # linux/amd64 native — full gate
          - name: ubuntu-24.04-arm       # linux/arm64 native — full gate ($0.005/min)
          - name: macos-15               # darwin/arm64 native — full gate
          - name: darwin-amd64-cross     # cross-compile cell on Linux
            runs-on: ubuntu-latest
            goos: darwin
            goarch: amd64
            cross: true
    runs-on: ${{ matrix.runs-on || matrix.name }}
    name: gate (${{ matrix.name }})
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - if: matrix.cross
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        run: go build ./... && go vet ./...
      - if: ${{ !matrix.cross }}
        run: |
          go vet ./...
          go test -race ./...
          go mod tidy
          git diff --exit-code go.mod go.sum
```

Hard constraints on any adaptation:
- Job display names feed required-check contexts: `gate (ubuntu-latest)`, `gate (macos-15)`, etc.
  Matrix changes are ruleset changes ([R] Pitfall 6, lines 426-429).
- No `run:` step interpolates `github.event.*` untrusted fields ([R] Security Domain, V5).
- Matrix values are static literals; no `pull_request_target`.

**VERSION WARNING for the planner:** the [G] examples/assets pin older majors —
`checkout@v4`/`setup-go@v5`/`golangci-lint-action@v7` with `version: v2.13`, and
`go run golang.org/x/vuln/cmd/govulncheck@latest`. These are superseded by the phase's locked,
re-verified set: `checkout@v7`, `setup-go@v7`, action `v9`, `version: v2.15.0`, `go tool
govulncheck` (D-04/D-07; [R] lines 93-98, Pitfall 4). Copy [G] structure, never [G] versions.

---

### `.github/workflows/vulncheck.yml` (config, event-driven scheduled)

**No repo analog — greenfield.** Verified reference [R] lines 517-540:

```yaml
name: vulncheck
on:
  schedule:
    - cron: "0 6 * * 1"   # weekly, Monday 06:00 UTC (discretion — pick and record)
  pull_request:
    paths:
      - go.mod
      - go.sum
permissions:
  contents: read
concurrency:
  group: vulncheck-${{ github.ref }}
  cancel-in-progress: true
jobs:
  vuln:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - run: go tool govulncheck ./...
```

Same version warning as ci.yml (the [G] variant uses `go run ...@latest` — do not copy; the
tool-directive invocation keeps local/CI versions identical). Cron time is planner discretion;
consider adding `workflow_dispatch:` for testability ([R] line 636).

---

### `.github/dependabot.yml` (config, event-driven updates)

**No repo analog — greenfield.** Verified reference (keys checked against the official options
reference) — [R] Pattern 4, lines 338-367:

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
    commit-message:
      prefix: go-deps
      include: scope
    open-pull-requests-limit: 10
    groups:
      gomod-safe-updates:
        update-types:
          - minor
          - patch
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
    commit-message:
      prefix: ci
      include: scope
    open-pull-requests-limit: 10
    groups:
      actions-safe-updates:
        update-types:
          - minor
          - patch
```

Alerts half of CI-02 is GHAS-gated on private repos — verify updates + govulncheck, document
alerts as plan-gated ([R] Pitfall 9).

---

### `Makefile` (config, task runner)

**No repo analog — greenfield.** Reference [G] repo-and-ci.md "Task runner" (lines 329-342):

```makefile
.PHONY: lint test generate build
lint:     ; go tool golangci-lint run ./...
test:     ; go test -race -cover ./...
generate: ; go generate ./...
build:    ; go build ./...
```

Constraint: **CI runs the same targets** — if CI and the Makefile drift, the Makefile becomes a
lie ([G] lines 340-342; [R] line 375). Invocations must go through `go tool` (same-version
guarantee vs the tool directives).

---

### `AGENTS.md` (config, agent conventions)

**No repo analog — greenfield; the file is currently MISSING and go-ultimate lists it as a
required file** ([G] repo-and-ci.md lines 25-38; [R] assumption A7). Template ([G] lines 43-53):

```markdown
# AGENTS

## Conventions
- Go version: as pinned in `go.mod`. Do not raise it without an ADR.
- Run `go tool golangci-lint run ./...` and `go test -race ./...` before pushing.
- Deviations from go-ultimate: <none | list them with the reason>

## Layout
- <anything a reader could not infer from the tree>
```

Rule: short and specific — deviations only, not a restatement of general Go practice ([G] line 40).
Project deviations to record here: depguard boundary rules, vendored-config provenance rule (D-10),
nolint justification rule (D-12).

---

### `docs/setup/` ruleset script + doc (utility + docs; names are discretion)

**No repo analog — greenfield.** Verified reference [R] Pattern 3 (lines 294-330) — the payload
was field-checked against the REST reference:

```bash
gh api --method POST repos/Djarvur/sup-kv/rulesets --input ruleset.json
```

```json
{
  "name": "protect master",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["refs/heads/master"] } },
  "rules": [
    { "type": "pull_request",
      "parameters": {
        "dismiss_stale_reviews_on_push": true,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_approving_review_count": 1,
        "required_review_thread_resolution": true
      } },
    { "type": "required_status_checks",
      "parameters": {
        "required_status_checks": [
          { "context": "lint" },
          { "context": "gate (ubuntu-latest)" },
          { "context": "gate (ubuntu-24.04-arm)" },
          { "context": "gate (macos-15)" },
          { "context": "gate (darwin-amd64-cross)" }
        ],
        "strict_required_status_checks_policy": true
      } },
    { "type": "non_fast_forward" },
    { "type": "deletion" }
  ]
}
```

Read-side verification and repo setting ([R] lines 327-330):

```bash
gh api repos/Djarvur/sup-kv/rulesets --jq '.[] | {name, enforcement, rules: [.rules[].type]}'
gh api --method PATCH repos/Djarvur/sup-kv -f delete_branch_on_merge=true
```

Constraints: context strings must equal the exact matrix check-run names `<job> (<matrix values>)`
([R] Pitfall 6); `required_approving_review_count: 1` is an example value, not user-locked ([R]
A5 — planner decides with user); **the whole task is plan-gated by the live 403 on rulesets**
([R] Pitfall 7 / Open Question Q1) — plan it behind a `checkpoint:human-verify` and record the
commands in this doc as re-appliable either way (D-08).

---

### `.gitignore` (modify, config)

**Closest in-repo file is itself** — current content (5 blocks: `/sup-kv`, `/bin/`, `*.test`,
`*.out`, `coverage.*`, IDE, OS) already covers the Go essentials. The [G] baseline
(repo-and-ci.md lines 66-97) suggests additions the current file lacks: `dist/`, `*.exe`,
`*.coverprofile`, `coverage.html`, `.cache/`, `.gocache/`, `.gomodcache/`, `*.swp`, `*~`,
`Thumbs.db`, `.env`.

Mandatory rule for the edit ([G] lines 55-58): **merge — do not overwrite**; the existing entries
encode project decisions.

---

### Verification-only: `internal/app/probe_violation.go` (NOT committed)

Reference [R] lines 543-561 — copied into place, gate asserted RED citing `sup-kv-app-no-io`,
removed, gate asserted GREEN. Expected verbatim finding: `import 'net' is not allowed from list
'sup-kv-app-no-io': internal/app is the pure decision core - no networking (depguard)` (the strict
config will also flag `prealloc` on the same function — both findings prove the config is live).

## Shared Patterns

### Workflow hygiene block
**Source:** [G] repo-and-ci.md lines 221-245 + 270-281; [R] Pattern 5 (lines 369-375)
**Apply to:** both workflow files (`ci.yml`, `vulncheck.yml`)

```yaml
permissions:
  contents: read
concurrency:
  group: ci-${{ github.event_name }}-${{ github.ref }}   # event_name prevents push/PR mutual cancel
  cancel-in-progress: true
# setup-go always with:
#   go-version-file: go.mod                                # Go version lives in exactly one place
```

Plus the tidy gate on native cells: `go mod tidy && git diff --exit-code go.mod go.sum` (D-06).

### Version pin set (non-negotiable, overrides all example templates)
**Source:** [R] lines 82, 93-98, Pitfall 4
**Apply to:** `go.mod`, `ci.yml`, `vulncheck.yml`

golangci-lint v2.15.0 (tool directive + action `version: v2.15.0` full patch pin) · govulncheck
v1.9.0 (tool directive) · golangci-lint-action v9 · actions/checkout@v7 · actions/setup-go@v7 ·
powerman strict config exactly 2.15.0. The [G] asset files (checkout@v4, setup-go@v5, action@v7,
`version: v2.13`, govulncheck `@latest`) are STALE for this phase — structure only, never versions.

### Tool invocation
**Source:** [G] repo-and-ci.md lines 138-159
**Apply to:** `go.mod`, `Makefile`, both workflows, AGENTS.md

Every tool pinned as a `tool` directive and invoked via `go tool <name>` — never `tools.go`,
never bare `go install`, never `go run ...@latest` in CI.

### Strict-config demands on placeholder code
**Source:** [R] Pitfall 10 (lines 445-447)
**Apply to:** `cmd/sup-kv/main.go`, both `doc.go` files, `app_test.go`

Package doc comments on every package, table-test shape, prealloc-safe loops — the first CI run
must not go red for style on placeholders.

### `master`, not `main`
**Source:** repo default branch (live-verified, [R] line 610); [G] examples use `branches: [main]`
**Apply to:** `ci.yml` trigger, ruleset `ref_name`, every doc that names the default branch

### Vendored-config provenance
**Source:** D-10 + [R] Pattern 1; [G] repo-and-ci.md lines 174-195
**Apply to:** `.golangci.yml`, AGENTS.md

`# Origin: https://github.com/powerman/golangci-lint-strict version 2.15.0` header preserved;
upstream content byte-unmodified; project rules only as additive keys inside
`linters.settings.depguard.rules`; bump = re-fetch. Never write `extends:` (silently disables
the whole config — verified).

## No Analog Found

All 13 files have no codebase analog — the repository is greenfield. The planner should build from
the [R] verified examples above (preferred) and [G] conventions (structure only), not from
codebase search.

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `go.mod` | config | declarative | No Go module exists yet |
| `cmd/sup-kv/main.go` | component | n/a | No Go source of any kind in the repo |
| `internal/port/doc.go` | component | n/a | No Go source; layout name mandated by go-ultimate service layout |
| `internal/app/doc.go` | component | n/a | No Go source; layout name mandated by go-ultimate service layout |
| `internal/app/app_test.go` | test | batch | No tests exist; stdlib `testing` table-driven per go-ultimate |
| `.golangci.yml` | config | batch | No lint config exists; [R] Pattern 1 is the executed-verified replacement |
| `.github/workflows/ci.yml` | config | event-driven | No `.github/` directory exists |
| `.github/workflows/vulncheck.yml` | config | event-driven | No `.github/` directory exists |
| `.github/dependabot.yml` | config | event-driven | No `.github/` directory exists |
| `Makefile` | config | request-response | No Makefile exists; [G] task-runner block is the template |
| `AGENTS.md` | config | n/a | Missing (go-ultimate required-file finding, [R] A7) |
| `docs/setup/` ruleset script + doc | utility/docs | request-response | Nothing like it exists; [R] Pattern 3 payload is field-verified |
| `.gitignore` (modify) | config | n/a | Merge-don't-overwrite rule applies ([G] lines 55-58) |

## Metadata

**Analog search scope:** full tracked tree (`git ls-files`) — planning docs, README, LICENSE,
.gitignore, docs/IDEA.md, .claude/CLAUDE.md only; `.claude/skills/` and `.agents/skills/` do not
exist in the repo. No Go, CI, lint, or build files anywhere.
**Tracked-source gate:** trivially satisfied — zero repo analogs named; all excerpt sources are
either phase research (`01-RESEARCH.md`, tracked) or the external mandated go-ultimate skill
(plugin cache, clearly labeled, never presented as repo analogs).
**Files scanned:** 61 tracked files (all docs/planning; none code)
**Pattern extraction date:** 2026-10-31
