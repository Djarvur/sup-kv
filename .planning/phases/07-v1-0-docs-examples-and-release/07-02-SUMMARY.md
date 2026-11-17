---
phase: 07-v1-0-docs-examples-and-release
plan: 02
subsystem: docs
tags: [readme, quickstart, docs-as-test, grpcurl, udp-broadcast, tdd, option-b]

# Dependency graph
requires:
  - phase: 05-grpc-api-and-binary
    provides: the shipped sup-kv binary and its flag surface documented by the quickstart
  - phase: 06-hardening-failure-scenarios-observability-benchmarks
    provides: the exec kill -9/restart/convergence evidence the quickstart re-proves in the README's own words
  - phase: 07-v1-0-docs-examples-and-release
    provides: plan 07-01 — the executable-doc harness (extractMarkedBlocks/runBashVerbatim/supkvBinDir/grpcurlShim), the doc-port band [27900, 28100), and the option-b broadcast verdict that shaped the block
provides:
  - TestReadmeQuickstart — the DOCS-01 machine contract (README.md is the test input; the quickstart cannot drift from reality without a red gate)
  - Rewritten README.md per D-02/D-03 — verified quickstart (Linux block + marked darwin variant per option-b), docs links, honest v1.0 status, no ops detail inline
  - The quickstart-verify marker/fence contract frozen between README.md and CI (D-06, reversibility: costly)
affects: [07-03 (docs expansion — EXAMPLES/OPERATIONS link targets), 07-04 (release — Status wording flips when the tag lands), 07-05]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 3943    # chars/4 over git diff 601a250..0a0c255 (README + test + evidence record; no dep churn)
  tasks: 2
  commits: 3      # MEASURED: git rev-list --count 601a250..HEAD (docs commit follows this write)

plan_head_before: 601a250545dca927810de8221bee680987e9108d
plan_head_after: 0a0c25535a5e88df3a37ef5f64e5567d0f82d46b

# Tech tracking
tech-stack:
  added: []       # no new libraries/tools — the plan consumes the 07-01 harness and tool directives
  patterns:
    - "README-as-test: the quickstart-verify marker selects fenced bash blocks that CI executes verbatim — the same doc-is-the-test-input contract as examples-verify, now frozen for the repo-root README (D-06 reversibility: costly)"
    - "Hard-assert retry loops: a bounded if-form retry followed by one unconditional command — the tolerant loop provides the wait, the final command makes exhaustion fatal under bash -e and prints the proof"

key-files:
  created:
    - cmd/sup-kv/readme_test.go
    - .planning/phases/07-v1-0-docs-examples-and-release/tdd-red-07-02.json
  modified:
    - README.md

key-decisions:
  - "Option-b README shape landed as recorded: both marked blocks (the Linux block with the literal 127.255.255.255, and the darwin variant with the BCAST derive line) carry the quickstart-verify marker, so CI executes BOTH on Linux — the darwin variant runs through its || echo fallback arm there, which keeps the variant block itself from rotting while its macOS broadcast form stays documented-not-CI-verified"
  - "TestReadmeQuickstart's darwin leg skips deterministically before any block runs (readmeSkipDarwin, quickstartBroadcastOption constant) rather than calling probeSendBroadcast live — the 07-01 close-out decision (64a3cbb): determinism of the macos-15 gate cell beats re-gathering evidence the recorded verdict already carries"
  - "requirements-completed is empty: DOCS-01 stays unchecked because D-06/D-07 are PARTIALLY satisfied (macos-15 leg is a named skip under option-b) — the partial satisfaction must be adjudicated at phase verification, where option-c (requirement rewording) remains available; the 01-05/01-06/07-01 declared-but-not-fully-delivered precedent"
  - "The pre-flight marker check (t.Fatalf naming the DOCS-01 contract) sits ahead of extractMarkedBlocks so the RED fatal names DOCS-01 exactly as the plan's behavior case words it; the shared helper still guards the unclosed-fence class"

patterns-established:
  - "Quickstart block shape for executable docs: background jobs with captured PIDs, fixed doc-band ports, relative dump paths, bounded if-form retries + one unconditional hard-assert command, cleanup line that kills and waits — every bounded wait in a marked block ends in a command whose failure is fatal under bash -e"

requirements-completed: []  # DOCS-01 deliberately NOT marked — see key-decisions; adjudication at phase verification

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "The README quickstart executes verbatim (bash -e) against the built binary on Linux: two nodes on one broadcast segment, Set on A visible via Get on B, kill -9 of A, restart with identical args, Get on A, cleanup — D-06/D-07 in the README's own words, with grpcurl as the first client (D-08)"
    requirement: DOCS-01
    verification:
      - kind: integration
        ref: "cmd/sup-kv/readme_test.go#TestReadmeQuickstart (5.4s standalone; green with TestExamplesDoc; green in the full -race package suite)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Option-b darwin handling: TestReadmeQuickstart's darwin leg loud-skips naming the probe and the recorded macos-15 verdict; the README ships the marked darwin variant block labeled documentation-only with the honest sendto reason"
    requirement: DOCS-01
    verification:
      - kind: unit
        ref: "cmd/sup-kv/readme_test.go#readmeSkipDarwin (darwin-only path; not executable on this linux host)"
        status: unknown
    human_judgment: true
    rationale: "The darwin skip path and the macos-15 gate cell's green-with-skip can only be confirmed on the next PR gate run (this executor ran on linux/amd64); D-06/D-07 partial satisfaction under option-b needs human sign-off at phase verification — the same adjudication WINDOWS.md id 5 carries"
  - id: D3
    description: "D-02 README shape: what sup-kv is, the verified quickstart, links to docs/OPERATIONS.md and docs/EXAMPLES.md, docs/IDEA.md design pointer, MIT license — zero ops detail inline, 'Design phase' wording gone, honest pending-release status"
    verification:
      - kind: other
        ref: "TestReadmeQuickstart link assertions (docs/OPERATIONS.md, docs/EXAMPLES.md) + grep acceptance set (marker, kill -9 $NODE_A, Design-phase count 0)"
        status: pass
    human_judgment: true
    rationale: "Presence is machine-proven, but 'no ops detail inline' and the honesty of the status/variant wording are qualitative doc judgments for the phase verifier"

# Metrics
duration: 12min
completed: 2026-11-17
status: complete
---

# Phase 7 Plan 2: README Quickstart Machine-Verification Summary

**TestReadmeQuickstart turns README.md into the CI test input (RED then GREEN): the option-b quickstart — a CI-verified Linux block plus a marked darwin variant — runs verbatim through bash -e including kill -9/restart/convergence, and the README is rewritten to D-02 shape**

## Performance

- **Duration:** 12 min (2026-11-17 12:48–13:00 UTC)
- **Started:** 2026-11-17T12:48:52Z
- **Completed:** 2026-11-17T13:00:32Z
- **Tasks:** 2 (both TDD: RED then GREEN)
- **Files modified:** 2 code/doc files + 1 planning artifact

## TDD Gate Compliance

RED `7eb65ce` (TestReadmeQuickstart failing on the unmodified design-phase README — the zero-block fatal naming the DOCS-01 contract) → GREEN `0a0c255` (README rewritten; the marked blocks execute verbatim). RED evidence captured from the actual run, classified `RED_EVIDENCE_OK` (reason: `target_test_failed`) by `gsd_run check tdd-red-evidence`, and committed as `6ab66fa` BEFORE the GREEN commit. RED → GREEN ancestry verified (`git log --grep` gates both present, correct order). No REFACTOR needed: the GREEN code is the reused 07-01 harness plus the README; nothing to clean without behavior change.

## Accomplishments

- The README quickstart is now a machine contract (D-06): `<!-- quickstart-verify -->` selects fenced blocks that CI extracts from README.md itself and executes verbatim — zero marked blocks is a test failure, so the contract cannot silently disappear
- The full DOCS-01 story runs in the README's own words (D-07): two nodes on one broadcast segment (UDP 27901/27902, gRPC 27911/27912 — the reserved doc band), Set on A, bounded-retry Get on B, `kill -9 $NODE_A`, restart with identical arguments, bounded-retry Get on A, cleanup — grpcurl first (D-08), default timers, relative dump paths
- Option-b landed exactly as the 07-01 verdict recorded: the Linux block carries the literal `127.255.255.255`, the darwin variant block carries the `BCAST=$(ipconfig getoption en0 broadcast 2>/dev/null || echo 127.255.255.255)` derive line, and the variant is labeled documentation-only for macOS with the honest sendto reason
- README rewritten to D-02 shape: product statement preserved, Status honest (v1.0 feature-complete, release pending — no tag claim), docs links, IDEA.md pointer, license; zero ops detail inline; `grep -c 'Design phase'` = 0

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): failing TestReadmeQuickstart** - `7eb65ce` (test)
2. **TDD RED evidence record** - `6ab66fa` (docs)
3. **Task 2 (GREEN): README rewrite** - `0a0c255` (feat)

**Plan metadata:** this commit (docs: complete plan).

## Files Created/Modified

- `cmd/sup-kv/readme_test.go` - TestReadmeQuickstart: marker pre-flight fatal, option-b darwin gate (deterministic loud skip), verbatim execution of every marked block via the 07-01 harness, c3VwLWt2 output assertion, D-02 link assertions
- `README.md` - rewritten per D-02: verified quickstart (Linux block + marked darwin variant), docs links, honest v1.0 status, highlights refresh
- `.planning/phases/07-v1-0-docs-examples-and-release/tdd-red-07-02.json` - RED evidence record (RED_EVIDENCE_OK)

## Decisions Made

- Both marked blocks carry the same quickstart-verify marker so CI executes both on Linux (the darwin variant via its fallback arm) — neither block can rot; the macOS broadcast form of the variant stays documented-not-CI-verified, exactly the option-b scope
- Deterministic darwin skip before any probing/block execution (the 64a3cbb precedent) — the gate encodes the recorded verdict as the `quickstartBroadcastOption` constant with the flip condition documented
- Every bounded retry loop in the marked blocks ends with one unconditional command (the final grpcurl Get): the loop provides the wait, the final command makes exhaustion fatal under bash -e and prints the proof — a silent-exhaustion loop can never pass
- DOCS-01 not marked complete in REQUIREMENTS.md (partial D-06/D-07 satisfaction pending phase-verification adjudication)

## Deviations from Plan

None - plan executed exactly as written.

## Flagged Assumptions Surfaced (per plan must_haves)

- **Orphan-process cleanup on MID-BLOCK interruption (carried, unresolved by design):** parallel-suite and interruption guarantees rest on the reserved doc-port band plus bash -e failing the block on any step failure; orphan-process cleanup when a CI run is cancelled between node start and the cleanup line is not separately handled — `runBashVerbatim`'s process-group SIGKILL covers the post-exit case, not a hard runner cancel. Surfaced for phase verification, not silently assumed safe.

## Issues Encountered

None.

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. No placeholder or unwired data paths exist; the darwin variant block is documentation by recorded decision (option-b), not a stub.

## Next Phase Readiness

- **07-03 unblocked:** EXAMPLES.md expansion and OPERATIONS.md authoring; the README already links both docs (OPERATIONS.md does not exist until 07-03 lands it — same-phase sequencing, resolved before release)
- **07-04 note:** the README Status section says "release artifacts are in preparation; until the tag lands, build from source" — flip that wording when the tag lands
- **Phase verification must adjudicate:** the D-06/D-07 partial satisfaction (macos-15 named skip) — WINDOWS.md ids 5 (probe) and 6 (readme skip) both block /gsd-ship on it; option-c (requirement rewording) remains available there
- The next PR gate run must show the macos-15 cell green-with-skip for TestReadmeQuickstart (the skip path is darwin-only; this executor verified linux/amd64 only)

## Self-Check: PASSED

All three key files exist on disk; all three task commits (7eb65ce, 6ab66fa, 0a0c255) are ancestors of HEAD; evaluation-scope resolved with the plan's commits on this branch; acceptance criteria re-verified (TestReadmeQuickstart green standalone/with TestExamplesDoc/in the full -race suite; marker + both doc links + kill -9 line present; Design-phase count 0; option-b broadcast forms: 3 literal loopback in the Linux block, derive line + 3 $BCAST forms in the darwin variant; lint clean on ./cmd/sup-kv/...).

---
*Phase: 07-v1-0-docs-examples-and-release*
*Completed: 2026-11-17*
