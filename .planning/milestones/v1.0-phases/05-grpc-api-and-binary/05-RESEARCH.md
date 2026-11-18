# Phase 5: gRPC API and Binary - Research

**Researched:** 2026-11-08
**Domain:** gRPC service layer over an existing pure decision core (Go 1.27, zero-direct-dep policy, depguard architecture boundary); binary packaging, config, graceful shutdown
**Confidence:** HIGH (codebase seams verified by reading source this session; grpc-go dependency facts probe-verified in a scratch module; gRPC shutdown/health semantics cited from official docs and web-verified)

## Summary

Phase 5 is a pure adapter phase: every storage, persistence and replication decision already exists behind exported seams (`persist.Engine`, `repl.Core`, `udprepl.Adapter`), and the gRPC layer's job is to translate RPCs onto them without adding a single new decision into `internal/app` beyond four small additive seams. The critical planning insight is that **four things are missing today and must be added before the gRPC handlers can satisfy API-03/04/05 and D-19**: (1) a change-event tap — `storage.Store` has no observer, and the *only* point where "actual state change" (D-06) is decidable for every source (local writes, remote merges, replicated tombstone deletes, snapshot-restore replay) is `storage.Store.install` under the store write lock; (2) a first-DataSync-received signal in `repl.Core` for the startup gate — the core records no such state today (verified by reading all 912 lines of `core.go`); (3) an `Acked` set in `repl.PendingInfo` — `WaitingOn` alone cannot distinguish "peer Acked" from "peer evicted by peer_timeout" (handleAck vs evictPeers both shrink the wait-set), which would corrupt the D-08 aggregate report; (4) an exported `Engine.Flush()` — `flush()` is unexported and `Run`'s exit sweep resolves waiters with `ErrClosed` *without* a final dump, so the D-19 "final dump with write-confirmation" has no callable today.

Dependency-wise this is the phase where gRPC enters `go.mod` per the AGENTS.md policy. Empirically probed in a scratch module this session: **grpc-go v1.85.0** (go directive `go 1.26.0`, compatible with the repo's `go 1.27.1`) pulls exactly four indirects — `golang.org/x/net`, `x/sys`, `x/text`, `google.golang.org/genproto/googleapis/rpc` — and selects exactly `protobuf v1.36.12`, the version already a direct require. `protoc-gen-go-grpc v1.6.2` (latest) builds via `go build` from the module pin, generates stubs requiring `grpc.SupportPackageIsVersion9` and the `mustEmbedUnimplementedSupKVServer` pattern, and the full protoc → generate → compile → test round-trip was verified with the machine's protoc 3.21.12 under go 1.27. grpc v1.86.0 appears in the proxy version list but its tag fetch failed twice this session (TLS timeouts); v1.85.0 is the verified pin.

**Primary recommendation:** one new adapter package `internal/grpcapi` (+ `internal/grpcapi/apiproto` for the committed generated code, following the `udprepl/replproto` and Makefile `proto` precedent — protoc, not buf), grpc-go v1.85.0 + protoc-gen-go-grpc v1.6.2 as a third `tool` directive, the four additive seams above, a hub-goroutine watch fan-out with per-subscriber bounded channels and disconnect-on-full (etcd-style, preserving D-02 "no holes"), a gate-resolution goroutine flipping `health.SetServingStatus("")`, NOT_SERVING→SERVING, and a `Type=simple` systemd posture (no sd_notify dependency in v1).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Watch stream (API-03)**
- **D-01:** A Watch event carries the full record — key, version, value, deleted flag. The subscriber never needs a second Get to act on a change. — **Reversibility:** costly — this is the client-facing wire contract.
- **D-02:** Tombstones are visible: deletes arrive as events with `deleted=true`. The stream has no holes — a subscriber sees every state change.
- **D-03:** Subscription starts with a snapshot of the current state followed by live changes. The snapshot is delivered as ordinary Watch events in the same single format — no special `initial_state` message type.
- **D-04:** Watch accepts a prefix filter, symmetric with List (API-02). `prefix=""` means all keys.
- **D-05:** Event order is arrival order at the node (local and replicated alike). No global re-sorting by version; the stream is notifications, not a log.
- **D-06:** Events fire only on actual state changes; a merge loser is silently absorbed (no event for a version that did not become the stored record).
- **D-07:** No origin marking — local and replicated changes are indistinguishable in the stream.

**wait_for_replication (API-05)**
- **D-08:** The report is an aggregate: `acked` boolean plus counts (`acked_count` / `total_peers`). No per-peer breakdown.
- **D-09:** The wait timeout is a per-request RPC field with a config default (`wait_timeout`). A client may wait longer than the default on a cold start; the default is operator-tunable.
- **D-10:** The peer set is the snapshot of known peers at write time. Peers discovered (Hello) during the wait do not extend the awaited set.
- **D-11:** Zero known peers ⇒ `acked=true` immediately.

**List under prefix (API-02)**
- **D-12:** One RPC returns the full sorted key list for the prefix — no pagination.
- **D-13:** List returns keys only.
- **D-14:** Tombstones are not listed — List reflects live state.
- **D-15:** Empty prefix means all keys.

**Binary and configuration (TOOL-01)**
- **D-16:** Environment variables use the `SUPKV_` prefix: `SUPKV_NODE_ID`, `SUPKV_BIND_ADDR`, etc. — **Reversibility:** one-way.
- **D-17:** Precedence: flag > env > default.
- **D-18:** `--version` prints version + commit + Go build version, injected via ldflags at build time.
- **D-19:** SIGTERM triggers a graceful shutdown: stop accepting gRPC requests, stop replication, then a final dump with write-confirmation. Committed data is never left only in memory across shutdown.

### Claude's Discretion
- Codegen layout and tooling for the gRPC .proto (package placement, protoc+Makefile precedent vs buf).
- Internal structure of the API adapter between the gRPC service and `persist.Engine`/`repl.Core` — follow the phase-4 D-01 decoded-structs port-seam precedent.
- gRPC server options (keepalive, max message sizes consistent with `value_max_bytes`, interceptors).

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| API-01 | gRPC service with protobuf definitions: Get/Set (incl. create-if-absent)/Delete with errors OK/NotFound/AlreadyExists/InvalidArgument/Unavailable | Engine surface verified (Set/SetIfAbsent/Delete block until dump-confirmed); sentinel errors already document the 1:1 RPC-code mapping — see Code Examples; codegen round-trip probe-verified |
| API-02 | List keys under a prefix, sorted | `Engine.Records()` returns tombstones-inclusive full snapshot (D-14 filtering is adapter-side); `slices.Sorted` over filtered keys |
| API-03 | Watch stream covering local and replicated changes (engine event taps designed in from the start) | Gap analysis: the only correct single tap point is `storage.Store.install` (covers local writes, remote merges, replicated tombstones, restore replay; fires only on actual install = D-06); watch-hub + boundary-cut algorithm below |
| API-04 | Writes Unavailable during startup gate; reads immediate | Gate = first-DataSync-arrived OR `startup_gate` elapsed; repl.Core has NO such state today (verified) — additive seam + udprepl atomic mirror + gate-resolution goroutine |
| API-05 | `wait_for_replication`: wait for Acks within timeout, report, never change write outcome | `PendingSnapshot` seam exists but cannot distinguish ack from eviction — additive `Acked` field recommended; wait loop polls an adapter-mirrored copy (core is single-goroutine); D-11 short-circuit |
| API-06 | Standard gRPC health endpoint reflecting gate state | `google.golang.org/grpc/health` in-module (no new dep); probe-verified `SetServingStatus("")` NOT_SERVING→SERVING transitions; "" = whole-server semantics cited from grpc.io |
| API-07 | Snapshot create/load through the API | `Engine.Snapshot`/`Engine.Restore` exist and are exactly the right shape; snapshot names already validated at persistfile (path traversal closed) → map `ErrInvalidSnapshotName` to InvalidArgument |
| TOOL-01 | Single static binary, flags/env config, `--version`, systemd-friendly | stdlib `flag` + `SUPKV_` env (flag>env>default pattern below); ldflags `-X` injection; SIGTERM shutdown sequence (D-19) with GracefulStop-timeout pitfall; Type=simple recommendation |
</phase_requirements>

## Project Constraints (from AGENTS.md)

- **Go version:** pinned in `go.mod` (`go 1.27.1`); do not raise without an ADR. [VERIFIED: go.mod:3]
- **Gates before push:** `make lint` (`go tool golangci-lint run ./...`) and `make test` (`go test -race ./...`); CI runs the same targets. [VERIFIED: AGENTS.md + Makefile:6-12]
- **Depguard boundary (machine-enforced):** `internal/app` may not import `net`, `os`, `syscall`, `google.golang.org/*` (rule `sup-kv-app-no-io`); `internal/port` may import stdlib and the own module path only (rule `sup-kv-port-stdlib-only`). Rules live in `.golangci.yml`; any boundary change is a visible `.golangci.yml` diff. [VERIFIED: .golangci.yml:70-88] Consequence: the generated `*.pb.go` (which imports `google.golang.org/grpc`) must live OUTSIDE `internal/app` and `internal/port` — e.g. `internal/grpcapi/apiproto`.
- **Vendored lint config:** `.golangci.yml` is the powerman strict config for golangci-lint 2.15.0, byte-unmodified; project rules additive inside `linters.settings.depguard.rules` only. The strict `main` deny rule already forbids `github.com/golang/protobuf` (use `google.golang.org/protobuf`) and old grpc-middleware paths — the planned stack complies. [VERIFIED: .golangci.yml:30-51]
- **No lint suppressions without written justification:** every `//nolint` names its linter and carries a reason (`nolintlint.require-explanation`, `require-specific`). Generated `.pb.go` files are lint-exempt via the path pattern `\.[\w-]+\.go$` in both linter and formatter exclusions. [VERIFIED: .golangci.yml:491-521]
- **Layout:** `cmd/sup-kv/` single binary; `main()` holds no testable logic (flags/env/signals live there from Phase 5, nothing else); adapter packages appear with the phases that need them; no empty adapter directories. [VERIFIED: AGENTS.md]
- **Dependency policy:** Phase 1 carries zero direct runtime requires; gRPC and protobuf enter `go.mod` in the phases that first use them — this phase. Tools always invoked via `go tool <name>`, never `tools.go`, never bare `go install`. The tool-directive indirect graph (~200 lines) is the accepted reading; gRPC now also becomes a *direct runtime* require. [VERIFIED: AGENTS.md + go.mod:5-11]
- **CI tidy gate:** `go vet ./... && go test -race ./... && go mod tidy && git diff --exit-code go.mod go.sum` — the new dependency set must be tidy-exact. [VERIFIED: .github/workflows/ci.yml:59-63]
- **CI matrix:** Linux + macOS, amd64 + arm64 — the binary and tests must stay cross-platform (no cgo, no syscall tricks outside the existing udprepl socket options; CGO_ENABLED=0 static build is the natural fit). [VERIFIED: REQUIREMENTS.md CI-01]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| RPC surface (Get/Set/Delete/List/Watch/Snapshots) | API adapter (`internal/grpcapi`) | — | Thin translation layer; all decisions live in `internal/app` behind existing seams (depguard forbids gRPC in the core) |
| Change-event generation (Watch source) | Decision core (`internal/app/storage`) | grpcapi watch hub | Only `storage.Store.install` knows "this record actually became the stored record" (D-06) across local/remote/restore sources |
| Watch fan-out, slow-consumer isolation, prefix filter | grpcapi watch hub (one goroutine) | per-subscriber buffered channels | Never block the store; hub serializes fan-out making the snapshot/live cut exact |
| Startup gate decision | grpcapi gate resolver (goroutine) | repl.Core (new sync-arrived seam), udprepl (atomic mirror) | Gate = pure input composition (sync-arrived OR deadline); state lives in core, mirror crosses goroutines |
| wait_for_replication wait/report | grpcapi write handler | repl.Core (`PendingSnapshot` + new Acked), udprepl (mirrored pending copy) | Pure read/poll of replication state; write outcome already decided by engine |
| Health status | grpcapi gate resolver | grpc in-module health server | API-06 binds health 1:1 to gate state |
| Write confirmation / durability | `internal/app/persist.Engine` | — | PERS-04 unchanged; RPC codes mapped from sentinels |
| Replication notification of writes | `udprepl.Adapter.NotifyLocalWrite` | — | Phase-4 D-05 explicit-call contract; gRPC write handler is "the author" |
| Config/flags/env/signals/shutdown orchestration | `cmd/sup-kv/main.go` | — | AGENTS.md: main holds flags/env/signals, nothing testable |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/grpc | v1.85.0 | gRPC server, client, health, bufconn test transport | The canonical Go gRPC implementation (google.golang.org namespace); the only dependency the design permits (IDEA.md §Product principles: "Minimal dependencies: gRPC + protobuf") [VERIFIED: scratch-module probe — server+health+bufconn compile and run; go directive `go 1.26.0` read from the module cache] |
| google.golang.org/protobuf | v1.36.12 | protobuf runtime + protoc-gen-go plugin | Already the direct require (go.mod:11); grpc v1.85.0's own MVS selection is exactly v1.36.12 — zero-version-change integration [VERIFIED: probe go.mod resolution] |
| google.golang.org/grpc/cmd/protoc-gen-go-grpc | v1.6.2 (tool directive) | gRPC stub code generator | The official generator; latest release; builds from the module pin exactly like the existing protoc-gen-go tool pattern [VERIFIED: probe — `go build` of the tool and generated stubs compile against grpc v1.85.0] |

### Supporting (indirect, arrives via tidy)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| golang.org/x/net | v0.60.0 | HTTP/2 (grpc transport) | Indirect only — grpc-go imports it directly [VERIFIED: probe tidy graph] |
| golang.org/x/sys, golang.org/x/text | v0.48.0 / v0.42.0 (or repo's newer) | grpc internals | Indirect only; both already present in the tool graph [VERIFIED: probe + go.mod:215,217] |
| google.golang.org/genproto/googleapis/rpc | v0.0.0-2026… | status/error-details types | Indirect only [VERIFIED: probe tidy graph] |

Notable absence vs older grpc integrations: **no `github.com/golang/protobuf` shim** appears in the v1.85.0 graph — good, since the lint config denies it. [VERIFIED: probe final go.mod]

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| protoc + Makefile target (extended) | buf | One .proto file; buf adds a config surface and its own dependency for zero gain; the repo already has a working `proto` Makefile pattern (04-01) and committed generated output — **use protoc** [VERIFIED: Makefile:22-29 precedent] |
| grpc-go v1.86.0 | v1.85.0 | v1.86.0 is listed on the proxy but its tag fetch failed twice this session (TLS timeout / unknown revision) — flaky evidence; v1.85.0 is fully probe-verified. Pin v1.85.0; Dependabot's weekly gomod group will carry upgrades [VERIFIED: failed fetch attempts + working v1.85.0 probe] |
| stdlib `flag` | spf13/pflag | pflag is in the *tool* graph (indirect via cobra) but promoting it to direct contradicts the minimal-deps principle; stdlib `flag` accepts both `-flag` and `--flag` and fully covers D-16..D-18 — **use stdlib** [CITED: flag package behavior; ASSUMED: single-dash acceptance adequate] |
| Type=simple systemd | Type=notify + sd_notify | notify needs coreos/go-systemd (new dep) or a hand-rolled NOTIFY_SOCKET writer; v1's health-gate already gives readiness to clients; simple + Restart=on-failure is functionally fine — **simple for v1**, notify documented as upgrade path [CITED: freedesktop.org systemd.service; coreos/go-systemd] |

**Installation:**
```bash
go get google.golang.org/grpc@v1.85.0 google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
go mod tidy   # CI tidy gate demands the exact result be committed
```
`go.mod` result: `google.golang.org/grpc v1.85.0` joins `google.golang.org/protobuf v1.36.12` as the two direct runtime requires; the `tool (...)` block gains `google.golang.org/grpc/cmd/protoc-gen-go-grpc`. [VERIFIED: probe]

**Version verification (run this session):** `go list -m -versions google.golang.org/grpc` → latest stable line v1.83.2 / v1.84.0 / v1.85.0 (+v1.86.0 listed but unfetchable); `go list -m -versions google.golang.org/grpc/cmd/protoc-gen-go-grpc` → v1.6.2 latest. protoc on this machine: `libprotoc 3.21.12` (the Makefile comment already standardizes "3.21.x"); local go1.27.3.

## Package Legitimacy Audit

> The seam's `package-legitimacy check` supports npm/pypi/crates only — Go is outside its coverage (verified by invocation this session). For Go modules the equivalent legitimacy mechanism is the module proxy + checksum database, and the strongest available verification is empirical: this session downloaded the exact modules from proxy.golang.org (sumdb-verified), built the tooling, generated stubs, compiled and ran a real server+client against them.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| google.golang.org/grpc v1.85.0 | Go module proxy | ~13 yrs (project) | de-facto standard | github.com/grpc/grpc-go (googlesource mirror) | OK (empirical) | Approved |
| google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.2 | Go module proxy | ~5 yrs | official toolchain | same repo, cmd/ subdir | OK (empirical) | Approved |
| google.golang.org/protobuf v1.36.12 | Go module proxy | in repo already | — | github.com/protocolbuffers/protobuf-go | OK (already direct require) | Approved |
| google.golang.org/genproto/googleapis/rpc | Go module proxy | ~6 yrs | standard | google.golang.org/genproto | OK (indirect via grpc) | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

No package names were taken from WebSearch or training data alone: every recommendation above was fetched, built and executed from the registry this session.

## Architecture Patterns

### System Architecture Diagram

```
                          sup-kv binary (cmd/sup-kv, static, CGO_ENABLED=0)
 ┌────────────────────────────────────────────────────────────────────────────┐
 │ flags/env (SUPKV_*, flag>env>default) ── ldflags version ── SIGTERM/SIGINT │
 │                                                                            │
 │  gRPC clients ──TCP──▶ grpc.Server (:bind_addr)                            │
 │                          ├─ health (grpc_health_v1) ◀── gate resolver ─────┼──▶ SetServingStatus("")
 │                          ├─ SupKV service (internal/grpcapi)               │ │   NOT_SERVING ─▶ SERVING
 │                          │   Get/List ──── Engine.Get / Records            │ │   (sync-arrived OR startup_gate)
 │                          │   Set/Delete ── gate check ─▶ Engine.Set/...    │ │
 │                          │   │                     └─▶ Adapter.NotifyLocalWrite
 │                          │   │   (wait_for_replication: poll mirrored     │
 │                          │   │    PendingSnapshot; Acked seam; D-11)      │
 │                          │   Watch ──▶ watch hub goroutine                 │
 │                          │              ▲ tap (storage.install, COW)      │
 │                          │              │  snapshot-then-live, prefix,    │
 │                          │              │  bounded chans, disconnect-full │
 │                          │   SnapshotCreate/Load ─▶ Engine.Snapshot/Restore
 │                          ▼                                                │
 │  UDP segment ◀──▶ udprepl.Adapter (event loop, D-04) ──▶ repl.Core        │
 │        Hello/DataReplication/Ack/DataRequest/DataSync   (pure, 1 goroutine)│
 │                          │ new: sync-arrived seam ─▶ atomic mirror ──▶ gate│
 │                          ▼                                                │
 │  persist.Engine (Run dumper) ──▶ persistfile (atomic JSON dump)            │
 │        Set blocks until dump-confirmed (PERS-04)                           │
 │        new: exported Flush() ◀── shutdown: final confirmed dump (D-19)     │
 └────────────────────────────────────────────────────────────────────────────┘

Shutdown (D-19, in order): health NOT_SERVING → cancel Watch ctxs →
GracefulStop (goroutine + timeout → Stop fallback) → cancel repl ctx →
Engine.Flush() (final confirmed dump) → cancel engine ctx → exit 0
```

A gRPC write flows: client → gate check (Unavailable if gated) → `Engine.Set` (blocks until dump-confirmed; sentinels map to codes) → `NotifyLocalWrite` (enqueue, never blocks) → optional wait poll → response. A replicated change flows: UDP → udprepl loop → `repl.Core.handleDatagram` → `Engine.ApplyRemote` → `storage.Merge` → `install` → tap → watch hub → subscribers.

### Recommended Project Structure
```
cmd/sup-kv/main.go          # flags/env/signals/wiring ONLY (AGENTS.md); no testable logic
internal/grpcapi/           # the gRPC adapter package (this phase's new adapter)
  server.go                 # SupKV service implementation over Engine + repl seams
  watch.go                  # watch hub: subscription, snapshot cut, fan-out, drop policy
  gate.go                   # startup gate resolver + health wiring
  wait.go                   # wait_for_replication poll loop (or folded into server.go)
  server_test.go            # bufconn integration tests (external _test package)
internal/grpcapi/apiproto/  # committed generated code + .proto (replproto precedent)
  supkvapi.proto
  supkvapi.pb.go            # lint-exempt via \.[\w-]+\.go$ exclusion
  supkvapi_grpc.pb.go
internal/app/storage/store.go   # ADDITIVE: OnChange tap in Config, fired from install()
internal/app/repl/core.go       # ADDITIVE: sync-arrived state + PendingInfo.Acked
internal/app/persist/engine.go  # ADDITIVE: exported Flush()
internal/udprepl/repl.go        # ADDITIVE: mirror sync-arrived (+ pending copy) into atomics
Makefile                    # proto target extended with protoc-gen-go-grpc
```

### Pattern 1: The four additive seams (what Phase 5 must add inside internal/app — nothing more)

**What:** Four small, purely additive changes; the gRPC layer consumes everything else through existing exported methods.
**When to use:** These are forced — each maps to a requirement nothing else can satisfy.

1. **Change tap for Watch (API-03).** `storage.Store` gains an optional callback in `Config` (the `repl.Config.Log func(msg string)` precedent [VERIFIED: core.go:86-96]), fired from `install()` under the store write lock with the installed record (key + final Record incl. recomputed checksum and Deleted flag). `install` is the *single* point where every state change converges — local writes (`Set`/`SetIfAbsent`/`Delete` → `write`), remote merges (`Merge` → `join` → `install`), replicated tombstone deletes, and snapshot-restore replay (`Restore` → `replayRecord` → `store.Set`/`store.Delete`) — and it fires exactly when a record "became the stored record", which is D-06 verbatim. [VERIFIED: store.go:145-156,210-238,491-498 engine replay path] An engine-level tap is *insufficient*: `Merge`'s public return cannot distinguish equal-version redelivery (no change, no event) from a fresh-key install (change, event) — both return nil conflicts — and `Get` cannot see winning tombstones. [VERIFIED: store.go:189-226,114-126] Caveat to document on the callback: "runs under the store write lock; must not block" (fan-out is non-blocking channel sends). `Open`'s initial load fires events into an empty hub — harmless.
2. **First-DataSync-arrived state (API-04).** `repl.Core` gains an internal flag set in `handleDataSync` (any received DataSync chunk) plus a copying accessor; `udprepl` mirrors it into an `atomic.Bool` after each loop iteration (the `mirrorStats` precedent). "First full sync round arrives" is read as *first received DataSync datagram*: chunk-completeness tracking would need sequence state the protocol deliberately lacks (chunks are self-sufficient, no continuation state [VERIFIED: core.go:660-671 chunkSyncEntries doc]); the `startup_gate` timeout is the backstop. Flagged as interpretation in Assumptions Log.
3. **`PendingInfo.Acked` (API-05).** `handleAck` already knows the acking peer; extend `PendingInfo` with `Acked []uint16` (sorted, the `WaitingOn` discipline). Without it, `acked_count = total − |WaitingOn|` over-reports: `evictPeers` removes peers from wait-sets without any Ack arriving [VERIFIED: core.go:469-478 handleAck vs 709-722 evictPeers]. With it: `acked_count = |Acked|` (all Acked peers are write-time peers by construction — only discovery's empty-table flush adds waiters, and D-11 short-circuits that case). No callbacks/channels — keeps "results, not callbacks" intact.
4. **Exported `Engine.Flush()` (D-19).** `flush()` exists but is unexported [VERIFIED: engine.go:535]; `Run`'s exit sweep resolves waiters with `ErrClosed` **without a final dump** [VERIFIED: engine.go:230-250], and remote-merge dirtiness (ApplyRemote, no waiter) would otherwise be lost on cancel. Shutdown calls `Flush()` *before* cancelling the engine context.

### Pattern 2: Watch hub — snapshot-then-live with an exact cut, hub-goroutine owned

**What:** One hub goroutine owns subscriber state; all taps and subscribe requests reach it via channels; fan-out is serialized on the hub.
**When to use:** API-03 + D-02/D-03/D-06.

- **Subscribe:** enqueue a subscribe request to the hub. The hub (single-threaded, so the cut is exact by construction) takes `Records()` (tombstones included — D-02), registers the subscriber, then emits snapshot events followed by buffered live events.
- **Boundary correctness without lock gymnastics:** after registering, suppress any live event for key K whose version does not *strictly dominate* the snapshot's version for K (`event.Version.Compare(snapshot[K].Version) <= 0` ⇒ already reflected in or superseded by the snapshot ⇒ suppress). Soundness: only strictly greater versions ever install (Merge law), and every event is an install — so an event not dominating the snapshot version is necessarily contained in the snapshot state. This tolerates any tap/subscribe interleaving, avoids a store-lock↔hub-lock deadlock (tap holds store.mu→hub.mu; subscribe must never take hub.mu→store.mu), and yields no misses and no duplicates. [VERIFIED premises: store.go:210-226 strictly-greater install; D-06 event=install]
- **Slow consumers:** per-subscriber bounded channel (size ~256–1024). On full: **close the stream** with a status error (e.g. `ResourceExhausted`/`Canceled` with a "watch overwhelmed, resubscribe" message) — the etcd-style policy. Dropping events would violate D-02 ("stream has no holes"); disconnect preserves the contract because resubscription re-snapshots. The store is never blocked: the tap's fan-out send is a `select` with `default` into the hub ingress; the hub's per-subscriber send is a `select` with default→disconnect.
- **Ordering (D-05):** firing the tap from `install` under the store write lock makes event order = install order (commit order) for all sources — a total order consistent with state; no re-sorting anywhere.
- **Prefix (D-04):** filter both snapshot and live events by prefix in the hub (a pure string comparison; `prefix=""` = all keys).

### Pattern 3: Startup gate + health

**What:** A gate-resolution goroutine in grpcapi: waits until (mirrored sync-arrived `atomic.Bool`) OR (`startup_gate` elapses since process start), then flips an internal `atomic.Bool` gate-open and `health.SetServingStatus("", SERVING)`.
**When to use:** API-04 + API-06. Write handlers check the gate first → `status.Error(codes.Unavailable, …)`. Reads (Get/List/Watch) never check. Health starts at `NOT_SERVING` before `Serve`. The empty service name "" is the whole-server status by the health protocol. [CITED: grpc.io/docs/guides/health-checking — "use an empty string ("") to represent the health of your whole server"] The health server also implements streaming Watch, used by client-side health checking — no extra work needed. Health flips to NOT_SERVING at shutdown start (traffic shedding, Pattern 6).

### Pattern 4: wait_for_replication

**What:** After a successful write (engine-confirmed) and `NotifyLocalWrite`:
1. `total = adapter.KnownPeers()` (existing atomic) [VERIFIED: repl.go:267-269]; if `total == 0` → report `acked=true, acked_count=0, total_peers=0` immediately (D-11).
2. Otherwise poll a mirrored pending snapshot until the entry for (key, version) disappears (drained via full Ack coverage — or superseded by a strictly greater remote version, which "fulfills the tracking promise by winning" [VERIFIED: core.go:449-455]) or the per-request timeout (default from config `wait_timeout`) fires.
3. Report `acked = len(Acked) == total` style aggregate per D-08; the write's own outcome was already returned by the engine — the wait never changes it.
**Concurrency constraint:** `repl.Core` is single-goroutine ("Not safe for concurrent use: exactly one driver goroutine…" [VERIFIED: core.go:220-225]) — the gRPC handler must NOT call `PendingSnapshot()` itself. Mirror it in the udprepl adapter after each loop iteration into an `atomic.Pointer[map[string]repl.PendingInfo]` (copy-on-write; pending sets are small at this project's scale — entries drain within ~1 RTT normally). **Poll-appearance subtlety:** `NotifyLocalWrite` is asynchronous through the 1024-deep notify queue [VERIFIED: repl.go:60,226-251]; the wait loop must first wait for the entry to *appear* in the mirrored pending snapshot before treating "absent" as drained — otherwise an instant poll reports a false "acked".

### Pattern 5: Binary, config, version (TOOL-01)

- **Config (D-16/D-17):** stdlib `flag`; for each knob, `flag.XxxVar(&v, name, envOrDefault("SUPKV_X", default), usage)` — env sets the default, an explicit flag overrides ⇒ flag > env > default with zero extra machinery. A `Resolvable Config Struct` (go-ultimate) resolved once in main and passed down. Required knobs from IDEA.md §Parameters: `node_id`, `bind_addr`, `broadcast_addr`, `dump_path` (required); plus gRPC bind (`grpc_addr` — distinct from the UDP `bind_addr`; naming is planner's), `startup_gate`, `wait_timeout` (new, D-09), and pass-throughs for existing defaults.
- **`--version` (D-18):** `var version, commit string` injected via `-ldflags "-X main.version=… -X main.commit=…"`; print version + commit + `runtime.Version()`; exit 0. Makefile `build` target gains the ldflags. (Optional belt: `debug.ReadBuildInfo()` vcs.revision — but D-18 pins ldflags as the mechanism.)
- **Static binary:** `CGO_ENABLED=0 go build` — pure-Go deps only; works on the CI matrix (linux/macOS, amd64/arm64).
- **systemd:** `Type=simple`, `Restart=on-failure`, `ExecStart=/usr/local/bin/sup-kv …`; readiness = the gRPC health endpoint (clients and orchestrators poll it); SIGTERM triggers D-19 graceful shutdown. `Type=notify` is the documented upgrade path (needs coreos/go-systemd or a hand-rolled sd_notify — rejected for v1 under the minimal-deps principle). [CITED: freedesktop.org systemd.service, coreos/go-systemd]

### Pattern 6: Graceful shutdown (D-19) — the order is the contract

1. Receive SIGTERM/SIGINT (`signal.NotifyContext` or `signal.Notify` + channel).
2. Health → `NOT_SERVING` (shed client traffic; load balancers stop routing). [CITED: grpc.io graceful-stop guidance]
3. Cancel the Watch/shutdown context so Watch handlers return — **GracefulStop blocks forever while any stream is open** [CITED: grpc-go issue #8776 + SO 58480775].
4. `GracefulStop()` in a goroutine; wait with a bounded timeout (e.g. 10s); on timeout call `Stop()` (force).
5. Cancel the replication context; wait for `udprepl.Run` to return.
6. Call the new `Engine.Flush()` — the final confirmed dump (in-flight writes already resolved through their own dumps; remote-merged dirtiness lands here).
7. Cancel the engine context (Run sweeps any residual waiter with ErrClosed — by now there are none).

### Anti-Patterns to Avoid
- **Putting generated proto or any `google.golang.org/*` import under `internal/app` or `internal/port`** — the depguard gate fails the build; the adapter owns protobuf (phase-4 D-01 precedent).
- **Calling `repl.Core` methods from gRPC handler goroutines** — the core is single-driver; cross only through adapter mirrors.
- **Dropping watch events on slow consumers** — violates D-02; disconnect instead.
- **`GracefulStop()` without a timeout or without ending streams first** — hangs on open Watch streams.
- **Skipping `go mod tidy` alignment** — the CI gate diffs go.mod/go.sum (`git diff --exit-code`) and fails the PR. [VERIFIED: ci.yml:59-63]
- **Adding buf/pflag/testify/go-systemd "because standard"** — each violates the repo's minimal-deps or zero-test-deps conventions; AGENTS.md and project test style (vanilla testing, hand-written fakes) take precedence over generic recommendations.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| gRPC server, HTTP/2, flow control | custom RPC/TCP | grpc-go | Protocol correctness, streaming, keepalive — the one sanctioned dependency |
| Health checking protocol | custom health service | `google.golang.org/grpc/health` + `grpc_health_v1` | In-module (no new dep); standard clients (grpcurl, k8s, Envoy) speak it; API-06 mandates the standard endpoint |
| In-memory gRPC test transport | real sockets in unit tests | `google.golang.org/grpc/test/bufconn` | In-module; fast, hermetic; the probe verified the NewClient+WithContextDialer pattern |
| Stub codegen orchestration | build scripts/cgo codegen | protoc + `go build`-pinned plugins via Makefile | The 04-01 precedent, extended with one plugin |
| Version/commit metadata | runtime shell-outs | ldflags `-X` | D-18 pins it; standard, zero-cost |
| sd_notify readiness | hand-rolled NOTIFY_SOCKET writer | defer to v2 / coreos/go-systemd | Not needed for Type=simple v1; a hand-rolled socket writer is untestable surface for zero v1 value |
| Watch change detection | diff-polling `Records()` | the storage tap (Pattern 1) | Polling can't distinguish loser-absorption (D-06), costs O(n) per poll, and misses the install-order guarantee |

**Key insight:** every "clever" piece of this phase (LWW merge, HLC versions, ack tracking, dump confirmation) already exists in-repo — the phase's entire risk lives in *seams and wiring*, which is exactly where the four additive changes and the shutdown order carry the weight.

## Common Pitfalls

### Pitfall 1: GracefulStop hangs on open Watch streams
**What goes wrong:** shutdown never completes; systemd SIGKILLs after timeout; D-19's final dump never runs.
**Why:** `GracefulStop` waits for *all* in-flight RPCs; server-streaming Watch RPCs never end on their own.
**How to avoid:** end streams via a shutdown context (Watch handlers select on it), then GracefulStop-with-timeout → Stop fallback (Pattern 6).
**Warning signs:** integration test hangs at shutdown; systemd stop timing out.

### Pitfall 2: `mustEmbedUnimplementedSupKVServer` build break
**What goes wrong:** the service struct doesn't embed `UnimplementedSupKVServer` → compile error at `RegisterSupKVServer`.
**Why:** protoc-gen-go-grpc v1.6.2 generates the forward-compat requirement (verified in the generated interface: `mustEmbedUnimplementedSupKVServer()`).
**How to avoid:** embed `apiproto.UnimplementedSupKVServer` in the server struct (standard practice; also gives automatic Unimplemented for future RPCs).

### Pitfall 3: wait-set eviction misread as Acks
**What goes wrong:** `wait_for_replication` reports `acked=true` for a peer that went silent and was evicted.
**Why:** `evictPeers` removes evicted peers from every pending wait-set [VERIFIED: core.go:709-722]; `WaitingOn` shrink alone is ambiguous with handleAck.
**How to avoid:** the `Acked` seam (Pattern 1.3); report counts from `Acked`, total fixed at write time (D-10).
**Warning signs:** a wait test with an expiring peer reporting success.

### Pitfall 4: false "acked" from the notify-queue race
**What goes wrong:** the wait poll runs before the event loop drains `NotifyLocalWrite`; the (key, version) is absent from pending ⇒ misread as drained.
**Why:** notification is asynchronous through a 1024-deep queue [VERIFIED: repl.go:226-251].
**How to avoid:** wait loop phase 1 = wait-for-appearance (with a small bound), phase 2 = wait-for-drain.

### Pitfall 5: Watch boundary duplicates/misses
**What goes wrong:** subscriber sees a change twice (snapshot + event) or misses one that happened during subscribe.
**Why:** snapshot copy and hub registration are not atomic with installs unless the boundary algorithm handles it.
**How to avoid:** register-then-snapshot on the single hub goroutine + the version-dominance suppression rule (Pattern 2).
**Warning signs:** flaky watch tests under `-race` with concurrent writers.

### Pitfall 6: CI tidy gate red
**What goes wrong:** PR fails at `git diff --exit-code go.mod go.sum`.
**Why:** dependency added without tidy-exact go.mod/go.sum (the 01-02 precedent — 54 tidy-required entries once missing).
**How to avoid:** run `go mod tidy` with GOTOOLCHAIN matching go.mod before every commit that touches deps; commit both files atomically with the code.

### Pitfall 7: gRPC default message limits vs `value_max_bytes` and List size
**What goes wrong:** Set of a 60 KiB value works, but a List response over tens of thousands of keys approaches the 4 MiB default send limit.
**Why:** grpc defaults (4 MiB recv/send) are fine for values (≤ 60 KiB + overhead) but List is unbounded by design (D-12).
**How to avoid:** keep defaults for now (envelope math: 10k keys × ~150 B ≈ 1.5 MiB — inside 4 MiB), note `grpc.MaxSendMsgSize` as the documented knob if the envelope grows; do not silently hand-roll pagination (D-12 forbids).

### Pitfall 8: engine write semantics mis-mapped to codes
**What goes wrong:** `ErrDumpFailed` mapped to something other than Unavailable, or the write reported failed when it actually persisted in memory.
**Why:** `Engine.Set` returns ErrDumpFailed while the write *stays in memory and replicates* (PERS-04).
**How to avoid:** map exactly per the sentinel docs — the errors files already name the mapping (see Code Examples).

## Code Examples

Verified against this session's scratch probe (grpc v1.85.0, go 1.27) and the repo's actual source.

### Error mapping (from the repo's own sentinel docs)
```go
// The mapping is already pinned by the in-repo docs [VERIFIED: storage/errors.go:7-19, persist/errors.go:6-18]:
//   storage.ErrNotFound       → codes.NotFound         ("key not found" — missing OR tombstoned)
//   storage.ErrAlreadyExists  → codes.AlreadyExists    ("key already exists" — any record incl. tombstone)
//   storage.ErrValueTooLarge  → codes.InvalidArgument  ("value exceeds the maximum allowed size")
//   persist.ErrDumpFailed     → codes.Unavailable      ("dump failed; write not confirmed")
//   persist.ErrClosed         → codes.Unavailable      ("persist engine closed")
//   udprepl.ErrRunLoopExited  → codes.Unavailable      ("udprepl: replication event loop has exited")
//   gate closed (API-04)      → codes.Unavailable
//   persistfile.ErrInvalidSnapshotName → codes.InvalidArgument ("invalid snapshot name")
// Match with errors.Is, never ==.
```

### Health server wiring + gate (probe-verified API shape)
```go
// Source: scratch probe this session (compiles + runs on grpc v1.85.0) + grpc.io health guide
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

hs := health.NewServer()
hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING) // "" = whole server
srv := grpc.NewServer()
healthpb.RegisterHealthServer(srv, hs)
// gate resolution goroutine:
//   <-syncArrivedAtomic || <-time.After(startupGate):
//       gateOpen.Store(true)
//       hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
```

### bufconn test harness (probe-verified pattern)
```go
// Source: scratch probe this session. bufconn is in the grpc module — no new dependency.
import "google.golang.org/grpc/test/bufconn"

lis := bufconn.Listen(1 << 20)
go func() { _ = srv.Serve(lis) }()
conn, err := grpc.NewClient("passthrough:bufnet",
    grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
// health check over bufconn:
resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
```
Note: modern grpc-go clients refuse to connect without explicit credentials even on bufconn — `insecure.NewCredentials()` is mandatory (the probe panicked without it). For cross-package tests, run a real `net.Listen("tcp", "127.0.0.1:0")` server instead — the exported-test seam pattern (repl_internal_test.go precedent) if needed.

### Makefile proto target extension (following the verified 04-01 pattern)
```makefile
# Source: adapted from Makefile:26-29 (existing protoc-gen-go pattern), probe-verified plugin build
proto:
	go build -o .tmp/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	go build -o .tmp/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc
	protoc --plugin=.tmp/protoc-gen-go \
	       --go_out=. --go_opt=paths=source_relative \
	       --plugin=.tmp/protoc-gen-go-grpc \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       internal/grpcapi/apiproto/supkvapi.proto
```

### Storage tap seam (additive — the repl.Config.Log precedent)
```go
// Precedent: repl.Config.Log func(msg string) [VERIFIED: core.go:95]; config-func, not interface
type Config struct {
    ValueMaxBytes int
    // OnChange, when non-nil, is called from install under the store write
    // lock with the record that just became the stored record. It must not
    // block; it observes commit order for every change source.
    OnChange func(key string, rec Record)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.Dial` + `WithBlock` | `grpc.NewClient` (lazy, name-resolution based) | grpc-go v1.63+ | The probe verified NewClient is the current API; `Dial` is deprecated — use NewClient even in tests |
| `github.com/golang/protobuf` shim | `google.golang.org/protobuf` | shim frozen years ago | Gone from the v1.85.0 graph entirely; also denied by the lint config |
| protoc-gen-go-grpc v1.x "require_unimplemented" opt-out | always-on embed pattern | v1.5+ | Server must embed `UnimplementedSupKVServer` (Pitfall 2) |
| hand-rolled health endpoints | standard `grpc_health_v1` | long-standing | Standard clients interoperate; required by API-06 |
| `time.Timer` drain selects | Go 1.23+ unbuffered-reset semantics | Go 1.23 | Repo already pins this discipline (engine/udprepl timerChan/stopTimer precedents) — reuse the pattern |

**Deprecated/outdated:** `grpc.Dial` (use `grpc.NewClient`); `github.com/golang/protobuf` (lint-denied); `WaitForReady`-less blocking dials in tests.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | "First full sync round arrives" = first received DataSync datagram (any chunk), not a completed multi-chunk round | Patterns 1.2 / gate | A stricter reading needs chunk-completeness state the protocol lacks; if the user wants stricter semantics, a small core counter redesign is needed (low risk — the 1s timeout bounds either way) |
| A2 | `wait_timeout` default 1s (new knob; IDEA.md §Parameters lists no `wait_timeout`) | Pattern 5 / D-09 | Only the default value is at stake (operator-tunable, per-request overridable); 200ms ack_timeout/5s LAN norms suggest 1s is comfortable |
| A3 | Slow-watch policy = disconnect the stream (etcd-style), not drop events | Pattern 2 | CONTEXT says "drop policy consistent with 'never block the store'" — disconnect IS a drop policy (drop the subscriber, not events) and preserves D-02 no-holes; if the user prefers silent event drops, D-02 would be violated — needs explicit override |
| A4 | grpc v1.86.0 exists and is stable (listed on proxy; fetch failed this session on network flakiness) | Standard Stack | None if pinned to v1.85.0; only matters if the planner insists on latest — verify at implementation time |
| A5 | Gate applies to writes only; SnapshotCreate/Load counted as writes (gated) — restore replays writes | Pattern 3 | If snapshots should bypass the gate, API-04's "writes rejected" wording is the tie-breaker; planner should pin this |
| A6 | `List` of ~10k keys fits the 4 MiB default (≈1.5 MiB at ~150 B/key) | Pitfall 7 | If key names are much longer, MaxSendMsgSize must be raised — one-line knob, documented |
| A7 | Empty key is rejected with InvalidArgument at the API layer (store itself permits "") | Security V5 | Cosmetic contract choice; flag for the planner to pin |
| A8 | A gate-resolution polling goroutine (or sync-arrived atomic wait) is acceptable vs purely event-driven wiring | Pattern 3 | Negligible — worst case one extra goroutine woken once per process start |

## Open Questions

1. **gRPC listen address naming** — IDEA.md's `bind_addr` is the UDP bind; gRPC needs its own (e.g. `grpc_addr`, default `:7901`?). What we know: IDEA §Parameters lists `bind_addr` under UDP semantics; unclear: the gRPC port default. Recommendation: `SUPKV_GRPC_ADDR` default `127.0.0.1:7901`... planner picks; the trusted-LAN trust model makes any default defensible (document it).
2. **Supersession-during-wait reporting** — if a greater remote version supersedes the tracked entry mid-wait, is the report `acked=false` (the version itself wasn't acked) with counts frozen at last poll? What we know: the core treats supersession as "promise fulfilled by winning" for *tracking*; unclear: D-08's aggregate intent. Recommendation: `acked=false` (honest: Acks did not arrive), counts from the last mirrored snapshot; pin in the plan.
3. **Watch snapshot includes tombstoned keys as `deleted=true` events** — Records() includes tombstones and D-02/D-3 deliver them as ordinary events; confirm the client contract reading (List excludes them, D-14 — the asymmetry is deliberate). Recommendation: as stated; pin in plan.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | go1.27.3 linux/amd64 (repo pins 1.27.1) | — |
| protoc | `make proto` (dev machines only; CI never regenerates) | ✓ | libprotoc 3.21.12 (Makefile expects 3.21.x) | committed generated code — regeneration optional |
| proxy.golang.org | `go get`/`tidy` (once, at implementation) | ✓ flaky | TLS handshake timeouts observed this session (retries succeeded) | retry; or vendor from a warm cache; CI runners are unaffected |
| grpc-go v1.85.0 | runtime | ✓ | probe-downloaded, built, run | — |
| protoc-gen-go-grpc v1.6.2 | codegen tool | ✓ | probe-built | — |
| systemd | deployment only (docs) | n/a | — | Type=simple works everywhere, containers included |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** network flakiness to the Go proxy is transient — the dependency set is fully cached once fetched.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vanilla `testing` + `go test -race` (repo convention; no testify — zero test deps) |
| Config file | none — Makefile targets (`make test` = `go test -race -cover ./...`) [VERIFIED: Makefile:10-12] |
| Quick run command | `go test -race ./internal/grpcapi/... ./internal/app/...` |
| Full suite command | `make test` && `make lint` (CI parity: vet + race + tidy gate) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| API-01 | Get/Set/SetIfAbsent/Delete + full error set incl. dump-failure Unavailable (fake DumpStore injecting failure) | integration (bufconn) | `go test -race ./internal/grpcapi/ -run TestServer_` | ❌ Wave 0 |
| API-02 | List prefix sorted, tombstones excluded, empty prefix = all | integration (bufconn) | `go test -race ./internal/grpcapi/ -run TestList` | ❌ Wave 0 |
| API-03 | Watch snapshot-then-live exact cut (concurrent writers under -race), tombstone events, replicated deletes via `Engine.ApplyRemote`, prefix filter, slow consumer → stream closed, store never blocked | integration (bufconn) | `go test -race ./internal/grpcapi/ -run TestWatch` | ❌ Wave 0 |
| API-04 | Writes Unavailable until sync-arrived (drive `repl.Core.HandleDatagram` with a port DataSync / flip mirrored atomic) and until `startup_gate` elapses; reads OK throughout | integration (bufconn) | `go test -race ./internal/grpcapi/ -run TestGate` | ❌ Wave 0 |
| API-05 | zero peers ⇒ immediate acked (D-11); acks arrive ⇒ acked=true; timeout ⇒ acked=false; eviction ≠ ack (A: Acked seam); write outcome unchanged; notify-queue appearance wait | integration (bufconn + driven core) | `go test -race ./internal/grpcapi/ -run TestWait` | ❌ Wave 0 |
| API-06 | health `""` NOT_SERVING before gate, SERVING after; grpc_health_v1 client over bufconn | integration (bufconn) | `go test -race ./internal/grpcapi/ -run TestHealth` | ❌ Wave 0 |
| API-07 | SnapshotCreate/Load through RPC; invalid names → InvalidArgument (persistfile validation already unit-tested — adapter mapping test only) | integration (bufconn) | `go test -race ./internal/grpcapi/ -run TestSnapshot` | ❌ Wave 0 |
| TOOL-01 | `--version` output; flag>env>default table; binary smoke: build cmd/sup-kv, exec on ephemeral ports + tmp dump dir, health over real TCP, SIGTERM ⇒ clean exit + final dump file exists | unit + binary smoke (real socket, the udprepl D-08 smoke precedent) | `go test -race ./cmd/sup-kv/...` | ❌ Wave 0 |
| seams | storage tap fires on install only (no event on loser/equal merge); PendingInfo.Acked accuracy; Core sync-arrived accessor; Engine.Flush semantics | unit (in-package, storage/repl/persist test styles) | `go test -race ./internal/app/...` | ✅ existing files extend |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/grpcapi/... ./internal/app/...` + `make lint` (fast gate)
- **Per wave merge:** `make test` (full suite, race + cover)
- **Phase gate:** full suite green + `make lint` + binary smoke + `go mod tidy` clean diff before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/grpcapi/apiproto/` — proto + generated code + Makefile target + go.mod deps (the whole gRPC surface starts here)
- [ ] `internal/grpcapi/server_test.go` — bufconn harness fixture (server-on-bufconn builder shared by all API tests)
- [ ] `cmd/sup-kv/` smoke test — binary exec + real TCP + SIGTERM lifecycle
- [ ] The four additive seams' tests extend existing `internal/app` test files (no new framework needed)

## Security Domain

**Level:** ASVS L1 (`security_asvs_level: 1`, `security_block_on: high`). Trust model: the replication segment AND the API are on a trusted LAN in v1 — no auth/encryption is a documented accepted limitation (IDEA.md §Trust model; DOCS-02 phases the prominent restatement). Nothing in this phase introduces auth; DO NOT add any.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | v1 accepted limitation (trusted segment); documented future work is replication HMAC (SEC-01 v2), API auth unspecified |
| V3 Session Management | no | no sessions; gRPC connection lifecycle owned by grpc-go |
| V4 Access Control | no | single-tenant trusted surface; API exposes everything the local operator has |
| V5 Input Validation | yes | value size via `value_max_bytes` (existing, InvalidArgument); snapshot names validated at persistfile — charset `[A-Za-z0-9._-]`, length 1..64, `.`/`..` rejected, validation before any filesystem touch [VERIFIED: snapshot.go:78-106 — verbatim: `if name == "" || len(name) > maxSnapshotName || name == "." || name == ".."` and charset `c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9'`, `c == '.', c == '_', c == '-'`]; recommend rejecting empty keys (A7); grpc's 4 MiB recv cap bounds request size |
| V6 Cryptography | no | CRC32 is corruption detection, never tamper protection (documented); no secrets handled in this phase |
| V12 Logging (partial) | informally | structured logging of discards exists in repl; OBS-03 is Phase 6 — do not expand scope |

### Known Threat Patterns for gRPC-on-trusted-LAN (Go)

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via snapshot names | Tampering/Elevation | Already closed at persistfile (V5 above); adapter maps ErrInvalidSnapshotName → InvalidArgument; do NOT bypass validation anywhere |
| Slow-reader resource exhaustion on Watch | DoS | per-subscriber bounded buffers + disconnect-on-full (Pattern 2); hub ingress bounded; store never blocked |
| Oversized/overspeed request flooding | DoS | grpc default 4 MiB recv cap + `value_max_bytes` InvalidArgument; no rate limiting in v1 (trusted segment, documented) |
| Unauthenticated RPC + replication | Spoofing | Accepted v1 limitation (IDEA §Trust model); docs task (DOCS-02) states it prominently; SEC-01 (HMAC) is v2 |
| Crash-on-malformed input | DoS | grpc-go handles wire-level framing; domain validation maps to InvalidArgument; handler panics would crash the process — keep handlers total over any input (the `applyRemote` "never panic on hostile input" precedent [VERIFIED: core.go:441-448]) |

## Sources

### Primary (HIGH confidence)
- Codebase (read in full this session): `internal/app/persist/engine.go` (573 lines), `internal/app/repl/core.go` (912 lines), `internal/app/storage/store.go`, `internal/port/{dump,repl}.go`, `internal/udprepl/repl.go`, `internal/persistfile/snapshot.go`, `cmd/sup-kv/main.go`, `Makefile`, `go.mod`, `.golangci.yml` (depguard + exclusions), `.github/workflows/ci.yml`, `docs/IDEA.md` (§API, §Node lifecycle, §Parameters, §Trust model, §Product principles), error sentinel files
- Scratch-module probes (this session, /tmp): grpc v1.85.0 download/build/run — server + health (`""` status transitions) + bufconn + NewClient + insecure creds; minimal tidy graph (4 indirects, protobuf resolves to v1.36.12); grpc v1.85.0 `go 1.26.0` directive; protoc-gen-go-grpc v1.6.2 build + protoc 3.21.12 generation + compile against v1.85.0 (`SupportPackageIsVersion9`, `mustEmbedUnimplementedSupKVServer`)

### Secondary (MEDIUM confidence)
- [grpc.io health checking guide](https://grpc.io/docs/guides/health-checking/) — "" = whole server; SERVING/NOT_SERVING; Check vs Watch
- [grpc.io graceful shutdown guide](https://grpc.io/docs/guides/server-graceful-stop), [grpc-go issue #8776](https://github.com/grpc/grpc-go/issues/8776), [SO 58480775](https://stackoverflow.com/questions/58480775/how-to-close-all-grpc-server-streams-using-gracefulstop) — GracefulStop blocks on open streams; NOT_SERVING + timeout + Stop pattern
- [freedesktop.org systemd.service](https://www.freedesktop.org/software/systemd/man/systemd.service.html), [coreos/go-systemd](https://github.com/coreos/go-systemd), [containerd issue #5142](https://github.com/coreos/containerd/issues/5142) — Type=notify vs simple tradeoffs
- go-ultimate skill v0.15.0 — service/testing/config conventions (project AGENTS.md takes precedence on conflicts)

### Tertiary (LOW confidence)
- None material; every external claim used for planning is probe-verified or official-doc cited

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version and the minimal dependency graph empirically built and run this session
- Architecture: HIGH — all seams verified against actual source; watch-cut algorithm derived from verified Merge/install laws
- Pitfalls: HIGH for GracefulStop/health/toolchain (probe + official docs); MEDIUM for policy interpretations (A1–A8) — all flagged in the Assumptions Log

**Research date:** 2026-11-08
**Valid until:** 2027-01-08 (Go ecosystem moves slower than 30 days for pinned versions; grpc v1.86+ may supersede the pin — the CI tidy gate and Dependabot govern drift)

## RESEARCH COMPLETE

**Phase:** 5 - gRPC API and Binary
**Confidence:** HIGH

### Key Findings
- Four additive seams are required inside `internal/app` (storage OnChange tap in `install`, repl sync-arrived state, `PendingInfo.Acked`, exported `Engine.Flush`) — each mapped to a requirement nothing else can satisfy; everything else crosses via existing exported methods and the udprepl mirror pattern.
- grpc-go v1.85.0 probe-verified end-to-end under go 1.27 with the machine's protoc 3.21.12 and protoc-gen-go-grpc v1.6.2; minimal graph = 4 indirects, protobuf stays at the existing v1.36.12; v1.86.0 unfetchable this session — pin v1.85.0.
- Generated code must live outside `internal/app`/`internal/port` (depguard); recommended layout `internal/grpcapi` + `internal/grpcapi/apiproto`, committed generated output, Makefile `proto` target extended (protoc, not buf).
- Watch correctness hinges on: tap at `storage.install` (D-06 exact), hub-goroutine cut with version-dominance boundary suppression (no dupes/misses, no lock-order deadlock), disconnect-on-full slow-consumer policy (D-02 no-holes).
- Shutdown (D-19) is a strict order — health NOT_SERVING → end Watch streams → GracefulStop+timeout→Stop → stop repl → `Engine.Flush()` final confirmed dump → cancel engine; GracefulStop alone hangs on open streams.

### File Created
`.planning/phases/05-grpc-api-and-binary/05-RESEARCH.md`

### Confidence Assessment
| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | Empirical probe: download, build, generate, run |
| Architecture | HIGH | Seams verified line-level against engine/core/store/adapter source |
| Pitfalls | HIGH | Probe + official docs for grpc behavior; interpretations flagged A1–A8 |

### Open Questions
1. gRPC listen-address naming/default (IDEA's `bind_addr` is UDP). 2. Supersession-during-wait report semantics. 3. Watch-snapshot tombstone visibility (recommended: visible as deleted=true events). All carry recommendations; planner pins them.

### Ready for Planning
Research complete. Planner can now create PLAN.md files.
