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
