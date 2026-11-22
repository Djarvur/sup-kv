package port

import (
	"io"
	"time"
)

// DumpStore is the whole-dump persistence seam the pure persistence core
// (internal/app/persist) saves through and the one file adapter
// (internal/persistfile) implements; tests may substitute in-memory fakes
// that record timestamps and inject failures. The seam carries whole dump
// bytes, never records — D-01/D-02: the adapter knows nothing about record
// structure, the engine knows nothing about files, and the signatures see
// only standard-library types (the sup-kv-port-stdlib-only depguard rule).
//
// Concurrency: SaveStream serializes a single dumper goroutine's dumps —
// the engine's contract — while Load, CreateSnapshot and ReadSnapshot may
// be called independently at startup (Open) and on operator request.
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

	// Load resolves the fallback chain main then .prev and returns the
	// first generation whose bytes pass the caller-supplied parse oracle,
	// reporting the outcome, the accepted generation and every candidate
	// fault through LoadResult (D-02). parse is supplied by the engine
	// (codec.Decode) so the adapter stays record-blind. Load itself never
	// reports an error in the Go error position: every per-generation fault
	// is a LoadResult failure. Load never deletes evidence — preserving an
	// unusable generation is the adapter's duty, and now is the clock the
	// adapter uses for the preserved name.
	Load(now time.Time, parse func([]byte) error) LoadResult

	// CreateSnapshot byte-copies the current on-disk main dump under
	// name (D-10): the engine forces the current dump to complete first,
	// so the copy contains every confirmed write.
	CreateSnapshot(name string) error

	// ReadSnapshot returns the named snapshot's bytes for codec
	// decoding; a missing snapshot surfaces a wrapped fs error.
	ReadSnapshot(name string) ([]byte, error)
}

// LoadOutcome classifies one startup load through the fallback chain (D-02):
// it is the explicit outcome the engine turns into exactly one WARN or ERROR
// (prompt is silent), never an inference from a nil byte slice.
type LoadOutcome uint8

// The four startup load outcomes.
const (
	// LoadPrompt is a fresh start: no generation file existed at all, so
	// there is no WARN and no ERROR to report.
	LoadPrompt LoadOutcome = iota
	// LoadMain means the main generation parsed and was accepted.
	LoadMain
	// LoadBackup means the .prev generation parsed and was accepted after
	// the main generation was absent or unusable; the caller reports exactly
	// one WARN naming the cause.
	LoadBackup
	// LoadUnusable means at least one candidate existed and none parsed; the
	// caller reports an ERROR and a non-zero metric, and the files are
	// preserved rather than deleted.
	LoadUnusable
)

// Generation names one candidate in the fallback chain.
type Generation uint8

// The two dump generations the fallback chain walks, in order.
const (
	// GenMain is the main dump generation.
	GenMain Generation = iota
	// GenPrev is the rotated previous generation (.prev).
	GenPrev
)

// GenerationFailure records one candidate generation the load could not
// accept: which generation, its path, the read or parse cause, and — once
// preservation lands — where the unusable bytes were preserved.
type GenerationFailure struct {
	Candidate   Generation
	Path        string
	Cause       error
	Preserved   string
	PreserveErr error
}

// LoadResult is the load's full report (D-02, PERSIST-11). Dump is the
// accepted generation's bytes (nil for LoadPrompt and LoadUnusable),
// Generation names which candidate was accepted, Failures carries one entry
// per rejected candidate in chain order, and Swept/SweepErr report the
// orphan-temp sweep (zero until the sweep lands).
type LoadResult struct {
	Dump       []byte
	Outcome    LoadOutcome
	Generation Generation
	Failures   []GenerationFailure
	Swept      int
	SweepErr   error
}
