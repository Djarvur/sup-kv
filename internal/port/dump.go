package port

import "io"

// DumpStore is the whole-dump persistence seam the pure persistence core
// (internal/app/persist) saves through and the one file adapter
// (internal/persistfile) implements; tests may substitute in-memory fakes
// that record timestamps and inject failures. The seam carries whole dump
// bytes, never records — D-01/D-02: the adapter knows nothing about record
// structure, the engine knows nothing about files, and the signatures see
// only standard-library types (the sup-kv-port-stdlib-only depguard rule).
//
// Concurrency: Save and SaveStream serialize a single dumper goroutine's
// dumps — the engine's contract — while Load, CreateSnapshot and
// ReadSnapshot may be called independently at startup (Open) and on
// operator request.
//
//nolint:iface // port seams exist for other packages: the engine consumes and the adapter implements DumpStore, so no same-package usage is possible or wanted (D-01).
type DumpStore interface {
	// SaveStream durably writes the generation produced by fill as the new
	// main generation, rotating the old main generation to .prev (D-06).
	// fill runs exactly once on the caller's goroutine and writes the dump
	// payload incrementally; its error aborts the generation — the adapter
	// removes the temp file and surfaces the error, so no partial main is
	// possible. The fill must not retain w beyond the call. A non-nil error
	// means the generation did not become durable; the caller treats every
	// such error as a dump failure (D-05).
	SaveStream(fill func(io.Writer) error) error

	// Save durably writes dump as the new main generation, rotating the
	// old main generation to .prev. It is the byte-slice convenience for
	// fixtures; the engine streams through SaveStream. A non-nil error
	// means the generation did not become durable; the caller treats every
	// such error as a dump failure (D-05).
	Save(dump []byte) error

	// Load resolves the fallback chain main then .prev and returns the
	// first generation whose bytes pass the caller-supplied parse
	// oracle; a truncated or unparseable file counts as unreadable and
	// advances the chain (D-02). ok false is the empty start. parse is
	// supplied by the engine (codec.Parse) so the adapter stays
	// record-blind, and Load itself never reports an error: every
	// per-generation fault is chain business.
	Load(parse func([]byte) error) (dump []byte, ok bool)

	// CreateSnapshot byte-copies the current on-disk main dump under
	// name (D-10): the engine forces the current dump to complete first,
	// so the copy contains every confirmed write.
	CreateSnapshot(name string) error

	// ReadSnapshot returns the named snapshot's bytes for codec
	// decoding; a missing snapshot surfaces a wrapped fs error.
	ReadSnapshot(name string) ([]byte, error)
}
