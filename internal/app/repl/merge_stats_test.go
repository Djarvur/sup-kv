package repl_test

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/port"
)

// TestMergeStats_counts proves the OBS-01 counter layer at the applyRemote
// seam: a strictly greater remote version counts one conflict (the
// displaced local version — an LWW loser is an ordinary overwrite, counted
// and never journaled, D-10), an equal-version redelivery counts nothing
// (merge idempotence), and a record stamped beyond the store clock's
// drift-warn threshold — the signal clock.Update already gates to zero
// within tolerance — counts one drift warning while a first record under an
// absent key conflicts nothing.
func TestMergeStats_counts(t *testing.T) {
	t.Parallel()
	core, st := newTestCore()
	now := at(0)
	var out port.Outbox

	_, _, err := st.store.Set(keyAlpha, []byte("local"))
	if err != nil {
		t.Fatalf("store Set(%q): %v", keyAlpha, err)
	}
	_, err = st.store.Get(keyAlpha)
	if err != nil {
		t.Fatalf("store Get(%q): %v", keyAlpha, err)
	}

	// A greater remote version (l strictly ahead of the fixed local
	// millisecond) displaces the local record: exactly one conflict.
	greater := remoteRecord(testNowMs+5000, peerID1, "remote-wins")
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, greater), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 0 {
		t.Fatalf("after the displacing apply MergeStats() = %+v, want Conflicts 1, DriftWarnings 0", got)
	}
	served, err := st.store.Get(keyAlpha)
	if err != nil || served.Version != greater.Version {
		t.Fatalf("store serves (%x, %v) after the apply, want the remote version %x", served.Version[:], err, greater.Version[:])
	}

	// The identical redelivery conflicts nothing: merge idempotence, the
	// equal-version full no-op.
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, greater), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 0 {
		t.Fatalf("after the equal redelivery MergeStats() = %+v, want the unchanged Conflicts 1, DriftWarnings 0", got)
	}

	// A record stamped beyond the drift-warn threshold drives the drift
	// counter on the apply AFTER its absorption: clock.Update measures the
	// clock's own divergence from the wall clock at the event (the Kulkarni
	// receive rule advances l first), so the far-ahead stamp lands under
	// keyBeta with no conflict and no warning yet, and the next apply — an
	// equal redelivery of the same record — observes l one warning window
	// past the wall clock and counts the warning. keyBeta holds no local
	// record, so no conflict ever joins it.
	aheadMs := testNowMs + 5000 + int64(testDriftWarn/time.Millisecond) + 1000
	ahead := remoteRecord(aheadMs, peerID1, "from-the-future")
	core.HandleDatagram(now, replFrom(peerID1, keyBeta, ahead), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 0 {
		t.Fatalf("after the far-ahead apply MergeStats() = %+v, want Conflicts 1, DriftWarnings 0 (the stamp is absorbed, not yet warned)", got)
	}
	core.HandleDatagram(now, replFrom(peerID1, keyBeta, ahead), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 1 {
		t.Fatalf("after the post-absorption apply MergeStats() = %+v, want Conflicts still 1, DriftWarnings 1", got)
	}
}

// TestRetransmit_counter proves the retransmissions counter climbs with the
// ladder: nothing before the first due step, exactly one per fired
// retransmission at t0+ack_timeout and t0+3×ack_timeout (the cumulative
// 200ms/600ms steps of the pinned ladder).
func TestRetransmit_counter(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out) // startup Hello
	out.Drain()
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	out.Drain()
	localWrite(t, core, st, now, keyAlpha, "pending")

	if got := core.Retransmissions(); got != 0 {
		t.Fatalf("Retransmissions() = %d before any ladder fire, want 0", got)
	}
	core.Advance(at(ackTimeout), &out)
	out.Drain()
	if got := core.Retransmissions(); got != 1 {
		t.Fatalf("Retransmissions() = %d after the first ladder fire, want 1", got)
	}
	core.Advance(at(3*ackTimeout), &out)
	out.Drain()
	if got := core.Retransmissions(); got != 2 {
		t.Fatalf("Retransmissions() = %d after the second ladder fire, want 2", got)
	}
}
