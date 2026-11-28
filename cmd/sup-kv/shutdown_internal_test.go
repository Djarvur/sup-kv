package main

// Internal tests for the bounded shutdown wait (PERSIST-08, T-08.4). The
// process-level shutdown is smoke-tested end to end; what cannot be wedged
// from outside the process is a dumper stuck in a kernel fsync, so the expiry
// branch is exercised here through the same helper main() calls.

import (
	"bytes"
	"context"
	"errors"
	"log"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// errTestDumperDied is the abnormal engine-Run outcome the reported-branch
// subtest feeds the wait (a static sentinel: the strict lint rejects dynamic
// errors in tests).
var errTestDumperDied = errors.New("dumper died")

// TestAwaitEngineStop covers every branch of the bounded engine wait: a
// prompt clean stop, a prompt abnormal stop (reported), and a wedged dumper
// abandoned at the grace with the operator WARN.
//
//nolint:paralleltest // the subtests swap the process-global slog/log writers, so they must run sequentially
func TestAwaitEngineStop(t *testing.T) {
	var slogBuf bytes.Buffer
	oldSlog := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&slogBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(oldSlog) })

	var logBuf bytes.Buffer
	oldLog := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(oldLog) })

	t.Run("a clean cancellation returns without waiting", func(t *testing.T) {
		done := make(chan error, 1)
		done <- context.Canceled
		start := time.Now()
		awaitEngineStop(done, time.Second)
		if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
			t.Fatalf("awaitEngineStop waited %v for an already-canceled run, want a prompt return", elapsed)
		}
		if strings.Contains(logBuf.String(), "engine Run returned") {
			t.Fatalf("a clean cancellation was reported as an engine error: %q", logBuf.String())
		}
	})

	t.Run("an abnormal stop is reported", func(t *testing.T) {
		done := make(chan error, 1)
		done <- errTestDumperDied
		start := time.Now()
		awaitEngineStop(done, time.Second)
		if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
			t.Fatalf("awaitEngineStop waited %v for a returned run error, want a prompt return", elapsed)
		}
		if !strings.Contains(logBuf.String(), "engine Run returned dumper died") {
			t.Fatalf("the abnormal run error was not logged: %q", logBuf.String())
		}
	})

	t.Run("a wedged engine is abandoned at the grace with a WARN", func(t *testing.T) {
		done := make(chan error) // never written: the dumper is stuck
		const grace = 25 * time.Millisecond
		start := time.Now()
		awaitEngineStop(done, grace)
		elapsed := time.Since(start)
		if elapsed < grace {
			t.Fatalf("awaitEngineStop returned after %v, before the %v grace", elapsed, grace)
		}
		if elapsed > 2*time.Second {
			t.Fatalf("awaitEngineStop blocked %v on a wedged engine, want the %v grace bound", elapsed, grace)
		}
		if !strings.Contains(slogBuf.String(), "Engine did not stop within the shutdown grace") {
			t.Fatalf("the abandoned-engine WARN was not emitted: %q", slogBuf.String())
		}
	})
}
