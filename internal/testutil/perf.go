package testutil

// This file is the ONE threshold place for Phase 8 (D-06): compiled
// constants, no environment parsing, no threshold file. It stays in
// internal/testutil (not internal/app/internal/testutil) because Go's internal
// rule forbids internal/udprepl and internal/grpcapi from importing the
// app-internal helper package.

import (
	"os"
	"testing"
	"time"
)

const (
	// perfEnvVar is the one environment read a perf test makes: the operator
	// target opens the gate, and no perf body parses a threshold from it.
	perfEnvVar = "SUPKV_PERF"

	// perfSkipNoEnv is the skip a perf test reports outside `make perf`.
	perfSkipNoEnv = "perf: operator-facing threshold run — invoke `make perf` (SUPKV_PERF=1, no -race)"

	// perfSkipRace is the skip a perf test reports under the race detector.
	perfSkipRace = "perf: numbers are measured without -race — run `make perf`"
)

// The Phase 8 threshold table (D-06). Every value is a documented bound with
// the basis that produced it — not a stored profile and not a self-pacing
// target. A consumer names the PERF ID and the constant in its failure
// message and never retypes a literal; a threshold that proves too tight is
// changed here only after a recorded rerun.
const (
	// PerfDefaultP50 and PerfDefaultP99 bound the default profile (10k x
	// 1 KiB, 1000 writes/s, sustained flow): the review's 0.528-0.560 s p50 /
	// 1.019-1.054 s p99 plus ~40% headroom, bounded by the model's
	// dump_max_wait + dump time.
	PerfDefaultP50 = 750 * time.Millisecond
	PerfDefaultP99 = 1500 * time.Millisecond

	// PerfFastMean and PerfFastP99 bound the fast profile (Debounce 1 ms /
	// MaxWait 50 ms, the benchmark's conditions): the review's 11.53 ms mean /
	// 44.93 ms p99 with roughly 2x headroom; p99 is also about 2 x MaxWait.
	PerfFastMean = 25 * time.Millisecond
	PerfFastP99  = 100 * time.Millisecond

	// PerfBurstMinRate and PerfBurstDrain are REQUIREMENTS PERF-02's bounds,
	// identical to the frozen red contract's constants
	// (internal/stand/stand_red_test.go:114-115); the drift guard
	// TestPerfBurstBoundsMatchRedContract keeps both sides identical.
	PerfBurstMinRate = 990.0
	PerfBurstDrain   = 5 * time.Second

	// PerfBytesPerDumpRatio bounds disk amplification: each dump writes one
	// generation, so submitted bytes per dump stay within 10% of the dump
	// file's size.
	PerfBytesPerDumpRatio = 1.10

	// PerfAbsentPeerRatio bounds the absent-peer write cost at twice the
	// live-peer path (D-05, XPORT-02 acceptance).
	PerfAbsentPeerRatio = 2.0

	// PerfMemoryRSSRatio, PerfMemoryRSSBase and PerfDumpPeakLiveRatio are the
	// memory tripwires only — RSS at 1k x 60 KiB stays under 12x the logical
	// state plus a 256 MiB allowance (the review measured 5.7-9x on the old
	// design), and one dump cycle's peak live bytes stay under 2x the logical
	// state (PERSIST-06 accepts <= 1.3x).
	PerfMemoryRSSRatio    = 12.0
	PerfMemoryRSSBase     = 256 << 20
	PerfDumpPeakLiveRatio = 2.0
)

// PerfEnabled reports whether the operator opened a perf threshold run: only
// the exact value "1" (the `make perf` spelling) is true. Every other value,
// including unset, keeps the gate shut — a typo must not measure the clock.
func PerfEnabled() bool { return os.Getenv(perfEnvVar) == "1" }

// RequirePerf is the one gate call a perf test needs: it SKIPs (never fails)
// when SUPKV_PERF is not exactly 1, naming `make perf`, and SKIPs under the
// race detector because the detector measures itself rather than the product.
// A perf test's first statement is testutil.RequirePerf(t).
func RequirePerf(tb testing.TB) {
	tb.Helper()
	if !PerfEnabled() {
		tb.Skip(perfSkipNoEnv)
	}
	if RaceEnabled() {
		tb.Skip(perfSkipRace)
	}
}
