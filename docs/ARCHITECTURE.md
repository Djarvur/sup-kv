# sup-kv architecture

The as-built description of sup-kv: what the shipped package graph is, which
boundaries the lint gate enforces between the layers, and how a write flows
through them. It is the deep description [docs/IDEA.md](IDEA.md)'s historical
banner promises; the design intent itself stays there, bannered as history.
Every package paragraph below is written from the package's own doc comment
at the time the paragraph landed, and `TestArchitectureDocPackages` (in
`cmd/sup-kv`) binds this file to the live `go list ./...` graph — a package
added or removed without an edit here fails `make test`.

## Overview

sup-kv is a leaderless replicated in-memory key-value store for small
clusters on one trusted LAN broadcast segment: every node accepts reads and
writes locally and independently (AP), and replicas converge to identical
state — the pointwise supremum of all writes under the record-version order.
The store is a Last-Write-Wins-Element-Map CRDT whose merge is
commutative, associative and idempotent, so datagram loss, duplication and
reordering cannot affect the outcome. Record versions are Hybrid Logical
Clock readings packed into RFC 9562 UUIDv7, so correctness never depends on
wall-clock synchronization. The core value: every node stays writable at all
times with no coordination, and a write confirmed to a client is never lost.

## Package graph

`go list ./...` yields nineteen packages. They group into layers: the public
API, the pure decision core, the port seam between them, the adapters, the
composition root, and the test harness packages. One paragraph per package,
each grounded in the package's own doc comment.

### Public API

**`api/supkvapi/v1`** — the client-facing wire contract of a sup-kv node:
the seven RPCs of SupKVService (Get, Set, Delete, List, Watch — the only
server-streaming method — SnapshotCreate and SnapshotLoad), every
state-carrying message a full Record. The proto schema is the source of
truth; the committed generated files are produced by `make proto`. The
package exists to be importable from other modules and must never import
`github.com/Djarvur/sup-kv/internal` — the boundary is machine-enforced by
the `sup-kv-public-api-no-internal` rule.

### Decision core (`internal/app/*` — pure, no I/O)

**`internal/app/version`** — the record version: a 16-byte RFC 9562 UUIDv7
carrying the HLC reading (unix_ts_ms, counter) and the writing node's ID.
The big-endian layout guarantees byte-wise comparison equals
`(l, counter, nodeID)` order — the total order the store's LWW join sorts
by. Standard library only.

**`internal/app/clock`** — the Hybrid Logical Clock (Kulkarni et al., 2014):
node state `(l, c)`. The wall clock enters through exactly one seam — the
`now func() time.Time` constructor parameter; a restart seeds the clock with
the dump's full watermark so every post-restart reading dominates it, and
readings saturate at the 48-bit timestamp ceiling instead of wrapping out of
the version domain. Standard library only.

**`internal/app/storage`** — the in-memory LWW-Element-Map store. Per key,
the record with the greater `version.RecordVersion` wins on merge. Tombstone
visibility is deliberately asymmetric: Get treats missing and tombstoned
alike as ErrNotFound, SetIfAbsent treats any existing record (tombstone
included) as ErrAlreadyExists, so delete-then-create-if-absent cannot
resurrect a key; tombstones stay forever. Merge reports every displaced
version as a Conflict and every clock-advancing operation returns a drift
signal — nothing is logged or emitted inside the package. The store declares
the consumer-side `Clock` interface the concrete clock satisfies. Entry
points: Get/Set/SetIfAbsent/Delete, Merge, Version/Lookup/Versions,
ValueMaxBytes. Standard library only.

**`internal/app/persist`** — the persistence decision core: the pure JSONL
dump codec plus the write-confirmation engine, saving whole dump bytes
through the port.DumpStore seam. `Engine.Set`/`SetIfAbsent`/`Delete` block
until the covering dump resolves: nil confirms the write is on disk
(PERS-04), ErrDumpFailed reports the failed dump with the record still
readable, ErrUnconfirmed the deadline expiry. The dump scheduler runs in the
caller-launched `Run` goroutine with the bounded retry ladder (100 ms
doubling to a 5 s cap); Snapshot/Restore replay through the same engine. The
loader (Open) classifies every rejected generation and never deletes
evidence. Standard library plus the sibling core packages only.

**`internal/app/repl`** — the replication decision core: a pure,
synchronous state machine owning every replication decision — datagram
discard classification with its counters, the peer table over opaque address
handles, Hello/DataReplication/Ack/DataRequest/DataSync processing, the
paced egress and the retransmit ladder. `NotifyLocalWrite` is the only way a
local write reaches the core; `HandleDatagram` the only inbound door. The
package takes no locks and starts no goroutines — exactly one driver (the
production adapter's event loop or the simulator) calls the handlers, and
the wall clock enters only through handler parameters. Standard library and
own module only.

### Core test harness (`internal/app/internal/*`)

**`internal/app/internal/simnet`** — the deterministic simulator: a
discrete-event scheduler ordered by (Time, Seq), a fault-injecting virtual
network and an N-node cluster harness driving the same repl.Core handlers
the production UDP adapter drives — no simulation mode anywhere. It starts
no goroutines and never reads a wall clock or sleeps; every random draw
flows through the one rng injected at construction, so the same seed
reproduces the whole run byte for byte. See
[docs/SIMULATOR.md](SIMULATOR.md).

**`internal/app/internal/testutil`** — the seeded `math/rand/v2` property
generators shared by the core packages' property tests (no third-party PBT
library), re-exporting the module-wide percentile reporter for its own
consumers.

### Ports

**`internal/port`** — the interface definitions that decouple the pure core
from the outside world: DumpStore (whole dump bytes between the persistence
engine and the file adapter) and the Repl message types (decoded message
structs, because protobuf cannot cross into `internal/app`). The lint gate
restricts this package to standard-library imports plus the own module path
(`sup-kv-port-stdlib-only`).

### Adapters

**`internal/persistfile`** — the dump file adapter: the only os-importing
persistence code, implementing port.DumpStore with atomic writes (temp file,
fsync, rotate, rename), `.prev` rotation and the main → `.prev` → empty
fallback chain. Load preserves every rejected generation under
`.corrupt-<UTC>` and sweeps old orphan temps at startup — no load ever
deletes evidence. On darwin the durability flush is F_FULLFSYNC (a strict
superset of File.Sync; see ADR 0010 for the platform's status).

**`internal/udprepl`** — the UDP replication adapter: the only net- and
protobuf-importing replication code — mechanical by design. It maps port
message structs onto the wire schema, verifies the CRC before any body
decode, owns the socket (SO_RCVBUF/SO_SNDBUF set and read back at
construction) and feeds the pure core from one event-loop goroutine. It owns
no protocol decision; production binds never set SO_REUSEADDR, so a second
process fails with EADDRINUSE instead of silently splitting traffic.

**`internal/udprepl/replproto/supkv/v1`** — the replication wire schema:
one Envelope per UDP datagram carrying a CRC32 over the marshaled Body and
one of the five protocol messages inside the Body oneof. Internal — unlike
the public API package, no client imports it.

**`internal/grpcapi`** — the gRPC API adapter: the Server implementing the
seven RPCs over the persist.Engine store facade, the startup gate
(ServerOptions: the sync-arrived signal source and the backstop deadline),
the wait_for_replication seams and tracker, the Watch hub with its exact
snapshot-then-live cut, and mapError — the one RPC-to-status mapping. The
zero ServerOptions value is legal: a single-node server whose gate opens
after one second and whose waits report the honest zero-peer answer.

**`internal/grpcapi/apiproto/supkvapi/v1`** — the legacy home of the client
API, kept only so importers that predate the public move keep compiling; it
re-exports the generated identifiers as thin aliases. New code uses
`api/supkvapi/v1`. It deliberately carries no Go "Deprecated:" marker: the
frozen red tests must keep importing it byte-unchanged.

**`internal/metrics`** — the metrics/debug endpoint adapter: the repo's only
HTTP-serving code, rendering the counter seams as the classic Prometheus
text format and assembling the one opt-in listener's whole debug surface
(`/metrics` plus `/debug/vars` and `/debug/pprof/*`, optionally behind
client-certificate mTLS). Values come from the closed Sources snapshot set
the caller supplies; the adapter owns rendering and surface assembly only —
main owns the listener lifecycle.

### Composition

**`internal/config`** — the single parse, validate and usage point for the
launch configuration: it resolves every SUPKV_ env twin into a flag default
(flag > env > default), rejects unknown flags and positional arguments by
name, and owns Validate — the one function every startup range and relation
check lives in, so misconfiguration fails once, naming the parameter, before
any socket or file work. It deliberately lives outside `internal/app`: it
reads the environment and imports the adapters whose datagram ceiling it
validates, which the core's no-I/O rule must not cover.

**`cmd/sup-kv`** — the single static binary. Per the layout contract, main()
is wiring only: configuration resolves once, is validated, then `wire`
connects the adapters and the core (the gRPC server receives the engine
facade, the replication adapter's NotifyLocalWrite and the SyncArrived
mirror); signals and the process lifecycle live here, and the pinned
graceful-shutdown order's final step is the engine's write-confirmed dump.
`--version` reports the ldflags-stamped identity.

### Test harness (outside the app tree)

**`internal/testutil`** — the module-wide test helpers: the shared p50/p99
percentile reporter (Go's internal-package rule forbids `internal/udprepl`
and `internal/grpcapi` from importing the app-internal testutil, so all
benchmark packages share this one implementation) and the socket-owning
packages' one port allocator — disjoint per-package bands below the kernel's
ephemeral range, claimed across the bind probe.

**`internal/stand`** — the two-process real-socket test harness (Linux
only): it spawns the real cmd/sup-kv binary as child processes over loopback
UDP and gRPC, drives the whole lifecycle (build-once, launch, SIGKILL and
SIGTERM exit contracts, reaping) and polls convergence through the serving
endpoints. The scenario suite, red tests, perf rows and the nightly run all
build on it.

## Enforced boundaries

Six depguard rules in `.golangci.yml` hold the layering — the machine form
of the architecture. The written justifications (including the one allowed
composition edge) are recorded in [AGENTS.md](AGENTS.md); this section names
what each rule forbids and stops there — AGENTS.md, not this file, is where
the boundary contract lives.

| Rule | Held between | Forbids |
|------|--------------|---------|
| `sup-kv-app-no-io` | `internal/app/**` | imports of `net`, `os`, `syscall` and `google.golang.org/*` — the decision core stays pure |
| `sup-kv-port-stdlib-only` | `internal/port/**` | everything except standard library and the own module path — ports are pure interface definitions |
| `sup-kv-public-api-no-internal` | `api/**` | imports of `github.com/Djarvur/sup-kv/internal` — the published client package stays importable from another module |
| `sup-kv-adapters-not-in-core` | `internal/app/**`, `internal/port/**` | imports of any adapter — adapters are wired only from cmd |
| `sup-kv-metrics-no-adapters` | `internal/metrics/**` | imports of `internal/udprepl`, `internal/persistfile`, `internal/grpcapi` — the exporter reads value seams, never a socket, dump or API adapter |
| `sup-kv-grpcapi-no-repl-metrics` | `internal/grpcapi/**` | imports of `internal/udprepl` and `internal/metrics` — the API adapter consumes port seams only |

## Parameter surface

Every launch knob enters through one door: `internal/config` registers the
flags, resolves their `SUPKV_` env twins and validates the ranges and
relations in one pass, and `cmd/sup-kv`'s `wire` is the only place the
parsed values become adapters and core configuration. The as-built flag
table — every registered flag with its default and env twin, in `--help`
order — is [docs/CONFIGURATION.md](CONFIGURATION.md), which the
`TestConfigurationDoc` iterator keeps byte-honest against the binary; this
document carries the surface's *role* (one parse, one validate, one wiring
point), never a second copy of the table. The five env-only knobs
(`SUPKV_LOG_LEVEL`, `SUPKV_LOG_FORMAT`, and the `SUPKV_METRICS_CERT` /
`SUPKV_METRICS_KEY` / `SUPKV_METRICS_CLIENT_CA` mTLS triple) enter outside
the flag parser by the same rule: they are documented where the table is,
not re-listed here.

## Data flow

**A local write.** A gRPC `Set` arrives at `internal/grpcapi`'s Server: the
startup gate is the handler's first statement (a closed node answers
Unavailable), then the key and value bounds. The call enters
`internal/app/persist`'s Engine facade, whose Set stamps the version — the
clock (`internal/app/clock`) through the store's consumer-side Clock
interface, packed by `internal/app/version` — and applies it into
`internal/app/storage`'s LWW map. The engine registers a confirmation
waiter (the dump-confirm discipline), and the handler then notifies the
replication core through the adapter's `NotifyLocalWrite` —
`internal/app/repl` broadcasts the record into its Outbox, and
`internal/udprepl`'s event loop paces it onto the wire. Only after the
covering dump resolved does the RPC return; `wait_for_replication`, when
requested, resolves separately from the Outbox result stream once the peer
Acks (processed, not merely received) cover the wait set.

**The replication path (receiving node).** A datagram lands in
`internal/udprepl`, which verifies the CRC before any body decode and hands
the decoded port message to `internal/app/repl`'s `HandleDatagram` — the
core validates (protocol version, version layout, the max-clock-lead future
bound), absorbs the record's version into the clock, then applies it through
the engine's ApplyRemote into the store by plain LWW — or answers a strictly
older replica with one unicast reply carrying the held record, no Ack. An
applied record Acks. Every discard is classified and counted, never silent.

**The persistence path.** The engine's dump scheduler (its `Run` goroutine)
debounces writes (the quiet window, bounded by `dump_max_wait` from the
oldest unconfirmed write), streams the store's records through the codec —
one sorted JSON line per record, byte-stable — into
`internal/persistfile`'s FileStore, which writes atomically: temp file,
fsync, `.prev` rotation, rename. Completed dumps resolve the confirmation
waiters; a failed dump arms the bounded retry ladder and leaves the records
readable and replicating.

## Client-interop conventions

Two conventions govern how clients use the store across services; both are
documented consumer contracts, not store behavior:

- The **key-prefix convention** — services sharing a cluster keep their keys
  under one '/'-prefixed namespace each, so concurrent services never
  collide and List/Watch prefixes line up with ownership. The store's map is
  deliberately prefix-agnostic.
- The **client-side last-read-value cache** — a consumer that must answer
  through an outage keeps its own fallback cache; sup-kv ships no
  client-side cache and no RPC promises one (ADR 0007).

The patterns, their boundaries and the reasoning live in
[docs/API.md](API.md) (Key-prefix convention, Client patterns); this
document names them for orientation and defers to that reference — one copy
of each pattern.

## Where to go next

- Conventions, the recorded deviations and the dependency policy live in
  [AGENTS.md](AGENTS.md) — the canonical conventions file; the boundaries
  above are its machine form, not a restatement of it.
- Operating a cluster: [docs/OPERATIONS.md](OPERATIONS.md) (trust model,
  failure behaviour, runbooks, metrics, log catalog).
- On-disk and wire formats: [docs/FORMATS.md](FORMATS.md).
- Measured performance and the measurement rules:
  [docs/PERFORMANCE.md](PERFORMANCE.md).
- Testing and the deterministic simulator: [docs/SIMULATOR.md](SIMULATOR.md).
- Recorded decisions: [docs/adr/README.md](adr/README.md) (ADR 0001–0012).
