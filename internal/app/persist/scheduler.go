package persist

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The bounded dump-failure ladder's base and cap (D-05, PERSIST-05): the
// first failure retries after the base delay, every later one doubles it, and
// the cap bounds the wait.
const (
	defaultRetryBase = 100 * time.Millisecond
	defaultRetryCap  = 5 * time.Second

	// retryGrowth is the ladder's growth factor: every failure doubles the
	// previous wait until the cap binds.
	retryGrowth = 2
)

// dumpStats is the engine's one unexported atomic stats holder (OBS-01,
// PERSIST-11): the scheduler records every attempt's span and outcome here,
// the confirmer's timeout branch increments unconfirmed, and DumpStats reads
// the fields into the exported value copy. One published holder means the
// two writers and the exporter never see two copies of a counter.
type dumpStats struct {
	durationNanos atomic.Int64
	total         atomic.Uint64
	errors        atomic.Uint64
	diskFull      atomic.Uint64
	permission    atomic.Uint64
	other         atomic.Uint64
	unconfirmed   atomic.Uint64
}

// scheduler is the dump-scheduling role (D-10, PERSIST-11): it owns the
// dumper loop's timer channels, the bounded retry ladder and the one
// unexported dumpStats atomic holder behind Engine.DumpStats. It talks to
// the confirmation role only through that role's methods — capture before
// the Records snapshot, then confirm or fail the captured waiters — and its
// own mutex guards only the ladder state, so the two roles never nest locks
// (T-5-30).
//
// The scheduler is the only writer of the dump counters: runDump is the
// serialized dumper, so its atomic reads need no scheduler mutex.
type scheduler struct {
	mu         sync.Mutex
	retryDelay time.Duration
	retryAt    time.Time
	lastErr    error

	store         *storage.Store
	dumps         port.DumpStore
	cfg           Config
	conf          *confirmer
	stats         *dumpStats
	valueMaxBytes int
}

// newScheduler returns the dump-scheduling role wired to the store, the dump
// port, the engine's normalized config, the confirmation role and the one
// shared dump-stats holder.
func newScheduler(
	store *storage.Store,
	dumps port.DumpStore,
	cfg Config,
	conf *confirmer,
	stats *dumpStats,
) *scheduler {
	return &scheduler{
		store:         store,
		dumps:         dumps,
		cfg:           cfg,
		conf:          conf,
		stats:         stats,
		valueMaxBytes: store.ValueMaxBytes(),
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
// inside the select (PERSIST-12, research Pitfall 17). flushCh requests force
// a dump immediately, bypassing the debounce quiet window; an explicit flush
// attempts even while a retry rung is pending, while a snapshot request defers
// to that rung (WR-02). Run exits when ctx is
// cancelled, sweeping every stranded waiter with ErrClosed, closing
// runDone — the exit signal flush races its flushCh send and reply against,
// so a request stranding between the closed-check and the send can never
// hang — and returns ctx's error.
func (e *Engine) Run(ctx context.Context) error {
	return e.sched.run(ctx)
}

// run is the loop behind Run: the exported method keeps the documented API
// contract, and this one owns the scheduler's state.
func (s *scheduler) run(ctx context.Context) error {
	var debounce, deadline, retry <-chan time.Time
	var retryArmedAt time.Time
	for {
		var fired bool
		select {
		case <-ctx.Done():
			s.conf.exit()
			return ctx.Err()
		case <-s.conf.changed:
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
		case req := <-s.conf.flushCh:
			req.reply <- s.flushDump(req)
		}
		if fired {
			_ = s.runDump() // the error belongs to the waiters; nobody is left to report to.
		}
		debounce, deadline, retry, retryArmedAt = s.rearm(deadline, retry, retryArmedAt)
	}
}

// flushDump services a flushRequest: a dirty engine dumps immediately — the
// forced dump that must complete before a snapshot copies the main file
// (D-10) — while a clean engine replies nil without a redundant save. A
// snapshot request does not bypass a pending retry rung (WR-02): while the
// ladder is armed it answers the last classified failure, wrapped as
// ErrDumpFailed for the RPC's Unavailable class, instead of forcing an extra
// attempt, so a client snapshot loop cannot drive one full Records copy and
// one ERROR line per request. An explicit Flush/FlushTimeout always attempts
// one dump, even while a rung is pending: shutdown must never skip the final
// dump (N-1).
func (s *scheduler) flushDump(req flushRequest) error {
	if !s.conf.isDirty() {
		return nil
	}
	if req.snapshot {
		err := s.pendingFailure()
		if err != nil {
			return err
		}
	}
	return s.runDump()
}

// rearm re-syncs the scheduler's timer channels with the retry and dirty
// state: a pending retry owns the next attempt — arming only the retry
// channel, because a debounce or deadline would fire earlier than the ladder
// -- while a dirty engine counts a fresh quiet window on the debounce
// channel and keeps the deadline channel armed from the epoch's oldest
// change untouched; clean means the last dump covered everything, so no
// channel is armed. The debounce channel is replaced on every re-arm (a
// replaced channel is unreachable and simply dropped); the retry channel
// keeps its already-armed instant while the pending rung is unchanged, so a
// write arriving during backoff registers a waiter covered by that attempt
// instead of postponing it, and is replaced when a flush-forced failure
// advanced the ladder, so the next attempt fires at the newly computed
// instant rather than the stale one (WR-02). retryArmedAt is the run loop's
// record of the instant the current retry channel was armed for.
func (s *scheduler) rearm(
	deadline, retry <-chan time.Time,
	retryArmedAt time.Time,
) (nextDebounce, nextDeadline, nextRetry <-chan time.Time, nextRetryArmedAt time.Time) {
	dirty, oldest := s.conf.epoch()
	s.mu.Lock()
	delay, at := s.retryDelay, s.retryAt
	s.mu.Unlock()
	if delay > 0 {
		if retry == nil || !retryArmedAt.Equal(at) {
			retry = s.cfg.After(max(at.Sub(s.cfg.Now()), 0))
			retryArmedAt = at
		}
		return nil, nil, retry, retryArmedAt
	}
	if !dirty {
		return nil, nil, nil, time.Time{}
	}
	nextDebounce = s.cfg.After(s.cfg.Debounce)
	if deadline == nil {
		deadline = s.cfg.After(oldest.Add(s.cfg.MaxWait).Sub(s.cfg.Now()))
	}
	return nextDebounce, deadline, nil, time.Time{}
}

// runDump is the serialized dump cycle: the confirmation role captures the
// current waiter list and clears the dirty epoch before any encode or I/O —
// write-path dirty-marking must never block on adapter I/O. The content
// generation is read immediately BEFORE Records() so the published watermark
// covers the snapshot it describes (APIS-04, D-04): an install between the
// two reads is in the snapshot but publishes one dump later, while an
// install before the read is in both. The cycle takes
// exactly one Records() copy and streams it through SaveStream, whose fill
// runs EncodeTo against the adapter's buffered temp writer: the encoded dump
// is never materialized as a second buffer (D-06, PERSIST-06). The
// Encode+Save span is measured and recorded into the dumpStats atomics on
// success and failure alike (OBS-01). On success every captured waiter
// resolves nil, and that is sound without a version watermark (PERS-04,
// RESEARCH Pitfall 15): await appends a waiter only after its store write
// installed the record, under the same mutex the capture holds, and the
// capture happens BEFORE Records takes the snapshot — so every captured
// waiter's record is in the snapshot this dump saves. A write racing the
// capture either appends before it (covered by this snapshot) or re-marks
// the engine dirty after it (covered by the next), so no waiter is ever
// confirmed against a dump that misses its record. After the waiters are
// confirmed, the successful dump publishes the captured generation through
// cfg.OnConfirmed — the Watch hub's durability watermark (D-04); a failed
// dump publishes nothing. On failure every current
// waiter resolves ErrDumpFailed while the state stays readable in memory,
// dirty re-arms from a fresh epoch and the bounded retry ladder arms the
// next attempt — 100 ms doubling to the 5 s cap, reset by a success — so a
// persistent failure never busy-loops (D-05, PERSIST-05, research A8).
func (s *scheduler) runDump() error {
	captured := s.conf.capture()

	start := s.cfg.Now()
	gen := s.store.Generation()
	records := s.store.Records()
	err := s.dumps.SaveStream(func(w io.Writer) error {
		return EncodeTo(w, records, s.valueMaxBytes)
	})
	s.recordDump(s.cfg.Now().Sub(start), err)
	if err != nil {
		s.handleDumpFailure(err)
		s.conf.failWaiters(captured)
		return err
	}
	s.resetRetry()
	s.conf.confirmWaiters(captured)
	if s.cfg.OnConfirmed != nil {
		s.cfg.OnConfirmed(gen)
	}
	return nil
}

// handleDumpFailure applies one failed attempt's decay and report (D-05,
// PERSIST-05): the class counters, the next ladder rung fixed at this
// failure's instant — so a write arriving during the backoff registers a
// waiter the armed retry covers instead of postponing it — the classified
// failure the next flush answers with instead of forcing an extra attempt
// (WR-02), and exactly one OnDumpFailure callback carrying the underlying
// error and the adapter's class. It runs on the dumper goroutine once per
// failed attempt; the callback must not block.
func (s *scheduler) handleDumpFailure(err error) {
	class := port.FailureClassOf(err)
	s.mu.Lock()
	s.retryDelay = nextRetry(s.retryDelay)
	s.retryAt = s.cfg.Now().Add(s.retryDelay)
	s.lastErr = err
	s.mu.Unlock()
	switch class {
	case port.FailureDiskFull:
		s.stats.diskFull.Add(1)
	case port.FailurePermission:
		s.stats.permission.Add(1)
	case port.FailureOther:
		s.stats.other.Add(1)
	}
	if s.cfg.OnDumpFailure != nil {
		s.cfg.OnDumpFailure(err, class)
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
// starts again from the base delay, covers the waiters immediately and is
// again the last classified failure a flush may report.
func (s *scheduler) resetRetry() {
	s.mu.Lock()
	s.retryDelay = 0
	s.retryAt = time.Time{}
	s.lastErr = nil
	s.mu.Unlock()
}

// recordDump stores one dump cycle's observability (OBS-01): the span of
// the Encode+Save pair, the attempt total and — when err is non-nil — the
// error count, all through the lock-free atomics DumpStats reads.
func (s *scheduler) recordDump(span time.Duration, err error) {
	s.stats.durationNanos.Store(int64(span))
	s.stats.total.Add(1)
	if err != nil {
		s.stats.errors.Add(1)
	}
}

// DumpStats returns the dump-cycle counters (OBS-01) — a value copy over
// the atomics, safe from any goroutine; runDump is the single writer.
func (e *Engine) DumpStats() DumpStats {
	stats := e.sched.stats
	return DumpStats{
		DurationSeconds: float64(stats.durationNanos.Load()) / float64(time.Second),
		Total:           stats.total.Load(),
		Errors:          stats.errors.Load(),
		DiskFull:        stats.diskFull.Load(),
		Permission:      stats.permission.Load(),
		Other:           stats.other.Load(),
		Unconfirmed:     stats.unconfirmed.Load(),
	}
}

// pendingFailure returns the armed retry rung's last classified failure,
// wrapped as ErrDumpFailed so a snapshot RPC maps it to Unavailable rather
// than exposing the adapter's raw path error, or nil when no rung owns the
// next attempt (WR-02). It is read under the scheduler mutex, like every
// other ladder access.
func (s *scheduler) pendingFailure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retryDelay > 0 && s.lastErr != nil {
		return fmt.Errorf("%w: %w", ErrDumpFailed, s.lastErr)
	}
	return nil
}
