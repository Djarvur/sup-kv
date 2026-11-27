# Phase 9: Documentation and Release - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-27
**Phase:** 9-Documentation and Release
**Areas discussed:** Registry closure and mismatch tracking, New document set and machine checks, Release mechanics, Code-side items riding the phase

**Mode note:** run from /gsd-manager with empty manager flags; the user instructed
"continue in order, stop only when answers are unavoidable". All gray areas were
auto-selected and every question auto-answered with the option grounded in the
settled evidence base (owner decisions 14.1–14.11, ADR 0010, the registry preamble,
review D-01, Q-09 routing, and phases 7–8's --auto precedent). No interactive
AskUserQuestion turns were held; every selection below is an `[auto]` choice.

---

## Registry closure and mismatch tracking

| Option | Description | Selected |
|--------|-------------|----------|
| Distribute flips back into completed phase plans | Phases 2–8 reopen to flip their own rows | |
| One dedicated registry-audit plan | Resweep the whole table with per-row evidence; stale citations and a machine check ride the same pass | ✓ |
| Flip without evidence | Fastest, violates the registry's own contract | |

**User's choice:** [auto] One dedicated registry-audit plan.
**Notes:** The registry preamble prescribes "Phase 9 greps it for unclosed rows" — the
closure check is a test. M-01…M-52 tracked in per-plan disposition tables
(docs-audit.md stays the source list; no new sibling registry).

---

## New document set and machine checks

| Option | Description | Selected |
|--------|-------------|----------|
| New docs/RUNBOOKS.md + docs/METRICS.md + docs/LOGS.md | Three new files, three new anchor tests | |
| OPERATIONS.md extended | One operator reference; existing TestOperationsDoc anchoring continues | ✓ |

**User's choice:** [auto] OPERATIONS.md extension; a docs/METRICS.md split stays planner
discretion if a section grows past readable size.
**Notes:** Machine-check tests follow the cmd/sup-kv doc-anchor pattern
(readme/operations/api/examples tests are precedent). CONFIGURATION.md / DEPLOYMENT.md /
FORMATS.md paths are fixed by REQUIREMENTS; API.md extends in place; the systemd unit and
DEPLOYMENT.md run in CI and ship in release archives.

---

## Release mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| Drop darwin builds entirely | Stricter than 14.10 asks | |
| Keep darwin archives but annotate | Weakest reading of the demotion | |
| Development-only per ADR 0010 | Buildable, README states it, darwin archives leave the release asset set, CI keeps compile/vet leg | ✓ |

**User's choice:** [auto] Development-only per ADR 0010 (decision 14.10 canonical).
**Notes:** CHANGELOG hand-written in the 1.0.0 prose style (no chglog generation);
phase/decision numbers scrubbed from existing CHANGELOG text (the 1.0.0 entry carries
`(Phase 2)`-style suffixes) and proto comments (scout grep found none today — verify).
The v1.1.0 tag lands as the phase's final plan after all gates (ROADMAP criterion 4).

---

## Code-side items riding the phase

| Option | Description | Selected |
|--------|-------------|----------|
| Blind delete of named targets | Risks deleting already-removed symbols | |
| Verify-then-delete | Scout grep found no lastLocal/ownHandle in non-test code — verify presence first | ✓ |

**User's choice:** [auto] Verify-then-delete.
**Notes:** Stack sections of AGENTS.md and .claude/CLAUDE.md audited against go.mod /
go list ./...; the ruleset check command in docs/setup/branch-protection.md executed as
written with the result recorded.

---

## Claude's Discretion

- Exact file split of operator references (OPERATIONS.md vs docs/METRICS.md)
- Machine-check test names and package placement details
- goreleaser archive shape for the demoted darwin
- Whether M-01…M-52 dispositions get one consolidated summary table across plans
- Mechanism for the metric-rename-breaks-CI coupling (string grep vs structured iteration)

## Deferred Ideas

None — discussion stayed within phase scope.
