package grpcapi_test

// The TEST-05 bounded-confirmation-latency benchmark (D-13, plan 06-08) and
// PERF-01's two-profile surface (plan 08-08): BenchmarkConfirmationLatency
// measures per-write wait_for_replication submission-to-resolution latency
// under a continuous ~1000 writes/s background burst, through the REAL stack
// — one server wired with the tracker wait seam over a real two-adapter
// asymmetric UDP topology (the wait_integration_test.go construction), a
// bufconn-dialed gRPC client, the production result-event timing — under
// BOTH profiles decision 14.3 names: `fast` (Debounce 1 ms / MaxWait 50 ms,
// in-memory dump) and `default` (Debounce 50 ms / MaxWait 1 s, real
// filesystem dump). An unacked report FAILS the benchmark: bounded
// confirmation latency is the property under test, a miss is a bug, never
// noise. Manual `make bench` target only, no thresholds (D-13); the fast
// arm's executable bound is TestPerfFastProfileLatency under `make perf`
// (D-06).

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// benchWaitTimeout is the operator default the benchmarked server runs
// with: generous relative to the ~1ms burst cadence and the sub-millisecond
// loopback ack path, so a healthy run never reports unacked — the bound
// exists to fail a genuinely stuck drain, not to be reached.
const benchWaitTimeout = 5 * time.Second

// benchBurstPace is the background burst's ~1ms write cadence (~1000
// writes/s, the D-13 shape the confirmation latency must stay bounded
// under).
const benchBurstPace = time.Millisecond

// benchWaitRunSeq gives every framework b.N-escalation run a fresh measured
// keyspace.
var benchWaitRunSeq atomic.Uint64

// The two payloads: the background burst's and the measured wait writes'.
var (
	benchBurstValue = []byte("bench-burst-value-over-real-udp-0123456789")
	benchWaitValue  = []byte("bench-wait-value")
)

// benchDumpFile is the default arm's dump file name inside its own temp
// directory — the production persistfile store's path.
const benchDumpFile = "dump.json"

// benchProfile is one benchmarked dump profile: the two persist knobs and
// the dump implementation whose cost the measured span includes (D-01: a
// number is only honest with its conditions).
type benchProfile struct {
	name     string
	debounce time.Duration
	maxWait  time.Duration
	// realDump selects the production file store over the in-memory fake:
	// only the real store puts encode + fsync + rotate inside the span.
	realDump bool
}

// benchFastProfile is the fast-response arm the D-01 "~10 ms under 1000
// writes/s" claim is measured under (research PERF-01 "Doc conditions"):
// Debounce 1 ms and MaxWait 50 ms, an IN-MEMORY dump (fakeDumpStore encodes
// into a bytes.Buffer — no file, no fsync, no rotate, so the dump time the
// claim excludes is genuinely absent), two real loopback UDP adapters over
// the asymmetric broadcast topology, a bufconn-dialed gRPC client, a ~1 ms
// background burst writing through the author-notification path, one
// Set(WaitForReplication) per measured iteration, and race off. An unacked
// report fails the run: bounded confirmation latency under this burst is the
// property under test, a miss is a bug, never noise.
var benchFastProfile = benchProfile{
	name:     "fast",
	debounce: time.Millisecond,
	maxWait:  50 * time.Millisecond,
	realDump: false,
}

// benchDefaultProfile is the operator profile the default configuration runs
// (dump_debounce 50 ms, dump_max_wait 1 s — internal/config): the same stack
// as the fast arm, but the dump is the production persistfile store at a
// file in the benchmark's temp directory, so the measured span INCLUDES
// encode + fsync + rotate. An unacked report fails the run, as in the fast
// arm.
var benchDefaultProfile = benchProfile{
	name:     "default",
	debounce: 50 * time.Millisecond,
	maxWait:  time.Second,
	realDump: true,
}

// benchProfiles is the two-arm table decision 14.3 requires: one benchmarked
// profile per dump configuration the product documents, both through the one
// shared body (benchConfirmationLatency).
var benchProfiles = []benchProfile{benchFastProfile, benchDefaultProfile}

// BenchmarkConfirmationLatency benchmarks both documented dump profiles
// through one shared body. Each arm's conditions are stated on its profile;
// the fast arm keeps the exact knobs and in-memory dump the "~10 ms" claim
// cites, the default arm the operator knobs against a real filesystem dump.
func BenchmarkConfirmationLatency(b *testing.B) {
	for _, profile := range benchProfiles {
		b.Run(profile.name, func(b *testing.B) {
			benchConfirmationLatency(b, profile)
		})
	}
}

// benchConfirmationLatency is the one measured body both arms run: a
// background burst writer on node A's store (the author-notification path —
// store.Set plus NotifyLocalWrite, the scenarioWrite precedent — so the
// burst cadence never serializes on dump confirmation) for the whole
// measured window while each iteration performs one gRPC Set with
// WaitForReplication through the real wait seam. The measured span is
// submission to acked report; accumulated totals are reported as
// latency-ns/op plus max-latency-ns (whitespace-free units), and
// p50-ms/p99-ms give the same per-op spans' distribution (TEST-10: measured
// durations, no self-pacing in the loop — the background burst's ~1ms
// cadence is the workload the bound must hold under), while ns/op stays
// visible for the write itself. The fast arm's dump is in-memory, so its
// span excludes dump-file work; the default arm's is persistfile, so its
// span includes encode + fsync + rotate.
func benchConfirmationLatency(b *testing.B, profile benchProfile) {
	b.Helper()
	portA := integrationUDPPort(b)
	portB := integrationUDPPort(b)
	probeLoopbackBroadcast(b, portB)

	// Writer node A, production stack shape (the wait_integration_test.go
	// construction, except the store is built directly so the background
	// burst can write through the author-notification path without waiting
	// on dump confirmation): engine over one store with the hub tap, the
	// adapter owning post-write notification and both wait seams.
	hub := grpcapi.NewWatchHub()
	storeA := storage.NewStore(
		42,
		clock.NewClock(time.Now, 0, 0, time.Minute),
		storage.Config{ValueMaxBytes: testValueLimit, OnChange: hub.Ingest},
	)
	// The dump implementation is the profile's: the fast arm keeps the
	// in-memory fake (no file I/O in its span), the default arm the
	// production file store at a file in this run's temp directory.
	dumps := port.DumpStore(&fakeDumpStore{})
	if profile.realDump {
		dumps = persistfile.New(filepath.Join(b.TempDir(), benchDumpFile))
	}
	eng := persist.New(storeA, dumps, persist.Config{
		Debounce:    profile.debounce,
		MaxWait:     profile.maxWait,
		OnConfirmed: hub.Confirm,
	})
	engCtx, cancelEng := context.WithCancel(b.Context())
	go func() { _ = eng.Run(engCtx) }()
	b.Cleanup(cancelEng)
	hubCtx, cancelHub := context.WithCancel(b.Context())
	go func() { _ = hub.Run(hubCtx) }()
	b.Cleanup(cancelHub)

	adapterA, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", portA),
		fmt.Sprintf("127.255.255.255:%d", portB),
		replNodeA, eng, repl.Config{},
	)
	if err != nil {
		b.Fatalf("node A adapter: %v", err)
	}
	storeB := storage.NewStore(replNodeB, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	adapterB, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", portB),
		fmt.Sprintf("127.255.255.255:%d", portA),
		replNodeB, peerStore{store: storeB}, repl.Config{},
	)
	if err != nil {
		b.Fatalf("node B adapter: %v", err)
	}

	// The adapters share the benchmark's own context — canceled just before
	// cleanups — and the cleanup waits for both Run returns: the adapters
	// close their sockets only there, so repeated runs never leak ports.
	runCtx := b.Context()
	runA := make(chan struct{})
	runB := make(chan struct{})
	b.Cleanup(func() {
		<-runA
		<-runB
	})
	go func() {
		defer close(runA)
		_ = adapterA.Run(runCtx)
	}()
	go func() {
		defer close(runB)
		_ = adapterB.Run(runCtx)
	}()

	// Mutual discovery over real broadcast Hellos: A's write-time peer
	// snapshot must seed the wait-set, and B must know A to Ack.
	waitFor(b, integrationDeadline, "mutual discovery (both known-peer counts == 1)", func() bool {
		return adapterA.KnownPeers() == 1 && adapterB.KnownPeers() == 1
	})

	opts := openGateOptions()
	opts.WritePeers = adapterA.KnownPeerIDs
	opts.WaitReplicated = adapterA.WaitReplicated
	opts.WaitTimeout = benchWaitTimeout
	api := grpcapi.NewServer(eng, adapterA.NotifyLocalWrite, hub, opts)
	gateCtx, cancelGate := context.WithCancel(b.Context())
	b.Cleanup(cancelGate)
	go api.RunGate(gateCtx)

	srv := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(srv, api)
	healthpb.RegisterHealthServer(srv, api.Health())
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	b.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStaticStreamWindowSize(testFlowWindowSize),
	)
	if err != nil {
		b.Fatalf("grpc.NewClient: %v", err)
	}
	b.Cleanup(func() { _ = conn.Close() })
	client := apiproto.NewSupKVServiceClient(conn)
	health := healthpb.NewHealthClient(conn)
	waitFor(b, gateWait, "the health endpoint to flip SERVING (gate open)", func() bool {
		resp, err := health.Check(b.Context(), &healthpb.HealthCheckRequest{})
		return err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	})

	// The background burst: distinct keyspace, ~1ms cadence, stopped by the
	// measured loop's completion. burstErr is written at most once and read
	// only after burstDone closes (the channel close is the happens-before
	// edge).
	var burstErr error
	stopBurst := make(chan struct{})
	burstDone := make(chan struct{})
	go func() {
		defer close(burstDone)
		ticker := time.NewTicker(benchBurstPace)
		defer ticker.Stop()
		for i := 0; ; i++ {
			select {
			case <-stopBurst:
				return
			case <-ticker.C:
			}
			key := fmt.Sprintf("bench/burst/%d", i)
			_, _, err := storeA.Set(key, benchBurstValue)
			if err != nil {
				burstErr = fmt.Errorf("burst Set(%q): %w", key, err)
				return
			}
			rec, err := storeA.Get(key)
			if err != nil {
				burstErr = fmt.Errorf("burst Get(%q): %w", key, err)
				return
			}
			err = adapterA.NotifyLocalWrite(key, rec, false, nil)
			if err != nil {
				burstErr = fmt.Errorf("burst NotifyLocalWrite(%q): %w", key, err)
				return
			}
		}
	}()

	run := benchWaitRunSeq.Add(1)
	b.ResetTimer()
	durations := make([]time.Duration, 0, b.N)
	var totalDrainNanos, maxDrainNanos int64
	for i := range b.N {
		key := fmt.Sprintf("bench/wait/%d/%07d", run, i)
		start := time.Now()
		resp, err := client.Set(b.Context(), &apiproto.SetRequest{
			Key:                key,
			Value:              benchWaitValue,
			WaitForReplication: true,
		})
		drain := time.Since(start)
		if err != nil {
			b.Fatalf("Set %s: want OK, got error: %v", key, err)
		}
		if !resp.GetAcked() {
			b.Fatalf("Set %s: unacked report %d/%d — bounded confirmation latency under burst is the property under test, a miss is a bug",
				key, resp.GetAckedCount(), resp.GetTotalPeers())
		}
		durations = append(durations, drain)
		totalDrainNanos += int64(drain)
		if int64(drain) > maxDrainNanos {
			maxDrainNanos = int64(drain)
		}
	}
	b.StopTimer()

	close(stopBurst)
	<-burstDone
	if burstErr != nil {
		b.Fatalf("background burst failed: %v", burstErr)
	}
	testutil.ReportLatencyPercentiles(b, durations)
	b.ReportMetric(float64(totalDrainNanos)/float64(b.N), "latency-ns/op")
	b.ReportMetric(float64(maxDrainNanos), "max-latency-ns")
}

// TestPerfFastProfileLatency is PERF-01's executable bound for the fast
// profile: it wraps benchConfirmationLatency with benchFastProfile in
// testing.Benchmark and asserts the reported mean (`latency-ns/op`) and
// `p99-ms` against testutil.PerfFastMean and testutil.PerfFastP99, under
// exactly the conditions that profile states — Debounce 1 ms, MaxWait 50 ms,
// the in-memory dump, two real loopback UDP adapters, the bufconn gRPC
// client, the ~1 ms background burst, one Set(WaitForReplication) per
// iteration, race off. The benchmark auto-scales its own sample count (a
// test cannot set -benchtime), so the measured span is per-op and never
// paced by the harness. It runs only under `make perf`: inside `make test` it
// SKIPs, and under -race it SKIPs too (the detector measures the detector).
//
//nolint:paralleltest // Wall-clock latency bound: a parallel sibling sharing the machine would corrupt the measurement.
func TestPerfFastProfileLatency(t *testing.T) {
	testutil.RequirePerf(t)
	block := testutil.HardwareBlock()
	t.Log(block)
	result := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		benchConfirmationLatency(b, benchFastProfile)
	})
	metric := func(key string) float64 {
		value, ok := result.Extra[key]
		if !ok {
			t.Fatalf("PERF-01 %s profile: benchmark reported no %q metric — a missing metric is a failure, never a silent zero",
				benchFastProfile.name, key)
		}
		return value
	}
	mean := time.Duration(metric("latency-ns/op"))
	p50 := time.Duration(metric("p50-ms") * float64(time.Millisecond))
	p99 := time.Duration(metric("p99-ms") * float64(time.Millisecond))
	maxLatency := time.Duration(metric("max-latency-ns"))
	if mean > testutil.PerfFastMean {
		t.Fatalf(
			"PERF-01 %s profile: mean %s exceeds PerfFastMean %s — measured with Debounce %s, MaxWait %s, IN-MEMORY dump, two loopback UDP adapters, bufconn gRPC client, ~%s background burst, %d-byte values, race=off; record the hardware block and load instead of relaxing the bound",
			benchFastProfile.name,
			mean,
			testutil.PerfFastMean,
			benchFastProfile.debounce,
			benchFastProfile.maxWait,
			benchBurstPace,
			len(benchWaitValue),
		)
	}
	if p99 > testutil.PerfFastP99 {
		t.Fatalf(
			"PERF-01 %s profile: p99 %s exceeds PerfFastP99 %s — measured with Debounce %s, MaxWait %s, IN-MEMORY dump, two loopback UDP adapters, bufconn gRPC client, ~%s background burst, %d-byte values, race=off; record the hardware block and load instead of relaxing the bound",
			benchFastProfile.name,
			p99,
			testutil.PerfFastP99,
			benchFastProfile.debounce,
			benchFastProfile.maxWait,
			benchBurstPace,
			len(benchWaitValue),
		)
	}
	t.Logf(
		"PERF-01 %s profile: mean %s (bound %s), p99 %s (bound %s), p50 %s, max %s, samples %d [debounce %s, max-wait %s, dump=in-memory, adapters=2 loopback UDP, client=bufconn, burst=%s, value=%d B, race=off]",
		benchFastProfile.name,
		mean,
		testutil.PerfFastMean,
		p99,
		testutil.PerfFastP99,
		p50,
		maxLatency,
		result.N,
		benchFastProfile.debounce,
		benchFastProfile.maxWait,
		benchBurstPace,
		len(benchWaitValue),
	)
}
