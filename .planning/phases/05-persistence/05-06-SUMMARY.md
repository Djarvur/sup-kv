---
phase: 05-persistence
plan: 06
subsystem: persistence
tags: [load-outcome, load-stats, logging, dirty-start, markdirty-nudge, skip-accounting, evidence-window, metrics, prometheus, process-tests]
requires:
  - phase: 05-persistence
    provides: the four-outcome LoadResult and preservation/sweep from 05-05
  - phase: 05-persistence
    provides: the configured-limit DecodeStats census from 05-01/05-02
  - phase: 04-protocol
    provides: the write-confirmation engine and the Phase-4 New-1 nudge finding
affects: [persistence, observability, load-outcome, dirty-lifecycle, skip-evidence, metrics-surface, p07, p08, p12]
actuals:
  tokens: 13811
  tasks: 3
  commits: 7
plan_head_before: 151211186578ba71a665a6ec1e6a66d446c11e54
plan_head_after: e3eb2bd469a93b75921775756a67025d2d304a2b
tech-stack:
  added: []
  patterns:
    - "Load report through the engine: LoadStats maps port.LoadResult once in Open and the binary reads it once"
    - "Dirty-on-outcome: the outcome identity (not the discard count) selects the post-load lifecycle"
    - "Nudge ownership: markDirty performs the non-blocking changed send, so every dirty path wakes the dumper exactly once"
    - "Candidate-keyed log attributes: failures are found/iterated by Candidate, never by slice index"
    - "Full label vocabulary at zero: both load families always render every label value"
key-files:
  created:
    - internal/stand/persist_test.go
    - internal/app/persist/load_stats_test.go
    - internal/app/persist/load_stats_internal_test.go
  modified:
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
key-decisions:
  - "LoadStats is the engine's own report type mapped from port.LoadResult in Open; the port stays the seam and Open keeps its frozen (*Engine, int) signature"
  - "Dirty exactly when the outcome is not Main: Prompt/Backup/Unusable flush before a snapshot copies the file, while a Main load — even one that discarded records — stays clean to preserve the PERSIST-10 evidence window"
  - "markDirty owns the non-blocking e.changed send; ApplyRemote's duplicate is deleted and the restore replay's mid-error path gains the wake-up for free"
  - "Backup logs one WARN whose main_path/cause/preserve_error come from the GenMain failure found by scanning Failures, so an absent main names fs.ErrNotExist instead of panicking on an empty list"
  - "Unusable logs one ERROR with one <candidate>_path/_cause/_preserve_error group per failure actually present, so one corrupt generation cannot produce a phantom .prev entry"
  - "A value-limit skip is counted by class, logged once with count and value_max_bytes, published as a metric, and left on disk until normal rotation displaces it"
  - "The two metric families render all four outcome labels and all four discard classes at zero on an idle or New-built engine"
  - "Process proofs seed dumps by running the real binary and SIGKILLing it after a confirmed write, so fixtures and loader share the production format by construction"
patterns-established:
  - "One log record per load class with static message and path/cause attributes; Prompt and Main stay silent"
  - "White-box nudge test in *_internal_test.go; lifecycle and evidence-window proofs via the public API and the real adapter"
  - "Stand scenarios assert exact log-record counts after exit and preserved .corrupt-* evidence by name prefix"
requirements-completed: [PERSIST-02, PERSIST-04, PERSIST-10]
coverage:
  - id: D1
    description: "LoadStats maps every port outcome (prompt/main/backup/unusable) with generation, per-candidate failures, decode census and sweep fields; the binary emits exactly one WARN for Backup (main path and cause, absent main included), one ERROR for Unusable with one group per real candidate failure, none for Prompt/Main, one INFO for a non-zero sweep and one WARN for a sweep error"
    requirement: "PERSIST-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_loadStatsMapsResult$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_persistLoad'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Prompt/Backup/Unusable starts are dirty and a Snapshot right after them flushes and succeeds (decoding to the loaded state, or empty for Prompt/Unusable); a Main start stays clean; every markDirty wakes an idle dumper through one nudge at Debounce and ApplyRemote carries no duplicate send"
    requirement: "PERSIST-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_(startDirty|markDirtyNudge)$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_redundantDirty$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "A record skipped by a shrunk configured limit is counted by class, logged once with the count and the limit, published as a non-zero value_limit metric, and stays in the main dump through zero writes, in .prev after one write and gone only after two dumps"
    requirement: "PERSIST-10"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_skipEvidenceWindow$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_persistShrunkLimit$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "supkv_dump_load_total{outcome} and supkv_dump_records_discarded_total{class} render every label at zero and the loaded outcome/counts after a load, on both the exporter unit surface and the running binary's scrape"
    requirement: "PERSIST-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/metrics/ -run '^TestMetrics_'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMetrics_'"
        status: pass
    human_judgment: false
duration: 15min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 06: Load Visibility, Dirty Start and Skip Accounting Summary

**Every load outcome is now reported once with paths and causes, a non-Main start is dirty so a snapshot can never copy a corrupt main, every dirty path nudges the dumper, and value-limit skips are counted, logged, metricized and left on disk through normal rotation.**

## Performance

- **Duration:** 15min
- **Started:** 2026-11-22T14:48:10Z
- **Completed:** 2026-11-22T15:02:51Z
- **Tasks:** 3
- **Files modified:** 9 (3 created, 6 modified)

## Accomplishments

- `persist.LoadStats` is the engine's own report over the port result: `Loaded`, `Outcome`, `Generation`, per-candidate `Failures`, the accepted generation's `DecodeStats`, and `Swept`/`SweepErr`. `Open` maps it without changing its frozen `(*Engine, int)` signature; `New`-built engines report the zero value and all four outcome metric samples at 0.
- The binary reports one load class per line: a Backup WARN names the main path, the cause (including `fs.ErrNotExist` for an absent main) and the preservation error, found by scanning `Failures` for `GenMain`; an Unusable ERROR carries one `main_*`/`prev_*` attribute group per candidate that actually failed — one corrupt generation yields one group and no phantom entry; Prompt and Main stay silent. A non-zero sweep logs one INFO with the count, a sweep error one WARN.
- The post-load lifecycle is outcome-driven: Prompt, Backup and Unusable start dirty, so `Snapshot` right after them flushes first and succeeds — the Prompt/Unusable snapshots decode to the empty state, the Backup snapshot to the loaded records — while a Main load, even one that discarded records, stays clean and lets `Snapshot` copy the accepted on-disk generation without a rewrite.
- `markDirty` now owns the non-blocking `changed` send: `ApplyRemote`'s duplicate is gone, each write path nudges exactly once, and the restore replay's mid-error path gains the wake-up (the Phase-4 New-1 fix). An idle `markDirty` produces exactly one dump at `Debounce`, never earlier, and the clean engine stays quiet.
- A shrunk-limit skip is a named class: the codec counts it, the binary logs one WARN with `count` and `value_max_bytes`, the exporter publishes it, and the file keeps the evidence — main through zero writes, `.prev` after one dump, gone after two.
- Two metric families joined the surface: `supkv_dump_load_total{outcome}` and `supkv_dump_records_discarded_total{class}`, both rendering every label value at zero and the loaded values after startup. `Sources.LoadStats` is wired to `engine.LoadStats` in the binary.
- Process-level proofs run the real binary over seeded dump directories: corrupt main + good `.prev`, absent main + good `.prev`, two corrupt generations, one corrupt generation, a fresh start with an orphan temp, and the shrunk-limit restart all assert exact log-record counts, served/empty stores, preserved `.corrupt-*` evidence and the metrics scrape.

## Task Commits

1. **Task 1 RED: process-level load reporting** - `6e0e234` (test)
2. **Task 1 GREEN: LoadStats and the one-WARN/one-ERROR policy** - `881d681` (feat)
3. **Task 2 RED: dirty start, nudge and skip evidence** - `06ff294` (test)
4. **Task 2 GREEN: dirty-off-Main, markDirty nudge, skip WARN** - `83b14fe` (feat)
5. **Task 3 RED: load and discard metric families** - `7b91314` (test)
6. **Task 3 GREEN: publish the two families and wire the source** - `5c4f800` (feat)
7. **Sweep-log process proof** - `e3eb2bd` (test)

**Plan metadata:** the `docs(05-06)` completion commit (this commit).

## Files Created/Modified

- `internal/app/persist/engine.go` — `LoadStats`/`LoadStats()`, the `Open` mapping, `startLoadDirty`, and the nudge in `markDirty` (with `ApplyRemote`'s duplicate removed).
- `internal/app/persist/engine_test.go` — `fakeDumpPort.seedSave` for clean-start fixtures.
- `internal/app/persist/load_stats_test.go` — report mapping, dirty-start/snapshot and skip evidence-window tests.
- `internal/app/persist/load_stats_internal_test.go` — the white-box `markDirty` nudge test (unexported method).
- `cmd/sup-kv/main.go` — `logLoadStats`/`logBackupFallback`/`logUnusableGenerations`, candidate lookup, the skip WARN with the remaining-class INFO, and the `LoadStats` metrics source.
- `cmd/sup-kv/main_test.go` — the binary scrape lists both new families.
- `internal/metrics/metrics.go` — `Sources.LoadStats`, `loadFamilies`, `discardClassFamilies`, and the `labelOutcome` constant.
- `internal/metrics/metrics_test.go` — family table, zero-state samples, zero source and by-effect assertions for the two families.
- `internal/stand/persist_test.go` — six process scenarios: backup, absent-main backup, unusable, single-candidate unusable, prompt (with sweep INFO), and shrunk-limit.

## Decisions Made

- **Report in the engine, seam in the port.** `LoadStats` maps `port.LoadResult` once in `Open`; no second port contract, no signature change.
- **Outcome identity decides dirtiness.** `outcome != LoadMain` is the whole rule; discard counts never dirty a Main load.
- **Attribute lookup by candidate.** Backup scans for `GenMain`; Unusable iterates `Failures`. Zero, one or two entries are all handled without indexing.
- **One skip line, one class.** Only `value_limit` gains a WARN; the other decode classes keep the existing INFO and the total is split, never double-reported.
- **Seed through the binary.** Stand fixtures are produced by the real node and SIGKILLed after `Set` returned, so dump bytes, checksums and the loader agree by construction.
- **Metrics read the source once per scrape.** `Snapshot` takes one `LoadStats` and feeds both load families.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Lint/testpackage] The nudge test needs a white-box file**
- **Found during:** Task 2
- **Issue:** `markDirty` is unexported while every `internal/app/persist` test file is `persist_test`; the strict `testpackage` gate exempts only `*_internal_test.go`.
- **Fix:** Created `load_stats_internal_test.go` with its own in-memory `nudgeDumpPort` and the `synctest` nudge proof; the other three tests stay in the planned `load_stats_test.go`.
- **Files modified:** `internal/app/persist/load_stats_internal_test.go`
- **Verification:** `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_markDirtyNudge$'` green; lint clean.
- **Commit:** `06ff294` (RED), `83b14fe` (GREEN)

**2. [Rule 1 - Preserved-name prefixes] The corrupt-file counter matched only the main prefix**
- **Found during:** Task 1 GREEN
- **Issue:** The `.prev` candidate is preserved as `<dump>.prev.corrupt-*`, so a counter matching only `<dump>.corrupt-*` saw one file where the both-corrupt scenario correctly produces two.
- **Fix:** The stand helper counts both candidate prefixes; the loader behavior was already correct.
- **Files modified:** `internal/stand/persist_test.go`
- **Verification:** `TestStand_persistLoadUnusable` green with two preserved files.
- **Commit:** `881d681`

**3. [Rule 3 - Compile-coupled RED] Declaration stubs and the accessor wiring landed in the RED commit**
- **Found during:** Task 3
- **Issue:** The new tests reference `metrics.Sources.LoadStats` and the binary must not panic while the families are absent; a nil source would have turned the RED verdict into an HTTP EOF instead of an assertion.
- **Fix:** The RED commit landed the `Sources.LoadStats` field, family functions returning nil, and the `LoadStats: engine.LoadStats` wiring. The GREEN commit replaced the stubs; the RED failures were the missing family assertions.
- **Files modified:** `internal/metrics/metrics.go`, `cmd/sup-kv/main.go`
- **Verification:** RED: `TestMetrics_families`, `TestMetrics_values`, `TestStand_persistShrunkLimit` and `TestMetrics_optIn` failed with "carries no sample"/"lacks the family"; GREEN: all pass.
- **Commit:** `7b91314` (RED), `5c4f800` (GREEN)

**4. [Rule 3 - Lint/toolchain] Strict-gate shaping of the new tests and one constant**
- **Found during:** Task 3 lint
- **Issue:** The vendored strict config flagged `noinlineerr` (two inline error checks), `paralleltest`/`tparallel` on the synctest subtest loop, `unparam` on four test helpers with constant arguments, `err113` on a dynamic test error, `goconst` on the repeated `"outcome"` label and `mnd` on the attribute-capacity factor; the standalone stand file also needed a blank line before its package clause (`godoclint`).
- **Fix:** Split the error checks, added `t.Parallel()` to the synctest subtests, trimmed the constant helper parameters, used `fs.ErrPermission` as the fixture error, added `labelOutcome` and `loadFailureAttrsPerCandidate`, and separated the file comment from the package clause.
- **Files modified:** `internal/app/persist/load_stats_test.go`, `internal/app/persist/load_stats_internal_test.go`, `internal/stand/persist_test.go`, `internal/metrics/metrics.go`, `cmd/sup-kv/main.go`
- **Verification:** `make lint` clean.
- **Commit:** `5c4f800` (and `e3eb2bd` for the sweep-test check)

**5. [Rule 1 - Stale commit ledger] The plan's commit ledger pointed at a 2025 base**
- **Found during:** Before the first task commit
- **Issue:** `.git/gsd-plan-head-before-05-06` existed from a prior milestone and would have measured months of history.
- **Fix:** Reset it to this plan's base (`1512111`), the same bookkeeping fix 05-04 recorded.
- **Files modified:** `.git/gsd-plan-head-before-05-06` (local git state, not tracked)
- **Verification:** `git rev-list --count 1512111..e3eb2bd` = 7, matching the task commits.
- **Commit:** n/a (pre-commit bookkeeping)

### Additions Beyond the Plan's File List

- **Sweep-log process proof.** The must-have truth requires the INFO for a non-zero sweep; `TestStand_persistLoadPrompt` now seeds an hour-old `.tmp~` orphan, asserts the one INFO with `count:1` and the file's removal, and keeps the load classes silent.

## Issues Encountered

- **Full-suite flake, not this plan's code.** One `make test` run failed `TestLog_formatText` with "timed out waiting for the DuplicateNodeID event" while the 174 s `internal/app/repl` race suite ran in parallel; the test passes alone (0.11 s) and the whole `cmd/sup-kv` package gate passed in this plan. Recorded in `deferred-items.md`.
- **Plan verification grep vs prior-phase ripple.** The literal `git diff <CONTEXT-commit> -- internal/grpcapi/` check reports prior plans' `server_test.go` interface adaptation; this plan's own diff (`1512111..e3eb2bd`) contains no `internal/grpcapi` file and adds no health transition or RPC change.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The load report, the dirty lifecycle, the nudge and the skip accounting are complete and observable; plans 05-07/05-08 can extend the same metrics family table with `supkv_dumps_failures_total` and `supkv_dumps_unconfirmed_total`.
- The stand package now has the process-level load fixtures (backup/unusable/prompt/shrunk) later documentation plans can cite.
- Ready for 05-07.

## Self-Check: PASSED
