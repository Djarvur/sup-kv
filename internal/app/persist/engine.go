package persist

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Design defaults a zero or negative Config field is normalized to (D-04),
// the bounded dump-failure ladder's base and cap (D-05, PERSIST-05): the
// first failure retries after the base delay, every later one doubles it, and
// the cap bounds the wait — the confirmation timeout sits above it — and the
// confirmation deadline itself (PERSIST-08, D-07).
const (
	defaultDebounce       = 50 * time.Millisecond
	defaultMaxWait        = time.Second
	defaultRetryBase      = 100 * time.Millisecond
	defaultRetryCap       = 5 * time.Second
	defaultConfirmTimeout = 10 * time.Second

	// retryGrowth is the ladder's growth factor: every failure doubles the
	// previous wait until the cap binds.
	retryGrowth = 2
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

// waiter is one write's confirmation ticket: done is closed exactly once,
// and err — written before the close — carries the resolution. The ticket
// needs no version: runDump captures the list before taking the Records
// snapshot, so every captured waiter is covered by that dump (see
// confirmWaiters). Channel close over [sync.Cond] because only channels
// broadcast an error to every waiter (D-03, D-05).
//
// claimed is the single-claim protocol's guard (PERSIST-08, T-5-26): every
// resolver — a dump's success or failure fan-out, Run's exit sweep, or the
// awaiting caller's timeout — sets it under the engine mutex before writing
// err and closing done, and a resolver that finds it already set leaves both
// untouched. Capture alone never claims: a dump that merely took the waiter
// into its captured slice never owns it, so a dump hanging after capture
// cannot keep the caller's timeout from resolving the write.
type waiter struct {
	done    chan struct{}
	err     error
	claimed bool
}

// Engine is the write-confirmation engine over one storage.Store: a local
// write registers a waiter and blocks until a dump whose snapshot contains
// that record completed (nil — the write is on disk, PERS-04), failed
// (ErrDumpFailed — the record stays readable in memory, D-05), or did not
// complete within ConfirmTimeout (ErrUnconfirmed — the record is installed
// but not yet durable, PERSIST-08). Engine is safe for concurrent use by
// multiple goroutines; the engine mutex is held only to swap waiter lists and
// the dirty epoch, never across codec or port I/O, and exactly one dumper
// goroutine — the caller-launched Run — serializes every Save.
type Engine struct {
	mu      sync.Mutex
	store   *storage.Store
	dumps   port.DumpStore
	cfg     Config
	waiters []*waiter
	dirty   bool
	oldest  time.Time

	// retryDelay and retryAt are the dump-failure ladder state (D-05,
	// PERSIST-05): zero while healthy; a failed attempt advances the delay
	// and fixes the instant the retry channel is armed for, so intervening
	// writes cannot postpone the next attempt. Both are written by the dumper
	// goroutine under mu and read by rearm.
	retryDelay time.Duration
	retryAt    time.Time

	changed chan struct{}
	flushCh chan chan error
	runDone chan struct{}
	closed  bool

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

	// dumpNanos, dumpTotal and dumpErrors carry DumpStats lock-free for
	// scrapers (OBS-01): runDump is the single writer — the serialized
	// dumper — so atomic reads need no engine mutex. dumpDiskFull and
	// dumpPermission split the failed attempts by the adapter's class
	// (PERSIST-05); both are subsets of dumpErrors. dumpUnconfirmed counts
	// confirmation timeouts (PERSIST-08): its writers are the awaiting write
	// goroutines, one increment per timed-out waiter, and it is no subset of
	// dumpErrors — the dump's outcome may still be unknown.
	dumpNanos       atomic.Int64
	dumpTotal       atomic.Uint64
	dumpErrors      atomic.Uint64
	dumpDiskFull    atomic.Uint64
	dumpPermission  atomic.Uint64
	dumpUnconfirmed atomic.Uint64
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
// launches Run, whose lifecycle is the documented API contract.
func New(store *storage.Store, dumps port.DumpStore, cfg Config) *Engine {
	return &Engine{
		store:         store,
		dumps:         dumps,
		cfg:           cfg.normalized(),
		changed:       make(chan struct{}, 1),
		flushCh:       make(chan chan error, 1),
		runDone:       make(chan struct{}),
		valueMaxBytes: store.ValueMaxBytes(),
	}
}

// Open is the one startup composition point: it loads the fallback chain
// through dumps, decodes the first generation that decodes (per-record
// faults discard and are counted by class, never abort), seeds a clock with
// the dump's FULL watermark (VERS-04), rebuilds the store key by key via
// Merge, and wraps it all in an engine. The returned count is
// DecodeStats.Total() — the four discard classes summed, for observability.
// No decodable generation at all is the empty start, never an error: the
// chain already exhausted .prev, and an empty node recovers from peers
// (PERS-02). Decoding rides Load's own oracle argument: the generation Load
// accepts is exactly the one whose Decode succeeded, so Open holds no
// Decode-error fallback at all — the former guard after Load was dead code
// for every contract-honoring DumpStore (Decode is the readability oracle
// over the same bytes, codec.go) and is gone (03 IN-01, D-15).
//
// The decode runs before any store exists — its clock is seeded from the
// decoded records — so it reads the configured limit through
// storeCfg.ValueLimit(), the same helper NewStore normalizes with: the bound
// the decoder applies and the limit the store enforces cannot diverge
// (PERSIST-01).
func Open(
	nodeID uint16,
	now func() time.Time,
	driftWarn time.Duration,
	dumps port.DumpStore,
	storeCfg storage.Config,
	engineCfg Config,
) (eng *Engine, discarded int) {
	limit := storeCfg.ValueLimit()
	var records map[string]storage.Record
	var stats DecodeStats
	decode := func(dump []byte) error {
		decoded, decodedStats, err := Decode(dump, limit)
		if err != nil {
			return err
		}
		records, stats = decoded, decodedStats
		return nil
	}
	result := dumps.Load(now(), decode)
	if records == nil {
		records = make(map[string]storage.Record)
	}
	initialL, initialC := Watermark(records)
	clk := clock.NewClock(now, initialL, initialC, driftWarn)
	store := storage.NewStore(nodeID, clk, storeCfg)
	mergeRecords(store, records)
	eng = New(store, dumps, engineCfg)
	eng.loadStats = LoadStats{
		Loaded:     true,
		Outcome:    result.Outcome,
		Generation: result.Generation,
		Failures:   slices.Clone(result.Failures),
		Decoded:    stats,
		Swept:      result.Swept,
		SweepErr:   result.SweepErr,
	}
	eng.startLoadDirty(result.Outcome, now())
	return eng, stats.Total()
}

// startLoadDirty arms the post-load dirty epoch for every outcome but Main
// (D-04, PERSIST-04): a Prompt start has no generation at all, a Backup
// start's memory came from .prev, and an Unusable start is empty — each must
// flush before a Snapshot bytes-copies the file, otherwise the snapshot could
// capture a corrupt main and the first rotation could displace the last good
// generation. A Main load — even one whose decode discarded records — starts
// clean: the loaded generation is on disk and the same decoder reproduces the
// same in-memory state, while rewriting it would shorten the PERSIST-10
// evidence window. Called once by Open before Run exists, so no nudge is
// needed here; the first write's markDirty wakes the dumper.
func (e *Engine) startLoadDirty(outcome port.LoadOutcome, at time.Time) {
	if outcome == port.LoadMain {
		return
	}
	e.mu.Lock()
	e.dirty = true
	e.oldest = at
	e.mu.Unlock()
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

// mergeRecords installs every record via Merge, skipping one the
// config-shrunk value limit rejects — an oversized record must never abort
// startup. The decoder already applies the same configured limit, so this
// skip is the belt over that bound: Merge has no other error.
func mergeRecords(store *storage.Store, records map[string]storage.Record) {
	for key, rec := range records {
		_, _, err := store.Merge(key, rec)
		if errors.Is(err, storage.ErrValueTooLarge) {
			continue
		}
	}
}

// Run is the single dumper goroutine the caller launches. It serializes
// every dump, debounces bursts at cfg.Debounce and forces a dump start no
// later than the oldest unconfirmed write plus cfg.MaxWait — the deadline
// timer is armed from the clean-to-dirty transition timestamp and is never
// reset by later changes, only the debounce timer resets, so a 1 ms write
// cadence can never push the dump start past the bound (D-04, Pitfall 2).
// Every timer is a channel armed through cfg.After and held in a loop
// variable: re-arming replaces the channel, so a stale channel is
// unreachable and can never fire a dump, and no channel is ever created
// inside the select (PERSIST-12, research Pitfall 17). flushCh requests
// force a dump immediately — bypassing the debounce quiet window — and
// receive the dump's error after it completed; a clean engine has nothing
// unconfirmed on disk and replies nil without a redundant save (the D-10
// snapshot flush seam Snapshot and Restore drive). Run exits when ctx is
// cancelled, sweeping every stranded waiter with ErrClosed, closing
// runDone — the exit signal flush races its flushCh send and reply against,
// so a request stranding between the closed-check and the send can never
// hang — and returns ctx's error.
func (e *Engine) Run(ctx context.Context) error {
	var debounce, deadline, retry <-chan time.Time
	for {
		var fired bool
		select {
		case <-ctx.Done():
			e.sweepClosed()
			close(e.runDone)
			return ctx.Err()
		case <-e.changed:
		case <-debounce:
			// The debounce channel is re-armed unconditionally below, so a
			// fired one needs no clearing.
			fired = true
		case <-deadline:
			deadline = nil
			fired = true
		case <-retry:
			retry = nil
			fired = true
		case reply := <-e.flushCh:
			reply <- e.flushDump()
		}
		if fired {
			_ = e.runDump() // the error belongs to the waiters; nobody is left to report to.
		}
		debounce, deadline, retry = e.rearm(deadline, retry)
	}
}

// flushDump services a flushCh request: a dirty engine dumps immediately —
// the forced dump that must complete before a snapshot copies the main file
// (D-10) — while a clean engine has nothing unconfirmed on disk and replies
// nil without a redundant save.
func (e *Engine) flushDump() error {
	e.mu.Lock()
	dirty := e.dirty
	e.mu.Unlock()
	if !dirty {
		return nil
	}
	return e.runDump()
}

// rearm re-syncs the scheduler's timer channels with the engine's retry and
// dirty state: a pending retry owns the next attempt — arming only the retry
// channel, because a debounce or deadline would fire earlier than the ladder
// -- while a dirty engine counts a fresh quiet window on the debounce
// channel and keeps the deadline channel armed from the epoch's oldest
// change untouched; clean means the last dump covered everything, so no
// channel is armed. The debounce channel is replaced on every re-arm (a
// replaced channel is unreachable and simply dropped); the retry channel
// keeps its already-armed instant while one is pending, so a write arriving
// during backoff registers a waiter covered by that attempt instead of
// postponing it.
func (e *Engine) rearm(deadline, retry <-chan time.Time) (nextDebounce, nextDeadline, nextRetry <-chan time.Time) {
	e.mu.Lock()
	dirty, oldest := e.dirty, e.oldest
	delay, at := e.retryDelay, e.retryAt
	e.mu.Unlock()
	if delay > 0 {
		if retry == nil {
			retry = e.cfg.After(max(at.Sub(e.cfg.Now()), 0))
		}
		return nil, nil, retry
	}
	if !dirty {
		return nil, nil, nil
	}
	nextDebounce = e.cfg.After(e.cfg.Debounce)
	if deadline == nil {
		deadline = e.cfg.After(oldest.Add(e.cfg.MaxWait).Sub(e.cfg.Now()))
	}
	return nextDebounce, deadline, nil
}

// sweepClosed is Run's exit sweep: under the same mutex hold the write
// paths check, closed flips before the waiters are taken, so no writer can
// register after the sweep and block forever; every stranded waiter then
// resolves with ErrClosed, as does every flush request already handed to
// the exiting Run — the reply channels are buffered, so the sends never
// block (Pitfall 4). Run closes runDone only after the sweep returns: a
// reply observed once runDone is closed is exactly one the sweep answered,
// so flush can tell the answered from the stranded apart. A waiter the
// caller's timeout already claimed is skipped — the timeout owns its close
// (PERSIST-08, T-5-26).
func (e *Engine) sweepClosed() {
	e.mu.Lock()
	e.closed = true
	stranded := e.waiters
	e.waiters = nil
	claimed := make([]*waiter, 0, len(stranded))
	for _, w := range stranded {
		if w.claimed {
			continue
		}
		w.claimed = true
		w.err = ErrClosed
		claimed = append(claimed, w)
	}
	e.mu.Unlock()
	for _, w := range claimed {
		close(w.done)
	}
	for {
		select {
		case reply := <-e.flushCh:
			reply <- ErrClosed
		default:
			return
		}
	}
}

// runDump is the serialized dump cycle: under the mutex it captures the
// current waiter list and clears the dirty epoch, then releases the mutex
// before any encode or I/O — write-path dirty-marking must never block on
// adapter I/O. The cycle takes exactly one Records() copy and streams it
// through SaveStream, whose fill runs EncodeTo against the adapter's
// buffered temp writer: the encoded dump is never materialized as a second
// buffer (D-06, PERSIST-06). The Encode+Save span is measured and recorded
// into the DumpStats atomics on success and failure alike (OBS-01). On
// success every captured waiter resolves nil, and that is sound without a
// version watermark (PERS-04, RESEARCH Pitfall 15): await appends a waiter
// only after its store write installed the record, under the same mutex the
// capture holds, and the capture happens BEFORE Records takes the snapshot
// — so every captured waiter's record is in the snapshot this dump saves. A
// write racing the capture either appends before it (covered by this
// snapshot) or re-marks the engine dirty after it (covered by the next), so
// no waiter is ever confirmed against a dump that misses its record. On
// failure every current waiter resolves ErrDumpFailed while the state stays
// readable in memory, dirty re-arms from a fresh epoch and the bounded retry
// ladder arms the next attempt — 100 ms doubling to the 5 s cap, reset by a
// success — so a persistent failure never busy-loops (D-05, PERSIST-05,
// research A8).
func (e *Engine) runDump() error {
	e.mu.Lock()
	captured := e.waiters
	e.waiters = nil
	e.dirty = false
	e.mu.Unlock()

	start := e.cfg.Now()
	records := e.store.Records()
	err := e.dumps.SaveStream(func(w io.Writer) error {
		return EncodeTo(w, records, e.valueMaxBytes)
	})
	e.recordDump(e.cfg.Now().Sub(start), err)
	if err != nil {
		e.handleDumpFailure(err)
		e.failWaiters(captured)
		return err
	}
	e.resetRetry()
	e.confirmWaiters(captured)
	return nil
}

// handleDumpFailure applies one failed attempt's decay and report (D-05,
// PERSIST-05): the class counters, the next ladder rung fixed at this
// failure's instant — so a write arriving during the backoff registers a
// waiter the armed retry covers instead of postponing it — and exactly one
// OnDumpFailure callback carrying the underlying error and the adapter's
// class. It runs on the dumper goroutine once per failed attempt; the
// callback must not block.
func (e *Engine) handleDumpFailure(err error) {
	class := port.FailureClassOf(err)
	e.mu.Lock()
	e.retryDelay = nextRetry(e.retryDelay)
	e.retryAt = e.cfg.Now().Add(e.retryDelay)
	e.mu.Unlock()
	switch class {
	case port.FailureDiskFull:
		e.dumpDiskFull.Add(1)
	case port.FailurePermission:
		e.dumpPermission.Add(1)
	case port.FailureOther:
	}
	if e.cfg.OnDumpFailure != nil {
		e.cfg.OnDumpFailure(err, class)
	}
}

// nextRetry returns the ladder rung after d: the first failure (d == 0)
// arms the base delay, every later one doubles it, and the cap bounds the
// wait.
func nextRetry(d time.Duration) time.Duration {
	if d <= 0 {
		return defaultRetryBase
	}
	next := d * retryGrowth
	if next > defaultRetryCap {
		return defaultRetryCap
	}
	return next
}

// resetRetry clears the ladder after a successful attempt: the next failure
// starts again from the base delay and covers the waiters immediately.
func (e *Engine) resetRetry() {
	e.mu.Lock()
	e.retryDelay = 0
	e.retryAt = time.Time{}
	e.mu.Unlock()
}

// recordDump stores one dump cycle's observability (OBS-01): the span of
// the Encode+Save pair, the attempt total and — when err is non-nil — the
// error count, all through the lock-free atomics DumpStats reads.
func (e *Engine) recordDump(span time.Duration, err error) {
	e.dumpNanos.Store(int64(span))
	e.dumpTotal.Add(1)
	if err != nil {
		e.dumpErrors.Add(1)
	}
}

// DumpStats returns the dump-cycle counters (OBS-01) — a value copy over
// the atomics, safe from any goroutine; runDump is the single writer.
func (e *Engine) DumpStats() DumpStats {
	return DumpStats{
		DurationSeconds: float64(e.dumpNanos.Load()) / float64(time.Second),
		Total:           e.dumpTotal.Load(),
		Errors:          e.dumpErrors.Load(),
		DiskFull:        e.dumpDiskFull.Load(),
		Permission:      e.dumpPermission.Load(),
		Unconfirmed:     e.dumpUnconfirmed.Load(),
	}
}

// confirmWaiters resolves every waiter captured by a completed dump with
// nil. Nothing is retained: the capture proof in runDump is that await
// appends a waiter only after the caller's store write installed the record
// and under the same mutex the capture holds, while the capture precedes
// Encode(e.store.Records()) — every captured record is therefore inside the
// snapshot the dump saved, and confirmation always means on-disk
// containment, never elapsed time (PERS-04, RESEARCH Pitfall 15). A write
// that loses the race to the capture is not in captured; it stays
// registered and the next dump — or Run's exit sweep — resolves it. A
// captured waiter the caller's timeout already claimed is skipped: the
// timeout owns its resolution and close (PERSIST-08, T-5-26).
func (e *Engine) confirmWaiters(captured []*waiter) {
	e.mu.Lock()
	claimed := make([]*waiter, 0, len(captured))
	for _, w := range captured {
		if w.claimed {
			continue
		}
		w.claimed = true
		w.err = nil
		claimed = append(claimed, w)
	}
	e.mu.Unlock()
	for _, w := range claimed {
		close(w.done)
	}
}

// failWaiters resolves every current waiter — the captured list plus any
// registered during the failed dump — with ErrDumpFailed, and re-dirties
// the engine from a fresh epoch so the surviving records reach the next
// successful dump. Waiters the caller's timeout already claimed are skipped:
// the timeout owns their resolution and close (PERSIST-08, T-5-26).
func (e *Engine) failWaiters(captured []*waiter) {
	e.mu.Lock()
	stranded := make([]*waiter, 0, len(captured)+len(e.waiters))
	stranded = append(stranded, captured...)
	stranded = append(stranded, e.waiters...)
	e.waiters = nil
	e.dirty = true
	e.oldest = e.cfg.Now()
	claimed := make([]*waiter, 0, len(stranded))
	for _, w := range stranded {
		if w.claimed {
			continue
		}
		w.claimed = true
		w.err = ErrDumpFailed
		claimed = append(claimed, w)
	}
	e.mu.Unlock()
	for _, w := range claimed {
		close(w.done)
	}
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

// await registers a waiter — the caller's record is freshly installed in the
// store — marks the engine dirty and blocks until a dump resolution closes
// the waiter or the confirmation timeout expires. Registering under the same
// mutex hold that marks dirty is what lets runDump treat every captured
// waiter as covered: the store write happened before this call, and the
// dump's capture cannot interleave between that write and the append. The
// dirty-mark path is generic — any state change dirties; only local writes
// register waiters, and remote merges reuse the dirty path without waiters.
//
// The timeout branch is the single-claim protocol's other half (PERSIST-08,
// T-5-26/T-5-27): under the mutex it first re-checks claimed — a resolver
// that won the race set the error and owns the close, so await only waits
// for that close and returns the resolver's answer. Otherwise it claims the
// waiter, stores ErrUnconfirmed, removes it from the slice so a permanently
// hung dumper cannot grow memory, unlocks, counts the unconfirmed outcome and
// returns. The record stays installed; the engine stays dirty exactly as the
// capture proof promises. Only ErrUnconfirmed is returned — never nil,
// never ErrDumpFailed.
func (e *Engine) await() error {
	w := &waiter{done: make(chan struct{})}
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return ErrClosed
	}
	if !e.dirty {
		e.oldest = e.cfg.Now()
	}
	e.dirty = true
	e.waiters = append(e.waiters, w)
	e.mu.Unlock()
	select {
	case e.changed <- struct{}{}:
	default:
	}
	timeout := e.cfg.After(e.cfg.ConfirmTimeout)
	select {
	case <-w.done:
		return w.err
	case <-timeout:
	}
	e.mu.Lock()
	if w.claimed {
		e.mu.Unlock()
		<-w.done // the resolver owns the close; return its answer.
		return w.err
	}
	w.claimed = true
	w.err = ErrUnconfirmed
	e.waiters = slices.DeleteFunc(e.waiters, func(candidate *waiter) bool { return candidate == w })
	e.mu.Unlock()
	e.dumpUnconfirmed.Add(1)
	return ErrUnconfirmed
}

// ApplyRemote joins one remotely replicated record into the store and,
// only when the merge actually changed the stored record, marks the engine
// dirty with NO waiter — REPL-03's memory-apply contract, the seam await's
// doc reserved in Phase 3 ("only local writes register waiters, and Phase
// 4's remote merges will reuse the dirty path without waiters"): the
// replication core Acks a DataReplication as soon as the record is
// applied to memory, and the dump that persists it is asynchronous to that
// confirmation — the running Run picks the dirty flag up and dumps on the
// debounce schedule, so markDirty's non-blocking e.changed nudge is what wakes
// an idle dumper to re-arm its timers.
// The dirty decision is exactly storage.Store.MergeChanged's changed result
// (D-15, 03 IN-02 + 04 IN-02, threat T-6-01; PERSIST-07, ADR 0012): the
// store alone knows whether the incoming record became the stored record —
// a strictly newer version, the first record under the key, or an
// equal-version content tie-break win — while a stale rejection, a losing
// tie-break and an identical redelivery install nothing. The old
// Conflict/version heuristic cannot tell a tie-break win from a loss,
// because both pair the same version bytes, and the live-read probe could
// not see a stored tombstone; the changed result removes both ambiguities,
// so a no-op redelivery always leaves a clean engine clean (no redundant
// dump) and a content change never skips its dump.
// Store errors surface unchanged: an oversize record's ErrValueTooLarge
// merges nothing, dirties nothing and never wedges the engine (T-04-04).
// Not gated on Run's exit: the merge keeps the store converging even while
// the dumper is gone, exactly like reads keep serving (Get).
func (e *Engine) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	conflicts, changed, drift, err := e.store.MergeChanged(key, rec)
	if err != nil {
		return conflicts, drift, err
	}
	if !changed {
		return conflicts, drift, nil
	}
	e.markDirty()
	return conflicts, drift, nil
}

// Get returns a copy of the live record for key — reads are never gated on
// dumps, and a failed dump leaves every record readable in memory (D-05).
func (e *Engine) Get(key string) (storage.Record, error) {
	return e.store.Get(key)
}

// Lookup returns a copy of the stored record for key, tombstones included:
// the tombstone-visible replication view, unlike Get's live-only API read.
// It consults no dump or confirmation state — a failed dump leaves every
// record readable in memory (D-05).
func (e *Engine) Lookup(key string) (storage.Record, bool) {
	return e.store.Lookup(key)
}

// Version returns the stored version for key, tombstones included, and
// false when the key holds no record — the cheap per-entry replication
// probe, copying no Data and consulting no dump state.
func (e *Engine) Version(key string) (version.RecordVersion, bool) {
	return e.store.Version(key)
}

// Versions returns the caller-owned key→version snapshot, tombstones
// included — the listing view that copies no record data and consults no
// dump state.
func (e *Engine) Versions() map[string]version.RecordVersion {
	return e.store.Versions()
}

// Observe absorbs one received version into the store's clock without
// touching any record: the per-DataSync-part HLC update (PROTO-01). It
// consults no dump or confirmation state.
func (e *Engine) Observe(ver version.RecordVersion) time.Duration {
	return e.store.Observe(ver)
}

// KeyCount returns the number of stored records — tombstones included — in
// constant time: the startup empty-vs-data selection (PROTO-02). It clones
// no record data and consults no dump or confirmation state.
func (e *Engine) KeyCount() int {
	return e.store.Len()
}

// Generation returns the store's monotone content generation — moved only
// when a stored record actually changed — which the listing build records
// and reuses only while unchanged (04-08, T-4-43). It consults no dump or
// confirmation state.
func (e *Engine) Generation() uint64 {
	return e.store.Generation()
}

// Records returns a full snapshot of the map, tombstones included, with
// Data defensively copied — the same pass-through contract as Get.
func (e *Engine) Records() map[string]storage.Record {
	return e.store.Records()
}

// Snapshot forces the current dump to complete — every confirmed write is
// on disk — then byte-copies the finished main dump under name through the
// adapter (D-10, PERS-05): the snapshot file is byte-identical to the main
// dump at creation, and a flush triggered while state is dirty forces a new
// dump before the copy. A flush error aborts with nothing copied: state
// that is not confirmed on disk must never become a recovery point. Safe
// against a dump running concurrently with the copy: the adapter's atomic
// renames mean no torn read, and its save-snapshot serialization closes
// the rotate window in which the main path is briefly absent — the copy
// always reads one complete generation. Requires a running Run goroutine
// like every blocking engine call.
func (e *Engine) Snapshot(name string) error {
	err := e.flush()
	if err != nil {
		return err
	}
	return e.dumps.CreateSnapshot(name)
}

// RestoredRecord is one record a Restore actually replayed: the snapshot
// key, the fresh version the store stamped for the replay, the record's
// data and whether the replay installed a tombstone. Records a replay
// skips — a tombstone with nothing to beat, a value the store's shrunk
// limit rejects — are absent, and the order is the replay's deterministic
// sorted-key order, so the notify sequence a snapshot load drives is stable
// across runs.
type RestoredRecord struct {
	Key     string
	Version version.RecordVersion
	Data    []byte
	Deleted bool
}

// Restore replays the named snapshot's records as ordinary writes and
// deletes with fresh versions (D-09, PERS-05): a snapshot-live record is a
// fresh Set that beats any current tombstone, a snapshot tombstone a fresh
// Delete that beats any current live value, and keys absent from the
// snapshot keep their state untouched — the snapshot's own versions are
// never re-installed (no Merge); fresh versions are what make the restore
// replicate and win by normal LWW. A snapshot that already equals the
// store's current state — every record, version and tombstone — replays
// as a whole-restore no-op: the engine stays clean, the returned record
// list is empty and the final flush replies nil without a redundant dump
// (D-15, 03 IN-02 + 04 IN-02); ANY difference replays in full, every
// snapshot record re-stamped fresh, preserving the fresh-version dominance
// the restore replicates and wins by. The replay runs in deterministic
// sorted-key order through the store's write path, and the call returns
// only after one final flush whose dump covers every replayed write — the
// same write-confirmation contract as Set, batched into one flush because
// the per-key semantics are identical while sequential per-write blocking
// would serialize N writes at one debounce each. The returned list names
// exactly the records the replay installed — sorted by key, skipped records
// absent — and is returned even when the confirming dump failed, so a
// snapshot load can replicate what is already in memory (PROTO-08) while
// the error still reports the dump honestly: a failed final dump surfaces
// as ErrDumpFailed carrying its cause, and a closed engine as ErrClosed.
// A mid-replay store error aborts deterministically instead: the records
// installed before it stay marked dirty (the dumper persists them) and come
// back with the error, so no installed-but-unreturned record can be lost
// (IN-04) — the caller's snapshot-load notify covers exactly that list.
// Load and decode errors surface unchanged — a missing snapshot or a
// generation-level decode fault — never a silent empty success.
func (e *Engine) Restore(name string) ([]RestoredRecord, error) {
	data, err := e.dumps.ReadSnapshot(name)
	if err != nil {
		return nil, err
	}
	records, _, err := Decode(data, e.valueMaxBytes)
	if err != nil {
		return nil, err
	}
	restored, err := e.replaySnapshot(records)
	if err != nil {
		return restored, err
	}
	err = e.flush()
	if err != nil && !errors.Is(err, ErrClosed) {
		return restored, fmt.Errorf("%w: %w", ErrDumpFailed, err)
	}
	return restored, err
}

// replaySnapshot replays the snapshot's records in deterministic sorted-key
// order, marks the engine dirty whenever anything was installed, and returns
// exactly the records the replay installed. A mid-replay store error aborts
// the replay deterministically and returns the PARTIAL list with the error
// (IN-04): the records installed before the abort are already in memory, so
// they are marked dirty for the next dump and returned so the caller's
// totality notify covers them — an installed-but-unreturned record would be
// neither dumped nor replicated.
func (e *Engine) replaySnapshot(records map[string]storage.Record) ([]RestoredRecord, error) {
	restored := make([]RestoredRecord, 0, len(records))
	if !restoreChanges(e.store.Records(), records) {
		return restored, nil
	}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		ver, replayed, err := replayRecord(e.store, key, records[key])
		if err != nil {
			if len(restored) > 0 {
				e.markDirty()
			}
			return restored, err
		}
		if !replayed {
			continue
		}
		restored = append(restored, RestoredRecord{
			Key:     key,
			Version: ver,
			Data:    bytes.Clone(records[key].Data),
			Deleted: records[key].Deleted,
		})
	}
	e.markDirty()
	return restored, nil
}

// restoreChanges reports whether replaying the snapshot's records would
// change the store's current state: the snapshot differs from the store
// when any of its keys holds a different record — version, data or
// tombstone flag — than the store does now. Keys the snapshot does not
// carry never matter, exactly as the replay never touches them; a
// snapshot equal to the store's current state is a whole-restore no-op
// (D-15), while any difference replays in full with every snapshot record
// re-stamped fresh (D-09's dominance contract).
func restoreChanges(current, snap map[string]storage.Record) bool {
	for key, rec := range snap {
		stored, ok := current[key]
		if !ok || stored.Version != rec.Version || stored.Deleted != rec.Deleted || !bytes.Equal(stored.Data, rec.Data) {
			return true
		}
	}
	return false
}

// replayRecord installs one snapshot record through the store's ordinary
// write path (D-09): a tombstone replays as a fresh-version Delete whose
// ErrNotFound is a skip — a missing or already-tombstoned key has nothing
// to beat, and Delete writes nothing there anyway — and a live record as a
// fresh-version Set whose ErrValueTooLarge is a skip, mirroring Open, so a
// config-shrunk value limit never aborts a restore. The returned version is
// the store-stamped fresh version and replayed reports whether the store
// installed the record; a skip returns the zero version, replayed=false and
// a nil error, so a real replay can never read as a skip. The tombstone flag
// is not part of the signal — the caller already knows it from the snapshot
// entry.
func replayRecord(store *storage.Store, key string, rec storage.Record) (version.RecordVersion, bool, error) {
	if rec.Deleted {
		ver, _, err := store.Delete(key)
		if errors.Is(err, storage.ErrNotFound) {
			return version.RecordVersion{}, false, nil
		}
		if err != nil {
			return version.RecordVersion{}, false, err
		}
		return ver, true, nil
	}
	ver, _, err := store.Set(key, rec.Data)
	if errors.Is(err, storage.ErrValueTooLarge) {
		return version.RecordVersion{}, false, nil
	}
	if err != nil {
		return version.RecordVersion{}, false, err
	}
	return ver, true, nil
}

// Flush is the shutdown seam (D-19): the binary's signal path calls it
// BEFORE cancelling the engine context, turning residual dirtiness — remote
// merges applied via ApplyRemote that have no waiter — into one final
// write-confirmed dump. It is the only such path: Run's exit sweep resolves
// stranded waiters with ErrClosed without performing a final dump, so state
// merged after the last scheduled dump would otherwise die with the
// process. Delegates to flush with its exact contract: a dirty engine
// completes one covering dump and returns its error, a clean engine replies
// nil without a redundant save (the D-10 clean decision), and ErrClosed
// once Run has exited. Requires a running Run goroutine like every blocking
// engine call.
func (e *Engine) Flush() error {
	return e.flush()
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
// instead of hanging.
func (e *Engine) flush() error {
	reply := make(chan error, 1)
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return ErrClosed
	}
	select {
	case e.flushCh <- reply:
	case <-e.runDone:
		return ErrClosed
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
	}
}

// markDirty marks the engine dirty from a fresh epoch without registering a
// waiter and wakes an idle dumper through the non-blocking e.changed nudge:
// the restore replay's store writes are unconfirmed until the final flush's
// covering dump — the cover discipline of await without the per-write
// blocking — and the mid-replay error path needs the same wake-up (Phase-4
// review New-1). The nudge lives here, not at each caller, so every dirty
// path sends exactly one signal; the channel has capacity one and the send is
// non-blocking, so repeated marks collapse into one wake-up.
func (e *Engine) markDirty() {
	e.mu.Lock()
	if !e.dirty {
		e.oldest = e.cfg.Now()
	}
	e.dirty = true
	e.mu.Unlock()
	select {
	case e.changed <- struct{}{}:
	default:
	}
}
