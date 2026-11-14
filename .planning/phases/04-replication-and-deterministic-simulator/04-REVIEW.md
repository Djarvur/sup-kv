---
phase: 04-replication-and-deterministic-simulator
reviewed: 2026-11-14T19:36:57Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/core.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sync_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replproto/supkv.pb.go
  - internal/udprepl/smoke_test.go
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
  necessity: 0
status: issues_found
---

# Phase 4: Code Review Report (incremental re-review)

**Reviewed:** 2026-11-14T19:36:57Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Incremental re-review of phase 04's contract against the CURRENT file state (post-phase-5 changes: the udprepl copy-on-write pending mirror with drained-row retention, the repl/core.go applyRemote error-path fix, the exported persist `Flush`, and the regenerated `supkv.pb.go`).

Verified sound, with the phase-04 contract re-checked point by point:

- **applyRemote error path (CR-01, 957fd0a)** — `internal/app/repl/core.go:454-470`: a store error logs and returns BEFORE the Ack and BEFORE pending supersession; regression-locked by the "store-rejected apply is never Acked and never supersedes tracking" case in `retransmit_test.go`.
- **Copy-on-write pending mirror** — `internal/udprepl/repl.go:469-535`: fresh map per iteration, every `WaitingOn`/`Acked` slice cloned, wholesale `atomic.Pointer` swap, published maps never mutated; the held-snapshot stability is test-pinned. One defect found in the new drained-row retention (WR-01 below).
- **persist `Flush` (86d33a6)** — correct delegation to `flush` with the dirty/clean/closed contract tested (`TestFlush_remoteDirtiness`, `TestFlush_clean`, `TestFlush_closed`); the flush-vs-Run-exit races remain covered by the synctest suite.
- **Envelope-domain validation before the CRC gate** (`udprepl/repl.go:601-617`): empty/undecodable envelope, `node_id > 0xFFFF`, `run_id != 16 bytes` all classify Undecodable before the CRC check; BadCRC carries the envelope fields; body faults classify Undecodable only after a passing CRC.
- **Own-echo on run_id equality and both duplicate-node_id variants** (`repl/core.go:315-349`), with the restart branch (different run_id) never pre-empted by the classifier.
- **Retransmit ladder** 200/400/800/1600/3200ms saturating at 5000ms, strictly-future `Advance` deadlines, total suspension freeze on the empty peer table, and flush-on-discovery only on the empty-to-non-empty transition — all pinned by `TestRetransmit_backoffLadder` / `TestRetransmit_suspendAndFlush`.
- **Egress limiter**: one FIFO over the three reactive-unicast producer sites, floor(1s/rate) integer spacing, first-immediate, late-Advance clamping; restart-targeted listings bypass the limiter via direct `out.Unicast` in `handleHello`.
- `go build`, `go vet` and `go test -race` pass for all three reviewed packages.

Two warnings found, both in code this re-run was asked to re-examine: a duplicate node id the new drained-row union can publish (WR-01), and a config-domain edge that silently disables the egress limiter (WR-02). The generated `supkv.pb.go` carries a clean protoc-gen-go v1.36.12-devel header and a structurally consistent 8-message schema matching the adapter's codec use; no hand-edits detected. No structural-findings block was provided; no necessity findings met the evidence rule (the defensive guards examined — `maxNodeIDWire`, empty-datagram, oversize-chunk, flush-race handling — all guard trust-boundary or test-proven-reachable states).

## Warnings

### WR-01: Drained-row retention can publish a duplicate node id in Acked

**File:** `internal/udprepl/repl.go:513-523`
**Issue:** `retainDrained` builds the resolved row's Acked as `append(slices.Clone(prev.Acked), prev.WaitingOn...)` then `slices.Sort` — a plain union with no dedup. The two sets are NOT disjoint: `handleHello`'s discovery flush re-adds a peer to `entry.waitSet` without touching `entry.acked`, so a peer that Acked while the table was empty (recorded in `acked`, entry retained in suspension) and is then discovered again sits in BOTH sets. The natural production sequence:

1. Table empty; local write under key K — wait-set seeded empty (`NotifyLocalWrite`, core.go:623-626).
2. Peer B's Ack for (K, ver): `handleAck` records `acked=[B]`; the drain rule's `len(c.peers) > 0` guard fails (0 peers) so the entry is retained with `WaitingOn=[]`, `Acked=[B]` — exactly the state `TestMirror_pendingAckVisible` pins.
3. B's Hello ends the empty table (discovery): the flush adds B to the wait-set and broadcasts the record; `acked` stays `[B]`. Entry now `WaitingOn=[B]`, `Acked=[B]`.
4. B receives the flushed re-delivery and re-Acks (the always-Ack-on-duplicate rule): wait-set empties with peers present, the core deletes the entry.
5. `retainDrained` correlates the Ack, computes `clone([B]) + [B] = [B, B]`, sorts (both kept) and publishes the retained row with `Acked: [B, B]`.

This violates the documented "union is exactly the core's acked set at deletion" invariant (the core's acked at deletion is `[B]`, len 1) and the sorted-idempotent set discipline `PendingInfo.Acked` carries everywhere else; a `wait_for_replication` consumer counting `len(Acked)` over-reports by the duplicate.
**Fix:** Dedupe the union — both inputs are sorted, so `slices.Compact` after the sort:

```go
acked := append(slices.Clone(prev.Acked), prev.WaitingOn...)
slices.Sort(acked)
acked = slices.Compact(acked)
```

### WR-02: SyncResponseRate above 1e9 silently disables the egress rate limiter

**File:** `internal/app/repl/core.go:122-124` (normalization), `core.go:285` (division); effect at `core.go:595-605`
**Issue:** `normalized()` guards only `SyncResponseRate <= 0`; `syncResponseInterval = time.Second / time.Duration(cfg.SyncResponseRate)` integer-divides to ZERO for any rate above 1e9. With a zero interval, `drainEgress`'s loop condition never goes false on slot advance (`nextSendAt.Add(0)` stays at `now`), so one `Advance` emits the ENTIRE queued response set in a single burst — the limiter's floor(1s/rate) spacing and its flood bound (REPL-05, T-04-07's empty-node recovery guarantee) are silently gone, with no error surfaced. The value is reachable from production config: `cmd/sup-kv/main.go:244` exposes `--sync-response-rate` / `SUPKV_SYNC_RESPONSE_RATE` as an unbounded int, and `finalizeConfig` (main.go:255-267) validates only the node-id range and required knobs. (The strictly-future `Advance` deadline invariant survives vacuously — the zero-interval loop empties the queue before `nextDeadline` runs.)
**Fix:** Clamp the pathological domain in `normalized()`:

```go
if c.SyncResponseRate <= 0 || c.SyncResponseRate > int(time.Second) {
    c.SyncResponseRate = defaultSyncResponseRate
}
```

(equivalently, clamp the interval to at least 1ns after the division), so an out-of-domain rate normalizes to the default instead of disabling the limiter.

## Info

### IN-01: NotifyLocalWrite pre-check leaves a narrow nil-return window during Run's exit

**File:** `internal/udprepl/repl.go:327-338`
**Issue:** The `<-a.done` pre-check narrows but does not close the TOCTOU: between the non-blocking check and the buffered send, Run can complete its final iteration and run `defer close(a.done)`; if the notify buffer has room, the select may then pick the send case (uniform random choice among ready cases) and return nil for a notification nobody will ever drain. The doc contract ("once Run has returned ... ErrRunLoopExited") still holds literally — the race happens before Run has returned — and the consequence is bounded (the write persists locally via the engine and reconciles after restart), so this is an accepted-risk note, not a correctness break.
**Fix:** Re-check `done` after a successful send and report `ErrRunLoopExited` (accepting the enqueue loss), or document the at-most-once shutdown semantics on the method.

### IN-02: Restore marks the engine dirty unconditionally, forcing a redundant dump on a no-op replay

**File:** `internal/app/persist/engine.go:491-498`
**Issue:** `Restore` calls `e.markDirty()` even when the replay loop changed nothing — an empty decoded snapshot, or one whose every record skips (tombstones onto a fresh store). On an otherwise clean engine the final `flush` then performs a full dump of identical state, against the D-10 no-redundant-save discipline `flushDump` enforces elsewhere. (Any snapshot with at least one live record always writes fresh versions via `store.Set`, so the dirty mark is justified there — the redundancy is limited to the all-skip shapes.)
**Fix:** Track whether any `replayRecord` call actually wrote (a boolean set in the non-skip branches is simplest, since `store.Set` re-stamps even identical values) and skip `markDirty` when nothing replayed; the final `flush` then returns nil without a save on a clean engine.

---

_Reviewed: 2026-11-14T19:36:57Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
