---
phase: 07-v1-0-docs-examples-and-release
plan: 01
subsystem: docs
tags: [grpc, grpcurl, reflection, go-mod-tool-directives, docs-as-test, udp-broadcast, tdd]

# Dependency graph
requires:
  - phase: 05-grpc-api-and-binary
    provides: supkvapi.v1.SupKVService gRPC surface and the shipped sup-kv binary
  - phase: 06-hardening-failure-scenarios-observability-benchmarks
    provides: scenario harness (private-window UDP ports, probeLoopbackSend shape), the D-12 loud-skip precedent
provides:
  - gRPC reflection live on the shipped binary (grpcurl enumerates the API with zero proto files)
  - go.mod tool directives pinning goreleaser v2.20.2 and grpcurl v1.9.4 (zero runtime require changes)
  - doc-port band [27900, 28100) reserved via scenarioPortLo 28100 (doc-test binds never collide with scenario ports)
  - executable-doc harness in package main_test (docBlock/extractMarkedBlocks/runBashVerbatim/supkvBinDir/grpcurlShim)
  - docs/EXAMPLES.md seed with one CI-executed verbatim block (start a node, grpcurl list, Set/Get round-trip)
  - the macos-15 broadcast probe verdict and the option-b README quickstart topology decision
  - the darwin loud-skip contract for the probe (macos-15 gate cell green with the named skip)
affects: [07-02 (README quickstart — unblocked by the option-b verdict), 07-03 (examples expansion), 07-04 (release, goreleaser directive), 07-05]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 58218   # chars/4 over git diff 31bd810..64a3cbb (go.mod/go.sum dominated by the tool-directive graph)
  tasks: 3
  commits: 5      # MEASURED: git rev-list --count 31bd810..HEAD (docs commit follows this write)

plan_head_before: 31bd810cc180f0aebfcc38271f13303e480d18b8
plan_head_after: 64a3cbb23d4e83e2c92f65f9f0514ee0c34b0ba0

# Tech tracking
tech-stack:
  added:
    - "go.mod tool directive: github.com/fullstorydev/grpcurl/cmd/grpcurl v1.9.4 (dev tool, PATH-shimmed in tests)"
    - "go.mod tool directive: github.com/goreleaser/goreleaser/v2 v2.20.2 (dev tool, consumed by 07-04)"
    - "google.golang.org/grpc/reflection (import of the already-required grpc module — zero new modules)"
  patterns:
    - "doc-as-test: EXAMPLES.md fenced blocks behind a <!-- examples-verify --> marker are extracted and executed verbatim via bash -e against the real binary (the doc is the test input)"
    - "PATH shim over go tool directives: a generated /bin/sh script exec'ing `go -C <modroot> tool grpcurl` — CI and strangers never resolve an unpinned binary"

key-files:
  created:
    - cmd/sup-kv/examples_test.go
    - cmd/sup-kv/broadcast_probe_test.go
    - docs/EXAMPLES.md
  modified:
    - cmd/sup-kv/main.go
    - cmd/sup-kv/scenario_test.go
    - go.mod
    - go.sum

key-decisions:
  - "Option-b selected by the maintainer (2026-11-17, Task 3 checkpoint): the README quickstart ships a CI-verified Linux block plus a marked darwin variant block; the probe's darwin leg is a named skip carrying the recorded macos-15 verdict (phase-6 D-12/507228a precedent). D-06/D-07 are PARTIALLY satisfied — the macos-15 leg is a named skip, not a real run — recorded here and in WINDOWS.md so it surfaces at phase verification"
  - "requirements-completed is empty: DOCS-01 (README quickstart) is 07-02's deliverable and DOCS-03 (full API examples) is 07-03's; this tracer delivers their prerequisites only — the 01-05-SUMMARY precedent (declared-but-not-delivered requirements stay unchecked)"
  - "Probe send-error exhaustion on darwin is moot post-decision: the whole darwin leg loud-skips before any probe runs, so the macos-15 cell is deterministically green with the skip message"

patterns-established:
  - "Executable doc block: marker line immediately above a fenced bash block; extractMarkedBlocks picks title from the nearest ### heading; zero blocks or an unclosed fence is a t.Fatal"
  - "Loud-skip conversion after an empirical CI verdict: the first gate run fails visibly to deliver evidence, the maintainer decides, then the capability gap becomes a named skip citing the PR run (507228a shape, now applied to the quickstart probe)"

requirements-completed: []

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "gRPC reflection registered in listenGRPC — grpcurl enumerates supkvapi.v1.SupKVService with zero proto files"
    verification:
      - kind: integration
        ref: "cmd/sup-kv/examples_test.go#TestExamplesDoc (grpcurl list output contains supkvapi.v1.SupKVService)"
        status: pass
    human_judgment: false
  - id: D2
    description: "One EXAMPLES.md block executes verbatim via bash -e against the built binary with the pinned grpcurl (Set/Get round-trip asserted from captured output)"
    verification:
      - kind: integration
        ref: "cmd/sup-kv/examples_test.go#TestExamplesDoc (output contains c3VwLWt2 base64 value)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Tool directives pinned (goreleaser v2.20.2, grpcurl v1.9.4) with a clean tidy gate and zero runtime require changes"
    verification:
      - kind: other
        ref: "go mod tidy && git diff --exit-code go.mod go.sum (clean)"
        status: pass
    human_judgment: false
  - id: D4
    description: "macos-15 broadcast probe verdict recorded and the option-b topology selected; darwin leg of the probe loud-skips naming the verdict, Linux still asserts for real"
    requirement: DOCS-01
    verification:
      - kind: unit
        ref: "cmd/sup-kv/broadcast_probe_test.go#TestProbeQuickstartBroadcast (linux PASS, all three forms reached)"
        status: pass
    human_judgment: true
    rationale: "The Option A/B/C selection was a maintainer checkpoint decision over externally-produced CI evidence (PR #32 run 44083351667); the macos-15 skip message itself can only be confirmed green on the next PR gate run, and D-06/D-07 partial satisfaction needs human sign-off at phase verification"

# Metrics
duration: 23min active execution across two sessions (excludes the PR #32 gate wait between the Task 3 checkpoint and the option-b close-out)
completed: 2026-11-17
status: complete
---

# Phase 7 Plan 1: Executable-Docs Tracer and Broadcast Probe Verdict Summary

**gRPC reflection on the shipped binary plus a CI-executed verbatim EXAMPLES.md block through the pinned grpcurl — and the macos-15 probe verdict that selected option-b: a CI-verified Linux quickstart block with a marked darwin variant**

## Performance

- **Duration:** ~23 min active execution (2026-11-17 06:47–07:00 UTC executor session; 12:38–12:45 UTC option-b close-out; between them the PR #32 gate watch and the maintainer checkpoint)
- **Started:** 2026-11-17T06:47:52Z
- **Completed:** 2026-11-17T12:45:00Z
- **Tasks:** 3 (1 tracer TDD, 1 probe, 1 decision checkpoint — resolved option-b)
- **Files modified:** 8 (7 code/doc + 1 planning artifact)

## Broadcast probe verdict

**GOOS/GOARCH:** darwin/arm64 (macos-15 gate cell) and linux/amd64 (local Task 2 verify).

**Derived BCAST:** on linux `127.255.255.255` (the `|| echo` fallback arm); on darwin the `ipconfig getoption en0 broadcast` arm — the run never reached a derived-row verdict because the test failed earlier (see verbatim evidence).

**Per-form results:**

| Platform | derived (quickstart line) | loopback broadcast | limited broadcast |
|----------|---------------------------|--------------------|-------------------|
| linux/amd64 (local, Task 2 verify + this close-out) | 127.255.255.255 — reached | 127.255.255.255 — reached | 255.255.255.255 — reached |
| darwin/arm64 (macos-15, PR #32) | probe was sound (no exhaustion) but the run failed at the limited-broadcast row before a derived verdict could print | row completed without exhaustion; phase 6 already proved lo0 carries no BROADCAST flag, so loopback broadcast does not fan out on darwin | **sendto: no route to host** — the failure below |

**Verdict evidence (verbatim, macos-15 cell of PR #32 run 44083351667, job 134039317468, 2026-11-17):**

```
--- FAIL: TestProbeQuickstartBroadcast (0.32s)
broadcast_probe_test.go:155: probe of 255.255.255.255 could not get a sound setup in 5 attempts: send to 255.255.255.255:32617: write udp4 0.0.0.0:60971->255.255.255.255:32617: sendto: no route to host
```

All other PR #32 cells green: lint, gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (darwin-amd64-cross), vuln.

**Chosen option: option-b** (maintainer decision, 2026-11-17, Task 3 checkpoint):

> **MAINTAINER DECISION: option-b** — Linux block CI-verified + marked darwin variant block in the README quickstart; the probe loud-skips on darwin (the phase-6 D-12 precedent). D-06/D-07 are satisfied partially — record this honestly in SUMMARY.md; it must surface at phase verification.

**Honest scope of what this satisfies:** D-06 demanded the quickstart run for real on ubuntu-latest AND macos-15; under option-b the macos-15 leg is a **named skip, not a real run** — the darwin variant block in the README will be documented and probe-annotated, not CI-verified. Recorded in WINDOWS.md (deviation) so `/gsd-ship` surfaces it. Option-c (requirement-level rewording of DOCS-01) was NOT taken: DOCS-01's wording is owned by 07-02's freeze and phase verification, where the partial satisfaction must be adjudicated.

**Follow-up landed in this PR before merge** (commit 64a3cbb): the probe's darwin leg now `t.Skipf`s deterministically, naming the PR run, the verbatim sendto error and the option-b contract — the macos-15 gate cell goes green with the skip message. Linux behavior is byte-identical: the derive line, loopback and limited-broadcast rows still assert for real (locally all three reached).

## Accomplishments

- gRPC reflection registered in `listenGRPC` — `grpcurl -plaintext ... list` enumerates `supkvapi.v1.SupKVService` with zero proto files (T-7-01 accepted: loopback-default bind, trusted segment, public schema)
- The doc-as-test architecture proven end-to-end: `<!-- examples-verify -->` marker → extractor → verbatim `bash -e` → real binary → reflection → PATH-shimmed pinned grpcurl v1.9.4
- goreleaser v2.20.2 and grpcurl v1.9.4 pinned as go.mod tool directives (07-04 never touches go.mod); tidy gate clean; zero runtime require changes
- Doc-port band [27900, 28100) reserved (`scenarioPortLo` 21000 → 28100) so scenario port picks can never collide with documented fixed ports
- The quickstart derive line probed for real on both platforms; the macos-15 verdict recorded verbatim; option-b selected and its follow-up (darwin loud-skip) landed

## TDD Gate Compliance

Task 1 was the `tdd="true"` tracer: RED `a88b05b` (TestExamplesDoc failing — no marked block / reflection missing) → GREEN `9a536b7` (reflection + tools + seed doc + port band). RED evidence classified `RED_EVIDENCE_OK` and recorded in `tdd-red-07-01.json` (commit `8703da3`). Task 2's probe was not TDD-scoped (capability probe, no `<behavior>` block).

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): failing TestExamplesDoc + seed doc** - `a88b05b` (test)
2. **Task 1 (GREEN): reflection.Register + pinned tools + doc-port band** - `9a536b7` (feat)
3. **Task 2: macos-15 broadcast probe** - `3f5d6ac` (test)
4. **TDD RED evidence record** - `8703da3` (docs)
5. **Task 3 close-out: option-b follow-up — darwin loud-skip** - `64a3cbb` (test)

**Plan metadata:** this commit (docs: complete plan).

## Files Created/Modified

- `cmd/sup-kv/examples_test.go` - TestExamplesDoc + docBlock/extractMarkedBlocks/runBashVerbatim/supkvBinDir/grpcurlShim helpers
- `cmd/sup-kv/broadcast_probe_test.go` - TestProbeQuickstartBroadcast + probeSendBroadcast/probeBroadcastForm/deriveQuickstartBroadcast; darwin leg loud-skips (option-b)
- `cmd/sup-kv/main.go` - reflection.Register(gs) in listenGRPC + doc comment
- `cmd/sup-kv/scenario_test.go` - scenarioPortLo 28100 + doc-port band comment
- `docs/EXAMPLES.md` - seed: one CI-executed block (start node, list, Set/Get round-trip)
- `go.mod` / `go.sum` - two tool directives + their graphs
- `.planning/phases/07-v1-0-docs-examples-and-release/tdd-red-07-01.json` - RED evidence record

## Decisions Made

- Option-b (see Broadcast probe verdict) — maintainer-owned, recorded verbatim above
- requirements-completed empty (delivery honesty, 01-05 precedent) — DOCS-01/DOCS-03 are owned by 07-02/07-03+
- The darwin skip fires before any probing: determinism of the gate cell beats re-gathering evidence the verdict already recorded

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] runBashVerbatim pipe-deadlock**
- **Found during:** Task 1 (tracer harness)
- **Issue:** capturing combined output through a pipe could deadlock once the buffer filled while bash waited on the test
- **Fix:** file-based capture plus process-group SIGKILL after bash exits
- **Files modified:** cmd/sup-kv/examples_test.go
- **Verification:** TestExamplesDoc green across repeated runs
- **Committed in:** 9a536b7 (part of Task 1 GREEN)

**2. [Rule 1 - Bug] startup-gate wait polled reflection instead of health**
- **Found during:** Task 1 (seed doc block)
- **Issue:** reflection answers before the startup gate opens, so a `grpcurl list` success did not mean the node was serving
- **Fix:** the gate wait polls `grpc.health.v1.Health/Check` for "SERVING"
- **Files modified:** cmd/sup-kv/examples_test.go, docs/EXAMPLES.md
- **Verification:** TestExamplesDoc asserts the Set/Get round-trip, which only works post-gate
- **Committed in:** 9a536b7 (part of Task 1 GREEN)

**3. [Rule 3 - Blocking] strict-lint fixes in new test files**
- **Found during:** Tasks 1-2 (lint gate)
- **Issue:** removed unused `nolint:errcheck`, `exec.Command` → `exec.CommandContext`, inline-error restructuring
- **Files modified:** cmd/sup-kv/examples_test.go, cmd/sup-kv/broadcast_probe_test.go
- **Verification:** `go tool golangci-lint run ./cmd/sup-kv/...` green
- **Committed in:** 9a536b7 / 3f5d6ac (part of the task commits)

**4. [Rule 3 - Blocking] go directive 1.27.1 → 1.27.4**
- **Found during:** Task 1 (tool directives)
- **Issue:** goreleaser v2.20.2's module floor required go >= 1.27.2; the toolchain pin became redundant
- **Fix:** go directive bumped to 1.27.4, toolchain directive dropped (consistent with the 2026-11-15 maintainer toolchain decision closing GO-2026-6859)
- **Files modified:** go.mod
- **Verification:** tidy gate clean, full suite + lint green
- **Committed in:** 9a536b7 (part of Task 1 GREEN)

---

**Total deviations:** 4 auto-fixed (2 bugs, 2 blocking)
**Impact on plan:** All four were necessary for the tracer to prove its seams under the strict gate; no scope creep. The Task 3 darwin loud-skip was NOT a deviation — it is the option-b follow-up the plan's Task 3 prescribed ("the chosen option's follow-up... happens in this PR before merge").

## Issues Encountered

None beyond the deviations above. The macos-15 red cell was the designed probe outcome, not an issue: the plan explicitly made the first darwin gate run the Option-B/C evidence carrier.

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. docs/EXAMPLES.md is a seed by design (one block, single node) — 07-03 expands it to the two-node replication walkthrough per the plan's tracer scope; no placeholder or unwired data paths exist.

## Next Phase Readiness

- **07-02 (README quickstart) unblocked**: the contract to freeze is option-b — one CI-verified Linux block (loopback broadcast 127.255.255.255, probed working) plus a marked darwin variant block (documented, not CI-verified; the probe names the macos-15 sendto verdict). The doc-port band [27900, 28100) is reserved for its fixed ports (279xx README, 280xx EXAMPLES).
- **Surfaces at phase verification (by maintainer instruction):** D-06/D-07 are PARTIALLY satisfied — the macos-15 leg is a named skip. Option-c (rewording DOCS-01) remains available at phase verification if the partial satisfaction is rejected there; it was not preempted by this plan.
- WINDOWS.md carries the open deviation entry so `/gsd-ship` blocks on it until adjudicated.
- The orchestrator pushes this PR and re-watches the gate cells: the macos-15 cell must come back green with the skip message.

## Self-Check: PASSED

All six key files exist on disk; all five task commits (a88b05b, 9a536b7, 3f5d6ac, 8703da3, 64a3cbb) are ancestors of HEAD; acceptance criteria re-verified (reflection.Register(gs) in listenGRPC, go.mod tool directives, EXAMPLES.md marker + supkvapi.v1.SupKVService/Set, scenarioPortLo 28100, darwin loud-skip present, linux probe PASS, tidy gate clean, lint green).

---
*Phase: 07-v1-0-docs-examples-and-release*
*Completed: 2026-11-17*
