package repl_test

import (
	"fmt"
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

// TestDiscard_warnLimiterKeyedByNodeAndClass pins the shared WARN limiter's
// key as the PAIR (node_id, class) on event time (T-4-04, PROTO-07): a
// second class from a peer that already warned inside the interval still
// WARNs once (the class is part of the key), a repeat of an already-warned
// pair stays silent, a second peer's first occurrence of that same class
// gets its own WARN (the peer is part of the key), and a pair warns again
// exactly at the interval boundary. The discard counters stay per datagram —
// the limiter suppresses only the log line.
func TestDiscard_warnLimiterKeyedByNodeAndClass(t *testing.T) {
	t.Parallel()
	now := at(0)
	const warnInterval = 10 * time.Second // the pinned discardWarnInterval
	var logs []string
	core, _ := newTestCoreWithConfig(repl.Config{Log: func(msg string) { logs = append(logs, msg) }})
	var out port.Outbox

	feedBadProtoVersion := func(peer uint16, at time.Time) {
		t.Helper()
		core.HandleDatagram(at, port.InboundDatagram{
			Source:       handleFor(peer),
			Status:       port.ProtoVersionMismatch,
			ProtoVersion: repl.SupportedProtoVersion + 1,
			NodeID:       peer,
			RunID:        testRunID(peer),
		}, &out)
		out.Drain()
	}
	feedBadSyncHeader := func(peer uint16, at time.Time) {
		t.Helper()
		core.HandleDatagram(at, datagramFrom(peer, port.DataSync{SyncID: 1, ChunkCount: 0}), &out)
		out.Drain()
	}
	// count returns the WARN lines naming class from node — the exact pair
	// the limiter keys on.
	count := func(class string, node uint16) int {
		want := fmt.Sprintf("class %s from node %d", class, node)
		n := 0
		for _, line := range logs {
			if strings.Contains(line, want) {
				n++
			}
		}
		return n
	}

	// The same peer's two classes inside one interval: each fires once, so
	// the class is part of the key and one class never suppresses another.
	feedBadProtoVersion(peerID1, now)
	feedBadSyncHeader(peerID1, now)
	if got := count(repl.DiscardBadProtoVersion, peerID1); got != 1 {
		t.Fatalf("bad_proto_version WARNs for peer 1 = %d, want 1: %v", got, logs)
	}
	if got := count(repl.DiscardBadSyncHeader, peerID1); got != 1 {
		t.Fatalf("bad_sync_header WARNs for peer 1 = %d, want 1: %v", got, logs)
	}
	if got := len(logs); got != 2 {
		t.Fatalf("Config.Log fired %d time(s) for two classes of one peer, want 2: %v", got, logs)
	}

	// Inside the interval a warned pair stays silent, but a second peer's
	// first occurrence of the same class — and its own second class — each
	// get their own WARN: the peer is the key's other half.
	inside := now.Add(warnInterval - time.Nanosecond)
	feedBadProtoVersion(peerID1, inside)
	feedBadSyncHeader(peerID1, inside)
	if got := len(logs); got != 2 {
		t.Fatalf("Config.Log fired %d time(s) for repeats inside the interval, want still 2: %v", got, logs)
	}
	feedBadProtoVersion(peerID2, inside)
	feedBadSyncHeader(peerID2, inside)
	if got := count(repl.DiscardBadProtoVersion, peerID2); got != 1 {
		t.Fatalf("bad_proto_version WARNs for peer 2 inside peer 1's interval = %d, want 1 — one peer must never suppress another: %v", got, logs)
	}
	if got := count(repl.DiscardBadSyncHeader, peerID2); got != 1 {
		t.Fatalf("bad_sync_header WARNs for peer 2 inside peer 1's interval = %d, want 1: %v", got, logs)
	}
	if got := len(logs); got != 4 {
		t.Fatalf("Config.Log fired %d time(s) after two peers' two classes, want 4: %v", got, logs)
	}

	// At exactly the interval the pair fires again (`>=` is allowed).
	feedBadProtoVersion(peerID1, now.Add(warnInterval))
	if got := count(repl.DiscardBadProtoVersion, peerID1); got != 2 {
		t.Fatalf("bad_proto_version WARNs for peer 1 at the interval boundary = %d, want 2: %v", got, logs)
	}
	if got := len(logs); got != 5 {
		t.Fatalf("Config.Log fired %d time(s) at the boundary, want 5: %v", got, logs)
	}

	// The limiter never suppresses the counters: every datagram counted.
	if got := core.Discards().BadProtoVersion; got != 4 {
		t.Fatalf("BadProtoVersion = %d, want 4 — the limiter suppresses the WARN, never the counter", got)
	}
	if got := core.Discards().BadSyncHeader; got != 3 {
		t.Fatalf("BadSyncHeader = %d, want 3 — the limiter suppresses the WARN, never the counter", got)
	}

	// The clock_drift class rides the same limiter with its own slot: even
	// though this peer already warned for the discard classes in this
	// interval, one leading record still logs its fallback WARN, and a second
	// inside the interval stays silent while the counter moves per record.
	aheadMs := testNowMs + int64(testDriftWarn/time.Millisecond) + 10_000
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, remoteRecord(aheadMs, peerID1, "ahead-a")), &out)
	out.Drain()
	core.HandleDatagram(now.Add(time.Second), replFrom(peerID1, keyBeta, remoteRecord(aheadMs, peerID1, "ahead-b")), &out)
	out.Drain()
	driftWarns := 0
	for _, line := range logs {
		if strings.Contains(line, "clock drift") {
			driftWarns++
		}
	}
	if driftWarns != 1 {
		t.Fatalf("clock_drift WARNs = %d for two leading records in one interval, want 1 — the class has its own slot: %v", driftWarns, logs)
	}
	if got := core.MergeStats().DriftWarnings; got != 2 {
		t.Fatalf("DriftWarnings = %d after two leading records, want 2 — the limiter suppresses the WARN, never the counter", got)
	}
}

// TestApplyFailure_warnLimiterKeyedByNodeAndClass pins the apply-failure
// WARN's flood guard and key-free text (Phase-3 IN-01, T-07-17, T-07-18):
// repeated datagrams whose store apply fails produce exactly ONE captured
// WARN per (node_id, apply_failure) per limiter interval, the message names
// the quoted source handle and never the key, no Ack and no Superseded result
// is emitted, the pending entry survives untouched, and a second peer's
// failure gets its own WARN.
func TestApplyFailure_warnLimiterKeyedByNodeAndClass(t *testing.T) {
	t.Parallel()
	now := at(0)
	const warnInterval = 10 * time.Second // the pinned discardWarnInterval
	var logs []string
	core, st := newLimitedTestCore(8, &logs)
	var out port.Outbox

	// "local" fits the store's 8-byte value limit; the replicated record never
	// does, so every apply below fails with ErrValueTooLarge after validation.
	tracked := localWrite(t, core, st, now, keyAlpha, "local")
	oversize := remoteRecord(testNowMs+5000, peerID1, "over-the-limit")

	fail := func(peer uint16, at time.Time) {
		t.Helper()
		core.HandleDatagram(at, replFrom(peer, keyAlpha, oversize), &out)
		if got := len(out.Drain()); got != 0 {
			t.Fatalf("the refused apply from peer %d emitted %d message(s), want none — a refused record is never Acked", peer, got)
		}
		if got := len(out.DrainResults()); got != 0 {
			t.Fatalf("the refused apply from peer %d emitted %d result(s), want none — no supersession", peer, got)
		}
	}

	fail(peerID1, now)
	if got := len(logs); got != 1 {
		t.Fatalf("Config.Log fired %d time(s) for the first failed apply, want 1: %v", got, logs)
	}
	if strings.Contains(logs[0], keyAlpha) {
		t.Fatalf("the apply-failure WARN names the key %q, want the key absent: %q", keyAlpha, logs[0])
	}
	if !strings.Contains(logs[0], handleFor(peerID1)) {
		t.Fatalf("the apply-failure WARN does not name the quoted source handle %q: %q", handleFor(peerID1), logs[0])
	}

	// Inside the interval the same peer's pair stays silent: the limiter, not
	// the datagram count, bounds the log.
	fail(peerID1, now.Add(warnInterval-time.Nanosecond))
	if got := len(logs); got != 1 {
		t.Fatalf("Config.Log fired %d time(s) for a repeat inside the interval, want still 1: %v", got, logs)
	}

	// A second peer's first failure has its own slot: the node id is the
	// key's other half.
	fail(peerID2, now.Add(warnInterval-time.Nanosecond))
	if got := len(logs); got != 2 {
		t.Fatalf("Config.Log fired %d time(s) after a second peer's failure, want 2: %v", got, logs)
	}

	// At exactly the interval the first peer's pair fires again.
	fail(peerID1, now.Add(warnInterval))
	if got := len(logs); got != 3 {
		t.Fatalf("Config.Log fired %d time(s) at the interval boundary, want 3: %v", got, logs)
	}

	// The failures changed no tracking state and no stored record.
	assertPending(t, core.PendingSnapshot(), keyAlpha, tracked, nil)
	served, err := st.store.Get(keyAlpha)
	if err != nil || served.Version != tracked {
		t.Fatalf("store serves (%x, %v) after the refused applies, want the untouched pending version %x", served.Version[:], err, tracked[:])
	}
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
