package reviewprobe

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// versionsOnlyStore answers Records() with a shallow map (no value copy) —
// a stand-in for a StateStore that offers a cheap key->version view.
type versionsOnlyStore struct{ s *storage.Store }

func (v versionsOnlyStore) ApplyRemote(k string, r storage.Record) ([]storage.Conflict, time.Duration, error) {
	return v.s.Merge(k, r)
}

var shallowCache map[string]storage.Record

var shallowCalls atomic.Int64

func (v versionsOnlyStore) Records() map[string]storage.Record { shallowCalls.Add(1); return shallowCache }

// Control experiment for the tail starvation: same sockets, same burst, same
// 212 992 B receive buffer — only the per-chunk full-store deep copy is gone.
func TestProbe_tailStarvationControl(t *testing.T) {
	const keys = 10_000
	key := func(i int) string { return fmt.Sprintf("svc/v1/key-%05d", i) }
	mk := func(id uint16, skip int) *storage.Store {
		st := storage.NewStore(id, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
		val := make([]byte, 1024)
		for i := range keys {
			if i == skip {
				continue
			}
			_, _, _ = st.Merge(key(i), storage.Record{Version: version.PackVersion(1730000000000, 0, 9), Data: val})
		}
		return st
	}
	rStore, sStore := mk(2, 9995), mk(1, -1)
	shallowCache = rStore.Records()
	sink, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer sink.Close()
	r, err := New("127.0.0.1:0", sink.LocalAddr().String(), 2, versionsOnlyStore{rStore}, repl.Config{SyncInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.conn.SetReadBuffer(212992); err != nil {
		t.Fatal(err)
	}
	s, err := New("127.0.0.1:0", r.conn.LocalAddr().String(), 1, memStore{sStore}, repl.Config{SyncInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = r.Run(ctx) }()
	time.Sleep(200 * time.Millisecond)
	base := shallowCalls.Load()
	go func() { _ = s.Run(ctx) }()
	time.Sleep(1500 * time.Millisecond)
	t.Logf("sender sent %d chunks; receiver processed %d; receiver sent %d DataRequests; sender sent %d DataReplication", s.SentByType().Sync, shallowCalls.Load()-base, r.SentByType().DataRequest, s.SentByType().Data)
	_, err = rStore.Get(key(9995))
	t.Logf("without the per-chunk deep copy, after ONE listing round: tail key #9995 repaired=%v", err == nil)
	if err != nil {
		t.Errorf("tail key still missing")
	}
}
