---
phase: 07-observability-and-configuration
plan: 02
subsystem: configuration
tags:
  - obsv-02
  - obsv-08
  - config-extraction
  - validation-rule-table
  - codec-value-guard
  - drift-window
  - exit-code-2
requires:
  - phase: 03-transport
    provides: the udprepl datagram-ceiling oracle (MaxDatagramBytes, MaxRecordDatagramBytes) and the WR-06 sync-chunk ceiling
  - phase: 04-protocol
    provides: the startup-gate positivity contract and the MaxClockLead inbound version bound
  - phase: 05-persistence
    provides: the routed IN-04 library-level value-limit guard site (persist.EncodeTo) and the dump format's maxDumpValueBytes
  - phase: 07-observability-and-configuration
    provides: plan 07-01's split adapter and the SentByType relocation that keep the new package's imports one-directional
provides:
  - internal/config with Config, Parse, Validate, Usage and ErrHelp as the only configuration entry points
  - the 12-rule validation catalog (ranges and relations) with a meta-test pinning length, names, order and owning subtest
  - a wiring-only cmd/sup-kv/main.go (no flag registration, no env-name constant, no check* function)
  - persist.EncodeTo's out-of-domain value-limit guard, wrapping storage.ErrValueTooLarge before any write
  - repl.MaxSyncResponseRate exported and the 2 m max-clock-lead default consistent across binary, core, help and docs
affects:
  - 07-04 (the injectable run orchestration consumes config.Parse/config.Validate)
  - 07-03 (the metrics depguard rule set may now treat internal/config as an adapter-side package)
  - 07-05 (the version property's edge window and near-version generator must straddle the 2 m bound)
  - 07-06..07-11 (every later plan validates through the one rule table)
actuals:
  tokens: 26254
  tasks: 3
  commits: 5
  plan_head_before: 7325bf577b82447ce9170b2fa77a7394ecb62cc0
  plan_head_after: 7b6efc8562396f5a5af78797cc045a58deb9d1ba
tech-stack:
  added: []
  patterns:
    - "One validation catalog: a `rules()` function returns an immutable `{name, check}` slice; Validate walks it and errors.Join's every failure under its rule name, so an operator sees all bad knobs in one pass."
    - "Local FlagSet per call: Parse builds a fresh flag.NewFlagSet with io.Discard, resolves env twins through a `getenv func(string) string` seam, and keeps no package state — parallel tests and callers cannot observe each other."
    - "Usage renders the parser's own registration (with the process environment as the default source), so the flag table, the `(env SUPKV_X)` twins and the env-derived `(default ...)` values cannot drift from Parse."
    - "Library guard before the first write: an out-of-domain limit returns a wrapped sentinel with an empty buffer, so no caller can emit a partial generation."
key-files:
  created:
    - internal/config/config.go
    - internal/config/usage.go
    - internal/config/validate.go
    - internal/config/config_test.go
    - internal/config/usage_test.go
    - internal/config/validate_test.go
  modified:
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - internal/app/repl/core.go
    - internal/app/repl/discard_test.go
    - internal/app/persist/codec.go
    - internal/app/persist/codec_test.go
    - internal/app/persist/doc.go
    - internal/app/version/version.go
key-decisions:
  - "internal/config stays outside internal/app: it reads the environment and imports the adapters whose ceilings it validates, so the no-I/O depguard rule must not cover it (plan prohibition, AGENTS.md deviation 1)."
  - "Validate joins every rule failure, wrapping each with its rule name; the sentinel texts the process tests assert (`datagram ceiling`, `value-max-bytes must be positive`, `startup gate must be positive`, `sync-chunk-bytes exceeds the datagram ceiling`) are preserved as substrings."
  - "The max-clock-lead default moved to 2 m in the same commit as the drift-vs-lead rule: leaving the 1 m default would make the shipped defaults fail their own validation, violating the plan's 'every default stays accepted' prohibition."
  - "Usage resolves defaults from the process environment, preserving the pre-existing contract that SUPKV_MAX_CLOCK_LEAD=90s --help shows (default 1m30s) while Parse itself stays pure over its getenv seam."
  - "The validation test is an internal test file (package config) with a justified //nolint:testpackage so the meta-test can iterate the unexported rules() the plan specifies."
  - "The codec guard wraps storage.ErrValueTooLarge (not a new sentinel) so callers keep one size-failure identity across the store, the encoder and the config validator."
patterns-established:
  - "Pattern: a startup check is a named rule in one table, and the meta-test pins the table's length, order and the subtest that owns each rule — a rule cannot be added untested."
  - "Pattern: config validation reads the adapter ceiling constants (repl.MaxSyncChunkBytes, repl.MaxSyncResponseRate, udprepl.MaxDatagramBytes) instead of duplicating literals."
requirements-completed: []
coverage:
  - id: D1
    description: "internal/config exports Config, Parse, Usage and ErrHelp; flag > env > default holds and --version short-circuits before any env error"
    requirement: OBSV-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/config/ -run 'TestParse|TestUsage' -v"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_helpListsEnvOnly|TestMain_version|TestMain_precedence' -v"
        status: pass
    human_judgment: false
  - id: D2
    description: "One Validate walks the 12-rule catalog: positivity for 13 durations plus the three integer knobs, retransmit floor, sync-response cap and chunk ceiling, datagram ceiling, positional rejection, and the peer/hello, ack/max, drift/lead and metrics-triple relations"
    requirement: OBSV-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/config/ -run 'TestValidate|TestRules|TestMaxSyncResponseRateExport' -v"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_validationRelations|TestMain_valueMaxNonPositive|TestMain_syncChunkBytesCeiling|TestMain_startupGateEmptyRange' -v"
        status: pass
    human_judgment: false
  - id: D3
    description: "cmd/sup-kv/main.go is wiring only: no flag registration, no env-name constant and no check* function remain; the 16-bit narrowing lives on Config.WireNodeID"
    requirement: OBSV-08
    verification:
      - kind: contract
        ref: "grep -n 'flag\\.|func check|SUPKV_' cmd/sup-kv/main.go (only doc-comment mentions remain) and go build ./..."
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ (whole smoke suite)"
        status: pass
    human_judgment: false
  - id: D4
    description: "persist.EncodeTo rejects a non-positive or over-ceiling value limit before the header write, wrapping storage.ErrValueTooLarge with an empty buffer"
    requirement: OBSV-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run 'TestCodec' -v"
        status: pass
    human_judgment: false
  - id: D5
    description: "The max-clock-lead default is 2 m in the binary, the core and --help, strictly above the unchanged 60 s drift-warn default, and the docs state the relation"
    requirement: OBSV-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ (discard model mirrors 2 m)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_maxClockLead' -v"
        status: pass
      - kind: contract
        ref: "make lint && make test (whole repo, 18 packages green)"
        status: pass
    human_judgment: false
duration: 22min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 02: One Configuration Package and One Validation Function Summary

**Extracted every flag, env twin, default, startup check and help renderer into `internal/config`, reduced `cmd/sup-kv/main.go` to wiring, and enforced the full range/relation catalog — plus the routed `sync_chunk_bytes` bounds, the retransmit floor, the drift/max-lead relation and the library-level codec value guard — so misconfiguration exits 2 naming the parameter before any socket or file work.**

## Performance

- **Duration:** 22min
- **Started:** 2026-11-23T07:14:24Z
- **Completed:** 2026-11-23T07:36:28Z
- **Tasks:** 3
- **Files modified:** 14 (6 created, 8 modified)

## Accomplishments

- `internal/config` is the single configuration surface: `Config` (exported fields, `NodeID uint64`), `Parse(args, getenv)`, `Validate(Config)`, `Usage(io.Writer)` and `ErrHelp`. No package-level flag set, no global config, no `slog.SetDefault` — `Parse` is pure over its inputs, so the package tests run fully parallel.
- `Validate` walks one immutable 12-rule catalog and returns `errors.Join` of every failure, each prefixed with its rule name and naming the offending parameter: required knobs, node-id range, positivity (13 durations + value-max-bytes + sync-chunk-bytes + sync-response-rate), retransmit 1 ms floor, sync-response cap (the newly exported `repl.MaxSyncResponseRate`), sync-chunk ceiling, datagram ceiling, peer-vs-hello (strict), ack-vs-max (equality legal), drift-vs-lead at millisecond resolution, the metrics certificate triple and host:port address shape.
- `cmd/sup-kv/main.go` holds no flag registration, no env-name constant and no `check*` function; `main` parses, handles help/version, validates, installs logging and wires. The 16-bit narrowing is `Config.WireNodeID()`, documented as total on a validated config.
- `--help` lists every flag with its `SUPKV_` twin, keeps the env-derived default (`SUPKV_MAX_CLOCK_LEAD=90s` shows `(default 1m30s)`), and ends with the `Environment-only (no flag):` block naming `SUPKV_LOG_LEVEL`, `SUPKV_LOG_FORMAT`, `SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY` and `SUPKV_METRICS_CLIENT_CA`.
- The max-clock-lead default is 2 m in the binary flag, the core constant/field comment, the discard test model and `--help`; the 60 s drift-warn default is unchanged, so the drift WARN window is reachable and `version.Validate`'s docs state the `max_clock_lead >= drift_warn + 1 ms` relation.
- `persist.EncodeTo` rejects `valueMaxBytes <= 0` and `> maxDumpValueBytes` as its first statement with a wrapped `storage.ErrValueTooLarge` and an empty buffer; `doc.go` states the shipped guard instead of deferring it (IN-04 closed).

## Task Commits

1. **Task 1 RED: failing parse, usage and env-only help tests** - `f38d644` (test)
2. **Task 1 GREEN: move flag/env parsing, validation and usage into internal/config** - `da86482` (feat)
3. **Task 2 RED: failing validation rule-table and relation process tests** - `f595ae8` (test)
4. **Task 2 GREEN: enforce every configuration range and relation in one rule table** - `555d74a` (feat)
5. **Task 3: guard the codec value limit and document the 2 m drift window** - `7b6efc8` (feat)

**Plan metadata:** `docs(07-02): complete configuration plan` (this commit)

## Files Created/Modified

- `internal/config/config.go` - package doc, `Config`, `ErrHelp`, `Parse`, the local `newFlagSet`/`register`/`registerDurationFlags` tables, the env twin constants, the design defaults and the `resolver` env seam
- `internal/config/usage.go` - `Usage` rendering the parser's registration plus the five-name env-only block
- `internal/config/validate.go` - the 12-rule catalog, `Validate`, and one named check per rule (datagram ceiling and metrics triple moved verbatim; all other rules new)
- `internal/config/config_test.go` - `TestParse`: help/ErrHelp, version-before-malformed-env, flag > env > default, empty env as unset, positional/unknown-flag naming, malformed env naming
- `internal/config/usage_test.go` - `TestUsage`: every flag with its env twin, `-version`, the env-only header and all five names
- `internal/config/validate_test.go` - `TestValidate` (one subtest per rule plus every boundary/relation row), `TestRules` (catalog length/names/order/owning subtest) and `TestMaxSyncResponseRateExport`
- `cmd/sup-kv/main.go` - wiring only; `usageError` helper keeps `os.Exit` in `main`
- `cmd/sup-kv/main_test.go` - `TestMain_helpListsEnvOnly`, `TestMain_validationRelations` (seven exit-2 rows) and the 2 m help assertions
- `internal/app/repl/core.go` - `MaxSyncResponseRate int = 1e9` exported and used by `normalized()`; `defaultMaxClockLead = 2 * time.Minute` with the reachability comment; the `MaxClockLead` field doc updated
- `internal/app/repl/discard_test.go` - the test model's `defaultMaxClockLead` and its two comments moved to 2 m
- `internal/app/persist/codec.go` - the value-limit guard before the header write, with the doc comment stating the domain and the empty-buffer contract
- `internal/app/persist/codec_test.go` - `TestCodec_encodeValueLimitGuard`: zero, negative and ceiling+1 rejected with an empty buffer, the ceiling itself writing a decodable generation
- `internal/app/persist/doc.go` - the IN-04 paragraph now documents the shipped guard
- `internal/app/version/version.go` - the `Validate` doc states the 2 m default and the drift/lead relation

## Decisions Made

- **`internal/config` outside `internal/app`**: it reads the environment and imports `internal/udprepl`/`internal/app/repl` for the ceiling oracles, which the no-I/O depguard rule forbids inside the pure core; the package uses the standard library and own-module imports only.
- **Rule names prefix every error**: `positive-values: value-max-bytes must be positive: --value-max-bytes=0`. The existing sentinel substrings survive, so every pre-existing process assertion still matches.
- **Default moved with the rule**: the drift-vs-lead rule would reject the shipped 1 m default in the same commit; the 2 m default and its help assertions therefore moved in Task 2 (see deviations).
- **`Usage` reads the process environment**: the help text must show the effective default (the pre-existing `SUPKV_MAX_CLOCK_LEAD=90s` contract), so the exported renderer resolves defaults from `os.Getenv` while `Parse` keeps the injectable seam.
- **Internal validation test**: `rules()` and `addrShape` are white-box by design; the plan's `validate_test.go` filename is kept with a justified `//nolint:testpackage` (the `internal/stand` precedent).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Moved the max-clock-lead default and its help assertions in Task 2, not Task 3**
- **Found during:** Task 2 (the drift-vs-lead rule landing)
- **Issue:** with the rule in place but the flag default still 1 m and `drift_warn` 60 s, the shipped default configuration fails its own validation (1 m < 60 s + 1 ms), so every default-configured binary run would exit 2 — violating the plan's prohibition "must not reject a valid existing configuration: every default stays accepted".
- **Fix:** `internal/config`'s `defaultMaxClockLead` moved to 2 m in the same commit as the rule (the core constant moved as the plan's Task 2 specifies), and `TestMain_maxClockLead`'s doc comment, subtest name and `(default 2m0s)` assertion moved with it. Task 3 then verified the state instead of re-editing it.
- **Files modified:** `internal/config/config.go`, `cmd/sup-kv/main_test.go`
- **Commit:** `555d74a`

**2. [Rule 2 - Missing critical functionality] `TestRules` and the export cross-check landed with Task 2's GREEN commit**
- **Found during:** Task 2 RED
- **Issue:** the meta-test iterates the unexported `rules()` and `TestMaxSyncResponseRateExport` reads `repl.MaxSyncResponseRate`; neither symbol exists before the implementation, so including them in the RED commit would have been a `[build failed]` run — an invalid RED record.
- **Fix:** the RED commit carries `TestValidate` (compile-clean, assertion-failing); the two symbol-dependent tests landed with the implementation, the 06-05/06-12 precedent.
- **Files modified:** `internal/config/validate_test.go`
- **Commit:** `555d74a`

**3. [Rule 1 - Bug] gosec G706/G115 false positives introduced by the extraction**
- **Found during:** Task 1 lint
- **Issue:** moving env reads behind `config.Parse(os.Args[1:], os.Getenv)` made gosec's taint analysis follow the parsed node id into `persist.Open`'s discarded count and flag `log.Printf("%d", count)` as log injection (G706). The `uint64 → uint16` narrowing then tripped G115.
- **Fix:** a helper with a reasoned `//nolint:gosec` on the count line (constant message, counted int, never operator text) and a reasoned `//nolint:gosec` on `WireNodeID` (Validate rejects the range first). Both carry the specific-linter and explanation AGENTS.md deviation 3 requires.
- **Files modified:** `cmd/sup-kv/main.go`, `internal/config/config.go`
- **Commit:** `da86482`

**4. [Rule 2 - Missing critical functionality] `validate_test.go` is an internal test with a justified `//nolint:testpackage`**
- **Found during:** Task 2 planning
- **Issue:** the plan requires `TestRules` to pin `len(rules())` and the rule names; `rules()` is unexported, and `testpackage`'s external-package rule cannot express the white-box pairing.
- **Fix:** the file stays at the planned path in `package config` with a one-line justification (the `internal/stand`/`internal/persistfile` precedent).
- **Files modified:** `internal/config/validate_test.go`
- **Commit:** `f595ae8`

## Issues Encountered

- A stale `gsd-plan-head-before-07-02` ledger pointed at commit `601a250` (not in this branch's history); it was reset to the plan-start HEAD `7325bf5` before measuring, following the 06-01/06-03 and 07-01 precedent.
- `requirements.ready-ids` reports OBSV-02 and OBSV-08 blocked: both ids are shared across plans 07-01/02/03/10/11 (#2388 shared-ID gate), so no checkbox was marked and `requirements-completed` stays empty; 07-11 closes the phase-level checkboxes. OBSV-08's "shutdown is testable" clause is also still 07-04's to prove.
- The known v1.0 row collision fired in `roadmap update-plan-progress`: the tool bumped the historical `7. v1.0 — Docs, Examples and Release` row to 2/11 while leaving the v1.1 phase-7 row at 0/11. Repaired by hand (v1.0 restored to 1/11, v1.1 phase 7 set to 2/11 In Progress).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

`config.Parse`/`config.Validate`/`config.Usage` are the only configuration entry points, so plan 07-04 can inject the `run` orchestration over them and plan 07-03 can treat `internal/config` as adapter-side for the depguard direction. The 2 m default leaves plan 07-05 to widen the version property's edge window and near-version generator around the new bound. No blockers; 9 plans remain in Phase 07 — ready for 07-03.

## Self-Check: PASSED

- Created files exist: `internal/config/config.go`, `usage.go`, `validate.go`, `config_test.go`, `usage_test.go`, `validate_test.go` — all FOUND.
- Commits exist: `f38d644`, `da86482`, `f595ae8`, `555d74a`, `7b6efc8` — all ancestors of HEAD.
- Full gates green: `make lint` exit 0; `make test` exit 0 (18 packages, `internal/config` 90.3% statements).
