package persistfile_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Adapter-test constants: the marker a readable generation carries for the
// oracle, the dump file mode (os.CreateTemp's 0600 default, values
// privacy), and the read-only / read-write directory modes of the failure
// hygiene test.
const (
	readableMarker = "READABLE"
	dumpFileMode   = 0o600
	roDirMode      = 0o500
	rwDirMode      = 0o700
)

// errUnreadable is the marker oracle's rejection.
var errUnreadable = errors.New("unreadable generation")

// permissiveOracle accepts every generation.
func permissiveOracle([]byte) error { return nil }

// markerOracle accepts exactly the generations carrying the readable
// marker — a legitimate stand-in for codec.Parse: the adapter is
// oracle-agnostic by design (D-02), so any total function probes the chain.
func markerOracle(dump []byte) error {
	if bytes.Contains(dump, []byte(readableMarker)) {
		return nil
	}
	return errUnreadable
}

// writeFile writes data to path with the dump file mode; a test-setup
// helper with no assertions of its own.
func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	err := os.WriteFile(path, data, dumpFileMode)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestFileStore_atomicRotation proves the rotation half of PERS-02 and the
// save/load smoke: two saves leave the second generation at the main path,
// the first at .prev, the main file keeps os.CreateTemp's 0600 mode, and
// Load with a permissive oracle serves the valid main generation.
func TestFileStore_atomicRotation(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	store := persistfile.New(dumpPath)
	gen1 := []byte("GENERATION-ONE")
	gen2 := []byte("GENERATION-TWO")
	err := store.Save(gen1)
	if err != nil {
		t.Fatalf("first Save() error = %v, want nil", err)
	}
	err = store.Save(gen2)
	if err != nil {
		t.Fatalf("second Save() error = %v, want nil", err)
	}
	main, err := os.ReadFile(dumpPath)
	if err != nil || !bytes.Equal(main, gen2) {
		t.Errorf("main generation = (%q, %v), want generation two", main, err)
	}
	prev, err := os.ReadFile(dumpPath + ".prev")
	if err != nil || !bytes.Equal(prev, gen1) {
		t.Errorf(".prev generation = (%q, %v), want generation one", prev, err)
	}
	info, err := os.Stat(dumpPath)
	if err != nil {
		t.Fatalf("Stat(main) error = %v", err)
	}
	if info.Mode().Perm() != dumpFileMode {
		t.Errorf("main mode = %o, want %o (the CreateTemp default, values privacy)",
			info.Mode().Perm(), dumpFileMode)
	}
	res := store.Load(time.Now(), permissiveOracle)
	if res.Outcome != port.LoadMain || res.Generation != port.GenMain || !bytes.Equal(res.Dump, gen2) {
		t.Errorf("Load() = (outcome %v, generation %v, %q), want the valid main generation",
			res.Outcome, res.Generation, res.Dump)
	}
	if len(res.Failures) != 0 {
		t.Errorf("Load() failures = %v, want none for an accepted main generation", res.Failures)
	}
}

// TestFileStore_fallbackChain proves the chain half of PERS-02 over the
// crash-residue table: a truncated, garbage, missing or oracle-rejecting
// main generation advances the chain to .prev's bytes, the outcome names
// exactly which generation served, and both generations missing is the
// prompt start. Rows with a present-but-unparseable candidate also pin the
// failure report and the unusable classification (D-02).
func TestFileStore_fallbackChain(t *testing.T) {
	t.Parallel()
	prevGen := []byte("PREV-" + readableMarker)
	mainGen := []byte(readableMarker + "-MAIN")
	tests := []struct {
		name        string
		main        []byte // nil removes the main generation entirely.
		prev        []byte // nil removes the .prev generation entirely.
		want        []byte // nil expects no accepted generation.
		wantOutcome port.LoadOutcome
		wantGen     port.Generation
		wantFails   int
	}{
		{
			name:        "main truncated mid-line serves prev",
			main:        prevGen[:len(prevGen)-2],
			prev:        prevGen,
			want:        prevGen,
			wantOutcome: port.LoadBackup,
			wantGen:     port.GenPrev,
			wantFails:   1,
		},
		{
			name:        "main containing not-JSON garbage serves prev",
			main:        []byte("not json at all"),
			prev:        prevGen,
			want:        prevGen,
			wantOutcome: port.LoadBackup,
			wantGen:     port.GenPrev,
			wantFails:   1,
		},
		{
			name:        "main removed serves prev",
			prev:        prevGen,
			want:        prevGen,
			wantOutcome: port.LoadBackup,
			wantGen:     port.GenPrev,
			wantFails:   0,
		},
		{
			name:        "both generations removed is the prompt start",
			wantOutcome: port.LoadPrompt,
			wantGen:     port.GenMain,
			wantFails:   0,
		},
		{
			name:        "an oracle-rejecting main serves prev",
			main:        []byte("{\"format_version\":1}\n"),
			prev:        prevGen,
			want:        prevGen,
			wantOutcome: port.LoadBackup,
			wantGen:     port.GenPrev,
			wantFails:   1,
		},
		{
			name:        "an accepted main is the main outcome",
			main:        mainGen,
			want:        mainGen,
			wantOutcome: port.LoadMain,
			wantGen:     port.GenMain,
			wantFails:   0,
		},
		{
			name:        "both generations unparseable is unusable",
			main:        []byte("not json at all"),
			prev:        []byte("also not json"),
			wantOutcome: port.LoadUnusable,
			wantGen:     port.GenMain,
			wantFails:   2,
		},
		{
			name:        "one unparseable main with no prev is unusable",
			main:        []byte("not json at all"),
			wantOutcome: port.LoadUnusable,
			wantGen:     port.GenMain,
			wantFails:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			dumpPath := filepath.Join(dir, "dump.json")
			if tt.main != nil {
				writeFile(t, dumpPath, tt.main)
			}
			if tt.prev != nil {
				writeFile(t, dumpPath+".prev", tt.prev)
			}
			res := persistfile.New(dumpPath).Load(time.Now(), markerOracle)
			if res.Outcome != tt.wantOutcome {
				t.Errorf("Load() outcome = %v, want %v", res.Outcome, tt.wantOutcome)
			}
			if res.Generation != tt.wantGen {
				t.Errorf("Load() generation = %v, want %v", res.Generation, tt.wantGen)
			}
			if tt.want == nil {
				if res.Dump != nil {
					t.Errorf("Load() dump = %q, want no accepted generation", res.Dump)
				}
			} else if !bytes.Equal(res.Dump, tt.want) {
				t.Errorf("Load() dump = %q, want %q", res.Dump, tt.want)
			}
			if len(res.Failures) != tt.wantFails {
				t.Errorf("Load() failures = %d, want %d: %v", len(res.Failures), tt.wantFails, res.Failures)
			}
		})
	}
}

// TestFileStore_saveFailureHygiene proves the temp-file hygiene of Pitfall
// 5: a Save that cannot create its temp file fails, and the directory keeps
// exactly its pre-existing dump — no dump-* litter. Skipped as root, where
// directory modes do not bind.
func TestFileStore_saveFailureHygiene(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory modes - the read-only residue cannot be produced")
	}
	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.json")
	store := persistfile.New(dumpPath)
	err := store.Save([]byte("first"))
	if err != nil {
		t.Fatalf("seed Save() error = %v, want nil", err)
	}
	err = os.Chmod(dir, roDirMode)
	if err != nil {
		t.Fatalf("Chmod read-only error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, rwDirMode) })
	err = store.Save([]byte("second"))
	if err == nil {
		t.Fatal("Save() into a read-only directory = nil, want an error")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v, want nil", err)
	}
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("directory after the failed save holds %v, want only the pre-existing dump", names)
	}
}

// TestFileStore_snapshotSmoke proves the snapshot surface: a snapshot is a
// byte-identical copy under the .snapshots directory beside the dump, a
// missing main dump is ErrNoDump, and a missing snapshot surfaces the
// wrapped fs error.
func TestFileStore_snapshotSmoke(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.json")
	store := persistfile.New(dumpPath)
	dump := []byte("the-dump-bytes")
	err := store.Save(dump)
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	err = store.CreateSnapshot("snap1")
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v, want nil", err)
	}
	copied, err := os.ReadFile(filepath.Join(dir, "dump.json.snapshots", "snap1"))
	if err != nil || !bytes.Equal(copied, dump) {
		t.Errorf("snapshot = (%q, %v), want a byte-identical copy of the dump", copied, err)
	}
	empty := persistfile.New(filepath.Join(t.TempDir(), "dump.json"))
	err = empty.CreateSnapshot("nope")
	if !errors.Is(err, persistfile.ErrNoDump) {
		t.Errorf("CreateSnapshot without a main dump = %v, want ErrNoDump", err)
	}
	_, err = store.ReadSnapshot("missing")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadSnapshot(missing) = %v, want an error wrapping fs.ErrNotExist", err)
	}
}

// Concurrency-stress constants: the savers keep rotating generations while
// the snapshotters hammer the copy against those rotations — sized small
// on purpose, every save and snapshot fsyncs, so the deterministic
// white-box test carries the regression proof and this one only exercises
// the live overlap.
const (
	concurrentSavers      = 2
	concurrentSnappers    = 2
	savesPerSaver         = 20
	hotSnapshotName       = "hot"
	settledSnapshotName   = "settled"
	firstConfirmedGen     = "confirmed-first-generation"
	concurrentSavePattern = "generation-%03d"
)

// TestFileStore_snapshotConcurrentWithSave is WR-01's concurrent half:
// with a confirmed generation on disk from the seed save on, savers keep
// rotating generations — every Save's placeFile briefly leaves the main
// path absent between its two renames — while snapshotters hammer
// CreateSnapshot, and no snapshot may ever surface ErrNoDump: the store
// mutex serializes the copy against the rotation window. The concurrent
// Saves themselves serialize on the same mutex, so the run also pins
// Save's own added concurrency safety.
func TestFileStore_snapshotConcurrentWithSave(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	store := persistfile.New(dumpPath)
	err := store.Save([]byte(firstConfirmedGen))
	if err != nil {
		t.Fatalf("seed Save() error = %v, want nil", err)
	}
	var savers, snappers sync.WaitGroup
	stop := make(chan struct{})
	for range concurrentSnappers {
		snappers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				err := store.CreateSnapshot(hotSnapshotName)
				if err != nil {
					if errors.Is(err, persistfile.ErrNoDump) {
						t.Error("CreateSnapshot() = ErrNoDump despite the confirmed dump on disk (rotate-window race)")
					} else {
						t.Errorf("concurrent CreateSnapshot() error = %v, want nil", err)
					}
					return
				}
			}
		})
	}
	for range concurrentSavers {
		savers.Go(func() {
			for i := range savesPerSaver {
				err := store.Save([]byte(fmt.Sprintf(concurrentSavePattern, i)))
				if err != nil {
					t.Errorf("concurrent Save() error = %v, want nil", err)
					return
				}
			}
		})
	}
	savers.Wait()
	close(stop)
	snappers.Wait()
	err = store.CreateSnapshot(settledSnapshotName)
	if err != nil {
		t.Errorf("settled CreateSnapshot() error = %v, want nil", err)
	}
}
