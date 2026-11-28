# Phase 9: Documentation and Release - Context

**Gathered:** 2026-11-27
**Status:** Ready for planning
**Mode:** auto-selected per user instruction ("continue in order, stop only when answers are unavoidable") — all gray areas auto-selected; every choice logged below and reviewable in git

<domain>
## Phase Boundary

The documentation becomes the as-built contract and the milestone ships: all 52 documented
mismatches (M-01…M-52) closed by a doc or code fix; `docs/CONFIGURATION.md`,
`docs/DEPLOYMENT.md` (+ systemd unit), `docs/FORMATS.md`, `docs/API.md` (extended), the Spec's
18-row failure-behavior table, seven runbooks, a metrics reference and a log-message catalog
land — every reference anchored to something that exists in the binary and machine-checked by
tests; the spec-deviation registry reaches zero unclosed rows (milestone exit criterion);
`AGENTS.md`/`.claude/CLAUDE.md` stack sections match the module; CONTRIBUTING, SECURITY, ADR
index, simulator guide, release procedure and benchmark results land; darwin is demoted to
development-only in README and release archives (14.10); phase/decision numbers are scrubbed
from proto comments and CHANGELOG; the ruleset check command works as written; the Q-09
dead-code deletion lands; v1.1 is released. Requirements DOCS-01…DOCS-10.

**Depends on:** Phase 8 (all fixes in; performance claims documented in `docs/PERFORMANCE.md`).

</domain>

<decisions>
## Implementation Decisions

### Registry closure and mismatch tracking (DOCS-01, DOCS-08)
- **D-01:** One dedicated registry-audit plan resweeps the whole `docs/SPEC-DEVIATIONS.md`
  table: the 22 remaining `fixed-in-v1.1(...)` rows flip to `implemented` with per-row
  evidence (their fixes landed in Phases 2–8; the flips were not all done per-phase), the
  stale line-citation resweep (Phase-7 re-verification New-3) rides the same pass, and the
  gate asserts zero `fixed-in-v1.1(` rows — machine-checked, not eyeballed, per the
  registry preamble's own "Phase 9 greps it for unclosed rows". Distributing flips back into
  completed phases 2–8 was rejected: those plans are verified and closed; a fresh single
  auditable pass is honest. — [auto] [Registry] Q: where do the flips happen? → Selected:
  one dedicated registry-audit plan over the whole table.
  — **Reversibility:** reversible — flips are table edits with git history.
- **D-02:** M-01…M-52 closure is tracked in per-plan disposition tables and plan summaries;
  `docs/reviews/2026-11-19-v1.0/docs-audit.md` stays the source list and gains no new
  sibling registry. Each M is closed by a doc fix or a code fix and the disposition names
  which. — [auto] [Tracking] Q: one consolidated file or per-plan tables? → Selected:
  per-plan disposition tables.

### Document set and machine checks (DOCS-02…07, DOCS-09)
- **D-03:** The Spec's 18-row failure-behavior table, the seven runbooks (dump failure/full
  disk, missing peer, growing unconfirmed set, write conflict, incompatible versions,
  corrupted dump, duplicate `node_id`), the metrics reference and the log-message catalog
  extend `docs/OPERATIONS.md` — it already hosts the clock-drift runbook, Monitoring and
  Lifecycle logs sections, and the `TestOperationsDoc` anchor infrastructure lives there.
  A separate `docs/METRICS.md` split is planner discretion only if a section grows past
  readable size; the default is one operator reference. — [auto] [Operator docs] Q: new
  files or OPERATIONS extension? → Selected: OPERATIONS.md extension.
  — **Reversibility:** reversible.
- **D-04:** The new machine-check tests follow the existing `cmd/sup-kv` doc-anchor pattern
  (`readme_test.go`, `operations_test.go`, `api_docs_test.go`, `examples_test.go` are the
  precedents): the registered-flags vs `docs/CONFIGURATION.md` table iterator, the
  metrics-families/log-catalog cross-check, the `docs/FORMATS.md` byte-identical encode
  test, the "proven" claim checker, and the metric-rename/default-change-breaks-CI coupling
  all live in that package's idiom unless a codec-level test (FORMATS encode) belongs beside
  the codec it pins. — [auto] [Machine checks] Q: test home? → Selected: cmd/sup-kv
  doc-anchor pattern.
- **D-05:** `docs/CONFIGURATION.md`, `docs/DEPLOYMENT.md` (+ systemd unit), `docs/FORMATS.md`
  are new files at the paths REQUIREMENTS names — not a decision, a fixed target. `API.md`
  extends in place to full per-RPC semantics, code tables, Watch contract, write-latency
  model and client patterns (with APIS-06). The systemd unit and DEPLOYMENT.md run in CI
  and ship in the release archives (goreleaser config gains them).
  — **Reversibility:** reversible.

### Release mechanics (DOCS-10, milestone exit)
- **D-06:** darwin demotion follows ADR 0010 / decision 14.10 exactly: darwin stays
  buildable as a development-only platform, README states the demotion, darwin archives
  leave the release asset set (`.goreleaser.yaml` change), CI keeps the darwin compile/vet
  leg. The ADR is canonical — the planner grounds the exact goreleaser shape in its wording.
  — [auto] [darwin] Q: demotion shape? → Selected: development-only per ADR 0010.
  — **Reversibility:** costly — the v1.1.0 release assets, once published, define the
  shipped contract; changing the asset set after publishing means a new release.
- **D-07:** The CHANGELOG gains a hand-written `1.1.0` entry in the established
  Keep-a-Changelog prose style (the 1.0.0 entry is the precedent; no chglog generation), and
  phase/decision numbers are scrubbed from the existing CHANGELOG text (the 1.0.0 entry
  carries `(Phase 2)`-style suffixes today) and from proto comments (the scout grep found
  none in `api/supkvapi/v1/supkvapi.proto` — the plan verifies and keeps it clean).
  — [auto] [Changelog] Q: generated or hand-written? → Selected: hand-written.
  — **Reversibility:** reversible.
- **D-08:** A release-procedure document documents the tag → goreleaser → asset-verification
  path, and the actual `v1.1.0` tag lands as the phase's final plan after all gates —
  ROADMAP success criterion 4 makes "the v1.1 release ships" part of this phase.
  — [auto] [Tag timing] Q: when does the tag land? → Selected: final plan after all gates.

### Code-side items riding the phase
- **D-09:** The Q-09 dead-code deletion (`lastLocal`/`cover`, exported `Parse`, `ownHandle`)
  is verify-then-delete: the scout grep found no `lastLocal`/`ownHandle` in non-test code —
  prior-phase rewrites may already have removed them — so the plan verifies current presence
  first and deletes only what remains, with tests adjusted. — [auto] [Dead code] Q: blind
  or verified deletion? → Selected: verify-then-delete.
  — **Reversibility:** reversible.
- **D-10:** `AGENTS.md` and `.claude/CLAUDE.md` stack sections are audited against `go.mod`
  and `go list ./...` and corrected (DOCS-08); the ruleset check command in
  `docs/setup/branch-protection.md` is executed as written and fixed if it fails.
  — [auto] [Stack audit] Q: scope? → Selected: both files audited, command executed.

### Claude's Discretion
- The exact file split of operator references (OPERATIONS.md vs `docs/METRICS.md`), the
  machine-check test names and package placement details, the goreleaser archive shape for
  the demoted darwin, whether M-01…M-52 dispositions get one consolidated summary table
  across plans, and the mechanism for the metric-rename-breaks-CI coupling (string grep of
  families/labels vs structured iteration).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` — DOCS-01…DOCS-10 verbatim acceptance
- `.planning/ROADMAP.md` — Phase 9 goal + 4 success criteria (the milestone exit criterion
  lives in criterion 3: "no unclosed rows")

### Review evidence
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — D-01 (the M-01…M-52 closure criterion: "closed
  by a doc fix or a code fix") and the documentation section
- `docs/reviews/2026-11-19-v1.0/docs-audit.md` — M-01…M-52 and DOC-01…26, the source list
  this phase closes

### Registry and decisions
- `docs/SPEC-DEVIATIONS.md` — the registry itself: status vocabulary preamble ("Phase 9
  greps it for unclosed rows"), the 22 `fixed-in-v1.1(...)` rows, and the stale citations
  from `internal/app/repl/core.go` growth
- `docs/adr/0010-darwin-development-only.md` — decision 14.10, canonical for the demotion
  shape (D-06)
- `docs/adr/` — the ADR 0001–0012 corpus the new ADR index documents
- `.planning/phases/07-observability-and-configuration/deferred-items.md` — re-verification
  New-3: the stale line-citation list this phase's registry audit resweeps
- `.planning/phases/08-performance/08-CONTEXT.md` — deferred section routing the dead-code
  deletion and registry resweep here

### Documents to extend or audit
- `README.md` — darwin demotion statement (14.10), macOS quickstart pair
- `docs/OPERATIONS.md` — failure table, runbooks, metrics reference, log catalog (D-03)
- `docs/API.md` — per-RPC semantics extension (DOCS-05)
- `docs/EXAMPLES.md`, `docs/PERFORMANCE.md` — cross-checked surfaces; PERFORMANCE.md already
  carries Phase-8 hardware blocks
- `docs/setup/branch-protection.md` — the ruleset check command that must work as written
- `AGENTS.md`, `.claude/CLAUDE.md` — stack sections audited against `go.mod` / `go list ./...`
- `CHANGELOG.md` — 1.1.0 entry + phase/decision-number scrub (D-07)

### Release and CI mechanics
- `.goreleaser.yaml` — build matrix (`linux, darwin` × `amd64, arm64`) and archives: darwin
  leaves the asset set, systemd unit + DEPLOYMENT.md join it (D-05, D-06)
- `.github/workflows/release.yml` — tag-triggered release the procedure document describes
- `Makefile` — `release`, `release-snapshot` targets; CI keeps its existing doc gates

### Code
- `cmd/sup-kv/readme_test.go`, `cmd/sup-kv/operations_test.go`, `cmd/sup-kv/api_docs_test.go`,
  `cmd/sup-kv/examples_test.go` — the doc-anchor test pattern the new machine checks follow
  (D-04)
- `internal/config/config.go` — the registered-flags source the CONFIGURATION.md iterator
  walks
- `internal/metrics/metrics.go` — the exported families/labels the metrics reference and its
  cross-check test cover
- `api/supkvapi/v1/supkvapi.proto` — comment scrub verification (scout grep found no
  phase/decision numbers today)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The doc-anchor test harness in `cmd/sup-kv` — reads repo docs and verifies claims against
  the binary; new machine checks extend it
- `TestOperationsDoc` fenced-block execution — the 18-row failure table and runbooks inherit
  the same verification
- Phase-8 `docs/PERFORMANCE.md` hardware-block pattern — the benchmark-results requirement
  (DOCS-10) already has its shape
- goreleaser as a go.mod tool directive and the tag-triggered `release.yml` — the release
  procedure documents what exists, not something new

### Established Patterns
- doc-is-test-input: docs cannot rot silently — CI executes fenced blocks and doc tests
  verify claims against the binary
- Registry status vocabulary is a fixed, machine-checkable enum — closure is a grep-able
  property, not an opinion
- Every published number carries a hardware block and the command that produced it

### Integration Points
- `release.yml` and `.goreleaser.yaml` — archives gain the systemd unit and DEPLOYMENT.md,
  lose darwin
- `Makefile` — new doc-check targets join `lint`/`test`/`release` conventions
- The registry gate — the milestone exit criterion reads the table, so the closure check
  belongs where CI runs it

</code_context>

<specifics>
## Specific Ideas

- The unclosed-row check is a test, not a manual review step — the registry preamble's own
  words prescribe it.
- The dead-code targets may already be gone; verify presence before deleting anything.
- CHANGELOG 1.0.0 prose carries `(Phase 2)`-style suffixes — the scrub is over existing text,
  not just new entries.
- The ruleset check command must be executed as written once, and its result recorded —
  "works as written" is an executed claim, not a reading claim.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 9-Documentation and Release*
*Context gathered: 2026-11-27*
