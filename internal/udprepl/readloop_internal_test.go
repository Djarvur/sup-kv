package udprepl

// White-box on purpose (the *_internal_test.go filename is the configured
// testpackage exemption, the syncsize_internal_test.go precedent): the
// receive-path contract XPORT-08 pins — only net.ErrClosed may terminate the
// read loop — cannot be produced by a healthy socket on demand, so the
// unexported packetConn seam is driven by a scripted fake here. The
// production socket stays *net.UDPConn; only the interface differs.

import (
	"errors"
	"net"
	"sync"
	"syscall"
	"testing"
	"time"
)

// waitLoop polls cond every pollStep until it holds or waitDeadline — the
// bounded observation the two read-loop cases share.
const (
	waitLoopDeadline = 5 * time.Second
	waitLoopStep     = 5 * time.Millisecond
)

// The scripted read faults: package-level sentinels, because the lint gate's
// err113 rule forbids constructing errors at use sites and the tests only
// need distinguishable, stable values.
var (
	errFakeNoRawSocket     = errors.New("fakePacketConn: no raw socket")
	errInjectedReadFault   = errors.New("injected receive fault")
	errPersistentReadFault = errors.New("persistent receive fault")
)

// fakeRead is one scripted read result: a datagram to deliver or an error
// to return.
type fakeRead struct {
	data []byte
	err  error
}

// fakePacketConn is the scripted [packetConn] behind the read-loop tests.
// queued reads are served in order; an empty queue blocks until feed adds a
// read or Close fires (which makes the next read return net.ErrClosed, the
// real socket's closed-socket error). persistent makes every read return one
// error immediately, the shape of a kernel fault that outlasts the loop. The
// mutex is load-bearing: the read loop runs in its own goroutine while the
// test feeds and closes it.
type fakePacketConn struct {
	mu         sync.Mutex
	queue      []fakeRead
	persistent error
	closed     bool
	reads      int
	notify     chan struct{}
	closeCh    chan struct{}
	closeOnce  sync.Once
	source     *net.UDPAddr
}

// newFakePacketConn returns a fake whose datagrams carry writeAddr as their
// source, so the queued inbound events can be asserted byte- and
// address-exact.
func newFakePacketConn() *fakePacketConn {
	return &fakePacketConn{
		notify:  make(chan struct{}, 1),
		closeCh: make(chan struct{}),
		source:  &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 7900},
	}
}

// feed queues one datagram for the next read and wakes a blocked reader.
func (c *fakePacketConn) feed(data []byte) {
	c.mu.Lock()
	c.queue = append(c.queue, fakeRead{data: data})
	c.mu.Unlock()
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

// fail queues one error for the next read and wakes a blocked reader.
func (c *fakePacketConn) fail(err error) {
	c.mu.Lock()
	c.queue = append(c.queue, fakeRead{err: err})
	c.mu.Unlock()
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

// failEveryRead makes every subsequent read return err immediately.
func (c *fakePacketConn) failEveryRead(err error) {
	c.mu.Lock()
	c.persistent = err
	c.mu.Unlock()
}

// readsServed returns how many ReadFromUDP calls have been made — the
// no-spin evidence.
func (c *fakePacketConn) readsServed() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

// ReadFromUDP serves the scripted queue in order, then blocks until feed
// grows it or Close fires; once closed every call returns net.ErrClosed —
// exactly the errors.Is test the production loop must honor.
func (c *fakePacketConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	for {
		c.mu.Lock()
		c.reads++
		if c.closed {
			c.mu.Unlock()
			return 0, nil, net.ErrClosed
		}
		if c.persistent != nil {
			err := c.persistent
			c.mu.Unlock()
			return 0, nil, err
		}
		if len(c.queue) > 0 {
			step := c.queue[0]
			c.queue = c.queue[1:]
			c.mu.Unlock()
			if step.err != nil {
				return 0, nil, step.err
			}
			n := copy(b, step.data)
			return n, c.source, nil
		}
		c.mu.Unlock()
		select {
		case <-c.notify:
		case <-c.closeCh:
		}
	}
}

// WriteToUDP accepts and discards one datagram — the read-loop tests never
// exercise the send path.
func (*fakePacketConn) WriteToUDP(b []byte, _ *net.UDPAddr) (int, error) {
	return len(b), nil
}

// SetReadBuffer is a no-op: no kernel socket exists here.
func (*fakePacketConn) SetReadBuffer(_ int) error { return nil }

// SetWriteBuffer is a no-op: no kernel socket exists here.
func (*fakePacketConn) SetWriteBuffer(_ int) error { return nil }

// SyscallConn reports the absent raw socket; the read-loop tests never read
// socket options through it.
func (*fakePacketConn) SyscallConn() (syscall.RawConn, error) {
	return nil, errFakeNoRawSocket
}

// LocalAddr returns the scripted datagram source address.
func (c *fakePacketConn) LocalAddr() net.Addr { return c.source }

// Close flips the closed state and unblocks a waiting reader; repeated calls
// are safe (the real Close's contract the loop relies on).
func (c *fakePacketConn) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	c.closeOnce.Do(func() { close(c.closeCh) })
	return nil
}

// waitReadLoop polls cond every waitLoopStep until it holds or the deadline
// expires.
func waitReadLoop(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitLoopDeadline)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(waitLoopStep)
	}
	t.Fatalf("timed out after %v waiting for %s", waitLoopDeadline, what)
}

// TestReadLoop_survivesNonClosedError pins XPORT-08's continuation half: a
// non-closed read error is counted, backed off and survived — the next
// datagram still reaches inbound — while a subsequent Close, whose read
// returns net.ErrClosed, is the one and only exit.
func TestReadLoop_survivesNonClosedError(t *testing.T) {
	t.Parallel()
	conn := newFakePacketConn()
	adapter := newAdapterWithConn(conn)
	done := make(chan struct{})
	go func() {
		defer close(done)
		adapter.readLoop()
	}()
	t.Cleanup(func() {
		_ = conn.Close()
		<-done
	})

	// Script one fault followed by one real datagram; the backoff starts
	// before the datagram can be read, so the measured wait is the pause.
	start := time.Now()
	conn.fail(errInjectedReadFault)
	conn.feed([]byte("after-the-fault"))

	var ev inboundEvent
	select {
	case ev = <-adapter.inbound:
	case <-time.After(waitLoopDeadline):
		t.Fatalf("no datagram reached inbound within %v after a non-closed read error — the loop died or stopped reading", waitLoopDeadline)
	}
	elapsed := time.Since(start)
	if got := string(ev.datagram); got != "after-the-fault" {
		t.Fatalf("inbound datagram = %q, want %q", got, "after-the-fault")
	}
	if ev.source != conn.source.String() {
		t.Fatalf("inbound source = %q, want %q", ev.source, conn.source.String())
	}
	if elapsed < readErrorBackoff {
		t.Fatalf("the datagram after the error arrived in %v, want at least the %v read-error backoff", elapsed, readErrorBackoff)
	}
	if got := adapter.ReadErrors(); got != 1 {
		t.Fatalf("ReadErrors() = %d after one non-closed read error, want 1", got)
	}

	// net.ErrClosed is the only exit: closing the fake returns the loop.
	_ = conn.Close()
	select {
	case <-done:
	case <-time.After(waitLoopDeadline):
		t.Fatalf("read loop still running %v after Close returned net.ErrClosed", waitLoopDeadline)
	}
}

// TestReadLoop_persistentErrorExitsOnClose pins XPORT-08's bound half: a
// persistent read error never terminates the loop and never spins it — the
// counter climbs at the backoff cadence — and Close still returns it through
// net.ErrClosed.
func TestReadLoop_persistentErrorExitsOnClose(t *testing.T) {
	t.Parallel()
	conn := newFakePacketConn()
	conn.failEveryRead(errPersistentReadFault)
	adapter := newAdapterWithConn(conn)
	done := make(chan struct{})
	go func() {
		defer close(done)
		adapter.readLoop()
	}()

	// Two survived errors prove continuation; at the 100 ms backoff that is
	// a bounded observation window, never a spin.
	waitReadLoop(t, "two survived read errors", func() bool {
		return adapter.ReadErrors() >= 2
	})
	_ = conn.Close()
	select {
	case <-done:
	case <-time.After(waitLoopDeadline):
		t.Fatalf("read loop still running %v after Close returned net.ErrClosed", waitLoopDeadline)
	}
	if got := adapter.ReadErrors(); got < 2 {
		t.Fatalf("ReadErrors() = %d, want at least 2 persistent errors survived", got)
	}
	// Three reads at the backoff cadence (two faults and the closed read) —
	// a spin would be thousands. The bound is loose to stay scheduler-safe.
	if reads := conn.readsServed(); reads > 20 {
		t.Fatalf("read loop served %d reads for 2 backed-off errors, want a handful — the loop is spinning", reads)
	}
}
