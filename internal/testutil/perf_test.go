package testutil //nolint:testpackage // the tests assert the unexported gate constants (perfEnvVar, perfSkipNoEnv, perfSkipRace), which an external test package cannot reach.

import (
	"strings"
	"testing"
	"time"
)

// TestPerfEnabled pins the gate's exact contract: only SUPKV_PERF=1 opens a
// perf threshold run; every other spelling (including unset) keeps it shut.
// The rows mutate the process environment, so they run sequentially — Go
// forbids t.Setenv in a parallel test.
func TestPerfEnabled(t *testing.T) {
	rows := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset", value: "", want: false}, // t.Setenv("") is the unset spelling Getenv cannot distinguish
		{name: "one", value: "1", want: true},
		{name: "zero", value: "0", want: false},
		{name: "true", value: "true", want: false},
		{name: "yes", value: "yes", want: false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Setenv(perfEnvVar, row.value)
			if got := PerfEnabled(); got != row.want {
				t.Fatalf("PerfEnabled() with SUPKV_PERF=%q = %v, want %v (only the exact value 1 opens the gate)", row.value, got, row.want)
			}
		})
	}
}

// TestRequirePerfSkips pins the one gate call a perf test makes: it must SKIP
// (never fail) a test that runs outside the operator target, and its skip
// messages must name `make perf` so the operator knows the command.
func TestRequirePerfSkips(t *testing.T) {
	t.Setenv(perfEnvVar, "0")
	var skipped bool
	t.Run("without the operator target", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		RequirePerf(t)
	})
	if !skipped {
		t.Fatal("RequirePerf did not skip without SUPKV_PERF=1: a perf test must never measure outside make perf")
	}

	if RaceEnabled() {
		skipped = false
		t.Run("under the race detector", func(t *testing.T) {
			t.Setenv(perfEnvVar, "1")
			defer func() { skipped = t.Skipped() }()
			RequirePerf(t)
		})
		if !skipped {
			t.Fatal("RequirePerf did not skip under -race with SUPKV_PERF=1: the race detector measures itself, not the product")
		}
	}

	if !strings.Contains(perfSkipNoEnv, "make perf") {
		t.Fatalf("the no-env skip message %q must name `make perf`", perfSkipNoEnv)
	}
	if !strings.Contains(perfSkipRace, "make perf") {
		t.Fatalf("the race skip message %q must name `make perf`", perfSkipRace)
	}
}

// TestPerfThresholdRelations is the truncated-or-inverted table's tripwire:
// every threshold is positive and the pair relations a writer would most
// plausibly invert (p50/p99, mean/p99, a ratio below one) hold immediately.
func TestPerfThresholdRelations(t *testing.T) {
	t.Parallel()
	for name, value := range map[string]time.Duration{
		"PerfDefaultP50": PerfDefaultP50,
		"PerfDefaultP99": PerfDefaultP99,
		"PerfFastMean":   PerfFastMean,
		"PerfFastP99":    PerfFastP99,
		"PerfBurstDrain": PerfBurstDrain,
	} {
		if value <= 0 {
			t.Fatalf("%s = %v, want a positive threshold (a zero row disables its gate)", name, value)
		}
	}
	for name, value := range map[string]float64{
		"PerfBurstMinRate":      PerfBurstMinRate,
		"PerfBytesPerDumpRatio": PerfBytesPerDumpRatio,
		"PerfAbsentPeerRatio":   PerfAbsentPeerRatio,
		"PerfMemoryRSSRatio":    PerfMemoryRSSRatio,
		"PerfDumpPeakLiveRatio": PerfDumpPeakLiveRatio,
	} {
		if value <= 0 {
			t.Fatalf("%s = %v, want a positive threshold (a zero row disables its gate)", name, value)
		}
	}
	if PerfMemoryRSSBase <= 0 {
		t.Fatalf("PerfMemoryRSSBase = %d, want a positive allowance", PerfMemoryRSSBase)
	}

	if PerfDefaultP50 >= PerfDefaultP99 {
		t.Fatalf("PerfDefaultP50 = %v, PerfDefaultP99 = %v: want p50 < p99", PerfDefaultP50, PerfDefaultP99)
	}
	if PerfFastMean >= PerfFastP99 {
		t.Fatalf("PerfFastMean = %v, PerfFastP99 = %v: want mean < p99", PerfFastMean, PerfFastP99)
	}
	if PerfBurstMinRate <= 0 {
		t.Fatalf("PerfBurstMinRate = %v, want a positive rate", PerfBurstMinRate)
	}
	if PerfAbsentPeerRatio < 1 {
		t.Fatalf("PerfAbsentPeerRatio = %v, want at least 1 (an absent peer may never be faster than a live one)", PerfAbsentPeerRatio)
	}
	if PerfBytesPerDumpRatio < 1 {
		t.Fatalf("PerfBytesPerDumpRatio = %v, want at least 1 (the dump submits its own size, never less)", PerfBytesPerDumpRatio)
	}
}

// TestRaceEnabledStable pins the build-tag pair's contract without probing
// the runtime: the same test file compiles in both modes, and each mode's
// race_on.go/race_off.go answers a constant. Two calls must agree; the tag's
// value itself is asserted by which side of the pair the compiler selected.
func TestRaceEnabledStable(t *testing.T) {
	t.Parallel()
	first := RaceEnabled()
	second := RaceEnabled()
	if first != second {
		t.Fatalf("RaceEnabled() = %v on the first call and %v on the second, want the build tag's constant", first, second)
	}
	t.Logf("race build tag: %v", first)
}
