# sup-kv

## What This Is

**sup-kv** ("supremum storage") is a leaderless replicated in-memory key-value store for small clusters — two to a handful of nodes on one trusted LAN broadcast segment. Every node accepts reads and writes locally and independently (AP); replicas exchange changes over UDP and converge to identical state — the pointwise supremum of all writes under the record-version order (strong eventual consistency via a Last-Write-Wins-Element-Map CRDT). It ships as one static Go binary with minimal dependencies and human-readable JSON persistence.

For service builders on small HA clusters (e.g. two management boards in one appliance) who need a shared, always-writable store for configuration and durable service state where quorum-based stores (etcd, Consul) cannot survive the loss of a single node.

## Core Value

Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.

## Current Milestone: v1.1 Spec Compliance and Resilience

**Goal:** Close the findings of the 2026-11-19 v1.0 review (`docs/reviews/2026-11-19-v1.0/`) — recovery and burst scenarios meet the stated bounds on real sockets, and every deviation from the Spec is either fixed in code or recorded in a deviation registry.

**Target features:**
- Spec deviation registry + owner decisions 14.1–14.11 as ADRs (S-12)
- Stand & simulator as tests: F9/F9b/F14/burst/F2c automated on real Linux sockets; simulator gains datagram budgets, in-loop codec, dump delay, gate, restart-with-state, deletions; chaos test stabilized (T-01…T-16)
- Transport: paced sync/retransmit sending, socket buffer setup, event-loop cost bounded, cheap key→version reads, key/value size bounds, received-version validation (R-01, R-02, R-06–R-09, R-11, R-14–R-16, Q-01, Q-02)
- Protocol: interface closed until the full key list arrives, `startup_gate_empty` for empty disk, `DataSync` header (SyncID/ChunkIndex/ChunkCount), third Replication outcome, immediate sync for a newly seen peer, 3+-node behavior and tests (R-03–R-05, R-10, R-12, R-13, S-02, S-08)
- Persistence: single source for the value limit, dump-load diagnostics, generation integrity, dirty-after-fallback, decaying dump retries, streaming encode (P-01–P-10, Q-03, Q-04, Q-06)
- API & semantics: honest `wait_for_replication`, snapshot load as full rollback, error codes, `List` pagination, public proto package + Go client example in CI (A-01–A-06, S-01)
- Observability & configuration: event logging, config validation, honest metric definitions incl. the conflict counter, exit codes (O-01–O-05, S-06, S-07)
- Performance: benchmarks at defaults and at the fast-response profile with thresholds (PERF-01–PERF-06)
- Documentation & release: D-01–D-10; darwin demoted to development-only (14.10)

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
- ✓ **API** — Phase 5 (gRPC service: Get/Set/SetIfAbsent/Delete/List, Watch stream with exact snapshot-then-live cut, startup gate + standard health, `wait_for_replication`, snapshot RPCs)
- ✓ **Tooling** — Phase 5 (single static binary, 19 flags with SUPKV_ env twins, `--version`, D-19 ordered graceful shutdown; systemd unit lands in Phase 7)
- ✓ **Observability** — Phase 6 (opt-in mTLS metrics/debug endpoint with in-repo Prometheus text exporter at `/metrics` + expvar/pprof `/debug`; slog JSON default with SUPKV_LOG_LEVEL/FORMAT; OnDiscard seam with distinguishable discard classes and per-class counters; counter seams across repl/persist/udprepl)
- ✓ **Testing (remaining)** — Phase 6 (TEST-04 failure-scenario suite: 9 deterministic simulator scenarios, real-socket two-node scenarios, 4 two-process exec scenarios on the real binary; TEST-05 benchmarks behind `make bench`: dump-10k, replication burst ~975 w/s, confirmation latency ~10ms under 1000 w/s load)
- ✓ **Docs, examples, release** — Phase 7 (README quickstart executed VERBATIM by CI incl. kill −9/restart convergence — Linux cell green, macOS leg a named loud-skip per ratified option-b; docs/OPERATIONS.md five trust-model statements machine-anchored + human-verified; docs/EXAMPLES.md 10 CI-executed grpcurl blocks over every RPC; CHANGELOG 1.0.0; goreleaser tool-directive pipeline — tag v1.0.0 → GitHub Release with 4 archives + SBOMs + checksums, shipped binary verified `sup-kv v1.0.0`)

### Active

All Active requirements are hypotheses until shipped and validated.

*(v1.1 requirements are defined in REQUIREMENTS.md — derived from the 2026-11-19 v1.0 review with its acceptance criteria carried verbatim; owner decisions 14.1–14.11 of the review are settled and not re-litigated)*

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
- **Process**: PR-only development — branch protection on `master` (server-side ruleset ENFORCED since 2026-11-18, repo public; payload and audit trail in docs/setup/branch-protection.md), direct pushes forbidden, every change lands through a PR from a feature branch; CI on every push and PR
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
| gRPC API with Watch stream and `wait_for_replication` | Typed clients, streaming change notifications, optional replication confirmation | ✓ Validated (Phase 5) |
| Trusted-segment trust model in v1 (CRC32 only, no HMAC) | Accepted limitation; HMAC is documented future work | ✓ Validated (Phase 4 — accepted risks AR-04-01..03 in 04-SECURITY.md)
| Quality gates from first commit (strict golangci-lint + go-ultimate, PR-only, CI matrix, Dependabot, govulncheck) | Correctness-critical infrastructure; gates are cheaper the earlier they land | ✓ Validated (Phase 1)
| Deterministic simulation testing as a first-class feature | Replication correctness under chaos cannot be shown by unit tests alone | ✓ Validated (Phase 4) |
| In-repo Prometheus text exporter; opt-in mTLS metrics/debug endpoint | Counters/gauges only — client_golang would add 3+ modules against the "gRPC + protobuf only" bar; opt-in + mTLS keeps the pprof/expvar surface closed by default (T-6-07..10) | ✓ Validated (Phase 6) |
| slog JSON at the binary, OnDiscard seam in the core | Structured logs as the inspectability floor; `internal/app` stays free of slog — the adapter bridges the tap at its own edge (D-08/D-09) | ✓ Validated (Phase 6) |
| darwin F_FULLFSYNC durability flush in persistfile | Strict superset of File.Sync on darwin (fcntl(2)); closes the phase-3 open planning decision resolved-with-code; other platforms byte-identical | ✓ Validated (Phase 6) |
| Documentation as machine-verified contract (doc-is-test-input) | CI extracts fenced blocks from README/EXAMPLES/OPERATIONS and executes them verbatim against the built binary — docs cannot rot silently | ✓ Validated (Phase 7) |
| goreleaser + grpcurl as go.mod tool directives; tag-triggered release workflow | Same-version guarantee as all dev tools; release tooling never a runtime dep; ci.yml (ruleset contract) untouched | ✓ Validated (Phase 7 — v1.0.0 shipped) |
| macOS quickstart = option-b (Linux CI-verified + marked darwin variant, loud-skip) | macos-15 has no usable broadcast form (255.255.255.255 no-route, probe verdict recorded); D-06 satisfied partially by maintainer decision, honestly recorded | ⚠️ Revisit at v1.0.1+/ruleset window |

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
*Last updated: 2026-11-19 — milestone v1.1 started (input: v1.0 review at `docs/reviews/2026-11-19-v1.0/`, owner decisions 14.1–14.11 accepted 2026-11-19)*
