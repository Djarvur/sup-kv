package repl_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/port"
)

// Shared peer-table test constants: the design-default peer liveness bound
// (docs/IDEA.md §Parameters) whose boundary TestPeerTable_lifecycle pins on
// both sides, and the third lifecycle peer (peerID1/peerID2 and the write
// keys are shared from retransmit_test.go — same test package).
const (
	peerTimeout = 5 * time.Second
	peerID3     = uint16(4)
)

// TestPeerTable_lifecycle proves the peer-table lifecycle (REPL-06, probe
// #18 pinned on both sides): a peer is known after its first Hello, only a
// Hello refreshes liveness, the first Advance at or after
// lastHello+peer_timeout evicts — one tick before it nothing is evicted —
// and eviction removes the peer from every pending wait-set without
// draining the entry. The evicted peer's late Ack is a no-op and its next
// Hello re-learns it like a first Hello with no blacklist.
func TestPeerTable_lifecycle(t *testing.T) {
	t.Parallel()
	t.Helper()
	now0 := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now0, &out) // startup Hello (plan 04-01)
	out.Drain()

	known := []uint16{peerID1, peerID2, peerID3}
	for _, id := range known {
		core.HandleDatagram(now0, helloFrom(id), &out)
	}
	out.Drain()
	if got := core.PeerSnapshot(); len(got) != len(known) {
		t.Fatalf("PeerSnapshot holds %d peers after three Hellos, want %d", len(got), len(known))
	}

	ver := localWrite(t, core, st, now0, keyAlpha, "one")
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, known)

	// One nanosecond before the boundary nobody is evicted (probe #18's
	// early side; wall-nanosecond math on a UnixMilli base).
	core.Advance(at(peerTimeout-1), &out)
	out.Drain()
	if got := core.PeerSnapshot(); len(got) != len(known) {
		t.Fatalf("PeerSnapshot holds %d peers one tick before peer_timeout, want %d", len(got), len(known))
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, known)

	// peerID3 goes silent; peerID1 and peerID2 keep Helloing every second.
	for off := time.Second; off < peerTimeout; off += time.Second {
		core.HandleDatagram(at(off), helloFrom(peerID1), &out)
		core.HandleDatagram(at(off), helloFrom(peerID2), &out)
		core.Advance(at(off), &out)
		out.Drain()
	}

	// Exactly at lastHello+peer_timeout the silent peer is gone, the other
	// two remain, and the pending entry keeps tracking minus the evictee.
	core.Advance(at(peerTimeout), &out)
	out.Drain()
	peers := core.PeerSnapshot()
	if _, ok := peers[peerID3]; ok {
		t.Fatal("silent peer still in PeerSnapshot at exactly lastHello+peer_timeout")
	}
	if _, ok := peers[peerID1]; !ok {
		t.Fatal("peerID1 missing from PeerSnapshot at the eviction boundary")
	}
	if _, ok := peers[peerID2]; !ok {
		t.Fatal("peerID2 missing from PeerSnapshot at the eviction boundary")
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})

	// The evicted peer's late Ack is a silent no-op (unknown-peer path).
	core.HandleDatagram(at(peerTimeout), ackFrom(peerID3, keyAlpha, ver), &out)
	out.Drain()
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})

	// Its next Hello re-learns it exactly like a first Hello — no blacklist.
	// The table was not empty, so the flush rule does not apply and the
	// wait-sets gain nothing (only discovery adds waiters).
	core.HandleDatagram(at(peerTimeout+100*time.Millisecond), helloFrom(peerID3), &out)
	out.Drain()
	if _, ok := core.PeerSnapshot()[peerID3]; !ok {
		t.Fatal("evicted peer not re-learned by its next Hello (blacklist state?)")
	}
	if got := broadcastRepls(out.Drain()); len(got) != 0 {
		t.Fatalf("re-Hello into a non-empty table flushed pending entries: %v", replKeys(got))
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})
}

// TestPeerTable_nPeers proves per-peer ack tracking over an N-peer table
// (REPL-06, N nodes from day one): five peers know the write, five Acks
// arrive at five different times in a scrambled order, WaitingOn shrinks by
// exactly the Acking peer each time and the entry drains on the fifth.
func TestPeerTable_nPeers(t *testing.T) {
	t.Parallel()
	t.Helper()
	now0 := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now0, &out)
	out.Drain()

	ids := []uint16{2, 3, peerID3, 5, 6}
	for _, id := range ids {
		core.HandleDatagram(now0, helloFrom(id), &out)
	}
	out.Drain()
	ver := localWrite(t, core, st, now0, keyAlpha, "one")
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, ids)

	remaining := make(map[uint16]struct{}, len(ids))
	for _, id := range ids {
		remaining[id] = struct{}{}
	}
	ackOrder := []uint16{peerID3, 2, 6, 3, 5} // scrambled on purpose.
	for i, id := range ackOrder {
		core.HandleDatagram(at(time.Duration(i+1)*50*time.Millisecond), ackFrom(id, keyAlpha, ver), &out)
		out.Drain()
		delete(remaining, id)
		if len(remaining) == 0 {
			if _, ok := core.PendingSnapshot()[keyAlpha]; ok {
				t.Fatal("pending entry retained after the fifth and last Ack")
			}
			continue
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, ver, slices.Sorted(maps.Keys(remaining)))
	}
}

// TestPeerTable_solePeer proves both single-peer suspension paths (REPL-02,
// REPL-06): a node that never receives any Hello retains its pending
// entries indefinitely with zero emissions, and the sole known peer's
// eviction empties the table, empties the entry's wait-set without draining
// it and stops retransmission — until the peer's later Hello flushes the
// entry again and its Ack drains it.
func TestPeerTable_solePeer(t *testing.T) {
	t.Parallel()
	t.Run("a node that never receives any Hello retains entries indefinitely", func(t *testing.T) {
		t.Parallel()
		t.Helper()
		now0 := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now0, &out)
		out.Drain()
		ver := localWrite(t, core, st, now0, keyAlpha, "one")

		for off := time.Second; off <= suspendWindow; off += time.Second {
			core.Advance(at(off), &out)
			if got := broadcastRepls(out.Drain()); len(got) != 0 {
				t.Fatalf("N=1 cluster with no peer ever known emitted DataReplication for %v at t0+%v, want none",
					replKeys(got), off)
			}
		}
		if got := len(core.PeerSnapshot()); got != 0 {
			t.Fatalf("PeerSnapshot holds %d peers, want 0 (no Hello ever arrived)", got)
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, ver, nil)
		if got := core.UnackedStats(at(suspendWindow)); got.Count != 1 {
			t.Fatalf("UnackedStats().Count = %d after 60 virtual seconds of suspension, want 1", got.Count)
		}
	})
	t.Run("the sole peer's eviction suspends; its return Hello flushes and its Ack drains", func(t *testing.T) {
		t.Parallel()
		t.Helper()
		now0 := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now0, &out)
		out.Drain()
		core.HandleDatagram(now0, helloFrom(peerID1), &out)
		out.Drain()
		ver := localWrite(t, core, st, now0, keyAlpha, "one")
		assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1})

		// While the sole peer is known, the ladder fires.
		core.Advance(at(ackTimeout), &out)
		if got := broadcastRepls(out.Drain()); len(got) != 1 || got[0].Key != keyAlpha {
			t.Fatalf("first ladder step with a known sole peer emitted %v, want one DataReplication for %s", got, keyAlpha)
		}

		// The peer goes silent: at lastHello+peer_timeout the table empties,
		// the entry's wait-set empties with it, and the entry is retained
		// (eviction never drains) — no emission at the eviction Advance even
		// though later ladder steps came due meanwhile.
		core.Advance(at(peerTimeout), &out)
		if got := broadcastRepls(out.Drain()); len(got) != 0 {
			t.Fatalf("eviction Advance emitted DataReplication for %v, want suspension", replKeys(got))
		}
		if got := len(core.PeerSnapshot()); got != 0 {
			t.Fatalf("PeerSnapshot holds %d peers after the sole peer's timeout, want 0", got)
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, ver, nil)

		// Suspension holds across the window where the ladder steps fell due.
		for off := peerTimeout + 100*time.Millisecond; off <= peerTimeout+time.Second; off += 100 * time.Millisecond {
			core.Advance(at(off), &out)
			if got := broadcastRepls(out.Drain()); len(got) != 0 {
				t.Fatalf("suspended sole-peer scanner emitted DataReplication for %v at t0+%v, want none", replKeys(got), off)
			}
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, ver, nil)

		// The peer returns: the Hello ends the empty table, so the same event
		// enqueues the flush send, re-adds the waiter and restarts the ladder;
		// the next Advance drains the queued send, and the peer's Ack then
		// drains the entry.
		backAt := at(peerTimeout + 2*time.Second)
		core.HandleDatagram(backAt, helloFrom(peerID1), &out)
		if got := len(out.Drain()); got != 0 {
			t.Fatalf("the returning Hello emitted %d messages inside the handler, want 0 — it only enqueues", got)
		}
		core.Advance(backAt, &out)
		if got := broadcastRepls(out.Drain()); len(got) != 1 || got[0].Key != keyAlpha || got[0].Record.Version != ver {
			t.Fatalf("returning sole peer's Hello flushed %v, want exactly one DataReplication for (%s, %x)",
				got, keyAlpha, ver[:])
		}
		snap := core.PendingSnapshot()
		assertPending(t, snap, keyAlpha, ver, []uint16{peerID1})
		if got := snap[keyAlpha].NextRetry; !got.Equal(backAt.Add(ackTimeout)) {
			t.Fatalf("post-return NextRetry = %v, want the restarted ladder step %v", got, backAt.Add(ackTimeout))
		}
		core.HandleDatagram(backAt, ackFrom(peerID1, keyAlpha, ver), &out)
		out.Drain()
		if _, ok := core.PendingSnapshot()[keyAlpha]; ok {
			t.Fatal("pending entry retained after the returning sole peer's Ack")
		}
	})
}
