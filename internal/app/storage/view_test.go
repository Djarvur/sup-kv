package storage_test

// The store read-view tests: the executable contract of Version, Lookup and
// Versions (XPORT-11). The view exists because reconciliation needs tombstones
// and a whole-store Records() deep copy per datagram is too expensive; these
// tests pin both properties — tombstone visibility and caller-owned copies —
// plus the concurrency safety of the read path under the race detector.

import (
	"bytes"
	"maps"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// TestStore_version proves the per-key version probe: absent keys report
// false, a live write reports its stamped version, a tombstone stays
// visible, and the reported version matches the Records() entry (the
// tombstone-inclusive snapshot the view is a cheaper alternative to).
func TestStore_version(t *testing.T) {
	t.Parallel()
	store := newStore()
	if _, ok := store.Version("missing"); ok {
		t.Fatal("Version(missing) ok = true, want false")
	}

	live, _, err := store.Set("k", []byte("value"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	got, ok := store.Version("k")
	if !ok {
		t.Fatal("Version(k) ok = false after Set, want true")
	}
	if got != live {
		t.Fatalf("Version(k) = %x, want the written version %x", got[:], live[:])
	}
	if rec := store.Records()["k"]; rec.Version != got {
		t.Fatalf("Version(k) = %x, want the Records() version %x", got[:], rec.Version[:])
	}

	tombstone, _, err := store.Delete("k")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	got, ok = store.Version("k")
	if !ok {
		t.Fatal("Version(k) ok = false for a tombstone, want true")
	}
	if got != tombstone {
		t.Fatalf("Version(k) = %x, want the tombstone version %x", got[:], tombstone[:])
	}
	if got.Compare(live) <= 0 {
		t.Fatalf("tombstone version %x does not dominate the live version %x", got[:], live[:])
	}
}

// TestStore_lookup_tombstoneVisible proves Lookup is the tombstone-visible
// point read Get deliberately is not: a live record equals its Records()
// entry, a tombstone returns Deleted=true with no data, a missing key
// reports false, and mutating the returned Data never reaches the store.
func TestStore_lookup_tombstoneVisible(t *testing.T) {
	t.Parallel()
	store := newStore()
	if _, ok := store.Lookup("missing"); ok {
		t.Fatal("Lookup(missing) ok = true, want false")
	}

	liveVer, _, err := store.Set("k", []byte("value"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	rec, ok := store.Lookup("k")
	if !ok {
		t.Fatal("Lookup(k) ok = false after Set, want true")
	}
	want := store.Records()["k"]
	if rec.Deleted {
		t.Error("Lookup(k).Deleted = true for a live record, want false")
	}
	if rec.Version != want.Version || rec.Checksum != want.Checksum || !bytes.Equal(rec.Data, want.Data) {
		t.Errorf("Lookup(k) = (%x, %q, %#x), want the Records() entry (%x, %q, %#x)",
			rec.Version[:], rec.Data, rec.Checksum, want.Version[:], want.Data, want.Checksum)
	}

	// Copy isolation: the caller owns the returned bytes.
	rec.Data[0] = 'X'
	again, ok := store.Lookup("k")
	if !ok {
		t.Fatal("second Lookup(k) ok = false, want true")
	}
	if !bytes.Equal(again.Data, []byte("value")) {
		t.Fatalf("Lookup(k) after mutating the returned Data = %q, want %q", again.Data, "value")
	}
	got, err := store.Get("k")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if !bytes.Equal(got.Data, []byte("value")) {
		t.Fatalf("Get(k) after mutating the Lookup Data = %q, want %q", got.Data, "value")
	}

	// Tombstone visibility: Lookup returns it, Get stays live-only.
	tombstoneVer, _, err := store.Delete("k")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	rec, ok = store.Lookup("k")
	if !ok {
		t.Fatal("Lookup(k) ok = false for a tombstone, want true")
	}
	if !rec.Deleted {
		t.Error("Lookup(k).Deleted = false for a tombstone, want true")
	}
	if rec.Data != nil {
		t.Errorf("Lookup(k).Data = %q for a tombstone, want nil", rec.Data)
	}
	if rec.Version != tombstoneVer {
		t.Errorf("Lookup(k) version = %x, want %x", rec.Version[:], tombstoneVer[:])
	}
	_, err = store.Get("k")
	if err == nil {
		t.Fatal("Get(k) error = nil for a tombstone, want ErrNotFound")
	}
	if gotVer, found := store.Version("k"); !found || gotVer != tombstoneVer {
		t.Errorf("Version(k) = (%x, %t), want (%x, true)", gotVer[:], found, tombstoneVer[:])
	}
	if liveVer == tombstoneVer {
		t.Fatal("the tombstone reused the live version")
	}
}

// TestStore_versions_snapshot proves the key→version snapshot: tombstones
// are included, the returned map is caller-owned, repeated calls with no
// intervening write are equal, and a merge of a newer record changes exactly
// that key's entry.
func TestStore_versions_snapshot(t *testing.T) {
	t.Parallel()
	store := newStore()
	if got := store.Versions(); len(got) != 0 {
		t.Fatalf("Versions() = %v on a fresh store, want an empty map", got)
	}
	alphaVer, _, err := store.Set("alpha", []byte("a"))
	if err != nil {
		t.Fatalf("Set(alpha) error = %v, want nil", err)
	}
	betaVer, _, err := store.Set("beta", []byte("b"))
	if err != nil {
		t.Fatalf("Set(beta) error = %v, want nil", err)
	}
	betaTombstone, _, err := store.Delete("beta")
	if err != nil {
		t.Fatalf("Delete(beta) error = %v, want nil", err)
	}

	before := store.Versions()
	if len(before) != 2 {
		t.Fatalf("Versions() size = %d, want both keys including the tombstone", len(before))
	}
	if got := before["alpha"]; got != alphaVer {
		t.Errorf("Versions()[alpha] = %x, want %x", got[:], alphaVer[:])
	}
	if got := before["beta"]; got != betaTombstone {
		t.Errorf("Versions()[beta] = %x, want the tombstone version %x", got[:], betaTombstone[:])
	}
	if got := before["beta"]; got == betaVer {
		t.Error("Versions()[beta] still reports the pre-delete version")
	}

	// Repeated calls with no intervening write are equal.
	if again := store.Versions(); !maps.Equal(before, again) {
		t.Errorf("repeated Versions() = %v, want the equal map %v", again, before)
	}

	// The map is caller-owned: mutating it never reaches the store.
	mutated := maps.Clone(before)
	mutated["alpha"] = version.RecordVersion{}
	delete(mutated, "beta")
	mutated["intruder"] = alphaVer
	if got := store.Versions(); !maps.Equal(got, before) {
		t.Errorf("Versions() after mutating a returned map = %v, want %v", got, before)
	}

	// A merge of a strictly newer record changes exactly that key.
	newer := storage.Record{
		Version: version.PackVersion(testNowMs+1000, 0, testNodeID),
		Data:    []byte("a-newer"),
	}
	_, _, err = store.Merge("alpha", newer)
	if err != nil {
		t.Fatalf("Merge(alpha) error = %v, want nil", err)
	}
	after := store.Versions()
	if len(after) != len(before) {
		t.Fatalf("Versions() size after merge = %d, want %d", len(after), len(before))
	}
	for key, ver := range before {
		if key == "alpha" {
			continue
		}
		if got := after[key]; got != ver {
			t.Errorf("Versions()[%s] = %x after the alpha merge, want the unchanged %x", key, got[:], ver[:])
		}
	}
	if got := after["alpha"]; got != newer.Version {
		t.Errorf("Versions()[alpha] = %x, want the merged version %x", got[:], newer.Version[:])
	}
}

// TestStore_liveVersions proves the tombstone-free key→version view the List
// pagination mode reads (D-02, APIS-02): live keys only — an empty live value
// included, because liveness is the Deleted flag and never len(Data) — with a
// caller-owned map and stable repeated calls.
func TestStore_liveVersions(t *testing.T) {
	t.Parallel()
	store := newStore()
	if got := store.LiveVersions(); len(got) != 0 {
		t.Fatalf("LiveVersions() = %v on a fresh store, want an empty map", got)
	}
	alphaVer, _, err := store.Set("alpha", []byte("a"))
	if err != nil {
		t.Fatalf("Set(alpha) error = %v, want nil", err)
	}
	emptyVer, _, err := store.Set("empty", nil)
	if err != nil {
		t.Fatalf("Set(empty) error = %v, want nil", err)
	}
	_, _, err = store.Set("beta", []byte("b"))
	if err != nil {
		t.Fatalf("Set(beta) error = %v, want nil", err)
	}
	_, _, err = store.Delete("beta")
	if err != nil {
		t.Fatalf("Delete(beta) error = %v, want nil", err)
	}

	live := store.LiveVersions()
	if len(live) != 2 {
		t.Fatalf("LiveVersions() size = %d, want the two live keys (the tombstone is excluded)", len(live))
	}
	if got := live["alpha"]; got != alphaVer {
		t.Errorf("LiveVersions()[alpha] = %x, want %x", got[:], alphaVer[:])
	}
	if got := live["empty"]; got != emptyVer {
		t.Errorf("LiveVersions()[empty] = %x, want the empty-valued live record's version %x", got[:], emptyVer[:])
	}
	if _, ok := live["beta"]; ok {
		t.Error("LiveVersions() includes the tombstoned key beta")
	}
	if _, ok := live["missing"]; ok {
		t.Error("LiveVersions() includes the absent key missing")
	}

	// The map is caller-owned: mutating it never reaches the store.
	mutated := maps.Clone(live)
	mutated["alpha"] = version.RecordVersion{}
	delete(mutated, "empty")
	mutated["intruder"] = alphaVer
	if again := store.LiveVersions(); !maps.Equal(again, live) {
		t.Errorf("LiveVersions() after mutating a returned map = %v, want %v", again, live)
	}

	// Repeated calls with no intervening write are equal.
	if again := store.LiveVersions(); !maps.Equal(again, live) {
		t.Errorf("repeated LiveVersions() = %v, want the equal map %v", again, live)
	}
}

// TestStore_view_concurrentHammer races the write surface (Set, SetIfAbsent,
// Delete, Merge) against the read view (Version, Lookup, Versions) across
// goroutines under the race detector. Only per-call validity is asserted —
// a returned record's checksum must match its own contents — never an
// ordering between two observations: the seeded keys never leave the map
// (deletes leave tombstones), so presence and checksum integrity are the
// invariants that must hold at every instant.
func TestStore_view_concurrentHammer(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	keys := []string{"view-0", "view-1", "view-2", "view-3"}
	store := storage.NewStore(testNodeID, clock.NewClock(atomicNow(testNowMs), 0, 0, testDriftWarn), storage.Config{})
	for _, key := range keys {
		_, _, err := store.Set(key, []byte("seed"))
		if err != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, err)
		}
	}

	checkRecord := func(key string, rec storage.Record) {
		if got := storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted); got != rec.Checksum {
			t.Errorf("Lookup(%q) record checksum = %#x, want %#x (torn record)", key, rec.Checksum, got)
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
				default:
					_, _, _ = store.Merge(key, storage.Record{
						Version: version.PackVersion(
							testNowMs+workerRng.Int64N(4001),
							uint16(workerRng.IntN(testCounterMax+1)),
							uint16(worker)),
						Data: payload,
					})
				}
			}
		})
	}
	for reader := range 2 {
		wg.Go(func() {
			readerRng := rand.New(rand.NewPCG(hammerSeed, uint64(reader)))
			for range 200 {
				key := keys[readerRng.IntN(len(keys))]
				switch readerRng.IntN(3) {
				case 0:
					if _, ok := store.Version(key); !ok {
						t.Errorf("Version(%q) ok = false, want true for a seeded key", key)
					}
				case 1:
					rec, ok := store.Lookup(key)
					if !ok {
						t.Errorf("Lookup(%q) ok = false, want true for a seeded key", key)
						continue
					}
					checkRecord(key, rec)
				default:
					versions := store.Versions()
					for _, k := range keys {
						if _, ok := versions[k]; !ok {
							t.Errorf("Versions() missing seeded key %q", k)
						}
					}
				}
			}
		})
	}
	wg.Wait()

	records := store.Records()
	for _, key := range keys {
		want, ok := records[key]
		if !ok {
			t.Errorf("key %q missing from Records() after the hammer", key)
			continue
		}
		rec, ok := store.Lookup(key)
		if !ok {
			t.Errorf("Lookup(%q) ok = false after the hammer, want true", key)
			continue
		}
		if rec.Version != want.Version || rec.Deleted != want.Deleted || !bytes.Equal(rec.Data, want.Data) {
			t.Errorf("Lookup(%q) = (%x, %t, %q), want the Records() entry (%x, %t, %q)",
				key, rec.Version[:], rec.Deleted, rec.Data, want.Version[:], want.Deleted, want.Data)
		}
		if ver, found := store.Version(key); !found || ver != want.Version {
			t.Errorf("Version(%q) = (%x, %t), want (%x, true)", key, ver[:], found, want.Version[:])
		}
	}
}
