---
phase: 04-protocol
reviewed: 2026-11-22T07:26:18Z
depth: deep
files_reviewed: 52
files_reviewed_list:
  - README.md
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - docs/04-MULTI-NODE.md
  - docs/EXAMPLES.md
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - internal/app/internal/simnet/simnet.go
  - internal/app/internal/simnet/simnet_ext_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_replay_internal_test.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/bench_test.go
  - internal/app/repl/core.go
  - internal/app/repl/decision_coverage_test.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/egress_internal_test.go
  - internal/app/repl/listedkeys_internal_test.go
  - internal/app/repl/listing_internal_test.go
  - internal/app/repl/merge_stats_test.go
  - internal/app/repl/peertable_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/repl/sync_test.go
  - internal/app/repl/warn_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/gate_test.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait_integration_test.go
  - internal/grpcapi/watch.go
  - internal/grpcapi/watch_senderr_test.go
  - internal/metrics/metrics.go
  - internal/metrics/metrics_test.go
  - internal/port/repl.go
  - internal/stand/stand.go
  - internal/stand/stand_metrics_test.go
  - internal/stand/stand_test.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/fuzz_decode_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replproto/supkv/v1/supkv.pb.go
  - internal/udprepl/replproto/supkv/v1/supkv.proto
  - internal/udprepl/syncsize_internal_test.go
  - internal/udprepl/three_node_test.go
  - scripts/red-expectations.tsv
  - scripts/red-runner
findings:
  critical: 1
  warning: 6
  info: 6
  total: 13
  necessity: 1
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-11-22T07:26:18Z
**Depth:** deep
**Files Reviewed:** 52
**Status:** issues_found

## Summary

Reviewed the complete Phase-4 (04-protocol) change scope against D-01…D-09: the `DataSync`
completeness header and the per-`(node_id, run_id, sync_id)` accumulator, the retained
listed-key union and its one-pointer mirror, the restart/first-seen listing machinery with the
shared generation-checked build (CR-01), the identity-only egress queue (CR-02), the three
replication outcomes, the identity-only notification guard, the duplicate-identity ring, the
pre-body `proto_version` verdict, the gate/read/listed-key RPC contract, totality for dump-failed
writes and snapshot loads, the N=3 evidence and the red-gate flip.

**Gates run for this review.** `go build ./...` clean; `go tool golangci-lint run ./...` clean
(exit 0, no new suppressions); `go test -race -count=1` green for `internal/app/repl` (75 s),
`internal/udprepl`, `internal/grpcapi`, `internal/app/persist`, `internal/app/storage`,
`internal/metrics`. `git diff d9042cb -- internal/stand/stand_red_test.go
internal/grpcapi/red_gate_test.go` is empty (both frozen tests byte-unchanged);
`.golangci.yml`, `go.mod` and `go.sum` are untouched; `scripts/red-expectations.tsv` flips exactly
the two R-03 rows and leaves both P-01 rows red. Depguard boundaries hold: no new direct
`net`/`os`/`syscall`/`google.golang.org/*` import inside `internal/app`, `internal/port` stays
stdlib+own-module, and every added `//nolint` names its linter with a reason.

**The two Phase-3 criticals are genuinely closed.** The egress queue is `struct{ id egressKey }`
only, dedupe precedes every producer's probe, and the drain re-reads through `Lookup`
(`core.go:497-499`, `1832-1908`); listings are keyed by `(node_id, run_id)` with a shared build, a
destination cap, a queued-chunk cap and a counted drop (`core.go:505-546`, `2067-2113`).
`handleHello`'s first-seen branch falls through to the discovery flush (Pitfall 5 is handled and
pinned by `TestRestart_firstHelloStillFlushesPending`), equal-version `DataReplication` still
Acks, and the per-part `Observe` sits after validation.

**The one shipped blocker is new and security-relevant:** the retained listed-key union is
installed from *any* datagram identity, with no peer-table or `run_id` binding and no expiry, so
one unauthenticated 1-of-1 listing can make arbitrary key names permanently unreadable and
uncreatable, and can flush a legitimate peer's protective set (reopening F14). I proved both
variants with an overlay test against the real core (no tree modification; see CR-01). The six
warnings are listing-scale envelope arithmetic, the restart-time listed-key drop, a lost
`Superseded` resolution on the notification guard, the unpaced snapshot-load burst that
contradicts its own threat row, and one allocation amplification in the new stream accumulator.

## Narrative Findings (AI reviewer)

### Critical Issues

#### CR-01: The retained listed-key union is filled from unauthenticated, unbound senders and never expires — one datagram permanently denies reads/writes for arbitrary keys and flushes a real peer's protective set

**File:** `internal/app/repl/core.go:1560-1601` (`accumulateSync` completion → `retainListedKeys(in.NodeID, …)`), `1729-1759` (`retainListedKeys`), `2315-2352` (`evictPeers`, the only expiry path), `1303-1307` (restart drop); consumed at `internal/grpcapi/server.go:141-144` and `:190-192`, `internal/udprepl/repl.go:892-899`
**Issue:** `handleDataSync` accepts a listing from *any* decoded envelope identity. Nothing requires the sender to be a peer-table row, and `retainListedKeys` keys the retained set by the envelope `in.NodeID` alone — the envelope `run_id` is never compared with the row's `run_id`, unlike every other stateful path. Two consequences, both proved with an overlay test (`go test -overlay=…`, package `repl`, no tree change):

1. **Unknown identity, permanent state.** One `DataSync{sync_id:1, chunk_index:1, chunk_count:1, entries:[victim/key]}` from a `node_id` that never sent a Hello installs `victim/key` in the union. No expiry exists: `dropListedPeer` runs only from `evictPeers` (which iterates `c.peers`, and a DataSync never creates a row) and from a restart Hello of that same id. Measured: after `HandleDatagram` plus `Advance(now+24h)`, `ListedKeysSnapshot()` still returns `victim/key` with `len(core.peers) == 0`. The same datagram also flips `listingComplete`/`syncArrived`, so the resolver opens the gate within one 20 ms poll — the empty-disk 60 s backstop (D-02) is bypassed by the attacker's own forged listing.
2. **Known peer, forged run_id.** `retainListedKeys` *replaces* the whole per-node set, and the duplicate-identity classifier only fires on run_id *equality* (`core.go:985-993`), so a `DataSync` from a known `node_id` with any different run_id passes the classifier, reaches `handleDataSync`, and overwrites that peer's set. Measured: after the legitimate peer listed `legit/listed`, one datagram claiming the same `node_id` with `run_id=0xEE…` left the union as `{attacker/key}` only, with the peer row untouched.

Impact through the shipped RPC layer, with the gate open (the attacker opens it themselves in variant 1): `Get` on `victim/key` or `attacker/key` returns `Unavailable` forever (`server.go:141-144`) — a key that exists nowhere is unreadable; `create_if_absent` returns `AlreadyExists` forever (`server.go:190-192`) — the name is permanently uncreatable; and dropping `legit/listed` from the union re-opens the exact F14 path this phase exists to close, since `create_if_absent` on that key now succeeds locally and its fresh version overwrites the peer's value by LWW. Because the union is capped at `maxListedKeysTotal` with whole-set LRU eviction, ~65 datagrams of attacker keys also flush legitimate peers' sets on demand. The memory bound (T-4-41) holds; the *honesty* dimension the union is trusted for does not. Additionally, an empty listing from an unknown id (`touchListedPeer`, `core.go:1763-1770`) creates a `listedByPeer` row and an LRU element while `listedKeyCount` stays 0, so the LRU cap never applies to those rows: 65 536 empty listings grow both maps to the full uint16 domain un-evicted.
**Fix:** bind retention to a real peer identity and make it expire:
```go
// accumulateSync, before retaining:
known, ok := c.peers[in.NodeID]
if !ok || known.runID != in.RunID {
    c.discards.DuplicateNodeID++ // or a new "unbound_listing" class
    c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf("node_id %d listing from unbound run at handle %q", in.NodeID, in.Source))
    return // completeness/gate stay as they are; nothing is retained
}
changed := c.retainListedKeys(in.NodeID, stream.entries)
```
plus (a) a `listedAt time.Time` per retained peer dropped when `now.Sub(listedAt) >= 2*cfg.SyncInterval` (a peer that stops listing must stop protecting keys), and (b) a cap on the number of `listedByPeer` rows, or skip `touchListedPeer`'s row creation for an empty listing that was never published.

### Warnings

#### WR-01: The destination cap and the queued-chunk cap multiply, so at the documented scale only a few listings fit and a refused first-seen listing is never retried

**File:** `internal/app/repl/core.go:121-129` (the two caps), `:2067-2089` (`enqueueListing`), `:1317-1320` (`handleHello` row updated before the enqueue result), `:2048-2056` (`syncListing`)
**Issue:** `maxListingDestinations = 32` is only reachable when a round is ≤ `65536/32 = 2048` chunks. `chunkSyncEntries` packs one entry per chunk once `2*(len(key)+24) + 96 > 1400`, i.e. keys longer than 628 bytes — a supported and tested size (the 1 024-byte ceiling, `TestStand_oversizeKeyF2c`). A documented-scale store of 20 000 such keys is a 20 000-chunk round, so `enqueueListing` accepts 3 destinations (60 000 ≤ 65 536) and refuses the fourth forever, and at 500 chunks/s one round alone takes 40 s — longer than the 30 s `sync_interval`. Two consequences: (a) the periodic broadcast competes for the same 3 slots and is refused whole at its next tick(s) while they drain, so its effective period becomes 60-120 s rather than 30 s; (b) a refused first-seen/restart listing is *permanently* lost for that run: `handleHello` upserts the row before calling `enqueueListing` and does not roll back, so the peer's next Hello takes the repeat branch and emits nothing — PROTO-04's "exactly one full listing toward a newly seen peer" becomes "zero, counted". The same arithmetic means a listing larger than `maxListingChunks` (a store that chunks above 65 536) can never be enqueued at all, and a peer's listing that accumulates more than `maxSyncEntriesPerStream` (65 536) entries is dropped whole by `enforceSyncBounds`, so a store above 65 536 keys cannot open a joiner's gate through the listing path — although `maxListedKeysTotal` (131 072) claims to model that scale.
**Fix:** stop using queued chunk *count* as the memory bound (a FIFO now costs O(1) — one pointer to the shared build), and bound what actually matters: reserve the broadcast slot so it can never be starved (`if dst == broadcastDst() || len(c.listingByDst) < maxListingDestinations`), raise `maxListingChunks` to at least `maxListingDestinations ×` the worst-case round (or drop the chunk cap in favour of the destination cap plus a drain-time `startIndex`), and re-arm a refused first-seen listing (keep the FIFO request pending, or retry at the next `enqueueListing` for that dst) instead of losing it. Record the resulting `2 × sync_interval` worst case in `docs/04-MULTI-NODE.md`.

#### WR-02: A restart drops the peer's retained listed keys before its replacement listing arrives, reopening the listed-but-unarrived hole (F14)

**File:** `internal/app/repl/core.go:1303-1305` (`dropListedPeer` on the restart branch), `:2329-2331` (the equivalent drop on eviction)
**Issue:** A run_id change immediately removes the peer's whole listed-key contribution from the union and bumps the generation, while `listingComplete` stays true and the gate stays open. Until the restarted peer's own complete listing arrives (at best one hello interval plus pacing, and longer under WR-01's back-pressure), `ListedKey` answers false for every key that peer holds, so `Get` returns `NotFound` instead of `Unavailable` and `create_if_absent` succeeds — the F14 overwrite the phase's D-01 acceptance forbids. The replacement path makes the drop redundant: `retainListedKeys` already replaces the peer's set wholesale on the next complete listing, so removing it first only widens the window. A spoofed restart Hello (CR-01 variant 2 via the Hello path) triggers the same drop at will.
**Fix:** keep the previous run's keys until the new run's first complete listing replaces them; `retainListedKeys`'s replace-on-completion already bounds the set, so the restart-time `dropListedPeer` can be deleted (eviction keeps its drop — the peer is gone).

#### WR-03: The notification guard drops a superseded notification without emitting `ResultSuperseded`, so a tracked waiter blocks for its full timeout

**File:** `internal/app/repl/core.go:1932-1936` (guard), `:2018-2026` / `:2843-2850` (the supersession path that does emit), `internal/udprepl/repl.go:693-763` (tracker created at notify, resolved only by results)
**Issue:** `Adapter.NotifyLocalWrite` creates the tracker synchronously for `track=true` before enqueueing, and only `applyResults` resolves it. Two concurrent writes to one key can reach the notify channel in the opposite order from the store: `Set(v_A)` stamps first, `Set(v_B)` second, but `lw_B` is dequeued before `lw_A`. Processing `lw_B` tracks `v_B`; processing `lw_A` then takes the guard's `rec.Version.Compare(tracked) <= 0` branch, which increments `notificationsRegressed` and returns — no `Superseded` result for `(key, v_A)`. `WaitReplicated(key, v_A, …)` therefore finds an unresolved tracker and waits the whole `wait_timeout_ms` (client-chosen) before returning `(false, 0)`, although `v_A` was displaced by a strictly greater *local* write and `removeSupersededLocal` would have emitted exactly that result had `v_A` been tracked. The same reordering is counted as a "regressed notification" although nothing regressed (the mirror correctly keeps `v_B`), so the new counter conflates D-05's regression signal with ordinary write supersession.
**Fix:** resolve the superseded identity on the guard's strictly-older branch:
```go
if tracked, ok := c.trackedVersion(key); ok {
    if cmp := rec.Version.Compare(tracked); cmp < 0 {
        out.Superseded(key, rec.Version) // displaced by a greater local write; the waiter must resolve
        c.notificationsRegressed++      // keep the D-05 counter, or split it into supersededNotify
        return
    } else if cmp == 0 {
        c.notificationsRegressed++
        return
    }
}
```
(`applyResults` drops a result for an absent tracker, so this is safe for untracked writes.)

#### WR-04: Snapshot load notifies every replayed record as an immediate unpaced broadcast — contradicting T-4-20 and bursting one datagram per key

**File:** `internal/grpcapi/server.go:329-347` (notify loop), `internal/app/repl/core.go:1959-1965` (`NotifyLocalWrite` → `out.Broadcast`), `internal/udprepl/repl.go:614-646` (channel enqueue only), `.planning/phases/04-protocol/04-06-PLAN.md:198` (T-4-20)
**Issue:** T-4-20's mitigation states "Replayed records ride the existing paced record-send queue; notify is enqueue-and-return, never a synchronous broadcast burst." The code does the opposite: `notifyLocalWrite` per replayed key only crosses the 1024-deep notify channel, and the core's `NotifyLocalWrite` goes straight to `out.Broadcast` — the paced `recordQueue` is used only by the discovery flush and the retransmission ladder. A `SnapshotLoad` of the documented 20 000-key store therefore emits 20 000 `DataReplication` broadcasts back-to-back (at the 61 440-byte value ceiling, ~1.2 GB of datagrams against a 64-deep receiver inbound queue), losing most of them and leaving the paced ladder to re-deliver over ~40 s, while the gRPC handler blocks on the notify channel once the event loop falls behind. This is the un-paced initial burst the Phase-3 pacers removed everywhere else.
**Fix:** add a paced path for replayed records (e.g. `Core.NotifyReplayedWrite` that calls `enqueueRecordSend` for the already-installed record, or have the recall/load seam enqueue `recordSend` entries) and keep the immediate `Broadcast` for live writes only; or, if the burst is accepted, correct T-4-20 and `docs/04-MULTI-NODE.md` and document the recovery-time consequence.

#### WR-05: The new stream accumulator pre-sizes its chunk-index map from the attacker-supplied `chunk_count`

**File:** `internal/app/repl/core.go:1563-1572`
**Issue:** `make(map[uint32]struct{}, msg.ChunkCount)` sizes a map for a claim of up to `maxSyncChunks = 65 536` entries from a datagram that is only required to carry one valid index. Measured on the review host: 16 such maps (exactly `maxSyncStreams`) allocate 19 488 824 bytes — 1 218 052 bytes each. Sixteen ~60-byte `DataSync` datagrams (one entry, `chunk_count: 65536`, distinct `sync_id`s) therefore buy ~18.6 MiB of retained heap; the state is bounded, but the amplification is ~13 000× and it is remotely repeatable (evict-then-refill costs only the same 16 maps). Every other new bound is proportional to received bytes; this one is proportional to a claimed count.
**Fix:** never hint from the claim, or clamp it: `got: make(map[uint32]struct{}, min(int(msg.ChunkCount), 8))` (the map grows as chunks actually arrive, which `maxSyncEntriesTotal` already bounds), and add a white-box probe asserting the per-stream allocation is independent of `chunk_count`.

#### WR-06: The listing-build cache turns an out-of-range `sync_chunk_bytes` into a permanently retained allocation

**File:** `internal/app/repl/core.go:2100-2113` (`listingBuild` stores `c.build`), `:2280` (`cur := make(…, 0, budget/entryOverheadBytes)`), `:285-287` (only `<= 0` normalizes), `cmd/sup-kv/main.go:358` (no range validation)
**Issue:** Phase-3 WR-02 (open, deferred to Phase 7) records that `--sync-chunk-bytes` is unbounded and that the first `listingChunks()` pre-allocates `budget/entryOverheadBytes` entries — ~41.6 M `SyncEntry` values (~1.6 GB) at 1 000 000 000. This phase changes the *lifetime* of that allocation: `listingChunks()`'s result is now stored in `Core.build` and reused until the store generation or `sync_interval` age changes, so the oversized first chunk (the one carrying the pre-allocated capacity) is pinned for the process lifetime instead of being transient garbage after one emission. The same retention applies on every rebuild, so an operator typo now guarantees a standing OOM risk rather than a spike.
**Fix:** clamp `Config.SyncChunkBytes` to the datagram ceiling in `normalized()` and reject an explicit out-of-range flag at startup (the Phase-7 catalog can own the final wording), and size the chunk slice from the entry estimate of the entries actually appended rather than from `budget/entryOverheadBytes`.

### Info

#### IN-01: `ServerOptions.SyncArrived`'s doc still describes the pre-D-01 "any DataSync counts" semantics

**File:** `internal/grpcapi/gate.go:48-54`
**Issue:** The field comment says "Production injects the udprepl adapter's atomic mirror (A1 semantics: any received DataSync datagram counts)". Since D-01 the adapter reports a *complete* listing (`mirrorStats` → `SyncArrived`), and the same file's `gated`/`RunGate` doc now describes the completeness contract. A future wiring author reading only the option comment would expect the old signal.
**Fix:** replace "any received DataSync datagram counts" with "true once a complete listing round has been processed".

#### IN-02: `docs/OPERATIONS.md`'s discard-class list omits five of the eleven published classes

**File:** `docs/OPERATIONS.md:152-155`
**Issue:** The phase edited this list to add `bad_sync_header`, but it still names only `bad_crc`, `undecodable`, `bad_proto_version`, `own_broadcast`, `duplicate_node_id` and `bad_sync_header`, while `metrics.go:330-340` publishes `empty_key`, `key_too_long`, `tombstone_with_data`, `bad_checksum` and `bad_version` too. An operator following the doc will not know five labels exist.
**Fix:** enumerate all eleven labels (or point at the metrics family as the catalog) and keep `docs/SPEC-DEVIATIONS.md`'s class references aligned.

#### IN-03: Four new counters have no production surface, while the phase artifact claims nothing is dropped silently

**File:** `internal/app/repl/core.go:2565-2572`, `:2618-2630`, `:2717-2724`; `docs/04-MULTI-NODE.md:97-99`
**Issue:** `NotificationsRegressed`, `SyncStreamDropped`, `ListedKeysEvicted` and `ListingDropped` are exported accessors with no reader outside tests: no metrics family, no adapter mirror, no log. `04-MULTI-NODE.md` states "Each refused or evicted item is counted by an exported accessor; nothing is dropped silently" — in a shipped binary with `/metrics` enabled, all four are invisible, including the two that signal an active hostile flood (CR-01/WR-01/WR-05).
**Fix:** mirror `ListingDropped`, `SyncStreamDropped` and `ListedKeysEvicted` in `Adapter` and publish them from `metrics.Sources` (Phase 7's catalog can rename them), or soften the doc claim to name the accessors as test-visible until the OBSV surface lands.

#### IN-04: `Engine.Restore`'s mid-replay error path loses installed-but-unreturned records

**File:** `internal/app/persist/engine.go:627-650`
**Issue:** When `replayRecord` returns an error, the loop returns `nil, err` although earlier iterations already installed records in the store: `e.markDirty()` (line 644) is never reached, so those records are never dumped, and the empty list means `SnapshotLoad` never notifies them (PROTO-08's totality). Today this is unreachable — `storage.Store.Set` returns only `ErrValueTooLarge` and `Delete` only `ErrNotFound`, both handled as skips — but the return shape invites the loss as soon as the store gains another error.
**Fix:** collect the first error, finish or abort deterministically, always `e.markDirty()` when anything was installed, and return the partial `restored` list with the error (the handler already notifies on non-`ErrClosed` errors).

#### IN-05: The stand harness's empty-gate derivation misses the space-separated and single-dash flag forms

**File:** `internal/stand/stand.go:619-633`
**Issue:** `emptyGateArg` recognizes `--startup-gate-empty` and `--startup-gate-empty=…`, and derives the value only from `--startup-gate=…`. A scenario written as `--startup-gate 30s` (valid for Go's flag package, and the form README's examples use for `--startup-gate-empty`) is not detected, so the harness appends a derived `--startup-gate-empty=1s` and the node's empty gate becomes 1 s while its normal gate is 30 s — precisely the race RESEARCH Pitfall 1 exists to prevent (the closed-side assertions would observe an opened gate under load). Same for `-startup-gate-empty=…`.
**Fix:** normalize each extra with `strings.TrimLeft(arg, "-")` and accept both `name=value` and the `name value` pair form when scanning, and add one table-driven unit test over the forms.

#### IN-06: `syncArrived` duplicates `listingComplete` — the two fields can never differ

**File:** `internal/app/repl/core.go:1591-1599` (both writes), `:815` / `:848` (fields), `:2583-2602` (accessors)
**Category:** necessity
**Issue:** Subtype: already-defended-elsewhere (duplicated state). `syncArrived` and `listingComplete` are written in exactly one block: `listingComplete` is set once under `if !c.listingComplete`, and `syncArrived = true` is then assigned unconditionally in the same completion branch; neither is ever written false and no other site writes either. The invariant `c.syncArrived == c.listingComplete` therefore holds at every observable point, so the second field carries no independent information — `SyncArrived()` could return `c.listingComplete` (the frozen seam *name* is what the red test needs, not a second boolean). Proof: `grep -n 'syncArrived' internal/app/repl/core.go` yields only the field declaration, this one assignment and the accessor; the same holds for `listingComplete`.
**Fix:** delete `syncArrived` and return `c.listingComplete` from `SyncArrived()`; the frozen test and the adapter mirror are unaffected.

---

_Reviewed: 2026-11-22T07:26:18Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
