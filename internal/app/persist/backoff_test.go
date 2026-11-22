package persist_test

// The PERSIST-05 failure-decay suite: the retry ladder's exact virtual
// instants (100 ms doubling to the 5 s cap, reset by a success), the one
// OnDumpFailure callback per failed attempt with the adapter's class, and
// the class counters DumpStats publishes. Every assertion runs in a synctest
// bubble over the in-memory fake port, so the instants are exact and no real
// filesystem I/O can deadlock the bubble (research Pitfall 18).

import (
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The ladder table: seven failed attempts walk the delays 100/200/400/800/
// 1600/3200/5000 ms, so the attempt instants after the first failure are
// these cumulative offsets; the eighth attempt then succeeds. backoffReset
// is the delay a failure after a success starts from again.
var backoffLadder = []time.Duration{
	100 * time.Millisecond,
	300 * time.Millisecond,
	700 * time.Millisecond,
	1500 * time.Millisecond,
	3100 * time.Millisecond,
	6300 * time.Millisecond,
	11300 * time.Millisecond,
}

// backoffCollected is the number of OnDumpFailure callbacks the ladder run
// records: one per failed attempt.
const backoffCollected = 7

// failureLog records the OnDumpFailure callbacks in order — the callback
// must fire exactly once per failed attempt, on the dumper goroutine.
type failureLog struct {
	mu      sync.Mutex
	errs    []error
	classes []port.FailureClass
}

// record is the Config.OnDumpFailure seam.
func (l *failureLog) record(err error, class port.FailureClass) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errs = append(l.errs, err)
	l.classes = append(l.classes, class)
}

// snapshot returns copies of the recorded errors and classes.
func (l *failureLog) snapshot() ([]error, []port.FailureClass) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]error(nil), l.errs...), append([]port.FailureClass(nil), l.classes...)
}

// classedTestError is a save failure carrying a port.FailureClass across the
// seam — the adapter's classedSaveError test twin, so the engine's class
// extraction is proven without importing any errno knowledge into the core.
type classedTestError struct {
	err   error
	class port.FailureClass
}

// Error returns the underlying error's text.
func (e classedTestError) Error() string { return e.err.Error() }

// Unwrap exposes the original error to errors.Is.
func (e classedTestError) Unwrap() error { return e.err }

// FailureClass returns the classified class.
func (e classedTestError) FailureClass() port.FailureClass { return e.class }

// offsetsFrom returns every instant's offset from base — the diagnostic
// form the ladder assertions print on failure.
func offsetsFrom(base time.Time, instants []time.Time) []time.Duration {
	out := make([]time.Duration, len(instants))
	for i, at := range instants {
		out[i] = at.Sub(base)
	}
	return out
}

// newFailureEngine builds an engine over an empty store and the shared fake
// port, running the design-default timing on the bubble clock with the
// failure recorder wired in.
func newFailureEngine(log *failureLog) (*persist.Engine, *fakeDumpPort) {
	dumps := &fakeDumpPort{}
	clk := clock.NewClock(time.Now, 0, 0, time.Minute)
	store := storage.NewStore(synctestNodeID, clk, storage.Config{})
	return persist.New(store, dumps, persist.Config{OnDumpFailure: log.record}), dumps
}

// TestEngine_backoffLadder proves the bounded retry ladder (PERSIST-05,
// T-5-21): a port failing seven attempts walks 100 ms, 200, 400, 800, 1.6 s,
// 3.2 s and the 5 s cap, so the attempts land at exactly +100/+300/+700/
// +1500/+3100/+6300/+11300 ms after the first failure; a write registered
// during the backoff is covered by the next successful attempt, not by an
// extra dump; and a failure after that success restarts the ladder at
// 100 ms.
func TestEngine_backoffLadder(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		log := &failureLog{}
		eng, dumps := newFailureEngine(log)
		dumps.failSaves = backoffCollected
		defer startDumper(t, eng)()

		start := time.Now()
		firstErr := make(chan error, 1)
		go func() {
			_, _, err := eng.Set("ladder", []byte("ladder-value"))
			firstErr <- err
		}()
		err := <-firstErr
		if !errors.Is(err, persist.ErrDumpFailed) {
			t.Fatalf("Set() error = %v, want ErrDumpFailed — the first attempt fails", err)
		}

		// A write registered after the seventh failure waits out the capped
		// 5 s rung: the eighth attempt succeeds and covers it — never an
		// extra dump.
		synctest.Sleep(backoffLadder[len(backoffLadder)-2])
		duringErr := make(chan error, 1)
		go func() {
			_, _, err := eng.Set("during", []byte("during-value"))
			duringErr <- err
		}()

		err = <-duringErr
		if err != nil {
			t.Fatalf("the write registered during the backoff error = %v, want nil from the recovering attempt", err)
		}
		instants := dumps.attemptInstants()
		if len(instants) != len(backoffLadder)+1 {
			t.Fatalf("attempts = %d at %v, want %d (seven failures, one success)",
				len(instants), offsetsFrom(instants[0], instants), len(backoffLadder)+1)
		}
		if want := start.Add(synctestDebounce); !instants[0].Equal(want) {
			t.Errorf("the first attempt landed at %v, want the debounce instant %v", instants[0], want)
		}
		for i, offset := range backoffLadder {
			want := instants[0].Add(offset)
			if !instants[i+1].Equal(want) {
				t.Errorf("attempt %d landed at %v, want %v (offset %v from the first failure)",
					i+2, instants[i+1], want, offset)
			}
		}
		if got, _ := log.snapshot(); len(got) != backoffCollected {
			t.Fatalf("OnDumpFailure fired %d times, want exactly %d — one per failed attempt", len(got), backoffCollected)
		}

		// A success resets the ladder: the next failure retries at 100 ms.
		dumps.mu.Lock()
		dumps.failSaves = 1
		dumps.mu.Unlock()
		resetStart := time.Now()
		resetErr := make(chan error, 1)
		go func() {
			_, _, err := eng.Set("reset", []byte("reset-value"))
			resetErr <- err
		}()
		err = <-resetErr
		if !errors.Is(err, persist.ErrDumpFailed) {
			t.Fatalf("the post-recovery Set() error = %v, want ErrDumpFailed", err)
		}
		synctest.Sleep(backoffLadder[0]) // let the reset ladder's first rung fire
		instants = dumps.attemptInstants()
		if len(instants) != len(backoffLadder)+3 {
			t.Fatalf("attempts after the reset failure = %d, want %d", len(instants), len(backoffLadder)+3)
		}
		failedAt := instants[len(instants)-2]
		retriedAt := instants[len(instants)-1]
		if want := resetStart.Add(synctestDebounce); !failedAt.Equal(want) {
			t.Errorf("the reset failure landed at %v, want the debounce instant %v", failedAt, want)
		}
		if want := failedAt.Add(backoffLadder[0]); !retriedAt.Equal(want) {
			t.Errorf("the retry after the reset landed at %v, want %v (the ladder restarts at 100 ms)",
				retriedAt, want)
		}
	})
}

// TestEngine_dumpFailureClasses proves the class surface: a failed attempt
// invokes OnDumpFailure exactly once with the failing error and
// port.FailureClassOf's class, and DumpStats counts the disk-full and
// permission failures beside the existing total and error totals.
func TestEngine_dumpFailureClasses(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		log := &failureLog{}
		eng, dumps := newFailureEngine(log)
		defer startDumper(t, eng)()

		cases := []struct {
			name  string
			err   error
			class port.FailureClass
		}{
			{"disk full", classedTestError{err: errFakeSave, class: port.FailureDiskFull}, port.FailureDiskFull},
			{"permission", classedTestError{err: errFakeSave, class: port.FailurePermission}, port.FailurePermission},
			{"unclassified", errFakeSave, port.FailureOther},
		}
		for _, tc := range cases {
			dumps.setFailErr(tc.err)
			dumps.mu.Lock()
			dumps.failSaves = 1
			dumps.mu.Unlock()
			setErr := make(chan error, 1)
			go func() {
				_, _, err := eng.Set("class-"+tc.name, []byte("value"))
				setErr <- err
			}()
			err := <-setErr
			if !errors.Is(err, persist.ErrDumpFailed) {
				t.Fatalf("%s: Set() error = %v, want ErrDumpFailed", tc.name, err)
			}
		}

		errs, classes := log.snapshot()
		if len(errs) != len(cases) || len(classes) != len(cases) {
			t.Fatalf("OnDumpFailure fired %d times, want exactly %d — one per failed attempt", len(errs), len(cases))
		}
		for i, tc := range cases {
			if !errors.Is(errs[i], tc.err) {
				t.Errorf("%s: the callback error = %v, want the failing error %v", tc.name, errs[i], tc.err)
			}
			if classes[i] != tc.class {
				t.Errorf("%s: the callback class = %v, want %v", tc.name, classes[i], tc.class)
			}
		}
		stats := eng.DumpStats()
		if stats.Total != uint64(len(cases)) || stats.Errors != uint64(len(cases)) {
			t.Errorf("DumpStats() = %+v, want Total and Errors = %d", stats, len(cases))
		}
		if stats.DiskFull != 1 || stats.Permission != 1 {
			t.Errorf("DumpStats() class counters = (disk_full %d, permission %d), want (1, 1)",
				stats.DiskFull, stats.Permission)
		}
	})
}
