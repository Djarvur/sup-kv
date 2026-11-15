package main_test

// The exec half of TEST-04 (D-11): exactly four two-process scenarios on
// the shipped binary — kill -9 one node then restart with convergence and
// no operator action, both-nodes-down restart with state preserved through
// a corrupted dump record, SIGTERM's final confirmed dump provably covering
// replicated writes, and an empty-disk node recovering from a live peer
// bounded by the sync response rate. These are the only scenarios that
// prove the artifact itself — process lifecycle, real signal delivery,
// real dump files on disk — so they extend the existing main_test.go
// harness (build-once, startNode/reap, signalAndWaitExit/assertExitCode,
// awaitServing/supKVConn), never rebuild it.
//
// Every scenario rides the ASYMMETRIC-PORT topology (06-RESEARCH Pattern
// 5, the empirically load-bearing finding): node A binds P1 and broadcasts
// to 127.255.255.255:P2, node B binds P2 and broadcasts to 127.255.255.255:P1
// — every broadcast and every unicast (the Ack and DataRequest answers the
// shared-port smoke topology cannot carry) crosses to exactly the peer, so
// convergence assertions are deterministic. The one platform capability
// that buys — loopback broadcast fan-out — is probed up front and skips
// loudly naming the probe when the platform lacks it (D-12): asymmetric
// ports remove darwin's shared-bind EADDRINUSE reason, but macOS lo0 still
// carries no BROADCAST flag.

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

// Exec-scenario constants: the pair's wire identities (distinct, the one
// configuration rule the design states), the shrunken timers every node
// runs with (06-RESEARCH Pattern 6 — discovery and sync rounds complete
// inside CI deadlines while every wait stays far away from them), the
// timer-derived convergence bound, and the kill -9 scenario's keys.
const (
	execNodeAID = 1
	execNodeBID = 2

	execHelloInterval = 100 * time.Millisecond
	execSyncInterval  = 200 * time.Millisecond
	execStartupGate   = time.Second

	// execConvergeDeadline bounds one write-to-visible cycle on either
	// endpoint, derived from the shrunken timers rather than wall-clock
	// precision (Pitfall 6): one hello round for discovery, a few sync
	// rounds for the listing/request/replay ladder, and real-process
	// scheduling slack — ten of each interval covers every contribution.
	execConvergeDeadline = 10*execHelloInterval + 10*execSyncInterval

	execKeyKillA = "exec/kill9/key-on-a"
	execKeyKillB = "exec/kill9/key-on-b"
)

// execValue is the deterministic per-key scenario value — every visibility
// assertion checks it, so a stale or half-converged record never passes.
func execValue(key string) string {
	return "exec-scenario-value:" + key
}

// scenarioProbeLoopback pre-flights the one platform capability the exec
// scenarios need — a datagram sent to the loopback broadcast address
// reaches the socket bound to the destination port (loopback broadcast
// fan-out is a Linux-only kernel behavior; macOS lo0 carries no BROADCAST
// flag — D-12/A6) — and skips loudly naming what was probed when it is
// missing. The asymmetric topology needs exactly this capability for the
// pair's cross-delivery; TEST-04's in-process real-socket half and the
// single-node lifecycle stay proven on every platform by
// internal/udprepl/scenario_test.go and TestMain_lifecycle.
func scenarioProbeLoopback(t *testing.T, port int) {
	t.Helper()
	rx, err := logProbeSocket(fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		t.Fatalf("probe rx bind: %v", err)
	}
	defer func() { _ = rx.Close() }()
	tx, err := logProbeSocket("0.0.0.0:0")
	if err != nil {
		t.Fatalf("probe tx bind: %v", err)
	}
	defer func() { _ = tx.Close() }()
	dstAddr := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: port}
	_, err = tx.WriteToUDP([]byte("sup-kv exec scenario probe"), dstAddr)
	if err != nil {
		t.Fatalf("probe send to %s: %v", dstAddr, err)
	}
	err = rx.SetReadDeadline(time.Now().Add(logProbeWait))
	if err != nil {
		t.Fatalf("probe read deadline: %v", err)
	}
	buf := make([]byte, 64)
	for {
		n, _, readErr := rx.ReadFromUDP(buf)
		if readErr != nil {
			t.Skipf(
				"loopback broadcast to %s did not reach the bound socket within %v — platform lacks the capability the two-process scenarios need (the in-process real-socket scenarios stay proven by internal/udprepl/scenario_test.go and the single-node lifecycle by TestMain_lifecycle)",
				dstAddr,
				logProbeWait,
			)
		}
		if n > 0 {
			return
		}
	}
}

// execPair is one scenario's asymmetric two-node topology: the binary
// under test, the two UDP port slots, the two gRPC endpoints, and the two
// dump paths — everything a restart reuses to stay the SAME node.
type execPair struct {
	bin   string
	udpA  int
	udpB  int
	grpcA string
	grpcB string
	dumpA string
	dumpB string
}

// newExecPair probes the platform capability and reserves the topology's
// ports and dump paths without starting anything — the empty-disk scenario
// starts its nodes at different times, so the scaffold separates topology
// construction from node launch.
func newExecPair(t *testing.T) *execPair {
	t.Helper()
	bin := smokeBinary(t)
	udpA, udpB := freeUDPPort(t), freeUDPPort(t)
	scenarioProbeLoopback(t, udpB)
	return &execPair{
		bin:   bin,
		udpA:  udpA,
		udpB:  udpB,
		grpcA: fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t)),
		grpcB: fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t)),
		dumpA: filepath.Join(t.TempDir(), "dump.json"),
		dumpB: filepath.Join(t.TempDir(), "dump.json"),
	}
}

// execNodeArgs builds the flag set every scenario node runs with: identity,
// the node's asymmetric topology slots, its dump path and gRPC endpoint,
// and the shrunken timers (the TestMain_precedence/lifecycle precedent for
// --startup-gate, plus the discovery and sync rounds — 06-RESEARCH
// Pattern 6).
func execNodeArgs(nodeID, bindPort, bcastPort int, grpcAddr, dumpPath string) []string {
	return []string{
		"--node-id=" + strconv.Itoa(nodeID),
		"--bind-addr=0.0.0.0:" + strconv.Itoa(bindPort),
		"--broadcast-addr=127.255.255.255:" + strconv.Itoa(bcastPort),
		"--grpc-addr=" + grpcAddr,
		"--dump-path=" + dumpPath,
		"--hello-interval=" + execHelloInterval.String(),
		"--sync-interval=" + execSyncInterval.String(),
		"--startup-gate=" + execStartupGate.String(),
	}
}

// startExecNode launches one scenario node with the scaffold's flag set
// (extra args appended by the scenarios that need them).
func startExecNode(t *testing.T, p *execPair, nodeID, bindPort, bcastPort int, grpcAddr, dumpPath string, extra ...string) *smokeNode {
	t.Helper()
	return startNode(t, p.bin, nil, append(execNodeArgs(nodeID, bindPort, bcastPort, grpcAddr, dumpPath), extra...)...)
}

// startExecPair launches both nodes of a fresh topology — the three
// scenarios whose nodes start together; the empty-disk scenario drives the
// pieces itself.
func startExecPair(t *testing.T) (p *execPair, nodeA, nodeB *smokeNode) {
	t.Helper()
	p = newExecPair(t)
	nodeA = startExecNode(t, p, execNodeAID, p.udpA, p.udpB, p.grpcA, p.dumpA)
	nodeB = startExecNode(t, p, execNodeBID, p.udpB, p.udpA, p.grpcB, p.dumpB)
	return p, nodeA, nodeB
}

// execClient opens one SupKV client on the node's gRPC endpoint (the
// harness's connection helper with its cleanup).
func execClient(t *testing.T, addr string) apiproto.SupKVServiceClient {
	t.Helper()
	return apiproto.NewSupKVServiceClient(supKVConn(t, addr))
}

// execSet writes key with its scenario value through the real gRPC
// endpoint.
func execSet(t *testing.T, client apiproto.SupKVServiceClient, key string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	defer cancel()
	_, err := client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte(execValue(key))})
	if err != nil {
		t.Fatalf("Set(%q): %v", key, err)
	}
}

// execGet reads key through the real gRPC endpoint: NotFound (missing OR
// tombstoned, the API's one mapped not-found code) reports found=false;
// any other error fails the test.
func execGet(t *testing.T, client apiproto.SupKVServiceClient, key string) (value string, found bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	defer cancel()
	resp, err := client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err == nil {
		rec := resp.GetRecord()
		return string(rec.GetValue()), true
	}
	if status.Code(err) == codes.NotFound {
		return "", false
	}
	t.Fatalf("Get(%q): %v", key, err)
	return "", false
}

// execServing reports whether the endpoint serves key with its scenario
// value — the convergence poll's condition. Transient RPC errors (an
// endpoint mid-restart, a reconnecting client) read as not-yet; the
// caller's waitFor deadline governs, never a fast failure.
func execServing(t *testing.T, client apiproto.SupKVServiceClient, key string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	resp, err := client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err != nil {
		return false
	}
	rec := resp.GetRecord()
	return rec.GetKey() == key && !rec.GetDeleted() && string(rec.GetValue()) == execValue(key)
}

// killAndReap drives the kill -9 half of a scenario: SIGKILL semantics via
// Process.Kill — never the SIGTERM contract — then waits for the exit and
// asserts the harness's kill exit shape (an ExitError reporting -1, the
// "signal: killed" Wait result assertExitCode documents).
func killAndReap(t *testing.T, node *smokeNode) {
	t.Helper()
	err := node.cmd.Process.Kill()
	if err != nil {
		t.Fatalf("SIGKILL %d: %v", node.cmd.Process.Pid, err)
	}
	var exitErr error
	waitFor(t, exitDeadline, "process exit after SIGKILL", func() bool {
		select {
		case exitErr = <-node.exit:
			return true
		default:
			return false
		}
	})
	assertExitCode(t, exitErr, -1)
}

// TestExec_killNineRestart is the flagship TEST-04 scenario on the real
// binary: two nodes converge a write each way, node A dies to a kill -9
// (the SIGKILL exit shape, never SIGTERM's), node B keeps serving both
// keys through the peer's death, and A restarted with the SAME dump path
// and ports — a fresh process and run id — converges back to both keys
// with no operator action beyond restarting the process: the dump restored
// what it held and the REPL-07 resync healed the rest from the peer.
func TestExec_killNineRestart(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startExecPair(t)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := execClient(t, p.grpcA)
	clientB := execClient(t, p.grpcB)

	// Baseline convergence: one write on each node, both keys visible on
	// both endpoints through real gRPC.
	execSet(t, clientA, execKeyKillA)
	execSet(t, clientB, execKeyKillB)
	waitFor(t, execConvergeDeadline, "the baseline pair converged (both keys on both endpoints)", func() bool {
		return execServing(t, clientA, execKeyKillA) && execServing(t, clientA, execKeyKillB) &&
			execServing(t, clientB, execKeyKillA) && execServing(t, clientB, execKeyKillB)
	})

	// kill -9 node A: the hard death — no dump, no goodbye.
	killAndReap(t, nodeA)

	// Node B stays available through the peer's death: both keys readable,
	// its own endpoint unaffected.
	for _, key := range []string{execKeyKillA, execKeyKillB} {
		if value, found := execGet(t, clientB, key); !found || value != execValue(key) {
			t.Fatalf("node B after the peer's kill -9: Get(%q) = (%q, %t), want the pre-kill value", key, value, found)
		}
	}

	// Restart node A: same dump path, same ports — a fresh process with a
	// fresh run id (the REPL-07 path).
	nodeARestart := startExecNode(t, p, execNodeAID, p.udpA, p.udpB, p.grpcA, p.dumpA)
	awaitServing(t, p.grpcA, servingDeadline)

	// Convergence with no operator action: both pre-kill keys readable via
	// A's own endpoint — whatever the dump restored plus whatever the resync
	// healed from B, the union is the pre-kill state.
	waitFor(t, execConvergeDeadline, "the restarted node A serving both pre-kill keys", func() bool {
		return execServing(t, clientA, execKeyKillA) && execServing(t, clientA, execKeyKillB)
	})

	// Clean teardown of the survivors proves the restarted node is a full
	// participant, and no process left a panic behind.
	signalAndWaitExit(t, nodeARestart)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the kill -9 node A")
	assertNoPanic(t, nodeARestart, "the restarted node A")
	assertNoPanic(t, nodeB, "node B through the peer's death and restart")
}
