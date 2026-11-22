package repl_test

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/port"
)

// runIDFor returns the run ID of generation gen for the restarted peer B
// (peerID1): byte 14 carries the generation, so the restart tests can hand B
// several distinct run ids (R1/R2/R3 of a stale return and rapid double
// restarts) while byte 0 keeps every generation distinct from every other
// node's testRunID.
func runIDFor(gen byte) [16]byte {
	id := testRunID(peerID1)
	id[14] = gen
	return id
}

// helloFromRun builds a decoded Hello datagram from the restarted peer B
// (peerID1) announcing runID — the run-id-parameterized discovery input the
// restart tests drive (REPL-07; helloFrom itself carries only the node's
// default testRunID).
func helloFromRun(runID [16]byte) port.InboundDatagram {
	in := helloFrom(peerID1)
	in.RunID = runID
	return in
}

// targetedSyncs returns every unicast DataSync chunk of drained, failing the
// test when one is addressed anywhere but the expected handle — the
// restart-listing probe (plan 04-03's broadcast dataSyncs filters these out
// precisely because the restart listing is unicast toward the restarted
// peer).
func targetedSyncs(t *testing.T, drained []port.Outbound, handle string) []port.DataSync {
	t.Helper()
	var syncs []port.DataSync
	for _, ob := range drained {
		if !ob.Broadcast {
			if msg, ok := ob.Msg.(port.DataSync); ok {
				if ob.To != handle {
					t.Fatalf("a targeted DataSync chunk is addressed to %q, want the restarted peer's handle %q", ob.To, handle)
				}
				syncs = append(syncs, msg)
			}
		}
	}
	return syncs
}

// seedGraduatedKeys fills the store with the eight graduated keys plus the
// two 50-byte follow-ups — the TestSync_chunking shape whose full listing
// spans exactly two chunks (64 + 1336 = 1400 closes the first, the two
// follow-ups open the second) — so a restart reaction has a multi-chunk
// listing to emit.
func seedGraduatedKeys(t *testing.T, st *stateStore) {
	t.Helper()
	for i, keyLen := range graduatedKeyLens {
		_, _, err := st.store.Set(graduatedKey(i, keyLen), []byte("v"))
		if err != nil {
			t.Fatalf("store Set(graduated key %d): %v", i, err)
		}
	}
	for i := range 2 {
		_, _, err := st.store.Set(graduatedKey(8+i, 50), []byte("v"))
		if err != nil {
			t.Fatalf("store Set(graduated key %d): %v", 8+i, err)
		}
	}
}

// freshListing returns the startup DataSync chunk sequence a brand-new core
// over st builds — the independent plan-04-03 listing build the restart
// reaction must reproduce (same chunks, same start order in round 0): the
// reaction reuses the same machinery, never a second implementation. The
// listing pacer drains one chunk per interval, so the helper finishes the
// round across Advances.
func freshListing(t *testing.T, st *stateStore) []port.DataSync {
	t.Helper()
	fresh := repl.New(localID, testRunID(localID), st, repl.Config{})
	var out port.Outbox
	syncs := dataSyncs(advanceDrain(fresh, &out, at(0)))
	if len(syncs) != 2 {
		t.Fatalf("the fresh listing build emitted %d chunks, want the 2-chunk graduated shape", len(syncs))
	}
	return syncs
}

// rotateChunks returns syncs rotated left by one chunk — the start order the
// next targeted round produces (start = round % len(chunks)): the fixed tail
// of one round becomes the head of the next, which is what covers the whole
// key space under systematic buffer loss.
func rotateChunks(syncs []port.DataSync) []port.DataSync {
	if len(syncs) < 2 {
		return syncs
	}
	out := make([]port.DataSync, 0, len(syncs))
	out = append(out, syncs[1:]...)
	out = append(out, syncs[0])
	return out
}

// TestRestart_runIDChangeTriggersFullSync proves REPL-07's reaction and its
// idempotence: a Hello from a known node_id whose run_id differs from the
// remembered one updates the entry in place (new run_id, new handle, one
// table row) and fires exactly the full plan-04-03 listing — every chunk in
// chunk order, unicast toward the restarted peer's handle, bypassing the
// egress limiter (proactive, once per detected change) — and a repeat Hello
// with the same new run_id fires nothing further.
func TestRestart_runIDChangeTriggersFullSync(t *testing.T) {
	t.Parallel()
	t.Helper()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out) // startup Hello + startup DataSync (plan 04-03)
	out.Drain()
	seedGraduatedKeys(t, st)
	want := freshListing(t, st)

	r1, r2 := runIDFor(1), runIDFor(2)
	core.HandleDatagram(now, helloFromRun(r1), &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("the learn Hello from the unknown node emitted %d messages synchronously, want 0 — its first-seen listing is queued on the pacer", got)
	}

	core.HandleDatagram(now, helloFromRun(r2), &out)
	drained := advanceDrain(core, &out, now)
	got := targetedSyncs(t, drained, handleFor(peerID1))
	if len(drained) != len(got) {
		t.Fatalf("the restart Hello drained %d messages of which %d are targeted DataSync chunks, want exactly the chunk sequence and nothing else", len(drained), len(got))
	}
	if !dataSyncsEqual(got, want) {
		t.Fatalf("the targeted listing = %v, want the exact fresh plan-04-03 chunk sequence %v over the same store", got, want)
	}
	if q := core.QueuedResponses(); q != 0 {
		t.Fatalf("the restart listing enqueued %d egress responses, want 0 — proactive traffic bypasses the limiter (the plan-04-03 egress-scope pin)", q)
	}
	snap := core.PeerSnapshot()
	if len(snap) != 1 {
		t.Fatalf("PeerSnapshot holds %d entries after the restart, want exactly one row for the restarted node_id", len(snap))
	}
	if p := snap[peerID1]; p.RunID != r2 || p.Handle != handleFor(peerID1) {
		t.Fatalf("the restarted peer's row = (run %x, handle %q), want (run %x, handle %q)", p.RunID[:], p.Handle, r2[:], handleFor(peerID1))
	}

	core.HandleDatagram(now, helloFromRun(r2), &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("a repeat Hello with the already-remembered run_id emitted %d messages, want 0 — idempotence: refresh only, never a second listing", got)
	}
	if q := core.QueuedResponses(); q != 0 {
		t.Fatalf("after the repeat Hello QueuedResponses() = %d, want 0", q)
	}
}

// TestRestart_firstHelloExactlyOneListing proves PROTO-04's first-seen half:
// an unknown node_id is learned into the peer table and receives exactly one
// targeted, complete listing — one chunk for a small store, with the chunk
// header and every stored key — without waiting for a sync_interval tick.
func TestRestart_firstHelloExactlyOneListing(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out) // startup Hello + startup empty listing
	out.Drain()
	_, _, err := st.store.Set(keyAlpha, []byte("v"))
	if err != nil {
		t.Fatalf("store Set(%q): %v", keyAlpha, err)
	}

	core.HandleDatagram(now, helloFrom(peerID1), &out)
	drained := advanceDrain(core, &out, now)
	got := targetedSyncs(t, drained, handleFor(peerID1))
	if len(got) != 1 {
		t.Fatalf("the first Hello emitted %d targeted DataSync chunks, want exactly 1 — a newly seen peer gets one complete listing", len(got))
	}
	if len(drained) != len(got) {
		t.Fatalf("the first Hello drained %d messages of which %d are targeted listing chunks, want exactly the listing and nothing else",
			len(drained), len(got))
	}
	if sync := got[0]; sync.ChunkIndex != 1 || sync.ChunkCount != 1 || len(sync.Entries) != 1 || sync.Entries[0].Key != keyAlpha {
		t.Fatalf("the first-seen listing chunk = %+v, want the one complete chunk carrying %q", sync, keyAlpha)
	}
	if _, ok := core.PeerSnapshot()[peerID1]; !ok {
		t.Fatal("the first Hello was not learned into PeerSnapshot")
	}
	if got := core.ListingDropped(); got != 0 {
		t.Fatalf("ListingDropped() = %d after one first-seen listing, want 0", got)
	}
}

// TestRestart_repeatHelloNoListing pins the repeat contract: a second Hello
// with the already-remembered run_id refreshes liveness and emits nothing.
func TestRestart_repeatHelloNoListing(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, _ := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	advanceDrain(core, &out, now) // the first-seen listing, outside the assertion
	out.Drain()
	core.HandleDatagram(at(time.Millisecond), helloFrom(peerID1), &out)
	drained := advanceDrain(core, &out, at(time.Millisecond))
	if len(drained) != 0 {
		t.Fatalf("a repeat Hello with the remembered run_id emitted %d messages, want 0 — idempotence: refresh only", len(drained))
	}
}

// TestRestart_returningAfterEvictionGetsListing proves the returning half
// (PROTO-04): a peer evicted by peer_timeout is unknown again, so its next
// Hello takes the first-seen path and gets exactly one complete listing.
func TestRestart_returningAfterEvictionGetsListing(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, _ := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	advanceDrain(core, &out, now) // the initial first-seen listing
	out.Drain()

	core.Advance(at(peerTimeout), &out) // lastHello+peer_timeout evicts
	out.Drain()
	if _, ok := core.PeerSnapshot()[peerID1]; ok {
		t.Fatal("the silent peer survived peer_timeout; the eviction probe is vacuous")
	}

	helloAt := at(peerTimeout + 100*time.Millisecond)
	core.HandleDatagram(helloAt, helloFrom(peerID1), &out)
	drained := advanceDrain(core, &out, helloAt)
	got := targetedSyncs(t, drained, handleFor(peerID1))
	if len(got) != 1 {
		t.Fatalf("the evicted peer's returning Hello emitted %d targeted DataSync chunks, want exactly 1 — a returning peer is a first-seen peer", len(got))
	}
	if _, ok := core.PeerSnapshot()[peerID1]; !ok {
		t.Fatal("the returning peer was not re-learned into PeerSnapshot")
	}
}

// TestRestart_firstHelloStillFlushesPending proves the two first-Hello
// behaviors are not mutually exclusive (Pitfall 5): a Hello that ends an empty
// table both enqueues the targeted listing AND flushes every pending write, so
// the discovery flush's R-02 drain bound survives the first-seen listing.
func TestRestart_firstHelloStillFlushesPending(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	ver := localWrite(t, core, st, now, keyAlpha, "one")
	if _, ok := core.PendingSnapshot()[keyAlpha]; !ok {
		t.Fatalf("the pending write %q left tracking while the table was empty", keyAlpha)
	}

	helloAt := at(time.Millisecond)
	core.HandleDatagram(helloAt, helloFrom(peerID1), &out)
	drained := advanceDrain(core, &out, helloAt)
	if got := targetedSyncs(t, drained, handleFor(peerID1)); len(got) != 1 {
		t.Fatalf("the first Hello emitted %d targeted listing chunks, want exactly 1 beside the discovery flush", len(got))
	}
	repls := broadcastRepls(drained)
	if len(repls) != 1 || repls[0].Key != keyAlpha || repls[0].Record.Version != ver {
		t.Fatalf("the first Hello flushed %v, want exactly one DataReplication for (%s, %x) — the discovery flush was dropped",
			replKeys(repls), keyAlpha, ver[:])
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1})
}

// TestRestart_helloDoesNotLeakListingsToStranger proves the two-destination
// independence: two alternating unknown identities each get exactly their own
// one-chunk first-seen listing, repeats add nothing, and both destinations fit
// under the cap with no counted drop.
func TestRestart_helloDoesNotLeakListingsToStranger(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, _ := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	for range 2 {
		core.HandleDatagram(now, helloFrom(peerID1), &out)
		core.HandleDatagram(now, helloFrom(strangerID), &out)
	}
	drained := advanceDrain(core, &out, now)
	perHandle := make(map[string]int, 2)
	for _, ob := range drained {
		sync, ok := ob.Msg.(port.DataSync)
		if !ok || ob.Broadcast {
			continue
		}
		perHandle[ob.To]++
		if len(sync.Entries) != 0 {
			t.Fatalf("an empty-store first-seen listing carries %d entries, want the one empty chunk", len(sync.Entries))
		}
	}
	for _, handle := range []string{handleFor(peerID1), handleFor(strangerID)} {
		if got := perHandle[handle]; got != 1 {
			t.Fatalf("handle %q received %d listing chunks, want exactly 1 (per-handle counts: %v)", handle, got, perHandle)
		}
	}
	if got := core.ListingDropped(); got != 0 {
		t.Fatalf("ListingDropped() = %d with two destinations under the cap, want 0", got)
	}
}

// TestRestart_generationInvalidatesCachedListing is the freshness regression
// the generation check exists for: both subtests fire the startup Advance
// (caching an empty build), seed the store afterwards, and then trigger a
// listing. The emitted chunks must equal a fresh listing of the seeded store —
// a reused pre-mutation build would emit the empty listing and fail.
func TestRestart_generationInvalidatesCachedListing(t *testing.T) {
	t.Parallel()
	r1, r2 := runIDFor(1), runIDFor(2)
	t.Run("a restart after a cached build and a store mutation emits the mutated store", func(t *testing.T) {
		t.Parallel()
		now := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now, &out) // caches the startup build over the empty store
		out.Drain()
		core.HandleDatagram(now, helloFromRun(r1), &out) // learn, reuses the empty build
		out.Drain()
		seedGraduatedKeys(t, st)
		want := freshListing(t, st)
		core.HandleDatagram(now, helloFromRun(r2), &out)
		got := targetedSyncs(t, advanceDrain(core, &out, now), handleFor(peerID1))
		if !dataSyncsEqual(got, want) {
			t.Fatalf("the post-mutation restart listing = %v, want the fresh seeded listing %v — a cached pre-mutation build was served", got, want)
		}
	})
	t.Run("a first-seen Hello after a cached build and a store mutation emits the mutated store", func(t *testing.T) {
		t.Parallel()
		now := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now, &out) // caches the startup build over the empty store
		out.Drain()
		seedGraduatedKeys(t, st)
		want := freshListing(t, st)
		core.HandleDatagram(now, helloFrom(strangerID), &out)
		got := targetedSyncs(t, advanceDrain(core, &out, now), handleFor(strangerID))
		if !dataSyncsEqual(got, want) {
			t.Fatalf("the post-mutation first-seen listing = %v, want the fresh seeded listing %v — a cached pre-mutation build was served", got, want)
		}
	})
}

// TestRestart_staleRunIdAndInFlight pins the three adversarial run_id edges
// of probe #19: a stale run_id returning after a newer one was seen is just
// another change (inequality with the remembered value is the only test —
// random 16-byte run_ids carry no temporal order), in-flight protocol
// datagrams from the old run process normally on their (key, version)
// merits outside the Hello handler, and a rapid double restart updates the
// one row in place with exactly one full listing per change.
func TestRestart_staleRunIdAndInFlight(t *testing.T) {
	t.Parallel()
	t.Run("a stale run_id returning is just another change: idempotent second listing, one row", func(t *testing.T) {
		t.Parallel()
		t.Helper()
		now := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now, &out)
		out.Drain()
		seedGraduatedKeys(t, st)
		want := freshListing(t, st)

		r1, r2 := runIDFor(1), runIDFor(2)
		core.HandleDatagram(now, helloFromRun(r1), &out)
		out.Drain() // learn only
		core.HandleDatagram(now, helloFromRun(r2), &out)
		first := targetedSyncs(t, advanceDrain(core, &out, now), handleFor(peerID1))
		if !dataSyncsEqual(first, want) {
			t.Fatalf("the R1→R2 reaction = %v, want the full fresh listing %v", first, want)
		}

		// A delayed Hello from the old run's R1 arrives: inequality with the
		// remembered R2 is indistinguishable from a fresh restart, so the
		// entry flips back and one more full listing fires — rotated one chunk
		// by the per-peer round counter — idempotent,
		// safe (reconciliation no-ops on equal state).
		core.HandleDatagram(now, helloFromRun(r1), &out)
		second := targetedSyncs(t, advanceDrain(core, &out, now), handleFor(peerID1))
		if !dataSyncsEqual(second, rotateChunks(want)) {
			t.Fatalf("the stale-return reaction = %v, want the next round's rotation %v of the fresh listing %v", second, rotateChunks(want), want)
		}
		if got := len(first) + len(second); got != 2*len(want) {
			t.Fatalf("the two changes emitted %d targeted chunks in total, want exactly two full listings (%d)", got, 2*len(want))
		}
		snap := core.PeerSnapshot()
		if len(snap) != 1 {
			t.Fatalf("PeerSnapshot holds %d rows for the one node_id, want exactly 1 — a run_id change never forks the table", len(snap))
		}
		if p := snap[peerID1]; p.RunID != r1 {
			t.Fatalf("the row's run_id after the stale return = %x, want the last-announced R1 %x", p.RunID[:], r1[:])
		}
	})
	t.Run("in-flight old-run traffic processes on (key, version) merits", func(t *testing.T) {
		t.Parallel()
		t.Helper()
		now := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now, &out)
		out.Drain()
		core.HandleDatagram(now, helloFromRun(runIDFor(1)), &out)
		out.Drain()

		ver := localWrite(t, core, st, now, keyAlpha, "one")
		assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1})
		core.HandleDatagram(now, helloFromRun(runIDFor(2)), &out)
		core.Advance(now, &out) // drains the targeted listing's single chunk
		out.Drain()             // the targeted listing; this sequence is about what follows it

		// An in-flight Ack from the old run for the still-pending version
		// drains its waiter normally — the Ack path never looks at run_id.
		core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, ver), &out)
		out.Drain()
		if _, ok := core.PendingSnapshot()[keyAlpha]; ok {
			t.Fatal("an in-flight Ack from the old run failed to drain the pending entry")
		}

		// An in-flight DataReplication from the old run carrying a strictly
		// greater version supersedes a pending entry normally and is Acked.
		ver2 := localWrite(t, core, st, now, keyBeta, "two")
		greater := remoteRecord(testNowMs+1, peerID1, "remote wins")
		if greater.Version.Compare(ver2) <= 0 {
			t.Fatalf("test remote version %x not strictly greater than the tracked %x", greater.Version[:], ver2[:])
		}
		core.HandleDatagram(now, replFrom(peerID1, keyBeta, greater), &out)
		drained := out.Drain()
		if _, ok := core.PendingSnapshot()[keyBeta]; ok {
			t.Fatal("an in-flight strictly-greater DataReplication from the old run failed to supersede the pending entry")
		}
		got, err := st.store.Get(keyBeta)
		if err != nil || got.Version != greater.Version {
			t.Fatalf("store Get(%q) after the old-run DataReplication = (%x, %v), want the remote version %x", keyBeta, got.Version[:], err, greater.Version[:])
		}
		acked := false
		for _, ob := range drained {
			if ack, ok := ob.Msg.(port.Ack); ok && !ob.Broadcast && ob.To == handleFor(peerID1) &&
				ack.Key == keyBeta && ack.Version == greater.Version {
				acked = true
			}
		}
		if !acked {
			t.Fatalf("the old-run DataReplication was not Acked: drained %v", drained)
		}
	})
	t.Run("rapid double restart updates in place: one listing per change, never two rows", func(t *testing.T) {
		t.Parallel()
		t.Helper()
		now0 := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now0, &out)
		out.Drain()
		seedGraduatedKeys(t, st)
		want := freshListing(t, st)
		r1, r2, r3 := runIDFor(1), runIDFor(2), runIDFor(3)

		core.HandleDatagram(now0, helloFromRun(r1), &out)
		if got := len(out.Drain()); got != 0 {
			t.Fatalf("the learn Hello emitted %d messages synchronously, want 0", got)
		}
		advanceDrain(core, &out, now0) // the learn's first-seen listing
		out.Drain()
		core.Advance(at(100*time.Millisecond), &out)
		if got := len(out.Drain()); got != 0 {
			t.Fatalf("the Advance after the learn emitted %d messages, want 0", got)
		}
		core.HandleDatagram(at(200*time.Millisecond), helloFromRun(r2), &out)
		first := targetedSyncs(t, advanceDrain(core, &out, at(200*time.Millisecond)), handleFor(peerID1))
		core.Advance(at(300*time.Millisecond), &out)
		if got := len(out.Drain()); got != 0 {
			t.Fatalf("the Advance between the restarts emitted %d messages, want 0", got)
		}
		core.HandleDatagram(at(400*time.Millisecond), helloFromRun(r3), &out)
		second := targetedSyncs(t, advanceDrain(core, &out, at(400*time.Millisecond)), handleFor(peerID1))
		core.Advance(at(500*time.Millisecond), &out)
		if got := len(out.Drain()); got != 0 {
			t.Fatalf("the Advance after the second restart emitted %d messages, want 0", got)
		}

		if len(first) != len(want) || !dataSyncsEqual(first, want) {
			t.Fatalf("the R1→R2 reaction = %v, want exactly one full fresh listing %v", first, want)
		}
		if len(second) != len(want) || !dataSyncsEqual(second, rotateChunks(want)) {
			t.Fatalf("the R2→R3 reaction = %v, want the next round's rotation %v", second, rotateChunks(want))
		}
		snap := core.PeerSnapshot()
		if len(snap) != 1 {
			t.Fatalf("PeerSnapshot holds %d rows for the one node_id, want exactly 1", len(snap))
		}
		if p := snap[peerID1]; p.RunID != r3 {
			t.Fatalf("the row's run_id after the double restart = %x, want the last-announced R3 %x", p.RunID[:], r3[:])
		}
	})
}
