# Phase 4: Protocol - Research

**Researched:** 2026-11-22
**Domain:** sup-kv replication protocol semantics (UDP, protobuf, gRPC startup gate, deterministic simulator)
**Confidence:** HIGH for the current code and test inventory; MEDIUM for the proposed design constants and the 3-node real-socket topology (see Assumptions Log)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Closed interface until the complete listing (PROTO-01 / R-03 / 14.5)
- **D-01:** The startup gate opens only when the COMPLETE listing has arrived and been processed: `DataSync` gains `SyncID`/`ChunkIndex`/`ChunkCount`; the receiver accumulates parts of a SyncID and marks the listing complete only when all chunks 1..Count processed; `startup_gate` (1 s) remains only as the documented loss-backstop (a node that never receives a complete listing opens on it, per the Spec's open question on explicit re-request). While closed: reads and writes answer `Unavailable`; health stays `NOT_SERVING`; a read of a key known from the listing but whose value has not arrived returns `Unavailable` (14.5(6), accepted); `create_if_absent` for any listed key (live or tombstoned) returns `AlreadyExists`; the HLC updates from each part's max version. — [auto] [Closed interface] Q: how is "complete" recognized? → Selected: SyncID/ChunkIndex/ChunkCount accumulation, gate backstop only on loss (recommended default: the Spec's stated mechanism, review R-03/§14.5). — **Reversibility:** costly — gate semantics ride the RPC contract.
- **D-02:** An **empty-disk node waits for a peer**: new `startup_gate_empty` parameter (default 60 s, flag + `SUPKV_` env twin, validated in the same one-place surface Phase 7 will formalize). A node with data on disk opens via `startup_gate` as today; two empty nodes exchange empty listings and open without waiting for the new parameter; a node whose peer never appears stays closed until the parameter elapses (documented in the node-replacement runbook text, Phase 9's home). — [auto] [Empty-disk start] Q: how long does an empty node wait? → Selected: startup_gate_empty 60 s (recommended default: 14.6/ADR 0006). — **Reversibility:** costly — new public parameter.

#### Peer interactions (PROTO-03/04/05/06/07)
- **D-03:** A stale `DataReplication` (received version ≤ held version) is answered with **exactly one unicast `DataReplication` carrying the newer record and NO `Ack`**; the sender's `wait_for_replication` for the stale version reports `acked: false` (Outbox result semantics from Phase 3). — [auto] [Stale records] Q: ack or answer? → Selected: answer with the newer record, no Ack (recommended default: R-04/third Replication outcome). — **Reversibility:** reversible.
- **D-04:** First `Hello` from an unknown `node_id` triggers **exactly one full listing toward it** (a repeat produces nothing); a peer returning after `peer_timeout` + 1 s receives the neighbor's state within N / `sync_response_rate` plus margin **without** the `sync_interval` term; `TestRestart_newPeerNoFullSync` is rewritten to the new contract. — [auto] [New peer sync] Q: when is a full listing sent? → Selected: first-seen Hello, exactly once (recommended default: R-05/PROTO-04). — **Reversibility:** reversible.
- **D-05:** Local-write notifications with a version not greater than the tracked one are **ignored** (never regress the mirror). — [auto] [Notification order] Q: accept regressive notifications? → Selected: ignore with a counter (recommended default: R-10). — **Reversibility:** reversible.
- **D-06:** A third process under a known `node_id` with **alternating `run_id`s is a named configuration error**: counted, logged at WARN on the Spec-mandated entry, receives no full listing. — [auto] [Duplicate identity] Q: how is a duplicate node_id handled? → Selected: config-error class + no listing (recommended default: R-12). — **Reversibility:** reversible.
- **D-07:** `proto_version` is checked **before body decode**; incompatible → class `bad_proto_version`, WARN rate-limited per peer per interval (the rate-limited WARN helper from 03-08's discard classes). — [auto] [Version mismatch] Q: where is the version checked? → Selected: pre-decode with its own class (recommended default: R-13). — **Reversibility:** reversible.

#### Totality and N nodes (PROTO-08/09)
- **D-08:** Replication is **total**: a write whose dump failed still broadcasts (counted as unconfirmed) and a snapshot load broadcasts its result (with APIS-07's full-rollback semantics, Phase 6). 3+-node behavior and timing documented in the phase artifact; `wait_for_replication` semantics at N > 2 documented there too (Phase 6 folds both into docs/API.md; DOCS-05/APIS-05 own the final wording — handoff recorded in the phase report). Real-socket three-node tests land here. — [auto] [Totality/N] Q: scope of totality + N≥3? → Selected: full totality + documented/tested N≥3 (recommended default: S-02/PROTO-09). — **Reversibility:** reversible.

#### Phase-3 review criticals in this domain (CR-01/CR-02)
- **D-09:** Fix both Phase-3 review criticals here, as explicit tasks: **CR-01** — bound the restart-listing queue and key dedupe by `(node_id, run_id)` instead of the datagram source handle (the D-04 first-Hello path is the same code; a hostile segment host must not accumulate FIFOs/queued chunks, and the per-trigger `Versions()` copy must not scale with store size); **CR-02** — the egress queue carries identity only and builds the message at drain (no pinned record copies, no Lookup before dedupe). Both get regression tests (overlay-style probes converted to ordinary tests where feasible). — [auto] [Review criticals] Q: where do CR-01/CR-02 land? → Selected: this phase (same domain, D-04/D-08 code paths). — **Reversibility:** reversible.

### Claude's Discretion
- Exact SyncID allocation (random vs HLC-seeded), chunk-count arithmetic, which node-replacement doc text lands here vs Phase 9, the unconfirmed-write counter name, the restart-pending state shape.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description (from REQUIREMENTS.md) | Research Support |
|----|------------------------------------|------------------|
| PROTO-01 | The interface stays closed until the full key list arrives (R-03, 14.5). Acceptance: F14 on the stand and in the simulator; before the list B answers `Unavailable` for reads and writes; after it `create_if_absent` returns `AlreadyExists`; a three-part list keeps the interface closed for the first two; after the max-version part `(now + 10 s)` the next local write gets a greater version; `DataSync` carries `SyncID`/`ChunkIndex`/`ChunkCount`; HLC `Update` on each part's max version; health `NOT_SERVING` until complete; listed-but-not-arrived reads return `Unavailable` (14.5(6), accepted). | Patterns 1–3; Verified Facts (wire/port/core/gate); Pitfalls 1–3, 10; Test Impact Map; frozen R-03 analysis; proposed `StateStore.Observe`, listing mirror, and read gating skeletons. |
| PROTO-02 | An empty-disk node waits for a peer longer than a node with data (14.6). Acceptance: new `startup_gate_empty` (default 60 s); a node with data opens via `startup_gate`; two empty nodes exchange empty lists and open without the new parameter. | Pattern 3 (main selection), Verified Facts (main/gate/harness), Pitfalls 1, 11, Open Question 7; proposed `KeyCount` selection and flag/env/validation. |
| PROTO-03 | A stale `DataReplication` is answered with the newer record, not acknowledged (R-04). Acceptance: holds `k@v5`, receives `k@v3` → exactly one unicast `DataReplication k@v5`, no `Ack`; the sender's wait for `v3` reports `acked:false`. | Pattern 4 (stale branch), Verified Facts (`applyRemote`, egress identity), Pitfalls 9, 10, Open Question 1. |
| PROTO-04 | A newly seen or returning peer gets an immediate sync (R-05). Acceptance: first `Hello` from an unknown `node_id` → exactly one full listing, a repeat → nothing; a peer returning after `peer_timeout` + 1 s receives the state within N / `sync_response_rate` plus margin without the `sync_interval` term; `TestRestart_newPeerNoFullSync` rewritten. | Pattern 4 (first-seen/restart), Pattern 5 (shared build), Verified Facts (`handleHello`, `enqueueRestartListing`), Pitfalls 4–6, Test Impact Map (restart_test/sim). |
| PROTO-05 | Local-write notifications never regress the tracked version (R-10). Acceptance: a notification with a version not greater than the tracked one is ignored. | Pattern 4 (notification guard), Verified Facts (`NotifyLocalWrite`), proposed `NotificationsRegressed` counter. |
| PROTO-06 | Duplicate `node_id` with alternating `run_id`s is a named configuration error (R-12). Acceptance: a third process under a known `node_id` is counted and logged as a configuration error; no full listing. | Pattern 4 (seen-run ring), Pattern 5 (listing bounds), Verified Facts (duplicate classifier), Pitfalls 4–5, 8. |
| PROTO-07 | `proto_version` is checked before body decode (R-13). Acceptance: incompatible version → `bad_proto_version` class and a WARN rate-limited per peer per interval. | Pattern 4 (version mismatch), Verified Facts (decode order/core check), Pitfall 12, proposed `DecodeStatus` + per-(node_id,class) WARN helper. |
| PROTO-08 | Every local change replicates, including dump-failed writes and snapshot loads (S-02). Acceptance: a write whose dump failed is broadcast and counted as unconfirmed; a snapshot load broadcasts its result (with APIS-07). | Pattern 7 (totality), Verified Facts (`Engine.Set`/`Restore`/`applyRemote`), Pitfall 13. |
| PROTO-09 | Three-or-more-node behavior is described and tested (14.9). Acceptance: behavior and timing for 3+ nodes documented; real-socket tests with three nodes; `wait_for_replication` semantics at N > 2 in the API docs. | Open Question 3 (topology options), Pattern 5 (N-node traffic), Architectural Responsibility Map, Validation Architecture (three-node tests + `04-MULTI-NODE.md`). |
| D-09 (CR-01/CR-02) | Phase-3 review criticals in the same code paths: bound/dedupe the restart-listing queue by `(node_id, run_id)` and stop the per-trigger `Versions()` copy; make egress identity-only and build at drain. | Patterns 5–6, Pitfalls 3, 7, 15, Test Impact Map, proposed regression probes `TestListing_hostile/sharedBuild` and `TestEgress_identityOnly`. |
</phase_requirements>

## Summary

Phase 4 is a **semantics phase on an existing protocol core**, not a new subsystem. Phase 3 already shipped every mechanism this phase needs: the pure `repl.Core` decision state machine, the `port` message seam, the `Outbox` result stream, the unacked ledger, three deterministic pacers, bounded/deduplicated egress, the wire-validation pipeline, the adapter's atomic mirror, and the grpcapi startup gate bound 1:1 to health. The work is to add a **completeness header** to `DataSync` and derive the gate from it, expose the **listed-key set** to the RPC layer, define the four peer-interaction outcomes (stale record, first-seen peer, regressed notification, alternating identity), move the `proto_version` verdict into the adapter before body decode, make replication total on dump-failed writes and snapshot loads, and fix the two Phase-3 queue defects (CR-01 restart-listing flood, CR-02 pinned record copies).

Two facts dominate planning. First, the **current `syncArrived` flag flips on the first received `DataSync` chunk** and the gate's reads are ungated; both are the R-03 defect the frozen stand test pins. The seam name `SyncArrived` can stay (the frozen grpcapi red test's resolver is parked behind a release channel and therefore passes unmodified once the production signal means "complete listing"), but its **production meaning changes**, and every test that encodes the first-chunk reading must be rewritten. Second, the **first-seen-peer listing changes a large number of test invariants** (`QueuedSends` offsets after a seed Hello, exact emission sequences, the eviction lower-bound assertion in `TestScenario_longPartition`) — this is a mechanical but broad adaptation wave, not a protocol risk.

The primary recommendation: implement in six waves — (1) wire header + proto-version bump + chunker accounting, (2) receiver accumulation + gate/listed mirror + `startup_gate_empty`, (3) peer-interaction semantics + CR-01/CR-02, (4) totality (dump-failed writes, snapshot-load broadcast), (5) simulator/stand test adaptation and the 3-node tests + phase artifact, (6) full gate and the two TSV flips. Keep `SyncArrived` as the injected seam name, publish a **single atomic mirror pointer** carrying `{complete, listedKeys}` so the gate can never open before the keys are visible, and add one explicit `StateStore.Observe(version)` method for the per-part HLC update.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Complete-listing accumulation, gate-complete signal, listed-key set | API/Backend — `internal/app/repl` core | gRPC adapter mirror (`internal/udprepl`) | The core owns every protocol decision (D-02/D-04 Phase 3); the adapter only mirrors core state to atomics. |
| Wire header (`SyncID`/`ChunkIndex`/`ChunkCount`) and pre-body version verdict | UDP adapter/wire — `internal/udprepl` + proto | Core validation | Schema framing and body decode live in the adapter (D-01); the core owns the class/counter and the completeness state. |
| Closed-interface read/write/health answers and listed-key RPC semantics | API/Backend — `internal/grpcapi` | Core-provided listed snapshot | RPC codes are the adapter's contract; the core cannot import gRPC (depguard). |
| `startup_gate_empty` parameter and empty-vs-data selection | Process/config — `cmd/sup-kv` | grpcapi gate resolver | Config parsing/validation lives in main; the gate layer takes one resolved backstop duration. |
| First-seen/restart listings, alternation detection, stale-record answer, notification guard | API/Backend — core | Pacers | All are decision rules over the peer table/ledger; emission stays on the existing pacers. |
| Dump-failed write and snapshot-load broadcast | gRPC API + persist engine | Core notify seam | The API knows the write outcome; the engine knows replayed versions; the core broadcasts. |
| 3+-node tests and timing documentation | Real sockets (`internal/udprepl` in-process adapters) + simulator | Phase artifact | The binary has no address-reuse flag, so the real-socket N=3 test constructs adapters in-process (the smoke-test precedent); the simulator proves semantics at N. |

## Structure of this document

- `## Standard Stack`, `## Package Legitimacy Audit` — no new dependencies.
- `## Verified In-Repo Facts` — the exact current shapes with verbatim quotes; the planner should treat this as the ground truth instead of the Phase-3 review's older line numbers.
- `## Architecture Patterns` — the recommended design, end-to-end.
- `## Common Pitfalls` — the traps this exact codebase will spring, including the frozen-test/harness interaction.
- `## Test Impact Map` — every file that must change and why.
- `## Code Examples` — proposed skeletons.
- `## Open Questions`, `## Assumptions Log`, `## Validation Architecture`, `## Security Domain`.

## Standard Stack

No new runtime or test dependency enters this phase. The phase uses the already pinned stack only.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | `go 1.27.4` (go.mod), toolchain go1.27.4 | Language/runtime | Project pin; Phase 3 code already uses Go 1.25+ `WaitGroup.Go`, `atomic.Pointer`, `testing/synctest`. |
| google.golang.org/grpc | v1.85.0 | gRPC API, health, Watch | Already the API layer; Phase 4 changes handler-level gating only. |
| google.golang.org/protobuf | v1.36.13-0.20261016172727-e9ee2a76ab8e | Wire schema/runtime | Adding `DataSync` fields needs only a regenerate with the pinned plugin. |
| buf (tool directive) | v1.75.0 (`go tool buf --version` → `1.75.0`) | Proto generate/lint/breaking | `make proto`, `make buf-lint`; no protoc install. |
| protoc-gen-go (tool directive) | v1.36.12-devel (`go tool protoc-gen-go --version`) | Go codegen | Pinned by the buf template (`local: ["go", "tool", "protoc-gen-go"]`). |

`[VERIFIED: go.mod:3-17,150]` — quotes: `go 1.27.4`; `google.golang.org/grpc v1.85.0`; `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e`; `github.com/bufbuild/buf v1.75.0 // indirect`.

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| stdlib `sync/atomic` | Go 1.27 | Publish `{complete, listedKeys}` mirror | Single-pointer publication avoids the gate-before-keys race; do not add a mutex to the hot mirror. `[CITED: https://pkg.go.dev/sync/atomic]` — "all the atomic operations executed in a program behave as though executed in some sequentially consistent order." |
| stdlib `hash/fnv` | Go 1.27 | (Only if a deterministic SyncID hash is chosen) | The core already uses FNV-1a for jitter; a monotone counter is simpler and deterministic. |
| proto3 field rules | — | Wire evolution | Adding fields is safe; field numbers must never be reused. `[CITED: https://protobuf.dev/programming-guides/proto3/#updating]` — "Adding new fields is safe." / "Changing field numbers for any existing field is not safe." |

### Alternatives Considered
| Instead of | Could use | Tradeoff |
|------------|-----------|----------|
| `uint64 sync_id` | 16-byte `bytes sync_id` | 16 bytes costs ~10 extra wire bytes per chunk and a larger `chunkOverheadBytes`; the accumulation key already includes `(node_id, run_id)`, so a uint64 is unambiguous across restarts. |
| Core-side listing cache invalidated by a `StateStore.Generation()` | Shared listing build with a max age | A generation method must be added to every `StateStore` fake; the shared build needs no interface change and directly satisfies CR-01's "chunk index into the sorted listing at drain time". |
| Pre-decode version check in the core only | Adapter returns a dedicated `DecodeStatus` | The adapter has the envelope before the body; a core-only check still pays the body decode the review calls out. Keep the core's field check as defence-in-depth for the simulator/direct tests. |

**Installation:** none.

**Version verification:** `go version` → `go version go1.27.4 linux/amd64`; `go tool buf --version` → `1.75.0`; `go tool protoc-gen-go --version` → `protoc-gen-go v1.36.12-devel`. `[VERIFIED: shell, this session]`

## Package Legitimacy Audit

**No external packages are installed in this phase.** The only toolchain interaction is regenerating the committed wire code with the already-pinned `go tool buf` / `go tool protoc-gen-go`; no registry lookup or new module is involved.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| — | — | — | — | — | — | none |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Verified In-Repo Facts

These are the exact current shapes. Line numbers are from the files as read in this session (2026-11-22, post-Phase-3, HEAD `28dcaba`). The Phase-3 review quotes older numbers; use these.

### Wire and port

| Fact | Source | Verbatim |
|---|---|---|
| Replication proto version is 1 | `internal/app/repl/core.go:34-37` | `// SupportedProtoVersion is the wire protocol version this core speaks` … `const SupportedProtoVersion = 1` |
| DataSync has only the entry list today | `internal/udprepl/replproto/supkv/v1/supkv.proto:64-68` | `message DataSync {` / `repeated SyncEntry entries = 1;` |
| Port DataSync has only entries | `internal/port/repl.go:75-79` | `type DataSync struct {` / `Entries []SyncEntry` |
| DecodeStatus set | `internal/port/repl.go:14-26` | `Decoded`, `BadCRC`, `Undecodable` only |
| Decode order: envelope → domain → CRC → body; version not judged | `internal/udprepl/repl.go:1096-1125` | `if crc32.ChecksumIEEE(env.GetBody()) != env.GetCrc32()` … `var body replproto.Body` … `proto.Unmarshal(env.GetBody(), &body)` |
| Adapter never judges proto version; core does | `internal/app/repl/core.go:706-710` | `if in.ProtoVersion != SupportedProtoVersion {` / `c.discards.BadProtoVersion++` |

### Core constants

| Fact | Source | Verbatim |
|---|---|---|
| Egress cap | `internal/app/repl/core.go:97-101` | `// maxEgressEntries bounds the reactive-unicast egress queue` … `maxEgressEntries = 32768` |
| Per-entry wire estimate | `internal/app/repl/core.go:138` | `entryOverheadBytes = 24` |
| Per-chunk fixed estimate | `internal/app/repl/core.go:146` | `chunkOverheadBytes = 64` |
| Discard class tokens (10 today) | `internal/app/repl/core.go:44-61` | `DiscardBadCRC = "bad_crc"`, `DiscardUndecodable = "undecodable"`, `DiscardBadProtoVersion = "bad_proto_version"`, `DiscardOwnBroadcast = "own_broadcast"`, `DiscardDuplicateNodeID = "duplicate_node_id"`, `DiscardEmptyKey = "empty_key"`, `DiscardKeyTooLong = "key_too_long"`, `DiscardTombstoneWithData = "tombstone_with_data"`, `DiscardBadChecksum = "bad_checksum"`, `DiscardBadVersion = "bad_version"` |
| Chunker starts each chunk at the fixed overhead | `internal/app/repl/core.go:1534-1557` | `used := chunkOverheadBytes` … `if len(cur) > 0 && used+est > budget {` |
| Egress identity is `(handle, kind, key)` | `internal/app/repl/core.go:350-359` | `type egressKey struct {` / `handle string` / `kind egressKind` / `key string` |
| Queued unicast currently pins a message | `internal/app/repl/core.go:361-372` | `type queuedUnicast struct {` / `id egressKey` / `msg port.Message` |
| Listing FIFO is per destination handle | `internal/app/repl/core.go:374-383` | `type listingFIFO struct {` / `dst string` / `entries [][]port.SyncEntry` |

### Core behavior (the exact seams Phase 4 changes)

| Fact | Source | Verbatim |
|---|---|---|
| Gate signal flips on any DataSync | `internal/app/repl/core.go:605-611`, `1116-1117` | `syncArrived bool` … `c.syncArrived = true` before the entry loop |
| Unknown node is learned with NO targeted listing | `internal/app/repl/core.go:948-966` | `// an unknown node_id is learned — entry created, no targeted` … `if !discovery { return }` |
| Restart listing is keyed by source handle | `internal/app/repl/core.go:1386-1399` | `func (c *Core) enqueueRestartListing(now time.Time, peerID uint16, handle string)` / `if _, busy := c.listingByDst[handle]; busy {` |
| Every trigger rebuilds the listing from `Versions()` | `internal/app/repl/core.go:1390`, `1513-1520` | `chunks := c.listingChunks()` … `versions := c.store.Versions()` |
| DataReplication applies and Acks (no stale pre-check; only a store error suppresses the Ack) | `internal/app/repl/core.go:993-1019` | `conflicts, drift, err := c.store.ApplyRemote(...)` … `out.Unicast(source, port.Ack{Key: msg.Key, Version: msg.Record.Version})` |
| DataRequest does Lookup before dedupe | `internal/app/repl/core.go:1088-1094` | `rec, ok := c.store.Lookup(key)` … `c.enqueueResponse(now, source, port.DataReplication{...})` |
| DataSync local-newer branch does a second Lookup | `internal/app/repl/core.go:1116-1133` | `rec, ok := c.store.Lookup(entry.Key)` … `c.enqueueResponse(now, source, port.DataReplication{...})` |
| Local notification broadcasts even when stale | `internal/app/repl/core.go:1245-1281` | `// a stale smaller write still broadcasts without displacing` … `out.Broadcast(port.DataReplication{Key: key, Record: own})` |
| Duplicate-node-id log is one line per datagram | `internal/app/repl/core.go:711-733` | `c.cfg.Log(fmt.Sprintf("sup-kv: configuration error: node_id %d ...` |
| Listing chunk emitted without header | `internal/app/repl/core.go:1434-1438` | `out.Broadcast(port.DataSync{Entries: chunk})` / `out.Unicast(dst, port.DataSync{Entries: chunk})` |

### Gate and RPC layer

| Fact | Source | Verbatim |
|---|---|---|
| Gate defaults to 1 s | `internal/grpcapi/gate.go:22-30` | `defaultStartupGate = time.Second`; `gateMessage = "startup gate: waiting for first sync round or startup_gate"` |
| Only writes consult the gate | `internal/grpcapi/gate.go:104-112` | `// Reads never consult it: Get, List and Watch are served from the first moment` … `return !s.gateOpen.Load()` |
| Gate opens on the earlier of `SyncArrived()` and the backstop | `internal/grpcapi/gate.go:124-147` | `case <-backstop.C:` … `case <-poll.C: if s.opts.SyncArrived != nil && s.opts.SyncArrived() {` |
| Get is ungated | `internal/grpcapi/server.go:111-120` | `func (s *Server) Get(...)` has no `s.gated()` call |
| Set/Delete/Snapshot are gated | `internal/grpcapi/server.go:144-147`, `189-192`, `244-247`, `264-267` | `if s.gated() { return nil, gateUnavailable() }` |
| `Engine.Version` exists for the tombstone-visible probe | `internal/app/persist/engine.go:520-525` | `func (e *Engine) Version(key string) (version.RecordVersion, bool)` |

### Engine and storage

| Fact | Source | Verbatim |
|---|---|---|
| Restore returns only an error | `internal/app/persist/engine.go:576` | `func (e *Engine) Restore(name string) error` |
| Replay discards the fresh versions | `internal/app/persist/engine.go:621-636` | `func replayRecord(store *storage.Store, key string, rec storage.Record) error` / `_, _, err := store.Set(key, rec.Data)` |
| Merge is the only clock.Update path | `internal/app/storage/store.go:272-282` | `remoteL, remoteC, _ := rec.Version.Unpack()` / `_, _, drift = s.clock.Update(remoteL, remoteC)` |
| install is the single convergence point | `internal/app/storage/store.go:312-322` | `func (s *Store) install(key string, rec Record)` … `s.records[key] = stored` |
| Store has no version-only clock update | `internal/app/storage/store.go` (whole file) | only `Set`, `SetIfAbsent`, `Delete`, `Merge` reach the clock |

### Config/main

| Fact | Source | Verbatim |
|---|---|---|
| Startup gate flag/env/default | `cmd/sup-kv/main.go:100`, `135`, `342-343` | `envStartupGate = "SUPKV_STARTUP_GATE"`; `defaultStartupGate = time.Second`; `flag.DurationVar(&cfg.startupGate, "startup-gate", envDur(envStartupGate, defaultStartupGate),` |
| Gate wiring in `wire` | `cmd/sup-kv/main.go:532-541` | `SyncArrived: adapter.SyncArrived,` … `StartupGate: cfg.startupGate,` … `go srv.RunGate(apiCtx)` |
| `finalizeConfig` is the validation point | `cmd/sup-kv/main.go:418-435` | `func finalizeConfig(cfg *config, nodeID uint64, envErr error) error` (node-id, required knobs, certs, datagram ceiling) |
| Engine load is synchronous before the gate starts | `cmd/sup-kv/main.go:505-541` | `engine, engineCancel, engineDone := startEngine(cfg, hub)` … `go srv.RunGate(apiCtx)` |

### Harness and frozen tests

| Fact | Source | Verbatim |
|---|---|---|
| Harness node args always pass `--startup-gate=1s` | `internal/stand/stand.go:585-601` | `"--startup-gate=" + standStartupGate.String(),` |
| Frozen R-03 stand test passes `--startup-gate=30s` to B | `internal/stand/stand_red_test.go:616-623`, `770-773` | `redR03StartupGate = 30 * time.Second`; `"--startup-gate="+redR03StartupGate.String(),` |
| Frozen closed-side assertions | `internal/stand/stand_red_test.go:782-808` | `redAssertCode(t, "Get on the replaced node B before any listing ...", err, codes.Unavailable)`; health `NOT_SERVING` |
| Frozen open-side assertions | `internal/stand/stand_red_test.go:810-876` | health `SERVING`; `create_if_absent` → `codes.AlreadyExists`; next local version greater than `maxVersion` |
| Current first-seen test expects no listing | `internal/app/repl/restart_test.go:158-196` | `func TestRestart_newPeerNoFullSync` … `want 0 — no targeted listing before the node is known` |
| grpcapi red R-03 test parks the resolver until after "full" | `internal/grpcapi/red_gate_test.go:54-74`, `112-118` | `release := make(chan struct{})` … `case <-release` … `syncState.Store(redSyncFull); close(release)` |
| Gate unit test currently expects reads OK while closed | `internal/grpcapi/gate_test.go:64-107` | `func TestGate_writesUnavailableReadsOK` … `Get during the gate` expects `codes.NotFound`; List/Watch succeed |
| Stand startup-gate timing test | `internal/stand/stand_metrics_test.go:244-266` | `t.Run("startup_gate_delays_gate_open"` … `"--startup-gate="+standMetricsStartupGate.String()` |
| Red expectation rows | `scripts/red-expectations.tsv` | `TestRed_R03_gateClosedUntilFullList	red`; `TestRed_R03_f14ReplacedNode	red` |

**Key reading of the grpcapi red test:** its fake is `SyncArrived: func() bool { <-release; return syncState.Load() != redSyncNone }`. The resolver's first poll blocks until `close(release)`, which happens only after `syncState.Store(redSyncFull)`. The test therefore **never observes a partial state through the seam**; the partial/full distinction lives only in comments. If the option keeps the name `SyncArrived` and the gate still opens when it returns true, **this frozen test passes unmodified** — no REVISION_CONFLICT, and the "only the TSV flips" contract holds for both R-03 rows.

## Architecture Patterns

### System Architecture Diagram

```
                            ┌──────────────────────────────┐
                            │        Peer datagram         │
                            └──────────────┬───────────────┘
                                           v
                     ┌─────────────────────────────────────────┐
                     │ udprepl.decode (adapter)                │
                     │  envelope unmarshal                     │
                     │  proto_version != Supported? ──yes──┐   │
                     │  domain checks / CRC / body unmarshal│  │
                     └──────────────┬───────────────────────┘  │
                                    v                          v
                     ┌──────────────────────────────┐   port.ProtoVersionMismatch
                     │ port.InboundDatagram + Msg   │          │
                     └──────────────┬───────────────┘          │
                                    v                          │
        ┌───────────────────────────────────────────────────────┴────┐
        │ repl.Core.HandleDatagram (driver goroutine only)           │
        │  status/version/own-echo/duplicate/validation              │
        │       │                                                    │
        │       ├─ DataSync ─> validate header ─> accumulate by      │
        │       │   (node_id, run_id, sync_id) ─> per-part Observe   │
        │       │   ─> complete? ─> listed set + gate signal         │
        │       │   ─> per-entry Version compare ─> egress/DataReq   │
        │       ├─ Hello ─> seen-run ring ─> first-seen/restart      │
        │       │   listing | alternation config error               │
        │       ├─ DataReplication ─> held > incoming ? answer       │
        │       │   held == incoming ? Ack : apply + Ack             │
        │       ├─ DataRequest ─> egress identity (key only)         │
        │       └─ NotifyLocalWrite ─> regression guard ─> ledger    │
        │                                                            │
        │  Outbox: messages + results                                │
        └───────┬───────────────────────────────────────────┬────────┘
                v                                           v
   ┌──────────────────────────┐               ┌────────────────────────────┐
   │ pacers: listing / record │               │ result stream -> trackers  │
   │          / egress        │               │ (wait_for_replication)     │
   └────────────┬─────────────┘               └────────────────────────────┘
                v
   ┌──────────────────────────┐      ┌─────────────────────────────────────┐
   │ udprepl.flush -> wire    │      │ adapter mirror after each event:    │
   └──────────────────────────┘      │  atomic.Pointer{complete, listed}   │
                                     └──────────────┬──────────────────────┘
                                                    v
   ┌──────────────────────────┐      ┌─────────────────────────────────────┐
   │ gRPC client              │─────>│ grpcapi: gate closed? -> Unavailable│
   └──────────────────────────┘      │  open: listed key semantics         │
                                     │  engine -> notify -> core           │
                                     └─────────────────────────────────────┘
```

### Recommended Project Structure (no new packages)

```
internal/app/repl/        # core: completeness state, peer rules, bounds (all in core.go)
internal/port/            # DataSync fields + new DecodeStatus
internal/udprepl/         # pre-body version verdict, new field mapping, mirror publication
internal/grpcapi/         # read gating, listed-key answers, totality notify
internal/app/persist/     # Observe, KeyCount/Len, Restore returning replayed records
internal/app/storage/     # Observe (version-only clock update)
internal/stand/           # harness gate derivation (two-node topologies only)
cmd/sup-kv/               # --startup-gate-empty, selection, ListedKey wiring
```

### Pattern 1: Completeness header + per-stream accumulation (PROTO-01)

**What:** `DataSync` carries `sync_id` (uint64), `chunk_index` (uint32, 1-based), `chunk_count` (uint32), plus the existing entries. The receiver accumulates chunks per `(peer node_id, peer run_id, sync_id)`; the listing is complete exactly when every index `1..chunk_count` has been received. The gate's `SyncArrived` becomes true only on the first complete listing and stays true.

**When to use:** every `DataSync` emission — startup broadcast, periodic broadcast, first-seen targeted, restart targeted. Every round gets a fresh `sync_id`; a rotated round keeps each chunk's original `chunk_index`.

**Design details:**
- `sync_id` is a **monotone uint64 counter** per core, not `crypto/rand`: deterministic for the simulator replay, and the accumulation key includes `run_id` so a restart's reset counter cannot collide.
- Chunker: `chunkSyncEntries` returns the unrotated chunk list; `chunk_count = len(chunks)`, `chunk_index = original position + 1`. Rotation changes only the transmission order, never the indices.
- Accumulation bounds (new core state, attacker-controlled): max concurrent streams (proposed 16), max entries per stream (proposed 65 536), max total accumulated entries (proposed 131 072). On overflow drop the least-progressed stream deterministically (sorted key order) and count it.
- Validation: `chunk_count >= 1`, `1 <= chunk_index <= chunk_count`, `chunk_count <= maxSyncChunks` (proposed 65 536), and a consistent `chunk_count` for an existing stream. A violation is a new discard class (proposed `bad_sync_header`) with its own counter and WARN.
- Per part: after validation, compute the part's max version (`version.Compare`) and call `StateStore.Observe(max)` exactly once. An empty part makes no clock call. Invalid datagrams make no clock call (the pipeline position is the contract).
- On completion: publish the listing's key set for that peer, bump the listing generation, set `listingComplete = true` once, then discard the stream.

**Why the version bump:** adding fields is wire-safe, but a v1.0 DataSync lacks the header, so accepting it would force the old "first chunk means complete" reading — the exact R-03 defect. Mixed versions are already rejected by `proto_version`; raise `SupportedProtoVersion` to 2 (proposed value; the review evidence explicitly says "new `proto_version`" for these fields). `[CITED: docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md:210]`.

### Pattern 2: One atomic listing mirror for the gate and the RPC layer

**What:** the adapter publishes one immutable `listingMirror{complete bool; keys map[string]struct{}}` via `atomic.Pointer`, after each event-loop iteration, only when the core's listing generation changed. `SyncArrived()` loads the pointer and returns `complete`; `ListedKey(key)` loads the same pointer and checks membership.

**Why:** the gate poll and the first RPC can race. Publishing keys and completeness in one pointer, with the pointer store before the gate's next poll, makes "gate open implies listed keys visible" structural. Two separate atomics (bool + pointer) would leave a window. Go atomics are sequentially consistent, so a reader that observes `complete` also observes the pointer that carried it. `[CITED: https://pkg.go.dev/sync/atomic]`.

**Core side:** `ListingGeneration() uint64` (monotone; bumps on completion, peer eviction, and restart-set drop), `ListingComplete() bool`, `ListedKeysSnapshot() map[string]struct{}` (clone). The adapter calls the snapshot only when the generation changes; the clone is O(keys) at listing cadence, never per event.

**Listed-set semantics:** per-peer sets, replaced on each complete listing from that peer; `ListedKey` is the union. Drop a peer's set on eviction and on a detected restart. This makes rule 14.5(6) survive multiple peers and bounded memory (keys × peers; bounded by the stream caps).

### Pattern 3: Gate and RPC semantics (PROTO-01 / 14.5(6))

**What:** while `gated()`:
- `Set`, `Delete`, `SnapshotCreate`, `SnapshotLoad` → `Unavailable` (already, keep first-statement order).
- `Get`, `List`, `Watch` → `Unavailable` (new).
- Health stays `NOT_SERVING` (already bound to the gate).
- `Heartbeat`/reflection are not data RPCs; leave them.

After the gate opens:
- `Get`: local live record → OK; local tombstone → `NotFound`; locally absent and listed → `Unavailable`; otherwise `NotFound`.
- `Set` with `create_if_absent=true`: listed (live or tombstoned) → `AlreadyExists` before the engine call; otherwise the engine's `SetIfAbsent` decides.
- `Set` without `create_if_absent`, `List`, `Watch`: unchanged.
- `Delete` on a listed-but-not-arrived key is not specified by 14.5; recommendation is to keep the current local `NotFound` and document it (see Open Questions).

The local-presence distinction must use `Engine.Version(key)` (tombstone-visible), never `Get` (which collapses tombstone and absence into `ErrNotFound`). `[VERIFIED: internal/app/persist/engine.go:520-525]`.

For the empty-vs-data selection add an O(1) `Store.Len()` / `Engine.KeyCount()` (RLock + `len(records)`); do **not** use `Records()` — it clones every value and would allocate the whole store at startup. `[VERIFIED: internal/app/storage/store.go:327-340]` (Records clones Data).

### Pattern 4: Peer interactions (PROTO-03/04/05/06)

**Stale DataReplication (D-03):**
```
held, present := store.Version(key)
switch {
case present && held.Compare(incoming) > 0:
    enqueue egress DataReplication(handle, key)  // build at drain; no Ack; no Merge
case present && held.Compare(incoming) == 0:
    out.Unicast(source, Ack{key, held})          // equal: Ack only, no reply
default:
    applyRemote(...); on success Ack
}
```
The equal-version Ack is load-bearing: retransmission recovery depends on duplicate/equal deliveries being Acked. D-03's "≤" must be read as the *strictly older* no-Ack rule; S-PROTO-16 says an Ack goes for an applied or already-equal version. Read v5 vs v3 is the acceptance case.

**First-seen and restart (D-04/D-06):**
- Peer row gains a bounded `seenRuns` ring (proposed 8 entries).
- Unknown `node_id` → learn, enqueue exactly one full targeted listing, and (if the table was empty) run the existing discovery flush. **Do not return early on this branch** (unlike the restart branch): the first Hello that ends an empty table must still flush the pending set — `TestRetransmit_suspendAndFlush` and R-02 depend on it.
- Known `node_id`, incoming `run_id ==` current → refresh `lastHello` only.
- Known `node_id`, incoming `run_id` in `seenRuns` (not current) → **duplicate-node-id configuration error**: count `DuplicateNodeID`, WARN (rate-limited), drop the datagram, **no listing**, do not update the row. This is the "alternating run_ids" case and replaces the old "stale run_id is just another change" behavior.
- Known `node_id`, unseen `run_id` → restart: push the old current run into `seenRuns`, update the row, enqueue one full targeted listing, no wait-set change.
- A returning peer after `peer_timeout` is evicted → unknown → first-seen listing. This is the R-05 bound without the `sync_interval` term.
- The first-seen listing uses the same listing machinery as the restart listing (shared build, pacer, caps); no second implementation.

**Notification regression (D-05):** compare the notified version against `store.Version(key)`; if the store already holds a strictly greater version, ignore (no broadcast, no ledger entry) and increment a counter (proposed `NotificationsRegressed`). Equal or absent → proceed as today.

**Version mismatch (D-07):** adapter returns a new `DecodeStatus` (proposed `ProtoVersionMismatch`) immediately after envelope unmarshal and before CRC/body; the core counts `BadProtoVersion`, fires `OnDiscard`, and WARNs at most once per `(node_id, class)` per `discardWarnInterval` (10 s event time). The core's `in.ProtoVersion != SupportedProtoVersion` check stays for simulator/direct inputs.

### Pattern 5: Bounded, shared listing build (CR-01)

**What:** a shared `listingBuild{chunks [][]port.SyncEntry; at time.Time; refs int}` is built lazily from `store.Versions()` on the first listing trigger after idle, reused by every FIFO created while it is fresh (max age = `cfg.SyncInterval`), and freed when the last referencing FIFO drains. Each FIFO stores `{build, syncID, startIndex, emitted}` — O(1) per destination, no per-destination chunk copy.

**Bounds:** max destinations (proposed 32), max queued chunk work (proposed 65 536), drop counter + accessor. A destination's FIFO is keyed by `(node_id, run_id)`; the broadcast round has its own key. A repeat Hello for the same `(node_id, run_id)` finds the FIFO and is a no-op; a rotated round keeps indices.

**Result:** one `Versions()` copy per build age, not per trigger; hostile restart/run-id floods cannot accumulate FIFOs or per-destination chunk copies. This satisfies CR-01's three asks (bound, `(node_id, run_id)` dedupe, drop per-trigger O(store)).

### Pattern 6: Identity-only egress (CR-02)

**What:** `queuedUnicast` becomes `struct{ id egressKey }`; `enqueueResponse(now, handle, kind, key)` never receives a message. At drain, `resolveQueuedResponse` builds `port.DataRequest{Key}` or `DataReplication{Key, Lookup(key)}`; a vanished key skips the slot. `handleDataRequest` uses a `Version` presence probe (no Data copy) before enqueue; `handleDataSync`'s local-newer branch enqueues the identity without a second `Lookup`.

**Why:** the queue is remotely fillable (a broadcast listing shows every key; stale-but-valid listings make a node answer with full values). Identity-only bounds retention by `32 768 × (key + struct)`, not `× value_max_bytes`, and removes the per-duplicate garbage.

### Pattern 7: Totality (PROTO-08)

- **Dump-failed write:** `engine.Set/SetIfAbsent/Delete` return the stamped version plus `ErrDumpFailed` after installing in memory. The gRPC handler notifies the record (and tombstone) before returning `Unavailable`; the write then appears in the unacked ledger like any unconfirmed write. Notify only on `ErrDumpFailed` (not `ErrClosed`, which is shutdown).
- **Snapshot load:** `Engine.Restore` returns the freshly replayed `(key, version, data, deleted)` list (proposed `[]RestoredRecord`) in deterministic sorted-key order, even when the final flush fails; the gRPC handler notifies each with `track=false`. A no-op restore returns an empty list. Phase 6 APIS-07 builds full rollback on the same return shape.

**N≥3 content for `04-MULTI-NODE.md` (handoff material for Phase 6's `docs/API.md`):**
- **Traffic shape:** one broadcast listing per `sync_interval` reaches every peer in one datagram per chunk; each first-seen/restart peer also gets one unicast targeted listing. A joining node therefore receives up to N−1 targeted listings and N−1 periodic listings, but a sender's listing work is one chunk stream, not N copies. Replies (`DataRequest`/`DataReplication`) are paced on each node's single egress queue at `sync_response_rate`.
- **Timing:** normal write visibility is one RTT (the write broadcast); a partition heal without eviction converges within the existing `2 × sync_interval + response` bound; a heal after eviction converges at the next Hello plus `N / sync_response_rate` (first-seen listing, no `sync_interval` term).
- **`wait_for_replication` at N > 2:** `total` is the number of peers known at write time; `acked=true` requires every one of them to Ack (a single unreachable peer yields `acked=false` at the timeout even if N−1 Acked). An Ack from a peer not known at write time counts in the core set but is clamped to `total` (WR-01). Known limitation IN-05: a peer discovered after the write-time snapshot but before the core processes the notification joins the core wait-set but not `total`, so a write fully covered by the write-time peers can report `false` at timeout — the safe direction; Phase 6 APIS-01 may tighten it.

### Anti-Patterns to Avoid

- **Gating reads by store emptiness or by "any DataSync received".** The gate must key on listing completeness; otherwise F14 returns.
- **Using `engine.Get` to test local presence** for listed-key semantics — it hides tombstones.
- **Publishing `complete` before the listed keys.** One pointer, one publish.
- **Keying listings by the source handle.** A source port is attacker-chosen; key by `(node_id, run_id)`.
- **Copying `Data` into the egress queue.** Build at drain.
- **Random SyncID.** It breaks same-seed simulator replay; a counter is sufficient with `run_id` in the key.
- **Treating an equal-version DataReplication as stale.** Equal must Ack or retransmissions never drain.
- **Bumping the proto version without the pre-decode check.** A future version's body may not follow this version's CRC/body contract; the verdict must precede CRC/body.
- **Rotating chunks without keeping their original indices.** Completeness would never be recognized.
- **Wall-clock reads in the core.** WARN limits, stream bounds and max-age use the handler's `now`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Wire field evolution | Custom versioned binary format | proto3 + `make proto` + `buf breaking` | Unknown-field skipping, field-number safety; the repo already commits generated code. |
| Chunk assembly | Reassembly buffers with cursors/continuation state | `chunkSyncEntries` + `chunk_index/chunk_count` header | The chunker already packs to a conservative budget; completeness is the only new state. |
| HLC absorption | A second clock in the repl core | `StateStore.Observe` → `storage.Store` → `clock.Update` | The clock is the single wall-clock seam; the store is the composition point. |
| Atomic publication | A mutex around the mirror | `atomic.Pointer` of an immutable snapshot | Sequentially consistent; no lock on the event-loop hot path. |
| Deterministic IDs | `crypto/rand` in the core | Monotone counters + FNV where a hash is needed | Same-seed replay must stay byte-identical. |
| Peer/listing flood defense | Ad-hoc goroutine timeouts | Existing cap+drop+count pattern (`maxEgressEntries`, `EgressDropped`) | One bounded-queue idiom across the core. |
| 3-node transport test | A bespoke socket framework | The in-process smoke-test shared-port recipe (`udprepl.New(..., WithAddressReuse())`) plus the simulator | Reuses the real codec/socket path and the capability probe; the production binary has no reuse flag by design. |

**Key insight:** this phase adds *state*, not mechanisms. Every mechanism (pacer, queue, ledger, discard class, gate resolver) already exists with tests; new code should extend those seams rather than introduce parallel ones.

## Common Pitfalls

### Pitfall 1: The frozen stand R-03 test vs the harness's new empty-gate default
**What goes wrong:** the stand harness's `nodeArgs` always passes `--startup-gate=1s`; the frozen test passes `--startup-gate=30s` (later flag wins) to keep B's timer from racing the closed-side assertions. If the harness also appends a *default* `--startup-gate-empty=1s`, B (empty disk) opens after 1 s and the frozen closed-side `Get` returns `NotFound` instead of `Unavailable` under load.
**Why it happens:** D-02 makes the empty gate a separate parameter; the frozen test predates it and only names `startup-gate`.
**How to avoid:** `startStandNode` must derive the empty gate from the extras: scan for `--startup-gate=<v>` and append `--startup-gate-empty=<v>`; otherwise use `standStartupGate`. This makes the frozen test's 30 s govern both timers and keeps every other stand test at 1 s. Do not edit `stand_red_test.go`.
**Warning signs:** `TestRed_R03_f14ReplacedNode` fails at the first `Get` with `NotFound` only under parallel load; conversely, the default stand suite hangs at `awaitServing` if the harness forgets the small empty-gate default for non-R-03 tests.

### Pitfall 2: Gate-before-listed-keys race
**What goes wrong:** the resolver observes `complete=true` before the listed keys are visible; a `create_if_absent` in that window succeeds locally and overwrites the peer value (F14 returns through the back door).
**Why it happens:** two separate atomics or a mirror that sets the bool before copying the map.
**How to avoid:** publish one immutable `{complete, keys}` pointer per generation; the resolver and `ListedKey` load the same pointer. Assert with a test that calls `create_if_absent` immediately after health flips.
**Warning signs:** a flaky `TestRed_R03_f14ReplacedNode` only; a consistent unit test with an injected `ListedKey` passes.

### Pitfall 3: Chunk header overhead breaks the exact-boundary fixtures
**What goes wrong:** `chunkOverheadBytes = 64` is exactly accounted by the graduated-key fixtures (sum of entry estimates 1336 + 64 = 1400). Adding a header (~7-23 wire bytes) makes every budgeted chunk exceed `sync_chunk_bytes`, and the exact-boundary fixture closes early.
**Why it happens:** the fixed per-chunk estimate is load-bearing in both `TestSync_chunking` and `TestSync_chunkWireBudget`.
**How to avoid:** raise `chunkOverheadBytes` (proposed 96), re-derive the graduated lens (sum(len) must be 1400 − 96 − 8×24 = 1112, e.g. subtract 4 from each of the eight lengths), and keep the real-codec cross-seam test as the proof. Do not weaken the budget assertion.
**Warning signs:** `TestSync_chunkWireBudget` reports a chunk over 1400; `TestSync_chunking` produces a different chunk count.

### Pitfall 4: First-seen listings silently shift test queue counts
**What goes wrong:** a seed Hello now enqueues a listing (one empty chunk at minimum), so `QueuedSends()` assertions after `out.Drain()` are off by the listing, and `advanceDrain` returns an extra `DataSync`.
**Why it happens:** the egress, record-send and listing queues are separate from the Outbox; `out.Drain()` does not drain them.
**How to avoid:** add a `seedPeer`/`drainListing` helper that Hellos and then `advanceDrain`s (or drains the discovery listing) wherever the test's subject is not the listing; filter by message type where the test already does (`dataSyncs`, `broadcastRepls`, `unicasts`). Audit `discard_test.go`'s `wantQueued` rows and `retransmit_test.go`'s storm count.
**Warning signs:** dozens of tests fail only on `QueuedSends()`/total-emission counts, not on behavior.

### Pitfall 5: Returning early on the first-seen Hello skips the discovery flush
**What goes wrong:** adding `enqueueFirstSeenListing(...); return` mirrors the restart branch, but the first Hello that ends an empty peer table must also flush every pending write (REPL-02/A6). The two behaviors are mutually exclusive only for restarts; a first-seen Hello can be both.
**Why it happens:** the restart branch returns because `discovery` is necessarily false there; copying that shape into the unknown branch silently drops the flush.
**How to avoid:** enqueue the listing, then fall through to the existing `if !discovery { return }` + flush block. `TestRetransmit_suspendAndFlush` is the regression pin; R-02's drain bound also depends on it.
**Warning signs:** R-02 drain goes red; `TestRetransmit_suspendAndFlush` emits only the listing and no `DataReplication`.

### Pitfall 6: The evicted-partition lower bound in the simulator
**What goes wrong:** `TestScenario_longPartition/peerEvicted` asserts the cluster does **not** converge before the first post-heal sync tick. With first-seen listings, the evicted peer's next Hello triggers an immediate listing, so convergence lands before the tick and the assertion fails.
**Why it happens:** the test encodes the v1.0 "only a periodic listing heals an evicted peer" contract.
**How to avoid:** rewrite the evicted subtest to the D-04 contract: convergence at/after the next Hello and before the next sync tick; keep the surviving-table subtest as-is. Update the comment and `scnReplay` message.
**Warning signs:** only the `peerEvicted` subtest fails, at the `usedSyncTerm` check.

### Pitfall 7: Mutant kill seeds may drift
**What goes wrong:** `scripts/red-runner`'s mutation stage requires "sync without tombstones" and "periodic sync disabled" mutants to be killed on fixed seeds. First-seen listings provide an independent reconciliation path, so the periodic-sync mutant's sensitivity can change; the runner treats a surviving mutant as an error.
**Why it happens:** the kill lists were verified against Phase-3 behavior (documented in the script comments).
**How to avoid:** re-run the full `make red-run` after the behavior change; if a mutant survives, re-select kill seeds (or strengthen the mutant to disable both listing paths) and record the new lists in the script. Never relax the control arm.
**Warning signs:** red-runner fails at the mutant stage while all seven rows pass.

### Pitfall 8: New SyncID accumulation is itself a DoS surface
**What goes wrong:** a peer can send many distinct SyncIDs with partial chunks; an unbounded accumulator map grows with attacker input, and the gate never opens (so the node stays closed).
**Why it happens:** completeness requires retained partial state; the review's CR-01 threat applies to the new state too.
**How to avoid:** cap streams, entries per stream and total entries; drop deterministically and count; optionally expire streams/peer rows. Add a regression test mirroring the CR-01 probe (many partial SyncIDs → bounded memory, gate still opens on the backstop).
**Warning signs:** memory grows under `FuzzDecode`-style listing floods; a unit test needs minutes to drain.

### Pitfall 9: Equal-version records
**What goes wrong:** implementing D-03's "≤" literally makes an equal-version DataReplication draw no Ack. Duplicate deliveries are the normal loss-recovery path; the sender's ladder never drains and R-02 regresses.
**Why it happens:** the phrase "received version ≤ held version" includes equality; S-PROTO-16 says an equal version is Acked.
**How to avoid:** strictly older → answer + no Ack; equal → Ack, no reply; strictly newer → apply + Ack. Pin all three with a table test, including the "duplicate redelivery still Acks" row (`internal/app/repl/retransmit_test.go:670`).
**Warning signs:** R-02 drain red; `TestRetransmit_...duplicate redelivery` fails; retransmission counts climb.

### Pitfall 10: `Get` vs `Version` for tombstone visibility
**What goes wrong:** the listed-key logic uses `engine.Get`, which returns `ErrNotFound` for a tombstone; a tombstoned listed key is then treated as "not arrived" and returns `Unavailable` forever instead of `NotFound`.
**Why it happens:** the API's Get intentionally hides tombstones; the replication view does not.
**How to avoid:** use `engine.Version(key)` to distinguish tombstone from absence in the Get handler; use `Lookup` only where the record is needed.
**Warning signs:** Get on a deleted key returns `Unavailable` after the delete arrived.

### Pitfall 11: Startup-gate test sprawl in `cmd/sup-kv`
**What goes wrong:** a fresh `cmd/sup-kv` test node has an empty dump; with `startup_gate_empty` defaulting to 60 s it never becomes SERVING inside `servingDeadline` (10 s). Fourteen sites pass `--startup-gate=10ms` and will hang/fail.
**Why it happens:** the empty-vs-data selection is new; those tests only tuned the old uniform gate.
**How to avoid:** add `--startup-gate-empty=10ms` beside every `--startup-gate=10ms` in `cmd/sup-kv/main_test.go`. Keep the production default at 60 s; do not special-case "explicit startup-gate also sets empty" (it would contradict ADR 0006's separate parameter).
**Warning signs:** `TestMain_precedence` and the single-node lifecycle tests time out at `awaitServing`.

### Pitfall 12: Pre-decode version verdict changes the classification order
**What goes wrong:** a datagram with a future version and a corrupt body now counts `bad_proto_version`, not `bad_crc`/`undecodable`. Existing negative-table rows and docs that pin "BadCRC is always first" fail.
**Why it happens:** the review's R-13 fix checks the version immediately after envelope unmarshal, before CRC, because a future version may not follow this version's body/CRC contract.
**How to avoid:** make the order explicit and pin it with a combined row (future version + corrupted body): `bad_proto_version`. Update `docs/OPERATIONS.md`'s class discussion as a handoff (DOCS owns the final catalog).
**Warning signs:** `TestDecode_negativeTable` fails on the wrong-version row (it currently expects `port.Decoded` and a decoded `Hello`).

### Pitfall 13: `Restore`'s signature change ripples
**What goes wrong:** changing `Engine.Restore(name) error` to return replayed records breaks ~12 call sites, including tests that only check the error; the notify loop might also run on a failed/no-op restore.
**Why it happens:** totality needs the fresh versions the replay assigned; the current signature discards them.
**How to avoid:** return the records in deterministic sorted-key order (empty on a no-op), update call sites with `_, err :=`, and notify only what was replayed. Take the notify path even when the final flush fails (the state is in memory), but not on `ErrClosed`.
**Warning signs:** snapshot tests fail to compile; a dump-failed snapshot load does not broadcast.

### Pitfall 14: The giant `core.go` and the strict lint gate
**What goes wrong:** new structs/handlers push `core.go` over function-length/complexity limits; new casts need justified `//nolint:gosec // G115` comments; a new `DecodeStatus` value can trip an exhaustive switch.
**Why it happens:** `core.go` is already 2 108 lines and every new decision lands there (the depguard boundary forbids moving I/O-bound code in).
**How to avoid:** keep new logic in small, single-purpose methods; reuse `discardWarnInterval`/`discardValidated`; run `make lint` per wave; add the new status case to every switch the `exhaustive` linter sees (core and the negative-table helper).
**Warning signs:** the lint gate fails on funlen/cyclop/exhaustive after a wave.

### Pitfall 15: Determinism creep
**What goes wrong:** a `crypto/rand` SyncID, a wall-clock rate limit, or map-order iteration in the "drop least-progressed stream" path changes same-seed replay.
**Why it happens:** the simulator replays by seed and compares a trace digest (`TestChaos_determinismReplay`, `TestSim_determinism_withExtensions`); behavior changes are allowed, nondeterminism is not.
**How to avoid:** counters for IDs, event time for limits, sorted keys for any eviction/emission decision; run both determinism tests after each core wave.
**Warning signs:** same seed two different digests; `TraceDigest` length changes between identical runs (length is fine; inequality is not).

## Test Impact Map

The planner should budget a full wave for this table; most items are mechanical, a few encode changed semantics.

| File/area | Why it changes | Kind |
|---|---|---|
| `internal/app/repl/sync_test.go` | `syncFrom` helper must stamp a valid complete header; `TestSync_syncArrived` becomes "partial does not flip, complete flips"; `TestSync_chunking`/`TestSync_emptyListing`/`TestSync_pacing`/`TestSync_rotation`/`TestSync_listingBusy` assert header-aware chunks; `dataSyncsEqual` may need a header-aware variant for the new wire tests | semantic + mechanical |
| `internal/app/repl/restart_test.go` | `TestRestart_runIDChangeTriggersFullSync` learn-Hello now emits a listing; `TestRestart_newPeerNoFullSync` is rewritten to "first Hello = exactly one listing, repeat = nothing"; `TestRestart_staleRunIdAndInFlight`'s stale-return case becomes an alternation config error (no listing) | semantic |
| `internal/app/repl/discard_test.go` | Seed Hellos enqueue listings (`wantQueued` offsets); DataSync fixtures need headers; new `bad_sync_header` rows; pre-decode status case in the helper switches | mechanical + new |
| `internal/app/repl/retransmit_test.go` | Seed Hellos enqueue listings; stale DataReplication now draws a reply/no Ack; equal still Acks; storm `QueuedSends` count may shift | semantic + mechanical |
| `internal/app/repl/peertable_test.go` | Re-Hello after eviction adds a first-seen listing (filtered out today; fine) and the seen-run ring needs its own tests | minor |
| `internal/app/repl/bench_test.go` | DataSync benchmark must send a complete valid listing; `Versions`/`Lookup` accounting for CR-01/CR-02 probes | mechanical |
| `internal/app/repl/sim_*.go` | First-seen listings alter discovery traffic; `TestScenario_emptyDisk` comment/bound; `TestScenario_longPartition/peerEvicted` lower bound; mutant seed re-verification; determinism stays green | semantic |
| `internal/app/repl/egress_internal_test.go` | `queuedUnicast.msg` disappears; identity-only drain | mechanical |
| `internal/udprepl/repl.go` + tests | New fields, new `DecodeStatus`, pre-body check; `TestDecode_negativeTable` wrong-version row; `TestMirror_syncArrivedFlip` complete-listing fixture; fuzz seeds; `TestDiscardLog_classes` still counts OnDiscard | semantic |
| `internal/udprepl/syncsize_internal_test.go` | Overhead constant and graduated lens; wire budget proof | mechanical |
| `internal/udprepl/scenario_test.go` / `smoke_test.go` / `wait_integration_test.go` | `StateStore.Observe` delegation; bad-proto scenario still counts | mechanical |
| `internal/grpcapi/gate_test.go` | `TestGate_writesUnavailableReadsOK` becomes "reads and writes unavailable"; add listed-key + completeness tests | semantic |
| `internal/grpcapi/server_test.go` / harness | `ListedKey` option; notify capture for totality tests | mechanical |
| `internal/grpcapi/red_gate_test.go` | **No edit expected** (resolver parked until full); keep the row red→green | none |
| `internal/grpcapi/wait_integration_test.go` | `StateStore.Observe` on `peerStore` | mechanical |
| `internal/app/persist/engine.go` + tests | `Observe`, `KeyCount`, `Restore` return shape; snapshot tests | semantic |
| `internal/app/storage/store.go` + tests | `Observe`; no other behavior change | new method |
| `internal/app/internal/simnet/simnet.go` + ext tests | `nodeStore.Observe`; extra listings in budgets/digests | mechanical |
| `internal/stand/stand.go` | Empty-gate derivation (the three-node real-socket test lives in `internal/udprepl`, not here — the binary has no address-reuse flag) | harness |
| `internal/stand/stand_test.go` / `stand_metrics_test.go` | Derived empty gate keeps them working; `TestStand_emptyDiskRecovery` comment/bound | semantic |
| `internal/udprepl/smoke_test.go` + new three-node test | In-process three adapters on one shared SO_REUSEADDR port; broadcast-convergence assertions | new |
| `cmd/sup-kv/main.go` + `main_test.go` | New flag/env/validation/selection; 14 `--startup-gate-empty` test sites; selection unit test | semantic + mechanical |
| `internal/metrics/metrics.go` + tests | Optional `bad_sync_header` label if the new class is published | mechanical |
| `scripts/red-expectations.tsv` | Two R-03 rows `red` → `green` | one-line |
| `docs/OPERATIONS.md` | Discard class list/flag docs (handoff; Phase 9 owns the final docs) | handoff |

## Code Examples

> Values marked `proposed` are recommendations, not verified constants. The planner must choose and record them.

### Wire schema (proposed addition)

```proto
// internal/udprepl/replproto/supkv/v1/supkv.proto
message DataSync {
  repeated SyncEntry entries = 1;  // verified existing field
  uint64 sync_id = 2;              // proposed: fresh per listing round
  uint32 chunk_index = 3;          // proposed: 1..chunk_count
  uint32 chunk_count = 4;          // proposed: number of chunks in the round
}
```
`[VERIFIED: internal/udprepl/replproto/supkv/v1/supkv.proto:66-68]` for the existing `entries = 1`.

### Port seam

```go
// internal/port/repl.go (proposed additions)
type DataSync struct {
    SyncID     uint64     // proposed
    ChunkIndex uint32     // proposed, 1-based
    ChunkCount uint32     // proposed
    Entries    []SyncEntry
}

const ProtoVersionMismatch DecodeStatus = 3 // proposed; Decoded/BadCRC/Undecodable are 0..2
```
`[VERIFIED: internal/port/repl.go:14-26, 75-79]` for the existing status values and `Entries`.

### Core accumulation sketch (proposed)

```go
// internal/app/repl/core.go (proposed)
type syncStreamKey struct {
    peer uint16
    run  [16]byte
    id   uint64
}

type syncStream struct {
    count   uint32
    got     map[uint32]struct{}
    entries map[string]version.RecordVersion
    max     version.RecordVersion
}

// handleDataSync(now, in, msg) after validateSync passes the header:
// 1. if len(entries) > 0 { max := maxVersion(entries); store.Observe(max) }
// 2. accumulate by (in.NodeID, in.RunID, msg.SyncID), enforcing the caps
// 3. if all indices 1..count received:
//      c.listedByPeer[in.NodeID] = stream.entries keys
//      c.listingGen++
//      c.listingComplete = true
//      delete(c.syncs, key)
// 4. per-entry Version compare and egress enqueue (existing logic)
```
Every in-repo value used above is in the Verified Facts table (`syncArrived`, `Versions`, `Version`); the names/limits are proposed.

### StateStore addition

```go
// internal/app/repl/core.go StateStore interface (proposed method)
Observe(ver version.RecordVersion) (drift time.Duration)

// internal/app/storage/store.go (proposed)
func (s *Store) Observe(ver version.RecordVersion) time.Duration {
    l, c, _ := ver.Unpack()
    _, _, drift := s.clock.Update(l, c)
    return drift
}
```
`[VERIFIED: internal/app/storage/store.go:84-87]` for the `Clock` interface with `Update(remoteL int64, remoteC uint16)`.

### Adapter mirror publication (proposed)

```go
// internal/udprepl/repl.go (proposed)
type listingMirror struct {
    complete bool
    keys     map[string]struct{}
}

// event loop only:
if gen := a.core.ListingGeneration(); gen != a.listingGen {
    a.listingGen = gen
    a.listing.Store(&listingMirror{
        complete: a.core.ListingComplete(),
        keys:     a.core.ListedKeysSnapshot(),
    })
}

func (a *Adapter) SyncArrived() bool {
    m := a.listing.Load()
    return m != nil && m.complete
}

func (a *Adapter) ListedKey(key string) bool {
    m := a.listing.Load()
    if m == nil || !m.complete {
        return false
    }
    _, ok := m.keys[key]
    return ok
}
```
`[CITED: https://pkg.go.dev/sync/atomic]` for the pointer-then-flag ordering guarantee.

### Gate/RPC changes (proposed)

```go
// internal/grpcapi/server.go
func (s *Server) Get(_ context.Context, req *apiproto.GetRequest) (*apiproto.GetResponse, error) {
    if s.gated() {
        return nil, gateUnavailable()
    }
    // Keep the existing empty-key InvalidArgument check here (before the
    // engine call); key length stays permissive on reads.
    rec, err := s.engine.Get(req.GetKey())
    if err != nil {
        if errors.Is(err, storage.ErrNotFound) {
            if _, present := s.engine.Version(req.GetKey()); present {
                return nil, status.Error(codes.NotFound, storage.ErrNotFound.Error())
            }
            if s.opts.ListedKey != nil && s.opts.ListedKey(req.GetKey()) {
                return nil, status.Error(codes.Unavailable, "startup gate: key is listed by a peer but its value has not arrived")
            }
            return nil, status.Error(codes.NotFound, storage.ErrNotFound.Error())
        }
        return nil, mapError(err)
    }
    return &apiproto.GetResponse{Record: recordToProto(req.GetKey(), rec)}, nil
}

// Set, create_if_absent branch (after gate + key validation):
if req.GetCreateIfAbsent() && s.opts.ListedKey != nil && s.opts.ListedKey(req.GetKey()) {
    return nil, status.Error(codes.AlreadyExists, storage.ErrAlreadyExists.Error())
}
```
`[VERIFIED: internal/grpcapi/server.go:111-120, 144-147]` for the current Get/Set shapes; `[VERIFIED: internal/grpcapi/gate.go:104-112]`.

### main: empty-vs-data gate selection (proposed)

```go
// cmd/sup-kv/main.go (proposed)
gateBackstop := cfg.startupGate
if engine.KeyCount() == 0 {
    gateBackstop = cfg.startupGateEmpty
}
srv := grpcapi.NewServer(engine, adapter.NotifyLocalWrite, hub, grpcapi.ServerOptions{
    SyncArrived: adapter.SyncArrived,
    ListedKey:   adapter.ListedKey,
    // ...
    StartupGate: gateBackstop,
})
```
`[VERIFIED: cmd/sup-kv/main.go:532-541]` for the existing option set; `[VERIFIED: cmd/sup-kv/main.go:135]` for `defaultStartupGate = time.Second`.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| Gate opens on the first DataSync chunk | Gate opens on a complete `SyncID` listing | Phase 4 (this phase) | The R-03 defect and F14 overwrite disappear; requires the wire header and proto bump. |
| Reads served during the startup gate | Reads and writes both `Unavailable` while closed | Phase 4 | R-03's full-interface contract; changes `TestGate_writesUnavailableReadsOK` and the RPC docs. |
| First-seen peer waits for the periodic round | First-seen peer gets an immediate targeted listing | Phase 4 | R-01/R-05 and empty-node recovery; broad test-adaptation wave. |
| Stale DataReplication is merged and Acked | Strictly older draws the newer record and no Ack | Phase 4 | F14/`wait_for_replication` honesty; equal versions still Ack. |
| `proto_version` judged after body decode | Judged after envelope unmarshal, before CRC/body | Phase 4 | Rolling-upgrade skew is classified `bad_proto_version`, not garbage. |
| Egress pins a record copy per answer | Egress carries `(handle, kind, key)` only | Phase 4 (CR-02) | ~1.9 GiB retention at cap becomes key-sized. |
| Restart listings keyed by source handle, rebuilt per trigger | Keyed by `(node_id, run_id)`, shared bounded build | Phase 4 (CR-01) | Hostile Hello floods cannot grow FIFOs or force per-trigger `Versions()` copies. |

**Deprecated/outdated:**
- v1.0's "chunks are self-sufficient, so the first chunk is the honest sync signal" reading (A1) — the Spec amendment replaced it with completeness.
- The Phase-3 comment "a first-seen peer gets no targeted listing" — replaced by D-04.
- The Phase-3 comment "a stale smaller write still broadcasts" — replaced by D-05.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The wire proto version should become 2 (proposed value; the review calls for a new `proto_version`) | Pattern 1 | If the planner instead keeps 1 and treats zero headers as legacy, the closed-interface guarantee weakens; mixed-version peers could open the gate on a first chunk. |
| A2 | "Empty disk" means zero records loaded by `persist.Open` (not "no dump file exists") | Pattern 3 / PROTO-02 | A node with an empty dump file would wait 60 s; a node with only discarded records would too. Both are defensible; confirm. |
| A3 | The stand harness may derive `--startup-gate-empty` from an explicit `--startup-gate` in the test extras | Pitfall 1 | If the planner instead edits the frozen test or accepts a 1 s empty gate, the closed-side assertions become racy and D-01's "only the TSV flips" breaks. |
| A4 | Equal-version DataReplication keeps the existing Ack (no reply) | Pattern 4 | If implemented literally as "≤ no Ack", retransmission drain regresses and R-02 goes red. |
| A5 | Proposed bounds (16 streams, 65 536 entries/round, 32 destinations, 65 536 queued chunks, 96-byte chunk overhead, 8-entry seen-run ring) | Patterns 1/4/5 | Too small drops legitimate large listings; too large weakens the DoS bound. The planner must size them against the documented 20k-key scale and record the choice. |
| A6 | A 3-node real-socket test can be broadcast-only on a shared SO_REUSEADDR port, constructed in-process with `udprepl.New(..., WithAddressReuse())` | PROTO-09 | It proves wiring/convergence but not unicast at N=3; the simulator covers semantics. If stronger evidence is required, a relay topology driving the real binary is needed. |
| A7 | `Delete` on a listed-but-not-arrived key stays `NotFound` (unspecified by 14.5) | Pattern 3 | If the owner wants it `Unavailable`, add one check and a test; no wire impact. |
| A8 | No new packages are needed | Standard Stack | A new package would need the legitimacy gate; none is currently imagined. |

## Open Questions (RESOLVED)

**Resolved at phase planning, 2026-11-22.** The decisions below are recorded in `04-CONTEXT.md`'s research-resolutions addendum and the named plan sections; the question bodies are retained as the evidence trail, not as open work.

| OQ | Resolution | Recorded in |
|----|------------|-------------|
| 1 | A strictly older `DataReplication` is answered with the newer record and no Ack; an equal version is Acked (S-PROTO-16 wins over the first "≤" draft). | CONTEXT addendum (D-03 refined); 04-05 task 1 |
| 2 | Delete on a listed-but-not-arrived key keeps the local `NotFound`; 14.5(6) names reads, so no `Unavailable` is required. | CONTEXT addendum; 04-02 task 2 step 4 |
| 3 | Three-node real-socket coverage uses three in-process adapters on one shared `SO_REUSEADDR` port; the unicast limitation is documented. | CONTEXT addendum; 04-12 (`docs/04-MULTI-NODE.md`) |
| 4 | Caps chosen: `maxListingDestinations = 32` and `maxListingChunks = 65 536` — one worst-case 20k-key listing round (20 000 chunks) fits. | 04-08 task 1 step 5 |
| 5 | The peer table stays uncapped this phase; the unbounded table is recorded as a residual trusted-LAN risk owned by Phase 7. | 04-12 truth 2; CONTEXT handoffs |
| 6 | Listing drift updates the clock only and does not feed `MergeStats`; OBSV-05 owns drift semantics in Phase 7. | CONTEXT addendum; 04-11 task 1 step 1 |
| 7 | `startup_gate_empty` validates positivity only (non-positive values exit through the usage path); Phase 7's one-point validation owns cross-parameter relations. | CONTEXT addendum; 04-04 task 1 step 3 |
| 8 | The literal "any neighbor's complete list opens the gate" reading stands; the mixed-cluster empty-list window is documented rather than changed. | CONTEXT addendum; 04-12 (`docs/04-MULTI-NODE.md`), pinned by 04-04's two-empty-nodes test |

1. **Equal-version DataReplication (D-03 wording).**
   - What we know: D-03 says "received version ≤ held version" → answer + no Ack; S-PROTO-16 says an Ack goes for an applied or already-equal version; retransmission recovery relies on duplicate Acks.
   - What's unclear: whether the owner wants "≤" literally.
   - Recommendation: strictly older → answer/no Ack; equal → Ack/no reply. Pin all three in a table test and record the reading in the phase report. If the owner insists on "≤", expect R-02's drain to regress.

2. **Delete on a listed-but-not-arrived key.**
   - What we know: rule 14.5(6) covers reads; `create_if_absent` is explicitly blocked.
   - What's unclear: whether a plain `Delete` should proceed (it would write a fresh tombstone dominating the unseen value) or answer `Unavailable`.
   - Recommendation: keep `NotFound` locally and document the gap for the API phase, unless the owner prefers `Unavailable`; either is a five-line change.

3. **Three-node real-socket topology.**
   - What we know: the stand harness spawns the production binary, which has no address-reuse flag, so it cannot bind three processes to one UDP port; the `udprepl` smoke test constructs adapters in-process with the test-only `WithAddressReuse` and proves same-host shared-port broadcast works on Linux; unicast to a shared port is kernel-arbitrary.
   - What's unclear: whether "real-socket tests with three nodes" requires unicast correctness or only convergence.
   - Recommendation: add an **in-process** three-adapter real-socket test in `internal/udprepl` (OPTION A: three stores/cores/adapters, one shared SO_REUSEADDR port, broadcast convergence) plus simulator N=3 tests, and document the unicast limitation; OPTION B (a test-side UDP relay with per-pair sockets driving the real binary) is available if full unicast at N=3 is demanded.

4. **Listing caps and stream caps.**
   - What we know: the documented scale is 20 000 keys; R-01 uses short keys (~32 entries/chunk), but the key ceiling is 1 024 bytes (one entry/chunk worst case).
   - What's unclear: the exact cap values the owner wants.
   - Recommendation: size the queued-work cap for at least one worst-case 20k-key round (20 000 chunks) — proposed 65 536 — and destinations 32; record the rationale in the phase artifact.

5. **Peer-table cap.**
   - What we know: `peers` is unbounded and an unknown-node Hello flood grows it (and the eviction scan) — pre-existing, not in CR-01.
   - What's unclear: whether Phase 4 should harden it.
   - Recommendation: optional; if added, cap at a handful-plus (e.g., 64) and count drops. Otherwise record as a residual trusted-LAN risk for Phase 7.

6. **Counting listing drift in `MergeStats`.**
   - What we know: `Observe` returns a drift signal; OBSV-05 owns drift semantics in Phase 7.
   - Recommendation: update the clock only; do not count listing drift in `MergeStats` yet. Record the handoff.

7. **`startup_gate_empty` relation validation.**
   - What we know: ADR 0006 wants empty nodes to wait longer; a user could set `startup_gate_empty < startup_gate`.
   - Recommendation: Phase 4 validates positivity only (exit 2 on ≤0); Phase 7's OBSV-02 one-point validation owns cross-parameter relations.

8. **Gate opens on a complete but *empty* listing in a mixed cluster.**
   - What we know: D-01 says a complete listing opens the gate; ADR 0006 explicitly wants two empty nodes to open on each other's empty lists. In a 3-node cluster where one peer is empty and another has data, the joiner can open on the empty peer's complete list before the data peer's listing arrives, then write with a clock that has not absorbed the data peer's versions.
   - What's unclear: whether the owner accepts the literal "any neighbor's complete list" reading for N≥3 (it re-creates a narrow F14-style window) or wants a stronger rule (e.g., wait for all known peers, or ignore an empty list when other peers are known).
   - Recommendation: keep the literal reading (it is what D-01/ADR 0006 state and what the frozen tests assert), and **document the residual window** in `04-MULTI-NODE.md`; the first-Hello listings from every peer shrink it to the hello cadence. Do not silently invent an all-peers rule — it would contradict the locked decision and the two-empty-nodes acceptance.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go toolchain | Everything | ✓ | go1.27.4 linux/amd64 | — |
| `go tool buf` | `make proto`, `buf-lint` | ✓ | 1.75.0 | Hand-edit generated `*.pb.go` (not recommended; bypasses lint/breaking) |
| `go tool protoc-gen-go` | codegen | ✓ | v1.36.12-devel | — |
| Linux loopback broadcast | stand/harness | ✓ (this host; the probe is Linux-only) | kernel 7.3.4-3-t2-noble | Simulator for semantics; real-socket tests are Linux-only by design |
| Python 3 | `scripts/red-runner` JSON parsing | ✓ | 3.15.2 | — |
| CPU governor `performance` | wall-clock red tests at documented scale | ✗ (current: `powersave`, 16 CPUs) | — | Run simulator/default-scale locally; re-boost for the phase gate / scheduled job (host note) |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** `performance` governor — wall-clock R-01/R-02/burst measurements must not be trusted under `powersave`; use `make red-run` at default scale or the scheduled jobs, and re-boost for the phase gate.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib), `go test -race`; `pgregory.net/rapid` for existing property tests; the in-repo simulator and stand harness |
| Config file | none — `go test` targets; `Makefile` targets `lint`, `test`, `red-run` |
| Quick run command | `go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./internal/grpcapi/ ./internal/app/persist/ ./internal/app/storage/` |
| Full suite command | `make test` (`go test -race -cover ./...`) + `make lint`; phase gate also `make red-run` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Proposed automated command / file | File Exists? |
|--------|----------|-----------|-----------------------------------|--------------|
| PROTO-01 | Completeness accumulation, gate signal, listed answers, HLC per part | unit/integration | `go test -race ./internal/app/repl/ -run 'TestSync_(completeListingGate|threePartListing|observePerPart|chunkHeaders)'`; `go test -race ./internal/udprepl/ -run 'TestSync_chunkWireBudget'`; `go test -race ./internal/grpcapi/ -run 'TestGate_|TestServer_listed'` | ❌ Wave 0 (some exist, need rewrite) |
| PROTO-01 | Frozen R-03 rows | stand/red | `SUPKV_RED_TESTS=1 go test -race ./internal/stand/ -run '^TestRed_R03_f14ReplacedNode$'`; `SUPKV_RED_TESTS=1 go test -race ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$'` | ✅ (frozen; flip TSV) |
| PROTO-02 | Empty-vs-data gate selection, two empty nodes, lone empty node | unit + stand | `go test -race ./cmd/sup-kv/ -run 'TestMain_startupGateEmpty'`; `go test -race ./internal/stand/ -run 'TestStand_startupGateEmpty'` | ❌ Wave 0 |
| PROTO-03 | Stale answer/no Ack; equal Ack; wait false | unit | `go test -race ./internal/app/repl/ -run 'TestReplication_(stale|equal)'`; adapter integration for the report | ❌ Wave 0 |
| PROTO-04 | First-seen exactly once; repeat nothing; returning peer immediate | unit + sim | `go test -race ./internal/app/repl/ -run 'TestRestart_(newPeer|repeat|returning)'`; `go test -race ./internal/app/repl/ -run 'TestScenario_emptyDisk'` | ❌/✅ (rewrite) |
| PROTO-05 | Regressed notification ignored + counted | unit | `go test -race ./internal/app/repl/ -run 'TestNotify_regressed'` | ❌ Wave 0 |
| PROTO-06 | Alternating run_ids config error, counted/WARN, no listing | unit | `go test -race ./internal/app/repl/ -run 'TestPeer_alternating'` | ❌ Wave 0 |
| PROTO-07 | Pre-body version verdict, class + per-peer WARN limit | unit | `go test -race ./internal/udprepl/ -run 'TestDecode_protoVersion'`; `go test -race ./internal/app/repl/ -run 'TestDiscard_badProtoVersionWarn'` | ❌ Wave 0 (negative table exists, needs update) |
| PROTO-08 | Dump-failed write broadcasts; snapshot load broadcasts | unit | `go test -race ./internal/grpcapi/ -run 'TestServer_(dumpFailedWrite|snapshotLoadNotifies)'`; `go test -race ./internal/app/persist/ -run 'TestEngine_restoreReturnsReplayed'` | ❌ Wave 0 |
| PROTO-09 | N≥3 docs + real-socket three nodes + wait semantics | real-socket adapter + sim + artifact | `go test -race ./internal/udprepl/ -run 'TestSmoke_threeNode'` (three in-process adapters on one shared port; the binary has no reuse flag); `go test -race ./internal/app/repl/ -run 'TestScenario_threeNode'`; `04-MULTI-NODE.md` | ❌ Wave 0 |
| D-09 (CR-01) | Hostile Hellos bounded; one `Versions()` per build | unit | `go test -race ./internal/app/repl/ -run 'TestListing_(hostile|sharedBuild)'` | ❌ Wave 0 |
| D-09 (CR-02) | Egress identity-only, no record pins, no Lookup before dedupe | unit | `go test -race ./internal/app/repl/ -run 'TestEgress_identityOnly'` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** the quick package set above (plus `make lint` for core changes).
- **Per wave merge:** `make test` + the simulator determinism tests (`TestChaos_determinismReplay`, `TestSim_determinism_withExtensions`).
- **Phase gate:** `make lint`, `make test`, `make red-run` (both R-03 rows green, mutants killed), one stand R-03 run, one simulator seed run; re-boost the governor for the wall-clock rows.

### Wave 0 Gaps
- [ ] `internal/app/repl/` — new tests for completeness, listed set, first-seen/restart, stale answers, regression, alternation, CR-01/CR-02.
- [ ] `internal/udprepl/` — codec/negative/fuzz fixtures for the new fields and pre-body verdict.
- [ ] `internal/grpcapi/` — read gating, listed-key semantics, totality notify; test harness `notify` seam.
- [ ] `internal/stand/` — empty-gate derivation and the startup-gate-empty tests (two-node; the three-node real-socket test is the in-process `internal/udprepl` one above).
- [ ] `cmd/sup-kv/` — flag/env/validation/selection tests; 14 existing `--startup-gate-empty` additions.
- [ ] Phase artifact `04-MULTI-NODE.md` (N-node behavior/timing + wait semantics at N>2).
- [ ] No new framework install: the existing Go test stack covers everything.

## Security Domain

> `security_enforcement` is enabled (level 1). This phase ships no authentication or cryptography; the trusted-LAN model is documented and AUTH-01 is deferred. The applicable controls are input validation, bounded state, honest classification and the closed-interface availability rule.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Trusted LAN; datagram authentication is deferred (AUTH-01, v2). |
| V3 Session Management | no | Stateless UDP protocol; no sessions. |
| V4 Access Control | no | The startup gate is an availability/consistency gate, not authorization. |
| V5 Input Validation | yes | Adapter envelope/CRC/body checks; core header/key/version validation; caps on streams/listings/peers; no hand-rolled parser beyond the pinned proto. |
| V6 Cryptography | no | CRC32 is corruption detection, never tamper protection (documented). Do not introduce crypto here. |

### Known Threat Patterns for UDP + proto + gRPC
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Spoofed Hello / node_id impersonation | Spoofing | `run_id` own-echo test, duplicate-node-id classifier, seen-run ring (D-06); no cryptographic auth (documented trusted LAN). |
| Partial Listing / SyncID flood exhausting memory | Denial of Service | Stream/entry caps + deterministic drop + counters; one shared listing build; one WARN per (node, class). |
| Restart-listing FIFO flood from attacker-chosen source handles | Denial of Service | Key listings by `(node_id, run_id)`, bound destinations/queued work, count drops (CR-01). |
| Full-value egress retention via stale listings / duplicate requests | Denial of Service | Identity-only egress + `Version` presence probe; no Data copy before dedupe (CR-02). |
| Log flooding through duplicate-node/bad-version datagrams | Denial of Service | Per-(node_id, class) WARN rate limit on event time. |
| Future-version garbage misclassified as undecodable | Tampering | Pre-body `proto_version` verdict, `bad_proto_version` class, WARN (D-07). |
| Stale record falsely Acked | Spoofing/Repudiation | Strictly older → newer-record reply, no Ack; equal → Ack (D-03/S-PROTO-16). |
| Partial-state overwrite (F14) | Tampering | Closed gate until complete listing; HLC update from listing parts; `create_if_absent` → `AlreadyExists` for listed keys. |
| Listed-but-unarrived reads served as absent | Information disclosure / correctness | `Unavailable` via the tombstone-visible presence probe (14.5(6)). |

## Sources

### Primary (HIGH confidence — read in this session)
- `internal/app/repl/core.go` — all core behavior, constants, discard classes, Hello/DataSync/Apply/Notify paths.
- `internal/port/repl.go` — `Message`, `DataSync`, `InboundDatagram`, `Outbox`, `Result` shapes.
- `internal/udprepl/repl.go` — decode/encode, mirror, SyncArrived, tracker, read loop.
- `internal/udprepl/replproto/supkv/v1/supkv.proto` — wire schema.
- `internal/grpcapi/gate.go`, `internal/grpcapi/server.go` — gate/RPC semantics.
- `internal/app/persist/engine.go`, `internal/app/storage/store.go`, `internal/app/clock/clock.go` — engine/clock seams.
- `cmd/sup-kv/main.go` — config/flags/gate wiring.
- `internal/stand/stand.go`, `internal/stand/stand_red_test.go`, `internal/stand/stand_metrics_test.go` — harness and frozen tests.
- `.planning/phases/04-protocol/04-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.planning/ROADMAP.md` — locked decisions, acceptance, phase goal.
- `docs/adr/0005-interface-closed-until-full-key-list.md`, `0006-startup-gate-empty-wait.md`, `0009-multi-node-support.md` — decision text.
- `docs/SPEC-DEVIATIONS.md` — S-HLC-04, S-HLC-10, S-ALG-RC-05, S-ALG-ST-06, S-PROTO-10/14/15/16/18, S-SCN-03/04, S-FAIL-07, S-PAR-14.
- `scripts/red-expectations.tsv` — the seven rows and the two R-03 red rows.
- `.planning/phases/03-transport/03-REVIEW.md` + `03-REVIEW-DISPOSITION.md` — CR-01/CR-02 exact fixes and reproductions.

### Secondary (MEDIUM confidence — official docs, fetched this session)
- `[CITED: https://protobuf.dev/programming-guides/proto3/#updating]` — adding fields is safe; changing field numbers is not; old binaries ignore new fields; proto3 preserves unknown fields.
- `[CITED: https://protobuf.dev/programming-guides/encoding/]` — field numbers are the wire key; parsers must accept fields in any order; unknown fields are skippable.
- `[CITED: https://pkg.go.dev/sync/atomic]` — sequentially consistent atomics; store-then-flag / load-flag-then-pointer publication.
- `[CITED: https://grpc.io/docs/guides/health-checking/]` — empty service name = whole-server health; `NOT_SERVING` means cannot accept requests; clients stop and resume.
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` and `evidence/review-repl-api.md` — R-03/R-04/R-05/R-10/R-12/R-13, RPL-13's "new `proto_version`", RPL-08.

### Tertiary (LOW confidence — recommendations / training-based)
- Proposed cap values, the seen-run ring size, the shared-build max age, and the 3-node relay option are engineering recommendations, not verified facts (see Assumptions Log).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; versions verified from `go.mod` and the local toolchain.
- Current code shapes: HIGH — every claim read from the source file this session; discrete values quoted.
- Architecture/design: MEDIUM — the design follows existing seams, but the cap values, equal-version reading, and 3-node topology need planner/owner choices.
- Pitfalls: HIGH — derived from the actual test suite and the frozen red tests; the harness-gate interaction is a concrete, reproducible trap.
- External claims: MEDIUM — official docs fetched, cited as CITED.

**Research date:** 2026-11-22
**Valid until:** 2026-12-22 (stable internal protocol; re-check if Phase 5 changes the engine/store seams first)
