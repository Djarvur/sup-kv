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
// seam — by delegating both methods (the persist engine's ApplyRemote seam
// lands with plan 04-06).
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			t.Parallel()
			tt.run(t)
		})
	}
}
