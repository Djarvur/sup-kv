---
phase: 09-documentation-and-release
plan: 03
subsystem: documentation
tags: [docs, stack-audit, agents-md, claude-md, make-help, dependabot, branch-protection, doc-anchor-tests, tdd]

requires:
  - phase: 09-documentation-and-release (09-01, 09-02)
    provides: the doc-is-test-input tracer shape and the green cmd/sup-kv doc-anchor suite these audit tests join
  - phase: 03-persistence (XPORT-05 via Phase 3 transport work)
    provides: the internal/udprepl/socket.go buffer setup that settles M-40's verdict
  - phase: 07-observability-and-configuration
    provides: the ci.yml/Makefile buf-gate shape M-49 corrects and the executed-claim discipline M-51 follows
provides:
  - cmd/sup-kv/stack_audit_test.go — TestAgentsStackAudit, TestClaudeStackAudit, TestAgentsStackAuditPackageInventory: the bidirectional stack audit comparing AGENTS.md and .claude/CLAUDE.md against go.mod and `go list ./...` at test time
  - AGENTS.md and .claude/CLAUDE.md as-built (M-32…M-44): go 1.27.4, the tools-directly CI sentence, the five-package internal/app split, four named adapters, the two direct runtime requires, the seven named tool directives, the as-built stack table with the GSD marker source re-pointed at go.mod + go list ./...
  - The six hygiene closures: M-48/M-23 comment scrubs, M-49 buf comments (dead Makefile probe branch removed), M-50 dependabot comment, M-51 the executed by-id ruleset check with recorded output, M-52 make help
affects: [09-04 (the registry sweep cites these M closures), 09-07 (owns the next ci.yml change, incl. the leftover cat-file probe), 09-09 (lands docs/ARCHITECTURE.md and the second architecture pointer)]

actuals:
  tokens: 15492
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "line-scoped tool-directive inventory: the audit parses only lines naming the tool directives, so each doc keeps the complete seven-name list on one source line and ordinary prose backticks never count"
    - "derived-not-typed expectations: the audit's expected go directive, tool set and package inventory are read from go.mod and `go list ./...` at test time, so the expectation moves with the module"

key-files:
  created:
    - cmd/sup-kv/stack_audit_test.go
    - .planning/phases/09-documentation-and-release/red-evidence/09-03-task1-red.json
  modified:
    - AGENTS.md
    - .claude/CLAUDE.md
    - internal/app/storage/doc.go
    - internal/app/storage/store.go
    - internal/app/repl/core.go
    - Makefile
    - .github/dependabot.yml
    - .github/workflows/ci.yml
    - docs/setup/branch-protection.md

key-decisions:
  - "the tool-directive inventory is a single source line in AGENTS.md's dependency policy and one table row in the CLAUDE.md stack section — the audit scans lines naming the tool directives, so the list is machine-scoped rather than scattered through prose"
  - "M-40's SetReadBuffer advice is verified against internal/udprepl/socket.go (XPORT-05 sets SO_RCVBUF/SO_SNDBUF and reads the kernel values back) and kept with as-built wording — the advice became accurate when Phase 3 shipped, so it stays"
  - "M-51 is an executed claim: the by-id command ran 2026-11-28 and printed exactly the documented shape, recorded verbatim beside the command in branch-protection.md"
  - "M-23 is verify-then-delete on the Go side: zero journal references survive in storage/repl production comments, so the only fix was the CLAUDE.md inspectability line (Task 2)"
  - "the ci.yml half of M-49 touches comment lines only and leaves the pre-Phase-6 cat-file probe in the recipe body with an honest comment — the plan scopes ci.yml changes to comments because plan 09-07 owns the next workflow change"
  - "the conventions/architecture pointers name AGENTS.md alone because docs/ARCHITECTURE.md does not exist yet (09-09 lands it); the plan's conditional wording applied"

patterns-established:
  - "stack-audit bidirectionality: a doc-cited path or tool that does not exist fails, and so does a module package or tool the docs omit — a dependency or package change without a doc edit fails make test"

requirements-completed: []

coverage:
  - id: D1
    description: "the stack-audit guards: TestAgentsStackAudit (go directive, path existence with a >=15 vacuity floor, tool inventory both ways, three forbidden v1.0 sentences), TestClaudeStackAudit (marker contract fatal, slice directive/paths/tools, five stale research claims, two whole-file not-yet lines) and TestAgentsStackAuditPackageInventory (every `go list ./...` package named in AGENTS.md)"
    requirement: DOCS-08
    verification:
      - kind: unit
        ref: "cmd/sup-kv/stack_audit_test.go#TestAgentsStackAudit"
        status: pass
      - kind: unit
        ref: "cmd/sup-kv/stack_audit_test.go#TestClaudeStackAudit"
        status: pass
      - kind: unit
        ref: "cmd/sup-kv/stack_audit_test.go#TestAgentsStackAuditPackageInventory"
        status: pass
    human_judgment: false
  - id: D2
    description: "AGENTS.md states the as-built stack (M-32…M-37): go 1.27.4, the tools-directly CI sentence, the five-package internal/app split plus both harness packages, the four named adapters with stand/config outside the set, the two direct runtime requires and the seven named tool directives; deviations 1-3 and the .golangci.yml provenance prose byte-identical"
    requirement: DOCS-08
    verification:
      - kind: unit
        ref: "cmd/sup-kv/stack_audit_test.go#TestAgentsStackAudit"
        status: pass
      - kind: other
        ref: "git diff -- .golangci.yml empty; grep 'go 1.27.4' AGENTS.md green"
        status: pass
    human_judgment: false
  - id: D3
    description: ".claude/CLAUDE.md's stack section (M-38…M-44) is the as-built table (no rapid/synctest/MarshalIndent/ReadBuildInfo/v0.23.0 in the slice), the marker source names go.mod + go list ./..., the not-yet lines are replaced with AGENTS.md pointers, and the inspectability line drops the never-built conflict journal (M-23's CLAUDE half)"
    requirement: DOCS-08
    verification:
      - kind: unit
        ref: "cmd/sup-kv/stack_audit_test.go#TestClaudeStackAudit"
        status: pass
      - kind: other
        ref: "greps: no pgregory.net/rapid, no 'Conventions not yet established', no 'conflict journal' in .claude/CLAUDE.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "the M-48 comment scrubs state the real boundary in storage/doc.go, storage/store.go and repl/core.go (the consumer-side Clock/StateStore rationale kept, the false 'internal/port holds no interfaces' claim dropped, whitespace-normalized grep clean); M-23's Go side verified and found already gone"
    requirement: DOCS-01
    verification:
      - kind: other
        ref: "grep -rn 'port holds no interfaces' internal/ empty (raw and whitespace-normalized)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ ./internal/app/repl/ (both ok)"
        status: pass
    human_judgment: false
  - id: D5
    description: "M-49/M-50/M-52: the Makefile buf-breaking comment and recipe perform the real comparison (dead probe branch removed, make buf-breaking green); the ci.yml buf-gate comment states the current reality (comments only); the dependabot comment states the public-since-2026-11-18 reality with alerts active (vulnerability-alerts probe HTTP 204); make help lists every target from the ## target: comments"
    requirement: DOCS-01
    verification:
      - kind: other
        ref: "make buf-breaking exit 0; make help lists lint/test/release-snapshot; 'self-destructs' and 'GHAS-gated on this private repo' greps empty"
        status: pass
    human_judgment: false
  - id: D6
    description: "M-51 as an executed claim: the by-id ruleset verify command ran once (2026-11-28) and its exact output {\"enforcement\":\"active\",\"name\":\"protect master\",\"rules\":[\"pull_request\",\"required_status_checks\",\"non_fast_forward\",\"deletion\"]} is recorded beside the command in docs/setup/branch-protection.md"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "gh api repos/Djarvur/sup-kv/rulesets/27576625 --jq '{name, enforcement, rules: [.rules[].type]}' exit 0; output matches the recorded block"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-11-28
status: complete
---

# Phase 9: Documentation and Release — Plan 03 Summary

**The contributor stack docs are as-built and machine-audited in both directions, and the six hygiene rows (M-48/M-49/M-50/M-51/M-52 + M-23's halves) are closed, one as an executed claim with recorded output.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-11-28T00:48:00+03:00
- **Completed:** 2026-11-28T01:18:46+03:00
- **Tasks:** 3
- **Files modified:** 11 (2 created, 9 modified)

## Accomplishments

- The stack-audit guard landed RED-first and is green: all three tests compare the docs against go.mod and `go list ./...` read at test time — a doc-cited path/tool that does not exist fails, and a module package/tool the docs omit fails too — so a dependency or package change without a doc update fails `make test` (DOCS-08's machine-checkable audit shape).
- AGENTS.md (M-32…M-37) and the .claude/CLAUDE.md stack section (M-38…M-44) state the as-built truth: `go 1.27.4`, CI runs the tools directly, `internal/app` is five packages plus both harness packages, the four adapters are named with `internal/stand`/`internal/config` outside the set, grpc/protobuf are the two direct requires, the seven tool directives are named; deviations 1–3 and the `.golangci.yml` provenance prose are byte-identical and `.golangci.yml` is untouched.
- The M-23 inspectability line, M-48's three false "internal/port holds no interfaces" comment sites, M-49's self-destruct buf-guard prose (dead Makefile branch removed), M-50's private-repo dependabot comment, M-51's broken verify command (replaced with the proven by-id form, executed, output recorded) and M-52's missing `make help` are all closed.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the stack-audit test comparing docs to the module** — `b6f77fc` (test)
2. **Task 2: Fix the stack sections — AGENTS.md (M-32…M-37) and .claude/CLAUDE.md (M-38…M-44, M-23 half)** — `2a66559` (feat)
3. **Task 3: Hygiene sweep — M-48/M-23 comment scrubs, M-49, M-50, M-51 executed, M-52 make help** — `af1e3eb` (docs)

**Plan metadata:** the SUMMARY/docs commit follows.

_Note: Task 1 is the plan's TDD task; its RED commit precedes the GREEN (docs) commit that satisfies it._

## M-closure dispositions (D-02's per-plan table)

| M row | Disposition |
|-------|-------------|
| M-23 | `.claude/CLAUDE.md:21` inspectability line drops the conflict journal (doc fix); Go side verify-then-delete — zero journal survivors in storage/repl production comments; IDEA.md rows stay for the 09-04 sweep's banner dispositions |
| M-32 | AGENTS.md and the CLAUDE.md stack section cite `go 1.27.4` — go.mod's directive value, derived at test time (doc fix, audit-enforced) |
| M-33 | the Conventions bullet states the Makefile gate exactly (config verify + run; `-race -cover`) and that CI runs the tools directly, calling no make target (doc fix) |
| M-34 | `internal/app` is five named packages plus `internal/app/internal/{simnet → named individually}, testutil` and `internal/testutil`; the audit requires every `go list ./...` package to be named (doc fix) |
| M-35 | the four adapters are named with their roles and `internal/stand`/`internal/config` are placed outside the adapter set (doc fix) |
| M-36 | the dependency policy states the two direct runtime requires and the historical Phase-1 count without pretending Phase 1 is current (doc fix; the FULL v1.0 sentence is a forbidden claim) |
| M-37 | the seven tool directives are named and the ~500 `// indirect` consequence stated (doc fix; the audit compares the tool inventory both ways) |
| M-38 | `pgregory.net/rapid` gone; the hand-rolled seeded `math/rand/v2` generators in `internal/app/internal/testutil` are the stated truth (doc fix) |
| M-39 | the simulator is `internal/app/internal/simnet`, a discrete-event scheduler with no goroutines and no wall clock (doc fix) |
| M-40 | verified against `internal/udprepl/socket.go`: XPORT-05 ships the buffer setup, so the advice is accurate and kept with as-built wording (verify-and-keep) |
| M-41 | dumps are `json.Encoder` one-line JSONL (doc fix) |
| M-42 | version identity is ldflags-stamped (`-X main.version`, `-X main.commit`) (doc fix) |
| M-43 | x/sync v0.25.0, the golangci-lint-action row is marked not-a-pinned-tool, protoc-gen-go is one module with its runtime (doc fix) |
| M-44 | the not-yet conventions/architecture lines are replaced with AGENTS.md pointers; the goreleaser v2.20.2 and grpcurl v1.9.4 tool rows exist (doc fix) |
| M-48 | the three "internal/port holds no interfaces" sites state the real boundary; consumer-side Clock/StateStore rationale kept (code-comment fix) |
| M-49 | Makefile comment+recipe simplified (dead probe branch removed, real comparison runs green); ci.yml buf-gate comment fixed, body left for 09-07 (doc/comment fix) |
| M-50 | the dependabot comment states the repo is public since 2026-11-18 and alerts are active (comment fix; alerts probe HTTP 204) |
| M-51 | the verify command is the by-id form; executed once with the exact output recorded in the doc (executed claim) |
| M-52 | `make help` added to the Makefile and executed — lists every `## target:` target including lint/test/release-snapshot (build-file fix) |

## Files Created/Modified

- `cmd/sup-kv/stack_audit_test.go` — the honest-scope stack-audit guard: go.mod/`go list` inventory readers, the line-scoped tool-citation extractor, path-existence and tool-set comparators, the three tests and the forbidden-claim tables
- `.planning/phases/09-documentation-and-release/red-evidence/09-03-task1-red.json` — the RED evidence record (classifier RED_EVIDENCE_OK)
- `AGENTS.md` — as-built Conventions/Layout/Dependency-policy prose; deviations 1–3 and the provenance prose byte-identical
- `.claude/CLAUDE.md` — as-built stack section between the GSD markers (source re-pointed), conventions/architecture pointers, inspectability line fix
- `internal/app/storage/doc.go`, `internal/app/storage/store.go`, `internal/app/repl/core.go` — the M-48 boundary-comment scrubs (prose only)
- `Makefile` — the M-52 `help` target (from the `## target:` convention) and the M-49 buf-breaking simplification
- `.github/dependabot.yml` — the M-50 comment
- `.github/workflows/ci.yml` — the M-49 buf-gate comment (comments only)
- `docs/setup/branch-protection.md` — the M-51 by-id command, the failure explanation and the recorded executed output

## Decisions Made

- The tool-directive inventory lives on a single source line per doc (AGENTS.md dependency-policy bullet; one CLAUDE.md table row) because the audit scopes tool citations to lines naming the tool directives — machine-scoped inventory, no prose scanning.
- Expected values are derived at test time (go directive and tool block parsed from go.mod; packages from `go list ./...`), so the audit stays correct if the module moves between planning and execution — the plan's stated assumption.
- M-49's split: the Makefile recipe lost its dead pre-Phase-6 probe branch (the plan's action: remove the dead branch, keep the plain comparison, `make buf-breaking` proven green against origin/master), while ci.yml got comment-only edits per the plan's read-first constraint that 09-07 owns the next workflow change; the leftover probe carries an honest comment naming that.
- The CLAUDE.md conventions/architecture pointers name AGENTS.md alone because `docs/ARCHITECTURE.md` does not exist at edit time (the plan's conditional wording; 09-09 adds the second pointer).
- M-40's verdict: `internal/udprepl/socket.go` sets SO_RCVBUF/SO_SNDBUF and reads the kernel values back (XPORT-05), so the research's SetReadBuffer advice is kept and aligned with the as-built behavior.
- REQUIREMENTS.md checkboxes are untouched: DOCS-08 and DOCS-10 are composite requirements whose other slices belong to sibling plans, and DOCS-01 stays pending on the #2388 shared-ID gate.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The plan's brace-form harness path is not an existing path**
- **Found during:** Task 2 (AGENTS.md Layout rewrite)
- **Issue:** the plan's gloss writes `internal/app/internal/{simnet,testutil}`; a brace path does not exist in the tree and the audit's path-existence check would reject it.
- **Fix:** the doc names both paths literally (`internal/app/internal/simnet`, `internal/app/internal/testutil`).
- **Files modified:** AGENTS.md
- **Verification:** TestAgentsStackAudit path-existence check green
- **Committed in:** 2a66559 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 Rule-1 plan-gloss correction)
**Impact on plan:** cosmetic plan-gloss adaptation; no scope creep.

## TDD Gate Compliance

- Task 1 (`tdd="true"`) has its RED commit (`b6f77fc`, `test(09-03)`) before the GREEN commit (`2a66559`, `feat(09-03)`, which descends from the RED); the RED evidence record passed `gsd_run check tdd-red-evidence` with verdict RED_EVIDENCE_OK (all three tests failing on the stale docs' planned assertions: the go-directive mismatch, the vacuous tool inventories, the three forbidden v1.0 sentences and the eight unnamed module packages). Tasks 2 and 3 are `type="auto"` and carry one commit each. No REFACTOR commit was needed.

## Issues Encountered

- The first RED run's `go list ./...` executed with the test binary's cwd (cmd/sup-kv) and returned 1 package, tripping the audit's own vacuity floor — the RED run caught it; the exec now runs with `cmd.Dir` at the module root and the recorded RED evidence is from the corrected run (go vet green, failures exactly on the planned assertions).
- TestClaudeStackAudit's RED reported the slice path-vacuity floor before its tool/forbidden checks (deliberate Fatal order); the AGENTS-side complaints (go directive, tool vacuity, three forbidden sentences) carried the rest of the RED evidence, and the slice checks all run against the rewritten section.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 09-04's registry sweep can cite these M closures (the disposition table above is the D-02 source), and the audit tests ride `go test -race ./...`, so stack drift now breaks CI.
- 09-07 owns the next ci.yml change: the leftover pre-Phase-6 cat-file probe and its honest comment are named in the workflow and in the decisions above.
- 09-09 lands `docs/ARCHITECTURE.md` and should add the second pointer to `.claude/CLAUDE.md`'s architecture section (deferred per the plan's conditional wording).
- No blockers. DOCS-08 and DOCS-10 stay pending on their remaining sibling slices; DOCS-01 stays pending on the #2388 shared-ID gate declared by the phase's sibling plans.

---
*Phase: 09-documentation-and-release*
*Completed: 2026-11-28*
