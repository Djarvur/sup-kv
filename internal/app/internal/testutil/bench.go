package testutil

import (
	"testing"
	"time"

	benchreport "github.com/Djarvur/sup-kv/internal/testutil"
)

// ReportLatencyPercentiles reports the p50/p99 metrics of a benchmark's
// collected per-op durations in milliseconds (p50-ms and p99-ms). The
// implementation lives in the module-wide internal/testutil package: Go's
// internal-package rule forbids the internal/udprepl and internal/grpcapi
// benchmark packages from importing this one, and TEST-10's three
// benchmarks must share one percentile implementation. This wrapper keeps
// the internal/app call-site spelling (see internal/app/persist/bench_test.go).
func ReportLatencyPercentiles(b *testing.B, durations []time.Duration) {
	b.Helper()
	benchreport.ReportLatencyPercentiles(b, durations)
}
