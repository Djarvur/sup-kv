package repl_test

import (
	"bytes"
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
// seam — by delegating the core's four methods (the production applier is the
// persist engine's ApplyRemote, plan 04-06). Records stays as a test-only
// convenience for expected-value construction; the core interface no longer
// declares it.
type stateStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s stateStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Version returns the stored version for key, tombstones included.
func (s stateStore) Version(key string) (version.RecordVersion, bool) { return s.store.Version(key) }

// Lookup returns the stored record for key, tombstones included.
func (s stateStore) Lookup(key string) (storage.Record, bool) { return s.store.Lookup(key) }

// Versions returns the shallow key→version view, tombstones included.
func (s stateStore) Versions() map[string]version.RecordVersion { return s.store.Versions() }

// Observe absorbs one received version into the store's clock.
func (s stateStore) Observe(ver version.RecordVersion) time.Duration { return s.store.Observe(ver) }

// Generation returns the store's content generation.
func (s stateStore) Generation() uint64 { return s.store.Generation() }

// Records returns the full snapshot, tombstones included — test expectations only.
func (s stateStore) Records() map[string]storage.Record { return s.store.Records() }

// newTestCore returns the local node's core over a real store whose clock
// parks at the fixed test millisecond (the D-02 seam), carrying the default
// replication Config (docs/IDEA.md §Parameters: ack_timeout 200ms,
// ack_timeout_max 5s, retransmit_interval 100ms, peer_timeout 5s,
// hello_interval 1s) — the exact timings the ladder and eviction tests
// assert against. Peers are always synthetic datagram sources, never cores.
func newTestCore() (*repl.Core, *stateStore) {
	return newTestCoreWithConfig(repl.Config{})
}

// newTestCoreWithConfig is newTestCore over an explicit Config: the seam for
// tests whose own scenario legitimately runs away from a default — the
// far-ahead drift scenario's widened MaxClockLead — while the production
// default stays untouched everywhere else.
func newTestCoreWithConfig(cfg repl.Config) (*repl.Core, *stateStore) {
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})}
	return repl.New(localID, testRunID(localID), st, cfg), st
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
	return repl.New(localID, testRunID(localID), st, cfg), st
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
// rec under key. The record checksum is recomputed through the production
// domain — what a correct sender's store already stamped — so the fixture
// is wire-valid and the validation pipeline's checksum step passes; a test
// that wants the bad_checksum class builds its own mutated record.
func replFrom(peerID uint16, key string, rec storage.Record) port.InboundDatagram {
	rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
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
func remoteRecord(l int64, nodeID uint16, data string) storage.Record {
	return storage.Record{Version: version.PackVersion(l, 0, nodeID), Data: []byte(data)}
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

// TestPending_gauges pins the O(1) pending gauges (XPORT-02(1), RESEARCH
// Pitfalls 7 and 8): an empty pending set reports a zero count and age, the
// oldest age follows the earliest createdAt even when writes are notified
// out of time order, draining the oldest entry advances the age to the next
// creation time, and PeerCount is exactly the peer-table length. The accessor
// reads only the map length and the front of the createdAt-ordered list — no
// walk of the pending set — so the empty case here is the zero-cost pin.
func TestPending_gauges(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out) // startup Hello + empty listing
	out.Drain()

	if got := core.UnackedStats(now); got != (repl.UnackedStats{}) {
		t.Fatalf("UnackedStats() on the empty pending set = %+v, want the zero gauge", got)
	}
	if got := core.PeerCount(); got != 0 {
		t.Fatalf("PeerCount() on the empty peer table = %d, want 0", got)
	}

	core.HandleDatagram(now, helloFrom(peerID1), &out)
	core.HandleDatagram(now, helloFrom(peerID2), &out)
	out.Drain()
	if got := core.PeerCount(); got != 2 {
		t.Fatalf("PeerCount() after two Hellos = %d, want the peer-table length 2", got)
	}

	// Out-of-order creation times: beta first at t0+2s, then alpha at t0 and
	// gamma at t0+1s. The oldest age must follow the earliest createdAt, not
	// the insertion order.
	localWrite(t, core, st, at(2*time.Second), keyBeta, "b")
	localWrite(t, core, st, at(0), keyAlpha, "a")
	localWrite(t, core, st, at(time.Second), keyGamma, "g")
	if got := core.UnackedStats(at(3 * time.Second)); got != (repl.UnackedStats{Count: 3, OldestAge: 3 * time.Second}) {
		t.Fatalf("UnackedStats() after out-of-order insertions = %+v, want count 3 oldest age 3s (the alpha createdAt)", got)
	}

	// Drain the oldest entry: both waiters Ack alpha, so the age steps to
	// gamma's createdAt.
	alphaVer := pendingVersion(t, core, keyAlpha)
	core.HandleDatagram(at(3*time.Second), ackFrom(peerID1, keyAlpha, alphaVer), &out)
	core.HandleDatagram(at(3*time.Second), ackFrom(peerID2, keyAlpha, alphaVer), &out)
	out.Drain()
	if got := core.UnackedStats(at(3 * time.Second)); got != (repl.UnackedStats{Count: 2, OldestAge: 2 * time.Second}) {
		t.Fatalf("UnackedStats() after draining the oldest entry = %+v, want count 2 oldest age 2s (the gamma createdAt)", got)
	}
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
		first := replKeys(broadcastRepls(advanceDrain(core, &out, at(200*time.Millisecond))))
		if !slices.Equal(first, []string{keyAlpha, keyBeta, keyGamma}) {
			t.Fatalf("first ladder step emitted keys %v, want ascending [cfg/alpha cfg/beta cfg/gamma]", first)
		}

		// keyBeta Acks between the steps; the next fire point emits only the
		// not-yet-drained keys, still ascending. Each entry's second step is
		// due at its own emission time plus the doubled backoff, so the test
		// drives the paced window rather than one exact instant.
		core.HandleDatagram(at(250*time.Millisecond), ackFrom(peerID1, keyBeta, pendingVersion(t, core, keyBeta)), &out)
		out.Drain()
		var secondStep []string
		for off := 600 * time.Millisecond; off <= 620*time.Millisecond; off += listingDrainStep {
			core.Advance(at(off), &out)
			secondStep = append(secondStep, replKeys(broadcastRepls(out.Drain()))...)
		}
		if !slices.Equal(secondStep, []string{keyAlpha, keyGamma}) {
			t.Fatalf("second ladder step emitted keys %v, want ascending [cfg/alpha cfg/gamma]", secondStep)
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
	// enqueues the pending record on the paced queue (it emits no datagram
	// itself), gains the peer as its only waiter and restarts the ladder at
	// ack_timeout; the next Advance drains the queued send.
	helloAt := at(suspendWindow + time.Second)
	core.HandleDatagram(helloAt, helloFrom(peerID1), &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("the discovery flush emitted %d messages inside the Hello handler, want 0 — it only enqueues", got)
	}
	core.Advance(helloAt, &out)
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
// stranger, out-of-order for another key — is a silent no-op. The Acked
// rows pin the Phase 5 honesty seam (T-05-04): an Acked peer reads from
// Acked and vanishes from WaitingOn, a peer_timeout eviction shrinks
// WaitingOn while Acked stays empty, and a re-Ack is idempotent.
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

			greater := remoteRecord(testNowMs+1, peerID1, "remote wins")
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
			rejected := remoteRecord(testNowMs+1, peerID2, "remote payload over the local limit")
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
		{"an Ack moves the peer from WaitingOn to Acked, sorted by node id", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			core.HandleDatagram(now, helloFrom(peerID3), &out)
			out.Drain()

			ver := localWrite(t, core, st, now, keyAlpha, "one")
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1, peerID2, peerID3})

			// Acks arrive scrambled — the highest node id first — but the
			// reported Acked set stays sorted ascending (the WaitingOn
			// discipline): with the entry surviving on its last waiter, the
			// Acking peers read from Acked and nobody reads from WaitingOn.
			core.HandleDatagram(now, ackFrom(peerID3, keyAlpha, ver), &out)
			out.Drain()
			core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, ver), &out)
			out.Drain()

			info, ok := core.PendingSnapshot()[keyAlpha]
			if !ok {
				t.Fatalf("pending entry for key %q drained before its last waiter Acked", keyAlpha)
			}
			if !slices.Equal(info.Acked, []uint16{peerID1, peerID3}) {
				t.Fatalf("PendingSnapshot[%q].Acked = %v, want sorted [%d %d] despite the scrambled arrival order",
					keyAlpha, info.Acked, peerID1, peerID3)
			}
			if !slices.Equal(info.WaitingOn, []uint16{peerID2}) {
				t.Fatalf("PendingSnapshot[%q].WaitingOn = %v, want [%d]", keyAlpha, info.WaitingOn, peerID2)
			}
		}},
		{"peer_timeout eviction shrinks WaitingOn while Acked stays empty", func(t *testing.T) {
			t.Helper()
			now0 := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.Advance(now0, &out) // startup Hello + listing; arms the timers.
			out.Drain()
			core.HandleDatagram(now0, helloFrom(peerID1), &out)
			out.Drain()
			ver := localWrite(t, core, st, now0, keyAlpha, "one")
			assertPending(t, core.PendingSnapshot(), keyAlpha, ver, []uint16{peerID1})

			// The peer goes silent; the first Advance at or after
			// lastHello+peer_timeout evicts it from the table and the
			// entry's wait-set in that same event (REPL-06, the at-or-after
			// boundary). The entry survives in suspension — no Ack ever
			// arrived, so Acked must stay empty: eviction is never an Ack
			// (T-05-04, the wait_for_replication honesty regression lock).
			core.Advance(at(peerTimeout), &out)
			out.Drain()

			info, ok := core.PendingSnapshot()[keyAlpha]
			if !ok {
				t.Fatal("eviction drained the pending entry, want it retained in suspension (an empty table retains, never drains)")
			}
			if len(info.WaitingOn) != 0 {
				t.Fatalf("post-eviction WaitingOn = %v, want empty (the peer left by expiry)", info.WaitingOn)
			}
			if len(info.Acked) != 0 {
				t.Fatalf("post-eviction Acked = %v, want empty — an eviction must never read as an Ack", info.Acked)
			}
		}},
		{"a duplicate Ack adds no second Acked entry", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.HandleDatagram(now, helloFrom(peerID1), &out)
			core.HandleDatagram(now, helloFrom(peerID2), &out)
			out.Drain()

			ver := localWrite(t, core, st, now, keyAlpha, "one")
			core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, ver), &out)
			out.Drain()
			// The retransmitted record reaches the peer again and it
			// re-Acks (the always-Ack-on-duplicate truth of plan 04-01):
			// the second Ack is idempotent — the wait-set stays drained of
			// the peer and Acked gains no second entry.
			core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, ver), &out)
			out.Drain()

			info, ok := core.PendingSnapshot()[keyAlpha]
			if !ok {
				t.Fatalf("pending entry for key %q drained before its last waiter Acked", keyAlpha)
			}
			if !slices.Equal(info.Acked, []uint16{peerID1}) {
				t.Fatalf("PendingSnapshot[%q].Acked = %v after a duplicate Ack, want exactly [%d]", keyAlpha, info.Acked, peerID1)
			}
			if !slices.Equal(info.WaitingOn, []uint16{peerID2}) {
				t.Fatalf("PendingSnapshot[%q].WaitingOn = %v, want [%d]", keyAlpha, info.WaitingOn, peerID2)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			t.Parallel()
			tt.run(t)
		})
	}
}

// TestRetransmit_heapInvariant pins the indexed due-heap (XPORT-02(1),
// RESEARCH "Don't Hand-Roll"): after a scripted insert/ack/supersede/evict/
// re-arm sequence, the heap's earliest deadline agrees with the minimum
// NextRetry over a walk of PendingSnapshot — no deletion or re-arm leaves a
// stale item that could fire early — and entries with equal deadlines emit
// deterministically in their insertion order. Silence one nanosecond before
// the minimum and the exactly-at-minimum emission are the observable proxy
// for the heap root; advancing past it without an emission proves a stale
// item did not survive.
func TestRetransmit_heapInvariant(t *testing.T) {
	t.Parallel()
	cfg := repl.Config{
		HelloInterval: 10 * time.Second,
		SyncInterval:  10 * time.Second,
		PeerTimeout:   2 * time.Second,
	}
	core, st := newSyncCore(cfg)
	var out port.Outbox
	core.Advance(at(0), &out) // startup Hello + empty listing
	out.Drain()
	core.HandleDatagram(at(0), helloFrom(peerID1), &out)
	core.HandleDatagram(at(0), helloFrom(peerID2), &out)
	out.Drain()

	// Equal-deadline insertions in ascending key order: emission order must
	// be the insertion order (which is also the standing sorted-key pin). The
	// pacer emits one datagram per interval, so the three first-step sends
	// drain across successive Advances; each entry's next ladder step is armed
	// from ITS emission time (never the common enqueue tick).
	localWrite(t, core, st, at(0), keyAlpha, "a")
	localWrite(t, core, st, at(0), keyBeta, "b")
	localWrite(t, core, st, at(0), keyGamma, "g")
	core.Advance(at(200*time.Millisecond), &out)
	var firstStep []string
	emittedAt := make(map[string]time.Time, 3)
	for _, msg := range broadcastRepls(out.Drain()) {
		firstStep = append(firstStep, msg.Key)
		emittedAt[msg.Key] = at(200 * time.Millisecond)
	}
	for off := 200*time.Millisecond + listingDrainStep; len(firstStep) < 3; off += listingDrainStep {
		core.Advance(at(off), &out)
		for _, msg := range broadcastRepls(out.Drain()) {
			firstStep = append(firstStep, msg.Key)
			emittedAt[msg.Key] = at(off)
		}
	}
	if !slices.Equal(firstStep, []string{keyAlpha, keyBeta, keyGamma}) {
		t.Fatalf("equal-deadline emission order = %v, want the insertion order [cfg/alpha cfg/beta cfg/gamma]", firstStep)
	}
	for _, key := range firstStep {
		wantNext := emittedAt[key].Add(2 * ackTimeout)
		if got := core.PendingSnapshot()[key].NextRetry; !got.Equal(wantNext) {
			t.Fatalf("after the first ladder step %q NextRetry = %v, want emission+backoff %v", key, got.Sub(at(0)), wantNext.Sub(at(0)))
		}
	}

	// Ack beta away (both waiters), supersede alpha with a newer local write,
	// and supersede gamma with a strictly greater remote version.
	betaVer := pendingVersion(t, core, keyBeta)
	core.HandleDatagram(at(250*time.Millisecond), ackFrom(peerID1, keyBeta, betaVer), &out)
	core.HandleDatagram(at(250*time.Millisecond), ackFrom(peerID2, keyBeta, betaVer), &out)
	out.Drain()
	localWrite(t, core, st, at(300*time.Millisecond), keyAlpha, "a2")
	greaterGamma := remoteRecord(testNowMs+1, peerID1, "remote gamma")
	if greaterGamma.Version.Compare(pendingVersion(t, core, keyGamma)) <= 0 {
		t.Fatalf("test remote version %x not strictly greater than the tracked gamma version", greaterGamma.Version[:])
	}
	core.HandleDatagram(at(400*time.Millisecond), replFrom(peerID1, keyGamma, greaterGamma), &out)
	out.Drain()
	if snap := core.PendingSnapshot(); len(snap) != 1 {
		t.Fatalf("after the Ack and both supersessions PendingSnapshot holds %d entries, want only the replacement alpha: %v", len(snap), snap)
	}

	// The replacement alpha is due at 500ms; the next ladder steps land at
	// 900ms and 1700ms. Silence one nanosecond before each proves the heap
	// root (and only it) is the next retry.
	assertRetryBeat := func(off time.Duration, want []string, wantNext time.Duration) {
		t.Helper()
		core.Advance(at(off-time.Nanosecond), &out)
		if repls := broadcastRepls(out.Drain()); len(repls) != 0 {
			t.Fatalf("Advance one nanosecond before t0+%v emitted %v, want silence — a stale heap item fired early", off, replKeys(repls))
		}
		core.Advance(at(off), &out)
		if got := replKeys(broadcastRepls(out.Drain())); !slices.Equal(got, want) {
			t.Fatalf("Advance at t0+%v emitted %v, want %v", off, got, want)
		}
		if got := core.PendingSnapshot()[keyAlpha].NextRetry; !got.Equal(at(wantNext)) {
			t.Fatalf("after the t0+%v emission NextRetry = %v, want the re-armed %v", off, got, at(wantNext))
		}
	}
	assertRetryBeat(500*time.Millisecond, []string{keyAlpha}, 900*time.Millisecond)
	assertRetryBeat(900*time.Millisecond, []string{keyAlpha}, 1700*time.Millisecond)

	// Both peers expire at 2s (last Hello at 0 + peer_timeout, the at-or-after
	// boundary): eviction precedes the scan, so the overdue 1700ms item fires
	// nothing and the entry stays tracked with its ladder paused.
	core.Advance(at(2*time.Second), &out)
	if repls := broadcastRepls(out.Drain()); len(repls) != 0 {
		t.Fatalf("the eviction Advance emitted %v, want none — an empty peer table suspends the scanner", replKeys(repls))
	}
	if got := core.PeerCount(); got != 0 {
		t.Fatalf("PeerCount() after the eviction = %d, want 0", got)
	}
	if got := core.PendingSnapshot()[keyAlpha].NextRetry; !got.Equal(at(1700 * time.Millisecond)) {
		t.Fatalf("suspension moved NextRetry to %v, want it retained at 1700ms", got)
	}

	// The returning Hello is discovery: it enqueues the pending entry, re-adds
	// the peer and restarts the ladder at 2.1s+200ms; the same-tick Advance
	// drains the queued send.
	core.HandleDatagram(at(2100*time.Millisecond), helloFrom(peerID1), &out)
	core.Advance(at(2100*time.Millisecond), &out)
	if got := replKeys(broadcastRepls(out.Drain())); !slices.Equal(got, []string{keyAlpha}) {
		t.Fatalf("the discovery flush emitted %v, want exactly the pending alpha", got)
	}
	assertRetryBeat(2300*time.Millisecond, []string{keyAlpha}, 2700*time.Millisecond)
	assertRetryBeat(2700*time.Millisecond, []string{keyAlpha}, 3500*time.Millisecond)
}

// TestOutboxResults_drainResets pins the result stream's two contracts
// (XPORT-12): DrainResults returns every result in emission order and resets
// the stream, and the message stream is independent — draining one never
// touches the other. An undrained result stream is the growth trap the
// production Run loop's per-event drain exists to prevent.
func TestOutboxResults_drainResets(t *testing.T) {
	t.Parallel()

	ver := version.PackVersion(testNowMs, 0, localID)
	var out port.Outbox
	out.Broadcast(port.Hello{})
	out.Ack("k", ver, 7)
	out.Superseded("k", ver)
	out.Drained("k", ver)
	out.WaitSetEmpty("k", ver)

	got := out.DrainResults()
	if len(got) != 4 {
		t.Fatalf("DrainResults returned %d results, want 4", len(got))
	}
	wantKinds := []port.ResultKind{port.ResultAck, port.ResultSuperseded, port.ResultDrained, port.ResultWaitSetEmpty}
	for i, want := range wantKinds {
		if got[i].Kind != want {
			t.Fatalf("result %d kind = %v, want %v (emission order preserved)", i, got[i].Kind, want)
		}
		if got[i].Key != "k" || got[i].Version != ver {
			t.Fatalf("result %d identity = (%q, %x), want (\"k\", %x)", i, got[i].Key, got[i].Version[:], ver[:])
		}
	}
	if got[0].Peer != 7 {
		t.Fatalf("Ack result peer = %d, want 7", got[0].Peer)
	}
	if again := out.DrainResults(); len(again) != 0 {
		t.Fatalf("second DrainResults returned %d results, want the stream reset", len(again))
	}

	// The message stream is independent: DrainResults did not reset it.
	if drained := out.Drain(); len(drained) != 1 {
		t.Fatalf("Drain after DrainResults returned %d messages, want the one broadcast", len(drained))
	}
	out.Ack("other", ver, 1)
	if drained := out.Drain(); len(drained) != 0 {
		t.Fatalf("Drain after Drain returned %d messages, want none", len(drained))
	}
	if left := out.DrainResults(); len(left) != 1 || left[0].Key != "other" {
		t.Fatalf("Drain touched the result stream: DrainResults = %+v, want the one pending result", left)
	}
}

// TestOutboxResults_resolutionEvents drives the four emission sites through
// the core (XPORT-12): a matching Ack (stranger included) emits ResultAck,
// the wait-set-emptying Ack additionally emits ResultDrained with the
// TRACKED version, a strictly greater remote apply emits ResultSuperseded
// with the tracked version, and evicting the last waiter emits
// ResultWaitSetEmpty while the entry stays suspended.
func TestOutboxResults_resolutionEvents(t *testing.T) {
	t.Parallel()

	t.Run("ack_and_drained", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		var out port.Outbox
		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		ver := localWrite(t, core, st, at(0), keyAlpha, "alpha")

		core.HandleDatagram(at(10*time.Millisecond), ackFrom(peerID1, keyAlpha, ver), &out)
		out.Drain()
		results := out.DrainResults()
		if len(results) != 2 {
			t.Fatalf("drain results = %+v, want exactly Ack then Drained", results)
		}
		if results[0].Kind != port.ResultAck || results[0].Peer != peerID1 || results[0].Version != ver {
			t.Fatalf("first result = %+v, want the peer's Ack for the tracked version", results[0])
		}
		if results[1].Kind != port.ResultDrained || results[1].Version != ver {
			t.Fatalf("second result = %+v, want Drained for the tracked version", results[1])
		}
	})

	t.Run("stranger_ack_counted", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		var out port.Outbox
		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		ver := localWrite(t, core, st, at(0), keyAlpha, "alpha")

		// A node outside the wait-set still produces the counting Ack
		// (WR-01); the wait-set remains non-empty, so no Drained follows.
		core.HandleDatagram(at(10*time.Millisecond), ackFrom(strangerID, keyAlpha, ver), &out)
		out.Drain()
		results := out.DrainResults()
		if len(results) != 1 || results[0].Kind != port.ResultAck || results[0].Peer != strangerID {
			t.Fatalf("drain results = %+v, want exactly the stranger's Ack", results)
		}
		if got := core.PendingSnapshot()[keyAlpha].WaitingOn; !slices.Equal(got, []uint16{peerID1}) {
			t.Fatalf("the stranger Ack moved the wait-set to %v, want [%d]", got, peerID1)
		}
	})

	t.Run("remote_superseded", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		var out port.Outbox
		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		ver := localWrite(t, core, st, at(0), keyAlpha, "alpha")

		core.HandleDatagram(at(10*time.Millisecond), replFrom(peerID2, keyAlpha, remoteRecord(testNowMs+1000, peerID2, "newer")), &out)
		out.Drain()
		results := out.DrainResults()
		if len(results) != 1 || results[0].Kind != port.ResultSuperseded || results[0].Version != ver {
			t.Fatalf("drain results = %+v, want Superseded for the tracked version %x", results, ver[:])
		}
		if _, tracked := core.PendingSnapshot()[keyAlpha]; tracked {
			t.Fatal("the superseded entry is still tracked")
		}
	})

	t.Run("local_superseded", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		var out port.Outbox
		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		ver := localWrite(t, core, st, at(0), keyAlpha, "alpha")

		newer := storage.Record{Version: version.PackVersion(testNowMs+1000, 0, localID), Data: []byte("newer")}
		core.NotifyLocalWrite(at(10*time.Millisecond), keyAlpha, newer, &out)
		out.Drain()
		results := out.DrainResults()
		if len(results) != 1 || results[0].Kind != port.ResultSuperseded || results[0].Version != ver {
			t.Fatalf("drain results = %+v, want Superseded for the older tracked version %x", results, ver[:])
		}
	})

	t.Run("eviction_wait_set_empty", func(t *testing.T) {
		t.Parallel()
		core, st := newTestCore()
		var out port.Outbox
		core.HandleDatagram(at(0), helloFrom(peerID1), &out)
		out.Drain()
		ver := localWrite(t, core, st, at(0), keyAlpha, "alpha")

		core.Advance(at(5*time.Second), &out) // peer_timeout eviction at the at-or-after boundary.
		out.Drain()
		results := out.DrainResults()
		if len(results) != 1 || results[0].Kind != port.ResultWaitSetEmpty || results[0].Version != ver {
			t.Fatalf("drain results = %+v, want WaitSetEmpty for the tracked version %x", results, ver[:])
		}
		if _, tracked := core.PendingSnapshot()[keyAlpha]; !tracked {
			t.Fatal("eviction dropped the suspended entry, want it retained")
		}
	})
}

// TestRetransmit_ladderFromEmission pins the emission-time arming contract
// (D-05, RESEARCH Pitfall 4): with a slow 10/s pacer, three entries that
// become due together are emitted one per paced slot, and each entry's next
// retry deadline is its own emission time plus the doubled backoff — never
// the common enqueue tick plus that backoff. Arming at enqueue would make
// every backed-up entry overdue the moment the pacer reached it, restoring
// the R-02 storm.
func TestRetransmit_ladderFromEmission(t *testing.T) {
	t.Parallel()
	now0 := at(0)
	core, st := newSyncCore(repl.Config{SyncResponseRate: 10})
	var out port.Outbox
	core.Advance(now0, &out)
	out.Drain()
	core.HandleDatagram(now0, helloFrom(peerID1), &out)
	out.Drain()
	for _, key := range []string{keyAlpha, keyBeta, keyGamma} {
		localWrite(t, core, st, now0, key, "v")
	}

	core.Advance(now0.Add(ackTimeout), &out)
	emittedAt := make(map[string]time.Time, 3)
	for _, msg := range broadcastRepls(out.Drain()) {
		emittedAt[msg.Key] = now0.Add(ackTimeout)
	}
	now := now0.Add(ackTimeout)
	for len(emittedAt) < 3 {
		now = now.Add(listingDrainStep)
		core.Advance(now, &out)
		for _, msg := range broadcastRepls(out.Drain()) {
			emittedAt[msg.Key] = now
		}
	}

	snap := core.PendingSnapshot()
	enqueueArmed := now0.Add(3 * ackTimeout)
	for _, key := range []string{keyAlpha, keyBeta, keyGamma} {
		emitted, ok := emittedAt[key]
		if !ok {
			t.Fatalf("key %q never emitted its first retry", key)
		}
		want := emitted.Add(2 * ackTimeout)
		if got := snap[key].NextRetry; !got.Equal(want) {
			t.Fatalf("%q NextRetry = emission+%v, want emission+backoff %v (enqueue+backoff would be %v)",
				key, got.Sub(emitted), want.Sub(emitted), enqueueArmed.Sub(now0))
		}
	}
	if snap[keyGamma].NextRetry.Equal(enqueueArmed) {
		t.Fatal("no entry was emitted after the enqueue tick — the paced-arm assertion would be vacuous")
	}
}

// TestRetransmit_pacedStormBound proves the anti-storm bound (D-05): a
// discovery flush of 1 000 pending entries — the whole-set event that
// produced the measured 949 552-retry storm — emits at most one datagram per
// pacer interval, so a fixed virtual window sees at most window/interval + 1
// record sends, never one per entry. The queue is still draining at the
// window's end, so the fixture really exercises the backlog.
func TestRetransmit_pacedStormBound(t *testing.T) {
	t.Parallel()
	const (
		stormPending = 1000
		stormWindow  = time.Second
		stormRate    = 500
	)
	now0 := at(0)
	// One large chunk: the seed Hello's first-seen listing is a single queued
	// chunk (PROTO-04), so the record-send count stays exactly stormPending
	// and the listing never muddies the storm measurement.
	core, st := newTestCoreWithConfig(repl.Config{SyncChunkBytes: 1 << 20})
	var out port.Outbox
	core.Advance(now0, &out)
	out.Drain()
	for i := range stormPending {
		localWrite(t, core, st, now0, fmt.Sprintf("storm/%04d", i), "v")
	}
	helloAt := now0.Add(time.Millisecond)
	core.HandleDatagram(helloAt, helloFrom(peerID1), &out)
	out.Drain()
	if got, want := core.QueuedSends(), stormPending+1; got != want {
		t.Fatalf("the discovery flush plus its first-seen listing queued %d sends, want %d (one record send per pending entry and one listing chunk)",
			got, want)
	}

	interval := time.Second / stormRate
	emitted := 0
	end := helloAt.Add(stormWindow)
	for now := helloAt; !now.After(end); now = now.Add(time.Millisecond) {
		core.Advance(now, &out)
		got := len(broadcastRepls(out.Drain()))
		if got > 1 {
			t.Fatalf("Advance at +%v emitted %d record sends, want at most one per call", now.Sub(now0), got)
		}
		emitted += got
	}
	if maxEmissions := int(stormWindow/interval) + 1; emitted > maxEmissions {
		t.Fatalf("the flush emitted %d record sends in %v, want <= %d (window/interval + 1) — an unpaced storm returned",
			emitted, stormWindow, maxEmissions)
	}
	if core.QueuedSends() == 0 {
		t.Fatalf("the queue emptied within %v — the fixture is too small to prove the bound", stormWindow)
	}
}

// TestRetransmit_skipDrainedAndSuperseded pins the queued-send skip rules
// (RESEARCH Pitfall 5): a send whose entry Acked away after enqueue emits
// nothing, and a send whose stored version changed emits nothing either — the
// retry path removes the untrustworthy entry instead of resurrecting a stale
// record.
func TestRetransmit_skipDrainedAndSuperseded(t *testing.T) {
	t.Parallel()
	t.Run("a queued send whose entry Acked away emits nothing", func(t *testing.T) {
		t.Parallel()
		core, st := newSyncCore(repl.Config{SyncResponseRate: 10})
		var out port.Outbox
		core.Advance(at(0), &out)
		out.Drain()
		localWrite(t, core, st, at(0), keyAlpha, "a")
		localWrite(t, core, st, at(0), keyBeta, "b")
		localWrite(t, core, st, at(0), keyGamma, "g")
		helloAt := at(time.Millisecond)
		core.HandleDatagram(helloAt, helloFrom(peerID1), &out)
		out.Drain()
		core.Advance(helloAt, &out)
		if got := replKeys(broadcastRepls(out.Drain())); !slices.Equal(got, []string{keyAlpha}) {
			t.Fatalf("the flush head emitted %v, want the sorted head alpha", got)
		}

		core.HandleDatagram(helloAt, ackFrom(peerID1, keyBeta, pendingVersion(t, core, keyBeta)), &out)
		out.Drain()
		core.Advance(helloAt.Add(200*time.Millisecond), &out)
		got := replKeys(broadcastRepls(out.Drain()))
		if slices.Contains(got, keyBeta) {
			t.Fatalf("the drained beta was still broadcast: %v", got)
		}
		if !slices.Equal(got, []string{keyGamma}) {
			t.Fatalf("the slot emitted %v, want exactly gamma after beta's skip", got)
		}
	})

	t.Run("a queued send whose stored version changed emits nothing", func(t *testing.T) {
		t.Parallel()
		core, st := newSyncCore(repl.Config{SyncResponseRate: 10})
		var out port.Outbox
		core.Advance(at(0), &out)
		out.Drain()
		localWrite(t, core, st, at(0), keyAlpha, "a")
		localWrite(t, core, st, at(0), keyBeta, "b")
		helloAt := at(time.Millisecond)
		core.HandleDatagram(helloAt, helloFrom(peerID1), &out)
		out.Drain()
		core.Advance(helloAt, &out)
		if got := replKeys(broadcastRepls(out.Drain())); !slices.Equal(got, []string{keyAlpha}) {
			t.Fatalf("the flush head emitted %v, want alpha", got)
		}

		// The store record changes under the queued beta send without a core
		// notification: the queued version is stale.
		_, _, err := st.store.Set(keyBeta, []byte("changed"))
		if err != nil {
			t.Fatalf("store Set(%q): %v", keyBeta, err)
		}
		core.Advance(helloAt.Add(150*time.Millisecond), &out)
		if got := broadcastRepls(out.Drain()); len(got) != 0 {
			t.Fatalf("the stale queued send emitted %v, want nothing", got)
		}
		if _, tracked := core.PendingSnapshot()[keyBeta]; !tracked {
			t.Fatal("the flush skip dropped the pending entry, want it retained until the ladder due time")
		}

		// At beta's ladder due time the scan queues a retry, and the retry
		// path removes the untrustworthy entry instead of resurrecting it.
		core.Advance(helloAt.Add(500*time.Millisecond), &out)
		out.Drain() // alpha's due retry
		core.Advance(helloAt.Add(700*time.Millisecond), &out)
		if got := replKeys(broadcastRepls(out.Drain())); len(got) != 0 {
			t.Fatalf("the stale retry emitted %v, want nothing", got)
		}
		if _, tracked := core.PendingSnapshot()[keyBeta]; tracked {
			t.Fatal("the stale retry did not remove the untrustworthy entry")
		}
	})
}

// seedVersion joins rec under key through the store's LWW merge — the
// exact-version seeding the replication-outcome rows need — and returns the
// installed record (checksum included) for use as a wire fixture.
func seedVersion(t *testing.T, st *stateStore, key string, rec storage.Record) storage.Record {
	t.Helper()
	_, _, err := st.store.Merge(key, rec)
	if err != nil {
		t.Fatalf("store Merge(%q): %v", key, err)
	}
	got, err := st.store.Get(key)
	if err != nil {
		t.Fatalf("store Get(%q) after seeding: %v", key, err)
	}
	return got
}

// TestReplication_staleAnsweredNotAcked proves the strictly-older outcome
// (D-03 refined, PROTO-03): a held k@v5 that receives k@v3 from a peer
// answers with exactly one unicast DataReplication carrying the held record
// and sends NO Ack. The store is not merged (no conflict, no record change),
// the tracking ledger is untouched — the stale datagram neither creates nor
// removes an entry — and the reply leaves through the same paced egress queue
// as any other answer.
func TestReplication_staleAnsweredNotAcked(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out) // startup Hello + empty listing
	out.Drain()
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	// Drain the first-seen listing so the only unicast left below is the
	// stale-record answer (PROTO-04).
	advanceDrain(core, &out, now)
	out.Drain()

	v5 := version.PackVersion(testNowMs, 5, peerID1)
	held := seedVersion(t, st, keyAlpha, storage.Record{Version: v5, Data: []byte("five")})
	core.NotifyLocalWrite(now, keyAlpha, held, &out)
	out.Drain()
	assertPending(t, core.PendingSnapshot(), keyAlpha, v5, []uint16{peerID1})

	stale := storage.Record{Version: version.PackVersion(testNowMs, 3, peerID1), Data: []byte("three")}
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, stale), &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("the stale DataReplication emitted %d messages directly, want 0 — the answer queues", got)
	}
	if got := core.QueuedResponses(); got != 1 {
		t.Fatalf("QueuedResponses() = %d after the stale record, want 1", got)
	}
	if stats := core.MergeStats(); stats.Conflicts != 0 || stats.DriftWarnings != 0 {
		t.Fatalf("the stale record counted merge stats %+v, want none — the store was not merged", stats)
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, v5, []uint16{peerID1})

	drained := advanceDrain(core, &out, now)
	answered := unicasts(drained)
	if len(answered) != 1 || answered[0].To != handleFor(peerID1) {
		t.Fatalf("the drain emitted %v, want exactly one unicast answer to %q and no Ack", answered, handleFor(peerID1))
	}
	reply, ok := answered[0].Msg.(port.DataReplication)
	if !ok {
		t.Fatalf("the answer is a %T, want port.DataReplication — a stale record must never be Acked", answered[0].Msg)
	}
	if reply.Key != keyAlpha || reply.Record.Version != v5 || !bytes.Equal(reply.Record.Data, []byte("five")) {
		t.Fatalf("the answer = (%q, %+v), want the held v5 record", reply.Key, reply.Record)
	}
	got, err := st.store.Get(keyAlpha)
	if err != nil || got.Version != v5 {
		t.Fatalf("store Get(%q) after the stale record = (%x, %v), want the untouched v5", keyAlpha, got.Version[:], err)
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, v5, []uint16{peerID1})
}

// TestReplication_equalVersionAcked proves the equal-version outcome
// (S-PROTO-16): a held k@v5 that receives k@v5 emits exactly one Ack and no
// reply, with no merge-side conflict counted. Equality must keep Acking or
// retransmission ladders never drain (R-02).
func TestReplication_equalVersionAcked(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	v5 := version.PackVersion(testNowMs, 5, peerID1)
	held := seedVersion(t, st, keyAlpha, storage.Record{Version: v5, Data: []byte("five")})

	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, held), &out)
	assertAck(t, out.Drain(), handleFor(peerID1), keyAlpha, v5)
	if got := core.QueuedResponses(); got != 0 {
		t.Fatalf("QueuedResponses() = %d after an equal-version record, want 0 — equality Acks and never replies", got)
	}
	if stats := core.MergeStats(); stats.Conflicts != 0 || stats.DriftWarnings != 0 {
		t.Fatalf("the equal version counted merge stats %+v, want none", stats)
	}
	got, err := st.store.Get(keyAlpha)
	if err != nil || got.Version != v5 {
		t.Fatalf("store Get(%q) after the equal delivery = (%x, %v), want the unchanged v5", keyAlpha, got.Version[:], err)
	}
}

// TestReplication_newerApplied proves the strictly-newer outcome: a held
// k@v5 that receives k@v6 applies it through ApplyRemote (one displaced-
// version conflict) and Acks v6, emitting no reply.
func TestReplication_newerApplied(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	seedVersion(t, st, keyAlpha, storage.Record{Version: version.PackVersion(testNowMs, 5, peerID1), Data: []byte("five")})
	v6 := version.PackVersion(testNowMs, 6, peerID1)
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, storage.Record{Version: v6, Data: []byte("six")}), &out)
	assertAck(t, out.Drain(), handleFor(peerID1), keyAlpha, v6)
	if got := core.QueuedResponses(); got != 0 {
		t.Fatalf("QueuedResponses() = %d after a strictly newer record, want 0", got)
	}
	if stats := core.MergeStats(); stats.Conflicts != 1 {
		t.Fatalf("MergeStats().Conflicts = %d after displacing v5, want 1", stats.Conflicts)
	}
	got, err := st.store.Get(keyAlpha)
	if err != nil || got.Version != v6 || !bytes.Equal(got.Data, []byte("six")) {
		t.Fatalf("store Get(%q) after the newer apply = (%x, %v), want v6", keyAlpha, got.Version[:], err)
	}
}

// TestReplication_absentApplies proves the absent-key outcome: with no local
// record, an incoming k@v3 applies and Acks — there is no stale case without
// a held record.
func TestReplication_absentApplies(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	v3 := version.PackVersion(testNowMs, 3, peerID1)
	core.HandleDatagram(now, replFrom(peerID1, keyGamma, storage.Record{Version: v3, Data: []byte("three")}), &out)
	assertAck(t, out.Drain(), handleFor(peerID1), keyGamma, v3)
	if got := core.QueuedResponses(); got != 0 {
		t.Fatalf("QueuedResponses() = %d after an absent-key record, want 0", got)
	}
	got, err := st.store.Get(keyGamma)
	if err != nil || got.Version != v3 {
		t.Fatalf("store Get(%q) after the absent-key apply = (%x, %v), want v3", keyGamma, got.Version[:], err)
	}
}

// TestNotify_firstNotificationProceeds pins the notification guard's normal
// path (PROTO-05, D-05): the author's first notification of a stored record
// has no tracked entry, so it always broadcasts and creates the tracking
// entry. The guard must never compare against store.Version with <= — the
// store already holds exactly the notified version here, so a store
// comparison would swallow this case.
func TestNotify_firstNotificationProceeds(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	ver, _, err := st.store.Set(keyAlpha, []byte("v5"))
	if err != nil {
		t.Fatalf("store Set(%q): %v", keyAlpha, err)
	}
	rec, err := st.store.Get(keyAlpha)
	if err != nil {
		t.Fatalf("store Get(%q): %v", keyAlpha, err)
	}
	core.NotifyLocalWrite(now, keyAlpha, rec, &out)
	drained := out.Drain()
	if len(drained) != 1 || !drained[0].Broadcast {
		t.Fatalf("the first notification drained %d messages, want exactly 1 broadcast: %+v", len(drained), drained)
	}
	msg, ok := drained[0].Msg.(port.DataReplication)
	if !ok || msg.Key != keyAlpha || msg.Record.Version != ver {
		t.Fatalf("the first notification emitted %+v, want a DataReplication for (%s, %x)", drained[0].Msg, keyAlpha, ver[:])
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, nil)
	if got := core.NotificationsRegressed(); got != 0 {
		t.Fatalf("NotificationsRegressed() = %d after a first notification, want 0 — the normal path is never a regression", got)
	}
}

// TestNotify_equalTrackedIgnored proves the equal-duplicate half of the guard
// (PROTO-05, D-05): re-notifying the already-tracked version broadcasts
// nothing, creates no second entry, leaves the tracked version alone and
// counts one regression.
func TestNotify_equalTrackedIgnored(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	ver := localWrite(t, core, st, now, keyAlpha, "v5")

	rec, err := st.store.Get(keyAlpha)
	if err != nil {
		t.Fatalf("store Get(%q): %v", keyAlpha, err)
	}
	core.NotifyLocalWrite(now, keyAlpha, rec, &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("an equal duplicate notification emitted %d messages, want 0 — it must never reach the wire", got)
	}
	if got := core.NotificationsRegressed(); got != 1 {
		t.Fatalf("NotificationsRegressed() = %d after one equal duplicate, want 1", got)
	}
	if got := core.UnackedStats(now).Count; got != 1 {
		t.Fatalf("UnackedStats().Count = %d after an equal duplicate, want 1 — no second tracking entry", got)
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, nil)
}

// TestNotify_regressedIgnored proves the regressive half of the guard
// (PROTO-05, D-05): notifying a version older than the tracked one
// broadcasts nothing, keeps the newer tracking entry and counts one
// regression.
func TestNotify_regressedIgnored(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	ver := localWrite(t, core, st, now, keyAlpha, "v5")

	older := storage.Record{
		Version: version.PackVersion(testNowMs-1, 0, localID),
		Data:    []byte("v3"),
	}
	if older.Version.Compare(ver) >= 0 {
		t.Fatalf("test fixture version %x is not strictly older than the tracked %x", older.Version[:], ver[:])
	}
	core.NotifyLocalWrite(now, keyAlpha, older, &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("a regressive notification emitted %d messages, want 0 — it must never reach the wire", got)
	}
	if got := core.NotificationsRegressed(); got != 1 {
		t.Fatalf("NotificationsRegressed() = %d after one regressive notification, want 1", got)
	}
	if got := core.UnackedStats(now).Count; got != 1 {
		t.Fatalf("UnackedStats().Count = %d after a regressive notification, want 1 — tracking must stay on the newer version", got)
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, nil)
}

// TestNotify_newerBroadcasts proves the accepted-change side of the guard
// (PROTO-05): a notification newer than the tracked version supersedes the
// old tracking — emitting its Superseded result — and broadcasts.
func TestNotify_newerBroadcasts(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, st := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()
	older := localWrite(t, core, st, now, keyAlpha, "v5")

	ver, _, err := st.store.Set(keyAlpha, []byte("v9"))
	if err != nil {
		t.Fatalf("store Set(%q): %v", keyAlpha, err)
	}
	if ver.Compare(older) <= 0 {
		t.Fatalf("second store write version %x is not strictly newer than %x", ver[:], older[:])
	}
	rec, err := st.store.Get(keyAlpha)
	if err != nil {
		t.Fatalf("store Get(%q): %v", keyAlpha, err)
	}
	core.NotifyLocalWrite(now, keyAlpha, rec, &out)

	drained := out.Drain()
	if len(drained) != 1 || !drained[0].Broadcast {
		t.Fatalf("the newer notification drained %d messages, want exactly 1 broadcast: %+v", len(drained), drained)
	}
	msg, ok := drained[0].Msg.(port.DataReplication)
	if !ok || msg.Key != keyAlpha || msg.Record.Version != ver {
		t.Fatalf("the newer notification emitted %+v, want a DataReplication for (%s, %x)", drained[0].Msg, keyAlpha, ver[:])
	}
	results := out.DrainResults()
	if len(results) != 1 || results[0].Kind != port.ResultSuperseded || results[0].Key != keyAlpha || results[0].Version != older {
		t.Fatalf("the newer notification results = %+v, want exactly one Superseded for (%s, %x)", results, keyAlpha, older[:])
	}
	assertPending(t, core.PendingSnapshot(), keyAlpha, ver, nil)
	if got := core.NotificationsRegressed(); got != 0 {
		t.Fatalf("NotificationsRegressed() = %d after an accepted newer notification, want 0", got)
	}
}

// TestNotify_absentKeyBroadcasts proves the empty case of the guard: a key
// with no tracked entry proceeds and creates tracking — absent means accept,
// never a regression.
func TestNotify_absentKeyBroadcasts(t *testing.T) {
	t.Parallel()
	now := at(0)
	core, _ := newTestCore()
	var out port.Outbox
	core.Advance(now, &out)
	out.Drain()

	ver := version.PackVersion(testNowMs, 0, peerID2)
	core.NotifyLocalWrite(now, keyGamma, storage.Record{Version: ver, Data: []byte("first")}, &out)
	drained := out.Drain()
	if len(drained) != 1 || !drained[0].Broadcast {
		t.Fatalf("an untracked-key notification drained %d messages, want exactly 1 broadcast: %+v", len(drained), drained)
	}
	msg, ok := drained[0].Msg.(port.DataReplication)
	if !ok || msg.Key != keyGamma || msg.Record.Version != ver {
		t.Fatalf("the untracked-key notification emitted %+v, want a DataReplication for (%s, %x)", drained[0].Msg, keyGamma, ver[:])
	}
	assertPending(t, core.PendingSnapshot(), keyGamma, ver, nil)
	if got := core.NotificationsRegressed(); got != 0 {
		t.Fatalf("NotificationsRegressed() = %d after an untracked-key notification, want 0", got)
	}
}
