package grpcapi_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	"github.com/Djarvur/sup-kv/internal/grpcapi/apiproto"
)

// testValueLimit is the shrunken value limit the harness runs the store
// under: the design default (61 440) is far above anything the tests write,
// so a small limit makes the value-too-large path reachable with one
// oversized write instead of a 60 KiB payload.
const testValueLimit = 1024

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

// testHarness bundles one running engine and its bufconn-dialed client.
type testHarness struct {
	client apiproto.SupKVClient
	dumps  *fakeDumpStore
}

// newTestServer builds the reusable bufconn harness: a hand-written
// in-memory fake DumpStore, a real persist engine (test node id, wall-clock
// now, shrunken value limit, dump timings tuned down to 1ms/50ms of real
// time — bufconn and synctest bubbles do not mix), the grpcapi Server with
// a nil notify (the legal single-node configuration, D-11), and a gRPC
// client dialed over bufconn with insecure credentials — mandatory on
// modern gRPC clients even for an in-memory transport. Everything stops via
// t.Cleanup, so every test using the harness can run t.Parallel().
func newTestServer(t *testing.T) *testHarness {
	t.Helper()

	dumps := &fakeDumpStore{}
	eng, _ := persist.Open(
		42,
		time.Now,
		time.Minute,
		dumps,
		storage.Config{ValueMaxBytes: testValueLimit},
		persist.Config{Debounce: time.Millisecond, MaxWait: 50 * time.Millisecond},
	)
	engCtx, cancelEng := context.WithCancel(t.Context())
	go func() { _ = eng.Run(engCtx) }()
	t.Cleanup(cancelEng)

	srv := grpc.NewServer()
	apiproto.RegisterSupKVServer(srv, grpcapi.NewServer(eng, nil))

	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: want nil, got error: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &testHarness{client: apiproto.NewSupKVClient(conn), dumps: dumps}
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
