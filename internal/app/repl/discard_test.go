package repl_test

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Discard-classification test constants: the adversarial round count of the
// totality proof (D-02) and the second handle a claiming node announces from
// (REPL-08's two-sources-one-identity signature).
const (
	discardRounds = 500
	cloneHandle   = "n2-clone"
)

// newDiscardCore returns a fresh local-node core whose Config.Log appends to
// logs — the classification tests' REPL-08 observation seam. Same shape as
// newTestCore (fixed clock millisecond, default Config) except the log hook.
func newDiscardCore(logs *[]string) *repl.Core {
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})}
	cfg := repl.Config{Log: func(msg string) { *logs = append(*logs, msg) }}
	return repl.New(localID, testRunID(localID), handleFor(localID), st, cfg)
}

// TestDiscard_classes proves the five discard classes distinguishable and
// isolated (REPL-09, D-02): every row feeds one datagram and asserts the FULL
// counter set — the named class increments and nothing else — plus that a
// discarded datagram never reaches protocol processing (empty outbox, peer
// table unchanged) while both duplicate-node_id variants additionally log a
// configuration error naming the node_id and the conflicting handles through
// the Config.Log hook (REPL-08). The restart row pins the discrimination: a
// known node_id whose announced run_id differs from the remembered one is a
// restart (REPL-07, plan 04-04), never a duplicate — the classifier never
// pre-empts the restart branch.
func TestDiscard_classes(t *testing.T) {
	t.Parallel()
	now := at(0)
	rows := []struct {
		name        string
		seedPeer1   bool
		in          port.InboundDatagram
		want        repl.DiscardStats
		wantLogged  bool
		logMustName []string
		wantPeerID  uint16
		wantPeer    *repl.PeerInfo
		wantEmits   int
		wantQueued  int
	}{
		{
			// What udprepl.decode hands over for an empty read: the bare
			// Undecodable verdict, no field parsed (REPL-09's empty edge).
			name: "zero-length datagram classifies Undecodable",
			in:   port.InboundDatagram{Status: port.Undecodable},
			want: repl.DiscardStats{Undecodable: 1},
		},
		{
			// Garbage bytes meet the same verdict before any envelope field
			// exists — separate from BadCRC, BadProtoVersion and the
			// node_id classes.
			name: "garbage bytes classify Undecodable",
			in:   port.InboundDatagram{Status: port.Undecodable},
			want: repl.DiscardStats{Undecodable: 1},
		},
		{
			// Adapter contract (D-02): the framing verdict carries the
			// envelope fields it did parse and never decodes the body.
			name: "CRC failure with Msg nil classifies BadCRC",
			in: port.InboundDatagram{
				Status:       port.BadCRC,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
			},
			want: repl.DiscardStats{BadCRC: 1},
		},
		{
			name: "proto version supported+1 classifies BadProtoVersion",
			in: port.InboundDatagram{
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion + 1,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
				Msg:          port.Hello{},
			},
			want: repl.DiscardStats{BadProtoVersion: 1},
		},
		{
			name: "own node_id with own run_id classifies OwnEcho",
			in: port.InboundDatagram{
				Source:       handleFor(localID),
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       localID,
				RunID:        testRunID(localID),
				Msg:          port.Hello{},
			},
			want: repl.DiscardStats{OwnEcho: 1},
		},
		{
			// The tracer skeleton's variant: a foreign run claiming OUR
			// node_id — an impostor cannot guess the 16-byte run_id, so
			// run_id inequality is the duplicate test.
			name: "own node_id with a foreign run_id classifies DuplicateNodeID and logs a configuration error",
			in: port.InboundDatagram{
				Source:       handleFor(peerID1),
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       localID,
				RunID:        testRunID(peerID1),
				Msg:          port.Hello{},
			},
			want:        repl.DiscardStats{DuplicateNodeID: 1},
			wantLogged:  true,
			logMustName: []string{fmt.Sprintf("node_id %d", localID), handleFor(peerID1)},
		},
		{
			// The second variant: two sources claiming one identity — the
			// known peer's exact run_id announced from a second handle.
			name:      "known node_id repeating its run_id from a second handle classifies DuplicateNodeID and names both handles",
			seedPeer1: true,
			in: port.InboundDatagram{
				Source:       cloneHandle,
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
				Msg:          port.Hello{},
			},
			want:        repl.DiscardStats{DuplicateNodeID: 1},
			wantLogged:  true,
			logMustName: []string{fmt.Sprintf("node_id %d", peerID1), handleFor(peerID1), cloneHandle},
			wantPeerID:  peerID1,
			wantPeer:    &repl.PeerInfo{Handle: handleFor(peerID1), RunID: testRunID(peerID1), LastHello: now},
		},
		{
			name:       "a valid Hello from a stranger moves no counter and learns the peer",
			in:         helloFrom(strangerID),
			want:       repl.DiscardStats{},
			wantPeerID: strangerID,
			wantPeer:   &repl.PeerInfo{Handle: handleFor(strangerID), RunID: testRunID(strangerID), LastHello: now},
		},
		{
			// The discrimination pin: run_id EQUALITY with the remembered
			// row is what makes a duplicate; a differing run_id reaches the
			// Hello handler and fires the plan-04-04 restart reaction.
			name:      "a known node_id with a changed run_id is a restart, never a duplicate",
			seedPeer1: true,
			in: port.InboundDatagram{
				Source:       cloneHandle,
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       peerID1,
				RunID:        testRunID(strangerID),
				Msg:          port.Hello{},
			},
			want:       repl.DiscardStats{},
			wantPeerID: peerID1,
			wantPeer:   &repl.PeerInfo{Handle: cloneHandle, RunID: testRunID(strangerID), LastHello: now},
			// The targeted listing is enqueued on the listing pacer, not
			// emitted by the handler: one queued chunk proves the restart
			// reaction ran while the empty handler outbox proves the
			// duplicate classifier did not discard it.
			wantEmits:  0,
			wantQueued: 1,
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			var logs []string
			core := newDiscardCore(&logs)
			if row.seedPeer1 {
				var seed port.Outbox
				core.HandleDatagram(now, helloFrom(peerID1), &seed)
				seed.Drain()
				logs = nil
			}
			var out port.Outbox
			core.HandleDatagram(now, row.in, &out)
			if got := core.Discards(); got != row.want {
				t.Errorf("Discards() = %+v, want %+v — each discarded datagram must increment exactly one class",
					got, row.want)
			}
			if got := len(out.Drain()); got != row.wantEmits {
				t.Errorf("handler emitted %d messages, want %d — a discarded datagram is never also processed",
					got, row.wantEmits)
			}
			if got := core.QueuedSends(); got != row.wantQueued {
				t.Errorf("QueuedSends() = %d, want %d — the restart reaction enqueues its listing on the pacer",
					got, row.wantQueued)
			}
			snap := core.PeerSnapshot()
			if row.wantPeerID != 0 || row.wantPeer != nil {
				got, ok := snap[row.wantPeerID]
				if !ok || row.wantPeer == nil || got != *row.wantPeer {
					t.Errorf("PeerSnapshot[%d] = (%v, %t), want %v", row.wantPeerID, got, ok, row.wantPeer)
				}
			} else if len(snap) != 0 {
				t.Errorf("discarded datagram reached protocol processing: PeerSnapshot holds %d peers", len(snap))
			}
			if row.wantLogged {
				assertConfigErrorLog(t, logs, row.logMustName)
			} else if len(logs) != 0 {
				t.Errorf("healthy datagram logged %d line(s), want none (no spurious configuration errors): %v",
					len(logs), logs)
			}
		})
	}
}

// assertConfigErrorLog pins the REPL-08 surfacing half for one duplicate
// row: the latest Config.Log line carries the configuration-error marker and
// names every given identity (the node_id's decimal spelling and each
// conflicting handle).
func assertConfigErrorLog(t *testing.T, logs, mustName []string) {
	t.Helper()
	if len(logs) == 0 {
		t.Fatal("no line logged through Config.Log for a duplicate node_id (REPL-08 requires surfacing)")
	}
	line := logs[len(logs)-1]
	if !strings.Contains(line, "configuration error") {
		t.Errorf("log line %q lacks the configuration-error marker", line)
	}
	for _, name := range mustName {
		if !strings.Contains(line, name) {
			t.Errorf("log line %q does not name %q", line, name)
		}
	}
}

// modelPeer mirrors the classification-relevant peer-table row: the handle
// and run_id the second duplicate-node_id variant compares against. Only a
// processed Hello ever writes it — exactly the core's own mutation rule.
type modelPeer struct {
	handle string
	runID  [16]byte
}

// classify mirrors the classifier's pinned decision for one datagram against
// the model's current peer table, returning the discard class or "" when the
// datagram must reach protocol processing — the expectedDrift model-function
// precedent, applied to the classification order (D-02).
func classify(in port.InboundDatagram, peers map[uint16]modelPeer) string {
	switch in.Status {
	case port.BadCRC:
		return "BadCRC"
	case port.Undecodable:
		return "Undecodable"
	}
	if in.ProtoVersion != repl.SupportedProtoVersion {
		return "BadProtoVersion"
	}
	if in.NodeID == localID {
		if in.RunID == testRunID(localID) {
			return "OwnEcho"
		}
		return "DuplicateNodeID"
	}
	if known, ok := peers[in.NodeID]; ok && in.RunID == known.runID && in.Source != known.handle {
		return "DuplicateNodeID"
	}
	return ""
}

// bump adds one discard to stats' named class.
func bump(stats *repl.DiscardStats, class string) {
	switch class {
	case "BadCRC":
		stats.BadCRC++
	case "Undecodable":
		stats.Undecodable++
	case "BadProtoVersion":
		stats.BadProtoVersion++
	case "OwnEcho":
		stats.OwnEcho++
	case "DuplicateNodeID":
		stats.DuplicateNodeID++
	}
}

// Random-datagram draw pools: proto versions dominated by the supported one,
// node ids including the local and the two seeded peers plus adversarial
// edges (0, 0xFFFF, an unseeded stranger), and four source-handle namespaces.
var (
	protoVersions = []uint32{
		repl.SupportedProtoVersion, repl.SupportedProtoVersion, repl.SupportedProtoVersion,
		repl.SupportedProtoVersion + 1, 0, 7,
	}
	drawNodeIDs = []uint16{localID, peerID1, peerID2, strangerID, 0, 0xFFFF, 4242}
)

// randDatagram builds one adversarial InboundDatagram: a random framing
// verdict, envelope fields over any bit pattern (the testutil.RandRawVersion
// precedent — run_id bytes are arbitrary) and a random decoded body. Nothing
// here can make the classifier panic; everything can make it count
// (untrusted-input boundary, ASVS V5).
func randDatagram(rng *rand.Rand) port.InboundDatagram {
	in := port.InboundDatagram{Status: port.Decoded}
	switch rng.IntN(8) {
	case 0:
		in.Status = port.Undecodable
	case 1:
		in.Status = port.BadCRC
	}
	if in.Status == port.Undecodable {
		return in // the adapter hands over the bare verdict, nothing parsed.
	}
	// BadCRC still carries the envelope fields it did parse (D-02).
	in.ProtoVersion = protoVersions[rng.IntN(len(protoVersions))]
	in.NodeID = drawNodeIDs[rng.IntN(len(drawNodeIDs))]
	switch rng.IntN(4) {
	case 0, 1:
		in.RunID = testRunID(in.NodeID) // the node's canonical run id.
	case 2:
		in.RunID = [16]byte(testutil.RandRawVersion(rng)) // any bit pattern.
	default:
		in.RunID = testRunID(in.NodeID + 7) // some other node's canonical run id.
	}
	switch rng.IntN(4) {
	case 0, 1:
		in.Source = handleFor(in.NodeID)
	case 2:
		in.Source = cloneHandle
	default:
		in.Source = fmt.Sprintf("src-%d", rng.IntN(4))
	}
	in.Msg = randBody(rng)
	return in
}

// randBody draws one decoded message body from all five protocol messages
// plus the two adversarial shapes: no body at all on a Decoded verdict
// (violates the adapter contract; the dispatcher's no-op) and a standalone
// SyncEntry that never travels alone.
func randBody(rng *rand.Rand) port.Message {
	switch rng.IntN(7) {
	case 0:
		return nil
	case 1:
		return port.Hello{}
	case 2:
		return port.DataReplication{Key: testutil.RandKey(rng), Record: testutil.RandRecord(rng)}
	case 3:
		return port.Ack{Key: testutil.RandKey(rng), Version: testutil.RandRawVersion(rng)}
	case 4:
		return port.DataRequest{Key: testutil.RandKey(rng)}
	case 5:
		entries := make([]port.SyncEntry, 1+rng.IntN(4))
		for i := range entries {
			entries[i] = port.SyncEntry{Key: testutil.RandKey(rng), Version: testutil.RandVersion(rng)}
		}
		return port.DataSync{Entries: entries}
	default:
		return port.SyncEntry{Key: testutil.RandKey(rng), Version: testutil.RandVersion(rng)}
	}
}

// TestDiscard_totality proves the classifier total over adversarial input
// (D-02, REPL-09): 500 seeded random datagrams — random framing verdicts,
// any run_id bit pattern, every message shape — never panic, and counter
// conservation holds exactly: each datagram is discarded into exactly one
// class or processed, so the counters equal the model's per-class discard
// counts, their sum equals the discarded count, and the core's peer table
// equals the model's (a misclassification shows up as both a counter and a
// table divergence).
func TestDiscard_totality(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	now := at(0)
	var logs []string
	core := newDiscardCore(&logs)
	peers := map[uint16]modelPeer{
		peerID1: {handle: handleFor(peerID1), runID: testRunID(peerID1)},
		peerID2: {handle: handleFor(peerID2), runID: testRunID(peerID2)},
	}
	var out port.Outbox
	for _, id := range []uint16{peerID1, peerID2} {
		core.HandleDatagram(now, helloFrom(id), &out)
		out.Drain()
	}
	logs = nil

	var want repl.DiscardStats
	var discarded uint64
	for range discardRounds {
		in := randDatagram(rng)
		class := classify(in, peers)
		core.HandleDatagram(now, in, &out)
		out.Drain()
		if class == "" {
			if _, ok := in.Msg.(port.Hello); ok {
				peers[in.NodeID] = modelPeer{handle: in.Source, runID: in.RunID}
			}
			continue
		}
		discarded++
		bump(&want, class)
	}

	got := core.Discards()
	if got != want {
		t.Errorf("after %d adversarial datagrams Discards() = %+v, want %+v — counter conservation broken",
			discardRounds, got, want)
	}
	if sum := got.BadCRC + got.Undecodable + got.BadProtoVersion + got.OwnEcho + got.DuplicateNodeID; sum != discarded {
		t.Errorf("counter sum %d != discarded datagrams %d — a datagram was counted twice or processed anyway", sum, discarded)
	}
	if table := core.PeerSnapshot(); len(table) != len(peers) {
		t.Errorf("core peer table holds %d peers, model holds %d — processed and discarded disagree", len(table), len(peers))
	}
}

// discardEvent is one captured Config.OnDiscard call.
type discardEvent struct {
	class  string
	detail string
}

// TestDiscard_onDiscardSeam proves the D-08 seam: every discard branch of
// HandleDatagram fires Config.OnDiscard exactly once with its stable class
// token and a metadata detail naming the evidence (source handle, node id),
// the DiscardStats total classification holds unchanged beside the seam, and
// a healthy datagram fires nothing at all — the seam is additive and the
// counters stay the reliable signal.
func TestDiscard_onDiscardSeam(t *testing.T) {
	t.Parallel()
	now := at(0)
	rows := []struct {
		name          string
		seedPeer1     bool
		in            port.InboundDatagram
		wantEvents    int
		wantClass     string
		wantDetailHas []string
		wantDiscards  repl.DiscardStats
	}{
		{
			name:         "zero-length datagram fires undecodable",
			in:           port.InboundDatagram{Status: port.Undecodable},
			wantEvents:   1,
			wantClass:    repl.DiscardUndecodable,
			wantDiscards: repl.DiscardStats{Undecodable: 1},
		},
		{
			name: "CRC failure fires bad_crc",
			in: port.InboundDatagram{
				Status:       port.BadCRC,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
			},
			wantEvents:   1,
			wantClass:    repl.DiscardBadCRC,
			wantDiscards: repl.DiscardStats{BadCRC: 1},
		},
		{
			name: "proto version mismatch fires bad_proto_version",
			in: port.InboundDatagram{
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion + 1,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
				Msg:          port.Hello{},
			},
			wantEvents:   1,
			wantClass:    repl.DiscardBadProtoVersion,
			wantDiscards: repl.DiscardStats{BadProtoVersion: 1},
		},
		{
			name: "own broadcast echo fires own_broadcast",
			in: port.InboundDatagram{
				Source:       handleFor(localID),
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       localID,
				RunID:        testRunID(localID),
				Msg:          port.Hello{},
			},
			wantEvents:   1,
			wantClass:    repl.DiscardOwnBroadcast,
			wantDiscards: repl.DiscardStats{OwnEcho: 1},
		},
		{
			name: "foreign run claiming our node id fires duplicate_node_id",
			in: port.InboundDatagram{
				Source:       handleFor(peerID1),
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       localID,
				RunID:        testRunID(peerID1),
				Msg:          port.Hello{},
			},
			wantEvents:    1,
			wantClass:     repl.DiscardDuplicateNodeID,
			wantDetailHas: []string{fmt.Sprintf("node_id %d", localID), handleFor(peerID1)},
			wantDiscards:  repl.DiscardStats{DuplicateNodeID: 1},
		},
		{
			name:      "second handle claiming a known identity fires duplicate_node_id naming both handles",
			seedPeer1: true,
			in: port.InboundDatagram{
				Source:       cloneHandle,
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
				Msg:          port.Hello{},
			},
			wantEvents:    1,
			wantClass:     repl.DiscardDuplicateNodeID,
			wantDetailHas: []string{fmt.Sprintf("node_id %d", peerID1), handleFor(peerID1), cloneHandle},
			wantDiscards:  repl.DiscardStats{DuplicateNodeID: 1},
		},
		{
			name:       "a valid Hello from a stranger fires nothing",
			in:         helloFrom(strangerID),
			wantEvents: 0,
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			var events []discardEvent
			fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
			st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})}
			core := repl.New(localID, testRunID(localID), handleFor(localID), st, repl.Config{
				OnDiscard: func(class, detail string) {
					events = append(events, discardEvent{class: class, detail: detail})
				},
			})
			if row.seedPeer1 {
				var seed port.Outbox
				core.HandleDatagram(now, helloFrom(peerID1), &seed)
				seed.Drain()
				events = nil
			}
			var out port.Outbox
			core.HandleDatagram(now, row.in, &out)
			if len(events) != row.wantEvents {
				t.Fatalf("OnDiscard fired %d times, want %d: %+v", len(events), row.wantEvents, events)
			}
			if row.wantEvents == 0 {
				if got := core.Discards(); got != row.wantDiscards {
					t.Errorf("Discards() = %+v, want %+v — the total classification must hold beside the seam", got, row.wantDiscards)
				}
				return
			}
			if events[0].class != row.wantClass {
				t.Fatalf("OnDiscard class = %q, want %q", events[0].class, row.wantClass)
			}
			for _, name := range row.wantDetailHas {
				if !strings.Contains(events[0].detail, name) {
					t.Errorf("OnDiscard detail %q does not name %q", events[0].detail, name)
				}
			}
			if got := core.Discards(); got != row.wantDiscards {
				t.Errorf("Discards() = %+v, want %+v — the total classification must hold beside the seam", got, row.wantDiscards)
			}
			if got := len(out.Drain()); got != 0 {
				t.Errorf("handler emitted %d messages, want none — a discarded datagram is never also processed", got)
			}
		})
	}
}
