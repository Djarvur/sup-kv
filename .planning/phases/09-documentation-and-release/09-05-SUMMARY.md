---
phase: 09-documentation-and-release
plan: 05
subsystem: documentation
tags: [operator-reference, failure-table, runbooks, metrics-reference, log-catalog, doc-anchor-tests, tdd, m-closure]

requires:
  - phase: 09-documentation-and-release (09-01…09-04)
    provides: the CONFIGURATION.md/FORMATS.md as-built truth the new sections link, the green cmd/sup-kv doc-anchor suite the new checks join, and the master M-assignment (this plan owns the OPERATIONS halves of M-05/M-06/M-07/M-15)
  - phase: 07-observability-and-configuration
    provides: the TestCatalogLevels renderers the log-catalog check reuses and the exporter's complete-catalog rule
provides:
  - docs/OPERATIONS.md completed as the one operator reference (D-03, no docs/METRICS.md split): the Spec's 18-row failure-behavior table keyed S-FAIL-01…18 in registry order, seven new runbooks in the clock-drift shape, the typed metrics reference and the full log-message catalog — every cell anchored to the binary
  - cmd/sup-kv/metrics_catalog_test.go — TestMetricsReferenceCatalog (two-directional metrics cross-check against metrics.Snapshot over zero sources, label values included), TestOperationsLogCatalog (every catalog literal is a main.go call site at its documented level) and TestOperationsFailureTable (18 rows in order; metric column against the snapshot vocabulary, log column against main.go)
  - cmd/sup-kv/operations_test.go anchor upkeep — the Phase 9 rows (section headings, seven runbook titles, honest none-log-only cell, conditional bound, three absences, named tests, CONFIGURATION.md pointer) plus the forbidden unconditional-bound sentence
  - the OPERATIONS halves of M-05/M-06/M-07/M-15 closed in place, and DOCS-09's binary-verification coupling made real (a metric rename or label change without a doc edit fails make test)
affects: [09-06…09-10 (their M rows and doc edits must keep these checks green), 09-08 (TestProvenClaims scans the OPERATIONS text this plan authored), 09-11 (milestone verification)]

actuals:
  tokens: 14200
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "two-directional doc cross-check with vacuity floors: the reference is parsed into family→{type, label values} rows and compared against metrics.Snapshot in both directions; either side empty, or fewer than 20 label-value comparisons, is fatal — the DOCS-09 coupling mechanism"
    - "level-prefixed source grep as the call-site check: the sloglint static-msg discipline makes every slog message a literal first argument, so `slog.Warn(\"<literal>` in main.go is the exact level-anchored presence proof; log.Print/Printf map to INFO via the SetDefault redirect and the usage path greps its `os.Stderr, \"` form"

key-files:
  created:
    - cmd/sup-kv/metrics_catalog_test.go
    - .planning/phases/09-documentation-and-release/red-evidence/09-05-task1-red.json
  modified:
    - docs/OPERATIONS.md
    - cmd/sup-kv/operations_test.go

key-decisions:
  - "TestOperationsDocFamilies is RETAINED, not folded: TestMetricsReferenceCatalog is stronger per name (type + label matching against Snapshot) but section-scoped to ## Metrics reference; the whole-doc exporter grep still covers family mentions in the runbooks, Monitoring and the failure-table prose — together the two checks close both gaps, and the deletion bar (equal or stronger coverage) is not met"
  - "metrics.Snapshot(metrics.Sources{}) as written in the plan would panic — Snapshot dereferences its accessor functions — so the test mirrors internal/metrics' own package-internal zeroSources() helper with zero-returning closures; the complete-catalog rule (all label values rendered at zero state) is what the check actually leans on"
  - "the vacuity floor for supkv_ families is 23, not the plan's ≥24: the exporter renders 23 supkv_ families plus 5 runtime ones (the plan's estimate was a research miscount); direction A still fails on any single missing family, so the floor's vacuity purpose is intact"
  - "the log-catalog test parses the ## Log-message catalog section (the section the plan's own Task 2 mandates), not the ## Lifecycle logs section the behavior bullet named — Lifecycle logs stays the attribute-contract prose and is anchored as before"
  - "the catalog's stderr rows (the usage path's plain text, not a slog record) carry the level cell `stderr` and are checked by their own `os.Stderr, \"` source form — the level matrix stays INFO/WARN/ERROR/DEBUG for slog records while the usage path stays machine-checked too"

patterns-established:
  - "the honest no-metric cell: a failure-table row without a dedicated metric carries the literal `none — log only` (or names the covering counter) — the table never invents a family to fill a cell, and the check enforces the form"

requirements-completed: [DOCS-01, DOCS-04, DOCS-07, DOCS-09]

coverage:
  - id: two-directional-catalog
    description: "The typed metrics reference cross-checked in BOTH directions against metrics.Snapshot (every exported family/label documented; every documented family/label exported), with vacuity floors"
    requirement: DOCS-07
    verification:
      - kind: tests
        ref: cmd/sup-kv/metrics_catalog_test.go#TestMetricsReferenceCatalog
        status: pass
      - kind: tests
        ref: cmd/sup-kv/metrics_catalog_test.go (RED evidence red-evidence/09-05-task1-red.json, classifier RED_EVIDENCE_OK)
        status: pass
    human_judgment: false
  - id: failure-table
    description: "The Spec's 18-row failure-behavior table keyed S-FAIL-01…18 in registry order, metric and log columns machine-checked against Snapshot and main.go"
    requirement: DOCS-04
    verification:
      - kind: tests
        ref: cmd/sup-kv/metrics_catalog_test.go#TestOperationsFailureTable
        status: pass
    human_judgment: false
  - id: log-catalog
    description: "The log-message catalog covering the binary's message literals with their levels, checked against main.go's call sites"
    requirement: DOCS-07
    verification:
      - kind: tests
        ref: cmd/sup-kv/metrics_catalog_test.go#TestOperationsLogCatalog
        status: pass
      - kind: tests
        ref: cmd/sup-kv/catalog_internal_test.go#TestCatalogLevels (the shared renderers' level pin)
        status: pass
    human_judgment: false
  - id: seven-runbooks
    description: "Seven new runbooks in the clock-drift shape, each anchored to metric/log evidence that exists in the binary"
    requirement: DOCS-04
    verification:
      - kind: tests
        ref: cmd/sup-kv/operations_test.go#TestOperationsDoc (the seven ## Runbook: heading anchors)
        status: pass
      - kind: command
        ref: "grep -cE '^## Runbook: ' docs/OPERATIONS.md = 7 new + 1 clock drift"
        status: pass
    human_judgment: false
  - id: m-closure-halves
    description: "M-05 (proven sites name their tests), M-06 (conditional ack_timeout_max bound), M-15 (three absences + only TLS surface), M-07 (CONFIGURATION.md link) closed in place"
    requirement: DOCS-01
    verification:
      - kind: tests
        ref: cmd/sup-kv/operations_test.go#TestOperationsDoc (the M-05/M-06/M-15/M-07 anchors plus the forbidden unconditional-bound sentence)
        status: pass
    human_judgment: false
  - id: docs09-coupling
    description: "A metric rename or label change without the matching doc edit fails make test (DOCS-09's binary-verification coupling)"
    requirement: DOCS-09
    verification:
      - kind: command
        ref: "executed scratch rename supkv_dumps_total→supkv_dumps_total_renamed in the doc: TestMetricsReferenceCatalog FAILED in both directions (exported family missing a row; documented family not exported); restored green — observation recorded, not committed"
        status: pass
      - kind: tests
        ref: go test -race -count=1 ./cmd/sup-kv/ (whole package green, 22.5s)
        status: pass
    human_judgment: false
  - id: operator-reference-prose
    description: "The runbooks' recovery advice and the failure table's symptom/behavior/recovery glosses read as honest, as-built operator guidance"
    verification: []
    human_judgment: true
    rationale: "The wording matches the registry glosses and the cited as-built sources (sim_bound_test.go's bound comments, scheduler.go's ladder, wait.go's honest-false report), but 'the runbook reads well and correctly during an incident' is judgment over prose no substring check can establish — this is the phase's manual UAT territory, as the anchor tests' own honest-scope headers state."
---

# Phase 9 Plan 5: The Complete Operator Reference Summary

docs/OPERATIONS.md is now the one operator reference (D-03): the Spec's 18-row failure table, seven new runbooks, the typed metrics reference and the full log-message catalog — every cell machine-checked against the binary by three new two-directional tests in cmd/sup-kv, with the OPERATIONS halves of M-05/M-06/M-07/M-15 closed in place and DOCS-09's "a rename without a doc update breaks CI" coupling proven live.

## Performance

- **Duration:** 10 min
- **Started:** 2026-11-27T23:14:49Z
- **Completed:** 2026-11-27T23:24:55Z
- **Tasks:** 3
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- **Task 1 (RED, TDD).** `cmd/sup-kv/metrics_catalog_test.go` in package `main_test` with the honest-scope header: `TestMetricsReferenceCatalog` builds the family→{type, label values} vocabulary from `metrics.Snapshot` over zero sources and checks the `## Metrics reference` section in BOTH directions (an exported family without a row fails; a documented family that is not exported fails; a label value documented or omitted wrongly fails; floors: 23 supkv_ families in the snapshot, 15 doc rows, 20 label-value comparisons); `TestOperationsLogCatalog` parses the `## Log-message catalog` table and proves every literal is a main.go call site at its documented level (level-prefixed source greps per the static-msg discipline; log.Print/Printf → INFO via the SetDefault redirect; the usage path via its `os.Stderr, "` form); `TestOperationsFailureTable` enforces exactly 18 rows keyed S-FAIL-01…18 in registry order, checks the metric column against the snapshot vocabulary (honest `none — log only` form accepted) with label citations verified, and every backticked log-column literal against main.go. RED recorded (red-evidence/09-05-task1-red.json, classifier RED_EVIDENCE_OK): all three failed naming their missing sections; vet and lint clean, so the failures were assertion failures, not build faults.
- **Task 2 (GREEN).** docs/OPERATIONS.md extended after `## Lifecycle and exit codes` (trust model stays FIRST): `## Failure behaviour` with the 18 rows grounded in the registry's S-FAIL glosses and the shipped behavior (ladder windows, eviction, gate semantics, preservation, the honest false for a displaced wait), honest `none — log only` cells where no dedicated metric exists; the seven runbooks in the clock-drift Symptom/Cause/Action shape — dump failure and full disk, missing peer, growing unconfirmed set, write conflict, incompatible versions, corrupted dump (linking docs/FORMATS.md's census), duplicate node_id; `## Metrics reference` with 28 typed rows (23 supkv_ + 5 runtime), meanings mirroring the exporter HELP text, alerts only where the binary can observe the condition; `## Log-message catalog` with the complete 28-row level matrix including the redirected log-package records and the stderr usage path. In place: M-05 (both proven sites now name `TestBound_*`/`TestRed_R*`/`TestScenario_farAhead`), M-06 (the conditional bound: a loss episode recovers when its delivery opportunity falls inside the ladder's window — attempts at 200/600/1400/3000 ms — sustained loss is the partition-heal bound's territory, mirroring sim_bound_test.go's restriction comment), M-15 (the trust model states no authentication, no authorization, no transport security together, and names the metrics-endpoint mTLS triple as the only TLS surface), M-07 (the Configuration validation section points at docs/CONFIGURATION.md).
- **Task 3 (anchor upkeep + coupling proof).** operationsAnchors gains 21 Phase 9 rows (each new section heading, all seven runbook titles, `none — log only`, the conditional-bound wording and its partition-territory clause, the three absences and the only-TLS-surface note, the two named tests, the docs/CONFIGURATION.md pointer); operationsForbidden gains the old unconditional sentence (`bounds recovery at \`ack_timeout_max\` (default 5s)`). TestOperationsDocFamilies retained (decision recorded below). The DOCS-09 coupling proof was executed as a scratch edit: renaming `supkv_dumps_total` → `supkv_dumps_total_renamed` in the doc made TestMetricsReferenceCatalog fail in both directions ("exported family supkv_dumps_total has no row… direction A" / "the reference names supkv_dumps_total_renamed, which the exporter does not render… direction B"); the doc was restored and the test is green again — the observation is recorded here, nothing of the scratch state was committed.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the two-directional checks** - `9aab6db` (test)
2. **Task 2: the OPERATIONS extension (GREEN)** - `d4f15b8` (feat)
3. **Task 3: anchor upkeep + DOCS-09 proof** - `a6af9b8` (test)

_Note: TDD task 1 produced the RED commit; Task 2 is its GREEN._

## Files Created/Modified

- `cmd/sup-kv/metrics_catalog_test.go` - the three two-directional machine checks (new)
- `docs/OPERATIONS.md` - the failure table, seven runbooks, metrics reference, log catalog, and the four M-fixes
- `cmd/sup-kv/operations_test.go` - the Phase 9 anchor rows and the forbidden unconditional-bound sentence
- `.planning/phases/09-documentation-and-release/red-evidence/09-05-task1-red.json` - the classifier-validated RED record

## Decisions Made

- **TestOperationsDocFamilies retained** (the plan left its fate open): the two-directional check is stronger per family (type + label values against Snapshot) but scoped to the `## Metrics reference` section; the old grep's whole-doc scan still covers family mentions in the runbooks, Monitoring and the failure table. Deleting it would lose that coverage, so the "equal or stronger" deletion bar is not met — both checks stay, complementary.
- **Zero-source construction:** `metrics.Snapshot(metrics.Sources{})` as the plan wrote it would panic (Snapshot dereferences the accessor funcs); the test builds zero-returning closures mirroring the exporter's own `zeroSources()` test helper. The complete-catalog rule — all label values render at zero state — is what the check leans on, so the vocabulary is still exactly the plan's intent, never hand-typed.
- **Vacuity floor 23, not 24:** the exporter renders 23 supkv_ families (the plan's ≥24 carried a research miscount). The floor matches reality; direction A fails on any missing family, so nothing is weakened.
- **The catalog test parses `## Log-message catalog`** (the section Task 2 itself mandates), not the `## Lifecycle logs` name in the behavior bullet — Lifecycle logs remains the attribute-contract prose, unchanged and still anchored.
- **`stderr` level rows:** the usage path is not a slog record; its catalog rows carry the level cell `stderr` and are checked by their own source form (`os.Stderr, "`), keeping the usage path machine-checked without polluting the INFO/WARN/ERROR/DEBUG matrix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan fact] `metrics.Snapshot(metrics.Sources{})` panics — zero-source closures required**
- **Found during:** Task 1 (writing the vocabulary helper)
- **Issue:** the plan's vocabulary source as literally written (`Snapshot(metrics.Sources{})`) dereferences nil accessor functions and would panic, not render
- **Fix:** the test mirrors internal/metrics' own package-internal `zeroSources()` helper with zero-returning closures; the complete-catalog rule still renders the full family/label vocabulary
- **Files modified:** cmd/sup-kv/metrics_catalog_test.go
- **Verification:** the vocabulary builds at RED (the failing assertion was the missing section, after the vocabulary constructed) and at GREEN
- **Committed in:** 9aab6db

**2. [Rule 1 - Plan fact] The supkv_ family count is 23, not the plan's ≥24**
- **Found during:** Task 1 (vacuity floor selection)
- **Issue:** the plan's vacuity guard "≥24 supkv families in the snapshot" can never pass — the exporter renders 23 supkv_ families plus 5 runtime ones (a research miscount propagated into the plan)
- **Fix:** floor set to 23 with a comment naming the count; direction A still fails on any single missing family, so the vacuity purpose is intact
- **Files modified:** cmd/sup-kv/metrics_catalog_test.go
- **Verification:** TestMetricsReferenceCatalog green with 23 supkv_ + 5 runtime rows documented
- **Committed in:** 9aab6db

**3. [Rule 1 - Plan fact] The log-catalog test parses the section the plan's Task 2 mandates**
- **Found during:** Task 1
- **Issue:** the Task 1 behavior bullet says "parse the `## Lifecycle logs` section" while the Task 2 action mandates a new `## Log-message catalog` section — the two names disagree
- **Fix:** the test parses `## Log-message catalog` (the complete literal surface the must_haves describe); `## Lifecycle logs` keeps its attribute-contract role and existing anchors
- **Files modified:** cmd/sup-kv/metrics_catalog_test.go
- **Verification:** TestOperationsLogCatalog green over the 28-row catalog
- **Committed in:** 9aab6db

---

**Total deviations:** 3 auto-fixed (3 × Rule 1 plan-fact corrections, each making the plan's intent executable). **Impact on plan:** none on scope — the checks are strictly what the plan specified semantically; every floor and mechanism survives.

## Issues Encountered

None beyond the recorded deviations. Lint demanded one formatting pass (golines line widths) inside Task 1 before its RED evidence was taken, and one shortened forbidden-phrase comment in Task 3 — both cosmetic, both before their task's verify ran.

## Verification

- `go test -race -count=1 ./cmd/sup-kv/ -run 'TestMetricsReferenceCatalog|TestOperationsLogCatalog|TestOperationsFailureTable' -v` — PASS (all three; RED evidence at red-evidence/09-05-task1-red.json, classifier RED_EVIDENCE_OK).
- Task 2 verify 2: 18 `| S-FAIL-` rows, the duplicate-node_id and corrupted-dump runbook headings and `## Metrics reference` present — PASS.
- Task 2 verify 3: `TestOperationsDoc|TestCatalogLevels` green — PASS.
- Task 3 verify 1: the four doc/catalog tests plus `TestOperationsDocFamilies` green — PASS.
- Task 3 verify 2: the whole `./cmd/sup-kv/` package green under `-race` (22.5 s) — PASS.
- `go vet ./cmd/sup-kv/` and `go tool golangci-lint run ./cmd/sup-kv/` — clean.
- DOCS-09 coupling: scratch doc-side rename failed TestMetricsReferenceCatalog in both directions; restored green (recorded above, uncommitted by design).

## TDD Gate Compliance

- RED: `test(09-05)` commit 9aab6db precedes GREEN; all three target tests failed on their planned missing-section assertions; classifier verdict RED_EVIDENCE_OK.
- GREEN: `feat(09-05)` commit d4f15b8 makes all three pass under -race.
- REFACTOR: none needed (lint fixes landed inside the RED commit before its evidence was recorded).

## Self-Check: PASSED

- All three tasks executed and committed individually (9aab6db, d4f15b8, a6af9b8).
- All acceptance criteria re-run after the final edit: the three tests, the anchor suite, the 18-row/heading greps, vet and lint — all green.
- Success criteria: every failure row references a metric or log line that exists in the binary; every exported family and label is documented in both directions; seven runbooks join clock drift; the log catalog covers the binary's literals with their levels; M-05, M-06, M-15 and the M-07 link are closed in this file; the rename-breaks-CI observation is recorded.

Next: Ready for 09-06 (the API.md extension owns its assigned M halves).
