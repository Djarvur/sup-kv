---
phase: 04-protocol
verified: 2026-11-23T20:41:49Z
status: passed
score: 33/34 must-haves verified
covered_files:
  - .planning/phases/03-transport/03-REVIEW-DISPOSITION.md
  - .planning/phases/04-protocol/04-01-PLAN.md
  - .planning/phases/04-protocol/04-01-SUMMARY.md
  - .planning/phases/04-protocol/04-02-PLAN.md
  - .planning/phases/04-protocol/04-02-SUMMARY.md
  - .planning/phases/04-protocol/04-03-PLAN.md
  - .planning/phases/04-protocol/04-03-SUMMARY.md
  - .planning/phases/04-protocol/04-04-PLAN.md
  - .planning/phases/04-protocol/04-04-SUMMARY.md
  - .planning/phases/04-protocol/04-05-PLAN.md
  - .planning/phases/04-protocol/04-05-SUMMARY.md
  - .planning/phases/04-protocol/04-06-PLAN.md
  - .planning/phases/04-protocol/04-06-SUMMARY.md
  - .planning/phases/04-protocol/04-07-PLAN.md
  - .planning/phases/04-protocol/04-07-SUMMARY.md
  - .planning/phases/04-protocol/04-08-PLAN.md
  - .planning/phases/04-protocol/04-08-SUMMARY.md
  - .planning/phases/04-protocol/04-09-PLAN.md
  - .planning/phases/04-protocol/04-09-SUMMARY.md
  - .planning/phases/04-protocol/04-10-PLAN.md
  - .planning/phases/04-protocol/04-10-SUMMARY.md
  - .planning/phases/04-protocol/04-11-PLAN.md
  - .planning/phases/04-protocol/04-11-SUMMARY.md
  - .planning/phases/04-protocol/04-12-PLAN.md
  - .planning/phases/04-protocol/04-12-SUMMARY.md
  - .planning/phases/04-protocol/04-13-PLAN.md
  - .planning/phases/04-protocol/04-13-SUMMARY.md
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - docs/04-MULTI-NODE.md
  - docs/API.md
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - docs/adr/0001-snapshot-load-full-rollback.md
  - docs/adr/0012-equal-version-tiebreak.md
  - internal/app/internal/simnet/simnet.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/persist/facade.go
  - internal/app/persist/snapshot.go
  - internal/app/repl/config_internal_test.go
  - internal/app/repl/core.go
  - internal/app/repl/decision_coverage_test.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/egress_internal_test.go
  - internal/app/repl/listedkeys_internal_test.go
  - internal/app/repl/listing_internal_test.go
  - internal/app/repl/merge_stats_test.go
  - internal/app/repl/peertable_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/repl/sync_test.go
  - internal/app/repl/syncstream_internal_test.go
  - internal/app/repl/wait_internal_test.go
  - internal/app/repl/warn_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/config/config.go
  - internal/config/validate.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/gate_test.go
  - internal/grpcapi/list.go
  - internal/grpcapi/red_gate_test.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait_integration_test.go
  - internal/grpcapi/watch.go
  - internal/port/repl.go
  - internal/stand/stand.go
  - internal/stand/stand_red_test.go
  - internal/stand/stand_test.go
  - internal/udprepl/adapter.go
  - internal/udprepl/codec.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/loop.go
  - internal/udprepl/mirror.go
  - internal/udprepl/mirror_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replay_notify_internal_test.go
  - internal/udprepl/smoke_test.go
  - internal/udprepl/socket.go
  - internal/udprepl/syncsize_internal_test.go
  - internal/udprepl/three_node_test.go
  - scripts/red-expectations.tsv
  - scripts/red-runner
covered_digest: "v4:sha256:8df968393ca9354e493f114d81d7b3f5885a965e67e47bd06e7be95ca1fb6c49"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 33/33
  gaps_closed: []
  gaps_remaining: []
  regressions: []
decision_coverage:
  honored: 9
  total: 9
  not_honored: []
deferred:
  - truth: "04-06 truth: existing snapshot semantics are unchanged — keys absent from the snapshot keep their state"
    addressed_in: "Phase 6 (APIS-07)"
    evidence: "Decision 14.1 / ADR 0001 (accepted 2026-11-19, before Phase 4) makes a snapshot load a full rollback: keys absent from the snapshot are deleted. REQUIREMENTS.md APIS-07 states that acceptance; internal/app/persist/snapshot.go replayKeys/rollbackKey implement it; TestEngine_snapshotCreateLoadRestore passes. Phase 4's own deliverable — Restore returns the replayed records and SnapshotLoad notifies them — is verified (truth 27)."
  - truth: "04-05 prohibition: an equal-version DataReplication must always be Acked"
    addressed_in: "Phase 5 (05-03, PERSIST-07)"
    evidence: "ADR 0012 (accepted 2026-11-22) resolves an equal version by the deterministic (version, content) law: identical content still Acks with no reply (the duplicate-delivery path this prohibition protects), while differing content falls through to the store and the answer follows the stored record — an incoming win Acks, a stored win answers with one unicast DataReplication and no Ack. internal/app/repl/core.go applyRemote implements it; TestReplication_equalVersionAcked and TestReplication_equalVersionContentLaw pass. Equality is never treated as stale."
---

# Phase 4: Protocol Verification Report

**Phase Goal:** The replication protocol implements the Spec's reconciliation semantics — a joining or restarting node's interface stays closed until it has the complete picture, and every peer interaction (stale records, new peers, duplicate identities, version mismatch) has defined, tested behavior.
**Verified:** 2026-11-23T20:41:49Z (HEAD `b19a46c`, branch `gsd/v1.1-milestone`)
**Status:** passed
**Re-verification:** Yes — regeneration of the stale 2026-11-22 report after Phases 5–8 changed the covered tree. The prior report had no `gaps:` section, so this pass re-established the must-have set from the plans and ROADMAP (34 truths: the prior 33 plus the 04-06 “snapshot semantics unchanged” truth the earlier report omitted), then verified every truth against the current code. One truth is deferred to a later phase (decision-superseded, below); nothing is a blocking gap.

**Verdict basis:** every ROADMAP success criterion and every PLAN must-have was checked against the code and re-run on this tree. SUMMARY claims were treated as hypotheses; the tables below name the assertion or command that proves each one locally. The tree is materially different from the one the stale report verified: `internal/udprepl/repl.go` was split into `adapter.go` / `codec.go` / `loop.go` / `mirror.go` / `socket.go` (Phase 7), `config` moved into `internal/config`, `Engine.Restore` moved into `internal/app/persist/snapshot.go`, and Phases 5/7/8 changed the equal-version tie-break, the discard-class catalog, snapshot rollback, the pending index and the replay seam. Each of those changes was checked against the phase-4 contract rather than assumed harmless.

---

## Goal Achievement

### Observable Truths

Column *Evidence* names the current-tree code and the test/command that asserts the behavior. Every named test was run by the verifier on this tree (raw commands in Behavioral Spot-Checks and the Phase Gate).

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | A `DataSync` datagram carries `sync_id`, a 1-based `chunk_index` and `chunk_count` through the real codec (core emit → protobuf → adapter decode) | ✓ VERIFIED | `internal/udprepl/replproto/supkv/v1/supkv.proto:70-75` (fields 2/3/4); `internal/udprepl/codec.go:122-132,168-183`; `TestWire_dataSyncHeaderRoundTrip` PASS (real encode/decode, not a struct copy); `TestSync_chunkWireBudget` PASS (re-encodes through the real codec) |
| 2 | A partial listing never completes: a 3-part listing leaves both the core and the adapter mirror false after parts 1 and 2, and the closing chunk flips them | ✓ VERIFIED | `internal/app/repl/core.go:1904-1964` (`complete()` requires every index 1..count); `TestSync_partialListingKeepsGateClosed` PASS; `TestMirror_listingComplete` PASS; `TestScenario_joinerWaitsForCompleteListing` PASS |
| 3 | While closed, `Get`/`List`/`Watch`/`Set`/`Delete`/`SnapshotCreate`/`SnapshotLoad` answer `Unavailable` and health is `NOT_SERVING` | ✓ VERIFIED | `s.gated()` is each handler's first statement — `internal/grpcapi/server.go:141,197,264,316,332,360`, `internal/grpcapi/watch.go:533`; `TestGate_readsAndWritesUnavailable` PASS (all seven RPCs + health), `TestGate_emptyKeyStillUnavailable` PASS; frozen `TestRed_R03_gateClosedUntilFullList` PASS under `SUPKV_RED_TESTS=1` |
| 4 | After the listing opens, `create_if_absent` for any listed key (live or tombstoned) returns `AlreadyExists` before the engine is called | ✓ VERIFIED | `internal/grpcapi/server.go:206-208` checks `ListedKey` before `SetIfAbsent`; `TestServer_createIfAbsentListed` PASS; the frozen F14 test's post-listing assertion PASS |
| 5 | A read of a listed-but-unarrived key returns `Unavailable`; a locally tombstoned listed key returns `NotFound` (14.5(6)); a create issued right after health flips to `SERVING` is already blocked | ✓ VERIFIED | `internal/grpcapi/server.go:140-164` (`Engine.Version` is the presence discriminator); `TestServer_getListedNotArrived`, `TestServer_getListedTombstoneAndAbsent`, `TestServer_getListedLive`, `TestServer_gateOpenListedRace` — all PASS |
| 6 | Each non-empty listing part absorbs its maximum entry version into the HLC exactly once; empty or discarded parts absorb nothing; the next local write beats a max-version part | ✓ VERIFIED | `internal/app/repl/core.go:1825-1834` calls `store.Observe(maxEntryVersion(...))` after validation and identity binding; `internal/app/storage/store.go:377-381` updates only the clock; `TestSync_observePerPart`, `TestSync_observePerPartEmptyAndDiscarded`, `TestSync_observeOncePerPart`, `TestMergeStats_listingAndStaleExcluded` — all PASS |
| 7 | The adapter publishes `{complete, listedKeys}` in one atomic pointer, content-gated by the core's listing generation, so `SyncArrived()==true` implies `ListedKey` sees the keys and an identical-listing flood republishes nothing | ✓ VERIFIED | `internal/udprepl/mirror.go:159-192` (one `atomic.Pointer` value), `:255-266` (generation-gated publish); `TestMirror_syncArrivedFlip`, `TestMirror_identicalListingFloodDoesNotRepublish` (100 repeats, same pointer), `TestSync_listingGenerationContentOnly` — all PASS |
| 8 | Retained listed-key state is bounded in size (`maxListedKeysTotal`, whole-peer LRU eviction, counted) and in time (2 × `sync_interval` expiry) | ✓ VERIFIED | `internal/app/repl/core.go:2081-2189` (`retainListedKeys`/`dropListedPeer`/`expireListedKeys`); `TestSync_listedKeysBounded` (200 hostile identities × 1000 keys) PASS; `TestSync_listedKeysExpire` PASS |
| 9 | A listing is accepted only from a live peer-table `(node_id, run_id)` row; an unbound or forged-run listing is counted, WARN rate-limited, and retains nothing/opens nothing; an empty listing creates no row | ✓ VERIFIED | `internal/app/repl/core.go:1818-1877` (`listingBound`); `TestSync_unboundListingDiscarded`, `TestSync_forgedRunIDCannotReplaceListing`, `TestSync_emptyListingCreatesNoRow` — all PASS. Phase 7 split the count into the named `unbound_listing` class (07-08); the binding, drop position and limiter are unchanged |
| 10 | `proto_version` is checked before CRC and body decode; an incompatible version yields `bad_proto_version` with envelope fields preserved and a per-`(node_id, class)` rate-limited WARN | ✓ VERIFIED | `internal/udprepl/codec.go:80-88` (version at :80, CRC at :84, body at :88); `internal/app/repl/core.go:1107-1126` (counted, attributed, warned); `TestDecode_negativeTable` incl. `future_proto_version_wins_over_corrupted_body` PASS; `TestDiscard_warnLimiterKeyedByNodeAndClass`, `TestDiscard_warnRateLimitedPerPeer` PASS |
| 11 | A `DataSync` header violation is discarded as `bad_sync_header`, counted, with no clock call and no accumulation | ✓ VERIFIED | `internal/app/repl/core.go:1888-1897` (`validSyncHeader`), `:1225-1246` (`validateSync` header first); `TestDiscard_badSyncHeader`, `TestDiscard_beforeClock` — PASS |
| 12 | An empty-disk node waits `startup_gate_empty` (default 60 s) while a node holding ≥1 record uses `startup_gate`; the choice is made once from an O(1) key count | ✓ VERIFIED | `cmd/sup-kv/main.go:347-350` (`gateBackstopFor` → `engine.KeyCount()`), `:639` wiring; `internal/app/persist/facade.go:99` → `storage.Store.Len` (`store.go:387-391`); `internal/config/config.go:89-93` (60 s default); `TestMain_startupGateEmptySelection` both subtests PASS |
| 13 | Two empty nodes exchange complete empty listings and open without waiting for the new parameter | ✓ VERIFIED | `TestStand_startupGateEmptyTwoEmptyNodes` PASS — both served 27.99 ms after launch against an explicit 30 s empty backstop |
| 14 | `startup_gate_empty` has a flag, a `SUPKV_` env twin, a config field and positivity validation that exits through the usage path (code 2) | ✓ VERIFIED | `internal/config/config.go:51,230-232` (flag + env + default), `internal/config/validate.go:126`; `cmd/sup-kv/main.go:181-185` (config error → `exitUsage` 2); `TestMain_startupGateEmptyRange` both subtests PASS |
| 15 | The stand harness derives `--startup-gate-empty` from an explicit `--startup-gate` without overriding an explicit empty gate, and the frozen R-03 test is untouched | ✓ VERIFIED | `internal/stand/stand.go:610-631` (`emptyGateArg`), `:633-660` (both dash/value forms); `TestStand_startupGateEmptyDerivedFromExtras`, `TestStand_startupGateEmptyLoneNode` PASS; `git log d9042cb^..HEAD -- internal/stand/stand_red_test.go internal/grpcapi/red_gate_test.go` is empty |
| 16 | The first `Hello` from an unknown `node_id` emits exactly one targeted listing (plus the identity echo the binding requires) and still runs the discovery flush; a repeat Hello emits nothing | ✓ VERIFIED | `internal/app/repl/core.go:1453-1544` (first-seen branch does not return early; repeat refreshes only); `TestRestart_firstHelloExactlyOneListing`, `TestRestart_repeatHelloNoListing`, `TestRestart_firstHelloStillFlushesPending` — all PASS |
| 17 | A known node's unseen `run_id` is a restart (one listing); a previously-current-but-not-now `run_id` is the named alternating-identity configuration error (counted, WARN rate-limited, no listing, row/liveness untouched) | ✓ VERIFIED | `internal/app/repl/core.go:1460-1511` (alternation branch before the restart branch); `TestRestart_runIDChangeTriggersFullSync`, `TestPeer_alternatingRunIDsIsConfigError`, `TestPeer_seenRingBounded`, `TestPeer_repeatSameRunStillRefresh` — all PASS |
| 18 | A peer returning after `peer_timeout` is an unknown node and gets the first-seen listing with no `sync_interval` term | ✓ VERIFIED | `TestRestart_returningAfterEvictionGetsListing` PASS; `TestScenario_emptyDisk` PASS (recovery within `oneWay + (2N+4)·pace`, no interval term) |
| 19 | Listings are keyed by `(node_id, run_id)` (never the source handle), bounded by destination/chunk caps with counted drops, and the broadcast round is reserved and drains first | ✓ VERIFIED | `internal/app/repl/core.go:563-611` (`listingDst`), `:2556-2611` (`enqueueListing`/`listingFits`/`oweListing`), `:2726-2757` (broadcast drain priority); `TestListing_hostileBounded`, `TestListing_broadcastNeverStarved`, `TestListing_chunkCapCoversEveryDestination`, `TestListing_refusedFirstSeenIsRearmed` — all PASS |
| 20 | All destinations within one `sync_interval` share one listing build while the store generation is unchanged; any mutation rebuilds exactly once; rotation preserves original chunk indices | ✓ VERIFIED | `internal/app/repl/core.go:2666-2679` (`listingBuild` age + `store.Generation()`), `:2584-2593` (rotation); `TestListing_sharedBuild`, `TestListing_generationRebuild`, `TestRestart_generationInvalidatesCachedListing`, `TestListing_rotationPreservesIndices` — all PASS |
| 21 | A strictly older `DataReplication` produces exactly one unicast reply carrying the held newer record and NO Ack; the store is not merged and the ledger is untouched | ✓ VERIFIED | `internal/app/repl/core.go:1638-1643`; `TestReplication_staleAnsweredNotAcked` PASS (one queued identity, one `DataReplication` at drain, no Ack, zero merge stats, pending unchanged) |
| 22 | An equal-version incoming `DataReplication` duplicate is Acked with no reply; newer/absent still applies and Acks | ✓ VERIFIED | `internal/app/repl/core.go:1644-1651` (identical content → one Ack, no reply); `TestReplication_equalVersionAcked`, `TestReplication_equalVersionContentLaw/identical_content_Acks_and_emits_no_reply`, `TestReplication_newerApplied`, `TestReplication_absentApplies` — all PASS. The differing-content equal-version case is Phase 5's ADR 0012 tie-break (deferred item 2), not a regression: equality is still never treated as stale |
| 23 | The sender's `wait_for_replication` for a version answered instead of Acked reports `acked: false` through the real adapter tracker | ✓ VERIFIED | `TestWaitIntegration_staleRecordNotAcked` PASS (real adapter pair + real gRPC server; `Acked=false`, total peers 1, B's record untouched) |
| 24 | A local-write notification whose version is not greater than the tracked one is ignored (no broadcast, no ledger change, counted); a strictly older one additionally resolves its waiter `Superseded`; a key's first notification proceeds | ✓ VERIFIED | `internal/app/repl/core.go:2394-2448` (`trackLocalWrite`), `:3479-3506` (`trackedVersion`/`removeSupersededLocal`, now the Phase-8 per-key index); `TestNotify_firstNotificationProceeds`, `TestNotify_equalTrackedIgnored`, `TestNotify_regressedIgnored`, `TestNotify_newerBroadcasts`, `TestNotify_absentKeyBroadcasts` — all PASS |
| 25 | The reactive egress queue carries identity only and builds the message at drain, re-reading the current record (no bytes retained, no lookup before dedupe) | ✓ VERIFIED | `internal/app/repl/core.go:563-599` (`egressKey`/`queuedUnicast{id}`), `:2233-2309` (`enqueueResponse`/`resolveQueuedResponse`); `TestEgress_identityOnly`, `TestEgress_noValueRetention` — PASS |
| 26 | A write/delete the engine installed but whose confirming dump failed is still notified (track=false) before the RPC answers `Unavailable`; `ErrClosed` is never notified | ✓ VERIFIED | `internal/grpcapi/server.go:225-236,277-288` (notify on `ErrDumpFailed`/`ErrUnconfirmed`, never `ErrClosed`), `:394-408` (track=false path); `TestServer_dumpFailedWriteNotifies`, `TestServer_dumpFailedDeleteNotifies`, `TestServer_dumpFailedCreateIfAbsentNotifies` — all PASS |
| 27 | `Engine.Restore` returns the replayed records in sorted-key order (even when the final flush fails) and a no-op returns an empty list; `SnapshotLoad` notifies every replayed record with track=false and nothing on a no-op/`ErrClosed` | ✓ VERIFIED | `internal/app/persist/snapshot.go:110-191` (`Restore`/`replaySnapshot`; `slices.Sorted`, partial list + `markDirty` on error); `internal/grpcapi/server.go:359-378`; `TestEngine_restoreReturnsReplayed`, `TestEngine_restoreNoOpReturnsEmpty`, `TestEngine_restoreFailedFlushReturnsReplayed`, `TestEngine_restoreSkipsReplay`, `TestServer_snapshotLoadNotifies`, `TestServer_snapshotLoadUsesReplaySeam`, `TestServer_snapshotLoadReplayFallsBackToLive`, `TestServer_snapshotLoadNoOpDoesNotNotify` — all PASS |
| 28 | Three in-process adapters on one shared port converge on writes originated by each of them over the real socket path, stating the unicast limitation | ✓ VERIFIED | `TestSmoke_threeNodeBroadcastConvergence` PASS in 0.06 s (not skipped: real `udprepl.New(..., WithAddressReuse())`, real codec/read loop/pacing; doc comment names the unicast caveat and the loud capability-skip path was not taken) |
| 29 | Simulator scenarios prove N=3 convergence, the joiner's complete-listing signal, and F14 value preservation on a replaced node; the 14.9 anchor points at the N=3 scenario | ✓ VERIFIED | `internal/app/repl/sim_scenario_test.go:758,799,912`; `TestScenario_threeNode`, `TestScenario_joinerWaitsForCompleteListing`, `TestScenario_f14ReplacedNode` — all PASS; `decision_coverage_test.go:44` maps `14.9 → TestScenario_threeNode`; determinism scenarios (`TestScenario_burst`, `TestScenario_partitionDays`, `TestScenario_restart`) PASS |
| 30 | N-node behavior/timing, `wait_for_replication` at N > 2, the chosen bounds, the test topology, residual windows and handoffs are documented; OPERATIONS lists `bad_sync_header`/`startup-gate-empty`; the phase's SPEC-DEVIATIONS rows are `implemented` | ✓ VERIFIED | `docs/04-MULTI-NODE.md` all seven H2 sections (Traffic shape, Timing, wait_for_replication at N > 2, Protocol bounds, Three-node topology in tests, Residual windows, Handoffs); `docs/API.md:52-88` now states the N > 2 contract (the stale report's deferred item is closed by Phase 6/7); `docs/OPERATIONS.md:228,313` (knob + complete discard catalog incl. `bad_sync_header`); `docs/SPEC-DEVIATIONS.md` S-PROTO-18, S-HLC-04, S-HLC-10, S-ALG-RC-05, S-ALG-ST-06 all `implemented` with test refs |
| 31 | The red-gate contract holds: both R-03 rows are `green`, the frozen tests are byte-unchanged and pass under `SUPKV_RED_TESTS=1`, and the runner's mutation stage still kills both mutants | ✓ VERIFIED | `scripts/red-expectations.tsv` (2 R-03 green; P-01 rows are green too now — Phase 5's own one-line flip, 7851cfc, which is the D-01 mechanism working, not a phase-4 edit); frozen diffs empty since before the phase; direct frozen runs PASS; verifier-run `bash scripts/red-runner`: 7/7 rows OK, 4/4 controls pass, both mutants killed, exit 0, evidence `20261123T203359Z` |
| 32 | The Phase-3 review criticals CR-01/CR-02 are closed with fixing commits and regression tests | ✓ VERIFIED | `03-REVIEW-DISPOSITION.md:15-16` rows cite `1e06c58`/`e42a20a`; the current code enforces identity-keyed bounded listings (`core.go:563-611,2556-2611`) and identity-only egress (`core.go:597-599`); `TestListing_hostileBounded`, `TestListing_sharedBuild`, `TestEgress_identityOnly`, `TestEgress_noValueRetention` PASS |
| 33 | The module is clean on the verified tree: build, vet, strict lint, race test suite, buf lint, buf breaking, darwin cross build/vet, tidy | ✓ VERIFIED | `go build ./...` 0; `go vet ./...` 0; `make lint` 0; `go test -race -cover ./...` 0; `make buf-lint` 0; `make buf-breaking` 0; `GOOS=darwin GOARCH=arm64 go build/vet ./...` 0; `go mod tidy -diff` empty |
| 34 | Existing snapshot semantics are unchanged: keys absent from the snapshot keep their state, fresh versions dominate, a whole-restore no-op stays clean (04-06 truth) | ↷ DEFERRED | Superseded by decision 14.1 / ADR 0001 (accepted 2026-11-19) and implemented in Phase 6 (APIS-07): `snapshot.go:198-246` deletes live keys absent from the snapshot and `TestEngine_snapshotCreateLoadRestore` pins the rollback. Phase 4's actual deliverable — the replayed records are returned and notified — is verified under truth 27. See Deferred Items |

**Score:** 33/34 truths verified (0 present-but-behavior-unverified; 1 deferred to a later phase)
**Overrides applied:** 0 (the two supersessions are recorded as deferred, not as overrides)

### Deferred Items

Items not met literally on the current tree but explicitly resolved by a later milestone phase. They do not affect the status determination.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | 04-06 truth: “keys absent from the snapshot keep their state” | Phase 6 (APIS-07) | Decision 14.1 / ADR 0001 makes snapshot load a full rollback; `REQUIREMENTS.md` APIS-07 states the acceptance; `internal/app/persist/snapshot.go:198-246` (`replayKeys`/`rollbackKey`) implements it; `TestEngine_snapshotCreateLoadRestore` PASS |
| 2 | 04-05 prohibition: “an equal-version DataReplication must always be Acked” | Phase 5 (05-03, PERSIST-07) | ADR 0012's deterministic (version, content) law; `internal/app/repl/core.go:1644-1651,1653-1701`; `TestReplication_equalVersionAcked` (identical duplicate still Acks) and `TestReplication_equalVersionContentLaw` PASS. The prohibition's rationale — never treat equality as stale, keep retransmission ladders draining — still holds for duplicate delivery |

### Post-Phase Evolution (verified, not gaps)

| Change | Phase | Why it does not break Phase 4 |
| ------ | ----- | ----------------------------- |
| `internal/udprepl/repl.go` split into `adapter.go` / `codec.go` / `loop.go` / `mirror.go` / `socket.go` | 7 | The plan-declared artifact's `provides` text is present at the new paths: `listingMirror`/`ListedKey` in `mirror.go`, pre-body version verdict and both header mappings in `codec.go`, the event loop and replay seam in `loop.go`, the socket in `socket.go`. The automated artifact query's single “missing” row is this path retirement, not absent behavior |
| P-01 red-expectation rows flipped to `green` | 5 (7851cfc) | The D-01 mechanism's intended one-line flip when the P-01 value-limit fix landed; the frozen test files remain byte-unchanged and the red-runner still passes 7/7 |
| `unbound_listing` split out of `duplicate_node_id` | 7 (e17ecaa) | The listing binding, drop-before-clock/accumulation/completeness, counter and per-`(node_id,class)` WARN are unchanged; only the class label is now honest (the phase-4 review's New-2) |
| `docs/API.md` shipped with the N > 2 `wait_for_replication` contract | 6/7 | Closes the stale report's only deferred item; truth 30 now verifies against the real public API document |
| Equal-version content tie-break | 5 (05-03) | Deferred item 2 above |
| Snapshot full rollback | 6 (APIS-07) | Deferred item 1 above |
| Phase-8 per-key pending index and version-only replay seam | 8 (08-02/08-04) | The phase-4 tests still pass on the new structures: `TestNotify_*`, `TestEgress_identityOnly`, `TestServer_snapshotLoad*`, `TestNotifyReplayed_pacedNotImmediate` all PASS; the replay seam still notifies every replayed record (truth 27) |

---

### Required Artifacts

Automated `gsd-tools query verify.artifacts` over the 13 plans: 24/25 declared artifacts exist and pass; the one exception is `internal/udprepl/repl.go` (plan 04-01), retired by the Phase-7 split — its behavior is verified at the replacement paths. Substance (not just existence) was spot-checked:

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/udprepl/replproto/supkv/v1/supkv.proto` / `.pb.go` | DataSync header fields, committed generated code | ✓ VERIFIED | `sync_id=2`, `chunk_index=3`, `chunk_count=4`; `supkv.pb.go` present and builds |
| `internal/port/repl.go` | Header fields + `ProtoVersionMismatch` verdict | ✓ VERIFIED | `DataSync` at :89-101; verdict documented at :26-33 with the pre-CRC/pre-body order |
| `internal/app/repl/core.go` | Accumulation, retention binding/expiry, listing build/queues, stale branch, notify guard, `SupportedProtoVersion=2` | ✓ VERIFIED | 3588 lines; every seam cited in the truths above is implemented; no stub returns in change paths |
| `internal/udprepl/repl.go` (declared path) | Mirror, pre-body version verdict, header mapping, `ListedKey` | ✓ VERIFIED (relocated) | Path retired by Phase 7; provides-text lives in `mirror.go` (mirror/ListedKey), `codec.go` (version verdict, both mappings), `loop.go` (event loop), `adapter.go` (seams). Automated query reports this one path missing — a citation drift, not absent behavior |
| `internal/grpcapi/gate.go`, `server.go`, `watch.go`, `list.go` | Gate-first reads/writes, listed-key semantics, documented `ListedKey` nil contract | ✓ VERIFIED | `gated()` first in all seven handlers; `ServerOptions.ListedKey` nil contract documented at `gate.go:69-76`; `List` delegates to `list.go` behind the gate |
| `internal/app/storage/store.go` | `Observe`, `Len`, `Generation` | ✓ VERIFIED | `Observe` :377-381 (clock only), `Len` :387-391 (O(1)), `Generation` :401+ (one bump inside `install` :456-467) |
| `internal/app/persist/engine.go`, `snapshot.go`, `facade.go` | `Observe`, `KeyCount`, `Generation`, `Restore []RestoredRecord` | ✓ VERIFIED | `Restore`/`replaySnapshot` in `snapshot.go:110-191`; `KeyCount`/`Generation` delegations in `facade.go:99-108`; `engine.go` remains the facade (304 lines) |
| `cmd/sup-kv/main.go`, `internal/config/*` | `startup_gate_empty` flag/env/config/selection/wiring | ✓ VERIFIED | flag :230, env :51, default 60 s :89-93, validation `validate.go:126`, selection `main.go:347-350`, wiring :634-639 (`ListedKey`, `SyncArrived`, `ReplayNotify`, gate) |
| `internal/stand/stand.go` | Empty-gate derivation | ✓ VERIFIED | both dash/value spellings; explicit empty gate wins (`:622-631`) |
| `docs/04-MULTI-NODE.md` | N-node artifact | ✓ VERIFIED | 242 lines, all seven sections, each claim tied to a test/code symbol |
| `scripts/red-expectations.tsv`, `docs/OPERATIONS.md`, `docs/SPEC-DEVIATIONS.md` | Red flip, ops handoff, registry rows | ✓ VERIFIED | 7 rows (all green); OPERATIONS full catalog; five phase-4 rows `implemented` |
| Phase test files (repl/udprepl/grpcapi/stand/persist/cmd) | The named proofs | ✓ VERIFIED | All exist and PASS (see Behavioral Spot-Checks); no disabled requirement test |

### Key Link Verification

The automated `verify.key-links` verb reports 0/N for most plans because the plan `via:` strings are prose and the core↔adapter links are method-level calls across a depguard-enforced boundary (the `to` path cannot appear literally in the `from` file). Each link was therefore traced manually on the current tree; all are wired.

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `internal/app/repl/core.go` | `internal/udprepl/mirror.go` | core listing accessors → adapter mirror | ✓ WIRED | `mirrorStats` reads `ListingGeneration/ListingComplete/ListedKeysSnapshot` once per event and stores one pointer (`mirror.go:255-266`) |
| `internal/udprepl/codec.go` | `internal/app/repl/core.go` | `ProtoVersionMismatch` envelope fields → attributed discard | ✓ WIRED | `decode` sets `ProtoVersion/NodeID/RunID` before returning (`codec.go:75-83`); `HandleDatagram` counts and rate-limits (`core.go:1107-1126`) |
| `internal/app/repl/core.go` | `internal/udprepl/syncsize_internal_test.go` | header size honesty | ✓ WIRED | `chunkOverheadBytes=96` (`core.go:275`); `TestSync_chunkWireBudget` re-encodes the emitted chunk's own header and PASSes |
| `cmd/sup-kv/main.go` | `internal/udprepl/mirror.go` / `internal/grpcapi/gate.go` | `ListedKey`/`SyncArrived` injection | ✓ WIRED | `main.go:634-635`; both read the same atomic pointer |
| `internal/grpcapi/server.go` | `internal/app/persist/engine.go` | `Engine.Version` distinguishes tombstone vs unarrived | ✓ WIRED | `server.go:157`; `Engine.Version` delegates to the store view |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | per-part `Observe` | ✓ WIRED | `core.go:1829`; `Store.Observe` → `clock.Update` |
| `internal/app/persist/facade.go` | `internal/app/storage/store.go` | `KeyCount`/`Generation` delegation | ✓ WIRED | `facade.go:99-108` |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `nodeStore.Observe`/`Generation` | ✓ WIRED | `simnet.go:488,491` |
| `internal/grpcapi/wait_integration_test.go` | `internal/app/repl/core.go` | real adapter tracker observes no-Ack | ✓ WIRED | `TestWaitIntegration_staleRecordNotAcked` PASS |
| `internal/app/repl/core.go` | `internal/app/repl/restart_test.go` / `sim_scenario_test.go` | first-seen/restart contracts | ✓ WIRED | tests named in truths 16-18 PASS |
| `internal/udprepl/three_node_test.go` | `internal/udprepl/smoke_test.go` | shared helpers | ✓ WIRED | `freeUDPPort`, `smokeSocket`, `probeBroadcastAddr` present and used |
| `internal/app/repl/decision_coverage_test.go` | `internal/app/repl/sim_scenario_test.go` | 14.9 anchor | ✓ WIRED | `{"14.9", "0009-multi-node-support.md", "TestScenario_threeNode"}` (:44) |
| `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go`, `internal/grpcapi/red_gate_test.go` | expected verdict only | ✓ WIRED | frozen diffs empty; red-runner 7/7 |
| `03-REVIEW-DISPOSITION.md` | `internal/app/repl/core.go` | fixing-commit references | ✓ WIRED | `1e06c58` / `e42a20a` exist and the named regression tests pass |
| `docs/04-MULTI-NODE.md` | `internal/udprepl/three_node_test.go`, `docs/SPEC-DEVIATIONS.md` | topology/limitation and registry citations | ✓ WIRED | both automated link checks that are syntactically resolvable report `verified: true`; the prose citations resolve by symbol |

### Data-Flow Trace (Level 4)

This is a backend protocol phase; the only “rendered” data are API answers derived from live state. Traced paths:

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `Get`/`Set` listed-key decisions | `ListedKey(key)` | `core.listedKeys` union ← `handleDataSync` ← decoded `DataSync` → adapter `listingMirror` → `ServerOptions.ListedKey` | Yes — real inbound listing entries, no literal/static fallback | ✓ FLOWING |
| Gate/health readiness | `SyncArrived()` | `core.listingComplete` ← `accumulateSync` completeness predicate → mirror → `RunGate` poll → `gateOpen`/health | Yes — flips from real chunk arrivals (partial-listing and frozen stand tests) | ✓ FLOWING |
| Stale-record answer | queued `egressKey` | `applyRemote` identity enqueue → `resolveQueuedResponse` → `store.Lookup` at drain | Yes — drain serves the record current at drain time (`TestEgress_identityOnly`) | ✓ FLOWING |
| Dump-failed/replay notify | `RestoredRecord`/record | real `engine.Set/Delete/Restore` result → `grpcapi` notify seam → adapter → `core.NotifyLocalWrite`/`NotifyReplayedWrite` | Yes — store-stamped versions asserted (`TestServer_dumpFailed*`, `TestServer_snapshotLoad*`) | ✓ FLOWING |
| Listing build cache | `listingBuild.chunks` | `store.Versions()` copy, invalidated by `store.Generation()` | Yes — rebuild-on-mutation asserted (`TestListing_generationRebuild`, `TestRestart_generationInvalidatesCachedListing`) | ✓ FLOWING |

No `STATIC`, `DISCONNECTED` or `HOLLOW_PROP` paths found.

### Behavioral Spot-Checks

Run on the current tree (2026-11-23, this session); nothing was started as a long-lived service or mutated.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Frozen R-03 gate closed-until-complete | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$'` (inside the named batch below) | PASS (0.04 s) | ✓ PASS |
| Frozen R-03 F14 replaced node | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_R03_f14ReplacedNode$'` | PASS (3.09 s) | ✓ PASS |
| Full red-gate table + mutation stage | `bash scripts/red-runner` | 7/7 rows OK; 4/4 control seeds pass; `sync-without-tombstones` and `periodic-sync-disabled` killed; exit 0; evidence `red-evidence/20261123T203359Z` | ✓ PASS |
| 52 named repl truth tests (sync/listing/restart/peer/notify/replication/egress/merge-stats/warn/discard/scenario/config/stream/budget) | `go test -race -count=1 ./internal/app/repl/ -run '^(TestSync_\|TestListing_\|TestRestart_\|TestPeer_\|TestNotify_\|TestNotifyReplayed_\|TestReplication_\|TestEgress_\|TestMergeStats_\|TestScenario_\|TestDiscard_\|TestConfig_syncChunkBytesClamped\|TestChunkSyncEntries_capacityFollowsEntries\|TestSyncStream_)$' -v` | 52/52 PASS (focused batch; 2.1 s) | ✓ PASS |
| Equal-version tie-break (Phase-5 refinement) | `go test -race -count=1 ./internal/app/repl/ -run '^TestReplication_equalVersion(ContentLaw\|SameSeedTrace)$' -v` | 3/3 subtests PASS | ✓ PASS |
| Listing-drift exclusion + multi-class WARN limiter | named batch rows `TestMergeStats_listingAndStaleExcluded`, `TestDiscard_warnLimiterKeyedByNodeAndClass` | PASS | ✓ PASS |
| Adapter mirror + wire header | `go test -race -count=1 ./internal/udprepl/ -run '^(TestWire_dataSyncHeaderRoundTrip\|TestSync_chunkWireBudget\|TestMirror_listingComplete\|TestMirror_syncArrivedFlip\|TestMirror_identicalListingFloodDoesNotRepublish)$' -v` | 5/5 PASS | ✓ PASS |
| N=3 real-socket convergence | `go test -race -count=1 ./internal/udprepl/ -run '^TestSmoke_threeNodeBroadcastConvergence$' -v` | PASS (0.06 s, not skipped) | ✓ PASS |
| Decode negative table (version before CRC/body) | `go test -race -count=1 ./internal/udprepl/ -run '^TestDecode_negativeTable$' -v` | PASS (17 subtests incl. `future_proto_version_wins_over_corrupted_body`) | ✓ PASS |
| Gate + listed-key + dump-failed + snapshot-load + wait-integration RPCs | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpcapi/ -run '^(TestGate_\|TestServer_createIfAbsentListed\|TestServer_getListed\|TestServer_gateOpenListedRace\|TestServer_dumpFailed\|TestServer_snapshotLoad\|TestWaitIntegration_)' -v` | all PASS | ✓ PASS |
| Restore/replay report | `go test -race -count=1 ./internal/app/persist/ -run '^(TestEngine_restore\|TestEngine_snapshot)' -v` | all PASS incl. `TestEngine_snapshotCreateLoadRestore` | ✓ PASS |
| Empty-gate selection/range | `go test -race -count=1 ./cmd/sup-kv/ -run '^(TestMain_startupGateEmptySelection\|TestMain_startupGateEmptyRange)$' -v` | both tests + 4 subtests PASS | ✓ PASS |
| Empty-gate stand behaviour | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^(TestStand_startupGateEmptyTwoEmptyNodes\|TestStand_startupGateEmptyDerivedFromExtras\|TestStand_startupGateEmptyLoneNode)$' -v` | 4/4 PASS (two empty nodes served in 27.99 ms vs a 30 s backstop) | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` exists and no plan declares a standalone probe script. The phase's executable probes are the red-runner's D-02 overlay mutants; they were re-run by the verifier:

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| Red expectation table | `bash scripts/red-runner` (expectation stage) | 7/7 rows checked, 0 errors; R-03 and P-01 rows present, all green on the current tree | PASS |
| `sync-without-tombstones` mutant | `bash scripts/red-runner` (mutation stage) | killed on seed 848612892191557850 (expected fail) | PASS |
| `periodic-sync-disabled` mutant | `bash scripts/red-runner` (mutation stage) | killed on seed 848612892191557850 (expected fail) | PASS |
| CR-01 overlay proofs (unbound listing, forged run) | Named tests above | 0 keys retained, 0 rows, gate closed; real run's set intact | PASS |

### Phase Gate (re-run by the verifier)

| Gate | Command | Exit | Result |
| ---- | ------- | ---- | ------ |
| Build | `go build ./...` | 0 | clean |
| Vet | `go vet ./...` | 0 | clean |
| Lint | `make lint` | 0 | config verify + `golangci-lint run ./...` clean |
| Tests (race) | `go test -race -cover ./...` | 0 | all packages ok; `internal/app/repl` 95.98 s / 93.2 % coverage; `internal/grpcapi` 98.5 %; `internal/udprepl` 89.9 %; `internal/stand` 83.6 %; `internal/app/persist` 95.0 % |
| Red gate | `bash scripts/red-runner` | 0 | 7/7 rows, mutants killed (above) |
| Buf lint | `make buf-lint` | 0 | clean |
| Buf breaking | `make buf-breaking` | 0 | fetch + comparison clean on the current `origin/master` |
| Darwin cross | `GOOS=darwin GOARCH=arm64 go build/vet ./...` | 0 | clean |
| Module tidy | `go mod tidy -diff` | 0 | no diff |

The known cross-cutting `freeUDPPort` race (see Notes) did not reproduce on this run.

### Requirements Coverage

All nine requirement IDs are declared in plan frontmatter (04-01…04-13) and are mapped to Phase 4 in REQUIREMENTS.md; no orphaned requirements.

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| PROTO-01 | 04-01, 04-02, 04-03, 04-10, 04-13 | Interface closed until the full key list (R-03, 14.5) | ✓ SATISFIED | Truths 1-9, 11; frozen `TestRed_R03_gateClosedUntilFullList` + F14 PASS; simulator F14/joiner scenarios; TSV green |
| PROTO-02 | 04-03, 04-04, 04-13 | Empty-disk node waits for a peer longer (14.6) | ✓ SATISFIED | Truths 12-15 |
| PROTO-03 | 04-05, 04-13 | Stale `DataReplication` answered with the newer record, not acked (R-04) | ✓ SATISFIED | Truths 21-23, 25; equality behaviour refined by Phase-5 ADR 0012 (deferred item 2) |
| PROTO-04 | 04-08, 04-11, 04-13 | New/returning peer gets an immediate sync (R-05) | ✓ SATISFIED | Truths 16-20 |
| PROTO-05 | 04-09, 04-13 | Local-write notifications never regress (R-10) | ✓ SATISFIED | Truth 24 |
| PROTO-06 | 04-09, 04-13 | Duplicate `node_id` with alternating `run_id`s is a named config error (R-12) | ✓ SATISFIED | Truth 17 |
| PROTO-07 | 04-01, 04-11, 04-13 | `proto_version` checked before body decode (R-13) | ✓ SATISFIED | Truth 10 (incl. future-version-over-corrupt-body row) |
| PROTO-08 | 04-06, 04-13 | Every local change replicates, incl. dump-failed writes and snapshot loads (S-02) | ✓ SATISFIED | Truths 26-27 |
| PROTO-09 | 04-07, 04-10, 04-12, 04-13 | Three-or-more-node behavior described and tested (14.9) | ✓ SATISFIED | Truths 28-30; `docs/API.md` now carries the N > 2 wording |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/app/repl/*_test.go` (sync/listedkeys/listing/restart/peertable/retransmit/egress/sim/warn/merge_stats/discard) | PROTO-01,03,04,05,06,09 | Yes | 0 (red tests live in stand/grpcapi/persistfile) | None | Value/Behavioral (versions, record bytes, counts, message shapes, convergence) | ✓ |
| `internal/udprepl/*_test.go` (decode_negative, syncsize, three_node, repl, mirror, replay_notify) | PROTO-07,01,09 | Yes; `TestSmoke_threeNode…` PASS (not skipped) | capability-probe skips only, loud with reason and not taken here | None | Value/Behavioral (wire bytes, store convergence) | ✓ |
| `internal/grpcapi/*_test.go` (gate, server, wait_integration) | PROTO-01,03,08 | Yes | 0 active-test skips; the frozen red test is env-gated by design and ran under the red-runner | None | Status + Value + Behavioral (real gRPC + real adapters) | ✓ |
| `internal/stand/*_test.go` | PROTO-01,02 | Yes | red tests env-gated by design; the two R-03 rows ran under `SUPKV_RED_TESTS=1` and PASS | None | Behavioral (process/socket lifecycles) | ✓ |
| `internal/app/storage/store_test.go`, `internal/app/persist/engine_test.go` | PROTO-01,02,08 | Yes | 0 | None (fixtures hand-built; `os.WriteFile` used for corruption fixtures only) | Value/Behavioral | ✓ |
| `cmd/sup-kv/main_test.go` | PROTO-02 | Yes | 0 | None | Behavioral (real binary, health/gRPC probes) | ✓ (port-pick race noted below) |

**Disabled tests on requirements:** 0 (the env-gated red tests are the D-01 mechanism, exercised via the red-runner). **Circular patterns detected:** 0. **Insufficient assertions:** 0.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX` marker in any phase-modified Go/proto/script file (the only `XXX` is a command placeholder inside a review-evidence Markdown file) | — | Debt-marker gate clean |
| — | — | No unreferenced `TODO`/`HACK`/`PLACEHOLDER`/empty implementation in production files | — | — |
| `cmd/sup-kv/main_test.go` | ~481, ~584 | comment word “placeholders” about loopback addresses | ℹ️ Info | Documentation comment, not a stub; the tests assert real behavior |
| `docs/04-MULTI-NODE.md` | ~124-129 | says the four replication-core counters are “test-visible today, not yet published” | ℹ️ Info | Stale after Phase 7 published them (`internal/metrics/metrics.go:206-222,392-399`; `docs/OPERATIONS.md:299-308`). Phase 7's OBSV handoff owned this paragraph (04 review IN-03); the phase-4 bounds themselves are unaffected |
| `docs/SPEC-DEVIATIONS.md` | 52, 58, 86, 99, 139 | line-number citations into `internal/app/repl/core.go` are pre-Phase-5/7 (e.g. `handleDataSync` is at :1818, not :1510-1543) | ℹ️ Info | Citation drift only; every named symbol and test resolves and the rows remain `implemented`. Phase 7's review repointed other stale citations; these were missed |
| `internal/udprepl/repl.go` | — | plan-declared path no longer exists | ℹ️ Info | Phase-7 split; `provides` text verified in `mirror.go`/`codec.go`/`loop.go`/`adapter.go` (see Required Artifacts) |
| `cmd/sup-kv/main_test.go`, `internal/udprepl/smoke_test.go` | 141, 47 | `freeUDPPort` binds `:0`, closes, returns the port — a parallel sibling can win the pick-to-bind window | ℹ️ Info (known cross-cutting defect) | Test-harness port allocation, not protocol behavior; see Notes |

### Advisory (New Scope, Unevidenced)

None. The two new-scope observations raised on this regeneration — the stale
“not yet published” sentence in `docs/04-MULTI-NODE.md` and the stale
`core.go` line numbers in `docs/SPEC-DEVIATIONS.md` — carry deterministic grep
evidence and are recorded as ℹ️ Info above; neither is a blocker and neither
reverts a must-have.

### Decision Coverage

All 9 trackable CONTEXT.md decisions (D-01…D-09) are honored by shipped artifacts: `{honored: 9, total: 9, not_honored: []}` (gate message: “All trackable CONTEXT.md decisions are honored by shipped artifacts”). The research-resolution refinements (strictly-older-only stale answer, identical-content Ack, pre-body version verdict, proto v1→v2, 96-byte overhead, harness empty-gate derivation) are present in code and tests; the differing-content equal-version case was later refined by ADR 0012 (deferred item 2).

### Review Fix Verification (04-REVIEW / 04-REVIEW-DISPOSITION)

The phase review's findings were checked in the code rather than trusted from the ledger:

| Finding | Ledger says | Independent check |
| ------- | ----------- | ----------------- |
| CR-01 (listed-key retention binding) | fixed `e5b8d0d` (+`41825f8`), re-verified | `listingBound()` requires a peer-table `(node_id, run_id)` row before any clock/accumulation/completeness; `expireListedKeys()` releases sets not republished within 2 × `sync_interval`; `touchListedPeer()` creates no row for an empty listing. `TestSync_unboundListingDiscarded`, `TestSync_forgedRunIDCannotReplaceListing`, `TestSync_emptyListingCreatesNoRow`, `TestSync_listedKeysExpire` all PASS |
| WR-01…WR-06 | fixed with commits | Each has a currently-passing regression test: `TestListing_broadcastNeverStarved`, `TestListing_refusedFirstSeenIsRearmed`, `TestRestart_retainsListedKeysUntilReplacement`, `TestNotify_regressedIgnored`, `TestNotifyReplayed_pacedNotImmediate`, `TestSync_streamMapHintIndependentOfClaim`, `TestConfig_syncChunkBytesClamped`, `TestChunkSyncEntries_capacityFollowsEntries` |
| IN-01…IN-06 | fixed (IN-03 deferred to Phase 7) | IN-02's complete discard catalog is present in OPERATIONS (now twelve classes incl. `unbound_listing`); IN-04 partial return + `markDirty` present; IN-05 both flag spellings handled; IN-06 single `listingComplete`. IN-03 is now shipped by Phase 7 (the stale sentence is the Info row above); New-1/3/5 were later-phase items and New-3 is addressed by 08-04's version-only replay seam |

### Notes

**Known cross-cutting defect — `freeUDPPort` pick-to-bind race (not a phase-4 gap).** `cmd/sup-kv/main_test.go:141` and `internal/udprepl/smoke_test.go:47` bind `:0`, close the throwaway socket and return the port; a parallel sibling can win the window before the adapter/probe binds, so `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/` can intermittently fail at `probe rx bind: address already in use` (`cmd/sup-kv/main_test.go:1719`, `internal/udprepl/repl_test.go:1215`, `scenario_test.go:107`, `internal/grpcapi/wait_integration_test.go:119`; the stand harness's own probe detects the race and re-picks, `internal/stand/stand.go:267-281`). This is a test-harness port-allocation race that fails before any protocol message is exchanged. It does not bear on reconciliation semantics, duplicate-identity handling or version mismatch: the single full race run, the 52-test repl batch, the three-node real-socket test and the red-runner all passed on this tree. It is recorded here as a note, not a score-lowering gap; it is a reliability risk for any future repeated soak of the real-socket evidence, not a semantic defect.

**Artifact-path and citation drift.** The plan declared `internal/udprepl/repl.go` (retired by Phase 7's split) and `internal/app/persist/engine.go` as the home of `Restore` (now `snapshot.go`). The automated artifact query therefore reports plan 04-01 as 5/6; all behavior is present at the replacement paths and was verified there. `docs/SPEC-DEVIATIONS.md` line numbers into `core.go` are stale for the same reason; symbol and test citations resolve.

### Human Verification Required

N/A — infrastructure/protocol phase with no user-facing elements. All acceptance criteria are covered by automated stand-level real-socket tests, simulator scenarios, core/RPC tests, and the frozen red-gate tests. No behavior-unverified truths (`behavior_unverified: 0`), so the infrastructure auto-pass is not masking an invariant gap.

### Gaps Summary

None blocking. The phase goal is achieved on the current tree:

- The interface is closed until the complete listing (frozen R-03 tests pass unmodified, on real sockets and in the simulator), with per-part HLC absorption and the accepted 14.5(6) read semantics.
- Empty-disk/returning peers, stale records, duplicate identities and version mismatch all have defined, implemented, tested behavior on the current tree.
- The review's critical (CR-01) and warnings are enforced by code plus named passing tests; the Phase-3 criticals CR-01/CR-02 remain closed.
- Replication totality (dump-failed writes, snapshot loads) and N=3 real-socket/simulator behavior are present and tested; the multi-node artifact documents the bounds and handoffs, and the public API document now states the N > 2 wait contract.
- All independent gates are green and the red-gate contract holds (frozen files byte-unchanged).

Two items are recorded as deferred because an accepted later-phase decision supersedes them: the 04-06 “absent snapshot keys keep their state” truth (decision 14.1 / Phase 6) and the 04-05 “equal version must always be Acked” prohibition (ADR 0012 / Phase 5). Neither is a phase-4 failure: the phase's own deliverables — returning and notifying replayed records, and never treating equality as stale — are verified.

---

_Verified: 2026-11-23T20:41:49Z_
_Verifier: Claude (gsd-verifier)_
