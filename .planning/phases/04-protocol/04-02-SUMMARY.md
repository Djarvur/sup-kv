---
phase: 04-protocol
plan: 02
subsystem: grpc-api-gate
tags: [proto-01, startup-gate, read-gating, listed-keys, create-if-absent, tombstone-visible, one-pointer-publication, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 01
    provides: "the complete-listing signal and the one atomic {complete, keys} mirror that Adapter.ListedKey reads"
  - phase: 05-api
    provides: "the grpcapi Server, its ServerOptions seam, the startup gate bound 1:1 to health, and the bufconn harness this plan extends"
provides:
  - "every data RPC — Get, List and Watch included — answers Unavailable while the startup gate is closed, with the gate check as the first statement"
  - "ServerOptions.ListedKey: the documented adapter listing-mirror membership seam (nil = no listing knowledge)"
  - "rule 14.5(6) read semantics: a listed-but-unarrived Get is Unavailable; a local tombstone stays NotFound; an unlisted absent key stays NotFound"
  - "create_if_absent on any listed key is AlreadyExists before the engine call; unlisted keys keep the engine's SetIfAbsent decision"
  - "a post-open race test proving completeness and membership travel in one publication"
affects:
  - "04-06 wires Adapter.ListedKey into main's ServerOptions"
  - "04-13 flips the frozen R-03 gate rows in the TSV; the grpcapi half is already green under SUPKV_RED_TESTS=1"

actuals:
  tokens: 7197
  tasks: 2
  commits: 4
  plan_head_before: fe474d0af538f22655e18636820bbe3d3fd94310
  plan_head_after: a3a212419aabda81444614de51b01b7274369cff

tech-stack:
  added: []
  patterns:
    - "The startup gate is every data handler's first statement — before validation, listing knowledge and even the nil-hub check — so a closed node's answer is always Unavailable."
    - "Tombstone-visible Engine.Version is the only presence probe on the Get error path: a local tombstone answers NotFound, a listed-but-unarrived key answers Unavailable."
    - "Listing membership is consulted only after the gate; the same atomic snapshot that reports completeness reports the keys, so a gate open cannot precede the listed set."

key-files:
  created:
    - .planning/phases/04-protocol/red-evidence/04-02-task1.json
    - .planning/phases/04-protocol/red-evidence/04-02-task2.json
  modified:
    - internal/grpcapi/gate.go
    - internal/grpcapi/server.go
    - internal/grpcapi/watch.go
    - internal/grpcapi/gate_test.go
    - internal/grpcapi/server_test.go
    - internal/grpcapi/watch_senderr_test.go

key-decisions:
  - "Get distinguishes a local tombstone from a listed-but-unarrived key with Engine.Version, never Engine.Get: Get hides tombstones, so probing it would answer Unavailable forever for a deleted listed key (RESEARCH Pitfall 10)."
  - "ListedKey is a nil-able seam: nil means no listing knowledge, which preserves the single-node configuration and every pre-existing harness while the production adapter supplies membership in 04-06."
  - "The post-open race test models the adapter contract with one atomic.Pointer publication, so the assertion is structural — completeness and membership are read from the same snapshot — rather than a race between two atomics."
  - "The Watch send-error fixture now opens its gate through the resolver and waits on the in-process health server: gating Watch made its resolver-less construction permanently cold (Rule 3 deviation)."
  - "Delete stays intentionally local-NotFound for a listed-but-unarrived key: rule 14.5(6) names reads, and no acceptance requires Unavailable for Delete (A7, recorded in 04-MULTI-NODE.md)."

patterns-established:
  - "Gate-first handlers: the gate check precedes request validation, so even an empty key is Unavailable while cold; the open side then keeps the A7 validation order."
  - "A listed-key answer is a three-way decision (present, listed-absent, unlisted-absent) derived from the tombstone-visible version probe plus the post-gate listing seam."
  - "Test harness configuration carries the listing-mirror seam (testServerConfig.listedKey) merged into ServerOptions only when non-nil, keeping the closed-gate helpers usable unchanged."

requirements-completed: [PROTO-01]

coverage:
  - id: D1
    description: "While the startup gate is closed, Get, List and Watch answer Unavailable exactly like Set/Delete/SnapshotCreate/SnapshotLoad, and health mirrors NOT_SERVING."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/grpcapi/gate_test.go#TestGate_readsAndWritesUnavailable"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/gate_test.go#TestGate_emptyKeyStillUnavailable"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/gate_test.go#TestGate_openRestoresReads"
        status: pass
    human_judgment: false
  - id: D2
    description: "ServerOptions.ListedKey exists as the documented listing-mirror membership seam (nil = no listing knowledge), and the gate/Get comments state the gate-first contract."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/grpcapi/gate_test.go#TestGate_readsAndWritesUnavailable (non-nil ListedKey under a closed gate still answers Unavailable)"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./internal/grpcapi/ (godoclint/godot over the new option field)"
        status: pass
    human_judgment: false
  - id: D3
    description: "After the gate opens, Get answers NotFound for a locally tombstoned listed key, the value for a locally live listed key, Unavailable for a listed-but-unarrived key, and NotFound for an unlisted absent key."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_getListedNotArrived"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_getListedTombstoneAndAbsent"
        status: pass
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_getListedLive"
        status: pass
    human_judgment: false
  - id: D4
    description: "create_if_absent on any listed key — locally absent or tombstoned — returns AlreadyExists before the engine is called; unlisted keys still reach SetIfAbsent."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/grpcapi/server_test.go#TestServer_createIfAbsentListed"
        status: pass
    human_judgment: false
  - id: D5
    description: "A create_if_absent issued immediately after health flips SERVING is already blocked: completeness and the listed keys come from one publication."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/grpcapi/gate_test.go#TestServer_gateOpenListedRace"
        status: pass
    human_judgment: false
  - id: D6
    description: "The plan's quality gates pass on its tree, and the frozen red-gate file is untouched."
    requirement: PROTO-01
    verification:
      - kind: other
        ref: "go test -race -count=1 ./internal/grpcapi/"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./internal/grpcapi/"
        status: pass
      - kind: other
        ref: "git diff --stat fe474d0 -- internal/grpcapi/red_gate_test.go (empty); SUPKV_RED_TESTS=1 go test -run '^TestRed_R03_gateClosedUntilFullList$'"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 02: Reads Close With the Gate and Listed Keys Get Their Honest Answer

**Every data RPC is now gated: Get, List and Watch answer Unavailable until the complete listing opens the gate, after which a listed-but-unarrived key answers Unavailable while a local tombstone stays NotFound and create_if_absent on any listed key is AlreadyExists before the engine is ever called.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-11-22T03:34:34Z
- **Completed:** 2026-11-22T03:39:26Z
- **Tasks:** 2
- **Files modified:** 8 (6 code/test files plus 2 RED evidence records)

## Accomplishments

- R-03's RPC half is closed: the interface refuses reads as well as writes while the gate is closed. `Get`, `List` and `Watch` check the gate before validation, before listing knowledge and — for `Watch` — before the nil-hub check, so a closed node can never answer from a picture it does not have.
- Rule 14.5(6) is implemented with tombstone-visible precision: the `Get` error path probes `Engine.Version` (the only presence probe there) so a locally deleted listed key is `NotFound` while a listed key whose value has not arrived is `Unavailable` with the pinned message `startup gate: key is listed by a peer but its value has not arrived`.
- F14's back door is shut at the RPC layer: `create_if_absent` for any listed key — live or tombstoned — returns `AlreadyExists` before the engine call. The test proves the bypass by asserting the fake dump store recorded no new generation.
- The gate-open instant has no window: `TestServer_gateOpenListedRace` publishes completeness and membership in one `atomic.Pointer` and issues the create immediately after health flips `SERVING`; it is already blocked.
- The frozen `internal/grpcapi/red_gate_test.go` was not edited and its R-03 acceptance now passes under `SUPKV_RED_TESTS=1`; the TSV flip remains 04-13's.

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1 RED: failing gate-read coverage** - `62e4e27` (test)
2. **Task 1 GREEN: gate every data RPC** - `d2a3b38` (feat)
3. **Task 2 RED: failing listed-key answer tests** - `dacb193` (test)
4. **Task 2 GREEN: listed reads and create_if_absent** - `a3a2124` (feat)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `internal/grpcapi/gate.go` — `ServerOptions.ListedKey` with its contract; the `gated()` comment now states that reads and writes both consult the gate
- `internal/grpcapi/server.go` — gate-first `Get`/`List`; the tombstone-visible listed-key branch in `Get`; the `create_if_absent` listed block in `Set`; `listedKeyNotArrivedMessage`; updated `Get`/`Set`/struct comments
- `internal/grpcapi/watch.go` — gate-first `Watch`, before the nil-hub check
- `internal/grpcapi/gate_test.go` — rewritten closed-gate test (`TestGate_readsAndWritesUnavailable`), empty-key precedence, open-restores-reads, and the post-open listed race
- `internal/grpcapi/server_test.go` — `testServerConfig.listedKey` seam, `newListedServer`, and the four listed-key answer tests
- `internal/grpcapi/watch_senderr_test.go` — sends the direct-handler fixture through the real resolver so its gate serves
- `.planning/phases/04-protocol/red-evidence/04-02-task1.json` / `04-02-task2.json` — persisted RED evidence

## Decisions Made

See frontmatter `key-decisions`: `Engine.Version` as the sole tombstone-visible presence probe; a nil-able `ListedKey` seam; the one-pointer race model; the gate-serving send-error fixture; and the intentional Delete asymmetry.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The Watch send-error fixture built a permanently cold server**
- **Found during:** Task 2 (full-package verification)
- **Issue:** `newSenderrServer` constructed `grpcapi.NewServer(eng, nil, hub, grpcapi.ServerOptions{})` and never launched `RunGate`. Once `Watch` became gated, its zero `StartupGate`/nil `SyncArrived` wiring left the gate closed forever, so both fake-stream subtests failed with `Unavailable` instead of reaching the Send-error branch.
- **Fix:** Construct with the existing `openGateOptions()`, launch `srv.RunGate(ctx)`, and wait on the in-process health server until `SERVING` — the same 1:1 gate/health binding the bufconn harness observes.
- **Files modified:** `internal/grpcapi/watch_senderr_test.go`
- **Verification:** `go test -race -count=1 ./internal/grpcapi/` and `go tool golangci-lint run ./internal/grpcapi/` both green; the Send-error subtests pass.
- **Committed in:** `a3a2124`

---

**Total deviations:** 1 auto-fixed (Rule 3)
**Impact on plan:** The fixture change is a direct consequence of gating `Watch`; no production behavior, bound or assertion was relaxed, and no scope was added.

## TDD Gate Compliance

Both tasks followed RED → GREEN with persisted evidence (`gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for each record):

- **Task 1:** RED `62e4e27` — all three target tests failed on the planned assertions (`Get` ungated returned `NotFound`; the empty-key check preceded the gate). GREEN `d2a3b38` made all three pass.
- **Task 2:** RED `dacb193` — the three behavior-adding targets failed on their planned assertions (`Get` listed → `NotFound`; `create_if_absent` → `OK` and an engine dump; post-open create admitted), while the tombstone/absent and live rows already passed as the preserved-behavior pins. GREEN `a3a2124` made all five pass.
- No REFACTOR commit: the implementation is already minimal, and no cleanup was warranted.

## Issues Encountered

None beyond the documented fixture deviation.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. No placeholder values, TODOs or unwired data sources were introduced; `ListedKey` is a complete seam whose production wiring is explicitly 04-06's (the plan's own scoping), and the nil value is a documented single-node configuration, not a stub.

## Threat Flags

None — no new trust-boundary surface beyond the plan's threat model. T-4-06 (tombstone-visible `Get` answer) and T-4-07 (`create_if_absent` block) are mitigated by this plan; T-4-08's membership disclosure is the accepted one.

## Next Phase Readiness

Ready for 04-03. The RPC half of PROTO-01 is in place: gated reads, the listed-key answers and the documented `ListedKey` seam are all that 04-06 needs to wire `Adapter.ListedKey` into `main`. Eleven plans remain in the phase.

---

*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Created files exist: `.planning/phases/04-protocol/04-02-SUMMARY.md`, `.planning/phases/04-protocol/red-evidence/04-02-task1.json`, `.planning/phases/04-protocol/red-evidence/04-02-task2.json`; every claimed modified code/test file exists.
- Commits exist and are ancestors of HEAD: `62e4e27`, `d2a3b38`, `dacb193`, `a3a2124`.
- The frozen `internal/grpcapi/red_gate_test.go` is byte-untouched against the pre-plan head `fe474d0`.
