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
	"os"
	"path/filepath"
	"slices"
	"strconv"
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

// perfDumps reads the node's supkv_dumps_total attempt counter, failing when
// the family is absent. It reuses the frozen file's parser rather than
// growing a second one.
func perfDumps(t *testing.T, addr string) float64 {
	t.Helper()
	value, found := redMetricValue(redMetricsBody(t, addr), "supkv_dumps_total")
	if !found {
		t.Fatalf("node %s /metrics carries no supkv_dumps_total family", addr)
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
	preloadDumps := perfDumps(t, metricsAddr)
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

// perfDumpSize returns the main dump file's size in bytes — the generation
// the amplification row multiplies by its dump count.
func perfDumpSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat the dump %s: %v", path, err)
	}
	return info.Size()
}

// perfProcIO reads one numeric field from /proc/<pid>/io — the node process's
// own I/O accounting, where write_bytes counts bytes the process caused to be
// sent to the storage layer. An absent field or a non-numeric value fails by
// name.
func perfProcIO(t *testing.T, pid int, field string) uint64 {
	t.Helper()
	path := fmt.Sprintf("/proc/%d/io", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		name, value, found := strings.Cut(line, ": ")
		if !found || name != field {
			continue
		}
		parsed, parseErr := strconv.ParseUint(value, 10, 64)
		if parseErr != nil {
			t.Fatalf("%s field %q = %q: %v", path, name, value, parseErr)
		}
		return parsed
	}
	t.Fatalf("%s carries no %q field", path, field)
	return 0
}

// perfFailures sums the three labelled supkv_dumps_failures_total class
// samples: the family renders one sample per class, never a total line, so a
// healthy window is "the sum is unchanged" read from the same parser the
// other metrics reads use.
func perfFailures(t *testing.T, addr string) float64 {
	t.Helper()
	body := redMetricsBody(t, addr)
	total := 0.0
	for _, class := range []string{"disk_full", "permission", "other"} {
		value, found := redMetricValue(body, `supkv_dumps_failures_total{class="`+class+`"}`)
		if !found {
			t.Fatalf("node %s /metrics carries no supkv_dumps_failures_total{class=%q} sample", addr, class)
		}
		total += value
	}
	return total
}

// perfDumpSnapshot is the amplification window's input tuple: the dump
// attempt counter, the main dump file's size, the node process's write_bytes
// and the summed failure classes. Its delta across the window is the whole
// measurement, so a failure can be diagnosed from the logged tuple alone.
type perfDumpSnapshot struct {
	dumps      float64
	dumpBytes  int64
	writeBytes uint64
	failures   float64
}

// perfTakeSnapshot reads the tuple from one node's live process and dump.
func perfTakeSnapshot(t *testing.T, addr string, node *standNode, dumpPath string) perfDumpSnapshot {
	t.Helper()
	return perfDumpSnapshot{
		dumps:      perfDumps(t, addr),
		dumpBytes:  perfDumpSize(t, dumpPath),
		writeBytes: perfProcIO(t, node.cmd.Process.Pid, "write_bytes"),
		failures:   perfFailures(t, addr),
	}
}

// TestPerfAmplification is PERF-03's measured row (D-03): disk-write
// amplification = (Δ supkv_dumps_total × dump file size) / the logical bytes
// the hot-key window's clients wrote, cross-checked against the named node's
// /proc/<pid>/io write_bytes delta per successful dump. The method is the
// documented five steps: a real-filesystem dump (a tmpfs/ramfs base is
// refused by name), a fixed 10 000 x 1 KiB preload whose covering dump is
// taken as the baseline, metrics and dump stat on both sides of an open-loop
// 1 000 writes/s x 15 s window, and a healthy-window check that the failure
// counters did not move (a failed attempt writes no generation, research
// A12). Each dump streams one fresh generation, so bytes submitted per dump
// must stay within testutil.PerfBytesPerDumpRatio of the dump size; the
// amplification figure itself is published, not asserted.
func TestPerfAmplification(t *testing.T) { //nolint:paralleltest // wall-clock write window: a parallel sibling sharing the machine would corrupt the measurement (file doc).
	testutil.RequirePerf(t)
	t.Log(testutil.HardwareBlock())

	p := newStandPair(t)
	p.dumpA = filepath.Join(testutil.PerfDumpDir(t), "dump.json")
	p.dumpB = filepath.Join(testutil.PerfDumpDir(t), "dump.json")
	dir := filepath.Dir(p.dumpA)
	fsType := testutil.PerfDumpFSType(dir)
	if fsType == "tmpfs" || fsType == "ramfs" {
		t.Fatalf("SUPKV_PERF_DUMP_DIR=%q lies on %s: a memory-backed dump measures memory, not disk, "+
			"so PERF-03's amplification row refuses it — point SUPKV_PERF_DUMP_DIR at a real filesystem "+
			"(make perf uses $(CURDIR)/.tmp/perf)", os.Getenv("SUPKV_PERF_DUMP_DIR"), fsType)
	}

	metricsAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	syncLong := "--sync-interval=" + perfSyncIntervalLong.String()
	nodeA := startStandNode(t, p, standNodeAID, syncLong, "--metrics-addr="+metricsAddr)
	nodeB := startStandNode(t, p, standNodeBID, syncLong)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)
	standMetricsReady(t, "http://"+metricsAddr+"/metrics")

	// The fixed preload, then its covering dump and an empty unacked set:
	// the baseline window then contains only the measured schedule's dumps.
	preloadDumps := perfDumps(t, metricsAddr)
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

	baseline := perfTakeSnapshot(t, metricsAddr, nodeA, p.dumpA)

	// The send window's end anchors the replication drain exactly as the
	// frozen burst row computes it.
	total := redBurstRate * redBurstSeconds
	interval := time.Second / time.Duration(redBurstRate)
	sendEnd := time.Now().Add(time.Duration(total-1) * interval)

	result := perfOpenLoop(t, p.grpcA)
	if result.ok == 0 {
		t.Fatalf("PERF-03 amplification: the hot-key window recorded no successful write "+
			"(errors by code: %s) — the logical-byte denominator would be zero", perfErrorSummary(result.errs))
	}
	if len(result.errs) > 0 {
		t.Logf("PERF-03 open-loop errors by code: %s (duration incl. drain %v)", perfErrorSummary(result.errs), result.elapsed)
	}
	waitFor(t, time.Until(sendEnd.Add(testutil.PerfBurstDrain+redBurstDrainMargin)), fmt.Sprintf(
		"node A's unacked set empty within %v of the send window's end before the final amplification snapshot",
		testutil.PerfBurstDrain,
	), redThrottled(func() bool {
		value, found := redMetricValue(redMetricsBody(t, metricsAddr), "supkv_replication_unacked_records")
		return found && value == 0
	}))

	final := perfTakeSnapshot(t, metricsAddr, nodeA, p.dumpA)

	// The method, in the order the failure messages name it: Δ dumps × dump
	// size against the logical bytes the clients wrote, cross-checked against
	// write_bytes per dump.
	dumpsDelta := final.dumps - baseline.dumps
	if dumpsDelta <= 0 {
		t.Fatalf("PERF-03 amplification: no dump completed inside the window "+
			"(dumps baseline %.0f, final %.0f; dump %s; write window %v at %d writes/s) — "+
			"an empty window is a measurement failure, not a zero amplification",
			baseline.dumps, final.dumps, p.dumpA, time.Duration(total-1)*interval, redBurstRate)
	}
	if final.failures != baseline.failures {
		t.Fatalf("PERF-03 amplification: the failure counter moved inside the window "+
			"(baseline %.0f, final %.0f) — a failed attempt writes no generation, so the "+
			"size×count formula needs a healthy window (research A12)", baseline.failures, final.failures)
	}
	if final.writeBytes < baseline.writeBytes {
		t.Fatalf("PERF-03 amplification: /proc/%d/io write_bytes went backwards (%d → %d) — "+
			"the counter is monotone; the tuple is unusable",
			nodeA.cmd.Process.Pid, baseline.writeBytes, final.writeBytes)
	}

	keyWidth := len(fmt.Sprintf(redBurstKeyFmt, 0))
	logicalBytes := result.ok * int64(keyWidth+len(redSeedValue()))
	dumpBytes := final.dumpBytes
	writeBytesDelta := final.writeBytes - baseline.writeBytes
	bytesPerDump := float64(writeBytesDelta) / dumpsDelta
	amplification := dumpsDelta * float64(dumpBytes) / float64(logicalBytes)
	bound := testutil.PerfBytesPerDumpRatio * float64(dumpBytes)

	t.Logf("PERF-03 amplification: dumps-baseline=%.0f dumps-final=%.0f dumps-delta=%.0f "+
		"dump-bytes-baseline=%d dump-bytes-final=%d write-bytes-baseline=%d write-bytes-final=%d "+
		"write-bytes-delta=%d bytes-per-dump=%.0f bytes-per-dump-bound=%.0f logical-bytes=%d "+
		"writes=%d key-bytes=%d value-bytes=%d amplification=%.2fx fs=%s dev=%s "+
		"node-a-pid=%d node-b-pid=%d failures=%.0f",
		baseline.dumps, final.dumps, dumpsDelta,
		baseline.dumpBytes, final.dumpBytes, baseline.writeBytes, final.writeBytes,
		writeBytesDelta, bytesPerDump, bound, logicalBytes,
		result.ok, keyWidth, len(redSeedValue()), amplification, fsType, testutil.PerfDumpDevice(dir),
		nodeA.cmd.Process.Pid, nodeB.cmd.Process.Pid, final.failures)

	if bytesPerDump > bound {
		t.Fatalf("PERF-03 amplification: bytes submitted per dump %.0f > "+
			"testutil.PerfBytesPerDumpRatio (%.2f) × dump size (%d = %.0f bound) "+
			"[method: (Δ dumps × dump size)/logical bytes = %.2fx; dumps=%.0f; logical-bytes=%d; "+
			"write-bytes-delta=%d; fs=%s; dev=%s; race=off]",
			bytesPerDump, testutil.PerfBytesPerDumpRatio, dumpBytes, bound, amplification,
			dumpsDelta, logicalBytes, writeBytesDelta, fsType, testutil.PerfDumpDevice(dir))
	}

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the amplification node A")
	assertNoPanic(t, nodeB, "the amplification peer node B")
}

// The PERF-04 memory points' scale and sampling shape. The three states are
// the value-dominated and key-dominated points the model must publish: two
// at 1 KiB and one at the 60 KiB value limit (60 * 1024 == storage's 61 440
// bytes).
const (
	perfMemKeyFmt      = "perf/mem/%05d"
	perfMemSmallKeys   = 1000
	perfMemSmallValue  = 1024
	perfMemLargeKeys   = 10000
	perfMemLargeValue  = 60 * 1024
	perfMemSampleEvery = 2 * time.Millisecond

	// perfKiB scales /proc/<pid>/status's KiB memory fields to bytes.
	perfKiB = 1024
)

// perfSeedKeys writes every key with the caller's value through the worker
// pool — redSeedKeys' shape with the value size parameterised for the memory
// points. Every write must succeed: a failed seed invalidates the
// measurement, so the first error fails the subtest.
func perfSeedKeys(t *testing.T, clients []supkvapi.SupKVServiceClient, keys []string, value []byte) {
	t.Helper()
	if len(clients) == 0 {
		t.Fatal("perfSeedKeys: no clients")
	}
	next := make(chan string, len(keys))
	for _, key := range keys {
		next <- key
	}
	close(next)
	var (
		wg       sync.WaitGroup
		errMu    sync.Mutex
		firstErr error
	)
	for w := range redSeedWorkers {
		client := clients[w%len(clients)]
		wg.Go(func() {
			for key := range next {
				ctx, cancel := context.WithTimeout(t.Context(), redSeedRPCTimeout)
				_, err := client.Set(ctx, &supkvapi.SetRequest{Key: key, Value: value})
				cancel()
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("seed Set(%q): %w", key, err)
					}
					errMu.Unlock()
				}
			}
		})
	}
	wg.Wait()
	if firstErr != nil {
		t.Fatalf("seeding %d keys: %v", len(keys), firstErr)
	}
}

// perfProcStatus reads one /proc/<pid>/status memory field in bytes: status
// reports VmRSS and VmHWM in KiB, so the value is scaled by perfKiB. ok is
// false for an unreadable file or an absent/non-numeric field, which lets
// the transient sampler skip a miss instead of failing from its own
// goroutine.
func perfProcStatus(pid int, field string) (uint64, bool) {
	path := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		name, value, found := strings.Cut(line, ":")
		if !found || name != field {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			return 0, false
		}
		kib, parseErr := strconv.ParseUint(fields[0], 10, 64)
		if parseErr != nil {
			return 0, false
		}
		return kib * perfKiB, true
	}
	return 0, false
}

// perfStatusBytes reads one /proc/<pid>/status memory field and fails the
// test by name when it is absent — the test-goroutine wrapper around
// perfProcStatus.
func perfStatusBytes(t *testing.T, pid int, field string) uint64 {
	t.Helper()
	value, ok := perfProcStatus(pid, field)
	if !ok {
		t.Fatalf("read %s from /proc/%d/status", field, pid)
	}
	return value
}

// perfDumpPeakRSS forces one dump (one covering write) and samples VmRSS
// every perfMemSampleEvery until the write returns, returning the RSS
// immediately before the write and the peak observed across the cycle. The
// write is synchronous at the engine: Set returns only after a dump
// containing its record completed (PERS-04), so the sampling window covers
// exactly one dump. The sampler runs in the test goroutine (the write blocks
// in its own goroutine), so a missing sample never calls FailNow off-thread.
func perfDumpPeakRSS(t *testing.T, client supkvapi.SupKVServiceClient, pid int, key string, value []byte) (idle, peak uint64) {
	t.Helper()
	idle = perfStatusBytes(t, pid, "VmRSS")
	peak = idle
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), redBurstRPCTimeout)
		defer cancel()
		_, err := client.Set(ctx, &supkvapi.SetRequest{Key: key, Value: value})
		done <- err
	}()
	ticker := time.NewTicker(perfMemSampleEvery)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("the forced dump's covering write: %v", err)
			}
			return idle, peak
		case <-ticker.C:
			if current, sampled := perfProcStatus(pid, "VmRSS"); sampled && current > peak {
				peak = current
			}
		}
	}
}

// TestPerfMemoryPoints is PERF-04's measured half (D-04): three state→RSS
// points — 1 000 x 1 KiB (a small state), 10 000 x 1 KiB (key-dominated
// overhead) and 1 000 x 60 KiB (value-dominated, the flag's limit) — each
// on its own fresh pair, real filesystem dump and a pinned sync interval.
// Every row logs the logical state size (Σ(len(key)+len(value)), with the
// formatted key width stated), the dump bytes, the current RSS
// (process_resident_memory_bytes), VmHWM, the pre-dump RSS and the peak
// observed across one further forced dump, plus both ratios and the
// filesystem type. Only the value-dominated 60 KiB row gates: the loose
// PerfMemoryRSSRatio tripwire and the PerfDumpPeakLiveRatio dump-transient
// gate (that constant's only consumer — a return to encode-into-bytes would
// show roughly 4-5x). The precise transient stays the persist benchmark's
// peak-live-B; this row's sampled peak is deliberately the loose check.
func TestPerfMemoryPoints(t *testing.T) { //nolint:paralleltest // one measurement per machine: the subtests are sequential by contract (file doc).
	testutil.RequirePerf(t)
	t.Log(testutil.HardwareBlock())

	points := []struct {
		name       string
		keys       int
		valueBytes int
	}{
		{name: "1k-1KiB", keys: perfMemSmallKeys, valueBytes: perfMemSmallValue},
		{name: "10k-1KiB", keys: perfMemLargeKeys, valueBytes: perfMemSmallValue},
		{name: "1k-60KiB", keys: perfMemSmallKeys, valueBytes: perfMemLargeValue},
	}
	for _, point := range points { //nolint:paralleltest // a shared machine would let one state's dump contaminate the next point's RSS.
		t.Run(point.name, func(t *testing.T) {
			perfAssertMemoryPoint(t, point.keys, point.valueBytes)
		})
	}
}

// perfAssertMemoryPoint measures one PERF-04 state→RSS point: preload the
// state, wait for its covering dump, record the five figures, force one more
// dump while sampling VmRSS, and apply the two gates on the value-dominated
// row only.
func perfAssertMemoryPoint(t *testing.T, keys, valueBytes int) {
	t.Helper()
	p := newStandPair(t)
	p.dumpA = filepath.Join(testutil.PerfDumpDir(t), "dump.json")
	p.dumpB = filepath.Join(testutil.PerfDumpDir(t), "dump.json")
	dir := filepath.Dir(p.dumpA)
	fsType := testutil.PerfDumpFSType(dir)

	metricsAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	metricsURL := "http://" + metricsAddr + "/metrics"
	syncLong := "--sync-interval=" + perfSyncIntervalLong.String()
	nodeA := startStandNode(t, p, standNodeAID, syncLong, "--metrics-addr="+metricsAddr)
	nodeB := startStandNode(t, p, standNodeBID, syncLong)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)
	standMetricsReady(t, metricsURL)

	preloadDumps := perfDumps(t, metricsAddr)
	value := []byte(strings.Repeat("v", valueBytes))
	keyList := make([]string, keys)
	for i := range keyList {
		keyList[i] = fmt.Sprintf(perfMemKeyFmt, i)
	}
	perfSeedKeys(t, redSeedClients(t, p.grpcA), keyList, value)
	waitFor(t, perfPreloadWait, "the preload's covering dump and an empty replication unacked set",
		redThrottled(func() bool {
			body := redMetricsBody(t, metricsAddr)
			dumps, dumpsFound := redMetricValue(body, "supkv_dumps_total")
			unacked, unackedFound := redMetricValue(body, "supkv_replication_unacked_records")
			return dumpsFound && unackedFound && dumps > preloadDumps && unacked == 0
		}))

	pid := nodeA.cmd.Process.Pid
	keyWidth := len(fmt.Sprintf(perfMemKeyFmt, 0))
	logical := keys * (keyWidth + valueBytes)
	dumpBytes := perfDumpSize(t, p.dumpA)
	rss := standMetricsSample(t, standMetricsScrape(t, metricsURL), "process_resident_memory_bytes")
	vmRSS := perfStatusBytes(t, pid, "VmRSS")
	vmHWM := perfStatusBytes(t, pid, "VmHWM")

	dumpsBefore := perfDumps(t, metricsAddr)
	rssIdle, rssPeak := perfDumpPeakRSS(t, standClient(t, p.grpcA), pid, keyList[0], value)
	if after := perfDumps(t, metricsAddr); after <= dumpsBefore {
		t.Fatalf("the forced write did not complete a dump (supkv_dumps_total %.0f → %.0f): "+
			"the dump-cycle RSS peak has no cycle to describe", dumpsBefore, after)
	}

	rssRatio := rss / float64(logical)
	peakRatio := float64(rssPeak-rssIdle) / float64(logical)
	t.Logf("PERF-04 memory point: keys=%d value-bytes=%d key-bytes=%d logical=%d dump-bytes=%d "+
		"rss=%.0f vmrss=%d vmhwm=%d rssIdle=%d rssPeak=%d rss-ratio=%.2fx peak-ratio=%.2fx fs=%s pid=%d",
		keys, valueBytes, keyWidth, logical, dumpBytes, rss, vmRSS, vmHWM, rssIdle, rssPeak,
		rssRatio, peakRatio, fsType, pid)
	t.Logf("PERF-04 formula: logical = Σ(len(key)+len(value)); dump_bytes ≈ 1.37 × Σ len(value) + " +
		"JSON/key overhead; live heap = one Records copy + one streamed record line (64 KiB bufio); " +
		"rss ≈ baseline + live heap + GC headroom; one dump cycle's peak live ≈ 1.3 × logical (PERSIST-06)")

	if valueBytes == perfMemLargeValue {
		rssBound := testutil.PerfMemoryRSSRatio*float64(logical) + testutil.PerfMemoryRSSBase
		if rss > rssBound {
			t.Fatalf("PERF-04 memory tripwire: RSS %.0f > PerfMemoryRSSRatio (%.1f) × logical (%d) + "+
				"PerfMemoryRSSBase (%d) = %.0f [point=%d keys × %d-byte values; dump-bytes=%d; "+
				"rss-ratio=%.2fx; fs=%s; pid=%d; race=off]",
				rss, testutil.PerfMemoryRSSRatio, logical, testutil.PerfMemoryRSSBase, rssBound,
				keys, valueBytes, dumpBytes, rssRatio, fsType, pid)
		}
		peakBound := testutil.PerfDumpPeakLiveRatio * float64(logical)
		if float64(rssPeak-rssIdle) > peakBound {
			t.Fatalf("PERF-04 dump transient: sampled peak RSS growth %d > PerfDumpPeakLiveRatio (%.1f) × "+
				"logical (%d) = %.0f [point=%d keys × %d-byte values; rssIdle=%d rssPeak=%d; "+
				"peak-ratio=%.2fx; dump-bytes=%d; fs=%s; pid=%d; race=off] — a return to "+
				"encode-into-bytes would show roughly 4-5x",
				rssPeak-rssIdle, testutil.PerfDumpPeakLiveRatio, logical, peakBound,
				keys, valueBytes, rssIdle, rssPeak, peakRatio, dumpBytes, fsType, pid)
		}
	}

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the memory-point node A")
	assertNoPanic(t, nodeB, "the memory-point peer node B")
}
