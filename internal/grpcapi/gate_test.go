package grpcapi_test

import (
	"bytes"
	"path/filepath"
	"strings"
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
	"github.com/Djarvur/sup-kv/internal/persistfile"
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
// becomes writable on the deadline alone. The sync fake blocks the
// resolver's first poll until the test has observed the closed gate, so
// the closed-side assertion is deterministic even under full-suite
// parallel load (a naive "Set before" assertion races the very 50ms
// deadline under test); after the release the fake answers false forever,
// so SERVING and the succeeding Set can only be the backstop's work.
func TestGate_opensOnTimeout(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	ts := newGateServer(t, grpcapi.ServerOptions{
		SyncArrived: func() bool {
			<-release
			return false
		},
		StartupGate: gateTestBackstop,
	})
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/timeout", Value: []byte("v")})
	assertCode(t, "Set before the backstop elapses", err, codes.Unavailable)

	close(release)
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

// newFileServer builds the snapshot-test harness: a real persistfile
// FileStore over a temp dir backs the engine, so snapshot names cross the
// production validation — persistfile's validateName is the single
// path-traversal control (T-05-07) and the fake DumpStore deliberately
// does not reproduce it — and round-trips cross real files. The gate is
// open: snapshot RPCs are gated writes, so the snapshot behavior is only
// reachable past it.
func newFileServer(t *testing.T) *testHarness {
	t.Helper()
	h := buildTestServer(t, testServerConfig{
		dumps: persistfile.New(filepath.Join(t.TempDir(), "state.json")),
		gate:  openGateOptions(),
	})
	awaitGateOpen(t, h)
	return h
}

// TestSnapshot_createLoadRoundTrip proves API-07 end-to-end through the
// real persistfile adapter: Set two keys, SnapshotCreate, delete both,
// SnapshotLoad — Get returns the restored values — and a Watch stream
// attached before the load observes the deletes as tombstone events and
// the restore replay as fresh live events: the 05-03 tap fires for the
// replay with no extra wiring, because a load replays ordinary writes.
func TestSnapshot_createLoadRoundTrip(t *testing.T) {
	t.Parallel()
	ts := newFileServer(t)
	ctx := t.Context()

	rows := []struct{ key, val string }{
		{key: "snap/a", val: "v1"},
		{key: "snap/b", val: "v2"},
	}
	for _, row := range rows {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: row.key, Value: []byte(row.val)})
		if err != nil {
			t.Fatalf("Set %s: want OK, got error: %v", row.key, err)
		}
	}

	// The witness stream attaches before the load; its snapshot is the two
	// live records.
	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: "snap/"})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}
	snapA := recvEvent(t, stream)
	snapB := recvEvent(t, stream)
	if snapA.GetRecord().GetKey() != "snap/a" || snapB.GetRecord().GetKey() != "snap/b" {
		t.Fatalf("watch snapshot keys = (%s, %s), want (snap/a, snap/b)",
			snapA.GetRecord().GetKey(), snapB.GetRecord().GetKey())
	}

	_, err = ts.client.SnapshotCreate(ctx, &apiproto.SnapshotCreateRequest{Name: "snap1"})
	if err != nil {
		t.Fatalf("SnapshotCreate snap1: want OK, got error: %v", err)
	}
	for _, row := range rows {
		_, delErr := ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: row.key})
		if delErr != nil {
			t.Fatalf("Delete %s: want OK, got error: %v", row.key, delErr)
		}
	}
	_, err = ts.client.SnapshotLoad(ctx, &apiproto.SnapshotLoadRequest{Name: "snap1"})
	if err != nil {
		t.Fatalf("SnapshotLoad snap1: want OK, got error: %v", err)
	}

	// Reads see the restored state.
	for _, row := range rows {
		get, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: row.key})
		if err != nil {
			t.Fatalf("Get %s after the load: want OK, got error: %v", row.key, err)
		}
		if !bytes.Equal(get.GetRecord().GetValue(), []byte(row.val)) {
			t.Fatalf("Get %s after the load: want %q, got %q", row.key, row.val, get.GetRecord().GetValue())
		}
	}

	// The witness observed the deletes, then the replay, in order.
	for _, row := range rows {
		del := recvEvent(t, stream)
		if del.GetRecord().GetKey() != row.key || !del.GetRecord().GetDeleted() {
			t.Fatalf("watch delete event = (%s, deleted=%t), want (%s, true)",
				del.GetRecord().GetKey(), del.GetRecord().GetDeleted(), row.key)
		}
	}
	lastReplayVer := []byte(nil)
	for _, row := range rows {
		replay := recvEvent(t, stream)
		if replay.GetRecord().GetKey() != row.key || replay.GetRecord().GetDeleted() ||
			!bytes.Equal(replay.GetRecord().GetValue(), []byte(row.val)) ||
			len(replay.GetRecord().GetVersion()) != 16 {
			t.Fatalf("watch replay event = (%s, deleted=%t, %q, version size %d), want the fresh restored record for %s",
				replay.GetRecord().GetKey(), replay.GetRecord().GetDeleted(),
				replay.GetRecord().GetValue(), len(replay.GetRecord().GetVersion()), row.key)
		}
		lastReplayVer = replay.GetRecord().GetVersion()
	}
	// The replay carries fresh versions (D-09): the restored record must
	// dominate the snapshot's own version of the same key (rows end on
	// snap/b, so lastReplayVer is snap/b's fresh version).
	if bytes.Compare(lastReplayVer, snapB.GetRecord().GetVersion()) <= 0 {
		t.Fatalf("replay version %x must dominate the snapshot's %x (fresh-version replay, D-09)",
			lastReplayVer, snapB.GetRecord().GetVersion())
	}
}

// TestSnapshot_invalidName proves the T-05-07 mapping for hostile names:
// the empty string, the parent-dot special case, a traversal attempt and
// an over-length name each return InvalidArgument on both snapshot RPCs.
// persistfile's validateName — the single closed control — rejects every
// one before any filesystem touch, so the adapter-side assertion is only
// the mapped code: the handler never validates or sanitizes itself.
func TestSnapshot_invalidName(t *testing.T) {
	t.Parallel()
	ts := newFileServer(t)
	ctx := t.Context()

	for _, name := range []string{"", "..", "a/b", strings.Repeat("x", 65)} {
		_, err := ts.client.SnapshotCreate(ctx, &apiproto.SnapshotCreateRequest{Name: name})
		assertCode(t, "SnapshotCreate hostile name", err, codes.InvalidArgument)
		_, err = ts.client.SnapshotLoad(ctx, &apiproto.SnapshotLoadRequest{Name: name})
		assertCode(t, "SnapshotLoad hostile name", err, codes.InvalidArgument)
	}
}

// TestSnapshot_gated proves the A5 pin from the closed side: both snapshot
// RPCs are writes — a create forces a confirmed dump, a load replays
// writes through the store — so during a closed gate both return
// Unavailable before any engine call.
func TestSnapshot_gated(t *testing.T) {
	t.Parallel()
	ts := newGateServer(t, closedGateOptions())
	ctx := t.Context()

	_, err := ts.client.SnapshotCreate(ctx, &apiproto.SnapshotCreateRequest{Name: "gated"})
	assertCode(t, "SnapshotCreate during the gate", err, codes.Unavailable)
	_, err = ts.client.SnapshotLoad(ctx, &apiproto.SnapshotLoadRequest{Name: "gated"})
	assertCode(t, "SnapshotLoad during the gate", err, codes.Unavailable)
}
