# sup-kv

## What This Is

**sup-kv** ("supremum storage") is a leaderless replicated in-memory key-value store for small clusters — two to a handful of nodes on one trusted LAN broadcast segment. Every node accepts reads and writes locally and independently (AP); replicas exchange changes over UDP and converge to identical state — the pointwise supremum of all writes under the record-version order (strong eventual consistency via a Last-Write-Wins-Element-Map CRDT). It ships as one static Go binary with minimal dependencies and human-readable JSON persistence.

For service builders on small HA clusters (e.g. two management boards in one appliance) who need a shared, always-writable store for configuration and durable service state where quorum-based stores (etcd, Consul) cannot survive the loss of a single node.

## Core Value

Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.

## Requirements

### Validated

- ✓ **Quality gates** — Phase 1 (strict golangci-lint + go-ultimate from the first commit; lint/vet/race on every push)
- ✓ **CI/process** — Phase 1 (CI matrix Linux/macOS × amd64/arm64 green live; Dependabot version updates active — nothing updatable at v2.15.0, upstream-latest; scheduled govulncheck fired on schedule; PR-only practiced. Ratified Partial: CI-02 alerts half GHAS-gated, CI-03 server-side ruleset org-plan-gated — recorded accepted overrides)
- ✓ **Core store** — Phase 2 (LWW-Element-Map in `internal/app/storage`; Get/Set/Delete/SetIfAbsent; tombstones; 60 KiB limit; conflict journal via Merge-returned losers)
- ✓ **Versioning** — Phase 2 (HLC in `internal/app/clock` with UUIDv7 packing in `internal/app/version`; byte-order comparison = `(l, c, NodeID)`; monotonic under adversarial clocks incl. restart watermark seeding and 48-bit ceiling saturation; drift signal both directions)
- ✓ **Property tests** — Phase 2 (merge laws incl. tombstones; HLC monotonicity, uniqueness, restart dominance — seeded, replayable, 100% statement coverage)
- ✓ **Persistence** — Phase 3 (JSON dump, atomic write with `.prev` fallback, debounced dumps, write-confirmed-after-dump guarantee; snapshots create/load)
- ✓ **Replication** — Phase 4 (protobuf UDP envelope; Hello discovery; DataReplication/Ack with retransmission backoff; DataRequest replies; chunked DataSync full reconciliation with rate-limited responses; N-peer table with per-peer ack tracking; restart resync on run_id change)
- ✓ **Deterministic simulator** — Phase 4 (simnet: goroutine-free discrete-event virtual network with loss/duplication/reordering/partition; convergence bounds proven on the virtual clock; chaos suite seeded and replayable via `SUPKV_SIM_SEED`)

### Active

All Active requirements are hypotheses until shipped and validated.

- [ ] **API**: gRPC service + protobuf definitions; Watch stream; startup gate; `wait_for_replication`
- [ ] **Observability**: metrics endpoint; structured logs; discarded-datagram counters
- [ ] **Testing (remaining)**: failure-scenario suite for the documented scenarios; benchmarks (dump at 10k keys, replication throughput)
- [ ] **Tooling**: single static binary, flags/env config, `--version`, systemd-friendly

### Out of Scope

- Authentication/encryption of replication traffic (HMAC) — documented future work, not a v1 goal; the replication segment is trusted
- Multi-key transactions — store is a KV map, not a database; atomicity is per key
- Conditional writes beyond create-if-absent — no identified use case for v1
- Tombstone garbage collection — deletes are rare, volumes are tiny; keeping tombstones forever is simpler and safe
- Per-prefix "delete beats concurrent update" policy — plain LWW per key is sufficient for v1 consumers
- WAN / multi-segment replication / explicit peer lists — product niche is a single trusted broadcast segment
- Not-in-memory storage — the whole state fits in RAM by design
- Values > 60 KiB — envelope/dump sizing assumes small values; larger blobs are a different product
- Client caching libraries — documented pattern only; shipped later

## Context

**The design is decided.** It comes from a reviewed architecture proposal (`docs/IDEA.md` — the authoritative, self-contained design document). Do not re-litigate it during planning; only implementation-level questions are open.

Decided design, in brief (full detail and parameter defaults in `docs/IDEA.md`):

- **Data model**: per key the store keeps `RecordVersion`, `Data` (bytes, ≤ 60 KiB), `Deleted` (tombstone), `Checksum` (CRC32 over key+version+data+deleted). In-memory Go map under RWMutex; thousands to tens of thousands of keys. The map is a LWW-Element-Map CRDT: merge per key by greater `RecordVersion`; merge is commutative, associative, idempotent — delivery order, duplicates and retransmissions cannot affect the outcome. Tombstones kept indefinitely.
- **Record versions**: Hybrid Logical Clock (Kulkarni et al., 2014) — node state `(l, c)`; `Tick` on local write, `Update` on receiving any foreign version; initialized from the dump's max `l` at startup; counter overflow bumps `l` by 1 ms. Packed into UUID v7 (RFC 9562): 48-bit `unix_ts_ms` = `l`, 12-bit `rand_a` = counter, 62-bit `rand_b` = 16-bit `NodeID` in the high bits + random filler. Byte-wise UUID comparison equals `(l, c, NodeID)` order; unique per write by construction. Correctness never depends on wall-clock synchronization; skew only affects which simultaneous write wins — deterministically and identically on all replicas.
- **Replication (UDP)**: one protobuf `Envelope` per datagram (`proto_version`, `node_id`, `run_id`, `crc32`, one message body). `Hello` broadcast on interval (new `run_id` from a known node ⇒ it restarted ⇒ full sync toward it; duplicate `node_id` is a config error — logged, dropped). `DataReplication` full record, broadcast on local change/retransmit, unicast as `DataRequest` reply or when receiver is stale. `Ack` unicast per (key, version) after applying to memory. `DataRequest` unicast "send me this key". `DataSync` chunked sorted key→version listing on timer, neighbor restart, and startup; receivers answer with `DataRequest`/unicast `DataReplication`, rate-limited. Retransmission: per-peer unacked tracking, exponential backoff `ack_timeout` → `ack_timeout_max`, suspended with no known peers; records leave the set on `Ack` or supersession. N nodes from day one: peer table `NodeID → {addr, run_id, last hello}`; peer gone after `peer_timeout`. Convergence bounds: normal one RTT; datagram loss ≤ `ack_timeout_max`; partition heal ≤ 2 × `sync_interval` + response time.
- **Persistence**: dump = human-readable JSON (format version + records with base64 `Data`); atomic write (temp file, fsync, rename) with previous generation kept as `.prev`; fallback chain main → `.prev` → empty start (bad-checksum records discarded; empty node recovers from peers). Debounced dumps; a client write returns `OK` only after a dump containing it completed — **confirmed writes are always on disk**; dump failure returns `Unavailable` (write stays in memory and replicates). Snapshots: named copies of the dump; restore replays the snapshot as ordinary writes/deletes with fresh versions so it replicates and wins by normal LWW.
- **Node lifecycle**: startup loads dump (checksums, `.prev` fallback), inits HLC from dump max, opens socket, sends first `Hello`; reads served immediately; writes rejected with `Unavailable` until first full sync round or `startup_gate` elapses.
- **API (gRPC)**: `Get`, `Set` (incl. create-if-absent), `Delete`, `Watch` (stream of local + replicated change notifications), snapshot create/load. Optional `wait_for_replication` on writes: wait for peer `Ack`s within a timeout and report; never changes the write outcome. Errors: `OK` / `NotFound` / `AlreadyExists` / `InvalidArgument` (value > limit) / `Unavailable` (startup gate, dump failure).
- **Observability**: metrics (unacked count and oldest age, dump duration/errors, discarded datagrams by class, merge conflicts, clock-drift warnings); conflict journal logging lost versions; structured logging with every dropped-datagram class distinguishable.
- **Parameters** (defaults, tunable at launch): `node_id`, `bind_addr`, `broadcast_addr`, `dump_path` (required); `value_max_bytes` 61440; `sync_chunk_bytes` 1400; `hello_interval` 1s; `peer_timeout` 5s; `ack_timeout` 200ms; `ack_timeout_max` 5s; `retransmit_interval` 100ms; `sync_interval` 30s; `sync_response_rate` 500/s; `startup_gate` 1s; `dump_debounce` 50ms; `drift_warn` 60s.
- **Trust model (accepted limitation)**: replication segment is trusted; CRC32 guards integrity only; no auth/encryption of datagrams in v1.

Target workloads: service configuration (writes/hour, KiB values, hundreds of keys, build-time defaults via create-if-absent), runtime configuration (writes/minute in bursts; confirmed writes must never be lost), durable service state (bursts up to ~1000 writes/s on hot keys; dump latency within the debounce window acceptable).

**Failure scenarios that must be covered by automated tests**: datagram loss; Ack loss; duplicates/reordering/own-broadcast echo; partition for seconds and for days (both sides serving writes, deterministic winner, conflict journal entries); one-way connectivity; node restart (RunID change triggers resync); empty-disk node recovery at `sync_response_rate`; both-nodes-down restart; corrupted dump records; wall-clock jumps on a live node and between restarts; a node's clock far ahead; ~1000 writes/s bursts; duplicate `node_id` misconfiguration; protocol version mismatch between nodes.

**v1.0 success criteria**: every failure scenario above is an automated simulation test demonstrating convergence within documented bounds; property tests pass (merge laws, HLC monotonicity under adversarial clock input, version uniqueness per write); quickstart works (two nodes on one localhost broadcast socket, writes visible on both within an RTT, kill −9 and restart converges without operator action); quality gates green in CI on Linux/macOS × amd64/arm64 with Dependabot and `govulncheck` active.

**Suggested milestone shape** (input for the roadmap, not binding): 0. CI bootstrap → 1. Engine (RecordVersion/HLC + UUIDv7 + LWW map + property tests) → 2. Persistence (dump + snapshots) → 3. Replication (UDP protocol + deterministic N-node simulator with chaotic virtual network) → 4. API (gRPC, startup gate, Watch, `wait_for_replication`, binary) → 5. Hardening (failure-scenario suite, metrics, conflict journal, benchmarks) → 6. v1.0 (docs, examples, release tag).

**References**: Kulkarni et al., *Logical Physical Clocks…*, 2014 (HLC); RFC 9562 (UUIDv7, counter in `rand_a`); Shapiro et al., *Conflict-free Replicated Data Types* (LWW-Element-Map; crdt.tech); design references: Corrosion (Fly.io), Marmot, Alertmanager cluster mode, CockroachDB `pkg/util/hlc`.

## Constraints

- **Tech stack**: Go; single static binary; minimal dependencies — gRPC + protobuf only; HLC, UUIDv7 packing, LWW map, UDP reliability layer all implemented in-repo (each is tens of lines against a known algorithm) — product principle: no invented consensus, correctness from proven components
- **Quality**: strict golangci-lint configuration (all applicable linters enabled; no suppressions without written justification) and the `go-ultimate` skill conventions (github.com/Djarvur/go-ultimate) govern all implementation from the first feature commit; enforced in CI, red gate blocks merge
- **Process**: PR-only development — branch protection on `master` (server-side ruleset deferred 2026-10-31 — ratified accepted override, org-plan gated; one-command re-apply in docs/setup/branch-protection.md), direct pushes forbidden, every change lands through a PR from a feature branch; CI on every push and PR
- **Scale envelope**: whole state in RAM (thousands to tens of thousands of keys); values ≤ 60 KiB; one UDP broadcast segment; trusted LAN
- **Correctness**: convergence never depends on wall-clock synchronization; determinism must be proven by simulation under a lossy/duplicating/reordering/partitioning virtual network
- **Performance**: bursts up to ~1000 writes/s on hot keys; disk flush latency within the dump debounce window is acceptable
- **Inspectability**: on-disk state human-readable; every artifact (dump, logs, conflict journal) inspectable with standard tools

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| LWW-Element-Map CRDT as the store | Strong eventual consistency; merge is commutative/associative/idempotent, so UDP loss/duplication/reordering cannot affect outcome | ✓ Validated (Phase 2)
| HLC packed into UUIDv7 as `RecordVersion` | Byte-wise comparison = `(l, c, NodeID)` order; unique per write; no clock-sync protocol needed | ✓ Validated (Phase 2)
| UDP broadcast replication: ack + retransmit + periodic chunked DataSync anti-entropy | Fits a single trusted segment; bounded convergence (loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval`) | ✓ Validated (Phase 4)
| Human-readable JSON dump; atomic write with `.prev` fallback; write confirmed only after dump | Inspectable state; confirmed writes always on disk | ✓ Validated (Phase 3)
| gRPC API with Watch stream and `wait_for_replication` | Typed clients, streaming change notifications, optional replication confirmation | — Pending
| Trusted-segment trust model in v1 (CRC32 only, no HMAC) | Accepted limitation; HMAC is documented future work | ✓ Validated (Phase 4 — accepted risks AR-04-01..03 in 04-SECURITY.md)
| Quality gates from first commit (strict golangci-lint + go-ultimate, PR-only, CI matrix, Dependabot, govulncheck) | Correctness-critical infrastructure; gates are cheaper the earlier they land | ✓ Validated (Phase 1)
| Deterministic simulation testing as a first-class feature | Replication correctness under chaos cannot be shown by unit tests alone | ✓ Validated (Phase 4) |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-11-07 after Phase 4*
