package repl_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

type nopDump struct{}

func (nopDump) Save([]byte) error                         { return nil }
func (nopDump) Load(func([]byte) error) ([]byte, bool)    { return nil, false }
func (nopDump) CreateSnapshot(string) error               { return nil }
func (nopDump) ReadSnapshot(string) ([]byte, error)       { return nil, nil }

func seedEngine(tb testing.TB, keys, valueBytes int) (*persist.Engine, *storage.Store) {
	tb.Helper()
	store := storage.NewStore(1, clock.NewClock(time.Now, 0, 0, 0), storage.Config{})
	for i := range keys {
		data := make([]byte, valueBytes)
		if _, _, err := store.Set(fmt.Sprintf("svc/v1/key-%05d", i), data); err != nil {
			tb.Fatal(err)
		}
	}
	return persist.New(store, nopDump{}, persist.Config{}), store
}

// One inbound DataRequest datagram, as served by the production core.
func benchDataRequest(b *testing.B, keys, valueBytes int) {
	eng, _ := seedEngine(b, keys, valueBytes)
	core := repl.New(1, repl.NewRunID(), "self", eng, repl.Config{})
	var out port.Outbox
	now := time.Now()
	in := port.InboundDatagram{
		Source: "peer", Status: port.Decoded, ProtoVersion: repl.SupportedProtoVersion, NodeID: 2, RunID: repl.NewRunID(),
		Msg: port.DataRequest{Key: "svc/v1/key-00042"},
	}
	runtime.GC()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		core.HandleDatagram(now, in, &out)
		core.Advance(now, &out)
		out.Drain()
	}
}

func BenchmarkReview_DataRequest_10k_1KiB(b *testing.B)  { benchDataRequest(b, 10_000, 1024) }
func BenchmarkReview_DataRequest_10k_60KiB(b *testing.B) { benchDataRequest(b, 10_000, 60*1024) }
