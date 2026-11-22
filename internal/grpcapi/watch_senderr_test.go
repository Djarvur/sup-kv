package grpcapi_test

// The Watch Send-error branch (TEST-14): watch.go has two Send sites — the
// snapshot loop and relayEvent — and both map a failed Send to the same
// action: unsubscribe the handler's subscriber and return the transport
// error unchanged, never blocking the hub or the store. The existing Watch
// suite pins the snapshot-then-live cut, the exact cut under concurrent
// writers, the replicated-change shape and the slow-consumer shed; the one
// uncovered branch is a Send that fails mid-stream.
//
// Two of the subtests drive that branch deterministically with a fake
// stream: real gRPC cannot lose a single Send without racing the stream-
// context branch the handler also selects on, while a fake stream makes
// the failing Send the only outcome. The real-transport subtest covers the
// surrounding contract the branch exists for — a subscriber whose
// transport breaks mid-stream is shed without affecting its peers or the
// store — and the final subtest re-asserts the two existing branches
// (cut ordering, shed with ResourceExhausted) through the same harness, so
// this test extends the suite rather than duplicating it.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

// The Send-error test's keys and fake-server knobs: the snapshot fixtures
// arrive sorted by key, so each subscription's expected order is visible in
// the constants; the flood's version tag is its fake record origin, and the
// flood size is four times the slow-consumer flood, enough to overpower the
// hub's own ingress drain — the other overflow flavor that ends a stream
// with the same resubscribe status.
const (
	senderrSnapKeyA = "senderr/snap-a"
	senderrSnapKeyB = "senderr/snap-b"
	senderrLiveKey  = "senderr/live"
	senderrPostKey  = "senderr/post"
	senderrFreshKey = "senderr/fresh"
	senderrSeedKey  = "senderr/seed"
	senderrCutKeyA  = "senderr/cut-a"
	senderrCutKeyB  = "senderr/cut-b"
	senderrShedKeys = "senderr/shed/"
)

const (
	senderrNodeID         = 42
	senderrEngineDebounce = time.Millisecond
	senderrEngineMaxWait  = 50 * time.Millisecond
	senderrFloodOrigin    = 0x0FE9
	senderrShedFlood      = 4 * watchSlowFlood
)

// errWatchSend is the sentinel a failing Send returns; Watch must return it
// unchanged from both Send sites — that identity is the branch's error
// mapping (watch.go returns the transport error, never a remapped status).
var errWatchSend = errors.New("watch send: transport gone")

// senderrStream is one Watch server stream whose Send fails at a chosen
// call number. The embedded grpc.ServerStream supplies the transport
// methods Watch never touches (it calls only Send and Context), so the
// zero value is a safe test double; sentOne receives one signal after the
// first successful Send, the deterministic hand-off the live-relay subtest
// needs to know the snapshot loop finished.
type senderrStream struct {
	grpc.ServerStream

	ctx     context.Context
	failAt  int
	sends   int
	sentOne chan struct{}
}

// Send counts the call and fails from failAt onward — failAt 1 breaks the
// snapshot loop, failAt 2 lets the snapshot through and breaks the first
// live relay.
func (s *senderrStream) Send(*apiproto.WatchResponse) error {
	s.sends++
	if s.failAt > 0 && s.sends >= s.failAt {
		return errWatchSend
	}
	if s.sentOne != nil {
		select {
		case s.sentOne <- struct{}{}:
		default:
		}
	}
	return nil
}

// Context is the stream context Watch selects on; the fake never cancels
// it, so only the Send failures can end the handler.
func (s *senderrStream) Context() context.Context { return s.ctx }

// newSenderrServer builds the minimal server the two fake-stream subtests
// drive directly: a real persist engine and watch hub with no gRPC
// transport, so the fake stream below is the only Send target. Construction
// order matches the harness: hub first, then persist.Open threading
// hub.Ingest as the store's change tap, then NewServer attaching the engine.
func newSenderrServer(t *testing.T) (*grpcapi.Server, *persist.Engine) {
	t.Helper()
	hub := grpcapi.NewWatchHub()
	eng, _ := persist.Open(
		senderrNodeID,
		time.Now,
		time.Minute,
		&fakeDumpStore{},
		storage.Config{ValueMaxBytes: testValueLimit, OnChange: hub.Ingest},
		persist.Config{Debounce: senderrEngineDebounce, MaxWait: senderrEngineMaxWait},
	)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = eng.Run(ctx) }()
	srv := grpcapi.NewServer(eng, nil, hub, openGateOptions())
	go func() { _ = hub.Run(ctx) }()
	// Watch is gated like every data RPC (API-04, D-01): this fixture calls
	// the handler directly with no gRPC health client, so it opens the gate
	// through the resolver and waits on the in-process health server — the
	// same 1:1 gate/health binding the bufconn harness observes.
	go srv.RunGate(ctx)
	waitFor(t, gateWait, "the senderr server's gate to open", func() bool {
		resp, err := srv.Health().Check(ctx, &healthpb.HealthCheckRequest{})
		return err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	})
	return srv, eng
}

// senderrEngineSet seeds the engine through the real write path and fails
// the test when the store rejects the write.
func senderrEngineSet(t *testing.T, eng *persist.Engine, key, value string) {
	t.Helper()
	_, _, err := eng.Set(key, []byte(value))
	if err != nil {
		t.Fatalf("engine Set %s: want nil, got error: %v", key, err)
	}
}

// senderrClientSet writes through the gRPC client and fails the test when
// the write does not return OK — the store-liveness assertion's shape.
func senderrClientSet(t *testing.T, client apiproto.SupKVServiceClient, key, value string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), watchTestTimeout)
	defer cancel()
	_, err := client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte(value)})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}
}

// TestWatch_sendErrorDisconnectsSubscriber covers the handler-side Send
// error branch (TEST-14, T-2-16): a failed Send deregisters the subscriber
// and ends its RPC without wedging the hub, its peers or the store. The
// first two subtests pin the exact error mapping of the snapshot and live
// Send sites; the third exercises a real transport break end-to-end; the
// fourth re-asserts the existing cut and shed branches so the extension
// keeps them covered in one place.
func TestWatch_sendErrorDisconnectsSubscriber(t *testing.T) {
	t.Parallel()

	t.Run("snapshot send fails", func(t *testing.T) {
		t.Parallel()
		srv, eng := newSenderrServer(t)
		senderrEngineSet(t, eng, senderrSeedKey, "snapshot")
		stream := &senderrStream{ctx: t.Context(), failAt: 1}
		err := srv.Watch(&apiproto.WatchRequest{}, stream)
		if !errors.Is(err, errWatchSend) {
			t.Fatalf("Watch with a failing snapshot Send = %v, want the Send error %v returned unchanged", err, errWatchSend)
		}
		if stream.sends != 1 {
			t.Fatalf("stream Send calls = %d, want 1 (the snapshot loop's failing send)", stream.sends)
		}
	})

	t.Run("live relay send fails", func(t *testing.T) {
		t.Parallel()
		srv, eng := newSenderrServer(t)
		senderrEngineSet(t, eng, senderrSeedKey, "snapshot")
		stream := &senderrStream{ctx: t.Context(), failAt: 2, sentOne: make(chan struct{}, 1)}
		done := make(chan error, 1)
		go func() { done <- srv.Watch(&apiproto.WatchRequest{}, stream) }()
		select {
		case <-stream.sentOne:
		case <-time.After(watchTestTimeout):
			t.Fatal("the snapshot Send did not succeed within the deadline")
		}
		senderrEngineSet(t, eng, senderrLiveKey, "live")
		select {
		case err := <-done:
			if !errors.Is(err, errWatchSend) {
				t.Fatalf("Watch with a failing live relay Send = %v, want the Send error %v returned unchanged", err, errWatchSend)
			}
			if stream.sends != 2 {
				t.Fatalf("stream Send calls = %d, want 2 (snapshot then the failing relay)", stream.sends)
			}
		case <-time.After(watchTestTimeout):
			t.Fatal("the failing live relay Send did not end Watch within the deadline")
		}
	})

	t.Run("broken transport leaves peers and store untouched", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ctx := t.Context()
		for _, key := range []string{senderrSnapKeyA, senderrSnapKeyB} {
			senderrClientSet(t, ts.client, key, "v1")
		}

		brokenCtx, cancelBroken := context.WithCancel(ctx)
		t.Cleanup(cancelBroken)
		broken, err := ts.client.Watch(brokenCtx, &apiproto.WatchRequest{})
		if err != nil {
			t.Fatalf("Watch (subscriber 1): want OK, got error: %v", err)
		}
		for _, want := range []string{senderrSnapKeyA, senderrSnapKeyB} {
			if ev := recvEvent(t, broken); ev.GetRecord().GetKey() != want {
				t.Fatalf("subscriber 1 snapshot event = %s, want %s", ev.GetRecord().GetKey(), want)
			}
		}

		healthy, err := ts.client.Watch(ctx, &apiproto.WatchRequest{})
		if err != nil {
			t.Fatalf("Watch (subscriber 2): want OK, got error: %v", err)
		}
		for _, want := range []string{senderrSnapKeyA, senderrSnapKeyB} {
			if ev := recvEvent(t, healthy); ev.GetRecord().GetKey() != want {
				t.Fatalf("subscriber 2 snapshot event = %s, want %s", ev.GetRecord().GetKey(), want)
			}
		}

		// Break subscriber 1's transport mid-stream, then drive a change:
		// the handler's context branch is the path a real cancel takes (the
		// Send-error mapping itself is pinned by the fake-stream subtests
		// above), and the contract under test is the cleanup it performs.
		cancelBroken()
		_, recvErr := broken.Recv()
		if recvErr == nil {
			t.Fatal("the broken stream delivered an event after its context was cancelled, want a transport error")
		}

		senderrClientSet(t, ts.client, senderrLiveKey, "live")
		live := recvEvent(t, healthy)
		rec := live.GetRecord()
		if rec.GetKey() != senderrLiveKey || rec.GetDeleted() || !bytes.Equal(rec.GetValue(), []byte("live")) {
			t.Fatalf("subscriber 2 live event = (%s, deleted=%t, %q), want (%s, false, live)",
				rec.GetKey(), rec.GetDeleted(), rec.GetValue(), senderrLiveKey)
		}

		// Store liveness: another write still confirms on the same server.
		senderrClientSet(t, ts.client, senderrPostKey, "post")
		resp, err := ts.health.Check(ctx, &healthpb.HealthCheckRequest{})
		if err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
			t.Fatalf("health after the broken stream = (%v, %v), want SERVING", resp.GetStatus(), err)
		}

		// Hub state intact: a fresh subscriber re-snapshots the full current
		// state in sorted order and then receives a new live change.
		fresh, err := ts.client.Watch(ctx, &apiproto.WatchRequest{})
		if err != nil {
			t.Fatalf("Watch (fresh subscriber): want OK, got error: %v", err)
		}
		for _, want := range []string{senderrLiveKey, senderrPostKey, senderrSnapKeyA, senderrSnapKeyB} {
			if ev := recvEvent(t, fresh); ev.GetRecord().GetKey() != want {
				t.Fatalf("fresh subscriber snapshot event = %s, want %s (sorted order)", ev.GetRecord().GetKey(), want)
			}
		}
		senderrClientSet(t, ts.client, senderrFreshKey, "fresh")
		if ev := recvEvent(t, fresh); ev.GetRecord().GetKey() != senderrFreshKey {
			t.Fatalf("fresh subscriber live event = %s, want %s", ev.GetRecord().GetKey(), senderrFreshKey)
		}
	})

	t.Run("cut and shed branches stay covered", func(t *testing.T) {
		t.Parallel()
		ts := newTestServer(t)
		ctx := t.Context()
		for _, key := range []string{senderrCutKeyA, senderrCutKeyB} {
			senderrClientSet(t, ts.client, key, "v1")
		}

		cutCtx, cancelCut := context.WithCancel(ctx)
		t.Cleanup(cancelCut)
		cut, err := ts.client.Watch(cutCtx, &apiproto.WatchRequest{})
		if err != nil {
			t.Fatalf("Watch (cut): want OK, got error: %v", err)
		}
		snapA := recvEvent(t, cut)
		snapB := recvEvent(t, cut)
		if snapA.GetRecord().GetKey() != senderrCutKeyA || snapB.GetRecord().GetKey() != senderrCutKeyB {
			t.Fatalf("cut snapshot keys = (%s, %s), want (%s, %s)",
				snapA.GetRecord().GetKey(), snapB.GetRecord().GetKey(), senderrCutKeyA, senderrCutKeyB)
		}
		senderrClientSet(t, ts.client, senderrCutKeyA, "v2")
		liveCut := recvEvent(t, cut)
		if liveCut.GetRecord().GetKey() != senderrCutKeyA {
			t.Fatalf("cut live event = %s, want %s", liveCut.GetRecord().GetKey(), senderrCutKeyA)
		}
		if bytes.Compare(liveCut.GetRecord().GetVersion(), snapA.GetRecord().GetVersion()) <= 0 {
			t.Fatalf("cut live version %x does not dominate the snapshot version %x",
				liveCut.GetRecord().GetVersion(), snapA.GetRecord().GetVersion())
		}
		cancelCut()

		// The shed half: a subscriber that stops reading is disconnected
		// with ResourceExhausted. The flood drives the hub's ingress — the
		// other overflow flavor beside TestWatch_slowConsumer's per-
		// subscriber channel — and both end the stream with the same
		// resubscribe status.
		shed, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: senderrShedKeys})
		if err != nil {
			t.Fatalf("Watch (shed): want OK, got error: %v", err)
		}
		nowMs := time.Now().UnixMilli()
		for i := range senderrShedFlood {
			ts.hub.Ingest(fmt.Sprintf("%s%05d", senderrShedKeys, i),
				storage.Record{Version: version.PackVersion(nowMs+int64(i), 0, senderrFloodOrigin), Data: []byte("f")})
		}
		got := make(chan error, 1)
		go func() {
			for {
				_, err := shed.Recv()
				if err != nil {
					got <- err
					return
				}
			}
		}()
		select {
		case err := <-got:
			assertCode(t, "Recv on the shed watch stream", err, codes.ResourceExhausted)
		case <-time.After(watchShedDeadline):
			t.Fatal("shed verdict did not arrive within the deadline")
		}
	})
}
