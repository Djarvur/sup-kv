# Project Research Summary

**Project:** sup-kv (from `.planning/PROJECT.md`)
**Domain:** Leaderless replicated in-memory key-value store — AP, LWW-Element-Map CRDT, HLC/UUIDv7 versioning, UDP broadcast replication, JSON dump persistence, gRPC API — in Go, single static binary, small trusted-LAN clusters
**Researched:** 2026-10-31
**Confidence:** HIGH overall — versions machine-verified against proxy.golang.org / GitHub Releases / Go 1.27.1 GOROOT; layout binding per project-mandated go-ultimate conventions; features/pitfalls MEDIUM and refinements only. The design is decided (`docs/IDEA.md`); research covers implementation, not design.

## Executive Summary

sup-kv is a leaderless, always-writable KV store for 2-to-a-handful-node clusters where quorum stores (etcd, Consul, rqlite) cannot survive a single node loss. The research consistently shows one expert pattern across every reference system (etcd raft, Alertmanager cluster, Marmot, Corrosion, CockroachDB HLC): a **deterministic decision core with all I/O at the edges** — protocol state owned by a single event loop, membership separate from data replication, anti-entropy separate from the hot path, a startup "settle" gate, and ordered step-isolated shutdown. The decided design maps onto this with zero friction: a bounded-context hexagonal layout (`internal/port`, `internal/app`, adapters for gRPC/UDP/dump, `internal/sim`) in which the three out-ports (`Transport`, `Dumper`, `WallClock`) are simultaneously the production seams and the simulator's injection points. This is not optional structure — the v1.0 success criterion (deterministic convergence under a chaotic virtual network) is only achievable if the whole decision core runs on a virtual network and virtual clock.

The recommended stack is deliberately tiny and fully verified: Go 1.27, grpc-go v1.85.0, protobuf v1.36.12 as the only required runtime modules; `pgregory.net/rapid` v1.3.0 as the single test-only dependency (property-based testing with shrinking — stdlib `testing/quick` is insufficient); an in-repo deterministic simulator (no Go DST framework exists — verified) built on `testing/synctest` plus a seeded-chaos virtual network; strict linting via powerman/golangci-lint-strict v2.15.0; all dev tools pinned with Go 1.24+ `tool` directives; Buf CLI v1.75.0 for hermetic protobuf codegen. The 7-phase build order suggested in PROJECT.md (CI → Engine → Persistence → Replication → API → Hardening → v1.0) is validated as dependency-forced, with two research refinements: the wire/main skeleton lands in Phase 1, and **the simulator lands in Phase 3 with replication — not in hardening — because convergence under chaos is the replication phase's own acceptance criterion**.

The risks are implementation slips, not design flaws, and every one has a cheap prevention that belongs to the phase building that subsystem: debounce must be bounded-coalescing with a version watermark (not reset-on-write); dumps must snapshot outside the map lock; HLC needs adversarial-clock property tests (counter reset, init-from-dump-max); tombstones must persist in dumps and delete must mint a fresh version; large datagrams depend on IP fragmentation (accept, model per-fragment loss, document); the UDP read buffer must be raised and its effective size logged; Watch fan-out must be non-blocking with per-stream senders; self-echo must be filtered by `node_id`, never by address. Feature research adds exactly two scope recommendations: a `List`/prefix RPC (the biggest table-stakes gap — no competitor ships a config store without prefix enumeration) and the standard gRPC health-checking endpoint (near-zero cost, reflects the startup gate).

## Key Findings

### Recommended Stack

The shipped runtime graph is **two required modules** — `google.golang.org/grpc` v1.85.0 and `google.golang.org/protobuf` v1.36.12 — plus at most two zero-marginal direct deps already inside grpc's module graph (`github.com/google/uuid` v1.6.0 for RunID, `golang.org/x/sync` v0.23.0 for errgroup). Everything else is stdlib or test-only. This satisfies the project's "gRPC + protobuf only" bar exactly; HLC, UUIDv7 packing, the LWW map, the UDP reliability layer, and the Prometheus text exporter (~100 lines) are all in-repo by design.

**Core technologies:**
- **Go 1.27** (`go 1.27` directive) — latest stable; forced floor is 1.26 by grpc-go v1.85; brings jsonv2 engine under the stable v1 API and stable `testing/synctest`
- **grpc-go v1.85.0** — API layer incl. Watch streams; enable `grpc/reflection` so grpcurl works for ops; `GracefulStop` fixes relevant to shutdown order
- **protobuf v1.36.12** (+ `protoc-gen-go` same version, `protoc-gen-go-grpc` v1.6.2 via tool directives) — envelope and API message encoding
- **stdlib `net` UDP** — `SO_BROADCAST` is default-on for datagram sockets on all platforms (GOROOT-verified); one unconnected socket serves broadcast + unicast; call `SetReadBuffer` early (see Pitfall 4)
- **stdlib `encoding/json` v1 API** — sorted map keys + automatic base64 give a deterministic, greppable dump; do NOT migrate to the v2 API mid-project
- **`log/slog` + `expvar` + in-repo Prometheus text exporter** — observability floor with zero new modules
- **`pgregory.net/rapid` v1.3.0** (test-only) — the one justified PBT dependency; shrinking is mandatory for HLC/merge counterexamples
- **In-repo simulator** (`testing/synctest` + seeded virtual network) — no Go DST framework exists (verified; "pterodactyl" library from the brief does not exist); assert convergence invariants and bounds, not bit-identical traces
- **Dev tooling pinned via `go tool`**: golangci-lint v2.15.0 with powerman/golangci-lint-strict config (exact-version match), govulncheck v1.9.0 (scheduled cron), Buf CLI v1.75.0 with `local: go tool` plugins (hermetic, no BSR); CI on checkout@v7 / setup-go@v7 with `go-version-file: go.mod`, 4-OS race matrix, `go mod tidy` gate, grouped weekly Dependabot

**Critical version constraints:** golangci-lint-strict config must match golangci-lint exactly (v2.15.0); protoc-gen-go must equal the protobuf runtime version; protoc-gen-go-grpc v1.6.x requires `Unimplemented<Service>Server` embedded by value; golangci-lint-action v9 pairs only with golangci-lint v2.

### Expected Features

The decided v1 list covers all hard table stakes and the key differentiators; competitor analysis (etcd, Consul, Redis+Sentinel, KeyDB, Marmot, Corrosion, rqlite, dqlite, LiteFS, Alertmanager) confirms the out-of-scope list is correct — every exclusion has a concrete "why problematic" justification. The validated pain point (2-node quorum clusters lose all writes on single-node failure; KeyDB's multi-master diverges silently with no conflict resolution) is exactly sup-kv's niche; Marmot v2 independently validates the HLC-LWW-with-node-ID-tiebreak recipe.

**Must have (table stakes, all in decided v1 scope):**
- KV CRUD + create-if-absent — category definition; create-if-absent serves build-time-defaults seeding
- Watch/change stream — every config-store consumer expects it
- Snapshot/backup + restore; crash-safe persistence with recovery chain — universal across all ten products; sup-kv's restore-as-writes is cleanly CRDT-native
- Metrics endpoint; health endpoint (gap — see below); value limits documented; single-binary flags/env deployment; structured logs with distinguishable drop classes
- **Durability on acknowledged writes** — sup-kv's write-confirmed-after-dump is a strength: stronger than Redis, clearer than Raft

**Recommended additions (research verdict — carry into requirements):**
- **`List`/prefix-range RPC** — the single most user-visible missing table-stakes feature; the project's own `netmgr/v2/...` key convention assumes it; LOW-MED complexity (read-locked scan + key sort, no replication involvement)
- **gRPC standard health-checking endpoint** — universal table stakes at near-zero cost; status reflects the startup gate so orchestrators avoid gated nodes
- **Documentation actions (no code):** add TTL/lease keys to out-of-scope explicitly; state "not a lock/election service — use etcd/Consul for coordination"; document local-read semantics (read-your-writes only on the same node)

**Differentiators (keep prominent):** always-writable no-quorum operation; deterministic multi-node simulation as a product feature; human-readable greppable JSON dumps with `.prev`; conflict journal surfacing lost versions; `wait_for_replication` (Redis WAIT analog); zero-config broadcast discovery; minimal audit surface.

**Defer (v1.x / v2+):** Watch prefix filter (v1.x, one filter arg); admin CLI/grpcurl recipes (docs carry v1); HMAC datagram auth (documented future work); unicast fast-path DataSync; TTL keys, client caching libraries, compression, tombstone GC (v2+, only on measured need).

### Architecture Approach

**The one principle to enforce in review: deterministic core, I/O at the edges.** Everything that decides (merge, version choice, retransmission timing, peer liveness, sync scheduling, gate state) lives in `internal/app` behind ports with no socket/file/wall-clock access; everything that touches the world lives in adapters. etcd raft's README is the canonical argument; for sup-kv it is additionally the precondition for deterministic simulation. Enforcement is mechanical: golangci-lint `depguard` forbids `net`/`os`/grpc/`api` imports inside `internal/app` — add the rule in Phase 0 so every phase lands inside the architecture from its first commit.

**Major components** (go-ultimate bounded-context hexagonal; no `pkg/`, `main.go` at module root):
1. `internal/port` — the whole contract: `App` in-port; `Transport`, `Dumper`, `WallClock` out-ports; port-level errors; boundary DTOs. The most-depended-on leaf; enables the simulator without monkey-patching
2. `internal/app` (the Node, one package many files) — `version.go` (HLC + UUIDv7 pack/compare, CockroachDB `Clock`/`WallClock` precedent), `store.go` (LWW map under RWMutex, merge, tombstones, conflict journal), `peers.go` (peer table keyed by NodeID with run_id), `repl.go`/`sync.go` (retransmit engine + chunked DataSync), `gate.go` (startup gate, dump debounce, write confirmations, `wait_for_replication`), `watch.go`, `node.go` (single-owner event loop: inbound envelopes / local writes / ticks — the determinism trick)
3. `internal/in/grpc` — proto⇄DTO conversion, error→status mapping, Watch stream lifecycle
4. `internal/out/udp` — socket, envelope codec, CRC32, own-echo discard, discard classification; **no business decisions**
5. `internal/dal/dump` — JSON dump, atomic write + `.prev` rotation, load fallback chain, snapshots
6. `internal/sim` (test-only) — seeded-chaos virtual network (loss/dup/reorder/partition/one-way), jumpable virtual clock, in-memory Dumper; runs N nodes via the same ports
7. `wire.go`/`main.go` — Wire builds the graph and starts nothing; main runs Units in parallel with ordered, step-isolated shutdown: gRPC `GracefulStop` → **final dump flush (persist unconfirmed writes too)** → node loop + socket → timers/watchers/Close

**Key data flow rule:** version → store → (watch, repl, dump) — the record flows outward; nothing flows back into the local version decision. Merges are idempotent/commutative, so the inbound path tolerates loss/duplication/reordering by construction.

### Critical Pitfalls

All twelve pitfalls in PITFALLS.md are critical-class; the top ones by blast radius, each with its prevention phase:

1. **Debounce starvation (Phase 2)** — reset-on-write debounce under sustained ~1000 writes/s never fires, starving every pending write. Prevent: bounded coalescing (max-wait cap after first dirty mark) + confirm by **version watermark** (write confirmed when dump's max-version watermark ≥ write's version). Test: burst writer at 1000 writes/s for ≥5 s sees bounded latency.
2. **HLC implementation slips (Phase 1)** — counter not resetting when physical time passes `l`, and restart init from wall clock instead of dump max (node's own writes lose to its own dump). Prevent: adversarial-clock property tests (jumps both directions, stuck clock, post-receive ordering, uniqueness, overflow bump, restart-from-dump-max) — this is Phase 1's core test deliverable.
3. **Tombstone resurrection (Phases 1/2/4)** — dumps must include `Deleted`; `Delete` must mint a fresh version exactly like `Set`; `CreateIfAbsent` succeeds iff no record **or tombstoned**; Watch must notify replicated deletes and lost merges. Round-trip test: delete → dump → restart → Get=NotFound → merge stale pre-delete → still NotFound.
4. **Atomic-dump durability (Phase 2)** — `rename` without parent-dir fsync (ext4 silent loss), temp file in `/tmp` (EXDEV at runtime only), macOS `fsync` not flushing drive cache. Prevent: `os.CreateTemp(filepath.Dir(dumpPath))` + dir fsync after rename + `.prev` rotation in same sequence; decide F_FULLFSYNC policy explicitly; dump failures must reach the `Unavailable` path and metrics.
5. **Own-echo / multi-homed duplicate delivery (Phase 3)** — filter self by `node_id`, never by address; peer table keyed by NodeID with atomic run_id replacement (stale Hello must not flip it); ack tracking keyed `(NodeID, key, version)` so one peer's two addresses share one unacked entry.
6. **Simulation-vs-reality gap (Phases 3/5/6)** — a per-datagram fault model omits per-fragment loss amplification (~40 fragments per 60 KiB datagram → 1% frame loss ≈ 33% datagram loss), kernel RCVBUF overflow, per-interface duplicates. Prevent: real-socket quickstart in CI, randomized seed schedules, simulator fault classes mirroring the documented discard counters.
7. **Watch fan-out blocking (Phase 4)** — one slow client's HTTP/2 flow control can freeze all watchers and the store. Prevent: per-subscriber bounded buffer, non-blocking send under lock, per-stream sender goroutine, server keepalive; Watch is best-effort notification + authoritative `Get`.
8. **Suspend-vs-drop unacked (Phase 3)** — never discard unacked records when the peer table empties (violates the documented loss bound); document unacked-age semantics under sustained Ack loss; assert the set drains after heal, not just final convergence.

Also load-bearing: dump must snapshot under short RLock and marshal/write outside it (never serialize under the lock); `SetReadBuffer(4<<20)` at socket open with the effective granted size logged; HLC future-clock poisoning is mitigated by an actionable drift metric + the snapshot-restore-with-fresh-versions recovery lever (documented in the Phase 6 runbook).

## Implications for Roadmap

The suggested milestone shape from PROJECT.md is validated by all four research files as **dependency-forced**, not merely reasonable: engine ← persistence (HLC init from dump max presupposes dumps) ← replication (restart/recovery scenarios) ← API (gate/Watch/`wait_for_replication` consume replication events); simulator with replication (convergence under chaos is that phase's acceptance criterion). Suggested structure, incorporating the two research refinements and the two feature additions:

### Phase 0: CI Bootstrap & Repo Skeleton
**Rationale:** Gates are cheapest before first feature code; boundary enforcement must exist before any subsystem can violate it.
**Delivers:** go.mod (Go 1.27) + grpc/protobuf deps; tool-directive-pinned dev tools (golangci-lint v2.15.0 + powerman strict config, protoc plugins, govulncheck); Buf config; GitHub Actions (checkout@v7/setup-go@v7 with `go-version-file`, 4-OS race matrix, vet + lint + tidy gate); Dependabot; `api/` proto skeletons with CI-verified generation.
**Avoids:** Architecture drift — land `depguard` boundary rules (`internal/app` may not import `net`/`os`/grpc/`api`; `internal/port` stdlib-only) here.
**Research flag:** None — STACK.md gives the exact recipe; standard patterns, skip research-phase.

### Phase 1: Engine (Versioning + LWW Map)
**Rationale:** Pure leaf, zero I/O; everything else imports it. Property tests live here and run forever after.
**Delivers:** `internal/app/version.go` (HLC Tick/Update, UUIDv7 pack/unpack, byte-order compare, drift signal), `store.go` (LWW map, tombstones, checksums, conflict journal tap — merge returns displaced version), `port.go` + `wire.go` + `main.go` skeleton (~100 lines; building binary, Runtime + serve loop + signal shutdown) — research refinement: land the skeleton here, not later.
**Addresses:** CORE + VERS requirements.
**Avoids:** Pitfalls 5, 6, 7 — adversarial-clock property tests (rapid), tombstone-aware merge-law tests (commutative/associative/idempotent with tombstoned entries), delete-with-fresh-version, byte-order-equals-(l,c,NodeID) property.
**Research flag:** None — CRDT/HLC patterns are well documented; the design doc is authoritative.

### Phase 2: Persistence (Dump + Write Confirmation)
**Rationale:** Depends only on engine types; the write-confirmation contract makes Phase 3 strictly additive.
**Delivers:** `internal/dal/dump` (format with `Deleted`, atomic write + `.prev` rotation + dir fsync, load fallback chain, snapshots), `gate.go` (bounded-coalescing debounce, version-watermark confirmation registry, dump worker unit). **Exit criterion (research refinement):** Set→OK-only-after-dump testable through `port.App` via `Wire` with fake Transport.
**Addresses:** PERSIST requirements.
**Avoids:** Pitfalls 1, 2, 8 + tombstone round-trip (Pitfall 7): burst test at 1000 writes/s ≥5 s with bounded latency; snapshot-outside-lock; temp-file-in-target-dir; kill −9 → main→`.prev`→empty chain test.
**Research flag:** None — standard patterns; the POSIX durability recipe is fully specified in PITFALLS.md. One open decision to settle in planning: macOS F_FULLFSYNC policy (see Gaps).

### Phase 3: Replication + Simulator (the heart of the project)
**Rationale:** Needs engine (versions/merge) and persistence (restart-from-dump). **The simulator lands in this phase, not in hardening** — convergence under chaos is the replication phase's own acceptance criterion; hardening only extends scenario coverage.
**Delivers:** `api/envelope/v1` proto; `internal/out/udp` (socket, codec, CRC, discard classes, `SetReadBuffer` + effective-size log, tight read loop); `internal/app`: `peers.go`, `repl.go`, `sync.go`, `node.go` single-owner loop; `internal/sim` (seeded-chaos virtual network, jumpable virtual clock, quiescence detection, convergence + bounds assertions, seed printed on failure).
**Addresses:** REPL requirements; TEST (simulator foundation).
**Avoids:** Pitfalls 3, 4, 10, 11, 12 — large-value loss scenarios (model fragment-amplified loss), own-echo filtering by identity on real sockets, suspend-not-drop unacked sets, stale-Hello run_id handling.
**Research flag:** **YES — the only phase flagged by ARCHITECTURE.md as having genuinely deep research needs.** Topics: UDP broadcast socket behavior across Linux/macOS, own-echo semantics, buffer sizing for 60 KiB values, `sync_response_rate` mechanics, loopback broadcast addressing (127.255.255.255 vs 255.255.255.255). Recommend `/gsd:plan-phase --research-phase 3`.

### Phase 4: API (gRPC Surface + Binary)
**Rationale:** After replication because the startup gate's "first full sync round" signal, `wait_for_replication` ack events, and replicated Watch events only exist once replication does. Delivers the real user-facing binary.
**Delivers:** `api/supkv/v1` proto (Get/Set/Delete/**List**/Watch, snapshots, `wait_for_replication`); `internal/in/grpc` (error→status mapping, Watch non-blocking fan-out + per-stream senders + keepalive); gate wiring; flags/env config, `--version`; **plus the two research-recommended additions: `List`/prefix RPC and the gRPC standard health endpoint reflecting the startup gate**.
**Addresses:** API requirements + the two feature-gap additions; TOOL.
**Avoids:** Pitfall 9 (slow-consumer isolation), Pitfall 7's Watch-delete-notification item.
**Research flag:** None — standard generated-server work; protoc-gen-go-grpc v1.6.2 semantics already researched in STACK.md.

### Phase 5: Hardening (Failure Suite + Observability + Benchmarks)
**Rationale:** Depends on everything; mostly test/observability code on existing seams; low architectural risk.
**Delivers:** Full failure-scenario suite from PROJECT.md executed in the simulator **plus real-socket runs** (quickstart: two processes, localhost broadcast, kill −9, restart, converge — in CI); randomized-seed fault schedules; metrics endpoint (in-repo Prometheus text exporter: unacked count/age, dump duration/errors, discard classes, merge conflicts, drift); drift monitor loop; conflict journal polish; benchmarks (dump@10k keys with large values, replication throughput, 1000 writes/s bursts, allocation in read loop).
**Addresses:** OBS + TEST completion; QUALITY verification.
**Avoids:** Pitfall 11 (sim-vs-reality gap) head-on; verification of Pitfalls 1, 2, 3, 4, 9, 12 at envelope scale.
**Research flag:** None — standard test/observability work; scenario list is fully enumerated in PROJECT.md and PITFALLS.md's checklist.

### Phase 6: v1.0 (Docs, Examples, Release)
**Rationale:** Success criteria demand a verified quickstart and operator documentation; release tooling (goreleaser optional here) is not runtime risk.
**Delivers:** Quickstart verified on Linux **and** macOS; ops docs: trust model in bold (CRC32-only, trusted segment), drift runbook (fix clock / isolate node / snapshot-restore to outrun poisoned versions), value-size guidance for lossy segments, platform durability matrix, "not a lock service", local-read semantics, TTL explicitly out of scope; examples; release tag.
**Addresses:** v1.0 success criteria; the three documentation actions from FEATURES.md.
**Avoids:** The UX pitfalls (indistinguishable `Unavailable` classes, alarming metrics during normal partitions, hand-edited dumps).
**Research flag:** None.

### Phase Ordering Rationale

- **Dependency-forced, not preference:** engine ← persistence ← replication ← API; each phase's exit criteria are testable through `port.App` with fakes, making the next phase strictly additive (ARCHITECTURE.md Build Order validation)
- **Grouping follows the boundaries:** each phase fills exactly one or two hexagonal components, so no phase touches another's interior — this is what makes the pitfall preventions local to their phase
- **Simulator with replication, not hardening:** convergence under loss/duplication/reordering/partition is a replication property; deferring the sim would leave Phase 3 verified only by unit tests — precisely the gap Pitfall 11 warns about
- **Pitfall-to-phase mapping is exhaustive:** every critical pitfall has a prevention phase and a verification phase (see PITFALLS.md mapping table); nothing is left to "catch later"

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Replication):** UDP broadcast cross-platform behavior, own-echo semantics, 60 KiB datagram buffer/fragmentation realities, `sync_response_rate` mechanics, loopback broadcast for CI — the only phase ARCHITECTURE.md flags as deep-research

Phases with standard patterns (skip research-phase):
- **Phase 0:** exact CI/tooling recipe already specified in STACK.md
- **Phases 1–2:** standard CRDT/HLC/durability patterns; preventions fully specified in PITFALLS.md
- **Phase 4:** standard generated gRPC server work
- **Phases 5–6:** standard test/observability/docs work; scenario lists already enumerated

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All version facts machine-verified 2026-10-31 (proxy.golang.org, GitHub Releases API, Go 1.27.1 GOROOT); practice recommendations cross-checked against official docs; binding go-ultimate conventions |
| Features | MEDIUM | Websearch/webfetch cross-checked against official docs and primary repos (etcd, Consul, Redis, KeyDB, Marmot, Corrosion, rqlite, LiteFS, Alertmanager); no curated HIGH tier available; findings refine rather than reshape the decided scope |
| Architecture | HIGH (project decisions) / MEDIUM (external facts) | Layout/wiring binding via mandated go-ultimate conventions; external precedents read directly from primary sources (CockroachDB hlc, Alertmanager cluster, Marmot, etcd raft, Corrosion) |
| Pitfalls | MEDIUM | Web-sourced findings cross-checked across multiple independent sources; items marked design-derived are LOW until covered by phase tests; macOS-fsync item is LOW-MEDIUM (single-main-source cluster) |

**Overall confidence:** HIGH — the decision-driving areas (stack, architecture) are HIGH because versions are machine-verified and layout is convention-binding; features and pitfalls are MEDIUM but only refine execution of an already-decided design. Residual uncertainty concentrates in Phase 3's real-network behavior, which is exactly where the research flag points.

### Gaps to Address

- **macOS durability policy (Phase 2 planning):** decide explicitly whether to call `F_FULLFSYNC` via `golang.org/x/sys/unix` on Darwin (one fcntl, matches "correctness from proven components", latency cost) or document macOS as best-effort dev/CI platform with Linux as the durability reference. LOW-MEDIUM source confidence — decide in planning, verify in Phase 5 crash scenarios.
- **Fragment-amplified loss modeling (Phase 3):** the simulator must model per-fragment loss for oversized datagrams (or multiply loss probability) or the failure suite will pass while real lossy segments struggle — design the fault-class inventory when planning Phase 3.
- **Loopback broadcast quickstart (Phase 3/6):** use subnet-directed `127.255.255.255`; behavior of `255.255.255.255` on loopback differs across OSes — verify on both Linux and macOS CI runners, not just in simulation.
- **Design-derived pitfalls unverified:** version-watermark confirmation, suspend-vs-drop unacked, HLC init-from-dump-max, create-if-absent-vs-tombstone are analysis of the design doc (LOW) until each phase's mandated tests land — they are already in the phase test lists above.
- **Negative claims are LOW by nature:** "no Go DST framework exists", "no pterodactyl PBT library" — two independent searches each found no counter-evidence; if a DST framework surfaces, reconsider the simulator build (unlikely; in-repo harness is low-risk regardless).

## Sources

### Primary (HIGH confidence)
- proxy.golang.org module listings — grpc v1.85.0, protobuf v1.36.12, golangci-lint v2.15.0, protoc-gen-go-grpc v1.6.2, x/vuln v1.9.0, rapid v1.3.0, uuid v1.6.0 (machine-verified)
- GitHub Releases API — buf v1.75.0, actions/checkout v7.0.1, setup-go v7.0.0, golangci-lint-action v9.3.0
- Go 1.27.1 GOROOT source — `SO_BROADCAST` default-on for `SOCK_DGRAM` on all platforms
- go-ultimate skill v0.15.0 (project-mandated) — layout, wiring, tooling, quality-gate conventions; binding for project decisions

### Secondary (MEDIUM confidence)
- Primary repos read directly: CockroachDB `pkg/util/hlc`, Alertmanager `cluster/`, Marmot, etcd raft README, Corrosion (Rust — design reference only)
- Official docs: etcd/Consul/Redis APIs, grpc.io flow-control & keepalive guides, protobuf.dev best practices, RFC 9562 (UUIDv7), RFC 4963 (reassembly), Go blog "Testing Time"
- Community/practitioner: Polar Signals DST-in-Go, powerman/golangci-lint-strict, Red Hat UDP tuning, PostgreSQL ext4 durability threads, Ted Ts'o delayed-allocation write-up, KeyDB/Marmot/Corrosion/rqlite/LiteFS READMEs and docs

### Tertiary (LOW confidence)
- macOS F_FULLFSYNC behavior cluster (misfra.me + SQLite discussions) — single main source
- Negative existence claims (no Go DST framework; no "pterodactyl" library) — inherently unverifiable
- Design-derived pitfall items (watermark confirm, suspend-vs-drop, restart-init) — LOW until phase tests validate

---
*Research completed: 2026-10-31*
*Ready for roadmap: yes*
