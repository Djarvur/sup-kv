package app_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app"
)

// Shared test constants: a fixed wall-clock millisecond (the D-02 seam), the
// test node ID, the default value limit from docs/IDEA.md parameters, the
// UUIDv7 marker bits, and Compare result codes.
const (
	testNowMs   = int64(1730000000000)
	testNodeID  = uint16(0xABCD)
	valueMax    = 61440
	valueOver   = valueMax + 1
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

// fixedNow returns a now func parked at the given Unix millisecond, built
// with time.UnixMilli so no monotonic reading is involved (D-02).
func fixedNow(ms int64) func() time.Time {
	return func() time.Time { return time.UnixMilli(ms) }
}

// newStore returns a store on the test node with the default config at the
// fixed wall clock.
func newStore() *app.Store {
	return app.NewStore(testNodeID, app.Config{}, fixedNow(testNowMs))
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
		if !errors.Is(err, app.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})
	t.Run("set over the size limit is rejected and stores nothing", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", make([]byte, valueOver))
		if !errors.Is(err, app.ErrValueTooLarge) {
			t.Errorf("Set() error = %v, want ErrValueTooLarge", err)
		}
		_, err = store.Get("k")
		if !errors.Is(err, app.ErrNotFound) {
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
func conflictsEqual(got, want []app.Conflict) bool {
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
		if !errors.Is(err, app.ErrNotFound) {
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
		if !errors.Is(err, app.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound (tombstone invisible)", err)
		}
	})
}

// TestStore_Set covers the Set contract: unconditional LWW replace with
// strictly increasing versions, same-millisecond disambiguation by the
// counter, zero-length values, and fresh versions on repeat writes
// (CORE-02).
func TestStore_Set(t *testing.T) {
	t.Parallel()
	t.Run("a second set replaces the value under a strictly greater version", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		_, _, err := store.Set("k", []byte("one"))
		if err != nil {
			t.Fatalf("first Set() error = %v, want nil", err)
		}
		second, _, err := store.Set("k", []byte("two"))
		if err != nil {
			t.Fatalf("second Set() error = %v, want nil", err)
		}
		rec, err := store.Get("k")
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
		if got := second.Compare(rec.Version); got != cmpSame || !bytes.Equal(rec.Data, []byte("two")) {
			t.Errorf("stored = (%x, %q), want the second write (%x, %q)",
				rec.Version[:], rec.Data, second[:], "two")
		}
	})
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
		if !errors.Is(err, app.ErrAlreadyExists) {
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
		if !errors.Is(err, app.ErrAlreadyExists) {
			t.Errorf("SetIfAbsent() error = %v, want ErrAlreadyExists (tombstone counts as existing)", err)
		}
		_, err = store.Get("k")
		if !errors.Is(err, app.ErrNotFound) {
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
		if !errors.Is(err, app.ErrNotFound) {
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
		if !errors.Is(err, app.ErrNotFound) {
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
		if !errors.Is(err, app.ErrNotFound) {
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
		{name: "set rejects one byte over the limit", op: "set", size: valueOver, wantErr: app.ErrValueTooLarge},
		{name: "setifabsent accepts exactly the limit", op: "setifabsent", size: valueMax},
		{name: "setifabsent rejects one byte over the limit", op: "setifabsent", size: valueOver, wantErr: app.ErrValueTooLarge},
		{name: "merge accepts exactly the limit", op: "merge", size: valueMax},
		{name: "merge rejects one byte over the limit", op: "merge", size: valueOver, wantErr: app.ErrValueTooLarge},
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
				rec := app.Record{Version: app.PackVersion(mergeL, 0, nodeA), Data: make([]byte, tt.size)}
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
		stored := app.Record{Version: app.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		incoming := app.Record{Version: app.PackVersion(mergeLNext, 0, nodeB), Data: []byte("b")}
		conflicts, _, err := store.Merge("k", incoming)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []app.Conflict{{Key: "k", Winner: incoming.Version, Loser: stored.Version}}
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
		stored := app.Record{Version: app.PackVersion(mergeLNext, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		incoming := app.Record{Version: app.PackVersion(mergeL, 0, nodeB), Data: []byte("b")}
		if got := incoming.Version.Compare(stored.Version); got != cmpOlder {
			t.Fatalf("fixture Compare() = %d, want %d", got, cmpOlder)
		}
		conflicts, _, err := store.Merge("k", incoming)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []app.Conflict{{Key: "k", Winner: stored.Version, Loser: incoming.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v", conflicts, want)
		}
		rec, err := store.Get("k")
		if err != nil || !bytes.Equal(rec.Data, []byte("a")) {
			t.Errorf("Get() = (%q, %v), want the stored value (%q, nil)", rec.Data, err, "a")
		}
	})
	t.Run("equal version is a no-op with no conflicts", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		stored := app.Record{Version: app.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", stored)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		dup := app.Record{Version: stored.Version, Data: []byte("x")}
		conflicts, _, err := store.Merge("k", dup)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		if conflicts != nil {
			t.Errorf("Merge() conflicts = %v, want nil (duplicate delivery is silent)", conflicts)
		}
		rec, err := store.Get("k")
		if err != nil || rec.Version != stored.Version || !bytes.Equal(rec.Data, []byte("a")) {
			t.Errorf("Get() = (%x, %q, %v), want the stored record unchanged", rec.Version[:], rec.Data, err)
		}
	})
	t.Run("merging into a missing key installs with no conflicts", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		incoming := app.Record{Version: app.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
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
		live := app.Record{Version: app.PackVersion(mergeL, 0, nodeA), Data: []byte("a")}
		_, _, err := store.Merge("k", live)
		if err != nil {
			t.Fatalf("seed Merge() error = %v, want nil", err)
		}
		tomb := app.Record{Version: app.PackVersion(mergeLNext, 0, nodeB), Deleted: true}
		conflicts, _, err := store.Merge("k", tomb)
		if err != nil {
			t.Fatalf("Merge() error = %v, want nil", err)
		}
		want := []app.Conflict{{Key: "k", Winner: tomb.Version, Loser: live.Version}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v", conflicts, want)
		}
		_, err = store.Get("k")
		if !errors.Is(err, app.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound (no resurrection)", err)
		}
		if rec, ok := store.Records()["k"]; !ok || !rec.Deleted {
			t.Errorf("Records()[k] = (%x, deleted=%t, ok=%t), want a present tombstone",
				rec.Version[:], rec.Deleted, ok)
		}
	})
}
