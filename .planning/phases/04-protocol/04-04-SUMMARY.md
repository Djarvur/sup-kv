---
phase: 04-protocol
plan: 04
subsystem: startup-gate-config
tags: [proto-02, startup-gate-empty, empty-disk-wait, gate-selection, harness-derivation, frozen-r03, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 03
    provides: "persist.Engine.KeyCount — the O(1), tombstone-inclusive, clone-free record count the selection reads"
  - phase: 04-protocol
    plan: 02
    provides: "grpcapi.ServerOptions.ListedKey, the listing-membership seam this plan wires to the adapter"
  - phase: 04-protocol
    plan: 01
    provides: "the complete-listing mirror behind Adapter.ListedKey/SyncArrived"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the stand harness, the frozen R-03 test (30 s extras) and the parameter-by-effect suite"
provides:
  - "the startup_gate_empty parameter end to end: --startup-gate-empty, SUPKV_STARTUP_GATE_EMPTY, the 60 s default, the config field and positivity validation through the usage path"
  - "the empty-vs-data backstop selection: one KeyCount() read after the synchronous engine load, before the gate resolver starts"
  - "ServerOptions.ListedKey wired to adapter.ListedKey in production"
  - "emptyGateArg in the stand harness: the empty gate derived from an explicit --startup-gate in the extras, yielding to an explicit --startup-gate-empty"
  - "the lone-empty-node, derivation and two-empty-node stand proofs, plus the 14-site cmd test adaptation"
affects:
  - "04-06's remaining ServerOptions wiring rides beside the ListedKey line added here"
  - "04-13 flips the frozen R-03 TSV rows; the harness derivation is what keeps that flip test-edit-free"
  - "Phase 7's one-point configuration validation owns cross-parameter relations; this plan checks positivity only"

actuals:
  tokens: 11377
  tasks: 2
  commits: 4
  plan_head_before: 7ae0338a0f3bc605cf01bb80a73618bdfe6ef517
  plan_head_after: 416eeb9c372dfd0195429064dc6dd2c1b80c1e3c

tech-stack:
  added: []
  patterns:
    - "Startup selection is one O(1) read at wiring time: the engine load is synchronous, so KeyCount is final before the resolver exists and no later code path can switch the backstop."
    - "Production never infers one parameter from another (ADR 0006): only the test harness derives the empty gate from an explicit normal gate, and the derivation yields to any explicit empty gate."
    - "A harness-synthesized flag is appended after the scenario extras so it beats the harness default, while a scenario that names the flag itself is left untouched."
    - "Executed documentation pins the timers it depends on: the demo pair names --startup-gate-empty so a lost first listing cannot outlast the block's own bounded wait."

key-files:
  created:
    - .planning/phases/04-protocol/red-evidence/04-04-task1.json
    - .planning/phases/04-protocol/red-evidence/04-04-task2.json
  modified:
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - internal/stand/stand.go
    - internal/stand/stand_test.go
    - internal/stand/stand_metrics_test.go
    - README.md
    - docs/EXAMPLES.md

key-decisions:
  - "Selection happens once in wire, after the synchronous engine load and before grpcapi.NewServer: gateBackstop := cfg.startupGate, replaced by cfg.startupGateEmpty only when KeyCount()==0. Tombstones count, and no production path copies startupGate into startupGateEmpty (ADR 0006's separate parameter)."
  - "checkStartupGates validates positivity only and names the offending flag; no cross-parameter relation is enforced this phase (Phase 7's one-point validation owns relations, Open Question 7)."
  - "emptyGateArg scans the extras for any explicit empty-gate form first and returns empty when found, so TestStand_startupGateEmptyLoneNode's 2 s is never overwritten; otherwise it takes the LAST --startup-gate= value (or the harness default) and appends the derived flag after the extras so later-flag-wins parsing gives it precedence."
  - "The derivation proof adds a 3 s executable row before the plan's conceptual 30 s/2 s row: a 30 s clause cannot distinguish a 30 s derivation from the 60 s default, while the 3 s row fails without the derivation (the RED evidence) and passes with it."
  - "The executed docs demos pin --startup-gate-empty=2s: an empty demo node otherwise waits 60 s when a first empty listing is lost, which flaked the docs tests and made their health wait misleading (Rule 3)."
  - "The selection test observes health timing only — an empty node closed past 300 ms and serving by 5 s, a dump-seeded node serving within 300 ms — never engine internals."

patterns-established:
  - "One-shot startup decisions read the loaded state at the single wiring point; the resolver receives only the resolved duration, so a later store change cannot alter the choice."
  - "Frozen-test protection lives in the harness, not the frozen file: a derived flag is appended only when the scenario did not name it, and the frozen R-03 file stays byte-untouched."
  - "A test whose bound cannot fail is a bug: the derivation proof includes a value small enough that the production default is observably different within the test's own deadline."

requirements-completed: [PROTO-02]

coverage:
  - id: D1
    description: "The startup_gate_empty parameter exists end to end: flag, SUPKV_STARTUP_GATE_EMPTY env twin, 60 s default, config field, and positivity validation that exits through the usage path naming the offending flag."
    requirement: PROTO-02
    verification:
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestMain_startupGateEmptyRange"
        status: pass
      - kind: other
        ref: "grep -c SUPKV_STARTUP_GATE_EMPTY / --startup-gate-empty over cmd/sup-kv/main.go (2/2)"
        status: pass
      - kind: other
        ref: "go build ./... && go vet ./..."
        status: pass
    human_judgment: false
  - id: D2
    description: "An empty store selects the empty backstop and a store with at least one record selects the normal one, decided once from an O(1) key count."
    requirement: PROTO-02
    verification:
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestMain_startupGateEmptySelection (empty node NOT_SERVING at 300 ms, SERVING by 5 s; dump-seeded node SERVING within 300 ms)"
        status: pass
    human_judgment: false
  - id: D3
    description: "ServerOptions carries ListedKey wired to adapter.ListedKey beside SyncArrived."
    requirement: PROTO-02
    verification:
      - kind: other
        ref: "grep -c 'ListedKey: *adapter.ListedKey' cmd/sup-kv/main.go (1); go build ./..."
        status: pass
    human_judgment: false
  - id: D4
    description: "Every existing --startup-gate=10ms cmd test site also passes --startup-gate-empty=10ms, and the full cmd package stays green under the race detector."
    requirement: PROTO-02
    verification:
      - kind: other
        ref: "normal==empty==15 (14 plan sites plus the selection test's seed run); empty >= 14"
        status: pass
      - kind: other
        ref: "go test -race -count=1 ./cmd/sup-kv/"
        status: pass
    human_judgment: false
  - id: D5
    description: "The stand harness derives --startup-gate-empty from an explicit --startup-gate in the extras, both starters use it, and an explicit --startup-gate-empty is never overridden."
    requirement: PROTO-02
    verification:
      - kind: integration
        ref: "internal/stand/stand_test.go#TestStand_startupGateEmptyDerivedFromExtras (3 s derivation row + frozen 30 s closed-at-2 s row)"
        status: pass
      - kind: integration
        ref: "internal/stand/stand_test.go#TestStand_startupGateEmptyLoneNode"
        status: pass
    human_judgment: false
  - id: D6
    description: "Two empty nodes exchange complete empty listings and both open inside a third of a 30 s empty backstop."
    requirement: PROTO-02
    verification:
      - kind: integration
        ref: "internal/stand/stand_test.go#TestStand_startupGateEmptyTwoEmptyNodes (both SERVING inside the 10 s servingDeadline against a 30 s backstop)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The frozen R-03 test file stays byte-untouched and the full stand/parameter suites stay green under the race detector."
    requirement: PROTO-02
    verification:
      - kind: other
        ref: "git diff --stat -- internal/stand/stand_red_test.go (empty)"
        status: pass
      - kind: other
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_' and -run '^TestConfig_parametersByEffect$'"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./cmd/sup-kv/ ./internal/stand/"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 04: An Empty-Disk Node Waits Its Own Gate Without Racing the Frozen Test

**A node that loaded no records now waits the separate 60 s `startup_gate_empty` backstop — chosen once from an O(1) record count after the synchronous engine load — while a node with data keeps the 1 s `startup_gate`, and the stand harness derives the empty gate from an explicit normal gate so the frozen R-03 30 s test stays deterministic without a single edit to its file.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-11-22T04:04:33Z
- **Completed:** 2026-11-22T04:14:51Z
- **Tasks:** 2
- **Files modified:** 9 (7 code/test/docs files plus 2 RED evidence records)

## Accomplishments

- PROTO-02's parameter is complete end to end: `--startup-gate-empty` with the `SUPKV_STARTUP_GATE_EMPTY` env twin, the documented 60 s default, the config field, and `checkStartupGates` rejecting a non-positive value through the usage path (exit 2) with the offending flag named (T-4-12).
- The selection is one O(1) read: `wire` computes `gateBackstop` from `engine.KeyCount()` after the synchronous load and before `grpcapi.NewServer`, so a node with any record — tombstone included — never consults the empty gate, and no later code path can switch it (T-4-14).
- The listed-key seam reached production: `ServerOptions.ListedKey` is wired to `adapter.ListedKey` beside `SyncArrived`, so 04-06's remaining wiring can ride the same literal.
- The harness rule that keeps the frozen test deterministic is executable, not aspirational: `emptyGateArg` derives the empty gate from an explicit `--startup-gate` and yields to any explicit `--startup-gate-empty`; the 3 s derivation row fails without it (RED timed out on the 60 s default) and passes with it, while the frozen 30 s shape stays closed at 2 s (T-4-13).
- Both ADR 0006 behaviors are pinned: a lone empty node with an explicit 2 s gate is still closed at 600 ms and then serves, and two empty nodes open in ~227 ms inside a 10 s bound that is a third of their 30 s backstop — only the complete empty-listing exchange can satisfy it.
- The 14 existing `--startup-gate=10ms` cmd sites now pass `--startup-gate-empty=10ms` (plus the selection test's seed run), and the full cmd package is green under `-race`; the frozen `stand_red_test.go` is byte-untouched.

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1 RED: failing startup-gate range test** - `34bb8b5` (test)
2. **Task 1 GREEN: parameter, validation, selection and ListedKey wiring** - `26d913b` (feat)
3. **Task 2 RED: selection test, stand proofs and the 14-site adaptation** - `ebba922` (test)
4. **Task 2 GREEN: harness derivation and demo gate pins** - `416eeb9` (feat)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `cmd/sup-kv/main.go` — `envStartupGateEmpty`, `defaultStartupGateEmpty` (60 s), the `config.startupGateEmpty` field, the flag registration, `errStartupGateRange`, `checkStartupGates` called from `finalizeConfig`, the `gateBackstop` selection in `wire`, and `ListedKey: adapter.ListedKey`
- `cmd/sup-kv/main_test.go` — `TestMain_startupGateEmptyRange`, `TestMain_startupGateEmptySelection` with the health-only helper, the 14 `--startup-gate-empty=10ms` adaptations, and the removed `unparam` suppression on `awaitServing`
- `internal/stand/stand.go` — `emptyGateArg`, `nodeArgsWithExtras`, the derivation appended by both starters, `startStandPair`'s variadic extras applied to both nodes, and the removed `unparam` suppression
- `internal/stand/stand_test.go` — the lone-node, derivation and two-empty-node proofs plus the re-derived empty-disk recovery comment
- `internal/stand/stand_metrics_test.go` — the startup-gate subtest comment names the derived empty gate it depends on
- `README.md` / `docs/EXAMPLES.md` — the executed demo pair pins `--startup-gate-empty 2s` and the health-wait comments describe the empty gate (Rule 3)
- `.planning/phases/04-protocol/red-evidence/04-04-task{1,2}.json` — persisted RED evidence records

## Decisions Made

See frontmatter `key-decisions`: one selection read at wiring time; production never infers one gate from the other; `emptyGateArg` yields to an explicit empty gate and takes the last explicit normal gate; the derivation proof needed a 3 s executable row beyond the plan's conceptual 30 s clause; the docs demos pin their empty gate; and the selection test observes health timing only.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The plan's 30 s/2 s derivation assertion could not fail without the derivation**
- **Found during:** Task 2 RED design
- **Issue:** With the 60 s production empty default, a node started with `--startup-gate=30s` is still closed at 2 s whether the harness derives 30 s or not, so the plan's conceptual assertion would pass in the exact state the test exists to reject.
- **Fix:** Added a first subtest with an explicit 3 s normal gate: it fails (5 s serving wait expires on the 60 s default) before `emptyGateArg` exists and passes after it. The plan's 30 s/2 s frozen-shape row is kept as the second subtest.
- **Files modified:** `internal/stand/stand_test.go`
- **Verification:** RED `ebba922` — the 3 s row timed out at 5 s; GREEN `416eeb9` — it passes in 3.02 s; the frozen-shape row stays closed at 2 s.
- **Committed in:** `ebba922` (test) and `416eeb9` (derivation)

**2. [Rule 3 - Blocking] The executed docs demo pair now waits the 60 s empty gate and flaked**
- **Found during:** Task 2 full-package verification (the first `go test -race ./cmd/sup-kv/` run failed)
- **Issue:** `README.md`'s quickstart and `docs/EXAMPLES.md`'s pair start two empty nodes and allow 10 s for SERVING. Before this plan, the 1 s uniform gate masked a lost first empty listing; with the 60 s empty default, one node stayed closed for the rest of the block (a Get answered `Unavailable` with the gate message) after a first listing did not complete.
- **Fix:** Both executed demo blocks (README's Linux and macOS-variant quickstarts, both node starts and the restarts, and EXAMPLES' pair) pin `--startup-gate-empty 2s`, and their health-wait comments describe the two gates. A node with data still opens on the 1 s normal gate; the pin only bounds a lost-listing wait inside the block's own deadline.
- **Files modified:** `README.md`, `docs/EXAMPLES.md`
- **Verification:** `go test -race -count=2 -run '^(TestExamplesDoc|TestReadmeQuickstart)$' ./cmd/sup-kv/` green; the full cmd package green; the blocks still execute verbatim (only flags added).
- **Committed in:** `416eeb9`

**3. [Rule 3 - Blocking] Lint and unused-suppression cleanup**
- **Found during:** Task 2 lint gate
- **Issue:** The new `healthAnswer` helper returned `(status, error, bool)`, which `revive`'s error-return rule rejects; the new callers also pass `awaitServing` timeouts other than `servingDeadline`, making the existing `//nolint:unparam` directives unused.
- **Fix:** Moved the error last in `healthAnswer` and removed both `unparam` suppressions (cmd and stand).
- **Files modified:** `cmd/sup-kv/main_test.go`, `internal/stand/stand.go`
- **Verification:** `go tool golangci-lint run ./cmd/sup-kv/ ./internal/stand/` green; both packages' suites green.
- **Committed in:** `416eeb9` (signature fix) and `ebba922` (suppression removal)

---

**Total deviations:** 3 auto-fixed (1 Rule 1, 2 Rule 3)
**Impact on plan:** The test change makes the derivation proof real rather than conceptual; the docs pin restores the demo's reliability without weakening the production default; the lint cleanup is style-only. No production behavior, bound or assertion was relaxed.

## TDD Gate Compliance

Both tasks followed RED → GREEN with persisted evidence (`gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for each record):

- **Task 1:** RED `34bb8b5` — `04-04-task1.json` records both range rows failing on their planned assertions: `--startup-gate-empty=-1s` was rejected by the flag parser (the flag did not exist) instead of the positivity check, and `--startup-gate=0` reached the socket bind and exited 1 instead of 2. GREEN `26d913b` made both exit 2 with `startup gate must be positive: --startup-gate-empty=-1s` / `--startup-gate=0s`.
- **Task 2:** RED `ebba922` — `04-04-task2.json` records `TestStand_startupGateEmptyDerivedFromExtras` failing at 5 s: the lone empty node kept the 60 s default because the harness did not derive the gate; the frozen-shape sibling passed. GREEN `416eeb9` added `emptyGateArg` and the derived flag; the same row now opens at 3.02 s. The selection test was green at the RED point because the plan's own Task 1 action placed the selection in production; the derivation test carried Task 2's RED.
- **REFACTOR:** none separately — the lint cleanup is folded into `416eeb9`, and no production simplification was warranted.

## Issues Encountered

- The one-off docs flake described in deviation 2 was the only failure; it did not reproduce in five consecutive docs runs after the gate pin, and the full package has been green since.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. No placeholder values, TODOs or unwired data sources were introduced; `ListedKey` is now wired to the adapter, and the empty-gate selection is complete rather than deferred.

## Threat Flags

None — no new trust-boundary surface beyond the plan's threat model. T-4-12 (positive gate validation), T-4-13 (frozen R-03 determinism) and T-4-14 (one-shot selection) are mitigated and pinned; no new dependency was added.

## Next Phase Readiness

Ready for 04-05. `startup_gate_empty` is selectable, validated, harness-safe and proven on the stand; the frozen R-03 rows can flip through the TSV alone in 04-13, and 04-06 can complete the remaining `ServerOptions` wiring beside the `ListedKey` line. Nine plans remain in the phase.

---

*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Created files exist: this SUMMARY, `red-evidence/04-04-task1.json` and `04-04-task2.json`.
- Commits exist and are ancestors of HEAD: `34bb8b5`, `26d913b`, `ebba922`, `416eeb9`.
- The measured plan commit count is 4 from the ledger base `7ae0338`, matching the frontmatter `commits: 4`.
- The frozen `internal/stand/stand_red_test.go` is byte-untouched (`git diff --stat` empty).
