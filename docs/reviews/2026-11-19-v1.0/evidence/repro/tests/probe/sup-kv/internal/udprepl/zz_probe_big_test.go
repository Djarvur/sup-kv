package udprepl_test

// AUDIT PROBE (scratch probe copy only): replicate a value at the documented
// envelope (value_max_bytes = 61440) over real UDP sockets, then a record
// whose key pushes the datagram past the 65507-byte UDP payload limit.

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestZZProbe_maxValueOverRealUDP(t *testing.T) {
	cfg := scenarioTimers(func(msg string) { t.Logf("core/adapter log: %.160s", msg) })
	a, b, _ := startScenarioPair(t, cfg, cfg)
	waitFor(t, scenarioDiscoveryDeadline, "mutual discovery", func() bool {
		return a.adapter.KnownPeers() == 1 && b.adapter.KnownPeers() == 1
	})
	for _, tc := range []struct {
		name   string
		keyLen int
		valLen int
	}{
		{"1KiB value", 16, 1024},
		{"32KiB value", 16, 32 * 1024},
		{"61440-byte value (value_max_bytes), short key", 16, 61440},
		{"61440-byte value + 5000-byte key (datagram > 65507)", 5000, 61440},
	} {
		key := "big/" + strings.Repeat("k", tc.keyLen)
		val := bytes.Repeat([]byte{0xAB}, tc.valLen)
		start := time.Now()
		ver := scenarioWrite(t, a, key, val)
		ok := false
		for time.Since(start) < 3*time.Second {
			if scenarioServes(b, key, ver, val) {
				ok = true
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Logf("PROBE5 %-55s replicated=%v after %v; A unacked=%d retransmitted=%d", tc.name, ok, time.Since(start).Round(time.Millisecond), a.adapter.UnackedStats().Count, a.adapter.Retransmissions())
	}
}
