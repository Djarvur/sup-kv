---
phase: 04-protocol
verified: 2026-11-22T08:44:54Z
status: passed
score: 33/33 must-haves verified
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
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - internal/app/internal/simnet/simnet.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/config_internal_test.go
  - internal/app/repl/core.go
  - internal/app/repl/decision_coverage_test.go
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
  - internal/app/repl/warn_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/gate_test.go
  - internal/grpcapi/red_gate_test.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait_integration_test.go
  - internal/grpcapi/watch.go
  - internal/port/repl.go
  - internal/stand/stand.go
  - internal/stand/stand_red_test.go
  - internal/stand/stand_test.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/replproto/supkv/v1/supkv.pb.go
  - internal/udprepl/replproto/supkv/v1/supkv.proto
  - internal/udprepl/syncsize_internal_test.go
  - internal/udprepl/three_node_test.go
  - scripts/red-expectations.tsv
  - scripts/red-runner
covered_digest: "v4:sha256:dadb2a6dbf071daeb998c43e468d68fe33b59577be36bb21c429d21af5f20a79"
behavior_unverified: 0
overrides_applied: 0
decision_coverage:
  honored: 9
  total: 9
  not_honored: []
deferred:
  - truth: "`wait_for_replication` semantics at N > 2 appear in the final public API docs (`docs/API.md`)"
    addressed_in: "Phase 9"
    evidence: "DOCS-05 acceptance names docs/API.md (Phase 9); Phase 6 APIS-01 owns the honest-wait semantics. This phase delivers the semantics in docs/04-MULTI-NODE.md#wait_for_replication-at-n--2 with the handoff recorded there and in 04-CONTEXT.md D-08 — the file docs/API.md does not exist yet in any phase's shipped set."
---

# Phase 4: Protocol Verification Report

**Phase Goal:** The replication protocol implements the Spec's reconciliation semantics — a joining or restarting node's interface stays closed until it has the complete picture, and every peer interaction (stale records, new peers, duplicate identities, version mismatch) has defined, tested behavior.
**Verified:** 2026-11-22T08:44:54Z
**Status:** passed
**Re-verification:** No — initial verification (no prior 04-VERIFICATION.md existed)

**Verdict basis:** every roadmap success criterion and every PLAN must-have was checked against the code and re-run on the current tree. The SUMMARY claims were treated as hypotheses; the tables below cite the assertion or command that proves each one locally.

---

## Goal Achievement

### Observable Truths

Column *Evidence* names the test/command that actually asserts the behavior. All named tests were run by the verifier on this tree (see Behavioral Spot-Checks and Phase Gate for the raw commands).

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | A `DataSync` datagram carries `sync_id`, a 1-based `chunk_index` and `chunk_count` through the real codec (core emit → protobuf → adapter decode) | ✓ VERIFIED | `supkv.proto` DataSync fields 2/3/4; `internal/udprepl/repl.go` encode/decode mapping; `TestWire_dataSyncHeaderRoundTrip` (PASS); `TestSync_chunkWireBudget` re-encodes through the real codec |
| 2 | A partial listing never completes: a 3-part listing leaves both the core and the adapter mirror false after parts 1 and 2, and the closing chunk flips them | ✓ VERIFIED | `TestSync_partialListingKeepsGateClosed` (asserts false/false then true/true, PASS); `TestMirror_listingComplete`; simulator `TestScenario_joinerWaitsForCompleteListing` (asserts the flip only when `completeParts == keys`) |
| 3 | While closed, `Get`/`List`/`Watch`/`Set`/`Delete`/`SnapshotCreate`/`SnapshotLoad` answer `Unavailable` and health is `NOT_SERVING` | ✓ VERIFIED | `Server.gated()` is each handler's first statement (`server.go:134,190,249,290,315,339`; `watch.go:370`); `TestGate_readsAndWritesUnavailable` (all seven RPCs + health), `TestGate_emptyKeyStillUnavailable`, frozen `TestRed_R03_gateClosedUntilFullList` (PASS under `SUPKV_RED_TESTS=1`) |
| 4 | After the listing opens, `create_if_absent` for any listed key (live or tombstoned) returns `AlreadyExists` before the engine is called | ✓ VERIFIED | `server.go:199-201` checks `ListedKey` before `SetIfAbsent`; `TestServer_createIfAbsentListed`; frozen F14 test post-listing assertion (`AlreadyExists` once SERVING) |
| 5 | A read of a listed-but-unarrived key returns `Unavailable`; a locally tombstoned listed key returns `NotFound` (14.5(6) exactly as accepted) | ✓ VERIFIED | `server.go:140-156` (`Engine.Version` is the presence discriminator); `TestServer_getListedNotArrived`, `TestServer_getListedTombstoneAndAbsent`, `TestServer_getListedLive`, `TestServer_gateOpenListedRace` |
| 6 | Each non-empty listing part absorbs its maximum entry version into the HLC exactly once; empty or discarded parts absorb nothing; the next local write beats a max-version part | ✓ VERIFIED | `handleDataSync` calls `store.Observe(maxEntryVersion(...))` after validation and identity binding; `TestSync_observePerPart`, `TestSync_observePerPartEmptyAndDiscarded`, `TestSync_observeOncePerPart`; `TestMergeStats_listingAndStaleExcluded`; frozen F14 rule-4 assertion (first local write > listed max version) |
| 7 | The adapter publishes `{complete, listedKeys}` in one atomic pointer, content-gated by the core's listing generation, so `SyncArrived()==true` implies `ListedKey` sees the keys and an identical-listing flood republishes nothing | ✓ VERIFIED | `internal/udprepl/repl.go` `listingMirror` + `mirrorStats` (single `listing.Store` on generation change); `TestMirror_syncArrivedFlip`, `TestMirror_identicalListingFloodDoesNotRepublish` (100 repeats, same pointer, PASS); `TestSync_listingGenerationContentOnly` |
| 8 | Retained listed-key state is bounded in size (`maxListedKeysTotal`, whole-peer LRU eviction, counted) and in time (2 × `sync_interval` expiry) | ✓ VERIFIED | `retainListedKeys`/`dropListedPeer`/`expireListedKeys`; `TestSync_listedKeysBounded` (200 hostile identities × 1000 keys, PASS), `TestSync_listedKeysExpire` (union empty past 2 intervals, peer row and completeness retained, `ListedKeysEvicted==1`) |
| 9 | A listing is accepted only from a live peer-table `(node_id, run_id)` row; an unbound or forged-run listing is counted as the configuration-error class, WARN rate-limited, and retains nothing/opens nothing; an empty listing creates no row | ✓ VERIFIED | `listingBound` (`core.go:1663-1675`); `TestSync_unboundListingDiscarded` (0 keys, 0 rows, gate closed, still closed after 24 h virtual), `TestSync_forgedRunIDCannotReplaceListing`, `TestSync_emptyListingCreatesNoRow` — all PASS independently. This is the Phase-4 review's new critical (CR-01), re-verified here from source and the named tests |
| 10 | `proto_version` is checked before CRC and body decode; an incompatible version yields the `bad_proto_version` class with envelope fields preserved and a WARN rate-limited per `(node_id, class)` per interval | ✓ VERIFIED | `internal/udprepl/repl.go` `decode` (version at :1208, CRC at :1212, body at :1217); `TestDecode_negativeTable` rows `wrong_proto_version_*`, `proto_version_{zero,one,max_uint32}`, `future_proto_version_wins_over_corrupted_body`; `TestDiscard_warnLimiterKeyedByNodeAndClass`, `TestDiscard_warnRateLimitedPerPeer` |
| 11 | A `DataSync` header violation is discarded as `bad_sync_header`, counted, with no clock call and no accumulation | ✓ VERIFIED | `validSyncHeader` + `validateSync` (header first, then entries); `TestDiscard_badSyncHeader`; `TestDiscard_beforeClock` |
| 12 | An empty-disk node waits `startup_gate_empty` (default 60 s) while a node holding ≥1 record uses `startup_gate`; the choice is made once from an O(1) key count | ✓ VERIFIED | `cmd/sup-kv/main.go:144` default, `gateBackstopFor` (`KeyCount()==0`), wired to `ServerOptions.StartupGate`; `TestMain_startupGateEmptySelection` both subtests (empty stays `NOT_SERVING` past the normal gate; seeded opens on the normal gate) |
| 13 | Two empty nodes exchange complete empty listings and open without waiting for the new parameter | ✓ VERIFIED | `TestStand_startupGateEmptyTwoEmptyNodes` (both SERVING within 10 s against a 30 s explicit empty backstop, PASS) |
| 14 | `startup_gate_empty` has a flag, a `SUPKV_` env twin, a config field and positivity validation that exits through the usage path (code 2) | ✓ VERIFIED | `main.go:103,353,492-498`; `TestMain_startupGateEmptyRange` (negative empty gate and zero normal gate both exit 2 naming the flag) |
| 15 | The stand harness derives `--startup-gate-empty` from an explicit `--startup-gate` without overriding an explicit empty gate, and the frozen R-03 test is untouched | ✓ VERIFIED | `emptyGateArg`/`flagValue` (`stand.go:622-689`); `TestStand_startupGateEmptyDerivedFromExtras`, `TestStand_startupGateEmptyLoneNode`; `git diff` of `internal/stand/stand_red_test.go` across the whole phase is empty |
| 16 | The first `Hello` from an unknown `node_id` emits exactly one targeted listing (plus the identity echo the CR-01 binding requires) and still runs the discovery flush; a repeat Hello emits nothing | ✓ VERIFIED | `handleHello`/`greetAndList`; `TestRestart_firstHelloExactlyOneListing` (one targeted chunk + one echo, nothing else), `TestRestart_repeatHelloNoListing`, `TestRestart_firstHelloStillFlushesPending` |
| 17 | A known node's unseen `run_id` is a restart (one listing); a previously-current-but-not-now `run_id` is the named alternating-identity configuration error (counted, WARN rate-limited, no listing, row/liveness untouched) | ✓ VERIFIED | `handleHello` alternation branch + `peer.seenRuns` ring; `TestRestart_runIDChangeTriggersFullSync`, `TestPeer_alternatingRunIDsIsConfigError` (0 messages, count 1, one WARN, row keeps current run), `TestPeer_seenRingBounded`, `TestPeer_repeatSameRunStillRefresh` |
| 18 | A peer returning after `peer_timeout` is an unknown node and gets the first-seen listing with no `sync_interval` term | ✓ VERIFIED | `TestRestart_returningAfterEvictionGetsListing` (one targeted chunk immediately after the eviction); simulator `TestScenario_emptyDisk` asserts recovery within `oneWay + (2N+4)·pace` with no interval term |
| 19 | Listings are keyed by `(node_id, run_id)` (never the source handle), bounded by destination and chunk caps with counted drops, and the broadcast round is reserved and drains first | ✓ VERIFIED | `listingDst`, `maxListingDestinations/maxListingChunks`, `oweListing`; `TestListing_hostileBounded` (200 identities + spoofed handles stay capped, drops counted), `TestListing_broadcastNeverStarved`, `TestListing_chunkCapCoversEveryDestination`, `TestListing_refusedFirstSeenIsRearmed` (all PASS) — Phase-3 CR-01 |
| 20 | All destinations within one `sync_interval` share one listing build while the store generation is unchanged; any mutation rebuilds exactly once; rotation preserves original chunk indices and the round's count | ✓ VERIFIED | `listingBuild` (age + `store.Generation()`); `TestListing_sharedBuild`, `TestListing_generationRebuild`, `TestRestart_generationInvalidatesCachedListing`, `TestListing_rotationPreservesIndices` |
| 21 | A strictly older `DataReplication` produces exactly one unicast reply carrying the held newer record and NO Ack; the store is not merged and the tracking ledger is untouched | ✓ VERIFIED | `applyRemote` three-way branch (`core.go:1476-1488`); `TestReplication_staleAnsweredNotAcked` (one queued identity, drain emits one `DataReplication` v5, no Ack, merge stats zero, pending unchanged) |
| 22 | An equal-version `DataReplication` produces exactly one Ack and no reply; newer/absent still applies and Acks | ✓ VERIFIED | `TestReplication_equalVersionAcked` (one Ack, 0 queued), `TestReplication_newerApplied`, `TestReplication_absentApplies` |
| 23 | The sender's `wait_for_replication` for a version answered instead of Acked reports `acked: false` through the real adapter tracker | ✓ VERIFIED | `TestWaitIntegration_staleRecordNotAcked` (real adapter pair + real gRPC server; `Acked=false`, total peers 1, B's record untouched) |
| 24 | A local-write notification whose version is not greater than the tracked one is ignored (no broadcast, no ledger change, counted); a strictly older one additionally resolves its waiter `Superseded`; a key's first notification always proceeds | ✓ VERIFIED | `trackLocalWrite` (`core.go:2183-2223`); `TestNotify_firstNotificationProceeds`, `TestNotify_equalTrackedIgnored`, `TestNotify_regressedIgnored` (Superseded result for the notified version, count 1, unacked stays 1), `TestNotify_newerBroadcasts`, `TestNotify_absentKeyBroadcasts` |
| 25 | The reactive egress queue carries identity only and builds the message at drain, re-reading the current record (no record bytes retained, no lookup before dedupe) | ✓ VERIFIED | `enqueueResponse` stores `queuedUnicast{id}`; `resolveQueuedResponse` looks up at drain; `TestEgress_identityOnly` (drain sees the updated record; vanished key skips), `TestEgress_noValueRetention` (Phase-3 CR-02) |
| 26 | A write/delete the engine installed but whose confirming dump failed is still notified (track=false) before the RPC answers `Unavailable`; `ErrClosed` is never notified | ✓ VERIFIED | `server.go:211-221,258-267`; `TestServer_dumpFailedWriteNotifies`, `TestServer_dumpFailedDeleteNotifies`, `TestServer_dumpFailedCreateIfAbsentNotifies`; the notified record enters the core's pending ledger through the same `NotifyLocalWrite` path covered by truth 24 |
| 27 | `Engine.Restore` returns the replayed records in sorted-key order (even when the final flush fails) and a no-op returns an empty list; `SnapshotLoad` notifies every replayed record with track=false and nothing on a no-op/`ErrClosed` | ✓ VERIFIED | `replaySnapshot` (sorted keys, partial list + `markDirty` on error); `TestEngine_restoreReturnsReplayed`, `TestEngine_restoreNoOpReturnsEmpty`, `TestEngine_restoreFailedFlushReturnsReplayed`, `TestEngine_restoreSkipsReplay`; `TestServer_snapshotLoadNotifies`, `TestServer_snapshotLoadUsesReplaySeam`, `TestServer_snapshotLoadReplayFallsBackToLive`, `TestServer_snapshotLoadNoOpDoesNotNotify` |
| 28 | Three in-process adapters on one shared port converge on writes originated by each of them over the real socket path, stating the unicast limitation | ✓ VERIFIED | `TestSmoke_threeNodeBroadcastConvergence` PASS (not skipped on this host: real `udprepl.New(..., WithAddressReuse())`, real codec/read loop/pacing; doc comment names the unicast caveat) |
| 29 | Simulator scenarios prove N=3 convergence, the joiner's complete-listing signal, and F14 value preservation on a replaced node; the 14.9 decision anchor points at the N=3 scenario | ✓ VERIFIED | `TestScenario_threeNode`, `TestScenario_joinerWaitsForCompleteListing`, `TestScenario_f14ReplacedNode` (all PASS); `decision_coverage_test.go:44` maps `14.9 → TestScenario_threeNode` |
| 30 | N-node behavior/timing, `wait_for_replication` at N > 2, the chosen protocol bounds, the test topology, residual windows and handoffs are documented; OPERATIONS lists `bad_sync_header`/`startup-gate-empty`; the phase's SPEC-DEVIATIONS rows are `implemented` | ✓ VERIFIED | `docs/04-MULTI-NODE.md` (all seven required sections present; each claim carries a test/code reference); `docs/OPERATIONS.md:154-157` (eleven discard classes incl. `bad_sync_header`) and `:99` (`--startup-gate-empty`); `docs/SPEC-DEVIATIONS.md` S-PROTO-18, S-HLC-04, S-HLC-10, S-ALG-RC-05, S-ALG-ST-06 rows = `implemented` with evidence links |
| 31 | The red-gate contract holds: both R-03 rows are `green`, both P-01 rows stay `red`, the frozen tests are byte-unchanged and pass under `SUPKV_RED_TESTS=1`, and the runner's mutation stage still kills both mutants | ✓ VERIFIED | `scripts/red-expectations.tsv` (2 green R-03, 2 red P-01); `git diff <phase-base>..HEAD` empty for `internal/stand/stand_red_test.go` and `internal/grpcapi/red_gate_test.go`; direct runs PASS; full `bash scripts/red-runner` independently run by the verifier: 7/7 rows OK, 4/4 control seeds pass, both mutants killed, exit 0 |
| 32 | The Phase-3 review criticals CR-01/CR-02 are closed with fixing commits and regression tests | ✓ VERIFIED | `03-REVIEW-DISPOSITION.md` rows cite `1e06c58` / `e42a20a`; the code enforces identity-keyed bounded listings and identity-only egress (truths 19/25); the named regression tests pass |
| 33 | The module is clean on the verified tree: build, vet, strict lint, race test suite, buf lint, darwin cross build/vet, tidy | ✓ VERIFIED | `go build ./...` 0; `go vet ./...` 0; `make lint` 0; `make test` (`go test -race -cover ./...`) 0; `make buf-lint` 0; `GOOS=darwin go build/vet ./...` 0; `go mod tidy -diff` empty |

**Score:** 33/33 truths verified (0 present-but-behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Final public API wording of `wait_for_replication` at N > 2 in `docs/API.md` | Phase 9 (DOCS-05; semantics owned by Phase 6 APIS-01) | `docs/API.md` is DOCS-05's deliverable and does not exist yet; this phase ships the semantics in `docs/04-MULTI-NODE.md` with the handoff recorded (D-08). Not a phase-4 gap: the phase artifact is the agreed location |

---

### Required Artifacts

All plan-declared artifacts passed `gsd-tools query verify.artifacts` (13/13 plans, `all_passed: true`, e.g. plan 04-01: 6/6). Spot-checked substance (not just existence):

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/udprepl/replproto/supkv/v1/supkv.proto` | DataSync header fields | ✓ VERIFIED | `sync_id=2`, `chunk_index=3`, `chunk_count=4`; generated `supkv.pb.go` committed |
| `internal/port/repl.go` | Header fields + `ProtoVersionMismatch` verdict | ✓ VERIFIED | 243 lines, documented verdict ordering |
| `internal/app/repl/core.go` | Accumulation, retention binding/expiry, listing build/queues, stale branch, notify guard, `SupportedProtoVersion=2` | ✓ VERIFIED | 3300 lines; every seam cited above is implemented, no stub returns in change paths |
| `internal/udprepl/repl.go` | Mirror, pre-body version verdict, header mapping | ✓ VERIFIED | 1327 lines; decode order :1200/:1208/:1212/:1217 |
| `internal/grpcapi/gate.go`, `server.go`, `watch.go` | Gate-first reads/writes, listed-key semantics | ✓ VERIFIED | all handlers gate first; `ListedKey`/`SyncArrived` options documented |
| `internal/app/storage/store.go` | `Observe`, `Len`, `Generation` | ✓ VERIFIED | generation bumps once inside `install` only |
| `internal/app/persist/engine.go` | `Observe`, `KeyCount`, `Generation`, `Restore []RestoredRecord` | ✓ VERIFIED | delegations + sorted replay with partial return |
| `cmd/sup-kv/main.go` | `startup_gate_empty` flag/env/config/selection/wiring | ✓ VERIFIED | `ListedKey: adapter.ListedKey`, `SyncArrived`, `ReplayNotify` all injected |
| `internal/stand/stand.go` | Empty-gate derivation | ✓ VERIFIED | both dash/value spellings; explicit empty gate wins |
| `docs/04-MULTI-NODE.md` | N-node artifact | ✓ VERIFIED | 237 lines, seven sections |
| `scripts/red-expectations.tsv` / `docs/OPERATIONS.md` / `docs/SPEC-DEVIATIONS.md` | Red flip, ops handoff, registry rows | ✓ VERIFIED | as in truths 30/31 |

### Key Link Verification

The automated `verify.key-links` verb reports 0/N for most plans because plan `via:` strings are prose (and the core↔adapter links are method-level calls across a depguard-enforced boundary, so the `to` path cannot appear literally in the `from` file). Each link was therefore traced manually; all are wired.

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `internal/app/repl/core.go` | `internal/udprepl/repl.go` | core listing accessors → adapter mirror | ✓ WIRED | `mirrorStats` calls `core.ListingGeneration/ListingComplete/ListedKeysSnapshot` (`repl.go:1118-1123`) |
| `internal/udprepl/repl.go` | `internal/app/repl/core.go` | `ProtoVersionMismatch` envelope fields → attributed discard | ✓ WIRED | `decode` sets `ProtoVersion/NodeID/RunID`; `HandleDatagram` counts and warns (`core.go:1017-1025`) |
| `internal/udprepl/repl.go` | `internal/app/repl/core.go` (whitelist test) | header size honesty | ✓ WIRED | `chunkOverheadBytes=96`; `TestSync_chunkWireBudget` uses the emitted chunk's own header |
| `internal/grpcapi/gate.go` | `internal/udprepl/repl.go` | `main` injects `adapter.ListedKey`/`SyncArrived` | ✓ WIRED | `main.go:599-600`; one atomic pointer backs both |
| `internal/grpcapi/server.go` | `internal/app/persist/engine.go` | `Engine.Version` distinguishes tombstone vs unarrived | ✓ WIRED | `server.go:150` |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | per-part `Observe` | ✓ WIRED | `core.go:1637`; `Engine.Observe` → `Store.Observe` → `clock.Update` |
| `internal/app/persist/engine.go` | `internal/app/storage/store.go` | `KeyCount`/`Generation` delegation | ✓ WIRED | `engine.go:542,550` |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `nodeStore.Observe/Generation` | ✓ WIRED | `simnet.go:453,456` |
| `internal/grpcapi/wait_integration_test.go` | `internal/app/repl/core.go` | real adapter tracker observes no-Ack | ✓ WIRED | `TestWaitIntegration_staleRecordNotAcked` PASS |
| `internal/app/repl/core.go` | `internal/app/repl/restart_test.go`, `sim_scenario_test.go` | first-seen/restart contracts | ✓ WIRED | tests named in truths 16-18 PASS |
| `internal/udprepl/three_node_test.go` | `internal/udprepl/smoke_test.go` | shared helpers | ✓ WIRED | `freeUDPPort`, `smokeSocket`, `probeBroadcastAddr` all present and used |
| `internal/app/repl/decision_coverage_test.go` | `internal/app/repl/sim_scenario_test.go` | 14.9 anchor | ✓ WIRED | `{"14.9", "0009-multi-node-support.md", "TestScenario_threeNode"}` |
| `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go`, `internal/grpcapi/red_gate_test.go` | expected verdict only | ✓ WIRED | TSV flip commit `a44e661` touches TSV + runner only; frozen diffs empty |
| `03-REVIEW-DISPOSITION.md` | `internal/app/repl/core.go` | fixing-commit references | ✓ WIRED | `1e06c58` / `e42a20a` exist and carry the fixes |

### Data-Flow Trace (Level 4)

This is a backend protocol phase; the only "rendered" data are API answers derived from live state. Traced paths:

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `Get`/`Set` listed-key decisions | `ListedKey(key)` | `core.listedKeys` union ← `handleDataSync` ← decoded `DataSync` datagram → adapter `listingMirror` → `ServerOptions.ListedKey` | Yes — real inbound listing entries, no literal/static fallback | ✓ FLOWING |
| Gate/health readiness | `SyncArrived()` | `core.listingComplete` ← `accumulateSync` completeness predicate → mirror → `RunGate` poll → `gateOpen`/health | Yes — flips from real chunk arrivals (asserted by partial-listing and frozen stand tests) | ✓ FLOWING |
| Stale-record answer | queued `egressKey` | `applyRemote` identity enqueue → `resolveQueuedResponse` → `store.Lookup` at drain | Yes — drain serves the record current at drain time (`TestEgress_identityOnly`) | ✓ FLOWING |
| Dump-failed/replay notify | `RestoredRecord`/record | real `engine.Set/Delete/Restore` result → `grpcapi` notify seam → adapter → `core.NotifyLocalWrite`/`NotifyReplayedWrite` | Yes — store-stamped versions asserted (`TestServer_dumpFailed*`, `TestServer_snapshotLoadNotifies`) | ✓ FLOWING |
| Listing build cache | `listingBuild.chunks` | `store.Versions()` copy, invalidated by `store.Generation()` | Yes — rebuild-on-mutation asserted (`TestListing_generationRebuild`, `TestRestart_generationInvalidatesCachedListing`) | ✓ FLOWING |

No `STATIC`, `DISCONNECTED` or `HOLLOW_PROP` paths found.

### Behavioral Spot-Checks

Run on the verified tree (2026-11-22, this session); nothing started or mutated.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Frozen R-03 gate closed-until-complete | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpcapi/ -run '^TestRed_R03_gateClosedUntilFullList$' -v` | `--- PASS (0.03s)` | ✓ PASS |
| Frozen R-03 F14 replaced node (full acceptance) | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_R03_f14ReplacedNode$' -v` | `--- PASS (3.15s)` | ✓ PASS |
| Full red-gate table + mutation stage | `bash scripts/red-runner` | 7/7 rows OK; 4/4 control seeds pass; `sync-without-tombstones` and `periodic-sync-disabled` both killed; exit 0; evidence `red-evidence/20261122T084157Z` | ✓ PASS |
| CR-01 retention binding/expiry/no-row | `go test -race -count=1 ./internal/app/repl/ -run '^(TestSync_unboundListingDiscarded\|TestSync_forgedRunIDCannotReplaceListing\|TestSync_emptyListingCreatesNoRow\|TestSync_listedKeysExpire\|TestSync_listedKeysBounded\|TestSync_listingGenerationContentOnly)$' -v` | 6/6 PASS | ✓ PASS |
| Listing/restart/peer/notify/replication suites | `go test -race -count=1 ./internal/app/repl/ -run '^(TestListing_\|TestRestart_\|TestPeer_\|TestNotify_\|TestReplication_)' -v` | 31/31 PASS | ✓ PASS |
| N=3 simulator scenarios | `go test -race -count=1 ./internal/app/repl/ -run '^(TestScenario_threeNode\|TestScenario_joinerWaitsForCompleteListing\|TestScenario_f14ReplacedNode)$' -v` | 3/3 PASS | ✓ PASS |
| N=3 real-socket convergence | `go test -race -count=1 ./internal/udprepl/ -run '^TestSmoke_threeNodeBroadcastConvergence$' -v` | PASS (0.06 s, not skipped) | ✓ PASS |
| Review-fix regressions (WR-01…WR-06) | `go test -race -count=1 ./internal/app/repl/ -run '^(TestNotifyReplayed_pacedNotImmediate\|TestSync_streamMapHintIndependentOfClaim\|TestConfig_syncChunkBytesClamped\|TestListing_broadcastNeverStarved\|TestListing_refusedFirstSeenIsRearmed\|TestRestart_retainsListedKeysUntilReplacement\|TestChunkSyncEntries_capacityFollowsEntries)$' -v` | 7/7 PASS | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` exists and no plan declares a standalone probe script. The phase's executable probes are the red-runner's D-02 overlay mutants and the review's overlay proofs; both were re-run:

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| `sync-without-tombstones` mutant | `bash scripts/red-runner` (mutation stage) | killed on seed 848612892191557850 (expected fail) | PASS |
| `periodic-sync-disabled` mutant | `bash scripts/red-runner` (mutation stage) | killed on seed 848612892191557850 (expected fail) | PASS |
| CR-01 overlay proofs (unbound listing, forged run) | Named tests above + the review's re-verification record | 0 keys retained, 0 rows, gate closed; real run's set intact | PASS |

### Phase Gate (re-run by the verifier)

| Gate | Command | Exit | Result |
| ---- | ------- | ---- | ------ |
| Build | `go build ./...` | 0 | clean |
| Vet | `go vet ./...` | 0 | clean |
| Lint | `make lint` | 0 | config verify + `golangci-lint run ./...` clean |
| Tests (race) | `make test` (`go test -race -cover ./...`) | 0 | all packages ok; `internal/app/repl` 185.1 s / 93.1% coverage; `internal/stand` 84.5%; `internal/udprepl` 87.0%; `internal/grpcapi` 94.6% |
| Red gate | `bash scripts/red-runner` | 0 | 7/7 rows, mutants killed (above) |
| Buf lint | `make buf-lint` | 0 | clean |
| Darwin cross | `GOOS=darwin GOARCH=arm64 go build/vet ./...` | 0 | clean |
| Module tidy | `go mod tidy -diff` | 0 | no diff |
| Buf breaking | not re-run by the verifier (needs `origin/master` fetch); recorded exit 0 in `04-13-SUMMARY.md` | — | accepted from the phase gate record |

`TestCore_keyCountCostFlat` (documented wall-clock ratio flake, deferred to Phase 8) did not reproduce on this run.

### Requirements Coverage

All nine requirement IDs are declared in plan frontmatter (04-01…04-13) and are mapped to Phase 4 in REQUIREMENTS.md; no orphaned requirements.

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| PROTO-01 | 04-01, 04-02, 04-03, 04-10, 04-13 | Interface closed until the full key list (R-03, 14.5) | ✓ SATISFIED | Truths 1-9, 11; frozen `TestRed_R03_gateClosedUntilFullList` + F14 PASS; simulator F14/joiner scenarios; TSV green |
| PROTO-02 | 04-03, 04-04, 04-13 | Empty-disk node waits for a peer longer (14.6) | ✓ SATISFIED | Truths 12-15 |
| PROTO-03 | 04-05, 04-13 | Stale `DataReplication` answered with the newer record, not acked (R-04) | ✓ SATISFIED | Truths 21-23, 25 |
| PROTO-04 | 04-08, 04-11, 04-13 | New/returning peer gets an immediate sync (R-05) | ✓ SATISFIED | Truths 16-20 |
| PROTO-05 | 04-09, 04-13 | Local-write notifications never regress (R-10) | ✓ SATISFIED | Truth 24 |
| PROTO-06 | 04-09, 04-13 | Duplicate `node_id` with alternating `run_id`s is a named config error (R-12) | ✓ SATISFIED | Truth 17 |
| PROTO-07 | 04-01, 04-11, 04-13 | `proto_version` checked before body decode (R-13) | ✓ SATISFIED | Truth 10 (incl. future-version-over-corrupt-body row) |
| PROTO-08 | 04-06, 04-13 | Every local change replicates, incl. dump-failed writes and snapshot loads (S-02) | ✓ SATISFIED | Truths 26-27 |
| PROTO-09 | 04-07, 04-10, 04-12, 04-13 | Three-or-more-node behavior described and tested (14.9) | ✓ SATISFIED | Truths 28-30; final API-doc location deferred (see Deferred Items) |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/app/repl/*_test.go` (sync/listedkeys/listing/restart/peertable/retransmit/egress/sim/warn/merge_stats) | PROTO-01,03,04,05,06,09 | Yes | 0 (red tests live in stand/grpcapi/persistfile) | None | Value/Behavioral (versions, record bytes, counts, message shapes) | ✓ |
| `internal/udprepl/*_test.go` (decode_negative, syncsize, three_node, repl, fuzz) | PROTO-07,01,09 | Yes; `TestSmoke_threeNode…` PASS (not skipped) | capability-probe skips only, loud with reason and not taken here | None | Value/Behavioral (wire bytes, store convergence) | ✓ |
| `internal/grpcapi/*_test.go` (gate, server, wait_integration) | PROTO-01,03,08 | Yes | 0 | None | Status + Value + Behavioral (real gRPC + real adapters) | ✓ |
| `internal/stand/*_test.go` | PROTO-01,02 | Yes | red tests env-gated by design; the two R-03 rows run under `SUPKV_RED_TESTS=1` and PASS | None | Behavioral (process/socket lifecycles) | ✓ |
| `internal/app/storage/store_test.go`, `internal/app/persist/engine_test.go` | PROTO-01,02,08 | Yes | 0 | None (fixtures hand-built; `os.WriteFile` used for corruption fixtures only) | Value/Behavioral | ✓ |

**Disabled tests on requirements:** 0 (the env-gated red tests are the D-01 mechanism, exercised via the red runner). **Circular patterns detected:** 0. **Insufficient assertions:** 0.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX` marker in any phase-modified Go/proto file | — | Debt-marker gate clean |
| — | — | No unreferenced `TODO`/`HACK`/`PLACEHOLDER`/empty implementation in production files | — | — |
| `cmd/sup-kv/main_test.go` | 481, 584 | comment word "placeholders" about loopback addresses | ℹ️ Info | Documentation comment, not a stub; the tests assert real behavior |
| — | — | No `bad_sync_header`/listing path uses static returns; all stubs checked for data-flow | — | — |

### Decision Coverage

All 9 trackable CONTEXT.md decisions (D-01…D-09) are honored by shipped artifacts: `{honored: 9, total: 9, not_honored: []}` (gate message: "All trackable CONTEXT.md decisions are honored by shipped artifacts"). The research-resolution refinements (strictly-older-only stale answer, equal-version Ack, pre-body version verdict, proto v1→v2, 96-byte overhead, harness empty-gate derivation) are also present in code and tests.

### Review Fix Verification (04-REVIEW / 04-REVIEW-DISPOSITION)

The phase review's findings were checked in the code rather than trusted from the ledger:

| Finding | Ledger says | Independent check |
| ------- | ----------- | ----------------- |
| CR-01 (new critical: listed-key retention binding) | fixed `e5b8d0d` (+`41825f8`), re-verified | Code enforces all three review requirements: `listingBound()` requires a peer-table `(node_id, run_id)` row before any clock/accumulation/completeness; `expireListedKeys()` releases sets not republished within 2 × `sync_interval`; `touchListedPeer()` creates no row for an empty listing. `TestSync_unboundListingDiscarded`, `TestSync_forgedRunIDCannotReplaceListing`, `TestSync_emptyListingCreatesNoRow`, `TestSync_listedKeysExpire` all PASS |
| WR-01…WR-06 | fixed with commits | Each has a currently-passing regression test (see Behavioral Spot-Checks: `TestListing_broadcastNeverStarved`, `TestListing_refusedFirstSeenIsRearmed`, `TestRestart_retainsListedKeysUntilReplacement`, `TestNotify_regressedIgnored` (Superseded), `TestNotifyReplayed_pacedNotImmediate`, `TestSync_streamMapHintIndependentOfClaim`, `TestConfig_syncChunkBytesClamped`) |
| IN-01…IN-06 | fixed (IN-03 deferred) | IN-02 eleven-class list present in OPERATIONS; IN-04 partial-return + markDirty present; IN-05 both flag spellings handled; IN-06 single `listingComplete`; IN-03/New-1/2/3/5 remain deferred to Phases 5/7/8 as recorded — info-level, none a phase must-have |

### Gaps Summary

None. The phase goal is achieved in the codebase and under test:

- The interface is closed until the complete listing (frozen R-03 tests pass unmodified, on real sockets and in the simulator), with per-part HLC absorption and the accepted 14.5(6) read semantics.
- Empty-disk/returning peers, stale records, duplicate identities and version mismatch all have defined, implemented, tested behavior.
- The review's new critical (CR-01 retention binding) is enforced by code plus four named tests, and the Phase-3 criticals CR-01/CR-02 are closed with fixing commits.
- Replication totality (dump-failed writes, snapshot loads) and N=3 real-socket/simulator behavior are present and tested; the multi-node artifact documents the bounds and handoffs.
- Independent gates are green, and the red-gate contract holds byte-for-byte (TSV rows only).

One roadmap fragment — the final `docs/API.md` location of the N > 2 `wait_for_replication` wording — is explicitly owned by later phases (DOCS-05/APIS-01) and is recorded under Deferred Items, not as a gap; the semantics ship here in `docs/04-MULTI-NODE.md`.

### Human Verification Required

N/A — infrastructure/protocol phase with no user-facing elements; every acceptance criterion is covered by automated stand-level real-socket tests, simulator scenarios, core/RPC tests, and the frozen red-gate tests. No behavior-unverified truths (`behavior_unverified: 0`).

---

_Verified: 2026-11-22T08:44:54Z_
_Verifier: Claude (gsd-verifier)_
