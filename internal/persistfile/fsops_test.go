package persistfile //nolint:testpackage // the plan mandates the internal test here so the fakes reach the unexported FileStore.ops field — the injection point (D-07); testpackage's external-package rule cannot express that pairing.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// fsops-test constants: the three dump generations the order and failure
// tests move around, and the injected step failure the failing fake
// returns.
const (
	fsopsOldGen      = "FSOPS-OLD-GENERATION"
	fsopsNewGen      = "FSOPS-NEW-GENERATION"
	fsopsRecovered   = "FSOPS-RECOVERED-GENERATION"
	prevName         = "dump.json" + prevSuffix
	fsopsInjectedMsg = "injected fsops step failure"
)

// errFsopsInjected is the failing fake's injected error; every failure-row
// assertion requires exactly it, so a step that fails for the wrong reason
// cannot pass.
var errFsopsInjected = errors.New(fsopsInjectedMsg)

// fsopStep names one seam step — the recording fake reports it and the
// failing fake picks its injection point by it.
type fsopStep string

// The five TEST-03 order steps, in seam order: write → fsync → rotate →
// rename → dir-fsync.
const (
	fsopWrite   fsopStep = "write"
	fsopFsync   fsopStep = "fsync"
	fsopRotate  fsopStep = "rotate"
	fsopRename  fsopStep = "rename"
	fsopSyncDir fsopStep = "syncDir"
)

// fsopsOrder is the exact per-Save seam sequence TEST-03 pins (D-07): one
// ordered slice, compared by equality, never by containment.
var fsopsOrder = []fsopStep{fsopWrite, fsopFsync, fsopRotate, fsopRename, fsopSyncDir}

// fsopCall is one recorded seam call: the step and its argument paths
// (fsync carries none — its state is the held temp handle).
type fsopCall struct {
	op   fsopStep
	args []string
}

// recordingFSOps is the order-assertion fake: every fsops call is recorded
// — op plus argument paths — and then delegated to the embedded real
// fileFSOps, so the recorded sequence is the production Save path's actual
// sequence and the rotation subtest sees the real files the delegation
// placed. Recording only (no failAt): the fake never alters behavior.
type recordingFSOps struct {
	fileFSOps

	calls []fsopCall
}

func (r *recordingFSOps) write(dir string, fill func(io.Writer) error) error {
	r.calls = append(r.calls, fsopCall{fsopWrite, []string{dir}})
	return r.fileFSOps.write(dir, fill)
}

func (r *recordingFSOps) fsync() error {
	r.calls = append(r.calls, fsopCall{fsopFsync, nil})
	return r.fileFSOps.fsync()
}

func (r *recordingFSOps) rotate(main, prev string) error {
	r.calls = append(r.calls, fsopCall{fsopRotate, []string{main, prev}})
	return r.fileFSOps.rotate(main, prev)
}

func (r *recordingFSOps) rename(main string) error {
	r.calls = append(r.calls, fsopCall{fsopRename, []string{main}})
	return r.fileFSOps.rename(main)
}

func (r *recordingFSOps) syncDir(dir string) error {
	r.calls = append(r.calls, fsopCall{fsopSyncDir, []string{dir}})
	return r.fileFSOps.syncDir(dir)
}

// recordedOps returns the recorded step names in call order.
func (r *recordingFSOps) recordedOps() []fsopStep {
	names := make([]fsopStep, 0, len(r.calls))
	for _, call := range r.calls {
		names = append(names, call.op)
	}
	return names
}

// failingFSOps is the failure-matrix fake: every step delegates to the
// embedded real fileFSOps except failAt, which returns the injected error.
// Each short-circuit performs the same temp hygiene the production step's
// own failure path performs — the shared production helpers
// closeRemoveTemp/removeTemp, not fake-side bookkeeping — so the matrix
// observes the real dump hygiene (no partial main, no temp litter) the
// seam contract promises.
type failingFSOps struct {
	fileFSOps

	failAt fsopStep
	err    error
}

func (f *failingFSOps) write(dir string, fill func(io.Writer) error) error {
	if f.failAt == fsopWrite {
		return f.err // nothing is held yet: no temp exists to clean
	}
	return f.fileFSOps.write(dir, fill)
}

func (f *failingFSOps) fsync() error {
	if f.failAt == fsopFsync {
		f.closeRemoveTemp() // the step's own hygiene: handle closed + temp removed
		return f.err
	}
	return f.fileFSOps.fsync()
}

func (f *failingFSOps) rotate(main, prev string) error {
	if f.failAt == fsopRotate {
		f.removeTemp() // the failed step's hygiene: the temp must not litter
		return f.err
	}
	return f.fileFSOps.rotate(main, prev)
}

func (f *failingFSOps) rename(main string) error {
	if f.failAt == fsopRename {
		f.removeTemp()
		return f.err
	}
	return f.fileFSOps.rename(main)
}

func (f *failingFSOps) syncDir(dir string) error {
	if f.failAt == fsopSyncDir {
		return f.err // nothing to clean: the temp already became main
	}
	return f.fileFSOps.syncDir(dir)
}

// fsopsGeneration asserts one on-disk generation: name "" means the file
// must be absent, any literal means the file's bytes are exactly it — a
// partial or mixed main fails the equality, the marker-oracle style of
// dumpfile_test.go.
func fsopsGeneration(t *testing.T, path, want string) {
	t.Helper()
	if want == "" {
		_, statErr := os.Stat(path)
		if !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("Stat(%s) = %v, want the generation absent", path, statErr)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if !bytes.Equal(got, []byte(want)) {
		t.Errorf("generation %s = %q, want exactly %q (no partial main)", path, got, want)
	}
}

// fsopsEntries returns dir's sorted base names — the temp-litter oracle.
func fsopsEntries(t *testing.T, dir string) []string {
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

// TestFsops_saveOperationOrder pins the TEST-03 order (D-07): one Save
// through FileStore drives the seam in exactly write → fsync → rotate →
// rename → dir-fsync — the recording fake's op-name list compared as one
// ordered slice — with each recorded call carrying the argument paths the
// real sequence uses, and the delegated steps really placing the
// generation on main.
func TestFsops_saveOperationOrder(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	dir := filepath.Dir(dumpPath)
	store := New(dumpPath)
	rec := &recordingFSOps{}
	store.ops = rec
	err := store.Save([]byte(fsopsOldGen))
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	if got := rec.recordedOps(); !slices.Equal(got, fsopsOrder) {
		t.Errorf("recorded ops = %v, want exactly %v", got, fsopsOrder)
	}
	if !slices.Equal(rec.calls[0].args, []string{dir}) {
		t.Errorf("write args = %v, want the dump dir %s", rec.calls[0].args, dir)
	}
	if !slices.Equal(rec.calls[2].args, []string{dumpPath, dumpPath + prevSuffix}) {
		t.Errorf("rotate args = %v, want main %s and prev %s", rec.calls[2].args, dumpPath, dumpPath+prevSuffix)
	}
	if !slices.Equal(rec.calls[3].args, []string{dumpPath}) {
		t.Errorf("rename args = %v, want main %s", rec.calls[3].args, dumpPath)
	}
	if !slices.Equal(rec.calls[4].args, []string{dir}) {
		t.Errorf("syncDir args = %v, want the dump dir %s", rec.calls[4].args, dir)
	}
	fsopsGeneration(t, dumpPath, fsopsOldGen)
}

// TestFsops_saveOperationOrder_rotation_chain is the two-consecutive-Saves
// probe (TEST-03 adjacency, plan assumption): the first save's main
// becomes .prev and the second Save rotates again, its own generation
// replacing main — each Save recording the same exact five-step sequence,
// the rotation chain asserted over the real files the delegating fake
// produced.
func TestFsops_saveOperationOrder_rotation_chain(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	store := New(dumpPath)
	rec := &recordingFSOps{}
	store.ops = rec
	err := store.Save([]byte(fsopsOldGen))
	if err != nil {
		t.Fatalf("first Save() error = %v, want nil", err)
	}
	err = store.Save([]byte(fsopsNewGen))
	if err != nil {
		t.Fatalf("second Save() error = %v, want nil", err)
	}
	got := rec.recordedOps()
	if len(got) != 2*len(fsopsOrder) {
		t.Fatalf("recorded ops = %v, want two saves' worth of the five-step sequence", got)
	}
	if !slices.Equal(got[:len(fsopsOrder)], fsopsOrder) || !slices.Equal(got[len(fsopsOrder):], fsopsOrder) {
		t.Errorf("recorded ops = %v, want each Save to produce exactly %v", got, fsopsOrder)
	}
	secondRotate := rec.calls[len(fsopsOrder)+2]
	if !slices.Equal(secondRotate.args, []string{dumpPath, dumpPath + prevSuffix}) {
		t.Errorf("second rotate args = %v, want main %s and prev %s", secondRotate.args, dumpPath, dumpPath+prevSuffix)
	}
	fsopsGeneration(t, dumpPath+prevSuffix, fsopsOldGen) // the first main became .prev
	fsopsGeneration(t, dumpPath, fsopsNewGen)            // the second save replaced main
}

// TestFsops_failureAtEachStep injects a failure at each of the five seam
// steps (TEST-03's failure-at-each-step acceptance): Save surfaces the
// injected error, no temp ever litters the dump directory, and the main
// path never holds a partial generation — each row's residue is exactly
// the crash-window state the fallback chain absorbs — and a subsequent
// successful Save recovers.
func TestFsops_failureAtEachStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		failAt    fsopStep
		mainAfter string // "" expects the main generation absent
		prevAfter string // "" expects no .prev generation
	}{
		{
			name:      "write failure leaves the seeded generation alone",
			failAt:    fsopWrite,
			mainAfter: fsopsOldGen,
			prevAfter: "",
		},
		{
			name:      "fsync failure cleans the temp, main untouched",
			failAt:    fsopFsync,
			mainAfter: fsopsOldGen,
			prevAfter: "",
		},
		{
			name:      "rotate failure leaves main in place",
			failAt:    fsopRotate,
			mainAfter: fsopsOldGen,
			prevAfter: "",
		},
		{
			name:      "rename failure leaves the rotate-window residue",
			failAt:    fsopRename,
			mainAfter: "",
			prevAfter: fsopsOldGen,
		},
		{
			name:      "dir-fsync failure already placed the complete generation",
			failAt:    fsopSyncDir,
			mainAfter: fsopsNewGen,
			prevAfter: fsopsOldGen,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			dumpPath := filepath.Join(dir, "dump.json")
			store := New(dumpPath)
			err := store.Save([]byte(fsopsOldGen))
			if err != nil {
				t.Fatalf("seed Save() error = %v, want nil", err)
			}
			ops := &failingFSOps{failAt: tt.failAt, err: errFsopsInjected}
			store.ops = ops
			err = store.Save([]byte(fsopsNewGen))
			if !errors.Is(err, errFsopsInjected) {
				t.Fatalf("Save() error = %v, want exactly the injected error", err)
			}
			var wantEntries []string
			if tt.mainAfter != "" {
				wantEntries = append(wantEntries, "dump.json")
			}
			if tt.prevAfter != "" {
				wantEntries = append(wantEntries, prevName)
			}
			slices.Sort(wantEntries)
			if got := fsopsEntries(t, dir); !slices.Equal(got, wantEntries) {
				t.Errorf("directory after the failed save holds %v, want %v (no temp litter)", got, wantEntries)
			}
			fsopsGeneration(t, dumpPath, tt.mainAfter)
			fsopsGeneration(t, dumpPath+prevSuffix, tt.prevAfter)
			// Recovery: a subsequent successful Save rotates whatever
			// residue stands and places its generation on main.
			ops.failAt = ""
			err = store.Save([]byte(fsopsRecovered))
			if err != nil {
				t.Fatalf("recovery Save() error = %v, want nil", err)
			}
			fsopsGeneration(t, dumpPath, fsopsRecovered)
		})
	}
}

// TestFsops_emptyDumpRoundTrip is TEST-03's empty probe: a Save of an
// empty dump then Load returns zero records with no error — the empty
// payload is a legitimate generation, written through the same seam order
// as any other.
func TestFsops_emptyDumpRoundTrip(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	store := New(dumpPath)
	err := store.Save([]byte{})
	if err != nil {
		t.Fatalf("Save(empty) error = %v, want nil", err)
	}
	info, err := os.Stat(dumpPath)
	if err != nil {
		t.Fatalf("Stat(main) error = %v, want the empty generation on disk", err)
	}
	if info.Size() != 0 {
		t.Errorf("main size = %d, want 0 (zero records)", info.Size())
	}
	got, ok := store.Load(func([]byte) error { return nil })
	if !ok {
		t.Fatal("Load() ok = false, want the empty generation")
	}
	if len(got) != 0 {
		t.Errorf("Load() = %d bytes, want zero records (the empty dump)", len(got))
	}
}
