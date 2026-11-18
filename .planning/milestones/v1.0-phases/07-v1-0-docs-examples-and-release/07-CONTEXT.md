# Phase 7: v1.0 — Docs, Examples and Release - Context

**Gathered:** 2026-11-16
**Status:** Ready for planning

<domain>
## Phase Boundary

The shipping phase: (1) README.md rewritten from its current "design phase" state into a machine-verified quickstart — two nodes on one localhost broadcast socket, write visible on both, kill −9 one node, restart, convergence without operator action; (2) docs/OPERATIONS.md written with the trust model (no auth/encryption in v1, HMAC is future work), the value-size envelope, local-read semantics, the "not a lock/election service" warning, and a clock-drift runbook; (3) docs/EXAMPLES.md with grpcurl examples for every RPC, executed by CI against the shipped binary; (4) release: CHANGELOG.md, goreleaser workflow, tag v1.0.0; (5) the overdue branch-protection ruleset re-apply folded in as a task.

Scope anchor: requirements DOCS-01, DOCS-02, DOCS-03 plus the ROADMAP success criterion "release tag". All product code already exists (Phases 2–6); this phase adds documentation, CI checks over documentation, and release tooling. No new product capability.

</domain>

<decisions>
## Implementation Decisions

### Ops docs (DOCS-02)
- **D-01:** One file — `docs/OPERATIONS.md` — carries the trust model, value-size envelope, local-read semantics (read-your-writes on the same node only), "not a lock/election service" warning, and the clock-drift runbook as a section. No split runbook file. — **Reversibility:** one-way — published operational contract, URLs get bookmarked/linked.
- **D-02:** README.md carries: what sup-kv is, the verified quickstart, and links to docs/OPERATIONS.md and docs/EXAMPLES.md. Ops detail is NOT duplicated inline.
- **D-03:** All documentation in English, matching existing README.md and docs/IDEA.md.

### API examples (DOCS-03)
- **D-04:** grpcurl-only walkthrough in `docs/EXAMPLES.md`: every RPC (Get/Set/Delete/Watch) including a Watch stream and a `wait_for_replication` write. No Go example programs, no examples/ source dir. Works against the shipped binary via the already-enabled gRPC reflection.
- **D-05:** A CI test spins up two nodes from the built binary and executes the documented example commands, asserting outputs — examples cannot silently rot after release. (grpcurl availability in CI is a planner concern.)

### Quickstart verification (DOCS-01)
- **D-06:** A CI test extracts the fenced quickstart command block(s) from README.md and executes them **verbatim** against the built binary — the README is the test input, so the doc cannot drift from reality. Runs on `ubuntu-latest` and `macos-15` (both already in the CI gate matrix) to satisfy "works on Linux and macOS". — **Reversibility:** costly — the extraction format (fenced blocks, markers) becomes a contract between README and CI.
- **D-07:** The kill −9 / restart / convergence sequence is part of what CI executes — the full DOCS-01 contract is machine-checked, not just the happy path. (The phase-6 exec harness remains as prior evidence; this check re-proves it in the README's own words.)
- **D-08:** The quickstart's first client is grpcurl — one mental model shared with docs/EXAMPLES.md.

### Release mechanics
- **D-09:** Tag `v1.0.0` (full SemVer) triggers a goreleaser workflow: 4-target archives + checksums + SBOM attached to a GitHub Release. The tech-stack research already sanctioned adopting goreleaser at exactly this phase ("fine to adopt at the v1.0 release phase — but do not block Phase 0 on it"). goreleaser is release tooling, not a runtime dependency; pin it as a go.mod tool directive consistent with the repo's tooling conventions. — **Reversibility:** costly — the release pipeline and artifact naming become the download contract.
- **D-10:** Keep-a-Changelog style `CHANGELOG.md` seeded with the v1.0.0 entry summarizing what shipped across Phases 1–6 (engine, persistence, replication, gRPC API, hardening/observability/benchmarks).
- **D-11:** The deferred branch-protection server-side ruleset re-apply (ratified override expired 2026-10-31; one command per docs/setup/branch-protection.md) is a task in this phase's plans.

### Claude's Discretion
- Exact goreleaser target list (align with the CI 4-cell matrix: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64) and archive naming.
- README structure beyond quickstart+links (status section update, planned-highlights list refresh, badges if any).
- Fate of docs/IDEA.md after release (keep as design doc with a note, or leave untouched) — nothing user-visible was decided.
- How the quickstart-extraction test locates the fenced blocks (markers vs code-fence language tags) — but the test must consume README.md itself, not a copy.
- CHANGELOG entry wording.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` §v1 Requirements — DOCS-01, DOCS-02, DOCS-03 (lines ~89–91) and the requirement-coverage table
- `.planning/ROADMAP.md` §Phase Details → Phase 7 — goal, dependencies, success criteria
- `.planning/PROJECT.md` — product framing, core value, constraints (trust model, scale envelope)

### Design decisions that docs must state truthfully
- `docs/IDEA.md` — original product idea; ops-doc trust model, drift behavior, and warnings derive from it
- `.planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-CONTEXT.md` — D-10 (conflict journal CANCELLED — merge losers are silent, counter only), D-01..D-09 (metrics/slog operational contracts), D-11/D-12 (exec harness, darwin loud-skip precedent)
- `.planning/phases/05-grpc-api-and-binary/05-CONTEXT.md` — gRPC surface, Watch, startup gate, wait_for_replication semantics for examples
- `.claude/CLAUDE.md` §Constraints — "Scale envelope" (whole state in RAM, values ≤ 60 KiB, one broadcast segment, trusted LAN) is the envelope ops docs must publish

### Release/infra
- `docs/setup/branch-protection.md` — the one-command ruleset re-apply (D-11)
- `.github/workflows/ci.yml` — existing gate matrix (ubuntu-latest, ubuntu-24.04-arm, macos-15, darwin-amd64-cross); the goreleaser workflow and doc-check tests extend this CI
- `Makefile` — existing lint/test/bench/build targets; new release targets follow this pattern
- `cmd/sup-kv/main.go` — the full flag/env surface the quickstart and ops docs document

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase-6 exec harness (`cmd/sup-kv/scenario_test.go` and friends): builds the real binary, drives two processes over real gRPC, already proves kill −9/restart convergence — the pattern (and much of the machinery) for D-05/D-06/D-07 CI checks
- gRPC reflection (enabled) — grpcurl works against the shipped binary with zero extra tooling in the docs
- `--version` flag prints version, commit, Go runtime — goreleaser version stamping should stay consistent with it
- Makefile target convention (lint/test/bench/generate/build) — add release targets in the same style

### Established Patterns
- Everything ships through PRs with CI on every push; docs changes are no exception
- PR-only development on `master`; the release tag is pushed after merge (planner sequences tag vs ruleset re-apply vs merge order)
- Loud-skip precedent on darwin for platform-broken socket scenarios — the quickstart check must run for real on macos-15, not loud-skip (the wildcard-bind issue was resolved for the scenarios in question)

### Integration Points
- `.github/workflows/` — new goreleaser release workflow (tag-triggered) plus the doc-verification tests riding the existing gate matrix
- README.md, docs/ — the three documents plus CHANGELOG.md are the user-facing deliverables
- git remote `github.com:Djarvur/sup-kv.git` — GitHub Release and branch-protection ruleset live there

</code_context>

<specifics>
## Specific Ideas

- "The README literally cannot drift from reality" — D-06's extraction test is the point of the phase: documentation correctness enforced the same way code correctness is (red gate blocks merge).
- goreleaser adoption was pre-sanctioned by the stack research at exactly this milestone moment; it stays release tooling, never a runtime dep.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 7-v1.0 — Docs, Examples and Release*
*Context gathered: 2026-11-16*
