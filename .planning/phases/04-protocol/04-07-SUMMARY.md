---
phase: 04-protocol
plan: 07
subsystem: replication-protocol
tags: [proto-09, three-node, real-sockets, broadcast-convergence, lww, test-only, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 01
    provides: "the DataSync v2 listing header the three production cores exchange during discovery"
  - phase: 04-protocol
    plan: 03
    provides: "the generation/install convergence point and Observe delegation the real stores exercise"
  - phase: 03-transport
    provides: "the production udprepl adapter, codec, read loop, pacers and WithAddressReuse seam"
provides:
  - "TestSmoke_threeNodeBroadcastConvergence — the N=3 real-socket proof of broadcast convergence"
  - "a documented broadcast-only limitation for shared reuse-address ports, alongside the two-port harness as the unicast owner"
affects:
  - "04-12's docs/04-MULTI-NODE.md can cite a real-socket N=3 test rather than only the simulator"
  - "the phase artifact's PROTO-09 evidence"

actuals:
  tokens: 1664
  tasks: 1
  commits: 1
  plan_head_before: 74a92817d74f1719d84083519a29a209594c720c
  plan_head_after: 21046ff01c4a6940fd692cdd73658ece1e7467f3

tech-stack:
  added: []
  patterns:
    - "N=3 real-socket proof: three production adapters, stores and cores on ONE shared wildcard port via WithAddressReuse, with the two-node smoke fixture's capability probe as the gate."
    - "Observed-state convergence waits: every visibility wait fails naming the store's actual record (absent, wrong version, or wrong data), never just a timeout."

key-files:
  created:
    - internal/udprepl/three_node_test.go
  modified: []

key-decisions:
  - "The plan's tdd=true flag cannot yield a legitimate RED: its only deliverable is the test and the asserted production behavior already exists. The target test passed on its first run; rather than fabricate a failing assertion or an unnecessary production change, the single test commit is the deliverable and the unexpected GREEN is disclosed as a process deviation."
  - "A local waitServing helper mirrors waitFor's cadence, visibilityDeadline and hard-deadline clamp but records the last observed record, because the plan requires a convergence timeout to name the per-store state instead of only the expectation."
  - "The capability probe is the existing probeBroadcastAddr/freeUDPPort/smokeStore fixture unchanged, so the loud-skip contract and the same-host socket recipe stay in one place."
  - "The same-key row waits for every store to hold A's version before B writes, so B's clock has observed A's version and B's store stamp strictly dominates — the convergence assertion therefore proves LWW rather than wall-clock luck."

requirements-completed: [PROTO-09]

coverage:
  - id: D1
    description: "Three in-process production adapters on one shared SO_REUSEADDR loopback port: discovery reaches two known peers each, a write originated by every adapter becomes byte-equal visible in both other stores, and a same-key double write converges on the later store-stamped version."
    requirement: PROTO-09
    verification:
      - kind: unit
        ref: "internal/udprepl/three_node_test.go#TestSmoke_threeNodeBroadcastConvergence"
        status: pass
      - kind: command
        ref: "go test -race -count=3 ./internal/udprepl/ -run '^TestSmoke_threeNodeBroadcastConvergence$'"
        status: pass
      - kind: command
        ref: "mutation check: with NotifyLocalWrite disabled the test failed at 5s naming 'record absent: key not found' (not committed)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The capability probe gates the test before any adapter construction and skips loudly with the probed address and platform reason on a host without same-host broadcast fan-out."
    requirement: PROTO-09
    verification:
      - kind: static
        ref: "internal/udprepl/three_node_test.go calls probeBroadcastAddr before udprepl.New; the inherited t.Skipf reasons are internal/udprepl/smoke_test.go#probeBroadcastAddr"
        status: pass
      - kind: command
        ref: "go tool golangci-lint run ./internal/udprepl/"
        status: pass
    human_judgment: false
  - id: D3
    description: "The test's doc comment states the broadcast-only limitation: unicast to a shared reuse-address port is kernel-arbitrary and remains the simulator's and the two-port real-adapter harness's responsibility."
    requirement: PROTO-09
    verification:
      - kind: static
        ref: "internal/udprepl/three_node_test.go doc comment, paragraphs 1-3"
        status: pass
    human_judgment: false
  - id: D4
    description: "No regression: the whole udprepl package stays green under -race, and the module builds and vets."
    requirement: PROTO-09
    verification:
      - kind: command
        ref: "go test -race -count=1 ./internal/udprepl/"
        status: pass
      - kind: command
        ref: "go build ./... && go vet ./internal/udprepl/"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 07: Three-Node Broadcast Convergence Summary

**PROTO-09's N=3 proof now runs on genuine loopback datagrams: three production adapters share one SO_REUSEADDR port and converge on every origin's broadcast write, including same-key LWW.**

## Performance

- **Duration:** 5min (287s)
- **Started:** 2026-11-22T04:44:30Z
- **Completed:** 2026-11-22T04:49:17Z
- **Tasks:** 1
- **Files modified:** 1 created, 0 modified

## Accomplishments

- `TestSmoke_threeNodeBroadcastConvergence` constructs three stores, three cores and three `udprepl.New(..., WithAddressReuse())` adapters over ONE shared wildcard port (`0.0.0.0:<port>`) and the probed broadcast address. The real codec, read loop, pacing and notification seam all run; no fake socket exists.
- Discovery is asserted at N=3: each adapter learns two peers from broadcast Hellos alone, with its own echo discarded.
- Every origin writes a distinct key and both other stores are polled until they serve it byte-equal through `Get`; the wait fails naming the observed state (record absent, version mismatch or data mismatch), and the mutation check confirms it fails at 5 s with `record absent: key not found` when the notification seam is disabled.
- Same-key LWW: node A's write converges everywhere, then node B writes the same key with a later store-stamped version and every store converges on B's value and version. B writes only after merging A's version, so its stamp strictly dominates.
- The capability probe (the two-node fixture's `probeBroadcastAddr`) gates the test before any adapter construction and skips loudly with the probed address and platform reason.
- The doc comment records the limitation: broadcast flows are asserted; unicast to a shared reuse-address port is kernel-arbitrary and remains covered by the simulator and the two-port real-adapter integration harness.

## Task Commits

1. **Task 1: Three-node shared-port broadcast convergence** - `21046ff` (test)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `internal/udprepl/three_node_test.go` — `TestSmoke_threeNodeBroadcastConvergence`, the `smokeNodeC` id and the observed-state `waitServing` convergence wait.

## Decisions Made

- **Unexpected GREEN is disclosed, not fabricated away.** The task is marked `tdd="true"`, but it is a test-only plan: the production adapter, codec and notification seam already implement N=3 broadcast convergence, so the target test passed on its first run. The TDD gate's fail-fast rule says to STOP and investigate; the investigation found no missing production behavior. The honest outcome is one `test(04-07)` commit plus this disclosure — not an invented failing assertion and not an unnecessary production change.
- **`waitServing` adds diagnostics, not semantics.** It mirrors `waitFor`'s `visibilityDeadline`, `pollEvery` cadence and hard-deadline clamp so the test keeps the fixture's timing model, but records the last observed record because the plan requires a timeout to name the per-store state.
- **The probe and socket recipe are reused unchanged.** `freeUDPPort`, `probeBroadcastAddr` and `smokeStore` keep the same-host topology and loud-skip contract in one place; the new test adds only the N=3 arrangement and assertions.

## Deviations from Plan

### Process deviation

**1. [Process - TDD Gate] A test-only task cannot produce a legitimate RED**
- **Found during:** Task 1 RED
- **Issue:** `tdd="true"` demands a target test that fails on a planned assertion before implementation. This plan's only deliverable is the test, and the asserted production behavior already existed, so the test passed on its first run — the gate's `unexpected_green` classification.
- **Fix:** Investigated per the TDD reference: the feature already exists, so there is no GREEN implementation to write. Committed the test as the atomic deliverable and recorded the gate outcome here instead of fabricating a RED record, a failing assertion, or a production change. A non-committed mutation check (notification seam disabled) proved the test fails on its planned convergence assertion, so the test is not vacuous.
- **Files modified:** `internal/udprepl/three_node_test.go`
- **Verification:** `go test -race -count=3 ./internal/udprepl/ -run '^TestSmoke_threeNodeBroadcastConvergence$'` green; the mutation run failed at 5 s with `record absent: key not found`.
- **Committed in:** `21046ff`

No behavioral deviation: the test asserts exactly what the plan's must-haves require.

## TDD Gate Compliance

- **RED:** not produced — `unexpected_green`. The plan is test-only; the production behavior the test targets already existed, so no legitimate failing target test was possible without fabricating one.
- **GREEN:** the test passes (`21046ff`); no `feat(...)` commit exists because no production change was needed.
- **REFACTOR:** none separately — the test was lint-clean as written.
- The end-of-phase TDD review will score this plan's RED/GREEN pair as missing; that finding is advisory and this section is the reason, recorded rather than hidden.

## Issues Encountered

- The skip branch could not be exercised on this Linux host because the loopback broadcast probe succeeds; the skip path is the inherited `probeBroadcastAddr` contract (the two-node fixture's loud-skip reasons), verified by inspection and covered by the existing macOS-CI history rather than by a run here.
- None else: the full `-race` package suite, three consecutive target runs, `go tool golangci-lint run ./internal/udprepl/`, `go build ./...` and `go vet ./internal/udprepl/` are all green.

## User Setup Required

None - no external service configuration required.

## Threat Flags

None. The test introduces no network endpoint, no credential path and no dependency; it binds one ephemeral loopback UDP port through the existing test-only option.

## Next Phase Readiness

- PROTO-09's real-socket half is proven: three production adapters on one shared loopback port converge for every origin, with same-key LWW and a loud capability skip. The limitation is documented for 04-12's `docs/04-MULTI-NODE.md`.
- 6 plans remain in Phase 4. Ready for `04-08-PLAN.md`.

## Self-Check: PASSED

- `internal/udprepl/three_node_test.go` exists (6655 bytes, 170 lines).
- The task commit `21046ff` is an ancestor of HEAD.
- `TestSmoke_threeNodeBroadcastConvergence` passes three consecutive `-race` runs, the package is green, and lint is clean.
