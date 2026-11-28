package persistfile_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Djarvur/sup-kv/internal/persistfile"
)

var errBad = errors.New("unparseable generation")

// parse accepts anything that starts with "GOOD".
func parse(b []byte) error {
	if bytes.HasPrefix(b, []byte("GOOD")) {
		return nil
	}
	return errBad
}

func ls(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// ENG-05: after a fallback start the first Save rotates the KNOWN-BAD main
// over the only good generation. From then on the chain is one deep; a crash
// in the window between Save's two renames leaves no loadable generation.
func TestReview_corruptMainRotatedOverGoodPrev(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")
	fs := persistfile.New(path)
	if err := fs.Save([]byte("GOOD generation 1")); err != nil {
		t.Fatal(err)
	}
	if err := fs.Save([]byte("GOOD generation 2")); err != nil {
		t.Fatal(err)
	}
	// Bit rot hits main; .prev (generation 1) is the only good copy on disk.
	if err := os.WriteFile(path, []byte("ROTTEN generation 2"), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, ok := persistfile.New(path).Load(parse)
	t.Logf("restart: Load -> %q ok=%t (fell back to .prev, silently)", loaded, ok)

	// Reproduce Save's own sequence up to the instant between its two renames
	// (placeFile: rename(main -> .prev), then rename(tmp -> main)).
	tmp := filepath.Join(dir, "dump-crash")
	if err := os.WriteFile(tmp, []byte("GOOD generation 3"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+".prev"); err != nil { // rename #1 of placeFile
		t.Fatal(err)
	}
	// <- power loss / kill -9 / EIO on rename #2 here.
	loaded, ok = persistfile.New(path).Load(parse)
	prev, _ := os.ReadFile(path + ".prev")
	t.Logf("crash between the renames: files=%v  .prev=%q  Load -> %q ok=%t", ls(t, dir), prev, loaded, ok)
	if !ok {
		t.Errorf("NO LOADABLE GENERATION: the good .prev was overwritten by the main that Load itself had just rejected")
	}
}

// ENG-03: a main dump that EXISTS but cannot be read (EACCES here; EIO or
// EMFILE behave the same) is indistinguishable from "no dump": Load reports
// the empty start with no error, and two ordinary Saves later the good data is
// gone from both generations.
func TestReview_unreadableMainIsSilentEmptyStartThenDestroyed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")
	fs := persistfile.New(path)
	if err := fs.Save([]byte("GOOD 10000 confirmed records")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o000); err != nil { // e.g. restored from backup by root, wrong owner/mode
		t.Fatal(err)
	}

	fs2 := persistfile.New(path)
	loaded, ok := fs2.Load(parse)
	t.Logf("Load with unreadable main: %q ok=%t  (no error value exists in the port signature)", loaded, ok)

	// The node starts empty, the startup gate opens, two writes get dumped.
	for _, gen := range []string{"GOOD 1 record", "GOOD 2 records"} {
		if err := fs2.Save([]byte(gen)); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	_ = os.Chmod(path+".prev", 0o600)
	main, _ := os.ReadFile(path)
	prev, _ := os.ReadFile(path + ".prev")
	t.Logf("after two dumps: main=%q .prev=%q", main, prev)
	if !bytes.Contains(main, []byte("10000")) && !bytes.Contains(prev, []byte("10000")) {
		t.Errorf("the unreadable-but-intact generation with all confirmed records was rotated out and destroyed")
	}
}

// ENG-12: temp files orphaned by a crash are never collected.
func TestReview_orphanTempFilesNeverCollected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")
	// Residue of three crashes that hit between CreateTemp and the rename.
	for _, n := range []string{"dump-111", "dump-222", "dump-333"} {
		if err := os.WriteFile(filepath.Join(dir, n), bytes.Repeat([]byte("x"), 1<<20), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fs := persistfile.New(path)
	fs.Load(parse)
	for range 3 {
		if err := fs.Save([]byte("GOOD")); err != nil {
			t.Fatal(err)
		}
	}
	names := ls(t, dir)
	t.Logf("after Load + 3 Saves: %v", names)
	if len(names) > 2 {
		t.Errorf("orphaned temp files survive startup and every later dump: %v", names)
	}
}

// ENG-17: snapshot names that collide on a case-insensitive filesystem or look
// like the adapter's own temp files are accepted.
func TestReview_snapshotOfMissingMain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")
	fs := persistfile.New(path)
	err := fs.CreateSnapshot("empty")
	t.Logf("CreateSnapshot on a node that never dumped: %v; side effect: %v", err, ls(t, dir))
}
