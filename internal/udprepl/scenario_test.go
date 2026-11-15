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
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
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

// Adversarial-scenario constants: the impostor's arbitrary fresh identity —
// a node id no scenario node owns, because claiming a live node's OWN id is
// the own-echo/restart territory the core suites already cover — its fixed
// 16-byte run id, and the wait bound for one crafted datagram's
// classification to surface through the adapter atomics (loopback delivery
// is sub-millisecond; two seconds is the established generous bound).
const (
	scenarioImpostorNode = uint16(47)
	scenarioEffectWait   = 2 * time.Second
)

// scenarioImpostorRun is the impostor's fixed run id — the SAME value from
// both adversarial sockets, because the second duplicate-node_id variant
// keys on remembered run_id equality plus a differing source handle.
var scenarioImpostorRun = [16]byte{0xC3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}

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

// scenarioConvergedPair runs the convergence prologue every scenario shares:
// start the asymmetric pair, wait for mutual discovery, write one record per
// node, wait for each store to serve the peer's record byte-equal, and
// return the pair, node A's bound port and both stores' full snapshots —
// the quiescent converged baseline the adversarial cases fire at.
func scenarioConvergedPair(t *testing.T, cfgA, cfgB repl.Config) (
	a, b *scenarioNode, portA int, snapA, snapB map[string]storage.Record,
) {
	t.Helper()
	a, b, portA = startScenarioPair(t, cfgA, cfgB)
	waitFor(t, scenarioDiscoveryDeadline, "mutual discovery (both known-peer counts == 1)", func() bool {
		return a.adapter.KnownPeers() == 1 && b.adapter.KnownPeers() == 1
	})
	verA := scenarioWrite(t, a, scenarioKeyA, scenarioValA)
	verB := scenarioWrite(t, b, scenarioKeyB, scenarioValB)
	waitFor(t, scenarioConvergeDeadline, "each store serving the peer's record byte-equal", func() bool {
		return scenarioServes(b, scenarioKeyA, verA, scenarioValA) &&
			scenarioServes(a, scenarioKeyB, verB, scenarioValB)
	})
	return a, b, portA, a.store.Records(), b.store.Records()
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
	a, b, _, _, _ := scenarioConvergedPair(t, cfg, cfg)

	// The Ack path: both pending mirrors drain to empty — the drained rows'
	// retention window included — proving the unicast Acks crossed.
	waitFor(t, scenarioConvergeDeadline, "both pending mirrors to drain to empty (the unicast Ack path)", func() bool {
		return len(a.adapter.PendingCopy()) == 0 && len(b.adapter.PendingCopy()) == 0
	})
}

// logCapture records every core Log callback message under a mutex — the
// adapter's event-loop goroutine is the only writer, the test goroutine the
// only reader.
type logCapture struct {
	mu   sync.Mutex
	msgs []string
}

// log appends one message.
func (c *logCapture) log(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, msg)
}

// containing returns every captured message that contains needle.
func (c *logCapture) containing(needle string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for _, msg := range c.msgs {
		if strings.Contains(msg, needle) {
			out = append(out, msg)
		}
	}
	return out
}

// snapshot returns a copy of every captured message.
func (c *logCapture) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.msgs)
}

// scenarioAssertClassExact fails unless the driven discard class rose by
// exactly one while the other four classes stay at their pre-shot values —
// per-class exactness over the real socket path.
func scenarioAssertClassExact(t *testing.T, driven string, got, before repl.DiscardStats) {
	t.Helper()
	want := before
	switch driven {
	case repl.DiscardBadCRC:
		want.BadCRC++
	case repl.DiscardUndecodable:
		want.Undecodable++
	case repl.DiscardBadProtoVersion:
		want.BadProtoVersion++
	case repl.DiscardOwnBroadcast:
		want.OwnEcho++
	case repl.DiscardDuplicateNodeID:
		want.DuplicateNodeID++
	}
	if got != want {
		t.Fatalf("%s shot: discards = %+v, want exactly %+v — only %s incremented", driven, got, want, driven)
	}
}

// scenarioAssertConvergenceIntact fails unless the converged state survives
// an adversarial shot unchanged: both stores' full snapshots equal their
// pre-shot snapshots and mutual discovery holds — node B still knows
// exactly A (the impostor datagrams never reach B's port), node A still
// knows at least B (it may additionally hold the learned impostor peer,
// exactly as the duplicate_node_id case arranges).
func scenarioAssertConvergenceIntact(t *testing.T, a, b *scenarioNode, snapA, snapB map[string]storage.Record) {
	t.Helper()
	if !recordsEqual(snapA, a.store.Records()) {
		t.Fatalf("node A's store changed under adversarial fire: before %d records, now %d",
			len(snapA), len(a.store.Records()))
	}
	if !recordsEqual(snapB, b.store.Records()) {
		t.Fatalf("node B's store changed under adversarial fire: before %d records, now %d",
			len(snapB), len(b.store.Records()))
	}
	if got := b.adapter.KnownPeers(); got != 1 {
		t.Fatalf("node B known peers = %d, want exactly 1 (the adversarial datagrams never reach B)", got)
	}
	if got := a.adapter.KnownPeers(); got < 1 {
		t.Fatal("node A no longer knows any peer — mutual discovery destroyed")
	}
}

// TestScenario_discardClasses drives the receiver's classifier over real
// sockets: two separately bound third-party sockets — two distinct source
// handles — fire hand-crafted datagrams at node A's port, and each case —
// isolated on its own converged pair, so the cases carry no baseline
// coupling — asserts ONLY its own Discards class increments while the other
// four stay at their pre-shot values and the converged two-node state
// survives untouched. The duplicate-node_id case is phase 4's pinned second
// variant: one node_id, remembered run_id equal, DIFFERENT source handles —
// a run's identity never migrates handles — and the classifier must count
// DuplicateNodeID and log the configuration error naming the node_id and
// both handles (REPL-08). The impostor's datagrams ride the test-binary
// encode helpers; the version-mismatch one carries a CORRECT CRC so the
// core's version check is the branch under test.
func TestScenario_discardClasses(t *testing.T) {
	t.Parallel()

	t.Run("bad_crc", func(t *testing.T) {
		t.Parallel()
		cfg := scenarioTimers(func(msg string) { t.Logf("%s", msg) })
		a, b, portA, snapA, snapB := scenarioConvergedPair(t, cfg, cfg)
		sock3 := peerSocket(t)
		before := a.adapter.Discards()
		datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, scenarioImpostorNode, scenarioImpostorRun)
		if err != nil {
			t.Fatalf("encode Hello: %v", err)
		}
		// One byte flipped inside the CRC-covered region (the body's last
		// byte): the envelope still parses, the CRC no longer covers the
		// body — the framing class exactly.
		datagram[len(datagram)-1] ^= corruptByte
		sendDatagram(t, sock3, portA, datagram)
		waitFor(t, scenarioEffectWait, "bad_crc to increment by exactly one", func() bool {
			return a.adapter.Discards().BadCRC == before.BadCRC+1
		})
		scenarioAssertClassExact(t, repl.DiscardBadCRC, a.adapter.Discards(), before)
		scenarioAssertConvergenceIntact(t, a, b, snapA, snapB)
	})

	t.Run("bad_proto_version", func(t *testing.T) {
		t.Parallel()
		cfg := scenarioTimers(func(msg string) { t.Logf("%s", msg) })
		a, b, portA, snapA, snapB := scenarioConvergedPair(t, cfg, cfg)
		sock3 := peerSocket(t)
		before := a.adapter.Discards()
		// A datagram carrying an unsupported proto version with a correct
		// CRC — the core's version check only runs past a CRC-passing
		// envelope, so the crafting rides the explicit-version test helper.
		datagram, err := udprepl.EncodeDatagramForTestVersion(
			port.Hello{},
			scenarioImpostorNode,
			scenarioImpostorRun,
			repl.SupportedProtoVersion+1,
		)
		if err != nil {
			t.Fatalf("encode version-mismatch Hello: %v", err)
		}
		sendDatagram(t, sock3, portA, datagram)
		waitFor(t, scenarioEffectWait, "bad_proto_version to increment by exactly one", func() bool {
			return a.adapter.Discards().BadProtoVersion == before.BadProtoVersion+1
		})
		scenarioAssertClassExact(t, repl.DiscardBadProtoVersion, a.adapter.Discards(), before)
		scenarioAssertConvergenceIntact(t, a, b, snapA, snapB)
	})

	t.Run("duplicate_node_id", func(t *testing.T) {
		t.Parallel()
		logs := &logCapture{}
		cfg := scenarioTimers(logs.log)
		a, b, portA, snapA, snapB := scenarioConvergedPair(t, cfg, cfg)
		// The adversarial sources: two separately bound loopback sockets
		// held for the case's lifetime, one handle each — a per-datagram
		// throwaway socket would itself look like a migrating identity to
		// the duplicate-node_id classifier.
		sock3 := peerSocket(t)
		sock4 := peerSocket(t)
		before := a.adapter.Discards()
		// One identity, two handles. First the third-party socket's Hello
		// claims node X with run id R, so node A learns peer X at that
		// handle — one more known peer, observable through the mirror.
		hello, err := udprepl.EncodeDatagramForTest(port.Hello{}, scenarioImpostorNode, scenarioImpostorRun)
		if err != nil {
			t.Fatalf("encode impostor Hello: %v", err)
		}
		sendDatagram(t, sock3, portA, hello)
		waitFor(t, scenarioEffectWait, "node A to learn the impostor peer (known peers 1 -> 2)", func() bool {
			return a.adapter.KnownPeers() == 2
		})
		// Then a DIFFERENT socket — a different source handle — claims the
		// same node X with the SAME run id: the classifier counts
		// DuplicateNodeID, the datagram is dropped, and the configuration
		// error names the node_id and both conflicting handles.
		sendDatagram(t, sock4, portA, hello)
		waitFor(t, scenarioEffectWait, "duplicate_node_id to increment by exactly one", func() bool {
			return a.adapter.Discards().DuplicateNodeID == before.DuplicateNodeID+1
		})
		scenarioAssertClassExact(t, repl.DiscardDuplicateNodeID, a.adapter.Discards(), before)
		scenarioAssertConvergenceIntact(t, a, b, snapA, snapB)
		handle3, handle4 := sock3.LocalAddr().String(), sock4.LocalAddr().String()
		var configErrors []string
		for _, msg := range logs.containing(fmt.Sprintf("node_id %d", scenarioImpostorNode)) {
			if strings.Contains(msg, handle3) && strings.Contains(msg, handle4) {
				configErrors = append(configErrors, msg)
			}
		}
		if len(configErrors) == 0 {
			t.Fatalf("no configuration-error log names node id %d with both handles %q and %q — captured: %v",
				scenarioImpostorNode, handle3, handle4, logs.snapshot())
		}
	})
}
