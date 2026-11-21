package udprepl_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/udprepl"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// Shared test constants: the fixed wall-clock millisecond (the D-02 now
// seam, the store-test precedent), distinct node IDs, run IDs and opaque
// handles for the two wired nodes.
const (
	testNowMs      = int64(1730000000000)
	nodeIDA        = uint16(1)
	nodeIDB        = uint16(2)
	nodeIDStranger = uint16(99)
	handleA        = "n1"
	handleB        = "n2"
)

// The two nodes' per-process run IDs — distinct 16-byte values, because
// run_id equality is the core's own-echo test.
var (
	runIDA = [16]byte{0xA1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	runIDB = [16]byte{0xB2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
)

// stateStore adapts *storage.Store to repl.StateStore — the Phase 4
// consumer seam — by delegating the core's four methods (the persist
// engine's ApplyRemote seam lands with plan 04-06).
type stateStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s stateStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Version returns the stored version for key, tombstones included.
func (s stateStore) Version(key string) (version.RecordVersion, bool) { return s.store.Version(key) }

// Lookup returns the stored record for key, tombstones included.
func (s stateStore) Lookup(key string) (storage.Record, bool) { return s.store.Lookup(key) }

// Versions returns the shallow key→version view, tombstones included.
func (s stateStore) Versions() map[string]version.RecordVersion { return s.store.Versions() }

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
	// The hello deadline carries a positive-only deterministic jitter of at
	// most a tenth of the interval (D-01's timer half): never before the
	// configured interval, never past the envelope.
	earliest := now.Add(time.Second) // hello_interval default, IDEA §Parameters.
	latest := now.Add(time.Second + time.Second/10)
	if next.Before(earliest) {
		t.Fatalf("node %s next hello deadline %v is before %v — the deadline must never fire early", node, next, earliest)
	}
	if !next.Before(latest) {
		t.Fatalf("node %s next hello deadline %v, want within the positive-only jitter envelope [%v, %v)", node, next, earliest, latest)
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
		go func() { got <- adapter.NotifyLocalWrite("post-run/key", rec, false) }()
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

// TestNotifyLocalWrite_afterDone proves the closed-loop NotifyLocalWrite
// contract after the loop's done channel is closed (D-15 P4-IN-01): with
// Run observed returned, the notification returns promptly with
// ErrRunLoopExited — never a silent nil for a write nobody will ever
// drain, never a hang, never a panic. The select deadline converts a hang
// into a failure; no wall-clock sleep is asserted.
func TestNotifyLocalWrite_afterDone(t *testing.T) {
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

	// Run has returned, so its deferred close(a.done) has run: the call
	// must come back with the closed-loop error promptly — a wedged or
	// nil-returning enqueue fails the deadline or the errors.Is check,
	// and a panic in the enqueue path crashes the test binary outright.
	got := make(chan error, 1)
	go func() {
		got <- adapter.NotifyLocalWrite("after-done/key", storage.Record{Data: []byte("post-done write")}, false)
	}()
	select {
	case err := <-got:
		if !errors.Is(err, udprepl.ErrRunLoopExited) {
			t.Fatalf("post-done NotifyLocalWrite = %v, want ErrRunLoopExited", err)
		}
	case <-time.After(postRunDeadline):
		t.Fatalf("post-done NotifyLocalWrite blocked past %v — the closed loop wedged the write path", postRunDeadline)
	}
}

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
	return newMirrorAdapterWithConfig(t, repl.Config{})
}

// newMirrorAdapterWithConfig is newMirrorAdapter with an explicit core
// configuration — the short-peer_timeout eviction test needs its knobs.
func newMirrorAdapterWithConfig(t *testing.T, cfg repl.Config) (*udprepl.Adapter, int) {
	t.Helper()
	return newMirrorAdapterWithStore(t, newTestStore(nodeIDA), cfg)
}

// newMirrorAdapterWithStore is newMirrorAdapterWithConfig over a store the
// caller keeps, so a test can write the record before notifying it —
// production's author sequence (a notification for a record the store does
// not hold is dropped by the retransmit scan).
func newMirrorAdapterWithStore(t *testing.T, st *stateStore, cfg repl.Config) (*udprepl.Adapter, int) {
	t.Helper()
	udpPort := freeUDPPort(t)
	adapter, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", udpPort),
		fmt.Sprintf("127.255.255.255:%d", udpPort),
		nodeIDA, st, cfg,
	)
	if err != nil {
		t.Fatalf("adapter New: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = adapter.Run(ctx) }()
	return adapter, udpPort
}

// writeStoredRecord performs the author's store write of mirrorKey and
// returns the stored record (the engine-stamped version is what the
// notification carries).
func writeStoredRecord(t *testing.T, st *stateStore, data []byte) storage.Record {
	t.Helper()
	_, _, err := st.store.Set(mirrorKey, data)
	if err != nil {
		t.Fatalf("store Set(%q): %v", mirrorKey, err)
	}
	rec, err := st.store.Get(mirrorKey)
	if err != nil {
		t.Fatalf("store Get(%q): %v", mirrorKey, err)
	}
	return rec
}

// replicationReport is one WaitReplicated outcome.
type replicationReport struct {
	acked bool
	count uint32
}

// startWait launches one WaitReplicated call and returns its report channel;
// the call is bounded by mirrorWait, so a stuck wait fails the test's
// awaitReport instead of hanging it.
func startWait(t *testing.T, adapter *udprepl.Adapter, ver version.RecordVersion) <-chan replicationReport {
	t.Helper()
	reports := make(chan replicationReport, 1)
	go func() {
		acked, count := adapter.WaitReplicated(t.Context(), mirrorKey, ver, 1, mirrorWait)
		reports <- replicationReport{acked: acked, count: count}
	}()
	return reports
}

// awaitReport returns the wait's report or fails the test after mirrorWait.
func awaitReport(t *testing.T, reports <-chan replicationReport) replicationReport {
	t.Helper()
	select {
	case report := <-reports:
		return report
	case <-time.After(mirrorWait):
		t.Fatal("WaitReplicated did not return within the wait deadline")
		return replicationReport{}
	}
}

// TestWaitReplicated_ackCoverage proves the tracker seam on a real event
// loop (XPORT-12): a wait-requesting local write's tracker is created
// before the notification is enqueued, a Hello's discovery flush makes the
// peer a waiter, and that peer's Ack resolves the wait as full coverage
// (true, 1/1). Everything reads through the adapter's public seam from this
// goroutine — never the single-driver core.
func TestWaitReplicated_ackCoverage(t *testing.T) {
	t.Parallel()
	st := newTestStore(nodeIDA)
	adapter, udpPort := newMirrorAdapterWithStore(t, st, repl.Config{})
	tx := peerSocket(t)

	rec := writeStoredRecord(t, st, []byte("wait me"))
	ver := rec.Version
	notifyErr := adapter.NotifyLocalWrite(mirrorKey, rec, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(%q, track=true): %v", mirrorKey, notifyErr)
	}

	// Discovery: the Hello ends the empty peer table, so the flush makes the
	// announcing peer a waiter on the tracked write.
	datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Hello: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "discovery to surface as KnownPeers == 1", func() bool {
		return adapter.KnownPeers() == 1
	})
	waitFor(t, mirrorWait, "the tracked write to surface in the unacked gauge", func() bool {
		return adapter.UnackedStats().Count == 1
	})

	reports := startWait(t, adapter, ver)

	datagram, err = udprepl.EncodeDatagramForTest(port.Ack{Key: mirrorKey, Version: ver}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Ack: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)

	report := awaitReport(t, reports)
	if !report.acked || report.count != 1 {
		t.Fatalf("wait report = (%v, %d), want full coverage (true, 1)", report.acked, report.count)
	}
}

// TestWaitReplicated_strangerAckCounts proves the WR-01 counting behavior
// through the tracker: an Ack from a node outside the write-time peer
// snapshot does not resolve the wait and cannot inflate the report — only
// the real waiter's Ack drains the tracked write, and the report stays
// (true, 1) with the count clamped to the write-time total.
func TestWaitReplicated_strangerAckCounts(t *testing.T) {
	t.Parallel()
	st := newTestStore(nodeIDA)
	adapter, udpPort := newMirrorAdapterWithStore(t, st, repl.Config{})
	tx := peerSocket(t)

	// Discovery first, so the tracked write seeds its wait-set with node B.
	datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Hello: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "discovery to surface as KnownPeers == 1", func() bool {
		return adapter.KnownPeers() == 1
	})

	rec := writeStoredRecord(t, st, []byte("stranger me"))
	ver := rec.Version
	notifyErr := adapter.NotifyLocalWrite(mirrorKey, rec, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(%q, track=true): %v", mirrorKey, notifyErr)
	}
	waitFor(t, mirrorWait, "the tracked write to surface in the unacked gauge", func() bool {
		return adapter.UnackedStats().Count == 1
	})
	reports := startWait(t, adapter, ver)

	// A stranger's Ack joins the core's acked set (WR-01) but is not a
	// waiter, so it resolves nothing.
	datagram, err = udprepl.EncodeDatagramForTest(port.Ack{Key: mirrorKey, Version: ver}, nodeIDStranger, runIDB)
	if err != nil {
		t.Fatalf("encode stranger Ack: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)

	// The real waiter's Ack drains the write.
	datagram, err = udprepl.EncodeDatagramForTest(port.Ack{Key: mirrorKey, Version: ver}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Ack: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)

	report := awaitReport(t, reports)
	if !report.acked || report.count != 1 {
		t.Fatalf("wait report = (%v, %d), want (true, 1) — the stranger Ack must neither resolve nor inflate", report.acked, report.count)
	}
}

// TestWaitReplicated_evictionNotAck proves the honest-false rule on the
// tracker: a tracked write whose only waiter is evicted by peer_timeout
// resolves as (false, 0) — an eviction is not an Ack (T-05-04), and the
// wait must never report coverage no peer granted.
func TestWaitReplicated_evictionNotAck(t *testing.T) {
	t.Parallel()
	st := newTestStore(nodeIDA)
	adapter, udpPort := newMirrorAdapterWithStore(t, st, repl.Config{PeerTimeout: 300 * time.Millisecond})
	tx := peerSocket(t)

	datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Hello: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "discovery to surface as KnownPeers == 1", func() bool {
		return adapter.KnownPeers() == 1
	})

	rec := writeStoredRecord(t, st, []byte("evict me"))
	ver := rec.Version
	notifyErr := adapter.NotifyLocalWrite(mirrorKey, rec, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(%q, track=true): %v", mirrorKey, notifyErr)
	}
	waitFor(t, mirrorWait, "the tracked write to surface in the unacked gauge", func() bool {
		return adapter.UnackedStats().Count == 1
	})
	reports := startWait(t, adapter, ver)

	report := awaitReport(t, reports)
	if report.acked || report.count != 0 {
		t.Fatalf("wait report = (%v, %d), want the honest false (false, 0)", report.acked, report.count)
	}
}

// TestWaitReplicated_superseded proves the other honest-false rule: a
// strictly greater remote version displaces the tracked write, whose Acks
// did not all arrive — the wait resolves (false, 0) and never reports the
// superseding version's coverage.
func TestWaitReplicated_superseded(t *testing.T) {
	t.Parallel()
	st := newTestStore(nodeIDA)
	adapter, udpPort := newMirrorAdapterWithStore(t, st, repl.Config{})
	tx := peerSocket(t)

	rec := writeStoredRecord(t, st, []byte("supersede me"))
	ver := rec.Version
	notifyErr := adapter.NotifyLocalWrite(mirrorKey, rec, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(%q, track=true): %v", mirrorKey, notifyErr)
	}
	waitFor(t, mirrorWait, "the tracked write to surface in the unacked gauge", func() bool {
		return adapter.UnackedStats().Count == 1
	})
	reports := startWait(t, adapter, ver)

	newer := storage.Record{Version: version.PackVersion(testNowMs+1000, 0, nodeIDB), Data: []byte("newer")}
	// The wire record carries the production checksum: the core's validation
	// pipeline rejects a mismatched one as bad_checksum before the apply, and
	// this test must observe the supersession the record is meant to trigger.
	newer.Checksum = storage.RecordChecksum(mirrorKey, newer.Version, newer.Data, newer.Deleted)
	datagram, err := udprepl.EncodeDatagramForTest(port.DataReplication{Key: mirrorKey, Record: newer}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode DataReplication: %v", err)
	}
	sendDatagram(t, tx, udpPort, datagram)

	report := awaitReport(t, reports)
	if report.acked || report.count != 0 {
		t.Fatalf("wait report = (%v, %d), want the honest false (false, 0)", report.acked, report.count)
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
	// the loop (the unacked gauge observes its entry) first.
	err = adapter.NotifyLocalWrite(mirrorKey, storage.Record{Version: version.PackVersion(testNowMs, 0, nodeIDA), Data: []byte("v")}, false)
	if err != nil {
		t.Fatalf("NotifyLocalWrite(%q): %v", mirrorKey, err)
	}
	waitFor(t, mirrorWait, "the notified entry to surface in the unacked gauge", func() bool {
		return adapter.UnackedStats().Count == 1
	})
	if !adapter.SyncArrived() {
		t.Fatal("SyncArrived() = false after further events, want it sticky true — the flag never resets")
	}
}

// corruptByte is the bit pattern XORed into a body byte to build a bad-CRC
// datagram: the envelope still parses, the CRC fails (D-08's framing class).
const corruptByte = 0xFF

// discardCapture is a slog.Handler recording every record it is handed —
// the capturing sink TestDiscardLog_classes installs as the default logger
// for the test's duration.
type discardCapture struct {
	mu      sync.Mutex
	records []slog.Record
}

// Enabled reports every level recordable.
func (*discardCapture) Enabled(context.Context, slog.Level) bool { return true }

// Handle stores a clone of r under the lock.
func (c *discardCapture) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r.Clone())
	return nil
}

// WithAttrs returns the handler itself — the default logger adds no attrs.
func (c *discardCapture) WithAttrs([]slog.Attr) slog.Handler { return c }

// WithGroup returns the handler itself.
func (c *discardCapture) WithGroup(string) slog.Handler { return c }

// fromSource counts the captured Debug-level records carrying exactly the
// class and detail attrs — the discard-event shape logDiscard emits. The
// detail filter (this test's third-party sender handle) isolates the test's
// own datagrams from any other parallel adapter's events riding the shared
// default logger; counting only Debug records makes the level itself part
// of the match (T-6-06: an Info-level discard line would never count).
func (c *discardCapture) fromSource(class, source string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, r := range c.records {
		if r.Level != slog.LevelDebug {
			continue
		}
		var gotClass, gotDetail string
		r.Attrs(func(a slog.Attr) bool {
			switch a.Key {
			case "class":
				gotClass = a.Value.String()
			case "detail":
				gotDetail = a.Value.String()
			}
			return true
		})
		if gotClass == class && gotDetail == source {
			n++
		}
	}
	return n
}

// TestDiscardLog_classes proves D-08's adapter half on a real socket: the
// adapter's default OnDiscard bridge turns each discard into one slog
// record at the Debug level whose class attr distinguishes bad_crc from
// bad_proto_version with the sender's handle as the detail, and silence
// reigns when nothing was sent. The bad-CRC datagram is an encoded Hello
// with one body byte corrupted (the envelope parses, the CRC fails); the
// proto-version datagram is a hand-built envelope whose CRC is correct —
// the core's version check only runs past a CRC-passing envelope.
func TestDiscardLog_classes(t *testing.T) {
	t.Parallel()
	_, udpPort := newMirrorAdapter(t)
	tx := peerSocket(t)
	source := tx.LocalAddr().String()

	capture := &discardCapture{}
	previous := slog.Default()
	slog.SetDefault(slog.New(capture))
	t.Cleanup(func() { slog.SetDefault(previous) })

	if n := capture.fromSource(repl.DiscardBadCRC, source); n != 0 {
		t.Fatalf("captured %d discard records from %s before anything was sent, want none", n, source)
	}

	datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode Hello: %v", err)
	}
	datagram[len(datagram)-1] ^= corruptByte
	sendDatagram(t, tx, udpPort, datagram)
	waitFor(t, mirrorWait, "the bad_crc discard record", func() bool {
		return capture.fromSource(repl.DiscardBadCRC, source) > 0
	})

	body, err := proto.Marshal(&replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}})
	if err != nil {
		t.Fatalf("marshal proto-version probe body: %v", err)
	}
	mismatch, err := proto.Marshal(&replproto.Envelope{
		ProtoVersion: repl.SupportedProtoVersion + 1,
		NodeId:       uint32(nodeIDB),
		RunId:        runIDB[:],
		Crc32:        crc32.ChecksumIEEE(body),
		Body:         body,
	})
	if err != nil {
		t.Fatalf("marshal proto-version probe envelope: %v", err)
	}
	sendDatagram(t, tx, udpPort, mismatch)
	waitFor(t, mirrorWait, "the bad_proto_version discard record", func() bool {
		return capture.fromSource(repl.DiscardBadProtoVersion, source) > 0
	})
}

// probeLoopbackBroadcast pre-flights the one platform capability the
// asymmetric-topology tests need — a datagram sent to the loopback broadcast
// address reaches the socket bound to the destination port (loopback
// broadcast fan-out is a Linux-only kernel behavior; macOS lo0 carries no
// BROADCAST flag — the wait_integration_test.go probe, D-12) — and skips
// loudly naming what was probed when it is missing.
func probeLoopbackBroadcast(t *testing.T, probePort int) {
	t.Helper()
	rx, err := smokeSocket(fmt.Sprintf("0.0.0.0:%d", probePort))
	if err != nil {
		t.Fatalf("probe rx bind: %v", err)
	}
	defer func() { _ = rx.Close() }()
	tx, err := smokeSocket("0.0.0.0:0")
	if err != nil {
		t.Fatalf("probe tx bind: %v", err)
	}
	defer func() { _ = tx.Close() }()
	dst := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: probePort}
	_, err = tx.WriteToUDP([]byte("sup-kv mirror probe"), dst)
	if err != nil {
		t.Fatalf("probe send to %s: %v", dst, err)
	}
	err = rx.SetReadDeadline(time.Now().Add(probeWait))
	if err != nil {
		t.Fatalf("probe read deadline: %v", err)
	}
	buf := make([]byte, 64)
	for {
		n, _, readErr := rx.ReadFromUDP(buf)
		if readErr != nil {
			t.Skipf(
				"loopback broadcast to %s did not reach the bound socket within %v — platform lacks the capability (the mirror fixtures stay proven by TestMirror_pendingAckVisible and the core suites)",
				dst,
				probeWait,
			)
		}
		if n > 0 {
			return
		}
	}
}

// newAsymAdapter returns the node's adapter running its real event loop on
// one port of an asymmetric pair (06-RESEARCH Pattern 5): the node binds
// bindPort and broadcasts to crossPort — the only same-host topology whose
// cross-unicast replies (an Ack above all) reach the peer deterministically.
func newAsymAdapter(t *testing.T, nodeID uint16, bindPort, crossPort int) (*udprepl.Adapter, *stateStore) {
	t.Helper()
	st := newTestStore(nodeID)
	adapter, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", bindPort),
		fmt.Sprintf("127.255.255.255:%d", crossPort),
		nodeID, st, repl.Config{},
	)
	if err != nil {
		t.Fatalf("adapter New (node %d, bind %d, cross %d): %v", nodeID, bindPort, crossPort, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = adapter.Run(ctx) }()
	return adapter, st
}

// TestMirror_unackedSent proves the OBS-01 mirror set over a real write's
// whole lifecycle on real sockets: node A notifies a local write while
// alone — the entry parks in suspension with no known peer, so the unacked
// gauge observably holds 1 — then node B joins the asymmetric pair; its
// startup Hello is A's discovery, the flush broadcasts the pending entry, B
// applies it, Acks, and the gauge drains to 0 with the peer known. The
// sent-by-type counters corroborate the cycle (A's Hello and Data, B's
// Ack) and node B's store serves the replicated record byte-equal.
func TestMirror_unackedSent(t *testing.T) {
	t.Parallel()
	portA, portB := freeUDPPort(t), freeUDPPort(t)
	probeLoopbackBroadcast(t, portA)

	adapterA, stA := newAsymAdapter(t, nodeIDA, portA, portB)
	ver, _, err := stA.store.Set(mirrorKey, []byte("unacked me"))
	if err != nil {
		t.Fatalf("store A Set(%q): %v", mirrorKey, err)
	}
	rec, err := stA.store.Get(mirrorKey)
	if err != nil {
		t.Fatalf("store A Get(%q): %v", mirrorKey, err)
	}
	err = adapterA.NotifyLocalWrite(mirrorKey, rec, false)
	if err != nil {
		t.Fatalf("NotifyLocalWrite(%q): %v", mirrorKey, err)
	}
	waitFor(t, mirrorWait, "the unacked gauge to reach 1", func() bool {
		return adapterA.UnackedStats().Count == 1
	})

	adapterB, stB := newAsymAdapter(t, nodeIDB, portB, portA)
	waitFor(t, mirrorWait, "the unacked gauge to drain to 0 with the peer known", func() bool {
		return adapterA.UnackedStats().Count == 0 && adapterA.KnownPeers() == 1
	})

	sent := adapterA.SentByType()
	if sent.Hello < 1 || sent.Data < 1 {
		t.Fatalf("node A sent by type = %+v, want at least one Hello and one Data", sent)
	}
	if got := adapterB.SentByType().Ack; got < 1 {
		t.Fatalf("node B sent %d Acks, want at least 1 — the drain must be a real Ack cycle", got)
	}
	served, err := stB.store.Get(mirrorKey)
	if err != nil || served.Version != ver || !bytes.Equal(served.Data, []byte("unacked me")) {
		t.Fatalf("node B serves (%x, %q, %v) for %q, want the replicated record (%x, %q)",
			served.Version[:], served.Data, err, mirrorKey, ver[:], "unacked me")
	}
}
