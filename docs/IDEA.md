# sup-kv — Idea Document

*Seed for `/gsd:new-project --auto @docs/IDEA.md`. Self-contained; describes the product,
the decided design, and the candidate v1 scope. The design is accepted and must not be
re-litigated during planning — it comes from a reviewed architecture proposal; only
implementation-level questions are open.*

## One-liner

**sup-kv** ("supremum storage") is a leaderless replicated in-memory key-value store for
small clusters — two to a handful of nodes on one trusted LAN broadcast segment. Every
node accepts reads and writes locally and independently (AP); replicas exchange changes
over UDP and converge to the identical state — the pointwise supremum of all writes under
the record-version order (strong eventual consistency via a Last-Write-Wins-Element-Map
CRDT).

## Problem

Small HA clusters (e.g. two management boards in one appliance) need a shared, always
writable store for configuration and service state. Existing options fail this niche:

- Raft-based stores (etcd, Consul, rqlite, dqlite) need a majority: with two nodes the
  loss of one stops writes. CP, violates availability needs.
- Primary/replica with failover (Redis+Sentinel, LiteFS) needs third parties or leases,
  and one side cannot write during a split.
- AP stores that fit mechanically (KeyDB active-replica, Cassandra CL=ONE, Marmot,
  Corrosion) are either not Go, not embedded-friendly, require quorums under the hood,
  or store data in binary formats that cannot be inspected by a human.

sup-kv is the missing product: Go, one static binary, minimal dependencies, both (all)
nodes writable with no coordination, data in human-readable JSON files, and a design
built entirely from well-known components (LWW CRDT, HLC, ack+retransmit, anti-entropy
sync).

## Users and use cases

- **Service configuration** (units of writes per hour, KiB values, hundreds of keys):
  read at process start; defaults shipped with the build are seeded via create-if-absent.
- **Runtime configuration** (users, roles, sessions; units per minute, bursts):
  mutated by operators on any node; confirmed writes must never be lost.
- **Durable service state** (persistence of network-manager-like services;
  bursts up to ~1000 writes/s on hot keys): disk flush latency within the dump window
  is acceptable.
- Generic use: any small set of nodes on a trusted segment needing a tiny, always
  writable, inspectable KV store with snapshots.

## Core design (decided)

### Data model

- Keys: strings. Recommended convention: per-service prefix with schema version
  (`netmgr/v2/...`) — enforced by docs, not by the store.
- Per key the store keeps: `RecordVersion`, `Data` (bytes, ≤ 60 KiB), `Deleted` (bool,
  tombstone), `Checksum` (CRC32 over key+version+data+deleted).
- Storage: in-memory Go map under RWMutex; whole state fits in RAM by design
  (thousands of keys, tens of thousands at most).
- The map is a **LWW-Element-Map CRDT**: merge per key by greater `RecordVersion`.
  Merge is commutative, associative, idempotent — delivery order, duplicates and
  retransmissions cannot affect the outcome. Tombstones are kept indefinitely
  (deletes are rare).

### Record versions

- Hybrid Logical Clock (Kulkarni et al., 2014): node state `(l, c)`; `Tick` on local
  write/delete; `Update` on receiving any foreign version; initialized at startup from
  the dump's max `l`; counter overflow bumps `l` by 1 ms.
- Packed into **UUID v7** (RFC 9562): 48-bit `unix_ts_ms` = `l`; 12-bit `rand_a` = `c`
  (counter, explicitly allowed); 2-bit `var`; 62-bit `rand_b` = 16-bit `NodeID` in the
  high bits + random filler.
- Byte-wise UUID comparison equals `(l, c, NodeID)` order; unique per write by
  construction (one node increments `c`, nodes differ in `NodeID`). No time
  synchronization protocol is needed: correctness never depends on wall clocks; clock
  skew only affects which simultaneous write wins, deterministically and identically
  on all replicas.

### Replication protocol (UDP)

- One protobuf `Envelope` per datagram: `proto_version`, `node_id`, `run_id` (fresh
  UUID per process start), `crc32`, one message body.
- `Hello` — broadcast every `hello_interval`; announces `node_id`+`run_id`. A new
  `run_id` from a known node triggers full sync toward it (it restarted). Duplicate
  `node_id` is a configuration error: logged, datagrams dropped.
- `DataReplication` — full record; broadcast on every local change and on retransmit;
  unicast as a reply to `DataRequest` and when the receiver has a newer version.
- `Ack` — unicast, per (key, version), sent after the record is applied to memory
  (not waiting for the dump).
- `DataRequest` — unicast "send me this key"; no reply if the key is unknown.
- `DataSync` — chunked full key→version listing (sorted, chunked to
  `sync_chunk_bytes`, chunks self-sufficient); sent by every node on timer
  (`sync_interval`), on neighbor restart, and at startup. Receivers compare per key and
  answer with `DataRequest` (missing/stale) or unicast `DataReplication` (newer);
  responses are rate-limited (`sync_response_rate`) to survive empty-node recovery.
- Retransmission: per-peer unacked tracking of (key, version); exponential backoff
  `ack_timeout` → `ack_timeout_max`; suspended while no peer is known; a record leaves
  the set on `Ack` or when superseded by a newer local or remote version.
- **N nodes from day one**: peer table is a map `NodeID → {addr, run_id, last hello}`;
  per-peer ack tracking; a peer is "known" after its first `Hello`, considered gone
  after `peer_timeout`. Broadcast discovery keeps configuration to a single segment.
- Convergence bounds: normal operation one RTT; datagram loss ≤ `ack_timeout_max`;
  after a partition heals ≤ 2 × `sync_interval` + response time.

### Persistence

- Dump = JSON file with the full map contents: format version + records
  (`Key`, `RecordVersion`, `Data` base64, `Deleted`, `Checksum`). Human-readable and
  greppable.
- Atomic dump: write temp file, fsync, rename; previous generation kept as `.prev`;
  fallback chain main → `.prev` → empty start (records with bad `Checksum` are
  discarded; a node that starts empty recovers from peers via full sync).
- Debounced dumps (`dump_debounce`); a client write returns `OK` only after a dump
  containing it completed → **confirmed writes are always on disk**. Dump failure
  returns `Unavailable`, the write stays in memory and replicates.
- Snapshots: named copies of the dump on API request; restore replays the snapshot as
  ordinary writes/deletes with fresh versions, so it replicates and wins by normal LWW.

### Node lifecycle

- Startup: load dump (checksums, `.prev` fallback), init HLC from dump max, new
  `RunID`, open socket, first `Hello`, timers. Reads served immediately; writes
  rejected with `Unavailable` until the first full sync round arrives or
  `startup_gate` elapses (closes the window of unseen peer writes).

### API (gRPC)

- `Get`, `Set` (create/update; `create-if-absent` variant for build-time defaults),
  `Delete`, `Watch` (stream of change notifications — local and replicated alike),
  snapshot create/load.
- Optional `wait_for_replication` on writes: additionally wait for `Ack`s from known
  peers within a timeout, report whether received; never changes the write outcome.
- Errors: `OK` / `NotFound` (missing or deleted) / `AlreadyExists` /
  `InvalidArgument` (value > `value_max_bytes`) / `Unavailable` (startup gate, dump
  failure).

### Observability

- Metrics: unacked records count and oldest age, dump duration/errors, discarded
  datagrams (bad crc / own broadcast / bad proto version), merge conflicts,
  clock-drift warnings (`drift_warn`).
- Conflict journal: lost versions logged on merge conflicts.
- Structured logging; every dropped datagram class is distinguishable.

### Parameters (defaults, tunable at launch)

`node_id`, `bind_addr`, `broadcast_addr`, `dump_path` (required);
`value_max_bytes` 61440; `sync_chunk_bytes` 1400; `hello_interval` 1s;
`peer_timeout` 5s; `ack_timeout` 200ms; `ack_timeout_max` 5s;
`retransmit_interval` 100ms; `sync_interval` 30s; `sync_response_rate` 500/s (sync responses spaced ⌊1s/rate⌋ — 2ms at the default, integer nanoseconds, first response immediate);
`startup_gate` 1s; `dump_debounce` 50ms; `dump_max_wait` 1s; `drift_warn` 60s.

### Trust model (accepted limitation)

The replication segment is trusted: CRC32 guards integrity only; there is no
authentication or encryption of datagrams in v1. Authenticating datagrams (HMAC with a
shared key) is a documented future work item, not a v1 goal.

## Product principles

- Minimal dependencies: gRPC + protobuf; everything else (HLC, UUIDv7 packing, LWW map,
  UDP reliability layer) implemented in-repo — each is tens of lines against a known
  algorithm.
- Correctness from proven components; no invented consensus.
- Deterministic simulation testing is a first-class feature, not an afterthought.
- On-disk state is human-readable; every artifact (dump, logs, conflict journal) can be
  inspected with standard tools.

## Candidate v1 requirements

- **CORE**: in-memory LWW map; Get/Set/Delete/CreateIfAbsent; tombstones; value size
  limit; conflict journal.
- **VERS**: HLC clock; UUIDv7 packing; byte-order comparison; monotonic under clock
  jumps (backwards and forwards), drift warning.
- **REPL**: UDP envelope; Hello discovery; DataReplication/Ack with retransmission
  backoff; DataRequest; chunked DataSync full reconciliation with rate-limited
  responses; N-peer table and per-peer ack tracking.
- **PERSIST**: JSON dump, atomic write with `.prev` fallback, debounced dumps,
  write-confirmed-after-dump guarantee; snapshots create/load.
- **API**: gRPC service + protobuf definitions; Watch stream; startup gate;
  `wait_for_replication`.
- **OBS**: metrics endpoint; structured logs; discarded-datagram counters.
- **TEST**: CRDT property tests (merge laws); HLC property tests (monotonicity,
  post-receive ordering); deterministic multi-node simulation with a lossy/duplicating/
  reordering/partitioning virtual network; failure-scenario suite (see below);
  benchmarks (dump at 10k keys, replication throughput).
- **TOOL**: single static binary, flags/env config, `--version`, systemd-friendly.

## Failure scenarios to be covered by automated tests

Datagram loss; Ack loss; duplicates/reordering/own-broadcast echo; partition for
seconds and for days (both sides keep serving writes, deterministic winner, conflict
journal entries); one-way connectivity; node restart (RunID change triggers resync);
empty-disk node recovery at `sync_response_rate`; both-nodes-down restart; corrupted
dump records; wall-clock jumps on a live node and between restarts; a node's clock far
ahead; ~1000 writes/s bursts; duplicate `node_id` misconfiguration; protocol version
mismatch between nodes.

## Out of scope for v1

- Authentication/encryption of replication traffic (HMAC — future work).
- Multi-key transactions; conditional writes beyond create-if-absent.
- Tombstone garbage collection (kept forever; volumes are tiny).
- Per-prefix "delete beats concurrent update" policy.
- WAN / multi-segment replication / explicit peer lists (one broadcast segment only).
- Not-in-memory storage; values > 60 KiB.
- Client caching libraries (documented pattern; shipped later).

## Suggested milestone shape (input for the roadmap, not binding)

1. **Engine**: `RecordVersion`/HLC + UUIDv7 packing + LWW map + CRDT/HLC property tests.
2. **Persistence**: dump + `.prev` + debounce + startup load; snapshots.
3. **Replication**: UDP protocol + peer table + retransmission + full sync +
   deterministic 2..N-node simulator with a chaotic virtual network.
4. **API**: gRPC service, startup gate, Watch, `wait_for_replication`, `sup-kv` binary.
5. **Hardening**: failure-scenario suite, metrics, conflict journal, benchmarks.
6. **v1.0**: README quickstart, ops docs, examples, release tag.

## Success criteria for v1.0

- Every failure scenario above is an automated simulation test that demonstrates
  convergence within the documented bounds.
- Property tests pass: merge is commutative/associative/idempotent; HLC is monotonic
  under adversarial clock input; versions are unique per write.
- Quickstart works: two nodes on one localhost broadcast socket, writes visible on
  both within an RTT, kill −9 one, restart, state converges without operator action.
- `go test`, `go vet`, linters green in CI on Linux and macOS, on amd64 and arm64.

## References

- Kulkarni et al., *Logical Physical Clocks and Consistent Snapshots in Globally
  Distributed Databases*, 2014 (HLC).
- RFC 9562 (UUIDv7; counter usage in `rand_a`).
- Shapiro et al., *Conflict-free Replicated Data Types* (LWW-Element-Map); crdt.tech.
- Design references: Corrosion (Fly.io), Marmot, Alertmanager cluster mode,
  CockroachDB `pkg/util/hlc`.
