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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	grpcstatus "google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/testutil"
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

// freeTCPPort returns one free TCP port from cmd/sup-kv's band inside the
// private port window (testutil.CmdBand: [29100, 30100), strictly below the
// kernel's ephemeral range, so no other test's :0 bind is ever issued there
// and the band is disjoint from every sibling package's). The gRPC and
// metrics listeners had the same pick-to-bind window as the UDP slot; the
// shared allocator claims the candidate while its bind probe is held, so no
// parallel test in this process can take it between pick and bind.
func freeTCPPort(t *testing.T) int {
	t.Helper()
	return testutil.TCPPort(t, testutil.CmdBand())
}

// freeUDPPort returns one free UDP port from the same cmd/sup-kv band, via
// the shared claim-while-held allocator (testutil.UDPPort).
func freeUDPPort(t *testing.T) int {
	t.Helper()
	return testutil.UDPPort(t, testutil.CmdBand())
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
// its single Wait result. stderr holds the plain post-exit buffer or the
// structured-logging tests' lock-guarded live buffer (outputCapture).
type smokeNode struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr outputCapture
	exit   chan error
	exited chan struct{}
}

// outputCapture is the stderr seam smokeNode holds: the plain bytes.Buffer
// read only after exit and the logBuffer read while the process still runs
// both satisfy it.
type outputCapture interface {
	Write(p []byte) (int, error)
	String() string
}

// logBuffer is a mutex-guarded bytes.Buffer: the structured-logging tests
// read the live process's stderr while the exec package's pump goroutine
// still writes it — a bare bytes.Buffer is not safe for that concurrency.
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

// startNode launches the binary with args (env appended to the inherited
// environment) and registers a kill-and-reap cleanup: a test path that
// never reaches its own SIGTERM still cannot leak the process.
func startNode(t *testing.T, bin string, env []string, args ...string) *smokeNode {
	t.Helper()
	return launchNode(t, bin, &bytes.Buffer{}, env, args...)
}

// startLoggedNode is startNode with the lock-guarded live stderr buffer —
// the shape the structured-logging tests poll while the node runs.
func startLoggedNode(t *testing.T, bin string, env []string, args ...string) *smokeNode {
	t.Helper()
	return launchNode(t, bin, &logBuffer{}, env, args...)
}

// launchNode is the shared startNode body over one stderr capture.
func launchNode(t *testing.T, bin string, stderr outputCapture, env []string, args ...string) *smokeNode {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, args...)
	stdout := &bytes.Buffer{}
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

// TestMain_helpListsEnvOnly pins the OBSV-02 usage contract at the real
// binary boundary: --help exits 0 and the Environment-only block names all
// five flagless knobs, so the logging and metrics-certificate surface is
// discoverable from the help output alone.
func TestMain_helpListsEnvOnly(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	out, err := runToCompletion(t, bin, nil, "--help")
	if err != nil {
		t.Fatalf("--help exited with %v, want the flag package's clean help exit 0", err)
	}
	if !strings.Contains(out, "Environment-only (no flag):") {
		t.Fatalf("--help output lacks the %q block:\n%s", "Environment-only (no flag):", out)
	}
	for _, name := range []string{
		"SUPKV_LOG_LEVEL",
		"SUPKV_LOG_FORMAT",
		"SUPKV_METRICS_CERT",
		"SUPKV_METRICS_KEY",
		"SUPKV_METRICS_CLIENT_CA",
	} {
		if !strings.Contains(out, name) {
			t.Fatalf("--help output does not name the env-only variable %s:\n%s", name, out)
		}
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
		"--startup-gate-empty=10ms",
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
		"--startup-gate-empty=10ms",
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

// TestMain_maxClockLead proves the XPORT-04 knob's flag/env twin at the
// configuration seam: --help lists the flag with its env twin named and the
// documented 2 m default (strictly above the 60 s drift-warn default, so the
// drift WARN window is reachable); the env twin is read into the flag's default (the
// help output shows the env-chosen value); a malformed env value aborts
// through the collected env-error path with exitUsage; and an explicit flag
// value is the parsed source even when a valid env twin is set (a malformed
// flag still fails the run — the env never shadows it). The bound's runtime
// behavior is the core's proof (TestDiscard_beforeClock and
// TestDiscard_validationClasses); this test owns the launch wiring.
func TestMain_maxClockLead(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	t.Run("help lists the flag, its env twin and the 2m default", func(t *testing.T) {
		t.Parallel()
		out, err := runToCompletion(t, bin, nil, "--help")
		if err != nil {
			t.Fatalf("--help exited with %v, want the flag package's clean help exit 0", err)
		}
		for _, want := range []string{"-max-clock-lead", "env SUPKV_MAX_CLOCK_LEAD", "(default 2m0s)"} {
			if !strings.Contains(out, want) {
				t.Fatalf("--help does not contain %q — the flag, its env twin or its documented default is missing:\n%s", want, out)
			}
		}
	})

	t.Run("the env twin sets the flag default", func(t *testing.T) {
		t.Parallel()
		out, err := runToCompletion(t, bin, []string{"SUPKV_MAX_CLOCK_LEAD=90s"}, "--help")
		if err != nil {
			t.Fatalf("--help with the env twin exited with %v, want exit 0", err)
		}
		if !strings.Contains(out, "(default 1m30s)") {
			t.Fatalf("--help with SUPKV_MAX_CLOCK_LEAD=90s does not show the env-derived default \"(default 1m30s)\":\n%s", out)
		}
	})

	t.Run("a malformed env value aborts with exitUsage", func(t *testing.T) {
		t.Parallel()
		// The configuration error fires before any bind, so no real port is
		// needed — and picking a free one here would only add another
		// parallel port-race window to the smoke harness.
		_, err := runToCompletion(t, bin, []string{"SUPKV_MAX_CLOCK_LEAD=not-a-duration"},
			"--node-id=1",
			"--bind-addr=127.0.0.1:9",
			"--broadcast-addr=127.255.255.255:9",
			"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
		)
		assertExitCode(t, err, exitUsageCode)
	})

	t.Run("an explicit flag is parsed over a valid env twin", func(t *testing.T) {
		t.Parallel()
		// The env twin carries a perfectly valid duration; the malformed
		// explicit flag must still fail the run, which it can only do if the
		// command-line flag is the parsed source. No bind happens, so the
		// addresses are placeholders.
		_, err := runToCompletion(t, bin, []string{"SUPKV_MAX_CLOCK_LEAD=90s"},
			"--node-id=1",
			"--bind-addr=127.0.0.1:9",
			"--broadcast-addr=127.255.255.255:9",
			"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
			"--max-clock-lead=not-a-duration",
		)
		assertExitCode(t, err, exitUsageCode)
	})
}

// runToCompletion runs the binary with env appended to the inherited
// environment and returns its combined output with the Wait result — the
// shape for --help probes and startup-error rows, which terminate on their
// own.
func runToCompletion(t *testing.T, bin string, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, args...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	err := cmd.Run()
	return out.String(), err
}

// TestMain_valueMaxDatagram pins XPORT-03's startup ceiling executably: the
// P-01 flag value (64000) and the exact largest fitting value (64 408) still
// start and serve, while one byte more (64 409) and a clearly oversized
// value (65 500) exit with the usage code and an error naming
// --value-max-bytes, the key limit and the ceiling. The equality case pins
// the comparison as `>` (the record is exactly 65 507 bytes and passes); the
// adjacent case pins it as exclusive. This test is the executable guard for
// the "64 000 is still accepted" claim; the P-01 expectation-row grep only
// guards that those rows stay unchanged for Phase 5.
func TestMain_valueMaxDatagram(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	for _, tc := range []struct {
		name     string
		valueMax int
	}{
		{"64000 starts and serves", 64000},
		{"the exact boundary starts and serves", startupBoundaryValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dumpPath := filepath.Join(t.TempDir(), "dump.json")
			udpPort := freeUDPPort(t)
			grpcPort := freeTCPPort(t)
			addr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
			node := startLoggedNode(t, bin, nil,
				"--node-id=1",
				"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
				"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
				"--grpc-addr="+addr,
				"--dump-path="+dumpPath,
				"--startup-gate=10ms",
				"--startup-gate-empty=10ms",
				"--value-max-bytes="+strconv.Itoa(tc.valueMax),
			)
			awaitServing(t, addr, servingDeadline)
			if stderr := node.stderr.String(); strings.Contains(stderr, datagramCeilingLine) {
				t.Fatalf("--value-max-bytes=%d produced the datagram-ceiling error while starting:\n%s", tc.valueMax, stderr)
			}
			signalAndWaitExit(t, node)
		})
	}

	for _, tc := range []struct {
		name     string
		valueMax int
	}{
		{"one byte over the ceiling exits 2", startupBoundaryOverValue},
		{"a clearly oversized value exits 2", 65500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := runToCompletion(t, bin, nil,
				"--node-id=1",
				"--bind-addr=127.0.0.1:9",
				"--broadcast-addr=127.255.255.255:9",
				"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
				"--value-max-bytes="+strconv.Itoa(tc.valueMax),
			)
			assertExitCode(t, err, exitUsageCode)
			for _, want := range []string{datagramCeilingLine, "--value-max-bytes=" + strconv.Itoa(tc.valueMax), "1024-byte key limit", "65507"} {
				if !strings.Contains(out, want) {
					t.Fatalf("over-ceiling startup output does not contain %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestMain_valueMaxNonPositive pins the value-limit positivity contract at
// the binary boundary (PERSIST-01): a zero or negative --value-max-bytes is
// a configuration error through the usage path (exit 2), naming the flag —
// never a silent fall back to the default limit. The probe bound only stops
// a RED-state binary, which would start and serve, from hanging the test.
func TestMain_valueMaxNonPositive(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	for _, tc := range []struct {
		name string
		flag string
	}{
		{"a zero value limit exits 2", "--value-max-bytes=0"},
		{"a negative value limit exits 2", "--value-max-bytes=-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), valueMaxRangeProbe)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin,
				"--node-id=1",
				"--bind-addr=127.0.0.1:"+strconv.Itoa(freeUDPPort(t)),
				"--broadcast-addr=127.255.255.255:9",
				"--grpc-addr=127.0.0.1:"+strconv.Itoa(freeTCPPort(t)),
				"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
				tc.flag,
			)
			out, err := cmd.CombinedOutput()
			assertExitCode(t, err, exitUsageCode)
			for _, want := range []string{"value-max-bytes must be positive", tc.flag} {
				if !strings.Contains(string(out), want) {
					t.Fatalf("the non-positive value-max-bytes run's output does not contain %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestMain_startupGateEmptyRange pins the startup-gate positivity contract at
// the binary boundary (T-4-12): a non-positive --startup-gate-empty or
// --startup-gate aborts through the usage path (exit 2) with a message naming
// the offending parameter, so a zero or negative value can never mean "stay
// closed forever". The configuration error fires before any bind, so the
// addresses are placeholders; the context bound is only a RED-state safety net
// for a value the validation does not reach.
func TestMain_startupGateEmptyRange(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	for _, tc := range []struct {
		name      string
		flag      string
		wantParam string
	}{
		{"a negative empty gate exits 2", "--startup-gate-empty=-1s", "--startup-gate-empty=-1s"},
		{"a zero normal gate exits 2", "--startup-gate=0", "--startup-gate=0s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), startupGateRangeProbe)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin,
				"--node-id=1",
				"--bind-addr=127.0.0.1:9",
				"--broadcast-addr=127.255.255.255:9",
				"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
				tc.flag,
			)
			out, err := cmd.CombinedOutput()
			assertExitCode(t, err, exitUsageCode)
			for _, want := range []string{"startup gate must be positive", tc.wantParam} {
				if !strings.Contains(string(out), want) {
					t.Fatalf("the non-positive startup-gate run's output does not contain %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestMain_syncChunkBytesCeiling pins the WR-06 binary boundary: an explicit
// --sync-chunk-bytes above the datagram ceiling aborts through the usage path
// (exit 2) with a message naming the knob, its value and the ceiling, before
// any bind — the core's clamp is defense in depth, never a silent substitute
// for telling the operator their budget is unsendable.
func TestMain_syncChunkBytesCeiling(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	ctx, cancel := context.WithTimeout(t.Context(), startupGateRangeProbe)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, bin,
		"--node-id=1",
		"--bind-addr=127.0.0.1:9",
		"--broadcast-addr=127.255.255.255:9",
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
		"--sync-chunk-bytes=65508",
	)
	out, err := cmd.CombinedOutput()
	assertExitCode(t, err, exitUsageCode)
	for _, want := range []string{"sync-chunk-bytes exceeds the datagram ceiling", "--sync-chunk-bytes=65508", "65507"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("the over-ceiling sync-chunk-bytes run's output does not contain %q:\n%s", want, out)
		}
	}
}

// TestMain_validationRelations pins the one-point validation contract at the
// binary boundary (OBSV-02, D-03): each relation and positivity violation
// exits 2 before any bind with an error naming the offending parameter, and a
// trailing positional argument is rejected by name — never silently ignored.
// The configuration error fires before the socket work, so the addresses are
// placeholders; the context bound only stops a RED-state binary, which would
// start and serve, from hanging the test.
func TestMain_validationRelations(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	rows := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "a zero sync chunk budget exits 2",
			args: []string{"--sync-chunk-bytes=0"},
			want: []string{"--sync-chunk-bytes=0"},
		},
		{
			name: "an over-cap sync response rate exits 2",
			args: []string{"--sync-response-rate=1000000001"},
			want: []string{"--sync-response-rate=1000000001"},
		},
		{
			name: "a one-nanosecond retransmit interval exits 2",
			args: []string{"--retransmit-interval=1ns"},
			want: []string{"--retransmit-interval=1ns"},
		},
		{
			name: "a peer timeout equal to the hello interval exits 2",
			args: []string{"--peer-timeout=1s", "--hello-interval=1s"},
			want: []string{"--peer-timeout=1s", "--hello-interval=1s"},
		},
		{
			name: "an ack timeout above its ceiling exits 2",
			args: []string{"--ack-timeout=6s", "--ack-timeout-max=5s"},
			want: []string{"--ack-timeout=6s", "--ack-timeout-max=5s"},
		},
		{
			name: "a max clock lead equal to the drift warn exits 2",
			args: []string{"--drift-warn=2m", "--max-clock-lead=2m"},
			want: []string{"--drift-warn=2m", "--max-clock-lead=2m"},
		},
		{
			name: "an explicitly empty gRPC address exits 2",
			args: []string{"--grpc-addr="},
			want: []string{"--grpc-addr", "127.0.0.1:7901"},
		},
		{
			name: "an out-of-range gRPC port exits 2 before any bind",
			args: []string{"--grpc-addr=127.0.0.1:99999"},
			want: []string{"--grpc-addr", "99999", "outside 0..65535"},
		},
		{
			name: "a broadcast destination port of 0 exits 2",
			args: []string{"--broadcast-addr=127.255.255.255:0"},
			want: []string{"--broadcast-addr", "can never be sent to"},
		},
		{
			name: "a trailing positional argument exits 2 naming it",
			args: []string{"stray-operand"},
			want: []string{"stray-operand"},
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), validationRelationProbe)
			defer cancel()
			args := append([]string{
				"--node-id=1",
				"--bind-addr=127.0.0.1:9",
				"--broadcast-addr=127.255.255.255:9",
				"--dump-path=" + filepath.Join(t.TempDir(), "dump.json"),
			}, row.args...)
			out, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
			assertExitCode(t, err, exitUsageCode)
			for _, want := range row.want {
				if !strings.Contains(string(out), want) {
					t.Fatalf("the invalid-configuration run's output does not contain %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestMain_startupGateEmptySelection pins the D-02 empty-vs-data selection end
// to end through health timing only: a node that loaded no records waits the
// empty backstop — still NOT_SERVING after the normal gate would have opened,
// then SERVING once the empty gate elapses — while a node restarted over a
// dump holding one key opens on the normal gate inside its bound. A seeded
// node never consults startup_gate_empty, and no engine internals are read.
func TestMain_startupGateEmptySelection(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	t.Run("an empty node waits the empty backstop", func(t *testing.T) {
		t.Parallel()
		dumpPath := filepath.Join(t.TempDir(), "dump.json")
		udpPort := freeUDPPort(t)
		addr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
		const normalGate, emptyGate = 50 * time.Millisecond, 1500 * time.Millisecond
		started := time.Now()
		node := startNode(t, bin, nil,
			"--node-id=1",
			"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
			"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
			"--dump-path="+dumpPath,
			"--grpc-addr="+addr,
			"--startup-gate="+normalGate.String(),
			"--startup-gate-empty="+emptyGate.String(),
		)
		// A lone empty node hears no peer, so only a backstop can open the
		// gate. The 300 ms probe is six normal gates past launch: a node
		// that chose the normal backstop is SERVING well before it.
		waitFor(t, servingDeadline, "the empty node's health endpoint answering", func() bool {
			_, ok, _ := healthAnswer(t, addr)
			return ok
		})
		if elapsed := time.Since(started); elapsed < startupGateSelectionProbe {
			time.Sleep(startupGateSelectionProbe - elapsed)
		}
		// healthAnswer documents ok=false as a transient transport error
		// ("not up yet"), not a status: under parallel package load one probe
		// can still fail after the endpoint first answered. Retry within a
		// bound that stays well inside the empty backstop, so a correct
		// selection can never have opened the gate when the answer lands.
		var status healthpb.HealthCheckResponse_ServingStatus
		statusOK := false
		waitFor(t, (emptyGate-startupGateSelectionProbe)/2, "the empty node's health answer after the probe", func() bool {
			s, ok, _ := healthAnswer(t, addr)
			if !ok {
				return false
			}
			status, statusOK = s, true
			return true
		})
		if !statusOK || status != healthpb.HealthCheckResponse_NOT_SERVING {
			t.Fatalf("the empty node's health at >= %v after launch = (%v, ok=%t), want NOT_SERVING: the empty backstop (%v), not the normal gate (%v), must govern",
				startupGateSelectionProbe, status, statusOK, emptyGate, normalGate)
		}
		awaitServing(t, addr, 5*time.Second)
		signalAndWaitExit(t, node)
		assertNoPanic(t, node, "the empty-gate node")
	})

	t.Run("a seeded node opens on the normal gate", func(t *testing.T) {
		t.Parallel()
		dumpPath := filepath.Join(t.TempDir(), "dump.json")
		udpPort := freeUDPPort(t)
		addr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
		seedKey := "startup-gate-empty/seed"
		const normalGate, emptyGate = 50 * time.Millisecond, 1500 * time.Millisecond
		baseFlags := []string{
			"--node-id=1",
			"--bind-addr=0.0.0.0:" + strconv.Itoa(udpPort),
			"--broadcast-addr=127.255.255.255:" + strconv.Itoa(udpPort),
			"--dump-path=" + dumpPath,
			"--grpc-addr=" + addr,
		}

		// First run: a fast empty-gated node writes one key and exits
		// cleanly, so the restart's dump holds exactly one record.
		seed := startNode(t, bin, nil, append(baseFlags,
			"--startup-gate=10ms",
			"--startup-gate-empty=10ms",
		)...)
		awaitServing(t, addr, servingDeadline)
		client := apiproto.NewSupKVServiceClient(supKVConn(t, addr))
		ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
		_, err := client.Set(ctx, &apiproto.SetRequest{Key: seedKey, Value: []byte(smokeValue)})
		cancel()
		if err != nil {
			t.Fatalf("Set(%q) on the seeding run: %v", seedKey, err)
		}
		signalAndWaitExit(t, seed)

		// The restart loads the record, so the normal gate governs: 300 ms
		// leaves room for process start plus the 50 ms gate while staying far
		// below the empty backstop a broken selection would choose.
		started := time.Now()
		node := startNode(t, bin, nil, append(baseFlags,
			"--startup-gate="+normalGate.String(),
			"--startup-gate-empty="+emptyGate.String(),
		)...)
		awaitServing(t, addr, servingDeadline)
		if elapsed := time.Since(started); elapsed > startupGateSeededBound {
			t.Fatalf("the seeded node opened %v after launch, want within %v (the normal gate, not the %v empty backstop)",
				elapsed, startupGateSeededBound, emptyGate)
		}
		signalAndWaitExit(t, node)
		assertNoPanic(t, node, "the seeded-gate node")
	})
}

// healthAnswer performs one health Check against addr. ok is false when the
// listener is not up yet — a transient transport error — so polls treat it as
// not-yet rather than failing.
func healthAnswer(t *testing.T, addr string) (healthpb.HealthCheckResponse_ServingStatus, bool, error) {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return healthpb.HealthCheckResponse_SERVICE_UNKNOWN, false, err
	}
	defer func() { _ = conn.Close() }()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return healthpb.HealthCheckResponse_SERVICE_UNKNOWN, false, err
	}
	return resp.GetStatus(), true, nil
}

// bindAddrWarnMsg is the stable substring of XPORT-09's non-wildcard bind
// WARN — the message, not its detail attr, so the assertion pins the one
// startup line the operator sees.
const bindAddrWarnMsg = "Bind address is not a wildcard"

// TestMain_bindAddrWarn pins XPORT-09's bind-address visibility half at the
// binary boundary: a --bind-addr naming a concrete interface starts serving
// normally (a unicast-only deployment stays usable — never a rejection) but
// emits exactly one explanatory WARN naming the broadcast-reception
// limitation, while a wildcard bind emits none.
func TestMain_bindAddrWarn(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	t.Run("a non-wildcard bind warns once and keeps serving", func(t *testing.T) {
		t.Parallel()
		udpPort := freeUDPPort(t)
		grpcAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
		node := startLoggedNode(t, bin, nil,
			"--node-id=1",
			"--bind-addr=127.0.0.1:"+strconv.Itoa(udpPort),
			"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
			"--grpc-addr="+grpcAddr,
			"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
			"--startup-gate=10ms",
			"--startup-gate-empty=10ms",
		)
		awaitServing(t, grpcAddr, servingDeadline)
		waitFor(t, servingDeadline, "the non-wildcard bind WARN", func() bool {
			return strings.Contains(node.stderr.String(), bindAddrWarnMsg)
		})
		if got := strings.Count(node.stderr.String(), bindAddrWarnMsg); got != 1 {
			t.Fatalf("a non-wildcard bind produced %d WARN(s), want exactly 1:\n%s", got, node.stderr.String())
		}
		signalAndWaitExit(t, node)
	})

	t.Run("a wildcard bind does not warn", func(t *testing.T) {
		t.Parallel()
		udpPort := freeUDPPort(t)
		grpcAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
		node := startLoggedNode(t, bin, nil,
			"--node-id=1",
			"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
			"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
			"--grpc-addr="+grpcAddr,
			"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
			"--startup-gate=10ms",
			"--startup-gate-empty=10ms",
		)
		// The WARN is emitted synchronously before wire, so a serving node
		// has already decided the question.
		awaitServing(t, grpcAddr, servingDeadline)
		if out := node.stderr.String(); strings.Contains(out, bindAddrWarnMsg) {
			t.Fatalf("a wildcard bind produced the non-wildcard WARN:\n%s", out)
		}
		signalAndWaitExit(t, node)
	})
}

// TestMain_duplicateBind pins XPORT-09's duplicate-instance half at the
// binary boundary: with the UDP address already occupied by another socket,
// the real process exits non-zero and its output carries the kernel's
// "address already in use" — a second instance can never start silently and
// split the node's unicast traffic between two sockets.
func TestMain_duplicateBind(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	// Occupy one explicit loopback address. A plain ListenConfig datagram
	// bind sets no SO_REUSEADDR, so the binary's production bind — which
	// sets none either — must be refused by the kernel.
	occupier, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy a loopback udp4 address: %v", err)
	}
	t.Cleanup(func() { _ = occupier.Close() })
	occupied := occupier.LocalAddr().(*net.UDPAddr)
	bindAddr := occupied.String()
	udpPort := strconv.Itoa(occupied.Port)

	out, err := runToCompletion(t, bin, nil,
		"--node-id=1",
		"--bind-addr="+bindAddr,
		"--broadcast-addr=127.255.255.255:"+udpPort,
		"--grpc-addr=127.0.0.1:"+strconv.Itoa(freeTCPPort(t)),
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	assertExitCode(t, err, 1)
	if !strings.Contains(out, "address already in use") {
		t.Fatalf("a second instance on occupied %s exited without %q in its output:\n%s", bindAddr, "address already in use", out)
	}
}

// TestMain_fatalBindLoggedAtError pins D-06's silent-exit fix at the binary
// boundary: a fatal bind at SUPKV_LOG_LEVEL=error exits 1 AND leaves exactly
// one parsed ERROR record naming the startup stage. The pre-D-06 log.Fatalf
// rendered as an INFO record through slog.SetDefault and vanished at error
// level, exiting 1 with no explanation (review O-01).
func TestMain_fatalBindLoggedAtError(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	// Occupy one explicit loopback address, exactly as TestMain_duplicateBind
	// does: a plain datagram bind sets no SO_REUSEADDR, so the binary's
	// production bind must be refused by the kernel.
	occupier, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy a loopback udp4 address: %v", err)
	}
	t.Cleanup(func() { _ = occupier.Close() })
	occupied := occupier.LocalAddr().(*net.UDPAddr)

	out, err := runToCompletion(t, bin, []string{"SUPKV_LOG_LEVEL=error"},
		"--node-id=1",
		"--bind-addr="+occupied.String(),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(occupied.Port),
		"--grpc-addr=127.0.0.1:"+strconv.Itoa(freeTCPPort(t)),
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
	)
	assertExitCode(t, err, exitStartupCode)

	var errRecords []logRecord
	for _, record := range parseLogRecords(out) {
		if record.Level == "ERROR" {
			errRecords = append(errRecords, record)
		}
	}
	if len(errRecords) != 1 {
		t.Fatalf("a fatal bind at SUPKV_LOG_LEVEL=error produced %d parsed ERROR records, want exactly 1:\n%s", len(errRecords), out)
	}
	if errRecords[0].Msg != "Startup failed" {
		t.Fatalf("the fatal-bind ERROR record msg = %q, want %q:\n%s", errRecords[0].Msg, "Startup failed", out)
	}
	if errRecords[0].Stage == "" {
		t.Fatalf("the fatal-bind ERROR record names no stage:\n%s", out)
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
		"--startup-gate-empty=10ms",
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

// TestMain_finalDumpFailureExitsThree pins D-06's distinct failed-final-dump
// code at the binary boundary. The dump directory is replaced by a regular
// file, so the atomic save's os.CreateTemp fails with ENOTDIR for every uid —
// unlike chmod 0500, which a root-run CI would bypass — and the bounded final
// dump cannot land: the process exits 3 with the ERROR record explaining it.
func TestMain_finalDumpFailureExitsThree(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.json")
	udpPort := freeUDPPort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	node := startLoggedNode(t, bin, []string{"SUPKV_LOG_LEVEL=error"},
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+dumpPath,
		"--grpc-addr="+addr,
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	awaitServing(t, addr, servingDeadline)

	// Block the dump path: a regular file where the directory was. ensureDir
	// stats the path and finds something, then os.CreateTemp fails with
	// ENOTDIR for every uid (RESEARCH D-06's root-proof construction).
	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("remove dump directory %s: %v", dir, err)
	}
	err = os.WriteFile(dir, []byte("blocked"), 0o600)
	if err != nil {
		t.Fatalf("replace dump directory %s with a regular file: %v", dir, err)
	}

	// A write whose covering dump cannot land answers Unavailable and leaves
	// the engine dirty, so shutdown's bounded final dump has one to attempt.
	client := apiproto.NewSupKVServiceClient(supKVConn(t, addr))
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	t.Cleanup(cancel)
	_, setErr := client.Set(ctx, &apiproto.SetRequest{Key: smokeKey, Value: []byte(smokeValue)})
	if grpcstatus.Code(setErr) != codes.Unavailable {
		t.Fatalf("Set over a blocked dump path = %v, want Unavailable (the setup needs a dirty engine)", setErr)
	}

	err = node.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("SIGTERM %d: %v", node.cmd.Process.Pid, err)
	}
	var exitErr error
	waitFor(t, exitDeadline, "process exit after SIGTERM over a blocked dump path", func() bool {
		select {
		case exitErr = <-node.exit:
			return true
		default:
			return false
		}
	})
	assertExitCode(t, exitErr, exitDumpFailedCode)

	explained := false
	for _, record := range parseLogRecords(node.stderr.String()) {
		if record.Level == "ERROR" && record.Msg == "Final dump failed" {
			explained = true
			break
		}
	}
	if !explained {
		t.Fatalf("the failed-final-dump run left no parsed ERROR %q record:\n%s", "Final dump failed", node.stderr.String())
	}
}

// TestMain_socketBuffersLogged pins XPORT-05's startup evidence end-to-end:
// the real binary logs one static "Socket buffers" line carrying positive
// recv_buf and send_buf attrs — the kernel's EFFECTIVE sizes, so the test
// never asserts an exact number (the kernel doubles and clamps; RESEARCH
// Pitfall 13).
func TestMain_socketBuffersLogged(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	udpPort := freeUDPPort(t)
	grpcPort := freeTCPPort(t)

	node := startLoggedNode(t, bin, []string{"SUPKV_LOG_FORMAT=json", "SUPKV_LOG_LEVEL=info"},
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+dumpPath,
		"--grpc-addr="+fmt.Sprintf("127.0.0.1:%d", grpcPort),
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	waitFor(t, servingDeadline, "the static Socket buffers startup line with positive sizes", func() bool {
		recv, send, ok := parseSocketBuffersLine(node.stderr.String())
		return ok && recv > 0 && send > 0
	})
	recv, send, ok := parseSocketBuffersLine(node.stderr.String())
	if !ok || recv <= 0 || send <= 0 {
		t.Fatalf("Socket buffers line = (recv %d, send %d, ok %t), want both positive", recv, send, ok)
	}
	t.Logf("startup logged effective socket buffers: recv_buf=%d send_buf=%d", recv, send)

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the socket-buffers startup run")
}

// TestMain_gateOpenLogged pins the gate catalog row at the binary boundary
// (D-01): a lone real node's startup gate opens on its backstop — no listing
// can ever arrive — and the process emits exactly one INFO "Startup gate
// opened" record carrying reason=backstop, then still exits 0 on SIGTERM.
func TestMain_gateOpenLogged(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	udpPort := freeUDPPort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))

	node := startLoggedNode(t, bin, []string{"SUPKV_LOG_FORMAT=json", "SUPKV_LOG_LEVEL=info"},
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+dumpPath,
		"--grpc-addr="+addr,
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	awaitServing(t, addr, servingDeadline)
	waitFor(t, servingDeadline, "the one Startup gate opened record", func() bool {
		return len(gateOpenRecords(node.stderr.String())) == 1
	})
	records := gateOpenRecords(node.stderr.String())
	if len(records) != 1 {
		t.Fatalf("the run emitted %d Startup gate opened records, want exactly 1:\n%s", len(records), node.stderr.String())
	}
	if records[0].Level != "INFO" {
		t.Fatalf("the Startup gate opened record level = %q, want INFO:\n%s", records[0].Level, node.stderr.String())
	}
	if records[0].Reason != "backstop" {
		t.Fatalf("the Startup gate opened record reason = %q, want backstop:\n%s", records[0].Reason, node.stderr.String())
	}

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the gate-open run")
}

// gateOpenRecords returns the parsed records with the gate catalog's static
// message.
func gateOpenRecords(output string) []logRecord {
	var records []logRecord
	for _, record := range parseLogRecords(output) {
		if record.Msg == "Startup gate opened" {
			records = append(records, record)
		}
	}
	return records
}

// TestMain_startupLogRecord pins the D-01 startup record at the binary
// boundary: one parsed "Starting sup-kv" INFO record carries the build
// identity and the effective configuration, and no certificate/key key ever
// reaches it (T-07-21).
func TestMain_startupLogRecord(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	udpPort := freeUDPPort(t)
	bind := "0.0.0.0:" + strconv.Itoa(udpPort)
	addr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))

	node := startLoggedNode(t, bin, []string{"SUPKV_LOG_FORMAT=json", "SUPKV_LOG_LEVEL=info"},
		"--node-id=1",
		"--bind-addr="+bind,
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+dumpPath,
		"--grpc-addr="+addr,
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	awaitServing(t, addr, servingDeadline)

	var record map[string]any
	waitFor(t, servingDeadline, "the Starting sup-kv record", func() bool {
		found, ok := findLogMap(node.stderr.String(), "Starting sup-kv")
		record = found
		return ok
	})
	if got, ok := record["level"]; !ok || got != "INFO" {
		t.Fatalf("the startup record level = %v (present %t), want INFO:\n%v", got, ok, record)
	}
	for key, want := range map[string]any{
		"version":        "dev",
		"node_id":        float64(1),
		"bind_addr":      bind,
		"dump_path":      dumpPath,
		"max_clock_lead": "2m0s",
		"metrics_tls":    false,
	} {
		if got, ok := record[key]; !ok || got != want {
			t.Fatalf("startup record[%q] = %v (present %t), want %v:\n%v", key, got, ok, want, record)
		}
	}
	if got, ok := record["go_version"]; !ok || got == "" {
		t.Fatalf("startup record[go_version] = %v (present %t), want a non-empty runtime version:\n%v", got, ok, record)
	}
	for _, forbidden := range []string{"metrics_cert", "metrics_key", "metrics_client_ca"} {
		if _, present := record[forbidden]; present {
			t.Fatalf("the startup record carries the secret-bearing key %q, want it absent:\n%v", forbidden, record)
		}
	}

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the startup-record run")
}

// TestMain_startupLogLoadOutcome pins the D-01 load-outcome record across a
// real restart: a fresh directory logs exactly one INFO "Dump load" with
// outcome=prompt and records=0 and, after the first run's final dump carries
// a committed key, the second run logs exactly one with outcome=main and
// records >= 1.
func TestMain_startupLogLoadOutcome(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	env := []string{"SUPKV_LOG_FORMAT=json", "SUPKV_LOG_LEVEL=info"}
	flags := func(addr string) []string {
		return []string{
			"--node-id=1",
			"--bind-addr=0.0.0.0:" + strconv.Itoa(freeUDPPort(t)),
			"--broadcast-addr=127.255.255.255:" + strconv.Itoa(freeUDPPort(t)),
			"--dump-path=" + dumpPath,
			"--grpc-addr=" + addr,
			"--startup-gate=10ms",
			"--startup-gate-empty=10ms",
		}
	}

	firstAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	first := startLoggedNode(t, bin, env, flags(firstAddr)...)
	awaitServing(t, firstAddr, servingDeadline)
	waitFor(t, servingDeadline, "the fresh start's Dump load record", func() bool {
		return countLogMaps(first.stderr.String(), "Dump load") == 1
	})
	prompt := assertOneLoadRecord(t, first.stderr.String(), "prompt")
	if got := prompt["records"]; got != float64(0) {
		t.Fatalf("fresh-start Dump load records = %v, want 0", got)
	}

	// Commit one key so the first run's final dump carries a record for the
	// restart to load.
	client := apiproto.NewSupKVServiceClient(supKVConn(t, firstAddr))
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	t.Cleanup(cancel)
	_, err := client.Set(ctx, &apiproto.SetRequest{Key: smokeKey, Value: []byte(smokeValue)})
	if err != nil {
		t.Fatalf("Set(%q): %v", smokeKey, err)
	}
	signalAndWaitExit(t, first)
	assertNoPanic(t, first, "the fresh-start run")

	secondAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	second := startLoggedNode(t, bin, env, flags(secondAddr)...)
	awaitServing(t, secondAddr, servingDeadline)
	waitFor(t, servingDeadline, "the restart's Dump load record", func() bool {
		return countLogMaps(second.stderr.String(), "Dump load") == 1
	})
	mainRecord := assertOneLoadRecord(t, second.stderr.String(), "main")
	if got, ok := mainRecord["records"].(float64); !ok || got < 1 {
		t.Fatalf("restart Dump load records = %v, want at least 1 — the first run's dump must be loaded", mainRecord["records"])
	}
	signalAndWaitExit(t, second)
	assertNoPanic(t, second, "the load-outcome restart")
}

// assertOneLoadRecord requires exactly one parsed "Dump load" record and the
// expected outcome token, returning its attributes for field assertions.
func assertOneLoadRecord(t *testing.T, output, wantOutcome string) map[string]any {
	t.Helper()
	var records []map[string]any
	for _, record := range parseLogMaps(output) {
		if record["msg"] == "Dump load" {
			records = append(records, record)
		}
	}
	if len(records) != 1 {
		t.Fatalf("%d Dump load record(s), want exactly 1:\n%s", len(records), output)
	}
	if got := records[0]["outcome"]; got != wantOutcome {
		t.Fatalf("Dump load outcome = %v, want %q:\n%v", got, wantOutcome, records[0])
	}
	return records[0]
}

// countLogMaps returns how many parsed records carry msg.
func countLogMaps(output, msg string) int {
	count := 0
	for _, record := range parseLogMaps(output) {
		if record["msg"] == msg {
			count++
		}
	}
	return count
}

// parseLogMaps parses every JSON-object line of output into its raw
// attribute map — the shape field-level assertions read when a fixed struct
// would not cover every catalog key.
func parseLogMaps(output string) []map[string]any {
	var records []map[string]any
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) != nil {
			continue
		}
		records = append(records, record)
	}
	return records
}

// findLogMap returns the first parsed record whose msg equals msg.
func findLogMap(output, msg string) (map[string]any, bool) {
	for _, record := range parseLogMaps(output) {
		if record["msg"] == msg {
			return record, true
		}
	}
	return nil, false
}

// parseSocketBuffersLine extracts recv_buf and send_buf from the startup
// "Socket buffers" JSON log line. ok reports the line appeared with both
// integer attrs present; the caller asserts their positivity.
func parseSocketBuffersLine(output string) (recv, send int, ok bool) {
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(line, "Socket buffers") {
			continue
		}
		var entry struct {
			Msg     string `json:"msg"`
			RecvBuf *int   `json:"recv_buf"`
			SendBuf *int   `json:"send_buf"`
		}
		if json.Unmarshal([]byte(line), &entry) != nil || entry.Msg != "Socket buffers" {
			continue
		}
		if entry.RecvBuf == nil || entry.SendBuf == nil {
			continue
		}
		return *entry.RecvBuf, *entry.SendBuf, true
	}
	return 0, 0, false
}

// logRecord is the one JSON record shape the lifecycle-log tests parse: the
// level and static message every handler emits, the optional stage a startup
// failure names (D-06), and the gate-open reason (D-01).
type logRecord struct {
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
}

// parseLogRecords parses every JSON-object line of output, skipping the
// non-JSON lines a usage error or the binary's own stdout may carry.
func parseLogRecords(output string) []logRecord {
	var records []logRecord
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var record logRecord
		if json.Unmarshal([]byte(line), &record) != nil {
			continue
		}
		records = append(records, record)
	}
	return records
}

// TestShutdown_secondSignal pins the P1-IN-01 contract end-to-end: signal
// ownership stays armed across the whole D-19 shutdown, so a second SIGTERM
// observed while the graceful drain is still in progress takes the force
// path instead of killing the process — the final write-confirmed dump
// still lands and the node still exits 0. The drain is held observably open
// by a standard health Watch stream: unlike the SupKV Watch (ended by
// apiCancel ahead of GracefulStop — the D-19 order), the health Watch is
// not tied to the API context, so GracefulStop waits on it until the client
// goes away or the server is forced. Against the pre-fix binary the same
// second signal kills the process with the default disposition (exit 143)
// and the final dump never happens.
func TestShutdown_secondSignal(t *testing.T) {
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
		"--startup-gate-empty=10ms",
	)

	// The stream GracefulStop waits on. A reader goroutine drains the
	// status updates (the flow-control window never fills), and the
	// statuses observed on it prove where the server is in the D-19
	// sequence — the initial SERVING doubles as the gate-open wait the
	// other tests use awaitServing for.
	wctx, wcancel := context.WithCancel(t.Context())
	t.Cleanup(wcancel)
	// WaitForReady: the stream may open before the node's gRPC listener is
	// up — the call then waits for the connection instead of failing fast.
	wstream, err := healthpb.NewHealthClient(supKVConn(t, addr)).Watch(wctx, &healthpb.HealthCheckRequest{}, grpc.WaitForReady(true))
	if err != nil {
		t.Fatalf("health Watch: %v", err)
	}
	statuses := make(chan healthpb.HealthCheckResponse_ServingStatus, 2)
	go func() {
		defer close(statuses)
		for {
			st, err := wstream.Recv()
			if err != nil {
				return
			}
			statuses <- st.GetStatus()
		}
	}()
	waitForStatus := func(want healthpb.HealthCheckResponse_ServingStatus, what string) {
		t.Helper()
		waitFor(t, servingDeadline, what, func() bool {
			select {
			case st, ok := <-statuses:
				return ok && st == want
			default:
				return false
			}
		})
	}
	waitForStatus(healthpb.HealthCheckResponse_SERVING, "the initial SERVING health status")

	// The earlier write the final dump must still contain after the forced
	// drain.
	client := apiproto.NewSupKVServiceClient(supKVConn(t, addr))
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	t.Cleanup(cancel)
	_, err = client.Set(ctx, &apiproto.SetRequest{Key: smokeKey, Value: []byte(smokeValue)})
	if err != nil {
		t.Fatalf("Set(%q): %v", smokeKey, err)
	}

	// SIGTERM #1: the D-19 sequence starts — its first statement flips the
	// health status, which arrives on the stream while GracefulStop is (and
	// stays) blocked on the still-open Watch.
	err = node.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("first SIGTERM %d: %v", node.cmd.Process.Pid, err)
	}
	waitForStatus(healthpb.HealthCheckResponse_NOT_SERVING, "NOT_SERVING after the first SIGTERM (the graceful drain in progress)")

	// SIGTERM #2 while the drain is blocked: the operator's hurry-up taking
	// the force path.
	err = node.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("second SIGTERM %d: %v", node.cmd.Process.Pid, err)
	}

	var exitErr error
	waitFor(t, exitDeadline, "process exit after the second SIGTERM", func() bool {
		select {
		case exitErr = <-node.exit:
			return true
		default:
			return false
		}
	})
	assertExitCode(t, exitErr, 0)
	assertNoPanic(t, node, "the second-signal shutdown run")

	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read dump %s: %v", dumpPath, err)
	}
	if !strings.Contains(string(dump), smokeKey) {
		t.Fatalf("the final dump at %s does not contain the committed key %q:\n%s", dumpPath, smokeKey, dump)
	}
}

// Structured-logging test constants (D-06/D-07/D-08): the broadcast
// capability probe's wait, and the D-06 exit table observed at the binary
// boundary — 1 for a startup or Serve failure, 2 for a usage/configuration
// error (the collected env-error path the malformed SUPKV_LOG_* values must
// abort through, never a silent default) and 3 for a failed final dump.
const (
	logProbeWait        = 300 * time.Millisecond
	exitStartupCode     = 1
	exitUsageCode       = 2
	exitDumpFailedCode  = 3
	datagramCeilingLine = "datagram ceiling"

	// startupGateRangeProbe bounds the non-positive startup-gate rows: the
	// validation is expected to reject before any bind, so the bound only
	// stops a RED-state binary (which would serve until the runner's kill)
	// from hanging the test.
	startupGateRangeProbe = 3 * time.Second

	// valueMaxRangeProbe bounds the non-positive value-max-bytes rows for the
	// same reason: the validation rejects before any bind, and the bound only
	// stops a RED-state binary from serving until the runner's kill.
	valueMaxRangeProbe = 3 * time.Second

	// validationRelationProbe bounds the relation/positivity process rows for
	// the same reason: every one of them must exit 2 before any bind, so the
	// bound only stops a RED-state binary (which would start and serve) from
	// hanging the test.
	validationRelationProbe = 3 * time.Second

	// The empty-vs-data selection bounds: the empty node must still be closed
	// this far past a 50 ms normal gate, and the seeded node must open inside
	// this bound — far below the 1500 ms empty backstop.
	startupGateSelectionProbe = 300 * time.Millisecond
	startupGateSeededBound    = 300 * time.Millisecond

	// startupBoundaryValue is the largest --value-max-bytes whose maximal-key
	// record encodes to exactly 65 507 bytes; equality passes because the
	// startup check is `>`. One byte more crosses the ceiling.
	startupBoundaryValue     = 64408
	startupBoundaryOverValue = startupBoundaryValue + 1
)

// logProbeSocket binds a wildcard udp4 socket with SO_REUSEADDR and
// SO_BROADCAST set before bind — the adapter's exact socket recipe, so the
// capability probe measures the production topology (the smoke_test.go and
// wait_integration_test.go precedent).
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

// probeLoopbackBroadcast pre-flights the one platform capability the
// two-node log tests need — a datagram sent to the loopback broadcast
// address reaches the socket bound to the destination port (loopback
// broadcast fan-out is a Linux-only kernel behavior; macOS lo0 carries no
// BROADCAST flag — the wait_integration_test.go probe, D-12) — and skips
// loudly naming what was probed when it is missing. The env-validation half
// of the logging contract stays proven on every platform.
func probeLoopbackBroadcast(t *testing.T, port int) {
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
	dst := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: port}
	_, err = tx.WriteToUDP([]byte("sup-kv log smoke probe"), dst)
	if err != nil {
		t.Fatalf("probe send to %s: %v", dst, err)
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
				"loopback broadcast to %s did not reach the bound socket within %v — platform lacks the capability (the env-validation half of the logging contract stays proven by TestLog_envValidation)",
				dst,
				logProbeWait,
			)
		}
		if n > 0 {
			return
		}
	}
}

// startLogPair launches the two-node asymmetric pair (research Pattern 5)
// with BOTH nodes claiming node-id 1: A binds portA and broadcasts to
// portB, B binds portB and broadcasts to portA — cross-delivery without
// self-echo — so every Hello crossing the pair is the DuplicateNodeID
// configuration error (REPL-08), the driven log event. Node A's stderr is
// the lock-guarded live buffer; envA is prepended to A's environment only.
func startLogPair(t *testing.T, bin string, envA []string, portA, portB int) (nodeA, nodeB *smokeNode) {
	t.Helper()
	grpcA, grpcB := freeTCPPort(t), freeTCPPort(t)
	nodeA = startLoggedNode(t, bin, envA,
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(portA),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(portB),
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
		"--grpc-addr=127.0.0.1:"+strconv.Itoa(grpcA),
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	nodeB = startNode(t, bin, nil,
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(portB),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(portA),
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
		"--grpc-addr=127.0.0.1:"+strconv.Itoa(grpcB),
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	return nodeA, nodeB
}

// TestLog_jsonDefault pins D-06/D-07's default contract on the real binary:
// with SUPKV_LOG_FORMAT unset every stderr line the running node emits is
// one JSON object carrying non-empty level and msg keys, and the driven
// event — the DuplicateNodeID configuration error crossing the asymmetric
// pair — rides as the detail attr under the static tap msg, never as the
// msg itself (D-08, T-6-05: the attr surface carries metadata only).
func TestLog_jsonDefault(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	portA, portB := freeUDPPort(t), freeUDPPort(t)
	probeLoopbackBroadcast(t, portB)
	nodeA, nodeB := startLogPair(t, bin, nil, portA, portB)

	waitFor(t, servingDeadline, "the DuplicateNodeID event in node A's stderr", func() bool {
		return strings.Contains(nodeA.stderr.String(), "configuration error")
	})

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the JSON-format run")

	sawDetailEvent := false
	for line := range strings.SplitSeq(nodeA.stderr.String(), "\n") {
		if line == "" {
			continue
		}
		var rec struct {
			Level  string `json:"level"`
			Msg    string `json:"msg"`
			Detail string `json:"detail"`
		}
		err := json.Unmarshal([]byte(line), &rec)
		if err != nil {
			t.Fatalf("stderr line %q is not one JSON object: %v", line, err)
		}
		if rec.Level == "" || rec.Msg == "" {
			t.Fatalf("stderr line %q lacks a non-empty level or msg key: %+v", line, rec)
		}
		if rec.Msg == "Replication adapter" && strings.Contains(rec.Detail, "configuration error") {
			sawDetailEvent = true
			// The level is part of the pinned catalog contract: the adapter
			// tap renders the Spec-mandated duplicate-node-id entry at WARN,
			// so a demoted tap fails here as well as in the level matrix.
			if rec.Level != "WARN" {
				t.Fatalf("the driven adapter record level = %q, want WARN: %+v", rec.Level, rec)
			}
			if !strings.Contains(rec.Detail, "node_id 1") {
				t.Fatalf("the driven event's detail %q does not name node_id 1", rec.Detail)
			}
		}
	}
	if !sawDetailEvent {
		t.Fatalf("no Replication adapter record carrying the configuration error appeared in:\n%s", nodeA.stderr.String())
	}
}

// TestLog_formatText pins the SUPKV_LOG_FORMAT=text half of D-06: the same
// driven event surfaces as key=value text — no JSON object on stderr — with
// the prebuilt message still riding the detail attr under the static msg.
func TestLog_formatText(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	portA, portB := freeUDPPort(t), freeUDPPort(t)
	probeLoopbackBroadcast(t, portB)
	nodeA, nodeB := startLogPair(t, bin, []string{"SUPKV_LOG_FORMAT=text"}, portA, portB)

	waitFor(t, servingDeadline, "the DuplicateNodeID event in node A's stderr", func() bool {
		return strings.Contains(nodeA.stderr.String(), "configuration error")
	})

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the text-format run")

	sawDetailEvent := false
	for line := range strings.SplitSeq(nodeA.stderr.String(), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "{") {
			t.Fatalf("text-format stderr line %q is a JSON object, want key=value text", line)
		}
		if strings.Contains(line, `msg="Replication adapter"`) && strings.Contains(line, `detail="sup-kv: configuration error`) {
			sawDetailEvent = true
			if !strings.Contains(line, "node_id 1") {
				t.Fatalf("the driven event's line %q does not name node_id 1", line)
			}
		}
	}
	if !sawDetailEvent {
		t.Fatalf("no Replication adapter text record carrying the configuration error appeared in:\n%s", nodeA.stderr.String())
	}
}

// TestLog_envValidation pins the D-07 fail-closed contract: a malformed
// SUPKV_LOG_LEVEL or SUPKV_LOG_FORMAT value aborts startup through the
// collected env-error path with exitUsage — never a silent default.
func TestLog_envValidation(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)
	for _, tc := range []struct{ name, env string }{
		{name: "malformed level", env: "SUPKV_LOG_LEVEL=bogus"},
		{name: "malformed format", env: "SUPKV_LOG_FORMAT=bogus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			udpPort := freeUDPPort(t)
			node := startNode(t, bin, []string{tc.env},
				"--node-id=1",
				"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
				"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
				"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
			)
			var exitErr error
			waitFor(t, exitDeadline, "process exit for "+tc.env, func() bool {
				select {
				case exitErr = <-node.exit:
					return true
				default:
					return false
				}
			})
			assertExitCode(t, exitErr, exitUsageCode)
		})
	}
}

// httpGetBody GETs url and returns the status with the body — the metrics
// tests' scrape helper (the noctx-clean request shape).
func httpGetBody(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext(%s): %v", url, err)
	}
	client := &http.Client{Timeout: rpcTimeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		t.Fatalf("read body of %s: %v", url, readErr)
	}
	if closeErr != nil {
		t.Fatalf("close body of %s: %v", url, closeErr)
	}
	return resp.StatusCode, string(body)
}

// TestMetrics_optIn pins D-01's set half end-to-end on the real binary: a
// node started with --metrics-addr serves /metrics over plain HTTP — the
// loopback/debug mode D-02 documents — and the scrape carries the full
// supkv_ family surface over the 06-03 seams plus the runtime families.
func TestMetrics_optIn(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	udpPort := freeUDPPort(t)
	grpcPort := freeTCPPort(t)
	metricsPort := freeTCPPort(t)
	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
	metricsURL := fmt.Sprintf("http://127.0.0.1:%d/metrics", metricsPort)

	node := startNode(t, bin, nil,
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
		"--grpc-addr="+grpcAddr,
		"--metrics-addr=127.0.0.1:"+strconv.Itoa(metricsPort),
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	awaitServing(t, grpcAddr, servingDeadline)

	status, body := httpGetBody(t, metricsURL)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", metricsURL, status, http.StatusOK)
	}
	for _, name := range []string{
		"supkv_replication_unacked_records",
		"supkv_replication_unacked_oldest_age_seconds",
		"supkv_replication_peers",
		"supkv_replication_datagrams_sent_total",
		"supkv_replication_datagrams_retransmitted_total",
		"supkv_replication_datagrams_paced_total",
		"supkv_replication_send_queue",
		"supkv_replication_egress_dropped_total",
		"supkv_replication_read_errors_total",
		"supkv_datagrams_discarded_total",
		"supkv_merges_conflicts_total",
		"supkv_clock_drift_warnings_total",
		"supkv_dumps_duration_seconds",
		"supkv_dumps_total",
		"supkv_dumps_errors_total",
		"supkv_dumps_failures_total",
		"supkv_dumps_unconfirmed_total",
		"supkv_dump_load_total",
		"supkv_dump_records_discarded_total",
		"go_goroutines",
		"process_cpu_seconds_total",
	} {
		if !strings.Contains(body, name) {
			t.Errorf("the scrape of %s lacks the family %q:\n%s", metricsURL, name, body)
		}
	}

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the opt-in metrics run")
}

// TestMetrics_noDefaultListener pins D-01's unset half discriminatingly. A
// refused dial on an arbitrary unpassed port would prove nothing — any
// implementation refuses there, including one with a default metrics
// listener on a fixed port, the exact D-01 regression this test exists to
// catch — so the assertion runs through a collision instead: two concurrent
// nodes, neither given --metrics-addr nor SUPKV_METRICS_ADDR, mutually
// isolated (distinct node ids, UDP bind/broadcast ports, dump dirs and
// gRPC ports), both must reach SERVING and both must exit 0. Under the
// regression — a default metrics address introduced the way defaultGRPCAddr
// was, a non-empty defaults-block value flowing through wire() — the second
// node's fixed-port metrics bind collides with the first node's, the fatal
// bind aborts its startup and its awaitServing times out. The mechanism is
// pure loopback TCP with per-node ports: portable to every matrix cell,
// deliberately not /proc or socket enumeration.
//
// Decision 14.8 (ADR 0008): metrics stay off by default — this is the
// binary-level pin of that decision (no listener without --metrics-addr or
// SUPKV_METRICS_ADDR); the rendering half is TestStand_metricsByEffect.
func TestMetrics_noDefaultListener(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	launch := func(nodeID int) (node *smokeNode, grpcAddr string) {
		udpPort := freeUDPPort(t)
		grpcPort := freeTCPPort(t)
		return startNode(t, bin, nil,
			"--node-id="+strconv.Itoa(nodeID),
			"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
			"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
			"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
			"--grpc-addr=127.0.0.1:"+strconv.Itoa(grpcPort),
			"--startup-gate=10ms",
			"--startup-gate-empty=10ms",
		), fmt.Sprintf("127.0.0.1:%d", grpcPort)
	}
	nodeA, addrA := launch(1)
	nodeB, addrB := launch(2)

	awaitServing(t, addrA, servingDeadline)
	awaitServing(t, addrB, servingDeadline)
	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
}

// TestMetrics_incompleteCertSet pins D-02's fail-closed contract: one
// certificate env set without the other two aborts startup through the
// collected-error path with exitUsage — before any bind (Pitfall 8: no path
// can reach the TLS builder with a half-loaded set) — and the message names
// all three envs of the triple. The cert envs have no flags, so the usage
// dump cannot mention them: only the error line can carry the names.
func TestMetrics_incompleteCertSet(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	udpPort := freeUDPPort(t)
	node := startNode(t, bin, []string{"SUPKV_METRICS_CERT=" + filepath.Join(t.TempDir(), "server-cert.pem")},
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
	)
	var exitErr error
	waitFor(t, exitDeadline, "process exit for the incomplete cert set", func() bool {
		select {
		case exitErr = <-node.exit:
			return true
		default:
			return false
		}
	})
	assertExitCode(t, exitErr, exitUsageCode)
	stderr := node.stderr.String()
	for _, env := range []string{"SUPKV_METRICS_CERT", "SUPKV_METRICS_KEY", "SUPKV_METRICS_CLIENT_CA"} {
		if !strings.Contains(stderr, env) {
			t.Errorf("stderr of the incomplete-cert-set run lacks %q:\n%s", env, stderr)
		}
	}
}

// metricsCertSuite is the in-test PKI the mTLS tests drive: one self-signed
// CA, a server leaf valid for the loopback IPs, and a client leaf — all
// generated at test runtime with crypto/x509 + crypto/ecdsa, no committed
// fixtures and no openssl (06-RESEARCH Pattern 3).
type metricsCertSuite struct {
	serverCertFile string
	serverKeyFile  string
	caFile         string
	caPool         *x509.CertPool
	clientCert     tls.Certificate
}

// generateMetricsCerts writes the suite's CA and server pair into dir and
// keeps the client pair in memory. Validity spans an hour around now so a
// slow cell never trips NotYetValid or Expired.
func generateMetricsCerts(t *testing.T, dir string) metricsCertSuite {
	t.Helper()
	notBefore := time.Now().Add(-time.Hour)
	notAfter := time.Now().Add(time.Hour)

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "sup-kv-metrics-test-ca"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}
	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	newLeaf := func(serial int64, commonName string, usage []x509.ExtKeyUsage, ips []net.IP) ([]byte, *ecdsa.PrivateKey) {
		key, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if keyErr != nil {
			t.Fatalf("generate %s key: %v", commonName, keyErr)
		}
		template := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: commonName},
			NotBefore:    notBefore,
			NotAfter:     notAfter,
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  usage,
			IPAddresses:  ips,
		}
		der, crtErr := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
		if crtErr != nil {
			t.Fatalf("create %s certificate: %v", commonName, crtErr)
		}
		return der, key
	}
	serverDER, serverKey := newLeaf(2, "sup-kv-metrics-test-server",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		[]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")})
	clientDER, clientKey := newLeaf(3, "sup-kv-metrics-test-client",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, nil)

	serverKeyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		t.Fatalf("marshal server key: %v", err)
	}
	write := func(name string, data []byte) string {
		path := filepath.Join(dir, name)
		wErr := os.WriteFile(path, data, 0o600)
		if wErr != nil {
			t.Fatalf("write %s: %v", path, wErr)
		}
		return path
	}
	certPEM := func(der []byte) []byte {
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	}
	return metricsCertSuite{
		serverCertFile: write("server-cert.pem", certPEM(serverDER)),
		serverKeyFile: write("server-key.pem",
			pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: serverKeyDER})),
		caFile:     write("ca.pem", certPEM(caDER)),
		caPool:     caPool,
		clientCert: tls.Certificate{Certificate: [][]byte{clientDER}, PrivateKey: clientKey},
	}
}

// startMTLSNode starts the real binary with the complete certificate triple
// and --metrics-addr, waiting for the gRPC gate — the shared setup of the
// mTLS enforcement and debug-surface tests.
func startMTLSNode(t *testing.T, bin string, certs metricsCertSuite, metricsPort int) (node *smokeNode, metricsAddr string) {
	t.Helper()
	udpPort := freeUDPPort(t)
	grpcPort := freeTCPPort(t)
	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
	metricsAddr = fmt.Sprintf("127.0.0.1:%d", metricsPort)
	node = startNode(t, bin, []string{
		"SUPKV_METRICS_CERT=" + certs.serverCertFile,
		"SUPKV_METRICS_KEY=" + certs.serverKeyFile,
		"SUPKV_METRICS_CLIENT_CA=" + certs.caFile,
	},
		"--node-id=1",
		"--bind-addr=0.0.0.0:"+strconv.Itoa(udpPort),
		"--broadcast-addr=127.255.255.255:"+strconv.Itoa(udpPort),
		"--dump-path="+filepath.Join(t.TempDir(), "dump.json"),
		"--grpc-addr="+grpcAddr,
		"--metrics-addr="+metricsAddr,
		"--startup-gate=10ms",
		"--startup-gate-empty=10ms",
	)
	awaitServing(t, grpcAddr, servingDeadline)
	return node, metricsAddr
}

// tlsGet fetches url over the suite's authorized client — the mTLS read
// path of the enforcement and debug-surface tests.
func tlsGet(t *testing.T, certs metricsCertSuite, url string) (status int, contentType, body string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext(%s): %v", url, err)
	}
	client := &http.Client{
		Timeout: rpcTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      certs.caPool,
				Certificates: []tls.Certificate{certs.clientCert},
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("authorized GET %s: %v", url, err)
	}
	raw, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		t.Fatalf("read body of %s: %v", url, readErr)
	}
	if closeErr != nil {
		t.Fatalf("close body of %s: %v", url, closeErr)
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(raw)
}

// TestMetricsTLS_required pins D-02's enforcement half on the real binary:
// with the full certificate triple the listener speaks TLS and requires a
// verified client certificate — a plain HTTP GET fails against it and a TLS
// client without a certificate fails the handshake — while the client pair
// generated in-test scrapes both /metrics and /debug/vars (D-03's one
// surface). The closing SIGTERM walk exercises the metrics Shutdown in its
// D-19 position with TLS active.
func TestMetricsTLS_required(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)
	certs := generateMetricsCerts(t, t.TempDir())
	node, metricsAddr := startMTLSNode(t, bin, certs, freeTCPPort(t))

	plainReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+metricsAddr+"/metrics", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	plainClient := &http.Client{Timeout: rpcTimeout}
	// A plain HTTP GET fails against the TLS listener. Go's stack answers
	// plaintext-to-TLS with a 400 "Client sent an HTTP request to an HTTPS
	// server" (net/http's record-header probe), so the failure surfaces as
	// that rejection or as a transport error — never as a served body.
	plainResp, plainErr := plainClient.Do(plainReq)
	if plainErr == nil {
		raw, readErr := io.ReadAll(plainResp.Body)
		closeErr := plainResp.Body.Close()
		if readErr != nil {
			t.Fatalf("read the plain-HTTP body: %v", readErr)
		}
		if closeErr != nil {
			t.Fatalf("close the plain-HTTP body: %v", closeErr)
		}
		if plainResp.StatusCode == http.StatusOK || strings.Contains(string(raw), "supkv_") {
			t.Fatalf("a plain HTTP GET against %s answered with the metrics surface (status %d) — the cert-triple listener is not speaking TLS:\n%s",
				metricsAddr, plainResp.StatusCode, raw)
		}
	}

	noCertReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+metricsAddr+"/metrics", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	noCertClient := &http.Client{
		Timeout:   rpcTimeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: certs.caPool}},
	}
	_, handshakeErr := noCertClient.Do(noCertReq)
	if handshakeErr == nil {
		t.Fatalf("a TLS client without a certificate scraped %s — RequireAndVerifyClientCert is not enforced", metricsAddr)
	}

	status, _, body := tlsGet(t, certs, "https://"+metricsAddr+"/metrics")
	if status != http.StatusOK {
		t.Fatalf("authorized GET /metrics status = %d, want %d", status, http.StatusOK)
	}
	if !strings.Contains(body, "supkv_replication_unacked_records") {
		t.Fatalf("the authorized /metrics scrape lacks the supkv_ family surface:\n%s", body)
	}
	status, _, body = tlsGet(t, certs, "https://"+metricsAddr+"/debug/vars")
	if status != http.StatusOK {
		t.Fatalf("authorized GET /debug/vars status = %d, want %d", status, http.StatusOK)
	}
	if !strings.Contains(body, `"cmdline"`) {
		t.Fatalf("the authorized /debug/vars body lacks expvar's cmdline key:\n%s", body)
	}

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the mTLS metrics run")
}

// TestDebugSurface pins D-03's debug half behind the same mTLS: the pprof
// index page answers the authorized client, and a named profile route —
// heap, mounted through pprof.Handler — returns profile content on the same
// listener as /metrics.
func TestDebugSurface(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)
	certs := generateMetricsCerts(t, t.TempDir())
	node, metricsAddr := startMTLSNode(t, bin, certs, freeTCPPort(t))

	status, _, index := tlsGet(t, certs, "https://"+metricsAddr+"/debug/pprof/")
	if status != http.StatusOK {
		t.Fatalf("authorized GET /debug/pprof/ status = %d, want %d", status, http.StatusOK)
	}
	if !strings.Contains(index, "heap") {
		t.Fatalf("the pprof index page does not list the heap profile:\n%s", index)
	}

	status, contentType, heap := tlsGet(t, certs, "https://"+metricsAddr+"/debug/pprof/heap")
	if status != http.StatusOK {
		t.Fatalf("authorized GET /debug/pprof/heap status = %d, want %d", status, http.StatusOK)
	}
	if heap == "" {
		t.Fatalf("the heap profile at %s is empty", metricsAddr)
	}
	if !strings.HasPrefix(contentType, "application/octet-stream") {
		t.Fatalf("heap profile Content-Type = %q, want application/octet-stream", contentType)
	}

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the debug-surface run")
}
