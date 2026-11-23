package udprepl

import (
	"slices"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
)

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
// It is the observability gauge (the metrics surface); the write path uses
// [Adapter.KnownPeerIDs] instead, because the wait report needs the ids, not
// the count.
func (a *Adapter) KnownPeers() uint64 {
	return a.peerCount.Load()
}

// KnownPeerIDs returns the known peer ids mirrored from the core's peer table
// after the latest event-loop iteration, sorted ascending — the write-time
// peer capture seam (D-01, A-01). The gRPC handler calls it exactly once
// immediately before an engine write and pins the wait report to that set, so
// a peer discovered after acceptance can neither extend the awaited set nor
// move the reported total. A nil/empty result means zero known peers. The
// returned slice is a clone the caller owns; safe from any goroutine.
func (a *Adapter) KnownPeerIDs() []uint16 {
	ids := a.peerIDs.Load()
	if ids == nil {
		return nil
	}
	return slices.Clone(*ids)
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

// SentByType is the alias of the core package's send-type value: the
// vocabulary moved to internal/app/repl so the metrics exporter reads it
// without importing this socket adapter (D-07). The alias — distinct from
// the accessor method below — keeps every existing udprepl test compiling
// unchanged.
type SentByType = repl.SentByType

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

// mirrorStats copies the core's discard counters, known-peer count and
// peer-id slice, the listing mirror, the OBS-01 gauge set — the pending-set
// gauge, the remote-merge counters and the retransmissions counter — and the
// XPORT-07 emission set (total queue depth, egress drops, paced sends) into
// the atomics after each loop iteration (D-04): cross-goroutine readers — the
// smoke test, the write-time capture, later observability — never touch core
// state. The listing and peer-id mirrors are published only when their
// generation moved, so their clone cost is paid at change cadence, never per
// event.
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
	// The peer-id mirror is generation-gated like the listing mirror: the
	// core bumps its peer generation only when an id enters or leaves the
	// table, so an unchanged table pays no clone. The slice is captured whole
	// and published by ONE pointer store, so a handler reading it observes a
	// complete, sorted write-time set (D-01, A-01).
	if gen := a.core.PeerGeneration(); gen != a.peerGen {
		a.peerGen = gen
		ids := a.core.PeerIDs()
		a.peerIDs.Store(&ids)
	}
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
