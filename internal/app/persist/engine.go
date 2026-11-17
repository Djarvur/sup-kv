package persist

import (
	"bytes"
	"context"
	"errors"
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
	runDone   chan struct{}
	closed    bool

	// dumpNanos, dumpTotal and dumpErrors carry DumpStats lock-free for
	// scrapers (OBS-01): runDump is the single writer — the serialized
	// dumper — so atomic reads need no engine mutex.
	dumpNanos  atomic.Int64
	dumpTotal  atomic.Uint64
	dumpErrors atomic.Uint64
}

// DumpStats is the dump-cycle gauge OBS-01 reads: the last dump's
// Encode+Save span in seconds — recorded on success and failure alike —
// the total dump attempts, and the failed ones.
type DumpStats struct {
	DurationSeconds float64
	Total           uint64
	Errors          uint64
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
		runDone: make(chan struct{}),
	}
}

// Open is the one startup composition point: it loads the fallback chain
// through dumps, decodes the first generation that decodes (per-record
// checksum faults discard, never abort), seeds a clock with the dump's
// FULL watermark (VERS-04), rebuilds the store key by key via Merge, and
// wraps it all in an engine. The returned count is the discarded-record
// tally, for observability. No decodable generation at all is the empty
// start, never an error: the chain already exhausted .prev, and an empty
// node recovers from peers (PERS-02). Decoding rides Load's own oracle
// argument: the generation Load accepts is exactly the one whose Decode
// succeeded, so Open holds no Decode-error fallback at all — the former
// guard after Load was dead code for every contract-honoring DumpStore
// (Parse is Decode's own error channel over the same bytes, codec.go) and
// is gone (03 IN-01, D-15).
func Open(
	nodeID uint16,
	now func() time.Time,
	driftWarn time.Duration,
	dumps port.DumpStore,
	storeCfg storage.Config,
	engineCfg Config,
) (eng *Engine, discarded int) {
	var records map[string]storage.Record
	decode := func(dump []byte) error {
		decoded, count, err := Decode(dump)
		if err != nil {
			return err
		}
		records, discarded = decoded, count
		return nil
	}
	dumps.Load(decode)
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
// flushCh requests force a dump immediately — bypassing the debounce quiet
// window — and receive the dump's error after it completed; a clean engine
// has nothing unconfirmed on disk and replies nil without a redundant save
// (the D-10 snapshot flush seam Snapshot and Restore drive). Go 1.23+ timer
// semantics apply: plain Reset, no drain selects (Pitfall 9). Run exits
// when ctx is cancelled, sweeping every stranded waiter with ErrClosed,
// closing runDone — the exit signal flush races its flushCh send and reply
// against, so a request stranding between the closed-check and the send can
// never hang — and returns ctx's error.
func (e *Engine) Run(ctx context.Context) error {
	var debounce, deadline *time.Timer
	for {
		var fired bool
		select {
		case <-ctx.Done():
			stopTimer(debounce)
			stopTimer(deadline)
			e.sweepClosed()
			close(e.runDone)
			return ctx.Err()
		case <-e.changed:
		case <-timerChan(debounce):
			debounce = nil
			fired = true
		case <-timerChan(deadline):
			deadline = nil
			fired = true
		case reply := <-e.flushCh:
			reply <- e.flushDump()
		}
		if fired {
			_ = e.runDump() // the error belongs to the waiters; nobody is left to report to.
		}
		debounce, deadline = e.rearm(debounce, deadline)
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
// resolves with ErrClosed, as does every flush request already handed to
// the exiting Run — the reply channels are buffered, so the sends never
// block (Pitfall 4). Run closes runDone only after the sweep returns: a
// reply observed once runDone is closed is exactly one the sweep answered,
// so flush can tell the answered from the stranded apart.
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
// cover version and the current waiter list and clears the dirty epoch,
// then releases the mutex before any encode or I/O — write-path
// dirty-marking must never block on adapter I/O. The Encode+Save span is
// measured and recorded into the DumpStats atomics on success and failure
// alike (OBS-01). On success every captured waiter resolves nil: the cover
// discipline is the PERS-04 invariant — lastLocal is updated under the
// mutex by every write path after its store write installed the record, so
// any waiter whose version is not greater than the captured cover had its
// record installed before the capture, hence inside the Records snapshot
// taken after, and confirmation always means on-disk containment, never
// elapsed time. On failure every current waiter resolves ErrDumpFailed
// while the state stays readable in memory, dirty re-arms from a fresh
// epoch and the re-armed debounce timer starts the next attempt one
// interval later — never a tight retry loop (D-05, research A8).
func (e *Engine) runDump() error {
	e.mu.Lock()
	cover := e.lastLocal
	captured := e.waiters
	e.waiters = nil
	e.dirty = false
	e.mu.Unlock()

	start := time.Now()
	block, err := Encode(e.store.Records())
	if err == nil {
		err = e.dumps.Save(block)
	}
	e.recordDump(time.Since(start), err)
	if err != nil {
		e.failWaiters(captured)
		return err
	}
	e.confirmWaiters(captured, cover)
	return nil
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
	}
}

// confirmWaiters resolves the covered waiters of a completed dump with nil.
// A captured waiter above the cover is kept for the next epoch — unreachable
// in practice, since the cover dominates every version registered before
// the capture, but stranded never: the kept waiter returns to e.waiters, so
// the next epoch's confirm/fail fan-out — or Run's exit sweep — provably
// resolves it. Beside the guard sits its invariant assertion (03 WR-01,
// D-15): a kept waiter must still be UNRESOLVED here, because resolution
// closes done exactly once and takes the waiter out of the engine — a
// resolved waiter reaching this branch is a double-resolution in the
// making, so it is answered by never re-queueing it (the next fan-out
// would double-close its done channel). The belt stays documented AND
// asserted; TestConfirmWaiters_aboveCoverInvariant pins both branches.
func (e *Engine) confirmWaiters(captured []*waiter, cover version.RecordVersion) {
	e.mu.Lock()
	var covered []*waiter
	for _, w := range captured {
		if w.ver.Compare(cover) > 0 {
			select {
			case <-w.done:
				// Invariant violated: already resolved, never re-queue.
				continue
			default:
			}
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

// ApplyRemote joins one remotely replicated record into the store and,
// only when the merge actually changed the stored record, marks the engine
// dirty with NO waiter — REPL-03's memory-apply contract, the seam await's
// doc reserved in Phase 3 ("only local writes register waiters, and Phase
// 4's remote merges will reuse the dirty path without waiters"): the
// replication core Acks a DataReplication as soon as the record is
// applied to memory, and the dump that persists it is asynchronous to that
// confirmation — the running Run picks the dirty flag up and dumps on the
// debounce schedule, so the changed e.changed nudge (the same non-blocking
// signal await sends) is what wakes an idle dumper to re-arm its timers.
// The dirty decision is coded to storage.Store.Merge's documented return
// contract (D-15, 03 IN-02 + 04 IN-02, threat T-6-01): a Conflict whose
// Winner is the incoming version is a displacement (the record became the
// stored record — changed), one whose Winner is the stored version a
// rejection (a stale redelivery installed nothing — unchanged), and no
// Conflict is either an equal-version redelivery over an existing record
// (a full no-op) or the first record under the key (installed with
// nothing displaced) — the live-read probe taken BEFORE the merge
// separates the two, so an equal redelivery leaves a clean engine clean
// with no redundant dump. The probe cannot see a stored tombstone, so an
// equal-version redelivery of a tombstone arms one redundant dump — the
// safe direction only: a genuinely changed record never skips its dump,
// and PERS-04's confirmation semantics are untouched.
// Store errors surface unchanged: an oversize record's ErrValueTooLarge
// merges nothing, dirties nothing and never wedges the engine (T-04-04).
// Not gated on Run's exit: the merge keeps the store converging even while
// the dumper is gone, exactly like reads keep serving (Get).
func (e *Engine) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	_, getErr := e.store.Get(key)
	conflicts, drift, err := e.store.Merge(key, rec)
	if err != nil {
		return conflicts, drift, err
	}
	displaced := len(conflicts) > 0 && conflicts[0].Winner == rec.Version
	if !displaced && getErr == nil {
		// Rejected stale redelivery or equal-version redelivery over the
		// live record the probe saw: the store did not change.
		return conflicts, drift, nil
	}
	e.markDirty()
	select {
	case e.changed <- struct{}{}:
	default:
	}
	return conflicts, drift, nil
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

// Restore replays the named snapshot's records as ordinary writes and
// deletes with fresh versions (D-09, PERS-05): a snapshot-live record is a
// fresh Set that beats any current tombstone, a snapshot tombstone a fresh
// Delete that beats any current live value, and keys absent from the
// snapshot keep their state untouched — the snapshot's own versions are
// never re-installed (no Merge); fresh versions are what make the restore
// replicate and win by normal LWW. A snapshot that already equals the
// store's current state — every record, version and tombstone — replays
// as a whole-restore no-op: the engine stays clean and the final flush
// replies nil without a redundant dump (D-15, 03 IN-02 + 04 IN-02);
// ANY difference replays in full, every snapshot record re-stamped fresh,
// preserving the fresh-version dominance the restore replicates and wins
// by. The replay runs in deterministic sorted-key order through the
// store's write path, and the call returns only after one final flush
// whose dump covers every replayed write — the same write-confirmation
// contract as Set, batched into one flush because the per-key semantics
// are identical while sequential per-write blocking would serialize N
// writes at one debounce each. Errors surface unchanged — a missing
// snapshot, a generation-level decode fault, a store rejection or a
// failed confirming dump — never a silent empty success.
func (e *Engine) Restore(name string) error {
	data, err := e.dumps.ReadSnapshot(name)
	if err != nil {
		return err
	}
	records, _, err := Decode(data)
	if err != nil {
		return err
	}
	if restoreChanges(e.store.Records(), records) {
		for _, key := range slices.Sorted(maps.Keys(records)) {
			err = replayRecord(e.store, key, records[key])
			if err != nil {
				return err
			}
		}
		e.markDirty()
	}
	return e.flush()
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
// config-shrunk value limit never aborts a restore.
func replayRecord(store *storage.Store, key string, rec storage.Record) error {
	if rec.Deleted {
		_, _, err := store.Delete(key)
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		return err
	}
	_, _, err := store.Set(key, rec.Data)
	if errors.Is(err, storage.ErrValueTooLarge) {
		return nil
	}
	return err
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

// markDirty marks the engine dirty from a fresh epoch without registering
// a waiter: the restore replay's store writes are unconfirmed until the
// final flush's covering dump — the cover discipline of await without the
// per-write blocking.
func (e *Engine) markDirty() {
	e.mu.Lock()
	if !e.dirty {
		e.oldest = time.Now()
	}
	e.dirty = true
	e.mu.Unlock()
}
