package testutil_test

// The shared percentile primitive's contract: one nearest-rank rule, the
// same one the benchmarks have always reported. The stand's latency row
// reads LatencyPercentiles directly, so a rank-rule change here changes a
// published PERF-01 figure and must fail this pin first.

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/testutil"
)

// TestLatencyPercentiles pins nearest-rank on a known sample set. For the
// durations 1..100 the p50 is the 50th sample and the p99 the 99th; a single
// sample is both percentiles; an empty slice yields zeroes rather than a
// panic or a dropped metric.
func TestLatencyPercentiles(t *testing.T) {
	t.Parallel()

	samples := make([]time.Duration, 100)
	for i := range samples {
		samples[i] = time.Duration(i+1) * time.Millisecond
	}
	p50, p99 := testutil.LatencyPercentiles(samples)
	if p50 != 50*time.Millisecond {
		t.Fatalf("LatencyPercentiles(1..100 ms) p50 = %v, want 50ms (nearest rank 50 of 100)", p50)
	}
	if p99 != 99*time.Millisecond {
		t.Fatalf("LatencyPercentiles(1..100 ms) p99 = %v, want 99ms (nearest rank 99 of 100)", p99)
	}

	single := []time.Duration{7 * time.Millisecond}
	p50, p99 = testutil.LatencyPercentiles(single)
	if p50 != single[0] || p99 != single[0] {
		t.Fatalf("LatencyPercentiles(single 7ms) = (%v, %v), want (7ms, 7ms)", p50, p99)
	}

	p50, p99 = testutil.LatencyPercentiles(nil)
	if p50 != 0 || p99 != 0 {
		t.Fatalf("LatencyPercentiles(nil) = (%v, %v), want (0, 0)", p50, p99)
	}
}
