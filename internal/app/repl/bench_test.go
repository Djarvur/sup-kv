package repl_test

// The XPORT-06 key-count cost gate (D-08's core-consumption half): the two
// inbound sync handlers must cost the same at 50 000 stored keys as at
// 1 000. The benchmarks build their fixture once outside the measured loop
// (the internal/app/persist/bench_test.go shape) and step virtual time by
// one limiter slot per iteration so the core's egress queue drains at the
// same rate the handler fills it. TestCore_keyCountCostFlat turns the 2x
// acceptance into an executable gate: it measures both sub-benchmark sizes
// with testing.Benchmark and fails naming both ns/op values and the ratio —
// never a hand measurement.

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

	// benchMaxRatio is XPORT-06's hard acceptance bound, never relaxed.
	benchMaxRatio = 2.0

	// benchSyncRate pins the egress limiter to the top of its valid domain
	// (1e9/s, the maxSyncResponseRate cap), so one VirtualTime step of 1ns
	// is exactly one limiter slot and each Advance drains the one response
	// the handler just queued. Hello and sync timers stay far ahead of the
	// measured window at this rate.
	benchSyncRate = 1_000_000_000

	// benchDataBytes is the seeded value size: small enough that seeding
	// 50 000 records stays cheap, large enough to be a real record.
	benchDataBytes = 16
)

// benchFixture builds the core under measurement over a store seeded with
// keys records through Merge — explicitly versioned, sorted-key seeds, so
// the fixture never burns 50 000 clock ticks. Construction runs once per
// benchmark invocation, outside the measured loop; the core's startup
// timers are armed here so no listing or Hello lands inside a measurement.
// The returned entries are the first benchChunkEntries seeded records, the
// fixed DataSync chunk the benchmark feeds back.
func benchFixture(tb testing.TB, keys int) (*repl.Core, []port.SyncEntry) {
	tb.Helper()
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	store := storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})
	head := make([]port.SyncEntry, 0, benchChunkEntries)
	for i := range keys {
		key := fmt.Sprintf("bench/key-%06d", i)
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
	core := repl.New(localID, testRunID(localID), handleFor(localID),
		&stateStore{store: store}, repl.Config{SyncResponseRate: benchSyncRate})
	var out port.Outbox
	core.Advance(at(0), &out) // startup Hello + listing, before the measured loop
	out.Drain()
	return core, head
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

// BenchmarkHandleDataRequest measures one DataRequest handler call against
// stores of 1 000 and 50 000 keys. Each iteration steps virtual time one
// limiter slot so Advance drains the reply the handler enqueued; the key is
// a real stored key, so the reply path is measured, never the absent-key
// silence.
func BenchmarkHandleDataRequest(b *testing.B) {
	for _, keys := range []int{benchSmallKeys, benchLargeKeys} {
		b.Run(fmt.Sprintf("keys=%d", keys), func(b *testing.B) {
			benchHandleDataRequest(b, keys)
		})
	}
}

// benchHandleDataRequest is the shared body of the DataRequest benchmark and
// the flatness gate's measurement.
func benchHandleDataRequest(b *testing.B, keys int) {
	b.Helper()
	core, head := benchFixture(b, keys)
	in := benchDatagram(port.DataRequest{Key: head[0].Key})
	var out port.Outbox
	b.ReportAllocs()
	b.ReportMetric(float64(keys), "store-keys")
	b.ResetTimer()
	for i := range b.N {
		now := at(time.Duration(i+1) * time.Nanosecond)
		core.HandleDatagram(now, in, &out)
		core.Advance(now, &out)
		out.Drain()
	}
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

// benchHandleDataSync is the shared body of the DataSync benchmark and the
// flatness gate's measurement.
func benchHandleDataSync(b *testing.B, keys int) {
	b.Helper()
	core, head := benchFixture(b, keys)
	in := benchDatagram(port.DataSync{Entries: head})
	var out port.Outbox
	b.ReportAllocs()
	b.ReportMetric(float64(keys), "store-keys")
	b.ResetTimer()
	for i := range b.N {
		now := at(time.Duration(i+1) * time.Nanosecond)
		core.HandleDatagram(now, in, &out)
		core.Advance(now, &out)
		out.Drain()
	}
}

// TestCore_keyCountCostFlat is XPORT-06's executable acceptance: each
// inbound handler is measured at 1 000 and 50 000 stored keys, and the large
// store may cost at most 2x the small one. Warm-up runs precede every
// measurement; a zero measurement fails by name instead of skipping. Both
// ns/op values and the ratio are logged on success and named in the failure.
func TestCore_keyCountCostFlat(t *testing.T) {
	t.Parallel()
	assertFlat := func(t *testing.T, name string, bench func(*testing.B, int)) {
		t.Helper()
		measure := func(keys int) float64 {
			testing.Benchmark(func(b *testing.B) {
				b.Helper()
				bench(b, keys)
			}) // warm-up
			result := testing.Benchmark(func(b *testing.B) {
				b.Helper()
				bench(b, keys)
			})
			if result.N == 0 {
				t.Fatalf("%s at %d keys: benchmark ran zero iterations", name, keys)
			}
			nsPerOp := float64(result.NsPerOp())
			if nsPerOp <= 0 {
				t.Fatalf("%s at %d keys: measured %v ns/op, want a positive measurement", name, keys, result.NsPerOp())
			}
			return nsPerOp
		}
		small := measure(benchSmallKeys)
		large := measure(benchLargeKeys)
		ratio := large / small
		t.Logf("%s: %d keys = %.1f ns/op, %d keys = %.1f ns/op, ratio %.2f",
			name, benchSmallKeys, small, benchLargeKeys, large, ratio)
		if ratio > benchMaxRatio {
			t.Fatalf("%s: %d keys = %.1f ns/op vs %d keys = %.1f ns/op (ratio %.2f > %.1f) — inbound handling scales with unrelated stored keys (XPORT-06)",
				name, benchLargeKeys, large, benchSmallKeys, small, ratio, benchMaxRatio)
		}
	}
	assertFlat(t, "HandleDatagram(DataRequest)", benchHandleDataRequest)
	assertFlat(t, "HandleDatagram(DataSync)", benchHandleDataSync)
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
		core.NotifyLocalWrite(at(0), key, rec, &out)
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
