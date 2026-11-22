package persist

import (
	"slices"
	"sync"
	"time"
)

// waiter is one write's confirmation ticket: done is closed exactly once,
// and err — written before the close — carries the resolution. The ticket
// needs no version: runDump captures the list before taking the Records
// snapshot, so every captured waiter is covered by that dump (see
// confirmWaiters). Channel close over [sync.Cond] because only channels
// broadcast an error to every waiter (D-03, D-05).
//
// claimed is the single-claim protocol's guard (PERSIST-08, T-5-26): every
// resolver — a dump's success or failure fan-out, Run's exit sweep, or the
// awaiting caller's timeout — sets it under the confirmer mutex before
// writing err and closing done, and a resolver that finds it already set
// leaves both untouched. Capture alone never claims: a dump that merely took
// the waiter into its captured slice never owns it, so a dump hanging after
// capture cannot keep the caller's timeout from resolving the write.
type waiter struct {
	done    chan struct{}
	err     error
	claimed bool
}

// confirmer is the confirmation-accounting role (D-10, PERSIST-11): it owns
// the write waiters, the dirty flag and its oldest epoch, the changed nudge,
// the closed flag, runDone and the flush request channel, all behind one
// mutex. Every blocking write registers and waits through await; the dump
// scheduler resolves a completed dump through confirmWaiters or failWaiters,
// and Run's exit resolves the stranded ones through exit. Because every
// entry point takes the one mutex and releases it before touching a channel
// or I/O, no cross-role call can deadlock (T-5-30).
//
// The role holds a pointer to the engine's one unexported dumpStats atomic
// holder — the same pointer the scheduler writes: the timeout branch
// increments stats.unconfirmed with an atomic Add, so DumpStats stays a
// race-free value copy and the two writers never see two copies of a counter.
type confirmer struct {
	mu      sync.Mutex
	waiters []*waiter
	dirty   bool
	oldest  time.Time

	changed chan struct{}
	flushCh chan flushRequest
	runDone chan struct{}
	closed  bool

	cfg   Config
	stats *dumpStats
}

// newConfirmer returns the confirmation role wired to the engine's normalized
// config (its clock, confirmation deadline and timer seam) and to the one
// dump-stats holder the scheduler also writes.
func newConfirmer(cfg Config, stats *dumpStats) *confirmer {
	return &confirmer{
		changed: make(chan struct{}, 1),
		flushCh: make(chan flushRequest, 1),
		runDone: make(chan struct{}),
		cfg:     cfg,
		stats:   stats,
	}
}

// nudge wakes an idle scheduler through the non-blocking changed send: the
// channel has capacity one and the send never blocks, so repeated nudges
// collapse into one wake-up.
func (c *confirmer) nudge() {
	select {
	case c.changed <- struct{}{}:
	default:
	}
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
// hung dumper cannot grow memory, unlocks, counts the unconfirmed outcome on
// the shared stats holder and returns. The record stays installed; the
// engine stays dirty exactly as the capture proof promises. Only
// ErrUnconfirmed is returned — never nil, never ErrDumpFailed.
func (c *confirmer) await() error {
	w := &waiter{done: make(chan struct{})}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClosed
	}
	if !c.dirty {
		c.oldest = c.cfg.Now()
	}
	c.dirty = true
	c.waiters = append(c.waiters, w)
	c.mu.Unlock()
	c.nudge()
	timeout := c.cfg.After(c.cfg.ConfirmTimeout)
	select {
	case <-w.done:
		return w.err
	case <-timeout:
	}
	c.mu.Lock()
	if w.claimed {
		c.mu.Unlock()
		<-w.done // the resolver owns the close; return its answer.
		return w.err
	}
	w.claimed = true
	w.err = ErrUnconfirmed
	c.waiters = slices.DeleteFunc(c.waiters, func(candidate *waiter) bool { return candidate == w })
	c.mu.Unlock()
	c.stats.unconfirmed.Add(1)
	return ErrUnconfirmed
}

// markDirty marks the engine dirty from a fresh epoch without registering a
// waiter and wakes an idle dumper through the non-blocking changed nudge:
// the restore replay's store writes are unconfirmed until the final flush's
// covering dump — the cover discipline of await without the per-write
// blocking — and the mid-replay error path needs the same wake-up (Phase-4
// review New-1). The nudge lives here, not at each caller, so every dirty
// path sends exactly one signal.
func (c *confirmer) markDirty() {
	c.mu.Lock()
	if !c.dirty {
		c.oldest = c.cfg.Now()
	}
	c.dirty = true
	c.mu.Unlock()
	c.nudge()
}

// startDirty marks the engine dirty from at without a nudge: Open runs it
// before Run exists, so there is no idle scheduler to wake — the first
// write's markDirty nudges.
func (c *confirmer) startDirty(at time.Time) {
	c.mu.Lock()
	c.dirty = true
	c.oldest = at
	c.mu.Unlock()
}

// epoch returns the dirty state and the clean-to-dirty transition instant
// the scheduler arms the deadline from, under one mutex hold.
func (c *confirmer) epoch() (dirty bool, oldest time.Time) {
	c.mu.Lock()
	dirty, oldest = c.dirty, c.oldest
	c.mu.Unlock()
	return dirty, oldest
}

// isDirty reports whether the state differs from the last completed dump:
// the forced-dump decision of flushDump and nothing else.
func (c *confirmer) isDirty() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dirty
}

// isClosed reports whether Run's exit sweep has run: every write and flush
// entering after it answers ErrClosed instead of registering.
func (c *confirmer) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// capture takes the current waiter list and clears the dirty epoch under one
// mutex hold: runDump calls it BEFORE taking the Records snapshot, so every
// captured waiter's record is inside the dump about to be saved.
func (c *confirmer) capture() []*waiter {
	c.mu.Lock()
	captured := c.waiters
	c.waiters = nil
	c.dirty = false
	c.mu.Unlock()
	return captured
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
func (c *confirmer) confirmWaiters(captured []*waiter) {
	c.mu.Lock()
	claimed := make([]*waiter, 0, len(captured))
	for _, w := range captured {
		if w.claimed {
			continue
		}
		w.claimed = true
		w.err = nil
		claimed = append(claimed, w)
	}
	c.mu.Unlock()
	for _, w := range claimed {
		close(w.done)
	}
}

// failWaiters resolves every current waiter — the captured list plus any
// registered during the failed dump — with ErrDumpFailed, and re-dirties
// the engine from a fresh epoch so the surviving records reach the next
// successful dump. Waiters the caller's timeout already claimed are skipped:
// the timeout owns their resolution and close (PERSIST-08, T-5-26).
func (c *confirmer) failWaiters(captured []*waiter) {
	c.mu.Lock()
	stranded := make([]*waiter, 0, len(captured)+len(c.waiters))
	stranded = append(stranded, captured...)
	stranded = append(stranded, c.waiters...)
	c.waiters = nil
	c.dirty = true
	c.oldest = c.cfg.Now()
	claimed := make([]*waiter, 0, len(stranded))
	for _, w := range stranded {
		if w.claimed {
			continue
		}
		w.claimed = true
		w.err = ErrDumpFailed
		claimed = append(claimed, w)
	}
	c.mu.Unlock()
	for _, w := range claimed {
		close(w.done)
	}
}

// sweepClosed is Run's exit sweep: under the same mutex hold the write
// paths check, closed flips before the waiters are taken, so no writer can
// register after the sweep and block forever; every stranded waiter then
// resolves with ErrClosed, as does every flush request already handed to
// the exiting Run — the reply channels are buffered, so the sends never
// block (Pitfall 4). A waiter the caller's timeout already claimed is
// skipped — the timeout owns its close (PERSIST-08, T-5-26).
func (c *confirmer) sweepClosed() {
	c.mu.Lock()
	c.closed = true
	stranded := c.waiters
	c.waiters = nil
	claimed := make([]*waiter, 0, len(stranded))
	for _, w := range stranded {
		if w.claimed {
			continue
		}
		w.claimed = true
		w.err = ErrClosed
		claimed = append(claimed, w)
	}
	c.mu.Unlock()
	for _, w := range claimed {
		close(w.done)
	}
	for {
		select {
		case req := <-c.flushCh:
			req.reply <- ErrClosed
		default:
			return
		}
	}
}

// exit is Run's exit path: the sweep resolves every stranded waiter and
// answers every queued flush request, then runDone closes. Program order
// here is the contract flush relies on: runDone closing after the sweep
// means a reply observed once runDone is closed is exactly one the sweep
// answered, so flush can tell the answered from the stranded apart.
func (c *confirmer) exit() {
	c.sweepClosed()
	close(c.runDone)
}
