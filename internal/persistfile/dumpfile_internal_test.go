package persistfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Internal-test constants: the grace a snapshot gets to prove it parks on
// the store mutex instead of reading the rotate window, the mode of the
// simulated post-window main generation, and the strings of the two
// generations and the snapshot name the test moves around.
const (
	windowGrace        = 50 * time.Millisecond
	simulatedMode      = 0o600
	generationOne      = "generation-one"
	generationTwo      = "generation-two"
	windowSnapshotName = "during-window"
)

// TestFileStore_snapshotWaitsOutSaveWindow is WR-01's deterministic
// regression, white-box on purpose (the *_internal_test.go name is the
// configured testpackage exemption): it holds the store mutex across a
// simulated rotate window — main already rotated to .prev, the new
// generation not yet renamed into place, exactly the state between
// placeFile's two renames — while CreateSnapshot runs. The snapshot must
// park on the mutex instead of observing the absent main as ErrNoDump,
// then copy the generation the completed save leaves on main. On the
// pre-fix code the snapshot completes inside the window and fails with
// ErrNoDump, which the grace-period check catches.
func TestFileStore_snapshotWaitsOutSaveWindow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")
	f := New(path)
	err := f.Save([]byte(generationOne))
	if err != nil {
		t.Fatalf("seed Save() error = %v, want nil", err)
	}
	// Simulate the rotate window: the old main has rotated to .prev, the
	// new main is not placed yet, and the store mutex is held exactly as a
	// concurrent Save holds it across placeFile.
	err = os.Rename(path, path+prevSuffix)
	if err != nil {
		t.Fatalf("simulate the rotate: %v", err)
	}
	f.mu.Lock()
	snapErr := make(chan error, 1)
	go func() { snapErr <- f.CreateSnapshot(windowSnapshotName) }()
	select {
	case err := <-snapErr:
		t.Fatalf("CreateSnapshot() completed inside the save window: %v, want it parked on the store mutex", err)
	case <-time.After(windowGrace):
	}
	// The saving side completes the window: the new generation lands on
	// main and the mutex releases.
	err = os.WriteFile(path, []byte(generationTwo), simulatedMode)
	if err != nil {
		t.Fatalf("complete the window: %v", err)
	}
	f.mu.Unlock()
	err = <-snapErr
	if err != nil {
		t.Fatalf("CreateSnapshot() after the window error = %v, want nil", err)
	}
	copied, err := os.ReadFile(filepath.Join(path+snapshotDirSuffix, windowSnapshotName))
	if err != nil {
		t.Fatalf("read the snapshot: %v", err)
	}
	if string(copied) != generationTwo {
		t.Errorf("snapshot = %q, want the generation the completed save left on main", copied)
	}
}
