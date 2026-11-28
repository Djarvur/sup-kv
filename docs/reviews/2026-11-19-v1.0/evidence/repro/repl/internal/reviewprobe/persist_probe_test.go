package reviewprobe

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

func openEngine(t *testing.T, path string) (*persist.Engine, int, func()) {
	t.Helper()
	eng, discarded := persist.Open(1, time.Now, time.Minute, persistfile.New(path), storage.Config{}, persist.Config{Debounce: time.Millisecond, MaxWait: 10 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _ = eng.Run(ctx) }()
	return eng, discarded, func() { cancel(); <-done }
}

// Cross-boundary (API SnapshotCreate -> persist.Snapshot -> persistfile):
// after a start that fell back to .prev because the main dump is corrupt,
// SnapshotCreate returns OK and the snapshot is a byte copy of the CORRUPT
// main file (the clean engine skips the forced dump).
func TestProbe_snapshotAfterPrevFallbackCopiesCorruptMain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dump.json")
	eng, _, stop := openEngine(t, path)
	if _, _, err := eng.Set("k", []byte("v1")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := eng.Set("k", []byte("v2")); err != nil { // second dump => .prev exists
		t.Fatal(err)
	}
	stop()
	if err := os.WriteFile(path, []byte("{ this is not a dump"), 0o600); err != nil { // main torn/corrupt
		t.Fatal(err)
	}
	eng2, _, stop2 := openEngine(t, path)
	defer stop2()
	rec, err := eng2.Get("k")
	t.Logf("restart with corrupt main: Get(k)=%q err=%v (served from .prev)", rec.Data, err)
	if err := eng2.Snapshot("backup"); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	snap, _ := os.ReadFile(path + ".snapshots/backup")
	t.Logf("SnapshotCreate returned OK; snapshot bytes = %q", snap)
	if err := eng2.Restore("backup"); err != nil {
		t.Errorf("the snapshot just created with an OK status cannot be loaded: %v", err)
	}
}

// Cross-boundary (startup): a start from an unreadable dump is silent — Open
// reports nothing the binary could log.
func TestProbe_emptyStartIsSilent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dump.json")
	if err := os.WriteFile(path, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	eng, discarded, stop := openEngine(t, path)
	defer stop()
	t.Logf("both generations unusable: records=%d discarded=%d — Open returns no signal that the fallback chain was exhausted", len(eng.Records()), discarded)
}
