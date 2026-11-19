package grpcapi_test

import (
	"bytes"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

// TestRed_R03_gateClosedUntilFullList pins review finding R-03 at the gate
// level as an expected-red test (D-01): R-03 / decision 14.5 (ADR 0005) —
// interface closed until the full key list arrives. The PROTO-01 acceptance
// (REVIEW.md §5 R-03) says the whole interface — reads and writes — stays
// closed until the node has received the FULL key list with versions from a
// neighbor, while v1.0's RunGate opens on ANY first DataSync chunk (the A1
// mirror semantics: any received datagram counts) and its Get serves reads
// even while the gate is closed. The test therefore fails against v1.0 code
// by construction, and it is written as the FULL acceptance assertions so
// the Phase 4 flip needs no edit here — only the runner's expectation-table
// row changes (red → green).
//
// Gating (D-01): the body skips unless SUPKV_RED_TESTS=1 — a closed gate is
// the only way the red test coexists with the normal `make test` gates;
// scripts/red-runner sets the variable, asserts the FAIL event via
// `go test -json` and records the evidence.
func TestRed_R03_gateClosedUntilFullList(t *testing.T) {
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	t.Parallel()

	// DataSync arrival states the fake models: none, the first chunk of a
	// multi-part listing (partial), the full listing.
	const (
		redSyncNone    int32 = 0
		redSyncPartial int32 = 1
		redSyncFull    int32 = 2
	)

	// release holds the gate resolver's first poll until the closed side of
	// the acceptance has been asserted — the TestGate_opensOnTimeout
	// determinism pattern. While the poll is parked the gate cannot open
	// (the backstop is an hour away), so every closed-side assertion below
	// is race-free even under full-suite parallel load.
	release := make(chan struct{})

	// syncState carries how much of the neighbor's DataSync listing this
	// node has received; the test goroutine writes it before the resolver
	// can ever read it (every read sits behind <-release).
	var syncState atomic.Int32

	ts := newGateServer(t, grpcapi.ServerOptions{
		SyncArrived: func() bool {
			<-release
			// v1.0 mirror semantics (gate.go, A1: any received DataSync
			// datagram counts): the first chunk of a multi-part listing
			// already reports true — the exact defect R-03 pins. Under
			// PROTO-01 the mirror's production meaning becomes "the full
			// key list has arrived", and the resolver may only open on a
			// complete listing; the assertions below are written so that
			// Phase 4's flip changes no line of this test.
			return syncState.Load() != redSyncNone
		},
		StartupGate: gateTestNeverBackstop,
	})
	ctx := t.Context()

	// The partial chunk's payload applies to the engine exactly as a
	// received DataSync chunk's contents would — the same seam
	// TestGate_writesUnavailableReadsOK uses for peer writes arriving
	// during the gate — and the chunk's arrival is what the mirror counts.
	nowMs := time.Now().UnixMilli()
	partialRecord := storage.Record{
		Version: version.PackVersion(nowMs+2, 0, 0x0BBB),
		Data:    []byte("user-value"),
	}
	_, _, err := ts.engine.ApplyRemote("cfg/color", partialRecord)
	if err != nil {
		t.Fatalf("ApplyRemote of the partial chunk's record: want nil, got error: %v", err)
	}
	syncState.Store(redSyncPartial)

	// Closed side of the PROTO-01 acceptance: with only a partial listing
	// arrived the whole interface — reads and writes alike — is closed.
	// v1.0 fails the first assertion twice over: its resolver opens the
	// gate on the chunk's bare arrival, and its Get handler serves reads
	// even while the gate is closed (server.go: reads never consult the
	// gate).
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "cfg/color"})
	assertCode(t, "Get while only a partial DataSync listing has arrived", err, codes.Unavailable)
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "cfg/color", Value: []byte("operator-value")})
	assertCode(t, "Set while only a partial DataSync listing has arrived", err, codes.Unavailable)
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "cfg/color"})
	assertCode(t, "Delete while only a partial DataSync listing has arrived", err, codes.Unavailable)
	health, err := ts.health.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health while only a partial DataSync listing has arrived: want OK, got error: %v", err)
	}
	if health.GetStatus() != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health while only a partial DataSync listing has arrived: want NOT_SERVING, got %v", health.GetStatus())
	}

	// Open side of the same acceptance: the remaining chunks arrive — the
	// full key listing is in — and the gate opens on it. Get now serves the
	// record the listing delivered, and health mirrors the opening (API-06
	// 1:1 binding).
	syncState.Store(redSyncFull)
	close(release)
	awaitGateOpen(t, ts)

	get, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: "cfg/color"})
	if err != nil {
		t.Fatalf("Get after the full listing arrived: want OK, got error: %v", err)
	}
	if !bytes.Equal(get.GetRecord().GetValue(), []byte("user-value")) {
		t.Fatalf("Get after the full listing arrived: want %q, got %q", "user-value", get.GetRecord().GetValue())
	}
	health, err = ts.health.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health after the full listing arrived: want OK, got error: %v", err)
	}
	if health.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("health after the full listing arrived: want SERVING, got %v", health.GetStatus())
	}
}
