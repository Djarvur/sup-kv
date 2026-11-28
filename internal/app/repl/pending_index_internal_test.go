package repl

// White-box proofs for the per-key pending index and the deterministic
// PendingVisits seam (XPORT-02, PERF-05). They live in package repl because
// pendingByKey, the pending entries and the counter are unexported state no
// external test can assert, and because the differential must compare the
// index against a naive scan from the inside. The _internal_test.go suffix is
// the documented testpackage exemption.

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The visit gates' two set sizes: the XPORT-02/PERF-05 acceptance pair. The
// claim is only meaningful when the large size is big enough that a whole-set
// scan would be visible, so the gates assert exact zeros rather than a ratio.
const (
	pendingIndexSmall = 10
	pendingIndexLarge = 10_000
)

// pendingIndexFixture bundles one core with its real store and the fixed
// virtual instant every fixture operation runs at — the in-package shape of
// listedStateStore/newConfigProbeCore, extended with the store handle the
// driver-level proofs need and the one peer identity they drive.
type pendingIndexFixture struct {
	core  *Core
	store *storage.Store
	now   time.Time
	peer  uint16
}

// newPendingIndexFixture builds a core over an empty real store at the fixed
// test instant with the default replication Config, so the pending paths run
// against the same ladder, eviction and limiter timings production uses.
func newPendingIndexFixture() *pendingIndexFixture {
	fixedNow := func() time.Time { return time.UnixMilli(listedTestNowMs) }
	store := storage.NewStore(listedTestLocal, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})
	return &pendingIndexFixture{
		core:  New(listedTestLocal, [16]byte{1}, listedStateStore{store: store}, Config{}),
		store: store,
		now:   time.UnixMilli(listedTestNowMs),
		peer:  listedTestPeer,
	}
}

// handle delivers one decoded datagram from the fixture's peer through the
// production entry point, discards the message stream and returns the result
// stream for assertion.
func (f *pendingIndexFixture) handle(runID [16]byte, msg port.Message) []port.Result {
	var out port.Outbox
	f.core.HandleDatagram(f.now, pendingIndexDatagram(f.peer, runID, msg), &out)
	out.Drain()
	return out.DrainResults()
}

// write performs one local write in production order — the store merge the
// author does first, then the notification carrying the current peer snapshot
// — and returns the notification's results.
func (f *pendingIndexFixture) write(t *testing.T, key string, ver version.RecordVersion, data []byte) []port.Result {
	t.Helper()
	rec := pendingIndexRecord(key, ver, data)
	_, _, err := f.store.Merge(key, rec)
	if err != nil {
		t.Fatalf("store.Merge(%q): %v", key, err)
	}
	return f.notify(t, key, ver)
}

// notify performs just the notification half of a local write, for the
// regressive-notification shapes whose engine write the store would reject.
func (f *pendingIndexFixture) notify(t *testing.T, key string, ver version.RecordVersion) []port.Result {
	t.Helper()
	var out port.Outbox
	f.core.NotifyLocalWrite(f.now, key, storage.Record{Version: ver}, f.core.PeerIDs(), &out)
	out.Drain()
	return out.DrainResults()
}

// seedPending installs n tracked pairs under distinct keys with strictly
// increasing versions and the caller's write-time peer snapshot, discarding
// every emission. Versions step the timestamp (never the 12-bit counter
// domain) so the fixture keeps a strict per-key order at any size.
func (f *pendingIndexFixture) seedPending(t *testing.T, n int, peers []uint16) {
	t.Helper()
	for i := range n {
		key := fmt.Sprintf("index/seed-%06d", i)
		ver := version.PackVersion(listedTestNowMs+int64(i), 0, listedTestLocal)
		var out port.Outbox
		f.core.NotifyLocalWrite(f.now, key, storage.Record{Version: ver}, peers, &out)
		out.Drain()
		out.DrainResults()
	}
}

// pendingIndexDatagram wraps msg in a decoded datagram from peer at runID —
// the in-package equivalent of the external tests' datagramFrom.
func pendingIndexDatagram(peer uint16, runID [16]byte, msg port.Message) port.InboundDatagram {
	return port.InboundDatagram{
		Source:       fmt.Sprintf("n%d", peer),
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       peer,
		RunID:        runID,
		Msg:          msg,
	}
}

// pendingIndexRunID returns a deterministic 16-byte run ID for peer; a
// restart test passes a different one to reach the restart branch.
func pendingIndexRunID(peer uint16) [16]byte {
	var id [16]byte
	id[0] = byte(peer)
	id[15] = 0x5A
	return id
}

// pendingIndexRecord builds a wire-valid record: the checksum is derived
// through the production domain, as a correct sender's store stamps it.
func pendingIndexRecord(key string, ver version.RecordVersion, data []byte) storage.Record {
	rec := storage.Record{Version: ver, Data: data}
	rec.Checksum = storage.RecordChecksum(key, ver, rec.Data, rec.Deleted)
	return rec
}

// TestCore_pendingVisitsFreshKeyWrite is the first RED-evidence gate: a
// fresh-key local write must inspect zero pending entries at both sizes, and
// the two deltas must be equal. Before the per-key index lands this observes
// 2N (the trackedVersion scan plus the removeSupersededLocal scan), so the
// failure is the phase's proof that the defect is real.
func TestCore_pendingVisitsFreshKeyWrite(t *testing.T) {
	t.Parallel()
	for _, size := range []int{pendingIndexSmall, pendingIndexLarge} {
		t.Run(fmt.Sprintf("pending=%d", size), func(t *testing.T) {
			t.Parallel()
			f := newPendingIndexFixture()
			f.handle(pendingIndexRunID(f.peer), port.Hello{})
			f.seedPending(t, size, []uint16{listedTestPeer})
			before := f.core.PendingVisits()
			var out port.Outbox
			f.core.NotifyLocalWrite(f.now, "index/fresh-write",
				storage.Record{Version: version.PackVersion(listedTestNowMs+int64(size)+1, 0, listedTestLocal)},
				f.core.PeerIDs(), &out)
			out.Drain()
			out.DrainResults()
			if got := f.core.PendingVisits() - before; got != 0 {
				t.Fatalf("a fresh-key local write over %d pending entries inspected %d pending entries, want 0 (XPORT-02/PERF-05: the write path must be key-local at any set size)",
					size, got)
			}
		})
	}
}

// TestCore_pendingVisitsFreshKeyApply is the second RED-evidence gate: an
// inbound DataReplication for a fresh key must inspect zero pending entries at
// both sizes. Before the index lands this observes N (the whole-set
// displacement loop).
func TestCore_pendingVisitsFreshKeyApply(t *testing.T) {
	t.Parallel()
	for _, size := range []int{pendingIndexSmall, pendingIndexLarge} {
		t.Run(fmt.Sprintf("pending=%d", size), func(t *testing.T) {
			t.Parallel()
			f := newPendingIndexFixture()
			f.handle(pendingIndexRunID(f.peer), port.Hello{})
			f.seedPending(t, size, []uint16{listedTestPeer})
			before := f.core.PendingVisits()
			key := "index/fresh-apply"
			rec := pendingIndexRecord(key, version.PackVersion(listedTestNowMs, 0, listedTestPeer), []byte("v"))
			f.handle(pendingIndexRunID(f.peer), port.DataReplication{Key: key, Record: rec})
			if got := f.core.PendingVisits() - before; got != 0 {
				t.Fatalf("an inbound replication for a fresh key over %d pending entries inspected %d pending entries, want 0 "+
					"(XPORT-02/PERF-05: the apply path must be key-local at any set size)", size, got)
			}
		})
	}
}

// TestCore_pendingVisitsIdleAdvance is a TRIPWIRE, never RED evidence: the
// idle path is already O(1) by construction — scanRetransmit pops only due
// heap items and popDueRetries starts at the heap root — so this test passes
// before the index lands. It pins the claim against a future whole-set scan
// being reintroduced on the timer path, and the comment must keep saying so.
func TestCore_pendingVisitsIdleAdvance(t *testing.T) {
	t.Parallel()
	for _, size := range []int{pendingIndexSmall, pendingIndexLarge} {
		t.Run(fmt.Sprintf("pending=%d", size), func(t *testing.T) {
			t.Parallel()
			f := newPendingIndexFixture()
			f.handle(pendingIndexRunID(f.peer), port.Hello{})
			f.seedPending(t, size, []uint16{listedTestPeer})
			before := f.core.PendingVisits()
			var out port.Outbox
			f.core.Advance(f.now, &out)
			out.Drain()
			out.DrainResults()
			if got := f.core.PendingVisits() - before; got != 0 {
				t.Fatalf("one idle Advance over %d pending entries inspected %d pending entries, want 0 (XPORT-02: the timer path must never walk the pending set)",
					size, got)
			}
		})
	}
}

// TestCore_pendingVisitsStructure pins the counter's increments: the hot paths
// report an honest count and the two deliberately O(n) loops report their
// exact size, so a future uninstrumented or over-instrumented loop fails by
// name instead of silently changing what the flatness gates measure.
func TestCore_pendingVisitsStructure(t *testing.T) {
	t.Parallel()
	t.Run("a repeated-key local write inspects two entries", func(t *testing.T) {
		t.Parallel()
		f := newPendingIndexFixture()
		key := "index/repeat"
		f.write(t, key, version.PackVersion(listedTestNowMs, 0, listedTestLocal), []byte("v1"))
		before := f.core.PendingVisits()
		f.write(t, key, version.PackVersion(listedTestNowMs+1, 0, listedTestLocal), []byte("v2"))
		if got := f.core.PendingVisits() - before; got != 2 {
			t.Fatalf("a repeated-key local write inspected %d pending entries, want 2 (the tracked read plus the supersession read)", got)
		}
	})
	t.Run("a due retransmission inspects one entry per due item", func(t *testing.T) {
		t.Parallel()
		f := newPendingIndexFixture()
		f.handle(pendingIndexRunID(f.peer), port.Hello{})
		for i := range 3 {
			key := fmt.Sprintf("index/due-%d", i)
			f.write(t, key, version.PackVersion(listedTestNowMs+int64(i), 0, listedTestLocal), []byte("v"))
		}
		before := f.core.PendingVisits()
		var out port.Outbox
		f.core.Advance(f.now.Add(defaultAckTimeout), &out)
		out.Drain()
		out.DrainResults()
		if got := f.core.PendingVisits() - before; got != 3 {
			t.Fatalf("one due retransmission scan with 3 due entries inspected %d pending entries, want 3", got)
		}
	})
	t.Run("a discovery flush inspects one entry per pending pair", func(t *testing.T) {
		t.Parallel()
		f := newPendingIndexFixture()
		f.seedPending(t, 4, nil)
		before := f.core.PendingVisits()
		f.handle(pendingIndexRunID(f.peer), port.Hello{})
		if got := f.core.PendingVisits() - before; got != 4 {
			t.Fatalf("a discovery flush over 4 pending pairs inspected %d pending entries, want 4 (the flush stays deliberately O(n) and prices itself here)", got)
		}
	})
}

// assertPendingIndexInvariant checks every structural claim the index makes:
// the index and the pending map have the same size, every index value is a
// live pending pair carrying the same version, every pending pair has its key
// indexed, every ledger set is exactly the pending pairs whose wait-set names
// that peer (in both directions, so neither a stale ledger entry nor a lost
// one survives), and the unacked gauge equals the index size.
func assertPendingIndexInvariant(t *testing.T, core *Core, now time.Time) {
	t.Helper()
	if len(core.pending) != len(core.pendingByKey) {
		t.Fatalf("len(pending) = %d, len(pendingByKey) = %d — the index and the pending set diverged (T-08-04)",
			len(core.pending), len(core.pendingByKey))
	}
	for key, pk := range core.pendingByKey {
		entry, ok := core.pending[pk]
		if !ok {
			t.Fatalf("pendingByKey[%q] = %v, but that pair is absent from pending", key, pk)
		}
		if pk.key != key {
			t.Fatalf("pendingByKey[%q] holds the pair for key %q", key, pk.key)
		}
		if entry.version != pk.version {
			t.Fatalf("pendingByKey[%q] version %x != the pending entry's version %x", key, pk.version, entry.version)
		}
	}
	for pk := range core.pending {
		if indexed, ok := core.pendingByKey[pk.key]; !ok || indexed != pk {
			t.Fatalf("pending pair %v has no matching index slot (got %v, present %v)", pk, indexed, ok)
		}
	}
	want := make(map[uint16]map[pendingKey]struct{}, len(core.peerUnacked))
	for pk, entry := range core.pending {
		for id := range entry.waitSet {
			if want[id] == nil {
				want[id] = make(map[pendingKey]struct{})
			}
			want[id][pk] = struct{}{}
		}
	}
	for id, got := range core.peerUnacked {
		exp := want[id]
		if len(got) != len(exp) {
			t.Fatalf("ledger[%d] holds %d pairs, want %d — exactly the pending pairs whose wait-set names that peer",
				id, len(got), len(exp))
		}
		for pk := range got {
			if _, ok := exp[pk]; !ok {
				t.Fatalf("ledger[%d] holds %v, which is not a pending pair waiting on that peer", id, pk)
			}
		}
	}
	for id, exp := range want {
		if len(core.peerUnacked[id]) != len(exp) {
			t.Fatalf("ledger[%d] holds %d pairs, want %d — the ledger lost a waiting pair",
				id, len(core.peerUnacked[id]), len(exp))
		}
	}
	if got := core.UnackedStats(now).Count; got != len(core.pendingByKey) {
		t.Fatalf("UnackedStats(%v).Count = %d, want %d (the index size)", now, got, len(core.pendingByKey))
	}
}

// naiveTrackedVersion is the pre-index definition — a scan of the pending set
// — kept in the test so the index can be compared against the fact it claims
// to encode. It also reports how many pairs the scan found, so a hidden
// second-pair path fails by name instead of silently corrupting behavior.
func naiveTrackedVersion(core *Core, key string) (version.RecordVersion, bool, int) {
	var found version.RecordVersion
	count := 0
	for pk := range core.pending {
		if pk.key != key {
			continue
		}
		count++
		if count == 1 || pk.version.Compare(found) > 0 {
			found = pk.version
		}
	}
	return found, count > 0, count
}

// assertTrackedVersionMatchesScan compares the index lookup against the naive
// scan and refuses the state the one-pair-per-key invariant forbids.
func assertTrackedVersionMatchesScan(t *testing.T, core *Core, key string) {
	t.Helper()
	wantVer, wantOK, count := naiveTrackedVersion(core, key)
	if count > 1 {
		t.Fatalf("pending holds %d pairs under key %q — the one-pair-per-key invariant is broken", count, key)
	}
	gotVer, gotOK := core.trackedVersion(key)
	if gotOK != wantOK || gotVer != wantVer {
		t.Fatalf("trackedVersion(%q) = (%x, %v), naive scan = (%x, %v)", key, gotVer, gotOK, wantVer, wantOK)
	}
}

// firstPendingHotPair returns the first pending pair among hotKeys in slice
// order — deliberately a naive scan, so index selection never hides an index
// defect from the differential.
func firstPendingHotPair(core *Core, hotKeys []string) (pendingKey, bool) {
	for _, key := range hotKeys {
		for pk := range core.pending {
			if pk.key == key {
				return pk, true
			}
		}
	}
	return pendingKey{}, false
}

// TestCore_pendingIndexInvariant drives every structure-changing path once —
// insert, local supersession, Ack drain, remote displacement, eviction and
// re-learn — and re-asserts the full invariant after each step, so a path that
// forgets the index or the ledger fails by name.
func TestCore_pendingIndexInvariant(t *testing.T) {
	t.Parallel()
	f := newPendingIndexFixture()
	assertPendingIndexInvariant(t, f.core, f.now)

	f.handle(pendingIndexRunID(f.peer), port.Hello{})
	f.seedPending(t, 5, []uint16{listedTestPeer})
	assertPendingIndexInvariant(t, f.core, f.now)

	f.write(t, "index/seed-000000", version.PackVersion(listedTestNowMs+5, 0, listedTestLocal), []byte("v"))
	assertPendingIndexInvariant(t, f.core, f.now)

	f.handle(pendingIndexRunID(f.peer),
		port.Ack{Key: "index/seed-000001", Version: version.PackVersion(listedTestNowMs+1, 0, listedTestLocal)})
	assertPendingIndexInvariant(t, f.core, f.now)

	key := "index/seed-000002"
	rec := pendingIndexRecord(key, version.PackVersion(listedTestNowMs+100, 0, listedTestPeer), []byte("remote"))
	f.handle(pendingIndexRunID(f.peer), port.DataReplication{Key: key, Record: rec})
	assertPendingIndexInvariant(t, f.core, f.now)

	f.now = f.now.Add(f.core.cfg.PeerTimeout)
	var out port.Outbox
	f.core.Advance(f.now, &out)
	out.Drain()
	out.DrainResults()
	assertPendingIndexInvariant(t, f.core, f.now)

	f.handle(pendingIndexRunID(f.peer), port.Hello{})
	assertPendingIndexInvariant(t, f.core, f.now)
}

// pendingIndexDiffOps is the differential's fixed script length: long enough
// to exercise repeated keys, drains and displacements many times over while
// staying a sub-second test.
const pendingIndexDiffOps = 2_000

// TestCore_pendingIndexDifferential runs a fixed-seed script mixing fresh-key
// writes, newer/equal/older repeated-key notifications, Acks, incoming
// replications that displace and that lose, evictions and run-id restarts.
// After EVERY operation it re-asserts the full invariant and compares
// trackedVersion with a naive scan of the pending map, so the index is proven
// equivalent to the scan over exactly the paths that mutate pending state.
func TestCore_pendingIndexDifferential(t *testing.T) {
	t.Parallel()
	f := newPendingIndexFixture()
	f.handle(pendingIndexRunID(f.peer), port.Hello{})
	hotKeys := []string{
		"diff/k0", "diff/k1", "diff/k2", "diff/k3", "diff/k4",
		"diff/k5", "diff/k6", "diff/k7", "diff/k8", "diff/k9",
	}
	rng := rand.New(rand.NewPCG(0x0802, 0x0002))
	fresh := 0
	for op := range pendingIndexDiffOps {
		touched := ""
		switch rng.IntN(9) {
		case 0:
			// A fresh key: the notification's first write always proceeds. The
			// timestamp band (5*diffOps) sits above every remote apply below, so
			// a local write never loses to the script's earlier remote records.
			key := fmt.Sprintf("diff/fresh-%05d", fresh)
			fresh++
			f.write(t, key, version.PackVersion(listedTestNowMs+5*pendingIndexDiffOps+int64(op), 0, listedTestLocal), []byte("v"))
			touched = key
		case 1:
			// A repeated key with a newer version: local supersession.
			key := hotKeys[rng.IntN(len(hotKeys))]
			f.write(t, key, version.PackVersion(listedTestNowMs+5*pendingIndexDiffOps+int64(op), 0, listedTestLocal), []byte("v"))
			touched = key
		case 2, 3:
			// Equal and older duplicates: the guard counts, tracking stays.
			if pk, ok := firstPendingHotPair(f.core, hotKeys); ok {
				ver := pk.version
				if rng.IntN(2) == 0 {
					ver = version.PackVersion(listedTestNowMs, 0, listedTestLocal)
				}
				f.notify(t, pk.key, ver)
				touched = pk.key
			}
		case 4:
			// An Ack from the one known peer: drain when the wait-set empties.
			if pk, ok := firstPendingHotPair(f.core, hotKeys); ok {
				f.handle(pendingIndexRunID(f.peer), port.Ack{Key: pk.key, Version: pk.version})
				touched = pk.key
			}
		case 5:
			// An inbound replication above every produced version: displaces.
			key := hotKeys[rng.IntN(len(hotKeys))]
			rec := pendingIndexRecord(key, version.PackVersion(listedTestNowMs+4*pendingIndexDiffOps+int64(op), 0, listedTestPeer), []byte("remote"))
			f.handle(pendingIndexRunID(f.peer), port.DataReplication{Key: key, Record: rec})
			touched = key
		case 6:
			// An inbound replication below the held version: it loses and
			// leaves the tracked pair alone.
			key := hotKeys[rng.IntN(len(hotKeys))]
			rec := pendingIndexRecord(key, version.PackVersion(listedTestNowMs, 0, listedTestPeer), []byte("stale"))
			f.handle(pendingIndexRunID(f.peer), port.DataReplication{Key: key, Record: rec})
			touched = key
		case 7:
			// Peer-timeout eviction: every wait-set empties, entries suspend.
			f.now = f.now.Add(f.core.cfg.PeerTimeout)
			var out port.Outbox
			f.core.Advance(f.now, &out)
			out.Drain()
			out.DrainResults()
		case 8:
			// A run-id restart of the known peer: the table moves, pending stays.
			runID := pendingIndexRunID(f.peer)
			runID[13] = byte(op >> 8)
			runID[14] = byte(op)
			f.handle(runID, port.Hello{})
		}
		assertPendingIndexInvariant(t, f.core, f.now)
		if touched != "" {
			assertTrackedVersionMatchesScan(t, f.core, touched)
		}
		for _, key := range hotKeys {
			assertTrackedVersionMatchesScan(t, f.core, key)
		}
	}
}

// assertPendingResults pins one step's exact result stream, so a dropped,
// duplicated or renamed emission fails by name rather than only in aggregate.
func assertPendingResults(t *testing.T, step string, got []port.Result, want ...port.Result) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: result stream = %+v, want %+v", step, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: result %d = %+v, want %+v (full stream %+v)", step, i, got[i], want[i], got)
		}
	}
}

// TestCore_pendingIndexProductionPaths drives the whole pending lifecycle
// through the production driver seams — HandleDatagram, NotifyLocalWrite and
// Advance — and never calls an unexported helper to mutate state. It asserts
// the full index/ledger invariant after every step and pins the exact
// Superseded/Drained/WaitSetEmpty/Ack sequence, so the index cannot silently
// change an XPORT-12 emission while staying internally consistent.
func TestCore_pendingIndexProductionPaths(t *testing.T) {
	t.Parallel()
	f := newPendingIndexFixture()
	key := "prod/key"
	peer := listedTestPeer
	runID := pendingIndexRunID(f.peer)
	v0 := version.PackVersion(listedTestNowMs, 0, listedTestLocal)
	v1 := version.PackVersion(listedTestNowMs+1, 0, listedTestLocal)
	v2 := version.PackVersion(listedTestNowMs+2, 0, listedTestLocal)

	// (a) A local write whose captured snapshot names a peer the table does
	// not hold yet: the promise is the write-time capture, so the waiter
	// learns the honest false immediately (A-01) and the entry is retained.
	rec1 := pendingIndexRecord(key, v1, []byte("v1"))
	_, _, err := f.store.Merge(key, rec1)
	if err != nil {
		t.Fatalf("(a): store.Merge(%q): %v", key, err)
	}
	var out port.Outbox
	f.core.NotifyLocalWrite(f.now, key, rec1, []uint16{peer}, &out)
	out.Drain()
	assertPendingResults(t, "a: write with an absent captured peer", out.DrainResults(), port.Result{
		Kind: port.ResultWaitSetEmpty, Key: key, Version: v1,
	})
	pk1 := pendingKey{key: key, version: v1}
	if entry, ok := f.core.pending[pk1]; !ok || len(entry.waitSet) != 0 {
		t.Fatalf("after (a): pending[pk1] present=%v wait-set=%v, want a retained entry with an empty wait-set", ok, entry.waitSet)
	}
	assertPendingIndexInvariant(t, f.core, f.now)

	// (b) The Hello discovers the peer and flushes the pending set: the pair
	// is re-armed and queued, but the wait-set and the ledger stay untouched.
	before := f.core.PendingVisits()
	assertPendingResults(t, "b: discovery Hello", f.handle(runID, port.Hello{}))
	if got := f.core.PendingVisits() - before; got != 1 {
		t.Fatalf("after (b): the discovery flush inspected %d pending entries, want 1", got)
	}
	if len(f.core.recordQueue) != 1 || f.core.recordQueue[0].pk != pk1 {
		t.Fatalf("after (b): recordQueue = %+v, want exactly the pk1 record send", f.core.recordQueue)
	}
	if entry := f.core.pending[pk1]; len(entry.waitSet) != 0 {
		t.Fatalf("after (b): the flush extended the wait-set to %v — a discovered peer is broadcast to, never awaited", entry.waitSet)
	}
	if len(f.core.peerUnacked[peer]) != 0 {
		t.Fatalf("after (b): the flush extended ledger[%d] to %v", peer, f.core.peerUnacked[peer])
	}
	assertPendingIndexInvariant(t, f.core, f.now)

	// (c) A newer local write under the same key supersedes the tracked pair
	// once, naming the older TRACKED version, and installs the new promise.
	assertPendingResults(t, "c: newer local write", f.write(t, key, v2, []byte("v2")), port.Result{
		Kind: port.ResultSuperseded, Key: key, Version: v1,
	})
	pk2 := pendingKey{key: key, version: v2}
	if _, ok := f.core.pending[pk1]; ok {
		t.Fatal("after (c): the superseded pair is still tracked")
	}
	if indexed, ok := f.core.pendingByKey[key]; !ok || indexed != pk2 {
		t.Fatalf("after (c): pendingByKey[%q] = (%v, %v), want pk2", key, indexed, ok)
	}
	assertPendingIndexInvariant(t, f.core, f.now)

	// (d) An equal duplicate notification counts and changes nothing.
	regressed := f.core.NotificationsRegressed()
	assertPendingResults(t, "d: equal duplicate notification", f.notify(t, key, v2))
	if got := f.core.NotificationsRegressed() - regressed; got != 1 {
		t.Fatalf("after (d): NotificationsRegressed delta = %d, want 1", got)
	}
	if indexed := f.core.pendingByKey[key]; indexed != pk2 {
		t.Fatalf("after (d): pendingByKey[%q] = %v, want pk2 unchanged", key, indexed)
	}
	assertPendingIndexInvariant(t, f.core, f.now)

	// (e) A strictly older notification supersedes its own waiter and leaves
	// the tracked pair in place.
	regressed = f.core.NotificationsRegressed()
	assertPendingResults(t, "e: older notification", f.notify(t, key, v0), port.Result{
		Kind: port.ResultSuperseded, Key: key, Version: v0,
	})
	if got := f.core.NotificationsRegressed() - regressed; got != 1 {
		t.Fatalf("after (e): NotificationsRegressed delta = %d, want 1", got)
	}
	if indexed := f.core.pendingByKey[key]; indexed != pk2 {
		t.Fatalf("after (e): pendingByKey[%q] = %v, want pk2 unchanged", key, indexed)
	}
	assertPendingIndexInvariant(t, f.core, f.now)

	// (f) An inbound replication above the tracked version displaces it: one
	// Conflicts, one Superseded with the TRACKED version, index entry gone.
	v3 := version.PackVersion(listedTestNowMs+10, 0, peer)
	conflicts := f.core.MergeStats().Conflicts
	rec := pendingIndexRecord(key, v3, []byte("remote"))
	assertPendingResults(t, "f: displacing replication",
		f.handle(runID, port.DataReplication{Key: key, Record: rec}),
		port.Result{Kind: port.ResultSuperseded, Key: key, Version: v2})
	if got := f.core.MergeStats().Conflicts - conflicts; got != 1 {
		t.Fatalf("after (f): MergeStats.Conflicts delta = %d, want 1", got)
	}
	if _, indexed := f.core.pendingByKey[key]; indexed {
		t.Fatalf("after (f): pendingByKey[%q] still present after displacement", key)
	}
	assertPendingIndexInvariant(t, f.core, f.now)

	// (g) A write drained by an Ack from the only peer: Ack then Drained.
	ackKey := "prod/ack"
	v4 := version.PackVersion(listedTestNowMs+3, 0, listedTestLocal)
	assertPendingResults(t, "g: write", f.write(t, ackKey, v4, []byte("v4")))
	assertPendingResults(t, "g: Ack drains", f.handle(runID, port.Ack{Key: ackKey, Version: v4}),
		port.Result{Kind: port.ResultAck, Key: ackKey, Version: v4, Peer: peer},
		port.Result{Kind: port.ResultDrained, Key: ackKey, Version: v4})
	assertPendingIndexInvariant(t, f.core, f.now)

	// (h) The only peer expires: WaitSetEmpty, the entry suspends, the index
	// keeps it (eviction never drains and never touches acked).
	suspendKey := "prod/suspend"
	v5 := version.PackVersion(listedTestNowMs+4, 0, listedTestLocal)
	assertPendingResults(t, "h: write", f.write(t, suspendKey, v5, []byte("v5")))
	pk5 := pendingKey{key: suspendKey, version: v5}
	f.now = f.now.Add(f.core.cfg.PeerTimeout)
	f.core.Advance(f.now, &out)
	out.Drain()
	assertPendingResults(t, "h: peer-timeout eviction", out.DrainResults(), port.Result{
		Kind: port.ResultWaitSetEmpty, Key: suspendKey, Version: v5,
	})
	if entry, ok := f.core.pending[pk5]; !ok || len(entry.waitSet) != 0 {
		t.Fatalf("after (h): the suspended entry is present=%v with wait-set %v, want retained with an empty wait-set", ok, entry.waitSet)
	}
	if len(f.core.peerUnacked[peer]) != 0 {
		t.Fatalf("after (h): ledger[%d] = %v, want it emptied by eviction", peer, f.core.peerUnacked[peer])
	}
	assertPendingIndexInvariant(t, f.core, f.now)

	// (i) The peer returns and Acks the suspended pair: the re-learn flush
	// re-arms but never re-seeds the promise, then the Ack drains it.
	assertPendingResults(t, "i: re-learn Hello", f.handle(runID, port.Hello{}))
	if len(f.core.peerUnacked[peer]) != 0 {
		t.Fatalf("after (i): the re-learn flush extended ledger[%d] to %v", peer, f.core.peerUnacked[peer])
	}
	assertPendingResults(t, "i: Ack of the suspended pair", f.handle(runID, port.Ack{Key: suspendKey, Version: v5}),
		port.Result{Kind: port.ResultAck, Key: suspendKey, Version: v5, Peer: peer},
		port.Result{Kind: port.ResultDrained, Key: suspendKey, Version: v5})
	assertPendingIndexInvariant(t, f.core, f.now)

	// (j) The stored version moves under a tracked pair; the next retry finds
	// the stale record and removes the pair through the third removePending
	// caller (emitRecordSend), emitting nothing.
	staleKey := "prod/stale"
	v6 := version.PackVersion(listedTestNowMs+5, 0, listedTestLocal)
	assertPendingResults(t, "j: write", f.write(t, staleKey, v6, []byte("v6")))
	v7 := version.PackVersion(listedTestNowMs+6, 0, peer)
	_, _, err = f.store.Merge(staleKey, pendingIndexRecord(staleKey, v7, []byte("changed")))
	if err != nil {
		t.Fatalf("j: store.Merge(%q): %v", staleKey, err)
	}
	f.now = f.now.Add(defaultAckTimeout)
	f.core.Advance(f.now, &out)
	out.Drain()
	assertPendingResults(t, "j: stale-record retry", out.DrainResults())
	if _, ok := f.core.pendingByKey[staleKey]; ok {
		t.Fatalf("after (j): pendingByKey[%q] still present after the stale-record removal", staleKey)
	}
	assertPendingIndexInvariant(t, f.core, f.now)
}
