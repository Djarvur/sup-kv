package grpcapi_test

import (
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
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
// while each seam answers the test's injected error. That reaches a failure
// class the real adapter produces — a missing snapshot (fs.ErrNotExist), a
// denied one (fs.ErrPermission), or a create with no main dump
// (persistfile.ErrNoDump) — without touching a filesystem.
type snapshotErrorDumpStore struct {
	*fakeDumpStore

	saveErr   error
	createErr error
	readErr   error
}

// SaveStream reports the injected save error, else delegates to the
// in-memory fake: the path-shape matrix needs the engine's dump failure to
// carry a real *fs.PathError so a leak would be observable.
func (s *snapshotErrorDumpStore) SaveStream(fill func(io.Writer) error) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	return s.fakeDumpStore.SaveStream(fill)
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

// pathShapePattern rejects an absolute-path shape in a status message: a
// leading slash, or a slash after whitespace, a quote, a parenthesis or '='.
// The mapper's constants and sentinels never match; error text that reached
// the wire would.
var pathShapePattern = regexp.MustCompile(`(^|[\s"'(=])/`)

// assertStatusMessagePathFree fails when one status message carries a server
// path shape: the injected path itself, any absolute-path shape, or a
// backslash separator (the D-03 predicate the RPC matrix applies to every
// error).
func assertStatusMessagePathFree(t *testing.T, what, message, secret string) {
	t.Helper()
	if strings.Contains(message, secret) {
		t.Errorf("%s: message %q contains the injected path %q", what, message, secret)
	}
	if pathShapePattern.MatchString(message) {
		t.Errorf("%s: message %q has an absolute-path shape", what, message)
	}
	if strings.Contains(message, `\`) {
		t.Errorf("%s: message %q contains a backslash separator", what, message)
	}
}

// TestRPC_statusMessagesPathFree is D-03's acceptance filter (A-03, T-06-11):
// it drives every data and snapshot RPC that can fail against a DumpStore
// whose save, create and read seams all surface a *fs.PathError carrying the
// test's own scratch path, in the fs.ErrNotExist and fs.ErrPermission
// variants, and rejects a path shape in every returned status message. The
// codes are pinned per row as well, so a path-free message with the wrong
// code cannot pass, and every error carries a non-Unknown code.
func TestRPC_statusMessagesPathFree(t *testing.T) {
	t.Parallel()

	variants := []struct {
		name     string
		errno    error
		loadCode codes.Code
	}{
		{name: "fs.ErrNotExist", errno: fs.ErrNotExist, loadCode: codes.NotFound},
		{name: "fs.ErrPermission", errno: fs.ErrPermission, loadCode: codes.PermissionDenied},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			t.Parallel()

			secret := filepath.Join(t.TempDir(), "secret")
			pathErr := &fs.PathError{Op: "open", Path: secret, Err: variant.errno}
			store := &snapshotErrorDumpStore{
				fakeDumpStore: &fakeDumpStore{},
				saveErr:       pathErr,
				createErr:     pathErr,
				readErr:       pathErr,
			}
			h := buildTestServer(t, testServerConfig{dumps: store, gate: openGateOptions()})
			awaitGateOpen(t, h)

			// The predicate's teeth: path-bearing text is rejected and the
			// mapper's constant passes, so the rows below cannot pass
			// vacuously.
			if !pathShapePattern.MatchString("internal error: open " + secret + ": no such file") {
				t.Fatalf("path-shape predicate %q does not reject a path-bearing text", pathShapePattern)
			}
			if pathShapePattern.MatchString(wantSnapshotNotFoundMessage) {
				t.Fatalf("path-shape predicate %q rejects the constant %q", pathShapePattern, wantSnapshotNotFoundMessage)
			}

			rows := []struct {
				name   string
				want   codes.Code
				invoke func(t *testing.T, h *testHarness) error
			}{
				{
					name: "Get of a missing key",
					want: codes.NotFound,
					invoke: func(t *testing.T, h *testHarness) error {
						t.Helper()
						_, err := h.client.Get(t.Context(), &apiproto.GetRequest{Key: "matrix/missing"})
						return err
					},
				},
				{
					name: "Set with a failing dump",
					want: codes.Unavailable,
					invoke: func(t *testing.T, h *testHarness) error {
						t.Helper()
						_, err := h.client.Set(t.Context(), &apiproto.SetRequest{Key: "matrix/set", Value: []byte("v")})
						return err
					},
				},
				{
					name: "Delete with a failing dump",
					want: codes.Unavailable,
					invoke: func(t *testing.T, h *testHarness) error {
						t.Helper()
						// The failed dump leaves the record installed in
						// memory (PERS-04), so the delete has a key to hit.
						_, _, _ = h.engine.Set("matrix/delete", []byte("v"))
						_, err := h.client.Delete(t.Context(), &apiproto.DeleteRequest{Key: "matrix/delete"})
						return err
					},
				},
				{
					name: "List with an empty prefix",
					want: codes.OK,
					invoke: func(t *testing.T, h *testHarness) error {
						t.Helper()
						_, err := h.client.List(t.Context(), &apiproto.ListRequest{Prefix: ""})
						return err
					},
				},
				{
					name: "SnapshotCreate",
					want: codes.Unavailable,
					invoke: func(t *testing.T, h *testHarness) error {
						t.Helper()
						_, err := h.client.SnapshotCreate(t.Context(), &apiproto.SnapshotCreateRequest{Name: "matrix"})
						return err
					},
				},
				{
					name: "SnapshotLoad",
					want: variant.loadCode,
					invoke: func(t *testing.T, h *testHarness) error {
						t.Helper()
						_, err := h.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "matrix"})
						return err
					},
				},
			}
			for _, row := range rows {
				err := row.invoke(t, h)
				if got := status.Code(err); got != row.want {
					t.Errorf("%s: code = %v, want %v (err=%v)", row.name, got, row.want, err)
				}
				if err == nil {
					continue
				}
				assertStatusMessagePathFree(t, row.name, status.Convert(err).Message(), secret)
			}
		})
	}
}
