---
phase: 06-api-and-semantics
plan: 07
subsystem: api-transport-health
tags:
  - apis-05
  - transport-limits
  - max-recv-msg-size
  - max-send-msg-size
  - keepalive
  - message-envelope
  - resource-exhausted
  - page-byte-budget
  - one-way-health
  - shutdown-race
requires:
  - phase: 06-api-and-semantics (06-05)
    provides: listPageByteBudget, listEnvelopeOverheadBytes, listRecordOverheadBytes and the serialized-entry page walk
  - phase: 06-api-and-semantics (06-01)
    provides: the public api/supkvapi/v1 schema the harness and the production server register
  - phase: 06-api-and-semantics (06-12) and 04-protocol (04-04)
    provides: the startup gate, RunGate and its 1:1 health binding
provides:
  - "internal/grpcapi/limits.go: TransportLimits, LimitsFor(valueMaxBytes int) and (TransportLimits).ServerOptions()"
  - "the production server built as grpc.NewServer(limits.ServerOptions()...) from cfg.valueMaxBytes"
  - "Server.Shutdown(): the standard health server's one-way Shutdown"
  - "the binary's shutdown calls node.api.Shutdown() before apiCancel"
  - "the over-limit ResourceExhausted proof and the worst-case page-fit proof"
affects:
  - 06-13-phase-gate
  - phase-9-api-docs
actuals:
  tokens: 6441
  tasks: 3
  commits: 7
  plan_head_before: c7fc3d9c648b13150be7242a672c2c3405d1545a
  plan_head_after: b50a4a2dc9351658ee96df96ceebfdfe678be95b
tech-stack:
  added: []
  patterns:
    - "A transport envelope is a pure function of the EFFECTIVE configured value limit plus the page-budget constants; no gRPC default reaches a message size or the keepalive cadence."
    - "A one-way health latch at shutdown: the standard health server's Shutdown, called before the API context is cancelled, ignores every later status flip, so a late gate backstop cannot resurrect SERVING."
    - "The test harness opts into production transport options only where a test is about the transport; every other harness keeps the plain server so its assertions stay about behavior."
key-files:
  created:
    - internal/grpcapi/limits.go
    - internal/grpcapi/limits_internal_test.go
    - internal/grpcapi/limits_test.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-07-task1.json
    - .planning/phases/06-api-and-semantics/red-evidence/06-07-task2.json
    - .planning/phases/06-api-and-semantics/red-evidence/06-07-task3.json
  modified:
    - internal/grpcapi/gate.go
    - internal/grpcapi/gate_test.go
    - internal/grpcapi/server_test.go
    - cmd/sup-kv/main.go
    - .planning/phases/06-api-and-semantics/deferred-items.md
key-decisions:
  - "MaxRecvBytes = storage.MaxKeyBytes + the EFFECTIVE value limit + apiEnvelopeOverheadBytes; at the storage default that is 63 846 bytes and at the 64 408 datagram-ceiling limit 66 814, so a legitimate maximum-size write is never rejected at the transport (Pitfall 11, T-06-24)."
  - "MaxSendBytes = listPageByteBudget + listEnvelopeOverheadBytes (6 MiB + 1 382); this plan consumes both 06-05 constants and redefines neither, and the ceiling is provable only because the page walk costs full serialized entries (key twice, 16-byte version, value, record framing)."
  - "apiEnvelopeOverheadBytes is exactly 06-05's listEnvelopeOverheadBytes: the request/response field framing plus the base64 next_page_token; the per-entry framing already lives inside listPageByteBudget."
  - "The keepalive triple is 2 m server ping, 20 s timeout, 30 s minimum client interval with PermitWithoutStream true — the documented trusted-LAN consumer contract: an idle connection between writes stays legal."
  - "The RED for Task 1 needed a compiling skeleton because the plan's literal first run was a build failure and the go-test RED classifier refuses [build failed]; the skeleton returned the zero TransportLimits so the target failed on its assertions, then GREEN replaced it."
  - "The over-limit harness knob is opt-in (nil keeps the plain grpc.NewServer()), so no pre-06-07 test changed its transport behavior; only the receive-ceiling test and the page-fit test run under LimitsFor."
  - "The one-way test waits for the backstop by observing a successful Set instead of sleeping: openGate's gate flip is what a successful write proves, and the late health flip it attempted is exactly what the test then requires to be ignored."
  - "The stale .git/gsd-plan-head-before-06-07 ledger (dated 2026-11-16) was reset to c7fc3d9, the plan-start HEAD, before measuring actuals — the 06-01/06-03/06-05 precedent."
patterns-established:
  - "A pure limits function: every transport value is derived from the configured protocol constants and pinned by a table at the default, the datagram ceiling and both non-positive normalizations."
  - "A transport-level proof next to the handler-level one: the same oversized value answers ResourceExhausted at the transport and InvalidArgument at the handler, so a test that asserts the former proves which layer spoke."
  - "A one-way shutdown latch is tested by making the late writer observable (the successful Set) rather than by racing a sleep against a timer."
requirements-completed: []
coverage:
  - id: D1
    description: "TransportLimits/LimitsFor/ServerOptions exist; LimitsFor derives the receive ceiling from the effective value limit and the send ceiling from the 06-05 page budget, normalizes non-positive input, and pins the 2m/20s/30s keepalive triple; the production server is grpc.NewServer(limits.ServerOptions()...) built from cfg.valueMaxBytes."
    requirement: "APIS-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestLimitsFor$' -v -> PASS"
        status: pass
      - kind: process
        ref: "grep 'func LimitsFor(valueMaxBytes int) TransportLimits', 'grpc.MaxRecvMsgSize', 'grpc.MaxSendMsgSize' (limits.go); 'grpc.NewServer(limits.ServerOptions()...)' and 'grpcapi.LimitsFor(cfg.valueMaxBytes)' (main.go) -> all exit 0"
        status: pass
      - kind: build
        ref: "go build ./... -> exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "A request whose value alone reaches MaxRecvBytes is rejected at the transport with codes.ResourceExhausted, distinct from the handler's InvalidArgument value limit, while a Set at the configured value limit succeeds."
    requirement: "APIS-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestGRPC_maxRecv$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "A 257-entry store of exactly maximum-length keys and maximum-size values answers a page_size 256 List with a non-empty page that stops on the byte budget below 256 and marshals within LimitsFor(storage.ValueMaxBytes).MaxSendBytes."
    requirement: "APIS-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestGRPC_worstCasePageFitsSendLimit$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D4
    description: "Server.Shutdown calls the standard health server's one-way Shutdown (idempotent, no Resume): after Shutdown and a late 50 ms gate backstop that provably opened the gate, health still reads NOT_SERVING, while the control harness without the pre-shutdown reaches SERVING; the binary calls node.api.Shutdown() before apiCancel and never flips health directly."
    requirement: "APIS-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestHealth_shutdownIsOneWay$' -v -> PASS (plus TestHealth_backstopWithoutShutdownServes control, -count=5)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestHealth_reflectsGate$' -v -> PASS"
        status: pass
      - kind: process
        ref: "grep -q 'node.api.Shutdown()' main.go && ! grep -q 'node.health.SetServingStatus' main.go -> exit 0; no health.Resume anywhere"
        status: pass
      - kind: build
        ref: "go build ./... && go vet ./... -> exit 0"
        status: pass
    human_judgment: false
duration: 9min
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 07: Transport Envelope and One-Way Health Summary

**The gRPC transport envelope is now a pure function of the effective value limit and 06-05's page budget — a 63 846-byte receive ceiling at the default (66 814 at the 64 408 datagram limit), a 6 MiB + 1 382 send ceiling, a 2 m/20 s/30 s keepalive policy — and shutdown closes health through the standard one-way `Shutdown`, so a gate backstop firing afterwards can never return `SERVING`.**

## Performance

- **Duration:** 9 min (522 s)
- **Started:** 2026-11-23T01:00:49Z
- **Completed:** 2026-11-23T01:09:31Z
- **Tasks:** 3
- **Files modified:** 11 (7 code/test, 3 RED records, 1 deferred-items row)
- **Commits:** 7 measured from ledger `c7fc3d9..b50a4a2`

## Accomplishments

- `internal/grpcapi/limits.go`: `TransportLimits` (receive/send ceilings and the keepalive triple), `LimitsFor(valueMaxBytes)` — normalizing a non-positive input to `storage.ValueMaxBytes`, deriving recv from `storage.MaxKeyBytes + effective limit + apiEnvelopeOverheadBytes` and send from `listPageByteBudget + listEnvelopeOverheadBytes` — and `ServerOptions()` rendering `MaxRecvMsgSize`, `MaxSendMsgSize`, `KeepaliveParams` and `KeepaliveEnforcementPolicy{MinTime: 30s, PermitWithoutStream: true}`.
- `cmd/sup-kv/main.go`: `listenGRPC` takes the limits and constructs `grpc.NewServer(limits.ServerOptions()...)`; `wire` passes `grpcapi.LimitsFor(cfg.valueMaxBytes)` — the effective configured limit, never the storage default.
- `TestGRPC_maxRecv`: the oversized Set — its value alone at `MaxRecvBytes` — now answers `ResourceExhausted` from the transport, while the same harness answers `InvalidArgument` at the handler for the store's value limit; a legitimate maximum-size Set at the configured limit still succeeds.
- `TestGRPC_worstCasePageFitsSendLimit`: 257 records with exactly `storage.MaxKeyBytes`-length keys and `storage.ValueMaxBytes` values, `page_size=256`; the page stops on the 6 MiB serialized budget below 256 entries and its marshaled size fits `MaxSendBytes` — the cross-plan proof that 06-05's accounting and this plan's ceiling agree.
- `Server.Shutdown()` calls `s.healthSrv.Shutdown()`, the standard health server's one-way latch; `TestHealth_shutdownIsOneWay` calls it twice (idempotent), proves the late 50 ms backstop did open the gate (a successful `Set`), and then requires `NOT_SERVING`; the control harness without the pre-shutdown reaches `SERVING`.
- The binary's `wired` now carries `api *grpcapi.Server`; `shutdown`'s first statement is `node.api.Shutdown()`, the direct `SetServingStatus` and the `google.golang.org/grpc/health` import are gone, `apiCancel` keeps its position, and `health.Resume` is never called.

## TDD Evidence

| Task | RED record | RED observation | Target |
|------|------------|-----------------|--------|
| 1 | `red-evidence/06-07-task1.json` | `LimitsFor` returned the zero struct: `MaxRecvBytes` 0 versus 63 846/66 814 | `TestLimitsFor` |
| 2 | `red-evidence/06-07-task2.json` | the unconfigured harness accepted the message and the handler answered `InvalidArgument`, not `ResourceExhausted` | `TestGRPC_maxRecv` |
| 3 | `red-evidence/06-07-task3.json` | the backstop flipped health back to `SERVING` after shutdown began — the T-06-23 race | `TestHealth_shutdownIsOneWay` |

All three passed `gsd_run check tdd-red-evidence` (`RED_EVIDENCE_OK`).

## Task Commits

1. **Task 1 RED: failing transport limits table** — `93e60cf` (test)
2. **Task 1 GREEN: derive explicit transport limits** — `b2577d7` (feat)
3. **Task 2 RED: failing transport ceiling and page tests** — `6dbf2d8` (test)
4. **Task 2 GREEN: prove the transport ceiling rejects oversized input** — `ded51aa` (feat)
5. **Task 3 RED: reproduce the late-backstop health race** — `a4a8721` (test)
6. **Task 3 GREEN: make health one-way at shutdown** — `36c6c59` (feat)
7. **Deferred-items row for the UDP port flake** — `b50a4a2` (docs)

**Plan metadata:** the final `docs(06-07)` commit.

## Files Created/Modified

- `internal/grpcapi/limits.go` — the envelope struct, `LimitsFor`, `ServerOptions`, and the derived `apiEnvelopeOverheadBytes`.
- `internal/grpcapi/limits_internal_test.go` — `TestLimitsFor`: the default, the 64 408 ceiling, and both non-positive normalizations, plus the keepalive triple.
- `internal/grpcapi/limits_test.go` — `TestGRPC_maxRecv` and `TestGRPC_worstCasePageFitsSendLimit`.
- `internal/grpcapi/gate.go` — `Server.Shutdown` with the one-way documentation.
- `internal/grpcapi/gate_test.go` — the one-way test and its control.
- `internal/grpcapi/server_test.go` — the opt-in `limits` harness knob and the `api` handle.
- `cmd/sup-kv/main.go` — `listenGRPC` limits parameter, `wired.api`, and the shutdown order.
- `.planning/phases/06-api-and-semantics/deferred-items.md` — one row for the recurring UDP port-contention flake.

## Decisions Made

- The receive ceiling follows the effective value limit, never `storage.ValueMaxBytes`; the table pins both ends so a legitimate maximum-size write is not rejected at the transport.
- The send ceiling consumes `listPageByteBudget` and `listEnvelopeOverheadBytes` verbatim; `apiEnvelopeOverheadBytes` is an alias constant, not a second definition.
- `apiEnvelopeOverheadBytes` covers framing and the base64 token only; per-entry framing stays inside the page budget, so the two constants cannot drift.
- The harness's transport options are opt-in: nil keeps `grpc.NewServer()` for every pre-existing test.
- The one-way test observes the gate transition through a successful `Set` rather than a sleep, making the late health flip provably attempted.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 1's RED could not be a compile-clean test without a skeleton**
- **Found during:** Task 1
- **Issue:** The plan's literal first run is a build failure (`undefined: LimitsFor`, `undefined: apiEnvelopeOverheadBytes`), and the go-test RED classifier refuses `[build failed]` runs.
- **Fix:** The test was written first and the build failure observed; a compiling skeleton (`LimitsFor` returning the zero `TransportLimits`) was committed with it so the target executed and failed on its assertions, producing a machine-valid RED record. GREEN replaced the skeleton with the real implementation.
- **Files modified:** `internal/grpcapi/limits.go`, `internal/grpcapi/limits_internal_test.go`
- **Commit:** `93e60cf`

**2. [Rule 3 - Blocking] Task 2's RED needed a harness seam that did not exist**
- **Found during:** Task 2
- **Issue:** The harness had no way to configure transport options, so the receive-ceiling test could only be written against the plain server, where it fails on the code assertion (`InvalidArgument` instead of `ResourceExhausted`).
- **Fix:** The test was committed in that state as the RED record, then GREEN added the opt-in `limits` knob to `testServerConfig`/`buildTestServer` and wired the two new tests to it. The assertion itself never changed.
- **Files modified:** `internal/grpcapi/server_test.go`, `internal/grpcapi/limits_test.go`
- **Commit:** `6dbf2d8`, `ded51aa`

**3. [Rule 3 - Blocking] Task 3's RED needed the harness handle and a compiling Shutdown**
- **Found during:** Task 3
- **Issue:** `h.api.Shutdown()` did not exist on the harness, and `Server.Shutdown` did not exist, so a literal test-first run is again a build failure.
- **Fix:** The harness gained `api *grpcapi.Server`, and the skeleton `Server.Shutdown` mirrored today's `main.go` behavior (`SetServingStatus`) — which reproduces the production race exactly; GREEN replaced the body with `s.healthSrv.Shutdown()`.
- **Files modified:** `internal/grpcapi/gate.go`, `internal/grpcapi/gate_test.go`, `internal/grpcapi/server_test.go`
- **Commit:** `a4a8721`

### Plan-text clarifications

- **Task 2's worst-case page test passes before its own GREEN step by design.** It pins the agreement between 06-05's serialized-entry budget and this plan's `MaxSendBytes`, both of which already existed after Task 1; only the receive-ceiling target was red in Task 2's RED run. The record and semantic assessment say so.
- **Task 3's control is a separate test.** The plan described it as a behavior case; it is `TestHealth_backstopWithoutShutdownServes`, while `TestHealth_reflectsGate` remains the pre-existing both-transitions pin the verify names.

## Issues Encountered

- One combined `go test -race ./internal/grpcapi/ ./cmd/sup-kv/` run failed in `cmd/sup-kv` `TestLog_formatText` with `probe rx bind: ... address already in use`. The package passes alone (5.8 s) and the flake is the documented cross-package UDP port-contention class already recorded for 04-01, 05-04, 06-01 and 06-03; a row was added to `deferred-items.md`. No bound or test was changed.
- The pre-existing `.git/gsd-plan-head-before-06-07` ledger was stale (dated 2026-11-16, base `cf1923b`, which counted 444 commits); it was reset to the plan-start HEAD `c7fc3d9` before measuring, following the 06-01/06-03/06-05 precedent.
- No authentication gates and no user setup were needed.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. The production listener uses `LimitsFor(cfg.valueMaxBytes)`, the shutdown path uses the one-way health call, and every constant is functional.

## Threat Surface Scan

No new surface beyond the plan's threat model. T-06-21 (explicit receive/send ceilings), T-06-22 (keepalive enforcement), T-06-23 (one-way health during shutdown) and T-06-24 (limits from the effective configured limit) are all mitigated and pinned by the tests above; T-06-SC is untouched (no package was installed).

## Next Phase Readiness

- APIS-05's REQUIREMENTS checkbox stays pending: `requirements ready-ids .planning/phases/06-api-and-semantics/06-07-PLAN.md APIS-05` reports it blocked (the #2388 shared-ID gate among the sibling plans that also declare it), so this plan marks no requirement complete.
- Plan 06-13's phase gate can now run `make lint && make test` with the transport limits and the one-way shutdown in place; the focused `-race` suite and the full `internal/grpcapi` and `cmd/sup-kv` packages are green.
- Four plans remain in the phase (06-08, 06-09, 06-10, 06-13). Ready for 06-08.

## Self-Check: PASSED

All listed key files exist and all seven task commits are ancestors of HEAD.
