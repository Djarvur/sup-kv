package persistfile //nolint:testpackage // classifySaveError is unexported: the errno-to-class table is white-box by design, and the fake fsops injection point is unexported too.

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/Djarvur/sup-kv/internal/port"
)

// TestClassifySaveError pins the errno-to-FailureClass table (D-05): ENOSPC
// is disk_full, EACCES/fs.ErrPermission is permission, anything else is
// other; every wrapper keeps errors.Is transparent and exposes the class
// through port.FailureClassOf.
func TestClassifySaveError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want port.FailureClass
	}{
		{
			name: "ENOSPC is disk_full",
			err:  &os.PathError{Op: "write", Path: "dump.json", Err: syscall.ENOSPC},
			want: port.FailureDiskFull,
		},
		{
			name: "EACCES is permission",
			err:  &os.PathError{Op: "open", Path: "dump.json", Err: syscall.EACCES},
			want: port.FailurePermission,
		},
		{
			name: "fs.ErrPermission is permission",
			err:  fs.ErrPermission,
			want: port.FailurePermission,
		},
		{
			name: "an unrelated errno is other",
			err:  &os.PathError{Op: "fsync", Path: "dump.json", Err: syscall.EIO},
			want: port.FailureOther,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifySaveError(tt.err)
			if got == nil {
				t.Fatal("classifySaveError() = nil, want a classified wrapper")
			}
			if class := port.FailureClassOf(got); class != tt.want {
				t.Errorf("FailureClassOf(classifySaveError(%v)) = %v, want %v", tt.err, class, tt.want)
			}
			if !errors.Is(got, tt.err) {
				t.Errorf("classifySaveError(%v) does not unwrap to the original error", tt.err)
			}
			if got.Error() != tt.err.Error() {
				t.Errorf("classifySaveError(%v).Error() = %q, want %q", tt.err, got.Error(), tt.err.Error())
			}
		})
	}
	got := classifySaveError(nil)
	if got != nil {
		t.Errorf("classifySaveError(nil) = %v, want nil", got)
	}
}

// TestFailureClassString pins the metric/log label vocabulary.
func TestFailureClassString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		class port.FailureClass
		want  string
	}{
		{port.FailureOther, "other"},
		{port.FailureDiskFull, "disk_full"},
		{port.FailurePermission, "permission"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.class.String(); got != tt.want {
				t.Errorf("FailureClass(%d).String() = %q, want %q", tt.class, got, tt.want)
			}
		})
	}
}
