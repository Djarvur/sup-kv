package storage_test

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// cloneStore returns a fresh store at the fixed wall clock holding exactly the
// records of src, installed key by key via Merge so every record keeps its own
// version: the laws drive stores exclusively through the exported Merge and
// Records surface (research Pattern 2, clock-free).
func cloneStore(t *testing.T, src *storage.Store) *storage.Store {
	t.Helper()
	dst := newStore()
	for key, rec := range src.Records() {
		_, _, err := dst.Merge(key, rec)
		if err != nil {
			t.Fatalf("clone Merge(%q) error = %v, want nil", key, err)
		}
	}
	return dst
}

// mergeAll applies every record of src into dst and returns the accumulated
// conflicts — the D-01 return-value contract the laws account for: the losers
// live in Merge's returned Conflicts, nothing accumulates inside the library.
func mergeAll(t *testing.T, dst, src *storage.Store) []storage.Conflict {
	t.Helper()
	var conflicts []storage.Conflict
	for key, rec := range src.Records() {
		got, _, err := dst.Merge(key, rec)
		if err != nil {
			t.Fatalf("Merge(%q) error = %v, want nil", key, err)
		}
		conflicts = append(conflicts, got...)
	}
	return conflicts
}

// recordsEqual reports whether two record maps hold the same keys with
// deep-equal records: version, deleted flag, checksum and data bytes.
func recordsEqual(got, want map[string]storage.Record) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantRec := range want {
		gotRec, ok := got[key]
		if !ok || gotRec.Version != wantRec.Version || gotRec.Deleted != wantRec.Deleted ||
			gotRec.Checksum != wantRec.Checksum || !bytes.Equal(gotRec.Data, wantRec.Data) {
			return false
		}
	}
	return true
}

// sortedConflicts returns the conflicts canonically ordered by key, so sets
// produced under nondeterministic map-iteration order compare element-wise;
// each key contributes at most one conflict per whole-store merge.
func sortedConflicts(conflicts []storage.Conflict) []storage.Conflict {
	sorted := slices.Clone(conflicts)
	slices.SortFunc(sorted, func(a, b storage.Conflict) int {
		return strings.Compare(a.Key, b.Key)
	})
	return sorted
}

// assertMergeLaws asserts the three LWW join laws over one generated store
// triple (CORE-06, TEST-01): commutativity — merging in either direction
// reaches the same state and reports the same conflict set; associativity —
// merging b then c into a equals merging a into (b merged with c); idempotence
// — re-applying a store's own records changes nothing and reports no
// conflicts.
func assertMergeLaws(t *testing.T, step int, a, b, c *storage.Store) {
	t.Helper()
	ab, ba := cloneStore(t, a), cloneStore(t, b)
	abConflicts := sortedConflicts(mergeAll(t, ab, b))
	baConflicts := sortedConflicts(mergeAll(t, ba, a))
	if !recordsEqual(ab.Records(), ba.Records()) {
		t.Fatalf("step %d commutativity: merge(a,b) state != merge(b,a) state (replay with the property seed above)", step)
	}
	if !conflictsEqual(abConflicts, baConflicts) {
		t.Fatalf("step %d commutativity: merge(a,b) conflicts %v != merge(b,a) conflicts %v (replay with the property seed above)",
			step, abConflicts, baConflicts)
	}
	left, right := cloneStore(t, a), cloneStore(t, b)
	mergeAll(t, left, b)
	mergeAll(t, left, c)
	mergeAll(t, right, c)
	mergeAll(t, right, a)
	if !recordsEqual(left.Records(), right.Records()) {
		t.Fatalf("step %d associativity: merge(merge(a,b),c) state != merge(a,merge(b,c)) state (replay with the property seed above)", step)
	}
	self := cloneStore(t, a)
	if selfConflicts := mergeAll(t, self, a); len(selfConflicts) != 0 {
		t.Fatalf("step %d idempotence: merge(a,a) conflicts = %v, want none (replay with the property seed above)",
			step, selfConflicts)
	}
	if !recordsEqual(self.Records(), a.Records()) {
		t.Fatalf("step %d idempotence: merge(a,a) changed the state (replay with the property seed above)", step)
	}
}

// randTombStore returns a store whose records are tombstones seven times out
// of eight — the mix inverted from RandStore — so the laws also run over
// deleted-dominated state.
func randTombStore(rng *rand.Rand) *storage.Store {
	store := newStoreFor(uint16(rng.IntN(maxNodeID + 1)))
	for range 1 + rng.IntN(16) {
		rec := testutil.RandRecord(rng)
		if rng.IntN(8) != 0 {
			rec.Deleted = true
			rec.Data = nil
		}
		_, _, _ = store.Merge(testutil.RandKey(rng), rec)
	}
	return store
}

// TestMerge_laws proves the CRDT join laws clock-free over 500 seeded random
// store triples with live and tombstoned records mixed: the per-key
// greater-version join is commutative, associative and idempotent, with
// identical conflict sets in both merge directions (TEST-01, CORE-06).
func TestMerge_laws(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	for step := range 500 {
		assertMergeLaws(t, step, testutil.RandStore(rng), testutil.RandStore(rng), testutil.RandStore(rng))
	}
}

// TestMerge_lawsTombstones proves the three laws over tombstone-heavy stores
// plus the no-resurrection rule: an older live record merged after a newer
// tombstone keeps the tombstone and reports the live record as the loser
// (TEST-01, CORE-06).
func TestMerge_lawsTombstones(t *testing.T) {
	t.Parallel()
	t.Run("the three laws hold over tombstone-heavy stores", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range 500 {
			assertMergeLaws(t, step, randTombStore(rng), randTombStore(rng), randTombStore(rng))
		}
	})
	t.Run("an older live record after a newer tombstone loses without resurrection", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		tomb := testutil.RandVersion(rng)
		live := testutil.RandVersion(rng)
		for tomb.Compare(live) == cmpSame {
			live = testutil.RandVersion(rng)
		}
		if tomb.Compare(live) == cmpOlder {
			tomb, live = live, tomb
		}
		store := newStore()
		tombRec := storage.Record{Version: tomb, Deleted: true}
		_, _, err := store.Merge("k", tombRec)
		if err != nil {
			t.Fatalf("seed Merge(tombstone) error = %v, want nil", err)
		}
		liveRec := storage.Record{Version: live, Data: []byte("stale write")}
		var conflicts []storage.Conflict
		conflicts, _, err = store.Merge("k", liveRec)
		if err != nil {
			t.Fatalf("Merge(live) error = %v, want nil", err)
		}
		want := []storage.Conflict{{Key: "k", Winner: tomb, Loser: live}}
		if !conflictsEqual(conflicts, want) {
			t.Errorf("Merge() conflicts = %v, want %v (replay with the property seed above)", conflicts, want)
		}
		rec, ok := store.Records()["k"]
		if !ok || !rec.Deleted || rec.Version != tomb {
			t.Errorf("Records()[k] = (%x, deleted=%t, ok=%t), want the tombstone %x (replay with the property seed above)",
				rec.Version[:], rec.Deleted, ok, tomb[:])
		}
		_, err = store.Get("k")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound (no resurrection)", err)
		}
	})
}

// TestMerge_duplicateDelivery proves a retransmitted record is a strict
// no-op: delivered twice into the same key, the second delivery returns nil
// conflicts and leaves Records() deep-equal — the conflict journal source
// cannot fill with noise (CORE-06, research Pitfall 3).
func TestMerge_duplicateDelivery(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	for step := range 200 {
		key := testutil.RandKey(rng)
		rec := testutil.RandRecord(rng)
		store := newStore()
		first, _, err := store.Merge(key, rec)
		if err != nil {
			t.Fatalf("step %d first Merge() error = %v, want nil", step, err)
		}
		if first != nil {
			t.Fatalf("step %d first Merge() conflicts = %v, want nil (install into a missing key)", step, first)
		}
		before := store.Records()
		second, _, err := store.Merge(key, rec)
		if err != nil {
			t.Fatalf("step %d second Merge() error = %v, want nil", step, err)
		}
		if second != nil {
			t.Fatalf("step %d second Merge() conflicts = %v, want nil (replay with the property seed above)", step, second)
		}
		if !recordsEqual(store.Records(), before) {
			t.Fatalf("step %d second Merge() changed Records() (replay with the property seed above)", step)
		}
	}
}

// TestMerge_losersProperty proves exact loser accounting over seeded random
// stored/incoming pairs: differing versions yield exactly one Conflict whose
// Winner is the greater full 16-byte version and Loser the lesser, equal
// versions yield none, and no Conflict ever pairs equal versions (CORE-07).
func TestMerge_losersProperty(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	for step := range 500 {
		key := testutil.RandKey(rng)
		stored, incoming := testutil.RandRecord(rng), testutil.RandRecord(rng)
		store := newStore()
		_, _, err := store.Merge(key, stored)
		if err != nil {
			t.Fatalf("step %d seed Merge() error = %v, want nil", step, err)
		}
		conflicts, _, err := store.Merge(key, incoming)
		if err != nil {
			t.Fatalf("step %d Merge() error = %v, want nil", step, err)
		}
		var want []storage.Conflict
		switch incoming.Version.Compare(stored.Version) {
		case cmpNewer:
			want = []storage.Conflict{{Key: key, Winner: incoming.Version, Loser: stored.Version}}
		case cmpOlder:
			want = []storage.Conflict{{Key: key, Winner: stored.Version, Loser: incoming.Version}}
		case cmpSame:
			want = nil
		}
		if !conflictsEqual(conflicts, want) {
			t.Fatalf("step %d Merge() conflicts = %v, want %v (replay with the property seed above)",
				step, conflicts, want)
		}
		for _, conflict := range conflicts {
			if conflict.Loser == conflict.Winner {
				t.Fatalf("step %d Conflict pairs equal versions %x (replay with the property seed above)",
					step, conflict.Loser[:])
			}
		}
	}
}
