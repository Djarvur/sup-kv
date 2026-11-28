---
phase: 09-documentation-and-release
plan: 09
subsystem: documentation
tags: [architecture-doc, simulator-guide, adr-index, contributing, security, depguard, doc-anchor-tests, tdd]

requires:
  - phase: 09-documentation-and-release (09-03)
    provides: the stack-audit guard whose `go list ./...` exec this plan's package-graph guard reuses, and the deferred second architecture pointer this plan lands
  - phase: 09-documentation-and-release (09-04)
    provides: the master M-01…M-52 assignment table this plan's disposition rows close their halves of
  - phase: 09-documentation-and-release (09-06)
    provides: the API.md key-prefix convention and client-pattern sections ARCHITECTURE.md points at (M-30's API half)
  - phase: 09-documentation-and-release (09-05)
    provides: the OPERATIONS trust-model wording SECURITY.md distills without contradicting
  - phase: 08-performance
    provides: the hardware-block ledger the DOCS-10 coverage check verifies
provides:
  - cmd/sup-kv/architecture_test.go — TestArchitectureDocPackages (every `go list ./...` package named in docs/ARCHITECTURE.md, all six depguard rule IDs present, 19-package vacuity floor), TestADRIndex (bidirectional file↔row correspondence over docs/adr/000*.md frontmatter, numeric order, rows link their files, 12-row floor) and TestArchitectureDocDivisionOfLabor (the ARCHITECTURE→AGENTS conventions pointer, AGENTS.md's contract intact, the vendored-config provenance never restated)
  - docs/ARCHITECTURE.md — the as-built description IDEA.md's banner promises: one paragraph per `go list ./...` package grounded in its own doc comment, the enforced-boundaries table, the parameter surface (M-29 architecture half), the three data-flow paths, the client-interop conventions (M-30 architecture half)
  - docs/SIMULATOR.md — the simulator guide: simnet's contracts, the scenario suites, seed replay (SUPKV_SIM_SEED, printed replay command, fixed corpus + fresh seeds), adding a scenario, the three Makefile entry points
  - docs/adr/README.md — the ADR index over 0001–0012 derived from each file's frontmatter
  - CONTRIBUTING.md — the real dev loop pointing at AGENTS.md as the canonical conventions file
  - SECURITY.md — the trust-model distillation with the three absences stated identically to OPERATIONS and the private disclosure path
  - the second architecture pointer in .claude/CLAUDE.md (the 09-03 deferred conditional)
affects: [09-10 (extends the release surface; owns the release-procedure pointer CONTRIBUTING deferred), 09-11 (the release gate greps this summary for the M-29 and M-30 dispositions and runs the completed doc set's verification)]

actuals:
  tokens: 16000
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "derived-at-test-time guards: the package-graph guard reuses the stack audit's `go list ./...` exec (same package) so the expectation moves with the module; the ADR index parser derives rows from the files' own `adr:` frontmatter — neither guard carries a hand-typed inventory"
    - "correspondence-both-ways indexing: an index row is a table line whose first cell starts with the 4-digit ADR number (bracketed or bare), so dates and years in other cells never count; file→row and row→file both fail loudly, rows must link their file and stay in numeric order"

key-files:
  created:
    - cmd/sup-kv/architecture_test.go
    - docs/ARCHITECTURE.md
    - docs/SIMULATOR.md
    - docs/adr/README.md
    - CONTRIBUTING.md
    - SECURITY.md
    - .planning/phases/09-documentation-and-release/red-evidence/09-09-task1-red.json
  modified:
    - .claude/CLAUDE.md

key-decisions:
  - "the package-graph guard reuses goListPackages from stack_audit_test.go (same package) instead of a second exec helper — one `go list ./...` seam per test package; its own 19-package floor pins the planning-time inventory, so a graph shrink demands a conscious floor edit"
  - "the ADR index's first-cell rule (`^\\[?(\\d{4})\\]?`) keeps the parser honest against dates: the Decision/Informs/Date cells never carry the matched number, and each row must contain its file's basename so a row points at the decision it indexes"
  - "CONTRIBUTING's release section references the Makefile release targets alone (release-snapshot as the mandatory pre-tag gate, release as the local invocation, the tag-triggered release.yml) because docs/setup/release-procedure.md does not exist at edit time — the plan's read_first conditional applied; 09-10 lands the procedure and adds the pointer"
  - "SECURITY.md mirrors OPERATIONS' three absences verbatim (no authentication / no authorization / no transport security, identical claims) and links the trust model rather than paraphrasing a different contract (T-09-25); the flagged assumption stands — the GitHub private security advisory on Djarvur/sup-kv is the maintainer-operated channel, and the doc's reporting section is the only edit needed if that changes"
  - "M-29's architecture half is stated as the parameter surface's role (one parse/validate/wire point: internal/config → cmd wiring) pointing at docs/CONFIGURATION.md as the flag table — never a second copy of the table; M-30's architecture half names the key-prefix convention and the client-side last-read-value cache as client-interop conventions pointing at docs/API.md — one copy of each pattern"
  - "the forbidden-phrase guard (golangci-lint-strict, powerman/) enforces the duplication prohibition on ARCHITECTURE.md: rule IDs are named, the vendored-config provenance and the boundary justifications stay in AGENTS.md only"

patterns-established:
  - "doc-is-test-input extends to the as-built architecture claim: a package added or removed without an ARCHITECTURE.md paragraph fails `make test` (TestArchitectureDocPackages), and an ADR added without an index row (or a row without a file) fails it too (TestADRIndex)"

requirements-completed: [DOCS-01, DOCS-08, DOCS-10]

coverage:
  - id: D1
    description: "the package-graph and division-of-labor guards in cmd/sup-kv/architecture_test.go: every `go list ./...` package named in docs/ARCHITECTURE.md, the six depguard rule IDs present, the 19-package vacuity floor, the AGENTS.md conventions pointer, AGENTS.md's role intact, the provenance never restated"
    requirement: DOCS-08
    verification:
      - kind: unit
        ref: "cmd/sup-kv/architecture_test.go#TestArchitectureDocPackages"
        status: pass
      - kind: unit
        ref: "cmd/sup-kv/architecture_test.go#TestArchitectureDocDivisionOfLabor"
        status: pass
    human_judgment: false
  - id: D2
    description: "the ADR-index correspondence guard plus docs/adr/README.md: one row per ADR 0001–0012 derived from frontmatter, bidirectional file↔row check, numeric order, rows link their files, 12-row and 12-file vacuity floors"
    requirement: DOCS-10
    verification:
      - kind: unit
        ref: "cmd/sup-kv/architecture_test.go#TestADRIndex"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/ARCHITECTURE.md — the as-built package graph (19 paragraphs grounded in the packages' own doc comments), the enforced-boundaries table with the six rule IDs, the parameter surface (M-29 architecture half, pointing at docs/CONFIGURATION.md), the three data-flow paths, the client-interop conventions (M-30 architecture half, pointing at docs/API.md) and the where-to-go-next links"
    requirement: DOCS-08
    verification:
      - kind: unit
        ref: "cmd/sup-kv/architecture_test.go#TestArchitectureDocPackages"
        status: pass
      - kind: other
        ref: "grep: docs/CONFIGURATION.md, docs/API.md, key-prefix convention, last-read-value cache all present in docs/ARCHITECTURE.md; TestArchitectureDocDivisionOfLabor green"
        status: pass
    human_judgment: false
  - id: D4
    description: "docs/SIMULATOR.md — simnet's own contracts, the scenario suites and go test entry points, seed replay (SUPKV_SIM_SEED, the printed replay command, the fixed corpus testdata/chaos-seeds.txt + fresh-seed model), adding a scenario with the assert-the-bound convention, and the three Makefile entry points (red-run via scripts/red-runner + scripts/red-expectations.tsv with the mutation stage, race-soak, perf)"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "plan verify greps: SUPKV_SIM_SEED, red-runner/red-run, '## Adding a scenario', AGENTS.md-in-ARCHITECTURE all green; race-soak, make perf, chaos-seeds.txt, SUPKV_SIM_FRESH_SEEDS, red-expectations.tsv present"
        status: pass
    human_judgment: false
  - id: D5
    description: "CONTRIBUTING.md (the real dev loop: make lint = config verify + run, make test = -race -cover, CI runs the tools directly; PR-only flow; make proto; make client-example; go tool for everything; AGENTS.md pointed at as canonical) and SECURITY.md (supported version v1.1.x; the three absences stated identically to OPERATIONS' trust model; CRC32 integrity-only; reflection always on; the metrics mTLS triple the only TLS surface; GitHub private security advisory as the private disclosure path; the OPERATIONS link)"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "plan verify greps: AGENTS.md in CONTRIBUTING.md; 'no authentication', docs/OPERATIONS.md, 'security advisory' in SECURITY.md — all green"
        status: pass
    human_judgment: false
  - id: D6
    description: "the DOCS-10 benchmark-results coverage check against docs/PERFORMANCE.md's Phase-8 ledger: every hardware-carrying section carries its producing command; result recorded with no gap found and no ledger edit needed"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "awk hardware↔command section check exit 0; 5 hardware blocks ≥ 3 floor (grep -c 'hardware: ' docs/PERFORMANCE.md)"
        status: pass
    human_judgment: false
  - id: D7
    description: "the five new documents' prose accuracy — every package paragraph grounded in the package's own doc comment and every claim mirrored from its source (OPERATIONS' trust model, simnet's doc comment, the Makefile target comments) — is authoring-time judgment the presence guards cannot prove"
    verification: []
    human_judgment: true
    rationale: "The guards prove coverage, correspondence and pointer presence only (their honest-scope headers say so); the truth of the prose rests on the grounding discipline recorded in key-decisions and needs the phase's human verification to confirm."

duration: 13min
completed: 2026-11-28
status: complete
---

# Phase 9 Plan 9: Maintainer and Community Documentation Summary

**The as-built architecture description, the simulator guide, the ADR index, CONTRIBUTING and SECURITY land as five new documents, machine-bound to the live repo by three new guards (package-graph coverage, ADR file↔row correspondence, the ARCHITECTURE/AGENTS division of labor) — RED-then-GREEN, with the PERFORMANCE.md benchmark-results coverage verified at five hardware blocks and no gap.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-11-28T00:47:09Z
- **Completed:** 2026-11-28T01:00:28Z
- **Tasks:** 3
- **Files modified:** 8 (7 created, 1 modified)

## Accomplishments

- **The guards (Task 1, TDD RED).** `cmd/sup-kv/architecture_test.go` in package `main_test` with the honest-scope header (coverage and correspondence only — not prose quality): `TestArchitectureDocPackages` reuses the stack audit's `goListPackages` exec, requires every `go list ./...` package named in docs/ARCHITECTURE.md above a 19-package vacuity floor, and requires all six depguard rule IDs; `TestADRIndex` derives each ADR's number from its own `adr:` frontmatter and checks the index bidirectionally (file without row, row without file), in numeric order, with each row linking its file, above 12-row/12-file floors; `TestArchitectureDocDivisionOfLabor` anchors the ARCHITECTURE→AGENTS conventions pointer, AGENTS.md's intact contract, and forbids the vendored-config provenance (`golangci-lint-strict`, `powerman/`) from ARCHITECTURE.md. RED evidence recorded (red-evidence/09-09-task1-red.json, classifier `RED_EVIDENCE_OK`): both target tests failed naming the missing docs, `go vet` green.
- **The architecture description and simulator guide (Task 2, GREEN).** docs/ARCHITECTURE.md: overview (leaderless AP store, LWW-map, HLC/UUIDv7 versions, the core value statement), one paragraph per `go list ./...` package grouped in layers (public API, decision core, core test harness, ports, adapters, composition, test harness) each grounded in the package's own doc comment, the enforced-boundaries table naming the six rule IDs with what each forbids and pointing at AGENTS.md for the justifications, the parameter surface (M-29's architecture half: one parse/validate/wire point, docs/CONFIGURATION.md is the flag table), the three data-flow paths (local write through dump-confirm/broadcast/Outbox-ack/wait resolution, the replication receive path through validation/clock-absorption/LWW-or-stale-answer, the persistence path through debounce/streaming encode/atomic rotate), the client-interop conventions (M-30's architecture half: key-prefix convention and last-read-value cache pointing at docs/API.md), and the where-to-go-next links. docs/SIMULATOR.md: simnet's own contracts (discrete-event (Time, Seq) scheduler, no goroutines/wall clock, same production handlers, one seeded rng, TraceDigest), the scenario suites and entry points, seed replay (SUPKV_SIM_SEED, the printed replay command, the fixed 16-seed corpus + fresh-seed model with the corpus regeneration constant), adding a scenario (NewCluster → drive virtual time → assert the bound, the assert-the-bound convention, the deterministic drain sweep), and the three Makefile entry points each deferring to its target's own comment as the authoritative flags.
- **The ADR index, CONTRIBUTING, SECURITY and the coverage check (Task 3).** docs/adr/README.md indexes 0001–0012 from frontmatter (ADR | Decision | Informs | Date | Status, numeric order, every row linking its file). CONTRIBUTING.md documents the dev loop exactly as the Makefile and ci.yml carry it (lint = config verify + run; test = `-race -cover`; CI runs the tools directly, never make), the PR-only branch flow with the ruleset doc pointer, `make proto`, `make client-example`, the `go tool` discipline, AGENTS.md pointed at as canonical, and the release section referencing the Makefile targets alone (the release-procedure doc does not exist yet; 09-10 lands it). SECURITY.md states the supported line (v1.1.x), mirrors the three absences verbatim from OPERATIONS' trust model (no authentication / no authorization / no transport security, the metrics mTLS triple the only TLS surface), keeps CRC32 integrity-only and reflection-always-on, names the GitHub private security advisory as the private disclosure path, and links docs/OPERATIONS.md rather than paraphrasing a different contract. The DOCS-10 benchmark-results coverage check ran against docs/PERFORMANCE.md: five hardware blocks, every hardware-carrying section carries its producing command — no gap; the Phase-8 ledger already satisfies the clause, no ledger edit needed.

## DOCS-01 disposition rows (D-02's per-plan table — the tokens 09-11's gate greps)

| M row | Disposition |
|-------|-------------|
| M-29 | doc fix, closing plan 09-09 — the ARCHITECTURE parameter-surface half: docs/ARCHITECTURE.md states the as-built parameter surface's role (internal/config registers and validates; cmd/sup-kv's wire is the only consumer) and points at docs/CONFIGURATION.md as the flag table, never duplicating it. 09-06 owns the API.md RPC-list half. |
| M-30 | doc fix, closing plan 09-09 — the ARCHITECTURE half of the two-half row: the key-prefix convention and the client-side last-read-value cache are named as client-interop conventions with the pointer to docs/API.md's documented pattern, never restated. 09-06 owns the API half; with both halves the conventions leave IDEA.md-only status. |

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the package-graph guard and the ADR-index correspondence check** - `784b33d` (test)
2. **Task 2: Author docs/ARCHITECTURE.md and docs/SIMULATOR.md** - `534e327` (feat)
3. **Task 3: Author the ADR index, CONTRIBUTING.md and SECURITY.md; verify the PERFORMANCE.md coverage claim** - `139e4e6` (docs)

Follow-up commits riding the plan:

4. **The 09-03 deferred second architecture pointer in .claude/CLAUDE.md** - `1093676` (docs)
5. **The golines wrap of the over-length t.Errorf** - `2403a07` (style)

**Plan metadata:** the SUMMARY/docs commit follows.

_Note: Task 1 is the plan's TDD task; its RED commit precedes the GREEN commit that satisfies it, and the GREEN descends from the RED (linear history)._

## Files Created/Modified

- `cmd/sup-kv/architecture_test.go` - the three guards (package-graph coverage, ADR correspondence, division of labor) with the honest-scope header and vacuity floors
- `docs/ARCHITECTURE.md` - the as-built description: package graph, enforced boundaries, parameter surface, data flow, client-interop conventions, links
- `docs/SIMULATOR.md` - the simulator guide: what the simulator is, running scenarios, seed replay, adding a scenario, Makefile entry points
- `docs/adr/README.md` - the ADR index over 0001–0012, rows derived from frontmatter
- `CONTRIBUTING.md` - the dev loop, branches/PRs, proto changes, client example, tools, conventions pointer, release targets
- `SECURITY.md` - supported versions, the trust-model distillation, the private disclosure path
- `.claude/CLAUDE.md` - the architecture section names docs/ARCHITECTURE.md as the as-built deep description (09-03's deferred pointer)
- `.planning/phases/09-documentation-and-release/red-evidence/09-09-task1-red.json` - the RED evidence record (classifier RED_EVIDENCE_OK)

## Decisions Made

- The package-graph guard reuses `goListPackages` from stack_audit_test.go (same package) — one `go list ./...` seam per test package; the derived-not-typed rule from 09-03 carries over.
- The ADR index parser's first-cell rule keeps dates from counting as ADR numbers, and the row→file link requirement makes every row point at the decision it indexes.
- CONTRIBUTING's release pointer defers to the Makefile targets alone per the plan's read_first conditional (docs/setup/release-procedure.md does not exist at edit time; 09-10 lands it and adds the pointer).
- SECURITY.md mirrors OPERATIONS' trust-model wording verbatim for the three absences and links it — the no-contradiction rule from the plan's threat model (T-09-25).
- The flagged assumption stands as authored: the GitHub private security advisory on Djarvur/sup-kv is the disclosure channel; only the doc's reporting section needs an edit if a different channel is ever wanted.
- The 09-03 deferred conditional was executed: .claude/CLAUDE.md's architecture section now names docs/ARCHITECTURE.md (TestClaudeStackAudit green after the edit).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The golines gate flagged an over-length t.Errorf call**
- **Found during:** the final gate (make lint)
- **Issue:** the division-of-labor guard's error call exceeded the 200-char golines max-len — the full `make lint` failed after the initial per-file runs had passed
- **Fix:** `go tool golangci-lint fmt ./cmd/sup-kv/` wrapped the call; no assertion or message semantics changed
- **Files modified:** cmd/sup-kv/architecture_test.go
- **Verification:** `make lint` green end to end
- **Committed in:** 2403a07

---

**Total deviations:** 1 auto-fixed (1 formatting violation the lint gate caught).
**Impact on plan:** none — formatting only; the RED evidence record predates the wrap and its semantics are unchanged.

## TDD Gate Compliance

- RED: `test(09-09)` commit 784b33d precedes GREEN; both target tests failed on their planned assertions (the missing docs/ARCHITECTURE.md and docs/adr/README.md named in the failure messages), classifier verdict `RED_EVIDENCE_OK` on red-evidence/09-09-task1-red.json.
- GREEN: `feat(09-09)` commit 534e327 makes TestArchitectureDocPackages and TestArchitectureDocDivisionOfLabor pass under -race (TestADRIndex stays red until Task 3's index lands, as the task boundary planned).
- REFACTOR: none needed; the style commit (2403a07) is a formatting fix, not a refactor.

## Issues Encountered

None beyond the lint-gate deviation above.

## Verification

- `go test -race -count=1 ./cmd/sup-kv/ -run 'TestArchitectureDocPackages|TestADRIndex|TestArchitectureDocDivisionOfLabor'` — PASS (all three guards green).
- Plan verify greps — PASS: SUPKV_SIM_SEED / red-runner / Adding-a-scenario / AGENTS.md greps; the Task 2 M-29/M-30 pointer greps (docs/CONFIGURATION.md, docs/API.md, key-prefix convention, last-read-value cache); the Task 3 greps (AGENTS.md in CONTRIBUTING; no authentication / docs/OPERATIONS.md / security advisory in SECURITY).
- PERFORMANCE.md coverage check — PASS: awk hardware↔command section check exit 0; `grep -c 'hardware: '` == 5 (≥ 3 floor). No gap found; disposition recorded above.
- `go vet ./cmd/sup-kv/` — clean (RED gate).
- `make lint` (config verify + `go tool golangci-lint run ./...`) — green after the formatting fix.
- `go test -race -count=1 ./cmd/sup-kv/` (full package, 26.8s) — PASS; every sibling test stays green with the new docs and guards.

## Self-Check: PASSED

- All three tasks executed; each committed individually (784b33d, 534e327, 139e4e6) plus the two follow-up commits (1093676, 2403a07).
- All acceptance criteria re-run after the final edit: the guards green, the greps green, the coverage check green, the full package green.
- The M-29 and M-30 disposition tokens are recorded in the DOCS-01 table above for 09-11's completeness grep.

## Next Phase Readiness

- 09-10 lands the release surface (goreleaser demotion + files:, the release procedure document, the replproto scrub) and should add CONTRIBUTING's release-procedure pointer when docs/setup/release-procedure.md exists.
- 09-11's release gate finds the M-29 and M-30 dispositions in this summary; the doc set it ships is complete and machine-guarded.
- No blockers. DOCS-01 and DOCS-10 stay pending on their remaining sibling slices (the #2388 shared-ID gate); DOCS-08's architecture-description slice is closed by this plan.

---
*Phase: 09-documentation-and-release*
*Completed: 2026-11-28*
