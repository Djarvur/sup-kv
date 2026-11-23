package persist

// The nil-tap regression for the startup load guard (APIS-04, T-06-20): the
// storage.Config{} caller shape carries no OnChange callback — storage's
// Config.normalized substitutes a no-op for direct NewStore callers — and
// Open now wraps the caller's tap itself. The wrapper must preserve that
// totality: a nil tap must not panic during the load merge or on a later
// write, and the write must still install. The _internal_test.go suffix is
// the documented testpackage exemption.

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// nilTapNodeID is the fixture store's node identity.
const nilTapNodeID = uint16(0x0312)

// errNilTapSnapshot is the regression's never-called snapshot seam error.
var errNilTapSnapshot = errors.New("nilTapDumpStore: no snapshots")

// nilTapDumpStore is the startup-load fake for the nil-tap regression: Load
// feeds one encoded generation through the parse oracle, exactly as the file
// adapter does, so Open merges a real record set. The other seam methods are
// never called.
type nilTapDumpStore struct {
	dump []byte
}

// SaveStream reports success; the regression's write path never reaches it
// because the load fake only has to survive Open and one confirmed write.
func (*nilTapDumpStore) SaveStream(_ func(io.Writer) error) error { return nil }

// Load accepts the fixture when parse decodes it and reports the main
// outcome.
func (s *nilTapDumpStore) Load(_ time.Time, parse func([]byte) error) port.LoadResult {
	err := parse(s.dump)
	if err != nil {
		return port.LoadResult{
			Outcome:    port.LoadUnusable,
			Generation: port.GenMain,
			Failures:   []port.GenerationFailure{{Candidate: port.GenMain, Cause: err}},
		}
	}
	return port.LoadResult{Dump: s.dump, Outcome: port.LoadMain, Generation: port.GenMain}
}

// CreateSnapshot is never called by the regression.
func (*nilTapDumpStore) CreateSnapshot(_ string) error { return errNilTapSnapshot }

// ReadSnapshot is never called by the regression.
func (*nilTapDumpStore) ReadSnapshot(_ string) ([]byte, error) { return nil, errNilTapSnapshot }

// TestOpen_nilTapStaysWritable pins the nil-callback caller contract (the
// APIS-04 empty/null probe row): storage.Config{} carries a nil OnChange,
// which Open's wrapper must substitute with a no-op. Open over a loaded
// generation followed by a write through the engine must neither panic nor
// drop the record — the loaded one stays readable and the new one installs.
func TestOpen_nilTapStaysWritable(t *testing.T) {
	t.Parallel()
	nowMs := time.Now().UnixMilli()
	loadedVer := version.PackVersion(nowMs, 0, nilTapNodeID)
	loadedData := []byte("loaded")
	loaded := map[string]storage.Record{
		"loaded/one": {
			Version:  loadedVer,
			Data:     loadedData,
			Checksum: storage.RecordChecksum("loaded/one", loadedVer, loadedData, false),
		},
	}
	dump, err := Encode(loaded, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("Encode fixture: want nil, got error: %v", err)
	}
	eng, discarded := Open(
		nilTapNodeID, time.Now, time.Minute,
		&nilTapDumpStore{dump: dump},
		storage.Config{}, Config{},
	)
	if discarded != 0 {
		t.Errorf("Open() discarded = %d, want 0", discarded)
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		runErr := <-done
		if !errors.Is(runErr, context.Canceled) {
			t.Errorf("Run() error = %v, want context.Canceled", runErr)
		}
	})

	_, _, setErr := eng.Set("written/one", []byte("written"))
	if setErr != nil {
		t.Fatalf("Set after Open with a nil tap: want nil, got error: %v", setErr)
	}
	_, writeErr := eng.Get("written/one")
	if writeErr != nil {
		t.Fatalf("Get(written/one) after the write: want nil, got error: %v", writeErr)
	}
	_, loadedErr := eng.Get("loaded/one")
	if loadedErr != nil {
		t.Fatalf("Get(loaded/one) after the write: want the loaded record, got error: %v", loadedErr)
	}
}
