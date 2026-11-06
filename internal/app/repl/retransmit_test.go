package repl_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Shared test constants (the store-test and udprepl-test precedents): a
// fixed wall-clock millisecond (the D-02 seam), the local writer's node ID,
// two peer node IDs, a stranger node ID that is never Hello'd, the store
// clock's drift tolerance, and the keys the retransmission tests write.
const (
	testNowMs  = int64(1730000000000)
	localID    = uint16(1)
	peerID1    = uint16(2)
	peerID2    = uint16(3)
	strangerID = uint16(99)

	testDriftWarn = time.Minute

	keyAlpha = "cfg/alpha"
	keyBeta  = "cfg/beta"
	keyGamma = "cfg/gamma"

	// Ladder timings from docs/IDEA.md §Parameters (the normalized Config
	// defaults): the per-key backoff starts at ack_timeout, doubles per
	// retry and saturates at ack_timeout_max; retransmit_interval is the
	// scan granularity. The suspension window is the long empty-table
	// stretch TestRetransmit_suspendAndFlush drives across.
	ackTimeout    = 200 * time.Millisecond
	ackTimeoutMax = 5 * time.Second
	suspendWindow = 60 * time.Second
)

// testRunID returns a per-node deterministic 16-byte run ID; distinctness
// between node IDs is the only property the tests rely on.
func testRunID(nodeID uint16) [16]byte {
	var id [16]byte
	id[0] = byte(nodeID)
	id[15] = 0x5A
	return id
}

// handleFor returns the opaque source handle of nodeID.
func handleFor(nodeID uint16) string {
	return fmt.Sprintf("n%d", nodeID)
}

// at returns the virtual time offset from the fixed test base, built on
// time.UnixMilli so no monotonic reading is involved (D-02).
func at(off time.Duration) time.Time {
	return time.UnixMilli(testNowMs).Add(off)
}

// stateStore adapts *storage.Store to repl.StateStore — the Phase 4 consumer
// seam — by delegating both methods (the production applier is the persist
// engine's ApplyRemote, plan 04-06).
type stateStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s stateStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Records returns the full snapshot, tombstones included.
func (s stateStore) Records() map[string]storage.Record { return s.store.Records() }

// newTestCore returns the local node's core over a real store whose clock
// parks at the fixed test millisecond (the D-02 seam), carrying the default
// replication Config (docs/IDEA.md §Parameters: ack_timeout 200ms,
// ack_timeout_max 5s, retransmit_interval 100ms, peer_timeout 5s,
// hello_interval 1s) — the exact timings the ladder and eviction tests
// assert against. Peers are always synthetic datagram sources, never cores.
func newTestCore() (*repl.Core, *stateStore) {
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})}
	return repl.New(localID, testRunID(localID), handleFor(localID), st, repl.Config{}), st
}

// newLimitedTestCore returns the local node's core over a real store whose
// ValueMaxBytes is valueMax — the heterogeneous-limit shape the failed-apply
// regression drives: a replicated record whose Data exceeds the local limit
// is rejected by Merge with ErrValueTooLarge — with Config.Log appending to
// logs, the newDiscardCore observation seam.
func newLimitedTestCore(valueMax int, logs *[]string) (*repl.Core, *stateStore) {
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{ValueMaxBytes: valueMax})}
	cfg := repl.Config{Log: func(msg string) { *logs = append(*logs, msg) }}
	return repl.New(localID, testRunID(localID), handleFor(localID), st, cfg), st
}

// helloFrom builds a decoded Hello datagram from peerID arriving from its
// handle (REPL-01 discovery).
func helloFrom(peerID uint16) port.InboundDatagram {
	return datagramFrom(peerID, port.Hello{})
}

// ackFrom builds a decoded Ack datagram from peerID for (key, ver).
func ackFrom(peerID uint16, key string, ver version.RecordVersion) port.InboundDatagram {
	return datagramFrom(peerID, port.Ack{Key: key, Version: ver})
}

// replFrom builds a decoded DataReplication datagram from peerID carrying
// rec under key.
func replFrom(peerID uint16, key string, rec storage.Record) port.InboundDatagram {
	return datagramFrom(peerID, port.DataReplication{Key: key, Record: rec})
}

// datagramFrom wraps one decoded message in an inbound datagram from peerID
// — hand-built port input, no adapter needed (D-01: the structs cross the
// seam, the codec never enters the core's tests).
func datagramFrom(peerID uint16, msg port.Message) port.InboundDatagram {
	return port.InboundDatagram{
		Source:       handleFor(peerID),
		Status:       port.Decoded,
		ProtoVersion: repl.SupportedProtoVersion,
		NodeID:       peerID,
		RunID:        testRunID(peerID),
		Msg:          msg,
	}
}

// localWrite performs the D-05 author sequence — Set, Get,
// NotifyLocalWrite — and returns the freshly stamped version, draining the
// write's own broadcast.
func localWrite(t *testing.T, core *repl.Core, st *stateStore, now time.Time, key, data string) version.RecordVersion {
	t.Helper()
	ver, _, err := st.store.Set(key, []byte(data))
	if err != nil {
		t.Fatalf("store Set(%q): %v", key, err)
	}
	rec, err := st.store.Get(key)
	if err != nil {
		t.Fatalf("store Get(%q): %v", key, err)
	}
	var out port.Outbox
	core.NotifyLocalWrite(now, key, rec, &out)
	out.Drain()
	return ver
}

// remoteRecord builds an externally versioned record; the store recomputes
// the checksum on install, so the wire checksum stays zero (the Merge
// never trusts it).
func remoteRecord(l int64, counter uint16, nodeID uint16, data string) storage.Record {
	return storage.Record{Version: version.PackVersion(l, counter, nodeID), Data: []byte(data)}
}

// assertPending fails unless snap holds an entry under (key, ver) whose
// WaitingOn equals want exactly.
func assertPending(t *testing.T, snap map[string]repl.PendingInfo, key string, ver version.RecordVersion, want []uint16) {
	t.Helper()
	info, ok := snap[key]
	if !ok {
		t.Fatalf("PendingSnapshot holds no entry for key %q, want one waiting on %v", key, want)
	}
	if info.Version != ver {
		t.Fatalf("PendingSnapshot[%q].Version = %x, want the tracked version %x", key, info.Version[:], ver[:])
	}
	if !slices.Equal(info.WaitingOn, want) {
		t.Fatalf("PendingSnapshot[%q].WaitingOn = %v, want %v", key, info.WaitingOn, want)
	}
}

// pendingVersion returns the exact tracked version of core's pending entry
// under key.
func pendingVersion(t *testing.T, core *repl.Core, key string) version.RecordVersion {
	t.Helper()
	info, ok := core.PendingSnapshot()[key]
	if !ok {
		t.Fatalf("PendingSnapshot holds no entry for key %q", key)
	}
	return info.Version
}

// assertAck fails unless drained is exactly one unicast Ack to handle for
// (key, ver) — the applyRemote confirmation contract: an Ack only ever
// follows a successful in-memory apply.
func assertAck(t *testing.T, drained []port.Outbound, handle, key string, ver version.RecordVersion) {
	t.Helper()
	if len(drained) != 1 || drained[0].Broadcast || drained[0].To != handle {
		t.Fatalf("applyRemote drained %d outbound messages, want exactly 1 unicast Ack to %q: %+v", len(drained), handle, drained)
	}
	ack, ok := drained[0].Msg.(port.Ack)
	if !ok {
		t.Fatalf("applyRemote emitted %T, want port.Ack", drained[0].Msg)
	}
	if ack.Key != key || ack.Version != ver {
		t.Fatalf("applyRemote Ack = (%q, %x), want (%q, %x)", ack.Key, ack.Version[:], key, ver[:])
	}
}

// broadcastRepls returns every broadcast DataReplication in drained, in
// emission order — the ladder tests' emission probe (Hello broadcasts and
// unicast Acks are filtered out).
func broadcastRepls(drained []port.Outbound) []port.DataReplication {
	var repls []port.DataReplication
	for _, ob := range drained {
		if !ob.Broadcast {
			continue
		}
		if msg, ok := ob.Msg.(port.DataReplication); ok {
			repls = append(repls, msg)
		}
	}
	return repls
}

// replKeys returns the keys of msgs in order.
func replKeys(msgs []port.DataReplication) []string {
	keys := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		keys = append(keys, msg.Key)
	}
	return keys
}

// TestRetransmit_backoffLadder proves the retransmission ladder (REPL-02,
// the pinned A5 reading): one known peer, one write at t0 — retries fire at
// exactly the cumulative deadlines 200/600/1400/3000/6200ms (backoff
// doubling 200→400→800→1600→3200ms) and then every 5000ms at the
// ack_timeout_max saturation cap, never between steps, never early; and
// every multi-key emission is sorted by key (probe #9).
func TestRetransmit_backoffLadder(t *testing.T) {
	t.Parallel()
	t.Run("the ladder fires exactly at the cumulative steps, then repeats at the cap", func(t *testing.T) {
		t.Parallel()
		t.Helper()
		now0 := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now0, &out) // startup Hello (plan 04-01)
		out.Drain()
		core.HandleDatagram(now0, helloFrom(peerID1), &out)
		out.Drain()
		ver := localWrite(t, core, st, now0, keyAlpha, "one")

		// Probes interleaved with the ladder steps, in virtual-time order:
		// t0+100ms and t0+5000ms are off-ladder and must emit nothing.
		probes := []struct {
			off      time.Duration
			wantFire bool
		}{
			{off: 100 * time.Millisecond, wantFire: false},
			{off: 200 * time.Millisecond, wantFire: true},
			{off: 600 * time.Millisecond, wantFire: true},
			{off: 1400 * time.Millisecond, wantFire: true},
			{off: 3000 * time.Millisecond, wantFire: true},
			{off: 5000 * time.Millisecond, wantFire: false},
			{off: 6200 * time.Millisecond, wantFire: true},
			{off: 11200 * time.Millisecond, wantFire: true},
		}
		nextHelloOff := time.Second
		for _, probe := range probes {
			off, wantFire := probe.off, probe.wantFire
			// Keep the peer alive across the 11.2-second span: it re-Hellos
			// every hello_interval, the wire behavior whose liveness the
			// peer-timeout scan (plan 04-02 Task 3) now enforces.
			for nextHelloOff < off {
				core.HandleDatagram(at(nextHelloOff), helloFrom(peerID1), &out)
				out.Drain()
				nextHelloOff += time.Second
			}
			next := core.Advance(at(off), &out)
			if !next.After(at(off)) {
				t.Fatalf("Advance at t0+%v returned deadline %v, want strictly future", off, next)
			}
			repls := broadcastRepls(out.Drain())
			if !wantFire {
				if len(repls) != 0 {
					t.Fatalf("off-ladder probe at t0+%v emitted DataReplication for %v, want none", off, replKeys(repls))
				}
				continue
			}
			if len(repls) != 1 || repls[0].Key != keyAlpha || repls[0].Record.Version != ver {
				t.Fatalf("ladder step at t0+%v emitted %v, want exactly one DataReplication for (%s, %x)",
					off, repls, keyAlpha, ver[:])
			}
		}
		// Two consecutive cap steps fired (6200ms and 11200ms); the ladder
		// keeps saturating — the next retry lands one ack_timeout_max later.
		wantNext := at(11200 * time.Millisecond).Add(ackTimeoutMax)
		if got := core.PendingSnapshot()[keyAlpha].NextRetry; !got.Equal(wantNext) {
			t.Fatalf("post-cap NextRetry = %v, want the next cap step %v (retries continue until Ack)", got, wantNext)
		}
	})
	t.Run("multi-key emissions are sorted by key at every fire point", func(t *testing.T) {
		t.Parallel()
		t.Helper()
		now0 := at(0)
		core, st := newTestCore()
		var out port.Outbox
		core.Advance(now0, &out)
		out.Drain()
		core.HandleDatagram(now0, helloFrom(peerID1), &out)
		out.Drain()
		localWrite(t, core, st, now0, keyGamma, "g")
		localWrite(t, core, st, now0, keyAlpha, "a")
		localWrite(t, core, st, now0, keyBeta, "b")

		core.Advance(at(200*time.Millisecond), &out)
		if got := replKeys(broadcastRepls(out.Drain())); !slices.Equal(got, []string{keyAlpha, keyBeta, keyGamma}) {
			t.Fatalf("first ladder step emitted keys %v, want ascending [cfg/alpha cfg/beta cfg/gamma]", got)
		}

		// keyBeta Acks between the steps; the next fire point emits only the
		// not-yet-drained keys, still ascending.
		core.HandleDatagram(at(250*time.Millisecond), ackFrom(peerID1, keyBeta, pendingVersion(t, core, keyBeta)), &out)
		out.Drain()
		core.Advance(at(600*time.Millisecond), &out)
		if got := replKeys(broadcastRepls(out.Drain())); !slices.Equal(got, []string{keyAlpha, keyGamma}) {
			t.Fatalf("second ladder step emitted keys %v, want ascending [cfg/alpha cfg/gamma]", got)
		}
	})
}

// TestRetransmit_suspendAndFlush proves the suspension and flush-on-discovery
// semantics (REPL-02, the pinned A6 reading): with no known peer the
// retransmission scanner fires nothing and advances no timer across a long
// virtual window — the entry and its ladder state are retained, not dropped
// — and the first Hello that ends the empty table flushes every pending
// entry in that same event, adds the peer to the wait-set and restarts the
// ladder at ack_timeout.
func TestRetransmit_suspendAndFlush(t *testing.T) {
	t.Parallel()
	t.Helper()
	now0 := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now0, &out)
	out.Drain()
	ver := localWrite(t, core, st, now0, keyAlpha, "one")

	snap := core.PendingSnapshot()
	assertPending(t, snap, keyAlpha, ver, nil)
	if got := snap[keyAlpha].NextRetry; !got.Equal(now0.Add(ackTimeout)) {
		t.Fatalf("entry NextRetry = %v, want the first ladder step %v", got, now0.Add(ackTimeout))
	}

	// 60 virtual seconds of empty-table Advance: zero DataReplication
	// emissions, the entry retained with its ladder state unchanged.
	for off := time.Second; off <= suspendWindow; off += time.Second {
		next := core.Advance(at(off), &out)
		if !next.After(at(off)) {
			t.Fatalf("Advance at t0+%v returned deadline %v, want strictly future", off, next)
		}
		if repls := broadcastRepls(out.Drain()); len(repls) != 0 {
			t.Fatalf("suspended scanner emitted DataReplication for %v at t0+%v, want none", replKeys(repls), off)
		}
	}
	snap = core.PendingSnapshot()
	assertPending(t, snap, keyAlpha, ver, nil)
	if got := snap[keyAlpha].NextRetry; !got.Equal(now0.Add(ackTimeout)) {
		t.Fatalf("suspension moved NextRetry to %v, want it retained at %v", got, now0.Add(ackTimeout))
	}

	// The first Hello ends the empty table: the same HandleDatagram call
	// broadcasts the pending record, gains the peer as its only waiter and
	// restarts the ladder at ack_timeout.
	helloAt := at(suspendWindow + time.Second)
	core.HandleDatagram(helloAt, helloFrom(peerID1), &out)
	repls := broadcastRepls(out.Drain())
	if len(repls) != 1 || repls[0].Key != keyAlpha || repls[0].Record.Version != ver {
		t.Fatalf("discovery flush emitted %v, want exactly one DataReplication for (%s, %x)", repls, keyAlpha, ver[:])
	}
	snap = core.PendingSnapshot()
	assertPending(t, snap, keyAlpha, ver, []uint16{peerID1})
	if got := snap[keyAlpha].NextRetry; !got.Equal(helloAt.Add(ackTimeout)) {
		t.Fatalf("post-flush NextRetry = %v, want the restarted ladder step %v", got, helloAt.Add(ackTimeout))
	}
}

// TestRetransmit_ackDrainSemantics proves the pending-set drain semantics
// (REPL-02, probes #2 and #10): an Ack drains exactly its own waiter on an
// exact (key, version) match, the entry survives until its last waiter Acks
// or a strictly greater version supersedes it, and every other Ack — stale,
// stranger, out-of-order for another key — is a silent no-op.
func TestRetransmit_ackDrainSemantics(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"an Ack drains one waiter; the entry survives until its last waiter Acks", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			out.Drain()

			ver := localWrite(t, core, st, now, keyAlpha, "one")
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})
			if got := core.UnackedStats(now); got != (repl.UnackedStats{Count: 1}) {
				t.Fatalf("UnackedStats() = %+v, want count 1 age 0", got)
			}

			core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, ver), &out)
			out.Drain()
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID2})
			if got := core.UnackedStats(at(time.Second)); got != (repl.UnackedStats{Count: 1, OldestAge: time.Second}) {
				t.Fatalf("UnackedStats() = %+v, want count 1 oldest age 1s", got)
			}

			core.HandleDatagram(now, ackFrom(peerID2, keyAlpha, ver), &out)
			out.Drain()
			if _, ok := core.PendingSnapshot()[keyAlpha]; ok {
				t.Fatalf("pending entry for key %q retained after an Ack from every waited-on peer", keyAlpha)
			}
			if got := core.UnackedStats(now); got != (repl.UnackedStats{}) {
				t.Fatalf("UnackedStats() = %+v, want zero after the drain", got)
			}
		}},
		{"a stale Ack for a superseded version never drains the newer entry", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			out.Drain()

			v1 := localWrite(t, core, st, now, keyAlpha, "one")
			v2 := localWrite(t, core, st, now, keyAlpha, "two")
			if v2.Compare(v1) <= 0 {
				t.Fatalf("second local write stamped %x, want strictly greater than the first %x", v2[:], v1[:])
			}

			core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, v1), &out)
			out.Drain()
			assertPending(t, core.PendingSnapshot(), keyAlpha, v2, []uint16{peerID1, peerID2})
		}},
		{"out-of-order Acks drain their own entries independently", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			out.Drain()

			type write struct {
				key string
				ver version.RecordVersion
			}
			keys := []string{keyAlpha, keyBeta, keyGamma}
			written := make([]write, 0, len(keys))
			for _, key := range keys {
				written = append(written, write{key: key, ver: localWrite(t, core, st, now, key, "v")})
			}
			versionOf := func(key string) version.RecordVersion {
				t.Helper()
				for _, w := range written {
					if w.key == key {
						return w.ver
					}
				}
				t.Fatalf("no write recorded for key %q", key)
				return version.RecordVersion{}
			}

			remaining := make(map[string]version.RecordVersion, len(written))
			for _, w := range written {
				remaining[w.key] = w.ver
			}
			for _, key := range []string{keyBeta, keyGamma, keyAlpha} { // Acks arrive B, C, A (probe #10).
				core.HandleDatagram(now, ackFrom(peerID1, key, versionOf(key)), &out)
				out.Drain()
				delete(remaining, key)
				snap := core.PendingSnapshot()
				if len(snap) != len(remaining) {
					t.Fatalf("after Ack of key %q PendingSnapshot holds %d entries, want %d: %v",
						key, len(snap), len(remaining), snap)
				}
				for wantKey, wantVer := range remaining {
					assertPending(t, snap, wantKey, wantVer, []uint16{peerID1})
				}
			}
		}},
		{"an Ack from a stranger is a silent no-op", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			out.Drain()

			ver := localWrite(t, core, st, now, keyAlpha, "one")
			core.HandleDatagram(now, ackFrom(strangerID, keyAlpha, ver), &out)
			out.Drain()
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})
			if got := len(core.PeerSnapshot()); got != 2 {
				t.Fatalf("the Ack path touched the peer table: PeerSnapshot holds %d peers, want 2", got)
			}
		}},
		{"remote supersession drops the entry; an equal redelivery does not", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			out.Drain()

			ver := localWrite(t, core, st, now, keyAlpha, "one")
			served, err := st.store.Get(keyAlpha)
			if err != nil {
				t.Fatalf("store Get(%q): %v", keyAlpha, err)
			}

			core.HandleDatagram(now, replFrom(peerID1, keyAlpha, served), &out)
			out.Drain()
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})

			greater := remoteRecord(testNowMs+1, 0, peerID1, "remote wins")
			if greater.Version.Compare(ver) <= 0 {
				t.Fatalf("test remote version %x not strictly greater than the tracked %x", greater.Version[:], ver[:])
			}
			core.HandleDatagram(now, replFrom(peerID1, keyAlpha, greater), &out)
			out.Drain()
			if _, ok := core.PendingSnapshot()[keyAlpha]; ok {
				t.Fatalf("pending entry for key %q retained after a strictly greater remote version", keyAlpha)
			}
			got, err := st.store.Get(keyAlpha)
			if err != nil || got.Version != greater.Version {
				t.Fatalf("store Get(%q) after supersession = (%x, %v), want the remote version %x", keyAlpha, got.Version[:], err, greater.Version[:])
			}
		}},
		{"a store-rejected apply is never Acked and never supersedes tracking — retransmission continues", func(t *testing.T) {
			t.Helper()
			now := at(0)
			var logs []string
			core, st := newLimitedTestCore(8, &logs)
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			out.Drain()

			ver := localWrite(t, core, st, now, keyAlpha, "one")
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})

			// A strictly greater remote version whose oversized Data the
			// store rejects with ErrValueTooLarge — the heterogeneous-limit
			// shape: the version would win the merge, but the store refuses
			// the record outright. It arrives from the second waited-on
			// peer, proving the rejection holds regardless of sender.
			rejected := remoteRecord(testNowMs+1, 0, peerID2, "remote payload over the local limit")
			if rejected.Version.Compare(ver) <= 0 {
				t.Fatalf("test remote version %x not strictly greater than the tracked %x", rejected.Version[:], ver[:])
			}
			core.HandleDatagram(now, replFrom(peerID2, keyAlpha, rejected), &out)

			if got := len(out.Drain()); got != 0 {
				t.Fatalf("a store-rejected apply emitted %d messages, want 0 — a refused record is never confirmed", got)
			}
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})
			served, err := st.store.Get(keyAlpha)
			if err != nil || served.Version != ver {
				t.Fatalf("store Get(%q) after the rejected apply = (%x, %v), want the local write (%x)", keyAlpha, served.Version[:], err, ver[:])
			}
			if len(logs) == 0 {
				t.Fatal("the failed apply surfaced nothing through Config.Log")
			}

			// Retransmission continues: the ladder still fires for the
			// tracked record at its armed retry time.
			core.Advance(now.Add(ackTimeout), &out)
			repls := broadcastRepls(out.Drain())
			if len(repls) != 1 || repls[0].Key != keyAlpha || repls[0].Record.Version != ver {
				t.Fatalf("post-rejection retransmit emitted %v, want exactly one DataReplication for (%s, %x)", repls, keyAlpha, ver[:])
			}
		}},
		{"an error-free duplicate redelivery of an already-stored record still Acks", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			out.Drain()

			ver := localWrite(t, core, st, now, keyAlpha, "one")
			served, err := st.store.Get(keyAlpha)
			if err != nil {
				t.Fatalf("store Get(%q): %v", keyAlpha, err)
			}

			// The peer's echo of our own write: an equal-version no-op merge
			// that still Acks, never superseding the tracked entry.
			core.HandleDatagram(now, replFrom(peerID1, keyAlpha, served), &out)
			assertAck(t, out.Drain(), handleFor(peerID1), keyAlpha, ver)
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})

			// The duplicate redelivery of the same datagram: the merge is
			// again a no-op, and the Ack is emitted again.
			core.HandleDatagram(now, replFrom(peerID1, keyAlpha, served), &out)
			assertAck(t, out.Drain(), handleFor(peerID1), keyAlpha, ver)
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2})
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			t.Parallel()
			tt.run(t)
		})
	}
}
