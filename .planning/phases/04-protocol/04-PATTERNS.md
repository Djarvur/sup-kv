# Phase 4: Protocol - Pattern Map

**Mapped:** 2026-11-22
**Files analyzed:** 49 (17 production/doc/script, 32 test-package entries: 29 modified/new + 1 compile-verification-only + 2 frozen reference-only)
**Analogs found:** 47 / 49 (the exceptions are the generated `supkv.pb.go`, which is mechanical output of `make proto`, and the new `docs/04-MULTI-NODE.md`, which has a style analog but no same-role predecessor)

**Tracked-source gate:** every analog path named below was verified with `git ls-files -- <path>` this session (all returned a tracked path). The review probe tree under `docs/reviews/2026-11-19-v1.0/evidence/repro/` is tracked but is **not** part of the module (`go list ./docs/...` → "matched no packages"); it is never an analog and never needs the `StateStore.Observe` ripple.

**Ground truth:** line numbers below are from the working tree at HEAD `061256e` (post-Phase-3 + phase-4 research commits). Where RESEARCH.md quotes older numbers, these win.

---

## File Classification

### Production, docs, scripts

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/udprepl/replproto/supkv/v1/supkv.proto` | schema | transform (wire codec) | self: `message DataSync` (lines 64-68), `Envelope.proto_version` (line 17) | exact (self) |
| `internal/udprepl/replproto/supkv/v1/supkv.pb.go` | generated code | transform | `make proto` output (Makefile:98-100) | mechanical (no hand edit) |
| `internal/port/repl.go` | port/types | request-response seam | self: `DecodeStatus` block (12-26), `DataSync` (75-79), `InboundDatagram` (96-103) | exact (self) |
| `internal/app/repl/core.go` | core state machine | event-driven (datagram handlers, timer `Advance`) | self: `handleHello` (948-977), `applyRemote` (993-1019), `handleDataSync` (1116-1133), `NotifyLocalWrite` (1251-1281), listing machinery (1363-1557) | exact (self) |
| `internal/app/storage/store.go` | model/store | CRUD (LWW) | self: `Merge` (272-282, the only `clock.Update` path), `Version` (168-176) | exact (self) |
| `internal/app/persist/engine.go` | engine/adapter | CRUD + file I/O | self: `Version`/`Versions`/`Records` (517-535), `Restore` (576-595), `replayRecord` (621-634) | exact (self) |
| `internal/udprepl/repl.go` | adapter/wire | request-response datagram I/O | self: `decode` (1089-1125), `SyncArrived` (863-870), `mirrorStats` (998-1030), `Run` (540-573) | exact (self) |
| `internal/grpcapi/gate.go` | gate/middleware | request-response | self: `ServerOptions` (47-85), `gated` (104-112), `RunGate` (124-147) | exact (self) |
| `internal/grpcapi/server.go` | controller (RPC handlers) | CRUD | self: `Get` (111-120), `Set` (144-173), `SnapshotLoad` (264-273), `notifyLocalWrite` (286-297), `mapError` (325-344) | exact (self) |
| `internal/grpcapi/watch.go` | stream handler | streaming | self: `Watch` (364-398) + `subscribe` (225-246) | role-match |
| `cmd/sup-kv/main.go` | config/bootstrap | transform (flags/env → config → wiring) | self: env consts (94-129), defaults (133-159), `registerConfig` (310-357), `finalizeConfig` (418-434), `wire` (510-542), `newReplicationAdapter` (586-603) | exact (self) |
| `internal/stand/stand.go` | test harness | process spawn/config | self: `nodeArgs` (585-597), `startStandNode` (610-615), `launchNode` (381-405) | exact (self) |
| `internal/app/internal/simnet/simnet.go` | simulator harness | event-driven | self: `nodeStore` (435-449) | exact (self) |
| `internal/metrics/metrics.go` | observability adapter | transform (counters → families) | self: `trafficFamilies` discard samples (320-345) | exact (self) |
| `docs/04-MULTI-NODE.md` (new) | phase artifact doc | transform (spec prose) | `docs/OPERATIONS.md` heading/table style (1-8, 147); content: `docs/adr/0009-multi-node-support.md` + 04-RESEARCH Pattern 7 | style-match (no content predecessor) |
| `docs/OPERATIONS.md` | operations doc | in-place edit | self: discard list at line 147 | exact (self) |
| `scripts/red-expectations.tsv` | test registry | one-line flip | self: existing R-03 rows | exact (self) |

### Test files (the adaptation wave)

| New/Modified File | Kind | Why | Closest analog / helper to copy |
|---|---|---|---|
| `internal/app/repl/sync_test.go` | semantic + mechanical | header-aware fixtures; partial-vs-complete gate signal; chunk count | self: `syncFrom` (797-801), `dataSyncs` (273-284), `advanceDrain` (297-306), `dataSyncsEqual` (349-359) |
| `internal/app/repl/restart_test.go` | semantic | first-seen listing; alternation config error | self: `helloFromRun` (24-29), `targetedSyncs` (34-55), `TestRestart_newPeerNoFullSync` (158-196) |
| `internal/app/repl/discard_test.go` | mechanical + new | seed Hellos now enqueue listings; `bad_sync_header` rows; header in DataSync fixtures | self: `newDiscardCore`/`wantQueued` table (whole file) |
| `internal/app/repl/retransmit_test.go` | semantic + mechanical | stale DataReplication reply/no Ack; equal still Acks; queue counts shift | self: `stateStore` (68-90), `helloFrom` (121-125), `datagramFrom` (141-150) |
| `internal/app/repl/peertable_test.go` | minor | re-Hello after eviction adds first-seen listing; seen-run ring tests | self: `PeerSnapshot` assertions |
| `internal/app/repl/bench_test.go` | mechanical | DataSync benchmark must send a complete valid listing | self: `benchHandleDataSync` (139-157), `TestCore_keyCountCostFlat` (159+) |
| `internal/app/repl/egress_internal_test.go` | mechanical | `queuedUnicast.msg` disappears; identity-only drain | self: `egressStore` (43-64), `TestEgress_identitySeparates` (118-137) |
| `internal/app/repl/listing_internal_test.go` (new) | new | CR-01 hostile/shared-build regression | `egress_internal_test.go` whole-file shape (white-box `package repl`) |
| `internal/app/repl/sim_scenario_test.go` | semantic | `peerEvicted` lower bound; empty-disk comment/bound; new closed-listing/3-node scenarios | self: `TestScenario_emptyDisk` (1138-1200), `TestScenario_longPartition` (529-640) |
| `internal/app/repl/sim_bound_test.go` | mechanical check | extra listing traffic vs. budget assertions | self: existing bounds |
| `internal/app/repl/decision_coverage_test.go` | optional one-line | 14.9 anchor may repoint to the new three-node test | self: `decisionIndex` (33-49) |
| `internal/app/persist/engine_test.go` | semantic | `Observe`, `KeyCount`, `Restore` return shape | self: restore/no-op tests |
| `internal/app/storage/store_test.go` (and `view_test.go` if placed there) | new method tests | `Observe` version-only clock update; `Len` | self: `Version`/`Versions` tests |
| `internal/udprepl/syncsize_internal_test.go` | mechanical | overhead 64→96; graduated lens 1336→1112; `wireStateStore.Observe` | self: constants (29-46), assertion (137-173), fake (48-66) |
| `internal/udprepl/decode_negative_test.go` | semantic | pre-body version row; combined future-version + corrupt-body row | self: `negativeRow` table (89-140) |
| `internal/udprepl/repl_test.go` | semantic | mirror flips only on complete listing; `ListedKey`; tracker fake `Observe` | self: `TestMirror_syncArrivedFlip` (618-680), `stateStore` (47-64) |
| `internal/udprepl/repl_internal_test.go` | mechanical | `trackerStore.Observe`; version-mismatch crafting helper | self: `EncodeDatagramForTestVersion` (50-70), `trackerStore` (85-100) |
| `internal/udprepl/scenario_test.go` | verify only (no edit expected) | constructs its nodes through `smokeStore`; compiles once `smokeStore` gains `Observe` | self: line 162 comment + `storage.NewStore` use (171) |
| `internal/udprepl/smoke_test.go` | mechanical | `smokeStore.Observe`; two-node baseline | self: `smokeStore` (184-198), `TestSmoke_twoNodesUDP` (301-330) |
| `internal/udprepl/fuzz_decode_test.go` | mechanical | version-mismatch seed now yields `ProtoVersionMismatch` | self: seed builder (150-175) |
| `internal/udprepl/three_node_test.go` (new) | new | real-socket N=3 broadcast convergence | `smoke_test.go` recipe: `smokeSocket` (68-105), `probeBroadcastAddr` (~250-280), shared-port adapters (305-324) |
| `internal/grpcapi/gate_test.go` | semantic | reads also Unavailable; listed-key answers; complete-listing fixture | self: `closedGateOptions` (40-45), `TestGate_writesUnavailableReadsOK` (56-107) |
| `internal/grpcapi/server_test.go` | mechanical + new | `ListedKey` option; notify capture for totality | self: `testServerConfig` (112-133), `buildTestServer` (181-211) |
| `internal/grpcapi/wait_integration_test.go` | mechanical | `peerStore.Observe` | self: `peerStore` (167+) |
| `internal/app/internal/simnet/simnet_test.go` | mechanical | `nodeStore.Observe` compile | self: `nodeStore` (435-449) |
| `internal/app/internal/simnet/simnet_ext_test.go` | mechanical | same; determinism expectations | self: `runSimExtScenario` (467+) |
| `internal/stand/stand_test.go` | harness-driven | empty-gate derivation keeps it green; empty-disk comment/bound | self: `TestStand_emptyDiskRecovery` (342+) |
| `internal/stand/stand_metrics_test.go` | harness-driven | `startup_gate_delays_gate_open` now depends on derivation | self: subtest (243-266) |
| `cmd/sup-kv/main_test.go` | semantic + mechanical | 14 `--startup-gate-empty` additions; selection test | self: `startNode` sites (396, 410, ...) |
| `internal/metrics/metrics_test.go` | mechanical (optional) | new `bad_sync_header` label if published | self: existing label assertions |
| `internal/stand/stand_red_test.go` | **FROZEN — NO EDITS** | R-03/F14 acceptance; flips only via the TSV | reference only (see Shared Patterns) |
| `internal/grpcapi/red_gate_test.go` | **FROZEN — NO EDITS** | resolver parked until "full"; passes unmodified once `SyncArrived` means complete | reference only |

---

## Pattern Assignments

### `internal/udprepl/replproto/supkv/v1/supkv.proto` (schema, transform)

**Analog (self):** the existing `DataSync` message and its comment (lines 64-68), plus the envelope's version field (line 17).

```proto
// DataSync is one self-sufficient chunk of the sorted key→version listing
// exchanged for full reconciliation (REPL-05).
message DataSync {
  repeated SyncEntry entries = 1;
}
```

**Pattern to extend:** keep `entries = 1` byte-identical and append new field numbers only (proto3 field-number safety; 04-RESEARCH Standard Stack). Proposed shape (research, mark chosen values in the phase report):

```proto
message DataSync {
  repeated SyncEntry entries = 1;
  uint64 sync_id = 2;      // fresh per listing round
  uint32 chunk_index = 3;  // 1..chunk_count
  uint32 chunk_count = 4;  // chunks in the round
}
```

Also update the file-header comment, which still says "DataRequest and DataSync land with the later plans" (lines 1-6). Regenerate with `make proto` (Makefile:98-100 runs `go tool buf generate` per tree); never hand-edit `supkv.pb.go`.

---

### `internal/port/repl.go` (port seam, request-response)

**Analog (self):** the `DecodeStatus` const block and `DataSync` struct.

**Decode verdict pattern** (lines 12-26) — add one value, keep the `iota` order and the doc-per-value discipline:

```go
type DecodeStatus int8

const (
	Decoded DecodeStatus = iota
	BadCRC
	Undecodable
)
```

**Message struct pattern** (lines 75-79) — the header fields go before `Entries`, each with a one-line contract comment:

```go
// DataSync is one self-sufficient chunk of the sorted key→version listing
// exchanged for full reconciliation (REPL-05).
type DataSync struct {
	Entries []SyncEntry
}
```

Add `ProtoVersionMismatch DecodeStatus` (value 3) with a doc comment stating it is decided **after envelope unmarshal, before CRC and body decode**, and that `Msg` is nil and the envelope fields (`ProtoVersion`, `NodeID`, `RunID`) are populated so the core can attribute and rate-limit the discard. `InboundDatagram` (96-103) already carries every field the verdict needs — no struct change there.

---

### `internal/app/repl/core.go` (core state machine, event-driven)

This file gets the majority of the phase. Copy the existing shapes; do not introduce a parallel mechanism.

**1. Protocol version and discard classes** — lines 34-61.

```go
const SupportedProtoVersion = 1
...
const (
	DiscardBadCRC          = "bad_crc"
	DiscardUndecodable     = "undecodable"
	DiscardBadProtoVersion = "bad_proto_version"
	DiscardOwnBroadcast    = "own_broadcast"
	DiscardDuplicateNodeID = "duplicate_node_id"
	...
	DiscardBadVersion        = "bad_version"
)
```

Raise `SupportedProtoVersion` to 2; add `DiscardBadSyncHeader = "bad_sync_header"` and its `DiscardStats.BadSyncHeader` counter (the class block at 286-304 and the switch in `discardValidated` at 853-865 are the two places a class must be registered — the `exhaustive` linter sees the switch). Add `NotificationsRegressed` as a counter near `MergeStats` (313-320) or a new small stats struct, with a copying accessor following `MergeStats()` (1802-1808).

**2. `StateStore` interface** — lines 249-268. Add one method in the same consumer-side, per-method-doc style:

```go
type StateStore interface {
	ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error)
	Version(key string) (version.RecordVersion, bool)
	Lookup(key string) (storage.Record, bool)
	Versions() map[string]version.RecordVersion
	// Observe absorbs one received version into the store's clock without
	// touching any record: the per-DataSync-part HLC update (PROTO-01).
	Observe(ver version.RecordVersion) (drift time.Duration)
}
```

**3. Queue/bound types** — lines 355-383. CR-02 turns `queuedUnicast` into identity only; CR-01 turns `listingFIFO` from per-handle chunk copies into a reference to one shared build:

```go
type egressKey struct {
	handle string
	kind   egressKind
	key    string
}

type queuedUnicast struct {
	id  egressKey
	msg port.Message
}

type listingFIFO struct {
	dst     string
	entries [][]port.SyncEntry
}
```

Copy the cap+drop+count idiom exactly: `maxEgressEntries` (97-101), the drop path `dropOldestEgress` (1182-1188), `egressDropped` (564-565), accessor `EgressDropped` (1908-1914). New bounds (proposed in research: 16 streams, 65 536 entries/stream, 131 072 total, 32 destinations, 65 536 queued chunks) must be declared in the same `const` block with the "why this number" comment style at 97-101.

**4. Core state** — lines 499-621. Extend, do not replace:

```go
// listingByDst holds the in-flight listing FIFOs keyed by destination ...
listingByDst  map[string]*listingFIFO
listingOrder  []string
listingCount  int
nextListingAt time.Time
syncRotation  uint64
...
// syncArrived flips true on the first received DataSync datagram and
// never resets — the startup gate's sync half (API-04).
syncArrived bool
```

New fields, same comment discipline: `syncs map[syncStreamKey]*syncStream` (accumulation by `(node_id, run_id, sync_id)`), `listedByPeer map[uint16]map[string]struct{}`, `listingGen uint64`, `listingComplete bool`, `nextSyncID uint64`, and `peers` gains a bounded `seenRuns` ring (proposal: 8). Re-key `listingByDst` by an explicit destination identity `(nodeID, runID)` (plus the reserved `broadcastDst` at line 105), never by the datagram source handle.

**5. `HandleDatagram` classification order** — lines 695-739. Insert the new adapter verdict in the same early-return chain; the existing field check stays as defence-in-depth:

```go
switch in.Status {
case port.BadCRC:
	c.discards.BadCRC++
	c.cfg.OnDiscard(DiscardBadCRC, in.Source)
	return
case port.Undecodable:
	c.discards.Undecodable++
	c.cfg.OnDiscard(DiscardUndecodable, in.Source)
	return
}
if in.ProtoVersion != SupportedProtoVersion {
	c.discards.BadProtoVersion++
	c.cfg.OnDiscard(DiscardBadProtoVersion, in.Source)
	return
}
```

Add `case port.ProtoVersionMismatch:` doing exactly the `BadProtoVersion` count + `OnDiscard` (no body exists; `Msg` is nil). Keep the WARN rate limit in the shared helper below.

**6. Header validation** — `validate` (763-775) and `validateSync` (802-811). The existing pattern is "the first failing item names one class and discards the whole datagram":

```go
case port.DataSync:
	return c.validateSync(now, in.Source, msg.Entries)
```

Extend `validateSync` to take the message (not just entries), and check the header first: `chunk_count >= 1`, `1 <= chunk_index <= chunk_count`, `chunk_count <= maxSyncChunks`, consistent `chunk_count` for an existing stream. Return `DiscardBadSyncHeader` on violation. Reuse `discardValidated` (853-876) unchanged for counting + `OnDiscard` + the per-class event-time WARN.

**7. `handleHello`** — lines 948-977, the exact current shape:

```go
func (c *Core) handleHello(now time.Time, in port.InboundDatagram) {
	c.helloSeen = true
	discovery := len(c.peers) == 0
	known, remembered := c.peers[in.NodeID]
	restarted := remembered && known.runID != in.RunID
	known.handle = in.Source
	known.runID = in.RunID
	known.lastHello = now
	c.peers[in.NodeID] = known
	if restarted {
		c.enqueueRestartListing(now, in.NodeID, in.Source)
		return
	}
	if !discovery {
		return
	}
	for _, pk := range sortedPendingKeys(c.pending) {
		...
		c.enqueueRecordSend(now, pk, false)
	}
}
```

D-04/D-06 rewrite: on an **unknown** node, enqueue exactly one first-seen listing **and then fall through** to the existing `if !discovery { return }` + flush block (Pitfall 5 — do not copy the restart branch's `return`). On a known node whose run_id is in `seenRuns` but not current → `DiscardDuplicateNodeID` count + rate-limited WARN + drop, **no listing**, no row update. On a known node with an unseen run_id → push the old run into `seenRuns`, update, one listing. The first-seen and restart paths share one listing enqueue function.

**8. `applyRemote` (stale record)** — lines 993-1019, current apply+Ack:

```go
conflicts, drift, err := c.store.ApplyRemote(msg.Key, msg.Record)
...
out.Unicast(source, port.Ack{Key: msg.Key, Version: msg.Record.Version})
```

Insert the three-way compare **before** `ApplyRemote` using `store.Version` (tombstone-visible, no Data copy): strictly older → enqueue an identity-only egress `DataReplication` and return with **no Ack**; equal → `out.Unicast(source, port.Ack{...})` and return; strictly newer/absent → the existing apply+Ack path. The equal-version Ack is load-bearing (S-PROTO-16 / Pitfall 9).

**9. `handleDataRequest` (CR-02)** — lines 1088-1094. Today it does a `Lookup` before dedupe:

```go
rec, ok := c.store.Lookup(key)
if !ok {
	return
}
c.enqueueResponse(now, source, port.DataReplication{Key: key, Record: rec})
```

Replace with a `Version` presence probe and an identity-only enqueue (`enqueueResponse(now, source, egressDataReplication, key)`); lookup moves to drain. Keep the "absent key produces nothing" contract.

**10. `handleDataSync` (PROTO-01 / 14.5(6) listing semantics)** — lines 1116-1133, current body:

```go
func (c *Core) handleDataSync(now time.Time, source string, entries []port.SyncEntry) {
	c.syncArrived = true
	for _, entry := range entries {
		local, present := c.store.Version(entry.Key)
		switch {
		case !present || local.Compare(entry.Version) < 0:
			c.enqueueResponse(now, source, port.DataRequest{Key: entry.Key})
		case local.Compare(entry.Version) > 0:
			rec, ok := c.store.Lookup(entry.Key)
			if !ok {
				continue
			}
			c.enqueueResponse(now, source, port.DataReplication{Key: entry.Key, Record: rec})
		}
	}
}
```

Rewrite the signature to `handleDataSync(now, in, msg, out)` (it needs `in.NodeID`, `in.RunID`, and the header). Sequence, following the existing per-entry compare verbatim:
1. If the part is non-empty, compute its max version and call `store.Observe(max)` exactly once.
2. Accumulate by `(in.NodeID, in.RunID, msg.SyncID)` with the caps.
3. If every index `1..ChunkCount` is present: publish the listing key set for that peer, bump `listingGen`, set `listingComplete = true`, delete the stream, then `c.syncArrived = true` (only now).
4. Per-entry compare exactly as today, but enqueue **identity only** for both response kinds (no second `Lookup`).

The current function's comments ("the first received chunk is the honest first-sync-round signal", lines 1102-1106) are the wrong contract and must be rewritten, not merely amended.

**11. Egress enqueue/drain (CR-02)** — lines 1146-1235. Preserve the dedupe/overflow/pacer shape exactly; change only the payload:

```go
func (c *Core) enqueueResponse(now time.Time, handle string, msg port.Message) {
	id, ok := egressIdentity(handle, msg)
	if !ok {
		return
	}
	if _, dup := c.egressIDs[id]; dup {
		return
	}
	if len(c.egress) == 0 {
		c.nextSendAt = now
	}
	c.egress = append(c.egress, queuedUnicast{id: id, msg: msg})
	c.egressIDs[id] = struct{}{}
	if len(c.egress) > maxEgressEntries {
		c.dropOldestEgress()
	}
}
```

Keep `egressIdentity` (1168-1176), `dropOldestEgress` (1182-1188), `drainEgress` (1202-1219), `resolveQueuedResponse` (1226-1235) — the drain-time `Lookup` and "gone key skips the slot, consumes no slot" behavior are exactly what CR-02 wants; the change is that a `DataReplication` queue entry carries only the key. Producing a `DataRequest` at drain is trivial (key only).

**12. `NotifyLocalWrite` regression guard (D-05)** — lines 1251-1281. Insert at the top, before supersession:

```go
if held, present := c.store.Version(key); present && held.Compare(rec.Version) > 0 {
	c.notificationsRegressed++
	return
}
```

Keep the existing supersession/broadcast body (1252-1280) unchanged for the accepted case. Note the current comment "a stale smaller write still broadcasts without displacing" (1245-1248) must be replaced.

**13. Listing build and emission (D-01 + CR-01)** — lines 1363-1557. The current shapes to copy:

```go
func (c *Core) syncListing(now time.Time) {
	...
	chunks := c.listingChunks()
	start := int(c.syncRotation % uint64(len(chunks)))
	for i := range chunks {
		c.enqueueListingChunk(now, broadcastDst, chunks[(start+i)%len(chunks)])
	}
	c.syncRotation++
}

func (c *Core) listingChunks() [][]port.SyncEntry {
	versions := c.store.Versions()
	entries := make([]port.SyncEntry, 0, len(versions))
	for _, key := range slices.Sorted(maps.Keys(versions)) {
		entries = append(entries, port.SyncEntry{Key: key, Version: versions[key]})
	}
	return chunkSyncEntries(entries, c.cfg.SyncChunkBytes)
}
```

Changes: `listingChunks` becomes the shared `listingBuild{chunks, at, refs, syncID}` (built at most once per `cfg.SyncInterval` age), each FIFO stores `{build, syncID, startIndex, emitted}` and stamps `SyncID/ChunkIndex/ChunkCount` at drain (lines 1434-1438); every destination key becomes `(nodeID, runID)`; a repeat Hello for the same pair is a no-op. Keep the rotation rule ("start chunk = rotation % len") — indices must stay the original chunk positions (rotating without keeping indices makes completeness unrecognizable). Keep `chunkSyncEntries` (1534-1557) as the single chunker and raise `chunkOverheadBytes` to 96 with the re-derived comment.

**14. Accessors** — 1798-1933. New state is read cross-goroutine only through copying accessors mirrored into atomics (the `Discards`/`MergeStats`/`SyncArrived` precedent): `ListingGeneration() uint64`, `ListingComplete() bool`, `ListedKeysSnapshot() map[string]struct{}` (clone), and counters for stream drops / notification regressions.

---

### `internal/app/storage/store.go` (store, CRUD/LWW)

**Analog (self):** `Merge` is the only `clock.Update` path (272-282); `Version` is the zero-copy probe (168-176); `Records` shows why an O(store) clone must not be the count source (327-340).

```go
// Merge joins an externally versioned record for key ...
func (s *Store) Merge(key string, rec Record) (conflicts []Conflict, drift time.Duration, err error) {
	...
	remoteL, remoteC, _ := rec.Version.Unpack()
	_, _, drift = s.clock.Update(remoteL, remoteC)
	return conflicts, drift, nil
}
```

**Add** `Observe(ver version.RecordVersion) time.Duration` with exactly this body shape (unpack → `clock.Update` → return drift), placed beside `Merge`; the `Clock` interface (84-90) needs no change. **Add** `Len() int` (RLock + `len(s.records)`) beside `Versions`, for the startup empty-vs-data selection — never `Records()` (clones every value).

---

### `internal/app/persist/engine.go` (engine, CRUD + file I/O)

**Analog (self):** the pass-throughs `Version`/`Versions`/`Records` (517-535) and `Restore`/`replayRecord` (556-634).

**Totality seam (D-08):** `Set` already returns the stamped version even when `await` fails, because `ver` comes from the store write before `e.await()` (395-401). That is the seam the gRPC handler uses to notify a dump-failed write; do not change the error contract.

```go
func (e *Engine) Set(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	ver, drift, err := e.store.Set(key, data)
	if err != nil {
		return ver, drift, err
	}
	return ver, drift, e.await()
}
```

**Add:** `KeyCount() int` delegating to `store.Len()` (copy the one-line `Version` shape); `Observe(ver)` delegating to `store.Observe`. **Change** `Restore` (576-595) to return the freshly replayed records in deterministic sorted-key order, even when the final flush fails:

```go
func (e *Engine) Restore(name string) error {
	data, err := e.dumps.ReadSnapshot(name)
	...
	if restoreChanges(e.store.Records(), records) {
		for _, key := range slices.Sorted(maps.Keys(records)) {
			err = replayRecord(e.store, key, records[key])
			...
		}
		e.markDirty()
	}
	return e.flush()
}
```

`replayRecord` (621-634) currently discards the fresh versions (`_, _, err := store.Delete/Set`); have it return the stamped `(version, deleted)` so `Restore` can build the sorted `[]RestoredRecord`. Callers use `_, err :=` (Pitfall 13 lists ~12 sites, all in tests plus `SnapshotLoad`).

---

### `internal/udprepl/repl.go` (adapter, request-response)

**Analog (self):** `decode` (1089-1125), the mirror accessor (863-870), the per-event mirror (998-1030), and the loop (540-573).

**Pre-body version verdict (D-07).** Current order — envelope domain → CRC → body → message:

```go
func decode(datagram []byte) port.InboundDatagram {
	var env replproto.Envelope
	if len(datagram) == 0 || proto.Unmarshal(datagram, &env) != nil {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	if env.GetNodeId() > maxNodeIDWire || len(env.GetRunId()) != wireIDSize {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	in := port.InboundDatagram{
		ProtoVersion: env.GetProtoVersion(),
		NodeID:       uint16(env.GetNodeId()),
	}
	copy(in.RunID[:], env.GetRunId())
	if crc32.ChecksumIEEE(env.GetBody()) != env.GetCrc32() {
		in.Status = port.BadCRC
		return in
	}
	var body replproto.Body
	err := proto.Unmarshal(env.GetBody(), &body)
	...
}
```

Insert, after `copy(in.RunID[:], ...)` and **before** the CRC check:

```go
if in.ProtoVersion != repl.SupportedProtoVersion {
	in.Status = port.ProtoVersionMismatch
	return in
}
```

This is the one place the order changes; `BadCRC` remains the verdict for same-version corrupt bodies. Keep the mapping helpers `protoBody` (1131-1156) and `portMessage` (1161-1205) as the single field-mapping sites: add the header fields in `portMessage`'s `Body_DataSync` branch (1186-1199) and in `protoBody`'s `port.DataSync` branch (1147-1152).

**One atomic listing mirror (Pattern 2).** `Adapter` currently has `syncArrived atomic.Bool` (310-312) and `mirrorStats` stores `a.syncArrived.Store(a.core.SyncArrived())` (1029). Replace with an `atomic.Pointer[listingMirror]`:

```go
type listingMirror struct {
	complete bool
	keys     map[string]struct{}
}
```

Publish only from the event loop, only when the generation changes, after `core.Advance`/handler + `flush` + `applyResults` (the loop at 540-573 is the template):

```go
if gen := a.core.ListingGeneration(); gen != a.listingGen {
	a.listingGen = gen
	a.listing.Store(&listingMirror{
		complete: a.core.ListingComplete(),
		keys:     a.core.ListedKeysSnapshot(),
	})
}
```

Accessors copy the `SyncArrived` shape (863-870): `SyncArrived()` loads the pointer and returns `m != nil && m.complete`; new `ListedKey(key string) bool` loads the **same** pointer, returns false when not complete, then checks membership. Never publish `complete` separately from the keys (Pitfall 2). Add a `listingGen uint64` field next to `syncArrived` in the struct.

---

### `internal/grpcapi/gate.go` (gate, request-response)

**Analog (self):** `gated` (104-112), `RunGate` (124-147), `openGate` (154-158).

```go
func (s *Server) gated() bool {
	return !s.gateOpen.Load()
}
```

Change the **comment** (the code is already the right predicate): reads now consult the gate too. Add to `ServerOptions` (47-85), following the exported-field-with-contract-doc style:

```go
ListedKey func(key string) bool
```

Document it as "the adapter's listing mirror membership; nil means no listing knowledge (single-node / tests)". Keep `gateMessage` (line 30) unless the phase report chooses new wording — only `gate_test.go` asserts the exact string, and the frozen red tests assert codes only.

Do **not** change `RunGate`'s poll/backstop mechanics; the completeness meaning lives entirely in what `SyncArrived()` reports.

---

### `internal/grpcapi/server.go` (RPC handlers, CRUD)

**Analog (self):** `Get` (111-120), `Set` (144-173), `SnapshotLoad` (264-273), `notifyLocalWrite` (286-297), `mapError` (325-344).

**Read gating + listed-but-unarrived (14.5(6), Pitfall 10):**

```go
func (s *Server) Get(_ context.Context, req *apiproto.GetRequest) (*apiproto.GetResponse, error) {
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	rec, err := s.engine.Get(req.GetKey())
	if err != nil {
		return nil, mapError(err)
	}
	return &apiproto.GetResponse{Record: recordToProto(req.GetKey(), rec)}, nil
}
```

Add `if s.gated() { return nil, gateUnavailable() }` as the first statement (matching Set/Delete/SnapshotCreate). In the error branch, distinguish tombstone from absence with `s.engine.Version(key)` (tombstone-visible) and only then consult `s.opts.ListedKey`: tombstone → `NotFound`; listed-but-absent → `Unavailable` with the pinned message; otherwise `NotFound` via `mapError`. Note `Get` currently has the empty-key check **before** any gate check; the gate must come first (the same "gate check is the handlers' first statement" contract Set documents at 144-147).

**`create_if_absent` listed key → `AlreadyExists` before the engine call** (Set, after gate + key validation at 148-153):

```go
if req.GetCreateIfAbsent() && s.opts.ListedKey != nil && s.opts.ListedKey(req.GetKey()) {
	return nil, status.Error(codes.AlreadyExists, storage.ErrAlreadyExists.Error())
}
```

**Totality (D-08):**
- `Set`/`SetIfAbsent`/`Delete` error paths: when `errors.Is(err, persist.ErrDumpFailed)`, notify the record/tombstone **before** returning. Reuse `notifyLocalWrite` (286-297) and the existing version+value inputs; do not notify on `ErrClosed`.
- `SnapshotLoad` (264-273): consume `[]RestoredRecord` from `Restore` and notify each with `track=false`; notify even when the final flush failed (state is in memory), but not on `ErrClosed`. Keep the name validation delegated to `persistfile` (don't re-validate).

---

### `internal/grpcapi/watch.go` (stream handler, streaming)

**Analog (self):** `Watch` (364-398). Add the gate as the first statement (the gate is the interface contract, so it precedes the nil-hub check):

```go
func (s *Server) Watch(req *apiproto.WatchRequest, stream apiproto.SupKVService_WatchServer) error {
	if s.gated() {
		return gateUnavailable()
	}
	if s.hub == nil {
		return status.Error(codes.Unimplemented, "watch requires a hub; construct the server through NewServer")
	}
	...
}
```

`List` (server.go 222-233) also needs the gate first statement; it needs no listed-key logic.

---

### `cmd/sup-kv/main.go` (config/bootstrap, transform)

**Analog (self):** the `startup-gate` twins and the validation/wiring surfaces.

**Flag/env/default trio** — env const block (94-129), defaults block (133-159), config struct (167-195), registration (342-343):

```go
envStartupGate = "SUPKV_STARTUP_GATE"
...
defaultStartupGate = time.Second
...
flag.DurationVar(&cfg.startupGate, "startup-gate", envDur(envStartupGate, defaultStartupGate),
	"gate backstop: writable this long after start even without a sync round (env SUPKV_STARTUP_GATE)")
```

Add `envStartupGateEmpty = "SUPKV_STARTUP_GATE_EMPTY"` beside `envStartupGate`; `defaultStartupGateEmpty = 60 * time.Second` in the defaults block; `startupGateEmpty time.Duration` in `config`; and one `flag.DurationVar` beside line 342 with the env name in the usage string.

**Validation (basic relation only, D-02 + addendum)** — model on `checkMetricsCerts` (441-449): a small `checkStartupGates(cfg config) error` returning a named sentinel (`errStartupGateRange`, declared with the other sentinels at 59-64) for `startup_gate <= 0` / `startup_gate_empty <= 0`; call it from `finalizeConfig` (418-434) **before** `checkDatagramCeiling`. Do not add cross-parameter relations (Phase 7's one-point validation owns them). Errors flow through main's existing `exitUsage` path.

**Selection + wiring** — the current wiring (532-538):

```go
srv := grpcapi.NewServer(engine, adapter.NotifyLocalWrite, hub, grpcapi.ServerOptions{
	SyncArrived:    adapter.SyncArrived,
	KnownPeers:     adapter.KnownPeers,
	WaitReplicated: adapter.WaitReplicated,
	StartupGate:    cfg.startupGate,
	WaitTimeout:    cfg.waitTimeout,
})
```

Compute `gateBackstop := cfg.startupGate; if engine.KeyCount() == 0 { gateBackstop = cfg.startupGateEmpty }` immediately before this call (the engine is loaded synchronously at 512, so the count is final), pass `StartupGate: gateBackstop`, and add `ListedKey: adapter.ListedKey`. The adapter is constructed by `newReplicationAdapter` (586-603); no change needed there unless new `repl.Config` knobs are added.

**Test ripple:** 14 sites pass `--startup-gate=10ms` (main_test.go:396, 410, 539, 600, 622, 661, 690, 743, 809, 1002, 1010, 1183, 1250, 1412). A fresh test node has an empty dump → with the 60 s default it never reaches SERVING inside `servingDeadline`. Add `--startup-gate-empty=10ms` beside every one (Pitfall 11); do **not** make an explicit `--startup-gate` imply the empty gate (contradicts ADR 0006).

---

### `internal/stand/stand.go` (harness, process config)

**Analog (self):** `nodeArgs` (585-597) and `startStandNode`/`startStandLoggedNode` (610-623).

```go
func startStandNode(t *testing.T, p *standPair, nodeID int, extra ...string) *standNode {
	t.Helper()
	return launchNode(t, p.bin, &bytes.Buffer{}, nil, append(p.nodeArgs(nodeID), extra...)...)
}
```

Add one derivation helper used by both starters (and only there):

```go
// emptyGateArg derives --startup-gate-empty from an explicit --startup-gate
// in the scenario extras (later flags win), else the harness default.
func emptyGateArg(extra []string) string { ... } // scan prefix "--startup-gate="
```

then append `emptyGateArg(extra)` after the extras. This keeps the frozen `stand_red_test.go` at 30 s for both timers (it passes `"--startup-gate="+redR03StartupGate.String()` via extras at 770-773), keeps `TestConfig_parametersByEffect/startup_gate_delays_gate_open` at 2 s for both (stand_metrics_test.go:246-250 asserts elapsed ≥ 2 s), and keeps every other stand test at `standStartupGate` (1 s). **Do not edit `stand_red_test.go`.**

---

### `internal/app/internal/simnet/simnet.go` (simulator harness, event-driven)

**Analog (self):** `nodeStore` (435-449), the four-method adapter:

```go
type nodeStore struct{ store *storage.Store }

func (s nodeStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}
func (s nodeStore) Version(key string) (version.RecordVersion, bool) { return s.store.Version(key) }
func (s nodeStore) Lookup(key string) (storage.Record, bool)        { return s.store.Lookup(key) }
func (s nodeStore) Versions() map[string]version.RecordVersion      { return s.store.Versions() }
```

Add `Observe(ver version.RecordVersion) time.Duration { return s.store.Observe(ver) }`. This one method makes every simulator node absorb listing versions into its clock. No other simnet behavior changes; the extra listings show up in existing budgets/digests (re-verify `TestScenario_*`, `TestChaos_determinismReplay`, `TestSim_determinism_withExtensions`).

---

### `internal/metrics/metrics.go` (observability adapter, transform)

**Analog (self):** the discard samples in `trafficFamilies` (320-345) — the whole vocabulary is always published, zero counts included:

```go
{Labels: []Label{{Name: labelClass, Value: "bad_version"}}, Value: float64(disc.BadVersion)},
```

If the new header class is published, append one sample with value `float64(disc.BadSyncHeader)` and add the counter to `DiscardStats`. Optional in this phase; the phase report records the choice (research Test Impact Map calls it optional).

---

### `docs/04-MULTI-NODE.md` (new phase artifact, doc)

**Style analog:** `docs/OPERATIONS.md` first lines (1-8) — `#` title, one bold-link intro paragraph naming the sibling docs, sentence-case `##` sections. **Content inputs:** `docs/adr/0009-multi-node-support.md`, ADR 0006 (empty gate), 04-RESEARCH "Pattern 7: Totality" bullet list (traffic shape, timing, `wait_for_replication` at N > 2, residual mixed-cluster window from Open Question 8), and the CR-01/CR-02 bounds rationale. Record the chosen cap values and the 3-node test topology/limitation there.

---

### `docs/OPERATIONS.md` + `scripts/red-expectations.tsv` (handoff edits)

- `docs/OPERATIONS.md:147`: add the `bad_sync_header` class and the `startup-gate-empty` knob to the existing lists (Phase 9 owns the final catalog; this is the handoff edit).
- `scripts/red-expectations.tsv`: flip exactly two rows — `TestRed_R03_gateClosedUntilFullList` and `TestRed_R03_f14ReplacedNode` from `red` to `green`. No runner change, no test edit (D-01). Leave `TestRed_P01_*` red for Phase 5.

---

## Shared Patterns

### Per-(peer, class) rate-limited WARN
**Source:** `internal/app/repl/core.go:853-876` (`discardValidated`) with `discardWarnInterval` (86) and `discardWarnAt` (592-597).
**Apply to:** `bad_sync_header` (PROTO-01), duplicate-node-id alternation (D-06), and the pre-decode version verdict (D-07).
```go
last, warned := c.discardWarnAt[class]
if warned && now.Sub(last) < discardWarnInterval {
	return
}
c.discardWarnAt[class] = now
c.cfg.Log(fmt.Sprintf("sup-kv: discarded inbound datagram: class %s (one warning per class per %s)", class, discardWarnInterval))
```
For D-06 the key must become `(node_id, class)` rather than class alone (research Pattern 4); keep event time (`now`), never a wall clock.

### Bounded queue: cap, dedupe, drop-oldest, count
**Source:** `maxEgressEntries` (97-101), `egressIDs` (560-565), `enqueueResponse` (1146-1162), `dropOldestEgress` (1182-1188), `EgressDropped` (1908-1914).
**Apply to:** SyncID stream accumulation (streams/entries caps), seen-run ring, listing destinations/queued chunk work. One idiom across the core — no goroutine timeouts, no ad-hoc maps.

### Copying accessor → adapter atomic mirror
**Source:** `Discards()` (1798-1800), `MergeStats()` (1802-1808), `SyncArrived()` (1817-1830) + `mirrorStats` (998-1030).
**Apply to:** listing generation/complete/listed keys, new counters. The core is single-goroutine; cross-goroutine readers only ever load an atomic the adapter stored after an event. For the listing, publish `{complete, keys}` in **one** immutable pointer.

### Determinism discipline
**Source:** `jitterOffset`/`jitteredSpacing` (1453-1501), monotone counters in `Core` (550-556), sorted iteration `sortedPendingKeys` / `slices.Sorted(maps.Keys(...))` (1516).
**Apply to:** SyncID (monotone counter, not `crypto/rand`), stream-drop choice (sorted key order), any new emission ordering. Warning limits and build max-age use the handler's `now`.

### Proto regeneration
**Source:** `Makefile:91-100` (`make proto`), `buf.gen.repl.yaml`.
**Apply to:** `supkv.proto` changes. Commit the regenerated `supkv.pb.go`; CI never regenerates; run `make buf-lint` and `make buf-breaking`.

### Harness flag derivation vs. frozen tests
**Source:** `internal/stand/stand.go` `startStandNode` (610-615) + `stand_red_test.go` extras (770-773).
**Apply to:** `--startup-gate-empty` in the stand. The frozen test names only `--startup-gate`; the harness derives the empty twin from it. This is the whole reason the two R-03 rows can flip by TSV alone.

### Frozen-test contract (do not touch)
- `internal/stand/stand_red_test.go`: R-03/F14. It asserts codes only (`Unavailable`, `AlreadyExists`, health `NOT_SERVING`/`SERVING`, next-version-greater) — nothing about the gate message string or the wire fields. No edits.
- `internal/grpcapi/red_gate_test.go`: its fake parks the resolver behind `release` and returns `syncState.Load() != redSyncNone`; it never observes a partial state through the seam, so it passes unmodified when production `SyncArrived` means "complete listing". No edits.

---

## Test Adaptation Wave (concrete anchors)

**repl unit fixtures** — `internal/app/repl/retransmit_test.go`:
```go
func handleFor(nodeID uint16) string { return fmt.Sprintf("n%d", nodeID) }
...
type stateStore struct{ store *storage.Store }
func (s stateStore) ApplyRemote(...) ... { return s.store.Merge(key, rec) }
func (s stateStore) Version(...) ...     { return s.store.Version(key) }
func (s stateStore) Lookup(...) ...      { return s.store.Lookup(key) }
func (s stateStore) Versions(...) ...    { return s.store.Versions() }
...
func datagramFrom(peerID uint16, msg port.Message) port.InboundDatagram {
	return port.InboundDatagram{
		Source: handleFor(peerID), Status: port.Decoded,
		ProtoVersion: repl.SupportedProtoVersion,
		NodeID: peerID, RunID: testRunID(peerID), Msg: msg,
	}
}
```
Add `Observe` to **both** repl fakes — `stateStore` (retransmit_test.go:68-90) and `egressStore` (egress_internal_test.go:43-64) — and make `datagramFrom` able to stamp a complete header (new `syncFrom` helper including `SyncID/ChunkIndex/ChunkCount`).

**sync tests** — `syncFrom` (797-801) is the single DataSync constructor; give it header parameters or add `syncPartFrom(peerID, syncID, index, count, entries...)`, then update `dataSyncs`/`advanceDrain`/`dataSyncsEqual` (273-359) to compare headers where the test's subject is the round. `TestSync_syncArrived` becomes two rows: a partial chunk leaves `SyncArrived()` false; the final index flips it.

**restart tests** — rewrite `TestRestart_newPeerNoFullSync` (158-196) to "first Hello = exactly one targeted listing, repeat = nothing" using `targetedSyncs` (34-55); the stale-return row of `TestRestart_staleRunIdAndInFlight` becomes the alternation config error (no listing, `DuplicateNodeID` counted).

**egress / CR-02** — `egress_internal_test.go` white-box tests (95-200) prove identity/dedupe/overflow; add `TestEgress_identityOnly` asserting a queued `DataReplication` entry carries no `Record` (the field no longer exists) and that a vanished key skips at drain. `egressStore` (43-64) is the mutable fake to reuse.

**listing / CR-01** — new `internal/app/repl/listing_internal_test.go` in `package repl` (same white-box rationale as `egress_internal_test.go:1-9`): many distinct `(node_id, run_id)` Hellos produce bounded destinations/queued work with a drop count; two triggers inside one build age produce one `Versions()` copy (a counting fake `StateStore`).

**wire budget / chunk overhead** — `internal/udprepl/syncsize_internal_test.go`: `wireBudget` stays 1400, `chunkOverheadBytes` 64→96, `wireGraduatedLens` must re-sum to `1400 − 96 − 8×24 = 1112` (e.g. subtract 4 from each of the eight lengths); keep the oversized-single-entry proof and the real-`encodeDatagram` assertion (137-173) unchanged in shape.

**decode table** — `internal/udprepl/decode_negative_test.go` rows (89-140): the wrong-version row currently expects a decoded `Hello`; it becomes `ProtoVersionMismatch` + `repl.DiscardBadProtoVersion` with `Msg == nil` and envelope fields preserved. Add the combined row (future version + corrupted body → `bad_proto_version`, not `bad_crc`).

**mirror** — `internal/udprepl/repl_test.go:618-680`: the DataSync fixture must carry a complete header (one chunk, index 1, count 1) to flip `SyncArrived`; add a partial-listing case asserting `SyncArrived()` stays false and `ListedKey` stays false. Use `EncodeDatagramForTest` / `EncodeDatagramForTestVersion` (`repl_internal_test.go:29-70`).

**grpc gate** — `internal/grpcapi/gate_test.go:56-107`: rewrite `TestGate_writesUnavailableReadsOK` to "reads and writes unavailable" (Get/List/Watch now `Unavailable`), keeping the existing `closedGateOptions` helper (40-45). Add: listed-but-unarrived Get → `Unavailable`; tombstoned listed Get → `NotFound`; `create_if_absent` on a listed key → `AlreadyExists` immediately after health flips (the Pitfall 2 race test).

**grpc harness** — `internal/grpcapi/server_test.go:112-133, 181-211`: add `ListedKey` to `testServerConfig`/`ServerOptions` merging, and a notify capture slice for the dump-failed/restore totality tests (the harness already owns the engine and hub).

**main tests** — `cmd/sup-kv/main_test.go` `startNode` sites: append `--startup-gate-empty=10ms` at the 14 `--startup-gate=10ms` lines; add one selection unit test (empty dump → empty gate; seeded dump → normal gate) by observing health timing, not internals.

**simulator scenarios** — `internal/app/repl/sim_scenario_test.go`: keep `TestScenario_emptyDisk` (1138-1200) and re-derive its bound if first-seen listings shorten recovery; rewrite the `peerEvicted` subtest's lower-bound assertion (Pitfall 6) to "converged at/after the next Hello, before the next sync tick". Add a closed-listing scenario (assert no local write is visible before the complete listing) and a three-node scenario (`TestScenario_threeNode`) using `cluster.AddNode` (simnet.go:634).

**three-node real sockets** — new `internal/udprepl/three_node_test.go`, copying `smoke_test.go`'s recipe: `freeUDPPort` (42-57), `smokeSocket` (68-105), `probeBroadcastAddr` (~250-280), then three stores/cores/adapters on one shared wildcard port via `udprepl.New(bind, broadcast, id, store, cfg, udprepl.WithAddressReuse())` (the two-node shape at 305-324). Assert broadcast convergence only; document the kernel-arbitrary unicast limitation (smoke_test.go:290-300).

**StateStore.Observe compile ripple** — the interface gains one method, so every implementer needs it. Production: `internal/app/storage/store.go` (the real one), `internal/app/persist/engine.go` (delegation). Fakes, all delegating to `store.Observe`:
- `internal/app/repl/retransmit_test.go:68-90` (`stateStore`)
- `internal/app/repl/egress_internal_test.go:43-64` (`egressStore`)
- `internal/udprepl/repl_test.go:47-64` (`stateStore`)
- `internal/udprepl/repl_internal_test.go:85-100` (`trackerStore`)
- `internal/udprepl/smoke_test.go:184-198` (`smokeStore`)
- `internal/udprepl/syncsize_internal_test.go:48-66` (`wireStateStore`)
- `internal/grpcapi/wait_integration_test.go:167+` (`peerStore`)
- `internal/app/internal/simnet/simnet.go:435-449` (`nodeStore`)

`internal/udprepl/scenario_test.go` has no fake of its own (its nodes go through `smokeStore`) and needs no edit once the above compile.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/udprepl/replproto/supkv/v1/supkv.pb.go` | generated code | transform | Regenerated by `make proto`; never hand-edited. Not a pattern target. |
| `docs/04-MULTI-NODE.md` | phase artifact | transform | No same-role predecessor exists; use `docs/OPERATIONS.md` for shape and ADR 0006/0009 + 04-RESEARCH Pattern 7 for content. |

Everything else extends an existing file or copies a named helper; no new mechanism is introduced by this phase.

---

## Metadata

**Analog search scope:** `internal/app/repl`, `internal/app/storage`, `internal/app/persist`, `internal/app/internal/simnet`, `internal/port`, `internal/udprepl` (+ `replproto`), `internal/grpcapi`, `internal/stand`, `internal/metrics`, `cmd/sup-kv`, `docs`, `scripts`.
**Files scanned:** 49 target files + ~30 adjacent helper/test files read for excerpts.
**Tracked-source check:** every named analog verified with `git ls-files --` (all tracked).
**Frozen files (reference only, never edited):** `internal/stand/stand_red_test.go`, `internal/grpcapi/red_gate_test.go`.
**Pattern extraction date:** 2026-11-22.
