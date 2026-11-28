package udprepl_test

// The TEST-05 replication-burst benchmark (D-13, plan 06-08):
// BenchmarkReplicationBurst drives the ~1000 writes/s burst shape over REAL
// UDP sockets — two adapters on the 06-06 asymmetric topology (this file's
// scenario helpers, including the loud darwin capability skip), shrunken
// scenario timers — and measures the wall time from the first write to full
// CONVERGENCE: the benchmark ends only when node B serves every written key
// under its exact version, so a lost write fails the benchmark instead of
// skewing the rate. Manual `make bench` target only, no thresholds (D-13).

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/testutil"
)

// benchBurstPace is the ~1ms write cadence of the D-13 burst shape (~1000
// writes/s): a ticker paces the send side while the benchmark's wall clock
// runs to full convergence, so ns/op honestly reads "time per converged
// write" and the derived writes/s metric carries the throughput.
const benchBurstPace = time.Millisecond

// benchConvergeDeadline bounds the post-burst convergence wait. Writes drain
// continuously DURING the burst — B applies and Acks each DataReplication
// within sub-millisecond loopback RTTs — so only the tail entries' ack
// ladder matters on a healthy link (first retransmit at 200ms, saturating
// steps at 5s); ten seconds covers the ladder to saturation plus a loaded
// CI machine's headroom while still failing a genuinely wedged run.
const benchConvergeDeadline = 10 * time.Second

// benchBurstRunSeq gives every framework b.N-escalation run a fresh key
// range, so "node B serves all b.N keys" can never be satisfied by residue
// of an earlier, smaller run.
var benchBurstRunSeq atomic.Uint64

// benchBurstValue is the burst's fixed replicated payload.
var benchBurstValue = []byte("bench-burst-value-over-real-udp-0123456789")

// BenchmarkReplicationBurst measures two-node burst convergence on real
// sockets: after mutual discovery on the asymmetric topology, b.N records
// are written on node A at the ~1ms burst cadence through the author's
// notification path (scenarioWrite), then the run waits — inside the
// measured window — until node B serves every key under its exact written
// version. Reports writes/s (b.N over the write-start-to-convergence span),
// max-unacked, the largest pending-entry backlog node A's core mirror
// showed during the burst, and p50-ms/p99-ms of each write's own
// author-side span (TEST-10: per-op durations from the measured loop; the
// ~1ms cadence is the burst's workload shape — the ticker paces the writes
// being measured, it does not throttle the measurement). The backlog is the
// shape of the run, not a threshold.
func BenchmarkReplicationBurst(b *testing.B) {
	cfg := scenarioTimers(nil) // quiet Log: repl.Config.normalized substitutes a no-op
	a, nodeB, _ := startScenarioPair(b, cfg, cfg)
	waitFor(b, scenarioDiscoveryDeadline, "mutual discovery (both known-peer counts == 1)", func() bool {
		return a.adapter.KnownPeers() == 1 && nodeB.adapter.KnownPeers() == 1
	})

	run := benchBurstRunSeq.Add(1)
	keys := make([]string, b.N)
	vers := make([]version.RecordVersion, b.N)
	for i := range keys {
		keys[i] = fmt.Sprintf("bench/burst/%d/%07d", run, i)
	}

	b.ResetTimer()
	ticker := time.NewTicker(benchBurstPace)
	start := time.Now()
	durations := make([]time.Duration, 0, b.N)
	var maxUnacked int
	for i := range b.N {
		<-ticker.C
		opStart := time.Now()
		vers[i] = scenarioWrite(b, a, keys[i], benchBurstValue)
		durations = append(durations, time.Since(opStart))
		if backlog := a.adapter.UnackedStats().Count; backlog > maxUnacked {
			maxUnacked = backlog
		}
	}
	ticker.Stop()
	waitFor(b, benchConvergeDeadline, "node B to serve every burst key under its written version", func() bool {
		for i := range keys {
			if !scenarioServes(nodeB, keys[i], vers[i], benchBurstValue) {
				return false
			}
		}
		return true
	})
	elapsed := time.Since(start)
	b.StopTimer()

	testutil.ReportLatencyPercentiles(b, durations)
	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "writes/s")
	b.ReportMetric(float64(maxUnacked), "max-unacked")
}
