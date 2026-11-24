package repl_test

// The XPORT-06 key-count cost gate is a deterministic operation count: the
// two inbound sync handlers must make a fixed number of consumer-seam store
// calls per datagram at 50 000 stored keys exactly as at 1 000. Counting
// operations instead of nanoseconds removes the host-load sensitivity the
// wall-clock ratio gate had; the ns/op benchmarks in this file remain
// measurement only (`make bench`/`make perf`), never the assertion.

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
		// The seeded version packs the key index into the counter field; the
		// gate below pins the version domain the fixture must stay inside
		// (03-REVIEW IN-04).
		ver := version.PackVersion(testNowMs, uint16(i), localID)
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
	b.ReportMetric(float64(keys), "store-keys")
	b.ResetTimer()
	benchHandleDataRequestBody(core, in, base, b.N)
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
	b.ReportMetric(float64(keys), "store-keys")
	b.ResetTimer()
	benchHandleDataSyncBody(core, in, base, b.N)
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
// the event-loop iteration cost at 10 and 10 000 unacked records may differ
// by at most 2x.
const (
	benchSmallUnacked = 10
	benchLargeUnacked = 10_000

	// benchUnackedMaxRatio is XPORT-02(1)'s hard acceptance bound, never
	// relaxed.
	benchUnackedMaxRatio = 2.0
)

// benchUnackedFixture builds the core under measurement with unacked pending
// entries from one author and one known peer, so the retransmit scanner is
// armed while every retry deadline stays in the future: the measured Advance
// never emits, and the pending entries hold no Data by construction.
func benchUnackedFixture(tb testing.TB, unacked int) *repl.Core {
	tb.Helper()
	core, _ := newTestCore()
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
	return core
}

// BenchmarkCoreEventLoop measures one idle event-loop iteration — Advance,
// UnackedStats and PeerCount — over pending sets of 10 and 10 000 records.
// The iteration is deliberately at a fixed virtual instant with every retry
// deadline in the future, so the measured work is the timer/deadline/gauges
// path only: no retransmission, no egress drain, no record lookup.
func BenchmarkCoreEventLoop(b *testing.B) {
	for _, unacked := range []int{benchSmallUnacked, benchLargeUnacked} {
		b.Run(fmt.Sprintf("unacked=%d", unacked), func(b *testing.B) {
			benchCoreEventLoop(b, unacked)
		})
	}
}

// benchCoreEventLoop is the shared body of the event-loop benchmark and the
// flatness gate's measurement.
func benchCoreEventLoop(b *testing.B, unacked int) {
	b.Helper()
	core := benchUnackedFixture(b, unacked)
	now := at(time.Nanosecond)
	var out port.Outbox
	b.ReportAllocs()
	b.ReportMetric(float64(unacked), "unacked")
	b.ResetTimer()
	for range b.N {
		core.Advance(now, &out)
		out.Drain()
		_ = core.UnackedStats(now)
		_ = core.PeerCount()
	}
}

// TestCore_unackedCostFlat is XPORT-02(1)'s executable acceptance: one
// event-loop iteration costs at most 2x more at 10 000 unacked records than
// at 10. Warm-up runs precede every measurement; a zero measurement fails by
// name instead of skipping, and the failure names both ns/op values and the
// ratio. The test stays sequential (no t.Parallel): it is a wall-clock ratio
// gate, and running it beside the sibling benchmark test would measure
// scheduler contention instead of the core.
//
//nolint:paralleltest // Wall-clock ratio gate: a parallel sibling benchmark would contaminate the measurement.
func TestCore_unackedCostFlat(t *testing.T) {
	measure := func(unacked int) float64 {
		t.Helper()
		testing.Benchmark(func(b *testing.B) {
			b.Helper()
			benchCoreEventLoop(b, unacked)
		}) // warm-up
		result := testing.Benchmark(func(b *testing.B) {
			b.Helper()
			benchCoreEventLoop(b, unacked)
		})
		if result.N == 0 {
			t.Fatalf("unacked=%d: benchmark ran zero iterations", unacked)
		}
		nsPerOp := float64(result.NsPerOp())
		if nsPerOp <= 0 {
			t.Fatalf("unacked=%d: measured %v ns/op, want a positive measurement", unacked, result.NsPerOp())
		}
		return nsPerOp
	}
	small := measure(benchSmallUnacked)
	large := measure(benchLargeUnacked)
	ratio := large / small
	t.Logf("core event loop: unacked=%d = %.1f ns/op, unacked=%d = %.1f ns/op, ratio %.2f",
		benchSmallUnacked, small, benchLargeUnacked, large, ratio)
	if ratio > benchUnackedMaxRatio {
		t.Fatalf("core event loop: unacked=%d = %.1f ns/op vs unacked=%d = %.1f ns/op (ratio %.2f > %.1f) — iteration cost scales with the unacked set (XPORT-02)",
			benchLargeUnacked, large, benchSmallUnacked, small, ratio, benchUnackedMaxRatio)
	}
}
