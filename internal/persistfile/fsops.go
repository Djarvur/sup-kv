package persistfile

import (
	"bufio"
	"io"
	"os"
)

// maxDumpBufioBytes is the adapter's writer buffer for one streamed dump
// generation: the fill writes through this buffer, so the engine never has
// to materialize the encoded dump (D-06). 64 KiB is large enough that the
// syscall count stays near one per line batch and small enough that the
// buffer is a rounding error against a 6 MB state (PERSIST-06).
const maxDumpBufioBytes = 64 << 10

// fsops is the filesystem durability seam TEST-03 pins (D-07): the five
// ordered operations one atomic dump generation performs — write → fsync →
// rotate → rename → dir-fsync — one method per step, so a recording fake can
// assert the exact order and a failing fake can inject a failure at each
// step. The seam is persistfile-local by design (D-07, threat T-2-07):
// internal/app may not import os (the sup-kv-app-no-io depguard rule), so
// the default implementation stays the real file operations here in the
// adapter, and the recording/failing/blocking fakes live only in _test.go
// files — never in the shipped default.
//
// Per-save state (the open temp handle and its path) lives in the
// implementation value: FileStore's store mutex serializes Save against
// CreateSnapshot and against concurrent Saves, so one implementation value
// is never mid-save twice.
//
// The platform durability detail deliberately lives in the standard
// library, not here: [os.File.Sync] on darwin goes through internal/poll's
// darwin Fsync, which issues F_FULLFSYNC with EINTR retry and an ENOTSUP
// fallback to plain fsync — a strict superset of the [syscall.Fsync] path a
// hand-rolled helper would reach. The seam therefore calls [os.File.Sync]
// directly and carries no platform file of its own (REVIEW ENG-18, P-09;
// Phase 5's P-09 absorbs the remaining orphan-temp/parent-dir work).
type fsops interface {
	// write creates the temp file in dir and lets fill write the payload
	// into it — TEST-03 order step 1, the open-temp-and-fill half. The
	// fill runs through a maxDumpBufioBytes buffer, so the caller streams
	// format lines instead of handing over a whole encoded generation
	// (D-06, PERSIST-06). The open temp handle stays with the
	// implementation; fsync closes it. A failure after the temp exists
	// closes and removes it — the temp-holding steps never litter
	// (Pitfall 5).
	write(dir string, fill func(io.Writer) error) error

	// fsync durably syncs the held temp handle and closes it — TEST-03
	// order step 2, the data-is-durable-before-any-rename half, through
	// [os.File.Sync] (the standard library's darwin path issues F_FULLFSYNC
	// with EINTR/ENOTSUP handling). On failure — a failed Sync or a failed
	// Close — the implementation closes the handle and removes the temp.
	fsync() error

	// rotate moves the current main generation to prev — TEST-03 order
	// step 3, the rotate-first half that keeps the crash chain correct
	// (D-02). An empty prev skips the rotation — the snapshot policy — and
	// targetExists(main) is the guard deciding a previous generation
	// exists at all; a first save has nothing to rotate. On failure the
	// implementation removes the held temp.
	rotate(main, prev string) error

	// rename moves the temp file onto main — TEST-03 order step 4, the
	// atomic replace on Unix. On failure the implementation removes the
	// held temp; a failure after rotate succeeded leaves exactly the
	// crash-window residue the fallback chain absorbs (main absent, the
	// old generation on .prev).
	rename(main string) error

	// syncDir durably syncs dir — TEST-03 order step 5, making the
	// completed rename itself durable, through the same platform flush as
	// the file data. Nothing to clean on failure: the temp already became
	// the main generation.
	syncDir(dir string) error
}

// fileFSOps is the default fsops implementation — the exact real operations
// the save path performed before the extraction (threat T-2-07): CreateTemp
// in dir, Write, [os.File.Sync], the rotate rename guarded by targetExists,
// the replace rename, and the directory fsync. A swapped or shortened default
// fails the existing dumpfile suite and the TEST-03 order test. The temp
// handle and path are the per-save state; FileStore serializes its saves,
// so the shared value is safe.
type fileFSOps struct {
	tmp     *os.File // the open temp handle write holds for fsync
	tmpName string   // the temp path, for rename and the failure hygiene
}

// The compiler enforces the seam at every FileStore construction; the
// assertion documents it.
var _ fsops = (*fileFSOps)(nil)

// write implements TEST-03 order step 1 (see fsops): CreateTemp in dir —
// the caller's MkdirAll already made dir exist — then run fill through a
// maxDumpBufioBytes buffer and flush it. The open handle and path are held
// for fsync. A fill or flush failure returns the error with the temp still
// held; the failure-hygiene completion consumes that state next.
func (o *fileFSOps) write(dir string, fill func(io.Writer) error) error {
	tmp, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return err
	}
	o.tmp, o.tmpName = tmp, tmp.Name()
	bufw := bufio.NewWriterSize(tmp, maxDumpBufioBytes)
	err = fill(bufw)
	if err != nil {
		return err
	}
	return bufw.Flush()
}

// fsync implements TEST-03 order step 2 (see fsops): [os.File.Sync] on the
// held handle, then Close. Any failure closes the handle and removes the
// temp — exactly the hygiene syncedTemp performed on its Sync/Close
// failures before the extraction; on success the handle is consumed and
// only the path survives, for rename.
func (o *fileFSOps) fsync() error {
	err := o.tmp.Sync()
	if err == nil {
		err = o.tmp.Close()
	}
	if err != nil {
		o.closeRemoveTemp()
		return err
	}
	o.tmp = nil
	return nil
}

// rotate implements TEST-03 order step 3 (see fsops): an existing main
// generation renames onto prev, so the fallback chain always has one
// complete generation behind the one being placed. No previous generation
// (first save, snapshot overwrite) is the no-op the guard existed for in
// placeFile before the extraction.
func (o *fileFSOps) rotate(main, prev string) error {
	if prev == "" || !targetExists(main) {
		return nil
	}
	err := os.Rename(main, prev)
	if err != nil {
		o.removeTemp()
		return err
	}
	return nil
}

// rename implements TEST-03 order step 4 (see fsops): the held temp path
// renames onto main — the atomic replace. A failure removes the temp; the
// rotate already performed stays performed, the crash-window residue D-02's
// fallback chain absorbs.
func (o *fileFSOps) rename(main string) error {
	err := os.Rename(o.tmpName, main)
	if err != nil {
		o.removeTemp()
		return err
	}
	o.tmpName = ""
	return nil
}

// syncDir implements TEST-03 order step 5 (see fsops): the parent
// directory's entry is fsynced through [os.File.Sync], so the rename under
// it becomes durable too; the file's own data was synced before the rename.
// The failure path cleans nothing — the temp is already the main
// generation.
func (*fileFSOps) syncDir(dir string) error {
	d, err := os.Open(dir) //nolint:gosec // G304: dir derives from the configured dump path, never caller input
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}

// closeRemoveTemp is the failure hygiene of the temp-holding steps (write,
// fsync): the held handle is closed — closing an already-closed file just
// errors, ignored — and the temp is removed, so a failed save never litters
// dump-* files (Pitfall 5).
func (o *fileFSOps) closeRemoveTemp() {
	if o.tmp != nil {
		_ = o.tmp.Close()
		o.tmp = nil
	}
	o.removeTemp()
}

// removeTemp removes the held temp path — the failure hygiene of the
// post-close steps (rotate, rename): the handle is already closed, the
// temp must not litter.
func (o *fileFSOps) removeTemp() {
	if o.tmpName != "" {
		_ = os.Remove(o.tmpName)
		o.tmpName = ""
	}
}

// targetExists reports whether path is statable — the guard deciding
// whether a previous generation exists to rotate.
func targetExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
