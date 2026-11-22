//go:build linux

// TEST-12 metrics and configuration by observed effect. The phase's metric
// definitions (decision 14.4 / ADR 0004: no conflict journal — the counter IS
// the record) and its opt-in metrics endpoint (decision 14.8 / ADR 0008: no
// listener without --metrics-addr — this test opts in explicitly) are pinned
// by what a live node actually renders, never by reading package state: the
// scenario constructs exactly one conflict and exactly one bad-CRC discard
// and scrapes the node's /metrics over HTTP, asserting the EXACT rendered
// values those constructions fix.
//
// The retransmit half refuses the same exactness honestly: the scenario
// brings node B down and writes on A so the record needs retransmission,
// but the number of retransmit cycles a real socket yields depends on
// scheduler-dependent Ack timing, so that family asserts the named lower
// bound >= 1 (and the test additionally checks the counter climbed above
// the pre-kill baseline, attributing the cycle to the stopped peer).
// Exactness holds where construction makes it possible: conflict == 1 and
// discard{class="bad_crc"} == 1 are single constructed events, retransmit
// is a lower bound on real sockets.
//
// The per-parameter half (TestConfig_parametersByEffect) observes one
// effect per parameter at the binary level. Parameters whose effect is
// only unit-observable are mapped to their unit tests in 02-08-SUMMARY.md's
// parameter table (the acceptance is one test per parameter, not one test
// per package).

package stand //nolint:testpackage // the metrics scenarios share the harness's unexported helpers (D-03/D-08); testpackage's external-package rule cannot express that pairing.

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// The metrics-by-effect scenario's keys, values and families. The two
// effect keys are distinct so neither scenario can contaminate the other;
// the rendered sample strings are the exact family prefixes the /metrics
// scrape carries (labels sorted by name, single-label samples).
const (
	standMetricsConflictKey      = "stand/metrics/conflict-key"
	standMetricsRetransmitKey    = "stand/metrics/retransmit-key"
	standMetricsValueA           = "from-node-a"
	standMetricsValueB           = "from-node-b"
	standMetricsRetransmitValue  = "while-b-is-down"
	standMetricsConflictSample   = "supkv_merges_conflicts_total"
	standMetricsRetransmitSample = "supkv_replication_datagrams_retransmitted_total"
	standMetricsBadCRCSample     = `supkv_datagrams_discarded_total{class="bad_crc"}`
	standMetricsHelloSample      = `supkv_replication_datagrams_sent_total{type="hello"}`
	standMetricsPeersSample      = "supkv_replication_peers"
)

// The scenario's timer knobs, pinned explicitly so every bound derives
// from a value this package sets: a short Ack timeout makes the first
// retransmit cycle quick after B stops, the scan period matches it, the
// long peer timeout keeps B in A's peer table for the whole retransmit
// window (the scanner suspends on an empty table), and the long sync
// interval removes proactive listings from the conflict construction —
// a listing generated before B's displacing write would otherwise be a
// second (rejected-version) conflict on A.
const (
	standMetricsAckTimeout     = 100 * time.Millisecond
	standMetricsAckTimeoutMax  = 200 * time.Millisecond
	standMetricsRetransmitScan = 20 * time.Millisecond
	standMetricsPeerTimeout    = 10 * time.Second
	standMetricsSyncInterval   = 10 * time.Second
)

// The wait bounds: the retransmit counter is observed within a window that
// dwarfs the pinned ladder's first fire, and the generic metrics deadline
// bounds counter-mirroring wait polls.
const (
	standMetricsRetransmitWait = 10 * time.Second
	standMetricsDeadline       = 5 * time.Second
)

// The corrupted-datagram construction: a foreign node id (the CRC verdict
// precedes identity classification, but a foreign id keeps the datagram
// out of the own-echo path for a reader), one flipped body byte, and the
// UDP payload the test's own encoder builds from the production envelope
// fields.
const (
	standMetricsImpostorNodeID = 999
	standMetricsCorruptByte    = 0xFF
)

// errMetricsStatus is the scrape helper's non-OK-status sentinel: the error
// wraps it so the listener-not-ready poll can treat any parse/fetch failure
// uniformly without a dynamic error at the return site.
var errMetricsStatus = errors.New("metrics endpoint returned a non-OK status")

// The per-parameter effect subtests' knobs. Each value is chosen so its
// effect is visible against the production default at the binary level.
const (
	standMetricsStartupGate      = 2 * time.Second
	standMetricsHelloInterval    = 50 * time.Millisecond
	standMetricsHelloWindow      = 500 * time.Millisecond
	standMetricsHelloMinDelta    = 3
	standMetricsRateKeys         = 10
	standMetricsRate             = 10
	standMetricsRateKeyFmt       = "stand/metrics/paced-key-%02d"
	standMetricsRateDeadline     = 10 * time.Second
	standMetricsValueLimit       = 16
	standMetricsValueKey         = "stand/metrics/value-limit-key"
	standMetricsValueAtLimit     = "0123456789abcdef"
	standMetricsValueOverLimit   = "0123456789abcdefg"
	standMetricsPeerEvictTimeout = 500 * time.Millisecond
	standMetricsPeerEvictWait    = 3 * time.Second
	standMetricsPeerEvictMinWait = 250 * time.Millisecond
	standMetricsRateHalfDivisor  = 2
)

// TestStand_metricsByEffect is TEST-12's exact-value half: two nodes, A
// with --metrics-addr, the three effects driven by construction, then the
// EXACT rendered values asserted from A's /metrics over HTTP.
//
// The scenario fixes each value's cardinality: (1) conflict — A writes the
// key, B (clock already past A's version, having applied that write) writes
// a strictly dominating version, and A's remote apply displaces its local
// record: exactly one conflict, so the family asserts ==. (2) discard —
// one hand-built Hello with a correct envelope and one flipped body byte
// reaches A's real UDP port: exactly one bad_crc, so the family asserts ==.
// (3) retransmit — B stops cleanly, A writes a new key, and the pending
// record's backoff ladder fires against the still-remembered peer: the
// family asserts >= 1 and the test checks it climbed past the pre-kill
// baseline; the cycle count is scheduler-dependent Ack timing on real
// sockets, so a lower bound is the honest exact-derivable assertion.
//
// The test is also the referencing link for decisions 14.8 (metrics are
// opt-in: the flag is what turns the listener on) and 14.4 (the conflict
// counter is the record — no journal, and the scrape is the only reader).
func TestStand_metricsByEffect(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	metricsAddr := "127.0.0.1:" + strconv.Itoa(freeTCPPort(t))
	metricsURL := "http://" + metricsAddr + "/metrics"

	nodeA := startStandNode(t, p, standNodeAID,
		"--metrics-addr="+metricsAddr,
		"--ack-timeout="+standMetricsAckTimeout.String(),
		"--ack-timeout-max="+standMetricsAckTimeoutMax.String(),
		"--retransmit-interval="+standMetricsRetransmitScan.String(),
		"--peer-timeout="+standMetricsPeerTimeout.String(),
		"--sync-interval="+standMetricsSyncInterval.String(),
	)
	nodeB := startStandNode(t, p, standNodeBID,
		"--sync-interval="+standMetricsSyncInterval.String(),
	)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)
	standMetricsReady(t, metricsURL)

	// Effect 1 — one displaced local record. A's first value is served by B
	// before B writes, so B's clock has absorbed A's version and B's write
	// strictly dominates it (HLC monotonicity, not wall-clock luck).
	standSetWith(t, clientA, standMetricsConflictKey, standMetricsValueA)
	waitFor(t, standConvergeDeadline, "node B serving A's first value", func() bool {
		return standServesValue(t, clientB, standMetricsConflictKey, standMetricsValueA)
	})
	standSetWith(t, clientB, standMetricsConflictKey, standMetricsValueB)
	waitFor(t, standConvergeDeadline, "node A serving B's displacing value", func() bool {
		return standServesValue(t, clientA, standMetricsConflictKey, standMetricsValueB)
	})
	waitFor(t, standMetricsDeadline, "node A's conflict counter reaching exactly one", func() bool {
		body := standMetricsScrape(t, metricsURL)
		return standMetricsSample(t, body, standMetricsConflictSample) == 1
	})

	// Effect 2 — one corrupted datagram on A's real UDP port.
	standMetricsSendCorruptHello(t, p.udpA)
	waitFor(t, standMetricsDeadline, `node A's bad_crc discard counter reaching exactly one`, func() bool {
		body := standMetricsScrape(t, metricsURL)
		return standMetricsSample(t, body, standMetricsBadCRCSample) == 1
	})

	// Effect 3 — one retransmit cycle against the stopped peer. The baseline
	// attributes the climb to this construction; the rendered family is a
	// running total, so the final assertion is the plan's >= 1 lower bound.
	signalAndWaitExit(t, nodeB)
	baseline := standMetricsSample(t, standMetricsScrape(t, metricsURL), standMetricsRetransmitSample)
	standSetWith(t, clientA, standMetricsRetransmitKey, standMetricsRetransmitValue)
	waitFor(t, standMetricsRetransmitWait, "node A's retransmit counter climbing after node B stopped", func() bool {
		body := standMetricsScrape(t, metricsURL)
		return standMetricsSample(t, body, standMetricsRetransmitSample) > baseline
	})

	body := standMetricsScrape(t, metricsURL)
	conflicts := standMetricsSample(t, body, standMetricsConflictSample)
	badCRC := standMetricsSample(t, body, standMetricsBadCRCSample)
	retransmitted := standMetricsSample(t, body, standMetricsRetransmitSample)
	t.Logf("metrics by effect: conflicts=%v bad_crc=%v retransmitted=%v", conflicts, badCRC, retransmitted)
	if conflicts != 1 {
		t.Fatalf("%s = %v after one constructed displacement, want exactly 1:\n%s",
			standMetricsConflictSample, conflicts, body)
	}
	if badCRC != 1 {
		t.Fatalf("%s = %v after one constructed corrupted datagram, want exactly 1:\n%s",
			standMetricsBadCRCSample, badCRC, body)
	}
	if retransmitted < 1 {
		t.Fatalf("%s = %v after a write needing retransmission to the stopped peer, want at least 1:\n%s",
			standMetricsRetransmitSample, retransmitted, body)
	}

	signalAndWaitExit(t, nodeA)
	assertNoPanic(t, nodeA, "the metrics-by-effect node A")
	assertNoPanic(t, nodeB, "the metrics-by-effect node B")
}

// TestConfig_parametersByEffect is TEST-12's per-parameter half, one named
// subtest per stand-observable parameter: startup-gate (gate-open timing),
// hello-interval (hello cadence against a measured window), sync-response-
// rate (empty-node recovery pacing, the emptyDiskRecovery arithmetic),
// value-max-bytes (the Set limit's accepted/rejected boundary) and
// peer-timeout (a silent peer's eviction from the peer gauge). Every
// subtest observes what changes when the parameter changes; parameters
// with no stand-observable effect are listed with their unit tests in
// 02-08-SUMMARY.md.
func TestConfig_parametersByEffect(t *testing.T) {
	t.Parallel()

	t.Run("startup_gate_delays_gate_open", func(t *testing.T) {
		t.Parallel()
		p := newStandPair(t)
		started := time.Now()
		node := startStandNode(t, p, standNodeAID,
			"--startup-gate="+standMetricsStartupGate.String(),
		)
		// The health endpoint is the gate's 1:1 mirror (API-06): a node
		// whose backstop is 2s must answer NOT_SERVING first. The poll
		// window is half the gate, so observing SERVING first is the
		// regression this assertion exists to catch. The harness derives
		// --startup-gate-empty from this explicit --startup-gate, so the
		// empty node's backstop is the same 2 s; without the derivation the
		// 60 s default would time out the serving wait below.
		waitFor(t, standMetricsStartupGate/standMetricsRateHalfDivisor, "a NOT_SERVING health verdict while the gate is closed", func() bool {
			got, err := standHealthStatus(t, p.grpcA)
			return err == nil && got == healthpb.HealthCheckResponse_NOT_SERVING
		})
		awaitServing(t, p.grpcA, servingDeadline)
		if elapsed := time.Since(started); elapsed < standMetricsStartupGate {
			t.Fatalf("the gate opened %v after launch, want at least the configured --startup-gate=%v",
				elapsed, standMetricsStartupGate)
		}
		signalAndWaitExit(t, node)
		assertNoPanic(t, node, "the startup-gate node")
	})

	t.Run("hello_interval_sets_hello_cadence", func(t *testing.T) {
		t.Parallel()
		p := newStandPair(t)
		metricsAddr := "127.0.0.1:" + strconv.Itoa(freeTCPPort(t))
		metricsURL := "http://" + metricsAddr + "/metrics"
		node := startStandNode(t, p, standNodeAID,
			"--hello-interval="+standMetricsHelloInterval.String(),
			"--metrics-addr="+metricsAddr,
		)
		awaitServing(t, p.grpcA, servingDeadline)
		standMetricsReady(t, metricsURL)

		before := standMetricsSample(t, standMetricsScrape(t, metricsURL), standMetricsHelloSample)
		time.Sleep(standMetricsHelloWindow)
		after := standMetricsSample(t, standMetricsScrape(t, metricsURL), standMetricsHelloSample)
		if delta := after - before; delta < standMetricsHelloMinDelta {
			t.Fatalf("hello sends grew by %v over %v with --hello-interval=%v, want at least %d (the 1s production default can produce at most one)",
				delta, standMetricsHelloWindow, standMetricsHelloInterval, standMetricsHelloMinDelta)
		}
		signalAndWaitExit(t, node)
		assertNoPanic(t, node, "the hello-interval node")
	})

	t.Run("sync_response_rate_paces_recovery", func(t *testing.T) {
		t.Parallel()
		p := newStandPair(t)
		keys := make([]string, standMetricsRateKeys)
		{
			nodeA := startStandNode(t, p, standNodeAID)
			awaitServing(t, p.grpcA, servingDeadline)
			clientA := standClient(t, p.grpcA)
			for i := range keys {
				keys[i] = fmt.Sprintf(standMetricsRateKeyFmt, i)
				standSet(t, clientA, keys[i])
			}
			// Stop A cleanly so its final dump carries the seeded records:
			// the restarted node loads them WITH NO PENDING SET, which is
			// what makes the recovery below paced. A freshly discovered
			// peer receives the sender's pending records as an immediate
			// broadcast (handleHello's discovery flush), so seeding a live
			// node would bypass the rate limiter entirely.
			signalAndWaitExit(t, nodeA)
		}
		nodeA := startStandNode(t, p, standNodeAID)
		awaitServing(t, p.grpcA, servingDeadline)
		clientA := standClient(t, p.grpcA)
		for _, key := range keys {
			if _, found := standGet(t, clientA, key); !found {
				t.Fatalf("the restarted node A does not serve the seeded key %q — the dump did not carry it", key)
			}
		}

		// Node B against the restarted seeded peer, empty disk, with its
		// OWN egress rate pinned: the recovery is driven by B's DataRequest
		// queue — one request per missing key, all enqueued by the listing
		// that also opens B's gate — and the limiter drains them at
		// floor(1s/rate) each (core.go's drainEgress arithmetic, the
		// TestStand_emptyDiskRecovery placement). Measured from B's gate
		// opening, the last of N requests needs at least (N-1) intervals;
		// the assertion halves the derived N/rate bound to absorb the
		// observation point, and the default 500/s would drain the batch
		// in milliseconds.
		nodeB := startStandNode(t, p, standNodeBID,
			"--sync-response-rate="+strconv.Itoa(standMetricsRate),
		)
		awaitServing(t, p.grpcB, servingDeadline)
		clientB := standClient(t, p.grpcB)
		started := time.Now()
		waitFor(t, standMetricsRateDeadline, "the empty node B serving every seeded key", func() bool {
			for _, key := range keys {
				if !standServing(t, clientB, key) {
					return false
				}
			}
			return true
		})
		paced := time.Duration(standMetricsRateKeys) * (time.Second / time.Duration(standMetricsRate))
		lowerBound := paced / standMetricsRateHalfDivisor
		if elapsed := time.Since(started); elapsed < lowerBound {
			t.Fatalf("B converged %d keys in %v with --sync-response-rate=%d, want at least the halved paced bound %v",
				len(keys), elapsed, standMetricsRate, lowerBound)
		}
		signalAndWaitExit(t, nodeA)
		signalAndWaitExit(t, nodeB)
		assertNoPanic(t, nodeA, "the pacing seed node A")
		assertNoPanic(t, nodeB, "the paced recovery node B")
	})

	t.Run("value_max_bytes_rejects_oversize", func(t *testing.T) {
		t.Parallel()
		p := newStandPair(t)
		node := startStandNode(t, p, standNodeAID,
			"--value-max-bytes="+strconv.Itoa(standMetricsValueLimit),
		)
		awaitServing(t, p.grpcA, servingDeadline)
		client := standClient(t, p.grpcA)

		// At the limit accepted, above it ErrValueTooLarge mapped to
		// InvalidArgument — the boundary the parameter moves.
		standSetWith(t, client, standMetricsValueKey, standMetricsValueAtLimit)
		if got, found := standGet(t, client, standMetricsValueKey); !found || got != standMetricsValueAtLimit {
			t.Fatalf("value at --value-max-bytes=%d round-trip = (%q, found=%t), want the accepted limit-sized value",
				standMetricsValueLimit, got, found)
		}
		ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
		defer cancel()
		_, err := client.Set(ctx, &apiproto.SetRequest{
			Key:   standMetricsValueKey,
			Value: []byte(standMetricsValueOverLimit),
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("Set one byte over --value-max-bytes=%d: want InvalidArgument, got %v (err=%v)",
				standMetricsValueLimit, status.Code(err), err)
		}
		if !strings.Contains(status.Convert(err).Message(), "value exceeds the maximum allowed size") {
			t.Fatalf("oversize Set message = %q, want the ErrValueTooLarge mapping", status.Convert(err).Message())
		}
		signalAndWaitExit(t, node)
		assertNoPanic(t, node, "the value-limit node")
	})

	t.Run("peer_timeout_evicts_silent_peer", func(t *testing.T) {
		t.Parallel()
		p := newStandPair(t)
		metricsAddr := "127.0.0.1:" + strconv.Itoa(freeTCPPort(t))
		metricsURL := "http://" + metricsAddr + "/metrics"
		nodeA := startStandNode(t, p, standNodeAID,
			"--peer-timeout="+standMetricsPeerEvictTimeout.String(),
			"--metrics-addr="+metricsAddr,
		)
		nodeB := startStandNode(t, p, standNodeBID)
		awaitServing(t, p.grpcA, servingDeadline)
		awaitServing(t, p.grpcB, servingDeadline)
		standMetricsReady(t, metricsURL)

		waitFor(t, standMetricsDeadline, "node A's peer gauge showing node B", func() bool {
			body := standMetricsScrape(t, metricsURL)
			return standMetricsSample(t, body, standMetricsPeersSample) == 1
		})
		killed := time.Now()
		signalAndWaitExit(t, nodeB)
		// The 3s deadline discriminates against the production 5s default:
		// a configured 500ms timeout evicts the silent peer well inside it.
		waitFor(t, standMetricsPeerEvictWait, "node A's peer gauge dropping to zero after the silent peer's timeout", func() bool {
			body := standMetricsScrape(t, metricsURL)
			return standMetricsSample(t, body, standMetricsPeersSample) == 0
		})
		// The last Hello can be at most one stand hello interval old at the
		// kill, so eviction cannot precede timeout-minus-interval; the
		// lower bound catches an eviction that fires eagerly instead of on
		// the configured timeout.
		if elapsed := time.Since(killed); elapsed < standMetricsPeerEvictMinWait {
			t.Fatalf("the peer evicted %v after the kill, want at least %v (configured --peer-timeout=%v)",
				elapsed, standMetricsPeerEvictMinWait, standMetricsPeerEvictTimeout)
		}
		signalAndWaitExit(t, nodeA)
		assertNoPanic(t, nodeA, "the peer-timeout node A")
		assertNoPanic(t, nodeB, "the peer-timeout node B")
	})
}

// standServesValue reports whether the endpoint currently serves exactly
// value under key — the value-specific convergence poll, mirroring
// standServing's transient-error tolerance.
func standServesValue(t *testing.T, client apiproto.SupKVServiceClient, key, value string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	resp, err := client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err != nil {
		return false
	}
	rec := resp.GetRecord()
	return rec.GetKey() == key && !rec.GetDeleted() && string(rec.GetValue()) == value
}

// standHealthStatus performs one health Check against addr and returns the
// reported status; a transport failure (the listener not up yet) returns
// the error and SERVICE_UNKNOWN, so a poll treats it as not-yet.
func standHealthStatus(t *testing.T, addr string) (healthpb.HealthCheckResponse_ServingStatus, error) {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return healthpb.HealthCheckResponse_SERVICE_UNKNOWN, err
	}
	defer func() { _ = conn.Close() }()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return healthpb.HealthCheckResponse_SERVICE_UNKNOWN, err
	}
	return resp.GetStatus(), nil
}

// standMetricsReady waits until the opt-in listener answers a scrape — the
// awaitServing counterpart for the HTTP surface the metrics tests read.
func standMetricsReady(t *testing.T, url string) {
	t.Helper()
	waitFor(t, servingDeadline, "the /metrics listener responding on "+url, func() bool {
		_, err := standMetricsGET(t.Context(), url)
		return err == nil
	})
}

// standMetricsGET performs one GET against the metrics URL and returns the
// body — the httpGetBody shape, context-carrying so the noctx gate holds.
func standMetricsGET(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("metrics request: %w", err)
	}
	client := &http.Client{Timeout: rpcTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("metrics GET %s: %w", url, err)
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return "", fmt.Errorf("metrics body: %w", readErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("metrics close: %w", closeErr)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s status %d", errMetricsStatus, url, resp.StatusCode)
	}
	return string(body), nil
}

// standMetricsScrape fetches the live scrape and fails the test when the
// listener does not answer.
func standMetricsScrape(t *testing.T, url string) string {
	t.Helper()
	body, err := standMetricsGET(t.Context(), url)
	if err != nil {
		t.Fatalf("scrape %s: %v", url, err)
	}
	return body
}

// standMetricsSample returns one rendered sample's value by its exact
// family-and-label prefix — the metrics-by-effect assertion seam. The
// caller asserts values the scenario's construction fixes; a missing
// sample fails the test rather than passing silently.
func standMetricsSample(t *testing.T, body, sample string) float64 {
	t.Helper()
	for line := range strings.SplitSeq(body, "\n") {
		rest, ok := strings.CutPrefix(line, sample+" ")
		if !ok {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(rest), 64)
		if err != nil {
			t.Fatalf("sample %q rendered non-numeric: %q (err=%v)", sample, line, err)
		}
		return value
	}
	t.Fatalf("the scrape carries no sample %q:\n%s", sample, body)
	return 0
}

// standMetricsSendCorruptHello builds one wire datagram with the test's own
// encoder: a correct envelope (supported proto version, foreign node id, a
// 16-byte run id, the body's real CRC) with one body byte flipped after
// encoding, then sends it to the node's real UDP port. The envelope still
// parses and the CRC no longer covers the body, so the adapter's decode
// classifies it BadCRC — exactly one discard event.
func standMetricsSendCorruptHello(t *testing.T, udpPort int) {
	t.Helper()
	body, err := proto.Marshal(&replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}})
	if err != nil {
		t.Fatalf("marshal Hello body: %v", err)
	}
	runID := [16]byte{0x5A, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	envelope := &replproto.Envelope{
		ProtoVersion: repl.SupportedProtoVersion,
		NodeId:       standMetricsImpostorNodeID,
		RunId:        runID[:],
		Crc32:        crc32.ChecksumIEEE(body),
		Body:         body,
	}
	datagram, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	// The body is the envelope's final field, so the last datagram byte
	// lies inside the CRC-covered region: the envelope parses and the CRC
	// fails — the framing class, not a truncation.
	datagram[len(datagram)-1] ^= standMetricsCorruptByte
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "udp4", "127.0.0.1:"+strconv.Itoa(udpPort))
	if err != nil {
		t.Fatalf("dial node UDP port %d: %v", udpPort, err)
	}
	defer func() { _ = conn.Close() }()
	written, err := conn.Write(datagram)
	if err != nil {
		t.Fatalf("send corrupted datagram to %d: %v", udpPort, err)
	}
	if written != len(datagram) {
		t.Fatalf("corrupted datagram write = %d bytes, want %d", written, len(datagram))
	}
}
