# Requirements: sup-kv

**Defined:** 2026-10-31
**Core Value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Core Store

- [x] **CORE-01**: User can Get a key; a missing or deleted (tombstoned) key returns NotFound
- [x] **CORE-02**: User can Set a key's value (create or update), replacing any prior value under LWW
- [x] **CORE-03**: User can create a key only if absent (create-if-absent) for seeding build-time defaults; an existing key returns AlreadyExists
- [x] **CORE-04**: User can Delete a key; the tombstone is written with a fresh version and replicates like any write
- [x] **CORE-05**: Values larger than `value_max_bytes` (default 60 KiB) are rejected with InvalidArgument
- [x] **CORE-06**: Merge follows LWW-Element-Map laws — per key the greater RecordVersion wins; merge is commutative, associative and idempotent, so delivery order, duplicates and retransmissions cannot affect the outcome
- [x] **CORE-07**: Every version displaced by a merge conflict is recorded in the conflict journal (the merge exposes the loser, not just the winner)

### Versioning

- [x] **VERS-01**: RecordVersion is an HLC state packed into UUID v7; byte-wise UUID comparison equals `(l, c, NodeID)` order
- [x] **VERS-02**: Versions are unique per write by construction (one node increments the counter; nodes differ in NodeID) — property-tested
- [x] **VERS-03**: The HLC stays monotonic under backward and forward wall-clock jumps (Tick on local write, Update on receive; counter overflow bumps `l` by 1 ms) — property-tested against adversarial clock input
- [x] **VERS-04**: The HLC initializes at startup from the dump's max `l`, not from the wall clock; restart seeds the dump's full watermark (max `l` plus max counter at that `l` via `clock.NewClock`'s `initialL`/`initialC` parameters — the clock is injected into `storage.NewStore` through the `storage.Clock` interface), so every post-restart version strictly dominates every dumped version — strict-dominance extension delivered by plan 02-04 (WR-01 closed)
- [x] **VERS-05**: Clock drift beyond `drift_warn` is logged as a warning

### Replication

- [x] **REPL-01**: Nodes discover peers via broadcast Hello (`node_id` + `run_id`) with no configuration beyond `broadcast_addr`
- [x] **REPL-02**: Every local change is broadcast as DataReplication and tracked per-peer (key, version) until Ack or supersession; retransmission uses exponential backoff `ack_timeout` → `ack_timeout_max`, suspended while no peer is known
- [x] **REPL-03**: A receiver Acks each (key, version) after applying the record to memory (not waiting for the dump)
- [x] **REPL-04**: A node can request a missing/stale key via unicast DataRequest; no reply if the key is unknown
- [x] **REPL-05**: Every node sends chunked DataSync (sorted key→version listing, chunks self-sufficient, `sync_chunk_bytes`) on the `sync_interval` timer, at startup, and on neighbor restart; receivers answer with DataRequest (missing/stale) or unicast DataReplication (newer), rate-limited by `sync_response_rate`
- [x] **REPL-06**: The peer table handles N nodes from day one — a peer is known after its first Hello and considered gone after `peer_timeout`; ack tracking is per-peer
- [x] **REPL-07**: A Hello with a new `run_id` from a known node (restart) triggers full sync toward it
- [x] **REPL-08**: Datagrams with a duplicate `node_id` are dropped and logged as a configuration error
- [x] **REPL-09**: Own broadcast echo and datagrams with bad CRC or bad proto version are dropped, with each discard class distinguishable
- [x] **REPL-10**: Documented convergence bounds hold and are proven in simulation: normal operation one RTT; datagram loss ≤ `ack_timeout_max`; after a partition heals ≤ 2 × `sync_interval` + response time

### Persistence

- [x] **PERS-01**: The dump is a human-readable JSON file of the full map (format version + records with base64 Data, RecordVersion, Deleted, Checksum) — greppable with standard tools
- [x] **PERS-02**: Dumps are atomic (temp file in the target directory, fsync, rename, parent-directory fsync) with the previous generation kept as `.prev`; fallback chain main → `.prev` → empty start; records with a bad Checksum are discarded
- [x] **PERS-03**: Dumps are debounced (`dump_debounce`) with a bounded max wait, so write bursts cannot starve dump completion
- [x] **PERS-04**: A write returns OK only after a dump containing it completed (version watermark) — confirmed writes are always on disk; dump failure returns Unavailable while the write stays in memory and replicates
- [x] **PERS-05**: User can create named snapshots (copies of the dump) and load them; restore replays the snapshot as ordinary writes/deletes with fresh versions so it replicates and wins by normal LWW

### API

- [x] **API-01**: gRPC service with protobuf definitions exposes Get/Set (incl. create-if-absent)/Delete with errors OK / NotFound / AlreadyExists / InvalidArgument / Unavailable
- [x] **API-02**: User can List keys under a prefix, returned in sorted order (prefix-range RPC)
- [x] **API-03**: User can Watch a stream of change notifications covering local and replicated changes alike (engine event taps designed in from the start)
- [x] **API-04**: Writes are rejected with Unavailable during the startup gate (until the first full sync round arrives or `startup_gate` elapses); reads are served immediately
- [ ] **API-05**: User can request `wait_for_replication` on a write: the server waits for Acks from known peers within a timeout and reports whether they were received; it never changes the write outcome
- [x] **API-06**: The gRPC standard health checking endpoint is served and reflects startup-gate state (NOT_SERVING until gated open)
- [x] **API-07**: Snapshot create/load is exposed through the API

### Observability

- [ ] **OBS-01**: A metrics endpoint exposes unacked-records count and oldest age, dump duration and errors, merge conflicts, and clock-drift warnings
- [ ] **OBS-02**: Discarded datagrams are counted by class (bad CRC / own broadcast / bad proto version)
- [ ] **OBS-03**: Logging is structured and every dropped-datagram class is distinguishable in logs

### Testing

- [x] **TEST-01**: CRDT property tests prove merge is commutative, associative and idempotent
- [x] **TEST-02**: HLC property tests prove monotonicity under adversarial clock input, post-receive ordering and version uniqueness; counter-reset and restart-from-dump-max cases included
- [x] **TEST-03**: A deterministic multi-node simulation runs 2..N nodes over a virtual network that loses, duplicates and reorders datagrams and simulates partitions, with seeded runs
- [ ] **TEST-04**: Every documented failure scenario is an automated test demonstrating convergence within the documented bounds: datagram loss; Ack loss; duplicates/reordering/own-broadcast echo; partition for seconds and for days (both sides serving writes, deterministic winner, conflict-journal entries); one-way connectivity; node restart (RunID change triggers resync); empty-disk recovery at `sync_response_rate`; both-nodes-down restart; corrupted dump records; wall-clock jumps live and between restarts; a node's clock far ahead; ~1000 writes/s bursts; duplicate `node_id`; protocol version mismatch
- [ ] **TEST-05**: Benchmarks cover dump at 10k keys and replication throughput (burst writes), with bounded confirmation latency under bursts

### Tooling

- [ ] **TOOL-01**: sup-kv ships as a single static binary with flags/env config, `--version`, and systemd-friendly behavior

### Quality

- [x] **QUAL-01**: Strict golangci-lint configuration (all applicable linters enabled; no suppressions without written justification) and go-ultimate conventions govern every package from the first feature commit; a red gate blocks merge

### CI

- [x] **CI-01**: GitHub Actions runs the quality gates (strict lint, `go vet`, tests with race detector) on every push and PR, on Linux and macOS, amd64 and arm64
- [ ] **CI-02**: Dependabot keeps dependencies current (updates and alerts) and a scheduled `govulncheck` job runs against the Go vulnerability database; no open critical vulnerabilities at release
- [ ] **CI-03**: Development is PR-only — branch protection on `master`, direct pushes forbidden, every change (code and docs) lands through a pull request

### Documentation

- [ ] **DOCS-01**: README quickstart works as written: two nodes on one localhost broadcast socket, writes visible on both within an RTT, kill −9 one node, restart, state converges without operator action
- [ ] **DOCS-02**: Ops docs state the trust model prominently (no auth/encryption in v1, HMAC is future work), the value-size envelope, local-read semantics (read-your-writes on the same node only), a "not a lock/election service" warning, and a clock-drift runbook
- [ ] **DOCS-03**: API usage examples are provided

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### API

- **API-08**: Watch prefix filtering (avoid whole-store chatter for multi-consumer deployments)
- **API-09**: Admin CLI subcommands (`status`, `dump`) or documented grpcurl recipes

### Replication

- **REPL-11**: DataSync unicast fast path for large divergences (skip per-key DataRequest when a whole range is stale) — only if empty-node recovery proves too slow at real volumes

### Security

- **SEC-01**: HMAC authentication of replication datagrams with a shared key (documented future work; trigger is first deployment outside a physically trusted segment)

### Clients

- **CLNT-01**: Client caching libraries (documented pattern only until the API surface stabilizes)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Multi-key transactions | Distributed transactions need coordination (2PC/Raft) — forfeits the no-quorum availability that defines the product; per-key atomicity only |
| TTL / lease keys | Expiry needs a reaper and tombstone-with-expiry semantics interacting with LWW merge; ephemeral registration is service-discovery territory, not durable config |
| Conditional writes beyond create-if-absent (CAS) | CAS on a specific version is a linearizable operation — breaks AP semantics; create-if-absent is the one safe variant |
| Tombstone garbage collection | Deletes are rare and volumes tiny; compaction-vs-anti-entropy is the hardest code path in comparable systems (Cassandra); tombstones-forever is safe at this scale |
| Per-prefix "delete beats concurrent update" policy | Plain per-key LWW is sufficient for v1 consumers |
| WAN / multi-segment replication / explicit peer lists | NAT traversal, relay and cross-trust-boundary auth are a different product; the niche is one trusted broadcast segment |
| Not-in-memory storage tiers | The envelope is tens of thousands of ≤ 60 KiB values in RAM by design |
| Values > 60 KiB | Envelope and dump sizing assume small values; larger blobs are a different product |
| Distributed locks / leader election on sup-kv | Locks need linearizability and fencing — impossible on an AP LWW store; documented as not-a-lock-service instead |
| Read-consistency options (quorum/linearizable reads) | Quorums are antithetical to the AP design; local-read semantics are documented instead |
| Merkle-tree anti-entropy | Pays off at millions of keys; at ~10k keys the chunked sorted listing is smaller and human-debuggable |
| Replication traffic compression | KiB values on a LAN; protobuf envelopes are already small; revisit only if benchmarks show segment saturation |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CI-01 | Phase 1 | Complete |
| CI-02 | Phase 1 | Partial — version updates + scheduled govulncheck delivered and green; alerts half deferred (GHAS-gated on private tier), ratified accepted override 2026-10-31; unlocks at GHAS license or the deferred public flip |
| CI-03 | Phase 1 | Partial — PR-only practiced since plan 01-01 (PRs #1-#10); server-side branch protection deferred 2026-10-31 (org plan 403, decision `defer`, ratified 2026-10-31 as accepted override (no org plan upgrade, no public flip); re-apply path in docs/setup/branch-protection.md) |
| QUAL-01 | Phase 1 | Complete |
| CORE-01 | Phase 2 | Complete |
| CORE-02 | Phase 2 | Complete |
| CORE-03 | Phase 2 | Complete |
| CORE-04 | Phase 2 | Complete |
| CORE-05 | Phase 2 | Complete |
| CORE-06 | Phase 2 | Complete |
| CORE-07 | Phase 2 | Complete |
| VERS-01 | Phase 2 | Complete |
| VERS-02 | Phase 2 | Complete |
| VERS-03 | Phase 2 | Complete |
| VERS-04 | Phase 2 | Complete |
| VERS-05 | Phase 2 | Complete |
| TEST-01 | Phase 2 | Complete |
| TEST-02 | Phase 2 | Complete |
| PERS-01 | Phase 3 | Complete |
| PERS-02 | Phase 3 | Complete |
| PERS-03 | Phase 3 | Complete |
| PERS-04 | Phase 3 | Complete |
| PERS-05 | Phase 3 | Complete |
| REPL-01 | Phase 4 | Complete |
| REPL-02 | Phase 4 | Complete |
| REPL-03 | Phase 4 | Complete |
| REPL-04 | Phase 4 | Complete |
| REPL-05 | Phase 4 | Complete |
| REPL-06 | Phase 4 | Complete |
| REPL-07 | Phase 4 | Complete |
| REPL-08 | Phase 4 | Complete |
| REPL-09 | Phase 4 | Complete |
| REPL-10 | Phase 4 | Complete |
| TEST-03 | Phase 4 | Complete |
| API-01 | Phase 5 | Complete |
| API-02 | Phase 5 | Complete |
| API-03 | Phase 5 | Complete |
| API-04 | Phase 5 | Complete |
| API-05 | Phase 5 | Pending |
| API-06 | Phase 5 | Complete |
| API-07 | Phase 5 | Complete |
| TOOL-01 | Phase 5 | Pending |
| OBS-01 | Phase 6 | Pending |
| OBS-02 | Phase 6 | Pending |
| OBS-03 | Phase 6 | Pending |
| TEST-04 | Phase 6 | Pending |
| TEST-05 | Phase 6 | Pending |
| DOCS-01 | Phase 7 | Pending |
| DOCS-02 | Phase 7 | Pending |
| DOCS-03 | Phase 7 | Pending |

**Coverage:**
- v1 requirements: 50 total
- Mapped to phases: 50
- Unmapped: 0

Note: the previously stated total of 41 did not match the enumerated requirement IDs; the actual v1 count is 50 (7 CORE + 5 VERS + 10 REPL + 5 PERS + 7 API + 3 OBS + 5 TEST + 1 TOOL + 1 QUAL + 3 CI + 3 DOCS). All 50 are mapped.

---
*Requirements defined: 2026-10-31*
*Last updated: 2026-10-31 after roadmap creation*
