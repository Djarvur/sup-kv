// Package persistfile is the sup-kv dump file adapter: the only os-importing
// persistence code, implementing port.DumpStore with atomic writes, .prev
// rotation and the fallback chain (D-01/D-02).
package persistfile

import "github.com/Djarvur/sup-kv/internal/port"

// FileStore persists whole dump bytes at one file path (stub).
type FileStore struct {
	path string
}

// FileStore implements the whole-dump persistence seam.
var _ port.DumpStore = (*FileStore)(nil)

// New returns a FileStore persisting to dumpPath; the .prev and .snapshots
// siblings derive from it (D-08).
func New(dumpPath string) *FileStore {
	return &FileStore{path: dumpPath}
}

// Save durably writes dump as the new main generation (stub).
func (f *FileStore) Save(_ []byte) error { return nil }

// Load resolves the fallback chain main then .prev (stub).
func (f *FileStore) Load(_ func([]byte) error) ([]byte, bool) { return nil, false }
