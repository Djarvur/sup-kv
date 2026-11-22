package grpcapi_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// The wait_for_replication integration harness (05-REVIEW CR-01). Unlike
// wait_test.go's scripted tracker fake — whose fixtures can publish states
// the real repl core never produces — this file drives the wait through a
// REAL udprepl.Adapter pair exchanging real datagrams over real UDP
// sockets: the production result seam, where the core deletes a pending
// entry inside the very handleAck call that completes Ack coverage and the
// adapter resolves the waiter tracker from the resulting Outbox event.
//
// The two nodes bind DIFFERENT loopback ports and target each other's port
// as their broadcast address: broadcast Hello discovery crosses over, while
// every unicast — the peer's Ack above all — is delivered to a port exactly
// one socket holds. The adapter's same-host caveat (unicast to a port
// shared by SO_REUSEADDR sockets is the kernel's choice) therefore never
// applies, and the Ack path this test asserts is deterministic.
const (
	// replNodeA and replNodeB are the two adapters' wire node ids; the
	// engine's node id is version-stamping only and carries no wire role.
	replNodeA = uint16(7)
	replNodeB = uint16(9)

	// integrationProbeWait bounds the broadcast capability probe, and
	// integrationDeadline bounds every state the real datagram paths owe
	// the test (discovery, gate open, replication visibility) — generous
	// real-time waits, since the assertions under test carry their own
	// tight bound (waitAssertBound).
	integrationProbeWait = 300 * time.Millisecond
	integrationDeadline  = 5 * time.Second
)

// integrationUDPPort picks one free UDP port: bind a throwaway wildcard
// socket to :0, read its port, close it (the udprepl smoke-test recipe —
// the adapter keeps no local-address accessor, so the test must know the
// ports before constructing the nodes). Takes testing.TB so the
// confirmation-latency benchmark reuses the same picker.
func integrationUDPPort(tb testing.TB) int {
	tb.Helper()
	var lc net.ListenConfig
	// tb.Context() does not exist (testing.TB carries no Context method,
	// Go 1.24 added it to T and B only), so usetesting's suggested fix does
	// not compile — the documented-nolint case.
	conn, err := lc.ListenPacket(context.Background(), "udp4", "0.0.0.0:0") //nolint:usetesting // suggestion does not compile on TB
	if err != nil {
		tb.Fatalf("bind throwaway port-picking socket: %v", err)
	}
	defer func() { _ = conn.Close() }()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

// integrationSocket binds a wildcard udp4 socket with SO_REUSEADDR and
// SO_BROADCAST set before bind — the adapter's exact socket recipe
// (research Pattern 3), replicated here because the adapter keeps it
// unexported and this test's capability probe needs the same topology.
func integrationSocket(bindAddr string) (*net.UDPConn, error) {
	var opErr error
	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if opErr != nil {
					return
				}
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			})
			if err != nil {
				return fmt.Errorf("socket control: %w", err)
			}
			if opErr != nil {
				return fmt.Errorf("socket options: %w", opErr)
			}
			return nil
		},
	}
	packetConn, err := lc.ListenPacket(context.Background(), "udp4", bindAddr)
	if err != nil {
		return nil, fmt.Errorf("bind udp4 %s: %w", bindAddr, err)
	}
	return packetConn.(*net.UDPConn), nil
}

// probeLoopbackBroadcast pre-flights the one platform capability this test
// needs — a datagram sent to the loopback broadcast address reaches the
// socket bound to the destination port (loopback broadcast fan-out is a
// Linux-only kernel behavior; macOS lo0 carries no BROADCAST flag — the
// udprepl smoke test's D-08 probe matrix) — and skips loudly when it is
// missing, naming the probed address. Protocol correctness under absence of
// the capability stays proven by wait_test.go's mirror fixtures. Takes
// testing.TB so the confirmation-latency benchmark skips through the SAME
// gate.
func probeLoopbackBroadcast(tb testing.TB, port int) {
	tb.Helper()
	rx, err := integrationSocket(fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		tb.Fatalf("probe rx bind: %v", err)
	}
	defer func() { _ = rx.Close() }()
	tx, err := integrationSocket("0.0.0.0:0")
	if err != nil {
		tb.Fatalf("probe tx bind: %v", err)
	}
	defer func() { _ = tx.Close() }()
	dst := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: port}
	_, sendErr := tx.WriteToUDP([]byte("sup-kv wait integration probe"), dst)
	if sendErr != nil {
		tb.Fatalf("probe send to %s: %v", dst, sendErr)
	}
	deadlineErr := rx.SetReadDeadline(time.Now().Add(integrationProbeWait))
	if deadlineErr != nil {
		tb.Fatalf("probe read deadline: %v", deadlineErr)
	}
	buf := make([]byte, 64)
	for {
		n, _, readErr := rx.ReadFromUDP(buf)
		if readErr != nil {
			tb.Skipf("loopback broadcast to %s did not reach the bound socket within %v — platform lacks the capability (protocol correctness stays proven by the mirror fixtures)",
				dst, integrationProbeWait)
		}
		if n > 0 {
			return
		}
	}
}

// peerStore adapts *storage.Store to repl.StateStore for the peer node (the
// udprepl smokeStore precedent): remote merges join through Merge, and the
// store keeps the corroborating copy this test reads back through the
// tombstone-visible point read.
type peerStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s peerStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Version returns the stored version for key, tombstones included.
func (s peerStore) Version(key string) (version.RecordVersion, bool) { return s.store.Version(key) }

// Lookup returns the stored record for key, tombstones included.
func (s peerStore) Lookup(key string) (storage.Record, bool) { return s.store.Lookup(key) }

// Versions returns the shallow key→version view, tombstones included.
func (s peerStore) Versions() map[string]version.RecordVersion { return s.store.Versions() }

// Observe absorbs one received version into the store's clock.
func (s peerStore) Observe(ver version.RecordVersion) time.Duration { return s.store.Observe(ver) }

// Generation returns the store's content generation.
func (s peerStore) Generation() uint64 { return s.store.Generation() }

// TestWaitIntegration_ackedThroughRealAdapterTracker drives 05-REVIEW CR-01's
// exact production scenario end to end: one known peer, a Set with
// wait_for_replication, the notify→broadcast→apply→ack cycle over real UDP
// datagrams, and the D-08 report coming back acked=true 1/1 from what the
// real adapter's tracker observed through the core's result stream. The
// tracker is created synchronously when the write is notified (before the
// notification is enqueued), so a cycle that finishes within microseconds
// cannot be missed — there is no appearance window to wait out. Finally the
// Ack chain is corroborated as real: the peer holds the record byte-equal.
func TestWaitIntegration_ackedThroughRealAdapterTracker(t *testing.T) {
	t.Parallel()

	portA := integrationUDPPort(t)
	portB := integrationUDPPort(t)
	probeLoopbackBroadcast(t, portB)

	// Writer node A, production stack shape (cmd/sup-kv wiring): one real
	// persist engine serves as both the gRPC server's store and the
	// adapter's StateStore; the adapter owns the post-write notification
	// (D-05) and both wait seams (API-05).
	hub := grpcapi.NewWatchHub()
	eng, _ := persist.Open(
		42,
		time.Now,
		time.Minute,
		&fakeDumpStore{},
		storage.Config{ValueMaxBytes: testValueLimit, OnChange: hub.Ingest},
		persist.Config{Debounce: time.Millisecond, MaxWait: 50 * time.Millisecond},
	)
	engCtx, cancelEng := context.WithCancel(t.Context())
	go func() { _ = eng.Run(engCtx) }()
	t.Cleanup(cancelEng)
	hubCtx, cancelHub := context.WithCancel(t.Context())
	go func() { _ = hub.Run(hubCtx) }()
	t.Cleanup(cancelHub)

	adapterA, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", portA),
		fmt.Sprintf("127.255.255.255:%d", portB),
		replNodeA, eng, repl.Config{},
	)
	if err != nil {
		t.Fatalf("node A adapter: %v", err)
	}

	// Peer node B: a plain storage store behind the repl.StateStore wrapper
	// — it applies A's DataReplication and answers with the unicast Ack.
	storeB := storage.NewStore(replNodeB, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	adapterB, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", portB),
		fmt.Sprintf("127.255.255.255:%d", portA),
		replNodeB, peerStore{store: storeB}, repl.Config{},
	)
	if err != nil {
		t.Fatalf("node B adapter: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = adapterA.Run(ctx) }()
	go func() { _ = adapterB.Run(ctx) }()

	// Discovery over real broadcast Hellos: A must know B before the write,
	// or the wait-set seeds empty, total is zero and the wait short-circuits
	// at acked=true 0/0 (D-11) — the degenerate path this test must not
	// confuse with success.
	waitFor(t, integrationDeadline, "adapter A to discover adapter B", func() bool {
		return adapterA.KnownPeers() == 1
	})

	opts := openGateOptions()
	opts.KnownPeers = adapterA.KnownPeers
	opts.WaitReplicated = adapterA.WaitReplicated
	api := grpcapi.NewServer(eng, adapterA.NotifyLocalWrite, hub, opts)
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
	client := apiproto.NewSupKVServiceClient(conn)
	health := healthpb.NewHealthClient(conn)
	waitFor(t, gateWait, "the health endpoint to flip SERVING (gate open)", func() bool {
		resp, err := health.Check(t.Context(), &healthpb.HealthCheckRequest{})
		return err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	})

	start := time.Now()
	resp, err := client.Set(t.Context(), &apiproto.SetRequest{
		Key:                "wait/integ",
		Value:              []byte("integration"),
		WaitForReplication: true,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set wait/integ: want OK, got error: %v", err)
	}
	if !resp.GetAcked() || resp.GetAckedCount() != 1 || resp.GetTotalPeers() != 1 {
		t.Fatalf("real-adapter report: want acked=true 1/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= waitAssertBound {
		t.Fatalf("Set took %v: want well under %v — a healthy cycle must resolve on the observed result event, with no appearance phase to burn",
			elapsed, waitAssertBound)
	}

	// Corroboration: the Ack chain was real replication, not a phantom
	// success — node B holds the record byte-equal under the written
	// version.
	var ver version.RecordVersion
	copy(ver[:], resp.GetVersion())
	waitFor(t, integrationDeadline, "node B serving the record byte-equal", func() bool {
		got, err := storeB.Get("wait/integ")
		return err == nil && got.Version == ver && bytes.Equal(got.Data, []byte("integration"))
	})
}

// TestWaitIntegration_staleRecordNotAcked drives PROTO-03 end to end over the
// real adapter pair: node B already holds "wait/stale" under a version about
// 10 s ahead of node A's clock, so A's write arrives as a strictly older
// DataReplication. B answers with its newer record and sends no Ack, and A's
// wait_for_replication reports acked=false — the truthful report the stale
// outcome exists for. B's store keeps the newer record: the stale write is
// answered, never merged.
//
// The seed happens after discovery on purpose: B's startup listing is built
// in the same event-loop iteration that emits the Hello A observes, so it was
// built from an empty store. Seeding earlier would let the listing carry the
// future version to A, where Observe/HLC absorption would make A's write
// strictly newer and turn B's stale branch into a normal apply. The core
// table (TestReplication_staleAnsweredNotAcked) owns the wire-level
// Ack/reply counting; this test asserts only what the real adapter's tracker
// reported and what the peer store retains.
func TestWaitIntegration_staleRecordNotAcked(t *testing.T) {
	t.Parallel()

	portA := integrationUDPPort(t)
	portB := integrationUDPPort(t)
	probeLoopbackBroadcast(t, portB)

	hub := grpcapi.NewWatchHub()
	eng, _ := persist.Open(
		42,
		time.Now,
		time.Minute,
		&fakeDumpStore{},
		storage.Config{ValueMaxBytes: testValueLimit, OnChange: hub.Ingest},
		persist.Config{Debounce: time.Millisecond, MaxWait: 50 * time.Millisecond},
	)
	engCtx, cancelEng := context.WithCancel(t.Context())
	go func() { _ = eng.Run(engCtx) }()
	t.Cleanup(cancelEng)
	hubCtx, cancelHub := context.WithCancel(t.Context())
	go func() { _ = hub.Run(hubCtx) }()
	t.Cleanup(cancelHub)

	adapterA, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", portA),
		fmt.Sprintf("127.255.255.255:%d", portB),
		replNodeA, eng, repl.Config{},
	)
	if err != nil {
		t.Fatalf("node A adapter: %v", err)
	}

	// Peer node B: a plain storage store behind the repl.StateStore wrapper.
	storeB := storage.NewStore(replNodeB, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	adapterB, err := udprepl.New(
		fmt.Sprintf("0.0.0.0:%d", portB),
		fmt.Sprintf("127.255.255.255:%d", portA),
		replNodeB, peerStore{store: storeB}, repl.Config{},
	)
	if err != nil {
		t.Fatalf("node B adapter: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = adapterA.Run(ctx) }()
	go func() { _ = adapterB.Run(ctx) }()

	waitFor(t, integrationDeadline, "adapter A to discover adapter B", func() bool {
		return adapterA.KnownPeers() == 1
	})

	// Seed B with a version ~10 s ahead of A's clock (inside the 60 s
	// max_clock_lead), now that B's startup listing is known to have been
	// built from the empty store.
	const staleKey = "wait/stale"
	staleData := []byte("newer")
	staleVersion := version.PackVersion(time.Now().UnixMilli()+10_000, 0, replNodeB)
	if _, _, mergeErr := storeB.Merge(staleKey, storage.Record{Version: staleVersion, Data: staleData}); mergeErr != nil {
		t.Fatalf("seed node B's newer record: %v", mergeErr)
	}

	opts := openGateOptions()
	opts.KnownPeers = adapterA.KnownPeers
	opts.WaitReplicated = adapterA.WaitReplicated
	api := grpcapi.NewServer(eng, adapterA.NotifyLocalWrite, hub, opts)
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
	client := apiproto.NewSupKVServiceClient(conn)
	health := healthpb.NewHealthClient(conn)
	waitFor(t, gateWait, "the health endpoint to flip SERVING (gate open)", func() bool {
		resp, err := health.Check(t.Context(), &healthpb.HealthCheckRequest{})
		return err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	})

	resp, err := client.Set(t.Context(), &apiproto.SetRequest{
		Key:                staleKey,
		Value:              []byte("older"),
		WaitForReplication: true,
		WaitTimeoutMs:      100,
	})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", staleKey, err)
	}
	if resp.GetAcked() {
		t.Fatalf("stale-write report: want acked=false, got acked=true %d/%d — B answered with its newer record instead of Acking",
			resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if resp.GetTotalPeers() != 1 {
		t.Fatalf("stale-write report total peers = %d, want 1 — A knew B at write time", resp.GetTotalPeers())
	}

	// B answered, never merged: its newer record is untouched.
	got, err := storeB.Get(staleKey)
	if err != nil {
		t.Fatalf("store B Get(%q) after the stale write: %v", staleKey, err)
	}
	if got.Version != staleVersion || !bytes.Equal(got.Data, staleData) {
		t.Fatalf("store B holds (%x, %q), want the untouched newer record (%x, %q) — the stale write was merged",
			got.Version[:], got.Data, staleVersion[:], staleData)
	}
}
