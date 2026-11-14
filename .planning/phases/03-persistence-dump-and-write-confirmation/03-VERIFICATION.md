---
phase: 03-persistence-dump-and-write-confirmation
verified: 2026-11-14T19:26:57Z
status: passed
score: 19/19 must-haves verified
covered_files: [".planning/phases/03-persistence-dump-and-write-confirmation/03-01-PLAN.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-01-SUMMARY.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-02-PLAN.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-02-SUMMARY.md", "docs/IDEA.md", "internal/app/persist/codec.go", "internal/app/persist/codec_test.go", "internal/app/persist/doc.go", "internal/app/persist/engine.go", "internal/app/persist/engine_test.go", "internal/app/persist/errors.go", "internal/persistfile/dumpfile.go", "internal/persistfile/dumpfile_internal_test.go", "internal/persistfile/dumpfile_test.go", "internal/persistfile/snapshot.go", "internal/persistfile/snapshot_test.go", "internal/port/doc.go", "internal/port/dump.go"]
covered_digest: "v4:sha256:beb1ffc7f0d49bd48d77c0d5bbaea549148fcb1a472cec58be6bb6e9010c8ad8"
behavior_unverified: 0 # Every behavior-dependent truth has a passing named test — both scoped suites re-run fresh under -race -count=1 for this regeneration
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 19/19
  gaps_closed: [] # No gaps were open; this run regenerates a stale fingerprint, not gap closure
  gaps_remaining: []
  regressions: [] # The only covered-source delta since 2026-11-08 is the additive Engine.Flush export (commit 86d33a6) — verified below
---

# Phase 3: Persistence — Dump and Write Confirmation Verification Report

**Phase Goal:** State survives restarts and a client-confirmed write is always on disk — the write-confirmation contract that makes every later phase strictly additive
**Verified:** 2026-11-14T19:26:57Z
**Status:** passed
**Re-verification:** Yes — regeneration after fingerprint staleness (covered inputs touched by later phases' commits)

## Regeneration Scope

The 2026-11-08 report passed 19/19 with no gaps. It went stale because a later commit touched phase-3-covered inputs. `git log d816234..HEAD -- internal/` shows exactly one such commit: 86d33a6 (plan 05-06) adding the exported `Engine.Flush` — 15 lines in `engine.go` plus 97 lines of tests. This run re-verifies every must-have against the current tree (HEAD c525c9c, branch gsd/phase-4-replication-and-deterministic-simulator), with focused adversarial scrutiny on that delta. It also cross-reads the two same-day governance artifacts: 03-SECURITY.md (retroactive, 7/7 threats closed, threats_open: 0, T-3-05 accepted risk — read, not re-derived) and 03-REVIEW.md / 03-REVIEW-DISPOSITION.md (0 critical open — CR-01 fixed with regression tests; 1 warning + 2 info open, advisory).

## Goal Achievement

### Observable Truths

Roadmap Success Criteria are rows 1-4; plan frontmatter truths that add detail follow. Plan truths that purely restate a roadmap SC are folded into its row. The truth set is unchanged from the plan contract (ROADMAP and both PLANs untouched since 2026-11-03).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | [SC1] The full map (tombstones included) round-trips through a human-readable JSON dump that is greppable with standard tools | ✓ VERIFIED | codec.go re-read: `slices.Sorted(maps.Keys(...))`, `hex.EncodeToString` 32-char versions, base64 data, `SetEscapeHTML(false)` (codec.go:62-85). `TestEngine_endToEndRestart` + `TestCodec_roundTripProperty` in the fresh `-race -count=1` run: PASS |
| 2 | [SC2] A write returns OK only after a dump containing it completed (version watermark); on dump failure it errors while staying in memory; kill-and-restart recovers via main → .prev → empty start, discarding bad-checksum records | ✓ VERIFIED | Cover discipline intact at engine.go:266-284 (`cover := e.lastLocal` captured under mu in the same hold as the waiter list, before `Records()`; `confirmWaiters` resolves only covered waiters). `TestEngine_dumpFailureFansOut`, `TestFileStore_fallbackChain` (5-row table), `TestEngine_restartDiscardsTornRecord` — all in the fresh PASS run |
| 3 | [SC3] A sustained ~1000 writes/s burst still sees every write confirmed within bounded latency — debounced dumps coalesce but never starve | ✓ VERIFIED | Deadline armed from clean-to-dirty `oldest`, never reset by later changes (engine.go:202-220). `TestEngine_burstBoundedConfirmation` (engine_test.go:432-474) asserts boundary-exact in virtual time: all 1000 returns at or before oldest+MaxWait, first save within the bound, at most 3 save attempts |
| 4 | [SC4] Named snapshots can be created and loaded; restore replays the snapshot as ordinary writes/deletes with fresh versions so it replicates and wins by normal LWW | ✓ VERIFIED | `TestEngine_snapshotCreateLoadRestore` (engine_test.go:928-1103) PASS: byte-equal copy, live-beats-tombstone (resurrect), tombstone-beats-live (re-tomb), no-op tombstone burns no version, absent keys untouched, fresh-version dominance over max pre-restore version, fresh-engine restore from carried snapshot dir |
| 5 | [VERS-04 persistence half] The loader computes the FULL watermark and Open seeds clock.NewClock with it | ✓ VERIFIED | `Watermark` keeps the (l, c) pair — max l plus max counter at that l (codec.go:168-179); `Open` calls `clock.NewClock(now, initialL, initialC, driftWarn)` (engine.go:120-121); `TestWatermark_fullProperty` + `TestWatermark_equalCounterDomain` PASS |
| 6 | [01-T2] Dump is JSONL: header object then one record per line sorted by key, byte-stable, tombstones included, lossless round-trip | ✓ VERIFIED | Codec re-read confirms; `TestCodec_byteStability` + `TestCodec_roundTripProperty` PASS |
| 7 | [01-T3] Dump greppable: each key on its own line, base64 values, UUID-shaped 32-char hex versions | ✓ VERIFIED | Tracer file-shape assertions in `TestEngine_endToEndRestart`; `storage.RecordChecksum` verbatim at codec.go:142; zero crc32/hash imports anywhere in persist (grep count 0) |
| 8 | [01-T4] Save atomic and crash-safe: temp in target dir, file fsync, rotate to .prev, rename, parent-dir fsync; Load serves .prev on missing/truncated/unparseable main; empty start only when neither reads | ✓ VERIFIED | `saveAtomically`/`syncedTemp`/`placeFile`/`syncDir` sequence re-read (dumpfile.go:97-174); `TestFileStore_atomicRotation` + `TestFileStore_fallbackChain` PASS |
| 9 | [01-T5] Bad-checksum record discarded and counted while siblings load; unknown format_version and structural faults make the generation unreadable | ✓ VERIFIED | `decodeRecord` per-record discard vs `errUnsupportedFormat` generation error (codec.go:104-151); `TestCodec_discardClasses` + `TestCodec_truncatedGenerationUnreadable` PASS |
| 10 | [01-T7] On dump failure every current waiter immediately receives ErrDumpFailed, record stays readable, no retry loop | ✓ VERIFIED | `failWaiters` fans out and re-dirties from a fresh epoch (engine.go:311-326); `TestEngine_dumpFailureFansOut` asserts exactly 2 saves across recovery — PASS |
| 11 | [01-T8] Run's goroutine exits on context cancel and every stranded waiter resolves with ErrClosed — no leaked goroutine, no blocked writer | ✓ VERIFIED | `sweepClosed` flips `closed` before taking waiters under one mutex hold, then drains flushCh with buffered replies (engine.go:230-250); `TestEngine_runExitClosesWaiters` + `TestEngine_closeSurface` + both flush-race regressions (`TestEngine_flushRacesRunExit`, `TestEngine_flushRacesExitMidDump` — the fixed CR-01) PASS |
| 12 | [02-T1] Snapshot(name) forces unconfirmed state to disk via a completed dump, then byte-copies the finished main dump; dirty state forces a new dump first | ✓ VERIFIED | `Snapshot` = flush then `CreateSnapshot` (engine.go:460-466); `flushDump` dirty-check fast path (engine.go:187-195); bytes.Equal assertion at engine_test.go:976 + `TestEngine_snapshotFlushForcesDump` exact attempt count PASS |
| 13 | [02-T2] Restore(name) replays ONLY the snapshot's keys as ordinary writes/deletes with fresh versions, returns only after one final dump covering every replayed write | ✓ VERIFIED | `Restore` (engine.go:482-499): `ReadSnapshot` → `Decode` → sorted-key `replayRecord` via store.Set/store.Delete (no Merge) → `markDirty` + single `flush`. Absent-keys-untouched asserted with version identity (engine_test.go:1047-1056) |
| 14 | [02-T3] Replayed records win by normal LWW; restoring twice converges | ✓ VERIFIED | Both tombstone directions + fresh-version dominance over max pre-restore version directly tested (PASS). Double-restore convergence remains entailed by the directly-tested per-restore dominance plus Phase 2's property-tested HLC monotonicity (stamps are drawn after prior writes by construction) — same composition observation as both prior reports, not a gap |
| 15 | [02-T4] Names outside [A-Za-z0-9._-]{1,64}, or "." and "..", rejected with ErrInvalidSnapshotName before any filesystem touch | ✓ VERIFIED | `validateName` runs first in both operations (snapshot.go:38-42, 59-63; charset+length+dot-specials at 96-106); `TestFileStore_nameValidation` PASS |
| 16 | [02-T5] Snapshot overwrite is atomic in place (no .prev); a concurrent dump cannot tear it | ✓ VERIFIED | `saveAtomically(..., prevPath="")` for snapshots (snapshot.go:54; empty-prevPath skip at dumpfile.go:148-155); FileStore mutex serializes CreateSnapshot against Save's rotate window (dumpfile.go:35-43, 62-66); `TestFileStore_snapshotOverwriteAtomic` + `TestFileStore_snapshotConcurrentWithSave` + `TestFileStore_snapshotWaitsOutSaveWindow` PASS |
| 17 | [02-T6] Missing snapshot read wraps fs.ErrNotExist; snapshotting with no dump returns ErrNoDump | ✓ VERIFIED | `readMainDump` maps fs.ErrNotExist→ErrNoDump (snapshot.go:70-76); `TestFileStore_snapshotErrorClasses` + `TestEngine_restoreMissingSnapshot` PASS |
| 18 | [01-T1 tracer] Local write flows end-to-end with on-disk confirmation at Set-return and post-restart dominance | ✓ VERIFIED | `TestEngine_endToEndRestart` against the real filesystem PASS (fresh run); `TestEngine_openEmptyStart` PASS |
| 19 | [01-T6] Debounce 50 ms, max wait forces dump start no later than oldest+1 s, exact in virtual time | ✓ VERIFIED | `defaultDebounce`/`defaultMaxWait` + `Config.normalized` (engine.go:18-43); `TestEngine_confirmDebounceTiming` asserts the save lands at exactly start+50ms and Set returns at that same instant; burst boundary-exact test PASS; `dump_max_wait 1s` at docs/IDEA.md |

**Score:** 19/19 truths verified (0 present, behavior-unverified)

### Covered-Source Delta Scrutiny (Regeneration Focus)

Since the verified 2026-11-08 tree, exactly one commit touched covered inputs: 86d33a6 exporting `Engine.Flush` (engine.go:522-535). Adversarially checked against PERS-04's write-confirmation discipline:

1. **Purely additive.** `git diff d816234..HEAD -- internal/app/persist/engine.go` is exactly +15 lines: one doc comment plus a one-line method delegating to the pre-existing unexported `flush` (the seam Snapshot and Restore already drive). Not one line of the confirmation machinery (`Run`, `runDump`, `confirmWaiters`, `failWaiters`, `await`, `sweepClosed`, `flush`, `rearm`) changed.
2. **Flush cannot over-confirm.** It routes through `flushDump`, whose dirty fast-path either replies nil (clean — nothing unconfirmed on disk) or runs the ordinary `runDump` with the same cover capture under the mutex. No waiter is resolved outside the cover discipline.
3. **The shutdown contract is behaviorally pinned** by three new tests in the fresh PASS run: `TestFlush_remoteDirtiness` (ApplyRemote-merged record persisted by exactly one Flush-forced save — the dump carries the merged record), `TestFlush_closed` (ErrClosed after Run's exit, no hang), `TestFlush_clean` (clean engine returns nil with no redundant save — the D-10 decision extended to shutdown).
4. **ApplyRemote re-checked in the current tree** (engine.go:424-435): `store.Merge` → `markDirty` → non-blocking `changed` nudge. It never touches `lastLocal` (only `await` raises it, under mu, engine.go:394-396) and registers no waiter, so a remote merge can never lift the cover or dump-confirm itself — REPL-03's documented memory-apply contract, behaviorally pinned by `TestEngine_applyRemoteTriggersDump` (returns before the dump; the debounce dump carries the record). The 4-06 no-waiter synctest Sleep paths live in tests only.

Also in the delta window: `internal/app/storage` gained an OnChange tap (commit c2746cb, phase 5) — not a phase-3-covered input, no persist-contract interaction (the engine consumes `store.Records()`/Set/Delete/Merge unchanged).

### Prohibitions

All eight plan prohibitions hold as negative checks in the current tree:

| Prohibition | Check | Status |
|-------------|-------|--------|
| No second checksum domain (D-07) | grep crc32/hash in persist+persistfile: 0 matches; `storage.RecordChecksum` verbatim at codec.go:142 | ✓ HELD |
| No engine mutex across Encode/Save I/O | runDump releases mu at engine.go:272 before `Encode`(274)/`Save`(276) | ✓ HELD |
| No two concurrent dumps | only Run's goroutine calls Save (runDump/flushDump); adapter mutex additionally serializes (dumpfile.go:62-66) | ✓ HELD |
| No waiter resolved without on-disk containment | cover+waiters captured under one mutex hold before `Records()` (engine.go:267-272) | ✓ HELD |
| No os/syscall/net/google.golang.org under internal/app/persist or internal/port (non-test) | grep: 0 matches; depguard machine-enforced — `make lint` exit 0 | ✓ HELD |
| No unvalidated name reaching filepath.Join | `validateName` first in CreateSnapshot (snapshot.go:39) and ReadSnapshot (snapshot.go:60) | ✓ HELD |
| No full-map replace on restore (D-09) | replay is sorted-key store.Set/Delete only; absent keys asserted untouched with version identity | ✓ HELD |
| No unconfirmed/torn state copied into a snapshot (D-10) | Snapshot = flush-then-copy; `TestEngine_snapshotFlushErrorAborts` proves a failed flush aborts with nothing copied | ✓ HELD |

### Required Artifacts

All artifacts exist, are substantive, and are wired. `verify.artifacts` (03-01: 6/10, 03-02: 1/3) flags only Go method-style "Missing export" names (`DumpStore.Save`, `Engine.Set`, `Engine.Snapshot`, ...) — false negatives of the generic grep against Go method-receiver syntax (`func (e *Engine) Set` vs `func Set`); every flagged method confirmed present by direct code reading this run. The `TestEngine_restoreReplaysSnapshot` contains-miss is the plan's own frontmatter-vs-task naming conflict, resolved by the executor to `TestEngine_snapshotCreateLoadRestore` (exists at engine_test.go:928).

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/port/dump.go` | DumpStore seam, 4 methods | ✓ VERIFIED | 41 lines; Save/Load/CreateSnapshot/ReadSnapshot declared with full contracts |
| `internal/app/persist/codec.go` | Encode/Decode/Parse/Watermark | ✓ VERIFIED | 179 lines; all exports + three per-record discard classes; RecordChecksum verbatim |
| `internal/app/persist/engine.go` | Write-confirmation engine | ✓ VERIFIED | 588 lines (573 + the additive Flush export); all exports incl. Snapshot/Restore, ApplyRemote, Flush |
| `internal/app/persist/errors.go` | ErrDumpFailed, ErrClosed | ✓ VERIFIED | Both sentinels with Phase 5 mapping docs |
| `internal/app/persist/doc.go` | Package contract | ✓ VERIFIED | Documents Run lifecycle, D-04 timing, snapshot contracts |
| `internal/persistfile/dumpfile.go` | Atomic save, .prev rotation, fallback Load | ✓ VERIFIED | 174 lines; canonical sequence + WR-01 mutex; dirMode 0700 |
| `internal/persistfile/snapshot.go` | Snapshot adapter | ✓ VERIFIED | 106 lines; validateName before fs touch, ErrNoDump, ErrInvalidSnapshotName |
| `internal/app/persist/engine_test.go` | Tracer + synctest suite | ✓ VERIFIED | 21 test functions: 18 prior + TestFlush_remoteDirtiness/closed/clean (707/752/771) |
| `internal/app/persist/codec_test.go` | Seeded property suite | ✓ VERIFIED | 6 test functions: round-trip/byte-stability/discard/watermark laws |
| `internal/persistfile/dumpfile_test.go` | Rotation, chain table, hygiene, smoke | ✓ VERIFIED | 5 test functions |
| `internal/persistfile/snapshot_test.go` | Name-validation security table | ✓ VERIFIED | TestFileStore_nameValidation + overwrite + error classes |
| `internal/persistfile/dumpfile_internal_test.go` | WR-01 regression (beyond plan) | ✓ VERIFIED | Internal-package rotate-window driver |

### Key Link Verification

`verify.key-links`: 03-01 4/4 verified; 03-02 1/2 — the single miss is the pattern `codec\.Decode`, a same-package call made without qualifier (`Decode(data)` at engine.go:487); the link intent (one format, one discard discipline for Restore and dump loads) holds by direct reading.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| persist/engine.go | port/dump.go | `port.DumpStore` | ✓ WIRED | Engine field `dumps port.DumpStore`; Save in runDump, Load with Parse in Open |
| persist/codec.go | storage/store.go | `storage.RecordChecksum` | ✓ WIRED | decodeRecord verifies via storage.RecordChecksum (codec.go:142); no crc32 import |
| persistfile/dumpfile.go | port/dump.go | `port.DumpStore` | ✓ WIRED | Compile-time assertion `var _ port.DumpStore = (*FileStore)(nil)` (dumpfile.go:46) |
| persist/engine.go | clock/clock.go | `clock.NewClock` | ✓ WIRED | Open seeds NewClock(now, initialL, initialC, driftWarn) (engine.go:121) |
| persist/engine.go | persistfile/snapshot.go | `CreateSnapshot\|ReadSnapshot` | ✓ WIRED | Snapshot → CreateSnapshot (engine.go:465); Restore → ReadSnapshot (engine.go:483) |
| persist/engine.go | persist/codec.go | `codec.Decode` | ✓ WIRED | Same-package unqualified `Decode(data)` at engine.go:112 (Open) and 487 (Restore); grep-pattern miss only |

### Data-Flow Trace (Level 4)

Library code — data-flow applies to the dump pipeline:

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| engine.runDump → codec.Encode → dumps.Save | `block` | `store.Records()` live snapshot | Yes — real store map, sorted, hex/base64 encoded, written through the real file adapter | ✓ FLOWING |
| Open → dumps.Load(Parse) → Decode → Watermark → NewClock | `records`/`initialL,initialC` | On-disk dump bytes (real fs in tracer test) | Yes | ✓ FLOWING |
| Restore → ReadSnapshot → Decode → store.Set/Delete | `records` | Snapshot file bytes | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

Run on branch gsd/phase-4-replication-and-deterministic-simulator at HEAD c525c9c, cache bypassed (`-count=1`):

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full persist suite under race (21 tests: tracer restart, burst bound, failure fan-out, snapshot/restore, flush races, close surface, ApplyRemote, config-shrunk limits, Flush shutdown seam) | `go test -race -count=1 ./internal/app/persist/...` | ok, 2.820s | ✓ PASS |
| Full persistfile suite under race (9 tests: rotation, fallback chain, hygiene, snapshot smoke, concurrent save, name validation, overwrite, error classes, save-window) | `go test -race -count=1 ./internal/persistfile/...` | ok, 5.714s | ✓ PASS |
| Strict lint gate incl. config verify + depguard boundary | `make lint` | exit 0 | ✓ PASS |
| Summary commit hashes resolvable | `verify.commits` on all 11 hashes found in both SUMMARYs | all_valid: true, 11/11 | ✓ PASS |
| Port/app boundary (prohibition 5) | grep os/syscall/net/google.golang.org under internal/app/persist + internal/port (non-test) | 0 matches | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by the plans; the plans' probe clauses (PERS-01/02/03/05 concurrency, boundary, idempotency) are discharged as in-tree tests enumerated above (e.g. `TestFileStore_snapshotConcurrentWithSave`, boundary-exact synctest assertions, `TestEngine_applyRemoteTriggersDump`, `TestFlush_remoteDirtiness`). Not applicable as external probe scripts.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|---------------------|--------|----------|
| PERS-01 | 03-01 | Human-readable greppable JSON dump of full map | ✓ SATISFIED | Truths 1/6/7; codec + tracer + properties, fresh PASS run |
| PERS-02 | 03-01 | Atomic dumps, .prev fallback chain, bad-checksum discard | ✓ SATISFIED | Truths 2/8/9; rotation + chain table + torn-record test |
| PERS-03 | 03-01 | Debounced dumps with bounded max wait, no starvation | ✓ SATISFIED | Truths 3/19; burst test exact in virtual time |
| PERS-04 | 03-01 | Write OK only after covering dump; failure stays in memory | ✓ SATISFIED | Truths 2/10/11/18 + Delta Scrutiny (ApplyRemote and Flush both preserve the discipline); ErrDumpFailed→Unavailable RPC mapping delivered in Phase 5 as documented |
| PERS-05 | 03-02 | Named snapshots; restore replays as ordinary writes, wins by LWW | ✓ SATISFIED | Truths 4/12-17; snapshot/restore suite |
| VERS-04 (persistence half) | 03-01 | Dump loader seeds full watermark via NewClock | ✓ SATISFIED | Truth 5; Watermark + Open wiring + dominance assertion |

Orphaned requirements: none — REQUIREMENTS.md maps exactly PERS-01..PERS-05 to Phase 3 (traceability rows all Complete; VERS-04 belongs to Phase 2 with its persistence half delivered by plan 02-04) and the plans claim exactly those IDs.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts: 10/10 (`check.decision-coverage-verify`, non-blocking gate).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/port/dump.go | 15 | `//nolint:iface` with linter name + written reason | ℹ️ Info | AGENTS.md deviation 3 sanctioned; cross-package seam by design (D-01) |
| internal/persistfile/dumpfile.go | 76,168 | `//nolint:gosec` G304 ×2 with reasons | ℹ️ Info | Variable-path reads are the adapter's purpose; paths derive from config |
| internal/persistfile/snapshot.go | 65,71 | `//nolint:gosec` G304 ×2 with reasons | ℹ️ Info | Validated names / configured paths only |
| internal/app/persist/engine.go | 293-296 | WR-01: above-cover keep branch in confirmWaiters provably unreachable (await raises lastLocal in the same mutex hold that appends the waiter, so every captured waiter is ≤ cover) | ⚠️ Warning | Defensive dead code in the safe direction; open as advisory in 03-REVIEW-DISPOSITION.md — no goal impact |
| internal/app/persist/engine.go | 110-116 | IN-01: Open's Decode-error fallback dead in the shipped stack (Parse already gated the generation) | ℹ️ Info | Belt-over-chain; open as advisory — no goal impact |
| internal/app/persist/engine.go | 429 | IN-02: ApplyRemote dirties on no-op (lost) merges — one redundant dump | ℹ️ Info | Extra I/O, no correctness effect; open as advisory |

No TBD/FIXME/XXX/TODO/HACK markers and no case-insensitive placeholder/not-yet-implemented strings in any phase file (test or non-test). No stubs, no empty implementations. One `t.Skip` in the suite — the documented root-euid environment guard in `TestFileStore_saveFailureHygiene` (dumpfile_test.go:163), not a disabled test. Nolint inventory: exactly 5, each naming its linter with a written reason; `.golangci.yml` Origin header intact (upstream strict 2.15.0).

### Human Verification Required

None — infrastructure/foundation phase (library-level persistence; no user-facing UI). Every behavior-dependent truth has a passing named behavioral test; the full scoped suite was re-run fresh under `-race -count=1` for this regeneration and passed, including the Flush shutdown-seam delta tests. No `<verify><human-check>` blocks are deferred in either plan.

### Gaps Summary

None. All 19 must-have truths re-verified against the current tree with fresh behavioral evidence; all artifacts present, substantive, wired, data-flowing; all six key links wired; all five PERS requirements plus the VERS-04 persistence half satisfied with no orphans; all eight prohibitions hold; lint gate (with config verify and the depguard boundary) green; decision coverage 10/10; security register closed (7/7, threats_open: 0). The only covered-source delta since the 2026-11-08 report — the exported `Engine.Flush` (commit 86d33a6) — is purely additive, routes through the pre-existing flush seam, cannot resolve any waiter outside the cover discipline, and is behaviorally pinned by three new passing tests. The open review findings (WR-01 defensive branch, IN-01 dead fallback, IN-02 no-op dirty) are advisory with no goal impact.

---

_Verified: 2026-11-14T19:26:57Z_
_Verifier: Claude (gsd-verifier)_
