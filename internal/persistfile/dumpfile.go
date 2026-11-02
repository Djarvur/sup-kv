// Package persistfile is the sup-kv dump file adapter: the only
// os-importing persistence code, implementing port.DumpStore with atomic
// writes, .prev rotation and the fallback chain (D-01/D-02). Save is not
// safe for concurrent use against one path — the engine's single dumper
// goroutine serializes every save; concurrent saves would race the .prev
// rotation and corrupt the fallback chain.
package persistfile

import (
	"os"
	"path/filepath"

	"github.com/Djarvur/sup-kv/internal/port"
)

// Adapter file modes and path suffixes: the dump directory and the
// snapshots directory are created 0700 (values privacy over non-owner
// readability; grep still works for the owner), the dump files keep
// [os.CreateTemp]'s 0600 default, and prevSuffix is the previous
// generation's path suffix — the fallback chain walks main then
// main+prevSuffix (D-02, D-08).
const (
	prevSuffix  = ".prev"
	tempPattern = "dump-*"
	dirMode     = 0o700
)

// FileStore persists whole dump bytes at one main file path, deriving the
// .prev generation and the .snapshots directory from it (D-08).
type FileStore struct {
	path string
}

// FileStore implements the whole-dump persistence seam.
var _ port.DumpStore = (*FileStore)(nil)

// New returns a FileStore persisting to dumpPath.
func New(dumpPath string) *FileStore {
	return &FileStore{path: dumpPath}
}

// Save durably writes dump as the new main generation, rotating the old
// main generation to .prev (PERS-02): temp file in the target directory —
// a system-temp rename onto the dump path can cross filesystems and fail
// EXDEV — write, fsync the data before any rename can be durable, rotate,
// rename, then fsync the parent directory so the rename itself becomes
// durable. The rotate-first order leaves a crash window where main is
// absent — exactly the residue the fallback chain absorbs.
func (f *FileStore) Save(dump []byte) error {
	return saveAtomically(f.path, f.path+prevSuffix, dump)
}

// Load resolves the fallback chain main then .prev and returns the first
// generation whose bytes pass parse; a read error or an oracle rejection —
// a truncated or unparseable file counts as unreadable, D-02 — advances
// the chain. Both generations exhausted is the empty start, ok false.
// Every per-generation fault is chain business, so Load never reports an
// error.
func (f *FileStore) Load(parse func([]byte) error) (dump []byte, ok bool) {
	for _, candidate := range []string{f.path, f.path + prevSuffix} {
		data, err := os.ReadFile(candidate) //nolint:gosec // G304: candidate derives from the configured dump path, never caller input
		if err != nil {
			continue
		}
		if parse(data) != nil {
			continue
		}
		return data, true
	}
	return nil, false
}

// saveAtomically implements the canonical atomic-write sequence (PERS-02,
// research Pattern 3): MkdirAll the parent so a fresh deployment's first
// dump creates its directory, a synced temp file in that directory,
// optionally rotate an existing target to prevPath, rename the temp over
// the target, then fsync the parent directory. An empty prevPath skips the
// rotation — the snapshot policy, which overwrites in place with no .prev
// of its own (research Open Question 2). The temp file is removed on every
// failure path — [os.CreateTemp] is caller-managed and must not litter
// (Pitfall 5).
func saveAtomically(path, prevPath string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, dirMode)
	if err != nil {
		return err
	}
	tmpName, err := syncedTemp(dir, data)
	if err != nil {
		return err
	}
	err = placeFile(path, prevPath, tmpName)
	if err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return syncDir(dir)
}

// syncedTemp creates a temp file in dir, writes data into it and fsyncs
// and closes it: the data is durable before any rename can be. The temp
// file is removed on every failure path.
func syncedTemp(dir string, data []byte) (tmpName string, err error) {
	tmp, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return "", err
	}
	tmpName = tmp.Name()
	err = writeSynced(tmp, data)
	if err == nil {
		err = tmp.Close()
	}
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", err
	}
	return tmpName, nil
}

// writeSynced writes data into tmp and fsyncs it.
func writeSynced(tmp *os.File, data []byte) error {
	_, err := tmp.Write(data)
	if err != nil {
		return err
	}
	return tmp.Sync()
}

// placeFile rotates an existing target to prevPath — only when prevPath is
// non-empty, the dump policy — then renames tmpName onto the target: the
// atomic replace on Unix.
func placeFile(path, prevPath, tmpName string) error {
	if prevPath != "" && targetExists(path) {
		err := os.Rename(path, prevPath)
		if err != nil {
			return err
		}
	}
	return os.Rename(tmpName, path)
}

// targetExists reports whether path is statable — the guard deciding
// whether a previous generation exists to rotate.
func targetExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// syncDir fsyncs the directory entry so a completed rename under it
// becomes durable too; the file's own data was synced before the rename.
func syncDir(dir string) error {
	d, err := os.Open(dir) //nolint:gosec // G304: dir derives from the configured dump path, never caller input
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}
