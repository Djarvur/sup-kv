package grpcapi_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// The pinned snapshot status messages (D-03), asserted by literal: the
// mapper's own constants are unexported, and the wire text is the contract,
// so a production rename must fail these tests rather than move silently.
const (
	wantSnapshotNotFoundMessage = "snapshot not found"
	wantNoDumpMessage           = "no dump on disk to snapshot"
)

// snapshotErrorDumpStore is the DumpStore fake the snapshot error-mapping
// tests drive: the embedded in-memory fake serves the ordinary save path,
// while each snapshot seam answers the test's injected error. That reaches a
// failure class the real adapter produces — a missing snapshot
// (fs.ErrNotExist), a denied one (fs.ErrPermission), or a create with no main
// dump (persistfile.ErrNoDump) — without touching a filesystem.
type snapshotErrorDumpStore struct {
	*fakeDumpStore

	createErr error
	readErr   error
}

// CreateSnapshot reports the injected create error.
func (s *snapshotErrorDumpStore) CreateSnapshot(string) error { return s.createErr }

// ReadSnapshot reports the injected read error.
func (s *snapshotErrorDumpStore) ReadSnapshot(string) ([]byte, error) { return nil, s.readErr }

// TestMapError_missingSnapshotIsNotFound proves D-03's snapshot-read row: a
// missing snapshot is NotFound with the constant message, never Internal and
// never the adapter's path-bearing fs error (the A-03 leak — the configured
// dump path must not cross the wire). The engine's reverse adapter surfaces
// ReadSnapshot's error unchanged, so this drives the real mapping path.
func TestMapError_missingSnapshotIsNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	secret := filepath.Join(dir, "secret")
	h := buildTestServer(t, testServerConfig{
		dumps: &snapshotErrorDumpStore{
			fakeDumpStore: &fakeDumpStore{},
			readErr:       &fs.PathError{Op: "open", Path: secret, Err: fs.ErrNotExist},
		},
		gate: openGateOptions(),
	})
	awaitGateOpen(t, h)

	_, err := h.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "missing"})
	assertCode(t, "SnapshotLoad of a missing snapshot", err, codes.NotFound)
	msg := status.Convert(err).Message()
	if msg != wantSnapshotNotFoundMessage {
		t.Fatalf("SnapshotLoad message = %q, want the constant %q", msg, wantSnapshotNotFoundMessage)
	}
	if strings.Contains(msg, dir) {
		t.Fatalf("SnapshotLoad message %q leaks the configured path %q", msg, dir)
	}
}

// TestMapError_noDumpIsFailedPrecondition proves D-03's no-dump row: a
// snapshot create with no dump on disk is FailedPrecondition with the
// adapter's sentinel text — the pinned reading of "no dump configured"
// (--dump-path is mandatory, so the adapter's absent-main sentinel is the
// reachable shape) — never Internal.
func TestMapError_noDumpIsFailedPrecondition(t *testing.T) {
	t.Parallel()

	h := buildTestServer(t, testServerConfig{
		dumps: &snapshotErrorDumpStore{
			fakeDumpStore: &fakeDumpStore{},
			createErr:     persistfile.ErrNoDump,
		},
		gate: openGateOptions(),
	})
	awaitGateOpen(t, h)

	_, err := h.client.SnapshotCreate(t.Context(), &apiproto.SnapshotCreateRequest{Name: "snap"})
	assertCode(t, "SnapshotCreate without a main dump", err, codes.FailedPrecondition)
	if msg := status.Convert(err).Message(); msg != wantNoDumpMessage {
		t.Fatalf("SnapshotCreate message = %q, want the sentinel text %q", msg, wantNoDumpMessage)
	}
}

// TestMapError_missingSnapshotRealAdapter proves the same row through the
// real persistfile adapter (newFileServer): the raw os.ReadFile PathError
// that carries the configured dump directory becomes NotFound with the
// constant message, so no server path can cross the wire — the concrete A-03
// leak this plan closes.
func TestMapError_missingSnapshotRealAdapter(t *testing.T) {
	t.Parallel()

	h := newFileServer(t)
	_, err := h.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "missing"})
	assertCode(t, "SnapshotLoad of a missing snapshot through the real adapter", err, codes.NotFound)
	msg := status.Convert(err).Message()
	if msg != wantSnapshotNotFoundMessage {
		t.Fatalf("SnapshotLoad message = %q, want the constant %q", msg, wantSnapshotNotFoundMessage)
	}
	if strings.Contains(msg, "/") || strings.Contains(msg, `\`) {
		t.Fatalf("SnapshotLoad message %q carries a filesystem path shape", msg)
	}
}
