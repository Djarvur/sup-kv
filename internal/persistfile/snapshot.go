package persistfile

import "errors"

// Sentinel errors of the snapshot adapter.
var (
	// ErrNoDump is returned by CreateSnapshot when no main dump exists.
	ErrNoDump = errors.New("no dump on disk to snapshot")

	// ErrInvalidSnapshotName is returned for names outside the validated
	// charset (Security V5).
	ErrInvalidSnapshotName = errors.New("invalid snapshot name")
)

// CreateSnapshot byte-copies the current dump under name (stub).
func (f *FileStore) CreateSnapshot(_ string) error { return nil }

// ReadSnapshot returns the named snapshot's bytes (stub).
func (f *FileStore) ReadSnapshot(_ string) ([]byte, error) { return nil, nil }
