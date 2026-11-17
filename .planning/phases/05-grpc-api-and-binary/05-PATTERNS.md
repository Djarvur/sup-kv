# Phase 5: gRPC API and Binary - Pattern Map

**Mapped:** 2026-11-08
**Files analyzed:** 16 (10 new, 6 modified)
**Analogs found:** 13 / 16 (3 rely on RESEARCH.md probe-verified patterns: main() orchestration, gate/health wiring, bufconn harness)

All analog paths verified git-tracked (`git ls-files`). Every analog is real source in this repo, not a generated mirror.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/grpcapi/apiproto/supkvapi.proto` (new) | contract (wire schema) | request-response + streaming | `internal/udprepl/replproto/supkv.proto` | exact |
| `internal/grpcapi/apiproto/supkvapi.pb.go` + `supkvapi_grpc.pb.go` (new, generated, committed) | generated code | — | `internal/udprepl/replproto/supkv.pb.go` (committed output) | exact |
| `internal/grpcapi/server.go` (new) | controller (API adapter) | request-response | `internal/udprepl/repl.go` (adapter role + struct↔proto mapping) | role-match |
| `internal/grpcapi/watch.go` (new) | service (hub goroutine) | pub-sub / streaming | `internal/udprepl/repl.go` Run loop (select + bounded queues) + `internal/app/storage/store.go` install (tap point) | role-match |
| `internal/grpcapi/gate.go` (new) | middleware (gate resolver + health) | event-driven | `internal/udprepl/repl.go` atomic-mirror discipline | partial (health wiring itself: no analog, use RESEARCH) |
| `internal/grpcapi/wait.go` (new, may fold into server.go) | service (poll loop) | polling | `internal/udprepl/smoke_test.go` waitFor + `internal/app/repl/core.go` PendingSnapshot | role-match |
| `internal/grpcapi/server_test.go` (new) | test (integration) | request-response + streaming | `internal/udprepl/repl_test.go` (external `_test` package) | role-match (bufconn transport: RESEARCH probe snippet) |
| `cmd/sup-kv/main_test.go` smoke (new) | test (binary lifecycle) | batch | `internal/udprepl/smoke_test.go` (real-socket smoke) | role-match |
| `cmd/sup-kv/main.go` (modified: filled from skeleton) | config/entrypoint | lifecycle | none in repo (skeleton exists; RESEARCH Patterns 5/6) | none |
| `internal/app/storage/store.go` (modified, additive) | model core (change tap) | event tap | `internal/app/repl/core.go` Config.Log (config-func precedent) + install in same file | exact |
| `internal/app/repl/core.go` (modified, additive) | model core (sync-arrived + Acked) | state | same file: handleDataSync / handleAck / PendingInfo | exact |
| `internal/app/persist/engine.go` (modified, additive) | model core (Flush export) | — | same file: unexported `flush()` | exact |
| `internal/udprepl/repl.go` (modified, additive) | adapter (atomic mirrors) | — | same file: mirrorStats + atomic fields | exact |
| `Makefile` (modified) | build config | — | same file: `proto` target | exact |
| `go.mod` / `go.sum` (modified) | config | — | same file: tool block + direct require | exact |
| `internal/port/grpcapi.go` (CONDITIONAL — only if planner decides a decoded-structs seam is needed) | port seam | request-response | `internal/port/repl.go` | exact (if needed) |

Note on the conditional port seam: RESEARCH's recommended structure omits it — `internal/grpcapi` is outside `internal/app`/`internal/port`, so it may import `internal/app/persist`, `internal/app/repl` types and `internal/udprepl` directly (exactly as `internal/udprepl/repl.go` imports `internal/app/storage`). No protobuf needs to cross into `internal/app` in this phase, so the depguard pressure that forced `port/repl.go` does not apply. The CONTEXT discretion item "follow the phase-4 D-01 decoded-structs port-seam precedent" is about the adapter's *mechanical, decision-free* style, not a mandated new port file.

## Pattern Assignments

### `internal/grpcapi/apiproto/supkvapi.proto` (contract, request-response + streaming)

**Analog:** `internal/udprepl/replproto/supkv.proto` (whole file, 74 lines)

**Header + package pattern** (lines 1-11):
```protobuf
// sup-kv replication wire schema: ... (design-doc section citation, scope note)
syntax = "proto3";

package supkv.v1;

option go_package = "github.com/Djarvur/sup-kv/internal/udprepl/replproto;replproto";
```
New file: `option go_package = "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto;apiproto";` and a package name like `supkvapi.v1`. Keep the header-comment style: cite `docs/IDEA.md §API`, name the decision IDs (D-01..D-15) on the messages they pin.

**Message style** (lines 44-50 — full record shape to mirror in the Watch event, D-01):
```protobuf
message DataReplication {
  string key = 1;
  bytes version = 2;
  bytes data = 3;
  bool deleted = 4;
  uint32 checksum = 5;
}
```
The Watch event carries the same field set (key, version as 16 bytes, value, deleted). Request/response messages for Get/Set/Delete/List/Snapshot follow the same flat, numbered-from-1 style with per-field decision-ID comments.

---

### `internal/grpcapi/apiproto/*.pb.go` (generated, committed)

**Analog:** `internal/udprepl/replproto/supkv.pb.go` — committed generated output; lint-exempt automatically via the `\.[\w-]+\.go$` path pattern in `.golangci.yml` lines 491-492 (linters) and 520-521 (formatters). No lint-config change needed. Regeneration is dev-machine-only; CI never regenerates (Makefile:22-25 comment).

---

### `Makefile` (modified: proto target extension + ldflags build)

**Analog:** same file, `proto` target (lines 22-29):
```makefile
proto:
	go build -o .tmp/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	protoc --plugin=.tmp/protoc-gen-go \
	       --go_out=. --go_opt=paths=source_relative internal/udprepl/replproto/supkv.proto
```
Extend with a second plugin built the same way (`go build -o .tmp/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc`) and `--go-grpc_out=. --go-grpc_opt=paths=source_relative` over `internal/grpcapi/apiproto/supkvapi.proto` — exactly the RESEARCH "Makefile proto target extension" snippet, which is itself adapted from these lines. `build` target (lines 19-20) gains `CGO_ENABLED=0` and `-ldflags "-X main.version=… -X main.commit=…"` (D-18).

---

### `go.mod` / `go.sum` (modified)

**Analog:** same file, tool block (lines 5-11):
```go
tool (
	github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	golang.org/x/vuln/cmd/govulncheck
	google.golang.org/protobuf/cmd/protoc-gen-go
)

require google.golang.org/protobuf v1.36.12
```
Delta: add `google.golang.org/grpc/cmd/protoc-gen-go-grpc` to the tool block; `google.golang.org/grpc v1.85.0` joins as the second direct runtime require. Must be `go mod tidy`-exact (CI gate: `go vet ./... && go test -race ./... && go mod tidy && git diff --exit-code go.mod go.sum`, `.github/workflows/ci.yml:59-63`).

---

### `internal/grpcapi/server.go` (controller/API adapter, request-response)

**Analog:** `internal/udprepl/repl.go` — the phase-4 adapter is the repo's one precedent for "the only protobuf-importing translation layer, mechanical, owns no decision".

**Package-doc pattern** (lines 1-12) — state the adapter role, the depguard rationale, and what the package deliberately does NOT decide:
```go
// Package udprepl is the sup-kv UDP replication adapter: the only net- and
// protobuf-importing replication code, mechanical per D-01/D-02 — ...
// It owns no protocol decision: discards, the peer table and every timer
// live in the core.
package udprepl
```

**Imports pattern** (lines 14-32) — gci sections: standard / default (google.golang.org) / localmodule, per `.golangci.yml:500-505`:
```go
import (
	"context"
	...

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	...
)
```
New file swaps `protobuf/proto` for `google.golang.org/grpc`, `grpc/codes`, `grpc/status`, and `internal/grpcapi/apiproto`; localmodule imports are `internal/app/persist`, `internal/app/storage`, `internal/app/repl` (types only), `internal/persistfile` (sentinel), `internal/udprepl`.

**Struct↔proto mapping pattern** (`protoBody`, lines 437-462, excerpt) — the mechanical field-by-field translation the RPC handlers mirror in reverse (domain struct → proto message and back; version travels as `ver[:]` / `recordVersion(b)` 16-byte slices):
```go
case port.DataReplication:
	return &replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
		Key:      m.Key,
		Version:  m.Record.Version[:],
		Data:     m.Record.Data,
		Deleted:  m.Record.Deleted,
		Checksum: m.Record.Checksum,
	}}}, nil
```
The decode direction (`portMessage`, lines 467-508) shows the inverse with domain validation (`recordVersion`, lines 513-520: reject any version not exactly 16 bytes). The server struct must embed `apiproto.UnimplementedSupKVServer` (protoc-gen-go-grpc v1.6.2 requirement, RESEARCH Pitfall 2).

**Engine surface the handlers call** — `internal/app/persist/engine.go`: `Set`/`Delete`/`SetIfAbsent` (lines 348-378, block until dump-confirmed), `Get` (439-441), `Records` (445-447, tombstones included — the List source, D-14 filtering is adapter-side), `Snapshot`/`Restore` (460-499). After a successful write: `udprepl.Adapter.NotifyLocalWrite` (`internal/udprepl/repl.go:226-251`).

**Error mapping** — see Shared Patterns (sentinels → codes).

---

### `internal/grpcapi/watch.go` (service/hub, pub-sub)

**Analog A — hub goroutine lifecycle:** `internal/udprepl/repl.go` Run loop (lines 188-215):
```go
func (a *Adapter) Run(ctx context.Context) error {
	go a.readLoop()
	defer func() { _ = a.conn.Close() }()
	defer close(a.done)
	var out port.Outbox
	next := a.core.Advance(time.Now(), &out)
	a.flush(&out)
	var timer *time.Timer
	for {
		timer = armTimer(timer, next)
		select {
		case <-ctx.Done():
			stopTimer(timer)
			return ctx.Err()
		case ev := <-a.inbound:
			...
		case lw := <-a.notify:
			...
		}
		...
	}
}
```
One goroutine owns all subscriber state; subscribe requests and tap events arrive via channels; ctx cancellation ends the loop. Bounded-queue constants pattern (lines 42-61: `inboundQueue = 64`, `notifyQueue = 1024` with load-bearing comments) is the precedent for the per-subscriber buffer size and hub ingress bound. Non-blocking send shape: `select { case ch <- x: default: }` (engine's changed-nudge, engine.go:403-406).

**Analog B — the tap point:** `internal/app/storage/store.go` `install` (lines 228-238) — the single convergence point of every state change (local `write`, remote `join`, restore replay):
```go
func (s *Store) install(key string, rec Record) {
	data := bytes.Clone(rec.Data)
	s.records[key] = Record{
		Version:  rec.Version,
		Data:     data,
		Deleted:  rec.Deleted,
		Checksum: RecordChecksum(key, rec.Version, data, rec.Deleted),
	}
}
```
The strictly-greater install law (`join`, lines 210-226) is what makes "event = install" satisfy D-06 (loser merges never reach install). `Records()` (lines 243-256) is the Watch snapshot source (tombstones included — D-02/D-03).

---

### `internal/app/storage/store.go` modification (OnChange tap in Config)

**Analog:** `internal/app/repl/core.go` Config.Log — the config-func, not interface, callback precedent (lines 86-96 declaration; 125-127 nil-normalization):
```go
type Config struct {
	...
	Log func(msg string)
}
...
	if c.Log == nil {
		c.Log = func(_ string) {}
	}
```
Add `OnChange func(key string, rec Record)` to `storage.Config` (lines 25-27) with the same nil→no-op normalization in `Config.normalized()` (lines 82-87), and fire it from `install()` under the write lock with the *stored* record (post `RecordChecksum` recompute). Doc-comment the constraint: runs under the store write lock, must not block (fan-out is a non-blocking channel send). RESEARCH "Storage tap seam" snippet is the exact target shape.

---

### `internal/grpcapi/gate.go` (middleware, event-driven)

**Analog:** `internal/udprepl/repl.go` atomic-mirror discipline — cross-goroutine readers never touch single-goroutine state.

Atomic fields (lines 108-113):
```go
	badCRC          atomic.Uint64
	...
	peerCount       atomic.Uint64
```
Copying accessors (lines 265-269):
```go
func (a *Adapter) KnownPeers() uint64 {
	return a.peerCount.Load()
}
```
Mirror-after-every-iteration (`mirrorStats`, lines 328-336):
```go
func (a *Adapter) mirrorStats() {
	d := a.core.Discards()
	...
	a.peerCount.Store(uint64(len(a.core.PeerSnapshot())))
}
```
The gate resolver consumes a NEW `syncArrived atomic.Bool` mirrored the same way (see udprepl modification below). Health wiring (`health.NewServer()`, `SetServingStatus("", …)`) has no in-repo analog — use the probe-verified snippet in RESEARCH Code Examples "Health server wiring + gate". `""` = whole-server status.

---

### `internal/grpcapi/wait.go` (service, polling)

**Analog A — poll loop:** `internal/udprepl/smoke_test.go` `waitFor` (lines 175-190):
```go
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	...
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(pollEvery)
	}
	t.Fatalf("timed out after %v waiting for %s", timeout, what)
}
```
Same shape, production-side: poll the mirrored pending snapshot until entry drained / timeout (D-09), with the wait-for-appearance phase first (RESEARCH Pitfall 4 — `NotifyLocalWrite` is asynchronous through the 1024-deep queue, `internal/udprepl/repl.go:226-251`).

**Analog B — the read seam:** `internal/app/repl/core.go` `PendingSnapshot`/`PendingInfo` (lines 852-876):
```go
type PendingInfo struct {
	Version   version.RecordVersion
	WaitingOn []uint16
	NextRetry time.Time
}
```
Sorted-keys copy discipline (`slices.Sorted(maps.Keys(...))` — "no map-iteration order reaches a read path"). The new `Acked []uint16` field joins here, sorted the same way.

---

### `internal/app/repl/core.go` modification (sync-arrived + PendingInfo.Acked)

**Analog:** same file — both changes land on existing handlers/accessors.

Sync-arrived: set an internal flag at the top of `handleDataSync` (lines 513-514) and expose a copying accessor in the style of `Discards()` (lines 834-838) / `PeerSnapshot()` (lines 844-850). Note the A1 interpretation: first *received* DataSync datagram (any chunk) — chunks carry no continuation state (lines 661-671 doc).

Acked: `handleAck` (lines 469-478) currently only shrinks the wait-set:
```go
func (c *Core) handleAck(peerID uint16, ack port.Ack) {
	entry, ok := c.pending[ack.Key]
	if !ok || entry.version != ack.Version {
		return
	}
	delete(entry.waitSet, peerID)
	if len(entry.waitSet) == 0 && len(c.peers) > 0 {
		delete(c.pending, ack.Key)
	}
}
```
…which is ambiguous with `evictPeers` (lines 716-720: `delete(c.pending[key].waitSet, id)` with no Ack). Add `Acked []uint16` (sorted append) to `pendingEntry` (lines 211-218) and surface it in `PendingInfo`. No callbacks, no channels — "results, not callbacks" (Outbox doc, `internal/port/repl.go:114-118`).

---

### `internal/app/persist/engine.go` modification (exported Flush)

**Analog:** same file — export the existing `flush()` (lines 535-547 core excerpt):
```go
func (e *Engine) flush() error {
	reply := make(chan error, 1)
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return ErrClosed
	}
	select {
	case e.flushCh <- reply:
	case <-e.runDone:
		return ErrClosed
	}
	...
}
```
`Flush()` = exported wrapper (or rename) with a doc comment citing D-19 (shutdown calls it BEFORE cancelling the engine context; `Run`'s exit sweep, lines 230-250, resolves waiters with `ErrClosed` *without* a final dump — that gap is why the export is needed). Existing callers of `flush()` (`Snapshot` line 461, `Restore` line 498) keep working unchanged.

---

### `internal/udprepl/repl.go` modification (sync-arrived + pending mirrors)

**Analog:** same file — extend `mirrorStats` (lines 328-336, quoted above) with `a.syncArrived.Store(...)` from the new core accessor, and mirror the pending copy into an `atomic.Pointer[...]` (copy-on-write) after each loop iteration. The Adapter struct gains the atomic fields in the style of lines 108-113, and a copying accessor in the style of `KnownPeers()` (lines 267-269). The `Run` loop already calls `a.mirrorStats()` after every event (line 213) — no loop changes needed.

---

### `internal/grpcapi/server_test.go` (test, integration)

**Analog:** `internal/udprepl/repl_test.go` — external `_test` package, vanilla `testing`, hand-written fakes, `t.Parallel()`:
```go
package udprepl_test

import (
	...
	"testing"
	...
)

// stateStore adapts *storage.Store to repl.StateStore — the Phase 4
// consumer seam — by delegating both methods ...
type stateStore struct{ store *storage.Store }

func (s stateStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}
```
(lines 1-55: package decl, shared constants block, fake adapter; failure style is `t.Fatalf` with want/got phrasing throughout). Zero test dependencies — no testify (repo convention). The bufconn transport itself has no in-repo analog: use the probe-verified harness in RESEARCH Code Examples ("bufconn test harness": `bufconn.Listen`, `grpc.NewClient("passthrough:bufnet", grpc.WithContextDialer(...), grpc.WithTransportCredentials(insecure.NewCredentials()))` — the insecure credentials are mandatory). Fake `port.DumpStore` for dump-failure injection: in-memory fake precedent documented in `internal/port/dump.go:4-6` ("tests may substitute in-memory fakes that record timestamps and inject failures"). If a white-box seam is needed for cross-package tests, follow `internal/udprepl/repl_internal_test.go` (exported `...ForTest` helpers in an in-package test file).

---

### `cmd/sup-kv/main_test.go` smoke + `cmd/sup-kv/main.go` (config/entrypoint)

**Smoke-test analog:** `internal/udprepl/smoke_test.go`:
- free-port picker (lines 42-51, `freeUDPPort` — bind `:0`, read port, close; the TCP variant for the gRPC listener is the same trick with `net.Listen`),
- environment capability probe + loud `t.Skipf` naming what was probed and why (lines 213-240),
- `waitFor` polling with deadline margins (lines 175-190),
- real lifecycle: `go func() { _ = adapter.Run(ctx) }()` + `t.Cleanup(cancel)` (lines 255-258).
The binary smoke execs the built binary on ephemeral ports + tmp dump dir, checks health over real TCP, sends SIGTERM, asserts clean exit + final dump file exists (TOOL-01 test map row in RESEARCH).

**main.go analog:** none — the skeleton (`cmd/sup-kv/main.go:1-9`) explicitly reserves this: "Flags, environment configuration, signal handling and graceful shutdown arrive with the Phase 5 binary." Build it from RESEARCH Pattern 5 (stdlib `flag`, `flag.XxxVar(&v, name, envOrDefault("SUPKV_X", default), usage)` — env sets the default, an explicit flag overrides ⇒ flag > env > default with zero extra machinery; `var version, commit string` via ldflags `-X`) and Pattern 6 (shutdown order: health NOT_SERVING → cancel Watch ctxs → GracefulStop-in-goroutine + timeout → Stop fallback → cancel repl ctx → `Engine.Flush()` → cancel engine ctx). AGENTS.md pins: main() holds flags/env/signals and nothing testable.

---

### `internal/port/grpcapi.go` (CONDITIONAL)

**Analog if needed:** `internal/port/repl.go` — decoded-structs seam (lines 28-37):
```go
// Message is the sealed interface of the replication message structs. The
// port seam carries these decoded structs, not bytes (D-01): ...
type Message interface {
	isMessage()
}
```
…and `internal/port/doc.go` documents the package's per-phase seam inventory and the stdlib-only lint rule. Only create this file if the planner finds a signature that would otherwise drag a `google.golang.org/*` type across a boundary the gate protects — per RESEARCH's recommended structure, none exists in this phase.

---

## Shared Patterns

### Error sentinel → RPC code mapping

**Source:** `internal/app/storage/errors.go` (whole file), `internal/app/persist/errors.go` (whole file), `internal/udprepl/repl.go:66-72`, `internal/persistfile/snapshot.go:24-28`.
**Apply to:** `internal/grpcapi/server.go` (single mapping helper used by every handler).

```go
// storage/errors.go — the mapping is pre-pinned in the sentinels' own docs:
//   ErrNotFound       → codes.NotFound
//   ErrAlreadyExists  → codes.AlreadyExists
//   ErrValueTooLarge  → codes.InvalidArgument
// persist/errors.go:
//   ErrDumpFailed     → codes.Unavailable
//   ErrClosed         → codes.Unavailable
// udprepl:
//   ErrRunLoopExited  → codes.Unavailable
// persistfile:
//   ErrInvalidSnapshotName → codes.InvalidArgument
```
Match with `errors.Is`, never `==` (both error files' header comments). Gate-closed (API-04) → `codes.Unavailable`. Sentinel declaration style to copy: package-level `var (...)` block, each error documented with its Phase-5 RPC code.

### Depguard architecture boundary

**Source:** `.golangci.yml` lines 68-88.
**Apply to:** every new file — generated code and grpc imports stay under `internal/grpcapi/` only; `internal/app/**` may not import `net`/`os`/`syscall`/`google.golang.org/*`; `internal/port/**` is stdlib+own-module only. Generated `*.pb.go` files are auto-exempt from lint/formatters via `\.[\w-]+\.go$` (`.golangci.yml:491-492,520-521`) — no config change this phase.

```yaml
        sup-kv-app-no-io:
          files:
            - '**/internal/app/**'
            - '!$test'
          deny:
            - pkg: net
            - pkg: os
            - pkg: syscall
            - pkg: google.golang.org/
        sup-kv-port-stdlib-only:
          files:
            - '**/internal/port/**'
            - '!$test'
          list-mode: strict
          allow:
            - $gostd
            - github.com/Djarvur/sup-kv
```

### Single-goroutine core + atomic-mirror concurrency discipline

**Source:** `internal/app/repl/core.go:220-225` (Core doc: "Not safe for concurrent use: exactly one driver goroutine…") + `internal/udprepl/repl.go:91-114, 253-269, 328-336`.
**Apply to:** `watch.go` hub, `gate.go`, `wait.go` — gRPC handler goroutines NEVER call `repl.Core` methods or `PendingSnapshot()` directly; they read adapter-mirrored atomics only. The hub is the single owner of subscriber state (same discipline, one level up).

### Defensive copy at every seam

**Source:** `internal/app/storage/store.go:230-238` (install clones Data), `store.go:243-256` (Records clones), `internal/udprepl/repl.go:226-235` (NotifyLocalWrite clones the record), `internal/app/repl/core.go:570-576` (core keeps its own copy).
**Apply to:** the OnChange tap payload crossing into the hub, and any record retained by watch subscribers — callers keep ownership of their slices.

### Timer discipline (Go 1.23+ semantics)

**Source:** `internal/udprepl/repl.go:341-366` (`armTimer`/`timerChan`/`stopTimer`), `internal/app/persist/engine.go:328-343` (same helpers).
**Apply to:** `gate.go` (startup_gate deadline) and `wait.go` (per-request timeout) — plain Reset/Stop, no drain selects; nil-timer → nil channel disables the select case.

### Config-func over interface

**Source:** `internal/app/repl/core.go:86-96` (`Config.Log func(msg string)` + nil→no-op normalization at 125-127).
**Apply to:** `storage.Config.OnChange` — optional plain function in Config, never an observer interface; keeps `internal/app` free of new coupling.

### Consumer-side interfaces declared where consumed

**Source:** `internal/app/repl/core.go:131-148` (`StateStore`, with the storage.Clock precedent cited), `internal/port/dump.go:16-40` (`DumpStore` with the `//nolint:iface` justification pattern at line 15).
**Apply to:** any interface grpcapi needs (e.g., the engine/adapter surface the server consumes) — declare it in the consuming package with a doc comment explaining why; `//nolint` only with linter name + reason.

### Testing conventions

**Source:** `internal/udprepl/repl_test.go` (external `_test` package, `t.Parallel()`, shared constants block, hand-written fakes, `t.Fatalf` want/got phrasing), `internal/udprepl/repl_internal_test.go` (white-box via exported `...ForTest` helpers with a "White-box on purpose" doc comment), `internal/udprepl/smoke_test.go` (capability probe + loud skip).
**Apply to:** `server_test.go` and the cmd smoke test — vanilla `testing` only, `-race` mandatory, no assertion libraries.

## No Analog Found

Files/parts with no close match in the codebase — planner uses RESEARCH.md patterns instead:

| File / Part | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `cmd/sup-kv/main.go` flags/env/signals/shutdown orchestration | config/entrypoint | lifecycle | Nothing in the repo reads flags, env or signals; skeleton is empty. RESEARCH Pattern 5 (flag>env>default via `envOrDefault`, ldflags version) and Pattern 6 (7-step shutdown order) are the pattern source |
| `internal/grpcapi/gate.go` health wiring + gate goroutine | middleware | event-driven | No health/goroutine-signal code exists. RESEARCH Code Examples "Health server wiring + gate" (probe-verified on grpc v1.85.0); lifecycle shape from `udprepl.Run` |
| `internal/grpcapi/server_test.go` bufconn harness | test | request-response | No gRPC test transport in repo. RESEARCH Code Examples "bufconn test harness" (probe-verified; `insecure.NewCredentials()` mandatory); test *structure* analog is `repl_test.go` |
| gRPC server-streaming (Watch RPC) | controller | streaming | No streaming code in repo. Hub design is RESEARCH Pattern 2 (snapshot-then-live cut, version-dominance suppression, disconnect-on-full); ordering guarantee comes from the storage tap under the write lock |

## Metadata

**Analog search scope:** whole tracked tree (`git ls-files`, 74 source files); deep reads: `internal/udprepl/` (3 files), `internal/port/` (3 files), `internal/app/{storage,persist,repl}` cores, `internal/persistfile/snapshot.go`, `cmd/sup-kv/main.go`, `Makefile`, `go.mod`, `.golangci.yml` (depguard + exclusion sections)
**Files scanned:** 18 read in full/section; repo-wide ls-files review
**Pattern extraction date:** 2026-11-08
**Tracked-source gate:** all analog paths verified via `git ls-files -- <path>` (non-empty for every path named above)
