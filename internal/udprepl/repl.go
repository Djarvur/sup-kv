// Package udprepl is the sup-kv UDP replication adapter: the only net- and
// protobuf-importing replication code, mechanical per D-01/D-02 — it maps
// the port message structs onto the wire schema, verifies the CRC before
// any body decode, owns the socket, and feeds the pure core
// (internal/app/repl) from one event-loop goroutine (D-04) — mirroring
// persistfile's adapter role. It owns no protocol decision: discards,
// the peer table and every timer live in the core. Production binds never
// set SO_REUSEADDR, so a second process on the node's address fails with
// EADDRINUSE instead of silently splitting unicast traffic (XPORT-09); only
// the explicit test-only [WithAddressReuse] option restores same-host port
// sharing. Same-host caveat under that option: two nodes sharing one UDP
// port receive each other's broadcasts, but unicast to the shared port is
// delivered to exactly one socket, the kernel's choice — the real-socket
// smoke test therefore asserts only broadcast-delivered behavior.
package udprepl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"log/slog"
	"math"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// maxNodeIDWire is the highest node_id the 16-bit NodeID domain admits;
// an envelope beyond it is schema-malformed and classifies Undecodable.
const maxNodeIDWire = 0xFFFF

// wireIDSize is the wire size of both the run_id and the RFC 9562
// RecordVersion fields: exactly 16 bytes each.
const wireIDSize = 16

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

	// inboundQueue bounds the datagrams waiting for the event loop; beyond
	// it the read goroutine blocks and the socket receive buffer
	// backpressences the segment.
	inboundQueue = 64

	// notifyQueue bounds the local-write notifications waiting for the
	// event loop. NotifyLocalWrite only enqueues and never blocks on
	// replication progress (D-05 — the write path stays available); a
	// wedged loop eventually fills the queue and blocks authors, a
	// bounded backpressure that never silently drops a write.
	notifyQueue = 1024

	// readErrorBackoff is the pause after a non-terminal receive error
	// (XPORT-08): a persistent kernel error must never turn the read loop
	// into a busy spin — the counter and the WARN are the signals, and the
	// loop keeps reading the next datagram after the pause.
	readErrorBackoff = 100 * time.Millisecond

	// readErrorLogInterval is the read-error rate limit: at most one log
	// line per interval keeps a flood of receive errors from flooding the
	// log (T-3-36) while the counter carries the reliable signal. Adapter
	// side, so the wall clock is the sanctioned time source here.
	readErrorLogInterval = time.Second
)

// errNoStandaloneBody rejects a port message that has no datagram body of
// its own on the wire schema.
var errNoStandaloneBody = errors.New("udprepl: message has no standalone datagram body")

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

// ErrRunLoopExited is the NotifyLocalWrite verdict once the adapter's event
// loop has returned: nothing drains the notify queue any more, so the
// notification is dropped rather than enqueued — the caller (Phase 5's
// gRPC write path) fails the write visibly instead of the call blocking
// forever on a queue nobody serves.
var ErrRunLoopExited = errors.New("udprepl: replication event loop has exited")

// inboundEvent is one received datagram on the event loop's inbound queue,
// with the bytes defensively copied off the read buffer (which is reused on
// the next read) and the sender's address as the opaque handle string
// (src.String(), D-03).
type inboundEvent struct {
	datagram []byte
	source   string
}

// localWrite is one author's NotifyLocalWrite notification on the event
// loop's queue, with the record defensively copied at enqueue — the author
// may reuse its buffer once the call returns. The queue carries no track
// flag: the tracker is created synchronously in NotifyLocalWrite before the
// enqueue, so no later code can mistake a notification for a tracker
// creation site.
// localWrite is one queued author notification: the key and the defensively
// copied record, plus whether the record is a REPLAY of already-installed
// state (a snapshot load) rather than a live write. A replay takes the core's
// paced record-send path instead of the immediate broadcast (WR-04).
type localWrite struct {
	key    string
	rec    storage.Record
	replay bool
}

// trackerKey identifies one wait-requesting local write: the key and the
// exact record version the wait observes. The version is part of the
// identity, never a field compared later, so a superseded version can never
// be confused with the write that replaced it.
type trackerKey struct {
	key     string
	version version.RecordVersion
}

// tracker is one write's replication observation state — the adapter-side
// waiter the core's result stream resolves. The mutex guards acked,
// resolved and ok; done is closed exactly once, on the first resolution,
// while the mutex is held, so a waiter waking on it observes the final
// state. The acked set is sorted ascending and idempotent (a re-Ack adds no
// second entry): full coverage means len(acked) reaches the write-time
// peer total.
type tracker struct {
	mu       sync.Mutex
	acked    []uint16
	resolved bool
	ok       bool
	done     chan struct{}
}

// newTracker returns a tracker with its done channel armed and no
// resolution yet.
func newTracker() *tracker {
	return &tracker{done: make(chan struct{})}
}

// resolveLocked records the first resolution and closes done; later results
// are ignored, so done is closed exactly once. The caller holds t.mu.
func (t *tracker) resolveLocked(ok bool) {
	if t.resolved {
		return
	}
	t.resolved = true
	t.ok = ok
	close(t.done)
}

// insertPeer returns peers with id inserted at its sorted position, or
// unchanged when already present — the sorted, idempotent Acked discipline
// the core's ackedInsert applies.
func insertPeer(peers []uint16, id uint16) []uint16 {
	pos, found := slices.BinarySearch(peers, id)
	if found {
		return peers
	}
	return slices.Insert(peers, pos, id)
}

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

// Adapter is the UDP replication adapter for one node: it owns the bound
// socket and drives one repl.Core. Exactly one goroutine — the caller's
// Run — mutates core state (D-04); the read goroutine only queues inbound
// datagrams, write authors only queue notifications and create waiter
// trackers, and the discard counters, known-peer count and the replication
// read states (the first-sync signal and each write's tracker) are
// mutex- or atomic-guarded so cross-goroutine readers never touch core
// state.
type Adapter struct {
	conn          packetConn
	broadcastAddr *net.UDPAddr
	nodeID        uint16
	runID         [16]byte
	core          *repl.Core
	log           func(msg string)
	inbound       chan inboundEvent
	notify        chan localWrite
	done          chan struct{}

	// recvBuf and sendBuf are the kernel's effective SO_RCVBUF/SO_SNDBUF
	// sizes read back in New — the doubled, clamped values, never the
	// requested constants (RESEARCH Pattern 7). Written once during
	// construction and immutable afterwards; SocketBuffers is the read
	// accessor.
	recvBuf int
	sendBuf int

	badCRC          atomic.Uint64
	undecodable     atomic.Uint64
	badProtoVersion atomic.Uint64
	ownEcho         atomic.Uint64
	duplicateNodeID atomic.Uint64
	emptyKey        atomic.Uint64
	keyTooLong      atomic.Uint64
	tombstoneData   atomic.Uint64
	badChecksum     atomic.Uint64
	badVersion      atomic.Uint64
	badSyncHeader   atomic.Uint64
	peerCount       atomic.Uint64

	// readErrors counts every receive-path error the read loop survived
	// (XPORT-08, T-3-33): the reliable deafness signal — an error that ends
	// the loop would let Hello broadcasts continue while nothing is read.
	// ReadErrors is the cross-goroutine read seam.
	readErrors atomic.Uint64

	// readErrMu guards readErrLogAt, the read-error rate limiter's last-log
	// wall clock (readErrorLogInterval). Read-goroutine-only mutation, but
	// the mutex keeps the field race-free if a second readLoop were ever
	// launched.
	readErrMu    sync.Mutex
	readErrLogAt time.Time

	// The OBS-01 mirror set: the pending-set gauge (unacked count and
	// oldest age), the remote-merge counters and the retransmissions
	// counter — all copied from the single-driver core after every loop
	// event (the Discards precedent) — plus the sent-by-type datagram
	// counts, incremented in flush where the adapter itself sees every
	// outbound message cross the encode step.
	unackedCount      atomic.Int64
	unackedOldestNs   atomic.Int64
	mergeConflicts    atomic.Uint64
	mergeDriftWarning atomic.Uint64
	retransmitted     atomic.Uint64
	sentHello         atomic.Uint64
	sentData          atomic.Uint64
	sentAck           atomic.Uint64
	sentDataRequest   atomic.Uint64
	sentSync          atomic.Uint64

	// The XPORT-07 emission set: the total datagram-queue depth (listing
	// chunks, record sends and egress responses), the egress overflow drops
	// and the paced emission count — core gauges/counters mirrored after
	// every loop event like the retransmissions counter.
	sendQueue     atomic.Int64
	egressDropped atomic.Uint64
	pacedSends    atomic.Uint64

	// listing is the gate's one publication point: an immutable snapshot of
	// the core's listing state — whether a complete listing has been
	// processed and the union of its keys — stored as ONE pointer, so a
	// reader that observes complete=true structurally also observes the keys
	// that complete listing named (PROTO-01, T-4-05, RESEARCH Pattern 2).
	// Publishing completeness and keys separately would leave a window where
	// the gate opens before ListedKey can answer. The event loop stores a
	// fresh pointer only when the core's listing generation changed, so an
	// identical repeated listing clones nothing (adapter-goroutine writes,
	// any-goroutine reads).
	listing    atomic.Pointer[listingMirror]
	listingGen uint64

	// trackers maps each wait-requesting local write's identity to its
	// waiter tracker (XPORT-12). NotifyLocalWrite is the ONLY production
	// creation site — synchronously, before the notification is enqueued —
	// so a wait starting immediately after that call always finds it. The
	// Run loop only looks trackers up: a result for an absent identity is
	// dropped and never causes a map insert, and every WaitReplicated return
	// path (resolution, zero-peer release, ctx cancel, timeout) deletes its
	// entry, so the map is bounded by concurrent waits.
	trackersMu sync.Mutex
	trackers   map[trackerKey]*tracker
}

// New binds the node's udp4 socket and constructs the replication core over
// it. bindAddr is the local bind address (for example "0.0.0.0:7900") and
// broadcastAddr the segment broadcast address datagrams are sent to (for
// example "127.255.255.255:7900"); the core's own opaque handle (D-03) is
// taken from the bound socket's local address. Production binds set no
// address-reuse option at all (XPORT-09, T-3-34): a second process binding
// the same address fails with EADDRINUSE instead of silently splitting
// traffic. SO_BROADCAST needs no call — Go sets it on every datagram socket
// (RESEARCH Pattern 7). The only option, [WithAddressReuse], is test-only.
//
// After the bind the socket's SO_RCVBUF/SO_SNDBUF are configured and read
// back: the values [Adapter.SocketBuffers] reports are the kernel's
// EFFECTIVE (doubled and clamped) sizes, never the requested constants
// (RESEARCH Pattern 7). A construction error is returned when the socket
// options cannot be applied or read, and when the effective send buffer
// cannot hold one maximal datagram — the node never starts in the
// silently-degraded state where the largest record could never leave.
//
// A nil cfg.OnDiscard is replaced by the adapter's own slog bridge
// (logDiscard, D-08): production discards surface as distinguishable
// structured events with no wiring at the call site.
func New(bindAddr, broadcastAddr string, nodeID uint16, store repl.StateStore, cfg repl.Config, opts ...Option) (*Adapter, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if cfg.OnDiscard == nil {
		cfg.OnDiscard = logDiscard
	}
	bound, err := listenDatagram(bindAddr, o)
	if err != nil {
		return nil, fmt.Errorf("udprepl: bind udp4 %s: %w", bindAddr, err)
	}
	conn := bound
	broadcast, err := net.ResolveUDPAddr("udp4", broadcastAddr)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("udprepl: resolve broadcast %s: %w", broadcastAddr, err)
	}
	logFn := cfg.Log
	if logFn == nil {
		logFn = func(_ string) {}
	}
	runID := repl.NewRunID()
	recvBuf, sendBuf, err := configureSocketBuffers(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("udprepl: configure socket buffers: %w", err)
	}
	err = checkSendBuffer(sendBuf)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Adapter{
		conn:          conn,
		broadcastAddr: broadcast,
		nodeID:        nodeID,
		runID:         runID,
		core:          repl.New(nodeID, runID, store, cfg),
		log:           logFn,
		inbound:       make(chan inboundEvent, inboundQueue),
		notify:        make(chan localWrite, notifyQueue),
		done:          make(chan struct{}),
		trackers:      make(map[trackerKey]*tracker),
		recvBuf:       recvBuf,
		sendBuf:       sendBuf,
	}, nil
}

// logDiscard is the adapter's default repl.Config.OnDiscard bridge (D-08,
// D-09): one structured event per discarded datagram with the class token
// and the metadata detail as attrs, so every discard class stays
// distinguishable in the log stream while the core stays silent — the class
// arrives as a first-class value, never a message prefix to parse. Debug
// level is deliberate hot-path discipline (T-6-06, 06-RESEARCH Pattern 4):
// an attacker on the segment can flood discard-class events, and at the
// default info level they are suppressed while the adapter's O(1) discard
// counters (Discards) carry the reliable signal — only an
// operator-configured SUPKV_LOG_LEVEL=debug ever lets a flood reach stderr.
func logDiscard(class, detail string) {
	slog.Debug("Discarded datagram", "class", class, "detail", detail)
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

// Run is the single event-loop goroutine the caller launches (D-04). It
// starts the read goroutine, emits the startup Hello through the first
// Advance (the core's zero initial hello deadline), then selects over
// context cancellation, inbound datagrams, local-write notifications and
// the timer channel re-armed from Advance's returned deadline. After every
// event it calls the core handler with [time.Now] (the production now
// seam), drains the shared Outbox's message stream through encode+send,
// applies the result stream to the waiter trackers, and mirrors the core's
// counters and replication read states into atomics. Run returns ctx's
// error; closing the context unblocks both the loop and the read goroutine.
func (a *Adapter) Run(ctx context.Context) error {
	go a.readLoop()
	defer func() { _ = a.conn.Close() }() // unblocks the read goroutine's ReadFromUDP.
	defer close(a.done)                   // unblocks the read goroutine's queued send.
	var out port.Outbox
	next := a.core.Advance(time.Now(), &out)
	a.flush(&out)
	a.applyResults(&out)
	var timer *time.Timer
	for {
		timer = armTimer(timer, next)
		select {
		case <-ctx.Done():
			stopTimer(timer)
			return ctx.Err()
		case ev := <-a.inbound:
			in := decode(ev.datagram)
			in.Source = ev.source
			a.core.HandleDatagram(time.Now(), in, &out)
		case lw := <-a.notify:
			// The tracker, when the write requested one, was created in
			// NotifyLocalWrite BEFORE this enqueue. Neither this case nor
			// the core handler creates one: a result for an absent trackers
			// entry is dropped (applyResults), never an insert. A replayed
			// record (a snapshot load) rides the paced record-send queue,
			// never an immediate broadcast (WR-04).
			if lw.replay {
				a.core.NotifyReplayedWrite(time.Now(), lw.key, lw.rec, &out)
			} else {
				a.core.NotifyLocalWrite(time.Now(), lw.key, lw.rec, &out)
			}
		case <-timerChan(timer):
			// A due deadline — the Advance below fires the due timers.
		}
		next = a.core.Advance(time.Now(), &out)
		a.flush(&out)
		a.applyResults(&out)
		a.mirrorStats()
	}
}

// NotifyLocalWrite enqueues the explicit post-write replication call
// (D-05) for the event loop: it defensively copies the record and returns
// without doing any protocol work, so the write path stays available. The
// event loop's core handler broadcasts the DataReplication. When track is
// true the write asked for a wait_for_replication report, and the tracker
// for its exact (key, version) identity is created HERE — synchronously,
// under trackersMu, BEFORE the channel send (XPORT-12): the gRPC handler
// calls [Adapter.WaitReplicated] immediately after this method returns, so
// a tracker created later (say, when the loop dequeues the notification)
// would be missed, and the old appearance race the polling mirror worked
// around could return. This is the sole production creation site — the
// queue struct carries no track flag, so no later code can create a
// tracker from a notification either.
//
// The enqueue and the closed-loop decision are one select over the notify
// channel and the loop's done channel, with a re-check after a won send
// (D-15 P4-IN-01 — the former pre-check-then-enqueue pair raced Run's exit
// and could report nil for a notification nobody will ever drain): while
// Run is active the send enqueues into the bounded 1024-deep buffer (D-05)
// and reports nil; once Run has returned, done is closed, so either the
// select's done branch or — when a buffered send still won the uniform
// choice against the already closed done — the post-send re-check reports
// [ErrRunLoopExited]. A failed notify deletes the just-created tracker
// before returning, so the zero-value path (Run already gone, no waiter
// will ever resolve it) cannot leak a map entry. A writer never blocks
// forever on a loop that no longer receives, never panics, and the drop is
// surfaced to the author, never silent.
//
//nolint:revive // flag-parameter: track is the wait_for_replication request crossing the D-05 notification seam (XPORT-12); a second method would split the producer contract.
func (a *Adapter) NotifyLocalWrite(key string, rec storage.Record, track bool) error {
	keyID := trackerKey{key: key, version: rec.Version}
	if track {
		a.ensureTracker(keyID)
	}
	err := a.enqueueLocalWrite(localWrite{key: key, rec: cloneLocalRecord(rec)})
	if err != nil {
		// A failed notify deletes the just-created tracker before returning, so
		// the zero-value path (Run already gone, no waiter will ever resolve it)
		// cannot leak a map entry. An absent tracker makes this a no-op.
		a.deleteTracker(keyID)
		return err
	}
	return nil
}

// NotifyReplayedWrite enqueues one snapshot-load (or recovery-replay) record
// for the event loop, where the core puts it on the PACED record-send queue
// instead of broadcasting immediately (WR-04, T-4-20): a large load drains one
// datagram per pacing slot rather than emitting one broadcast per key
// back-to-back. No tracker is created — a load has no wait_for_replication
// report — and the copy rules are NotifyLocalWrite's.
func (a *Adapter) NotifyReplayedWrite(key string, rec storage.Record) error {
	return a.enqueueLocalWrite(localWrite{key: key, rec: cloneLocalRecord(rec), replay: true})
}

// cloneLocalRecord defensively copies one record for the notification
// channel: the author may reuse its buffer once the call returns.
func cloneLocalRecord(rec storage.Record) storage.Record {
	return storage.Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Deleted:  rec.Deleted,
		Checksum: rec.Checksum,
	}
}

// enqueueLocalWrite is the notification channel handoff shared by the live and
// replay seams. The enqueue and the closed-loop decision are one select over
// the notify channel and the loop's done channel, with a re-check after a won
// send (D-15 P4-IN-01 — the former pre-check-then-enqueue pair raced Run's
// exit and could report nil for a notification nobody will ever drain): while
// Run is active the send enqueues into the bounded 1024-deep buffer (D-05) and
// reports nil; once Run has returned, done is closed, so either the select's
// done branch or — when a buffered send still won the uniform choice against
// the already closed done — the post-send re-check reports
// [ErrRunLoopExited]. A writer never blocks forever on a loop that no longer
// receives, never panics, and the drop is surfaced to the author, never
// silent.
func (a *Adapter) enqueueLocalWrite(lw localWrite) error {
	select {
	case a.notify <- lw:
		// The send won, but a send into a buffer nobody drains proves
		// nothing on its own: when the select ran with done already
		// closed, the uniform choice may still have picked the buffered
		// send. The re-check turns that overtaken enqueue into the same
		// deterministic error — never a nil report for a dropped write.
		select {
		case <-a.done:
			return ErrRunLoopExited
		default:
		}
		return nil
	case <-a.done:
		return ErrRunLoopExited
	}
}

// ensureTracker returns the tracker for k, creating it when absent. Only
// NotifyLocalWrite calls this — the sole production creation site; a
// tracker already present for the same identity is reused (a repeated
// tracked notification of one write shares its resolution).
func (a *Adapter) ensureTracker(k trackerKey) *tracker {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	if tr := a.trackers[k]; tr != nil {
		return tr
	}
	tr := newTracker()
	a.trackers[k] = tr
	return tr
}

// lookupTracker returns the tracker for k, or nil when the write was not
// tracked. Every caller drops a nil result — no path inserts through a
// lookup.
func (a *Adapter) lookupTracker(k trackerKey) *tracker {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	return a.trackers[k]
}

// deleteTracker removes k's tracker; a no-op when absent. Every
// WaitReplicated return path and a failed notify call it, so the map stays
// bounded by concurrent waits (T-3-16).
func (a *Adapter) deleteTracker(k trackerKey) {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	delete(a.trackers, k)
}

// applyResults consumes the core's result stream after one handler
// (XPORT-12): every result resolves or accumulates on the tracker for its
// exact (key, version) identity. ResultAck inserts the peer into acked
// without resolving (an Ack alone is not full coverage); ResultDrained
// resolves ok=true (full write-time coverage); ResultSuperseded and
// ResultWaitSetEmpty resolve ok=false (honest false — the tracked version
// lost, or its last waiter was evicted). A result for an absent tracker is
// DROPPED, never a map insert: NotifyLocalWrite is the sole creation site,
// so an absent identity is an untracked write (track=false) or a
// directly-driven core in a white-box test, and neither has a waiter.
// Event-loop goroutine only; each tracker's own mutex is the
// cross-goroutine edge.
func (a *Adapter) applyResults(out *port.Outbox) {
	for _, res := range out.DrainResults() {
		tr := a.lookupTracker(trackerKey{key: res.Key, version: res.Version})
		if tr == nil {
			continue
		}
		tr.mu.Lock()
		switch res.Kind {
		case port.ResultAck:
			tr.acked = insertPeer(tr.acked, res.Peer)
		case port.ResultDrained:
			tr.resolveLocked(true)
		case port.ResultSuperseded, port.ResultWaitSetEmpty:
			tr.resolveLocked(false)
		}
		tr.mu.Unlock()
	}
}

// WaitReplicated waits for one wait-requesting local write's replication
// outcome — the adapter half of the wait_for_replication seam (XPORT-12,
// D-04). total is the write-time peer snapshot the caller captured; the
// return is the observed report: acked is true only on full write-time
// coverage, and ackedCount is the observed Ack count clamped to total
// (WR-01: a matching Ack from a node outside the write-time snapshot still
// counts in the core, so the raw length can exceed total).
//
// The success rule is resolved && ok && clamped == total: full write-time
// coverage is the only success. Eviction (ResultWaitSetEmpty) and
// supersession (ResultSuperseded) are honest false reports, and a timeout
// or ctx cancellation returns the last observed count with acked=false —
// the wait never reports a success it did not observe.
//
// total == 0 is the D-11 single-node case: a write with no peers is acked
// immediately with nothing awaited, and this call is the RELEASE point for
// the tracker the adapter created at notify time (the zero-peer path never
// enters the waiting loop). An absent tracker reports an immediate
// (false, 0) without polling: two paths reach it — a write notified with
// track=false, and a white-box test driving the core directly — and
// neither has a waiter to resolve.
//
// One wait per write identity is expected (the gRPC handler issues exactly
// one). A second concurrent wait for the same identity shares the same
// tracker's resolution; the first return deletes the map entry, and the
// second reads the already-final tracker state before its own delete is a
// no-op. Every return path deletes the tracker, so the map is bounded by
// concurrent waits (T-3-16).
func (a *Adapter) WaitReplicated(ctx context.Context, key string, ver version.RecordVersion, total uint32, timeout time.Duration) (acked bool, ackedCount uint32) {
	k := trackerKey{key: key, version: ver}
	if total == 0 {
		a.deleteTracker(k)
		return true, 0
	}
	tr := a.lookupTracker(k)
	if tr == nil {
		return false, 0
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-tr.done:
	case <-ctx.Done():
	case <-timer.C:
	}
	tr.mu.Lock()
	// The acked set holds uint16 peer ids, so its length cannot truncate.
	count := min(uint32(len(tr.acked)), total) //nolint:gosec // G115: len(acked) <= 65536 by the uint16 peer-id domain.
	ok := tr.resolved && tr.ok && count == total
	tr.mu.Unlock()
	a.deleteTracker(k)
	return ok, count
}

// SocketBuffers returns the kernel's effective SO_RCVBUF/SO_SNDBUF sizes read
// back during [New] — the doubled, clamped values, never the requested
// constants (RESEARCH Pattern 7). The fields are written once at construction
// and immutable afterwards, so the accessor is safe from any goroutine.
func (a *Adapter) SocketBuffers() (recv, send int) {
	return a.recvBuf, a.sendBuf
}

// Discards returns the discard-class counters (D-02) mirrored after the
// latest event-loop iteration — safe from any goroutine. The wire-validation
// classes (empty_key, key_too_long, tombstone_with_data, bad_checksum,
// bad_version and bad_sync_header) are counted by the core's validation
// pipeline and mirrored here exactly like the original classification
// classes.
func (a *Adapter) Discards() repl.DiscardStats {
	return repl.DiscardStats{
		BadCRC:            a.badCRC.Load(),
		Undecodable:       a.undecodable.Load(),
		BadProtoVersion:   a.badProtoVersion.Load(),
		OwnEcho:           a.ownEcho.Load(),
		DuplicateNodeID:   a.duplicateNodeID.Load(),
		EmptyKey:          a.emptyKey.Load(),
		KeyTooLong:        a.keyTooLong.Load(),
		TombstoneWithData: a.tombstoneData.Load(),
		BadChecksum:       a.badChecksum.Load(),
		BadVersion:        a.badVersion.Load(),
		BadSyncHeader:     a.badSyncHeader.Load(),
	}
}

// KnownPeers returns the known-peer count mirrored from the core's peer
// table after the latest event-loop iteration — safe from any goroutine.
func (a *Adapter) KnownPeers() uint64 {
	return a.peerCount.Load()
}

// UnackedStats returns the pending-set gauge mirrored from the core's
// [repl.Core.UnackedStats] after the latest event-loop iteration (the
// OBS-01 unacked count and oldest-age seam) — safe from any goroutine.
func (a *Adapter) UnackedStats() repl.UnackedStats {
	return repl.UnackedStats{
		Count:     int(a.unackedCount.Load()),
		OldestAge: time.Duration(a.unackedOldestNs.Load()),
	}
}

// MergeStats returns the remote-merge counters mirrored from the core after
// the latest event-loop iteration (the OBS-01 merge-conflicts and
// drift-warning seam) — safe from any goroutine.
func (a *Adapter) MergeStats() repl.MergeStats {
	return repl.MergeStats{
		Conflicts:     a.mergeConflicts.Load(),
		DriftWarnings: a.mergeDriftWarning.Load(),
	}
}

// Retransmissions returns the retransmissions counter mirrored from the
// core after the latest event-loop iteration (the OBS-01 seam) — safe from
// any goroutine.
func (a *Adapter) Retransmissions() uint64 {
	return a.retransmitted.Load()
}

// PacedSends returns the number of datagrams the core's three pacers
// (listing chunks, record sends and egress responses) have handed to the
// Outbox, mirrored after the latest event-loop iteration — emissions, never
// enqueues (T-3-41), safe from any goroutine.
func (a *Adapter) PacedSends() uint64 {
	return a.pacedSends.Load()
}

// SendQueue returns the total datagram-queue depth — listing chunks, record
// sends and egress responses waiting to emit — mirrored from the core after
// the latest event-loop iteration (RESEARCH Pitfall 23's total-queue gauge)
// — safe from any goroutine.
func (a *Adapter) SendQueue() int {
	return int(a.sendQueue.Load())
}

// EgressDropped returns the egress-queue overflow drops mirrored from the
// core after the latest event-loop iteration (XPORT-07's drop-policy seam) —
// safe from any goroutine.
func (a *Adapter) EgressDropped() uint64 {
	return a.egressDropped.Load()
}

// SentByType counts the outbound datagrams this adapter has encoded and
// handed to its socket, by message kind (the OBS-01 replication-topology
// seam) — the vocabulary mirrors the wire Body oneof.
type SentByType struct {
	Hello       uint64
	Data        uint64
	Ack         uint64
	DataRequest uint64
	Sync        uint64
}

// SentByType returns the per-kind send counts — a value copy over the
// atomics flush increments, safe from any goroutine.
func (a *Adapter) SentByType() SentByType {
	return SentByType{
		Hello:       a.sentHello.Load(),
		Data:        a.sentData.Load(),
		Ack:         a.sentAck.Load(),
		DataRequest: a.sentDataRequest.Load(),
		Sync:        a.sentSync.Load(),
	}
}

// SyncArrived reports whether the core has processed a COMPLETE listing —
// every chunk of some peer's round received (the startup gate's sync half,
// API-04, PROTO-01). It reads the one listing mirror pointer the event loop
// publishes, so a true answer implies ListedKey can answer from the same
// snapshot; false covers both "no complete listing yet" and "mirror never
// published". Safe from any goroutine: the pointer is the only state read.
func (a *Adapter) SyncArrived() bool {
	m := a.listing.Load()
	return m != nil && m.complete
}

// ListedKey reports whether the completed listing named key — a key some
// peer's listing carried, whether or not its value has arrived yet. It loads
// the SAME mirror pointer SyncArrived reads and returns false for a missing
// mirror, an incomplete listing, or a key outside the union (PROTO-01,
// 14.5(6)). Reads never wait for the value itself; the RPC layer maps this to
// Unavailable/AlreadyExists. Safe from any goroutine.
func (a *Adapter) ListedKey(key string) bool {
	m := a.listing.Load()
	if m == nil || !m.complete {
		return false
	}
	_, ok := m.keys[key]
	return ok
}

// listingMirror is one immutable publication of the core's listing state:
// whether a complete listing has been processed and the listed-key union at
// that moment. Stored as one pointer so completeness and keys can never be
// observed from different generations (T-4-05).
type listingMirror struct {
	complete bool
	keys     map[string]struct{}
}

// ReadErrors returns the number of receive-path errors the read loop has
// survived (XPORT-08, T-3-33) — the reliable deafness signal, safe from any
// goroutine. A non-zero, growing value means the socket is delivering
// errors; the loop keeps running and the node stays honest about it.
func (a *Adapter) ReadErrors() uint64 {
	return a.readErrors.Load()
}

// newAdapterWithConn returns the bare adapter the receive-path white-box
// tests drive: the socket seam, the log sink and the read loop's two
// channels, with no core and no socket configuration. Production
// construction is [New]; this constructor exists only so a fake [packetConn]
// can prove a non-closed read error is survivable (XPORT-08).
func newAdapterWithConn(conn packetConn) *Adapter {
	return &Adapter{
		conn:    conn,
		log:     func(_ string) {},
		inbound: make(chan inboundEvent, inboundQueue),
		done:    make(chan struct{}),
	}
}

// readLoop is the one read goroutine feeding the event loop's inbound queue
// (D-04): it owns the socket's single 65 507-byte receive buffer for the
// socket's lifetime and defensively copies every read's bytes before
// handing them over, because the buffer is reused on the next read.
//
// XPORT-08's contract is the error branch: ONLY a read error matching
// [net.ErrClosed] exits — the socket closed, which is Run's own exit path.
// Every other error is counted, logged at most once per
// [readErrorLogInterval], paused for [readErrorBackoff] so a persistent
// kernel error cannot spin, and the loop reads the next datagram. A dead
// receive path while Hello broadcasts continue would be a silently deaf
// node; the counter and the log are what make that state visible instead.
func (a *Adapter) readLoop() {
	buf := make([]byte, datagramBufSize)
	for {
		n, src, err := a.conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return // the socket closed — Run's exit path, the only exit.
			}
			a.readErrors.Add(1)
			a.logReadError(err)
			time.Sleep(readErrorBackoff)
			continue
		}
		ev := inboundEvent{datagram: bytes.Clone(buf[:n]), source: src.String()}
		select {
		case a.inbound <- ev:
		case <-a.done:
			return
		}
	}
}

// logReadError surfaces one survived receive error, rate-limited to at most
// one line per [readErrorLogInterval] so an error flood cannot flood the log
// (T-3-36). The wall clock is the sanctioned time source on the adapter
// side; the core's event-time discipline does not apply to this goroutine.
func (a *Adapter) logReadError(err error) {
	a.readErrMu.Lock()
	defer a.readErrMu.Unlock()
	now := time.Now()
	if !a.readErrLogAt.IsZero() && now.Sub(a.readErrLogAt) < readErrorLogInterval {
		return
	}
	a.readErrLogAt = now
	a.log(fmt.Sprintf("udprepl: receive error (read loop continues): %v", err))
}

// flush encodes and sends every outbound message the core emitted for the
// just-handled event: broadcast to the configured broadcast address,
// unicast to the handle resolved through [net.ResolveUDPAddr] — handles are
// opaque to the core; only here do they become addresses (D-03). Each
// encode-successful message counts into its SentByType kind (OBS-01) — the
// datagram is on its way to the socket; an encode failure counts nowhere.
// Failures are logged, never panics: a transient send error must not take
// the node down.
func (a *Adapter) flush(out *port.Outbox) {
	for _, ob := range out.Drain() {
		datagram, err := encodeDatagram(a.nodeID, a.runID, ob.Msg)
		if err != nil {
			a.log(err.Error())
			continue
		}
		a.countSent(ob.Msg)
		if ob.Broadcast {
			a.write(datagram, a.broadcastAddr)
			continue
		}
		addr, err := net.ResolveUDPAddr("udp4", ob.To)
		if err != nil {
			a.log(fmt.Sprintf("udprepl: resolve unicast handle %q: %v", ob.To, err))
			continue
		}
		a.write(datagram, addr)
	}
}

// countSent adds one to the message kind's send counter (OBS-01) — the
// five-kind vocabulary of the wire Body oneof, counted in flush where every
// outbound message crosses the encode step on the event-loop goroutine.
func (a *Adapter) countSent(msg port.Message) {
	switch msg.(type) {
	case port.Hello:
		a.sentHello.Add(1)
	case port.DataReplication:
		a.sentData.Add(1)
	case port.Ack:
		a.sentAck.Add(1)
	case port.DataRequest:
		a.sentDataRequest.Add(1)
	case port.DataSync:
		a.sentSync.Add(1)
	}
}

// write sends one datagram to addr, logging (never panicking on) a failure.
func (a *Adapter) write(datagram []byte, addr *net.UDPAddr) {
	_, err := a.conn.WriteToUDP(datagram, addr)
	if err != nil {
		a.log(fmt.Sprintf("udprepl: send to %s: %v", addr, err))
	}
}

// mirrorStats copies the core's discard counters, known-peer count, the
// listing mirror, the OBS-01 gauge set — the pending-set gauge, the
// remote-merge counters and the retransmissions counter — and the XPORT-07
// emission set (total queue depth, egress drops, paced sends) into the
// atomics after each loop iteration (D-04): cross-goroutine readers — the
// smoke test, later observability — never touch core state. The listing
// mirror is published only when its generation moved, so its clone cost is
// paid at listing cadence, never per event.
func (a *Adapter) mirrorStats() {
	d := a.core.Discards()
	a.badCRC.Store(d.BadCRC)
	a.undecodable.Store(d.Undecodable)
	a.badProtoVersion.Store(d.BadProtoVersion)
	a.ownEcho.Store(d.OwnEcho)
	a.duplicateNodeID.Store(d.DuplicateNodeID)
	a.emptyKey.Store(d.EmptyKey)
	a.keyTooLong.Store(d.KeyTooLong)
	a.tombstoneData.Store(d.TombstoneWithData)
	a.badChecksum.Store(d.BadChecksum)
	a.badVersion.Store(d.BadVersion)
	a.badSyncHeader.Store(d.BadSyncHeader)
	// PeerCount returns the peer-table length, never a negative value.
	//nolint:gosec // G115: PeerCount is len(c.peers), a non-negative map length.
	a.peerCount.Store(uint64(a.core.PeerCount()))
	u := a.core.UnackedStats(time.Now())
	a.unackedCount.Store(int64(u.Count))
	a.unackedOldestNs.Store(int64(u.OldestAge))
	m := a.core.MergeStats()
	a.mergeConflicts.Store(m.Conflicts)
	a.mergeDriftWarning.Store(m.DriftWarnings)
	a.retransmitted.Store(a.core.Retransmissions())
	a.sendQueue.Store(int64(a.core.QueuedSends()))
	a.egressDropped.Store(a.core.EgressDropped())
	a.pacedSends.Store(a.core.PacedSends())
	// The listing mirror is content-gated: the core's generation moves only
	// on an actual {complete, union} change, so an identical listing flood
	// republishes no pointer and clones no map. Completeness and keys are
	// captured into ONE immutable value and published by ONE pointer store —
	// the gate can never open before ListedKey sees the keys (T-4-05).
	if gen := a.core.ListingGeneration(); gen != a.listingGen {
		a.listingGen = gen
		a.listing.Store(&listingMirror{
			complete: a.core.ListingComplete(),
			keys:     a.core.ListedKeysSnapshot(),
		})
	}
}

// armTimer re-arms (or first creates) t for the next deadline and returns
// the timer to hold. Go 1.23+ timer semantics make Reset without draining
// safe (the persist engine's timerChan/stopTimer precedent).
func armTimer(t *time.Timer, next time.Time) *time.Timer {
	d := time.Until(next)
	if t == nil {
		return time.NewTimer(d)
	}
	t.Reset(d)
	return t
}

// timerChan returns t's channel, or nil for a nil timer — a nil channel
// receives never, disabling that select case (the persist engine
// precedent).
func timerChan(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

// stopTimer stops t when non-nil. Go 1.23+ timer semantics make Stop safe
// without draining.
func stopTimer(t *time.Timer) {
	if t != nil {
		t.Stop()
	}
}

// encodeDatagram frames msg as one full wire datagram from nodeID's runID
// (D-01): the port struct maps onto the proto Body oneof, whose marshaled
// bytes travel as the Envelope body under a CRC32 — corruption detection
// only, never tamper protection, per the accepted trust model.
func encodeDatagram(nodeID uint16, runID [16]byte, msg port.Message) ([]byte, error) {
	body, err := protoBody(msg)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("udprepl: marshal body: %w", err)
	}
	env := &replproto.Envelope{
		ProtoVersion: repl.SupportedProtoVersion,
		NodeId:       uint32(nodeID),
		RunId:        runID[:],
		Crc32:        crc32.ChecksumIEEE(bodyBytes),
		Body:         bodyBytes,
	}
	datagram, err := proto.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("udprepl: marshal envelope: %w", err)
	}
	return datagram, nil
}

// decode maps one raw datagram onto the port seam, mechanically per D-02:
// an empty datagram or an envelope that fails to unmarshal classifies
// Undecodable; envelope fields violating their wire domain — a node_id
// beyond the 16-bit range, a run_id that is not 16 bytes — classify
// Undecodable too; a proto_version other than the one this build speaks
// classifies ProtoVersionMismatch with the envelope fields carried over and
// NO body decode — the verdict precedes the CRC gate because a foreign
// version's body need not follow this version's framing contract (D-07); a
// CRC mismatch classifies BadCRC with the envelope fields carried over and
// the body NOT unmarshaled; only a passing CRC is the body decoded, and any
// body fault classifies Undecodable.
func decode(datagram []byte) port.InboundDatagram {
	var env replproto.Envelope
	if len(datagram) == 0 || proto.Unmarshal(datagram, &env) != nil {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	if env.GetNodeId() > maxNodeIDWire || len(env.GetRunId()) != wireIDSize {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	in := port.InboundDatagram{
		ProtoVersion: env.GetProtoVersion(),
		NodeID:       uint16(env.GetNodeId()), //nolint:gosec // G115: the maxNodeIDWire check above proved the value fits 16 bits.
	}
	copy(in.RunID[:], env.GetRunId())
	if in.ProtoVersion != repl.SupportedProtoVersion {
		in.Status = port.ProtoVersionMismatch
		return in
	}
	if crc32.ChecksumIEEE(env.GetBody()) != env.GetCrc32() {
		in.Status = port.BadCRC
		return in
	}
	var body replproto.Body
	err := proto.Unmarshal(env.GetBody(), &body)
	if err != nil {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	msg, ok := portMessage(&body)
	if !ok {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	in.Status = port.Decoded
	in.Msg = msg
	return in
}

// protoBody maps one port message onto its wire Body oneof — constructed
// via the generated wrapper types (RESEARCH Pitfall 6). A standalone
// SyncEntry has no datagram body of its own — it only travels inside
// DataSync — so it is rejected here.
func protoBody(msg port.Message) (*replproto.Body, error) {
	switch m := msg.(type) {
	case port.Hello:
		return &replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}}, nil
	case port.DataReplication:
		return &replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
			Key:      m.Key,
			Version:  m.Record.Version[:],
			Data:     m.Record.Data,
			Deleted:  m.Record.Deleted,
			Checksum: m.Record.Checksum,
		}}}, nil
	case port.Ack:
		return &replproto.Body{Msg: &replproto.Body_Ack{Ack: &replproto.Ack{Key: m.Key, Version: m.Version[:]}}}, nil
	case port.DataRequest:
		return &replproto.Body{Msg: &replproto.Body_DataRequest{DataRequest: &replproto.DataRequest{Key: m.Key}}}, nil
	case port.DataSync:
		entries := make([]*replproto.SyncEntry, len(m.Entries))
		for i, entry := range m.Entries {
			entries[i] = &replproto.SyncEntry{Key: entry.Key, Version: entry.Version[:]}
		}
		return &replproto.Body{Msg: &replproto.Body_DataSync{DataSync: &replproto.DataSync{
			SyncId:     m.SyncID,
			ChunkIndex: m.ChunkIndex,
			ChunkCount: m.ChunkCount,
			Entries:    entries,
		}}}, nil
	default:
		return nil, fmt.Errorf("%w: %T", errNoStandaloneBody, msg)
	}
}

// portMessage maps a decoded wire Body onto its port message struct (D-01),
// reporting false when a field violates its wire domain — a version that is
// not exactly 16 bytes — which the caller classifies Undecodable.
func portMessage(body *replproto.Body) (port.Message, bool) {
	switch msg := body.GetMsg().(type) {
	case *replproto.Body_Hello:
		return port.Hello{}, true
	case *replproto.Body_DataReplication:
		dr := msg.DataReplication
		ver, ok := recordVersion(dr.GetVersion())
		if !ok {
			return nil, false
		}
		return port.DataReplication{
			Key: dr.GetKey(),
			Record: storage.Record{
				Version:  ver,
				Data:     dr.GetData(),
				Deleted:  dr.GetDeleted(),
				Checksum: dr.GetChecksum(),
			},
		}, true
	case *replproto.Body_Ack:
		ver, ok := recordVersion(msg.Ack.GetVersion())
		if !ok {
			return nil, false
		}
		return port.Ack{Key: msg.Ack.GetKey(), Version: ver}, true
	case *replproto.Body_DataRequest:
		return port.DataRequest{Key: msg.DataRequest.GetKey()}, true
	case *replproto.Body_DataSync:
		wire := msg.DataSync.GetEntries()
		entries := make([]port.SyncEntry, len(wire))
		for i, entry := range wire {
			ver, ok := recordVersion(entry.GetVersion())
			if !ok {
				return nil, false
			}
			entries[i] = port.SyncEntry{Key: entry.GetKey(), Version: ver}
		}
		return port.DataSync{
			SyncID:     msg.DataSync.GetSyncId(),
			ChunkIndex: msg.DataSync.GetChunkIndex(),
			ChunkCount: msg.DataSync.GetChunkCount(),
			Entries:    entries,
		}, true
	default:
		return nil, false
	}
}

// recordVersion converts one wire version field into a RecordVersion,
// reporting false unless it is exactly 16 bytes — RFC 9562's UUID size, the
// full record-version domain.
func recordVersion(b []byte) (version.RecordVersion, bool) {
	var ver version.RecordVersion
	if len(b) != len(ver) {
		return version.RecordVersion{}, false
	}
	copy(ver[:], b)
	return ver, true
}
