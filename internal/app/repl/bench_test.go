package repl_test

// The XPORT-06/PERF-05 cost gates in this file are deterministic operation
// counts: the inbound sync handlers must make a fixed number of consumer-seam
// store calls per datagram at 50 000 stored keys exactly as at 1 000, an idle
// event-loop iteration must inspect zero pending entries at any pending-set
// size, and a fresh-key local write or inbound replication must inspect zero
// too. Counting operations instead of nanoseconds removes the host-load
// sensitivity the two old wall-clock ratio gates had; the ns/op benchmarks in
// this file remain measurement only (`make bench`/`make perf`), never the
// assertion.

import (
	"fmt"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

const (
	// benchSmallKeys and benchLargeKeys are XPORT-06's acceptance pair.
	benchSmallKeys = 1_000
	benchLargeKeys = 50_000

	// benchChunkEntries is the fixed DataSync chunk size both store sizes
	// feed; the plan's assumption pins one fixed entry count per call.
	benchChunkEntries = 16

	// benchSyncRate pins the egress limiter to the top of its valid domain
	// (1e9/s, the maxSyncResponseRate cap), so one VirtualTime step of 1ns
	// is exactly one limiter slot and each Advance drains the one response
	// the handler just queued. Hello and sync timers stay far ahead of the
	// measured window at this rate.
	benchSyncRate = 1_000_000_000

	// benchDataBytes is the seeded value size: small enough that seeding
	// 50 000 records stays cheap, large enough to be a real record.
	benchDataBytes = 16

	// benchIterations is the fixed operation count the key-count gate runs:
	// large enough that a per-key loop over the store would be visible in the
	// count, small enough that the gate finishes in about a second under the
	// race detector.
	benchIterations = 1_000
)

// benchFixture builds the core under measurement over a store seeded with
// keys records through Merge — explicitly versioned, sorted-key seeds, so
// the fixture never burns 50 000 clock ticks. Construction runs once per
// benchmark invocation, outside the measured loop; the fixture learns the
// benchmark's peer identity (the CR-01 listing binding) and drains every
// startup/listing emission, returning the instant the drain stopped at so no
// queued datagram lands inside a measurement. The returned entries are the
// first benchChunkEntries seeded records, the fixed DataSync chunk the
// benchmark feeds back.
func benchFixture(tb testing.TB, keys int) (*repl.Core, []port.SyncEntry, time.Time) {
	tb.Helper()
	return benchFixtureWith(tb, keys, func(seam *stateStore) repl.StateStore { return seam })
}

// benchFixtureWith is benchFixture over a consumer-seam wrapper: the
// counting double installs here, so no production surface is added for it and
// the gate and the benchmark still drive one fixture. The wrapper is applied
// after the store is seeded, so a gate snapshots its counters after
// construction and never attributes the fixture's own listing calls to the
// measured datagram.
func benchFixtureWith(tb testing.TB, keys int, wrap func(*stateStore) repl.StateStore) (*repl.Core, []port.SyncEntry, time.Time) {
	tb.Helper()
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	store := storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})
	head := make([]port.SyncEntry, 0, benchChunkEntries)
	for i := range keys {
		key := fmt.Sprintf("bench/key-%06d", i)
		// A legal version domain: the millisecond offset goes in the l field
		// and the 12-bit counter stays 0 (03-REVIEW IN-04). The 50 000-key
		// seed spans 49 999 ms, inside the 60 s drift-warn default, so the
		// fixture's readings never look like a drifted peer.
		ver := version.PackVersion(testNowMs+int64(i), 0, localID)
		rec := storage.Record{Version: ver, Data: make([]byte, benchDataBytes)}
		_, _, err := store.Merge(key, rec)
		if err != nil {
			tb.Fatalf("seed Merge(%q): %v", key, err)
		}
		if i < benchChunkEntries {
			head = append(head, port.SyncEntry{Key: key, Version: ver})
		}
	}
	core := repl.New(localID, testRunID(localID),
		wrap(&stateStore{store: store}), repl.Config{SyncResponseRate: benchSyncRate})
	var out port.Outbox
	core.HandleDatagram(at(0), benchDatagram(port.Hello{}), &out)
	out.Drain()
	now := at(0)
	for core.QueuedSends() > 0 {
		now = now.Add(time.Nanosecond) // one limiter slot at the pinned 1e9/s rate
		core.Advance(now, &out)
		out.Drain()
	}
	return core, head, now
}

// benchDatagram wraps msg in a decoded datagram from peerID1 — the hand-built
// port input the handlers consume (D-01: the structs cross the seam).
func benchDatagram(msg port.Message) port.InboundDatagram {
	return port.InboundDatagram{
		Source:       handleFor(peerID1),
		Status:       port.Decoded,
		ProtoVersion: repl.SupportedProtoVersion,
		NodeID:       peerID1,
		RunID:        testRunID(peerID1),
		Msg:          msg,
	}
}

// benchHandleDataRequestBody drives n measured iterations of one DataRequest
// datagram: HandleDatagram, the same-instant Advance that drains the reply,
// and an Outbox drain. The benchmark and the key-count gate call this one
// body, so the gate measures exactly the path the benchmark reports.
func benchHandleDataRequestBody(core *repl.Core, in port.InboundDatagram, base time.Time, n int) {
	var out port.Outbox
	for i := range n {
		now := base.Add(time.Duration(i+1) * time.Nanosecond)
		core.HandleDatagram(now, in, &out)
		core.Advance(now, &out)
		out.Drain()
	}
}

// benchHandleDataSyncBody drives n measured iterations of one DataSync
// datagram carrying the fixed chunk: HandleDatagram, the same-instant Advance,
// and an Outbox drain. It is the single body of the DataSync benchmark and
// the key-count gate.
func benchHandleDataSyncBody(core *repl.Core, in port.InboundDatagram, base time.Time, n int) {
	var out port.Outbox
	for i := range n {
		now := base.Add(time.Duration(i+1) * time.Nanosecond)
		core.HandleDatagram(now, in, &out)
		core.Advance(now, &out)
		out.Drain()
	}
}

// BenchmarkHandleDataRequest measures one DataRequest handler call against
// stores of 1 000 and 50 000 keys. Each iteration steps virtual time one
// limiter slot so Advance drains the reply the handler enqueued; the key is
// a real stored key, so the reply path is measured, never the absent-key
// silence. The ns/op number is measurement only: the flatness claim is the
// operation count in TestCore_keyCountCostFlat.
func BenchmarkHandleDataRequest(b *testing.B) {
	for _, keys := range []int{benchSmallKeys, benchLargeKeys} {
		b.Run(fmt.Sprintf("keys=%d", keys), func(b *testing.B) {
			benchHandleDataRequest(b, keys)
		})
	}
}

// benchHandleDataRequest is the shared setup of the DataRequest benchmark and
// the flatness gate's measurement.
func benchHandleDataRequest(b *testing.B, keys int) {
	b.Helper()
	core, head, base := benchFixture(b, keys)
	in := benchDatagram(port.DataRequest{Key: head[0].Key})
	b.ReportAllocs()
	b.ResetTimer()
	benchHandleDataRequestBody(core, in, base, b.N)
	// ReportMetric after ResetTimer: ResetTimer clears the extra metrics, so a
	// metric reported before it would silently vanish from the result line.
	b.ReportMetric(float64(keys), "store-keys")
}

// BenchmarkHandleDataSync measures one DataSync handler call carrying the
// same fixed 16-entry chunk against stores of 1 000 and 50 000 keys. The
// entries carry the local versions, so the per-entry comparison is the whole
// measured cost; no assertion lives in the loop.
func BenchmarkHandleDataSync(b *testing.B) {
	for _, keys := range []int{benchSmallKeys, benchLargeKeys} {
		b.Run(fmt.Sprintf("keys=%d", keys), func(b *testing.B) {
			benchHandleDataSync(b, keys)
		})
	}
}

// benchHandleDataSync is the shared setup of the DataSync benchmark and the
// flatness gate's measurement.
func benchHandleDataSync(b *testing.B, keys int) {
	b.Helper()
	core, head, base := benchFixture(b, keys)
	// The 1-of-1 completeness header keeps the datagram on the compare path:
	// without it the handler would reject the fixture as a malformed listing
	// and the benchmark would measure the discard branch.
	in := benchDatagram(port.DataSync{SyncID: 1, ChunkIndex: 1, ChunkCount: 1, Entries: head})
	b.ReportAllocs()
	b.ResetTimer()
	benchHandleDataSyncBody(core, in, base, b.N)
	b.ReportMetric(float64(keys), "store-keys")
}

// TestCore_keyCountCostFlat is XPORT-06's deterministic acceptance: each
// inbound handler makes the same, exactly expected number of consumer-seam
// store calls per datagram at 1 000 and at 50 000 stored keys. A DataRequest
// is one Version probe plus the one Lookup the egress drain takes; a DataSync
// part is one Observe plus one Version per chunk entry. Versions, Records and
// ApplyRemote must stay at zero: a whole-store read or a per-key apply loop is
// exactly the O(store)-per-datagram cost XPORT-06 removed. No
// testing.Benchmark and no wall clock take part, so host load cannot move the
// verdict.
func TestCore_keyCountCostFlat(t *testing.T) {
	t.Parallel()
	handlers := []struct {
		name string
		want storeCounts
		in   func([]port.SyncEntry) port.InboundDatagram
		body func(*repl.Core, port.InboundDatagram, time.Time, int)
	}{
		{
			name: "HandleDatagram(DataRequest)",
			want: storeCounts{version: benchIterations, lookup: benchIterations},
			in: func(head []port.SyncEntry) port.InboundDatagram {
				return benchDatagram(port.DataRequest{Key: head[0].Key})
			},
			body: benchHandleDataRequestBody,
		},
		{
			name: "HandleDatagram(DataSync)",
			want: storeCounts{version: benchChunkEntries * benchIterations, observe: benchIterations},
			in: func(head []port.SyncEntry) port.InboundDatagram {
				return benchDatagram(port.DataSync{SyncID: 1, ChunkIndex: 1, ChunkCount: 1, Entries: head})
			},
			body: benchHandleDataSyncBody,
		},
	}
	for _, handler := range handlers {
		small := measureStoreCalls(t, handler.name, benchSmallKeys, handler.in, handler.body)
		large := measureStoreCalls(t, handler.name, benchLargeKeys, handler.in, handler.body)
		assertStoreCalls(t, handler.name, benchSmallKeys, small, handler.want)
		assertStoreCalls(t, handler.name, benchLargeKeys, large, handler.want)
		if small != large {
			t.Fatalf("%s: %d stored keys made %+v seam calls, %d stored keys made %+v — the vector must be identical at both sizes; a difference is a per-key loop over the store (XPORT-06)",
				handler.name, benchSmallKeys, small, benchLargeKeys, large)
		}
	}
}

// measureStoreCalls builds the counting fixture at keys stored keys, runs the
// shared handler body for benchIterations operations and returns the seam
// calls those operations made. The snapshot is taken after fixture
// construction, so the fixture's own listing calls are excluded.
func measureStoreCalls(t *testing.T, name string, keys int, in func([]port.SyncEntry) port.InboundDatagram, body func(*repl.Core, port.InboundDatagram, time.Time, int)) storeCounts {
	t.Helper()
	var counter *countingStore
	core, head, base := benchFixtureWith(t, keys, func(seam *stateStore) repl.StateStore {
		counter = &countingStore{stateStore: seam}
		return counter
	})
	if counter == nil {
		t.Fatalf("%s at %d stored keys: the fixture did not install the counting store", name, keys)
	}
	assertSeededVersions(t, name, keys, head)
	before := counter.counts()
	body(core, in(head), base, benchIterations)
	return diffStoreCounts(counter.counts(), before)
}

// assertStoreCalls fails naming the handler, the store method counts, the
// got and want vectors and the key count.
func assertStoreCalls(t *testing.T, name string, keys int, got, want storeCounts) {
	t.Helper()
	if got != want {
		t.Fatalf("%s at %d stored keys: seam calls = %+v, want %+v — one datagram must make a fixed number of store calls, never a per-key loop or a whole-store read (XPORT-06)",
			name, keys, got, want)
	}
}

// diffStoreCounts subtracts the construction snapshot from the measured one,
// so only the measured datagrams' calls remain.
func diffStoreCounts(after, before storeCounts) storeCounts {
	return storeCounts{
		version:     after.version - before.version,
		lookup:      after.lookup - before.lookup,
		versions:    after.versions - before.versions,
		records:     after.records - before.records,
		applyRemote: after.applyRemote - before.applyRemote,
		observe:     after.observe - before.observe,
		generation:  after.generation - before.generation,
	}
}

// assertSeededVersions pins the fixture's version domain: every seeded
// version packs its index as a millisecond offset with counter 0 (03-REVIEW
// IN-04), and the largest 50 000-key seed stays inside the 60 s drift-warn
// default. The head covers the first chunk; the counter-field bug fails on
// its second entry.
func assertSeededVersions(t *testing.T, name string, keys int, head []port.SyncEntry) {
	t.Helper()
	for i, entry := range head {
		l, counter, node := entry.Version.Unpack()
		if counter != 0 || l != testNowMs+int64(i) || node != localID {
			t.Fatalf("%s at %d stored keys: seeded %q as (l=%d counter=%d node=%d), want (l=%d counter=0 node=%d) — the fixture must stay inside the version domain (03-REVIEW IN-04)",
				name, keys, entry.Key, l, counter, node, testNowMs+int64(i), localID)
		}
	}
	if lead := time.Duration(benchLargeKeys-1) * time.Millisecond; lead >= testDriftWarn {
		t.Fatalf("%s: the 50 000-key seed lead %v must stay under the %v drift-warn default", name, lead, testDriftWarn)
	}
}

// benchSmallUnacked and benchLargeUnacked are XPORT-02(1)'s acceptance pair:
// the pending-entry inspection count per operation must be identical at 10 and
// at 10 000 unacked records.
const (
	benchSmallUnacked = 10
	benchLargeUnacked = 10_000
)

// newBenchPendingCore builds a core over a real store with unacked entries
// from one author and one known peer, so the retransmit scanner is armed
// while every retry deadline stays in the future: an idle Advance never
// emits, and the entries hold no Data by construction. The peer also lets the
// write path broadcast, which is the shape the write and apply gates measure.
func newBenchPendingCore(tb testing.TB, unacked int) (*repl.Core, *stateStore) {
	tb.Helper()
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(at(0), &out) // startup Hello + listing, before the measured loop
	out.Drain()
	core.HandleDatagram(at(0), helloFrom(peerID1), &out)
	out.Drain()
	for i := range unacked {
		key := fmt.Sprintf("bench/unacked-%06d", i)
		rec := storage.Record{Version: version.PackVersion(testNowMs+int64(i), 0, localID)}
		core.NotifyLocalWrite(at(0), key, rec, core.PeerIDs(), &out)
		out.Drain()
	}
	return core, st
}

// benchUnackedFixture is newBenchPendingCore when only the core is needed.
func benchUnackedFixture(tb testing.TB, unacked int) *repl.Core {
	tb.Helper()
	core, _ := newBenchPendingCore(tb, unacked)
	return core
}

// BenchmarkCoreEventLoop measures one idle event-loop iteration — Advance,
// UnackedStats and PeerCount — over pending sets of 10 and 10 000 records.
// The iteration is deliberately at a fixed virtual instant with every retry
// deadline in the future, so the measured work is the timer/deadline/gauges
// path only: no retransmission, no egress drain, no record lookup. The ns/op
// number is measurement only: the flatness claim is the operation count in
// TestCore_unackedCostFlat.
func BenchmarkCoreEventLoop(b *testing.B) {
	for _, unacked := range []int{benchSmallUnacked, benchLargeUnacked} {
		b.Run(fmt.Sprintf("unacked=%d", unacked), func(b *testing.B) {
			benchCoreEventLoop(b, unacked)
		})
	}
}

// benchCoreEventLoop is the shared body of the event-loop benchmark and the
// idle flatness gate's fixture.
func benchCoreEventLoop(b *testing.B, unacked int) {
	b.Helper()
	core := benchUnackedFixture(b, unacked)
	now := at(time.Nanosecond)
	var out port.Outbox
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		core.Advance(now, &out)
		out.Drain()
		_ = core.UnackedStats(now)
		_ = core.PeerCount()
	}
	// ReportMetric after ResetTimer: ResetTimer clears the extra metrics, so a
	// metric reported before it would silently vanish from the result line.
	b.ReportMetric(float64(unacked), "unacked")
}

// TestCore_unackedCostFlat is XPORT-02(1)'s deterministic acceptance: an idle
// event-loop iteration inspects zero pending entries, at 10 and at 10 000
// pending records. Advance pops only due heap items, so the count is flat by
// construction; the gate makes a future whole-set scan on the timer path fail
// by name instead of merely costing nanoseconds. No testing.Benchmark and no
// wall clock are involved, so host load cannot move the verdict.
func TestCore_unackedCostFlat(t *testing.T) {
	t.Parallel()
	small := idleAdvanceVisits(t, benchSmallUnacked)
	large := idleAdvanceVisits(t, benchLargeUnacked)
	assertZeroVisits(t, "an idle event-loop iteration", benchSmallUnacked, benchIterations, small)
	assertZeroVisits(t, "an idle event-loop iteration", benchLargeUnacked, benchIterations, large)
	if small != large {
		t.Fatalf("an idle event-loop iteration inspected %d pending entries at %d unacked and %d at %d, want the identical count — the timer path must not scale with the pending set (XPORT-02)",
			small, benchSmallUnacked, large, benchLargeUnacked)
	}
}

// idleAdvanceVisits runs benchIterations idle Advance+Drain iterations at one
// fixed virtual instant — every retry deadline is still in the future — and
// returns the pending-entry inspection delta.
func idleAdvanceVisits(t *testing.T, unacked int) uint64 {
	t.Helper()
	core := benchUnackedFixture(t, unacked)
	now := at(time.Nanosecond)
	before := core.PendingVisits()
	var out port.Outbox
	for range benchIterations {
		core.Advance(now, &out)
		out.Drain()
	}
	return core.PendingVisits() - before
}

// TestCore_notifyWriteCostFlat is the write-path tripwire: benchIterations
// fresh-key local writes inspect zero pending entries at both set sizes while
// the pending count grows by exactly benchIterations. It passes on arrival —
// 08-02's TestCore_pendingVisitsFreshKeyWrite is the RED evidence for this
// path — and exists to stop a whole-set scan from returning on the write
// path.
func TestCore_notifyWriteCostFlat(t *testing.T) {
	t.Parallel()
	small := notifyWriteVisits(t, benchSmallUnacked)
	large := notifyWriteVisits(t, benchLargeUnacked)
	assertZeroVisits(t, "a fresh-key local write", benchSmallUnacked, benchIterations, small)
	assertZeroVisits(t, "a fresh-key local write", benchLargeUnacked, benchIterations, large)
	if small != large {
		t.Fatalf("a fresh-key local write inspected %d pending entries at %d unacked and %d at %d, want the identical count — the write path must be key-local at any set size (XPORT-02/PERF-05)",
			small, benchSmallUnacked, large, benchLargeUnacked)
	}
}

// notifyWriteVisits drives benchIterations fresh-key writes through
// NotifyLocalWrite and returns the pending-entry inspection delta, failing
// unless the pending count grew by exactly that many entries.
func notifyWriteVisits(t *testing.T, unacked int) uint64 {
	t.Helper()
	core := benchUnackedFixture(t, unacked)
	beforeVisits := core.PendingVisits()
	beforeCount := core.UnackedStats(at(0)).Count
	var out port.Outbox
	for i := range benchIterations {
		key := fmt.Sprintf("bench/fresh-%06d", i)
		rec := storage.Record{Version: version.PackVersion(testNowMs+int64(i), 0, localID)}
		core.NotifyLocalWrite(at(0), key, rec, core.PeerIDs(), &out)
		out.Drain()
	}
	if got := core.UnackedStats(at(0)).Count - beforeCount; got != benchIterations {
		t.Fatalf("benchIterations fresh-key writes over %d unacked records grew the pending count by %d, want %d", unacked, got, benchIterations)
	}
	return core.PendingVisits() - beforeVisits
}

// TestCore_applyRemoteCostFlat is the apply-path tripwire: benchIterations
// inbound DataReplication datagrams for fresh keys inspect zero pending
// entries at both set sizes, and every apply installs its record. It passes
// on arrival — 08-02's TestCore_pendingVisitsFreshKeyApply is the RED
// evidence for this path — and exists to stop the whole-set displacement scan
// from returning.
func TestCore_applyRemoteCostFlat(t *testing.T) {
	t.Parallel()
	small := applyRemoteVisits(t, benchSmallUnacked)
	large := applyRemoteVisits(t, benchLargeUnacked)
	assertZeroVisits(t, "an inbound replication for a fresh key", benchSmallUnacked, benchIterations, small)
	assertZeroVisits(t, "an inbound replication for a fresh key", benchLargeUnacked, benchIterations, large)
	if small != large {
		t.Fatalf("an inbound replication for a fresh key inspected %d pending entries at %d unacked and %d at %d, want the identical count — the apply path must be key-local at any set size (XPORT-02/PERF-05)",
			small, benchSmallUnacked, large, benchLargeUnacked)
	}
}

// applyRemoteVisits drives benchIterations wire-valid DataReplication
// datagrams for fresh keys through the production entry point and returns the
// pending-entry inspection delta. replFrom recomputes the record checksum, so
// the validation pipeline accepts each datagram instead of discarding it as
// malformed, and the store read proves the apply really landed.
func applyRemoteVisits(t *testing.T, unacked int) uint64 {
	t.Helper()
	core, st := newBenchPendingCore(t, unacked)
	before := core.PendingVisits()
	var out port.Outbox
	for i := range benchIterations {
		key := fmt.Sprintf("bench/apply-%06d", i)
		ver := version.PackVersion(testNowMs+int64(unacked)+int64(i)+1, 0, peerID1)
		core.HandleDatagram(at(0), replFrom(peerID1, key, storage.Record{Version: ver, Data: []byte("v")}), &out)
		out.Drain()
		installed, err := st.store.Get(key)
		if err != nil || installed.Version != ver {
			t.Fatalf("apply %d of %q over %d unacked records: store holds (%x, %v), want the applied version %x", i, key, unacked, installed.Version[:], err, ver[:])
		}
	}
	return core.PendingVisits() - before
}

// TestCore_pendingVisitsRepeatedKey proves the visit counter is connected to
// the real write loops: a newer-version repeated write inspects exactly two
// pending entries — the tracked-version guard read and the supersession read —
// and documents the only nonzero hot-path count.
func TestCore_pendingVisitsRepeatedKey(t *testing.T) {
	t.Parallel()
	core := benchUnackedFixture(t, benchSmallUnacked)
	key := "bench/repeat"
	var out port.Outbox
	older := version.PackVersion(testNowMs, 0, localID)
	core.NotifyLocalWrite(at(0), key, storage.Record{Version: older, Data: []byte("one")}, core.PeerIDs(), &out)
	out.Drain()
	before := core.PendingVisits()
	newer := version.PackVersion(testNowMs+1, 0, localID)
	core.NotifyLocalWrite(at(0), key, storage.Record{Version: newer, Data: []byte("two")}, core.PeerIDs(), &out)
	out.Drain()
	if got := core.PendingVisits() - before; got != 2 {
		t.Fatalf("a newer-version repeated write over %d unacked records inspected %d pending entries, want exactly 2 (the tracked-version guard read plus the supersession read) — a count of 0 would mean the counter is not wired to the real write loops",
			benchSmallUnacked, got)
	}
}

// assertZeroVisits fails naming the claim, the operation count, the set size
// and the observed pending-entry inspections.
func assertZeroVisits(t *testing.T, claim string, unacked, ops int, got uint64) {
	t.Helper()
	if got != 0 {
		t.Fatalf("%s over %d unacked records inspected %d pending entries across %d operations, want 0 (XPORT-02/PERF-05: the path must be key-local at any set size)",
			claim, unacked, got, ops)
	}
}
