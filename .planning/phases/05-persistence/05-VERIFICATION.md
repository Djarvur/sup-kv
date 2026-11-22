---
phase: 05-persistence
verified: 2026-11-22T19:13:03Z
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
  - .planning/phases/05-persistence/05-REVIEW-FIX.md
  - .planning/phases/05-persistence/05-REVIEW.md
  - .planning/phases/05-persistence/05-VALIDATION.md
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
covered_digest: "v4:sha256:9a8ffb4da1e83c632ab8198b9b313a2afa130433e5f86673496c2d7accb7af27"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 63/64
  gaps_closed:
    - "Shutdown is bounded: the engineDone abandonment branch is now exercised by TestAwaitEngineStop (grace expiry, bounded return, WARN text)"
  gaps_remaining: []
  regressions: []
---

# Phase 5: Persistence Verification Report

**Phase Goal:** The dump path is honest about its limits and failures — one enforced value limit, visible load outcomes, generation integrity, bounded memory and blocking — so "a confirmed write is on disk" is a tested guarantee rather than a convention.
**Verified:** 2026-11-22T19:13:03Z
**Status:** passed
**Re-verification:** Yes — after the shutdown-abandonment gap closure (`2b281a9`)

## Re-verification Round

The previous pass returned `human_needed` for exactly one item: T-08.4 / PERSIST-08's bounded-shutdown truth, whose `engineDone` grace-expiry branch no test exercised. Commit `2b281a9` (22:05 local, after that verification at 22:03) extracts the wait into `awaitEngineStop(done <-chan error, grace time.Duration)` and adds `cmd/sup-kv/shutdown_internal_test.go`.

Verified against the tree, not the message:

- **Behavior identity.** `shutdown()` now calls `awaitEngineStop(node.engineDone, shutdownGrace)` and still returns its earlier `FlushTimeout` error first (`main.go:955-975`). The helper preserves the three branches verbatim: clean `context.Canceled` silent, other `Run` error logged via `log.Printf("sup-kv: engine Run returned %v")`, expiry `slog.Warn("Engine did not stop within the shutdown grace", "grace", grace.String())` (`main.go:977-987`).
- **The expiry branch is genuinely exercised.** `TestAwaitEngineStop` feeds a never-written `<-chan error` with a 25 ms grace and asserts the call returns no earlier than the grace, no later than 2 s, and emits the WARN text; the other two subtests pin the clean-cancel silence and the abnormal-error report. The subtests swap process-global log writers and are therefore sequential under one justified `//nolint:paralleltest` (linter + reason named; lint clean).
- **Ran it.** `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAwaitEngineStop$' -v` — all three subtests PASS (0.03 s), including `.../a_wedged_engine_is_abandoned_at_the_grace_with_a_WARN`.
- **No regression.** The commit touches exactly `cmd/sup-kv/main.go` and `cmd/sup-kv/shutdown_internal_test.go`; the frozen runner, both frozen P-01 tests, `.golangci.yml`, `go.mod` and `go.sum` are still byte-identical to the phase base (empty `git diff`). `go build ./...`, `make lint`, `make test`, the darwin build/vet cell, `make bench-smoke` and the 7/7 red gate (with both mutants killed, evidence `20261122T191142Z`) were re-run green on the current tree.

The item is now behaviourally proven, so the truth upgrades to VERIFIED and the phase has no human-verification items left.

## Goal Achievement

### Roadmap Success Criteria

| # | Criterion | Status | Evidence |
| - | --------- | ------ | -------- |
| 1 | Value limit defined once and enforced everywhere (P-01 turns green): exact `value_max_bytes` survives restart at any accepted flag value; invalid value exits 2; the 61 440 constant exists once | ✓ VERIFIED | `storage.ValueMaxBytes` is the only production `61440` (grep: `store.go:21` alone); `Config.ValueLimit()` (`store.go:131-136`) is used by `Open` before the store exists (`loader.go:42-46`) and by `NewStore` (`store.go:140-146`); `Store.ValueMaxBytes()` feeds encoder/snapshot decode; CLI default is the constant (`main.go:362`), 0 exits 2 naming the flag, 64 409 exits 2 with the ceiling text (direct CLI probe); frozen P-01 tests pass unedited in the red runner |
| 2 | Dump-load outcomes visible, never destroy evidence: `.prev` -> one WARN with path+cause; both unusable -> ERROR + nonzero metric; absent -> neither; preserved under a separate name; format carries a record count so line-boundary truncation is incomplete; one corrupted record discards only that record | ✓ VERIFIED | `persistfile.Load` four outcomes (`dumpfile.go:127-175`), `preserve` collision-probed rename before any save (`dumpfile.go:206-233`), `.prev` byte-identical across the first post-backup save (`TestFileStore_loadBackupPreservesMain`), header `record_count` (`codec.go:94-97,320-326`), per-record classes (`codec.go:337-361`), one WARN/one ERROR in `cmd` (`main.go:613-664`), metrics families (`metrics.go:223-247`); all seven stand process tests and the outcome/tables pass |
| 3 | Failure handling bounded and honest: dirty after backup/empty start (first rotation never displaces the last good generation; Snapshot right after decodes back); one ERROR per failure with cause, growing pause to a cap, health documented; skipped records counted/logged | ✓ VERIFIED | `startLoadDirty` (`loader.go:75-90`), `TestEngine_startDirty` (Prompt/Backup/Unusable dirty + Main-with-discard clean, snapshot decodes back), retry ladder `nextRetry` 100 ms..5 s (`scheduler.go:18-25,271-280`), `OnDumpFailure` once per attempt (`scheduler.go:248-266`), `TestEngine_backoffLadder`, `TestStand_persistDumpFailure` (EACCES: ERROR+class, health SERVING, bounded rate, recovery), skip accounting (`main.go:592-596`, `metrics.go:252-263`), `TestEngine_skipEvidenceWindow` |
| 4 | Resources bounded: streaming encode at most two state sizes with the memory model documented; a hung fsync cannot block writes, snapshots or shutdown indefinitely; orphan temps collected; parent synced; `fsync_darwin.go` gone | ✓ VERIFIED | Streaming path: `Records()` once + `SaveStream(EncodeTo)` (`scheduler.go:221-238`), `TestEncodeTo_maxSingleWrite`, bench reports B/op + sampled `peak-live-B`, model documented (`OPERATIONS.md:158-175`); writes/snapshots bounded and tested (`TestEngine_confirmationTimeout`, `TestFlush_timeout`, `TestSnapshot_timeout`); **shutdown now bounded by two tested bounds — `FlushTimeout(shutdownGrace)` and `awaitEngineStop`'s grace select, whose expiry branch `TestAwaitEngineStop` exercises (return ≥ grace, ≤ 2 s, WARN text)**; sweep/parent-sync/dir-fsync tested (`TestSweep_*`, `TestEnsureDir_*`, `TestSyncDirTolerant`); `fsync_darwin.go` absent from the tree |
| 5 | Structure serves seams: `persist.Engine` split into five roles with the `DumpStore` port returning outcome and failure cause; timers/`time.Now()` behind the repl-style seam so the simulator exercises dump↔replication; equal-version different content resolved by a recorded ADR, implemented and tested | ✓ VERIFIED | Five role files (`loader.go`, `facade.go`, `scheduler.go`, `confirm.go`, `snapshot.go`) + composition `engine.go`; `port.DumpStore.Load(now, parse) LoadResult` with `LoadOutcome`/`Generation`/`GenerationFailure` (`port/dump.go:22-110`); only wall-clock references are the two defaults in `normalized()` (`engine.go:67,70`); `TestScenario_dumpNotify`, `TestScenario_tiebreakConverges`, `TestEngine_injectedClock`; ADR 0012 records the deterministic `(version, content)` law; store and core wired. |

### Observable Truths (plan must-haves)

**Score:** 64/64 truths verified, 0 present-behaviour-unverified.

| # | Plan | Truth | Status | Evidence |
| - | ---- | ----- | ------ | -------- |
| T-01.1 | 05-01 | 61 440 exists exactly once in production; store default, pre-store decode and encoder all derive through `Config.ValueLimit()`/`Store.ValueMaxBytes()` | ✓ VERIFIED | `grep 61440` non-test = `store.go:21`; `loader.go:42`; `store.go:131-153`; codec takes the limit argument (`codec.go:126,240`); `main.go:362,541-555` |
| T-01.2 | 05-01 | A record of exactly the configured limit survives restart at two accepted flag values (64 000, 8); one byte over is rejected by the store | ✓ VERIFIED | Frozen `TestRed_P01_valueLimit` (63 000 and exactly 64 000, restart after SIGKILL) green in red-runner; `TestEngine_exactValueLimitRestart` (8-byte limit + 9-byte reject) |
| T-01.3 | 05-01 | `--value-max-bytes=0`/negative exit 2 naming the flag; 64 400 and 64 408 start; 64 409 exits 2 with the unchanged ceiling message | ✓ VERIFIED | `TestMain_valueMaxNonPositive`; `TestMain_valueMaxDatagram`; direct CLI probe: both cases exit 2 with the expected texts; `--help` default 61440 |
| T-01.4 | 05-01 | The two frozen P-01 tests pass unedited | ✓ VERIFIED | `git diff BASE..HEAD` on `stand_red_test.go`, `red_value_limit_test.go`, `scripts/red-runner` is empty; red-runner 7/7 rows OK |
| T-02.1 | 05-02 | The dump announces `record_count`; fewer record lines is a generation-level error; a v1.0 dump without the field takes the legacy path | ✓ VERIFIED | `codec.go:94-97,130,320-326`; `TestCodec_headerCensus` (writer count, truncated, extra-lines tolerated, legacy no-census) |
| T-02.2 | 05-02 | One syntax-corrupt line costs exactly that record; the rest load; the bad line counts `Malformed`; the `json.Decoder` poison behaviour is gone | ✓ VERIFIED | `Decode` per-line `json.Unmarshal` (`codec.go:256-273`); `grep -c json.NewDecoder codec.go` = 0; `TestCodec_corruptLineIsolated` (intact siblings load, Malformed=1) |
| T-02.3 | 05-02 | `DecodeStats` keeps checksum, version, value-limit and malformed separate, each counted, with `Total()` | ✓ VERIFIED | `codec.go:168-212`; `TestCodec_discardClasses` |
| T-02.4 | 05-02 | The line bound is a format property (config-independent); a dump written at the default still reads under a tiny limit; over-limit is `ValueLimit`, never `Malformed`/generation fault; only an impossible line is generation-level | ✓ VERIFIED | `maxDumpLineBytes()` takes no config (`codec.go:50-62`); `readDumpLine` bounded (`codec.go:281-303`); `TestCodec_shrunkLimitKeepsOtherRecords`; overlong-line row in `TestCodec_headerCensus` |
| T-02.5 | 05-02 | A 1 024-byte quote/control-char key with a 61 440-byte value round-trips | ✓ VERIFIED | `TestCodec_escapedMaxKeyRoundTrip` (escape-expanded bound) |
| T-02.6 | 05-02 | A v1.1 dump cut mid-line is a torn-tail generation error; a v1.0 line-boundary cut remains the documented legacy limitation | ✓ VERIFIED | `errTornTail` (`codec.go:266-268`), `errIncompleteDump`; `TestCodec_truncatedGenerationUnreadable`; header-census legacy row; contract stated (`codec.go:214-239`) |
| T-03.1 | 05-03 | Join order `(version, content)`: greater version wins; identical equal version is a no-op; differing equal version resolves remove-wins then `bytes.Compare` | ✓ VERIFIED | `store.join` (`store.go:386-415`); `TestMerge_equalVersionContentLaw`, `TestMerge_equalVersionBranchTable`, `TestProp_mergeSmallSpace` |
| T-03.2 | 05-03 | The law reaches the wire: identical content Acks with no reply; incoming tie-break win Acks; stored tie-break win answers with one unicast `DataReplication`, no Ack | ✓ VERIFIED | `core.applyRemote` (`core.go:1489-1538`); `TestReplication_equalVersionContentLaw` (all three branches) |
| T-03.3 | 05-03 | `ApplyRemote` dirties through `MergeChanged`'s `changed`, never a Conflict/version heuristic | ✓ VERIFIED | `facade.go:40-50`; `TestEngine_applyRemoteEqualVersionContent` (install dirties once, identical redelivery leaves clean) |
| T-03.4 | 05-03 | Explicit same-version/different-content `DataReplication` drives the wire rule; same seed replays identically; cross-node convergence via `NotifyInstalled` | ✓ VERIFIED | `TestReplication_equalVersionSameSeedTrace`; `TestScenario_tiebreakConverges` (announcing and receiving node hold the winner) — both ran green |
| T-03.5 | 05-03 | ADR 0012 records the deterministic deviation, the lost-operation consequence and explicit non-changes | ✓ VERIFIED | `docs/adr/0012-equal-version-tiebreak.md` read: context/decision/consequences/links; names S-HLC-07 and PFXD-01 as non-changes |
| T-04.1 | 05-04 | A dump streams: one `Records()` copy + `SaveStream(fill)` running `EncodeTo`; the five TEST-03 steps are unchanged | ✓ VERIFIED | `scheduler.go:221-238`; `fsops.go:101-117,346-369`; `TestFsops_saveOperationOrder`, `TestFsops_saveOperationOrder_rotation_chain`; bench drives the real path |
| T-04.2 | 05-04 | `EncodeTo` is byte-identical to `Encode`; no write larger than `maxDumpLineBytes()` | ✓ VERIFIED | `TestEncodeTo_byteIdentical`, `TestEncodeTo_maxSingleWrite` (one write per line; joined stream equals `Encode`) |
| T-04.3 | 05-04 | A fill error removes the temp and surfaces the error; the failure matrix keeps its hygiene | ✓ VERIFIED | `fileFSOps.write` cleanup (`fsops.go:101-117`); `TestFsops_failureAtEachStep` fill-failure row; `TestFileStore_saveFailureHygiene` |
| T-04.4 | 05-04 | The port carries `Load(now, parse) LoadResult` with outcome/generation/failures, implemented by the adapter and its doubles | ✓ VERIFIED | `port/dump.go:42,54-110`; adapter `dumpfile.go:127-175`; fakes in `persist/*_test.go`, `persistfile/*_test.go`; all packages compile and pass |
| T-04.5 | 05-04 | The memory model is measured, not guessed: B/op and a sampled peak live; the one `Records()` copy and writer pause stated as residual | ✓ VERIFIED | `bench_test.go:86-141`; `make bench-smoke` printed `B/op`, `dump-bytes`, `peak-live-B`; `OPERATIONS.md:158-175` states the residual pause |
| T-05.1 | 05-05 | Load reports Prompt/Main/Backup/Unusable; the accepted-`.prev` path always names the main candidate; evidence is never deleted | ✓ VERIFIED | `dumpfile.go:127-175`; `TestFileStore_loadOutcomeTable` (7 rows incl. absent-main `fs.ErrNotExist`); stand Prompt/Backup/Backup-absent/Unusable/Unusable-single all pass |
| T-05.2 | 05-05 | A generation failing the oracle is renamed `<path>.corrupt-<UTC>[-N]` before any save; only regular files; same-second collisions survive; rotation never overwrites | ✓ VERIFIED | `preserve` (`dumpfile.go:196-233`), `Lstat` regularity; `TestFileStore_loadBackupPreservesMain`, `TestFileStore_preserveCollision`, `TestFileStore_loadNonRegularCandidate` |
| T-05.3 | 05-05 | After a Backup start the first save's rotate is a no-op; `.prev` stays byte-identical | ✓ VERIFIED | `TestFileStore_loadBackupPreservesMain` compares `.prev` bytes before/after `Save` |
| T-05.4 | 05-05 | Every temp carries `.tmp~`, whose `~` is outside the snapshot charset, so no legal snapshot collides and the sweep never keys on a data prefix | ✓ VERIFIED | `orphanTempMarker`/`tempPattern` (`dumpfile.go:43-51`); `CreateTemp(dir, tempPattern)` (`fsops.go:102`); snapshot charset rejects `~` (`snapshot.go:87-114`); `TestFileStore_nameValidation` |
| T-05.5 | 05-05 | The sweep removes only old `.tmp~` regular files (both dirs), keeps young ones, protects main/`.prev`/`.corrupt-*`/snapshots, reports count and first error | ✓ VERIFIED | `sweepOrphans`/`protectedDataName`/`sweepEntry` (`dumpfile.go:246-310`); `TestSweep_removesOnlyOldTemps`, `TestSweep_protectsDataNames`, `TestSweep_reportsUnreadableDir`; stand Prompt test proves the old orphan is removed with count 1 |
| T-05.6 | 05-05 | A newly created dump directory is synced at its parent before the temp write; existing directories do not (five-step order unchanged) | ✓ VERIFIED | `ensureDir` (`dumpfile.go:319-332`); `TestEnsureDir_newDirectorySyncsParent`, `TestEnsureDir_existingDirectoryNoParentSync`; order test unchanged |
| T-05.7 | 05-05 | Directory fsync tolerates ENOTSUP/EINVAL; file-data fsync keeps every error; a save error carries `port.FailureClass` reachable via `errors.Is` | ✓ VERIFIED | `tolerateDirSyncErr` called only from `syncDir` (`fsops.go:175-182,220-225`); `classedSaveError` wraps cause (`saveerror.go:12-46`); `TestSyncDirTolerant`, `TestClassifySaveError`, failure matrix |
| T-06.1 | 05-06 | `Open` stores a `LoadStats` report and keeps `(*Engine, int)`; `New` engines report zero | ✓ VERIFIED | `loader.go:53-72`; `engine.go:181-185`; `TestEngine_loadStatsMapsResult` |
| T-06.2 | 05-06 | Dirty after Prompt/Backup/Unusable (Snapshot right after flushes and decodes back; first rotation never displaces the good generation); Main — even with discards — starts clean | ✓ VERIFIED | `startLoadDirty` (`loader.go:75-90`); `TestEngine_startDirty` (all five cases, snapshot decode assertions); `TestFileStore_loadBackupPreservesMain` |
| T-06.3 | 05-06 | Every `markDirty()` wakes an idle dumper via the `changed` nudge (incl. the restore mid-error path); one nudge per write path; a clean engine stays clean | ✓ VERIFIED | `markDirty` nudges (`confirm.go:139-147`); `await` nudges once; mid-error path calls `markDirty` (`snapshot.go:133-137`); `TestEngine_markDirtyNudge` (exactly one dump at Debounce, then quiet). The mid-error branch is inspection-only because the real store's error surface makes it unreachable (value-limit/NotFound are skips) — nothing human can trigger either |
| T-06.4 | 05-06 | Records skipped by a shrunken limit are counted by class, logged once with count+limit, exposed as a metric, and left on disk through two rotations | ✓ VERIFIED | `stats.Decoded.ValueLimit` (loader/engine), `main.go:592-596`, `metrics.go:252-263`; `TestEngine_skipEvidenceWindow` (byte-identical after load, in `.prev` after one write, gone after two); `TestStand_persistShrunkLimit` |
| T-06.5 | 05-06 | Binary logs exactly one WARN for Backup (path+cause incl. absent main), one ERROR for Unusable (one group per failed candidate), nothing for Prompt/Main, one INFO for sweep count, one WARN for sweep error | ✓ VERIFIED | `logLoadStats`/`logBackupFallback`/`logUnusableGenerations` (`main.go:606-664`); stand tests assert exact message counts; sweep-error WARN is the same switch with `SweepErr` produced/tested at adapter+mapping level (`TestSweep_reportsUnreadableDir`, `TestEngine_loadStatsMapsResult`) |
| T-06.6 | 05-06 | `supkv_dump_load_total{outcome}` and `supkv_dump_records_discarded_total{class}` render every label at zero and the load counts after a load | ✓ VERIFIED | `metrics.go:223-263`; `TestMetrics_families` (all zero lines), `TestMetrics_values` (backup=1, four discard classes) |
| T-07.1 | 05-07 | `internal/app/persist` owns no wall clock except the two defaults in `normalized()` | ✓ VERIFIED | `grep -E 'time\.(Now|NewTimer|After|Until|Since)'` non-test finds only `engine.go:67,70`; every read/arm goes through `cfg.Now`/`cfg.After` |
| T-07.2 | 05-07 | The scheduler arms debounce/deadline/retry from `Config.After`; re-arming replaces the channel; no channel is created inside the select | ✓ VERIFIED | `run` loop (`scheduler.go:109-137`), `rearm` (`scheduler.go:176-199`); no `time.After` in the select; `TestEngine_injectedClock`, `TestEngine_confirmDebounceTiming` |
| T-07.3 | 05-07 | Ladder 100 ms → 200 → 400 → 800 → 1.6 s → 3.2 s → 5 s cap; while pending no debounce/deadline is armed; success resets and covers the waiters | ✓ VERIFIED | `nextRetry`/`resetRetry` (`scheduler.go:271-291`); `TestEngine_backoffLadder` asserts the exact virtual instants, the "during backoff" write and the reset to 100 ms |
| T-07.4 | 05-07 | Each failed attempt invokes `OnDumpFailure(err, class)` exactly once; `DumpStats` counts DiskFull/Permission/Other beside total/errors | ✓ VERIFIED | `handleDumpFailure` (`scheduler.go:248-266`); `TestEngine_dumpFailureClasses`; `TestEngine_backoffLadder` asserts one callback per failed attempt |
| T-07.5 | 05-07 | Health stays SERVING under persistent dump failure; the EACCES stand asserts one ERROR, the class label, a bounded attempt rate and recovery | ✓ VERIFIED | No health path touched by dump code (`grpcapi` unchanged for this); `TestStand_persistDumpFailure` passed (permission class + cause, SERVING, 1 s window bounded, metric >=1, recovery write) |
| T-07.6 | 05-07 | The simulator exercises dump→confirmation→replication-notify incl. a failure window and shrunk-limit skip; two same-seed runs are identical; the same seam proves cross-node equal-version convergence | ✓ VERIFIED | `Cluster.NotifyInstalled` (`simnet.go`); `TestScenario_dumpNotify` (confirm/notify order, ladder offsets, failure still notifies, shrunk limit reported, two same-seed runs identical) — ran green |
| T-08.1 | 05-08 | `ConfirmTimeout` defaults to 10 s; an uncovered write returns `ErrUnconfirmed`; the record stays installed/dirty; not reported failed or durable | ✓ VERIFIED | `defaultConfirmTimeout` (`engine.go:18`), `await` timeout branch (`confirm.go:98-130`); `TestEngine_confirmationTimeout` asserts the exact deadline, sentinel, installed record and counter |
| T-08.2 | 05-08 | Single-claim resolution: capture never claims; the first resolver (success/failure/exit/timeout) claims under the mutex; a hung dump cannot block the timeout; the timeout removes its waiter | ✓ VERIFIED | `waiter.claimed` + every resolver's claim (`confirm.go:207-285`); `TestEngine_timeoutThenDumpSuccess` (late success skips the claimed waiter, no double close), `TestEngine_waiterRemoved` |
| T-08.3 | 05-08 | `Flush()` = `FlushTimeout(0)`; both bound send and reply; `Snapshot` uses it; `Restore` returns its partial list on timeout with `ErrUnconfirmed` distinguishable from `ErrDumpFailed` | ✓ VERIFIED | `engine.go:234-294`, `snapshot.go:36-42,89-115`; `TestFlush_timeout` (exact instants), `TestSnapshot_timeout`, `TestRestore_timeoutReturnsPartial` |
| T-08.4 | 05-08 | Shutdown is bounded: `FlushTimeout(shutdownGrace)`, cancel, `engineDone` wait at most the grace, WARN instead of hanging | ✓ VERIFIED | `shutdown` calls `FlushTimeout(shutdownGrace)` then `awaitEngineStop(node.engineDone, shutdownGrace)` (`main.go:955-975`); `awaitEngineStop` preserves the three branches (`main.go:977-987`); **`TestAwaitEngineStop` exercises all three, including a never-written channel with a 25 ms grace: returns ≥ grace, ≤ 2 s, WARN text asserted — ran PASS under `-race`** |
| T-08.5 | 05-08 | A timed-out write still replicates: Set/Delete notify on `ErrDumpFailed` or `ErrUnconfirmed`; `mapError` maps `ErrUnconfirmed` to Unavailable | ✓ VERIFIED | `server.go:218-221,266-269,441-445`; `TestServer_unconfirmedSetMapsAndNotifies`, `TestServer_unconfirmedDeleteMapsAndNotifies` |
| T-08.6 | 05-08 | The frozen blocking test stays green with its 200 ms / 1 s windows unedited | ✓ VERIFIED | `engine_blocking_test.go:24-25` byte-identical to base; `TestEngine_blockingSaveHoldsSet` passes |
| T-09.1 | 05-09 | Five internal roles separated: loader/composition root, store facade, dump scheduler, confirmation accounting, snapshot orchestration | ✓ VERIFIED | `loader.go`, `facade.go`, `scheduler.go`, `confirm.go`, `snapshot.go` (+ `engine.go` composition); `flush.go` only carries `flushRequest` |
| T-09.2 | 05-09 | `Engine`'s exported method set/signatures unchanged; consumers compile unedited | ✓ VERIFIED | `go build ./...` green; exported-name diff vs base adds only the phase's planned `FlushTimeout`, `EncodeTo`, `DecodeStats`, `LoadStats`, `Total`; nothing removed; the split added none |
| T-09.3 | 05-09 | Confirmation role owns waiters/dirty/epoch/changed/closed/runDone behind its mutex; shares one unexported `*dumpStats` for `unconfirmed` | ✓ VERIFIED | `confirmer` (`confirm.go:42-68`); `stats.unconfirmed.Add(1)` (`confirm.go:128`); capture-before-encode order preserved (`scheduler.go:222-228`) |
| T-09.4 | 05-09 | Scheduler owns ladder/channels/`dumpStats`; `New` and `Open` build one holder shared with the confirmer; facade thin; loader owns Open/mergeRecords/LoadStats | ✓ VERIFIED | `New` (`engine.go:163-175`) creates `stats` once; `Open` through `New` (`loader.go:61`); `DumpStats()` value copy (`scheduler.go:306-317`); `facade.go` forwards |
| T-09.5 | 05-09 | Pure move: suite, lint and darwin green; no behaviour change, no new interface, no new exported name | ✓ VERIFIED | `make test`, `make lint`, darwin build/vet green; `grep 'type .* interface' internal/app/persist/*.go` = none; export diff shows only the phase's other planned names |
| T-10.1 | 05-10 | OPERATIONS states the single limit, the non-positive exit-2 rule and keeps the 64 408 boundary arithmetic | ✓ VERIFIED | `OPERATIONS.md:64-92`; `TestOperationsDoc` anchors pass |
| T-10.2 | 05-10 | Monitoring lists the four dump/load families with the every-label-at-zero rule | ✓ VERIFIED | `OPERATIONS.md:238-278`; anchors (`operations_test.go:94-97`); names exist in `metrics.go` |
| T-10.3 | 05-10 | Lifecycle documents the ladder, health staying SERVING, `Unavailable` for unconfirmed, bounded shutdown | ✓ VERIFIED | `OPERATIONS.md:280-316` |
| T-10.4 | 05-10 | Load-outcomes section documents Prompt/Main/Backup/Unusable, one-WARN/one-ERROR, `.corrupt-<UTC>[-N]`, `preserve_error` hazard, the one-hour marker-only sweep | ✓ VERIFIED | `OPERATIONS.md:94-156`; `TestOperationsDoc` (`## Load outcomes` anchor) |
| T-10.5 | 05-10 | Memory model states one `Records()` copy + one line + 64 KiB writer and names the residual writer pause | ✓ VERIFIED | `OPERATIONS.md:158-175`; `## Memory model` anchor; benchmark prints the supporting figures |
| T-10.6 | 05-10 | Anchors cover the new sections and the file still opens with the trust model | ✓ VERIFIED | `operations_test.go` asserts `## Trust model` is the first heading (`:115`) and lists the new sections/families |
| T-11.1 | 05-11 | Exactly the eight named registry rows change (at the plan's commit) | ✓ VERIFIED | Diff base..`1f79eef` (05-11 commit) = exactly `S-PAR-05, S-PERS-08, S-PERS-09, S-ALG-REP-03, S-HLC-07, S-DM-04, S-DM-05, S-DM-08`; the later 8 re-pointed rows are the documented IN-01 review fix (`1f79eef..HEAD`), not plan drift |
| T-11.2 | 05-11 | No other row's status/links change; the preamble mentions ADR 0012 | ✓ VERIFIED | Per-range row sets above; preamble line 26 names ADR 0012 while the 14.1-14.11 mapping stays |
| T-11.3 | 05-11 | Every changed row cites a current code path/test/ADR; nothing says TBD/future/unchecked | ✓ VERIFIED | Spot-checked citations resolve; `grep -E 'TBD|future|unchecked'` over rows = none; statuses are the closed enum |
| T-11.4 | 05-11 | Status vocabulary stays the closed enum | ✓ VERIFIED | Field 4 values: implemented / superseded-by-amendment / fixed-in-v1.1(...) / consumer-zone / deferred-v2(...) / accepted-deviation(...) |
| T-12.1 | 05-12 | Both P-01 rows flip red→green; other five stay green; runner and frozen tests byte-identical | ✓ VERIFIED | `scripts/red-expectations.tsv` rows; `git diff BASE` on runner + both frozen tests is empty (re-checked after `2b281a9`) |
| T-12.2 | 05-12 | The full red gate is 7/7 with both mutants killed | ✓ VERIFIED | Re-ran `SUPKV_RED_TESTS=1 bash scripts/red-runner` on the current tree → exit 0, "7 expectation row(s) checked, 0 error(s)", both mutants killed, evidence `20261122T191142Z` |
| T-12.3 | 05-12 | `make lint`, `make test`, darwin build/vet and the bench smoke are green on the committed tree | ✓ VERIFIED | All re-run after `2b281a9`: lint exit 0; `go test -race -cover ./...` exit 0 (cmd/sup-kv 5.8 s, coverage 1.2%; repl 173.0 s; rest cached-green); darwin build/vet exit 0; `make bench-smoke` exit 0 |
| T-12.4 | 05-12 | No test, bound or runner line weakened; the only diff outside the phase's code is the TSV flip | ✓ VERIFIED | Frozen files untouched; the new `shutdown_internal_test.go` is additive (three assertions incl. time bounds and WARN text); the mechanical `persist.Encode/Decode` signature edits in `decode_negative_test.go` keep every assertion; no tolerance/skip added; TSV diff is 2+2 lines |

### Deferred Items

No must-have gap was deferred. Two review dispositions are informational:

| Item | Owner | Note |
| ---- | ----- | ---- |
| `EncodeTo`'s `valueMaxBytes <= maxDumpValueBytes` precondition (IN-04) | Phase 7 (OBSV-02 one-point validation) | Documented in `persist/doc.go`; the shipped binary enforces the ceiling at startup (64 409 exits 2), so no supported writer can emit an over-bound line |
| `TestCore_keyCountCostFlat` wall-clock ratio flake | Phase 8 (PERF) | Pre-existing, load-sensitive; did not fire in this verification's `make test` |

## Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/app/storage/store.go` | `ValueMaxBytes`, `Config.ValueLimit`, `Store.ValueMaxBytes`, `RecordsEqual`, `MergeChanged`, `join` | ✓ VERIFIED | All present; `defaultValueMaxBytes` gone; single production literal |
| `internal/app/persist/codec.go` | config-driven `Encode`/`Decode`, `EncodeTo`, census, four-class stats, format-bound reader | ✓ VERIFIED | `EncodeTo` + wrapper; `record_count`; `DecodeStats`; no `json.Decoder`; `maxDumpLineBytes()` |
| `internal/app/persist/engine.go` | composition + public write/read/`Flush`/`LoadStats`; `Config` with `ConfirmTimeout`/`Now`/`After`/`OnDumpFailure` | ✓ VERIFIED | 294 lines; five roles composed |
| `internal/app/persist/loader.go` | `Open`, `mergeRecords`, `LoadStats` construction | ✓ VERIFIED | Open decodes with `ValueLimit()` before the store; frozen signature |
| `internal/app/persist/facade.go` | thin forwards + `ApplyRemote` | ✓ VERIFIED | `ApplyRemote` dirties only on `changed` |
| `internal/app/persist/scheduler.go` | `Run`, `runDump`, `flushDump`, `rearm`, `recordDump`, `dumpStats`, ladder | ✓ VERIFIED | Streaming dump, ladder, class counters |
| `internal/app/persist/confirm.go` | `waiter`, `await`, `confirmWaiters`, `failWaiters`, `sweepClosed` | ✓ VERIFIED | Single-claim protocol |
| `internal/app/persist/snapshot.go` | `Snapshot`, `Restore`, replay helpers | ✓ VERIFIED | Bounded flush through the public seam |
| `internal/app/persist/errors.go` | `ErrUnconfirmed` | ✓ VERIFIED | Plus existing sentinels |
| `internal/app/persist/flush.go` | shared `flushRequest` type | ✓ VERIFIED | Snapshot vs explicit origin |
| `internal/app/persist/encode_stream_internal_test.go` | `TestEncodeTo_byteIdentical`, `TestEncodeTo_maxSingleWrite` | ✓ VERIFIED (renamed path) | Plan named `encode_stream_test.go`; the strict `testpackage` lint mandates `_internal_test.go` (documented deviation in 05-04-SUMMARY). Both tests exist and pass; the tool's artifact check flags only the path name |
| `cmd/sup-kv/shutdown_internal_test.go` (re-verification addition) | `TestAwaitEngineStop` covering clean-cancel, abnormal-error and wedged-expiry branches | ✓ VERIFIED | Added by `2b281a9`; three subtests PASS under `-race`; expiry asserts return ≥ grace, ≤ 2 s, WARN text |
| `internal/port/dump.go` | `SaveStream`, `Load(...) LoadResult`, outcome types, `FailureClass`/`ClassedError` | ✓ VERIFIED | Full contract |
| `internal/persistfile/dumpfile.go` | outcomes, preservation, sweep, marker, `SaveStream`, ordered save | ✓ VERIFIED | `Load` main→`.prev`; `preserve`; `sweepOrphans`; `saveAtomically` |
| `internal/persistfile/fsops.go` | five-step seam with buffered fill; parent sync; dir-fsync tolerance | ✓ VERIFIED | 64 KiB `bufio`; cleanup on failure |
| `internal/persistfile/saveerror.go` | errno → class mapping preserving `errors.Is` | ✓ VERIFIED | ENOSPC/EACCES/fs.ErrPermission |
| `internal/persistfile/snapshot.go` | validated names, snapshot copy under the save mutex | ✓ VERIFIED | charset `[A-Za-z0-9._-]` (rejects `~`) |
| `cmd/sup-kv/main.go` | flag default from the constant, range/ceiling checks, load logs, skip WARN, OnDumpFailure, bounded shutdown | ✓ VERIFIED | `exitUsage = 2`; `shutdownGrace`; `awaitEngineStop` extracted, behavior identical |
| `internal/metrics/metrics.go` | four new families with label enums | ✓ VERIFIED | `loadFamilies`, `discardClassFamilies`, failure classes, unconfirmed |
| `internal/app/repl/core.go` | content-aware equal-version branch | ✓ VERIFIED | Consults `storage.RecordsEqual`, re-reads stored record for the reply verdict |
| `internal/app/internal/simnet/simnet.go` | `Cluster.NotifyInstalled` | ✓ VERIFIED | Lookup + NotifyLocalWrite + route + write-log capture |
| `docs/adr/0012-equal-version-tiebreak.md` | recorded decision | ✓ VERIFIED | Full slim-MADR |
| `docs/OPERATIONS.md` | limits, load outcomes, memory model, monitoring, lifecycle | ✓ VERIFIED | All mandated sections; trust model first |
| `docs/SPEC-DEVIATIONS.md` | eight plan rows + eight review-fix rows | ✓ VERIFIED | Row-set reconciliation above |
| `scripts/red-expectations.tsv` | two P-01 rows green, five others green | ✓ VERIFIED | 7 data rows, all green |

## Key Link Verification

All 32 plan-declared links were checked manually. The `gsd_run query verify.key-links` heuristic reports `all_verified: false` for most plans because its matcher looks for the target *filename* in the source file ("Target not referenced in source"), while these Go links are package imports (`internal/app/storage`) and prose `via` clauses; the same tool reports `true` only for the doc rows whose text literally names the file. The manual evidence below is authoritative.

| # | From | To | Via (abridged) | Status | Details |
| - | ---- | -- | -------------- | ------ | ------- |
| 1 | `persist/loader.go` (was engine.go) | `storage/store.go` | `Open` decodes through `Config.ValueLimit()` before the store; `NewStore` normalizes through the same helper | WIRED | `loader.go:42-46,59`; `store.go:131-146` |
| 2 | `persist/codec.go` | `storage/store.go` | callers pass the limit; oversize encode is the `storage.ErrValueTooLarge` tripwire | WIRED | `codec.go:126-153`; `storage/store.go:21` |
| 3 | `cmd/sup-kv/main.go` | `storage/store.go` | flag default and ceiling effective value are `storage.ValueMaxBytes` | WIRED | `main.go:362,541-555`; CLI `--help` default 61440 |
| 4 | `persist/codec.go` | `persist/loader.go` (was engine.go) | decode closure stores stats; `Open` returns `stats.Total()` | WIRED | `loader.go:44-52,62-72` |
| 5 | `persist/codec.go` | `persistfile/dumpfile.go` | parse oracle is `Decode` at the configured limit; generation faults advance the chain | WIRED | `loader.go:45-53`; `dumpfile.go:150-154` |
| 6 | `storage/store.go` | `repl/core.go` | `RecordsEqual` is the one content comparison; core Acks equal version only after consulting it | WIRED | `core.go:1494-1533`; `store.go:184-186` |
| 7 | `repl/core.go` | `persist/facade.go` | tie-break install reaches `ApplyRemote`; core re-reads the stored record for Ack vs reply | WIRED | `core.go:1502-1533`; `facade.go:40-50` |
| 8 | `persist/facade.go` | `storage/store.go` | `MergeChanged` returns the installation decision; `Merge` stays the 3-result wrapper | WIRED | `facade.go:40-41`; `store.go:318-338` |
| 9 | `persist/scheduler.go` | `persistfile/dumpfile.go` | engine fill runs on the dumper goroutine through the adapter's buffered temp writer; fill error returns via the port | WIRED | `scheduler.go:226-228`; `fsops.go:101-117` |
| 10 | `port/dump.go` | `persist/loader.go` | `Load` returns the outcome/causes; `Open` records them; signature unchanged | WIRED | `port/dump.go:42,98-110`; `loader.go:53-72` |
| 11 | `persistfile/fsops.go` | `persistfile/fsops_test.go` | recording fake records the `write` step with its fill and delegates to the real implementation | WIRED | `fsops_test.go:69-72`; `TestFsops_saveOperationOrder` |
| 12 | `persistfile/dumpfile.go` | `persist/loader.go` | the engine's parse oracle decides a candidate's fate; the adapter preserves and reports | WIRED | `dumpfile.go:150-154`; `loader.go:45-53` |
| 13 | `persistfile/dumpfile.go` | `persistfile/fsops.go` | preservation uses `Lstat`/`Rename` directly; the save path's parent sync goes through the seam | WIRED | `dumpfile.go:206-244,319-331`; `fsops.go:175-182` |
| 14 | `persistfile/saveerror.go` | `port/dump.go` | errno knowledge stays in the adapter; the core receives `port.FailureClassOf(err)` | WIRED | `saveerror.go:32-45`; `port/dump.go:149-157` |
| 15 | `persist/loader.go` | `port/dump.go` | `Open` maps `LoadResult` into the engine's `LoadStats` | WIRED | `loader.go:62-70`; `engine.go:147-155` |
| 16 | `cmd/sup-kv/main.go` | `persist/engine.go` | `startEngine` reads `LoadStats()` once after open and decides one WARN/ERROR/INFO | WIRED | `main.go:590-626` |
| 17 | `metrics/metrics.go` | `persist/engine.go` | `Sources.LoadStats` is wired to `engine.LoadStats`; families render at zero | WIRED | `main.go:802`; `metrics.go:191,214-263` |
| 18 | `persist/engine.go` | `simnet.go` | tests bind `Config.Now`/`After` to virtual time; the engine shares `Cluster.Store(0)` and announces via `NotifyInstalled` | WIRED | `sim_dump_test.go`; `simnet.go`; `TestScenario_dumpNotify` green |
| 19 | `persist/scheduler.go` | `port/dump.go` | `FailureClassOf` extracts the adapter class; the core never imports `syscall` | WIRED | `scheduler.go:248-265`; `port/dump.go:112-157` |
| 20 | `cmd/sup-kv/main.go` | `persist/engine.go` | main wires `OnDumpFailure` to one `slog.Error` with the class | WIRED | `main.go:586-589` |
| 21 | `persist/engine.go` | `grpcapi/server.go` | RPC failure branches notify for either unconfirmed class; `mapError` is total over it | WIRED | `server.go:218-221,266-269,441-445` |
| 22 | `persist/engine.go` | `cmd/sup-kv/main.go` | `FlushTimeout(shutdownGrace)` and the bounded `engineDone` wait | WIRED | `main.go:955-987`; both bounds now behaviorally tested (`TestFlush_timeout`, `TestAwaitEngineStop`) |
| 23 | `persist/engine.go` | `persist/confirm.go` | engine composes one confirmer; writes register/wait through it; the scheduler resolves | WIRED | `engine.go:93-99,163-175`; `confirm.go:98-130,207-250` |
| 24 | `persist/scheduler.go` | `persist/confirm.go` | capture-before-encode and the single-claim resolution cross the two roles | WIRED | `scheduler.go:221-238`; `confirm.go:187-250` |
| 25 | `persist/snapshot.go` | `persist/engine.go` | `Snapshot` forces the flush through the public method; `Restore` reuses the facade | WIRED | `snapshot.go:36-42,89-115` |
| 26 | `docs/OPERATIONS.md` | `cmd/sup-kv/operations_test.go` | each mandated statement has an anchor literal | WIRED | `operations_test.go:46,87-97`; `TestOperationsDoc` green |
| 27 | `docs/OPERATIONS.md` | `internal/metrics/metrics.go` | every family in the monitoring list exists in the exporter | WIRED | Names grepped in `metrics.go:242,253,429,438` |
| 28 | `docs/SPEC-DEVIATIONS.md` | `storage/store.go` | rows cite `ValueMaxBytes`/`ValueLimit`/`Store.ValueMaxBytes`/`MergeChanged` | WIRED | Rows 38/39/42/232; symbols present |
| 29 | `docs/SPEC-DEVIATIONS.md` | `persistfile/dumpfile.go` | `S-PERS-08`/`S-PERS-09` cite the load/preserve path and the load metric | WIRED | Rows 166/167; code present |
| 30 | `docs/SPEC-DEVIATIONS.md` | `docs/adr/0012-...md` | `S-ALG-REP-03` and `S-HLC-07` link the ADR | WIRED | Rows 55/127 |
| 31 | `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go` | flipped row names the frozen stand test | WIRED | Red runner row `TestRed_P01_valueLimit` OK |
| 32 | `scripts/red-expectations.tsv` | `internal/persistfile/red_value_limit_test.go` | flipped row names the frozen engine test | WIRED | Red runner row `TestRed_P01_valueLimitEngine` OK |

## Data-Flow Trace (Level 4)

This phase has no rendered values; the equivalent is the dump/load/metrics data path. Every traced value ends in a real store, a real file or a real counter — no static fallback or mock on a production path.

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Dump cycle (`scheduler.runDump`) | `records` | `store.Records()` deep copy of the live map | Yes — 10 000-key bench writes 5.1 MB | ✓ FLOWING |
| Dump encode (`EncodeTo`) | line bytes | sorted real records + `RecordChecksum` | Yes — determinism test + bench | ✓ FLOWING |
| Load (`persistfile.Load` → `Decode`) | `LoadResult.Dump` / records | real `os.ReadFile` of main/`.prev` | Yes — restart tests (`TestEngine_endToEndRestart`, frozen P-01) | ✓ FLOWING |
| Load report (`LoadStats`) | outcome/failures/decoded/sweep | adapter `LoadResult` → `Open` | Yes — stand process tests read the emitted log lines | ✓ FLOWING |
| Load/dump metrics | `LoadStats`, `DumpStats` | `engine.LoadStats`/`DumpStats` seams | Yes — `TestMetrics_values` by-effect samples; stand scrape | ✓ FLOWING |
| Tie-break reply (`applyRemote`) | stored record | `store.Lookup` **after** the merge | Yes — `TestReplication_equalVersionContentLaw` | ✓ FLOWING |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Whole-tree build | `go build ./...` | exit 0 | ✓ PASS |
| Strict lint | `make lint` | `config verify` + `golangci-lint run ./...` exit 0 | ✓ PASS |
| Full race suite | `make test` (`go test -race -cover ./...`) | exit 0; cmd/sup-kv ran 5.8 s (coverage 1.2%, includes `TestAwaitEngineStop`), repl 173.0 s, persist 95.5%, persistfile 91.2%, storage 100%, metrics 69.4%; no flake fired | ✓ PASS |
| Darwin cell | `GOOS=darwin GOARCH=arm64 go build ./... && go vet ./...` | exit 0 both | ✓ PASS |
| Red gate (current tree) | `SUPKV_RED_TESTS=1 bash scripts/red-runner` | exit 0; 7/7 rows OK; both mutants killed; evidence `20261122T191142Z` | ✓ PASS |
| Bounded engine stop (all branches) | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAwaitEngineStop$' -v` | 3/3 subtests PASS (0.03 s); expiry branch asserts ≥ 25 ms grace, ≤ 2 s, WARN text | ✓ PASS |
| Phase core packages, uncached | `go test -race -count=1 ./internal/app/persist/ ./internal/persistfile/ ./internal/app/storage/ ./internal/metrics/` | all `ok` | ✓ PASS |
| Tie-break + simulator, uncached | `go test -race -count=1 ./internal/app/repl/ -run 'TestScenario_dumpNotify|TestScenario_tiebreakConverges|TestReplication_equalVersion'` | all PASS | ✓ PASS |
| Process load outcomes | `go test -race -count=1 ./internal/stand/ -run TestStand_persist -v` | 7/7 PASS, 0 skips | ✓ PASS |
| Unconfirmed RPC mapping | `go test -race -count=1 ./internal/grpcapi/ -run 'TestServer_unconfirmed'` | both PASS | ✓ PASS |
| Bench smoke | `make bench-smoke` | exit 0; p50/p99, B/op, dump-bytes, peak-live-B printed | ✓ PASS |
| Single production literal | `grep -rn 61440 --include=*.go . \| grep -v _test.go` | only `internal/app/storage/store.go:21` | ✓ PASS |
| No streaming decoder poison | `grep -c json.NewDecoder internal/app/persist/codec.go` | 0 | ✓ PASS |
| No wall clock in persist | `grep -E 'time\.(Now\|NewTimer\|After\|Until\|Since)'` non-test | only the two defaults in `normalized()` | ✓ PASS |
| CLI: zero limit | built binary `--value-max-bytes=0` | exit 2, "value-max-bytes must be positive: --value-max-bytes=0" | ✓ PASS |
| CLI: over ceiling | built binary `--value-max-bytes=64409` | exit 2, unchanged datagram-ceiling message naming 65 507 | ✓ PASS |
| Vet/format on changed files | `go vet ./cmd/sup-kv/`; `gofmt -l cmd/sup-kv/main.go cmd/sup-kv/shutdown_internal_test.go` | vet exit 0; gofmt lists nothing | ✓ PASS |

## Probe Execution

No `scripts/*/tests/probe-*.sh` exists and no plan declares a probe script. The phase's declared acceptance runner is the red gate, re-executed above on the current tree (`scripts/red-runner`, 7/7 rows + mutant stage green). The latest evidence directory is `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261122T191142Z/`.

## Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
| --------- | ---------- | ------ | ------- | -------- | --------------- | ------- |
| `internal/persistfile/red_value_limit_test.go` | PERSIST-01 | 1 | 0 (env-gated; runner opens it) | No | Behavioral (write→restart→full value) | PASS |
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
| `internal/metrics/metrics_test.go` | PERSIST-02/05/08/10 | 4 | 0 | No | Value (by-effect samples) | PASS |

**Disabled tests on requirements:** 0 (the env-gated red tests are the designed mechanism and ran via the red runner; they are not disabled). **Circular patterns:** 0 — fixtures carry literal records/checksums; no test generates expected values by running the system under test. **Assertion strength:** `TestAwaitEngineStop`'s expiry subtest asserts a lower time bound (≥ grace), an upper bound (≤ 2 s) and the exact WARN substring, so a removed timeout, an immediate return or a dropped WARN each fail. **Insufficient assertions:** 0.

## Requirements Coverage

| Requirement | Source Plan(s) | Status | Evidence |
| ----------- | -------------- | ------ | -------- |
| PERSIST-01 | 01, 11, 12 | ✓ SATISFIED | Single source + config path; exact-limit restart at 64 000 and 8; invalid exit 2; frozen tests green; CLI probes |
| PERSIST-02 | 05, 06, 10, 11, 12 | ✓ SATISFIED | Four outcomes, preservation, one WARN/one ERROR, nonzero unusable metric; docs + stand tests |
| PERSIST-03 | 02, 12 | ✓ SATISFIED | `record_count` census, torn-tail fault, per-record isolation, four classes |
| PERSIST-04 | 06, 12 | ✓ SATISFIED | Dirty after Prompt/Backup/Unusable, snapshot decodes back, `.prev` survives first rotation, Main clean |
| PERSIST-05 | 07, 10, 12 | ✓ SATISFIED | 100 ms→5 s ladder, one ERROR+class per attempt, health SERVING, docs; process EACCES test |
| PERSIST-06 | 04, 10, 12 | ✓ SATISFIED | Streaming `EncodeTo`/`SaveStream`, max-single-write bound, measured memory model + documented residual |
| PERSIST-07 | 03, 11, 12 | ✓ SATISFIED | Deterministic `(version, content)` law in store and core, ADR 0012, same-seed replay + convergence |
| PERSIST-08 | 08, 12 | ✓ SATISFIED | Writes/snapshots/`FlushTimeout` bounds tested; `mapError`/notify tested; **the `engineDone` grace branch is now exercised by `TestAwaitEngineStop`** |
| PERSIST-09 | 05, 12 | ✓ SATISFIED | Marker-only age-bounded sweep, parent sync on create, ENOTSUP/EINVAL dir-sync tolerance, `fsync_darwin.go` absent |
| PERSIST-10 | 01, 02, 06, 12 | ✓ SATISFIED | `ValueLimit` class, skip count + one WARN + metric, evidence window through two rotations |
| PERSIST-11 | 04, 09, 12 | ✓ SATISFIED | Five roles, public API stable, port returns outcome/cause, no new interface |
| PERSIST-12 | 07, 12 | ✓ SATISFIED | `Now`/`After` seam; no wall clock in the package; simulator drives dump↔notify and the tie-break |

No ORPHANED requirements: all twelve PERSIST IDs are mapped to Phase 5 in `.planning/REQUIREMENTS.md` and claimed by at least one plan.

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX` in any file modified by this phase | — | None |
| — | — | No `TODO`/`HACK`/`PLACEHOLDER`/stub markers in the phase's production files | — | None |
| — | — | No empty implementations; `return []Conflict{...}`/`return []Family{...}` hits are non-empty literals | — | None |
| — | — | No raw `console.log`/`fmt.Println` on a production path | — | None |
| — | — | No new `//nolint` without a named linter + reason; `.golangci.yml` byte-unchanged. The re-verification test's single `//nolint:paralleltest` names the linter and the reason (subtests swap process-global log writers) | — | None |

`math/rand` appears only as `crypto/rand` for the replication run ID (`core.go:402`) — it never decides a tie-break.

## Decision Coverage

All 12 trackable CONTEXT.md decisions (D-01…D-12) are honored by shipped artifacts (`check.decision-coverage-verify`: total 12, honored 12, not_honored []).

## Human Verification Required

None. The single item from the previous pass (bounded-shutdown abandonment under a wedged dumper) is now covered by the named test `TestAwaitEngineStop`, which exercises the expiry branch through the same helper `shutdown()` calls. Because this is an infrastructure/foundation phase with no remaining behaviour-unverified truth and no user-facing element, all acceptance criteria are verified programmatically.

## Gaps Summary

No gaps. All 12 requirements have shipped implementations wired end to end, all five roadmap success criteria are met, and the previously flagged bounded-shutdown clause is now behaviourally proven. The two review-driven registry re-points and the `encode_stream_internal_test.go` filename are documented, authorized deviations — not drift. Re-verification round: the change is confined to `cmd/sup-kv` (helper extraction plus an additive test), every gate was re-run green on the current tree, and the frozen red-gate artifacts are still byte-identical to the phase base.

---

_Verified: 2026-11-22T19:13:03Z_
_Verifier: Claude (gsd-verifier)_
