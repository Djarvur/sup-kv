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

	// perfDumpEnvVar is the operator's dump-directory override; `make perf`
	// sets it to a directory on the repository's real filesystem so a
	// measurement never lands on a memory-backed /tmp.
	perfDumpEnvVar = "SUPKV_PERF_DUMP_DIR"

	// perfDumpDirMode is the per-test dump subdirectory's owner-only mode.
	perfDumpDirMode = 0o750

	// perfUnknown is the hardware block's field placeholder: a missing
	// kernel file degrades one field to it instead of failing the run.
	perfUnknown = "unknown"
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

// raceLabel renders the race build-tag state for the hardware block.
func raceLabel() string {
	if RaceEnabled() {
		return "on"
	}
	return "off"
}

// perfDumpDir is the platform-independent dump-directory rule: a fresh
// per-test subdirectory under SUPKV_PERF_DUMP_DIR when the operator set it,
// else the test's TempDir, logged with the media facts the platform probes
// report. Both PerfDumpDir variants delegate here so the rule has one
// definition.
func perfDumpDir(tb testing.TB, fstype, device func(string) string) string {
	tb.Helper()
	var dir string
	if base := os.Getenv(perfDumpEnvVar); base != "" {
		//nolint:gosec // G703: the operator supplies the dump base; a wrong path fails the run loudly, and no request input reaches it.
		err := os.MkdirAll(base, perfDumpDirMode)
		if err != nil {
			tb.Fatalf("SUPKV_PERF_DUMP_DIR %q: %v", base, err)
		}
		//nolint:usetesting // t.TempDir() would create under the test's own root; the operator's base must hold the dump so a -v run reports its media.
		created, err := os.MkdirTemp(base, "perf-*")
		if err != nil {
			tb.Fatalf("SUPKV_PERF_DUMP_DIR %q: create per-test dump directory: %v", base, err)
		}
		tb.Cleanup(func() { _ = os.RemoveAll(created) }) //nolint:gosec // G703: created is the subdirectory MkdirTemp just made under the operator's base.
		dir = created
	} else {
		dir = tb.TempDir()
	}
	tb.Logf("perf dump dir: %s fs=%s dev=%s", dir, fstype(dir), device(dir))
	return dir
}
