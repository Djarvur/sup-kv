---
phase: 05-persistence
verified: 2026-11-23T20:57:31Z
status: passed
score: 64/64 must-haves verified
covered_files:
  - .golangci.yml
  - .planning/phases/05-persistence/05-01-PLAN.md
  - .planning/phases/05-persistence/05-01-SUMMARY.md
  - .planning/phases/05-persistence/05-02-PLAN.md
  - .planning/phases/05-persistence/05-02-SUMMARY.md
  - .planning/phases/05-persistence/05-03-PLAN.md
  - .planning/phases/05-persistence/05-03-SUMMARY.md
  - .planning/phases/05-persistence/05-04-PLAN.md
  - .planning/phases/05-persistence/05-04-SUMMARY.md
  - .planning/phases/05-persistence/05-05-PLAN.md
  - .planning/phases/05-persistence/05-05-SUMMARY.md
  - .planning/phases/05-persistence/05-06-PLAN.md
  - .planning/phases/05-persistence/05-06-SUMMARY.md
  - .planning/phases/05-persistence/05-07-PLAN.md
  - .planning/phases/05-persistence/05-07-SUMMARY.md
  - .planning/phases/05-persistence/05-08-PLAN.md
  - .planning/phases/05-persistence/05-08-SUMMARY.md
  - .planning/phases/05-persistence/05-09-PLAN.md
  - .planning/phases/05-persistence/05-09-SUMMARY.md
  - .planning/phases/05-persistence/05-10-PLAN.md
  - .planning/phases/05-persistence/05-10-SUMMARY.md
  - .planning/phases/05-persistence/05-11-PLAN.md
  - .planning/phases/05-persistence/05-11-SUMMARY.md
  - .planning/phases/05-persistence/05-12-PLAN.md
  - .planning/phases/05-persistence/05-12-SUMMARY.md
  - .planning/phases/05-persistence/05-CONTEXT.md
  - .planning/phases/05-persistence/05-REVIEW-DISPOSITION.md
  - .planning/phases/05-persistence/deferred-items.md
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - cmd/sup-kv/operations_test.go
  - cmd/sup-kv/shutdown_internal_test.go
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - docs/adr/0012-equal-version-tiebreak.md
  - go.mod
  - go.sum
  - internal/app/internal/simnet/simnet.go
  - internal/app/persist/backoff_test.go
  - internal/app/persist/bench_test.go
  - internal/app/persist/codec.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/confirm.go
  - internal/app/persist/doc.go
  - internal/app/persist/encode_stream_internal_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_blocking_test.go
  - internal/app/persist/engine_replay_internal_test.go
  - internal/app/persist/engine_test.go
  - internal/app/persist/errors.go
  - internal/app/persist/facade.go
  - internal/app/persist/flush.go
  - internal/app/persist/fuzz_dump_test.go
  - internal/app/persist/load_stats_internal_test.go
  - internal/app/persist/load_stats_test.go
  - internal/app/persist/loader.go
  - internal/app/persist/scheduler.go
  - internal/app/persist/snapshot.go
  - internal/app/persist/timeout_internal_test.go
  - internal/app/persist/timeout_test.go
  - internal/app/repl/core.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_dump_test.go
  - internal/app/storage/merge_test.go
  - internal/app/storage/prop_small_space_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/config/config.go
  - internal/config/usage.go
  - internal/config/validate.go
  - internal/grpcapi/limits.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/metrics/metrics.go
  - internal/metrics/metrics_test.go
  - internal/persistfile/dumpfile.go
  - internal/persistfile/dumpfile_test.go
  - internal/persistfile/fsops.go
  - internal/persistfile/fsops_test.go
  - internal/persistfile/load_outcome_test.go
  - internal/persistfile/open_seed_test.go
  - internal/persistfile/red_value_limit_test.go
  - internal/persistfile/saveerror.go
  - internal/persistfile/saveerror_test.go
  - internal/persistfile/snapshot.go
  - internal/persistfile/sweep_test.go
  - internal/port/dump.go
  - internal/stand/persist_test.go
  - internal/stand/stand_red_test.go
  - internal/udprepl/decode_negative_test.go
  - scripts/red-expectations.tsv
  - scripts/red-runner
covered_digest: "v4:sha256:eabfdaf0d620eb1ab36e62841a0dd727499f6a8207e6b776c29b61808196ee08"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 64/64
  gaps_closed: []
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "`make test` is fully green on the current tree"
    addressed_in: "Phase 8"
    evidence: "Phase 8 goal 'make perf with thresholds' and plans 08-05 (deterministic store-call and pending-visit gates) / 08-11 (phase gate: full make gates, recorded make perf). The only red is `TestCore_keyCountCostFlat`, a load-sensitive wall-clock ratio gate (`internal/app/repl/bench_test.go`) recorded as a pre-existing Phase-8-owned flake in `deferred-items.md`; it passes standalone (ratios 0.91 / 1.01)."
  - truth: "Test harness port picking is race-free (`freeUDPPort` bind-close-return)"
    addressed_in: "Phase 8"
    evidence: "Phase 8 plan 08-06 — 'Shared port allocator with package bands; every socket package adopts it'. `deferred-items.md` records the pick-to-bind class as pre-existing and unrelated to the Phase-5 plans."
advisory: []
behavior_unverified_items: []
coincidental_reliance_items: []
human_verification: []
---

# Phase 5: Persistence Verification Report

**Phase Goal:** The dump path is honest about its limits and failures — one enforced value limit, visible load outcomes, generation integrity, bounded memory and blocking — so "a confirmed write is on disk" is a tested guarantee rather than a convention.
**Verified:** 2026-11-23T20:57:31Z
**Status:** passed
**Re-verification:** Yes — freshness regeneration, not a gap-closure round. The previous report was written at `633f918` on 2026-11-22; phases 6, 7 and 8 have since edited covered files (`cmd/sup-kv/main.go` was decomposed by the `internal/config` extraction, `.golangci.yml` gained the depguard rules, `docs/OPERATIONS.md` gained the Phase-7 catalog, and the persist store/facade gained the Watch-durability seam). Everything below was re-derived from the current tree; the prior report's claims were used only as a checklist.

## Re-verification Basis

The prior report (`633f918`, `passed`, 64/64) was treated as a hypothesis and re-tested. The later phases' edits are additive in the areas this phase owns:

- `internal/app/storage/store.go`: the `OnChange` tap gained a generation parameter (APIS-04 watermark); `join`, `RecordsEqual` and `MergeChanged` are semantically unchanged.
- `internal/app/persist`: `OnConfirmed` was added to the scheduler's success path; `ErrSnapshotIncomplete` and the partial-snapshot refusal were added; `LiveVersions` was added to the facade. The five roles, the confirmation protocol, the retry ladder and the flush/timeout bounds are unchanged.
- `cmd/sup-kv/main.go`: flag parsing and validation moved to `internal/config`; the value-limit and datagram-ceiling messages are byte-identical to the ones phase 5 verified, `exitUsage` is still 2, and `shutdown`/`awaitEngineStop` are unchanged.
- `.golangci.yml`: additive `depguard` rules only, with the documented written justification (AGENTS.md deviation 1); `make lint` is green.

No phase-05 truth regressed. Two unrelated flakes are recorded under **Deferred Items** below.

## Goal Achievement

### Roadmap Success Criteria

| # | Criterion | Status | Evidence |
| - | --------- | ------ | -------- |
| 1 | Value limit defined once and enforced everywhere (P-01 turns green): exact `value_max_bytes` survives restart at any accepted flag value; invalid value exits 2; the 61 440 constant exists once | ✓ VERIFIED | Production `grep 61440 --include=*.go` outside the module cache finds only `internal/app/storage/store.go:21`; `Config.ValueLimit()` (`store.go:141-147`) is used by `Open` before the store exists (`loader.go:47`) and by `NewStore` (`store.go:150-156`); `Store.ValueMaxBytes()` (`store.go:161-163`) feeds the encoder (`scheduler.go:82`) and snapshot decode (`snapshot.go:283`); the CLI default is the constant (`internal/config/config.go:235`), 0 exits 2 naming the flag, 64 409 exits 2 with the ceiling text (direct binary probe this round); frozen P-01 tests ran green through the red runner |
| 2 | Dump-load outcomes visible, never destroy evidence: `.prev` -> one WARN with path+cause; both unusable -> ERROR + nonzero metric; absent -> neither; preserved under a separate name; format carries a record count so line-boundary truncation is incomplete; one corrupted record discards only that record | ✓ VERIFIED | `persistfile.Load` four outcomes (`dumpfile.go:127-193`), `preserve` collision-probes before rename (`dumpfile.go:206-241`), header `record_count` (`codec.go:102-105`, census `codec.go:331-344`), per-record classes (`codec.go:349-384`), one WARN/one ERROR in `cmd` (`main.go:481-579`), metrics families (`metrics.go:234-283`); all seven stand process tests and the load-outcome table pass |
| 3 | Failure handling bounded and honest: dirty after backup/empty start (first rotation never displaces the last good generation; Snapshot right after decodes back); one ERROR per failure with cause, growing pause to a cap, health documented; skipped records counted/logged | ✓ VERIFIED | `startLoadDirty` (`loader.go:113-120`), `TestEngine_startDirty` (Prompt/Backup/Unusable dirty; Main-with-discard clean; snapshot decodes back); ladder 100 ms..5 s (`scheduler.go:19-20,282-291`), `OnDumpFailure` once per attempt (`scheduler.go:259-278`), `TestEngine_backoffLadder`, `TestStand_persistDumpFailure` (EACCES: ERROR+class, SERVING, bounded rate, recovery); skip accounting (`main.go:451-455`, `metrics.go:263-283`), `TestEngine_skipEvidenceWindow` |
| 4 | Resources bounded: streaming encode at most two state sizes with the memory model documented; a hung fsync cannot block writes, snapshots or shutdown indefinitely; orphan temps collected; parent synced; `fsync_darwin.go` gone | ✓ VERIFIED | Streaming path: one `Records()` + `SaveStream(EncodeTo)` (`scheduler.go:228-252`); `TestEncodeTo_maxSingleWrite`, bench printed `bytes/op 5 147 571`, `dump-bytes 5 147 571`, `peak-live-B 11 036 512`, p50/p99; model documented (`OPERATIONS.md:190-215`); writes/snapshots bounded (`TestEngine_confirmationTimeout`, `TestFlush_timeout`, `TestSnapshot_timeout`); shutdown bounded by `FlushTimeout(shutdownGrace)` and `awaitEngineStop`, whose expiry branch `TestAwaitEngineStop` exercises; sweep/parent-sync/dir-fsync (`TestSweep_*`, `TestEnsureDir_*`, `TestSyncDirTolerant`); no `fsync_darwin.go` in `internal/persistfile/` |
| 5 | Structure serves seams: `persist.Engine` split into five roles with the `DumpStore` port returning outcome and failure cause; timers/`time.Now()` behind the repl-style seam so the simulator exercises dump↔replication; equal-version different content resolved by a recorded ADR, implemented and tested | ✓ VERIFIED | Five role files (`loader.go`, `facade.go`, `scheduler.go`, `confirm.go`, `snapshot.go`) + composition `engine.go`; `port.DumpStore.Load(now, parse) LoadResult` with `LoadOutcome`/`Generation`/`GenerationFailure` (`port/dump.go:28-110`); only wall-clock references are the two defaults in `normalized()` (`engine.go:77,80`); `TestScenario_dumpNotify`, `TestScenario_tiebreakConverges`, `TestEngine_injectedClock` ran green; ADR 0012 records the deterministic `(version, content)` law; store and core wired |

### Observable Truths (plan must-haves)

**Score:** 64/64 truths verified, 0 present-behaviour-unverified.

| # | Plan | Truth | Status | Evidence |
| - | ---- | ----- | ------ | -------- |
| T-01.1 | 05-01 | 61 440 exists exactly once in production; store default, pre-store decode and encoder all derive through `Config.ValueLimit()`/`Store.ValueMaxBytes()` | ✓ VERIFIED | `store.go:21` is the only production literal; `loader.go:47` decodes through `ValueLimit()`; `store.go:141-163`; the codec takes the limit as an argument (`codec.go:140,258`); `internal/config/config.go:235` sets the flag default from the constant |
| T-01.2 | 05-01 | A record of exactly the configured limit survives restart at two accepted flag values (64 000, 8); one byte over is rejected by the store | ✓ VERIFIED | Frozen `TestRed_P01_valueLimit` (63 000 and exactly 64 000, SIGKILL restart) green in the red runner; `TestEngine_exactValueLimitRestart` (8-byte limit + 9-byte reject) in the green persist suite |
| T-01.3 | 05-01 | `--value-max-bytes=0`/negative exit 2 naming the flag; 64 400 and 64 408 start; 64 409 exits 2 with the unchanged ceiling message | ✓ VERIFIED | `TestMain_valueMaxNonPositive`, `TestMain_valueMaxDatagram` (both subtests of the boundary, 64 000 and 64 408 start-and-serve) pass; direct binary probe this round: 0 -> exit 2, `value-max-bytes must be positive: --value-max-bytes=0`; 64 409 -> exit 2, the same `65507-byte UDP ceiling` text phase 5 verified (moved to `internal/config/validate.go:222`, text unchanged); `--help` default 61440 |
| T-01.4 | 05-01 | The two frozen P-01 tests pass unedited | ✓ VERIFIED | `git diff 4f81fd7..HEAD` on `internal/persistfile/red_value_limit_test.go` and `internal/stand/stand_red_test.go` is empty; red runner 7/7 rows OK |
| T-02.1 | 05-02 | The dump announces `record_count`; fewer record lines is a generation-level error; a v1.0 dump without the field takes the legacy path | ✓ VERIFIED | `codec.go:102-105,130`; `checkRecordCount` (`codec.go:331-344`); `TestCodec_headerCensus` (writer count, truncated, extra-lines tolerated, legacy no-census) |
| T-02.2 | 05-02 | One syntax-corrupt line costs exactly that record; the rest load; the bad line counts `Malformed`; the `json.Decoder` poison behaviour is gone | ✓ VERIFIED | Per-line `json.Unmarshal` (`codec.go:349-384`); `grep -c json.NewDecoder codec.go` = 0; `TestCodec_corruptLineIsolated` (intact siblings load, `Malformed=1`) |
| T-02.3 | 05-02 | `DecodeStats` keeps checksum, version, value-limit and malformed separate, each counted, with `Total()` | ✓ VERIFIED | `codec.go:194-200`; four counters and `Total()`; `TestCodec_discardClasses` |
| T-02.4 | 05-02 | The line bound is a format property (config-independent); a dump written at the default still reads under a tiny limit; over-limit is `ValueLimit`, never `Malformed`/generation fault; only an impossible line is generation-level | ✓ VERIFIED | `MaxDumpLineBytes` constant takes no configuration (`codec.go:50-67`); bounded `readDumpLine` (`codec.go:299-322`); `TestCodec_shrunkLimitKeepsOtherRecords`; overlong-line row in `TestCodec_headerCensus` |
| T-02.5 | 05-02 | A 1 024-byte quote/control-char key with a 61 440-byte value round-trips | ✓ VERIFIED | `TestCodec_escapedMaxKeyRoundTrip` (escape-expanded bound) |
| T-02.6 | 05-02 | A v1.1 dump cut mid-line is a torn-tail generation error; a v1.0 line-boundary cut remains the documented legacy limitation | ✓ VERIFIED | `errTornTail` (`codec.go:90`), `errIncompleteDump` (`codec.go:80`), `Decode`'s unterminated-malformed branch (`codec.go:284-286`); `TestCodec_truncatedGenerationUnreadable`; legacy row in the census test; contract stated (`codec.go:230-257`) |
| T-03.1 | 05-03 | Join order `(version, content)`: greater version wins; identical equal version is a no-op; differing equal version resolves remove-wins then `bytes.Compare` | ✓ VERIFIED | `store.join` (`store.go:417-450`); `TestMerge_equalVersionContentLaw`, `TestMerge_equalVersionBranchTable`, `TestProp_mergeSmallSpace` |
| T-03.2 | 05-03 | The law reaches the wire: identical content Acks with no reply; incoming tie-break win Acks; stored tie-break win answers with one unicast `DataReplication`, no Ack | ✓ VERIFIED | `core.applyRemote` (`core.go:1635-1705`); `TestReplication_equalVersionContentLaw` (all three branches ran green) |
| T-03.3 | 05-03 | `ApplyRemote` dirties through `MergeChanged`'s `changed`, never a Conflict/version heuristic | ✓ VERIFIED | `facade.go:40-50` (unchanged in substance; the generation parameter is Phase-6 additive); `TestEngine_applyRemoteEqualVersionContent` |
| T-03.4 | 05-03 | Explicit same-version/different-content `DataReplication` drives the wire rule; same seed replays identically; cross-node convergence via `NotifyInstalled` | ✓ VERIFIED | `TestReplication_equalVersionSameSeedTrace`; `TestScenario_tiebreakConverges` (announcing and receiving node hold the winner) — both passed this round; `Cluster.NotifyInstalled` (`simnet.go:850`) |
| T-03.5 | 05-03 | ADR 0012 records the deterministic deviation, the lost-operation consequence and explicit non-changes | ✓ VERIFIED | `docs/adr/0012-equal-version-tiebreak.md` read: context/decision/consequences/links; names S-HLC-07 and PFXD-01 as non-changes |
| T-04.1 | 05-04 | A dump streams: one `Records()` copy + `SaveStream(fill)` running `EncodeTo`; the five TEST-03 steps are unchanged | ✓ VERIFIED | `scheduler.go:228-252`; `fsops.go:96-117,346-369`; `TestFsops_saveOperationOrder`, `TestFsops_saveOperationOrder_rotation_chain` |
| T-04.2 | 05-04 | `EncodeTo` is byte-identical to `Encode`; no write larger than `maxDumpLineBytes()` | ✓ VERIFIED | `TestEncodeTo_byteIdentical`, `TestEncodeTo_maxSingleWrite` (one write per line; joined stream equals `Encode`) |
| T-04.3 | 05-04 | A fill error removes the temp and surfaces the error; the failure matrix keeps its hygiene | ✓ VERIFIED | `fileFSOps.write` cleanup (`fsops.go:96-117`); `TestFsops_failureAtEachStep` includes the `fill failure cleans the temp` row; `TestFileStore_saveFailureHygiene` |
| T-04.4 | 05-04 | The port carries `Load(now, parse) LoadResult` with outcome/generation/failures, implemented by the adapter and its doubles | ✓ VERIFIED | `port/dump.go:42,54-110`; adapter `dumpfile.go:127-193`; fakes across `persist/*_test.go` and `persistfile/*_test.go`; whole tree builds and vets |
| T-04.5 | 05-04 | The memory model is measured, not guessed: B/op and a sampled peak live; the one `Records()` copy and writer pause stated as residual | ✓ VERIFIED | `bench_test.go:86-141`; this round's run printed `bytes/op 5147571`, `dump-bytes 5147571`, `peak-live-B 11036512`, p50/p99; `OPERATIONS.md:190-215` states the residual writer pause |
| T-05.1 | 05-05 | Load reports Prompt/Main/Backup/Unusable; the accepted-`.prev` path always names the main candidate; evidence is never deleted | ✓ VERIFIED | `dumpfile.go:127-193`; `TestFileStore_loadOutcomeTable`; stand Prompt/Backup/Backup-absent/Unusable/Unusable-single all passed |
| T-05.2 | 05-05 | A generation failing the oracle is renamed `<path>.corrupt-<UTC>[-N]` before any save; only regular files; same-second collisions survive; rotation never overwrites | ✓ VERIFIED | `preserve` (`dumpfile.go:206-241`), `Lstat` regularity, free-name probe; `TestFileStore_loadBackupPreservesMain`, `TestFileStore_preserveCollision`, `TestFileStore_loadNonRegularCandidate` |
| T-05.3 | 05-05 | After a Backup start the first save's rotate is a no-op; `.prev` stays byte-identical | ✓ VERIFIED | `TestFileStore_loadBackupPreservesMain` compares `.prev` bytes before/after `Save` |
| T-05.4 | 05-05 | Every temp carries `.tmp~`, whose `~` is outside the snapshot charset, so no legal snapshot collides and the sweep never keys on a data prefix | ✓ VERIFIED | `orphanTempMarker`/`tempPattern` (`dumpfile.go:43-51`); `CreateTemp(dir, tempPattern)` (`fsops.go:102`); snapshot charset `[A-Za-z0-9._-]` rejects `~` (`snapshot.go:104-127`); `TestFileStore_nameValidation` |
| T-05.5 | 05-05 | The sweep removes only old `.tmp~` regular files (both dirs), keeps young ones, protects main/`.prev`/`.corrupt-*`/snapshots, reports count and first error | ✓ VERIFIED | `sweepOrphans`/`protectedDataName`/`sweepEntry` (`dumpfile.go:252-317`); `TestSweep_removesOnlyOldTemps`, `TestSweep_protectsDataNames`, `TestSweep_reportsUnreadableDir`; stand Prompt test proved the old orphan is removed with count 1 |
| T-05.6 | 05-05 | A newly created dump directory is synced at its parent before the temp write; existing directories do not (five-step order unchanged) | ✓ VERIFIED | `ensureDir` (`dumpfile.go:319-334`); `TestEnsureDir_newDirectorySyncsParent`, `TestEnsureDir_existingDirectoryNoParentSync`; order test unchanged |
| T-05.7 | 05-05 | Directory fsync tolerates ENOTSUP/EINVAL; file-data fsync keeps every error; a save error carries `port.FailureClass` via `errors.Is` | ✓ VERIFIED | `tolerateDirSyncErr` called only from `syncDir` (`fsops.go:175-182,220-228`); `classedSaveError` wraps the cause (`saveerror.go:12-46`); `TestSyncDirTolerant` (ENOTSUP, wrapped EINVAL, EACCES/ENOSPC preserved), `TestClassifySaveError`, failure matrix |
| T-06.1 | 05-06 | `Open` stores a `LoadStats` report and keeps `(*Engine, int)`; `New` engines report zero | ✓ VERIFIED | `loader.go:40-111` (report assignment at 95-107); `engine.go:173-197`; `TestEngine_loadStatsMapsResult` |
| T-06.2 | 05-06 | Dirty after Prompt/Backup/Unusable (Snapshot right after flushes and decodes back; first rotation never displaces the good generation); Main — even with discards — starts clean | ✓ VERIFIED | `startLoadDirty` (`loader.go:113-120`); `TestEngine_startDirty` (all five cases with snapshot decode assertions); `TestFileStore_loadBackupPreservesMain` |
| T-06.3 | 05-06 | Every `markDirty()` wakes an idle dumper via the `changed` nudge (incl. the restore mid-error path); one nudge per write path; a clean engine stays clean | ✓ VERIFIED | `markDirty` nudges (`confirm.go:139-149`); `await` nudges once; mid-error path calls `markDirty` (`snapshot.go`); `TestEngine_markDirtyNudge` (exactly one dump at Debounce, then quiet) |
| T-06.4 | 05-06 | Records skipped by a shrunken limit are counted by class, logged once with count+limit, exposed as a metric, and left on disk through two rotations | ✓ VERIFIED | `stats.Decoded.ValueLimit`; `main.go:451-455`; `metrics.go:263-283`; `TestEngine_skipEvidenceWindow` (byte-identical after load, in `.prev` after one write, gone after two); `TestStand_persistShrunkLimit` |
| T-06.5 | 05-06 | Binary logs exactly one WARN for Backup (path+cause incl. absent main), one ERROR for Unusable (one group per failed candidate), nothing (no WARN/ERROR) for Prompt/Main, one INFO for sweep count, one WARN for sweep error | ✓ VERIFIED | `logLoadStats`/`logBackupFallback`/`logUnusableGenerations` (`main.go:481-579`); stand tests assert the exact message counts (Backup=1, Unusable=0/1, sweep INFO count); the Phase-7 INFO `Dump load` record for Prompt/Main is additive and not a WARN/ERROR, so the truth's enumeration holds |
| T-06.6 | 05-06 | `supkv_dump_load_total{outcome}` and `supkv_dump_records_discarded_total{class}` render every label at zero and the load counts after a load | ✓ VERIFIED | `metrics.go:234-283`; `TestMetrics_families` (all zero lines), `TestMetrics_values` (by-effect samples) |
| T-07.1 | 05-07 | `internal/app/persist` owns no wall clock except the two defaults in `normalized()` | ✓ VERIFIED | `grep -E 'time\.(Now\|NewTimer\|After\|Until\|Since)'` non-test finds only `engine.go:77,80`; every read/arm goes through `cfg.Now`/`cfg.After` |
| T-07.2 | 05-07 | The scheduler arms debounce/deadline/retry from `Config.After`; re-arming replaces the channel; no channel created inside the select | ✓ VERIFIED | `run` loop (`scheduler.go:109-147`), `rearm` (`scheduler.go:176-214`); no `time.After` in the select; `TestEngine_injectedClock`, `TestEngine_confirmDebounceTiming` |
| T-07.3 | 05-07 | Ladder 100 ms -> 200 -> 400 -> 800 -> 1.6 s -> 3.2 s -> 5 s cap; while pending no debounce/deadline armed; success resets and covers waiters | ✓ VERIFIED | `defaultRetryBase`/`defaultRetryCap` (`scheduler.go:19-20`), `nextRetry`/`resetRetry` (`scheduler.go:282-305`); `TestEngine_backoffLadder` asserts the exact virtual instants, the during-backoff write and the reset |
| T-07.4 | 05-07 | Each failed attempt invokes `OnDumpFailure(err, class)` exactly once; `DumpStats` counts DiskFull/Permission/Other beside total/errors | ✓ VERIFIED | `handleDumpFailure` (`scheduler.go:259-278`); `TestEngine_dumpFailureClasses`; `TestEngine_backoffLadder` asserts one callback per failed attempt |
| T-07.5 | 05-07 | Health stays SERVING under persistent dump failure; the EACCES stand test asserts one ERROR, the class label, a bounded attempt rate and recovery | ✓ VERIFIED | No health transition in the dump path; `TestStand_persistDumpFailure` passed this round (permission class + cause, SERVING, bounded window, metric >= 1, recovery) |
| T-07.6 | 05-07 | The simulator exercises dump→confirmation→replication-notify incl. a failure window and shrunk-limit skip; two same-seed runs identical; the same seam proves cross-node equal-version convergence | ✓ VERIFIED | `Cluster.NotifyInstalled` (`simnet.go:850`); `TestScenario_dumpNotify` (confirm/notify order, ladder offsets, failure still notifies, shrunk limit reported, same-seed runs identical) and `TestScenario_tiebreakConverges` both passed this round |
| T-08.1 | 05-08 | `ConfirmTimeout` defaults to 10 s; an uncovered write returns `ErrUnconfirmed`; the record stays installed/dirty; not reported failed or durable | ✓ VERIFIED | `defaultConfirmTimeout` (`engine.go:18`), `await` timeout branch (`confirm.go:98-136`); `TestEngine_confirmationTimeout` asserts the exact deadline, sentinel, installed record and counter |
| T-08.2 | 05-08 | Single-claim resolution: capture never claims; the first resolver claims under the mutex; a hung dump cannot block the timeout; the timeout removes its waiter | ✓ VERIFIED | `waiter.claimed` + every resolver's claim (`confirm.go:187-290`); `TestEngine_timeoutThenDumpSuccess` (late success skips the claimed waiter, no double close), `TestEngine_waiterRemoved` (slice empty after every timeout) |
| T-08.3 | 05-08 | `Flush()` = `FlushTimeout(0)`; both bound send and reply; `Snapshot` uses it; `Restore` returns its partial list on timeout with `ErrUnconfirmed` distinguishable from `ErrDumpFailed` | ✓ VERIFIED | `engine.go:244-294`, `snapshot.go:36-42,110-145`; `TestFlush_timeout` (exact instants), `TestSnapshot_timeout`, `TestRestore_timeoutReturnsPartial` |
| T-08.4 | 05-08 | Shutdown is bounded: `FlushTimeout(shutdownGrace)`, cancel, `engineDone` at most the grace, WARN instead of hanging | ✓ VERIFIED | `shutdown` (`main.go:869-909`) calls `FlushTimeout(shutdownGrace)` then `awaitEngineStop(node.engineDone, shutdownGrace)`; `awaitEngineStop` (`main.go:916-925`); `TestAwaitEngineStop` ran PASS under `-race`: clean cancel silent, abnormal error reported, wedged channel returns >= 25 ms grace, <= 2 s, WARN text asserted |
| T-08.5 | 05-08 | A timed-out write still replicates: Set/Delete notify on `ErrDumpFailed` or `ErrUnconfirmed`; `mapError` maps `ErrUnconfirmed` to Unavailable | ✓ VERIFIED | `server.go:218-221,266-269,496-513`; `TestServer_unconfirmedSetMapsAndNotifies`, `TestServer_unconfirmedDeleteMapsAndNotifies` pass |
| T-08.6 | 05-08 | The frozen blocking test stays green with its 200 ms / 1 s windows unedited | ✓ VERIFIED | `engine_blocking_test.go:24-25` still `200 * time.Millisecond` / `time.Second`, identical to the phase-start base; the file's only change in the phase is the documented `Save`→`SaveStream` fake migration (`bc43705`, plan 05-04), which the prohibition explicitly permits; `TestEngine_blockingSaveHoldsSet` passed |
| T-09.1 | 05-09 | Five internal roles separated: loader/composition root, store facade, dump scheduler, confirmation accounting, snapshot orchestration | ✓ VERIFIED | `loader.go`, `facade.go`, `scheduler.go`, `confirm.go`, `snapshot.go` (+ `engine.go` composition); `flush.go` only carries `flushRequest` |
| T-09.2 | 05-09 | `Engine`'s exported method set/signatures unchanged by the split; consumers compile unedited | ✓ VERIFIED | The split commits' added exported declarations are exact relocations (`+func Open`/`-func Open`, `+type RestoredRecord`/`-type RestoredRecord`, no net new name); `go build ./...` and `go vet ./...` green on the current tree |
| T-09.3 | 05-09 | Confirmation role owns waiters/dirty/epoch/changed/closed/runDone behind its mutex; shares one unexported `*dumpStats` for `unconfirmed` | ✓ VERIFIED | `confirmer` (`confirm.go:42-55`); `stats.unconfirmed.Add(1)` (`confirm.go:135`); capture-before-encode order preserved (`scheduler.go:229-231`) |
| T-09.4 | 05-09 | Scheduler owns ladder/channels/`dumpStats`; `New` and `Open` build one holder shared with the confirmer; facade thin; loader owns Open/mergeRecords/LoadStats | ✓ VERIFIED | `New` (`engine.go:173-189`) creates `stats` once and passes the same pointer to `newConfirmer` and `newScheduler`; `Open` goes through `New` (`loader.go:104`); `DumpStats()` value copy (`scheduler.go:317-329`); `facade.go` forwards |
| T-09.5 | 05-09 | Pure move: suite, lint and darwin green; no behaviour change, no new interface, no new exported name | ✓ VERIFIED | `make test` (all packages except the deferred Phase-8 flake), `make lint`, darwin build/vet green; `grep 'type .* interface' internal/app/persist/*.go` = none; export diff above is relocation-only |
| T-10.1 | 05-10 | OPERATIONS states the single limit, the non-positive exit-2 rule and keeps the 64 408 boundary arithmetic | ✓ VERIFIED | `OPERATIONS.md:64-97`; `TestOperationsDoc` anchors pass |
| T-10.2 | 05-10 | Monitoring lists the four dump/load families with the every-label-at-zero rule | ✓ VERIFIED | `OPERATIONS.md:300-325`; anchors in `operations_test.go:115-118`; names exist in `metrics.go:234,263,480,490` |
| T-10.3 | 05-10 | Lifecycle documents the ladder, health staying SERVING, `Unavailable` for unconfirmed, bounded shutdown | ✓ VERIFIED | `OPERATIONS.md:416-436` |
| T-10.4 | 05-10 | Load-outcomes section documents Prompt/Main/Backup/Unusable, one-WARN/one-ERROR, `.corrupt-<UTC>[-N]`, `preserve_error` hazard, the one-hour marker-only sweep | ✓ VERIFIED | `OPERATIONS.md:125-188`; `TestOperationsDoc` (`## Load outcomes`) |
| T-10.5 | 05-10 | Memory model states one `Records()` copy + one line + 64 KiB writer and names the residual writer pause | ✓ VERIFIED | `OPERATIONS.md:190-215`; `## Memory model` anchor; benchmark printed the supporting figures |
| T-10.6 | 05-10 | Anchors cover the new sections and the file still opens with the trust model | ✓ VERIFIED | `operations_test.go:52` asserts `## Trust model` is the first heading (run green); new section/family anchors listed at `:108-118` |
| T-11.1 | 05-11 | Exactly the eight named registry rows change (at the plan's commit) | ✓ VERIFIED | `git diff 4f81fd7..1f79eef -- docs/SPEC-DEVIATIONS.md` changes exactly `S-PAR-05, S-PERS-08, S-PERS-09, S-ALG-REP-03, S-HLC-07, S-DM-04, S-DM-05, S-DM-08` (13+/11-: the 8 rows plus the 3-line preamble rewrite); later edits are the documented Phase-7 review repoints, not plan drift |
| T-11.2 | 05-11 | No other row's status/links change; the preamble mentions ADR 0012 | ✓ VERIFIED | The same diff's only non-row change is the preamble sentence adding ADR 0012 while keeping the 14.1-14.11 mapping; every other row untouched in that commit |
| T-11.3 | 05-11 | Every changed row cites a current code path/test/ADR; nothing says TBD/future/unchecked | ✓ VERIFIED | At `1f79eef` every cited line resolves to the named symbol (`core.go:1487` = `applyRemote`, `store.go:252` = `Set`, `store.go:318` = `MergeChanged`, `store.go:386` = `join`, `store.go:178` = `RecordsEqual`); no changed row says TBD/future/unchecked. On HEAD the named file/symbol/test citations still resolve; line ranges drift with later-phase insertions (Phase-9 registry-sweep maintenance, not a Phase-5 gap) |
| T-11.4 | 05-11 | Status vocabulary stays the closed enum | ✓ VERIFIED | Column values on HEAD: `implemented` 126, `fixed-in-v1.1` 22, `superseded-by-amendment` 12, `consumer-zone` 4, `accepted-deviation` 2, `deferred-v2` 1 — no other status |
| T-12.1 | 05-12 | Both P-01 rows flip red->green; other five stay green; runner and frozen tests byte-identical | ✓ VERIFIED | `git diff 7851cfc^..7851cfc -- scripts/red-expectations.tsv` is exactly the two-row flip; `git diff 4f81fd7..HEAD` on the runner and both frozen tests is empty; current TSV has 7 green data rows |
| T-12.2 | 05-12 | The full red gate is 7/7 with both mutants killed | ✓ VERIFIED | This round: `SUPKV_RED_TESTS=1 bash scripts/red-runner` -> exit 0, "7 expectation row(s) checked, 0 error(s)", both mutants killed, evidence `20261123T204945Z` |
| T-12.3 | 05-12 | `make lint`, `make test`, darwin build/vet and the bench smoke are green on the committed tree | ✓ VERIFIED | `make lint` exit 0; `GOOS=darwin GOARCH=arm64 go build ./...` and `go vet ./...` exit 0; bench smoke printed B/op, dump-bytes, peak-live-B, p50/p99; `make test` passed every package except `internal/app/repl`'s `TestCore_keyCountCostFlat` wall-clock ratio gate, which passes standalone (0.91 / 1.01) and is the documented, pre-existing Phase-8-owned flake — see Deferred Items |
| T-12.4 | 05-12 | No test, bound or runner line weakened; the only diff outside the phase's code is the TSV flip | ✓ VERIFIED | Frozen runner/tests unchanged vs base; the blocking test's windows unchanged; the mechanical `persist.Encode/Decode` signature edits in `decode_negative_test.go` keep every assertion (read the diff); the 05-12 commit's non-TSV additions are red-runner evidence JSON, not tests or bounds; later review-fix edits are authorized by `05-REVIEW-DISPOSITION.md` |

### Deferred Items

Not actionable Phase-5 gaps; each is explicitly owned by a later milestone phase.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | `make test` is not fully green on HEAD: `TestCore_keyCountCostFlat` fails under full-suite parallel load (DataSync ratio 3.36 > 2.0) and passes standalone (0.91 / 1.01) | Phase 8 | Phase-8 goal "make perf with thresholds"; plans 08-05 (deterministic gates) and 08-11 (full make gates); `deferred-items.md` records the wall-clock ratio flake with Phase 8 as owner |
| 2 | `freeUDPPort` binds `:0`, closes, and returns the port (`cmd/sup-kv/main_test.go:141-150`; same helper shape in `internal/udprepl/smoke_test.go:47-...`), so a parallel sibling can win the pick-to-bind window | Phase 8 | Phase-8 plan 08-06 "Shared port allocator with package bands; every socket package adopts it"; `deferred-items.md` records the pick-to-bind class as pre-existing and unrelated to the Phase-5 plans |

**Note on the freeUDPPort defect (per the verification brief):** it does not bear on any Phase-5 must-have — the Phase-5 truths that use `cmd/sup-kv/main_test.go` are `TestMain_valueMaxNonPositive` / `TestMain_valueMaxDatagram`, which drive the binary over a single socket and passed this round (full uncached package run, 6.2 s). It is reported as a note, not a score deduction.

### Advisory (New Scope, Unevidenced)

None. This regeneration formed no unevidenced Step-7 blocker. The registry line-number drift noted under T-11.3 is an informational maintenance observation, not a Phase-5 blocker.

## Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/app/storage/store.go` | `ValueMaxBytes`, `Config.ValueLimit`, `Store.ValueMaxBytes`, `RecordsEqual`, `MergeChanged`, `join` | ✓ VERIFIED | All present at `store.go:21,141,161,194,349,417`; single production literal |
| `internal/app/persist/codec.go` | config-driven `Encode`/`Decode`, `EncodeTo`, census, four-class stats, format-bound reader | ✓ VERIFIED | `EncodeTo` at 140, `Encode` wrapper at 177, `Decode` at 258, `DecodeStats` at 194, `MaxDumpLineBytes` at 65; no `json.Decoder` |
| `internal/app/persist/engine.go` | composition + public write/read/`Flush`/`LoadStats`; `Config` with `ConfirmTimeout`/`Now`/`After`/`OnDumpFailure` | ✓ VERIFIED | 300+ lines; five roles composed in `New` |
| `internal/app/persist/loader.go` | `Open`, `mergeRecords`, `LoadStats` construction, post-load dirty | ✓ VERIFIED | `Open` decodes with `ValueLimit()` before the store; frozen `(*Engine, int)` signature |
| `internal/app/persist/facade.go` | thin forwards + `ApplyRemote` | ✓ VERIFIED | `ApplyRemote` dirties only on `changed` |
| `internal/app/persist/scheduler.go` | `Run`, `runDump`, `rearm`, `recordDump`, `dumpStats`, ladder | ✓ VERIFIED | Streaming dump, ladder 100 ms..5 s, class counters, one `OnDumpFailure` per attempt |
| `internal/app/persist/confirm.go` | `waiter`, `await`, `confirmWaiters`, `failWaiters`, `sweepClosed` | ✓ VERIFIED | Single-claim protocol and timeout waiter removal |
| `internal/app/persist/snapshot.go` | `Snapshot`, `Restore`, replay helpers | ✓ VERIFIED | Bounded flush through the public seam; partial list on timeout |
| `internal/app/persist/errors.go` | `ErrUnconfirmed` | ✓ VERIFIED | Plus `ErrDumpFailed`, `ErrClosed`, `ErrSnapshotIncomplete` (Phase-6 additive) |
| `internal/app/persist/flush.go` | shared `flushRequest` type | ✓ VERIFIED | Snapshot vs explicit origin |
| `internal/app/persist/encode_stream_internal_test.go` | `TestEncodeTo_byteIdentical`, `TestEncodeTo_maxSingleWrite` | ✓ VERIFIED (renamed path) | Plan named `encode_stream_test.go`; the strict `testpackage` lint mandates `_internal_test.go` (documented deviation in 05-04-SUMMARY, "Auto-fixed Issues" 1). Both named tests exist and pass; `verify.artifacts` flags only the path name (5/6) |
| `cmd/sup-kv/shutdown_internal_test.go` | `TestAwaitEngineStop` covering clean-cancel, abnormal-error and wedged-expiry branches | ✓ VERIFIED | Three subtests PASS under `-race`; expiry asserts return >= grace, <= 2 s, WARN text |
| `internal/port/dump.go` | `SaveStream`, `Load(...) LoadResult`, outcome types, `FailureClass`/`ClassedError` | ✓ VERIFIED | Full contract at `port/dump.go:28-157` |
| `internal/persistfile/dumpfile.go` | outcomes, preservation, sweep, marker, `SaveStream`, ordered save | ✓ VERIFIED | `Load` main->`.prev`; `preserve`; `sweepOrphans`; `saveAtomically` |
| `internal/persistfile/fsops.go` | five-step seam with buffered fill; parent sync; dir-fsync tolerance | ✓ VERIFIED | 64 KiB `bufio`; cleanup on failure; tolerance only on the dir step |
| `internal/persistfile/saveerror.go` | errno -> class mapping preserving `errors.Is` | ✓ VERIFIED | ENOSPC/EACCES/fs.ErrPermission |
| `internal/persistfile/snapshot.go` | validated names, snapshot copy under the save mutex | ✓ VERIFIED | Charset `[A-Za-z0-9._-]` (rejects `~`) |
| `cmd/sup-kv/main.go` (+ `internal/config/*`) | flag default from the constant, range/ceiling checks, load logs, skip WARN, `OnDumpFailure`, bounded shutdown | ✓ VERIFIED | `config.go:235`; `validate.go:152-154,208-224`; `main.go:63-65,432-463,481-579,869-925`; the Phase-7 extraction kept the value-limit and ceiling messages byte-identical |
| `internal/metrics/metrics.go` | four new families with label enums | ✓ VERIFIED | `loadFamilies`, `discardClassFamilies`, `supkv_dumps_failures_total` (disk_full/permission/other), `supkv_dumps_unconfirmed_total` |
| `internal/app/repl/core.go` | content-aware equal-version branch | ✓ VERIFIED | Consults `storage.RecordsEqual`, re-reads the stored record for the reply verdict (`core.go:1644-1705`) |
| `internal/app/internal/simnet/simnet.go` | `Cluster.NotifyInstalled` | ✓ VERIFIED | Lookup + NotifyLocalWrite + route + write-log capture (`simnet.go:850`) |
| `docs/adr/0012-equal-version-tiebreak.md` | recorded decision | ✓ VERIFIED | Full slim-MADR |
| `docs/OPERATIONS.md` | limits, load outcomes, memory model, monitoring, lifecycle | ✓ VERIFIED | All mandated sections; trust model first |
| `docs/SPEC-DEVIATIONS.md` | eight plan rows + review-fix rows | ✓ VERIFIED | Row-set reconciliation above |
| `scripts/red-expectations.tsv` | two P-01 rows green, five others green | ✓ VERIFIED | 7 data rows, all green |

## Key Link Verification

The `verify.key-links` heuristic reports `all_verified: false` for eleven of the twelve plans (only 05-11, whose links are doc-internal text matches, reports 3/3). Its matcher looks for the target *filename* in the source file, while these Go links are package imports or prose `via` clauses; the manual evidence below is authoritative. All 32 plan-declared links were re-checked on the current tree.

| # | From | To | Via (abridged) | Status | Details |
| - | ---- | -- | -------------- | ------ | ------- |
| 1 | `persist/loader.go` (was engine.go) | `storage/store.go` | `Open` decodes through `Config.ValueLimit()` before the store; `NewStore` normalizes through the same helper | WIRED | `loader.go:47`; `store.go:141-156` |
| 2 | `persist/codec.go` | `storage/store.go` | callers pass the limit; oversize encode is the `storage.ErrValueTooLarge` tripwire | WIRED | `codec.go:140-175`; `store.go:21` |
| 3 | `cmd/sup-kv/main.go` -> `internal/config` | `storage/store.go` | flag default and ceiling effective value are `storage.ValueMaxBytes` | WIRED | `config.go:235`; `validate.go:208-224`; CLI `--help` default 61440 |
| 4 | `persist/codec.go` | `persist/loader.go` | decode closure stores stats; `Open` returns `stats.Total()` | WIRED | `loader.go:49-56,110` |
| 5 | `persist/codec.go` | `persistfile/dumpfile.go` | parse oracle is `Decode` at the configured limit; generation faults advance the chain | WIRED | `loader.go:51`; `dumpfile.go:127-193` |
| 6 | `storage/store.go` | `repl/core.go` | `RecordsEqual` is the one content comparison; core Acks equal version only after consulting it | WIRED | `core.go:1647`; `store.go:194-196` |
| 7 | `repl/core.go` | `persist/facade.go` | tie-break install reaches `ApplyRemote`; core re-reads the stored record for Ack vs reply | WIRED | `core.go:1653,1675-1676`; `facade.go:40-50` |
| 8 | `persist/facade.go` | `storage/store.go` | `MergeChanged` returns the installation decision; `Merge` stays the 3-result wrapper | WIRED | `facade.go:41`; `store.go:349-371` |
| 9 | `persist/scheduler.go` | `persistfile/dumpfile.go` | engine fill runs on the dumper goroutine through the adapter's buffered temp writer; fill error returns via the port | WIRED | `scheduler.go:229-231`; `fsops.go:96-117` |
| 10 | `port/dump.go` | `persist/loader.go` | `Load` returns the outcome/causes; `Open` records them; signature unchanged | WIRED | `port/dump.go:42,98-110`; `loader.go:60,95-107` |
| 11 | `persistfile/fsops.go` | `persistfile/fsops_test.go` | recording fake records the `write` step with its fill and delegates to the real implementation | WIRED | `fsops_test.go:69-72`; `TestFsops_saveOperationOrder` |
| 12 | `persistfile/dumpfile.go` | `persist/loader.go` | the engine's parse oracle decides a candidate's fate; the adapter preserves and reports | WIRED | `dumpfile.go:150-160`; `loader.go:49-56` |
| 13 | `persistfile/dumpfile.go` | `persistfile/fsops.go` | preservation uses `Lstat`/`Rename` directly; the save path's parent sync goes through the seam | WIRED | `dumpfile.go:206-241,319-334`; `fsops.go:175-182` |
| 14 | `persistfile/saveerror.go` | `port/dump.go` | errno knowledge stays in the adapter; the core receives `port.FailureClassOf(err)` | WIRED | `saveerror.go:32-45`; `port/dump.go:149-157` |
| 15 | `persist/loader.go` | `port/dump.go` | `Open` maps `LoadResult` into the engine's `LoadStats` | WIRED | `loader.go:95-107`; `engine.go:147-167` |
| 16 | `cmd/sup-kv/main.go` | `persist/engine.go` | `startEngine` reads `LoadStats()` once after open and decides one WARN/ERROR/INFO | WIRED | `main.go:449-457,481-579` |
| 17 | `metrics/metrics.go` | `persist/engine.go` | `Sources.LoadStats` is wired to `engine.LoadStats`; families render at zero | WIRED | `main.go:724-752`; `metrics.go:191,234-283` |
| 18 | `persist/engine.go` | `simnet.go` | tests bind `Config.Now`/`After` to virtual time; the engine shares `Cluster.Store(0)` and announces via `NotifyInstalled` | WIRED | `sim_dump_test.go`; `simnet.go:850`; `TestScenario_dumpNotify` green |
| 19 | `persist/scheduler.go` | `port/dump.go` | `FailureClassOf` extracts the adapter class; the core never imports `syscall` | WIRED | `scheduler.go:260`; `port/dump.go:112-157` |
| 20 | `cmd/sup-kv/main.go` | `persist/engine.go` | main wires `OnDumpFailure` to one `slog.Error` with the class | WIRED | `main.go:423-430,447` |
| 21 | `persist/engine.go` | `grpcapi/server.go` | RPC failure branches notify for either unconfirmed class; `mapError` is total over it | WIRED | `server.go:218-221,266-269,496-513` |
| 22 | `persist/engine.go` | `cmd/sup-kv/main.go` | `FlushTimeout(shutdownGrace)` and the bounded `engineDone` wait | WIRED | `main.go:901-907`; both bounds behaviorally tested |
| 23 | `persist/engine.go` | `persist/confirm.go` | engine composes one confirmer; writes register/wait through it; the scheduler resolves | WIRED | `engine.go:173-189`; `confirm.go:98-136,207-290` |
| 24 | `persist/scheduler.go` | `persist/confirm.go` | capture-before-encode and the single-claim resolution cross the two roles | WIRED | `scheduler.go:229-252`; `confirm.go:187-290` |
| 25 | `persist/snapshot.go` | `persist/engine.go` | `Snapshot` forces the flush through the public method; `Restore` reuses the facade | WIRED | `snapshot.go:36-42,110-145` |
| 26 | `docs/OPERATIONS.md` | `cmd/sup-kv/operations_test.go` | each mandated statement has an anchor literal | WIRED | `operations_test.go:52,80-118`; `TestOperationsDoc` green |
| 27 | `docs/OPERATIONS.md` | `internal/metrics/metrics.go` | every family in the monitoring list exists in the exporter | WIRED | names grep-present in `metrics.go` |
| 28 | `docs/SPEC-DEVIATIONS.md` | `storage/store.go` | rows cite `ValueMaxBytes`/`ValueLimit`/`Store.ValueMaxBytes`/`MergeChanged` | WIRED | Rows 38/39/42/232; symbols present |
| 29 | `docs/SPEC-DEVIATIONS.md` | `persistfile/dumpfile.go` | `S-PERS-08`/`S-PERS-09` cite the load/preserve path and the load metric | WIRED | Rows 166/167; code present |
| 30 | `docs/SPEC-DEVIATIONS.md` | `docs/adr/0012-...md` | `S-ALG-REP-03` and `S-HLC-07` link the ADR | WIRED | Rows 55/127 |
| 31 | `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go` | flipped row names the frozen stand test | WIRED | Red runner row `TestRed_P01_valueLimit` OK |
| 32 | `scripts/red-expectations.tsv` | `internal/persistfile/red_value_limit_test.go` | flipped row names the frozen engine test | WIRED | Red runner row `TestRed_P01_valueLimitEngine` OK |

## Data-Flow Trace (Level 4)

No rendered values in this phase; the equivalent is the dump/load/metrics data path. Every traced value ends in a real store, a real file or a real counter.

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Dump cycle (`scheduler.runDump`) | `records` | `store.Records()` deep copy of the live map | Yes — the 10 000-key bench wrote 5 147 571 dump bytes this round | ✓ FLOWING |
| Dump encode (`EncodeTo`) | line bytes | sorted real records + `RecordChecksum` | Yes — `TestEncodeTo_byteIdentical` + bench | ✓ FLOWING |
| Load (`persistfile.Load` -> `Decode`) | `LoadResult.Dump` / records | real `os.ReadFile` of main/`.prev` | Yes — restart tests (`TestEngine_endToEndRestart`, frozen P-01) | ✓ FLOWING |
| Load report (`LoadStats`) | outcome/failures/decoded/sweep | adapter `LoadResult` -> `Open` | Yes — the seven stand process tests read the emitted log lines | ✓ FLOWING |
| Load/dump metrics | `LoadStats`, `DumpStats` | `engine.LoadStats`/`DumpStats` seams | Yes — `TestMetrics_values` by-effect samples; stand scrape | ✓ FLOWING |
| Tie-break reply (`applyRemote`) | stored record | `store.Lookup` **after** the merge | Yes — `TestReplication_equalVersionContentLaw` | ✓ FLOWING |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Whole-tree build | `go build ./...` | exit 0 | ✓ PASS |
| Whole-tree vet | `go vet ./...` | exit 0 | ✓ PASS |
| Strict lint | `make lint` | `config verify` + `golangci-lint run ./...` exit 0 | ✓ PASS |
| Full race suite | `make test` (`go test -race -cover ./...`) | all packages ok except `internal/app/repl`: `TestCore_keyCountCostFlat` ratio 3.36 (documented Phase-8 wall-clock flake; see Deferred Items) | ⚠️ DEFERRED (unrelated later-phase gate) |
| That test standalone | `go test -race -count=1 ./internal/app/repl/ -run '^TestCore_keyCountCostFlat$' -v` | PASS; ratios 0.91 / 1.01 | ✓ PASS |
| Darwin cell | `GOOS=darwin GOARCH=arm64 go build ./... && go vet ./...` | exit 0 both | ✓ PASS |
| Red gate (current tree) | `SUPKV_RED_TESTS=1 bash scripts/red-runner` | exit 0; 7/7 rows OK; both mutants killed; evidence `20261123T204945Z` | ✓ PASS |
| Bounded engine stop (all branches) | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAwaitEngineStop$' -v` | 3/3 subtests PASS (0.03 s) | ✓ PASS |
| Phase core packages, uncached | `go test -race -count=1 ./internal/app/persist/ ./internal/persistfile/ ./internal/app/storage/ ./internal/metrics/` | all `ok` | ✓ PASS |
| Full `cmd/sup-kv` package, uncached | `go test -race -count=1 ./cmd/sup-kv/` | ok, 6.2 s | ✓ PASS |
| Value-limit CLI boundary | `TestMain_valueMaxNonPositive`, `TestMain_valueMaxDatagram` | both PASS (0/negative exit 2; 64 000 and 64 408 serve; 64 409 exit 2) | ✓ PASS |
| Direct binary probe | built `./cmd/sup-kv` with `--value-max-bytes=0` / `64409` | exit 2 both; flag named; `65507-byte UDP ceiling` text | ✓ PASS |
| Tie-break + simulator | `go test -race -count=1 ./internal/app/repl/ -run 'TestScenario_dumpNotify|TestScenario_tiebreakConverges|TestReplication_equalVersion'` | all PASS | ✓ PASS |
| Process load outcomes | `go test -race -count=1 ./internal/stand/ -run TestStand_persist -v` | 7/7 PASS, 0 skips | ✓ PASS |
| Unconfirmed RPC mapping | `go test -race -count=1 ./internal/grpcapi/ -run 'TestServer_unconfirmed'` | PASS | ✓ PASS |
| Dump bench smoke | `go test -run '^$' -bench BenchmarkDump10kKeys -benchtime=1x ./internal/app/persist/` | `bytes/op 5147571`, `dump-bytes 5147571`, `peak-live-B 11036512`, p50/p99 printed | ✓ PASS |
| Single production literal | `grep -rn 61440 --include=*.go . \| grep -v _test.go` | only `internal/app/storage/store.go:21` (other hits live in the module cache) | ✓ PASS |
| No streaming decoder poison | `grep -c json.NewDecoder internal/app/persist/codec.go` | 0 | ✓ PASS |
| No wall clock in persist | `grep -E 'time\.(Now\|NewTimer\|After\|Until\|Since)'` non-test | only `engine.go:77,80` | ✓ PASS |
| No interfaces in persist | `grep 'type .* interface' internal/app/persist/*.go` | none | ✓ PASS |
| Debt markers in phase files | `grep -n -E 'TBD\|FIXME\|XXX'` over the phase's production files | none | ✓ PASS |

## Probe Execution

No `scripts/*/tests/probe-*.sh` exists and no plan declares a probe script. The phase's declared acceptance runner is the red gate, re-executed this round on the current tree (`scripts/red-runner`, 7/7 rows + mutant stage green). Latest evidence: `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261123T204945Z/`.

## Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
| --------- | ---------- | ------ | ------- | -------- | --------------- | ------- |
| `internal/persistfile/red_value_limit_test.go` | PERSIST-01 | 1 | 0 (env-gated; the runner opened it) | No | Behavioral (write -> restart -> full value) | PASS |
| `internal/stand/stand_red_test.go` (P-01) | PERSIST-01 | 2 subtests | 0 (env-gated) | No | Behavioral (process SIGKILL/restart) | PASS |
| `internal/app/persist/engine_test.go` | PERSIST-01/04/07/10 | all | 0 | No | Value + behavioral | PASS |
| `internal/app/persist/codec_test.go` | PERSIST-03/10 | all | 0 | No | Value (exact classes/bytes) | PASS |
| `internal/app/persist/timeout_test.go` | PERSIST-08 | all | 0 | No | Behavioral (exact virtual instants) | PASS |
| `internal/app/persist/backoff_test.go` | PERSIST-05/08 | all | 0 | No | Behavioral (exact attempt instants) | PASS |
| `cmd/sup-kv/shutdown_internal_test.go` | PERSIST-08 | 1 (3 subtests) | 0 | No | Behavioral (time bound at the grace + WARN text) | PASS |
| `internal/persistfile/load_outcome_test.go` | PERSIST-02 | all | 1 root-only (ran as non-root) | No | Behavioral | PASS |
| `internal/persistfile/sweep_test.go` | PERSIST-09 | all | 1 root-only (ran as non-root) | No | Behavioral | PASS |
| `internal/stand/persist_test.go` | PERSIST-02/05/10 | 7 | 1 root-only (ran as non-root) | No | Behavioral (real binary/gRPC/logs/metrics) | PASS |
| `internal/app/repl/sim_dump_test.go` | PERSIST-05/07/12 | 2 | 0 | No | Behavioral (same-seed traces) | PASS |
| `internal/metrics/metrics_test.go` | PERSIST-02/05/08/10 | all | 0 | No | Value (by-effect samples) | PASS |

**Disabled tests on requirements:** 0 (the env-gated red tests are the designed mechanism and ran through the runner). **Circular patterns:** 0 — fixtures carry literal records/checksums; no test generates expected values by running the system under test. **Assertion strength:** `TestAwaitEngineStop`'s expiry subtest asserts a lower time bound (>= grace), an upper bound (<= 2 s) and the exact WARN substring, so a removed timeout, an immediate return or a dropped WARN each fail. **Insufficient assertions:** 0.

## Requirements Coverage

| Requirement | Source Plan(s) | Status | Evidence |
| ----------- | -------------- | ------ | -------- |
| PERSIST-01 | 01, 11, 12 | ✓ SATISFIED | Single source + config path; exact-limit restart at 64 000 and 8; invalid exit 2; frozen tests green; CLI probes |
| PERSIST-02 | 05, 06, 10, 11, 12 | ✓ SATISFIED | Four outcomes, preservation, one WARN/one ERROR, nonzero unusable metric; docs + stand tests |
| PERSIST-03 | 02, 12 | ✓ SATISFIED | `record_count` census, torn-tail fault, per-record isolation, four classes |
| PERSIST-04 | 06, 12 | ✓ SATISFIED | Dirty after Prompt/Backup/Unusable, snapshot decodes back, `.prev` survives first rotation, Main clean |
| PERSIST-05 | 07, 10, 12 | ✓ SATISFIED | 100 ms -> 5 s ladder, one ERROR + class per attempt, health SERVING, docs; process EACCES test |
| PERSIST-06 | 04, 10, 12 | ✓ SATISFIED | Streaming `EncodeTo`/`SaveStream`, max-single-write bound, measured memory model + documented residual |
| PERSIST-07 | 03, 11, 12 | ✓ SATISFIED | Deterministic `(version, content)` law in store and core, ADR 0012, same-seed replay + convergence |
| PERSIST-08 | 08, 12 | ✓ SATISFIED | Writes/snapshots/`FlushTimeout` bounds tested; `mapError`/notify tested; the `engineDone` grace branch is exercised by `TestAwaitEngineStop` |
| PERSIST-09 | 05, 12 | ✓ SATISFIED | Marker-only age-bounded sweep, parent sync on create, ENOTSUP/EINVAL dir-sync tolerance, `fsync_darwin.go` absent |
| PERSIST-10 | 01, 02, 06, 12 | ✓ SATISFIED | `ValueLimit` class, skip count + one WARN + metric, evidence window through two rotations |
| PERSIST-11 | 04, 09, 12 | ✓ SATISFIED | Five roles, public API stable, port returns outcome/cause, no new interface |
| PERSIST-12 | 07, 12 | ✓ SATISFIED | `Now`/`After` seam; no wall clock in the package; simulator drives dump<->notify and the tie-break |

No ORPHANED requirements: all twelve PERSIST IDs are mapped to Phase 5 in `.planning/REQUIREMENTS.md` and claimed by at least one plan (`05-01`, `05-02`, `05-03`, `05-04`, `05-05`, `05-06`, `05-07`, `05-08`, `05-09`, `05-10`, `05-11`, `05-12`).

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX` in any file modified by this phase | — | None |
| — | — | No `TODO`/`HACK`/`PLACEHOLDER`/stub markers in the phase's production files | — | None |
| — | — | No empty implementations or bare `{}` function bodies in the phase's production files | — | None |
| — | — | No raw `fmt.Println`/`log.Println` on a production path; remaining `log.*` calls are the documented slog-set-default bridge | — | None |
| — | — | Every `//nolint` names its linter and a human-readable reason; `.golangci.yml`'s only changes since Phase 5 are the documented additive `depguard` rules with written justification | — | None |
| `internal/app/persist/bench_test.go` | 93-98, 122 | `peak-live-B` samples `HeapAlloc`, which includes not-yet-collected line garbage; the 11.0 MB sample is ~2.1x the 5.15 MB state | ℹ️ Info | The documented model (one copy + one line + 64 KiB) and the 5.15 MB `bytes/op` (one state size) meet PERSIST-06; the sampled figure is a high-water mark, not live reachable bytes. No action for this phase |

`math/rand` appears in the replication run-ID and test-harness port picking only — never in the tie-break, which is `bytes.Compare` over explicit fields.

## Decision Coverage

All 12 trackable CONTEXT.md decisions (D-01…D-12) are honored by shipped artifacts (`check.decision-coverage-verify`: total 12, honored 12, not_honored []).

## Human Verification Required

None. This is an infrastructure/foundation phase with no user-facing element, every truth is verified programmatically, and the one behaviour-dependent truth (bounded shutdown) is exercised by the named test `TestAwaitEngineStop`, which drives the same helper `shutdown()` calls.

## Gaps Summary

No gaps. All 12 requirements have shipped implementations wired end to end, all five roadmap success criteria are met, and every plan truth has current-tree evidence. This pass was a freshness regeneration: the prior report's 64/64 was re-derived against a tree that phases 6-8 had edited, not trusted. Two unrelated test-harness flaws are recorded as deferred to Phase 8 — the `TestCore_keyCountCostFlat` wall-clock ratio gate (passes standalone) and the `freeUDPPort` pick-to-bind window — and neither bears on a Phase-5 must-have. The `encode_stream_internal_test.go` filename and the later registry line-number drift are documented deviations/maintenance notes, not drift.

---

_Verified: 2026-11-23T20:57:31Z_
_Verifier: Claude (gsd-verifier)_
