# Phase 3: Transport - Pattern Map

**Mapped:** 2026-11-21
**Files analyzed:** 31 (20 production/CI files, 11 test files)
**Analogs found:** 31 / 31 (five concerns with no direct analog are listed under "No Analog Found")

This map is built from the CURRENT tree (post-Phase-2 line numbers — they differ from the review's citations). Every analog path below is git-tracked source; the `docs/reviews/**/evidence/repro/**` copies are frozen evidence and are NOT analogs (never edit or copy from them).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/app/repl/core.go` | core state machine | event-driven (timer loop) | itself (Phase-2 core) | exact (self) |
| `internal/udprepl/repl.go` | transport adapter | request-response + streaming read | itself; timer-loop shape `internal/app/persist/engine.go` | exact (self) |
| `internal/app/storage/store.go` | model/store | CRUD (in-memory map) | itself (`Get`/`Records`) | exact (self) |
| `internal/grpcapi/wait.go` | service helper | event-driven wait | `internal/app/persist/engine.go` waiter/`await` | role-match |
| `internal/grpcapi/gate.go` | options/resolver | event-driven poll | itself | exact (self) |
| `internal/grpcapi/server.go` | controller | request-response | itself (`Set` + `mapError`) | exact (self) |
| `internal/port/repl.go` | port contract | event-driven results | itself (`Outbox`) | exact (self) |
| `internal/app/version/version.go` | domain utility | transform/validate | itself (`Unpack`, layout consts) | exact (self) |
| `internal/app/persist/engine.go` | facade/service | CRUD + async dump | itself | exact (self) |
| `internal/app/persist/codec.go` | codec/utility | transform | itself | exact (self) |
| `internal/persistfile/fsops.go` | adapter | file-I/O | itself | exact (self) |
| `internal/persistfile/fsync_darwin.go` | adapter | file-I/O | — **DELETE** | n/a |
| `internal/persistfile/fsync_other.go` | adapter | file-I/O | — **DELETE** | n/a |
| `internal/app/internal/simnet/simnet.go` | test harness | event-driven | itself (`nodeStore`, `Write`, `Delete`) | exact (self) |
| `internal/metrics/metrics.go` | exporter | transform | itself (`Sources`/`trafficFamilies`) | exact (self) |
| `cmd/sup-kv/main.go` | composition root | startup/request-response | itself (`registerConfig`/`wire`) | exact (self) |
| `scripts/red-runner` | CI script | batch | itself (mutation stage) | exact (self) |
| `scripts/red-expectations.tsv` | config data | batch | itself | exact (self) |
| `.github/workflows/scheduled.yml` | CI config | batch | existing `red-run` job | role-match |
| `Makefile` | build config | batch | existing `race-soak` target | exact (self) |
| `internal/app/repl/bench_test.go` (NEW) | test | benchmark | `internal/app/persist/bench_test.go` | role-match |
| `internal/app/repl/egress_test.go` (NEW) | test | unit | `internal/app/repl/sync_test.go` | exact (role+flow) |
| `internal/app/version/validate_test.go` (NEW) | test | unit/table | `internal/app/version/version_test.go` | exact (role+flow) |
| `internal/app/storage/view_test.go` (NEW) | test | unit/table | `internal/app/storage/store_test.go` | exact (role+flow) |
| `internal/udprepl/socket_internal_test.go` (NEW) | test | unit (fake socket) | `internal/udprepl/syncsize_internal_test.go` | partial (white-box, no fake conn exists) |
| `internal/udprepl/decode_negative_test.go` | test | table | itself | exact (self) |
| `internal/app/repl/discard_test.go` | test | table/totality | itself | exact (self) |
| `internal/grpcapi/wait_test.go` | test | integration fakes | itself + `engine_blocking_test.go` | role-match |
| `cmd/sup-kv/main_test.go` | test | exec/table | itself | exact (self) |
| `internal/metrics/metrics_test.go` | test | family table | itself | exact (self) |
| `internal/stand/stand_test.go` (+ new stand file) | test | exec/real socket | itself (`TestStand_oversizeKeyF2c`) | exact (self) |

## Pattern Assignments

### `internal/app/repl/core.go` (core, event-driven)

This is the phase's biggest rewrite. The analog is the current file: preserve its doc-comment discipline, its "results, never callbacks" contract, and its deterministic sorted-iteration rule; replace the state shapes and scan loops.

**Imports / package boundary** (lines 14-27):
```go
package repl

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)
```
`internal/app/**` may not import `net`, `os`, `syscall`, `google.golang.org/*` (AGENTS depguard rule). `container/heap` is stdlib and legal here.

**Config normalization — the pattern for every new knob** (lines 136-168):
```go
func (c Config) normalized() Config {
	if c.HelloInterval <= 0 {
		c.HelloInterval = defaultHelloInterval
	}
	// ...
	if c.SyncResponseRate <= 0 || c.SyncResponseRate > maxSyncResponseRate {
		c.SyncResponseRate = defaultSyncResponseRate
	}
	if c.Log == nil {
		c.Log = func(_ string) {}
	}
	if c.OnDiscard == nil {
		c.OnDiscard = func(_, _ string) {}
	}
	return c
}
```
New `MaxClockLead time.Duration` follows the same shape (zero/negative → `defaultMaxClockLead = 60 * time.Second`).

**Consumer-side store interface to SHRINK** (lines 170-187):
```go
type StateStore interface {
	ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error)
	Records() map[string]storage.Record // DataSync compare source
}
```
Target: `ApplyRemote`, `Version(key) (version.RecordVersion, bool)`, `Lookup(key) (storage.Record, bool)`, `Versions() map[string]version.RecordVersion`; **drop `Records`** (RESEARCH Pattern 4 / Anti-Patterns).

**Pacer to clone verbatim** — `enqueueResponse` + `drainEgress` (lines 646-678). This is the exact per-send spacing arithmetic D-01/RESEARCH says to reuse for the two new queues:
```go
func (c *Core) enqueueResponse(now time.Time, handle string, msg port.Message) {
	if len(c.egress) == 0 {
		c.nextSendAt = now
	}
	c.egress = append(c.egress, queuedUnicast{handle: handle, msg: msg})
}

func (c *Core) drainEgress(now time.Time, out *port.Outbox) {
	for len(c.egress) > 0 && !now.Before(c.nextSendAt) {
		q := c.egress[0]
		c.egress = c.egress[1:]
		out.Unicast(q.handle, q.msg)
		if now.After(c.nextSendAt) {
			c.nextSendAt = now
		}
		c.nextSendAt = c.nextSendAt.Add(c.syncResponseInterval)
	}
}
```
`syncResponseInterval = time.Second / time.Duration(cfg.SyncResponseRate)` is computed once in `New` (line 342) after the `maxSyncResponseRate` clamp (line 158) — new pacers MUST reuse this value, never recompute from raw config (RESEARCH Pitfall 22).

**State to replace (D-03)** — `pendingEntry` retains a record today (lines 255-273) and is keyed by bare `key` (line 289):
```go
type pendingEntry struct {
	rec       storage.Record
	version   version.RecordVersion
	waitSet   map[uint16]struct{}
	acked     []uint16
	nextRetry time.Time
	backoff   time.Duration
	createdAt time.Time
}
```
Target: key `pending map[pendingKey]pendingEntry` with `pendingKey{key string; version version.RecordVersion}`; drop `rec`; add `peerUnacked map[uint16]map[pendingKey]struct{}`, an indexed due-heap, and an insertion-ordered list for O(1) oldest-age.

**Sorted-iteration discipline** (no map order reaches an emit path) — lines 499-506, 774-776, 876-878, 1028-1035:
```go
for _, key := range slices.Sorted(maps.Keys(c.pending)) {
	entry := c.pending[key]
	// ...
}
```
The due-heap must tie-break on `(nextRetry, seq)` deterministically (the `simnet` `eventHeap` shape — `internal/app/internal/simnet/simnet.go` scheduler around lines 80-120). Never `math/rand` in the core; jitter is a hash of `(runID, counter, salt)` (RESEARCH Pitfall 21).

**Timer/deadline contract to preserve** — `Advance` (lines 724-734) and `nextDeadline` (895-912):
```go
func (c *Core) Advance(now time.Time, out *port.Outbox) time.Time {
	if !now.Before(c.nextHello) {
		out.Broadcast(port.Hello{})
		c.nextHello = now.Add(c.cfg.HelloInterval)
	}
	c.syncListing(now, out)
	c.evictPeers(now)
	c.scanRetransmit(now, out)
	c.drainEgress(now, out)
	return c.nextDeadline(now)
}
```
Every new queue contributes its `nextSendAt` to `nextDeadline` via `min`; the returned deadline is always strictly future. `nextSync` must advance even when a listing enqueue is skipped, or a short `sync_interval` livelocks (RESEARCH Pitfall 3).

**Discard classification to extend (D-06)** — lines 39-45, 375-415. Add tokens and counters in exactly this shape:
```go
const (
	DiscardBadCRC          = "bad_crc"
	DiscardUndecodable     = "undecodable"
	DiscardBadProtoVersion = "bad_proto_version"
	DiscardOwnBroadcast    = "own_broadcast"
	DiscardDuplicateNodeID = "duplicate_node_id"
)
// ...
if in.ProtoVersion != SupportedProtoVersion {
	c.discards.BadProtoVersion++
	c.cfg.OnDiscard(DiscardBadProtoVersion, in.Source)
	return
}
```
New tokens: `empty_key`, `key_too_long`, `tombstone_with_data`, `bad_checksum`, `bad_version`. Insert the new validation block after the existing classifier and **before `handleMessage`** (line 414) so no store/clock call can precede the verdict. Details passed to `OnDiscard` must not carry key names (T-6-05).

**Error/log pattern** — `applyRemote` (lines 523-543) is the model for "never panic, never confirm what the store refused":
```go
conflicts, drift, err := c.store.ApplyRemote(msg.Key, msg.Record)
if err != nil {
	c.cfg.Log(fmt.Sprintf("sup-kv: applying replicated record for key %q from handle %q: %v", msg.Key, source, err))
	return // Not applied: no Ack, no supersession.
}
```

**Ack-drain + wait-set discipline** (lines 567-579) — keep the empty-peer-table guard; emit the new `Outbox` results (`Ack`/`Drained`) at the same points:
```go
entry, ok := c.pending[ack.Key]
if !ok || entry.version != ack.Version {
	return
}
delete(entry.waitSet, peerID)
entry.acked = ackedInsert(entry.acked, peerID)
if len(entry.waitSet) == 0 && len(c.peers) > 0 {
	delete(c.pending, ack.Key)
	return
}
```

**Accessor/mirror pattern to make O(1)** — `UnackedStats` (lines 1055-1073) currently walks the whole map; `PeerSnapshot` (995-1005) allocates per call. Keep the copying-accessor style but back the gauges with maintained counters and add `PeerCount()`.

**Current listing/retransmit call sites that change**:
- `syncListing` burst emission — lines 751-759.
- `listingChunks()` `Records()` walk — lines 771-778.
- discovery flush inside `handleHello` — lines 499-506 (10 000 broadcasts in one call).
- `scanRetransmit` sorted full walk — lines 872-887.
- `handleAck`/`applyRemote` supersession — lines 539-541, 574-577.
- `handleDataRequest` whole-store copy — line 606 (`c.store.Records()[key]`).
- `handleDataSync` whole-store copy — line 633 (`snapshot := c.store.Records()`; must become per-entry `Version(key)`, RESEARCH Pitfall 17).

---

### `internal/udprepl/repl.go` (transport adapter, request-response + streaming)

**Adapter struct + mirror set** (lines 137-207) — this is the seam the phase removes/replaces. Note the existing atomics naming and the "mirrored after every loop iteration" contract:
```go
type Adapter struct {
	conn          *net.UDPConn
	broadcastAddr *net.UDPAddr
	nodeID        uint16
	runID         [16]byte
	core          *repl.Core
	// ...
	badCRC          atomic.Uint64
	undecodable     atomic.Uint64
	badProtoVersion atomic.Uint64
	ownEcho         atomic.Uint64
	duplicateNodeID atomic.Uint64
	peerCount       atomic.Uint64
	unackedCount      atomic.Int64
	unackedOldestNs   atomic.Int64
	mergeConflicts    atomic.Uint64
	mergeDriftWarning atomic.Uint64
	retransmitted     atomic.Uint64
	// ...
	pendingCopy atomic.Pointer[map[string]repl.PendingInfo]
}
```

**Socket construction to replace (D-02, D-07)** — `New` (219-250) and `reuseAddrBroadcastListenPacket` (266-291):
```go
func reuseAddrBroadcastListenPacket(bindAddr string) (net.PacketConn, error) {
	var opErr error
	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if opErr != nil { return }
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			})
			// ...
		},
	}
	return lc.ListenPacket(context.Background(), "udp4", bindAddr)
}
```
Target: production `New` uses `net.ListenUDP`/`ListenConfig` with **no socket options** (Go sets `SO_BROADCAST` already); `SO_REUSEADDR` moves behind a test-only functional option `WithAddressReuse()` used by `internal/udprepl/smoke_test.go` (which binds two adapters to one wildcard port — `smokeSocket`, lines 60-90 replicates this recipe). Buffer setup belongs right after bind in `New`; read-back through `SyscallConn().Control` + `syscall.GetsockoptInt` per RESEARCH Pattern 7.

**Read loop to harden (D-07)** — current loop (491-505) exits on ANY error:
```go
func (a *Adapter) readLoop() {
	buf := make([]byte, datagramBufSize)
	for {
		n, src, err := a.conn.ReadFromUDP(buf)
		if err != nil {
			return // the socket closed — Run's exit path.
		}
		ev := inboundEvent{datagram: bytes.Clone(buf[:n]), source: src.String()}
		select {
		case a.inbound <- ev:
		case <-a.done:
			return
		}
	}
}
```
Target: `if errors.Is(err, net.ErrClosed) { return }`, else count + rate-limited log + `time.Sleep(readErrorBackoff)` + continue. Introduce an unexported `packetConn` interface (`ReadFromUDP`, `WriteToUDP`, `Close`) on `Adapter.conn` for the fake-conn test; `New` keeps `*net.UDPConn`.

**Event loop to simplify (D-04)** — `Run` (303-341) is the model for where `DrainResults` must be called and where the mirror calls are removed:
```go
case ev := <-a.inbound:
	in := decode(ev.datagram)
	in.Source = ev.source
	a.core.HandleDatagram(time.Now(), in, &out)
	// ... lastAck correlation ...
case lw := <-a.notify:
	a.core.NotifyLocalWrite(time.Now(), lw.key, lw.rec, &out)
case <-timerChan(timer):
}
next = a.core.Advance(time.Now(), &out)
a.flush(&out)
a.mirrorStats()
a.mirrorState()
```
After the change: `a.flush(&out)` then `a.drainResults(&out)` then the reduced `a.mirrorStats()` — **never** `mirrorState()`/`PendingCopy`/`retainDrained` (delete lines 584-666, the `drainedRow` type 108-127, the `lastAck`/`prevPending`/`drained` fields 188-206, and `drainedRetention` 64-77).

**Encode/flush pattern to keep** — `flush` (515-534), `countSent` (539-552), `write` (555-560). Counting stays post-encode, pre-send; do not add a pre-enqueue "sent" counter (OBSV-03 is Phase 7).

**Timer pattern to keep** — `armTimer`/`timerChan`/`stopTimer` (671-696), Go 1.23+ semantics, no drain.

**Datagram size arithmetic** — `datagramBufSize = 65507` (line 50) is the constant the startup ceiling check and the send-buffer floor reuse. `encodeDatagram` (702-723) is the real codec oracle for the "largest accepted record fits" check.

---

### `internal/app/storage/store.go` (store, CRUD)

**Get — the locking pattern to copy for the new reads** (lines 139-153):
```go
func (s *Store) Get(key string) (Record, error) {
	s.mu.RLock()
	rec, ok := s.records[key]
	s.mu.RUnlock()
	if !ok || rec.Deleted {
		return Record{}, ErrNotFound
	}
	return Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Checksum: rec.Checksum,
	}, nil
}
```
`Get` hides tombstones, so it cannot serve reconciliation. Add `Version(key)` (RLock, no Data copy), `Lookup(key)` (RLock, tombstone-visible, clone Data), `Versions()` (RLock, shallow `map[string]version.RecordVersion`) exactly as RESEARCH Pattern 4 shows. `Lookup` is what `handleDataRequest` and the sync-newer reply use; `Versions` is only for the listing.

**Records — the anti-pattern being demoted** (lines 271-287):
```go
func (s *Store) Records() map[string]Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]Record, len(s.records))
	for key, rec := range s.records {
		out[key] = Record{
			Version:  rec.Version,
			Data:     bytes.Clone(rec.Data),
			Deleted:  rec.Deleted,
			Checksum: rec.Checksum,
		}
	}
	return out
}
```
Keep `Records()` for the dump/watch/facade paths; remove it from `repl.StateStore`.

**Key-limit placement** — sentinel errors live in `internal/app/storage/errors.go` (lines 7-19) with `errors.Is` matching. Add `MaxKeyBytes = 1024` beside the `defaultValueMaxBytes` constant (store.go lines 13-18) and `ErrKeyTooLong` in `errors.go`; `Set`/`SetIfAbsent` validate it before taking the lock, mirroring the `len(data) > s.cfg.ValueMaxBytes` check (155-167).

**Single convergence point** — `install` (252-269) must stay the only writer, so the change tap and checksum stay honest.

---

### `internal/grpcapi/wait.go` (service helper, event-driven wait)

The current polling implementation is the anti-pattern (D-04). Keep its **externally observable semantics**, replace the mechanism.

**Current constants to delete** (lines 12-54): `waitPoll = 5 * time.Millisecond`, `waitAppear = 500 * time.Millisecond`. Keep `defaultWaitTimeout`, `maxWaitTimeoutMs`, `waitTimeoutCeilingFactor` and `waitTimeout` (210-225) unchanged — those are WR-02 hardening and still apply.

**Current `waitForReplication` algorithm doc** (lines 56-93) is the behavior spec to re-express over the new seam: `total` captured once from `KnownPeers`; `total == 0` → `(true,0,0)`; success only at `min(len(acked), total) == total`; superseded/evicted → honest false; timeout/cancel → last observed state with `acked=false`; count never exceeds the write-time total.

**Best in-repo analog for the new mechanism — the engine waiter** (`internal/app/persist/engine.go` lines 47-56):
```go
type waiter struct {
	ver  version.RecordVersion
	done chan struct{}
	err  error
}
```
and its close-once fan-out (`confirmWaiters`, 347-368; `await`, 443-472):
```go
w := &waiter{ver: ver, done: make(chan struct{})}
e.mu.Lock()
// ...
e.waiters = append(e.waiters, w)
e.mu.Unlock()
<-w.done
return w.err
```
The adapter tracker should copy this: mutex-guarded map `map[trackerKey]tracker`, results applied under the lock, waiter returned with `(acked bool, ackedCount uint32)` on timeout/channel close. **Do not** copy `drainedRetention` (udprepl lines 64-77, 635-666) — the tracker-at-notify removes the appearance race that retention worked around.

**Server seam to change** (`internal/grpcapi/gate.go` lines 46-83): `ServerOptions.PendingCopy func() map[string]repl.PendingInfo` is replaced by a `WaitReplicated` function seam. `KnownPeers` stays (write-time total capture is still required). Keep the "nil seam = single-node" doc contract (D-11).

---

### `internal/grpcapi/server.go` (controller, request-response)

**Validation order to extend (XPORT-03)** — `Set` (131-157) and `Delete` (171-189):
```go
if s.gated() {
	return nil, gateUnavailable()
}
if req.GetKey() == "" {
	return nil, status.Error(codes.InvalidArgument, "key must not be empty")
}
```
Add the key-length check immediately after the empty-key check (before any engine call). Keep gate-first ordering: while gated, even an invalid key surfaces `Unavailable`.

**Error mapping** — `mapError` (291-307) matches sentinels with `errors.Is`; add `storage.ErrKeyTooLong` → `codes.InvalidArgument` there rather than at each handler.

**Notify seam change** — `notifyLocalWrite` (263-289) and `LocalNotify` (33-38):
```go
type LocalNotify func(key string, rec storage.Record) error
```
When the tracker-at-notify lands, this becomes `func(key string, rec storage.Record, track bool) error` (RESEARCH Pattern 5). Call sites: `Set` line 150 (`false` today) and `Delete` line 182; the adapter's method must register the tracker **before** enqueueing when `track` is true.

---

### `internal/port/repl.go` (port contract, event-driven results)

**Outbox pattern to extend** (lines 114-140):
```go
type Outbox struct {
	outbound []Outbound
}

func (o *Outbox) Broadcast(msg Message) { /* append Outbound{Broadcast: true, Msg: msg} */ }
func (o *Outbox) Unicast(handle string, msg Message) { /* ... */ }
func (o *Outbox) Drain() []Outbound {
	out := o.outbound
	o.outbound = nil
	return out
}
```
Add a parallel `results []Result` slice + `Ack`/`Drained`/`Superseded`/`WaitSetEmpty` methods + `DrainResults() []Result`. Keep `Drain()`'s meaning untouched so all existing tests keep passing. Define `Result` with a `Kind` and `(Key, Version, Peer)` fields; keep the struct plain data (no behavior) as the rest of the port package. Add the `version` import (already present).

**Commitment:** the package allows stdlib + own module only (`sup-kv-port-stdlib-only`), so no protobuf/net here.

---

### `internal/app/version/version.go` (domain utility, validate)

**Layout constants + Unpack** (lines 23-32, 49-58) are the single source for the validation masks:
```go
const (
	uuidVerBits = 0x70 // version 7 nibble in octet 6.
	uuidVarBits = 0x80 // variant 0b10 in the top two bits of octet 8.
	// ...
)

func (v RecordVersion) Unpack() (l int64, counter uint16, nodeID uint16) {
	l = int64(binary.BigEndian.Uint64(v[0:8]) >> lPadding)
	counter = binary.BigEndian.Uint16(v[6:8]) & counterMask
	nodeID = binary.BigEndian.Uint16(v[8:10])<<nodeLsbDrop | uint16(v[10]>>fillerShift)
	return l, counter, nodeID
}
```
Add `func (v RecordVersion) Validate(now time.Time, maxLead time.Duration) error` in this file, using `v[6]&0xF0 != uuidVerBits || v[8]&0xC0 != uuidVarBits` for layout and the subtraction form `l >= nowMs && l-nowMs > maxLeadMs` for the future bound (RESEARCH Pattern 6 — the subtraction avoids overflow). Do not reject the 46 filler bits (Open Question 9). Sentinels go in this package (there is no `errors.go` here today) or as documented sentinel values; keep `Unpack` total.

---

### `internal/app/persist/engine.go` + `internal/app/persist/codec.go` (facade, dead-code removal)

**Dead code to delete (XPORT-13)**:
- `lastLocal` field — engine.go line 72; update site lines 457-459; capture line 294; above-cover branch 347-360. Also delete `TestConfirmWaiters_aboveCoverInvariant` (`internal/app/persist/engine_internal_test.go:73-...`) or repurpose it. **Proof obligation** (RESEARCH Pitfall 15): `await` raises the watermark and appends the waiter in the same critical section (lines 450-465), and `runDump` captures the waiter list before `Encode(e.store.Records())` (292-304) — state this in the plan/summary.
- Exported `persist.Parse` — codec.go lines 153-161:
```go
func Parse(dump []byte) error {
	_, _, err := Decode(dump)
	return err
}
```
Only `codec_test.go` uses it (lines 357, 403, 512, 520); `Open` uses its own Decode closure; `engine.go:122` mentions it in a comment. Tests call `persist.Decode` and assert its error.
- Forward the three new reads on the facade, mirroring the existing pass-through style:
```go
func (e *Engine) Get(key string) (storage.Record, error) {
	return e.store.Get(key)
}

func (e *Engine) Records() map[string]storage.Record {
	return e.store.Records()
}
```
Add `Version`, `Lookup` (tombstone-visible, unlike `Get` — RESEARCH Pitfall 18), `Versions`.

**owned boundary:** `internal/app/persist` stays within the app package set; no new I/O imports.

---

### `internal/persistfile/fsops.go` (adapter, file-I/O) + deletions

**Replace `syncDurable` with `File.Sync`** at both call sites (lines 96-107, 145-152):
```go
func (o *fileFSOps) fsync() error {
	err := syncDurable(o.tmp)
	if err == nil {
		err = o.tmp.Close()
	}
	// ...
}
```
becomes `o.tmp.Sync()`. `syncDir` likewise `d.Sync()`. Then **delete** `fsync_darwin.go` (lines 23-29 hand-rolled `F_FULLFSYNC`) and `fsync_other.go` (lines 11-13) — `os.File.Sync` already issues `F_FULLFSYNC` with EINTR/ENOTSUP handling on darwin (Go 1.27 `internal/poll/fd_fsync_darwin.go`), so the helper is strictly worse. Update the `fsops` interface comments (lines 28-34, 50-54) and `docs/SPEC-DEVIATIONS.md` S-PERS-03; note the Phase 5 P-09 discharge.

**`ownHandle` removal** is in `core.go`: field line 285, constructor parameter line 332, assignment line 337; never read elsewhere. There are 13 non-evidence `repl.New(...)` call-site files (see inventory below); update them mechanically. `udprepl.New(...)` (adapter) is a different constructor and is unaffected.

---

### `internal/app/internal/simnet/simnet.go` (test harness, event-driven)

**StateStore adapter must gain the new methods** (lines 429-440):
```go
type nodeStore struct{ store *storage.Store }

func (s nodeStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

func (s nodeStore) Records() map[string]storage.Record { return s.store.Records() }
```
Add `Version`/`Lookup`/`Versions` delegations; drop `Records` if the interface drops it.

**Write's swallowed error + whole-store copy (WR-02)** — lines 710-724:
```go
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
	now := c.sched.Now()
	_, _, _ = c.nodes[i].store.Set(key, data)      // swallows ErrValueTooLarge
	rec := c.nodes[i].store.Records()[key]         // full-store copy per write
	var out port.Outbox
	c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
	c.route(i, out.Drain())
	c.drive(i, now)
	c.writes = append(c.writes, LoggedWrite{Key: key, Rec: rec})
	return rec
}
```
Mirror `Delete`'s error path (741-754). Target: `if _, _, err := ...Set(...); err != nil { panic(fmt.Sprintf("simnet: Cluster.Write: Set %q on node %d: %v", key, i, err)) }` and `rec, _ := c.nodes[i].store.Lookup(key)`.

**Results must be drained in simulation too** — `deliver` (649-657) and `drive` (678-696) call `out.Drain()` and route; add `out.DrainResults()` after every handler/Advance (the ack seam is otherwise benign but the Outbox would accumulate — RESEARCH Pitfall 11).

**Quiescence must include the new queues** — `chaosQuiescent` (`internal/app/repl/sim_chaos_test.go:449-459`) currently checks `UnackedStats(now).Count == 0 && QueuedResponses() == 0`; add a total-queue accessor (RESEARCH Pitfall 23).

---

### `internal/metrics/metrics.go` (exporter, transform)

**Sources + Snapshot shape** (lines 176-203):
```go
type Sources struct {
	Discards      func() repl.DiscardStats
	KnownPeers    func() uint64
	Unacked       func() repl.UnackedStats
	Sent          func() udprepl.SentByType
	Retransmitted func() uint64
	MergeStats    func() repl.MergeStats
	DumpStats     func() persist.DumpStats
}

func Snapshot(s Sources) []Family {
	families := replicationFamilies(s.Unacked(), s.KnownPeers(), s.Retransmitted(), s.MergeStats())
	families = append(families, trafficFamilies(s.Discards(), s.Sent())...)
	families = append(families, dumpFamilies(s.DumpStats())...)
	return append(families, runtimeFamilies()...)
}
```
Add new families by extending this table, not by a registry. The label vocabulary is fixed and zero-count samples are always rendered — extend `trafficFamilies` (264-289) for the new discard classes, and add paced/queue/read-error families alongside `replicationFamilies` (208-257). Names are provisional per Open Question 8 (`supkv_replication_read_errors_total`, `supkv_replication_datagrams_paced_total`, `supkv_replication_send_queue`, `supkv_replication_egress_dropped_total`).

**Family test to extend** — `internal/metrics/metrics_test.go` `families` table (186-206), `zeroSources()` (210-220), `TestMetrics_families` zero lines (227-264). Every new family needs a `# TYPE` entry, a zero sample line, and a zeroSources seam.

---

### `cmd/sup-kv/main.go` (composition root, startup)

**Flag/env twin pattern** (lines 91-131, 268-347). Every knob is a flag whose default is its `SUPKV_` env twin:
```go
const (
	envValueMaxBytes      = "SUPKV_VALUE_MAX_BYTES"
	envSyncChunkBytes     = "SUPKV_SYNC_CHUNK_BYTES"
	// ...
)

flag.DurationVar(&cfg.driftWarn, "drift-warn", envDur(envDriftWarn, defaultDriftWarn),
	"clock drift warning threshold (env SUPKV_DRIFT_WARN)")
```
Add `--max-clock-lead` / `SUPKV_MAX_CLOCK_LEAD` the same way (`envDur`, default 60s), then pass `MaxClockLead` in `newReplicationAdapter` (505-523).

**Startup validation / exit-2 pattern** (66-73, 205-232):
```go
const (
	exitDumpFailed = 1
	exitUsage      = 2
)
// main:
err := finalizeConfig(&cfg, *nodeID, envErr)
if err != nil {
	fmt.Fprintf(os.Stderr, "sup-kv: %s\n", err)
	flag.Usage()
	os.Exit(exitUsage)
}
setupLogging(cfg)
node, listener, err := wire(cfg)
```
The "largest accepted record exceeds 65507 → exit 2" check belongs in `finalizeConfig` (379-393) or the `wire`/adapter construction path, and must go through the real codec (`replproto` marshal of a worst-case fixture) — RESEARCH Code Examples "Startup ceiling check". Do not reject `--value-max-bytes=64000` (RESEARCH Pitfall 19; P-01 red tests use it).

**Bind-address WARN** (D-07): after `setupLogging`, before `wire`, parse `cfg.bindAddr` with `net.SplitHostPort`; a non-wildcard host logs one WARN naming the broadcast-reception limitation. `wire` (429-491) shows the construction order and the compensation path (`abortWire` 496-503) to follow if the new startup check fails after sockets exist.

**Adapter options** (505-523): `udprepl.New` gains functional options; keep `newReplicationAdapter` the single production construction point.

---

### `scripts/red-runner` + `scripts/red-expectations.tsv` (CI gate, batch)

**Mutation-stage structure that must be generalized** (lines 74-101, 103-127, 161-226). The current mutant hardcodes one source:
```bash
MUTANT_PACKAGE="./internal/app/repl/"
MUTANT_TEST='^TestChaos_convergence$'
MUTANT_SOURCE="$ROOT/internal/app/repl/core.go"
MUTANT_CORPUS="$ROOT/internal/app/repl/testdata/chaos-seeds.txt"
```
and one mutator sed targets the exact Phase-2 line:
```bash
sed 's|entries = append(entries, port.SyncEntry{Key: key, Version: snapshot\[key\].Version})|if !snapshot[key].Deleted { entries = append(entries, port.SyncEntry{Key: key, Version: snapshot[key].Version}) }|' "$1" >"$2"
```
A rewritten `listingChunks` using `Versions()` has no `Deleted` field, so re-point this mutant at `internal/app/storage/store.go` (`Versions` skipping tombstones) and generalize `MUTANT_SOURCE`/overlay JSON (line 174) to a per-mutant source. Keep the loud failures intact: sed no-op → exit 1 (170-173), non-compiling mutant → exit 1 (175-178), timeout ≠ kill (186-189), post-stage `git status` clean check (203-206, 221-224). Re-verify the committed kill seeds.

**Expectation flip is one-line only (D-01)** — `scripts/red-expectations.tsv` rows 24-30; change only `TestRed_R01_pacedSync20k`, `TestRed_R02_unackedDrain`, `TestRed_R02_burst` from `red` to `green`. Never edit the red test bodies, never touch the runner's flip logic.

**20× evidence job** — copy the `red-run` job shape in `.github/workflows/scheduled.yml` (lines 79-110): `runs-on: ubuntu-latest`, explicit `timeout-minutes` (120 per RESEARCH A5), `actions/checkout@v7` + `actions/setup-go@v7` with `go-version-file: go.mod`, `env: SUPKV_STAND_NIGHTLY: "1"`, and an evidence-summary step with `if: always()`. Command: `SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 go test -race -count=20 ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'`. Make `race-soak` (Makefile lines 57-58) gain `./internal/stand/` only after measuring cost (RESEARCH Pitfall 25).

---

### Test files

All tests are vanilla `testing`, table-driven, no assertion libraries, `t.Parallel()` unless wall-clock-bound (the stand file doc explains why red measurements opt out).

**`internal/app/storage/view_test.go` (NEW)** — copy `TestStore_Get`'s subtest style (`store_test.go:179-220`) and the shared `newStore()` helper. Cases: tombstone visible in `Lookup` but not `Get`; `Version` matches `Records()[k].Version` incl. tombstones; `Versions` is a snapshot (mutating the returned map doesn't affect the store) and consistent with `Records`; concurrent hammer shape (`TestStore_concurrentHammer`, line 716).

**`internal/app/version/validate_test.go` (NEW)** — copy `version_test.go`'s fixed-vector/table style (fixtures lines 14-31). Cases: valid v7 passes; wrong version nibble; wrong variant; exactly `now+lead` valid; `now+lead+1ms` invalid; far-future `ff…ff` invalid at any lead; absurd/negative lead normalizes to 0.

**`internal/app/repl/egress_test.go` (NEW)** — copy `sync_test.go`'s pacing assertions (`QueuedResponses()` checks at 98-244, 646-700). Cases: two listings with the same 1000 keys leave 1000 queued (dedupe by `(handle, kind, key)`); cap drops oldest and deletes its dedupe-map entry (RESEARCH Pitfall 10); single-entry listing still emits; listing-busy skip still advances `nextSync`.

**`internal/app/repl/bench_test.go` (NEW)** — copy `internal/app/persist/bench_test.go`'s shape: build the store/core once, `b.ResetTimer`, one operation per iteration. Benchmarks: `BenchmarkCoreEventLoop` with 10 vs 10 000 unacked (assert ratio manually in a test or report via `b.ReportMetric`), `BenchmarkHandleDataRequest` and `BenchmarkHandleDataSync` at 1 000 vs 50 000 keys. Use `internal/testutil/bench.go` if it already has fixture builders (check before adding).

**`internal/udprepl/socket_internal_test.go` (NEW)** — white-box `package udprepl`, following `syncsize_internal_test.go`'s file-doc style. No fake-socket analog exists (see No Analog Found). Build a tiny `packetConn` fake returning a non-`net.ErrClosed` error once, then a datagram; assert the loop continued, `readErrors` incremented, and the datagram reached `inbound`. Buffer test: open a real loopback `*net.UDPConn`, apply the new setup helper, assert read-back logged/accessor returns > 0 and the send floor passes; never assert exact kernel values (doubled/clamped — RESEARCH Pitfall 13).

**`internal/udprepl/decode_negative_test.go` (MODIFY)** — the rows to flip are 190-244 (`oversize_datagram_over_udp_ceiling_decodes_today`, `empty_key_decoded_today_xport_10_forward_reference`) and 246-270 (tombstone-with-data). Their comments already name the Phase-3 flip; update `wantStatus`/`wantClass` and the `check` assertions. Add rows for `bad_version` (layout and future) and `bad_checksum`.

**`internal/app/repl/discard_test.go` (MODIFY)** — extend `TestDiscard_classes` (45-...) with the five new classes, and teach `TestDiscard_totality` (367-...) the new class domain; `testutil.RandRecord` produces zero checksums (`internal/app/internal/testutil/gen.go:51-62`), so either compute checksums in the generator or count them as `bad_checksum` (RESEARCH Pitfall 16). Also give `TestScenario_farAhead` (24 h offset, `sim_scenario_test.go:200`) and `TestScenario_clockJumps` (`:118`) an explicit large `repl.Config.MaxClockLead` so their intent survives.

**`internal/grpcapi/wait_test.go` (MODIFY)** — rewrite `fakeMirror` (lines 22-71) as a `fakeTracker`/`WaitReplicated` fake. Keep the existing test names and semantic assertions (`TestWait_acksArrive`, `TestWait_timeout`, `TestWait_evictionNotAck`, `TestWait_superseded`, `TestWait_strangerAcksClampedToTotal`, `TestWait_overflowingTimeoutClamped`, `TestWait_perRequestTimeout`) — they are the behavior contract. `newWaitServer` (77-87) is the wiring point.

**`cmd/sup-kv/main_test.go` (MODIFY)** — follow existing exec/table tests: `--value-max-bytes` exit 2 through the real binary; `--max-clock-lead` env twin; bind-address WARN capture from stderr. For the second-instance EADDRINUSE test, reuse the free-port helper (`internal/udprepl/smoke_test.go:41-58`) or the stand's private port window.

**`internal/metrics/metrics_test.go` (MODIFY)** — extend `families` (186-206), `zeroSources` (210-220), and the zero-line list (239-259) exactly as described above.

**`internal/stand/stand_test.go` (MODIFY) / new stand file** — copy `TestStand_oversizeKeyF2c` (549-...) for the exec/real-socket shape; platform-tagged tests go in `stand_nondarwin_test.go` (build-tag precedent, line 36). Add: buffers logged + send floor; 1/20/60 KiB delivery on Linux; second-instance EADDRINUSE. Test-only same-port adapters need `WithAddressReuse()`.

## Shared Patterns

### Architecture boundary (lint-enforced)
**Source:** `.golangci.yml` rules `sup-kv-app-no-io` and `sup-kv-port-stdlib-only` (lines ~68-88).
**Apply to:** every new file.
- `internal/app/**` (core, storage, version, persist, simnet, testutil) must not import `net`, `os`, `syscall`, `google.golang.org/*`; `$test` is exempt.
- `internal/port/**` allows `$gostd` + `github.com/Djarvur/sup-kv` only.
- All socket code stays in `internal/udprepl`; all wire validation decisions stay in `internal/app/repl`/`version`.

### Error handling
**Source:** `internal/app/storage/errors.go` (sentinel + `errors.Is`), `internal/grpcapi/server.go:291-307` (`mapError`), `internal/udprepl/repl.go:515-560` (log, never panic), `internal/app/repl/core.go:523-543` (never confirm a refused record).
**Apply to:** all new store/API/adapter paths. Sentinel errors, `%w` wrapping, no panics on remote input. Adapter send failures are logged; the core returns decisions.

### Logging and discard counters
**Source:** `internal/app/repl/core.go:39-45, 375-415`; `internal/udprepl/repl.go:252-264` (`logDiscard` is Debug by design); `cmd/sup-kv/main.go:517-521` (static slog msg + `detail` attr).
**Apply to:** all new discard classes and WARN sites. Per-class rate limiting uses a tiny `lastWarn time.Time` field in the core driven by the event `now` — not a global throttle utility (RESEARCH "Don't Hand-Roll"). Counter details never carry key names.

### Validation order
**Source:** `internal/grpcapi/server.go:131-157`; `internal/app/repl/core.go:375-414`.
**Apply to:** API writes and inbound datagrams. API: gate → empty key → key length → engine. Wire: framing verdict → proto version → own-echo/duplicate → empty key → key length → tombstone-with-data → checksum → version → handler. Version validation runs **before** `ApplyRemote` (the only `Clock.Update` is inside `storage.Merge`, `store.go:226-228`).

### Determinism
**Source:** `internal/app/repl/core.go:771-778, 872-887` (sorted emission); `internal/app/internal/simnet/simnet.go:441-451` (`simRunID` — no crypto); `internal/app/repl/sim_chaos_test.go:449-459` (quiescence).
**Apply to:** all core pacing/retransmit/jitter. No `math/rand`, no wall clock, no map-order emission; same-seed replay must stay byte-identical.

### Metrics
**Source:** `internal/metrics/metrics.go:176-203, 208-289`; `internal/metrics/metrics_test.go:186-264`.
**Apply to:** every new counter/gauge. Extend `Sources` + `Snapshot` + a family builder; always render zero-valued samples; add the family to the test table.

### Testing conventions
**Source:** `internal/app/storage/store_test.go:179-220` (subtests), `internal/app/version/version_test.go:14-40` (fixtures), `internal/app/repl/discard_test.go:45-...` (table with full-counter assertions), `internal/udprepl/syncsize_internal_test.go` (white-box cross-seam), `internal/udprepl/bench_test.go` / `internal/app/persist/bench_test.go` (benchmarks), `internal/stand/stand_test.go` (exec/real socket).
**Apply to:** all new tests. Vanilla `testing`; table-driven; `t.Parallel()` unless timing-bound; test-only infrastructure in `_test.go` or `internal/testutil`; no assertion libraries.

### Red-gate contract
**Source:** `scripts/red-runner` header (lines 3-44) + mutation stage (74-226); `scripts/red-expectations.tsv` header (1-23).
**Apply to:** the phase gate. Red test files stay byte-untouched; only TSV rows flip; mutation seds move with the code or the runner fails loudly.

## Signature-Change Call-Site Inventory

These are the mechanical edits the planner must allocate (verified in the current tree; `docs/reviews/**` repro copies are frozen evidence — **do not edit**):

| Change | Call sites |
|---|---|
| `repl.New` drops `ownHandle` | 13 non-evidence files: `internal/udprepl/{repl.go,repl_test.go,scenario_test.go,smoke_test.go,syncsize_internal_test.go}`, `internal/grpcapi/{bench_test.go,wait_integration_test.go}`, `internal/app/internal/simnet/simnet.go`, `internal/app/repl/{discard_test.go,retransmit_test.go,sync_test.go,restart_test.go}`, `cmd/sup-kv/main.go` |
| `repl.StateStore` drops `Records`, gains `Version`/`Lookup`/`Versions` | `internal/app/internal/simnet/simnet.go:429-440`; test adapters `internal/app/repl/retransmit_test.go:66-74`, `internal/udprepl/repl_test.go:48-56`, `internal/udprepl/smoke_test.go:185-191`, `internal/udprepl/syncsize_internal_test.go:44-52` |
| `LocalNotify` gains `track bool` | `internal/grpcapi/server.go:150,182,263`; `internal/udprepl/repl.go:358`; `cmd/sup-kv/main.go:453` |
| `ServerOptions.PendingCopy` → `WaitReplicated` | `internal/grpcapi/gate.go:69-76`; `internal/grpcapi/server.go`; `cmd/sup-kv/main.go:453-459`; `internal/grpcapi/wait_test.go` |
| `pending` key becomes `pendingKey` (internal) | no external call sites; `PendingSnapshot`/`PendingInfo` stay for tests/observability |

## No Analog Found

| File / concern | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/udprepl/socket_internal_test.go` fake `packetConn` | test | unit/fake socket | No test in the repo fakes a `net.PacketConn`; closest is the `fsops` seam comment (`persistfile/fsops.go:5-18`) and the white-box style of `syncsize_internal_test.go`. Use RESEARCH Pattern 8's interface and the `fsops` "default real, fake only in `_test.go`" rule. |
| Bounded deduplicated egress queue | core data structure | request-response | The current `egress []queuedUnicast` (`core.go:250-253, 297`) is an unbounded FIFO with no dedupe map; there is no bounded-queue analog in the repo. Follow RESEARCH Pattern 9 / "Don't Hand-Roll" (FIFO slice + `map[identity]struct{}` + identity stored per item). |
| Indexed due-time heap | core data structure | event-driven | No indexed heap exists. Closest shape is `simnet`'s scheduler `eventHeap` (around `simnet.go:80-120`); use `container/heap` with index tracking and `(nextRetry, seq)` tie-break (RESEARCH "Don't Hand-Roll"). |
| `max_clock_lead` config end-to-end | config | startup | No prior core knob has a rejection bound; follow the flag/env twin pattern (`main.go`) + `Config.normalized` pattern (`core.go`), but the semantics are new (RESEARCH Pattern 6). |
| 20× scheduled red-evidence job | CI config | batch | No 20× job exists; copy the `red-run`/`nightly-stand` job shape in `scheduled.yml` (role-match only). |

## Metadata

**Analog search scope:** `internal/app/**`, `internal/udprepl/**`, `internal/grpcapi/**`, `internal/persistfile/**`, `internal/metrics/**`, `internal/stand/**`, `internal/port/**`, `cmd/sup-kv/**`, `scripts/**`, `.github/workflows/**`, `Makefile`, `.golangci.yml`.
**Files scanned:** 31 analyzed; 26 read in full or targeted sections; 5 checked by grep/count only (`repl_test.go`, `smoke_test.go`, `scenario_test.go`, `restart_test.go`, `sim_*_test.go`).
**Tracked-source check:** every named analog verified under `git ls-files`.
**Pattern extraction date:** 2026-11-21.
