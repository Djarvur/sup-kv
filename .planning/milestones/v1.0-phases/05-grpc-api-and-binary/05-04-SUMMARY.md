---
phase: 05-grpc-api-and-binary
plan: 04
subsystem: api
tags: [startup-gate, grpc-health, snapshot-rpc, unavailable-writes, readiness, bufconn]

# Dependency graph
requires:
  - phase: 05-grpc-api-and-binary
    provides: 05-01's bufconn harness and SupKV server with the pinned error map; 05-02's udprepl Adapter.SyncArrived atomic mirror (the production gate signal 05-06 injects); 05-03's WatchHub whose tap makes restore replays watchable
  - phase: 04-replication-and-deterministic-simulator
    provides: persist.Engine Snapshot/Restore with write-confirmed flush semantics; persistfile name validation closed before any filesystem touch
provides:
  - grpcapi.ServerOptions{SyncArrived func() bool, StartupGate time.Duration} — the gate wiring main (05-06) fills with adapter.SyncArrived; zero value legal (nil sync + 1s default backstop)
  - grpcapi.Server.Health() *health.Server, Server.RunGate(ctx), Server.gated() — the standard health endpoint bound 1:1 to the gate for the empty service name (NOT_SERVING at construction, SERVING exactly when the resolver opens)
  - Write gating: Set and Delete begin with the gate check returning Unavailable before any validation; Get/List/Watch never consult the gate
  - SnapshotCreate/SnapshotLoad handlers — gated writes (A5) mapping persistfile.ErrInvalidSnapshotName to InvalidArgument via the shared mapError, never validating names adapter-side
  - Harness: config-struct builder registering healthpb.RegisterHealthServer alongside SupKV and launching the resolver; awaitGateOpen/waitFor helpers; file-backed variant (real persistfile.FileStore) for snapshot tests
affects: [05-05, 05-06, phase-6-hardening]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 9222    # chars/4 over the realized diff (36887 chars)
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns: [poll-loop gate resolver with timer backstop (A8 pin, one wake per interval), 1:1 gate-to-health binding through a single CAS'd openGate, gate check as first statement of every write handler (before validation), fake-with-real-adapter harness variant when a fake would silently reproduce the control under test]

key-files:
  created:
    - internal/grpcapi/gate.go
    - internal/grpcapi/gate_test.go
  modified:
    - internal/grpcapi/server.go
    - internal/grpcapi/server_test.go

key-decisions:
  - "Gate check is the first statement of Set and Delete — before empty-key validation (API-04 rejects writes during the gate, full stop); the closed-gate test pins an empty key surfacing Unavailable, not InvalidArgument"
  - "RunGate is a 20ms poll loop (A8 pin) with a time.NewTimer backstop started at the RunGate call; nil SyncArrived never fires the sync branch and zero/negative StartupGate normalizes to the 1s design default, so a bare ServerOptions{} is a legal single-node configuration"
  - "openGate is one CAS'd transition storing the gate flag and flipping health under the same branch — the resolver is the only opener, so health is bound 1:1 to the gate (T-05-08); the empty service name is the whole-server status per the health protocol"
  - "Snapshot RPCs are gated exactly like writes (A5: a create forces a confirmed dump, a load replays writes) and names cross to persistfile untouched — its validateName is the single closed path-traversal control (T-05-07); the adapter only maps the sentinel"
  - "Snapshot tests run against a real persistfile.FileStore over a temp dir (newFileServer) — the in-memory fake deliberately does not reproduce name validation, so testing against the fake would have proven nothing about T-05-07"

patterns-established:
  - "Readiness is a resolved flag + a mirrored standard endpoint: handlers read only the resolved gate atomic, the resolver is the only writer, and the health protocol carries the observable state"
  - "Deterministic closed-gate observation under parallel load: block the resolver's injected closure until the test has observed the state (the naive 'assert before the deadline' pattern races the deadline under test)"

requirements-completed: [API-04, API-06, API-07]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Writes are Unavailable while the gate is closed — the check precedes every validation (empty key surfaces Unavailable during the gate) — while Get, List and a Watch subscription are served, including a replicated record applied during the gate"
    requirement: API-04
    verification:
      - kind: integration
        ref: "internal/grpcapi/gate_test.go#TestGate_writesUnavailableReadsOK"
        status: pass
    human_judgment: false
  - id: D2
    description: "The gate opens on the earlier of the injected sync-arrived flip or the StartupGate backstop elapsing since RunGate start (A8); nil sync never opens, zero gate normalizes to the 1s default"
    requirement: API-04
    verification:
      - kind: integration
        ref: "internal/grpcapi/gate_test.go#TestGate_opensOnSync"
        status: pass
      - kind: integration
        ref: "internal/grpcapi/gate_test.go#TestGate_opensOnTimeout"
        status: pass
    human_judgment: false
  - id: D3
    description: "The standard gRPC health endpoint reports NOT_SERVING for the empty service name before the gate opens and SERVING after — bound 1:1 to the gate, registered alongside SupKV on the same grpc.Server in the harness (the registration 05-06 main must copy)"
    requirement: API-06
    verification:
      - kind: integration
        ref: "internal/grpcapi/gate_test.go#TestHealth_reflectsGate"
        status: pass
      - kind: other
        ref: "command:grep -n RegisterHealthServer internal/grpcapi/server_test.go (buildTestServer registers healthpb.RegisterHealthServer(srv, api.Health()))"
        status: pass
    human_judgment: false
  - id: D4
    description: "SnapshotCreate/SnapshotLoad round trip through the API over the real persistfile adapter: Set, create, delete both keys, load — Get returns restored values; a Watch stream attached before the load observes the deletes as tombstone events and the replay as fresh live events (fresh versions dominate)"
    requirement: API-07
    verification:
      - kind: integration
        ref: "internal/grpcapi/gate_test.go#TestSnapshot_createLoadRoundTrip"
        status: pass
    human_judgment: false
  - id: D5
    description: "Hostile snapshot names (empty, '..', 'a/b', 65 chars) return InvalidArgument on both snapshot RPCs, mapped from persistfile.ErrInvalidSnapshotName through errors.Is — never validated or sanitized adapter-side, never touching the filesystem"
    requirement: API-07
    verification:
      - kind: integration
        ref: "internal/grpcapi/gate_test.go#TestSnapshot_invalidName"
        status: pass
    human_judgment: false
  - id: D6
    description: "Both snapshot RPCs are gated exactly like writes (A5): during a closed gate both return Unavailable before any engine call"
    requirement: API-04
    verification:
      - kind: integration
        ref: "internal/grpcapi/gate_test.go#TestSnapshot_gated"
        status: pass
    human_judgment: false
  - id: D7
    description: "Plan-level gates: full race+cover suite and the full-repo strict lint green; internal/app untouched (the gate lives entirely in the adapter)"
    requirement: API-04
    verification:
      - kind: other
        ref: "command:make test && go tool golangci-lint config verify && go tool golangci-lint run ./... (all exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-11-09
status: complete
commits: 2
plan_head_before: 2d41087ef36a196515343724c7c7c94bcb686032
plan_head_after: 3231c00bfc4bd66a6cd0d41a02e6567605f27606
---

# Phase 05 Plan 04: Startup Gate, Health Endpoint, and Snapshot RPCs Summary

**Startup gate with the standard gRPC health endpoint bound 1:1 to it — writes Unavailable until first sync or the backstop, reads immediate — plus SnapshotCreate/SnapshotLoad as gated, persistfile-validated RPCs, all proven over bufconn under -race**

## Performance

- **Duration:** 12 min
- **Started:** 2026-11-09T12:30:22Z
- **Completed:** 2026-11-09T12:42:38Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- `grpcapi.ServerOptions{SyncArrived func() bool, StartupGate time.Duration}` — the gate wiring as exported struct fields so 05-06 main injects `adapter.SyncArrived` and future options grow without another NewServer change; the zero value is legal (nil sync + 1s design-default backstop)
- `Server.RunGate(ctx)` — the A8-pinned poll resolver: one goroutine, 20ms poll, `time.NewTimer` backstop started at the RunGate call, opening on the earlier of the sync flip or the deadline; `Server.Health()` exposes the standard health server whose empty-service status starts NOT_SERVING at construction (the protocol's default SERVING explicitly overridden before any client can observe it) and flips SERVING in the same CAS'd branch that stores the gate flag (T-05-08's 1:1 binding)
- Write gating: Set and Delete begin with the gate check returning `Unavailable` ("startup gate: waiting for first sync round or startup_gate") before any validation — even an empty key surfaces Unavailable during the gate; Get, List and Watch never consult it, so reads — including replicated changes merging in from peers — are served from the first moment
- SnapshotCreate/SnapshotLoad: gated exactly like writes (A5), names crossing to persistfile untouched (its validateName is the single closed path-traversal control, T-05-07), errors flowing through the shared mapError — `ErrInvalidSnapshotName` → InvalidArgument; a load replays as ordinary fresh-version writes so the 05-03 tap fires watch events for the replay with no extra wiring
- Harness: config-struct builder (`buildTestServer`) registering `healthpb.RegisterHealthServer(srv, api.Health())` alongside SupKV and launching the resolver — the registration contract 05-06 main copies — with `awaitGateOpen`/`waitFor` helpers, plus `newFileServer` (real persistfile.FileStore over a temp dir) so snapshot tests exercise the production validation rather than a fake
- Seven new tests in `gate_test.go` (TestGate_writesUnavailableReadsOK, _opensOnSync, _opensOnTimeout, TestHealth_reflectsGate, TestSnapshot_createLoadRoundTrip, _invalidName, _gated), stable under `-count=2`; full `make test` and the full-repo strict lint gate green

## Task Commits

Each task was committed atomically:

1. **Task 1: Gate resolver, health wiring, write gating** - `24406ff` (feat)
2. **Task 2: SnapshotCreate/SnapshotLoad RPCs — gated, name-validated (A5, API-07)** - `3231c00` (feat)

## Files Created/Modified
- `internal/grpcapi/gate.go` - ServerOptions, gateUnavailable, Health, gated, RunGate (poll resolver), openGate (single CAS'd transition)
- `internal/grpcapi/gate_test.go` - the seven gate/health/snapshot tests, newGateServer/newFileServer builders, closedGateOptions
- `internal/grpcapi/server.go` - Server gains opts/healthSrv/gateOpen; NewServer(engine, notify, hub, opts) creates the health server and pins NOT_SERVING; Set/Delete gate checks first; the two snapshot handlers; Server doc updated (all seven phase-5 RPCs implemented)
- `internal/grpcapi/server_test.go` - harness refactored over testServerConfig: health client + health registration + resolver launch in buildTestServer; newTestServer/newTestServerWithWatchBuffer wrappers keep their signatures; awaitGateOpen/waitFor helpers

## Decisions Made
- The gate check precedes every validation on writes (API-04 "full stop") — asserted by the empty-key-during-gate row, not just documented
- `openGate` is one `CompareAndSwap` storing the flag and flipping health under the same branch: the resolver's two opening paths can both fire in one select iteration, and the CAS keeps the health flip single-shot
- Snapshot tests use a real `persistfile.FileStore` (the fake DumpStore deliberately does not reproduce name validation — testing the mapping against a fake would have proven nothing about T-05-07)
- TestGate_opensOnTimeout blocks the resolver's first `SyncArrived` poll until the test has observed the closed gate: the plan's naive "assert Unavailable, then wait ~150ms" races the very 50ms deadline under test under full-suite parallel load (found live by `make test`)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Harness refactor in server_test.go (file outside the plan's files_modified)**
- **Found during:** Task 1
- **Issue:** The NewServer signature change does not compile without updating the harness's call site, and acceptance criterion 4 explicitly requires the health server registered on the same grpc.Server as SupKV in the harness — neither reachable without touching server_test.go
- **Fix:** Refactored newTestServer/newTestServerWithWatchBuffer over a testServerConfig builder (buildTestServer) that registers `healthpb.RegisterHealthServer(srv, api.Health())` and launches RunGate; added the health client to testHarness and the awaitGateOpen/waitFor helpers; wrapper signatures kept so watch_test.go needed no change
- **Files modified:** internal/grpcapi/server_test.go
- **Verification:** the entire pre-existing TestServer_*/TestWatch_* suites pass unmodified; grep evidence for the registration recorded in coverage D3
- **Committed in:** 24406ff

**2. [Rule 1 - Bug] TestGate_opensOnTimeout's closed-state assertion raced the deadline under test**
- **Found during:** Plan-level verification (make test)
- **Issue:** The plan's test spec asserts Unavailable before waiting ~150ms for a 50ms backstop; under `make test` parallel race load, harness construction plus scheduling occasionally exceeded 50ms, so the gate had already opened and the assertion failed ("want code Unavailable, got OK")
- **Fix:** the test's SyncArrived closure blocks the resolver's first poll until the test observes the closed state, then returns false forever — the closed-side assertion is deterministic and the opening can only be the backstop's work; stable under `go test -race -count=2`
- **Files modified:** internal/grpcapi/gate_test.go
- **Verification:** make test green end-to-end; focused gate tests green twice in a row
- **Committed in:** 3231c00 (part of task commit)

---

**Total deviations:** 2 auto-fixed (1 blocking-environment adaptation, 1 test-determinism bug fix)
**Impact on plan:** Both fixes are test-infrastructure/harness adaptations forced by the signature change the plan itself specifies; the gate semantics, wire behavior and all acceptance criteria are unchanged. No scope creep.

## Issues Encountered
- Seven strict-lint findings fixed in code direction across the two tasks (unlambda on the `sync.Load` method value; noinlineerr on the new handlers and tests) — no suppressions added; every gate re-run green after the fixes.

## Known Stubs
None new. The `wait_for_replication` fields on Set/Delete remain accepted-but-deferred to plan 05-05 exactly as documented since 05-01 (zero-valued D-08 report fields) — pre-existing, deliberate, and consumed by the next plan.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 05-05's wait loop lands on a fully gated server whose snapshots are live; the D-08 report fields are the only remaining zero-value surface
- 05-06 main wires the gate in one call: `ServerOptions{SyncArrived: adapter.SyncArrived, StartupGate: cfg.StartupGate}`, then `healthpb.RegisterHealthServer(gs, srv.Health())` and `go srv.RunGate(ctx)` — the harness (buildTestServer) is the working reference implementation of that contract
- Shutdown (D-19, 05-06) can flip health to NOT_SERVING independently — the resolver is the only opener, and a cancelled RunGate leaves the gate closed
- No blockers; go.mod/go.sum untouched (T-05-SC holds — no package installs)

---
*Phase: 05-grpc-api-and-binary*
*Completed: 2026-11-09*

## Self-Check: PASSED

All four created/modified files exist on disk; commits 24406ff and 3231c00 are ancestors of HEAD; `commits: 2` is measured from the persisted ledger (`git rev-list --count 2d41087e..HEAD`); plan-level gates green at SUMMARY time: `make test` (full race+cover suite) and `go tool golangci-lint config verify && go tool golangci-lint run ./...` both exit 0; internal/app untouched (gate lives entirely in the adapter; go.mod/go.sum unchanged — T-05-SC).
