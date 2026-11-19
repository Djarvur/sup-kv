package udprepl

// AUDIT PROBE (scratch probe copy only): a fuzz harness for the datagram
// decoder + core classifier — the trust-boundary parser the shipped suite
// never feeds raw malformed bytes.

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

type zzStore struct{ s *storage.Store }

func (z zzStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return z.s.Merge(key, rec)
}
func (z zzStore) Records() map[string]storage.Record { return z.s.Records() }

func FuzzZZDecode(f *testing.F) {
	run := [16]byte{9, 9, 9}
	ver := version.PackVersion(1_700_000_000_000, 7, 3)
	for _, msg := range []port.Message{
		port.Hello{},
		port.DataReplication{Key: "k", Record: storage.Record{Version: ver, Data: []byte("v")}},
		port.DataReplication{Key: "k", Record: storage.Record{Version: ver, Deleted: true}},
		port.Ack{Key: "k", Version: ver},
		port.DataRequest{Key: "k"},
		port.DataSync{Entries: []port.SyncEntry{{Key: "a", Version: ver}, {Key: "b", Version: ver}}},
	} {
		d, err := encodeDatagram(3, run, msg)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(d)
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		in := decode(data)
		if in.Status == port.Decoded && in.Msg == nil {
			t.Fatalf("Decoded verdict with nil Msg for %x", data)
		}
		now := func() time.Time { return time.UnixMilli(1_700_000_000_000) }
		st := storage.NewStore(1, clock.NewClock(now, 0, 0, time.Minute), storage.Config{})
		core := repl.New(1, [16]byte{1}, "self", zzStore{st}, repl.Config{})
		var out port.Outbox
		in.Source = "peer"
		core.HandleDatagram(now(), in, &out)
		core.Advance(now(), &out)
		for _, ob := range out.Drain() {
			if _, err := encodeDatagram(1, [16]byte{1}, ob.Msg); err != nil {
				t.Fatalf("core emitted an unencodable message %T: %v", ob.Msg, err)
			}
		}
	})
}
