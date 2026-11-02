package port

// DumpStore is the whole-dump persistence seam the pure persistence core
// (internal/app/persist) saves through and the one file adapter
// (internal/persistfile) implements; tests may substitute in-memory fakes.
// The seam carries whole dump bytes, never records — D-01/D-02: the adapter
// knows nothing about record structure, the engine knows nothing about files.
//
//nolint:iface // port seams exist for other packages: the engine consumes and the adapter implements DumpStore, so no same-package usage is possible or wanted (D-01).
type DumpStore interface {
	// Save durably writes dump as the new main generation, rotating the old
	// main generation to .prev.
	Save(dump []byte) error

	// Load resolves the fallback chain main then .prev and returns the first
	// generation whose bytes pass the caller-supplied parse oracle; a
	// truncated or unparseable file counts as unreadable and advances the
	// chain (D-02). ok false is the empty start. parse is supplied by the
	// engine (codec.Parse) so the adapter stays record-blind.
	Load(parse func([]byte) error) (dump []byte, ok bool)

	// CreateSnapshot byte-copies the current on-disk dump under name (D-10).
	CreateSnapshot(name string) error

	// ReadSnapshot returns the named snapshot's bytes for codec decoding.
	ReadSnapshot(name string) ([]byte, error)
}
