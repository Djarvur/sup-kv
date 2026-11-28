package persist_test

// AUDIT PROBE (scratch probe copy only): the PERS-04 containment invariant
// under concurrency — when Set returns nil, the most recently COMPLETED save
// must already contain that key at that version or a newer one. This is the
// test proposed for the surviving mutant M21 (snapshot taken before the
// waiter/cover capture).

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

type zzRecordingDumps struct {
	mu   sync.Mutex
	last map[string]storage.Record
}

func (d *zzRecordingDumps) Save(dump []byte) error {
	recs, _, err := persist.Decode(dump)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.last = recs
	d.mu.Unlock()
	return nil
}
func (*zzRecordingDumps) Load(func([]byte) error) ([]byte, bool) { return nil, false }
func (*zzRecordingDumps) CreateSnapshot(string) error            { return nil }
func (*zzRecordingDumps) ReadSnapshot(string) ([]byte, error)    { return nil, nil }

func TestZZProbe_confirmedImpliesContained(t *testing.T) {
	store := storage.NewStore(7, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	// A few thousand filler keys make each snapshot/encode take long enough
	// for concurrent writes to land inside a dump cycle.
	for i := range 3000 {
		_, _, _ = store.Set(fmt.Sprintf("filler/%04d", i), make([]byte, 128))
	}
	dumps := &zzRecordingDumps{}
	eng := persist.New(store, dumps, persist.Config{Debounce: 50 * time.Microsecond, MaxWait: time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = eng.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()

	var wg sync.WaitGroup
	var violations sync.Map
	var count int64
	var cmu sync.Mutex
	for w := range 16 {
		wg.Go(func() {
			for i := range 150 {
				key := fmt.Sprintf("hot/%d/%d", w, i%4)
				ver, _, err := eng.Set(key, []byte(fmt.Sprintf("v-%d-%d", w, i)))
				if err != nil {
					t.Errorf("Set: %v", err)
					return
				}
				dumps.mu.Lock()
				rec, ok := dumps.last[key]
				dumps.mu.Unlock()
				if !ok || rec.Version.Compare(ver) < 0 {
					violations.Store(fmt.Sprintf("%s@%x", key, ver[:8]), true)
					cmu.Lock()
					count++
					cmu.Unlock()
				}
			}
		})
	}
	wg.Wait()
	if count > 0 {
		t.Fatalf("PROBE6 %d confirmed writes were NOT contained in the last completed dump at confirmation time (PERS-04 violated)", count)
	}
	t.Logf("PROBE6 2400 confirmed writes, 0 containment violations")
}
