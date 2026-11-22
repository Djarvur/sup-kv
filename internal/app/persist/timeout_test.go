package persist_test

// The PERSIST-08 deadline suite: every blocking engine call — a write's
// confirmation, Flush, Snapshot and Restore — is bounded by a confirmation
// timeout, and a hung dump resolves the caller with the explicit
// ErrUnconfirmed outcome instead of parking it forever. The whole suite runs
// on virtual time through a port whose SaveStream never completes until the
// test releases it, so "exactly at the timeout" is an exact assertion rather
// than a timing tolerance (Pitfall 18: no real I/O inside a bubble).

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The deadline-suite constants: the fake node's identity, the documented
// default confirmation timeout (D-07) the default-config test observes, the
// timeout guard a blocked call gets beyond its own deadline, and the key and
// value the timed-out write uses.
const (
	timeoutNodeID     = uint16(0x0305)
	defaultTimeout    = 10 * time.Second
	timeoutGuard      = time.Second
	timeoutWriteKey   = "timeout-key"
	timeoutWriteValue = "timeout-value"
)

// errNoTimeoutSnapshot is the fake's missing-snapshot error.
var errNoTimeoutSnapshot = errors.New("timeout fake: no snapshot installed")

// timeoutDumpPort is the PERSIST-08 fake: every SaveStream parks on release
// until the test closes it, so a dump stands — the hung-fsync stand-in —
// while the caller's confirmation deadline runs. Completed saves are
// recorded byte-for-byte. No filesystem: real I/O inside a synctest bubble
// is not durably blocking (Pitfall 18).
type timeoutDumpPort struct {
	startOnce   sync.Once
	started     chan struct{}
	release     chan struct{}
	releaseOnce sync.Once

	mu        sync.Mutex
	saves     [][]byte
	snapshots map[string][]byte
	installed map[string][]byte
	copies    int
}

// newTimeoutDumpPort returns a fake whose every SaveStream parks until
// releaseAll closes the release channel.
func newTimeoutDumpPort() *timeoutDumpPort {
	return &timeoutDumpPort{
		started:   make(chan struct{}),
		release:   make(chan struct{}),
		snapshots: make(map[string][]byte),
		installed: make(map[string][]byte),
	}
}

// SaveStream parks before running fill: the dump stands while the
// confirmation deadline runs, and the fill (the encoded generation) only
// happens once the test releases the port.
func (p *timeoutDumpPort) SaveStream(fill func(io.Writer) error) error {
	p.startOnce.Do(func() { close(p.started) })
	<-p.release
	var buf bytes.Buffer
	err := fill(&buf)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.saves = append(p.saves, bytes.Clone(buf.Bytes()))
	p.mu.Unlock()
	return nil
}

// Load reports the empty start: the deadline suite never restarts.
func (*timeoutDumpPort) Load(_ time.Time, _ func([]byte) error) port.LoadResult {
	return port.LoadResult{}
}

// CreateSnapshot stores the latest completed save under the name — the
// adapter's in-memory stand-in; the timeout proofs assert it is never
// reached.
func (p *timeoutDumpPort) CreateSnapshot(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.copies++
	if len(p.saves) == 0 {
		return persist.ErrDumpFailed
	}
	p.snapshots[name] = bytes.Clone(p.saves[len(p.saves)-1])
	return nil
}

// ReadSnapshot serves an installed fixture, or the pinned missing error.
func (p *timeoutDumpPort) ReadSnapshot(name string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if data, ok := p.installed[name]; ok {
		return bytes.Clone(data), nil
	}
	data, ok := p.snapshots[name]
	if !ok {
		return nil, errNoTimeoutSnapshot
	}
	return bytes.Clone(data), nil
}

// waitStarted blocks until the first SaveStream has entered — the dump
// stands.
func (p *timeoutDumpPort) waitStarted() { <-p.started }

// releaseAll lets the standing dump (and every later one) complete; safe to
// call more than once.
func (p *timeoutDumpPort) releaseAll() { p.releaseOnce.Do(func() { close(p.release) }) }

// saveCount returns the number of completed saves.
func (p *timeoutDumpPort) saveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.saves)
}

// copyCount returns the number of CreateSnapshot attempts.
func (p *timeoutDumpPort) copyCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.copies
}

// installSnapshot supplies a ReadSnapshot fixture for the Restore proof.
func (p *timeoutDumpPort) installSnapshot(name string, data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.installed[name] = bytes.Clone(data)
}

// lastSave returns the latest completed generation.
func (p *timeoutDumpPort) lastSave() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.saves) == 0 {
		return nil
	}
	return slices.Clone(p.saves[len(p.saves)-1])
}

// newTimeoutEngine builds an engine over an empty store and a hanging port,
// with the caller's confirmation-timeout configuration.
func newTimeoutEngine(cfg persist.Config) (*persist.Engine, *timeoutDumpPort) {
	clk := clock.NewClock(time.Now, 0, 0, time.Minute)
	store := storage.NewStore(timeoutNodeID, clk, storage.Config{})
	dumps := newTimeoutDumpPort()
	return persist.New(store, dumps, cfg), dumps
}

// TestEngine_confirmationTimeout proves the PERSIST-08 write bound: with a
// port whose SaveStream never completes, Set returns ErrUnconfirmed at
// exactly the confirmation timeout — the documented 10 s default for a
// zero Config, and a supplied positive value when one is configured — while
// the record stays installed in the store and the timeout is counted.
func TestEngine_confirmationTimeout(t *testing.T) {
	t.Parallel()

	t.Run("the default is 10 seconds", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			assertWriteTimeout(t, persist.Config{}, defaultTimeout)
		})
	})
	t.Run("a supplied positive timeout wins", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			assertWriteTimeout(t, persist.Config{ConfirmTimeout: 3 * time.Second}, 3*time.Second)
		})
	})
}

// assertWriteTimeout drives one timed-out Set at a hung dump and asserts the
// exact virtual instant, the sentinel, the installed record and the counter.
func assertWriteTimeout(t *testing.T, cfg persist.Config, want time.Duration) {
	t.Helper()
	eng, dumps := newTimeoutEngine(cfg)
	stop := startDumper(t, eng)
	defer stop()
	defer dumps.releaseAll()

	start := time.Now()
	deadline := start.Add(want)
	setDone := make(chan error, 1)
	go func() {
		_, _, err := eng.Set(timeoutWriteKey, []byte(timeoutWriteValue))
		setDone <- err
	}()
	dumps.waitStarted() // a dump stands and never completes

	// Not one virtual nanosecond before the deadline.
	synctest.Sleep(time.Until(deadline) - time.Millisecond)
	select {
	case err := <-setDone:
		t.Fatalf("Set() returned %v before the confirmation deadline %v", err, deadline)
	default:
	}

	synctest.Sleep(time.Millisecond)
	var err error
	select {
	case err = <-setDone:
	case <-time.After(timeoutGuard):
		t.Fatalf("Set() did not return at the confirmation deadline %v — the wait is unbounded", deadline)
	}
	if !errors.Is(err, persist.ErrUnconfirmed) {
		t.Fatalf("Set() error = %v, want ErrUnconfirmed", err)
	}
	if now := time.Now(); !now.Equal(deadline) {
		t.Errorf("Set() returned at %v, want exactly %v", now, deadline)
	}

	rec, getErr := eng.Get(timeoutWriteKey)
	if getErr != nil {
		t.Fatalf("Get(%q) after the timeout error = %v, want the installed record", timeoutWriteKey, getErr)
	}
	if !bytes.Equal(rec.Data, []byte(timeoutWriteValue)) {
		t.Errorf("Get(%q) data = %q, want %q", timeoutWriteKey, rec.Data, timeoutWriteValue)
	}
	if got := eng.DumpStats().Unconfirmed; got != 1 {
		t.Errorf("DumpStats().Unconfirmed = %d, want 1", got)
	}
}

// TestEngine_timeoutThenDumpSuccess proves the single-claim protocol's late
// half (T-5-26): after the write times out, the standing dump completes
// successfully — it finds the waiter already claimed by the timeout, so it
// resolves nothing and double-closes nothing — and the record is still in
// that dump. A second write afterwards is confirmed normally, proving the
// engine is not wedged.
func TestEngine_timeoutThenDumpSuccess(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newTimeoutEngine(persist.Config{ConfirmTimeout: 2 * time.Second})
		defer startDumper(t, eng)()
		defer dumps.releaseAll()

		firstDone := make(chan error, 1)
		go func() {
			_, _, err := eng.Set(timeoutWriteKey, []byte(timeoutWriteValue))
			firstDone <- err
		}()
		dumps.waitStarted()
		synctest.Sleep(2*time.Second + time.Millisecond)
		var firstErr error
		select {
		case firstErr = <-firstDone:
		case <-time.After(timeoutGuard):
			t.Fatal("the first Set() did not return at its confirmation timeout")
		}
		if !errors.Is(firstErr, persist.ErrUnconfirmed) {
			t.Fatalf("first Set() error = %v, want ErrUnconfirmed", firstErr)
		}

		// The standing dump now completes: the timeout already claimed and
		// removed the waiter, so this must not panic on a second close and
		// must still save the record the waiter had installed.
		dumps.releaseAll()
		synctest.Wait()

		if got := dumps.saveCount(); got != 1 {
			t.Fatalf("completed saves after the late success = %d, want 1", got)
		}
		records, _, decodeErr := persist.Decode(dumps.lastSave(), storage.ValueMaxBytes)
		if decodeErr != nil {
			t.Fatalf("Decode(the late dump) error = %v", decodeErr)
		}
		if _, ok := records[timeoutWriteKey]; !ok {
			t.Errorf("the late dump lacks %q — a timed-out record must still be dumped", timeoutWriteKey)
		}

		secondDone := make(chan error, 1)
		go func() {
			_, _, err := eng.Set("second-key", []byte("second-value"))
			secondDone <- err
		}()
		var secondErr error
		select {
		case secondErr = <-secondDone:
		case <-time.After(2 * time.Second):
			t.Fatal("the second Set() did not return after the engine recovered")
		}
		if secondErr != nil {
			t.Errorf("second Set() error = %v, want nil (the engine recovered)", secondErr)
		}
		if got := eng.DumpStats().Unconfirmed; got != 1 {
			t.Errorf("DumpStats().Unconfirmed = %d, want 1 (only the first write timed out)", got)
		}
	})
}

// TestFlush_timeout proves Flush and FlushTimeout are bounded: Flush uses
// the confirmation timeout, an explicit FlushTimeout uses its own deadline,
// and both return the wrapped ErrUnconfirmed when the dumper never answers.
func TestFlush_timeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newTimeoutEngine(persist.Config{ConfirmTimeout: 2 * time.Second})
		stop := startDumper(t, eng)
		defer stop()
		defer dumps.releaseAll()

		hungWrite(t, eng, dumps, 2*time.Second)

		start := time.Now()
		flushDone := make(chan error, 1)
		go func() { flushDone <- eng.Flush() }()
		var err error
		select {
		case err = <-flushDone:
		case <-time.After(3 * time.Second):
			t.Fatal("Flush() did not return within its confirmation timeout")
		}
		if !errors.Is(err, persist.ErrUnconfirmed) {
			t.Fatalf("Flush() error = %v, want ErrUnconfirmed", err)
		}
		if want := start.Add(2 * time.Second); !time.Now().Equal(want) {
			t.Errorf("Flush() returned at %v, want exactly %v", time.Now(), want)
		}

		explicit := time.Now()
		go func() { flushDone <- eng.FlushTimeout(500 * time.Millisecond) }()
		select {
		case err = <-flushDone:
		case <-time.After(2 * time.Second):
			t.Fatal("FlushTimeout(500ms) did not return within its deadline")
		}
		if !errors.Is(err, persist.ErrUnconfirmed) {
			t.Fatalf("FlushTimeout(500ms) error = %v, want ErrUnconfirmed", err)
		}
		if want := explicit.Add(500 * time.Millisecond); !time.Now().Equal(want) {
			t.Errorf("FlushTimeout(500ms) returned at %v, want exactly %v", time.Now(), want)
		}
	})
}

// TestSnapshot_timeout proves Snapshot is bounded by the confirmation
// timeout and copies nothing when the forced flush cannot complete: state
// that is not confirmed on disk never becomes a recovery point.
func TestSnapshot_timeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newTimeoutEngine(persist.Config{ConfirmTimeout: 2 * time.Second})
		stop := startDumper(t, eng)
		defer stop()
		defer dumps.releaseAll()

		hungWrite(t, eng, dumps, 2*time.Second)
		start := time.Now()
		snapDone := make(chan error, 1)
		go func() { snapDone <- eng.Snapshot("timeout-snap") }()
		var err error
		select {
		case err = <-snapDone:
		case <-time.After(3 * time.Second):
			t.Fatal("Snapshot() did not return within the confirmation timeout")
		}
		if !errors.Is(err, persist.ErrUnconfirmed) {
			t.Fatalf("Snapshot() error = %v, want ErrUnconfirmed", err)
		}
		if want := start.Add(2 * time.Second); !time.Now().Equal(want) {
			t.Errorf("Snapshot() returned at %v, want exactly %v", time.Now(), want)
		}
		if got := dumps.copyCount(); got != 0 {
			t.Errorf("CreateSnapshot calls = %d, want 0 — an unconfirmed state must not be copied", got)
		}
	})
}

// TestRestore_timeoutReturnsPartial proves Restore keeps its partial-list
// contract under the timeout: every replayed record is returned even though
// the confirming flush expired, and the error is distinguishable — it
// matches ErrUnconfirmed and never ErrDumpFailed.
func TestRestore_timeoutReturnsPartial(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newTimeoutEngine(persist.Config{ConfirmTimeout: 2 * time.Second})
		stop := startDumper(t, eng)
		defer stop()
		defer dumps.releaseAll()

		dumps.installSnapshot("snap", timeoutSnapshotFixture(t, "snap-key", "snap-value"))

		type restoreResult struct {
			records []persist.RestoredRecord
			err     error
		}
		restoreDone := make(chan restoreResult, 1)
		go func() {
			records, err := eng.Restore("snap")
			restoreDone <- restoreResult{records: records, err: err}
		}()
		var result restoreResult
		select {
		case result = <-restoreDone:
		case <-time.After(3 * time.Second):
			t.Fatal("Restore() did not return within the confirmation timeout")
		}
		if !errors.Is(result.err, persist.ErrUnconfirmed) {
			t.Fatalf("Restore() error = %v, want ErrUnconfirmed", result.err)
		}
		if errors.Is(result.err, persist.ErrDumpFailed) {
			t.Error("Restore() error matches ErrDumpFailed — the unconfirmed class must stay distinguishable")
		}
		if len(result.records) != 1 || result.records[0].Key != "snap-key" {
			t.Fatalf("Restore() partial list = %+v, want exactly the replayed snap-key", result.records)
		}
	})
}

// hungWrite installs one record and lets its confirmation time out — the
// prerequisite that leaves the dumper standing for the flush and snapshot
// proofs.
func hungWrite(t *testing.T, eng *persist.Engine, dumps *timeoutDumpPort, timeout time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		_, _, err := eng.Set(timeoutWriteKey, []byte(timeoutWriteValue))
		done <- err
	}()
	dumps.waitStarted()
	synctest.Sleep(timeout + time.Millisecond)
	var err error
	select {
	case err = <-done:
	case <-time.After(timeoutGuard):
		t.Fatal("the prerequisite write did not time out")
	}
	if !errors.Is(err, persist.ErrUnconfirmed) {
		t.Fatalf("the prerequisite write error = %v, want ErrUnconfirmed", err)
	}
}

// timeoutSnapshotFixture encodes one live record as a dump generation the Restore
// proof can replay.
func timeoutSnapshotFixture(t *testing.T, key, value string) []byte {
	t.Helper()
	rec := storage.Record{Data: []byte(value)}
	rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, false)
	dump, err := persist.Encode(map[string]storage.Record{key: rec}, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("Encode(snapshot fixture) error = %v", err)
	}
	return dump
}
