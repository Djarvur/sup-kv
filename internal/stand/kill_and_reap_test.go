//go:build linux

// The kill/readiness diagnosis rows (08-09 Task 1, F-6): a child that
// exited on its own between the confirmed Set and the kill must explain
// itself — pid, Wait status and the captured output tails — instead of the
// bare "os: process already finished" the kill returned. The genuine
// SIGKILL path stays unchanged and still asserts the harness's -1 exit
// shape, and a readiness wait for a freshly spawned child fails immediately
// when the child has already exited: a crashed child can never look like a
// slow one.

package stand //nolint:testpackage // the diagnosis rows exercise unexported harness helpers (reapAfterKill, exitDiagnosis, awaitNodeServing); testpackage's external-package rule cannot express that pairing.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// earlyExitReadinessBound bounds the dead-child readiness assertion: the
// diagnosis must land on the first poll iteration, far below the serving
// timeout it replaces.
const earlyExitReadinessBound = 2 * time.Second

// waitNodeExited waits for the node's reaper to close exited — the process
// has finished and its Wait result is stored.
func waitNodeExited(t *testing.T, node *standNode, what string) {
	t.Helper()
	waitFor(t, exitDeadline, what, func() bool {
		select {
		case <-node.exited:
			return true
		default:
			return false
		}
	})
}

// TestKillAndReap drives reapAfterKill's three outcomes: an already-exited
// child is diagnosed with its status and captured output (stderr and stdout
// halves), while a live child takes the unchanged SIGKILL path and returns
// nil.
func TestKillAndReap(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)

	// The already-exited window (F-6): the child fails its configuration
	// validation on its own before the kill, so Process.Kill reports
	// os.ErrProcessDone. The diagnosis must carry the evidence instead of
	// the bare sentinel text.
	dead := launchNode(t, p.bin, &bytes.Buffer{}, "--node-id=0")
	waitNodeExited(t, dead, "the invalid-configuration child's exit")
	err := reapAfterKill(t, dead)
	if err == nil {
		t.Fatal("reapAfterKill on an already-exited child = nil, want the exit diagnosis")
	}
	message := err.Error()
	for _, want := range []string{
		"process exited before SIGKILL",
		fmt.Sprintf("process %d", dead.cmd.Process.Pid),
		"exit code 2",
		"stderr tail",
		"required unique 16-bit node ID",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("reapAfterKill diagnosis lacks %q:\n%s", want, message)
		}
	}
	if strings.Contains(message, "want -1") {
		t.Fatalf("reapAfterKill reported the genuine SIGKILL assertion for an already-exited child:\n%s", message)
	}

	// The stdout half: a --version child exits 0 with its identity on
	// stdout, so the diagnosis carries that tail too.
	versioned := launchNode(t, p.bin, &bytes.Buffer{}, "--version")
	waitNodeExited(t, versioned, "the --version child's exit")
	err = reapAfterKill(t, versioned)
	if err == nil {
		t.Fatal("reapAfterKill on the --version child = nil, want the exit diagnosis")
	}
	message = err.Error()
	for _, want := range []string{
		"process exited before SIGKILL",
		"exit 0 (clean)",
		"stdout tail",
		"sup-kv ",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("reapAfterKill stdout diagnosis lacks %q:\n%s", want, message)
		}
	}

	// The genuine path is unchanged: a live node SIGKILLed and reaped
	// returns nil — the -1 assertion holds.
	live := startStandNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	err = reapAfterKill(t, live)
	if err != nil {
		t.Fatalf("reapAfterKill on a live SIGKILLed node = %v, want nil (the genuine -1 path)", err)
	}
}

// TestExitDiagnosis proves the stored Wait result is readable repeatedly:
// nil while the child runs, then the full diagnosis — pid, status and both
// output tails — on every call after it exited, with no channel consumption.
func TestExitDiagnosis(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)

	node := startStandNode(t, p, standNodeAID)
	diagnosis := node.exitDiagnosis()
	if diagnosis != nil {
		t.Fatalf("exitDiagnosis on a running child = %v, want nil", diagnosis)
	}
	awaitServing(t, p.grpcA, servingDeadline)
	signalAndWaitExit(t, node)

	first := node.exitDiagnosis()
	if first == nil {
		t.Fatal("exitDiagnosis after a clean exit = nil, want the exit status")
	}
	second := node.exitDiagnosis()
	if second == nil || second.Error() != first.Error() {
		t.Fatalf("exitDiagnosis second call = %v, want the same diagnosis %q (no channel consumption)", second, first)
	}
	for _, want := range []string{
		fmt.Sprintf("process %d", node.cmd.Process.Pid),
		"exit 0 (clean)",
		"stderr tail",
		"stdout tail",
	} {
		if !strings.Contains(first.Error(), want) {
			t.Fatalf("exitDiagnosis lacks %q:\n%s", want, first)
		}
	}
}

// TestAwaitNodeServing proves the node-aware readiness wait: the healthy arm
// returns normally through the fatal wrapper, and an already-exited child is
// diagnosed on the first poll rather than waited on as if it were slow.
func TestAwaitNodeServing(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)

	healthy := startStandNode(t, p, standNodeAID)
	requireNodeServing(t, healthy, p.grpcA, servingDeadline)
	signalAndWaitExit(t, healthy)

	dead := launchNode(t, p.bin, &bytes.Buffer{}, "--node-id=0")
	waitNodeExited(t, dead, "the invalid-configuration child's exit")
	started := time.Now()
	err := awaitNodeServing(t, dead, p.grpcA, servingDeadline)
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("awaitNodeServing on an exited child = nil, want the exit diagnosis")
	}
	if elapsed > earlyExitReadinessBound {
		t.Fatalf("awaitNodeServing returned after %v on an exited child, want the diagnosis on the first poll (bound %v)", elapsed, earlyExitReadinessBound)
	}
	for _, want := range []string{
		fmt.Sprintf("process %d", dead.cmd.Process.Pid),
		"exit code 2",
		"stderr tail",
		"required unique 16-bit node ID",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("awaitNodeServing diagnosis lacks %q:\n%s", want, err)
		}
	}
}
