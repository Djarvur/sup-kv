---
phase: 04-protocol
fixed_at: 2026-11-22T08:16:33Z
review_path: .planning/phases/04-protocol/04-REVIEW.md
iteration: 1
findings_in_scope: 13
fixed: 13
skipped: 0
status: all_fixed
---

# Phase 04: Code Review Fix Report

**Fixed at:** 2026-11-22T08:16:33Z
**Source review:** `.planning/phases/04-protocol/04-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 13 (CR-01, WR-01…WR-06, IN-01…IN-06)
- Fixed: 13
- Skipped: 0

All fixes landed in the main checkout (`workflow.use_worktrees` is `false` in
`.planning/config.json`, so no worktree was created). `make lint` and
`make test` (`go test -race -cover ./...`) ran in the main checkout before
this report was written; the frozen tests, `.golangci.yml`, `go.mod`, `go.sum`
and `scripts/red-expectations.tsv` are byte-untouched by this fix pass
(`git diff bebfb4f..HEAD -- <those paths>` is empty).

## Fixed Issues

### CR-01: The retained listed-key union is filled from unauthenticated, unbound senders and never expires

**Status:** fixed: requires human verification
**Files modified:** `internal/app/repl/core.go`, plus the listing fixtures in
`internal/app/repl/{bench,discard,egress_internal,listedkeys_internal,merge_stats,peertable,restart,retransmit,sync}_test.go`
and `internal/udprepl/{repl_internal,repl}_test.go`
**Commits:** e5b8d0d (code + tests), 41825f8 (the trust-binding paragraph in
`docs/04-MULTI-NODE.md` and the old-run DataSync note in `handleHello`'s doc)
**Applied fix:**
- `Core.listingBound` (called first in `handleDataSync`, after header
  validation) accepts a listing only when the envelope `(node_id, run_id)`
  matches a live peer-table row. An unbound identity or a forged `run_id` is
  counted as `duplicate_node_id` (the named configuration-error class from
  D-06), fires `OnDiscard`, WARNs through the shared per-`(node_id, class)`
  limiter, and drops the datagram before `Observe`, accumulation, the
  completeness flag, the listing generation and the gate — no retention, no
  completeness, no gate opening.
- Retention is bounded in time: `listedAt` records each set's last publish;
  `expireListedKeys` (from `Advance`, with a `nextDeadline` contribution)
  releases a set nobody republished for `2 × sync_interval`, counts each
  released key in `ListedKeysEvicted`, and moves the listing generation. The
  peer row and `ListingComplete` stay (expiry is not an eviction, not a
  re-close).
- `touchListedPeer` no longer creates a `listedByPeer` row for an empty
  listing that was never published, so listing-shaped datagrams cannot grow
  the row maps while `listedKeyCount` stays 0.
- **Mechanism note (identity echo):** the binding makes a peer's row a
  precondition for accepting its listing, but PROTO-04's first-seen/restart
  listing is triggered by the *receiver's* Hello — the lister may not have
  heard the receiver yet. `handleHello` therefore emits one **unicast identity
  echo** (`port.Hello`) to the triggering handle on the first-seen and restart
  branches, flushed ahead of the paced listing chunks. The exchange terminates
  (the receiver of an unsolicited echo answers with its own echo and listing,
  which reaches us as a repeat Hello), so each side echoes at most once per
  pair. Without it the targeted listing would be unbound and D-04's "no
  `sync_interval` term" heal bound would regress to next-Hello + next-periodic
  listing; the simulator scenarios (`TestScenario_joinerWaitsForCompleteListing`,
  `TestScenario_emptyDisk`, `TestScenario_f14ReplacedNode`) caught exactly
  that when the binding was first applied without the echo.
- Regression tests: `TestSync_unboundListingDiscarded` (the review's
  unknown-identity proof, including the 24-hour no-expiry-check and the gate),
  `TestSync_forgedRunIDCannotReplaceListing` (the review's forged-`run_id`
  replacement proof, then a legitimate restart acceptance),
  `TestSync_emptyListingCreatesNoRow`, `TestSync_listedKeysExpire` (expiry,
  peer/complete preservation, republish recovery). Every listing fixture now
  sends its Hello first; assertion counts updated for the identity echo.
- The frozen stand/RPC tests were not modified and still pass.

### WR-01: The destination cap and queued-chunk cap multiply; the broadcast can be starved and a refused first-seen listing is lost

**Status:** fixed: requires human verification
**Files modified:** `internal/app/repl/core.go`,
`internal/app/repl/listing_internal_test.go`, `docs/04-MULTI-NODE.md`
**Commit:** 146d4ee
**Applied fix:**
- `maxListingChunks` is now `(maxListingDestinations + 1) × maxSyncChunks`
  (2 162 688): one full receiver-acceptable round per targeted destination
  plus the broadcast round. The count is virtual (shared build pointer), so
  this costs no per-chunk memory, and the destination cap — not the chunk cap
  — is the binding bound for any round a receiver accepts.
- The periodic broadcast is exempt from the destination cap and has its chunk
  work reserved, so a targeted flood can never refuse it; `drainListings`
  serves the broadcast FIFO ahead of targeted FIFOs (completion no longer
  assumes the head, via `removeListingDst`).
- A caps-refused targeted listing is re-armed: `oweListing` keeps
  `{dst, rotation}` (one per destination, bounded by `maxListingDestinations`)
  and `retryOwedListings` — run at the end of every `Advance`, after the drain
  — enqueues it as capacity frees. Peer eviction and a run change drop the
  owed entries (`dropOwedPeer`), and the rotation only advances on an admitted
  restart round.
- Tests: `TestListing_chunkCapCoversEveryDestination` (the cap identity),
  `TestListing_broadcastNeverStarved` (admission under a full destination table
  + drain priority), `TestListing_refusedFirstSeenIsRearmed` (full round
  delivered after re-arm).
- `docs/04-MULTI-NODE.md` records the new bound arithmetic and the drain-time
  worst case: a round is `chunks ÷ rate` (~1.4 s for 20 000 ordinary keys,
  40 s at one entry per chunk), the broadcast is admitted at its tick and
  drains first, and the last of a full destination table costs the sum of the
  rounds ahead of it (~45 s typical, ~22 min at the one-entry-per-chunk
  extreme); when a round exceeds `sync_interval` the effective broadcast
  period becomes the round time.

### WR-02: A restart drops the peer's retained listed keys before its replacement listing arrives

**Status:** fixed: requires human verification
**Files modified:** `internal/app/repl/core.go`, `internal/app/repl/restart_test.go`
**Commit:** dc82f8c
**Applied fix:** the restart branch no longer calls `dropListedPeer`.
`retainListedKeys` already replaces a peer's set wholesale on its first
complete listing, so the new run's set replaces the old one without a window;
the two-interval expiry bounds the wait when no replacement ever arrives.
Regression test `TestRestart_retainsListedKeysUntilReplacement` proves the old
keys and the generation survive the run change, completeness stays sticky, and
the replacement listing replaces the set.

### WR-03: The notification guard drops a superseded notification without emitting `ResultSuperseded`

**Status:** fixed: requires human verification
**Files modified:** `internal/app/repl/core.go`, `internal/app/repl/retransmit_test.go`
**Commit:** 39b58de
**Applied fix:** the guard is now a three-way compare: strictly older emits
`out.Superseded(key, rec.Version)` (the displaced identity's waiter resolves
immediately) and counts; equal counts only; greater proceeds. The
`NotificationsRegressed` doc and the `NotifyLocalWrite` doc state both
branches. `TestNotify_regressedIgnored` now asserts exactly one
`ResultSuperseded` for the older version (and no message), and
`TestNotify_equalTrackedIgnored` asserts an equal duplicate emits no result.

### WR-04: Snapshot load notifies every replayed record as an immediate unpaced broadcast

**Status:** fixed: requires human verification
**Files modified:** `internal/app/repl/core.go`,
`internal/app/repl/retransmit_test.go`, `internal/udprepl/repl.go`,
`internal/udprepl/repl_test.go`, `internal/grpcapi/gate.go`,
`internal/grpcapi/server.go`, `internal/grpcapi/server_test.go`,
`cmd/sup-kv/main.go`, `docs/04-MULTI-NODE.md`
**Commit:** a6768e3
**Applied fix:** the paced path was chosen (not the documented-burst
alternative). `NotifyLocalWrite` and the new `NotifyReplayedWrite` share
`trackLocalWrite` (guard + supersession + tracking); the replay variant calls
`enqueueRecordSend(now, pk, false)` instead of broadcasting, so replayed
records drain one datagram per pacing slot and the ladder still covers loss.
The adapter's notify channel carries a `replay` flag and the event loop calls
the replay method; `ServerOptions.ReplayNotify` (`LocalReplayNotify`) is the
new seam, `main` injects `adapter.NotifyReplayedWrite`, and a nil seam falls
back to the live notify with `track=false`. Tests:
`TestNotifyReplayed_pacedNotImmediate`, `TestReplayNotify_paced`,
`TestServer_snapshotLoadUsesReplaySeam`,
`TestServer_snapshotLoadReplayFallsBackToLive`. `docs/04-MULTI-NODE.md` now
states that a load replicates through the paced record-send queue (T-4-20 is
true of the code).

### WR-05: The new stream accumulator pre-sizes its chunk-index map from the attacker-supplied `chunk_count`

**Status:** fixed
**Files modified:** `internal/app/repl/core.go`,
`internal/app/repl/syncstream_internal_test.go` (new file)
**Commit:** d115651
**Applied fix:** the hint is
`min(msg.ChunkCount, syncStreamGotHint)` (`syncStreamGotHint = 8`); the map
grows as chunks actually arrive, which the entry caps already bound. The new
white-box probe `TestSync_streamMapHintIndependentOfClaim` compares the
retained heap of `maxSyncStreams` one-chunk rounds claiming 8 versus 65 536
chunks (min of three samples, 1 MiB allowance). Verified with teeth: with the
unclamped hint re-applied temporarily the probe fails at **19 515 328 bytes**
(matching the review's 19 488 824), and passes with the clamp.

### WR-06: The listing-build cache turns an out-of-range `sync_chunk_bytes` into a permanently retained allocation

**Status:** fixed: requires human verification
**Files modified:** `internal/app/repl/core.go`,
`internal/app/repl/config_internal_test.go` (new file),
`internal/udprepl/maxdatagram_test.go`, `cmd/sup-kv/main.go`,
`cmd/sup-kv/main_test.go`
**Commit:** 7ba917e
**Applied fix:** exported `repl.MaxSyncChunkBytes = 65507` (the IPv4 UDP
payload ceiling, pinned to `udprepl.MaxDatagramBytes` by a new cross-seam
test); `Config.normalized()` clamps a larger budget to it; `chunkSyncEntries`
sizes each chunk slice from `min(remaining entries, budget/entryOverheadBytes)`
instead of the byte budget alone; `cmd/sup-kv` rejects an explicit
`--sync-chunk-bytes` above the ceiling at startup (usage exit 2) with a message
naming the flag, the value and the ceiling, so the clamp is defense in depth
rather than a silent substitution. Tests: `TestConfig_syncChunkBytesClamped`
(six rows: above/exact/in-domain/zero/negative),
`TestChunkSyncEntries_capacityFollowsEntries`,
`TestMaxDatagramBytes_matchesCoreChunkBudget`, `TestMain_syncChunkBytesCeiling`.

### IN-01: `ServerOptions.SyncArrived`'s doc still describes the pre-D-01 semantics

**Status:** fixed
**Files modified:** `internal/grpcapi/gate.go`
**Commit:** 2ea996e
**Applied fix:** the field comment now states "reports whether a COMPLETE
listing round has been processed — every chunk of some peer's DataSync round
received, not merely the first arrival", and names the adapter's
`mirrorStats` completeness bit as the production source.

### IN-02: `docs/OPERATIONS.md`'s discard-class list omits five of the eleven published classes

**Status:** fixed
**Files modified:** `docs/OPERATIONS.md`
**Commit:** 93dec41
**Applied fix:** the metrics bullet enumerates all eleven labels
(`bad_crc`, `undecodable`, `bad_proto_version`, `own_broadcast`,
`duplicate_node_id`, `empty_key`, `key_too_long`, `tombstone_with_data`,
`bad_checksum`, `bad_version`, `bad_sync_header`) and states that the label
set is the complete catalog, published even at zero. `docs/SPEC-DEVIATIONS.md`
carries no per-label discard list (it references the classes generically), so
nothing there needed realigning.

### IN-03: Four new counters have no production surface, while the phase artifact claims nothing is dropped silently

**Status:** fixed (documented alternative — no metrics surface added, per the
orchestrator's binding scope; Phase 7's OBSV catalog owns it)
**Files modified:** `docs/04-MULTI-NODE.md`
**Commit:** a03e374
**Applied fix:** the "Protocol bounds" intro now says the published surfaces
are the discard-class and egress-drop families, and that
`Core.SyncStreamDropped`, `Core.ListingDropped` and
`Core.NotificationsRegressed` are **test-visible today, not yet published**,
with Phase 7 (OBSV-03/OBSV-05) owning their families and names. The counter
paragraph was updated to the current semantics (expiry counts in
`ListedKeysEvicted`, re-arm in `ListingDropped`, Superseded in
`NotificationsRegressed`). No metrics family or adapter mirror was added.

### IN-04: `Engine.Restore`'s mid-replay error path loses installed-but-unreturned records

**Status:** fixed: requires human verification
**Files modified:** `internal/app/persist/engine.go`
**Commit:** 4f8c256
**Applied fix:** the replay loop moved into `Engine.replaySnapshot`. A
mid-replay store error now aborts deterministically, returns the **partial**
`restored` list with the error, and marks the engine dirty whenever anything
was installed, so the installed records reach the next dump and the handler's
totality notify covers exactly the returned list. The `Restore` doc states the
contract. (The error remains unreachable with today's store — the fix is about
the return shape the next store error will meet.)

### IN-05: The stand harness's empty-gate derivation misses the space-separated and single-dash flag forms

**Status:** fixed
**Files modified:** `internal/stand/stand.go`,
`internal/stand/stand_form_test.go` (new file)
**Commit:** 6406b63
**Applied fix:** `emptyGateArg` now uses `hasFlag`/`flagValue` helpers that
strip one or two leading dashes and accept both `name=value` and the
space-separated `name value` pair (later flags win), so `--startup-gate 30s`,
`-startup-gate=45s` and `-startup-gate 45s` all derive the empty twin, and any
spelling of an explicit empty gate still wins. Table-driven
`TestEmptyGateArg_forms` covers eleven rows including the bare flag, both
dashes, both value forms and a value that merely looks like a flag.

### IN-06: `syncArrived` duplicates `listingComplete`

**Status:** fixed
**Files modified:** `internal/app/repl/core.go`
**Commit:** 0058a04
**Applied fix:** the `syncArrived` field and its assignment are gone;
`SyncArrived()` returns `c.listingComplete`, the single fact that backs both
reads. The frozen seam name and the adapter mirror are unchanged. No test
needed changing (both accessors stay exported and consistent).

## Skipped Issues

None — every in-scope finding was applied.

## Verification

- `make lint` (`go tool golangci-lint config verify` +
  `go tool golangci-lint run ./...`): clean, exit 0. Every added `//nolint`
  names its linter and carries a reason (the project's policy): `gosec` G115
  on a bounded key-count conversion and `revive` flag-parameter on the
  first-seen/restart branch selector in `internal/app/repl/core.go`,
  `paralleltest`/`revive` call-to-gc with reasons in
  `internal/app/repl/syncstream_internal_test.go`.
- `make test` (`go test -race -cover ./...`): green in the main checkout.
  One earlier full-suite run hit a pre-existing TOCTOU port-probe flake in
  `cmd/sup-kv#TestLog_jsonDefault` ("probe rx bind: address already in use");
  the package passes alone and the subsequent full runs are green.
- Per-fix verification: each fix re-read the modified section and compiled the
  affected packages; behavior is pinned by the regression tests named above.
- Frozen artifacts untouched by this fix pass:
  `internal/stand/stand_red_test.go`, `internal/grpcapi/red_gate_test.go`,
  `scripts/red-expectations.tsv`, `.golangci.yml`, `go.mod`, `go.sum`
  (`git diff bebfb4f..HEAD -- <those paths>` empty).

---

_Fixed: 2026-11-22T08:16:33Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
