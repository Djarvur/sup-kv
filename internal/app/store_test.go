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
