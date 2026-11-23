package main

// Internal tests for the run orchestration seam (D-06, OBSV-04). The fakes
// record every lifecycle call in order and stage each outcome, so the exit
// table, the signal-before-wire ordering and the Serve-error-still-dumps rule
// are proven without a socket; main_test.go pins the same codes at the binary
// boundary.

import (
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"testing"

	"github.com/Djarvur/sup-kv/internal/config"
)

// The staged outcomes the fakes return. Static package-level sentinels because
// the strict lint rejects errors.New inside a function body.
var (
	errTestBindRefused = errors.New("address already in use")
	errTestWire        = &startupError{
		stage: stageReplicationBind,
		err:   fmt.Errorf("bind replication socket: %w", errTestBindRefused),
	}
)

// runFake is the injectable runtime over an ordered call log: every error
// field stages one outcome, and the wire fake refuses to run before the
// signals are armed.
type runFake struct {
	calls         []string
	signalsArmed  bool
	stopCalls     int
	wireErr       error
	serveErr      error
	shutdownErr   error
	shutdownCalls int
}

// runtime renders the fake as run's seam. t receives the ordering violation
// the wire fake detects.
func (f *runFake) runtime(t *testing.T) runtime {
	t.Helper()
	return runtime{
		installSignals: func() chan os.Signal {
			f.calls = append(f.calls, "installSignals")
			f.signalsArmed = true
			return make(chan os.Signal, signalBuffer)
		},
		stopSignals: func(chan<- os.Signal) {
			f.calls = append(f.calls, "stopSignals")
			f.stopCalls++
		},
		wire: func(config.Config) (*wired, net.Listener, error) {
			f.calls = append(f.calls, "wire")
			if !f.signalsArmed {
				t.Error("wire ran before installSignals: dump load and bind are not signal-covered")
			}
			if f.wireErr != nil {
				return nil, nil, f.wireErr
			}
			return &wired{}, nil, nil
		},
		serve: func(*wired, net.Listener, <-chan os.Signal) error {
			f.calls = append(f.calls, "serve")
			return f.serveErr
		},
		shutdown: func(*wired, <-chan os.Signal) error {
			f.calls = append(f.calls, "shutdown")
			f.shutdownCalls++
			return f.shutdownErr
		},
	}
}

// assertCalls pins the recorded orchestration order.
func (f *runFake) assertCalls(t *testing.T, want ...string) {
	t.Helper()
	if !slices.Equal(f.calls, want) {
		t.Fatalf("lifecycle order = %v, want %v", f.calls, want)
	}
}

// TestRun_signalInstalledBeforeWire pins D-06's ordering: the signal channel
// is armed first, so a SIGTERM during dump load or UDP bind takes the graceful
// path instead of the default disposition.
func TestRun_signalInstalledBeforeWire(t *testing.T) {
	t.Parallel()
	fake := &runFake{}
	code := run(config.Config{}, fake.runtime(t))
	if code != 0 {
		t.Fatalf("run = %d, want 0 for a clean path", code)
	}
	fake.assertCalls(t, "installSignals", "wire", "serve", "shutdown", "stopSignals")
}

// TestRun_cleanExitZero pins the clean row of the exit table: a signal-driven
// stop whose final dump succeeded exits 0.
func TestRun_cleanExitZero(t *testing.T) {
	t.Parallel()
	fake := &runFake{}
	code := run(config.Config{}, fake.runtime(t))
	if code != 0 {
		t.Fatalf("run = %d, want 0", code)
	}
	if fake.shutdownCalls != 1 {
		t.Fatalf("shutdown ran %d times on the clean path, want 1", fake.shutdownCalls)
	}
	if fake.stopCalls != 1 {
		t.Fatalf("stopSignals ran %d times on the clean path, want 1", fake.stopCalls)
	}
}

// TestRun_startupFailureExitsOneNoDump pins the startup row: a staged wire
// failure exits 1, never calls shutdown (no client-confirmed write exists
// yet) and still releases the signal channel.
func TestRun_startupFailureExitsOneNoDump(t *testing.T) {
	t.Parallel()
	fake := &runFake{wireErr: errTestWire}
	code := run(config.Config{}, fake.runtime(t))
	if code != exitStartup {
		t.Fatalf("run = %d, want %d for a startup failure", code, exitStartup)
	}
	if fake.shutdownCalls != 0 {
		t.Fatalf("shutdown ran %d times after a startup failure, want 0", fake.shutdownCalls)
	}
	if fake.stopCalls != 1 {
		t.Fatalf("stopSignals ran %d times after a startup failure, want 1", fake.stopCalls)
	}
	fake.assertCalls(t, "installSignals", "wire", "stopSignals")
}
