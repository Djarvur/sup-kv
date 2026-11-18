---
phase: 03-persistence-dump-and-write-confirmation
verified: 2026-11-16T11:12:20Z
status: passed
score: 19/19 must-haves verified
covered_files: [".planning/phases/03-persistence-dump-and-write-confirmation/03-01-PLAN.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-01-SUMMARY.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-02-PLAN.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-02-SUMMARY.md", "docs/IDEA.md", "internal/app/persist/bench_test.go", "internal/app/persist/codec.go", "internal/app/persist/codec_test.go", "internal/app/persist/doc.go", "internal/app/persist/engine.go", "internal/app/persist/engine_internal_test.go", "internal/app/persist/engine_test.go", "internal/app/persist/errors.go", "internal/persistfile/dumpfile.go", "internal/persistfile/dumpfile_internal_test.go", "internal/persistfile/dumpfile_test.go", "internal/persistfile/fsync_darwin.go", "internal/persistfile/fsync_other.go", "internal/persistfile/snapshot.go", "internal/persistfile/snapshot_test.go", "internal/port/doc.go", "internal/port/dump.go"]
covered_digest: "v4:sha256:a26510f1393ba250976b4f691c170828dbb873f9086dc99f0b429a573bc6e935"
behavior_unverified: 0 # Every behavior-dependent truth has a passing named test — both scoped suites re-run fresh under -race -count=1 for this regeneration, plus a darwin cross-compile type-check of the platform-specific fsync path
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 19/19
  gaps_closed: [] # No gaps were open; this run regenerates a stale fingerprint, not gap closure
  gaps_remaining: []
  regressions: [] # The covered-source delta since 2026-11-14 is three phase-6 commits (06-01 fix 0f5e03d, 06-03 DumpStats 3ff81db, 06-09 F_FULLFSYNC 4f75a3b) — each verified below to preserve every must-have
---

# Phase 3: Persistence — Dump and Write Confirmation Verification Report

**Phase Goal:** State survives restarts and a client-confirmed write is always on disk — the write-confirmation contract that makes every later phase strictly additive
**Verified:** 2026-11-16T11:12:20Z
**Status:** passed
**Re-verification:** Yes — regeneration after fingerprint staleness (covered inputs touched by phase-6 commits)

## Regeneration Scope

The 2026-11-14 report passed 19/19 with no gaps. It went stale because phase-6 work (this session) directly touched phase-3 covered inputs. `git log --since=2026-11-14T19:26:57Z -- internal/ docs/IDEA.md` shows exactly three covered-source commits:

1. **0f5e03d (plan 06-01)** — the review-fix commit: conditional dirty in `ApplyRemote`, whole-snapshot Restore no-op, `confirmWaiters` invariant assertion, dead Open fallback removal (fixes this phase's WR-01/IN-01/IN-02, dispositions flipped in-fix in 03-REVIEW-DISPOSITION.md)
2. **3ff81db (plan 06-03)** — additive `DumpStats` observability atomics in `runDump`
3. **4f75a3b (plan 06-09)** — `File.Sync` replaced by an F_FULLFSYNC `fcntl` on darwin via `syncDurable` (`fsync_darwin.go`, `fsync_other.go`); resolves-with-code the open macOS F_FULLFSYNC planning decision from phase 3 (STATE.md Blockers/Concerns records it RESOLVED WITH CODE)

This run re-verifies every must-have against the current tree (HEAD 5e62bc7, branch gsd/v1.0-milestone), with focused adversarial scrutiny on that delta. Cross-read governance artifacts: 03-SECURITY.md (retroactive, 7/7 threats closed, threats_open: 0, T-3-05 accepted risk — read, not re-derived) and 03-REVIEW.md / 03-REVIEW-DISPOSITION.md (CR-01 fixed with regression tests; WR-01/IN-01/IN-02 now fixed by 06-01; IN-03/IN-04 info-level open by triage — advisory).

## Goal Achievement

### Observable Truths

Roadmap Success Criteria are rows 1-4; plan frontmatter truths that add detail follow. Plan truths that purely restate a roadmap SC are folded into its row. The truth set is unchanged from the plan contract (ROADMAP and both PLANs untouched since 2026-11-03).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | [SC1] The full map (tombstones included) round-trips through a human-readable JSON dump that is greppable with standard tools | ✓ VERIFIED | codec.go re-read (unchanged since 2026-11-03): `slices.Sorted(maps.Keys(...))`, `hex.EncodeToString` 32-char versions, base64 data, `SetEscapeHTML(false)` (codec.go:62-85). `TestEngine_endToEndRestart` + `TestCodec_roundTripProperty` in the fresh `-race -count=1` run: PASS |
| 2 | [SC2] A write returns OK only after a dump containing it completed (version watermark); on dump failure it errors while staying in memory; kill-and-restart recovers via main → .prev → empty start, discarding bad-checksum records | ✓ VERIFIED | Cover discipline intact at engine.go:292-312 (`cover := e.lastLocal` captured under mu in the same hold as the waiter list, before `Records()` at 301; `confirmWaiters` resolves only covered waiters). The 06-03 DumpStats timing wraps only Encode+Save and never touches cover/waiters. `TestEngine_dumpFailureFansOut`, `TestFileStore_fallbackChain` (5-row table), `TestEngine_restartDiscardsTornRecord` — all in the fresh PASS run |
| 3 | [SC3] A sustained ~1000 writes/s burst still sees every write confirmed within bounded latency — debounced dumps coalesce but never starve | ✓ VERIFIED | Deadline armed from clean-to-dirty `oldest`, never reset by later changes (engine.go:227-245). `TestEngine_burstBoundedConfirmation` (engine_test.go:432) asserts boundary-exact in virtual time: all 1000 returns at or before oldest+MaxWait, first save within the bound, at most 3 save attempts — PASS |
| 4 | [SC4] Named snapshots can be created and loaded; restore replays the snapshot as ordinary writes/deletes with fresh versions so it replicates and wins by normal LWW | ✓ VERIFIED | `TestEngine_snapshotCreateLoadRestore` (engine_test.go:928, re-read: byte-equal copy, live-beats-tombstone, tombstone-beats-live, no-op tombstone burns no version, absent keys untouched, fresh-version dominance, fresh-engine restore) PASS. The 06-01 whole-snapshot no-op (engine.go:581) skips replay ONLY when the store already equals the snapshot on every key/version/data/tombstone — `TestEngine_redundantDirty` pins the no-op saves nothing while ANY difference still replays in full (the pinned dominance test) |
| 5 | [VERS-04 persistence half] The loader computes the FULL watermark and Open seeds clock.NewClock with it | ✓ VERIFIED | `Watermark` keeps the (l, c) pair (codec.go:168-179); `Open` calls `clock.NewClock(now, initialL, initialC, driftWarn)` (engine.go:146); `TestWatermark_fullProperty` + `TestWatermark_equalCounterDomain` PASS |
| 6 | [01-T2] Dump is JSONL: header object then one record per line sorted by key, byte-stable, tombstones included, lossless round-trip | ✓ VERIFIED | Codec re-read confirms; `TestCodec_byteStability` + `TestCodec_roundTripProperty` PASS |
| 7 | [01-T3] Dump greppable: each key on its own line, base64 values, UUID-shaped 32-char hex versions | ✓ VERIFIED | Tracer file-shape assertions in `TestEngine_endToEndRestart`; `storage.RecordChecksum` verbatim at codec.go:142; zero crc32/hash imports anywhere in persist+persistfile (grep count 0) |
| 8 | [01-T4] Save atomic and crash-safe: temp in target dir, file fsync, rotate to .prev, rename, parent-dir fsync; Load serves .prev on missing/truncated/unparseable main; empty start only when neither reads | ✓ VERIFIED | `saveAtomically`/`syncedTemp`/`placeFile`/`syncDir` sequence re-read (dumpfile.go:97-176). The 06-09 change routes `writeSynced` and `syncDir` through `syncDurable` — `File.Sync` unchanged on every non-darwin platform (`fsync_other.go` pass-through), `fcntl(F_FULLFSYNC)` strictly stronger on darwin; the fsync-before-rename and dir-fsync-after-rename steps both still execute on every platform — a durability strengthening, PERS-02 semantics intact. `TestFileStore_atomicRotation` + `TestFileStore_fallbackChain` PASS (linux); `GOOS=darwin go build ./...` + `go vet ./internal/persistfile/` type-check the fcntl path |
| 9 | [01-T5] Bad-checksum record discarded and counted while siblings load; unknown format_version and structural faults make the generation unreadable | ✓ VERIFIED | `decodeRecord` per-record discard vs `errUnsupportedFormat` generation error (codec.go:104-151); `TestCodec_discardClasses` + `TestCodec_truncatedGenerationUnreadable` PASS |
| 10 | [01-T7] On dump failure every current waiter immediately receives ErrDumpFailed, record stays readable, no retry loop | ✓ VERIFIED | `failWaiters` fans out and re-dirties from a fresh epoch (engine.go:374-389); `TestEngine_dumpFailureFansOut` asserts exactly 2 saves across recovery — PASS |
| 11 | [01-T8] Run's goroutine exits on context cancel and every stranded waiter resolves with ErrClosed — no leaked goroutine, no blocked writer | ✓ VERIFIED | `sweepClosed` flips `closed` before taking waiters under one mutex hold, drains flushCh with buffered replies, closes `runDone` only after the sweep (engine.go:255-275); `TestEngine_runExitClosesWaiters` + `TestEngine_closeSurface` + both flush-race regressions (`TestEngine_flushRacesRunExit`, `TestEngine_flushRacesExitMidDump` — the fixed CR-01) PASS |
| 12 | [02-T1] Snapshot(name) forces unconfirmed state to disk via a completed dump, then byte-copies the finished main dump; dirty state forces a new dump first | ✓ VERIFIED | `Snapshot` = flush then `CreateSnapshot` (engine.go:544-550); `flushDump` dirty-check fast path (engine.go:212-220); bytes.Equal assertion at engine_test.go:972 + `TestEngine_snapshotFlushForcesDump` exact attempt count PASS |
| 13 | [02-T2] Restore(name) replays ONLY the snapshot's keys as ordinary writes/deletes with fresh versions, returns only after one final dump covering every replayed write | ✓ VERIFIED | `Restore` (engine.go:572-591): `ReadSnapshot` → `Decode` → sorted-key `replayRecord` via store.Set/store.Delete (no Merge) → `markDirty` + single `flush`. The 06-01 `restoreChanges` guard (engine.go:601-609) makes the equal-state case a no-op whose final flush replies nil with nothing replayed — vacuous confirmation (zero replayed writes, engine clean); any difference replays in full. Absent-keys-untouched asserted with version identity (engine_test.go:1047-1056) — PASS |
| 14 | [02-T3] Replayed records win by normal LWW; restoring twice converges | ✓ VERIFIED | Both tombstone directions + fresh-version dominance over max pre-restore version directly tested (PASS). Double-restore convergence remains entailed by the directly-tested per-restore dominance plus Phase 2's property-tested HLC monotonicity — same composition observation as both prior reports, not a gap |
| 15 | [02-T4] Names outside [A-Za-z0-9._-]{1,64}, or "." and "..", rejected with ErrInvalidSnapshotName before any filesystem touch | ✓ VERIFIED | `validateName` runs first in both operations (snapshot.go:38-42, 59-63; charset+length+dot-specials at 96-106); `TestFileStore_nameValidation` PASS |
| 16 | [02-T5] Snapshot overwrite is atomic in place (no .prev); a concurrent dump cannot tear it | ✓ VERIFIED | `saveAtomically(..., prevPath="")` for snapshots (snapshot.go:54; empty-prevPath skip at dumpfile.go:150-155); FileStore mutex serializes CreateSnapshot against Save's rotate window (dumpfile.go:35-43, 62-66); `TestFileStore_snapshotOverwriteAtomic` + `TestFileStore_snapshotConcurrentWithSave` + `TestFileStore_snapshotWaitsOutSaveWindow` PASS |
| 17 | [02-T6] Missing snapshot read wraps fs.ErrNotExist; snapshotting with no dump returns ErrNoDump | ✓ VERIFIED | `readMainDump` maps fs.ErrNotExist→ErrNoDump (snapshot.go:70-76); `TestFileStore_snapshotErrorClasses` + `TestEngine_restoreMissingSnapshot` PASS |
| 18 | [01-T1 tracer] Local write flows end-to-end with on-disk confirmation at Set-return and post-restart dominance | ✓ VERIFIED | `TestEngine_endToEndRestart` against the real filesystem PASS (fresh run); `TestEngine_openEmptyStart` PASS |
| 19 | [01-T6] Debounce 50 ms, max wait forces dump start no later than oldest+1 s, exact in virtual time | ✓ VERIFIED | `defaultDebounce`/`defaultMaxWait` + `Config.normalized` (engine.go:18-45); `TestEngine_confirmDebounceTiming` asserts the save lands at exactly start+50ms and Set returns at that same instant; burst boundary-exact test PASS; `dump_debounce 50ms` / `dump_max_wait 1s` at docs/IDEA.md:148 |

**Score:** 19/19 truths verified (0 present, behavior-unverified)

### Covered-Source Delta Scrutiny (Regeneration Focus)

Since the verified 2026-11-14 tree, three commits touched covered inputs. Each adversarially checked against PERS-04's write-confirmation discipline:

**1. 0f5e03d (plan 06-01) — engine.go 129 lines changed.** Four fixes, each read in the current tree and behaviorally pinned:

- **ApplyRemote conditional dirty (engine.go:501-519).** The dirty decision is coded to Merge's Conflict contract: a Conflict with `Winner == rec.Version` is a displacement (dirty), one with `Winner == stored` a rejection; no-Conflict is either an equal-version redelivery (probe saw the key live → skip) or a first install (probe saw the key absent → dirty). Case analysis confirms no genuine store change can skip its dump: a changed record is either a displacement (conflict, winner=incoming → dirty) or a first install under an absent key (getErr != nil → dirty); equal versions imply the identical record (UUIDv7 uniqueness), so the no-op skip is sound. The probe cannot see a stored tombstone, so an equal tombstone redelivery arms one redundant dump — the safe direction only. `TestEngine_redundantDirty` pins all of this, including case (b): a greater remote version arms a dump on a clean engine AND the same covering dump resolves a concurrent local write's waiter (one save covers both — PERS-04 unchanged, the starvation case directly falsified). ApplyRemote still registers no waiter and never raises `lastLocal` (only `await` does, under mu) — REPL-03's memory-apply contract intact, also pinned by `TestEngine_applyRemoteTriggersDump`.
- **Whole-snapshot Restore no-op (engine.go:581, 601-609).** `restoreChanges` compares only the snapshot's keys (version, data, tombstone flag) against current state; equal ⇒ no replay, no `markDirty`, clean-engine flush replies nil. Any difference replays in full with fresh versions. The D-09 dominance contract is preserved — it exists to make the restore beat a DIFFERENT pre-restore state, and an exactly-equal state has nothing to beat; the pinned `TestEngine_snapshotCreateLoadRestore` (all difference cases) passes unmodified, and `TestEngine_redundantDirty` pins the no-op case saving nothing.
- **confirmWaiters invariant assertion (engine.go:347-368).** The above-cover belt is kept, now with a never-requeue guard for an already-resolved waiter (double-close prevention). Cover discipline untouched — the guard sits on the unreachable keep branch only. `TestConfirmWaiters_aboveCoverInvariant` (new engine_internal_test.go, white-box — the state is unreachable through the public API) pins kept/nil-resolved/ErrDumpFailed/never-re-queued.
- **Open dead fallback removal (engine.go:132-141).** Decoding now rides Load's own oracle argument — the generation Load accepts is exactly the one whose Decode succeeded, so the post-Load guard was dead for every contract-honoring DumpStore. The persistfile adapter is such a store (Load applies the oracle per candidate, dumpfile.go:74-86). Fallback-chain and torn-record tests pass unmodified.

**2. 3ff81db (plan 06-03) — DumpStats observability.** Purely additive: three atomics recorded around the Encode+Save pair in `runDump` (engine.go:300-305) after the mutex is released; `recordDump` never touches cover, waiters, or the dirty epoch. The cover capture under one mutex hold before `Records()` is byte-identical to the verified discipline. `TestDumpStats_records` pins success and injected-failure accounting.

**3. 4f75a3b (plan 06-09) — darwin F_FULLFSYNC.** `writeSynced` and `syncDir` route through `syncDurable`: `fcntl(fd, F_FULLFSYNC, 0)` on darwin (strict superset of fsync per fcntl(2); Go's File.Sync binds libc_fsync only), plain `File.Sync` everywhere else — behavior byte-identical on all non-darwin platforms including this one. The canonical atomic-write sequence (temp in target dir, file sync, rotate, rename, dir sync) is intact; only the flush primitive strengthened on one platform. This resolves-with-code the open macOS F_FULLFSYNC planning decision from phase 3 (STATE.md records it RESOLVED WITH CODE, plan 06-09, commit 4f75a3b). Verified: linux race suites green, `GOOS=darwin GOARCH=arm64 go build ./...` and `go vet ./internal/persistfile/` green (the fcntl path never compiles on linux — the cross-compile is the honest local type-check; the 4-cell CI matrix covers the darwin cell). Kill-9 semantics unaffected on every platform: only power loss distinguishes the two flushes, and no CI test simulates power loss.

Also in the delta window: `bench_test.go` (06-08, same package — new file, no production interaction). The 03-REVIEW-DISPOSITION.md was updated in-fix by 0f5e03d: WR-01/IN-01/IN-02 flipped to fixed; CR-01 already fixed; IN-03/IN-04 remain info-level open by triage.

### Prohibitions

All eight plan prohibitions hold as negative checks in the current tree:

| Prohibition | Check | Status |
|-------------|-------|--------|
| No second checksum domain (D-07) | grep crc32/hash in persist+persistfile: 0 matches; `storage.RecordChecksum` verbatim at codec.go:142 | ✓ HELD |
| No engine mutex across Encode/Save I/O | runDump releases mu at engine.go:298 before `Encode`(301)/`Save`(303) | ✓ HELD |
| No two concurrent dumps | only Run's goroutine calls Save (runDump/flushDump); adapter mutex additionally serializes (dumpfile.go:62-66) | ✓ HELD |
| No waiter resolved without on-disk containment | cover+waiters captured under one mutex hold before `Records()` (engine.go:293-298) | ✓ HELD |
| No os/syscall/net/google.golang.org under internal/app/persist or internal/port (non-test) | grep: 0 matches; depguard machine-enforced — `make lint` exit 0. Note: fsync_darwin.go imports syscall but lives in internal/persistfile — the boundary's designated IO package | ✓ HELD |
| No unvalidated name reaching filepath.Join | `validateName` first in CreateSnapshot (snapshot.go:39) and ReadSnapshot (snapshot.go:60) | ✓ HELD |
| No full-map replace on restore (D-09) | replay is sorted-key store.Set/Delete only; absent keys asserted untouched with version identity; the 06-01 no-op guard skips only when nothing would change | ✓ HELD |
| No unconfirmed/torn state copied into a snapshot (D-10) | Snapshot = flush-then-copy; `TestEngine_snapshotFlushErrorAborts` proves a failed flush aborts with nothing copied | ✓ HELD |

### Required Artifacts

All artifacts exist, are substantive, and are wired. `verify.artifacts` (03-01: 6/10, 03-02: 1/3) flags only Go method-style "Missing export" names (`DumpStore.Save`, `Engine.Set`, `Engine.Snapshot`, ...) — false negatives of the generic grep against Go method-receiver syntax (`func (e *Engine) Set` vs `func Set`); every flagged method confirmed present by direct code reading this run. The `TestEngine_restoreReplaysSnapshot` contains-miss is the plan's own frontmatter-vs-task naming conflict, resolved by the executor to `TestEngine_snapshotCreateLoadRestore` (exists at engine_test.go:928).

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/port/dump.go` | DumpStore seam, 4 methods | ✓ VERIFIED | 40 lines; Save/Load/CreateSnapshot/ReadSnapshot declared with full contracts |
| `internal/app/persist/codec.go` | Encode/Decode/Parse/Watermark | ✓ VERIFIED | 179 lines; all exports + three per-record discard classes; RecordChecksum verbatim; unchanged since 2026-11-03 |
| `internal/app/persist/engine.go` | Write-confirmation engine | ✓ VERIFIED | 698 lines; all exports incl. Snapshot/Restore, ApplyRemote, Flush, DumpStats; the 06-01/06-03 deltas verified above |
| `internal/app/persist/errors.go` | ErrDumpFailed, ErrClosed | ✓ VERIFIED | Both sentinels with Phase 5 mapping docs |
| `internal/app/persist/doc.go` | Package contract | ✓ VERIFIED | Documents Run lifecycle, D-04 timing, snapshot contracts |
| `internal/persistfile/dumpfile.go` | Atomic save, .prev rotation, fallback Load | ✓ VERIFIED | 176 lines; canonical sequence routes flushes through syncDurable; WR-01 mutex; dirMode 0700 |
| `internal/persistfile/snapshot.go` | Snapshot adapter | ✓ VERIFIED | 106 lines; validateName before fs touch, ErrNoDump, ErrInvalidSnapshotName |
| `internal/app/persist/engine_test.go` | Tracer + synctest suite | ✓ VERIFIED | 23 test functions: 21 prior + TestEngine_redundantDirty (1489) + TestDumpStats_records (1586) |
| `internal/app/persist/engine_internal_test.go` | White-box WR-01 invariant regression | ✓ VERIFIED | New (06-01): TestConfirmWaiters_aboveCoverInvariant pins the above-cover belt |
| `internal/app/persist/codec_test.go` | Seeded property suite | ✓ VERIFIED | 6 test functions: round-trip/byte-stability/discard/watermark laws |
| `internal/persistfile/dumpfile_test.go` | Rotation, chain table, hygiene, smoke | ✓ VERIFIED | 5 test functions |
| `internal/persistfile/snapshot_test.go` | Name-validation security table | ✓ VERIFIED | TestFileStore_nameValidation + overwrite + error classes |
| `internal/persistfile/dumpfile_internal_test.go` | WR-01 regression (beyond plan) | ✓ VERIFIED | Internal-package rotate-window driver |
| `internal/persistfile/fsync_darwin.go` / `fsync_other.go` | Platform durability flush (06-09) | ✓ VERIFIED | Build-tagged pair; darwin fcntl F_FULLFSYNC / other File.Sync; both compile (darwin cross-build verified) |

### Key Link Verification

`verify.key-links`: 03-01 4/4 verified; 03-02 1/2 — the single miss is the pattern `codec\.Decode`, a same-package call made without qualifier (`Decode(data)` at engine.go:577 in Restore, and Open's `decode` closure at 133); the link intent (one format, one discard discipline for Restore and dump loads) holds by direct reading.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| persist/engine.go | port/dump.go | `port.DumpStore` | ✓ WIRED | Engine field `dumps port.DumpStore`; Save in runDump, Load with decode oracle in Open |
| persist/codec.go | storage/store.go | `storage.RecordChecksum` | ✓ WIRED | decodeRecord verifies via storage.RecordChecksum (codec.go:142); no crc32 import |
| persistfile/dumpfile.go | port/dump.go | `port.DumpStore` | ✓ WIRED | Compile-time assertion `var _ port.DumpStore = (*FileStore)(nil)` (dumpfile.go:46) |
| persist/engine.go | clock/clock.go | `clock.NewClock` | ✓ WIRED | Open seeds NewClock(now, initialL, initialC, driftWarn) (engine.go:146) |
| persist/engine.go | persistfile/snapshot.go | `CreateSnapshot\|ReadSnapshot` | ✓ WIRED | Snapshot → CreateSnapshot (engine.go:549); Restore → ReadSnapshot (engine.go:573) |
| persist/engine.go | persist/codec.go | `codec.Decode` | ✓ WIRED | Same-package unqualified `Decode(data)` at engine.go:134 (Open) and 577 (Restore); grep-pattern miss only |

### Data-Flow Trace (Level 4)

Library code — data-flow applies to the dump pipeline:

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| engine.runDump → codec.Encode → dumps.Save | `block` | `store.Records()` live snapshot | Yes — real store map, sorted, hex/base64 encoded, written through the real file adapter | ✓ FLOWING |
| Open → dumps.Load(decode) → Decode → Watermark → NewClock | `records`/`initialL,initialC` | On-disk dump bytes (real fs in tracer test) | Yes | ✓ FLOWING |
| Restore → ReadSnapshot → Decode → store.Set/Delete | `records` | Snapshot file bytes | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

Run on branch gsd/v1.0-milestone at HEAD 5e62bc7, cache bypassed (`-count=1`):

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full persist suite under race (30 tests: tracer restart, burst bound, failure fan-out, snapshot/restore incl. whole-restore no-op, redundant-dirty regression, confirmWaiters invariant, flush races, close surface, ApplyRemote, DumpStats, config-shrunk limits, Flush shutdown seam) | `go test -race -count=1 ./internal/app/persist/...` | ok, 2.535s | ✓ PASS |
| Full persistfile suite under race (9 tests: rotation, fallback chain, hygiene, snapshot smoke, concurrent save, name validation, overwrite, error classes, save-window) | `go test -race -count=1 ./internal/persistfile/...` | ok, 5.975s | ✓ PASS |
| Strict lint gate incl. config verify + depguard boundary | `make lint` | exit 0 | ✓ PASS |
| darwin fsync path type-check (never compiled on linux) | `GOOS=darwin GOARCH=arm64 go build ./... && go vet ./internal/persistfile/` | both OK | ✓ PASS |
| Summary commit hashes resolvable | `verify.commits` on all 11 hashes found in both SUMMARYs | all_valid: true, 11/11 | ✓ PASS |
| Port/app boundary (prohibition 5) | grep os/syscall/net/google.golang.org under internal/app/persist + internal/port (non-test) | 0 matches | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by the plans; the plans' probe clauses (PERS-01/02/03/05 concurrency, boundary, idempotency) are discharged as in-tree tests enumerated above (e.g. `TestFileStore_snapshotConcurrentWithSave`, boundary-exact synctest assertions, `TestEngine_applyRemoteTriggersDump`, `TestEngine_redundantDirty`). Not applicable as external probe scripts.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|---------------------|--------|----------|
| PERS-01 | 03-01 | Human-readable greppable JSON dump of full map | ✓ SATISFIED | Truths 1/6/7; codec + tracer + properties, fresh PASS run |
| PERS-02 | 03-01 | Atomic dumps, .prev fallback chain, bad-checksum discard | ✓ SATISFIED | Truths 2/8/9; rotation + chain table + torn-record test; durability flush strengthened on darwin (06-09), byte-identical elsewhere |
| PERS-03 | 03-01 | Debounced dumps with bounded max wait, no starvation | ✓ SATISFIED | Truths 3/19; burst test exact in virtual time |
| PERS-04 | 03-01 | Write OK only after covering dump; failure stays in memory | ✓ SATISFIED | Truths 2/10/11/18 + Delta Scrutiny (conditional dirty, whole-restore no-op, DumpStats and F_FULLFSYNC all preserve the discipline; the starvation case directly falsified by TestEngine_redundantDirty case b); ErrDumpFailed→Unavailable RPC mapping delivered in Phase 5 as documented |
| PERS-05 | 03-02 | Named snapshots; restore replays as ordinary writes, wins by LWW | ✓ SATISFIED | Truths 4/12-17; snapshot/restore suite incl. the no-op refinement |
| VERS-04 (persistence half) | 03-01 | Dump loader seeds full watermark via NewClock | ✓ SATISFIED | Truth 5; Watermark + Open wiring + dominance assertion |

Orphaned requirements: none — REQUIREMENTS.md maps exactly PERS-01..PERS-05 to Phase 3 (traceability rows all Complete; VERS-04 belongs to Phase 2 with its persistence half delivered by plan 02-04) and the plans claim exactly those IDs.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts: 10/10 (`check.decision-coverage-verify`, non-blocking gate). The phase-3-ratified assumptions carry forward unchanged: D-09 Restore semantics (replay-as-ordinary-writes, never full-map replace — the 06-01 no-op guard skips only when nothing would change, so the ratified semantics are untouched) and D-10 snapshot-overwrite assumptions (flush-then-copy, atomic in-place overwrite, no .prev for snapshots — intact in the current tree).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/port/dump.go | 15 | `//nolint:iface` with linter name + written reason | ℹ️ Info | AGENTS.md deviation 3 sanctioned; cross-package seam by design (D-01) |
| internal/persistfile/dumpfile.go | 76,170 | `//nolint:gosec` G304 ×2 with reasons | ℹ️ Info | Variable-path reads are the adapter's purpose; paths derive from config |
| internal/persistfile/snapshot.go | 65,71 | `//nolint:gosec` G304 ×2 with reasons | ℹ️ Info | Validated names / configured paths only |
| (prior pass) engine.go WR-01/IN-01/IN-02 | — | Previously-open advisories | ✓ FIXED | All three fixed by 06-01 this session (disposition flipped in-fix): above-cover guard now carries its invariant assertion + white-box regression; dead Open fallback removed; conditional dirty + whole-restore no-op — verified in Delta Scrutiny |
| 03-REVIEW-DISPOSITION.md | IN-03, IN-04 | Out-of-range checksum collapses generation; .snapshots dir created before ErrNoDump | ℹ️ Info | Open by triage, info-severity, carried forward — no goal impact |

No TBD/FIXME/XXX/TODO/HACK markers and no case-insensitive placeholder/not-yet-implemented strings in any phase file (test or non-test). No stubs, no empty implementations. One `t.Skip` — the documented root-euid environment guard in `TestFileStore_saveFailureHygiene` (dumpfile_test.go:163), not a disabled test. Nolint inventory: exactly 5, each naming its linter with a written reason; the new fsync files added none.

### Advisory (New Scope, Unevidenced)

None — no new-scope findings arose on this pass; the only open review items (IN-03/IN-04) are carried-forward info-level dispositions recorded above.

### Human Verification Required

None — infrastructure/foundation phase (library-level persistence; no user-facing UI). Every behavior-dependent truth has a passing named behavioral test; the full scoped suites were re-run fresh under `-race -count=1` for this regeneration and passed, including the 06-01 regression tests and the darwin cross-compile type-check of the 06-09 platform flush. No `<verify><human-check>` blocks are deferred in either plan.

### Gaps Summary

None. All 19 must-have truths re-verified against the current tree with fresh behavioral evidence; all artifacts present, substantive, wired, data-flowing; all six key links wired; all five PERS requirements plus the VERS-04 persistence half satisfied with no orphans; all eight prohibitions hold; lint gate (with config verify and the depguard boundary) green; darwin cross-compile green; decision coverage 10/10; security register closed (7/7, threats_open: 0). The three covered-source commits since the 2026-11-14 report each preserve the phase-3 contract: 06-01's fixes strengthen the discipline (conditional dirty falsified for starvation, no-op restore pinned, invariant asserted), 06-03's DumpStats is purely additive, and 06-09's F_FULLFSYNC strictly strengthens durability on darwin while leaving every other platform byte-identical — it also resolves-with-code the open macOS F_FULLFSYNC planning decision from phase 3. The remaining review findings (IN-03/IN-04) are info-level, open by triage, with no goal impact.

---

_Verified: 2026-11-16T11:12:20Z_
_Verifier: Claude (gsd-verifier)_
