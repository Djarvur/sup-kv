// Package persistfile is the sup-kv dump file adapter: the only
// os-importing persistence code, implementing port.DumpStore with atomic
// writes, .prev rotation and the fallback chain (D-01/D-02). Load also
// preserves every rejected generation under .corrupt-<UTC> and sweeps old
// orphan temps at startup (D-02, D-08) — no load ever deletes evidence. The
// store mutex serializes SaveStream against CreateSnapshot: the rotate-first
// order that keeps the crash chain correct (D-02) briefly leaves the main
// path absent between the two renames, and a concurrent snapshot copy must
// never observe that window as ErrNoDump. Concurrent save calls serialize
// on the same mutex; the engine's single dumper goroutine remains the
// design's serializer. Load and ReadSnapshot stay lock-free — every
// observable generation is always complete bytes.
package persistfile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Djarvur/sup-kv/internal/port"
)

// Adapter file modes, temp marker and path suffixes: the dump directory and
// the snapshots directory are created 0700 (values privacy over non-owner
// readability; grep still works for the owner), the dump files keep
// [os.CreateTemp]'s 0600 default, and prevSuffix is the previous
// generation's path suffix — the fallback chain walks main then
// main+prevSuffix (D-02, D-08). A generation the load cannot accept is
// preserved under corruptSuffix plus a UTC stamp (stampLayout), never
// deleted: the corrupted bytes are the evidence an operator inspects
// (D-02, T-5-12). Every in-flight temp carries orphanTempMarker: '~' is
// outside the snapshot-name charset [A-Za-z0-9._-], so no legal snapshot or
// dump generation can contain it and the startup sweep can select temps
// without keying on a data-file prefix (D-08, T-5-14). Old temps are
// removed only after defaultOrphanTempAge, so a concurrent instance's
// in-flight write is never swept.
const (
	prevSuffix           = ".prev"
	corruptSuffix        = ".corrupt-"
	stampLayout          = "20060102T150405Z"
	orphanTempMarker     = ".tmp~"
	tempPattern          = orphanTempMarker + "*"
	defaultOrphanTempAge = time.Hour
	dirMode              = 0o700
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
	// mu serializes SaveStream against CreateSnapshot (and concurrent save
	// calls): between rotate's and rename's renames the main path is briefly
	// absent, and a snapshot copy racing that window would report ErrNoDump
	// despite confirmed data on disk. Load and ReadSnapshot need no lock —
	// each rename leaves every observable generation complete.
	mu sync.Mutex
}

// FileStore implements the whole-dump persistence seam.
var _ port.DumpStore = (*FileStore)(nil)

// errNotRegular is the preservation rejection of a candidate that is not a
// regular file: only regular files are renamed out of the rotation path, so
// no symlink is followed and no device is touched (T-5-13).
var errNotRegular = errors.New("dump path is not a regular file")

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

// Load runs the startup sweep first, then resolves the fallback chain main
// then .prev and reports the first generation whose bytes pass parse
// together with the outcome, accepted generation and every candidate fault
// (D-02). A read error or an oracle rejection — a truncated or unparseable
// file counts as unreadable — is recorded against its candidate, preserved
// under a collision-free .corrupt-<UTC> name and the chain advances; an
// absent candidate is not a failure. No candidate at all is the prompt
// start, and candidates that all fail are unusable. Presence is never lost
// from the report: when .prev is accepted after an absent main, the main
// candidate is recorded with [fs.ErrNotExist] as its cause, so the engine's
// single Backup WARN always names why the main was not used. Preservation
// keeps the loaded .prev alive: the corrupt main has left the rotation path
// before any save, so the first rotate finds no main and is a no-op. now is
// the clock that names a preserved generation; the adapter stays clock-free.
func (f *FileStore) Load(now time.Time, parse func([]byte) error) port.LoadResult {
	var result port.LoadResult
	result.Swept, result.SweepErr = sweepOrphans(f.path, now)
	candidates := []struct {
		generation port.Generation
		path       string
	}{
		{port.GenMain, f.path},
		{port.GenPrev, f.path + prevSuffix},
	}
	mainAbsent := false
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate.path) // safe: the path derives from the configured dump path, never caller input
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if candidate.generation == port.GenMain {
					mainAbsent = true
				}
				continue // absent is the prompt start's shape, never a failure.
			}
			result.Failures = append(result.Failures, failedCandidate(now, candidate.generation, candidate.path, err))
			continue
		}
		err = parse(data)
		if err != nil {
			result.Failures = append(result.Failures, failedCandidate(now, candidate.generation, candidate.path, err))
			continue
		}
		result.Dump = data
		result.Generation = candidate.generation
		if candidate.generation == port.GenPrev {
			if mainAbsent {
				result.Failures = append([]port.GenerationFailure{{
					Candidate: port.GenMain,
					Path:      f.path,
					Cause:     fs.ErrNotExist,
				}}, result.Failures...)
			}
			result.Outcome = port.LoadBackup
		} else {
			result.Outcome = port.LoadMain
		}
		return result
	}
	if len(result.Failures) > 0 {
		result.Outcome = port.LoadUnusable
	}
	return result
}

// failedCandidate records one candidate the load could not accept and moves
// its bytes out of the rotation path: the failure carries the candidate, its
// path and the read or parse cause, plus the preservation result (D-02,
// T-5-12). A read error and an oracle rejection are the same evidence class.
func failedCandidate(
	now time.Time,
	candidate port.Generation,
	path string,
	cause error,
) port.GenerationFailure {
	failure := port.GenerationFailure{
		Candidate: candidate,
		Path:      path,
		Cause:     cause,
	}
	failure.Preserved, failure.PreserveErr = preserve(now, path)
	return failure
}

// preserve renames the never-loaded candidate at path out of the rotation
// path under <path>+corruptSuffix+<UTC stamp> (D-02, T-5-12). Lstat decides
// regularity — a directory, device or symlink at the dump path is a failure
// with a cause and no rename, so no link is ever followed; [os.Rename]
// replaces the renamed entry itself (the directory is 0700, so the probe
// window is owner-only). Two loads in the same second would derive the same
// stamp, so base, base-1, base-2 … are probed with Lstat and the first free
// name wins, never overwriting earlier evidence (Pitfall 15). After a
// successful rename the parent directory is synced best-effort: the
// evidence's durability must not turn a load into an error.
func preserve(now time.Time, path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", &os.PathError{Op: "preserve", Path: path, Err: errNotRegular}
	}
	base := path + corruptSuffix + now.UTC().Format(stampLayout)
	for n := 0; ; n++ {
		candidate := base
		if n > 0 {
			candidate = fmt.Sprintf("%s-%d", base, n)
		}
		_, err = os.Lstat(candidate)
		if errors.Is(err, fs.ErrNotExist) {
			err = os.Rename(path, candidate)
			if err != nil {
				return "", err
			}
			syncDirBestEffort(filepath.Dir(path))
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
}

// syncDirBestEffort fsyncs dir and discards every error — the preservation
// path's evidence-durability attempt, which must never fail a load.
func syncDirBestEffort(dir string) {
	d, err := os.Open(dir) //nolint:gosec // G304: dir derives from the configured dump path, never caller input
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}

// sweepOrphans removes old orphan temp files from the dump directory and the
// snapshots directory beside it (D-08, PERSIST-09): every candidate carries
// orphanTempMarker, is a regular file and predates defaultOrphanTempAge, so
// a concurrent instance's in-flight write and every live data file survive.
// A missing directory is not an error; the count and the first error are
// returned so Load can report them without aborting the chain.
func sweepOrphans(mainPath string, now time.Time) (int, error) {
	dumpDir := filepath.Dir(mainPath)
	mainBase := filepath.Base(mainPath)
	var swept int
	var firstErr error
	for _, dir := range []string{dumpDir, mainPath + snapshotDirSuffix} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) && firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, entry := range entries {
			if dir == dumpDir && protectedDataName(entry.Name(), mainBase) {
				continue
			}
			removed, removeErr := sweepEntry(dir, entry.Name(), now)
			if removeErr != nil && firstErr == nil {
				firstErr = removeErr
			}
			if removed {
				swept++
			}
		}
	}
	return swept, firstErr
}

// protectedDataName reports whether name is the configured main, its .prev
// or one of its .corrupt-* evidence names — the dump-directory exclusions
// that protect even a main whose own name carries the temp marker.
func protectedDataName(name, mainBase string) bool {
	return name == mainBase ||
		name == mainBase+prevSuffix ||
		strings.HasPrefix(name, mainBase+corruptSuffix)
}

// sweepEntry decides one directory entry and removes it when it is an old
// orphan temp: the marker is the only selection key, so no data-file prefix
// can match. No symlink is followed (Lstat) and no young file is touched.
func sweepEntry(dir, name string, now time.Time) (bool, error) {
	if !strings.Contains(name, orphanTempMarker) {
		return false, nil
	}
	path := filepath.Join(dir, name)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() || !info.ModTime().Before(now.Add(-defaultOrphanTempAge)) {
		return false, nil
	}
	err = os.Remove(path)
	return err == nil, err
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
