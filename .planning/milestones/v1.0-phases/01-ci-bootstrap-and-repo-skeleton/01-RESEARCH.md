# Phase 1: CI Bootstrap and Repo Skeleton - Research

**Researched:** 2026-10-31
**Domain:** Go CI/CD on GitHub Actions (private repo), golangci-lint v2 strict config + depguard architecture boundary, GitHub rulesets/Dependabot automation
**Confidence:** HIGH (all load-bearing syntax and tool facts verified today by execution against go 1.27.1 + golangci-lint v2.15.0 in a scratch module, or live via `gh api` against the real repo; doc claims cite official docs fetched today)

## Summary

This phase is pure infrastructure: a zero-dependency Go module skeleton (`cmd/sup-kv`, `internal/port`, `internal/app`), a pinned toolchain via Go 1.24+ `tool` directives, a GitHub Actions gate matrix, the vendored powerman strict lint config with depguard-enforced architecture boundaries, Dependabot, a scheduled govulncheck job, and scripted branch protection. The pinned version set (golangci-lint v2.15.0, golangci-lint-action v9.3.0, checkout v7.0.1, setup-go v7.0.0, govulncheck v1.9.0) was re-verified today against proxy.golang.org and the GitHub Releases API, and the `go get -tool` directives were executed successfully on go 1.27.1.

Three findings materially change the plan. **First — blocker:** branch protection on `Djarvur/sup-kv` is currently impossible: `gh api repos/Djarvur/sup-kv/rulesets` returns HTTP 403 "Upgrade to GitHub Pro or make this repository public to enable this feature." The owner is an Organization on a plan without rulesets for private repos, so CI-03 needs a user decision (org plan upgrade, or descope, since D-09 locks the repo private). **Second:** golangci-lint v2.15.0 has **no `extends:` config mechanism** (verified three ways, including the source struct and JSON schema) — and worse, `golangci-lint run` *silently ignores the entire config file* when an `extends:` key is present. The strict config must instead be vendored into a single `.golangci.yml` with project depguard rules inserted additively under the existing `linters.settings.depguard.rules` mapping — a pattern I verified end-to-end with all three rule sets firing. **Third:** the runner landscape shifted: `macos-13` is gone, `macos-15-intel`/`macos-26-intel` exist, and `ubuntu-24.04-arm` hosted runners ARE available for private repos at $0.005/min — cheaper than x64 — so the linux/arm64 cell can be a *native* full-gate cell; D-05's stated premise ("linux arm64 unavailable for private repos") is outdated.

**Primary recommendation:** Vendor powerman 2.15.0 into a single `.golangci.yml` (Origin header intact) with the two sup-kv depguard rules inserted under `depguard.rules` (verified syntax below); run native full gates on `ubuntu-latest`, `ubuntu-24.04-arm`, `macos-15` + a darwin/amd64 cross-compile cell on Linux; gate the branch-protection task behind a `checkpoint:human-verify` for the org-plan decision (everything else lands normally).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Repo Skeleton and Module
- **D-01:** Create the full target skeleton with compileable placeholders in this phase: `cmd/sup-kv` (minimal `main`), `internal/port`, `internal/app` (package declarations / doc.go, plus one trivial test so the race gate exercises a real test target). The depguard boundary (success criterion 4) is only meaningful once these packages exist. Adapter packages appear later with the phases that need them — do not create empty adapter dirs now.
- **D-02:** Module path is `github.com/Djarvur/sup-kv` — **Reversibility:** costly — matches the existing remote; changing a module path later rewrites every import in every file.
- **D-03:** `go.mod` stays **zero-dependency** in Phase 1 (`go 1.27` directive). gRPC and protobuf enter `go.mod` in the phases that first use them (protobuf wire envelopes in Phase 4, gRPC in Phase 5). No speculative dependencies.

#### Pinned Dev Tools
- **D-04:** Pin only the tools the gate needs as `go tool` directives (Go 1.24+ pattern; never `tools.go`, never bare `go install`): golangci-lint **v2.15.0** and govulncheck **v1.9.0**. The CI lint job pins the same version (`golangci-lint-action` v9 with `version: v2.15`) so local and CI linters match. Buf CLI and protoc plugins are deferred to the first codegen phase (see Deferred).

#### CI Workflow Structure
- **D-05:** Matrix covers all four OS/arch cells using hosted runners on the **private** repo: native full gates on `ubuntu-latest` (linux/amd64) and `macos-15` (darwin/arm64) — `go vet ./...` + `go test -race ./...`; cross-compile cells for linux/arm64 and darwin/amd64 — `GOOS`/`GOARCH go build ./...` + `go vet`. Lint is a single job on ubuntu. All matrix jobs are required checks, so a red cell blocks merge.
  **Interpretation note for criterion 1:** race-enabled tests on all four cells natively is not achievable with GitHub-hosted runners on a private repo (linux arm64 hosted runners are free for public repos only; Intel macOS runners are retiring). Criterion 1 is read as: the gate matrix runs on every Linux/macOS × amd64/arm64 cell — full gates where hosted runners exist natively, compile+vet gates on cross cells. **The phase researcher MUST verify current runner availability** (macos Intel retirement timeline, private-repo arm64 options, current macos-15 label) and the planner adjusts cells accordingly. If the repo is later made public, upgrade linux/arm64 to a native `ubuntu-24.04-arm` cell.
- **D-06:** CI hygiene per go-ultimate repo-and-ci conventions: `concurrency` group with `cancel-in-progress` per ref; `setup-go` with `go-version-file: go.mod` (the Go version lives in exactly one place); tidy gate (`go mod tidy` then `git diff --exit-code go.mod go.sum`); govulncheck as a weekly scheduled cron workflow plus on dependency PRs.
- **D-07:** Actions versions from research: `actions/checkout@v7`, `actions/setup-go@v7` (research-verified current).

#### GitHub-Side Setup
- **D-08:** Branch protection is applied **scripted via `gh api`** as part of phase execution (repository Ruleset: require the CI status checks, require pull request before merging, block force pushes and deletions on `master`), with the exact commands recorded in a setup doc in the repo for re-application/audit; enable delete-branch-on-merge. `gh` is authenticated locally (account `onokonem`, write access to `Djarvur/sup-kv` confirmed). Verification asserts the ruleset via `gh api` — not by eyeballing the UI.
- **D-09:** The repo **stays PRIVATE** through this phase. Flipping to public is a user decision outside this workflow (it would unlock a free native linux/arm64 hosted runner and Dependabot alerts — see Deferred).

#### Lint Configuration
- **D-10:** Vendor the strict config byte-unmodified: fetch `powerman/golangci-lint-strict` for **exactly** golangci-lint v2.15.0 into a dedicated file with provenance documented (fetch URL + version); the project `.golangci.yml` `extends:` it and holds only project overrides. Never edit the vendored file; bump = re-fetch per version.
- **D-11:** depguard rules encode the architecture boundary: `internal/port` — stdlib only (deny third-party imports); `internal/app` — deny `net`, `os`, `syscall`, `google.golang.org/*` (the pure decision core has no I/O or infrastructure). Exact deny-list finalized by the planner against go-ultimate and the roadmap wording; a probe (a violating file making the gate red) demonstrates enforcement in verification.
- **D-12:** No `//nolint` without a written justification — project rule, enforced by the strict config's nolint handling (already part of powerman strict).

### Claude's Discretion
- Placeholder file names and content within the skeleton (doc.go vs real minimal types), workflow file naming/layout (one workflow vs split lint/test/schedule), the exact cron schedule for govulncheck, and the setup-doc location/format. All guided by go-ultimate references.

### Deferred Ideas (OUT OF SCOPE)
- **Flip repo to public** — unlocks a free native linux/arm64 hosted runner and Dependabot alerts; a user decision to revisit at/after v1.0. Not automated in this phase.
- **Buf CLI v1.75 + protoc-gen-go v1.36.12 + protoc-gen-go-grpc v1.6.2 pinning** — deferred to the first codegen phase (Phase 4 wire envelopes / Phase 5 gRPC).
- **goreleaser for 4-target release archives + checksums + SBOM** — release tooling for Phase 7 / v1.0; explicitly not blocking this phase (STACK research recommendation).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CI-01 | GitHub Actions runs the quality gates (strict lint, `go vet`, tests with race detector) on every push and PR, on Linux and macOS, amd64 and arm64 | Verified runner labels for private repos (all four cells exist natively; pricing table); verified workflow hygiene patterns (go-ultimate + tested config); required-check contexts must match matrix job names `<job> (<matrix values>)` |
| CI-02 | Dependabot keeps dependencies current (updates and alerts) and a scheduled `govulncheck` job runs against the Go vulnerability database; no open critical vulnerabilities at release | Verified dependabot.yml keys (gomod + github-actions, weekly, grouped minor+patch); govulncheck v1.9.0 tool directive executed locally; **alerts half is GHAS-gated on private repos** (see Open Questions) |
| CI-03 | Development is PR-only — branch protection on `master`, direct pushes forbidden, every change (code and docs) lands through a pull request from a feature branch | Ruleset REST payload verified; **live 403: rulesets are plan-gated on this private repo** — needs user decision before the protection task can run (see Open Questions) |
| QUAL-01 | Strict golangci-lint configuration (all applicable linters enabled; no suppressions without written justification) and go-ultimate conventions govern every package from the first feature commit; a red gate blocks merge | powerman 2.15.0 config fetched and RUN successfully on go 1.27.1; `nolintlint.require-explanation/require-specific: true` verified in the config; depguard boundary rules tested end-to-end (probe outputs below); `extends:` disproven — single-file vendoring pattern verified instead |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

`.claude/CLAUDE.md` exists (no root `CLAUDE.md`). Actionable directives relevant to this phase:

- **Quality:** strict golangci-lint configuration (all applicable linters enabled; no suppressions without written justification) and the `go-ultimate` skill conventions (github.com/Djarvur/go-ultimate) govern all implementation from the first feature commit; enforced in CI, red gate blocks merge.
- **Process:** PR-only development — branch protection on `master`, direct pushes forbidden, every change lands through a PR from a feature branch; CI on every push and PR.
- **Tech stack:** Go; single static binary; minimal dependencies — gRPC + protobuf only (later phases); HLC, UUIDv7 packing, LWW map, UDP reliability layer all in-repo.
- **Standing tooling reference:** the embedded Technology Stack section (from `.planning/research/STACK.md`) carries the machine-verified pinned versions — treat as authoritative until code supersedes it (re-verified today; one correction: `macos-13` no longer exists).
- **GSD workflow enforcement:** repo edits go through GSD entry points; do not make direct repo edits outside a GSD workflow.
- **User global instructions:** feature branches are cut from an up-to-date default branch (`git fetch origin && git pull --ff-only`, then branch) — never work directly on `master`/`main`. This aligns with CI-03.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Quality gates (lint/vet/test/build) | CI (GitHub Actions) | Local dev (`go tool`, Makefile) | Enforcement lives in CI; local runs mirror it via the same pinned `go tool` directives |
| Architecture boundary (depguard) | CI lint gate (golangci-lint) | Local lint | Machine-enforced in the lint gate so a violation is a red check, not a review comment |
| Go version source of truth | Repo `go.mod` | CI (`setup-go` `go-version-file`) | Single place (go-ultimate mandate); CI reads it |
| Tool version source of truth | Repo `go.mod` `tool` directives | CI lint action `version:` input | Same-version guarantee local vs CI |
| Branch protection / merge gating | GitHub repo settings (Rulesets API) | — | Server-side enforcement independent of CI config; scripted via `gh api` for auditability |
| Dependency currency | Dependabot (github service) | govulncheck cron workflow | Dependabot proposes updates; govulncheck scans reachability against the Go vuln DB |
| Vulnerability scanning | CI scheduled workflow (govulncheck) | Dependabot PR triggers | CVEs appear without code changes → scheduled scan is required, PR-triggered is supplementary |

## Standard Stack

### Core

| Component | Version | Purpose | Why Standard | Provenance |
|-----------|---------|---------|--------------|------------|
| Go | 1.27.1 (directive `go 1.27`) | Language + toolchain (`tool` directives need 1.24+) | go-ultimate: applications pin latest stable; verified locally | [VERIFIED: local `go version`] |
| golangci-lint | v2.15.0 (2026-10-16, latest) | Lint gate | powerman strict config publishes exactly 2.15.0; `go get -tool` executed successfully today | [VERIFIED: proxy.golang.org + executed] |
| govulncheck | v1.9.0 (latest) | Scheduled vuln scan | Official Go tool; call-graph reachability = low noise | [VERIFIED: proxy + executed] |
| golangci-lint-action | v9.3.0 (2026-06-29, latest) | CI lint job | v9 pairs with golangci-lint v2; `verify: true` default validates the merged config against the JSON schema in CI for free | [VERIFIED: GitHub Releases API + README] |
| actions/checkout | v7 (v7.0.1 latest) | CI checkout | Locked D-07; re-verified today | [VERIFIED: GitHub Releases API] |
| actions/setup-go | v7 (v7.0.0 latest) | CI Go toolchain from `go-version-file: go.mod` | Locked D-07; re-verified today | [VERIFIED: GitHub Releases API] |
| powerman/golangci-lint-strict | 2.15.0 | Strict lint config to vendor | "All applicable linters enabled" without hand-assembling; per-exact-version publishing | [VERIFIED: raw file fetched and run] |

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| GitHub hosted runners | labels below | Gate matrix | Always (no self-hosted infra wanted) |
| Dependabot | config v2 | gomod + github-actions updates | Always; grouped minor+patch weekly |
| `gh` CLI | 2.103.3 | Ruleset scripting + verification | Branch-protection task and phase verification |

**Installation (verified by execution on go 1.27.1):**

```bash
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
go get -tool golang.org/x/vuln/cmd/govulncheck@v1.9.0
```

Resulting `go.mod` shape (verbatim from the executed scratch module):

```
module example.com/tooltest

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

**Consequence the planner must accept (D-03 vs D-04):** `tool` directives drag golangci-lint's full module graph (~200 `// indirect` requires) into the main `go.mod`/`go.sum`. "Zero-dependency" can only mean *zero direct runtime requires* — the shipped build graph stays empty. go-ultimate's alternative (a separate `tools/go.mod`) is NOT taken because D-04 locks tool directives in the module (and `go tool` from the repo root is what CI/Makefile invoke). This is fine for an application ("the churn is usually acceptable and one module is simpler") — but the phase doc should state it so "zero-dependency" isn't misread from the go.mod face.

**Version verification (2026-10-31):** `go list -m -versions github.com/golangci/golangci-lint/v2` → latest v2.15.0; `go list -m golang.org/x/vuln@latest` → v1.9.0; GitHub Releases API → checkout v7.0.1, setup-go v7.0.0, golangci-lint-action v9.3.0. All match the locked pins.

## Package Legitimacy Audit

This phase installs **no runtime packages** (D-03). The only external code entering the module graph is the two pinned dev tools; both were verified today on the authoritative Go registries, not via web search:

| Package | Registry | Age | Source Repo | Verdict | Disposition |
|---------|----------|-----|-------------|---------|-------------|
| github.com/golangci/golangci-lint/v2 v2.15.0 | proxy.golang.org | mature project (v2 line since 2025-03) | github.com/golangci/golangci-lint | OK | Approved |
| golang.org/x/vuln v1.9.0 | proxy.golang.org | mature (official golang.org/x) | go.googlesource.com/vuln (github.com/golang/vuln mirror) | OK | Approved |
| powerman/golangci-lint-strict 2.15.0 | raw.githubusercontent.com (config file, not a module) | per-version configs since 1.60 | github.com/powerman/golangci-lint-strict | OK | Approved (vendored as data, never executed at fetch time) |

The `gsd-tools package-legitimacy check` seam covers npm/pypi/crates only; for Go modules the equivalent verification (registry existence + canonical source repo + exact version resolution + successful `go get -tool` execution) was performed directly. **Packages removed/flagged: none.**

## Architecture Patterns

### System Architecture Diagram

```
                      developer machine
   ┌──────────────────────────────────────────────┐
   │ feature branch (from up-to-date master)      │
   │    go tool golangci-lint run ./...           │
   │    go vet ./... && go test -race ./...       │
   │    go mod tidy && git diff --exit-code       │
   └──────────────────┬───────────────────────────┘
                      │ push feature branch + open PR
                      ▼
        GitHub (private repo Djarvur/sup-kv)
                      │
        ┌─────────────┴──────────────────────────────────────────┐
        │ Ruleset on master (plan-gated, see Open Questions Q1)  │
        │  requires: pull_request, required_status_checks,       │
        │            non_fast_forward, deletion-blocked          │
        └─────────────┬──────────────────────────────────────────┘
                      │ PR triggers workflows
                      ▼
   .github/workflows/ci.yml
   ┌─────────────────────────────────────────────────────────────┐
   │ lint (ubuntu-latest): golangci-lint-action v9               │
   │   └─ version: v2.15.0 (== go.mod tool pin)                  │
   │   └─ reads .golangci.yml = vendored powerman 2.15.0          │
   │        + sup-kv depguard rules (arch boundary)               │
   │                                                              │
   │ gate matrix (all required checks):                           │
   │  ubuntu-latest      → vet + test -race        (native)       │
   │  ubuntu-24.04-arm   → vet + test -race        (native, $0.005/min) │
   │  macos-15           → vet + test -race        (native, $0.062/min) │
   │  ubuntu-latest      → GOOS=darwin GOARCH=amd64 build + vet  │
   │                        (cross cell; avoids 10x Intel-mac cost) │
   │  + tidy gate: go mod tidy && git diff --exit-code go.mod go.sum │
   └─────────────┬───────────────────────────────────────────────┘
                 │ all green?
                 ▼
   .github/workflows/vulncheck.yml (cron weekly + dependency PRs)
   ┌─────────────────────────────────────────────────────────────┐
   │ go tool govulncheck ./...  → Go vulnerability database       │
   └─────────────────────────────────────────────────────────────┘
                 │
   .github/dependabot.yml → weekly grouped minor+patch PRs
   (gomod + github-actions ecosystems; updates work on private,
    alerts are GHAS-gated — documented, not assertable)

   Repo skeleton enforced by the lint gate from commit #1:
   cmd/sup-kv/main.go · internal/port (stdlib-only) · internal/app (no I/O)
```

### Recommended Project Structure

```
.github/
├── workflows/
│   ├── ci.yml            # lint + gate matrix + tidy gate (one workflow; split is fine too — discretion)
│   └── vulncheck.yml     # schedule + dependency-PR triggers
├── dependabot.yml
cmd/
└── sup-kv/
    └── main.go           # minimal main; no testable logic (go-ultimate §package main)
internal/
├── app/
│   ├── doc.go            # package comment (strict revive demands it — verified)
│   └── app_test.go       # one trivial test so -race has a real target (D-01)
├── port/
│   └── doc.go            # placeholder; adapter dirs are NOT created now (D-01)
.golangci.yml             # vendored powerman 2.15.0 (Origin header intact) + sup-kv depguard rules
go.mod                    # module github.com/Djarvur/sup-kv; go 1.27; tool directives; no direct requires
Makefile                  # lint/test/generate/build targets (go-ultimate task runner; CI runs same targets)
AGENTS.md                 # go-ultimate required file (conventions + deviations) — currently MISSING in repo
docs/
└── setup/                # (discretion) recorded gh api ruleset commands for re-application/audit
```

### Pattern 1: Single-file vendored strict config + additive project rules (VERIFIED)

**What:** golangci-lint v2.15.0 has exactly these top-level config keys — `version`, `run`, `output`, `linters`, `issues`, `severity`, `formatters` — and no inheritance mechanism ([VERIFIED: golangci-lint v2.15.0 source `pkg/config/config.go:29-40` in module cache; the bundled JSON schema rejects any other key]). Therefore `.golangci.yml` = the vendored powerman 2.15.0 content byte-unmodified (its own `# Origin:` header preserved as provenance) with the sup-kv depguard rules inserted as *new keys under the existing* `linters.settings.depguard.rules` *mapping*, behind a marked comment.

**When to use:** always in this phase (there is no working alternative — see Pitfall 1).

**Verified example** (executed; `config verify` clean; all three rules fired):

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
            # ... more upstream entries ...
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
            - $gostd                   # NOTE: also denies internal/* and the own module path — see Pitfall 5
```

Verified probe output (go 1.27.1, golangci-lint v2.15.0 via `go tool`):

```
internal/app/randprobe.go:3:8: import 'math/rand' is not allowed from list 'main': use math/rand/v2 (depguard)
internal/app/violating.go:3:8: import 'net' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no networking (depguard)
internal/port/port.go:6:2: import 'golang.org/x/mod/semver' is not allowed from list 'sup-kv-port-stdlib-only' (depguard)
```

### Pattern 2: depguard v2 semantics (source-verified, depguard v2.2.1 as bundled with golangci-lint v2.15.0)

From the depguard source in the module cache (`github.com/!open!pee!dee!p/depguard/v2@v2.2.1`):

- `whichLists` collects **all** rules whose `files` match — multiple rules stack per file (`settings.go:205-213`).
- `fileMatch` globs are matched against the file path as recorded in the Fset — effectively an **absolute path** (`depguard.go:88`, `filepath.ToSlash(pass.Fset.Position(...).Filename)`) — hence globs must start with `**/` (`settings.go:133-137`).
- Variable expansion (`internal/utils/variables.go`): `$all` → `**/*.go`; `$test` → `**/*_test.go`; `$gostd` → top-level directory names of `GOROOT/src` (so `net`, `os`, `syscall` ARE stdlib — allowed for `internal/port` by design).
- Deny/allow entries: plain string = **prefix** match; entry ending in `$` = **exact** match. There are no `^` anchors (`settings.go:224-235`, `strInPrefixList` trims `$` and prefix-compares).
- `list-mode: strict` = allow-list only; `lax`/`original` = deny-based.

### Pattern 3: Ruleset via `gh api` (payload verified against docs.github.com REST reference)

```bash
gh api --method POST repos/Djarvur/sup-kv/rulesets --input ruleset.json
# ruleset.json — field names verified from docs.github.com/en/rest/repos/rules:
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
# 201 = created; 422 = validation error. Verification (read side):
gh api repos/Djarvur/sup-kv/rulesets --jq '.[] | {name, enforcement, rules: [.rules[].type]}'
# Delete-branch-on-merge (field exists on the repo today, currently false — live-verified):
gh api --method PATCH repos/Djarvur/sup-kv -f delete_branch_on_merge=true
```

The `context` strings must equal the exact check-run names the workflow produces. For matrix jobs the check run is named `<job-name> (<matrix values, comma-separated>)` — pin job `name:` explicitly and update the ruleset whenever the matrix changes (Pitfall 6).

### Pattern 4: Dependabot config (keys verified from the official options reference)

`.github/dependabot.yml`:

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

### Pattern 5: Workflow hygiene (go-ultimate repo-and-ci, mandated by D-06)

- `permissions: contents: read` at workflow top (default token is over-privileged).
- `concurrency: group: ci-${{ github.event_name }}-${{ github.ref }}` + `cancel-in-progress: true` (include `event_name` so a push and a PR for the same ref don't cancel each other).
- `setup-go` with `go-version-file: go.mod`.
- Tidy gate: `go mod tidy && git diff --exit-code go.mod go.sum` (D-06's locked form; `go mod tidy -diff` is the Go 1.23+ one-liner equivalent).
- Makefile standardizes entry points and CI runs the same targets: `lint: ; go tool golangci-lint run ./...`, `test: ; go test -race -cover ./...`.

### Anti-Patterns to Avoid

- **`extends:` in `.golangci.yml`** — does not exist in v2.15.0 and *silently disables the whole config* (Pitfall 1). Never write it.
- **Relative depguard `files:` globs without leading `**/` or without quotes** — rule never matches / YAML parse error (Pitfalls 2-3).
- **`^net$`-style regex anchors in depguard deny entries** — entries are prefix or `suffix$`-exact, never regex; `^net$` matches nothing (verified).
- **`tools.go` blank-import hack or bare `go install`** — replaced by `tool` directives (locked D-04).
- **Unpinned linter in CI** (`version: latest` or omitted) — local green / CI red with no reproduction.
- **Hardcoded `go-version:` in workflows** — diverges from `go.mod`; use `go-version-file`.
- **Requiring matrix checks by job id instead of full display name** — contexts must be `gate (macos-15)`-style strings.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Strict linter enable-list | Hand-assembled "all applicable linters" list | Vendored powerman/golangci-lint-strict 2.15.0 | ~100 linters with per-linter settings; powerman maintains it per exact version; hand lists go stale |
| Import boundary enforcement | CI script grepping imports / custom AST tool | depguard v2 rules in the strict config | Battle-tested glob/prefix matching, per-file rules, suggestions in diagnostics; verified working here |
| Vulnerability scanning | Ad-hoc `go list -m all` + OSV API script | `govulncheck` (tool directive) | Call-graph reachability (not every advisory), official Go tool, DB maintained by the Go team |
| Branch protection clicking | Manual UI configuration | `gh api` POST /rulesets scripted + setup doc | Auditable, re-appliable, verifiable-by-assertion (D-08) |
| CI caching of Go modules | Custom cache steps | `setup-go` built-in module caching | Maintained upstream; keys handled correctly for go.sum hashes |
| Release archives/checksums/SBOM | Makefile release target | goreleaser — **deferred to Phase 7 by decision** | Explicitly out of scope (Deferred) |

**Key insight:** every enforcement mechanism in this phase exists as maintained upstream tooling; the phase's real work is *wiring and verifying*, plus exactly one piece of project-specific content — the depguard rules encoding the architecture boundary.

## Common Pitfalls

### Pitfall 1: `extends:` silently disables the entire lint config (CRITICAL, verified)
**What goes wrong:** A `.golangci.yml` containing `extends:` (any form) passes `golangci-lint run` with **zero issues reported** — the config is silently not applied; the strict gate quietly becomes the default (much weaker) linter set. Only `golangci-lint config verify` catches it: `jsonschema: "" does not validate with "/additionalProperties": additional properties 'extends' not allowed`.
**Why it happens:** golangci-lint's run-time config loading ignores unknown top-level keys; `extends:` is not a v2.15.0 key (verified in source `pkg/config/config.go` — the struct has exactly `version`, `run`, `output`, `linters`, `issues`, `severity`, `formatters`).
**How to avoid:** Single-file vendored config with additive rules (Pattern 1); run `go tool golangci-lint config verify` in CI (the golangci-lint-action does it by default via `verify: true`) AND assert the gate actually fires with a probe file during verification.
**Warning signs:** lint suddenly reports ~0 issues after a config change; `config verify` failing while `run` is green.

### Pitfall 2: depguard `files:` globs don't match (verified)
**What goes wrong:** Rules with `files: [internal/app/**]` or `files: [internal/app/]` never fire — silence, no error.
**Why:** globs are matched against the **absolute** file path (source-verified), so a relative pattern can't match; directory-prefix strings aren't globs at all.
**How to avoid:** `'**/internal/app/**'` — quoted (leading `*` breaks YAML) and with the leading `**/`. `$all`/`$test` expand to `**/*.go` / `**/*_test.go`.
**Warning signs:** the D-11 probe fails to make the gate red.

### Pitfall 3: YAML duplicate top-level key when appending overrides
**What goes wrong:** Appending a second `linters:` block to the vendored file → `yaml: unmarshal errors: mapping key "linters" already defined` — config won't load at all.
**How to avoid:** insert project rules *inside* the existing `linters.settings.depguard.rules` mapping (new map keys), not a new top-level section (verified working).

### Pitfall 4: `golangci-lint-action` version drift vs the `go tool` pin
**What goes wrong:** `version: v2.15` resolves the *latest patch* of v2.15 — if v2.15.1 ships with behavior changes, CI lints differently from the local `go tool` v2.15.0.
**How to avoid:** pin the full version `version: v2.15.0` (the action documents full `v2.3.4`-style values) — same-decision refinement of D-04's intent ("local and CI linters match"). Alternatively bypass the action and run `go tool golangci-lint run ./...` as a plain step (loses annotations/caching).

### Pitfall 5: the port-stdlib-only rule denies the own module path too
**What goes wrong:** `list-mode: strict` + `allow: [$gostd]` also denies imports of `github.com/Djarvur/sup-kv/...` (own module) — a port package referencing app types fails the gate.
**Why:** verified semantics: `$gostd` = GOROOT/src top-level dirs only; strict mode is allow-list-only.
**How to avoid:** decide per D-11's wording ("deny **third-party** imports"): if port may reference project types, use `allow: [$gostd, github.com/Djarvur/sup-kv]` (planner finalization point). Also note `!$test` exempts test files from the boundary — decide if intended (tests legitimately need `os` for temp files).

### Pitfall 6: required-status-check contexts vs matrix job names
**What goes wrong:** ruleset contexts like `gate` don't match the check runs a matrix produces (`gate (macos-15)`); PRs show stuck "Expected" checks and cannot merge — or worse, nothing is actually required.
**How to avoid:** contexts must be the exact check-run names `<job-name> (<matrix values comma-separated>)`; give every matrix job an explicit `name:`; treat any matrix change as a ruleset update (record in the setup doc). Verified behavior (community-documented; GitHub docs).

### Pitfall 7: rulesets are plan-gated on private repos (LIVE BLOCKER)
**What goes wrong:** `gh api repos/Djarvur/sup-kv/rulesets` → 403 `Upgrade to GitHub Pro or make this repository public to enable this feature.` Branch protection cannot be created on this repo as-is.
**Why:** free-plan enforcement is public-repo-only; owner `Djarvur` is an Organization (live-verified type) — needs Team plan (or the personal-account equivalent for user-owned repos).
**How to avoid:** surface as a user decision (checkpoint) BEFORE the branch-protection task; everything else in the phase is unaffected. See Open Questions Q1.
**Warning signs:** any 403 mentioning plan upgrade on a protection endpoint.

### Pitfall 8: pushing workflow files with the wrong token scope
**What goes wrong:** commits adding `.github/workflows/*.yml` are rejected unless the credential has the `workflow` scope.
**How to avoid:** the local `gh` token (onokonem) already has `repo` + `workflow` (live-verified) — use it for the branch pushes in this phase.

### Pitfall 9: Dependabot *alerts* expectation on a private repo
**What goes wrong:** CI-02 says "updates and alerts"; alerts require a GHAS/Code Security license on private repos — not purchasable for this org tier in scope here; verification would fail if it asserted alerts.
**How to avoid:** verification asserts what is assertable: dependabot.yml present/valid, version-update PRs arriving, govulncheck green; document alerts as plan-gated (unlocked by the deferred public flip). See Open Questions Q2.

### Pitfall 10: strict config demands on placeholder code
**What goes wrong:** the *first* CI run goes red for style: the strict config enforces package comments (`revive: package-comments`), prealloc, etc. — verified firing on trivial two-line placeholder files.
**How to avoid:** write placeholders that already satisfy the gate (package doc comments on every package, table-test shape for the trivial test), or expect the first iteration to be lint-fix commits. Also: `run.timeout: 1m` (upstream) — fine for this phase's tiny targets.

## Code Examples

### Workflow: gate matrix + lint + tidy (verified components assembled)

```yaml
# Source: go-ultimate repo-and-ci.md minimum workflow, adapted to D-05/D-06/D-07
# with runner labels verified 2026-10-31 (docs.github.com runners reference)
name: CI
on:
  push:
    branches: [master]
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
          version: v2.15.0        # full patch pin == go.mod tool directive
          # verify: true is the default — validates .golangci.yml against the schema

  gate:                            # required checks are named "gate (<cell>)"
    strategy:
      fail-fast: false             # a red cell must not cancel the others' evidence
      matrix:
        include:
          - name: ubuntu-latest          # linux/amd64 native — full gate
          - name: ubuntu-24.04-arm       # linux/arm64 native — full gate ($0.005/min, cheaper than x64)
          - name: macos-15               # darwin/arm64 native — full gate ($0.062/min)
          - name: darwin-amd64-cross     # cross-compile cell on Linux (avoids 10x Intel-mac rate)
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

### Workflow: scheduled govulncheck (cron time is discretion; schedule + dep-PR triggers per D-06)

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

### Depguard boundary probe (verification artifact — makes the gate red on purpose)

```go
// internal/app/probe_violation.go — NOT committed to master; used in verification:
// copied into place, gate asserted RED, removed, gate asserted GREEN.
package app

import "net"

// ConnNames violates the sup-kv-app-no-io boundary on purpose.
func ConnNames() []string {
	var out []string
	for _, a := range []net.Addr{nil} {
		out = append(out, a.String())
	}
	return out
}
```

Expected red output (verified verbatim today): `import 'net' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no networking (depguard)` — note the strict config will *also* flag `prealloc` on this exact function; both findings prove the config is live.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `macos-13` runner | retired; Intel macOS now `macos-15-intel` / `macos-26-intel` | by 2026-10 (absent from runner-images table today) | STACK.md's example matrix (`macos-13`) is stale; darwin/amd64 = `macos-15-intel` (native, $0.062/min) or cross-compile |
| "linux arm64 hosted runners = public repos only" | `ubuntu-24.04-arm`/`-22.04`/`-26.04-arm` available for private repos, $0.005/min (SKU `actions_linux_arm`) — *cheaper* than x64 $0.006 | verified today (docs + pricing reference) | D-05's interpretation-note premise is outdated; a native linux/arm64 full-gate cell is available and cheaper than the amd64 cell |
| Classic branch protection | Repository Rulesets (REST `POST /repos/{owner}/{repo}/rulesets`) | GA since 2023; classic is legacy | D-08's ruleset approach is the current mechanism; both remain plan-gated on private repos |
| `tools.go` blank imports | `go 1.24+` `tool` directives in go.mod | Go 1.24 (2025-02) | locked D-04; executed-verified here |
| golangci-lint v1 config schema | `version: "2"` schema (linters/formatters split); config verified against a bundled JSON schema | v2.0.0 (2025-03) | powerman publishes per-v2-version configs; mismatch fails to parse or silently drops sections |

**Deprecated/outdated to avoid:** `macos-13` label (gone); `extends:` config key (never existed in v2 — web claims to the contrary are wrong, disproven by execution); `golangci-lint-action` v7/v8 with v2 linter (v9 is the v2-compatible major); `go-version:` hardcoded in workflows.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Proposed matrix (3 native + 1 cross) is the planner's adjustment of D-05 given verified runner facts; linux/arm64 native cell adopted (cheaper), darwin/amd64 stays cross to avoid 10x macOS rate | Architecture Patterns | Low — worst case a cell label needs changing; all labels verified to exist today |
| A2 | `version: v2.15.0` (full patch) is an acceptable refinement of D-04's `version: v2.15` | Pitfall 4 | Low — both documented-valid; full pin better matches D-04's stated intent |
| A3 | One CI workflow + one vulncheck workflow (file layout is discretion) | Project Structure | None — discretion area |
| A4 | Weekly Monday 06:00 UTC cron for govulncheck (exact time is discretion) | Code Examples | None — discretion area |
| A5 | Ruleset `required_approving_review_count: 1` and review-thread settings — chosen example values, not user-locked | Pattern 3 | Low — single-maintainer repo may want 0 approvals + still PR-only; planner decides with user |
| A6 | `!$test` exemption on boundary rules (tests may import `os`/`net` for helpers) matches D-11 intent | Pattern 1 | Low — flip to `$all` if tests must also be boundary-checked |
| A7 | `AGENTS.md` should be created this phase (go-ultimate required-files list; repo currently lacks it) | Project Structure | Low — a review finding if missing |
| A8 | Port rule should allow the own module path (`github.com/Djarvur/sup-kv`) in addition to `$gostd`, per D-11's "deny third-party" wording | Pitfall 5 | Medium — wrong choice blocks legitimate port↔app type references in later phases |

## Open Questions (RESOLVED)

All four questions carry adopted dispositions as of plan finalization (2026-10-31); one-line status per question below.

1. **Branch protection is blocked by the GitHub plan (Q1 — affects CI-03)** — RESOLVED: routed to plan 04 Task 2 as a blocking `checkpoint:decision` (apply-now vs defer); everything else lands first, and the defer path records the scripted commands as pending in the setup doc.
   - What we know: live 403 on `repos/Djarvur/sup-kv/rulesets` — "Upgrade to GitHub Pro or make this repository public"; owner is an Organization; D-09 locks the repo private; D-08 requires scripted rulesets.
   - What's unclear: whether the user will upgrade the org plan (Team) for this repo.
   - Recommendation: plan the phase so everything except protection lands first (workflows, skeleton, lint boundary, Dependabot, govulncheck); make the ruleset task a `checkpoint:human-verify` — apply immediately if the plan is upgraded by then, otherwise record the scripted commands in the setup doc as "pending plan upgrade". CI-03's verification can assert the *workflows + PR flow* while flagging protection as plan-gated. Alternatives: flip public (locked out by D-09) or descope protection (violates CI-03 — needs explicit user sign-off).
2. **Dependabot alerts interpretation (affects CI-02 verification wording)** — RESOLVED: plan 03 handles it — verification asserts updates + govulncheck and documents alerts as GHAS-gated/plan-gated, unlocked by the deferred public flip.
   - What we know: version updates work on private repos; alerts need GHAS — unavailable here.
   - Recommendation: verify updates + govulncheck; document alerts as unlocked-by-public-flip (already a Deferred item). Success criterion 3 should be read as "Dependabot updates active; alerts plan-gated".
3. **linux/arm64 native cell (D-05 adjustment)** — RESOLVED: D-05 amended (2026-10-31) — the native `ubuntu-24.04-arm` full-gate cell is adopted; the matrix is 3 native cells + darwin/amd64 cross.
   - What we know: `ubuntu-24.04-arm` is available for private repos and cheaper than x64; D-05's premise was "unavailable for private".
   - Recommendation: adopt the native cell now (D-05's own "if public, upgrade to ubuntu-24.04-arm" note shows that was always the desired end state; it just turns out to be available and cheap while private). Planner confirms.
4. **macos-15 vs macos-26 for the darwin/arm64 cell** — RESOLVED: `macos-15` kept per locked D-05 (explicit stable label); revisit when macOS 15 approaches retirement.
   - What we know: D-05 locks `macos-15` (still available, verified); `macos-latest` now aliases `macos-26`.
   - Recommendation: keep `macos-15` as locked (explicit label, stable); revisit when macOS 15 approaches retirement.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | skeleton, tool directives, local gates | ✓ | 1.27.1 linux/amd64 (matches `go 1.27` directive) | — |
| `gh` CLI | ruleset scripting, verification | ✓ | 2.103.3, authed as onokonem (active, scopes `repo`+`workflow`) | — |
| Git + SSH remote | branch workflow | ✓ | remote `github.com:Djarvur/sup-kv`, default branch `master` | — |
| proxy.golang.org reachability | tool directives, CI | ✓ | resolved + ran golangci-lint v2.15.0 / govulncheck v1.9.0 locally today | — |
| raw.githubusercontent.com | vendoring powerman config | ✓ | fetched today | — |
| GitHub Actions hosted runners | gate matrix | ✓ | labels verified (ubuntu-latest, ubuntu-24.04-arm, macos-15) | — |
| Rulesets API | branch protection (CI-03) | ✗ | 403 plan-gated on this private repo | checkpoint for org-plan decision (Q1) |
| Dependabot alerts | CI-02 "alerts" | ✗ | GHAS-gated on private repos | version updates + govulncheck; alerts via future public flip |

**Missing dependencies with no fallback:** none that block the phase start — the rulesets blocker (Q1) has a fallback shape (defer the protection task behind a human checkpoint) but no technical fallback.

**Missing dependencies with fallback:** rulesets (plan upgrade or defer), Dependabot alerts (document as gated).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (go-ultimate mandate: vanilla testing, table-driven, no assertion library) |
| Config file | none needed (no framework config for stdlib testing) |
| Quick run command | `go test -race ./... && go vet ./...` |
| Full suite command | `go tool golangci-lint run ./... && go vet ./... && go test -race ./... && go build ./... && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./... && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CI-01 | Matrix gates run on all 4 cells; red blocks merge | e2e (CI) | `gh api repos/Djarvur/sup-kv/commits/<sha>/check-runs --jq '.check_runs[] | {name, conclusion}'` after a PR; assert all four `gate (...)` contexts present and a deliberately-red probe PR is blocked | ❌ created this phase (workflow files) |
| CI-02a | Dependabot version updates active | e2e (service) | `gh api repos/Djarvur/sup-kv/contents/.github/dependabot.yml` (present) + observe first update PR within schedule window | ❌ created this phase |
| CI-02b | Scheduled govulncheck green | e2e (CI) | `gh run list --workflow=vulncheck.yml` (or trigger via workflow_dispatch added for testability) and assert success | ❌ created this phase |
| CI-03 | Direct pushes to master rejected; PRs required | e2e (git+API) | `gh api repos/Djarvur/sup-kv/rulesets --jq '.[].enforcement'` = active **(plan-gated, Q1)**; probe: attempt `git push origin master:master` from a non-bypass token → expect rejection | ❌ blocked by plan (checkpoint) |
| QUAL-01 | Strict config live; boundary machine-enforced | integration (local, fast) | depguard probe: create `internal/app/probe_violation.go` (example above) → `go tool golangci-lint run ./internal/...` exits non-zero citing `sup-kv-app-no-io`; remove → exit 0. Also `go tool golangci-lint config verify` passes on the merged config | ❌ probe script created this phase (verification-time, not committed to master) |
| QUAL-01 (support) | Placeholders satisfy the strict gate | unit (trivial) | `go test -race ./...` runs the one `internal/app` test from D-01 | ❌ created this phase (D-01 test) |

### Sampling Rate
- **Per task commit:** `go test -race ./... && go vet ./... && go tool golangci-lint run ./...`
- **Per wave merge (PR):** the CI workflow itself — full matrix + lint + tidy gate (that is the point of the phase)
- **Phase gate:** full suite green + the depguard probe demonstration + `gh api` assertions above before `/gsd:verify-work`

### Wave 0 Gaps
- `internal/app/app_test.go` — the D-01 trivial test (race-gate target). No framework install needed; no conftest equivalent in Go.

## Security Domain

`security_enforcement: true`, ASVS L1. This phase handles no user input, no secrets, no crypto — the applicable surface is CI supply-chain and repository access control.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | no authn in scope (CI uses GitHub's OIDCO-less default token; no third-party credentials) |
| V3 Session Management | no | — |
| V4 Access Control | yes | Ruleset on `master` (require PR, block force-push/deletion) = the repository access-control control; plan-gated (Q1). `GITHUB_TOKEN` scoped to `permissions: contents: read` |
| V5 Input Validation | marginal | no runtime inputs; workflow triggers use pinned `branches`/`paths` filters, no `pull_request_target`, no dynamic interpolation into `run:` from untrusted PR content |
| V6 Cryptography | no | none in phase |
| V14 Config (supply chain) | yes | Actions pinned to major versions verified today (checkout v7.0.1, setup-go v7.0.0, golangci-lint-action v9.3.0); tools pinned by exact version via `tool` directives; Dependabot keeps actions+modules current; govulncheck scans the Go vuln DB; tidy gate prevents stealth dependency drift |

### Known Threat Patterns for CI-on-private-repo

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious/moved action tag | Tampering, Elevation | Major-tag pinning (go-ultimate floor; verified current majors); optional commit-SHA pinning for any future secret-handling action |
| Over-privileged workflow token | Elevation | `permissions: contents: read` per workflow; no `pull_request_target` |
| Workflow injection via PR metadata | Tampering | No `run:` step interpolates `github.event.*` untrusted fields (head_ref, PR title); matrix values are static literals |
| Dependency confusion / slopsquatting | Tampering | Zero runtime deps (D-03); `tool` directives pinned to canonical upstream paths verified on proxy.golang.org; govulncheck + Dependabot as detectors |
| Config-swap weakening the gate (e.g., silent `extends:` no-op) | Repudiation | `config verify` (action default) + the red/green depguard probe in phase verification (Pitfall 1) |
| Ruleset drift after matrix change | Elevation | Setup doc records re-application commands; verification re-asserts contexts via `gh api` (D-08) |

## Sources

### Primary (HIGH confidence — executed or live-verified today)
- Scratch-module execution on go 1.27.1 — `go get -tool` for golangci-lint v2.15.0 + govulncheck v1.9.0; `go tool golangci-lint config verify`; `run` with `extends:` (silent no-op) vs vendored config (strict findings); depguard boundary probes (all three rules firing) — the basis for Patterns 1-2 and Pitfalls 1-3, 5
- golangci-lint v2.15.0 source in module cache — `pkg/config/config.go` (top-level keys), bundled JSON schema (no `extends`); depguard v2.2.1 `depguard.go`/`settings.go`/`internal/utils/variables.go` (file-match semantics, macro expansion, prefix/exact matching)
- powerman/golangci-lint-strict `2.15.0/.golangci.yml` — fetched raw; Origin header, `version: "2"`, depguard rules, `nolintlint.require-explanation/require-specific`
- Live `gh api` — repo settings (private, master, delete_branch_on_merge=false), auth status (onokonem, repo+workflow), rulesets 403 message, owner type=Organization, action latest releases
- docs.github.com (fetched today) — hosted-runners reference (all labels incl. `ubuntu-24.04-arm`, `macos-15-intel`, `macos-latest`→macos-26), actions-runner-pricing ($0.006/$0.005/$0.062/$0.010 per min), REST rules reference (ruleset payload), Dependabot options reference
- actions/runner-images README (fetched today) — macos-13 absent from available images; current label list
- golangci-lint-action README (fetched today) — v9 inputs incl. `install-mode`, `verify` default, full-version values
- proxy.golang.org via `go list -m` — golangci-lint v2.15.0 latest, x/vuln v1.9.0 latest

### Secondary (MEDIUM confidence)
- Matrix check-run naming `<job> (<matrix values>)` and required-check brittleness — GitHub community discussions #33579, "The Required GitHub Status Check That Wasn't" (consistent across sources; not asserted in official REST docs page text)
- Rulesets/branch-protection plan gating — community discussions #190190, #174400 corroborating the live 403 (the 403 itself is primary)
- Dependabot alerts GHAS requirement — docs.github.com about-dependabot-alerts + corroborating blog/community posts

### Tertiary (LOW confidence)
- None material. (A web-search claim that `extends:` exists was **disproven by execution** — treated as refuted, not sourced.)

## Metadata

**Confidence breakdown:**
- Standard stack / pinned versions: HIGH — every version re-verified today against its authoritative registry; tool directives executed
- Lint config architecture: HIGH — the exact pattern this phase must ship was run end-to-end today with probe evidence; the locked `extends:` approach is disproven with a verified replacement inside the decisions' intent
- CI matrix / runners: HIGH — official docs fetched today + pricing pinned; one locked premise (arm64-for-private) corrected with data
- Branch protection: HIGH on facts (live 403), but execution itself is blocked pending a user decision (Q1)
- Pitfalls: HIGH — the top four were reproduced or disproven by execution, not inferred

**Research date:** 2026-10-31
**Valid until:** 2026-11-30 (runner labels/pricing and golangci-lint releases move slowly; re-check if execution slips past ~30 days — especially action major versions and macOS image availability)
