package grpcapi_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
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

// assertWatchCode checks the status code a Watch subscription ends with: a
// server-streaming call may surface the handler's rejection either from the
// call itself or from the first receive, and both are the same verdict. The
// bounded receive keeps a mis-behaving handler from hanging the test.
func assertWatchCode(t *testing.T, ts *testHarness, what string, want codes.Code) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), gateWait)
	defer cancel()
	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{})
	if err != nil {
		assertCode(t, what, err, want)
		return
	}
	_, err = stream.Recv()
	assertCode(t, what, err, want)
}

// TestGate_readsAndWritesUnavailable proves the API-04/D-01 gate from the
// closed side: with the sync fake never flipping and the StartupGate
// backstop an hour away, every data RPC — Get, List, Watch, Set, Delete,
// SnapshotCreate, SnapshotLoad — answers Unavailable, and health mirrors
// the closed state as NOT_SERVING (API-06). The non-nil ListedKey seam
// claims every key is listed and changes nothing: the gate check is the
// handlers' first statement, before any listing knowledge or request
// validation.
func TestGate_readsAndWritesUnavailable(t *testing.T) {
	t.Parallel()
	opts := closedGateOptions()
	opts.ListedKey = func(string) bool { return true }
	ts := newGateServer(t, opts)
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/a", Value: []byte("v")})
	assertCode(t, "Set during the gate", err, codes.Unavailable)
	if status.Convert(err).Message() != "startup gate: waiting for first sync round or startup_gate" {
		t.Fatalf("Set during the gate message: want the gate wording, got %q", status.Convert(err).Message())
	}
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "gate/a"})
	assertCode(t, "Delete during the gate", err, codes.Unavailable)

	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "gate/a"})
	assertCode(t, "Get during the gate", err, codes.Unavailable)
	_, err = ts.client.List(ctx, &apiproto.ListRequest{})
	assertCode(t, "List during the gate", err, codes.Unavailable)
	assertWatchCode(t, ts, "Watch during the gate", codes.Unavailable)

	_, err = ts.client.SnapshotCreate(ctx, &apiproto.SnapshotCreateRequest{Name: "gated"})
	assertCode(t, "SnapshotCreate during the gate", err, codes.Unavailable)
	_, err = ts.client.SnapshotLoad(ctx, &apiproto.SnapshotLoadRequest{Name: "gated"})
	assertCode(t, "SnapshotLoad during the gate", err, codes.Unavailable)

	resp, err := ts.health.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Check during the gate: want OK, got error: %v", err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health during the gate: want NOT_SERVING, got %v", resp.GetStatus())
	}
}

// TestGate_emptyKeyStillUnavailable proves the gate check precedes request
// validation: a Get with an empty key while closed is Unavailable, not
// InvalidArgument — the A7 empty-key check runs only past the gate.
func TestGate_emptyKeyStillUnavailable(t *testing.T) {
	t.Parallel()
	ts := newGateServer(t, closedGateOptions())
	ctx := t.Context()

	_, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: ""})
	assertCode(t, "Get empty key during the gate", err, codes.Unavailable)
}

// TestGate_openRestoresReads proves the open side: once the sync flip opens
// the gate, Get and List serve the engine's answers again — including the
// records peers merged during the closed window.
func TestGate_openRestoresReads(t *testing.T) {
	t.Parallel()
	var sync atomic.Bool
	ts := newGateServer(t, grpcapi.ServerOptions{
		SyncArrived: sync.Load,
		StartupGate: gateTestNeverBackstop,
	})
	ctx := t.Context()

	_, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: "gate/restored"})
	assertCode(t, "Get before the gate opens", err, codes.Unavailable)
	_, err = ts.client.List(ctx, &apiproto.ListRequest{})
	assertCode(t, "List before the gate opens", err, codes.Unavailable)

	peer := storage.Record{
		Version: version.PackVersion(time.Now().UnixMilli()+2, 0, 0x0BBB),
		Data:    []byte("restored"),
	}
	_, _, err = ts.engine.ApplyRemote("gate/restored", peer)
	if err != nil {
		t.Fatalf("ApplyRemote while the gate is closed: want nil, got error: %v", err)
	}

	sync.Store(true)
	awaitGateOpen(t, ts)

	get, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: "gate/restored"})
	if err != nil {
		t.Fatalf("Get after the gate opens: want OK, got error: %v", err)
	}
	if !bytes.Equal(get.GetRecord().GetValue(), []byte("restored")) {
		t.Fatalf("Get after the gate opens: want %q, got %q", "restored", get.GetRecord().GetValue())
	}
	list, err := ts.client.List(ctx, &apiproto.ListRequest{})
	if err != nil {
		t.Fatalf("List after the gate opens: want OK, got error: %v", err)
	}
	if keys := list.GetKeys(); len(keys) != 1 || keys[0] != "gate/restored" {
		t.Fatalf("List after the gate opens: want [gate/restored], got %v", keys)
	}
}

// TestServer_gateOpenListedRace pins Pitfall 2's structural guarantee: the
// adapter publishes completeness and the listed key set in one atomic
// snapshot, so a create_if_absent issued immediately after health reports
// SERVING is already blocked by the listed set — no window exists in which
// the gate is open but the keys are invisible (F14's back door).
func TestServer_gateOpenListedRace(t *testing.T) {
	t.Parallel()

	// One immutable publication carries both answers, exactly like the
	// adapter's listing mirror: readers load the same pointer for
	// completeness and membership.
	type listingMirror struct {
		complete bool
		keys     map[string]struct{}
	}
	var mirror atomic.Pointer[listingMirror]

	ts := buildTestServer(t, testServerConfig{
		gate: grpcapi.ServerOptions{
			SyncArrived: func() bool {
				m := mirror.Load()
				return m != nil && m.complete
			},
			StartupGate: gateTestNeverBackstop,
		},
		listedKey: func(key string) bool {
			m := mirror.Load()
			if m == nil || !m.complete {
				return false
			}
			_, ok := m.keys[key]
			return ok
		},
	})
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "race/key", Value: []byte("local"), CreateIfAbsent: true,
	})
	assertCode(t, "create_if_absent before the gate opens", err, codes.Unavailable)

	// The peer's complete listing arrives: one store publishes the gate's
	// completeness and the key membership together, and the resolver's next
	// poll opens the gate on that same snapshot.
	mirror.Store(&listingMirror{complete: true, keys: map[string]struct{}{"race/key": {}}})
	awaitGateOpen(t, ts)

	_, err = ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "race/key", Value: []byte("local"), CreateIfAbsent: true,
	})
	assertCode(t, "create_if_absent immediately after the gate opens", err, codes.AlreadyExists)
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
//
// Decision 14.6 (ADR 0006): an empty-disk node waits longer for a peer —
// startup_gate_empty is the Phase 4 (PROTO-02) extension of this decision;
// the uniform backstop asserted here is the current shape the extension
// lengthens for the empty case.
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

// healthOneWaySetKey is the key the one-way test writes to prove the gate
// opened: a successful Set is observable proof that openGate ran, hence that
// its health flip was attempted after Shutdown.
const healthOneWaySetKey = "health/one-way"

// TestHealth_shutdownIsOneWay proves the shutdown half of the health
// contract (D-05, APIS-05, T-06-23, RESEARCH Pitfall 9): the gate resolver's
// backstop can fire after shutdown began, and its SERVING flip must be
// ignored. The test calls Shutdown twice (the call is idempotent) before the
// 50 ms backstop elapses, waits for the backstop by observing a successful
// Set — the write proves openGate ran, so the late status flip was attempted
// — and then requires health to still read NOT_SERVING.
func TestHealth_shutdownIsOneWay(t *testing.T) {
	t.Parallel()

	ts := buildTestServer(t, testServerConfig{
		gate: grpcapi.ServerOptions{
			SyncArrived: func() bool { return false },
			StartupGate: gateTestBackstop,
		},
	})
	ctx := t.Context()

	ts.api.Shutdown()
	ts.api.Shutdown()

	waitFor(t, gateWait, "the startup-gate backstop to open the gate", func() bool {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: healthOneWaySetKey, Value: []byte("v")})
		return err == nil
	})

	resp, err := ts.health.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health Check after Shutdown and the late backstop: want OK, got error: %v", err)
	}
	if got := resp.GetStatus(); got != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health after Shutdown: want NOT_SERVING even though the backstop opened the gate, got %v", got)
	}
}

// TestHealth_backstopWithoutShutdownServes is the control for the one-way
// test: the same 50 ms backstop harness without a prior Shutdown reaches
// SERVING, so the one-way assertion observes a real ignored transition
// rather than a gate that never opens.
func TestHealth_backstopWithoutShutdownServes(t *testing.T) {
	t.Parallel()

	ts := buildTestServer(t, testServerConfig{
		gate: grpcapi.ServerOptions{
			SyncArrived: func() bool { return false },
			StartupGate: gateTestBackstop,
		},
	})
	awaitGateOpen(t, ts)
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
