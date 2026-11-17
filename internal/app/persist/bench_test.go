package persist_test

// The TEST-05 dump benchmark (D-13, plan 06-08): the first benchmark in the
// repo. BenchmarkDump10kKeys drives the REAL dump path — persist.Encode over
// store.Records() into a persistfile.FileStore rooted at b.TempDir, exactly
// the Encode+Save pair runDump performs per dump cycle (engine.go) — over a
// store seeded with exactly 10,000 keys of varied small value sizes from a
// deterministic size table. Manual before/after comparison via `make bench`
// only: no thresholds anywhere, CI never runs it (D-13/D-14) — run-to-run
// comparison is the human's job.

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
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

// BenchmarkDump10kKeys measures one full dump cycle at 10k keys through the
// real path: Encode over the store's full snapshot, then the persistfile
// FileStore's durable Save (temp file, fsync, rotate, rename, dir fsync —
// the PERS-02 atomic-write sequence) into one temp directory per benchmark
// run, per-iteration Save following the FileStore contract. bytes/op is the
// mean encoded size per iteration and dump-bytes the (constant) dump size —
// both derived via ReportMetric with whitespace-free unit strings; ns/op
// stays visible for the Encode+Save span itself. No thresholds: a shared
// machine's numbers are noise, D-13 keeps the gate human.
func BenchmarkDump10kKeys(b *testing.B) {
	store := benchSeedStore(b)
	dumps := persistfile.New(filepath.Join(b.TempDir(), "dump.json"))
	b.ResetTimer()
	var totalBytes, lastSize int64
	for range b.N {
		block, err := persist.Encode(store.Records())
		if err != nil {
			b.Fatalf("Encode 10k-key dump: %v", err)
		}
		err = dumps.Save(block)
		if err != nil {
			b.Fatalf("persistfile Save: %v", err)
		}
		totalBytes += int64(len(block))
		lastSize = int64(len(block))
	}
	b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(lastSize), "dump-bytes")
}
