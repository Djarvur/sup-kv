---
phase: 04-protocol
verified: 2026-11-27T17:03:29Z
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
  - Makefile
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
  - internal/testutil/ports.go
  - internal/testutil/ports_test.go
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
covered_digest: "v4:sha256:ecf24ed104a5868bd29833fa56609459f539bed6ddf907e2e723480c29e9ff5b"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 33/34
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Stale-fingerprint regeneration, not a gap-closure round. The previous report's digest covered the tree at b19a46c (2026-11-23); Phase 8 performance work since then touched transport-adjacent inputs (the shared claim-while-held port allocator 074ccd0/5c7a3d7/3e7b465/7aeec35 with its adoption in stand.go/main_test.go/smoke_test.go/wait_integration_test.go, the retransmit countingStore cost fixture d9e9d44-lineage in retransmit_test.go, bench/sim_dump/config-internal test edits, the stand.go harness rework, Makefile and docs edits). This run re-verified every must-have against the current tree 0b01c1b: all machine-verifiable protocol semantics re-run green first-hand, the red-runner mutation stage reproduced standalone green, and the R-01 expectation row re-confirmed as the adjudicated throttled-host wall-clock class (03-12/03-13 decisions; scheduled red-r01-x20 CI acceptance)."
decision_coverage:
  honored: 9
  total: 9
  not_honored: []
deferred:
  - truth: "04-06 truth: existing snapshot semantics are unchanged — keys absent from the snapshot keep their state"
    addressed_in: "Phase 6 (APIS-07)"
    evidence: "Decision 14.1 / ADR 0001 (accepted 2026-11-19, before Phase 4) makes a snapshot load a full rollback: keys absent from the snapshot are deleted. REQUIREMENTS.md APIS-07 states that acceptance; internal/app/persist/snapshot.go replayKeys/rollbackKey implement it; TestEngine_snapshotCreateLoadRestore passes (re-run 2026-11-27). Phase 4's own deliverable — Restore returns the replayed records and SnapshotLoad notifies them — is verified (truth 27)."
  - truth: "04-05 prohibition: an equal-version DataReplication must always be Acked"
    addressed_in: "Phase 5 (05-03, PERSIST-07)"
    evidence: "ADR 0012 (accepted 2026-11-22) resolves an equal version by the deterministic (version, content) law: identical content still Acks with no reply (the duplicate-delivery path this prohibition protects), while differing content falls through to the store and the answer follows the stored record — an incoming win Acks, a stored win answers with one unicast DataReplication and no Ack. internal/app/repl/core.go applyRemote implements it; TestReplication_equalVersionAcked and TestReplication_equalVersionContentLaw pass (re-run 2026-11-27). Equality is never treated as stale."
---

# Phase 4: Protocol Verification Report

**Phase Goal:** The replication protocol implements the Spec's reconciliation semantics — a joining or restarting node's interface stays closed until it has the complete picture, and every peer interaction (stale records, new peers, duplicate identities, version mismatch) has defined, tested behavior.
**Verified:** 2026-11-27T17:03:29Z (HEAD `0b01c1b`, branch `gsd/v1.1-milestone`)
**Status:** passed
**Re-verification:** Yes — regeneration of the stale 2026-11-23 report whose digest covered the tree at `b19a46c`. No source commit between `b19a46c` and `0b01c1b` touched a protocol production file: `internal/app/repl/core.go`, `codec.go`, `mirror.go`, `loop.go`, `adapter.go`, `socket.go`, `server.go`, `gate.go`, `watch.go`, `list.go`, `storage`, `persist`, `config` and `cmd/sup-kv/main.go` are byte-identical (verified by `git diff --name-only b19a46c..HEAD`). The Phase-8 changes are test-harness and docs: the shared claim-while-held port allocator (`internal/testutil/ports.go`, new) with its adoption in `stand.go` / `main_test.go` / `smoke_test.go` / `wait_integration_test.go`, the deterministic cost-count fixtures in `retransmit_test.go` / `config_internal_test.go`, the stand harness rework (exit diagnosis, probe retries, `launchNode` env removal), bench/perf test files, `Makefile` perf targets and doc edits. Every changed file that phase-4 evidence runs through was re-checked and every truth's named test was re-run first-hand on this tree.

**Verdict basis:** SUMMARY claims treated as hypotheses; the tables below name the assertion or command that proves each one locally on the current tree. The must-have set is the 34-truth set established from the 13 plans and the ROADMAP success criteria by the previous regeneration (no `gaps:` section existed; this run re-verified the same contract, adding the two Phase-8 files the evidence now runs through — `internal/testutil/ports.go`, `internal/testutil/ports_test.go`, `Makefile` — to the covered-input fingerprint).

---

## Goal Achievement

### Observable Truths

Column *Evidence* names the current-tree code and the test/command that asserts the behavior. Every named test was re-run by the verifier on this tree (raw commands in Behavioral Spot-Checks and the Phase Gate).

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | A `DataSync` datagram carries `sync_id`, a 1-based `chunk_index` and `chunk_count` through the real codec (core emit → protobuf → adapter decode) | ✓ VERIFIED | `internal/udprepl/replproto/supkv/v1/supkv.proto` (fields 2/3/4); `internal/udprepl/codec.go:122-183`; `TestWire_dataSyncHeaderRoundTrip` PASS and `TestSync_chunkWireBudget` PASS (re-run 2026-11-27, real encode/decode) |
| 2 | A partial listing never completes: a 3-part listing leaves both the core and the adapter mirror false after parts 1 and 2, and the closing chunk flips them | ✓ VERIFIED | `internal/app/repl/core.go:1954` (`syncStream.complete()` requires every index 1..count); `TestSync_partialListingKeepsGateClosed`, `TestMirror_listingComplete`, `TestScenario_joinerWaitsForCompleteListing` — all PASS in the re-run batches |
| 3 | While closed, `Get`/`List`/`Watch`/`Set`/`Delete`/`SnapshotCreate`/`SnapshotLoad` answer `Unavailable` and health is `NOT_SERVING` | ✓ VERIFIED | `s.gated()` is each handler's first statement — `server.go:141,197,264,316,332,360`, `watch.go:533` (re-verified by grep on this tree); `TestGate_readsAndWritesUnavailable`, `TestGate_emptyKeyStillUnavailable`, `TestGate_opensOnTimeout`, `TestGate_opensOnSync` PASS; frozen `TestRed_R03_gateClosedUntilFullList` PASS under `SUPKV_RED_TESTS=1` |
| 4 | After the listing opens, `create_if_absent` for any listed key (live or tombstoned) returns `AlreadyExists` before the engine is called | ✓ VERIFIED | `internal/grpcapi/server.go:206-208` checks `ListedKey` before `SetIfAbsent`; `TestServer_createIfAbsentListed` PASS |
| 5 | A read of a listed-but-unarrived key returns `Unavailable`; a locally tombstoned listed key returns `NotFound` (14.5(6)); a create issued right after health flips to `SERVING` is already blocked | ✓ VERIFIED | `server.go:140-164` (`Engine.Version` is the presence discriminator); `TestServer_getListedNotArrived`, `TestServer_getListedTombstoneAndAbsent`, `TestServer_getListedLive`, `TestServer_gateOpenListedRace` — all PASS |
| 6 | Each non-empty listing part absorbs its maximum entry version into the HLC exactly once; empty or discarded parts absorb nothing; the next local write beats a max-version part | ✓ VERIFIED | `internal/app/repl/core.go:1829` calls `store.Observe(maxEntryVersion(...))` after validation and identity binding (re-verified by grep on this tree); `TestSync_observePerPart`, `TestSync_observePerPartEmptyAndDiscarded`, `TestSync_observeOncePerPart`, `TestMergeStats_listingAndStaleExcluded` — all PASS |
| 7 | The adapter publishes `{complete, listedKeys}` in one atomic pointer, content-gated by the core's listing generation, so `SyncArrived()==true` implies `ListedKey` sees the keys and an identical-listing flood republishes nothing | ✓ VERIFIED | `internal/udprepl/mirror.go:185-266` read directly on this tree: completeness and keys captured into ONE immutable value, published by ONE pointer store (`a.listing.Store(&listingMirror{...})`), gated by `ListingGeneration()` — `TestMirror_syncArrivedFlip`, `TestMirror_identicalListingFloodDoesNotRepublish`, `TestSync_listingGenerationContentOnly` PASS |
| 8 | Retained listed-key state is bounded in size (`maxListedKeysTotal`, whole-peer LRU eviction, counted) and in time (2 × `sync_interval` expiry) | ✓ VERIFIED | `core.go:218` (`maxListedKeysTotal = 131072`), `:2081` (`retainListedKeys`) with `dropListedPeer`/`expireListedKeys`; `TestSync_listedKeysBounded` (200 hostile identities × 1000 keys) PASS; `TestSync_listedKeysExpire` PASS |
| 9 | A listing is accepted only from a live peer-table `(node_id, run_id)` row; an unbound or forged-run listing is counted, WARN rate-limited, and retains nothing/opens nothing; an empty listing creates no row | ✓ VERIFIED | `core.go:1860` (`listingBound`); `TestSync_unboundListingDiscarded`, `TestSync_forgedRunIDCannotReplaceListing`, `TestSync_emptyListingCreatesNoRow` — all PASS. The named `unbound_listing` class (Phase 7, 07-08) is unchanged |
| 10 | `proto_version` is checked before CRC and body decode; an incompatible version yields `bad_proto_version` with envelope fields preserved and a per-`(node_id, class)` rate-limited WARN | ✓ VERIFIED | `internal/udprepl/codec.go:81` (verdict precedes the CRC gate, envelope fields carried over); `core.go:1107-1126` (counted, attributed, warned); `TestDecode_negativeTable` PASS (17 subtests incl. `future_proto_version_wins_over_corrupted_body`); `TestDiscard_warnLimiterKeyedByNodeAndClass`, `TestDiscard_warnRateLimitedPerPeer` PASS |
| 11 | A `DataSync` header violation is discarded as `bad_sync_header`, counted, with no clock call and no accumulation | ✓ VERIFIED | `core.go` (`validSyncHeader`, header-first `validateSync`); `TestDiscard_badSyncHeader`, `TestDiscard_beforeClock` PASS |
| 12 | An empty-disk node waits `startup_gate_empty` (default 60 s) while a node holding ≥1 record uses `startup_gate`; the choice is made once from an O(1) key count | ✓ VERIFIED | `cmd/sup-kv/main.go:347-352` (`gateBackstopFor` → `engine.KeyCount()`), `:639` wiring; `internal/app/persist/facade.go:99` → `storage.Store.Len`; `internal/config/config.go:89-92` (60 s default); `TestMain_startupGateEmptySelection` both subtests PASS (re-run 2026-11-27: 0.21 s / 1.57 s) |
| 13 | Two empty nodes exchange complete empty listings and open without waiting for the new parameter | ✓ VERIFIED | `TestStand_startupGateEmptyTwoEmptyNodes` PASS (0.22 s) on the Phase-8 harness — both served against an explicit 30 s empty backstop |
| 14 | `startup_gate_empty` has a flag, a `SUPKV_` env twin, a config field and positivity validation that exits through the usage path (code 2) | ✓ VERIFIED | `internal/config/config.go:51,89-92,230-231` (flag + env + default), `internal/config/validate.go:126`; `cmd/sup-kv/main.go:181-185` (config error → `exitUsage` 2); `TestMain_startupGateEmptyRange` both subtests PASS |
| 15 | The stand harness derives `--startup-gate-empty` from an explicit `--startup-gate` without overriding an explicit empty gate, and the frozen R-03 test is untouched | ✓ VERIFIED | `internal/stand/stand.go:724-744,792` (`emptyGateArg`, both dash/value forms) — survived the Phase-8 harness rework verbatim; `TestStand_startupGateEmptyDerivedFromExtras` (both subtests), `TestStand_startupGateEmptyLoneNode` PASS; `git log` over `stand_red_test.go` / `red_gate_test.go` / `red-expectations.tsv` / `red-runner` shows no commit after their origin (`7851cfc`, 2026-11-22) — byte-unchanged |
| 16 | The first `Hello` from an unknown `node_id` emits exactly one targeted listing (plus the identity echo the binding requires) and still runs the discovery flush; a repeat Hello emits nothing | ✓ VERIFIED | `core.go:1453-1544` (first-seen branch does not return early); `TestRestart_firstHelloExactlyOneListing`, `TestRestart_repeatHelloNoListing`, `TestRestart_firstHelloStillFlushesPending` — all PASS |
| 17 | A known node's unseen `run_id` is a restart (one listing); a previously-current-but-not-now `run_id` is the named alternating-identity configuration error (counted, WARN rate-limited, no listing, row/liveness untouched) | ✓ VERIFIED | `core.go:1460-1511`; `TestRestart_runIDChangeTriggersFullSync`, `TestPeer_alternatingRunIDsIsConfigError`, `TestPeer_seenRingBounded`, `TestPeer_repeatSameRunStillRefresh` — all PASS |
| 18 | A peer returning after `peer_timeout` is an unknown node and gets the first-seen listing with no `sync_interval` term | ✓ VERIFIED | `TestRestart_returningAfterEvictionGetsListing` PASS; `TestScenario_emptyDisk` PASS (recovery within the documented bound, no interval term) |
| 19 | Listings are keyed by `(node_id, run_id)` (never the source handle), bounded by destination/chunk caps with counted drops, and the broadcast round is reserved and drains first | ✓ VERIFIED | `core.go:563-611` (`listingDst`), `:2556-2611` (`enqueueListing`/`listingFits`/`oweListing`), `:2726-2757` (broadcast drain priority); `TestListing_hostileBounded` (subtests re-run), `TestListing_broadcastNeverStarved`, `TestListing_chunkCapCoversEveryDestination`, `TestListing_refusedFirstSeenIsRearmed` — all PASS |
| 20 | All destinations within one `sync_interval` share one listing build while the store generation is unchanged; any mutation rebuilds exactly once; rotation preserves original chunk indices | ✓ VERIFIED | `core.go:2666-2679` (build age + `store.Generation()`), `:2584-2593` (rotation); `TestListing_sharedBuild`, `TestListing_generationRebuild`, `TestRestart_generationInvalidatesCachedListing`, `TestListing_rotationPreservesIndices` — all PASS |
| 21 | A strictly older `DataReplication` produces exactly one unicast reply carrying the held newer record and NO Ack; the store is not merged and the ledger is untouched | ✓ VERIFIED | `core.go:1638-1643`; `TestReplication_staleAnsweredNotAcked` PASS (one queued identity, one `DataReplication` at drain, no Ack, zero merge stats) |
| 22 | An equal-version incoming `DataReplication` duplicate is Acked with no reply; newer/absent still applies and Acks | ✓ VERIFIED | `core.go:1644-1651`; `TestReplication_equalVersionAcked`, `TestReplication_equalVersionContentLaw`, `TestReplication_equalVersionSameSeedTrace`, `TestReplication_newerApplied`, `TestReplication_absentApplies` — all PASS (re-run 2026-11-27). The differing-content case is Phase 5's ADR 0012 law (deferred item 2); equality is never treated as stale |
| 23 | The sender's `wait_for_replication` for a version answered instead of Acked reports `acked: false` through the real adapter tracker | ✓ VERIFIED | `TestWaitIntegration_staleRecordNotAcked` PASS (re-run 2026-11-27 on the Phase-8 allocator-adopted file: real adapter pair + real gRPC server; `Acked=false`, B's record untouched) |
| 24 | A local-write notification whose version is not greater than the tracked one is ignored (no broadcast, no ledger change, counted); a strictly older one additionally resolves its waiter `Superseded`; a key's first notification proceeds | ✓ VERIFIED | `core.go:2394-2448` (`trackLocalWrite`), `:3479-3506` (tracked-version check over the Phase-8 per-key index); `TestNotify_firstNotificationProceeds`, `TestNotify_equalTrackedIgnored`, `TestNotify_regressedIgnored`, `TestNotify_newerBroadcasts`, `TestNotify_absentKeyBroadcasts` — all PASS |
| 25 | The reactive egress queue carries identity only and builds the message at drain, re-reading the current record (no bytes retained, no lookup before dedupe) | ✓ VERIFIED | `core.go:563-599` (`egressKey`/`queuedUnicast{id}`), `:2233-2309`; `TestEgress_identityOnly`, `TestEgress_noValueRetention`, `TestEgress_capDropsOldest` — PASS |
| 26 | A write/delete the engine installed but whose confirming dump failed is still notified (track=false) before the RPC answers `Unavailable`; `ErrClosed` is never notified | ✓ VERIFIED | `internal/grpcapi/server.go:225-236,277-288,394-408`; `TestServer_dumpFailedWriteNotifies`, `TestServer_dumpFailedDeleteNotifies`, `TestServer_dumpFailedCreateIfAbsentNotifies`, `TestServer_closedWriteDoesNotNotify` — all PASS |
| 27 | `Engine.Restore` returns the replayed records in sorted-key order (even when the final flush fails) and a no-op returns an empty list; `SnapshotLoad` notifies every replayed record with track=false and nothing on a no-op/`ErrClosed` | ✓ VERIFIED | `internal/app/persist/snapshot.go:110-191`; `TestEngine_restoreReturnsReplayed`, `TestEngine_restoreNoOpReturnsEmpty`, `TestEngine_restoreFailedFlushReturnsReplayed`, `TestEngine_restoreSkipsReplay`, `TestServer_snapshotLoadNotifies`, `TestServer_snapshotLoadUsesReplaySeam`, `TestServer_snapshotLoadReplayFallsBackToLive`, `TestServer_snapshotLoadNoOpDoesNotNotify` — all PASS |
| 28 | Three in-process adapters on one shared port converge on writes originated by each of them over the real socket path, stating the unicast limitation | ✓ VERIFIED | `TestSmoke_threeNodeBroadcastConvergence` PASS in 0.07 s (re-run 2026-11-27 on the allocator-adopted file; not skipped: real `udprepl.New(..., WithAddressReuse())`, real codec/read loop/pacing) |
| 29 | Simulator scenarios prove N=3 convergence, the joiner's complete-listing signal, and F14 value preservation on a replaced node; the 14.9 anchor points at the N=3 scenario | ✓ VERIFIED | `TestScenario_threeNode`, `TestScenario_joinerWaitsForCompleteListing`, `TestScenario_f14ReplacedNode` — all PASS; `decision_coverage_test.go:44` maps `14.9 → TestScenario_threeNode` (re-verified by grep); determinism scenarios (`TestScenario_burst`, `TestScenario_partitionDays`, `TestScenario_restart`) PASS |
| 30 | N-node behavior/timing, `wait_for_replication` at N > 2, the chosen bounds, the test topology, residual windows and handoffs are documented; OPERATIONS lists `bad_sync_header`/`startup-gate-empty`; the phase's SPEC-DEVIATIONS rows are `implemented` | ✓ VERIFIED | `docs/04-MULTI-NODE.md` all seven H2 sections re-verified by heading grep (Traffic shape, Timing, wait_for_replication at N > 2, Protocol bounds, Three-node topology in tests, Residual windows, Handoffs); `docs/API.md` states the N > 2 contract; `docs/OPERATIONS.md:214,299` (knob + discard catalog incl. `bad_sync_header`); `docs/SPEC-DEVIATIONS.md` S-PROTO-18, S-HLC-04, S-HLC-10, S-ALG-RC-05, S-ALG-ST-06 all `implemented` (re-verified by grep; the S-FAIL-15 rewording in the Phase-8 edit does not touch these rows) |
| 31 | The red-gate contract holds: both R-03 rows are `green`, the frozen tests are byte-unchanged and pass under `SUPKV_RED_TESTS=1`, and the runner's mutation stage still kills both mutants | ✓ VERIFIED | Both R-03 rows PASS first-hand under `SUPKV_RED_TESTS=1` (`TestRed_R03_gateClosedUntilFullList` 0.06 s; `TestRed_R03_f14ReplacedNode` 3.31 s); frozen files byte-unchanged since origin (git log empty); mutation stage reproduced standalone at this tree: control passes 4/4 corpus seeds, `sync-without-tombstones` (store.go overlay) and `periodic-sync-disabled` (core.go overlay, post-Phase-8 file) are each a non-no-op edit that compiles and is killed by seed 848612892191557850. The full expectation stage is 6/7 today: the R-01 row misses at the local throttled-host scale — the adjudicated class (see Notes), not a phase-4 row and not a phase-4 regression |
| 32 | The Phase-3 review criticals CR-01/CR-02 are closed with fixing commits and regression tests | ✓ VERIFIED | `03-REVIEW-DISPOSITION.md` rows cite `1e06c58`/`e42a20a` (both resolve in git history — re-verified); `TestListing_hostileBounded`, `TestListing_sharedBuild`, `TestEgress_identityOnly`, `TestEgress_noValueRetention` PASS |
| 33 | The module is clean on the verified tree: build, vet, strict lint, race test suite, buf lint, buf breaking, darwin cross build/vet, tidy | ✓ VERIFIED | `go build ./...` 0; `go vet ./...` 0; `make lint` 0; `go test -race -count=1 ./...` 0 (15 packages ok, no FAIL); `make buf-lint` 0; `make buf-breaking` 0; `GOOS=darwin GOARCH=arm64 go build/vet ./...` 0; `go mod tidy -diff` empty — all re-run 2026-11-27 |
| 34 | Existing snapshot semantics are unchanged: keys absent from the snapshot keep their state, fresh versions dominate, a whole-restore no-op stays clean (04-06 truth) | ↷ DEFERRED | Superseded by decision 14.1 / ADR 0001 and implemented in Phase 6 (APIS-07): `snapshot.go` deletes live keys absent from the snapshot and `TestEngine_snapshotCreateLoadRestore` (re-run PASS) pins the rollback. Phase 4's actual deliverable — Restore returns the replayed records and SnapshotLoad notifies them — is verified under truth 27. See Deferred Items |

**Score:** 33/34 truths verified (0 present-but-behavior-unverified; 1 deferred to a later phase)
**Overrides applied:** 0 (the two supersessions are recorded as deferred, not as overrides)

### Deferred Items

Items not met literally on the current tree but explicitly resolved by a later milestone phase. They do not affect the status determination.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | 04-06 truth: "keys absent from the snapshot keep their state" | Phase 6 (APIS-07) | Decision 14.1 / ADR 0001 makes snapshot load a full rollback; `REQUIREMENTS.md` APIS-07 states the acceptance; `internal/app/persist/snapshot.go` (`replayKeys`/`rollbackKey`) implements it; `TestEngine_snapshotCreateLoadRestore` PASS (re-run) |
| 2 | 04-05 prohibition: "an equal-version DataReplication must always be Acked" | Phase 5 (05-03, PERSIST-07) | ADR 0012's deterministic (version, content) law; `TestReplication_equalVersionAcked` (identical duplicate still Acks) and `TestReplication_equalVersionContentLaw` PASS (re-run). The prohibition's rationale — never treat equality as stale — still holds |

### Post-Phase Evolution (verified, not gaps)

| Change | Phase | Why it does not break Phase 4 |
| ------ | ----- | ----------------------------- |
| `internal/udprepl/repl.go` split into `adapter.go` / `codec.go` / `loop.go` / `mirror.go` / `socket.go` | 7 | The plan-declared artifact's `provides` text is present at the new paths (mirror/ListedKey in `mirror.go`, version verdict + header mappings in `codec.go`, event loop in `loop.go`, socket in `socket.go`). The automated artifact query's single "missing" row is this path retirement, not absent behavior |
| Shared claim-while-held port allocator (`internal/testutil/ports.go`) adopted in `stand.go`, `main_test.go`, `smoke_test.go`, `wait_integration_test.go` | 8 (08-06/07/09) | Test-harness only; per-package disjoint bands below the ephemeral range structurally remove the pick-to-bind race the previous report recorded as a known defect note. All stand/RPC/smoke evidence re-run green through it on this tree |
| `retransmit_test.go`/`config_internal_test.go` cost-count fixtures and bench files | 8 (08-05/08-10) | Test-only `countingStore` fixture over the real consumer seam (compile-forced completeness); no production surface added; the repl truth batch passes unchanged |
| stand.go harness rework (exit diagnosis, probe retries, `launchNode` env removal, reaper fence) | 8 (08-09) | The phase-4 seams (`emptyGateArg` at :724-744, both flag spellings, kill-and-reap) survive; all stand-level phase-4 tests PASS on the reworked harness |
| Equal-version content tie-break | 5 (05-03) | Deferred item 2 above |
| Snapshot full rollback | 6 (APIS-07) | Deferred item 1 above |
| Phase-8 per-key pending index and version-only replay seam | 8 (08-02/08-04) | `core.go` byte-identical since `b19a46c`; the phase-4 tests re-pass on the current tree (`TestNotify_*`, `TestEgress_*`, `TestServer_snapshotLoad*` all PASS) |
| `docs/SPEC-DEVIATIONS.md` S-FAIL-15 reworded; `docs/OPERATIONS.md` memory model points at `docs/PERFORMANCE.md` | 8 (08-10) | Phase-4 registry rows (S-PROTO-18, S-HLC-04, S-HLC-10, S-ALG-RC-05, S-ALG-ST-06) untouched and still `implemented`; OPERATIONS still lists the knob and the complete discard catalog |

---

### Required Artifacts

Automated `gsd-tools query verify.artifacts` over the 13 plans (re-run 2026-11-27): 28/29 declared artifacts pass; the one exception is `internal/udprepl/repl.go` (plan 04-01), retired by the Phase-7 split — its behavior is verified at the replacement paths.

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/udprepl/replproto/supkv/v1/supkv.proto` / `.pb.go` | DataSync header fields, committed generated code | ✓ VERIFIED | `sync_id=2`, `chunk_index=3`, `chunk_count=4`; generated package present and builds |
| `internal/port/repl.go` | Header fields + `ProtoVersionMismatch` verdict | ✓ VERIFIED | `DataSync` header block; verdict documented with the pre-CRC/pre-body order |
| `internal/app/repl/core.go` | Accumulation, retention binding/expiry, listing build/queues, stale branch, notify guard, `SupportedProtoVersion=2` | ✓ VERIFIED | `SupportedProtoVersion=2` at `:40`, `applyRemote:1635`, `listingBound:1860`, `complete:1954`, `trackLocalWrite:2394`; every seam cited in the truths above is implemented; byte-identical since `b19a46c` |
| `internal/udprepl/repl.go` (declared path) | Mirror, pre-body version verdict, header mapping, `ListedKey` | ✓ VERIFIED (relocated) | Path retired by Phase 7; provides-text lives in `mirror.go` / `codec.go` / `loop.go` / `adapter.go`. Automated query reports this one path missing — citation drift, not absent behavior |
| `internal/grpcapi/gate.go`, `server.go`, `watch.go`, `list.go` | Gate-first reads/writes, listed-key semantics, documented `ListedKey` nil contract | ✓ VERIFIED | `gated()` first in all seven handlers (re-verified by grep); `List` delegates to `list.go` behind the gate |
| `internal/app/storage/store.go` | `Observe`, `Len`, `Generation` | ✓ VERIFIED | `Observe` clock-only; `Len` O(1); `Generation` one bump inside `install` |
| `internal/app/persist/engine.go`, `snapshot.go`, `facade.go` | `Observe`, `KeyCount`, `Generation`, `Restore []RestoredRecord` | ✓ VERIFIED | `Restore`/`replaySnapshot` in `snapshot.go:110-191`; `KeyCount`/`Generation` delegations in `facade.go` |
| `cmd/sup-kv/main.go`, `internal/config/*` | `startup_gate_empty` flag/env/config/selection/wiring | ✓ VERIFIED | flag `config.go:230`, env `:51`, default 60 s `:89-92`, validation `validate.go:126`, selection `main.go:347-352`, wiring `:634-645` (`ListedKey`, `SyncArrived`, `ReplayNotify`, `RunGate` — re-verified by grep) |
| `internal/stand/stand.go` | Empty-gate derivation | ✓ VERIFIED | `emptyGateArg` `:724-744`, both dash/value spellings; explicit empty gate wins |
| `docs/04-MULTI-NODE.md` | N-node artifact | ✓ VERIFIED | all seven H2 sections present (re-verified by heading grep), each claim tied to a test/code symbol |
| `scripts/red-expectations.tsv`, `docs/OPERATIONS.md`, `docs/SPEC-DEVIATIONS.md` | Red flip, ops handoff, registry rows | ✓ VERIFIED | 7 rows (all green expected); OPERATIONS full catalog; five phase-4 rows `implemented` |
| Phase test files (repl/udprepl/grpcapi/stand/persist/cmd) | The named proofs | ✓ VERIFIED | All exist and PASS (see Behavioral Spot-Checks); no disabled requirement test |

### Key Link Verification

The automated `verify.key-links` verb reports 0/N for most plans because the plan `via:` strings are prose and the core↔adapter links are method-level calls across a depguard-enforced boundary (the `to` path cannot appear literally in the `from` file). Each link was therefore traced manually on the current tree; all are wired.

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `internal/app/repl/core.go` | `internal/udprepl/mirror.go` | core listing accessors → adapter mirror | ✓ WIRED | `mirrorStats` reads `ListingGeneration/ListingComplete/ListedKeysSnapshot` once per event and stores one pointer (`mirror.go:260-266`, read directly this session) |
| `internal/udprepl/codec.go` | `internal/app/repl/core.go` | `ProtoVersionMismatch` envelope fields → attributed discard | ✓ WIRED | `decode` sets `ProtoVersion/NodeID/RunID` before returning (`codec.go:81`); `HandleDatagram` counts and rate-limits |
| `internal/app/repl/core.go` | `internal/udprepl/syncsize_internal_test.go` | header size honesty | ✓ WIRED | `TestSync_chunkWireBudget` re-encodes the emitted chunk's own header and PASSes |
| `cmd/sup-kv/main.go` | `internal/udprepl/mirror.go` / `internal/grpcapi/gate.go` | `ListedKey`/`SyncArrived` injection | ✓ WIRED | `main.go:634-635,645` (`SyncArrived`, `ListedKey`, `RunGate` — re-verified by grep); both read the same atomic pointer |
| `internal/grpcapi/server.go` | `internal/app/persist/engine.go` | `Engine.Version` distinguishes tombstone vs unarrived | ✓ WIRED | `server.go:157`; `Engine.Version` delegates to the store view |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | per-part `Observe` | ✓ WIRED | `core.go:1829` (re-verified by grep); `Store.Observe` → `clock.Update` |
| `internal/app/persist/facade.go` | `internal/app/storage/store.go` | `KeyCount`/`Generation` delegation | ✓ WIRED | facade delegations present; `gateBackstopFor` consumes `KeyCount()` |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `nodeStore.Observe`/`Generation` | ✓ WIRED | simnet implements the consumer seam; simulator scenarios PASS |
| `internal/grpcapi/wait_integration_test.go` | `internal/app/repl/core.go` | real adapter tracker observes no-Ack | ✓ WIRED | `TestWaitIntegration_staleRecordNotAcked` PASS (re-run) |
| `internal/app/repl/core.go` | `internal/app/repl/restart_test.go` / `sim_scenario_test.go` | first-seen/restart contracts | ✓ WIRED | tests named in truths 16-18, 29 PASS |
| `internal/udprepl/three_node_test.go` | `internal/udprepl/smoke_test.go` | shared helpers | ✓ WIRED | shared socket helpers present; `TestSmoke_threeNodeBroadcastConvergence` PASS |
| `internal/app/repl/decision_coverage_test.go` | `internal/app/repl/sim_scenario_test.go` | 14.9 anchor | ✓ WIRED | `{"14.9", "0009-multi-node-support.md", "TestScenario_threeNode"}` (`:44`) |
| `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go`, `internal/grpcapi/red_gate_test.go` | expected verdict only | ✓ WIRED | frozen files byte-unchanged; both R-03 rows PASS; mutation stage reproduced standalone green |
| `03-REVIEW-DISPOSITION.md` | `internal/app/repl/core.go` | fixing-commit references | ✓ WIRED | `1e06c58` / `e42a20a` resolve in git history (re-verified) and the named regression tests pass |
| `internal/stand/stand.go`, `cmd/sup-kv/main_test.go`, `internal/udprepl/smoke_test.go`, `internal/grpcapi/wait_integration_test.go` | `internal/testutil/ports.go` | shared claim-while-held allocator | ✓ WIRED | adopters call `testutil.UDPPort/TCPPort` with their own disjoint bands (re-verified by grep); `TestStand_*` and the RPC/smoke batches PASS through it |

### Data-Flow Trace (Level 4)

This is a backend protocol phase; the only "rendered" data are API answers derived from live state. Traced paths (unchanged by the Phase-8 test-harness edits; production sources byte-identical since `b19a46c`):

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `Get`/`Set` listed-key decisions | `ListedKey(key)` | `core.listedKeys` union ← `handleDataSync` ← decoded `DataSync` → adapter `listingMirror` → `ServerOptions.ListedKey` | Yes — real inbound listing entries, no literal/static fallback | ✓ FLOWING |
| Gate/health readiness | `SyncArrived()` | `core.listingComplete` ← `accumulateSync` completeness predicate → mirror → `RunGate` poll → `gateOpen`/health | Yes — flips from real chunk arrivals (partial-listing and frozen stand tests) | ✓ FLOWING |
| Stale-record answer | queued `egressKey` | `applyRemote` identity enqueue → `resolveQueuedResponse` → `store.Lookup` at drain | Yes — drain serves the record current at drain time (`TestEgress_identityOnly`) | ✓ FLOWING |
| Dump-failed/replay notify | `RestoredRecord`/record | real `engine.Set/Delete/Restore` result → `grpcapi` notify seam → adapter → `core.NotifyLocalWrite`/`NotifyReplayedWrite` | Yes — store-stamped versions asserted (`TestServer_dumpFailed*`, `TestServer_snapshotLoad*`) | ✓ FLOWING |
| Listing build cache | `listingBuild.chunks` | `store.Versions()` copy, invalidated by `store.Generation()` | Yes — rebuild-on-mutation asserted (`TestListing_generationRebuild`, `TestRestart_generationInvalidatesCachedListing`) | ✓ FLOWING |

No `STATIC`, `DISCONNECTED` or `HOLLOW_PROP` paths found.

### Behavioral Spot-Checks

Re-run by the verifier on the current tree (2026-11-27); nothing was started as a long-lived service or mutated.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| 96 top-level repl truth tests (sync/listing/restart/peer/notify/replication/egress/merge-stats/warn/discard/scenario/config/budget) | `go test -race -count=1 ./internal/app/repl/ -run '^(TestSync_\|TestListing_\|TestRestart_\|TestPeer_\|TestNotify_\|TestNotifyReplayed_\|TestReplication_\|TestEgress_\|TestMergeStats_\|TestScenario_\|TestDiscard_\|TestConfig_syncChunkBytesClamped\|TestChunkSyncEntries_capacityFollowsEntries)'` | 96/96 PASS (3.07 s) | ✓ PASS |
| Equal-version tie-break (Phase-5 refinement) | `go test -race -count=1 ./internal/app/repl/ -run '^TestReplication_equalVersion'` | 3/3 PASS | ✓ PASS |
| Adapter mirror + wire header + decode table + N=3 real-socket | `go test -race -count=1 ./internal/udprepl/ -run '^(TestWire_dataSyncHeaderRoundTrip\|TestSync_chunkWireBudget\|TestMirror_listingComplete\|TestMirror_syncArrivedFlip\|TestMirror_identicalListingFloodDoesNotRepublish\|TestDecode_negativeTable\|TestSmoke_threeNodeBroadcastConvergence)$'` | 7/7 PASS (three-node 0.07 s, not skipped) | ✓ PASS |
| Gate + listed-key + dump-failed + snapshot-load + wait-integration RPCs (incl. frozen R-03 gate row) | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpcapi/ -run '^(TestGate_\|TestServer_createIfAbsentListed\|TestServer_getListed\|TestServer_gateOpenListedRace\|TestServer_dumpFailed\|TestServer_snapshotLoad\|TestWaitIntegration_\|TestRed_R03)'` | 22/22 PASS incl. `TestRed_R03_gateClosedUntilFullList` and `TestWaitIntegration_staleRecordNotAcked` | ✓ PASS |
| Restore/replay report | `go test -race -count=1 ./internal/app/persist/ -run '^(TestEngine_restore\|TestEngine_snapshot)'` | 11/11 PASS incl. `TestEngine_snapshotCreateLoadRestore` | ✓ PASS |
| Empty-gate selection/range | `go test -race -count=1 ./cmd/sup-kv/ -run '^(TestMain_startupGateEmptySelection\|TestMain_startupGateEmptyRange)$'` | both tests + 4 subtests PASS | ✓ PASS |
| Empty-gate stand behaviour + frozen F14 row | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^(TestStand_startupGateEmptyTwoEmptyNodes\|TestStand_startupGateEmptyDerivedFromExtras\|TestStand_startupGateEmptyLoneNode\|TestRed_R03_f14ReplacedNode)$'` | 4/4 PASS (two empty nodes served in 0.22 s vs a 30 s backstop; F14 3.31 s) | ✓ PASS |
| Red-runner expectation stage | `bash scripts/red-runner` | 6/7 rows OK; R-01 row fails locally at the throttled-host scale (7896/8000 keys listed at the 20.6 s bound in a direct run — steady progress, deadline miss; adjudicated, see Notes) | ⚠️ ADJUDICATED (R-01 row only; not a phase-4 truth) |
| Red-runner mutation stage (standalone reproduction) | Same procedure as `scripts/red-runner` `run_mutation_stage` (temp-dir copy + overlay build + kill check) | 4/4 controls pass; `sync-without-tombstones` and `periodic-sync-disabled` each a non-no-op edit that compiles and is killed by seed 848612892191557850 (no timeout) | ✓ PASS |
| Full race suite | `go test -race -count=1 ./...` | exit 0, 15 packages ok, no FAIL | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` exists and no plan declares a standalone probe script. The phase's executable probes are the red-runner's D-02 overlay mutants; they were re-run by the verifier (standalone reproduction of the runner's own mutation stage, since the runner's expectation stage stops at the adjudicated R-01 row on this host today):

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| Control probes (4 corpus seeds) | control arm of the mutation stage | 4/4 pass | PASS |
| `sync-without-tombstones` mutant | temp-dir sed overlay on `internal/app/storage/store.go` + compile + kill check | non-no-op, compiles, killed on seed 848612892191557850 (expected fail) | PASS |
| `periodic-sync-disabled` mutant | temp-dir sed overlay on `internal/app/repl/core.go` (post-Phase-8 file) + compile + kill check | non-no-op, compiles, killed on seed 848612892191557850 (expected fail) | PASS |
| R-03 expectation rows | `SUPKV_RED_TESTS=1 go test -race` on the two frozen R-03 tests | both PASS | PASS |
| R-01 expectation row | `SUPKV_RED_TESTS=1 go test` `TestRed_R01_pacedSync20k` | fails locally at the 20.6 s documented bound (7896/8000 keys, steady ~285 keys/s progress) | ⚠️ ADJUDICATED — throttled-host wall-clock class (see Notes); green state held by full-speed-host flip evidence `20261121T225048Z` (commit `4a83919`) and the scheduled `red-r01-x20` job |

### Phase Gate (re-run by the verifier)

| Gate | Command | Exit | Result |
| ---- | ------- | ---- | ------ |
| Build | `go build ./...` | 0 | clean |
| Vet | `go vet ./...` | 0 | clean |
| Lint | `make lint` | 0 | config verify + `golangci-lint run ./...` clean |
| Tests (race) | `go test -race -count=1 ./...` | 0 | 15 packages ok, no FAIL |
| Buf lint | `make buf-lint` | 0 | clean |
| Buf breaking | `make buf-breaking` | 0 | fetch + comparison clean on the current `origin/master` |
| Darwin cross | `GOOS=darwin GOARCH=arm64 go build/vet ./...` | 0 | clean |
| Module tidy | `go mod tidy -diff` | 0 | no diff |

### Requirements Coverage

All nine requirement IDs are declared in plan frontmatter (04-01…04-13) and are mapped to Phase 4 in REQUIREMENTS.md; no orphaned requirements (re-verified by grep: the phase table lists PROTO-01…09 → Phase 4 → Complete).

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| PROTO-01 | 04-01, 04-02, 04-03, 04-10, 04-13 | Interface closed until the full key list (R-03, 14.5) | ✓ SATISFIED | Truths 1-9, 11; frozen `TestRed_R03_gateClosedUntilFullList` + F14 PASS; simulator F14/joiner scenarios |
| PROTO-02 | 04-03, 04-04, 04-13 | Empty-disk node waits for a peer longer (14.6) | ✓ SATISFIED | Truths 12-15 |
| PROTO-03 | 04-05, 04-13 | Stale `DataReplication` answered with the newer record, not acked (R-04) | ✓ SATISFIED | Truths 21-23, 25; equality behaviour refined by Phase-5 ADR 0012 (deferred item 2) |
| PROTO-04 | 04-08, 04-11, 04-13 | New/returning peer gets an immediate sync (R-05) | ✓ SATISFIED | Truths 16-20 |
| PROTO-05 | 04-09, 04-13 | Local-write notifications never regress (R-10) | ✓ SATISFIED | Truth 24 |
| PROTO-06 | 04-09, 04-13 | Duplicate `node_id` with alternating `run_id`s is a named config error (R-12) | ✓ SATISFIED | Truth 17 |
| PROTO-07 | 04-01, 04-11, 04-13 | `proto_version` checked before body decode (R-13) | ✓ SATISFIED | Truth 10 (incl. future-version-over-corrupt-body row) |
| PROTO-08 | 04-06, 04-13 | Every local change replicates, incl. dump-failed writes and snapshot loads (S-02) | ✓ SATISFIED | Truths 26-27 |
| PROTO-09 | 04-07, 04-10, 04-12, 04-13 | Three-or-more-node behavior described and tested (14.9) | ✓ SATISFIED | Truths 28-30 |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/app/repl/*_test.go` (sync/listedkeys/listing/restart/peertable/retransmit/egress/sim/warn/merge_stats/discard) | PROTO-01,03,04,05,06,09 | Yes | 0 (red tests live in stand/grpcapi/persistfile) | None — the new `countingStore` fixture delegates to the real store and counts seam calls for cost gates; it asserts counts, not self-computed expectations | Value/Behavioral (versions, record bytes, counts, message shapes, convergence) | ✓ |
| `internal/udprepl/*_test.go` (decode_negative, syncsize, three_node, repl, mirror, replay_notify) | PROTO-07,01,09 | Yes; `TestSmoke_threeNode…` PASS (not skipped) | capability-probe skips only, loud with reason and not taken here | None | Value/Behavioral (wire bytes, store convergence) | ✓ |
| `internal/grpcapi/*_test.go` (gate, server, wait_integration) | PROTO-01,03,08 | Yes | 0 active-test skips; the frozen red test is env-gated by design and ran under `SUPKV_RED_TESTS=1` | None | Status + Value + Behavioral (real gRPC + real adapters) | ✓ |
| `internal/stand/*_test.go` | PROTO-01,02 | Yes | red tests env-gated by design; the two R-03 rows ran under `SUPKV_RED_TESTS=1` and PASS | None | Behavioral (process/socket lifecycles) | ✓ |
| `internal/app/storage/store_test.go`, `internal/app/persist/engine_test.go` | PROTO-01,02,08 | Yes | 0 | None (fixtures hand-built) | Value/Behavioral | ✓ |
| `cmd/sup-kv/main_test.go` | PROTO-02 | Yes | 0 | None | Behavioral (real binary, health/gRPC probes) | ✓ |

**Disabled tests on requirements:** 0 (the env-gated red tests are the D-01 mechanism, exercised via the red-runner). **Circular patterns detected:** 0. **Insufficient assertions:** 0.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX` marker in any phase-covered Go/proto/script file (re-scanned 2026-11-27 incl. the Phase-8-changed files) | — | Debt-marker gate clean |
| — | — | No unreferenced `TODO`/`HACK`/`PLACEHOLDER`/empty implementation in production files | — | — |
| `cmd/sup-kv/main_test.go` | 672, 814, 882 | comment word "placeholders" about loopback addresses | ℹ️ Info | Documentation comment, not a stub; the tests assert real behavior |
| `docs/04-MULTI-NODE.md` | 127 | says the four replication-core counters are "test-visible today, not yet published" | ℹ️ Info | Stale since Phase 7 published them (`internal/metrics/metrics.go:392-399` publishes `supkv_replication_listed_keys_evicted_total` and `supkv_replication_notifications_regressed_total`; the adapter routes all four, `mirror.go:251-254`). Phase 7's OBSV handoff owned this paragraph; the phase-4 bounds themselves are unaffected |
| `docs/SPEC-DEVIATIONS.md` | 52, 58, 86, 99, 139 | line-number citations into `internal/app/repl/core.go` are pre-Phase-5/7 (e.g. `handleDataSync` is at `:1818` on this tree, not `:1510-1543`) | ℹ️ Info | Citation drift only; every named symbol and test resolves and the rows remain `implemented` |
| `internal/udprepl/repl.go` | — | plan-declared path no longer exists | ℹ️ Info | Phase-7 split; `provides` text verified in `mirror.go`/`codec.go`/`loop.go`/`adapter.go` (see Required Artifacts) |

### Advisory (New Scope, Unevidenced)

None — no new-scope Step 7 finding lacked deterministic evidence. The two doc-drift observations above are carried-forward Info rows from the previous report (same finding, same evidence), not new-scope blockers.

### Decision Coverage

All 9 trackable CONTEXT.md decisions (D-01…D-09) are honored by shipped artifacts: `{honored: 9, total: 9, not_honored: []}` (gate message: "All trackable CONTEXT.md decisions are honored by shipped artifacts" — re-run 2026-11-27). The research-resolution refinements (strictly-older-only stale answer, identical-content Ack, pre-body version verdict, proto v1→v2, 96-byte overhead, harness empty-gate derivation) are present in code and tests; the differing-content equal-version case was later refined by ADR 0012 (deferred item 2).

### Review Fix Verification (04-REVIEW / 04-REVIEW-DISPOSITION)

The phase review's findings were checked in the code rather than trusted from the ledger:

| Finding | Ledger says | Independent check |
| ------- | ----------- | ----------------- |
| CR-01 (listed-key retention binding) | fixed `e5b8d0d` (+`41825f8`), re-verified | `listingBound()` requires a peer-table `(node_id, run_id)` row before any clock/accumulation/completeness; `expireListedKeys()` releases sets not republished within 2 × `sync_interval`; an empty listing creates no row. `TestSync_unboundListingDiscarded`, `TestSync_forgedRunIDCannotReplaceListing`, `TestSync_emptyListingCreatesNoRow`, `TestSync_listedKeysExpire` all PASS |
| CR-02 (identity-only egress) | fixed `e42a20a` | commit resolves in history; `TestEgress_identityOnly`, `TestEgress_noValueRetention` PASS |
| WR-01…WR-06 | fixed with commits | Each has a currently-passing regression test: `TestListing_broadcastNeverStarved`, `TestListing_refusedFirstSeenIsRearmed`, `TestRestart_retainsListedKeysUntilReplacement`, `TestNotify_regressedIgnored`, `TestNotifyReplayed_pacedNotImmediate`, `TestSync_streamMapHintIndependentOfClaim`, `TestConfig_syncChunkBytesClamped`, `TestChunkSyncEntries_capacityFollowsEntries` |
| IN-01…IN-06 | fixed (IN-03 deferred to Phase 7) | IN-02's complete discard catalog is present in OPERATIONS (twelve classes incl. `unbound_listing`); IN-04 partial return + `markDirty` present; IN-05 both flag spellings handled (`emptyGateArg` re-verified on the reworked harness); IN-06 single `listingComplete`. IN-03 is shipped by Phase 7 (the stale sentence is the Info row above) |

### Notes

**The R-01 red-runner row misses locally today — adjudicated throttled-host class, not a phase-4 gap.** `bash scripts/red-runner` stops at its expectation stage with `TestRed_R01_pacedSync20k: expected green, got fail` (evidence dirs `20261127T161311Z`, `20261127T161629Z`, `20261127T165618Z`; also reproduced directly: steady progress to 7896/8000 keys against the 20.6 s documented bound — a deadline miss while converging, not a stall). The row belongs to Phase 3's contract (XPORT-01), not Phase 4's truth 31, which concerns the two R-03 rows, the frozen files and the mutants — all of which pass. The failure class is the one adjudicated by the recorded Phase-3 decisions: the 03-12 halt record documented the identical artifact pre-Phase-8 (7453–7734/8000 at the same bound; only a 4.2 s margin over the 16 s paced term) on this powersave-throttled host, the flip was cleared by full-speed-host evidence (`20261121T225048Z`, commit `4a83919`), and 03-13 assigns the 20/20 acceptance to the scheduled `red-r01-x20` CI job (`.github/workflows/scheduled.yml:149-183`). The row passed locally on 2026-11-23 and 2026-11-24 (`20261123T210724Z`, `20261124T221356Z`) on the same production sources — the host's effective speed, not the code, moved. Phase 3's own verification (2026-11-27, tree `ef370a0`) reached the same adjudication.

**Known-defect note closed.** The previous report recorded the `freeUDPPort` pick-to-bind race as a known cross-cutting test-harness defect. Phase 8 (08-06/08-09, commits `074ccd0`/`5c7a3d7`/`3e7b465`/`7aeec35`) replaced every throwaway-`:0` picker with the shared claim-while-held allocator `internal/testutil/ports.go`: per-package disjoint bands strictly below the kernel's ephemeral range, candidate claimed while the bind probe is held. Adopters re-verified by grep (`stand.go`, `main_test.go`, `smoke_test.go`, `wait_integration_test.go`); all phase-4 stand/RPC/smoke evidence passed through it on this tree.

**Artifact-path and citation drift.** The plan declared `internal/udprepl/repl.go` (retired by Phase 7's split); the automated artifact query reports 28/29 with that one path missing — all behavior is present at the replacement paths and was verified there. `docs/SPEC-DEVIATIONS.md` line numbers into `core.go` are stale for the same reason; symbol and test citations resolve.

### Human Verification Required

N/A — infrastructure/protocol phase with no user-facing elements. All acceptance criteria are covered by automated stand-level real-socket tests, simulator scenarios, core/RPC tests, and the frozen red-gate tests. No behavior-unverified truths (`behavior_unverified: 0`), so the infrastructure auto-pass is not masking an invariant gap.

### Gaps Summary

None blocking. The phase goal is achieved on the current tree:

- The interface is closed until the complete listing (frozen R-03 tests pass unmodified, on real sockets and in the simulator), with per-part HLC absorption and the accepted 14.5(6) read semantics.
- Empty-disk/returning peers, stale records, duplicate identities and version mismatch all have defined, implemented, tested behavior on the current tree.
- The review's critical (CR-01/CR-02) and warnings are enforced by code plus named passing tests; the Phase-3 criticals remain closed.
- Replication totality (dump-failed writes, snapshot loads) and N=3 real-socket/simulator behavior are present and tested; the multi-node artifact documents the bounds and handoffs, and the public API document states the N > 2 wait contract.
- All independent gates are green on this tree; the red-gate contract holds (frozen files byte-unchanged, both mutants killed, both R-03 rows green), with the R-01 expectation row adjudicated as the documented throttled-host wall-clock class owned by the scheduled CI job.

Two items are recorded as deferred because an accepted later-phase decision supersedes them: the 04-06 "absent snapshot keys keep their state" truth (decision 14.1 / Phase 6) and the 04-05 "equal version must always be Acked" prohibition (ADR 0012 / Phase 5). Neither is a phase-4 failure: the phase's own deliverables — returning and notifying replayed records, and never treating equality as stale — are verified.

---

_Verified: 2026-11-27T17:03:29Z_
_Verifier: Claude (gsd-verifier)_
