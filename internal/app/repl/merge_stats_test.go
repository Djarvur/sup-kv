package repl_test

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// driftLead is the inbound version bound this test's cores run with: the
// drift-warning scenario stamps a record one drift-warn window (60 s) plus a
// second ahead, which is legitimately beyond the 60 s production default, so
// the test widens the bound explicitly rather than weakening the default.
const driftLead = 2 * time.Minute

// TestMergeStats_counts proves the OBS-01 counter layer at the applyRemote
// seam: a strictly greater remote version counts one conflict (the
// displaced local version — an LWW loser is an ordinary overwrite, counted
// and never journaled, D-10), an equal-version redelivery counts nothing
// (the equal branch Acks without merging, PROTO-03), and a record stamped
// beyond the store clock's drift-warn threshold — the signal clock.Update
// already gates to zero within tolerance — counts one drift warning on the
// first apply AFTER its absorption, while a first record under an absent key
// conflicts nothing.
func TestMergeStats_counts(t *testing.T) {
	t.Parallel()
	core, st := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
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

	// The identical redelivery conflicts nothing: the equal-version branch
	// Acks and never merges, so the counters stay exactly as they were.
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, greater), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 0 {
		t.Fatalf("after the equal redelivery MergeStats() = %+v, want the unchanged Conflicts 1, DriftWarnings 0", got)
	}

	// A record stamped beyond the drift-warn threshold drives the drift
	// counter on the apply AFTER its absorption: clock.Update measures the
	// clock's own divergence from the wall clock at the event (the Kulkarni
	// receive rule advances l first), so the far-ahead stamp lands under
	// keyBeta with no conflict and no warning yet, and the next apply — a
	// record under the still-absent keyGamma, since an equal-version
	// redelivery now Acks without merging (PROTO-03) — observes l one
	// warning window past the wall clock and counts the warning. keyGamma
	// holds no local record, so no conflict ever joins it.
	aheadMs := testNowMs + 5000 + int64(testDriftWarn/time.Millisecond) + 1000
	ahead := remoteRecord(aheadMs, peerID1, "from-the-future")
	core.HandleDatagram(now, replFrom(peerID1, keyBeta, ahead), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 0 {
		t.Fatalf("after the far-ahead apply MergeStats() = %+v, want Conflicts 1, DriftWarnings 0 (the stamp is absorbed, not yet warned)", got)
	}
	core.HandleDatagram(now, replFrom(peerID1, keyGamma, ahead), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 1 {
		t.Fatalf("after the post-absorption apply MergeStats() = %+v, want Conflicts still 1, DriftWarnings 1", got)
	}
}

// TestMergeStats_listingAndStaleExcluded pins the two non-merge paths that
// must leave the remote-merge counters untouched. A listing part absorbs its
// max version into the store clock through Observe — that absorption is real
// and visible (the next local write strictly beats the listed state,
// PROTO-01) — but listing drift is Phase 7's OBSV territory and counts
// nothing here (Open Question 6). A strictly older DataReplication takes the
// answer-not-Ack branch, which returns before ApplyRemote, so it counts
// nothing either (PROTO-03): the counters describe merges, not answers.
func TestMergeStats_listingAndStaleExcluded(t *testing.T) {
	t.Parallel()
	core, st := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
	now := at(0)
	var out port.Outbox

	// A complete one-part listing whose max version sits one drift-warn
	// window plus a second past the wall clock — exactly the stamp that
	// would count a DriftWarning if it travelled the merge path.
	past := testNowMs + 5000 + int64(testDriftWarn/time.Millisecond) + 1000
	listed := version.PackVersion(past, 0, peerID1)
	core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{
		SyncID:     1,
		ChunkIndex: 1,
		ChunkCount: 1,
		Entries:    []port.SyncEntry{{Key: keyAlpha, Version: listed}},
	}), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
		t.Fatalf("after the listing part MergeStats() = %+v, want none — listing drift is clock-only this phase (OQ6)", got)
	}

	// The absorption is real, not skipped: the next local write strictly
	// beats the listed max version, so the part reached Observe.
	written := localWrite(t, core, st, now, keyBeta, "after-the-listing")
	if written.Compare(listed) <= 0 {
		t.Fatalf("the local write stamped %x, want strictly greater than the listed %x — the listing part must reach Observe (PROTO-01)", written[:], listed[:])
	}

	// A held record above the sender's stale copy: the answer-not-Ack branch
	// queues its reply and never calls ApplyRemote, so the counters are
	// unchanged while the held record is still served.
	v5 := version.PackVersion(testNowMs, 5, peerID1)
	held := seedVersion(t, st, keyGamma, storage.Record{Version: v5, Data: []byte("five")})
	queuedBefore := core.QueuedResponses()
	stale := storage.Record{Version: version.PackVersion(testNowMs, 3, peerID1), Data: []byte("three")}
	core.HandleDatagram(now, replFrom(peerID1, keyGamma, stale), &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("the stale DataReplication emitted %d messages directly, want 0 — the answer queues", got)
	}
	if got := core.QueuedResponses(); got != queuedBefore+1 {
		t.Fatalf("QueuedResponses() = %d after the stale record, want %d — exactly one answer queues", got, queuedBefore+1)
	}
	if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
		t.Fatalf("after the stale record MergeStats() = %+v, want none — the stale branch never merges", got)
	}
	served, err := st.store.Get(keyGamma)
	if err != nil || served.Version != held.Version {
		t.Fatalf("store Get(%q) after the stale record = (%x, %v), want the held %x", keyGamma, served.Version[:], err, held.Version[:])
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
