package persist_test

// The load-report, dirty-start, nudge and skip-evidence suite (PERSIST-02,
// PERSIST-04, PERSIST-10; D-02, D-04, D-09). The report tests drive a fake
// DumpStore with fixed LoadResults so each outcome's mapping is asserted
// exactly; the lifecycle tests use the synctest bubble and the fake port for
// virtual-time determinism, and the evidence-window test uses the real
// persistfile adapter because it is exactly the on-disk rotation that must be
// observed.

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
)

// loadResultPort is the fake port.DumpStore of the load-report tests: Load
// returns the fixed result and runs the caller's parse oracle over the
// result's Dump exactly as the file adapter does, while the embedded
// fakeDumpPort supplies the recording save and snapshot halves.
type loadResultPort struct {
	*fakeDumpPort

	result port.LoadResult
}

// Load reports the configured result, running the oracle over an accepted
// generation's bytes as the adapter does.
func (p *loadResultPort) Load(_ time.Time, parse func([]byte) error) port.LoadResult {
	if len(p.result.Dump) > 0 {
		_ = parse(p.result.Dump)
	}
	return p.result
}

// newLoadResultPort builds a fresh fake port reporting result.
func newLoadResultPort(result port.LoadResult) *loadResultPort {
	return &loadResultPort{fakeDumpPort: &fakeDumpPort{}, result: result}
}

// loadFixtureTime is the fixture records' HLC timestamp.
const loadFixtureTime = int64(1730000000000)

// loadFixture builds one live record whose checksum covers its line, so an
// encoded dump decodes back without a checksum discard.
func loadFixture(key, value string, counter uint16) storage.Record {
	ver := version.PackVersion(loadFixtureTime, counter, tracerNodeID)
	data := []byte(value)
	return storage.Record{Version: ver, Data: data, Checksum: storage.RecordChecksum(key, ver, data, false)}
}

// encodeLoadDump encodes records at limit into a valid generation.
func encodeLoadDump(t *testing.T, records map[string]storage.Record) []byte {
	t.Helper()
	dump, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("Encode fixture: %v", err)
	}
	return dump
}

// TestEngine_loadStatsMapsResult proves Open maps the port's LoadResult into
// the engine's LoadStats without changing its own signature: every outcome,
// the accepted generation, the per-candidate failures, the accepted
// generation's decode census and the sweep fields are carried through, and an
// engine built by New reports the zero (unloaded) value.
func TestEngine_loadStatsMapsResult(t *testing.T) {
	t.Parallel()
	mainDump := encodeLoadDump(t, map[string]storage.Record{
		"main": loadFixture("main", "main-value", 1),
	})
	mixedDump := encodeLoadDump(t, map[string]storage.Record{
		"slim": loadFixture("slim", "slim", 2),
		"fat":  loadFixture("fat", strings.Repeat("f", 64), 3),
	})
	sweepErr := fs.ErrPermission

	cases := []struct {
		name          string
		result        port.LoadResult
		storeCfg      storage.Config
		want          persist.LoadStats
		wantDiscarded int
	}{
		{
			name:   "prompt",
			result: port.LoadResult{},
			want:   persist.LoadStats{Loaded: true},
		},
		{
			name:   "main",
			result: port.LoadResult{Dump: mainDump, Outcome: port.LoadMain, Generation: port.GenMain},
			want: persist.LoadStats{
				Loaded: true, Outcome: port.LoadMain, Generation: port.GenMain,
			},
		},
		{
			name: "backup",
			result: port.LoadResult{
				Dump:       mainDump,
				Outcome:    port.LoadBackup,
				Generation: port.GenPrev,
				Failures: []port.GenerationFailure{{
					Candidate: port.GenMain, Path: "/data/dump.json", Cause: fs.ErrNotExist,
				}},
			},
			want: persist.LoadStats{
				Loaded: true, Outcome: port.LoadBackup, Generation: port.GenPrev,
				Failures: []port.GenerationFailure{{
					Candidate: port.GenMain, Path: "/data/dump.json", Cause: fs.ErrNotExist,
				}},
			},
		},
		{
			name: "unusable",
			result: port.LoadResult{
				Outcome: port.LoadUnusable,
				Failures: []port.GenerationFailure{
					{Candidate: port.GenMain, Path: "/data/dump.json", Cause: fs.ErrInvalid, PreserveErr: fs.ErrPermission},
					{Candidate: port.GenPrev, Path: "/data/dump.json.prev", Cause: fs.ErrInvalid},
				},
				Swept:    2,
				SweepErr: sweepErr,
			},
			want: persist.LoadStats{
				Loaded: true, Outcome: port.LoadUnusable,
				Failures: []port.GenerationFailure{
					{Candidate: port.GenMain, Path: "/data/dump.json", Cause: fs.ErrInvalid, PreserveErr: fs.ErrPermission},
					{Candidate: port.GenPrev, Path: "/data/dump.json.prev", Cause: fs.ErrInvalid},
				},
				Swept: 2, SweepErr: sweepErr,
			},
		},
		{
			name: "main with a value-limit discard",
			result: port.LoadResult{
				Dump: mixedDump, Outcome: port.LoadMain, Generation: port.GenMain,
			},
			storeCfg:      storage.Config{ValueMaxBytes: 16},
			wantDiscarded: 1,
			want: persist.LoadStats{
				Loaded: true, Outcome: port.LoadMain, Generation: port.GenMain,
				Decoded: persist.DecodeStats{ValueLimit: 1},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dumps := newLoadResultPort(tc.result)
			eng, discarded := persist.Open(
				tracerNodeID, time.Now, tracerDriftWarn, dumps, tc.storeCfg, persist.Config{},
			)
			if discarded != tc.wantDiscarded {
				t.Errorf("Open() discarded = %d, want %d", discarded, tc.wantDiscarded)
			}
			got := eng.LoadStats()
			if got.Loaded != tc.want.Loaded || got.Outcome != tc.want.Outcome ||
				got.Generation != tc.want.Generation || got.Decoded != tc.want.Decoded ||
				got.Swept != tc.want.Swept {
				t.Errorf("LoadStats() = %+v, want %+v", got, tc.want)
			}
			if !errors.Is(got.SweepErr, tc.want.SweepErr) {
				t.Errorf("LoadStats().SweepErr = %v, want %v", got.SweepErr, tc.want.SweepErr)
			}
			if len(got.Failures) != len(tc.want.Failures) {
				t.Fatalf("LoadStats().Failures = %+v, want %+v", got.Failures, tc.want.Failures)
			}
			for i := range got.Failures {
				gotFail, wantFail := got.Failures[i], tc.want.Failures[i]
				if gotFail.Candidate != wantFail.Candidate || gotFail.Path != wantFail.Path ||
					gotFail.Preserved != wantFail.Preserved ||
					!errors.Is(gotFail.Cause, wantFail.Cause) ||
					!errors.Is(gotFail.PreserveErr, wantFail.PreserveErr) {
					t.Errorf("LoadStats().Failures[%d] = %+v, want %+v", i, gotFail, wantFail)
				}
			}
		})
	}

	t.Run("New reports the zero value", func(t *testing.T) {
		t.Parallel()
		fresh, _ := newFakeEngine()
		if got := fresh.LoadStats(); got.Loaded {
			t.Errorf("New-built engine LoadStats() = %+v, want the zero value (Loaded false)", got)
		}
	})
}

// TestEngine_startDirty proves the D-04 dirty rule and its snapshot
// consequence: a Prompt, Backup or Unusable load starts dirty, so a Snapshot
// right after the start flushes a covering generation and succeeds with no
// ErrNoDump — the Prompt snapshot decodes to the empty state, the Backup one
// to the loaded records — while a Main load (even one that discarded a
// record) starts clean, so Snapshot copies the accepted on-disk generation
// without a rewrite. The save counts are the observable difference.
func TestEngine_startDirty(t *testing.T) {
	t.Parallel()
	loadedDump := encodeLoadDump(t, map[string]storage.Record{
		"loaded": loadFixture("loaded", "loaded-value", 1),
	})
	discardDump := encodeLoadDump(t, map[string]storage.Record{
		"slim": loadFixture("slim", "slim", 2),
		"fat":  loadFixture("fat", strings.Repeat("f", 64), 3),
	})

	cases := []struct {
		name      string
		result    port.LoadResult
		storeCfg  storage.Config
		wantDirty bool
		wantKeys  []string
	}{
		{
			name:      "prompt",
			result:    port.LoadResult{},
			wantDirty: true,
		},
		{
			name: "backup",
			result: port.LoadResult{
				Dump:       loadedDump,
				Outcome:    port.LoadBackup,
				Generation: port.GenPrev,
				Failures: []port.GenerationFailure{{
					Candidate: port.GenMain, Path: "dump.json", Cause: fs.ErrNotExist,
				}},
			},
			wantDirty: true,
			wantKeys:  []string{"loaded"},
		},
		{
			name: "unusable",
			result: port.LoadResult{
				Outcome: port.LoadUnusable,
				Failures: []port.GenerationFailure{{
					Candidate: port.GenMain, Path: "dump.json", Cause: fs.ErrInvalid,
				}},
			},
			wantDirty: true,
		},
		{
			name: "main",
			result: port.LoadResult{
				Dump: loadedDump, Outcome: port.LoadMain, Generation: port.GenMain,
			},
			wantKeys: []string{"loaded"},
		},
		{
			name: "main with a discarded record",
			result: port.LoadResult{
				Dump: discardDump, Outcome: port.LoadMain, Generation: port.GenMain,
			},
			storeCfg: storage.Config{ValueMaxBytes: 16},
			wantKeys: []string{"slim"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				dumps := newLoadResultPort(tc.result)
				eng, _ := persist.Open(
					tracerNodeID, time.Now, tracerDriftWarn, dumps, tc.storeCfg, persist.Config{},
				)
				if !tc.wantDirty {
					dumps.seedSave(tc.result.Dump)
				}
				baseline := dumps.saveCount()
				stop := startDumper(t, eng)
				defer stop()

				err := eng.Snapshot("start")
				if err != nil {
					t.Fatalf("Snapshot right after a %s start: %v", tc.name, err)
				}
				wantSaves := baseline
				if tc.wantDirty {
					wantSaves++
				}
				if got := dumps.saveCount(); got != wantSaves {
					t.Fatalf("%s start ran %d dump(s), want %d (dirty=%t)",
						tc.name, got-baseline, wantSaves-baseline, tc.wantDirty)
				}
				records, _, decodeErr := persist.Decode(dumps.snapshot("start"), tc.storeCfg.ValueLimit())
				if decodeErr != nil {
					t.Fatalf("decode the created snapshot: %v", decodeErr)
				}
				if len(records) != len(tc.wantKeys) {
					t.Fatalf("the created snapshot holds %d record(s), want %d: %v",
						len(records), len(tc.wantKeys), records)
				}
				for _, key := range tc.wantKeys {
					if _, ok := records[key]; !ok {
						t.Fatalf("the created snapshot lacks %q: %v", key, records)
					}
				}
			})
		})
	}
}

// TestEngine_skipEvidenceWindow proves the PERSIST-10 evidence window: a
// record skipped because the configured limit shrank stays on disk through
// exactly the normal rotation discipline — zero writes leave the main file
// byte-identical, the first write rotates it to .prev, and the second dump
// displaces it. The shrunk load is a Main outcome with one ValueLimit
// discard, so it starts clean and nothing rewrites the file on its own.
func TestEngine_skipEvidenceWindow(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	fat := bytes.Repeat([]byte("f"), 64)

	seed, _ := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn, persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	stopSeed := runEngine(t, seed)
	_, _, err := seed.Set("keep", []byte("keep"))
	if err != nil {
		t.Fatalf("seed Set(keep) error = %v, want nil", err)
	}
	_, _, err = seed.Set("fat", fat)
	if err != nil {
		t.Fatalf("seed Set(fat) error = %v, want nil", err)
	}
	stopSeed()

	mainBefore, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read the seeded dump: %v", err)
	}
	if !strings.Contains(string(mainBefore), `"key":"fat"`) {
		t.Fatalf("the seeded dump lacks the fat record:\n%s", mainBefore)
	}

	shrunk, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn, persistfile.New(dumpPath),
		storage.Config{ValueMaxBytes: 16}, persist.Config{},
	)
	if discarded != 1 {
		t.Fatalf("shrunk Open() discarded = %d, want 1 (the value-limit class)", discarded)
	}
	stats := shrunk.LoadStats()
	if stats.Outcome != port.LoadMain || stats.Decoded.ValueLimit != 1 {
		t.Fatalf("shrunk LoadStats() = %+v, want a Main outcome with one ValueLimit discard", stats)
	}
	_, err = shrunk.Get("fat")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("shrunk Get(fat) error = %v, want ErrNotFound", err)
	}
	mainAfterOpen, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read the main dump after the shrunk load: %v", err)
	}
	if !bytes.Equal(mainBefore, mainAfterOpen) {
		t.Fatalf("a Main load with a skipped record rewrote the dump — the evidence window shrank (PERSIST-10):\nbefore:\n%s\nafter:\n%s",
			mainBefore, mainAfterOpen)
	}

	stopShrunk := runEngine(t, shrunk)
	_, _, err = shrunk.Set("first", []byte("first"))
	if err != nil {
		t.Fatalf("Set(first) error = %v, want nil", err)
	}
	prev, err := os.ReadFile(dumpPath + ".prev")
	if err != nil {
		t.Fatalf("read the rotated .prev: %v", err)
	}
	if !strings.Contains(string(prev), `"key":"fat"`) {
		t.Fatalf("after one write the skipped record is not in .prev:\n%s", prev)
	}

	_, _, err = shrunk.Set("second", []byte("second"))
	if err != nil {
		t.Fatalf("Set(second) error = %v, want nil", err)
	}
	stopShrunk()
	for _, path := range []string{dumpPath, dumpPath + ".prev"} {
		blob, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s after two dumps: %v", path, readErr)
		}
		if strings.Contains(string(blob), `"key":"fat"`) {
			t.Fatalf("the skipped record is still in %s after two dumps, want it displaced by normal rotation:\n%s",
				path, blob)
		}
	}
}
