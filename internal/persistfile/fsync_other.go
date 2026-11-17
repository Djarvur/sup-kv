//go:build !darwin

package persistfile

import "os"

// syncDurable is the non-darwin durability flush: plain File.Sync,
// behavior byte-identical to the pre-helper call sites — the platform
// fsync already defines the durability contract here and nothing stronger
// is warranted; darwin's F_FULLFSYNC variant lives in fsync_darwin.go.
func syncDurable(f *os.File) error {
	return f.Sync()
}
