package grpcapi_test

// The TEST-05 bounded-confirmation-latency benchmark (D-13, plan 06-08):
// BenchmarkConfirmationLatency measures per-write wait_for_replication
// appearance-to-drain latency under a continuous ~1000 writes/s background
// burst, through the REAL stack — one server wired with the wait seam over
// a real two-adapter asymmetric UDP topology (the wait_integration_test.go
// construction), a bufconn-dialed gRPC client, the production mirror
// timing. An unacked report FAILS the benchmark: bounded confirmation
// latency is the property under test, a miss is a bug, never noise.
// Manual `make bench` target only, no thresholds (D-13).

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
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

// BenchmarkConfirmationLatency runs a background burst writer on node A's
// store (the author-notification path — store.Set plus NotifyLocalWrite,
// the scenarioWrite precedent — so the burst cadence never serializes on
// dump confirmation) for the whole measured window while each iteration
// performs one gRPC Set with WaitForReplication through the real wait seam.
// The measured span is submission to acked report; accumulated totals are
// reported as latency-ns/op plus max-latency-ns (whitespace-free units),
// p50-ms/p99-ms give the same per-op spans' distribution (TEST-10: measured
// durations, no self-pacing in the loop — the background burst's ~1ms
// cadence is the workload the bound must hold under), and ns/op stays
// visible for the write itself.
func BenchmarkConfirmationLatency(b *testing.B) {
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
	eng := persist.New(storeA, &fakeDumpStore{}, persist.Config{Debounce: time.Millisecond, MaxWait: 50 * time.Millisecond})
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
	opts.KnownPeers = adapterA.KnownPeers
	opts.PendingCopy = adapterA.PendingCopy
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
			err = adapterA.NotifyLocalWrite(key, rec)
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
