# Phase 4: Replication and Deterministic Simulator - Context

**Gathered:** 2026-11-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Nodes converge to identical state — the pointwise supremum of all writes — over a lossy UDP broadcast segment: Hello discovery, DataReplication/Ack with per-peer retransmission backoff, DataRequest, chunked DataSync full reconciliation, the N-peer table, and datagram discard classification — plus the deterministic simulator that proves the documented convergence bounds under a virtual network that loses, duplicates, reorders and partitions datagrams. Covers REPL-01..REPL-10 and TEST-03. The wire protocol itself is fully designed in `docs/IDEA.md` §Replication protocol — this phase implements it, it does not redesign it. No gRPC/API surface (Phase 5), no failure-scenario hardening beyond the simulator bounds (Phase 6), no metrics endpoint (Phase 6, though discard-class counters must exist as the observability seam).

</domain>

<decisions>
## Implementation Decisions

### Core/adapter seam (what crosses the boundary)
- **D-01:** The port seam carries decoded message structs, not bytes. The UDP adapter fully decodes the datagram (protobuf envelope + body) and hands the core its own message types — plain Go structs, standard-library types only. The send direction mirrors it: the core emits a struct, the adapter encodes and sends. Rationale: protobuf cannot live in `internal/app` (depguard `sup-kv-app-no-io` forbids `google.golang.org/*`), so unlike Phase 3's whole-dump-bytes seam (where the JSON codec was pure), the decode must happen adapter-side. — **Reversibility:** costly — this is the interface Phases 5–6 code against, same contract class as Phase 3 D-02.
- **D-02:** All discard decisions and discard-class counters live in the core. The adapter works mechanically: it verifies CRC (on failure it does NOT decode the body), decodes the envelope, and passes the data plus the check results to the core. The core decides and counts every discard class — own broadcast echo, duplicate `node_id` (logged as configuration error, REPL-08), bad CRC, bad proto version (REPL-09) — so the simulator exercises exactly the discard logic that runs in production.
- **D-03:** The peer table lives in the core; a peer's network address is an opaque handle (a string) stored alongside it. Only the adapter can resolve a handle to a real address. The core never touches `net.Addr`. — **Reversibility:** costly — same seam consumers as D-01.

### Core concurrency shape
- **D-04:** The replication core is a single event-loop goroutine: inbound datagrams, local-write notifications and timer ticks all arrive on one queue, and only the loop mutates replication state (peer table, unacked sets, backoff timers). No side goroutines inside the core. This makes deterministic simulation natural — the simulator chooses the event order and controls time — and throughput is a non-issue at project scale (tens of thousands of keys, a handful of nodes). — **Reversibility:** costly — the simulator architecture is built on it.
- **D-05:** A local write reaches replication by an explicit call from the write's author (tests now, the gRPC handler in Phase 5) — no store subscriptions, no observers, no callbacks. Follows Phase 2 D-01 "results, not callbacks"; the write path stays explicit: store, then broadcast call.

### Deterministic simulator
- **D-06:** The simulator is a test-only facility inside the repo driven by ordinary `go test` — no separate binary. A failing run prints its seed; re-running the failing case is setting the seed via an environment variable and re-running the one test (exact variable name is planner's discretion, following the property-test precedent of Phase 2's `internal/app/internal/testutil`). Nothing simulator-specific ships in the production binary.
- **D-07:** Scenario set = named bound tests + one randomized chaos test. Each documented convergence bound (normal operation one RTT; datagram loss ≤ `ack_timeout_max`; partition heal ≤ 2 × `sync_interval` + response time, REPL-10) gets its own named test asserting that bound precisely, plus one long seeded test where the virtual network randomly loses/duplicates/reorders/partitions — catching failure combinations no one scripted. The virtual network and its fault injection are seeded from the same rng discipline as existing property tests.

### Real-network coverage in this phase
- **D-08:** One real-socket smoke test ships in this phase: two nodes over genuine UDP on localhost — Hello discovery, a write on one node visible on the other. It catches what the simulator cannot (real protobuf serialization, real socket behavior, real addressing). Real-network hostile scenarios remain in Phase 6, per the roadmap split.

### Claude's Discretion
- Package names and locations (a UDP adapter package under `internal/` appearing with this phase per AGENTS.md layout rules; a replication core package inside `internal/app`), exact port interface names/signatures in `internal/port`, the wire `.proto` file location and protobuf code-generation wiring (first `google.golang.org/*` entries in `go.mod` arrive with this phase per the dependency policy), the message-struct field spellings, the seed environment-variable name, the virtual network's internal API, and how timer ticks reach the event loop (the planner follows the Phase 2 D-02 `now`-seam precedent so the simulator controls time).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design and Requirements
- `docs/IDEA.md` — §Replication protocol (UDP) (authoritative protocol design: Envelope, Hello, DataReplication/Ack, DataRequest, DataSync, retransmission backoff, peer table, convergence bounds), §Node lifecycle (startup sync, first Hello), §Parameters (all defaults: `sync_chunk_bytes` 1400, `hello_interval` 1s, `peer_timeout` 5s, `ack_timeout` 200ms, `ack_timeout_max` 5s, `retransmit_interval` 100ms, `sync_interval` 30s, `sync_response_rate` 500/s), §Trust model (CRC32 accepted), §Failure scenarios
- `.planning/REQUIREMENTS.md` — REPL-01..REPL-10 are this phase's requirements; TEST-03 is the simulator requirement; REPL-11 is explicitly deferred (see Deferred)
- `.planning/ROADMAP.md` §Phase Details → Phase 4 — goal and the five success criteria (the verification contract)
- `.planning/PROJECT.md` — Context (decided design, do not re-litigate), Constraints (zero-dependency principle: protobuf enters go.mod only now, tools-only indirect requires were the accepted Phase 1 reading)

### Mandated Conventions
- `AGENTS.md` — layout rules (adapters appear with the phases that need them), dependency policy (gRPC/protobuf enter in the phases that first use them), depguard boundary (`sup-kv-app-no-io`, `sup-kv-port-stdlib-only`)
- `.planning/phases/03-persistence-dump-and-write-confirmation/03-CONTEXT.md` — Phase 3 decisions still binding: D-05 (keep serving after dump failure; remote merges trigger dumps), and the seam precedents D-01/D-02 this phase's D-01 deliberately diverges from (bytes there, structs here — protobuf forces it)
- `.planning/phases/02-engine-hlc-versioning-and-lww-store/02-CONTEXT.md` — Phase 2 decisions still binding: D-01 (results, not callbacks — basis of this phase's D-05), D-02 (`now` seam — the time-injection precedent the simulator relies on)
- go-ultimate skill (v0.15.0, available locally) — engineering and testing policy

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/storage` — the LWW-Element-Map store; `Merge` applies remote records and returns losers (conflict journal input); idempotent/commutative merge means duplicates and reordering from the network cannot affect outcome
- `internal/app/clock` — HLC `Update` on receiving any foreign version (REPL-driven `Update` calls feed the clock); `NewClock(now, initialL, initialC, driftWarn)` already the restart seam
- `internal/app/version` — `RecordVersion` with byte-order `Compare`; every tracked (key, version) pair and every "supersession" check uses it
- `internal/app/internal/testutil` — seeded deterministic generators/rng (Phase 2 property tests) — the discipline D-06/D-07 extend to the simulator
- `internal/app/persist` + `internal/port.DumpStore` — the seam precedent: consumer-side interface in `port`, adapter implements; replication's port seam follows the same shape
- `internal/persistfile` — the existing adapter exemplar outside the app boundary

### Established Patterns
- Depguard gate machine-enforces where the UDP adapter and protobuf must NOT live (`internal/app`), and stdlib-only in `internal/port`
- Strict golangci-lint + go-ultimate from first commit; `make lint` / `go test -race` red blocks merge — new packages inherit this with zero direct runtime requires before this phase
- Package-per-concern split inside `internal/app` (`version`, `clock`, `storage`, `persist`) sets the naming precedent for the replication core package

### Integration Points
- Phase 3: remote merges trigger dumps; the node keeps serving after dump failure (03-CONTEXT D-05)
- Phase 5 (gRPC): the write handler will call store + broadcast explicitly (this phase's D-05 defines that contract); `wait_for_replication` will read per-peer ack state; the startup gate consults first-full-sync state
- Phase 6 (hardening): failure-scenario suite on real networks; observability consumes the discard-class counters and unacked-set gauges this phase must expose as seams

</code_context>

<specifics>
## Specific Ideas

- The determinism argument rides on D-04: the simulator drives the same single event loop the production adapter feeds — there is no separate "simulation mode" code path in the core
- Phase 2's rng/testutil discipline is the explicit model for seed handling: print on failure, re-run by seed (D-06)
- The smoke test (D-08) is deliberately minimal — discovery + one replicated write; everything hostile stays virtual here and real in Phase 6

</specifics>

<deferred>
## Deferred Ideas

- REPL-11 (DataSync unicast fast path for large divergences) — already recorded in REQUIREMENTS.md as conditional future work; only if empty-node recovery proves too slow at real volumes
- A standalone long-run simulator binary (`cmd/sim`) for overnight chaos soaks — rejected for this phase (D-06); reconsider if randomized chaos tests ever need runtimes beyond CI budgets

</deferred>

---

*Phase: 4-Replication and Deterministic Simulator*
*Context gathered: 2026-11-03*
