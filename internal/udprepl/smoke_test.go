package udprepl_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// Smoke-test constants (docs/IDEA.md quickstart shape, D-08): the two node
// ids sharing one host, the capability probe's wait, the three wait
// deadlines with their poll cadence (real time — real sockets), and the
// margin kept ahead of the test's own hard deadline so a timeout fails with
// this test's message rather than the runner's kill.
const (
	smokeNodeA = uint16(1)
	smokeNodeB = uint16(2)

	probeWait          = 300 * time.Millisecond
	discoveryDeadline  = 5 * time.Second
	visibilityDeadline = 5 * time.Second
	echoDeadline       = 5 * time.Second
	pollEvery          = 10 * time.Millisecond
	hardDeadlineMargin = 250 * time.Millisecond

	smokeKey = "smoke/key"
)

// smokeValue is the one replicated payload.
var smokeValue = []byte("smoke-value-over-real-udp")

// freeUDPPort picks one free UDP port: bind a throwaway wildcard socket to
// :0, read its port, close it.
func freeUDPPort(t *testing.T) int {
	t.Helper()
	var lc net.ListenConfig
	conn, err := lc.ListenPacket(t.Context(), "udp4", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("bind throwaway port-picking socket: %v", err)
	}
	defer func() { _ = conn.Close() }()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

// smokeSocket binds a wildcard udp4 socket with SO_REUSEADDR and
// SO_BROADCAST set before bind — the adapter's exact socket recipe (research
// Pattern 3), replicated here because the adapter keeps it unexported and
// the external smoke test's capability probe needs the same same-host
// topology.
func smokeSocket(bindAddr string) (*net.UDPConn, error) {
	var opErr error
	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if opErr != nil {
					return
				}
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			})
			if err != nil {
				return fmt.Errorf("socket control: %w", err)
			}
			if opErr != nil {
				return fmt.Errorf("socket options: %w", opErr)
			}
			return nil
		},
	}
	packetConn, err := lc.ListenPacket(context.Background(), "udp4", bindAddr)
	if err != nil {
		return nil, fmt.Errorf("bind udp4 %s: %w", bindAddr, err)
	}
	return packetConn.(*net.UDPConn), nil
}

// broadcastReachable proves same-host broadcast delivery on port toward
// bcast with the adapter's socket recipe: two sockets share the port, one
// sends a probe datagram, the other must receive it within probeWait
// (research Pattern 3's probed matrix; Pitfall 3's pre-flight). A bind or
// send fault is an error, not evidence of unreachability.
func broadcastReachable(port int, bcast *net.UDPAddr) (bool, error) {
	tx, err := smokeSocket(fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Close() }()
	rx, err := smokeSocket(fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return false, err
	}
	defer func() { _ = rx.Close() }()
	_, err = tx.WriteToUDP([]byte("sup-kv smoke probe"), bcast)
	if err != nil {
		return false, fmt.Errorf("send probe to %s: %w", bcast, err)
	}
	err = rx.SetReadDeadline(time.Now().Add(probeWait))
	if err != nil {
		return false, fmt.Errorf("set probe read deadline: %w", err)
	}
	buf := make([]byte, 64)
	for {
		n, _, err := rx.ReadFromUDP(buf)
		if err != nil {
			return false, nil // the deadline elapsed — unreachable, not broken.
		}
		if n > 0 {
			return true, nil
		}
	}
}

// subnetBroadcast returns the subnet broadcast address of the first
// broadcast-capable non-loopback IPv4 interface, or nil when none exists —
// macOS lo0 carries no BROADCAST flag, so the loopback probe fails there and
// a real interface's subnet broadcast is the one remaining same-host
// candidate (research Pitfall 3).
func subnetBroadcast(port int) *net.UDPAddr {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagBroadcast == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			broadcast := make(net.IP, net.IPv4len)
			for i := range ip4 {
				broadcast[i] = ip4[i] | ^ipNet.Mask[i]
			}
			return &net.UDPAddr{IP: broadcast, Port: port}
		}
	}
	return nil
}

// smokeStore adapts *storage.Store to repl.StateStore (the retransmit-test
// precedent): remote merges join through Merge. Dumps stay out of the smoke
// scope — the persist seam has its own plan-04-06 proof; D-08 keeps this
// test about real protobuf datagrams, real sockets and real addressing.
type smokeStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s smokeStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Records returns the full snapshot, tombstones included.
func (s smokeStore) Records() map[string]storage.Record { return s.store.Records() }

// waitFor polls cond every pollInterval until it holds or the deadline —
// the earlier of timeout and the test's own hard deadline minus a margin —
// and fails with what on expiry.
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	if d, ok := t.Deadline(); ok {
		if hard := d.Add(-hardDeadlineMargin); hard.Before(deadline) {
			deadline = hard
		}
	}
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(pollEvery)
	}
	t.Fatalf("timed out after %v waiting for %s", timeout, what)
}

// TestSmoke_twoNodesUDP is D-08's one real-socket smoke test: two nodes on
// one host discover each other purely via broadcast Hello over genuine UDP
// with no configuration beyond the broadcast address (REPL-01), one local
// write on node A becomes byte-equal visible on node B, and node A's own
// broadcast echo really loops back — counted in the OwnEcho discard class
// (REPL-09, probed Linux behavior). The pre-flight capability probe tries
// the loopback broadcast first, then one real-interface subnet broadcast,
// and skips loudly naming both addresses when neither reaches the sharing
// peer (research Pitfall 3 — loopback broadcast fan-out is a Linux-only
// kernel behavior; macOS lo0 lacks the BROADCAST flag).
//
// The test deliberately NEVER asserts Ack or DataRequest or any unicast
// delivery between the two same-host nodes: unicast to a port shared by
// SO_REUSEADDR sockets is delivered to exactly ONE socket, the kernel's
// choice (research Pitfall 2, probed) — those paths are simulator territory
// (plans 04-02..04-05), and the adapter's doc comment documents the same
// limitation.
func TestSmoke_twoNodesUDP(t *testing.T) {
	t.Parallel()
	port := freeUDPPort(t)

	loopback := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: port}
	broadcast := loopback
	reachable, err := broadcastReachable(port, loopback)
	if err != nil {
		t.Fatalf("loopback broadcast probe setup: %v", err)
	}
	if !reachable {
		subnet := subnetBroadcast(port)
		if subnet == nil {
			t.Skipf(
				"same-host UDP broadcast unavailable: %s did not reach the sharing socket and no broadcast-capable non-loopback IPv4 interface exists (loopback broadcast fan-out is a Linux-only kernel behavior; macOS lo0 carries no BROADCAST flag — research Pitfall 3); protocol correctness stays proven by the deterministic simulator (plans 04-02..04-05)",
				loopback,
			)
		}
		reachable, err = broadcastReachable(port, subnet)
		if err != nil {
			t.Fatalf("subnet broadcast probe setup: %v", err)
		}
		if !reachable {
			t.Skipf(
				"same-host UDP broadcast unavailable: neither %s nor %s reached the sharing socket within %v (platform lacks broadcast fan-out to SO_REUSEADDR sockets — research Pitfall 3); protocol correctness stays proven by the deterministic simulator (plans 04-02..04-05)",
				loopback,
				subnet,
				probeWait,
			)
		}
		broadcast = subnet
	}

	bind := fmt.Sprintf("0.0.0.0:%d", port)
	cfg := repl.Config{Log: func(msg string) { t.Logf("%s", msg) }}
	storeA := storage.NewStore(smokeNodeA, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	storeB := storage.NewStore(smokeNodeB, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	adapterA, err := udprepl.New(bind, broadcast.String(), smokeNodeA, smokeStore{storeA}, cfg)
	if err != nil {
		t.Fatalf("node A adapter: %v", err)
	}
	adapterB, err := udprepl.New(bind, broadcast.String(), smokeNodeB, smokeStore{storeB}, cfg)
	if err != nil {
		t.Fatalf("node B adapter: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = adapterA.Run(ctx) }()
	go func() { _ = adapterB.Run(ctx) }()

	// Discovery: each node learns exactly one peer from broadcast Hellos
	// alone (REPL-01 — zero-config).
	waitFor(t, discoveryDeadline, "mutual discovery (both known-peer counts == 1)", func() bool {
		return adapterA.KnownPeers() == 1 && adapterB.KnownPeers() == 1
	})

	// One local write on node A, announced by its author (D-05): the
	// broadcast DataReplication must surface it on node B byte-equal.
	ver, _, err := storeA.Set(smokeKey, smokeValue)
	if err != nil {
		t.Fatalf("node A store Set(%q): %v", smokeKey, err)
	}
	rec, err := storeA.Get(smokeKey)
	if err != nil {
		t.Fatalf("node A store Get(%q): %v", smokeKey, err)
	}
	adapterA.NotifyLocalWrite(smokeKey, rec)
	waitFor(t, visibilityDeadline, "node B serving the record byte-equal", func() bool {
		got, err := storeB.Get(smokeKey)
		return err == nil && got.Version == ver && bytes.Equal(got.Data, smokeValue)
	})

	// The sender's own broadcast echo is real (probed on Linux): node A
	// counted its own datagrams in OwnEcho (REPL-09).
	waitFor(t, echoDeadline, "node A's own broadcast echo counted in OwnEcho (>= 1)", func() bool {
		return adapterA.Discards().OwnEcho >= 1
	})
}
