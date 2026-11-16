# Phase 7: v1.0 — Docs, Examples and Release - Research

**Researched:** 2026-11-17
**Domain:** documentation-as-code (README quickstart machine-verification), grpcurl-driven example docs, Go release engineering (goreleaser), Keep-a-Changelog, GitHub branch-protection re-apply
**Confidence:** MEDIUM (one unresolved platform question gates the DOCS-01 macOS leg; everything else verified)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Ops docs (DOCS-02)**
- **D-01:** One file — `docs/OPERATIONS.md` — carries the trust model, value-size envelope, local-read semantics (read-your-writes on the same node only), "not a lock/election service" warning, and the clock-drift runbook as a section. No split runbook file. — **Reversibility:** one-way — published operational contract, URLs get bookmarked/linked.
- **D-02:** README.md carries: what sup-kv is, the verified quickstart, and links to docs/OPERATIONS.md and docs/EXAMPLES.md. Ops detail is NOT duplicated inline.
- **D-03:** All documentation in English, matching existing README.md and docs/IDEA.md.

**API examples (DOCS-03)**
- **D-04:** grpcurl-only walkthrough in `docs/EXAMPLES.md`: every RPC (Get/Set/Delete/Watch) including a Watch stream and a `wait_for_replication` write. No Go example programs, no examples/ source dir. Works against the shipped binary via the already-enabled gRPC reflection.
- **D-05:** A CI test spins up two nodes from the built binary and executes the documented example commands, asserting outputs — examples cannot silently rot after release. (grpcurl availability in CI is a planner concern.)

**Quickstart verification (DOCS-01)**
- **D-06:** A CI test extracts the fenced quickstart command block(s) from README.md and executes them **verbatim** against the built binary — the README is the test input, so the doc cannot drift from reality. Runs on `ubuntu-latest` and `macos-15` (both already in the CI gate matrix) to satisfy "works on Linux and macOS". — **Reversibility:** costly — the extraction format (fenced blocks, markers) becomes a contract between README and CI.
- **D-07:** The kill −9 / restart / convergence sequence is part of what CI executes — the full DOCS-01 contract is machine-checked, not just the happy path. (The phase-6 exec harness remains as prior evidence; this check re-proves it in the README's own words.)
- **D-08:** The quickstart's first client is grpcurl — one mental model shared with docs/EXAMPLES.md.

**Release mechanics**
- **D-09:** Tag `v1.0.0` (full SemVer) triggers a goreleaser workflow: 4-target archives + checksums + SBOM attached to a GitHub Release. The tech-stack research already sanctioned adopting goreleaser at exactly this phase ("fine to adopt at the v1.0 release phase — but do not block Phase 0 on it"). goreleaser is release tooling, not a runtime dependency; pin it as a go.mod tool directive consistent with the repo's tooling conventions. — **Reversibility:** costly — the release pipeline and artifact naming become the download contract.
- **D-10:** Keep-a-Changelog style `CHANGELOG.md` seeded with the v1.0.0 entry summarizing what shipped across Phases 1–6 (engine, persistence, replication, gRPC API, hardening/observability/benchmarks).
- **D-11:** The deferred branch-protection server-side ruleset re-apply (ratified override expired 2026-10-31; one command per docs/setup/branch-protection.md) is a task in this phase's plans.

### Claude's Discretion
- Exact goreleaser target list (align with the CI 4-cell matrix: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64) and archive naming.
- README structure beyond quickstart+links (status section update, planned-highlights list refresh, badges if any).
- Fate of docs/IDEA.md after release (keep as design doc with a note, or leave untouched) — nothing user-visible was decided.
- How the quickstart-extraction test locates the fenced blocks (markers vs code-fence language tags) — but the test must consume README.md itself, not a copy.
- CHANGELOG entry wording.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DOCS-01 | README quickstart works as written: two nodes on one localhost broadcast socket, writes visible on both within an RTT, kill −9 one node, restart, state converges without operator action | Fence-extraction test pattern (Pattern 2); phase-6 harness reuse (Pattern 3); asymmetric-port topology with fixed ports in the private window; the **macOS loopback-broadcast blocker** (Open Question 1) must be resolved before the README contract is frozen |
| DOCS-02 | Ops docs state the trust model prominently (no auth/encryption in v1, HMAC is future work), the value-size envelope, local-read semantics, a "not a lock/election service" warning, and a clock-drift runbook | All five statements sourced verbatim from docs/IDEA.md, REQUIREMENTS.md Out-of-Scope, main.go defaults and phase-5/6 CONTEXTs (Pattern 6 content map) |
| DOCS-03 | API usage examples are provided and work against the shipped binary | EXAMPLES.md + CI execution test (Pattern 4); exact grpcurl method paths and bytes-as-base64 semantics (Pattern 5); **prerequisite: gRPC reflection is NOT currently registered — Pattern 1** |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Tech stack bar:** Go; single static binary; minimal dependencies — gRPC + protobuf only at runtime; everything else in-repo. goreleaser/grpcurl are *tools* (go.mod `tool` directives), never runtime deps — D-09 explicitly sanctions this.
- **Quality:** strict golangci-lint (powerman strict config, no suppressions without written justification) and go-ultimate conventions govern all code from the first commit; new test files in `cmd/sup-kv` must pass the same gate (external `main_test` package with `t.Parallel()` is the established shape).
- **Process:** PR-only development; every change (docs included) lands through a PR; CI on every push and PR. The release tag is pushed **after** the phase's PRs merge (planner sequences merge → tag → release → ruleset re-apply).
- **Tool pinning:** all dev tools pinned via go.mod `tool` directives, invoked `go tool <name>` — never `tools.go`, never bare `go install` (AGENTS.md). goreleaser and grpcurl follow `buf`/`golangci-lint` precedent.
- **CI check-run names are a ruleset contract:** `lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)`, `gate (darwin-amd64-cross)` — any ci.yml job addition/rename is a `docs/setup/ruleset.json` change in the same commit.
- **Docs language:** English (D-03).
- **go-ultimate documentation skill conventions apply** to the docs deliverables: write for humans, funnel structure (what/why → quickstart → next steps), self-contained examples, no unsafe patterns, CI-automated doc quality (exactly what D-06/D-05 deliver).

## Summary

Phase 7 ships documentation whose correctness is machine-enforced the same way code is, plus the v1.0.0 release pipeline. All product capability exists (Phases 2–6); this phase adds: README.md rewritten around a verbatim-CI-verified quickstart, docs/OPERATIONS.md with the five mandated truthful statements, docs/EXAMPLES.md as a grpcurl walkthrough executed by CI, CHANGELOG.md (Keep-a-Changelog), a tag-triggered goreleaser workflow (4 targets + checksums + SBOM), and the overdue branch-protection ruleset re-apply.

**Two load-bearing findings dominate planning.** First: **gRPC reflection is NOT currently registered on the server** — `listenGRPC` in `cmd/sup-kv/main.go:583-586` registers only `SupKVServiceServer` and the health server; the CONTEXT's "already-enabled reflection" premise (D-04) is false in code. The fix is one line (`reflection.Register(gs)` from the already-required grpc module, zero new deps) but it is real product code and must land before any grpcurl example works — it is the phase's only product-code change and should be its own early task. Second: **the DOCS-01 macOS leg hits the phase-6 empirical wall** — macOS lo0 carries no BROADCAST flag, so loopback broadcast to `127.255.255.255` does not fan out on darwin; the phase-6 exec scenarios loud-skip on macos-15 for exactly this reason, while D-06 demands the quickstart run *for real* there. Subnet broadcast via the runner's real interface was never probed (phase-6 research A6 flagged it as unexplored); a one-run probe on macos-15 must decide between three concrete options before the README contract is frozen (Open Question 1).

The release engineering is low-risk and fully specified: goreleaser v2.20.2 (2026-11-12, requires go 1.27.4 — exactly the repo toolchain) as a go.mod tool directive invoked `go tool goreleaser` in a tag-triggered workflow; syft for SBOM; Keep-a-Changelog 2.0.0 (published 2026-06-07) for the 1.0.0 entry. grpcurl v1.9.4 is available as a second tool directive whose dependency floors sit below the repo's pins (no downgrades). No adoptable library fits the minimal-deps bar for markdown doc-testing — an in-repo stdlib fence extractor (~60 lines) riding the existing phase-6 exec harness is the right shape, matching the ecosystem pattern (shelldoc/zed-mdtest) without the dependency.

**Primary recommendation:** Sequence the phase as (1) reflection registration + tool directives, (2) the macos-15 broadcast probe deciding the quickstart topology, (3) README+EXAMPLES+OPERATIONS+CHANGELOG authored against that decision with their CI verification tests, (4) goreleaser config + workflow verified by `--snapshot`, (5) merge → tag v1.0.0 → release → ruleset re-apply.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| gRPC reflection registration | `cmd/sup-kv/main.go` (listenGRPC) | — | The only product-code change; server wiring lives in main per the layout contract; `google.golang.org/grpc/reflection` is part of the already-required grpc module |
| Quickstart fence-extraction test | `cmd/sup-kv` external test package (`main_test`) | — | Reuses TestMain's build-once harness, startNode/reap, awaitServing; README lives at repo root, read via relative path — test code, not product |
| EXAMPLES.md execution test | `cmd/sup-kv` external test package | — | Same harness; spins the pair, runs extracted grpcurl blocks with PATH-shimmed `go tool grpcurl` |
| README/OPERATIONS/EXAMPLES/CHANGELOG content | docs tier (repo root + `docs/`) | — | User-facing deliverables; content sourced from IDEA.md, REQUIREMENTS.md, phase CONTEXTs, main.go |
| goreleaser config + release workflow | `.goreleaser.yaml` + `.github/workflows/release.yml` | Makefile `release`/`release-snapshot` targets | CI tier; tag-triggered, separate from ci.yml so the ruleset's five check contexts stay untouched |
| Branch-protection ruleset re-apply | Ops task (`gh api`, docs/setup/branch-protection.md) | — | Live GitHub state outside git; one command, verified read-side; may still 403 (org-plan gated) — the task records the attempt either way |
| Tag + GitHub Release | git tag + goreleaser | — | Tag pushed after merge; goreleaser creates the Release and uploads archives/checksums/SBOMs with `GITHUB_TOKEN` |

## Standard Stack

No new runtime dependencies. The shipped graph stays `google.golang.org/grpc v1.85.0` + `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e` (go.mod, read this session). Reflection adds zero modules.

### Core (tools added this phase, pinned via go.mod `tool` directives)

| Tool | Version | Purpose | Why Standard | Provenance |
|------|---------|---------|--------------|------------|
| `github.com/goreleaser/goreleaser/v2` | v2.20.2 (2026-11-12) | 4-target archives + checksums.txt + SBOM + GitHub Release on tag push | Pre-sanctioned by stack research at exactly this phase (D-09); module root IS the main package (`main.go` at repo root — no `/cmd` path in the directive); its `.mod` requires go 1.27.4 — identical to the repo's `toolchain go1.27.4`, zero toolchain friction; much of its dep tree (charmbracelet, cobra…) is already in go.sum via buf's graph | [VERIFIED: proxy.golang.org `go list -m`; GitHub Releases API tag v2.20.2 published 2026-11-12; goreleaser official docs via Context7 incl. the `go install github.com/goreleaser/goreleaser/v2@latest` form] |
| `github.com/fullstorydev/grpcurl/cmd/grpcurl` | v1.9.4 | The client for every EXAMPLES.md command and the README quickstart's first client (D-04/D-08) | De-facto standard gRPC debugging client; works against reflection-serving servers with zero proto files; its floors (grpc v1.83.2, protobuf v1.36.12 per its `.mod`) sit **below** the repo's pins — MVS keeps v1.85.0/v1.36.13-…, no downgrades | [VERIFIED: proxy.golang.org `go list -m`; grpcurl official docs via Context7] |

### Supporting (workflow actions, pinned majors)

| Action | Version | Purpose | When to Use |
|--------|---------|---------|-------------|
| `anchore/sbom-action/download-syft` | v0.24.3 (latest) | Installs `syft` — goreleaser's SBOM pipe shells out to it (`cmd: syft` default) | Release workflow only, before `go tool goreleaser release` |
| `goreleaser/goreleaser-action` | v7 (v7.3.0, 2026-10-12) | Alternative to the tool-directive invocation — downloads its own goreleaser binary | **Not recommended** — the tool-directive path (`go tool goreleaser`) matches repo conventions (same pinned version as local, recorded in go.mod) and needs no third-party binary download |
| `actions/checkout@v7`, `actions/setup-go@v7` | existing | Release workflow steps | `go-version-file: go.mod` per go-ultimate (the Go version lives in exactly one place) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-repo fence extractor (~60 stdlib lines) | `github.com/brimdata/zed/mdtest` (Go pkg), shelldoc (Python) | New module against the minimal-deps bar / non-Go tool in CI; the extraction is line-scanning for ``` fences — not worth a dependency [CITED: pkg.go.dev/github.com/brimdata/zed/mdtest, github.com/endocode/shelldoc] |
| `go tool goreleaser` in the workflow | goreleaser-action@v7 | Action installs a second, differently-pinned binary; tool directive keeps local/CI parity and go.mod as the single version record — repo convention wins |
| goreleaser | hand-rolled build matrix + `gh release upload` | Rejected by D-09; goreleaser gives archives+checksums+SBOM+release in one pinned tool |
| Keep-a-Changelog hand-written 1.0.0 entry | git-log-generated changelog (`git-cliff`, goreleaser's changelog pipe) | KaC 2.0.0 explicitly: commit logs are not changelogs; curated human entries only — first release is one Added section, cheap by hand |
| grpcurl | compiled Go example programs | Rejected by D-04 (grpcurl-only decision, recorded 2026-11-16) |

**Installation:**
```bash
go get -tool github.com/goreleaser/goreleaser/v2@v2.20.2
go get -tool github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.4
```

**Version verification (run this session):** `go list -m github.com/goreleaser/goreleaser/v2@latest` → `v2.20.2`; `go list -m github.com/fullstorydev/grpcurl@latest` → `v1.9.4`; goreleaser-action latest `v7.3.0` (2026-10-12); sbom-action latest `v0.24.3`. Both `.mod` files fetched from proxy.golang.org.

## Package Legitimacy Audit

> The seam's `package-legitimacy check` supports npm/pypi/crates only (Go rejected at arg-parse). Go-module legitimacy was verified through equivalent authoritative channels: proxy.golang.org (registry), GitHub Releases/contents API (source repo + release cadence), Context7 (official docs).

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| github.com/goreleaser/goreleaser/v2 | Go proxy | ~6 yrs (v2 line since 2024; project since 2016) | de-facto industry standard (used by most Go releases) | github.com/goreleaser/goreleaser (verified: releases API, contents API, official docs) | OK | Approved |
| github.com/fullstorydev/grpcurl | Go proxy | ~8 yrs (v1 line since 2018) | standard gRPC debugging tool | github.com/fullstorydev/grpcurl (verified: releases/contents API, official docs) | OK | Approved |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

Both names are training-data-famous AND registry-verified AND official-docs-confirmed — the slopsquatting risk profile these two names carry is negligible.

## Architecture Patterns

### System Architecture Diagram — how the phase's pieces flow

```
                        ┌────────────────────────────────────────────────────────┐
                        │  PR flow (branch protection; ruleset re-apply D-11)    │
                        └────────────────────────────────────────────────────────┘
  feature branch ──PR──▶ ci.yml gate matrix (5 check contexts, UNCHANGED this phase)
   README.md  ─┐            │
   EXAMPLES.md ─┤            ▼  go test -race ./... includes:
   OPERATIONS  ─┤   ┌─────────────────────────────────────────────┐
   CHANGELOG   ─┘   │ TestMain (existing) builds bin once         │
                    │   ├─ TestReadmeQuickstart:                  │
                    │   │   extract fenced blocks from README.md  │◀─ README is the test input
                    │   │   bash -e each block verbatim in tmpdir │
                    │   │   (PATH: built sup-kv + shimmed grpcurl)│
                    │   │   incl. kill −9 → restart → convergence │
                    │   └─ TestExamplesDoc:                        │
                    │       startExecPair (phase-6 harness)       │
                    │       extract grpcurl blocks from EXAMPLES │◀─ EXAMPLES.md is the test input
                    │       .md, execute, assert JSON output      │
                    └─────────────────────────────────────────────┘
                                    │ merge to master
                                    ▼
                       git tag -a v1.0.0 && git push origin v1.0.0
                                    │
                                    ▼
                    .github/workflows/release.yml (NEW, tag-triggered)
                      checkout(fetch-depth:0) → setup-go(go-version-file)
                      → download-syft → go tool goreleaser release --clean
                      → GitHub Release v1.0.0: 4 archives + checksums.txt
                        + per-archive SBOM (spdx-json)
                                    │
                                    ▼
                    gh api POST rulesets (D-11 re-apply; may 403-plan-gate
                    → record the attempt, docs/setup/branch-protection.md)
```

### Recommended Project Structure (additions only)

```
README.md                    # rewritten: what/why + verified quickstart + links
CHANGELOG.md                 # NEW: Keep-a-Changelog 2.0.0, single 1.0.0 entry
docs/
├── OPERATIONS.md            # NEW: trust model, envelope, local reads, not-a-lock, drift runbook
├── EXAMPLES.md              # NEW: grpcurl walkthrough, every RPC
├── IDEA.md                  # existing (discretion: note or untouched)
└── setup/                   # existing (branch-protection.md unchanged this phase)
.goreleaser.yaml             # NEW
.github/workflows/
├── ci.yml                   # existing — NO job/context changes (ruleset contract)
└── release.yml              # NEW: tag-triggered
cmd/sup-kv/
├── main.go                  # +1 line: reflection.Register(gs) in listenGRPC
├── main_test.go             # existing harness (untouched or minimally extended)
├── readme_test.go           # NEW: fence extraction + verbatim execution
└── examples_test.go         # NEW: EXAMPLES.md grpcurl execution
Makefile                     # + release / release-snapshot targets
go.mod / go.sum              # +2 tool directives (goreleaser, grpcurl)
```

### Pattern 1: Register gRPC reflection (the D-04 prerequisite that does not exist yet)

**What:** `listenGRPC` currently registers exactly two services — verified this session, `cmd/sup-kv/main.go:583-586`:
```go
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	return listener, gs, nil
```
No `reflection` string occurs anywhere in the repo's `.go` files (grep across the tree, this session). The phase-7 CONTEXT/D-04 premise "already-enabled gRPC reflection" is false; the 05-REVIEW mention of "visible via descriptors/reflection" refers to the descriptors embedded in generated code, not a running reflection service. **grpcurl without proto files requires the server reflection service.**

**Fix (one line + import):**
```go
import "google.golang.org/grpc/reflection"
// in listenGRPC, after the two registrations:
reflection.Register(gs)
```
`google.golang.org/grpc/reflection` ships inside the already-required grpc module (verified: `$(go env GOMODCACHE)/google.golang.org/grpc@v1.85.0/reflection/serverreflection.go` exists) — zero new modules. Land this as the phase's first task; every example test depends on it. [VERIFIED: main.go:583-586; module cache listing; repo-wide grep]

**When to use:** exactly here, once. Consider whether ops docs should note reflection is always on (it exposes the schema — see Security Domain).

### Pattern 2: The README fence-extraction contract (D-06/D-07)

**What:** A test in `cmd/sup-kv` (external `main_test` package) reads `../../README.md`, extracts the quickstart command block(s), and executes each verbatim. Ecosystem precedent (shelldoc, zed/mdtest, runme): select blocks → run with `bash -e` in a temp working dir → assert exit codes/output. No library needed — extraction is a line scan for ``` fences (~60 stdlib lines).

**Block selection (Claude's discretion resolved — recommendation):** an HTML-comment marker line immediately above the fenced block, e.g. `<!-- quickstart-verify -->`, with the fence language `bash`. Rationale: a language tag alone (`bash`) silently enrolls every future `bash` fence (build instructions, dev setup) into the execution contract; an explicit marker keeps the contract opt-in and self-documenting, and the marker line renders invisibly on GitHub. The test fails loudly if **zero** marked blocks exist (the contract cannot silently disappear).

**Verbatim-runnable README shape (the real design constraint):** every marked block must be copy-paste shell. Consequences for the README author:
- Nodes start as background jobs with captured PIDs: `sup-kv ... & NODE_A=$!` so the kill/restart steps (`kill -9 $NODE_A`, rerun the same command) are literally executable and portable to bash on Linux+macOS.
- Ports are **fixed constants in the doc** (strangers run this too). Use the phase-6 private window below the ephemeral range so the parallel `-race` suite can never collide: `[VERIFIED: cmd/sup-kv/scenario_test.go:101-103 — `scenarioPortLo    = 21000` / `scenarioPortHi    = 32678`, with the comment "strictly below the kernel's ephemeral range (ip_local_port_range starts at 32768 on Linux, 49152 on darwin)"] — e.g. UDP 27901/27902, gRPC TCP 27911/27912.
- Topology: the asymmetric-port pairing from phase 6 (A binds P1 broadcasting to peer-port P2, B the mirror) — the only two-node localhost topology proven to converge on Linux [VERIFIED: 06-RESEARCH Pattern 5, scenario_test.go:14-23].
- Both nodes need explicit `--grpc-addr` (the default is one shared value — `[VERIFIED: main.go:131 — `defaultGRPCAddr       = "127.0.0.1:7901"`]`; two nodes on defaults would collide).
- The block ends with cleanup (`kill`/`wait` on survivors) so CI leaves no orphans; `bash -e` semantics are the execution contract.
- Downloads/build instructions stay OUTSIDE the marked blocks (CI runs against the locally built binary); the test's temp working dir contains the built binary named `sup-kv` with that dir prepended to PATH, so the doc can say `sup-kv` exactly as a stranger with the release archive or `go install` would.

**Test skeleton:**
```go
// Source: in-repo pattern per shelldoc/mdtest precedent; harness from cmd/sup-kv/main_test.go
func TestReadmeQuickstart(t *testing.T) {
	t.Parallel()
	blocks := extractMarkedBlocks(t, "../../README.md", "<!-- quickstart-verify -->")
	if len(blocks) == 0 {
		t.Fatal("README.md carries no <!-- quickstart-verify --> block — the DOCS-01 contract is gone")
	}
	bin := smokeBinary(t) // existing: build-once, loud skip on missing go
	dir := t.TempDir()
	// copy bin to dir/sup-kv; write dir/grpcurl shim → exec "go tool grpcurl" "$@"
	for i, block := range blocks {
		runBashVerbatim(t, dir, block, fmt.Sprintf("README quickstart block %d", i))
	}
}
```
`runBashVerbatim` execs `bash -e` with `cmd.Dir = dir` and `cmd.Env` carrying the augmented PATH; stderr/stdout captured into the failure message. The kill −9/restart/convergence steps assert by being part of the block (a broken convergence path fails the block itself; optional: assert the final grpcurl Get output line contains the expected base64 value).

### Pattern 3: Reuse the phase-6 exec harness — do not rebuild it

**What:** `cmd/sup-kv/main_test.go` already provides the whole harness: `TestMain` builds the binary once (`buildSmokeBinary`, main_test.go:86-108), `smokeBinary(t)` loud-skips on toolchain-probe failure, `startNode`/`launchNode` (stderr capture), `signalAndWaitExit`/`assertExitCode`, `awaitServing` + `supKVConn` over real gRPC, `waitFor` polling, `assertNoPanic`; `scenario_test.go` adds `newExecPair` (asymmetric topology + private-window ports), `killAndReap`, `execSet/execGet/execServing`. [VERIFIED: main_test.go and scenario_test.go read this session]

**Consequences:**
- `readme_test.go`/`examples_test.go` join package `main_test` and consume these helpers directly — zero new harness code beyond the extractor and the grpcurl PATH shim.
- The darwin capability probe precedent (`probeLoopbackSend`) is the exact shape for Open Question 1's macos-15 subnet-broadcast probe: probe first, then either run for real or skip loudly naming the probe [VERIFIED: scenario_test.go:163-193, 218-249].
- Shrunken timers for CI deadlines follow the scenario constants (`execHelloInterval = 100ms`, `execSyncInterval = 200ms`, `execStartupGate = 1s` — scenario_test.go:53-55) — but the README quickstart should use **defaults or near-defaults** (it is the stranger's doc); the extraction test's generous `waitFor`-style polling (the `execConvergeDeadline = 20*hello + 20*sync` shape) tolerates default 1s hello / 30s sync only via the startup-gate + unicast-replay paths — verify convergence bounds with default timers are inside CI timeout (they are: baseline convergence is one RTT + discovery ≤ a few hello intervals; the 30s sync timer is a fallback, not the primary path).

### Pattern 4: EXAMPLES.md — grpcurl walkthrough executed by CI (D-04/D-05)

**What:** One doc, every RPC as copy-paste `grpcurl` commands, executed by a test that first starts the pair via `startExecPair` on the SAME fixed ports the doc names. Full method paths (from the shipped proto, package `supkvapi.v1`, service `SupKVService` [VERIFIED: internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto — `package supkvapi.v1;`, `service SupKVService` with rpc Get/Set/Delete/List/Watch/SnapshotCreate/SnapshotLoad]):

```bash
# Get (missing key → NotFound error output)
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get
# Set — value is proto3 bytes ⇒ base64 in JSON:  -d '{"key":"demo/hello","value":"'"$(printf sup-kv | base64)"'"}'
grpcurl -plaintext -d '{"key":"demo/hello","value":"c3VwLWt2"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Set
# Set with wait_for_replication (D-08 aggregate report in the output)
grpcurl -plaintext -d '{"key":"demo/wait","value":"c3VwLWt2","wait_for_replication":true,"wait_timeout_ms":2000}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Set
# Delete / List
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Delete
grpcurl -plaintext -d '{"prefix":"demo/"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/List
# Watch (server-streaming; snapshot arrives first — background it, make a change, read the stream)
grpcurl -plaintext -d '{"prefix":"demo/"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Watch &
WATCH=$!; sleep 1
grpcurl -plaintext -d '{"key":"demo/live","value":"c3VwLWt2"}' 127.0.0.1:27912 supkvapi.v1.SupKVService/Set
sleep 1; kill $WATCH
# Health (standard endpoint, gate-reflecting)
grpcurl -plaintext -d '{}' 127.0.0.1:27911 grpc.health.v1.Health/Check
```

**Execution-test mechanics:** the test extracts the marked grpcurl blocks from `docs/EXAMPLES.md` and runs each with `bash -e` in a temp dir, PATH carrying (a) a `grpcurl` shim that execs `go tool grpcurl "$@"` (keeps the doc's commands plain-`grpcurl` for strangers while CI runs the pinned v1.9.4) and (b) nothing else special — nodes are already running on the doc's fixed ports from `startExecPair`-equivalent launch args. Output assertions: grep the captured stdout for the expected JSON shape (e.g. `"key":"demo/hello"`, the base64 value, `"acked":true`).

**Block-boundary rule:** every marked block must be independently runnable against the running pair (state may carry across blocks in doc order — Get-before-Set returning NotFound is itself a documented behavior). Prefer 3–5 medium blocks over one monolith so a failure names its step.

### Pattern 5: grpcurl semantics that shape the docs (verified flags)

- `-plaintext` — required; the shipped binary serves h2c on the gRPC port. [CITED: grpcurl CLI reference]
- `-d '{json}'` request body; proto3 `bytes` fields are **base64 strings in JSON, both directions** (SetRequest.value, Record.value, Record.version) — the single easiest thing to get wrong in every example; the doc should show the `printf | base64` idiom once and use literal base64 in the copy-paste commands. [CITED: proto3 JSON mapping; grpcurl docs]
- grpcurl **omits zero-valued fields** in JSON output unless `-emit-defaults`. SetResponse without `wait_for_replication` carries `acked:false, acked_count:0, total_peers:0` as zero values — the documented expected output must either show them absent or the assertion must be omission-tolerant. Prefer examples whose asserted fields are non-zero (the wait example shows `"acked":true,"acked_count":1,"total_peers":1`). [CITED: grpcurl CLI reference, `-emit-defaults`]
- `-max-time <dur>` bounds an RPC; on expiry grpcurl exits non-zero (DeadlineExceeded) — for the endless Watch stream, the background-and-`kill` shape (Pattern 4) avoids exit-code ambiguity and is more readable in a doc than `|| true`. [CITED: grpcurl CLI reference — flag semantics; exit behavior ASSUMED from "RPC fails → non-zero exit" README wording]
- Reflection-served servers: `grpcurl -plaintext ADDR list` / `list supkvapi.v1.SupKVService` / `describe …` work with no proto files — worth one example line in EXAMPLES.md ("discover the API yourself"). [CITED: grpcurl README]

### Pattern 6: OPERATIONS.md content map — every mandated statement sourced (D-01)

| Mandated statement (DOCS-02) | Source of truth (read this session) | Wording anchor |
|-------------------------------|--------------------------------------|----------------|
| Trust model, prominent | `docs/IDEA.md` §Trust model (accepted limitation): "The replication segment is trusted: CRC32 guards integrity only; there is no authentication or encryption of datagrams in v1. Authenticating datagrams (HMAC with a shared key) is a documented future work item, not a v1 goal." | Quote-adapt; link v2 SEC-01 from REQUIREMENTS.md |
| Value-size envelope | CLAUDE.md §Scale envelope + `main.go:134` — `defaultValueMaxBytes  = 61440` (flag `--value-max-bytes`, env `SUPKV_VALUE_MAX_BYTES`, larger values → InvalidArgument per CORE-05); whole state in RAM (thousands to tens of thousands of keys); one UDP broadcast segment | Numbers verbatim from code |
| Local-read semantics (read-your-writes same node only) | REQUIREMENTS.md Out-of-Scope ("Read-consistency options… quorums antithetical to the AP design; local-read semantics documented instead"); REPL-10 bounds (normal one RTT; loss ≤ `ack_timeout_max`; partition heal ≤ 2×`sync_interval` + response) | State reads are local, writes visible on the writing node immediately, on peers within the REPL-10 bounds; restarted node serves reads immediately but may briefly miss peer writes until sync (reads are never gated — API-04) |
| Not-a-lock/election warning | REQUIREMENTS.md Out-of-Scope: "Distributed locks / leader election on sup-kv — Locks need linearizability and fencing — impossible on an AP LWW store; documented as not-a-lock-service instead" | Near-verbatim; explain LWW concurrent-write resolution |
| Clock-drift runbook | IDEA.md §Record versions: "correctness never depends on wall clocks; clock skew only affects which simultaneous write wins, deterministically and identically on all replicas"; VERS-05 (`drift_warn`, default 60s — `main.go:145` `defaultDriftWarn      = 60 * time.Second`); phase-6 drift-warnings counter (`MergeStats.DriftWarnings`) and `supkv_*` metrics families | Runbook: symptom (drift warnings in logs/metrics) → cause (wall-clock skew vs peers) → action (fix NTP/PTP; no sup-kv-side repair exists or is needed — no data loss, only which concurrent write wins; a far-ahead node's writes dominate until clocks catch up — behavior tested in phase 6) |
| (Worth adding, one paragraph each) observability + lifecycle | `--metrics-addr` opt-in (no listener when unset — main.go:304-305), SUPKV_LOG_FORMAT/LEVEL, exit codes (`exitDumpFailed = 1`, `exitUsage = 2` — main.go:69-72), SIGTERM graceful path (D-19) | Operators need these; keeps OPERATIONS.md the one ops file (D-01) |

**Structure (go-ultimate docs funnel):** Trust model FIRST and prominent (a warning callout before anything else), then what it is/monitoring, envelope, semantics, drift runbook, lifecycle/exit codes. English (D-03).

### Pattern 7: goreleaser config + workflow (D-09)

`.goreleaser.yaml` (minimal, target list = the CI matrix minus the cross cell — all four real targets are native builds by goreleaser):
```yaml
# Source: goreleaser official docs (Context7) — reproducible-builds blog + sbom/checksum/archives references
version: 2
builds:
  - id: sup-kv
    main: ./cmd/sup-kv
    env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -X main.version={{ .Version }} -X main.commit={{ .ShortCommit }}
checksum:
  name_template: "checksums.txt"
sboms:
  - id: archive
    artifacts: archive   # shells out to syft (installed by download-syft step)
archives:
  - name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ title .Os }}_{{ if eq .Arch "amd64" }}x86_64{{ else }}{{ .Arch }}{{ end }}
release:
  header:
    from_file: CHANGELOG.md   # the curated KaC entry becomes the Release body header
```
- `--version` consistency (D-18): `printVersion` prints `version commit runtime` [VERIFIED: main.go:705-707 `fmt.Printf("sup-kv %s %s %s\n", version, commit, runtime.Version())`]; the Makefile stamps `VERSION = git describe --tags --always --dirty`, `COMMIT = short sha` [VERIFIED: Makefile:5-6]. Use `{{ .Version }}` (tag minus `v`) + `{{ .ShortCommit }}` — one deliberate normalization decision: goreleaser-built binaries report `1.0.0`, make-built report `v1.0.0-x-g…`. Either accept and document, or use `{{ .Tag }}` to keep the `v` — planner picks, but pick consciously (Pitfall 6).
- Workflow (`.github/workflows/release.yml`) — tool-directive style, no goreleaser-action:
```yaml
name: release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write   # create the GitHub Release and upload assets
jobs:
  release:
    runs-on: ubuntu-latest
    timeout-minutes: 20
    steps:
      - uses: actions/checkout@v7
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v7
        with: { go-version-file: go.mod }
      - uses: anchore/sbom-action/download-syft@v0.24.3
      - run: go tool goreleaser release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```
[CITED: goreleaser docs — actions page (tag trigger, `contents: write`, fetch-depth 0), sbom page (syft default cmd), reproducible-builds blog (CGO/ldflags shape); versions VERIFIED via proxy/GitHub API this session]
- Makefile: add `release` (`go tool goreleaser release --clean`) and `release-snapshot` (`go tool goreleaser release --snapshot --clean`) targets in the existing comment-header style; the snapshot run is the task-level verification of `.goreleaser.yaml` **before** any tag exists (`goreleaser check` also exists for config validation — `cmd/check.go` present in the repo listing [VERIFIED: repo contents API; subcommand semantics ASSUMED]). `.gitignore` needs `/dist/`.
- **No ci.yml changes**: the doc tests ride the existing `gate` matrix (`go test -race ./...` picks them up everywhere the suite runs), and release.yml is tag-triggered — the five ruleset check contexts stay byte-identical (Pitfall 4).

### Pattern 8: CHANGELOG.md — Keep-a-Changelog 2.0.0, first release (D-10)

[CITED: keepachangelog.com/en/2.0.0/ — fetched in full this session; convention published 2026-06-07; six types, dates, link refs unchanged from 1.1.0]
```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/2.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-XX

First stable release: leaderless replicated in-memory KV store for small
trusted-LAN clusters (one optional summary sentence).

### Added
- LWW-Element-Map engine with HLC/UUIDv7 record versions, proven by property tests (Phases 2)
- Atomic human-readable JSON dumps with `.prev` fallback and write-confirmed-after-dump guarantee; named snapshots (Phase 3)
- UDP broadcast replication — discovery, ack+retransmit, chunked anti-entropy sync — with convergence bounds proven by a deterministic simulator (Phase 4)
- gRPC API: Get/Set (create-if-absent)/Delete/List/Watch, startup gate, `wait_for_replication`, health endpoint (Phase 5)
- Observability: opt-in metrics/debug endpoint (Prometheus text + expvar + pprof, optional mTLS), structured slog JSON logs (Phase 6)
- Failure-scenario suite over real sockets and processes, benchmarks (Phase 6)
- Single static binary with flags/env config and `--version` (Phase 5)

[Unreleased]: https://github.com/Djarvur/sup-kv/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/Djarvur/sup-kv/releases/tag/v1.0.0
```
KaC 2.0.0 first-release guidance: begin at the current version, reconstructing history optional — one curated `Added` section over Phases 1–6 is the canonical initial shape; the **oldest** version's link goes to its tag (not a compare). Never generate from `git log`. Whether to record pre-release deferrals (CI-02/CI-03 Partial) or the pre-release toolchain CVE bump under `Security` is maintainer wording discretion (D-10/CHANGELOG wording is Claude's discretion) — nothing released was ever vulnerable.

### Anti-Patterns to Avoid

- **Testing a copy of the quickstart instead of README.md** — D-06's whole point is the README IS the test input; a copied fixture reintroduces drift.
- **Enrolling every `bash` fence into the contract** — language-tag-only selection silently binds future prose (build instructions, dev setup) into CI execution; use explicit markers.
- **Adding a new ci.yml job for doc checks** — changes check-run contexts ⇒ requires same-commit `ruleset.json` change ⇒ a ruleset the org plan still 403s; ride the existing `gate` matrix instead.
- **`go install`-ing goreleaser/grpcurl in CI or locally** — violates the repo tool-pinning mandate (AGENTS.md: "never bare `go install`"); tool directives + `go tool` only.
- **Writing examples with raw string values in `-d` JSON for bytes fields** — proto3 JSON requires base64; raw strings fail at runtime, not compile time, and rot silently if not CI-executed.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform release artifacts + checksums + SBOM + GitHub Release | Scripted `go build` matrix, `sha256sum`, `gh release upload` | goreleaser v2.20.2 (tool directive) | Archives naming, per-target builds, checksum aggregation, syft SBOM, Release upload, templating — all solved, tested, and pre-sanctioned by D-09 |
| gRPC request/response shell examples | Custom Go CLI client for docs (rejected by D-04) | grpcurl v1.9.4 against reflection | Zero client code; the stranger runs the same commands CI verifies |
| Changelog generation | Commit-log scraping / goreleaser's git changelog pipe | Hand-curated KaC 2.0.0 entry | KaC: "a raw git history is not a changelog"; first release is one section |
| Markdown parsing for fence extraction | A markdown AST dependency (goldmark etc.) | stdlib line scan for ``` + marker | The contract needs exactly "lines between fence markers after a marker comment"; an AST is dependency weight for zero added correctness |
| Two-node process orchestration in tests | New exec harness | phase-6 `main_test.go` harness (`smokeBinary`, `startNode`, `waitFor`, `killAndReap`, `awaitServing`) | Already proven in CI on all 4 cells; D-07 explicitly builds on it |

**Key insight:** this phase's engineering risk is concentrated in exactly one place — making documentation *executable*; everything else (release mechanics, changelog) is standard tooling the ecosystem solved years ago. Spend the planning effort on the README-as-test contract and the macOS topology question; take the rest off the shelf.

## Common Pitfalls

### Pitfall 1: macOS cannot do loopback broadcast — the DOCS-01 "works on macOS" leg
**What goes wrong:** the quickstart written for `127.255.255.255` loopback broadcast passes ubuntu-latest and loud-skips (or worse, runs the happy half and hangs at convergence) on macos-15.
**Why it happens:** macOS lo0 carries no BROADCAST flag; loopback broadcast fan-out is a Linux-only kernel behavior — established empirically in phase 6 (probeLoopbackSend loud-skip, STATE.md D-08 note, 06-RESEARCH A6). Phase-6 exec/socket scenarios all skip on macos-15 today. D-06 and the 07-CONTEXT "must run for real on macos-15, not loud-skip" premise collide with this finding unless a different broadcast address works.
**How to avoid:** resolve Open Question 1 FIRST — one probe run on macos-15 of subnet broadcast via the runner's real interface (`ipconfig getoption en0 broadcast`, e.g. `192.168.x.255`) and/or limited broadcast `255.255.255.255`. If subnet broadcast fans out locally on darwin, the README derives the address per-platform in one verbatim shell line (`BCAST=$(ipconfig getoption en0 broadcast 2>/dev/null || echo 127.255.255.255)`); if nothing works, the maintainer amends the macOS claim (Option C) — a docs decision, not a code one.
**Warning signs:** macos-15 gate cell green-with-skip on the new quickstart test; README claiming macOS support no CI proves.

### Pitfall 2: Docs written against a reflection that isn't there
**What goes wrong:** every grpcurl example fails with "server does not support the reflection API" at execution time.
**Why it happens:** the CONTEXT/D-04 premise says reflection is "already-enabled"; the code registers no reflection service (Pattern 1 — verified). The stack research *recommended* enabling it in phase-0; nobody landed it.
**How to avoid:** reflection registration is task #1; the examples test fails loudly without it (good).
**Warning signs:** `grpcurl ... list` returning the reflection error in any local dry-run.

### Pitfall 3: proto3 bytes vs JSON in grpcurl examples
**What goes wrong:** `-d '{"value":"hello"}'` returns InvalidArgument or stores garbage; Get output shows base64 that novices don't recognize.
**Why it happens:** proto3 JSON maps `bytes` to base64 strings both directions.
**How to avoid:** examples use explicit base64 literals plus one showed `printf | base64` idiom; the CI assertion greps the base64 form (deterministic).
**Warning signs:** examples asserting human-readable values in Set output.

### Pitfall 4: ci.yml check-run names are a ruleset contract
**What goes wrong:** adding a `docs` job (or renaming anything) to ci.yml leaves the branch-protection ruleset requiring contexts that no longer exist (stuck "Expected" checks) or silently requiring nothing.
**Why it happens:** the five contexts (`lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`, `gate (macos-15)`, `gate (darwin-amd64-cross)`) are recorded in `docs/setup/ruleset.json` and must equal ci.yml's produced check names byte-for-byte [VERIFIED: docs/setup/branch-protection.md:62-68; ci.yml matrix, read this session].
**How to avoid:** zero ci.yml job changes in this phase — doc tests ride the existing `gate` cells; the release workflow is a separate file triggered only by tags (never a PR check).
**Warning signs:** any PR diff touching ci.yml `jobs:` keys.

### Pitfall 5: grpcurl zero-value omission breaks output assertions
**What goes wrong:** the documented Set output lacks `acked`/`acked_count`/`total_peers` (zero-valued without a wait) and the CI grep for them fails — or worse, the doc shows output that grpcurl never prints.
**Why it happens:** grpcurl omits zero-valued scalar fields by default (`-emit-defaults` opts in).
**How to avoid:** choose assertion fields that are non-zero in the example flow, or document `-emit-defaults` where the zero values matter; generate every documented output block by actually running the command (the CI test is the generator of truth).
**Warning signs:** hand-written expected outputs in EXAMPLES.md.

### Pitfall 6: version-string drift between make and goreleaser builds
**What goes wrong:** `sup-kv --version` prints `1.0.0` from a release archive but `v1.0.0-3-g…` from a make build — two identities for one release.
**Why it happens:** goreleaser's `{{ .Version }}` is the tag minus `v`; the Makefile's `git describe` keeps it; `{{ .Commit }}` is the full SHA vs Makefile's short.
**How to avoid:** pin ldflags to `{{ .Version }}` + `{{ .ShortCommit }}` (or `{{ .Tag }}` for the `v`-prefixed form) as ONE conscious decision recorded in the plan; D-18's contract is "version, commit, Go runtime" — both forms satisfy it, drift doesn't.
**Warning signs:** two different `--version` shapes surviving into the release task.

### Pitfall 7: fixed doc ports colliding with the parallel race suite
**What goes wrong:** the README quickstart's fixed ports get taken by a parallel test's `:0`-derived bind (or a previous CI run's straggler) and the verbatim block fails nondeterministically.
**Why it happens:** the doc's ports are constants; the suite runs parallel.
**How to avoid:** pick doc ports inside the phase-6 private window `[21000, 32678)` — below `ip_local_port_range` on both Linux (32768) and darwin (49152), where no rival `:0` bind can ever be issued [VERIFIED: scenario_test.go:96-103]; UDP and gRPC-TCP both.
**Warning signs:** quickstart test flaking only under `-race ./...` full-suite load.

### Pitfall 8: goreleaser SBOM step fails for missing syft
**What goes wrong:** release workflow dies at the sboms pipe — `syft: command not found`.
**Why it happens:** goreleaser's SBOM pipe shells out to `syft` (default `cmd: syft`); nothing installs it on the runner.
**How to avoid:** `anchore/sbom-action/download-syft@v0.24.3` step before `go tool goreleaser release`; verify the whole pipeline once with `--snapshot` (which builds archives+SBOMs without publishing) before tagging.
**Warning signs:** first-ever tag push failing mid-release (recovering a botched tag is exactly the "costly reversibility" D-09 warns about — snapshot first).

### Pitfall 9: the release tag is the first tag in the repo
**What goes wrong:** surprises in tooling that assumed prior tags exist (Makefile `VERSION ?= $(shell git describe … || echo dev)` has always resolved `dev` until now; goreleaser changelog "since last tag" has no previous tag; KaC compare links need the oldest = tag-link form).
**How to avoid:** verify: no tags exist today (`git tag -l` empty — checked this session); Makefile falls back cleanly; the 1.0.0 entry uses the tag link, not compare; release notes body from CHANGELOG.md.
**Warning signs:** none — just a fact to plan around.

### Pitfall 10: ruleset re-apply still 403s
**What goes wrong:** D-11's `gh api --method POST repos/Djarvur/sup-kv/rulesets --input docs/setup/ruleset.json` fails with the same org-plan 403 that deferred it in phase 1.
**Why it happens:** the override expired, but the org plan may not have changed; the 403 is plan-gated, not decision-gated.
**How to avoid:** the task's acceptance is the honest attempt + read-side verify + updated marker in branch-protection.md either way ("applied" or "still plan-gated, re-verified <date>") — mirroring plan 01-04's auto-select defer precedent; do not mark CI-03's protection sub-item complete unless the ruleset is live.
**Warning signs:** treating "run the command" as done without the verify read.

## Code Examples

### Reflection registration (Pattern 1, the one product-code line)
```go
// Source: cmd/sup-kv/main.go listenGRPC + google.golang.org/grpc/reflection (module already required)
import "google.golang.org/grpc/reflection"

func listenGRPC(grpcAddr string, srv *grpcapi.Server) (net.Listener, *grpc.Server, error) {
	// ... listener open ...
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	reflection.Register(gs) // grpcurl/EXAMPLES.md discover the API with zero proto files
	return listener, gs, nil
}
```

### Fence extraction (stdlib, ~60 lines — Pattern 2)
```go
// extractMarkedBlocks returns the fenced-block bodies that immediately follow
// marker comment lines. The README is the contract: zero blocks is a failure.
func extractMarkedBlocks(t *testing.T, path, marker string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	var blocks []string
	for i := 0; i < len(lines); i++ {
		if !strings.Contains(lines[i], marker) {
			continue
		}
		// advance to the opening fence, then capture until the closing fence
		for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```"); i++ {
		}
		var body []string
		for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```"); i++ {
			body = append(body, lines[i])
		}
		if i >= len(lines) {
			t.Fatalf("%s: marker at line %d has no closed fence", path, i)
		}
		blocks = append(blocks, strings.Join(body, "\n"))
	}
	return blocks
}
```

### Verbatim block execution with PATH shims (Pattern 2/4)
```go
// runBashVerbatim executes one doc block with bash -e in dir; PATH carries the
// built binary's dir and a grpcurl shim that execs the pinned go tool binary.
func runBashVerbatim(t *testing.T, dir, block, what string) {
	t.Helper()
	cmd := exec.Command("bash", "-e")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+shimDir+string(os.PathListSeparator)+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Stdin = strings.NewReader(block + "\n")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s failed: %v\n--- block ---\n%s\n--- output ---\n%s", what, err, block, out.String())
	}
}
// grpcurl shim (written once into shimDir by the test):
//   #!/bin/sh
//   exec go tool grpcurl "$@"
```

### Quickstart block shape (README, verbatim-runnable)
```bash
# <!-- quickstart-verify --> (marker renders invisibly; block is copy-paste shell)
sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr 127.255.255.255:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path /tmp/supkv-a.json & NODE_A=$!
sup-kv --node-id 2 --bind-addr 0.0.0.0:27902 --broadcast-addr 127.255.255.255:27901 \
  --grpc-addr 127.0.0.1:27912 --dump-path /tmp/supkv-b.json & NODE_B=$!
grpcurl -plaintext -d '{"key":"demo/hello","value":"c3VwLWt2"}' \
  127.0.0.1:27911 supkvapi.v1.SupKVService/Set          # write on A
grpcurl -plaintext -d '{"key":"demo/hello"}' \
  127.0.0.1:27912 supkvapi.v1.SupKVService/Get          # read on B — replicated
kill -9 $NODE_A                                          # hard death
sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr 127.255.255.255:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path /tmp/supkv-a.json & NODE_A=$!
grpcurl -plaintext -d '{"key":"demo/hello"}' \
  127.0.0.1:27911 supkvapi.v1.SupKVService/Get          # converged, no operator action
kill $NODE_A $NODE_B 2>/dev/null; wait 2>/dev/null || true   # cleanup (CI hygiene)
```
(Exact broadcast-address form for the macOS leg depends on Open Question 1; Linux form shown. `wait_for_replication`, Watch, kill-window timing margins are planner refinements — the shape, not the final text, is the deliverable here.)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Keep a Changelog 1.1.0 (the widely-linked version) | Keep a Changelog **2.0.0** — same six types/dates/link-refs; restructured guidance, `**Breaking:**` markers, pinned-version preamble links | 2026-06-07 | A fresh 2026 changelog pins `/en/2.0.0/` in its preamble; the 1.0.0 entry shape is unchanged |
| goreleaser-action downloads its own binary | `go tool` directive invocation (Go 1.24+ tool directives) | Go 1.24 (2025); goreleaser v2 line current | Same binary local and CI, version recorded in go.mod — matches the repo's buf/golangci-lint precedent |
| protoc-based API discovery in docs | Server reflection + grpcurl (`list`/`describe` with zero proto files) | long-standing | The whole EXAMPLES.md approach rests on it — and on Pattern 1's registration |
| Manual release notes on the GitHub Release page | CHANGELOG.md as the canonical record; Release body pulls from it (`release.header.from_file`) | KaC 2.0.0 guidance | One source of truth; the Release page is a distribution surface |

**Deprecated/outdated:**
- `tools.go` blank-import pinning — replaced by tool directives (AGENTS.md bans it).
- goreleaser v1 configs (`goreleaser.yaml` without `version: 2`) — v2 requires the schema version key.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | macOS subnet broadcast (en0's `…255` address) fans out to local sockets on darwin — unprobed | Open Question 1 / Pitfall 1 | High — decides whether DOCS-01's macOS leg can run for real; the probe is cheap and must precede the README contract |
| A2 | `go get -tool` on grpcurl causes zero version downgrades (its floors grpc v1.83.2/protobuf v1.36.12 < repo pins v1.85.0/v1.36.13-…) | Standard Stack | Low — `.mod` floors verified; MVS picks the higher; worst case is go.sum churn, caught by the tidy gate |
| A3 | grpcurl exits non-zero when `-max-time` expires mid-stream | Pattern 5 | Low — examples use background+kill, sidestepping exit-code semantics entirely |
| A4 | `goreleaser check` validates `.goreleaser.yaml` (cmd/check.go exists in the repo) | Pattern 7 | Low — `release --snapshot --clean` is the verified fallback verification |
| A5 | Default-timer convergence (1s hello, 30s sync fallback) fits CI deadlines for the verbatim quickstart | Pattern 3 | Medium — phase-6 exec constants shrank timers for speed; the README's default-timer run is bounded by discovery+RTT (a few seconds), but if flaky, the README may pin explicit smaller `--hello-interval`/`--sync-interval` flags (honest for a demo) |
| A6 | The GitHub macOS runner exposes a usable broadcast-capable interface (en0) and permits broadcast sends on the runner network | Open Question 1 / Environment | Medium — same probe resolves it; GitHub could filter broadcast on the runner VLAN |
| A7 | goreleaser works on the private repo with the default `GITHUB_TOKEN` (`contents: write`) | Pattern 7 | Low — standard behavior; private repos create private Releases fine |
| A8 | `--version` output shape `sup-kv <version> <commit> <goversion>` satisfies D-18 consistency with goreleaser stamping `{{ .Version }}` (no `v`) | Pitfall 6 | Low — cosmetic identity question; one conscious plan decision resolves it |
| A9 | README/EXAMPLES docs on a private repo are fine for release (public flip is a separate deferred decision) | Summary / Scope | Low — the release ships regardless; "stranger" in the success criteria is prospective |

## Open Questions

1. **The macOS quickstart leg (BLOCKING for the README contract — probe before freezing the doc).**
   - What we know: macOS lo0 has no BROADCAST flag → loopback broadcast (`127.255.255.255`) does not fan out on darwin (phase-6 empirical finding, macos-15 CI skips all two-node socket/exec scenarios today). D-06 demands the quickstart run for real on macos-15. Phase-6 research A6 explicitly left subnet broadcast unprobed.
   - What's unclear: whether (a) subnet broadcast via the runner's real interface (`ipconfig getoption en0 broadcast`) or (b) limited broadcast `255.255.255.255` fans out to local sockets on darwin, and whether the runner network permits it.
   - Recommendation: one probe run on macos-15 (a tiny workflow_dispatch job or a PR whose gate cell runs a `probeLoopbackSend`-shaped probe against both address forms) as the phase's FIRST executed check. Then: **Option A** — works: README derives the broadcast address per-platform in one verbatim shell line (`BCAST=$(ipconfig getoption en0 broadcast 2>/dev/null || echo 127.255.255.255)`), CI runs for real on both OSes. **Option B** — only Linux loopback works but macOS subnet works only off-CI: README shows the Linux block verified + a marked macOS variant block skipped loudly in CI with the probe reason (matches the phase-6 loud-skip precedent, partially satisfying D-06). **Option C** — nothing works on darwin: maintainer amends DOCS-01's macOS claim (docs-level success-criterion change, recorded in REQUIREMENTS/ROADMAP); CI loud-skips on macos-15 naming the probe. Do NOT write the README before this resolves.
2. **goreleaser-action vs tool-directive invocation in the workflow.**
   - What we know: both are standard; the action (v7.3.0) downloads its own pinned binary; the tool directive keeps go.mod as the single version record.
   - Recommendation (decided direction, planner confirms): tool directive — `go tool goreleaser release --clean` — repo convention parity (AGENTS.md tool-pinning mandate).
3. **Draft vs published Release.**
   - What we know: goreleaser default publishes; `release.draft: true` allows review before going public.
   - Recommendation: publish directly (solo maintainer, snapshot-verified pipeline); flip to draft only if the first release reveals formatting problems.
4. **Does OPERATIONS.md get a CI presence-guard (a cheap test asserting the file contains its five mandated topics)?**
   - What we know: DOCS-02 verification is otherwise manual/UAT; the two other docs get hard CI.
   - Recommendation: optional one-page test (`strings.Contains` on the five headings/keywords) — cheap drift-guard for deletion/moves; content truth stays UAT. Planner's call; not required by the requirement text.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | go1.27.4 linux/amd64 [VERIFIED: `go version`] | — |
| git + GitHub remote | release, ruleset | ✓ | remote `git@github.com:Djarvur/sup-kv.git` [VERIFIED: `git remote get-url`] | — |
| `gh` CLI | D-11 ruleset re-apply + verify | ✓ | 2.104.0 [VERIFIED: `gh --version`] | — |
| `grpcurl` | doc tests, examples | ✗ local (not on PATH) | — | Pinned as go.mod tool directive — `go tool grpcurl` needs no separate install anywhere [VERIFIED: proxy] |
| `goreleaser` | release | ~ (mise shim present, no working install verified) | — | Pinned as tool directive — `go tool goreleaser`; local mise shim irrelevant [VERIFIED: proxy] |
| `syft` | SBOM generation | ✗ local | — | Release workflow installs via `anchore/sbom-action/download-syft@v0.24.3`; local snapshot runs without sboms pipe or with syft installed ad hoc |
| macos-15 runner | DOCS-01 macOS leg | ✓ (in existing gate matrix) | — | Open Question 1 decides real-run vs loud-skip |

**Missing dependencies with no fallback:** none — every gap is closed by a tool directive or a workflow step.
**Missing dependencies with fallback:** grpcurl/syft local absence → tool directive / workflow install.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | stdlib `testing` (repo mandate; no assertion library) |
| Config file | none needed — tests join `cmd/sup-kv` package `main_test` with the existing `TestMain` build-once harness |
| Quick run command | `go test -race ./cmd/sup-kv/ -run 'TestReadme|TestExamples'` |
| Full suite command | `make test` (`go test -race -cover ./...`) — the doc tests run inside the existing gate cells, no CI job changes |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOCS-01 | README quickstart fenced blocks execute verbatim: two nodes, write visible both, kill −9, restart, convergence | integration (real binary, real sockets, real processes) | `go test -race ./cmd/sup-kv/ -run TestReadmeQuickstart` | ❌ Wave 0 (`cmd/sup-kv/readme_test.go`) |
| DOCS-01 (macOS leg) | same, on macos-15 gate cell | integration (platform-gated by Open Question 1 outcome) | same (runs in `gate (macos-15)` cell) | ❌ Wave 0 |
| DOCS-02 | OPERATIONS.md exists and carries the five mandated topics | docs presence-guard (optional; content truth is UAT) | `go test ./cmd/sup-kv/ -run TestOperationsDoc` (if adopted) | ❌ Wave 0 (optional) |
| DOCS-03 | EXAMPLES.md grpcurl blocks execute against a running pair; outputs asserted | integration (real binary + pinned grpcurl) | `go test -race ./cmd/sup-kv/ -run TestExamplesDoc` | ❌ Wave 0 (`cmd/sup-kv/examples_test.go`) |
| (release) | goreleaser config builds 4 targets + checksums + SBOM locally | manual gate in task | `make release-snapshot` (`go tool goreleaser release --snapshot --clean`) | ❌ Wave 0 (Makefile target) |
| (release) | tag v1.0.0 produces the GitHub Release | e2e, once | push tag → observe release.yml + Release assets | n/a (one-shot) |
| D-11 | ruleset re-applied (or attempt recorded) | ops, once | `gh api` POST + verify read per docs/setup/branch-protection.md | n/a (one-shot) |

### Sampling Rate
- **Per task commit:** quick run command above.
- **Per wave merge:** `make lint && make test` (full gate parity).
- **Phase gate:** full suite green + snapshot release green before the tag is pushed.

### Wave 0 Gaps
- [ ] `cmd/sup-kv/readme_test.go` — extractor + verbatim runner + PATH shims (covers DOCS-01)
- [ ] `cmd/sup-kv/examples_test.go` — EXAMPLES.md execution (covers DOCS-03)
- [ ] `cmd/sup-kv/main.go` — `reflection.Register(gs)` (prerequisite for DOCS-03/D-04)
- [ ] go.mod tool directives: goreleaser v2.20.2, grpcurl v1.9.4 (+ tidy gate)
- [ ] Makefile `release`/`release-snapshot`; `.gitignore` `/dist/`
- [ ] Optional: operations-doc presence guard

## Security Domain

> security_enforcement enabled (level 1). This phase adds no new user-input surface; its security content is (a) one enabling change — gRPC reflection — (b) supply-chain hardening of artifacts, and (c) documentation that must not overstate safety.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | None this phase (v1 trust model: no auth — documented, not implemented; SEC-01 HMAC is v2) |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | no new surface | Docs/tests only; grpcurl input goes through the existing validated API |
| V6 Cryptography | no | Checksums (SHA256 via goreleaser) are integrity artifacts, not crypto features |
| V14 Config/Ops | yes | Trust model documented prominently (DOCS-02); artifact checksums + SBOM (supply chain); pinned tools (tool directives) — no unpinned downloads in the pipeline |

### Known Threat Patterns for this phase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Artifact tampering in distribution | Tampering | `checksums.txt` over all archives on the Release; (attestations via `actions/attest@v4` + `id-token: write` are available if wanted — optional, not decided) [CITED: goreleaser attestations docs] |
| Dependency/supply-chain compromise | Tampering/Elevation | SBOM per archive (syft spdx-json); go.mod-pinned goreleaser/grpcurl (no floating `latest` in CI); existing govulncheck cron keeps scanning |
| Information disclosure via gRPC reflection | Information Disclosure | `reflection.Register` exposes the service/method/message schema to any client reaching the gRPC port. Accepted: default bind is loopback (`127.0.0.1:7901` — [VERIFIED: main.go:131]), the segment is trusted by design, and the schema is public in the repo anyway. State it in OPERATIONS.md alongside the trust model; no flag needed for v1 |
| Docs teaching unsafe patterns | — (docs quality) | go-ultimate docs rule: no unsafe patterns in examples; OPERATIONS.md leads with the trust-model warning before any usage instruction |

## Sources

### Primary (HIGH confidence)
- Repo sources read this session (all `[VERIFIED: …]` in-text citations): `cmd/sup-kv/main.go` (flags/env/defaults/exit codes/listenGRPC registration block), `cmd/sup-kv/main_test.go` (harness helpers), `cmd/sup-kv/scenario_test.go` (asymmetric topology, private port window, darwin probe precedent, exec constants), `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` (package/service/methods/fields), `.github/workflows/ci.yml` (matrix/check contexts), `Makefile` (targets, VERSION stamping), `go.mod` (tool directives, pins), `docs/setup/branch-protection.md` (ruleset commands, context-coupling rule), `docs/IDEA.md` (trust model, parameters, bounds), `README.md` (current state), `.planning/REQUIREMENTS.md` (DOCS-01..03, Out-of-Scope wording), `.planning/ROADMAP.md` (phase 7 criteria), `.planning/STATE.md` (macOS empirical findings, decision log), 05/06/07 CONTEXTs, 06-RESEARCH (Pattern 5/A6)
- proxy.golang.org — goreleaser v2.20.2 (+`.mod`: go 1.27.4), grpcurl v1.9.4 (+`.mod`: grpc v1.83.2, protobuf v1.36.12 floors)
- GitHub API — goreleaser v2.20.2 release (2026-11-12), goreleaser-action v7.3.0 (2026-10-12), sbom-action v0.24.3, goreleaser repo layout (main.go at root; cmd/check.go exists), grpcurl cmd/grpcurl package
- grpc module cache — `google.golang.org/grpc@v1.85.0/reflection/` present (zero-new-dep claim)

### Secondary (MEDIUM confidence)
- Context7 `/goreleaser/goreleaser` — builds/checksum/sbom/archives/release config, GH Actions workflow + permissions, snapshot mode, install command [CITED: goreleaser.com docs]
- Context7 `/fullstorydev/grpcurl` — CLI reference: `-plaintext`, `-d`, `-emit-defaults`, `-max-time`, list/describe via reflection [CITED: grpcurl docs]
- keepachangelog.com/en/2.0.0/ — full convention fetched (publishedTime 2026-06-07) [CITED]

### Tertiary (LOW confidence)
- DuckDuckGo search results — shelldoc, zed/mdtest, runme doc-testing patterns (pattern shape only; nothing adopted)

## Metadata

**Confidence breakdown:**
- Standard stack (goreleaser/grpcurl versions, pins, flags): HIGH — registry, GitHub API, and official docs all cross-checked this session
- Docs/test architecture (fence extraction, harness reuse, grpcurl semantics): HIGH for in-repo facts; MEDIUM for grpcurl behavioral nuances (A3)
- Release pipeline: MEDIUM — config/workflow shapes cited from official docs; the pipeline is verified only by the planned snapshot run
- DOCS-01 macOS leg: LOW until Open Question 1's probe runs — this is the phase's single material uncertainty

**Research date:** 2026-11-17
**Valid until:** 2026-12-17 (tool versions move monthly: goreleaser v2.20.2 is 5 days old; re-check `go list -m …@latest` before pinning if planning slips past ~2 weeks)
