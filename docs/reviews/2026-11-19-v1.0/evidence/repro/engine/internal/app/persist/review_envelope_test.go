package persist_test

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// ENG-10: one dump cycle at the design envelope — time, transient allocation
// and peak heap relative to the live state.
func TestReview_dumpCycleAtEnvelope(t *testing.T) {
	for _, tc := range []struct{ keys, size int }{{10_000, 1024}, {10_000, 60 * 1024}} {
		store := storage.NewStore(1, clock.NewClock(time.Now, 0, 0, 0), storage.Config{})
		rng := rand.New(rand.NewPCG(1, 2))
		for i := range tc.keys {
			data := make([]byte, tc.size)
			for j := range data {
				data[j] = byte(rng.Uint32())
			}
			if _, _, err := store.Set(fmt.Sprintf("svc/v1/key-%05d", i), data); err != nil {
				t.Fatal(err)
			}
		}
		dumps := persistfile.New(filepath.Join(t.TempDir(), "dump.json"))
		runtime.GC()
		var base runtime.MemStats
		runtime.ReadMemStats(&base)

		var peak atomic.Uint64
		stop := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			var ms runtime.MemStats
			for {
				select {
				case <-stop:
					return
				default:
					runtime.ReadMemStats(&ms)
					if ms.HeapInuse > peak.Load() {
						peak.Store(ms.HeapInuse)
					}
					time.Sleep(time.Millisecond)
				}
			}
		}()
		start := time.Now()
		recs := store.Records()
		tCopy := time.Since(start)
		block, err := persist.Encode(recs)
		if err != nil {
			t.Fatal(err)
		}
		tEncode := time.Since(start) - tCopy
		if err := dumps.Save(block); err != nil {
			t.Fatal(err)
		}
		total := time.Since(start)
		close(stop)
		<-done
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		mb := func(v uint64) float64 { return float64(v) / (1 << 20) }
		t.Logf("%d keys x %d B: live heap %.0f MiB | dump file %.0f MiB | Records() %v, Encode %v, Save %v, total %v | allocated during the cycle %.0f MiB | peak heap %.0f MiB (%.1fx live)",
			tc.keys, tc.size, mb(base.HeapInuse), mb(uint64(len(block))), tCopy.Round(time.Millisecond),
			tEncode.Round(time.Millisecond), (total - tCopy - tEncode).Round(time.Millisecond), total.Round(time.Millisecond),
			mb(after.TotalAlloc-base.TotalAlloc), mb(peak.Load()), float64(peak.Load())/float64(base.HeapInuse))
		runtime.KeepAlive(recs)
	}
}
