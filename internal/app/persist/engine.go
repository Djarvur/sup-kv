package persist

import (
	"fmt"
	"slices"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Design defaults a zero or negative Config field is normalized to (D-04),
// and the confirmation deadline itself (PERSIST-08, D-07).
const (
	defaultDebounce       = 50 * time.Millisecond
	defaultMaxWait        = time.Second
	defaultConfirmTimeout = 10 * time.Second
)

// Config holds the dump timing parameters; it is immutable once handed to
// New. Debounce is the quiet window a burst of writes may keep extending,
// MaxWait the bound on that extension measured from the oldest unconfirmed
// write: a dump starts no later than oldest+MaxWait under any write cadence
// (D-04, PERS-03).
type Config struct {
	Debounce time.Duration
	MaxWait  time.Duration

	// ConfirmTimeout bounds every confirmation wait (PERSIST-08, D-07): a
	// write whose covering dump has not completed by then resolves with
	// ErrUnconfirmed, and a flush — Flush, Snapshot, Restore — is bounded the
	// same way. A zero or negative value defaults to 10 s in normalized.
	ConfirmTimeout time.Duration

	// Now is the engine's clock and After its timer seam (PERSIST-12): every
	// read and every armed timer goes through them so the simulator can drive
	// the dump loop on its own virtual time. A nil pair defaults to
	// time.Now/time.After in normalized (the only wall-clock references the
	// package keeps).
	Now   func() time.Time
	After func(time.Duration) <-chan time.Time

	// OnDumpFailure receives every failed dump attempt exactly once with the
	// underlying error and the adapter's failure class (D-05, PERSIST-05):
	// the binary wires it to one ERROR log line and must not block. Nil is
	// valid — the counters and the retry ladder still apply.
	OnDumpFailure func(err error, class port.FailureClass)
}

// normalized returns cfg with zero or negative fields replaced by the
// design defaults, and the two clock seams filled with the wall-clock
// functions (PERSIST-12). These two assignments are the package's only
// wall-clock references: every other site reads through cfg.Now or arms
// through cfg.After.
func (c Config) normalized() Config {
	if c.Debounce <= 0 {
		c.Debounce = defaultDebounce
	}
	if c.MaxWait <= 0 {
		c.MaxWait = defaultMaxWait
	}
	if c.ConfirmTimeout <= 0 {
		c.ConfirmTimeout = defaultConfirmTimeout
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.After == nil {
		c.After = time.After
	}
	return c
}

// Engine is the write-confirmation engine over one storage.Store: a local
// write registers a waiter and blocks until a dump whose snapshot contains
// that record completed (nil — the write is on disk, PERS-04), failed
// (ErrDumpFailed — the record stays readable in memory, D-05), or did not
// complete within ConfirmTimeout (ErrUnconfirmed — the record is installed
// but not yet durable, PERSIST-08). Engine is safe for concurrent use by
// multiple goroutines; the confirmation mutex is held only to swap waiter
// lists and the dirty epoch, never across codec or port I/O, and exactly one
// dumper goroutine — the caller-launched Run — serializes every Save.
//
// Engine is the composition root of the five internal roles (D-10,
// PERSIST-11): the embedded confirmer owns the waiters and the dirty epoch,
// *scheduler owns the dump loop, the retry ladder and the dump counters, and
// the loader, facade and snapshot method groups stay methods on Engine so
// every exported signature is unchanged. The confirmer is embedded rather
// than held in a named field because the pre-split white-box tests read
// Engine.mu and Engine.waiters and call Engine.markDirty directly — the move
// is internal-only and those tests are its API assertion.
type Engine struct {
	*confirmer

	store *storage.Store
	dumps port.DumpStore
	cfg   Config
	sched *scheduler

	// valueMaxBytes is the store's effective value limit, captured once at
	// construction: the dump encoder and the snapshot decoder both pass it, so
	// the two sides cannot disagree (PERSIST-01).
	valueMaxBytes int

	// loadStats is the startup load report Open captures over the port's
	// LoadResult (D-02, PERSIST-02/PERSIST-11): the outcome, the accepted
	// generation, the per-candidate failures, the accepted generation's decode
	// census and the sweep result. It is written once by Open before the
	// engine is returned and read-only afterwards, so LoadStats needs no lock.
	// An engine built by New leaves it zero (Loaded false).
	loadStats LoadStats
}

// DumpStats is the dump-cycle gauge OBS-01 reads: the last dump's
// Encode+Save span in seconds — recorded on success and failure alike —
// the total dump attempts, and the failed ones, split by the adapter's
// failure class so a persistent ENOSPC and a persistent EACCES are
// distinguishable without parsing the log (PERSIST-05). Errors stays the
// total failed attempts; DiskFull and Permission are subsets of it.
// Unconfirmed counts writes answered with ErrUnconfirmed before their
// covering dump completed (PERSIST-08) — a durability answer independent of
// the failure counters.
type DumpStats struct {
	DurationSeconds float64
	Total           uint64
	Errors          uint64
	DiskFull        uint64
	Permission      uint64
	Unconfirmed     uint64
}

// LoadStats is the startup load report Open stores on the engine (D-02,
// PERSIST-02/PERSIST-11): the port's LoadResult mapped into the engine's own
// report type — the port remains the seam, this is the engine's view of it.
// Loaded distinguishes a real startup load from an engine built by New
// (whose report stays zero); Outcome is Prompt/Main/Backup/Unusable,
// Generation names the accepted candidate, Failures carries one entry per
// rejected candidate, Decoded is the accepted generation's per-class discard
// census (PERSIST-10), and Swept/SweepErr report the startup orphan sweep.
// The binary turns this one report into exactly one WARN or ERROR per class
// and the two load-metric families.
type LoadStats struct {
	Loaded     bool
	Outcome    port.LoadOutcome
	Generation port.Generation
	Failures   []port.GenerationFailure
	Decoded    DecodeStats
	Swept      int
	SweepErr   error
}

// New returns an engine persisting store's state through dumps. store and
// dumps must be non-nil. The engine spawns no goroutine itself: the caller
// launches Run, whose lifecycle is the documented API contract. It
// constructs the one dumpStats holder and hands the same pointer to the
// scheduler and the confirmer, so the timeout counter and the dump counters
// are one published value (PERSIST-11).
func New(store *storage.Store, dumps port.DumpStore, cfg Config) *Engine {
	cfg = cfg.normalized()
	stats := &dumpStats{}
	conf := newConfirmer(cfg, stats)
	return &Engine{
		confirmer:     conf,
		store:         store,
		dumps:         dumps,
		cfg:           cfg,
		sched:         newScheduler(store, dumps, cfg, conf, stats),
		valueMaxBytes: store.ValueMaxBytes(),
	}
}

// LoadStats returns the startup load report Open captured — a value copy,
// with the failure slice cloned so a caller can never mutate the engine's
// record. Engines built by New carry the zero value: Loaded false and the
// Prompt outcome, which the exporter renders as four zero outcome samples.
func (e *Engine) LoadStats() LoadStats {
	report := e.loadStats
	report.Failures = slices.Clone(report.Failures)
	return report
}

// Set stores data under key and blocks until a dump containing the record
// completed: nil confirms the write is on disk (PERS-04). A store rejection
// — ErrValueTooLarge — returns unchanged with no waiter registered.
func (e *Engine) Set(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	ver, drift, err := e.store.Set(key, data)
	if err != nil {
		return ver, drift, err
	}
	return ver, drift, e.await()
}

// Delete writes a fresh-version tombstone for key and blocks until a dump
// containing the tombstone completed. A missing or already-tombstoned key
// returns the store's ErrNotFound unchanged with no waiter registered.
func (e *Engine) Delete(key string) (version.RecordVersion, time.Duration, error) {
	ver, drift, err := e.store.Delete(key)
	if err != nil {
		return ver, drift, err
	}
	return ver, drift, e.await()
}

// SetIfAbsent stores data under key only when the key holds no record,
// including no tombstone, and blocks until a dump containing the record
// completed. Any existing record — a live value or a tombstone — returns
// the store's ErrAlreadyExists unchanged with no waiter registered, so a
// Delete-then-SetIfAbsent cannot resurrect the key.
func (e *Engine) SetIfAbsent(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	ver, drift, err := e.store.SetIfAbsent(key, data)
	if err != nil {
		return ver, drift, err
	}
	return ver, drift, e.await()
}

// Flush is the shutdown seam (D-19): the binary's signal path calls it
// BEFORE cancelling the engine context, turning residual dirtiness — remote
// merges applied via ApplyRemote that have no waiter — into one final
// write-confirmed dump. It is the only such path: Run's exit sweep resolves
// stranded waiters with ErrClosed without performing a final dump, so state
// merged after the last scheduled dump would otherwise die with the
// process. It is exactly FlushTimeout with the confirmation timeout: a dirty
// engine completes one covering dump and returns its error, a clean engine
// replies nil without a redundant save (the D-10 clean decision), ErrClosed
// once Run has exited, and ErrUnconfirmed when the dump outlives the
// deadline (PERSIST-08). Requires a running Run goroutine like every
// blocking engine call.
func (e *Engine) Flush() error {
	return e.FlushTimeout(0)
}

// FlushTimeout is the bounded flush seam (PERSIST-08, D-07): it forces the
// same covering dump as Flush but gives up after d — zero or negative means
// the configured ConfirmTimeout — returning ErrUnconfirmed when the dumper
// never answers. Both the flush-channel send and the reply wait are bound by
// that one deadline, so a dumper stuck in a kernel fsync can never park the
// caller. Snapshot and Restore ride this seam with the confirmation timeout,
// and the binary's shutdown passes its grace period.
func (e *Engine) FlushTimeout(d time.Duration) error {
	if d <= 0 {
		d = e.cfg.ConfirmTimeout
	}
	return e.flush(d)
}

// flush forces the dumper to complete one dump covering every unconfirmed
// write and returns that dump's error: it hands a reply channel to Run
// through flushCh, which Run services immediately — bypassing the debounce
// quiet window, D-10's "forces the current dump to complete" — replying
// after the forced dump completed. ErrClosed when Run has already exited.
// The closed check under the mutex and the flushCh send cannot be one
// atomic step — the send must happen outside mu — so a flush passing the
// check while Run's exit sweep completes in between would strand its send
// with no receiver forever. Both the send and the reply wait therefore
// also race runDone, the channel Run closes after the sweep: a reply found
// after runDone closed is exactly one the sweep answered (program order in
// Run), an absent one is the stranded case, and flush returns ErrClosed
// instead of hanging. Both waits are additionally bounded by one deadline
// armed through cfg.After (PERSIST-08, T-5-24): expiry — the send never
// accepted or the reply never delivered — is the wrapped ErrUnconfirmed,
// because the dump's outcome is unknown rather than failed.
func (e *Engine) flush(d time.Duration) error {
	reply := make(chan error, 1)
	if e.isClosed() {
		return ErrClosed
	}
	timeout := e.cfg.After(d)
	select {
	case e.flushCh <- reply:
	case <-e.runDone:
		return ErrClosed
	case <-timeout:
		return fmt.Errorf("%w: flush", ErrUnconfirmed)
	}
	select {
	case err := <-reply:
		return err
	case <-e.runDone:
		select {
		case err := <-reply: // the exit sweep answered us after all
			return err
		default:
			return ErrClosed
		}
	case <-timeout:
		return fmt.Errorf("%w: flush", ErrUnconfirmed)
	}
}
