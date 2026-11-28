package storage_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

func seed(tb testing.TB, keys, valueBytes int) *storage.Store {
	tb.Helper()
	store := storage.NewStore(1, clock.NewClock(time.Now, 0, 0, 0), storage.Config{})
	for i := range keys {
		data := make([]byte, valueBytes)
		for j := range data {
			data[j] = byte(i + j)
		}
		if _, _, err := store.Set(fmt.Sprintf("svc/v1/key-%05d", i), data); err != nil {
			tb.Fatal(err)
		}
	}
	return store
}

func benchRecords(b *testing.B, keys, valueBytes int) {
	store := seed(b, keys, valueBytes)
	runtime.GC()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = store.Records()
	}
}

func BenchmarkReview_Records_10k_1KiB(b *testing.B)  { benchRecords(b, 10_000, 1024) }
func BenchmarkReview_Records_10k_60KiB(b *testing.B) { benchRecords(b, 10_000, 60*1024) }
func BenchmarkReview_Records_50k_1KiB(b *testing.B)  { benchRecords(b, 50_000, 1024) }

// What repl.Core.handleDataRequest does per inbound DataRequest datagram.
func BenchmarkReview_SingleKeyViaRecords_10k_1KiB(b *testing.B) {
	store := seed(b, 10_000, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = store.Records()["svc/v1/key-00042"]
	}
}

func BenchmarkReview_SingleKeyViaGet_10k_1KiB(b *testing.B) {
	store := seed(b, 10_000, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = store.Get("svc/v1/key-00042")
	}
}

// How long a writer is locked out while one Records() snapshot runs.
func TestReview_writerStallDuringRecords(t *testing.T) {
	for _, tc := range []struct{ keys, size int }{{10_000, 1024}, {10_000, 60 * 1024}} {
		store := seed(t, tc.keys, tc.size)
		runtime.GC()
		stop := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				select {
				case <-stop:
					return
				default:
					_ = store.Records()
				}
			}
		}()
		var worst time.Duration
		for range 200 {
			start := time.Now()
			_, _, _ = store.Set("probe", []byte("x"))
			worst = max(worst, time.Since(start))
			time.Sleep(time.Millisecond)
		}
		close(stop)
		<-done
		t.Logf("%d keys x %d B: worst Set latency while Records() snapshots run = %v", tc.keys, tc.size, worst)
	}
}
