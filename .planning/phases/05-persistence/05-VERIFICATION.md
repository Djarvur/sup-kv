---
phase: 05-persistence
verified: 2026-11-27T17:21:33Z
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
covered_digest: "v4:sha256:c4a0d6bf61527864c9e8449dd9c0ca9423f849b5f7392cab051737957bc54013"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 64/64
  gaps_closed:
    - "`make test` fully green on the current tree — the prior `TestCore_keyCountCostFlat` wall-clock ratio flake was replaced by Phase 8 plan 08-05's deterministic operation-count gate (`internal/app/repl/bench_test.go:210-219` 'deterministic acceptance'); this round's full `make test` (race, cover) passed every package including `internal/app/repl` (46.9 s)"
    - "Test-harness port picking race-free — Phase 8 plan 08-06's shared claim-while-held allocator (074ccd0, adoption 5c7a3d7/3e7b465, claim-before-probe 7aeec35, probe retry 9cdc0f1) replaced the `freeUDPPort` bind-close-return; `make test` green including the socket packages"
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "The red gate's `TestRed_R01_pacedSync20k` row is green locally"
    addressed_in: "Phase 3 acceptance (scheduled CI job `red-r01-x20`)"
    evidence: "The row pins Phase 3's XPORT-01 pacing contract, not a Phase-5 must-have. It misses only on this powersave-throttled host while converging (7896/8000 keys at the 20.6 s documented bound) — the class adjudicated by the Phase-3 halt record (03-12), the Phase-4 verification (2026-11-27, tree `ef370a0`, evidence dirs `20261127T161311Z`/`20261127T161629Z`/`20261127T165618Z`) and Phase 3's own 2026-11-27 verification. Green state held by full-speed-host flip evidence `20261121T225048Z` (commit `4a83919`) and by the 2026-11-23/24 local runs on identical production sources; 20/20 acceptance assigned to the scheduled `red-r01-x20` job (`.github/workflows/scheduled.yml:149-183`). This round's runs (`20261127T171347Z`, `20261127T171623Z`) check 6/7 rows OK — both P-01 rows green — and the mutation stage is green standalone."
advisory: []
behavior_unverified_items: []
coincidental_reliance_items: []
human_verification: []
---

# Phase 5: Persistence Verification Report

**Phase Goal:** The dump path is honest about its limits and failures — one enforced value limit, visible load outcomes, generation integrity, bounded memory and blocking — so "a confirmed write is on disk" is a tested guarantee rather than a convention.
**Verified:** 2026-11-27T17:21:33Z
**Status:** passed
**Re-verification:** Yes — stale-fingerprint regeneration, not a gap-closure round. The prior report (regenerated at `bc71ece`) covered the tree as of the phase-6/7/8 mid-stream state; phases 6-8 have since edited covered files (`internal/app/persist/snapshot.go` gained the full-rollback Restore and `ErrSnapshotIncomplete`, `internal/app/storage/store.go` gained `LiveVersions` and the generation-bearing `OnChange`, `cmd/sup-kv/main.go` was decomposed into `internal/config`, `internal/udprepl` was split, the gRPC API moved to `api/supkvapi/v1`, Phase 8 rewrote the test harness and the flaky gates). Everything below was re-derived from the current tree (`bd88889`, implementation sources unchanged since `ef370a0`); the prior report's claims were used only as a checklist.

## Re-verification Basis

The prior 64/64 report was treated as a hypothesis and re-tested against the current tree. The later phases' edits in the covered files are additive or out-of-scope in the areas this phase owns:

- `internal/app/persist`: `Config.OnConfirmed` + its publication after `confirmWaiters` (`scheduler.go:245-247`, APIS-04 watermark), `ErrSnapshotIncomplete` + the partial-snapshot refusal (`errors.go`, `snapshot.go`), the Restore full-rollback (decision 14.1, ADR 0001), the exported `MaxDumpLineBytes` constant with the `maxDumpLineBytes()` wrapper preserved, and the `EncodeTo` limit-domain guard (the review's IN-04, delivered by Phase 7 as planned). The five roles, the confirmation protocol, the capture-before-encode order (`scheduler.go:229-231`), the retry ladder and the flush/timeout bounds are unchanged; `confirm.go`, `flush.go`, `backoff_test.go`, `bench_test.go` (persist), `encode_stream_internal_test.go`, `engine_blocking_test.go`, `timeout_*_test.go`, `load_stats*_test.go`, `fuzz_dump_test.go` are byte-identical to the phase-5 completion state.
- `internal/app/storage/store.go`: the `OnChange` tap gained a generation parameter (APIS-04); `join` (`store.go:417`), `RecordsEqual`, `MergeChanged` (`store.go:349`) and `Merge` are semantically unchanged; `LiveVersions` is additive (APIS-02).
- `cmd/sup-kv/main.go`: flag parsing/validation live in `internal/config` (additive Phase-7 decomposition); the value-limit and datagram-ceiling messages are byte-identical (`internal/config/validate.go:222`), `exitUsage` is still 2, and `shutdown`/`awaitEngineStop` are unchanged in substance. Commit `2b281a9` (test-only, postdates 05-REVIEW.md) extracted `awaitEngineStop` (`main.go:916-925`) from `shutdown` as the testable seam — the select, the WARN text and the grace bound are identical to the reviewed code — and added the wedged-dumper subtest; all three subtests pass under `-race` this round.
- `internal/persistfile/snapshot.go`: `CreateSnapshot` now classifies filesystem errors through `classifySaveError` (D-03) — additive wrapping; `ErrNoDump` passes through; name validation and the copy-under-mutex are unchanged. `dumpfile.go`, `fsops.go`, `saveerror.go` are unchanged.
- `internal/stand/persist_test.go`: one import moved from the internal `apiproto` to the public `api/supkvapi/v1` package (Phase-7 API move); no assertion changed.
- `.golangci.yml`: additive `depguard` rules only, with the documented written justification (AGENTS.md deviation 1); `make lint` is green.

No phase-05 truth regressed. The one currently-failing red-runner row (`TestRed_R01_pacedSync20k`) is Phase 3's contract on this throttled host — recorded under **Deferred Items**.

## Goal Achievement

### Roadmap Success Criteria

| # | Criterion | Status | Evidence |
| - | --------- | ------ | -------- |
| 1 | Value limit defined once and enforced everywhere (P-01 turns green): exact `value_max_bytes` survives restart at any accepted flag value; invalid value exits 2; the 61 440 constant exists once | ✓ VERIFIED | Production `grep 61440 --include=*.go` finds only `internal/app/storage/store.go:21`; `Config.ValueLimit()` (`store.go:141-147`) is used by `Open` before the store exists (`loader.go:48`) and by `NewStore`; `Store.ValueMaxBytes()` (`store.go:161-163`) feeds the encoder (`scheduler.go:82`) and snapshot decode (`snapshot.go:283` doc); the CLI default is the constant (`internal/config/config.go:235`); 0/-5 exit 2 naming the flag, 6409 starts and serves, 64409 exits 2 with the ceiling text (direct binary probe this round); frozen P-01 tests green through the red runner and directly |
| 2 | Dump-load outcomes visible, never destroy evidence: `.prev` -> one WARN with path+cause; both unusable -> ERROR + nonzero metric; absent -> neither; preserved under a separate name; format carries a record count so line-boundary truncation is incomplete; one corrupted record discards only that record | ✓ VERIFIED | `persistfile.Load` four outcomes (`dumpfile.go:127-193`), `preserve` collision-probes before rename (`dumpfile.go:206-241`), header `record_count` (`codec.go:102-105`, census `codec.go:331-344`), per-record classes (`codec.go:349-384`), one WARN/one ERROR in `cmd` (`main.go:481-579`), metrics families (`metrics.go:234-283`); all seven stand process tests and the load-outcome table pass |
| 3 | Failure handling bounded and honest: dirty after backup/empty start (first rotation never displaces the last good generation; Snapshot right after decodes back); one ERROR per failure with cause, growing pause to a cap, health documented; skipped records counted/logged | ✓ VERIFIED | `startLoadDirty` (`loader.go:113-120`), `TestEngine_startDirty` (Prompt/Backup/Unusable dirty; Main-with-discard clean; snapshot decodes back); ladder 100 ms..5 s (`scheduler.go:19-20,282-291`), `OnDumpFailure` once per attempt (`scheduler.go:259-278`), `TestEngine_backoffLadder`, `TestStand_persistDumpFailure` (EACCES: ERROR+class, SERVING, bounded rate, recovery); skip accounting (`main.go:451-455`, `metrics.go:263-283`), `TestEngine_skipEvidenceWindow` |
| 4 | Resources bounded: streaming encode at most two state sizes with the memory model documented; a hung fsync cannot block writes, snapshots or shutdown indefinitely; orphan temps collected; parent synced; `fsync_darwin.go` gone | ✓ VERIFIED | Streaming path: one `Records()` + `SaveStream(EncodeTo)` (`scheduler.go:228-252`); `TestEncodeTo_maxSingleWrite`, bench printed `bytes/op 5 147 571`, `dump-bytes 5 147 571`, `peak-live-B 10 857 392`, p50/p99; model documented (`OPERATIONS.md` `## Memory model`); writes/snapshots bounded (`TestEngine_confirmationTimeout`, `TestFlush_timeout`, `TestSnapshot_timeout`); shutdown bounded by `FlushTimeout(shutdownGrace)` and `awaitEngineStop`, whose clean/abnormal/wedged branches `TestAwaitEngineStop` exercises; sweep/parent-sync/dir-fsync (`TestSweep_*`, `TestEnsureDir_*`, `TestSyncDirTolerant`); no `fsync_darwin.go` in `internal/persistfile/` |
| 5 | Structure serves seams: `persist.Engine` split into five roles with the `DumpStore` port returning outcome and failure cause; timers/`time.Now()` behind the repl-style seam so the simulator exercises dump↔replication; equal-version different content resolved by a recorded ADR, implemented and tested | ✓ VERIFIED | Five role files (`loader.go`, `facade.go`, `scheduler.go`, `confirm.go`, `snapshot.go`) + composition `engine.go`; `port.DumpStore.Load(now, parse) LoadResult` with `LoadOutcome`/`Generation`/`GenerationFailure` (`port/dump.go:42-110`); only wall-clock references are the two defaults in `normalized()` (`engine.go:77,80`); `TestScenario_dumpNotify`, `TestScenario_tiebreakConverges`, `TestEngine_injectedClock` ran green; ADR 0012 records the deterministic `(version, content)` law; store and core wired (`core.go:1647,1676`) |

### Observable Truths (plan must-haves)

**Score:** 64/64 truths verified, 0 present-behaviour-unverified.

| # | Plan | Truth | Status | Evidence |
| - | ---- | ----- | ------ | -------- |
| T-01.1 | 05-01 | 61 440 exists exactly once in production; store default, pre-store decode and encoder all derive through `Config.ValueLimit()`/`Store.ValueMaxBytes()` | ✓ VERIFIED | `store.go:21` is the only production literal; `loader.go:48` decodes through `ValueLimit()`; `store.go:141-163`; the codec takes the limit as an argument (`codec.go:143-147,261`); `internal/config/config.go:235` sets the flag default from the constant |
| T-01.2 | 05-01 | A record of exactly the configured limit survives restart at two accepted flag values (64 000, 8); one byte over is rejected by the store | ✓ VERIFIED | Frozen `TestRed_P01_valueLimit` (63 000 and exactly 64 000, SIGKILL restart) green this round (4.28 s) and OK in the red-runner evidence `20261127T171623Z`; `TestEngine_exactValueLimitRestart` (8-byte limit + 9-byte reject) green |
| T-01.3 | 05-01 | `--value-max-bytes=0`/negative exit 2 naming the flag; 64 400 and 64 408 start; 64 409 exits 2 with the unchanged ceiling message | ✓ VERIFIED | `TestMain_valueMaxNonPositive`, `TestMain_valueMaxDatagram` pass; direct binary probe this round: 0 and -5 -> exit 2, `value-max-bytes must be positive: --value-max-bytes=0`; 64409 -> exit 2, the unchanged `over the 65507-byte UDP ceiling` text (`internal/config/validate.go:222`); 6409 with full knobs starts, logs `value_max_bytes:6409` and serves until SIGTERM; `--help` default 61440 |
| T-01.4 | 05-01 | The two frozen P-01 tests pass unedited | ✓ VERIFIED | `git diff 4f81fd7..HEAD` on `internal/persistfile/red_value_limit_test.go` and `internal/stand/stand_red_test.go` is empty; both pass directly and through the runner |
| T-02.1 | 05-02 | The dump announces `record_count`; fewer record lines is a generation-level error; a v1.0 dump without the field takes the legacy path | ✓ VERIFIED | `codec.go:102-105,130`; `checkRecordCount` (`codec.go:331-344`); `TestCodec_headerCensus` green |
| T-02.2 | 05-02 | One syntax-corrupt line costs exactly that record; the rest load; the bad line counts `Malformed`; the `json.Decoder` poison behaviour is gone | ✓ VERIFIED | Per-line `json.Unmarshal` (`codec.go:349-384`); `grep -c json.NewDecoder codec.go` = 0; `TestCodec_corruptLineIsolated` green |
| T-02.3 | 05-02 | `DecodeStats` keeps checksum, version, value-limit and malformed separate, each counted, with `Total()` | ✓ VERIFIED | `codec.go:194-200`; four counters and `Total()`; `TestCodec_discardClasses` in the package run |
| T-02.4 | 05-02 | The line bound is a format property (config-independent); a dump written at the default still reads under a tiny limit; over-limit is `ValueLimit`, never `Malformed`/generation fault; only an impossible line is generation-level | ✓ VERIFIED | `MaxDumpLineBytes` is now an exported format constant taking no configuration (`codec.go:62-66`) with the reader wrapper `maxDumpLineBytes()` (`codec.go:68-69`); bounded `readDumpLine` (`codec.go:299-322`); `TestCodec_shrunkLimitKeepsOtherRecords`; overlong-line row in `TestCodec_headerCensus` |
| T-02.5 | 05-02 | A 1 024-byte quote/control-char key with a 61 440-byte value round-trips | ✓ VERIFIED | `TestCodec_escapedMaxKeyRoundTrip` in the green package run |
| T-02.6 | 05-02 | A v1.1 dump cut mid-line is a torn-tail generation error; a v1.0 line-boundary cut remains the documented legacy limitation | ✓ VERIFIED | `errTornTail` (`codec.go:90`), `errIncompleteDump` (`codec.go:80`), `Decode`'s unterminated-malformed branch (`codec.go:284-286`); `TestCodec_truncatedGenerationUnreadable`; legacy row in the census test |
| T-03.1 | 05-03 | Join order `(version, content)`: greater version wins; identical equal version is a no-op; differing equal version resolves remove-wins then `bytes.Compare` | ✓ VERIFIED | `store.join` (`store.go:417-450`); `TestMerge_equalVersionContentLaw`, `TestMerge_equalVersionBranchTable`, `TestProp_mergeSmallSpace` in the green storage run |
| T-03.2 | 05-03 | The law reaches the wire: identical content Acks with no reply; incoming tie-break win Acks; stored tie-break win answers with one unicast `DataReplication`, no Ack | ✓ VERIFIED | `core.applyRemote` (`core.go:1635-1705`); `TestReplication_equalVersionContentLaw` green (all three branches) |
| T-03.3 | 05-03 | `ApplyRemote` dirties through `MergeChanged`'s `changed`, never a Conflict/version heuristic | ✓ VERIFIED | `facade.go:40-50` (the generation parameter is Phase-6 additive); `TestEngine_applyRemoteEqualVersionContent` in the package run |
| T-03.4 | 05-03 | Explicit same-version/different-content `DataReplication` drives the wire rule; same seed replays identically; cross-node convergence via `NotifyInstalled` | ✓ VERIFIED | `TestReplication_equalVersionSameSeedTrace`; `TestScenario_tiebreakConverges` — both green this round; `Cluster.NotifyInstalled` (`simnet.go:850`) |
| T-03.5 | 05-03 | ADR 0012 records the deterministic deviation, the lost-operation consequence and explicit non-changes | ✓ VERIFIED | `docs/adr/0012-equal-version-tiebreak.md` read: context/decision/consequences/links; names S-HLC-07 and PFXD-01 as non-changes |
| T-04.1 | 05-04 | A dump streams: one `Records()` copy + `SaveStream(fill)` running `EncodeTo`; the five TEST-03 steps are unchanged | ✓ VERIFIED | `scheduler.go:228-252`; `fsops.go:96-117,346-369`; `TestFsops_saveOperationOrder`, `TestFsops_saveOperationOrder_rotation_chain` |
| T-04.2 | 05-04 | `EncodeTo` is byte-identical to `Encode`; no write larger than `maxDumpLineBytes()` | ✓ VERIFIED | `TestEncodeTo_byteIdentical`, `TestEncodeTo_maxSingleWrite` green |
| T-04.3 | 05-04 | A fill error removes the temp and surfaces the error; the failure matrix keeps its hygiene | ✓ VERIFIED | `fileFSOps.write` cleanup (`fsops.go:96-117`); `TestFsops_failureAtEachStep` includes the `fill failure cleans the temp` row; `TestFileStore_saveFailureHygiene` |
| T-04.4 | 05-04 | The port carries `Load(now, parse) LoadResult` with outcome/generation/failures, implemented by the adapter and its doubles | ✓ VERIFIED | `port/dump.go:42,54-110`; adapter `dumpfile.go:127-193`; fakes across the persist and persistfile test suites; whole tree builds and vets |
| T-04.5 | 05-04 | The memory model is measured, not guessed: B/op and a sampled peak live; the one `Records()` copy and writer pause stated as residual | ✓ VERIFIED | `bench_test.go:86-141` (persist; unchanged); this round's run printed `bytes/op 5147571`, `dump-bytes 5147571`, `peak-live-B 10857392`, p50/p99; `OPERATIONS.md` `## Memory model` states the residual writer pause |
| T-05.1 | 05-05 | Load reports Prompt/Main/Backup/Unusable; the accepted-`.prev` path always names the main candidate; evidence is never deleted | ✓ VERIFIED | `dumpfile.go:127-193`; `TestFileStore_loadOutcomeTable` green; stand Prompt/Backup/Backup-absent/Unusable/Unusable-single all passed this round |
| T-05.2 | 05-05 | A generation failing the oracle is renamed `<path>.corrupt-<UTC>[-N]` before any save; only regular files; same-second collisions survive; rotation never overwrites | ✓ VERIFIED | `preserve` (`dumpfile.go:206-241`), `Lstat` regularity, free-name probe; `TestFileStore_loadBackupPreservesMain`, `TestFileStore_preserveCollision`, `TestFileStore_loadNonRegularCandidate` |
| T-05.3 | 05-05 | After a Backup start the first save's rotate is a no-op; `.prev` stays byte-identical | ✓ VERIFIED | `TestFileStore_loadBackupPreservesMain` compares `.prev` bytes before/after `Save` |
| T-05.4 | 05-05 | Every temp carries `.tmp~`, whose `~` is outside the snapshot charset, so no legal snapshot collides and the sweep never keys on a data prefix | ✓ VERIFIED | `orphanTempMarker`/`tempPattern` (`dumpfile.go:43-51`); `CreateTemp(dir, tempPattern)` (`fsops.go:102`); snapshot charset `[A-Za-z0-9._-]` rejects `~` (`snapshot.go:104-127`); `TestFileStore_nameValidation` |
| T-05.5 | 05-05 | The sweep removes only old `.tmp~` regular files (both dirs), keeps young ones, protects main/`.prev`/`.corrupt-*`/snapshots, reports count and first error | ✓ VERIFIED | `sweepOrphans`/`protectedDataName`/`sweepEntry` (`dumpfile.go:252-317`); `TestSweep_removesOnlyOldTemps`, `TestSweep_protectsDataNames`, `TestSweep_reportsUnreadableDir`; stand Prompt test proves the old orphan is removed with count 1 |
| T-05.6 | 05-05 | A newly created dump directory is synced at its parent before the temp write; existing directories do not (five-step order unchanged) | ✓ VERIFIED | `ensureDir` (`dumpfile.go:319-334`); `TestEnsureDir_newDirectorySyncsParent`, `TestEnsureDir_existingDirectoryNoParentSync` green |
| T-05.7 | 05-05 | Directory fsync tolerates ENOTSUP/EINVAL; file-data fsync keeps every error; a save error carries `port.FailureClass` via `errors.Is` | ✓ VERIFIED | `tolerateDirSyncErr` called only from `syncDir` (`fsops.go:175-182,220-228`); `classedSaveError` wraps the cause (`saveerror.go:12-46`); `TestSyncDirTolerant` (ENOTSUP, wrapped EINVAL, EACCES/ENOSPC preserved), `TestClassifySaveError`, failure matrix; Phase-6 extended the same classifier to `CreateSnapshot` (`persistfile/snapshot.go`) without touching the save path's tolerance |
| T-06.1 | 05-06 | `Open` stores a `LoadStats` report and keeps `(*Engine, int)`; `New` engines report zero | ✓ VERIFIED | `loader.go:40-88` (`Open(...) (*Engine, int)`, report assignment at 90+); `engine.go:191-197`; `TestEngine_loadStatsMapsResult` green |
| T-06.2 | 05-06 | Dirty after Prompt/Backup/Unusable (Snapshot right after flushes and decodes back; first rotation never displaces the good generation); Main — even with discards — starts clean | ✓ VERIFIED | `startLoadDirty` (`loader.go:113-120`); `TestEngine_startDirty` green (all five cases with snapshot decode assertions); `TestFileStore_loadBackupPreservesMain` |
| T-06.3 | 05-06 | Every `markDirty()` wakes an idle dumper via the `changed` nudge (incl. the restore mid-error path); one nudge per write path; a clean engine stays clean | ✓ VERIFIED | `markDirty` nudges (`confirm.go:139-149`); `await` nudges once; the replay's mid-error path calls `markDirty` (`snapshot.go`, unchanged in the Phase-6 rewrite); `TestEngine_markDirtyNudge` green |
| T-06.4 | 05-06 | Records skipped by a shrunken limit are counted by class, logged once with count+limit, exposed as a metric, and left on disk through two rotations | ✓ VERIFIED | `stats.Decoded.ValueLimit`; `main.go:451-455`; `metrics.go:263-283`; `TestEngine_skipEvidenceWindow` green (byte-identical after load, in `.prev` after one write, gone after two); `TestStand_persistShrunkLimit` green |
| T-06.5 | 05-06 | Binary logs exactly one WARN for Backup (path+cause incl. absent main), one ERROR for Unusable (one group per failed candidate), nothing (no WARN/ERROR) for Prompt/Main, one INFO for sweep count, one WARN for sweep error | ✓ VERIFIED | `logLoadStats`/`logBackupFallback`/`logUnusableGenerations` (`main.go:481-579`); stand tests assert the exact message counts (Backup=1, Unusable=0/1, sweep INFO count) and all passed this round; the Phase-7 INFO `Dump load` record for Prompt/Main is additive and not a WARN/ERROR |
| T-06.6 | 05-06 | `supkv_dump_load_total{outcome}` and `supkv_dump_records_discarded_total{class}` render every label at zero and the load counts after a load | ✓ VERIFIED | `metrics.go:254,265` (families at `loadFamilies`, `discardClassFamilies`); `TestMetrics_families`, `TestMetrics_values` in the green metrics run |
| T-07.1 | 05-07 | `internal/app/persist` owns no wall clock except the two defaults in `normalized()` | ✓ VERIFIED | `grep` over non-test persist sources finds only `engine.go:77,80` (`c.Now = time.Now`, `c.After = time.After`); every read/arm goes through `cfg.Now`/`cfg.After` |
| T-07.2 | 05-07 | The scheduler arms debounce/deadline/retry from `Config.After`; re-arming replaces the channel; no channel created inside the select | ✓ VERIFIED | `run` loop (`scheduler.go:109-147`), `rearm` (`scheduler.go:176-214`); no `time.After` in the select; `TestEngine_injectedClock`, `TestEngine_confirmDebounceTiming` green |
| T-07.3 | 05-07 | Ladder 100 ms -> 200 -> 400 -> 800 -> 1.6 s -> 3.2 s -> 5 s cap; while pending no debounce/deadline armed; success resets and covers waiters | ✓ VERIFIED | `defaultRetryBase`/`defaultRetryCap` (`scheduler.go:19-20`), `nextRetry`/`resetRetry`; `TestEngine_backoffLadder` green (exact virtual instants, during-backoff write, reset) |
| T-07.4 | 05-07 | Each failed attempt invokes `OnDumpFailure(err, class)` exactly once; `DumpStats` counts DiskFull/Permission/Other beside total/errors | ✓ VERIFIED | `handleDumpFailure` (`scheduler.go:259-278`); `TestEngine_dumpFailureClasses` green; `TestEngine_backoffLadder` asserts one callback per failed attempt |
| T-07.5 | 05-07 | Health stays SERVING under persistent dump failure; the EACCES stand test asserts one ERROR, the class label, a bounded attempt rate and recovery | ✓ VERIFIED | No health transition in the dump path; `TestStand_persistDumpFailure` green this round (permission class + cause, SERVING, bounded window, metric >= 1, recovery) |
| T-07.6 | 05-07 | The simulator exercises dump→confirmation→replication-notify incl. a failure window and shrunk-limit skip; two same-seed runs identical; the same seam proves cross-node equal-version convergence | ✓ VERIFIED | `Cluster.NotifyInstalled` (`simnet.go:850`); `TestScenario_dumpNotify` (confirm/notify order, ladder offsets, failure still notifies, shrunk limit reported, same-seed runs identical) and `TestScenario_tiebreakConverges` both green this round on the Phase-8 synctest-sequenced harness |
| T-08.1 | 05-08 | `ConfirmTimeout` defaults to 10 s; an uncovered write returns `ErrUnconfirmed`; the record stays installed/dirty; not reported failed or durable | ✓ VERIFIED | `defaultConfirmTimeout` (`engine.go:18`), `await` timeout branch (`confirm.go:98-136`); `TestEngine_confirmationTimeout` green |
| T-08.2 | 05-08 | Single-claim resolution: capture never claims; the first resolver claims under the mutex; a hung dump cannot block the timeout; the timeout removes its waiter | ✓ VERIFIED | `waiter.claimed` + every resolver's claim (`confirm.go:187-290`, file unchanged); `TestEngine_timeoutThenDumpSuccess` and `TestEngine_waiterRemoved` green |
| T-08.3 | 05-08 | `Flush()` = `FlushTimeout(0)`; both bound send and reply; `Snapshot` uses it; `Restore` returns its partial list on timeout with `ErrUnconfirmed` distinguishable from `ErrDumpFailed` | ✓ VERIFIED | `engine.go:244-294`, `snapshot.go:36-42,110-145`; `TestFlush_timeout`, `TestSnapshot_timeout`, `TestRestore_timeoutReturnsPartial` green; the Phase-6 full-rollback and `ErrSnapshotIncomplete` refusal are additive semantics in front of the same bounded seam |
| T-08.4 | 05-08 | Shutdown is bounded: `FlushTimeout(shutdownGrace)`, cancel, `engineDone` at most the grace, WARN instead of hanging | ✓ VERIFIED | `shutdown` (`main.go:855-909`) calls `FlushTimeout(shutdownGrace)` then `awaitEngineStop(node.engineDone, shutdownGrace)` (`main.go:901-907`); `awaitEngineStop` (`main.go:916-925`, extracted by `2b281a9` with identical semantics); `TestAwaitEngineStop` 3/3 subtests green under `-race` this round: clean cancel silent, abnormal error reported, wedged channel returns at the grace with the WARN text |
| T-08.5 | 05-08 | A timed-out write still replicates: Set/Delete notify on `ErrDumpFailed` or `ErrUnconfirmed`; `mapError` maps `ErrUnconfirmed` to Unavailable | ✓ VERIFIED | `server.go:232-235,284-287,496-513` (notify branches unchanged; Phase-6 added the `ErrSnapshotIncomplete`/`ErrNoDump` rows *after* the `ErrDumpFailed` row so writes still answer Unavailable); `TestServer_unconfirmedSetMapsAndNotifies`, `TestServer_unconfirmedDeleteMapsAndNotifies` green |
| T-08.6 | 05-08 | The frozen blocking test stays green with its 200 ms / 1 s windows unedited | ✓ VERIFIED | `engine_blocking_test.go:24-25` still `200 * time.Millisecond` / `time.Second`; the file's only diff since the phase base is the documented `Save`→`SaveStream` fake migration (`bc43705`, plan 05-04) and nothing after the original verification; `TestEngine_blockingSaveHoldsSet` in the green package run |
| T-09.1 | 05-09 | Five internal roles separated: loader/composition root, store facade, dump scheduler, confirmation accounting, snapshot orchestration | ✓ VERIFIED | `loader.go`, `facade.go`, `scheduler.go`, `confirm.go`, `snapshot.go` (+ `engine.go` composition); `flush.go` only carries `flushRequest` |
| T-09.2 | 05-09 | `Engine`'s exported method set/signatures unchanged by the split; consumers compile unedited | ✓ VERIFIED | The split commits' added exported declarations are exact relocations; `go build ./...` and `go vet ./...` green on the current tree (plus darwin) |
| T-09.3 | 05-09 | Confirmation role owns waiters/dirty/epoch/changed/closed/runDone behind its mutex; shares one unexported `*dumpStats` for `unconfirmed` | ✓ VERIFIED | `confirmer` (`confirm.go:42-55`); `stats.unconfirmed.Add(1)` (`confirm.go:135`); capture-before-encode order preserved (`scheduler.go:229-231`) |
| T-09.4 | 05-09 | Scheduler owns ladder/channels/`dumpStats`; `New` and `Open` build one holder shared with the confirmer; facade thin; loader owns Open/mergeRecords/LoadStats | ✓ VERIFIED | `New` creates `stats` once and passes the same pointer to `newConfirmer` and `newScheduler`; `Open` goes through `New` (`loader.go:88`); `DumpStats()` value copy (`scheduler.go:317-329`); `facade.go` forwards (+ additive `LiveVersions`) |
| T-09.5 | 05-09 | Pure move: suite, lint and darwin green; no behaviour change, no new interface, no new exported name | ✓ VERIFIED | `make test` green (every package, this round), `make lint` exit 0, darwin build/vet green; `grep 'type .* interface' internal/app/persist/*.go` = none |
| T-10.1 | 05-10 | OPERATIONS states the single limit, the non-positive exit-2 rule and keeps the 64 408 boundary arithmetic | ✓ VERIFIED | `docs/OPERATIONS.md` `## Record and datagram limits` (line 64); `TestOperationsDoc` anchors green |
| T-10.2 | 05-10 | Monitoring lists the four dump/load families with the every-label-at-zero rule | ✓ VERIFIED | `docs/OPERATIONS.md` `## Monitoring` (line 264) names `supkv_dumps_failures_total` (308), `supkv_dumps_unconfirmed_total` (311), `supkv_dump_load_total` (312), `supkv_dump_records_discarded_total` (314); anchors in `operations_test.go`; names exist in `metrics.go:254,265,481,490` |
| T-10.3 | 05-10 | Lifecycle documents the ladder, health staying SERVING, `Unavailable` for unconfirmed, bounded shutdown | ✓ VERIFIED | `docs/OPERATIONS.md` `## Lifecycle and exit codes` (415-436 area, unconfirmed counter at 415) |
| T-10.4 | 05-10 | Load-outcomes section documents Prompt/Main/Backup/Unusable, one-WARN/one-ERROR, `.corrupt-<UTC>[-N]`, `preserve_error` hazard, the one-hour marker-only sweep | ✓ VERIFIED | `docs/OPERATIONS.md` `## Load outcomes` (line 125); `TestOperationsDoc` green |
| T-10.5 | 05-10 | Memory model states one `Records()` copy + one line + 64 KiB writer and names the residual writer pause | ✓ VERIFIED | `docs/OPERATIONS.md` `## Memory model` (line 190); benchmark printed the supporting figures |
| T-10.6 | 05-10 | Anchors cover the new sections and the file still opens with the trust model | ✓ VERIFIED | `## Trust model` is the first heading (`operations_test.go` asserts it; heading list confirms line 8); section/family anchors green |
| T-11.1 | 05-11 | Exactly the eight named registry rows change (at the plan's commit) | ✓ VERIFIED | `git diff 4f81fd7..1f79eef -- docs/SPEC-DEVIATIONS.md` changes exactly `S-PAR-05, S-PERS-08, S-PERS-09, S-ALG-REP-03, S-HLC-07, S-DM-04, S-DM-05, S-DM-08`; later edits are the documented Phase-7/8 review repoints, not plan drift |
| T-11.2 | 05-11 | No other row's status/links change; the preamble mentions ADR 0012 | ✓ VERIFIED | The same diff's only non-row change is the preamble sentence adding ADR 0012 (`docs/SPEC-DEVIATIONS.md:26`); every other row untouched in that commit |
| T-11.3 | 05-11 | Every changed row cites a current code path/test/ADR; nothing says TBD/future/unchecked | ✓ VERIFIED | Spot-checked on HEAD this round: `S-PERS-08`/`S-PERS-09` cite `dumpfile.go:127-175`/`206-233` (Load chain/preserve, exact), `cmd/sup-kv/main.go:481` (`logLoadStats`, exact) and `main.go:565` (`logUnusableGenerations`, exact), `metrics.go:223-247` (`loadFamilies`, in range); `S-DM-04`/`S-DM-08`/`S-ALG-REP-03` cite `store.go` symbols that resolve (`MergeChanged` 349, `join` 417, `RecordsEqual`) and ADR 0012; line ranges drift a few lines with later-phase insertions (registry-sweep maintenance, not a Phase-5 gap) |
| T-11.4 | 05-11 | Status vocabulary stays the closed enum | ✓ VERIFIED | Column values on HEAD: `implemented` 126, `fixed-in-v1.1` 22, `superseded-by-amendment` 12, `consumer-zone` 4, `accepted-deviation` 2, `deferred-v2` 1 — identical counts to the prior report, no other status |
| T-12.1 | 05-12 | Both P-01 rows flip red->green; other five stay green; runner and frozen tests byte-identical | ✓ VERIFIED | `git diff 7851cfc^..7851cfc -- scripts/red-expectations.tsv` is exactly the two-row flip (both P-01 rows red→green, five R-rows untouched); `git diff 4f81fd7..HEAD` on the runner and both frozen tests is empty; current TSV has 7 green data rows. Per the brief: the frozen test files carry NO changes at all — the flip lives only in the TSV data, exactly as plan 05-12 required (`05-12-PLAN.md:36` prohibition "The runner and the frozen tests must not be edited") |
| T-12.2 | 05-12 | The full red gate is 7/7 with both mutants killed | ✓ VERIFIED | This round: expectation stage checks 7 rows, 6 OK including both P-01 rows (`TestRed_P01_valueLimit`, `TestRed_P01_valueLimitEngine` — evidence `20261127T171623Z`, also green standalone); the one non-OK row is `TestRed_R01_pacedSync20k`, Phase 3's XPORT-01 wall-clock pacing contract, failing on this powersave-throttled host while converging — the adjudicated environmental class owned by the scheduled `red-r01-x20` CI job (see Deferred Items), not a Phase-5 must-have; the mutation stage is green standalone this round (control 4/4 seeds, both mutants killed, clean-tree asserts) |
| T-12.3 | 05-12 | `make lint`, `make test`, darwin build/vet and the bench smoke are green on the committed tree | ✓ VERIFIED | `make lint` exit 0; `make test` exit 0 with **every** package ok (the prior `TestCore_keyCountCostFlat` deferral is closed by Phase 8's deterministic 08-05 gate); `GOOS=darwin GOARCH=arm64 go build ./... && go vet ./...` exit 0; bench smoke printed B/op, dump-bytes, peak-live-B, p50/p99 |
| T-12.4 | 05-12 | No test, bound or runner line weakened; the only diff outside the phase's code is the TSV flip | ✓ VERIFIED | Frozen runner/tests unchanged vs base `4f81fd7`; the blocking test's windows unchanged (`engine_blocking_test.go:24-25`); post-phase test edits are later phases' authorized work (Phase-8 harness rework, the mechanical `persist.Encode/Decode` signature edits in `decode_negative_test.go` keep every assertion — read the diff: only the two new `UnboundListing` cases); the 05-12 commit's non-TSV additions are red-runner evidence JSON, not tests or bounds; review-fix edits are authorized by `05-REVIEW-DISPOSITION.md` |

### Deferred Items

Not actionable Phase-5 gaps; owned elsewhere with recorded adjudications.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | `TestRed_R01_pacedSync20k` misses locally on this powersave-throttled host (converging at 7896/8000 keys against the 20.6 s documented bound) | Phase 3 acceptance (scheduled CI job `red-r01-x20`) | The row pins Phase 3's XPORT-01 contract, not a Phase-5 must-have; adjudicated by the Phase-3 halt record (03-12), the Phase-4 verification (2026-11-27, tree `ef370a0`) and Phase 3's own 2026-11-27 verification; green on full-speed-host evidence `20261121T225048Z` and on the 2026-11-23/24 local runs of identical production sources. This round's P-01 rows and the mutation stage are green. |

**Note on the prior report's two deferred items:** both are now closed by Phase 8 work and moved to `re_verification.gaps_closed` — the `TestCore_keyCountCostFlat` wall-clock ratio gate was replaced by plan 08-05's deterministic operation-count gate (`internal/app/repl/bench_test.go:210` "deterministic acceptance"), and the `freeUDPPort` pick-to-bind window was replaced by plan 08-06's shared claim-while-held allocator (`074ccd0`, claim-before-probe `7aeec35`, probe retry `9cdc0f1`). This round's full `make test` passed every package.

**Note on commit `2b281a9` (per the verification brief):** the test-only commit postdating 05-REVIEW.md extracted `awaitEngineStop` from `shutdown` as the testable seam (`cmd/sup-kv/main.go:916-925`) — the select branches, the grace parameterization and the WARN text are identical to the reviewed code — and added `TestAwaitEngineStop`'s wedged-dumper subtest. All three subtests pass under `-race` this round; the production `shutdown` sequence (GracefulStop bound → flush → cancel → bounded stop wait) is unchanged. Included in the evidence review; no deviation.

**Note on the P-01 flip state (per the verification brief):** plan 05-12's flip commit `7851cfc` changes exactly the two P-01 TSV rows red→green; the runner (`scripts/red-runner`) and both frozen tests (`internal/persistfile/red_value_limit_test.go`, `internal/stand/stand_red_test.go`) are byte-identical to the phase base `4f81fd7` on HEAD. The frozen tests carry no flip at all — neither data nor expectation edits — exactly as the plan required. Both tests pass on the current tree directly and through the runner.

### Advisory (New Scope, Unevidenced)

None. This regeneration formed no unevidenced Step-7 blocker. The registry line-number drift noted under T-11.3 is an informational maintenance observation, not a Phase-5 blocker.

## Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/app/storage/store.go` | `ValueMaxBytes`, `Config.ValueLimit`, `Store.ValueMaxBytes`, `RecordsEqual`, `MergeChanged`, `join` | ✓ VERIFIED | All present at `store.go:21,141,161,180,349,417`; single production literal; `LiveVersions` and the generation-bearing tap are additive |
| `internal/app/persist/codec.go` | config-driven `Encode`/`Decode`, `EncodeTo`, census, four-class stats, format-bound reader | ✓ VERIFIED | `EncodeTo` at 143 (with the additive Phase-7 limit-domain guard), `Encode` wrapper, `Decode` at 261, `DecodeStats` at 194, `MaxDumpLineBytes` at 62; no `json.Decoder` |
| `internal/app/persist/engine.go` | composition + public write/read/`Flush`/`LoadStats`; `Config` with `ConfirmTimeout`/`Now`/`After`/`OnDumpFailure` | ✓ VERIFIED | 300+ lines; five roles composed in `New`; additive `OnConfirmed` (nil-safe, dumper goroutine) |
| `internal/app/persist/loader.go` | `Open`, `mergeRecords`, `LoadStats` construction, post-load dirty | ✓ VERIFIED | `Open` decodes with `ValueLimit()` before the store (`loader.go:48`); frozen `(*Engine, int)` signature; additive OnChange loading guard |
| `internal/app/persist/facade.go` | thin forwards + `ApplyRemote` | ✓ VERIFIED | `ApplyRemote` dirties only on `changed` (`facade.go:40-50`); additive `LiveVersions` |
| `internal/app/persist/scheduler.go` | `Run`, `runDump`, `rearm`, `recordDump`, `dumpStats`, ladder | ✓ VERIFIED | Streaming dump, ladder 100 ms..5 s, class counters, one `OnDumpFailure` per attempt; generation read before `Records()` (additive, APIS-04) |
| `internal/app/persist/confirm.go` | `waiter`, `await`, `confirmWaiters`, `failWaiters`, `sweepClosed` | ✓ VERIFIED | Single-claim protocol and timeout waiter removal; byte-identical to the phase state |
| `internal/app/persist/snapshot.go` | `Snapshot`, `Restore`, replay helpers | ✓ VERIFIED | Bounded flush through the public seam; partial list on timeout; Phase-6 full-rollback and `ErrSnapshotIncomplete` refusal are additive semantics |
| `internal/app/persist/errors.go` | `ErrUnconfirmed` | ✓ VERIFIED | Plus `ErrDumpFailed`, `ErrClosed`, `ErrSnapshotIncomplete` (Phase-6 additive) |
| `internal/app/persist/flush.go` | shared `flushRequest` type | ✓ VERIFIED | Snapshot vs explicit origin; unchanged |
| `internal/app/persist/encode_stream_internal_test.go` | `TestEncodeTo_byteIdentical`, `TestEncodeTo_maxSingleWrite` | ✓ VERIFIED (renamed path) | Plan named `encode_stream_test.go`; the strict `testpackage` lint mandates `_internal_test.go` (documented deviation in 05-04-SUMMARY, "Auto-fixed Issues" 1). Both named tests exist and pass; `verify.artifacts` flags only the path name (5/6) |
| `cmd/sup-kv/shutdown_internal_test.go` | `TestAwaitEngineStop` covering clean-cancel, abnormal-error and wedged-expiry branches | ✓ VERIFIED | Three subtests PASS under `-race` (added by `2b281a9`; reviewed this round) |
| `internal/port/dump.go` | `SaveStream`, `Load(...) LoadResult`, outcome types, `FailureClass`/`ClassedError` | ✓ VERIFIED | Full contract at `port/dump.go:28-157`; unchanged |
| `internal/persistfile/dumpfile.go` | outcomes, preservation, sweep, marker, `SaveStream`, ordered save | ✓ VERIFIED | `Load` main->`.prev`; `preserve`; `sweepOrphans`; `saveAtomically`; unchanged |
| `internal/persistfile/fsops.go` | five-step seam with buffered fill; parent sync; dir-fsync tolerance | ✓ VERIFIED | 64 KiB `bufio`; cleanup on failure; tolerance only on the dir step; unchanged |
| `internal/persistfile/saveerror.go` | errno -> class mapping preserving `errors.Is` | ✓ VERIFIED | ENOSPC/EACCES/fs.ErrPermission; unchanged; Phase-6 reuses it for snapshot errors |
| `internal/persistfile/snapshot.go` | validated names, snapshot copy under the save mutex | ✓ VERIFIED | Charset `[A-Za-z0-9._-]` (rejects `~`); Phase-6 adds `classifySaveError` wrapping (additive) |
| `cmd/sup-kv/main.go` (+ `internal/config/*`) | flag default from the constant, range/ceiling checks, load logs, skip WARN, `OnDumpFailure`, bounded shutdown | ✓ VERIFIED | `config.go:235`; `validate.go:152-154,208-224`; `main.go:63-65,432-463,447-455,481-579,855-925`; the Phase-7 extraction kept the value-limit and ceiling messages byte-identical |
| `internal/metrics/metrics.go` | four new families with label enums | ✓ VERIFIED | `loadFamilies`, `discardClassFamilies`, `supkv_dumps_failures_total` (disk_full/permission/other), `supkv_dumps_unconfirmed_total`; Phase-8's additions sit beside them |
| `internal/app/repl/core.go` | content-aware equal-version branch | ✓ VERIFIED | Consults `storage.RecordsEqual`, re-reads the stored record for the reply verdict (`core.go:1647,1676`) |
| `internal/app/internal/simnet/simnet.go` | `Cluster.NotifyInstalled` | ✓ VERIFIED | Lookup + NotifyLocalWrite + route + write-log capture (`simnet.go:850`); Phase-8's peer-snapshot parameter is additive |
| `docs/adr/0012-equal-version-tiebreak.md` | recorded decision | ✓ VERIFIED | Full slim-MADR |
| `docs/OPERATIONS.md` | limits, load outcomes, memory model, monitoring, lifecycle | ✓ VERIFIED | All mandated sections; trust model first |
| `docs/SPEC-DEVIATIONS.md` | eight plan rows + review-fix rows | ✓ VERIFIED | Row-set reconciliation above; closed status enum |
| `scripts/red-expectations.tsv` | two P-01 rows green, five others green | ✓ VERIFIED | 7 data rows, all green; the flip is the only edit |

## Key Link Verification

The `verify.key-links` heuristic reports `all_verified: false` for eleven of the twelve plans (only 05-11, whose links are doc-internal text matches, reports 3/3). Its matcher looks for the target *filename* in the source file, while these Go links are package imports or prose `via` clauses; the manual evidence below is authoritative. All 32 plan-declared links were re-checked on the current tree.

| # | From | To | Via (abridged) | Status | Details |
| - | ---- | -- | -------------- | ------ | ------- |
| 1 | `persist/loader.go` | `storage/store.go` | `Open` decodes through `Config.ValueLimit()` before the store; `NewStore` normalizes through the same helper | WIRED | `loader.go:48`; `store.go:141-156` |
| 2 | `persist/codec.go` | `storage/store.go` | callers pass the limit; oversize encode is the `storage.ErrValueTooLarge` tripwire | WIRED | `codec.go:143-175`; `store.go:21` |
| 3 | `cmd/sup-kv/main.go` -> `internal/config` | `storage/store.go` | flag default and ceiling effective value are `storage.ValueMaxBytes` | WIRED | `config.go:235`; `validate.go:208-224`; CLI `--help` default 61440 |
| 4 | `persist/codec.go` | `persist/loader.go` | decode closure stores stats; `Open` returns `stats.Total()` | WIRED | `loader.go:52-59,110` |
| 5 | `persist/codec.go` | `persistfile/dumpfile.go` | parse oracle is `Decode` at the configured limit; generation faults advance the chain | WIRED | `loader.go:54`; `dumpfile.go:127-193` |
| 6 | `storage/store.go` | `repl/core.go` | `RecordsEqual` is the one content comparison; core Acks equal version only after consulting it | WIRED | `core.go:1647,1676`; `store.go:194-196` |
| 7 | `repl/core.go` | `persist/facade.go` | tie-break install reaches `ApplyRemote`; core re-reads the stored record for Ack vs reply | WIRED | `core.go:1653,1675-1676`; `facade.go:40-50` |
| 8 | `persist/facade.go` | `storage/store.go` | `MergeChanged` returns the installation decision; `Merge` stays the 3-result wrapper | WIRED | `facade.go:41`; `store.go:349,366` |
| 9 | `persist/scheduler.go` | `persistfile/dumpfile.go` | engine fill runs on the dumper goroutine through the adapter's buffered temp writer; fill error returns via the port | WIRED | `scheduler.go:229-252`; `fsops.go:96-117` |
| 10 | `port/dump.go` | `persist/loader.go` | `Load` returns the outcome/causes; `Open` records them; signature unchanged | WIRED | `port/dump.go:42,98-110`; `loader.go:60,90-107` |
| 11 | `persistfile/fsops.go` | `persistfile/fsops_test.go` | recording fake records the `write` step with its fill and delegates to the real implementation | WIRED | `fsops_test.go:69-72`; `TestFsops_saveOperationOrder` |
| 12 | `persistfile/dumpfile.go` | `persist/loader.go` | the engine's parse oracle decides a candidate's fate; the adapter preserves and reports | WIRED | `dumpfile.go:150-160`; `loader.go:49-56` |
| 13 | `persistfile/dumpfile.go` | `persistfile/fsops.go` | preservation uses `Lstat`/`Rename` directly; the save path's parent sync goes through the seam | WIRED | `dumpfile.go:206-241,319-334`; `fsops.go:175-182` |
| 14 | `persistfile/saveerror.go` | `port/dump.go` | errno knowledge stays in the adapter; the core receives `port.FailureClassOf(err)` | WIRED | `saveerror.go:32-45`; `port/dump.go:149-157` |
| 15 | `persist/loader.go` | `port/dump.go` | `Open` maps `LoadResult` into the engine's `LoadStats` | WIRED | `loader.go:90-107`; `engine.go:147-167` |
| 16 | `cmd/sup-kv/main.go` | `persist/engine.go` | `startEngine` reads `LoadStats()` once after open and decides one WARN/ERROR/INFO | WIRED | `main.go:447-455,481-579` |
| 17 | `metrics/metrics.go` | `persist/engine.go` | `Sources.LoadStats` is wired to `engine.LoadStats`; families render at zero | WIRED | `main.go:724-752` area; `metrics.go:191,234-283` |
| 18 | `persist/engine.go` | `simnet.go` | tests bind `Config.Now`/`After` to virtual time; the engine shares `Cluster.Store(0)` and announces via `NotifyInstalled` | WIRED | `sim_dump_test.go`; `simnet.go:850`; `TestScenario_dumpNotify` green |
| 19 | `persist/scheduler.go` | `port/dump.go` | `FailureClassOf` extracts the adapter class; the core never imports `syscall` | WIRED | `scheduler.go:260`; `port/dump.go:112-157` |
| 20 | `cmd/sup-kv/main.go` | `persist/engine.go` | main wires `OnDumpFailure` to one `slog.Error` with the class | WIRED | `main.go:423-430,447` |
| 21 | `persist/engine.go` | `grpcapi/server.go` | RPC failure branches notify for either unconfirmed class; `mapError` is total over it | WIRED | `server.go:232-235,284-287,496-513` |
| 22 | `persist/engine.go` | `cmd/sup-kv/main.go` | `FlushTimeout(shutdownGrace)` and the bounded `engineDone` wait | WIRED | `main.go:901-907,916-925`; both bounds behaviorally tested |
| 23 | `persist/engine.go` | `persist/confirm.go` | engine composes one confirmer; writes register/wait through it; the scheduler resolves | WIRED | `engine.go:173-189`; `confirm.go:98-136,207-290` |
| 24 | `persist/scheduler.go` | `persist/confirm.go` | capture-before-encode and the single-claim resolution cross the two roles | WIRED | `scheduler.go:229-252`; `confirm.go:187-290` |
| 25 | `persist/snapshot.go` | `persist/engine.go` | `Snapshot` forces the flush through the public method; `Restore` reuses the facade | WIRED | `snapshot.go:36-42,110-145` |
| 26 | `docs/OPERATIONS.md` | `cmd/sup-kv/operations_test.go` | each mandated statement has an anchor literal | WIRED | `TestOperationsDoc`/`TestOperationsDocFamilies` green |
| 27 | `docs/OPERATIONS.md` | `internal/metrics/metrics.go` | every family in the monitoring list exists in the exporter | WIRED | names grep-present in `metrics.go` |
| 28 | `docs/SPEC-DEVIATIONS.md` | `storage/store.go` | rows cite `ValueMaxBytes`/`ValueLimit`/`Store.ValueMaxBytes`/`MergeChanged` | WIRED | Rows 38/39/42/232; symbols present (`MergeChanged` at 349) |
| 29 | `docs/SPEC-DEVIATIONS.md` | `persistfile/dumpfile.go` | `S-PERS-08`/`S-PERS-09` cite the load/preserve path and the load metric | WIRED | Rows 166/167; code present |
| 30 | `docs/SPEC-DEVIATIONS.md` | `docs/adr/0012-...md` | `S-ALG-REP-03` and `S-HLC-07` link the ADR | WIRED | Rows 55/127 |
| 31 | `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go` | flipped row names the frozen stand test | WIRED | Red runner row `TestRed_P01_valueLimit` OK; test passes |
| 32 | `scripts/red-expectations.tsv` | `internal/persistfile/red_value_limit_test.go` | flipped row names the frozen engine test | WIRED | Red runner row `TestRed_P01_valueLimitEngine` OK; test passes |

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
| Full race suite | `make test` (`go test -race -cover ./...`) | exit 0 — every package ok, incl. `internal/app/repl` (46.9 s) and `cmd/sup-kv`; the prior flake deferrals are closed by Phase 8's deterministic gates | ✓ PASS |
| Darwin cell | `GOOS=darwin GOARCH=arm64 go build ./... && go vet ./...` | exit 0 both | ✓ PASS |
| Red gate (current tree) | `SUPKV_RED_TESTS=1 bash scripts/red-runner` (2 runs) | expectation stage: 7 rows checked, 6 OK incl. both P-01 rows; `TestRed_R01_pacedSync20k` misses on this throttled host (adjudicated Phase-3 class, see Deferred Items); evidence `20261127T171347Z`, `20261127T171623Z` | ⚠️ DEFERRED (one out-of-scope wall-clock row) |
| Both P-01 rows directly | `SUPKV_RED_TESTS=1 go test -race -count=1 ... -run 'TestRed_P01_valueLimit'` (both packages) | PASS (4.28 s stand; 0.08 s persistfile) | ✓ PASS |
| Mutation stage standalone | `bash /tmp/red-mutant-standalone.sh` (exact re-execution of the runner's `run_mutation_stage`) | control 4/4 seeds pass; both mutants compile and are killed on the kill seed; clean-tree asserts pass | ✓ PASS |
| Bounded engine stop (all branches) | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAwaitEngineStop$' -v` | 3/3 subtests PASS (clean cancel silent, abnormal reported, wedged abandoned at the grace with the WARN) | ✓ PASS |
| Phase core packages, uncached | `go test -race -count=1 ./internal/app/persist/ ./internal/persistfile/ ./internal/app/storage/ ./internal/metrics/` | all `ok` | ✓ PASS |
| Value-limit CLI boundary | `TestMain_valueMaxNonPositive`, `TestMain_valueMaxDatagram` | both PASS (0/negative exit 2; 64 000 and 64 408 serve; 64 409 exit 2) | ✓ PASS |
| Direct binary probe | built `./cmd/sup-kv`; `--value-max-bytes=0`/-5/64409 | exit 2 all; flag named; `65507-byte UDP ceiling` text; 6409 with full knobs starts, logs `value_max_bytes:6409`, serves | ✓ PASS |
| Tie-break + simulator | `go test -race -count=1 ./internal/app/repl/ -run 'TestScenario_dumpNotify|TestScenario_tiebreakConverges|TestReplication_equalVersion'` | all PASS | ✓ PASS |
| Process load outcomes | `go test -race -count=1 ./internal/stand/ -run TestStand_persist -v` | 7/7 PASS, 0 skips | ✓ PASS |
| Unconfirmed RPC mapping | `go test -race -count=1 ./internal/grpcapi/ -run 'TestServer_unconfirmed'` | PASS | ✓ PASS |
| Confirmation/timeout/backoff/skip named tests | `go test -race -count=1 ./internal/app/persist/ -run 'TestEngine_confirmationTimeout|TestFlush_timeout|TestSnapshot_timeout|TestRestore_timeoutReturnsPartial|TestEngine_backoffLadder|TestEngine_skipEvidenceWindow|TestEngine_markDirtyNudge|TestEngine_loadStatsMapsResult|TestEngine_timeoutThenDumpSuccess|TestEngine_waiterRemoved|TestEngine_injectedClock|TestEngine_dumpFailureClasses|TestEngine_confirmDebounceTiming|TestEngine_exactValueLimitRestart|TestEngine_startDirty|TestCodec_headerCensus|TestCodec_corruptLineIsolated|TestEncodeTo'` | all PASS | ✓ PASS |
| Load-outcome/preserve/sweep named tests | `go test -race -count=1 ./internal/persistfile/ -run 'TestFileStore_loadOutcomeTable|TestFileStore_loadBackupPreservesMain|TestSweep|TestEnsureDir|TestSyncDirTolerant'` | all PASS | ✓ PASS |
| Docs anchors | `go test -race -count=1 ./cmd/sup-kv/ -run 'TestOperationsDoc'` | PASS | ✓ PASS |
| Dump bench smoke | `go test -run '^$' -bench BenchmarkDump10kKeys -benchtime=1x ./internal/app/persist/` | `bytes/op 5147571`, `dump-bytes 5147571`, `peak-live-B 10857392`, p50/p99 printed | ✓ PASS |
| Single production literal | `grep -rn 61440 --include=*.go .` (production files) | only `internal/app/storage/store.go:21` | ✓ PASS |
| No streaming decoder poison | `grep -c json.NewDecoder internal/app/persist/codec.go` | 0 | ✓ PASS |
| No wall clock in persist | `grep 'time.(Now\|After)'` over non-test persist sources | only `engine.go:77,80` (the `normalized()` defaults) | ✓ PASS |
| No interfaces in persist | `grep 'type .* interface' internal/app/persist/*.go` | none | ✓ PASS |
| Debt markers in phase files | `grep -n -E 'TBD\|FIXME\|XXX'` over the phase's production packages | none | ✓ PASS |

## Probe Execution

No `scripts/*/tests/probe-*.sh` exists and no plan declares a probe script. The phase's declared acceptance runner is the red gate, re-executed this round twice on the current tree (expectation stage 6/7 with both P-01 rows OK; the R-01 miss is the adjudicated Phase-3 environmental class), plus the mutation stage reproduced standalone green following the runner's own `run_mutation_stage` procedure (control 4/4, both mutants killed). Latest evidence: `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261127T171347Z/` and `.../20261127T171623Z/`.

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
| `internal/app/repl/sim_dump_test.go` | PERSIST-05/07/12 | 2 | 0 | No | Behavioral (same-seed traces, synctest-sequenced) | PASS |
| `internal/metrics/metrics_test.go` | PERSIST-02/05/08/10 | all | 0 | No | Value (by-effect samples) | PASS |

**Disabled tests on requirements:** 0 (the env-gated red tests are the designed mechanism and ran through the runner and directly). **Circular patterns:** 0 — fixtures carry literal records/checksums; no test generates expected values by running the system under test. **Assertion strength:** `TestAwaitEngineStop`'s wedged subtest asserts a lower time bound (>= grace), an upper bound (<= 2 s) and the exact WARN substring, so a removed timeout, an immediate return or a dropped WARN each fail. **Insufficient assertions:** 0.

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
| PERSIST-08 | 08, 12 | ✓ SATISFIED | Writes/snapshots/`FlushTimeout` bounds tested; `mapError`/notify tested; the `engineDone` grace branches (clean/abnormal/wedged) are exercised by `TestAwaitEngineStop` |
| PERSIST-09 | 05, 12 | ✓ SATISFIED | Marker-only age-bounded sweep, parent sync on create, ENOTSUP/EINVAL dir-sync tolerance, `fsync_darwin.go` absent |
| PERSIST-10 | 01, 02, 06, 12 | ✓ SATISFIED | `ValueLimit` class, skip count + one WARN + metric, evidence window through two rotations |
| PERSIST-11 | 04, 09, 12 | ✓ SATISFIED | Five roles, public API stable, port returns outcome/cause, no new interface |
| PERSIST-12 | 07, 12 | ✓ SATISFIED | `Now`/`After` seam; no wall clock in the package; simulator drives dump<->notify and the tie-break |

No ORPHANED requirements: all twelve PERSIST IDs are mapped to Phase 5 in `.planning/REQUIREMENTS.md` (lines 251-262, "Complete") and claimed by at least one plan — 05-12 claims all twelve; the others claim the subsets listed above.

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX` in any file modified by this phase | — | None |
| — | — | No `TODO`/`HACK`/`PLACEHOLDER`/stub markers in the phase's production files | — | None |
| — | — | No empty implementations or bare `{}` function bodies in the phase's production files | — | None |
| — | — | Every `//nolint` names its linter and a human-readable reason; `.golangci.yml`'s changes since Phase 5 are the documented additive `depguard` rules with written justification | — | None |
| `internal/app/persist/bench_test.go` | 93-98, 122 | `peak-live-B` samples `HeapAlloc`, which includes not-yet-collected line garbage; the 10.9 MB sample is ~2.1x the 5.15 MB state | ℹ️ Info | The documented model (one copy + one line + 64 KiB) and the 5.15 MB `bytes/op` (one state size) meet PERSIST-06; the sampled figure is a high-water mark, not live reachable bytes. No action for this phase |

`math/rand` appears in the replication run-ID only — never in the tie-break, which is `bytes.Compare` over explicit fields. (The Phase-8 port allocator removed the test-harness `math/rand` port picking.)

## Decision Coverage

All 12 trackable CONTEXT.md decisions (D-01…D-12) are honored by shipped artifacts (`check.decision-coverage-verify`: total 12, honored 12, not_honored []).

## Human Verification Required

None. This is an infrastructure/foundation phase with no user-facing element, every truth is verified programmatically, and the behaviour-dependent truths (bounded shutdown, confirmation timeout, waiter removal, backoff ladder, restart durability) are each exercised by named passing tests this round.

## Gaps Summary

No gaps. All 12 requirements have shipped implementations wired end to end, all five roadmap success criteria are met, and every plan truth has current-tree evidence. This pass was a stale-fingerprint regeneration: the prior report's 64/64 was re-derived against the phase-8-complete tree (`bd88889`; implementation sources unchanged since `ef370a0`), not trusted. The later phases' covered-file edits (full-rollback Restore, `OnConfirmed` watermark, `LiveVersions`, `ErrSnapshotIncomplete`, the `internal/config` extraction, the public API move, the Phase-8 harness rework) are additive in every area phase 5 owns, and the review's deferred IN-04 library guard landed in Phase 7 exactly as planned. The prior report's two deferrals are closed by Phase 8's deterministic gates (recorded in `re_verification.gaps_closed`). One environmental deferral is recorded: the red gate's `TestRed_R01_pacedSync20k` row misses on this throttled host while converging — Phase 3's contract, adjudicated with CI acceptance scheduled; phase 5's own acceptance rows (both P-01) and the mutation stage are green first-hand. The `encode_stream_internal_test.go` filename and the registry line-number drift are documented deviations/maintenance notes, not drift.

---

_Verified: 2026-11-27T17:21:33Z_
_Verifier: Claude (gsd-verifier)_
