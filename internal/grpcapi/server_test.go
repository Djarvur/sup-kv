package grpcapi_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	"github.com/Djarvur/sup-kv/internal/port"
)

// testValueLimit is the shrunken value limit the harness runs the store
// under: the design default (61 440) is far above anything the tests write,
// so a small limit makes the value-too-large path reachable with one
// oversized write instead of a 60 KiB payload.
const testValueLimit = 1024

// listedNotArrivedMessage is the pinned 14.5(6) status message for a read of
// a key a peer's complete listing named whose value has not arrived.
const listedNotArrivedMessage = "startup gate: key is listed by a peer but its value has not arrived"

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

// maxTestRecvBytes is the harness client's receive ceiling: the List
// byte-budget test legitimately returns a page up to the 6 MiB serialized
// budget, which the gRPC 4 MiB default client receive limit would reject.
// Plan 06-07 raises the production limits from the same page constants; the
// harness client only has to accept what the budget permits.
const maxTestRecvBytes = 8 << 20

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
// fully parallel-safe per harness instance. A non-nil snapshot backs the
// ReadSnapshot seam the snapshot-load totality tests drive. hold is the
// PERSIST-08 stand-in for a hung fsync: a non-nil channel parks every
// SaveStream until the test releases it, so the engine's confirmation
// deadline expires against a real standing dump.
type fakeDumpStore struct {
	mu       sync.Mutex
	failSave bool
	// discard keeps the completed-generation fact without the bytes: the
	// List cost tests seed hundreds of maximum-size values, and retaining
	// every dump generation would dominate the test's memory.
	discard  bool
	saved    [][]byte
	snapshot []byte
	hold     chan struct{}
}

// SaveStream runs fill into memory and records the generation, or fails
// while failSave is set — the injection point for persist.ErrDumpFailed. A
// hold channel parks the call first, the hung-dumper stand-in.
func (f *fakeDumpStore) SaveStream(fill func(io.Writer) error) error {
	f.mu.Lock()
	hold := f.hold
	f.mu.Unlock()
	if hold != nil {
		<-hold
	}
	var buf bytes.Buffer
	err := fill(&buf)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failSave {
		return errSaveFailed
	}
	if f.discard {
		return nil
	}
	f.saved = append(f.saved, bytes.Clone(buf.Bytes()))
	return nil
}

// holdDumps parks every subsequent SaveStream until releaseDumps runs.
func (f *fakeDumpStore) holdDumps() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hold == nil {
		f.hold = make(chan struct{})
	}
}

// releaseDumps unparks any standing SaveStream; safe when nothing is held.
func (f *fakeDumpStore) releaseDumps() {
	f.mu.Lock()
	hold := f.hold
	f.hold = nil
	f.mu.Unlock()
	if hold != nil {
		close(hold)
	}
}

// Load reports the empty start: every test begins as a fresh node — the
// fallback chain (main, .prev) is startup business no API test needs.
func (*fakeDumpStore) Load(_ time.Time, _ func([]byte) error) port.LoadResult {
	return port.LoadResult{}
}

// CreateSnapshot and ReadSnapshot complete the port.DumpStore interface;
// the engine is the only caller. CreateSnapshot always fails (snapshot
// creation is not exercised by this harness), while ReadSnapshot serves the
// configured snapshot bytes — a hand-built persist.Encode fixture — or the
// pinned no-generations error when the test supplies none.
func (*fakeDumpStore) CreateSnapshot(_ string) error {
	return errFakeSnapshot
}

// ReadSnapshot serves the snapshot fixture a test installed, or the pinned
// no-generations error.
func (f *fakeDumpStore) ReadSnapshot(_ string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshot == nil {
		return nil, errFakeSnapshot
	}
	return bytes.Clone(f.snapshot), nil
}

// notifyCall is one LocalNotify invocation the tests observe: the key, the
// record the handler built (version, data, tombstone flag, checksum), the
// wait_for_replication flag it carried and the write-time peer ids it carried
// (cloned, so a test may keep the call after the handler returns).
type notifyCall struct {
	key   string
	rec   storage.Record
	track bool
	peers []uint16
}

// notifyCapture records LocalNotify calls under a mutex: the handlers run on
// gRPC goroutines, so an unguarded slice would race.
type notifyCapture struct {
	mu    sync.Mutex
	calls []notifyCall
}

// notify is the LocalNotify seam the harness installs; it records and
// accepts every call.
func (c *notifyCapture) notify(key string, rec storage.Record, track bool, peers []uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, notifyCall{key: key, rec: rec, track: track, peers: slices.Clone(peers)})
	return nil
}

// snapshot returns a copy of the calls recorded so far, in call order.
func (c *notifyCapture) snapshot() []notifyCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.calls)
}

// testHarness bundles one running engine, its watch hub, the
// bufconn-dialed client and the health client.
type testHarness struct {
	client       apiproto.SupKVServiceClient
	health       healthpb.HealthClient
	api          *grpcapi.Server
	engine       *persist.Engine
	hub          *grpcapi.WatchHub
	dumps        *fakeDumpStore // nil on the file-backed harness
	cancelEngine context.CancelFunc
	cancelHub    context.CancelFunc
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

	// confirmTimeout is the engine's PERSIST-08 confirmation deadline; zero
	// keeps the 10 s design default.
	confirmTimeout time.Duration

	// valueMaxBytes is the store's value limit; zero keeps the shrunken
	// testValueLimit harness limit. The List cost tests run the production
	// storage.ValueMaxBytes limit so a maximum-size record is the real one.
	valueMaxBytes int

	// discardDumps keeps the fake dump store from retaining each completed
	// generation — the cost tests seed maximum-size values and would
	// otherwise hold every generation in memory.
	discardDumps bool

	// limits, when non-nil, builds the grpc.Server with the explicit
	// production transport options (plan 06-07): the receive and send
	// ceilings plus the keepalive parameters. Nil keeps the plain
	// grpc.NewServer() every pre-06-07 test was written against.
	limits *grpcapi.TransportLimits

	// gate is the ServerOptions the api server is constructed with. The
	// builder always launches the gate resolver; tests that need an open
	// gate pass always-open wiring and call awaitGateOpen.
	gate grpcapi.ServerOptions

	// listedKey is the listing-mirror membership seam (D-01, rule 14.5(6)):
	// it reports whether a peer's complete listing named a key. Nil means no
	// listing knowledge, the single-node configuration.
	listedKey func(key string) bool

	// notify is the LocalNotify seam the server is constructed with; nil is
	// the single-node configuration (D-11). The totality tests install a
	// capturing seam to observe which writes and replays reach replication.
	notify func(key string, rec storage.Record, track bool, peers []uint16) error

	// replayNotify is the paced replay seam (WR-04); nil leaves replay
	// notifications on the live notify seam.
	replayNotify func(key string, rec storage.Record) error

	// waitTimeout, writePeers and waitReplicated are the
	// wait_for_replication seams (API-05, plan 05-05; A-01 in 06-02), merged
	// into the same ServerOptions the gate wiring rides on. waitTimeout zero
	// means NewServer's 1s default (D-09, A2); nil writePeers/waitReplicated
	// mean the single-node configuration, which now reports the honest
	// (acked=false, 0, 0) (A-01).
	waitTimeout    time.Duration
	writePeers     func() []uint16
	waitReplicated func(ctx context.Context, key string, ver version.RecordVersion, total uint32, timeout time.Duration) (acked bool, ackedCount uint32)
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

// newNotifyServer builds the open-gate harness with a notify capture wired
// to the server's LocalNotify seam — the totality tests' observation point.
func newNotifyServer(t *testing.T) (*testHarness, *notifyCapture) {
	t.Helper()
	capture := &notifyCapture{}
	h := buildTestServer(t, testServerConfig{
		gate:   openGateOptions(),
		notify: capture.notify,
	})
	awaitGateOpen(t, h)
	return h, capture
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
// storage.Config (OnChange: hub.Ingest) and the durability watermark through
// persist.Config (OnConfirmed: hub.Confirm) — then the engine, then go hub.Run
// under its own context, cancelled via t.Cleanup (and exposed on the harness
// as cancelHub, so a shutdown test can end the hub deliberately):
// a slow or wedged Watch hub must never hold the harness's writes hostage.
// The api server is constructed with the config's gate wiring and LocalNotify
// seam (nil unless a test installs a capture), its health server registered
// alongside SupKV on the same grpc.Server (API-06's registration contract,
// the one main must copy in 05-06), and the gate resolver launched under its
// own context. Everything else as the tracer built it: a real persist engine
// (test node id, wall-clock now, shrunken value limit, dump timings tuned
// down to 1ms/50ms of real time — bufconn and synctest bubbles do not mix),
// and a gRPC client dialed over bufconn with insecure credentials —
// mandatory on modern gRPC clients even for an in-memory transport.
// Everything stops via t.Cleanup, so every test using the harness can run
// t.Parallel().
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
		fake = &fakeDumpStore{discard: cfg.discardDumps}
		dumps = fake
		// A test may park the dumper; release it at cleanup so the engine
		// goroutine can exit after cancelEng.
		t.Cleanup(fake.releaseDumps)
	}
	valueLimit := cfg.valueMaxBytes
	if valueLimit <= 0 {
		valueLimit = testValueLimit
	}
	eng, _ := persist.Open(
		42,
		time.Now,
		time.Minute,
		dumps,
		storage.Config{ValueMaxBytes: valueLimit, OnChange: hub.Ingest},
		persist.Config{
			Debounce:       time.Millisecond,
			MaxWait:        50 * time.Millisecond,
			ConfirmTimeout: cfg.confirmTimeout,
			// The durability seam the Watch tests observe: a successful dump
			// publishes its covered generation, releasing the held events.
			OnConfirmed: hub.Confirm,
		},
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
	if cfg.listedKey != nil {
		opts.ListedKey = cfg.listedKey
	}
	opts.WaitTimeout = cfg.waitTimeout
	opts.WritePeers = cfg.writePeers
	opts.WaitReplicated = cfg.waitReplicated
	opts.ReplayNotify = cfg.replayNotify

	api := grpcapi.NewServer(eng, cfg.notify, hub, opts)
	gateCtx, cancelGate := context.WithCancel(t.Context())
	t.Cleanup(cancelGate)
	go api.RunGate(gateCtx)

	// The transport envelope is explicit only where a test asks for it
	// (plan 06-07's receive-ceiling and page-fit proofs); every other
	// harness keeps the plain server so its assertions stay about behavior,
	// not transport settings.
	var serverOpts []grpc.ServerOption
	if cfg.limits != nil {
		serverOpts = cfg.limits.ServerOptions()
	}
	srv := grpc.NewServer(serverOpts...)
	apiproto.RegisterSupKVServiceServer(srv, api)
	healthpb.RegisterHealthServer(srv, api.Health())

	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStaticStreamWindowSize(testFlowWindowSize),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxTestRecvBytes)),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: want nil, got error: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &testHarness{
		client:       apiproto.NewSupKVServiceClient(conn),
		health:       healthpb.NewHealthClient(conn),
		api:          api,
		engine:       eng,
		hub:          hub,
		dumps:        fake,
		cancelEngine: cancelEng,
		cancelHub:    cancelHub,
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
// expiry. Takes testing.TB so the confirmation-latency benchmark reuses the
// same bounded poll.
func waitFor(tb testing.TB, timeout time.Duration, what string, cond func() bool) {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(gatePoll)
	}
	tb.Fatalf("timed out after %v waiting for %s", timeout, what)
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
//
// Decision 14.7 (ADR 0007): the public proto package is the consumer zone's
// boundary — this Set/Get surface is what a consumer integrates through, and
// the client-side last-read-value file stays in the consumer's own client
// library; sup-kv ships no such file (the boundary the round trip pins).
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
	// Delete-then-SetIfAbsent cannot resurrect the key. Decision 14.2
	// (ADR 0002): a deleted key stays blocked by its tombstone, so the
	// tombstone is a state, never a hole a create-if-absent may fill.
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

// TestServer_keyLimit pins XPORT-03's API half: one key constant,
// storage.MaxKeyBytes, inclusive at 1 024 bytes and exclusive at 1 025, on
// Set (create_if_absent rides the same path) and Delete, before any engine
// call. The empty-key message stays its own error — the length check never
// shadows it — and Get stays permissive so pre-upgrade long-key records
// remain readable. The multi-byte half proves the limit counts bytes, not
// runes: 512 two-byte runes are at the limit, one extra ASCII byte is over.
func TestServer_keyLimit(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	ctx := t.Context()

	maxKey := strings.Repeat("k", storage.MaxKeyBytes)
	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: maxKey, Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set at the %d-byte key limit: want OK, got error: %v", storage.MaxKeyBytes, err)
	}
	resp, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: maxKey})
	if err != nil {
		t.Fatalf("Get the %d-byte key: want OK, got error: %v", storage.MaxKeyBytes, err)
	}
	if got := string(resp.GetRecord().GetValue()); got != "v" {
		t.Fatalf("Get the %d-byte key: want value %q, got %q", storage.MaxKeyBytes, "v", got)
	}

	overKey := strings.Repeat("k", storage.MaxKeyBytes+1)
	if len(overKey) != storage.MaxKeyBytes+1 {
		t.Fatalf("over-limit key is %d bytes, want %d", len(overKey), storage.MaxKeyBytes+1)
	}
	overCalls := []struct {
		name string
		call func() error
	}{
		{"Set", func() error {
			_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: overKey, Value: []byte("v")})
			return err
		}},
		{"SetIfAbsent", func() error {
			_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: overKey, Value: []byte("v"), CreateIfAbsent: true})
			return err
		}},
		{"Delete", func() error {
			_, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: overKey})
			return err
		}},
	}
	for _, tc := range overCalls {
		err := tc.call()
		assertCode(t, tc.name+" one byte over the key limit", err, codes.InvalidArgument)
		msg := status.Convert(err).Message()
		for _, want := range []string{"1025", "1024", "bytes"} {
			if !strings.Contains(msg, want) {
				t.Fatalf("%s over the key limit: message %q does not name %q", tc.name, msg, want)
			}
		}
	}

	// The empty key keeps its own message: the length check runs after the
	// empty-key check, so an empty key is never reported as a length fault.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "", Value: []byte("v")})
	assertCode(t, "Set empty key", err, codes.InvalidArgument)
	emptyMsg := status.Convert(err).Message()
	if !strings.Contains(emptyMsg, "must not be empty") {
		t.Fatalf("Set empty key: message %q is not the empty-key message", emptyMsg)
	}
	if strings.Contains(emptyMsg, "exceeds the maximum") {
		t.Fatalf("Set empty key: message %q is the length message, want the empty-key message", emptyMsg)
	}

	// Bytes, not runes: 512 two-byte runes are exactly at the limit.
	multiByteMax := strings.Repeat("é", storage.MaxKeyBytes/2)
	if len(multiByteMax) != storage.MaxKeyBytes {
		t.Fatalf("multi-byte key is %d bytes, want %d", len(multiByteMax), storage.MaxKeyBytes)
	}
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: multiByteMax, Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set the multi-byte key of exactly %d bytes: want OK, got error: %v", storage.MaxKeyBytes, err)
	}
	multiByteOver := multiByteMax + "x"
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: multiByteOver, Value: []byte("v")})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Set the multi-byte key of %d bytes: want InvalidArgument, got %v", len(multiByteOver), err)
	}

	// Reads stay permissive: a long key an older version could have stored
	// is a NotFound, never InvalidArgument (RESEARCH OQ2).
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: overKey})
	assertCode(t, "Get one byte over the key limit", err, codes.NotFound)
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: strings.Repeat("g", 70000)})
	assertCode(t, "Get a 70 000-byte key", err, codes.NotFound)

	// Delete accepts the boundary key; the tombstone is writable at 1 024.
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: maxKey})
	if err != nil {
		t.Fatalf("Delete at the %d-byte key limit: want OK, got error: %v", storage.MaxKeyBytes, err)
	}
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: maxKey})
	assertCode(t, "Get the deleted boundary key", err, codes.NotFound)
}

// TestServer_list is the table-driven List contract: the legacy
// page_size-unset listing (D-12..D-15) covers every live key under a
// prefix, tombstoned keys never appear, an unmatched prefix lists nothing,
// and the paged mode returns the same head of the listing as aligned
// records. slices.Equal is order-sensitive, so the want-list comparison
// pins the strict ascending order.
func TestServer_list(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	ctx := t.Context()

	// Seed keys across two prefixes plus one key that gets tombstoned.
	for _, key := range []string{"cfg/a", "cfg/b", "svc/a", "svc/b", "tmp/x"} {
		_, _, seedErr := ts.engine.Set(key, []byte("v"))
		if seedErr != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, seedErr)
		}
	}
	_, _, seedErr := ts.engine.Delete("tmp/x")
	if seedErr != nil {
		t.Fatalf("seed Delete(tmp/x) error = %v, want nil", seedErr)
	}

	cases := []struct {
		name string
		req  *apiproto.ListRequest
		want []string
	}{
		{
			name: "empty prefix lists every live key",
			req:  &apiproto.ListRequest{},
			want: []string{"cfg/a", "cfg/b", "svc/a", "svc/b"},
		},
		{
			name: "a prefix lists its live subset",
			req:  &apiproto.ListRequest{Prefix: "cfg/"},
			want: []string{"cfg/a", "cfg/b"},
		},
		{
			name: "an unmatched prefix lists nothing",
			req:  &apiproto.ListRequest{Prefix: "zzz/"},
			want: nil,
		},
		{
			name: "a page carries the head as records",
			req:  &apiproto.ListRequest{PageSize: 2},
			want: []string{"cfg/a", "cfg/b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp, err := ts.client.List(ctx, tc.req)
			if err != nil {
				t.Fatalf("List %+v: want OK, got error: %v", tc.req, err)
			}
			if got := resp.GetKeys(); !slices.Equal(got, tc.want) {
				t.Fatalf("List %+v: want keys %v, got %v", tc.req, tc.want, got)
			}
			if tc.req.GetPageSize() > 0 {
				if got := len(resp.GetRecords()); got != len(tc.want) {
					t.Fatalf("List %+v: want %d aligned records, got %d", tc.req, len(tc.want), got)
				}
				return
			}
			if got := len(resp.GetRecords()); got != 0 {
				t.Fatalf("List %+v: want 0 records in the legacy mode, got %d", tc.req, got)
			}
			if got := resp.GetNextPageToken(); got != "" {
				t.Fatalf("List %+v: want an empty next_page_token in the legacy mode, got %q", tc.req, got)
			}
		})
	}
}

// seedPagesFixture seeds the pagination fixture directly through the
// engine: live keys a, b, d and e, with b holding an empty value — a live
// record, never a tombstone — and c holding a tombstone. The read path is
// what these tests exercise; hundreds of confirmed RPC writes would only
// slow the fixture down.
func seedPagesFixture(t *testing.T, ts *testHarness) {
	t.Helper()
	for _, key := range []string{"a", "b", "c", "d", "e"} {
		value := []byte("value-" + key)
		if key == "b" {
			value = []byte{}
		}
		_, _, seedErr := ts.engine.Set(key, value)
		if seedErr != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, seedErr)
		}
	}
	_, _, seedErr := ts.engine.Delete("c")
	if seedErr != nil {
		t.Fatalf("seed Delete(c) error = %v, want nil", seedErr)
	}
}

// TestServer_listLegacyAll pins the page_size-unset contract (the frozen
// R-01 call shape): one List call with no page size returns every matching
// live key, byte-wise ascending, with zero records and an empty
// next_page_token — tombstones excluded, an empty live value included. The
// 512-key half proves the mode is not bounded by a page size.
func TestServer_listLegacyAll(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	seedPagesFixture(t, ts)
	ctx := t.Context()

	resp, err := ts.client.List(ctx, &apiproto.ListRequest{})
	if err != nil {
		t.Fatalf("List with no page size: want OK, got error: %v", err)
	}
	if got, want := resp.GetKeys(), []string{"a", "b", "d", "e"}; !slices.Equal(got, want) {
		t.Fatalf("List with no page size: want keys %v sorted and tombstone-free, got %v", want, got)
	}
	if got := len(resp.GetRecords()); got != 0 {
		t.Fatalf("List with no page size: want 0 records, got %d", got)
	}
	if got := resp.GetNextPageToken(); got != "" {
		t.Fatalf("List with no page size: want an empty next_page_token, got %q", got)
	}

	// The legacy mode covers every matching key in one response, however
	// many that is: 512 live keys come back from one page_size-0 call.
	for i := range 512 {
		key := fmt.Sprintf("bulk/%03d", i)
		_, _, seedErr := ts.engine.Set(key, []byte("v"))
		if seedErr != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, seedErr)
		}
	}
	big, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "bulk/"})
	if err != nil {
		t.Fatalf("List 512 keys with no page size: want OK, got error: %v", err)
	}
	if got := len(big.GetKeys()); got != 512 {
		t.Fatalf("List 512 keys with no page size: want 512 keys, got %d", got)
	}
	if got := len(big.GetRecords()); got != 0 {
		t.Fatalf("List 512 keys with no page size: want 0 records, got %d", got)
	}
	if got := big.GetNextPageToken(); got != "" {
		t.Fatalf("List 512 keys with no page size: want an empty next_page_token, got %q", got)
	}
	if !slices.IsSorted(big.GetKeys()) {
		t.Fatal("List 512 keys with no page size: keys are not byte-wise ascending")
	}
	if got := slices.Compact(slices.Clone(big.GetKeys())); len(got) != 512 {
		t.Fatalf("List 512 keys with no page size: want 512 distinct keys, got %d", len(got))
	}
}

// TestServer_listFirstPage pins the page_size > 0 contract's first page: at
// most page_size records with values, keys aligned to records, and a
// non-empty next_page_token unless the page ends the list.
func TestServer_listFirstPage(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	seedPagesFixture(t, ts)
	ctx := t.Context()

	resp, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("List page_size 2: want OK, got error: %v", err)
	}
	if got, want := resp.GetKeys(), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Fatalf("List page_size 2: want the first page keys %v, got %v", want, got)
	}
	records := resp.GetRecords()
	if len(records) != 2 {
		t.Fatalf("List page_size 2: want 2 records, got %d", len(records))
	}
	if got, want := records[0].GetKey(), "a"; got != want {
		t.Fatalf("first record key: want %q, got %q", want, got)
	}
	if got, want := records[1].GetKey(), "b"; got != want {
		t.Fatalf("second record key: want %q, got %q", want, got)
	}
	if got := records[0].GetValue(); !bytes.Equal(got, []byte("value-a")) {
		t.Fatalf("record a value: want %q, got %q", "value-a", got)
	}
	if got := records[1].GetValue(); len(got) != 0 {
		t.Fatalf("record b value: want the empty live value, got %q", got)
	}
	if got := resp.GetNextPageToken(); got == "" {
		t.Fatal("List page_size 2: want a non-empty next_page_token while more keys remain, got empty")
	}
	for _, rec := range records {
		if key := rec.GetKey(); key == "c" || key == "d" || key == "e" {
			t.Fatalf("List page_size 2: first page contains the off-page key %q", key)
		}
	}
}

// TestServer_listPageContinuation proves the token is a position that walks
// the matching keys exactly once: following next_page_token with the same
// page size yields the remaining keys, then the empty end-of-list token,
// and the union is the live key set, strictly ascending and duplicate-free.
func TestServer_listPageContinuation(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	seedPagesFixture(t, ts)
	ctx := t.Context()

	var walked []string
	token := ""
	for pages := 1; ; pages++ {
		if pages > 8 {
			t.Fatalf("the page walk did not terminate after %d pages", pages-1)
		}
		resp, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 2, PageToken: token})
		if err != nil {
			t.Fatalf("List page %d (token %q): want OK, got error: %v", pages, token, err)
		}
		if len(resp.GetRecords()) != len(resp.GetKeys()) {
			t.Fatalf("page %d: want keys and records aligned, got %d keys and %d records",
				pages, len(resp.GetKeys()), len(resp.GetRecords()))
		}
		for i, key := range resp.GetKeys() {
			if got := resp.GetRecords()[i].GetKey(); got != key {
				t.Fatalf("page %d: records[%d] key = %q, want the aligned key %q", pages, i, got, key)
			}
		}
		walked = append(walked, resp.GetKeys()...)
		token = resp.GetNextPageToken()
		if token == "" {
			if pages != 2 {
				t.Fatalf("the page walk ended after %d pages, want 2 (2 keys per page over 4 live keys)", pages)
			}
			break
		}
	}

	if want := []string{"a", "b", "d", "e"}; !slices.Equal(walked, want) {
		t.Fatalf("the page walk = %v, want the live key set %v", walked, want)
	}
	if !slices.IsSorted(walked) {
		t.Fatal("the page walk is not strictly ascending")
	}
	if got := slices.Compact(slices.Clone(walked)); len(got) != len(walked) {
		t.Fatalf("the page walk repeats a key: %v", walked)
	}
}

// seedListKeys seeds count live keys under prefix directly through the
// engine and returns them in ascending order. The pagination boundary tests
// need hundreds of live keys; the read path is what they exercise, so the
// expensive confirmed-RPC write path is bypassed.
func seedListKeys(t *testing.T, ts *testHarness, prefix string, count int) []string {
	t.Helper()
	keys := make([]string, count)
	for i := range count {
		key := fmt.Sprintf("%s%03d", prefix, i)
		keys[i] = key
		_, _, seedErr := ts.engine.Set(key, []byte("v"))
		if seedErr != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, seedErr)
		}
	}
	return keys
}

// TestServer_listBoundaries pins the page-size contract: a negative page
// size and a page_token without a positive page_size are InvalidArgument
// with the constant message, page_size 0 still returns every matching live
// key with no records, and a page size above the maximum clamps to it.
func TestServer_listBoundaries(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	bulk := seedListKeys(t, ts, "bulk/", 100)
	seedListKeys(t, ts, "more/", 200)
	ctx := t.Context()

	// A page_token with page_size 0 cannot continue the legacy listing:
	// there is no page.
	tokenAtBulk0 := base64.RawURLEncoding.EncodeToString([]byte(bulk[0]))
	invalid := []struct {
		name string
		req  *apiproto.ListRequest
	}{
		{"negative page size", &apiproto.ListRequest{PageSize: -1}},
		{"page token without a page size", &apiproto.ListRequest{PageToken: tokenAtBulk0}},
	}
	for _, tc := range invalid {
		_, err := ts.client.List(ctx, tc.req)
		assertCode(t, tc.name, err, codes.InvalidArgument)
		if got := status.Convert(err).Message(); got != "invalid page size" {
			t.Fatalf("%s: want the constant %q, got %q", tc.name, "invalid page size", got)
		}
	}

	// page_size 0 with no token keeps the legacy shape: every matching key
	// in one response, no records, no token.
	all, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "bulk/", PageSize: 0})
	if err != nil {
		t.Fatalf("List bulk/ with page_size 0: want OK, got error: %v", err)
	}
	if got := all.GetKeys(); !slices.Equal(got, bulk) {
		t.Fatalf("List bulk/ with page_size 0: want the %d seeded keys, got %d keys", len(bulk), len(got))
	}
	if got := len(all.GetRecords()); got != 0 {
		t.Fatalf("List bulk/ with page_size 0: want 0 records, got %d", got)
	}
	if got := all.GetNextPageToken(); got != "" {
		t.Fatalf("List bulk/ with page_size 0: want an empty next_page_token, got %q", got)
	}

	// A page size above the maximum clamps to 256: 300 live keys yield 256
	// keys and records, and a token for the rest.
	clamped, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 257})
	if err != nil {
		t.Fatalf("List page_size 257: want OK, got error: %v", err)
	}
	if got := len(clamped.GetKeys()); got != 256 {
		t.Fatalf("List page_size 257: want the clamped 256 keys, got %d", got)
	}
	if got := len(clamped.GetRecords()); got != 256 {
		t.Fatalf("List page_size 257: want 256 aligned records, got %d", got)
	}
	if !slices.IsSorted(clamped.GetKeys()) {
		t.Fatal("List page_size 257: keys are not byte-wise ascending")
	}
	if got := clamped.GetNextPageToken(); got == "" {
		t.Fatal("List page_size 257: want a non-empty token for the remaining keys, got empty")
	}
}

// TestServer_listEmpty pins the empty and single-key corners in both modes:
// an empty store lists nothing and ends immediately, and one live key comes
// back as one key (legacy) or one keyed record (paged) with an empty token.
func TestServer_listEmpty(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	ctx := t.Context()

	legacy, err := ts.client.List(ctx, &apiproto.ListRequest{})
	if err != nil {
		t.Fatalf("List an empty store with no page size: want OK, got error: %v", err)
	}
	if len(legacy.GetKeys()) != 0 || len(legacy.GetRecords()) != 0 || legacy.GetNextPageToken() != "" {
		t.Fatalf("List an empty store with no page size: want an empty response, got keys=%v records=%d token=%q",
			legacy.GetKeys(), len(legacy.GetRecords()), legacy.GetNextPageToken())
	}
	paged, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("List an empty store with page_size 2: want OK, got error: %v", err)
	}
	if len(paged.GetKeys()) != 0 || len(paged.GetRecords()) != 0 || paged.GetNextPageToken() != "" {
		t.Fatalf("List an empty store with page_size 2: want an empty terminal page, got keys=%v records=%d token=%q",
			paged.GetKeys(), len(paged.GetRecords()), paged.GetNextPageToken())
	}

	_, _, seedErr := ts.engine.Set("only", []byte("v"))
	if seedErr != nil {
		t.Fatalf("seed Set(only) error = %v, want nil", seedErr)
	}
	one, err := ts.client.List(ctx, &apiproto.ListRequest{})
	if err != nil {
		t.Fatalf("List one key with no page size: want OK, got error: %v", err)
	}
	if got := one.GetKeys(); !slices.Equal(got, []string{"only"}) {
		t.Fatalf("List one key with no page size: want [only], got %v", got)
	}
	if len(one.GetRecords()) != 0 || one.GetNextPageToken() != "" {
		t.Fatalf("List one key with no page size: want no records and an empty token, got %d records, token %q",
			len(one.GetRecords()), one.GetNextPageToken())
	}
	onePaged, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("List one key with page_size 2: want OK, got error: %v", err)
	}
	if got := onePaged.GetKeys(); !slices.Equal(got, []string{"only"}) {
		t.Fatalf("List one key with page_size 2: want [only], got %v", got)
	}
	if got := len(onePaged.GetRecords()); got != 1 {
		t.Fatalf("List one key with page_size 2: want 1 record, got %d", got)
	}
	if got := onePaged.GetNextPageToken(); got != "" {
		t.Fatalf("List one key with page_size 2: want the empty end-of-list token, got %q", got)
	}
}

// TestServer_listToken pins the page-token contract: an undecodable token
// and one whose decoded key exceeds storage.MaxKeyBytes are InvalidArgument
// with the constant message, while a well-formed token past the last key is
// a legal empty terminal page.
func TestServer_listToken(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	seedPagesFixture(t, ts)
	ctx := t.Context()

	for _, token := range []string{"!!!!", "a"} {
		_, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 2, PageToken: token})
		assertCode(t, "List malformed token "+token, err, codes.InvalidArgument)
		if got := status.Convert(err).Message(); got != "invalid page token" {
			t.Fatalf("List malformed token %q: want the constant %q, got %q", token, "invalid page token", got)
		}
	}

	oversized := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte("k"), storage.MaxKeyBytes+1))
	_, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 2, PageToken: oversized})
	assertCode(t, "List an oversized token", err, codes.InvalidArgument)
	if got := status.Convert(err).Message(); got != "invalid page token" {
		t.Fatalf("List an oversized token: want the constant %q, got %q", "invalid page token", got)
	}

	pastEnd := base64.RawURLEncoding.EncodeToString([]byte("zzz"))
	resp, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 2, PageToken: pastEnd})
	if err != nil {
		t.Fatalf("List a token past the last key: want OK, got error: %v", err)
	}
	if len(resp.GetKeys()) != 0 || len(resp.GetRecords()) != 0 || resp.GetNextPageToken() != "" {
		t.Fatalf("List a token past the last key: want an empty terminal page, got keys=%v records=%d token=%q",
			resp.GetKeys(), len(resp.GetRecords()), resp.GetNextPageToken())
	}
}

// TestServer_listPrefix pins prefix filtering before pagination in both
// modes: a prefix selects exactly its live subset, and the page size counts
// only matching keys, so no out-of-prefix key ever appears on a page.
func TestServer_listPrefix(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	for _, key := range []string{"cfg/a", "cfg/b", "svc/a", "svc/b", "svc/c"} {
		_, _, seedErr := ts.engine.Set(key, []byte("v"))
		if seedErr != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, seedErr)
		}
	}
	ctx := t.Context()

	legacy, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "svc/"})
	if err != nil {
		t.Fatalf("List svc/ with no page size: want OK, got error: %v", err)
	}
	if got, want := legacy.GetKeys(), []string{"svc/a", "svc/b", "svc/c"}; !slices.Equal(got, want) {
		t.Fatalf("List svc/ with no page size: want %v, got %v", want, got)
	}
	if len(legacy.GetRecords()) != 0 || legacy.GetNextPageToken() != "" {
		t.Fatalf("List svc/ with no page size: want no records and an empty token, got %d records, token %q",
			len(legacy.GetRecords()), legacy.GetNextPageToken())
	}

	first, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "svc/", PageSize: 2})
	if err != nil {
		t.Fatalf("List svc/ page_size 2: want OK, got error: %v", err)
	}
	if got, want := first.GetKeys(), []string{"svc/a", "svc/b"}; !slices.Equal(got, want) {
		t.Fatalf("List svc/ page_size 2: want the prefixed first page %v, got %v", want, got)
	}
	if got := first.GetNextPageToken(); got == "" {
		t.Fatal("List svc/ page_size 2: want a non-empty token for the remaining prefixed key, got empty")
	}
	second, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "svc/", PageSize: 2, PageToken: first.GetNextPageToken()})
	if err != nil {
		t.Fatalf("List svc/ page 2: want OK, got error: %v", err)
	}
	if got, want := second.GetKeys(), []string{"svc/c"}; !slices.Equal(got, want) {
		t.Fatalf("List svc/ page 2: want the remaining prefixed key %v, got %v", want, got)
	}
	if got := second.GetNextPageToken(); got != "" {
		t.Fatalf("List svc/ page 2: want the empty end-of-list token, got %q", got)
	}
}

// listEnvelopeCeiling bounds a ListResponse's serialized bytes beyond its
// entries: the 1 366-character base64 token ceiling, its field framing and
// the gRPC message prefix. It restates the package's
// listEnvelopeOverheadBytes (1 382) with slack because this external test
// cannot see the unexported constant.
const listEnvelopeCeiling = 2048

// newProdLimitServer builds the open-gate harness at the production value
// limit with a discarding dump store: the List cost tests seed and return
// maximum-size values, which the shrunken harness limit rejects and the
// recording fake would retain generation after generation.
func newProdLimitServer(t *testing.T) *testHarness {
	t.Helper()
	h := buildTestServer(t, testServerConfig{
		gate:          openGateOptions(),
		valueMaxBytes: storage.ValueMaxBytes,
		discardDumps:  true,
	})
	awaitGateOpen(t, h)
	return h
}

// seedMaxValueRecords seeds count records holding maximum-size values under
// keys derived from prefix, through ApplyRemote — the bulk merge path — so
// the seeding neither waits for one confirmed dump per record nor grows the
// dump history. The cost tests care only about the store's contents.
func seedMaxValueRecords(t *testing.T, ts *testHarness, prefix string, count int) {
	t.Helper()
	value := bytes.Repeat([]byte("v"), storage.ValueMaxBytes)
	for i := range count {
		key := fmt.Sprintf("%s%03d", prefix, i)
		rec := storage.Record{
			Version: version.PackVersion(int64(i)+1, 0, 7),
			Data:    value,
		}
		_, _, seedErr := ts.engine.ApplyRemote(key, rec)
		if seedErr != nil {
			t.Fatalf("seed ApplyRemote(%q) error = %v, want nil", key, seedErr)
		}
	}
}

// TestServer_listLegacyCostIndependent proves the legacy path copies no
// value bytes: with a hundred maximum-size values in the store, a
// page_size-unset List returns the single small key, no records and no
// token, and the marshaled response is bounded by that key plus the
// envelope ceiling — the off-page values cross nothing.
func TestServer_listLegacyCostIndependent(t *testing.T) {
	t.Parallel()

	ts := newProdLimitServer(t)
	_, _, seedErr := ts.engine.Set("a", []byte("small"))
	if seedErr != nil {
		t.Fatalf("seed Set(a) error = %v, want nil", seedErr)
	}
	seedMaxValueRecords(t, ts, "z/bulk-", 100)
	ctx := t.Context()

	resp, err := ts.client.List(ctx, &apiproto.ListRequest{Prefix: "a"})
	if err != nil {
		t.Fatalf("List a with no page size: want OK, got error: %v", err)
	}
	if got := resp.GetKeys(); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("List a with no page size: want the one small key, got %v", got)
	}
	if got := len(resp.GetRecords()); got != 0 {
		t.Fatalf("List a with no page size: want 0 records, got %d", got)
	}
	if got := resp.GetNextPageToken(); got != "" {
		t.Fatalf("List a with no page size: want an empty token, got %q", got)
	}
	raw, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("proto.Marshal(ListResponse): want nil error, got %v", err)
	}
	if got, ceiling := len(raw), len("a")+listEnvelopeCeiling; got > ceiling {
		t.Fatalf("legacy response is %d bytes, want at most %d (the key plus envelope) despite %d off-page maximum-size values",
			got, ceiling, 100)
	}
}

// TestServer_listCostIndependent proves a page pays only for its own
// entries: on the same maximum-value store, a page_size-1 request returns
// one record and marshals below one value limit plus the envelope ceiling,
// independent of the hundred off-page values.
func TestServer_listCostIndependent(t *testing.T) {
	t.Parallel()

	ts := newProdLimitServer(t)
	_, _, seedErr := ts.engine.Set("a", []byte("small"))
	if seedErr != nil {
		t.Fatalf("seed Set(a) error = %v, want nil", seedErr)
	}
	seedMaxValueRecords(t, ts, "z/bulk-", 100)
	ctx := t.Context()

	resp, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 1})
	if err != nil {
		t.Fatalf("List page_size 1: want OK, got error: %v", err)
	}
	if got := len(resp.GetRecords()); got != 1 {
		t.Fatalf("List page_size 1: want 1 record, got %d", got)
	}
	raw, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("proto.Marshal(ListResponse): want nil error, got %v", err)
	}
	if got, ceiling := len(raw), storage.ValueMaxBytes+listEnvelopeCeiling; got > ceiling {
		t.Fatalf("page_size-1 response is %d bytes, want at most %d (one value plus envelope) despite %d off-page values",
			got, ceiling, 100)
	}
}

// TestServer_listByteBudget proves the serialized-page budget stops a page
// below the page size while guaranteeing progress: 120 maximum-size values
// exceed the 6 MiB budget, the first page is smaller than 256 entries and
// carries a token, and walking the token reaches every record exactly once.
func TestServer_listByteBudget(t *testing.T) {
	t.Parallel()

	ts := newProdLimitServer(t)
	seedMaxValueRecords(t, ts, "v/", 120)
	ctx := t.Context()

	first, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 256})
	if err != nil {
		t.Fatalf("List page_size 256: want OK, got error: %v", err)
	}
	if got := len(first.GetRecords()); got == 0 {
		t.Fatal("List page_size 256: want a non-empty first page, got 0 records")
	}
	if got := len(first.GetRecords()); got >= 256 {
		t.Fatalf("List page_size 256: want the byte budget to stop the page below 256, got %d records", got)
	}
	if got := first.GetNextPageToken(); got == "" {
		t.Fatal("List page_size 256: want a non-empty token for the remaining records, got empty")
	}

	seen := make(map[string]struct{}, 120)
	page := 0
	token := ""
	for {
		page++
		if page > 4 {
			t.Fatalf("the byte-budget walk did not terminate after %d pages", page-1)
		}
		resp, err := ts.client.List(ctx, &apiproto.ListRequest{PageSize: 256, PageToken: token})
		if err != nil {
			t.Fatalf("List page %d (token %q): want OK, got error: %v", page, token, err)
		}
		if len(resp.GetKeys()) != len(resp.GetRecords()) {
			t.Fatalf("page %d: want keys and records aligned, got %d keys and %d records",
				page, len(resp.GetKeys()), len(resp.GetRecords()))
		}
		for _, key := range resp.GetKeys() {
			if _, dup := seen[key]; dup {
				t.Fatalf("page %d repeats key %q: the byte-budget walk must progress without duplicates", page, key)
			}
			seen[key] = struct{}{}
		}
		if token = resp.GetNextPageToken(); token == "" {
			break
		}
	}
	if got := len(seen); got != 120 {
		t.Fatalf("the byte-budget walk saw %d distinct keys, want all 120", got)
	}
}

// newListedServer builds the open-gate harness with the listing-mirror
// membership seam wired: listed reports whether a peer's complete listing
// named a key (rule 14.5(6)).
func newListedServer(t *testing.T, listed func(key string) bool) *testHarness {
	t.Helper()
	h := buildTestServer(t, testServerConfig{gate: openGateOptions(), listedKey: listed})
	awaitGateOpen(t, h)
	return h
}

// TestServer_getListedNotArrived pins rule 14.5(6)'s read answer: a key a
// peer's complete listing named, but whose value has not arrived locally,
// is Unavailable with the listed-key message — never NotFound, which would
// claim the peer's key does not exist.
func TestServer_getListedNotArrived(t *testing.T) {
	t.Parallel()

	ts := newListedServer(t, func(key string) bool { return key == "listed/pending" })
	ctx := t.Context()

	_, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: "listed/pending"})
	assertCode(t, "Get listed-but-unarrived key", err, codes.Unavailable)
	if got := status.Convert(err).Message(); got != listedNotArrivedMessage {
		t.Fatalf("Get listed-but-unarrived key message: want %q, got %q", listedNotArrivedMessage, got)
	}
}

// TestServer_getListedTombstoneAndAbsent pins the tombstone-visible
// discriminator: a listed key that is tombstoned locally is NotFound (the
// local tombstone wins over "not arrived"), and a key no listing names that
// is absent locally is NotFound too. Engine.Get alone would collapse both
// into the listed-but-unarrived answer, which is why the handler probes
// Engine.Version (RESEARCH Pitfall 10).
func TestServer_getListedTombstoneAndAbsent(t *testing.T) {
	t.Parallel()

	ts := newListedServer(t, func(key string) bool { return key == "listed/dead" })
	ctx := t.Context()

	// Local tombstone: the listed key was deleted here.
	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "listed/dead", Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set listed/dead: want OK, got error: %v", err)
	}
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "listed/dead"})
	if err != nil {
		t.Fatalf("Delete listed/dead: want OK, got error: %v", err)
	}

	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "listed/dead"})
	assertCode(t, "Get listed tombstoned key", err, codes.NotFound)

	// Unlisted and locally absent: no local record and no listing names it.
	_, err = ts.client.Get(ctx, &apiproto.GetRequest{Key: "listed/absent"})
	assertCode(t, "Get unlisted absent key", err, codes.NotFound)
}

// TestServer_getListedLive pins local presence winning: a listed key whose
// value is already here is served normally — the listing seam is consulted
// only when the local read misses.
func TestServer_getListedLive(t *testing.T) {
	t.Parallel()

	ts := newListedServer(t, func(key string) bool { return key == "listed/live" })
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "listed/live", Value: []byte("here")})
	if err != nil {
		t.Fatalf("Set listed/live: want OK, got error: %v", err)
	}
	get, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: "listed/live"})
	if err != nil {
		t.Fatalf("Get listed live key: want OK, got error: %v", err)
	}
	if !bytes.Equal(get.GetRecord().GetValue(), []byte("here")) {
		t.Fatalf("Get listed live key: want %q, got %q", "here", get.GetRecord().GetValue())
	}
}

// TestServer_createIfAbsentListed pins the F14 guard: create_if_absent on
// any listed key — locally absent or tombstoned — is AlreadyExists before
// the engine is called, because a peer already owns that key and a local
// create would overwrite it. An unlisted key still reaches the engine's
// SetIfAbsent, whose own AlreadyExists applies after the first write.
func TestServer_createIfAbsentListed(t *testing.T) {
	t.Parallel()

	ts := newListedServer(t, func(key string) bool {
		return key == "listed/absent" || key == "listed/dead"
	})
	ctx := t.Context()

	// A tombstoned listed key starts as a local record the test deletes.
	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "listed/dead", Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set listed/dead: want OK, got error: %v", err)
	}
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "listed/dead"})
	if err != nil {
		t.Fatalf("Delete listed/dead: want OK, got error: %v", err)
	}

	saves := func() int {
		ts.dumps.mu.Lock()
		defer ts.dumps.mu.Unlock()
		return len(ts.dumps.saved)
	}
	before := saves()

	_, err = ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "listed/absent", Value: []byte("local"), CreateIfAbsent: true,
	})
	assertCode(t, "create_if_absent on a listed absent key", err, codes.AlreadyExists)
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "listed/dead", Value: []byte("local"), CreateIfAbsent: true,
	})
	assertCode(t, "create_if_absent on a listed tombstoned key", err, codes.AlreadyExists)
	if after := saves(); after != before {
		t.Fatalf("create_if_absent on listed keys reached the engine: dump generations %d -> %d", before, after)
	}

	// Unlisted: the engine decides — the first create succeeds and dumps,
	// the second is the engine's own AlreadyExists.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "listed/unlisted", Value: []byte("local"), CreateIfAbsent: true,
	})
	if err != nil {
		t.Fatalf("create_if_absent on an unlisted key: want OK, got error: %v", err)
	}
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "listed/unlisted", Value: []byte("other"), CreateIfAbsent: true,
	})
	assertCode(t, "create_if_absent on an existing unlisted key", err, codes.AlreadyExists)
}

// TestServer_dumpFailedWriteNotifies proves PROTO-08's write half (D-08
// totality, T-4-18): a Set the engine installed in memory but whose
// confirming dump failed still reaches the replication notify seam before
// the RPC answers Unavailable, so the write broadcasts and appears in the
// unacked ledger like any unconfirmed write. track is false even though the
// request asked for a replication report: the RPC errors, so no report
// exists.
func TestServer_dumpFailedWriteNotifies(t *testing.T) {
	t.Parallel()

	ts, capture := newNotifyServer(t)
	ctx := t.Context()
	ts.dumps.mu.Lock()
	ts.dumps.failSave = true
	ts.dumps.mu.Unlock()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "notify/set", Value: []byte("set-value"), WaitForReplication: true,
	})
	assertCode(t, "Set with failing dump", err, codes.Unavailable)

	calls := capture.snapshot()
	if len(calls) != 1 {
		t.Fatalf("notify calls = %d, want exactly 1 for the unconfirmed write: %+v", len(calls), calls)
	}
	call := calls[0]
	if call.key != "notify/set" || !bytes.Equal(call.rec.Data, []byte("set-value")) || call.rec.Deleted {
		t.Errorf("notified record = (key %q, data %q, deleted %t), want the request's live record",
			call.key, call.rec.Data, call.rec.Deleted)
	}
	if call.track {
		t.Error("dump-failed write notified with track=true, want false (the RPC reports an error)")
	}
	ver, present := ts.engine.Version("notify/set")
	if !present || call.rec.Version != ver {
		t.Errorf("notified version %x = engine version %x (present=%t), want the store-stamped version",
			call.rec.Version[:], ver[:], present)
	}
}

// TestServer_dumpFailedDeleteNotifies proves the tombstone half: a Delete
// the engine installed but whose dump failed notifies exactly one tombstone
// with track=false before Unavailable.
func TestServer_dumpFailedDeleteNotifies(t *testing.T) {
	t.Parallel()

	ts, capture := newNotifyServer(t)
	ctx := t.Context()
	_, _, seedErr := ts.engine.Set("notify/delete", []byte("doomed"))
	if seedErr != nil {
		t.Fatalf("seed Set(notify/delete) error = %v, want nil", seedErr)
	}
	ts.dumps.mu.Lock()
	ts.dumps.failSave = true
	ts.dumps.mu.Unlock()

	_, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{
		Key: "notify/delete", WaitForReplication: true,
	})
	assertCode(t, "Delete with failing dump", err, codes.Unavailable)

	calls := capture.snapshot()
	if len(calls) != 1 {
		t.Fatalf("notify calls = %d, want exactly 1 for the unconfirmed tombstone: %+v", len(calls), calls)
	}
	call := calls[0]
	if call.key != "notify/delete" || !call.rec.Deleted || len(call.rec.Data) != 0 {
		t.Errorf("notified record = (key %q, deleted %t, data %q), want the request's tombstone",
			call.key, call.rec.Deleted, call.rec.Data)
	}
	if call.track {
		t.Error("dump-failed delete notified with track=true, want false")
	}
}

// TestServer_dumpFailedCreateIfAbsentNotifies proves SetIfAbsent rides the
// same unconfirmed-write path: one notify with the request's live record and
// track=false, then Unavailable.
func TestServer_dumpFailedCreateIfAbsentNotifies(t *testing.T) {
	t.Parallel()

	ts, capture := newNotifyServer(t)
	ctx := t.Context()
	ts.dumps.mu.Lock()
	ts.dumps.failSave = true
	ts.dumps.mu.Unlock()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{
		Key: "notify/create", Value: []byte("create-value"), CreateIfAbsent: true,
	})
	assertCode(t, "SetIfAbsent with failing dump", err, codes.Unavailable)

	calls := capture.snapshot()
	if len(calls) != 1 {
		t.Fatalf("notify calls = %d, want exactly 1 for the unconfirmed create: %+v", len(calls), calls)
	}
	call := calls[0]
	if call.key != "notify/create" || !bytes.Equal(call.rec.Data, []byte("create-value")) ||
		call.rec.Deleted || call.track {
		t.Errorf("notified record = (key %q, data %q, deleted %t, track %t), want the unconfirmed live record with track=false",
			call.key, call.rec.Data, call.rec.Deleted, call.track)
	}
}

// TestServer_closedWriteDoesNotNotify proves the shutdown half: a write the
// closing engine rejects with ErrClosed is work the process is abandoning, so
// replication is never told about it. The empty capture is asserted, not
// merely an absent branch.
func TestServer_closedWriteDoesNotNotify(t *testing.T) {
	t.Parallel()

	ts, capture := newNotifyServer(t)
	ctx := t.Context()
	_, _, seedErr := ts.engine.Set("notify/live", []byte("live"))
	if seedErr != nil {
		t.Fatalf("seed Set(notify/live) error = %v, want nil", seedErr)
	}
	ts.cancelEngine()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "notify/set", Value: []byte("v")})
	assertCode(t, "Set after engine close", err, codes.Unavailable)
	_, err = ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "notify/live"})
	assertCode(t, "Delete after engine close", err, codes.Unavailable)

	if calls := capture.snapshot(); len(calls) != 0 {
		t.Fatalf("notify calls after closed writes = %d, want 0: %+v", len(calls), calls)
	}
}

// unconfirmedHarness builds an open-gate notify harness whose engine has a
// short confirmation timeout and whose dumper is parked — the PERSIST-08
// timed-out-write setup (a hung fsync, stood in by the fake's hold).
func unconfirmedHarness(t *testing.T) (*testHarness, *notifyCapture) {
	t.Helper()
	capture := &notifyCapture{}
	h := buildTestServer(t, testServerConfig{
		gate:           openGateOptions(),
		notify:         capture.notify,
		confirmTimeout: 50 * time.Millisecond,
	})
	awaitGateOpen(t, h)
	return h, capture
}

// TestServer_unconfirmedSetMapsAndNotifies proves PROTO-08's totality for
// the PERSIST-08 class: a Set whose covering dump outlives the confirmation
// timeout is installed in memory, so it is notified to replication before
// the RPC answers Unavailable. The status message is the sentinel's own, so
// the unconfirmed class is distinguishable from a dump failure, and the
// engine counts the outcome.
func TestServer_unconfirmedSetMapsAndNotifies(t *testing.T) {
	t.Parallel()

	ts, capture := unconfirmedHarness(t)
	ctx := t.Context()
	ts.dumps.holdDumps()

	const key = "unconfirmed/set"
	_, err := ts.client.Set(ctx, &apiproto.SetRequest{
		Key: key, Value: []byte("set-value"), WaitForReplication: true,
	})
	assertCode(t, "Set with a hung dump", err, codes.Unavailable)
	if got := status.Convert(err).Message(); got != persist.ErrUnconfirmed.Error() {
		t.Errorf("Set with a hung dump message = %q, want %q", got, persist.ErrUnconfirmed.Error())
	}

	calls := capture.snapshot()
	if len(calls) != 1 {
		t.Fatalf("notify calls = %d, want exactly 1 for the unconfirmed write: %+v", len(calls), calls)
	}
	call := calls[0]
	if call.key != key || !bytes.Equal(call.rec.Data, []byte("set-value")) || call.rec.Deleted {
		t.Errorf("notified record = (key %q, data %q, deleted %t), want the request's live record",
			call.key, call.rec.Data, call.rec.Deleted)
	}
	if call.track {
		t.Error("unconfirmed write notified with track=true, want false (the RPC reports an error)")
	}
	ver, present := ts.engine.Version(key)
	if !present || call.rec.Version != ver {
		t.Errorf("notified version %x = engine version %x (present=%t), want the store-stamped version",
			call.rec.Version[:], ver[:], present)
	}
	if got := ts.engine.DumpStats().Unconfirmed; got != 1 {
		t.Errorf("DumpStats().Unconfirmed = %d, want 1", got)
	}
}

// TestServer_unconfirmedDeleteMapsAndNotifies proves the tombstone half: a
// Delete whose dump outlives the confirmation timeout notifies exactly one
// tombstone with track=false before Unavailable.
func TestServer_unconfirmedDeleteMapsAndNotifies(t *testing.T) {
	t.Parallel()

	ts, capture := unconfirmedHarness(t)
	ctx := t.Context()
	_, _, seedErr := ts.engine.Set("unconfirmed/delete", []byte("doomed"))
	if seedErr != nil {
		t.Fatalf("seed Set(unconfirmed/delete) error = %v, want nil", seedErr)
	}
	ts.dumps.holdDumps()

	_, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{
		Key: "unconfirmed/delete", WaitForReplication: true,
	})
	assertCode(t, "Delete with a hung dump", err, codes.Unavailable)
	if got := status.Convert(err).Message(); got != persist.ErrUnconfirmed.Error() {
		t.Errorf("Delete with a hung dump message = %q, want %q", got, persist.ErrUnconfirmed.Error())
	}

	calls := capture.snapshot()
	if len(calls) != 1 {
		t.Fatalf("notify calls = %d, want exactly 1 for the unconfirmed tombstone: %+v", len(calls), calls)
	}
	call := calls[0]
	if call.key != "unconfirmed/delete" || !call.rec.Deleted || len(call.rec.Data) != 0 {
		t.Errorf("notified record = (key %q, deleted %t, data %q), want the request's tombstone",
			call.key, call.rec.Deleted, call.rec.Data)
	}
	if call.track {
		t.Error("unconfirmed delete notified with track=true, want false")
	}
}

// TestServer_snapshotLoadNotifies proves PROTO-08's snapshot half (T-4-19):
// a load replays the snapshot's records into memory and notifies every
// replayed record — sorted by key, track=false — even when the confirming
// dump failed and the RPC reports Unavailable. Notifying what is already in
// memory is what makes the load replicate; the error still reports the dump
// honestly.
func TestServer_snapshotLoadNotifies(t *testing.T) {
	t.Parallel()

	ts, capture := newNotifyServer(t)
	ctx := t.Context()

	snapVersion := version.PackVersion(1_700_000_000_000, 3, 7)
	fixture := map[string]storage.Record{
		"snap/alpha": {Version: snapVersion, Data: []byte("alpha-value")},
		"snap/beta":  {Version: snapVersion, Data: []byte("beta-value")},
		"snap/gamma": {Version: snapVersion, Data: []byte("gamma-value")},
	}
	for key, rec := range fixture {
		rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
		fixture[key] = rec
	}
	dump, err := persist.Encode(fixture, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("persist.Encode(snapshot fixture) error = %v, want nil", err)
	}
	ts.dumps.mu.Lock()
	ts.dumps.snapshot = dump
	ts.dumps.failSave = true
	ts.dumps.mu.Unlock()

	_, err = ts.client.SnapshotLoad(ctx, &apiproto.SnapshotLoadRequest{Name: "snap"})
	assertCode(t, "SnapshotLoad with failing final dump", err, codes.Unavailable)

	calls := capture.snapshot()
	gotKeys := make([]string, len(calls))
	for i, call := range calls {
		gotKeys[i] = call.key
	}
	wantKeys := []string{"snap/alpha", "snap/beta", "snap/gamma"}
	if !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("notify keys = %v, want %v in sorted replay order", gotKeys, wantKeys)
	}
	for _, call := range calls {
		if call.track {
			t.Errorf("snapshot-load notify for %q has track=true, want false", call.key)
		}
		want := fixture[call.key]
		if !bytes.Equal(call.rec.Data, want.Data) || call.rec.Deleted {
			t.Errorf("notified %q = (data %q, deleted %t), want the snapshot's live record",
				call.key, call.rec.Data, call.rec.Deleted)
		}
		if call.rec.Version == (version.RecordVersion{}) || call.rec.Version.Compare(snapVersion) != 1 {
			t.Errorf("notified %q version %x does not dominate the snapshot's %x",
				call.key, call.rec.Version[:], snapVersion[:])
		}
	}
}

// TestServer_snapshotLoadUsesReplaySeam proves WR-04's wiring half: when the
// paced replay seam is installed, every replayed record reaches it (never the
// live notify seam), so a load drains through the record-send pacer instead of
// emitting one immediate broadcast per key. The live seam is empty because a
// load installs no wait_for_replication waiter.
func TestServer_snapshotLoadUsesReplaySeam(t *testing.T) {
	t.Parallel()

	live := &notifyCapture{}
	replayed := make([]string, 0, 3)
	var replayMu sync.Mutex
	h := buildTestServer(t, testServerConfig{
		gate:   openGateOptions(),
		notify: live.notify,
		replayNotify: func(key string, _ storage.Record) error {
			replayMu.Lock()
			defer replayMu.Unlock()
			replayed = append(replayed, key)
			return nil
		},
	})
	awaitGateOpen(t, h)

	snapVersion := version.PackVersion(1_700_000_000_000, 3, 7)
	fixture := map[string]storage.Record{
		"replay/alpha": {Version: snapVersion, Data: []byte("alpha-value")},
		"replay/beta":  {Version: snapVersion, Data: []byte("beta-value")},
	}
	for key, rec := range fixture {
		rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
		fixture[key] = rec
	}
	dump, err := persist.Encode(fixture, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("persist.Encode(snapshot fixture) error = %v, want nil", err)
	}
	h.dumps.mu.Lock()
	h.dumps.snapshot = dump
	h.dumps.mu.Unlock()

	_, err = h.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "replay"})
	if err != nil {
		t.Fatalf("SnapshotLoad error = %v, want nil", err)
	}

	replayMu.Lock()
	got := slices.Clone(replayed)
	replayMu.Unlock()
	want := []string{"replay/alpha", "replay/beta"}
	if !slices.Equal(got, want) {
		t.Fatalf("paced replay seam received keys %v, want %v in sorted replay order (WR-04)", got, want)
	}
	if calls := live.snapshot(); len(calls) != 0 {
		t.Fatalf("the live notify seam received %d replay notifications, want 0 — replayed records take the paced seam", len(calls))
	}
}

// TestServer_snapshotLoadReplayFallsBackToLive proves the nil-seam contract:
// without a paced replay seam the replayed records still notify through the
// live seam with track=false, so no load is ever silently unreplicated.
func TestServer_snapshotLoadReplayFallsBackToLive(t *testing.T) {
	t.Parallel()

	h, capture := newNotifyServer(t)
	fixture := map[string]storage.Record{
		"fallback/key": {Version: version.PackVersion(1_700_000_000_000, 4, 9), Data: []byte("v")},
	}
	for key, rec := range fixture {
		rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
		fixture[key] = rec
	}
	dump, err := persist.Encode(fixture, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("persist.Encode(snapshot fixture) error = %v, want nil", err)
	}
	h.dumps.mu.Lock()
	h.dumps.snapshot = dump
	h.dumps.mu.Unlock()

	_, loadErr := h.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "fallback"})
	if loadErr != nil {
		t.Fatalf("SnapshotLoad error = %v, want nil", loadErr)
	}
	calls := capture.snapshot()
	if len(calls) != 1 || calls[0].key != "fallback/key" || calls[0].track {
		t.Fatalf("live-seam fallback calls = %+v, want one track=false notify for \"fallback/key\"", calls)
	}
}

// TestServer_snapshotLoadNoOpDoesNotNotify proves the no-op half: a load
// whose snapshot already equals the current state answers OK and notifies
// nothing — an unchanged load has nothing to replicate.
func TestServer_snapshotLoadNoOpDoesNotNotify(t *testing.T) {
	t.Parallel()

	ts, capture := newNotifyServer(t)
	ctx := t.Context()

	_, _, seedErr := ts.engine.Set("same/key", []byte("same-value"))
	if seedErr != nil {
		t.Fatalf("seed Set(same/key) error = %v, want nil", seedErr)
	}
	ts.dumps.mu.Lock()
	if len(ts.dumps.saved) == 0 {
		ts.dumps.mu.Unlock()
		t.Fatal("seed write left no saved generation")
	}
	ts.dumps.snapshot = bytes.Clone(ts.dumps.saved[len(ts.dumps.saved)-1])
	ts.dumps.mu.Unlock()

	_, err := ts.client.SnapshotLoad(ctx, &apiproto.SnapshotLoadRequest{Name: "same"})
	if err != nil {
		t.Fatalf("SnapshotLoad(same) error = %v, want nil", err)
	}
	if calls := capture.snapshot(); len(calls) != 0 {
		t.Fatalf("notify calls after the no-op load = %d, want 0: %+v", len(calls), calls)
	}
}
