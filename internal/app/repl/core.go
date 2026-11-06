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
)

// Config holds the replication timing parameters; it is immutable once
// handed to New. Zero or negative fields normalize to the docs/IDEA.md
// §Parameters defaults, and a nil Log normalizes to a no-op: the core is
// total over any Config. RetransmitInterval bounds how long the driver may
// sleep while any pending entry exists (the scan-granularity reading of
// IDEA.md's parameter list); the per-key backoff ladder itself doubles from
// AckTimeout and saturates at AckTimeoutMax — documented with the ladder in
// plan 04-02, where retransmission lands.
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
	discards  DiscardStats
}

// New returns a core for nodeID with the per-process runID. ownHandle is
// this node's own address as the adapter sees it (the local socket address
// string) — opaque to the core (D-03), kept for the later own-echo and
// duplicate-node_id classification work. store must be non-nil.
func New(nodeID uint16, runID [16]byte, ownHandle string, store StateStore, cfg Config) *Core {
	return &Core{
		nodeID:    nodeID,
		runID:     runID,
		ownHandle: ownHandle,
		store:     store,
		cfg:       cfg.normalized(),
		peers:     make(map[uint16]peer),
		pending:   make(map[string]pendingEntry),
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
// table, DataReplication applies the record and Acks it, and Ack drains
// the pending-set wait-set (REPL-02). DataRequest and DataSync arrive with
// their later plans and are ignored meanwhile.
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
		c.peers[in.NodeID] = peer{handle: in.Source, runID: in.RunID, lastHello: now}
	case port.DataReplication:
		c.applyRemote(in.Source, msg, out)
	case port.Ack:
		c.handleAck(in.NodeID, msg)
	case port.DataRequest, port.DataSync, port.SyncEntry, nil:
		// DataRequest/DataSync processing lands with plans 04-03/04-04; a
		// nil Msg on a Decoded verdict violates the adapter contract and
		// matches no case — either way there is nothing to do yet.
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

// Advance fires the due timers — the hello timer at this stage — and
// returns the next deadline for the driver to sleep on. The zero initial
// hello deadline makes the very first Advance emit the startup Hello
// (docs/IDEA.md §Node lifecycle); every later one repeats every
// cfg.HelloInterval.
func (c *Core) Advance(now time.Time, out *port.Outbox) time.Time {
	if !now.Before(c.nextHello) {
		out.Broadcast(port.Hello{})
		c.nextHello = now.Add(c.cfg.HelloInterval)
	}
	return c.nextHello
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
