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

type countingStore struct {
	s       *storage.Store
	records atomic.Int64
}

func (c *countingStore) ApplyRemote(k string, r storage.Record) ([]storage.Conflict, time.Duration, error) {
	return c.s.Merge(k, r)
}

func (c *countingStore) Records() map[string]storage.Record {
	c.records.Add(1)
	return c.s.Records()
}

// Two production adapters (copied code) on ephemeral loopback ports, both
// holding the SAME 10k x 1 KiB records. The sender's startup listing is one
// unpaced burst; how many of its chunks does the receiver actually process?
func TestProbe_listingBurstVsReceiver(t *testing.T) {
	for _, rcvbuf := range []int{0, 212992} { // 0 = OS default; 212992 = Linux rmem_default
		t.Run(fmt.Sprintf("rcvbuf=%d", rcvbuf), func(t *testing.T) {
			const keys = 10_000
			now := time.Now
			mk := func(id uint16) *countingStore {
				st := storage.NewStore(id, clock.NewClock(now, 0, 0, time.Minute), storage.Config{})
				val := make([]byte, 1024)
				for i := range keys {
					_, _, _ = st.Merge(fmt.Sprintf("svc/v1/key-%05d", i), storage.Record{Version: version.PackVersion(1730000000000, 0, 9), Data: val})
				}
				return &countingStore{s: st}
			}
			sink, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}) // receiver's own broadcasts go nowhere useful
			defer sink.Close()
			rStore, sStore := mk(2), mk(1)
			r, err := New("127.0.0.1:0", sink.LocalAddr().String(), 2, rStore, repl.Config{SyncInterval: time.Hour})
			if err != nil {
				t.Fatal(err)
			}
			if rcvbuf > 0 {
				if err := r.conn.SetReadBuffer(rcvbuf); err != nil {
					t.Fatal(err)
				}
			}
			s, err := New("127.0.0.1:0", r.conn.LocalAddr().String(), 1, sStore, repl.Config{SyncInterval: time.Hour})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = r.Run(ctx) }()
			time.Sleep(200 * time.Millisecond) // receiver's own startup listing done
			base := rStore.records.Load()
			start := time.Now()
			go func() { _ = s.Run(ctx) }() // sender: Hello + full listing burst
			time.Sleep(3 * time.Second)
			sent := s.SentByType().Sync
			processed := rStore.records.Load() - base
			t.Logf("sender emitted %d DataSync chunks in one burst; receiver processed %d (%.0f%%) within %v", sent, processed, 100*float64(processed)/float64(sent), time.Since(start).Round(time.Millisecond))
			if uint64(processed) < sent {
				t.Errorf("%d of %d listing chunks were dropped before the receiver's event loop saw them — always the tail of the sorted keyspace", sent-uint64(processed), sent)
			}
		})
	}
}
