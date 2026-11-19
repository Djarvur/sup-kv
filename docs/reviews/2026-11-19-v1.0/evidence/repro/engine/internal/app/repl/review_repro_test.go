package repl_test

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// ENG-06 (cross-boundary): the HLC is advanced only inside Store.Merge. A
// DataSync listing — the message that opens the startup write gate — never
// reaches the clock, because neither Store nor Engine exposes an "observe a
// remote version" seam. A write accepted right after the gate opens is
// stamped BELOW a version the node has already been told about.
func TestReview_dataSyncDoesNotAdvanceHLC(t *testing.T) {
	wall := time.UnixMilli(1_700_000_000_000)
	store := storage.NewStore(1, clock.NewClock(func() time.Time { return wall }, 0, 0, time.Minute), storage.Config{})
	eng := persist.New(store, nopDump{}, persist.Config{})
	core := repl.New(1, repl.NewRunID(), "self", eng, repl.Config{})

	// Peer 2's clock runs 90 s ahead; while we were down it wrote key "k".
	peerVersion := version.PackVersion(wall.Add(90*time.Second).UnixMilli(), 0, 2)
	var out port.Outbox
	core.HandleDatagram(wall, port.InboundDatagram{
		Source: "peer", Status: port.Decoded, ProtoVersion: repl.SupportedProtoVersion, NodeID: 2, RunID: repl.NewRunID(),
		Msg: port.DataSync{Entries: []port.SyncEntry{{Key: "k", Version: peerVersion}}},
	}, &out)
	if !core.SyncArrived() {
		t.Fatal("sync did not arrive")
	}
	// SyncArrived() == true opens the gRPC write gate; a client now writes "k".
	local, _, err := store.Set("k", []byte("written after the listing was received"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("peer version seen in the listing: %x\nlocal version stamped afterwards:  %x", peerVersion[:], local[:])
	if local.Compare(peerVersion) <= 0 {
		t.Errorf("the write stamped AFTER receiving the peer's version sorts below it: " +
			"when the DataRequest reply arrives, the peer's older value displaces this confirmed write")
	}
}
