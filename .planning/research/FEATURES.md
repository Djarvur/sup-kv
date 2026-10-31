# Feature Research

**Domain:** Leaderless replicated in-memory key-value stores (AP / CRDT-based) for small clusters
**Researched:** 2026-10-31
**Confidence:** MEDIUM (all claims cross-checked against official docs / primary repos via search + fetch; seam tier for verified websearch is MEDIUM)

**Scope note:** The v1 feature scope is pre-decided (`docs/IDEA.md`, candidate v1 requirements). This file maps the broader feature landscape of comparable products so requirements definition can sanity-check the v1 list, spot gaps, and confirm the out-of-scope list. The design is not re-litigated.

**Products analyzed:** etcd, Consul, Redis+Sentinel, KeyDB, Marmot (v2), Corrosion (Fly.io), rqlite, dqlite, LiteFS, Alertmanager cluster mode. Design-pattern references: Dynamo (anti-entropy), Cassandra (repair/streaming), Redis WAIT, etcd Leases.

## Feature Landscape

### Table Stakes (Users Expect These)

Features nearly every comparable product ships. Missing them makes the product feel incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| KV Get/Set/Delete | Every product has CRUD; it is the definition of the category | LOW | **In v1 scope** (CORE). Include create-if-absent variant — etcd txn / Consul CAS / Redis SET NX all offer it; seeding build-time defaults is a stated sup-kv use case |
| Range / prefix listing | etcd Range (prefix), Consul `?keys&prefix`, Redis SCAN/MATCH, Corrosion SQL `WHERE`; config consumers enumerate "all keys under my service prefix" at startup | LOW-MED | **GAP: not in v1 scope.** The recommended key convention `netmgr/v2/...` begs for `List(prefix)`. At 10k keys a prefix scan under RWMutex read lock is trivial; recommend adding a `List`/`GetPrefix` RPC to v1 (see MVP check) |
| Watch / change subscription | etcd Watch (from revision, prefix), Consul watches/blocking queries, Corrosion HTTP streaming subscriptions, Redis keyspace notifications; expected by every config-store consumer | MEDIUM | **In v1 scope** (API: Watch stream, local + replicated events). Consider a prefix filter argument — cost is a string comparison per event (LOW); without it every client sees whole-store chatter |
| Snapshot / backup + restore | etcd `snapshot save/restore` (Maintenance API), Consul operator snapshot, Redis SAVE/BGSAVE + replica RDB transfer, rqlite hot backup over HTTP, Marmot snapshot transfer, LiteFS export/import | MEDIUM | **In v1 scope** (PERSIST: snapshots create/load; restore-as-writes is a clean CRDT-native approach). Confirmed table stakes across all ten products |
| Crash-safe persistence with recovery chain | Every product: etcd WAL+snap, Redis RDB/AOF, SQLite WAL (Marmot/Corrosion/rqlite/LiteFS); atomic-write + fallback generation is the universal shape | MEDIUM | **In v1 scope** (PERSIST: atomic write, `.prev` fallback, bad-checksum discard). `.prev` generation fallback matches Redis RDB+AOF practice |
| Metrics endpoint (Prometheus text) | etcd/Consul/Alertmanager `/metrics`, Marmot prometheus section, Redis INFO; ops refuse to run infra without it | LOW | **In v1 scope** (OBS: unacked age, dump duration/errors, discarded-datagram classes, merge conflicts, drift warnings — better-than-average metric selection) |
| Health endpoint | etcd `/health`, Consul status/health endpoints, rqlite `/health`, Redis PING; load balancers and systemd need a liveness probe | LOW | **GAP: not named in v1 scope.** Use the gRPC standard health checking protocol (grpc-health, ~zero code) with status reflecting the startup gate. Strongly recommend adding — it is nearly free |
| Value/size limits, documented | etcd ~1.5 MiB request default, Consul 512 KiB, sup-kv 60 KiB; users must know the envelope | LOW | **In v1 scope** (value_max_bytes 61440, InvalidArgument error). Document the rationale (envelope/dump sizing) prominently |
| Single-binary deployment, flags/env config, `--version`, systemd-friendliness | etcd/consul/redis/marmot/rqlite/corrosion all ship one binary; appliance users expect `systemctl edit` + restart safety | LOW | **In v1 scope** (TOOL). Static Go binary, no cgo — matches rqlite, beats SQLite-based products on portability |
| Structured logging with distinguishable drop reasons | Every product logs; class-separated datagram drops (bad crc / own echo / bad proto) are unusually good for debugging | LOW | **In v1 scope** (OBS + structured logs) |
| Durability guarantee on acknowledged writes | Redis is the cautionary tale (async replication + everysec AOF can lose confirmed writes); etcd/Consul give it via Raft log; users of "durable service state" demand it | MEDIUM | **In v1 scope and a strength:** write returns OK only after a dump containing it; dump failure → Unavailable. Explicit, testable, stronger than Redis, clearer to reason about than Raft |

### Differentiators (Competitive Advantage)

Where sup-kv wins against the analyzed field. Aligned with Core Value: "every node stays writable at all times; a confirmed write is never lost."

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Always-writable N-node operation, no quorum | The validated pain point: 2-node etcd/Consul/rqlite clusters lose all writes when one node dies (etcd issues #863, #7906; "2-node cluster is an anti-pattern"). LiteFS loses its primary; Redis+Sentinel needs a third witness. sup-kv serves writes on every node through any partition | (decided design) | Only Marmot, Corrosion and KeyDB offer multi-writer; Marmot/Corrosion are SQLite-based with heavier machinery (2PC, CR-SQLite); KeyDB multi-master has NO conflict resolution and diverges silently — sup-kv's deterministic LWW CRDT is the defensible middle |
| Deterministic multi-node simulation testing (lossy/duping/reordering/partitioning virtual net) | None of the ten competitors ships deterministic chaos simulation as a product feature; it is FoundationDB-tier methodology. Convergence bounds ("loss ≤ ack_timeout_max, partition heal ≤ 2 × sync_interval") become executable claims | MEDIUM-HIGH | In v1 scope (TEST). Keep the simulator exported/testable — it is a marketing asset as much as a QA asset |
| Human-readable, greppable on-disk state | Competitors persist as binary (RDB, etcd bbolt/snap) or SQLite (inspectable, but not `grep`-able JSON); sup-kv dump is JSON with base64 data + `.prev` generation. An appliance operator can diagnose state with `cat`/`jq` on a box with zero extra tooling | LOW | In v1 scope (PERSIST). Aligns with "inspectability" constraint; also makes corrupted-dump recovery humanly possible |
| Conflict journal (lost versions surfaced) | LWW systems silently discard concurrent losers (KeyDB diverges, Alertmanager drops silently); sup-kv logs every lost version. Turns the known LWW tradeoff from a hidden hazard into an auditable event | LOW | In v1 scope (CORE/OBS). Pairs with the documented clock-skew caveat of LWW (skew decides which write wins, deterministically) |
| HLC + UUIDv7 record versions | Version is unique per write, byte-comparable, monotonic without clock sync protocols; Marmot v2 independently validates the recipe (per-row LWW on commit HLC, ties by node ID). No vector clocks, no sync protocol, no invented consensus | (decided design) | Internal, but it is what makes the convergence proofs tractable; keep it invisible to API clients |
| `wait_for_replication` on writes | Direct analog of Redis `WAIT numreplicas timeout` — Redis users immediately understand it; etcd has no equivalent knob (Raft makes it implicit). Optional, never changes outcome | LOW | In v1 scope (API). Report acks received within timeout |
| Zero-config broadcast discovery | Competitors require explicit join/peer lists (etcd `--initial-cluster`, rqlite `-join`, Corrosion peers, Marmot seeds); sup-kv nodes find each other with Hello broadcasts — one flag (`broadcast_addr`) and a 2-node cluster works | (decided design) | Constrained to one trusted segment — the accepted tradeoff that makes zero-config possible |
| Minimal dependencies (gRPC + protobuf only) | rqlite pulls a Raft library; Marmot needs NATS/gRPC stack; Corrosion is Rust+QUIC. HLC/UUIDv7/LWW/UDP-reliability in-repo (each tens of lines) keeps the audit surface tiny for a correctness-critical component | (decided design) | Matches the "no invented consensus, proven components" principle |

### Anti-Features (Commonly Requested, Often Problematic)

Features to deliberately NOT build — each aligns with the documented out-of-scope list, here reinforced by competitor evidence.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Multi-key transactions | etcd If/Then/Else mini-txn, Consul `/txn` (64 ops), Marmot 2PC exist | Distributed txns need coordination (2PC/Raft) — forfeits no-quorum availability; 2PC is exactly the machinery Marmot carries and sup-kv exists to avoid; no v1 consumer need | Document per-key atomicity; consumers compose multi-key sequences from Watch + prefix reads. Already out of scope — keep |
| TTL / lease keys | etcd Leases (keep-alive, auto-delete), Redis TTL/EXPIRE, Consul sessions are heavily used | TTL needs a reaper + tombstone-with-expiry semantics interacting with LWW merge; adds a whole subsystem for a use case (ephemeral registration) that belongs to service discovery, not durable config | **Action for requirements:** the out-of-scope list does not mention TTL — add it explicitly ("no key expiry in v1; keys are durable") to prevent scope creep |
| Authentication / encryption of replication traffic (HMAC, mTLS) | Near-universal table stakes: etcd/Consul RBAC+TLS, Redis ACL, Alertmanager cluster mTLS, rqlite TLS | Trusted-LAN appliance niche is the premise; HMAC/mTLS adds key distribution to a zero-config product; CRC32 integrity only is the accepted v1 limitation | Already documented future work (HMAC). Keep — but state the trust model in README ops docs in bold |
| Tombstone garbage collection / history compaction | etcd compaction (old revisions reclaimed), Cassandra GC grace; ops fear unbounded growth | Compaction interacts with anti-entropy (compacted vs stale replicas) — Cassandra's hardest code paths; deletes are rare and volumes tiny (tens of thousands of keys), so tombstones-forever is safe | Already out of scope. Document the growth bound (tombstone = small constant per deleted key) so operators can size RAM |
| Read consistency options (quorum/linearizable reads) | etcd linearizable-vs-serializable, Consul default/consistent/stale, rqlite none/weak/strong offer per-request knobs | Quorum reads require quorums — antithetical to the AP design; offering "strong" reads would be a lie under partition | Document the semantics instead: reads are local-node state; Watch gives propagation; `wait_for_replication` gives write confirmation. State read-your-writes holds on the same node only |
| Merkle-tree anti-entropy | Dynamo/Cassandra pedigree; feels "serious" | Merkle trees pay off at millions of keys/ranges; at 10k keys a chunked sorted key→version listing (decided DataSync) is smaller than the tree overhead and human-debuggable | Decided: chunked listing sync. Do not add Merkle |
| Replication traffic compression | rqlite compresses; feels free | KiB values on a LAN; protobuf envelopes are already small; compression adds CPU + a knob for no measured gain | Skip in v1; revisit only if benchmarks show broadcast-segment saturation at ~1000 writes/s bursts |
| WAN / multi-segment / explicit peer lists | Every competitor does it; users will ask | NAT traversal, relay, peer auth across trust boundaries — a different product | Out of scope. One trusted broadcast segment is the niche |
| Disk-backed storage tier (FLASH à la KeyDB) | "What if state outgrows RAM?" | Different product; KeyDB FLASH exists because Redis datasets are GB-scale; sup-kv envelope is tens of thousands of × ≤60 KiB values | Out of scope; point oversized-blob users at rqlite (Blob API) or etcd |
| Client caching libraries | etcd client-go caches watches; nice DX | Library maintenance burden across languages; documented pattern suffices at v1 scale | Out of scope v1 ("documented pattern; shipped later") — keep |
| Distributed locks / leader election on sup-kv | Consul sessions, etcd election API make KV stores double as coordinators; users will ask | Locks need linearizability/fencing — impossible on an AP LWW store; naive locks would be silently unsafe (the Redlock debate) | **Action for requirements/docs:** explicitly document "sup-kv is not a lock service; use etcd/Consul for coordination" — otherwise users will build unsafe things on it |
| Conditional writes beyond create-if-absent (CAS/compare-and-swap) | etcd txn version checks are idiomatic | CAS on a specific version is a linearizable operation — breaks AP semantics; create-if-absent is the one safe variant (version-zero compare) | Already out of scope — keep, and document why |

## Feature Dependencies

```
[LWW map engine (CORE)] ──requires──> [HLC + UUIDv7 versions (VERS)]
        │
        ├──enables──> [Watch stream]        (needs merge/local-write event taps)
        ├──enables──> [Conflict journal]    (needs merge loser visibility)
        └──enables──> [Prefix/List RPC]     (needs lock-consistent iteration)

[Persistence dump (PERSIST)] ──requires──> [CORE]
        └──enables──> [Snapshots create/load] (named dump copies)
                     [write-confirmed-after-dump guarantee] (dump pipeline hook)

[UDP replication (REPL)] ──requires──> [CORE + VERS]
        ├──enables──> [per-peer Ack tracking]
        │                └──enables──> [wait_for_replication]
        ├──enables──> [Peer table / discovery] ──enables──> [startup gate]
        └──enables──> [DataSync anti-entropy] ──enables──> [empty-node recovery]

[OBS metrics/logs] ──requires──> counters from every subsystem (engine, dump, replication)

[deterministic simulator (TEST)] ──requires──> [REPL with injectable virtual network]
                                              [CORE merge, VERS clock as pure functions]

[gRPC health endpoint] ──requires──> [startup gate state]  (trivial)
```

### Dependency Notes

- **Watch requires engine event taps:** local writes and replication merges must both emit notifications; retrofitting event emission after the engine is built is invasive — design the hook when building CORE.
- **`wait_for_replication` requires per-peer ack tracking:** the REPL unacked-set already tracks (key, version) per peer; the API layer only reads it — cheap because REPL built it first.
- **Conflict journal requires merge-loser visibility:** the merge function must return the displaced version, not just the winner — an engine API decision, not a logging afterthought.
- **Simulator requires pure-function engine + injectable network:** HLC tick/update, merge, and datagram handling must be deterministic and time-independent (virtual clock) — this constrains engine design from day one (no `time.Now()` in logic paths).
- **List/prefix RPC requires lock-consistent iteration:** a read-locked scan of the map; ordering by key sort. No dependency on replication.
- **Health endpoint conflicts with nothing; enhances systemd/LB integration** and should reflect startup-gate state so orchestrators do not route writes to a gated node.

## MVP Definition

### v1 Scope Sanity-Check (Gaps Found)

Checked against the decided candidate v1 requirements. Verdict: the v1 list is well-chosen — it covers all hard table stakes and the key differentiators. Three findings for requirements definition:

1. **GAP (recommend adding): `List`/prefix-range RPC.** No competitor ships a config store without prefix enumeration; sup-kv's own key convention (`netmgr/v2/...`) assumes it. Without it, a service cannot discover its own config keys. Complexity LOW-MED (read-locked scan + key sort, no replication involvement). This is the single most user-visible missing table-stakes feature.
2. **GAP (recommend adding): gRPC health checking endpoint.** Universal table stakes; near-zero cost via grpc-health; should report NOT_SERVING/SERVING per the startup gate.
3. **Documentation gaps (no code): (a) add TTL/lease keys to the out-of-scope list explicitly; (b) add "not a lock/election service — use etcd/Consul for coordination" to README/docs; (c) document local-read semantics (read-your-writes only on the same node; Watch for cross-node propagation).**

Everything else in the v1 list (CORE/VERS/REPL/PERSIST/API/OBS/TEST/TOOL/QUALITY/CI) matches or exceeds table stakes; the out-of-scope list is confirmed by competitor evidence (each exclusion has a "why problematic" row above).

### Launch With (v1)

The decided candidate list, plus the two gap fixes:

- [x] CORE: LWW map; Get/Set/Delete/CreateIfAbsent; tombstones; 60 KiB limit; conflict journal — table stakes + differentiator
- [x] VERS: HLC + UUIDv7 packing; monotonic under clock jumps; drift warning — the correctness core
- [x] REPL: UDP envelope; Hello discovery; DataReplication/Ack + retransmit; DataRequest; chunked DataSync; N-peer table — table-stakes anti-entropy shaped for the niche
- [x] PERSIST: JSON dump, atomic write, `.prev` fallback, debounce, write-confirmed-after-dump; snapshots — table stakes, done better (human-readable)
- [x] API: gRPC Get/Set/Delete/Watch; startup gate; `wait_for_replication` — table stakes + Redis-WAIT parity
- [x] **API addition: `List`/prefix-range RPC — closes the biggest table-stakes gap (recommendation of this research)**
- [x] **API addition: gRPC health checking protocol endpoint — universal table stakes at near-zero cost**
- [x] OBS: metrics endpoint; structured logs; discarded-datagram counters — table stakes
- [x] TEST: property tests + deterministic simulation + failure-scenario suite + benchmarks — differentiator
- [x] TOOL/QUALITY/CI: single static binary, flags/env, `--version`, systemd-friendly; gates from first commit — table stakes

### Add After Validation (v1.x)

- [ ] Watch prefix filtering — when a multi-tenant consumer complains about whole-store chatter (cost: LOW, a filter arg)
- [ ] Admin CLI subcommands (`sup-kv status`, `sup-kv dump`) or documented grpcurl recipes — when users ask (every competitor has etcdctl-class tooling; docs can carry v1)
- [ ] HMAC datagram authentication — already documented future work; trigger = first deployment outside a physically trusted segment
- [ ] `DataSync` over unicast snapshot-style fast path for large divergences (skip per-key DataRequest when a whole range is stale) — only if empty-node recovery proves too slow at real volumes

### Future Consideration (v2+)

- [ ] TTL/lease keys — only with a concrete consumer need for ephemeral registration semantics; interacts with tombstone-forever policy
- [ ] Client caching libraries — after the API surface stabilizes
- [ ] Compression of replication traffic — only if benchmarks show segment saturation
- [ ] Tombstone GC — only if a real deployment's delete volume makes RAM growth measurable

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| CORE CRUD + create-if-absent | HIGH | LOW | P1 (decided) |
| VERS HLC/UUIDv7 | HIGH | MEDIUM | P1 (decided) |
| REPL UDP + acks + DataSync | HIGH | HIGH | P1 (decided) |
| PERSIST dump + snapshots | HIGH | MEDIUM | P1 (decided) |
| API gRPC + Watch + wait_for_replication | HIGH | MEDIUM | P1 (decided) |
| **List/prefix RPC** | **HIGH** | **LOW-MED** | **P1 (recommended addition)** |
| **gRPC health endpoint** | **MEDIUM** | **LOW** | **P1 (recommended addition)** |
| Startup gate | HIGH | LOW | P1 (decided) |
| OBS metrics/logs | HIGH | LOW | P1 (decided) |
| TEST simulation suite | HIGH | HIGH | P1 (decided) |
| Conflict journal | MEDIUM | LOW | P1 (decided) |
| Watch prefix filter | MEDIUM | LOW | P2 |
| Admin CLI / grpcurl docs recipes | MEDIUM | LOW | P2 |
| HMAC auth | MEDIUM (niche-dependent) | MEDIUM | P3 (documented future work) |
| TTL keys | LOW (no consumer) | HIGH | P3 (explicitly out) |
| Multi-key txns | LOW (no consumer) | VERY HIGH | P3 (explicitly out) |
| Merkle anti-entropy | LOW (at this scale) | HIGH | P3 (explicitly out — simple listing wins) |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have / future consideration / deliberately excluded

## Competitor Feature Analysis

| Feature | etcd / Consul (Raft, CP) | Redis+Sentinel / KeyDB (primary-replica) | Marmot / Corrosion (gossip, AP) | rqlite / LiteFS (Raft/lease, CP) | sup-kv (planned) |
|---------|--------------------------|------------------------------------------|----------------------------------|-----------------------------------|------------------|
| Multi-writer without quorum | No — 2-node cluster loses writes on 1 failure | KeyDB yes but unsafe (no conflict resolution, silent divergence); Redis no (single master) | Yes (Marmot 2PC+HLC-LWW; Corrosion CR-SQLite CRDTs) | No (LiteFS lease; rqlite leader) | **Yes — LWW CRDT, deterministic, no coordination** |
| Watch/change stream | etcd Watch from revision; Consul blocking queries | Redis keyspace notifications (weak, fire-and-forget) | Corrosion HTTP streaming SQL subs; Marmot CDC via NATS publisher | rqlite queue-based CDC only (v9.0 changesets — no per-key watch); LiteFS none | gRPC Watch stream, local + replicated events |
| Prefix/range listing | Yes (Range, prefix) | Yes (SCAN/MATCH) | Yes (SQL WHERE) | Yes (SQL) | **Gap — recommend `List` RPC** |
| Multi-key transactions | Yes (If/Then/Else; /txn) | Yes (MULTI/EXEC, single node) | Marmot 2PC (ONE/QUORUM/ALL) | Yes (single SQLite txns) | **No — per-key atomicity only (documented)** |
| TTL/lease keys | Yes (Leases; sessions) | Yes (EXPIRE) | No notable | No | **No — document explicitly** |
| Read consistency options | Linearizable vs serializable (etcd); default/consistent/stale (Consul) | Local reads on replicas (stale) | Local reads (AP) | none/weak/strong per query | Local reads always (AP by design; semantics documented) |
| Write ack to replicas | Implicit via Raft commit | WAIT numreplicas (best-effort) | Marmot configurable write consistency | Implicit via Raft | `wait_for_replication` (optional, advisory) |
| Snapshot/backup | snapshot save/restore; operator snapshot API | RDB SAVE/BGSAVE; RDB shipped on full sync | Marmot snapshot transfer (parallel, cached) | rqlite hot backup HTTP; LiteFS export | Named dump copies; restore-as-writes |
| Persistence format | Binary (WAL/snap) | Binary RDB / text-ish AOF | SQLite (WAL) | SQLite | **Human-readable JSON + `.prev` (unique)** |
| Anti-entropy | Raft log replay | PSYNC partial resync + backlog; full resync ships RDB | Marmot log pull; Corrosion periodic subset sync | Raft; LiteFS LTX streaming | Ack+retransmit + chunked listing DataSync (rate-limited) |
| Metrics/health | /metrics + /health | INFO + PING | Prometheus (Marmot); agent (Corrosion) | /health + metrics | Metrics endpoint + **add gRPC health** |
| Auth/encryption | RBAC + TLS/mTLS | ACL + password + TLS (Redis); KeyDB same | Marmot cluster_secret PSK; Corrosion QUIC encrypted | rqlite TLS + basic auth | **None v1 (trusted segment; HMAC future work)** |
| Deployment | Single binary, explicit initial-cluster config | Single binary + Sentinel (3rd party) | Marmot single binary + seeds; Corrosion agent + peers | Single binary + join | Single static binary + **zero-config broadcast discovery** |
| Conflict safety story | Linearizable — no conflicts (blocks writes) | Sentinel split-brain risk; KeyDB silent divergence | Marmot per-row LWW on HLC (node-ID tie-break — same recipe as sup-kv); Corrosion CRDT merge | Linearizable — no conflicts | Deterministic LWW supremum + conflict journal (losers surfaced) |

## Sources

Confidence per seam: websearch cross-checked = MEDIUM; primary-repo fetches (Marmot, Corrosion) corroborated by independent search summaries = MEDIUM. No HIGH tier available (no curated/docs provider configured).

- etcd API and glossary (etcd.io/docs v3.6/v3.7 `learning/api`, `learning/glossary`) — KV/Watch/Lease/Auth/Maintenance, compaction — MEDIUM
- Consul docs (developer.hashicorp.com: blocking queries, `/txn`, consistency modes) — MEDIUM
- Redis docs + Stack Overflow (redis.io/docs replication; PSYNC/backlog; WAIT) — MEDIUM
- KeyDB docs + GitHub wiki (docs.keydb.dev active replication, multiple-masters, cluster-spec; valkey #512 discussion) — MEDIUM
- Marmot v2 README (github.com/maxpert/marmot, fetched 2026-10-31) — leaderless gossip, 2PC ONE/QUORUM/ALL, HLC-LWW, snapshot transfer, MySQL wire, limitations — MEDIUM
- Corrosion README + fly.io/blog/corrosion (github.com/superfly/corrosion, fetched 2026-10-31) — CR-SQLite CRDTs, SWIM/Foca, QUIC, streaming SQL subscriptions, Rhai templates — MEDIUM
- rqlite GitHub + rqlite.io features docs + author blog — read consistency levels, hot backup, Blob API, CDC (v9.0, queue-based changesets, no per-key watch — negative claim verified against docs/issues) — MEDIUM
- LiteFS docs (docs.fly.io/litefs how-it-works, config, ARCHITECTURE.md) — leases, LTX, import/export — MEDIUM
- Prometheus Alertmanager HA docs + repo design doc (prometheus.io/docs/alerting/latest/high_availability) — gossip, full replication, LWW, mTLS — MEDIUM
- etcd quorum pain: GitHub etcd issues #863, #7906, #12128; DBA StackExchange quorum math — MEDIUM
- LWW clock-skew critique: "Clocks Are Bad" (2013) + 2025-2026 articles on clock skew/lost updates — MEDIUM
- Anti-entropy: Dynamo paper discussions, DataStax anti-entropy repair docs, Cassandra/Riak streaming — MEDIUM

---
*Feature research for: leaderless replicated in-memory KV store (AP/CRDT, small clusters)*
*Researched: 2026-10-31*
