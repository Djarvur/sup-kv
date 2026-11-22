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
	"container/heap"
	"container/list"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// SupportedProtoVersion is the wire protocol version this core speaks: the
// envelope version of every datagram it emits, and any inbound datagram
// whose version differs is discarded as BadProtoVersion (REPL-09). Version 2
// carries the DataSync completeness header (sync_id, chunk_index,
// chunk_count): a version-1 listing cannot express completeness, so accepting
// it would restore the first-chunk reading this version removes (PROTO-01).
const SupportedProtoVersion = 2

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

	// DiscardEmptyKey and its four siblings below are the wire-validation
	// classes (XPORT-03, XPORT-04, XPORT-10, D-06), in the pinned
	// intra-message order [Core.validate] applies: a datagram carrying any
	// one of them is discarded before [Core.handleMessage], so no store
	// write, no clock call and no timer ever observes it.
	DiscardEmptyKey          = "empty_key"
	DiscardKeyTooLong        = "key_too_long"
	DiscardTombstoneWithData = "tombstone_with_data"
	DiscardBadChecksum       = "bad_checksum"
	DiscardBadVersion        = "bad_version"

	// DiscardBadSyncHeader is the DataSync header class (PROTO-01, T-4-02):
	// a listing part whose chunk_count/chunk_index pair is outside its domain
	// — count 0, an index outside 1..count, a count above the stream cap, or
	// a count disagreeing with the live round — is discarded before any
	// accumulation, clock call or entry comparison.
	DiscardBadSyncHeader = "bad_sync_header"
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

	// defaultMaxClockLead is the inbound version bound's design default
	// (XPORT-04, D-06): the documented drift tolerance, equal to
	// drift_warn's default, so a peer inside the drift window stays
	// accepted while a crafted far-future version is rejected before any
	// store or clock call. A Config.MaxClockLead of zero or less is
	// normalized to this value.
	defaultMaxClockLead = 60 * time.Second

	// discardWarnInterval is the per-class rate limit of the validation
	// WARNs (T-3-25): one WARN per class per this much event-loop time.
	// Driven by the handler's now, never a wall clock read.
	discardWarnInterval = 10 * time.Second

	// maxSyncResponseRate is the top of SyncResponseRate's valid domain.
	// At any rate above it, one second divided by the rate as a
	// [time.Duration] truncates to a zero interval and silently disables
	// the egress limiter (D-15 P4-WR-02), so normalized() clamps every
	// out-of-domain rate — zero, negative or above this cap — to
	// defaultSyncResponseRate: the floor(1s/rate) spacing stays non-zero
	// at every configuration, and the limiter is never silently off.
	maxSyncResponseRate = 1e9

	// maxEgressEntries bounds the reactive-unicast egress queue (XPORT-07,
	// RESEARCH Pattern 9): twice the documented 20 000-key scale plus
	// headroom. An overflow drops the OLDEST queued answer together with its
	// dedupe identity (T-3-21), and the drop is counted in EgressDropped.
	maxEgressEntries = 32768

	// maxListingDestinations bounds the in-flight listing destinations — the
	// concurrent targeted and periodic FIFOs (CR-01, T-4-23). One FIFO per
	// known peer plus the broadcast round is the healthy shape; a hostile
	// segment can claim at most this many distinct (node_id, run_id)
	// identities, and every further trigger is refused and counted instead of
	// minting a queue entry (the destination is never the datagram's source
	// handle, so a spoofed port buys nothing, T-4-25). The value leaves room
	// for a 32-node segment.
	maxListingDestinations = 32

	// maxListingChunks bounds the total queued listing chunk work across every
	// destination (CR-01, T-4-23): a worst-case 20 000-key round at one entry
	// per chunk (20 000 chunks) fits whole, with headroom. A trigger that
	// would cross the bound is refused whole — never partially enqueued — and
	// counted, so a hostile restart flood cannot accumulate listings faster
	// than the shared pacer drains them.
	maxListingChunks = 65536

	// maxSyncStreams bounds the partial-listing accumulator: a peer may send
	// many distinct sync_ids with missing chunks, so the number of retained
	// streams is capped and an overflow evicts the least-progressed stream
	// deterministically (T-4-03). The cap is deliberately small: a healthy
	// segment has one in-flight round per peer plus transient stragglers.
	maxSyncStreams = 16

	// maxSyncEntriesPerStream bounds one stream's accumulated key→version
	// entries (T-4-03). A stream beyond it is dropped whole, so a single
	// hostile round cannot fill memory before the total cap reacts.
	maxSyncEntriesPerStream = 65536

	// maxSyncEntriesTotal bounds the accumulator's total retained entries
	// across every stream — the in-flight entry cap, matching
	// maxListedKeysTotal below. Overflow evicts whole streams, least
	// progressed first, and counts each eviction (T-4-03).
	maxSyncEntriesTotal = 131072

	// maxSyncChunks bounds the accepted chunk_count (T-4-02): sized for a
	// worst-case 20 000-key round at one entry per chunk, with headroom. A
	// header above it is a bad_sync_header violation, never accumulated
	// state.
	maxSyncChunks = 65536

	// maxListedKeysTotal bounds the retained per-peer listed-key payload —
	// the union's size, coincidentally equal to maxSyncEntriesTotal because
	// both model the same 20 000-key scale with headroom (T-4-41). An
	// overflow evicts whole peer sets in LRU order, so the union, the
	// refcounts and the retained total shrink together; each released key is
	// counted in ListedKeysEvicted.
	maxListedKeysTotal = 131072

	// seenRunsCap bounds the per-peer history of previously current run_ids
	// (PROTO-06, T-4-30): a fixed ring behind the alternating-identity
	// discrimination, so a hostile flapping identity cannot grow per-peer
	// memory without bound. When the ring wraps, the oldest entry is forgotten
	// and an ancient run_id is treated as a restart again — documented,
	// bounded behavior rather than unbounded memory.
	seenRunsCap = 8

	// jitterDivisor is the jitter envelope's denominator: timer jitter spans
	// a tenth of the interval (positive-only), pacer jitter a tenth of the
	// spacing (±).
	jitterDivisor = 10

	// Deterministic jitter salts: one per call site, so the three independent
	// jitter streams never alias and a same-seed replay stays byte-identical
	// (T-3-22).
	helloJitterSalt   = 0x9e37_79b9_7f4a_7c15
	syncJitterSalt    = 0xbf58_476d_1ce4_e5b9
	spacingJitterSalt = 0x94d0_49bb_1331_11eb

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
	// tag plus length varint), the Body oneof wrapper, the DataSync
	// repeated-field framing and the completeness header (sync_id,
	// chunk_index, chunk_count — roughly 60-70 bytes on the real wire with
	// their field tags and a multi-byte sync_id). The stated guarantee of the
	// two constants together: the real encoded datagram for any budgeted
	// chunk is at most SyncChunkBytes.
	chunkOverheadBytes = 96
)

// Config holds the replication timing parameters; it is immutable once
// handed to New. Zero or negative fields normalize to the docs/IDEA.md
// §Parameters defaults, and a nil Log or OnDiscard normalizes to a no-op:
// the core is total over any Config. RetransmitInterval bounds how long the
// driver may sleep while any pending entry exists (the scan-granularity
// reading of IDEA.md's parameter list, research A5); the per-key backoff
// ladder itself doubles from AckTimeout and saturates at AckTimeoutMax —
// documented with every step at Core.scanRetransmit. SyncResponseRate paces
// the three outbound streams: the egress queue, the listing queue and the
// record-send queue are each spaced floor(1s/SyncResponseRate)
// nanoseconds apart by integer division (probe #16); the first response
// after an idle queue is immediate. Rates outside the domain (0, 1e9] clamp
// to the 500/s default —
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

	// MaxClockLead bounds how far ahead of the handler's now a received
	// record version's unix_ts_ms reading may lie (XPORT-04, D-06): a
	// version beyond now+MaxClockLead is discarded as DiscardBadVersion
	// before any store or clock call, so a crafted far-future stamp can
	// never poison the hybrid logical clock. A zero or negative value
	// normalizes to defaultMaxClockLead (60 s, the documented drift
	// tolerance) — a scenario that legitimately needs more (a far-ahead
	// clock under test) sets its own explicit value; the production
	// default is never widened for tests.
	MaxClockLead time.Duration

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
	if c.MaxClockLead <= 0 {
		c.MaxClockLead = defaultMaxClockLead
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
// records join the store through ApplyRemote, and the sync paths read the
// tombstone-visible view — one version probe per listing entry, one point
// read per reply, and the shallow version map for building listings. The
// old whole-store snapshot method is deliberately absent: no sync path has a
// legitimate use for it, and its removal makes the O(store)-per-datagram
// cost compile-impossible (XPORT-06). The concrete engine (internal/app/persist
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

	// Version returns the stored version for key — tombstones included — and
	// false when the key holds no record. One lock-protected map lookup and
	// no Data copy: the per-entry probe a DataSync comparison takes.
	Version(key string) (version.RecordVersion, bool)

	// Lookup returns a copy of the stored record for key, tombstones
	// included. This is the drain-time exact-record source for every queued
	// DataReplication answer (a DataRequest reply or a strictly-newer DataSync
	// or stale-record response), so no producer needs a value copy before the
	// egress dedupe check (CR-02).
	Lookup(key string) (storage.Record, bool)

	// Versions returns a fresh map from every key — tombstones included — to
	// its stored version, copying no Data: the DataSync listing source.
	Versions() map[string]version.RecordVersion

	// Observe absorbs one received version into the store's clock without
	// touching any record: the per-DataSync-part HLC update (PROTO-01), so
	// the next local write strictly beats the peer's listed state. The
	// returned drift is the clock's own signal; MergeStats does not count it
	// this phase.
	Observe(ver version.RecordVersion) (drift time.Duration)

	// Generation returns the store's monotone content generation: it moves
	// only when a stored record actually changed and never for a rejected or
	// read path. A listing build records it and is reused only while it is
	// unchanged (04-08, T-4-43).
	Generation() uint64
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

	// The wire-validation classes (XPORT-03, XPORT-04, XPORT-10): one
	// counter per rejection [Core.validate] produces. EmptyKey and
	// KeyTooLong cover the key domain of DataReplication, DataRequest, Ack
	// and every DataSync entry; TombstoneWithData, BadChecksum and
	// BadVersion apply to DataReplication records (and BadVersion to
	// DataSync entry versions too).
	EmptyKey          uint64
	KeyTooLong        uint64
	TombstoneWithData uint64
	BadChecksum       uint64
	BadVersion        uint64

	// BadSyncHeader covers the DataSync completeness header's domain
	// (PROTO-01): a part outside chunk_count>=1, 1<=chunk_index<=count,
	// count<=maxSyncChunks, or inconsistent with a live round is discarded
	// before any accumulation or clock update.
	BadSyncHeader uint64
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
// rotation is the per-peer listing round counter: it rotates the start chunk
// of every targeted (restart) listing toward this peer, so repeated rounds
// cover the whole key space under buffer loss (D-01's targeted half). The
// counter lives in the row across restarts, which keeps the rotation
// advancing without any disk state. seenRuns is the bounded history of run
// ids this row announced before the current one: every accepted run change
// pushes the outgoing current run, so a return to an old identity is an
// alternating-run_id configuration error instead of a restart (PROTO-06).
type peer struct {
	handle    string
	runID     [16]byte
	lastHello time.Time
	rotation  uint64

	// seenRuns is the fixed 8-slot ring behind the alternating-identity
	// discrimination (T-4-27, T-4-30): a fixed array, never a growing slice,
	// so a hostile flapping identity cannot grow per-peer memory without
	// bound. seenLen counts the valid entries (from slot 0 while the ring has
	// not wrapped) and seenPos is the next write position.
	seenRuns [seenRunsCap][16]byte
	seenLen  uint8
	seenPos  uint8
}

// pushSeenRun records run as one of this row's previously current run_ids,
// keeping the most recent seenRunsCap entries and overwriting the oldest when
// the ring wraps.
func (p *peer) pushSeenRun(run [16]byte) {
	p.seenRuns[p.seenPos] = run
	p.seenPos = (p.seenPos + 1) % seenRunsCap
	if p.seenLen < seenRunsCap {
		p.seenLen++
	}
}

// seen reports whether run is one of the previously announced run_ids still
// inside the bounded ring. Slots past seenLen are never consulted, so an
// all-zero run_id only matches after it was genuinely announced.
func (p *peer) seen(run [16]byte) bool {
	for i := range p.seenLen {
		if p.seenRuns[i] == run {
			return true
		}
	}
	return false
}

// egressKind separates the two reactive unicast message kinds that share the
// egress queue: a DataRequest and a DataReplication answer.
type egressKind uint8

const (
	egressDataRequest egressKind = iota
	egressDataReplication
)

// egressKey is the egress queue's dedupe identity — (handle, kind, key),
// exactly the RESEARCH RPL-11 wording (T-3-20/T-3-21). The record version is
// deliberately excluded: a DataRequest carries none, and a DataReplication
// answer re-reads the stored record at drain time. The queue never pins Data
// and no producer looks a record up before the dedupe check (CR-02): a
// queued answer costs a key plus this small struct, so a remote listing that
// reveals every key cannot make the node retain a value copy per entry, and a
// repeated request allocates nothing before the identity collapses it.
type egressKey struct {
	handle string
	kind   egressKind
	key    string
}

// queuedUnicast is one reactive unicast response waiting in the egress queue
// (REPL-05's rate limiter): its dedupe identity — destination handle (D-03),
// kind and key — and NOTHING else. No message is retained: the drain builds
// the DataRequest or DataReplication from the identity (resolveQueuedResponse),
// so an updated record simply ships as its current version and a vanished key
// emits nothing. Exactly three producers feed it — a DataRequest for a
// missing-or-older listing entry, a DataRequest reply, and a DataSync
// local-newer answer (the egress-limiter scope pin) — and nothing else ever
// queues: proactive traffic (Hello, listings, retransmits) bypasses the
// limiter, which is what bounds an empty node's recovery instead of letting
// it become a flood (T-04-07).
type queuedUnicast struct {
	id egressKey
}

// listingDst identifies one listing destination: a peer's announced
// (node_id, run_id) — never the datagram's source handle, which any host
// chooses freely and which therefore cannot key a bounded queue (CR-01,
// T-4-25) — or, as the zero identity, the periodic broadcast round.
type listingDst struct {
	nodeID uint16
	runID  [16]byte
}

// broadcastDst returns the listing queue's destination identity for the
// periodic broadcast round: the zero identity, which no peer can hold (node
// ids start at 1 and every run_id is 16 random bytes). A function rather than
// a package global keeps the core's state model free of mutable package state.
func broadcastDst() listingDst { return listingDst{} }

// listingBuild is one immutable DataSync listing built from a
// StateStore.Versions view: the sorted chunk sequence, the build instant, the
// store content generation it was copied at, and the round's sync_id. Every
// FIFO created while the build is younger than cfg.SyncInterval AND the store
// generation is unchanged holds a pointer to it, so a burst of triggers costs
// one O(store) copy, never one per trigger (CR-01, T-4-24). The generation is
// what keeps the cache honest: a build is a snapshot, so any store mutation
// since makes it stale and the next trigger rebuilds exactly once — a cached
// pre-mutation listing can never be served.
type listingBuild struct {
	chunks     [][]port.SyncEntry
	at         time.Time
	syncID     uint64
	generation uint64
}

// listingFIFO is one destination's in-flight DataSync listing: a reference to
// the shared build, the chunk the round starts at (rotation) and how many
// chunks were already emitted. The shared build makes a queued destination
// O(1) — no per-destination chunk copy (CR-01) — while the round's sync_id
// and chunk count travel on every emitted chunk. Exactly one FIFO exists per
// destination, so a trigger that arrives while a listing is in flight is a
// no-op (T-3-23); rotation changes only the emission order, because each
// chunk keeps its ORIGINAL 1-based index and completeness would otherwise be
// unrecognizable.
type listingFIFO struct {
	build      *listingBuild
	dst        listingDst
	startIndex int
	emitted    int
}

// scanDiscardKey identifies one WARN rate-limit slot: a datagram's node_id
// and its discard class (D-06/D-07, T-4-04). The peer identity belongs in the
// key — a duplicate-node-id or foreign-version flood from one peer must not
// suppress the first WARN another peer deserves.
type scanDiscardKey struct {
	nodeID uint16
	class  string
}

// syncStreamKey identifies one peer's in-flight listing round: its node_id,
// its run_id and the round's sync_id. run_id belongs in the key so a
// restarted peer's fresh sync_id counter can never collide with the partial
// state its previous run left behind.
type syncStreamKey struct {
	peer uint16
	run  [16]byte
	id   uint64
}

// syncStream is one peer's partial listing round: the round's chunk count,
// the set of chunk indices received so far, and the accumulated key→version
// entries. The round is complete exactly when every index 1..count is in
// got — arrival of any single chunk proves nothing (PROTO-01).
type syncStream struct {
	count   uint32
	got     map[uint32]struct{}
	entries map[string]version.RecordVersion
}

// recordSend is one queued record emission: the tracked pair and whether the
// send is a ladder retransmission (retry) or the discovery flush's first
// send. The record itself is never retained — drainRecordSends re-reads it
// through StateStore.Lookup at emission time (RESEARCH Pattern 2).
type recordSend struct {
	pk    pendingKey
	retry bool
}

// pendingKey identifies one tracked local write in the pending set: the key
// and the exact record version awaiting Acks. The version is part of the
// identity, not a field of the entry alone, so a superseded version can never
// be confused with the write that replaced it (REPL-02, D-03).
type pendingKey struct {
	key     string
	version version.RecordVersion
}

// pendingRef is one node of Core.pendingOrder: the tracked pair plus its
// creation time, kept beside the key so the order walk never needs a map
// lookup. The list is maintained in createdAt order, so its front is the
// oldest entry UnackedStats reports.
type pendingRef struct {
	pk        pendingKey
	createdAt time.Time
}

// pendingEntry is one local write awaiting peer Acks (REPL-02). It retains
// no record: the store is the single record source, and every emission
// resolves the record through StateStore.Lookup, skipping when the stored
// version no longer matches the tracked pair (the memory half of R-02 and
// the stale-send guard). The entry holds the exact tracked version, the
// wait-set of known peers that have not yet Acked that version, the set of
// peers that have (sorted ascending, idempotent — handleAck inserts;
// eviction never touches it, which is what keeps an Ack distinguishable from
// a peer_timeout eviction in PendingSnapshot, T-05-04), the retransmission
// ladder state (next retry time and current backoff — the ladder itself
// fires from Advance), the creation time behind UnackedStats' oldest age,
// and the entry's node in Core.pendingOrder.
type pendingEntry struct {
	version   version.RecordVersion
	waitSet   map[uint16]struct{}
	acked     []uint16
	nextRetry time.Time
	backoff   time.Duration
	createdAt time.Time
	elem      *list.Element
}

// retryItem is one due-heap node: the tracked pair, its armed retransmission
// deadline and the monotonically increasing push sequence that breaks
// equal-deadline ties deterministically (the simnet eventHeap shape).
type retryItem struct {
	nextRetry time.Time
	seq       uint64
	pk        pendingKey
}

// retryHeap is the indexed min-heap of pending retry deadlines: items order
// by (nextRetry, seq), and index locates a tracked pair's item in O(1), so a
// re-arm is [heap.Fix] and a deletion is [heap.Remove] — both O(log n) and
// leaving no stale item behind (T-3-14). The heap is the only due-time index
// Advance consults: an iteration with nothing due touches the root alone.
type retryHeap struct {
	items []retryItem
	index map[pendingKey]int
}

// Len implements [heap.Interface].
func (h *retryHeap) Len() int { return len(h.items) }

// Less implements [heap.Interface]: earlier deadline first, then the push
// sequence — a deterministic total order for equal deadlines.
func (h *retryHeap) Less(i, j int) bool {
	if !h.items[i].nextRetry.Equal(h.items[j].nextRetry) {
		return h.items[i].nextRetry.Before(h.items[j].nextRetry)
	}
	return h.items[i].seq < h.items[j].seq
}

// Swap implements [heap.Interface] and keeps index in step with the heap.
func (h *retryHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.index[h.items[i].pk] = i
	h.index[h.items[j].pk] = j
}

// Push implements [heap.Interface].
func (h *retryHeap) Push(x any) {
	item := x.(retryItem)
	h.index[item.pk] = len(h.items)
	h.items = append(h.items, item)
}

// Pop implements [heap.Interface]: container/heap pops the root by swapping
// it to the end first, so removing the last item is the root removal; the
// index entry goes with it.
func (h *retryHeap) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	old[n-1] = retryItem{}
	h.items = old[:n-1]
	delete(h.index, item.pk)
	return item
}

// Core is the replication decision state machine for one node. Not safe for
// concurrent use: exactly one driver goroutine — the adapter's event loop
// in production, the simulator's scheduler in tests — calls the handlers
// (D-04); the copying accessors (Discards, SyncArrived, PeerSnapshot,
// PendingSnapshot) are the sanctioned cross-goroutine read seams, consumed
// through the production adapter's post-event atomics. Results, never
// callbacks: no handler takes a callback, a channel or a goroutine.
type Core struct {
	nodeID uint16
	runID  [16]byte
	store  StateStore
	cfg    Config
	peers  map[uint16]peer

	// pending holds one entry per tracked (key, version) pair — no record
	// Data (see pendingEntry). peerUnacked is the per-peer ledger: for every
	// known peer, the set of pending keys it is still waiting on, so peer
	// eviction touches only that peer's unacked entries instead of scanning
	// the whole pending set (D-03). pendingOrder lists every pending pair in
	// createdAt order for the O(1) oldest-age gauge.
	pending      map[pendingKey]pendingEntry
	peerUnacked  map[uint16]map[pendingKey]struct{}
	pendingOrder *list.List

	// retry is the indexed due-heap over the same pending pairs;
	// retrySeq assigns each push's equal-deadline tie-break.
	retry    retryHeap
	retrySeq uint64

	nextHello time.Time

	// nextSync is the next sync-timer deadline; the zero time until the
	// first Advance, whose startup trigger fires the initial full listing
	// alongside the startup Hello (IDEA §Node lifecycle). Each firing arms
	// now + SyncInterval + a positive-only deterministic jitter, so the
	// deadline never fires before the configured interval.
	nextSync time.Time

	// listingByDst holds the in-flight listing FIFOs keyed by destination
	// identity (a peer's (node_id, run_id) or the zero broadcast identity);
	// listingOrder is their enqueue order — the deterministic drain order — and
	// listingCount the total queued chunk count across every FIFO. build is the
	// shared, generation-checked listing build the FIFOs reference; nextListingAt
	// is the shared listing pacer slot and syncRotation the broadcast round
	// counter that rotates each round's start chunk (D-01, RESEARCH Pattern 5).
	// listingDropped counts every trigger the destination or chunk cap refused
	// (CR-01, T-4-23).
	listingByDst   map[listingDst]*listingFIFO
	listingOrder   []listingDst
	listingCount   int
	nextListingAt  time.Time
	syncRotation   uint64
	build          *listingBuild
	listingDropped uint64

	// recordQueue is the paced record-send FIFO fed by the discovery flush and
	// the retransmit scanner; recordQueued is its (key, version) dedupe set and
	// nextRecordAt the shared pacer slot (D-05, RESEARCH Pattern 2).
	recordQueue  []recordSend
	recordQueued map[pendingKey]struct{}
	nextRecordAt time.Time

	// Per-site monotone counters: the deterministic jitter sequence for the
	// Hello and sync timers and for the listing and record pacer spacings
	// (T-3-22 — deterministic hash only, same-seed replay stays byte-identical).
	helloJitterSeq   uint64
	syncJitterSeq    uint64
	listingJitterSeq uint64
	recordJitterSeq  uint64

	egress []queuedUnicast

	// egressIDs is the egress queue's dedupe set, kept in lockstep with the
	// queue: a pop deletes its identity and an overflow drop deletes the
	// evicted identity in the same operation (T-3-21). egressDropped counts
	// every drop.
	egressIDs     map[egressKey]struct{}
	egressDropped uint64

	// nextSendAt is the earliest virtual time the next queued response may
	// emit; it is meaningful exactly while the egress queue is non-empty
	// (enqueueResponse arms it, the drain advances it).
	nextSendAt time.Time

	// pacedSends counts every datagram the three pacers — listing chunks,
	// record sends and egress responses — hand to the Outbox, the emission
	// volume the metrics surface publishes. It counts EMISSIONS, never
	// enqueues: a queue insertion is not a send (T-3-41), and RPL-18's
	// broader send-failure accounting stays Phase 7's (OBSV-03), not
	// compounded here. Driver-goroutine-only mutation behind the copying
	// accessor, the Discards precedent.
	pacedSends uint64

	// syncResponseInterval is the shared pacer spacing — the egress queue's
	// per-response and the listing and record pacers' base — derived once
	// at construction by the pinned deterministic integer arithmetic:
	// floor(1s/SyncResponseRate) nanoseconds by integer division — exactly
	// 2ms at the 500/s default and 3333333ns at 300/s (probe #16, echoed in
	// docs/IDEA.md §Parameters). The pacers never recompute it from a raw
	// config value (RESEARCH Pitfall 22).
	syncResponseInterval time.Duration

	discards DiscardStats

	// discardWarnAt is the per-(node_id, class) rate-limit state of the
	// validation WARNs (T-3-25, T-4-04): the event-loop time of the last WARN
	// per slot. An absent entry (never warned) always fires; afterwards a slot
	// warns at most once per discardWarnInterval of event time. Map access is
	// driver-goroutine-only, like every other core field (D-04).
	discardWarnAt map[scanDiscardKey]time.Time

	// mergeStats counts the remote-merge signals (OBS-01); retransmitted
	// counts the ladder's retransmission emissions — driver-goroutine-only
	// mutations behind the copying accessors, the Discards precedent.
	mergeStats    MergeStats
	retransmitted uint64

	// notificationsRegressed counts local-write notifications the guard
	// refused because their version was not greater than the version tracked
	// for that key — regressive and equal duplicate notifications alike
	// (PROTO-05, D-05). Driver-goroutine-only mutation behind the copying
	// NotificationsRegressed accessor; Phase 7's OBSV surface owns publishing
	// it (no adapter mirror this phase).
	notificationsRegressed uint64

	// syncArrived flips true on the first received DataSync datagram and
	// never resets — the startup gate's sync half (API-04). Set by
	// handleDataSync on the driver goroutine; read through the copying
	// SyncArrived accessor, which the production adapter mirrors into an
	// atomic after every loop iteration so cross-goroutine readers never
	// touch core state.
	syncArrived bool

	// The listing accumulator (PROTO-01): one syncStream per
	// (peer, run, sync_id) round, bounded by maxSyncStreams,
	// maxSyncEntriesPerStream and maxSyncEntriesTotal; syncEntries is the
	// total retained entry count the total cap reads, and syncStreamDropped
	// counts every stream the caps evicted.
	syncs             map[syncStreamKey]*syncStream
	syncEntries       int
	syncStreamDropped uint64

	// The retained listed-key payload (PROTO-01, T-4-41): each peer's last
	// complete listing as a key set, the union's per-key refcounts, the
	// union itself and its size, and the LRU over peers (listedOrder front =
	// least recently published) that bounds the payload. listedKeysEvicted
	// counts every key released by an eviction.
	listedByPeer      map[uint16]map[string]struct{}
	listedRefs        map[string]int
	listedKeys        map[string]struct{}
	listedKeyCount    int
	listedOrder       *list.List
	listedPos         map[uint16]*list.Element
	listedKeysEvicted uint64

	// listingGen is the published listing content version: it moves exactly
	// when the union membership or the complete flag actually changes, so an
	// identical repeated complete listing republishes nothing downstream.
	// listingComplete flips true on the first complete listing and never
	// resets; nextSyncID allocates one fresh sync_id per listing BUILD (never
	// per trigger — the shared build carries it, so two destinations reusing
	// one build share the round identity, T-4-24; build-time counter, never
	// crypto/rand — same-seed replay stays byte-identical).
	listingGen      uint64
	listingComplete bool
	nextSyncID      uint64

	// The no-Hello WARN state (XPORT-09, D-07): firstAdvance is the event
	// time of the first Advance — the warning's clock origin — helloSeen
	// flips on the first Hello received from any peer, and peerWarned
	// latches the one-shot WARN. Driver-goroutine-only, like every other
	// core field (D-04); the WARN never repeats per interval.
	firstAdvance time.Time
	helloSeen    bool
	peerWarned   bool
}

// New returns a core for nodeID with the per-process runID. store must be
// non-nil: the core itself owns no I/O (the depguard boundary), so the
// adapter passes the store in and every decision reads and merges through
// it.
func New(nodeID uint16, runID [16]byte, store StateStore, cfg Config) *Core {
	cfg = cfg.normalized()
	return &Core{
		nodeID:               nodeID,
		runID:                runID,
		store:                store,
		cfg:                  cfg,
		peers:                make(map[uint16]peer),
		pending:              make(map[pendingKey]pendingEntry),
		peerUnacked:          make(map[uint16]map[pendingKey]struct{}),
		pendingOrder:         list.New(),
		retry:                retryHeap{index: make(map[pendingKey]int)},
		listingByDst:         make(map[listingDst]*listingFIFO),
		recordQueued:         make(map[pendingKey]struct{}),
		egressIDs:            make(map[egressKey]struct{}),
		discardWarnAt:        make(map[scanDiscardKey]time.Time),
		syncs:                make(map[syncStreamKey]*syncStream),
		listedByPeer:         make(map[uint16]map[string]struct{}),
		listedRefs:           make(map[string]int),
		listedKeys:           make(map[string]struct{}),
		listedOrder:          list.New(),
		listedPos:            make(map[uint16]*list.Element),
		nextSyncID:           1,
		syncResponseInterval: time.Second / time.Duration(cfg.SyncResponseRate),
	}
}

// HandleDatagram processes one inbound datagram. The classification order
// is pinned (D-02, REPL-08, REPL-09): a BadCRC framing verdict counts
// BadCRC (the adapter never decoded the body); Undecodable counts
// Undecodable; a ProtoVersionMismatch verdict — or a Decoded verdict whose
// ProtoVersion field differs, the defence-in-depth check for direct inputs —
// counts BadProtoVersion without any body; a datagram
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
// differs always reaches the Hello handler, which classifies a run_id it has
// seen before as the alternating-identity configuration error and only an
// unseen run_id as the targeted-listing restart (PROTO-06) — a genuine
// restart is never counted as a duplicate. Every discard site additionally
// fires cfg.OnDiscard with its class token and a metadata detail (D-08) —
// additive to the configuration-error Log calls, never their replacement;
// those Log calls now share the per-(node_id, class) rate limiter, so a
// flapping host cannot flood the log from any duplicate-node path (T-4-28).
//
// The surviving datagram then passes the wire-validation pipeline
// ([Core.validate], XPORT-03/04/10, D-06) BEFORE handleMessage — the
// position is the contract: handleMessage is the only path to the store
// (applyRemote is the sole ApplyRemote call, and therefore the sole
// Clock.Update path), so a rejected datagram can touch neither the store
// nor the clock, and no timer observes it either. The pinned
// intra-message order: DataReplication — empty key, key longer than
// storage.MaxKeyBytes, Deleted with non-empty Data, a record checksum that
// does not match storage.RecordChecksum, a version beyond
// now+cfg.MaxClockLead; DataRequest and Ack — empty key, key too long;
// DataSync — every entry in received order: empty key, key too long,
// version beyond the bound, with the first failing entry discarding the
// whole datagram; Hello and a nil Msg — nothing. Every rejection counts its
// own DiscardStats class, fires OnDiscard with the source handle as the
// detail (never a key name, the T-6-05 metadata contract), and logs one
// per-class rate-limited WARN on the event-loop's now (T-3-25).
//
// Everything else reaches protocol processing:
// Hello learns a previously unknown node and answers it with one targeted
// full listing (the first-seen listing, PROTO-04), refreshes a known node's
// liveness, or — when a known node's run_id changed — replaces the old run's
// listing with one targeted full listing toward the restarted peer (REPL-07),
// DataReplication applies the record and Acks it, Ack drains
// the pending-set wait-set (REPL-02), DataRequest answers through the
// rate-limited egress queue (REPL-04), and DataSync compares per entry
// against the store's tombstone-visible version view and queues DataRequest
// / unicast DataReplication responses on the same queue (REPL-05).
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
	case port.ProtoVersionMismatch:
		// The adapter judged the envelope's version before decoding the body
		// (D-07: a foreign version's body need not follow this version's CRC
		// contract), so there is no message — only the counted, attributed
		// discard, WARNed at most once per (node_id, class) per interval.
		c.discards.BadProtoVersion++
		c.cfg.OnDiscard(DiscardBadProtoVersion, in.Source)
		c.warnDiscard(now, in.NodeID, DiscardBadProtoVersion)
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
		c.warnRateLimited(now, in.NodeID, DiscardDuplicateNodeID, fmt.Sprintf(
			"sup-kv: configuration error: node_id %d announced by another run from handle %q, datagram discarded (REPL-08)",
			in.NodeID, in.Source,
		))
		return
	}
	if known, ok := c.peers[in.NodeID]; ok && in.RunID == known.runID && in.Source != known.handle {
		c.discards.DuplicateNodeID++
		c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf("node_id %d handles %q/%q", in.NodeID, known.handle, in.Source))
		c.warnRateLimited(now, in.NodeID, DiscardDuplicateNodeID, fmt.Sprintf(
			"sup-kv: configuration error: node_id %d known at handle %q announced again from handle %q, datagram discarded (REPL-08)",
			in.NodeID, known.handle, in.Source,
		))
		return
	}
	if class, detail, valid := c.validate(now, in); !valid {
		c.discardValidated(now, in.NodeID, class, detail)
		return
	}
	c.handleMessage(now, in, out)
}

// validate is the wire-validation pipeline (XPORT-03, XPORT-04, XPORT-10,
// D-06): it runs between HandleDatagram's framing/identity classification
// and handleMessage, so a rejected datagram never reaches a handler — and
// therefore never the store, the clock or a timer. The intra-message order
// is pinned (probe XPORT-10, tests pin every step):
//
//   - DataReplication: empty key, key longer than storage.MaxKeyBytes,
//     tombstone carrying Data, a checksum that does not match
//     storage.RecordChecksum(key, version, data, deleted), then a version
//     beyond now+cfg.MaxClockLead;
//   - DataRequest and Ack: empty key, then key too long (an Ack carries no
//     record, so no checksum or version validation applies);
//   - DataSync: the completeness header first — chunk_count>=1,
//     1<=chunk_index<=chunk_count, chunk_count<=maxSyncChunks, and a count
//     consistent with a live round — then every entry in received order:
//     empty key, key too long, version beyond the bound; the first failing
//     item discards the whole datagram;
//   - Hello, a standalone SyncEntry and a nil Msg: nothing (a Hello carries
//     no key or version, a standalone SyncEntry never travels, and a nil
//     Msg on a Decoded verdict violates the adapter contract).
//
// The returned detail is the source handle alone — never a key name or a
// record value (the T-6-05 metadata contract, T-3-27). valid reports
// whether the datagram may proceed.
func (c *Core) validate(now time.Time, in port.InboundDatagram) (class, detail string, valid bool) {
	switch msg := in.Msg.(type) {
	case port.DataReplication:
		return c.validateReplication(now, in.Source, msg)
	case port.DataRequest:
		return keyVerdict(in.Source, msg.Key)
	case port.Ack:
		return keyVerdict(in.Source, msg.Key)
	case port.DataSync:
		return c.validateSync(now, in, msg)
	}
	return "", "", true
}

// validateReplication applies the pinned record order: key domain,
// tombstone shape, checksum, then version. The version check is last so a
// datagram with several faults reports the earliest structural one, and —
// the point of the whole pipeline — a forgeable checksum never gets a
// version verdict instead.
func (c *Core) validateReplication(now time.Time, source string, msg port.DataReplication) (class, detail string, valid bool) {
	if violation := keyClass(msg.Key); violation != "" {
		return violation, source, false
	}
	if msg.Record.Deleted && len(msg.Record.Data) > 0 {
		return DiscardTombstoneWithData, source, false
	}
	sum := storage.RecordChecksum(msg.Key, msg.Record.Version, msg.Record.Data, msg.Record.Deleted)
	if sum != msg.Record.Checksum {
		return DiscardBadChecksum, source, false
	}
	if !c.versionAccepted(now, msg.Record.Version) {
		return DiscardBadVersion, source, false
	}
	return "", "", true
}

// validateSync applies the listing pipeline in pinned order: the completeness
// header FIRST (chunk_count>=1, 1<=chunk_index<=chunk_count,
// chunk_count<=maxSyncChunks, and a chunk_count consistent with a live
// round), then every entry in received order — key domain, then version. The
// first failing item discards the whole datagram, so a listing cannot smuggle
// one bad entry or one impossible header past a per-chunk verdict, and a
// rejected header leaves the live round's stored count untouched (T-4-02).
func (c *Core) validateSync(now time.Time, in port.InboundDatagram, msg port.DataSync) (class, detail string, valid bool) {
	if !validSyncHeader(msg) {
		return DiscardBadSyncHeader, fmt.Sprintf(
			"%s: chunk_count %d chunk_index %d", in.Source, msg.ChunkCount, msg.ChunkIndex,
		), false
	}
	key := syncStreamKey{peer: in.NodeID, run: in.RunID, id: msg.SyncID}
	if stream, ok := c.syncs[key]; ok && stream.count != msg.ChunkCount {
		return DiscardBadSyncHeader, fmt.Sprintf(
			"%s: chunk_count %d disagrees with the live round's %d", in.Source, msg.ChunkCount, stream.count,
		), false
	}
	for _, entry := range msg.Entries {
		if violation := keyClass(entry.Key); violation != "" {
			return violation, in.Source, false
		}
		if !c.versionAccepted(now, entry.Version) {
			return DiscardBadVersion, in.Source, false
		}
	}
	return "", "", true
}

// keyVerdict is the key-only half shared by DataRequest and Ack: both carry
// a key and no record, so nothing else is applicable.
func keyVerdict(source, key string) (class, detail string, valid bool) {
	if violation := keyClass(key); violation != "" {
		return violation, source, false
	}
	return "", "", true
}

// keyClass returns the discard class for key's domain violation — empty
// first, then the storage.MaxKeyBytes ceiling (XPORT-03) — or "" when the
// key is acceptable. Validation is receive-side only: the store keeps
// serving pre-upgrade records under longer keys, so reads stay permissive.
func keyClass(key string) string {
	if key == "" {
		return DiscardEmptyKey
	}
	if len(key) > storage.MaxKeyBytes {
		return DiscardKeyTooLong
	}
	return ""
}

// versionAccepted reports whether ver is a valid inbound record version at
// the handler's now under cfg.MaxClockLead — the layout and future-bound
// check of version.RecordVersion.Validate, which reads no clock (the D-02
// seam: now is the caller's) and never panics.
func (c *Core) versionAccepted(now time.Time, ver version.RecordVersion) bool {
	return ver.Validate(now, c.cfg.MaxClockLead) == nil
}

// discardValidated records one wire-validation rejection: it counts the
// class's own counter, fires cfg.OnDiscard with the class and the source
// detail, and logs a per-(node_id, class) rate-limited WARN naming only the
// class (T-3-25, T-3-27, T-4-04). The rate limit is driven by the
// event-loop's now — the lastWarn bookkeeping is a plain per-slot timestamp,
// never a wall clock read and never a global throttle. Counters stay the
// reliable signal; the WARN is the operator's coarse hint, and per-datagram
// discard logging stays the adapter's Debug-level OnDiscard bridge.
func (c *Core) discardValidated(now time.Time, nodeID uint16, class, detail string) {
	switch class {
	case DiscardEmptyKey:
		c.discards.EmptyKey++
	case DiscardKeyTooLong:
		c.discards.KeyTooLong++
	case DiscardTombstoneWithData:
		c.discards.TombstoneWithData++
	case DiscardBadChecksum:
		c.discards.BadChecksum++
	case DiscardBadVersion:
		c.discards.BadVersion++
	case DiscardBadSyncHeader:
		c.discards.BadSyncHeader++
	}
	c.cfg.OnDiscard(class, detail)
	c.warnDiscard(now, nodeID, class)
}

// warnRateLimited logs msg unless the same (node_id, class) slot already
// warned inside the last discardWarnInterval of event time — the one shared
// limiter behind every counted classification WARN: the validation classes,
// bad_sync_header and all three duplicate-node_id paths, so a flapping,
// hostile or misconfigured peer can flood neither the log nor the listing
// path (T-3-25, T-4-04, T-4-28). A slot's first warning always fires, and
// the bookkeeping is a plain per-slot timestamp on the handler's now, never
// a wall-clock read and never a global throttle.
func (c *Core) warnRateLimited(now time.Time, nodeID uint16, class, msg string) {
	key := scanDiscardKey{nodeID: nodeID, class: class}
	last, warned := c.discardWarnAt[key]
	if warned && now.Sub(last) < discardWarnInterval {
		return
	}
	c.discardWarnAt[key] = now
	c.cfg.Log(msg)
}

// warnDiscard logs one discard-class WARN through the shared limiter, naming
// only the class — the operator's coarse hint beside the reliable counters,
// while per-datagram discard logging stays the adapter's Debug-level
// OnDiscard bridge (T-3-25, T-3-27).
func (c *Core) warnDiscard(now time.Time, nodeID uint16, class string) {
	c.warnRateLimited(now, nodeID, class, fmt.Sprintf(
		"sup-kv: discarded inbound datagram: class %s from node %d (one warning per node and class per %s)",
		class, nodeID, discardWarnInterval,
	))
}

// handleMessage dispatches one classified datagram's decoded message to its
// protocol handler.
func (c *Core) handleMessage(now time.Time, in port.InboundDatagram, out *port.Outbox) {
	switch msg := in.Msg.(type) {
	case port.Hello:
		c.handleHello(now, in)
	case port.DataReplication:
		c.applyRemote(now, in.Source, msg, out)
	case port.Ack:
		c.handleAck(in.NodeID, msg, out)
	case port.DataRequest:
		c.handleDataRequest(now, in.Source, msg.Key)
	case port.DataSync:
		c.handleDataSync(now, in, msg, out)
	case port.SyncEntry, nil:
		// A standalone SyncEntry never travels (it lives inside DataSync),
		// and a nil Msg on a Decoded verdict violates the adapter contract
		// — nothing to do for either.
	}
}

// handleHello upserts the peer-table row from one Hello datagram (REPL-01)
// and owns the three-way outcome (REPL-07, PROTO-04, probe #19), pinned:
//
//   - an unknown node_id is learned and gets exactly one immediate targeted
//     listing toward its announced identity — the first-seen listing that
//     gives a newly seen or returning peer its state without a sync_interval
//     term — and, when the Hello also ends an empty table, the same event
//     still runs the discovery flush. The two behaviors are not mutually
//     exclusive and this branch must never return early (Pitfall 5): an evicted
//     peer's return Hello, whatever its run_id, is this unknown-node case.
//   - a known node_id repeating its remembered run_id refreshes lastHello
//     only, never a second listing (a per-Hello full listing at hello_interval
//     cadence would be a self-inflicted listing flood; the flap case is bounded
//     by this idempotence, not by rate limiting).
//   - a known node_id announcing a run_id it carried BEFORE but which is not
//     current is the alternating-run_id configuration error (PROTO-06, R-12):
//     counted as DuplicateNodeID, WARNed once per (node_id, class) interval
//     through the shared limiter, dropped, and given NO listing — a flapping
//     or hostile process must not be rewarded with a full listing per
//     datagram. The row and its liveness stamp are untouched, so the
//     currently remembered run remains authoritative.
//   - a known node_id announcing an unseen run_id is a detected restart — the
//     duplicate-node_id classifier never pre-empts this branch
//     (HandleDatagram's second variant requires run_id equality with the
//     remembered row) — the outgoing current run joins the row's bounded
//     seen-run history, the old run's queued listing, retained keys and
//     partial rounds are dropped so no dead state outlives the row, the entry
//     updates in place (handle, runID, lastHello; the table stays keyed by
//     node_id, one row per node — a change never forks a second row, which
//     would fork ack tracking and liveness; the row's listing rotation
//     survives) and one targeted listing is enqueued. A restart necessarily
//     has discovery == false, so this branch returns.
//
// Both listing paths call the single [Core.enqueueListing] — there is no
// second implementation. The first-seen round starts at chunk 0; the restart
// round uses and advances the row's rotation counter, so repeated restarts
// cover the whole key space under buffer loss (D-01's targeted half).
// The restart test is "run_id unseen by this row", not "run_id differs": the
// bounded seen-run ring remembers the previous 8 current run_ids, so an
// alternation back to a recent identity is the named configuration error
// while a genuinely new run — 16 random bytes per process start, carrying no
// temporal order — is still a restart. When the ring wraps, the oldest
// identity is forgotten and treated as a restart again (documented, bounded
// behavior: T-4-30). In-flight
// protocol datagrams from the old run — Ack, DataReplication, DataSync —
// process normally on their (key, version) merits, run_id-agnostic outside
// this handler. The listing is proactive traffic on the shared listing pacer,
// never the sync_response_rate egress queue (the plan-04-03 egress-scope pin):
// it is bounded by one in-flight listing per destination plus the queue caps,
// and fires once per detected first Hello or run change, while the heavy
// replay traffic it solicits IS limiter-gated — that flows as DataRequest
// replies on the receiving node's side and newer-responses on ours. A detected
// restart never re-adds the peer to any pending wait-set and never touches the
// backoff ladder (the REPL-02 interplay): the restarted node heals through
// reconciliation via the targeted listing, not through retransmission, and an
// entry that already drained stays drained — the dump/restart path, not
// retransmission, is the durability story. Only a Hello refreshes lastHello —
// the field's name is the liveness contract, and no other datagram type ever
// touches it. A Hello that ends an empty table is discovery: in this same
// event the core enqueues the pending set on the paced record-send queue —
// one broadcast DataReplication per entry in sorted key order, the new peer
// added to every wait-set, every ladder restarted at cfg.AckTimeout — so a
// brand-new peer receives the pending set promptly, not at the next backoff
// expiry (research A6), while the pacer prevents the whole-set burst R-01/R-02
// measured. A Hello that does not end an empty table modifies no wait-set:
// only discovery adds waiters.
func (c *Core) handleHello(now time.Time, in port.InboundDatagram) {
	// Any received Hello — a discovery, a refresh or a restart — proves the
	// receive path hears peers, so the no-Hello WARN stands down for the
	// process lifetime (XPORT-09).
	c.helloSeen = true
	discovery := len(c.peers) == 0
	known, remembered := c.peers[in.NodeID]
	if remembered {
		switch {
		case known.runID == in.RunID:
			// A repeat Hello is idempotence: refresh the liveness stamp, emit
			// nothing, never a second listing.
			known.handle = in.Source
			known.lastHello = now
			c.peers[in.NodeID] = known
			return
		case known.seen(in.RunID):
			// A run_id this row carried before and no longer has is an
			// alternating identity — a named configuration error (PROTO-06):
			// counted, WARNed through the shared per-(node_id, class) limiter,
			// dropped with no listing and no row change, so a flapping
			// process cannot force a full listing per datagram.
			c.discards.DuplicateNodeID++
			c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf(
				"node_id %d announced a previously seen run_id from handle %q", in.NodeID, in.Source))
			c.warnRateLimited(now, in.NodeID, DiscardDuplicateNodeID, fmt.Sprintf(
				"sup-kv: configuration error: node_id %d alternated to a previously seen run_id from handle %q, datagram discarded (PROTO-06)",
				in.NodeID, in.Source,
			))
			return
		}
		// A new run under this node_id no longer stands behind the old run's
		// listing: its keys leave the retained union (a content change the
		// mirror must publish), its partial rounds are discarded and its
		// queued FIFO is dropped, so neither the bounded listing queue nor the
		// accumulator inherits dead state (CR-01). The outgoing current run
		// joins the row's bounded history before the row moves on, which is
		// what makes a later return to it an alternation.
		if c.dropListedPeer(in.NodeID) > 0 {
			c.listingGen++
		}
		c.dropPeerSyncStreams(in.NodeID)
		c.dropListing(listingDst{nodeID: in.NodeID, runID: known.runID})
		known.pushSeenRun(known.runID)
	}
	known.handle = in.Source
	known.runID = in.RunID
	known.lastHello = now
	c.peers[in.NodeID] = known
	// One targeted listing per (node_id, run_id) destination. Only a restart
	// advances the rotation: the first-seen round is round 0, and the repeated
	// rounds that must cover the key space are the restarts.
	if c.enqueueListing(now, listingDst{nodeID: in.NodeID, runID: in.RunID}, known.rotation) && remembered {
		known.rotation++
		c.peers[in.NodeID] = known
	}
	if remembered || !discovery {
		return
	}
	for _, pk := range sortedPendingKeys(c.pending) {
		entry := c.pending[pk]
		entry.waitSet[in.NodeID] = struct{}{}
		entry.backoff = c.cfg.AckTimeout
		entry.nextRetry = now.Add(c.cfg.AckTimeout)
		c.pending[pk] = entry
		c.fixRetry(pk, entry.nextRetry)
		c.ledgerAdd(in.NodeID, pk)
		c.enqueueRecordSend(now, pk, false)
	}
}

// applyRemote resolves one inbound DataReplication against the store's
// current state, with exactly three outcomes (PROTO-03, the refined D-03):
//
//   - strictly older incoming (a held version sorts after it): answer with
//     exactly one unicast DataReplication carrying the held record and send
//     NO Ack. The answer is queued as an identity-only egress entry and
//     re-reads the stored record at drain time, so the newer data is always
//     current. The store is not merged, the merge counters are untouched and
//     the tracking ledger neither gains nor loses an entry — an Ack here
//     would tell the sender a record it did not get was accepted.
//   - equal version: Ack it and emit no reply. The record is already held
//     (merge idempotence), and every equal duplicate must be confirmed or
//     retransmission ladders never drain (S-PROTO-16, R-02). The clock
//     already absorbed this version when the record was installed, so
//     skipping the merge changes no state.
//   - strictly newer or absent: fall through to ApplyRemote and Ack after
//     the in-memory apply — nothing here waits for a dump (REPL-03).
//
// The probe is one point Version read before ApplyRemote, so the merge can
// never change the verdict mid-decision. A store error on the apply path
// (for example a value over the configured limit) is surfaced through Log,
// never a panic: a hostile or oversized datagram must not take the node
// down, and a record the store refused is never confirmed — no Ack and no
// supersession of pending tracking, so the sender keeps retransmitting and
// the unacked gauge keeps climbing: the truthful signal of divergence,
// never a "delivered" report for a record that was never stored. The
// ApplyRemote returns feed the MergeStats counters (OBS-01, D-10): every
// displaced-or-rejected version counts a conflict — counted only, never
// journaled — and a nonzero drift return (the store clock's beyond-threshold
// signal) counts a drift warning.
func (c *Core) applyRemote(now time.Time, source string, msg port.DataReplication, out *port.Outbox) {
	if held, present := c.store.Version(msg.Key); present {
		switch cmp := held.Compare(msg.Record.Version); {
		case cmp > 0:
			// The sender is behind: answer with the held record, never Ack.
			c.enqueueResponse(now, source, egressDataReplication, msg.Key)
			return
		case cmp == 0:
			// Already held at this exact version: Ack and emit no reply.
			out.Unicast(source, port.Ack{Key: msg.Key, Version: held})
			return
		}
	}
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
	// tracking unchanged (REPL-02). Every older tracked version under the
	// key leaves, because the record they tracked no longer exists, and
	// each emits its Superseded result with the TRACKED version — the
	// displaced write's resolution is an honest false (XPORT-12).
	for pk := range c.pending {
		if pk.key == msg.Key && msg.Record.Version.Compare(pk.version) > 0 {
			out.Superseded(pk.key, pk.version)
			c.removePending(pk)
		}
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
// the write-time total (grpcapi's waitForReplication). The empty-table
// guard is load-bearing: an entry whose every waiter was evicted stays
// retained in suspension, and an Ack can only drain when peers actually
// exist. The peer is identified by the datagram's NodeID alone; nothing
// here writes the peer table. Every matching Ack emits an [port.Result] —
// including a stranger's, the count the waiter tracker needs — and a
// drain emits the Drained result at the deletion point (XPORT-12): the
// result stream is how a waiter learns what this call observed, without a
// callback entering the core.
func (c *Core) handleAck(peerID uint16, ack port.Ack, out *port.Outbox) {
	pk := pendingKey{key: ack.Key, version: ack.Version}
	entry, ok := c.pending[pk]
	if !ok {
		return
	}
	delete(entry.waitSet, peerID)
	c.ledgerRemove(peerID, pk)
	entry.acked = ackedInsert(entry.acked, peerID)
	out.Ack(pk.key, pk.version, peerID)
	if len(entry.waitSet) == 0 && len(c.peers) > 0 {
		out.Drained(pk.key, pk.version)
		c.removePending(pk)
		return
	}
	c.pending[pk] = entry
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

// handleDataRequest answers one DataRequest (REPL-04): one O(1)
// Version presence probe serves the whole handler call — the view
// discipline, never a whole-store snapshot and never a Data copy before the
// dedupe check (CR-02) — and a key the store holds (tombstones included, so
// deletes heal through the same LWW path as writes) enqueues one unicast
// DataReplication IDENTITY on the egress queue; the drain re-reads the
// current record and builds the message. A key the store does not hold
// produces nothing at all — no error reply, no nack, nothing enqueued, ever.
// The reply addresses the datagram's source handle regardless of peer-table
// membership: a stranger's DataRequest is still protocol traffic from a live
// node, and replies are unicast only — a broadcast reply would amplify one
// request into N datagrams.
func (c *Core) handleDataRequest(now time.Time, source, key string) {
	if _, ok := c.store.Version(key); !ok {
		return
	}
	c.enqueueResponse(now, source, egressDataReplication, key)
}

// handleDataSync processes one DataSync part (REPL-05, PROTO-01). The header
// names one peer's listing round; parts accumulate by
// (node_id, run_id, sync_id) and the round is complete exactly when every
// chunk index 1..chunk_count has arrived — the first part alone is never the
// listing (the R-03 defect this shape removes). Before accumulation, a
// non-empty part absorbs its maximum entry version into the store clock
// exactly once (PROTO-01): the call sits after header validation — so a
// datagram the validation pipeline discarded never reaches the clock (T-4-10)
// — and is independent of stream completion, so a partial round's versions
// already advance the HLC. An empty part has no version to absorb. Only a
// complete round publishes its key set through retainListedKeys, sets
// listingComplete once, moves the listing generation when the published
// content actually changed, and opens the gate (syncArrived) — all in this
// same handler, so no partial listing can open it. The part's entries then
// compare per key against the store exactly as before: a locally absent key
// or a strictly newer listing version enqueues a unicast DataRequest
// identity; a strictly newer local record — the point Version probe already
// proved presence, and the drain re-reads the current record (tombstones
// included, so a local delete heals the sender's live copy) — enqueues a
// unicast DataReplication identity; equal versions produce silence. Both
// enqueues are identity-only, so no Data is copied and no record is looked
// up before the dedupe check (CR-02).
// Entries are processed in received order (the compare is per key and
// order-independent by construction) and every response flows through the
// single rate-limited egress queue — the scope pin; no reactive unicast has a
// bypass path, which is what survives empty-node recovery (T-04-07). A
// directly-driven core receives no header validation (the wired path
// classifies that before this handler), so an impossible header returns
// without touching any state.
func (c *Core) handleDataSync(now time.Time, in port.InboundDatagram, msg port.DataSync, _ *port.Outbox) {
	if !validSyncHeader(msg) {
		return
	}
	if len(msg.Entries) > 0 {
		c.store.Observe(maxEntryVersion(msg.Entries))
	}
	c.accumulateSync(in, msg)
	for _, entry := range msg.Entries {
		local, present := c.store.Version(entry.Key)
		switch {
		case !present || local.Compare(entry.Version) < 0:
			c.enqueueResponse(now, in.Source, egressDataRequest, entry.Key)
		case local.Compare(entry.Version) > 0:
			// The probe above already proved presence; the drain owns the
			// race between this probe and the record read.
			c.enqueueResponse(now, in.Source, egressDataReplication, entry.Key)
		}
	}
}

// maxEntryVersion returns the greatest version among entries by the full
// 16-byte comparison; entries must be non-empty. The absorbed reading is the
// exact maximum of the part's validated versions — no averaging, rounding or
// truncation (PROTO-01 precision).
func maxEntryVersion(entries []port.SyncEntry) version.RecordVersion {
	maxVer := entries[0].Version
	for _, entry := range entries[1:] {
		if entry.Version.Compare(maxVer) > 0 {
			maxVer = entry.Version
		}
	}
	return maxVer
}

// validSyncHeader reports whether msg's completeness header is inside its
// domain: at least one chunk, a 1-based index inside the round, and a count
// at or under the stream cap (T-4-02). One step outside any bound is
// invalid; an empty listing is a single empty chunk, never chunk_count 0.
func validSyncHeader(msg port.DataSync) bool {
	return msg.ChunkCount >= 1 &&
		msg.ChunkIndex >= 1 &&
		msg.ChunkIndex <= msg.ChunkCount &&
		msg.ChunkCount <= maxSyncChunks
}

// accumulateSync folds one valid part into its round's stream, enforcing the
// three bounds, and completes the listing when every index 1..count is
// present. The stream key includes the peer's run_id, so a restart's fresh
// sync_id counter cannot merge into the previous run's partial state.
func (c *Core) accumulateSync(in port.InboundDatagram, msg port.DataSync) {
	key := syncStreamKey{peer: in.NodeID, run: in.RunID, id: msg.SyncID}
	stream := c.syncs[key]
	if stream == nil {
		c.makeSyncStreamRoom()
		stream = &syncStream{
			count:   msg.ChunkCount,
			got:     make(map[uint32]struct{}, msg.ChunkCount),
			entries: make(map[string]version.RecordVersion, len(msg.Entries)),
		}
		c.syncs[key] = stream
	} else if stream.count != msg.ChunkCount {
		// A part disagreeing with the live round's count never merges: the
		// wired path classifies it as bad_sync_header and the stored count is
		// the round's identity.
		return
	}
	stream.got[msg.ChunkIndex] = struct{}{}
	for _, entry := range msg.Entries {
		if _, seen := stream.entries[entry.Key]; seen {
			continue // idempotent: the same key twice contributes once
		}
		stream.entries[entry.Key] = entry.Version
		c.syncEntries++
	}
	if c.enforceSyncBounds(key) {
		return // the bounds dropped this round; nothing may complete from it
	}
	if !stream.complete() {
		return
	}
	changed := c.retainListedKeys(in.NodeID, stream.entries)
	if !c.listingComplete {
		c.listingComplete = true
		changed = true
	}
	if changed {
		c.listingGen++
	}
	c.syncArrived = true
	c.discardSyncStream(key)
}

// complete reports whether every chunk index 1..count has arrived — the
// single completeness predicate: arrival of any one chunk proves nothing.
func (s *syncStream) complete() bool {
	if len(s.got) < int(s.count) {
		return false
	}
	for i := uint32(1); i <= s.count; i++ {
		if _, ok := s.got[i]; !ok {
			return false
		}
	}
	return true
}

// makeSyncStreamRoom evicts least-progressed rounds until another stream can
// be admitted. Eviction order is deterministic — fewest received chunks
// first, then (peer, run bytes, sync_id) ascending — so same-seed replay
// stays byte-identical (T-4-03, T-3-22).
func (c *Core) makeSyncStreamRoom() {
	for len(c.syncs) >= maxSyncStreams {
		if !c.dropWeakestSyncStream() {
			return
		}
	}
}

// enforceSyncBounds applies the per-stream and total entry caps after one
// accumulation, reporting whether the round was dropped. A stream past
// maxSyncEntriesPerStream is evicted whole; then least-progressed streams are
// evicted until the total fits maxSyncEntriesTotal. Every eviction counts in
// syncStreamDropped.
func (c *Core) enforceSyncBounds(key syncStreamKey) bool {
	stream := c.syncs[key]
	if stream == nil {
		return true
	}
	if len(stream.entries) > maxSyncEntriesPerStream {
		c.evictSyncStream(key)
		return true
	}
	for c.syncEntries > maxSyncEntriesTotal {
		if !c.dropWeakestSyncStream() {
			break
		}
	}
	_, kept := c.syncs[key]
	return !kept
}

// dropWeakestSyncStream evicts the least-progressed round and reports
// whether one existed. The comparator is a total order over the streams, so
// the choice never depends on map iteration order.
func (c *Core) dropWeakestSyncStream() bool {
	var weakest syncStreamKey
	var weakestStream *syncStream
	found := false
	for key, stream := range c.syncs {
		if !found || syncStreamLess(key, stream, weakest, weakestStream) {
			weakest, weakestStream, found = key, stream, true
		}
	}
	if !found {
		return false
	}
	c.evictSyncStream(weakest)
	return true
}

// syncStreamLess orders two partial rounds for deterministic eviction:
// fewer received chunks is weaker (less likely to complete soon), then the
// key's peer, run bytes and sync_id break ties ascending.
func syncStreamLess(a syncStreamKey, aStream *syncStream, b syncStreamKey, bStream *syncStream) bool {
	if len(aStream.got) != len(bStream.got) {
		return len(aStream.got) < len(bStream.got)
	}
	if a.peer != b.peer {
		return a.peer < b.peer
	}
	if cmp := bytes.Compare(a.run[:], b.run[:]); cmp != 0 {
		return cmp < 0
	}
	return a.id < b.id
}

// evictSyncStream drops one partial round and counts the drop — the only
// deletion path that counts; completion uses discardSyncStream.
func (c *Core) evictSyncStream(key syncStreamKey) {
	if _, ok := c.syncs[key]; !ok {
		return
	}
	c.discardSyncStream(key)
	c.syncStreamDropped++
}

// discardSyncStream deletes one partial round and releases its accumulated
// entries from the total counter — the shared completion/eviction deletion.
func (c *Core) discardSyncStream(key syncStreamKey) {
	stream, ok := c.syncs[key]
	if !ok {
		return
	}
	c.syncEntries -= len(stream.entries)
	delete(c.syncs, key)
}

// dropPeerSyncStreams discards every partial round a peer left behind — the
// restart and peer-eviction cleanup, so a dead run's partial state cannot
// occupy the bounded accumulator.
func (c *Core) dropPeerSyncStreams(peer uint16) {
	for key := range c.syncs {
		if key.peer == peer {
			c.discardSyncStream(key)
		}
	}
}

// retainListedKeys installs entries as peer's whole listed-key set, releases
// the set it replaces through the union refcounts, refreshes the peer's LRU
// position, and then evicts whole peer sets from the LRU front while the
// retained total exceeds maxListedKeysTotal (T-4-41). It reports whether the
// UNION membership actually changed — a key entering or leaving the union,
// not merely a peer's own set being rewritten — so a key shared with another
// peer neither enters nor leaves it, and an identical repeated listing moves
// no generation and the adapter's mirror clones nothing. A key shared by
// several peers stays in the union until its last lister drops it; eviction
// therefore shrinks the union, the refcounts and the retained total together.
func (c *Core) retainListedKeys(peer uint16, entries map[string]version.RecordVersion) bool {
	if sameListedKeys(c.listedByPeer[peer], entries) {
		c.touchListedPeer(peer)
		return false
	}
	released := c.dropListedPeer(peer)
	set := make(map[string]struct{}, len(entries))
	added := 0
	for key := range entries {
		set[key] = struct{}{}
		c.listedRefs[key]++
		if c.listedRefs[key] == 1 {
			c.listedKeys[key] = struct{}{}
			c.listedKeyCount++
			added++
		}
	}
	c.listedByPeer[peer] = set
	c.listedPos[peer] = c.listedOrder.PushBack(peer)
	evicted := 0
	for c.listedKeyCount > maxListedKeysTotal {
		front := c.listedOrder.Front()
		if front == nil {
			break
		}
		victim := front.Value.(uint16)
		evicted += c.dropListedPeer(victim)
	}
	c.listedKeysEvicted += uint64(evicted)
	return released > 0 || added > 0 || evicted > 0
}

// touchListedPeer moves peer to the LRU back without touching its set — the
// identical-repeat path: recency updates, content does not.
func (c *Core) touchListedPeer(peer uint16) {
	if elem := c.listedPos[peer]; elem != nil {
		c.listedOrder.MoveToBack(elem)
		return
	}
	c.listedByPeer[peer] = make(map[string]struct{})
	c.listedPos[peer] = c.listedOrder.PushBack(peer)
}

// dropListedPeer removes peer's whole listed-key contribution: every key whose
// last lister it was leaves the union, the refcounts shrink, and the peer row
// and LRU element go. It returns how many keys left the union — the count
// ListedKeysEvicted publishes. Used both for replacement and eviction, so a
// dead peer's keys can never survive in the mirror.
func (c *Core) dropListedPeer(peer uint16) int {
	set, ok := c.listedByPeer[peer]
	if !ok {
		return 0
	}
	released := 0
	for key := range set {
		c.listedRefs[key]--
		if c.listedRefs[key] > 0 {
			continue
		}
		delete(c.listedRefs, key)
		if _, listed := c.listedKeys[key]; listed {
			delete(c.listedKeys, key)
			c.listedKeyCount--
			released++
		}
	}
	delete(c.listedByPeer, peer)
	if elem := c.listedPos[peer]; elem != nil {
		c.listedOrder.Remove(elem)
		delete(c.listedPos, peer)
	}
	return released
}

// sameListedKeys reports whether peer's retained set already holds exactly
// entries' keys — the identical-listing predicate the content-gated
// generation depends on.
func sameListedKeys(old map[string]struct{}, entries map[string]version.RecordVersion) bool {
	if len(old) != len(entries) {
		return false
	}
	for key := range entries {
		if _, ok := old[key]; !ok {
			return false
		}
	}
	return true
}

// enqueueResponse appends one reactive unicast identity to the egress queue
// (REPL-05): the FIFO drains inside Advance at exactly one
// syncResponseInterval per response, enqueue order IS drain order, and the
// first response after an idle queue emits immediately — nextSendAt clamps
// to this call's now, so no head-of-line delay ever applies to the first
// datagram of a burst. The entry carries (handle, kind, key) and no message
// (CR-02): the producer must not look a record up first, so a request for a
// present key copies no Data, and the drain builds the message from the
// current store state. The queue is bounded and deduplicated (XPORT-07): the
// identity is (handle, kind, key), so the same answer requested twice — the
// two-listings-1000-missing-keys case — occupies one slot, and an overflow
// drops the OLDEST entry together with its identity (T-3-21) while counting
// the drop. Duplicates are silent: the existing answer already covers the
// request.
func (c *Core) enqueueResponse(now time.Time, handle string, kind egressKind, key string) {
	id := egressKey{handle: handle, kind: kind, key: key}
	if _, dup := c.egressIDs[id]; dup {
		return
	}
	if len(c.egress) == 0 {
		c.nextSendAt = now
	}
	c.egress = append(c.egress, queuedUnicast{id: id})
	c.egressIDs[id] = struct{}{}
	if len(c.egress) > maxEgressEntries {
		c.dropOldestEgress()
	}
}

// dropOldestEgress evicts the queue's oldest entry and deletes its dedupe
// identity in the same operation: a retained identity would make a later
// identical request look already queued and silently starve it (T-3-21,
// RESEARCH Pitfall 10). The emptied slot is cleared to release its identity
// strings.
func (c *Core) dropOldestEgress() {
	oldest := c.egress[0]
	c.egress[0] = queuedUnicast{}
	c.egress = c.egress[1:]
	delete(c.egressIDs, oldest.id)
	c.egressDropped++
}

// drainEgress emits every due queued response (REPL-05): while nextSendAt
// is at or before now, the FIFO's head emits unicast and the next slot
// moves one syncResponseInterval ahead — the spacing uses no map state and
// no re-sorting; the producers iterate sorted listings or arrival order,
// so enqueue order is already deterministic (probe #9 discipline). An
// Advance that arrives late (past several slots) still emits only one
// response per call of this loop's steady state: the slot advances from
// the actual emission time, never bursting a backlog in one tick — the
// limiter is what survives empty-node recovery (T-04-07). A queued
// DataReplication answer re-reads the stored record at drain time and is
// skipped when the key is gone (the store has changed since enqueue); a
// skipped entry consumes no slot.
func (c *Core) drainEgress(now time.Time, out *port.Outbox) {
	for len(c.egress) > 0 && !now.Before(c.nextSendAt) {
		q := c.egress[0]
		c.egress[0] = queuedUnicast{}
		c.egress = c.egress[1:]
		delete(c.egressIDs, q.id)
		msg, ok := c.resolveQueuedResponse(q)
		if !ok {
			continue
		}
		out.Unicast(q.id.handle, msg)
		c.pacedSends++
		if now.After(c.nextSendAt) {
			c.nextSendAt = now
		}
		c.nextSendAt = c.nextSendAt.Add(c.syncResponseInterval)
	}
}

// resolveQueuedResponse builds the message to emit for one dequeued
// response. A DataRequest is key-only by nature; a DataReplication answer
// re-reads the record through Lookup and reports false when the key is gone
// — the queue pins no Data, so the answer always carries the current record
// (XPORT-07, CR-02).
func (c *Core) resolveQueuedResponse(q queuedUnicast) (port.Message, bool) {
	switch q.id.kind {
	case egressDataReplication:
		rec, ok := c.store.Lookup(q.id.key)
		if !ok {
			return nil, false
		}
		return port.DataReplication{Key: q.id.key, Record: rec}, true
	case egressDataRequest:
		return port.DataRequest{Key: q.id.key}, true
	}
	return nil, false
}

// NotifyLocalWrite is the explicit call a write's author makes after the
// store write (D-05: no store subscriptions, no observers, no callbacks —
// tests now, the gRPC handler in Phase 5). The store is the single record
// source (the write is already installed there), so the core keeps only the
// (key, version) tracking entry — wait-set seeded with every currently-known
// peer, ladder armed at cfg.AckTimeout — and broadcasts the record as a
// DataReplication datagram from a defensive copy (the author may reuse its
// buffer once the call returns).
//
// The notification guard runs first (PROTO-05, D-05): a notification whose
// version is NOT GREATER than the version the core already tracks for that
// key — a regressive write or an equal duplicate — is ignored: no broadcast,
// no ledger entry or other tracking change, and one NotificationsRegressed
// count. The comparison is against the tracked pending version, never
// store.Version: the store already holds the first notified version when
// this call runs, so a <= store comparison would swallow the normal path.
// A key's first notification (no tracked entry) always proceeds.
//
// A newer local write under the same key REPLACES the older tracking: every
// entry with a strictly smaller version leaves, fresh wait-set, ladder reset
// — local supersession (REPL-02). Replication progress is never a
// precondition of the write path: this enqueues-and-returns (D-05).
func (c *Core) NotifyLocalWrite(now time.Time, key string, rec storage.Record, out *port.Outbox) {
	if tracked, ok := c.trackedVersion(key); ok && rec.Version.Compare(tracked) <= 0 {
		c.notificationsRegressed++
		return
	}
	c.removeSupersededLocal(key, rec.Version, out)
	pk := pendingKey{key: key, version: rec.Version}
	_, tracked := c.pending[pk]
	if !tracked {
		waitSet := make(map[uint16]struct{}, len(c.peers))
		for id := range c.peers {
			waitSet[id] = struct{}{}
		}
		entry := pendingEntry{
			version:   rec.Version,
			waitSet:   waitSet,
			nextRetry: now.Add(c.cfg.AckTimeout),
			backoff:   c.cfg.AckTimeout,
			createdAt: now,
		}
		entry.elem = c.insertPendingOrder(pk, now)
		c.pending[pk] = entry
		for id := range waitSet {
			c.ledgerAdd(id, pk)
		}
		c.pushRetry(pk, entry.nextRetry)
	}
	own := storage.Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Deleted:  rec.Deleted,
		Checksum: rec.Checksum,
	}
	out.Broadcast(port.DataReplication{Key: key, Record: own})
}

// Advance fires the due timers — the hello timer, the DataSync startup and
// interval triggers, the paced listing drain, the peer-timeout eviction scan,
// the retransmission scan and its paced record-send drain, and the
// rate-limited egress drain — and returns the
// next deadline for the driver to sleep on: the minimum of the following
// hello deadline, the next sync deadline, the earliest peer expiry, while
// any pending entry exists the retransmit scan deadline (the
// scan-granularity reading of retransmit_interval, research A5), while any
// listing chunk is queued the next listing slot, while any record send is
// queued the next record slot, and while
// any response is queued the next egress slot. The returned deadline is
// always strictly future: work due at now fires inside this same call.
// Eviction runs before the retransmit scan so a peer expiring at now is
// already gone when the scanner asks whether any peer remains; the egress
// drain runs last so a tick's reactive responses follow its proactive
// emissions. Before any timer work it fires the one-shot no-Hello WARN
// ([Core.warnIfNoPeer]) — a diagnostic, never a behavior change.
func (c *Core) Advance(now time.Time, out *port.Outbox) time.Time {
	c.warnIfNoPeer(now)
	if !now.Before(c.nextHello) {
		out.Broadcast(port.Hello{})
		c.nextHello = now.Add(c.cfg.HelloInterval + c.helloJitter())
	}
	c.syncListing(now)
	c.drainListings(now, out)
	c.evictPeers(now, out)
	c.scanRetransmit(now)
	c.drainRecordSends(now, out)
	c.drainEgress(now, out)
	return c.nextDeadline(now)
}

// warnIfNoPeer surfaces a node that has heard no peer since it started
// (XPORT-09, D-07): when no Hello has arrived by firstAdvance +
// cfg.PeerTimeout, one WARN names the condition — the likely causes are a
// bind address that receives no broadcasts, a wrong broadcast address, or a
// silent segment. The warning fires at most once per process lifetime
// (peerWarned latches it) because the deaf state persists and a per-interval
// repeat would be noise, not signal; it never changes serving behavior.
// Counting from the first Advance rather than from construction keeps the
// deadline on the driver's own clock, so a core built long before its loop
// runs does not warn before it ever had a chance to read a datagram.
func (c *Core) warnIfNoPeer(now time.Time) {
	if c.firstAdvance.IsZero() {
		c.firstAdvance = now
	}
	if c.helloSeen || c.peerWarned {
		return
	}
	if now.Before(c.firstAdvance.Add(c.cfg.PeerTimeout)) {
		return
	}
	c.peerWarned = true
	c.cfg.Log(fmt.Sprintf(
		"sup-kv: no Hello received within peer_timeout (%s) of startup: no peer discovered — check the broadcast address and that the bind address receives broadcasts",
		c.cfg.PeerTimeout))
}

// syncListing fires the DataSync full-listing emission (REPL-05): the
// startup trigger on the first Advance — immediately after the startup
// Hello, Hello first, both broadcast, the IDEA §Node lifecycle order
// pinned by TestSync_triggers — and the timer trigger on every subsequent
// Advance at or after lastSync+cfg.SyncInterval plus a positive-only jitter
// of at most a tenth of the interval. The listing comes from the shared
// build ([Core.listingBuild]): one O(store) Versions copy per
// (sync_interval age, store generation) is reused by every trigger while both
// still hold, so a hostile trigger burst can never drive one copy per datagram
// (CR-01, T-4-24). Every round stamps the build's sync_id and chunk_count on
// each chunk; the chunks are ENQUEUED in rotated order — round r starts at
// chunk r % len(chunks) — while each chunk keeps its ORIGINAL 1-based index,
// and the shared listing pacer drains one datagram per
// floor(1s/SyncResponseRate) interval (D-01), so a burst can never overflow
// the receiver's buffer. A store with zero records still emits one DataSync
// with an empty entry list and count 1 — an empty listing is a valid, complete
// sync round that invites newer-flooding from receivers (probe #7). The first-
// seen and restart triggers share the same [Core.enqueueListing] from
// handleHello (REPL-07, PROTO-04). The timer advances even when the enqueue is
// skipped: a trigger arriving while a broadcast listing is in flight must
// never leave nextSync in the past, or every Advance would retrigger and the
// driver would never sleep (RESEARCH Pitfall 3).
func (c *Core) syncListing(now time.Time) {
	if !c.nextSync.IsZero() && now.Before(c.nextSync) {
		return
	}
	c.nextSync = now.Add(c.cfg.SyncInterval + c.syncJitter())
	if c.enqueueListing(now, broadcastDst(), c.syncRotation) {
		c.syncRotation++
	}
}

// enqueueListing enqueues one full listing toward dst, rotated by the caller's
// round counter — the single enqueue function the first-seen Hello, the
// restart branch and the periodic broadcast all share (no second
// implementation). It is a no-op when dst already has a FIFO in flight (one
// in-flight listing per destination, T-3-23), refuses when the queue is at its
// destination or queued-chunk bound (counting the drop, T-4-23), and
// otherwise appends the FIFO with startIndex = rotation % len(chunks). It
// reports whether a FIFO was enqueued; a refused or duplicate trigger advances
// nothing.
func (c *Core) enqueueListing(now time.Time, dst listingDst, rotation uint64) bool {
	if _, busy := c.listingByDst[dst]; busy {
		return false
	}
	if len(c.listingByDst) >= maxListingDestinations {
		c.listingDropped++
		return false
	}
	build := c.listingBuild(now)
	if c.listingCount+len(build.chunks) > maxListingChunks {
		c.listingDropped++
		return false
	}
	//nolint:gosec // G115: the modulus is len(build.chunks), an int, so the result fits.
	start := int(rotation % uint64(len(build.chunks)))
	if c.listingCount == 0 {
		c.nextListingAt = now
	}
	c.listingByDst[dst] = &listingFIFO{build: build, dst: dst, startIndex: start}
	c.listingOrder = append(c.listingOrder, dst)
	c.listingCount += len(build.chunks)
	return true
}

// listingBuild returns the shared listing build for now: a cached build is
// reused only while it is younger than cfg.SyncInterval AND the store's
// content generation is unchanged; otherwise the store is copied and chunked
// once, stamped with a fresh monotone sync_id and the generation it was built
// at. The generation check is what keeps the cache honest: a build is a
// snapshot of the store, so serving it after any mutation would ship stale
// state and defeat PROTO-04's no-sync_interval-term freshness; a mutation
// costs exactly one rebuild before the next trigger, never one per trigger
// (T-4-24).
func (c *Core) listingBuild(now time.Time) *listingBuild {
	gen := c.store.Generation()
	if c.build != nil && now.Sub(c.build.at) < c.cfg.SyncInterval && c.build.generation == gen {
		return c.build
	}
	c.build = &listingBuild{
		chunks:     c.listingChunks(),
		at:         now,
		syncID:     c.nextSyncID,
		generation: gen,
	}
	c.nextSyncID++
	return c.build
}

// dropListing removes dst's in-flight FIFO and releases its remaining queued
// chunk work from listingCount — the peer-eviction and run-change cleanup, so
// stale work cannot outlive the peer row it was built for (CR-01). A
// destination with no FIFO is a no-op.
func (c *Core) dropListing(dst listingDst) {
	q, ok := c.listingByDst[dst]
	if !ok {
		return
	}
	c.listingCount -= len(q.build.chunks) - q.emitted
	delete(c.listingByDst, dst)
	for i, d := range c.listingOrder {
		if d == dst {
			c.listingOrder = append(c.listingOrder[:i], c.listingOrder[i+1:]...)
			break
		}
	}
}

// chunkOrdinal returns the 0-based chunk position as the wire's 1-based
// chunk_index.
func chunkOrdinal(pos int) uint32 {
	return uint32(pos + 1) //nolint:gosec // G115: bounded by maxSyncChunks, the chunker's ceiling.
}

// syncChunkCount returns the round's chunk count as the wire's uint32
// chunk_count; the listing source is the in-memory store and the chunker
// emits at most one chunk per entry, so the value is far below 2^32 for any
// reachable store and within the receiver's maxSyncChunks for the documented
// 20 000-key scale.
func syncChunkCount(chunks [][]port.SyncEntry) uint32 {
	return uint32(len(chunks)) //nolint:gosec // G115: a store cannot hold 2^32 entries in memory.
}

// drainListings emits queued listing chunks at the shared pacer cadence
// (D-01): while a listing slot is due, the FIFO at the head of the
// destination order emits one datagram carrying its shared build's sync_id and
// chunk_count and the chunk at its own rotated position — broadcast for the
// zero destination, unicast at the peer row's current handle otherwise — and
// the slot advances by the jittered interval. Chunk indices are the chunk's
// ORIGINAL 1-based position in the unrotated build, so rotation only reorders
// emission and a receiver can still recognize completeness. The clamp shape is
// drainEgress': an Advance that arrives late emits one datagram, never a
// backlog burst, and the next slot is measured from the actual emission time.
// A destination whose FIFO has emitted every chunk, or whose peer row
// disappeared, leaves the order, so its in-flight mark clears exactly when its
// listing completes (T-3-23).
func (c *Core) drainListings(now time.Time, out *port.Outbox) {
	for c.listingCount > 0 && !now.Before(c.nextListingAt) {
		dst := c.listingOrder[0]
		q := c.listingByDst[dst]
		idx := (q.startIndex + q.emitted) % len(q.build.chunks)
		msg := port.DataSync{
			SyncID:     q.build.syncID,
			ChunkIndex: chunkOrdinal(idx),
			ChunkCount: syncChunkCount(q.build.chunks),
			Entries:    q.build.chunks[idx],
		}
		if dst == broadcastDst() {
			out.Broadcast(msg)
		} else {
			out.Unicast(c.peers[dst.nodeID].handle, msg)
		}
		c.pacedSends++
		q.emitted++
		c.listingCount--
		if q.emitted == len(q.build.chunks) {
			delete(c.listingByDst, dst)
			c.listingOrder = c.listingOrder[1:]
		}
		if now.After(c.nextListingAt) {
			c.nextListingAt = now
		}
		c.listingJitterSeq++
		c.nextListingAt = c.nextListingAt.Add(c.jitteredSpacing(c.syncResponseInterval, c.listingJitterSeq))
	}
}

// jitterOffset returns a deterministic offset in [0, span) derived from the
// core's runID, the site's monotone sequence number and the site salt:
// FNV-1a over the run identity, so identical seeds replay identically and no
// PRNG state enters the core (T-3-22). A non-positive span returns zero.
func (c *Core) jitterOffset(seq, salt uint64, span time.Duration) time.Duration {
	if span <= 0 {
		return 0
	}
	h := fnv.New64a()
	_, _ = h.Write(c.runID[:])
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], seq)
	_, _ = h.Write(buf[:])
	binary.BigEndian.PutUint64(buf[:], salt)
	_, _ = h.Write(buf[:])
	//nolint:gosec // G115: the modulus is a positive Duration, so the result is a valid Duration.
	return time.Duration(h.Sum64() % uint64(span))
}

// helloJitter returns the Hello timer's positive-only jitter in
// [0, HelloInterval/10): the deadline never fires before the configured
// interval.
func (c *Core) helloJitter() time.Duration {
	c.helloJitterSeq++
	return c.jitterOffset(c.helloJitterSeq, helloJitterSalt, c.cfg.HelloInterval/jitterDivisor)
}

// syncJitter returns the sync timer's positive-only jitter in
// [0, SyncInterval/10): the periodic listing never fires before the
// configured interval.
func (c *Core) syncJitter() time.Duration {
	c.syncJitterSeq++
	return c.jitterOffset(c.syncJitterSeq, syncJitterSalt, c.cfg.SyncInterval/jitterDivisor)
}

// jitteredSpacing returns base with a deterministic +-10% offset —
// [-base/10, +base/10] — for the listing and record pacers (D-01/D-05,
// RESEARCH Pattern 10's jitter scope). A computed spacing below one
// nanosecond floors to 1ns, so a pacer can never be silently disabled; the
// retransmit ladder's deadlines never pass through this helper (the loss
// bound depends on them).
func (c *Core) jitteredSpacing(base time.Duration, seq uint64) time.Duration {
	span := base / jitterDivisor
	if span <= 0 {
		return base
	}
	offset := c.jitterOffset(seq, spacingJitterSalt, 2*span+1) - span
	return max(base+offset, time.Nanosecond)
}

// listingChunks copies the store's shallow Versions view into the full
// DataSync listing chunk sequence (REPL-05): every record, tombstones included
// (deletes reconcile through the same listing), sorted by key in byte order and
// packed by the conservative-estimate chunker. It runs only when the shared
// build is (re)built — at most once per (cfg.SyncInterval age, store
// generation) — so every emission trigger, the periodic broadcast and the
// first-seen/restart unicast alike, reads one immutable chunk sequence from
// [Core.listingBuild] (CR-01, T-4-24). Identical store state produces the
// identical chunk sequence: the walk is over the sorted listing, never a map
// (probe #11), and an empty listing yields one empty chunk (probe #7).
func (c *Core) listingChunks() [][]port.SyncEntry {
	versions := c.store.Versions()
	entries := make([]port.SyncEntry, 0, len(versions))
	for _, key := range slices.Sorted(maps.Keys(versions)) {
		entries = append(entries, port.SyncEntry{Key: key, Version: versions[key]})
	}
	return chunkSyncEntries(entries, c.cfg.SyncChunkBytes)
}

// chunkSyncEntries packs the sorted listing into chunks (REPL-05, the A4
// reading): the walk accumulates entries while the conservative estimate
// usedEstimate + entryEstimate(next) stays at most budget — a chunk filled to
// exactly the budget closes (probe #15), the entry that would cross it opens
// the next chunk, and a single entry whose own estimate already exceeds the
// budget ships alone as its own oversized chunk — never dropped, never split.
// Every chunk is independently comparable; completeness is the round header's
// job (sync_id/chunk_index/chunk_count), never a cursor inside the chunks. An
// empty listing yields one empty chunk, so an empty store still announces
// itself (probe #7). Identical store state produces the identical chunk
// sequence — the walk is over the sorted listing, never a map (probe #11).
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
// holds any number of peers (N nodes from day one). Removing a tracked
// entry's LAST waiter emits the WaitSetEmpty result with the tracked
// version — the entry stays suspended, and a waiter learns the honest
// false (an eviction is never an Ack, XPORT-12).
func (c *Core) evictPeers(now time.Time, out *port.Outbox) {
	expired := make([]uint16, 0, len(c.peers))
	for id := range c.peers {
		if !now.Before(c.peers[id].lastHello.Add(c.cfg.PeerTimeout)) {
			expired = append(expired, id)
		}
	}
	for _, id := range expired {
		c.dropListing(listingDst{nodeID: id, runID: c.peers[id].runID})
		delete(c.peers, id)
		// The evicted peer's retained listing and partial rounds go with it:
		// the mirror must never serve a dead peer's keys, and the union must
		// not retain them (PROTO-01, T-4-41). A union change moves the
		// generation so the publication follows.
		if c.dropListedPeer(id) > 0 {
			c.listingGen++
		}
		c.dropPeerSyncStreams(id)
		// The per-peer ledger bounds the walk: only this peer's unacked
		// entries lose it as a waiter. Entries stay tracked (suspension),
		// and acked stays untouched (an eviction is never an Ack).
		for pk := range c.peerUnacked[id] {
			entry, ok := c.pending[pk]
			if !ok {
				continue
			}
			if _, waiting := entry.waitSet[id]; !waiting {
				continue
			}
			delete(entry.waitSet, id)
			if len(entry.waitSet) == 0 {
				out.WaitSetEmpty(pk.key, pk.version)
			}
			c.pending[pk] = entry
		}
		delete(c.peerUnacked, id)
	}
}

// scanRetransmit enqueues every due pending entry's retransmission on the
// paced record-send queue (REPL-02, D-05).
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
// The ladder is armed at EMISSION, never at enqueue: the scan only queues a
// send, and drainRecordSends re-arms the entry when the pacer actually
// emits it (RESEARCH Pitfall 4) — arming at enqueue would make every
// backed-up entry overdue the moment the pacer reached it, restoring the
// R-02 storm. Firing is one broadcast DataReplication per entry: every peer
// shares each
// send, and peers that already Acked re-merge idempotently and re-Ack (the
// always-Ack-on-duplicate truth of plan 04-01). retransmit_interval is the
// scan granularity — while any entry is pending, Advance's returned
// deadline is never more than cfg.RetransmitInterval past now, so a due
// retry is honored at scan granularity (research A5); with no pending
// entries this scanner contributes no deadline. Only due heap items are
// touched — an Advance with nothing due reads the heap root and nothing
// else, so the scan cost is flat in the pending-set size (XPORT-02). Due
// emissions are ordered by (key, version) — no map-iteration or heap-order
// leak reaches an emit path (REPL-02, research Pitfall 1 / probe #9). Every
// retry emission counts in the retransmissions counter (OBS-01) — the
// adapter mirrors it post-event.
func (c *Core) scanRetransmit(now time.Time) {
	if len(c.peers) == 0 || len(c.pending) == 0 {
		return
	}
	due := c.popDueRetries(now)
	slices.SortFunc(due, compareRetryItems)
	for _, item := range due {
		c.enqueueRecordSend(now, item.pk, true)
	}
}

// compareRetryItems orders due items by key then version bytes — the sorted
// emission discipline shared with every other emit path.
func compareRetryItems(a, b retryItem) int {
	return comparePendingKeys(a.pk, b.pk)
}

// enqueueRecordSend appends one record emission to the paced queue unless
// the exact (key, version) identity is already queued (XPORT-02's dedupe):
// the discovery flush's first send and a ladder retry for the same tracked
// pair collapse into one datagram, so a slow pacer can never multiply the
// storm. The first send after an idle queue clamps nextRecordAt to now —
// the first send is immediate when nothing is ahead of it.
func (c *Core) enqueueRecordSend(now time.Time, pk pendingKey, retry bool) {
	if _, dup := c.recordQueued[pk]; dup {
		return
	}
	if len(c.recordQueue) == 0 {
		c.nextRecordAt = now
	}
	c.recordQueue = append(c.recordQueue, recordSend{pk: pk, retry: retry})
	c.recordQueued[pk] = struct{}{}
}

// drainRecordSends emits queued record sends at the shared pacer cadence
// (D-05): while a record slot is due, the FIFO's head resolves and emits —
// at most one datagram per call, measured from the actual emission time —
// and the slot advances by the jittered interval. The emitted entry's
// ladder is re-armed here, from this emission time (RESEARCH Pitfall 4);
// the queue never pins a record, because the lookup happens now.
func (c *Core) drainRecordSends(now time.Time, out *port.Outbox) {
	for len(c.recordQueue) > 0 && !now.Before(c.nextRecordAt) {
		rs := c.recordQueue[0]
		c.recordQueue[0] = recordSend{}
		c.recordQueue = c.recordQueue[1:]
		delete(c.recordQueued, rs.pk)
		if !c.emitRecordSend(now, rs, out) {
			continue
		}
		if now.After(c.nextRecordAt) {
			c.nextRecordAt = now
		}
		c.recordJitterSeq++
		c.nextRecordAt = c.nextRecordAt.Add(c.jitteredSpacing(c.syncResponseInterval, c.recordJitterSeq))
	}
}

// emitRecordSend resolves and emits one queued record send, reporting
// whether a datagram left. A pending entry that drained or was superseded
// after enqueue emits nothing (RESEARCH Pitfall 5). A store miss or a
// version mismatch emits nothing either: a retry removes the untrustworthy
// entry (the stale-entry cleanup the retransmit scan owns); the discovery
// flush's first send leaves the entry tracked, so the next ladder due time
// performs the removal — the flush's wait-set half is unconditional. A
// resolved send broadcasts the current record, counts a retry, doubles the
// entry's backoff up to AckTimeoutMax, and arms nextRetry from this
// emission time ([armRetry]).
func (c *Core) emitRecordSend(now time.Time, rs recordSend, out *port.Outbox) bool {
	entry, ok := c.pending[rs.pk]
	if !ok {
		return false
	}
	rec, ok := c.store.Lookup(rs.pk.key)
	if !ok || rec.Version != rs.pk.version {
		if rs.retry {
			c.removePending(rs.pk)
		}
		return false
	}
	out.Broadcast(port.DataReplication{Key: rs.pk.key, Record: rec})
	c.pacedSends++
	if rs.retry {
		c.retransmitted++
		entry.backoff = min(backoffGrowth*entry.backoff, c.cfg.AckTimeoutMax)
	}
	entry.nextRetry = now.Add(entry.backoff)
	c.pending[rs.pk] = entry
	c.armRetry(rs.pk, entry.nextRetry)
	return true
}

// nextDeadline returns the earliest strictly-future deadline over the armed
// timers: the hello timer, the sync timer, the earliest peer expiry, while
// any pending entry exists the retransmit scan deadline, while any listing
// chunk is queued the next listing slot, while any record send is queued the
// next record slot, and while
// any response is queued the next egress slot. After syncListing,
// drainListings, evictPeers, scanRetransmit, drainRecordSends and
// drainEgress have fired everything due, every
// contribution is strictly future.
func (c *Core) nextDeadline(now time.Time) time.Time {
	deadline := c.nextHello
	if !c.nextSync.IsZero() && c.nextSync.Before(deadline) {
		deadline = c.nextSync
	}
	if c.listingCount > 0 && c.nextListingAt.Before(deadline) {
		deadline = c.nextListingAt
	}
	if len(c.pending) > 0 {
		if scan := c.retransmitDeadline(now); scan.Before(deadline) {
			deadline = scan
		}
	}
	if len(c.recordQueue) > 0 && c.nextRecordAt.Before(deadline) {
		deadline = c.nextRecordAt
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
// by the due-heap root when the scanner is armed. During suspension (an
// empty peer table) the ladder is paused — its stale retry times must not
// leak into the deadline — so the ceiling alone bounds the sleep.
func (c *Core) retransmitDeadline(now time.Time) time.Time {
	ceiling := now.Add(c.cfg.RetransmitInterval)
	if len(c.peers) == 0 {
		return ceiling
	}
	if root := c.retryRoot(); !root.IsZero() && root.Before(ceiling) {
		return root
	}
	return ceiling
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

// NotificationsRegressed returns the number of local-write notifications the
// not-greater-than-tracked guard ignored (PROTO-05, D-05) — both regressive
// notifications (an older version) and equal duplicate notifications of the
// tracked version. A value read like the discard counters; Phase 7's OBSV
// surface owns publishing it (no adapter mirror this phase).
func (c *Core) NotificationsRegressed() uint64 {
	return c.notificationsRegressed
}

// SyncArrived reports whether a COMPLETE listing has been processed — every
// chunk of some peer's round received, not merely the first arrival (the
// startup gate's sync half, API-04, PROTO-01). The flag flips in
// handleDataSync's all-chunks branch and never resets; ListingComplete is the
// same fact read from the listing state. Set on the driver goroutine only;
// like Discards this is a value read sanctioned for the adapter's post-event
// mirroring, so cross-goroutine readers (Phase 5's gate resolver) see the
// state through the adapter's one atomic listing pointer, never by calling
// into a core the driver owns.
func (c *Core) SyncArrived() bool {
	return c.syncArrived
}

// ListingGeneration returns the listing content version the adapter's mirror
// compares against: it moves exactly when the retained union membership or
// the listing-complete flag actually changes, so an identical repeated
// complete listing moves nothing and triggers no downstream publication
// (PROTO-01, T-4-05). A value read the adapter mirrors from its event loop.
func (c *Core) ListingGeneration() uint64 {
	return c.listingGen
}

// ListingComplete reports whether a complete listing has been processed —
// every chunk of some peer's round received. It flips on the first complete
// listing and never resets; the startup gate's sync half reads it through the
// adapter's mirror.
func (c *Core) ListingComplete() bool {
	return c.listingComplete
}

// ListedKeysSnapshot returns a defensive clone of the retained listed-key
// union — every key named by some peer's last complete listing. The union is
// maintained incrementally by the refcounts, never recomputed from the
// per-peer sets, so the read is O(keys) at listing cadence only; the clone
// shares no backing map with core state (the storage.Records copying
// precedent).
func (c *Core) ListedKeysSnapshot() map[string]struct{} {
	out := make(map[string]struct{}, len(c.listedKeys))
	for key := range c.listedKeys {
		out[key] = struct{}{}
	}
	return out
}

// SyncStreamDropped returns the number of partial-listing streams the bounds
// evicted — the T-4-03 observability signal: a growing value means a peer is
// flooding distinct sync_ids with missing chunks.
func (c *Core) SyncStreamDropped() uint64 {
	return c.syncStreamDropped
}

// ListedKeysEvicted returns the number of retained listed keys released by
// whole-peer-set eviction (T-4-41) — each key removed from the union by the
// maxListedKeysTotal bound increments it once.
func (c *Core) ListedKeysEvicted() uint64 {
	return c.listedKeysEvicted
}

// PeerSnapshot returns a defensive copy of the peer table keyed by NodeID
// (the storage.Records copying precedent) — the peer-table read seam
// consumed by the later Phase 4 plans (retransmission fan-out, peer
// timeouts, sync targets) and observability, without exposing core state.
func (c *Core) PeerSnapshot() map[uint16]PeerInfo {
	out := make(map[uint16]PeerInfo, len(c.peers))
	for id := range c.peers {
		p := c.peers[id]
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
// core state (the storage.Records defensive-copy precedent). Two versions
// under one key can coexist transiently (a stale smaller local write beside
// a newer tracked one); the snapshot reports the greatest version under each
// key, the one the store's LWW join would keep, so the public keyed-by-key
// contract stays single-valued.
func (c *Core) PendingSnapshot() map[string]PendingInfo {
	out := make(map[string]PendingInfo, len(c.pending))
	for _, pk := range sortedPendingKeys(c.pending) {
		entry := c.pending[pk]
		out[pk.key] = PendingInfo{
			Version:   pk.version,
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
// queue is empty has finished answering its sync round. The meaning is the
// egress queue ONLY; QueuedSends is the total-queue accessor quiescence
// needs now that listings and record sends have their own pacers
// (RESEARCH Pitfall 23).
func (c *Core) QueuedResponses() int {
	return len(c.egress)
}

// QueuedSends returns the total number of datagrams waiting in every core
// queue — listing chunks, record sends and egress responses — the
// total-queue accessor the simulator's quiescence predicate consults: a
// scenario must never declare quiescence while an unsent datagram is
// queued (RESEARCH Pitfall 23, T-3-20).
func (c *Core) QueuedSends() int {
	return c.listingCount + len(c.recordQueue) + len(c.egress)
}

// EgressDropped returns the number of egress entries evicted by the
// maxEgressEntries cap (XPORT-07) — the observability seam for the drop
// policy: every eviction is counted, and the evicted identity is re-queueable
// (T-3-21).
func (c *Core) EgressDropped() uint64 {
	return c.egressDropped
}

// ListingDropped returns the number of listing triggers the destination or
// queued-chunk cap refused (CR-01, T-4-23) — the observability seam for the
// listing drop policy, mirroring EgressDropped: every refused first-seen or
// restart listing is counted once, and a refused trigger leaves no FIFO
// behind.
func (c *Core) ListingDropped() uint64 {
	return c.listingDropped
}

// PacedSends returns the number of datagrams the listing, record and egress
// pacers have handed to the Outbox — the emission-volume seam the metrics
// surface publishes as supkv_replication_datagrams_paced_total. The count is
// taken at the Outbox handoff, never at enqueue (T-3-41): a queued datagram
// has not been sent, so an enqueue-counting counter would publish the
// RPL-18 lie this plan deliberately does not compound. A value read the
// production adapter mirrors post-event like the discard counters.
func (c *Core) PacedSends() uint64 {
	return c.pacedSends
}

// PeerCount returns the number of known peers — the O(1) peer-table gauge
// the production adapter's per-event mirror reads (RESEARCH Pitfall 8: the
// copying PeerSnapshot allocates a whole map per call and is fit for tests
// and observability, never the hot mirror).
func (c *Core) PeerCount() int {
	return len(c.peers)
}

// UnackedStats returns the pending-set gauge observed at now: the unacked
// entry count and, with at least one entry, now minus the oldest entry's
// creation time. Both halves are O(1): the count is the map length and the
// oldest entry is the front of the createdAt-ordered pending list — the
// accessor never walks the set (RESEARCH Pitfall 7). A pure read; the wall
// clock enters through the now parameter (the D-02 seam — the core never
// reads a clock itself).
func (c *Core) UnackedStats(now time.Time) UnackedStats {
	stats := UnackedStats{Count: len(c.pending)}
	front := c.pendingOrder.Front()
	if front == nil {
		return stats
	}
	ref := front.Value.(pendingRef)
	stats.OldestAge = now.Sub(ref.createdAt)
	return stats
}

// sortedPendingKeys returns every pending pair ordered by key then version
// bytes — the deterministic iteration order the emission and snapshot paths
// share (probe #9: no map-iteration order reaches a read or emit path). The
// slice is fresh; the pending set is never exposed.
func sortedPendingKeys(pending map[pendingKey]pendingEntry) []pendingKey {
	keys := make([]pendingKey, 0, len(pending))
	for pk := range pending {
		keys = append(keys, pk)
	}
	slices.SortFunc(keys, comparePendingKeys)
	return keys
}

// comparePendingKeys orders two pending pairs by key in byte order, then by
// their record-version bytes — the total order sortedPendingKeys and
// pendingOrder use.
func comparePendingKeys(a, b pendingKey) int {
	if cmp := strings.Compare(a.key, b.key); cmp != 0 {
		return cmp
	}
	return bytes.Compare(a.version[:], b.version[:])
}

// insertPendingOrder links pk into the createdAt-ordered pending list and
// returns its node. The driver's now is non-decreasing, so the append is the
// steady-state O(1) path; an out-of-order creation time (a test or a
// recovered writer) inserts at its sorted position instead of corrupting the
// oldest-age gauge.
func (c *Core) insertPendingOrder(pk pendingKey, createdAt time.Time) *list.Element {
	if back := c.pendingOrder.Back(); back == nil || !createdAt.Before(back.Value.(pendingRef).createdAt) {
		return c.pendingOrder.PushBack(pendingRef{pk: pk, createdAt: createdAt})
	}
	for e := c.pendingOrder.Back(); e != nil; e = e.Prev() {
		if !createdAt.Before(e.Value.(pendingRef).createdAt) {
			return c.pendingOrder.InsertAfter(pendingRef{pk: pk, createdAt: createdAt}, e)
		}
	}
	return c.pendingOrder.PushFront(pendingRef{pk: pk, createdAt: createdAt})
}

// removePending deletes one tracked pair from every structure that holds it
// — the pending map, the createdAt-ordered list and every still-waiting
// peer's ledger set. The pointed-to deletion is shared by the drain,
// supersession and stale-record paths, so no path can leak a list node or a
// ledger entry (T-3-14).
func (c *Core) removePending(pk pendingKey) {
	entry, ok := c.pending[pk]
	if !ok {
		return
	}
	delete(c.pending, pk)
	if entry.elem != nil {
		c.pendingOrder.Remove(entry.elem)
	}
	c.removeRetry(pk)
	for id := range entry.waitSet {
		c.ledgerRemove(id, pk)
	}
}

// trackedVersion returns the version currently tracked for key, if any.
// The notification guard keeps the pending set at most one pair per key: a
// not-greater notification never creates a second entry, and a newer one
// first removes every smaller pair (removeSupersededLocal). The scan is over
// the pending set, never the store — see NotifyLocalWrite for why.
func (c *Core) trackedVersion(key string) (version.RecordVersion, bool) {
	for pk := range c.pending {
		if pk.key == key {
			return pk.version, true
		}
	}
	return version.RecordVersion{}, false
}

// removeSupersededLocal deletes every tracked pair under key whose version is
// strictly smaller than newer — the local-write supersession rule
// (REPL-02). An equal version is not smaller, so an equal-version duplicate
// notification leaves tracking untouched. Each displaced pair emits its
// Superseded result with the TRACKED (older) version — a waiter for that
// write learns its version lost to a newer local write (XPORT-12).
func (c *Core) removeSupersededLocal(key string, newer version.RecordVersion, out *port.Outbox) {
	for pk := range c.pending {
		if pk.key == key && pk.version.Compare(newer) < 0 {
			out.Superseded(pk.key, pk.version)
			c.removePending(pk)
		}
	}
}

// ledgerAdd records that peerID is waiting on pk — the per-peer index behind
// O(peer unacked) eviction.
func (c *Core) ledgerAdd(peerID uint16, pk pendingKey) {
	set := c.peerUnacked[peerID]
	if set == nil {
		set = make(map[pendingKey]struct{})
		c.peerUnacked[peerID] = set
	}
	set[pk] = struct{}{}
}

// ledgerRemove drops pk from peerID's ledger: the peer Acked it, or the
// entry left tracking.
func (c *Core) ledgerRemove(peerID uint16, pk pendingKey) {
	if set := c.peerUnacked[peerID]; set != nil {
		delete(set, pk)
	}
}

// pushRetry arms pk's deadline in the due-heap with the next sequence
// number — the equal-deadline tie-break order of first insertion.
func (c *Core) pushRetry(pk pendingKey, nextRetry time.Time) {
	c.retrySeq++
	heap.Push(&c.retry, retryItem{nextRetry: nextRetry, seq: c.retrySeq, pk: pk})
}

// fixRetry re-arms an already tracked pair's deadline in place (O(log n));
// the pair keeps its original sequence, so re-arms never reorder equal
// deadlines.
func (c *Core) fixRetry(pk pendingKey, nextRetry time.Time) {
	idx, ok := c.retry.index[pk]
	if !ok {
		return
	}
	c.retry.items[idx].nextRetry = nextRetry
	heap.Fix(&c.retry, idx)
}

// armRetry sets pk's retry deadline, pushing when the pair has no heap item
// and fixing it in place otherwise. The distinction matters for the record
// queue: a retry send was popped from the heap by the scan and must be
// re-pushed at emission, while a discovery-flush send still owns its heap
// item and must only be re-timed (a second push would leave a duplicate
// item behind).
func (c *Core) armRetry(pk pendingKey, nextRetry time.Time) {
	if _, ok := c.retry.index[pk]; ok {
		c.fixRetry(pk, nextRetry)
		return
	}
	c.pushRetry(pk, nextRetry)
}

// removeRetry deletes pk's due-heap item (O(log n)), leaving no stale entry
// for the scan or the deadline read to trip over.
func (c *Core) removeRetry(pk pendingKey) {
	idx, ok := c.retry.index[pk]
	if !ok {
		return
	}
	heap.Remove(&c.retry, idx)
}

// retryRoot returns the earliest armed deadline, the zero time when the heap
// is empty — the O(1) read behind nextDeadline.
func (c *Core) retryRoot() time.Time {
	if len(c.retry.items) == 0 {
		return time.Time{}
	}
	return c.retry.items[0].nextRetry
}

// popDueRetries removes and returns every heap item due at now — the only
// entries Advance touches, regardless of pending-set size. The caller owns
// the returned slice; popped items are absent from the heap index already.
func (c *Core) popDueRetries(now time.Time) []retryItem {
	var due []retryItem
	for len(c.retry.items) > 0 && !c.retry.items[0].nextRetry.After(now) {
		due = append(due, heap.Pop(&c.retry).(retryItem))
	}
	return due
}
