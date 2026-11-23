package repl_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// driftLead is the inbound version bound this test's cores run with: the
// drift-warning scenario stamps a record one drift-warn window (60 s) plus a
// second ahead. The bound is set explicitly so the test does not depend on
// the binary's configuration, whose production default is also 2 m.
const driftLead = 2 * time.Minute

// TestMergeStats_counts proves the OBS-01 counter layer at the applyRemote
// seam under the received-lead definition (OBSV-05): a strictly greater
// remote version counts one conflict (the displaced local version — an LWW
// loser is an ordinary overwrite, counted and never journaled, D-10), an
// equal-version redelivery counts nothing (the equal branch Acks without
// merging, PROTO-03), and a record whose version leads local wall time by
// more than the store clock's drift-warn threshold counts one drift warning
// on its FIRST apply — the signal is the received reading's lead, not the
// clock's absorbed state. A record behind the wall counts no warning, and a
// record beyond max_clock_lead is discarded as bad_version before any store
// or clock call.
func TestMergeStats_counts(t *testing.T) {
	t.Parallel()
	core, st := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
	now := at(0)
	var out port.Outbox

	// The local write goes through the real author sequence so the pending
	// set holds it: this displacement is what plan 07-06's pending-only
	// conflict definition will count.
	localWrite(t, core, st, now, keyAlpha, "local")

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

	// A record stamped beyond drift_warn ahead of the wall counts its drift
	// on THIS apply: clock.Update compares the received reading with the
	// wall, so no prior absorption is needed (OBSV-05). keyBeta holds no
	// local record, so no conflict joins it.
	aheadMs := testNowMs + 5000 + int64(testDriftWarn/time.Millisecond) + 1000
	ahead := remoteRecord(aheadMs, peerID1, "from-the-future")
	core.HandleDatagram(now, replFrom(peerID1, keyBeta, ahead), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 1 {
		t.Fatalf("after the far-ahead apply MergeStats() = %+v, want Conflicts still 1, DriftWarnings 1 on the first apply", got)
	}

	// The far-ahead redelivery is the equal-version branch again: the
	// version was already examined, and its repeat counts nothing new.
	core.HandleDatagram(now, replFrom(peerID1, keyBeta, ahead), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 1 {
		t.Fatalf("after the far-ahead redelivery MergeStats() = %+v, want the unchanged Conflicts 1, DriftWarnings 1", got)
	}

	// A record whose version sits behind the wall beyond drift_warn is never
	// a warning: only a positive lead counts (OBSV-05). keyGamma holds no
	// local record, so the record still applies and counts no conflict.
	behind := remoteRecord(testNowMs-int64(testDriftWarn/time.Millisecond)-1000, peerID1, "behind-the-wall")
	core.HandleDatagram(now, replFrom(peerID1, keyGamma, behind), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 1 {
		t.Fatalf("after the behind-the-wall apply MergeStats() = %+v, want Conflicts still 1, DriftWarnings still 1", got)
	}

	// A record beyond max_clock_lead is discarded as bad_version by the
	// validation pipeline before any store or clock call, so it counts no
	// drift and no conflict (T-07-14).
	beyond := remoteRecord(testNowMs+int64(driftLead/time.Millisecond)+1000, peerID1, "beyond-the-bound")
	core.HandleDatagram(now, replFrom(peerID1, keyBeta, beyond), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 1 {
		t.Fatalf("after the beyond-lead record MergeStats() = %+v, want the unchanged Conflicts 1, DriftWarnings 1", got)
	}
	if got := core.Discards().BadVersion; got != 1 {
		t.Fatalf("Discards().BadVersion = %d after the beyond-lead record, want 1", got)
	}
}

// TestMergeStats_equalVersionTieBreakCountsDrift proves the tie-break half
// of the drift contract: an equal-version record that loses the content
// tie-break still reaches ApplyRemote, so a version leading the wall by more
// than drift_warn is examined and counted (OBSV-05). The stored record wins
// the tie-break, so the counter moves on the drift alone.
func TestMergeStats_equalVersionTieBreakCountsDrift(t *testing.T) {
	t.Parallel()
	core, st := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
	now := at(0)
	var out port.Outbox

	// A leading remote reading is absorbed first (under another key, through
	// the store's merge, so it creates no pending entry): the clock's next
	// local stamp then leads the wall by the same window, and the tie-break
	// row below can compare an equal-version record beyond drift_warn.
	leadMs := int64(testDriftWarn/time.Millisecond) + 1000
	seedVersion(t, st, keyBeta, storage.Record{Version: version.PackVersion(testNowMs+leadMs, 0, peerID1), Data: []byte("seed")})

	// The local record carries the lexicographically greater data and the
	// leading stamp; it goes through the real author sequence so the pending
	// set holds it.
	held := localWrite(t, core, st, now, keyAlpha, "zzz")
	if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
		t.Fatalf("after the local write MergeStats() = %+v, want none", got)
	}

	// The equal-version record loses the tie-break ("aaa" < "zzz") but its
	// received reading leads the wall beyond drift_warn, so the drift is
	// counted.
	loser := storage.Record{Version: held, Data: []byte("aaa")}
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, loser), &out)
	out.Drain()
	if got := core.MergeStats().DriftWarnings; got != 1 {
		t.Fatalf("DriftWarnings = %d after an equal-version tie-break loss with a leading stamp, want 1", got)
	}
	stored, err := st.store.Get(keyAlpha)
	if err != nil || !bytes.Equal(stored.Data, []byte("zzz")) {
		t.Fatalf("store serves (%q, %v) after the tie-break loss, want the stored %q", stored.Data, err, "zzz")
	}
}

// TestMergeStats_listingAndStaleExcluded pins the DataSync branch and the
// non-merge path. A listing part's maximum entry version is a received
// reading: when it leads local wall time by more than drift_warn the part
// counts exactly ONE drift warning — never one per offending entry — while
// the absorption stays real and visible (the next local write strictly beats
// the listed state, PROTO-01). A strictly older DataReplication takes the
// answer-not-Ack branch, which returns before ApplyRemote, so it counts
// nothing (PROTO-03): the counters describe merges, not answers.
func TestMergeStats_listingAndStaleExcluded(t *testing.T) {
	t.Parallel()
	core, st := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
	now := at(0)
	var out port.Outbox

	// A complete one-part listing whose entries both sit one drift-warn
	// window plus a second past the wall clock — every entry would count a
	// DriftWarning if the counter ran per entry. Its Hello comes first: a
	// received listing is bound to a learned peer identity (CR-01).
	past := testNowMs + 5000 + int64(testDriftWarn/time.Millisecond) + 1000
	listed := version.PackVersion(past, 0, peerID1)
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	out.Drain()
	core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{
		SyncID:     1,
		ChunkIndex: 1,
		ChunkCount: 1,
		Entries: []port.SyncEntry{
			{Key: keyAlpha, Version: listed},
			{Key: keyBeta, Version: version.PackVersion(past-1000, 0, peerID1)},
		},
	}), &out)
	out.Drain()
	if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 1 {
		t.Fatalf("after the listing part MergeStats() = %+v, want Conflicts 0, DriftWarnings 1 — one per offending part, never per entry (OBSV-05)", got)
	}

	// The absorption is real, not skipped: the next local write strictly
	// beats the listed max version, so the part reached Observe.
	written := localWrite(t, core, st, now, keyAlpha, "after-the-listing")
	if written.Compare(listed) <= 0 {
		t.Fatalf("the local write stamped %x, want strictly greater than the listed %x — the listing part must reach Observe (PROTO-01)", written[:], listed[:])
	}
	if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 1 {
		t.Fatalf("after the local write MergeStats() = %+v, want the unchanged Conflicts 0, DriftWarnings 1", got)
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
	if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 1 {
		t.Fatalf("after the stale record MergeStats() = %+v, want the unchanged Conflicts 0, DriftWarnings 1 — the stale branch never merges", got)
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
