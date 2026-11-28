---
phase: 03-transport
verified: 2026-11-27T16:45:15Z
status: passed
score: 17/17 must-haves verified
covered_files:
  - .github/workflows/scheduled.yml
  - .planning/phases/03-transport/03-01-PLAN.md
  - .planning/phases/03-transport/03-01-SUMMARY.md
  - .planning/phases/03-transport/03-02-PLAN.md
  - .planning/phases/03-transport/03-02-SUMMARY.md
  - .planning/phases/03-transport/03-03-PLAN.md
  - .planning/phases/03-transport/03-03-SUMMARY.md
  - .planning/phases/03-transport/03-04-PLAN.md
  - .planning/phases/03-transport/03-04-SUMMARY.md
  - .planning/phases/03-transport/03-05-PLAN.md
  - .planning/phases/03-transport/03-05-SUMMARY.md
  - .planning/phases/03-transport/03-06-PLAN.md
  - .planning/phases/03-transport/03-06-SUMMARY.md
  - .planning/phases/03-transport/03-07-PLAN.md
  - .planning/phases/03-transport/03-07-SUMMARY.md
  - .planning/phases/03-transport/03-08-PLAN.md
  - .planning/phases/03-transport/03-08-SUMMARY.md
  - .planning/phases/03-transport/03-09-PLAN.md
  - .planning/phases/03-transport/03-09-SUMMARY.md
  - .planning/phases/03-transport/03-10-PLAN.md
  - .planning/phases/03-transport/03-10-SUMMARY.md
  - .planning/phases/03-transport/03-11-PLAN.md
  - .planning/phases/03-transport/03-11-SUMMARY.md
  - .planning/phases/03-transport/03-12-PLAN.md
  - .planning/phases/03-transport/03-12-SUMMARY.md
  - .planning/phases/03-transport/03-13-PLAN.md
  - .planning/phases/03-transport/03-13-SUMMARY.md
  - Makefile
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - docs/adr/0010-darwin-development-only.md
  - internal/app/internal/simnet/simnet.go
  - internal/app/internal/simnet/simnet_ext_test.go
  - internal/app/internal/testutil/gen.go
  - internal/app/persist/codec.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/persist/facade.go
  - internal/app/repl/bench_test.go
  - internal/app/repl/config_internal_test.go
  - internal/app/repl/core.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/egress_internal_test.go
  - internal/app/repl/merge_stats_test.go
  - internal/app/repl/peertable_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_chaos_test.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/repl/sync_test.go
  - internal/app/repl/warn_test.go
  - internal/app/storage/store.go
  - internal/app/storage/view_test.go
  - internal/app/version/validate_test.go
  - internal/app/version/version.go
  - internal/config/config.go
  - internal/config/validate.go
  - internal/config/validate_test.go
  - internal/grpcapi/bench_test.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait.go
  - internal/grpcapi/wait_integration_test.go
  - internal/grpcapi/wait_test.go
  - internal/metrics/metrics.go
  - internal/metrics/metrics_test.go
  - internal/persistfile/fsops.go
  - internal/port/repl.go
  - internal/stand/stand.go
  - internal/stand/stand_buffers_test.go
  - internal/stand/stand_red_test.go
  - internal/stand/stand_test.go
  - internal/testutil/ports.go
  - internal/testutil/ports_test.go
  - internal/udprepl/adapter.go
  - internal/udprepl/codec.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/loop.go
  - internal/udprepl/maxdatagram.go
  - internal/udprepl/maxdatagram_test.go
  - internal/udprepl/mirror.go
  - internal/udprepl/readloop_internal_test.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/scenario_test.go
  - internal/udprepl/smoke_test.go
  - internal/udprepl/socket.go
  - internal/udprepl/socket_internal_test.go
  - internal/udprepl/syncsize_internal_test.go
  - scripts/red-expectations.tsv
  - scripts/red-runner
covered_digest: "v4:sha256:36ef46805b1695507e39cbd85fd81d0c632bce8ebf1fd52fcd353059c6a5df0c"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 17/17
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Stale-fingerprint regeneration, not a gap-closure round. The previous report's digest covered the pre-Phase-8 tree; Phase 8 perf work subsequently touched transport-adjacent inputs (the per-key pending index f162bd5/462a8d6/f914bba, the pending-visit cost gates and retransmit-fixture gates d9e9d44/6acb2a6/147a154, the shared claim-while-held port allocator 074ccd0/5c7a3d7/3e7b465/7aeec35 and its adoption in main_test.go/stand.go/udprepl/grpcapi, plus Makefile/docs edits). This run re-verified every must-have against the current tree ef370a0: all machine-verifiable bounds re-run green first-hand, the TEST-11 class re-confirmed at count=3, the red-runner mutation stage reproduced standalone green (its core.go-targeted sed survives the Phase-8 file), and the wall-clock R-01 class re-confirmed as the adjudicated host artifact (03-12 halt record, 4a83919 resolution, scheduled red-r01-x20 CI acceptance)."
---

# Phase 3: Transport Verification Report

**Phase Goal:** The UDP transport meets the review's stated bounds on real sockets — sending is paced and bounded, buffers are configured and reported, input is validated before it can touch the clock — so the recovery and burst scenarios converge within their documented limits.
**Verified:** 2026-11-27T16:45:15Z at tree `ef370a0`
**Status:** passed (all 17 must-haves verified against the current tree)
**Re-verification:** Yes — stale-fingerprint regeneration. The previous VERIFICATION.md (passed, 17/17, no gaps) covered source files Phase 8 performance work later changed; this run re-verified the phase goal against the CURRENT codebase and recomputed the covered-input fingerprint through `verification.fingerprint`.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | DataSync listings are enqueued and drained at one datagram per `floor(1s/sync_response_rate)` with one in-flight round per destination, and the sending start chunk rotates across rounds (R-01) | ✓ VERIFIED | `syncResponseInterval = time.Second/SyncResponseRate` (`core.go:1036`); `syncListing:2534`, one FIFO per destination in `enqueueListing:2556`, `drainListings:2726` emits one datagram per Advance and advances by `jitteredSpacing:2814` (±10 %); `admitListing:2584` starts at `rotation % len(chunks)` (`:2586`). `TestSync_pacing` (spacing inside `[interval−10 %, interval+10 %]`, at most one datagram per Advance, first chunk immediate), `TestSync_rotation`, `TestSync_listingBusy` and `TestSync_triggers` all PASS under `-race` at this tree (2026-11-27). |
| 2 | Two nodes on real Linux sockets with 20 000 keys × 1 KiB into an empty receiver converge within 2 × `sync_interval` + N / `sync_response_rate`; the R-01 red row is green; a 20× documented-scale CI job carries the 20/20 acceptance (R-01) | ✓ VERIFIED | The bound is derived from the flags the nodes run with and is unchanged since the blessed alignment (`stand_red_test.go:321` `deadline = 2*syncInterval + paced + margin`; file last touched `6c17db3` 2026-11-22). All machine-verifiable halves re-verified first-hand at this tree: pacing/rotation/one-in-flight (truth 1), buffer read-back logging (truth 3), the cheap key→version view (truth 13), and — on real sockets on this host — R-02 drain and burst PASS (truths 5-6). The 20/20 wall-clock acceptance is the scheduled `red-r01-x20` job (`.github/workflows/scheduled.yml:149-183`: `-count=20`, `-timeout 110m`, `timeout-minutes: 120`, both env gates, no `|| true`/skip flags). Evidence-scope note 1: this host runs powersave-throttled; today's two local red-runner runs (untracked evidence `20261127T161311Z`, `20261127T161629Z`) fail only the R-01 row at the LOCAL 8 000-key scale (42.64 s) — the exact artifact the 03-12 halt record documents (7 453–7 734/8 000 at the ~20.6 s bound; only a 4.2 s margin over the 16 s paced term). Per the recorded Phase-3 decisions (03-12 halt + orchestrator resolution; 03-13: "the nightly schedule owns XPORT-01's 20/20 acceptance; the job's first scheduled run is the remaining observation rather than a code gate"; flip cleared by full-speed-host evidence commit `4a83919`, `red-evidence/20261121T225048Z`) a local wall-clock miss of this class is a host environment artifact, not a code gap — the flip stays cleared and the acceptance stays with the scheduled job. |
| 3 | `SO_RCVBUF`/`SO_SNDBUF` are set with the actual kernel values read back and logged at startup; the core consumes a cheap key→version view instead of a store copy (R-01) | ✓ VERIFIED | `configureSocketBuffers` sets then `getsockopt`-reads both sizes (`socket.go:136-176`); `checkSendBuffer:202` is the pure inclusive floor over `datagramBufSize = 65507` (`:18`); `udprepl.New` stores the read-back and fails construction on the send floor; `logSocketBuffers` emits the effective sizes (`main.go:679-682`, wired at `:622`). `TestSocketBuffers` (values equal a second `getsockopt` read, not the requested constants) and `TestMain_socketBuffersLogged` PASS. `repl.StateStore` is the six-method interface `ApplyRemote/Version/Lookup/Versions/Observe/Generation` (`core.go:402-435`) — no `Records` (the only "Records" hits are `storage.RecordsEqual` and comments), so the O(store) snapshot path stays compile-impossible; the listing is built from `Versions()`. |
| 4 | Event-loop iteration cost at 10 and 10 000 unacked records differs by at most 2× (R-02(1)) | ✓ VERIFIED | Phase 8 (08-05) replaced the old wall-clock ratio gate with a STRONGER deterministic one: `TestCore_unackedCostFlat` (`bench_test.go:398`) asserts an idle event-loop iteration inspects ZERO pending entries at 10 and at 10 000 unacked records ("the timer path must not scale with the pending set (XPORT-02)") — zero inspection implies time-flatness, and the count cannot be moved by host load. PASS at this tree; `BenchmarkCoreEventLoop` remains measurement-only. The ledger is record-free (`pendingEntry` at `core.go:726`, `pending` keyed by `(key, version)`, per-peer `peerUnacked` ledger, indexed `retryHeap:750`, Phase-8 `pendingByKey` per-key index with the two-site maintenance contract documented at `core.go:820-835`); `UnackedStats:3397` and `PeerCount:3370` are O(1). New companion gates `TestCore_notifyWriteCostFlat`/`TestCore_applyRemoteCostFlat` prove the write and apply paths key-local at any set size. |
| 5 | A peer returning to a node holding 10 000 unacked 1 KiB records drains within N / `sync_response_rate` + 10 s with at most 3N retransmits (R-02(2)) | ✓ VERIFIED | `TestRed_R02_unackedDrain` PASS first-hand at this tree (2026-11-27, 41.03 s; precondition logged `10000/10000 unacked`); the assertion polls to the documented bound and caps retransmits at `3*redDrainKeys` (`stand_red_test.go:441`) from `supkv_replication_datagrams_retransmitted_total`. Drain path paced (`drainRecordSends:3032`), the ladder re-arms from actual emission (`emitRecordSend:3060` — `c.retransmitted++` only when `rs.retry`, `:3075`), and the record is re-read via `Lookup` with a version-mismatch skip. |
| 6 | Open-loop 1000 writes/s × 15 s over 10 000 distinct keys achieves ≥990 writes/s with the unacked set empty 5 s after the send window; the rate window is the open-loop send window and the ≤5 s drain assertion is separate (R-02(3)) | ✓ VERIFIED | `TestRed_R02_burst` PASS first-hand at this tree (2026-11-27, 15.91 s). The rate denominator is the open-loop send window and the drain bound is a separate send-end + 5 s + poll-margin check (`stand_red_test.go`, unchanged since `6c17db3`) — matching `REQUIREMENTS.md`'s two independent clauses. |
| 7 | Key length and `value_max_bytes` ceilings are enforced: `Set`/`Delete` return `InvalidArgument` over 1 024 bytes before any engine call, startup exits 2 when the largest accepted record cannot fit one datagram, and any accepted record encodes into ≤65 507 bytes (R-03, XPORT-03) | ✓ VERIFIED | `storage.MaxKeyBytes = 1024` (`store.go:33`); API checks at `grpcapi/server.go:203,270` with `keyTooLongError`, while `Get` stays permissive. `checkDatagramCeiling` measures through `udprepl.MaxRecordDatagramBytes` and is in the rule table (`internal/config/validate.go:56-57`); `main` maps any `config.Validate` failure to `exitUsage = 2` (`main.go:64,172,184`). `TestMain_valueMaxDatagram` proves 64 000/64 408 start and 64 409/65 500 exit 2; `TestValidate` (config) rejects 64 409 naming 65 507; `TestMaxRecordDatagramBytes_acceptedDomain` walks the accepted rectangle through the real codec; `TestStand_oversizeKeyF2c` PASS first-hand (3.90 s) proves local rejection of a 70 000-char key, non-replication, and the positive boundary. |
| 8 | Received versions (including versions inside `DataSync`) are validated for RFC 9562 layout and the `now + max_clock_lead` bound before any store or clock call, counted as their own class, and WARN-logged rate-limited (XPORT-04) | ✓ VERIFIED | `HandleDatagram` runs `c.validate(now, in)` between classification and `handleMessage` (`core.go:1094-1180`, pipeline documented at `:1171-1181`), and `handleMessage` is the only path to `ApplyRemote` (the sole `Clock.Update`). `validateReplication:1201` pins empty key → too long → tombstone-with-data → checksum → version; `validateSync:1225` validates the header then every entry in order; `version.RecordVersion.Validate` is pure/total, bounds via subtraction, accepts exactly `now+maxLead`, ignores the 46 RFC 9562 filler bits. `TestDiscard_beforeClock` (counting clock: `Tick == 0 && Update == 0`, no store record), `TestDiscard_warnRateLimited`, and the version boundary/property tests all PASS. `--max-clock-lead` → `repl.Config.MaxClockLead` (`main.go:708`); default 2 m (XPORT-04 requires a configurable bound, pinned by `TestMain_maxClockLead`). |
| 9 | Invalid datagram combinations (empty key, key too long, tombstone with data, bad checksum, bad version) are discarded with their own counters and source-only details; envelope CRC coverage is documented (XPORT-10) | ✓ VERIFIED | `DiscardStats` has one counter per class; `discardValidated:1287` increments the class, fires `OnDiscard(class, source)`, and never carries a key name; `validateReplication` recomputes `storage.RecordChecksum` before apply. `TestDiscard_validationClasses`, `TestDiscard_beforeClock`, `TestDecode_negativeTable` PASS. Docs: `docs/SPEC-DEVIATIONS.md` S-PROTO-01 (`:69`) states the CRC covers the body bytes only; `docs/OPERATIONS.md:296-298` lists the discard classes. The adapter mirrors twelve classes into `supkv_datagrams_discarded_total` (`mirror.go:14-33`, `metrics.go:426`). |
| 10 | The read loop exits only on `net.ErrClosed`; any other read error is counted, logged rate-limited, backed off, and survived (XPORT-08) | ✓ VERIFIED | `Adapter.readLoop` returns only on `errors.Is(err, net.ErrClosed)` (`loop.go:480`); otherwise `readErrors.Add(1)` (`:483`), rate-limited log, 100 ms backoff, continue. `TestReadLoop_survivesNonClosedError` (fake conn: error then datagram still processed) and `TestReadLoop_persistentErrorExitsOnClose` PASS; `TestSmoke_twoNodesUDP`/`TestMirror_unackedSent` PASS. |
| 11 | Startup logs the actual `SO_RCVBUF`/`SO_SNDBUF` and fails when the effective send buffer cannot hold a maximal datagram; 1, 20 and 60 KiB values deliver on Linux; the darwin ~9 KiB limit is documented, not fixed (XPORT-05) | ✓ VERIFIED | `checkSendBuffer:202-207`: 65 506 rejected naming requested and actual sizes, 65 507 accepted (`TestCheckSendBuffer`); `New` closes the socket and returns the error. `TestStand_bufferSizesDeliver` PASS first-hand (0.51 s): 1 024 / 20 480 / 61 440-byte values delivered byte-exact between two real binaries, with positive parsed buffer lines on both. darwin: `docs/adr/0010-darwin-development-only.md` records "~9 KiB send limit documented, not fixed". |
| 12 | Bind-address misuse is loud, duplicate instances fail, and timers jitter: a non-wildcard `--bind-addr` WARNs, a second instance fails with "address already in use", a node hearing no `Hello` for `peer_timeout` WARNs once, and Hello/sync deadlines are positive-only while the retransmit ladder is never jittered (XPORT-09) | ✓ VERIFIED | `warnOnNonWildcardBind` (`main.go:328`), `TestMain_bindAddrWarn` PASS. Production `listenDatagram` (`socket.go:97`) sets no `SO_REUSEADDR` unless the explicit `WithAddressReuse()` test option (`:83`) is passed; `TestMain_duplicateBind` PASS ("address already in use"), `TestListen_noAddressReuse` PASS. `warnIfNoPeer` latches one WARN after `firstAdvance + PeerTimeout` (`core.go:2496`), `TestCore_noPeerWarn` PASS. `jitteredSpacing:2814` is positive-only ±10 % on pacers; `TestSync_pacing`/`TestSync_triggers` pin the envelopes and `TestRetransmit_backoffLadder`/`TestBound_lossWithinAckTimeoutMax` stay un-jittered — all PASS. |
| 13 | The store exposes a key→version view and point reads with tombstone visibility, and `DataRequest`/`DataSync` handling does not depend on foreign records (flat between 1 000 and 50 000 stored keys) (XPORT-11, XPORT-06) | ✓ VERIFIED | `Store.Version` (one map read, no Data copy), `Store.Lookup` (clone, `Deleted` visible), `Store.Versions` (fresh shallow map); `Get` stays live-only. `view_test.go` (tombstone visibility, clone ownership, snapshot independence, concurrent hammer) PASS. `Engine.Lookup/Version/Versions` forward via `facade.go`; `simnet.nodeStore` implements the six-method interface. `TestCore_keyCountCostFlat` PASS — now the deterministic form: exactly one `Version`+`Lookup` per DataRequest iteration and exactly `chunkEntries` `Version` calls per DataSync entry, identical at 1 000 and 50 000 keys (`bench_test.go:219-237`); the ns/op benchmarks remain measurement-only. |
| 14 | "Record acknowledged by peer X" travels as an `Outbox` result to the adapter and to waiters, removing snapshot diffing and the 5 ms polling in `wait.go` (XPORT-12) | ✓ VERIFIED | `port.Outbox` carries two independent streams: `Drain()` and `Ack/Drained/Superseded/WaitSetEmpty` + `DrainResults()` (`internal/port/repl.go`); the core emits at every resolution site. The adapter creates the tracker synchronously in `NotifyLocalWrite` before the enqueue (`loop.go:234-252`) and deletes it on every return path; `wait.go` has no `time.Sleep`/`Ticker`/poll/snapshot-diffing (grep over the file finds only the doc comment); `ServerOptions.WaitReplicated: adapter.WaitReplicated` wired at `main.go:637`. `TestNotify_createsTrackerBeforeCore`, `TestWaitReplicated_*` (ackCoverage, strangerAckCounts, evictionNotAck, superseded, ctx/timeout release) PASS. |
| 15 | The sync-answer queue is bounded and deduplicated: 1 000 missing keys listed twice leave 1 000 answers, not 2 000; eviction drops the oldest and deletes its dedupe identity (XPORT-07) | ✓ VERIFIED | `maxEgressEntries = 32768` (`core.go:151`); identity is `(handle, kind, key)`; `enqueueResponse:2233` skips duplicates and `dropOldestEgress:2253` deletes the identity in lockstep with a counted drop. `TestEgress_boundedDeduped` (exactly 1 000 after two 1 000-key listings), `TestEgress_identitySeparates`, `TestEgress_capDropsOldest`, `TestEgress_drainSkipsGoneRecords`, `TestEgress_identityOnly`, `TestEgress_noValueRetention` PASS; the drain re-reads a DataReplication answer via `Lookup`. |
| 16 | Dead code is removed: `lastLocal`/`cover` and the above-cover branch, exported `persist.Parse`, `ownHandle`, `fsync_darwin.go`/`fsync_other.go` are gone; the darwin cross build and vet pass (XPORT-13) | ✓ VERIFIED | Repo-wide grep over `cmd/ internal/ api/ scripts/ Makefile .github/ docs/` finds no `lastLocal`, `ownHandle` or `persist.Parse` in source; the only `func Parse(` in the tree is `internal/config.Parse`. `internal/persistfile/fsync_darwin.go`/`fsync_other.go` are absent; `fsops.go` calls `o.tmp.Sync()`/`d.Sync()` (`:125,181`). `go build ./... && go vet ./...` clean and `GOOS=darwin GOARCH=arm64 go build ./... && go vet ./...` OK at this tree. S-PERS-03 documents the stdlib `File.Sync` path. |
| 17 | The red-gated expectation table and mutation stage work: the three transport rows pass under `SUPKV_RED_TESTS=1`, the other rows reflect their own phase, and the mutation stage kills every listed seed while the control passes (TEST-04, D-01/D-02) | ✓ VERIFIED | The table (`scripts/red-expectations.tsv`) and runner are unchanged since the phase-4/5 flips (2026-11-22): seven rows, all `green`. First-hand at this tree: R-02 drain and R-02 burst rows PASS (`SUPKV_RED_TESTS=1 go test`, truths 5-6), R-03 ×2 and P-01 ×2 rows PASS in today's local red-runner evidence; the R-01 row fails locally only at the throttled-host local scale (evidence-scope note 1 — adjudicated; the row's green state is the full-speed-host flip evidence `20261121T225048Z`, committed, and the scheduled `red-r01-x20` job). The mutation stage — whose core.go-targeted sed was the stale-risk this regeneration had to check — was reproduced standalone at this tree: control passes 4/4 corpus seeds; `sync-without-tombstones` (store.go overlay) and `periodic-sync-disabled` (core.go overlay, post-Phase-8 file) are each a non-no-op edit that compiles and is killed by seed 848612892191557850. Previous full red-runner passes: `20261123T201708Z` (7/7, both mutants killed) and the 2026-11-24 count-20 sessions. |

**Score:** 17/17 truths verified (0 present, behavior-unverified).

### Advisory (New Scope, Unevidenced)

None — no new-scope Step 7 finding lacked deterministic evidence. The Phase-8 changes inside the covered set (pending index, cost-gate rewrite, port allocator) were re-verified directly rather than flagged: the index maintenance contract is documented and tested, the cost gates are strictly stronger than the ones they replaced, and the allocator adoption is green under the count-3 package check.

### Required Artifacts

34 artifact declarations across the 13 plans. `gsd-tools query verify.artifacts` reports 31/33 pass, with the only two failures being `internal/udprepl/repl.go` declared by plans 03-01 and 03-06 — the known relocation: Phases 4/7 split that file into named files whose declared content survives and is wired (verified by reading and by the package test suite). Not a missing artifact.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/udprepl/repl.go` | Buffer seam, read loop, tracker, policy | ✓ RELOCATED | Split into `adapter.go` (`New`, `SocketBuffers`, trackers), `loop.go` (`Run`, `NotifyLocalWrite`, `WaitReplicated`, `readLoop`), `socket.go` (`configureSocketBuffers`, `checkSendBuffer`, `listenDatagram`), `codec.go`, `mirror.go`; every declared behavior present and wired. |
| `internal/udprepl/socket_internal_test.go` | Buffer read-back and floor proof | ✓ EXISTS + SUBSTANTIVE | 5 tests over a real loopback socket; 65 506 / 65 507 boundary |
| `internal/stand/stand_buffers_test.go` | 1/20/60 KiB delivery | ✓ EXISTS + SUBSTANTIVE | Two real binaries; byte-exact values; startup lines parsed; PASS 0.51 s |
| `cmd/sup-kv/main.go` | Wiring, flags, startup checks | ✓ EXISTS + SUBSTANTIVE | Buffer log `:679-682`, bind WARN `:328`, clock lead `:708`, exit-2 via `config.Validate` |
| `internal/app/storage/store.go` | `Version`/`Lookup`/`Versions`, `MaxKeyBytes:33` | ✓ EXISTS + SUBSTANTIVE + WIRED | Clone/ownership contracts documented and tested |
| `internal/app/storage/view_test.go` | View contract proofs | ✓ EXISTS + SUBSTANTIVE | Tombstone visibility, clone ownership, snapshot, concurrent hammer — PASS |
| `internal/app/persist/engine.go` | Engine forwarding, dead-code removal | ✓ EXISTS + SUBSTANTIVE + WIRED | Forwards the view reads through `facade.go` |
| `internal/app/persist/facade.go` | Store-facade forwards | ✓ EXISTS + SUBSTANTIVE + WIRED | `Lookup`/`Version`/`Versions`/`Observe`/`Generation` |
| `internal/app/internal/simnet/simnet.go` | `nodeStore` six-method seam | ✓ EXISTS + SUBSTANTIVE + WIRED | Write/Delete read back via `Lookup`, panic on refusal |
| `internal/app/version/version.go` | `Validate` predicate | ✓ EXISTS + SUBSTANTIVE + WIRED | Pure/total; sentinels; used by the validation pipeline |
| `internal/app/version/validate_test.go` | Boundary + property proofs | ✓ EXISTS + SUBSTANTIVE | `TestValidate`, `TestValidate_property` PASS |
| `internal/app/repl/core.go` | Pacers, ledger, heap, results, validation | ✓ EXISTS + SUBSTANTIVE | All Phase-3 mechanisms present; Phase-8 `pendingByKey` index documented and maintained at exactly two sites |
| `internal/app/repl/bench_test.go` | Both cost gates | ✓ EXISTS + SUBSTANTIVE | Deterministic count gates (Phase 8): zero-inspection unacked gate, exact seam-call key-count gate, write/apply key-local gates |
| `internal/port/repl.go` | Result stream | ✓ EXISTS + SUBSTANTIVE + WIRED | `Drain`/`DrainResults` independent; four result kinds |
| `internal/grpcapi/wait.go` | Thin honest wait | ✓ EXISTS + SUBSTANTIVE + WIRED | No poll, no diffing, no retention window |
| `internal/app/repl/egress_internal_test.go` | Bounded/deduped queue | ✓ EXISTS + SUBSTANTIVE | 1 000-not-2 000, cap drop, identity separation, no value retention — PASS |
| `internal/app/repl/discard_test.go` | Classes and before-clock proof | ✓ EXISTS + SUBSTANTIVE | Counting-clock and rate-limit tests PASS |
| `internal/udprepl/decode_negative_test.go` | Negative table | ✓ EXISTS + SUBSTANTIVE | Includes flipped bad_checksum/bad_version rows — PASS |
| `internal/udprepl/maxdatagram.go` | Datagram ceiling oracle | ✓ EXISTS + SUBSTANTIVE + WIRED | Real-schema marshal; consumed by `config.checkDatagramCeiling` |
| `internal/udprepl/maxdatagram_test.go` | Oracle and accepted-domain property | ✓ EXISTS + SUBSTANTIVE | Boundary rows and domain walk through the real codec — PASS |
| `docs/SPEC-DEVIATIONS.md` | S-DM-03 / S-PERS-03 closures | ✓ EXISTS + SUBSTANTIVE | Rows updated with code links; S-PROTO-01 CRC scope; S-FAIL-15 burst row links the bounds |
| `internal/udprepl/readloop_internal_test.go` | Read-loop error contract | ✓ EXISTS + SUBSTANTIVE | Fake conn survives non-closed errors; close exits — PASS |
| `internal/app/repl/warn_test.go` | No-Hello WARN | ✓ EXISTS + SUBSTANTIVE | One-shot latch — PASS |
| `internal/persistfile/fsops.go` | `os.File.Sync` durability | ✓ EXISTS + SUBSTANTIVE | Standard-library F_FULLFSYNC path (`:125,181`) |
| `scripts/red-runner` | Per-mutant source + clean-tree check | ✓ EXISTS + SUBSTANTIVE | Overlay-only, compile check, no-op rejection, timeout rejection, clean-tree before/after; mutation stage reproduced green at this tree |
| `scripts/red-expectations.tsv` | Seven expectation rows | ✓ EXISTS + SUBSTANTIVE | All rows `green`; transport rows are the Phase-3 flips; R-03/P-01 flipped by Phases 4/5 |
| `internal/metrics/metrics.go` | Pacing families and discard classes | ✓ EXISTS + SUBSTANTIVE + WIRED | Four emission families + twelve discard labels wired from the mirror |
| `.github/workflows/scheduled.yml` | `red-r01-x20` job | ✓ EXISTS + SUBSTANTIVE | `-count=20`, 110m/120m timeouts, both env gates, no tolerance flags (`:149-183`) |
| `Makefile` | `race-soak` includes `internal/stand` | ✓ EXISTS + SUBSTANTIVE | `race-soak:98-99` lists all seven packages incl. `./internal/stand/` |

**Artifacts:** 28/28 unique plan-declared paths exist and are substantive; 1 (the adapter package entry file) relocated intact into five named files.

### Key Link Verification

`gsd-tools query verify.key-links` reports 3/37 verified across the 13 plans — the verb's heuristic looks for the literal target path inside the source file, while every Phase-3 link is expressed in Go as an import/package-qualified call (the same false-negative pattern the 2026-11-22 and 2026-11-23 reports recorded). All 37 links were verified manually by reading the call sites at this tree; **37/37 wired**.

| From | To | Via | Status | Current evidence |
|------|----|----|--------|------------------|
| `cmd/sup-kv/main.go` | `internal/udprepl/adapter.go` | `wire()` logs `adapter.SocketBuffers()` | ✓ WIRED | `main.go:622,679-682` |
| `internal/udprepl/adapter.go` | `internal/udprepl/socket.go` | `New` → `configureSocketBuffers` / send floor | ✓ WIRED | `adapter.go`; `socket.go:136,202` |
| `internal/stand/stand_buffers_test.go` | `internal/stand/stand.go` | Harness drives real binaries | ✓ WIRED | Test PASS 0.51 s |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | Listing/DataRequest/DataSync consume `Version`/`Versions`/`Lookup` | ✓ WIRED | `core.go:1647,1676,2534+`; `StateStore:402-435` |
| `internal/app/persist/facade.go` | `internal/app/storage/store.go` | `Lookup`/`Version`/`Versions` delegate | ✓ WIRED | `facade.go` |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `Cluster.Write/Delete` read via `Lookup` | ✓ WIRED | `simnet.go` nodeStore |
| `internal/app/repl/core.go` | `internal/app/version/version.go` | `Validate` before any clock touch | ✓ WIRED | `core.go:1182` between classification and `handleMessage:1180` |
| `internal/app/version/validate_test.go` | `internal/app/version/version.go` | `PackVersion`/layout constants as oracle | ✓ WIRED | Tests PASS |
| `internal/app/repl/bench_test.go` | `internal/app/repl/core.go` | Cost gates drive `HandleDatagram`/`Advance`/`NotifyLocalWrite`/`ApplyRemote` | ✓ WIRED | Deterministic gates PASS |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | Retransmit resolves via `Lookup`, skips version mismatch | ✓ WIRED | `core.go:3060-3075` |
| `internal/udprepl/mirror.go` | `internal/app/repl/core.go` | Mirror uses `PeerCount`/O(1) `UnackedStats`/`Discards` | ✓ WIRED | `mirror.go:14-60` |
| `internal/app/repl/core.go` | `internal/port/repl.go` | Result events appended to the `Outbox` | ✓ WIRED | Resolution sites + `port/repl.go` |
| `internal/udprepl/adapter.go` | `internal/grpcapi/wait.go` | `WaitReplicated` injected as `ServerOptions.WaitReplicated` | ✓ WIRED | `main.go:637`; `loop.go:234-252,409+` |
| `internal/grpcapi/server.go` | `internal/udprepl/loop.go` | `track` flag → adapter tracker before enqueue | ✓ WIRED | `server.go:203,270`; `loop.go:234` |
| `internal/app/repl/core.go` | `internal/app/repl/core.go` | Shared normalized `syncResponseInterval`, min deadlines | ✓ WIRED | `core.go:912-919,1036,2288,2756,3045` |
| `internal/app/repl/sync_test.go` | `internal/stand/stand_red_test.go` | Unit pacing asserts the same arithmetic | ✓ WIRED | Same `floor(1s/rate)` expression; both PASS |
| `internal/app/repl/core.go` | `internal/app/version/version.go` | `Validate(now, MaxClockLead)` for records and entries | ✓ WIRED | `core.go:1201,1225` |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | `RecordChecksum` recomputed; `MaxKeyBytes` key bound | ✓ WIRED | `core.go:1201+` |
| `internal/udprepl/mirror.go` | `internal/metrics/metrics.go` | Twelve counters mirrored into `supkv_datagrams_discarded_total` | ✓ WIRED | `mirror.go:14-33`; `metrics.go:426` |
| `cmd/sup-kv/main.go` | `internal/app/repl/core.go` | `--max-clock-lead` → `repl.Config.MaxClockLead` | ✓ WIRED | `main.go:708` |
| `internal/grpcapi/server.go` | `internal/app/storage/store.go` | API compares against `storage.MaxKeyBytes` | ✓ WIRED | `server.go:203,270` |
| `internal/config/validate.go` | `internal/udprepl/maxdatagram.go` | Ceiling through the real codec → exit 2 | ✓ WIRED | `validate.go:56-57`; `main.go:64,172,184` |
| `internal/udprepl/maxdatagram_test.go` | `internal/udprepl/codec.go` | Encodes via the real codec helper | ✓ WIRED | `EncodeDatagramForTest` — PASS |
| `internal/udprepl/loop.go` | `net` | `errors.Is(err, net.ErrClosed)` only exit | ✓ WIRED | `loop.go:480` |
| `cmd/sup-kv/main.go` | `internal/udprepl/socket.go` | Bind WARN from the bound address; no reuse option | ✓ WIRED | `main.go:195,328`; `socket.go:83-124` |
| `internal/app/repl/core.go` | `internal/app/repl/core.go` | `Advance` → `warnIfNoPeer` one-shot | ✓ WIRED | `core.go:2496` |
| `internal/app/repl/core.go` | all `repl.New` call sites | Four-argument constructor | ✓ WIRED | `go build ./...` clean |
| `internal/persistfile/fsops.go` | `os.File.Sync` | `syncDurable` calls `Sync()` | ✓ WIRED | `fsops.go:125,181` |
| `internal/metrics/metrics.go` | `internal/udprepl/mirror.go` | Mirror accessors feed the families | ✓ WIRED | `metrics.go:183,223` |
| `cmd/sup-kv/main.go` | `internal/metrics/metrics.go` | metrics sources wired | ✓ WIRED | `main.go` wire() |
| `scripts/red-runner` | `internal/app/storage/store.go` | Tombstone mutant overlay on the store source | ✓ WIRED | Reproduced: non-no-op, compiles, killed |
| `scripts/red-runner` | `internal/app/repl/core.go` | Periodic-sync mutant overlay on the core source | ✓ WIRED | Reproduced against the Phase-8 file: non-no-op, compiles, killed |
| `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go` | Three transport rows green | ✓ WIRED | R-02 rows PASS first-hand; R-01 row adjudicated (note 1) |
| `.github/workflows/scheduled.yml` | `scripts/red-expectations.tsv` | 20× job asserts the green R-01 behavior | ✓ WIRED | `scheduled.yml:149-183` |
| `Makefile` | `internal/stand` | `race-soak` includes the package | ✓ WIRED | `Makefile:98-99` |
| `internal/udprepl/socket_internal_test.go` | `internal/udprepl/socket.go` | Real loopback getsockopt read-back proof | ✓ WIRED | `TestSocketBuffers` PASS |
| `cmd/sup-kv/main_test.go` / `internal/stand/stand.go` / `internal/udprepl/smoke_test.go` / `internal/grpcapi/wait_integration_test.go` | `internal/testutil/ports.go` | Shared claim-while-held port allocator replaces pick-then-bind | ✓ WIRED | `ports.go:105-238`; adopters updated in `5c7a3d7`/`3e7b465`/`7aeec35`; count-3 package run green |

**Wiring:** 37/37 key links verified.

### Data-Flow Trace (Level 4)

The phase's dynamic values are metrics and wire sizes, not rendered UI. Every published value traces to live core/adapter state; no static return, canned fixture, or mock is on a production path.

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `internal/metrics/metrics.go` `supkv_replication_unacked_records` / `_peers` / `_oldest_age_seconds` | unacked count, oldest age, peer count | `Adapter` post-event atomics (`mirror.go`) ← `Core.UnackedStats`/`PeerCount` (O(1)) | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_replication_datagrams_retransmitted_total` | retransmit count | `Core.retransmitted` incremented at emission (`core.go:3075`), mirrored by the adapter | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_datagrams_discarded_total` (12 classes) | class counters | `Core.discards` incremented at each discard site, mirrored via `Adapter.Discards()` | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_replication_datagrams_paced_total` / `_send_queue` / `_egress_dropped_total` / `_read_errors_total` | pacer count, queue depth, drops, read errors | `Core.PacedSends`, `QueuedSends()`, egress drop counter, `Adapter.readErrors` | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_replication_datagrams_sent_total` | sent by type | `Adapter.SentByType` incremented only after the socket accepted the write | Yes | ✓ FLOWING |
| `internal/grpcapi/wait.go` report | `acked`, `ackedCount`, `total` | `Adapter.WaitReplicated` tracker fed by `Outbox.DrainResults`; count clamped to write-time total | Yes | ✓ FLOWING |
| `internal/udprepl/maxdatagram.go` | datagram size | Real `proto.Marshal` of the generated schema at worst-case widths | Yes | ✓ FLOWING |
| `cmd/sup-kv/main.go` startup buffer line | effective recv/send sizes | `getsockopt` read-back from the bound socket (`socket.go:136-176`) | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

All machine-verifiable bounds re-run first-hand at tree `ef370a0` (2026-11-27). The full workspace suite was not run; named tests per package.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Pacing / rotation / one-in-flight / triggers | `go test -race -count=1 ./internal/app/repl/ -run '^(TestSync_pacing\|TestSync_rotation\|TestSync_listingBusy\|TestSync_triggers\|…egress/discard/cost/jitter/listing named tests…)$'` (21 tests) | `ok` 2.791 s | ✓ PASS |
| Version boundary + property, store view contract, config rule table | `go test -race -count=1 ./internal/app/version/ ./internal/app/storage/ ./internal/config/` | `ok` (1.015 s / 1.460 s / 1.034 s) | ✓ PASS |
| Buffer seam, floor, read loop, no-reuse bind, wait semantics, ceiling property, negative table, tracker-before-core | `go test -race -count=1 ./internal/udprepl/ -run '^(TestSocketBuffers\|TestCheckSendBuffer\|TestReadLoop_\|TestListen_noAddressReuse\|TestMaxRecordDatagramBytes_acceptedDomain\|TestDecode_negativeTable\|TestNotify_createsTrackerBeforeCore\|TestWaitReplicated_\|TestSmoke_twoNodesUDP\|TestMirror_unackedSent…)$'` (16 tests) | `ok` 1.340 s | ✓ PASS |
| Binary boundary: buffer log, datagram ceiling exit 2, bind WARN, duplicate bind, clock lead, precedence | `go test -race -count=1 ./cmd/sup-kv/ -run '^(TestMain_socketBuffersLogged\|TestMain_valueMaxDatagram\|TestMain_bindAddrWarn\|TestMain_duplicateBind\|TestMain_maxClockLead\|TestMain_precedence)$'` | `ok` 3.303 s | ✓ PASS |
| 1/20/60 KiB delivery on real sockets | `go test -race -count=1 ./internal/stand/ -run '^(TestStand_bufferSizesDeliver\|TestStand_oversizeKeyF2c)$'` | PASS: delivery 0.51 s (bound 3.4 s), F2c 3.90 s | ✓ PASS |
| R-02 drain (10 000 unacked, returning peer) | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_R02_unackedDrain$'` | `--- PASS (41.03s)`; precondition `10000/10000 unacked` | ✓ PASS |
| R-02 burst (1000/s × 15 s) | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_R02_burst$'` | `--- PASS (15.91s)` | ✓ PASS |
| Red-gate row stage, this host | two local `scripts/red-runner` runs today (untracked evidence `20261127T161311Z`, `20261127T161629Z`) | 6/7 rows `pass,run`; R-01 row `fail,run` at the local 8 000-key scale (42.64 s) — the adjudicated throttled-host class | ✓ PASS (with note 1) |
| Mutation stage (control + both mutants) | standalone reproduction of `mutant_probe`/`control_probe` at this tree | control 4/4 corpus seeds PASS; both mutants non-no-op, compile, killed by seed 848612892191557850 | ✓ PASS |
| TEST-11 flake class at the current tree | `go test -race -count=3 ./cmd/sup-kv/ ./internal/udprepl/` | `ok` 52.396 s + `ok` 6.289 s | ✓ PASS |
| Build, vet | `go build ./... && go vet ./...` | clean | ✓ PASS |
| Darwin availability after deletions | `GOOS=darwin GOARCH=arm64 go build ./... && go vet ./...` | OK | ✓ PASS |
| Documented-scale R-01 (20 000 keys × 1 KiB) | not re-run locally — wall-clock bound, powersave host (evidence-scope note 1) | 20/20 acceptance = scheduled `red-r01-x20` CI job; flip cleared by full-speed evidence `4a83919`/`20261121T225048Z` | ? SKIP (adjudicated) |

### Probe Execution

Not applicable — Phase 3 declares no runnable `scripts/*/tests/probe-*.sh` probes (the only `scripts/` executables are `red-runner` and its expectation table, both exercised above; `find scripts -path '*/tests/probe-*.sh'` is empty). The plans' "Probe XPORT-xx" labels are design probes pinned by named tests, verified in the truth table.

### Requirements Coverage

All 13 requirement IDs are declared across the plan frontmatter (`03-01`…`03-13` collectively name XPORT-01…13), and REQUIREMENTS.md's Transport section maps exactly XPORT-01…13 with no orphaned requirement.

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| XPORT-01 | 03-07, 03-12, 03-13 | Sync listing paced and robust to buffer loss (R-01) | ✓ SATISFIED | Truths 1-3; documented-scale R-01 bound derivation unchanged; pacing tests green; 20× job wired |
| XPORT-02 | 03-05, 03-07, 03-12, 03-13 | Event-loop cost flat; retransmits paced with jitter (R-02) | ✓ SATISFIED | Truths 4-6; deterministic zero-inspection gate (stronger than the old 2× ratio), record-free ledger, O(1) gauges, drain and burst PASS first-hand |
| XPORT-03 | 03-08, 03-09, 03-12 | Key length and `value_max_bytes` validated (R-06) | ✓ SATISFIED | Truth 7; single `MaxKeyBytes`, exit 2, ≤65 507 property, F2c |
| XPORT-04 | 03-03, 03-08, 03-12 | Received versions validated before the clock (R-07) | ✓ SATISFIED | Truth 8; counting-clock proof, configurable lead |
| XPORT-05 | 03-01, 03-13 | Socket buffers configured and reported (R-08) | ✓ SATISFIED | Truths 3, 11; read-back logging, send floor, 1/20/60 KiB first-hand, darwin ADR |
| XPORT-06 | 03-02, 03-04, 03-12 | `DataRequest`/sync chunks independent of foreign records (R-09) | ✓ SATISFIED | Truth 13; six-method `StateStore`, deterministic exact-count flatness gate |
| XPORT-07 | 03-07, 03-12 | Sync-answer queue bounded and deduplicated (R-11) | ✓ SATISFIED | Truth 15; 1 000-not-2 000 proof, lockstep eviction |
| XPORT-08 | 03-10, 03-13 | Read loop does not silently die (R-14) | ✓ SATISFIED | Truth 10; fake-conn tests |
| XPORT-09 | 03-07, 03-10, 03-13 | Bind misuse loud, duplicate instances fail, timers jitter (R-15) | ✓ SATISFIED | Truth 12; WARN, EADDRINUSE, no-Hello WARN, jitter envelopes |
| XPORT-10 | 03-08, 03-12 | Invalid datagram combinations rejected and counted (R-16) | ✓ SATISFIED | Truth 9; classes + counters + docs |
| XPORT-11 | 03-02, 03-04, 03-12 | Key→version view and tombstone-visible point reads (Q-01) | ✓ SATISFIED | Truth 13; `Version`/`Lookup`/`Versions` with clone/fresh-map tests |
| XPORT-12 | 03-06, 03-12 | Ack-by-peer-X as an `Outbox` result to adapter and waiters (Q-02) | ✓ SATISFIED | Truth 14; result stream, tracker, poll removal |
| XPORT-13 | 03-11, 03-12 | Dead code removed (Q-09) | ✓ SATISFIED | Truth 16; grep + darwin cross build/vet |

**Coverage:** 13/13 requirements satisfied; 0 blocked; 0 orphaned.

### Decision Coverage

`gsd-tools query check.decision-coverage-verify` → `{skipped: false, blocking: false, total: 8, honored: 8, not_honored: [], message: "All trackable CONTEXT.md decisions are honored by shipped artifacts."}` — 8/8 honored (non-blocking gate).

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/stand/stand_red_test.go` | XPORT-01/02 | 3 transport rows (+2 R-03/P-01 rows from later phases) | 0 with `SUPKV_RED_TESTS=1` (the env gate is the D-01 mechanism; the runner opens it) | No — asserts real-binary convergence/metrics | Behavioral (bounds + metrics + process lifecycle) | ✓ |
| `internal/stand/stand_buffers_test.go` | XPORT-05 | 1 | 0 | No | Behavioral (byte-exact delivery on real sockets) | ✓ |
| `internal/app/repl/bench_test.go` | XPORT-02/06 | 4 gates | 0 | No — counts consumer-seam calls, an independent measurement of the two fixture sizes | Value/behavioral (exact counts, hard 0-inspection bound) — strengthened by Phase 8 from wall-clock ratios | ✓ |
| `internal/app/repl/discard_test.go` | XPORT-03/04/10 | 10 | 0 | No | Behavioral (zero Tick/Update, per-class counters) | ✓ |
| `internal/app/repl/egress_internal_test.go` | XPORT-07 | 6 | 0 | No | Value (queue depth 1 000 vs 2 000; identity-only) | ✓ |
| `internal/udprepl/socket_internal_test.go` | XPORT-05 | 5 | 0 | No | Value (getsockopt equality, floor boundary) | ✓ |
| `internal/udprepl/readloop_internal_test.go` | XPORT-08 | 2 | 0 | No | Behavioral (survive/exit branches) | ✓ |
| `internal/udprepl/maxdatagram_test.go` | XPORT-03 | 3 | 0 | No — oracle compared against real encoding | Value + domain walk | ✓ |
| `internal/app/version/validate_test.go` | XPORT-04 | 2 | 0 | No | Value (boundary ±1 ms) + totality property | ✓ |
| `internal/grpcapi/wait_test.go`, `wait_integration_test.go`, `internal/udprepl/repl_test.go` | XPORT-12 | 15 | 0 (one capability-probe skip is loud, only if loopback broadcast is unavailable) | No | Behavioral (full-coverage success, honest false paths) | ✓ |
| `cmd/sup-kv/main_test.go` | XPORT-03/05/09 | 6 phase-03 tests | 0 | No | Behavioral (process exit codes, parsed log lines) | ✓ — now ported to the shared claim-while-held allocator (`testutil.UDPPort`); green at count=3 |

**Disabled tests on requirements:** 0 effective — the only requirement-test skips are the intentional `SUPKV_RED_TESTS` gate, which the red-runner opens (rows report `pass,run`, not `skip`), and the loud loopback-broadcast capability probe.
**Circular patterns detected:** 0 (the mutant overlays write derived copies, not expected values; the counting double is applied at the seam, not derived from output).
**Insufficient assertions:** 0 — every requirement-linked test asserts values or multi-step behavior, not mere existence.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | `TBD`/`FIXME`/`XXX` in any covered non-planning file (179 tracked files scanned) | none | Debt-marker gate: zero matches |
| — | — | `TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented" in production files | none | Only benign comment hits: "addresses are placeholders" in `cmd/sup-kv/main_test.go` (describing placeholder bind addresses, not unfinished work) and `internal/testutil/perf.go:34` `perfUnknown` (a named sentinel constant for an unknown hardware field) |
| — | — | Stub returns, empty handlers, static metric values | none | Every metric/discard/wait value traces to live state (Level 4 above) |

**Anti-patterns:** 0 found (0 blockers, 0 warnings, 0 advisories).

### Human Verification Required

N/A — infrastructure/foundation phase with no user-facing elements. Every acceptance criterion is programmatic (real-socket bounds, process exit codes, metrics, buffer read-backs, codec sizes, cross builds). `behavior_unverified: 0`, so nothing routes to UAT.

## Gaps Summary

None. All 17 must-haves are verified against the current tree `ef370a0`; the phase goal is achieved in the codebase:

- **Paced and bounded sending is real:** listings, records and reactive egress share one normalized pacer (`floor(1s/sync_response_rate)`), one in-flight listing per destination, rotating start chunks, and a deduplicated record queue whose ladder re-arms from actual emission — all machine-verified at this tree, with drain and burst passing on real sockets first-hand.
- **Buffers are configured and reported honestly:** the kernel's doubled/clamped values are read back with `getsockopt`, logged at startup, and the send floor fails startup at 65 506 bytes while accepting 65 507; 1/20/60 KiB deliver byte-exact.
- **Input is validated before the clock:** the validation pipeline sits between classification and the only store/clock path, proven with a counting clock (zero `Tick`/`Update` for every discard class), with per-class counters and rate-limited WARNs.
- **The recovery and burst scenarios converge within their documented limits:** R-02 drain PASS (41.03 s) and R-02 burst PASS (15.91 s) first-hand on this host; the documented-scale R-01 bound derivation is unchanged and its 20/20 acceptance is the scheduled `red-r01-x20` CI job per the phase's recorded decisions.
- **The seams later phases build on exist:** the tombstone-visible key→version view, the `Outbox` result stream with a tracker-backed honest wait and no polling, the bounded deduped sync-answer queue, and the dead-code removals.

**Stale-regeneration result:** Phase 8's transport-adjacent changes do not regress any Phase-3 must-have. The per-key `pendingByKey` index carries a documented two-site maintenance contract and its cost gates are now deterministic and strictly stronger (zero pending-entry inspection at 10 and 10 000 unacked; exact seam-call counts at 1 000 and 50 000 keys); the retransmit counter still increments only at actual emission; the shared claim-while-held port allocator replaces every pick-then-bind picker the phase-3 harnesses used, and the TEST-11 class holds (`go test -race -count=3 ./cmd/sup-kv/ ./internal/udprepl/` green at this tree); the red-runner's core.go-targeted mutation sed still matches the Phase-8 file and both mutants are killed.

Evidence-scope notes (recorded, not gaps):
1. **R-01 wall-clock class (adjudicated):** this host's CPU governor is `powersave` (~1.6 GHz). Today's two local red-runner runs (untracked evidence `20261127T161311Z`, `20261127T161629Z`) fail only `TestRed_R01_pacedSync20k` at the local 8 000-key scale — the same artifact the 03-12 halt record measured (7 453–7 734/8 000 at the ~20.6 s bound; ~360-375 replies/s vs 500/s on this host class). The phase's recorded decisions govern: 03-12 halted rather than weaken any bound; the flip was cleared by full-speed-host evidence (commit `4a83919`, `red-evidence/20261121T225048Z`); 03-13 moved the 20/20 acceptance to the scheduled `red-r01-x20` job (`-count=20` at the documented scale, no tolerance flags), whose first scheduled execution remains a CI observation. Every machine-verifiable component of the R-01 truth (pacing semantics, rotation, one-in-flight, buffer read-back, cheap view, R-02 rows) passes at this tree first-hand.
2. **Phase-8 gate evolution, not a regression:** `bench_test.go`'s two wall-clock ratio gates were replaced by deterministic operation-count gates (08-05). The new gates assert strictly more (zero pending-entry inspection on idle/write/apply paths at any set size; exact store-call counts at 1 000 and 50 000 keys) and are immune to host load; the ns/op benchmarks remain measurement-only. The XPORT-02(1)/XPORT-06 acceptances are therefore enforced more strongly than the roadmap's literal "within 2×" wording.
3. **Artifact relocation (unchanged from prior rounds):** `internal/udprepl/repl.go` was split by Phases 4/7 into `adapter.go`/`loop.go`/`socket.go`/`codec.go`/`mirror.go`; the artifact verb's two failures are this known false-negative, and the declared content is present and wired.
4. **Key-links verb false-negative:** the verb reports 3/37 because it greps for the literal target path inside the source file; Phase-3 links are Go import/package-qualified calls. All 37 were verified by reading the call sites.
5. **Working tree not git-clean at verification time:** untracked red-evidence directories (including today's two runs) and modified `.planning` reports for Phases 1/2 written by other agents. No production source or test file is modified in the working tree.
6. **Fingerprint scope:** 95 covered inputs — the 69 implementation/test/config/doc files listed in frontmatter (the previous 65 plus the Phase-8 transport-adjacent `internal/testutil/ports.go`, `ports_test.go`, `internal/stand/stand.go`, `internal/app/repl/config_internal_test.go`) plus the 26 phase-03 PLAN/SUMMARY files added by `verification.fingerprint`; the report itself is excluded.

---

_Verified: 2026-11-27T16:45:15Z_
_Re-verified: stale-fingerprint regeneration after Phase 8 perf work — all machine-verifiable bounds re-run green at tree `ef370a0`_
_Verifier: ZCode (gsd-verifier)_
