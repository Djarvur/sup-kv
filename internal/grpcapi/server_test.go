package grpcapi_test

import (
	"bytes"
	"context"
	"errors"
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

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
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
// ReadSnapshot seam the snapshot-load totality tests drive.
type fakeDumpStore struct {
	mu       sync.Mutex
	failSave bool
	saved    [][]byte
	snapshot []byte
}

// Save is the byte-slice convenience over SaveStream for fixtures that
// already hold a generation.
func (f *fakeDumpStore) Save(dump []byte) error {
	return f.SaveStream(func(w io.Writer) error {
		_, err := w.Write(dump)
		return err
	})
}

// SaveStream runs fill into memory and records the generation, or fails
// while failSave is set — the injection point for persist.ErrDumpFailed.
func (f *fakeDumpStore) SaveStream(fill func(io.Writer) error) error {
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
	f.saved = append(f.saved, bytes.Clone(buf.Bytes()))
	return nil
}

// Load reports the empty start: every test begins as a fresh node — the
// fallback chain (main, .prev) is startup business no API test needs.
func (*fakeDumpStore) Load(_ func([]byte) error) ([]byte, bool) {
	return nil, false
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
// record the handler built (version, data, tombstone flag, checksum) and the
// wait_for_replication flag it carried.
type notifyCall struct {
	key   string
	rec   storage.Record
	track bool
}

// notifyCapture records LocalNotify calls under a mutex: the handlers run on
// gRPC goroutines, so an unguarded slice would race.
type notifyCapture struct {
	mu    sync.Mutex
	calls []notifyCall
}

// notify is the LocalNotify seam the harness installs; it records and
// accepts every call.
func (c *notifyCapture) notify(key string, rec storage.Record, track bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, notifyCall{key: key, rec: rec, track: track})
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

	// listedKey is the listing-mirror membership seam (D-01, rule 14.5(6)):
	// it reports whether a peer's complete listing named a key. Nil means no
	// listing knowledge, the single-node configuration.
	listedKey func(key string) bool

	// notify is the LocalNotify seam the server is constructed with; nil is
	// the single-node configuration (D-11). The totality tests install a
	// capturing seam to observe which writes and replays reach replication.
	notify func(key string, rec storage.Record, track bool) error

	// replayNotify is the paced replay seam (WR-04); nil leaves replay
	// notifications on the live notify seam.
	replayNotify func(key string, rec storage.Record) error

	// waitTimeout, knownPeers and waitReplicated are the
	// wait_for_replication seams (API-05, plan 05-05), merged into the same
	// ServerOptions the gate wiring rides on. waitTimeout zero means
	// NewServer's 1s default (D-09, A2); nil knownPeers/waitReplicated mean
	// the single-node configuration reporting acked immediately (D-11).
	waitTimeout    time.Duration
	knownPeers     func() uint64
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
// storage.Config (OnChange: hub.Ingest, no persist change) — then the
// engine, then go hub.Run under its own context, cancelled via t.Cleanup:
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
	if cfg.listedKey != nil {
		opts.ListedKey = cfg.listedKey
	}
	opts.WaitTimeout = cfg.waitTimeout
	opts.KnownPeers = cfg.knownPeers
	opts.WaitReplicated = cfg.waitReplicated
	opts.ReplayNotify = cfg.replayNotify

	api := grpcapi.NewServer(eng, cfg.notify, hub, opts)
	gateCtx, cancelGate := context.WithCancel(t.Context())
	t.Cleanup(cancelGate)
	go api.RunGate(gateCtx)

	srv := grpc.NewServer()
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
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: want nil, got error: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &testHarness{
		client:       apiproto.NewSupKVServiceClient(conn),
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
