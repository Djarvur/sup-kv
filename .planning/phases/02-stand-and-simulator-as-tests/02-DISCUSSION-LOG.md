# Phase 2: Stand and Simulator as Tests - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-19
**Phase:** 2-Stand and Simulator as Tests
**Mode:** --auto (all areas auto-selected; recommended option chosen for every question)

---

## Red-test gating

| Option | Description | Selected |
|--------|-------------|----------|
| Env-gated skip + scheduled red-assert job | Red tests skip unless `SUPKV_RED_TESTS=1`; scheduled workflow asserts the named Criticals FAIL; Phases 3–5 flip expectations | ✓ |
| Build tags | Separate `-tags red` test build — duplicates the binary graph | |
| Separate always-red workflow without assertion | Red visible but not machine-checked | |

**User's choice:** [auto] Red-test gating — Q: "how do intentionally-red tests merge without breaking CI?" → Selected: "env-gated skip + scheduled job asserting red" (recommended default)
**Notes:** STATE.md's roadmap note explicitly deferred this mechanism choice to Phase 2 planning.

---

## Mutation sensitivity

| Option | Description | Selected |
|--------|-------------|----------|
| go build -overlay temp mutants | Mutants built via overlay in temp copy; production tree untouched | ✓ |
| In-place sed + revert | Mutates the working tree; git state juggling in CI | |

**User's choice:** [auto] Mutation sensitivity — Q: "how are mutants realized safely in CI?" → Selected: "go build -overlay temp mutants" (recommended default)

---

## Stand placement

| Option | Description | Selected |
|--------|-------------|----------|
| internal/stand package + linux build tag | Harness + scenarios in one package; darwin leg loud-skips | ✓ |
| test/ directory outside internal | Not importable by package tests; diverges from layout | |
| Extend existing cmd/sup-kv tests | Keeps plumbing where v1.0 put it but conflates binary and harness concerns | |

**User's choice:** [auto] Stand placement — Q: "where does the real-socket harness live?" → Selected: "internal/stand package + linux build tag" (recommended default)

---

## Scenario set, simulator shape, chaos corpus, seam shape, pinning set, CI topology

**User's choice:** [auto] each — recommended defaults, detailed as D-04…D-09 in CONTEXT.md (scenario set = review §15 verbatim; simnet additive extension; fixed seed corpus + fresh seeds; fsops interface fakes; full TEST-acceptance breadth; one scheduled workflow + tightened ci.yml).

**Notes:** All acceptance wording comes verbatim from REQUIREMENTS.md TEST-01…TEST-16; the reviewer's measured numbers are the encoded bounds.

---

## Claude's Discretion

- Helper APIs, test naming, seed-corpus format, runner-script details, red-evidence layout
- Which v1.0 two-process tests move into internal/stand

## Deferred Ideas

None — discussion stayed within phase scope.
