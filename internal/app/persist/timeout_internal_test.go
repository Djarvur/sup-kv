package persist

// White-box counterpart of the PERSIST-08 deadline suite: the timeout
// branch's waiter removal is an engine-internal fact — the slice is not
// observable through the public API — so this test lives in package persist.
// The _internal_test.go suffix is the documented testpackage exemption.

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// waiterRemovedNode is the internal proof's store identity.
const waiterRemovedNode = uint16(0x0306)

// stuckDumpPort parks every SaveStream forever until release closes — the
// permanently hung dumper whose waiter slice the test watches.
type stuckDumpPort struct {
	startOnce sync.Once
	started   chan struct{}
	release   chan struct{}
}

// newStuckDumpPort returns a port whose first SaveStream parks until release
// closes.
func newStuckDumpPort() *stuckDumpPort {
	return &stuckDumpPort{started: make(chan struct{}), release: make(chan struct{})}
}

// SaveStream parks before the fill: the dump stands forever from the
// engine's point of view.
func (p *stuckDumpPort) SaveStream(fill func(io.Writer) error) error {
	p.startOnce.Do(func() { close(p.started) })
	<-p.release
	return fill(io.Discard)
}

// Load reports the empty start.
func (*stuckDumpPort) Load(_ time.Time, _ func([]byte) error) port.LoadResult {
	return port.LoadResult{}
}

// CreateSnapshot and ReadSnapshot mirror the empty in-memory state.
func (*stuckDumpPort) CreateSnapshot(string) error { return ErrDumpFailed }

func (*stuckDumpPort) ReadSnapshot(string) ([]byte, error) { return nil, ErrDumpFailed }

// startStuckDumper launches Run in the bubble and returns a drain func.
func startStuckDumper(t *testing.T, eng *Engine) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	return func() {
		cancel()
		err := <-done
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
	}
}

// waiterLen inspects the engine's registered-waiter slice under the engine
// mutex.
func waiterLen(eng *Engine) int {
	eng.mu.Lock()
	defer eng.mu.Unlock()
	return len(eng.waiters)
}

// TestEngine_waiterRemoved proves the timeout branch's memory bound: under a
// permanently hung dumper every timed-out write removes its own waiter, so
// the slice is empty after each timeout however many writes time out, the
// counter names exactly how many were unconfirmed, and a later successful
// dump finds no waiter to double-close.
func TestEngine_waiterRemoved(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		clk := clock.NewClock(time.Now, 0, 0, time.Minute)
		store := storage.NewStore(waiterRemovedNode, clk, storage.Config{})
		dumps := newStuckDumpPort()
		eng := New(store, dumps, Config{ConfirmTimeout: time.Second})
		defer startStuckDumper(t, eng)()
		defer func() { close(dumps.release) }()

		keys := []string{"waiter-a", "waiter-b", "waiter-c"}
		for i, key := range keys {
			done := make(chan error, 1)
			go func() {
				_, _, err := eng.Set(key, []byte("value"))
				done <- err
			}()
			if i == 0 {
				<-dumps.started // the dumper is parked forever
			}
			var err error
			select {
			case err = <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("Set(%q) did not return at the confirmation timeout", key)
			}
			if !errors.Is(err, ErrUnconfirmed) {
				t.Fatalf("Set(%q) error = %v, want ErrUnconfirmed", key, err)
			}
			if got := waiterLen(eng); got != 0 {
				t.Fatalf("after the timeout of %q the waiter slice holds %d entries, want 0 — the timeout must remove its waiter", key, got)
			}
		}
		if got := eng.DumpStats().Unconfirmed; got != uint64(len(keys)) {
			t.Errorf("DumpStats().Unconfirmed = %d, want %d", got, len(keys))
		}
	})
}
