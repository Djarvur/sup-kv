package udprepl

// White-box on purpose (the *_internal_test.go testpackage exemption, the
// readloop_internal_test.go precedent): the send-accounting contract lives in
// flush's unexported write path, and the mirror accessors are unexported
// atomics. Only a scripted fake socket in this package can fail a write on
// demand and observe the resulting counter state.

import (
	"errors"
	"net"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// errInjectedWriteFault is the scripted send failure — a package-level
// sentinel because the lint gate's err113 rule forbids constructing errors at
// use sites, and the tests only need one distinguishable, stable value.
var errInjectedWriteFault = errors.New("injected write fault")

// flushBroadcastAddr and flushUnicastHandle are the two addresses the flush
// accounting test sends to. The fake socket never touches the network, so the
// values only need to be resolvable shapes.
var (
	flushBroadcastAddr = &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 7900}
	flushUnicastHandle = "127.0.0.1:7900"
	flushBrokenHandle  = "not-a-resolvable-handle"
)

// flushPacketConn is the scripted [packetConn] behind the send-accounting
// test — the write-side twin of readloop_internal_test.go's fakePacketConn
// (the two names must differ: both live in this test binary). WriteToUDP
// honors a switchable fault; every read blocks until Close, because the flush
// path never receives.
type flushPacketConn struct {
	mu        sync.Mutex
	writeErr  error
	writes    int
	closed    chan struct{}
	closeOnce sync.Once
}

// newFlushPacketConn returns a fake that accepts writes until failWrites
// installs a fault.
func newFlushPacketConn() *flushPacketConn {
	return &flushPacketConn{closed: make(chan struct{})}
}

// failWrites makes every subsequent WriteToUDP return err; nil restores the
// accepting path.
func (c *flushPacketConn) failWrites(err error) {
	c.mu.Lock()
	c.writeErr = err
	c.mu.Unlock()
}

// writesServed returns how many WriteToUDP calls were attempted — the
// evidence that a refused send still reached the socket and was refused
// there.
func (c *flushPacketConn) writesServed() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writes
}

// WriteToUDP attempts one datagram: the configured fault wins, otherwise the
// write is accepted.
func (c *flushPacketConn) WriteToUDP(b []byte, _ *net.UDPAddr) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes++
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	return len(b), nil
}

// ReadFromUDP blocks until Close: the flush path never receives, and a
// blocking read keeps this fake honest about the socket's two directions.
func (c *flushPacketConn) ReadFromUDP(_ []byte) (int, *net.UDPAddr, error) {
	<-c.closed
	return 0, nil, net.ErrClosed
}

// SetReadBuffer is a no-op: no kernel socket exists here.
func (*flushPacketConn) SetReadBuffer(_ int) error { return nil }

// SetWriteBuffer is a no-op: no kernel socket exists here.
func (*flushPacketConn) SetWriteBuffer(_ int) error { return nil }

// SyscallConn reports the absent raw socket; the flush path never reads
// socket options.
func (*flushPacketConn) SyscallConn() (syscall.RawConn, error) { return nil, errFakeNoRawSocket }

// LocalAddr returns a scripted datagram address.
func (*flushPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 7900}
}

// Close unblocks a waiting reader; repeated calls are safe.
func (c *flushPacketConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

// newFlushTestAdapter returns the bare adapter over conn with the broadcast
// address and a capturing log installed — flush's whole seam.
func newFlushTestAdapter(conn *flushPacketConn) (*Adapter, *[]string) {
	logs := []string{}
	adapter := newAdapterWithConn(conn)
	adapter.broadcastAddr = flushBroadcastAddr
	adapter.log = func(msg string) { logs = append(logs, msg) }
	return adapter, &logs
}

// TestFlush_countsOnlySuccessfulWrites pins the honest sent accounting
// (OBSV-03, D-05): a datagram counts into its SentByType kind only after
// WriteToUDP accepted it. A failed write, a resolve failure and an encode
// failure count nothing — UDP sent is not delivery — while a later successful
// write on the same path counts exactly once.
func TestFlush_countsOnlySuccessfulWrites(t *testing.T) {
	t.Parallel()

	t.Run("a failed write is not counted as sent", func(t *testing.T) {
		t.Parallel()
		conn := newFlushPacketConn()
		adapter, _ := newFlushTestAdapter(conn)
		conn.failWrites(errInjectedWriteFault)

		var out port.Outbox
		out.Broadcast(port.Hello{})
		adapter.flush(&out)

		if got := conn.writesServed(); got != 1 {
			t.Fatalf("WriteToUDP calls = %d, want 1 — the failed write must still be attempted", got)
		}
		if got := adapter.SentByType().Hello; got != 0 {
			t.Fatalf("SentByType().Hello = %d after a failed write, want 0 — a refused datagram was never sent", got)
		}
	})

	t.Run("a write accepted after a failure counts exactly once", func(t *testing.T) {
		t.Parallel()
		conn := newFlushPacketConn()
		adapter, _ := newFlushTestAdapter(conn)
		conn.failWrites(errInjectedWriteFault)

		var failed port.Outbox
		failed.Broadcast(port.Hello{})
		adapter.flush(&failed)

		conn.failWrites(nil)
		var sent port.Outbox
		sent.Broadcast(port.Hello{})
		adapter.flush(&sent)

		if got := adapter.SentByType().Hello; got != 1 {
			t.Fatalf("SentByType().Hello = %d after one refused and one accepted write, want 1", got)
		}
	})

	t.Run("a unicast resolve failure counts nothing", func(t *testing.T) {
		t.Parallel()
		conn := newFlushPacketConn()
		adapter, _ := newFlushTestAdapter(conn)

		var out port.Outbox
		out.Unicast(flushBrokenHandle, port.Ack{Key: "k"})
		adapter.flush(&out)

		if got := conn.writesServed(); got != 0 {
			t.Fatalf("WriteToUDP calls = %d for an unresolvable handle, want 0", got)
		}
		if got := adapter.SentByType(); got != (SentByType{}) {
			t.Fatalf("SentByType() = %+v after a resolve failure, want the zero value", got)
		}
	})

	t.Run("an encode failure counts nothing and the flush continues", func(t *testing.T) {
		t.Parallel()
		conn := newFlushPacketConn()
		adapter, logs := newFlushTestAdapter(conn)

		var out port.Outbox
		out.Broadcast(nil) // encodeDatagram rejects a nil message.
		out.Broadcast(port.Hello{})
		adapter.flush(&out)

		if got := len(*logs); got != 1 {
			t.Fatalf("logged %d lines for one encode failure, want 1: %q", got, *logs)
		}
		if !strings.Contains((*logs)[0], "standalone") {
			t.Fatalf("the logged encode failure = %q, want the no-standalone-body error", (*logs)[0])
		}
		if got := conn.writesServed(); got != 1 {
			t.Fatalf("WriteToUDP calls = %d, want 1 — the flush must continue past the encode failure", got)
		}
		if got := adapter.SentByType().Hello; got != 1 {
			t.Fatalf("SentByType().Hello = %d after an encode failure followed by a Hello, want 1", got)
		}
	})

	t.Run("a successful unicast Ack counts exactly once", func(t *testing.T) {
		t.Parallel()
		conn := newFlushPacketConn()
		adapter, _ := newFlushTestAdapter(conn)

		var out port.Outbox
		out.Unicast(flushUnicastHandle, port.Ack{Key: "k"})
		adapter.flush(&out)

		got := adapter.SentByType()
		if got.Ack != 1 {
			t.Fatalf("SentByType().Ack = %d after one accepted unicast Ack, want 1", got.Ack)
		}
		if got != (SentByType{Ack: 1}) {
			t.Fatalf("SentByType() = %+v, want only Ack set to 1", got)
		}
	})
}

// TestMirror_listingCounters proves the four routed Phase-4 counters reach
// the adapter mirror with the core's values (OBSV-03, D-05/14.4). EVERY
// counter is driven to a non-zero sample before the mirror pass: the
// regression counter by an equal duplicate of one tracked notification, the
// listed-key eviction counter by a retained listing whose two-sync-interval
// lifetime expires, the listing-drop counter by overrunning the
// targeted-listing destination cap, and the sync-stream counter by
// overrunning the partial-round accumulator. The equality assertions
// therefore compare real values — a missing or swapped mirror store cannot
// pass.
func TestMirror_listingCounters(t *testing.T) {
	t.Parallel()
	const (
		localNode = uint16(21)
		peerNode  = uint16(22)
		key       = "mirror/listed"
		syncEvery = time.Second
		// The floods deliberately differ in size (and so in resulting
		// count): 40 triggers against 32 destination slots cannot yield
		// fewer than 8 refusals, while 20 partial rounds against 16 stream
		// slots yield 4 evictions — so a mirror store that swapped the two
		// accessors compares against the wrong non-zero value.
		listingFlood = 40
		streamFlood  = 20
	)
	fixedNow := func() time.Time { return time.UnixMilli(1730000000000) }
	now := fixedNow()
	st := storage.NewStore(localNode, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})
	core := repl.New(localNode, [16]byte{1}, trackerStore{store: st}, repl.Config{SyncInterval: syncEvery})
	adapter := &Adapter{core: core, log: func(string) {}}
	var out port.Outbox

	// A Hello binds the peer identity, then a complete one-part listing
	// retains one key — the state the expiry releases.
	core.HandleDatagram(now, port.InboundDatagram{
		Source:       "n22",
		Status:       port.Decoded,
		ProtoVersion: repl.SupportedProtoVersion,
		NodeID:       peerNode,
		RunID:        [16]byte{byte(peerNode)},
		Msg:          port.Hello{},
	}, &out)
	core.HandleDatagram(now, port.InboundDatagram{
		Source:       "n22",
		Status:       port.Decoded,
		ProtoVersion: repl.SupportedProtoVersion,
		NodeID:       peerNode,
		RunID:        [16]byte{byte(peerNode)},
		Msg: port.DataSync{SyncID: 1, ChunkIndex: 1, ChunkCount: 1, Entries: []port.SyncEntry{{
			Key:     key,
			Version: version.PackVersion(now.UnixMilli(), 1, peerNode),
		}}},
	}, &out)
	out.Drain()

	// Nothing republishes: past two sync intervals the retained key leaves
	// the union and the counter records the release.
	core.Advance(now.Add(2*syncEvery+time.Millisecond), &out)
	out.Drain()
	if got := core.ListedKeysEvicted(); got != 1 {
		t.Fatalf("core.ListedKeysEvicted() = %d after the retention expiry, want 1", got)
	}

	// Two equal notifications of one tracked write: the second is the
	// not-greater-than-tracked regression the counter names.
	rec := storage.Record{Version: version.PackVersion(now.UnixMilli(), 5, localNode), Data: []byte("v5")}
	core.NotifyLocalWrite(now, key, rec, nil, &out)
	core.NotifyLocalWrite(now, key, rec, nil, &out)
	out.Drain()
	if got := core.NotificationsRegressed(); got != 1 {
		t.Fatalf("core.NotificationsRegressed() = %d after one equal duplicate, want 1", got)
	}

	// The two counters the earlier rounds left at zero need real samples on
	// both sides. Fresh first-seen Hellos each ask for a targeted listing;
	// past the destination cap every further trigger is refused and counted
	// (ListingDropped). Distinct sync_ids with a missing second chunk each
	// retain a partial round; past the stream bound every admission evicts
	// the least-progressed round and counts it (SyncStreamDropped). The peer
	// stays bound because the run id matches its Hello and the flood is well
	// inside the default peer timeout.
	later := now.Add(2*syncEvery + time.Millisecond)
	for i := range listingFlood {
		id := uint16(100 + i)
		core.HandleDatagram(later, port.InboundDatagram{
			Source:       "listing-flood",
			Status:       port.Decoded,
			ProtoVersion: repl.SupportedProtoVersion,
			NodeID:       id,
			RunID:        [16]byte{byte(id)},
			Msg:          port.Hello{},
		}, &out)
	}
	for i := range streamFlood {
		core.HandleDatagram(later, port.InboundDatagram{
			Source:       "n22",
			Status:       port.Decoded,
			ProtoVersion: repl.SupportedProtoVersion,
			NodeID:       peerNode,
			RunID:        [16]byte{byte(peerNode)},
			Msg: port.DataSync{
				SyncID:     uint64(1000 + i),
				ChunkIndex: 1,
				ChunkCount: 2,
				Entries: []port.SyncEntry{{
					Key:     "stream/part",
					Version: version.PackVersion(now.UnixMilli(), 1, peerNode),
				}},
			},
		}, &out)
	}
	out.Drain()
	if got := core.ListingDropped(); got == 0 {
		t.Fatalf("core.ListingDropped() = 0 after the %d-Hello destination flood, want a non-zero count", listingFlood)
	}
	if got := core.SyncStreamDropped(); got == 0 {
		t.Fatalf("core.SyncStreamDropped() = 0 after the %d-round accumulator flood, want a non-zero count", streamFlood)
	}

	adapter.mirrorStats()
	for _, tc := range []struct {
		name     string
		mirrored uint64
		core     uint64
	}{
		{"NotificationsRegressed", adapter.NotificationsRegressed(), core.NotificationsRegressed()},
		{"ListedKeysEvicted", adapter.ListedKeysEvicted(), core.ListedKeysEvicted()},
		{"ListingDropped", adapter.ListingDropped(), core.ListingDropped()},
		{"SyncStreamDropped", adapter.SyncStreamDropped(), core.SyncStreamDropped()},
	} {
		if tc.mirrored != tc.core {
			t.Errorf("%s() = %d, want the core's %d", tc.name, tc.mirrored, tc.core)
		}
	}
	if got := adapter.NotificationsRegressed(); got != 1 {
		t.Fatalf("NotificationsRegressed() = %d after the mirror pass, want 1", got)
	}
	if got := adapter.ListedKeysEvicted(); got != 1 {
		t.Fatalf("ListedKeysEvicted() = %d after the mirror pass, want 1", got)
	}
	if got := adapter.ListingDropped(); got == 0 {
		t.Fatal("ListingDropped() = 0 after the mirror pass, want the core's non-zero count")
	}
	if got := adapter.SyncStreamDropped(); got == 0 {
		t.Fatal("SyncStreamDropped() = 0 after the mirror pass, want the core's non-zero count")
	}
}
