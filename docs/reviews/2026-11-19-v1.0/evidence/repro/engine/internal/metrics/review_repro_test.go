package metrics_test

import (
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/metrics"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

func rss(t *testing.T) float64 {
	t.Helper()
	src := metrics.Sources{
		Discards:      func() repl.DiscardStats { return repl.DiscardStats{} },
		KnownPeers:    func() uint64 { return 0 },
		Unacked:       func() repl.UnackedStats { return repl.UnackedStats{} },
		Sent:          func() udprepl.SentByType { return udprepl.SentByType{} },
		Retransmitted: func() uint64 { return 0 },
		MergeStats:    func() repl.MergeStats { return repl.MergeStats{} },
		DumpStats:     func() persist.DumpStats { return persist.DumpStats{} },
	}
	for _, f := range metrics.Snapshot(src) {
		if f.Name == "process_resident_memory_bytes" {
			return f.Samples[0].Value
		}
	}
	t.Fatal("family missing")
	return 0
}

var sink [][]byte

// ENG-11: process_resident_memory_bytes is fed from ru_maxrss — the PEAK
// resident set — so it never falls after a dump-time spike.
func TestReview_residentMemoryGaugeIsPeak(t *testing.T) {
	before := rss(t)
	for range 512 { // a 512 MiB transient, like one dump cycle of a ~100 MiB store
		b := make([]byte, 1<<20)
		for i := range b {
			b[i] = byte(i)
		}
		sink = append(sink, b)
	}
	during := rss(t)
	sink = nil
	runtime.GC()
	debug.FreeOSMemory()
	after := rss(t)
	mb := func(v float64) float64 { return v / (1 << 20) }
	t.Logf("gauge before=%.0f MiB, with 512 MiB live=%.0f MiB, after free+scavenge=%.0f MiB", mb(before), mb(during), mb(after))
	if after >= during {
		t.Errorf("the gauge did not fall after the memory was released: it reports the high-water mark, not the resident set")
	}
}
