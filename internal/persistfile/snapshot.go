package persistfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Snapshot-name domain: the length bound of a validated name and the
// .snapshots directory suffix derived from the dump path — snapshots live
// beside the dump with no extra launch parameter (D-08).
const (
	snapshotDirSuffix = ".snapshots"
	maxSnapshotName   = 64
)

// Sentinel errors of the snapshot half of the adapter.
var (
	// ErrNoDump is returned by CreateSnapshot when no main dump exists on
	// disk to copy.
	ErrNoDump = errors.New("no dump on disk to snapshot")

	// ErrInvalidSnapshotName is returned for a name outside the validated
	// charset or length bounds, or one of the two dot special cases —
	// validation runs before any filesystem touch (Security V5).
	ErrInvalidSnapshotName = errors.New("invalid snapshot name")
)

// CreateSnapshot byte-copies the current on-disk main dump under name in
// the .snapshots directory beside the dump (D-08), atomically overwriting
// an earlier snapshot of the same name with no .prev of its own —
// snapshots are operator-named recovery points, not crash generations; the
// dump's own .prev chain is the crash mechanism.
func (f *FileStore) CreateSnapshot(name string) error {
	err := validateName(name)
	if err != nil {
		return err
	}
	snapshotDir := f.path + snapshotDirSuffix
	err = os.MkdirAll(snapshotDir, dirMode)
	if err != nil {
		return err
	}
	data, err := readMainDump(f.path)
	if err != nil {
		return err
	}
	return saveAtomically(filepath.Join(snapshotDir, name), "", data)
}

// ReadSnapshot returns the named snapshot's bytes for codec decoding; a
// missing snapshot surfaces the wrapped fs error.
func (f *FileStore) ReadSnapshot(name string) ([]byte, error) {
	err := validateName(name)
	if err != nil {
		return nil, err
	}
	snapshotPath := filepath.Join(f.path+snapshotDirSuffix, name)
	return os.ReadFile(snapshotPath) //nolint:gosec // G304: the joined path is a validated name under the configured snapshots dir
}

// readMainDump returns the main dump's bytes; no main dump on disk is
// ErrNoDump, any other read fault surfaces unchanged.
func readMainDump(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the path is the configured dump path, never caller input
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoDump
	}
	return data, err
}

// validNameByte reports whether c is inside the validated snapshot-name
// charset [A-Za-z0-9._-] — a byte-wise test, so any multi-byte UTF-8
// sequence fails it on the continuation bytes alone.
func validNameByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '.', c == '_', c == '-':
		return true
	default:
		return false
	}
}

// validateName rejects names outside the charset [A-Za-z0-9._-] with length
// 1..64, plus the two dot special cases the charset alone leaves open —
// together the exclusions close the path-traversal hole no allow-list of
// separators-free names otherwise would (Security V5, Pitfall 10).
func validateName(name string) error {
	if name == "" || len(name) > maxSnapshotName || name == "." || name == ".." {
		return ErrInvalidSnapshotName
	}
	for i := range name {
		if !validNameByte(name[i]) {
			return ErrInvalidSnapshotName
		}
	}
	return nil
}
