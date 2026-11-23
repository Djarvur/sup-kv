---
phase: 07-observability-and-configuration
plan: 11
subsystem: observability
tags: [phase-gate, frozen-red-evidence, mutation-testing, lint, buf-breaking, darwin-cross, client-example]
requires:
  - phase: 07-observability-and-configuration
    provides: plans 07-01…07-10 — the config extraction, udprepl split, metric semantics and lifecycle edits this gate verifies
  - phase: 06-api-and-semantics
    provides: the pinned-hash phase-gate shape this plan repeats, and the published proto that makes buf-breaking a real comparison
provides:
  - a recorded Phase-7 gate result — three frozen content hashes, both mutation sed targets, red-runner 7/7 with both mutants killed, make lint/test/buf-lint/buf-breaking/client-example green, darwin build/vet green, unchanged vendored lint body and direct runtime require block
affects: [phase-completion, 08-performance]
actuals:
  tokens: 3923
  tasks: 2
  commits: 0
plan_head_before: 1c7c871697df79cf4a71384a294caf1a59f57505
plan_head_after: 1c7c871697df79cf4a71384a294caf1a59f57505
tech-stack:
  added: []
  patterns:
    - "Phase gate: content hashes (never refs) for frozen evidence, literal shape greps for mutation seds, then the unmodified standing gates once on the committed tree"
key-files:
  created:
    - .planning/phases/07-observability-and-configuration/07-11-SUMMARY.md
  modified:
    - .planning/phases/07-observability-and-configuration/deferred-items.md
key-decisions:
  - "The first full red-runner attempt failed on an ordinary stand test (TestStand_persistLoadUnusableSingle: 'SIGKILL <pid>: os: process already finished') while all seven expectation rows were green; the full gate re-ran unchanged on the same tree and passed, so the failure is recorded as an environment/harness observation and deferred, never narrowed around"
  - "buf-breaking ran the real FILE-policy comparison this time — supkv.proto is now on origin/master (d6b2a75), so the D-08 deferral guard took the comparison branch and passed"
  - "This gate plan makes no task commit: it changes no source and every gate passed on the committed tree, so committing a no-op would fabricate history (the 01-04 precedent); the record lives in this SUMMARY"
  - "The three frozen pins are content hashes captured at phase start, so they are compared with git hash-object on the current tree and no ref or merge-base participates"
patterns-established:
  - "Pattern: a transient phase-gate failure is investigated from the recorded evidence, then the full unmodified gate is re-run (node_repair RETRY) with both attempts recorded"
requirements-completed: [OBSV-01, OBSV-02, OBSV-03, OBSV-04, OBSV-05, OBSV-06, OBSV-07, OBSV-08]
coverage:
  - id: D1
    description: "The three frozen red-evidence paths hash to their planning-time literals"
    requirement: "OBSV-01…OBSV-08"
    verification:
      - kind: contract
        ref: "git hash-object: red_gate=519789bd1f7abb2af09053b81c26e0ce45d9bdaf, stand_red=573fed68d2f1a039b8ac023a342446103064273b, tsv=add45b810d36d69fd574ca83cb575dab5722a55b"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both mutation sed targets still exist and the full red-runner is 7/7 with both mutants killed and the control green"
    requirement: "OBSV-01…OBSV-08"
    verification:
      - kind: contract
        ref: "grep -F targets: internal/app/repl/core.go:2510 'if !c.nextSync.IsZero() && now.Before(c.nextSync) {'; internal/app/storage/store.go:253,274 'out[key] = rec.Version'"
        status: pass
      - kind: integration
        ref: "bash scripts/red-runner -> exit 0; 'red-runner: 7 expectation row(s) checked, 0 error(s)'; control 4/4 pass; both mutants killed on seed 848612892191557850; 'mutation stage green'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The standing gates are green on the final tree: make lint, make test, make buf-lint, make buf-breaking and make client-example"
    requirement: "OBSV-01…OBSV-08"
    verification:
      - kind: contract
        ref: "make lint -> config verify + golangci-lint run ./... exit 0 (3.47 s); make test -> go test -race -cover ./... exit 0 in 5m54.668s, no FAIL; make buf-lint exit 0; make buf-breaking exit 0 with the real comparison against refs/remotes/origin/master; make client-example exit 0 (build+vet+tidy+diff)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The darwin cross build and vet pass"
    requirement: "OBSV-01…OBSV-08"
    verification:
      - kind: contract
        ref: "GOOS=darwin go build ./... -> DARWIN_BUILD_OK; GOOS=darwin go vet ./... -> DARWIN_VET_OK"
        status: pass
    human_judgment: false
  - id: D5
    description: "The vendored lint body and the direct runtime require set are unchanged"
    requirement: "OBSV-07"
    verification:
      - kind: contract
        ref: "sed -n '1,64p' .golangci.yml | git hash-object --stdin = a121e0466bc211e48ca898f6c07434dc50eba07c; awk first require block | git hash-object --stdin = 9679f1431d1815116871a4f7af79e727283a2875 (grpc v1.85.0 + protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The three doc-anchor test families are green"
    requirement: "OBSV-01…OBSV-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestOperationsDoc|TestAPIDoc|TestExamplesDoc' -> TestAPIDoc, TestExamplesDoc (5.91s), TestOperationsDoc, TestOperationsDocFamilies PASS; ok 7.663s"
        status: pass
    human_judgment: false
duration: 18min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 11: Phase Gate Summary

The Phase-7 gate is green on the committed tree: the frozen red evidence still hashes to its planning-time bytes, both mutation seds still bite (7/7 rows, both mutants killed), and lint, test, buf-lint, the real buf-breaking comparison, the nested client module and the darwin cross build/vet all pass with the vendored lint body and the direct runtime requires unchanged.

## Performance

- **Duration:** 18min (1105 s)
- **Started:** 2026-11-23T11:53:32Z
- **Completed:** 2026-11-23T12:11:57Z
- **Tasks:** 2
- **Files modified:** 0 source files (this gate plan changes no source; only `.planning/` artifacts)

## Accomplishments

- **Frozen content pins match.** `git hash-object` (no ref, no merge-base) returns `red_gate=519789bd1f7abb2af09053b81c26e0ce45d9bdaf`, `stand_red=573fed68d2f1a039b8ac023a342446103064273b` and `tsv=add45b810d36d69fd574ca83cb575dab5722a55b` — byte-identical to the planning-time literals. No frozen test and no expectation row was edited.
- **Both mutation targets still exist, un-re-anchored.** The periodic-sync gate line is at `internal/app/repl/core.go:2510` and the tombstone write at `internal/app/storage/store.go:253,274`; `scripts/red-runner` is untouched (no diff), so Phase-7's core/storage edits preserved both shapes.
- **Full red-runner 7/7 with mutants.** `bash scripts/red-runner` (no flags) exits 0: every one of the seven green rows reports `observed=pass,run`, the control arm passes all four corpus seeds, and both mutants are killed on seed 848612892191557850 (`sync-without-tombstones`, `periodic-sync-disabled`), ending with `mutation stage green`. Evidence: `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261123T120147Z/` (untracked, as designed).
- **Standing gates green.** `make lint` (config verify + `go tool golangci-lint run ./...`, 3.47 s) exit 0; `make test` (`go test -race -cover ./...`) exit 0 in 5m54.668s with no FAIL line in any package; `make buf-lint` exit 0; `make buf-breaking` exit 0 — and this time it ran the **real** FILE-policy comparison, because `refs/remotes/origin/master` (`d6b2a75`) now carries `internal/udprepl/replproto/supkv/v1/supkv.proto`, so the D-08 deferral guard took the comparison branch; `make client-example` exit 0 (build, vet, `go mod tidy`, `git diff --exit-code` on the nested module's go.mod/go.sum).
- **Darwin cross gate green.** `GOOS=darwin go build ./...` and `GOOS=darwin go vet ./...` both succeeded.
- **Frozen configuration pins hold.** `.golangci.yml` lines 1-64 hash to `a121e0466bc211e48ca898f6c07434dc50eba07c`; the first `require ( … )` block of `go.mod` hashes to `9679f1431d1815116871a4f7af79e727283a2875` (still exactly `google.golang.org/grpc v1.85.0` and `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e`) — no direct runtime dependency was added in Phase 7.
- **Doc anchors green.** `TestOperationsDoc`, `TestOperationsDocFamilies`, `TestAPIDoc` and `TestExamplesDoc` (5.91 s) all pass.
- **The production tree is untouched by this plan.** `git diff 1c7c871 -- . ':!.planning'` is empty; the only untracked paths are the two red-evidence directories the runner created.

## Task Commits

This gate plan produced **no task commits**: `files_modified: []`, no source edit, and every gate passed on the committed tree. Committing a no-op would fabricate history (the Phase-01 plan 01-04 precedent); the green gate record is this SUMMARY.

- Plan head before: `1c7c871697df79cf4a71384a294caf1a59f57505`
- Plan head at SUMMARY time: `1c7c871697df79cf4a71384a294caf1a59f57505` (0 commits between)
- The single metadata commit (`docs(07-11): complete phase-gate plan`) follows this file and carries SUMMARY.md, STATE.md, ROADMAP.md, REQUIREMENTS.md and the deferred-items row.

## Files Created/Modified

- `.planning/phases/07-observability-and-configuration/07-11-SUMMARY.md` — this gate record.
- `.planning/phases/07-observability-and-configuration/deferred-items.md` — one row for the transient stand-harness failure observed on the first runner attempt (out-of-scope for a gate plan; Phase-8 flake owner).

No source, test, proto, lint-config, Makefile or `go.mod`/`go.sum` file was modified.

## Decisions Made

- The first red-runner attempt's failure was treated as an infrastructure observation, not a phase regression: all seven expectation rows were green in that same run, the failing test is an ordinary (un-tabled) stand test, the plan changes no source, and the full gate passed unchanged on the immediate re-run. The gate was re-run whole (no flags, no narrowed package list, no `--skip-mutants`) per `workflow.node_repair` RETRY; both attempts are recorded below.
- `buf-breaking` was not "deferred" this gate: the proto now exists on `origin/master`, so the D-08 guard executed the real wire comparison. That is a stronger result than the deferral the plan's wording anticipated, and it was left to run as written.
- Zero task commits, because a gate plan that changes no file and passes every gate has nothing to commit; the evidence and timings live in this SUMMARY (and the untracked runner evidence).

## Deviations from Plan

**1. [Rule 3 - Blocking environment] Re-ran the red-runner after a transient stand-harness failure**
- **Found during:** Task 1 (`bash scripts/red-runner`, first attempt)
- **Issue:** The runner exited 1 with one error: `TestStand_persistLoadUnusableSingle: unexpected failing test — not in the expectation table [github.com/Djarvur/sup-kv/internal/stand]`. The test itself reported `persist_test.go:297: SIGKILL 2116885: os: process already finished` after 0.16 s (the call site is `seedPersistDump` → `killAndReap`). All seven expectation rows in the same run were OK (`pass,run`), and every other package/test passed. The child node exited between the confirmed `Set` and the harness `Process.Kill`; `startStandNode` captures the child's stderr into a buffer that this path never surfaces, and `/proc/vmstat` `oom_kill` was 0, so no cause could be attributed from the run itself. The host was under the documented external load (foreign `bfs` scan at ~289 % CPU, `acpfront.test` from another project, load average 11-12 on 16 cores).
- **Fix:** No source or test change. Per the plan's verification-failure gate and `workflow.node_repair: true`, the full runner was re-run unchanged (no flags, no `--skip-mutants`, no narrowed package list, no frozen-file or TSV edit). Attempt 2 was green: 7/7 rows, 0 errors, both mutants killed, mutation stage green.
- **Files modified:** `.planning/phases/07-observability-and-configuration/deferred-items.md` (the observation, for the Phase-8 flake owner)
- **Verification:** Attempt 1 evidence `red-evidence/20261123T115557Z/`; attempt 2 evidence `red-evidence/20261123T120147Z/` (`summary.txt` shows every row and both mutants OK)
- **Commit:** none (gate plan; recorded in this SUMMARY and in the deferred-items row)

**2. [Rule 2 - Metadata] Repaired the known ROADMAP v1.0 Phase-7 row collision**
- **Found during:** Close-out (`gsd_run query roadmap.update-plan-progress 07`)
- **Issue:** The update reported `updated: true, summary_count: 11` but wrote the v1.1 plan count into the **v1.0** row (`| 7. v1.0 — Docs, Examples and Release | v1.0 | 10/11 | In Progress | |` — accumulated from earlier plans too), leaving the v1.1 row at `6/11`. The ROADMAP's own checklist records v1.0 Phase 7 as `5/5 plans — completed 2026-11-18`.
- **Fix:** Restored the v1.0 row to `5/5 | Complete | 2026-11-18` and set the v1.1 row to `11/11 | In Progress` (plan progress complete; the phase checkbox stays open for the orchestrator's verification step). No other roadmap line changed.
- **Files modified:** `.planning/ROADMAP.md`
- **Verification:** `git diff 1c7c871 -- .planning/ROADMAP.md` shows exactly three hunks: the phase's `11/11 plans executed` line, the `07-11-PLAN.md` checkbox, and the two progress rows.
- **Commit:** none (part of the metadata commit that follows)

No plan prohibition was crossed: no frozen file, expectation row or pinned literal was edited, no `//nolint` was added, no gate was narrowed, and the runner ran without `--skip-mutants` both times.

## Issues Encountered

- **Red-runner attempt 1 — one ordinary stand test failed on a SIGKILL race (not reproducible):** details in Deviation 1. The failure mode is a harness diagnostic gap worth a Phase-8 look: `killAndReap` treats `os: process already finished` as fatal without naming the child's exit status or the stderr already captured by `startStandNode`/`startStandLoggedNode`. Logged to `deferred-items.md`; not fixed here because a gate plan changes no source.
- **Host load:** the known external load (foreign `bfs` scan, another project's race suite, renderers) was present throughout; nevertheless `make test` was green on its first invocation (5m54.668s) and neither `TestCore_unackedCostFlat` nor `TestCore_keyCountCostFlat` failed in it. The two previously recorded ratio-gate rows therefore needed no retry this time.

## User Setup Required

None - no external service configuration required.

## Threat Flags

None — this plan adds no source, endpoint, auth path or schema change; it only records results of existing gates.

## Next Phase Readiness

Phase 7 is complete: all eleven plans have summaries, the phase gate is green on the committed tree, and OBSV-01…OBSV-08 are ready for the requirement close-out. Ready for phase completion (review/verification). Phase 8 (performance) still owns the two wall-clock ratio gates recorded in `deferred-items.md` plus the new stand-harness SIGKILL-race observation.

## Self-Check: PASSED

- Every gate was executed as written and its observed output is recorded above (hashes, greps, runner line, make targets, darwin, doc anchors).
- No source file changed: `git diff 1c7c871 -- . ':!.planning'` is empty; the tree carries only the two untracked red-evidence directories.
- `07-11-SUMMARY.md` exists at the expected path; plan_head_before equals the recorded base and the plan produced 0 commits before this file, as a no-source gate plan must.
