---
phase: 03-transport
plan: 01
subsystem: transport
tags: [udp, socket-buffers, getsockopt, xport-05, real-socket, linux]

# Dependency graph
requires: []
provides:
  - Socket-buffer configuration seam (configureSocketBuffers) with request-width validation and getsockopt read-back
  - Effective SO_RCVBUF/SO_SNDBUF reported through Adapter.SocketBuffers and the startup "Socket buffers" log line
  - Startup fail-fast when the effective send buffer cannot hold one maximal datagram (65 507 B)
  - Real-Linux-socket evidence that 1/20/60 KiB values deliver byte-exact
affects: [03-transport remaining plans, observability metrics wiring, docs/darwin statement]

actuals:
  tokens: 7150
  tasks: 3
  commits: 3
  plan_head_before: d7ffe9d010e3e9735f2560dce30d94f61cc3b76d
  plan_head_after: 30cef98286b096e612f31528f3301d6ba8f2d911

tech-stack:
  added: []
  patterns:
    - "Socket buffers: request, read the kernel's effective value back via SyscallConn+GetsockoptInt, log/report only the read-back"
    - "Static startup observability line with typed attrs (slog.Info(\"Socket buffers\", ...)), never the requested constants"
    - "Pure floor predicate (checkSendBuffer) plus construction-time fail-fast closing the socket"

key-files:
  created:
    - internal/udprepl/socket_internal_test.go
    - internal/stand/stand_buffers_test.go
  modified:
    - internal/udprepl/repl.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go

key-decisions:
  - "Only the kernel read-back values are reported: the requested constants never reach the startup log (the fiction XPORT-05 removes)"
  - "The request-width guard is strict: exactly MaxInt32/2 is accepted, anything above it rejected before setsockopt"
  - "Buffer configuration failure is a construction error that closes the bound socket; New never returns a half-configured adapter"
  - "The stand test passes --sync-response-rate explicitly so the delivery bound's paced term derives from a flag the nodes actually run with"

patterns-established:
  - "Effective-value observability: read kernel state back, assert positivity/shape, never assert the runner-specific doubled/clamped numbers"

requirements-completed: [XPORT-05]

coverage:
  - id: D1
    description: "Effective SO_RCVBUF/SO_SNDBUF sizes are read back via getsockopt, exposed through SocketBuffers, and logged once at startup with positive recv_buf/send_buf attrs"
    requirement: XPORT-05
    verification:
      - kind: unit
        ref: "go test -race ./internal/udprepl/ -run '^TestSocketBuffers$'"
        status: pass
      - kind: integration
        ref: "go test -race ./cmd/sup-kv/ -run '^TestMain_socketBuffersLogged$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Startup fails when the effective send buffer is below one maximal datagram, naming both actual and required sizes"
    requirement: XPORT-05
    verification:
      - kind: unit
        ref: "go test -race ./internal/udprepl/ -run '^TestCheckSendBuffer$|^TestSocketBuffers_boundaries$'"
        status: pass
      - kind: other
        ref: "internal/udprepl/repl.go New: checkSendBuffer(sendBuf) returns a construction error after closing the socket"
        status: pass
    human_judgment: false
  - id: D3
    description: "1 KiB, 20 KiB and 60 KiB values are delivered byte-exact between two real binaries on Linux"
    requirement: XPORT-05
    verification:
      - kind: integration
        ref: "go test -race ./internal/stand/ -run '^TestStand_bufferSizesDeliver$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "The floor is pinned at exactly 65 506/65 507 bytes and every request width above MaxInt32/2 is rejected before any kernel call"
    requirement: XPORT-05
    verification:
      - kind: unit
        ref: "go test -race ./internal/udprepl/ -run '^TestSocketBuffers_boundaries$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "darwin's ~9 KiB send limit remains documented in ADR 0010, not fixed"
    requirement: XPORT-05
    verification:
      - kind: other
        ref: "docs/adr/0010-darwin-development-only.md (consumed unchanged; ADR 0010 informs XPORT-05)"
        status: pass
    human_judgment: false

duration: 7min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 1: Socket buffers configured, read back, reported and floor-enforced Summary

**The adapter now requests SO_RCVBUF/SO_SNDBUF, reports the kernel's effective doubled/clamped sizes at startup, refuses to start below the 65 507-byte send floor, and proves 1/20/60 KiB delivery on real Linux sockets.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-11-21T18:16:33Z
- **Completed:** 2026-11-21T18:23:50Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- `configureSocketBuffers` validates the request widths (T-3-01), applies `SetReadBuffer`/`SetWriteBuffer`, reads the effective values back through `SyscallConn` + `getsockopt`, and wraps every failure with the operation and size.
- `New` now fails loudly when the effective send buffer cannot hold one maximal datagram (T-3-03), closing the socket on every configuration error; the effective sizes are stored as immutable state behind `Adapter.SocketBuffers`.
- `wire()` logs one static `Socket buffers` line with `recv_buf`/`send_buf` from the adapter — the kernel's read-back, never the requested constants.
- The stand test drives two real binaries: 1 024-, 20 480- and 61 440-byte values arrive byte-exact on the peer within a bound derived from the running flags; both nodes report positive effective send buffers.
- On this Linux host the read-back shows SO_RCVBUF=8 388 608 (requested 4 MiB) and SO_SNDBUF=2 097 152 (requested 1 MiB) — the kernel's doubling applied to requests clamped at or below `rmem_max`/`wmem_max` (4 MiB each).

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — socket buffers configured, read back, logged at startup, send floor enforced** — `21572bf` (feat)
2. **Task 2: Real-socket delivery of 1, 20 and 60 KiB values on Linux** — `7178124` (test)
3. **Task 3: Buffer boundary, int-width guard and concurrency checks** — `30cef98` (test)

**Plan metadata:** this docs commit (SUMMARY/STATE/ROADMAP/REQUIREMENTS)

## Files Created/Modified

- `internal/udprepl/repl.go` — buffer request constants, `configureSocketBuffers`, `validateBufferRequest`, `socketOption`, `checkSendBuffer`, `Adapter.SocketBuffers`, `New` wiring and doc contract.
- `internal/udprepl/socket_internal_test.go` (new) — real-loopback read-back contract, floor/adjacency tables, request-width tables, kernel read-back logging, concurrent accessor proof.
- `cmd/sup-kv/main.go` — static `Socket buffers` startup log after adapter construction.
- `cmd/sup-kv/main_test.go` — `TestMain_socketBuffersLogged` exec test parsing the JSON startup line and asserting positive sizes.
- `internal/stand/stand_buffers_test.go` (new, `//go:build linux`) — two-binary byte-exact 1/20/60 KiB delivery plus both startup lines.

## Decisions Made

- **Read-back only:** the log and accessor expose the kernel's effective values; the requested constants are never reported, matching XPORT-05's wording and RESEARCH Pattern 7.
- **Strict width boundary:** `validateBufferRequest` accepts exactly MaxInt32/2 and rejects anything above it, because the kernel doubles the request and the read-back must fit a 32-bit int (T-3-01).
- **Construction-time failure:** all buffer-configuration errors (set, read-back, floor) return from `New` after closing the socket — no degraded node starts.
- **Explicit stand rate:** the delivery test passes `--sync-response-rate=500` so its paced bound term is tied to a flag the nodes actually run with.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan Task 3's boundary table contradicted the guard spec**
- **Found during:** Task 3 (buffer boundary, int-width guard and concurrency checks)
- **Issue:** Task 3's action said the last three table rows — including `math.MaxInt32/2` — must reject, while Task 1's implementation contract, Task 1 acceptance criteria and threat register T-3-01 all specify `requested > math.MaxInt32/2` rejects, leaving exactly MaxInt32/2 accepted. Enforcing both is impossible.
- **Fix:** Implemented the primary contract (strict `>`), pinned by `TestSocketBuffers_boundaries/request_width/the_largest_safe_width` (accepted) and `.../one_past_the_largest_safe_width` (rejected), alongside Task 1's reject-table.
- **Files modified:** internal/udprepl/repl.go, internal/udprepl/socket_internal_test.go
- **Verification:** `go test -race ./internal/udprepl/ -run '^TestSocketBuffers_boundaries$'` green; lint green.
- **Commit:** `30cef98`

**2. [Rule 3 - Blocking issue] New stand caller surfaced `unparam` on `startStandLoggedNode`**
- **Found during:** Task 2 lint pass
- **Issue:** `go tool golangci-lint run ./internal/stand/` reported `startStandLoggedNode - extra always receives nil` once the new scenario became another no-extras caller, failing the strict gate.
- **Fix:** Passed the explicit `--sync-response-rate=500` flag through `extra` on both nodes — removing the lint finding and making the delivery bound's paced term derive from a flag the nodes actually run with.
- **Files modified:** internal/stand/stand_buffers_test.go
- **Verification:** `go tool golangci-lint run ./internal/stand/` clean; stand test green.
- **Commit:** `7178124`

**Total deviations:** 2 (both auto-fixed).
**Impact:** No behavioral divergence from the plan's intent; deviation 1 resolves an internal plan contradiction in favor of the threat-model contract, deviation 2 strengthens the test's flag derivation.

## Issues Encountered

- The strict lint gate flagged the new production helpers on first pass (`err113` dynamic errors, `noinlineerr`, `mnd` on `math.MaxInt32/2`). Fixed by adding the two wrapped sentinels (`errBufferRequestRange`, `errSendBufferTooSmall`), a named `maxBufferRequestWidth` constant, and plain-assignment error handling; no suppressions were added.
- `TestSocketBuffers_concurrent` initially used `wg.Add`/`go`; the Go 1.27 `revive`/`modernize` rules required `wg.Go`. Rewritten, race-clean.

## Known Stubs

None.

## Threat Flags

None — the change adds no new network endpoint, auth path, or trust boundary. The two threat-register mitigations T-3-01 and T-3-03 are implemented and pinned by tests.

## User Setup Required

None.

## Next Phase Readiness

- The buffer seam is ready for 03-10's bind/reuse policy work: `reuseAddrBroadcastListenPacket` was deliberately left untouched.
- Remaining Phase 3 plans can rely on `Adapter.SocketBuffers` for startup evidence and on the Linux stand pattern for real-socket acceptance.
- darwin remains development-only per ADR 0010; the delivery test is Linux-tagged and carries no skip on Linux.

---

*Phase: 03-transport*
*Completed: 2026-11-21*

## Self-Check: PASSED

- Files verified: `internal/udprepl/repl.go`, `internal/udprepl/socket_internal_test.go`, `cmd/sup-kv/main.go`, `cmd/sup-kv/main_test.go`, `internal/stand/stand_buffers_test.go` all exist.
- Commits verified: `21572bf`, `7178124`, `30cef98` are ancestors of HEAD.
- Plan verification re-run green: adapter buffer tests, `TestMain_socketBuffersLogged`, `TestStand_bufferSizesDeliver`, targeted lint, `go mod tidy` clean.
- `gsd_tools check evaluation-scope --plan 03-01 --commits-only` reported all three plan commits.
