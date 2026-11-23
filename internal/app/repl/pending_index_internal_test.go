package repl

// White-box proofs for the per-key pending index and the deterministic
// PendingVisits seam (XPORT-02, PERF-05). They live in package repl because
// pendingByKey, the pending entries and the counter are unexported state no
// external test can assert, and because the differential must compare the
// index against a naive scan from the inside. The _internal_test.go suffix is
// the documented testpackage exemption.

import (
	"fmt"
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
// driver-level proofs need.
type pendingIndexFixture struct {
	core  *Core
	store *storage.Store
	now   time.Time
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
	}
}

// handle delivers one decoded datagram through the production entry point,
// discards the message stream and returns the result stream for assertion.
func (f *pendingIndexFixture) handle(peer uint16, runID [16]byte, msg port.Message) []port.Result {
	var out port.Outbox
	f.core.HandleDatagram(f.now, pendingIndexDatagram(peer, runID, msg), &out)
	out.Drain()
	return out.DrainResults()
}

// write performs one local write in production order — the store merge the
// author does first, then the notification carrying the current peer snapshot
// — and returns the notification's results.
func (f *pendingIndexFixture) write(t *testing.T, key string, ver version.RecordVersion, data []byte) []port.Result {
	t.Helper()
	rec := pendingIndexRecord(key, ver, data)
	if _, _, err := f.store.Merge(key, rec); err != nil {
		t.Fatalf("store.Merge(%q): %v", key, err)
	}
	var out port.Outbox
	f.core.NotifyLocalWrite(f.now, key, rec, f.core.PeerIDs(), &out)
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
			f.handle(listedTestPeer, pendingIndexRunID(listedTestPeer), port.Hello{})
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
			f.handle(listedTestPeer, pendingIndexRunID(listedTestPeer), port.Hello{})
			f.seedPending(t, size, []uint16{listedTestPeer})
			before := f.core.PendingVisits()
			key := "index/fresh-apply"
			rec := pendingIndexRecord(key, version.PackVersion(listedTestNowMs, 0, listedTestPeer), []byte("v"))
			f.handle(listedTestPeer, pendingIndexRunID(listedTestPeer), port.DataReplication{Key: key, Record: rec})
			if got := f.core.PendingVisits() - before; got != 0 {
				t.Fatalf("an inbound replication for a fresh key over %d pending entries inspected %d pending entries, want 0 (XPORT-02/PERF-05: the apply path must be key-local at any set size)",
					size, got)
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
			f.handle(listedTestPeer, pendingIndexRunID(listedTestPeer), port.Hello{})
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
		f.handle(listedTestPeer, pendingIndexRunID(listedTestPeer), port.Hello{})
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
		f.handle(listedTestPeer, pendingIndexRunID(listedTestPeer), port.Hello{})
		if got := f.core.PendingVisits() - before; got != 4 {
			t.Fatalf("a discovery flush over 4 pending pairs inspected %d pending entries, want 4 (the flush stays deliberately O(n) and prices itself here)", got)
		}
	})
}
