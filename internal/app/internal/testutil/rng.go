// Package testutil holds the property-test generators shared by the
// internal/app packages: the seeded random source behind the printed-seed
// and -seed replay contract (D-04), plus version, record, key and store
// generators. It is a normal package linked only into test binaries; the
// no-test-libs-in-prod lint rule exempts the testutil directory, which is
// what allows the testing import here.
package testutil

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"path"
	"regexp"
	"runtime"
	"strings"
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

// modulePath is this module's import path. The replay hint needs it to scope
// `go test` to the calling package: the earlier `./...` form failed in every
// package whose test binary does not register the -seed flag (flag parsing
// aborts with "flag provided but not defined: -seed").
const modulePath = "github.com/Djarvur/sup-kv"

// appAnchor is the path segment every testutil caller sits under: Go's
// internal rule confines the importers to internal/app/..., so a caller's
// source file resolves to its import path by the tail after this segment.
const appAnchor = "/internal/app/"

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
	t.Logf("property seed: %d,%d (replay: %s)", hi, lo, replayFor(t.Name(), seedNano))
	return rand.New(rand.NewPCG(hi, lo)) //nolint:gosec // G404: deterministic seeded randomness is the D-04 contract, not a crypto secret
}

// ReplayCommand returns the exact replay command NewRNG prints for testName
// with seed. It is exported so the replay-executor test runs the printed form
// rather than a lookalike reconstructed elsewhere: the package path here is
// the caller's, so the command targets the calling package and nothing else
// (T-2-18 — fixed parts and a numeric seed, never parsed log text).
func ReplayCommand(t *testing.T, testName string, seed int64) string {
	t.Helper()
	return replayFor(testName, seed)
}

// replayFor builds the printed command: a package-scoped `go test` with the
// test pinned by -run and the seed passed through -args, which the calling
// package's test binary accepts because importing testutil registers the
// -seed flag. -v keeps the seed line visible on a replay.
func replayFor(name string, seed int64) string {
	return fmt.Sprintf("go test %s -run '%s' -v -args -seed=%d",
		callerPackagePath(3), runPattern(name), seed)
}

// callerPackagePath returns the import path of the package owning the source
// file skip frames above callerPackagePath. The file path is read as the tail
// after internal/app/, so the derivation is identical for absolute and
// -trimpath builds; a non-testutil caller outside the anchor falls back to the
// module root rather than guessing.
func callerPackagePath(skip int) string {
	_, file, _, ok := runtime.Caller(skip)
	if !ok {
		return modulePath
	}
	idx := strings.LastIndex(file, appAnchor)
	if idx < 0 {
		return modulePath
	}
	dir := path.Dir(file[idx+len(appAnchor):])
	if dir == "." {
		return modulePath + strings.TrimSuffix(appAnchor, "/")
	}
	return modulePath + appAnchor + dir
}

// runPattern renders a test name as the component-wise -run regexp the go
// command expects (`^Parent$/^subtest$`), quoting metacharacters so a name
// with punctuation replays exactly the same test.
func runPattern(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		parts[i] = "^" + regexp.QuoteMeta(part) + "$"
	}
	return strings.Join(parts, "/")
}
