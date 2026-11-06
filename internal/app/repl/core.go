// Package repl implements the sup-kv replication decision core: a pure,
// synchronous state machine that owns every replication decision — the
// datagram discard classification and its counters (D-02), the peer table
// with opaque address handles (D-03), and the Hello/DataReplication/Ack
// processing of the tracer path. The wall clock enters only through the now
// parameters of the handler methods (the Phase 2 D-02 now-seam precedent);
// the package takes no locks and starts no goroutines — exactly one driver
// goroutine, the production adapter's event loop (D-04) or the test
// simulator, calls the handlers — and a local write reaches the core only
// by its author's explicit NotifyLocalWrite call (D-05). The package
// depends on the standard library and the own module only: protobuf stays
// adapter-side behind the port structs (D-01, the sup-kv-app-no-io depguard
// rule).
package repl

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// SupportedProtoVersion is the wire protocol version this core speaks: the
// envelope version of every datagram it emits, and any inbound datagram
// whose version differs is discarded as BadProtoVersion (REPL-09).
const SupportedProtoVersion = 1

// Design defaults a zero or negative Config field is normalized to, from
// docs/IDEA.md §Parameters.
const (
	defaultHelloInterval      = time.Second
	defaultPeerTimeout        = 5 * time.Second
	defaultAckTimeout         = 200 * time.Millisecond
	defaultAckTimeoutMax      = 5 * time.Second
	defaultRetransmitInterval = 100 * time.Millisecond
	defaultSyncInterval       = 30 * time.Second
	defaultSyncChunkBytes     = 1400
	defaultSyncResponseRate   = 500

	// backoffGrowth is the retransmission ladder's doubling factor: each
	// retry doubles the previous delay from ack_timeout up to the
	// ack_timeout_max saturation cap (REPL-02, docs/IDEA.md §Parameters).
	backoffGrowth = 2
)

// Config holds the replication timing parameters; it is immutable once
// handed to New. Zero or negative fields normalize to the docs/IDEA.md
// §Parameters defaults, and a nil Log normalizes to a no-op: the core is
// total over any Config. RetransmitInterval bounds how long the driver may
// sleep while any pending entry exists (the scan-granularity reading of
// IDEA.md's parameter list, research A5); the per-key backoff ladder itself
// doubles from AckTimeout and saturates at AckTimeoutMax — documented with
// every step at Core.scanRetransmit. SyncResponseRate paces the egress
// queue: responses are spaced floor(1s/SyncResponseRate) nanoseconds apart
// by integer division (probe #16), the first response immediate.
type Config struct {
	HelloInterval      time.Duration
	PeerTimeout        time.Duration
	AckTimeout         time.Duration
	AckTimeoutMax      time.Duration
	RetransmitInterval time.Duration
	SyncInterval       time.Duration
	SyncChunkBytes     int
	SyncResponseRate   int
	Log                func(msg string)
}

// normalized returns cfg with zero or negative fields replaced by the
// design defaults and a nil Log replaced by a no-op.
func (c Config) normalized() Config {
	if c.HelloInterval <= 0 {
		c.HelloInterval = defaultHelloInterval
	}
	if c.PeerTimeout <= 0 {
		c.PeerTimeout = defaultPeerTimeout
	}
	if c.AckTimeout <= 0 {
		c.AckTimeout = defaultAckTimeout
	}
	if c.AckTimeoutMax <= 0 {
		c.AckTimeoutMax = defaultAckTimeoutMax
	}
	if c.RetransmitInterval <= 0 {
		c.RetransmitInterval = defaultRetransmitInterval
	}
	if c.SyncInterval <= 0 {
		c.SyncInterval = defaultSyncInterval
	}
	if c.SyncChunkBytes <= 0 {
		c.SyncChunkBytes = defaultSyncChunkBytes
	}
	if c.SyncResponseRate <= 0 {
		c.SyncResponseRate = defaultSyncResponseRate
	}
	if c.Log == nil {
		c.Log = func(_ string) {}
	}
	return c
}

// StateStore is the record source the replication core consumes: remote
// records join the store through ApplyRemote and the sync paths read the
// full snapshot through Records. The concrete engine (internal/app/persist
// wrapping storage.Store; the ApplyRemote seam lands with plan 04-06) and
// tests substituting storage.Store directly both satisfy it. Declared
// consumer-side here, per the storage.Clock precedent — internal/port holds
// no interfaces.
type StateStore interface {
	// ApplyRemote joins rec under key by the store's LWW merge and reports
	// the displaced versions and the clock drift, exactly like a local
	// write's bookkeeping but triggered by replication.
	ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error)

	// Records returns a full snapshot of the map, tombstones included,
	// with Data defensively copied — the DataSync compare source.
	Records() map[string]storage.Record
}

// NewRunID returns a fresh per-process-start run identifier: 16 random
// bytes from crypto/rand (uniqueness is the only requirement — an impostor
// must not be able to guess it, which is what makes run_id equality the
// own-echo test in HandleDatagram). crypto/rand is depguard-legal inside
// internal/app; per Go 1.24+ the Read never fails, so the error is
// discarded.
func NewRunID() [16]byte {
	var id [16]byte
	_, _ = rand.Read(id[:]) // crypto/rand.Read never returns an error since Go 1.24.
	return id
}

// DiscardStats counts every inbound-datagram discard class (D-02): all
// discard decisions live in the core, so the simulator exercises exactly
// the counters that run in production, and every class stays
// distinguishable for observability (REPL-08, REPL-09).
type DiscardStats struct {
	BadCRC          uint64
	Undecodable     uint64
	BadProtoVersion uint64
	OwnEcho         uint64
	DuplicateNodeID uint64
}

// PeerInfo is one known neighbour as exposed by PeerSnapshot: the opaque
// address handle (D-03), the run_id it announced, and the time of its last
// Hello. The run_id comparison against the stored one is what detects a
// restarted peer (REPL-07).
type PeerInfo struct {
	Handle    string
	RunID     [16]byte
	LastHello time.Time
}

// peer is the core's internal peer-table row, keyed by NodeID (D-03).
type peer struct {
	handle    string
	runID     [16]byte
	lastHello time.Time
}

// queuedUnicast is one reactive unicast response waiting in the egress
// queue (REPL-05's rate limiter): the destination's opaque address handle
// (D-03) and the message. Exactly three producer sites exist — DataRequest
// emissions, DataRequest replies and DataSync-newer responses (the
// egress-limiter scope pin) — and nothing else ever queues: proactive
// traffic (Hello, retransmits, the timer/startup DataSync itself) bypasses
// the limiter, which is what bounds an empty node's recovery instead of
// letting it become a flood (T-04-07).
type queuedUnicast struct {
	handle string
	msg    port.Message
}

// pendingEntry is one local write awaiting peer Acks (REPL-02): a
// defensive copy of the written record (the storage.install precedent — the
// author may reuse its buffer after NotifyLocalWrite returns), the exact
// tracked version, the wait-set of known peers that have not yet Acked that
// version, the retransmission ladder state (next retry time and current
// backoff — the ladder itself fires from Advance, plan 04-02 Task 2), and
// the creation time behind UnackedStats' oldest age.
type pendingEntry struct {
	rec       storage.Record
	version   version.RecordVersion
	waitSet   map[uint16]struct{}
	nextRetry time.Time
	backoff   time.Duration
	createdAt time.Time
}

// Core is the replication decision state machine for one node. Not safe for
// concurrent use: exactly one driver goroutine — the adapter's event loop
// in production, the simulator's scheduler in tests — calls the handlers
// (D-04); the discard counters and the peer-table snapshot are the
// sanctioned cross-goroutine reads, through their copying accessors after
// each event.
type Core struct {
	nodeID    uint16
	runID     [16]byte
	ownHandle string
	store     StateStore
	cfg       Config
	peers     map[uint16]peer
	pending   map[string]pendingEntry
	nextHello time.Time
	egress    []queuedUnicast

	// nextSendAt is the earliest virtual time the next queued response may
	// emit; it is meaningful exactly while the egress queue is non-empty
	// (enqueueResponse arms it, the drain advances it).
	nextSendAt time.Time

	// syncResponseInterval is the egress queue's per-response spacing,
	// derived once at construction by the pinned deterministic integer
	// arithmetic: floor(1s/SyncResponseRate) nanoseconds by integer
	// division — exactly 2ms at the 500/s default and 3333333ns at 300/s
	// (probe #16, echoed in docs/IDEA.md §Parameters).
	syncResponseInterval time.Duration

	discards DiscardStats
}

// New returns a core for nodeID with the per-process runID. ownHandle is
// this node's own address as the adapter sees it (the local socket address
// string) — opaque to the core (D-03), kept for the later own-echo and
// duplicate-node_id classification work. store must be non-nil.
func New(nodeID uint16, runID [16]byte, ownHandle string, store StateStore, cfg Config) *Core {
	cfg = cfg.normalized()
	return &Core{
		nodeID:               nodeID,
		runID:                runID,
		ownHandle:            ownHandle,
		store:                store,
		cfg:                  cfg,
		peers:                make(map[uint16]peer),
		pending:              make(map[string]pendingEntry),
		syncResponseInterval: time.Second / time.Duration(cfg.SyncResponseRate),
	}
}

// HandleDatagram processes one inbound datagram. The classification order
// is pinned (D-02, REPL-08, REPL-09): a BadCRC framing verdict counts
// BadCRC (the adapter never decoded the body); Undecodable counts
// Undecodable; a wrong ProtoVersion counts BadProtoVersion; a datagram
// carrying our own node_id counts OwnEcho when its run_id equals ours (an
// impostor cannot guess a 16-byte random run_id, so run_id equality is the
// robust own-echo test — the sender's own broadcast really does loop back)
// and DuplicateNodeID — a logged configuration error — when it differs.
// Everything else reaches protocol processing: Hello upserts the peer
// table, DataReplication applies the record and Acks it, Ack drains
// the pending-set wait-set (REPL-02), and DataRequest answers through the
// rate-limited egress queue (REPL-04). DataSync arrives with this plan's
// Task 3 and is ignored meanwhile.
func (c *Core) HandleDatagram(now time.Time, in port.InboundDatagram, out *port.Outbox) {
	switch in.Status {
	case port.BadCRC:
		c.discards.BadCRC++
		return
	case port.Undecodable:
		c.discards.Undecodable++
		return
	}
	if in.ProtoVersion != SupportedProtoVersion {
		c.discards.BadProtoVersion++
		return
	}
	if in.NodeID == c.nodeID {
		if in.RunID == c.runID {
			c.discards.OwnEcho++
			return
		}
		c.discards.DuplicateNodeID++
		c.cfg.Log(fmt.Sprintf(
			"sup-kv: configuration error: node_id %d announced by another run from handle %q, datagram discarded (REPL-08)",
			in.NodeID, in.Source,
		))
		return
	}
	c.handleMessage(now, in, out)
}

// handleMessage dispatches one classified datagram's decoded message to its
// protocol handler.
func (c *Core) handleMessage(now time.Time, in port.InboundDatagram, out *port.Outbox) {
	switch msg := in.Msg.(type) {
	case port.Hello:
		c.handleHello(now, in, out)
	case port.DataReplication:
		c.applyRemote(in.Source, msg, out)
	case port.Ack:
		c.handleAck(in.NodeID, msg)
	case port.DataRequest:
		c.handleDataRequest(now, in.Source, msg.Key)
	case port.DataSync, port.SyncEntry, nil:
		// DataSync compare lands with this plan's Task 3; a standalone
		// SyncEntry never travels (it lives inside DataSync), and a nil Msg
		// on a Decoded verdict violates the adapter contract — nothing to
		// do for either.
	}
}

// handleHello upserts the peer-table row from one Hello datagram (REPL-01):
// only a Hello refreshes lastHello — the field's name is the liveness
// contract, and no other datagram type ever touches it. A Hello that ends
// an empty table is discovery: in this same event the core flushes every
// pending entry — one broadcast DataReplication per entry in sorted key
// order, the new peer added to every wait-set, every ladder restarted at
// cfg.AckTimeout — so a brand-new peer receives the pending set promptly,
// not at the next backoff expiry (research A6). A Hello that does not end
// an empty table modifies no wait-set: only discovery adds waiters.
func (c *Core) handleHello(now time.Time, in port.InboundDatagram, out *port.Outbox) {
	discovery := len(c.peers) == 0
	c.peers[in.NodeID] = peer{handle: in.Source, runID: in.RunID, lastHello: now}
	if !discovery {
		return
	}
	for _, key := range slices.Sorted(maps.Keys(c.pending)) {
		entry := c.pending[key]
		entry.waitSet[in.NodeID] = struct{}{}
		entry.backoff = c.cfg.AckTimeout
		entry.nextRetry = now.Add(c.cfg.AckTimeout)
		c.pending[key] = entry
		out.Broadcast(port.DataReplication{Key: key, Record: entry.rec})
	}
}

// applyRemote joins one replicated record through the store seam and Acks
// it immediately after the in-memory apply — nothing here waits for a dump
// (REPL-03). A store error (for example a value over the configured limit)
// is surfaced through Log, never a panic: a hostile or oversized datagram
// must not take the node down, and merge idempotence makes the duplicate
// delivery of the same record a no-op that still Acks (REPL-09's adjacency
// edge).
func (c *Core) applyRemote(source string, msg port.DataReplication, out *port.Outbox) {
	_, _, err := c.store.ApplyRemote(msg.Key, msg.Record)
	if err != nil {
		c.cfg.Log(fmt.Sprintf("sup-kv: applying replicated record for key %q from handle %q: %v", msg.Key, source, err))
	}
	// Remote supersession: a strictly greater remote version under the same
	// key fulfills the tracking promise by winning — the same byte order
	// storage.Merge joins by; an equal version (a redelivery) leaves
	// tracking unchanged (REPL-02).
	if entry, ok := c.pending[msg.Key]; ok && msg.Record.Version.Compare(entry.version) > 0 {
		delete(c.pending, msg.Key)
	}
	out.Unicast(source, port.Ack{Key: msg.Key, Version: msg.Record.Version})
}

// handleAck drains one waiter of the pending entry matching the Ack's
// (key, version) exactly: the sender leaves the entry's wait-set, and the
// entry leaves tracking only when its wait-set empties while at least one
// peer is known — the promise is until-Ack-or-supersession, never a timeout
// (REPL-02). A stale Ack (version mismatch), an Ack for an unknown key and
// an Ack from a peer outside the wait-set are silent no-ops — no counter,
// no log (probe #2). The empty-table guard is load-bearing: an entry whose
// every waiter was evicted stays retained in suspension, and an Ack can
// only drain when peers actually exist. The peer is identified by the
// datagram's NodeID alone; nothing here writes the peer table.
func (c *Core) handleAck(peerID uint16, ack port.Ack) {
	entry, ok := c.pending[ack.Key]
	if !ok || entry.version != ack.Version {
		return
	}
	delete(entry.waitSet, peerID)
	if len(entry.waitSet) == 0 && len(c.peers) > 0 {
		delete(c.pending, ack.Key)
	}
}

// handleDataRequest answers one DataRequest (REPL-04): a single Records()
// snapshot serves the whole handler call — the snapshot discipline every
// sync path of this plan reuses, never one snapshot per response — and a
// key present in the snapshot (tombstones included, so deletes heal
// through the same LWW path as writes) enqueues one unicast
// DataReplication carrying the exact snapshot record on the egress queue,
// which Advance drains at the sync_response_rate cadence. A key absent
// from the snapshot produces nothing at all — no error reply, no nack,
// nothing enqueued, ever. The reply addresses the datagram's source handle
// regardless of peer-table membership: a stranger's DataRequest is still
// protocol traffic from a live node, and replies are unicast only — a
// broadcast reply would amplify one request into N datagrams.
func (c *Core) handleDataRequest(now time.Time, source, key string) {
	rec, ok := c.store.Records()[key]
	if !ok {
		return
	}
	c.enqueueResponse(now, source, port.DataReplication{Key: key, Record: rec})
}

// enqueueResponse appends one reactive unicast to the egress queue
// (REPL-05): the FIFO drains inside Advance at exactly one
// syncResponseInterval per response, enqueue order IS drain order, and the
// first response after an idle queue emits immediately — nextSendAt clamps
// to this call's now, so no head-of-line delay ever applies to the first
// datagram of a burst.
func (c *Core) enqueueResponse(now time.Time, handle string, msg port.Message) {
	if len(c.egress) == 0 {
		c.nextSendAt = now
	}
	c.egress = append(c.egress, queuedUnicast{handle: handle, msg: msg})
}

// drainEgress emits every due queued response (REPL-05): while nextSendAt
// is at or before now, the FIFO's head emits unicast and the next slot
// moves one syncResponseInterval ahead — the spacing uses no map state and
// no re-sorting; the producers iterate sorted listings or arrival order,
// so enqueue order is already deterministic (probe #9 discipline). An
// Advance that arrives late (past several slots) still emits only one
// response per call of this loop's steady state: the slot advances from
// the actual emission time, never bursting a backlog in one tick — the
// limiter is what survives empty-node recovery (T-04-07).
func (c *Core) drainEgress(now time.Time, out *port.Outbox) {
	for len(c.egress) > 0 && !now.Before(c.nextSendAt) {
		q := c.egress[0]
		c.egress = c.egress[1:]
		out.Unicast(q.handle, q.msg)
		if now.After(c.nextSendAt) {
			c.nextSendAt = now
		}
		c.nextSendAt = c.nextSendAt.Add(c.syncResponseInterval)
	}
}

// NotifyLocalWrite is the explicit call a write's author makes after the
// store write (D-05: no store subscriptions, no observers, no callbacks —
// tests now, the gRPC handler in Phase 5). The core keeps a defensive copy
// of rec (the storage.install precedent) as the pending entry — wait-set
// seeded with every currently-known peer, ladder armed at cfg.AckTimeout —
// and broadcasts the record as a DataReplication datagram. A newer local
// write under the same key REPLACES the entry: fresh wait-set, ladder reset
// — local supersession (REPL-02). Replication progress is never a
// precondition of the write path: this enqueues-and-returns (D-05).
func (c *Core) NotifyLocalWrite(now time.Time, key string, rec storage.Record, out *port.Outbox) {
	own := storage.Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Deleted:  rec.Deleted,
		Checksum: rec.Checksum,
	}
	waitSet := make(map[uint16]struct{}, len(c.peers))
	for id := range c.peers {
		waitSet[id] = struct{}{}
	}
	c.pending[key] = pendingEntry{
		rec:       own,
		version:   own.Version,
		waitSet:   waitSet,
		nextRetry: now.Add(c.cfg.AckTimeout),
		backoff:   c.cfg.AckTimeout,
		createdAt: now,
	}
	out.Broadcast(port.DataReplication{Key: key, Record: own})
}

// Advance fires the due timers — the hello timer, the peer-timeout
// eviction scan, the retransmission scan and the rate-limited egress drain
// at this stage — and returns the next deadline for the driver to sleep
// on: the minimum of the following hello deadline, the earliest peer
// expiry, while any pending entry exists the retransmit scan deadline (the
// scan-granularity reading of retransmit_interval, research A5), and while
// any response is queued the next egress slot. The returned deadline is
// always strictly future: work due at now fires inside this same call.
// Eviction runs before the retransmit scan so a peer expiring at now is
// already gone when the scanner asks whether any peer remains; the egress
// drain runs last so a tick's reactive responses follow its proactive
// emissions.
func (c *Core) Advance(now time.Time, out *port.Outbox) time.Time {
	if !now.Before(c.nextHello) {
		out.Broadcast(port.Hello{})
		c.nextHello = now.Add(c.cfg.HelloInterval)
	}
	c.evictPeers(now)
	c.scanRetransmit(now, out)
	c.drainEgress(now, out)
	return c.nextDeadline(now)
}

// evictPeers removes every peer whose liveness expired (REPL-06): the first
// Advance with now at or after lastHello+cfg.PeerTimeout evicts — the
// at-or-after boundary, probe #18 — and eviction removes the peer from the
// table AND from every pending entry's wait-set in the same event. It never
// drains an entry by itself: with the drain rule, an entry whose every
// waiter was evicted is retained in suspension until a future Hello
// re-adds waiters via the flush rule. Only a Hello ever refreshed lastHello
// (handleHello), so this scan is exactly the gone-after-peer_timeout
// reading. An evicted peer is re-learned by its next Hello exactly like a
// first Hello — there is no blacklist state of any kind, and the table
// holds any number of peers (N nodes from day one).
func (c *Core) evictPeers(now time.Time) {
	expired := make([]uint16, 0, len(c.peers))
	for id := range c.peers {
		if !now.Before(c.peers[id].lastHello.Add(c.cfg.PeerTimeout)) {
			expired = append(expired, id)
		}
	}
	for _, id := range expired {
		delete(c.peers, id)
		for key := range c.pending {
			delete(c.pending[key].waitSet, id)
		}
	}
}

// scanRetransmit fires every due pending entry's retransmission (REPL-02).
// An entry is due when now is at or after its nextRetry AND the peer table
// is non-empty; while the table is empty the scanner is suspended — it
// fires nothing and advances no nextRetry or backoff, so entries and their
// ladder state are retained (suspending, not dropping; the first Hello that
// ends an empty table flushes them, see handleHello).
//
// The backoff ladder, with the design defaults of docs/IDEA.md §Parameters:
// after each firing, backoff doubles from the initial cfg.AckTimeout and
// saturates at cfg.AckTimeoutMax — saturate, never wrap (the clock's
// counter-overflow saturation precedent, clock.go). The ladder delays from
// a write at t0 are therefore exactly 200ms, 400ms, 800ms, 1600ms, 3200ms,
// then 5000ms repeated forever — cumulative retries at t0+200, +600,
// +1400, +3000, +6200ms and every 5000ms after that, until the entry drains
// on full Ack coverage or leaves on supersession (REPL-02's "until").
// Firing is one broadcast DataReplication per entry: every peer shares each
// send, and peers that already Acked re-merge idempotently and re-Ack (the
// always-Ack-on-duplicate truth of plan 04-01). retransmit_interval is the
// scan granularity — while any entry is pending, Advance's returned
// deadline is never more than cfg.RetransmitInterval past now, so a due
// retry is honored at scan granularity (research A5); with no pending
// entries this scanner contributes no deadline. Emission order is keys
// sorted ascending: no map-iteration order ever reaches an emit path
// (REPL-02, research Pitfall 1 / probe #9).
func (c *Core) scanRetransmit(now time.Time, out *port.Outbox) {
	if len(c.peers) == 0 || len(c.pending) == 0 {
		return
	}
	for _, key := range slices.Sorted(maps.Keys(c.pending)) {
		entry := c.pending[key]
		if now.Before(entry.nextRetry) {
			continue
		}
		out.Broadcast(port.DataReplication{Key: key, Record: entry.rec})
		entry.backoff = min(backoffGrowth*entry.backoff, c.cfg.AckTimeoutMax)
		entry.nextRetry = now.Add(entry.backoff)
		c.pending[key] = entry
	}
}

// nextDeadline returns the earliest strictly-future deadline over the armed
// timers: the hello timer, the earliest peer expiry, while any pending
// entry exists the retransmit scan deadline, and while any response is
// queued the next egress slot. After evictPeers, scanRetransmit and
// drainEgress have fired everything due, every contribution is strictly
// future.
func (c *Core) nextDeadline(now time.Time) time.Time {
	deadline := c.nextHello
	if len(c.pending) > 0 {
		if scan := c.retransmitDeadline(now); scan.Before(deadline) {
			deadline = scan
		}
	}
	if expiry := c.earliestPeerExpiry(); !expiry.IsZero() && expiry.Before(deadline) {
		deadline = expiry
	}
	if len(c.egress) > 0 && c.nextSendAt.Before(deadline) {
		deadline = c.nextSendAt
	}
	return deadline
}

// earliestPeerExpiry returns the earliest lastHello+cfg.PeerTimeout over
// the known peers — the eviction scan's deadline contribution — and the
// zero time when the table is empty (never a deadline).
func (c *Core) earliestPeerExpiry() time.Time {
	var earliest time.Time
	for id := range c.peers {
		expiry := c.peers[id].lastHello.Add(c.cfg.PeerTimeout)
		if earliest.IsZero() || expiry.Before(earliest) {
			earliest = expiry
		}
	}
	return earliest
}

// retransmitDeadline returns the retransmit scan's deadline contribution:
// the granularity ceiling now+cfg.RetransmitInterval (research A5 — while
// any entry is pending, the driver never sleeps longer than this), clamped
// by the earliest pending nextRetry when the scanner is armed. During
// suspension (an empty peer table) the ladder is paused — its stale retry
// times must not leak into the deadline — so the ceiling alone bounds the
// sleep.
func (c *Core) retransmitDeadline(now time.Time) time.Time {
	ceiling := now.Add(c.cfg.RetransmitInterval)
	if len(c.peers) == 0 {
		return ceiling
	}
	if earliest := c.earliestRetry(); earliest.Before(ceiling) {
		return earliest
	}
	return ceiling
}

// earliestRetry returns the earliest pending nextRetry (the zero time when
// nothing is pending); callers reach it only with a non-empty pending set.
func (c *Core) earliestRetry() time.Time {
	var earliest time.Time
	for key := range c.pending {
		nextRetry := c.pending[key].nextRetry
		if earliest.IsZero() || nextRetry.Before(earliest) {
			earliest = nextRetry
		}
	}
	return earliest
}

// Discards returns the discard-class counters (D-02) — a value copy, safe
// to read from any goroutine the driver permits.
func (c *Core) Discards() DiscardStats {
	return c.discards
}

// PeerSnapshot returns a defensive copy of the peer table keyed by NodeID
// (the storage.Records copying precedent) — the peer-table read seam
// consumed by the later Phase 4 plans (retransmission fan-out, peer
// timeouts, sync targets) and observability, without exposing core state.
func (c *Core) PeerSnapshot() map[uint16]PeerInfo {
	out := make(map[uint16]PeerInfo, len(c.peers))
	for id, p := range c.peers {
		out[id] = PeerInfo{Handle: p.handle, RunID: p.runID, LastHello: p.lastHello}
	}
	return out
}

// PendingInfo is one pending entry as exposed by PendingSnapshot: the exact
// tracked version, the known peers that have not yet Acked it (sorted
// ascending) and the retransmission ladder's next retry time.
type PendingInfo struct {
	Version   version.RecordVersion
	WaitingOn []uint16
	NextRetry time.Time
}

// PendingSnapshot returns a copy of the pending set keyed by key — the
// observability seam Phase 5's wait_for_replication and Phase 6's OBS-01
// consume. Keys are iterated sorted and every WaitingOn is sorted ascending:
// no map-iteration order reaches a read path either (probe #9 discipline).
func (c *Core) PendingSnapshot() map[string]PendingInfo {
	out := make(map[string]PendingInfo, len(c.pending))
	for _, key := range slices.Sorted(maps.Keys(c.pending)) {
		entry := c.pending[key]
		out[key] = PendingInfo{
			Version:   entry.version,
			WaitingOn: slices.Sorted(maps.Keys(entry.waitSet)),
			NextRetry: entry.nextRetry,
		}
	}
	return out
}

// UnackedStats is the pending-set gauge: the unacked entry count and the age
// of the oldest entry — the Phase 6 OBS-01 observability seam.
type UnackedStats struct {
	Count     int
	OldestAge time.Duration
}

// QueuedResponses returns the number of reactive unicast responses waiting
// in the egress queue (REPL-05's rate limiter) — the simulator and
// observability seam plan 04-05's quiescence check reads: a node whose
// queue is empty has finished answering its sync round.
func (c *Core) QueuedResponses() int {
	return len(c.egress)
}

// UnackedStats returns the pending-set gauge observed at now: the unacked
// entry count and, with at least one entry, now minus the oldest entry's
// creation time. A pure read; the wall clock enters through the now
// parameter (the D-02 seam — the core never reads a clock itself).
func (c *Core) UnackedStats(now time.Time) UnackedStats {
	stats := UnackedStats{Count: len(c.pending)}
	if stats.Count == 0 {
		return stats
	}
	var oldest time.Time
	for key := range c.pending {
		createdAt := c.pending[key].createdAt
		if oldest.IsZero() || createdAt.Before(oldest) {
			oldest = createdAt
		}
	}
	stats.OldestAge = now.Sub(oldest)
	return stats
}
