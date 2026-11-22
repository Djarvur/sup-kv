# Phase 6: API and Semantics - Pattern Map

**Mapped:** 2026-11-22
**Files analyzed:** 41 entries (7 new production/schema/doc, 15 modified production, 12 test groups, 4 config/tooling, 4 doc groups)
**Analogs found:** 40 / 41 (one no-analog: the descriptor-pin test)
**Repo state:** commit `abadd1b` (`docs(06): research phase domain`), clean tree at read time
**Source gate:** every analog path named below is git-TRACKED source (`git ls-files`-verified for the 29 core analogs). The move source `internal/grpcapi/apiproto/supkvapi/v1/*` is tracked, not a gitignored mirror; after the move the planner must name `api/supkvapi/v1/*` and must never reintroduce the old internal path in plans.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/grpcapi/limits.go` (NEW) | config | request-response | `internal/grpcapi/gate.go` | role-match |
| `api/supkvapi/v1/supkvapi.proto` (moved + List fields) | schema | request-response + streaming | `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` | exact (move) |
| `api/supkvapi/v1/supkvapi.pb.go`, `supkvapi_grpc.pb.go` (regen) | generated | — | the same two files under `internal/grpcapi/apiproto/supkvapi/v1/` | exact (regen) |
| `api/supkvapi/v1/doc.go` (NEW) | doc | — | `internal/app/storage/doc.go` | role-match |
| `examples/client/{go.mod,go.sum,main.go,README.md}` (NEW module) | client/example | request-response + streaming | `internal/stand/stand.go` client setup | partial |
| `internal/grpcapi/server.go` | controller | CRUD, request-response | itself (handler + mapper sections) + `internal/app/repl/core.go` cheap-view discipline | exact (self) |
| `internal/grpcapi/wait.go` | controller helper | request-response | itself + `internal/udprepl/repl.go` tracker | exact (self) |
| `internal/grpcapi/watch.go` | provider/hub | event-driven, streaming | itself + Phase-4 listing mirror (`internal/udprepl/repl.go`, `internal/app/repl/core.go`) | exact (self) |
| `internal/grpcapi/gate.go` | middleware/config | request-response | itself | exact (self) |
| `internal/udprepl/repl.go` | adapter/service | event-driven | itself (tracker/result/mirror sections) | exact (self) |
| `internal/app/repl/core.go` | service/state machine | event-driven | itself (`trackLocalWrite`, `handleHello`, `handleAck`, accessors) | exact (self) |
| `internal/app/storage/store.go` | model/store | CRUD | itself (`install`, `Generation`, `Versions`, `Lookup`) | exact (self) |
| `internal/app/persist/engine.go` | config/service | batch | itself (`Config.OnDumpFailure`, `normalized`) | exact (self) |
| `internal/app/persist/scheduler.go` | service | batch/streaming | itself (`runDump`) | exact (self) |
| `internal/app/persist/confirm.go` (only if the watermark lives here) | service | event-driven | itself (`capture`, `confirmWaiters`) | exact (self) |
| `internal/app/persist/facade.go` | service | CRUD | itself (`Versions`, `KeyCount` forwards) | exact (self) |
| `internal/app/persist/snapshot.go` | service | batch/transform | itself (`replaySnapshot`, `restoreChanges`, `replayRecord`) | exact (self) |
| `internal/app/persist/loader.go` | service | file-I/O | itself (`Open`, `mergeRecords`) | exact (self) |
| `cmd/sup-kv/main.go` | composition root/config | request-response | itself (`wire`, `listenGRPC`, `shutdown`) | exact (self) |
| `internal/app/internal/simnet/simnet.go` | test-support driver | event-driven | itself (`Write`/`Delete` notify call sites) | exact (self) |
| `internal/grpcapi/errors_test.go` (NEW) | test | request-response | `gate_test.go` (`newFileServer`) + `maxdatagram_test.go` table style + `shutdown_internal_test.go` slog capture | role-match |
| `internal/grpcapi/limits_test.go` (NEW) | test | request-response | `maxdatagram_test.go` + `gate_test.go` health test | role-match |
| `api/supkvapi/v1/schema_test.go` (NEW) | test | — | none (no `protoreflect` test exists); nearest behavior pin is `server_test.go` roundtrips | no-analog |
| `internal/grpcapi/wait_test.go` | test | request-response | itself (`fakeTracker`, zero-peer tests) | exact (self) |
| `internal/grpcapi/watch_test.go` | test | event-driven | itself | exact (self) |
| `internal/grpcapi/gate_test.go` | test | request-response | itself (`TestHealth_reflectsGate`) | exact (self) |
| `internal/grpcapi/server_test.go` | test | CRUD | itself (`TestServer_list`, snapshot-load totality tests) | exact (self) |
| `internal/grpcapi/{bench_test.go,wait_integration_test.go,watch_senderr_test.go}` | test | ripple | their own `OnChange: hub.Ingest` / `NotifyLocalWrite` call sites | exact (self) |
| `internal/app/storage/{store_test.go,view_test.go}` | test | CRUD | `changeTapRecorder`, `TestStore_versions_snapshot` | exact (self) |
| `internal/app/persist/engine_test.go` | test | batch | `TestEngine_snapshotCreateLoadRestore` | exact (self) |
| `internal/udprepl/repl_test.go` (+ internal tests) | test | event-driven | `TestWaitReplicated_ackCoverage` family | exact (self) |
| `internal/app/repl/{retransmit_test.go,peertable_test.go,sim_*}` | test | event-driven | `TestRetransmit_suspendAndFlush` | exact (self) |
| `cmd/sup-kv/main_test.go` | test | request-response | health-check helpers + `TestShutdown_secondSignal` | exact (self) |
| 19 API importers (see list below) | mechanical | — | `internal/stand/stand.go:44` alias line | exact |
| `internal/grpcapi/red_gate_test.go`, `internal/stand/stand_red_test.go` | frozen test exception | — | their own import block (one line only) | exact |
| `buf.yaml`, `buf.gen.api.yaml`, `Makefile` | config | — | their current internal-path lines | exact |
| `.golangci.yml` | config | — | existing `sup-kv-app-no-io` rule | exact |
| `.github/workflows/ci.yml` | config | — | existing `lint` job / `gate` steps | role-match |
| `docs/API.md` (NEW) | doc | — | `docs/OPERATIONS.md` + `docs/04-MULTI-NODE.md` | role-match |
| `docs/EXAMPLES.md`, `docs/04-MULTI-NODE.md`, `docs/SPEC-DEVIATIONS.md`, `AGENTS.md` | doc | — | their own sections | exact |
| `scripts/red-expectations.tsv` | untouched | — | — | no change |

## Pattern Assignments

### `internal/grpcapi/limits.go` (NEW — config, request-response)

**Analog:** `internal/grpcapi/gate.go`

**Config-struct + constant pattern** (`internal/grpcapi/gate.go:15-31`):

```go
const (
	// gatePollInterval is the resolver's poll period. A8 pins the design:
	// one small goroutine polling every 20ms is acceptable ...
	gatePollInterval = 20 * time.Millisecond

	// defaultStartupGate is the design default (IDEA.md §Parameters,
	// startup_gate 1s), substituted for a zero or negative StartupGate so
	// a misconfigured option can never mean "never open".
	defaultStartupGate = time.Second

	// gateMessage is the status message every gated write RPC carries. ...
	gateMessage = "startup gate: waiting for first sync round or startup_gate"
)
```

**Injected-options struct pattern** (`internal/grpcapi/gate.go:47-56`, abridged):

```go
type ServerOptions struct {
	SyncArrived func() bool
	StartupGate time.Duration
	ListedKey   func(key string) bool
	KnownPeers  func() uint64
	WaitReplicated func(ctx context.Context, key string, ver version.RecordVersion, total uint32, timeout time.Duration) (acked bool, ackedCount uint32)
	WaitTimeout time.Duration
	ReplayNotify LocalReplayNotify
}
```

**Size constants the limits derive from** (`internal/app/storage/store.go:20-33`):

```go
const (
	ValueMaxBytes  = 61440
	notDeletedFlag = 0
	deletedFlag    = 1
)
...
const MaxKeyBytes = 1024
```

**Apply:**
- Define `TransportLimits{MaxRecvBytes, MaxSendBytes, KeepaliveTime, KeepaliveTimeout, MinPingInterval}`, `LimitsFor(valueMaxBytes int) TransportLimits`, and `(TransportLimits) ServerOptions() []grpc.ServerOption` in a new file that imports `google.golang.org/grpc`, `.../keepalive`, and `storage`.
- Compute recv from `storage.MaxKeyBytes + valueMaxBytes + envelope`, send from the List page byte budget + envelope — never from `storage.ValueMaxBytes` alone (research Pitfall 11). Document every constant inline in the gate.go style (why, not just what).
- Keep the comment discipline: exported knobs explain the operational envelope and the pinned default.

---

### `api/supkvapi/v1/*` (moved schema + regenerated code)

**Analog:** the tracked files at `internal/grpcapi/apiproto/supkvapi/v1/` (`supkvapi.proto`, `supkvapi.pb.go`, `supkvapi_grpc.pb.go`).

**Current `go_package`** (`internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:10-14`):

```proto
syntax = "proto3";

package supkvapi.v1;

option go_package = "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1;apiproto";
```

**Generation config today** (`buf.yaml:8-20`):

```yaml
modules:
  - path: internal/udprepl/replproto
  - path: internal/grpcapi/apiproto
```

(`buf.gen.api.yaml:7-13`):

```yaml
plugins:
  - local: ["go", "tool", "protoc-gen-go"]
    out: internal/grpcapi/apiproto
    opt: paths=source_relative
  - local: ["go", "tool", "protoc-gen-go-grpc"]
    out: internal/grpcapi/apiproto
    opt: paths=source_relative
```

(`Makefile:98-100`):

```make
proto:
	go tool buf generate internal/udprepl/replproto --template buf.gen.repl.yaml
	go tool buf generate internal/grpcapi/apiproto --template buf.gen.api.yaml
```

**Generated header to preserve in shape** (`supkvapi.pb.go:9-16`):

```go
// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.12-devel
// 	protoc        (unknown)
// source: supkvapi/v1/supkvapi.proto

package apiproto
```

**Apply:**
- Move the tree to `api/supkvapi/v1/`, set `option go_package = "github.com/Djarvur/sup-kv/api/supkvapi/v1;supkvapi";`, change `buf.yaml` module path to `api`, `buf.gen.api.yaml` `out: api`, Makefile to `buf generate api`.
- Add the additive List fields (`page_size`, `page_token`, `next_page_token`, `repeated Record records`) in the same codegen pass; strip internal phase/decision references from the now-public comments (research "Deprecated/outdated").
- Add the scoped `ignore_only: FILE_SAME_GO_PACKAGE` for `api/supkvapi/v1/supkvapi.proto` (research D-06 probe evidence); keep the explanation comment in `buf.yaml`.
- Generated code is regenerated, not hand-edited; pin the *wire* descriptor in `schema_test.go` instead of claiming byte-identity.

**Importer alias pattern** (`internal/stand/stand.go:44`, `internal/grpcapi/server.go:29`, `cmd/sup-kv/main.go:41`):

```go
apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
```

**All 19 real importers** (one-line change each, alias preserved):

```text
cmd/sup-kv/main.go
cmd/sup-kv/main_test.go
internal/grpcapi/{server.go,watch.go,bench_test.go,gate_test.go,red_gate_test.go,
  server_test.go,wait_integration_test.go,wait_test.go,watch_senderr_test.go,watch_test.go}
internal/stand/{stand.go,persist_test.go,stand_buffers_test.go,stand_delete_test.go,
  stand_metrics_test.go,stand_red_test.go,stand_test.go}
```

Do **not** touch the two historical evidence files under `docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/` (never built; historical evidence) and do not add a forwarding package.

---

### `api/supkvapi/v1/doc.go` (NEW — doc)

**Analog:** `internal/app/storage/doc.go:1-37` (the package-doc voice: what the package is, the invariants, the deliberate asymmetries, what it does not do).

Representative excerpt (`internal/app/storage/doc.go:1-8`):

```go
// Package storage is the sup-kv in-memory LWW-Element-Map store: a pure,
// I/O-free library over records versioned by the sibling version package
// and stamped by a clock supplied by the caller through the consumer-side
// Clock interface declared here — the concrete clock (internal/app/clock)
// satisfies it and tests may substitute fakes; per D-03, internal/port
// holds no interfaces. The lint gate forbids networking, filesystem,
// syscall and gRPC/protobuf imports here — see the sup-kv-app-no-io
// depguard rule in .golangci.yml.
```

Also see `internal/app/persist/doc.go:1-28` for the "five internal roles + API contract in prose" shape.

**Apply:** one package comment stating the public contract: the full-record wire shape, codes/semantics promised by the service, that generation is reproducible via `make proto`, and that this package must not import `internal/` (depguard enforces it). No code in `doc.go`.

---

### `internal/grpcapi/server.go` (controller, request-response + CRUD)

**Analog:** itself; the sections below are the exact code being replaced/extended.

**Write handler gate-first + validation order** (`internal/grpcapi/server.go:189-230`, abridged):

```go
func (s *Server) Set(ctx context.Context, req *apiproto.SetRequest) (*apiproto.SetResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	if len(req.GetKey()) > storage.MaxKeyBytes {
		return nil, keyTooLongError(len(req.GetKey()))
	}
	...
	ver, _, err = s.engine.Set(req.GetKey(), req.GetValue())
	if err != nil {
		if errors.Is(err, persist.ErrDumpFailed) || errors.Is(err, persist.ErrUnconfirmed) {
			s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, false)
		}
		return nil, mapError(err)
	}
	s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, req.GetWaitForReplication())
	resp := &apiproto.SetResponse{Version: ver[:]}
	if req.GetWaitForReplication() {
		timeout := s.waitTimeout(req.GetWaitTimeoutMs())
		resp.Acked, resp.AckedCount, resp.TotalPeers = s.waitForReplication(ctx, req.GetKey(), ver, timeout)
	}
	return resp, nil
}
```

**Current `List` (the A-02 defect)** (`internal/grpcapi/server.go:292-306`):

```go
func (s *Server) List(_ context.Context, req *apiproto.ListRequest) (*apiproto.ListResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	records := s.engine.Records()
	keys := make([]string, 0, len(records))
	for key, rec := range records {
		if rec.Deleted || !strings.HasPrefix(key, req.GetPrefix()) {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return &apiproto.ListResponse{Keys: keys}, nil
}
```

**Cheap key→version view to copy** (`internal/app/storage/store.go:234-246`):

```go
func (s *Store) Versions() map[string]version.RecordVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]version.RecordVersion, len(s.records))
	for key, rec := range s.records {
		out[key] = rec.Version
	}
	return out
}
```

**Tombstone-visible point read for the page's values** (`internal/app/storage/store.go:215-232`):

```go
func (s *Store) Lookup(key string) (Record, bool) {
	s.mu.RLock()
	rec, ok := s.records[key]
	s.mu.RUnlock()
	if !ok {
		return Record{}, false
	}
	return Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Deleted:  rec.Deleted,
		Checksum: rec.Checksum,
	}, true
}
```

**Current `mapError` (the A-03 leak)** (`internal/grpcapi/server.go:433-454`):

```go
func mapError(err error) error {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return status.Error(codes.NotFound, storage.ErrNotFound.Error())
	case errors.Is(err, storage.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, storage.ErrAlreadyExists.Error())
	case errors.Is(err, storage.ErrValueTooLarge):
		return status.Error(codes.InvalidArgument, storage.ErrValueTooLarge.Error())
	case errors.Is(err, persist.ErrDumpFailed):
		return status.Error(codes.Unavailable, persist.ErrDumpFailed.Error())
	case errors.Is(err, persist.ErrUnconfirmed):
		return status.Error(codes.Unavailable, persist.ErrUnconfirmed.Error())
	case errors.Is(err, persist.ErrClosed):
		return status.Error(codes.Unavailable, persist.ErrClosed.Error())
	case errors.Is(err, persistfile.ErrInvalidSnapshotName):
		return status.Error(codes.InvalidArgument, persistfile.ErrInvalidSnapshotName.Error())
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Errorf(codes.Internal, "internal error: %v", err)
}
```

**Error classes the total table must cover** — sentinels and their path status:

- `internal/app/storage/errors.go:5-19` — `ErrNotFound` ("key not found"), `ErrAlreadyExists` ("key already exists"), `ErrValueTooLarge` ("value exceeds the maximum allowed size").
- `internal/app/persist/errors.go:7-25` — `ErrDumpFailed` ("dump failed; write not confirmed"), `ErrUnconfirmed` ("write not confirmed before the confirmation timeout"), `ErrClosed` ("persist engine closed").
- `internal/persistfile/snapshot.go:20-29` — `ErrNoDump` ("no dump on disk to snapshot") and `ErrInvalidSnapshotName` ("invalid snapshot name").
- `internal/persistfile/snapshot.go:67-74` — `ReadSnapshot` returns the raw `os.ReadFile` error (`*fs.PathError` with the configured path); this is the concrete A-03 leak and must map to `NotFound` for `fs.ErrNotExist`.
- `internal/port/dump.go:127-157` — `FailureClass` vocabulary (`FailureDiskFull`, `FailurePermission`, `FailureOther`) and `FailureClassOf` (returns `FailureOther` for unclassified errors).
- `internal/persistfile/saveerror.go:32-46` — the classifier; note `SaveStream` wraps with it (`dumpfile.go:107-111`) while `CreateSnapshot` currently does not (`snapshot.go:46-59`) — wrap it too (Pitfall 12).

**Static log + detail convention** (`cmd/sup-kv/main.go:583-588`):

```go
		// One ERROR per failed attempt with the class label the metric uses
		// (D-05, PERSIST-05): a static message plus attributes, never dynamic
		// text, and the engine's dumper goroutine only formats and logs.
		OnDumpFailure: func(err error, class port.FailureClass) {
			slog.Error("Dump failed", "class", class.String(), "detail", err.Error())
		},
```

**Apply:**
- `List`: gate → page-size resolve/clamp → base64 token decode + `MaxKeyBytes` bound → `engine.LiveVersions()` (new, tombstone-free key→version) → prefix filter/seek past token → sorted walk to page size or byte budget → `engine.Lookup` only for page entries → `next_page_token` = last examined key. Page values are copied only for the page; `Engine.Records()` must disappear from the request path.
- `mapError`: keep the existing `errors.Is` chain order, add `ErrNoDump → FailedPrecondition`, `fs.ErrNotExist → NotFound` ("snapshot not found"), `fs.ErrPermission → PermissionDenied`, class checks **after** `ErrDumpFailed` (`port.FailureClassOf`), context errors via `status.FromContextError`, pass-through status errors, then log the detail and return the constant `"internal error"` — never `%v` in the message. Totality comes from the constant fallback.
- `Set`/`Delete`: capture `peers := s.writePeers()` immediately before the engine call and pass them to both the notify and the wait; keep the failed-write notify-before-error branch exactly as today.
- `SnapshotLoad`: no structural change — the rollback tombstones arrive in `Restore`'s returned list and already broadcast (lines 341-360).

---

### `internal/grpcapi/wait.go` (controller helper, request-response)

**Analog:** itself + `internal/udprepl/repl.go` tracker contract.

**Current function** (`internal/grpcapi/wait.go:66-84`):

```go
func (s *Server) waitForReplication(ctx context.Context, key string, ver version.RecordVersion, timeout time.Duration) (acked bool, ackedCount, total uint32) {
	if s.opts.WaitReplicated == nil {
		return true, 0, 0
	}
	var peerCount uint32
	if s.opts.KnownPeers != nil {
		peerCount = uint32(s.opts.KnownPeers()) //nolint:gosec // G115: ...
	}
	if peerCount == 0 {
		// Zero known peers at write time (D-11): immediate acked with
		// nothing awaited. ...
		_, _ = s.opts.WaitReplicated(ctx, key, ver, 0, timeout)
		return true, 0, 0
	}
	acked, count := s.opts.WaitReplicated(ctx, key, ver, peerCount, timeout)
	return acked, min(count, peerCount), peerCount
}
```

**Retained timeout-clamp discipline** (`internal/grpcapi/wait.go:86-113`) — keep verbatim; the request field is hostile input and the write is already confirmed when this resolves.

**Tracker success rule to mirror** (`internal/udprepl/repl.go:800-806`):

```go
	tr.mu.Lock()
	// The acked set holds uint16 peer ids, so its length cannot truncate.
	count := min(uint32(len(tr.acked)), total)
	ok := tr.resolved && tr.ok && count == total
	tr.mu.Unlock()
	a.deleteTracker(k)
	return ok, count
```

**Apply:**
- Signature becomes `waitForReplication(ctx, key, ver, peers []uint16, timeout)` (or `total uint32` computed from the captured slice). Zero peers → call the seam with `total=0` for tracker release, return `(false, 0, 0)`.
- A nil `WaitReplicated` seam is also `(false, 0, 0)` now — honesty over convenience (D-01).
- Never read `KnownPeers()` here; the write-time capture is handler state.
- Add the contract prose to `docs/API.md`: write-time set, zero-peer answer, superseded fast path, caller-bounded timeout.

---

### `internal/grpcapi/watch.go` (provider/hub, event-driven streaming)

**Analog:** itself for the loop/hub discipline; the Phase-4 listing mirror for generation-gated publication.

**Single-owner loop** (`internal/grpcapi/watch.go:184-202`):

```go
func (h *WatchHub) Run(ctx context.Context) error {
	defer close(h.done)
	defer h.shutdownSubscribers()
	for {
		if h.overflow.CompareAndSwap(true, false) {
			h.disconnectAll()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev := <-h.ingress:
			h.dispatch(ev)
		case req := <-h.subscribeCh:
			h.handleSubscribe(req)
		case sub := <-h.unsubscribeCh:
			h.remove(sub)
		}
	}
}
```

**Non-blocking tap ingress** (`internal/grpcapi/watch.go:213-219`):

```go
func (h *WatchHub) Ingest(key string, rec storage.Record) {
	select {
	case h.ingress <- watchEvent{key: key, rec: rec}:
	default:
		h.overflow.Store(true)
	}
}
```

**Exact-cut dispatch** (`internal/grpcapi/watch.go:290-315`, abridged):

```go
func (h *WatchHub) dispatch(ev watchEvent) {
	for sub := range h.subs {
		if !h.wants(sub, ev) {
			continue
		}
		select {
		case sub.events <- ev:
		default:
			h.disconnect(sub)
		}
	}
}

func (*WatchHub) wants(sub *watchSubscriber, ev watchEvent) bool {
	if !strings.HasPrefix(ev.key, sub.prefix) {
		return false
	}
	snapVer, seen := sub.snapshot[ev.key]
	return !seen || ev.rec.Version.Compare(snapVer) > 0
}
```

**Shutdown status hook** (`internal/grpcapi/watch.go:347-355` and `392-399`):

```go
func (h *WatchHub) shutdownSubscribers() {
	for sub := range h.subs {
		delete(h.subs, sub)
		close(sub.events)
	}
}
...
		case ev, open := <-got.sub.events:
			if !open {
				return got.sub.err
			}
```

**Generation-gated immutable publication (the Phase-4 mirror pattern)** (`internal/udprepl/repl.go:1079-1125`, abridged):

```go
func (a *Adapter) mirrorStats() {
	d := a.core.Discards()
	...
	// The listing mirror is content-gated: the core's generation moves only
	// on an actual {complete, union} change, so an identical listing flood
	// republishes no pointer and clones no map. ...
	if gen := a.core.ListingGeneration(); gen != a.listingGen {
		a.listingGen = gen
		a.listing.Store(&listingMirror{
			complete: a.core.ListingComplete(),
			keys:     a.core.ListedKeysSnapshot(),
		})
	}
}
```

with the one-pointer publication (`internal/udprepl/repl.go:944-951`):

```go
type listingMirror struct {
	complete bool
	keys     map[string]struct{}
}
```

and the generation contract (`internal/app/repl/core.go:2973-2989`):

```go
func (c *Core) SyncArrived() bool { ... }
func (c *Core) ListingGeneration() uint64 { return c.listingGen }
func (c *Core) ListingComplete() bool { return c.listingComplete }
```

**Apply:**
- `watchEvent` gains `gen uint64`; `Ingest(key, rec, gen)` enqueues it.
- Hub keeps `pending []watchEvent` (bounded by `watchIngress`), a `confirmed atomic.Uint64`, and a buffered(1) wake channel; `Confirm(gen)` does atomic-max + non-blocking wake; `Run` releases every leading pending event with `gen <= confirmed` in arrival order before/after each select iteration.
- On overflow: disconnect all, clear pending (resubscribe+snapshot heals) — same policy, documented.
- `shutdownSubscribers` writes a fresh `status.Error(codes.Unavailable, "watch hub is shutting down")` into `sub.err` before `close`; the handler already returns `sub.err`.
- Keep `handleSubscribe`'s snapshot semantics unchanged (current in-memory state; document the asymmetry in `docs/API.md`).

---

### `internal/grpcapi/gate.go` (middleware/config, request-response)

**Analog:** itself.

**Options/health binding** (`internal/grpcapi/gate.go:114-133`, abridged):

```go
func (s *Server) Health() *health.Server {
	return s.healthSrv
}

func (s *Server) gated() bool {
	return !s.gateOpen.Load()
}
```

**One-way opening CAS** (`internal/grpcapi/gate.go:170-179`):

```go
func (s *Server) openGate() {
	if s.gateOpen.CompareAndSwap(false, true) {
		s.healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	}
}
```

**Apply:**
- Replace `KnownPeers func() uint64` with `WritePeers func() []uint16` (sorted; nil = zero peers). Keep `ListedKey` and `SyncArrived` untouched.
- Add `func (s *Server) Shutdown()` calling `s.healthSrv.Shutdown()` and nothing else (idempotent by the health server's own contract; never call `Resume`). Comment the one-way behavior and reference the grpc v1.85 implementation.
- Keep options exported and documented in the gate.go style (production injects adapter mirrors; tests inject fakes).

---

### `internal/udprepl/repl.go` (adapter/service, event-driven)

**Analog:** itself.

**Tracker state to extend** (`internal/udprepl/repl.go:153-166`):

```go
type tracker struct {
	mu       sync.Mutex
	acked    []uint16
	resolved bool
	ok       bool
	done     chan struct{}
}
```

Add `total uint32` and `wants map[uint16]struct{}`; success becomes `resolved && ok && count == total` where `count` counts `wants` members only.

**Synchronous creation site — do not move** (`internal/udprepl/repl.go:594-637`, abridged):

```go
func (a *Adapter) NotifyLocalWrite(key string, rec storage.Record, track bool) error {
	keyID := trackerKey{key: key, version: rec.Version}
	if track {
		a.ensureTracker(keyID)
	}
	err := a.enqueueLocalWrite(localWrite{key: key, rec: cloneLocalRecord(rec)})
	if err != nil {
		a.deleteTracker(keyID)
		return err
	}
	return nil
}
```

**Result application to extend** (`internal/udprepl/repl.go:736-753`):

```go
func (a *Adapter) applyResults(out *port.Outbox) {
	for _, res := range out.DrainResults() {
		tr := a.lookupTracker(trackerKey{key: res.Key, version: res.Version})
		if tr == nil {
			continue
		}
		tr.mu.Lock()
		switch res.Kind {
		case port.ResultAck:
			tr.acked = insertPeer(tr.acked, res.Peer)
		case port.ResultDrained:
			tr.resolveLocked(true)
		case port.ResultSuperseded, port.ResultWaitSetEmpty:
			tr.resolveLocked(false)
		}
		tr.mu.Unlock()
	}
}
```

**Wait release/cleanup contract** (`internal/udprepl/repl.go:783-807`): total==0 deletes and returns; absent tracker returns `(false, 0)`; every return path deletes.

**Content-gated mirror to copy for `KnownPeerIDs`** (`internal/udprepl/repl.go:1100-1124`) — see the Watch section excerpt; add `peerIDs atomic.Pointer[[]uint16]`, publish only when `core.PeerGeneration()` moved, reading `core.PeerIDs()`.

**Adapter mirror/state fields** (`internal/udprepl/repl.go:289-336`) show where the new atomic pointer and generation belong; keep the "event-loop writes, any-goroutine reads" comment.

**Apply:**
- `NotifyLocalWrite(key, rec, track bool, peers []uint16)` — clone the slice into the tracker and carry it on `localWrite` to the core.
- `applyResults` inserts into `acked` only for `wants` members; `ResultDrained` remains the only `ok=true` signal.
- Add `KnownPeerIDs() []uint16` (sorted clone) and the peer-generation publication; keep `KnownPeers()` for metrics.
- `WaitReplicated` may drop the `total` parameter only if all call sites are updated; the tracker owns the total either way.

---

### `internal/app/repl/core.go` (service/state machine, event-driven)

**Analog:** itself.

**Current write-time set seeding** (`internal/app/repl/core.go:2226-2246`, abridged):

```go
	pk := pendingKey{key: key, version: rec.Version}
	_, tracked := c.pending[pk]
	if !tracked {
		waitSet := make(map[uint16]struct{}, len(c.peers))
		for id := range c.peers {
			waitSet[id] = struct{}{}
		}
		entry := pendingEntry{
			version:   rec.Version,
			waitSet:   waitSet,
			nextRetry: now.Add(c.cfg.AckTimeout),
			backoff:   c.cfg.AckTimeout,
			createdAt: now,
		}
		entry.elem = c.insertPendingOrder(pk, now)
		c.pending[pk] = entry
		for id := range waitSet {
			c.ledgerAdd(id, pk)
		}
		c.pushRetry(pk, entry.nextRetry)
	}
	return pk, true
```

**Discovery flush to split** (`internal/app/repl/core.go:1394-1406`):

```go
	if remembered || !discovery {
		return
	}
	for _, pk := range sortedPendingKeys(c.pending) {
		entry := c.pending[pk]
		entry.waitSet[in.NodeID] = struct{}{}
		entry.backoff = c.cfg.AckTimeout
		entry.nextRetry = now.Add(c.cfg.AckTimeout)
		c.pending[pk] = entry
		c.fixRetry(pk, entry.nextRetry)
		c.ledgerAdd(in.NodeID, pk)
		c.enqueueRecordSend(now, pk, false)
	}
```

Keep the ladder reset / `fixRetry` / `enqueueRecordSend`; delete the `waitSet` insert and `ledgerAdd` (Pitfall 4).

**Ack drain to keep** (`internal/app/repl/core.go:1567-1583`):

```go
func (c *Core) handleAck(peerID uint16, ack port.Ack, out *port.Outbox) {
	pk := pendingKey{key: ack.Key, version: ack.Version}
	entry, ok := c.pending[pk]
	if !ok {
		return
	}
	delete(entry.waitSet, peerID)
	c.ledgerRemove(peerID, pk)
	entry.acked = ackedInsert(entry.acked, peerID)
	out.Ack(pk.key, pk.version, peerID)
	if len(entry.waitSet) == 0 && len(c.peers) > 0 {
		out.Drained(pk.key, pk.version)
		c.removePending(pk)
		return
	}
	c.pending[pk] = entry
}
```

**Accessor/copying precedent for `PeerIDs`/`PeerGeneration`** (`internal/app/repl/core.go:2983-2989`, `3001-3011`, `3028-3035`):

```go
func (c *Core) ListingGeneration() uint64 { return c.listingGen }
func (c *Core) ListingComplete() bool { return c.listingComplete }

func (c *Core) ListedKeysSnapshot() map[string]struct{} {
	out := make(map[string]struct{}, len(c.listedKeys))
	for key := range c.listedKeys {
		out[key] = struct{}{}
	}
	return out
}

func (c *Core) PeerSnapshot() map[uint16]PeerInfo {
	out := make(map[uint16]PeerInfo, len(c.peers))
	for id := range c.peers {
		p := c.peers[id]
		out[id] = PeerInfo{Handle: p.handle, RunID: p.runID, LastHello: p.lastHello}
	}
	return out
}
```

**Superseded fast path already exists** at `core.go:2211-2218` (older local notification), `core.go:1523-1528` (newer remote record), and `removeSupersededLocal` (`core.go:3236-3243`) — the 20 ms path needs tests, not production changes.

**Apply:**
- `NotifyLocalWrite`/`NotifyReplayedWrite`/`trackLocalWrite` gain `peers []uint16`. Seed `waitSet` from the notification's set intersected with `c.peers`; when the captured set was non-empty and the intersection is empty, emit `WaitSetEmpty` immediately (honest false, no timeout).
- Add `PeerIDs() []uint16` (sorted clone) and `PeerGeneration() uint64`; bump the generation only when the peer table's id set actually changes.
- Update the tests that pin the old wait-set growth (`peertable_test.go`, `retransmit_test.go`, sim tests) deliberately.

---

### `internal/app/storage/store.go` (model/store, CRUD)

**Analog:** itself.

**Install point — generation + tap** (`internal/app/storage/store.go:417-436`):

```go
func (s *Store) install(key string, rec Record) {
	data := bytes.Clone(rec.Data)
	stored := Record{
		Version:  rec.Version,
		Data:     data,
		Deleted:  rec.Deleted,
		Checksum: RecordChecksum(key, rec.Version, data, rec.Deleted),
	}
	s.records[key] = stored
	s.generation++
	s.cfg.OnChange(key, stored)
}
```

**Tap signature + nil normalization** (`internal/app/storage/store.go:41-67`, `138-146`):

```go
type Config struct {
	ValueMaxBytes int
	OnChange func(key string, rec Record)
}

func (c Config) normalized() Config {
	c.ValueMaxBytes = c.ValueLimit()
	if c.OnChange == nil {
		c.OnChange = func(_ string, _ Record) {}
	}
	return c
}
```

**Generation contract** (`internal/app/storage/store.go:362-374`) — one per actual install, never for no-ops/reads.

**Cheap view to model `LiveVersions` on** (`internal/app/storage/store.go:234-246`), with the tombstone caveat from the `Versions` doc: `len(Data)==0` is NOT a tombstone test.

**Apply:**
- `OnChange func(key string, rec Record, generation uint64)`; pass `s.generation` after the increment. Update the nil no-op and every construction site (list below).
- Add `LiveVersions() map[string]version.RecordVersion` skipping `rec.Deleted` under `RLock`, copying no `Data` (Pitfall 5).
- Do not change `Generation()` itself; it is the watermark the persist engine reads.

---

### `internal/app/persist/engine.go` (config/service, batch)

**Analog:** itself, the `OnDumpFailure` callback shape (`internal/app/persist/engine.go:44-49`):

```go
	// OnDumpFailure receives every failed dump attempt exactly once with the
	// underlying error and the adapter's failure class (D-05, PERSIST-05):
	// the binary wires it to one ERROR log line and must not block. Nil is
	// valid — the counters and the retry ladder still apply.
	OnDumpFailure func(err error, class port.FailureClass)
```

**Normalization precedent** (`engine.go:51-73`): zero/negative fields get design defaults; nil seams stay nil when legal.

**Apply:** add `OnConfirmed func(generation uint64)` with the same "called once per successful dump, must not block, nil is valid" doc voice. Bump `DumpStats` only if a new counter is actually needed (the watermark lives on the engine/scheduler, not a metric in this phase).

---

### `internal/app/persist/scheduler.go` (service, batch/streaming)

**Analog:** itself, `runDump` (`internal/app/persist/scheduler.go:201-238`):

```go
func (s *scheduler) runDump() error {
	captured := s.conf.capture()

	start := s.cfg.Now()
	records := s.store.Records()
	err := s.dumps.SaveStream(func(w io.Writer) error {
		return EncodeTo(w, records, s.valueMaxBytes)
	})
	s.recordDump(s.cfg.Now().Sub(start), err)
	if err != nil {
		s.handleDumpFailure(err)
		s.conf.failWaiters(captured)
		return err
	}
	s.resetRetry()
	s.conf.confirmWaiters(captured)
	return nil
}
```

**Ordering proof the watermark inherits** (`scheduler.go:201-220` doc comment): capture happens **before** `Records()`, so every captured waiter/record is in the saved snapshot. The watermark generation must be read at the same place (`gen := s.store.Generation()` immediately before `records := s.store.Records()`), and published **after** `confirmWaiters` only on success.

**Apply:**
- On success: store the watermark (`confirmed.Store(gen)` under the scheduler/engine) and call `s.cfg.OnConfirmed(gen)`; nil check like `OnDumpFailure`.
- On failure: do nothing — pending Watch events stay pending and a later successful dump releases them exactly once.
- Do not take `Records()` earlier/later; the capture-before-snapshot invariant is load-bearing.

---

### `internal/app/persist/confirm.go` (service, event-driven — only if the watermark is published here)

**Analog:** itself, `capture` / `confirmWaiters` (`internal/app/persist/confirm.go:184-222`):

```go
func (c *confirmer) capture() []*waiter {
	c.mu.Lock()
	captured := c.waiters
	c.waiters = nil
	c.dirty = false
	c.mu.Unlock()
	return captured
}

func (c *confirmer) confirmWaiters(captured []*waiter) {
	c.mu.Lock()
	claimed := make([]*waiter, 0, len(captured))
	for _, w := range captured {
		if w.claimed {
			continue
		}
		w.claimed = true
		w.err = nil
		claimed = append(claimed, w)
	}
	c.mu.Unlock()
	for _, w := range claimed {
		close(w.done)
	}
}
```

The pattern to copy if the watermark becomes confirmer state: mutate under one mutex, close/wake outside it (never hold the mutex across a channel send that a waiter might need).

---

### `internal/app/persist/facade.go` (service, CRUD)

**Analog:** itself, the thin forwarding pattern (`internal/app/persist/facade.go:73-92`):

```go
func (e *Engine) Versions() map[string]version.RecordVersion {
	return e.store.Versions()
}
...
func (e *Engine) KeyCount() int {
	return e.store.Len()
}
```

**Apply:** add `func (e *Engine) LiveVersions() map[string]version.RecordVersion { return e.store.LiveVersions() }` with a doc comment stating "tombstone-free, copies no Data, consults no dump state" — same shape as `Versions`.

---

### `internal/app/persist/snapshot.go` (service, batch/transform)

**Analog:** itself.

**Current additive replay (the S-01 defect)** (`internal/app/persist/snapshot.go:117-168`, abridged):

```go
func (e *Engine) replaySnapshot(records map[string]storage.Record) ([]RestoredRecord, error) {
	restored := make([]RestoredRecord, 0, len(records))
	if !restoreChanges(e.store.Records(), records) {
		return restored, nil
	}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		ver, replayed, err := replayRecord(e.store, key, records[key])
		if err != nil {
			if len(restored) > 0 {
				e.markDirty()
			}
			return restored, err
		}
		if !replayed {
			continue
		}
		restored = append(restored, RestoredRecord{
			Key:     key,
			Version: ver,
			Data:    bytes.Clone(records[key].Data),
			Deleted: records[key].Deleted,
		})
	}
	e.markDirty()
	return restored, nil
}

func restoreChanges(current, snap map[string]storage.Record) bool {
	for key, rec := range snap {
		stored, ok := current[key]
		if !ok || stored.Version != rec.Version || stored.Deleted != rec.Deleted || !bytes.Equal(stored.Data, rec.Data) {
			return true
		}
	}
	return false
}
```

**Replay primitive to reuse** (`internal/app/persist/snapshot.go:170-200`):

```go
func replayRecord(store *storage.Store, key string, rec storage.Record) (version.RecordVersion, bool, error) {
	if rec.Deleted {
		ver, _, err := store.Delete(key)
		if errors.Is(err, storage.ErrNotFound) {
			return version.RecordVersion{}, false, nil
		}
		...
	}
	ver, _, err := store.Set(key, rec.Data)
	if errors.Is(err, storage.ErrValueTooLarge) {
		return version.RecordVersion{}, false, nil
	}
	...
}
```

**Deterministic sorted pass** — the `slices.Sorted(maps.Keys(...))` idiom here is the one to keep for the union pass.

**Store.Delete no-version-burn semantics** (`internal/app/storage/store.go:293-304`): a missing/already-tombstoned key is `ErrNotFound`, writes nothing.

**Apply:**
- Compute `current := e.store.Records()` once; `restoreChanges` also returns true when any current **live** key is absent from the snapshot (extra current tombstones are not a change).
- Build the sorted union of snapshot keys + current live keys absent from the snapshot; snapshot key → `replayRecord`; current-live-only key → `store.Delete` + append `RestoredRecord{Deleted:true, Data:nil}` (a concurrent `ErrNotFound` is a benign skip).
- One `markDirty()` at the end (and on the mid-replay error path, as today); `Restore`'s flush and the handler's notify list stay unchanged.
- Rewrite the three additive-pinning tests listed under "Modified tests".

---

### `internal/app/persist/loader.go` (service, file-I/O)

**Analog:** itself, `Open` + `mergeRecords` (`internal/app/persist/loader.go:34-73`, `100-104`):

```go
func Open(
	nodeID uint16,
	now func() time.Time,
	driftWarn time.Duration,
	dumps port.DumpStore,
	storeCfg storage.Config,
	engineCfg Config,
) (eng *Engine, discarded int) {
	...
	store := storage.NewStore(nodeID, clk, storeCfg)
	mergeRecords(store, records)
	eng = New(store, dumps, engineCfg)
	...
}

func mergeRecords(store *storage.Store, records map[string]storage.Record) {
	for key, rec := range records {
		_, _, _ = store.Merge(key, rec)
	}
}
```

**Apply:**
- Wrap the caller's `storeCfg.OnChange` in a synchronous `loading` guard around `mergeRecords` (single goroutine, no race), so a startup dump announces nothing and cannot fill the hub's pending queue (Pitfall 8).
- Initialize the engine's confirmed watermark to the post-load store generation (`store.Generation()` after the merge) so no loaded event is ever pending.
- Keep the `ValueLimit()`/decode-before-store ordering untouched.

---

### `cmd/sup-kv/main.go` (composition root/config, request-response)

**Analog:** itself.

**ServerOptions wiring** (`cmd/sup-kv/main.go:714-728`):

```go
	srv := grpcapi.NewServer(engine, adapter.NotifyLocalWrite, hub, grpcapi.ServerOptions{
		SyncArrived:    adapter.SyncArrived,
		ListedKey:      adapter.ListedKey,
		KnownPeers:     adapter.KnownPeers,
		WaitReplicated: adapter.WaitReplicated,
		ReplayNotify:   adapter.NotifyReplayedWrite,
		StartupGate:    gateBackstopFor(cfg, engine),
		WaitTimeout:    cfg.waitTimeout,
	})

	apiCtx, apiCancel := context.WithCancel(context.Background())
	go srv.RunGate(apiCtx)
	go func() { _ = hub.Run(apiCtx) }()
```

**Server construction / registration** (`cmd/sup-kv/main.go:854-864`):

```go
func listenGRPC(grpcAddr string, srv *grpcapi.Server) (net.Listener, *grpc.Server, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen tcp %s: %w", grpcAddr, err)
	}
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	reflection.Register(gs)
	return listener, gs, nil
}
```

**Shutdown ordering** (`cmd/sup-kv/main.go:923-933`):

```go
func shutdown(node *wired, sig <-chan os.Signal) error {
	node.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	if node.metricsSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		err := node.metricsSrv.Shutdown(ctx)
		cancel()
		if err != nil {
			log.Printf("sup-kv: metrics listener shutdown: %v", err)
		}
	}
	node.apiCancel()
	...
```

**Engine tap wiring** (`cmd/sup-kv/main.go:575-589`): `OnChange: hub.Ingest` becomes the generation-carrying signature; add the `OnConfirmed` seam here.

**Apply:**
- `ServerOptions`: `WritePeers: adapter.KnownPeerIDs` (remove `KnownPeers` from the server options; keep the adapter method for metrics).
- `listenGRPC` takes `grpcapi.TransportLimits` (or `cfg.valueMaxBytes`) and calls `grpc.NewServer(limits.ServerOptions()...)`.
- `wired` carries `api *grpcapi.Server` (or the health server plus a shutdown func); `shutdown` calls `node.api.Shutdown()` **before** `apiCancel` and never `SetServingStatus` directly.
- Wire `OnConfirmed: hub.Confirm` in `persist.Config` inside `startEngine`.
- Keep the D-19 order otherwise byte-for-byte (health shed → metrics → API cancel → GracefulStop → repl → Flush → engine cancel).

---

### `internal/app/internal/simnet/simnet.go` (test-support driver, event-driven)

**Analog:** itself, the production-author sequence (`simnet.go:764-790`, abridged):

```go
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
	c.checkIndex(i, "Write")
	now := c.sched.Now()
	_, _, err := c.nodes[i].store.Set(key, data)
	if err != nil {
		panic(fmt.Sprintf("simnet: Cluster.Write: Set %q on node %d: %v", key, i, err))
	}
	rec, ok := c.nodes[i].store.Lookup(key)
	if !ok {
		panic(...)
	}
	var out port.Outbox
	c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
	c.route(i, out.Drain())
	out.DrainResults()
	c.drive(i, now)
	...
}
```

The other two call sites are `simnet.go:824` (Delete) and `simnet.go:854` (replay/recovery path).

**Apply:** pass a **sorted** peer-id slice captured at virtual write time (`core.PeerIDs()`), so same-seed replay stays byte-identical. Do not derive it from map iteration.

---

### New tests

#### `internal/grpcapi/errors_test.go`

**Analogs:**
- Real-adapter harness: `internal/grpcapi/gate_test.go:323-338`:

```go
func newFileServer(t *testing.T) *testHarness {
	t.Helper()
	h := buildTestServer(t, testServerConfig{
		dumps: persistfile.New(filepath.Join(t.TempDir(), "state.json")),
		gate:  openGateOptions(),
	})
	awaitGateOpen(t, h)
	return h
}
```

- Sized-facts table voice: `internal/udprepl/maxdatagram_test.go:24-38` (named constants + doc explaining the oracle).
- Non-parallel slog capture: `cmd/sup-kv/shutdown_internal_test.go:24-38`:

```go
//nolint:paralleltest // the subtests swap the process-global slog/log writers, so they must run sequentially
func TestAwaitEngineStop(t *testing.T) {
	var slogBuf bytes.Buffer
	oldSlog := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&slogBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(oldSlog) })
	...
}
```

**Apply:** one table test over the error matrix (`errors.Is` for each sentinel/class + `status.Code(want)`), one RPC-matrix test with a fake `DumpStore` returning `&fs.PathError{Path: filepath.Join(t.TempDir(),"secret"), Err: fs.ErrNotExist}` asserting every status message is path-free (`t.TempDir()`, `"/"`-rooted, `\`), and one non-parallel test asserting the detail **is** logged while the returned message is the constant.

#### `internal/grpcapi/limits_test.go`

**Analogs:** `maxdatagram_test.go:24-38` (typed constant table), `gate_test.go:289-321` (health client over bufconn), `server_test.go:294-377` (harness + `grpc.NewClient`).

**Apply:** a `LimitsFor` table (default value limit, max configured limit, keepalive constants), a bufconn over-limit unary request asserting `ResourceExhausted` (transport, not the handler's `InvalidArgument`), a marshal-size test for the worst-case List page/record, and a `Shutdown`-then-backstop test asserting `NOT_SERVING` after the late `openGate` (Pitfall 9).

#### `api/supkvapi/v1/schema_test.go`

**No analog.** There is no `protoreflect`/`protodesc` test in the repo. The nearest behavioral pin is `server_test.go`'s roundtrips. Write a descriptor test that walks `File_supkvapi_v1_supkvapi_proto` (service name, method set, streaming flags, field numbers/types of `Record`, `ListRequest`, `ListResponse`) so the wire cannot drift silently. This is the one file the planner should design from the RESEARCH recommendation, not copy.

### Modified tests

#### `internal/grpcapi/wait_test.go`

**Analog:** itself, the scripted tracker fake (`wait_test.go:30-90`):

```go
type waitCall struct {
	key     string
	ver     version.RecordVersion
	total   uint32
	timeout time.Duration
}
...
type fakeTracker struct {
	mu    sync.Mutex
	peers uint64
	calls []waitCall
	waits map[string]*fakeWait
}

func (f *fakeTracker) KnownPeers() uint64 { ... }
```

**Apply:** replace `KnownPeers` with `WritePeers func() []uint16`; add cases for zero peers → `(false,0,0)`, nil seam → `(false,0,0)`, write-time set pinned (a later `WritePeers` change cannot alter the report), and the total==0 release still calls the seam. Keep the existing generous elapsed-bound style (`waitAssertBound`).

#### `internal/grpcapi/watch_test.go` / `watch_senderr_test.go`

**Analog:** themselves; the flood harness at `watch_test.go:475-540` reaches `hub.Ingest` directly, so the signature change is mechanical. Add: unconfirmed write is not announced; confirmed write announced exactly once; dump-failed-then-confirmed announced once; deletions are events; shutdown ends the stream `Unavailable`; overflow clears pending and forces resubscribe.

#### `internal/grpcapi/gate_test.go`

**Analog:** `TestHealth_reflectsGate` (`gate_test.go:289-321`). Add `TestHealth_shutdownIsOneWay`: call `srv.Shutdown()` before the backstop fires, then assert `NOT_SERVING` persists after the resolver would have opened (Pitfall 9).

#### `internal/grpcapi/server_test.go`

**Analog:** `TestServer_list` (`server_test.go:656-707`) and the snapshot totality family (`1074-1219`). Rewrite `TestServer_list` for page size/token/prefix/tombstone/end-of-list; add a rollback/broadcast test seeding an extra live key and asserting `Get → NotFound` plus a `Deleted=true` replay notification (including under a failed final dump).

#### `internal/app/storage/store_test.go` / `view_test.go`

**Analog:** `changeTapRecorder` (`store_test.go:905-951`):

```go
type changeTapRecorder struct {
	mu     sync.Mutex
	events []changeTapEvent
}

func (r *changeTapRecorder) tap(key string, rec storage.Record) { ... }
```

and `TestStore_versions_snapshot` (`view_test.go:143-200`) for the `LiveVersions` contract (tombstone-free, caller-owned, repeats equal, no value copies).

**Apply:** add generation to the recorder + assert “one per actual install, none for no-op/loss”; add `TestStore_liveVersions` proving a tombstoned key is absent and an empty live value is present.

#### `internal/app/persist/engine_test.go`

**Analog:** `TestEngine_snapshotCreateLoadRestore` (`engine_test.go:1104-1269`) — rewrite:

- `post-key` becomes deleted and appears in `restored` with `Deleted=true` and a version dominating the pre-restore max.
- `gone` stays a skip (already tombstoned in snapshot, no version burned).
- `re-tomb` / `resurrect` keep their assertions.
- Add the no-op-with-extra-live-key test (rollback must fire even when the snapshot otherwise matches).
- `TestEngine_restoreReturnsReplayed` (`~1599`) becomes alpha/beta/delta/gamma with gamma deleted.

#### `internal/udprepl/repl_test.go`

**Analog:** `TestWaitReplicated_ackCoverage` (`repl_test.go:487-532`) and `TestWaitReplicated_strangerAckCounts` (`534-584`) — the real-socket, public-seam wait style with `waitFor`/`startWait`/`awaitReport`. Extend with: an Ack from a post-write peer cannot resolve; a captured-but-gone peer does not block forever (`WaitSetEmpty`); superseded resolves well under the caller timeout (generous real-adapter bound, deterministic core assertion beside it).

#### `internal/app/repl/{retransmit_test.go,peertable_test.go,sim_*}`

**Analog:** `TestRetransmit_suspendAndFlush` (`retransmit_test.go:419`), `TestRetransmit_backoffLadder` (`312-419`) for the virtual-time ladder style. Update the discovery-flush assertions: a new peer still receives the record (`enqueueRecordSend`), still gets the ladder reset, but **does not** join `waitSet` or the ledger.

#### `cmd/sup-kv/main_test.go`

**Analog:** `TestShutdown_secondSignal` (`main_test.go:1037`) + `grpc.NewClient` health polling (`315-332`). Add a process-level test that `Shutdown()` before the gate backstop leaves health `NOT_SERVING`, and a keepalive/limit smoke only if the process can observe it cheaply (otherwise the `limits_test.go` unit test is the pin).

---

## Shared Patterns

### Error vocabulary and path-freedom

**Sources:** `internal/app/storage/errors.go:5-19`, `internal/app/persist/errors.go:7-25`, `internal/persistfile/snapshot.go:20-29`, `internal/persistfile/saveerror.go:32-46`, `internal/port/dump.go:127-157`.

All public errors are package-level sentinels matched with `errors.Is`; adapter errno knowledge stays behind `port.FailureClassOf`. The mapper must format **no** error into a status message and must log detail in the `OnDumpFailure` style (static message, snake_case attributes).

```go
func (c FailureClass) String() string {
	switch c {
	case FailureDiskFull:
		return "disk_full"
	case FailurePermission:
		return "permission"
	case FailureOther:
		return "other"
	}
	return "other"
}
```

### Confirmation / watermark seam

**Sources:** `internal/app/persist/confirm.go:184-222` (capture/confirm order) and `internal/app/persist/scheduler.go:201-238` (capture before `Records`, confirm after a successful save).

The invariant to preserve in new code: everything captured before the snapshot is covered by the dump; a write racing the capture is covered by the next dump. The Watch watermark generation must be read at the same position and published only after `confirmWaiters`.

### Generation-gated immutable publication

**Sources:** `internal/app/repl/core.go:2973-3001` (generation moves only on a real change), `internal/udprepl/repl.go:314-325` (one atomic pointer publication), `internal/udprepl/repl.go:1079-1125` (publish only when the generation moved), `internal/udprepl/repl.go:944-951` (one immutable value holding correlated fields).

Use this shape for the hub's `confirmed` watermark publication and the adapter's `KnownPeerIDs` mirror: correlated facts ride one pointer/value; clones happen at publication cadence, not per event.

### Single-owner event loop

**Sources:** `internal/grpcapi/watch.go:107-202` (hub owns subscriber state; tap only enqueues), `internal/udprepl/repl.go:560-591` (adapter loop is the only core driver), `internal/app/repl/core.go:742-870` (core fields are driver-goroutine-only).

Any new queue (Watch pending, tracker wants) is owned by exactly one goroutine; cross-goroutine reads go through atomics/copying accessors with the documented "mirrored after the loop iteration" language.

### Test harness: hub first, engine second

**Source:** `internal/grpcapi/server_test.go:278-377`.

```go
	hub := grpcapi.NewWatchHub()
	...
	eng, _ := persist.Open(
		42,
		time.Now,
		time.Minute,
		dumps,
		storage.Config{ValueMaxBytes: testValueLimit, OnChange: hub.Ingest},
		persist.Config{
			Debounce:       time.Millisecond,
			MaxWait:        50 * time.Millisecond,
			ConfirmTimeout: cfg.confirmTimeout,
		},
	)
	...
	api := grpcapi.NewServer(eng, cfg.notify, hub, opts)
	...
	lis := bufconn.Listen(1 << 20)
	...
	conn, err := grpc.NewClient(
		"passthrough:bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
```

The hub must exist before `persist.Open` threads the tap; the engine before `NewServer`; the resolver runs under its own context; everything stops via `t.Cleanup`. New tests copy this ordering, not a variant.

### Import alias discipline

All 19 importers alias the generated package as `apiproto`, so the public move changes only the quoted path:

```go
apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
```

After the move:

```go
apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
```

No identifier changes anywhere; no forwarding package.

### Doc-anchor guards

**Source:** `cmd/sup-kv/operations_test.go:21-110` — a doc path constant, an anchor table of (literal, pins) pairs, an `os.ReadFile`, and a structural assertion (first `## ` heading). `docs/API.md` should ship with the phase-owned anchors and a sibling guard only if the phase creates the test; otherwise leave the anchor test to Phase 9's DOCS work and state the split in the plan.

### Lint/config discipline

**Source:** `.golangci.yml:65-88` — project additions live only inside `linters.settings.depguard.rules`, after the byte-unmodified upstream banner; every rule has `files` with `$test` exemption, `pkg` + `desc`. The new `sup-kv-public-api-no-internal` rule copies this shape exactly. Any change to the vendored block above the banner is forbidden (AGENTS.md deviation 2). Every `//nolint` names its linter and carries a reason.

---

## Cross-Cutting Signature Ripple (exact call sites)

These are the mechanical fan-outs the planner must schedule together with the type changes:

**`storage.Config.OnChange` → `func(key string, rec Record, generation uint64)`**

- `cmd/sup-kv/main.go:579` (`OnChange: hub.Ingest`)
- `internal/grpcapi/server_test.go:318`
- `internal/grpcapi/bench_test.go:82`
- `internal/grpcapi/wait_integration_test.go:201, 337`
- `internal/grpcapi/watch_senderr_test.go:120`
- `internal/app/storage/store_test.go:924, 949`
- `internal/grpcapi/watch.go:213` (`Ingest` signature) and its direct callers/flood tests (`watch_test.go:475-540`)

**`NotifyLocalWrite` / `LocalNotify` gains the peer set**

- `internal/grpcapi/server.go:47` (type), `:373-384` (`notifyLocalWrite`), `:219, 223, 267, 271` (call sites)
- `internal/grpcapi/server_test.go:148-171` (`notifyCall`/`notifyCapture`), `:219`
- `internal/grpcapi/bench_test.go:196`
- `internal/udprepl/repl.go:623`
- `internal/app/repl/core.go:2169, 2194, 2208`
- `internal/app/internal/simnet/simnet.go:786, 824, 854`
- `cmd/sup-kv/main.go:716` (method value)

**`ServerOptions.KnownPeers` → `WritePeers`**

- `internal/grpcapi/gate.go:72-78`
- `internal/grpcapi/wait.go:70-81`
- `internal/grpcapi/wait_test.go:70-90` (`fakeTracker`)
- `cmd/sup-kv/main.go:719`
- Any harness `testServerConfig.knownPeers` (`server_test.go:231`)

**`persist.Config.OnConfirmed`**

- `internal/app/persist/engine.go:26-49` (field), `:56-73` (normalized)
- `internal/app/persist/scheduler.go:221-238` (call on success)
- `cmd/sup-kv/main.go:580-589` (wiring)
- `internal/grpcapi/server_test.go:319-323` and every harness that expects Watch release (`bench_test.go:82`, `wait_integration_test.go:201,337`, `watch_senderr_test.go:120`)

**Import move** — the 19 files listed above plus the two frozen tests (one line each) and the historical evidence files (do not edit).

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `api/supkvapi/v1/schema_test.go` | test | — | No `protoreflect`/`protodesc` descriptor test exists anywhere; write from the RESEARCH descriptor-pin recommendation |
| `examples/client/` as a separate module | client/example | request-response + streaming | The repo has no nested Go module; `internal/stand/stand.go:485-512` is only a transport-setup analog, and `.github/workflows/ci.yml` has no nested-module job |
| `docs/API.md` | doc | — | File does not exist; `docs/OPERATIONS.md` and `docs/04-MULTI-NODE.md` supply voice/structure, not content |
| List page tokens/byte budget | controller | request-response | No pagination exists anywhere; the adjacent patterns are `Store.Versions`/`Lookup` and the sorted-key idiom |

---

## Metadata

**Analog search scope:** `internal/grpcapi/`, `internal/app/{storage,persist,repl}/`, `internal/udprepl/`, `internal/persistfile/`, `internal/port/`, `internal/stand/`, `internal/app/internal/simnet/`, `cmd/sup-kv/`, `docs/`, `.github/workflows/`, repo root config (`Makefile`, `buf*.yaml`, `.golangci.yml`).
**Files read in this session (analog extraction):** 40+; every excerpt above was read this session from the named file and line range.
**Pattern extraction date:** 2026-11-22.
**Handoff:** the planner should cite this file per task. Reference `docs/SPEC-DEVIATIONS.md` rows by ID (S-PERS-12, S-ALG-CU-11, S-SEM-04, S-IMP-01…03), not by line number — the file's line numbers are already stale relative to current source.
