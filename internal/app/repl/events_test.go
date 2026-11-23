package repl_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// farAheadLead is the received-version lead every drift row stamps: one
// drift-warn window (60 s) plus five seconds of slack, comfortably inside the
// two-minute max_clock_lead the rows configure.
const farAheadLead = testDriftWarn + 5*time.Second

// collectPeerEvents returns a core whose OnPeerEvent appends every event to
// the returned slice. HandleDatagram and Advance drive the callback
// synchronously on the caller's goroutine, so the test reads the slice after
// the call without a lock.
func collectPeerEvents() (*repl.Core, *[]repl.PeerEvent) {
	var events []repl.PeerEvent
	core, _ := newTestCoreWithConfig(repl.Config{
		OnPeerEvent: func(ev repl.PeerEvent) { events = append(events, ev) },
	})
	return core, &events
}

// collectDriftEvents returns a core over the driftLead bound whose OnDrift
// appends every admitted event to the returned slice.
func collectDriftEvents() (*repl.Core, *[]repl.DriftEvent) {
	var events []repl.DriftEvent
	core, _ := newTestCoreWithConfig(repl.Config{
		MaxClockLead: driftLead,
		OnDrift:      func(ev repl.DriftEvent) { events = append(events, ev) },
	})
	return core, &events
}

// TestPeerEvents pins the D-01 peer-lifecycle event matrix: exactly one event
// per real transition — first-seen appear, unseen-run restart, silent-peer
// eviction — and none for the shapes that are not transitions: a repeat
// Hello, an alternation back to a remembered run, and the node's own echo.
func TestPeerEvents(t *testing.T) {
	t.Parallel()

	t.Run("a first Hello emits one PeerAppeared", func(t *testing.T) {
		t.Parallel()
		core, events := collectPeerEvents()
		var out port.Outbox

		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()

		if len(*events) != 1 {
			t.Fatalf("a first Hello emitted %d peer event(s), want exactly 1: %+v", len(*events), *events)
		}
		ev := (*events)[0]
		if ev.Kind != repl.PeerAppeared || ev.NodeID != peerID1 || ev.Handle != handleFor(peerID1) ||
			ev.RunID != testRunID(peerID1) || ev.PreviousRunID != ([16]byte{}) || ev.LastSeenAge != 0 {
			t.Fatalf("first-Hello event = %+v, want PeerAppeared(node %d, handle %q, run %x)",
				ev, peerID1, handleFor(peerID1), testRunID(peerID1))
		}
	})

	t.Run("a repeat Hello emits nothing", func(t *testing.T) {
		t.Parallel()
		core, events := collectPeerEvents()
		var out port.Outbox

		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		core.HandleDatagram(at(time.Second), helloFrom(peerID1), &out)
		out.Drain()

		if len(*events) != 1 {
			t.Fatalf("a repeat Hello brought the event count to %d, want still 1 — a repeat refreshes liveness only: %+v",
				len(*events), *events)
		}
	})

	t.Run("a new run under a remembered id emits one PeerRestarted", func(t *testing.T) {
		t.Parallel()
		core, events := collectPeerEvents()
		var out port.Outbox

		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		newRun := testRunID(peerID1)
		newRun[1] = 0x77
		core.HandleDatagram(at(time.Second), datagramFromRun(peerID1, newRun, port.Hello{}), &out)
		out.Drain()

		if len(*events) != 2 {
			t.Fatalf("appear plus restart emitted %d peer event(s), want 2: %+v", len(*events), *events)
		}
		ev := (*events)[1]
		if ev.Kind != repl.PeerRestarted || ev.NodeID != peerID1 || ev.Handle != handleFor(peerID1) ||
			ev.RunID != newRun || ev.PreviousRunID != testRunID(peerID1) {
			t.Fatalf("restart event = %+v, want PeerRestarted(node %d, previous %x, new %x)",
				ev, peerID1, testRunID(peerID1), newRun)
		}
	})

	t.Run("an alternating run_id emits nothing", func(t *testing.T) {
		t.Parallel()
		core, events := collectPeerEvents()
		var out port.Outbox

		oldRun := testRunID(peerID2)
		newRun := oldRun
		newRun[1] = 0x77
		core.HandleDatagram(at(0), helloFrom(peerID2), &out)
		out.Drain()
		core.HandleDatagram(at(time.Second), datagramFromRun(peerID2, newRun, port.Hello{}), &out)
		out.Drain()
		core.HandleDatagram(at(2*time.Second), datagramFromRun(peerID2, oldRun, port.Hello{}), &out)
		out.Drain()

		if len(*events) != 2 {
			t.Fatalf("the alternation brought the event count to %d, want still 2 — a return to a seen run is a configuration error, never a transition: %+v",
				len(*events), *events)
		}
	})

	t.Run("the node's own echo emits nothing", func(t *testing.T) {
		t.Parallel()
		core, events := collectPeerEvents()
		var out port.Outbox

		core.HandleDatagram(at(0), datagramFrom(localID, port.Hello{}), &out)
		out.Drain()

		if len(*events) != 0 {
			t.Fatalf("the node's own echo emitted %d peer event(s), want none: %+v", len(*events), *events)
		}
	})

	t.Run("a silent peer's eviction emits one PeerEvicted with its age", func(t *testing.T) {
		t.Parallel()
		core, events := collectPeerEvents()
		var out port.Outbox

		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		core.Advance(at(peerTimeout), &out)
		out.Drain()

		if len(*events) != 2 {
			t.Fatalf("the eviction brought the event count to %d, want 2 (appear, evicted): %+v", len(*events), *events)
		}
		ev := (*events)[1]
		if ev.Kind != repl.PeerEvicted || ev.NodeID != peerID1 || ev.Handle != handleFor(peerID1) ||
			ev.RunID != testRunID(peerID1) || ev.LastSeenAge != peerTimeout {
			t.Fatalf("eviction event = %+v, want PeerEvicted(node %d, handle %q, last_seen_age %v)",
				ev, peerID1, handleFor(peerID1), peerTimeout)
		}
	})
}

// TestDriftEvent pins the D-02 drift seam: the counter is per offending
// record while the event is admitted once per (peer, limiter interval); a
// listing part's leading maximum counts and reports one event; a record
// behind the wall reports nothing; and a nil OnDrift falls back to the
// prebuilt Log string.
func TestDriftEvent(t *testing.T) {
	t.Parallel()

	t.Run("the counter is per record, the event per peer per interval", func(t *testing.T) {
		t.Parallel()
		core, events := collectDriftEvents()
		now := at(0)
		aheadMs := testNowMs + int64(farAheadLead/time.Millisecond)
		var out port.Outbox

		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, remoteRecord(aheadMs, peerID1, "ahead-a")), &out)
		out.Drain()
		core.HandleDatagram(now.Add(time.Second), replFrom(peerID1, keyBeta, remoteRecord(aheadMs, peerID1, "ahead-b")), &out)
		out.Drain()

		if got := core.MergeStats().DriftWarnings; got != 2 {
			t.Fatalf("DriftWarnings = %d after two leading records, want 2 — the counter is per record", got)
		}
		if len(*events) != 1 {
			t.Fatalf("two leading records inside one interval emitted %d DriftEvent(s), want 1 — the limiter suppresses the event, never the counter: %+v",
				len(*events), *events)
		}
		ev := (*events)[0]
		if ev.NodeID != peerID1 || ev.Source != handleFor(peerID1) || ev.Lead != farAheadLead {
			t.Fatalf("DriftEvent = %+v, want node %d, source %q, lead %v",
				ev, peerID1, handleFor(peerID1), farAheadLead)
		}
	})

	t.Run("a leading listing-part maximum emits one DriftEvent", func(t *testing.T) {
		t.Parallel()
		core, events := collectDriftEvents()
		now := at(0)
		var out port.Outbox

		core.HandleDatagram(now, helloFrom(peerID1), &out)
		out.Drain()
		listed := version.PackVersion(testNowMs+int64(farAheadLead/time.Millisecond), 0, peerID1)
		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{
			SyncID:     1,
			ChunkIndex: 1,
			ChunkCount: 1,
			Entries:    []port.SyncEntry{{Key: keyAlpha, Version: listed}},
		}), &out)
		out.Drain()

		if got := core.MergeStats().DriftWarnings; got != 1 {
			t.Fatalf("DriftWarnings = %d after a leading listing part, want 1", got)
		}
		if len(*events) != 1 {
			t.Fatalf("the leading listing part emitted %d DriftEvent(s), want 1: %+v", len(*events), *events)
		}
		ev := (*events)[0]
		if ev.NodeID != peerID1 || ev.Source != handleFor(peerID1) || ev.Lead != farAheadLead {
			t.Fatalf("listing DriftEvent = %+v, want node %d, source %q, lead %v",
				ev, peerID1, handleFor(peerID1), farAheadLead)
		}
	})

	t.Run("a record behind the wall emits nothing", func(t *testing.T) {
		t.Parallel()
		core, events := collectDriftEvents()
		now := at(0)
		behindMs := testNowMs - int64(farAheadLead/time.Millisecond)
		var out port.Outbox

		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, remoteRecord(behindMs, peerID1, "behind")), &out)
		out.Drain()

		if got := core.MergeStats().DriftWarnings; got != 0 {
			t.Fatalf("DriftWarnings = %d after a behind-the-wall record, want 0 — only a positive lead warns", got)
		}
		if len(*events) != 0 {
			t.Fatalf("a behind-the-wall record emitted %d DriftEvent(s), want none: %+v", len(*events), *events)
		}
	})

	t.Run("a nil OnDrift falls back to the Log seam", func(t *testing.T) {
		t.Parallel()
		var logs []string
		core, _ := newTestCoreWithConfig(repl.Config{
			MaxClockLead: driftLead,
			Log:          func(msg string) { logs = append(logs, msg) },
		})
		now := at(0)
		aheadMs := testNowMs + int64(farAheadLead/time.Millisecond)
		var out port.Outbox

		core.HandleDatagram(now, replFrom(peerID1, keyAlpha, remoteRecord(aheadMs, peerID1, "ahead-a")), &out)
		out.Drain()
		core.HandleDatagram(now.Add(time.Second), replFrom(peerID1, keyBeta, remoteRecord(aheadMs, peerID1, "ahead-b")), &out)
		out.Drain()

		if len(logs) != 1 {
			t.Fatalf("the nil-OnDrift fallback logged %d line(s) for two leading records in one interval, want 1: %v", len(logs), logs)
		}
		if !strings.Contains(logs[0], "clock drift") {
			t.Fatalf("the fallback drift line = %q, want it to name clock drift", logs[0])
		}
	})
}
