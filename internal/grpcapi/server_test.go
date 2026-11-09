package grpcapi_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	"github.com/Djarvur/sup-kv/internal/grpcapi/apiproto"
	"github.com/Djarvur/sup-kv/internal/port"
)

// testValueLimit is the shrunken value limit the harness runs the store
// under: the design default (61 440) is far above anything the tests write,
// so a small limit makes the value-too-large path reachable with one
// oversized write instead of a 60 KiB payload.
const testValueLimit = 1024

// testFlowWindowSize is the static gRPC stream flow-control window the
// harness client advertises: the transport floor (64 KiB — smaller values
// are ignored, http2_client.go) pinned as STATIC, which disables the BDP
// estimator that would otherwise grow the window on frame arrivals alone
// and let a stalled reader buffer an unbounded burst through bufconn's
// 1 MiB pipe. With the static window, a Watch subscriber that stops
// reading stalls the server handler's Send after at most 64 KiB of
// undelivered events — the deterministic precondition the slow-consumer
// shed test needs; every other test's client reads continuously, so the
// window refills and no unary or streaming test ever notices it.
const testFlowWindowSize = 64 * 1024

// errSaveFailed is the injected Save failure: the fake reports it for every
// Save while failSave is set, driving the engine's ErrDumpFailed path
// without touching a filesystem.
var errSaveFailed = errors.New("fakeDumpStore: injected save failure")

// errFakeSnapshot is what the fake reports for snapshot calls: snapshot
// traffic is staged by a later phase-5 plan, so the harness never provides
// real snapshot generations.
var errFakeSnapshot = errors.New("fakeDumpStore: no snapshot generations in this harness")

// fakeDumpStore is the hand-written in-memory port.DumpStore fake the port
// seam's doc sanctions (internal/port/dump.go): it records every saved
// generation in memory and injects failures — no filesystem, no cleanup,
// fully parallel-safe per harness instance.
type fakeDumpStore struct {
	mu       sync.Mutex
	failSave bool
	saved    [][]byte
}

// Save durably records the generation in memory, or fails while failSave
// is set — the injection point for persist.ErrDumpFailed.
func (f *fakeDumpStore) Save(dump []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failSave {
		return errSaveFailed
	}
	f.saved = append(f.saved, bytes.Clone(dump))
	return nil
}

// Load reports the empty start: every test begins as a fresh node — the
// fallback chain (main, .prev) is startup business no API test needs.
func (*fakeDumpStore) Load(_ func([]byte) error) ([]byte, bool) {
	return nil, false
}

// CreateSnapshot and ReadSnapshot complete the port.DumpStore interface;
// the engine is the only caller and snapshot paths are not exercised by
// this plan's tests.
func (*fakeDumpStore) CreateSnapshot(_ string) error {
	return errFakeSnapshot
}

// ReadSnapshot completes the port.DumpStore interface alongside
// CreateSnapshot.
func (*fakeDumpStore) ReadSnapshot(_ string) ([]byte, error) {
	return nil, errFakeSnapshot
}

// testHarness bundles one running engine, its watch hub, the
// bufconn-dialed client and the health client.
type testHarness struct {
	client       apiproto.SupKVClient
	health       healthpb.HealthClient
	engine       *persist.Engine
	hub          *grpcapi.WatchHub
	dumps        *fakeDumpStore // nil on the file-backed harness
	cancelEngine context.CancelFunc
}

// testServerConfig holds the harness knobs the test families vary: the
// watch hub's per-subscriber buffer, the DumpStore backing the engine and
// the startup-gate wiring.
type testServerConfig struct {
	// watchBuffer is the hub's per-subscriber buffer; 0 means the package
	// default through NewWatchHub.
	watchBuffer int

	// dumps backs the engine; nil means the in-memory fakeDumpStore.
	dumps port.DumpStore

	// gate is the ServerOptions the api server is constructed with. The
	// builder always launches the gate resolver; tests that need an open
	// gate pass always-open wiring and call awaitGateOpen.
	gate grpcapi.ServerOptions

	// waitTimeout, knownPeers and pendingCopy are the wait_for_replication
	// seams (API-05, plan 05-05), merged into the same ServerOptions the
	// gate wiring rides on. waitTimeout zero means NewServer's 1s default
	// (D-09, A2); nil knownPeers/pendingCopy mean zero peers — the
	// single-node configuration reporting acked immediately (D-11).
	waitTimeout time.Duration
	knownPeers  func() uint64
	pendingCopy func() map[string]repl.PendingInfo
}

// openGateOptions returns the gate wiring the open-gate harnesses use: the
// sync fake always answers true, so the resolver opens on its first poll
// (at most gatePollInterval after construction).
func openGateOptions() grpcapi.ServerOptions {
	return grpcapi.ServerOptions{SyncArrived: func() bool { return true }}
}

// newTestServer builds the harness with the default watch hub and an open
// gate.
func newTestServer(t *testing.T) *testHarness {
	t.Helper()
	h := buildTestServer(t, testServerConfig{gate: openGateOptions()})
	awaitGateOpen(t, h)
	return h
}

// newTestServerWithWatchBuffer builds the harness with the watch hub's
// per-subscriber buffer set to subscriberBuffer (0 = the package default
// through NewWatchHub) — the shrink-to-2 knob the overflow tests need to
// force the disconnect path deterministically — and an open gate.
func newTestServerWithWatchBuffer(t *testing.T, subscriberBuffer int) *testHarness {
	t.Helper()
	h := buildTestServer(t, testServerConfig{
		watchBuffer: subscriberBuffer,
		gate:        openGateOptions(),
	})
	awaitGateOpen(t, h)
	return h
}

// buildTestServer assembles the full bufconn harness from the config. The
// hub is constructed FIRST — persist.Open threads the change tap through
// storage.Config (OnChange: hub.Ingest, no persist change) — then the
// engine, then go hub.Run under its own context, cancelled via t.Cleanup:
// a slow or wedged Watch hub must never hold the harness's writes hostage.
// The api server is constructed with the config's gate wiring, its health
// server registered alongside SupKV on the same grpc.Server (API-06's
// registration contract, the one main must copy in 05-06), and the gate
// resolver launched under its own context. Everything else as the tracer
// built it: a real persist engine (test node id, wall-clock now, shrunken
// value limit, dump timings tuned down to 1ms/50ms of real time — bufconn
// and synctest bubbles do not mix), the grpcapi Server with a nil notify
// (the legal single-node configuration, D-11), and a gRPC client dialed
// over bufconn with insecure credentials — mandatory on modern gRPC
// clients even for an in-memory transport. Everything stops via
// t.Cleanup, so every test using the harness can run t.Parallel().
func buildTestServer(t *testing.T, cfg testServerConfig) *testHarness {
	t.Helper()

	var hub *grpcapi.WatchHub
	if cfg.watchBuffer > 0 {
		hub = grpcapi.NewWatchHubWithBuffer(cfg.watchBuffer)
	} else {
		hub = grpcapi.NewWatchHub()
	}

	var fake *fakeDumpStore
	dumps := cfg.dumps
	if dumps == nil {
		fake = &fakeDumpStore{}
		dumps = fake
	}
	eng, _ := persist.Open(
		42,
		time.Now,
		time.Minute,
		dumps,
		storage.Config{ValueMaxBytes: testValueLimit, OnChange: hub.Ingest},
		persist.Config{Debounce: time.Millisecond, MaxWait: 50 * time.Millisecond},
	)
	engCtx, cancelEng := context.WithCancel(t.Context())
	go func() { _ = eng.Run(engCtx) }()
	t.Cleanup(cancelEng)

	hubCtx, cancelHub := context.WithCancel(t.Context())
	go func() { _ = hub.Run(hubCtx) }()
	t.Cleanup(cancelHub)

	// The gate wiring and the wait seams ride one ServerOptions struct:
	// merge the config's wait knobs over the gate knobs before
	// construction (zero/nil values keep the defaults documented above).
	opts := cfg.gate
	opts.WaitTimeout = cfg.waitTimeout
	opts.KnownPeers = cfg.knownPeers
	opts.PendingCopy = cfg.pendingCopy

	api := grpcapi.NewServer(eng, nil, hub, opts)
	gateCtx, cancelGate := context.WithCancel(t.Context())
	t.Cleanup(cancelGate)
	go api.RunGate(gateCtx)

	srv := grpc.NewServer()
	apiproto.RegisterSupKVServer(srv, api)
	healthpb.RegisterHealthServer(srv, api.Health())

	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStaticStreamWindowSize(testFlowWindowSize),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: want nil, got error: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &testHarness{
		client:       apiproto.NewSupKVClient(conn),
		health:       healthpb.NewHealthClient(conn),
		engine:       eng,
		hub:          hub,
		dumps:        fake,
		cancelEngine: cancelEng,
	}
}

// gateWait bounds every wait for a state the gate resolver owes the test,
// and gatePoll is the poll period — the resolver itself polls every
// gatePollInterval, so the observation lag is one resolver interval at
// most.
const (
	gateWait = 2 * time.Second
	gatePoll = 10 * time.Millisecond
)

// waitFor polls cond every gatePoll until it holds or the deadline — the
// smoke waitFor pattern (udprepl/smoke_test.go) — and fails with what on
// expiry.
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(gatePoll)
	}
	t.Fatalf("timed out after %v waiting for %s", timeout, what)
}

// awaitGateOpen blocks until the health endpoint reports SERVING for the
// empty service name — the deterministic wait for the resolver's opening
// poll, since the gate flag itself is unexported and the health endpoint
// is its pinned 1:1 mirror (API-06).
func awaitGateOpen(t *testing.T, h *testHarness) {
	t.Helper()
	waitFor(t, gateWait, "the health endpoint to flip SERVING (gate open)", func() bool {
		resp, err := h.health.Check(t.Context(), &healthpb.HealthCheckRequest{})
		return err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	})
}

// TestServer_setGetRoundTrip proves the tracer slice end-to-end: a Set over
// bufconn returns OK with the full 16-byte RFC 9562 version (D-01 record
// shape), and Get returns the identical value and version. The Set returning
// at all is the write-confirmation contract (PERS-04): the engine released
// the write only after a dump containing it completed through the fake.
func TestServer_setGetRoundTrip(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	ctx := t.Context()

	set, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "svc/a", Value: []byte("hello")})
	if err != nil {
		t.Fatalf("Set svc/a: want OK, got error: %v", err)
	}
	if len(set.GetVersion()) != 16 {
		t.Fatalf("Set version size: want 16 bytes, got %d", len(set.GetVersion()))
	}
	ts.dumps.mu.Lock()
	saved := len(ts.dumps.saved)
	ts.dumps.mu.Unlock()
	if saved == 0 {
		t.Fatal("fakeDumpStore saved generations after a confirmed Set: want at least 1, got 0")
	}

	get, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: "svc/a"})
	if err != nil {
		t.Fatalf("Get svc/a: want OK, got error: %v", err)
	}
	if !bytes.Equal(get.GetRecord().GetValue(), []byte("hello")) {
		t.Fatalf("Get value: want %q, got %q", "hello", get.GetRecord().GetValue())
	}
	if !bytes.Equal(get.GetRecord().GetVersion(), set.GetVersion()) {
		t.Fatalf("Get version: want the Set's version %v, got %v", set.GetVersion(), get.GetRecord().GetVersion())
	}
	if get.GetRecord().GetKey() != "svc/a" {
		t.Fatalf("Get key: want %q, got %q", "svc/a", get.GetRecord().GetKey())
	}
}

// assertCode checks one RPC error's status code against the wanted code.
func assertCode(t *testing.T, what string, err error, want codes.Code) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Fatalf("%s: want code %v, got %v (err=%v)", what, want, got, err)
	}
}

// TestServer_errorMapping walks the full pinned error set through the RPC
// surface, asserting the code on status.Code(err) for every row of the
// mapError table. The dump-failure row additionally checks the message
// wording, distinguishing the mapped persist.ErrDumpFailed from a
// transport-level Unavailable, and proves the PERS-04 memory-retention
// semantics survive the API: the value is still readable via Get after the
// write was reported Unavailable.
func TestServer_errorMapping(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	ctx := t.Context()

	// Empty key is InvalidArgument on every unary handler (A7 pin), before
	// any engine call.
	_, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: ""})
	assertCode(t, "Get empty key", err, codes.InvalidArgument)
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "", Value: []byte("x")})
	assertCode(t, "Set empty key", err, codes.InvalidArgument)
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: ""})
	assertCode(t, "Delete empty key", err, codes.InvalidArgument)

	// NotFound for a missing key.
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "err/missing"})
	assertCode(t, "Get missing key", err, codes.NotFound)
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "err/missing"})
	assertCode(t, "Delete missing key", err, codes.NotFound)

	// InvalidArgument for a value over the harness's shrunken value limit —
	// the store's ErrValueTooLarge mapped through the engine.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{
		Key:   "err/big",
		Value: bytes.Repeat([]byte("x"), testValueLimit+1),
	})
	assertCode(t, "Set oversized value", err, codes.InvalidArgument)

	// AlreadyExists for create_if_absent over a live key.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "err/live", Value: []byte("v1")})
	if err != nil {
		t.Fatalf("Set err/live: want OK, got error: %v", err)
	}
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "err/live", Value: []byte("v2"), CreateIfAbsent: true})
	assertCode(t, "SetIfAbsent over live key", err, codes.AlreadyExists)

	// NotFound for a deleted key — the tombstone hides the record.
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "err/live"})
	if err != nil {
		t.Fatalf("Delete err/live: want OK, got error: %v", err)
	}
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "err/live"})
	assertCode(t, "Get deleted key", err, codes.NotFound)

	// AlreadyExists for create_if_absent over a tombstoned key — a
	// Delete-then-SetIfAbsent cannot resurrect the key.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "err/live", Value: []byte("v3"), CreateIfAbsent: true})
	assertCode(t, "SetIfAbsent over tombstoned key", err, codes.AlreadyExists)

	// Unavailable when the dump fails: the fake's Save fails for one write,
	// the engine reports ErrDumpFailed, and the value stays readable in
	// memory (PERS-04 memory retention survives the API).
	ts.dumps.mu.Lock()
	ts.dumps.failSave = true
	ts.dumps.mu.Unlock()
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "err/dumpfail", Value: []byte("v")})
	assertCode(t, "Set with failing dump", err, codes.Unavailable)
	if got := status.Convert(err).Message(); got != persist.ErrDumpFailed.Error() {
		t.Fatalf("Set with failing dump message: want %q, got %q", persist.ErrDumpFailed.Error(), got)
	}
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "err/dumpfail"})
	if err != nil {
		t.Fatalf("Get after dump failure: want OK (PERS-04 memory retention), got error: %v", err)
	}
	ts.dumps.mu.Lock()
	ts.dumps.failSave = false
	ts.dumps.mu.Unlock()

	// Unavailable via ErrClosed after the engine context is cancelled —
	// every write stranded by Run's exit resolves ErrClosed through the
	// same mapper.
	ts.cancelEngine()
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "err/closed", Value: []byte("v")})
	assertCode(t, "Set after engine close", err, codes.Unavailable)
}

// TestServer_list proves D-12..D-15: List returns the sorted live keys
// under any prefix in one response, tombstoned keys never appear, a prefix
// yields exactly its subset, and the empty prefix enumerates the whole
// live state. slices.Equal is order-sensitive, so the want-list comparison
// pins the strict ascending order.
func TestServer_list(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	ctx := t.Context()

	// Seed keys across two prefixes plus one key that gets tombstoned.
	for _, key := range []string{"cfg/a", "cfg/b", "svc/a", "svc/b", "tmp/x"} {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte("v")})
		if err != nil {
			t.Fatalf("Set %s: want OK, got error: %v", key, err)
		}
	}
	_, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "tmp/x"})
	if err != nil {
		t.Fatalf("Delete tmp/x: want OK, got error: %v", err)
	}

	// Empty prefix returns every live key (D-15): strictly ascending,
	// tombstone absent (D-14), keys only (D-13), single response (D-12).
	all, err := ts.client.List(ctx, &apiproto.ListRequest{})
	if err != nil {
		t.Fatalf("List empty prefix: want OK, got error: %v", err)
	}
	wantAll := []string{"cfg/a", "cfg/b", "svc/a", "svc/b"}
	if got := all.GetKeys(); !slices.Equal(got, wantAll) {
		t.Fatalf("List empty prefix: want %v sorted and tombstone-free, got %v", wantAll, got)
	}

	// A prefix returns exactly its live subset.
	cfg, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "cfg/"})
	if err != nil {
		t.Fatalf("List cfg/ prefix: want OK, got error: %v", err)
	}
	if got := cfg.GetKeys(); !slices.Equal(got, []string{"cfg/a", "cfg/b"}) {
		t.Fatalf("List cfg/ prefix: want [cfg/a cfg/b], got %v", got)
	}

	// A prefix matching nothing returns an empty list, not an error.
	none, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "zzz/"})
	if err != nil {
		t.Fatalf("List unmatched prefix: want OK, got error: %v", err)
	}
	if got := none.GetKeys(); len(got) != 0 {
		t.Fatalf("List unmatched prefix: want 0 keys, got %v", got)
	}
}
