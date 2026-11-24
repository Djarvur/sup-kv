# Requirements: sup-kv

**Defined:** 2026-11-19
**Core Value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.

**Source:** v1.0 review of 2026-11-19 (`docs/reviews/2026-11-19-v1.0/`). Review IDs (R-xx, P-xx, S-xx, A-xx, O-xx, T-xx, D-xx, PERF-xx, Q-xx) are kept in parentheses for traceability; acceptance criteria are carried from the review verbatim or near-verbatim. Owner decisions 14.1–14.11 of the review are settled (2026-11-19) and are not re-litigated here.

## v1.1 Requirements

### Spec Traceability

- [x] **SPEC-01**: Repository carries a spec deviation registry mapping every Spec statement ID to a decision and a code/ADR link, and every section-7 deviation of the review is closed by code or by registry entry (S-12)
  - Acceptance: registry exists; each review section-7 deviation (S-01, S-02, S-06, S-07, S-08, S-12) closed one of the two ways; `docs/IDEA.md` marked historical or replaced by an as-built description
- [x] **SPEC-02**: Each owner decision 14.1–14.11 is recorded as an ADR, including the two confirmations the review left with the owner
  - Acceptance: 11 ADRs exist; 14.5(6) read-before-value-arrives → `Unavailable` and 14.7 last-read-value-file outside sup-kv are recorded as accepted (owner may reopen at Phase 4/6 discussion before implementation)

### Test Harness

- [x] **TEST-01**: `TestChaos_convergence` is stable and mutation-sensitive (T-01)
  - Acceptance: 0 failures on 10 000 seeds; fixed seed corpus plus fresh seeds each run; test fails under "sync without tombstones" and "periodic sync disabled" mutants
- [x] **TEST-02**: Deletion replication is proven at every level (T-02)
  - Acceptance: codec round-trip of every message with `Deleted: true`; `Cluster.Delete` in the simulator and deletions in chaos; two processes: `Delete` on A → `NotFound` on B and `"deleted":true` in B's dump; a node down during the delete learns it after start
- [x] **TEST-03**: "Confirmed write is on disk" is pinned by tests, not by convention (T-03)
  - Acceptance: filesystem seam under `persistfile` with recording and failing fakes asserting the order "write → fsync → rotate → rename → dir fsync" and a failure at each step; blocking `Save` fake — while it stands, `Set` does not return; processes: single node `Set` → OK, SIGKILL, start, `Get` succeeds
- [x] **TEST-04**: Startup gate is tested as a property, not a boolean flag (T-04)
  - Acceptance: the R-03 acceptance tests (see PROTO-01) fail on the current code
- [x] **TEST-05**: Real-socket tests are never silently skipped (T-05)
  - Acceptance: in Linux CI a skipped socket test is an error; the macOS cell either runs at least one two-node test or the docs state darwin replication is untested; the test binary is built with `-race`
- [x] **TEST-06**: Simulator blind spots are closed (T-06)
  - Acceptance: simulator gains per-node incoming-datagram budget, size checks by the real encoder, record announcement after a virtual dump delay, restart with state, deletions; nightly real-socket run at 10 000 keys × 1 KiB scale with the review's section-15 scenarios
- [x] **TEST-07**: `persist.Open` clock seeding is proven for the real composition path (T-07)
  - Acceptance: `Open` with injected time earlier than the dump max — the first local write dominates all loaded versions
- [x] **TEST-08**: Datagram decoder has negative tests and fuzzing (T-08)
  - Acceptance: table of corrupted datagrams with the expected discard class; `FuzzDecode` and `FuzzDumpDecode` with an in-repo corpus run on a schedule
- [x] **TEST-09**: Convergence bounds hold under combined failures and long partitions (T-09)
  - Acceptance: three-node test with a partition longer than `peer_timeout` — convergence not before the first post-heal sync and not later than the bound; the bound is either documented with its no-fresh-loss qualifier or recalculated
- [x] **TEST-10**: Benchmarks measure what they claim (T-10)
  - Acceptance: no benchmark paces itself; real `persistfile` and default parameters; p50 and p99 reported; smoke run `-benchtime=1x` in CI
- [x] **TEST-11**: Known flakiness classes are eliminated (T-11)
  - Acceptance: `go test -race -count=20` over `cmd/sup-kv`, `udprepl` and the affected package green three times in a row
- [x] **TEST-12**: Metrics and configuration are tested by observed effect (T-12)
  - Acceptance: one test per parameter observing its effect; `/metrics` after a conflict+retransmit+discard scenario reports exact values
- [x] **TEST-13**: Every section-14 decision has a test referencing it by name (T-13)
- [x] **TEST-14**: Watch cut and backpressure branches are covered (T-14)
  - Acceptance: deterministic snapshot-then-live cut test; slow-subscriber disconnect and `Send` error branches covered
- [x] **TEST-15**: Property tests minimize counterexamples and explore equal-`l` versions (T-15)
  - Acceptance: half the drawn versions from a small `(l, c, node)` space; the printed replay command works
- [x] **TEST-16**: Scheduled CI workflow exists (T-16)
  - Acceptance: 1000 seeds, `-shuffle=on -count=3`, `-bench . -benchtime=1x` on a schedule

### Transport

- [x] **XPORT-01**: Sync listing is paced and robust to buffer loss (R-01)
  - Acceptance: two nodes on real Linux sockets with default buffers, 20 000 keys × 1 KiB, empty receiver — full convergence no later than 2 × `sync_interval` + N / `sync_response_rate`, 20 runs out of 20; test in CI; sending start chunk rotates across rounds; `SO_RCVBUF`/`SO_SNDBUF` set with actual values logged; the core gets a cheap key→version view instead of a store copy
- [x] **XPORT-02**: Event-loop cost does not grow with the unacked set; retransmits are paced with jitter (R-02)
  - Acceptance: (1) benchmark — iteration time at 10 and at 10 000 unacked records differs by at most 2×; (2) stand — a peer returning to a node holding 10 000 unacked 1 KiB records: set empty within N / `sync_response_rate` + 10 s, at most 3N retransmits; (3) open loop 1000 writes/s × 15 s over 10 000 distinct keys — at least 990 writes/s achieved, set empty 5 s after the end
- [x] **XPORT-03**: Key length and the `value_max_bytes` ceiling are validated (R-06)
  - Acceptance: `Set` with a key longer than the documented limit returns `InvalidArgument`; a `value_max_bytes` under which the largest record does not fit a datagram exits with code 2 at startup; property: any accepted record encodes into a datagram ≤ 65 507 bytes
- [x] **XPORT-04**: Received versions are validated before touching the clock (R-07)
  - Acceptance: a version with invalid `ver`/`var` fields or `l` beyond now + a configurable bound is discarded before any clock call, counted as its own class and WARN-logged; the same for versions inside `DataSync`
- [x] **XPORT-05**: Socket buffers are configured and reported (R-08)
  - Acceptance: startup logs the actual sizes of both buffers; startup fails if the send buffer is smaller than the maximal datagram; values of 1, 20 and 60 KiB are delivered on Linux; the darwin ~9 KiB send limit is documented, not fixed (14.10)
- [x] **XPORT-06**: Handling of `DataRequest` and sync chunks does not depend on foreign records (R-09)
  - Acceptance: benchmark on 1000 and 50 000 keys within 2×
- [x] **XPORT-07**: The sync-answer queue is bounded and deduplicated (R-11)
  - Acceptance: two listings with the same 1000 missing keys leave 1000 answers queued, not 2000
- [x] **XPORT-08**: The read loop does not silently die (R-14)
  - Acceptance: the loop exits only on `net.ErrClosed`; any other read error is logged and counted, or the process exits; a node never keeps sending `Hello` while deaf
- [x] **XPORT-09**: Bind-address misuse is loud; duplicate instances fail; timers jitter (R-15)
  - Acceptance: a non-wildcard `--bind-addr` is rejected or produces an explanatory WARN; a second instance on the same address fails with "address already in use"; a node that received no `Hello` for `peer_timeout` after start WARNs
- [x] **XPORT-10**: Invalid datagram combinations are rejected and counted (R-16)
  - Acceptance: empty key, tombstone with data, unverified `Checksum` discarded with their own counters; envelope CRC coverage defined in docs
- [x] **XPORT-11**: The store exposes a key→version view and point reads with tombstone visibility (Q-01; backs R-09, P-06, A-02)
- [x] **XPORT-12**: "Record acknowledged by peer X" travels as an `Outbox` result to the adapter and to waiters (Q-02; removes snapshot diffing and the 5 ms polling in `wait.go`, feeds A-01)
- [x] **XPORT-13**: Dead code is removed (Q-09)
  - Acceptance: `lastLocal`/`cover` branch, exported `persist.Parse`, `ownHandle` field and `fsync_darwin.go` are gone or justified

### Protocol

- [x] **PROTO-01**: The interface stays closed until the full key list arrives (R-03, 14.5)
  - Acceptance: F14 on the stand and in the simulator — the value on A does not change; before the list B answers `Unavailable` for both reads and writes; after it, `create_if_absent` for a peer-known key returns `AlreadyExists`; with a three-part list the first two parts keep the interface closed; after the part carrying the max version `(now + 10 s)` the next local write gets a greater version; `DataSync` carries `SyncID`/`ChunkIndex`/`ChunkCount`; HLC `Update` on each part's max version; health stays `NOT_SERVING` until the list is complete; reading a listed-but-not-yet-arrived key returns `Unavailable` (14.5(6), accepted)
- [x] **PROTO-02**: An empty-disk node waits for a peer longer than a node with data (14.6)
  - Acceptance: new `startup_gate_empty` parameter (default 60 s); a node with data opens via `startup_gate`; two empty nodes exchange empty lists and open without waiting for the new parameter
- [x] **PROTO-03**: A stale `DataReplication` is answered with the newer record, not acknowledged (R-04)
  - Acceptance: node holds `k@v5`, receives `k@v3` — exactly one unicast `DataReplication k@v5` leaves, no `Ack`; the sender's `wait_for_replication` for `v3` returns `acked: false`
- [x] **PROTO-04**: A newly seen or returning peer gets an immediate sync (R-05)
  - Acceptance: the first `Hello` from an unknown `node_id` produces exactly one full listing toward it, a repeat produces nothing; a peer returning after `peer_timeout` + 1 s receives the neighbor's state within N / `sync_response_rate` plus margin, without the `sync_interval` term; `TestRestart_newPeerNoFullSync` rewritten
- [x] **PROTO-05**: Local-write notifications never regress the tracked version (R-10)
  - Acceptance: a notification with a version not greater than the tracked one is ignored
- [x] **PROTO-06**: Duplicate `node_id` with alternating `run_id`s is a named configuration error (R-12)
  - Acceptance: a third process under a known `node_id` is counted and logged as a configuration error; no full listing is sent to it
- [x] **PROTO-07**: `proto_version` is checked before body decode (R-13)
  - Acceptance: an incompatible version yields the `bad_proto_version` class and a WARN rate-limited per peer per interval
- [x] **PROTO-08**: Every local change replicates, including dump-failed writes and snapshot loads (S-02)
  - Acceptance: a write whose dump failed is broadcast and counted as unconfirmed; a snapshot load broadcasts its result (with APIS-07)
- [x] **PROTO-09**: Three-or-more-node behavior is described and tested (14.9)
  - Acceptance: behavior and timing for 3+ nodes documented; real-socket tests with three nodes; `wait_for_replication` semantics at N > 2 in the API docs

### Persistence

- [x] **PERSIST-01**: The value limit is defined in one place and enforced (P-01, Q-04)
  - Acceptance: a record of exactly `value_max_bytes` survives a restart at any accepted flag value; an invalid value exits with code 2; the 61 440 constant exists once, passed by configuration
- [x] **PERSIST-02**: Dump-load outcomes are visible and never destroy evidence (P-02)
  - Acceptance: start from `.prev` — one WARN with path and cause; both generations unusable — ERROR and a nonzero metric; files absent — neither; unusable files are preserved under a separate name, not overwritten by rotation
- [x] **PERSIST-03**: Generation integrity is explicit (P-03)
  - Acceptance: the format carries a record count or a terminator; a file truncated at a line boundary is recognized as incomplete; one corrupted record discards only that record
- [x] **PERSIST-04**: After a backup or empty start the engine is marked dirty (P-04)
  - Acceptance: the first rotation never displaces the last good generation with a corrupt one; `SnapshotCreate` right after such a start yields a snapshot that decodes to the in-memory state
- [x] **PERSIST-05**: Dump failures decay and log (P-05)
  - Acceptance: each failure — one ERROR entry with the underlying error (ENOSPC, EACCES); retries with a growing pause up to a cap; health behavior under a persistent dump failure is defined and documented
- [x] **PERSIST-06**: A dump cycle allocates at most two state sizes (P-06)
  - Acceptance: streaming encode; the memory model is documented
- [x] **PERSIST-07**: Equal version with different content is resolved deliberately (P-07)
  - Acceptance: either the random tail as in the Spec or a deterministic tie-break — decision recorded in an ADR, implemented, tested
- [x] **PERSIST-08**: Blocking engine calls carry a deadline (P-08)
  - Acceptance: a hung fsync cannot block writes, snapshots or shutdown indefinitely
- [x] **PERSIST-09**: Temp-file hygiene and the darwin fsync helper are fixed (P-09)
  - Acceptance: orphan temp files collected; new directories synced at the parent; `fsync_darwin.go` removed or handles EINTR/ENOTSUP
- [x] **PERSIST-10**: Records skipped at start are accounted (P-10)
  - Acceptance: records skipped due to a reduced value limit are counted and logged, not silently dropped and later deleted from disk
- [x] **PERSIST-11**: `persist.Engine` is split along its responsibilities (Q-03)
  - Acceptance: loader/composition root, store facade, dump scheduler, confirmation accounting and snapshot orchestration are separate; the `DumpStore` port returns the load outcome and failure cause (backs P-02, S-02)
- [x] **PERSIST-12**: `internal/app/persist` timers and `time.Now()` sit behind the same seam as the repl core (Q-06)
  - Acceptance: the dump↔replication-notify interaction is exercised by the simulator

### API

- [x] **APIS-01**: `wait_for_replication` reports honestly (A-01)
  - Acceptance: zero known peers → `(false, 0, 0)`; an `Ack` from a node unknown at write time does not yield `true`; a displaced version answers within 20 ms instead of idling 500 ms
- [x] **APIS-02**: `List` is paginated or streaming and its cost does not depend on value sizes (A-02)
- [x] **APIS-03**: Error mapping is total and path-free (A-03)
  - Acceptance: missing snapshot → `NotFound`; no dump → `FailedPrecondition`; no server paths in messages
- [x] **APIS-04**: The Watch contract is defined and honest (A-04)
  - Acceptance: local changes are announced only after dump confirmation; server shutdown ends streams with `Unavailable`, not `OK`; the contract is in the API docs
- [x] **APIS-05**: gRPC server settings are deliberate; health never returns to `SERVING` after shutdown begins (A-05)
- [x] **APIS-06**: Proto and generated code are public (A-06, 14.7)
  - Acceptance: proto and generated Go code outside `internal/`; a Go client example builds in CI as a separate module; the last-read-value file pattern is documented as consumer-library territory (14.7, accepted)
- [x] **APIS-07**: Snapshot load is a full rollback (S-01, 14.1)
  - Acceptance: keys absent from the snapshot are deleted; the result is broadcast to peers (with PROTO-08); `TestEngine_snapshotCreateLoadRestore` rewritten for the new behavior

### Observability & Configuration

- [x] **OBSV-01**: The node logs its lifecycle (O-01)
  - Acceptance: startup entry with version and effective config; one entry each for loaded dump generation, dump failures, gate opening, peer appear/disappear/restart, drift warnings; a fatal bind error at `SUPKV_LOG_LEVEL=error` produces an ERROR entry, not a silent exit 1
- [x] **OBSV-02**: Configuration is validated in one place (O-02, Q-07)
  - Acceptance: one validation function with ranges and relations (negative/zero values, `sync-chunk-bytes` overflow panic, `retransmit-interval=1ns` busy loop, `peer_timeout ≤ hello_interval`, `ack_timeout > ack_timeout_max`, positional args ignored); exit code 2 naming the parameter; table test; flag parsing extracted from `main.go`
- [x] **OBSV-03**: Metric definitions are honest and test-confirmed (O-03, 14.4)
  - Acceptance: the conflict counter counts only the displacement of a local unconfirmed record by another value; a failed send is not counted as sent; `process_resident_memory_bytes` reports current RSS; dump duration semantics defined; the drift counter matches its documented definition; every definition confirmed by a test
- [x] **OBSV-04**: Exit codes and shutdown are distinct and ordered (O-05)
  - Acceptance: distinct codes for usage error, startup failure and final-dump failure; a `Serve` error does not skip the final dump; the final dump is time-bounded; the signal handler is installed before dump load and bind; `--help` lists the five env-only variables
- [x] **OBSV-05**: Clock-drift warnings match the Spec (S-06)
  - Acceptance: WARN log + metric when a received version leads the system time by more than `drift_warn`; the runbook describes a condition that exists in the binary
- [x] **OBSV-06**: Spec-mandated log entries exist (S-07)
  - Acceptance: dump failure, start without usable dumps, foreign proto_version, duplicate node_id — each has its entry at the right level (with OBSV-01)
- [x] **OBSV-07**: The dependency direction is machine-enforced (Q-05)
  - Acceptance: a linter rule forbidding adapter imports from `app`/`port` and the observed cross-imports (`metrics`→`udprepl`, `grpcapi`→`persistfile`/`*persist.Engine`); any boundary change is a visible `.golangci.yml` diff
- [x] **OBSV-08**: `repl.go` and `main.go` are decomposed (Q-07)
  - Acceptance: socket, loop, mirror and codec are separate units in `udprepl`; `main.go` is wiring only; validation and shutdown are testable

### Performance

- [x] **PERF-01**: The write-latency model is documented for both profiles (PERF-01, PERF-02, 14.3)
  - Acceptance: docs state that under sustained write flow latency reaches `dump_max_wait` + dump time by design; the fast-response profile (`Debounce`/`MaxWait` tuning) is documented with its disk cost; the "~10 ms under 1000 writes/s" claim is tied to its measurement conditions
- [x] **PERF-02**: The 1000 writes/s burst over distinct keys is a tested bound (PERF-03)
  - Acceptance: the XPORT-02(3) scenario is a standing test; the docs describe what is and is not guaranteed at that rate
- [x] **PERF-03**: Disk-write amplification is measured on target-grade media or the gap is recorded (PERF-04, Spec open question #2)
- [x] **PERF-04**: The memory model is documented from stand measurements (PERF-05, with PERSIST-06)
- [x] **PERF-05**: Write latency with an absent peer and a growing unconfirmed set stays near the live-peer latency (PERF-06; same fix as XPORT-02, proven by a benchmark)
- [x] **PERF-06**: `make perf` runs the stand suite with thresholds; published numbers carry a hardware description

### Documentation & Release

- [ ] **DOCS-01**: All 52 documented mismatches (M-01…M-52) are closed by a doc or code fix (D-01)
- [ ] **DOCS-02**: `docs/CONFIGURATION.md` covers every flag and env var — defaults, units, ranges, relations, "must match on all nodes"; a test iterates registered flags against the table (D-02)
- [ ] **DOCS-03**: `docs/DEPLOYMENT.md` + systemd unit — two-board topology, bind/broadcast address choice, ports, `node_id`, paths and permissions, sizes, time sync, upgrade and compatibility, node replacement; the unit runs in CI and ships in the release archives (D-03)
- [ ] **DOCS-04**: Failure-behavior table (the Spec's 18 rows: symptom → behavior → recovery → metric → log line) and runbooks for dump failure/full disk, missing peer, growing unconfirmed set, write conflict, incompatible versions, corrupted dump, duplicate `node_id`; every row references a metric or log line that exists in the binary (D-04)
- [ ] **DOCS-05**: `docs/API.md` — per-RPC semantics and code tables; Watch contract; write-latency model; consistency caveats; key-prefix convention; `Unavailable` handling; client last-value cache pattern; how to obtain the proto; Go client example (D-05, with APIS-06)
- [ ] **DOCS-06**: `docs/FORMATS.md` — dump grammar with example, checksum definition, rejection rules, format-version policy; envelope and messages, CRC coverage, record-version layout, manual decoding; a test encodes a fixed set byte-identical to the example (D-06)
- [ ] **DOCS-07**: Metrics reference (type, labels, exact meaning, recommended alert) and a log-message catalog; a test cross-checks exported families and labels (D-07)
- [ ] **DOCS-08**: As-built architecture description and the spec-deviation table (with SPEC-01); `AGENTS.md` and `.claude/CLAUDE.md` stack sections match `go.mod` and `go list ./...` (D-08)
- [ ] **DOCS-09**: Every "proven" claim names its test; `TestOperationsDoc` verifies against the binary; example health waits do not match `NOT_SERVING`; the macOS quickstart pair works; a metric rename or default change without a doc update breaks CI (D-09)
- [ ] **DOCS-10**: `CONTRIBUTING.md`, `SECURITY.md`, `docs/ARCHITECTURE.md`, ADR index, simulator guide (seed replay, adding a scenario), release procedure, benchmark results with hardware; darwin demoted to development-only in README and release archives (14.10); dump and metrics setup in the deployment guide (14.3, 14.8); phase/decision numbers removed from proto comments and CHANGELOG; the ruleset check command in `docs/setup/branch-protection.md` works as written (D-10)

## v2 Requirements

Deferred, tracked for a future milestone (review section 13 open questions):

- **AUTH-01**: Datagram authentication (HMAC) — open question #1, explicitly deferred
- **GARB-01**: Tombstone garbage collection — open question #5
- **PFXD-01**: Per-prefix "delete beats concurrent update" — open question #4
- **PEER-01**: Explicit peer lists / multi-segment replication — open question #8
- **SNAP-01**: Mapping snapshot operations onto the consumer's proto — open question #7, consumer zone

## Out of Scope

| Feature | Reason |
|---------|--------|
| Revisiting the Spec's architecture (AP, LWW-map, HLC in UUIDv7, UDP broadcast, JSON dump, confirm-after-dump) | Review section 0: accepted and not re-opened |
| Conflict journal | Owner decision 14.4: not built; the counter is fixed instead (OBSV-03) |
| darwin as a production platform | Owner decision 14.10: development-only; limits documented, not fixed |
| `create_if_absent` unblocking a deleted key | Owner decision 14.2: deleted key blocks with `AlreadyExists`; no code change |
| Changing the dump policy (full dump, debounce/max-wait defaults) | Owner decision 14.3: current defaults stay; only documentation and profiles change |
| Client caching libraries shipped in sup-kv | S-11 consumer zone; owner decision 14.7 — pattern documented, code lives with the consumer |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SPEC-01 | Phase 1 | Complete |
| SPEC-02 | Phase 1 | Complete |
| TEST-01 | Phase 2 | Complete |
| TEST-02 | Phase 2 | Complete |
| TEST-03 | Phase 2 | Complete |
| TEST-04 | Phase 2 | Complete |
| TEST-05 | Phase 2 | Complete |
| TEST-06 | Phase 2 | Complete |
| TEST-07 | Phase 2 | Complete |
| TEST-08 | Phase 2 | Complete |
| TEST-09 | Phase 2 | Complete |
| TEST-10 | Phase 2 | Complete |
| TEST-11 | Phase 2 | Complete |
| TEST-12 | Phase 2 | Complete |
| TEST-13 | Phase 2 | Complete |
| TEST-14 | Phase 2 | Complete |
| TEST-15 | Phase 2 | Complete |
| TEST-16 | Phase 2 | Complete |
| XPORT-01 | Phase 3 | Complete |
| XPORT-02 | Phase 3 | Complete |
| XPORT-03 | Phase 3 | Complete |
| XPORT-04 | Phase 3 | Complete |
| XPORT-05 | Phase 3 | Complete |
| XPORT-06 | Phase 3 | Complete |
| XPORT-07 | Phase 3 | Complete |
| XPORT-08 | Phase 3 | Complete |
| XPORT-09 | Phase 3 | Complete |
| XPORT-10 | Phase 3 | Complete |
| XPORT-11 | Phase 3 | Complete |
| XPORT-12 | Phase 3 | Complete |
| XPORT-13 | Phase 3 | Complete |
| PROTO-01 | Phase 4 | Complete |
| PROTO-02 | Phase 4 | Complete |
| PROTO-03 | Phase 4 | Complete |
| PROTO-04 | Phase 4 | Complete |
| PROTO-05 | Phase 4 | Complete |
| PROTO-06 | Phase 4 | Complete |
| PROTO-07 | Phase 4 | Complete |
| PROTO-08 | Phase 4 | Complete |
| PROTO-09 | Phase 4 | Complete |
| PERSIST-01 | Phase 5 | Complete |
| PERSIST-02 | Phase 5 | Complete |
| PERSIST-03 | Phase 5 | Complete |
| PERSIST-04 | Phase 5 | Complete |
| PERSIST-05 | Phase 5 | Complete |
| PERSIST-06 | Phase 5 | Complete |
| PERSIST-07 | Phase 5 | Complete |
| PERSIST-08 | Phase 5 | Complete |
| PERSIST-09 | Phase 5 | Complete |
| PERSIST-10 | Phase 5 | Complete |
| PERSIST-11 | Phase 5 | Complete |
| PERSIST-12 | Phase 5 | Complete |
| APIS-01 | Phase 6 | Complete |
| APIS-02 | Phase 6 | Complete |
| APIS-03 | Phase 6 | Complete |
| APIS-04 | Phase 6 | Complete |
| APIS-05 | Phase 6 | Complete |
| APIS-06 | Phase 6 | Complete |
| APIS-07 | Phase 6 | Complete |
| OBSV-01 | Phase 7 | Complete |
| OBSV-02 | Phase 7 | Complete |
| OBSV-03 | Phase 7 | Complete |
| OBSV-04 | Phase 7 | Complete |
| OBSV-05 | Phase 7 | Complete |
| OBSV-06 | Phase 7 | Complete |
| OBSV-07 | Phase 7 | Complete |
| OBSV-08 | Phase 7 | Complete |
| PERF-01 | Phase 8 | Complete |
| PERF-02 | Phase 8 | Complete |
| PERF-03 | Phase 8 | Complete |
| PERF-04 | Phase 8 | Complete |
| PERF-05 | Phase 8 | Complete |
| PERF-06 | Phase 8 | Complete |
| DOCS-01 | Phase 9 | Pending |
| DOCS-02 | Phase 9 | Pending |
| DOCS-03 | Phase 9 | Pending |
| DOCS-04 | Phase 9 | Pending |
| DOCS-05 | Phase 9 | Pending |
| DOCS-06 | Phase 9 | Pending |
| DOCS-07 | Phase 9 | Pending |
| DOCS-08 | Phase 9 | Pending |
| DOCS-09 | Phase 9 | Pending |
| DOCS-10 | Phase 9 | Pending | |

**Coverage:**
- v1.1 requirements: 83 total
- Mapped to phases: 83 (100%)
- Unmapped: 0

**Phase map:** SPEC→1, TEST→2, XPORT→3, PROTO→4, PERSIST→5, APIS→6, OBSV→7, PERF→8, DOCS→9

---
*Requirements defined: 2026-11-19*
*Last updated: 2026-11-19 — traceability filled by v1.1 roadmap creation (9 phases)*
