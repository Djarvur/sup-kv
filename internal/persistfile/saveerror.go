package persistfile

import (
	"errors"
	"io/fs"
	"syscall"

	"github.com/Djarvur/sup-kv/internal/port"
)

// classedSaveError carries a port.FailureClass across the port and unwraps
// to the original save error, so [errors.Is] keeps reaching the cause.
type classedSaveError struct {
	err   error
	class port.FailureClass
}

// Error returns the underlying error's text.
func (e *classedSaveError) Error() string { return e.err.Error() }

// Unwrap exposes the original error to [errors.Is] and [errors.As].
func (e *classedSaveError) Unwrap() error { return e.err }

// FailureClass returns the classified class.
func (e *classedSaveError) FailureClass() port.FailureClass { return e.class }

// classifySaveError maps a save failure onto the port's FailureClass
// vocabulary (D-05): the errno knowledge stays in this adapter and the core
// consumes only port.FailureClassOf. ENOSPC is disk_full, EACCES and
// [fs.ErrPermission] are permission, and everything else is other; the
// returned wrapper unwraps to err (T-5-16).
func classifySaveError(err error) error {
	if err == nil {
		return nil
	}
	var class port.FailureClass
	switch {
	case errors.Is(err, syscall.ENOSPC):
		class = port.FailureDiskFull
	case errors.Is(err, syscall.EACCES), errors.Is(err, fs.ErrPermission):
		class = port.FailurePermission
	default:
		class = port.FailureOther
	}
	return &classedSaveError{err: err, class: class}
}
