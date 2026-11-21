package repl_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/port"
)

// noPeerWarnLines filters the captured Config.Log lines down to the no-Hello
// WARNs — the lines naming peer_timeout, the XPORT-09 observable. Other Log
// traffic (apply failures, discard WARNs) never carries that token, so a
// count is the warning's exact frequency.
func noPeerWarnLines(logs []string) []string {
	var warns []string
	for _, line := range logs {
		if strings.Contains(line, "peer_timeout") {
			warns = append(warns, line)
		}
	}
	return warns
}

// TestCore_noPeerWarn pins XPORT-09's core half of D-07: a node that receives
// no Hello within peer_timeout after its first Advance WARNs exactly once —
// never per interval — naming peer_timeout, and a Hello arriving before the
// deadline suppresses the WARN entirely. The warning is diagnostic only: the
// core keeps serving and every Advance still returns a future deadline.
func TestCore_noPeerWarn(t *testing.T) {
	t.Parallel()

	t.Run("exactly one WARN at peer_timeout when no Hello ever arrives", func(t *testing.T) {
		t.Parallel()
		var logs []string
		core, _ := newTestCoreWithConfig(repl.Config{Log: func(msg string) { logs = append(logs, msg) }})
		var out port.Outbox

		// Before the deadline there is no warning: the node has not yet had
		// a full peer_timeout window to hear anyone.
		core.Advance(at(0), &out)
		out.Drain()
		core.Advance(at(peerTimeout-time.Nanosecond), &out)
		out.Drain()
		if got := len(noPeerWarnLines(logs)); got != 0 {
			t.Fatalf("Config.Log carried %d peer_timeout WARN(s) before the deadline, want none: %v", got, logs)
		}

		// The first Advance at or after firstAdvance+peer_timeout warns once.
		next := core.Advance(at(peerTimeout), &out)
		out.Drain()
		warns := noPeerWarnLines(logs)
		if len(warns) != 1 {
			t.Fatalf("Config.Log carried %d peer_timeout WARN(s) at the deadline, want exactly 1: %v", len(warns), logs)
		}
		if !strings.Contains(warns[0], peerTimeout.String()) {
			t.Fatalf("the no-Hello WARN %q does not name the configured %s bound", warns[0], peerTimeout)
		}
		if !next.After(at(peerTimeout)) {
			t.Fatalf("Advance returned %v, want a deadline strictly after the event time — the WARN must not change scheduling", next)
		}

		// Never per interval: many further advances add no second line.
		for off := peerTimeout + time.Second; off <= peerTimeout+time.Minute; off += time.Second {
			core.Advance(at(off), &out)
			out.Drain()
		}
		if got := len(noPeerWarnLines(logs)); got != 1 {
			t.Fatalf("Config.Log carried %d peer_timeout WARN(s) across a minute of intervals, want the one shot: %v", got, logs)
		}
	})

	t.Run("a Hello before the deadline suppresses the WARN", func(t *testing.T) {
		t.Parallel()
		var logs []string
		core, _ := newTestCoreWithConfig(repl.Config{Log: func(msg string) { logs = append(logs, msg) }})
		var out port.Outbox
		core.Advance(at(0), &out)
		out.Drain()
		core.HandleDatagram(at(time.Second), helloFrom(peerID1), &out)
		out.Drain()

		for off := time.Second; off <= 6*peerTimeout; off += time.Second {
			core.Advance(at(off), &out)
			out.Drain()
		}
		if got := len(noPeerWarnLines(logs)); got != 0 {
			t.Fatalf("Config.Log carried %d peer_timeout WARN(s) although a Hello arrived before the deadline, want none: %v", got, logs)
		}
	})
}
