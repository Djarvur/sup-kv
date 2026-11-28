---
phase: 09-documentation-and-release
plan: 04
subsystem: documentation
tags: [registry, spec-deviations, doc-anchor-tests, tdd, citation-resweep, m-assignment]

requires:
  - phase: 09-documentation-and-release (09-01, 09-02, 09-03)
    provides: the green cmd/sup-kv doc-anchor suite the gate joins, the FORMATS as-built truth M-24 cites, and the M-23/M-48…M-52 closures the master assignment records
  - phase: 07-observability-and-configuration
    provides: the New-3 stale-citation inventory this plan resolves
  - phase: 08-performance
    provides: the S-FAIL-15 burst evidence pattern (TestRed_R02_burst / TestPerfBurstDistinctKeys / TestEngine_burstBoundedConfirmation)
provides:
  - cmd/sup-kv/deviations_test.go — TestSpecDeviationsNoUnclosedRows (the milestone exit criterion: zero unclosed table cells, status-vocabulary enum enforced, 150-row vacuity floor) and TestSpecDeviationsLinksResolve (every cited path, file#TestName pair and path:lines (`Symbol`) pair resolves; 250-check vacuity floor; `:lines`/`#Test` continuation inheritance; glob citations resolved)
  - docs/SPEC-DEVIATIONS.md closed: all 22 `fixed-in-v1.1(...)` rows flipped to `implemented` with as-built summaries and re-derived evidence; every Links cell re-derived line by line with symbol verification (720 citations resolve under the gate)
  - .planning/phases/07-observability-and-configuration/deferred-items.md — the New-3 stale-citation item marked RESOLVED, citing this plan and the re-derived rows
  - the master M-01…M-52 assignment table (below) — every documented mismatch has exactly one primary owning plan and a fix kind; 09-11's completeness check greps this table
affects: [09-05…09-10 (own their assigned M rows), 09-11 (greps the M set complete and runs the release verification), the milestone exit criterion (ROADMAP success criterion 3)]

actuals:
  tokens: 52000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "registry gate by row-prefix form: the row matcher matches whitespace-normalized lines starting `| S-`, so the preamble's vocabulary token — the same `fixed-in-v1.1(` string — can never count as a row; the enum check rejects any status outside the five closed-vocabulary values"
    - "citation resolver with continuation inheritance: a `:lines` token inherits the last path in the cell, a `#TestName` token inherits the last file#Test citation, and glob citations resolve through filepath.Glob — the 06-10 neighbouring-declaration lesson is now a permanent gate"

key-files:
  created:
    - cmd/sup-kv/deviations_test.go
    - .planning/phases/09-documentation-and-release/red-evidence/09-04-task1-red.json
  modified:
    - docs/SPEC-DEVIATIONS.md
    - cmd/sup-kv/performance_test.go
    - .planning/phases/07-observability-and-configuration/deferred-items.md

key-decisions:
  - "the gate excludes the preamble by construction, not by line-number arithmetic: rows are lines whose normalized form starts `| S-`, and the status check parses the fourth `|` cell — the preamble's vocabulary line carries the token but never the row form (T-09-11's mitigation)"
  - "the exit criterion is zero unclosed rows, not zero non-implemented rows: deferred-v2(SNAP-01) and the two accepted-deviation(...) rows are legitimate permanent statuses and pass the enum; S-FAIL-15's row was still carrying the unclosed status cell (only its summary was as-built), so it flipped with the other 21"
  - "TestSpecDeviationsBurstRow's status anchor was re-pointed from the pre-sweep literal `fixed-in-v1.1(PERF-02)` to `implemented` — the anchor froze the status this plan's own exit criterion removes; its purpose (the row cites the shipped tests and bounds) is unchanged"
  - "citation re-derivation was symbol-extent based, never a path swap: ~70 drifted citations beyond the New-3 seeds were corrected (core.go growth moved applyRemote to 1636-1703, handleDataSync to 1819-1848, scanRetransmit to 2992-3002; the persist scheduler/engine split moved run/runDump/rearm; the watch hub moved subscribe/dispatch; loader.go's Open owns the clock seed), and S-PAR-01..04's flag-name symbols were re-derived to the registered form (node-id, not --node-id)"
  - "S-FAIL-15's three test citations were upgraded to the file#TestName form (stand_red_test.go, perf_test.go, engine_test.go) so the permanent gate resolves them like every other row"

patterns-established:
  - "the registry closure gate is a normal make-test citizen: `go test -race ./...` carries the milestone exit criterion from now on — a regression flip or a dead citation fails CI, not an eyeball"

requirements-completed: [DOCS-01, DOCS-08]

coverage:
  - id: registry-gate
    verification:
      - kind: tests
        ref: cmd/sup-kv/deviations_test.go#TestSpecDeviationsNoUnclosedRows
        status: pass
      - kind: tests
        ref: cmd/sup-kv/deviations_test.go#TestSpecDeviationsLinksResolve
        status: pass
      - kind: tests
        ref: cmd/sup-kv/deviations_test.go (RED evidence red-evidence/09-04-task1-red.json, classifier RED_EVIDENCE_OK)
        status: pass
    human_judgment: false
  - id: closed-registry
    verification:
      - kind: command
        ref: "grep -cE '^\\| S-' docs/SPEC-DEVIATIONS.md = 167 && grep -c 'fixed-in-v1.1(' = 1 (preamble only)"
        status: pass
      - kind: command
        ref: "New-3 stale grep (core.go:1510 / store.go:252 / server.go:433 on the six known rows) negative"
        status: pass
    human_judgment: false
  - id: new3-resolution
    verification:
      - kind: command
        ref: "grep -q 'New-3' .planning/phases/07-observability-and-configuration/deferred-items.md (row carries the RESOLVED record with commit a0665d2)"
        status: pass
    human_judgment: false
  - id: m-master-assignment
    verification:
      - kind: command
        ref: "all 52 M-xx tokens present in this summary (the 09-11 completeness grep)"
        status: pass
      - kind: command
        ref: "already-closed rows re-derived by grep against the current tree (evidence column)"
        status: pass
    human_judgment: false
---

# Phase 9 Plan 4: Close the Deviation Registry Summary

The deviation registry reached its terminal state: zero unclosed rows — all 22 `fixed-in-v1.1(...)` cells flipped to `implemented` with re-derived evidence — machine-checked by a two-test gate in `cmd/sup-kv` that cannot pass vacuously, plus the master M-01…M-52 assignment table that gives every remaining documented mismatch exactly one owning plan.

Duration: ~35 min (2026-11-27T22:23Z → 22:57Z); 3 tasks; 3 commits (f6980ab, a0665d2, 18f8392).

## Accomplishments

- **The registry gate (Task 1, TDD).** `cmd/sup-kv/deviations_test.go` in package `main_test` with the honest-scope header (the gate proves closure form and citation existence, never line-number currency): `TestSpecDeviationsNoUnclosedRows` parses the `| S-` table rows (150-row vacuity floor), enforces the five-value closed status vocabulary, and fails naming every row whose status carries the unclosed `fixed-in-v1.1(` marker; `TestSpecDeviationsLinksResolve` walks every Links cell for the three citation classes — repo paths (globs resolved through filepath.Glob), `file_test.go#TestName` pairs, and `path:lines` (`Symbol`) pairs with continuation inheritance — under a 250-check vacuity floor. RED recorded (red-evidence/09-04-task1-red.json, classifier RED_EVIDENCE_OK): the closure gate failed naming exactly the 22 unclosed rows; the link resolver failed on 4 genuine stale citations after 433 checks.
- **The closed registry (Task 2).** All 22 rows flipped to `implemented` with as-built summaries (the v1.0-gap wording dropped, what the shipped code now does stated) and re-derived Links cells: the flipped rows cite the closing work's symbols and tests (S-PROTO-04 → `discard_test.go#TestDiscard_classes`; S-PROTO-15/16/S-ALG-REP-04/S-SCN-03 → `applyRemote`'s branches + `TestReplication_staleAnsweredNotAcked` + `TestWaitIntegration_staleRecordNotAcked`; S-PROTO-14/S-ALG-CU-09 → the notify-before-mapError paths + `TestServer_dumpFailedWriteNotifies`; S-PERS-13 → the paced replay seam + `TestServer_snapshotLoadNotifies`; S-ALG-CU-08 → `handleDumpFailure`/`dumpFamilies`/`logDumpFailure`; S-ALG-RC-09 → the bounded identity queue + `TestEgress_boundedDeduped`/`TestEgress_capDropsOldest`; S-FAIL-09 → the paced recovery + `TestRed_R01_pacedSync20k` + the `red-r01-x20` job; S-FAIL-14 → the received-lead drift counter + both drift tests; S-SEM-08 → the three simulator bound tests + the two stretch-path notify tests). Every other row's Links cell was reswept line by line — ~70 further drifted citations corrected with symbol extents (the New-3 seeds plus core.go growth, the persist scheduler/engine split, the watch hub, loader.go's Open, the S-PAR flag-name symbols). Registry shape intact: 167 rows, identical ID set and order, exactly one `fixed-in-v1.1(` token (the preamble's vocabulary line). Gate green under -race: 720 citations resolved.
- **The New-3 resolution and the M master assignment (Task 3).** deferred-items.md's New-3 row records the resolution (commit a0665d2, the re-derived locations, the permanent guard). The master M-01…M-52 assignment table below assigns every documented mismatch a primary owner and fix kind; the already-closed rows were re-derived by grep against the current tree, not taken from research §2 on trust.

## Master M-01…M-52 assignment (D-02 — the table 09-11's completeness check greps)

Every M row appears exactly once with a primary owner; two-half rows name both halves. Fix kinds: doc fix, code fix, executed claim, banner-covers (DOC-12's bar: README's description of IDEA.md matches its HISTORICAL banner), already-closed (evidence re-derived from the current tree), verify-only.

| M | Closing plan | Fix kind | Evidence / notes |
|---|---|---|---|
| M-01 | 09-08 | doc fix | README's stale "release artifacts in preparation" sentence replaced by the install-from-release section (tag v1.0.0 exists) |
| M-02 | 09-08 | doc fix | the quickstart block's cleanup claim made true (the `rm` line) or the claim dropped; assertion-table upkeep |
| M-03 | 09-08 | doc fix | the macOS broadcast-derive description corrected (ipconfig/en0 hard-coding and the silent fallback stated) |
| M-04 | 09-10 (code half: `.goreleaser.yaml` `files:`) + 09-08 (doc half: README archive-links / EXAMPLES unpack wording) | two-half | archives gain DEPLOYMENT.md + the unit; the doc wording matches what archives carry |
| M-05 | 09-08 (README/CHANGELOG half) + 09-05 (OPERATIONS half) | two-half | every "proven" carries a named test — `TestProvenClaims` lands with 09-08 and covers the corpus |
| M-06 | 09-05 | doc fix | OPERATIONS' recovery-bound claim conditioned on the loss window (the partition bound's territory named) |
| M-07 | 09-01 (doc half — DONE) + 09-05 (OPERATIONS cross-link half) | two-half | docs/CONFIGURATION.md documents all 23 registered flags with defaults, machine-checked by TestConfigurationDoc; the OPERATIONS-side link is 09-05's edit |
| M-08 | already-closed | Phase 5 plan 05-01 (PERSIST-01, ADR 0011) | `storage.ValueMaxBytes` is the one authoritative limit (store.go:12-26); codec.go and main.go carry no second copy of 61440 (grep 0/0) |
| M-09 | already-closed | Phase 7 plan 07-06 (OBSV-03, ADR 0004) | `supkv_merges_conflicts_total` HELP = "Pending (unacked) local writes displaced by a replicated value." (metrics.go:316); counted from the pending set (core.go:1683) |
| M-10 | already-closed | Phase 7 plans 07-05 + 07-10 (OBSV-05) | drift is the received reading's lead (clock.go:129 `driftBetween(remoteL, …)`); the runbook states only that direction ("received version leads" in OPERATIONS.md) |
| M-11 | already-closed | Phase 7 plan 07-09 | `process_resident_memory_bytes` reads the current RSS from /proc/self/statm on Linux (metrics.go:539,552) |
| M-12 | already-closed | Phase 7 plan 07-04 | exit codes 0/1/2/3 (main.go:63-65 `exitStartup`/`exitUsage`/`exitDumpFailed`); startup failures log one static ERROR |
| M-13 | already-closed | Phase 7 plan 07-08 | SUPKV_LOG_LEVEL/FORMAT documented; the lifecycle catalog's level matrix pinned by `TestCatalogLevels` (catalog_internal_test.go:41) |
| M-14 | already-closed | Phase 7 plan 07-07 | one load-outcome record per start: Backup WARN / Unusable ERROR (`logLoadStats` main.go:450, `logUnusableGenerations` main.go:565) |
| M-15 | 09-05 | doc fix | OPERATIONS' trust model states all three properties: no authentication, no encryption, no transport-security option (the gRPC half disclosed) |
| M-16 | 09-07 | code fix | `deploy/sup-kv.service` lands with the exit-code-aware restart policy, CI-verified (systemd-analyze + ExecStart smoke) |
| M-17 | 09-08 | doc fix | EXAMPLES' wait loop uses the quoted `"SERVING"` grep and the assertion table's row is quoted too (NOT_SERVING-satisfiable forms both fixed) |
| M-18 | 09-08 | doc fix | the service-count sentence corrected to the four services the listing shows |
| M-19 | 09-08 (examples half: the create_if_absent block) + 09-06 (API half: the per-RPC error tables with AlreadyExists/Unavailable/ResourceExhausted) | two-half | the zero-occurrence strings land where they belong; assertion-table row added |
| M-20 | 09-08 | doc fix | EXAMPLES' restore text states the full-rollback semantics (keys absent from the snapshot are deleted) matching `restoreChanges` |
| M-21 | 09-08 | doc fix | EXAMPLES gains the platform note (the CI-verified Linux walkthrough; the darwin skip stated) |
| M-22 | already-closed | Phase 6 plan 06-02 (A-01) + Phase 7 EXAMPLES assertions | zero-peer answers the honest `(false, 0, 0)` (wait.go); EXAMPLES.md:186 asserts `ackedCount`/`totalPeers` |
| M-23 | 09-03 (DONE) | code fix | verify-then-delete: zero conflict-journal survivors in production comments; the CLAUDE.md inspectability line scrubbed (09-03 commit 2a66559) |
| M-24 | banner-covers | FORMATS.md (09-02) is the as-built dump-format truth (JSON Lines, not one JSON document); IDEA.md stays bannered — DOC-12's bar is README-vs-banner, and the banner names the as-built trio |
| M-25 | banner-covers | the HISTORICAL banner covers the zero-filler narration; the as-built truth is version.go's layout comment (S-HLC-07's accepted-deviation row) |
| M-26 | banner-covers | the banner covers the run_id narration; as-built truth is `NewRunID` (core.go:445-449, 16 random bytes) |
| M-27 | banner-covers | the banner covers the unicast-reply narration; the as-built stale-reply branch is registry row S-ALG-REP-04 (implemented, applyRemote:1641-1644) |
| M-28 | banner-covers | the banner covers the per-part clock narration; as-built truth is S-HLC-04/S-ALG-RC-05 (implemented, handleDataSync:1827-1834) |
| M-29 | 09-09 (ARCHITECTURE parameter truth) + 09-06 (API.md covers the RPC list) | doc fix, two-half | ARCHITECTURE.md carries the as-built parameter surface; API.md's RPC list covers the Get/Set/Delete/List/Watch/snapshot set |
| M-30 | 09-06 (API half: key-prefix convention + client last-value cache pattern) + 09-09 (ARCHITECTURE half) | two-half | both conventions leave IDEA.md-only status |
| M-31 | 09-08 | doc fix | README's topology wording: the crossed demo ports named as the same-host trick; the CI darwin cross-compile cell stated |
| M-32 | 09-03 (DONE) | doc fix | AGENTS.md cites go 1.27.4 (TestAgentsStackAudit, commit 2a66559) |
| M-33 | 09-03 (DONE) | doc fix | the tools-directly CI sentence; lint = config verify + run; test = -race -cover |
| M-34 | 09-03 (DONE) | doc fix | the five-package internal/app split plus the harness packages named; package inventory test-enforced |
| M-35 | 09-03 (DONE) | doc fix | the four adapters named (grpcapi, udprepl, persistfile, metrics) plus config/stand |
| M-36 | 09-03 (DONE) | doc fix | the two direct runtime requires stated; the historical Phase-1 count sentence forbidden by the audit |
| M-37 | 09-03 (DONE) | doc fix | the seven tool directives named on one source line each |
| M-38 | 09-03 (DONE) | doc fix | rapid removed from CLAUDE.md; the hand-rolled generators stated |
| M-39 | 09-03 (DONE) | doc fix | the discrete-event simulator stated (no goroutines, no wall clock) |
| M-40 | 09-03 (DONE) | doc fix | verified-then-kept: the SetReadBuffer advice became accurate when Phase 3 shipped XPORT-05 (socket.go) |
| M-41 | 09-03 (DONE) | doc fix | the one-line Encoder stated |
| M-42 | 09-03 (DONE) | doc fix | the ldflags version identity stated |
| M-43 | 09-03 (DONE) | doc fix | x/sync v0.25.0 indirect; golangci-lint v2.15.0; the plugin/runtime versions as-built |
| M-44 | 09-03 (DONE) | doc fix | "Conventions not yet established"/"Architecture not yet mapped" replaced by the as-built section |
| M-45 | 09-10 (verify-only) | already-closed evidence | supkvapi.proto's phase references were scrubbed in Phase 6 (06-10); grep for phase/plan-0 is clean today; 09-10 re-verifies both protos (DOC-21) |
| M-46 | 09-10 (verify-only) | already-closed evidence | replproto's "Phase 4 wave-1" wording is gone (04-13's package-doc rewrite); grep clean today; the decision-number scrub (D-01/D-02 comments) rides 09-10 |
| M-47 | already-closed | Phase 5/6 API work (A2/D-09) | the wait deadline: negative = default, capped at 10× the operator default (wait.go:32 `waitTimeoutCeilingFactor`, `waitTimeout` at :112); API.md documents the caller bound |
| M-48 | 09-03 (DONE) | code fix | the false "internal/port holds no interfaces" claim scrubbed from store.go/doc.go/core.go comments |
| M-49 | 09-03 (DONE) | code fix | the Makefile buf-breaking dead probe branch removed (real comparison proven green); ci.yml comment fixed comment-only |
| M-50 | 09-03 (DONE) | code fix | the dependabot comment states the 2026-11-18 public flip |
| M-51 | 09-03 (DONE) | executed claim | the by-id ruleset command ran 2026-11-28 and printed exactly the documented shape; recorded verbatim in branch-protection.md |
| M-52 | 09-03 (DONE) | code fix | `make help` parses the `## target:` comments |
| — release verification | 09-11 | executed claim | the release itself (tag, assets, archive contents) is 09-11's executed claim, not an M row |

Count: M-01…M-52 each exactly once (M-07/M-04/M-05/M-19/M-29/M-30 as two-half rows naming both owners; M-45/M-46 verify-only with their already-closed evidence cited).

## Deviations from Plan

- **[Rule 1 - Stale frozen anchor] TestSpecDeviationsBurstRow's status literal re-pointed** — Found during: Task 2 | Issue: the Phase-8 anchor requires the literal `fixed-in-v1.1(PERF-02)` in the S-FAIL-15 row, but the plan's own exit criterion (grep count of `fixed-in-v1.1(` == 1) requires that status cell to become `implemented` — the planner's "already flipped by 08-10" assumption held for the row's summary gloss only, not its status cell | Fix: flipped the S-FAIL-15 status cell with the other 21 rows and re-pointed the anchor's first required literal to `implemented`, keeping the three test citations (upgraded to file#TestName form) and both bounds | Files modified: docs/SPEC-DEVIATIONS.md, cmd/sup-kv/performance_test.go | Verification: TestSpecDeviationsBurstRow green; the grep-count verify passes | Commit: a0665d2
- **[Rule 2 - Missing critical] S-PAR-01..04 flag-name citations were genuinely stale** — Found during: Task 1 RED | Issue: the four rows cite backticked `--node-id`/`--bind-addr`/`--broadcast-addr`/`--dump-path` in internal/config/config.go, but the file registers `"node-id"` etc. without dashes — the gate's RED state included these 4 unresolvable citations | Fix: re-derived the parentheticals to the registered flag names during Task 2's resweep | Files modified: docs/SPEC-DEVIATIONS.md | Verification: TestSpecDeviationsLinksResolve green (720 checks) | Commit: a0665d2

**Total deviations:** 2 auto-fixed (1 stale frozen anchor, 1 stale-citation cluster the new gate caught by design). **Impact:** none on scope — both are the gate doing its job; the registry's shape, ID set and row order are untouched.

## Issues Encountered

None. (The gate's first implementation had an off-by-one that silently dropped symbol associations — caught by cross-checking the Go walk against a scratch Python census before recording RED; fixed before the RED evidence was taken, so the committed gate's RED record reflects its real behavior.)

## Verification

- `go test -race -count=1 ./cmd/sup-kv/ -run 'TestSpecDeviationsNoUnclosedRows|TestSpecDeviationsLinksResolve|TestSpecDeviationsBurstRow' -v` — PASS (0 unclosed rows; 720 path/symbol/test citations resolved across 167 rows; BurstRow anchor green).
- Plan verify 2: `grep -cE '^\| S-' docs/SPEC-DEVIATIONS.md` == 167 and `grep -c 'fixed-in-v1.1('` == 1 (the preamble's vocabulary line) — PASS.
- Plan verify 3: the New-3 stale grep (`core.go:1510|store.go:252|server.go:433` on rows S-HLC-04/HLC-10/DM-04/DM-08/ALG-REP-02/ALG-CU-07) — negative, PASS.
- `go vet ./cmd/sup-kv/` and `go tool golangci-lint run ./cmd/sup-kv/` — clean.
- RED evidence classifier: `RED_EVIDENCE_OK` on red-evidence/09-04-task1-red.json.

## TDD Gate Compliance

- RED: `test(09-04)` commit f6980ab precedes GREEN; both target tests failed on their planned assertions (22 unclosed rows named; 4 unresolvable citations named), classifier RED_EVIDENCE_OK.
- GREEN: `feat(09-04)` commit a0665d2 makes both tests pass under -race.
- REFACTOR: none needed (the lint fixes landed inside the RED commit before its evidence was recorded).

## Self-Check: PASSED

- All three tasks executed; each committed individually (f6980ab, a0665d2, 18f8392).
- All acceptance criteria re-run after the final edit: gate green, 167 rows, one legitimate token, New-3 grep negative, New-3 marked resolved, all 52 M tokens present above.
- Success criteria: zero unclosed registry rows enforced by a non-vacuously-passing test; every citation resolves; every M row has an owning plan. The registry — Phase 1's deliverable — is at its terminal state.

Next: Ready for 09-05 (the operator reference owns the OPERATIONS halves of M-05/M-06/M-07/M-15).
