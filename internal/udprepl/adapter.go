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
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
)

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

	// peerIDs is the peer-id mirror for the write_time capture (D-01, A-01):
	// the core's known peer ids, sorted ascending, published as ONE immutable
	// slice pointer so a handler reading it can never observe a half-built
	// set. The event loop stores a fresh pointer only when the core's peer
	// generation moved — an unchanged peer table republishes nothing and
	// clones no slice (the listing mirror's generation-gated precedent). The
	// gRPC handler captures this slice immediately before its engine write,
	// and the SAME copy feeds the replication notification and the wait
	// report, so the reported total is the acceptance-time set.
	peerIDs atomic.Pointer[[]uint16]
	peerGen uint64

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

// SocketBuffers returns the kernel's effective SO_RCVBUF/SO_SNDBUF sizes read
// back during [New] — the doubled, clamped values, never the requested
// constants (RESEARCH Pattern 7). The fields are written once at construction
// and immutable afterwards, so the accessor is safe from any goroutine.
func (a *Adapter) SocketBuffers() (recv, send int) {
	return a.recvBuf, a.sendBuf
}
