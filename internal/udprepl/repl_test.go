package udprepl_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
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
