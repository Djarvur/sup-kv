# Phase 1: CI Bootstrap and Repo Skeleton - Context

**Gathered:** 2026-10-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Quality gates and the machine-enforced repo skeleton exist **before any feature code**: GitHub Actions running strict golangci-lint + `go vet` + race-enabled tests across the Linux/macOS × amd64/arm64 matrix, Dependabot and a scheduled `govulncheck` job, PR-only development enforced by branch protection on `master`, and the depguard-enforced architecture boundary (`internal/app` free of `net`/`os`/grpc, `internal/port` stdlib-only). Covers requirements CI-01, CI-02, CI-03, QUAL-01. No feature code: the store engine, persistence, replication, and API all belong to later phases.

</domain>

<decisions>
## Implementation Decisions

### Repo Skeleton and Module
- **D-01:** Create the full target skeleton with compileable placeholders in this phase: `cmd/sup-kv` (minimal `main`), `internal/port`, `internal/app` (package declarations / doc.go, plus one trivial test so the race gate exercises a real test target). The depguard boundary (success criterion 4) is only meaningful once these packages exist. Adapter packages appear later with the phases that need them — do not create empty adapter dirs now.
- **D-02:** Module path is `github.com/Djarvur/sup-kv` — **Reversibility:** costly — matches the existing remote; changing a module path later rewrites every import in every file.
- **D-03:** `go.mod` stays **zero-dependency** in Phase 1 (`go 1.27` directive). gRPC and protobuf enter `go.mod` in the phases that first use them (protobuf wire envelopes in Phase 4, gRPC in Phase 5). No speculative dependencies.

### Pinned Dev Tools
- **D-04:** Pin only the tools the gate needs as `go tool` directives (Go 1.24+ pattern; never `tools.go`, never bare `go install`): golangci-lint **v2.15.0** and govulncheck **v1.9.0**. The CI lint job pins the same version (`golangci-lint-action` v9 with `version: v2.15`) so local and CI linters match. Buf CLI and protoc plugins are deferred to the first codegen phase (see Deferred).

### CI Workflow Structure
- **D-05:** Matrix covers all four OS/arch cells using hosted runners on the **private** repo: native full gates on `ubuntu-latest` (linux/amd64) and `macos-15` (darwin/arm64) — `go vet ./...` + `go test -race ./...`; cross-compile cells for linux/arm64 and darwin/amd64 — `GOOS`/`GOARCH go build ./...` + `go vet`. Lint is a single job on ubuntu. All matrix jobs are required checks, so a red cell blocks merge.
  **Interpretation note for criterion 1:** race-enabled tests on all four cells natively is not achievable with GitHub-hosted runners on a private repo (linux arm64 hosted runners are free for public repos only; Intel macOS runners are retiring). Criterion 1 is read as: the gate matrix runs on every Linux/macOS × amd64/arm64 cell — full gates where hosted runners exist natively, compile+vet gates on cross cells. **The phase researcher MUST verify current runner availability** (macos Intel retirement timeline, private-repo arm64 options, current macos-15 label) and the planner adjusts cells accordingly. If the repo is later made public, upgrade linux/arm64 to a native `ubuntu-24.04-arm` cell.
- **D-06:** CI hygiene per go-ultimate repo-and-ci conventions: `concurrency` group with `cancel-in-progress` per ref; `setup-go` with `go-version-file: go.mod` (the Go version lives in exactly one place); tidy gate (`go mod tidy` then `git diff --exit-code go.mod go.sum`); govulncheck as a weekly scheduled cron workflow plus on dependency PRs.
- **D-07:** Actions versions from research: `actions/checkout@v7`, `actions/setup-go@v7` (research-verified current).

### GitHub-Side Setup
- **D-08:** Branch protection is applied **scripted via `gh api`** as part of phase execution (repository Ruleset: require the CI status checks, require pull request before merging, block force pushes and deletions on `master`), with the exact commands recorded in a setup doc in the repo for re-application/audit; enable delete-branch-on-merge. `gh` is authenticated locally (account `onokonem`, write access to `Djarvur/sup-kv` confirmed). Verification asserts the ruleset via `gh api` — not by eyeballing the UI.
- **D-09:** The repo **stays PRIVATE** through this phase. Flipping to public is a user decision outside this workflow (it would unlock a free native linux/arm64 hosted runner and Dependabot alerts — see Deferred).

### Lint Configuration
- **D-10:** Vendor the strict config byte-unmodified: fetch `powerman/golangci-lint-strict` for **exactly** golangci-lint v2.15.0 into a dedicated file with provenance documented (fetch URL + version); the project `.golangci.yml` `extends:` it and holds only project overrides. Never edit the vendored file; bump = re-fetch per version.
- **D-11:** depguard rules encode the architecture boundary: `internal/port` — stdlib only (deny third-party imports); `internal/app` — deny `net`, `os`, `syscall`, `google.golang.org/*` (the pure decision core has no I/O or infrastructure). Exact deny-list finalized by the planner against go-ultimate and the roadmap wording; a probe (a violating file making the gate red) demonstrates enforcement in verification.
- **D-12:** No `//nolint` without a written justification — project rule, enforced by the strict config's nolint handling (already part of powerman strict).

### Research Corrections (2026-10-31, post-research — auto-mode decisions amended on verified facts)
- **D-05 amended:** `ubuntu-24.04-arm` hosted runners ARE available for private repos ($0.005/min — cheaper than x64), and `macos-13` is retired. The matrix therefore uses **native full gates** (vet + race) on `ubuntu-latest` (linux/amd64), `ubuntu-24.04-arm` (linux/arm64), and `macos-15` (darwin/arm64); darwin/amd64 stays a cross-compile cell (Intel macOS runners exist at ~10x cost — not worth it). Criterion 1 interpretation note stands, now with 3 of 4 cells native.
- **D-10 amended:** `extends:` does not exist in golangci-lint v2.15.0 — and a config containing it is **silently ignored entirely** (verified by execution). Replacement mechanism: a single `.golangci.yml` = the vendored powerman/golangci-lint-strict 2.15.0 config byte-unmodified **with project depguard rules inserted as new keys under the existing `linters.settings.depguard.rules` mapping**; provenance (fetch URL + version) documented; upstream content otherwise never edited.

### Claude's Discretion
- Placeholder file names and content within the skeleton (doc.go vs real minimal types), workflow file naming/layout (one workflow vs split lint/test/schedule), the exact cron schedule for govulncheck, and the setup-doc location/format. All guided by go-ultimate references.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design and Requirements
- `docs/IDEA.md` — authoritative, self-contained design document; §Product principles and §Candidate v1 requirements carry the quality-gate and CI mandates (QUALITY/CI items) this phase implements
- `.planning/REQUIREMENTS.md` — CI-01, CI-02, CI-03, QUAL-01 are the four requirements this phase covers
- `.planning/ROADMAP.md` §Phase Details → Phase 1 — goal and the four success criteria (the verification contract)
- `.planning/PROJECT.md` — Constraints section (tech stack, quality, process) and Key Decisions table

### Pinned Tooling Research
- `.planning/research/STACK.md` — machine-verified pinned versions (golangci-lint v2.15.0, golangci-lint-action v9, checkout@v7/setup-go@v7, govulncheck v1.9.0), tool-directive pattern, matrix variant guidance, Dependabot grouping. Also embedded in `.claude/CLAUDE.md` (Technology Stack section)

### Mandated Conventions
- go-ultimate skill (github.com/Djarvur/go-ultimate, v0.15.0; available locally as the `go-ultimate` skill from the djarvur-plugin-marketplace) — key references: `references/engineering-policy.md`, `references/repo-and-ci.md` (CI/tidy/go-version-file mandates), `references/libraries.md`
- `github.com/powerman/golangci-lint-strict` — per-version strict lint configs; source of the vendored config for exactly v2.15.0

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — the repo is greenfield: no Go code, no CI, no lint config yet (`README.md`, `LICENSE`, `.gitignore`, `docs/IDEA.md` only). Everything in this phase is new.

### Established Patterns
- `.claude/CLAUDE.md` embeds the STACK research (pinned versions + CI patterns) — treat it as the standing tooling reference until code supersedes it.

### Integration Points
- Remote `github.com:Djarvur/sup-kv` exists, default branch `master`, currently **private**; local `gh` CLI authenticated with write access. Branch protection and ruleset changes apply against this remote.
- Local toolchain: go 1.27.1 (linux/amd64) matches the `go 1.27` directive decision.

</code_context>

<specifics>
## Specific Ideas

- Pinned version set is non-negotiable input: golangci-lint v2.15.0 (powerman strict config 2.15.0), golangci-lint-action v9 (`version: v2.15`), actions/checkout v7, actions/setup-go v7, govulncheck v1.9.0 — all machine-verified in STACK research on 2026-10-31.
- The `-race` gate must run on every natively-available cell; cross cells still run `go build` + `go vet` so a darwin/amd64 or linux/arm64 compile break blocks merge.
- Research flags two CI facts to re-verify during planning: hosted Intel macOS runner retirement status, and linux arm64 hosted runner availability for private repos (this decides whether criterion 1's letter or its documented interpretation applies).

</specifics>

<deferred>
## Deferred Ideas

- **Flip repo to public** — unlocks a free native linux/arm64 hosted runner and Dependabot alerts; a user decision to revisit at/after v1.0. Not automated in this phase.
- **Buf CLI v1.75 + protoc-gen-go v1.36.12 + protoc-gen-go-grpc v1.6.2 pinning** — deferred to the first codegen phase (Phase 4 wire envelopes / Phase 5 gRPC).
- **goreleaser for 4-target release archives + checksums + SBOM** — release tooling for Phase 7 / v1.0; explicitly not blocking this phase (STACK research recommendation).

</deferred>

---

*Phase: 1-CI Bootstrap and Repo Skeleton*
*Context gathered: 2026-10-31*
