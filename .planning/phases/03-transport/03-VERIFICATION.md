---
phase: 03-transport
verified: 2026-11-22T00:01:44Z
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
  - internal/app/internal/simnet/simnet.go
  - internal/app/internal/simnet/simnet_ext_test.go
  - internal/app/internal/testutil/gen.go
  - internal/app/persist/codec.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/bench_test.go
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
  - internal/stand/stand_buffers_test.go
  - internal/stand/stand_red_test.go
  - internal/stand/stand_test.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/maxdatagram.go
  - internal/udprepl/maxdatagram_test.go
  - internal/udprepl/readloop_internal_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/scenario_test.go
  - internal/udprepl/smoke_test.go
  - internal/udprepl/socket_internal_test.go
  - internal/udprepl/syncsize_internal_test.go
  - scripts/red-expectations.tsv
  - scripts/red-runner
covered_digest: "v4:sha256:0f37070d43bd98cba437b925576823da9219088564da499898459268c62f069d"
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "R-03 red rows (TestRed_R03_gateClosedUntilFullList, TestRed_R03_f14ReplacedNode) still fail: the interface is not yet closed until the full key list arrives"
    addressed_in: "Phase 4"
    evidence: "ROADMAP Phase 4 SC 1: 'The interface is closed until the full key list arrives (R-03 red test turns green)' — PROTO-01"
  - truth: "P-01 red rows (TestRed_P01_valueLimit, TestRed_P01_valueLimitEngine) still fail: the value limit is not yet defined once and enforced"
    addressed_in: "Phase 5"
    evidence: "ROADMAP Phase 5 SC 1: 'The value limit is defined once and enforced everywhere (P-01 red test turns green)' — PERSIST-01"
---

# Phase 3: Transport Verification Report

**Phase Goal:** The UDP transport meets the review's stated bounds on real sockets — sending is paced and bounded, buffers are configured and reported, input is validated before it can touch the clock — so the recovery and burst scenarios converge within their documented limits.
**Verified:** 2026-11-22T00:01:44Z (tree `65a29dc`)
**Status:** passed
**Re-verification:** No — initial verification (no previous `03-VERIFICATION.md` exists)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | DataSync listings are enqueued and drained at one datagram per `floor(1s/sync_response_rate)` with one in-flight round per destination, and the sending start chunk rotates across rounds (R-01) | ✓ VERIFIED | `Core.syncListing`/`enqueueListingChunk`/`drainListings`/`jitteredSpacing` in `internal/app/repl/core.go`; `TestSync_pacing` (spacing inside [interval−10%, interval+10%], one datagram per Advance, first chunk immediate), `TestSync_rotation` (round 0 → sorted order, round 1 → one-chunk rotation, same key multiset), `TestSync_listingBusy` (trigger dropped while in flight, `nextSync` still advances). All green under `-race`. |
| 2 | Two nodes on real Linux sockets with 20 000 keys × 1 KiB into an empty receiver converge within 2 × `sync_interval` + N / `sync_response_rate`; the R-01 red row is green; a 20× documented-scale CI job carries the 20/20 acceptance (R-01) | ✓ VERIFIED | `TestRed_R01_pacedSync20k` is the full acceptance assertion derived from the nodes' flags. `bash scripts/red-runner` (2026-11-22, evidence `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T235409Z/summary.txt`): row `TestRed_R01_pacedSync20k  green  pass,run  OK`. Independent documented-scale run at this tree: `SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'` → `--- PASS (108.86s)`. The 20/20 repetition is the scheduled `red-r01-x20` job (`.github/workflows/scheduled.yml:149-183`): `-count=20`, `-timeout 110m`, `timeout-minutes: 120`, `SUPKV_STAND_NIGHTLY=1`, `SUPKV_RED_TESTS=1`, no `|| true` / `continue-on-error` / skip flags. Its first scheduled execution is a CI observation (plan 03-13's recorded acceptance vehicle), not a code gap. |
| 3 | `SO_RCVBUF`/`SO_SNDBUF` are set with the actual kernel values read back and logged at startup; the core consumes a cheap key→version view instead of a store copy (R-01) | ✓ VERIFIED | `configureSocketBuffers` sets then `getsockopt`-reads both sizes (`internal/udprepl/repl.go:457-498`); `New` stores the read-back (`:370-393`); `wire()` logs `slog.Info("Socket buffers", "recv_buf", recvBuf, "send_buf", sendBuf)` (`cmd/sup-kv/main.go:522-523`). `TestSocketBuffers` asserts `SocketBuffers()` equals the getsockopt values, not the requested constants; `TestMain_socketBuffersLogged` green. `repl.StateStore` is exactly `ApplyRemote/Version/Lookup/Versions` — no `Records` (`internal/app/repl/core.go:249-268`), so the O(store) snapshot path is compile-impossible; listing built from `Versions()` (`:1513-1520`). |
| 4 | Event-loop iteration cost at 10 and 10 000 unacked records differs by at most 2× (R-02(1)) | ✓ VERIFIED | `TestCore_unackedCostFlat` (wall-clock benchmark ratio, hard bound 2.0): measured `unacked=10 = 1218.0 ns/op, unacked=10000 = 1277.0 ns/op, ratio 1.05` → PASS. `pending` is keyed by `(key, version)` with no record Data; `retryHeap` is the only due index; `UnackedStats`/`PeerCount` are O(1) (`core.go:1942-1951, 1931-1933`). |
| 5 | A peer returning to a node holding 10 000 unacked 1 KiB records drains within N / `sync_response_rate` + 10 s with at most 3N retransmits (R-02(2)) | ✓ VERIFIED | `TestRed_R02_unackedDrain` (10 000 keys, gauge precondition, `redDrainGrace = 10s`, cap `3*redDrainKeys` read from `supkv_replication_datagrams_retransmitted_total`) → runner row `green / pass,run / OK`. The drain path is paced (`drainRecordSends`, ladder re-armed from actual emission, record re-read via `Lookup` and skipped on version mismatch, `core.go:1674-1729`). |
| 6 | Open-loop 1000 writes/s × 15 s over 10 000 distinct keys achieves ≥990 writes/s with the unacked set empty 5 s after the send window; the rate window is the open-loop send window and the ≤5 s drain assertion is separate (R-02(3)) | ✓ VERIFIED | `TestRed_R02_burst` → runner row `green / pass,run / OK`. `sendWindow = (total-1) × interval` (14.999 s) is the rate denominator; the drain bound is a separate `sendEnd + 5 s + 2 s` poll-margin wait (`stand_red_test.go:551-582`). Judged conforming to `REQUIREMENTS.md:56`'s two independent clauses (≥990 writes/s achieved; set empty 5 s after the end), matching the phase's code-review judgment of `6c17db3` — the earlier denominator silently demanded a ≤151 ms drain, stricter than the documented bound. No test bound was relaxed: the only diff in `stand_red_test.go` is this measurement alignment (`6c17db3`). |
| 7 | Key length and `value_max_bytes` ceilings are enforced: `Set`/`Delete` return `InvalidArgument` over 1 024 bytes before any engine call, startup exits 2 when the largest accepted record cannot fit one datagram, and any accepted record encodes into ≤65 507 bytes (R-03, XPORT-03) | ✓ VERIFIED | `storage.MaxKeyBytes = 1024` is the single constant (`store.go:20-27`); API checks at `internal/grpcapi/server.go:151,196` with `keyTooLongError` (`:299-307`) while `Get`/`List` stay permissive. `checkDatagramCeiling` in `finalizeConfig` exits with `exitUsage = 2` and names flag/key-limit/ceiling (`main.go:466-482`); `TestMain_valueMaxDatagram` proves 64 000 and 64 408 start and serve while 64 409 and 65 500 exit 2. `udprepl.MaxRecordDatagramBytes` marshals the real generated schema at worst-case varint widths (`maxdatagram.go:51-73`); `TestMaxRecordDatagramBytes_acceptedDomain` walks the accepted domain and asserts both real encoding and oracle ≤ 65 507; `TestStand_oversizeKeyF2c` proves local rejection of a 70 000-char key, non-replication, and the 1 024 + 61 440 positive boundary. |
| 8 | Received versions (including versions inside `DataSync`) are validated for RFC 9562 layout and the `now + max_clock_lead` bound before any store or clock call, counted as their own class, and WARN-logged rate-limited (XPORT-04) | ✓ VERIFIED | `Core.validate` runs between classification and `handleMessage` (`core.go:734-738`), and `handleMessage` is the only path to `ApplyRemote` (the sole `Clock.Update`) — pinned order for DataReplication/DataRequest/Ack/DataSync (`:741-812`). `version.RecordVersion.Validate` is pure/total, bounds via subtraction, accepts exactly `now+maxLead` and rejects +1 ms, ignores the 46 RFC 9562 filler bits (`version.go:84-119`). `TestDiscard_beforeClock` feeds one invalid datagram per class with a `countingClock` and asserts `Tick == 0 && Update == 0` and no store record; `TestDiscard_warnRateLimited` pins one WARN per class per 10 s of event-loop time; `--max-clock-lead`/`SUPKV_MAX_CLOCK_LEAD` default 60 s is passed as `repl.Config.MaxClockLead` (`main.go:378,596`; `TestMain_maxClockLead`). |
| 9 | Invalid datagram combinations (empty key, key too long, tombstone with data, bad checksum, bad version) are discarded with their own counters and source-only details; envelope CRC coverage is documented (XPORT-10) | ✓ VERIFIED | `DiscardStats` has one counter per class (`core.go:286-304`); `discardValidated` increments the class, fires `OnDiscard(class, source)`, and never carries a key name (`:845-875`); `validateReplication` recomputes `storage.RecordChecksum` before apply (`:782-797`); `keyClass` pins empty-before-too-long (`:823-835`). `TestDiscard_validationClasses`, `TestDiscard_totality`, `TestDiscard_onDiscardSeam`, `TestDecode_negativeTable` green. Docs: `docs/OPERATIONS.md:12` and `docs/SPEC-DEVIATIONS.md` S-PROTO-01 state the CRC covers the body bytes only. |
| 10 | The read loop exits only on `net.ErrClosed`; any other read error is counted, logged rate-limited, backed off, and survived (XPORT-08) | ✓ VERIFIED | `Adapter.readLoop` returns only on `errors.Is(err, net.ErrClosed)`; otherwise `readErrors.Add(1)`, rate-limited log (1/s), 100 ms backoff, then continue (`internal/udprepl/repl.go:894-941`). `TestReadLoop_survivesNonClosedError` (fake conn: error then datagram still processed) and `TestReadLoop_persistentErrorExitsOnClose` green; `TestMirror_unackedSent`/`TestSmoke_twoNodesUDP` green. |
| 11 | Startup logs the actual `SO_RCVBUF`/`SO_SNDBUF` and fails when the effective send buffer cannot hold a maximal datagram; 1, 20 and 60 KiB values deliver on Linux; the darwin ~9 KiB limit is documented, not fixed (XPORT-05) | ✓ VERIFIED | `checkSendBuffer` is the pure inclusive floor: 65 506 rejected, 65 507 accepted, naming requested and actual sizes (`socket_internal_test.go:69-167`); `New` closes the socket and returns the error (`repl.go:375-379`). `TestStand_bufferSizesDeliver` carries 1 024 / 20 480 / 61 440-byte values byte-exact between two real binaries and asserts positive parsed buffer lines on both. darwin: `docs/adr/0010-darwin-development-only.md` records "~9 KiB send limit documented, not fixed" (14.10). |
| 12 | Bind-address misuse is loud, duplicate instances fail, and timers jitter: a non-wildcard `--bind-addr` WARNs, a second instance fails with "address already in use", a node hearing no `Hello` for `peer_timeout` WARNs once, and Hello/sync deadlines are positive-only while the retransmit ladder is never jittered (XPORT-09) | ✓ VERIFIED | `warnOnNonWildcardBind` (`main.go:280`), `TestMain_bindAddrWarn` green; production bind sets no `SO_REUSEADDR` (`listenDatagram`, only the explicit `WithAddressReuse()` test option), `TestMain_duplicateBind` asserts exit non-zero + "address already in use", `TestListen_noAddressReuse` green; `warnIfNoPeer` latches one WARN after `firstAdvance + PeerTimeout` (`core.go:1325-1339`), `TestCore_noPeerWarn` green; `jitterOffset` is FNV over `(runID, seq, salt)` with `helloJitter`/`syncJitter` positive-only and `jitteredSpacing` ±10 % only on pacers (`core.go:1457-1501`), `TestSync_triggers`/`TestSync_pacing` pin the envelopes, and `TestRetransmit_backoffLadder`/`TestBound_lossWithinAckTimeoutMax` stay un-jittered and green. |
| 13 | The store exposes a key→version view and point reads with tombstone visibility, and `DataRequest`/`DataSync` handling does not depend on foreign records (flat between 1 000 and 50 000 stored keys) (XPORT-11, XPORT-06) | ✓ VERIFIED | `Store.Version` (one map read, no Data copy), `Store.Lookup` (clone, `Deleted` visible), `Store.Versions` (fresh shallow map) (`store.go:164-206`); `Get` stays live-only. `view_test.go` asserts tombstone visibility, clone ownership, snapshot independence and a concurrent hammer. `Engine.Lookup/Version/Versions` forward to the store (`engine.go:513-530`); `simnet.nodeStore` implements the four-method interface and `Cluster.Write/Delete` read back through `Lookup`, panicking on a rejected write (WR-02). `TestCore_keyCountCostFlat`: DataRequest ratio 0.97, DataSync ratio 1.00 (bound 2.0) → PASS. |
| 14 | "Record acknowledged by peer X" travels as an `Outbox` result to the adapter and to waiters, removing snapshot diffing and the 5 ms polling in `wait.go` (XPORT-12) | ✓ VERIFIED | `port.Outbox` carries two independent streams: `Drain()` and `Ack/Drained/Superseded/WaitSetEmpty` + `DrainResults()` (`internal/port/repl.go:114-221`); the core emits at every resolution site with the tracked version (`core.go:1014, 1056-1058, 1598, 2022`), including a stranger's Ack. The adapter creates the tracker synchronously in `NotifyLocalWrite` before the channel send and deletes it on every return path (`repl.go:604-634`; `TestNotify_createsTrackerBeforeCore`, `TestWaitReplicated_*Releases` green); `WaitReplicated` requires full write-time coverage and clamps to the write-time total (`TestWaitReplicated_ackCoverage`, `_strangerAckCounts`, `_evictionNotAck`, `_superseded`, `TestWaitIntegration_ackedThroughRealAdapterTracker`). `wait.go` is a 113-line projection with no poll, no snapshot diffing, no retention window; `mirrorState`/`retainDrained`/`drainedRetention`/`prevPending` are absent from production code. |
| 15 | The sync-answer queue is bounded and deduplicated: 1 000 missing keys listed twice leave 1 000 answers, not 2 000; eviction drops the oldest and deletes its dedupe identity (XPORT-07) | ✓ VERIFIED | `maxEgressEntries = 32768`; identity is `(handle, kind, key)`; `enqueueResponse` skips duplicates and `dropOldestEgress` deletes the identity in lockstep with a counted drop (`core.go:1146-1188`). `TestEgress_boundedDeduped` asserts exactly 1 000 after two 1 000-key listings; `TestEgress_identitySeparates`, `TestEgress_capDropsOldest`, `TestEgress_drainSkipsGoneRecords` green; drain re-reads a DataReplication answer via `Lookup` (`:1226-1235`). |
| 16 | Dead code is removed: `lastLocal`/`cover` and the above-cover branch, exported `persist.Parse`, `ownHandle`, `fsync_darwin.go`/`fsync_other.go` are gone; the darwin cross build and vet pass (XPORT-13) | ✓ VERIFIED | Repo-wide grep finds no `lastLocal`, `ownHandle`, `persist.Parse`, or `func Parse`; `internal/persistfile/fsync_darwin.go` and `fsync_other.go` are deleted from the tree (`fsops.go` calls `os.File.Sync`). `repl.New` is the four-argument constructor. `GOOS=darwin GOARCH=arm64 go build ./... && … go vet ./...` → OK at this tree. `S-PERS-03` registry row updated to the as-built durability path. |
| 17 | The red-gated expectation table reflects the phase: three transport rows pass under `SUPKV_RED_TESTS=1` (R-01, R-02 drain, R-02 burst), the four R-03/P-01 rows fail, and the mutation stage kills every listed seed while the control passes (TEST-04, D-01/D-02) | ✓ VERIFIED | `scripts/red-expectations.tsv` is 3 green / 4 red; `bash scripts/red-runner` at this tree exited 0 in 1m50.9s: "7 expectation row(s) checked, 0 error(s)", both mutants killed on every listed seed, control passing, evidence `…/red-evidence/20261121T235409Z`. Per-row verdicts: R-01/R-02 drain/R-02 burst = `pass,run`; R-03 ×2 and P-01 ×2 = `fail,run`. These four remain red by design (Phases 4/5 — see Deferred Items). |

**Score:** 17/17 truths verified (0 present, behavior-unverified).

### Deferred Items

Items not yet met but explicitly owned by later milestone phases. These are **not** Phase 3 gaps: the R-03 rows are `PROTO-01` (Phase 4) and the P-01 rows are `PERSIST-01` (Phase 5), and the expectation table keeps them red precisely so the flip happens there.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | R-03: interface not yet closed until the full key list arrives (`TestRed_R03_gateClosedUntilFullList`, `TestRed_R03_f14ReplacedNode` fail) | Phase 4 | ROADMAP Phase 4 SC 1 — "The interface is closed until the full key list arrives (R-03 red test turns green)" (PROTO-01) |
| 2 | P-01: value limit not yet defined once and enforced (`TestRed_P01_valueLimit`, `TestRed_P01_valueLimitEngine` fail) | Phase 5 | ROADMAP Phase 5 SC 1 — "The value limit is defined once and enforced everywhere (P-01 red test turns green)" (PERSIST-01) |

### Required Artifacts

36 artifact declarations across the 13 plans (28 unique paths); `gsd-tools query verify.artifacts` returns `all_passed: true` for every plan (exit 0). Unique paths:

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/udprepl/repl.go` | Buffer seam, read loop, tracker, policy | ✓ EXISTS + SUBSTANTIVE + WIRED | 1 214 lines; `configureSocketBuffers`/`checkSendBuffer`/`readLoop`/`WaitReplicated`; called by `main.wire` |
| `internal/udprepl/socket_internal_test.go` | Buffer read-back and floor proof | ✓ EXISTS + SUBSTANTIVE | 5 tests over a real loopback socket; boundary rows 65 506/65 507 |
| `internal/stand/stand_buffers_test.go` | 1/20/60 KiB delivery | ✓ EXISTS + SUBSTANTIVE | Two real binaries; byte-exact values; startup lines parsed |
| `cmd/sup-kv/main.go` | Wiring, flags, startup checks | ✓ EXISTS + SUBSTANTIVE | Ceiling, bind WARN, clock lead, buffer log, metrics seams |
| `internal/app/storage/store.go` | `Version`/`Lookup`/`Versions`, `MaxKeyBytes` | ✓ EXISTS + SUBSTANTIVE + WIRED | Clone/ownership contracts documented and tested |
| `internal/app/storage/view_test.go` | View contract proofs | ✓ EXISTS + SUBSTANTIVE | Tombstone visibility, clone ownership, snapshot, concurrent hammer |
| `internal/app/persist/engine.go` | Engine forwarding, dead-code removal | ✓ EXISTS + SUBSTANTIVE + WIRED | Forwards the three reads; `lastLocal`/`cover` gone |
| `internal/app/internal/simnet/simnet.go` | `nodeStore` four-method seam, WR-02 | ✓ EXISTS + SUBSTANTIVE + WIRED | Write/Delete read back via `Lookup`, panic on refusal |
| `internal/app/version/version.go` | `Validate` predicate | ✓ EXISTS + SUBSTANTIVE + WIRED | Pure/total; sentinels; used by `Core.versionAccepted` |
| `internal/app/version/validate_test.go` | Boundary + property proofs | ✓ EXISTS + SUBSTANTIVE | `TestValidate`, `TestValidate_property` |
| `internal/app/repl/core.go` | Pacers, ledger, heap, results, validation | ✓ EXISTS + SUBSTANTIVE | 2 108 lines; every Phase 3 mechanism lives here |
| `internal/app/repl/bench_test.go` | Both flatness gates | ✓ EXISTS + SUBSTANTIVE | `TestCore_keyCountCostFlat`, `TestCore_unackedCostFlat` |
| `internal/port/repl.go` | Result stream | ✓ EXISTS + SUBSTANTIVE + WIRED | `Drain`/`DrainResults` independent; four result kinds |
| `internal/grpcapi/wait.go` | Thin honest wait | ✓ EXISTS + SUBSTANTIVE + WIRED | No poll, no diffing; 113 lines |
| `internal/app/repl/egress_internal_test.go` | Bounded/deduped queue | ✓ EXISTS + SUBSTANTIVE | 1 000-not-2 000, cap drop, identity separation |
| `internal/app/repl/discard_test.go` | Classes and before-clock proof | ✓ EXISTS + SUBSTANTIVE | Counting-clock and rate-limit tests |
| `internal/udprepl/decode_negative_test.go` | Negative table | ✓ EXISTS + SUBSTANTIVE | Includes newly flipped bad_checksum/bad_version rows |
| `internal/udprepl/maxdatagram.go` | Datagram ceiling oracle | ✓ EXISTS + SUBSTANTIVE + WIRED | Real-schema marshal; used by `main.checkDatagramCeiling` |
| `internal/udprepl/maxdatagram_test.go` | Oracle and accepted-domain property | ✓ EXISTS + SUBSTANTIVE | Boundary rows and domain walk |
| `docs/SPEC-DEVIATIONS.md` | S-DM-03 / S-PERS-03 closures | ✓ EXISTS + SUBSTANTIVE | Rows updated with code links |
| `internal/udprepl/readloop_internal_test.go` | Read-loop error contract | ✓ EXISTS + SUBSTANTIVE | Fake conn survives non-closed errors; close exits |
| `internal/app/repl/warn_test.go` | No-Hello WARN | ✓ EXISTS + SUBSTANTIVE | One-shot latch |
| `internal/persistfile/fsops.go` | `os.File.Sync` durability | ✓ EXISTS + SUBSTANTIVE | Standard-library F_FULLFSYNC path |
| `scripts/red-runner` | Per-mutant source + clean-tree check | ✓ EXISTS + SUBSTANTIVE | Overlay-only, compile check, clean-tree before/after |
| `scripts/red-expectations.tsv` | 3 green / 4 red rows | ✓ EXISTS + SUBSTANTIVE | Three flips only |
| `internal/metrics/metrics.go` | New families and discard classes | ✓ EXISTS + SUBSTANTIVE + WIRED | `pacingFamilies` + ten discard labels |
| `.github/workflows/scheduled.yml` | `red-r01-x20` job | ✓ EXISTS + SUBSTANTIVE | `-count=20`, 110m/120m timeouts, no tolerance flags |
| `Makefile` | `race-soak` includes `internal/stand` | ✓ EXISTS + SUBSTANTIVE | Measured-cost comment (6.53 s / ~2.2 min at 20) |

**Artifacts:** 36/36 declarations verified (28/28 unique paths exist and are substantive).

### Key Link Verification

`gsd-tools query verify.key-links` returns `verified: 0/39` for these plans because its heuristic looks for the literal target path inside the source file; every Phase 3 link is expressed in Go as an import/package-qualified call, so the verb is a false negative here (`detail: "Target not referenced in source"` even for same-file links). All 39 links were verified manually by reading the call sites; **39/39 wired**.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `cmd/sup-kv/main.go` | `internal/udprepl/repl.go` | `wire()` logs `adapter.SocketBuffers()` | ✓ WIRED | `main.go:514-523` |
| `internal/udprepl/repl.go` | `internal/udprepl/repl.go` | `New` → `configureSocketBuffers` | ✓ WIRED | `repl.go:370-393` |
| `internal/stand/stand_buffers_test.go` | `internal/stand/stand.go` | Harness drives real binaries | ✓ WIRED | `startStandLoggedNode`/`standSetWith`/`waitFor` |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | Listing/DataRequest/DataSync consume `Version`/`Versions`/`Lookup` | ✓ WIRED | `core.go:1088-1093, 1116-1133, 1513-1520` |
| `internal/app/persist/engine.go` | `internal/app/storage/store.go` | `Engine.Lookup/Version/Versions` delegate | ✓ WIRED | `engine.go:513-530` |
| `internal/app/internal/simnet/simnet.go` | `internal/app/storage/store.go` | `Cluster.Write/Delete` read via `Lookup` | ✓ WIRED | `simnet.go:436-449, 739, 777` |
| `internal/app/repl/core.go` | `internal/app/version/version.go` | `RecordVersion.Validate` before any clock touch | ✓ WIRED | `core.go:841-843` |
| `internal/app/version/validate_test.go` | `internal/app/version/version.go` | `PackVersion`/layout constants as oracle | ✓ WIRED | test fixture |
| `internal/app/internal/simnet/simnet.go` | `internal/app/repl/core.go` | `nodeStore` implements the shrunk `StateStore` | ✓ WIRED | `simnet.go:435-449` |
| `internal/app/repl/bench_test.go` | `internal/app/repl/core.go` | Flatness tests drive `HandleDatagram` | ✓ WIRED | `bench_test.go:99-194` |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | Retransmit resolves via `Lookup`, skips version mismatch | ✓ WIRED | `core.go:1707-1728` |
| `internal/udprepl/repl.go` | `internal/app/repl/core.go` | Mirror uses `PeerCount`/O(1) `UnackedStats` | ✓ WIRED | `repl.go:1018-1019` |
| `internal/app/repl/bench_test.go` | `internal/app/repl/core.go` | `BenchmarkCoreEventLoop` at 10/10 000 | ✓ WIRED | `bench_test.go:229-300` |
| `internal/app/repl/core.go` | `internal/port/repl.go` | Result events appended to the `Outbox` | ✓ WIRED | `core.go:1014, 1056-1058, 1598, 2022` |
| `internal/udprepl/repl.go` | `internal/grpcapi/wait.go` | `WaitReplicated` injected as `ServerOptions.WaitReplicated` | ✓ WIRED | `main.go:535`; `wait.go:66-84` |
| `internal/grpcapi/server.go` | `internal/udprepl/repl.go` | `track` flag → adapter tracker before enqueue | ✓ WIRED | `server.go:166,203,281`; `repl.go:604-634` |
| `internal/app/repl/core.go` | `internal/app/repl/core.go` | Shared normalized `syncResponseInterval`, min deadlines | ✓ WIRED | `core.go:643, 1217, 1449, 1693, 1740-1766` |
| `internal/app/repl/sync_test.go` | `internal/stand/stand_red_test.go` | Unit pacing asserts the same arithmetic | ✓ WIRED | identical `floor(1s/rate)` expression |
| `internal/app/repl/sim_chaos_test.go` | `internal/app/repl/core.go` | `chaosQuiescent` consults `QueuedSends` | ✓ WIRED | `sim_chaos_test.go:446-463`; `core.go:1904-1906` |
| `internal/app/repl/core.go` | `internal/app/version/version.go` | `Validate(now, MaxClockLead)` for records and entries | ✓ WIRED | `core.go:793, 807` |
| `internal/app/repl/core.go` | `internal/app/storage/store.go` | `RecordChecksum` recomputed; `MaxKeyBytes` key bound | ✓ WIRED | `core.go:789, 831` |
| `internal/udprepl/repl.go` | `internal/app/repl/core.go` | Five counters mirrored into atomics / `Discards()` | ✓ WIRED | `repl.go:260-269` |
| `cmd/sup-kv/main.go` | `internal/app/repl/core.go` | `--max-clock-lead` → `repl.Config.MaxClockLead` | ✓ WIRED | `main.go:378, 596` |
| `internal/grpcapi/server.go` | `internal/app/storage/store.go` | API compares against `storage.MaxKeyBytes` | ✓ WIRED | `server.go:151,196,307` |
| `cmd/sup-kv/main.go` | `internal/udprepl/maxdatagram.go` | Ceiling through the real codec → exit 2 | ✓ WIRED | `main.go:466-482` |
| `internal/udprepl/maxdatagram_test.go` | `internal/udprepl/repl.go` | Encodes via the real codec helper | ✓ WIRED | `EncodeDatagramForTest` |
| `internal/udprepl/repl.go` | `net` | `errors.Is(err, net.ErrClosed)` only exit | ✓ WIRED | `repl.go:906-926` |
| `cmd/sup-kv/main.go` | `internal/udprepl/repl.go` | Bind WARN from the bound address; no reuse option | ✓ WIRED | `main.go:280`; `repl.go:347-445` |
| `internal/app/repl/core.go` | `internal/app/repl/core.go` | `Advance` → `warnIfNoPeer` one-shot | ✓ WIRED | `core.go:1300-1339` |
| `internal/app/persist/engine.go` | `internal/app/persist/engine_internal_test.go` | Cover tests deleted with the branch | ✓ WIRED | file deleted (`D` in phase diff) |
| `internal/app/repl/core.go` | all `repl.New` call sites | Four-argument constructor | ✓ WIRED | `go build ./...` clean |
| `internal/persistfile/fsops.go` | `os.File.Sync` | `syncDurable` calls `f.Sync()` | ✓ WIRED | `fsops.go:99-105, 149-159` |
| `internal/metrics/metrics.go` | `internal/udprepl/repl.go` | `Sources` gains four accessors | ✓ WIRED | `metrics.go:193-196, 208` |
| `cmd/sup-kv/main.go` | `internal/metrics/metrics.go` | `metricsSources` wires the four | ✓ WIRED | `main.go:616-619` |
| `scripts/red-runner` | `internal/app/storage/store.go` | Tombstone mutant overlay on the store source | ✓ WIRED | `red-runner:90,129-131` |
| `scripts/red-expectations.tsv` | `internal/stand/stand_red_test.go` | Three transport rows flipped after the fixes | ✓ WIRED | diff is exactly the three flips |
| `.github/workflows/scheduled.yml` | `scripts/red-expectations.tsv` | 20× job asserts the green R-01 behavior | ✓ WIRED | `scheduled.yml:149-183` |
| `Makefile` | `internal/stand` | `race-soak` adds the package | ✓ WIRED | `Makefile:66-67` |

**Wiring:** 39/39 key links verified.

### Data-Flow Trace (Level 4)

The phase's dynamic values are metrics and wire sizes, not rendered UI. Every published value traces to live core/adapter state; no static return, canned fixture, or mock is on a production path.

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `internal/metrics/metrics.go` `supkv_replication_unacked_records` / `_peers` / `_oldest_age_seconds` | unacked count, oldest age, peer count | `Adapter` post-event atomics (`repl.go:1018-1019`) ← `Core.UnackedStats`/`PeerCount` (O(1)) | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_replication_datagrams_retransmitted_total` | retransmit count | `Core.retransmitted` incremented at emission (`core.go:1721-1723`) mirrored by adapter | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_datagrams_discarded_total` (10 classes) | class counters | `Core.discards` incremented at each discard site (`core.go:696-737, 853-865`) | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_replication_datagrams_paced_total` / `_send_queue` / `_egress_dropped_total` / `_read_errors_total` | pacer count, queue depth, drops, read errors | `Core.pacedSends`, `QueuedSends()`, `egressDropped`, `Adapter.readErrors` | Yes | ✓ FLOWING |
| `internal/metrics/metrics.go` `supkv_replication_datagrams_sent_total` | sent by type | `Adapter.SentByType` incremented on encode-successful send | Yes | ✓ FLOWING |
| `internal/grpcapi/wait.go` report | `acked`, `ackedCount`, `total` | `Adapter.WaitReplicated` tracker fed by `Outbox.DrainResults`; count clamped to write-time total | Yes | ✓ FLOWING |
| `internal/udprepl/maxdatagram.go` | datagram size | Real `proto.Marshal` of the generated schema at worst-case widths | Yes | ✓ FLOWING |
| `cmd/sup-kv/main.go` startup buffer line | effective recv/send sizes | `getsockopt` read-back from the bound socket (`repl.go:474-478`) | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Documented-scale R-01 (20 000 keys × 1 KiB, empty receiver) | `SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 go test -race -count=1 -timeout 15m ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'` | `--- PASS: TestRed_R01_pacedSync20k (108.86s)` | ✓ PASS |
| Full red-gated expectation table + mutation stage | `bash scripts/red-runner` | exit 0; `7 expectation row(s) checked, 0 error(s)`; mutants killed on every listed seed; control pass; 1m50.9s | ✓ PASS |
| Green rows actually pass / red rows actually fail | Read `…/red-evidence/20261121T235409Z/summary.txt` | R-01, R-02 drain, R-02 burst = `pass`; R-03 ×2, P-01 ×2 = `fail`; all `OK` | ✓ PASS |
| XPORT-02(1) 10 vs 10 000 unacked flatness | `go test -race -count=1 ./internal/app/repl/ -run '^TestCore_unackedCostFlat$' -v` | ratio 1.05 (1218.0 → 1277.0 ns/op) | ✓ PASS |
| XPORT-06 1 000 vs 50 000 stored-key flatness | `go test -race -count=1 ./internal/app/repl/ -run '^TestCore_keyCountCostFlat$' -v` | DataRequest 0.97, DataSync 1.00 | ✓ PASS |
| Buffer seam and floor | `go test -race -count=1 ./internal/udprepl/ -run '^(TestSocketBuffers|TestCheckSendBuffer|TestSocketBuffers_boundaries|TestListen_noAddressReuse)$' -v` | all pass; 65 506 rejected, 65 507 accepted | ✓ PASS |
| Read-loop error contract | `go test -race -count=1 ./internal/udprepl/ -run '^(TestReadLoop_survivesNonClosedError|TestReadLoop_persistentErrorExitsOnClose)$'` | pass | ✓ PASS |
| 1/20/60 KiB delivery on real sockets | `go test -race -count=1 ./internal/stand/ -run '^TestStand_bufferSizesDeliver$'` (part of the green stand package run) | pass | ✓ PASS |
| Build, vet, lint | `go build ./... && go vet ./...`; `make lint` | clean; `config verify` + strict run clean | ✓ PASS |
| Darwin availability after deletions | `GOOS=darwin GOARCH=arm64 go build ./... && … go vet ./...` | OK | ✓ PASS |
| Package suites | `go test -race -count=1` over `version`, `storage`, `repl`, `udprepl`, `metrics`, `persistfile`, `grpcapi`, `cmd/sup-kv`, `stand`, `persist`, `simnet`, `clock` | all `ok` (repl 90.4 s, stand 6.0 s, cmd 5.9 s, others ≤ 5 s) | ✓ PASS |

### Probe Execution

Not applicable — Phase 3 declares no runnable `scripts/*/tests/probe-*.sh` probes (the only `scripts/` executables are `red-runner` and its expectation table, both exercised above). The plans' "Probe XPORT-xx" labels are design probes pinned by named tests, verified in the truth table.

### Requirements Coverage

All 13 requirement IDs are declared across the plan frontmatter (`03-01`…`03-13` collectively name XPORT-01…13), and REQUIREMENTS.md maps exactly XPORT-01…13 to Phase 3 with no orphaned requirement.

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| XPORT-01 | 03-07, 03-12, 03-13 | Sync listing paced and robust to buffer loss (R-01) | ✓ SATISFIED | Paced/rotating listing (truth 1), 20k convergence + green row + 20× job (truth 2), buffers logged + view (truth 3) |
| XPORT-02 | 03-05, 03-07, 03-12, 03-13 | Event-loop cost flat; retransmits paced with jitter (R-02) | ✓ SATISFIED | Truths 4–6; record-free ledger, O(1) gauges, paced record queue, ladder from emission |
| XPORT-03 | 03-08, 03-09, 03-12 | Key length and `value_max_bytes` validated (R-06) | ✓ SATISFIED | Truth 7; single `MaxKeyBytes`, exit 2, ≤65 507 property, F2c |
| XPORT-04 | 03-03, 03-08, 03-12 | Received versions validated before the clock (R-07) | ✓ SATISFIED | Truth 8; counting-clock proof, 60 s configurable lead |
| XPORT-05 | 03-01, 03-13 | Socket buffers configured and reported (R-08) | ✓ SATISFIED | Truth 11; read-back logging, send floor, 1/20/60 KiB, darwin ADR |
| XPORT-06 | 03-02, 03-04, 03-12 | `DataRequest`/sync chunks independent of foreign records (R-09) | ✓ SATISFIED | Truth 13; four-method `StateStore`, ratio 0.97/1.00 |
| XPORT-07 | 03-07, 03-12 | Sync-answer queue bounded and deduplicated (R-11) | ✓ SATISFIED | Truth 15; 1 000-not-2 000 proof, lockstep eviction |
| XPORT-08 | 03-10, 03-13 | Read loop does not silently die (R-14) | ✓ SATISFIED | Truth 10; fake-conn tests |
| XPORT-09 | 03-07, 03-10, 03-13 | Bind misuse loud, duplicate instances fail, timers jitter (R-15) | ✓ SATISFIED | Truth 12; WARN, EADDRINUSE, no-Hello WARN, jitter envelopes |
| XPORT-10 | 03-08, 03-12 | Invalid datagram combinations rejected and counted (R-16) | ✓ SATISFIED | Truth 9; five classes + counters + docs |
| XPORT-11 | 03-02, 03-04, 03-12 | Key→version view and tombstone-visible point reads (Q-01) | ✓ SATISFIED | Truth 13; `Version`/`Lookup`/`Versions` with clone/fresh-map tests |
| XPORT-12 | 03-06, 03-12 | Ack-by-peer-X as an `Outbox` result to adapter and waiters (Q-02) | ✓ SATISFIED | Truth 14; result stream, tracker, poll removal |
| XPORT-13 | 03-11, 03-12 | Dead code removed (Q-09) | ✓ SATISFIED | Truth 16; grep + darwin cross build |

**Coverage:** 13/13 requirements satisfied; 0 blocked; 0 orphaned.

### Decision Coverage

`gsd-tools query check.decision-coverage-verify` → `{total: 8, honored: 8, not_honored: [], skipped: false}` — "All trackable CONTEXT.md decisions are honored by shipped artifacts." (Non-blocking gate; 8/8 honored.)

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/stand/stand_red_test.go` | XPORT-01/02, (PROTO-01) | 5 | 0 when `SUPKV_RED_TESTS=1` (env gate is the D-01 mechanism, opened by the runner) | No — asserts real binary convergence/metrics | Behavioral (bounds + metrics + process lifecycle) | ✓ |
| `internal/stand/stand_buffers_test.go` | XPORT-05 | 1 | 0 | No | Behavioral (byte-exact delivery on real sockets) | ✓ |
| `internal/app/repl/bench_test.go` | XPORT-02/06 | 2 | 0 | No — independent measurement of two fixture sizes | Value (ratio with a hard 2.0 bound) | ✓ |
| `internal/app/repl/discard_test.go` | XPORT-03/04/10 | 5 | 0 | No — expected classes from the code's own contract | Behavioral (zero Tick/Update, per-class counters) | ✓ |
| `internal/app/repl/egress_internal_test.go` | XPORT-07 | 4 | 0 | No | Value (queue depth 1 000 vs 2 000) | ✓ |
| `internal/udprepl/socket_internal_test.go` | XPORT-05 | 5 | 0 | No | Value (getsockopt equality, floor boundary) | ✓ |
| `internal/udprepl/readloop_internal_test.go` | XPORT-08 | 2 | 0 | No | Behavioral (survive/exit branches) | ✓ |
| `internal/udprepl/maxdatagram_test.go` | XPORT-03 | 2 | 0 | No — oracle compared against real encoding | Value + domain walk | ✓ |
| `internal/app/version/validate_test.go` | XPORT-04 | 2 | 0 | No | Value (boundary ±1 ms) + totality property | ✓ |
| `internal/grpcapi/wait_test.go`, `wait_integration_test.go`, `internal/udprepl/repl_test.go` | XPORT-12 | 15 | 0 (one capability probe skip is loud, only if loopback broadcast is unavailable) | No | Behavioral (full-coverage success, honest false paths) | ✓ |

**Disabled tests on requirements:** 0 — the only `t.Skip` on requirement tests is the intentional `SUPKV_RED_TESTS` gate, which the red-runner opens (proven: rows report `pass,run`, not `skip`).
**Circular patterns detected:** 0.
**Insufficient assertions:** 0 — every requirement-linked test asserts values or multi-step behavior, not mere existence.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | `TBD`/`FIXME`/`XXX` in any phase-modified non-planning file | none | Debt-marker gate: scanned all Go/Makefile/workflow/docs files changed by the phase — zero matches |
| — | — | `TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented" | none | Zero matches in phase-modified files |
| — | — | Stub returns (`return nil, nil`, empty handlers, static metric values) | none | No production stub found; every metric/discard/wait value traces to live state (Level 4 above) |

**Anti-patterns:** 0 found (0 blockers, 0 warnings).

### Code Review Findings (advisory — not Phase 3 exit criteria)

The phase-directory review (`03-REVIEW.md`, 2026-11-21T23:38Z) recorded 2 critical, 4 warning, 7 info findings; all remain `open` in `03-REVIEW-DISPOSITION.md` and are routed to later phases. They are post-flip availability/efficiency observations in the new queue machinery, not failures of this phase's stated bounds — none is backed by a failing requirement test, and the review itself records that the design intent checks out on every XPORT acceptance point. They are reported here for the record and do not affect the verdict.

| Finding | Severity | Substance | Disposition |
|---------|----------|-----------|-------------|
| CR-01 | critical | Restart-listing FIFOs are unbounded in destination count and keyed by the attacker-controlled source handle; each restart Hello also rebuilds `Versions()` + sorts. Reproduced in an overlay test (100 spoofed Hellos → 96 FIFOs / 4 800 queued chunks). | Follow-up candidate Phase 4/7 window (per disposition ledger) |
| CR-02 | critical | The bounded deduped egress queue still retains a full record copy per queued answer (~1.9 GiB at the 32 768 cap with 61 440-byte values) and allocates before the dedupe check, though the drain re-reads the record anyway. | Follow-up candidate Phase 4 |
| WR-01 | warning | Tracker leak when `WaitReplicated` is nil (single-node wiring) while the notify seam creates trackers. | Open — Phase 4/6 |
| WR-02 | warning | `sync_chunk_bytes` unbounded — a typo can OOM the first listing or make unsendable chunks. | Open — Phase 7 config validation |
| WR-03 | warning | `red-r01-x20` evidence step can print a stale committed summary on a fresh checkout (it correctly states the direct-`go test` case otherwise). | Open — CI hygiene |
| WR-04 | warning | `freeUDPPort` pick-then-bind race (3/3 reruns green); known flake class carried from Phase 2. | Open — known flake class |
| IN-01..IN-07 | info | Apply-failure WARN names the key (Phase 7); flatness gate does not cover remaining whole-pending scans on the apply path (Phase 8 — see note); wall-clock ratio test is `t.Parallel`; 50 000-key fixture counter domain; `wait_for_replication` docs wording (Phase 6); `MaxKeyBytes` doc wording (Phase 9); `datagrams_sent_total` counts failed sends (Phase 7). | Open — routed per ledger |

**Residual note (IN-02):** `Core.applyRemote` scans the whole pending map per applied remote record (`core.go:1012-1017`), so a node that both holds a large unacked set and receives replicas pays O(unacked) on that path. The XPORT-02(1) acceptance is measured on the event-loop iteration (Advance/gauges) and passes at 1.05; the drain and burst scenarios pass; this residual scan is deliberately left to Phase 8 (PERF) and is recorded here rather than treated as an XPORT-02 failure.

### Human Verification Required

N/A — infrastructure/foundation phase with no user-facing elements. Every acceptance criterion is programmatic (real-socket bounds, process exit codes, metrics, buffer read-backs, codec sizes, cross builds). No behavior-unverified truth exists (`behavior_unverified: 0`), so nothing routes to UAT.

## Gaps Summary

No gaps. The phase goal is achieved in the codebase, not merely claimed in the summaries:

- **Paced and bounded sending is real:** listings, records and reactive egress share one normalized pacer (`floor(1s/sync_response_rate)`), one in-flight listing per destination, rotating start chunks, and a deduplicated record queue whose ladder re-arms from actual emission time.
- **Buffers are configured and reported honestly:** the kernel's doubled/clamped values are read back with `getsockopt`, logged at startup, and the send floor fails startup at 65 506 bytes while accepting 65 507.
- **Input is validated before the clock:** the validation pipeline sits between classification and the only store/clock path, proven with a counting clock (zero `Tick`/`Update` for every discard class), with five new classes/counters and rate-limited WARNs.
- **The recovery and burst scenarios converge within their documented limits:** the red gate reports R-01, R-02 drain and R-02 burst green (and I re-ran the documented-scale R-01 to a pass), while the four R-03/P-01 rows fail exactly as Phases 4/5 require.
- **The seams later phases build on exist:** the tombstone-visible key→version view, the `Outbox` result stream with a tracker-backed honest wait and no polling, the bounded deduped sync-answer queue, and the dead-code removals.

Evidence-scope notes (recorded, not gaps):
1. The 20/20 repetition of R-01 is the scheduled `red-r01-x20` CI job's standing acceptance (wired with `-count=20` at 20 000 keys and no tolerance flags); its first scheduled execution remains a CI observation. Locally I verified one documented-scale run (108.86 s) and the nightly-scale red gate row.
2. The `TestConfig_parametersByEffect/peer_timeout_evicts_silent_peer` flake class is recorded (one prior run directory, `20261121T231607Z`, shows it); my runs were green and idle re-runs are green.
3. `internal/stand/stand_red_test.go` differs from the phase-2 baseline only by the blessed `6c17db3` measurement alignment (rate window = open-loop send window; the ≤5 s drain check unchanged and separate). No bound, skip, or assertion strength was weakened.
4. `scripts/red-expectations.tsv` differs only by the three red→green flips.
5. The working tree is not git-clean at verification time: the untracked evidence directory from this verification run and an uncommitted `03-VALIDATION.md` wording fix (5 green/2 red → 3 green/4 red) are present. Both are post-phase records, not production changes; the phase's own final gate ran on a clean tree.

---

_Verified: 2026-11-22T00:01:44Z_
_Verifier: ZCode (gsd-verifier)_
