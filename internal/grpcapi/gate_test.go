package grpcapi_test

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	"github.com/Djarvur/sup-kv/internal/grpcapi/apiproto"
)

const (
	// gateSyncFlipDelay is how long the sync fake in the opens-on-sync and
	// health tests stays false before flipping — long enough that the gate
	// is observably closed first, short enough that the test stays fast.
	gateSyncFlipDelay = 50 * time.Millisecond

	// gateTestBackstop is the short StartupGate the opens-on-timeout test
	// runs under: the gate must open on the deadline alone.
	gateTestBackstop = 50 * time.Millisecond

	// gateTestNeverBackstop is the hour-long StartupGate the closed-gate
	// tests run under: nothing opens the gate inside the test window.
	gateTestNeverBackstop = time.Hour
)

// closedGateOptions returns the gate wiring of a node whose peers never
// sync and whose backstop never elapses inside a test: the resolver runs,
// but the gate stays closed for the whole test window.
func closedGateOptions() grpcapi.ServerOptions {
	return grpcapi.ServerOptions{
		SyncArrived: func() bool { return false },
		StartupGate: gateTestNeverBackstop,
	}
}

// newGateServer builds the gate-test harness: the fake-DumpStore engine
// with the resolver running on the test's ServerOptions. Never waits —
// tests that need an open gate call awaitGateOpen themselves; tests that
// need a closed one pass closedGateOptions.
func newGateServer(t *testing.T, opts grpcapi.ServerOptions) *testHarness {
	t.Helper()
	return buildTestServer(t, testServerConfig{gate: opts})
}

// TestGate_writesUnavailableReadsOK proves the API-04 gate from the closed
// side: with the sync fake never flipping and the StartupGate backstop an
// hour away, Set and Delete return Unavailable — the gate check is the
// handlers' first statement, so even an empty key surfaces Unavailable,
// not InvalidArgument — while Get, List and a Watch subscription are
// served normally. The Watch row rides a record applied through the engine
// seam: peers' writes merging during the gate is exactly the window
// API-04's read service exists for, and the subscriber must see them.
func TestGate_writesUnavailableReadsOK(t *testing.T) {
	t.Parallel()
	ts := newGateServer(t, closedGateOptions())
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/a", Value: []byte("v")})
	assertCode(t, "Set during the gate", err, codes.Unavailable)
	if status.Convert(err).Message() != "startup gate: waiting for first sync round or startup_gate" {
		t.Fatalf("Set during the gate message: want the gate wording, got %q", status.Convert(err).Message())
	}
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "", Value: []byte("v")})
	assertCode(t, "Set empty key during the gate", err, codes.Unavailable)
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "gate/a"})
	assertCode(t, "Delete during the gate", err, codes.Unavailable)

	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "gate/a"})
	assertCode(t, "Get during the gate", err, codes.NotFound)
	list, err := ts.client.List(ctx, &apiproto.ListRequest{})
	if err != nil {
		t.Fatalf("List during the gate: want OK, got error: %v", err)
	}
	if keys := list.GetKeys(); len(keys) != 0 {
		t.Fatalf("List during the gate: want no keys, got %v", keys)
	}

	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{})
	if err != nil {
		t.Fatalf("Watch during the gate: want OK, got error: %v", err)
	}
	nowMs := time.Now().UnixMilli()
	peer := storage.Record{Version: version.PackVersion(nowMs+2, 0, 0x0BBB), Data: []byte("peer-write")}
	_, _, err = ts.engine.ApplyRemote("gate/peer", peer)
	if err != nil {
		t.Fatalf("ApplyRemote during the gate: want nil, got error: %v", err)
	}
	ev := recvEvent(t, stream)
	if ev.GetRecord().GetKey() != "gate/peer" ||
		!bytes.Equal(ev.GetRecord().GetValue(), []byte("peer-write")) {
		t.Fatalf("watch event during the gate = (%s, %q), want (gate/peer, peer-write)",
			ev.GetRecord().GetKey(), ev.GetRecord().GetValue())
	}
}

// TestGate_opensOnSync proves the sync half of the opening rule: with the
// StartupGate backstop an hour away, the flip of the injected SyncArrived
// closure alone opens the gate — the resolver's next poll flips health to
// SERVING and Set succeeds.
func TestGate_opensOnSync(t *testing.T) {
	t.Parallel()
	var sync atomic.Bool
	ts := newGateServer(t, grpcapi.ServerOptions{
		SyncArrived: sync.Load,
		StartupGate: time.Hour,
	})
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/sync", Value: []byte("v")})
	assertCode(t, "Set before the sync arrives", err, codes.Unavailable)

	time.AfterFunc(gateSyncFlipDelay, func() { sync.Store(true) })
	awaitGateOpen(t, ts)

	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/sync", Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set after the sync arrived: want OK, got error: %v", err)
	}
}

// TestGate_opensOnTimeout proves the backstop half of the opening rule:
// with the sync fake never flipping, the gate still opens when
// StartupGate elapses since RunGate started — a partitioned or single node
// becomes writable on the deadline alone.
func TestGate_opensOnTimeout(t *testing.T) {
	t.Parallel()
	ts := newGateServer(t, grpcapi.ServerOptions{
		SyncArrived: func() bool { return false },
		StartupGate: gateTestBackstop,
	})
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/timeout", Value: []byte("v")})
	assertCode(t, "Set before the backstop elapses", err, codes.Unavailable)

	awaitGateOpen(t, ts)
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/timeout", Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set after the backstop elapsed: want OK, got error: %v", err)
	}
}

// TestHealth_reflectsGate proves the API-06 binding on both transitions
// through the standard health client over the same bufconn: the empty
// service name — the whole-server status per the health protocol — reads
// NOT_SERVING while the gate is closed and SERVING after the resolver
// opens it.
func TestHealth_reflectsGate(t *testing.T) {
	t.Parallel()
	var sync atomic.Bool
	ts := newGateServer(t, grpcapi.ServerOptions{
		SyncArrived: sync.Load,
		StartupGate: time.Hour,
	})
	ctx := t.Context()

	resp, err := ts.health.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Check before the gate opens: want OK, got error: %v", err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health before the gate opens: want NOT_SERVING, got %v", resp.GetStatus())
	}

	sync.Store(true)
	awaitGateOpen(t, ts)

	resp, err = ts.health.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Check after the gate opens: want OK, got error: %v", err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("health after the gate opens: want SERVING, got %v", resp.GetStatus())
	}
}
