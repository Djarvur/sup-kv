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
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	supkvapi "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/testutil"
)

// The PERF-01/03/04 rows' measurement shape. Every rate, key count and value
// size that shapes a published condition is named here and reused by the
// rows below; the scheduling constants themselves are the frozen burst row's
// (redBurstRate, redBurstSeconds, redBurstWorkers, redBurstConns,
// redBurstRPCTimeout), read-only.
const (
	// perfSyncIntervalLong pins the periodic anti-entropy listing far past
	// the measured windows: the startup gate still opens on the first-seen
	// targeted listing and the empty-disk timer, so ten minutes keeps
	// listing rounds and their paced egress out of every latency,
	// amplification and memory figure. Later flags win over nodeArgs'
	// 200 ms harness default.
	perfSyncIntervalLong = 10 * time.Minute

	// perfHotKeys is the PERF-01 latency row's hot-key set size: the
	// documented 1 000 writes/s over 100 keys is 10 writes/s per key, the
	// sustained-flow shape the default-profile model describes, with no
	// distinct-key retransmission storm in the window.
	perfHotKeys = 100

	// perfPreloadWait bounds the setup wait after a preload — the covering
	// dump and the empty replication unacked set. It is a precondition
	// bound, never a published measurement: 10 000 dump-confirmed 1 KiB
	// writes at the default max-wait take tens of seconds.
	perfPreloadWait = 2 * time.Minute
)

// perfMetric reads one rendered sample from the node's /metrics page,
// failing by name when the family is absent. It reuses the frozen file's
// parser rather than growing a second one.
func perfMetric(t *testing.T, addr, name string) float64 {
	t.Helper()
	value, found := redMetricValue(redMetricsBody(t, addr), name)
	if !found {
		t.Fatalf("node %s /metrics carries no %s family", addr, name)
	}
	return value
}

// perfErrorSummary renders an error-count map as sorted name=count pairs —
// the failure diagnostics the open-loop rows share.
func perfErrorSummary(errs map[codes.Code]int64) string {
	summary := make([]string, 0, len(errs))
	for code, count := range errs {
		summary = append(summary, fmt.Sprintf("%s=%d", code, count))
	}
	slices.Sort(summary)
	return strings.Join(summary, ", ")
}

// perfOpenLoopResult carries the open loop's outcome: the successful op
// count, the elapsed wall time (the send window plus the full in-flight
// drain), the per-op latencies measured from each op's SCHEDULED instant to
// its completion (queueing delay included — no coordinated omission), and
// the error counts by gRPC code.
type perfOpenLoopResult struct {
	ok        int64
	elapsed   time.Duration
	latencies []time.Duration
	errs      map[codes.Code]int64
}

// perfOpenLoop drives the documented open-loop schedule — redBurstRate
// writes/s for redBurstSeconds over perfHotKeys hot keys — with
// redBurstWorkers workers round-robin over redBurstConns connections and
// redBurstRPCTimeout per RPC (the frozen burst row's shape, reused
// read-only). The jobs channel buffers the ENTIRE schedule, so the send
// timeline stays open-loop while the in-flight drain lags behind it; each
// job carries its scheduled instant and a worker records completion minus
// that instant on success.
func perfOpenLoop(t *testing.T, addr string) perfOpenLoopResult {
	t.Helper()
	clients := make([]supkvapi.SupKVServiceClient, redBurstConns)
	for i := range clients {
		clients[i] = standClient(t, addr)
	}
	total := redBurstRate * redBurstSeconds
	interval := time.Second / time.Duration(redBurstRate)
	keys := make([]string, perfHotKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf(redBurstKeyFmt, i)
	}
	value := []byte(redSeedValue())
	type job struct {
		n         int
		scheduled time.Time
	}
	jobs := make(chan job, total)
	var (
		wg        sync.WaitGroup
		errMu     sync.Mutex
		okN       atomic.Int64
		latencies = make([]time.Duration, total)
		succeeded = make([]bool, total)
		errs      = make(map[codes.Code]int64)
	)
	for w := range redBurstWorkers {
		client := clients[w%len(clients)]
		wg.Go(func() {
			for j := range jobs {
				ctx, cancel := context.WithTimeout(t.Context(), redBurstRPCTimeout)
				_, err := client.Set(ctx, &supkvapi.SetRequest{Key: keys[j.n%len(keys)], Value: value})
				cancel()
				if err != nil {
					errMu.Lock()
					errs[status.Code(err)]++
					errMu.Unlock()
					continue
				}
				latencies[j.n] = time.Since(j.scheduled)
				succeeded[j.n] = true
				okN.Add(1)
			}
		})
	}
	begin := time.Now()
	for n := range total {
		scheduled := begin.Add(time.Duration(n) * interval)
		if d := time.Until(scheduled); d > 0 {
			time.Sleep(d)
		}
		jobs <- job{n: n, scheduled: scheduled}
	}
	close(jobs)
	wg.Wait()
	ok := okN.Load()
	samples := make([]time.Duration, 0, int(ok))
	for n := range total {
		if succeeded[n] {
			samples = append(samples, latencies[n])
		}
	}
	return perfOpenLoopResult{ok: ok, elapsed: time.Since(begin), latencies: samples, errs: errs}
}

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
			redDrain, testutil.PerfBurstDrain)
	}
}

// TestPerfDefaultProfileLatency is PERF-01's measured half for the DEFAULT
// profile (D-01): 10 000 x 1 KiB preloaded on a real filesystem, then an
// open-loop 1 000 writes/s x 15 s schedule over perfHotKeys hot keys, with
// --sync-interval pinned to perfSyncIntervalLong so no periodic listing
// round runs inside the window. Every sample is completion minus the op's
// SCHEDULED instant, so a backlog cannot hide behind a closed loop (no
// coordinated omission). The profile knobs are the production defaults
// (dump_debounce 50 ms, dump_max_wait 1 s); the bounds are the one const
// table's PerfDefaultP50/PerfDefaultP99, and the failure names the measured
// p50/p99, the bound, the profile, the key counts, the value size, the
// achieved rate, the media and the race state. The row skips in `make test`
// through testutil.RequirePerf and runs sequentially on the real binary.
func TestPerfDefaultProfileLatency(t *testing.T) { //nolint:paralleltest // wall-clock latency distribution: a parallel sibling sharing the machine would corrupt the measurement (file doc).
	testutil.RequirePerf(t)
	t.Log(testutil.HardwareBlock())

	p := newStandPair(t)
	p.dumpA = filepath.Join(testutil.PerfDumpDir(t), "dump.json")
	p.dumpB = filepath.Join(testutil.PerfDumpDir(t), "dump.json")
	fsType := testutil.PerfDumpFSType(filepath.Dir(p.dumpA))

	metricsAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	syncLong := "--sync-interval=" + perfSyncIntervalLong.String()
	nodeA := startStandNode(t, p, standNodeAID, syncLong, "--metrics-addr="+metricsAddr)
	nodeB := startStandNode(t, p, standNodeBID, syncLong)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)
	standMetricsReady(t, "http://"+metricsAddr+"/metrics")

	// The preload's covering dump and an empty replication unacked set are
	// the measurement precondition: the window then carries only the
	// schedule's own dumps, never the seed's backlog.
	preloadDumps := perfMetric(t, metricsAddr, "supkv_dumps_total")
	preloadKeys := make([]string, redBurstKeys)
	for i := range preloadKeys {
		preloadKeys[i] = fmt.Sprintf(redBurstKeyFmt, i)
	}
	redSeedKeys(t, redSeedClients(t, p.grpcA), preloadKeys)

	waitFor(t, perfPreloadWait, "the preload's covering dump and an empty replication unacked set",
		redThrottled(func() bool {
			body := redMetricsBody(t, metricsAddr)
			dumps, dumpsFound := redMetricValue(body, "supkv_dumps_total")
			unacked, unackedFound := redMetricValue(body, "supkv_replication_unacked_records")
			return dumpsFound && unackedFound && dumps > preloadDumps && unacked == 0
		}))

	// The send window's end is the schedule's last slot — the same anchor
	// the frozen burst row computes, so the drain bound counts from the
	// window's end rather than from after the in-flight drain.
	total := redBurstRate * redBurstSeconds
	interval := time.Second / time.Duration(redBurstRate)
	sendEnd := time.Now().Add(time.Duration(total-1) * interval)

	result := perfOpenLoop(t, p.grpcA)
	if result.ok == 0 {
		t.Fatalf("PERF-01 default profile: the open loop recorded no successful write "+
			"(errors by code: %s) — a zero-sample latency distribution is a measurement "+
			"precondition failure, never a pass", perfErrorSummary(result.errs))
	}
	p50, p99 := testutil.LatencyPercentiles(result.latencies)
	sendWindow := time.Duration(total-1) * interval
	achieved := float64(result.ok) / sendWindow.Seconds()

	waitFor(t, time.Until(sendEnd.Add(testutil.PerfBurstDrain+redBurstDrainMargin)), fmt.Sprintf(
		"node A's unacked set empty within %v of the send window's end (the PERF-01 drain precondition; the PERF-02 bound is asserted by TestPerfBurstDistinctKeys)",
		testutil.PerfBurstDrain,
	), redThrottled(func() bool {
		value, found := redMetricValue(redMetricsBody(t, metricsAddr), "supkv_replication_unacked_records")
		return found && value == 0
	}))
	drain := time.Since(sendEnd)
	valueBytes := len(redSeedValue())

	t.Logf("PERF-01 default profile: p50=%v p99=%v rate=%.1f writes/s keys=%d value-bytes=%d fs=%s race=off state-keys=%d drain=%v",
		p50, p99, achieved, perfHotKeys, valueBytes, fsType, redBurstKeys, drain)
	if len(result.errs) > 0 {
		t.Logf("PERF-01 open-loop errors by code: %s (duration incl. drain %v)", perfErrorSummary(result.errs), result.elapsed)
	}

	if p50 > testutil.PerfDefaultP50 {
		t.Fatalf("PERF-01 default profile: measured p50 %v > testutil.PerfDefaultP50 %v "+
			"[profile=default; state-keys=%d; hot-keys=%d; value-bytes=%d; rate=%.1f writes/s; "+
			"fs=%s; race=off; drain=%v; errors-by-code=%s]",
			p50, testutil.PerfDefaultP50, redBurstKeys, perfHotKeys, valueBytes, achieved, fsType, drain,
			perfErrorSummary(result.errs))
	}
	if p99 > testutil.PerfDefaultP99 {
		t.Fatalf("PERF-01 default profile: measured p99 %v > testutil.PerfDefaultP99 %v "+
			"[profile=default; state-keys=%d; hot-keys=%d; value-bytes=%d; rate=%.1f writes/s; "+
			"fs=%s; race=off; drain=%v; errors-by-code=%s]",
			p99, testutil.PerfDefaultP99, redBurstKeys, perfHotKeys, valueBytes, achieved, fsType, drain,
			perfErrorSummary(result.errs))
	}

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the default-profile node A")
	assertNoPanic(t, nodeB, "the default-profile peer node B")
}
