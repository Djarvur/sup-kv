//go:build linux

// Package stand is the two-process real-socket test harness: it spawns
// the real cmd/sup-kv binary as child processes over loopback UDP and
// gRPC, drives their whole lifecycle (build-once, launch, the SIGKILL and
// SIGTERM exit contracts, reaps) and polls convergence through the
// serving endpoints. The harness is moved wholesale from cmd/sup-kv's
// exec suite (D-03: moved/extended, never duplicated); the scenario tests
// live in stand_test.go beside it, and every stand-level deliverable of
// the phase (red tests, deletion levels, metrics by effect, the nightly
// 10k run) builds on this package.
//
// The package is test infrastructure linked only into test binaries. It
// keeps a non-test file so `go build ./internal/stand/` compiles the
// harness on Linux; the testing import below carries a depguard
// justification because the no-test-libs-in-prod rule matches by path and
// its test* directory exemptions cannot see a harness outside a test*
// directory.
package stand

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing" //nolint:depguard // internal/stand is test infrastructure (D-03): the harness helpers take *testing.T by design, and the no-test-libs-in-prod rule matches by path, so its test* directory exemptions cannot see a harness outside a test* directory.
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
)

// Harness constants: the wait deadlines with their poll cadence, the
// per-RPC bounds, the probe's wait, and the margin kept ahead of the
// test's own hard deadline so a timeout fails with the test's message
// rather than the runner's kill (moved verbatim from cmd/sup-kv's
// main_test.go harness block).
const (
	pollEvery          = 10 * time.Millisecond
	servingDeadline    = 10 * time.Second
	exitDeadline       = 15 * time.Second
	checkTimeout       = 500 * time.Millisecond
	rpcTimeout         = 5 * time.Second
	hardDeadlineMargin = 250 * time.Millisecond
	logProbeWait       = 300 * time.Millisecond
)

// The stand pair's wire identities (distinct, the one configuration rule
// the design states) and the shrunken timers every node runs with (the
// source scenario constants moved verbatim: discovery and sync rounds
// complete inside CI deadlines while every wait stays far away from
// them).
const (
	standNodeAID = 1
	standNodeBID = 2

	standHelloInterval = 100 * time.Millisecond
	standSyncInterval  = 200 * time.Millisecond
	standStartupGate   = time.Second
)

// The scenario UDP port window: strictly below the kernel's ephemeral
// range (ip_local_port_range starts at 32768 on Linux), so no other
// test's :0 bind can ever be issued there — a scenario's picked port is
// unreachable to every rival socket until its nodes bind it, which
// removes structurally the pick-to-bind race every free-port picker in
// the shared ephemeral pool suffers under the parallel burst (the T-11
// flake-class-1 mitigation; it must survive the harness move intact).
// The band [27900, 28100) below the window is RESERVED for the fixed
// documented ports of README.md and docs/EXAMPLES.md: the doc blocks bind
// constants, so scenario picks must start above the band to stay
// collision-free against the verbatim doc tests.
const (
	scenarioPortLo    = 28100
	scenarioPortHi    = 32678
	scenarioPortTries = 32
)

// standBuildTarget is the package path the build-once builder compiles:
// the real cmd/sup-kv binary, named by package path because the stand
// test binary's working directory is internal/stand, not cmd/sup-kv —
// the source harness's dot form cannot resolve here.
const standBuildTarget = "github.com/Djarvur/sup-kv/cmd/sup-kv"

// standBuildState is the build-once binary's shared state: the first
// standBinary call (or the eager wiring in stand_test.go's TestMain)
// builds once, and every later call reuses the result. The error is
// surfaced to the caller — standBinary turns it into a loud fatal on
// Linux, never a silent pass (TEST-05).
type standBuildState struct {
	once sync.Once
	dir  string
	bin  string
	err  error
}

var standBuild standBuildState //nolint:gochecknoglobals // the build-once contract is inherently package-level state; the harness is test infrastructure moved wholesale (D-03), the rng.go seedFlag precedent

// buildStandBinary runs the build-once builder: the probes are MkdirTemp,
// [exec.LookPath] for the go tool and one go build of the cmd/sup-kv
// package into the temp dir. The returned error names the probe that
// failed — standBinary turns it into a loud fatal, never a silent pass.
func buildStandBinary(ctx context.Context) (string, error) {
	standBuild.once.Do(func() {
		dir, err := os.MkdirTemp("", "sup-kv-stand-*")
		if err != nil {
			standBuild.err = fmt.Errorf("probe MkdirTemp: %w", err)
			return
		}
		standBuild.dir = dir
		standBuild.bin, standBuild.err = buildSupKvBinary(ctx, filepath.Join(dir, "sup-kv"))
	})
	return standBuild.bin, standBuild.err
}

// buildSupKvBinary builds the production binary once for all stand
// tests: the go build compiles ./cmd/sup-kv by its package path because
// the stand test binary's working directory is internal/stand, not
// cmd/sup-kv — the source harness's dot form cannot resolve here. The
// returned error names the probe that failed.
func buildSupKvBinary(ctx context.Context, bin string) (string, error) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("probe exec.LookPath(go): %w", err)
	}
	cmd := exec.CommandContext(ctx, goTool, "build", "-o", bin, standBuildTarget) //nolint:gosec // G204: the LookPath go tool builds this module's own cmd/sup-kv, never caller input.
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("probe go build -o %s: %w: %s", bin, err, strings.TrimSpace(out.String()))
	}
	return bin, nil
}

// cleanupStandBinary removes the build-once temp dir — stand_test.go's
// TestMain calls it after the suite.
func cleanupStandBinary() {
	if standBuild.dir != "" {
		_ = os.RemoveAll(standBuild.dir)
	}
}

// standBinary returns the built binary or fails loudly naming the failed
// probe: the stand scenarios exec the real binary, so without it there
// is nothing to observe — and on Linux a failed build is a fatal, never
// a silent skip (TEST-05).
func standBinary(t *testing.T) string {
	t.Helper()
	bin, err := buildStandBinary(t.Context())
	if err != nil {
		t.Fatalf(
			"sup-kv stand binary unavailable: %v — probed exec.LookPath(go) and the go build of ./cmd/sup-kv into %s; on Linux this is a fatal, never a skip (TEST-05)",
			err,
			standBuild.dir,
		)
	}
	return bin
}

// portRegistry is the scenario tests' port registry: the mutex-guarded
// in-use set keeps parallel scenario siblings from picking each other's
// ports during the pick-to-bind window. Entries live until the owning
// test's cleanup, so a scenario's ports stay reserved across its restarts
// too.
type portRegistry struct {
	mu    sync.Mutex
	taken map[int]bool
}

var scenarioPorts = portRegistry{taken: make(map[int]bool)} //nolint:gochecknoglobals // the shared claim map is the registry contract itself; test infrastructure moved wholesale (D-03).

// scenarioUDPPort picks one free UDP port from the private window: try
// random candidates until one binds (anything off-suite holding a port
// there just skips that candidate), claim it in the registry while STILL
// holding the bind — the claim-then-release order closes the sibling race
// — then release and hand the port to the test until its cleanup.
func scenarioUDPPort(t *testing.T) int {
	t.Helper()
	for range scenarioPortTries {
		candidate := scenarioPortLo + rand.IntN(scenarioPortHi-scenarioPortLo) //nolint:gosec // G404: the port pick is a convenience draw, not a security function.
		conn, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp4", fmt.Sprintf("0.0.0.0:%d", candidate))
		if err != nil {
			continue
		}
		scenarioPorts.mu.Lock()
		taken := scenarioPorts.taken[candidate]
		if !taken {
			scenarioPorts.taken[candidate] = true
		}
		scenarioPorts.mu.Unlock()
		_ = conn.Close()
		if taken {
			continue
		}
		t.Cleanup(func() {
			scenarioPorts.mu.Lock()
			delete(scenarioPorts.taken, candidate)
			scenarioPorts.mu.Unlock()
		})
		return candidate
	}
	t.Fatalf("no free UDP port in the private window [%d, %d) after %d tries", scenarioPortLo, scenarioPortHi, scenarioPortTries)
	return 0
}

// freeTCPPort picks one free TCP port: bind a throwaway listener to :0,
// read its port, close it.
func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind throwaway port-picking listener: %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port
}

// logProbeSocket binds a wildcard udp4 socket with SO_REUSEADDR and
// SO_BROADCAST set before bind — the adapter's exact socket recipe, so
// the capability probe measures the production topology (the source
// harness's socket recipe, moved verbatim).
func logProbeSocket(bindAddr string) (*net.UDPConn, error) {
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

// probeLoopbackSend probes whether a datagram sent to the loopback
// broadcast address reaches the socket bound to dstPort (loopback
// broadcast fan-out is a Linux-only kernel behavior; macOS lo0 carries no
// BROADCAST flag). Three outcomes, kept distinct so the caller can act:
// reached (the capability holds), not reached within the wait (the
// platform lacks the capability), or a probe error (the free-port race:
// a parallel test's socket won the port between the pick and this bind —
// the caller picks a fresh port and retries).
//
// not-reached on Linux is a fatal at the call site — TEST-05's no-silent-
// skip policy; on non-Linux this file does not compile at all
// (stand_nondarwin_test.go, created by same-wave sibling plan 02-14,
// reports there). Only the outcome handling changed from the source's
// loud skip: the three-outcome shape is kept intact.
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
	dstAddr := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: dstPort} //nolint:mnd // 127.255.255.255 is the design's fixed loopback broadcast segment address.
	_, err = tx.WriteToUDP([]byte("sup-kv stand scenario probe"), dstAddr)
	if err != nil {
		return false, fmt.Errorf("probe send to %s: %w", dstAddr, err)
	}
	err = rx.SetReadDeadline(time.Now().Add(logProbeWait))
	if err != nil {
		return false, fmt.Errorf("probe read deadline: %w", err)
	}
	buf := make([]byte, probeBufBytes)
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

// probeBufBytes bounds the probe's one-datagram read buffer.
const probeBufBytes = 64

// dumpFileMode is the corrupted-dump rewrite's owner-only file mode.
const dumpFileMode = 0o600

// waitFor polls cond every pollEvery until it holds or the deadline —
// the earlier of timeout and the test's own hard deadline minus a margin
// — and fails with what on expiry. Every stand wait goes through this
// clamp; a bare sleep loop has no place in the harness.
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

// standNode is one running binary under test with its captured output and
// its single Wait result. stderr holds the plain post-exit buffer or the
// live lock-guarded buffer (outputCapture).
type standNode struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr outputCapture
	exit   chan error
	exited chan struct{}
}

// outputCapture is the stderr seam standNode holds: the plain
// [bytes.Buffer] read only after exit and the logBuffer read while the
// process still runs both satisfy it.
type outputCapture interface {
	Write(p []byte) (int, error)
	String() string
}

// logBuffer is a mutex-guarded [bytes.Buffer]: the log-line polls read
// the live process's stderr while the exec package's pump goroutine still
// writes it — a bare buffer is not safe for that concurrency.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write buffers p under the lock.
func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the buffered output under the lock.
func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// launchNode launches the binary with args (env appended to the inherited
// environment) and registers a kill-and-reap cleanup: a test path that
// never reaches its own SIGTERM still cannot leak the process.
func launchNode(t *testing.T, bin string, stderr outputCapture, env []string, args ...string) *standNode {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, args...) //nolint:gosec // G204: bin is the build-once sup-kv binary and args the harness's fixed flag set, never caller input.
	stdout := &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	err := cmd.Start()
	if err != nil {
		t.Fatalf("start %s %v: %v", bin, args, err)
	}
	node := &standNode{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
		exit:   make(chan error, 1),
		exited: make(chan struct{}),
	}
	go func() {
		node.exit <- cmd.Wait()
		close(node.exited)
	}()
	t.Cleanup(node.reap)
	return node
}

// reap kills the node when it outlived its test.
func (n *standNode) reap() {
	select {
	case <-n.exited:
		return
	default:
	}
	_ = n.cmd.Process.Kill()
	<-n.exited
}

// killAndReap drives the kill -9 half of a scenario: SIGKILL semantics
// via Process.Kill — never the SIGTERM contract — then waits for the exit
// and asserts the harness's kill exit shape (an ExitError reporting -1,
// the "signal: killed" Wait result assertExitCode documents).
func killAndReap(t *testing.T, node *standNode) {
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

// signalAndWaitExit SIGTERMs the node and waits for a clean exit 0 — the
// D-19 contract every test path relies on for clean teardown.
func signalAndWaitExit(t *testing.T, node *standNode) {
	t.Helper()
	err := node.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("SIGTERM %d: %v", node.cmd.Process.Pid, err)
	}
	var exitErr error
	var exited bool
	waitFor(t, exitDeadline, "process exit after SIGTERM", func() bool {
		select {
		case exitErr = <-node.exit:
			exited = true
			return true
		default:
			return false
		}
	})
	if !exited {
		t.Fatalf("process %d still running %v after SIGTERM", node.cmd.Process.Pid, exitDeadline)
	}
	assertExitCode(t, exitErr, 0)
}

// assertExitCode checks a completed Wait against the wanted exit code —
// nil means 0; a signal kill reports -1 and fails a non-zero want loudly.
func assertExitCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		if want != 0 {
			t.Fatalf("process exited 0, want %d", want)
		}
		return
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("process wait error = %v, want an ExitError", err)
	}
	if ee.ExitCode() != want {
		t.Fatalf("process exit code = %d (%s), want %d", ee.ExitCode(), ee.String(), want)
	}
}

// awaitServing polls the gRPC health Check over real TCP until it reports
// SERVING — the startup gate opened. Each poll uses a fresh client so the
// poll cadence never couples to one connection's reconnect backoff.
func awaitServing(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	waitFor(t, timeout, "health SERVING on "+addr, func() bool {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return false
		}
		defer func() { _ = conn.Close() }()
		ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
		defer cancel()
		resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
		return err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	})
}

// supKVConn opens one gRPC client connection for the SupKV service calls.
func supKVConn(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient(%s): %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// assertNoPanic fails when the node's combined output carries a Go panic
// or runtime fatal error.
func assertNoPanic(t *testing.T, node *standNode, what string) {
	t.Helper()
	output := node.stdout.String() + node.stderr.String()
	if strings.Contains(output, "panic:") || strings.Contains(output, "fatal error:") {
		t.Fatalf("%s left a panic in the process output:\n%s", what, output)
	}
}

// standPair is one scenario's asymmetric two-node topology: the binary
// under test, the two UDP port slots, the two gRPC endpoints, and the two
// dump paths — everything a restart reuses to stay the SAME node.
type standPair struct {
	bin   string
	udpA  int
	udpB  int
	grpcA string
	grpcB string
	dumpA string
	dumpB string
}

// newStandPair probes the platform capability and reserves the topology's
// ports and dump paths without starting anything — the empty-disk
// scenario starts its nodes at different times, so the scaffold separates
// topology construction from node launch. The UDP ports come from the
// private window below the kernel's ephemeral range, so no parallel test
// can take them between pick and bind (scenarioUDPPort); a probe bind
// error anyway (an off-suite holder in the window) just re-picks and
// retries. A platform that cannot deliver loopback broadcast at all is a
// FATAL naming what was probed (TEST-05): the source's loud skip became
// the Linux error policy, and the asymmetric topology needs exactly this
// capability for the pair's cross-delivery.
func newStandPair(t *testing.T) *standPair {
	t.Helper()
	bin := standBinary(t)
	const portAttempts = 5
	var udpA, udpB int
	for attempt := 1; ; attempt++ {
		udpA, udpB = scenarioUDPPort(t), scenarioUDPPort(t)
		reached, err := probeLoopbackSend(udpB)
		if err == nil {
			if !reached {
				t.Fatalf(
					"loopback broadcast to 127.255.255.255:%d did not reach the bound socket within %v — the platform lacks the capability the two-process stand scenarios need, and on Linux a probe miss is a fatal, never a skip (TEST-05; the darwin report is the same-wave plan 02-14 named test)",
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
	return &standPair{
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
// shrunken timers (flag names per cmd/sup-kv main.go, each with its
// SUPKV_ env twin).
func (p *standPair) nodeArgs(nodeID int) []string {
	bindPort, bcastPort, grpcAddr, dumpPath := p.slots(nodeID)
	return []string{
		"--node-id=" + strconv.Itoa(nodeID),
		"--bind-addr=0.0.0.0:" + strconv.Itoa(bindPort),
		"--broadcast-addr=127.255.255.255:" + strconv.Itoa(bcastPort),
		"--grpc-addr=" + grpcAddr,
		"--dump-path=" + dumpPath,
		"--hello-interval=" + standHelloInterval.String(),
		"--sync-interval=" + standSyncInterval.String(),
		"--startup-gate=" + standStartupGate.String(),
	}
}

// slots returns the named node's (bindPort, bcastPort, grpcAddr, dumpPath)
// from the pair's topology: node A binds A's port broadcasting to B's,
// node B the mirror — a restart asks for its own slots and stays the SAME
// node by construction.
func (p *standPair) slots(nodeID int) (bindPort, bcastPort int, grpcAddr, dumpPath string) {
	if nodeID == standNodeAID {
		return p.udpA, p.udpB, p.grpcA, p.dumpA
	}
	return p.udpB, p.udpA, p.grpcB, p.dumpB
}

// emptyGateArg derives the --startup-gate-empty flag from an explicit
// --startup-gate in the scenario extras (RESEARCH Pitfall 1, T-4-13): the
// frozen R-03 test names only --startup-gate=30s and must keep BOTH timers at
// 30 s, while every other scenario stays on the harness's 1 s cadence. Later
// flags win in main.go's parsing, so the derived value is appended after the
// extras. A scenario that already names the empty gate wins: the function
// returns "" and nothing is appended, so an explicit --startup-gate-empty is
// never overridden — ADR 0006 keeps the two parameters separate and only this
// test harness derives one from the other. Every spelling Go's flag package
// accepts is recognized — single or double dash, and the `flag=value` or
// `flag value` pair form (IN-05: a scenario written `--startup-gate 30s` used
// to derive the harness default instead, racing the closed-side assertions).
func emptyGateArg(extra []string) string {
	if hasFlag(extra, "startup-gate-empty") {
		return ""
	}
	derived := standStartupGate.String()
	if value, ok := flagValue(extra, "startup-gate"); ok {
		derived = value
	}
	return "--startup-gate-empty=" + derived
}

// normalizeFlagArg strips a flag argument's leading dashes — one or two, per
// Go's flag package — so every recognized spelling compares the same way.
func normalizeFlagArg(arg string) string {
	return strings.TrimLeft(arg, "-")
}

// hasFlag reports whether args names flag in any form the flag package
// accepts: -name, --name, -name=value or --name=value. A bare -name/--name
// element counts too (it consumes the following argument as its value).
func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		trimmed := normalizeFlagArg(arg)
		if trimmed == name || strings.HasPrefix(trimmed, name+"=") {
			return true
		}
	}
	return false
}

// flagValue returns the LAST value args give name — later flags win in
// main.go's parsing — accepting both the `flag=value` and the space-separated
// `flag value` pair form. A bare flag with no following argument yields false:
// the flag package would reject it, and deriving a value for a malformed
// command line would only hide that.
func flagValue(args []string, name string) (string, bool) {
	value, found := "", false
	prefix := name + "="
	for i := 0; i < len(args); i++ {
		trimmed := normalizeFlagArg(args[i])
		switch {
		case strings.HasPrefix(trimmed, prefix):
			value, found = strings.TrimPrefix(trimmed, prefix), true
		case trimmed == name && i+1 < len(args):
			value, found = args[i+1], true
			i++ // the value is consumed; it is not an argument of its own
		}
	}
	return value, found
}

// nodeArgsWithExtras composes one node's command line: the topology defaults,
// the scenario extras, and — when the extras did not name one — the derived
// empty gate last, so it beats nodeArgs' own --startup-gate.
func (p *standPair) nodeArgsWithExtras(nodeID int, extra []string) []string {
	args := append(p.nodeArgs(nodeID), extra...)
	if derived := emptyGateArg(extra); derived != "" {
		args = append(args, derived)
	}
	return args
}

// startStandNode launches one scenario node in its topology slot (extra
// args appended by the scenarios that need them, with the empty gate derived
// from an explicit normal gate in those extras).
func startStandNode(t *testing.T, p *standPair, nodeID int, extra ...string) *standNode {
	t.Helper()
	return launchNode(t, p.bin, &bytes.Buffer{}, nil, p.nodeArgsWithExtras(nodeID, extra)...)
}

// startStandLoggedNode is startStandNode with the lock-guarded live
// stderr buffer — the shape the log-line polls use while the process
// still runs.
func startStandLoggedNode(t *testing.T, p *standPair, nodeID int, extra ...string) *standNode {
	t.Helper()
	return launchNode(t, p.bin, &logBuffer{}, nil, p.nodeArgsWithExtras(nodeID, extra)...)
}

// startStandPair launches both nodes of a fresh topology — the scenarios
// whose nodes start together; the empty-disk scenario drives the pieces
// itself. The variadic extras are appended to BOTH nodes (the zero-arg call
// sites stay unchanged), so a scenario-wide flag reaches the pair at once.
func startStandPair(t *testing.T, extra ...string) (p *standPair, nodeA, nodeB *standNode) {
	t.Helper()
	p = newStandPair(t)
	nodeA = startStandNode(t, p, standNodeAID, extra...)
	nodeB = startStandNode(t, p, standNodeBID, extra...)
	return p, nodeA, nodeB
}

// standClient opens one SupKV client on the node's gRPC endpoint (the
// harness's connection helper with its cleanup).
func standClient(t *testing.T, addr string) apiproto.SupKVServiceClient {
	t.Helper()
	return apiproto.NewSupKVServiceClient(supKVConn(t, addr))
}

// standValue is the deterministic per-key scenario value — every
// visibility assertion checks it, so a stale or half-converged record
// never passes.
func standValue(key string) string {
	return "stand-scenario-value:" + key
}

// standSet writes key with its scenario value through the real gRPC
// endpoint.
func standSet(t *testing.T, client apiproto.SupKVServiceClient, key string) {
	t.Helper()
	standSetWith(t, client, key, standValue(key))
}

// standSetWith writes key with the caller's value through the real gRPC
// endpoint — the variant the F2c pin needs: the review's oversize-key
// write carries a 1-byte value (any derived value would trip the store's
// value limit before the datagram ceiling is ever reached).
func standSetWith(t *testing.T, client apiproto.SupKVServiceClient, key, value string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	defer cancel()
	_, err := client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte(value)})
	if err != nil {
		t.Fatalf("Set(%q): %v", key, err)
	}
}

// standGet reads key through the real gRPC endpoint: NotFound (missing OR
// tombstoned, the API's one mapped not-found code) reports found=false;
// any other error fails the test.
func standGet(t *testing.T, client apiproto.SupKVServiceClient, key string) (value string, found bool) {
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

// standServing reports whether the endpoint serves key with its scenario
// value — the convergence poll's condition. Transient RPC errors (an
// endpoint mid-restart, a reconnecting client) read as not-yet; the
// caller's waitFor deadline governs, never a fast failure.
func standServing(t *testing.T, client apiproto.SupKVServiceClient, key string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	resp, err := client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err != nil {
		return false
	}
	rec := resp.GetRecord()
	return rec.GetKey() == key && !rec.GetDeleted() && string(rec.GetValue()) == standValue(key)
}

// standAbsent reports whether the endpoint lacks key — the negative
// assertion's mirror of standServing: NotFound is absence, a served
// record fails the condition, and a transient RPC error reads as
// not-yet-absent so the caller's waitFor deadline governs, never a fast
// failure.
func standAbsent(t *testing.T, client apiproto.SupKVServiceClient, key string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	_, err := client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err == nil {
		return false
	}
	return status.Code(err) == codes.NotFound
}

// readFinalDump reads the named dump file off disk and returns its text —
// the dump-grep seam: callers assert key presence or absence (and JSON
// fragments such as "deleted":true) against the returned text, naming the
// path and carrying the whole dump in the failure message.
func readFinalDump(t *testing.T, dumpPath string) string {
	t.Helper()
	dump, err := os.ReadFile(dumpPath) //nolint:gosec // G304: the path is the harness temp-dir dump path built in newStandPair, never caller input.
	if err != nil {
		t.Fatalf("read the final dump %s: %v", dumpPath, err)
	}
	return string(dump)
}

// corruptDumpRecord surgically corrupts the named record's line in the
// dump file on disk: one bit flipped inside the record's Checksum field
// (the value XOR 1 — guaranteed different, still a valid uint32), every
// other byte of the file untouched. On the next load the generation still
// parses — the JSONL line is structurally sound — but the record's
// checksum no longer matches storage.RecordChecksum, so the loader
// discards exactly that record (the bad-checksum path) and keeps the rest.
func corruptDumpRecord(t *testing.T, dumpPath, key string) {
	t.Helper()
	dump, err := os.ReadFile(dumpPath) //nolint:gosec // G304: the path is the harness temp-dir dump path built in newStandPair, never caller input.
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
	err = os.WriteFile(dumpPath, []byte(strings.Join(lines, "\n")), dumpFileMode) //nolint:gosec // G703: the path is the harness temp-dir dump path built in newStandPair, never caller input.
	if err != nil {
		t.Fatalf("write the corrupted dump %s: %v", dumpPath, err)
	}
}
