package persist

// White-box test for replayRecord's replayed/skip signal (PROTO-08). It
// lives in package persist because replayRecord is unexported — the signal
// only exists to tell Restore what to report — and the _internal_test.go
// suffix is the documented testpackage exemption.

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// replaySignalNode is the store's stamping node id: distinct from the
// snapshot versions' node so a version comparison names the store's stamp.
const replaySignalNode = uint16(0x0304)

// TestEngine_replayRecordSignal proves the replayed bool is an explicit
// install/skip signal, never conflated with the tombstone flag: a live
// replay and a tombstone replay both report replayed=true with the freshly
// stamped nonzero version, while the one skip shape — a tombstone with
// nothing to beat — reports replayed=false with the zero version and a nil
// error. A real replay and a skip can therefore never read the same.
//
// The over-limit class is deliberately NOT a skip (IN-01): Restore decodes
// with the store's own configured limit, so Decode discards such a record
// (and CR-01 refuses the whole load), and replayRecord never sees one. A
// direct call proves the store stays the bound's authority: the error
// surfaces instead of silently installing nothing.
func TestEngine_replayRecordSignal(t *testing.T) {
	t.Parallel()
	clk := clock.NewClock(time.Now, 0, 0, time.Minute)
	store := storage.NewStore(replaySignalNode, clk, storage.Config{ValueMaxBytes: 16})

	seeded, _, err := store.Set("live", []byte("seeded"))
	if err != nil {
		t.Fatalf("seed store.Set(live) error = %v, want nil", err)
	}

	liveVer, replayed, err := replayRecord(store, "live", storage.Record{Data: []byte("replayed")})
	if err != nil {
		t.Fatalf("replayRecord(live) error = %v, want nil", err)
	}
	if !replayed {
		t.Error("replayRecord(live) replayed = false, want true for an installed live record")
	}
	if liveVer == (version.RecordVersion{}) {
		t.Error("replayRecord(live) version is the zero version, want the store's fresh stamp")
	}
	if liveVer.Compare(seeded) != 1 {
		t.Errorf("replayRecord(live) version %x does not dominate the seeded %x", liveVer[:], seeded[:])
	}

	tombVer, replayed, err := replayRecord(store, "live", storage.Record{Deleted: true})
	if err != nil {
		t.Fatalf("replayRecord(tombstone) error = %v, want nil", err)
	}
	if !replayed {
		t.Error("replayRecord(tombstone) replayed = false, want true for an installed tombstone")
	}
	if tombVer == (version.RecordVersion{}) {
		t.Error("replayRecord(tombstone) version is the zero version, want the store's fresh stamp")
	}
	if tombVer.Compare(liveVer) != 1 {
		t.Errorf("replayRecord(tombstone) version %x does not dominate the live replay's %x", tombVer[:], liveVer[:])
	}

	skipVer, replayed, err := replayRecord(store, "absent", storage.Record{Deleted: true})
	if err != nil {
		t.Fatalf("replayRecord(absent tombstone) error = %v, want nil", err)
	}
	if replayed {
		t.Error("replayRecord(absent tombstone) replayed = true, want false for a skip")
	}
	if skipVer != (version.RecordVersion{}) {
		t.Errorf("replayRecord(absent tombstone) version = %x, want the zero version", skipVer[:])
	}

	_, replayed, err = replayRecord(store, "oversize", storage.Record{
		Data: bytes.Repeat([]byte("x"), 17),
	})
	if !errors.Is(err, storage.ErrValueTooLarge) {
		t.Fatalf("replayRecord(oversize) error = %v, want ErrValueTooLarge (the store is the bound's authority)", err)
	}
	if replayed {
		t.Error("replayRecord(oversize) replayed = true, want false on the rejected Set")
	}
	if liveVer == skipVer {
		t.Error("a live replay and a skip read the same version, want distinguishable signals")
	}
}
