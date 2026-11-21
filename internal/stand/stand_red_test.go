//go:build linux

// The stand-level red tests for the 2026-11-19 v1.0 review's Critical
// findings (D-01): R-01 paced sync over the documented key count
// (XPORT-01), R-02's unacked-set drain and its 1000 writes/s burst
// (XPORT-02(2)/(3)), and R-03's replaced-node F14 acceptance at stand
// level (PROTO-01, ADR 0005 / decision 14.5). Every test is written as
// the FULL documented acceptance assertion so the Phase 3-5 flip needs
// zero edits here — only the expectation-table row in
// scripts/red-expectations.tsv changes (the completed table and the
// runner's package list are plan 02-12's deliverables).
//
// Gating (D-01): each test body's first statement skips unless
// SUPKV_RED_TESTS=1, so `make test` stays green while the defects the
// tests pin are unfixed; scripts/red-runner opens the gate itself,
// asserts the FAIL events via `go test -json`, and records the evidence.
//
// Scale selection: SUPKV_STAND_NIGHTLY=1 (the hook declared by
// stand_test.go, set by the scheduled red-run job alongside
// SUPKV_RED_TESTS — plan 02-11) selects the review's documented scale per
// scenario; the default scale keeps a local red run viable while still
// breaching v1.0 code (verified failing empirically per scale before the
// nightly wiring landed).
//
// The tests run SEQUENTIALLY (no t.Parallel): R-01 and R-02 assert
// pacing and throughput bounds against the wall clock, and two such
// scenarios sharing the machine would corrupt each other's measurements.
// stand_test.go's parallel green pins resume only after every sequential
// test here has started, and they are small enough never to matter.

package stand //nolint:testpackage // the plan mandates internal tests so the red scenarios share the harness's unexported helpers (D-03); testpackage's external-package rule cannot express that pairing.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/version"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

// The red scenarios' scales and their bound inputs. Every timing bound
// below is DERIVED from the flags the nodes run with — the documented
// expressions (2 × sync_interval + N / sync_response_rate; N /
// sync_response_rate + 10 s; the burst's rate and window) plus an explicit
// margin, never a magic wall-clock constant. The reviewer's measured
// breaches (docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md) are
// cited in the comments, never asserted: F9b's ~1015 s convergence against
// an ~83 s bound; F9's storm at 949 552 retransmits and 14.5 min for the
// unacked set to fall 8909 → 0; the burst's 465/s achieved against 990
// required, with 4327 records still unacked 42 s after the end.
const (
	// redSyncResponseRate is the sync_response_rate every red node runs
	// with — main.go's default, passed explicitly so each bound derivation
	// names the flag its paced term reads.
	redSyncResponseRate = 500

	// R-01 (XPORT-01): the documented scale is 20 000 keys × 1 KiB against
	// the documented default timers (sync_interval 30 s); the default
	// local scale keeps the red run viable while still breaching v1.0 —
	// at 8000 keys the full listing spans roughly 8000 × 40 B ≈ 320 KiB of
	// DataSync chunks, well beyond the ~229 KiB Linux default receive
	// buffer the review measured (rb229376), so the same-tail-lost-on-
	// every-round lockout reproduces.
	redR01KeysDefault = 8000
	redR01KeysNightly = 20000
	redR01KeyFmt      = "red/r01/key-%05d"
	redR01Prefix      = "red/r01/"

	// R-01's documented timer terms, selected by standNightly: the bound's
	// 2 × sync_interval is the review's 30 s periodic cadence and the 1 s
	// hello interval (later flags override the stand's shrunken nodeArgs
	// defaults); the default scale runs the stand's shrunken timers.
	redR01NightlyHelloInterval = time.Second
	redR01NightlySyncInterval  = 30 * time.Second

	// R-02 part 1 (XPORT-02(2)): the documented scale — a node holding
	// 10 000 unacked 1 KiB records for a returning peer. The documented
	// scale is already locally viable (the red verdict fires at the bound,
	// not at the reviewer's measured 14.5-minute storm), so nightly and
	// default coincide. redDrainGrace is the acceptance's own "+ 10 s".
	redDrainKeys   = 10000
	redDrainKeyFmt = "red/r02drain/key-%05d"
	redDrainGrace  = 10 * time.Second

	// R-02 part 2 (XPORT-02(3)): the documented burst — 1000 writes/s ×
	// 15 s over 10 000 distinct keys, measured the reviewer's way (the
	// loadgen's open-loop shape: 1024 workers round-robin over 8
	// connections, achieved = successful ops / elapsed with elapsed
	// covering the full in-flight drain, per-RPC deadline 10 s — the
	// loadgen's -timeout default). The documented scale is the local
	// scale.
	redBurstRate        = 1000
	redBurstSeconds     = 15
	redBurstKeys        = 10000
	redBurstWorkers     = 1024
	redBurstConns       = 8
	redBurstRPCTimeout  = 10 * time.Second
	redBurstMinAchieved = 990
	redBurstDrainBound  = 5 * time.Second
	redBurstDrainMargin = 2 * time.Second
	redBurstKeyFmt      = "red/burst/key-%05d"

	// The seeding helper's shape: the review's own preload profile — 64
	// workers round-robin over 8 connections (the closed-loop 10k row's
	// measured ~700 writes/s).
	redSeedWorkers    = 64
	redSeedConns      = 8
	redSeedRPCTimeout = 30 * time.Second

	// redPollEvery paces the red tests' polling RPCs: waitFor's own
	// cadence is the harness's 10 ms, far too hot for full-listing List
	// calls and /metrics scrapes — each condition self-throttles to this
	// interval and reports not-yet between polls.
	redPollEvery = 500 * time.Millisecond
)

// redSeedValue is the scenarios' deterministic 1 KiB value.
func redSeedValue() string {
	return strings.Repeat("red-", 256)
}

// redSeedClients opens redSeedConns gRPC clients on addr — the parallel
// seeding helper's connection pool.
func redSeedClients(t *testing.T, addr string) []apiproto.SupKVServiceClient {
	t.Helper()
	clients := make([]apiproto.SupKVServiceClient, redSeedConns)
	for i := range clients {
		clients[i] = standClient(t, addr)
	}
	return clients
}

// redSeedKeys writes every key with the scenarios' 1 KiB value through the
// worker pool — the seeding half of the R-01/R-02 scenarios. Every write
// must succeed: a failed seed invalidates the scenario, so the first error
// fails the test.
func redSeedKeys(t *testing.T, clients []apiproto.SupKVServiceClient, keys []string) {
	t.Helper()
	value := redSeedValue()
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
				_, err := client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte(value)})
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

// redMetricsBody GETs the node's /metrics page — the noctx-clean request
// shape (the harness's httpGetBody pattern) over the opt-in --metrics-addr
// endpoint the stand topology binds to loopback (threat T-2-10's accepted
// surface). A transient scrape error returns an empty body the metric
// parser reads as not-yet.
func redMetricsBody(t *testing.T, addr string) string {
	t.Helper()
	url := "http://" + addr + "/metrics"
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext(%s): %v", url, err)
	}
	client := &http.Client{Timeout: checkTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil || closeErr != nil {
		return ""
	}
	return string(body)
}

// redMetricValue extracts one metric's sample value from a rendered
// /metrics body (the in-repo exporter's one-sample-per-family text shape):
// the line whose name matches exactly, parsed as its trailing float. found
// is false while the family is absent, so a mid-restart scrape reads as
// not-yet for the poll's caller.
func redMetricValue(body, name string) (value float64, found bool) {
	for line := range strings.SplitSeq(body, "\n") {
		rest, ok := strings.CutPrefix(line, name+" ")
		if !ok {
			continue
		}
		parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(rest), 64)
		if parseErr != nil {
			continue
		}
		return parsed, true
	}
	return 0, false
}

// redThrottled wraps a polling condition so its expensive body runs at
// most once per redPollEvery: waitFor's deadline clamp is kept, while the
// cheap calls between polls just report not-yet.
func redThrottled(cond func() bool) func() bool {
	var last time.Time
	return func() bool {
		if now := time.Now(); now.Sub(last) < redPollEvery {
			return false
		}
		last = time.Now()
		return cond()
	}
}

// redR01Keys returns R-01's key count: the review's documented 20 000-key
// scale under SUPKV_STAND_NIGHTLY=1, else the default local scale (still
// breaching: the listing overflows the default receive buffer either way).
func redR01Keys() int {
	if standNightly {
		return redR01KeysNightly
	}
	return redR01KeysDefault
}

// TestRed_R01_pacedSync20k pins review finding R-01 (the XPORT-01
// acceptance) at stand level as an expected-red test (D-01): two nodes on
// real Linux sockets with default buffers — a seeded node A and an empty
// receiver B — must fully converge no later than 2 × sync_interval +
// N / sync_response_rate, 20 runs out of 20. A seeds its keys while B is
// offline and is then RESTARTED, so A holds nothing pending and B's only
// path to the state is the periodic DataSync anti-entropy — the review's
// F9b pure-resync shape, the R-01 evidence (distinct from F9's R-02
// retransmit storm, which TestRed_R02_unackedDrain pins). v1.0 sends the
// whole listing as every chunk back-to-back in one event-loop iteration
// with unconfigured socket buffers: the listing overflows the ~229 KiB
// receive buffer and the same sorted-tail chunks are lost on every round —
// the review measured 11 740 keys stuck at 9908 from 160 s through 271 s
// (four rounds without progress, the missing keys all in the tail deciles)
// and fully converged only at ~1015 s against the ~83 s bound. The test
// asserts the DOCUMENTED expression derived from the flags the nodes run
// with, so Phase 3's paced-listing flip changes only the expectation
// table.
func TestRed_R01_pacedSync20k(t *testing.T) { //nolint:paralleltest // wall-clock pacing bounds: parallel red scenarios would corrupt each other's measurements (file doc).
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	p := newStandPair(t)

	keys := redR01Keys()
	hello, syncInterval := standHelloInterval, standSyncInterval
	var nodeExtras []string
	if standNightly {
		// The documented scale runs the documented default timers (the
		// review's 30 s sync cadence and 1 s hello interval; later flags
		// override the stand's shrunken nodeArgs defaults).
		hello = redR01NightlyHelloInterval
		syncInterval = redR01NightlySyncInterval
		nodeExtras = append(nodeExtras,
			"--hello-interval="+hello.String(),
			"--sync-interval="+syncInterval.String(),
		)
	}
	nodeExtras = append(nodeExtras, "--sync-response-rate="+strconv.Itoa(redSyncResponseRate))

	// Node A alone: seed N keys × 1 KiB, then restart so nothing is left
	// pending — the pure anti-entropy shape (F9b).
	nodeA := startStandNode(t, p, standNodeAID, nodeExtras...)
	awaitServing(t, p.grpcA, servingDeadline)
	seedKeys := make([]string, keys)
	for i := range seedKeys {
		seedKeys[i] = fmt.Sprintf(redR01KeyFmt, i)
	}
	redSeedKeys(t, redSeedClients(t, p.grpcA), seedKeys)

	signalAndWaitExit(t, nodeA)
	nodeA = startStandNode(t, p, standNodeAID, nodeExtras...)
	awaitServing(t, p.grpcA, servingDeadline)

	// The documented convergence bound, derived from the flags the nodes
	// run with: 2 × sync_interval (a brand-new peer gets no immediate
	// listing on v1.0 — R-05 — so the listing starts at the periodic
	// cadence, and the second interval admits one retry) plus
	// N / sync_response_rate (the paced per-record egress term) plus the
	// margin: B's startup gate, discovery hellos, and process slack on a
	// loaded runner.
	paced := time.Duration(keys) * (time.Second / time.Duration(redSyncResponseRate))
	margin := standStartupGate + 2*hello + 3*time.Second
	deadline := 2*syncInterval + paced + margin

	// Node B against the live peer, empty disk: every key must arrive over
	// the wire within the bound.
	nodeB := startStandNode(t, p, standNodeBID, nodeExtras...)
	awaitServing(t, p.grpcB, servingDeadline)
	clientB := standClient(t, p.grpcB)

	want := make(map[string]struct{}, keys)
	for _, key := range seedKeys {
		want[key] = struct{}{}
	}
	var lastLog time.Time
	waitFor(t, deadline, fmt.Sprintf(
		"the empty-disk node B serving every one of the %d seeded keys within the documented XPORT-01 bound (2×sync_interval %v + N/rate %v + margin %v)",
		keys, 2*syncInterval, paced, margin,
	), redThrottled(func() bool {
		ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
		resp, listErr := clientB.List(ctx, &apiproto.ListRequest{Prefix: redR01Prefix})
		cancel()
		if listErr != nil {
			return false
		}
		got := resp.GetKeys()
		if len(got) != len(want) {
			if now := time.Now(); now.Sub(lastLog) >= 2*time.Second {
				lastLog = now
				t.Logf("r01 progress: %d/%d keys listed on B", len(got), len(want))
			}
			return false
		}
		for _, key := range got {
			if _, ok := want[key]; !ok {
				return false
			}
		}
		return true
	}))

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the seeded node A")
	assertNoPanic(t, nodeB, "the empty-disk recovered node B")
}

// TestRed_R02_unackedDrain pins review finding R-02 part 1 (the
// XPORT-02(2) acceptance) as an expected-red test (D-01): a peer arriving
// at a node holding 10 000 unacked 1 KiB records — the set must empty
// within N / sync_response_rate + 10 s, with at most 3N retransmissions
// read from A's /metrics (supkv_replication_datagrams_retransmitted_total,
// the exact family name internal/metrics renders). v1.0 re-copies and
// re-sorts the whole unacked set on every event (mirrorState,
// PendingSnapshot, retainDrained) and bursts the entire set on peer
// discovery and at every retransmit-ladder step, unpaced and jitter-free:
// Ack loss feeds a self-sustaining storm. The review's F9 measured the
// set falling 8909 → 0 over ~14.5 minutes with 949 552 retransmits —
// against the ~30 s bound and the 30 000-datagram cap asserted here. The
// drain verdict fires at the bound (not at the storm's full length), so
// the documented scale is already locally viable and nightly coincides
// with the default scale.
func TestRed_R02_unackedDrain(t *testing.T) { //nolint:paralleltest // wall-clock pacing bounds: parallel red scenarios would corrupt each other's measurements (file doc).
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	p := newStandPair(t)
	metricsAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))

	// Node A alone, metrics on: the seed writes land while B is down, and
	// with an empty peer table every record is retained unacked in
	// suspension (the core's empty-table guard), so A's gauge carries the
	// scenario's precondition before B ever exists.
	nodeA := startStandNode(t, p, standNodeAID,
		"--sync-response-rate="+strconv.Itoa(redSyncResponseRate),
		"--metrics-addr="+metricsAddr,
	)
	awaitServing(t, p.grpcA, servingDeadline)

	drainKeys := make([]string, redDrainKeys)
	for i := range drainKeys {
		drainKeys[i] = fmt.Sprintf(redDrainKeyFmt, i)
	}
	redSeedKeys(t, redSeedClients(t, p.grpcA), drainKeys)

	// The precondition wait is generous (60 s), not a documented bound: a
	// write's unacked entry lands on the core when the event loop drains
	// its notification queue, at the loop's per-event cost — the very R-02
	// cost this scenario pins (the review's 3.6 ms/event at 10 000
	// records), so the last entries trail the seeding by tens of seconds.
	var lastLog time.Time
	waitFor(t, 60*time.Second, "node A's unacked gauge holding every seeded record while B is down",
		redThrottled(func() bool {
			value, found := redMetricValue(redMetricsBody(t, metricsAddr), "supkv_replication_unacked_records")
			if now := time.Now(); found && now.Sub(lastLog) >= 2*time.Second {
				lastLog = now
				t.Logf("r02drain precondition: %d/%d unacked", int(value), redDrainKeys)
			}
			return found && value == float64(redDrainKeys)
		}))

	// The peer returns: B's Hello ends A's empty peer table — discovery
	// flushes the whole pending set in one burst and the retransmit ladder
	// takes over. The documented drain bound: N / sync_response_rate + 10 s
	// plus the poll/setup margin.
	nodeB := startStandNode(t, p, standNodeBID)
	awaitServing(t, p.grpcB, servingDeadline)
	paced := time.Duration(redDrainKeys) * (time.Second / time.Duration(redSyncResponseRate))
	margin := 5 * time.Second
	waitFor(t, paced+redDrainGrace+margin, fmt.Sprintf(
		"node A's unacked set drained within the documented XPORT-02(2) bound (N/rate %v + 10 s + margin %v)",
		paced, margin,
	), redThrottled(func() bool {
		value, found := redMetricValue(redMetricsBody(t, metricsAddr), "supkv_replication_unacked_records")
		return found && value == 0
	}))

	// The retransmission cap over the same episode, read from A's metrics.
	retransmits, found := redMetricValue(redMetricsBody(t, metricsAddr), "supkv_replication_datagrams_retransmitted_total")
	if !found {
		t.Fatal("node A's /metrics carries no supkv_replication_datagrams_retransmitted_total family")
	}
	if retransmits > 3*float64(redDrainKeys) {
		t.Fatalf("node A retransmitted %.0f datagrams draining %d unacked records, want at most 3N = %d (XPORT-02(2))",
			retransmits, redDrainKeys, 3*redDrainKeys)
	}

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the unacked-set node A")
	assertNoPanic(t, nodeB, "the returning peer node B")
}

// redBurstResult carries the open-loop burst's outcome: the successful
// op count, the elapsed wall time (send window plus the full in-flight
// drain — the loadgen's achieved denominator) and the error counts by
// gRPC code for the failure diagnostics.
type redBurstResult struct {
	ok      int64
	elapsed time.Duration
	errs    map[codes.Code]int64
}

// redRunBurst drives the documented open-loop schedule the reviewer's
// loadgen drives: redBurstRate × redBurstSeconds operations pushed on a
// fixed interval timeline (queueing included — no coordinated omission),
// 1024 workers round-robin over 8 connections, per-RPC deadline 10 s (the
// loadgen's -timeout default), each write over redBurstKeys distinct keys.
// The jobs channel buffers the ENTIRE schedule (the loadgen's shape): a
// scheduler that never blocks on a backlog is what keeps the send timeline
// open-loop while the in-flight drain lags behind it.
func redRunBurst(t *testing.T, addr string) redBurstResult {
	t.Helper()
	clients := make([]apiproto.SupKVServiceClient, redBurstConns)
	for i := range clients {
		clients[i] = standClient(t, addr)
	}
	total := redBurstRate * redBurstSeconds
	interval := time.Second / time.Duration(redBurstRate)
	jobs := make(chan int, total)
	var (
		wg    sync.WaitGroup
		errMu sync.Mutex
		okN   atomic.Int64
		errs  = make(map[codes.Code]int64)
	)
	for w := range redBurstWorkers {
		client := clients[w%len(clients)]
		wg.Go(func() {
			for n := range jobs {
				ctx, cancel := context.WithTimeout(t.Context(), redBurstRPCTimeout)
				_, err := client.Set(ctx, &apiproto.SetRequest{
					Key:   fmt.Sprintf(redBurstKeyFmt, n%redBurstKeys),
					Value: []byte(redSeedValue()),
				})
				cancel()
				if err != nil {
					errMu.Lock()
					errs[status.Code(err)]++
					errMu.Unlock()
					continue
				}
				okN.Add(1)
			}
		})
	}
	begin := time.Now()
	for n := range total {
		if d := time.Until(begin.Add(time.Duration(n) * interval)); d > 0 {
			time.Sleep(d)
		}
		jobs <- n
	}
	close(jobs)
	wg.Wait()
	return redBurstResult{ok: okN.Load(), elapsed: time.Since(begin), errs: errs}
}

// TestRed_R02_burst pins review finding R-02 part 2 (the XPORT-02(3)
// acceptance) as an expected-red test (D-01): an open-loop 1000 writes/s ×
// 15 s burst over 10 000 distinct keys against one node must achieve at
// least 990 writes/s and leave the unacked set empty 5 s after the end
// (supkv_replication_unacked_records via /metrics). The burst's ~1000 new
// unacked records per second drive the same event-loop cost R-02 part 1
// pins: the review measured 465 writes/s achieved (p99 10.9 s), with 4327
// records still unacked and 102 835 retransmits 42 s after the end, and
// 277 835 socket drops — against the ≥990 writes/s and the 5 s drain
// asserted here. Achieved is measured the reviewer's own way (successful
// ops over elapsed, elapsed covering the full in-flight drain), so the
// Phase 3 flip is the expectation-table row only.
func TestRed_R02_burst(t *testing.T) { //nolint:paralleltest // wall-clock pacing bounds: parallel red scenarios would corrupt each other's measurements (file doc).
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	p := newStandPair(t)
	metricsAddr := fmt.Sprintf("127.0.0.1:%d", freeTCPPort(t))
	nodeA := startStandNode(t, p, standNodeAID, "--metrics-addr="+metricsAddr)
	nodeB := startStandNode(t, p, standNodeBID)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	// The send window's end is the schedule's last slot — the anchor the
	// acceptance's "5 s after the end" reads, independent of how long the
	// in-flight drain takes.
	total := redBurstRate * redBurstSeconds
	interval := time.Second / time.Duration(redBurstRate)
	sendEnd := time.Now().Add(time.Duration(total-1) * interval)

	result := redRunBurst(t, p.grpcA)
	achieved := float64(result.ok) / result.elapsed.Seconds()
	if achieved < redBurstMinAchieved {
		errSummary := make([]string, 0, len(result.errs))
		for code, count := range result.errs {
			errSummary = append(errSummary, fmt.Sprintf("%s=%d", code, count))
		}
		slices.Sort(errSummary)
		t.Fatalf("burst achieved %.1f writes/s over %v (%d/%d writes OK), want at least %d writes/s (XPORT-02(3)); errors by code: %s",
			achieved, result.elapsed, result.ok, total, redBurstMinAchieved, strings.Join(errSummary, ", "))
	}

	// The unacked set must empty within the documented 5 s after the send
	// window's end, plus the poll margin. A schedule that outlived its own
	// drain deadline (the drain wait starting already-expired) fails
	// immediately — the drain took too long, which is the breach.
	drainTimeout := time.Until(sendEnd.Add(redBurstDrainBound + redBurstDrainMargin))
	waitFor(t, drainTimeout, fmt.Sprintf(
		"node A's unacked set empty within %v of the burst's send window end (XPORT-02(3))",
		redBurstDrainBound,
	), redThrottled(func() bool {
		value, found := redMetricValue(redMetricsBody(t, metricsAddr), "supkv_replication_unacked_records")
		return found && value == 0
	}))

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the burst node A")
	assertNoPanic(t, nodeB, "the burst peer node B")
}

// The R-03 (PROTO-01, ADR 0005 / decision 14.5) scenario's shape. The
// listing must span MANY DataSync datagrams so "the full listing" is a
// real multi-datagram completeness question: the core's chunker budgets
// 1400 B per chunk with a len(key)+24 per-entry estimate
// (internal/app/repl/core.go's SyncChunkBytes and entryOverheadBytes), so
// a 16-character key packs roughly 32 entries per chunk — the default 512
// keys is ~16 chunks, the nightly 4096 ~128, every chunk one datagram.
const (
	// redR03KeysDefault is the local scale: many chunks, a viable local
	// red run. redR03KeysNightly is the scheduled run's larger listing;
	// both breach v1.0 for the SAME structural reason (the gate opens on
	// the first chunk and reads never consult it), so the nightly scale
	// only widens the listing.
	redR03KeysDefault = 512
	redR03KeysNightly = 4096
	redR03KeyFmt      = "red/r03/key-%04d"

	// The F14 keys: the peer-known user value the replaced node must not
	// overwrite, the default it seeds, and the standalone key whose
	// version is the listing's maximum.
	redR03MarkerKey   = "cfg/color"
	redR03MarkerValue = "user-value"
	redR03SeedValue   = "default"
	redR03MaxKey      = "red/r03/zz-max-version"
	redR03ProbeKey    = "red/r03/closed-probe"

	// redR03StartupGate pushes v1.0's one-second gate backstop far beyond
	// the closed-interface assertion window: the replaced node must open
	// on the COMPLETE listing, never on a timer. A node with data on disk
	// opens via startup_gate under PROTO-02; an empty-disk node waits for
	// its separate, longer parameter — either way the backstop cannot
	// race the closed-side assertions here.
	redR03StartupGate = 30 * time.Second
)

// redR03Keys returns R-03's listing scale: the nightly hook selects the
// larger multi-chunk listing; both scales span many DataSync datagrams.
func redR03Keys() int {
	if standNightly {
		return redR03KeysNightly
	}
	return redR03KeysDefault
}

// redVersion converts a wire version field to the typed RecordVersion,
// failing loudly on any other length (the wire contract is 16 bytes).
func redVersion(t *testing.T, raw []byte) version.RecordVersion {
	t.Helper()
	if len(raw) != len(version.RecordVersion{}) {
		t.Fatalf("wire version is %d bytes, want %d", len(raw), len(version.RecordVersion{}))
	}
	var v version.RecordVersion
	copy(v[:], raw)
	return v
}

// redAssertCode fails unless err carries want — the code assertion the
// closed-interface acceptance needs. The harness's standGet treats any
// non-NotFound error as fatal, so the red tests read raw errors here.
func redAssertCode(t *testing.T, what string, err error, want codes.Code) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Fatalf("%s: want code %v, got %v (err=%v)", what, want, got, err)
	}
}

// redGetRecord reads one record through the raw client, preserving the
// error for the acceptance's code assertions (standGet would fatal on any
// non-NotFound error, which is exactly what the closed-read assertion
// expects instead).
func redGetRecord(t *testing.T, client apiproto.SupKVServiceClient, key string) (*apiproto.Record, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	defer cancel()
	resp, err := client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err != nil {
		return nil, fmt.Errorf("Get(%q): %w", key, err)
	}
	return resp.GetRecord(), nil
}

// redHealthStatus polls the health endpoint once for its serving status;
// ok is false while the server is not answering yet, so the caller's
// waitFor deadline governs, never a fast failure.
func redHealthStatus(t *testing.T, addr string) (healthpb.HealthCheckResponse_ServingStatus, bool) {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, false
	}
	defer func() { _ = conn.Close() }()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, false
	}
	return resp.GetStatus(), true
}

// awaitHealthAnswer waits until the health endpoint answers at all — any
// status — the "gRPC is listening" signal. awaitServing is the wrong wait
// here: under PROTO-01 the gate must stay closed before the full listing,
// so waiting for SERVING would be exactly what the closed side denies.
func awaitHealthAnswer(t *testing.T, addr string, timeout time.Duration) healthpb.HealthCheckResponse_ServingStatus {
	t.Helper()
	var serving healthpb.HealthCheckResponse_ServingStatus
	waitFor(t, timeout, "the gRPC health endpoint answering on "+addr, func() bool {
		got, ok := redHealthStatus(t, addr)
		if !ok {
			return false
		}
		serving = got
		return true
	})
	return serving
}

// TestRed_R03_f14ReplacedNode pins review finding R-03 at stand level as
// the FULL observable PROTO-01 acceptance (ADR 0005 / decision 14.5,
// REVIEW.md §5 R-03; D-01 env gate). A replaced node — empty disk, the F14
// shape — must keep the whole interface closed, reads and writes alike,
// until the neighbor's COMPLETE key listing has arrived, and only then
// serve. v1.0 fails by construction: its Get never consults the gate
// (reads are served from the first moment) and its gate opens on ANY first
// DataSync chunk. F14 measured the consequence — B's create_if_absent with
// a default succeeded right after the premature opening and three seconds
// later BOTH nodes held the default instead of the user value
// (docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md F14; REVIEW.md
// §5 R-03); the first closed-side read assertion below is the
// deterministic red verdict.
//
// The closed window is shaped, not raced: node A is seeded (including the
// user value and a multi-datagram listing) and stopped, the replaced node
// B starts ALONE with a long startup-gate backstop, the closed side is
// asserted while no listing can arrive at all, and only then is A
// restarted so its full multi-chunk listing reaches B. Every assertion is
// the documented acceptance expressed against the configured flags, so the
// Phase 4 flip changes only the expectation-table row: reads and writes
// answer Unavailable before the full listing, health stays NOT_SERVING
// until it, create_if_absent answers AlreadyExists for a peer-known key
// after it, the peer's user value survives the replaced node's seeding
// attempt, and the first local write after the part carrying the max
// version takes a greater version. SyncID/ChunkIndex wire-format
// assertions are Phase 4's own tests — they are not externally observable
// today. The P-01 value-limit red tests are plan 02-12's, not this
// scenario's.
func TestRed_R03_f14ReplacedNode(t *testing.T) { //nolint:paralleltest // a sequential node lifecycle with wall-clock windows: a parallel red scenario would corrupt the closed-side window (file doc).
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	p := newStandPair(t)
	rate := "--sync-response-rate=" + strconv.Itoa(redSyncResponseRate)
	keys := redR03Keys()

	// Node A alone: the peer-known state — the F14 user value plus the
	// multi-chunk listing — then a clean stop, so B's closed window has no
	// live peer at all.
	nodeA := startStandNode(t, p, standNodeAID, rate)
	awaitServing(t, p.grpcA, servingDeadline)
	clientA := standClient(t, p.grpcA)
	standSetWith(t, clientA, redR03MarkerKey, redR03MarkerValue)
	seedKeys := make([]string, keys)
	for i := range seedKeys {
		seedKeys[i] = fmt.Sprintf(redR03KeyFmt, i)
	}
	redSeedKeys(t, redSeedClients(t, p.grpcA), seedKeys)

	// One synchronous write last: the store serializes writes and the HLC
	// is monotonic, so this key carries A's maximum version — the version
	// the listing part with the max version delivers.
	standSetWith(t, clientA, redR03MaxKey, redR03SeedValue)
	maxRec, err := redGetRecord(t, clientA, redR03MaxKey)
	if err != nil {
		t.Fatalf("Get(%q) on the seeded node A: want the record, got %v", redR03MaxKey, err)
	}
	maxVersion := redVersion(t, maxRec.GetVersion())
	signalAndWaitExit(t, nodeA)

	// Node B is the replaced node: empty disk, A down, long backstop.
	nodeB := startStandNode(t, p, standNodeBID,
		rate,
		"--startup-gate="+redR03StartupGate.String(),
	)
	if serving := awaitHealthAnswer(t, p.grpcB, servingDeadline); serving != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health on the replaced node B before any listing: want NOT_SERVING, got %v (PROTO-01: the interface is closed until the full listing)", serving)
	}
	clientB := standClient(t, p.grpcB)

	// (a) Before the full listing — here, before any listing at all — the
	// whole interface is closed, reads and writes alike. v1.0's Get serves
	// the empty store and answers NotFound, never Unavailable.
	_, err = redGetRecord(t, clientB, redR03MarkerKey)
	redAssertCode(t, "Get on the replaced node B before any listing (PROTO-01: reads stay closed until the full list)", err, codes.Unavailable)

	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	_, err = clientB.Set(ctx, &apiproto.SetRequest{Key: redR03ProbeKey, Value: []byte(redR03SeedValue)})
	cancel()
	redAssertCode(t, "Set on the replaced node B before any listing", err, codes.Unavailable)
	ctx, cancel = context.WithTimeout(t.Context(), rpcTimeout)
	_, err = clientB.Delete(ctx, &apiproto.DeleteRequest{Key: redR03ProbeKey})
	cancel()
	redAssertCode(t, "Delete on the replaced node B before any listing", err, codes.Unavailable)

	// The F14 seeding attempt itself: the replaced node must not be able
	// to seed its default over the peer-known key before the full listing.
	ctx, cancel = context.WithTimeout(t.Context(), rpcTimeout)
	_, err = clientB.Set(ctx, &apiproto.SetRequest{
		Key:            redR03MarkerKey,
		Value:          []byte(redR03SeedValue),
		CreateIfAbsent: true,
	})
	cancel()
	redAssertCode(t, "the replaced node's create_if_absent seeding of the peer-known key before any listing", err, codes.Unavailable)

	// (c) Health still NOT_SERVING at the end of the closed-side window.
	if serving, ok := redHealthStatus(t, p.grpcB); !ok || serving != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health on the replaced node B after the closed-side assertions: want NOT_SERVING, got %v (ok=%t)", serving, ok)
	}

	// Node A returns with its dump: the multi-chunk listing starts. Under
	// PROTO-01 the gate opens only on the COMPLETE listing — never on the
	// first chunk.
	nodeA = startStandNode(t, p, standNodeAID, rate)
	awaitServing(t, p.grpcA, servingDeadline)
	listBound := standStartupGate + 2*standSyncInterval + 2*standHelloInterval + 3*time.Second
	waitFor(t, listBound, fmt.Sprintf(
		"health SERVING on B once the %d-key listing (many DataSync datagrams) has fully arrived", keys,
	), redThrottled(func() bool {
		serving, ok := redHealthStatus(t, p.grpcB)
		return ok && serving == healthpb.HealthCheckResponse_SERVING
	}))

	// (d) After the full listing, create_if_absent for the peer-known key
	// answers AlreadyExists — before its value has necessarily arrived, the
	// remembered listing alone carries the answer (PROTO-01 rule 3). v1.0
	// answers OK exactly while the value has not arrived yet, which is the
	// F14 seeding defect.
	ctx, cancel = context.WithTimeout(t.Context(), rpcTimeout)
	_, err = clientB.Set(ctx, &apiproto.SetRequest{
		Key:            redR03MarkerKey,
		Value:          []byte(redR03SeedValue),
		CreateIfAbsent: true,
	})
	cancel()
	redAssertCode(t, "create_if_absent for a peer-known key after the full listing (PROTO-01 rule 3)", err, codes.AlreadyExists)

	// The user value converges to B within the documented paced bound —
	// N records at one limiter-paced reply each — plus margin.
	paced := time.Duration(keys) * (time.Second / time.Duration(redSyncResponseRate))
	margin := standStartupGate + 2*standSyncInterval + 5*time.Second
	waitFor(t, paced+margin, fmt.Sprintf(
		"the replaced node B serving the peer's user value %q within the paced bound (N/rate %v + margin %v)",
		redR03MarkerValue, paced, margin,
	), redThrottled(func() bool {
		rec, getErr := redGetRecord(t, clientB, redR03MarkerKey)
		return getErr == nil && !rec.GetDeleted() && string(rec.GetValue()) == redR03MarkerValue
	}))

	// (b) The seeding attempt never changed the peer's user value, on
	// either node — the F14 overwrite did not happen.
	recA, err := redGetRecord(t, clientA, redR03MarkerKey)
	if err != nil {
		t.Fatalf("Get(%q) on node A after convergence: want the user value, got %v", redR03MarkerKey, err)
	}
	if got := string(recA.GetValue()); got != redR03MarkerValue {
		t.Fatalf("node A's %q after the replaced node's create_if_absent attempt: want %q, got %q (the F14 overwrite)",
			redR03MarkerKey, redR03MarkerValue, got)
	}
	recB, err := redGetRecord(t, clientB, redR03MarkerKey)
	if err != nil || string(recB.GetValue()) != redR03MarkerValue {
		t.Fatalf("node B's %q after convergence: want %q, got (%q, %v)", redR03MarkerKey, redR03MarkerValue, recB.GetValue(), err)
	}

	// (e) The first local write after the listing part carrying the max
	// version takes a greater version — the HLC-update half of PROTO-01
	// rule 4, asserted against the peer's maximum version.
	ctx, cancel = context.WithTimeout(t.Context(), rpcTimeout)
	setResp, err := clientB.Set(ctx, &apiproto.SetRequest{Key: redR03ProbeKey, Value: []byte(redR03MarkerValue)})
	cancel()
	if err != nil {
		t.Fatalf("the first local write on the opened replaced node B: want OK, got %v", err)
	}
	if localVersion := redVersion(t, setResp.GetVersion()); localVersion.Compare(maxVersion) <= 0 {
		t.Fatalf("B's first local write after the listing has version %x, want greater than the listing's max version %x (PROTO-01 rule 4)",
			localVersion, maxVersion)
	}

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the F14 seeded node A")
	assertNoPanic(t, nodeB, "the replaced node B")
}

// The P-01 (decision 14.11 / ADR 0011) scenario's constants: the review's
// F15 flag value — an ACCEPTED --value-max-bytes above the codec's frozen
// 61 440-byte decode bound — the reviewer's 63 000-byte record and the
// acceptance's exact-limit boundary case (a value of exactly
// value_max_bytes). Both sizes pass the store's ValueMaxBytes check and
// both are discarded by the decode bound today.
const (
	redP01ValueMax    = 64000
	redP01ReviewBytes = 63000
	redP01KeyFmt      = "red/p01/value-%d"
)

// TestRed_P01_valueLimit pins review finding P-01 at process level as an
// expected-red test (D-01): a write confirmed to the client must survive a
// restart at any ACCEPTED --value-max-bytes value (decision 14.11 /
// ADR 0011; REVIEW.md §6 P-01: a record of exactly value_max_bytes survives
// a restart at any accepted flag value). The decoder's 61 440-byte bound is
// not the flag: with --value-max-bytes=64000 the store accepts a 63 000-byte
// value and one of exactly 64 000 bytes, the engine writes the record into
// the dump and confirms the Set (PERS-04), and the next start's Decode
// discards it as oversize — the key silently disappears. The review's F15
// measured exactly that: Set OK, then "after restart the key is gone" with
// `discarded 1 undecodable dumped records at startup`. The test asserts the
// documented acceptance (the full value comes back after a restart), so
// Phase 5's single-sourced limit changes only the expectation-table row.
//
// Both subtests restart the same node over the same dump path by SIGKILL:
// the dump is what the client's Set was confirmed against, so the kill
// cannot be blamed for a loss — the loss is the loader's.
func TestRed_P01_valueLimit(t *testing.T) { //nolint:paralleltest // two sequential SIGKILL/restart node lifecycles: the wall-clock windows and the shared harness stay sequential by the file contract.
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	for _, size := range []int{redP01ReviewBytes, redP01ValueMax} { //nolint:paralleltest // the subtests are one sequential SIGKILL/restart lifecycle over the real stand, not independent parallel units (file doc).
		t.Run(fmt.Sprintf("%d-bytes", size), func(t *testing.T) {
			redAssertP01SurvivesRestart(t, size)
		})
	}
}

// redAssertP01SurvivesRestart drives one P-01 scenario: a node with the
// review's accepted --value-max-bytes confirms a value of size bytes,
// takes SIGKILL, and restarts over the same dump; the acceptance is that
// Get returns the full value. The dump assertion between the two phases
// separates the writer's persistence from the loader's loss.
func redAssertP01SurvivesRestart(t *testing.T, size int) {
	t.Helper()
	p := newStandPair(t)
	limit := "--value-max-bytes=" + strconv.Itoa(redP01ValueMax)
	key := fmt.Sprintf(redP01KeyFmt, size)
	value := strings.Repeat("v", size)

	node := startStandNode(t, p, standNodeAID, limit)
	awaitServing(t, p.grpcA, servingDeadline)
	standSetWith(t, standClient(t, p.grpcA), key, value)

	// Set returned, so the dump covering the record completed (PERS-04):
	// the record is on disk before the kill.
	dump := readFinalDump(t, p.dumpA)
	if !strings.Contains(dump, `"key":"`+key+`"`) {
		t.Fatalf("the confirmed %d-byte record is not in the dump %s after Set returned", size, p.dumpA)
	}
	killAndReap(t, node)

	// Restart the SAME node — same identity, same ports, same dump path.
	node = startStandNode(t, p, standNodeAID, limit)
	awaitServing(t, p.grpcA, servingDeadline)
	got, found := standGet(t, standClient(t, p.grpcA), key)
	if !found {
		t.Fatalf("the %d-byte value confirmed with --value-max-bytes=%d is gone after restart: the dump holds the record but the loader discards it (P-01, decision 14.11 / ADR 0011)",
			size, redP01ValueMax)
	}
	if got != value {
		t.Fatalf("the %d-byte value after restart = %d bytes, want the full value (P-01)", size, len(got))
	}

	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the restarted P-01 node")
}
