package udprepl_test

// The real-socket in-process half of TEST-04 (plan 06-06): two-node
// convergence and the adversarial-input scenarios — hand-crafted bad-CRC
// datagrams, protocol-version mismatch and duplicate node_id — classified
// by their Discards counters over real UDP sockets. These classes involve
// the real socket path (kernel delivery, encode/decode, handle shaping)
// that the deterministic simulator cannot express; the existing shared-port
// smoke test stays untouched as the own-echo assertion home.
//
// Every two-node scenario here rides the ASYMMETRIC-PORT topology
// (06-RESEARCH Pattern 5, the empirically load-bearing finding): node A
// binds P1 and broadcasts to 127.255.255.255:P2, node B binds P2 and
// broadcasts to 127.255.255.255:P1 — every broadcast and every unicast
// (an Ack above all, the exact path the shared-port topology cannot carry)
// crosses to exactly the peer. A two-node convergence assertion on the
// shared-port smoke topology would hang: Linux delivers unicast to a port
// shared by SO_REUSEADDR sockets to the last-bound socket only, so the
// peer's Ack self-delivers and is discarded as own echo.

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// Scenario test constants: the two wire node ids, the shrunken timers every
// scenario runs with (the exec-harness values, 06-RESEARCH Pattern 6 —
// discovery and sync rounds complete inside CI deadlines while everything
// stays far from them), the capability probe's wait, and the two state
// deadlines with their timer-derived bounds.
const (
	scenarioNodeA = uint16(41)
	scenarioNodeB = uint16(43)

	scenarioHelloInterval = 100 * time.Millisecond
	scenarioSyncInterval  = 200 * time.Millisecond

	scenarioProbeWait = 300 * time.Millisecond

	// scenarioDiscoveryDeadline bounds broadcast-Hello mutual discovery: a
	// couple of hello rounds at the shrunken interval with generous real-time
	// headroom for a loaded CI runner.
	scenarioDiscoveryDeadline = 5 * time.Second

	// scenarioConvergeDeadline bounds one full write cycle after discovery,
	// derived from the timers rather than wall-clock precision: one
	// broadcast RTT for the DataReplication plus one unicast RTT for the
	// Ack, each sub-millisecond on loopback, plus the pending mirror's 250ms
	// drained-row retention window before a drained map observably empties —
	// ten hello intervals cover every contribution with margin.
	scenarioConvergeDeadline = 10 * scenarioHelloInterval

	scenarioKeyA = "scenario/key-a"
	scenarioKeyB = "scenario/key-b"
)

// The two replicated payloads.
var (
	scenarioValA = []byte("scenario-value-a-over-real-udp")
	scenarioValB = []byte("scenario-value-b-over-real-udp")
)

// scenarioProbeBroadcast pre-flights the one platform capability every
// scenario in this file needs — a datagram sent to the loopback broadcast
// address reaches the socket bound to the destination port — and skips
// loudly when it is missing, naming the probed address and the platform
// reason: loopback broadcast fan-out is a Linux-only kernel behavior and
// darwin's lo0 carries no BROADCAST flag (D-12). No build tag hides the
// scenarios — the capability gate is the darwin story; protocol correctness
// under its absence stays proven by the deterministic simulator (plans
// 04-02..04-05) and the core suites.
func scenarioProbeBroadcast(t *testing.T, probePort int) {
	t.Helper()
	rx, err := smokeSocket(fmt.Sprintf("0.0.0.0:%d", probePort))
	if err != nil {
		t.Fatalf("probe rx bind: %v", err)
	}
	defer func() { _ = rx.Close() }()
	tx, err := smokeSocket("0.0.0.0:0")
	if err != nil {
		t.Fatalf("probe tx bind: %v", err)
	}
	defer func() { _ = tx.Close() }()
	dst := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: probePort}
	_, err = tx.WriteToUDP([]byte("sup-kv scenario probe"), dst)
	if err != nil {
		t.Fatalf("probe send to %s: %v", dst, err)
	}
	err = rx.SetReadDeadline(time.Now().Add(scenarioProbeWait))
	if err != nil {
		t.Fatalf("probe read deadline: %v", err)
	}
	buf := make([]byte, 64)
	for {
		n, _, readErr := rx.ReadFromUDP(buf)
		if readErr != nil {
			t.Skipf(
				"loopback broadcast to %s did not reach the bound socket within %v — platform lacks broadcast fan-out (loopback broadcast fan-out is a Linux-only kernel behavior; darwin lo0 carries no BROADCAST flag — D-12); protocol correctness stays proven by the deterministic simulator (plans 04-02..04-05) and the core suites",
				dst,
				scenarioProbeWait,
			)
		}
		if n > 0 {
			return
		}
	}
}

// scenarioTimers returns the shrunken-timer Config every scenario runs with
// (the exec-harness values, 06-RESEARCH Pattern 6) around the given Log
// callback.
func scenarioTimers(log func(msg string)) repl.Config {
	return repl.Config{
		HelloInterval: scenarioHelloInterval,
		SyncInterval:  scenarioSyncInterval,
		Log:           log,
	}
}

// scenarioNode is one node of the scenario pair: its adapter and the store
// behind it, directly readable by the test goroutine (store reads are
// lock-free snapshots; adapter reads ride the post-event atomics).
type scenarioNode struct {
	adapter *udprepl.Adapter
	store   *storage.Store
}

// newScenarioNode builds one node — a store over a real wall clock (writes
// stamp real versions; the fixed-clock store fixtures would freeze version
// time across the two nodes' real convergence window) wrapped in the
// smoke-test StateStore adapter, plus the node's adapter bound to bindPort
// with its broadcast aimed at crossPort, running its real event loop.
func newScenarioNode(t *testing.T, nodeID uint16, bindPort, crossPort int, cfg repl.Config) *scenarioNode {
	t.Helper()
	store := storage.NewStore(nodeID, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	adapter, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", bindPort),
		fmt.Sprintf("127.255.255.255:%d", crossPort),
		nodeID, smokeStore{store: store}, cfg,
	)
	if err != nil {
		t.Fatalf("node %d adapter New (bind %d, cross %d): %v", nodeID, bindPort, crossPort, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = adapter.Run(ctx) }()
	return &scenarioNode{adapter: adapter, store: store}
}

// startScenarioPair probes the loopback-broadcast capability (the loud
// darwin skip, D-12) and starts the asymmetric two-node topology
// (06-RESEARCH Pattern 5): node A binds portA broadcasting to
// 127.255.255.255:portB, node B binds portB broadcasting to
// 127.255.255.255:portA. Every broadcast and every unicast crosses to
// exactly the peer and no self-echo occurs, so the unicast Ack path — the
// path the shared-port smoke topology cannot carry — is deterministic
// here. Returns the pair plus node A's bound port, the target the
// adversarial datagrams of the discard scenarios aim at.
func startScenarioPair(t *testing.T, cfgA, cfgB repl.Config) (a, b *scenarioNode, portA int) {
	t.Helper()
	pA, pB := freeUDPPort(t), freeUDPPort(t)
	scenarioProbeBroadcast(t, pA)
	return newScenarioNode(t, scenarioNodeA, pA, pB, cfgA),
		newScenarioNode(t, scenarioNodeB, pB, pA, cfgB), pA
}

// scenarioWrite stores key=val on node n and announces it through the
// author's explicit notification (D-05), returning the stored version.
func scenarioWrite(t *testing.T, n *scenarioNode, key string, val []byte) version.RecordVersion {
	t.Helper()
	ver, _, err := n.store.Set(key, val)
	if err != nil {
		t.Fatalf("node store Set(%q): %v", key, err)
	}
	rec, err := n.store.Get(key)
	if err != nil {
		t.Fatalf("node store Get(%q): %v", key, err)
	}
	err = n.adapter.NotifyLocalWrite(key, rec)
	if err != nil {
		t.Fatalf("NotifyLocalWrite(%q): %v", key, err)
	}
	return ver
}

// scenarioServes reports whether n's store serves key under exactly ver,
// byte-equal data.
func scenarioServes(n *scenarioNode, key string, ver version.RecordVersion, val []byte) bool {
	got, err := n.store.Get(key)
	return err == nil && got.Version == ver && bytes.Equal(got.Data, val)
}

// TestScenario_convergence is the plan's tracer: two real adapters on the
// asymmetric topology discover each other from broadcast Hellos alone, one
// local write on each node crosses to the peer byte-equal, and BOTH pending
// mirrors drain to empty — the unicast Ack path, the exact path the
// shared-port topology cannot carry (research Pattern 5 / Pitfall 1). The
// write cycle is bounded by the timer-derived scenarioConvergeDeadline,
// never wall-clock precision: broadcast RTT plus unicast Ack RTT plus the
// drained-row retention window, each datagram sub-millisecond on loopback.
func TestScenario_convergence(t *testing.T) {
	t.Parallel()
	cfg := scenarioTimers(func(msg string) { t.Logf("%s", msg) })
	a, b, _ := startScenarioPair(t, cfg, cfg)

	// Mutual discovery from broadcast Hellos alone (REPL-01 — zero config).
	waitFor(t, scenarioDiscoveryDeadline, "mutual discovery (both known-peer counts == 1)", func() bool {
		return a.adapter.KnownPeers() == 1 && b.adapter.KnownPeers() == 1
	})

	// One local write per node, each announced by its author (D-05).
	verA := scenarioWrite(t, a, scenarioKeyA, scenarioValA)
	verB := scenarioWrite(t, b, scenarioKeyB, scenarioValB)

	// Convergence: each store serves the peer's record byte-equal under the
	// peer's exact version.
	waitFor(t, scenarioConvergeDeadline, "each store serving the peer's record byte-equal", func() bool {
		return scenarioServes(b, scenarioKeyA, verA, scenarioValA) &&
			scenarioServes(a, scenarioKeyB, verB, scenarioValB)
	})

	// The Ack path: both pending mirrors drain to empty — the drained rows'
	// retention window included — proving the unicast Acks crossed.
	waitFor(t, scenarioConvergeDeadline, "both pending mirrors to drain to empty (the unicast Ack path)", func() bool {
		return len(a.adapter.PendingCopy()) == 0 && len(b.adapter.PendingCopy()) == 0
	})
}
