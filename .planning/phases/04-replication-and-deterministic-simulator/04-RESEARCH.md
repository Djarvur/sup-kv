# Phase 4: Replication and Deterministic Simulator - Research

**Researched:** 2026-11-03
**Domain:** UDP replication protocol (protobuf on UDP broadcast), deterministic discrete-event simulation, Go testing infrastructure
**Confidence:** HIGH (every load-bearing platform claim verified by live probes on this machine, or quoted verbatim from in-repo sources; the only LOW items are macOS CI runner behaviors no Linux box can probe)

## Summary

Phase 4 implements the fully-designed protocol in `docs/IDEA.md` §Replication protocol: a protobuf `Envelope` per UDP datagram, broadcast `Hello` discovery, `DataReplication`/`Ack` with per-peer retransmission backoff, `DataRequest`, chunked `DataSync` anti-entropy, an N-peer table, and datagram discard classification — plus the deterministic simulator (TEST-03) that proves the three documented convergence bounds. The architecture is already locked by 04-CONTEXT D-01..D-08: structs-across-the-seam (not bytes), discards decided in the core, a single event-loop core with no goroutines, an explicit broadcast call from the write's author, a test-only simulator driven by `go test`, and one real-socket smoke test.

The research's most consequential findings are platform facts, all verified by live probes run this session on this Linux box (kernel 7.3.1): (1) two UDP sockets bound `0.0.0.0:P` with `SO_REUSEADDR`+`SO_BROADCAST` set before bind DO all receive broadcast datagrams to `127.255.255.255:P`, `255.255.255.255:P` and the subnet broadcast — and the sender receives its own echo (REPL-09's own-broadcast class is real, not hypothetical); (2) **unicast to a port shared by REUSEADDR sockets is delivered to exactly ONE socket, kernel's choice** — so the same-host smoke test can exercise Hello discovery and broadcast DataReplication visibility, but Ack/DataRequest unicast paths must stay simulator-verified; (3) sockets bound to *specific* addresses (`127.0.0.1:P`, `127.0.0.2:P`) get perfect unicast separation but receive **no broadcast at all** on Linux; (4) loopback broadcast fan-out is a **Linux-only kernel behavior** — macOS `lo0` lacks the BROADCAST flag, so the smoke test needs a pre-flight capability probe with a subnet-broadcast fallback and a documented skip. On the toolchain side: `protoc` 3.21.12 (system) + `protoc-gen-go` v1.36.12 (already in the module graph — `go build -o` and the one-line `go tool` directive both work with **zero go.mod/go.sum changes beyond that directive and the indirect→direct promotion**) generate compiling open-API code; a full 60 KiB record marshals to 61 483 bytes, inside the 65 507-byte IPv4 UDP payload ceiling, and 62 000-byte datagrams transit loopback intact.

For determinism, `testing/synctest` is definitively the wrong tool here — its own package doc states blocking on network I/O is not "durably blocking" and panics as a deadlock, and its guideline says "Avoid using the network" — while D-04's event-loop core plus the Phase 2 D-02 `now`-seam make the simulator goroutine-free and trivially deterministic without it. The two determinism hazards that will actually bite the planner: unsorted Go map iteration anywhere in the core's emit path, and under-specified tie-breaking in the virtual network's event queue.

**Primary recommendation:** implement the core as a synchronous pure state machine (`internal/app/repl`) whose handler methods the production adapter invokes from one goroutine and the simulator calls directly in virtual-time order; commit generated `.pb.go` files (lint-exempt by the vendored config's path pattern) so CI never needs `protoc`; write the smoke test as a capability-probing wildcard+REUSEADDR two-node test asserting discovery and write visibility only.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Core/adapter seam (what crosses the boundary)**

- **D-01:** The port seam carries decoded message structs, not bytes. The UDP adapter fully decodes the datagram (protobuf envelope + body) and hands the core its own message types — plain Go structs, standard-library types only. The send direction mirrors it: the core emits a struct, the adapter encodes and sends. Rationale: protobuf cannot live in `internal/app` (depguard `sup-kv-app-no-io` forbids `google.golang.org/*`), so unlike Phase 3's whole-dump-bytes seam (where the JSON codec was pure), the decode must happen adapter-side. — **Reversibility:** costly — this is the interface Phases 5–6 code against, same contract class as Phase 3 D-02.
- **D-02:** All discard decisions and discard-class counters live in the core. The adapter works mechanically: it verifies CRC (on failure it does NOT decode the body), decodes the envelope, and passes the data plus the check results to the core. The core decides and counts every discard class — own broadcast echo, duplicate `node_id` (logged as configuration error, REPL-08), bad CRC, bad proto version (REPL-09) — so the simulator exercises exactly the discard logic that runs in production.
- **D-03:** The peer table lives in the core; a peer's network address is an opaque handle (a string) stored alongside it. Only the adapter can resolve a handle to a real address. The core never touches `net.Addr`.

**Core concurrency shape**

- **D-04:** The replication core is a single event-loop goroutine: inbound datagrams, local-write notifications and timer ticks all arrive on one queue, and only the loop mutates replication state (peer table, unacked sets, backoff timers). No side goroutines inside the core. This makes deterministic simulation natural — the simulator chooses the event order and controls time — and throughput is a non-issue at project scale (tens of thousands of keys, a handful of nodes). — **Reversibility:** costly — the simulator architecture is built on it.
- **D-05:** A local write reaches replication by an explicit call from the write's author (tests now, the gRPC handler in Phase 5) — no store subscriptions, no observers, no callbacks. Follows Phase 2 D-01 "results, not callbacks"; the write path stays explicit: store, then broadcast call.

**Deterministic simulator**

- **D-06:** The simulator is a test-only facility inside the repo driven by ordinary `go test` — no separate binary. A failing run prints its seed; re-running the failing case is setting the seed via an environment variable and re-running the one test (exact variable name is planner's discretion, following the property-test precedent of Phase 2's `internal/app/internal/testutil`). Nothing simulator-specific ships in the production binary.
- **D-07:** Scenario set = named bound tests + one randomized chaos test. Each documented convergence bound (normal operation one RTT; datagram loss ≤ `ack_timeout_max`; partition heal ≤ 2 × `sync_interval` + response time, REPL-10) gets its own named test asserting that bound precisely, plus one long seeded test where the virtual network randomly loses/duplicates/reorders/partitions. The virtual network and its fault injection are seeded from the same rng discipline as existing property tests.

**Real-network coverage in this phase**

- **D-08:** One real-socket smoke test ships in this phase: two nodes over genuine UDP on localhost — Hello discovery, a write on one node visible on the other. It catches what the simulator cannot (real protobuf serialization, real socket behavior, real addressing). Real-network hostile scenarios remain in Phase 6, per the roadmap split.

### Claude's Discretion

- Package names and locations (a UDP adapter package under `internal/` appearing with this phase per AGENTS.md layout rules; a replication core package inside `internal/app`), exact port interface names/signatures in `internal/port`, the wire `.proto` file location and protobuf code-generation wiring (first `google.golang.org/*` entries in `go.mod` arrive with this phase per the dependency policy), the message-struct field spellings, the seed environment-variable name, the virtual network's internal API, and how timer ticks reach the event loop (the planner follows the Phase 2 D-02 `now`-seam precedent so the simulator controls time).

### Deferred Ideas (OUT OF SCOPE)

- REPL-11 (DataSync unicast fast path for large divergences) — already recorded in REQUIREMENTS.md as conditional future work; only if empty-node recovery proves too slow at real volumes
- A standalone long-run simulator binary (`cmd/sim`) for overnight chaos soaks — rejected for this phase (D-06); reconsider if randomized chaos tests ever need runtimes beyond CI budgets
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REPL-01 | Nodes discover peers via broadcast Hello (`node_id` + `run_id`) with no configuration beyond `broadcast_addr` | Peer table in core (D-03); smoke-test topology probed (wildcard+REUSEADDR receives broadcast on Linux); Hello carries `node_id`+`run_id` per IDEA.md; addr learned as the datagram source handle |
| REPL-02 | Every local change broadcast as DataReplication, tracked per-peer (key, version) until Ack or supersession; exponential backoff `ack_timeout` → `ack_timeout_max`; suspended while no peer is known | Event-loop core (D-04) with explicit `NotifyLocalWrite` (D-05); backoff data-structure sketch below; suspension semantics + drain in simulator tests |
| REPL-03 | Receiver Acks each (key, version) after applying to memory | `storage.Store.Merge` + `persist.Engine` dirty-path entry point (existing comment anticipates it); Ack emission from the core on the inbound-datagram path |
| REPL-04 | Unicast DataRequest for missing/stale key; no reply if unknown | Core emits DataRequest against store snapshot (`Records()` exists, tombstones included); DataRequest→DataReplication unicast reply path in core |
| REPL-05 | Chunked DataSync (sorted, self-sufficient, `sync_chunk_bytes`) on `sync_interval` timer, at startup, on neighbor restart; responses rate-limited by `sync_response_rate` | Chunking algorithm below (sorted keys + proto.Size budget); virtual-time token-bucket rate limiter in the event loop; chunk = 1400 bytes default (IDEA.md parameters, quoted below) |
| REPL-06 | Peer table handles N nodes; peer known after first Hello, gone after `peer_timeout`; ack tracking per-peer | Probed topology generalizes to N (broadcast fans out to all same-port REUSEADDR sockets on Linux); peer-table + per-peer ack-set sketch below |
| REPL-07 | Hello with new `run_id` from known node triggers full sync toward it | `run_id` comparison in core's Hello handler; full sync = DataSync emission toward that peer handle |
| REPL-08 | Duplicate `node_id` dropped + logged as configuration error | Classification algorithm below: own echo vs duplicate node_id distinguished by source-handle comparison (core knows its own handle, D-02/D-03) |
| REPL-09 | Own broadcast echo, bad CRC, bad proto version dropped, classes distinguishable | Probed: own echo is real on Linux; CRC verified adapter-side before body decode (D-02); bad-proto-version check owned by core with a supported-version constant |
| REPL-10 | Convergence bounds proven in simulation: normal one RTT; loss ≤ `ack_timeout_max`; partition heal ≤ 2 × `sync_interval` + response time | Named bound tests over the discrete-event scheduler (D-07); virtual network loses/duplicates/reorders/partitions; rate-limit math for response-time component |
| TEST-03 | Deterministic multi-node simulation, 2..N nodes, virtual network losing/duplicating/reordering/partitioning, seeded runs | simnet design below; testutil rng discipline reuse (`-seed` flag, printed seed); determinism rules (no unsorted map iteration, seq-tiebroken queue) |
</phase_requirements>

## Project Constraints (from AGENTS.md)

- Go version pinned in `go.mod` (`go 1.27`); do not raise without an ADR. Runtime here: go1.27.3 linux/amd64 `[VERIFIED: go version output]`.
- `make lint` / `make test` (`go tool golangci-lint run ./...`, `go test -race ./...`) red blocks merge; CI runs the same targets on the 4-cell gate matrix `[VERIFIED: .github/workflows/ci.yml:34-62]`.
- Depguard boundary (machine-enforced, `.golangci.yml`): `internal/app` may not import `net`, `os`, `syscall`, `google.golang.org/*` (rule `sup-kv-app-no-io`); `internal/port` may import `$gostd` and `github.com/Djarvur/sup-kv` only (rule `sup-kv-port-stdlib-only`) `[VERIFIED: .golangci.yml:68-86]` — quoted verbatim below in Patterns.
- Vendored lint config is byte-unmodified powerman/golangci-lint-strict for exactly 2.15.0; project rules are additive depguard keys only; version bump = re-fetch config.
- No `//nolint` without specific linter + written reason (`nolintlint.require-explanation`, `require-specific`).
- Layout: `cmd/sup-kv/` single binary, no testable logic in `main()`; `internal/app/*` pure packages split per concern; adapter packages under `internal/` appear with the phases that need them; do not create empty adapter directories.
- Dependency policy: zero *direct runtime* requires until this phase; **protobuf enters `go.mod` in Phase 4** (first runtime requires); tools via `go tool <name>` only, never `tools.go`, never bare `go install`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Protocol decisions (discards, peer table, backoff, sync scheduling, rate limits) | Pure core (`internal/app/repl`) | — | D-02/D-03/D-04: single decision point, simulator-executable, depguard-locked purity |
| Protobuf encode/decode, CRC, envelope framing | UDP adapter (outside `internal/app`) | — | D-01: protobuf forbidden in app; adapter mechanical |
| Socket lifecycle (bind, SO_REUSEADDR/SO_BROADCAST, read loop, send) | UDP adapter | — | I/O tier; goroutines live here (D-04: none in core) |
| Event dispatch (what the core handles next) | Core's synchronous handlers | Adapter goroutine (production) / scheduler (sim) | D-04: one loop mutates state; drivers only pump events |
| Time source | Injected `now` seam (D-02 precedent) | Adapter `time.Now` / simulator virtual clock | Required for wall-clock-free bound assertions |
| Record application (merge + dirty + Ack trigger) | `persist.Engine` remote path (to be exported) | `storage.Store` directly in simulator | 03-CONTEXT D-05 remote merges trigger dumps; sim asserts store state without dumps |
| Convergence proof | Simulator (test-only package) | — | D-06/D-07 |
| Real-network sanity | Smoke test (test-only) | — | D-08; runs in CI gate cells incl. macOS |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/protobuf | v1.36.12 (already pinned `// indirect` in go.mod; this phase promotes it to direct) | Wire codec for the Envelope and five message types | The official Go protobuf runtime; `github.com/golang/protobuf` is its deprecated ancestor and is already denied by the vendored depguard config ("Should be replaced by google.golang.org/protobuf package" `[VERIFIED: .golangci.yml:57-58]`) |
| protoc (system binary) | 3.21.12 (`libprotoc 3.21.12` on this machine) | Descriptor compiler driving protoc-gen-go | Only needed on developer machines that regenerate; **CI never needs it** because generated `.pb.go` is committed |
| google.golang.org/protobuf/cmd/protoc-gen-go | v1.36.12 via one `tool` directive line | Go codegen plugin, version-pinned by go.mod | Same module as the runtime — zero new modules, zero go.sum churn (probed) |
| Standard library only, everything else | go 1.27 | `net`, `hash/crc32` (IEEE — matches `RecordChecksum`), `crypto/rand` (run_id), `container/heap` or sorted slices, `math/rand/v2` (testutil seeds) | Project principle: "everything else … implemented in-repo — each is tens of lines against a known algorithm" `[VERIFIED: docs/IDEA.md:156-160]` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/app/internal/testutil` | in-repo | Seeded rng (`NewRNG`, `-seed` flag, printed seed), generators | Simulator seeding (D-06/D-07) — exact reuse of the Phase 2 discipline |
| `testing/synctest` | go1.27 stdlib | — | **NOT USED — rejected.** Package doc: blocking on network I/O is not durably blocking and panics as deadlock; guideline "Avoid using the network. Use a fake network implementation as needed." `[VERIFIED: go doc testing/synctest, local]`. The event-loop core needs no goroutines, so synctest solves nothing here; the smoke test uses real sockets, which synctest cannot host |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| protoc + protoc-gen-go (committed output) | buf (`github.com/bufbuild/buf/cmd/buf` as go tool) | buf replaces system protoc with a pinned Go tool but drags a large new module graph into go.mod; with committed generated files the system protoc is a dev-machine-only concern — not worth the graph |
| protobuf messages | hand-rolled binary codec | Violates the project's minimal-dependency reading — protobuf IS the one sanctioned dependency (IDEA.md product principles) |
| UDP broadcast (design-locked) | multicast discovery | Multicast would fix same-host fan-out portably (per-socket membership delivers to all, specific binds included) but REPL-01/IDEA.md lock broadcast; redesign is out of scope this phase. Worth noting for Phase 7's one-host quickstart (DOCS-01), which hits the same same-host ambiguity |
| Discrete-event scheduler (sim) | `testing/synctest` bubble | Rejected above — synctest virtualizes time but not cross-node event ordering; determinism of the *fault schedule* requires the queue anyway |

**Installation:** none. No new module downloads: `go mod tidy` after importing `google.golang.org/protobuf` promotes the existing `v1.36.12 // indirect` to direct and (if the tool directive is added) adds one line to the `tool` block — probe-verified byte-diff of go.mod was exactly that one line, go.sum unchanged.

**Version verification (this session):** `protoc-gen-go v1.36.12` built and executed from the repo's module graph `[VERIFIED: /tmp build probe]`; generated code compiled and round-tripped under go1.27.3; `libprotoc 3.21.12` present at `/usr/bin/protoc`; generated file carries `// Code generated by protoc-gen-go. DO NOT EDIT.` header.

## Package Legitimacy Audit

Go modules are outside the npm/PyPI/crates registry gate; the equivalent verification here is stronger — the exact module version is already pinned in the repo's own `go.sum` and was built and executed locally this session.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| google.golang.org/protobuf v1.36.12 | Go module proxy | mature (google.golang.org since 2020; module present in repo graph since Phase 1 tooling) | n/a (Go module) | github.com/protocolbuffers/protobuf-go | OK (already in repo go.sum; probed) | Approved |
| protoc 3.21.12 | system package (`/usr/bin/protoc`) | 2022 release | n/a | github.com/protocolbuffers/protobuf | OK (probed locally) | Approved — dev machines only; CI unaffected |

**Packages removed due to [SLOP] verdict:** none (no new packages adopted).
**Packages flagged as suspicious [SUS]:** none.

No postinstall-script class risk applies (Go toolchain). `github.com/golang/protobuf` (the deprecated module, already `// indirect` for tooling) must NOT be imported directly — depguard denies it `[VERIFIED: .golangci.yml:57-58]`.

## Architecture Patterns

### System Architecture Diagram

```
                       ┌──────────────────────────── production process ────────────────────────────┐
                       │                                                                             │
  local write author ──► store write (persist.Engine.Set) ──► explicit NotifyLocalWrite(key, ver) ──┤ (D-05)
  (test now, gRPC P5)  │                                                                      │      │
                       │                                                                      ▼      │
                       │   ┌────────────── event queue (chan) ──────────────┐   ┌─── repl.Core ───▼───┐
                       │   │ • InboundDatagram   (adapter read goroutine)   │──►│ pure state machine: │
                       │   │ • timer wakeups     (adapter timer goroutine)  │   │ peer table, pending │
                       │   │ • NotifyLocalWrite  (write author, D-05)       │   │ (key,version)+acks, │
                       │   └────────────────────────────────────────────────┘   │ backoff, sync       │
                       │                     ▲                                  │ chunks, rate limit, │
                       │                     │ one loop goroutine calls         │ discard classes,    │
                       │                     │ core handlers (D-04)             │ counters (D-02)     │
                       │   ┌───────────── UDP adapter (internal/, has net+protobuf) ──┐             │
                       │   │ encode port struct → Envelope proto + CRC32 → send       │◄────────────┘
                       │   │ receive → CRC check → (ok? decode body) → port struct    │─── Outbound: ─►
                       │   │ SO_REUSEADDR + SO_BROADCAST, ReadFromUDP(65507 buf)     │  broadcast /   │
                       │   │ source addr → opaque handle string (D-03)               │  unicast-to-   │
                       │   └──────────────────────────────────────────────────────────┘  handle        │
                       └─────────────────────────────────────────────────────────────────────────────────┘

                       ┌──────────────── test binary (simulator, no goroutines) ─────────────────────┐
                       │  seeded rng (testutil) → fault schedule + traffic generator                  │
                       │        │                                                                   │
                       │        ▼                                                                   │
                       │  priority queue (virtualTime, seq) ── next event ──┬──► node1.Core.Handle*  │
                       │  virtual network: drop / dup / delay / partition   ├──► node2.Core.Handle*  │
                       │  delivery = schedule(HandleDatagram at t+delay)    ├──► ...nodeN...         │
       assertions ◄────│  convergence windows measured on the virtual clock └──► Advance(now) ticks  │
                       └─────────────────────────────────────────────────────────────────────────────┘
```

Entry points: local writes (explicit call), inbound datagrams (adapter read loop / sim scheduler), timers (adapter wakeup / sim `Advance`). Decision points: CRC + proto-version + node_id classification (core), backoff/retransmit scheduling, rate-limited response queue. External boundary: the UDP socket is the only service edge.

### Recommended Project Structure

```
internal/
├── app/
│   ├── repl/                  # replication decision core (pure; depguard-guarded):
│   │                          #   Core state machine, peer table, pending/ack sets,
│   │                          #   backoff, DataSync chunking, rate limiter, discard classes+counters
│   │   └── *_test.go          # unit tests: classification, chunking, backoff table, table lifecycle
│   └── internal/
│       └── simnet/            # test-only discrete-event scheduler + fault-injecting virtual network
│                              # (normal package linked into test binaries, testutil precedent;
│                              #  no-test-libs-in-prod exempts the test*/ dir per rng.go's comment)
├── port/
│   └── repl.go                # message structs + inbound/outbound datagram types (stdlib+own-module
│                              # types only) — consumer-side seam like port.DumpStore
└── udprepl/                   # UDP adapter: protobuf codec, CRC32, socket (SO_REUSEADDR+SO_BROADCAST),
    └── replproto/             #   goroutines feeding the core; owns nothing decided
        ├── supkv.proto        #   wire schema (option go_package → this package)
        └── supkv.pb.go        #   COMMITTED generated code (lint-exempt: matches '\.[\w-]+\.go$')
```

Names are planner's discretion per CONTEXT (Claude's Discretion); the shape above is the recommendation. Key placement constraint: generated protobuf code must NOT live under `internal/app/` — the depguard glob `'**/internal/app/**'` would deny its own imports; the adapter subtree keeps it legal `[VERIFIED: .golangci.yml:68-71]`.

### Pattern 1: Pure synchronous core, pluggable drivers

**What:** the replication core exposes synchronous handler methods and holds all state without locks (only one goroutine ever calls in, per D-04). Production adapter and simulator are two drivers that both pump the same methods.

**When to use:** always in this phase — it *is* the determinism argument (CONTEXT §specifics: "there is no separate 'simulation mode' code path in the core").

**Example:**

```go
// Sketch — planner owns exact names/signatures (Claude's Discretion).
// internal/app/repl — no net, no os, no goroutines, no time.Now.

type Core struct {
    nodeID uint16
    runID  [16]byte
    ownAddr string // opaque handle, D-03 — compared against inbound source to split
                   // own-echo from duplicate-node_id discards (REPL-08/09)
    // peers, pending, backoff, sync state, discard counters ...
}

// HandleDatagram processes one inbound datagram; crcOK false means the adapter
// rejected the CRC and did NOT decode the body (D-02).
func (c *Core) HandleDatagram(now time.Time, in port.InboundDatagram, out *port.Outbox)

// NotifyLocalWrite is the explicit author call after the store write (D-05).
func (c *Core) NotifyLocalWrite(now time.Time, key string, ver version.RecordVersion, out *port.Outbox)

// Advance fires due timers (hello, retransmit, peer timeout, sync, rate-limited
// responses) and returns the next deadline for the driver's sleep/schedule.
func (c *Core) Advance(now time.Time, out *port.Outbox) (nextDeadline time.Time)
```

The `Outbox` accumulator (rather than callbacks) keeps "results, not callbacks" (Phase 2 D-01): the driver drains it after each call — adapter sends, simulator schedules with faults.

### Pattern 2: Envelope framing — CRC before body decode

**What:** the datagram is an outer envelope whose body travels as bytes; the adapter decodes the outer message, verifies CRC over the body bytes, and only then decodes the typed body. This is the only framing that satisfies D-02's "on failure it does NOT decode the body".

**When to use:** the wire format design (exact field spellings are planner's discretion).

**Example:**

```protobuf
// internal/udprepl/replproto/supkv.proto — sketch; field numbers/labels planner's call.
syntax = "proto3";
package supkv.v1;
option go_package = "github.com/Djarvur/sup-kv/internal/udprepl/replproto;replproto";

message Envelope {
  uint32 proto_version = 1;
  uint32 node_id       = 2;
  bytes  run_id        = 3;   // 16 bytes
  uint32 crc32         = 4;   // IEEE CRC32 over body bytes (hash/crc32, IEEE table —
                              // same polynomial family as storage.RecordChecksum)
  bytes  body          = 5;   // marshaled Body wrapper; decoded only after CRC passes
}

message Body {                 // the wrapper unmarshaled adapter-side after the CRC gate
  oneof msg {
    Hello            hello            = 1;
    DataReplication  data_replication = 2;
    Ack              ack              = 3;
    DataRequest      data_request     = 4;
    DataSync         data_sync        = 5;
  }
}
```

Probe-verified generation + round-trip with exactly this shape (open API: direct exported fields plus getters — `protogen:"open.v1"`, not the opaque API) `[VERIFIED: /tmp codegen probe]`. The v1.36.12 default API is open; keep the default — direct field access maps naturally onto the port structs (opaque would force setter ceremony adapter-side).

### Pattern 3: Adapter socket setup (Linux/macOS-probed semantics)

**What:** bind with `SO_REUSEADDR` + `SO_BROADCAST` set *before* bind via `net.ListenConfig.Control` — the only combination that (a) allows two same-host nodes to share the port, (b) receives broadcast, (c) still permits unicast send.

```go
// Source: probe run this session against kernel 7.3.1-1-t2-noble (pattern verified end-to-end).
lc := net.ListenConfig{Control: func(network, address string, c syscall.RawConn) error {
    return c.Control(func(fd uintptr) error {
        if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
            return err
        }
        return syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
    })
}}
conn, err := lc.ListenPacket(ctx, "udp4", bindAddr) // bindAddr "0.0.0.0:7900"
...
buf := make([]byte, 65507) // max IPv4 UDP payload 65535-20-8; Go truncates oversize
                           // reads SILENTLY — a small buffer corrupts, it does not error
n, src, err := conn.ReadFromUDP(buf)
```

**Probed platform matrix (this machine, wildcard x2 SO_REUSEADDR unless noted)** `[VERIFIED: live UDP probes]`:

| Send → bind pattern | Delivered to |
|---|---|
| bcast `127.255.255.255:P` → two `0.0.0.0:P` | BOTH sockets, sender included (own echo real) |
| bcast `255.255.255.255:P` → two `0.0.0.0:P` | BOTH sockets |
| subnet bcast (`192.168.7.255:P`) → two `0.0.0.0:P` | BOTH sockets |
| unicast `127.0.0.1:P` → two `0.0.0.0:P` | EXACTLY ONE socket — kernel picks (probe: most recently bound) |
| bcast → specific binds `127.0.0.1:P` + `127.0.0.2:P` | NEITHER receives |
| unicast → specific binds | clean per-address separation (only the addressee) |
| 62 000-byte datagram, loopback | delivered intact, single read |

macOS: `lo0` lacks the BROADCAST flag; loopback broadcast fan-out is Linux-only kernel behavior (FreeBSD does not apply broadcast logic on loopback either) `[CITED: serverfault.com/questions/554503, lkoskela.fi/blog/2025-10-19-udp-loopback-bcast]`. BSD-classic semantics report broadcast delivered to all SO_REUSEADDR sockets when it works at all `[CITED: stackoverflow.com/questions/5077887]`.

### Pattern 4: Deterministic virtual network + seeded chaos (TEST-03, D-06/D-07)

**What:** a discrete-event scheduler with a strict total order — `(virtualTime, seq)` where `seq` is a monotonically increasing insertion counter. Node = `repl.Core` + real `storage.Store` with a fake `now` (Phase 2 fake-clock precedent: build time with `time.UnixMilli`, no monotonic reading). Faults are decided **at send time** by the seeded rng (drop / deliver(+delay) / duplicate(+delay)) against per-ordered-node-pair partition intervals; broadcast datagrams are fanned out to each recipient with an independent fate draw. Determinism contract: same seed → identical event trace.

**Seeding:** reuse `testutil.NewRNG` verbatim — it registers `-seed` and prints the exact replay line:

```go
// Source: internal/app/internal/testutil/rng.go:35,41-51 [VERIFIED: read this session]
var seedFlag = flag.Int64("seed", 0, "replay a property test with a fixed seed")
...
t.Logf("property seed: %d,%d (replay: go test ./... -run '%s' -args -seed=%d)", hi, lo, t.Name(), seedNano)
```

CONTEXT D-06 names an *environment variable* as the re-run hook ("setting the seed via an environment variable") while the existing precedent is the `-seed` **flag** — the planner must reconcile: recommended reading is "follow the testutil precedent exactly" (a flag is the standard replay hook per rng.go's own comment); an env var (e.g. `SUPKV_SIM_SEED`) read by a chaos test that has no `testing.T`-flag access is equally defensible. Either way the failure output must print the exact re-run command.

### Pattern 5: Discard classification in the core (D-02, REPL-08/REPL-09)

**What:** the core owns the classes and their counters. Algorithm sketch for the two node_id-based classes (they share a trigger and split on the source handle):

1. `crcOK == false` → class `badCRC` (adapter sent no body — D-02).
2. `envelope.ProtoVersion != supported` → class `badProtoVersion` (a body that fails to decode under the supported version lands here too).
3. `envelope.NodeID == own nodeID`:
   - source handle == own handle → class `ownBroadcastEcho` (the probed Linux behavior: the sender's own broadcast really does come back);
   - source handle differs → class `duplicateNodeID` — *configuration error, logged* (REPL-08). A foreign node configured with our node_id is exactly this signature.
4. otherwise → protocol processing.

The core therefore needs its own handle at construction (D-03 keeps it an opaque string). Counters per class are the Phase 6 observability seam (OBS-02/OBS-03 consume them later).

### Pattern 6: Pending set, per-peer acks, backoff

**What:** one pending entry per key holding the latest local version (older versions are superseded by construction — the store already replaced them); per-peer "not-yet-acked" membership; one backoff timer per key (retransmission is broadcast — all peers share each send).

- `pending map[string]entry{ver, nextRetry, backoff}` — `backoff` doubles from `ack_timeout` (200ms), capped at `ack_timeout_max` (5s): 200→400→800→1600→3200→5000ms; retries continue at the cap until Ack/supersession (REPL-02 "until").
- `peer.acked`-negation: a peer leaves the entry's wait-set on its Ack for exactly `(key, version)`; entry drains when every currently-known peer has acked.
- Peer eviction (`peer_timeout`, 5s) removes the peer *and* its wait-set membership; zero known peers → timers pause, entries retained (suspension, REPL-02 "suspending (not dropping)").
- Supersession: a newer local write replaces the entry (reset backoff); a remote strictly-greater version (via Merge or DataSync compare) drops it.
- Scale check: at 10k pending keys a full scan per `Advance` is 10k map probes every 100ms tick — trivially fast at project scale; no heap needed. Keep the linear scan (ponytail: no unrequested abstraction).

### Pattern 7: DataSync chunking + rate-limited responses

**What:** sorted key→version listing, packed into chunks whose *serialized* size stays ≤ `sync_chunk_bytes` (1400): accumulate `[]Entry` sorted by key, checking `proto.Size` of the candidate chunk; flush when adding the next entry would exceed the budget. "Chunks self-sufficient" reads as: each chunk is independently interpretable — no continuation state, any single chunk processable `[ASSUMED — design reading, see Open Questions]`. Responses (`DataRequest` for missing/stale, unicast `DataReplication` for newer) go through a virtual-time token bucket at `sync_response_rate` (500/s = one per 2ms): the queue drains inside `Advance` in deterministic order (sorted by key), never `time.Sleep`.

**Response-time bound math** (feeds REPL-10's third bound): an empty node fed K stale keys takes ≥ K/500 seconds of replay — bound tests should size stores so the rate-limited term is small and assertable (tens–hundreds of keys, not 10k).

### Anti-Patterns to Avoid

- **Simulation-mode code in the core** — the determinism argument (CONTEXT §specifics) is that the simulator drives the *same* handlers; any `if simulating` branch destroys it.
- **Unsorted map iteration in the core's emit path** — Go randomizes map order per iteration; one unsorted range over peers/keys makes the chaos test irreproducible. Everywhere the core produces ordered output (DataSync entries, retransmit batches, rate-limited queues), iterate `slices.Sorted(maps.Keys(...))` (the `persist.Restore` precedent does exactly this `[VERIFIED: internal/app/persist/engine.go:465]`).
- **Real time or goroutines in the core** — `time.Now`, `time.Sleep`, `go func` inside `internal/app/repl` break simulation; the `now`-seam (Phase 2 D-02) and driver-owned goroutines are the sanctioned shapes.
- **Small ReadFromUDP buffer** — Go silently truncates UDP reads (no MSG_TRUNC surfaced); anything under 65507 risks quiet corruption that surfaces only as bad-CRC discards.
- **Sending broadcast without SO_BROADCAST** — Linux returns EACCES ("permission denied") on broadcast sends; the socket option is mandatory (probe pattern above).
- **Relying on unicast to a shared port** — kernel picks one recipient (probed); never assert Ack delivery between two same-host nodes sharing a port.
- **Two event sources mutating core state** — only the loop calls handlers (D-04); the adapter must never touch core state from its read goroutine directly.
- **Regenerating protobuf in CI** — generated code is committed; CI runs `go test -race ./...` and lint only (both proven green paths need no protoc).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Wire codec | Custom binary marshal | google.golang.org/protobuf | The one sanctioned dependency (IDEA.md principles); schema evolution, cross-language tooling, known size semantics (probe: 60KiB record = 61 483 bytes) |
| Integrity checksum | Custom hash | `hash/crc32` with `crc32.IEEETable` | Stdlib; matches the existing record checksum's polynomial (`storage.RecordChecksum` uses `crc32.IEEETable` `[VERIFIED: internal/app/storage/store.go:106-109]`) |
| UUID packing for run_id | New UUID format | 16 random bytes via `crypto/rand` + RFC 9562 v4 bits (6 lines) — or simply opaque 16 random bytes | run_id needs uniqueness-per-process-start, not HLC semantics; `crypto/rand` is stdlib and depguard-legal even in app (deny list is `net`, `os`, `syscall`, `google.golang.org/`) `[VERIFIED: .golangci.yml:72-78]` |
| Seeded randomness / seed replay | New mechanism | `testutil.NewRNG` + `-seed` | D-06 says "following the property-test precedent" — the discipline exists and prints its replay line |
| Discrete-event priority queue | Timer wheels / heaps of timers | `(virtualTime, seq)` priority queue over `container/heap` or sorted slice | Ties must break deterministically; wall-clock timers can't guarantee order |

**Key insight:** the product principle is explicit — "everything else (HLC, UUIDv7 packing, LWW map, UDP reliability layer) implemented in-repo — each is tens of lines against a known algorithm" `[VERIFIED: docs/IDEA.md:156-160]`. The reliability layer (backoff, ack tracking, chunking, rate limiting) is *supposed* to be hand-written here — against the algorithm, with the simulator as its proof. The only true don't-hand-roll items are the codec and CRC.

## Common Pitfalls

### Pitfall 1: Non-determinism leaking into the simulator
**What goes wrong:** chaos test passes locally, fails on CI with the same seed — or fails locally a second time with the same seed.
**Why it happens:** unsorted map iteration anywhere the core produces output; equal-timestamp events delivered in queue-insertion order that depends on goroutine scheduling; iteration over peers in map order when generating retransmits.
**How to avoid:** sorted iteration for every emit path; `(time, seq)` total order in the scheduler; zero goroutines in the core; seeded rng as the *only* randomness (fault draws at send time).
**Warning signs:** a chaos test that fails spuriously; a replay that diverges from the original run.

### Pitfall 2: Same-host smoke test asserting unicast delivery
**What goes wrong:** the smoke test asserts node B's Ack reaches node A; it fails (or worse, silently passes on the machine where the kernel happened to pick A) — two nodes share one port on one host, and unicast to that port reaches exactly ONE socket, kernel's choice (probed).
**Why it happens:** Linux/BSD deliver unicast to a single best-match socket even when SO_REUSEADDR fans broadcast out to all.
**How to avoid:** smoke test asserts only broadcast-delivered behaviors (Hello discovery, DataReplication visibility, own-echo discard count); Ack/DataRequest/unicast paths are proven in the simulator; document the same-host limitation in the adapter's doc comment.
**Warning signs:** flaky smoke test on one OS only; Acks that "sometimes" arrive.

### Pitfall 3: Loopback broadcast assumed portable
**What goes wrong:** the smoke test passes on Linux CI cells, hangs on `macos-15` — lo0 has no BROADCAST flag on Darwin; loopback broadcast fan-out is Linux-only.
**How to avoid:** pre-flight capability probe in the smoke test (bind both sockets, send one probe broadcast, wait ~200ms); on failure fall back to the first broadcast-capable non-loopback IPv4 interface's subnet broadcast; if still nothing, `t.Skipf` with an explicit platform note (a loud, documented skip — not silence; the simulator carries the correctness burden, D-08 keeps the smoke minimal).
**Warning signs:** gate (macos-15) cell timing out while ubuntu cells are green.

### Pitfall 4: Truncated reads with an undersized datagram buffer
**What goes wrong:** 60KiB records never replicate; they show up as bad-CRC discards.
**Why it happens:** Go's `ReadFromUDP` truncates to the buffer size with NO error and no MSG_TRUNC flag; a 4096-byte buffer silently clips.
**How to avoid:** one 65 507-byte buffer for the socket's lifetime (probe: a 62 000-byte datagram arrives intact on loopback).
**Warning signs:** discard counter climbing exactly on large values.

### Pitfall 5: Own echo treated as a duplicate-node_id configuration error
**What goes wrong:** every node logs "duplicate node_id" on every Hello it broadcasts — because the sender's own broadcast loops back (probed: the sender really does receive its own datagram).
**Why it happens:** both classes trigger on `envelope.NodeID == own nodeID`; they split only on the source handle (Pattern 5).
**How to avoid:** compare the inbound source handle against the own-handle constructor state before classifying.
**Warning signs:** configuration-error logs on a healthy two-node cluster.

### Pitfall 6: protobuf API-style surprises in generated code
**What goes wrong:** the executor writes struct literals or copies generated structs; or regenerates with different flags and the API changes underneath.
**Why it happens:** protoc-gen-go v1.36.12 generates the OPEN API by default (`protogen:"open.v1"`, exported fields + getters — probe-verified), but opaque-API guidance circulates for newer toolchains; oneof wrappers need the wrapper type (`Envelope_Hello{...}`) when constructing.
**How to avoid:** pin the plugin via the go.mod tool directive (probe: one line, no other changes); keep the default API; commit generated code; construct oneof bodies via their wrapper types (probe snippet shows the shape).
**Warning signs:** compile errors mentioning unexported `state` fields; regeneration diffs that touch every struct.

### Pitfall 7: Persist engine never dumping replicated writes
**What goes wrong:** remote merges land in the store but never reach disk — PERS-04's on-disk guarantee silently regresses.
**Why it happens:** the engine's write paths dirty it only on local writes today; the seam for remote merges is documented but not exported: "only local writes register waiters, and Phase 4's remote merges will reuse the dirty path without waiters" `[VERIFIED: internal/app/persist/engine.go:384-386]` — `markDirty` is currently unexported `[VERIFIED: engine.go:539-546]`.
**How to avoid:** the phase adds an exported remote-apply entry to the engine (merge via store + dirty, no waiter) and the core calls it on every applied DataReplication/DataSync result; unit-test that a remote merge triggers a dump.
**Warning signs:** dump files on a node missing records its store serves.

### Pitfall 8: Unbounded retransmit storm after partition with pending backlog
**What goes wrong:** after a heal, every pending entry fires its retransmit in the same `Advance` tick — one giant burst broadcast.
**Why it happens:** all backoff timers expire during the partition.
**How to avoid:** retransmits within one tick are naturally batched into one broadcast each (same key set, one datagram per record — the design is one record per DataReplication datagram, so the burst is N datagrams; acceptable at project scale, and the drain bound is what REPL-10's third bound test asserts). Keep the emission order sorted for determinism (Pitfall 1).
**Warning signs:** chaos test convergence exceeding the documented heal bound.

### Pitfall 9: wall-clock dependence in bound assertions
**What goes wrong:** simulator bounds tested with real sleeps — slow on CI, flaky under load, and 2×30s sync_interval bounds make the suite crawl.
**Why it happens:** default `sync_interval` is 30s; a naive test sleeps 60+s.
**How to avoid:** all bounds measured on the virtual clock (D-07's point); 2×sync_interval at virtual-ms granularity is microseconds of CPU per simulated second.
**Warning signs:** simulator tests taking seconds of wall time.

## Code Examples

### Codegen invocation (Makefile `proto` target — dev machines only)

```make
# Source: probe-verified commands (protoc 3.21.12 + go tool pinning), this session.
PROTO_FILES := internal/udprepl/replproto/supkv.proto

.PHONY: proto
proto:
	go build -o .tmp/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	protoc --plugin=.tmp/protoc-gen-go \
	       --go_out=. --go_opt=paths=source_relative $(PROTO_FILES)
```

With the `tool` directive `google.golang.org/protobuf/cmd/protoc-gen-go` added to go.mod's existing tool block, the plugin version is go.mod-pinned; the `go build -o` materializes the binary where protoc's `--plugin` wants it (satisfies the AGENTS.md "tools via go tool/go.mod pinning" policy; protoc itself is the system driver). Probe diff of go.mod was exactly one line in the tool block; go.sum unchanged; `go mod tidy` additionally promotes `google.golang.org/protobuf v1.36.12 // indirect` → direct.

### Wire-format probe (round-trip incl. max-size record) — verified output

```go
// Source: /tmp codegen+runtime probe this session (protoc-gen-go v1.36.12, open API).
env := &replproto.Envelope{ProtoVersion: 1, NodeId: 7, RunId: runID16, Crc32: 42,
    Body: &replproto.Envelope_Hello{Hello: &replproto.Hello{}}}
b, err := proto.Marshal(env)          // 26 bytes, err=nil
// oneof construction uses the wrapper type: &Envelope_DataReplication{DataReplication: ...}
// full record, 61440-byte value:
//   proto.Marshal → 61483 bytes < 65507 (max IPv4 UDP payload)
```

### Design parameters (verbatim, the numbers tests assert against)

```text
// Source: docs/IDEA.md:144-148 [VERIFIED: read this session]
`node_id`, `bind_addr`, `broadcast_addr`, `dump_path` (required);
`value_max_bytes` 61440; `sync_chunk_bytes` 1400; `hello_interval` 1s;
`peer_timeout` 5s; `ack_timeout` 200ms; `ack_timeout_max` 5s;
`retransmit_interval` 100ms; `sync_interval` 30s; `sync_response_rate` 500/s;
`startup_gate` 1s; `dump_debounce` 50ms; `dump_max_wait` 1s; `drift_warn` 60s.
```

```text
// Source: docs/IDEA.md:99-100 [VERIFIED: read this session]
- Convergence bounds: normal operation one RTT; datagram loss ≤ `ack_timeout_max`;
  after a partition heals ≤ 2 × `sync_interval` + response time.
```

### Depguard boundary (verbatim — what machine-enforces the placement)

```yaml
# Source: .golangci.yml:66-86 (project additions inside linters.settings.depguard.rules)
# [VERIFIED: read this session]
        sup-kv-app-no-io:
          files:
            - '**/internal/app/**' # leading **/ mandatory (depguard globs match absolute paths); quotes mandatory (leading * breaks YAML)
            - '!$test' # exempts *_test.go from the boundary
          deny:
            - pkg: net # plain string = prefix match (catches net, net/http, net/url)
              desc: internal/app is the pure decision core - no networking
            - pkg: os
              desc: internal/app is the pure decision core - no filesystem
            - pkg: syscall
              desc: internal/app is the pure decision core - no syscalls
            - pkg: google.golang.org/
              desc: internal/app is the pure decision core - no gRPC/protobuf
        sup-kv-port-stdlib-only:
          files:
            - '**/internal/port/**'
            - '!$test'
          list-mode: strict
          allow:
            - $gostd
            - github.com/Djarvur/sup-kv # own module path allowed per D-11's deny-third-party wording
```

Two consequences: `internal/port` may reference `version.RecordVersion`/`storage.Record` (own module path allowed — the Phase 01 decision's stated purpose: "port-to-app type references stay legal" `[VERIFIED: STATE.md Phase 01 decisions]`); and the lint gate skips generated files via `paths: - \.[\w-]+\.go$` under `issues.exclusions` plus `generated: strict` under formatters `[VERIFIED: .golangci.yml:491-492,519-521]` — a committed `supkv.pb.go` is exempt.

### Existing seams the core consumes (verbatim signatures)

```go
// Source: internal/app/storage/store.go:30-36,189-195,243-244 [VERIFIED: read this session]
type Record struct {
	Version  version.RecordVersion
	Data     []byte
	Deleted  bool
	Checksum uint32
}
func (s *Store) Merge(key string, rec Record) (conflicts []Conflict, drift time.Duration, err error)
func (s *Store) Records() map[string]Record  // full snapshot, tombstones included — the DataSync compare source
```

```go
// Source: internal/app/persist/engine.go:384-386 [VERIFIED: read this session]
// ...any state change dirties; only local writes register waiters, and Phase 4's
// remote merges will reuse the dirty path without waiters.
```

The engine seam to add this phase (shape per that comment): an exported `ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error)` that merges and marks dirty with no waiter — the simulator needs none of it (it drives `storage.Store` directly with a fake clock and asserts store convergence; dumps stay Phase-3 territory).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `github.com/golang/protobuf` | `google.golang.org/protobuf` | old module deprecated years back; already denied by the vendored strict config | Import the new module only; the deprecated one stays an unused tooling indirect |
| protoc-gen-go closed/legacy API | Open API default in v1.36.12 (probe: `protogen:"open.v1"`); opaque API behind flags/newer toolchains | v1.36 era | Generated structs have exported fields + getters — direct field mapping to port structs |
| GOEXPERIMENT synctest | `testing/synctest` stable (go1.25+; project on 1.27) | go1.25 | Available but wrong tool here (no goroutines in core; real sockets prohibited in bubbles) |
| `math/rand` | `math/rand/v2` (PCG) | go1.22 | Already the project standard via testutil (`rand.NewPCG`); simulator reuses it |

**Deprecated/outdated:** `github.com/golang/protobuf` (do not import); hand-maintained wire codecs (none existed — this phase introduces the first); `time.Sleep`-based test pacing (already banned by project practice; simulator uses virtual time).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | macOS/BSD deliver subnet broadcast (real interface) to all SO_REUSEADDR sockets on the port, so the smoke test's fallback path works on `macos-15` runners | Patterns 3, Pitfall 3 | Smoke test skips on macOS (documented skip, gate still green); worst case: Phase 6 revisits |
| A2 | Kernel's recipient choice for unicast on a shared port is unspecified (probe observed most-recently-bound; nothing relies on which) | Pitfall 2 | None — design asserts nothing about it |
| A3 | `Ack` carries a single (key, version) pair per datagram; `DataReplication` carries a single record per datagram ("one message body" reading of IDEA.md:78) | Pattern 2 | Batching would change datagram counts, not semantics; planner may batch if tests show burst pressure (TEST-05 territory) |
| A4 | "Chunks self-sufficient" = each DataSync chunk independently interpretable (no continuation state) | Pattern 7 | If continuity were intended, chunk-loss handling changes; IDEA wording favors independence |
| A5 | `retransmit_interval` (100ms) is the retransmit scan granularity / inter-retransmit spacing, with backoff per key doubling from `ack_timeout` — IDEA lists both without disambiguation | Pattern 6 | Wrong reading shifts RETRY cadence; convergence bounds unaffected (they key on ack_timeout_max); planner should fix one reading in the implementation docs |
| A6 | A brand-new peer (first Hello, unknown node_id) receives the pending unacked set promptly (flush on first Hello) rather than waiting out backoff/sync timers | Pattern 6 | Slower new-node catch-up only; still bounded by sync_interval exchange |
| A7 | `broadcast_addr` config string is `ip:port`; all nodes listen on the same UDP port (the broadcast's destination port) | Patterns 3, Pitfall 2 | If per-node ports were intended, same-host semantics change entirely; IDEA's single `broadcast_addr` strongly implies shared port |
| A8 | `run_id` is 16 random bytes (UUIDv4-shaped or plain random) generated at process start | Pattern "Don't Hand-Roll" | Cosmetic; any unique-per-start value satisfies REPL-07 |

## Open Questions (all three RESOLVED — Q1 with deferred empirical outcome; settled by plans 04-02/04-05/04-06, back-annotated 2026-11-06)

1. **macOS smoke-test delivery path**
   - What we know: Linux behavior fully probed; lo0 lacks BROADCAST on Darwin; BSD-classic semantics say subnet broadcast fans out to REUSEADDR sockets.
   - What's unclear: whether `macos-15` runners' network filtering lets two processes exchange subnet broadcast on en0.
   - Recommendation: ship the pre-flight probe + fallback + loud skip (Pitfall 3); first CI run is the experiment; record the outcome in PHASE.md for Phase 6.
   - **(RESOLVED — deferred empirical outcome)** Pre-flight capability probe + subnet-broadcast fallback + loud documented skip shipped as recommended; the smoke test lands in plan 04-06 (Task 2), which additionally pins the Linux assertion (the probe must succeed there — SKIP fails the gate). Whether macOS en0 subnet broadcast reaches REUSEADDR sockets stays empirical until the first macos-15 CI run; the outcome is to be recorded for Phase 6 per the roadmap split (real-network hostile scenarios are Phase 6 territory).
2. **`retransmit_interval` semantics** (A5) — planner picks one reading and documents it next to the backoff table; the bounds tests are phrased on `ack_timeout_max` either way.
   - **(RESOLVED)** Scan-granularity reading: `retransmit_interval` bounds how long Advance's returned deadline may sleep while any entry is pending, while the per-key ladder doubles from `ack_timeout` to the `ack_timeout_max` saturation cap (200/400/800/1600/3200ms, then 5000ms repeated). Documented next to the ladder in the core's doc comment; bounds tests phrase on `ack_timeout_max`. Settled in plan 04-02 (Task 2 action + the A5 flagged assumption).
3. **Seed re-run hook: flag vs env var** (Pattern 4) — D-06 says "environment variable"; the testutil precedent is the `-seed` flag. Recommend following testutil (flag) since D-06's own text says "following the property-test precedent"; if the chaos test lives in a package where a flag is awkward, an env var read in the test is fine — the binding requirement is the printed, copy-paste re-run command.
   - **(RESOLVED)** Both hooks, env var primary: the chaos test honors `SUPKV_SIM_SEED` (D-06's locked text — and the only hook that works when the test binary runs without `-args`), while testutil's `-seed` flag remains the property-test hook; every run prints the copy-paste env-var replay command. Settled in plan 04-05 (Task 3 + the D-06 flag-vs-env flagged assumption).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | go1.27.3 linux/amd64 (go.mod: `go 1.27.1`) | — |
| google.golang.org/protobuf | wire codec | ✓ (already in module graph) | v1.36.12 | — |
| protoc (system) | codegen (dev machines only; CI unaffected — generated code committed) | ✓ | libprotoc 3.21.12 at /usr/bin/protoc | regenerate via any ≥3.x protoc on another machine; output is committed |
| protoc-gen-go | codegen | ✓ (built from module graph) | v1.36.12 | — |
| UDP broadcast on loopback | smoke test | ✓ Linux (probed) | kernel 7.3.1 | subnet-broadcast fallback; documented skip |
| UDP broadcast on macOS lo0 | smoke test on macos-15 cell | ✗ expected (lo0 lacks BROADCAST flag) | — | subnet broadcast on real interface; else skip with message |
| `go test -race` with real sockets | smoke test in CI | ✓ (current gate already runs -race everywhere) | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** macOS loopback broadcast (fallback = subnet broadcast / documented skip).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` (project standard; go-ultimate: vanilla testing, `package xxx_test`, no assertion libs in prod) |
| Config file | none needed — `make lint` / `make test` wrap the gates (`Makefile` [VERIFIED: read this session]: `go tool golangci-lint config verify && go tool golangci-lint run ./...`, `go test -race -cover ./...`) |
| Quick run command | `go test -race ./internal/app/repl/... ./internal/udprepl/...` |
| Full suite command | `make lint && make test` (identical to CI gate cells) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REPL-01 | Hello discovery, peer table population | integration (sim) + smoke (real sockets) | `go test -race ./internal/app/repl/ -run TestDiscovery` ; smoke: `go test -race ./internal/udprepl/ -run TestSmoke` | ❌ Wave 0 |
| REPL-02 | Per-peer tracking, backoff 200ms→5s, suspension, drain after heal | integration (sim, virtual clock) | `go test -race ./internal/app/repl/ -run TestBackoff` | ❌ Wave 0 |
| REPL-03 | Ack after in-memory apply (not dump) | unit + integration (sim) | `go test -race ./internal/app/repl/ -run TestAck` | ❌ Wave 0 |
| REPL-04 | DataRequest reply; no reply for unknown key | unit | `go test -race ./internal/app/repl/ -run TestDataRequest` | ❌ Wave 0 |
| REPL-05 | Chunked sorted sync ≤1400B, startup/timer/restart triggers, rate-limited responses | unit (chunking) + integration (sim) | `go test -race ./internal/app/repl/ -run 'TestSync'` | ❌ Wave 0 |
| REPL-06 | N-node table lifecycle (first Hello → known, 5s timeout → gone) | unit + integration (sim, N=3+) | `go test -race ./internal/app/repl/ -run TestPeerTable` | ❌ Wave 0 |
| REPL-07 | New run_id → full sync toward restarted node | integration (sim) | `go test -race ./internal/app/repl/ -run TestRestart` | ❌ Wave 0 |
| REPL-08 | Duplicate node_id dropped + logged as config error | unit (classification, Pattern 5) | `go test -race ./internal/app/repl/ -run TestDiscard` | ❌ Wave 0 |
| REPL-09 | Own echo / bad CRC / bad proto version distinguishable classes | unit (classification) + smoke (own echo real on Linux) | `go test -race ./internal/app/repl/ -run TestDiscard` | ❌ Wave 0 |
| REPL-10 | Bounds: 1 RTT normal; ≤ack_timeout_max under loss; ≤2×sync_interval+response after heal | integration (named sim tests, virtual clock) | `go test -race ./internal/app/repl/ -run 'TestBound'` | ❌ Wave 0 |
| TEST-03 | 2..N nodes over seeded lossy/dup/reorder/partition network; chaos test prints seed and replays | integration (sim + chaos) | `go test -race ./internal/app/repl/ -run TestChaos` (replay: `-args -seed=N` or env var per planner) | ❌ Wave 0 |
| persist seam | Remote merge triggers dump (no waiter) | unit (persist engine) | `go test -race ./internal/app/persist/ -run TestApplyRemote` | ❌ Wave 0 |
| smoke | Real UDP: discovery + write visible (D-08) | smoke (real sockets, localhost) | `go test -race ./internal/udprepl/ -run TestSmoke` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/...` quick subset touching the changed packages.
- **Per wave merge:** `make lint && make test` (the full gate).
- **Phase gate:** full suite green before `/gsd:verify-work`; chaos test additionally run once with a fixed seed (replay determinism proof: same seed → same trace → same verdict).

### Wave 0 Gaps
- `internal/app/repl/` package + classification/chunking/backoff unit tests — covers REPL-02..09 unit surface
- `internal/app/internal/simnet/` scheduler + virtual network (test-only package, testutil precedent)
- Named bound tests + seeded chaos test in `internal/app/repl/` (or a sibling sim test package) — REPL-10, TEST-03
- `internal/app/persist/` ApplyRemote test — remote-merge dump trigger (Pitfall 7)
- `internal/udprepl/` adapter + smoke test with pre-flight capability probe — REPL-01/09 real-network surface, D-08
- `.proto` + committed generated code + Makefile `proto` target + go.mod tool directive (code compiles before any logic lands)

## Security Domain

`security_enforcement: true`, ASVS L1 `[VERIFIED: .planning/config.json workflow block]`. This phase's attack surface is the untrusted UDP segment.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (v1 trust model) | Documented limitation: no auth/encryption in v1; HMAC is future work (SEC-01) `[VERIFIED: docs/IDEA.md:150-154 Trust model]` |
| V3 Session Management | no | No sessions on the wire |
| V4 Access Control | no | Single-trust-segment product by design (Out of Scope: WAN/multi-segment) |
| V5 Input Validation | **yes** | Every inbound datagram is untrusted input: 65 507-byte read cap, CRC gate before body decode (D-02), proto-version gate, envelope field validation (run_id length 16, node_id range), store-level `value_max_bytes` rejection (ErrValueTooLarge — exists), malformed-proto discard class. Untrusted-input boundary rules apply: decoded values cross into the core only as validated port structs |
| V6 Cryptography | no (integrity only) | CRC32 (IEEE) = corruption detection, explicitly not tamper protection — per the accepted trust model; never present it as a security control in docs/logs |
| V12 File ops | no | No new file I/O this phase (dumps are Phase 3, unchanged) |

### Known Threat Patterns for UDP protobuf replication

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed/spoofed datagrams crash a node | DoS | Total decode path: size-capped reads, CRC-before-decode, `proto.Unmarshal` errors → discard class, classification never panics on adversarial bytes (testutil's `RandRawVersion` precedent for arbitrary 16-byte inputs) |
| Duplicate node_id misconfiguration splits the cluster | Tampering (accidental) | REPL-08: drop + loud config-error log — an operational signal, not a silent fork |
| Replay/duplication/reordering by the network | Tampering | Inherent CRDT defense: merge is commutative/associative/idempotent (CORE-06 complete); duplicates are no-ops — protocol correctness does not depend on dedup |
| Datagram flooding amplification (sync-response loops) | DoS | `sync_response_rate` rate limiter bounds reply volume per node; DataSync responses are unicast (not re-broadcast), so no amplification chain |
| Own-broadcast feedback loops | DoS | REPL-09 own-echo class drops the sender's own datagrams before protocol processing |

## Sources

### Primary (HIGH confidence)
- Live probes this session (this machine, kernel 7.3.1-1-t2-noble, go1.27.3): UDP broadcast/unicast/echo matrix, 62KB loopback datagram, protoc 3.21.12 + protoc-gen-go v1.36.12 codegen + runtime round-trip + 60KiB record size, `go tool` directive diff, `go doc testing/synctest`
- In-repo files read verbatim: docs/IDEA.md, .planning/REQUIREMENTS.md, 04-CONTEXT.md, AGENTS.md, .golangci.yml, go.mod, Makefile, .github/workflows/ci.yml, internal/port/dump.go, internal/app/{storage/store.go, clock/clock.go, version/version.go, persist/engine.go, internal/testutil/rng.go, internal/testutil/gen.go}, cmd/sup-kv/main.go

### Secondary (MEDIUM confidence)
- [lkoskela.fi — Loopback UDP broadcasts and multiple listeners (2025-10)](https://www.lkoskela.fi/blog/2025-10-19-udp-loopback-bcast) — Linux-only loopback fan-out; FreeBSD negative; macOS untested/reportedly no
- [Server Fault — broadcast IP on macOS loopback](https://serverfault.com/questions/554503/how-do-i-add-a-broadcast-ip-to-the-loopback-interface-under-os-x-using-ifconfig) — lo0 lacks BROADCAST flag
- [Stack Overflow — SO_REUSEADDR and UDP behavior](https://stackoverflow.com/questions/5077887/so-reuseaddr-and-udp-behavior-in-windows) — *nix broadcast-to-all-REUSEADDR vs Windows single-delivery
- [socket(7) man page](https://man7.org/linux/man-pages/man7/socket.7.html) — SO_REUSEADDR semantics

### Tertiary (LOW confidence)
- None remaining — every claim is probe-verified, file-quoted, or explicitly logged in the Assumptions Log

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — protobuf module already pinned in-repo and probe-executed; no new dependencies; protoc verified present; generation round-trip proven
- Architecture: HIGH — every seam already has an in-repo precedent (port.DumpStore, storage.Clock, testutil rng, persist dirty-path comment); platform behavior probe-verified
- Pitfalls: HIGH — the top three (determinism leaks, same-host unicast, macOS loopback) are all grounded in live probes or package docs, not folklore

**Research date:** 2026-11-03
**Valid until:** 2027-01-03 (stable domain; platform behaviors are kernel-facts, not fashion)
