---
phase: 03-persistence-dump-and-write-confirmation
verified: 2026-11-08T18:11:18Z
status: passed
score: 19/19 must-haves verified
covered_files: [".planning/phases/03-persistence-dump-and-write-confirmation/03-01-PLAN.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-01-SUMMARY.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-02-PLAN.md", ".planning/phases/03-persistence-dump-and-write-confirmation/03-02-SUMMARY.md", "docs/IDEA.md", "internal/app/persist/codec.go", "internal/app/persist/codec_test.go", "internal/app/persist/doc.go", "internal/app/persist/engine.go", "internal/app/persist/engine_test.go", "internal/app/persist/errors.go", "internal/persistfile/dumpfile.go", "internal/persistfile/dumpfile_internal_test.go", "internal/persistfile/dumpfile_test.go", "internal/persistfile/snapshot.go", "internal/persistfile/snapshot_test.go", "internal/port/doc.go", "internal/port/dump.go"]
covered_digest: "v4:sha256:a4f6b377d751e05254ba3a5cc6bd2c7edd1236f5c9123f3eb7cac831d1450dae"
behavior_unverified: 0 # Every behavior-dependent truth has a passing named test — the full scoped -race suite was re-run fresh (cache bypassed) for this regeneration
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 19/19
  gaps_closed: [] # No gaps were open; this run regenerates a stale fingerprint, not gap closure
  gaps_remaining: []
  regressions: [] # The phase-4 ApplyRemote seam added to covered source is purely additive — verified below
---

# Phase 3: Persistence — Dump and Write Confirmation Verification Report

**Phase Goal:** State survives restarts and a client-confirmed write is always on disk — the write-confirmation contract that makes every later phase strictly additive
**Verified:** 2026-11-08T18:11:18Z
**Status:** passed
**Re-verification:** Yes — regeneration after fingerprint staleness (covered inputs touched by later phases' commits)

## Regeneration Scope

The 2026-11-03 report passed 19/19 with no gaps. It went stale because later commits touched phase-3-covered inputs: `internal/app/persist/engine.go` gained the phase-4 `Engine.ApplyRemote` seam (plan 04-06, commits f839c0d + 4042699), and `go.mod`/`Makefile` moved with earlier phase-1/4 commits. This run re-verifies every must-have against the current tree (HEAD d816234, branch gsd/phase-4-replication-and-deterministic-simulator), with focused adversarial scrutiny on the delta.

## Goal Achievement

### Observable Truths

Roadmap Success Criteria are rows 1-4; plan frontmatter truths that add detail follow. Plan truths that purely restate a roadmap SC are folded into its row.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | [SC1] The full map (tombstones included) round-trips through a human-readable JSON dump that is greppable with standard tools | ✓ VERIFIED | Re-read codec.go: `slices.Sorted(maps.Keys(...))`, `hex.EncodeToString` 32-char versions, base64 data, `SetEscapeHTML(false)` (codec.go:62-85). `TestEngine_endToEndRestart` + `TestCodec_roundTripProperty` in the fresh `-race -count=1` run: PASS |
| 2 | [SC2] A write returns OK only after a dump containing it completed (version watermark); on dump failure it errors while staying in memory; kill-and-restart recovers via main → .prev → empty start, discarding bad-checksum records | ✓ VERIFIED | Cover discipline intact at engine.go:266-284 (`cover := e.lastLocal` captured under mu before `Records()`; confirmWaiters resolves only covered waiters). `TestEngine_dumpFailureFansOut`, `TestFileStore_fallbackChain` (5-row table), `TestEngine_restartDiscardsTornRecord` — all in the fresh PASS run |
| 3 | [SC3] A sustained ~1000 writes/s burst still sees every write confirmed within bounded latency — debounced dumps coalesce but never starve | ✓ VERIFIED | Deadline armed from clean-to-dirty `oldest`, never reset by later changes (engine.go:202-220). `TestEngine_burstBoundedConfirmation` (synctest, boundary-exact) in the fresh PASS run |
| 4 | [SC4] Named snapshots can be created and loaded; restore replays the snapshot as ordinary writes/deletes with fresh versions so it replicates and wins by normal LWW | ✓ VERIFIED | `TestEngine_snapshotCreateLoadRestore` PASS (fresh run): byte-equal copy, both tombstone directions, absent keys untouched, fresh-version dominance. **Flagged assumption (a) re-verified in code** — see below |
| 5 | [VERS-04 persistence half] The loader computes the FULL watermark and Open seeds clock.NewClock with it | ✓ VERIFIED | `Watermark` keeps the (l, c) pair (codec.go:168-179); `Open` calls `clock.NewClock(now, initialL, initialC, driftWarn)` (engine.go:120-121); `TestWatermark_fullProperty` + `TestWatermark_equalCounterDomain` PASS |
| 6 | [01-T2] Dump is JSONL: header object then one record per line sorted by key, byte-stable, tombstones included, lossless round-trip | ✓ VERIFIED | Codec re-read confirms; `TestCodec_byteStability` + `TestCodec_roundTripProperty` PASS |
| 7 | [01-T3] Dump greppable: each key on its own line, base64 values, UUID-shaped 32-char hex versions | ✓ VERIFIED | Tracer file-shape assertions (engine_test.go:146-164); `storage.RecordChecksum` verbatim at codec.go:142, no crc32 import anywhere in persist |
| 8 | [01-T4] Save atomic and crash-safe: temp in target dir, file fsync, rotate to .prev, rename, parent-dir fsync; Load serves .prev on missing/truncated/unparseable main; empty start only when neither reads | ✓ VERIFIED | `saveAtomically`/`syncedTemp`/`placeFile`/`syncDir` sequence re-read (dumpfile.go:97-174); `TestFileStore_atomicRotation` + `TestFileStore_fallbackChain` PASS |
| 9 | [01-T5] Bad-checksum record discarded and counted while siblings load; unknown format_version and structural faults make the generation unreadable | ✓ VERIFIED | `decodeRecord` per-record discard vs `errUnsupportedFormat` generation error (codec.go:104-151); `TestCodec_discardClasses` + `TestCodec_truncatedGenerationUnreadable` PASS |
| 10 | [01-T7] On dump failure every current waiter immediately receives ErrDumpFailed, record stays readable, no retry loop | ✓ VERIFIED | `failWaiters` fans out, re-dirties from a fresh epoch (engine.go:311-326); `TestEngine_dumpFailureFansOut` PASS |
| 11 | [01-T8] Run's goroutine exits on context cancel and every stranded waiter resolves with ErrClosed — no leaked goroutine, no blocked writer | ✓ VERIFIED | `sweepClosed` flips `closed` before taking waiters under one mutex hold (engine.go:230-250); `TestEngine_runExitClosesWaiters` + `TestEngine_closeSurface` + both flush-race regressions PASS |
| 12 | [02-T1] Snapshot(name) forces unconfirmed state to disk via a completed dump, then byte-copies the finished main dump; dirty state forces a new dump first | ✓ VERIFIED | `Snapshot` = flush then `CreateSnapshot` (engine.go:460-466); `flushDump` dirty-check fast path (engine.go:187-195); `TestEngine_snapshotCreateLoadRestore` bytes.Equal + `TestEngine_snapshotFlushForcesDump` exact attempt count PASS |
| 13 | [02-T2] Restore(name) replays ONLY the snapshot's keys as ordinary writes/deletes with fresh versions, returns only after one final dump covering every replayed write | ✓ VERIFIED | `Restore` (engine.go:482-499): `ReadSnapshot` → `Decode` → sorted-key `replayRecord` via store.Set/store.Delete (no Merge) → `markDirty` + final `flush`. **Assumption (a) — batched confirming flush — still holds in code and tests** |
| 14 | [02-T3] Replayed records win by normal LWW; restoring twice converges | ✓ VERIFIED | Both tombstone directions + fresh-version dominance directly tested (PASS). Double-restore convergence remains entailed by Phase 2's property-tested HLC monotonicity + tested per-restore dominance (same composition argument as the prior report — observation, not gap) |
| 15 | [02-T4] Names outside [A-Za-z0-9._-]{1,64}, or "." and "..", rejected with ErrInvalidSnapshotName before any filesystem touch | ✓ VERIFIED | `validateName` runs first in both operations (snapshot.go:38-42, 59-63; charset+length+dot-specials at 96-106); `TestFileStore_nameValidation` PASS |
| 16 | [02-T5] Snapshot overwrite is atomic in place (no .prev); a concurrent dump cannot tear it | ✓ VERIFIED | **Assumption (b) re-verified**: `saveAtomically(..., prevPath="")` for snapshots (snapshot.go:54; empty-prevPath skip documented dumpfile.go:92-94); WR-01 FileStore mutex serializes CreateSnapshot against Save's rotate window (dumpfile.go:35-43, 62-66); `TestFileStore_snapshotOverwriteAtomic` + `TestFileStore_snapshotConcurrentWithSave` + `TestFileStore_snapshotWaitsOutSaveWindow` PASS |
| 17 | [02-T6] Missing snapshot read wraps fs.ErrNotExist; snapshotting with no dump returns ErrNoDump | ✓ VERIFIED | `readMainDump` maps fs.ErrNotExist→ErrNoDump (snapshot.go:70-76); `TestFileStore_snapshotErrorClasses` + `TestEngine_restoreMissingSnapshot` PASS |
| 18 | [01-T1 tracer] Local write flows end-to-end with on-disk confirmation at Set-return and post-restart dominance | ✓ VERIFIED | `TestEngine_endToEndRestart` against the real filesystem PASS (fresh run); `TestEngine_openEmptyStart` PASS |
| 19 | [01-T6] Debounce 50 ms, max wait forces dump start no later than oldest+1 s, exact in virtual time | ✓ VERIFIED | `defaultDebounce`/`defaultMaxWait` + `Config.normalized` (engine.go:18-43); `TestEngine_confirmDebounceTiming` + burst boundary-exact test PASS; `dump_max_wait 1s` at docs/IDEA.md:148 |

**Score:** 19/19 truths verified (0 present, behavior-unverified)

### Phase-4 Delta Scrutiny (Regeneration Focus)

The one covered-source change since the verified tree — `Engine.ApplyRemote` (engine.go:424-435, commits f839c0d/404269c) — was adversarially checked against PERS-04's write-confirmation discipline:

1. **Purely additive.** `git diff 617951e HEAD -- internal/app/persist/engine.go` is exactly +26 lines: one new method inserted between `await` and `Get`. Not one line of the confirmation machinery (`runDump`, `confirmWaiters`, `failWaiters`, `await`, `flush`, `sweepClosed`) changed.
2. **The cover cannot be raised by a remote write.** `ApplyRemote` never touches `lastLocal` — only `await` raises it, under mu, after the local record's store installation. Therefore a remote merge can never lift the cover above what local writes justify, so no local waiter can be falsely confirmed by a remote-driven cover increase.
3. **No remote write is ever dump-confirmed.** `ApplyRemote` registers no waiter; it merges, marks dirty and nudges `changed` (non-blocking). Pinned behaviorally by `TestEngine_applyRemoteTriggersDump` (engine_test.go:1236-1312): it asserts ApplyRemote returned BEFORE the dump (line 1273-1276 — the no-waiter proof), that the eventual debounce dump carries the merged record, and that an oversize record's ErrValueTooLarge surfaces unchanged without wedging the engine (T-04-04).
4. **A remote merge racing a local waiter's dump cannot break on-disk containment.** A concurrent Merge on a confirmed waiter's key either dominates (the dump carries a dominating record — the same LWW-supersession outcome a second local write already produced in phase 3) or loses (LWW keeps the local record; the dump carries it exactly). Either way the confirmed write's effect is on disk.
5. **Async persistence of remote records is the documented REPL-03 memory-apply contract** (Ack on memory-apply, dump on the debounce schedule) — a phase-4 design decision reserved by phase 3's own `await` doc ("Phase 4's remote merges will reuse the dirty path without waiters"), not a PERS-04 violation: PERS-04 governs the local write path, which is unchanged.

Also in the delta: `port/doc.go` +10 lines (documentation of the phase-4 repl seam — no contract change) and `engine_test.go` +92 lines (the two new tests above, both passing).

### Executor-Flagged Assumptions (Re-Verified)

| Assumption | Still Holds | Evidence |
|------------|-------------|----------|
| (a) Restore is semantics-level D-09: per-key ordinary writes with fresh versions, one batched confirming flush | ✓ | engine.go:482-499 — `replayRecord` via store.Set/store.Delete only (never Merge), sorted keys, then `markDirty` + single `flush`; `TestEngine_snapshotCreateLoadRestore` PASS in fresh run |
| (b) Snapshot overwrite atomically in place, no .prev of its own | ✓ | snapshot.go:54 empty prevPath; dumpfile.go:148-155 skips rotation when prevPath==""; `TestFileStore_snapshotOverwriteAtomic` PASS |
| File modes 0600/0700 | ✓ | `dirMode 0o700` (dumpfile.go:30) with in-code rationale; dump/snapshot files keep CreateTemp's 0600 |
| Nolint suppressions in AGENTS.md-sanctioned form | ✓ | Exactly 5, each naming its linter with a written reason: 4× gosec G304 (config-derived paths) + 1× iface (port seam, dump.go:15); `.golangci.yml` Origin header intact (upstream strict 2.15.0); `make lint` — which runs `golangci-lint config verify` — exit 0 |

### Required Artifacts

All artifacts exist, are substantive, and are wired. `verify.artifacts` reports Go method-style "Missing export" flags (`DumpStore.Save`, `Engine.Set`, `FileStore.Load`, `Engine.Snapshot`, ...) — false negatives of the generic grep against Go method-receiver syntax (`func (e *Engine) Set` vs `func Set`); every flagged method confirmed present by direct code reading this run. The `TestEngine_restoreReplaysSnapshot` contains-miss is the plan's own frontmatter-vs-task naming conflict, resolved by the executor to `TestEngine_snapshotCreateLoadRestore` (the name the plan's own `<verify>` command greps for); the test exists at engine_test.go:831.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/port/dump.go` | DumpStore seam, 4 methods | ✓ VERIFIED | 40 lines; Save/Load/CreateSnapshot/ReadSnapshot declared with full contracts |
| `internal/app/persist/codec.go` | Encode/Decode/Parse/Watermark | ✓ VERIFIED | 179 lines; all exports + three per-record discard classes; RecordChecksum verbatim |
| `internal/app/persist/engine.go` | Write-confirmation engine | ✓ VERIFIED | 573 lines (546 + the additive ApplyRemote seam); all exports incl. Snapshot/Restore and phase-4's ApplyRemote |
| `internal/app/persist/errors.go` | ErrDumpFailed, ErrClosed | ✓ VERIFIED | Both sentinels with Phase 5 mapping docs |
| `internal/app/persist/doc.go` | Package contract | ✓ VERIFIED | Present, documents Run lifecycle, D-04 timing, snapshot contracts |
| `internal/persistfile/dumpfile.go` | Atomic save, .prev rotation, fallback Load | ✓ VERIFIED | 174 lines; canonical sequence + WR-01 mutex; dirMode 0700 |
| `internal/persistfile/snapshot.go` | Snapshot adapter | ✓ VERIFIED | 106 lines; validateName before fs touch, ErrNoDump, ErrInvalidSnapshotName |
| `internal/app/persist/engine_test.go` | Tracer + synctest suite | ✓ VERIFIED | 19 test functions incl. the phase-4 TestEngine_applyRemoteTriggersDump (1236) and TestEngine_configShrunkValueLimits (1320) |
| `internal/app/persist/codec_test.go` | Seeded property suite | ✓ VERIFIED | 6 test functions: round-trip/byte-stability/discard/watermark laws |
| `internal/persistfile/dumpfile_test.go` | Rotation, chain table, hygiene, smoke | ✓ VERIFIED | 5 test functions |
| `internal/persistfile/snapshot_test.go` | Name-validation security table | ✓ VERIFIED | TestFileStore_nameValidation + overwrite + error classes |
| `internal/persistfile/dumpfile_internal_test.go` | WR-01 regression (beyond plan) | ✓ VERIFIED | Internal-package rotate-window driver |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| persist/engine.go | port/dump.go | `port.DumpStore` | ✓ WIRED | Engine field `dumps port.DumpStore`; Save in runDump, Load with Parse in Open |
| persist/codec.go | storage/store.go | `storage.RecordChecksum` | ✓ WIRED | decodeRecord verifies via storage.RecordChecksum (codec.go:142); no crc32 import |
| persistfile/dumpfile.go | port/dump.go | `port.DumpStore` | ✓ WIRED | Compile-time assertion `var _ port.DumpStore = (*FileStore)(nil)` (dumpfile.go:46) |
| persist/engine.go | clock/clock.go | `clock.NewClock` | ✓ WIRED | Open seeds NewClock(now, initialL, initialC, driftWarn) (engine.go:121) |
| persist/engine.go | persistfile/snapshot.go | `CreateSnapshot\|ReadSnapshot` | ✓ WIRED | Snapshot → CreateSnapshot (engine.go:465); Restore → ReadSnapshot (engine.go:483) |
| persist/engine.go | persist/codec.go | `codec.Decode` | ✓ WIRED | Same-package call — `Decode(data)` unqualified at engine.go:112 (Open) and 487 (Restore); the grep pattern misses the qualifier-free form; link intent (one format, one discard discipline) holds |

### Data-Flow Trace (Level 4)

Library code — data-flow applies to the dump pipeline:

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| engine.runDump → codec.Encode → dumps.Save | `block` | `store.Records()` live snapshot | Yes — real store map, sorted, hex/base64 encoded, written through the real file adapter | ✓ FLOWING |
| Open → dumps.Load(Parse) → Decode → Watermark → NewClock | `records`/`initialL,initialC` | On-disk dump bytes (real fs in tracer test) | Yes | ✓ FLOWING |
| Restore → ReadSnapshot → Decode → store.Set/Delete | `records` | Snapshot file bytes | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

Run on branch gsd/phase-4-replication-and-deterministic-simulator at HEAD d816234, cache bypassed (`-count=1`):

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full persist suite under race (19 tests: tracer restart, burst bound, failure fan-out, snapshot/restore, flush races, close surface, applyRemote, config-shrunk limits) | `go test -race -count=1 ./internal/app/persist/...` | ok, 2.841s | ✓ PASS |
| Full persistfile suite under race (8 tests: rotation, fallback chain, hygiene, snapshot smoke, concurrent save, name validation, overwrite, error classes, save-window) | `go test -race -count=1 ./internal/persistfile/...` | ok, 5.636s | ✓ PASS |
| Strict lint gate incl. config verify + depguard boundary | `make lint` | exit 0 | ✓ PASS |
| Phase-3 commits left the dependency files untouched | `git diff --stat 05123bb 617951e -- go.mod go.sum` | empty | ✓ PASS |
| Summary commit hashes resolvable | `verify.commits 0cdca449... c34d8c56... 617951e 05123bb` | all_valid: true, 4/4 | ✓ PASS |
| Port/app boundary | no os/syscall/net import under internal/app/persist or internal/port (non-test) | clean | ✓ PASS |

Note on `go.mod`: the `require google.golang.org/protobuf v1.36.12` now present entered in phase 4 (commit 9690b4a lineage) — exactly the AGENTS.md dependency policy ("protobuf enter go.mod in the phases that first use them"). Phase 3's own commit range is clean; not a violation.

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by the plans; the plans' probe clauses (PERS-01/02/03/05 concurrency, boundary, idempotency) are discharged as in-tree tests enumerated above (e.g. `TestFileStore_snapshotConcurrentWithSave`, boundary-exact synctest assertions, `TestEngine_applyRemoteTriggersDump`). Not applicable as external probe scripts.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|---------------------|----------|----------|
| PERS-01 | 03-01 | Human-readable greppable JSON dump of full map | ✓ SATISFIED | Truths 1/6/7; codec + tracer + properties, fresh PASS run |
| PERS-02 | 03-01 | Atomic dumps, .prev fallback chain, bad-checksum discard | ✓ SATISFIED | Truths 2/8/9; rotation + chain table + torn-record test |
| PERS-03 | 03-01 | Debounced dumps with bounded max wait, no starvation | ✓ SATISFIED | Truths 3/19; burst test exact in virtual time |
| PERS-04 | 03-01 | Write OK only after covering dump; failure stays in memory | ✓ SATISFIED | Truths 2/10/11/18 + Phase-4 Delta Scrutiny (ApplyRemote preserves the discipline) |
| PERS-05 | 03-02 | Named snapshots; restore replays as ordinary writes, wins by LWW | ✓ SATISFIED | Truths 4/12-17; snapshot/restore suite |
| VERS-04 (persistence half) | 03-01 | Dump loader seeds full watermark via NewClock | ✓ SATISFIED | Truth 5; Watermark + Open wiring + dominance assertion |

Orphaned requirements: none — REQUIREMENTS.md maps exactly PERS-01..PERS-05 to Phase 3 (traceability rows all Complete) and the plans claim exactly those IDs.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/port/dump.go | 15 | `//nolint:iface` with linter name + written reason | ℹ️ Info | AGENTS.md deviation 3 sanctioned; cross-package seam by design (D-01) |
| internal/persistfile/dumpfile.go | 76,168 | `//nolint:gosec` G304 ×2 with reasons | ℹ️ Info | Variable-path reads are the adapter's purpose; paths derive from config |
| internal/persistfile/snapshot.go | 65,71 | `//nolint:gosec` G304 ×2 with reasons | ℹ️ Info | Validated names / configured paths only |

No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any phase file (non-test or test). No stubs, no empty implementations. One `t.Skip` in the suite — the documented root-euid environment guard in `TestFileStore_saveFailureHygiene` (dumpfile_test.go:163), not a disabled test.

### Human Verification Required

None — infrastructure/foundation phase (library-level persistence; no user-facing UI). Every behavior-dependent truth has a passing named behavioral test; the full scoped suite was re-run fresh under `-race` for this regeneration and passed, including the phase-4 delta tests.

### Gaps Summary

None. All 19 must-have truths re-verified against the current tree with fresh behavioral evidence; all artifacts present, substantive, wired, data-flowing; all six key links wired; all five PERS requirements plus the VERS-04 persistence half satisfied with no orphans; lint gate (with config verify and the depguard boundary) green; the dependency-freeze truth holds for phase 3's own commit range. The phase-4 `Engine.ApplyRemote` seam that stale-dated the prior report is purely additive and provably preserves PERS-04's cover discipline — it cannot raise the cover, registers no waiter, and is behaviorally pinned to return before the covering dump. Both executor-flagged assumptions (batched restore confirmation; snapshot overwrite without .prev) and the two prior flagged items (0600/0700 modes; documented nolint suppressions) still hold in code and tests.

---

_Verified: 2026-11-08T18:11:18Z_
_Verifier: Claude (gsd-verifier)_
