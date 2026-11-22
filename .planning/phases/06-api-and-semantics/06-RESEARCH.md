# Phase 6: API and Semantics - Research

**Researched:** 2026-11-22
**Domain:** Go gRPC API semantics, durability-gated watch feed, keyset pagination, total error mapping, public wire contract
**Confidence:** HIGH for the current-state map (every claim read from source this session or probed in a throwaway clone); MEDIUM for the design recommendations (the constants and shapes are discretion, not facts)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** `wait_for_replication` reports the write-time peer set: the set of peers known when the write was accepted is captured, zero known peers returns `(false, 0, 0)` immediately, and an `Ack` from a node that was not in that set never yields `true`. A displaced version resolves within 20 ms through the existing `Superseded` Outbox result (Phases 3/5), not by idling to the caller's timeout. The wait stays caller-bounded and its contract is documented. — [auto] [Honest wait] Q: what does honesty mean here? → Selected: write-time peer set + immediate zero-peer answer + Superseded fast path (A-01). — **Reversibility:** costly — public RPC semantics.
- **D-02:** `List` becomes keyset-paginated over the sorted key→version view: an opaque page token (empty = first page), bounded default and maximum page sizes, and per-page cost independent of value sizes — the request never copies the whole store. Values ride only for the entries of the page. The semantics (ordering, token stability, end-of-list signal) are documented. — [auto] [List] Q: paginate or stream? → Selected: keyset pagination with bounded pages (A-02). — **Reversibility:** costly — public RPC shape.
- **D-03:** Error mapping is total and path-free: one mapping table covers every internal error; a missing snapshot maps to `NotFound`, a request needing a dump with no dump configured maps to `FailedPrecondition`, and no message carries a server filesystem path. Internal detail is logged, never returned. A test walks the RPC matrix and rejects path shapes in every status message. — [auto] [Errors] Q: how is totality proven? → Selected: exhaustive mapping table + path-free message test (A-03). — **Reversibility:** reversible.
- **D-04:** The Watch contract: local changes are announced only after the dump confirmation that made them durable; each confirmed write yields exactly one event; deletions are events; ordering follows versions; server shutdown ends streams with `Unavailable`, never `OK`. The contract text lands in the API docs (Phase 9 completes the surrounding document). — [auto] [Watch] Q: when is an event announced? → Selected: after dump confirmation, exactly once, Unavailable on shutdown (A-04). — **Reversibility:** costly — consumer-visible ordering.
- **D-05:** gRPC server settings are deliberate and documented: explicit keepalive parameters, explicit max receive/send message sizes tied to the protocol constants, and a health state machine that goes `NOT_SERVING` when shutdown begins and never returns to `SERVING` afterwards. A test pins the health transition and the settings. — [auto] [Server settings] Q: which settings are pinned? → Selected: keepalive + message-size ceilings + one-way health transition (A-05). — **Reversibility:** reversible.
- **D-06:** The consumer-facing API proto and its generated Go code move outside `internal/` (a public package path), while the internal replication wire proto stays internal. A Go client example builds in CI as a separate module. The last-read-value file pattern is documented as consumer-library territory (14.7, accepted), not a server feature. — [auto] [Public proto] Q: what moves, what stays internal? → Selected: API proto public, replication proto internal, client example module in CI (A-06/14.7). — **Reversibility:** costly — import paths.
- **D-07:** Snapshot load is a full rollback (14.1): keys absent from the snapshot are deleted, the result broadcasts to peers (Phase 5 totality), and `TestEngine_snapshotCreateLoadRestore` is rewritten to prove the new behavior. The full-rollback semantics are documented. — [auto] [Rollback] Q: additive or rollback? → Selected: full rollback with broadcast (S-01/14.1). — **Reversibility:** costly — data semantics.

### Claude's Discretion
- Page-size defaults and maximum, token encoding, keepalive constants, the public package path, and the exact API-docs split between this phase and Phase 9.

### Deferred Ideas (OUT OF SCOPE)
- `docs/API.md` final wording and the 18-row failure table are Phase 9/DOCS-05; this phase authors the contract sections it owns.
- The metrics surface for Phase-4/5 counters and the value-limit library guard are Phase 7.
</user_constraints>

## Summary

The phase is mostly *semantics*, not new machinery. Three of the seven requirements are
already half-built by Phases 3-5 and need only a missing capture point and tests
(APIS-01), a one-way shutdown call and explicit server limits (APIS-05), and a
relocation plus one codegen pass (APIS-06). The real code work is APIS-04 (Watch events
must wait for the dump confirmation), APIS-03 (the error mapper currently formats
arbitrary errors into status messages and has no mapping for a missing snapshot or a
missing dump), APIS-02 (List copies every value through `Engine.Records()`), and APIS-07
(`Restore` replays only the snapshot's keys and a test pins the additive behavior).

Two findings shape the plan. First, `wait_for_replication` has **no 500 ms polling left**
— Phase 3 replaced the mirror with the Outbox result stream, and a superseded version
already resolves through `ResultSuperseded`; the remaining defects are the zero-peer
`acked=true` answer, a peer total read at *wait* time instead of write-acceptance time,
and the discovery flush extending the core's wait-set. Second, the Watch tap fires inside
`storage.Store.install`, i.e. before the dump that makes the record durable; the fix needs
a confirmation watermark from the persist engine and a hub-held pending queue, not a
handler-side notification (which would lose exactly-once for dump-failed writes).

**Primary recommendation:** capture the write-time peer set in the gRPC handler and pin it
through the notification into the core (zero peers → `acked=false 0/0`); relocate the API
proto to `api/supkvapi/v1` and add keyset page fields in the same codegen pass; rewrite
`mapError` as a total table with a constant `Internal` fallback plus adapter-side error
classes; gate Watch live events on a dump-confirmation generation watermark; make
`List` page over a new live-only key→version view with a byte budget; pin gRPC
keepalive/message limits and use the standard health server's one-way `Shutdown`; and
implement full-rollback `Restore` as one sorted union pass that returns rollback
tombstones for replication.

<phase_requirements>
## Phase Requirements

| ID | Description (REQUIREMENTS.md) | Research Support |
|----|------------------------------|------------------|
| APIS-01 | `wait_for_replication` reports honestly: zero known peers → `(false, 0, 0)`; an `Ack` from a node unknown at write time does not yield `true`; a displaced version answers within 20 ms instead of idling 500 ms | Current state map §APIS-01; Design D-01; Pitfalls 1-4; tests in `internal/grpcapi/wait_test.go`, `internal/udprepl/repl_test.go`, `internal/app/repl/retransmit_test.go` |
| APIS-02 | `List` is paginated or streaming and its cost does not depend on value sizes | §APIS-02; Design D-02; the new live-only view; tests in `internal/app/storage/store_test.go` and `internal/grpcapi/server_test.go` |
| APIS-03 | Error mapping is total and path-free: missing snapshot → `NotFound`; no dump → `FailedPrecondition`; no server paths in messages | §APIS-03; Design D-03; the mapping table; `internal/grpcapi/errors_test.go` (new) |
| APIS-04 | The Watch contract: local changes announced only after dump confirmation; shutdown ends streams with `Unavailable`, not `OK`; contract in the API docs | §APIS-04; Design D-04; the confirmation watermark; `internal/grpcapi/watch_test.go`, `cmd/sup-kv/main_test.go` |
| APIS-05 | gRPC server settings are deliberate; health never returns to `SERVING` after shutdown begins | §APIS-05; Design D-05; `internal/grpcapi/limits_test.go` (new), `internal/grpcapi/gate_test.go`, `cmd/sup-kv/main_test.go` |
| APIS-06 | Proto and generated Go code outside `internal/`; a Go client example builds in CI as a separate module; last-read-value pattern documented as consumer-library territory | §APIS-06; Design D-06; the move probe evidence; `api/supkvapi/v1`, `examples/client`, CI job |
| APIS-07 | Snapshot load is a full rollback: keys absent from the snapshot are deleted; the result broadcasts to peers; `TestEngine_snapshotCreateLoadRestore` rewritten | §APIS-07; Design D-07; `internal/app/persist/snapshot.go`, `internal/app/persist/engine_test.go`, `internal/grpcapi/server_test.go` |
</phase_requirements>

## Project Constraints (from AGENTS.md)

No `CLAUDE.md` and no `.claude/skills/` or `.agents/skills/` exist in this repo; the
project instructions live in `/home/nil/DiskD/W/Djarvur/sup-kv/AGENTS.md`. The directives
that bind this phase:

- **Go version:** as pinned in `go.mod` (`go 1.27`; the module says `go 1.27.4`). Do not
  raise it without an ADR.
- **Gate commands:** `make lint` (`go tool golangci-lint config verify` + `run ./...`) and
  `make test` (`go test -race -cover ./...`) before pushing; CI runs the same targets.
- **Depguard architecture boundary:** `internal/app` may not import `net`, `os`, `syscall`
  or `google.golang.org/*` (rule `sup-kv-app-no-io`); `internal/port` may import the
  standard library and the own module path only (`sup-kv-port-stdlib-only`). Any change is
  a visible `.golangci.yml` diff, never a quiet code-side workaround.
- **Vendored lint config provenance:** `.golangci.yml` is powerman/golangci-lint-strict
  for golangci-lint **exactly** 2.15.0, byte-unmodified behind its `# Origin:` header;
  project rules are additive keys inside `linters.settings.depguard.rules` only. A version
  bump means re-fetching the matching strict config.
- **No lint suppressions without a written justification:** every `//nolint` names its
  linter and carries a human-readable reason (`nolintlint.require-explanation` and
  `require-specific`).
- **Layout:** `cmd/sup-kv/` is flags/env/signals/wiring only; `internal/app/` is pure and
  goroutine-free; `internal/port/` is interface definitions, stdlib-only; no empty adapter
  directories.
- **Dependency policy:** zero direct runtime requires is the v1.1 default; gRPC/protobuf
  are already direct requires from v1.0 and stay pinned. Tools are invoked via
  `go tool <name>`, never `tools.go`, never a bare `go install`.
- **User global workflow:** feature work on a branch from an up-to-date master; this
  research writes exactly one file and commits only it.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Write-time peer capture and wait report | API/backend (`internal/grpcapi`) | Replication adapter (`internal/udprepl`) and core (`internal/app/repl`) | The API owns the client contract; the adapter owns the published peer mirror; the core owns the Ack ledger and the supersession result |
| Keyset pagination of `List` | API/backend (`internal/grpcapi`) | Storage view (`internal/app/storage`, `persist.Engine`) | The API owns page tokens and bounds; the store must offer a cheap live key→version view so no value is copied for the whole store |
| Total error mapping | API/backend (`internal/grpcapi`) | File adapter (`internal/persistfile`) for failure classes | The mapping is a wire contract; errno/FS knowledge stays in the adapter per the existing `FailureClass` pattern |
| Watch durability gate | Persistence engine (`internal/app/persist`) publishes the watermark; API hub (`internal/grpcapi`) holds events | Storage tap (`internal/app/storage`) tags install generation | Durability is the engine's fact; only the hub may decide what a subscriber sees |
| gRPC limits and health | API/backend (`internal/grpcapi` builder) | Binary wiring (`cmd/sup-kv`) | Limits are a function of the effective value limit; the standard health server owns the one-way state machine |
| Public proto and generated code | Public package (`api/supkvapi/v1`) | Buf config + CI | The wire schema is a consumer artifact; the server imports it like any other package |
| Snapshot full rollback | App core (`internal/app/persist`) | API handler (`internal/grpcapi` SnapshotLoad) | The rollback is a store/data decision; the handler only broadcasts the returned record list (PROTO-08) |

## Standard Stack

### Core

| Library / Tool | Version | Purpose | Why Standard |
|----------------|---------|---------|--------------|
| Go | `go 1.27.4` in `go.mod` | Language/runtime | Pinned by the project; do not raise |
| `google.golang.org/grpc` | `v1.85.0` `[VERIFIED: go.mod:16]` | gRPC server, health, keepalive options | Existing direct require; the only server framework in the repo |
| `google.golang.org/protobuf` | `v1.36.13-0.20261016172727-e9ee2a76ab8e` `[VERIFIED: go.mod:17]` | Generated message runtime | Existing direct require; generated code pins `protoimpl` |
| `github.com/bufbuild/buf` (tool) | `v1.75.0` `[VERIFIED: go tool buf --version]` | Proto lint/format/breaking/generate | Existing tool directive; the repo's only proto gate |
| `protoc-gen-go-grpc` (tool) | `v1.6.2` `[VERIFIED: internal/grpcapi/apiproto/supkvapi/v1/supkvapi_grpc.pb.go:11]` | Service codegen | Existing tool directive; generated code asserts `SupportPackageIsVersion9` |
| `protoc-gen-go` (tool) | `v1.36.12-devel` `[VERIFIED: internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go:10]` | Message codegen | Existing tool directive |
| `golangci-lint` (tool) | `v2.15.0` | Strict lint gate | Pinned to the vendored strict config (AGENTS.md) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `google.golang.org/grpc/health` | same module | Standard health server, one-way `Shutdown` | APIS-05 health state machine |
| `google.golang.org/grpc/keepalive` | same module | Server keepalive params and enforcement policy | APIS-05 explicit settings |
| `google.golang.org/grpc/reflection` | same module | Reflection (existing) | Keep registered; no change |
| `encoding/base64` (stdlib) | Go 1.27 | Opaque page-token encoding | APIS-02 token |
| `slices`, `maps`, `strings`, `bytes` (stdlib) | Go 1.27 | Sorting, prefix filters, union passes | Existing idiom |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Keyset pagination | Server-streaming `List` | D-02 chose keyset; streaming changes the RPC shape and the client example, and a stream still needs a stop rule. Rejected |
| Hand-written health state machine | `health.Server.Shutdown()` | The standard server already ignores all `SetServingStatus` calls after `Shutdown`; hand-rolling duplicates a tested implementation. Rejected |
| A public `api` submodule | One module + a public package | A submodule needs prefixed release tags and extra release machinery; the example module can `require` + `replace` the root. Rejected for v1.1 |
| Error-class methods on every adapter error | One `mapError` table + adapter-side `FailureClass` | The class vocabulary already exists (`port.FailureClassOf`); a second vocabulary would drift. Rejected |
| Handler-side post-`Set` Watch notification | Engine watermark + hub pending | Handler-side loses the dump-failed-then-confirmed event and per-key ordering against replicated changes. Rejected |

**Installation:** none. This phase adds no dependency to `go.mod`. The client example
module reuses the root's pinned `grpc`/`protobuf` versions; its `go mod tidy` probe
produced an 18-line `go.sum` `[VERIFIED: probe /tmp/bufprobe/examples/client/go.sum]`.

## Package Legitimacy Audit

Not applicable: this phase installs no external package. The only new module is
`examples/client`, which requires `github.com/Djarvur/sup-kv` (this repo) plus
`google.golang.org/grpc v1.85.0` and the protobuf version already pinned in the root
`go.mod`; both were already direct runtime requires.

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Current-State Map (verbatim, with line ranges)

> Every claim below was read from the named file in this session. Quoted values are
> verbatim. The probe evidence in §APIS-06 came from a throwaway `git clone --shared`
> under `/tmp/bufprobe`; the working tree was not modified.

### APIS-01 — `wait_for_replication` today

The handler captures the peer total **at wait time**, not at write acceptance, and both
the nil-seam and zero-peer paths answer `acked=true`:

`internal/grpcapi/wait.go:66-84` `[VERIFIED: internal/grpcapi/wait.go:66-84]`
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
The comment says "at write time", but `KnownPeers()` is read here, after the engine write
and after the dump confirmation; a peer discovered during that window is counted.

The tracker is created synchronously **before** the notification is enqueued, and it
counts every matching `Ack` (including a stranger's):

`internal/udprepl/repl.go:623-637` `[VERIFIED: internal/udprepl/repl.go:623-637]`
```go
func (a *Adapter) NotifyLocalWrite(key string, rec storage.Record, track bool) error {
	keyID := trackerKey{key: key, version: rec.Version}
	if track {
		a.ensureTracker(keyID)
	}
	err := a.enqueueLocalWrite(localWrite{key: key, rec: cloneLocalRecord(rec)})
```

`internal/udprepl/repl.go:736-753` `[VERIFIED: internal/udprepl/repl.go:736-753]`
```go
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
```

`internal/udprepl/repl.go:783-807` `[VERIFIED: internal/udprepl/repl.go:783-807]`
```go
	if total == 0 {
		a.deleteTracker(k)
		return true, 0
	}
	...
	count := min(uint32(len(tr.acked)), total) //nolint:gosec // G115: ...
	ok := tr.resolved && tr.ok && count == total
	...
	a.deleteTracker(k)
	return ok, count
```

The core pins the awaited set at **notification-processing** time from the current peer
table, and the discovery flush **adds a newly seen peer to every pending wait-set**:

`internal/app/repl/core.go:2226-2245` `[VERIFIED: internal/app/repl/core.go:2226-2245]`
```go
	pk := pendingKey{key: key, version: rec.Version}
	_, tracked := c.pending[pk]
	if !tracked {
		waitSet := make(map[uint16]struct{}, len(c.peers))
		for id := range c.peers {
			waitSet[id] = struct{}{}
		}
```

`internal/app/repl/core.go:1397-1406` `[VERIFIED: internal/app/repl/core.go:1397-1406]`
```go
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

Drain requires an empty wait-set and a non-empty peer table:

`internal/app/repl/core.go:1567-1583` `[VERIFIED: internal/app/repl/core.go:1567-1583]`
```go
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
```

**What is already correct:** the 500 ms idle is gone. A superseded version resolves through
`ResultSuperseded` in three places: a strictly older local notification (core.go:2211-2218),
a newer local write under the same key (`removeSupersededLocal`, core.go:3236-3243), and a
strictly newer remote record (core.go:1523-1528). `Ack` from a stranger also does not by
itself yield `true`, because success needs `tr.ok` (only `ResultDrained` sets it) and
`ResultDrained` needs the core's wait-set empty. The real remaining defects are:
(1) zero peers and a nil seam answer `true`; (2) the reported total is read at wait time,
not acceptance time; (3) the discovery flush mutates the awaited set, which is the IN-05
limitation already documented in `docs/04-MULTI-NODE.md:110-115`:
> "Known limitation IN-05: a peer discovered after the write-time snapshot but before the core processes the notification joins the core wait set without joining `total`, so a write fully covered by the write-time peers can still report `false` at the timeout."

### APIS-02 — `List` today

The handler takes a full `Records()` snapshot (every value deep-copied) and then builds a
sorted key list:

`internal/grpcapi/server.go:292-306` `[VERIFIED: internal/grpcapi/server.go:292-306]`
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

`Engine.Records()` delegates to `Store.Records`, which clones every record's `Data`
(`internal/app/storage/store.go:441-452`) `[VERIFIED: internal/app/storage/store.go:441-452]`.
The wire has no pagination fields at all:

`internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:100-109` `[VERIFIED: internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:100-109]`
```proto
// ListRequest filters by prefix; the empty prefix means all keys (D-15).
message ListRequest {
  string prefix = 1;
}

// ListResponse carries the sorted live (non-tombstoned) keys only (D-13,
// D-14) in one response — no pagination fields exist (D-12).
message ListResponse {
  repeated string keys = 1;
}
```

The cheap view that exists today is `Store.Versions()`: it copies every key's version and
**no Data**, but it is tombstone-inclusive and carries no `Deleted` flag:

`internal/app/storage/store.go:238-246` `[VERIFIED: internal/app/storage/store.go:238-246]`
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
`len(rec.Data) == 0` is not a tombstone test: a live record may hold an empty value, and
`Delete` writes a tombstone with empty data. A live-only projection is therefore needed.

### APIS-03 — Error mapping today

The mapper has seven `errors.Is` cases and a `%v` fallback:

`internal/grpcapi/server.go:433-454` `[VERIFIED: internal/grpcapi/server.go:433-454]`
```go
	case errors.Is(err, persistfile.ErrInvalidSnapshotName):
		return status.Error(codes.InvalidArgument, persistfile.ErrInvalidSnapshotName.Error())
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Errorf(codes.Internal, "internal error: %v", err)
```

Every path-bearing error lands in that fallback:

- A missing snapshot: `ReadSnapshot` returns the raw `os.ReadFile` error, a
  `*fs.PathError` whose `Error()` embeds the configured path:
  `internal/persistfile/snapshot.go:67-74` `[VERIFIED: internal/persistfile/snapshot.go:67-74]`
  ```go
  snapshotPath := filepath.Join(f.path+snapshotDirSuffix, name)
  return os.ReadFile(snapshotPath) //nolint:gosec // G304: ...
  ```
  Result today: `codes.Internal` with the path in the message. A-03's exact finding.
- No dump to copy: `ErrNoDump` is a sentinel but has **no** mapper case:
  `internal/persistfile/snapshot.go:20-29` `[VERIFIED: internal/persistfile/snapshot.go:20-29]`
  ```go
  ErrNoDump = errors.New("no dump on disk to snapshot")
  ...
  ErrInvalidSnapshotName = errors.New("invalid snapshot name")
  ```
  Result today: `codes.Internal`, message path-free but the wrong code; the acceptance
  wants `FailedPrecondition`.
- `ensureDir`/`saveAtomically` failures from `CreateSnapshot` can carry temp paths and
  are not classified (`CreateSnapshot` calls `saveAtomically` directly, unlike
  `SaveStream` which wraps with `classifySaveError` in
  `internal/persistfile/saveerror.go:32-46`).
- Decode/generation faults (`persist: dump header unreadable: ...`,
  `persist: dump generation incomplete: ...`) are path-free but become
  `Internal` with "internal error: ...".
- `context.Canceled`/`DeadlineExceeded` become `Internal`.
- A snapshot save that fails with ENOSPC/EACCES has a class available only if the adapter
  attaches one; `port.FailureClassOf` otherwise returns `FailureOther`
  (`internal/port/dump.go:149-157`).

The sentinel messages themselves are path-free
(`internal/app/storage/errors.go:5-19`, `internal/app/persist/errors.go:7-25`), and the
handler-authored messages are constants (`listedKeyNotArrivedMessage`, `gateMessage`,
`keyTooLongError`). The leak is the fallback, plus the missing classes.

### APIS-04 — Watch today

The change tap fires on **every install**, including a local write whose dump has not
happened yet and whose author will get `Unavailable`:

`internal/app/storage/store.go:425-436` `[VERIFIED: internal/app/storage/store.go:425-436]`
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

The hub forwards each ingested event immediately and does not know a dump exists:

`internal/grpcapi/watch.go:184-202` `[VERIFIED: internal/grpcapi/watch.go:184-202]`
```go
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
```

`internal/grpcapi/watch.go:213-219` `[VERIFIED: internal/grpcapi/watch.go:213-219]`
```go
func (h *WatchHub) Ingest(key string, rec storage.Record) {
	select {
	case h.ingress <- watchEvent{key: key, rec: rec}:
	default:
		h.overflow.Store(true)
	}
}
```

On shutdown the hub closes each subscriber's channel with **no** termination status, so the
handler returns `nil` and gRPC sends `OK`:

`internal/grpcapi/watch.go:347-355` `[VERIFIED: internal/grpcapi/watch.go:347-355]`
```go
// shutdownSubscribers closes every subscriber's channel with a nil
// termination status — the ctx-cancel path: the streams end cleanly while
// the process is shutting down around them.
func (h *WatchHub) shutdownSubscribers() {
	for sub := range h.subs {
		delete(h.subs, sub)
		close(sub.events)
	}
}
```

The handler's read of `sub.err` on close is already the right mechanism; only the value
must change:

`internal/grpcapi/watch.go:392-399` `[VERIFIED: internal/grpcapi/watch.go:392-399]`
```go
		case ev, open := <-got.sub.events:
			if !open {
				// The hub closed the channel: sub.err was written before
				// the close, and the zero-value receive happens after it,
				// so the read is synchronized by the close itself — the
				// resubscribe status on a disconnect, nil on hub shutdown.
				return got.sub.err
			}
```

The exact-cut floor (`sub.snapshot`, compared in `wants`) makes live events strictly
dominate the snapshot version; it does not currently track later live versions, but
install order is version order per key because the store only installs a strictly greater
version (or an equal-version tie-break winner). `watchIngress = 4096` bounds the ingress
and `watchBuffer = 256` the per-subscriber channel
(`internal/grpcapi/watch.go:18-35`).

The engine already knows when a dump succeeded and which waiters it covered, but it
publishes no confirmation fact:

`internal/app/persist/scheduler.go:221-238` `[VERIFIED: internal/app/persist/scheduler.go:221-238]`
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

`persist.Config` has no confirmation hook today; its callbacks are `OnDumpFailure` only
(`internal/app/persist/engine.go:26-49`). `LoadStats` is the startup load report
(`engine.go:137-183`) and is unrelated to Watch confirmation.

**Startup-load trap:** `persist.Open` rebuilds the store via `store.Merge` for every loaded
record (`internal/app/persist/loader.go:100-104`), and `install` fires the tap. With the hub
wired, a large startup dump would enqueue up to `watchIngress` events before `hub.Run`
starts, and a clean `Main` load (no first dump) would leave them pending forever — see
Pitfall 8.

### APIS-05 — gRPC server settings and health today

The production server is constructed with **no options**; gRPC v1.85 defaults apply:

`cmd/sup-kv/main.go:854-864` `[VERIFIED: cmd/sup-kv/main.go:854-864]`
```go
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	reflection.Register(gs)
```

The v1.85 defaults (read from the module cache this session)
`[VERIFIED: /home/nil/go/pkg/mod/google.golang.org/grpc@v1.85.0/server.go:59-62]`:
```go
	defaultServerMaxReceiveMessageSize = 1024 * 1024 * 4
	defaultServerMaxSendMessageSize    = math.MaxInt32
```
and `[VERIFIED: /home/nil/go/pkg/mod/google.golang.org/grpc@v1.85.0/internal/transport/defaults.go:35-40]`:
```go
	defaultServerKeepaliveTime    = 2 * time.Hour
	defaultServerKeepaliveTimeout = 20 * time.Second
	defaultKeepalivePolicyMinTime = 5 * time.Minute
```

Health starts `NOT_SERVING` and opens with the gate:

`internal/grpcapi/gate.go:102-107` `[VERIFIED: internal/grpcapi/gate.go:102-107]`
```go
	healthSrv := health.NewServer()
	// The health protocol defaults the whole-server status to SERVING; the
	// gate starts closed, so the binding begins at NOT_SERVING before any
	// client can observe the endpoint (API-06).
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
```

`openGate` is a CAS from false to true, and shutdown currently calls
`health.SetServingStatus` directly, **without** closing the gate resolver first:

`internal/grpcapi/gate.go:175-179` `[VERIFIED: internal/grpcapi/gate.go:175-179]`
```go
func (s *Server) openGate() {
	if s.gateOpen.CompareAndSwap(false, true) {
		s.healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	}
}
```

`cmd/sup-kv/main.go:923-933` `[VERIFIED: cmd/sup-kv/main.go:923-933]`
```go
func shutdown(node *wired, sig <-chan os.Signal) error {
	node.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	if node.metricsSrv != nil {
		...
	}
	node.apiCancel()
```
The race: shutdown sets `NOT_SERVING`, then `apiCancel` cancels `RunGate`; if the backstop
or a poll wins the select between those two statements, `openGate` flips the health back
to `SERVING`. The standard health server already solves this: its `Shutdown` sets
`s.shutdown = true` and `SetServingStatus` returns early afterwards
`[VERIFIED: /home/nil/go/pkg/mod/google.golang.org/grpc@v1.85.0/health/server.go:139-148,169-176]`:
```go
	if s.shutdown {
		logger.Infof("health: status changing for %s to %v is ignored because health service is shutdown", service, servingStatus)
		return
	}
...
func (s *Server) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = true
	for service := range s.statusMap {
		s.setServingStatusLocked(service, healthpb.HealthCheckResponse_NOT_SERVING)
	}
}
```
`Resume()` reverses it; the server must never call it.

### APIS-06 — Proto and generated code today

The API schema lives under `internal/` and its `go_package` points there:

`internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:10-14` `[VERIFIED: internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:10-14]`
```proto
syntax = "proto3";

package supkvapi.v1;

option go_package = "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1;apiproto";
```

Buf modules, codegen out and the Makefile all pin the internal path:

`buf.yaml:18-19` `[VERIFIED: buf.yaml:18-19]`
```yaml
  - path: internal/udprepl/replproto
  - path: internal/grpcapi/apiproto
```
`buf.gen.api.yaml:7-13` `[VERIFIED: buf.gen.api.yaml:7-13]`
```yaml
plugins:
  - local: ["go", "tool", "protoc-gen-go"]
    out: internal/grpcapi/apiproto
    opt: paths=source_relative
  - local: ["go", "tool", "protoc-gen-go-grpc"]
    out: internal/grpcapi/apiproto
    opt: paths=source_relative
```
`Makefile:98-100` `[VERIFIED: Makefile:98-100]`
```make
proto:
	go tool buf generate internal/udprepl/replproto --template buf.gen.repl.yaml
	go tool buf generate internal/grpcapi/apiproto --template buf.gen.api.yaml
```

Importers of the generated package today (from `grep -rl`, excluding `.planning/` and
`docs/reviews/` history): 19 Go files —
`cmd/sup-kv/main.go`, `cmd/sup-kv/main_test.go` (2),
`internal/grpcapi/{server.go,watch.go,bench_test.go,gate_test.go,red_gate_test.go,server_test.go,wait_integration_test.go,wait_test.go,watch_senderr_test.go,watch_test.go}` (10),
and `internal/stand/{stand.go,persist_test.go,stand_buffers_test.go,stand_delete_test.go,stand_metrics_test.go,stand_red_test.go,stand_test.go}` (7).
Every one imports it with the alias `apiproto`. Two further Go files in the historical
review evidence module carry the old path; they are never built and must not be edited.

The historical review evidence under `docs/reviews/2026-11-19-v1.0/` is a separate scratch
module (`module review-repro-scratch`) and is never built; those two Go files also contain
the old path but must be left alone (historical evidence).

### APIS-07 — Snapshot load today

`Restore` replays only the snapshot's keys and never touches absent keys:

`internal/app/persist/snapshot.go:125-150` `[VERIFIED: internal/app/persist/snapshot.go:125-150]`
```go
func (e *Engine) replaySnapshot(records map[string]storage.Record) ([]RestoredRecord, error) {
	restored := make([]RestoredRecord, 0, len(records))
	if !restoreChanges(e.store.Records(), records) {
		return restored, nil
	}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		ver, replayed, err := replayRecord(e.store, key, records[key])
		...
	}
	e.markDirty()
	return restored, nil
}
```

`restoreChanges` compares only the snapshot's keys and its doc says the opposite of a
rollback:

`internal/app/persist/snapshot.go:152-168` `[VERIFIED: internal/app/persist/snapshot.go:152-168]`
```go
// restoreChanges reports whether replaying the snapshot's records would
// change the store's current state: the snapshot differs from the store
// when any of its keys holds a different record — version, data or
// tombstone flag — than the store does now. Keys the snapshot does not
// carry never matter, exactly as the replay never touches them; ...
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

The frozen test's doc comment already *claims* 14.1 rollback, while its assertions pin
additive behavior — the exact contradiction S-01 describes:

`internal/app/persist/engine_test.go:1113-1116` `[VERIFIED: internal/app/persist/engine_test.go:1113-1116]`
```go
// Decision 14.1 (ADR 0001): the snapshot family rides the accepted rule that
// a snapshot load is a full rollback, so the named snapshot is an exact copy
// of the current confirmed state and Restore replays its keys under the same
// install discipline — never a partial merge into whatever the target holds.
```
`internal/app/persist/engine_test.go:1236-1245` `[VERIFIED: internal/app/persist/engine_test.go:1236-1245]`
```go
	// post-key: absent from the snapshot — its post-snapshot state stands.
	rec, err := eng.Get("post-key")
	if err != nil || string(rec.Data) != "post-value" {
		t.Errorf("post-restore Get(post-key) = (%q, %v), want the untouched post-snapshot value",
			rec.Data, err)
	}
	if got, want := restored["post-key"].Version, preRestore["post-key"].Version; got != want {
		t.Errorf("post-key version moved across the restore: %x then %x, want identical (absent keys untouched)",
			want[:], got[:])
	}
```

Other tests that pin the additive behavior and must be rewritten with it:
`TestEngine_restoreReturnsReplayed` (`engine_test.go:1599-1603`, "gamma is absent from the
snapshot: its current state stands"), and the RPC-level no-op/notify tests only because
their stores hold no extra live key (`TestServer_snapshotLoadNotifies`,
`TestServer_snapshotLoadUsesReplaySeam`, `TestServer_snapshotLoadReplayFallsBackToLive`,
`TestServer_snapshotLoadNoOpDoesNotNotify`). `TestEngine_restoreSkipsReplay` uses an empty
store and stays valid.

The handler already broadcasts exactly the returned list (PROTO-08 totality), including a
failed final dump:

`internal/grpcapi/server.go:341-360` `[VERIFIED: internal/grpcapi/server.go:341-360]`
```go
func (s *Server) SnapshotLoad(_ context.Context, req *apiproto.SnapshotLoadRequest) (*apiproto.SnapshotLoadResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	records, err := s.engine.Restore(req.GetName())
	// PROTO-08 totality (D-08, T-4-19): ...
	if !errors.Is(err, persist.ErrClosed) {
		for _, rec := range records {
			s.notifyReplayedWrite(rec)
		}
	}
	if err != nil {
		return nil, mapError(err)
	}
	return &apiproto.SnapshotLoadResponse{}, nil
}
```
Rollback tombstones appended to `Restore`'s return value therefore broadcast with zero
extra handler wiring.

## Design Options and Recommendations

### D-01 — Honest `wait_for_replication`

**Recommended shape: capture the write-time set in the gRPC handler; pin it in the core.**

1. Add `ServerOptions.WritePeers func() []uint16` (sorted; nil = zero peers).
   Production wiring: `adapter.KnownPeerIDs`. Capture it in `Set`/`Delete` immediately
   before the engine call (write acceptance), not at wait time.
2. Extend the notification seam to carry the captured ids:
   `LocalNotify func(key string, rec storage.Record, track bool, peers []uint16) error`.
   `Adapter.NotifyLocalWrite` clones the slice, stores it on the tracker (`total`,
   `wants map[uint16]struct{}`), and passes it through `localWrite` to the core.
3. `Core.trackLocalWrite` seeds the pending entry's `waitSet` from the **notification's**
   set, intersected with the current peer table (a captured peer already gone cannot Ack;
   dropping it avoids an unfulfillable entry). If the captured set was non-empty but the
   intersection is empty, emit `WaitSetEmpty` immediately so the waiter gets an honest
   false instead of a timeout.
4. The discovery flush (`core.go:1397-1406`) keeps the ladder reset, the
   `enqueueRecordSend` broadcast and the retry re-arm, but **stops** adding the new peer to
   `waitSet` and `ledgerAdd`. The record still reaches the new peer; the promise does not
   grow. This closes IN-05 and makes the write-time set exact.
5. `Adapter.WaitReplicated` drops the caller `total` (the tracker owns it) or keeps it for
   compatibility; success is `resolved && ok && count == total`, with `Ack`s counted only
   for `wants` members. `ResultDrained` still means full promise coverage; `Superseded` and
   `WaitSetEmpty` remain honest false.
6. `waitForReplication` uses the handler's `len(peers)` as `total`; zero peers (or nil
   seam) returns `(false, 0, 0)` immediately and still calls the total==0 release path so
   the notify-time tracker cannot leak.
7. `LocalNotify` is created before the enqueue; do not move it (Pitfall 6).

**Signature ripple (plan for it):** `LocalNotify`, `Adapter.NotifyLocalWrite`,
`Core.NotifyLocalWrite`, `Core.NotifyReplayedWrite` and `ServerOptions.WaitReplicated`
change shape. Every `LocalNotify` fake in `internal/grpcapi/*_test.go`, the
`internal/grpcapi/bench_test.go` direct call (`NotifyLocalWrite(key, rec, false)` at
line 196), `wait_integration_test.go` and the three `simnet` call sites
(`internal/app/internal/simnet/simnet.go`) must pass the captured set. The simulator must
pass **sorted** peer ids so same-seed replay stays deterministic.

**Rejected options:**
- *Capture at notification time in the adapter* — simpler, but a peer discovered during the
  dump-confirmation wait joins the set, which is exactly the IN-05 hole.
- *Have the core emit the set as an Outbox result* — the total would arrive asynchronously
  after `WaitReplicated` starts, reintroducing an appearance race.
- *Only filter Acks by membership, keep wait-time total* — fixes the report count but not
  the zero-peer `true` or the exact total.

**The 20 ms path needs no production change.** It is a regression guard: `ResultSuperseded`
already exists on all three displacement paths. Prove it deterministically at the core
(`Outbox` carries `ResultSuperseded` in the same handler call) and at the real adapter with
a generous wall-clock bound (see Pitfall 2).

**Proposed constants/types (discretion):** `WritePeers func() []uint16`; tracker fields
`total uint32`, `wants map[uint16]struct{}`; `Core.PeerIDs() []uint16` +
`Core.PeerGeneration() uint64`; adapter `peerIDs atomic.Pointer[[]uint16]` published in
`mirrorStats` only when the peer generation moves.

### D-02 — `List` keyset pagination

**Recommended shape: pages of records, byte-budgeted, over a live-only key→version view.**

Wire additions (additive, buf-breaking safe — probed):

```proto
message ListRequest {
  string prefix = 1;
  int32 page_size = 2;      // 0 = default; > max clamps
  string page_token = 3;    // opaque; empty = first page
}
message ListResponse {
  repeated string keys = 1;       // page keys, retained for shape compatibility
  string next_page_token = 2;     // empty = end of list
  repeated Record records = 3;    // page entries, values included, same order as keys
}
```

Why records and not keys-only: D-02's sentence "Values ride only for the entries of the
page" is most naturally read as a value-bearing page; it also removes the N+1 Get pattern
for consumers. The byte budget keeps the page's cost bounded and independent of the store's
total value bytes, satisfying A-02. A keys-only page is a valid fallback if the planner
judges the value clause vacuous, but it contradicts the plain reading.

Handler algorithm:

1. Gate first, as today.
2. Resolve `page_size`: `< 0` → `InvalidArgument`; `0` → default; `> max` → clamp (documented).
3. Decode `page_token`: `base64.RawURLEncoding`; reject a decode failure or a decoded key
   longer than `storage.MaxKeyBytes` with `InvalidArgument` ("invalid page token").
4. `live := s.engine.LiveVersions()` — a new store/engine method returning key→version for
   non-tombstoned records only, under `RLock`, copying no `Data`. Sort the matching keys
   (`strings.HasPrefix`, then `> token`), byte-wise ascending (the current `slices.Sort`
   order).
5. Walk the sorted keys until `page_size` entries or `listPageByteBudget` of copied value
   bytes is reached. `Lookup` each included key (copies only that page's `Data`); skip a
   key that raced into a tombstone, and keep walking. `records` and `keys` stay aligned.
6. `next_page_token` = base64 of the last *examined* key when more matches remain
   (including the case where every examined key raced away); empty at the end.
7. Document: ordering is byte-wise lexicographic; the token is a position, not a snapshot;
   keys added/deleted between pages may appear/disappear; the page is live-only and
   tombstone-free; `page_size` is an upper bound because of the byte budget; the end signal
   is an empty token.

Proposed constants (discretion, must be tied to the message-size ceilings):
`defaultListPageSize = 64`, `maxListPageSize = 256`, `listPageByteBudget = 6 << 20`,
`apiMaxSendBytes` computed to cover the budget plus envelope overhead, and a test that
marshals a synthetic worst-case page and asserts it fits.

**Rejected:** server-streaming `List` (D-02 chose keyset); a full `Records()` scan with a
window (still copies the world); `Versions()` + `len(Data)==0` tombstone detection (wrong).

### D-03 — Total, path-free error mapping

**Recommended mapping table** (check order matters):

| Error | Code | Message (constant) |
|-------|------|--------------------|
| `storage.ErrNotFound` | `NotFound` | `storage.ErrNotFound.Error()` ("key not found") |
| `storage.ErrAlreadyExists` | `AlreadyExists` | `storage.ErrAlreadyExists.Error()` |
| `storage.ErrValueTooLarge` | `InvalidArgument` | `storage.ErrValueTooLarge.Error()` |
| `persist.ErrDumpFailed` | `Unavailable` | `persist.ErrDumpFailed.Error()` |
| `persist.ErrUnconfirmed` | `Unavailable` | `persist.ErrUnconfirmed.Error()` |
| `persist.ErrClosed` | `Unavailable` | `persist.ErrClosed.Error()` |
| `persistfile.ErrInvalidSnapshotName` | `InvalidArgument` | `persistfile.ErrInvalidSnapshotName.Error()` |
| `persistfile.ErrNoDump` | `FailedPrecondition` | `persistfile.ErrNoDump.Error()` |
| `fs.ErrNotExist` (snapshot read) | `NotFound` | "snapshot not found" |
| `fs.ErrPermission` | `PermissionDenied` | "permission denied" |
| `port.FailureClassOf(err) == FailureDiskFull` (not `ErrDumpFailed`) | `ResourceExhausted` | "disk full" |
| `port.FailureClassOf(err) == FailurePermission` | `PermissionDenied` | "permission denied" |
| `context.Canceled` / `DeadlineExceeded` | `Canceled` / `DeadlineExceeded` | `status.FromContextError(err).Err()` |
| a pre-built `status` error | pass through | — |
| anything else | `Internal` | the constant "internal error" |

Implementation notes:
- Never format `err` into a status message. The `Internal` fallback logs
  `slog.Error("Unmapped RPC error", "detail", err.Error())` (static msg, capitalized,
  `detail` snake-case key) and returns the constant.
- `CreateSnapshot` should wrap its save errors with `classifySaveError` so ENOSPC/EACCES
  reach the table as classes (the `SaveStream` path already does).
- "No dump configured" is realized as `persistfile.ErrNoDump` because `--dump-path` is
  mandatory and `persist.Engine` always has a `DumpStore`; document that reading.
- Totality is structural: the constant fallback covers every error; the table test proves
  each named class.
- The path-shape test walks the RPC matrix with a fake `DumpStore` returning
  `&fs.PathError{Path: <the test's temp dir> + "/secret", Err: ...}` and asserts no message
  contains `t.TempDir()`, `"/"`-rooted path shapes, or `\` separators. It also captures slog
  and asserts the detail *is* logged.

### D-04 — Watch contract and confirmation gate

**Recommended shape: an engine-confirmed generation watermark + a hub-held pending queue.**

1. **Tag installs with their generation.** Change `storage.Config.OnChange` to
   `func(key string, rec Record, generation uint64)`; `install` already bumps
   `s.generation` immediately before firing, so pass the stored generation. No new counter.
2. **Publish the confirmation watermark.** In `runDump`, read
   `gen := s.store.Generation()` **before** `records := s.store.Records()` (safe: an install
   between the two reads is in the snapshot but is confirmed one dump later; an install
   before the read is in both). On success, store the watermark and call a new
   `persist.Config.OnConfirmed func(generation uint64)` after `confirmWaiters`. On failure,
   do nothing.
3. **Hold and release in the hub.** `Ingest` becomes `Ingest(key, rec, gen)`; the hub keeps
   a bounded `pending []event` in arrival order and a `confirmed` watermark fed by
   `hub.Confirm(gen)` (atomic max + a buffered(1) wake channel). On each `Run` iteration,
   dispatch every pending event whose `gen <= watermark`, in order. This yields
   exactly-once, install/version order, and the "after the dump confirmation" rule for
   **every** install (local and replicated — replicated records become durable in the same
   dump).
4. **Dump-failed-then-confirmed writes are covered.** A local write whose dump failed stays
   pending; a later successful dump releases it (it is now durable), exactly once.
5. **Startup load is suppressed.** `persist.Open` must wrap the caller's `OnChange` with a
   `loading` guard around `mergeRecords` (synchronous, no race) so a startup dump does not
   announce the whole loaded state or fill the pending queue. Initialize the engine's
   confirmed watermark to the post-load store generation.
6. **Shutdown ends streams `Unavailable`.** `shutdownSubscribers` writes
   `status.Error(codes.Unavailable, "watch hub is shutting down")` into `sub.err` before
   closing each channel (fresh status value per subscriber, matching the existing
   "single-use response payload" convention). The handler already returns `sub.err`.
7. **Snapshot semantics:** keep the subscribe snapshot as the current in-memory state (it
   is what `Get` sees) and document that live events are the durable change feed. Filtering
   the snapshot to confirmed records was considered and rejected: it needs the pending set
   at subscribe time, and overflow (which clears pending) would make the guarantee
   inconsistent anyway.
8. **Overflow:** keep the existing disconnect-all policy. A pending queue at
   `watchIngress` bounds memory (4096 record references — the store owns the bytes); on
   overflow disconnect every subscriber and clear pending; they resubscribe and re-snapshot
   (the documented no-holes recovery).
9. **Ordering:** dispatch order is install order, which is per-key version order because
   `install` only runs for a strictly greater version or an equal-version tie-break winner.
   The existing exact-cut floor suppresses an event already contained in a subscriber's
   snapshot.

**Signature ripple (plan for it):** the `Storage.Config.OnChange` signature change touches
`internal/app/storage/store_test.go`'s tap recorder, the `grpcapi` harness and its siblings
(`server_test.go`, `bench_test.go`, `wait_integration_test.go`, `watch_senderr_test.go`),
and `cmd/sup-kv/main.go`. `persist.Config.OnConfirmed` is new wiring in `main` and every
harness that expects Watch events to be released.

**Rejected:** handler-side post-`Set` notification (loses the dump-failed-then-confirmed
event and can order-invert against replicated changes); local-only delay with per-key
barriers (more machinery for a worse contract); announcing remote changes immediately
(breaks exactly-once when a delayed local event is superseded by a delivered remote one).

### D-05 — gRPC settings and one-way health

**Recommended shape: a testable limits struct plus the standard health server's `Shutdown`.**

```go
type TransportLimits struct {
	MaxRecvBytes     int
	MaxSendBytes     int
	KeepaliveTime    time.Duration
	KeepaliveTimeout time.Duration
	MinPingInterval  time.Duration
}

func LimitsFor(valueMaxBytes int) TransportLimits
func (l TransportLimits) ServerOptions() []grpc.ServerOption
```

- `MaxRecvBytes` = `storage.MaxKeyBytes + valueMaxBytes + apiEnvelopeOverheadBytes`
  (covers the largest `SetRequest`; the effective `valueMaxBytes` comes from `config`, not
  the default constant — a node may run up to the datagram ceiling).
- `MaxSendBytes` = `listPageByteBudget + apiEnvelopeOverheadBytes` (covers a max page and
  every single-record response).
- Keepalive: server `Time: 2 * time.Minute`, `Timeout: 20 * time.Second`; enforcement
  `MinTime: 30 * time.Second`, `PermitWithoutStream: true` (consumers may run a 30 s client
  keepalive and may hold an idle connection between writes). Rationale documented in code
  and `docs/API.md`.
- `cmd/sup-kv.listenGRPC` takes the limits (from `cfg.valueMaxBytes`) and calls
  `grpc.NewServer(limits.ServerOptions()...)`; registration stays as is.
- **Health:** add `func (s *Server) Shutdown()` that calls `s.healthSrv.Shutdown()` (one-way
  by the health server's own implementation) and is idempotent. Change `wired.health` to
  `wired.api *grpcapi.Server` (or add both) so `shutdown()` calls `node.api.Shutdown()`
  instead of `SetServingStatus`. Never call `Resume`.
- Tests: a `LimitsFor` table (recv/send values for the default and max value limits;
  keepalive constants), a behavioral over-limit unary request → `ResourceExhausted`
  (distinct from the handler's `InvalidArgument`), a marshal-size test for the worst-case
  page/record, and a health test that calls `Shutdown` before the backstop fires and then
  observes `NOT_SERVING` after the backstop (the late `openGate` is ignored).

### D-06 — Public proto and client example

**Recommended shape: `api/supkvapi/v1` in the same Go module, package name `supkvapi`,
internal call sites keep the `apiproto` alias.**

- Module root `api`, proto at `api/supkvapi/v1/supkvapi.proto`, so `buf`'s
  `paths=source_relative` source path stays `supkvapi/v1/supkvapi.proto` and only the
  `go_package` option, the generated `package` clause and the leading comments change.
- `option go_package = "github.com/Djarvur/sup-kv/api/supkvapi/v1;supkvapi";`
- `buf.yaml`: replace `internal/grpcapi/apiproto` with `api`; keep
  `internal/udprepl/replproto`.
- `buf.gen.api.yaml`: `out: api` for both plugins.
- `Makefile`: `go tool buf generate api --template buf.gen.api.yaml`.
- Import churn: the 19 files above, one import line each; keep the alias
  `apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"` so no identifier changes. The
  probe changed 26 paths — 3 renames (the `.proto` and its two generated files), 21 Go
  files (the 19 real importers plus the two historical evidence files) and 2 buf configs —
  with `31 insertions(+), 30 deletions(-)` before the Makefile/docs edits, and built green.
- **buf breaking:** a pure move (same `go_package`) passes (exit 0 — the module-relative
  path is identical). Changing `go_package` trips `FILE_SAME_GO_PACKAGE`
  (probe output: `File option "go_package" changed from "...internal..." to "...api..."`).
  Add a narrowly scoped waiver:
  ```yaml
  breaking:
    use:
      - FILE
    ignore_only:
      FILE_SAME_GO_PACKAGE:
        - api/supkvapi/v1/supkvapi.proto
  ```
  Keep it after the move (a future go_package change on the public file is still reviewed
  by a human diff); record why in the buf.yaml comment.
- **Generated code:** regenerate once after the move + List fields + comment rewrite.
  "Byte-identical, only relocated" cannot be literally true: the `go_package` option is
  embedded in the descriptor and the generated package clause changes. The **wire schema**
  (messages, field numbers, service methods, streaming) is unchanged; pin it with a
  descriptor test in the public package instead of a byte diff.
- **Public package hygiene:** add `api/supkvapi/v1/doc.go` with a package comment (the
  generated files are lint-excluded; a hand-written doc file states the contract), and an
  additive depguard rule:
  ```yaml
  sup-kv-public-api-no-internal:
    files:
      - '**/api/**'
      - '!$test'
    deny:
      - pkg: github.com/Djarvur/sup-kv/internal
        desc: the public API package must remain importable by other modules
  ```
- **Client example:** `examples/client/{go.mod,main.go,README.md}` as a separate module
  `github.com/Djarvur/sup-kv/examples/client`, `require github.com/Djarvur/sup-kv v0.0.0` +
  `replace github.com/Djarvur/sup-kv => ../..`, plus the pinned `grpc`/`protobuf`. The probe
  built it with an 18-line `go.sum` and `go list ./...` at the root excludes it. Add a
  `client-example` Makefile target and a CI job (build + vet + `go mod tidy` +
  scoped `git diff --exit-code`). The example should demonstrate Set-with-wait, Get,
  a List page loop, and one Watch event; it must NOT ship a last-read-value cache — that
  pattern is documented in `docs/API.md` as consumer-library territory (ADR 0007).
- **Docs ripple:** `docs/SPEC-DEVIATIONS.md` lines 116, 180, 249-251 (old proto paths);
  `docs/EXAMPLES.md` List output (records/token); `AGENTS.md` Layout (add `api/supkvapi/v1/`);
  `buf.yaml`/`buf.gen.api.yaml` comments. Do **not** edit `docs/reviews/**` history.
- **Frozen tests:** `internal/grpcapi/red_gate_test.go` and `internal/stand/stand_red_test.go`
  (R-03/P-01 stand tests) import the proto, so each needs the one-line import edit. Their
  names, bodies and the red-runner rows stay byte-identical otherwise. Record this as the
  disclosed exception in the phase evidence and gate it with a diff that shows only the
  import line.

### D-07 — Snapshot full rollback

**Recommended shape: one sorted union pass that replays the snapshot and deletes absent
live keys.**

1. `replaySnapshot` computes `current := e.store.Records()` once (needed anyway for the
   no-op check).
2. `restoreChanges` becomes: true when any snapshot record differs from the current one
   **or** any current live (`!rec.Deleted`) key is absent from the snapshot. Extra current
   tombstones absent from the snapshot are not a change (they are already deleted), so the
   no-op contract stays meaningful.
3. If changed, build the sorted union of snapshot keys and current live keys absent from
   the snapshot. One pass:
   - key in the snapshot → `replayRecord` (existing Set/Delete rules, skips included);
   - key only in current and live → `store.Delete` (fresh tombstone), append a
     `RestoredRecord{Key, Version, Data: nil, Deleted: true}`; a concurrent `ErrNotFound`
     is a benign skip.
   This gives one deterministic sorted notify order for replay and rollback records.
4. `markDirty` once at the end (and on the mid-replay error path, as today); the final
   flush is unchanged. The returned list (including rollback tombstones) is what
   `SnapshotLoad` broadcasts — no handler change.
5. Semantics to document:
   - keys absent from the snapshot are deleted; keys already tombstoned locally stay
     tombstoned (no version burned, no new record);
   - a snapshot equal to current state *and* no extra live keys is the no-op (empty list,
     no dump);
   - concurrent writes after the rollback set is computed are ordinary LWW; this is not a
     linearizable snapshot isolation guarantee;
   - fresh HLC versions can collide only after a lost-disk/clock-rollback; ADR 0012's
     remove-wins law makes the rollback tombstone win an equal-version tie, which is the
     desired direction;
   - the load replicates by PROTO-08 totality, including after a failed final dump.
6. Rewrite `TestEngine_snapshotCreateLoadRestore`: `post-key` becomes deleted and appears in
   `restored` with `Deleted=true` and a version dominating the pre-restore max; `gone`
   stays a skip (snapshot tombstone on an already-tombstoned key, no version burned);
   `re-tomb` and `resurrect` keep their existing assertions. Update
   `TestEngine_restoreReturnsReplayed` (gamma deleted, list becomes
   alpha/beta/delta/gamma) and add a no-op test where an extra live key makes the restore
   non-no-op. Add an RPC-level rollback/broadcast test: seed an extra live key, load a
   snapshot without it, assert `Get` → `NotFound` and the replay seam receives a
   `Deleted=true` record for it (including under a failed final dump).
7. Registry: flip `docs/SPEC-DEVIATIONS.md` row S-PERS-12 from
   `fixed-in-v1.1(APIS-07)` to `implemented` with the new code/test links (Phase 9
   re-audits the whole registry).

## Architecture Patterns

### System Architecture Diagram

```
Client
  │ gRPC (api/supkvapi/v1)
  ▼
┌──────────────────────── internal/grpcapi ─────────────────────────┐
│ Server                                                            │
│  Set/Delete: capture peers (WritePeers) → engine.Set              │
│       ├─ success: notifyLocalWrite(peers) → replication adapter   │
│       └─ wait: waitForReplication(total=len(peers)) ──┐           │
│  List: Engine.LiveVersions → sort/filter → Lookup page │           │
│  Watch: hub.subscribe → snapshot (Engine.Records)      │           │
│           ▲                                            │           │
└───────────┼────────────────────────────────────────────┼──────────┘
            │ Ingest(key, rec, gen)                      │ WaitReplicated
            │                                            ▼
┌───────────┴────────── internal/app/storage ──┐  ┌──── internal/udprepl ──────────────┐
│ Store.install: gen++ → Config.OnChange tap   │  │ Adapter event loop                 │
│  (local Set/Delete, remote Merge, replay)    │  │  trackLocalWrite(promise set)      │
└───────────┬──────────────────────────────────┘  │  Outbox results → tracker          │
            │                                     └────────────────────────────────────┘
            │ (the same installs feed the dump)
            ▼
┌──────────────── internal/app/persist ─────────────────┐
│ scheduler.runDump: gen := Generation(); Records()     │
│ success → confirmWaiters + OnConfirmed(gen) ──────────┼──► hub.Confirm(gen)
└───────────────────────────────────────────────────────┘
```

Reading the diagram: a write enters at the API, the engine confirms it on disk, the
confirmation watermark reaches the Watch hub, and only then is the event released. The
replication path is orthogonal: it observes Acks against the promise set captured at the
write's acceptance, and the wait reports exactly that set.

### Recommended Project Structure (delta only)

```
api/
└── supkvapi/v1/          # NEW public wire package
    ├── supkvapi.proto    # moved from internal/grpcapi/apiproto/supkvapi/v1
    ├── supkvapi.pb.go    # regenerated (package supkvapi)
    ├── supkvapi_grpc.pb.go
    ├── doc.go            # package contract, hand-written
    └── schema_test.go    # descriptor pin (service, methods, field numbers)
examples/
└── client/               # NEW separate Go module (require + replace ../..)
    ├── go.mod
    ├── go.sum
    └── main.go
internal/grpcapi/
    ├── server.go         # mapError rewrite, List pagination, WritePeers capture
    ├── wait.go           # write-time total, zero-peer false
    ├── watch.go          # pending queue + Confirm + Unavailable shutdown
    ├── gate.go           # ServerOptions.WritePeers, Server.Shutdown
    └── limits.go         # NEW TransportLimits/LimitsFor
internal/app/storage/     # OnChange generation, LiveVersions
internal/app/persist/     # OnConfirmed hook, confirmed watermark, rollback Restore
internal/udprepl/         # KnownPeerIDs mirror, notification peer set, tracker wants
internal/app/repl/        # promise-set seeding, discovery flush without wait-set growth
```

### Pattern 1: Write-time promise set

**What:** Capture the peer ids once, at write acceptance, and carry them as data through the
notification into the core's pending entry and the adapter's tracker.
**When to use:** Every write that requests a wait report, and every replayed record (the
load starts with the peers known at load time).
**Why:** The Ack ledger must answer "did the peers known when the write was accepted
confirm it?" — a moving peer table cannot answer that later.

### Pattern 2: Durability watermark change feed

**What:** The store tags each install with its monotonic content generation; the engine
publishes the generation captured by the last successful dump; the hub releases pending
events with `gen <= watermark` in order.
**When to use:** Any consumer-visible event stream whose contract is "durable changes".
**Why:** It gives exactly-once, install-order/version-order delivery and naturally covers a
write whose first dump failed but whose later dump succeeded.

### Pattern 3: Keyset pagination over a cheap view

**What:** Enumerate live keys with versions (no values), sort, seek past the token, then copy
only the page's values under a byte budget.
**When to use:** `List` of a store whose values are up to the 60 KiB limit.
**Why:** The page's cost is a function of the page, not of the store's value bytes.

### Pattern 4: Total mapping with a constant fallback

**What:** One `errors.Is`/class table plus a fallback that logs the detail and returns a
constant message.
**When to use:** Any adapter translating internal errors into a public protocol.
**Why:** Totality is structural; path-freeness is by construction, not by review.

### Pattern 5: One-way health via the standard server

**What:** Call `health.Server.Shutdown()` at the start of shutdown; never call `Resume`.
**When to use:** Any grpc-go server with the standard health service.
**Why:** The library already ignores every later `SetServingStatus`, closing the
`RunGate`-after-shutdown race without a second state machine.

### Pattern 6: Public generated package with alias-preserving moves

**What:** Move the proto tree to a public module root, keep the `go_package` package-name
suffix stable at call sites via an import alias, regenerate once, and pin the descriptor.
**When to use:** Promoting an internal schema to a consumer contract.
**Why:** The import churn is mechanical, reviewable, and does not touch identifiers or the
wire.

### Pattern 7: Full-rollback replay as a sorted union pass

**What:** Diff the snapshot against the current live keys, then replay/delete the sorted
union in one deterministic pass and return the installed records.
**When to use:** Snapshot load semantics that mean "this is the state", not "merge this in".
**Why:** The order is deterministic, the returned list is exactly what must replicate, and
the no-op condition is explicit.

### Anti-Patterns to Avoid

- **Polling or appearance windows in `wait_for_replication`:** the Outbox result stream
  already resolves supersession; a timer reintroduces the 500 ms class of bug.
- **Using `Engine.Records()` on any request path:** it deep-copies every value; that is the
  A-02 defect.
- **Formatting arbitrary errors into a status message:** the `%v` fallback is the A-03 path
  leak.
- **A second health state machine:** `health.Server.Shutdown` is the one-way switch.
- **Announcing a local write at install time:** the tap runs before the dump; A-04's whole
  point.
- **Notifying Watch from the gRPC handler after `Set` returns:** loses confirmed-after-
  failure writes and can invert ordering against replicated changes.
- **Keeping the proto at both the internal and public paths:** two schemas drift; one move
  with a buf waiver is enough.
- **Editing frozen red-test bodies:** import path only (Pitfall 1).
- **Computing gRPC limits from `storage.ValueMaxBytes` alone:** the effective configured
  limit may be higher; the transport would reject valid records.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Opaque page tokens | Custom encoding/checksum | `encoding/base64.RawURLEncoding` of the last key | Stdlib, URL-safe, no state |
| Health one-way transition | Mutex + state enum + `SetServingStatus` calls | `health.Server.Shutdown()` | Already ignores future status changes (verified in grpc v1.85) |
| Errno classification | `syscall` switches in `grpcapi` | `port.FailureClassOf` + `classifySaveError` in the adapter | One class vocabulary; app stays syscall-free |
| Server keepalive/size knobs | Hand-written connection management | `grpc.KeepaliveParams`, `grpc.KeepaliveEnforcementPolicy`, `grpc.MaxRecvMsgSize`, `grpc.MaxSendMsgSize` | The library implements the protocol |
| Proto lint/codegen | Hand-edited `.pb.go` | `go tool buf generate` + the pinned plugins | Generated files must stay reproducible |
| Sorting/prefix/union | Custom ordering structures | `slices.Sort`, `strings.HasPrefix`, `slices.Sorted(maps.Keys(...))` | Existing idiom, deterministic |
| Client example transport | Raw HTTP/2 or a bespoke client | The generated `supkvapi.NewSupKVServiceClient` | The example proves the public package, not a second client |
| Ack bookkeeping | A second ack table in the API | The existing tracker + core Outbox results | The core is the single driver of replication state |

**Key insight:** every hard part of this phase already has a seam from Phases 3-5 (Outbox
results, confirmation waiters, append-only generation, health server). The work is to
*publish the existing facts* at the right boundary, not to build parallel machinery.

## Common Pitfalls

### Pitfall 1: The frozen red tests and the P-01 rows

**What goes wrong:** the public proto move edits every importer, including
`internal/grpcapi/red_gate_test.go` and `internal/stand/stand_red_test.go`, which carry the
frozen R-03/P-01 acceptance tests. A gate that demands byte-identity against the phase base
fails on the one-line import change.
**Why it happens:** Go import paths are absolute; there is no alias that keeps the old
path compiling without a forwarding package (which would be worse).
**How to avoid:** permit exactly the import line edit, keep test names/bodies/rows
unchanged, and record the one-line diff as disclosed evidence. Run `bash scripts/red-runner`
(7/7 green) in the phase gate. The red-runner is name-based, so the rows are unaffected.
**Warning signs:** a diff in those files beyond the import line; a red-runner row that
disappears or flips.

### Pitfall 2: A 20 ms wall-clock assertion under `-race`

**What goes wrong:** asserting `elapsed < 20ms` on a real adapter in CI flakes under
`-race` and parallel load, even though the behavior is correct.
**Why it happens:** scheduler latency, not protocol latency, dominates a 20 ms bound.
**How to avoid:** prove the property deterministically at the core/Outbox level (the
`ResultSuperseded` event is emitted in the same handler call that displaces the version),
then assert a generous real-adapter bound (e.g. < 250 ms) with a long wait timeout,
documenting the 20 ms design target in the test comment.
**Warning signs:** a retry loop around a 20 ms assertion; a test that only passes alone.

### Pitfall 3: Moving the tracker creation into the core

**What goes wrong:** if the tracker is created when the event loop dequeues the
notification, the handler can call `WaitReplicated` before it exists and report a phantom
false (or spin).
**Why it happens:** the notification is asynchronous; the handler returns from
`NotifyLocalWrite` and waits immediately.
**How to avoid:** keep `ensureTracker` in `Adapter.NotifyLocalWrite` **before** the channel
send (the Phase-3 single-creation contract); the new peer set is additional tracker state,
not a new creation site.
**Warning signs:** a poll loop in `WaitReplicated`; a test that passes only with a sleep.

### Pitfall 4: Discovery flush and the promise set

**What goes wrong:** the discovery flush currently adds a newly seen peer to every pending
wait-set. If a write-time set is introduced but this flush is left unchanged, the core
awaits peers outside the captured set again — the IN-05 false-negative persists, and a
new peer's Ack can be the one that empties the set.
**Why it happens:** `waitSet` serves two purposes today: the Ack promise and the
retransmission/announcement target.
**How to avoid:** split the responsibilities: keep the broadcast/ladder reset in the flush,
remove `waitSet[in.NodeID]` and `ledgerAdd`; seed the promise set only from the
notification. Add a test that a peer appearing after the write receives the record but does
not join the awaited set.
**Warning signs:** `TestRetransmit_suspendAndFlush` and the peertable tests asserting the
old wait-set growth; update them deliberately.

### Pitfall 5: `Versions()` cannot tell live from tombstoned

**What goes wrong:** `Versions()` is tombstone-inclusive and returns only a version; using
`len(Data)==0` to filter tombstones drops legitimate empty values.
**Why it happens:** the replication view needs tombstones; the List view does not.
**How to avoid:** add `Store.LiveVersions()` (and the `Engine` delegate) that skips
`Deleted` under `RLock` and copies no `Data`; use it only in `List`.
**Warning signs:** a List test with an empty live value that disappears; a List path that
calls `Records()`.

### Pitfall 6: `wait_for_replication` tracker lifecycle

**What goes wrong:** a new total/peer-set parameter tempts a refactor that creates the
tracker later or deletes it on the wrong path; the zero-peer release path is easy to lose.
**Why it happens:** the tracker is bounded by concurrent waits, and every return path owns
a delete.
**How to avoid:** keep the Phase-3 rules: creation only in `NotifyLocalWrite` before the
send; a failed notify deletes the just-created tracker; every `WaitReplicated` return path
deletes; the zero-peer path still calls the seam to release. Add the captured set to the
tracker; do not add a second creation site.
**Warning signs:** a tracker map size test that grows; a zero-peer test that asserts a
non-zero tracker map.

### Pitfall 7: `mapError` logging global state

**What goes wrong:** capturing `slog.Default()` to assert the logged detail makes the test
non-parallel and can race other tests that swap the handler.
**Why it happens:** `mapError` is a package function.
**How to avoid:** one non-parallel test (the `TestAwaitEngineStop` pattern) that installs a
handler, triggers a path-bearing error, asserts the response is constant and the log carries
the path; keep all other tests parallel.
**Warning signs:** `t.Parallel()` in the slog-capturing test; a flaky log assertion.

### Pitfall 8: Startup load fires the Watch tap

**What goes wrong:** `persist.Open` merges every loaded record through `Store.Merge`, which
fires `OnChange`; with the hub wired, the whole loaded state is enqueued as "changes". A
clean `Main` load never dumps, so those events never confirm and sit pending forever; a
large load overflows the ingress and sheds (no) subscribers.
**Why it happens:** the loader is a store writer like any other.
**How to avoid:** wrap the caller's `OnChange` in `Open` with a synchronous `loading` guard
around `mergeRecords`, and initialize the engine's confirmed watermark to the post-load
store generation. Add a test that `Open` with a dump fires no tap.
**Warning signs:** a Watch test that sees the seeded keys twice; a pending queue that never
drains after a restart.

### Pitfall 9: `RunGate` wins the race after shutdown began

**What goes wrong:** `shutdown` sets `NOT_SERVING`, then cancels the API context; the
resolver's backstop/poll can still fire `openGate` in between and flip health back to
`SERVING`.
**Why it happens:** two independent state stores (`gateOpen` and the health map) with no
one-way latch.
**How to avoid:** call `health.Server.Shutdown()` from a new `Server.Shutdown()` before
`apiCancel`; the health server ignores every later `SetServingStatus`. Never call `Resume`.
Test with a short backstop that fires after `Shutdown`.
**Warning signs:** a health test that sometimes sees `SERVING` after `NOT_SERVING`.

### Pitfall 10: Unbounded pending watch events

**What goes wrong:** if dumps fail persistently, the pending queue grows with every write
and Watch becomes a memory leak.
**Why it happens:** events cannot be released without a confirmation.
**How to avoid:** bound pending at `watchIngress` (4096 record references; the store owns
the bytes); on overflow disconnect all subscribers, clear pending, and let resubscribe +
snapshot heal. Keep the existing `watchBuffer` shed for slow readers.
**Warning signs:** a growing heap under a failed dump; a subscriber that never reconnects.

### Pitfall 11: gRPC limits from the wrong constant

**What goes wrong:** computing `MaxRecvMsgSize` from `storage.ValueMaxBytes` (61 440) while
a node runs `--value-max-bytes=64408` rejects a legitimate maximally sized Set at the
transport layer.
**Why it happens:** the default constant and the effective config differ.
**How to avoid:** compute `LimitsFor(cfg.valueMaxBytes)` (validated positive at startup) and
add a process-level round-trip at the largest accepted value limit.
**Warning signs:** a `ResourceExhausted` on a record the store would accept.

### Pitfall 12: `CreateSnapshot` errors are not classified

**What goes wrong:** ENOSPC/EACCES from the snapshot copy reach `mapError` without a class,
so they cannot map to `ResourceExhausted`/`PermissionDenied` and fall to `Internal`.
**Why it happens:** `CreateSnapshot` calls `saveAtomically` directly, unlike `SaveStream`.
**How to avoid:** wrap `CreateSnapshot`'s save error with `classifySaveError` (adapter-side);
keep `ErrDumpFailed` ordered before the class check in the mapper.
**Warning signs:** a disk-full snapshot test that expects `ResourceExhausted` and gets
`Internal`.

### Pitfall 13: Full rollback and the no-op check

**What goes wrong:** adding the delete pass without updating `restoreChanges` makes a
rollback that only deletes absent live keys look like a no-op (empty list, no dump, no
notification) — the extra key survives.
**Why it happens:** the current check iterates only the snapshot's keys.
**How to avoid:** treat any current live key absent from the snapshot as a change; test a
snapshot equal to current state plus one extra live key.
**Warning signs:** a rollback test that deletes nothing because the snapshot otherwise
matches.

### Pitfall 14: Tombstoned-but-absent keys in a rollback

**What goes wrong:** writing a fresh tombstone for an already-tombstoned key absent from the
snapshot burns a version and broadcasts a redundant record; `Store.Delete` returns
`ErrNotFound`, so a naive "delete every absent key" loop reports errors or skips
inconsistently.
**Why it happens:** "absent from the snapshot" is not "live in the store".
**How to avoid:** delete only current **live** keys absent from the snapshot; skip
already-tombstoned ones (they are already deleted, no version burned). Document it.
**Warning signs:** a rollback test where `gone`'s version moves.

### Pitfall 15: Editing `docs/EXAMPLES.md` and the SPEC-DEVIATIONS paths is mandatory, not optional

**What goes wrong:** the public move and the List response make documented paths and sample
outputs stale in the same commit.
**Why it happens:** the docs are the as-built contract and are test-anchored in Phase 9.
**How to avoid:** update the List output block, the wait paragraph (zero peers), the
SPEC-DEVIATIONS proto/API paths, and the AGENTS.md Layout in the phase; leave the review
evidence history untouched. Add a grep gate over source/config/docs excluding
`docs/reviews/**` and `.planning/**` for the old import path.
**Warning signs:** `grep -rn internal/grpcapi/apiproto --include='*.go'` hits; a doc link
that no longer resolves.

### Pitfall 16: `buf breaking` and the `go_package` change

**What goes wrong:** the move changes `go_package`, and `buf breaking` (FILE policy) fails
the lint job with `File option "go_package" changed ...`.
**Why it happens:** the package option is part of the file descriptor comparison.
**How to avoid:** a one-entry `ignore_only: FILE_SAME_GO_PACKAGE` scoped to
`api/supkvapi/v1/supkvapi.proto`; probe-verified exit 0, and a pure move with the same
`go_package` also passes.
**Warning signs:** the lint job red only on that message; adding a broad ignore instead of
the one file.

### Pitfall 17: The public package importing `internal`

**What goes wrong:** someone adds a helper to `api/supkvapi/v1` that imports an internal
package; external consumers can then no longer import the public package (Go's internal
rule), and the API stops being public in practice.
**Why it happens:** the same module compiles it, so local tests pass.
**How to avoid:** keep the public package generated-only plus `doc.go`/tests; add the
`sup-kv-public-api-no-internal` depguard rule; the client example module is the cross-module
proof.
**Warning signs:** an `internal/` import in `api/**`; a consumer build failure.

### Pitfall 18: The client example's separate-module hygiene

**What goes wrong:** the root `go test ./...` does not see the example; its `go.sum` or
`go.mod` drifts, or the CI job forgets `go mod tidy`.
**Why it happens:** Go excludes nested modules from the parent's package patterns.
**How to avoid:** a dedicated CI job (`go build`, `go vet`, `go mod tidy`, scoped
`git diff --exit-code`) and a Makefile target; use `replace ../..`; keep the example
dependency-free beyond the public package + grpc.
**Warning signs:** a stale example go.sum; an example that builds only from the repo root.

### Pitfall 19: `TestEngine_snapshotCreateLoadRestore` and the P-01 frozen tests

**What goes wrong:** assuming the snapshot test is frozen because its comment says 14.1.
It is not one of the P-01 tests; the P-01 tests live in `internal/persistfile/` and
`internal/stand/` and must stay byte-identical.
**Why it happens:** the review wording couples S-01 and P-01 in the same milestone table.
**How to avoid:** rewrite the snapshot test and its additive siblings; do not touch
`red_value_limit_test.go`/the P-01 stand test bodies; keep the TSV rows unchanged and green.
**Warning signs:** a diff in the P-01 files; a red-runner row renamed.

## Runtime State Inventory

> This phase contains one relocation (the API proto/generated package). It is a source
> import move, not a runtime rename; the categories are answered explicitly so the planner
> can see that nothing outside git needs migrating.

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — no dump format, key, schema, node identity or version layout changes. The rollback changes snapshot-load *semantics*, not the on-disk grammar. | None |
| Live service config | None — no external service holds a sup-kv import path or proto path. Deployments hold flags/env only, unchanged. | None |
| OS-registered state | None — no systemd/Task Scheduler/launchd/pm2 unit embeds the package path. | None |
| Secrets/env vars | None — the `SUPKV_*` env names and the metrics certificate envs are unchanged. | None |
| Build artifacts / installed packages | The committed `api/supkvapi/v1/*.pb.go` (regenerated), the new `examples/client/go.sum` (new module), and any developer's stale `internal/grpcapi/apiproto` build cache. The historical `docs/reviews/.../evidence/**` Go files still carry the old import path and are never built. | Regenerate + commit; `go clean -cache` is optional; do not edit the historical evidence module |

## Code Examples

> The blocks below are the recommended shapes. Existing names/values quoted from source
> carry `[VERIFIED]`; new constants and signatures are proposals under Claude's discretion.

### Write-time capture at the handler (proposed shape)

```go
// ServerOptions addition (names are proposals; capture point is the contract).
WritePeers func() []uint16 // sorted; nil = zero peers

// Set/Delete, immediately before the engine call:
peers := s.writePeers() // clones the adapter mirror
ver, _, err := s.engine.Set(req.GetKey(), req.GetValue())
...
s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, req.GetWaitForReplication(), peers)
...
if req.GetWaitForReplication() {
	resp.Acked, resp.AckedCount, resp.TotalPeers =
		s.waitForReplication(ctx, req.GetKey(), ver, uint32(len(peers)), timeout)
}
```
`storage.MaxKeyBytes = 1024` and `ValueMaxBytes  = 61440`
`[VERIFIED: internal/app/storage/store.go:21,33]` are the two protocol constants the
message-size limits derive from.

### Core seeding from the notification's set (proposed shape)

```go
func (c *Core) trackLocalWrite(..., peers []uint16, ...) (pendingKey, bool) {
	// ... guard/supersession unchanged ...
	waitSet := make(map[uint16]struct{}, len(peers))
	for _, id := range peers {
		if _, known := c.peers[id]; known { // a gone peer cannot Ack
			waitSet[id] = struct{}{}
		}
	}
	if len(peers) > 0 && len(waitSet) == 0 {
		out.WaitSetEmpty(key, rec.Version) // honest immediate false
	}
	// ... entry + ledger for waitSet members + retry push ...
}
```
The `Superseded` result already exists verbatim in three sites
(`internal/app/repl/core.go:2216,3239,1525`) `[VERIFIED: internal/app/repl/core.go:2216,3239,1525]`.

### Confirmation watermark (proposed shape)

```go
// scheduler.runDump, before the snapshot:
gen := s.store.Generation()
records := s.store.Records()
// on success, after confirmWaiters:
s.confirmed.Store(gen)
if s.cfg.OnConfirmed != nil {
	s.cfg.OnConfirmed(gen) // must not block
}
```
`Store.Generation` is the existing monotone content generation
(`internal/app/storage/store.go:370`) `[VERIFIED: internal/app/storage/store.go:370]`; it
moves by exactly one per `install` (`store.go:434`) `[VERIFIED: internal/app/storage/store.go:434]`.

### Hub release (proposed shape)

```go
func (h *WatchHub) Confirm(gen uint64) {
	for {
		cur := h.confirmed.Load()
		if gen <= cur || h.confirmed.CompareAndSwap(cur, gen) {
			break
		}
	}
	select { case h.confirmCh <- struct{}{}: default: }
}

// in Run, after the select:
for len(h.pending) > 0 && h.pending[0].gen <= h.confirmed.Load() {
	ev := h.pending[0]
	h.pending = h.pending[1:]
	h.dispatch(ev)
}
```

### Total mapper fallback (proposed shape)

```go
	if _, ok := status.FromError(err); ok {
		return err
	}
	slog.Error("Unmapped RPC error", "detail", err.Error())
	return status.Error(codes.Internal, "internal error")
```
The current line being replaced is `status.Errorf(codes.Internal, "internal error: %v", err)`
`[VERIFIED: internal/grpcapi/server.go:453]`.

### Buf config after the move (probe-verified)

```yaml
modules:
  - path: internal/udprepl/replproto
  - path: api

breaking:
  use:
    - FILE
  ignore_only:
    FILE_SAME_GO_PACKAGE:
      - api/supkvapi/v1/supkvapi.proto
```
With this, `go tool buf breaking --against '.git#ref=HEAD'` exits 0 after the move; without
the waiver it fails on the `go_package` option change. The probe also confirmed:
`go tool buf lint` OK, `go tool buf format --diff --exit-code` OK, additive List fields OK,
`go tool buf generate api --template buf.gen.api.yaml` writes
`api/supkvapi/v1/{supkvapi.pb.go,supkvapi_grpc.pb.go}` with `package supkvapi` and
`source: supkvapi/v1/supkvapi.proto`, and `go build ./...` is green.

### Full-rollback pass (proposed shape)

```go
func (e *Engine) replaySnapshot(records map[string]storage.Record) ([]RestoredRecord, error) {
	current := e.store.Records()
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	for key, rec := range current {
		if rec.Deleted {
			continue // already rolled back
		}
		if _, inSnap := records[key]; !inSnap {
			keys = append(keys, key) // rollback delete
		}
	}
	if !restoreChanges(current, records) && len(keys) == len(records) {
		return nil, nil
	}
	slices.Sort(keys)
	// for each key: in snapshot -> replayRecord; else -> store.Delete + tombstone record
	...
}
```
`Store.Delete` returns `ErrNotFound` for a missing/already-tombstoned key and burns no
version (`internal/app/storage/store.go:293-304` comment) `[VERIFIED: internal/app/storage/store.go:293-304]`.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Wait polls a mirror for appearance | Outbox result stream resolves the tracker | Phase 3 (XPORT-12) | No polling; a superseded version already resolves fast |
| Wait total = live peer count at wait time | Write-time peer set captured at acceptance | This phase (APIS-01) | Zero peers ⇒ false; discovered peers never join the promise; IN-05 closes |
| List deep-copies every value | Page over a live key→version view + page values | This phase (APIS-02) | Cost independent of store-wide value bytes |
| Watch announces at store install | Watch announces after the covering dump | This phase (APIS-04) | Durable feed; exactly once; version order |
| Default gRPC server + direct health set | Explicit limits/keepalive + `health.Shutdown` | This phase (APIS-05) | Pinned transport; one-way health |
| API proto under `internal/` | Public `api/supkvapi/v1` + example module | This phase (APIS-06) | Consumers can build against it |
| Snapshot load is additive | Snapshot load is a full rollback | This phase (APIS-07) | 14.1 implemented; result broadcasts |

**Deprecated/outdated:**
- `ServerOptions.KnownPeers` as a wait seam — replaced by `WritePeers`; the adapter's
  `KnownPeers()` stays for metrics only.
- `List`'s comment about the 4 MiB envelope — pagination makes it obsolete.
- The proto comments carrying phase/decision ids — DOCS-10 removes them; doing it during
  the move avoids shipping a public schema that references internal planning.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A wall-clock 20 ms assertion is too flaky under `-race`; a deterministic Outbox assertion plus a generous real-adapter bound is the honest proof | D-01, Pitfall 2 | A plan that writes the literal 20 ms test may flake; the property is still correct |
| A2 | "Values ride only for the entries of the page" means value-bearing pages; a keys-only page is an acceptable fallback if the planner reads it as vacuous | D-02 | A keys-only implementation may be judged to under-deliver D-02, or a value-bearing page may be judged beyond A-02's minimum |
| A3 | The Watch snapshot should stay the current in-memory state rather than a confirmed-only view | D-04 | If the owner wants a fully durable snapshot, the hub needs a subscribe-time pending filter and an overflow policy for it |
| A4 | Page defaults (64/256, 6 MiB budget) and keepalive values (2 m/20 s, 30 s) are appropriate for the trusted-LAN small-cluster envelope | D-02, D-05 | Tunable constants; wrong values are a UX/performance issue, not a contract break |
| A5 | Keeping `ListResponse.keys` populated alongside `records` is worth the redundancy for shape compatibility | D-02 | Removing it would be a cleaner schema but a larger client break |

**If this table is not empty:** the planner should treat A1-A5 as discretion choices to pin
in the plans; A2 and A3 are the two a reviewer could reasonably decide differently.

## Open Questions

1. **Value-bearing List pages vs keys-only?**
   - What we know: D-02 says "Values ride only for the entries of the page"; A-02 only
     requires cost independence.
   - What's unclear: whether the decision author meant records or just "no whole-store
     copy".
   - Recommendation: value-bearing `records` with a byte budget (the literal reading);
     keep `keys` populated for shape compatibility. If the planner wants the minimal wire
     change, keys-only still passes A-02 but should be recorded as a deliberate reading of
     D-02.

2. **How strictly to assert the 20 ms superseded bound?**
   - What we know: `ResultSuperseded` is emitted synchronously in the displacing handler;
     the tracker wakes on it.
   - What's unclear: whether CI can hold a 20 ms wall-clock bound under `-race`.
   - Recommendation: deterministic core assertion + real-adapter < 250 ms, with the 20 ms
     target in the test comment (Pitfall 2). Do not add a retry loop.

3. **Keep the `FILE_SAME_GO_PACKAGE` waiver forever?**
   - What we know: the move cannot pass buf breaking without it; a pure move passes.
   - What's unclear: whether a later phase will remove it.
   - Recommendation: keep it scoped to the one file with an explanatory comment; removing it
     after the move merges is safe but not required. A future change to the public
     `go_package` would then be silently accepted — call that out in the comment.

4. **Does Watch need a confirmed-only snapshot?**
   - What we know: the live feed will be durable-only; `Get` and the snapshot show current
     in-memory state.
   - What's unclear: whether the owner considers the snapshot part of the "durable" claim.
   - Recommendation: document the asymmetry (snapshot = current state; live events =
     durable changes). A confirmed-only snapshot needs a subscribe-time pending filter and
     an overflow story; not worth it for v1.1's acceptance.

5. **Remove `ServerOptions.KnownPeers`?**
   - What we know: it is only used by `waitForReplication`; metrics use
     `adapter.KnownPeers` directly.
   - Recommendation: remove it from `ServerOptions` (and main/test wiring), replace with
     `WritePeers`; keep `Adapter.KnownPeers()` for metrics.

6. **Public package name and path?**
   - What we know: `api/supkvapi/v1` keeps the proto's source-relative path unchanged and
     the module root simple; the package name can be `supkvapi` with internal aliases.
   - Recommendation: `api/supkvapi/v1`, package `supkvapi`, internal alias `apiproto`.

7. **Separate Go module for the public API?**
   - What we know: a submodule isolates consumers from the server's tool graph but needs
     prefixed tags and release machinery.
   - Recommendation: no; one module plus the `examples/client` module with `replace`.

8. **When to flip registry row S-PERS-12?**
   - What we know: Phase 5's precedent flips rows in-phase; Phase 9 closes the registry.
   - Recommendation: flip S-PERS-12 in this phase with the new evidence; leave S-PERS-13
     (PROTO-08) to Phase 9's audit unless the phase touches it.

9. **`docs/API.md` now or Phase 9?**
   - What we know: D-04 says the contract text lands in the API docs; DOCS-05 owns the final
     document.
   - Recommendation: create `docs/API.md` with the phase-owned sections (wait, errors,
     List, Watch, snapshots, transport settings, last-read-value pattern) and mark the
     remaining per-RPC/failure-table/latency sections for Phase 9.

10. **Do we need a real-socket two-node full-rollback test?**
    - What we know: the handler's replay seam is the broadcast contract, and PROTO-08 tests
      already prove the replication of replayed records.
    - Recommendation: the RPC-level notify-capture test plus the existing PROTO-08 suite is
      enough for acceptance; a stand two-node test is a nice-to-have if budget allows.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Everything | ✓ | `go1.27.4 linux/amd64` | — |
| `go tool buf` | Proto lint/format/breaking/generate | ✓ | `1.75.0` | — |
| `go tool protoc-gen-go` / `-grpc` | Codegen | ✓ | v1.36.12-devel / v1.6.2 | — |
| `go tool golangci-lint` | Lint gate | ✓ | v2.15.0 (pinned) | — |
| git + module cache | Builds/tests | ✓ | — | CI has network |
| `grpcurl` (tool) | Docs examples only | ✓ | tool directive | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none.

## Validation Architecture

> `workflow.nyquist_validation` is absent from `.planning/config.json`, so this section is
> included.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go 1.27 `testing` + `go test -race`; `synctest` for engine timing tests; `bufconn` for gRPC; real UDP/TCP in `internal/stand` and `cmd/sup-kv` |
| Config file | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `race-soak`); `scripts/red-expectations.tsv` |
| Quick run command | `go test -race -count=1 ./internal/grpcapi/ ./internal/app/repl/ ./internal/udprepl/ ./internal/app/persist/ ./internal/app/storage/` |
| Full suite command | `make lint && make test && bash scripts/red-runner && make buf-lint && make buf-breaking && make client-example` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command / Test | File Exists? |
|--------|----------|-----------|--------------------------|--------------|
| APIS-01 | zero peers ⇒ `false,0,0`; nil seam ⇒ `false,0,0` | unit | `go test -race ./internal/grpcapi/ -run 'TestWait_zeroPeers|TestWait_trackerNil'` | ❌ rewrite existing |
| APIS-01 | write-time set pinned; a post-write peer's Ack cannot make true | unit + core | `go test -race ./internal/app/repl/ -run 'TestNotify|TestWait'`; `./internal/udprepl/ -run 'TestWaitReplicated'` | ❌ new/updated |
| APIS-01 | superseded resolves on the result, not the timeout | core (deterministic) + adapter (generous bound) | `TestNotify_supersededResult`; `TestWaitReplicated_supersededFast` | ❌ new |
| APIS-02 | page/token/size/prefix/tombstone/end semantics | unit | `go test -race ./internal/grpcapi/ -run TestServer_list` | ❌ rewrite |
| APIS-02 | live view copies no values; page cost value-independent | unit/allocs | `TestStore_liveVersions`; `TestServer_listCostIndependent` | ❌ new |
| APIS-03 | every sentinel/class maps; unknown ⇒ constant Internal; no paths | unit + RPC matrix | `go test -race ./internal/grpcapi/ -run 'TestMapError|TestRPC_noPaths'` | ❌ new |
| APIS-03 | missing snapshot ⇒ NotFound; no dump ⇒ FailedPrecondition | integration (real persistfile) | same file, real-adapter cases | ❌ new |
| APIS-04 | unconfirmed write is not announced; confirmed is exactly once | integration (bufconn + held/failing dump) | `go test -race ./internal/grpcapi/ -run 'TestWatch_unconfirmed|TestWatch_confirmed'` | ❌ new |
| APIS-04 | dump-failed-then-confirmed is announced once | integration | `TestWatch_dumpFailedThenConfirmed` | ❌ new |
| APIS-04 | shutdown ends streams Unavailable | unit + process | `TestWatch_shutdownUnavailable`; `cmd/sup-kv` SIGTERM test | ❌ new |
| APIS-05 | limits table + over-limit recv ⇒ ResourceExhausted | unit + bufconn | `go test -race ./internal/grpcapi/ -run 'TestLimitsFor|TestGRPC_maxRecv'` | ❌ new |
| APIS-05 | health never returns to SERVING | unit | `TestHealth_shutdownIsOneWay` | ❌ new |
| APIS-06 | public package imports/builds; schema pinned | compile + descriptor | `go build ./...`; `go test ./api/supkvapi/v1/ -run TestSchema` | ❌ new |
| APIS-06 | separate example module builds; buf gates | CI/module | `make client-example`; `make buf-lint`; `make buf-breaking` | ❌ new |
| APIS-07 | full rollback deletes absent live keys; result in the notify list | unit | `TestEngine_snapshotCreateLoadRestore`; `TestEngine_restoreReturnsReplayed` | ❌ rewrite |
| APIS-07 | RPC broadcast includes rollback tombstones (even failed flush) | integration | `TestServer_snapshotLoadRollback` | ❌ new |
| All | frozen red tests and rows stay green | gate | `bash scripts/red-runner` (7/7) | ✅ existing |

### Sampling Rate

- **Per task commit:** the task's own `go test -race -count=1 <pkg> -run <focused>` (≤ 60 s).
- **Per wave merge:** `go build ./... && go test -race ./internal/grpcapi/ ./internal/app/repl/ ./internal/udprepl/ ./internal/app/persist/ ./internal/app/storage/`.
- **Phase gate:** `make lint && make test`, `bash scripts/red-runner`, `make buf-lint && make buf-breaking`, `make client-example`, and the process-level tests in `cmd/sup-kv`.
- **Max feedback latency:** ~300 s full suite; the buf/example gates are ~10 s.

### Wave 0 Gaps

- [ ] `internal/grpcapi/errors_test.go` — APIS-03 mapping table + path-shape matrix.
- [ ] `internal/grpcapi/limits_test.go` — APIS-05 limits/one-way health.
- [ ] `internal/grpcapi/watch_test.go` additions — APIS-04 confirmation cases.
- [ ] `internal/grpcapi/wait_test.go` rewrites — APIS-01 semantics.
- [ ] `internal/app/storage/store_test.go` additions — live view + generation tap.
- [ ] `internal/app/persist/engine_test.go` rewrites — rollback + confirmed watermark + startup tap.
- [ ] `api/supkvapi/v1/{doc.go,schema_test.go}` — public package hygiene + descriptor pin.
- [ ] `examples/client/{go.mod,main.go,README.md}` + CI job + Makefile target.
- [ ] Documentation: `docs/API.md` (phase-owned sections), `docs/EXAMPLES.md` List/wait updates, `docs/04-MULTI-NODE.md` IN-05 closure, `docs/SPEC-DEVIATIONS.md` paths + S-PERS-12, `AGENTS.md` layout.
- [ ] Framework install: none — the existing Go test/buf/CI infrastructure covers everything.

### Red-runner impact

No new rows. All seven rows stay green:
`TestRed_R03_gateClosedUntilFullList`, `TestRed_R01_pacedSync20k`, `TestRed_R02_unackedDrain`,
`TestRed_R02_burst`, `TestRed_R03_f14ReplacedNode`, `TestRed_P01_valueLimit`,
`TestRed_P01_valueLimitEngine`. The only edits in the two frozen test files are the
`apiproto` import lines; record the diff and run the runner in the phase gate.

## Security Domain

`security_enforcement` is not disabled in `.planning/config.json`, so this section applies.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Trusted-LAN model; datagram auth is AUTH-01/v2; gRPC has no authn in v1.1 |
| V3 Session Management | no | No sessions; gRPC connections; health/keepalive only |
| V4 Access Control | no | No per-key authorization; loopback default bind; trusted segment |
| V5 Input Validation | yes | Key/value limits (`storage.MaxKeyBytes`, `ValueMaxBytes`), snapshot-name allow-list (`validateName`), page token decode bound, page size validation, proto_version/CRC on the UDP wire |
| V6 Cryptography | no (nothing new) | No new crypto; CRC32 is corruption detection, not tamper protection; metrics mTLS is pre-existing |
| V7 Error Handling & Logging | yes | Total path-free mapping; detail logged, never returned; no `%v` in statuses |
| V8 Data Protection | yes (at rest) | Dump fsync/rename integrity is pre-existing; the rollback must not bypass the confirmation contract |
| V12 Files & Resources | yes | Snapshot path traversal is closed by `validateName`; page/list/recv bounds prevent resource exhaustion |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path disclosure via status messages | Information Disclosure | Constant messages; log the detail; path-shape test over the RPC matrix |
| Path traversal via snapshot name | Tampering | `validateName` charset/length + `.`/`..` rejection before any FS touch (existing) |
| Memory/CPU exhaustion via `List` | Denial of Service | Keyset pages, `maxListPageSize`, `listPageByteBudget`, explicit `MaxRecvMsgSize`/`MaxSendMsgSize` |
| Memory exhaustion via Watch subscribers | Denial of Service | Bounded ingress/pending/per-subscriber buffers; disconnect-all overflow; resubscribe+snapshot |
| Slowloris / dead client holding a stream | Denial of Service | gRPC keepalive Time/Timeout + enforcement policy; `GracefulStop` grace then `Stop` |
| Forged/hostile page token | Tampering | Base64 decode + length bound; token is only a position, never a path or index |
| Error-text injection into client logs | Spoofing/Repudiation | No server detail in messages; the client sees stable codes/constants |
| Health-state race during shutdown | Denial of Service (traffic to a dying node) | One-way `health.Server.Shutdown`; orchestrator gates on the endpoint |
| Cross-module import of internal code | Elevation of Privilege (API surface) | Public package generated-only + depguard rule; Go's internal rule for the rest |

## Sources

### Primary (HIGH confidence)

- In-repo source read this session:
  `internal/grpcapi/{server.go,wait.go,watch.go,gate.go}`,
  `internal/udprepl/repl.go`, `internal/app/repl/core.go`,
  `internal/app/persist/{engine.go,scheduler.go,snapshot.go,loader.go,facade.go,errors.go}`,
  `internal/app/storage/{store.go,errors.go}`, `internal/persistfile/snapshot.go`,
  `internal/port/dump.go`, `cmd/sup-kv/main.go`, the API proto, `buf.yaml`,
  `buf.gen.api.yaml`, `Makefile`, `go.mod`, the Phase-6 test files and `engine_test.go`.
- gRPC v1.85.0 module source: `server.go:59-62`, `internal/transport/defaults.go:35-40`,
  `health/server.go:139-148,169-176` (the one-way `Shutdown` contract).
- Throwaway-clone probes (working tree untouched):
  - pure proto move with the same `go_package` → `buf breaking` exit 0;
  - move + `go_package` change → `FILE_SAME_GO_PACKAGE` failure, and
    `ignore_only` scoped to the new path → exit 0;
  - additive List fields → `buf lint` OK and `buf breaking` exit 0;
  - `go tool buf generate api --template buf.gen.api.yaml` → `api/supkvapi/v1/*.pb.go`,
    `package supkvapi`, `source: supkvapi/v1/supkvapi.proto`;
  - 21 import lines sed-updated (19 real importers + the 2 evidence files) + `gofmt` →
    `go build ./...` green, 26 paths changed, 31 insertions/30 deletions;
  - `examples/client` separate module (require + replace) → build OK, 18-line `go.sum`,
    `go list ./...` at the root excludes it.

### Secondary (MEDIUM confidence)

- `.planning/REQUIREMENTS.md` APIS-01…07 acceptance; `.planning/ROADMAP.md` Phase 6
  criteria; `docs/reviews/2026-11-19-v1.0/REVIEW.md` §7 (A-01…A-06), §14.1/14.7;
  `docs/adr/0001-*`, `docs/adr/0007-*`, `docs/adr/0012-*`;
  `docs/SPEC-DEVIATIONS.md` (S-PERS-12/13, S-ALG-CU-11, S-SEM-04, S-IMP-01…03);
  `docs/04-MULTI-NODE.md` IN-05 and wait section; `docs/EXAMPLES.md` List/wait sections.

### Tertiary (LOW confidence)

- None. No claim in this document rests on training data alone; the design constants and
  shapes are marked as discretion/proposals above.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versions read from `go.mod`, the generated headers, and the tool
  binary this session.
- Current-state map: HIGH — every behavior claim cites a file:line and a verbatim quote
  read this session.
- Architecture/design: MEDIUM — the recommendations are sound but the page sizes, keepalive
  values, token shape and `go_package` name are discretion choices.
- Pitfalls: HIGH — each pitfall is tied to a code path, a test, or a probe result.
- Validation: HIGH for the commands/framework; MEDIUM for the exact test names (the planner
  may split/rename).

**Research date:** 2026-11-22
**Valid until:** ~2026-12-22 for the code-state claims (fast-moving phase); the gRPC/buf
facts are version-pinned and stable while `go.mod` pins them.


