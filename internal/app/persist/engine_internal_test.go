package persist

import (
	"errors"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// errInvSave is the invariant suite's injected Save failure.
var errInvSave = errors.New("invariant suite injected save failure")

// invDumps is the minimal in-memory port.DumpStore of the white-box
// confirmWaiters suite: Save succeeds except on the configured 1-based
// index, Load is the empty start, and the snapshot methods are unused —
// the suite drives runDump/confirmWaiters/failWaiters directly, no Run
// goroutine and no real I/O.
type invDumps struct {
	saves  int
	failOn int // 1-based Save index to fail; 0 never fails.
}

// Save counts the dump and fails on the configured index.
func (d *invDumps) Save([]byte) error {
	d.saves++
	if d.failOn > 0 && d.saves == d.failOn {
		return errInvSave
	}
	return nil
}

// Load is the empty start; this suite never restarts.
func (*invDumps) Load(func([]byte) error) ([]byte, bool) { return nil, false }

// CreateSnapshot is unused by the invariant suite.
func (*invDumps) CreateSnapshot(string) error { return nil }

// ReadSnapshot is unused by the invariant suite.
func (*invDumps) ReadSnapshot(string) ([]byte, error) { return nil, ErrDumpFailed }

// newInvariantEngine returns an engine over an empty store and the given
// fake port, driven synchronously by the white-box tests.
func newInvariantEngine(dumps *invDumps) *Engine {
	store := storage.NewStore(0x0304, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	return New(store, dumps, Config{})
}

// invVersions are the fixed cover/waiter versions of the invariant suite:
// invCoverL's record orders strictly below any version packed from
// invAboveL, by the big-endian l dominance of the 16-byte order.
const (
	invCoverL = int64(1000)
	invAboveL = int64(2000)
)

// registerAboveCover installs the provably-unreachable state directly
// (white-box — no public path can produce it, which is exactly why the
// guard exists): a waiter whose version sits above the engine's cover,
// with the engine dirty so the next runDump captures both.
func registerAboveCover(eng *Engine, ver version.RecordVersion) *waiter {
	w := &waiter{ver: ver, done: make(chan struct{})}
	eng.mu.Lock()
	eng.lastLocal = version.PackVersion(invCoverL, 0, 0x0304)
	eng.waiters = []*waiter{w}
	eng.dirty = true
	eng.mu.Unlock()
	return w
}

// TestConfirmWaiters_aboveCoverInvariant pins the above-cover guard and
// the invariant assertion beside it (D-15 P3-WR-01): the guard — provably
// unreachable through await's registration order, kept as the documented
// belt — holds an above-cover waiter for the next epoch instead of
// resolving it under a dump that does not cover it (a false nil
// confirmation), the held waiter is provably resolved later (nil under a
// covering dump, ErrDumpFailed under a failed one — stranded never), and
// the invariant assertion never re-queues an already-resolved waiter, the
// double-resolution the next fan-out would turn into a double close.
func TestConfirmWaiters_aboveCoverInvariant(t *testing.T) {
	t.Parallel()

	above := version.PackVersion(invAboveL, 5, 0x0305)

	t.Run("the guard keeps an unresolved above-cover waiter and a covering dump resolves it nil", func(t *testing.T) {
		t.Parallel()
		eng := newInvariantEngine(&invDumps{})
		w := registerAboveCover(eng, above)
		err := eng.runDump()
		if err != nil {
			t.Fatalf("first runDump() error = %v, want nil", err)
		}
		select {
		case <-w.done:
			t.Fatal("the above-cover waiter resolved under a dump that does not cover it — a false confirmation")
		default:
		}
		eng.mu.Lock()
		held := len(eng.waiters)
		eng.mu.Unlock()
		if held != 1 {
			t.Fatalf("the guard re-queued %d waiters, want the 1 above-cover waiter held for the next epoch", held)
		}

		// The next epoch's cover dominates the held version: resolved nil.
		eng.mu.Lock()
		eng.lastLocal = above
		eng.dirty = true
		eng.mu.Unlock()
		err = eng.runDump()
		if err != nil {
			t.Fatalf("second runDump() error = %v, want nil", err)
		}
		select {
		case <-w.done:
		default:
			t.Fatal("the held waiter stayed unresolved after a covering dump — the guard strands it")
		}
		if w.err != nil {
			t.Fatalf("the held waiter resolved with %v, want nil", w.err)
		}
	})

	t.Run("a failed epoch resolves the held waiter with ErrDumpFailed — never stranded", func(t *testing.T) {
		t.Parallel()
		eng := newInvariantEngine(&invDumps{failOn: 2})
		w := registerAboveCover(eng, above)
		err := eng.runDump()
		if err != nil {
			t.Fatalf("first runDump() error = %v, want nil", err)
		}
		eng.mu.Lock()
		eng.dirty = true
		eng.mu.Unlock()
		err = eng.runDump()
		if !errors.Is(err, errInvSave) {
			t.Fatalf("second runDump() error = %v, want the injected save failure", err)
		}
		select {
		case <-w.done:
		default:
			t.Fatal("the held waiter stayed unresolved after a failed dump — failWaiters fans out to it")
		}
		if !errors.Is(w.err, ErrDumpFailed) {
			t.Fatalf("the held waiter resolved with %v, want ErrDumpFailed", w.err)
		}
	})

	t.Run("the invariant assertion never re-queues an already-resolved waiter", func(t *testing.T) {
		t.Parallel()
		eng := newInvariantEngine(&invDumps{})
		resolved := &waiter{ver: above, done: make(chan struct{})}
		resolved.err = ErrDumpFailed
		close(resolved.done)
		eng.confirmWaiters([]*waiter{resolved}, version.PackVersion(invCoverL, 0, 0x0304))
		eng.mu.Lock()
		requeued := len(eng.waiters)
		eng.mu.Unlock()
		if requeued != 0 {
			t.Fatalf("the invariant assertion re-queued %d resolved waiters, want 0 — the next fan-out would double-close it", requeued)
		}
		// The hazard made loud: had the resolved waiter been re-queued,
		// this fan-out would close its done channel a second time and
		// panic the test binary — a double resolution can never hide.
		eng.failWaiters(nil)
	})
}
