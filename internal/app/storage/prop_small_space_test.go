package storage_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// TEST-15's dependency-free shrinking mechanism: a small (l, c, node) space
// half the record versions are drawn from, so counterexamples reduce to short
// adjacent-version sequences that read directly off the printed seed (D-08;
// no property-testing dependency is added). The space is three consecutive l
// values with counters and node IDs 0..2 / 1..3, which puts equal-l,
// equal-l-equal-c-different-node and adjacent-l collisions in reach by
// construction. The replay constants pin the exact command the executor test
// runs; the timeout covers a cold `go test` build of this package.
const (
	smallSpaceLBase      = testNowMs
	smallSpaceLRange     = 3
	smallSpaceCRange     = 3
	smallSpaceNodeRange  = 3
	smallSpaceKeyRange   = 3
	smallSpaceRecords    = 8
	smallSpaceSteps      = 300
	smallSpaceCollisions = 100
	replayTest           = "TestProp_mergeSmallSpace"
	replaySeed           = int64(20261121)
	replayTimeout        = 5 * time.Minute
)

// smallSpaceVersion returns a version drawn from the tiny space: l in
// {base, base+1, base+2}, counter 0..2, node 1..3 — adjacent l values and
// equal-l collisions appear by construction.
func smallSpaceVersion(rng *rand.Rand) version.RecordVersion {
	return version.PackVersion(
		smallSpaceLBase+int64(rng.IntN(smallSpaceLRange)),
		uint16(rng.IntN(smallSpaceCRange)),
		uint16(1+rng.IntN(smallSpaceNodeRange)),
	)
}

// mixedVersion draws from the small space half the time and wide the other
// half: TEST-15 fixes the distribution, and the wide half keeps the volume
// of the property unchanged.
func mixedVersion(rng *rand.Rand) version.RecordVersion {
	if rng.IntN(2) == 0 {
		return smallSpaceVersion(rng)
	}
	return testutil.RandVersion(rng)
}

// recordForVersion derives the whole record from its version: the payload is
// the version bytes and the tombstone flag a function of the first byte, so
// two records with equal versions are byte-identical. That is the premise of
// the random draws: a version is unique per production write, so state two
// different writes could not produce must not enter the property's random
// half — the deliberate different-payload collision subtest is where the
// equal-version content law is exercised.
func recordForVersion(ver version.RecordVersion) storage.Record {
	if ver[0]%3 == 0 {
		return storage.Record{Version: ver, Deleted: true}
	}
	return storage.Record{Version: ver, Data: ver[:]}
}

// randSmallSpaceStore returns a store of small-space-mixed records over a
// three-key space, so versions and keys collide constantly.
func randSmallSpaceStore(rng *rand.Rand) *storage.Store {
	store := newStoreFor(uint16(1 + rng.IntN(smallSpaceNodeRange)))
	for range 1 + rng.IntN(smallSpaceRecords) {
		key := fmt.Sprintf("small-%d", rng.IntN(smallSpaceKeyRange))
		_, _, _ = store.Merge(key, recordForVersion(mixedVersion(rng)))
	}
	return store
}

// pointwiseSupremum returns the per-key record of greatest version across the
// stores — the LWW join's closed-form expectation (equal versions are
// identical records by construction, so the choice among them is immaterial).
func pointwiseSupremum(stores ...*storage.Store) map[string]storage.Record {
	supremum := make(map[string]storage.Record)
	for _, store := range stores {
		for key, rec := range store.Records() {
			best, ok := supremum[key]
			if !ok || rec.Version.Compare(best.Version) > 0 {
				supremum[key] = rec
			}
		}
	}
	return supremum
}

// TestProp_mergeSmallSpace is TEST-15's small-space property: the CRDT laws
// run over store triples drawn half from the small (l, c, node) space, the
// merged state must be the pointwise supremum of the drawn versions, and an
// equal-version collision with different content resolves deterministically by
// the (version, content) law. Every run prints the
// seed and the package-scoped replay command (testutil.NewRNG).
func TestProp_mergeSmallSpace(t *testing.T) {
	t.Parallel()
	t.Run("the three laws hold over small-space-drawn triples", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range smallSpaceSteps {
			assertMergeLaws(t, step, randSmallSpaceStore(rng), randSmallSpaceStore(rng), randSmallSpaceStore(rng))
		}
	})
	t.Run("the merged state is the pointwise supremum of the drawn versions", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range smallSpaceSteps {
			a, b, c := randSmallSpaceStore(rng), randSmallSpaceStore(rng), randSmallSpaceStore(rng)
			merged := cloneStore(t, a)
			mergeAll(t, merged, b)
			mergeAll(t, merged, c)
			if want := pointwiseSupremum(a, b, c); !recordsEqual(merged.Records(), want) {
				t.Fatalf("step %d merged state is not the pointwise supremum of the three stores (replay with the property seed above)", step)
			}
		}
	})
	t.Run("an equal-version collision resolves by the content law", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range smallSpaceCollisions {
			ver := smallSpaceVersion(rng)
			stored := recordForVersion(ver)
			// Equal version, deliberately different payload: the deterministic
			// (version, content) law decides — remove-wins first, then the
			// greater Data bytes — and the same delivery twice changes nothing.
			incoming := storage.Record{Version: ver, Data: []byte("different payload")}
			store := newStore()
			_, _, err := store.Merge("collision", stored)
			if err != nil {
				t.Fatalf("step %d seed Merge() error = %v, want nil", step, err)
			}
			conflicts, _, err := store.Merge("collision", incoming)
			if err != nil {
				t.Fatalf("step %d collision Merge() error = %v, want nil", step, err)
			}
			if want := equalVersionConflicts("collision", stored, incoming); !conflictsEqual(conflicts, want) {
				t.Fatalf("step %d equal-version Merge() conflicts = %v, want %v (replay with the property seed above)",
					step, conflicts, want)
			}
			want := contentWinner(stored, incoming)
			rec, ok := store.Records()["collision"]
			if !ok || rec.Version != want.Version || rec.Deleted != want.Deleted ||
				!bytes.Equal(rec.Data, want.Data) {
				t.Fatalf("step %d equal-version Merge() stored (%x, deleted=%t, ok=%t), want the content winner (%x, deleted=%t) (replay with the property seed above)",
					step, rec.Version[:], rec.Deleted, ok, want.Version[:], want.Deleted)
			}
			before := store.Records()
			_, _, err = store.Merge("collision", incoming)
			if err != nil {
				t.Fatalf("step %d redelivery Merge() error = %v, want nil", step, err)
			}
			if !recordsEqual(store.Records(), before) {
				t.Fatalf("step %d equal-version redelivery changed the merged state (replay with the property seed above)", step)
			}
		}
	})
}

// TestProp_replayCommandExecutes proves TEST-15's second half: the printed
// replay command works. It asks testutil for the exact string NewRNG prints
// for TestProp_mergeSmallSpace with a pinned seed, asserts the command is
// package-scoped, then runs it through the shell — exit 0 plus the re-run and
// pass of that very test are the acceptance. The command comes from the
// shared format helper, never from parsing log text (T-2-18).
func TestProp_replayCommandExecutes(t *testing.T) {
	t.Parallel()
	command := testutil.ReplayCommand(t, replayTest, replaySeed)
	if strings.Contains(command, "./...") {
		t.Fatalf("replay command %q is not package-scoped", command)
	}
	if !strings.Contains(command, "github.com/Djarvur/sup-kv/internal/app/storage") {
		t.Fatalf("replay command %q does not target this package", command)
	}
	ctx, cancel := context.WithTimeout(t.Context(), replayTimeout)
	t.Cleanup(cancel)
	out, err := exec.CommandContext(ctx, "sh", "-c", command).CombinedOutput()
	if err != nil {
		t.Fatalf("replay command %q failed: %v\n%s", command, err, out)
	}
	if !strings.Contains(string(out), "=== RUN   "+replayTest) {
		t.Fatalf("replay output does not re-run %s:\n%s", replayTest, out)
	}
	if !strings.Contains(string(out), "--- PASS: "+replayTest) {
		t.Fatalf("replay output does not pass %s:\n%s", replayTest, out)
	}
}
