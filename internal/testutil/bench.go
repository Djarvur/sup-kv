// Package testutil holds the module-wide test helpers shared by benchmark
// packages outside internal/app. It is a normal package linked only into
// test binaries; the no-test-libs-in-prod lint rule exempts the testutil
// directory (the rng.go precedent), which is what allows the testing import.
//
// This module-wide home exists because Go's internal-package rule forbids
// internal/udprepl and internal/grpcapi from importing
// internal/app/internal/testutil, while TEST-10's three benchmarks must
// share one percentile implementation. The internal/app-internal testutil
// re-exports the reporter for its own consumers.
package testutil

import (
	"slices"
	"testing"
	"time"
)

// The percentile ranks the reporter asks for and the scale they are
// expressed in: nearest-rank ceilings over a sorted sample.
const (
	p50Rank   = 50
	p99Rank   = 99
	rankScale = 100
)

// ReportLatencyPercentiles reports the p50 and p99 of a benchmark's
// collected per-op durations in milliseconds, under the whitespace-free
// metric names p50-ms and p99-ms (this repo's ReportMetric convention: see
// the benchmarks' latency-ns/op and dump-bytes). It sorts durations in
// place. A single sample reports p50 == p99 == that sample (the
// -benchtime=1x smoke), and an empty slice reports zeros rather than
// dropping the metrics.
func ReportLatencyPercentiles(b *testing.B, durations []time.Duration) {
	b.Helper()
	slices.Sort(durations)
	var p50, p99 time.Duration
	if len(durations) > 0 {
		p50 = percentileRank(durations, p50Rank)
		p99 = percentileRank(durations, p99Rank)
	}
	b.ReportMetric(millis(p50), "p50-ms")
	b.ReportMetric(millis(p99), "p99-ms")
}

// percentileRank returns the nearest-rank p-th percentile of sorted: the
// ceil(p/100*n)-th smallest sample, clamped into range. p is a rank in
// (0, 100] and sorted is non-empty, so rank is at least 1.
func percentileRank(sorted []time.Duration, p int) time.Duration {
	rank := (p*len(sorted) + rankScale - 1) / rankScale
	return sorted[max(rank-1, 0)]
}

// millis converts a duration to the float64 millisecond value ReportMetric
// prints.
func millis(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}
