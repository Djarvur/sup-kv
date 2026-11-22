---
phase: 04-protocol
updated: 2026-11-22T08:45:00Z
total: 18
---

# Code Review Disposition — Phase 04

One row per finding from `04-REVIEW.md` plus the re-verification's new items. Edit the
**Disposition** column by hand when triaging (`open` / `fixed` / `skipped` / `deferred`); put the
reason in **Source**. This ledger is rebuilt from REVIEW.md on each review run — decisions
recorded here are preserved.

Re-verification verdict (same reviewer, read-only, overlay proofs re-run): CR-01 and WR-01…WR-06
FIXED; IN-01…IN-06 FIXED (IN-03 as the documented doc-only alternative); no new critical or
warning-level defect. Five new info items were raised by that pass (New-1…New-5 below).

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | fixed | `e5b8d0d` (+ doc `41825f8`): listings accepted only from a peer-table `(node_id, run_id)` row; unbound/forged-run listings dropped before clock/accumulation/gate; retained sets expire after 2 × `sync_interval`; no rows for unpublished empty listings. Re-verified: both original proofs re-run (0 keys retained, 0 rows, gate stays closed, counter once; forged run leaves the real set intact); identity echo assessed against D-04/D-01 — one targeted listing still, repeat emits nothing, no loop. |
| WR-01 | warning | fixed | `146d4ee`: broadcast slot reserved in both caps (`maxListingChunks = 33 × 65 536`), broadcast drains first, refused targeted listings re-armed via `listingOwed`; worst case documented. Re-verified by probes. |
| WR-02 | warning | fixed | `dc82f8c`: restart no longer drops retained keys; replacement replaces wholesale, expiry bounds the stale set (`TestRestart_retainsListedKeysUntilReplacement`). Re-verified. |
| WR-03 | warning | fixed | `39b58de`: strictly older notification emits `ResultSuperseded` and resolves the waiter; equal duplicate only counts. Re-verified. |
| WR-04 | warning | fixed | `a6768e3`: `NotifyReplayedWrite` rides the paced record-send queue; `ServerOptions.ReplayNotify` wired in `main`; T-4-20 now matches the code. Re-verified (3 replays → 3 queued, one datagram per pacing slot). |
| WR-05 | warning | fixed | `d115651`: map hint clamped to 8; `TestSync_streamMapHintIndependentOfClaim` fails on the old hint and passes now. Re-verified. |
| WR-06 | warning | fixed | `7ba917e`: `repl.MaxSyncChunkBytes = 65507` clamp pinned cross-seam to `udprepl.MaxDatagramBytes`, startup rejection exit 2, chunk slices sized from appended entries. Re-verified. |
| IN-01 | info | fixed | `2ea996e`: `ServerOptions.SyncArrived` doc corrected to the complete-listing contract. |
| IN-02 | info | fixed | `93dec41`: OPERATIONS discard list now enumerates all eleven classes. |
| IN-03 | info | deferred | `a03e374` softened the `04-MULTI-NODE.md` claim (accessors test-visible until then). The metrics surface for `ListingDropped`/`SyncStreamDropped`/`ListedKeysEvicted`/`NotificationsRegressed` belongs to **Phase 7's OBSV catalog**. |
| IN-04 | info | fixed | `4f8c256`: `replaySnapshot` returns the partial list and marks dirty before returning the error; `Restore`/handler propagate and notify. See New-1 for the residual timer nudge. |
| IN-05 | info | fixed | `6406b63`: `hasFlag`/`flagValue` accept both dash forms and both value forms without cross-matching `startup-gate`; `TestEmptyGateArg_forms` green. |
| IN-06 | info | fixed | `0058a04`: `syncArrived` removed; `SyncArrived()` returns `listingComplete`. |
| New-1 | info | deferred | **Phase 5 (Persistence):** `Restore`'s mid-replay error path calls `markDirty()` without waking the dumper (`e.changed`), so an idle clean dumper may not persist installed records until unrelated activity. Unreachable with today's `storage.Store` (only two handled sentinels). The fix is the nudge; Phase 5 owns the durability loop. |
| New-2 | info | deferred | **Phase 7 (OBSV/classification):** an unbound listing that legitimately overtakes its Hello is counted as the D-06 `duplicate_node_id` configuration error; a dedicated `unbound_listing` class would be honest and joins the Phase-7 discard catalog. Bounded by the per-(node,class) limiter. |
| New-3 | info | deferred | **Phase 8 (Performance):** the paced replay path copies `rec.Data` twice before discarding it (drain re-reads via `Lookup`); ~2.4 GB transient allocation on a 20 000 × 61 440-byte load. Correctness unaffected. |
| New-4 | info | fixed | `57728c2`: repaired the two truncated/orphaned doc comments (`core.go` field comment, `udprepl` `localWrite` duplicate paragraph). |
| New-5 | info | deferred | **Phase 8 (test teeth):** `TestChunkSyncEntries_capacityFollowsEntries` does not pin the WR-06 slice-sizing claim; the harm is covered by `TestConfig_syncChunkBytesClamped`. Strengthen the probe when the benchmark fixtures are revisited. |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).

**Post-fix gates (2026-11-22):** `make lint` clean; `make test` (`go test -race -cover ./...`)
green; `SUPKV_RED_TESTS=1 bash scripts/red-runner` green — 7/7 rows, 0 errors, both mutants
killed; frozen tests, TSV, `.golangci.yml`, `go.mod`/`go.sum` byte-unchanged.
