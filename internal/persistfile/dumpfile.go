// Package persistfile is the sup-kv dump file adapter: the only
// os-importing persistence code, implementing port.DumpStore with atomic
// writes, .prev rotation and the fallback chain (D-01/D-02). The store
// mutex serializes Save against CreateSnapshot: the rotate-first order
// that keeps the crash chain correct (D-02) briefly leaves the main path
// absent between the two renames, and a concurrent snapshot copy must
// never observe that window as ErrNoDump. Concurrent Save calls serialize
// on the same mutex; the engine's single dumper goroutine remains the
// design's serializer. Load and ReadSnapshot stay lock-free — every
// observable generation is always complete bytes.
package persistfile

import (
	"io"
	"os"
	"path/filepath"
	"sync"

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
	// ops is the durability seam Save and CreateSnapshot drive (D-07,
	// TEST-03): the default fileFSOps performs the real file operations;
	// tests substitute recording/failing fakes through this unexported
	// field. Its per-save state is safe in one shared value because mu
	// serializes every save below.
	ops fsops
	// mu serializes Save against CreateSnapshot (and concurrent Saves):
	// between rotate's and rename's renames the main path is briefly
	// absent, and a snapshot copy racing that window would report ErrNoDump
	// despite confirmed data on disk. Load and ReadSnapshot need no lock —
	// each rename leaves every observable generation complete.
	mu sync.Mutex
}

// FileStore implements the whole-dump persistence seam.
var _ port.DumpStore = (*FileStore)(nil)

// New returns a FileStore persisting to dumpPath. The exported constructor
// signature is unchanged by the fsops seam (D-07): the store builds its own
// default fileFSOps; tests inject fakes through the unexported ops field.
func New(dumpPath string) *FileStore {
	return &FileStore{path: dumpPath, ops: &fileFSOps{}}
}

// Save durably writes dump as the new main generation, rotating the old
// main generation to .prev (PERS-02): the byte-slice convenience over
// SaveStream for fixtures and tests. The engine streams its dumps through
// SaveStream, so a real dump cycle never holds a second encoded copy.
func (f *FileStore) Save(dump []byte) error {
	return f.SaveStream(func(w io.Writer) error {
		_, err := w.Write(dump)
		return err
	})
}

// SaveStream durably writes the generation produced by fill as the new main
// generation, rotating the old main generation to .prev (PERS-02, D-06):
// temp file in the target directory — a system-temp rename onto the dump path
// can cross filesystems and fail EXDEV — fill it through the adapter's
// buffered writer, fsync the data before any rename can be durable, rotate,
// rename, then fsync the parent directory so the rename itself becomes
// durable. The rotate-first order leaves a crash window where main is absent
// — exactly the residue the fallback chain absorbs — and the store mutex
// closes that window against a concurrent CreateSnapshot: the copy must never
// see the absent main as ErrNoDump.
func (f *FileStore) SaveStream(fill func(io.Writer) error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return saveAtomically(f.ops, f.path, f.path+prevSuffix, fill)
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
// research Pattern 3) through the fsops seam (D-07, TEST-03): MkdirAll the
// parent so a fresh deployment's first dump creates its directory — setup,
// not one of the five seam steps — then the five ordered operations, one
// fsops method per step, in the exact order the TEST-03 order test asserts:
// write → fsync → rotate → rename → dir-fsync. fill runs inside step 1 on
// the adapter's buffered temp writer, so the encoded generation streams
// instead of arriving as one slice (D-06). An empty prevPath skips the
// rotation — the snapshot policy, which overwrites in place with no .prev
// of its own (research Open Question 2). The temp file is removed on every
// failure path — the seam steps' own hygiene, close + remove (Pitfall 5).
func saveAtomically(ops fsops, path, prevPath string, fill func(io.Writer) error) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, dirMode)
	if err != nil {
		return err
	}
	err = ops.write(dir, fill) // TEST-03 step 1: create temp + stream payload
	if err != nil {
		return err
	}
	err = ops.fsync() // TEST-03 step 2: fsync the temp, close it
	if err != nil {
		return err
	}
	err = ops.rotate(path, prevPath) // TEST-03 step 3: rotate main → .prev
	if err != nil {
		return err
	}
	err = ops.rename(path) // TEST-03 step 4: rename temp → main
	if err != nil {
		return err
	}
	return ops.syncDir(dir) // TEST-03 step 5: fsync the directory
}
