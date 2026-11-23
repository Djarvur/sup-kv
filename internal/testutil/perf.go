package testutil

// This file is the ONE threshold place for Phase 8 (D-06): compiled
// constants, no environment parsing, no threshold file. It stays in
// internal/testutil (not internal/app/internal/testutil) because Go's internal
// rule forbids internal/udprepl and internal/grpcapi from importing the
// app-internal helper package.

import (
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

// The Phase 8 threshold table (RED stub: the basis values land in GREEN).
const (
	PerfDefaultP50        time.Duration = 0
	PerfDefaultP99        time.Duration = 0
	PerfFastMean          time.Duration = 0
	PerfFastP99           time.Duration = 0
	PerfBurstMinRate      float64       = 0
	PerfBurstDrain        time.Duration = 0
	PerfBytesPerDumpRatio float64       = 0
	PerfAbsentPeerRatio   float64       = 0
	PerfMemoryRSSRatio    float64       = 0
	PerfMemoryRSSBase     int           = 0
	PerfDumpPeakLiveRatio float64       = 0
)

// PerfEnabled reports whether SUPKV_PERF is exactly "1".
func PerfEnabled() bool { return true }

// RequirePerf is the one gate call a perf test makes.
func RequirePerf(tb testing.TB) { tb.Helper() }

// RaceEnabled reports whether this binary was built with -race.
func RaceEnabled() bool { return false }
