package persistfile_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
)

// fixedPreserveNow is the injected load clock of the preservation tests: a
// whole UTC second, so the derived .corrupt-<stamp> name is exact.
var fixedPreserveNow = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

// preservedName is the first preservation name the adapter derives from
// fixedPreserveNow for path.
func preservedName(path string) string {
	return path + ".corrupt-20261122T120000Z"
}

// TestFileStore_loadBackupPreservesMain proves the D-02 preservation rule and
// its rotation consequence: a corrupt main is renamed out of the rotation
// path with its bytes intact, the failure names the main path and the parse
// cause, the accepted .prev stays untouched, and the first save after the
// backup start leaves .prev byte-identical because the rotate found no main.
func TestFileStore_loadBackupPreservesMain(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.json")
	prevGen := []byte("PREV-" + readableMarker)
	corruptMain := []byte("not json at all")
	writeFile(t, dumpPath, corruptMain)
	writeFile(t, dumpPath+".prev", prevGen)
	store := persistfile.New(dumpPath)

	res := store.Load(fixedPreserveNow, markerOracle)
	if res.Outcome != port.LoadBackup || res.Generation != port.GenPrev {
		t.Fatalf("Load() = (outcome %v, generation %v), want the backup generation", res.Outcome, res.Generation)
	}
	if !bytes.Equal(res.Dump, prevGen) {
		t.Errorf("Load() dump = %q, want .prev's bytes %q", res.Dump, prevGen)
	}
	if len(res.Failures) != 1 {
		t.Fatalf("Load() failures = %d, want exactly the main candidate: %v", len(res.Failures), res.Failures)
	}
	failure := res.Failures[0]
	if failure.Candidate != port.GenMain || failure.Path != dumpPath {
		t.Errorf("failure = (%v, %s), want the main candidate %s", failure.Candidate, failure.Path, dumpPath)
	}
	if !errors.Is(failure.Cause, errUnreadable) {
		t.Errorf("failure cause = %v, want the oracle rejection %v", failure.Cause, errUnreadable)
	}
	if failure.PreserveErr != nil {
		t.Errorf("failure preserve error = %v, want nil", failure.PreserveErr)
	}
	wantPreserved := preservedName(dumpPath)
	if failure.Preserved != wantPreserved {
		t.Fatalf("failure preserved = %q, want %q", failure.Preserved, wantPreserved)
	}
	preserved, err := os.ReadFile(wantPreserved)
	if err != nil || !bytes.Equal(preserved, corruptMain) {
		t.Errorf("preserved bytes = (%q, %v), want the corrupt main's exact bytes", preserved, err)
	}
	_, err = os.Stat(dumpPath)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(main) = %v, want the corrupt main out of the rotation path", err)
	}

	prevBefore, err := os.ReadFile(dumpPath + ".prev")
	if err != nil {
		t.Fatalf("read .prev before the save: %v", err)
	}
	newMain := []byte("NEW-MAIN-" + readableMarker)
	err = store.Save(newMain)
	if err != nil {
		t.Fatalf("Save() after the backup start error = %v, want nil", err)
	}
	prevAfter, err := os.ReadFile(dumpPath + ".prev")
	if err != nil {
		t.Fatalf("read .prev after the save: %v", err)
	}
	if !bytes.Equal(prevBefore, prevAfter) {
		t.Errorf(".prev = %q, want the loaded generation %q untouched by the first save", prevAfter, prevBefore)
	}
	main, err := os.ReadFile(dumpPath)
	if err != nil || !bytes.Equal(main, newMain) {
		t.Errorf("main after the save = (%q, %v), want the fresh generation", main, err)
	}
}

// TestFileStore_preserveCollision proves the same-second collision rule
// (Pitfall 15): the first preserve takes the bare .corrupt-<stamp> name and
// the second one takes -1, so both corrupt generations survive instead of
// the second rename overwriting the first.
func TestFileStore_preserveCollision(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.json")
	prevGen := []byte("PREV-" + readableMarker)
	corruptMain := []byte("not json at all")
	writeFile(t, dumpPath+".prev", prevGen)
	store := persistfile.New(dumpPath)

	writeFile(t, dumpPath, corruptMain)
	first := store.Load(fixedPreserveNow, markerOracle)
	if first.Outcome != port.LoadBackup || len(first.Failures) != 1 {
		t.Fatalf("first Load() = (outcome %v, %d failures), want a backup with one main failure",
			first.Outcome, len(first.Failures))
	}
	firstPreserved := preservedName(dumpPath)
	if got := first.Failures[0].Preserved; got != firstPreserved {
		t.Fatalf("first preserved = %q, want %q", got, firstPreserved)
	}

	writeFile(t, dumpPath, corruptMain)
	second := store.Load(fixedPreserveNow, markerOracle)
	if second.Outcome != port.LoadBackup || len(second.Failures) != 1 {
		t.Fatalf("second Load() = (outcome %v, %d failures), want a backup with one main failure",
			second.Outcome, len(second.Failures))
	}
	secondPreserved := firstPreserved + "-1"
	if got := second.Failures[0].Preserved; got != secondPreserved {
		t.Fatalf("second preserved = %q, want the collision suffix %q", got, secondPreserved)
	}
	for _, path := range []string{firstPreserved, secondPreserved} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, corruptMain) {
			t.Errorf("preserved generation %s = (%q, %v), want the corrupt main's bytes", path, got, err)
		}
	}
}
