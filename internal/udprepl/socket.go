package udprepl

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"syscall"
)

const (
	// datagramBufSize is the one receive buffer size for the socket's
	// lifetime: 65 507 bytes is the maximum IPv4 UDP payload (65 535 − 20
	// IP − 8 UDP headers). A smaller buffer silently truncates reads — Go
	// surfaces no MSG_TRUNC — so large records would arrive as quiet
	// corruption surfacing only as bad-CRC discards (RESEARCH Pitfall 4).
	datagramBufSize = 65507

	// socketRecvBuffer and socketSendBuffer are the REQUESTED kernel
	// socket-buffer sizes. The kernel doubles each request and clamps it to
	// rmem_max/wmem_max, and getsockopt returns the doubled, clamped value
	// (RESEARCH Pattern 7); the requested constants are therefore never the
	// honest socket sizes and never reach the startup log — only the
	// read-back values do. darwin's ~9 KiB datagram send limit
	// (net.inet.udp.maxdgram) is documented by ADR 0010, not fixed here.
	socketRecvBuffer = 4 << 20
	socketSendBuffer = 1 << 20

	// maxBufferRequestWidth bounds a requested socket-buffer width: the
	// kernel doubles the request, so a larger request would overflow a
	// 32-bit int on read-back (T-3-01).
	maxBufferRequestWidth = math.MaxInt32 / 2
)

// errBufferRequestRange and errSendBufferTooSmall are the socket-buffer
// configuration sentinels: the request-width guard (T-3-01) and the
// one-maximal-datagram send floor (T-3-03) wrap them with the compared
// sizes, so callers can match the class without parsing the numbers.
// errUnexpectedSocketType guards the one assertion the udp4 listen path needs
// ([net.PacketConn] to [*net.UDPConn]): unreachable with the standard library,
// but a construction failure must return, never panic, if it ever were.
var (
	errBufferRequestRange   = errors.New("udprepl: socket buffer request out of range")
	errSendBufferTooSmall   = errors.New("udprepl: effective send buffer cannot hold one maximal datagram")
	errUnexpectedSocketType = errors.New("udprepl: unexpected socket type from udp4 listen")
)

// packetConn is the adapter's socket seam: the datagram operations the
// replication adapter uses, exactly as [*net.UDPConn] provides them. The
// interface exists so the receive path can be driven by a fake in the
// white-box read-loop test (XPORT-08) — a non-closed read error must be
// provably survivable, and no real socket produces one on demand. LocalAddr
// joins the read/write/buffer/raw/close set because construction derives the
// core's own opaque handle from it (D-03).
type packetConn interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	SetReadBuffer(size int) error
	SetWriteBuffer(size int) error
	SyscallConn() (syscall.RawConn, error)
	LocalAddr() net.Addr
	Close() error
}

// options is the resolved construction option set — the values [New]
// threads into its socket construction.
type options struct {
	// addressReuse requests SO_REUSEADDR before bind. Test-only: production
	// never sets it (XPORT-09, T-3-34).
	addressReuse bool
}

// Option adjusts one aspect of [New]'s socket construction. The only
// production-relevant shape is the default: no options at all.
type Option func(*options)

// WithAddressReuse makes [New] set SO_REUSEADDR before bind, restoring the
// same-host shared-port topology the real-socket smoke test needs. It is
// explicitly test-only: in production a second process binding the node's
// address must fail with EADDRINUSE rather than silently split unicast
// traffic between two sockets the kernel chooses between (XPORT-09).
func WithAddressReuse() Option {
	return func(o *options) {
		o.addressReuse = true
	}
}

// listenDatagram binds the udp4 socket. Only when the resolved options ask
// for address reuse does it set SO_REUSEADDR through a pre-bind Control hook;
// production resolves the zero options, so a second process binding the same
// address fails with EADDRINUSE and can never silently split the node's
// unicast traffic between two sockets (XPORT-09, T-3-34). SO_BROADCAST is
// deliberately absent: Go sets it on every datagram socket (RESEARCH
// Pattern 7), and setting it here would only disguise that the reuse option
// is the sole socket-option concern.
func listenDatagram(bindAddr string, o options) (*net.UDPConn, error) {
	var lc net.ListenConfig
	if o.addressReuse {
		var opErr error
		lc.Control = func(_, _ string, c syscall.RawConn) error {
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			})
			if err != nil {
				return fmt.Errorf("socket control: %w", err)
			}
			if opErr != nil {
				return fmt.Errorf("socket options: %w", opErr)
			}
			return nil
		}
	}
	packetConn, err := lc.ListenPacket(context.Background(), "udp4", bindAddr)
	if err != nil {
		return nil, err
	}
	conn, ok := packetConn.(*net.UDPConn)
	if !ok {
		_ = packetConn.Close()
		return nil, fmt.Errorf("%w: got %T", errUnexpectedSocketType, packetConn)
	}
	return conn, nil
}

// configureSocketBuffers applies the adapter's requested SO_RCVBUF/SO_SNDBUF
// sizes and returns the kernel's effective values read back through
// getsockopt. The kernel doubles each request and clamps it to
// rmem_max/wmem_max, so the returned values are the honest socket sizes and
// the only ones safe to report (RESEARCH Pattern 7).
//
// The requested widths are validated before any setsockopt: their doubled
// read-back must still fit a 32-bit int (T-3-01). Every failure is wrapped
// with the operation and the requested size; a non-nil error means the
// socket must not be used.
func configureSocketBuffers(conn *net.UDPConn) (recv, send int, err error) {
	err = validateBufferRequest(socketRecvBuffer)
	if err != nil {
		return 0, 0, err
	}
	err = validateBufferRequest(socketSendBuffer)
	if err != nil {
		return 0, 0, err
	}
	err = conn.SetReadBuffer(socketRecvBuffer)
	if err != nil {
		return 0, 0, fmt.Errorf("udprepl: set receive buffer %d: %w", socketRecvBuffer, err)
	}
	err = conn.SetWriteBuffer(socketSendBuffer)
	if err != nil {
		return 0, 0, fmt.Errorf("udprepl: set send buffer %d: %w", socketSendBuffer, err)
	}
	recv, err = socketOption(conn, syscall.SO_RCVBUF)
	if err != nil {
		return 0, 0, fmt.Errorf("udprepl: read receive buffer after requesting %d: %w", socketRecvBuffer, err)
	}
	send, err = socketOption(conn, syscall.SO_SNDBUF)
	if err != nil {
		return 0, 0, fmt.Errorf("udprepl: read send buffer after requesting %d: %w", socketSendBuffer, err)
	}
	return recv, send, nil
}

// validateBufferRequest rejects a requested buffer width the kernel read-back
// cannot represent honestly: a non-positive request is meaningless, and a
// request above MaxInt32/2 would overflow a 32-bit int once the kernel
// doubles it (T-3-01). The doubled read-back itself is never computed in Go
// code — the kernel returns it directly.
func validateBufferRequest(requested int) error {
	if requested <= 0 || requested > maxBufferRequestWidth {
		return fmt.Errorf("%w: %d (valid 1..%d)", errBufferRequestRange, requested, maxBufferRequestWidth)
	}
	return nil
}

// socketOption reads one SOL_SOCKET integer option from conn's descriptor —
// the getsockopt read-back behind configureSocketBuffers.
func socketOption(conn *net.UDPConn, opt int) (int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("udprepl: socket syscall conn: %w", err)
	}
	var value int
	var opErr error
	err = raw.Control(func(fd uintptr) {
		value, opErr = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, opt)
	})
	if err != nil {
		return 0, fmt.Errorf("udprepl: socket control: %w", err)
	}
	if opErr != nil {
		return 0, fmt.Errorf("udprepl: getsockopt option %d: %w", opt, opErr)
	}
	return value, nil
}

// checkSendBuffer enforces the startup floor (T-3-03): the effective send
// buffer must hold one maximal datagram, or every largest record would be
// rejected by the kernel while the node appeared healthy. The error names
// both the actual and the required size so the operator can compare them
// against wmem_max. A pure function — no socket involved.
func checkSendBuffer(actual int) error {
	if actual < datagramBufSize {
		return fmt.Errorf("%w: effective %d bytes, need %d bytes", errSendBufferTooSmall, actual, datagramBufSize)
	}
	return nil
}
