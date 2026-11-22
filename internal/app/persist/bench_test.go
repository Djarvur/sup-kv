package persist_test

// The TEST-05 dump benchmark (D-13, plan 06-08, extended by 05-04): the
// first benchmark in the repo. BenchmarkDump10kKeys drives the REAL streamed
// dump path — one store.Records() copy encoded by persist.EncodeTo inside
// persistfile.FileStore.SaveStream (the same pair runDump performs per dump
// cycle) — over a store seeded with exactly 10,000 keys of varied small
// value sizes from a deterministic size table. Manual before/after
// comparison via `make bench` only: no thresholds anywhere, CI never runs it
// (D-13/D-14) — run-to-run comparison is the human's job.

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// benchKeyCount is the D-13 dump size: exactly ten thousand keys.
const benchKeyCount = 10_000

// benchNodeID stamps the seeded store's versions; it carries no wire role.
const benchNodeID = uint16(99)

// benchValueSizes is the deterministic value-size table the seeding cycles
// over — reproducible by construction, never an rng draw: the same table and
// fill pattern produce byte-identical dumps across runs, so bytes/op and the
// dump size are comparable without pinning fixtures.
var benchValueSizes = [...]int{16, 32, 64, 128, 256, 512, 1024}

// benchSeedStore builds the store under measurement: a fixed clock (the
// sim_chaos_test.go clock-fake pattern — one frozen reading) stamps 10,000
// strictly monotonic versions, because Tick's counterMax overflow rule bumps
// l by one millisecond instead of ever wrapping a counter (clock.go), so a
// frozen wall never repeats a version across the seed. The deterministic
// size table plus the i-derived fill pattern keep every value reproducible.
func benchSeedStore(tb testing.TB) *storage.Store {
	tb.Helper()
	now := func() time.Time { return time.UnixMilli(1_700_000_000_000) }
	store := storage.NewStore(benchNodeID, clock.NewClock(now, 0, 0, time.Minute), storage.Config{})
	for i := range benchKeyCount {
		key := fmt.Sprintf("bench/key-%05d", i)
		data := make([]byte, benchValueSizes[i%len(benchValueSizes)])
		for j := range data {
			data[j] = byte(i*31 + j)
		}
		_, _, err := store.Set(key, data)
		if err != nil {
			tb.Fatalf("seed Set(%q): %v", key, err)
		}
	}
	return store
}

// countingWriter counts the bytes written through it: the streamed dump's
// size, observed without materializing the encoded generation.
type countingWriter struct {
	w io.Writer
	n int64
}

// Write forwards p and accumulates its length.
func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// liveHeapBytes returns the current live heap allocation (runtime.MemStats
// HeapAlloc) — the stdlib's closest measure of live state at the sample
// instant.
func liveHeapBytes() uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}

// BenchmarkDump10kKeys measures one full streamed dump cycle at 10k keys
// through the real path: one store.Records() copy, EncodeTo feeding the
// persistfile FileStore's buffered temp writer, then the durable save (temp
// file, fsync, rotate, rename, dir fsync — the PERS-02 atomic-write
// sequence) into one temp directory per benchmark run.
//
// Reported figures (D-06, PERSIST-06):
//   - B/op (-benchmem) is the allocation churn of the whole cycle.
//   - peak-live-B samples runtime.ReadMemStats at the end of the fill, the
//     cycle's peak moment: the one unavoidable Records() copy and the
//     adapter's 64 KiB bufio are both live. That copy plus the writer pause
//     is the accepted residual; a second encoded state would show up here as
//     a roughly doubled figure.
//   - bytes/op and dump-bytes are the streamed generation's size, counted at
//     the writer so the dump is never held as a second buffer.
//
// ns/op stays visible for the cycle itself, and p50-ms/p99-ms report the
// same per-iteration spans' distribution (TEST-10: measured per-op durations,
// no self-pacing — the loop runs as fast as the real path allows). No
// thresholds: a shared machine's numbers are noise, D-13 keeps the gate
// human.
func BenchmarkDump10kKeys(b *testing.B) {
	store := benchSeedStore(b)
	dumps := persistfile.New(filepath.Join(b.TempDir(), "dump.json"))
	durations := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	var totalBytes, lastSize int64
	var peakLive uint64
	for range b.N {
		start := time.Now()
		records := store.Records()
		var sampled, written int64
		err := dumps.SaveStream(func(w io.Writer) error {
			counter := &countingWriter{w: w}
			err := persist.EncodeTo(counter, records, storage.ValueMaxBytes)
			written = counter.n
			sampled = int64(liveHeapBytes())
			return err
		})
		if err != nil {
			b.Fatalf("streamed dump cycle: %v", err)
		}
		runtime.KeepAlive(records)
		durations = append(durations, time.Since(start))
		totalBytes += written
		lastSize = written
		if sampled > int64(peakLive) {
			peakLive = uint64(sampled)
		}
	}
	b.StopTimer()
	testutil.ReportLatencyPercentiles(b, durations)
	b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(lastSize), "dump-bytes")
	b.ReportMetric(float64(peakLive), "peak-live-B")
}
