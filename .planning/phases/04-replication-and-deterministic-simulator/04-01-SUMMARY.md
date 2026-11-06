---
phase: 04-replication-and-deterministic-simulator
plan: 01
subsystem: replication
tags: [protobuf, udp, udp-broadcast, protoc, crc32, event-loop, replication, crdt, go]

# Dependency graph
requires:
  - phase: 02-engine-hlc-versioning-and-lww-store
    provides: storage.Record and version.RecordVersion — the record shape crossing the new seam
  - phase: 03-persistence-dump-and-write-confirmation
    provides: the port seam precedent (port.DumpStore) this plan's decoded-struct seam deliberately diverges from
provides:
  - port/repl.go decoded-message seam — Message, the five message structs, DecodeStatus (Decoded/BadCRC/Undecodable), InboundDatagram, Outbound, Outbox (Broadcast/Unicast/Drain)
  - internal/app/repl pure core skeleton — New, NewRunID, HandleDatagram (pinned classification order + counters), NotifyLocalWrite (pending entry with defensive copy), Advance (startup Hello), Discards, PeerSnapshot, Config with the eight IDEA.md defaults
  - internal/udprepl adapter — New (SO_REUSEADDR+SO_BROADCAST before bind), Run single event loop, 65507-byte read buffer, CRC-before-body-decode, encode/decode over the wire schema, counters mirrored to atomics
  - internal/udprepl/replproto wire schema (Envelope + Body oneof, all five messages) with committed protoc-gen-go v1.36.12 output
  - Makefile proto target; go.mod tool directive google.golang.org/protobuf/cmd/protoc-gen-go and protobuf promoted to the first direct runtime require (go.sum byte-identical)
affects: [04-02 retransmission and ack drain, 04-03 DataRequest, 04-04 DataSync, 04-05 simulator and chaos, 04-06 classification totality, persist.ApplyRemote and smoke test, phase 05 gRPC write handler, phase 06 observability counters]

# Actuals (#2632) — pairs with the plan's `estimate` (estimateTokens scale: chars/4 over the realized diff).
actuals:
  tokens: 18009
  tasks: 1
  commits: 1

plan_head_before: f0b259475a23be8e7d59f344d31a8e9179f7c125
plan_head_after: 9690b4a4b8396692118a81b6402185f461a1210c

# Tech tracking
tech-stack:
  added: [google.golang.org/protobuf v1.36.12 (first direct runtime require; already pinned in go.sum), protoc-gen-go v1.36.12 (go.mod tool directive), system protoc 3.21.12 (dev machines only)]
  patterns: [decoded-structs port seam (D-01) vs Phase 3's whole-bytes seam, consumer-side StateStore interface (storage.Clock precedent), single-event-loop core driven by handlers with now parameters, Outbox accumulator (results, not callbacks), atomic counter mirroring for cross-goroutine reads]

key-files:
  created:
    - internal/port/repl.go
    - internal/app/repl/core.go
    - internal/udprepl/repl.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/repl_internal_test.go
    - internal/udprepl/replproto/supkv.proto
    - internal/udprepl/replproto/supkv.pb.go
  modified:
    - internal/port/doc.go
    - go.mod
    - Makefile
    - .gitignore

key-decisions:
  - "Own-echo classification keys on run_id equality, not source-handle comparison: an impostor cannot guess a 16-byte random run_id, so same-nodeID+same-runID is own echo and same-nodeID+different-runID is the logged DuplicateNodeID configuration error (plan's pinned order; RESEARCH Pattern 5 modified split)"
  - "Envelope-domain validation (node_id within 16 bits, run_id exactly 16 bytes) classifies Undecodable before the CRC gate — research ASVS V5 field validation; the BadCRC verdict still carries the envelope fields of every on-schema datagram"
  - "go mod tidy on go1.27 extracts a sole direct require into a single-line directive (require google.golang.org/protobuf v1.36.12); the CI tidy gate (go mod tidy && git diff --exit-code) makes that the only shippable form, so the plan's tab-indented grep was verified via its tidy-stable semantic equivalent"
  - "The plan-prescribed //nolint:iface directives were dropped: the iface linter does not fire on Message (marker) or StateStore (two methods), and nolintlint forbids unused directives — zero unplanned nolint in the new code; one sanctioned gosec G115 nolint guards the range-checked uint16 node_id conversion"
  - "NotifyLocalWrite enqueues into a 1024-deep buffer: never blocks on replication progress (D-05), with bounded backpressure (never silent loss) if the loop ever wedges"
  - "Encode/decode stay unexported in the adapter; the external tracer test reaches them through exported test-binary-only helpers in repl_internal_test.go (the plan's sanctioned white-box alternative, dumpfile_internal_test.go precedent)"

patterns-established:
  - "Wire framing: marshaled Body rides inside Envelope.body under a CRC32 IEEE computed over exactly those bytes — CRC gate before body decode is adapter-mechanical (D-02)"
  - "Driver shape: one event loop selects on ctx/inbound/notify/timer, calls the core handler with time.Now() and one shared Outbox, drains through encode+send, mirrors counters — the simulator will drive the same handlers directly (D-04)"
  - "gci import order for the first mixed file: stdlib, third-party (protobuf), localmodule"

requirements-completed: [REPL-03]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Port seam types and wire schema with committed generated code (Envelope/Body oneof, all five messages; Makefile proto target; go.mod tool directive + protobuf promotion, go.sum byte-identical)"
    verification:
      - kind: other
        ref: "go build ./... && go tool golangci-lint run ./internal/port/... ./internal/app/repl/... ./internal/udprepl/... && grep counts (generated header=1, tool directive=1, direct require=1) && git diff --quiet -- go.sum"
        status: pass
    human_judgment: false
  - id: D2
    description: "Pure repl core skeleton: pinned discard classification with per-class counters, opaque-handle peer table with PeerSnapshot, hello timer emitting the startup Hello on first Advance, NotifyLocalWrite pending entry with defensive copy — no locks, no goroutines, no wall clock inside internal/app/repl"
    requirement: REPL-03
    verification:
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestWire_twoNodeHandoff (startup Hello assertion; acceptance greps: 'sync.'=0, 'go func'=0, google.golang.org in internal/app=0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "REPL-03 tracer path: a local write on A broadcast as DataReplication through real protobuf encode, decoded through the port seam, applied on B, Acked unicast immediately after the in-memory apply — and duplicate delivery a no-op that still Acks (REPL-09 adjacency)"
    requirement: REPL-03
    verification:
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestWire_twoNodeHandoff (go test -race ./internal/udprepl/ -run '^TestWire_twoNodeHandoff$')"
        status: pass
    human_judgment: false
  - id: D4
    description: "UDP adapter runtime: SO_REUSEADDR+SO_BROADCAST socket before bind, single event loop with read goroutine and re-armed timer, 65507-byte receive buffer, atomic counter mirroring"
    verification: []
    human_judgment: true
    rationale: "The tracer test covers the adapter's encode/decode paths white-box; the socket/event-loop runtime is exercised only by the real-socket smoke test that lands in sibling plan 04-06 (D-08) — no test in this plan asserts live socket behavior, so a human (or the 04-06 verifier) must sign off."

# Metrics
duration: 21 min
completed: 2026-11-06
status: complete
---

# Phase 4 Plan 01: Replication Tracer Summary

**One write travels node-to-node through a real protobuf datagram — port seam (D-01), core-owned discards (D-02), opaque-handle peer table (D-03), single event loop (D-04), explicit author call (D-05) — and the receiver Acks immediately after the in-memory apply.**

## Performance

- **Duration:** 21 min
- **Started:** 2026-11-06T17:28:54Z
- **Completed:** 2026-11-06T17:50:09Z
- **Tasks:** 1 (tracer)
- **Files modified:** 11 (7 created, 4 modified)

## Accomplishments
- The D-01 port seam in `internal/port/repl.go`: decoded message structs (Message, Hello, DataReplication, Ack, DataRequest, SyncEntry, DataSync), DecodeStatus, InboundDatagram, Outbound and the Outbox accumulator — stdlib + own-module types only, depguard-verified
- The wire schema `internal/udprepl/replproto/supkv.proto` (Envelope + Body oneof with all five messages) plus committed protoc-gen-go v1.36.12 output, a `make proto` dev-only target, and go.mod's first direct runtime require — go.sum byte-identical
- The pure core `internal/app/repl`: pinned discard classification with counters (BadCRC / Undecodable / BadProtoVersion / OwnEcho / DuplicateNodeID, the last logged as a configuration error per REPL-08), the opaque-handle peer table with a copying PeerSnapshot read seam, the hello timer whose zero initial deadline emits the startup Hello on the first Advance, and NotifyLocalWrite recording a defensively copied pending entry (the 04-02 retransmission input) and broadcasting the record
- The `internal/udprepl` adapter: SO_REUSEADDR + SO_BROADCAST set before bind, one 65507-byte receive buffer, a read goroutine feeding the single event loop, CRC-before-body-decode with envelope-domain validation, unicast handle resolution, and discard/peer counters mirrored into atomics after every loop iteration
- `TestWire_twoNodeHandoff` green under `-race`: write → broadcast → encode → decode → apply → unicast Ack to A's handle, byte-equal record on B, duplicate delivery a store-unchanged no-op that still Acks, and exactly one broadcast Hello per core on first Advance

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — port seam, wire schema, minimal core, adapter; one write replicated node-to-node** - `9690b4a` (feat)

**Plan metadata:** (recorded below after this SUMMARY commit)

## Files Created/Modified
- `internal/port/repl.go` - the D-01 decoded-message seam (Message, message structs, DecodeStatus, InboundDatagram, Outbound, Outbox)
- `internal/port/doc.go` - package doc extended with the Phase 4 replication seam paragraph
- `internal/app/repl/core.go` - the pure replication decision core skeleton (D-02/D-03/D-04/D-05)
- `internal/udprepl/replproto/supkv.proto` - wire schema: Envelope, Body oneof, the five messages
- `internal/udprepl/replproto/supkv.pb.go` - committed generated code (protoc-gen-go v1.36.12, protoc 3.21.12)
- `internal/udprepl/repl.go` - the UDP adapter: socket, event loop, encode/decode, atomic mirroring
- `internal/udprepl/repl_test.go` - the two-node tracer test (package udprepl_test)
- `internal/udprepl/repl_internal_test.go` - white-box encode/decode helpers exported for the external test (test binary only)
- `go.mod` - tool directive + protobuf promoted to the first direct runtime require
- `Makefile` - `proto` target (dev machines only)
- `.gitignore` - `.tmp/` (the make proto plugin binary location)

## Decisions Made
- Own-echo vs duplicate-node_id split keys on run_id equality (a 16-byte random value an impostor cannot guess) rather than source-handle comparison, exactly the plan's pinned classification order
- Envelope-domain faults (node_id beyond 16 bits, run_id not 16 bytes) classify Undecodable ahead of the CRC gate — the research ASVS V5 field validation; every on-schema datagram's BadCRC verdict still carries its envelope fields
- The adapter keeps its own no-op-default log function while repl.New normalizes the core's copy (normalized is unexported; the duplication is three lines and keeps the core's Config normalization authoritative)
- Import order for the first mixed-dependency file follows the vendored gci sections: stdlib, third-party, localmodule

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Verify-command adjustment] go.mod direct-require grep form**
- **Found during:** Task 1 (verify block 2)
- **Issue:** The plan's verify greps for `^\tgoogle\.golang\.org/protobuf v1\.36\.12$` (a tab-indented line inside the require block). go mod tidy on go1.27.3 always extracts a sole direct require into a single-line directive (`require google.golang.org/protobuf v1.36.12`); hand-placing the tab-indented or parenthesized form is normalized back by tidy, and CI's tidy gate (`go mod tidy && git diff --exit-code go.mod go.sum`) would fail the shippable tree.
- **Fix:** Verified the semantic intent with the tidy-stable equivalent `grep -cP '^require google\.golang\.org/protobuf v1\.36\.12$' go.mod` (= 1); the other two counts (tool directive = 1, generated-code header = 1) and the go.sum byte-identity pass verbatim.
- **Files modified:** none (verification-only adjustment)
- **Verification:** all counts exactly 1; `git diff --quiet -- go.sum` clean; `go mod tidy` idempotent.
- **Committed in:** n/a (documentation of the verify substitution)

**2. [Rule 1 - Bug] Research Pattern 3 socket-Control sketch does not compile**
- **Found during:** Task 1 (adapter implementation)
- **Issue:** syscall.RawConn.Control's callback returns nothing (`func(fd uintptr)`), but the quoted-verbatim research sketch returns an error from it.
- **Fix:** Canonical capture shape — the closure assigns to an outer `opErr`, the Control method's own error and opErr are wrapped with %w; behavior identical to the probe's intent.
- **Files modified:** internal/udprepl/repl.go
- **Verification:** go build/vet green; lint green; socket options set before bind.
- **Committed in:** 9690b4a

**3. [Rule 1 - Lint-direction] Plan-prescribed //nolint:iface directives are unused**
- **Found during:** Task 1 (lint gate)
- **Issue:** The plan says to carry port.DumpStore's sanctioned `//nolint:iface` on Message and StateStore, but the iface linter does not fire on either (marker interface / two-method interface), and nolintlint rejects unused directives — keeping them would fail `make lint`.
- **Fix:** Removed both (the Phase 01 decision: fix in code direction, never weaken the vendored config). One sanctioned `//nolint:gosec // G115:` with reason guards the range-checked uint16 node_id conversion.
- **Files modified:** internal/port/repl.go, internal/app/repl/core.go
- **Verification:** `go tool golangci-lint run ./internal/port/... ./internal/app/repl/... ./internal/udprepl/...` exits 0.
- **Committed in:** 9690b4a

---

**Total deviations:** 3 auto-fixed (2 Rule 1 code/verify adjustments, 1 Rule 1 lint-direction)
**Impact on plan:** All fixes preserve the plan's semantics; none touch the locked D-01..D-05 decisions. Two plan-unlisted files were added within the plan's own discretion space: `.gitignore` (`.tmp/` for the make proto plugin binary — untracked-file hygiene) and `repl_internal_test.go` (the plan's explicitly sanctioned white-box alternative for reaching unexported encode/decode). No scope creep.

## Issues Encountered
None beyond the deviations above — the tracer test passed green under `-race` on the first full run.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The skeleton is ready for every later Phase 4 plan: 04-02 consumes the pending map (entry with defensively copied record + backoff state already recorded per local write) and turns the Ack ignore into the pending drain; 04-03/04-04 already have DataRequest/DataSync decoded through the port and mapped on the wire; 04-05's simulator can drive the same handlers directly; 04-06 lands the classification totality proof, the persist.ApplyRemote seam (the tracer test's stateStore wrapper marks the seam point) and the real-socket smoke test
- The adapter's live socket behavior (Run loop, SO_REUSEADDR/SO_BROADCAST, buffer sizing) is deliberately test-uncovered until 04-06's smoke test (D-08) — flagged in coverage D4 for human/UAT sign-off
- REPL-03 remains marked incomplete in REQUIREMENTS.md until sibling plans 04-02/04-03/04-06 (which also declare it) finish — the shared-ID gate holds the marking automatically

## Self-Check: PASSED

All seven key-files exist on disk; task commit 9690b4a is an ancestor of HEAD; `go build ./...` green; `go test -race ./internal/udprepl/ -run '^TestWire_twoNodeHandoff$'` green; all five acceptance-criteria greps re-verified (protobuf in internal/app = 0; `sync.` and `go func` in the repl core = 0; lint on the three trees exits 0).

---
*Phase: 04-replication-and-deterministic-simulator*
*Completed: 2026-11-06*
