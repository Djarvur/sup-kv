package storage_test

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// Shared test constants: a fixed wall-clock millisecond (the D-02 seam), the
// test node ID, the default value limit from docs/IDEA.md parameters, the
// drift tolerance (the design default the store's clock used to carry), the
// RFC 9562 domain ceilings, the UUIDv7 marker bits, and Compare result
// codes.
const (
	testNowMs      = int64(1730000000000)
	testNodeID     = uint16(0xABCD)
	valueMax       = 61440
	valueOver      = valueMax + 1
	testDriftWarn  = time.Minute
	maxNodeID      = 0xFFFF
	testCounterMax = 4095

	uuidVerMask = 0xF0 // high nibble of octet 6 carries the UUID version.
	uuidVerBits = 0x70 // version 7.
	uuidVarMask = 0xC0 // top two bits of octet 8 carry the UUID variant.
	uuidVarBits = 0x80 // variant 0b10.

	cmpOlder           = -1
	cmpSame            = 0
	cmpNewer           = 1
	firstWriteCounter  = 0
	secondWriteCounter = 1
)

// fixedNow returns a now func parked at the fixed test wall-clock
// millisecond, built with time.UnixMilli so no monotonic reading is involved
// (D-02).
func fixedNow() func() time.Time {
	return func() time.Time { return time.UnixMilli(testNowMs) }
}

// newStoreFor returns a store on the given node with the default config at
// the fixed wall clock, its clock carrying the design-default drift
// tolerance.
func newStoreFor(nodeID uint16) *storage.Store {
	return storage.NewStore(nodeID, clock.NewClock(fixedNow(), 0, 0, testDriftWarn), storage.Config{})
}

// newStore returns a store on the test node with the default config at the
// fixed wall clock.
func newStore() *storage.Store {
	return newStoreFor(testNodeID)
}

// TestStore_endToEnd wires one local write through every layer: the injected
// now seam, the HLC Tick, the UUIDv7 pack, Set and Get (D-01, D-02).
func TestStore_endToEnd(t *testing.T) {
	t.Parallel()
	t.Run("set and get round-trip exact bytes with the stamped version", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		ver, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		l, counter, nodeID := ver.Unpack()
		if l != testNowMs || counter != firstWriteCounter || nodeID != testNodeID {
			t.Errorf("Unpack() = (%d, %d, %d), want (%d, %d, %d)",
				l, counter, nodeID, testNowMs, firstWriteCounter, testNodeID)
		}
		rec, err := store.Get("k")
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
		if rec.Version != ver {
			t.Errorf("Get() version = %x, want %x", rec.Version[:], ver[:])
		}
		if !bytes.Equal(rec.Data, []byte("v")) {
			t.Errorf("Get() data = %q, want %q", rec.Data, "v")
		}
	})
	t.Run("the packed version carries the UUIDv7 marker bits", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		ver, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		if got := ver[6] & uuidVerMask; got != uuidVerBits {
			t.Errorf("version octet 6 version bits = %#x, want %#x", got, uuidVerBits)
		}
		if got := ver[8] & uuidVarMask; got != uuidVarBits {
			t.Errorf("version octet 8 variant bits = %#x, want %#x", got, uuidVarBits)
		}
	})
	t.Run("a second set replaces the value under a strictly greater version", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		first, _, err := store.Set("k", []byte("one"))
		if err != nil {
			t.Fatalf("first Set() error = %v, want nil", err)
		}
		second, _, err := store.Set("k", []byte("two"))
		if err != nil {
			t.Fatalf("second Set() error = %v, want nil", err)
		}
		if got := second.Compare(first); got != cmpNewer {
			t.Errorf("Compare() = %d, want %d", got, cmpNewer)
		}
		if _, counter, _ := second.Unpack(); counter != secondWriteCounter {
			t.Errorf("second Unpack() counter = %d, want %d", counter, secondWriteCounter)
		}
		rec, err := store.Get("k")
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
		if !bytes.Equal(rec.Data, []byte("two")) {
			t.Errorf("Get() data = %q, want %q", rec.Data, "two")
		}
	})
	t.Run("get on a missing key returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, err := store.Get("missing")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})
	t.Run("set over the size limit is rejected and stores nothing", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", make([]byte, valueOver))
		if !errors.Is(err, storage.ErrValueTooLarge) {
			t.Errorf("Set() error = %v, want ErrValueTooLarge", err)
		}
		_, err = store.Get("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound (nothing stored)", err)
		}
	})
}

// Merge-test fixtures: remote records are constructed directly with
// PackVersion, so the merge unit tests are clock-free and fully
// deterministic.
const (
	mergeL     = testNowMs
	mergeLNext = mergeL + 1
	nodeA      = uint16(0x1111)
	nodeB      = uint16(0x2222)
)

// conflictsEqual reports whether two conflict slices match element-wise.
func conflictsEqual(got, want []storage.Conflict) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestStore_Get covers the Get visibility rule: a live key returns its data
// and version, while a missing key and a tombstoned key are both
// ErrNotFound — the tombstone is invisible to Get (CORE-01).
func TestStore_Get(t *testing.T) {
	t.Parallel()
	t.Run("live key returns its data and version", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		ver, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		rec, err := store.Get("k")
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
		if rec.Version != ver || !bytes.Equal(rec.Data, []byte("v")) {
			t.Errorf("Get() = (%x, %q), want (%x, %q)", rec.Version[:], rec.Data, ver[:], "v")
		}
	})
	t.Run("missing key returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, err := store.Get("missing")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})
	t.Run("tombstoned key returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		_, _, err = store.Delete("k")
		if err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
		_, err = store.Get("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound (tombstone invisible)", err)
		}
	})
}

// TestStore_Set covers the Set contract: strictly increasing versions with
// same-millisecond disambiguation by the counter, zero-length values, and
// fresh versions on repeat writes (CORE-02; the replace-under-greater-
// version observable, including the Get-read of the second write, is pinned
// once in TestStore_endToEnd — the P2-IN-03 dedup).
func TestStore_Set(t *testing.T) {
	t.Parallel()
	t.Run("two sets inside one millisecond get distinct strictly increasing versions", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		first, _, err := store.Set("k", []byte("one"))
		if err != nil {
			t.Fatalf("first Set() error = %v, want nil", err)
		}
		second, _, err := store.Set("k", []byte("two"))
		if err != nil {
			t.Fatalf("second Set() error = %v, want nil", err)
		}
		l1, c1, _ := first.Unpack()
		l2, c2, _ := second.Unpack()
		if l1 != l2 || l1 != testNowMs {
			t.Errorf("timestamps = (%d, %d), want both %d", l1, l2, testNowMs)
		}
		if got := second.Compare(first); got != cmpNewer {
			t.Errorf("Compare() = %d, want %d", got, cmpNewer)
		}
		if c1 != firstWriteCounter || c2 != secondWriteCounter {
			t.Errorf("counters = (%d, %d), want (%d, %d)", c1, c2, firstWriteCounter, secondWriteCounter)
		}
	})
	t.Run("a zero-length value is stored and returned", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", nil)
		if err != nil {
			t.Fatalf("Set(nil) error = %v, want nil", err)
		}
		rec, err := store.Get("k")
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
		if len(rec.Data) != 0 {
			t.Errorf("Get() data length = %d, want 0", len(rec.Data))
		}
	})
	t.Run("writing the same value twice stamps a fresh version each time", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		first, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("first Set() error = %v, want nil", err)
		}
		second, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("second Set() error = %v, want nil", err)
		}
		if got := second.Compare(first); got != cmpNewer {
			t.Errorf("Compare() = %d, want %d (fresh version per write)", got, cmpNewer)
		}
	})
}

// TestStore_SetIfAbsent covers the create-if-absent contract: creation on an
// absent key, ErrAlreadyExists on any existing record including a tombstone,
// so a Delete-then-SetIfAbsent cannot recreate the key (CORE-03).
func TestStore_SetIfAbsent(t *testing.T) {
	t.Parallel()
	t.Run("creates on an absent key", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		ver, _, err := store.SetIfAbsent("k", []byte("v"))
		if err != nil {
			t.Fatalf("SetIfAbsent() error = %v, want nil", err)
		}
		if _, _, nodeID := ver.Unpack(); nodeID != testNodeID {
			t.Errorf("Unpack() nodeID = %d, want %d", nodeID, testNodeID)
		}
		rec, err := store.Get("k")
		if err != nil || !bytes.Equal(rec.Data, []byte("v")) {
			t.Errorf("Get() = (%q, %v), want (%q, nil)", rec.Data, err, "v")
		}
	})
	t.Run("existing live key returns ErrAlreadyExists and keeps the stored value", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", []byte("one"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		_, _, err = store.SetIfAbsent("k", []byte("two"))
		if !errors.Is(err, storage.ErrAlreadyExists) {
			t.Errorf("SetIfAbsent() error = %v, want ErrAlreadyExists", err)
		}
		rec, err := store.Get("k")
		if err != nil || !bytes.Equal(rec.Data, []byte("one")) {
			t.Errorf("Get() = (%q, %v), want (%q, nil)", rec.Data, err, "one")
		}
	})
	t.Run("existing tombstoned key returns ErrAlreadyExists without recreation", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", []byte("one"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		_, _, err = store.Delete("k")
		if err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
		_, _, err = store.SetIfAbsent("k", []byte("two"))
		if !errors.Is(err, storage.ErrAlreadyExists) {
			t.Errorf("SetIfAbsent() error = %v, want ErrAlreadyExists (tombstone counts as existing)", err)
		}
		_, err = store.Get("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound (no resurrection)", err)
		}
	})
}

// TestStore_Delete covers the tombstone contract: a live key gets a fresh
// strictly greater tombstone that stays in the map, while missing and
// already-tombstoned keys are ErrNotFound and write nothing, so no version
// is burned on a no-op (CORE-04).
func TestStore_Delete(t *testing.T) {
	t.Parallel()
	t.Run("live key gets a strictly greater fresh tombstone", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		live, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		tomb, _, err := store.Delete("k")
		if err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
		if got := tomb.Compare(live); got != cmpNewer {
			t.Errorf("tombstone Compare(live) = %d, want %d", got, cmpNewer)
		}
		_, err = store.Get("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
		rec, ok := store.Records()["k"]
		if !ok || !rec.Deleted {
			t.Errorf("Records()[k] = (%x, deleted=%t, ok=%t), want a present tombstone",
				rec.Version[:], rec.Deleted, ok)
		}
	})
	t.Run("missing key returns ErrNotFound and burns no version", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Delete("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Delete() error = %v, want ErrNotFound", err)
		}
		if got := len(store.Records()); got != 0 {
			t.Errorf("len(Records()) = %d, want 0 (nothing written)", got)
		}
	})
	t.Run("already tombstoned key returns ErrNotFound and writes nothing", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		first, _, err := store.Delete("k")
		if err != nil {
			t.Fatalf("first Delete() error = %v, want nil", err)
		}
		_, _, err = store.Delete("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("second Delete() error = %v, want ErrNotFound", err)
		}
		rec, ok := store.Records()["k"]
		if !ok || rec.Version != first {
			t.Errorf("Records()[k] version = %x, want the first tombstone %x", rec.Version[:], first[:])
		}
	})
}

// TestStore_valueLimit covers the value size limit at its boundary on all
// three write paths: exactly ValueMaxBytes bytes are stored, one byte more
// is rejected with ErrValueTooLarge before anything enters the map, and a
// zero-length value is legal (CORE-05).
func TestStore_valueLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		op      string
		size    int
		wantErr error
	}{
		{name: "set accepts exactly the limit", op: "set", size: valueMax},
		{name: "set rejects one byte over the limit", op: "set", size: valueOver, wantErr: storage.ErrValueTooLarge},
		{name: "setifabsent accepts exactly the limit", op: "setifabsent", size: valueMax},
		{name: "setifabsent rejects one byte over the limit", op: "setifabsent", size: valueOver, wantErr: storage.ErrValueTooLarge},
		{name: "merge accepts exactly the limit", op: "merge", size: valueMax},
		{name: "merge rejects one byte over the limit", op: "merge", size: valueOver, wantErr: storage.ErrValueTooLarge},
		{name: "set accepts a zero-length value", op: "set", size: 0},
		{name: "setifabsent accepts a zero-length value", op: "setifabsent", size: 0},
		{name: "merge accepts a zero-length value", op: "merge", size: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newStore()
			var err error
			switch tt.op {
			case "set":
				_, _, err = store.Set("k", make([]byte, tt.size))
			case "setifabsent":
				_, _, err = store.SetIfAbsent("k", make([]byte, tt.size))
			case "merge":
				rec := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: make([]byte, tt.size)}
				_, _, err = store.Merge("k", rec)
			default:
				t.Fatalf("unknown op %q", tt.op)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("%s of %d bytes: error = %v, want %v", tt.op, tt.size, err, tt.wantErr)
			}
			_, ok := store.Records()["k"]
			if tt.wantErr != nil && ok {
				t.Errorf("Records() holds %q despite rejection", "k")
			}
			if tt.wantErr == nil && !ok {
				t.Errorf("Records() lacks %q after an accepted write", "k")
			}
		})
	}
}

// TestMerge_losers covers the merge contract: remote records keep their own
// versions, only a strictly greater full 16-byte version displaces, an equal
// version is a full no-op, and every displaced or rejected version is
// returned exactly once as a Conflict (CORE-06, CORE-07).
func TestMerge_losers(t *testing.T) {
	t.Parallel()
	t.Run("newer incoming displaces stored and reports it as the loser", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		stored := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		incoming := storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeB), Data: []byte("b")}
		conflicts, _, err := store.Merge("k", incoming)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: incoming.Version, Loser: stored.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v", conflicts, want)
		}
		rec, err := store.Get("k")
		if err != nil || rec.Version != incoming.Version || !bytes.Equal(rec.Data, []byte("b")) {
			t.Errorf("Get() = (%x, %q, %v), want the incoming record", rec.Version[:], rec.Data, err)
		}
	})
	t.Run("older incoming loses and is reported as the loser", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		stored := storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		incoming := storage.Record{Version: version.PackVersion(mergeL, 0, nodeB), Data: []byte("b")}
		if got := incoming.Version.Compare(stored.Version); got != cmpOlder {
			t.Fatalf("fixture Compare() = %d, want %d", got, cmpOlder)
		}
		conflicts, _, err := store.Merge("k", incoming)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: stored.Version, Loser: incoming.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v", conflicts, want)
		}
		rec, err := store.Get("k")
		if err != nil || !bytes.Equal(rec.Data, []byte("a")) {
			t.Errorf("Get() = (%q, %v), want the stored value (%q, nil)", rec.Data, err, "a")
		}
	})
	t.Run("equal version with greater content wins and reports one conflict", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		stored := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		dup := storage.Record{Version: stored.Version, Data: []byte("x")}
		conflicts, _, err := store.Merge("k", dup)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: stored.Version, Loser: stored.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want exactly one pairing the equal version", conflicts)
		}
		rec, err := store.Get("k")
		if err != nil || rec.Version != stored.Version || !bytes.Equal(rec.Data, []byte("x")) {
			t.Errorf("Get() = (%x, %q, %v), want the greater-content winner %q", rec.Version[:], rec.Data, err, "x")
		}
	})
	t.Run("equal version with identical content is a full no-op", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		stored := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		conflicts, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("redelivery Merge() error = %v, want nil", err)
		}
		if conflicts != nil {
			t.Errorf("redelivery Merge() conflicts = %v, want nil (identical content is silent)", conflicts)
		}
		rec, err := store.Get("k")
		if err != nil || rec.Version != stored.Version || !bytes.Equal(rec.Data, []byte("a")) {
			t.Errorf("Get() = (%x, %q, %v), want the stored record unchanged", rec.Version[:], rec.Data, err)
		}
	})
	t.Run("merging into a missing key installs with no conflicts", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		incoming := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		conflicts, _, err := store.Merge("k", incoming)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		if conflicts != nil {
			t.Errorf("Merge() conflicts = %v, want nil (nothing was displaced)", conflicts)
		}
		rec, err := store.Get("k")
		if err != nil || rec.Version != incoming.Version || !bytes.Equal(rec.Data, []byte("a")) {
			t.Errorf("Get() = (%x, %q, %v), want the incoming record", rec.Version[:], rec.Data, err)
		}
	})
	t.Run("a greater-version tombstone beats an older live record", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		live := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", live)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		tomb := storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeB), Deleted: true}
		conflicts, _, err := store.Merge("k", tomb)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: tomb.Version, Loser: live.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v", conflicts, want)
		}
		_, err = store.Get("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound (no resurrection)", err)
		}
		if rec, ok := store.Records()["k"]; !ok || !rec.Deleted {
			t.Errorf("Records()[k] = (%x, deleted=%t, ok=%t), want a present tombstone",
				rec.Version[:], rec.Deleted, ok)
		}
	})
}

// TestStore_restartDominance covers the counter-seeded restart seam at the
// store level (VERS-04, WR-01): a store whose injected clock carries the
// dump's full watermark — initialL plus initialC — stamps its first
// post-restart write strictly past that watermark even with the wall clock
// parked exactly on the watermark millisecond, the adversarial same-ms
// restart. The regression rows encode both WR-01 consequences as dead: a
// peer holding the dump watermark accepts the fresh write (no silent
// revert of an acknowledged write, probe variant A), and the stale
// watermark record displaces against the fresh write with exactly one
// Conflict (no equal-version no-op silently dropping fresh data, probe
// variant B).
func TestStore_restartDominance(t *testing.T) {
	t.Parallel()
	// watermarkC is the dump watermark's counter at testNowMs, carried by a
	// version of the store's own NodeID — the hardest case the seam must beat.
	const watermarkC = 5
	t.Run("the first post-restart Set stamps strictly past the seeded watermark", func(t *testing.T) {
		t.Parallel()
		store := storage.NewStore(
			testNodeID,
			clock.NewClock(fixedNow(), testNowMs, watermarkC, testDriftWarn),
			storage.Config{},
		)
		ver, _, err := store.Set("k", []byte("fresh"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		l, counter, nodeID := ver.Unpack()
		if l != testNowMs || counter != watermarkC+1 || nodeID != testNodeID {
			t.Errorf("Unpack() = (%d, %d, %d), want (%d, %d, %d)",
				l, counter, nodeID, testNowMs, watermarkC+1, testNodeID)
		}
		watermark := version.PackVersion(testNowMs, watermarkC, testNodeID)
		if got := ver.Compare(watermark); got != cmpNewer {
			t.Errorf("Compare(watermark) = %d, want %d", got, cmpNewer)
		}
	})
	t.Run("InitialC at counterMax bumps the store's first write past the watermark", func(t *testing.T) {
		t.Parallel()
		store := storage.NewStore(
			testNodeID,
			clock.NewClock(fixedNow(), testNowMs, testCounterMax, testDriftWarn),
			storage.Config{},
		)
		ver, _, err := store.Set("k", []byte("fresh"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		l, counter, nodeID := ver.Unpack()
		if l != testNowMs+1 || counter != 0 || nodeID != testNodeID {
			t.Errorf("Unpack() = (%d, %d, %d), want (%d, 0, %d)",
				l, counter, nodeID, testNowMs+1, testNodeID)
		}
		watermark := version.PackVersion(testNowMs, testCounterMax, testNodeID)
		if got := ver.Compare(watermark); got != cmpNewer {
			t.Errorf("Compare(watermark) = %d, want %d", got, cmpNewer)
		}
	})
	t.Run("a peer holding the dump watermark accepts the first post-restart write", func(t *testing.T) {
		t.Parallel()
		restarted := storage.NewStore(
			testNodeID,
			clock.NewClock(fixedNow(), testNowMs, watermarkC, testDriftWarn),
			storage.Config{},
		)
		fresh, _, err := restarted.Set("k", []byte("fresh"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		peer := newStoreFor(nodeB)
		watermark := storage.Record{
			Version: version.PackVersion(testNowMs, watermarkC, testNodeID),
			Data:    []byte("old"),
		}
		_, _, err = peer.Merge("k", watermark)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		conflicts, _, err := peer.Merge("k", storage.Record{Version: fresh, Data: []byte("fresh")})
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: fresh, Loser: watermark.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v (the acknowledged write must not revert)",
				conflicts, want)
		}
		rec, err := peer.Get("k")
		if err != nil || !bytes.Equal(rec.Data, []byte("fresh")) {
			t.Errorf("Get() = (%q, %v), want the fresh write (%q, nil)", rec.Data, err, "fresh")
		}
	})
	t.Run("the stale watermark record cannot collide with the fresh write", func(t *testing.T) {
		t.Parallel()
		restarted := storage.NewStore(
			testNodeID,
			clock.NewClock(fixedNow(), testNowMs, watermarkC, testDriftWarn),
			storage.Config{},
		)
		fresh, _, err := restarted.Set("k", []byte("fresh"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		stale := storage.Record{
			Version: version.PackVersion(testNowMs, watermarkC, testNodeID),
			Data:    []byte("old"),
		}
		conflicts, _, err := restarted.Merge("k", stale)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: fresh, Loser: stale.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v (an equal-version no-op would silently drop the fresh write)",
				conflicts, want)
		}
		rec, err := restarted.Get("k")
		if err != nil || !bytes.Equal(rec.Data, []byte("fresh")) {
			t.Errorf("Get() = (%q, %v), want the fresh data (%q, nil)", rec.Data, err, "fresh")
		}
	})
}

// atomicNow returns a goroutine-safe fake now over an atomic millisecond
// counter: every call advances the wall clock by one millisecond, so
// concurrent Tick and Update callers observe strictly increasing readings
// (the D-02 seam under concurrency — a bare closure over a shared int64
// would race).
func atomicNow(startMs int64) func() time.Time {
	var ms atomic.Int64
	ms.Store(startMs)
	return func() time.Time { return time.UnixMilli(ms.Add(1)) }
}

// TestStore_concurrentHammer races the whole exported surface of one store —
// Set, SetIfAbsent, Delete, Merge and Get — across 8 goroutines x 200
// deterministic-PCG operations over a small shared key set: no operation
// panics, the race detector finds nothing, and once the hammer drains every
// key is still present in Records() with Get agreeing with Records() — a
// live record returns its data, a tombstone returns ErrNotFound (CORE-02
// concurrency probe).
func TestStore_concurrentHammer(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	keys := []string{"hammer-0", "hammer-1", "hammer-2", "hammer-3"}
	store := storage.NewStore(testNodeID, clock.NewClock(atomicNow(testNowMs), 0, 0, testDriftWarn), storage.Config{})
	for _, key := range keys {
		_, _, err := store.Set(key, []byte("seed"))
		if err != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, err)
		}
	}
	hammerSeed := rng.Uint64()
	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Go(func() {
			workerRng := rand.New(rand.NewPCG(hammerSeed, uint64(worker)))
			for range 200 {
				key := keys[workerRng.IntN(len(keys))]
				payload := make([]byte, workerRng.IntN(32))
				for i := range payload {
					payload[i] = byte(workerRng.IntN(256))
				}
				switch workerRng.IntN(10) {
				case 0, 1, 2, 3:
					_, _, _ = store.Set(key, payload)
				case 4, 5:
					_, _, _ = store.SetIfAbsent(key, payload)
				case 6:
					_, _, _ = store.Delete(key)
				case 7, 8:
					rec := storage.Record{
						Version: version.PackVersion(
							testNowMs+workerRng.Int64N(4001),
							uint16(workerRng.IntN(testCounterMax+1)),
							uint16(worker)),
						Data: payload,
					}
					if workerRng.IntN(4) == 0 {
						rec.Deleted = true
						rec.Data = nil
					}
					_, _, _ = store.Merge(key, rec)
				default:
					_, _ = store.Get(key)
				}
			}
		})
	}
	wg.Wait()
	records := store.Records()
	for _, key := range keys {
		rec, ok := records[key]
		if !ok {
			t.Errorf("key %q missing from Records() after the hammer", key)
			continue
		}
		got, err := store.Get(key)
		if rec.Deleted {
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("Get(%q) error = %v, want ErrNotFound for the tombstone", key, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("Get(%q) error = %v, want nil for a live record", key, err)
			continue
		}
		if got.Version != rec.Version || !bytes.Equal(got.Data, rec.Data) {
			t.Errorf("Get(%q) = (%x, %q), want the Records() entry (%x, %q)",
				key, got.Version[:], got.Data, rec.Version[:], rec.Data)
		}
	}
}

// TestStore_concurrentSetIfAbsent races 16 create-if-absent callers on one
// fresh key with distinct payloads: exactly one caller succeeds, every other
// caller gets ErrAlreadyExists, and the stored data equals the single
// winner's payload (CORE-03 concurrency probe).
func TestStore_concurrentSetIfAbsent(t *testing.T) {
	t.Parallel()
	store := newStore()
	const callers = 16
	errs := make([]error, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range callers {
		wg.Go(func() {
			<-start
			_, _, errs[i] = store.SetIfAbsent("fresh", []byte{byte(i)})
		})
	}
	close(start)
	wg.Wait()
	winners := 0
	var winnerPayload []byte
	for i, err := range errs {
		if err == nil {
			winners++
			winnerPayload = []byte{byte(i)}
			continue
		}
		if !errors.Is(err, storage.ErrAlreadyExists) {
			t.Errorf("caller %d error = %v, want nil or ErrAlreadyExists", i, err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful SetIfAbsent callers = %d, want exactly 1", winners)
	}
	rec, err := store.Get("fresh")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if !bytes.Equal(rec.Data, winnerPayload) {
		t.Errorf("Get() data = %q, want the winner payload %q", rec.Data, winnerPayload)
	}
}

// TestStore_concurrentDeleteVsSet duels Delete against Set on one live key
// for 50 rounds of fresh keys, both sides capturing their returned versions:
// after each round the key's Records() entry holds exactly the greater of the
// two captured versions — the tombstone when Delete won, the live payload
// when Set won (CORE-04 concurrency probe).
func TestStore_concurrentDeleteVsSet(t *testing.T) {
	t.Parallel()
	store := newStore()
	for round := range 50 {
		key := fmt.Sprintf("duel-%d", round)
		_, _, err := store.Set(key, []byte("seed"))
		if err != nil {
			t.Fatalf("round %d seed Set() error = %v, want nil", round, err)
		}
		var wg sync.WaitGroup
		var setVer, delVer version.RecordVersion
		wg.Go(func() {
			ver, _, err := store.Set(key, []byte("set"))
			if err != nil {
				t.Errorf("round %d Set() error = %v, want nil", round, err)
				return
			}
			setVer = ver
		})
		wg.Go(func() {
			ver, _, err := store.Delete(key)
			if err != nil {
				t.Errorf("round %d Delete() error = %v, want nil", round, err)
				return
			}
			delVer = ver
		})
		wg.Wait()
		rec, ok := store.Records()[key]
		if !ok {
			t.Fatalf("round %d: key %q missing from Records()", round, key)
		}
		if delVer.Compare(setVer) == cmpNewer {
			if rec.Version != delVer || !rec.Deleted {
				t.Fatalf("round %d: Records()[%q] = (%x, deleted=%t), want the delete tombstone %x",
					round, key, rec.Version[:], rec.Deleted, delVer[:])
			}
			continue
		}
		if rec.Version != setVer || rec.Deleted || !bytes.Equal(rec.Data, []byte("set")) {
			t.Fatalf("round %d: Records()[%q] = (%x, deleted=%t, %q), want the set write (%x, live)",
				round, key, rec.Version[:], rec.Deleted, rec.Data, setVer[:])
		}
	}
}

// changeTapEvent is one OnChange tap firing as the tests observe it: the key
// and the handed record — the store's stored copy, which is replaced
// wholesale on the next install and never mutated in place, so retaining it
// is safe per the field's documented contract.
type changeTapEvent struct {
	key    string
	record storage.Record
}

// changeTapRecorder collects the events a storage.Config.OnChange tap
// fires. Every firing happens under the store write lock, so calls are
// serialized by the store itself; the mutex keeps the recorder race-clean
// for tests collecting after concurrent writers have joined.
type changeTapRecorder struct {
	mu     sync.Mutex
	events []changeTapEvent
}

// tap is the Config.OnChange callback.
func (r *changeTapRecorder) tap(key string, rec storage.Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, changeTapEvent{key: key, record: rec})
}

// count returns how many events have fired so far.
func (r *changeTapRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

// event returns the i-th fired event.
func (r *changeTapRecorder) event(i int) changeTapEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.events[i]
}

// newTapStore returns a store at the fixed wall clock wired with a fresh
// recorder as its OnChange tap.
func newTapStore() (*storage.Store, *changeTapRecorder) {
	rec := &changeTapRecorder{}
	store := storage.NewStore(testNodeID, clock.NewClock(fixedNow(), 0, 0, testDriftWarn),
		storage.Config{OnChange: rec.tap})
	return store, rec
}

// TestStore_onChange covers the Config.OnChange tap (the phase-5 API-03
// seam): it fires exactly once per actual install for every change source —
// fresh and overwriting local Sets, a tombstone Delete, a winning remote
// Merge install and a Merge-delivered winning tombstone — and stays silent
// for a strictly-older losing merge and an equal-version redelivery (D-06:
// events fire only on actual state changes). The handed record is asserted
// field-by-field against the record a following Get or Records returns.
func TestStore_onChange(t *testing.T) {
	t.Parallel()

	t.Run("a fresh Set fires once with the stored record", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		ver, _, err := store.Set("k", []byte("v1"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		if got := tap.count(); got != 1 {
			t.Fatalf("tap fired %d times after one Set, want 1", got)
		}
		ev := tap.event(0)
		if ev.key != "k" || ev.record.Deleted {
			t.Fatalf("tap event = (%q, deleted=%t), want (k, deleted=false)", ev.key, ev.record.Deleted)
		}
		if ev.record.Version != ver {
			t.Fatalf("tap version = %x, want the Set's version %x", ev.record.Version[:], ver[:])
		}
		got, err := store.Get("k")
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
		if ev.record.Version != got.Version || ev.record.Checksum != got.Checksum ||
			!bytes.Equal(ev.record.Data, got.Data) {
			t.Fatalf("tap record = (%x, %q, sum %x), want the Get record (%x, %q, sum %x)",
				ev.record.Version[:], ev.record.Data, ev.record.Checksum,
				got.Version[:], got.Data, got.Checksum)
		}
	})

	t.Run("an overwrite Set fires for the second install", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		_, _, err := store.Set("k", []byte("v1"))
		if err != nil {
			t.Fatalf("first Set() error = %v, want nil", err)
		}
		ver2, _, err := store.Set("k", []byte("v2"))
		if err != nil {
			t.Fatalf("second Set() error = %v, want nil", err)
		}
		if got := tap.count(); got != 2 {
			t.Fatalf("tap fired %d times after two Sets, want 2", got)
		}
		ev := tap.event(1)
		if ev.key != "k" || ev.record.Deleted || ev.record.Version != ver2 ||
			!bytes.Equal(ev.record.Data, []byte("v2")) {
			t.Fatalf("second tap event = (%q, deleted=%t, %x, %q), want (k, false, the second version, v2)",
				ev.key, ev.record.Deleted, ev.record.Version[:], ev.record.Data)
		}
	})

	t.Run("a Delete fires with the tombstone install", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		_, _, err := store.Set("k", []byte("v1"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		ver, _, err := store.Delete("k")
		if err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
		if got := tap.count(); got != 2 {
			t.Fatalf("tap fired %d times after Set+Delete, want 2", got)
		}
		ev := tap.event(1)
		if ev.key != "k" || !ev.record.Deleted || ev.record.Version != ver {
			t.Fatalf("second tap event = (%q, deleted=%t, %x), want (k, deleted=true, the Delete version %x)",
				ev.key, ev.record.Deleted, ev.record.Version[:], ver[:])
		}
		stored, ok := store.Records()["k"]
		if !ok {
			t.Fatal("Records()[k] missing after Delete, want the stored tombstone")
		}
		if ev.record.Version != stored.Version || ev.record.Deleted != stored.Deleted ||
			ev.record.Checksum != stored.Checksum || !bytes.Equal(ev.record.Data, stored.Data) {
			t.Fatalf("tap record = (%x, deleted=%t, sum %x, %q), want the stored tombstone (%x, deleted=%t, sum %x, %q)",
				ev.record.Version[:], ev.record.Deleted, ev.record.Checksum, ev.record.Data,
				stored.Version[:], stored.Deleted, stored.Checksum, stored.Data)
		}
	})

	t.Run("a winning remote Merge install fires", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		stored := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		winner := storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeB), Data: []byte("b")}
		conflicts, _, err := store.Merge("k", winner)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: winner.Version, Loser: stored.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Fatalf("Merge() conflicts = %v, want %v", conflicts, want)
		}
		if got := tap.count(); got != 2 {
			t.Fatalf("tap fired %d times, want 2 (the seed install plus the winning install)", got)
		}
		ev := tap.event(1)
		got, err := store.Get("k")
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
		if ev.key != "k" || ev.record.Deleted || ev.record.Version != winner.Version ||
			ev.record.Checksum != got.Checksum || !bytes.Equal(ev.record.Data, got.Data) {
			t.Fatalf("second tap event = (%q, deleted=%t, %x, sum %x, %q), want the winning install matching Get (k, false, %x, sum %x, %q)",
				ev.key, ev.record.Deleted, ev.record.Version[:], ev.record.Checksum, ev.record.Data,
				got.Version[:], got.Checksum, got.Data)
		}
	})

	t.Run("a Merge-delivered winning tombstone fires with deleted=true", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		live := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", live)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		tomb := storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeB), Deleted: true}
		conflicts, _, err := store.Merge("k", tomb)
		if err != nil {
			t.Fatalf("Merge(tombstone) error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: tomb.Version, Loser: live.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Fatalf("Merge(tombstone) conflicts = %v, want %v", conflicts, want)
		}
		if got := tap.count(); got != 2 {
			t.Fatalf("tap fired %d times, want 2 (the seed install plus the tombstone install)", got)
		}
		ev := tap.event(1)
		if ev.key != "k" || !ev.record.Deleted || ev.record.Version != tomb.Version {
			t.Fatalf("second tap event = (%q, deleted=%t, %x), want (k, deleted=true, %x)",
				ev.key, ev.record.Deleted, ev.record.Version[:], tomb.Version[:])
		}
	})

	t.Run("a strictly older losing Merge fires nothing", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		stored := storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		loser := storage.Record{Version: version.PackVersion(mergeL, 0, nodeB), Data: []byte("b")}
		conflicts, _, err := store.Merge("k", loser)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		if len(conflicts) != 1 || conflicts[0].Winner != stored.Version {
			t.Fatalf("Merge() conflicts = %v, want the incoming version rejected as the loser", conflicts)
		}
		if got := tap.count(); got != 1 {
			t.Fatalf("tap fired %d times, want 1 (the seed install only — a loser never installs, D-06)", got)
		}
	})

	t.Run("an equal-version content win fires the tap once more", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		rec := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", rec)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		winner := storage.Record{Version: rec.Version, Data: []byte("x")}
		conflicts, _, err := store.Merge("k", winner)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		if len(conflicts) != 1 {
			t.Fatalf("Merge() conflicts = %v, want exactly one tie-break conflict", conflicts)
		}
		if got := tap.count(); got != 2 {
			t.Fatalf("tap fired %d times, want 2 — the seed install plus the tie-break install (D-06)", got)
		}
	})

	t.Run("an identical equal-version redelivery fires nothing", func(t *testing.T) {
		t.Parallel()
		store, tap := newTapStore()
		rec := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", rec)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		conflicts, _, err := store.Merge("k", rec)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		if conflicts != nil {
			t.Fatalf("Merge() conflicts = %v, want nil (identical content is a full no-op)", conflicts)
		}
		if got := tap.count(); got != 1 {
			t.Fatalf("tap fired %d times, want 1 (the seed install only — an identical redelivery never installs, D-06)", got)
		}
	})

	t.Run("a nil OnChange is a no-op", func(t *testing.T) {
		t.Parallel()
		store := newStore() // zero-value Config: OnChange normalizes to a no-op.
		_, _, err := store.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}
		_, _, err = store.Merge("k", storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeA), Data: []byte("b")})
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		_, _, err = store.Delete("k")
		if err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
		// Reaching here without a nil-callback panic is the assertion: the
		// zero-value Config behaves exactly like the pre-tap store.
	})
}

// requireGeneration fails unless store.Generation() equals want — the
// content-generation assertions' single spelling.
func requireGeneration(t *testing.T, store *storage.Store, want uint64) {
	t.Helper()
	if got := store.Generation(); got != want {
		t.Fatalf("Generation() = %d, want %d", got, want)
	}
}

// TestStore_observeAdvancesClock pins the version-only absorption seam
// (PROTO-01): Observe feeds the store clock exactly the received version and
// touches no record, so the very next local Set stamps a strictly greater
// version while Len and Lookup stay exactly as they were.
func TestStore_observeAdvancesClock(t *testing.T) {
	t.Parallel()
	store := newStore()
	_, _, err := store.Set("seat", []byte("seat"))
	if err != nil {
		t.Fatalf("seed Set() error = %v, want nil", err)
	}
	beforeLen := store.Len()
	absorbed := version.PackVersion(testNowMs+10_000, 3, nodeB)
	store.Observe(absorbed)
	if got := store.Len(); got != beforeLen {
		t.Fatalf("Len() after Observe = %d, want the unchanged %d", got, beforeLen)
	}
	if _, ok := store.Lookup("absorbed"); ok {
		t.Fatal("Lookup(absorbed) found a record, want none (Observe is clock-only)")
	}
	ver, _, err := store.Set("absorbed", []byte("v"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got := ver.Compare(absorbed); got != cmpNewer {
		t.Fatalf("next Set version %x does not strictly beat the absorbed %x", ver[:], absorbed[:])
	}
}

// TestStore_observeMonotone pins the absorption monotonicity: observing an
// older version after a newer one never lowers the clock, so the next local
// write still beats the newer reading (clock.Update's receive rule).
func TestStore_observeMonotone(t *testing.T) {
	t.Parallel()
	store := newStore()
	newer := version.PackVersion(testNowMs+10_000, 9, nodeB)
	older := version.PackVersion(testNowMs+5_000, 0, nodeB)
	store.Observe(newer)
	store.Observe(older)
	ver, _, err := store.Set("k", []byte("v"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got := ver.Compare(newer); got != cmpNewer {
		t.Fatalf("next Set version %x does not strictly beat the newer absorbed %x", ver[:], newer[:])
	}
}

// TestStore_observeReturnsDrift pins the drift return: the signal is the
// clock's own divergence from the wall clock at the event (the Kulkarni
// receive rule absorbs the remote reading first), exactly like Merge's
// return. An in-tolerance observation returns zero; a stamp beyond the
// drift-warn threshold drives a nonzero reading on the observation AFTER
// its absorption (the absorb-then-warn sequence TestMergeStats_counts pins).
func TestStore_observeReturnsDrift(t *testing.T) {
	t.Parallel()

	t.Run("an in-tolerance version returns zero drift", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("seat", []byte("seat"))
		if err != nil {
			t.Fatalf("seed Set() error = %v, want nil", err)
		}
		drift := store.Observe(version.PackVersion(testNowMs+1000, 0, nodeB))
		if drift != 0 {
			t.Fatalf("Observe() drift = %v, want 0 within tolerance", drift)
		}
	})

	t.Run("a version beyond the drift threshold drives a nonzero drift after its absorption", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("seat", []byte("seat"))
		if err != nil {
			t.Fatalf("seed Set() error = %v, want nil", err)
		}
		far := version.PackVersion(testNowMs+int64(testDriftWarn/time.Millisecond)+1000, 0, nodeB)
		if drift := store.Observe(far); drift != 0 {
			t.Fatalf("first Observe(far) drift = %v, want 0 (the stamp is absorbed, not yet warned)", drift)
		}
		if drift := store.Observe(far); drift <= 0 {
			t.Fatalf("second Observe(far) drift = %v, want a positive warning past the threshold", drift)
		}
	})
}

// TestStore_len covers the O(1) record count: a fresh store reads 0, every
// install path adds one — Set, a successful SetIfAbsent, a winning Merge and
// a Delete's tombstone — while a rejected SetIfAbsent, a missing-key Delete
// and an equal-version Merge change nothing. Tombstones stay counted: Len is
// the record count, not the live count, and it clones no Data (PROTO-02,
// T-4-11).
func TestStore_len(t *testing.T) {
	t.Parallel()
	store := newStore()
	if got := store.Len(); got != 0 {
		t.Fatalf("fresh store Len() = %d, want 0", got)
	}
	_, _, err := store.Set("live", []byte("v"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got := store.Len(); got != 1 {
		t.Fatalf("Len() after Set = %d, want 1", got)
	}
	_, _, err = store.SetIfAbsent("absent", []byte("v"))
	if err != nil {
		t.Fatalf("SetIfAbsent() error = %v, want nil", err)
	}
	if got := store.Len(); got != 2 {
		t.Fatalf("Len() after successful SetIfAbsent = %d, want 2", got)
	}
	_, _, err = store.SetIfAbsent("live", []byte("dup"))
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("SetIfAbsent(existing) error = %v, want ErrAlreadyExists", err)
	}
	if got := store.Len(); got != 2 {
		t.Fatalf("Len() after rejected SetIfAbsent = %d, want the unchanged 2", got)
	}
	_, _, err = store.Merge("merged", storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("m")})
	if err != nil {
		t.Fatalf("Merge() error = %v, want nil", err)
	}
	if got := store.Len(); got != 3 {
		t.Fatalf("Len() after Merge install = %d, want 3", got)
	}
	_, _, err = store.Merge("merged", storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("x")})
	if err != nil {
		t.Fatalf("equal Merge() error = %v, want nil", err)
	}
	if got := store.Len(); got != 3 {
		t.Fatalf("Len() after equal-version Merge = %d, want the unchanged 3", got)
	}
	_, _, err = store.Delete("live")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	if got := store.Len(); got != 3 {
		t.Fatalf("Len() after Delete = %d, want 3 (the tombstone stays a record)", got)
	}
	_, _, err = store.Delete("missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Delete(missing) error = %v, want ErrNotFound", err)
	}
	if got := store.Len(); got != 3 {
		t.Fatalf("Len() after missing-key Delete = %d, want the unchanged 3", got)
	}
}

// TestStore_generationContentOnly pins the content-generation contract
// (T-4-43): it moves by exactly one per record-changing install — Set, a
// successful SetIfAbsent, a successful Delete and every winning Merge all
// converge on install — and stays put for a rejected SetIfAbsent, a
// missing-key Delete, an equal-or-older Merge, the clock-only Observe and
// every read path. A call counter would move where this must not.
func TestStore_generationContentOnly(t *testing.T) {
	t.Parallel()
	store := newStore()
	requireGeneration(t, store, 0)

	_, _, err := store.Set("k", []byte("v"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	requireGeneration(t, store, 1)

	// Every read path leaves it unchanged.
	_, err = store.Get("k")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	_, _ = store.Version("k")
	_, _ = store.Lookup("k")
	_ = store.Versions()
	_ = store.Records()
	_ = store.Len()
	requireGeneration(t, store, 1)

	// A rejected SetIfAbsent installs nothing.
	_, _, err = store.SetIfAbsent("k", []byte("dup"))
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("SetIfAbsent(existing) error = %v, want ErrAlreadyExists", err)
	}
	requireGeneration(t, store, 1)

	// A successful SetIfAbsent is an install.
	_, _, err = store.SetIfAbsent("created", []byte("v"))
	if err != nil {
		t.Fatalf("SetIfAbsent() error = %v, want nil", err)
	}
	requireGeneration(t, store, 2)

	// Deleting a missing key burns no version and installs nothing.
	_, _, err = store.Delete("missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Delete(missing) error = %v, want ErrNotFound", err)
	}
	requireGeneration(t, store, 2)

	// A successful Delete installs its tombstone.
	_, _, err = store.Delete("k")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	requireGeneration(t, store, 3)

	// A merge into a missing key installs with no displacement.
	stored := storage.Record{Version: version.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
	_, _, err = store.Merge("merged", stored)
	if err != nil {
		t.Fatalf("Merge() error = %v, want nil", err)
	}
	requireGeneration(t, store, 4)

	// Equal and older merges are no-ops.
	_, _, err = store.Merge("merged", storage.Record{Version: stored.Version, Data: []byte("x")})
	if err != nil {
		t.Fatalf("equal Merge() error = %v, want nil", err)
	}
	requireGeneration(t, store, 4)
	_, _, err = store.Merge("merged", storage.Record{Version: version.PackVersion(mergeL-1, 0, nodeB), Data: []byte("old")})
	if err != nil {
		t.Fatalf("older Merge() error = %v, want nil", err)
	}
	requireGeneration(t, store, 4)

	// A strictly greater merge installs.
	_, _, err = store.Merge("merged", storage.Record{Version: version.PackVersion(mergeLNext, 0, nodeB), Data: []byte("b")})
	if err != nil {
		t.Fatalf("greater Merge() error = %v, want nil", err)
	}
	requireGeneration(t, store, 5)

	// Observe is clock-only: no record moves the generation.
	store.Observe(version.PackVersion(testNowMs+10_000, 0, nodeB))
	requireGeneration(t, store, 5)
}

// TestStorage_valueLimit pins the single-source value limit (PERSIST-01):
// the exported constant is the documented 61 440 default, Config.ValueLimit
// resolves both the zero case and an explicit configured limit, and a store
// built from a positive Config reports that same effective limit through its
// accessor. The literal in this package's constants is the pin; production
// derives every use from storage.ValueMaxBytes.
func TestStorage_valueLimit(t *testing.T) {
	t.Parallel()
	if got := storage.ValueMaxBytes; got != valueMax {
		t.Errorf("storage.ValueMaxBytes = %d, want %d", got, valueMax)
	}
	if got := (storage.Config{}).ValueLimit(); got != valueMax {
		t.Errorf("Config{}.ValueLimit() = %d, want %d", got, valueMax)
	}
	if got := (storage.Config{ValueMaxBytes: 8}).ValueLimit(); got != 8 {
		t.Errorf("Config{ValueMaxBytes: 8}.ValueLimit() = %d, want 8", got)
	}
	store := storage.NewStore(testNodeID, clock.NewClock(fixedNow(), 0, 0, testDriftWarn), storage.Config{ValueMaxBytes: 8})
	if got := store.ValueMaxBytes(); got != 8 {
		t.Errorf("NewStore(ValueMaxBytes: 8).ValueMaxBytes() = %d, want 8", got)
	}
}
