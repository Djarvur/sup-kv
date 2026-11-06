package repl_test

import (
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/port"
)

// runIDFor returns the run ID of generation gen for nodeID: byte 14 carries
// the generation, so the restart tests can hand one node several distinct run
// ids (R1/R2/R3 of a stale return and rapid double restarts) while byte 0
// keeps every generation distinct from every other node's testRunID.
func runIDFor(nodeID uint16, gen byte) [16]byte {
	id := testRunID(nodeID)
	id[14] = gen
	return id
}

// helloFromRun builds a decoded Hello datagram from peerID announcing runID —
// the run-id-parameterized discovery input the restart tests drive (REPL-07;
// helloFrom itself carries only the node's default testRunID).
func helloFromRun(peerID uint16, runID [16]byte) port.InboundDatagram {
	in := helloFrom(peerID)
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
// reaction must reproduce exactly (same chunks, same order): the reaction
// reuses the same machinery, never a second implementation.
func freshListing(t *testing.T, st *stateStore) []port.DataSync {
	t.Helper()
	fresh := repl.New(localID, testRunID(localID), handleFor(localID), st, repl.Config{})
	var out port.Outbox
	fresh.Advance(at(0), &out)
	syncs := dataSyncs(out.Drain())
	if len(syncs) != 2 {
		t.Fatalf("the fresh listing build emitted %d chunks, want the 2-chunk graduated shape", len(syncs))
	}
	return syncs
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

	r1, r2 := runIDFor(peerID1, 1), runIDFor(peerID1, 2)
	core.HandleDatagram(now, helloFromRun(peerID1, r1), &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("the learn Hello from the unknown node emitted %d messages, want 0 — no targeted listing before the node is known", got)
	}

	core.HandleDatagram(now, helloFromRun(peerID1, r2), &out)
	drained := out.Drain()
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

	core.HandleDatagram(now, helloFromRun(peerID1, r2), &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("a repeat Hello with the already-remembered run_id emitted %d messages, want 0 — idempotence: refresh only, never a second listing", got)
	}
	if q := core.QueuedResponses(); q != 0 {
		t.Fatalf("after the repeat Hello QueuedResponses() = %d, want 0", q)
	}
}

// TestRestart_newPeerNoFullSync proves the discrimination edge: a first
// Hello from a brand-new node_id is learned into the peer table but never
// targeted — the reaction fires only on a run_id CHANGE of a known node, so
// a new node reconciles through its own startup DataSync and the periodic
// rounds (a per-Hello full listing at hello_interval cadence would be a
// self-inflicted listing flood).
func TestRestart_newPeerNoFullSync(t *testing.T) {
	t.Parallel()
	t.Helper()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	out.Drain()
	for i, keyLen := range graduatedKeyLens[:4] {
		_, _, err := st.store.Set(graduatedKey(i, keyLen), []byte("v"))
		if err != nil {
			t.Fatalf("store Set(graduated key %d): %v", i, err)
		}
	}

	core.HandleDatagram(now, helloFrom(strangerID), &out)
	drained := out.Drain()
	if got := targetedSyncs(t, drained, handleFor(strangerID)); len(got) != 0 {
		t.Fatalf("a brand-new node_id received %d targeted DataSync chunks, want 0", len(got))
	}
	if len(drained) != 0 {
		t.Fatalf("the learn Hello from the brand-new node emitted %d messages, want 0", len(drained))
	}
	snap := core.PeerSnapshot()
	if _, ok := snap[strangerID]; !ok {
		t.Fatal("the brand-new node was not learned into PeerSnapshot")
	}
	if q := core.QueuedResponses(); q != 0 {
		t.Fatalf("after the learn QueuedResponses() = %d, want 0", q)
	}
}
