# Phase 1: CI Bootstrap and Repo Skeleton - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-31
**Phase:** 1-CI Bootstrap and Repo Skeleton
**Mode:** `--auto` — all areas auto-selected; every question resolved with the recommended default (no interactive user session). Alternatives below are the options that were considered.
**Areas discussed:** Skeleton shape & module path, Dependencies & pinned tools, CI matrix structure, GitHub-side setup, Lint config layout

---

## Skeleton Shape and Module Path

| Option | Description | Selected |
|--------|-------------|----------|
| Full skeleton with placeholders | `cmd/sup-kv` + `internal/port` + `internal/app` with compileable placeholders and one trivial test; depguard boundary meaningful from day one | ✓ |
| go.mod + lint + CI only | Packages appear in Phase 2+; depguard rules vacuous until then | |

**Notes:** Auto-selected: full skeleton — success criterion 4 (machine-enforced boundary) requires the packages to exist.

| Option | Description | Selected |
|--------|-------------|----------|
| `github.com/Djarvur/sup-kv` | Matches the existing remote; go-ultimate rule: module path = repo path | ✓ |
| Other path | Would diverge from the GitHub repo location | |

---

## Dependencies and Pinned Tools

| Option | Description | Selected |
|--------|-------------|----------|
| Zero runtime deps now | go.mod = `go 1.27` only; grpc/protobuf added in their phases | ✓ |
| Add grpc+protobuf now | Pre-wire the full shipped graph in Phase 1 | |

**Notes:** Auto-selected: zero deps — keeps the Phase 1 gate minimal; Dependabot stays near-silent until real deps land.

| Option | Description | Selected |
|--------|-------------|----------|
| Gate-only tool directives | golangci-lint v2.15.0 + govulncheck v1.9.0 via `go get -tool` | ✓ |
| All future tools now | Include Buf CLI + protoc plugins in Phase 1 | |

---

## CI Matrix Structure

| Option | Description | Selected |
|--------|-------------|----------|
| Native + cross hybrid | Full gates on ubuntu-latest (linux/amd64) + macos-15 (darwin/arm64); cross-compile build+vet for linux/arm64 and darwin/amd64; works on private repo with hosted runners | ✓ |
| Force native 4 cells | Self-hosted or paid arm64/Intel runners to satisfy criterion 1's letter | |
| Flip repo public | Unlocks free linux/arm64 hosted runner — but visibility is a user decision | |

**Notes:** Auto-selected: hybrid — criterion 1 interpreted as "gate matrix on every cell" with the race detector running on all natively-available cells; runner facts (Intel mac retirement, private-repo arm64) flagged for research verification. Public flip recorded as deferred idea.

| Option | Description | Selected |
|--------|-------------|----------|
| Full go-ultimate CI hygiene | concurrency+cancel-in-progress, go-version-file, tidy gate, weekly cron govulncheck + dependency-PR trigger | ✓ |
| Gates only | Bare lint/vet/test jobs | |

---

## GitHub-Side Setup

| Option | Description | Selected |
|--------|-------------|----------|
| Scripted `gh api` ruleset | Required checks, require PR, block force-push/deletion on master; commands recorded in a setup doc; verification via `gh api` | ✓ |
| Manual UI runbook only | Document the clicks; no scripted application | |

**Notes:** Auto-selected: scripted — `gh` authenticated locally with confirmed write access to `Djarvur/sup-kv`; delete-branch-on-merge enabled alongside.

| Option | Description | Selected |
|--------|-------------|----------|
| Stay PRIVATE | No visibility change inside the workflow | ✓ |
| Flip public now | Would unlock free arm64 runner + Dependabot alerts | |

---

## Lint Config Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Vendored strict config + `extends:` | powerman/golangci-lint-strict@2.15.0 byte-unmodified; project overrides in extending config | ✓ |
| Inline copy-edit | Copy the strict config and edit in place — loses upstream provenance | |
| Hand-rolled enable-list | go-ultimate default 11-linter list — weaker than the stated "all applicable linters" gate | |

| Option | Description | Selected |
|--------|-------------|----------|
| stdlib-only port + no-I/O app | `internal/port`: stdlib only; `internal/app`: deny `net`, `os`, `syscall`, `google.golang.org/*` | ✓ |
| Roadmap-letter minimum | Only `net`/`os`/grpc denials, exact roadmap wording | |

**Notes:** Auto-selected the broader deny-list; planner finalizes the exact list with go-ultimate references.

---

## Claude's Discretion

Placeholder file naming/content in the skeleton, workflow file layout (one vs split), govulncheck cron schedule, setup-doc location/format.

## Deferred Ideas

- Flip repo to public — user decision, revisit at/after v1.0
- Buf CLI + protoc plugin pinning — first codegen phase (Phase 4/5)
- goreleaser release tooling — Phase 7 / v1.0
