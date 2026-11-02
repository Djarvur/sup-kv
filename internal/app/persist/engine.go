package persist

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Design defaults a zero or negative Config field is normalized to (D-04).
const (
	defaultDebounce = 50 * time.Millisecond
	defaultMaxWait  = time.Second
)

// Config holds the dump timing parameters; it is immutable once handed to
// New. Debounce is the quiet window a burst of writes may keep extending,
// MaxWait the bound on that extension measured from the oldest unconfirmed
// write: a dump starts no later than oldest+MaxWait under any write cadence
// (D-04, PERS-03).
type Config struct {
	Debounce time.Duration
	MaxWait  time.Duration
}

// normalized returns cfg with zero or negative fields replaced by the
// design defaults.
func (c Config) normalized() Config {
	if c.Debounce <= 0 {
		c.Debounce = defaultDebounce
	}
	if c.MaxWait <= 0 {
		c.MaxWait = defaultMaxWait
	}
	return c
}

// waiter is one write's confirmation ticket: done is closed exactly once —
// by a covering dump's success or failure fan-out, or by Run's exit sweep —
// and err, written under the engine mutex before the close, carries the
// resolution. Channel close over [sync.Cond] because only channels
// broadcast an error to every waiter (D-03, D-05).
type waiter struct {
	ver  version.RecordVersion
	done chan struct{}
	err  error
}

// Engine is the write-confirmation engine over one storage.Store: a local
// write registers a waiter for its freshly stamped version and blocks until
// a dump whose snapshot contains that record completed (nil — the write is
// on disk, PERS-04) or failed (ErrDumpFailed — the record stays readable in
// memory, D-05). Engine is safe for concurrent use by multiple goroutines;
// the engine mutex is held only to swap waiter lists and the cover version,
// never across codec or port I/O, and exactly one dumper goroutine — the
// caller-launched Run — serializes every Save.
type Engine struct {
	mu        sync.Mutex
	store     *storage.Store
	dumps     port.DumpStore
	cfg       Config
	waiters   []*waiter
	lastLocal version.RecordVersion
	dirty     bool
	oldest    time.Time
	changed   chan struct{}
	flushCh   chan chan error
	closed    bool
}

// New returns an engine persisting store's state through dumps. store and
// dumps must be non-nil. The engine spawns no goroutine itself: the caller
// launches Run, whose lifecycle is the documented API contract.
func New(store *storage.Store, dumps port.DumpStore, cfg Config) *Engine {
	return &Engine{
		store:   store,
		dumps:   dumps,
		cfg:     cfg.normalized(),
		changed: make(chan struct{}, 1),
		flushCh: make(chan chan error, 1),
	}
}

// Open is the one startup composition point: it loads the fallback chain
// through dumps with Parse as the readability oracle, decodes the first
// generation that reads (per-record checksum faults discard, never abort),
// seeds a clock with the dump's FULL watermark (VERS-04), rebuilds the
// store key by key via Merge, and wraps it all in an engine. The returned
// count is the discarded-record tally, for observability. ok false — no
// readable generation at all, or a generation-level decode error — is the
// empty start, never an error: the chain already exhausted .prev, and an
// empty node recovers from peers (PERS-02).
func Open(
	nodeID uint16,
	now func() time.Time,
	driftWarn time.Duration,
	dumps port.DumpStore,
	storeCfg storage.Config,
	engineCfg Config,
) (eng *Engine, discarded int) {
	var records map[string]storage.Record
	if dump, ok := dumps.Load(Parse); ok {
		decoded, count, err := Decode(dump)
		if err == nil {
			records, discarded = decoded, count
		}
	}
	if records == nil {
		records = make(map[string]storage.Record)
	}
	initialL, initialC := Watermark(records)
	clk := clock.NewClock(now, initialL, initialC, driftWarn)
	store := storage.NewStore(nodeID, clk, storeCfg)
	mergeRecords(store, records)
	return New(store, dumps, engineCfg), discarded
}

// mergeRecords installs every record via Merge, skipping one the
// config-shrunk value limit rejects — an oversized record must never abort
// startup (the belt over the codec's maxRecordBytes bound; Merge has no
// other error).
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
// flushCh requests force a dump immediately and receive its error after it
// completed (the D-10 snapshot flush seam; plan 03-02 wires the sender).
// Go 1.23+ timer semantics apply: plain Reset, no drain selects (Pitfall 9).
// Run exits when ctx is cancelled, sweeping every stranded waiter with
// ErrClosed, and returns ctx's error.
func (e *Engine) Run(ctx context.Context) error {
	var debounce, deadline *time.Timer
	for {
		var fired bool
		select {
		case <-ctx.Done():
			stopTimer(debounce)
			stopTimer(deadline)
			e.sweepClosed()
			return ctx.Err()
		case <-e.changed:
		case <-timerChan(debounce):
			debounce = nil
			fired = true
		case <-timerChan(deadline):
			deadline = nil
			fired = true
		case reply := <-e.flushCh:
			reply <- e.runDump()
		}
		if fired {
			_ = e.runDump() // the error belongs to the waiters; nobody is left to report to.
		}
		debounce, deadline = e.rearm(debounce, deadline)
	}
}

// rearm re-syncs the debounce and deadline timers with the engine's dirty
// state: while dirty, the debounce timer counts a fresh quiet window and
// the deadline timer — armed once per dirty epoch from that epoch's oldest
// change — keeps running untouched; clean means the last dump covered
// everything, so both timers stop.
func (e *Engine) rearm(debounce, deadline *time.Timer) (nextDebounce, nextDeadline *time.Timer) {
	e.mu.Lock()
	dirty, oldest := e.dirty, e.oldest
	e.mu.Unlock()
	if !dirty {
		stopTimer(debounce)
		stopTimer(deadline)
		return nil, nil
	}
	if debounce == nil {
		debounce = time.NewTimer(e.cfg.Debounce)
	} else {
		debounce.Reset(e.cfg.Debounce)
	}
	if deadline == nil {
		deadline = time.NewTimer(time.Until(oldest.Add(e.cfg.MaxWait)))
	}
	return debounce, deadline
}

// sweepClosed is Run's exit sweep: under the same mutex hold the write
// paths check, closed flips before the waiters are taken, so no writer can
// register after the sweep and block forever; every stranded waiter then
// resolves with ErrClosed (Pitfall 4).
func (e *Engine) sweepClosed() {
	e.mu.Lock()
	e.closed = true
	stranded := e.waiters
	e.waiters = nil
	for _, w := range stranded {
		w.err = ErrClosed
	}
	e.mu.Unlock()
	for _, w := range stranded {
		close(w.done)
	}
}

// runDump is the serialized dump cycle: under the mutex it captures the
// cover version and the current waiter list and clears the dirty epoch,
// then releases the mutex before any encode or I/O — write-path
// dirty-marking must never block on adapter I/O. On success every captured
// waiter resolves nil: the cover discipline is the PERS-04 invariant —
// lastLocal is updated under the mutex by every write path after its store
// write installed the record, so any waiter whose version is not greater
// than the captured cover had its record installed before the capture,
// hence inside the Records snapshot taken after, and confirmation always
// means on-disk containment, never elapsed time. On failure every current
// waiter resolves ErrDumpFailed while the state stays readable in memory,
// dirty re-arms from a fresh epoch and the re-armed debounce timer starts
// the next attempt one interval later — never a tight retry loop (D-05,
// research A8).
func (e *Engine) runDump() error {
	e.mu.Lock()
	cover := e.lastLocal
	captured := e.waiters
	e.waiters = nil
	e.dirty = false
	e.mu.Unlock()

	block, err := Encode(e.store.Records())
	if err == nil {
		err = e.dumps.Save(block)
	}
	if err != nil {
		e.failWaiters(captured)
		return err
	}
	e.confirmWaiters(captured, cover)
	return nil
}

// confirmWaiters resolves the covered waiters of a completed dump with nil.
// A captured waiter above the cover is kept for the next epoch — unreachable
// in practice, since the cover dominates every version registered before
// the capture, but stranded never.
func (e *Engine) confirmWaiters(captured []*waiter, cover version.RecordVersion) {
	e.mu.Lock()
	var covered []*waiter
	for _, w := range captured {
		if w.ver.Compare(cover) > 0 {
			e.waiters = append(e.waiters, w)
			continue
		}
		w.err = nil
		covered = append(covered, w)
	}
	e.mu.Unlock()
	for _, w := range covered {
		close(w.done)
	}
}

// failWaiters resolves every current waiter — the captured list plus any
// registered during the failed dump — with ErrDumpFailed, and re-dirties
// the engine from a fresh epoch so the surviving records reach the next
// successful dump.
func (e *Engine) failWaiters(captured []*waiter) {
	e.mu.Lock()
	stranded := make([]*waiter, 0, len(captured)+len(e.waiters))
	stranded = append(stranded, captured...)
	stranded = append(stranded, e.waiters...)
	e.waiters = nil
	e.dirty = true
	e.oldest = time.Now()
	for _, w := range stranded {
		w.err = ErrDumpFailed
	}
	e.mu.Unlock()
	for _, w := range stranded {
		close(w.done)
	}
}

// timerChan returns t's channel, or nil for a nil timer — a nil channel
// receives never, disabling that select case.
func timerChan(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

// stopTimer stops t when non-nil. Go 1.23+ timer semantics make Stop safe
// without draining (Pitfall 9).
func stopTimer(t *time.Timer) {
	if t != nil {
		t.Stop()
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
	return ver, drift, e.await(ver)
}

// Delete writes a fresh-version tombstone for key and blocks until a dump
// containing the tombstone completed. A missing or already-tombstoned key
// returns the store's ErrNotFound unchanged with no waiter registered.
func (e *Engine) Delete(key string) (version.RecordVersion, time.Duration, error) {
	ver, drift, err := e.store.Delete(key)
	if err != nil {
		return ver, drift, err
	}
	return ver, drift, e.await(ver)
}

// await registers a waiter for ver — freshly installed in the store by the
// caller — marks the engine dirty and blocks until a dump resolution closes
// the waiter. lastLocal tracks the highest locally stamped version under
// the mutex, keeping the cover discipline runDump resolves by. The
// dirty-mark path is generic — any state change dirties; only local writes
// register waiters, and Phase 4's remote merges will reuse the dirty path
// without waiters.
func (e *Engine) await(ver version.RecordVersion) error {
	w := &waiter{ver: ver, done: make(chan struct{})}
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return ErrClosed
	}
	if ver.Compare(e.lastLocal) > 0 {
		e.lastLocal = ver
	}
	if !e.dirty {
		e.oldest = time.Now()
	}
	e.dirty = true
	e.waiters = append(e.waiters, w)
	e.mu.Unlock()
	select {
	case e.changed <- struct{}{}:
	default:
	}
	<-w.done
	return w.err
}

// Get returns a copy of the live record for key — reads are never gated on
// dumps, and a failed dump leaves every record readable in memory (D-05).
func (e *Engine) Get(key string) (storage.Record, error) {
	return e.store.Get(key)
}

// Records returns a full snapshot of the map, tombstones included, with
// Data defensively copied — the same pass-through contract as Get.
func (e *Engine) Records() map[string]storage.Record {
	return e.store.Records()
}
