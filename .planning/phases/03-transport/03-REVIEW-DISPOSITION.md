---
phase: 03-transport
updated: 2026-11-22T03:00:00Z
total: 13
---

# Code Review Disposition — Phase 03

One row per finding from `03-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**. This ledger is rebuilt
from REVIEW.md on each review run — decisions recorded here are preserved.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | Restart-listing queue unbounded + keyed by attacker-controlled source handle (100 spoofed Hellos → 96 FIFOs / 4 800 queued chunks; O(store) Versions copy per trigger). Fix: bound the per-destination listing set, cap queue depth, dedupe by (node_id, run_id) instead of handle. **Follow-up plan candidate (Phase 4/7 window).** |
| CR-02 | critical | open | Egress queue pins a full record copy per answer (~1.9 GiB at cap) and allocates before dedupe (60 KiB garbage per 65-byte request). Fix: carry identity only; build the message at drain. **Follow-up plan candidate (Phase 4).** |
| WR-01 | warning | open | Tracker leaks when `WaitReplicated` is nil while the notify seam creates trackers. Fix: skip tracking when the wait seam is absent (single-node wiring) or release on the nil path. |
| WR-02 | warning | open | `sync_chunk_bytes` unbounded — typo can OOM first listing or make unsendable chunks. Fix: startup validation (Phase 7's one-point config validation will host it; a cheap bound now if touched). |
| WR-03 | warning | open | `red-r01-x20` evidence step prints a stale committed summary on fresh checkouts. Fix: restrict the evidence print to the current run's artifacts. |
| WR-04 | warning | open | `freeUDPPort` pick-then-bind race reproduced once (3/3 reruns green). Known flake class; Phase 2's deferred-items entry covers it. |
| IN-01 | info | open | Apply-failure WARN names the key and is not rate-limited (remote-triggerable log flood). Phase 7 (OBSV) window. |
| IN-02 | info | open | XPORT-02(1) flatness gate doesn't cover paths that still scan the whole pending set. Note for PERF/Phase 8. |
| IN-03 | info | open | `TestCore_keyCountCostFlat` is a wall-clock ratio gate marked `t.Parallel`. Test-hygiene note. |
| IN-04 | info | open | 50 000-key benchmark fixture packs counters outside `PackVersion`'s documented domain. Note for Phase 8. |
| IN-05 | info | open | "Write-time set" in `wait_for_replication` docs vs the set the core awaits. Phase 6 (APIS) wording. |
| IN-06 | info | open | `storage.MaxKeyBytes` doc claims Set/Delete enforce the ceiling; enforcement lives at the API. Doc fix (Phase 9). |
| IN-07 | info | open | `datagrams_sent_total` counts failed sends (RPL-18 class — deliberately left to Phase 7 per 03-12). |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).

**Critical-finding routing note:** CR-01/CR-02 are availability defects in Phase 3's own queue
machinery, discovered post-flip. They do not block Phase 3's stated exit criteria (reds green,
bounds met) but are flagged for a dedicated fix before the milestone ships — the natural windows
are Phase 4 (protocol) and Phase 7 (robustness/observability).
