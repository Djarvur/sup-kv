package persist_test

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// Blocking-proof constants: the node identity and drift tolerance of the
// blocking engine, the virtual window the not-returned assertion waits
// through expecting the timeout, the release-wait bound, and the key/value
// the blocked Set writes.
const (
	blockingNodeID    = uint16(0x0303)
	blockingDriftWarn = time.Minute
	notReturnedWindow = 200 * time.Millisecond
	releaseWait       = time.Second
	blockingKey       = "blocked-key"
	blockingValue     = "blocked-value"
)

// blockingDumpPort is the D-07 blocking fake (TEST-03): Save parks on
// release until the test lets it complete — while parked, the dump stands
// and every Set waiter must stay blocked, the engine's dump path being
// synchronous end to end. parked closes exactly once when the first Save
// arrives, so the test waits deterministically for the dump to stand
// before asserting; attempts counts Save entries and saves the completed
// ones. No filesystem — real I/O inside a synctest bubble is not durably
// blocking (Pitfall 6).
type blockingDumpPort struct {
	parkOnce sync.Once
	parked   chan struct{}
	release  chan struct{}
	mu       sync.Mutex
	attempts int
	saves    int
}

// newBlockingDumpPort returns a fake whose first Save parks until release
// is closed.
func newBlockingDumpPort() *blockingDumpPort {
	return &blockingDumpPort{parked: make(chan struct{}), release: make(chan struct{})}
}

// SaveStream parks before running fill: the dump stands — the fill has not
// run yet — while the test holds release closed, so every Set waiter must
// stay blocked.
func (b *blockingDumpPort) SaveStream(fill func(io.Writer) error) error {
	b.mu.Lock()
	b.attempts++
	b.mu.Unlock()
	b.parkOnce.Do(func() { close(b.parked) })
	<-b.release
	var buf bytes.Buffer
	err := fill(&buf)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.saves++
	b.mu.Unlock()
	return nil
}

// Save is the byte-slice convenience over SaveStream; the blocking proof
// drives the engine's streamed path.
func (b *blockingDumpPort) Save(dump []byte) error {
	return b.SaveStream(func(w io.Writer) error {
		_, err := w.Write(dump)
		return err
	})
}

// Load is the empty start; the blocking proof never restarts.
func (*blockingDumpPort) Load(_ func([]byte) error) ([]byte, bool) { return nil, false }

// CreateSnapshot and ReadSnapshot mirror the fake port's in-memory
// semantics — nothing is on disk to copy or read.
func (*blockingDumpPort) CreateSnapshot(string) error { return persist.ErrDumpFailed }

func (*blockingDumpPort) ReadSnapshot(string) ([]byte, error) { return nil, persist.ErrDumpFailed }

// waitParked blocks until a Save has parked — the dump stands.
func (b *blockingDumpPort) waitParked() { <-b.parked }

// completedSaves returns the number of saves the release let complete.
func (b *blockingDumpPort) completedSaves() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.saves
}

// TestEngine_blockingSaveHoldsSet is D-07's blocking proof (TEST-03): while
// a Save stands, Set does not return — the dump path is synchronous end to
// end, the write-confirmation contract PERS-04 riding the completing dump —
// and releasing the standing Save lets the parked Set return with its
// confirmation (nil, the write is on disk).
func TestEngine_blockingSaveHoldsSet(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		clk := clock.NewClock(time.Now, 0, 0, blockingDriftWarn)
		store := storage.NewStore(blockingNodeID, clk, storage.Config{})
		dumps := newBlockingDumpPort()
		eng := persist.New(store, dumps, persist.Config{})
		defer startDumper(t, eng)()

		setDone := make(chan error, 1)
		go func() {
			_, _, err := eng.Set(blockingKey, []byte(blockingValue))
			setDone <- err
		}()

		dumps.waitParked() // a Save stands: the dump is in flight
		select {
		case err := <-setDone:
			t.Fatalf("Set() returned %v while the dump stood — Set must not return until the covering dump completes (D-07)", err)
		case <-time.After(notReturnedWindow):
			// the expected outcome: the window fired first, Set is still blocked
		}

		close(dumps.release) // the standing dump completes
		var err error
		select {
		case err = <-setDone:
		case <-time.After(releaseWait):
			t.Fatal("Set() did not return after the standing dump was released")
		}
		if err != nil {
			t.Errorf("Set() error after the dump completed = %v, want nil (the write is confirmed)", err)
		}
		synctest.Wait()
		if got := dumps.completedSaves(); got != 1 {
			t.Errorf("completed saves = %d, want 1", got)
		}
	})
}
