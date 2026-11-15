package udprepl_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// Shared test constants: the fixed wall-clock millisecond (the D-02 now
// seam, the store-test precedent), distinct node IDs, run IDs and opaque
// handles for the two wired nodes.
const (
	testNowMs = int64(1730000000000)
	nodeIDA   = uint16(1)
	nodeIDB   = uint16(2)
	handleA   = "n1"
	handleB   = "n2"
)

// The two nodes' per-process run IDs — distinct 16-byte values, because
// run_id equality is the core's own-echo test.
var (
	runIDA = [16]byte{0xA1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	runIDB = [16]byte{0xB2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
)

// stateStore adapts *storage.Store to repl.StateStore — the Phase 4
// consumer seam — by delegating both methods (the persist engine's
// ApplyRemote seam lands with plan 04-06).
type stateStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s stateStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Records returns the full snapshot, tombstones included.
func (s stateStore) Records() map[string]storage.Record { return s.store.Records() }

// newTestStore returns a store for nodeID parked at the fixed test
// wall-clock millisecond (built with time.UnixMilli so no monotonic
// reading is involved — the D-02 seam).
func newTestStore(nodeID uint16) *stateStore {
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	return &stateStore{store: storage.NewStore(nodeID, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})}
}

// TestWire_twoNodeHandoff proves the tracer path end-to-end (REPL-03): a
// local write on node A is broadcast as a DataReplication datagram through
// the adapter's real protobuf encode, decoded back through the port seam,
// applied to node B's store, and Acked unicast to A's handle — with no
// goroutine inside the core, the whole path driven handler by handler. A
// duplicate delivery of the same datagram is an idempotent no-op merge
// that still Acks (REPL-09's adjacency edge), and the first Advance on
// each core emits exactly one broadcast startup Hello followed by the
// startup DataSync listing (IDEA §Node lifecycle, plan 04-03's startup
// trigger).
func TestWire_twoNodeHandoff(t *testing.T) {
	t.Parallel()

	now := time.UnixMilli(testNowMs)
	storeA, storeB := newTestStore(nodeIDA), newTestStore(nodeIDB)
	coreA := repl.New(nodeIDA, runIDA, handleA, storeA, repl.Config{})
	coreB := repl.New(nodeIDB, runIDB, handleB, storeB, repl.Config{})

	// Startup: the first Advance on each core emits exactly one broadcast
	// Hello and schedules the next one a hello_interval later.
	var outA, outB port.Outbox
	assertStartupHello(t, "A", coreA.Advance(now, &outA), outA.Drain(), now)
	assertStartupHello(t, "B", coreB.Advance(now, &outB), outB.Drain(), now)

	// A local write on A: the author stores, then calls NotifyLocalWrite
	// (D-05).
	const key = "netmgr/v2/hosts/alpha"
	data := []byte("10.0.0.7 peanut")
	ver, _, err := storeA.store.Set(key, data)
	if err != nil {
		t.Fatalf("store A Set(%q): %v", key, err)
	}
	written, err := storeA.store.Get(key)
	if err != nil {
		t.Fatalf("store A Get(%q): %v", key, err)
	}
	coreA.NotifyLocalWrite(now, key, written, &outA)

	// The wire: encode each outbound datagram with the adapter's encode
	// path, decode it on B's side with A's handle as the source, feed B's
	// core (D-01/D-02).
	sent := outA.Drain()
	if len(sent) != 1 || !sent[0].Broadcast {
		t.Fatalf("NotifyLocalWrite drained %d outbound messages, want exactly 1 broadcast: %+v", len(sent), sent)
	}
	if _, ok := sent[0].Msg.(port.DataReplication); !ok {
		t.Fatalf("NotifyLocalWrite emitted %T, want port.DataReplication", sent[0].Msg)
	}
	inboundB := make([]port.InboundDatagram, 0, len(sent))
	for _, ob := range sent {
		datagram, err := udprepl.EncodeDatagramForTest(ob.Msg, nodeIDA, runIDA)
		if err != nil {
			t.Fatalf("encode %T: %v", ob.Msg, err)
		}
		in := udprepl.DecodeDatagramForTest(datagram, handleA)
		if in.Status != port.Decoded {
			t.Fatalf("decode %T: status %d, want port.Decoded", ob.Msg, in.Status)
		}
		inboundB = append(inboundB, in)
		coreB.HandleDatagram(now, in, &outB)
	}

	// B's store serves the record byte-equal under the same version.
	served, err := storeB.store.Get(key)
	if err != nil {
		t.Fatalf("store B Get(%q) after replication: %v", key, err)
	}
	if served.Version != ver || !bytes.Equal(served.Data, data) {
		t.Fatalf("store B serves version %v data %q, want version %v data %q", served.Version, served.Data, ver, data)
	}

	// B Acks exactly once — unicast to A's handle, for (key, version),
	// immediately after the in-memory apply (REPL-03).
	assertOneAck(t, "after apply", outB.Drain(), key, ver)

	// Duplicate delivery of the same datagram: the store is unchanged
	// (merge idempotence) and B still Acks.
	before := storeB.store.Records()
	for _, in := range inboundB {
		coreB.HandleDatagram(now, in, &outB)
	}
	if after := storeB.store.Records(); !recordsEqual(before, after) {
		t.Fatalf("duplicate delivery changed B's store: before %d records, after %d", len(before), len(after))
	}
	assertOneAck(t, "after duplicate delivery", outB.Drain(), key, ver)
}

// assertStartupHello fails unless next is the following hello deadline and
// drained carries exactly one broadcast Hello followed by the startup
// DataSync (plan 04-03's startup trigger: the full listing fires alongside
// the first Hello; both nodes' stores are empty here, so the listing is
// one zero-entry chunk).
func assertStartupHello(t *testing.T, node string, next time.Time, drained []port.Outbound, now time.Time) {
	t.Helper()
	if len(drained) != 2 || !drained[0].Broadcast || !drained[1].Broadcast {
		t.Fatalf("node %s startup drained %d outbound messages, want exactly 2 broadcasts: Hello then the startup DataSync", node, len(drained))
	}
	if _, ok := drained[0].Msg.(port.Hello); !ok {
		t.Fatalf("node %s startup emitted %T first, want port.Hello", node, drained[0].Msg)
	}
	if _, ok := drained[1].Msg.(port.DataSync); !ok {
		t.Fatalf("node %s startup emitted %T second, want the startup port.DataSync", node, drained[1].Msg)
	}
	if want := now.Add(time.Second); !next.Equal(want) { // hello_interval default, IDEA §Parameters.
		t.Fatalf("node %s next hello deadline %v, want %v", node, next, want)
	}
}

// assertOneAck fails unless drained carries exactly one outbound message:
// a unicast port.Ack to A's handle for (key, ver).
func assertOneAck(t *testing.T, stage string, drained []port.Outbound, key string, ver [16]byte) {
	t.Helper()
	if len(drained) != 1 || drained[0].Broadcast || drained[0].To != handleA {
		t.Fatalf("%s: drained %d outbound messages, want exactly 1 unicast to %q: %+v", stage, len(drained), handleA, drained)
	}
	ack, ok := drained[0].Msg.(port.Ack)
	if !ok {
		t.Fatalf("%s: B emitted %T, want port.Ack", stage, drained[0].Msg)
	}
	if ack.Key != key || ack.Version != ver {
		t.Fatalf("%s: B Ack = (%q, %v), want (%q, %v)", stage, ack.Key, ack.Version, key, ver)
	}
}

// recordsEqual compares two full store snapshots field by field, Data
// included.
func recordsEqual(a, b map[string]storage.Record) bool {
	if len(a) != len(b) {
		return false
	}
	for key, ra := range a {
		rb, ok := b[key]
		if !ok || ra.Version != rb.Version || ra.Deleted != rb.Deleted ||
			ra.Checksum != rb.Checksum || !bytes.Equal(ra.Data, rb.Data) {
			return false
		}
	}
	return true
}

// Post-Run NotifyLocalWrite constants: the call count must cross the
// adapter's 1024-entry notify buffer (the WR-01 window — an unguarded send
// swallows the first 1024 post-Run notifications and blocks forever on the
// 1025th), and the per-call deadline converts that block into a test
// failure instead of a hung run.
const (
	postRunNotifyCalls = 1100
	postRunDeadline    = 2 * time.Second
)

// TestNotifyLocalWrite_afterRunReturns proves the NotifyLocalWrite exit
// guard (WR-01): once Run has returned — context cancelled, the deferred
// close(a.done) done — the public write-path seam never blocks and never
// panics, however many calls arrive; every call returns promptly with
// ErrRunLoopExited so a caller (Phase 5's gRPC write handler) fails the
// write visibly instead of wedging on a queue nobody drains.
func TestNotifyLocalWrite_afterRunReturns(t *testing.T) {
	t.Parallel()
	udpPort := freeUDPPort(t)
	adapter, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", udpPort),
		fmt.Sprintf("127.255.255.255:%d", udpPort),
		nodeIDA, newTestStore(nodeIDA), repl.Config{},
	)
	if err != nil {
		t.Fatalf("adapter New: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	runReturned := make(chan error, 1)
	go func() { runReturned <- adapter.Run(ctx) }()
	cancel()
	err = <-runReturned
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned %v, want context.Canceled", err)
	}

	// Run has returned, so its deferred close(a.done) has run — every
	// NotifyLocalWrite, past the buffered cap included, must report the
	// drop instead of blocking.
	rec := storage.Record{Data: []byte("post-run write")}
	for call := range postRunNotifyCalls {
		got := make(chan error, 1)
		go func() { got <- adapter.NotifyLocalWrite("post-run/key", rec) }()
		select {
		case err := <-got:
			if !errors.Is(err, udprepl.ErrRunLoopExited) {
				t.Fatalf("post-Run NotifyLocalWrite call %d = %v, want ErrRunLoopExited", call+1, err)
			}
		case <-time.After(postRunDeadline):
			t.Fatalf("post-Run NotifyLocalWrite call %d blocked past %v — the write path wedged after Run exited", call+1, postRunDeadline)
		}
	}
}

// mirrorWait is the mirror tests' poll deadline (real time — a real event
// loop over a real socket; loopback unicast delivery is sub-millisecond, so
// two seconds is generous).
const mirrorWait = 2 * time.Second

// mirrorKey is the one key the mirror tests write and sync about.
const mirrorKey = "mirror/key"

// peerSocket returns the one loopback socket the fake peer (node B) sends
// from, held for the test's lifetime: one node owns one socket, so every
// datagram it sends carries the same source handle — a per-datagram
// throwaway socket would make the second-variant duplicate-node_id
// classifier (REPL-08: one identity claimed from two handles) discard the
// peer's later datagrams.
func peerSocket(t *testing.T) net.PacketConn {
	t.Helper()
	var lc net.ListenConfig
	tx, err := lc.ListenPacket(t.Context(), "udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind loopback peer socket: %v", err)
	}
	t.Cleanup(func() { _ = tx.Close() })
	return tx
}

// sendDatagram delivers one encoded datagram from the peer's socket to the
// adapter's wildcard socket over loopback unicast: the adapter exclusively
// holds its port (no SO_REUSEPORT sibling receiver), so kernel delivery is
// deterministic — the same-host ambiguity documented on the package affects
// only ports shared between SO_REUSEADDR receivers.
func sendDatagram(t *testing.T, tx net.PacketConn, udpPort int, datagram []byte) {
	t.Helper()
	dst := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: udpPort}
	_, err := tx.WriteTo(datagram, dst)
	if err != nil {
		t.Fatalf("send datagram to %s: %v", dst, err)
	}
}

// newMirrorAdapter returns the adapter under test running its real event
// loop on a free port, with the loopback broadcast address (the
// postRunNotify test's shape). The loop is stopped by the test's cleanup.
func newMirrorAdapter(t *testing.T) (*udprepl.Adapter, int) {
	t.Helper()
	udpPort := freeUDPPort(t)
	adapter, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", udpPort),
		fmt.Sprintf("127.255.255.255:%d", udpPort),
		nodeIDA, newTestStore(nodeIDA), repl.Config{},
	)
	if err != nil {
		t.Fatalf("adapter New: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = adapter.Run(ctx) }()
	return adapter, udpPort
}

// TestMirror_pendingAckVisible proves the pending mirror (API-05): a
// notified local write surfaces in PendingCopy once its event traverses the
// loop, an Ack datagram surfaces as the peer joining the entry's Acked
// while the entry survives on its empty wait-set (zero known peers — the
// suspension rule), and a snapshot held across a later event stays stable
// while the mirror replaces the published map wholesale. Everything reads
// through the adapter atomics from this goroutine — never the
// single-driver core.
func TestMirror_pendingAckVisible(t *testing.T) {
	t.Parallel()
	adapter, udpPort := newMirrorAdapter(t)
	tx := peerSocket(t)

	if got := adapter.PendingCopy(); len(got) != 0 {
		t.Fatalf("PendingCopy() before the write holds %d entries, want none", len(got))
	}

	// The author's notification traverses the loop: the entry appears
	// waiting on nobody (nothing known to wait on) with Acked empty.
	ver := version.PackVersion(testNowMs, 0, nodeIDA)
	rec := storage.Record{Version: ver, Data: []byte("mirror me")}
	err := adapter.NotifyLocalWrite(mirrorKey, rec)
	if err != nil {
		t.Fatalf("NotifyLocalWrite(%q): %v", mirrorKey, err)
	}
	waitFor(t, mirrorWait, "the pending entry to appear in PendingCopy", func() bool {
		info, ok := adapter.PendingCopy()[mirrorKey]
		return ok && info.Version == ver && len(info.WaitingOn) == 0 && len(info.Acked) == 0
	})

	// An Ack traverses the loop: the peer joins Acked, read back through
	// the atomic snapshot.
	datagram, err := udprepl.EncodeDatagramForTest(port.Ack{Key: mirrorKey, Version: ver}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Ack: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "the Ack to surface in PendingCopy Acked", func() bool {
		info, ok := adapter.PendingCopy()[mirrorKey]
		return ok && slices.Equal(info.Acked, []uint16{nodeIDB}) && len(info.WaitingOn) == 0
	})

	// A snapshot held across a later event stays stable: the mirror
	// replaces the published pointer wholesale and never mutates a
	// published map. The Hello ends the empty peer table (discovery), so
	// the fresh snapshot additionally shows the flushed peer as a waiter —
	// with the earlier Ack still recorded in Acked (eviction and discovery
	// never erase it).
	held := adapter.PendingCopy()
	datagram, err = udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Hello: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "discovery to surface as KnownPeers == 1", func() bool {
		return adapter.KnownPeers() == 1
	})
	if !slices.Equal(held[mirrorKey].Acked, []uint16{nodeIDB}) || len(held[mirrorKey].WaitingOn) != 0 {
		t.Fatalf("the snapshot held across the discovery event changed: %+v", held[mirrorKey])
	}
	info := adapter.PendingCopy()[mirrorKey]
	if !slices.Equal(info.Acked, []uint16{nodeIDB}) || !slices.Equal(info.WaitingOn, []uint16{nodeIDB}) {
		t.Fatalf("after discovery the mirrored entry = (WaitingOn %v, Acked %v), want the flushed waiter plus the kept Ack",
			info.WaitingOn, info.Acked)
	}
}

// TestMirror_syncArrivedFlip proves the sync mirror (API-04): after the
// mirror has demonstrably run (a Hello traversed the loop) SyncArrived() is
// still false, one DataSync datagram traversing the loop flips it true
// through the atomic, and it stays true across further unrelated events —
// the flag never resets.
func TestMirror_syncArrivedFlip(t *testing.T) {
	t.Parallel()
	adapter, udpPort := newMirrorAdapter(t)
	tx := peerSocket(t)

	// A first event (the peer's Hello) proves the mirror has run at least
	// once while the flag is still false — not merely unset.
	datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Hello: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "the Hello to surface as KnownPeers == 1", func() bool {
		return adapter.KnownPeers() == 1
	})
	if adapter.SyncArrived() {
		t.Fatal("SyncArrived() = true after a Hello event, want false — only a DataSync receipt counts")
	}

	// One DataSync datagram from the same peer handle flips the mirrored
	// flag.
	datagram, err = udprepl.EncodeDatagramForTest(port.DataSync{
		Entries: []port.SyncEntry{{Key: mirrorKey, Version: version.PackVersion(testNowMs, 1, nodeIDB)}},
	}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode DataSync: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "SyncArrived() to flip true", func() bool {
		return adapter.SyncArrived()
	})

	// Further unrelated events never reset it: a notified write traverses
	// the loop (its entry appears in the pending mirror) first.
	err = adapter.NotifyLocalWrite(mirrorKey, storage.Record{Version: version.PackVersion(testNowMs, 0, nodeIDA), Data: []byte("v")})
	if err != nil {
		t.Fatalf("NotifyLocalWrite(%q): %v", mirrorKey, err)
	}
	waitFor(t, mirrorWait, "the notified entry to appear in PendingCopy", func() bool {
		_, ok := adapter.PendingCopy()[mirrorKey]
		return ok
	})
	if !adapter.SyncArrived() {
		t.Fatal("SyncArrived() = false after further events, want it sticky true — the flag never resets")
	}
}

// TestMirror_drainedAckedDedup proves the drained-row Acked dedup (D-15
// P4-WR-01): a peer can legitimately sit in BOTH the last observed Acked
// and WaitingOn — it Acked while the peer table was empty (the handleAck
// counting behavior), then a discovery flush re-added it to the wait-set —
// so the drained row's Acked union would publish the peer twice without
// the sort-then-Compact. The scenario drives exactly that shape through
// the real event loop: write, stranger Ack, discovery Hello (the flush
// re-add), then the draining Ack — and asserts the published drained row
// holds the peer exactly once.
func TestMirror_drainedAckedDedup(t *testing.T) {
	t.Parallel()
	adapter, udpPort := newMirrorAdapter(t)
	tx := peerSocket(t)

	// The notified entry appears waiting on nobody (empty peer table).
	ver := version.PackVersion(testNowMs, 0, nodeIDA)
	rec := storage.Record{Version: ver, Data: []byte("dedup me")}
	err := adapter.NotifyLocalWrite(mirrorKey, rec)
	if err != nil {
		t.Fatalf("NotifyLocalWrite(%q): %v", mirrorKey, err)
	}
	waitFor(t, mirrorWait, "the pending entry to appear in PendingCopy", func() bool {
		info, ok := adapter.PendingCopy()[mirrorKey]
		return ok && info.Version == ver && len(info.WaitingOn) == 0 && len(info.Acked) == 0
	})

	// A stranger's Ack joins Acked while the entry survives in suspension
	// (empty table): the peer is now in Acked and nowhere else.
	datagram, err := udprepl.EncodeDatagramForTest(port.Ack{Key: mirrorKey, Version: ver}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Ack: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "the stranger Ack to surface in PendingCopy Acked", func() bool {
		info, ok := adapter.PendingCopy()[mirrorKey]
		return ok && slices.Equal(info.Acked, []uint16{nodeIDB}) && len(info.WaitingOn) == 0
	})

	// The peer's Hello ends the empty table — discovery — and the flush
	// re-adds it to the wait-set: the last observed row now holds the peer
	// in BOTH sets, the exact P4-WR-01 pre-state.
	datagram, err = udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Hello: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "the discovery flush to put the peer in both WaitingOn and Acked", func() bool {
		info, ok := adapter.PendingCopy()[mirrorKey]
		return ok && slices.Equal(info.WaitingOn, []uint16{nodeIDB}) && slices.Equal(info.Acked, []uint16{nodeIDB})
	})

	// The draining Ack empties the wait-set with the peer known: the entry
	// leaves the core, retainDrained publishes the resolved row, and its
	// Acked union must hold the peer exactly once — the naive
	// clone(Acked)+WaitingOn union would hold it twice.
	datagram, err = udprepl.EncodeDatagramForTest(port.Ack{Key: mirrorKey, Version: ver}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode draining Ack: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "the drained row to appear with each peer exactly once in Acked", func() bool {
		info, ok := adapter.PendingCopy()[mirrorKey]
		if !ok {
			return false
		}
		deduped := slices.Compact(slices.Clone(info.Acked))
		return len(info.WaitingOn) == 0 && len(info.Acked) == len(deduped) &&
			slices.Equal(info.Acked, []uint16{nodeIDB})
	})
}
