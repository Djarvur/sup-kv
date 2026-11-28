---
phase: 08-performance
plan: 06
subsystem: test-port-allocation
tags: [performance, flake-class, ports, allocator, perf-06, tdd]
requires:
  - phase: 08-performance
    provides: the single threshold table, `make perf` and the `race-soak` target (08-01)
provides:
  - one claim-while-held port allocator in internal/testutil with four disjoint package bands below the kernel's ephemeral range
  - every socket-owning test package adopted it (internal/stand, cmd/sup-kv, internal/udprepl, internal/grpcapi); both duplicate private registries deleted
  - the race-soak flake-class-1 note describes the whole-suite mitigation, not only the stand
affects: [08-09, internal/testutil, internal/stand, cmd/sup-kv, internal/udprepl, internal/grpcapi, Makefile]
actuals:
  tokens: 8761
  tasks: 3
  commits: 4
plan_head_before: 43f43ebccf5cf2f4c7a71e2f05a74de381ffc969
plan_head_after: 3e7b465558622238d61826d94e0dae34b065275e
tech-stack:
  added: []
  patterns:
    - "Claim-while-held port pick: bind the candidate, claim it in a network:port registry, release the bind, and drop the claim from the test's cleanup"
    - "Disjoint per-package bands below the kernel's ephemeral range make cross-process collisions impossible by construction, not by coordination"
    - "Assert the host assumption (ip_local_port_range) loudly with the sysctl path and the band in the message, so a non-default host is a named environment error"
key-files:
  created:
    - internal/testutil/ports.go
    - internal/testutil/ports_test.go
  modified:
    - internal/testutil/bench.go
    - internal/stand/stand.go
    - cmd/sup-kv/main_test.go
    - cmd/sup-kv/broadcast_probe_test.go
    - internal/udprepl/smoke_test.go
    - internal/grpcapi/wait_integration_test.go
    - Makefile
key-decisions:
  - "The allocator's failure path is split into claim (returns an error) and pick (turns it into testing.TB.Fatalf): testing.TB cannot be implemented outside the testing package, so this is the only way to test the empty/full band contract without a subprocess."
  - "TCP probes use net.Listen and UDP probes use net.ListenConfig.ListenPacket: ListenPacket rejects the tcp network, which the first GREEN run exposed as 32 failed TCP probes."
  - "The bands are constructor functions, not exported globals, so gochecknoglobals has nothing to waive and a package cannot invent an overlapping band."
  - "The registry is keyed by network:port, so a TCP and a UDP pick may share a number; the claim map is the single permitted package global, with the stand's precedent and a written reason."
  - "The four package wrappers keep their existing names and signatures (scenarioUDPPort, freeTCPPort, freeUDPPort, integrationUDPPort), so no call site changed."
patterns-established:
  - "A flake class is closed by making the collision impossible (disjoint bands + held claim), not by retrying at the call site or removing t.Parallel."
requirements-completed: [PERF-02, PERF-06]
coverage:
  - id: D1
    description: "One allocator, one network:port registry, four pairwise disjoint bands inside [28100, 32100) and above the [27900, 28100) doc-port reserve; claim-while-held with a 32-candidate bound and a testing.TB signature"
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/testutil/ -run 'TestBands|TestUDPPort|TestTCPPort|TestRegistry|TestEphemeral' -v (6/6 PASS)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The ephemeral-range assumption is asserted on Linux (named path and band) and skipped with a comment elsewhere; an injected low bound at the band edge fails, one port above passes"
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "TestEphemeralRangeAssertion against the host sysctl 32768 60999; GOOS=darwin go vet ./internal/testutil/ exits 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every socket-owning package delegates to the allocator and both private registries are gone (scenarioPorts, probePorts, portRegistry); call sites and newStandPair's probe-retry unchanged"
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestLog_formatText|TestMain_lifecycle|TestBroadcastProbe|TestMain_version' -v and ./internal/stand/ -run 'TestStand|TestScenario|TestDelete' (PASS); grep finds no private registry in either file"
        status: pass
    human_judgment: false
  - id: D4
    description: "No advertised port is picked from :0 in the five former picker files; twenty repetitions of the fast socket tests and five of the stand's non-kill scenarios pass under -race"
    requirement: PERF-06
    verification:
      - kind: command
        ref: "go test -race -count=20 -run 'TestProbeQuickstartBroadcast|TestSmoke_twoNodesUDP|TestWaitIntegration_' ./cmd/sup-kv/ ./internal/udprepl/ ./internal/grpcapi/ (20 PASS per test, 160 RUN/PASS lines, load ~1.6); go test -race -count=5 -run 'TestStand_delete|TestStand_bufferSizesDeliver|TestStand_startupGate|TestStand_oversizeKeyF2c' ./internal/stand/ (ok, 20.4s)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Makefile's race-soak flake-class-1 note names internal/testutil and all four socket packages instead of only the stand's window"
    requirement: PERF-06
    verification:
      - kind: command
        ref: "grep -q 'internal/testutil' Makefile; the five-file no-:0 grep chain exits 0"
        status: pass
    human_judgment: false
duration: 9min
completed: 2026-11-24
status: complete
---

# Phase 08 Plan 06: One Port Allocator, Four Disjoint Bands Summary

**The pick-then-bind flake class is structurally gone: every socket-owning test package now draws its advertised ports from a single claim-while-held allocator whose four disjoint bands sit below the kernel's ephemeral range, and both duplicate private registries are deleted.**

## Performance
- **Duration:** 9min
- **Started:** 2026-11-24T18:38:59Z
- **Completed:** 2026-11-24T18:47:52Z
- **Tasks:** 3
- **Files modified:** 9 (2 created, 7 modified)

## Accomplishments

- `internal/testutil/ports.go` is the module's one port allocator. `Band` is an unexported-bounds struct with four constructor functions — `StandBand()` `[28100, 29100)`, `CmdBand()` `[29100, 30100)`, `UDPReplBand()` `[30100, 31100)`, `GRPCApiBand()` `[31100, 32100)` — all above the `[27900, 28100)` doc-port reserve and below the kernel's ephemeral range. Constructors rather than exported globals keep `gochecknoglobals` empty and make an overlapping band impossible to declare.
- `UDPPort`/`TCPPort(tb testing.TB, band Band)` share `claim`, which walks up to 32 random candidates: it binds the candidate on the wildcard address, claims `network:port` in the one mutex-guarded registry **while the bind is still held**, closes the probe, and returns the port; `pick` registers the release as `tb.Cleanup`, so the claim outlives the call and every later pick in the same test skips it. `UDPPort`/`TCPPort` fail with the band and attempt count, never with a zero port.
- The sysctl assumption is asserted, not assumed: on Linux `checkEphemeral` reads `/proc/sys/net/ipv4/ip_local_port_range` and `ephemeralOK(low, band)` fails naming the path, the low bound and the band when `low <= band.hi`; on non-Linux the check skips with a comment (darwin is development-only, 14.10). The test drives the failure through the injected `low`, so it never needs a non-default host.
- `internal/stand`'s `scenarioUDPPort`/`freeTCPPort` and `cmd/sup-kv`'s `freeTCPPort`/`freeUDPPort`/`scenarioUDPPort` became thin delegations with their names and signatures intact; `internal/stand`'s `portRegistry`/`scenarioPorts` and `cmd/sup-kv`'s `portRegistry`/`probePorts` plus their duplicated window constants are deleted. `newStandPair`'s probe-retry still re-picks a fresh pair when the *probe* cannot bind — that retry belongs to the capability probe, not the allocator.
- `internal/udprepl`'s `freeUDPPort(tb)` and `internal/grpcapi`'s `integrationUDPPort(tb)` delegate to `UDPReplBand()`/`GRPCApiBand()` and keep their `testing.TB` signatures, so the burst and confirmation-latency benchmarks are untouched.
- The `Makefile`'s race-soak note now says class 1 is mitigated for the whole suite by the one allocator and the four bands, rather than only by the stand's private window.

## Task Commits

1. **Task 1 RED: failing band, claim and sysctl tests** — `6c03330` (test)
2. **Task 1 GREEN: the shared allocator** — `074ccd0` (feat)
3. **Task 2: stand and cmd/sup-kv adoption, both registries deleted** — `5c7a3d7` (refactor)
4. **Task 3: udprepl and grpcapi adoption, Makefile note** — `3e7b465` (refactor)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/testutil/ports.go` — `Band` + four constructors, `UDPPort`/`TCPPort`, `pick`, `claim`, `probeBind`, `reserve`/`release`, `checkEphemeral`/`ephemeralOK`, and the `network:port` registry.
- `internal/testutil/ports_test.go` — six in-package tests (`testpackage` waived with the reason): band disjointness and window bounds, 64 distinct in-band picks, network-keyed claims, taken-candidate skip, the ephemeral-range assertion, and the empty/full band failure contract.
- `internal/testutil/bench.go` — the package doc gains the allocator paragraph (the package doc has one home).
- `internal/stand/stand.go` — wrappers on the stand band; `portRegistry`, `scenarioPorts` and the window constants removed; `math/rand/v2` import dropped.
- `cmd/sup-kv/main_test.go`, `cmd/sup-kv/broadcast_probe_test.go` — delegations on the cmd band; the probe's private registry and constants removed.
- `internal/udprepl/smoke_test.go`, `internal/grpcapi/wait_integration_test.go` — delegations on their own bands, `testing.TB` signatures preserved.
- `Makefile` — flake-class-1 note rewritten for the whole suite.

## Decisions Made

- **Claim failures are data, pick failures are test failures.** `claim` returns `(int, error)`; `pick` turns the error into `tb.Fatalf`. Go forbids implementing `testing.TB` outside the testing package, so this split is the only way to assert the empty-band and exhausted-band messages in-process; `pick` cannot return `0` because it fatals first.
- **TCP probes need `net.Listen`.** `ListenPacket` rejects the `tcp` network; the first GREEN run failed every TCP candidate. `probeBind` now branches per transport, and the same `io.Closer` contract closes both.
- **Two assertions per candidate, in this order.** The probe bind happens first (a foreign holder simply skips the candidate), then the claim is taken while the bind is still held. A consumer bind failure therefore means an off-suite holder inside the band, never a sibling test.
- **The band is the doc-bound rationale's new home.** The deleted window comments moved onto the wrappers, which name `testutil.StandBand`/`CmdBand` and the doc-port reserve, so the harness still explains why its ports are unreachable.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `ListenPacket` does not accept the `tcp` network**
- **Found during:** Task 1 GREEN (`TestRegistryKeysByNetwork` failed: every TCP probe was rejected, so the TCP claim never shared the UDP number)
- **Issue:** The plan's single `ListenPacket(ctx, network, ...)` helper works for `udp4` but not for `tcp`; `net.ListenPacket` supports datagram networks only.
- **Fix:** Added `probeBind`, which uses `net.ListenConfig.Listen` for `tcp` and `ListenPacket` otherwise, returning the open socket as `io.Closer`.
- **Files modified:** `internal/testutil/ports.go`
- **Commit:** `074ccd0`

**2. Task 1 RED shape: a compiling scaffold instead of a missing-symbol build failure**
- **Found during:** Task 1 RED
- **Issue:** The plan's behavior text says the tests fail "because the allocator does not exist"; in Go that is a build failure, which `gsd_run check tdd-red-evidence` refuses (`build_failure`/INVALID_RED, #3770).
- **Fix:** The RED commit ships the band table and the two old throwaway-`:0` pickers verbatim (the class being removed) plus no-op registry/sysctl stubs, so the package compiles and five tests fail on real assertions — the target `TestUDPPortPickIsInBandAndDistinct` returned 55293 outside `[28100, 29100)`, and the registry, empty-band and sysctl tests failed likewise. GREEN replaced the scaffold with the claim-while-held implementation.
- **Files modified:** `internal/testutil/ports.go`, `internal/testutil/ports_test.go`
- **Commit:** `6c03330` (RED), `074ccd0` (GREEN)

**3. `internal/testutil/bench.go` touched for the package doc**
- **Found during:** Task 1
- **Issue:** The plan asks `ports.go` to extend the package doc, but the package doc lives in `bench.go`; a second `// Package testutil` comment would be a duplicate package comment.
- **Fix:** The allocator paragraph was added to `bench.go`'s package doc, and `ports.go` carries a file comment after the package clause. The file is an addition to the plan's `files_modified`.
- **Files modified:** `internal/testutil/bench.go`
- **Commit:** `074ccd0`

### Accepted residue (named, not "fixed")

- Throwaway transmit sockets that are never advertised: `smokeSocket`, `integrationSocket`, `logProbeSocket` and `probeLoopbackSend` bind `0.0.0.0:0` for the capability probe and the adapter's exact socket recipe. They are not pick-then-bind sites.
- The pre-existing broadcast sink at `internal/udprepl/decode_negative_test.go:625` (`ListenPacket(t.Context(), "udp4", "0.0.0.0:0")`) is a test-held sink the adapter broadcasts to, never a port advertised to a second binder. It stays unchanged, as the plan names it.

### Scope note

- The full-package `-count=20` repetition remains out of scope for this wave-2 plan: it would repeat the doc-walkthrough stalls (F-7) and the `killAndReap` diagnostic path (F-6), which 08-09 closes. The repetition proof is scoped to the socket-owning test names, exactly as the plan states.

## TDD Gate Compliance

| Gate | Commit | Result |
|------|--------|--------|
| RED (Task 1) | `6c03330` test(08-06): add failing band, claim and sysctl tests for the shared port allocator | `RED_EVIDENCE_OK` (`target_test_failed`): `TestUDPPortPickIsInBandAndDistinct` failed on `UDPPort(t, internal/stand [28100, 29100)) = 55293, outside the band`; 5 of 6 tests failed on their planned assertions, the band table passed as the tripwire |
| GREEN (Task 1) | `074ccd0` feat(08-06): implement the shared claim-while-held port allocator | 6/6 tests PASS under `-race`; `golangci-lint run ./internal/testutil/` and `GOOS=darwin go vet` both exit 0 |
| Task 2 | `5c7a3d7` refactor(08-06): adopt the shared port allocator in stand and cmd/sup-kv | Adoption task (`type=auto`), no TDD flag |
| Task 3 | `3e7b465` refactor(08-06): adopt the shared port allocator in udprepl and grpcapi | Adoption task (`type=auto`), no TDD flag |

The RED evidence record, console output and exit file are under `.planning/phases/08-performance/red-evidence/20261124T183934Z/` (untracked, matching the phase's red-evidence convention). The semantic assessment: the target executed and failed on the out-of-band pick it received, not on a build, load or unrelated failure; the same run's other four failures name the registry key, the taken-candidate skip, the sysctl assertion and the empty/full band contract.

## Issues Encountered

- `make lint` flagged `nilnil` (probeBind's `return listener, err`), `noinlineerr` (five inline error checks in the production file and tests) and two unused band-bound constants. Fixed with separate returns, plain assignments and by deleting the constants; the full gate now exits 0.
- `GOOS=darwin go vet ./internal/testutil/` compiles the `runtime.GOOS` branch with the `/proc` path as a string only — no platform build tags were needed.
- The full gates after the work: `make lint` exit 0; `make test` (`go test -race -cover ./...`) exit 0 in 51.5 s at load ~1.3-1.7, with `internal/testutil` at 75.0 % coverage, `internal/grpcapi` 98.5 %, `internal/udprepl` 89.9 % and `internal/stand` 84.0 %.
- The `-count=20` run reproduced 20 PASS per test (160 RUN/PASS lines) for `TestProbeQuickstartBroadcast`, `TestSmoke_twoNodesUDP` and both `TestWaitIntegration_*` tests; the stand's five scoped scenarios passed in one 20.4 s run.
- Prohibited/frozen files untouched: `git diff --name-only 43f43eb..HEAD` lists exactly the nine files above — `internal/stand/stand_red_test.go`, `internal/grpcapi/red_gate_test.go` and the `decode_negative_test.go` sink are byte-identical.

## User Setup Required

None - no external service configuration required. `make perf` was not required by this plan's verification (the port allocator is exercised by the ordinary race suite).

## Known Stubs

None. The allocator's every failure path is asserted; the accepted residue above is named design, not a stub.

## Threat Flags

None. No production surface changed: the new `net`/`os` reads live only in `internal/testutil`, which the `no-test-libs-in-prod` exemption already covers, and the threat register's T-08-14/15/16 mitigations are the implemented claim-while-held, the sysctl assertion and the disjoint-band test.

## Next Phase Readiness

Ready for the rest of wave 2. Every socket-owning test package now shares one allocator, so a port-contention flake can only come from the readiness/deadline classes that 08-09 owns (F-5's timeout half, F-6, F-7). The allocator's band/claim/ephemeral tests are the regression guard; widening a band is now a single visible constant change in `internal/testutil/ports.go` plus its disjointness test.

## Self-Check: PASSED

- Files: `internal/testutil/ports.go`, `internal/testutil/ports_test.go`, `internal/testutil/bench.go`, `internal/stand/stand.go`, `cmd/sup-kv/main_test.go`, `cmd/sup-kv/broadcast_probe_test.go`, `internal/udprepl/smoke_test.go`, `internal/grpcapi/wait_integration_test.go`, `Makefile` all found.
- Commits: `6c03330`, `074ccd0`, `5c7a3d7`, `3e7b465` all ancestors of HEAD (measured: `git rev-list --count 43f43eb..HEAD` = 4).
