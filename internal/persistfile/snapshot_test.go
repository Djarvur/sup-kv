package persistfile_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// entryNames returns dir's entry names sorted — the directory-content scan
// proving a rejected snapshot name never produced a file anywhere.
func entryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v, want nil", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names
}

// TestFileStore_nameValidation proves the snapshot-name security boundary
// (T-3-04, Security V5, PERS-05): CreateSnapshot and ReadSnapshot reject
// every traversal, separator, dot-special, out-of-charset and over-length
// shape with ErrInvalidSnapshotName BEFORE any filesystem touch, while the
// valid spellings — the 64-char length cap's boundary included — pass and
// read back byte-equal, and nothing ever escapes the .snapshots directory.
func TestFileStore_nameValidation(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	store := persistfile.New(dumpPath)
	// Two saves so the directory scan sees the settled main/.prev/.snapshots
	// shape rather than the first-save no-prev window.
	for _, gen := range [][]byte{[]byte("GENERATION-ONE"), []byte("GENERATION-TWO")} {
		err := store.Save(gen)
		if err != nil {
			t.Fatalf("seed Save() error = %v, want nil", err)
		}
	}
	main, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read the main dump: %v", err)
	}

	tests := []struct {
		name  string
		valid bool
	}{
		{name: "../evil", valid: false},
		{name: "a/b", valid: false},
		{name: `a\b`, valid: false},
		{name: "", valid: false},
		{name: ".", valid: false},
		{name: "..", valid: false},
		{name: strings.Repeat("n", 65), valid: false},
		{name: "has space", valid: false},
		{name: "v1", valid: true},
		{name: "pre-2026_11.03", valid: true},
		{name: "backup_1", valid: true},
		{name: strings.Repeat("n", 64), valid: true},
	}
	for _, tt := range tests {
		if tt.valid {
			err := store.CreateSnapshot(tt.name)
			if err != nil {
				t.Errorf("CreateSnapshot(%q) error = %v, want nil", tt.name, err)
				continue
			}
			got, err := store.ReadSnapshot(tt.name)
			if err != nil || !bytes.Equal(got, main) {
				t.Errorf("ReadSnapshot(%q) = (%q, %v), want the main dump's bytes byte-equal",
					tt.name, got, err)
			}
			continue
		}
		err := store.CreateSnapshot(tt.name)
		if !errors.Is(err, persistfile.ErrInvalidSnapshotName) {
			t.Errorf("CreateSnapshot(%q) error = %v, want ErrInvalidSnapshotName", tt.name, err)
		}
		_, err = store.ReadSnapshot(tt.name)
		if !errors.Is(err, persistfile.ErrInvalidSnapshotName) {
			t.Errorf("ReadSnapshot(%q) error = %v, want ErrInvalidSnapshotName", tt.name, err)
		}
	}

	// The rejected names never produced a file: the dump's parent holds
	// exactly the dump, its .prev and the .snapshots directory, and
	// .snapshots holds exactly the valid names.
	wantParent := []string{"dump.json", "dump.json.prev", "dump.json.snapshots"}
	if got := entryNames(t, filepath.Dir(dumpPath)); !slices.Equal(got, wantParent) {
		t.Errorf("dump directory entries = %v, want exactly %v", got, wantParent)
	}
	wantSnapshots := []string{"backup_1", strings.Repeat("n", 64), "pre-2026_11.03", "v1"}
	if got := entryNames(t, dumpPath+".snapshots"); !slices.Equal(got, wantSnapshots) {
		t.Errorf("snapshot directory entries = %v, want exactly the valid names %v", got, wantSnapshots)
	}

	// The strongest ordering proof: with NO dump on disk at all, an invalid
	// name still rejects with the sentinel — never ErrNoDump or a wrapped fs
	// error — so validateName provably ran before any filesystem touch.
	fresh := persistfile.New(filepath.Join(t.TempDir(), "dump.json"))
	err = fresh.CreateSnapshot("../evil")
	if !errors.Is(err, persistfile.ErrInvalidSnapshotName) {
		t.Errorf("fresh CreateSnapshot(../evil) error = %v, want ErrInvalidSnapshotName before any filesystem touch", err)
	}
	_, err = fresh.ReadSnapshot("a/b")
	if !errors.Is(err, persistfile.ErrInvalidSnapshotName) {
		t.Errorf("fresh ReadSnapshot(a/b) error = %v, want ErrInvalidSnapshotName before any filesystem touch", err)
	}
}

// TestFileStore_snapshotOverwriteAtomic proves the snapshot overwrite
// policy (plan 03-02 flagged assumption 2): re-creating an existing name
// atomically replaces it in place with the later dump's bytes exactly —
// no .prev sibling accumulates beside it.
func TestFileStore_snapshotOverwriteAtomic(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	store := persistfile.New(dumpPath)
	gen1 := []byte("GENERATION-ONE")
	gen2 := []byte("GENERATION-TWO")
	for _, gen := range [][]byte{gen1, gen2} {
		err := store.Save(gen)
		if err != nil {
			t.Fatalf("seed Save() error = %v, want nil", err)
		}
	}
	for range 2 {
		err := store.CreateSnapshot("tag")
		if err != nil {
			t.Fatalf("CreateSnapshot(tag) error = %v, want nil", err)
		}
	}
	got, err := store.ReadSnapshot("tag")
	if err != nil || !bytes.Equal(got, gen2) {
		t.Errorf("overwritten snapshot = (%q, %v), want generation two's bytes exactly", got, err)
	}
	if names := entryNames(t, dumpPath+".snapshots"); len(names) != 1 || names[0] != "tag" {
		t.Errorf("snapshot directory entries = %v, want exactly [tag] — no .prev accumulation", names)
	}
}

// TestFileStore_snapshotErrorClasses proves the two snapshot error classes
// stay distinguishable (PERS-05): snapshotting without a main dump on disk
// is ErrNoDump, reading a missing snapshot is an error wrapping
// fs.ErrNotExist — never a silent empty success.
func TestFileStore_snapshotErrorClasses(t *testing.T) {
	t.Parallel()
	store := persistfile.New(filepath.Join(t.TempDir(), "dump.json"))
	err := store.CreateSnapshot("too-early")
	if !errors.Is(err, persistfile.ErrNoDump) {
		t.Errorf("CreateSnapshot without a main dump = %v, want ErrNoDump", err)
	}
	_, err = store.ReadSnapshot("missing")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadSnapshot(missing) = %v, want an error wrapping fs.ErrNotExist", err)
	}
}
