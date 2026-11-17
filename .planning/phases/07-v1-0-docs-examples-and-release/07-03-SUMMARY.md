---
phase: 07-v1-0-docs-examples-and-release
plan: 03
subsystem: docs
tags: [grpc, grpcurl, docs-as-test, operations-runbook, tdd, option-b, reflection]

# Dependency graph
requires:
  - phase: 07-v1-0-docs-examples-and-release
    provides: plan 07-01 — the executable-doc harness (extractMarkedBlocks/runBashVerbatim/supkvBinDir/grpcurlShim), the doc-port band [27900, 28100), reflection registration, and the option-b broadcast verdict
  - phase: 07-v1-0-docs-examples-and-release
    provides: plan 07-02 — the 07-02 hard-assert block idiom (bounded retry + one unconditional final command) and the deterministic darwin-skip precedent this plan's examples leg follows
  - phase: 06-hardening-failure-scenarios-observability-benchmarks
    provides: the supkv_* metrics families, the D-10 conflict-journal cancellation, and the operational contract (exit codes, D-19 shutdown order) OPERATIONS.md states
provides:
  - docs/EXAMPLES.md as the full every-RPC grpcurl walkthrough — ten titled CI-executed blocks with per-block output assertions (DOCS-03 / D-04 / D-05)
  - docs/OPERATIONS.md — the single operations file carrying the five mandated DOCS-02 statements plus monitoring/lifecycle, trust model first (D-01)
  - TestOperationsDoc presence guard (41 anchors + trust-model-firstness assertion)
  - The blockSurvivors harness mode (cross-block pair survival with a PID-file cleanup backstop)
affects: [07-04 (release — README's docs/OPERATIONS.md link now resolves), 07-05, phase verification (DOCS-02 manual truth UAT)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 11494   # chars/4 over git diff 629fff4..19be25b (docs-dominated diff, no dep churn)
  tasks: 2
  commits: 6      # MEASURED: git rev-list --count 629fff4..19be25b

plan_head_before: 629fff4fd438da8976aecbe1693a36deb0d42680
plan_head_after: 19be25b6d8d78f6feda596880346807c735edfb1

# Tech tracking
tech-stack:
  added: []       # no new libraries/tools — consumes the 07-01 harness and tool directives
  patterns:
    - "Title-keyed assertion table: the doc's ### headings ARE the keys; table↔doc coverage fatals in both directions, so the walkthrough cannot silently lose an RPC and no marked block can run unasserted"
    - "blockSurvivors harness mode: reap (default, the 07-01 leak guarantee) vs keep (the pair-start block whose two nodes are the walkthrough's shared state); a PID-file t.Cleanup backstops any mid-walk abort"

key-files:
  created:
    - docs/OPERATIONS.md
    - cmd/sup-kv/operations_test.go
    - .planning/phases/07-v1-0-docs-examples-and-release/tdd-red-07-03-task1.json
    - .planning/phases/07-v1-0-docs-examples-and-release/tdd-red-07-03-task2.json
  modified:
    - docs/EXAMPLES.md
    - cmd/sup-kv/examples_test.go
    - cmd/sup-kv/readme_test.go

key-decisions:
  - "TestExamplesDoc's contract became a title-keyed assertion table: every marked block must carry an asserted output (a block outside the table is refused), and every table entry must have its block — the every-RPC completeness of the walkthrough is now machine-enforced by title, not by prose review"
  - "runBashVerbatim gained a blockSurvivors parameter (named type, not a bool control flag — revive flag-parameter): the walkthrough's pair-start block keeps its two nodes alive across blocks because the 07-01 group-reaper killed the shared pair; failed blocks still reap, and a PID-file t.Cleanup kills the pair if any block fatals before the stop block; the README test path is unchanged (reapSurvivors)"
  - "Every asserted output substring is captured from a real run: grpcurl prints pretty JSON with camelCase protojson names (\"acked\": true, ackedCount, totalPeers) and omits zero-valued fields — the plan's prose field names (acked_count/total_peers, \"acked\":true) do not match the wire bytes and were corrected from the captured output (Pitfall 3/5 discipline: the CI test is the generator of truth)"
  - "OPERATIONS.md's drift symptom names the metric (supkv_clock_drift_warnings_total), not 'warnings in logs' — the code has no dedicated drift log line; the doc states only what the binary actually does; the runbook recommends fixing NTP/PTP and explicitly does NOT recommend restarts or any sup-kv-side repair (the plan's prohibition)"
  - "DOCS-02 and DOCS-03 marked complete: the presence guard and the CI-executed examples are delivered; OPERATIONS.md content TRUTH stays the phase's manual UAT (07-VALIDATION.md manual table) — recorded as WINDOWS id 7 (unmet-truth), surfaced for phase verification"

patterns-established:
  - "Final-line hard asserts in shared-state doc blocks: every bounded wait ends in a command whose failure propagates as the block's exit status (grep && cat as the last line; ps -p PID && exit 1 before the closing echo) — mid-list && failures do not trip bash -e, so the assert must be the block's last command to be fatal"
  - "PID-file state handoff between marked blocks: shell variables cannot cross block boundaries, so the pair-start block writes ex-a.pid/ex-b.pid and the stop block (and the harness's cleanup backstop) read them"

requirements-completed: [DOCS-02, DOCS-03]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "docs/EXAMPLES.md full every-RPC walkthrough: ten titled CI-executed blocks (pair start on the 280xx doc band, reflection discovery, missing-key Get NotFound, cross-node write/read, wait_for_replication acked aggregate, delete+list with absence proof, Watch snapshot-then-live, health, snapshot round-trip, graceful stop) — D-04/D-05"
    requirement: DOCS-03
    verification:
      - kind: integration
        ref: "cmd/sup-kv/examples_test.go#TestExamplesDoc (green standalone, with TestReadme|TestOperations, and inside the full -race package suite; 10 markers)"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/OPERATIONS.md exists as ONE file with the five mandated DOCS-02 statements (trust model first and prominent, 60 KiB/61440 envelope, read-your-writes local reads, not-a-lock warning, clock-drift runbook) plus monitoring and lifecycle/exit codes — D-01"
    requirement: DOCS-02
    verification:
      - kind: unit
        ref: "cmd/sup-kv/operations_test.go#TestOperationsDoc (41 anchors + first-##-is-Trust-model assertion, green)"
        status: pass
    human_judgment: true
    rationale: "The presence guard proves the topics exist, not that each statement is TRUE of the shipped binary — content truth is 07-VALIDATION.md's manual UAT table (WINDOWS id 7); overstatement risk is exactly what T-7-04 assigns to human review"
  - id: D3
    description: "Option-b darwin legs for both doc tests: deterministic loud skips naming the recorded 07-01 macos-15 verdict — the walkthrough runs for real on Linux gate cells only"
    verification:
      - kind: unit
        ref: "cmd/sup-kv/examples_test.go#examplesSkipDarwin (darwin-only path; not executable on this linux host)"
        status: unknown
    human_judgment: true
    rationale: "The darwin skip path and the macos-15 gate cell's green-with-skip can only be confirmed on the next PR gate run; same adjudication family as WINDOWS ids 5-6 (the 07-01/07-02 option-b skips)"

# Metrics
duration: 20min
completed: 2026-11-17
status: complete
---

# Phase 7 Plan 3: Full API Examples and Operations Doc Summary

**EXAMPLES.md grows into the ten-block every-RPC grpcurl walkthrough (per-block output assertions, real-run pretty-printed camelCase outputs) and OPERATIONS.md lands as the one-file ops contract with the trust model first — both behind TDD RED/GREEN gates, both drift-guarded**

## Performance

- **Duration:** 20 min (2026-11-17 13:10–13:31 UTC)
- **Started:** 2026-11-17T13:10:35Z
- **Completed:** 2026-11-17T13:31:28Z
- **Tasks:** 2 (both TDD: RED → GREEN)
- **Files modified:** 7 (3 code/doc + 2 planning evidence + 2 committed docs/tests overlap — see Files below)

## TDD Gate Compliance

Both tasks ran the full cycle; every RED was classified by `gsd_run check tdd-red-evidence` before its GREEN.

- **Task 1 (EXAMPLES walkthrough):** RED `c36f983` (assertion table extended; the seed doc fails the coverage pre-flight naming "Start the demo pair") — evidence `tdd-red-07-03-task1.json`, `RED_EVIDENCE_OK` (target_test_failed), committed `a842229` BEFORE GREEN. GREEN `4cc72ed` (full doc; test green). No REFACTOR: the GREEN code is the reused harness plus the doc; the only post-GREEN changes were strict-lint code-direction fixes folded into the GREEN commit.
- **Task 2 (OPERATIONS):** RED `ad8bb9a` (read error naming the missing file — the planned RED) — evidence `tdd-red-07-03-task2.json`, `RED_EVIDENCE_OK` (target_test_failed), committed `e4fe92a` BEFORE GREEN. GREEN `19be25b`. No REFACTOR needed (one modernize fix folded into GREEN).
- Gate log check: `test(07-03):` and `feat(07-03):` commits present in RED→GREEN order for both tasks.

## Accomplishments

- Every RPC of `supkvapi.v1.SupKVService` now has a copy-paste grpcurl example executed verbatim by CI: Get (missing and present), Set (plain and `wait_for_replication`), Delete, List, Watch (snapshot-then-live stream, background-and-kill shape), SnapshotCreate/SnapshotLoad round-trip, plus reflection discovery (`list`/`describe`) and the standard health Check — DOCS-03/D-04/D-05
- Per-block output assertions: the wait block must print `"acked": true` with non-zero `ackedCount`/`totalPeers`; the cross-node Get must carry `c3VwLWt2`; the missing-key Get must surface the NotFound error text; the Watch output must carry the live event key — all substrings captured from a real run (pretty-printed camelCase protojson; grpcurl omits zero-valued fields)
- docs/OPERATIONS.md authored as ONE file (D-01) with the trust model FIRST and prominent: no authentication, no encryption, CRC32 integrity-only, HMAC future work, the always-on reflection disclosure beside it — then the REPL-10 convergence bounds, the 60 KiB/61440 envelope with flag/env twins and InvalidArgument rejection, read-your-writes local-read semantics, the not-a-lock warning (linearizability/fencing, LWW, `supkv_merges_conflicts_total` as the observable), the clock-drift runbook (metric symptom, `--drift-warn` 60s, fix NTP/PTP — no restart recommendation), monitoring (opt-in `--metrics-addr`, the full `supkv_*` family list, env-only mTLS triple), and lifecycle exit codes 0/1/2 with `Restart=on-failure` and the `SUPKV_LOG_*` knobs — DOCS-02
- TestOperationsDoc pins 41 anchors plus a firstness assertion (the first `##` section must be `## Trust model` — usage instructions cannot creep ahead of the trust warning); its header states the honest scope: presence guard only, content truth is manual UAT
- Full gate parity green: `make lint` and `make test` (race) both clean at plan close

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): assertion table + coverage pre-flight** - `c36f983` (test)
2. **Task 1 RED evidence record** - `a842229` (docs)
3. **Task 1 (GREEN): full walkthrough + blockSurvivors harness mode** - `4cc72ed` (feat)
4. **Task 2 (RED): failing TestOperationsDoc** - `ad8bb9a` (test)
5. **Task 2 RED evidence record** - `e4fe92a` (docs)
6. **Task 2 (GREEN): docs/OPERATIONS.md** - `19be25b` (feat)

**Plan metadata:** this commit (docs: complete plan).

## Files Created/Modified

- `docs/EXAMPLES.md` - full walkthrough: ten titled marked blocks, real outputs in prose, base64/camelCase/zero-omission wire facts up front, PID-file handoff explained
- `cmd/sup-kv/examples_test.go` - title-keyed assertion table (10 entries), coverage pre-flight both directions, per-block mustContain, option-b darwin skip, blockSurvivors mode + PID-file t.Cleanup backstop
- `cmd/sup-kv/readme_test.go` - call-site update for the runBashVerbatim signature (reapSurvivors — behavior unchanged)
- `docs/OPERATIONS.md` - the five mandated statements + monitoring/lifecycle, trust model first
- `cmd/sup-kv/operations_test.go` - TestOperationsDoc: 41 anchors + firstness assertion, honest-scope header
- `.planning/phases/07-v1-0-docs-examples-and-release/tdd-red-07-03-task1.json` / `tdd-red-07-03-task2.json` - RED evidence records

## Decisions Made

See key-decisions in the frontmatter. Additionally: the snapshot round-trip landed as a fully executed marked block (the executor judgment the plan allowed) — two unary RPCs with `{}` responses fit the bounded-block discipline trivially, exceeding the "runnable commands" minimum.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The 07-01 process-group reaper killed the walkthrough's shared pair**
- **Found during:** Task 1 (GREEN iteration)
- **Issue:** `runBashVerbatim` SIGKILLs the block's whole process group after bash exits (the 07-01 pipe-deadlock/leak fix) — the pair-start block's two nodes died at block end, so the Discover block could not dial 127.0.0.1:28011
- **Fix:** Added a `blockSurvivors` parameter (named type — the code-direction answer to revive flag-parameter): `reapSurvivors` (default; README path byte-identical) and `keepSurvivors` for the pair-start block only, suppressed solely on successful bash exit; plus a PID-file t.Cleanup backstop in TestExamplesDoc that kills the pair if any block fatals before the stop block
- **Files modified:** cmd/sup-kv/examples_test.go, cmd/sup-kv/readme_test.go
- **Verification:** TestExamplesDoc green; no orphan processes after standalone, combined, and full-suite runs (pgrep clean); TestReadmeQuickstart still green
- **Committed in:** 4cc72ed (part of Task 1 GREEN)

**2. [Rule 1 - Bug] Assertion substrings did not match the real grpcurl bytes**
- **Found during:** Task 1 (GREEN iteration — the real-run capture)
- **Issue:** The plan's prose assertion literals (`"acked":true`, `acked_count`, `total_peers`) do not match what grpcurl prints: pretty-printed JSON with a space after the colon and camelCase protojson names (`ackedCount`, `totalPeers`)
- **Fix:** Ran the exact walkthrough sequence against the built binary first, captured the real outputs, pasted them into the doc prose, and keyed both the block greps and the table's expect substrings to the captured bytes (Pitfall 3/5: outputs generated by running the commands)
- **Files modified:** cmd/sup-kv/examples_test.go, docs/EXAMPLES.md
- **Verification:** TestExamplesDoc green (the wait block asserts `"acked": true` + `ackedCount` + `totalPeers`)
- **Committed in:** c36f983 (table intent) / 4cc72ed (real-byte correction, both in Task 1's cycle)

**3. [Rule 3 - Blocking] Strict-lint findings in the new/changed test files**
- **Found during:** Tasks 1-2 (lint gate)
- **Issue:** QF1001 (De Morgan on the reaper condition), revive flag-parameter (bool control-coupling parameter), govet shadow (cleanup closure), modernize stringsseq (Split → SplitSeq)
- **Fix:** Code direction throughout: De Morgan form, the named `blockSurvivors` type, renamed closure variables, `strings.SplitSeq` — zero nolint
- **Files modified:** cmd/sup-kv/examples_test.go, cmd/sup-kv/operations_test.go
- **Verification:** `go tool golangci-lint run ./cmd/sup-kv/...` and `make lint` green
- **Committed in:** 4cc72ed / 19be25b (part of the task GREEN commits)

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All three were required to make the walkthrough's shared-pair design work under the existing harness guarantees and the strict gate; no scope creep — the reaper change preserves the 07-01 leak guarantee everywhere except the one block whose survivors are the design.

## Flagged Assumptions Surfaced (per plan must_haves)

- **OPERATIONS.md content truth (WINDOWS id 7, unmet-truth):** TestOperationsDoc is a presence/anchor guard only — no automated check establishes that the five statements are TRUE of the shipped binary; content truth is 07-VALIDATION.md's manual UAT table.
- **Examples concurrency/interruption residual (WINDOWS id 8, deviation):** blocks run sequentially sharing the pair on the reserved 280xx band; a mid-walk test abort is backstopped by the PID-file t.Cleanup, but a hard runner cancel (SIGKILL to the go test process) between blocks skips cleanup and leaks the pair — surfaced for phase verification, not silently assumed safe.

## Issues Encountered

None beyond the deviations above.

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. Every example is CI-executed with asserted output; OPERATIONS.md statements are sourced from code constants and the recorded phase-6 contracts.

## Threat Flags

None. T-7-04's mitigation is delivered as planned: the trust model is the FIRST section (machine-asserted), no wording claims auth/encryption/tamper-protection, examples bind localhost only, every example is CI-executed, and the reflection disclosure sits beside the trust model.

## Next Phase Readiness

- **07-04 (release) unblocked:** README's `docs/OPERATIONS.md` link now resolves; EXAMPLES/OPERATIONS are complete and drift-guarded
- **Phase verification:** DOCS-02 truth is manual UAT (WINDOWS id 7); the examples darwin legs (WINDOWS ids 5-6-8 family) need the next macos-15 gate run to show green-with-skip
- No open blockers from this plan

## Self-Check: PASSED

All seven key files exist on disk; all six task commits (c36f983, a842229, 4cc72ed, ad8bb9a, e4fe92a, 19be25b) are ancestors of HEAD; evaluation-scope resolved via plan-subjects with the plan's commits on this branch; acceptance criteria re-verified (10 examples-verify markers ≥ 7; every titled block present incl. wait/watch/snapshot/stop; wait block asserts "acked": true + non-zero ackedCount/totalPeers; broadcast form is the option-b literal 127.255.255.255; TestExamplesDoc green standalone, combined, and inside the full -race package suite; OPERATIONS.md greps — ## Trust model, 61440, ## Clock drift runbook, read-your-writes — all present; trust model is the first ## section; make lint and make test green; no orphan processes).

---
*Phase: 07-v1-0-docs-examples-and-release*
*Completed: 2026-11-17*
