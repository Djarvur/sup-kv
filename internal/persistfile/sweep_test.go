package persistfile_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Sweep-test constants: the marker-bearing temp names and the ages that put
// one on each side of the one-hour bound.
const (
	sweepOldTempName   = ".tmp~old-orphan"
	sweepYoungTempName = ".tmp~young-orphan"
	sweepOldAge        = 2 * time.Hour
	sweepYoungAge      = 30 * time.Minute
)

// sweepNow is the injected load clock of the sweep tests: the seeded old
// temps predate the age bound and the young ones do not.
var sweepNow = time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)

// seedSweepTemp writes one temp-shaped file and backdates it relative to
// sweepNow.
func seedSweepTemp(t *testing.T, path string, age time.Duration) {
	t.Helper()
	writeFile(t, path, []byte("temp"))
	modTime := sweepNow.Add(-age)
	err := os.Chtimes(path, modTime, modTime)
	if err != nil {
		t.Fatalf("Chtimes(%s): %v", path, err)
	}
}

// backdate moves path's mtime behind the sweep's age bound, so only an
// explicit data-name exclusion can keep it alive.
func backdate(t *testing.T, path string) {
	t.Helper()
	modTime := sweepNow.Add(-sweepOldAge)
	err := os.Chtimes(path, modTime, modTime)
	if err != nil {
		t.Fatalf("Chtimes(%s): %v", path, err)
	}
}

// sweepDirNames returns dir's sorted base names.
func sweepDirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names
}

// TestSweep_removesOnlyOldTemps proves the age bound and both sweep
// directories: an old marker-named temp is removed from the dump directory
// and from the snapshots directory, a young one survives in each, and the
// result reports the count with no error.
func TestSweep_removesOnlyOldTemps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.json")
	snapshots := dumpPath + ".snapshots"
	err := os.MkdirAll(snapshots, 0o700)
	if err != nil {
		t.Fatalf("MkdirAll(%s): %v", snapshots, err)
	}
	seedSweepTemp(t, filepath.Join(dir, sweepOldTempName), sweepOldAge)
	seedSweepTemp(t, filepath.Join(dir, sweepYoungTempName), sweepYoungAge)
	seedSweepTemp(t, filepath.Join(snapshots, sweepOldTempName), sweepOldAge)
	seedSweepTemp(t, filepath.Join(snapshots, sweepYoungTempName), sweepYoungAge)

	res := persistfile.New(dumpPath).Load(sweepNow, permissiveOracle)
	if res.Outcome != port.LoadPrompt {
		t.Errorf("Load() outcome = %v, want the prompt start (no generations seeded)", res.Outcome)
	}
	if res.Swept != 2 {
		t.Errorf("Load() swept = %d, want the two old temps", res.Swept)
	}
	if res.SweepErr != nil {
		t.Errorf("Load() sweep error = %v, want nil", res.SweepErr)
	}
	for _, path := range []string{
		filepath.Join(dir, sweepOldTempName),
		filepath.Join(snapshots, sweepOldTempName),
	} {
		_, statErr := os.Lstat(path)
		if !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("Lstat(%s) = %v, want the old temp removed", path, statErr)
		}
	}
	wantDump := []string{sweepYoungTempName, filepath.Base(snapshots)}
	slices.Sort(wantDump)
	if got := sweepDirNames(t, dir); !slices.Equal(got, wantDump) {
		t.Errorf("dump directory after the sweep holds %v, want %v", got, wantDump)
	}
	wantSnapshots := []string{sweepYoungTempName}
	if got := sweepDirNames(t, snapshots); !slices.Equal(got, wantSnapshots) {
		t.Errorf("snapshots directory after the sweep holds %v, want %v", got, wantSnapshots)
	}
}

// TestSweep_protectsDataNames proves the sweep selects by the temp marker,
// never by a data-file prefix, and excludes the configured main's derived
// names for any main name: a dump-prefixed main with its .prev, a
// .corrupt-<UTC> file, a legal dump-backup snapshot and a legal .tmp- (dash)
// snapshot all survive, and a marker-bearing main with its .prev and
// .corrupt-* survives too — while the one old temp is removed in each
// scenario.
func TestSweep_protectsDataNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		main  string
		snaps []string // snapshot names that must survive
	}{
		{
			name:  "dump-prefixed main and snapshot",
			main:  "dump-prod.json",
			snaps: []string{"dump-backup", "dump-prod.json.tmp-legacy"},
		},
		{
			name:  "marker-bearing main",
			main:  "state.tmp~backup.json",
			snaps: []string{"dump-backup"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			dumpPath := filepath.Join(dir, tt.main)
			writeFile(t, dumpPath, []byte(readableMarker))
			writeFile(t, dumpPath+".prev", []byte("prev-"+readableMarker))
			corrupt := dumpPath + ".corrupt-20260101T000000Z"
			writeFile(t, corrupt, []byte("corrupt-generation"))
			backdate(t, dumpPath)
			backdate(t, corrupt)
			backdate(t, dumpPath+".prev")
			snapshots := dumpPath + ".snapshots"
			err := os.MkdirAll(snapshots, 0o700)
			if err != nil {
				t.Fatalf("MkdirAll(%s): %v", snapshots, err)
			}
			for _, name := range tt.snaps {
				writeFile(t, filepath.Join(snapshots, name), []byte("snapshot"))
			}
			oldTemp := filepath.Join(dir, sweepOldTempName)
			seedSweepTemp(t, oldTemp, sweepOldAge)

			res := persistfile.New(dumpPath).Load(sweepNow, permissiveOracle)
			if res.Outcome != port.LoadMain {
				t.Errorf("Load() outcome = %v, want the seeded main generation", res.Outcome)
			}
			if res.Swept != 1 {
				t.Errorf("Load() swept = %d, want exactly the old temp", res.Swept)
			}
			if res.SweepErr != nil {
				t.Errorf("Load() sweep error = %v, want nil", res.SweepErr)
			}
			_, statErr := os.Lstat(oldTemp)
			if !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("Lstat(%s) = %v, want the old temp removed", oldTemp, statErr)
			}
			survivors := []string{dumpPath, dumpPath + ".prev", corrupt}
			for _, path := range survivors {
				if _, err := os.Lstat(path); err != nil {
					t.Errorf("Lstat(%s) = %v, want the data name kept", path, err)
				}
			}
			for _, name := range tt.snaps {
				path := filepath.Join(snapshots, name)
				if _, err := os.Lstat(path); err != nil {
					t.Errorf("Lstat(%s) = %v, want the legal snapshot kept", path, err)
				}
			}
		})
	}
}

// TestSweep_reportsUnreadableDir proves the sweep records the first error
// and the load still proceeds: an unreadable snapshots directory reports
// SweepErr while the chain runs to the prompt outcome. Skipped as root,
// where directory modes do not bind.
func TestSweep_reportsUnreadableDir(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory modes - the unreadable residue cannot be produced")
	}
	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.json")
	snapshots := dumpPath + ".snapshots"
	err := os.MkdirAll(snapshots, 0o700)
	if err != nil {
		t.Fatalf("MkdirAll(%s): %v", snapshots, err)
	}
	err = os.Chmod(snapshots, 0o300)
	if err != nil {
		t.Fatalf("Chmod(%s): %v", snapshots, err)
	}
	t.Cleanup(func() { _ = os.Chmod(snapshots, 0o700) })

	res := persistfile.New(dumpPath).Load(sweepNow, permissiveOracle)
	if res.SweepErr == nil {
		t.Error("Load() sweep error = nil, want the unreadable directory's error")
	}
	if res.Outcome != port.LoadPrompt {
		t.Errorf("Load() outcome = %v, want the load to proceed to the prompt start", res.Outcome)
	}
}
