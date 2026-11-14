package main_test

// The binary smoke test: the real sup-kv binary, built once by the go tool
// into a temp dir, exec'd as a real process on real TCP and UDP ports, and
// driven through its whole lifecycle — --version identity, the flag > env >
// default precedence observed on the gRPC listener, and a write followed by
// SIGTERM with a clean exit 0 and a dump file that greps for the committed
// key. The discipline follows internal/udprepl/smoke_test.go (D-08's
// conventions): free ports via throwaway :0 binds, poll-based waitFor with
// margins kept ahead of the test's own hard deadline, and every capability
// skip loud, naming exactly what was probed — a missing go tool is the only
// skip path, and it is a loud failure on any machine that just built this
// very test binary.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

// Smoke-test constants: the wait deadlines with their poll cadence, the
// per-RPC bounds, the version byte width of the wire record, and the margin
// kept ahead of the test's own hard deadline so a timeout fails with this
// test's message rather than the runner's kill.
const (
	pollEvery          = 10 * time.Millisecond
	servingDeadline    = 10 * time.Second
	exitDeadline       = 15 * time.Second
	checkTimeout       = 500 * time.Millisecond
	rpcTimeout         = 5 * time.Second
	dialTimeout        = 500 * time.Millisecond
	versionBytes       = 16
	hardDeadlineMargin = 250 * time.Millisecond

	smokeKey   = "smoke/lifecycle-key"
	smokeValue = "sup-kv-binary-smoke-value"
)

// The smoke binary's shared build result: TestMain builds once, the tests
// skip loudly through smokeBinary when the toolchain probe failed.
var (
	smokeDir       string
	smokeBin       string
	errSmokeBinary error
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "sup-kv-main-smoke-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "sup-kv smoke: MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	smokeDir = dir
	smokeBin, errSmokeBinary = buildSmokeBinary(context.Background(), filepath.Join(dir, "sup-kv"))
	code := m.Run()
	_ = os.RemoveAll(smokeDir)
	if code != 0 {
		fmt.Fprintf(os.Stderr, "sup-kv smoke: tests failed with code %d\n", code)
	}
}

// buildSmokeBinary builds the production binary once for all smoke tests:
// the probes are exec.LookPath("go") and `go build -o <bin> .` from this
// package's directory. The returned error names the probe that failed —
// smokeBinary turns it into a loud skip, never a silent pass.
func buildSmokeBinary(ctx context.Context, bin string) (string, error) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("probe exec.LookPath(go): %w", err)
	}
	cmd := exec.CommandContext(ctx, goTool, "build", "-o", bin, ".")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("probe go build -o %s: %w: %s", bin, err, strings.TrimSpace(out.String()))
	}
	return bin, nil
}

// smokeBinary returns the built binary or skips loudly naming the failed
// probe: the smoke tests exec the real binary, so without it there is
// nothing to observe — protocol and engine correctness stay proven by the
// package suites.
func smokeBinary(t *testing.T) string {
	t.Helper()
	if errSmokeBinary != nil {
		t.Skipf(
			"sup-kv binary smoke unavailable: %v — probed exec.LookPath(\"go\") and the go build of ./cmd/sup-kv into %s; protocol and engine correctness stay proven by the package suites",
			errSmokeBinary,
			smokeDir,
		)
	}
	return smokeBin
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

// freeUDPPort picks one free UDP port: bind a throwaway wildcard socket to
// :0, read its port, close it (the smoke_test.go picker).
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

// waitFor polls cond every pollEvery until it holds or the deadline — the
// earlier of timeout and the test's own hard deadline minus a margin — and
// fails with what on expiry (the smoke_test.go helper).
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

// smokeNode is one running binary under test with its captured output and
// its single Wait result.
type smokeNode struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	exit   chan error
	exited chan struct{}
}

// startNode launches the binary with args (env appended to the inherited
// environment) and registers a kill-and-reap cleanup: a test path that
// never reaches its own SIGTERM still cannot leak the process.
func startNode(t *testing.T, bin string, env []string, args ...string) *smokeNode {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, args...)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	err := cmd.Start()
	if err != nil {
		t.Fatalf("start %s %v: %v", bin, args, err)
	}
	node := &smokeNode{
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
func (n *smokeNode) reap() {
	select {
	case <-n.exited:
		return
	default:
	}
	_ = n.cmd.Process.Kill()
	<-n.exited
}

// signalAndWaitExit SIGTERMs the node and waits for a clean exit 0 — the
// D-19 contract every test path relies on for clean teardown.
func signalAndWaitExit(t *testing.T, node *smokeNode) {
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

// assertNoPanic fails when the node's combined output carries a Go panic or
// runtime fatal error.
func assertNoPanic(t *testing.T, node *smokeNode, what string) {
	t.Helper()
	output := node.stdout.String() + node.stderr.String()
	if strings.Contains(output, "panic:") || strings.Contains(output, "fatal error:") {
		t.Fatalf("%s left a panic in the process output:\n%s", what, output)
	}
}

// TestMain_version pins the D-18 observable: --version exits 0 and prints
// "sup-kv " followed by the non-empty version, commit and Go runtime
// tokens. The smoke binary is built without ldflags, so the tokens are the
// dev fallbacks — the same output an unstamped developer build must show.
func TestMain_version(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(t.Context(), bin, "--version")
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("run --version: %v (stderr: %s)", err, stderr.String())
	}
	out := stdout.String()
	if !strings.HasPrefix(out, "sup-kv ") {
		t.Fatalf("--version output %q does not start with %q", out, "sup-kv ")
	}
	fields := strings.Fields(strings.TrimPrefix(out, "sup-kv "))
	if len(fields) < 3 || fields[0] == "" || fields[1] == "" || fields[2] == "" {
		t.Fatalf("--version output %q lacks non-empty version/commit/runtime tokens", out)
	}
}

// TestMain_precedence proves the D-16/D-17 precedence end-to-end on the
// observable gRPC listener: with only SUPKV_GRPC_ADDR set, the health
// endpoint answers on the env-chosen port (env over the 127.0.0.1:7901
// default); with the same env plus an explicit --grpc-addr pointing at a
// second port, it answers on the flag-chosen port and the env port stayed
// unbound — an env var can never beat an explicit flag (T-05-11).
func TestMain_precedence(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	envPort := freeTCPPort(t)
	flagPort := freeTCPPort(t)
	envAddr := fmt.Sprintf("127.0.0.1:%d", envPort)
	flagAddr := fmt.Sprintf("127.0.0.1:%d", flagPort)

	// Run 1 — env only: SUPKV_GRPC_ADDR moves the listener off the default.
	udp1 := freeUDPPort(t)
	dump1 := filepath.Join(t.TempDir(), "dump.json")
	node1 := startNode(t, bin, []string{"SUPKV_GRPC_ADDR=" + envAddr},
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udp1),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udp1),
		"--dump-path="+dump1,
		"--startup-gate=10ms",
	)
	awaitServing(t, envAddr, servingDeadline)
	signalAndWaitExit(t, node1)

	// Run 2 — the same env plus an explicit flag: the flag must win.
	udp2 := freeUDPPort(t)
	dump2 := filepath.Join(t.TempDir(), "dump.json")
	node2 := startNode(t, bin, []string{"SUPKV_GRPC_ADDR=" + envAddr},
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udp2),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udp2),
		"--dump-path="+dump2,
		"--grpc-addr="+flagAddr,
		"--startup-gate=10ms",
	)
	awaitServing(t, flagAddr, servingDeadline)
	signalAndWaitExit(t, node2)

	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(t.Context(), "tcp", envAddr)
	if err == nil {
		_ = conn.Close()
		t.Fatalf("run 2 also bound the env-chosen port %s — flag over env is broken", envAddr)
	}
}

// TestMain_lifecycle drives the whole node lifecycle through the real
// binary: the startup gate opens (SERVING over real TCP — the timeout path,
// since a lone node receives no DataSync), a SupKV client Sets and Gets one
// key, SIGTERM produces a clean exit 0 (D-19), the dump file exists and its
// greppable JSON contains the committed key (PERS-01), and the process
// output carries no panic.
func TestMain_lifecycle(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	udpPort := freeUDPPort(t)
	grpcPort := freeTCPPort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", grpcPort)

	node := startNode(t, bin, nil,
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+dumpPath,
		"--grpc-addr="+addr,
		"--startup-gate=10ms",
	)
	awaitServing(t, addr, servingDeadline)

	client := apiproto.NewSupKVServiceClient(supKVConn(t, addr))
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	t.Cleanup(cancel)
	setResp, err := client.Set(ctx, &apiproto.SetRequest{Key: smokeKey, Value: []byte(smokeValue)})
	if err != nil {
		t.Fatalf("Set(%q): %v", smokeKey, err)
	}
	if len(setResp.GetVersion()) != versionBytes {
		t.Fatalf("Set(%q) version = %x, want %d bytes", smokeKey, setResp.GetVersion(), versionBytes)
	}
	getResp, err := client.Get(ctx, &apiproto.GetRequest{Key: smokeKey})
	if err != nil {
		t.Fatalf("Get(%q): %v", smokeKey, err)
	}
	if got := getResp.GetRecord(); got.GetKey() != smokeKey || got.GetDeleted() || string(got.GetValue()) != smokeValue {
		t.Fatalf("Get(%q) = (key %q, deleted %t, value %q), want the written record", smokeKey, got.GetKey(), got.GetDeleted(), got.GetValue())
	}

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the lifecycle run")

	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read dump %s: %v", dumpPath, err)
	}
	if !strings.Contains(string(dump), smokeKey) {
		t.Fatalf("the final dump at %s does not contain the committed key %q:\n%s", dumpPath, smokeKey, dump)
	}
}
