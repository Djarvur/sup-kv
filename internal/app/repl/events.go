package repl

import (
	"fmt"
	"time"
)

// PeerEventKind names one peer-table lifecycle transition (D-01, OBSV-01).
// The zero value names no transition, so an uninitialized [PeerEvent] can
// never be mistaken for a real one.
type PeerEventKind uint8

const (
	// PeerAppeared is the first-seen identity path: an unknown node_id was
	// learned and rowed into the peer table, and the row is owed one
	// targeted listing.
	PeerAppeared PeerEventKind = iota + 1

	// PeerRestarted is a remembered node_id announcing a run_id this row has
	// not carried before: the outgoing run joins the row's bounded history
	// and the row moves to the new identity.
	PeerRestarted

	// PeerEvicted is a peer that stayed silent for cfg.PeerTimeout: the row,
	// its retained listing and its pending wait-set memberships are dropped.
	PeerEvicted
)

// PeerEvent is one peer-table lifecycle transition crossing the core boundary
// as a typed value (D-01): the core is the pure decision layer and never
// renders, so main turns each event into exactly one slog record. The
// callback runs on the single driver goroutine and must not block.
//
// Handle is the datagram source's opaque handle, meant for a quoted log
// attribute and never message text (T-07-20). RunID and PreviousRunID are the
// raw 16-byte run identifiers (main renders them as lowercase hex).
// LastSeenAge is set only on PeerEvicted and is the event-time silence — now
// minus the row's last Hello — never a wall-clock read.
type PeerEvent struct {
	Kind          PeerEventKind
	NodeID        uint16
	Handle        string
	RunID         [16]byte
	PreviousRunID [16]byte
	LastSeenAge   time.Duration
}

// DriftEvent is one clock-drift warning crossing the core boundary (D-01,
// D-02): a received record version, or a listing part's maximum entry
// version, led local wall time by more than the configured drift_warn
// threshold. Source is the datagram's quoted source handle and Lead the
// measured divergence. The [MergeStats.DriftWarnings] counter keeps the
// per-offending-record truth; this callback is admitted by the shared
// per-(node_id, clock_drift) limiter, so a peer flooding leading versions
// cannot flood the log (T-07-22).
type DriftEvent struct {
	NodeID uint16
	Source string
	Lead   time.Duration
}

// emitPeer reports ev through the nil-safe OnPeerEvent seam. Every call sits
// on the single driver goroutine, so the callback needs no synchronization
// and must return immediately (the OnDumpFailure contract).
func (c *Core) emitPeer(ev PeerEvent) {
	if c.cfg.OnPeerEvent == nil {
		return
	}
	c.cfg.OnPeerEvent(ev)
}

// emitDrift reports one clock-drift warning through the shared
// per-(node_id, clock_drift) limiter: [MergeStats.DriftWarnings] has already
// counted every offending record, while the WARN/event is admitted once per
// peer per interval (T-07-22). A nil OnDrift falls back to the prebuilt Log
// string, so harnesses that capture only Log keep observing drift.
func (c *Core) emitDrift(now time.Time, nodeID uint16, source string, lead time.Duration) {
	if !c.allowWarn(now, nodeID, warnClassClockDrift) {
		return
	}
	if c.cfg.OnDrift != nil {
		c.cfg.OnDrift(DriftEvent{NodeID: nodeID, Source: source, Lead: lead})
		return
	}
	c.cfg.Log(fmt.Sprintf(
		"sup-kv: clock drift: node %d at handle %q leads local wall time by %s", nodeID, source, lead))
}
