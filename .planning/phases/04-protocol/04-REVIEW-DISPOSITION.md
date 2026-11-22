---
phase: 04-protocol
updated: 2026-11-22T07:40:00Z
total: 13
---

# Code Review Disposition — Phase 04

One row per finding from `04-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**. This ledger is rebuilt
from REVIEW.md on each review run — decisions recorded here are preserved.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | Retained listed-key union filled from unauthenticated, unbound senders, no expiry: one forged 1-of-1 DataSync permanently denies arbitrary keys, flushes a real peer's set (reopens F14) and opens the gate past the 60 s empty backstop. **Fix in phase (fixing now):** bind retention to a known peer row with matching run_id, expire retained sets, cap the per-peer row count. |
| WR-01 | warning | open | Destination cap × queued-chunk cap multiply; broadcast can be starved and a refused first-seen listing is lost forever. **Fix in phase:** reserve the broadcast slot, make the destination cap binding for realistic rounds, re-arm a refused first-seen listing. |
| WR-02 | warning | open | Restart drops the peer's retained keys before the replacement listing arrives — reopens the F14 listed-but-unarrived hole. **Fix in phase:** delete the restart-time drop; retain replaces wholesale on the next complete listing. |
| WR-03 | warning | open | Notification guard swallows `Superseded`, so a raced waiter blocks its full timeout and the D-05 counter conflates supersession with regression. **Fix in phase:** emit `Superseded` on the strictly-older branch. |
| WR-04 | warning | open | Snapshot load notifies every replayed record as an unpaced immediate broadcast, contradicting T-4-20. **Fix in phase:** paced replay path through the record-send queue. |
| WR-05 | warning | open | `syncStream.got` pre-sized from the attacker-supplied `chunk_count` (~1.2 MB per stream). **Fix in phase:** clamp the hint; add a white-box allocation-independence probe. |
| WR-06 | warning | open | Listing-build cache pins an out-of-range `sync_chunk_bytes` allocation for the process lifetime. **Fix in phase:** clamp `SyncChunkBytes` to the datagram ceiling at startup and size the slice from appended entries. |
| IN-01 | info | open | `ServerOptions.SyncArrived` doc still describes pre-D-01 "any DataSync counts". **Fix in phase:** doc correction. |
| IN-02 | info | open | OPERATIONS discard-class list omits five of eleven classes. **Fix in phase:** enumerate all eleven. |
| IN-03 | info | deferred | Four new counters have no production surface. **Phase 7 (OBSV catalog)** owns the metrics families; this fix pass softens the `04-MULTI-NODE.md` claim to name the accessors as test-visible until then. |
| IN-04 | info | open | `Engine.Restore` mid-replay error loses installed-but-unreturned records. **Fix in phase:** always mark dirty and return the partial list with the error. |
| IN-05 | info | open | `emptyGateArg` misses `--startup-gate 30s` (space-separated) and single-dash forms. **Fix in phase:** accept both forms; table-driven test. |
| IN-06 | info | open | `syncArrived` duplicates `listingComplete`. **Fix in phase:** delete the duplicate field. |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).
