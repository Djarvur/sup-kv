package persistfile_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// Seed-test constants: the fixed node ID and drift tolerance of the
// composition, the injected wall-clock millisecond (the shared fixture value
// of the app packages' tests), the dump-ahead offset that makes a dumped
// watermark lead the injected now, the engine's shrunken debounce, and the
// local write's key and value.
const (
	seedNodeID    = uint16(0x0210)
	seedDriftWarn = time.Minute
	seedNowMs     = int64(1730000000000)
	seedAheadMs   = int64(time.Hour / time.Millisecond)
	seedDebounce  = time.Millisecond
	seedKey       = "local"
	seedValue     = "first-local-write"
	seedDumpedKey = "dumped"
	seedDumpedVal = "dumped-value"
)

// seedRow is one TEST-07 composition row: the crafted dump (nil is no dump
// file at all — the empty start), the injected now, and the expectation for
// the first local write. A reopen row skips the write and instead proves two
// Opens of the same directory load identical records.
type seedRow struct {
	name      string
	nowMs     int64
	records   map[string]storage.Record
	wantL     int64
	wantC     uint16
	dominates []string
	reopen    bool
}

// seedRecord returns a codec-legal dumped record for key at the (l, counter)
// version with payload data — the checksum is the storage domain the decoder
// verifies, so the crafted dump loads instead of discarding.
func seedRecord(key string, l int64, counter uint16, data string) storage.Record {
	ver := version.PackVersion(l, counter, seedNodeID)
	payload := []byte(data)
	return storage.Record{
		Version:  ver,
		Data:     payload,
		Checksum: storage.RecordChecksum(key, ver, payload, false),
	}
}

// seedFixedNow returns the injected now func parked at the given Unix
// millisecond, built with time.UnixMilli so no monotonic reading is involved
// (the clock package's D-02 test seam convention).
func seedFixedNow(ms int64) func() time.Time {
	return func() time.Time { return time.UnixMilli(ms) }
}

// seedDump writes records as a codec-encoded main generation at dumpPath —
// the real file the adapter's fallback chain serves to persist.Open.
func seedDump(t *testing.T, dumpPath string, records map[string]storage.Record) {
	t.Helper()
	block, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	writeFile(t, dumpPath, block)
}

// seedDumper launches eng's single dumper goroutine — Run's documented
// lifecycle — and stops it at cleanup, asserting Run's context.Canceled
// exit; the external-package twin of engine_test.go's runEngine.
func seedDumper(t *testing.T, eng *persist.Engine) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		err := <-done
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
	})
}

// seedRecordsEqual reports whether two record maps are deep-equal: same
// keys, versions, deleted flags, checksums and data bytes.
func seedRecordsEqual(got, want map[string]storage.Record) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantRec := range want {
		gotRec, ok := got[key]
		if !ok || gotRec.Version != wantRec.Version || gotRec.Deleted != wantRec.Deleted ||
			gotRec.Checksum != wantRec.Checksum || !bytes.Equal(gotRec.Data, wantRec.Data) {
			return false
		}
	}
	return true
}

// TestOpen_clockSeededFromDumpWatermark proves TEST-07 on the REAL
// composition path (D-07): a FileStore on a temp dir composed with
// persist.Open and the real codec — no fake dump port anywhere — loads a
// crafted dump, seeds the clock with the dump's FULL watermark, and the
// first local write strictly dominates every loaded version even when the
// injected now lies earlier than that watermark. The five rows are the
// probe's named cases: dominates, boundary-equal, empty start, equal-max-l
// and restart idempotence.
func TestOpen_clockSeededFromDumpWatermark(t *testing.T) {
	t.Parallel()

	aheadL := seedNowMs + seedAheadMs
	tests := []seedRow{
		{
			name:  "dump watermark ahead of the injected now is dominated",
			nowMs: seedNowMs,
			records: map[string]storage.Record{
				seedDumpedKey: seedRecord(seedDumpedKey, aheadL, 0, seedDumpedVal),
			},
			// Seeded (aheadL, 0); the load-time Merge's Update ticks once to
			// (aheadL, 1); the first Tick — the injected now is behind —
			// increments to (aheadL, 2).
			wantL:     aheadL,
			wantC:     2,
			dominates: []string{seedDumpedKey},
		},
		{
			name:  "injected now exactly at the watermark l still yields a greater write",
			nowMs: seedNowMs,
			records: map[string]storage.Record{
				seedDumpedKey: seedRecord(seedDumpedKey, seedNowMs, 7, seedDumpedVal),
			},
			// Seeded (now, 7); the load-time Update ticks to (now, 8); the
			// first Tick, its wall reading exactly at l, increments to
			// (now, 9) — strictly greater, never equal.
			wantL:     seedNowMs,
			wantC:     9,
			dominates: []string{seedDumpedKey},
		},
		{
			name:  "empty start seeds from the injected now alone",
			nowMs: seedNowMs,
			// No dump file: the store is empty and the clock starts at
			// (0, 0), so the first Tick takes the wall clock whole.
			wantL: seedNowMs,
			wantC: 0,
		},
		{
			name:  "two records sharing the max l take that counter pair",
			nowMs: seedNowMs,
			records: map[string]storage.Record{
				"dumped-low":  seedRecord("dumped-low", aheadL, 1, "low"),
				"dumped-high": seedRecord("dumped-high", aheadL, 5, "high"),
			},
			// Watermark takes (aheadL, 5); the two at-watermark Updates tick
			// to 7 under either map iteration order; the first Tick to 8.
			wantL:     aheadL,
			wantC:     8,
			dominates: []string{"dumped-low", "dumped-high"},
		},
		{
			name:  "reopening the same dump loads identical records",
			nowMs: seedNowMs,
			records: map[string]storage.Record{
				"alpha": seedRecord("alpha", seedNowMs-seedAheadMs, 0, "alpha-value"),
				"beta":  seedRecord("beta", seedNowMs, 3, "beta-value"),
			},
			reopen: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runSeedRow(t, tt)
		})
	}
}

// runSeedRow executes one composition row against a fresh temp-dir
// FileStore. Every assertion goes through the returned engine/store — the
// records the real codec decoded into the real store.
func runSeedRow(t *testing.T, row seedRow) {
	t.Helper()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	if row.records != nil {
		seedDump(t, dumpPath, row.records)
	}
	eng, discarded := persist.Open(
		seedNodeID, seedFixedNow(row.nowMs), seedDriftWarn,
		persistfile.New(dumpPath), storage.Config{},
		persist.Config{Debounce: seedDebounce},
	)
	if discarded != 0 {
		t.Fatalf("Open() discarded = %d, want 0", discarded)
	}
	loaded := eng.Records()
	if !seedRecordsEqual(loaded, row.records) {
		t.Fatalf("Open() loaded %d records, want the crafted dump's %d", len(loaded), len(row.records))
	}
	if row.reopen {
		again, discardedAgain := persist.Open(
			seedNodeID, seedFixedNow(row.nowMs), seedDriftWarn,
			persistfile.New(dumpPath), storage.Config{},
			persist.Config{Debounce: seedDebounce},
		)
		if discardedAgain != 0 {
			t.Fatalf("second Open() discarded = %d, want 0", discardedAgain)
		}
		if !seedRecordsEqual(again.Records(), loaded) {
			t.Error("reopening the same dump directory loaded different records")
		}
		return
	}

	seedDumper(t, eng)
	ver, _, err := eng.Set(seedKey, []byte(seedValue))
	if err != nil {
		t.Fatalf("Set(%q) error = %v, want nil", seedKey, err)
	}
	l, counter, node := ver.Unpack()
	if l != row.wantL {
		t.Errorf("first local write l = %d, want %d", l, row.wantL)
	}
	if counter != row.wantC {
		t.Errorf("first local write counter = %d, want %d", counter, row.wantC)
	}
	if node != seedNodeID {
		t.Errorf("first local write node = %#x, want %#x", node, seedNodeID)
	}
	for _, key := range row.dominates {
		dumped, ok := loaded[key]
		if !ok {
			t.Fatalf("crafted dump lost key %q", key)
		}
		if ver.Compare(dumped.Version) <= 0 {
			t.Errorf("first local write version %x does not dominate dumped key %q version %x",
				ver[:], key, dumped.Version[:])
		}
	}
	rec, err := eng.Get(seedKey)
	if err != nil {
		t.Fatalf("Get(%q) after the first local write error = %v, want nil", seedKey, err)
	}
	if rec.Version != ver {
		t.Errorf("stored version %x, want the returned %x", rec.Version[:], ver[:])
	}
}
