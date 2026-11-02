// Package testutil holds the property-test generators shared by the
// internal/app packages: the seeded random source behind the printed-seed
// and -seed replay contract (D-04), plus version, record, key and store
// generators. It is a normal package linked only into test binaries; the
// no-test-libs-in-prod lint rule exempts the testutil directory, which is
// what allows the testing import here.
package testutil

import (
	"flag"
	"math/rand/v2"
	"testing"
	"time"
)

// testNowMs is the fixed wall-clock millisecond the generators pin their
// stores' clocks to (the D-02 seam): arbitrary but shared, so generated
// stores and hand-built test fixtures agree.
const testNowMs = int64(1730000000000)

// maxNodeID is the 16-bit nodeID ceiling from RFC 9562.
const maxNodeID = 0xFFFF

// testCounterMax is the 12-bit rand_a counter ceiling from RFC 9562.
const testCounterMax = 4095

// seedHalfBits is where the int64 seed splits into the two 32-bit halves of
// the PCG generator.
const seedHalfBits = 32

// seedFlag replays a property test with a fixed seed: the D-04 contract
// behind NewRNG (a package-level test flag is the standard replay hook; the
// gochecknoglobals exclusion for test files does not reach this normal
// package, hence the explicit suppression).
var seedFlag = flag.Int64("seed", 0, "replay a property test with a fixed seed") //nolint:gochecknoglobals // test-only flag; package-level registration is the standard -seed replay hook (D-04)

// NewRNG returns a seeded generator for one property test. The seed derives
// from the wall clock unless -seed is non-zero, and is printed with the exact
// "property seed:" prefix on every run so a failing run replays exactly with
// the printed seed (D-04, research Pattern 3, Pitfall 5).
func NewRNG(t *testing.T) *rand.Rand {
	t.Helper()
	seedNano := *seedFlag
	if seedNano == 0 {
		seedNano = time.Now().UnixNano()
	}
	hi, lo := uint64(seedNano>>seedHalfBits), uint64(seedNano)&(1<<seedHalfBits-1)
	t.Logf("property seed: %d,%d (replay: go test ./... -run '%s' -args -seed=%d)",
		hi, lo, t.Name(), seedNano)
	return rand.New(rand.NewPCG(hi, lo)) //nolint:gosec // G404: deterministic seeded randomness is the D-04 contract, not a crypto secret
}
