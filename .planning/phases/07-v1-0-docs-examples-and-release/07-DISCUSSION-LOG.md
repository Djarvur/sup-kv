# Phase 7: v1.0 — Docs, Examples and Release - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-16
**Phase:** 7-v1.0 — Docs, Examples and Release
**Areas discussed:** Docs structure & README scope, API examples format, Quickstart verification, Release mechanics

---

## Docs structure & README scope

| Option | Description | Selected |
|--------|-------------|----------|
| Single ops file | One docs/OPERATIONS.md with trust model, envelope, local-read semantics, not-a-lock warning, clock-drift runbook as a section | ✓ |
| Split runbook out | docs/OPERATIONS.md + separate docs/DRIFT-RUNBOOK.md for on-call linking | |
| All in README | Everything in README.md — max discoverability, large README | |

| Option | Description | Selected |
|--------|-------------|----------|
| Quickstart + links | README = what it is + verified quickstart + pointers to docs | ✓ |
| Warn inline too | README also carries trust-model warning and envelope inline | |

| Option | Description | Selected |
|--------|-------------|----------|
| English | Match existing README/docs/IDEA.md, open-source audience | ✓ |
| Russian | For maintainer's own operational convenience | |
| EN + RU ops sheet | English primary + short Russian ops cheat-sheet | |

**User's choice:** Single ops file · Quickstart + links · English
**Notes:** Matches the project's small scope — one place to look, one file to keep current.

---

## API examples format

| Option | Description | Selected |
|--------|-------------|----------|
| grpcurl only | Shell commands for every RPC incl. Watch and wait_for_replication; works via reflection | ✓ |
| Go programs only | Compiled examples/ dir kept honest by CI go build | |
| Both | grpcurl walkthrough + one compiled Go example | |

| Option | Description | Selected |
|--------|-------------|----------|
| docs/EXAMPLES.md | One walkthrough document linked from README | ✓ |
| README section | Under the quickstart — max visibility, README grows | |
| Split by form | Snippets in EXAMPLES.md, Go programs under examples/ | |

| Option | Description | Selected |
|--------|-------------|----------|
| CI executes them | CI test spins up two nodes, runs documented commands, asserts outputs | ✓ |
| Manual at UAT | Verified by hand during UAT only | |

**User's choice:** grpcurl only · docs/EXAMPLES.md · CI executes them
**Notes:** grpcurl needs zero toolchain beyond itself and matches the reflection-enabled design.

---

## Quickstart verification

| Option | Description | Selected |
|--------|-------------|----------|
| Execute README | CI extracts fenced quickstart commands from README.md, executes verbatim on ubuntu-latest and macos-15 | ✓ |
| Mirror test | Dedicated CI test re-implements the quickstart steps | |
| Manual at UAT | Verified during phase UAT on the maintainer's machine only | |

| Option | Description | Selected |
|--------|-------------|----------|
| Include in check | kill −9 / restart / convergence is part of what CI executes | ✓ |
| Lean on harness | CI checks happy path only; kill −9 convergence trusted from phase-6 exec harness | |

| Option | Description | Selected |
|--------|-------------|----------|
| grpcurl | Same tool as docs/EXAMPLES.md, one mental model | ✓ |
| Own CLI | Would require new product code in a docs phase | |

**User's choice:** Execute README · Include in check · grpcurl
**Notes:** The extraction test makes the README itself the test input — the doc cannot drift from reality.

---

## Release mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| goreleaser + Release | Tag push triggers goreleaser: 4-target archives + checksums + SBOM on GitHub Release | ✓ |
| Plain tag only | Annotated git tag v1.0.0, no artifacts, users `go install` | |
| Manual artifacts | GitHub Release with hand-attached go build artifacts | |

| Option | Description | Selected |
|--------|-------------|----------|
| CHANGELOG.md | Keep-a-Changelog style seeded with v1.0.0 entry | ✓ |
| Release notes only | Notes live on the GitHub Release page | |

| Option | Description | Selected |
|--------|-------------|----------|
| Fold it in | Branch-protection ruleset re-apply becomes a phase task | ✓ |
| Out of scope | Maintainer re-applies the ruleset outside GSD | |

| Option | Description | Selected |
|--------|-------------|----------|
| v1.0.0 | Full SemVer, patch room from day one | ✓ |
| v1.0 | Matches the milestone label exactly | |

**User's choice:** goreleaser + Release · CHANGELOG.md · Fold it in · v1.0.0
**Notes:** goreleaser adoption was pre-sanctioned by the stack research for exactly this phase. The ruleset override expired 2026-10-31 (16 days overdue at discussion time).

---

## Claude's Discretion

- goreleaser target list and archive naming (align with CI 4-cell matrix)
- README structure beyond quickstart+links (status section, highlights refresh)
- Fate of docs/IDEA.md after release
- Fenced-block location strategy for the README extraction test
- CHANGELOG entry wording

## Deferred Ideas

None — discussion stayed within phase scope.
