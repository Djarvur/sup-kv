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
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Discard-classification test constants: the adversarial round count of the
// totality proof (D-02), the second handle a claiming node announces from
// (REPL-08's two-sources-one-identity signature), and the inbound version
// bound the model mirrors (the core's 2 m default MaxClockLead, strictly
// above the 60 s drift-warn default so the drift window is reachable).
const (
	discardRounds       = 500
	cloneHandle         = "n2-clone"
	defaultMaxClockLead = 2 * time.Minute
	longKeyLen          = storage.MaxKeyBytes + 1

	// tooManySyncChunks is one above the core's maxSyncChunks (65 536), the
	// first rejected chunk count. Kept as a literal because this external
	// test package cannot see the unexported constant.
	tooManySyncChunks = 65537
)

// newDiscardCore returns a fresh local-node core whose Config.Log appends to
// logs — the classification tests' REPL-08 observation seam. Same shape as
// newTestCore (fixed clock millisecond, default Config) except the log hook.
func newDiscardCore(logs *[]string) *repl.Core {
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})}
	cfg := repl.Config{Log: func(msg string) { *logs = append(*logs, msg) }}
	return repl.New(localID, testRunID(localID), st, cfg)
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
		wantWarn    string
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
			name: "proto version supported+1 classifies BadProtoVersion and warns",
			in: port.InboundDatagram{
				Status:       port.Decoded,
				ProtoVersion: repl.SupportedProtoVersion + 1,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
				Msg:          port.Hello{},
			},
			want:     repl.DiscardStats{BadProtoVersion: 1},
			wantWarn: repl.DiscardBadProtoVersion,
		},
		{
			name:     "data sync with chunk_count 0 classifies BadSyncHeader",
			in:       datagramFrom(peerID1, port.DataSync{SyncID: 1, ChunkIndex: 1, ChunkCount: 0}),
			want:     repl.DiscardStats{BadSyncHeader: 1},
			wantWarn: repl.DiscardBadSyncHeader,
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
			// The stranger's first Hello emits its identity echo and enqueues
			// its first-seen listing on the pacer (PROTO-04, CR-01): one
			// unicast Hello plus one empty-store chunk queued.
			wantEmits:  1,
			wantQueued: 1,
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
			// emitted by the handler; the handler outbox carries the one
			// identity echo that lets the peer bind that listing (CR-01). The
			// queued chunk proves the restart reaction ran.
			wantEmits:  1,
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
				// The seed Hello now enqueues its own first-seen listing; drain
				// it so the row's wantQueued counts only the row's own trigger
				// (PROTO-04, RESEARCH Pitfall 4).
				advanceDrain(core, &seed, now)
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
			switch {
			case row.wantLogged:
				assertConfigErrorLog(t, logs, row.logMustName)
			case row.wantWarn != "":
				if len(logs) != 1 || !strings.Contains(logs[0], row.wantWarn) {
					t.Errorf("Config.Log = %v, want exactly one WARN naming %q", logs, row.wantWarn)
				}
			case len(logs) != 0:
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
// and run_id the second duplicate-node_id variant compares against, plus the
// bounded 8-entry history of previously current run_ids the alternating-
// identity rule consults (PROTO-06). Only a processed Hello ever writes it —
// exactly the core's own mutation rule.
type modelPeer struct {
	handle   string
	runID    [16]byte
	seenRuns [8][16]byte
	seenLen  uint8
	seenPos  uint8
}

// pushSeenRun mirrors the core's ring push: the outgoing current run joins
// the bounded history, overwriting the oldest slot when the ring wraps.
func (p *modelPeer) pushSeenRun(run [16]byte) {
	p.seenRuns[p.seenPos] = run
	p.seenPos = (p.seenPos + 1) % 8
	if p.seenLen < 8 {
		p.seenLen++
	}
}

// seenRun mirrors the core's ring lookup, consulting only the valid slots.
func (p *modelPeer) seenRun(run [16]byte) bool {
	for i := range p.seenLen {
		if p.seenRuns[i] == run {
			return true
		}
	}
	return false
}

// classify mirrors the classifier's pinned decision for one datagram against
// the model's current peer table, returning the discard class or "" when the
// datagram must reach protocol processing — the expectedDrift model-function
// precedent, applied to the classification order (D-02) followed by the
// wire-validation pipeline the core runs before handleMessage (XPORT-10,
// D-06). now is the handler's event time, the clock the version bound reads.
// A Hello that survives every classification applies its identity transition
// to the model row here (learn, refresh or restart with the seen-run push),
// so the model stays in lockstep with the core's handleHello.
func classify(in port.InboundDatagram, peers map[uint16]*modelPeer, now time.Time) string {
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
	_, isHello := in.Msg.(port.Hello)
	if known, ok := peers[in.NodeID]; ok {
		switch {
		case in.RunID == known.runID && in.Source != known.handle:
			return "DuplicateNodeID"
		case isHello && in.RunID != known.runID && known.seenRun(in.RunID):
			return "DuplicateNodeID"
		}
	}
	if class := validateClass(in.Msg, now); class != "" {
		return class
	}
	if _, isSync := in.Msg.(port.DataSync); isSync {
		// The retained-listing binding (CR-01): a listing from an identity the
		// table does not hold, or from a run_id that differs from the row's, is
		// the unbound_listing class (New-2) — it never reaches the retention,
		// the completeness flag or the compare path, and it is NOT the
		// duplicate-node_id identity class.
		known, ok := peers[in.NodeID]
		if !ok || known.runID != in.RunID {
			return "UnboundListing"
		}
	}
	if isHello {
		known, ok := peers[in.NodeID]
		if !ok {
			peers[in.NodeID] = &modelPeer{handle: in.Source, runID: in.RunID}
			return ""
		}
		known.handle = in.Source
		if known.runID != in.RunID {
			known.pushSeenRun(known.runID)
			known.runID = in.RunID
		}
	}
	return ""
}

// validateClass mirrors the validation pipeline's pinned intra-message order
// (the core's validate): DataReplication — key, tombstone-with-data,
// checksum, version; DataRequest/Ack — key only; DataSync — per entry key
// then version, the first failure discarding the whole datagram; Hello, a
// standalone SyncEntry and a nil Msg — nothing.
func validateClass(msg port.Message, now time.Time) string {
	switch m := msg.(type) {
	case port.DataReplication:
		if class := keyViolation(m.Key); class != "" {
			return class
		}
		if m.Record.Deleted && len(m.Record.Data) > 0 {
			return "TombstoneWithData"
		}
		sum := storage.RecordChecksum(m.Key, m.Record.Version, m.Record.Data, m.Record.Deleted)
		if sum != m.Record.Checksum {
			return "BadChecksum"
		}
		if futureVersion(now, m.Record.Version) {
			return "BadVersion"
		}
	case port.DataRequest:
		if class := keyViolation(m.Key); class != "" {
			return class
		}
	case port.Ack:
		if class := keyViolation(m.Key); class != "" {
			return class
		}
	case port.DataSync:
		if class := syncHeaderViolation(m); class != "" {
			return class
		}
		for _, entry := range m.Entries {
			if class := keyViolation(entry.Key); class != "" {
				return class
			}
			if futureVersion(now, entry.Version) {
				return "BadVersion"
			}
		}
	}
	return ""
}

// syncHeaderViolation mirrors the completeness-header domain: count at least
// 1, a 1-based index inside the round, and a count at or under the stream
// cap.
func syncHeaderViolation(msg port.DataSync) string {
	if msg.ChunkCount < 1 || msg.ChunkIndex < 1 || msg.ChunkIndex > msg.ChunkCount || msg.ChunkCount > tooManySyncChunks {
		return "BadSyncHeader"
	}
	return ""
}

// keyViolation mirrors the key domain: empty first, then the
// storage.MaxKeyBytes ceiling.
func keyViolation(key string) string {
	if key == "" {
		return "EmptyKey"
	}
	if len(key) > storage.MaxKeyBytes {
		return "KeyTooLong"
	}
	return ""
}

// futureVersion mirrors the version half through the version package's own
// predicate: a layout fault or a reading beyond now+defaultMaxClockLead is
// the bad_version class.
func futureVersion(now time.Time, ver version.RecordVersion) bool {
	return ver.Validate(now, defaultMaxClockLead) != nil
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
	case "UnboundListing":
		stats.UnboundListing++
	case "EmptyKey":
		stats.EmptyKey++
	case "KeyTooLong":
		stats.KeyTooLong++
	case "TombstoneWithData":
		stats.TombstoneWithData++
	case "BadChecksum":
		stats.BadChecksum++
	case "BadVersion":
		stats.BadVersion++
	case "BadSyncHeader":
		stats.BadSyncHeader++
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
// SyncEntry that never travels alone. Keyed messages draw from the whole
// validation domain (empty, just over the ceiling, ordinary) and
// DataReplication records are wire-shaped with one adversarial mutation
// half the time, so the totality proof exercises every validation class.
func randBody(rng *rand.Rand) port.Message {
	switch rng.IntN(7) {
	case 0:
		return nil
	case 1:
		return port.Hello{}
	case 2:
		key := randWireKey(rng)
		rec := testutil.RandRecordForKey(rng, key)
		corruptRecord(rng, &rec)
		return port.DataReplication{Key: key, Record: rec}
	case 3:
		return port.Ack{Key: randWireKey(rng), Version: testutil.RandRawVersion(rng)}
	case 4:
		return port.DataRequest{Key: randWireKey(rng)}
	case 5:
		entries := make([]port.SyncEntry, 1+rng.IntN(4))
		for i := range entries {
			entries[i] = port.SyncEntry{Key: randWireKey(rng), Version: testutil.RandNearVersion(rng)}
		}
		return syncPart(entries...)
	default:
		return port.SyncEntry{Key: randWireKey(rng), Version: testutil.RandNearVersion(rng)}
	}
}

// syncPart stamps the 1-of-1 completeness header on entries — the shape an
// honest sender uses for a listing that fits one chunk.
func syncPart(entries ...port.SyncEntry) port.DataSync {
	return port.DataSync{SyncID: 1, ChunkIndex: 1, ChunkCount: 1, Entries: entries}
}

// randWireKey draws a key from the validation domain: mostly a small-space
// key, sometimes empty and sometimes one byte over storage.MaxKeyBytes, so
// the empty_key and key_too_long classes are reachable by the random draws
// rather than only by the explicit table rows.
func randWireKey(rng *rand.Rand) string {
	switch rng.IntN(16) {
	case 0:
		return ""
	case 1:
		return strings.Repeat("k", longKeyLen)
	default:
		return testutil.RandKey(rng)
	}
}

// corruptRecord applies one adversarial record shape: a flipped checksum bit
// or a tombstone left carrying Data — the two record-shape classes the
// validator owns. Most draws stay wire-valid, so both verdicts occur.
func corruptRecord(rng *rand.Rand, rec *storage.Record) {
	switch rng.IntN(8) {
	case 0:
		rec.Checksum ^= 1
	case 1:
		rec.Deleted = true
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
	peers := map[uint16]*modelPeer{
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
		class := classify(in, peers, now)
		core.HandleDatagram(now, in, &out)
		out.Drain()
		if class == "" {
			// classify already applied the Hello's identity transition to the
			// model row; nothing else mutates the table.
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
	sum := got.BadCRC + got.Undecodable + got.BadProtoVersion + got.OwnEcho + got.DuplicateNodeID +
		got.UnboundListing + got.EmptyKey + got.KeyTooLong + got.TombstoneWithData + got.BadChecksum +
		got.BadVersion + got.BadSyncHeader
	if sum != discarded {
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

// TestDiscard_unboundListingClass proves the CR-01 binding's own class
// (D-04's New-2): a listing from an unknown node id increments
// UnboundListing — never DuplicateNodeID — fires OnDiscard with the
// unbound_listing token and a metadata detail, and produces exactly one WARN
// that tells the unbound-listing story: such a listing may legitimately have
// overtaken its Hello, so the line must not claim a configuration error.
func TestDiscard_unboundListingClass(t *testing.T) {
	t.Parallel()
	now := at(0)
	var (
		logs   []string
		events []discardEvent
	)
	core, _ := newTestCoreWithConfig(repl.Config{
		OnDiscard: func(class, detail string) {
			events = append(events, discardEvent{class: class, detail: detail})
		},
		Log: func(msg string) { logs = append(logs, msg) },
	})
	var out port.Outbox
	entry := port.SyncEntry{Key: "unbound/key", Version: validWireVersion(1)}
	core.HandleDatagram(now, datagramFrom(strangerID, syncPart(entry)), &out)
	out.Drain()

	got := core.Discards()
	if got.UnboundListing != 1 {
		t.Fatalf("UnboundListing = %d after one unbound listing, want 1: %+v", got.UnboundListing, got)
	}
	if got.DuplicateNodeID != 0 {
		t.Fatalf("DuplicateNodeID = %d after one unbound listing, want 0 — the drop is not an identity collision: %+v", got.DuplicateNodeID, got)
	}
	if len(events) != 1 || events[0].class != repl.DiscardUnboundListing {
		t.Fatalf("OnDiscard events = %+v, want exactly one %q event", events, repl.DiscardUnboundListing)
	}
	if !strings.Contains(events[0].detail, fmt.Sprintf("node_id %d", strangerID)) {
		t.Fatalf("OnDiscard detail %q does not name node_id %d", events[0].detail, strangerID)
	}
	if len(logs) != 1 {
		t.Fatalf("Config.Log fired %d time(s) for one unbound listing, want 1: %v", len(logs), logs)
	}
	if !strings.Contains(logs[0], fmt.Sprintf("node_id %d", strangerID)) || !strings.Contains(logs[0], "CR-01") {
		t.Fatalf("the unbound-listing WARN %q does not name node_id %d and the CR-01 binding", logs[0], strangerID)
	}
	if strings.Contains(logs[0], "configuration error") {
		t.Fatalf("the unbound-listing WARN %q claims a configuration error — such a listing may legitimately overtake its Hello", logs[0])
	}
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
			// The adapter's pre-body verdict, consumed as protocol input: the
			// core counts and attributes it exactly like the field check, with
			// no message to process (D-07).
			name: "the adapter's proto-version verdict fires bad_proto_version without a body",
			in: port.InboundDatagram{
				Source:       handleFor(peerID1),
				Status:       port.ProtoVersionMismatch,
				ProtoVersion: repl.SupportedProtoVersion + 1,
				NodeID:       peerID1,
				RunID:        testRunID(peerID1),
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
			core := repl.New(localID, testRunID(localID), st, repl.Config{
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

// wireRecord builds a wire-valid record for key: the checksum is the
// production domain over exactly the given fields, so only a field the test
// deliberately mutates can invalidate it.
func wireRecord(key string, ver version.RecordVersion, data []byte, deleted bool) storage.Record {
	return storage.Record{
		Version:  ver,
		Data:     data,
		Deleted:  deleted,
		Checksum: storage.RecordChecksum(key, ver, data, deleted),
	}
}

// validWireVersion returns a layout-valid version one second inside the 2 m
// default lead.
func validWireVersion(counter uint16) version.RecordVersion {
	return version.PackVersion(testNowMs+1000, counter, peerID1)
}

// futureWireVersion returns a layout-valid version exactly one millisecond
// beyond now+defaultMaxClockLead — the boundary the future check rejects.
func futureWireVersion() version.RecordVersion {
	return version.PackVersion(testNowMs+defaultMaxClockLead.Milliseconds()+1, 0, peerID1)
}

// badLayoutWireVersion returns the all-ones 16-byte version: a version
// nibble of 0xF and variant bits 0b11, the RFC 9562 layout violation the
// poison version class is for.
func badLayoutWireVersion() version.RecordVersion {
	return version.RecordVersion{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
}

// onlyClass returns the DiscardStats with exactly class counted once — the
// token-to-counter mapping pinned on its own.
func onlyClass(class string) repl.DiscardStats {
	var stats repl.DiscardStats
	switch class {
	case repl.DiscardEmptyKey:
		stats.EmptyKey = 1
	case repl.DiscardKeyTooLong:
		stats.KeyTooLong = 1
	case repl.DiscardTombstoneWithData:
		stats.TombstoneWithData = 1
	case repl.DiscardBadChecksum:
		stats.BadChecksum = 1
	case repl.DiscardBadVersion:
		stats.BadVersion = 1
	case repl.DiscardBadSyncHeader:
		stats.BadSyncHeader = 1
	case repl.DiscardUnboundListing:
		stats.UnboundListing = 1
	}
	return stats
}

// sumDiscards totals every discard class — the conservation check across all
// twelve counters.
func sumDiscards(stats repl.DiscardStats) uint64 {
	return stats.BadCRC + stats.Undecodable + stats.BadProtoVersion + stats.OwnEcho +
		stats.DuplicateNodeID + stats.UnboundListing + stats.EmptyKey + stats.KeyTooLong +
		stats.TombstoneWithData + stats.BadChecksum + stats.BadVersion + stats.BadSyncHeader
}

// validationRow is one TestDiscard_validationClasses table row: the datagram,
// the class it must be discarded as ("" = the row is an accepted control),
// key material that must never reach a detail or a log line, and — for an
// accepted record — the key the store must then serve.
type validationRow struct {
	name       string
	in         port.InboundDatagram
	wantClass  string
	secret     string
	wantStored string
}

// TestDiscard_validationClasses pins the wire-validation pipeline row by row
// (XPORT-03/04/10, D-06): one row per class — including the DataSync-entry
// rows and a mixed listing whose second entry invalidates the whole datagram
// — asserts the class token, the exact counter set, the detail contract (the
// source handle only, never key material), and that the datagram reached
// neither the store nor the clock and ran no handler. The order rows carry
// two simultaneous faults and pin which one is reported. Two control rows
// prove the pipeline is not rejecting everything: a valid record is applied
// (and Acknowledged) and an Ack carrying a future version is processed,
// because Ack validates its key and nothing else.
func TestDiscard_validationClasses(t *testing.T) {
	t.Parallel()
	now := at(0)
	longKey := strings.Repeat("s3cr3t-key-material/", 100)[:storage.MaxKeyBytes+1]

	// Table-local fixtures, built once: every record is wire-valid except
	// the specific field its row mutates.
	emptyKeyRec := wireRecord("", validWireVersion(1), []byte("empty-value"), false)
	longKeyRec := wireRecord(longKey, validWireVersion(2), []byte("long-value"), false)
	tombstoneData := wireRecord("neg/tombstone", validWireVersion(3), []byte("payload"), true)
	badChecksum := wireRecord("neg/checksum", validWireVersion(4), []byte("payload"), false)
	badChecksum.Checksum ^= 1
	futureRec := wireRecord("neg/future", futureWireVersion(), []byte("payload"), false)
	futureNoChecksum := futureRec
	futureNoChecksum.Checksum ^= 1
	badLayoutRec := wireRecord("neg/layout", badLayoutWireVersion(), []byte("payload"), false)

	rows := []validationRow{
		{
			name:      "data replication with an empty key",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "", Record: emptyKeyRec}),
			wantClass: repl.DiscardEmptyKey,
		},
		{
			name:      "data replication with a key over the limit",
			in:        datagramFrom(peerID1, port.DataReplication{Key: longKey, Record: longKeyRec}),
			wantClass: repl.DiscardKeyTooLong,
			secret:    longKey,
		},
		{
			name:      "data replication tombstone carrying data",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "neg/tombstone", Record: tombstoneData}),
			wantClass: repl.DiscardTombstoneWithData,
			secret:    "neg/tombstone",
		},
		{
			name:      "data replication with a forged checksum",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "neg/checksum", Record: badChecksum}),
			wantClass: repl.DiscardBadChecksum,
			secret:    "neg/checksum",
		},
		{
			name:      "data replication with a far-future version",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "neg/future", Record: futureRec}),
			wantClass: repl.DiscardBadVersion,
			secret:    "neg/future",
		},
		{
			name:      "data replication with a bad version layout",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "neg/layout", Record: badLayoutRec}),
			wantClass: repl.DiscardBadVersion,
			secret:    "neg/layout",
		},
		{
			name:      "order: an empty key is reported before the record checksum",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "", Record: func() storage.Record { rec := emptyKeyRec; rec.Checksum ^= 1; return rec }()}),
			wantClass: repl.DiscardEmptyKey,
		},
		{
			name:      "order: tombstone-with-data is reported before the record checksum",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "neg/tombstone", Record: func() storage.Record { rec := tombstoneData; rec.Checksum ^= 1; return rec }()}),
			wantClass: repl.DiscardTombstoneWithData,
			secret:    "neg/tombstone",
		},
		{
			name:      "order: a bad checksum is reported before a far-future version",
			in:        datagramFrom(peerID1, port.DataReplication{Key: "neg/future", Record: futureNoChecksum}),
			wantClass: repl.DiscardBadChecksum,
			secret:    "neg/future",
		},
		{
			name:      "data request with an empty key",
			in:        datagramFrom(peerID1, port.DataRequest{Key: ""}),
			wantClass: repl.DiscardEmptyKey,
		},
		{
			name:      "data request with a key over the limit",
			in:        datagramFrom(peerID1, port.DataRequest{Key: longKey}),
			wantClass: repl.DiscardKeyTooLong,
			secret:    longKey,
		},
		{
			name:      "ack with an empty key",
			in:        datagramFrom(peerID1, port.Ack{Key: "", Version: validWireVersion(5)}),
			wantClass: repl.DiscardEmptyKey,
		},
		{
			name:      "ack with a key over the limit",
			in:        datagramFrom(peerID1, port.Ack{Key: longKey, Version: validWireVersion(6)}),
			wantClass: repl.DiscardKeyTooLong,
			secret:    longKey,
		},
		{
			name: "data sync entry with an empty key",
			in: datagramFrom(peerID1, syncPart(
				port.SyncEntry{Key: "", Version: validWireVersion(7)},
			)),
			wantClass: repl.DiscardEmptyKey,
		},
		{
			name: "data sync entry with a key over the limit",
			in: datagramFrom(peerID1, syncPart(
				port.SyncEntry{Key: longKey, Version: validWireVersion(8)},
			)),
			wantClass: repl.DiscardKeyTooLong,
			secret:    longKey,
		},
		{
			name: "data sync entry with a far-future version",
			in: datagramFrom(peerID1, syncPart(
				port.SyncEntry{Key: "sync/future", Version: futureWireVersion()},
			)),
			wantClass: repl.DiscardBadVersion,
			secret:    "sync/future",
		},
		{
			name: "mixed data sync: the second entry invalidates the whole datagram",
			in: datagramFrom(peerID1, syncPart(
				port.SyncEntry{Key: "sync/valid", Version: validWireVersion(9)},
				port.SyncEntry{Key: "sync/poison", Version: futureWireVersion()},
			)),
			wantClass: repl.DiscardBadVersion,
			secret:    "sync/poison",
		},
		{
			name: "order: an earlier entry's version is reported before a later entry's key",
			in: datagramFrom(peerID1, syncPart(
				port.SyncEntry{Key: "sync/future", Version: futureWireVersion()},
				port.SyncEntry{Key: "", Version: validWireVersion(10)},
			)),
			wantClass: repl.DiscardBadVersion,
			secret:    "sync/future",
		},
		{
			name:       "control: a valid record is applied and acknowledged",
			in:         datagramFrom(peerID1, port.DataReplication{Key: "control/valid", Record: wireRecord("control/valid", validWireVersion(11), []byte("value"), false)}),
			wantStored: "control/valid",
		},
		{
			name: "control: an ack's version is not validated",
			in:   datagramFrom(peerID1, port.Ack{Key: "control/ack", Version: futureWireVersion()}),
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			var events []discardEvent
			var logs []string
			clk := &countingClock{inner: clock.NewClock(func() time.Time { return time.UnixMilli(testNowMs) }, 0, 0, testDriftWarn)}
			st := &stateStore{store: storage.NewStore(localID, clk, storage.Config{})}
			core := repl.New(localID, testRunID(localID), st, repl.Config{
				OnDiscard: func(class, detail string) { events = append(events, discardEvent{class: class, detail: detail}) },
				Log:       func(msg string) { logs = append(logs, msg) },
			})
			var out port.Outbox
			core.HandleDatagram(now, row.in, &out)
			emitted := out.Drain()

			if row.wantClass == "" {
				assertAcceptedDatagram(t, row, core, st, clk, events)
				return
			}
			assertDiscardedDatagram(t, row, core, st, clk, events, logs, emitted)
		})
	}
}

// assertAcceptedDatagram asserts the positive control: no discard event, no
// counter movement, and — when the row names a key — the record really
// reached the store through a clock-observed apply.
func assertAcceptedDatagram(t *testing.T, row validationRow, core *repl.Core, st *stateStore, clk *countingClock, events []discardEvent) {
	t.Helper()
	if len(events) != 0 {
		t.Fatalf("accepted datagram fired %d discard event(s), want none: %+v", len(events), events)
	}
	if got := core.Discards(); got != (repl.DiscardStats{}) {
		t.Errorf("Discards() = %+v, want none for an accepted datagram", got)
	}
	if row.wantStored == "" {
		return
	}
	if _, ok := st.store.Lookup(row.wantStored); !ok {
		t.Errorf("store does not hold %q after an accepted datagram — the pipeline rejected valid input", row.wantStored)
	}
	if clk.updates == 0 {
		t.Error("the accepted record's apply never reached the clock")
	}
}

// assertDiscardedDatagram asserts one rejection's full contract: the class
// and its exact counter, the source-only detail, no handler side effects (no
// emission, no queued send, no sync flag), zero clock calls, an untouched
// store, and exactly one rate-limited WARN naming only the class.
func assertDiscardedDatagram(t *testing.T, row validationRow, core *repl.Core, st *stateStore, clk *countingClock, events []discardEvent, logs []string, emitted []port.Outbound) {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("OnDiscard fired %d times, want exactly 1: %+v", len(events), events)
	}
	if events[0].class != row.wantClass {
		t.Fatalf("OnDiscard class = %q, want %q", events[0].class, row.wantClass)
	}
	if events[0].detail != row.in.Source {
		t.Fatalf("OnDiscard detail = %q, want the source handle %q only (T-6-05 contract)", events[0].detail, row.in.Source)
	}
	if row.secret != "" && strings.Contains(events[0].detail, row.secret) {
		t.Errorf("OnDiscard detail %q leaks key material", events[0].detail)
	}
	if got, want := core.Discards(), onlyClass(row.wantClass); got != want {
		t.Errorf("Discards() = %+v, want exactly %+v — one datagram, one class", got, want)
	}
	if len(emitted) != 0 {
		t.Errorf("handler emitted %d message(s) for a discarded datagram, want none: %v", len(emitted), emitted)
	}
	if got := core.QueuedSends(); got != 0 {
		t.Errorf("QueuedSends() = %d, want none — no handler may run for a discarded datagram", got)
	}
	if core.SyncArrived() {
		t.Error("SyncArrived() = true — handleDataSync ran for a discarded listing")
	}
	if clk.ticks != 0 || clk.updates != 0 {
		t.Errorf("clock calls (Tick %d, Update %d) for a discarded datagram, want zero of each (T-3-24)", clk.ticks, clk.updates)
	}
	if got := len(st.store.Records()); got != 0 {
		t.Errorf("store holds %d records after a discarded datagram, want none", got)
	}
	if len(logs) != 1 {
		t.Fatalf("Config.Log fired %d time(s), want one rate-limited WARN: %v", len(logs), logs)
	}
	if !strings.Contains(logs[0], row.wantClass) {
		t.Errorf("WARN %q does not name its class %q", logs[0], row.wantClass)
	}
	if row.secret != "" && strings.Contains(logs[0], row.secret) {
		t.Errorf("WARN %q leaks key material", logs[0])
	}
}

// TestDiscard_badSyncHeader proves the listing-header class (PROTO-01,
// T-4-02): each of the four header-domain violations is discarded as
// bad_sync_header, counted once, WARNed once, with a detail naming the
// offending values — and no clock call, no store write and no gate opening
// happens. The last two rows prove the pipeline position: a valid part after
// a rejected one still accumulates (the rejection created no stream), and a
// part whose chunk_count disagrees with a live round is rejected without
// disturbing the round's stored count.
func TestDiscard_badSyncHeader(t *testing.T) {
	t.Parallel()
	now := at(0)
	entry := port.SyncEntry{Key: "hdr/key", Version: validWireVersion(1)}
	rows := []struct {
		name       string
		in         port.InboundDatagram
		wantDetail []string
	}{
		{
			name:       "chunk_count 0",
			in:         datagramFrom(peerID1, port.DataSync{SyncID: 1, ChunkIndex: 0, ChunkCount: 0, Entries: []port.SyncEntry{entry}}),
			wantDetail: []string{"chunk_count 0"},
		},
		{
			name:       "chunk_index 0",
			in:         datagramFrom(peerID1, port.DataSync{SyncID: 1, ChunkIndex: 0, ChunkCount: 2, Entries: []port.SyncEntry{entry}}),
			wantDetail: []string{"chunk_index 0"},
		},
		{
			name:       "chunk_index past chunk_count",
			in:         datagramFrom(peerID1, port.DataSync{SyncID: 1, ChunkIndex: 3, ChunkCount: 2, Entries: []port.SyncEntry{entry}}),
			wantDetail: []string{"chunk_index 3", "chunk_count 2"},
		},
		{
			name:       "chunk_count above the stream cap",
			in:         datagramFrom(peerID1, port.DataSync{SyncID: 1, ChunkIndex: 1, ChunkCount: tooManySyncChunks, Entries: []port.SyncEntry{entry}}),
			wantDetail: []string{fmt.Sprintf("chunk_count %d", tooManySyncChunks)},
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			var events []discardEvent
			var logs []string
			clk := &countingClock{inner: clock.NewClock(func() time.Time { return time.UnixMilli(testNowMs) }, 0, 0, testDriftWarn)}
			st := &stateStore{store: storage.NewStore(localID, clk, storage.Config{})}
			core := repl.New(localID, testRunID(localID), st, repl.Config{
				OnDiscard: func(class, detail string) { events = append(events, discardEvent{class: class, detail: detail}) },
				Log:       func(msg string) { logs = append(logs, msg) },
			})
			var out port.Outbox
			core.HandleDatagram(now, row.in, &out)
			out.Drain()

			if got, want := core.Discards(), (repl.DiscardStats{BadSyncHeader: 1}); got != want {
				t.Fatalf("Discards() = %+v, want exactly %+v — one datagram, one class", got, want)
			}
			if len(events) != 1 || events[0].class != repl.DiscardBadSyncHeader {
				t.Fatalf("OnDiscard events = %+v, want exactly one %q event", events, repl.DiscardBadSyncHeader)
			}
			for _, want := range row.wantDetail {
				if !strings.Contains(events[0].detail, want) {
					t.Errorf("OnDiscard detail %q does not name %q — the header verdict must name the offending values", events[0].detail, want)
				}
			}
			if core.SyncArrived() || core.ListingComplete() {
				t.Fatal("a rejected header opened the gate, want both flags false — no accumulation may run")
			}
			if clk.ticks != 0 || clk.updates != 0 {
				t.Errorf("clock calls (Tick %d, Update %d) for a rejected header, want zero of each", clk.ticks, clk.updates)
			}
			if got := len(st.store.Records()); got != 0 {
				t.Errorf("store holds %d records after a rejected header, want none", got)
			}
			if len(logs) != 1 || !strings.Contains(logs[0], repl.DiscardBadSyncHeader) {
				t.Fatalf("Config.Log = %v, want exactly one WARN naming %q", logs, repl.DiscardBadSyncHeader)
			}
		})
	}

	t.Run("a valid part after a rejected one still accumulates", func(t *testing.T) {
		t.Parallel()
		core, _ := newTestCore()
		var out port.Outbox
		core.Advance(now, &out)
		out.Drain()
		core.HandleDatagram(now, helloFrom(peerID1), &out) // a listing is bound to a learned identity (CR-01)
		out.Drain()

		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{SyncID: 5, ChunkIndex: 0, ChunkCount: 0, Entries: []port.SyncEntry{entry}}), &out)
		out.Drain()
		if got, want := core.Discards(), (repl.DiscardStats{BadSyncHeader: 1}); got != want {
			t.Fatalf("Discards() = %+v, want %+v after the rejected part", got, want)
		}
		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{SyncID: 5, ChunkIndex: 1, ChunkCount: 2, Entries: []port.SyncEntry{entry}}), &out)
		out.Drain()
		if core.SyncArrived() {
			t.Fatal("SyncArrived() = true after chunk 1 of 2, want false — the rejected part must not have completed anything")
		}
		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{SyncID: 5, ChunkIndex: 2, ChunkCount: 2, Entries: []port.SyncEntry{entry}}), &out)
		out.Drain()
		if !core.SyncArrived() {
			t.Fatal("SyncArrived() = false after the valid round completed, want true — the rejected part did not poison the stream")
		}
		if got, want := core.Discards(), (repl.DiscardStats{BadSyncHeader: 1}); got != want {
			t.Fatalf("Discards() = %+v, want %+v — the valid parts must not add a discard", got, want)
		}
	})

	t.Run("a part disagreeing with the live round keeps the round's count", func(t *testing.T) {
		t.Parallel()
		core, _ := newTestCore()
		var out port.Outbox
		core.Advance(now, &out)
		out.Drain()
		core.HandleDatagram(now, helloFrom(peerID1), &out) // a listing is bound to a learned identity (CR-01)
		out.Drain()

		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{SyncID: 7, ChunkIndex: 1, ChunkCount: 3, Entries: []port.SyncEntry{entry}}), &out)
		out.Drain()
		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{SyncID: 7, ChunkIndex: 2, ChunkCount: 5, Entries: []port.SyncEntry{entry}}), &out)
		out.Drain()
		if got, want := core.Discards(), (repl.DiscardStats{BadSyncHeader: 1}); got != want {
			t.Fatalf("Discards() = %+v, want %+v after the disagreeing part", got, want)
		}
		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{SyncID: 7, ChunkIndex: 2, ChunkCount: 3, Entries: []port.SyncEntry{entry}}), &out)
		out.Drain()
		if core.SyncArrived() {
			t.Fatal("SyncArrived() = true after chunks 1 and 2 of 3, want false")
		}
		core.HandleDatagram(now, datagramFrom(peerID1, port.DataSync{SyncID: 7, ChunkIndex: 3, ChunkCount: 3, Entries: []port.SyncEntry{entry}}), &out)
		out.Drain()
		if !core.SyncArrived() {
			t.Fatal("SyncArrived() = false after the round completed at its original count 3, want true — the disagreeing part must not have replaced the stored count")
		}
	})
}

// TestDiscard_warnRateLimitedPerPeer proves the WARN limiter's key is
// (node_id, class) on event time (D-06/D-07, T-4-04): one peer's repeated
// foreign-version flood WARNs once per interval, while a second peer's first
// occurrence inside the same interval gets its own WARN — one peer can never
// suppress another's signal. The counters and the per-datagram OnDiscard
// seam are untouched by the limiter.
func TestDiscard_warnRateLimitedPerPeer(t *testing.T) {
	t.Parallel()
	now := at(0)
	const warnInterval = 10 * time.Second // the pinned discardWarnInterval
	var logs []string
	var events []discardEvent
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})}
	core := repl.New(localID, testRunID(localID), st, repl.Config{
		OnDiscard: func(class, detail string) { events = append(events, discardEvent{class: class, detail: detail}) },
		Log:       func(msg string) { logs = append(logs, msg) },
	})
	var out port.Outbox
	feed := func(peer uint16, at time.Time) {
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

	feed(peerID1, now)
	feed(peerID1, now)
	feed(peerID2, now)
	if len(logs) != 2 {
		t.Fatalf("Config.Log fired %d time(s) for two peers in one interval, want 2 (one per peer): %v", len(logs), logs)
	}
	if len(events) != 3 {
		t.Fatalf("OnDiscard fired %d time(s), want 3 — the seam stays per datagram", len(events))
	}
	if got := core.Discards().BadProtoVersion; got != 3 {
		t.Fatalf("BadProtoVersion = %d, want 3 — the limiter never suppresses the counter", got)
	}

	feed(peerID2, now.Add(warnInterval-time.Nanosecond))
	if len(logs) != 2 {
		t.Fatalf("Config.Log fired %d time(s) inside the interval, want still 2: %v", len(logs), logs)
	}
	feed(peerID1, now.Add(warnInterval))
	if len(logs) != 3 {
		t.Fatalf("Config.Log fired %d time(s) at peer 1's interval boundary, want 3: %v", len(logs), logs)
	}
	feed(peerID2, now.Add(warnInterval))
	if len(logs) != 4 {
		t.Fatalf("Config.Log fired %d time(s) at peer 2's interval boundary, want 4: %v", len(logs), logs)
	}
	for _, event := range events {
		if event.class != repl.DiscardBadProtoVersion {
			t.Errorf("OnDiscard class = %q, want %q", event.class, repl.DiscardBadProtoVersion)
		}
		if event.detail != handleFor(peerID1) && event.detail != handleFor(peerID2) {
			t.Errorf("OnDiscard detail = %q, want the datagram's source handle", event.detail)
		}
	}
}

// otherwise delegates to a real clock, so a test asserts an invalid datagram
// never reaches the clock at all — not merely that its reading was unused.
type countingClock struct {
	inner   *clock.Clock
	ticks   int
	updates int
}

// Tick counts the call and delegates the reading.
func (c *countingClock) Tick() (int64, uint16, time.Duration) {
	c.ticks++
	return c.inner.Tick()
}

// Update counts the call and delegates the reading.
func (c *countingClock) Update(remoteL int64, remoteC uint16) (int64, uint16, time.Duration) {
	c.updates++
	return c.inner.Update(remoteL, remoteC)
}

// TestDiscard_beforeClock is the T-3-24 acceptance: one datagram per invalid
// class (and per invalid DataSync entry shape) is fed to a single core over a
// store whose clock counts every call, and after each feed the clock shows
// zero Tick and zero Update, the store is empty, the sync flag is false, and
// exactly one more discard has been counted. Validation is therefore strictly
// before any clock or store call, not merely before the value is used.
func TestDiscard_beforeClock(t *testing.T) {
	t.Parallel()
	now := at(0)
	longKey := strings.Repeat("clock-proof/", 100)[:storage.MaxKeyBytes+1]

	invalid := []struct {
		name string
		in   port.InboundDatagram
	}{
		{"empty key replication", datagramFrom(peerID1, port.DataReplication{
			Key: "", Record: wireRecord("", validWireVersion(1), []byte("v"), false),
		})},
		{"long key replication", datagramFrom(peerID1, port.DataReplication{
			Key: longKey, Record: wireRecord(longKey, validWireVersion(2), []byte("v"), false),
		})},
		{"tombstone with data", datagramFrom(peerID1, port.DataReplication{
			Key: "t/tombstone", Record: wireRecord("t/tombstone", validWireVersion(3), []byte("v"), true),
		})},
		{"bad checksum", datagramFrom(peerID1, port.DataReplication{
			Key: "t/checksum", Record: func() storage.Record {
				rec := wireRecord("t/checksum", validWireVersion(4), []byte("v"), false)
				rec.Checksum++
				return rec
			}(),
		})},
		{"future version", datagramFrom(peerID1, port.DataReplication{
			Key: "t/future", Record: wireRecord("t/future", futureWireVersion(), []byte("v"), false),
		})},
		{"bad layout version", datagramFrom(peerID1, port.DataReplication{
			Key: "t/layout", Record: wireRecord("t/layout", badLayoutWireVersion(), []byte("v"), false),
		})},
		{"empty key request", datagramFrom(peerID1, port.DataRequest{Key: ""})},
		{"long key request", datagramFrom(peerID1, port.DataRequest{Key: longKey})},
		{"empty key ack", datagramFrom(peerID1, port.Ack{Key: "", Version: validWireVersion(5)})},
		{"long key ack", datagramFrom(peerID1, port.Ack{Key: longKey, Version: validWireVersion(6)})},
		{"sync empty key", datagramFrom(peerID1, syncPart(
			port.SyncEntry{Key: "", Version: validWireVersion(7)},
		))},
		{"sync long key", datagramFrom(peerID1, syncPart(
			port.SyncEntry{Key: longKey, Version: validWireVersion(8)},
		))},
		{"sync future version", datagramFrom(peerID1, syncPart(
			port.SyncEntry{Key: "t/sync-future", Version: futureWireVersion()},
		))},
		{"sync valid then poisoned entry", datagramFrom(peerID1, syncPart(
			port.SyncEntry{Key: "t/sync-ok", Version: validWireVersion(9)},
			port.SyncEntry{Key: "t/sync-poison", Version: futureWireVersion()},
		))},
		{"sync header chunk_count 0", datagramFrom(peerID1, port.DataSync{
			SyncID: 1, ChunkIndex: 0, ChunkCount: 0,
			Entries: []port.SyncEntry{{Key: "t/sync-header", Version: validWireVersion(10)}},
		})},
		{"sync header index past count", datagramFrom(peerID1, port.DataSync{
			SyncID: 1, ChunkIndex: 3, ChunkCount: 2,
			Entries: []port.SyncEntry{{Key: "t/sync-header", Version: validWireVersion(11)}},
		})},
	}

	clk := &countingClock{inner: clock.NewClock(func() time.Time { return time.UnixMilli(testNowMs) }, 0, 0, testDriftWarn)}
	st := &stateStore{store: storage.NewStore(localID, clk, storage.Config{})}
	core := repl.New(localID, testRunID(localID), st, repl.Config{})
	var out port.Outbox
	for i, tc := range invalid {
		core.HandleDatagram(now, tc.in, &out)
		out.Drain()
		if got := sumDiscards(core.Discards()); got != uint64(i+1) {
			t.Fatalf("after %q the discard total is %d, want %d — one datagram, one class", tc.name, got, i+1)
		}
		if clk.ticks != 0 || clk.updates != 0 {
			t.Fatalf("after %q the clock was called (Tick %d, Update %d), want zero of each (T-3-24)", tc.name, clk.ticks, clk.updates)
		}
		if got := len(st.store.Records()); got != 0 {
			t.Fatalf("after %q the store holds %d records, want none", tc.name, got)
		}
		if core.SyncArrived() {
			t.Fatalf("after %q SyncArrived() = true — a discarded listing reached handleDataSync", tc.name)
		}
	}
}

// TestDiscard_warnRateLimited pins T-3-25's rate limiter: the validation WARN
// is per class and driven by the handler's event time — one WARN per class
// per 10 s, no matter how many datagrams flood in, with the boundary firing
// at exactly the interval (`>=`). The per-datagram OnDiscard seam still fires
// for every datagram (it is the counter path), and neither the WARN nor the
// detail carries key material.
func TestDiscard_warnRateLimited(t *testing.T) {
	t.Parallel()
	now := at(0)
	const warnInterval = 10 * time.Second // the pinned discardWarnInterval
	longKey := strings.Repeat("warn-secret/", 100)[:storage.MaxKeyBytes+1]

	var logs []string
	var events []discardEvent
	clk := &countingClock{inner: clock.NewClock(func() time.Time { return time.UnixMilli(testNowMs) }, 0, 0, testDriftWarn)}
	st := &stateStore{store: storage.NewStore(localID, clk, storage.Config{})}
	core := repl.New(localID, testRunID(localID), st, repl.Config{
		OnDiscard: func(class, detail string) { events = append(events, discardEvent{class: class, detail: detail}) },
		Log:       func(msg string) { logs = append(logs, msg) },
	})
	var out port.Outbox
	feedLongKey := func(at time.Time) {
		t.Helper()
		core.HandleDatagram(at, datagramFrom(peerID1, port.DataReplication{
			Key: longKey, Record: wireRecord(longKey, validWireVersion(1), []byte("v"), false),
		}), &out)
		out.Drain()
	}
	feedEmptyKey := func(at time.Time) {
		t.Helper()
		core.HandleDatagram(at, datagramFrom(peerID1, port.DataReplication{
			Key: "", Record: wireRecord("", validWireVersion(2), []byte("v"), false),
		}), &out)
		out.Drain()
	}

	// Six key_too_long datagrams inside one interval: exactly one WARN, six
	// per-datagram seam events.
	for range 3 {
		feedLongKey(now)
	}
	feedLongKey(now.Add(warnInterval - time.Nanosecond))
	if len(logs) != 1 {
		t.Fatalf("Config.Log fired %d time(s) inside one interval, want 1: %v", len(logs), logs)
	}
	if len(events) != 4 {
		t.Fatalf("OnDiscard fired %d time(s), want 4 — the seam is per datagram, the WARN is the limited surface", len(events))
	}

	// Exactly at the interval the class warns again, and a fresh class gets
	// its own slot immediately.
	feedLongKey(now.Add(warnInterval))
	if len(logs) != 2 {
		t.Fatalf("Config.Log fired %d time(s) at the interval boundary, want 2: %v", len(logs), logs)
	}
	feedEmptyKey(now.Add(warnInterval))
	if len(logs) != 3 {
		t.Fatalf("Config.Log fired %d time(s) after a second class, want 3 (per-class slots): %v", len(logs), logs)
	}
	for _, line := range logs {
		if !strings.Contains(line, "class ") {
			t.Errorf("WARN %q does not name its class", line)
		}
		if strings.Contains(line, longKey) || strings.Contains(line, "warn-secret") {
			t.Errorf("WARN %q leaks key material", line)
		}
	}
	for _, event := range events {
		if event.detail != handleFor(peerID1) {
			t.Errorf("OnDiscard detail = %q, want the source handle only", event.detail)
		}
	}
	if got := sumDiscards(core.Discards()); got != 6 {
		t.Errorf("discard total = %d, want 6 — the limiter never suppresses the counter", got)
	}
	if clk.ticks != 0 || clk.updates != 0 {
		t.Errorf("clock calls (Tick %d, Update %d) for discarded datagrams, want zero of each", clk.ticks, clk.updates)
	}
}

// TestProtoVersionWarns pins the two foreign-proto_version branches as one
// behaviour (OBSV-06, S-FAIL-17, T-07-24): the adapter's ProtoVersionMismatch
// verdict and the defence-in-depth Decoded verdict whose envelope field
// differs each count one BadProtoVersion, fire OnDiscard with the stable
// class and emit exactly one rate-limited WARN — a foreign version is never
// silent on either path. A repeat inside the interval raises the counter and
// stays silent, so the WARN belongs to the limiter, never a per-datagram
// flood.
func TestProtoVersionWarns(t *testing.T) {
	t.Parallel()
	now := at(0)
	var (
		logs   []string
		events []discardEvent
	)
	core, _ := newTestCoreWithConfig(repl.Config{
		Log: func(msg string) { logs = append(logs, msg) },
		OnDiscard: func(class, detail string) {
			events = append(events, discardEvent{class: class, detail: detail})
		},
	})
	var out port.Outbox

	feedVerdict := func(peer uint16) {
		t.Helper()
		core.HandleDatagram(now, port.InboundDatagram{
			Source:       handleFor(peer),
			Status:       port.ProtoVersionMismatch,
			ProtoVersion: repl.SupportedProtoVersion + 1,
			NodeID:       peer,
			RunID:        testRunID(peer),
		}, &out)
		out.Drain()
	}
	feedDecoded := func(peer uint16) {
		t.Helper()
		core.HandleDatagram(now, port.InboundDatagram{
			Source:       handleFor(peer),
			Status:       port.Decoded,
			ProtoVersion: repl.SupportedProtoVersion + 1,
			NodeID:       peer,
			RunID:        testRunID(peer),
			Msg:          port.Hello{},
		}, &out)
		out.Drain()
	}
	warnsFor := func(peer uint16) int {
		want := fmt.Sprintf("class %s from node %d", repl.DiscardBadProtoVersion, peer)
		n := 0
		for _, line := range logs {
			if strings.Contains(line, want) {
				n++
			}
		}
		return n
	}

	// Branch one: the adapter's framing verdict.
	feedVerdict(peerID1)
	if got := core.Discards().BadProtoVersion; got != 1 {
		t.Fatalf("BadProtoVersion after the ProtoVersionMismatch verdict = %d, want 1", got)
	}
	if got := warnsFor(peerID1); got != 1 {
		t.Fatalf("the ProtoVersionMismatch branch emitted %d WARN(s), want exactly 1: %v", got, logs)
	}

	// Branch two: a decoded envelope whose field still differs — the branch
	// that used to count silently.
	feedDecoded(peerID2)
	if got := core.Discards().BadProtoVersion; got != 2 {
		t.Fatalf("BadProtoVersion after the Decoded mismatch = %d, want 2 — the second branch must count", got)
	}
	if got := warnsFor(peerID2); got != 1 {
		t.Fatalf("the Decoded-mismatch branch emitted %d WARN(s), want exactly 1: %v", got, logs)
	}
	if got := len(logs); got != 2 {
		t.Fatalf("Config.Log fired %d time(s) for one datagram per branch, want 2: %v", got, logs)
	}

	// The shared limiter keys the repeats: the counter keeps moving, the log
	// does not.
	feedVerdict(peerID1)
	if got := core.Discards().BadProtoVersion; got != 3 {
		t.Fatalf("BadProtoVersion after a repeat inside the interval = %d, want 3 — the limiter never suppresses the counter", got)
	}
	if got := len(logs); got != 2 {
		t.Fatalf("Config.Log fired %d time(s) after the repeat, want still 2: %v", got, logs)
	}
	if got := len(events); got != 3 {
		t.Fatalf("OnDiscard fired %d time(s), want 3 — the seam stays per datagram", got)
	}
	for _, ev := range events {
		if ev.class != repl.DiscardBadProtoVersion {
			t.Fatalf("OnDiscard class = %q, want %q", ev.class, repl.DiscardBadProtoVersion)
		}
	}
}
