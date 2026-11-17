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

// The discard class tokens passed to [Config.OnDiscard] (D-08): stable
// first-class strings — one per DiscardStats counter — that the
// observability surfaces (structured log attrs, later the metrics labels)
// key on. Metadata only: a token and its detail never carry record values,
// user data or key names (the T-6-05 attr-surface contract).
const (
	DiscardBadCRC          = "bad_crc"
	DiscardUndecodable     = "undecodable"
	DiscardBadProtoVersion = "bad_proto_version"
	DiscardOwnBroadcast    = "own_broadcast"
	DiscardDuplicateNodeID = "duplicate_node_id"
)

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

	// maxSyncResponseRate is the top of SyncResponseRate's valid domain.
	// At any rate above it, one second divided by the rate as a
	// [time.Duration] truncates to a zero interval and silently disables
	// the egress limiter (D-15 P4-WR-02), so normalized() clamps every
	// out-of-domain rate — zero, negative or above this cap — to
	// defaultSyncResponseRate: the floor(1s/rate) spacing stays non-zero
	// at every configuration, and the limiter is never silently off.
	maxSyncResponseRate = 1e9

	// backoffGrowth is the retransmission ladder's doubling factor: each
	// retry doubles the previous delay from ack_timeout up to the
	// ack_timeout_max saturation cap (REPL-02, docs/IDEA.md §Parameters).
	backoffGrowth = 2

	// entryOverheadBytes is the conservative per-entry wire-size estimate
	// the core adds to len(key) when budgeting DataSync chunks. The real
	// proto cost of one SyncEntry inside a DataSync — the repeated-field
	// tag and length varint around the entry, the key field's tag and
	// length varint, and the version field's tag, length varint and 16
	// payload bytes — is len(key)+20 inside the entry message proper and at
	// most len(key)+23 with its outer framing, for every key that can share
	// a chunk (a key whose estimate alone reaches the budget ships as an
	// oversized single-entry chunk where the budget does not apply), so the
	// estimate over-counts by at least 1 per entry. The core cannot call
	// proto.Size — protobuf stays adapter-side (D-01) — so the budget is
	// enforced by estimate, and the udprepl cross-seam test
	// (TestSync_chunkWireBudget) keeps this constant honest against the
	// real codec.
	entryOverheadBytes = 24

	// chunkOverheadBytes is the conservative fixed per-chunk estimate: the
	// envelope fields (proto_version, node_id, run_id, crc32 and the body's
	// tag plus length varint), the Body oneof wrapper and the DataSync
	// repeated-field framing — roughly 45-50 bytes on the real wire. The
	// stated guarantee of the two constants together: the real encoded
	// datagram for any budgeted chunk is at most SyncChunkBytes.
	chunkOverheadBytes = 64
)

// Config holds the replication timing parameters; it is immutable once
// handed to New. Zero or negative fields normalize to the docs/IDEA.md
// §Parameters defaults, and a nil Log or OnDiscard normalizes to a no-op:
// the core is total over any Config. RetransmitInterval bounds how long the
// driver may sleep while any pending entry exists (the scan-granularity
// reading of IDEA.md's parameter list, research A5); the per-key backoff
// ladder itself doubles from AckTimeout and saturates at AckTimeoutMax —
// documented with every step at Core.scanRetransmit. SyncResponseRate paces
// the egress queue: responses are spaced floor(1s/SyncResponseRate)
// nanoseconds apart by integer division (probe #16), the first response
// immediate. Rates outside the domain (0, 1e9] clamp to the 500/s default —
// above 1e9 the integer division truncates to a zero interval and would
// silently disable the limiter (D-15 P4-WR-02, the maxSyncResponseRate cap).
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

	// OnDiscard is invoked at every HandleDatagram discard site with the
	// class token (DiscardBadCRC, ...) and a short metadata detail — the
	// source handle, the offending node id (D-08). Additive to Log: the
	// duplicate-node_id configuration errors keep their Log call, and the
	// core itself stays silent (D-09) — this is a plain callback the
	// adapter or main side may bridge into structured logging or counters;
	// the class is a first-class value, never a message prefix to parse.
	OnDiscard func(class, detail string)
}

// normalized returns cfg with out-of-domain fields replaced by the design
// defaults — zero or negative everywhere, and SyncResponseRate additionally
// above maxSyncResponseRate, where the spacing division would truncate to
// zero (D-15 P4-WR-02) — and a nil Log or OnDiscard replaced by a no-op.
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
	if c.SyncResponseRate <= 0 || c.SyncResponseRate > maxSyncResponseRate {
		c.SyncResponseRate = defaultSyncResponseRate
	}
	if c.Log == nil {
		c.Log = func(_ string) {}
	}
	if c.OnDiscard == nil {
		c.OnDiscard = func(_, _ string) {}
	}
	return c
}

// StateStore is the record source the replication core consumes: remote
// records join the store through ApplyRemote and the sync paths read the
// full snapshot through Records. The concrete engine (internal/app/persist
// wrapping storage.Store — its ApplyRemote is the remote-apply seam plan
// 04-06 shipped) and tests adapting storage.Store through a Merge-delegating
// wrapper both satisfy it. Declared
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

// MergeStats counts the remote-merge observability signals (OBS-01, D-10):
// Conflicts is every displaced-or-rejected version the store reported — an
// LWW loser is an ordinary overwrite, counted and never journaled — and
// DriftWarnings is every apply whose drift signal was nonzero: the store
// clock gates the signal at its drift-warn threshold (zero drift is within
// tolerance, any nonzero return is beyond it), so the counter mirrors
// exactly the warnings storage.Merge already decided.
type MergeStats struct {
	Conflicts     uint64
	DriftWarnings uint64
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
// version, the set of peers that have (sorted ascending, idempotent —
// handleAck inserts; eviction never touches it, which is what keeps an
// Ack distinguishable from a peer_timeout eviction in PendingSnapshot,
// T-05-04), the retransmission ladder state (next retry time and current
// backoff — the ladder itself fires from Advance, plan 04-02 Task 2), and
// the creation time behind UnackedStats' oldest age.
type pendingEntry struct {
	rec       storage.Record
	version   version.RecordVersion
	waitSet   map[uint16]struct{}
	acked     []uint16
	nextRetry time.Time
	backoff   time.Duration
	createdAt time.Time
}

// Core is the replication decision state machine for one node. Not safe for
// concurrent use: exactly one driver goroutine — the adapter's event loop
// in production, the simulator's scheduler in tests — calls the handlers
// (D-04); the copying accessors (Discards, SyncArrived, PeerSnapshot,
// PendingSnapshot) are the sanctioned cross-goroutine read seams, consumed
// through the production adapter's post-event atomics. Results, never
// callbacks: no handler takes a callback, a channel or a goroutine.
type Core struct {
	nodeID    uint16
	runID     [16]byte
	ownHandle string
	store     StateStore
	cfg       Config
	peers     map[uint16]peer
	pending   map[string]pendingEntry
	nextHello time.Time

	// nextSync is the next sync-timer deadline; the zero time until the
	// first Advance, whose startup trigger fires the initial full listing
	// alongside the startup Hello (IDEA §Node lifecycle).
	nextSync time.Time

	egress []queuedUnicast

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

	// mergeStats counts the remote-merge signals (OBS-01); retransmitted
	// counts the ladder's retransmission emissions — driver-goroutine-only
	// mutations behind the copying accessors, the Discards precedent.
	mergeStats    MergeStats
	retransmitted uint64

	// syncArrived flips true on the first received DataSync datagram and
	// never resets — the startup gate's sync half (API-04). Set by
	// handleDataSync on the driver goroutine; read through the copying
	// SyncArrived accessor, which the production adapter mirrors into an
	// atomic after every loop iteration so cross-goroutine readers never
	// touch core state.
	syncArrived bool
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
// DuplicateNodeID has a second variant with the same evidence shape, two
// sources claiming one identity: a foreign node_id the peer table already
// remembers whose remembered run_id repeats verbatim from a DIFFERENT
// source handle — a run's identity never migrates handles, so the second
// claim is a misconfiguration, dropped and logged with the node_id and
// both conflicting handles (REPL-08). The classifier never pre-empts the
// restart branch (REPL-07, plan 04-04): this second variant requires run_id
// EQUALITY with the remembered row, so a known node whose announced run_id
// differs always reaches the Hello handler and fires its targeted-listing
// restart reaction — a restart is never counted as a duplicate. Every
// discard site additionally fires cfg.OnDiscard with its class token and a
// metadata detail (D-08) — additive to the duplicate-node_id Log calls,
// never their replacement. Everything else reaches protocol processing:
// Hello upserts the peer
// table and — when a known node's run_id changed — answers with the
// targeted full listing toward the restarted peer (REPL-07),
// DataReplication applies the record and Acks it, Ack drains
// the pending-set wait-set (REPL-02), DataRequest answers through the
// rate-limited egress queue (REPL-04), and DataSync compares against the
// store snapshot and queues DataRequest / unicast DataReplication
// responses on the same queue (REPL-05).
func (c *Core) HandleDatagram(now time.Time, in port.InboundDatagram, out *port.Outbox) {
	switch in.Status {
	case port.BadCRC:
		c.discards.BadCRC++
		c.cfg.OnDiscard(DiscardBadCRC, in.Source)
		return
	case port.Undecodable:
		c.discards.Undecodable++
		c.cfg.OnDiscard(DiscardUndecodable, in.Source)
		return
	}
	if in.ProtoVersion != SupportedProtoVersion {
		c.discards.BadProtoVersion++
		c.cfg.OnDiscard(DiscardBadProtoVersion, in.Source)
		return
	}
	if in.NodeID == c.nodeID {
		if in.RunID == c.runID {
			c.discards.OwnEcho++
			c.cfg.OnDiscard(DiscardOwnBroadcast, in.Source)
			return
		}
		c.discards.DuplicateNodeID++
		c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf("node_id %d at handle %q", in.NodeID, in.Source))
		c.cfg.Log(fmt.Sprintf(
			"sup-kv: configuration error: node_id %d announced by another run from handle %q, datagram discarded (REPL-08)",
			in.NodeID, in.Source,
		))
		return
	}
	if known, ok := c.peers[in.NodeID]; ok && in.RunID == known.runID && in.Source != known.handle {
		c.discards.DuplicateNodeID++
		c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf("node_id %d handles %q/%q", in.NodeID, known.handle, in.Source))
		c.cfg.Log(fmt.Sprintf(
			"sup-kv: configuration error: node_id %d known at handle %q announced again from handle %q, datagram discarded (REPL-08)",
			in.NodeID, known.handle, in.Source,
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
	case port.DataSync:
		c.handleDataSync(now, in.Source, msg.Entries)
	case port.SyncEntry, nil:
		// A standalone SyncEntry never travels (it lives inside DataSync),
		// and a nil Msg on a Decoded verdict violates the adapter contract
		// — nothing to do for either.
	}
}

// handleHello upserts the peer-table row from one Hello datagram (REPL-01)
// and owns the restart discrimination (REPL-07, probe #19). Three cases,
// pinned: an unknown node_id is learned — entry created, no targeted listing
// (the peer's own startup DataSync and the periodic rounds reconcile it); a
// known node_id repeating its remembered run_id refreshes lastHello only,
// never a second listing (a per-Hello full listing at hello_interval cadence
// would be a self-inflicted listing flood; the flap case is bounded by this
// idempotence, not by rate limiting); a known node_id whose run_id differs
// from the remembered one is a detected restart — the duplicate-node_id
// classifier never pre-empts this branch (HandleDatagram's second variant
// requires run_id equality with the remembered row) — the entry updates in place
// (handle, runID, lastHello; the table stays keyed by node_id, one row per
// node — a change never forks a second row, which would fork ack tracking
// and liveness) and the full current DataSync listing goes out UNICAST to
// the peer's handle, every chunk in chunk order through the same plan-04-03
// listing/chunking machinery the sync timer uses. Inequality with the
// remembered run_id is the only trigger: run_ids are 16 random bytes per
// process start, so they carry no temporal order — a stale run_id returning
// after a newer one was seen is indistinguishable from a fresh restart and
// is handled as just another change (entry update plus one idempotent
// re-sync; reconciliation no-ops on equal state). In-flight protocol
// datagrams from the old run — Ack, DataReplication, DataSync — process
// normally on their (key, version) merits, run_id-agnostic outside this
// handler. The restart listing is proactive traffic and does NOT pass
// through the sync_response_rate egress queue (the plan-04-03 egress-scope
// pin): it is bounded by chunk framing and fires once per detected change,
// while the heavy replay traffic it solicits IS limiter-gated — that flows
// as DataRequest replies on the restarted node's side and newer-responses
// on ours. A detected restart never re-adds the peer to any pending
// wait-set and never touches the backoff ladder (the REPL-02 interplay):
// the restarted node heals through reconciliation via the targeted listing,
// not through retransmission, and an entry that already drained stays
// drained — the dump/restart path, not retransmission, is the durability
// story. Only a Hello refreshes lastHello — the field's name is the
// liveness contract, and no other datagram type ever touches it. A Hello
// that ends an empty table is discovery: in this same event the core
// flushes every pending entry — one broadcast DataReplication per entry in
// sorted key order, the new peer added to every wait-set, every ladder
// restarted at cfg.AckTimeout — so a brand-new peer receives the pending
// set promptly, not at the next backoff expiry (research A6). A Hello that
// does not end an empty table modifies no wait-set: only discovery adds
// waiters. Discovery and restart are mutually exclusive by construction —
// a restart requires a remembered row, and an empty table remembers
// nothing (an evicted peer's return Hello, whatever its run_id, is the
// unknown-node case).
func (c *Core) handleHello(now time.Time, in port.InboundDatagram, out *port.Outbox) {
	discovery := len(c.peers) == 0
	restarted := false
	if known, ok := c.peers[in.NodeID]; ok && known.runID != in.RunID {
		restarted = true
	}
	c.peers[in.NodeID] = peer{handle: in.Source, runID: in.RunID, lastHello: now}
	if restarted {
		for _, chunk := range c.listingChunks() {
			out.Unicast(in.Source, port.DataSync{Entries: chunk})
		}
		return
	}
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
// must not take the node down, and a record the store refused is never
// confirmed — no Ack and no supersession of pending tracking, so the sender
// keeps retransmitting and the unacked gauge keeps climbing: the truthful
// signal of divergence, never a "delivered" report for a record that was
// never stored. Merge idempotence keeps the error-free duplicate delivery
// of an already-stored record a no-op that still Acks (REPL-09's adjacency
// edge). The ApplyRemote returns feed the MergeStats counters (OBS-01,
// D-10): every displaced-or-rejected version counts a conflict — counted
// only, never journaled — and a nonzero drift return (the store clock's
// beyond-threshold signal) counts a drift warning.
func (c *Core) applyRemote(source string, msg port.DataReplication, out *port.Outbox) {
	conflicts, drift, err := c.store.ApplyRemote(msg.Key, msg.Record)
	if err != nil {
		c.cfg.Log(fmt.Sprintf("sup-kv: applying replicated record for key %q from handle %q: %v", msg.Key, source, err))
		// Not applied: no Ack, no supersession — never confirm a record the
		// store refused.
		return
	}
	c.mergeStats.Conflicts += uint64(len(conflicts))
	if drift != 0 {
		c.mergeStats.DriftWarnings++
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
// (key, version) exactly: the sender leaves the entry's wait-set, the
// sender joins the entry's acked set (sorted, idempotent — a re-Ack adds no
// second entry), and the entry leaves tracking only when its wait-set
// empties while at least one peer is known — the promise is
// until-Ack-or-supersession, never a timeout (REPL-02). The acked set is
// the honesty half of the Phase 5 wait_for_replication report (T-05-04):
// WaitingOn alone cannot distinguish "peer Acked" from "peer evicted by
// peer_timeout" — both shrink the wait-set — while a peer recorded in
// acked is an Ack by construction, because evictPeers never touches acked.
// A stale Ack (version mismatch) and an Ack for an unknown key are silent
// no-ops — no counter, no log (probe #2). Any OTHER node_id's matching Ack
// joins the acked set whether or not the sender was in the wait-set (the
// counting behavior — WR-01): the wait-set gate does not exist, so a peer
// may appear in acked without ever having been waited on, and the acked
// length can exceed the write-time peer snapshot; the D-08 report keeps
// its peer set fixed at write time (D-10) by reading the count clamped to
// the write-time total (grpcapi's waitForDrain). The empty-table guard is
// load-bearing: an entry whose every waiter was evicted stays retained in
// suspension, and an Ack can only drain when peers actually exist. The
// peer is identified by the datagram's NodeID alone; nothing here writes
// the peer table.
func (c *Core) handleAck(peerID uint16, ack port.Ack) {
	entry, ok := c.pending[ack.Key]
	if !ok || entry.version != ack.Version {
		return
	}
	delete(entry.waitSet, peerID)
	entry.acked = ackedInsert(entry.acked, peerID)
	if len(entry.waitSet) == 0 && len(c.peers) > 0 {
		delete(c.pending, ack.Key)
		return
	}
	c.pending[ack.Key] = entry
}

// ackedInsert returns acked with id inserted at its sorted position — the
// WaitingOn sorted discipline (probe #9: no map-iteration order reaches a
// read path) — or unchanged when already present, so a re-Ack stays
// idempotent.
func ackedInsert(acked []uint16, id uint16) []uint16 {
	pos, found := slices.BinarySearch(acked, id)
	if found {
		return acked
	}
	return slices.Insert(acked, pos, id)
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

// handleDataSync compares one received DataSync listing against the local
// store (REPL-05): a single Records() snapshot serves the whole handler
// call, and each listing entry compares per key by the full record-version
// order — version.Compare, the same order storage.Merge joins by. The
// first received DataSync datagram — any chunk, an empty listing included —
// sets the core's syncArrived flag (the startup gate's sync half, API-04,
// the A1 reading: the protocol deliberately carries no chunk-completeness
// state because chunks are self-sufficient, so the first received chunk is
// the honest first-sync-round signal and the startup_gate timeout is the
// backstop). A locally absent key or a strictly newer listing version
// enqueues a unicast DataRequest to the sender's handle; a strictly newer
// local record — tombstones included, so a local delete heals the sender's
// live copy through the reply — enqueues a unicast exact-record
// DataReplication; equal versions produce silence. Entries are processed
// in received order (senders emit sorted, but the compare is per key and
// order-independent by construction) and every response flows through the
// single rate-limited egress queue — the scope pin; no reactive unicast
// has a bypass path, which is what survives empty-node recovery (T-04-07).
func (c *Core) handleDataSync(now time.Time, source string, entries []port.SyncEntry) {
	c.syncArrived = true
	snapshot := c.store.Records()
	for _, entry := range entries {
		local, present := snapshot[entry.Key]
		cmp := local.Version.Compare(entry.Version)
		switch {
		case !present || cmp < 0:
			c.enqueueResponse(now, source, port.DataRequest{Key: entry.Key})
		case cmp > 0:
			c.enqueueResponse(now, source, port.DataReplication{Key: entry.Key, Record: local})
		}
	}
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

// Advance fires the due timers — the hello timer, the DataSync startup and
// interval triggers, the peer-timeout eviction scan, the retransmission
// scan and the rate-limited egress drain at this stage — and returns the
// next deadline for the driver to sleep on: the minimum of the following
// hello deadline, the next sync deadline, the earliest peer expiry, while
// any pending entry exists the retransmit scan deadline (the
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
	c.syncListing(now, out)
	c.evictPeers(now)
	c.scanRetransmit(now, out)
	c.drainEgress(now, out)
	return c.nextDeadline(now)
}

// syncListing fires the DataSync full-listing emission (REPL-05): the
// startup trigger on the first Advance — immediately after the startup
// Hello, Hello first, both broadcast, the IDEA §Node lifecycle order
// pinned by TestSync_triggers — and the timer trigger on every subsequent
// Advance at or after lastSync+cfg.SyncInterval. The listing is one
// Records() snapshot turned into entries for every record, tombstones
// included (deletes reconcile through the same listing), sorted by key in
// byte order and chunked by the conservative estimate; one broadcast
// datagram per chunk leaves in chunk order. A store with zero records
// still emits one DataSync with an empty entry list — an empty listing is
// a valid sync round that invites newer-flooding from receivers (probe
// #7). The neighbor-restart trigger — a Hello carrying a new run_id from
// a known node — emits the same chunk sequence UNICAST toward the
// restarted peer in handleHello (REPL-07, plan 04-04), through the shared
// listingChunks build.
func (c *Core) syncListing(now time.Time, out *port.Outbox) {
	if !c.nextSync.IsZero() && now.Before(c.nextSync) {
		return
	}
	c.nextSync = now.Add(c.cfg.SyncInterval)
	for _, chunk := range c.listingChunks() {
		out.Broadcast(port.DataSync{Entries: chunk})
	}
}

// listingChunks builds the full DataSync listing chunk sequence from one
// Records() snapshot (REPL-05): every record, tombstones included (deletes
// reconcile through the same listing), sorted by key in byte order and
// packed by the conservative-estimate chunker. One build serves both
// emission triggers — the sync timer's broadcast (syncListing) and the
// neighbor-restart unicast (handleHello, REPL-07) — so the restart
// reaction reuses the plan-04-03 machinery verbatim, never a second
// listing implementation. Identical store state produces the identical
// chunk sequence: the walk is over the sorted listing, never a map (probe
// #11), and an empty snapshot yields one empty chunk (probe #7).
func (c *Core) listingChunks() [][]port.SyncEntry {
	snapshot := c.store.Records()
	entries := make([]port.SyncEntry, 0, len(snapshot))
	for _, key := range slices.Sorted(maps.Keys(snapshot)) {
		entries = append(entries, port.SyncEntry{Key: key, Version: snapshot[key].Version})
	}
	return chunkSyncEntries(entries, c.cfg.SyncChunkBytes)
}

// chunkSyncEntries packs the sorted listing into self-sufficient chunks
// (REPL-05, the A4 reading): the walk accumulates entries while the
// conservative estimate usedEstimate + entryEstimate(next) stays at most
// budget — a chunk filled to exactly the budget closes (probe #15), the
// entry that would cross it opens the next chunk, and a single entry
// whose own estimate already exceeds the budget ships alone as its own
// oversized chunk — never dropped, never split, and carrying no
// continuation state (no sequence numbers, no resumption cursor): any
// single chunk is fully comparable alone. An empty listing yields one
// empty chunk, so an empty store still announces itself (probe #7).
// Identical store state produces the identical chunk sequence — the walk
// is over the sorted listing, never a map (probe #11).
func chunkSyncEntries(entries []port.SyncEntry, budget int) [][]port.SyncEntry {
	var chunks [][]port.SyncEntry
	// A budgeted chunk holds at most budget/entryOverheadBytes entries
	// (every entry estimates at least entryOverheadBytes), so that is the
	// capacity bound; a closed chunk's backing array is never reused —
	// cur resets to nil, not cur[:0], or every chunk after the first would
	// alias its predecessor's memory.
	cur := make([]port.SyncEntry, 0, budget/entryOverheadBytes)
	used := chunkOverheadBytes
	for _, entry := range entries {
		est := len(entry.Key) + entryOverheadBytes
		if len(cur) > 0 && used+est > budget {
			chunks = append(chunks, cur)
			cur = nil
			used = chunkOverheadBytes
		}
		cur = append(cur, entry)
		used += est
	}
	if len(cur) > 0 || len(chunks) == 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

// evictPeers removes every peer whose liveness expired (REPL-06): the first
// Advance with now at or after lastHello+cfg.PeerTimeout evicts — the
// at-or-after boundary, probe #18 — and eviction removes the peer from the
// table AND from every pending entry's wait-set in the same event. It never
// drains an entry by itself: with the drain rule, an entry whose every
// waiter was evicted is retained in suspension until a future Hello
// re-adds waiters via the flush rule. Eviction deliberately never touches a
// pending entry's acked set: an Ack that never arrived stays un-Acked, so
// PendingSnapshot keeps eviction distinguishable from an Ack (T-05-04,
// RESEARCH Pitfall 3). Only a Hello ever refreshed lastHello
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
			// wait-set only — acked stays (an eviction is never an Ack).
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
// (REPL-02, research Pitfall 1 / probe #9). Every emission counts in the
// retransmissions counter (OBS-01) — the adapter mirrors it post-event.
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
		c.retransmitted++
		entry.backoff = min(backoffGrowth*entry.backoff, c.cfg.AckTimeoutMax)
		entry.nextRetry = now.Add(entry.backoff)
		c.pending[key] = entry
	}
}

// nextDeadline returns the earliest strictly-future deadline over the armed
// timers: the hello timer, the sync timer, the earliest peer expiry, while
// any pending entry exists the retransmit scan deadline, and while any
// response is queued the next egress slot. After syncListing, evictPeers,
// scanRetransmit and drainEgress have fired everything due, every
// contribution is strictly future.
func (c *Core) nextDeadline(now time.Time) time.Time {
	deadline := c.nextHello
	if !c.nextSync.IsZero() && c.nextSync.Before(deadline) {
		deadline = c.nextSync
	}
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

// MergeStats returns the remote-merge counters (OBS-01, D-10) — a value
// copy, safe to read from any goroutine the driver permits (the Discards
// precedent); the production adapter mirrors them into atomics after every
// loop event.
func (c *Core) MergeStats() MergeStats {
	return c.mergeStats
}

// Retransmissions returns the count of retransmission datagrams the backoff
// ladder has emitted (OBS-01) — a value read the adapter mirrors post-event
// like the discard counters.
func (c *Core) Retransmissions() uint64 {
	return c.retransmitted
}

// SyncArrived reports whether at least one DataSync datagram has been
// received — the startup gate's sync half (API-04), false until the first
// handleDataSync and true forever after. The reading is pinned by A1: the
// first received DataSync datagram counts as the first sync round, because
// the protocol deliberately carries no chunk-completeness state — chunks
// are self-sufficient (chunkSyncEntries) — and the startup_gate timeout is
// the backstop for a node whose peers never announce. Set on the driver
// goroutine only; like Discards this is a value read sanctioned for the
// adapter's post-event mirroring, so cross-goroutine readers (Phase 5's
// gate resolver) see the state through the adapter's atomic, never by
// calling into a core the driver owns.
func (c *Core) SyncArrived() bool {
	return c.syncArrived
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
// ascending), the peers that have Acked it (sorted ascending, idempotent —
// eviction never adds here, so a peer absent from both sets was evicted by
// peer_timeout, never silently Acked: T-05-04) and the retransmission
// ladder's next retry time.
type PendingInfo struct {
	Version   version.RecordVersion
	WaitingOn []uint16
	Acked     []uint16
	NextRetry time.Time
}

// PendingSnapshot returns a copy of the pending set keyed by key — the
// observability seam Phase 5's wait_for_replication and Phase 6's OBS-01
// consume. Keys are iterated sorted and every WaitingOn is sorted ascending:
// no map-iteration order reaches a read path either (probe #9 discipline).
// Acked is cloned, not aliased — the snapshot shares no backing array with
// core state (the storage.Records defensive-copy precedent).
func (c *Core) PendingSnapshot() map[string]PendingInfo {
	out := make(map[string]PendingInfo, len(c.pending))
	for _, key := range slices.Sorted(maps.Keys(c.pending)) {
		entry := c.pending[key]
		out[key] = PendingInfo{
			Version:   entry.version,
			WaitingOn: slices.Sorted(maps.Keys(entry.waitSet)),
			Acked:     slices.Clone(entry.acked),
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
