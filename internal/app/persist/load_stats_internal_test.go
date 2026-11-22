package persist

// White-box test for the markDirty nudge (D-04, Phase-4 review New-1): the
// non-blocking e.changed send lives in markDirty itself, so every dirty-mark
// path — the restore replay's mid-error branch included — wakes an idle
// dumper without each caller remembering to nudge. markDirty is unexported,
// and the _internal_test.go suffix is the documented testpackage exemption.

import (
	"bytes"
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

// nudgeNode is the store's stamping identity in this file's one engine.
const nudgeNode = uint16(0x0305)

// nudgeDumpPort is the in-memory DumpStore of the nudge test: it counts
// completed saves and never touches the filesystem, so the whole scenario
// runs inside a synctest bubble (Pitfall 6).
type nudgeDumpPort struct {
	mu    sync.Mutex
	saves int
}

// SaveStream runs fill into memory and counts the completed generation.
func (p *nudgeDumpPort) SaveStream(fill func(io.Writer) error) error {
	var buf bytes.Buffer
	err := fill(&buf)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.saves++
	p.mu.Unlock()
	return nil
}

// Load reports the empty start: the nudge test never restarts.
func (*nudgeDumpPort) Load(time.Time, func([]byte) error) port.LoadResult {
	return port.LoadResult{}
}

// CreateSnapshot accepts the request without copying anything.
func (*nudgeDumpPort) CreateSnapshot(string) error { return nil }

// ReadSnapshot reports a missing snapshot.
func (*nudgeDumpPort) ReadSnapshot(string) ([]byte, error) { return nil, ErrDumpFailed }

// count returns the completed-save count.
func (p *nudgeDumpPort) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.saves
}

// TestEngine_markDirtyNudge proves the nudge is the markDirty path: one
// markDirty on an idle clean engine produces exactly one dump at the
// debounce instant — nothing earlier — and the engine stays quiet afterwards
// with no further activity.
func TestEngine_markDirtyNudge(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		store := storage.NewStore(nudgeNode, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
		dumps := &nudgeDumpPort{}
		eng := New(store, dumps, Config{})

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- eng.Run(ctx) }()
		defer func() {
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Errorf("Run() error = %v, want context.Canceled", err)
			}
		}()

		eng.markDirty()
		synctest.Sleep(defaultDebounce - time.Millisecond)
		synctest.Wait()
		if got := dumps.count(); got != 0 {
			t.Fatalf("the idle engine dumped %d time(s) before Debounce, want 0", got)
		}

		synctest.Sleep(2 * time.Millisecond)
		synctest.Wait()
		if got := dumps.count(); got != 1 {
			t.Fatalf("one markDirty on an idle clean engine ran %d dump(s) at Debounce, want exactly 1", got)
		}

		synctest.Sleep(defaultDebounce)
		synctest.Wait()
		if got := dumps.count(); got != 1 {
			t.Fatalf("the clean engine ran %d dump(s) with no further activity, want 1", got)
		}
	})
}
