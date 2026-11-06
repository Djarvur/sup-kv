# Roadmap: sup-kv

## Overview

sup-kv ships in seven dependency-forced phases. Quality gates and the depguard-enforced repo skeleton land before any feature code (Phase 1), the pure decision core — HLC/UUIDv7 versioning and the LWW-Element-Map store — follows with its correctness laws proven by property tests (Phase 2), durable human-readable persistence with the write-confirmed-after-dump contract third (Phase 3), then UDP broadcast replication proven under chaos by the deterministic simulator (Phase 4, the heart of the project), the user-facing gRPC API and single static binary (Phase 5), the full failure-scenario suite plus observability and benchmarks (Phase 6), and finally docs, examples and the v1.0 release (Phase 7). The order is not preference: engine ← persistence ← replication ← API is dependency-forced, each phase's exit criteria are testable through `port.App` with fakes, and the simulator lands with replication — not in hardening — because convergence under chaos is the replication phase's own acceptance criterion.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: CI Bootstrap and Repo Skeleton** - Quality gates, 4-OS CI matrix, Dependabot/govulncheck and the machine-enforced architecture boundary, before any feature code (completed 2026-11-02)
- [x] **Phase 2: Engine — HLC Versioning and LWW Store** - Pure in-memory decision core: UUIDv7-packed HLC versions, LWW-Element-Map with tombstones and conflict journal, proven by property tests (completed 2026-11-02)
- [x] **Phase 3: Persistence — Dump and Write Confirmation** - Human-readable JSON dumps with atomic write, `.prev` fallback chain and snapshots; a confirmed write is always on disk (completed 2026-11-03)
- [ ] **Phase 4: Replication and Deterministic Simulator** - UDP broadcast replication (Hello, DataReplication/Ack, DataSync) with N-peer tracking, plus the seeded-chaos simulator that proves the convergence bounds
- [ ] **Phase 5: gRPC API and Binary** - The user-facing surface: Get/Set/Delete/List/Watch, startup gate, `wait_for_replication`, health endpoint, snapshots, single static binary
- [ ] **Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks** - Every documented failure scenario as an automated test (simulator plus real sockets), metrics endpoint, structured logs, benchmarks
- [ ] **Phase 7: v1.0 — Docs, Examples and Release** - Verified quickstart, ops docs with trust model and drift runbook, API examples, release tag

## Phase Details

### Phase 1: CI Bootstrap and Repo Skeleton
**Goal**: Quality gates and the enforced repo skeleton exist before any feature code, so every subsequent change lands inside the architecture and behind a red-light-blocking gate
**Depends on**: Nothing (first phase)
**Requirements**: CI-01, CI-02, CI-03, QUAL-01
**Success Criteria** (what must be TRUE):
  1. A pull request runs the full gate matrix — strict golangci-lint, `go vet`, race-enabled tests — on Linux and macOS × amd64/arm64, and a red gate blocks the merge
  2. Direct pushes to `master` are rejected by branch protection; every change (code and docs) lands through a pull request from a feature branch
  3. Dependabot is active (updates and alerts) and the scheduled `govulncheck` job runs green against the Go vulnerability database
  4. The dependency boundary is machine-enforced from the first feature commit: code in `internal/app` importing `net`/`os`/grpc (or `internal/port` reaching beyond stdlib) fails the lint gate
**Plans**: 6/6 plans complete (4 executed, 2 gap-closure pending)
Plans:
**Wave 1**
- [x] 01-01-PLAN.md — Walking skeleton: module, packages, pinned tools, vendored strict lint + depguard boundary (QUAL-01)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 01-02-PLAN.md — CI gate matrix: lint + 4-cell matrix proven green through a PR, red-teeth proof (CI-01)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 01-03-PLAN.md — Dependency currency: scheduled govulncheck + Dependabot (CI-02)

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 01-04-PLAN.md — Branch protection on master: scripted ruleset + plan-blocker decision (CI-03)

**Wave 5** *(gap closure — blocked on Wave 4)*
- [x] 01-05-PLAN.md — Record ratified accepted overrides: VERIFICATION.md frontmatter override mechanism + REQUIREMENTS.md CI-02 Partial annotation, CI-03 ratification (CI-02, CI-03)

**Wave 6** *(gap closure — blocked on 01-05: 01-06's markers and decision-log entries cite 01-05's recorded overrides as accomplished fact)*
- [x] 01-06-PLAN.md — Accepted-deferral state markers: branch-protection.md DEFERRED status, PROJECT/CLAUDE process constraint (IN-02), WINDOWS.md item 2 waiver, STATE.md decision log (CI-02, CI-03)

### Phase 2: Engine — HLC Versioning and LWW Store
**Goal**: The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Depends on**: Phase 1
**Requirements**: CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, CORE-07, VERS-01, VERS-02, VERS-03, VERS-04, VERS-05, TEST-01, TEST-02
**Success Criteria** (what must be TRUE):
  1. All KV operations behave on a single node: Get returns NotFound for missing and tombstoned keys, Set creates or replaces under LWW, create-if-absent returns AlreadyExists for any existing record (including a tombstoned one), Delete writes a fresh-version tombstone, and values over `value_max_bytes` are rejected with InvalidArgument
  2. Property tests prove merge is commutative, associative and idempotent — including tombstoned entries — so delivery order, duplicates and retransmissions cannot affect the outcome, and every version displaced by a merge is recorded in the conflict journal
  3. Byte-wise UUIDv7 comparison equals `(l, c, NodeID)` order and versions are unique per write by construction — both property-tested
  4. The HLC stays monotonic under adversarial clock input (backward and forward jumps, stuck clock, counter overflow, restart initialized from the dump's max `l` rather than the wall clock), and drift beyond `drift_warn` is logged as a warning
**Plans**: 4/4 plans complete (3 executed, 1 gap-closure)
Plans:
**Wave 1**
- [x] 02-01-PLAN.md — Tracer: engine core wired end-to-end — errors, UUIDv7 RecordVersion, HLC clock with now seam, LWW store with merge losers returned (CORE-01..CORE-07)

**Wave 2** *(blocked on Wave 1)*
- [x] 02-02-PLAN.md — Clock and version hardening: Kulkarni unit tables, RFC 9562 layout proofs, seeded replayable TEST-02 property suite (VERS-01..VERS-05, TEST-02)

**Wave 3** *(blocked on Wave 2)*
- [x] 02-03-PLAN.md — TEST-01 merge-law property suite (tombstone-inclusive, exact loser accounting) + concurrency hammer + full phase gate (TEST-01, CORE-06, CORE-07)

**Wave 4** *(gap closure — blocked on Waves 1-3)*
- [x] 02-04-PLAN.md — Gap closure G-02-VERS04-DOMINANCE / WR-01: counter-seeded restart seam (NewClock initialC, Config.InitialC) with strict dominance over the full dump watermark (VERS-02, VERS-04, TEST-02)

### Phase 3: Persistence — Dump and Write Confirmation
**Goal**: State survives restarts and a client-confirmed write is always on disk — the write-confirmation contract that makes every later phase strictly additive
**Depends on**: Phase 2
**Requirements**: PERS-01, PERS-02, PERS-03, PERS-04, PERS-05
**Success Criteria** (what must be TRUE):
  1. The full map (tombstones included) round-trips through a human-readable JSON dump that is greppable with standard tools
  2. A write returns OK only after a dump containing it completed (version watermark); on dump failure it returns Unavailable while staying in memory; kill −9 and restart recovers via the fallback chain main → `.prev` → empty start, discarding bad-checksum records
  3. A sustained burst of ~1000 writes/s still sees every write confirmed within bounded latency — debounced dumps coalesce but never starve
  4. Named snapshots can be created and loaded, and restore replays the snapshot as ordinary writes/deletes with fresh versions so it replicates and wins by normal LWW
**Plans**: 2/2 plans complete
Plans:
**Wave 1**
- [x] 03-01-PLAN.md — Tracer: write-confirmed JSONL dump, atomic write with `.prev` fallback chain, restart watermark via persist.Open, engine confirmation/burst/failure synctest suite (PERS-01, PERS-02, PERS-03, PERS-04)

**Wave 2** *(blocked on Wave 1)*
- [x] 03-02-PLAN.md — Snapshots: flush-forced byte-copy create, restore-as-writes with fresh dominating versions, name-validation security table (PERS-05)

### Phase 4: Replication and Deterministic Simulator
**Goal**: Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment, with convergence under chaos proven by the deterministic simulator
**Depends on**: Phase 3
**Requirements**: REPL-01, REPL-02, REPL-03, REPL-04, REPL-05, REPL-06, REPL-07, REPL-08, REPL-09, REPL-10, TEST-03
**Success Criteria** (what must be TRUE):
  1. Two (or N) nodes discover each other via broadcast Hello with no configuration beyond `broadcast_addr`, and a write on one node becomes visible on every other within one RTT in normal operation
  2. Every local change is broadcast and tracked per-peer (key, version) until Ack or supersession, retransmitting with exponential backoff and suspending (not dropping) while no peer is known; receivers Ack after applying to memory; the unacked set drains after a partition heals
  3. Full reconciliation works in both directions: chunked DataSync runs on the sync timer, at startup and on neighbor restart; receivers answer with DataRequest or unicast DataReplication rate-limited by `sync_response_rate`; a DataRequest for an unknown key gets no reply; a Hello with a new `run_id` triggers full sync toward the restarted node
  4. Malformed traffic is dropped with distinguishable classes: duplicate `node_id` logged as a configuration error, plus own broadcast echo, bad CRC and bad proto version each identifiable
  5. The deterministic simulator runs 2..N nodes over a seeded virtual network that loses, duplicates, reorders and partitions datagrams, and asserts convergence within the documented bounds — normal one RTT, datagram loss ≤ `ack_timeout_max`, partition heal ≤ 2 × `sync_interval` + response time — printing the seed on failure
**Plans**: 4/6 plans executed
Plans:
**Wave 1**
- [x] 04-01-PLAN.md — Tracer: port seam + wire schema + minimal event-loop core + UDP adapter; one write replicated node-to-node end-to-end (REPL-03)

**Wave 2** *(blocked on Wave 1)*
- [x] 04-02-PLAN.md — Reliability layer: per-peer (key, version) ack tracking, exponential-backoff retransmission, suspension and flush-on-discovery, peer table lifecycle with peer_timeout eviction (REPL-02, REPL-06)

**Wave 3** *(blocked on Wave 2)*
- [x] 04-03-PLAN.md — Reconciliation: DataRequest replies (tombstones included), deterministic chunked DataSync listings at sync_chunk_bytes, startup/timer triggers, rate-limited egress queue (REPL-04, REPL-05)

**Wave 4** *(blocked on Wave 3)*
- [x] 04-04-PLAN.md — Restart semantics: new run_id from a known node triggers the targeted full listing; stale-run_id and in-flight-traffic edges pinned (REPL-07)

**Wave 5** *(blocked on Wave 4)*
- [ ] 04-05-PLAN.md — Deterministic simulator: simnet discrete-event virtual network, the three named convergence bound tests, seeded chaos test with SUPKV_SIM_SEED replay; phase gate (TEST-03, REPL-10)
- [ ] 04-06-PLAN.md — Expansion sibling of the tracer (parallel with 04-05): full discard classification with adversarial totality, persist.Engine.ApplyRemote seam, real-socket two-node smoke (REPL-01, REPL-08, REPL-09)

### Phase 5: gRPC API and Binary
**Goal**: sup-kv becomes a usable product — a single static binary exposing the typed gRPC surface with Watch, startup gate, `wait_for_replication` and standard health checking
**Depends on**: Phase 4
**Requirements**: API-01, API-02, API-03, API-04, API-05, API-06, API-07, TOOL-01
**Success Criteria** (what must be TRUE):
  1. A gRPC client can Get, Set (including create-if-absent), Delete and List keys under a prefix in sorted order, with the full error set OK / NotFound / AlreadyExists / InvalidArgument / Unavailable
  2. A Watch stream delivers change notifications covering local and replicated changes alike (including replicated deletes) without a slow consumer blocking the store or other watchers
  3. Writes are rejected with Unavailable during the startup gate (until the first full sync round arrives or `startup_gate` elapses) while reads are served immediately, and the standard gRPC health endpoint reflects the gate state (NOT_SERVING until gated open)
  4. A write with `wait_for_replication` waits for peer Acks within a timeout, reports whether they arrived, and never changes the write outcome; snapshots can be created and loaded through the API
  5. The shipped artifact is a single static binary with flags/env config, `--version`, and systemd-friendly behavior
**Plans**: TBD

### Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks
**Goal**: Every documented failure scenario is an automated passing test and the node is observable enough to operate — the evidence that the core value holds under chaos
**Depends on**: Phase 5
**Requirements**: OBS-01, OBS-02, OBS-03, TEST-04, TEST-05
**Success Criteria** (what must be TRUE):
  1. Every failure scenario from the design doc is an automated test demonstrating convergence within the documented bounds — datagram and Ack loss, duplicates/reordering/own-echo, partitions for seconds and for days (both sides serving writes, deterministic winner, conflict-journal entries), one-way connectivity, node restart via RunID change, empty-disk recovery at `sync_response_rate`, both-nodes-down restart, corrupted dump records, wall-clock jumps live and across restarts, a far-ahead clock, ~1000 writes/s bursts, duplicate `node_id`, protocol version mismatch — including real-socket two-process runs in CI, not only simulation
  2. The metrics endpoint exposes unacked-records count and oldest age, dump duration and errors, merge conflicts, clock-drift warnings, and discarded datagrams by class
  3. Logging is structured and every dropped-datagram class (bad CRC / own broadcast / bad proto version) is distinguishable in logs
  4. Benchmarks cover dump at 10k keys and replication throughput under burst writes, with confirmation latency staying bounded under bursts
**Plans**: TBD

### Phase 7: v1.0 — Docs, Examples and Release
**Goal**: A stranger can deploy and operate a two-node sup-kv cluster from the documentation alone, and v1.0 ships
**Depends on**: Phase 6
**Requirements**: DOCS-01, DOCS-02, DOCS-03
**Success Criteria** (what must be TRUE):
  1. The README quickstart works exactly as written on Linux and macOS: two nodes on one localhost broadcast socket, writes visible on both within an RTT, kill −9 one node, restart, state converges without operator action
  2. Ops docs state the trust model prominently (no auth/encryption in v1, HMAC is future work), the value-size envelope, local-read semantics (read-your-writes on the same node only), a "not a lock/election service" warning, and a clock-drift runbook
  3. API usage examples are provided and work against the shipped binary
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. CI Bootstrap and Repo Skeleton | 6/6 | Complete    | 2026-11-02 |
| 2. Engine — HLC Versioning and LWW Store | 4/4 | Complete    | 2026-11-02 |
| 3. Persistence — Dump and Write Confirmation | 2/2 | Complete    | 2026-11-03 |
| 4. Replication and Deterministic Simulator | 4/6 | Planning complete | - |
| 5. gRPC API and Binary | 0/TBD | Not started | - |
| 6. Hardening — Failure Scenarios, Observability, Benchmarks | 0/TBD | Not started | - |
| 7. v1.0 — Docs, Examples and Release | 0/TBD | Not started | - |
