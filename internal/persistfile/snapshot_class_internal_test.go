package persistfile

// Internal test for the snapshot create path's failure classification (D-03,
// APIS-03, T-06-13). It lives in the package because the classification is
// asserted through the unexported fsops injection point (the failingFSOps
// fake from fsops_test.go) — testpackage's skip-regexp sanctions the
// *_internal_test.go placement.

import (
	"errors"
	"io/fs"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Djarvur/sup-kv/internal/port"
)

// savedStore returns a FileStore with one complete main dump on disk: the
// precondition every CreateSnapshot classification row needs, built with the
// store's real ops so the dump is a genuine generation.
func savedStore(t *testing.T) *FileStore {
	t.Helper()
	store := New(filepath.Join(t.TempDir(), "dump.json"))
	err := store.Save([]byte("GENERATION-ONE"))
	if err != nil {
		t.Fatalf("seed Save() error = %v, want nil", err)
	}
	return store
}

// TestCreateSnapshot_failureClasses proves D-03's adapter half (T-06-13):
// every filesystem failure the snapshot copy can hit crosses the port as a
// port.FailureClass — ENOSPC is FailureDiskFull and fs.ErrPermission is
// FailurePermission — while the adapter's own ErrNoDump sentinel stays
// unclassified, so the API mapper can tell "no dump configured" from a
// filesystem fault and no path-bearing errno text reaches it. Each row drives
// a failing fsops write, the single point all three CreateSnapshot steps
// funnel through (saveAtomically).
func TestCreateSnapshot_failureClasses(t *testing.T) {
	t.Parallel()

	t.Run("ENOSPC is FailureDiskFull", func(t *testing.T) {
		t.Parallel()
		store := savedStore(t)
		store.ops = &failingFSOps{failAt: fsopWrite, err: syscall.ENOSPC}
		err := store.CreateSnapshot("snap")
		if !errors.Is(err, syscall.ENOSPC) {
			t.Fatalf("CreateSnapshot() error = %v, want the injected ENOSPC reachable through errors.Is", err)
		}
		if got := port.FailureClassOf(err); got != port.FailureDiskFull {
			t.Errorf("FailureClassOf(ENOSPC) = %v, want %v", got, port.FailureDiskFull)
		}
	})

	t.Run("fs.ErrPermission is FailurePermission", func(t *testing.T) {
		t.Parallel()
		store := savedStore(t)
		store.ops = &failingFSOps{failAt: fsopWrite, err: fs.ErrPermission}
		err := store.CreateSnapshot("snap")
		if !errors.Is(err, fs.ErrPermission) {
			t.Fatalf("CreateSnapshot() error = %v, want the injected permission error reachable through errors.Is", err)
		}
		if got := port.FailureClassOf(err); got != port.FailurePermission {
			t.Errorf("FailureClassOf(fs.ErrPermission) = %v, want %v", got, port.FailurePermission)
		}
	})

	t.Run("no main dump stays ErrNoDump and unclassified", func(t *testing.T) {
		t.Parallel()
		store := New(filepath.Join(t.TempDir(), "dump.json"))
		err := store.CreateSnapshot("snap")
		if !errors.Is(err, ErrNoDump) {
			t.Fatalf("CreateSnapshot() error = %v, want ErrNoDump", err)
		}
		if classed, ok := errors.AsType[port.ClassedError](err); ok {
			t.Errorf("ErrNoDump carries the class %v, want the unclassified sentinel", classed.FailureClass())
		}
		if got := port.FailureClassOf(err); got != port.FailureOther {
			t.Errorf("FailureClassOf(ErrNoDump) = %v, want %v", got, port.FailureOther)
		}
	})
}
