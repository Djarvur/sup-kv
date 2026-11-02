package app_test

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app"
)

// seedFlag replays a property test with a fixed seed: the D-04 contract
// behind newRNG (package-level test flags are the standard replay hook; the
// gochecknoglobals exclusion for test files covers the var).
var seedFlag = flag.Int64("seed", 0, "replay a property test with a fixed seed")

// newRNG returns a seeded generator for one property test. The seed derives
// from the wall clock unless -seed is non-zero, and is printed with the exact
// "property seed:" prefix on every run so a failing run replays exactly with
// the printed seed (D-04, research Pattern 3, Pitfall 5).
func newRNG(t *testing.T) *rand.Rand {
	t.Helper()
	seedNano := *seedFlag
	if seedNano == 0 {
		seedNano = time.Now().UnixNano()
	}
	hi, lo := uint64(seedNano>>32), uint64(seedNano&0xFFFFFFFF)
	t.Logf("property seed: %d,%d (replay: go test ./internal/app/ -run '%s' -args -seed=%d)",
		hi, lo, t.Name(), seedNano)
	return rand.New(rand.NewPCG(hi, lo))
}

// nextWallTime advances the adversarial wall-clock walk: one in ten a
// backward jump of up to four hours in milliseconds, one in ten a forward
// jump of the same scale, two in ten stuck at the same millisecond, else a
// 0..2 ms step (research "Code Examples", the TEST-02 chaos input).
//
//nolint:unused // consumed by hlc_prop_test.go (task 3); the suppression is removed when its call sites land.
func nextWallTime(rng *rand.Rand, prev int64) int64 {
	switch rng.IntN(10) {
	case 0:
		return prev - rng.Int64N(4*60*60*1000)
	case 1:
		return prev + rng.Int64N(4*60*60*1000)
	case 2, 3:
		return prev
	default:
		return prev + rng.Int64N(3)
	}
}

// randVersion packs a valid version from a random in-range HLC reading:
// l anywhere in the 48-bit domain, counter 0..4095, nodeID 0..0xFFFF.
func randVersion(rng *rand.Rand) app.RecordVersion {
	return app.PackVersion(
		rng.Int64N(1<<48),
		uint16(rng.IntN(testCounterMax+1)),
		uint16(rng.IntN(maxNodeID+1)),
	)
}

// randRawVersion fills all 16 bytes with random bits: an arbitrary bit
// pattern off the wire, the trust-boundary input of Unpack and Compare.
func randRawVersion(rng *rand.Rand) app.RecordVersion {
	var v app.RecordVersion
	for i := range v {
		v[i] = byte(rng.IntN(256))
	}
	return v
}

// randRecord returns a record carrying a random version: roughly one third
// tombstones with nil Data, the rest up to 63 random data bytes.
//
//nolint:unused // consumed by hlc_prop_test.go (task 3) and the merge laws (plan 02-03); removed when those land.
func randRecord(rng *rand.Rand) app.Record {
	if rng.IntN(3) == 0 {
		return app.Record{Version: randVersion(rng), Deleted: true}
	}
	data := make([]byte, rng.IntN(64))
	for i := range data {
		data[i] = byte(rng.IntN(256))
	}
	return app.Record{Version: randVersion(rng), Data: data}
}

// randKey returns a random key from a small space so records collide.
//
//nolint:unused // consumed via randStore by hlc_prop_test.go (task 3); removed when its call sites land.
func randKey(rng *rand.Rand) string {
	return fmt.Sprintf("key-%d", rng.IntN(32))
}

// randStore returns a store pre-populated by Merge with randRecord records
// over random keys: Merge installs externally versioned records, which is
// how tests construct stores with arbitrary versions (research Pattern 2).
//
//nolint:unused // consumed by hlc_prop_test.go (task 3) and the merge laws (plan 02-03); removed when those land.
func randStore(rng *rand.Rand) *app.Store {
	store := app.NewStore(uint16(rng.IntN(maxNodeID+1)), app.Config{}, fixedNow(testNowMs))
	for range 1 + rng.IntN(16) {
		_, _, _ = store.Merge(randKey(rng), randRecord(rng))
	}
	return store
}
