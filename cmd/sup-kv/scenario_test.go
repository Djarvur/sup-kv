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
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	execKeyBothA = "exec/bothdown/key-on-a"
	execKeyBothB = "exec/bothdown/key-on-b"

	execKeySigterm = "exec/sigterm/key-on-b"

	// execDiscardLine is the startup line main prints when the loader
	// discarded undecodable records (startEngine's discarded-count log) —
	// the proof a corrupted record took the per-record discard path, not
	// the whole-generation .prev fallback.
	execDiscardLine = "discarded 1 undecodable dumped records at startup"
)

// execValue is the deterministic per-key scenario value — every visibility
// assertion checks it, so a stale or half-converged record never passes.
func execValue(key string) string {
	return "exec-scenario-value:" + key
}

// probeLoopbackSend probes whether a datagram sent to the loopback
// broadcast address reaches the socket bound to dstPort (loopback
// broadcast fan-out is a Linux-only kernel behavior; macOS lo0 carries no
// BROADCAST flag — D-12/A6). Three outcomes, kept distinct so the caller
// can act: reached (the capability holds), not reached within the wait
// (the platform lacks the capability — the loud-skip reason), or a probe
// error (the free-port race: a parallel test's socket won the port between
// the pick and this bind — the caller picks a fresh port and retries).
func probeLoopbackSend(dstPort int) (reached bool, err error) {
	rx, err := logProbeSocket(fmt.Sprintf("0.0.0.0:%d", dstPort))
	if err != nil {
		return false, fmt.Errorf("probe rx bind: %w", err)
	}
	defer func() { _ = rx.Close() }()
	tx, err := logProbeSocket("0.0.0.0:0")
	if err != nil {
		return false, fmt.Errorf("probe tx bind: %w", err)
	}
	defer func() { _ = tx.Close() }()
	dstAddr := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: dstPort}
	_, err = tx.WriteToUDP([]byte("sup-kv exec scenario probe"), dstAddr)
	if err != nil {
		return false, fmt.Errorf("probe send to %s: %w", dstAddr, err)
	}
	err = rx.SetReadDeadline(time.Now().Add(logProbeWait))
	if err != nil {
		return false, fmt.Errorf("probe read deadline: %w", err)
	}
	buf := make([]byte, 64)
	for {
		n, _, readErr := rx.ReadFromUDP(buf)
		if readErr != nil {
			return false, nil
		}
		if n > 0 {
			return true, nil
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
// construction from node launch. A probe bind error means a parallel test
// won the picked port in the pick-to-probe gap (the free-port race the
// whole harness shares): a fresh port pair is picked and the probe retried,
// so the parallel scenario burst stays reliable (Pitfall 6); a platform
// that cannot deliver loopback broadcast at all skips loudly naming what
// was probed (D-12) — the asymmetric topology needs exactly that
// capability for the pair's cross-delivery.
func newExecPair(t *testing.T) *execPair {
	t.Helper()
	bin := smokeBinary(t)
	const portAttempts = 5
	var udpA, udpB int
	for attempt := 1; ; attempt++ {
		udpA, udpB = freeUDPPort(t), freeUDPPort(t)
		reached, err := probeLoopbackSend(udpB)
		if err == nil {
			if !reached {
				t.Skipf(
					"loopback broadcast to 127.255.255.255:%d did not reach the bound socket within %v — platform lacks the capability the two-process scenarios need (the in-process real-socket scenarios stay proven by internal/udprepl/scenario_test.go and the single-node lifecycle by TestMain_lifecycle)",
					udpB,
					logProbeWait,
				)
			}
			break
		}
		if attempt >= portAttempts {
			t.Fatalf("probe could not bind a fresh UDP port in %d attempts: %v", portAttempts, err)
		}
	}
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

// nodeArgs builds the flag set the named node runs with: its identity,
// its asymmetric topology slots, its dump path and gRPC endpoint, and the
// shrunken timers (the TestMain_precedence/lifecycle precedent for
// --startup-gate, plus the discovery and sync rounds — 06-RESEARCH
// Pattern 6).
func (p *execPair) nodeArgs(nodeID int) []string {
	bindPort, bcastPort, grpcAddr, dumpPath := p.slots(nodeID)
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

// slots returns the named node's (bindPort, bcastPort, grpcAddr, dumpPath)
// from the pair's topology: node A binds A's port broadcasting to B's,
// node B the mirror — a restart asks for its own slots and stays the SAME
// node by construction.
func (p *execPair) slots(nodeID int) (bindPort, bcastPort int, grpcAddr, dumpPath string) {
	if nodeID == execNodeAID {
		return p.udpA, p.udpB, p.grpcA, p.dumpA
	}
	return p.udpB, p.udpA, p.grpcB, p.dumpB
}

// startExecNode launches one scenario node in its topology slot (extra
// args appended by the scenarios that need them).
func startExecNode(t *testing.T, p *execPair, nodeID int, extra ...string) *smokeNode { //nolint:unparam // bridging: the empty-disk scenario passes --sync-response-rate through extra.
	t.Helper()
	return startNode(t, p.bin, nil, append(p.nodeArgs(nodeID), extra...)...)
}

// startExecLoggedNode is startExecNode with the lock-guarded live stderr
// buffer — the shape the discard-line assertion polls while the process
// still runs.
func startExecLoggedNode(t *testing.T, p *execPair, nodeID int, extra ...string) *smokeNode {
	t.Helper()
	return launchNode(t, p.bin, &logBuffer{}, nil, append(p.nodeArgs(nodeID), extra...)...)
}

// startExecPair launches both nodes of a fresh topology — the three
// scenarios whose nodes start together; the empty-disk scenario drives the
// pieces itself.
func startExecPair(t *testing.T) (p *execPair, nodeA, nodeB *smokeNode) {
	t.Helper()
	p = newExecPair(t)
	nodeA = startExecNode(t, p, execNodeAID)
	nodeB = startExecNode(t, p, execNodeBID)
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
	nodeARestart := startExecNode(t, p, execNodeAID)
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

// corruptDumpRecord surgically corrupts the named record's line in the dump
// file on disk: one bit flipped inside the record's Checksum field (the
// value XOR 1 — guaranteed different, still a valid uint32), every other
// byte of the file untouched. On the next load the generation still parses
// — the JSONL line is structurally sound — but the record's checksum no
// longer matches storage.RecordChecksum, so the loader discards exactly
// that record (PERS-02's bad-checksum path) and keeps the rest.
func corruptDumpRecord(t *testing.T, dumpPath, key string) {
	t.Helper()
	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read dump %s: %v", dumpPath, err)
	}
	marker := `"key":"` + key + `"`
	lines := strings.Split(string(dump), "\n")
	corrupted := false
	for i, line := range lines {
		if !strings.Contains(line, marker) {
			continue
		}
		if corrupted {
			t.Fatalf("dump %s carries more than one record line for %q", dumpPath, key)
		}
		const checksumTag = `"checksum":`
		at := strings.Index(line, checksumTag)
		if at < 0 {
			t.Fatalf("record line for %q carries no checksum field: %s", key, line)
		}
		digitsStart := at + len(checksumTag)
		digitsEnd := strings.IndexByte(line[digitsStart:], '}')
		if digitsEnd < 0 {
			t.Fatalf("record line for %q has no closing brace after the checksum: %s", key, line)
		}
		digitsEnd += digitsStart
		value, parseErr := strconv.ParseUint(line[digitsStart:digitsEnd], 10, 32)
		if parseErr != nil {
			t.Fatalf("record line for %q carries a non-numeric checksum: %s", key, line)
		}
		lines[i] = line[:digitsStart] + strconv.FormatUint(value^1, 10) + line[digitsEnd:]
		corrupted = true
	}
	if !corrupted {
		t.Fatalf("dump %s carries no record line for %q:\n%s", dumpPath, key, dump)
	}
	err = os.WriteFile(dumpPath, []byte(strings.Join(lines, "\n")), 0o600)
	if err != nil {
		t.Fatalf("write the corrupted dump %s: %v", dumpPath, err)
	}
}

// TestExec_bothNodesDown is the TEST-04 corrupted-dump-records scenario
// inside a both-nodes-down restart: both nodes exit cleanly through the
// D-19 SIGTERM contract, ONE record of node A's dump is corrupted on disk
// while both are down, and the restart proves the two halves separately —
// A restarted alone serves every uncorrupted record from its own disk but
// NOT the corrupted one (the loader discarded it, and with no peer nothing
// can heal it), then B's restart heals the discarded record back onto A
// through replication alone. No operator action beyond restarting the
// processes.
func TestExec_bothNodesDown(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startExecPair(t)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := execClient(t, p.grpcA)
	clientB := execClient(t, p.grpcB)

	// Baseline: one local write per node, converged both ways.
	execSet(t, clientA, execKeyBothA)
	execSet(t, clientB, execKeyBothB)
	waitFor(t, execConvergeDeadline, "the baseline pair converged (both keys on both endpoints)", func() bool {
		return execServing(t, clientA, execKeyBothA) && execServing(t, clientA, execKeyBothB) &&
			execServing(t, clientB, execKeyBothA) && execServing(t, clientB, execKeyBothB)
	})

	// Both nodes down, cleanly: the D-19 exit on each (assertExitCode 0 is
	// signalAndWaitExit's contract).
	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)

	// While both are down, corrupt A's own local write in A's dump: one
	// checksum bit of one record, file otherwise valid.
	corruptDumpRecord(t, p.dumpA, execKeyBothA)

	// Restart A ALONE first: with B still down, nothing can heal the
	// discarded record, so A's own endpoint proves the load's two halves —
	// the corrupted record gone, every other record served from disk.
	nodeARestart := startExecLoggedNode(t, p, execNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	waitFor(t, servingDeadline, "the discard line in restarted A's stderr", func() bool {
		return strings.Contains(nodeARestart.stderr.String(), execDiscardLine)
	})
	if _, found := execGet(t, clientA, execKeyBothA); found {
		t.Fatalf("the corrupted record %q survived the load — the discard path did not run", execKeyBothA)
	}
	if value, found := execGet(t, clientA, execKeyBothB); !found || value != execValue(execKeyBothB) {
		t.Fatalf("the uncorrupted record %q did not survive on disk: (%q, %t)", execKeyBothB, value, found)
	}

	// Bring B back: B's own dump kept the replicated copy of the discarded
	// record, and replication heals A — no operator action beyond the
	// restarts.
	nodeBRestart := startExecNode(t, p, execNodeBID)
	awaitServing(t, p.grpcB, servingDeadline)
	waitFor(t, execConvergeDeadline, "the corrupted record healed back from node B", func() bool {
		return execServing(t, clientA, execKeyBothA) && execServing(t, clientA, execKeyBothB) &&
			execServing(t, clientB, execKeyBothA) && execServing(t, clientB, execKeyBothB)
	})

	signalAndWaitExit(t, nodeARestart)
	signalAndWaitExit(t, nodeBRestart)
	assertNoPanic(t, nodeA, "the first-run node A")
	assertNoPanic(t, nodeB, "the first-run node B")
	assertNoPanic(t, nodeARestart, "the restarted node A through the corrupted dump")
	assertNoPanic(t, nodeBRestart, "the restarted node B")
}

// TestExec_sigtermFinalDump pins the D-19 final-dump contract's replicated
// half end to end: a write made on node B and replicated to node A is in
// A's FINAL confirmed dump after SIGTERM (the final Flush covers
// remote-merged dirtiness, not just local writes), the exit is 0 (a clean
// exit is reserved for a confirmed dump), and a LONE restart of A — B
// stopped before the restart, so no peer interaction can supply the record
// — serves the replicated key purely from the final dump on disk. The
// exit-code negatives (exitDumpFailed, exitUsage) are already pinned by
// the 06-02 and main-test suites and are not duplicated here.
func TestExec_sigtermFinalDump(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startExecPair(t)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := execClient(t, p.grpcA)
	clientB := execClient(t, p.grpcB)

	// A replicated write on A: written on B, converged across.
	execSet(t, clientB, execKeySigterm)
	waitFor(t, execConvergeDeadline, "the replicated write visible on node A", func() bool {
		return execServing(t, clientA, execKeySigterm)
	})

	// SIGTERM node A: exit 0 only after the final confirmed dump.
	signalAndWaitExit(t, nodeA)

	// The final dump on disk contains the REPLICATED record — the
	// TestMain_lifecycle dump-grep precedent.
	dump, err := os.ReadFile(p.dumpA)
	if err != nil {
		t.Fatalf("read the final dump %s: %v", p.dumpA, err)
	}
	if !strings.Contains(string(dump), execKeySigterm) {
		t.Fatalf("the final dump at %s does not contain the replicated key %q:\n%s", p.dumpA, execKeySigterm, dump)
	}

	// Stop B before the restart: with the peer gone, the only possible
	// source for the record on a restarted A is the final dump itself.
	killAndReap(t, nodeB)

	nodeARestart := startExecNode(t, p, execNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	if value, found := execGet(t, clientA, execKeySigterm); !found || value != execValue(execKeySigterm) {
		t.Fatalf("the lone restarted node A does not serve the replicated record from its final dump: (%q, %t)",
			value, found)
	}

	signalAndWaitExit(t, nodeARestart)
	assertNoPanic(t, nodeA, "the SIGTERM node A")
	assertNoPanic(t, nodeB, "the stopped node B")
	assertNoPanic(t, nodeARestart, "the lone restarted node A")
}
