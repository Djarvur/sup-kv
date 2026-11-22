package repl_test

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
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
		// drains the entry. The handler's only synchronous emission is the
		// identity echo (CR-01) — the flush itself only enqueues.
		backAt := at(peerTimeout + 2*time.Second)
		core.HandleDatagram(backAt, helloFrom(peerID1), &out)
		if got := out.Drain(); len(got) != 1 {
			t.Fatalf("the returning Hello emitted %d messages inside the handler, want 1 (the identity echo) — the flush only enqueues", len(got))
		} else if _, ok := got[0].Msg.(port.Hello); !ok || got[0].To != handleFor(peerID1) {
			t.Fatalf("the returning Hello's synchronous emission %v is not the unicast identity echo to %q", got[0], handleFor(peerID1))
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

// helloRun builds a decoded Hello datagram from nodeID announcing run
// generation gen: byte 14 carries the generation, so one peer can announce
// several distinct run ids while byte 0 keeps each distinct from every other
// node's testRunID. gen 0 is the node's default run id.
func helloRun(nodeID uint16, gen byte) port.InboundDatagram {
	in := helloFrom(nodeID)
	if gen != 0 {
		in.RunID[14] = gen
	}
	return in
}

// configErrorWarns filters captured Config.Log lines down to the
// configuration-error WARNs. Every duplicate-node_id path logs the
// Spec-mandated prefix, so a count is the shared rate limiter's exact
// frequency.
func configErrorWarns(logs []string) []string {
	var warns []string
	for _, line := range logs {
		if strings.Contains(line, "configuration error") {
			warns = append(warns, line)
		}
	}
	return warns
}

// TestPeer_alternatingRunIDsIsConfigError pins PROTO-06's named
// misconfiguration: a known node_id announcing a run_id it carried before but
// which is not current is counted as DuplicateNodeID, WARNed once as a
// configuration error, dropped without a listing and leaves the row (run_id
// and liveness) on the current run — alternating identities are never
// rewarded with a listing per datagram.
func TestPeer_alternatingRunIDsIsConfigError(t *testing.T) {
	t.Parallel()
	now := at(0)
	var logs []string
	core, _ := newTestCoreWithConfig(repl.Config{Log: func(msg string) { logs = append(logs, msg) }})
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	r1, r2 := helloRun(peerID1, 1).RunID, helloRun(peerID1, 2).RunID
	core.HandleDatagram(now, helloRun(peerID1, 1), &out)
	advanceDrain(core, &out, now) // the first-seen listing
	out.Drain()
	core.HandleDatagram(at(100*time.Millisecond), helloRun(peerID1, 2), &out)
	advanceDrain(core, &out, at(100*time.Millisecond)) // the legitimate restart listing
	out.Drain()
	logs = nil

	core.HandleDatagram(at(200*time.Millisecond), helloRun(peerID1, 1), &out)
	if got := len(advanceDrain(core, &out, at(200*time.Millisecond))); got != 0 {
		t.Fatalf("the alternating run_id emitted %d messages, want 0 — a previously seen identity gets no listing", got)
	}
	if got := core.Discards().DuplicateNodeID; got != 1 {
		t.Fatalf("Discards().DuplicateNodeID = %d after one alternation, want 1", got)
	}
	warns := configErrorWarns(logs)
	if len(warns) != 1 {
		t.Fatalf("Config.Log carried %d configuration-error WARN(s), want exactly 1: %v", len(warns), logs)
	}
	if !strings.Contains(warns[0], fmt.Sprintf("node_id %d", peerID1)) {
		t.Fatalf("the configuration-error WARN %q does not name node_id %d", warns[0], peerID1)
	}
	snap := core.PeerSnapshot()
	if len(snap) != 1 {
		t.Fatalf("PeerSnapshot holds %d rows after the alternation, want exactly 1", len(snap))
	}
	if p := snap[peerID1]; p.RunID != r2 {
		t.Fatalf("the row's run_id after the alternation = %x, want the current run %x — the dropped datagram never updates the row", p.RunID[:], r2[:])
	}
	if p := snap[peerID1]; !p.LastHello.Equal(at(100 * time.Millisecond)) {
		t.Fatalf("the row's LastHello after the alternation = %v, want the current run's Hello time %v — a dropped datagram never refreshes liveness",
			p.LastHello, at(100*time.Millisecond))
	}
	if r1 == r2 {
		t.Fatalf("the two test run generations collided: %x", r1[:])
	}
}

// TestPeer_alternatingWarnRateLimitedPerPeer proves the shared
// per-(node_id, class) limiter on the alternation path (T-4-28): two peers
// alternating inside one interval each get their own WARN, a repeated
// alternation from the same peer adds none until the interval elapses on
// event time, and the boundary fires exactly at the interval.
func TestPeer_alternatingWarnRateLimitedPerPeer(t *testing.T) {
	t.Parallel()
	now := at(0)
	const warnInterval = 10 * time.Second // the pinned discardWarnInterval
	var logs []string
	core, _ := newTestCoreWithConfig(repl.Config{Log: func(msg string) { logs = append(logs, msg) }})
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	for _, id := range []uint16{peerID1, peerID2} {
		core.HandleDatagram(now, helloRun(id, 1), &out)
		advanceDrain(core, &out, now) // each first-seen listing
		out.Drain()
	}
	logs = nil

	// One restart then one alternation per peer: each peer's first
	// configuration error must WARN, even inside the same interval.
	for _, id := range []uint16{peerID1, peerID2} {
		core.HandleDatagram(now, helloRun(id, 2), &out)
		advanceDrain(core, &out, now) // the legitimate restart listing
		out.Drain()
		core.HandleDatagram(now, helloRun(id, 1), &out)
		advanceDrain(core, &out, now) // the alternation: no listing
		out.Drain()
	}
	if got := len(configErrorWarns(logs)); got != 2 {
		t.Fatalf("Config.Log carried %d configuration-error WARN(s) for two alternating peers in one interval, want 2 (one per peer): %v", got, logs)
	}

	// The same peer's repeated alternation inside the interval adds nothing.
	core.HandleDatagram(now.Add(time.Second), helloRun(peerID1, 1), &out)
	advanceDrain(core, &out, now.Add(time.Second))
	out.Drain()
	if got := len(configErrorWarns(logs)); got != 2 {
		t.Fatalf("Config.Log carried %d configuration-error WARN(s) inside the interval, want still 2: %v", got, logs)
	}

	// Exactly at the interval peer 1's slot warns again.
	core.HandleDatagram(now.Add(warnInterval), helloRun(peerID1, 1), &out)
	advanceDrain(core, &out, now.Add(warnInterval))
	out.Drain()
	if got := len(configErrorWarns(logs)); got != 3 {
		t.Fatalf("Config.Log carried %d configuration-error WARN(s) at the interval boundary, want 3: %v", got, logs)
	}
}

// TestPeer_seenRingBounded pins the seen-run ring's fixed bound (T-4-30):
// after nine distinct runs the first is still inside the 8-entry ring (an
// alternation is a configuration error), while the tenth run evicts it — the
// ring never holds more than 8, so an ancient run id is treated as a restart
// again (documented, bounded behavior).
func TestPeer_seenRingBounded(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, _ := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	r1 := helloRun(peerID1, 1).RunID
	for gen := byte(1); gen <= 9; gen++ {
		core.HandleDatagram(now, helloRun(peerID1, gen), &out)
		advanceDrain(core, &out, now) // the first-seen or restart listing
		out.Drain()
	}
	r9 := helloRun(peerID1, 9).RunID

	// Nine runs leave nine previous-or-current identities but the ring keeps
	// the previous 8; run 1 was pushed when run 2 arrived and is still inside.
	core.HandleDatagram(now, helloRun(peerID1, 1), &out)
	if got := len(advanceDrain(core, &out, now)); got != 0 {
		t.Fatalf("run 1 after nine distinct runs emitted %d messages, want 0 — it is still inside the 8-entry ring", got)
	}
	if got := core.Discards().DuplicateNodeID; got != 1 {
		t.Fatalf("Discards().DuplicateNodeID = %d after the ninth-run alternation, want 1", got)
	}
	if p := core.PeerSnapshot()[peerID1]; p.RunID != r9 {
		t.Fatalf("the row's run_id after the in-ring alternation = %x, want run 9 %x", p.RunID[:], r9[:])
	}

	// The tenth distinct run pushes run 1 out of the ring.
	core.HandleDatagram(now, helloRun(peerID1, 10), &out)
	advanceDrain(core, &out, now)
	out.Drain()
	r10 := helloRun(peerID1, 10).RunID
	if p := core.PeerSnapshot()[peerID1]; p.RunID != r10 {
		t.Fatalf("the row's run_id after run 10 = %x, want run 10 %x", p.RunID[:], r10[:])
	}

	// Run 1 is now forgotten, so re-announcing it is an unseen run: a
	// legitimate restart with exactly one fresh listing.
	core.HandleDatagram(now, helloRun(peerID1, 1), &out)
	got := targetedSyncs(t, advanceDrain(core, &out, now), handleFor(peerID1))
	if len(got) != 1 {
		t.Fatalf("the evicted run 1 emitted %d targeted listing chunks, want exactly 1 — the ring forgot it, so it is a restart", len(got))
	}
	if p := core.PeerSnapshot()[peerID1]; p.RunID != r1 {
		t.Fatalf("the row's run_id after run 1's return = %x, want run 1 %x — an unseen run is accepted as a restart", p.RunID[:], r1[:])
	}
}

// TestPeer_repeatSameRunStillRefresh pins the preserved repeat contract: the
// current run_id still refreshes liveness, emits nothing, counts nothing and
// warns nothing — the ring must never swallow a legitimate repeat.
func TestPeer_repeatSameRunStillRefresh(t *testing.T) {
	t.Parallel()
	now := at(0)
	var logs []string
	core, _ := newTestCoreWithConfig(repl.Config{Log: func(msg string) { logs = append(logs, msg) }})
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	core.HandleDatagram(now, helloRun(peerID1, 1), &out)
	advanceDrain(core, &out, now) // the first-seen listing
	out.Drain()
	logs = nil
	r1 := helloRun(peerID1, 1).RunID

	later := at(500 * time.Millisecond)
	core.HandleDatagram(later, helloRun(peerID1, 1), &out)
	if got := len(advanceDrain(core, &out, later)); got != 0 {
		t.Fatalf("a repeat of the current run emitted %d messages, want 0 — refresh only", got)
	}
	if got := core.Discards().DuplicateNodeID; got != 0 {
		t.Fatalf("Discards().DuplicateNodeID = %d after a current-run repeat, want 0", got)
	}
	if got := len(configErrorWarns(logs)); got != 0 {
		t.Fatalf("Config.Log carried %d configuration-error WARN(s) for a current-run repeat, want 0: %v", got, logs)
	}
	p := core.PeerSnapshot()[peerID1]
	if !p.LastHello.Equal(later) {
		t.Fatalf("the repeat's LastHello = %v, want the refreshed stamp %v", p.LastHello, later)
	}
	if p.RunID != r1 {
		t.Fatalf("the repeat changed the row's run_id to %x, want %x", p.RunID[:], r1[:])
	}
}
