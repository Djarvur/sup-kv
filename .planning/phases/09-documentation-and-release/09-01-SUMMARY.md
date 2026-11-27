---
phase: 09-documentation-and-release
plan: 01
subsystem: documentation
tags: [docs, configuration-reference, doc-anchor-tests, doc-is-test-input, m-07, tdd]

requires:
  - phase: 08-performance
    provides: all fixes in and the PERFORMANCE.md hardware-block ledger the fast-profile relation cites
  - phase: 07-observability-and-configuration
    provides: internal/config Parse/Validate/Usage — the registered flag surface, the 12-rule validate.go catalog and the --help rendering the doc and iterator mirror
provides:
  - docs/CONFIGURATION.md — the as-built configuration reference: 23-flag table in --help order with byte-copied defaults and env twins, the five env-only knobs, the 12-rule validation-relations catalog with exit-2 behavior, the must-match-on-all-nodes list
  - cmd/sup-kv/configuration_test.go — the two-directional registered-flags iterator (TestConfigurationDoc) and the relations anchors (TestConfigurationDocRelations): a flag/default/registration change in internal/config without a doc edit fails make test, and a documented-but-unregistered flag is fatal
  - the phase tracer slice for doc-is-test-input — the shape every later 09 plan reuses
  - M-07's doc half: every registered flag named with its default, including the twelve flags the v1.0 audit found named nowhere
affects: [09-05 (OPERATIONS.md cross-link half of M-07), 09-02..09-11 doc plans reusing the tracer shape, DOCS-09 default-change-breaks-CI coupling]

actuals:
  tokens: 7250
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "registered-flags iterator: parse a clean-env --help capture of the built smoke binary and check the doc in both directions, with vacuity guards (>=22 flags, >=12 catalog rules, >=8 relation rows) so a broken parse can never pass silently"
    - "dynamic catalog anchoring: the relations test parses validate.go's rules() source, so an added validation rule without a doc row fails without a test edit"

key-files:
  created:
    - docs/CONFIGURATION.md
    - cmd/sup-kv/configuration_test.go
    - .planning/phases/09-documentation-and-release/red-evidence/09-01-task1-red.json
  modified: []

key-decisions:
  - "the --help capture is authoritative over the research prose — the registered surface is 23 flags (research section 4 rounded to 22); the >=22 vacuity guard holds and M-07's doc half covers all 23"
  - "dump_debounce <= dump_max_wait is documented as the dump cycle's intended relation the validator does NOT enforce — validate.go's catalog has no such rule, so the doc states the honest semantics instead of inventing a startup rule"
  - "M-07's doc half is closed here; the OPERATIONS.md cross-link half rides plan 09-05, which owns OPERATIONS.md — the two-half disposition is recorded in this summary"

patterns-established:
  - "doc-is-test-input tracer: author the doc from a machine source, verify by test, fail loudly on drift in either direction — reused by the later documentation plans"
  - "honest no-default rule: a flag whose --help prints no default may not carry a digit in the doc's Default cell — hand-typed defaults are structurally impossible"

requirements-completed: [DOCS-01, DOCS-02]

coverage:
  - id: D1
    description: "docs/CONFIGURATION.md flag table and Environment-only section: every registered flag in --help order with the printed default and env twin, --metrics-addr's no-default listener-off statement, --version's not-a-runtime-knob statement, the five env-only knobs, the flag > env > default precedence"
    requirement: DOCS-02
    verification:
      - kind: unit
        ref: "cmd/sup-kv/configuration_test.go#TestConfigurationDoc"
        status: pass
      - kind: other
        ref: "grep -cE '^\\| --' docs/CONFIGURATION.md >= 22 (23 rows) + 'no default' + SUPKV_METRICS_CLIENT_CA present"
        status: pass
    human_judgment: false
  - id: D2
    description: "## Validation relations (one row per validate.go catalog rule with exit-2 behavior and the operator-facing relation literals) and ## Must match on all nodes (node-id uniqueness, value-max-bytes, max-clock-lead, sync-chunk-bytes with disagreement effects, cross-linked to OPERATIONS)"
    requirement: DOCS-02
    verification:
      - kind: unit
        ref: "cmd/sup-kv/configuration_test.go#TestConfigurationDocRelations"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ (full package suite, siblings green)"
        status: pass
    human_judgment: false
  - id: D3
    description: "the two-directional registered-flags iterator with vacuity guards — the DOCS-09 default-change-breaks-CI coupling: a default/flag change in internal/config without a doc edit fails make test"
    requirement: DOCS-02
    verification:
      - kind: unit
        ref: "cmd/sup-kv/configuration_test.go#TestConfigurationDoc"
        status: pass
      - kind: other
        ref: ".planning/phases/09-documentation-and-release/red-evidence/09-01-task1-red.json (gsd_run check tdd-red-evidence verdict RED_EVIDENCE_OK — the tests failed on the missing doc before it existed)"
        status: pass
    human_judgment: false
  - id: D4
    description: "M-07's doc half: all 23 registered flags documented with defaults, including the twelve flags the v1.0 audit found named nowhere (startup-gate, wait-timeout, sync-chunk-bytes, hello-interval, peer-timeout, ack-timeout, ack-timeout-max, retransmit-interval, sync-interval, sync-response-rate, dump-debounce, dump-max-wait); the OPERATIONS link half rides 09-05"
    requirement: DOCS-01
    verification:
      - kind: unit
        ref: "cmd/sup-kv/configuration_test.go#TestConfigurationDoc (requires a default-carrying row for every registered flag)"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-11-27
status: complete
---

# Phase 9 Plan 1: Configuration Reference Summary

**docs/CONFIGURATION.md authored from the binary's own --help and made un-driftable in both directions by the TestConfigurationDoc/TestConfigurationDocRelations iterator — the phase's doc-is-test-input tracer and M-07's doc half.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-11-27T21:23:53Z
- **Completed:** 2026-11-27T21:33:55Z
- **Tasks:** 3
- **Files modified:** 3 (2 production artifacts + 1 RED evidence record)

## Accomplishments

- The registered-flags iterator (`TestConfigurationDoc`) was written FIRST and recorded RED: it parses a clean-environment `--help` capture of the built smoke binary (merged stdout+stderr, exit 0 — `config.Usage` resolves printed defaults from the environment, so env twins must not skew them), asserts >=22 flags parsed and the five env-only knobs in the footer block, and only then reads the doc — the RED state (doc absent) is the recorded evidence, classifier verdict RED_EVIDENCE_OK.
- `docs/CONFIGURATION.md` ships complete: 23 registered flags in `--help` registration order with byte-copied defaults and env twins, `--metrics-addr` stating the no-default listener-off behavior, `--version` stating it prints the build identity (not a runtime knob), the five env-only knobs under the help footer's own title, and the flag > env > default precedence with its OPERATIONS pointer.
- `## Validation relations` mirrors internal/config/validate.go's actual 12-rule catalog (required-knobs through address-shape) with the exit-2 behavior and the operator-facing literals (`peer_timeout > hello_interval`, `ack_timeout ≤ ack_timeout_max`, `drift_warn < max_clock_lead`, the 65 507-byte datagram ceiling with the 1 024-byte key limit, the positivity sentinels); `## Must match on all nodes` lists node-id uniqueness, value-max-bytes, max-clock-lead and sync-chunk-bytes with the effect of a disagreement, cross-linked to OPERATIONS rather than restated divergently.
- Both directions are machine-checked: every registered flag needs a row whose cells carry the printed default and env twin (a no-printed-default row may not carry a digit — hand-typed defaults are structurally impossible), every `--flag` token the doc names must be registered, and the rows must preserve the help order row-by-row. TestConfigurationDocRelations parses validate.go's rules() source dynamically, so a validation rule added without a doc row fails without a test edit.
- The full cmd/sup-kv suite is green under -race (no sibling doc-anchor test regressed); the vacuity guards (>=22 flags, >=12 catalog rules, >=8 relation rows) keep a broken parse from passing silently.
- M-07's doc half is closed: every registered flag appears with its default, including the twelve flags the v1.0 audit found named nowhere. The OPERATIONS-side link half rides plan 09-05 (which owns OPERATIONS.md).

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the registered-flags iterator, written before the doc exists** — `6256cd9` (test)
2. **Task 2: Author docs/CONFIGURATION.md from the binary — flag table and env-only section** — `97f2787` (feat)
3. **Task 3: The relations catalog and the must-match list** — `f502b0c` (feat)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `docs/CONFIGURATION.md` — the as-built configuration reference: scope statement naming the iterator, flag table (23 rows, help order, byte-copied defaults), Environment-only section, Validation relations (12 catalog rules + the two Parse-side facts + the unenforced dump relation), Must match on all nodes.
- `cmd/sup-kv/configuration_test.go` — the honest-scope iterator: `configurationDocPath`, vacuity-guard constants, anchor tables, `runHelpCleanEnv`/`parseHelpFlags`/`parseDocFlagRows`/`docRowCell`/`docSection`, `TestConfigurationDoc`, `TestConfigurationDocRelations`.
- `.planning/phases/09-documentation-and-release/red-evidence/09-01-task1-red.json` — the RED evidence record (classifier RED_EVIDENCE_OK).

## Decisions Made

- The live `--help` capture is authoritative over the research prose: the registered surface is 23 flags, not the inventory's rounded 22. The plan's own assumption mandates this; the >=22 vacuity guard holds and M-07's doc half covers all 23.
- `dump_debounce ≤ dump_max_wait` is documented as the dump cycle's intended relation that the validator does not enforce — validate.go's catalog contains no such rule, so the doc states the honest semantics (a smaller max-wait truncates every quiet window at the deadline) instead of inventing a startup rule.
- M-07's closure is a recorded two-half disposition: the doc half landed here (machine-checked), the OPERATIONS.md `## Configuration validation` cross-link lands with plan 09-05.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The plan's "positional args ignored" gloss contradicted the code**
- **Found during:** Task 3 (relations catalog)
- **Issue:** the plan's task text listed "positional args ignored" as a relations item, but `internal/config.Parse` rejects positional arguments by name (`errPositionalArguments`, exit 2) — documenting "ignored" would be a false claim.
- **Fix:** the doc states the as-built behavior (positional arguments rejected by name, exit code 2) with the catalog file as the source, per the plan's own "the catalog file is the source" instruction.
- **Files modified:** docs/CONFIGURATION.md
- **Verification:** TestConfigurationDocRelations anchors `positional arguments` and passes.
- **Committed in:** f502b0c (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 code-vs-plan-gloss correction). **Impact on plan:** none — the doc documents what the binary does, which is the plan's own grounding rule.

## Issues Encountered

- The first RED run failed for a test-side reason instead of the planned one: the validate.go catalog regexp lacked the `(?m)` multiline flag and parsed 0 rules, so the vacuity guard fired before the missing-doc complaint. The regexp was fixed and the RED evidence re-captured; the recorded run fails exactly on the missing doc with all pre-doc guards passing (this is the #3770 invalid-RED discipline applied within Task 1, before any GREEN).

## TDD Gate Compliance

- Task 1 (`tdd="true"`) has its RED commit (`6256cd9`, `test(09-01)`) before the implementation commits (`97f2787`, `f502b0c`, `feat(09-01)`, both descending from the RED); the RED evidence record passed `gsd_run check tdd-red-evidence` with verdict RED_EVIDENCE_OK. No REFACTOR commit was needed — the GREEN authoring needed no cleanup pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for 09-02 — the tracer slice is production quality and every later plan reuses its shape (author from a machine source, verify by test, fail loudly on drift in both directions).
- No blockers. The OPERATIONS.md cross-link half of M-07 is owned by 09-05; the doc-side anchor (`cmd/sup-kv/configuration_test.go` name, `flag > env > default`, the relations/must-match headings) is already in place for it.

---
*Phase: 09-documentation-and-release*
*Completed: 2026-11-27*
