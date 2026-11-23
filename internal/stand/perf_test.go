//go:build linux

// The Phase 8 operator-facing perf rows. The first row is the twin of
// TestRed_R02_burst: it calls the frozen file's redRunBurst and reuses its
// rate/keys/worker constants READ-ONLY (stand_red_test.go is byte-pinned),
// so both names exercise one implementation while the frozen red contract
// stays untouched. The bounds come from testutil.PerfBurstMinRate and
// testutil.PerfBurstDrain, and TestPerfBurstBoundsMatchRedContract (a pure
// constant check with no gate) fails by name if the two sides ever drift.
//
// Gating: every perf row's first statement is testutil.RequirePerf(t), so
// `make test` skips them (naming `make perf`) and only `make perf` measures.
// The rows run SEQUENTIALLY (no t.Parallel): they assert a wall-clock
// throughput bound, and a parallel sibling sharing the machine would corrupt
// the measurement (the same reason the red scenarios carry no t.Parallel).

package stand //nolint:testpackage // the perf rows reuse the harness's and the frozen red file's unexported helpers and constants (newStandPair, redRunBurst, redBurstMinAchieved); testpackage's external-package rule cannot express that pairing.

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/testutil"
)

// TestPerfBurstDistinctKeys is the operator-runnable twin of
// TestRed_R02_burst (PERF-02, D-02): an open-loop 1000 writes/s x 15 s burst
// over 10 000 distinct keys against one node must achieve at least
// testutil.PerfBurstMinRate writes/s and empty the unacked set within
// testutil.PerfBurstDrain of the send window's end, with the same
// measurement the frozen row documents. It logs the hardware block first and
// exactly one machine-greppable verdict line last.
func TestPerfBurstDistinctKeys(t *testing.T) { //nolint:paralleltest // wall-clock throughput bound: a parallel sibling would corrupt the measurement (file doc).
	testutil.RequirePerf(t)
	t.Log(testutil.HardwareBlock())

	p := newStandPair(t)
	// The perf rows never dump to /tmp: PerfDumpDir honors the operator's
	// base and logs the media facts, so a later amplification row can refuse
	// a memory-backed filesystem.
	p.dumpA = filepath.Join(testutil.PerfDumpDir(t), "dump.json")
	p.dumpB = filepath.Join(testutil.PerfDumpDir(t), "dump.json")

	metricsAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	nodeA := startStandNode(t, p, standNodeAID, "--metrics-addr="+metricsAddr)
	nodeB := startStandNode(t, p, standNodeBID)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	// The send window's end is the schedule's last slot — the anchor the
	// acceptance's "5 s after the end" reads, exactly as the frozen row
	// computes it.
	total := redBurstRate * redBurstSeconds
	interval := time.Second / time.Duration(redBurstRate)
	sendEnd := time.Now().Add(time.Duration(total-1) * interval)

	result := redRunBurst(t, p.grpcA)
	sendWindow := time.Duration(total-1) * interval
	achieved := float64(result.ok) / sendWindow.Seconds()
	if achieved < testutil.PerfBurstMinRate {
		errSummary := make([]string, 0, len(result.errs))
		for code, count := range result.errs {
			errSummary = append(errSummary, fmt.Sprintf("%s=%d", code, count))
		}
		slices.Sort(errSummary)
		t.Fatalf(
			"PERF-02 burst achieved %.1f writes/s over %v (%d/%d writes OK; total incl. drain %v), want at least %.0f writes/s (testutil.PerfBurstMinRate); errors by code: %s",
			achieved, sendWindow, result.ok, total, result.elapsed, testutil.PerfBurstMinRate,
			strings.Join(errSummary, ", "),
		)
	}

	// The unacked set must empty within the documented bound after the send
	// window's end, plus the frozen row's poll margin. A schedule that
	// outlived its own drain deadline fails immediately — the drain took
	// too long, which is the breach.
	drainTimeout := time.Until(sendEnd.Add(testutil.PerfBurstDrain + redBurstDrainMargin))
	waitFor(t, drainTimeout, fmt.Sprintf(
		"node A's unacked set empty within %v of the burst's send window end (PERF-02)",
		testutil.PerfBurstDrain,
	), redThrottled(func() bool {
		value, found := redMetricValue(redMetricsBody(t, metricsAddr), "supkv_replication_unacked_records")
		return found && value == 0
	}))

	t.Logf("PERF-02 burst: achieved=%.1f writes/s (min %.0f) drain=%v (bound %v) keys=%d value-bytes=%d fs=%s",
		achieved, testutil.PerfBurstMinRate, result.elapsed-sendWindow, testutil.PerfBurstDrain,
		redBurstKeys, len(redSeedValue()), testutil.PerfDumpFSType(filepath.Dir(p.dumpA)))

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the burst node A")
	assertNoPanic(t, nodeB, "the burst peer node B")
}

// TestPerfBurstBoundsMatchRedContract is the drift guard: the perf threshold
// and the frozen red contract must name the same burst bounds, so a change
// to either side fails here by name — in `make test`, with no env gate.
func TestPerfBurstBoundsMatchRedContract(t *testing.T) {
	t.Parallel()
	redRate := float64(redBurstMinAchieved)
	if redRate != testutil.PerfBurstMinRate {
		t.Fatalf("PERF-02 drift: the frozen red contract's redBurstMinAchieved = %d writes/s, the perf threshold testutil.PerfBurstMinRate = %v — both must name the same bound",
			redBurstMinAchieved, testutil.PerfBurstMinRate)
	}
	redDrain := redBurstDrainBound
	if redDrain != testutil.PerfBurstDrain {
		t.Fatalf("PERF-02 drift: the frozen red contract's redBurstDrainBound = %v, the perf threshold testutil.PerfBurstDrain = %v — both must name the same bound",
			redBurstDrainBound, testutil.PerfBurstDrain)
	}
}
