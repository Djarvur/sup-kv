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
// seam under both definitions this phase settled.
//
// Conflicts (OBSV-03, D-05, ADR 0004) measures the displacement of a pending
// (unacked) LOCAL write by an applied replicated record — the core's pending
// set, never the store's Conflict slice, which reports every LWW outcome. An
// ordinary overwrite of a confirmed local or remote record, a stale
// rejection, an identical redelivery and a refused apply count zero; a
// strictly greater incoming and an equal-version tie-break win that displace
// a pending write count one and resolve that write as Superseded with its
// tracked version.
//
// DriftWarnings (OBSV-05) counts a received record whose version leads local
// wall time by more than drift_warn on its FIRST apply — the signal is the
// received reading's lead, not the clock's absorbed state. A record behind
// the wall counts nothing, and a record beyond max_clock_lead is discarded as
// bad_version before any store or clock call.
//
// Every row runs on a fresh core so one row cannot leak a pending entry or a
// counter into the next; the two definitions live in one test so a change to
// either is visible beside the other.
func TestMergeStats_counts(t *testing.T) {
	t.Parallel()

	t.Run("a pending write displaced by a strictly greater version counts one", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		now := at(0)
		var out port.Outbox

		// The local write goes through the real author sequence, so the
		// pending set holds it and the displacement is what counts.
		tracked := localWrite(t, core, st, now, keyAlpha, "local")
		greater := remoteRecord(testNowMs+5000, peerID1, "remote-wins")
		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, greater), &out)
		out.Drain()

		if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 0 {
			t.Fatalf("after the displacing apply MergeStats() = %+v, want Conflicts 1, DriftWarnings 0", got)
		}
		results := out.DrainResults()
		if len(results) != 1 || results[0].Kind != port.ResultSuperseded ||
			results[0].Key != keyAlpha || results[0].Version != tracked {
			t.Fatalf("results = %+v, want exactly one Superseded(%q, %x) — the displaced write's honest false", results, keyAlpha, tracked[:])
		}
		if _, trackedStill := core.PendingSnapshot()[keyAlpha]; trackedStill {
			t.Fatal("the displaced pending entry is still tracked, want it removed")
		}
		served, err := st.store.Get(keyAlpha)
		if err != nil || served.Version != greater.Version {
			t.Fatalf("store serves (%x, %v) after the apply, want the remote version %x", served.Version[:], err, greater.Version[:])
		}
	})

	t.Run("a confirmed local record overwritten by a newer version counts zero", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		now := at(0)
		var out port.Outbox

		// A peer must be known at write time for the write to be tracked, and
		// its Ack is what removes the entry: the write is then confirmed and
		// has no promise left to displace.
		core.HandleDatagram(now, helloFrom(peerID1), &out)
		out.Drain()
		tracked := localWrite(t, core, st, now, keyAlpha, "confirmed")
		core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, tracked), &out)
		out.Drain()
		out.DrainResults()
		if _, stillTracked := core.PendingSnapshot()[keyAlpha]; stillTracked {
			t.Fatal("the Acked write is still tracked, want the Ack to drain it")
		}

		greater := remoteRecord(testNowMs+5000, peerID1, "remote-wins")
		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, greater), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("MergeStats() = %+v after overwriting a confirmed local record, want no conflict — an Acked write has no promise left to displace", got)
		}
		served, err := st.store.Get(keyAlpha)
		if err != nil || served.Version != greater.Version {
			t.Fatalf("store serves (%x, %v) after the overwrite, want the remote version %x", served.Version[:], err, greater.Version[:])
		}
	})

	t.Run("a remote-to-remote overwrite counts zero", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		now := at(0)
		var out port.Outbox

		seedVersion(t, st, keyAlpha, storage.Record{Version: version.PackVersion(testNowMs, 3, peerID1), Data: []byte("three")})
		greater := remoteRecord(testNowMs+5000, peerID2, "newer")
		core.HandleDatagram(now, replFrom(peerID2, keyAlpha, greater), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("MergeStats() = %+v after overwriting a remote record, want no conflict — no local write was displaced", got)
		}
		served, err := st.store.Get(keyAlpha)
		if err != nil || served.Version != greater.Version {
			t.Fatalf("store serves (%x, %v) after the overwrite, want the remote version %x", served.Version[:], err, greater.Version[:])
		}
	})

	t.Run("an equal-version tie-break win displacing a pending write counts one", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		now := at(0)
		var out port.Outbox

		// The pending local record carries the smaller data; the incoming
		// record at the SAME version wins the content tie-break and installs.
		tracked := localWrite(t, core, st, now, keyAlpha, "abc")
		incoming := storage.Record{Version: tracked, Data: []byte("zzz")}
		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, incoming), &out)
		out.Drain()

		if got := core.MergeStats(); got.Conflicts != 1 || got.DriftWarnings != 0 {
			t.Fatalf("MergeStats() = %+v after an equal-version tie-break win over a pending write, want Conflicts 1", got)
		}
		results := out.DrainResults()
		if len(results) != 1 || results[0].Kind != port.ResultSuperseded || results[0].Version != tracked {
			t.Fatalf("results = %+v, want exactly one Superseded(%q, %x)", results, keyAlpha, tracked[:])
		}
		served, err := st.store.Get(keyAlpha)
		if err != nil || !bytes.Equal(served.Data, []byte("zzz")) {
			t.Fatalf("store serves (%q, %v) after the tie-break win, want the incoming %q", served.Data, err, "zzz")
		}
	})

	t.Run("an equal-version tie-break loss counts zero and keeps the pending entry", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		now := at(0)
		var out port.Outbox

		tracked := localWrite(t, core, st, now, keyAlpha, "zzz")
		incoming := storage.Record{Version: tracked, Data: []byte("aaa")}
		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, incoming), &out)
		out.Drain()

		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("MergeStats() = %+v after an equal-version tie-break loss, want no conflict — the pending write was not displaced", got)
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, tracked, nil)
		served, err := st.store.Get(keyAlpha)
		if err != nil || !bytes.Equal(served.Data, []byte("zzz")) {
			t.Fatalf("store serves (%q, %v) after the tie-break loss, want the stored %q", served.Data, err, "zzz")
		}
	})

	t.Run("an identical redelivery counts zero", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		now := at(0)
		var out port.Outbox

		tracked := localWrite(t, core, st, now, keyAlpha, "local")
		stored, err := st.store.Get(keyAlpha)
		if err != nil {
			t.Fatalf("store Get(%q) before the redelivery: %v", keyAlpha, err)
		}
		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, stored), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("MergeStats() = %+v after an identical redelivery, want no counters — the equal branch Acks without merging", got)
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, tracked, nil)
	})

	t.Run("a stale incoming record counts zero and keeps the pending entry", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		now := at(0)
		var out port.Outbox

		tracked := localWrite(t, core, st, now, keyAlpha, "local")
		stale := remoteRecord(testNowMs-5000, peerID1, "stale")
		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, stale), &out)
		if got := len(out.Drain()); got != 0 {
			t.Fatalf("the stale record emitted %d messages directly, want 0 — the answer queues", got)
		}
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("MergeStats() = %+v after a stale incoming record, want no counters — the stale branch never merges", got)
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, tracked, nil)
	})

	t.Run("a refused apply counts zero and removes nothing", func(t *testing.T) {
		t.Parallel()
		var logs []string
		core, st := newLimitedTestCore(8, &logs)
		now := at(0)
		var out port.Outbox

		// "local" fits the store's 8-byte value limit; the replicated record
		// is over it, so the apply is refused after validation. The stamp is
		// one drift window plus a second ahead — beyond drift_warn, so an
		// accepted reading WOULD count, but still inside the 2 m clock lead.
		// The zero DriftWarnings below is therefore the refusal's doing, not
		// the stamp's.
		tracked := localWrite(t, core, st, now, keyAlpha, "local")
		oversize := remoteRecord(testNowMs+int64(testDriftWarn/time.Millisecond)+1000, peerID1, "over-the-limit")
		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, oversize), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("MergeStats() = %+v after a refused apply, want no counters", got)
		}
		if got := len(out.DrainResults()); got != 0 {
			t.Fatalf("the refused apply emitted %d result(s), want none — no Ack and no supersession", got)
		}
		assertPending(t, core.PendingSnapshot(), keyAlpha, tracked, nil)
		served, err := st.store.Get(keyAlpha)
		if err != nil || served.Version != tracked {
			t.Fatalf("store serves (%x, %v) after the refused apply, want the untouched pending version %x", served.Version[:], err, tracked[:])
		}
	})

	t.Run("a first far-ahead apply counts one drift warning", func(t *testing.T) {
		t.Parallel()
		core, _ := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
		now := at(0)
		var out port.Outbox

		aheadMs := testNowMs + 5000 + int64(testDriftWarn/time.Millisecond) + 1000
		ahead := remoteRecord(aheadMs, peerID1, "from-the-future")
		core.HandleDatagram(now, replFrom(peerID1, keyBeta, ahead), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 1 {
			t.Fatalf("after the far-ahead apply MergeStats() = %+v, want Conflicts 0, DriftWarnings 1 on the first apply", got)
		}

		// The far-ahead redelivery is the equal-version branch: the version
		// was already examined, and its repeat counts nothing new.
		core.HandleDatagram(now, replFrom(peerID1, keyBeta, ahead), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 1 {
			t.Fatalf("after the far-ahead redelivery MergeStats() = %+v, want the unchanged DriftWarnings 1", got)
		}
	})

	t.Run("a behind-the-wall record counts no drift warning", func(t *testing.T) {
		t.Parallel()
		core, _ := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
		now := at(0)
		var out port.Outbox

		behind := remoteRecord(testNowMs-int64(testDriftWarn/time.Millisecond)-1000, peerID1, "behind-the-wall")
		core.HandleDatagram(now, replFrom(peerID1, keyGamma, behind), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("after the behind-the-wall apply MergeStats() = %+v, want no counters — only a positive lead is a warning", got)
		}
	})

	t.Run("a record beyond max_clock_lead counts neither counter", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCoreWithConfig(repl.Config{MaxClockLead: driftLead})
		now := at(0)
		var out port.Outbox

		beyond := remoteRecord(testNowMs+int64(driftLead/time.Millisecond)+1000, peerID1, "beyond-the-bound")
		core.HandleDatagram(now, replFrom(peerID1, keyBeta, beyond), &out)
		out.Drain()
		if got := core.MergeStats(); got.Conflicts != 0 || got.DriftWarnings != 0 {
			t.Fatalf("after the beyond-lead record MergeStats() = %+v, want no counters", got)
		}
		if got := core.Discards().BadVersion; got != 1 {
			t.Fatalf("Discards().BadVersion = %d after the beyond-lead record, want 1", got)
		}
		_, err := st.store.Get(keyBeta)
		if err == nil {
			t.Fatal("the beyond-lead record reached the store, want it discarded before any store call")
		}
	})
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
