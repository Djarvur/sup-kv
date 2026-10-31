# Architecture Research

**Domain:** leaderless replicated in-memory KV store (LWW-Element-Map CRDT, AP, UDP gossip-style replication), Go, single static binary
**Researched:** 2026-10-31
**Confidence:** HIGH (layout/wiring — binding project conventions; MEDIUM for external-system facts, verified against primary sources)

The decided design (`docs/IDEA.md`) is taken as given. This research answers: how do comparable
systems organize these components, what Go package layout fits, where are the natural module
boundaries, the goroutine/shutdown topology, the testing seams — and what build order the
dependencies imply.

## Standard Architecture

### How the reference systems are organized

| sup-kv component | Corrosion (Fly.io)¹ | Marmot | Alertmanager cluster | etcd raft | CockroachDB `pkg/util/hlc` |
|---|---|---|---|---|---|
| HLC / version | — | top-level `hlc/` package | — | — | `Clock` (mutex-guarded state) + `Timestamp` value type + `WallClock` interface |
| CRDT store / merge | cr-sqlite CRDT layer | `db/` (row versions, applier) | `State` interface (`msg()`, `mergeState()`) + `clusterpb` protos | log + state machine (library core) | — |
| Replication engine | `agent/` task modules + `broadcast/` + `plumtree/` | `publisher/` + `replica/` + anti-entropy service | `delegate.go` (retransmit queue, push/pull anti-entropy) | deterministic state-machine core; transport is the embedder's job | — |
| Peer table / membership | `plum-foca` (SWIM) | `cluster/` + node registry | memberlist + `NotifyJoin/Leave/Update` | — | — |
| Transport seam | `transport.rs` | NATS JetStream | memberlist transport (replaced wholesale for TLS) | user-provided (`raftexample`) | — |
| Persistence | SQLite pool | SQLite + snapshot | none (in-memory state, gossip push/pull) | user-provided `Storage` | — |
| Startup gate | `bootstrap` module | — | `Settle(ctx, interval)` — wait for gossip to settle before serving | — | — |
| Shutdown | `tripwire` crate (signal fan-out) | `ShutdownOrchestrator` (ordered, isolated steps) | `Leave(timeout)` | — | — |
| Deterministic testing | none public (real-process integration tests) | unit/integration tests | interfaces (`ClusterPeer`/`ClusterMember`/`ClusterChannel`) | determinism by design (same input ⇒ same output) | `NewClockForTesting(WallClock)` + `HybridManualClock` (pause/forward) |

¹ Corrosion is **Rust**, not Go — it is a design reference only; its *module decomposition* (not its
language idioms) is what transfers.

Recurring structural facts across all four Go systems (MEDIUM confidence, verified against
source):

1. **A stateful "clock" object separated from a pure "timestamp/version" value type.**
   CockroachDB: `Clock` (mutex, `Now()`, `Update()`) vs `Timestamp` (immutable, `Less`/`Forward`,
   proto-backed). The physical wall clock is behind an interface (`WallClock`), with a manual
   clock for tests — exactly the seam sup-kv's deterministic simulation needs for wall-clock
   jumps.
2. **The replication protocol is a state machine with I/O at the edges.** etcd raft's README
   states it as the design principle: input is a Message (timer tick or network message), output
   is `{messages, state changes}`; same input ⇒ same output. Network and disk I/O belong to the
   embedder. This is what makes etcd raft testable and portable.
3. **Membership/peer bookkeeping is separate from data replication.** Alertmanager (memberlist +
   delegate), Corrosion (SWIM membership + plumtree broadcast as separate modules).
4. **Anti-entropy is a separate, timer-driven path from the hot replication path.**
   Alertmanager's `LocalState`/`MergeRemoteState` (push/pull) vs `GetBroadcasts` (incremental);
   sup-kv's `DataSync` vs `DataReplication` mirror this exactly.
5. **A startup "settle" gate before serving is an established pattern** (Alertmanager `Settle`).
6. **Ordered, step-isolated shutdown for clustered daemons** (Marmot's `ShutdownOrchestrator`:
   each step wrapped so one failure doesn't abort the rest; front-facing listeners stopped
   first).

### System Overview

sup-kv mapped onto a bounded-context hexagon (the layout mandated for this project — see
"Recommended Project Structure"):

```
                    ┌──────────────────────────────────────────────┐
                    │                 Clients (gRPC)               │
                    └──────────────────────┬───────────────────────┘
                                           │ Get/Set/Delete/Watch/snapshots
┌──────────────────────────────────────────▼─────────────────────────────────────────┐
│ Inbound adapter: internal/in/grpc                                                  │
│  • protobuf → port DTO mapping, error→status mapping, Watch stream fan-out          │
├────────────────────────────────────────────────────────────────────────────────────┤
│ Application boundary: internal/port (port.go)                                      │
│  • App (KV ops, Watch, snapshots) + Out ports: Transport, Dumper, WallClock         │
│  • port-level errors (ErrNotFound, ErrAlreadyExists, ErrUnavailable, ErrTooLarge)   │
├────────────────────────────────────────────────────────────────────────────────────┤
│ Application core: internal/app (the Node)                                          │
│  ┌────────────┐ ┌────────────┐ ┌─────────────┐ ┌──────────────┐ ┌───────────────┐ │
│  │ version    │ │ store      │ │ replication │ │ peers        │ │ gate + dump   │ │
│  │ HLC+UUIDv7 │ │ LWW map,   │ │ engine:     │ │ NodeID→addr/ │ │ schedule      │ │
│  │ Tick/Update│ │ merge,     │ │ retransmit, │ │ run_id, last │ │ startup gate, │ │
│  │            │ │ tombstones,│ │ ack sets,   │ │ hello,       │ │ wait_for_     │ │
│  │            │ │ checksums, │ │ DataSync    │ │ timeouts     │ │ replication,  │ │
│  │            │ │ conflict   │ │ chunker +   │ │ (reaper)     │ │ watchers      │ │
│  │            │ │ journal    │ │ rate limit  │ │              │ │               │ │
│  └────────────┘ └────────────┘ └─────────────┘ └──────────────┘ └───────────────┘ │
│  Owns ALL business decisions; imports no net/os/transport SDKs                     │
├────────────────────────────────────────────────────────────────────────────────────┤
│ Outbound / persistence adapters                                                    │
│  ┌─────────────────────────────┐      ┌────────────────────────────────────────┐   │
│  │ internal/out/udp            │      │ internal/dal/dump                       │ │
│  │ socket, Envelope marshal/   │      │ JSON dump (atomic temp+fsync+rename,    │ │
│  │ unmarshal, CRC32, own-echo  │      │ .prev fallback), load+verify, snapshot  │ │
│  │ discard, read loop          │      │ copies; synchronous, called by app      │ │
│  └─────────────┬───────────────┘      └────────────────────────────────────────┘   │
└────────────────┼───────────────────────────────────────────────────────────────────┘
                 │ Envelope datagrams (broadcast + unicast), UDP
   ┌─────────────▼─────────────┐          ┌────────────────────────────────────┐
   │ Peer nodes (same binary)  │          │ Deterministic simulator (test-side) │
   │ on the LAN broadcast      │          │ virtual net + virtual clock + fake  │
   │ segment                   │          │ Dumper, drives N nodes via ports     │
   └───────────────────────────┘          └────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Talks to | Precedent |
|-----------|----------------|----------|-----------|
| `internal/port` | The whole contract: `App` in-port; `Transport`, `Dumper`, `WallClock` out-ports; port-level errors; boundary DTOs (`Record`, `RecordVersion`, change events) | everyone (leaf tier) | go-ultimate `port.go`; Alertmanager `ClusterPeer` seams |
| `internal/app` (version) | HLC state `(l, c)`, `Tick`/`Update`, UUIDv7 pack/unpack, byte-order comparison, uniqueness, drift warning signal | store, replication engine, gate | CockroachDB `hlc.Clock` + `WallClock` |
| `internal/app` (store) | LWW-Element-Map under RWMutex; merge by greater `RecordVersion`; tombstones; checksum compute/verify; conflict-journal events; Watch notification source | version, replication engine, dump scheduling | CRDT map; Marmot row-version merge |
| `internal/app` (replication engine) | Protocol state machine: per-peer unacked sets + backoff, `DataSync` chunking + response rate limit, `DataRequest` handling, `Ack` correlation, supersession | store, version (`Update`), peers, Transport port | etcd raft "core state machine"; Alertmanager delegate |
| `internal/app` (peers) | Peer table `NodeID → {addr, run_id, last hello}`; duplicate `node_id` detection; restart detection (`run_id` change → trigger sync); peer-timeout reaper | replication engine, Transport | SWIM/membership modules (Corrosion `foca`, memberlist) |
| `internal/app` (gate + dump schedule) | Startup gate (`Unavailable` until first sync round or `startup_gate`); debounced dump triggering; per-write confirmation registry; `wait_for_replication` waiters | store, replication engine, Dumper port | Alertmanager `Settle`; Marmot draining flag |
| `internal/in/grpc` | gRPC service impl: proto⇄DTO conversion, error mapping, Watch stream lifecycle | port.App only | hexagonal in-adapter |
| `internal/out/udp` | UDP socket (broadcast + unicast), Envelope (de)serialization, CRC32 check, `proto_version` check, own-broadcast-echo discard, datagram discard classification/counters; delivers parsed messages to the app | Transport port implemented; app consumes its output | Corrosion `transport.rs`; raftexample transport |
| `internal/dal/dump` | Dump format (version + records), atomic write, `.prev` rotation, load with fallback chain, bad-record discard, snapshot create/load | Dumper port; app calls it | Marmot snapshot/restore; SQLite backup patterns |
| `wire.go` / `main.go` | Wiring (build graph, start nothing) / process lifecycle (signals, serve loop, ordered shutdown) | all | go-ultimate `Wire`+`Runtime`+`Unit`; Marmot `ShutdownOrchestrator` |
| Simulator (test-only, `internal/sim`) | Virtual network (loss/dup/reorder/partition/one-way), virtual clock with jumps, in-memory Dumper, seeded scenario runner | node instances via their ports | FoundationDB simulation; TigerBeetle VOPR |

### Key Architectural Principle (the one to enforce in review)

**Deterministic core, I/O at the edges** — everything that *decides* (merge, version choice,
retransmission timing, peer liveness, sync scheduling, gate state) lives in `internal/app`
behind ports, with no direct socket/file/wall-clock access. Everything that *touches the world*
(socket, files, gRPC, `time.Now`) lives in adapters. etcd raft's README makes the argument:
keeping I/O out of the algorithm is what buys determinism, testability, and portability. For
sup-kv it is additionally the enabler of the v1.0 success criterion — the failure-scenario
suite must demonstrate convergence deterministically, which is only possible if the whole
decision core can run on a virtual network and virtual clock.

## Recommended Project Structure

Binding constraint: the project's quality gates adopt **go-ultimate**
(github.com/Djarvur/go-ultimate), whose service architecture reference (bounded-context
hexagonal, condensed from powerman's `go-bounded-context-hexagonal`) *wins over generic Go
layout guidance*. Consequences: **no `pkg/` directory, ever**; `internal/` for private code;
single binary keeps `main.go` at the module root; no `internal/in/cli`; adapters contain no
business decisions; one `port.go` holding `App` + Out ports + errors.

```
sup-kv/
├── go.mod
├── main.go                        # process concerns only: flags/env→Config, signals, serve loop
├── wire.go                        # Wire(ctx, startupCtx, cfg) (*Runtime, []Unit, error); starts nothing
├── api/                           # wire formats BETWEEN processes (go-ultimate: api/ = cross-process)
│   ├── supkv/v1/                  # client gRPC: supkv.proto + generated pb + grpc pb
│   └── envelope/v1/               # replication: envelope.proto (Envelope, Hello, DataReplication,
│                                  #   Ack, DataRequest, DataSync) + generated pb
├── internal/
│   ├── port/
│   │   ├── port.go                # App in-port; Transport, Dumper, WallClock out-ports; errors
│   │   └── types.go               # boundary DTOs: Record, RecordVersion ([16]byte), ChangeEvent, PeerInfo
│   ├── app/                       # the Node — ALL business logic; no net/os/transport imports
│   │   ├── app.go                 # implements port.App; composes the parts below
│   │   ├── version.go             # HLC (l,c): Tick/Update; UUIDv7 pack/unpack; byte compare
│   │   ├── store.go               # LWW map, RWMutex, merge, tombstones, checksum, conflict journal
│   │   ├── peers.go               # peer table, run_id/restart detection, duplicate-node_id, reaper
│   │   ├── repl.go                # replication engine: send/retransmit/ack/supersession
│   │   ├── sync.go                # chunked DataSync builder + responder with rate limiting
│   │   ├── gate.go                # startup gate, dump-debounce scheduler, write confirmations,
│   │   │                          #   wait_for_replication waiters
│   │   ├── watch.go               # watcher registry + fan-out
│   │   └── node.go                # single event loop owning protocol state (see Goroutine Topology)
│   ├── in/
│   │   └── grpc/adapter.go        # supkv.v1 gRPC service → port.App (+ metrics interceptor)
│   ├── out/
│   │   └── udp/adapter.go         # Transport impl: socket, envelope codec, CRC, discard classes
│   ├── dal/
│   │   └── dump/                  # Dumper impl: dump.go (format+atomic write+prev), load.go
│   │                              #   (fallback chain), snapshot.go
│   └── sim/                       # deterministic harness (imported only by tests/CI)
│       ├── sim.go                 # Cluster builder: N nodes on fake Transport/Dumper/WallClock
│       ├── net.go                 # virtual network: seeded loss/dup/reorder/partition/one-way
│       ├── clock.go               # virtual clock w/ jumps; step-until-quiescent loop
│       └── scenarios_test.go      # the failure-scenario suite (seeded; seed overridable)
└── .golangci.yml                  # strict config incl. depguard enforcing the boundaries below
```

### Structure Rationale

- **`internal/port` as the single most-depended-on leaf**: adapters, `app`, `wire`, the
  simulator, and tests all import it; it imports only stdlib (+ pure value types). This is
  what makes the simulator possible without monkey-patching — the sim is just another set of
  port implementations.
- **`internal/app` holds engine + replication together** (files first, subpackages only if it
  grows — go-ultimate growth order). Splitting `engine/` and `replication/` into sibling
  packages would tempt a cycle: merge needs version minting; replication needs merge and
  version-update. One package, many files, is the honest shape until a real need appears.
- **`api/` holds both protobuf contracts** because both cross process boundaries (client↔node,
  node↔node) — they are wire formats, not business types. Adapters convert proto⇄`port` DTOs;
  `app` never sees generated types (keeps the core lintable as I/O-free).
- **`internal/sim` under `internal/`**: test-only machinery; CI runs it via
  `go test ./internal/sim/...`; repro by seed. No second binary needed, so `main.go` stays at
  root (go-ultimate: move to `cmd/` only when a second binary truly appears).
- **Boundary enforcement is mechanical, not aspirational**: golangci-lint `depguard` —
  `internal/app` may not import `net`, `net/http`, `os`, `google.golang.org/grpc`, or `api/…`;
  `internal/port` may not import anything beyond stdlib. This single CI rule keeps the
  deterministic-core principle true for the life of the project.

## Architectural Patterns

### Pattern 1: Ports & adapters with injection at the composition root

**What:** `port.go` declares the seams; `wire.go` builds the graph and injects the real
adapters; the simulator injects fakes through the *same* constructors.
**When to use:** always, in this project — it is both the go-ultimate mandate and the
precondition for deterministic simulation.
**Trade-offs:** one indirection layer between proto and store types; in exchange, the
failure-scenario suite requires zero socket privileges, runs in milliseconds, and is
byte-reproducible.

```go
// internal/port/port.go (sketch)
type App interface {
    Get(ctx context.Context, key string) (Record, error)
    Set(ctx context.Context, key string, data []byte, ifAbsent bool) (Record, error)
    Delete(ctx context.Context, key string) (Record, error)
    Watch(ctx context.Context) (<-chan ChangeEvent, func(), error)
    SnapshotCreate(ctx context.Context, name string) error
    SnapshotLoad(ctx context.Context, name string) error
}

type Transport interface { // implemented by internal/out/udp and by sim/net.go
    Send(ctx context.Context, addr string, env Envelope) error
    LocalAddr() string
}

type Dumper interface { // implemented by internal/dal/dump and by sim
    Dump(ctx context.Context, recs []Record) error
    Load(ctx context.Context) ([]Record, Source, error) // main → .prev → empty
}

type WallClock interface { Now() time.Time } // implemented by real clock and sim/clock.go
```

### Pattern 2: Single-owner event loop for protocol state (etcd-raft style core)

**What:** all mutable replication state (peer table, unacked sets, backoff timers, sync
cursor, gate state) is owned by **one** goroutine — the node loop — fed by three input
streams: inbound envelopes (from the UDP adapter), local write events (from `app.Set`/
`Delete`), and ticks (hello, sync, retransmit scan, reaper, dump-debounce deadline). The
store keeps its own RWMutex (per the decided design) because reads must not serialize behind
the protocol loop.
**When to use:** whenever protocol state is read and written from multiple timers; the
alternative (mutex soup across five tickers) is the classic source of replication races.
**Trade-offs:** one goroutine's throughput bounds protocol processing (~fine at 1000 writes/s
with 60 KiB payloads on a LAN); in the simulator the same loop can be driven *stepwise,
synchronously* — the single most important trick for determinism.

```go
// internal/app/node.go (sketch)
func (n *Node) Run(ctx context.Context) error {
    ticks := mergeTickers(ctx, n.cfg) // hello, sync, retransmit, reaper, debounce deadlines
    for {
        select {
        case env := <-n.inbound:  n.handleEnvelope(ctx, env) // dispatch to peers/repl/sync
        case ev := <-n.local:     n.handleLocal(ctx, ev)     // version, store, broadcast, dump-sched
        case t := <-ticks:        n.handleTick(ctx, t)
        case <-ctx.Done():        return n.shutdown(ctx)
        }
    }
}
```

### Pattern 3: Wire()/Runtime/Unit serve loop + ordered, step-isolated shutdown

**What:** go-ultimate's mandatory wiring shape, with Marmot's shutdown ordering. `Wire` returns
a `Runtime` (exposing `port.App`, health, `Close`) plus startable `Unit`s; `main` runs them in
parallel and shuts down in reverse startup order, each step isolated so one failure doesn't
abort the rest.
**When to use:** always in this project.
**Trade-offs:** slightly more ceremony than `func main() { go everything() }`; in exchange the
same wiring serves production, integration tests, and (with fakes) the simulator.

Shutdown order for sup-kv (derived from Marmot's orchestrator + the decided semantics):

1. gRPC server `GracefulStop` — no new writes; in-flight writes run to completion (they may
   still trigger one last dump).
2. Final dump flush — dump everything currently in memory (graceful stop should persist even
   *unconfirmed* writes; on kill −9 those survive only on peers, which the design accepts).
3. Stop the node loop, close the UDP socket (peers simply observe `peer_timeout`).
4. Cancel timers, close Watch streams (stream contexts derive from the root ctx), close
   metrics listener, `Runtime.Close()`.

### Pattern 4: Clock and time as injected dependencies

**What:** one `WallClock` port feeds the HLC; all timers in the node loop are created from a
clock/timer abstraction (or driven by the tick stream); tests and the sim inject jumps.
**When to use:** always — the failure-scenario list explicitly includes "wall-clock jumps on a
live node and between restarts" and "a node's clock far ahead".
**Trade-offs:** a thin timer abstraction is a few dozen lines; without it those scenarios are
untestable. Precedent: CockroachDB `NewClockForTesting(WallClock)` + `HybridManualClock`
(`Pause`/`Forward`), plus its `StartMonitoringForwardClockJumps` goroutine — the model for
sup-kv's `drift_warn` monitor.

### Pattern 5: Deterministic simulation harness (FoundationDB/TigerBeetle model, Go-sized)

**What:** a `sim.Cluster` of N node instances wired to: an in-memory `Transport`
(virtual network with seeded loss/duplicate/reorder/partition/one-way delivery), a virtual
`WallClock` (jumpable, drives all timers), and an in-memory `Dumper` (or real temp-dir dumps
when the scenario tests persistence — determinism of file I/O is fine when only the sim
thread runs). The runner: pick next event from a seeded priority queue (earliest-deadline
timer or next network delivery), advance virtual time, execute, repeat until quiescent, then
assert convergence and bounds. Fault flags per scenario (FoundationDB "BUGGIFY" style).
**When to use:** the v1.0 criteria demand it; build it with Phase 3 (replication), extend in
Phase 5 (hardening).
**Trade-offs:** the loop must be the only scheduler — no real goroutines racing inside the
simulated cluster; Pattern 2's single-owner loop is what makes that natural.

**Complement, not replacement:** Go 1.25's `testing/synctest` (stable; go.dev blog "Testing
Time") gives virtual-time bubbles with deterministic scheduling for goroutine-level tests —
excellent for the dump-debounce/gate/waiter interactions of a *single* node — but bubbles
forbid real network/file I/O, so the multi-node harness still needs the port-injected virtual
network above. Use synctest for node-internal timing tests; use the sim for cluster scenarios.

## Data Flow

### Local write path (the confirming path)

```
Client →gRPC Set→ in/grpc → port.App.Set
   → app: gate open? value ≤ 60 KiB?            (else Unavailable / InvalidArgument)
   → version: HLC Tick → RecordVersion (UUIDv7)
   → store: put record (local version is newest by construction)
   → watch: fan-out ChangeEvent (local)
   → repl: broadcast DataReplication(full record); record (key,version) in per-peer unacked sets
   → gate: register write confirmation; schedule dump after dump_debounce
   ← ... debounce fires → dal/dump: Dump(records) → atomic write → on success:
        all writes with version ≤ dumped watermark confirmed → gRPC replies OK to waiters
        (on dump failure: those waiters get Unavailable; write stays in memory + replicates)
   → (optional wait_for_replication) gate waits for per-peer Ack events / timeout → report
```

Direction of truth: **version → store → (watch, repl, dump) — the record flows outward;
nothing flows back into the local version decision** (a local write is never rejected by
replication).

### Replication inbound path

```
UDP socket → out/udp read loop
   → validate: proto_version? crc32? own node_id (broadcast echo)? duplicate node_id? → classify+count discards
   → port DTO → app node loop inbound
       Hello          → peers: upsert {NodeID, addr, run_id, last hello}; run_id change ⇒ enqueue full sync toward it
       DataReplication→ store.merge(greater RecordVersion wins)
                        ├─ local lost → conflict journal entry + watch event (replicated)
                        └─ applied → unicast Ack(key,version); schedule dump (debounced)
                        → version: HLC Update(received l)
       Ack            → repl: clear (key,version) from that peer's unacked set; notify wait_for_replication waiters
       DataRequest    → repl: unicast DataReplication for that key (if known)
       DataSync chunk → per-key compare → queue DataRequest (missing/stale, rate-limited)
                                          or unicast DataReplication (we are newer)
```

Merges are idempotent/commutative, so this path tolerates loss, duplication and reordering by
construction — the decided design's core property; the architecture must preserve it by never
letting delivery order affect anything but *when* the supremum is reached.

### Anti-entropy path (timer-driven)

```
tick(sync_interval) | peer run_id change | startup
   → repl: sorted key→version listing → chunks ≤ sync_chunk_bytes (self-sufficient)
   → broadcast DataSync chunks → receivers respond (rate-limited, see inbound) → convergence
```

### Startup path

```
main → Wire → Runtime+units → serve loop
unit[load]:   dal/dump Load (checksums; main → .prev → empty) → store seed → version init from max l
unit[udp]:    open socket → first Hello broadcast
unit[node]:   node loop; gate CLOSED for writes, reads already served
              gate opens on: first full sync round completed | startup_gate elapsed
unit[grpc]:   listener up (may start before gate opens — it must return Unavailable for writes)
```

### Shutdown path

see Pattern 3 order (1) gRPC drain → (2) final dump → (3) node loop + socket → (4) timers/watchers/Close.

### Key Data Flows (summary)

1. **Write confirmation watermark**: monotonic "highest version included in last successful
   dump" — gate compares pending writes against it; single scalar, owned by `gate.go`.
2. **Watch fan-out**: store merge (local or replicated) → ChangeEvent per watcher; slow
   watchers get their stream closed (see Anti-Patterns).
3. **Version inflow**: every applied foreign record updates the HLC (`Update`) — flows from
   repl into version, never the reverse.

## Goroutine Topology (production wiring)

| Goroutine (Unit) | Owns | Blocked on |
|---|---|---|
| serve loop (main) | lifecycle: runs units, first-return triggers shutdown | `errgroup` |
| gRPC server | inbound RPCs; Watch streams | GracefulStop |
| UDP read loop | socket reads; validation; discard counters | socket; ctx |
| node loop | ALL protocol state (peers, unacked, sync, gate, watchers) | inbound/local/ticks/ctx |
| dump worker | serialized dumps (one at a time; debounce coalesces) | dump queue |
| metrics HTTP (net/http, stdlib) | /metrics exposition | listener |
| drift monitor | forward-jump detection, `drift_warn` | WallClock polling |

Rules (go-ultimate concurrency hygiene): every goroutine has an exit condition (ctx or
errgroup); bounded fan-out; channels unbuffered or size 1 (except the inbound envelope queue,
which is bounded and counted as a metric). The store's RWMutex is the *only* shared mutable
state outside the node loop — reads stay lock-free of protocol processing.

## Testing Seams (and what each buys)

| Seam | Production impl | Test/Sim impl | Enables |
|---|---|---|---|
| `Transport` | `out/udp` socket | sim virtual net (seeded chaos) | every replication failure scenario |
| `WallClock` | real clock | sim clock (jumps, pause, far-ahead) | HLC properties, drift, jump scenarios |
| `Dumper` | `dal/dump` files | in-memory or temp-dir | restart/recovery scenarios, dump failure injection |
| `port.App` | `app` node | gomock mock | gRPC adapter unit tests (error mapping, Watch) |
| tick stream | `time.Ticker`s | sim event queue | deterministic timer interleaving |

Test layers: property tests (merge laws, HLC monotonicity/uniqueness — pure, Phase 1);
unit tests per package with mocks; single-node timing tests via `testing/synctest`;
multi-node deterministic scenarios in `internal/sim`; one smoke test of the real binary over
a real UDP broadcast socket on localhost (the quickstart, Phase 5/6).

## Anti-Patterns

### Anti-Pattern 1: Fat transport adapter (replication logic in `out/udp`)

**What people do:** put retransmission, peer tracking, or merge decisions in the socket
adapter because "that's where the packets are".
**Why it's wrong:** business decisions in an adapter (a) violate the binding go-ultimate
adapter discipline, (b) make the failure-scenario suite need real sockets — flaky, slow,
non-deterministic, and unable to reproduce by seed.
**Do this instead:** adapter does codec + validation + discard classification only; the node
loop decides. If a change requires `internal/app` to import `net`, the architecture is broken.

### Anti-Pattern 2: `time.Now()` / real tickers anywhere in the decision core

**What people do:** sprinkle `time.Now()` through HLC, gate, and retransmission code.
**Why it's wrong:** wall-clock-jump and far-ahead-clock scenarios become untestable; the sim
cannot be deterministic.
**Do this instead:** one `WallClock` port; timers arrive as events on the node loop's tick
stream (CockroachDB's `WallClock` interface is the precedent).

### Anti-Pattern 3: Protocol state shared across tickers under mutexes

**What people do:** five independent `time.Ticker` goroutines each locking the peer table /
unacked sets.
**Why it's wrong:** interleaving-dependent bugs — exactly the class the simulation must
control; races between supersession and retransmit are notorious.
**Do this instead:** single-owner node loop (Pattern 2); the store keeps its RWMutex, nothing
else shares.

### Anti-Pattern 4: Map-iteration-order dependence in sync/dump

**What people do:** iterate the Go map directly when building `DataSync` chunks or dumps.
**Why it's wrong:** Go map iteration is randomized — chunk sequences and diff outcomes become
non-deterministic across nodes/runs. (`encoding/json` sorts map keys, which saves the dump's
 readability/determinism — do not "fix" that with struct-of-slices.)
**Do this instead:** sort keys explicitly for sync chunking (the decided design already says
"sorted"); keep the dump as a map-shaped JSON doc for free determinism.

### Anti-Pattern 5: Unbounded Watch queues

**What people do:** per-watcher channels that grow forever under 1000 writes/s bursts.
**Why it's wrong:** OOM under exactly the hot-key workload the product targets.
**Do this instead:** small bounded buffer; on overflow close the stream with an error and let
the client re-Watch (loss handling documented; state re-read via Get).

### Anti-Pattern 6: Shutdown that skips the final dump

**What people do:** cancel ctx and exit, assuming "confirmed writes are on disk" covers it.
**Why it's wrong:** *unconfirmed* writes (in memory, possibly acked by peers) are silently
dropped on graceful stop — worse than kill −9 semantics where peers hold them.
**Do this instead:** ordered shutdown with an explicit final flush (Pattern 3, step 2).

### Anti-Pattern 7: `pkg/` directory / generic layout cargo-culting

**What people do:** copy etcd/Marmot-style trees (`pkg/…`, `server/`, `raft/`-per-subsystem).
**Why it's wrong:** go-ultimate explicitly forbids `pkg/`; deep per-subsystem trees re-create
bounded-context boundaries inside one bounded context.
**Do this instead:** the structure above; `depguard` in CI makes violations compile-time
visible.

## Scaling Considerations

Realistic for this product (trusted segment, N ≤ ~5 nodes, state in RAM):

| Concern | 100s keys, KiB values (config) | 10k keys × ~1 KiB | worst case 10k × 60 KiB |
|---|---|---|---|
| Dump rewrite (full JSON per debounce) | trivial | ~10 MB per dump; debounce coalesces 1000/s bursts into ~1 dump/50 ms — OK | ~600 MB per dump — **document as out of envelope**; benchmark gate in Phase 5 |
| Broadcast fan-out | O(N) datagrams per write — negligible | same; unicast Acks keep reply storm at O(N) per write | 60 KiB payloads near UDP datagram limits — envelope/chunk sizing must be verified by test (Phase 3) |
| Unacked sets during partition | tiny | bounded by distinct (key,version) in flight; supersession prunes | memory pressure point — metric (unacked count/age) is in the design; watch it in benchmarks |
| Watch fan-out | trivial | per-watcher bounded buffers | same |

Scaling priorities: (1) first bottleneck is dump rewrite amplification at high write rate on
large state — mitigated by debounce; keep it a benchmark, not a redesign. (2) second is unacked
growth during day-long partitions — the supersession rule already bounds it; assert in sim.
No premature optimization beyond these.

## Integration Points

### External Services

| Service | Integration pattern | Notes |
|---|---|---|
| systemd | `Type=notify` (or `simple`) + `SIGTERM` → graceful stop; `KillMode=process`; `Restart=on-failure` | startup gate means "ready" only after load+gate; kill −9 recovery is by design |
| Prometheus-style metrics | tiny in-repo text exporter on `net/http` stdlib listener | keeps the "gRPC+protobuf only" dependency constraint; the design's counters/gauges are trivial in text format (final call belongs to STACK research) |
| proto codegen | `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`, generated into `api/…`, committed | CI verifies generation is up to date |
| structured logging | stdlib `log/slog` JSON handler | every discard class distinguishable (design requirement) |

### Internal Boundaries

| Boundary | Communication | Notes |
|---|---|---|
| `in/grpc` ↔ `app` | `port.App` calls only | adapter maps port errors → gRPC statuses (`Unavailable`, `AlreadyExists`, `InvalidArgument`, `NotFound`) |
| `app` ↔ `out/udp` | `port.Transport` (send) + bounded inbound queue (receive) | adapter converts proto⇄`port` DTOs; app never imports `api/` |
| `app` ↔ `dal/dump` | `port.Dumper`, synchronous | dump serialized by the dump worker; debounce in app |
| `app` version ↔ store/repl | direct calls, same package | HLC mutex inside version; no other shared state outside store RWMutex |
| sim ↔ node instances | the same three ports, fakes injected | no `testonly` hooks inside `app` — the ports ARE the hooks |

## Build Order (validating the suggested milestone shape)

The doc's proposal — CI → Engine → Persistence → Replication → API → Hardening — is **correct
and dependency-sound**. Validation and refinements:

1. **Phase 0 — CI bootstrap** (before feature code; unchanged). Add the `depguard` boundary
   rules to the strict golangci-lint config here, so every later phase lands inside the
   architecture from its first commit.
2. **Phase 1 — Engine** (`version.go`, `store.go` + property tests). Pure leaf, zero I/O,
   everything else imports it; property tests (merge commutative/associative/idempotent; HLC
   monotonic under adversarial clock input; UUIDv7 uniqueness/byte-order) live here and run
   forever after. **Refinement:** land `port.go`, `wire.go`, `main.go` skeleton (Runtime +
   serve loop + signal shutdown, no units yet) in this phase — it is ~100 lines, gives CI a
   building binary, and fixes the process lifecycle pattern before any subsystem exists.
3. **Phase 2 — Persistence** (`dal/dump` + `gate.go` debounce/confirmation). Depends only on
   engine types. **Refinement:** its exit criterion is the write-confirmation contract
   (Set→OK only after dump) testable through `port.App` — i.e. the *node interior* (gate +
   dump worker unit) exists here, exposed to tests via `Wire` with fake Transport. This makes
   phase 3 strictly additive.
4. **Phase 3 — Replication** (`out/udp`, `peers.go`, `repl.go`, `sync.go`, `node.go` loop,
   `api/envelope` proto). Needs engine (versions/merge) and persistence (restart-from-dump
   scenarios). **The simulator lands in this phase, not in hardening** — convergence is a
   replication property; hardening only extends scenario coverage. Sim needs: seeded chaos
   net, virtual clock, 2..N nodes, quiescence detection, convergence + bounds assertions.
5. **Phase 4 — API** (`api/supkv` proto, `in/grpc`, Watch, `wait_for_replication`, gate
   wiring, flags/env, `--version`). After replication because the startup gate's "first full
   sync round" signal and `wait_for_replication` ack events only exist once replication does;
   Watch must carry replicated events. Delivers the real user-facing binary.
6. **Phase 5 — Hardening** (full failure-scenario suite on the sim, metrics endpoint +
   discard counters, conflict journal polish, drift monitor loop, benchmarks: dump@10k keys,
   replication throughput, 1000 writes/s bursts). Depends on everything; mostly test/observ-
   ability code, low architectural risk.
7. **Phase 6 — v1.0** (quickstart verified against the real binary on localhost broadcast,
   ops docs, examples, release tag).

Dependency rationale (why this order is forced): engine ← persistence, replication ← API;
persistence before replication because the restart/recovery scenarios and the HLC "init from
dump max" rule presuppose dumps; replication before API because gate/Watch/wait_for_
replication consume replication events; simulator with replication because convergence under
chaos is the acceptance criterion for the replication phase itself.

**Phase-flag for the roadmap:** Phase 3 is the only phase with genuinely deep research needs
(UDP broadcast socket behaviors across Linux/macOS, own-echo semantics, buffer sizes for
60 KiB values, `sync_response_rate` mechanics). Phases 1–2 are standard patterns; Phase 4 is
standard gRPC; Phase 5 is standard test/observability work on existing seams.

## Sources

Confidence per the source hierarchy (websearch/webfetch verified = MEDIUM; the project's own
binding conventions = HIGH for project decisions):

- **go-ultimate skill + references** (`architecture.md`, `wiring-and-runtime.md`,
  `engineering-policy.md`) — local normative source for this project; binding for layout,
  wiring, shutdown. HIGH (for project decisions).
- CockroachDB `pkg/util/hlc` source (`hlc.go`, `timestamp.go`) — read directly from
  github.com/cockroachdb/cockroach. MEDIUM (verified primary).
- Prometheus Alertmanager `cluster/` package (`cluster.go`, `delegate.go`) — read directly
  from github.com/prometheus/alertmanager. MEDIUM (verified primary).
- Marmot repo tree + `shutdown.go` — read directly from github.com/maxpert/marmot. MEDIUM
  (verified primary).
- etcd raft README (github.com/etcd-io/raft) — determinism/state-machine design principle.
  MEDIUM (verified primary).
- Corrosion repo tree (github.com/superfly/corrosion, `crates/`) — module decomposition;
  note: Rust, not Go; no public DST harness. MEDIUM (verified primary).
- Go blog, "Testing Time (and other asynchronicities)" — `testing/synctest` stable in Go 1.25.
  MEDIUM (official, verified).
- FoundationDB "Simulation" docs — canonical DST structure (virtual time, seeded RNG, fault
  injection). MEDIUM (official, verified).
- ivanyu/awesome-deterministic-simulation-testing — DST tool landscape (Go options thin;
  Polar Signals WASM approach). MEDIUM-LOW (community list).

---
*Architecture research for: sup-kv — leaderless replicated in-memory KV store (CRDT, AP), Go*
*Researched: 2026-10-31*
