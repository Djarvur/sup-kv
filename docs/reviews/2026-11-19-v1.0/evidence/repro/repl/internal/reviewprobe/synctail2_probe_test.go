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

// Anti-entropy starvation: the receiver lacks three keys (head, middle, tail
// of the sorted keyspace). The sender broadcasts its full listing five times
// (startup + 4 timer rounds). Which keys does anti-entropy repair?
func TestProbe_antiEntropyTailStarvation(t *testing.T) {
	const keys = 10_000
	missing := map[int]bool{5: true, 5000: true, 9995: true}
	key := func(i int) string { return fmt.Sprintf("svc/v1/key-%05d", i) }
	mk := func(id uint16, skip map[int]bool) *storage.Store {
		st := storage.NewStore(id, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
		val := make([]byte, 1024)
		for i := range keys {
			if skip[i] {
				continue
			}
			_, _, _ = st.Merge(key(i), storage.Record{Version: version.PackVersion(1730000000000, 0, 9), Data: val})
		}
		return st
	}
	rStore, sStore := mk(2, missing), mk(1, nil)
	sink, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer sink.Close()
	r, err := New("127.0.0.1:0", sink.LocalAddr().String(), 2, memStore{rStore}, repl.Config{SyncInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.conn.SetReadBuffer(212992); err != nil { // Linux rmem_default
		t.Fatal(err)
	}
	var sendErrs atomic.Int64
	s, err := New("127.0.0.1:0", r.conn.LocalAddr().String(), 1, memStore{sStore}, repl.Config{SyncInterval: time.Second, Log: func(string) { sendErrs.Add(1) }})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = r.Run(ctx) }()
	time.Sleep(200 * time.Millisecond)
	go func() { _ = s.Run(ctx) }()
	time.Sleep(4700 * time.Millisecond)
	rounds := s.SentByType().Sync / 304
	t.Logf("sender-side send errors logged: %d; receiver discards: %+v", sendErrs.Load(), r.Discards())
	for _, i := range []int{5, 5000, 9995} {
		_, err := rStore.Get(key(i))
		t.Logf("after %d full listing rounds: key #%d repaired=%v", rounds, i, err == nil)
		if err != nil {
			t.Errorf("key #%d is still missing on the receiver after %d anti-entropy rounds", i, rounds)
		}
	}
}
