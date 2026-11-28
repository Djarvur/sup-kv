---
phase: 07-observability-and-configuration
plan: 01
subsystem: replication-adapter-decomposition
tags:
  - obs-v08
  - obs-v07
  - udprepl-split
  - sentbytype-relocation
  - behavior-preserving-move
  - metrics-boundary
requires:
  - phase: 03-transport
    provides: the packetConn read-loop seam, socket-buffer configuration and the real-socket smoke topology
  - phase: 04-protocol
    provides: the one-pointer listing mirror and the per-peer peer-id generation the mirror unit publishes
  - phase: 06-api-and-semantics
    provides: the KnownPeerIDs write-time capture seam and the adapter accessor set the exporter reads
provides:
  - internal/udprepl split into socket.go, codec.go, loop.go, mirror.go and adapter.go in one package
  - internal/app/repl/sentbytype.go so the metrics exporter no longer imports the socket adapter
  - an unchanged udprepl test suite (all twelve files) compiling and passing against the split
affects:
  - 07-02 (loop.go and mirror.go are the later metric/lifecycle edit sites instead of a 1417-line repl.go)
  - 07-03 (the metrics depguard deny rule is now satisfiable)
  - internal/metrics (Sources.Sent is repl.SentByType)
actuals:
  tokens: 31199
  tasks: 2
  commits: 2
  plan_head_before: 31c9fe4d246cf31e5f3061229c131db1db382858
  plan_head_after: 3e560d32503c0db116f5db8c6043e30871de92a4
tech-stack:
  added: []
  patterns:
    - "Same-package file split: socket, codec, event loop and mirror units live in one package so unexported identifiers and every existing test compile unchanged (D-08)."
    - "Alias-plus-accessor compatibility: the moved value type keeps `type SentByType = repl.SentByType` in the adapter while the accessor method stays on *Adapter, so callers need no edits (D-07)."
    - "Composition root: adapter.go holds the package doc, the whole Adapter struct and New, while the moved methods live in their unit files."
key-files:
  created:
    - internal/udprepl/socket.go
    - internal/udprepl/codec.go
    - internal/udprepl/loop.go
    - internal/udprepl/mirror.go
    - internal/udprepl/adapter.go
    - internal/app/repl/sentbytype.go
  modified:
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
  deleted:
    - internal/udprepl/repl.go
key-decisions:
  - "The split is text-only inside one package: no identifier, signature, build tag or init order changed, so the twelve pre-existing udprepl test files are byte-untouched and are the oracle."
  - "The Adapter struct stays whole in adapter.go because Go cannot split one struct across files; only the accessor methods moved to mirror.go."
  - "SentByType moved verbatim to internal/app/repl with the udprepl alias and the pre-existing Adapter.SentByType() accessor both retained — the alias is not the accessor."
  - "logDiscard and SocketBuffers stayed in adapter.go as composition-root concerns (default OnDiscard bridge and construction read-back respectively), matching their call sites."
patterns-established:
  - "Pattern: before adding a depguard deny rule, relocate the shared value type into the core package and leave an alias behind, so the boundary is provable before it is enforced."
requirements-completed: []
coverage:
  - id: D1
    description: "internal/udprepl/socket.go owns socket construction, SO_REUSEADDR option, buffer configuration/read-back and sentinels"
    requirement: OBSV-08
    verification:
      - kind: unit
        ref: "go build ./... && go test -race -count=1 ./internal/udprepl/ ./internal/metrics/ ./internal/app/repl/"
        status: pass
    human_judgment: false
  - id: D2
    description: "internal/udprepl/codec.go owns encodeDatagram, decode, the Body/port mapping and recordVersion"
    requirement: OBSV-08
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ (includes decode_negative_test.go and FuzzDecode seeds)"
        status: pass
    human_judgment: false
  - id: D3
    description: "internal/udprepl/loop.go owns the single-driver Run loop, notifications, trackers, read loop, flush/send accounting and timers"
    requirement: OBSV-08
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ ./internal/metrics/"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestSmoke_twoNodesUDP$' -v"
        status: pass
    human_judgment: false
  - id: D4
    description: "internal/udprepl/mirror.go owns every atomic publication accessor, listingMirror and mirrorStats"
    requirement: OBSV-08
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ (repl_internal_test.go drives mirrorStats and the accessors)"
        status: pass
    human_judgment: false
  - id: D5
    description: "internal/udprepl/adapter.go is the composition root: package doc, complete Adapter struct, New, default discard bridge and SocketBuffers"
    requirement: OBSV-08
    verification:
      - kind: contract
        ref: "test -f internal/udprepl/adapter.go && ! test -f internal/udprepl/repl.go && grep New/Run/mirrorStats/encodeDatagram/listenDatagram location gates"
        status: pass
    human_judgment: false
  - id: D6
    description: "SentByType lives in internal/app/repl and the metrics exporter no longer imports internal/udprepl"
    requirement: OBSV-07
    verification:
      - kind: contract
        ref: "grep -c internal/udprepl internal/metrics/metrics.go == 0; go tool golangci-lint run ./internal/metrics/ ./internal/udprepl/ ./internal/app/repl/ clean"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/metrics/ ./internal/app/repl/"
        status: pass
    human_judgment: false
duration: 9min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 01: udprepl Decomposition and SentByType Relocation Summary

**Split the 1 417-line `internal/udprepl/repl.go` into socket, codec, loop, mirror and adapter units in one package, and moved `SentByType` to the core package so `internal/metrics` no longer imports the socket adapter — with all twelve pre-existing udprepl test files byte-untouched and green.**

## Performance

- **Duration:** 9min
- **Started:** 2026-11-23T06:59:36Z
- **Completed:** 2026-11-23T07:09:03Z
- **Tasks:** 2
- **Files modified:** 9 (6 created, 2 modified, 1 deleted)

## Accomplishments

- `internal/udprepl` is now four single-responsibility units plus a composition root, all in package `udprepl`: `socket.go`, `codec.go`, `loop.go`, `mirror.go`, `adapter.go`. No exported name, signature or behavior changed.
- `internal/app/repl/sentbytype.go` declares `SentByType`; `internal/udprepl` keeps both the `type SentByType = repl.SentByType` alias and the `Adapter.SentByType()` accessor, so every existing test and `cmd/sup-kv`'s method-value wiring compile unchanged.
- `internal/metrics` dropped its `internal/udprepl` import entirely and reads `repl.SentByType` through the `Sources` seam — the prerequisite for plan 07-03's depguard deny rule.
- The real-socket end-to-end smoke test (`TestSmoke_twoNodesUDP`) passes, proving bind → encode → send → receive → decode → core still works across the split.
- The tracer feedback gate ran in auto mode: the end-to-end `<verify>` was re-run and passed before the loop/mirror expansion.

## Task Commits

1. **Task 1: Extract codec and socket units, relocate SentByType to the core package** - `8903b0f` (refactor)
2. **Task 2: Extract loop and mirror units, leaving adapter.go as the composition root** - `3e560d3` (refactor)

**Plan metadata:** docs(07-01): complete udprepl decomposition plan (this commit)

## Files Created/Modified

- `internal/udprepl/socket.go` - packetConn seam, options/Option/WithAddressReuse, listenDatagram, configureSocketBuffers, validateBufferRequest, socketOption, checkSendBuffer, socket constants and sentinels
- `internal/udprepl/codec.go` - encodeDatagram, decode, protoBody, portMessage, recordVersion, maxNodeIDWire/wireIDSize and errNoStandaloneBody
- `internal/udprepl/loop.go` - inboundEvent/localWrite/event constants, tracker machinery, Run, NotifyLocalWrite/NotifyReplayedWrite, applyResults, WaitReplicated, newAdapterWithConn, readLoop, flush/countSent/write and timer helpers
- `internal/udprepl/mirror.go` - Discards, KnownPeers, KnownPeerIDs, UnackedStats, MergeStats, Retransmissions, PacedSends, SendQueue, EgressDropped, SentByType (alias + accessor), SyncArrived, ListedKey, ReadErrors, listingMirror and mirrorStats
- `internal/udprepl/adapter.go` - package doc, the complete `Adapter` struct, `New`, `logDiscard` and `SocketBuffers` (renamed from `repl.go`)
- `internal/app/repl/sentbytype.go` - the `SentByType` value type moved verbatim into the core package (D-07)
- `internal/metrics/metrics.go` - `Sources.Sent func() repl.SentByType`, `trafficFamilies(..., repl.SentByType)`; `internal/udprepl` import removed
- `internal/metrics/metrics_test.go` - `zeroSources` uses `repl.SentByType{}`; the `internal/udprepl` import removed
- `internal/udprepl/repl.go` - deleted (its declarations now live in the five unit files)

## Decisions Made

- **Same package, no subpackages** (D-08): every unexported identifier and all twelve pre-existing test files compile unchanged; the test suite is the behavioral oracle.
- **Struct stays whole**: Go cannot split one struct definition across files, so `Adapter` (including every atomic mirror field) remains in `adapter.go` while the accessor methods moved to `mirror.go`.
- **Alias and accessor are both required**: `SentByType` is a type alias in `udprepl` and `Adapter.SentByType()` is a method; dropping either would break callers or tests.
- **Placement of unlisted declarations**: `logDiscard` (the default `OnDiscard` bridge) and `SocketBuffers` (the construction read-back accessor) stayed in `adapter.go` as composition-root concerns; the plan's split map did not assign them to a unit.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- A stale `gsd-plan-head-before-07-01` ledger pointed at a commit not in this branch's history; it was reset to the plan-start HEAD (`31c9fe4`) before measuring, following the 06-01/06-03 precedent.
- `git` displayed the `git mv repl.go adapter.go` rename pair as `repl.go → loop.go` at commit time because loop.go is the most similar file; the working-tree operation was the planned `git mv` and the resulting file roster matches the plan exactly.
- `requirements.ready-ids` reports OBSV-07 and OBSV-08 blocked: both ids are shared across plans 07-01/02/03/10/11 (#2388 shared-ID gate), so no checkbox was marked here and `requirements-completed` stays empty, following the Phase 6 precedent; 07-11 closes the phase-level checkboxes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The split leaves five focused files; later plans can edit `loop.go`/`mirror.go` for metrics and lifecycle work instead of the former 1 417-line mix. `internal/metrics` no longer imports `internal/udprepl`, so plan 07-03 can add the depguard deny rule. No blockers; 10 plans remain in Phase 07 — ready for 07-02.

## Self-Check: PASSED

- Created files exist: `internal/udprepl/adapter.go`, `socket.go`, `codec.go`, `loop.go`, `mirror.go`, `internal/app/repl/sentbytype.go` — all FOUND.
- Deleted file absent: `internal/udprepl/repl.go` — FOUND absent.
- Commits exist: `8903b0f`, `3e560d3` both ancestors of HEAD.
