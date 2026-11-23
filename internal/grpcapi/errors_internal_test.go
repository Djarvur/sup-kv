package grpcapi

// Internal tests for the error mapper's totality (D-03, APIS-03). They live
// in the package because mapError and its message constants are unexported,
// and the external test file drives the RPC surface instead;
// testpackage's skip-regexp sanctions the placement (the repo's
// *_internal_test.go convention). The table names every switch case plus the
// fallback, so removing a case fails its row; the log test is the only
// non-parallel test here because it swaps the process-global slog handler.

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
)

// errMapperFault is the static cause the classed rows wrap: err113 forbids
// dynamic errors, and the class — not the text — is what the mapper reads.
var errMapperFault = errors.New("mapper table fault")

// classedError carries a port.FailureClass the way the persistfile adapter
// does: the mapper must consume only the port interface, never errno detail
// (T-06-13). It is a test-local copy so the adapter's concrete type stays
// unexported.
type classedError struct {
	err   error
	class port.FailureClass
}

// Error returns the underlying error's text.
func (e *classedError) Error() string { return e.err.Error() }

// Unwrap exposes the original error to [errors.Is] and [errors.As].
func (e *classedError) Unwrap() error { return e.err }

// FailureClass returns the carried class.
func (e *classedError) FailureClass() port.FailureClass { return e.class }

// unmappedError is the fallback rows' error: no sentinel, no class, no
// GRPCStatus — the shape an unforeseen failure takes when it reaches a
// handler.
type unmappedError struct{ detail string }

// Error returns the carried detail, the text the mapper must log and never
// return.
func (e unmappedError) Error() string { return e.detail }

// TestMapError_totalTable proves the mapper is total (D-03): every sentinel
// and failure class the engine and the adapters produce has a row, each row
// pins both the gRPC code and the exact client-visible message, and the
// fallback answers the constant. Each row names one switch case — remove the
// case and the row falls through to a different answer and fails — plus the
// pre-built status pass-through and the unknown-error fallback. The messages
// are literals on purpose: the wire text is the API contract, so a renamed
// sentinel must fail here rather than move silently.
func TestMapError_totalTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		want    codes.Code
		message string
	}{
		{
			name:    "storage.ErrNotFound",
			err:     storage.ErrNotFound,
			want:    codes.NotFound,
			message: "key not found",
		},
		{
			name:    "storage.ErrAlreadyExists",
			err:     storage.ErrAlreadyExists,
			want:    codes.AlreadyExists,
			message: "key already exists",
		},
		{
			name:    "storage.ErrValueTooLarge",
			err:     storage.ErrValueTooLarge,
			want:    codes.InvalidArgument,
			message: "value exceeds the maximum allowed size",
		},
		{
			name:    "persist.ErrDumpFailed",
			err:     persist.ErrDumpFailed,
			want:    codes.Unavailable,
			message: "dump failed; write not confirmed",
		},
		{
			name:    "port.FailureDiskFull",
			err:     &classedError{err: errMapperFault, class: port.FailureDiskFull},
			want:    codes.ResourceExhausted,
			message: diskFullMessage,
		},
		{
			name:    "port.FailurePermission",
			err:     &classedError{err: errMapperFault, class: port.FailurePermission},
			want:    codes.PermissionDenied,
			message: permissionDeniedMessage,
		},
		{
			name:    "persist.ErrUnconfirmed",
			err:     persist.ErrUnconfirmed,
			want:    codes.Unavailable,
			message: "write not confirmed before the confirmation timeout",
		},
		{
			name:    "persist.ErrClosed",
			err:     persist.ErrClosed,
			want:    codes.Unavailable,
			message: "persist engine closed",
		},
		{
			name:    "persist.ErrSnapshotIncomplete",
			err:     persist.ErrSnapshotIncomplete,
			want:    codes.DataLoss,
			message: "snapshot does not decode cleanly",
		},
		{
			name:    "persistfile.ErrInvalidSnapshotName",
			err:     persistfile.ErrInvalidSnapshotName,
			want:    codes.InvalidArgument,
			message: "invalid snapshot name",
		},
		{
			name:    "persistfile.ErrNoDump",
			err:     persistfile.ErrNoDump,
			want:    codes.FailedPrecondition,
			message: "no dump on disk to snapshot",
		},
		{
			name: "fs.ErrNotExist via a path error",
			err: &fs.PathError{
				Op:   "open",
				Path: "/var/lib/sup-kv/dump.json.snapshots/missing",
				Err:  fs.ErrNotExist,
			},
			want:    codes.NotFound,
			message: snapshotNotFoundMessage,
		},
		{
			name: "fs.ErrPermission via a path error",
			err: &fs.PathError{
				Op:   "open",
				Path: "/var/lib/sup-kv/dump.json.snapshots/denied",
				Err:  fs.ErrPermission,
			},
			want:    codes.PermissionDenied,
			message: permissionDeniedMessage,
		},
		{
			name:    "context.Canceled",
			err:     context.Canceled,
			want:    codes.Canceled,
			message: "context canceled",
		},
		{
			name:    "context.DeadlineExceeded",
			err:     context.DeadlineExceeded,
			want:    codes.DeadlineExceeded,
			message: "context deadline exceeded",
		},
		{
			name:    "a pre-built status error passes through",
			err:     status.Error(codes.Aborted, "aborted by the test"),
			want:    codes.Aborted,
			message: "aborted by the test",
		},
		{
			name:    "an unknown error is the constant fallback",
			err:     unmappedError{detail: "unmapped failure"},
			want:    codes.Internal,
			message: internalErrorMessage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapped := mapError(tt.err)
			if got := status.Code(mapped); got != tt.want {
				t.Fatalf("mapError(%v) code = %v, want %v", tt.err, got, tt.want)
			}
			if got := status.Convert(mapped).Message(); got != tt.message {
				t.Fatalf("mapError(%v) message = %q, want exactly %q", tt.err, got, tt.message)
			}
		})
	}
}

// TestMapError_unmappedIsConstantAndLogged proves the fallback's other half
// (T-06-11): the detail is logged exactly once with the static message and
// the snake_case attribute, while the client-visible message is the constant
// and carries no part of the detail. The unknown error's text embeds a path
// so a leak cannot pass by coincidence.
//
//nolint:paralleltest // the test swaps the process-global slog handler, so it must run sequentially
func TestMapError_unmappedIsConstantAndLogged(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	secret := filepath.Join(t.TempDir(), "secret")
	err := mapError(unmappedError{detail: "open " + secret + ": unmapped fault"})

	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("fallback code = %v, want %v", got, codes.Internal)
	}
	message := status.Convert(err).Message()
	if message != internalErrorMessage {
		t.Fatalf("fallback message = %q, want exactly %q", message, internalErrorMessage)
	}
	if strings.Contains(message, secret) {
		t.Fatalf("fallback message %q leaks the detail path %q", message, secret)
	}
	logged := buf.String()
	if !strings.Contains(logged, "Unmapped RPC error") {
		t.Fatalf("fallback log = %q, want the static message %q", logged, "Unmapped RPC error")
	}
	if !strings.Contains(logged, "detail=") {
		t.Fatalf("fallback log = %q, want the snake_case detail attribute", logged)
	}
	if !strings.Contains(logged, secret) {
		t.Fatalf("fallback log = %q, want the detail path %q recorded", logged, secret)
	}
}
