package reviewprobe

import (
	"context"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// The production adapter socket (New, copied verbatim) on EPHEMERAL loopback
// ports: can it send the DataReplication of a legal 20 KiB / 60 KiB value?
func TestProbe_productionSocketSendsLargeRecord(t *testing.T) {
	rx, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer rx.Close()
	var mu sync.Mutex
	var logs []string
	a, err := New("127.0.0.1:0", rx.LocalAddr().String(), 1, nopStore{}, repl.Config{
		Log: func(msg string) { mu.Lock(); logs = append(logs, msg); mu.Unlock() },
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _ = a.Run(ctx) }()

	got := map[string]int{}
	var gmu sync.Mutex
	go func() {
		buf := make([]byte, 65507)
		for {
			n, _, rerr := rx.ReadFromUDP(buf)
			if rerr != nil {
				return
			}
			var env replproto.Envelope
			var body replproto.Body
			if proto.Unmarshal(buf[:n], &env) == nil && proto.Unmarshal(env.GetBody(), &body) == nil {
				if dr := body.GetDataReplication(); dr != nil {
					gmu.Lock()
					got[dr.GetKey()] = len(dr.GetData())
					gmu.Unlock()
				}
			}
		}
	}()
	ver := version.PackVersion(time.Now().UnixMilli(), 0, 1)
	for key, size := range map[string]int{"small-1KiB": 1 << 10, "mid-20KiB": 20 << 10, "max-60KiB": 61440} {
		if err := a.NotifyLocalWrite(key, storage.Record{Version: ver, Data: make([]byte, size)}); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done
	gmu.Lock()
	defer gmu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	t.Logf("GOOS=%s received DataReplication: %v", runtime.GOOS, got)
	for _, l := range logs {
		if strings.Contains(l, "send to") {
			t.Logf("adapter log: %s", l)
		}
	}
	if len(got) != 3 {
		t.Errorf("only %d of 3 legal records left the production socket; unacked=%d sent-by-type(data)=%d (counted although the send failed)", len(got), a.UnackedStats().Count, a.SentByType().Data)
	}
}
