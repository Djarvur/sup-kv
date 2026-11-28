---
phase: 07-observability-and-configuration
plan: 03
subsystem: lint-gate-architecture-boundary
tags:
  - obsv-07
  - depguard
  - architecture-boundary
  - additive-lint-config
  - probe-verified
  - allowed-exception
requires:
  - phase: 07-observability-and-configuration
    provides: 07-01's SentByType relocation, so internal/metrics imports no socket adapter and the exporter deny rule is satisfiable
  - phase: 01-ci-bootstrap-and-repo-skeleton
    provides: the vendored powerman/golangci-lint-strict 2.15.0 body, its additive-keys project block and the `make lint` gate that runs `config verify` first
provides:
  - sup-kv-adapters-not-in-core, sup-kv-metrics-no-adapters and sup-kv-grpcapi-no-repl-metrics as additive depguard rules below the frozen body
  - the written justification for the one allowed grpcapi -> persistfile / app-persist composition edge
  - per-rule probe evidence that each new boundary fires on a forbidden import and leaves no file
affects:
  - 07-04 through 07-11 (a new cross-layer import is now a visible .golangci.yml diff or a code move, never a quiet workaround)
  - internal/config (composition level by design: no rule covers it)
actuals:
  tokens: 972
  tasks: 2
  commits: 2
  plan_head_before: b15395fe67c8964a04abd9f0ffc921ed8cc16dc4
  plan_head_after: b35e6067a33a91e80820cdc2cda33a4bae717c90
tech-stack:
  added: []
  patterns:
    - "Additive lint config over a vendored body: new depguard rules are inserted as whole key blocks at the end of the project rules, so the upstream lines stay byte-identical and the diff is insertions only."
    - "Probe-proven rule: a temporary non-test file with a real forbidden import must make the focused lint name the rule before the rule is called enforced; the probe is deleted in the same command."
    - "Allowed-exception-as-config: a legitimate cross-layer composition edge is written in the YAML beside the rule that permits it instead of being hidden behind a //nolint."
key-files:
  created: []
  modified:
    - .golangci.yml
key-decisions:
  - "Three rules, not one: core/ports, exporter and API adapter each get their own rule name, so a violation names the layer that broke the direction."
  - "Adapters are named individually and !$test is on every rule: the core packages import each other, internal/port imports internal/app/storage and internal/app/version, and internal/testutil is a shared helper — a wholesale deny of github.com/Djarvur/sup-kv/internal would break valid builds (T-07-10)."
  - "The grpcapi -> persistfile / app/persist edge is allowed with a written justification in the YAML: D-07 permits 'removed or explicitly allowed', and replacing *persist.Engine with a local interface is a larger refactor with no Phase-7 requirement."
  - "Probe shape is a used import (var _ = udprepl.SentByType{}), not a blank import: golangci-lint's uniq-by-line dedup drops the depguard finding when revive's blank-imports reports at the same position, so a blank-import probe cannot prove the rule under the full config."
patterns-established:
  - "Pattern: prove a depguard rule by name, then prove the tree still passes the whole gate — rule presence alone is not enforcement."
requirements-completed: []
coverage:
  - id: D1
    description: "Three additive depguard rules sit below the frozen vendored body with !$test and per-adapter denies, plus the written grpcapi composition exception"
    requirement: OBSV-07
    verification:
      - kind: contract
        ref: "go tool golangci-lint config verify -> clean (schema accepts the new keys)"
        status: pass
      - kind: contract
        ref: "grep -n 'sup-kv-adapters-not-in-core\\|sup-kv-metrics-no-adapters\\|sup-kv-grpcapi-no-repl-metrics\\|Allowed exception (Q-05)' .golangci.yml -> lines 105, 125, 138, 147"
        status: pass
      - kind: contract
        ref: "git diff b15395f..HEAD --stat -> .golangci.yml 1 file, 58 insertions(+), 0 deletions"
        status: pass
    human_judgment: false
  - id: D2
    description: "Each new rule demonstrably fires on a temporary forbidden import, and no probe file survives"
    requirement: OBSV-07
    verification:
      - kind: contract
        ref: "probe sweep: internal/{app,metrics,grpcapi}/probe_boundary.go each importing internal/udprepl -> focused lint names sup-kv-adapters-not-in-core / sup-kv-metrics-no-adapters / sup-kv-grpcapi-no-repl-metrics with exit 1; find internal -name probe_boundary.go -> empty"
        status: pass
      - kind: contract
        ref: "go tool golangci-lint run --default=none --enable=depguard on the plan's literal blank-import probe -> names sup-kv-adapters-not-in-core (the full-config run masks it via uniq-by-line; see Deviations)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The vendored body is byte-identical and the full lint gate is green on the committed tree, including the allowed grpcapi imports"
    requirement: OBSV-07
    verification:
      - kind: contract
        ref: "sed -n '1,64p' .golangci.yml | git hash-object --stdin == a121e0466bc211e48ca898f6c07434dc50eba07c"
        status: pass
      - kind: contract
        ref: "make lint (config verify + run ./...) on the committed tree -> green, proving grpcapi's persistfile/app-persist imports and every existing file still pass"
        status: pass
    human_judgment: false
duration: 4min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 03: Depguard Architecture Boundary Summary

**Three additive depguard rules now machine-enforce the cmd → adapters → core/ports direction — `sup-kv-adapters-not-in-core`, `sup-kv-metrics-no-adapters` and `sup-kv-grpcapi-no-repl-metrics`, each probe-proven to fire on a real forbidden import, with the one legitimate `grpcapi` → `persistfile`/`app-persist` composition edge written down in the config and the vendored 2.15.0 body byte-identical.**

## Performance

- **Duration:** 4min
- **Started:** 2026-11-23T07:45:15Z
- **Completed:** 2026-11-23T07:49:36Z
- **Tasks:** 2
- **Files modified:** 1 (`.golangci.yml`, +58 lines, 0 deletions)

## Accomplishments

- The architecture boundary is now a lint gate, not a review convention (OBSV-07, D-07): `internal/app/**` and `internal/port/**` may not import `udprepl`, `persistfile`, `grpcapi`, `metrics` or `stand`; `internal/metrics` may not import `udprepl`, `persistfile` or `grpcapi`; `internal/grpcapi` may not import `udprepl` or `metrics`.
- Every rule carries `!$test` and names adapters individually, so the legitimate same-layer imports (`app` packages importing each other, `port` → `app/storage`/`app/version`, `internal/testutil` as a shared helper) stay legal — `make lint` on the whole tree proves no existing file breaks.
- The single allowed exception is documented where the rule lives: `grpcapi` → `internal/persistfile` for `ErrInvalidSnapshotName`/`ErrNoDump` and `grpcapi` → `internal/app/persist` for the `*persist.Engine` facade (12 methods plus the Watch hub's `Records`/`Set`), with the reason a local interface is out of Phase-7 scope.
- Each rule was probe-proven: a temporary non-test file importing a forbidden package made the focused lint name that rule and exit non-zero; all probes were deleted and `find internal -name probe_boundary.go` is empty.
- The vendored powerman/golangci-lint-strict 2.15.0 body (lines 1-64) still hashes to `a121e0466bc211e48ca898f6c07434dc50eba07c`; `golangci-lint config verify` and `make lint` are green on the committed tree.

## Task Commits

1. **Task 1: Add the adapter-direction rule and prove it fires with a temporary probe** - `9776071` (chore)
2. **Task 2: Add the exporter and API-adapter rules with the written exception** - `b35e606` (chore)

**Plan metadata:** `docs(07-03): complete depguard boundary plan` (this commit)

## Files Created/Modified

- `.golangci.yml` - three additive `depguard.rules` entries after the existing project rules (`sup-kv-adapters-not-in-core` at line 105, `sup-kv-metrics-no-adapters` at 125, `sup-kv-grpcapi-no-repl-metrics` at 138) plus the allowed-exception comment block at 147; the vendored body and the existing `sup-kv-*` rules are untouched.

## Decisions Made

- **Three named rules instead of one.** A violation now names the layer that broke the direction (core/ports, exporter or API adapter), which is what an operator or reviewer needs from the gate.
- **Individual adapter names, never `github.com/Djarvur/sup-kv/internal`.** The research import audit shows a wholesale deny would break `port` → `app/storage`/`app/version` and the `internal/app/internal/testutil` → `internal/testutil` helper import; T-07-10's over-broad-rule risk is avoided by construction.
- **Exception in the config, not a `//nolint`.** D-07 allows "removed or explicitly allowed"; the `grpcapi` composition edge is explicitly allowed with its justification, and the prohibition on `//nolint` workarounds holds.
- **Used-import probe shape.** `var _ = udprepl.SentByType{}` beside a real import leaves the import line free of a `revive` diagnostic, so the full-config lint can report the depguard finding; a blank-import probe is deduped away (see Deviations).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The plan's probe command cannot surface the depguard finding under the full config**
- **Found during:** Task 1 (and again in Task 2's two probes)
- **Issue:** The plan's probe writes `import _ "github.com/Djarvur/sup-kv/internal/udprepl"`. `revive:blank-imports` reports at the same `file:3:8` position as `depguard`, and golangci-lint's `uniq-by-line` processor keeps one issue per line, so the depguard finding is dropped. The literal probe therefore exits non-zero but never names the rule, even though the rule fires. Phase 6's review (`06-REVIEW.md`, WR-03) recorded the identical masking on its `api/supkvapi/v1` probe.
- **Fix:** Kept the rule as planned and probed with a used import — `import "…/internal/udprepl"` plus `var _ = udprepl.SentByType{}` — which leaves no `revive` issue on the import line; the full-config lint then reports `sup-kv-<rule>` and exits 1. As a second, independent proof, the plan's exact blank-import probe was run with `go tool golangci-lint run --default=none --enable=depguard ./internal/app/`, which names `sup-kv-adapters-not-in-core`. All three rules were proven this way and every probe file was deleted.
- **Files modified:** none beyond `.golangci.yml` (probe files are temporary and removed in the same command)
- **Verification:** probe sweep `3/3` rules fire; `find internal -name probe_boundary.go` empty; `make lint` green; hash pin holds
- **Commit:** `9776071` (rule), `b35e606` (Task 2 rules); the evidence is the probe output quoted in D2 above

## Issues Encountered

- The pre-existing plan ledger `gsd-plan-head-before-07-03` pointed at `629fff4`, which is not an ancestor of this branch's HEAD; it was reset to the plan-start HEAD `b15395f` before measuring, following the 07-01 precedent.
- `requirements.ready-ids` reports `OBSV-07` blocked: the id is shared across plans 07-01/02/03/10/11 (#2388 shared-ID gate), so no checkbox was marked and `requirements-completed` stays empty; 07-11 closes the phase-level checkboxes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The dependency direction is machine-enforced, so any later plan that reaches for a cross-layer import gets a named lint failure instead of silent erosion; the exporter rule also locks in 07-01's `SentByType` relocation. `internal/config` stays composition-level and uncovered by design. No blockers; 8 plans remain in Phase 07 — ready for 07-04.

## Self-Check: PASSED

- Modified file exists: `.golangci.yml` — FOUND (three new rule keys at lines 105/125/138, exception comment at 147).
- Vendored body hash: `a121e0466bc211e48ca898f6c07434dc50eba07c` — MATCH.
- Commits exist: `9776071`, `b35e606` both ancestors of HEAD.
- Probe files: none remaining under `internal/`.
