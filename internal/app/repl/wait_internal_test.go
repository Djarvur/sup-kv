package repl

// White-box test for the discovery flush's promise-set boundary (A-01,
// IN-05, RESEARCH Pitfall 4). The ledger (peerUnacked) and the paced
// record-send queue are unexported core state, and the invariant under test
// is exactly "the flush broadcasts and re-arms without inserting the new
// peer into any wait-set or ledger" — a property only the same package can
// observe. The _internal_test.go suffix is the documented testpackage
// exemption.

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The white-box fixtures: a fixed virtual base and three distinct node ids
// (the local node, the captured peer the pending entry promises, and the
// peer discovered by the flush). Kept local so this file does not depend on
// another internal test's constants.
const (
	waitFlushNowMs = int64(1730000000000)
	waitFlushLocal = uint16(1)
	waitFlushPeer1 = uint16(2)
	waitFlushPeer2 = uint16(3)
)

// TestPeerTable_discoveryFlushKeepsPromise pins the IN-05 split (D-01,
// RESEARCH Pitfall 4): a Hello that ends an empty peer table still resets
// every pending entry's ladder and enqueues its record send — the new peer
// must receive the pending set — but it must NOT insert that peer into the
// entry's wait-set or its per-peer ledger. The promise is exactly the
// write-time capture; a peer discovered after acceptance can be broadcast
// to, never awaited.
//
// The entry is installed with a captured-but-currently-unrepresented peer
// (waitFlushPeer1 in the wait-set while the table is empty): the state the
// pre-split flush would grow. The synthetic installation is the point —
// through the public seams an eviction empties the wait-set before the table
// empties, so only a white-box fixture can hold the invariant's two halves
// apart.
func TestPeerTable_discoveryFlushKeepsPromise(t *testing.T) {
	t.Parallel()

	now := time.UnixMilli(waitFlushNowMs)
	core := New(waitFlushLocal, [16]byte{0xA1}, &egressStore{
		versions: make(map[string]version.RecordVersion),
		records:  make(map[string]storage.Record),
	}, Config{})

	key := "wait/no-growth"
	ver := version.PackVersion(waitFlushNowMs, 0, waitFlushLocal)
	pk := pendingKey{key: key, version: ver}
	entry := pendingEntry{
		version:   ver,
		waitSet:   map[uint16]struct{}{waitFlushPeer1: {}},
		nextRetry: now.Add(50 * time.Millisecond),
		backoff:   50 * time.Millisecond,
		createdAt: now,
	}
	entry.elem = core.insertPendingOrder(pk, now)
	core.pending[pk] = entry
	core.ledgerAdd(waitFlushPeer1, pk)
	core.pushRetry(pk, entry.nextRetry)

	// The Hello ends the empty table: discovery.
	hello := port.InboundDatagram{
		Source:       "n3",
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       waitFlushPeer2,
		RunID:        [16]byte{3},
		Msg:          port.Hello{},
	}
	var out port.Outbox
	core.HandleDatagram(now, hello, &out)

	got, tracked := core.pending[pk]
	if !tracked {
		t.Fatal("the discovery flush dropped the pending entry, want it retained")
	}
	if _, present := got.waitSet[waitFlushPeer2]; present {
		t.Fatalf("the discovery flush added peer %d to the wait-set %v, want the promise unchanged (IN-05)",
			waitFlushPeer2, got.waitSet)
	}
	if _, present := got.waitSet[waitFlushPeer1]; !present || len(got.waitSet) != 1 {
		t.Fatalf("the discovery flush changed the wait-set to %v, want exactly the captured {%d}", got.waitSet, waitFlushPeer1)
	}
	if _, present := core.peerUnacked[waitFlushPeer2]; present {
		t.Fatalf("the discovery flush gave peer %d a ledger entry %v, want none — the promise must not grow",
			waitFlushPeer2, core.peerUnacked[waitFlushPeer2])
	}
	if _, present := core.peerUnacked[waitFlushPeer1][pk]; !present {
		t.Fatalf("the captured peer's ledger entry for %v disappeared, want it retained", pk)
	}

	// The broadcast half stays: the ladder is restarted at ack_timeout and
	// the record send is queued for the new peer.
	if want := now.Add(core.cfg.AckTimeout); !got.nextRetry.Equal(want) {
		t.Fatalf("post-flush NextRetry = %v, want the restarted ladder step %v", got.nextRetry, want)
	}
	if len(core.recordQueue) != 1 || core.recordQueue[0].pk != pk {
		t.Fatalf("the discovery flush queued %v, want exactly one record send for %v", core.recordQueue, pk)
	}
	if _, queued := core.recordQueued[pk]; !queued {
		t.Fatalf("the discovery flush's record send for %v is not in the dedupe set %v", pk, core.recordQueued)
	}
}
