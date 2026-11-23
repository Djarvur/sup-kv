package grpcapi_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// watchTestTimeout bounds every wait for a delivery the pipeline owes the
// subscriber. watchQuietWindow is the silence window the negative proofs
// (merge losers, off-prefix writes) rely on — sound because the pipeline is
// known empty when they run: the preceding positive event arrived through a
// synchronous Recv, so any pending delivery would surface well within the
// window. watchSlowWrites is the slow-consumer test's real-write round,
// watchSlowFlood the Ingest-flood size that deterministically overpowers
// the stalled handler's relay, and watchShedDeadline bounds the wait for
// the shed verdict.
const (
	watchTestTimeout  = 2 * time.Second
	watchQuietWindow  = 200 * time.Millisecond
	watchSlowWrites   = 10
	watchSlowFlood    = 50000
	watchShedDeadline = 10 * time.Second
)

// watchRecv is one Recv outcome shipped out of a receive goroutine — the
// helpers below receive with a deadline, and Recv must run off the test
// goroutine to make that possible.
type watchRecv struct {
	ev  *apiproto.WatchResponse
	err error
}

// recvEvent receives one event from the stream or fails the test at the
// deadline — the deterministic wait for a delivery the buffered pipeline
// owes the subscriber.
func recvEvent(t *testing.T, stream grpc.ServerStreamingClient[apiproto.WatchResponse]) *apiproto.WatchResponse {
	t.Helper()
	ch := make(chan watchRecv, 1)
	go func() {
		ev, err := stream.Recv()
		ch <- watchRecv{ev: ev, err: err}
	}()
	select {
	case got := <-ch:
		if got.err != nil {
			t.Fatalf("Recv: want an event, got error: %v", got.err)
		}
		return got.ev
	case <-time.After(watchTestTimeout):
		t.Fatalf("Recv: no event within %s", watchTestTimeout)
		return nil
	}
}

// recvNone proves the stream stays quiet for watchQuietWindow: the negative
// proof for suppressed events (merge losers, off-prefix writes, a repeated
// confirmation). Sound because the pipeline is known empty when it runs — the
// preceding positive event arrived through a synchronous Recv — so it must be
// the stream's final assertion.
func recvNone(t *testing.T, stream grpc.ServerStreamingClient[apiproto.WatchResponse]) {
	t.Helper()
	ch := make(chan watchRecv, 1)
	go func() {
		ev, err := stream.Recv()
		ch <- watchRecv{ev: ev, err: err}
	}()
	select {
	case got := <-ch:
		t.Fatalf("Recv: want silence for %s, got event %v (err %v)", watchQuietWindow, got.ev, got.err)
	case <-time.After(watchQuietWindow):
	}
}

// TestWatch_snapshotThenLive proves the subscription contract (D-02, D-03
// and the Q3 pin): the first events are the whole current state as ordinary
// WatchResponse events in sorted-key order — the tombstoned key included with
// deleted=true, exactly one message shape, no initial_state type — and the
// next event is a live change whose version strictly dominates the
// snapshot's.
func TestWatch_snapshotThenLive(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	for _, key := range []string{"cut/a", "cut/b", "tmp/x"} {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte("v1")})
		if err != nil {
			t.Fatalf("Set %s: want OK, got error: %v", key, err)
		}
	}
	_, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "tmp/x"})
	if err != nil {
		t.Fatalf("Delete tmp/x: want OK, got error: %v", err)
	}

	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}

	// Snapshot phase: sorted keys, the tombstone as an ordinary deleted=true
	// event (D-02, Q3 — the deliberate asymmetry with List).
	snapA := recvEvent(t, stream)
	snapB := recvEvent(t, stream)
	snapTomb := recvEvent(t, stream)
	if snapA.GetRecord().GetKey() != "cut/a" || snapA.GetRecord().GetDeleted() ||
		!bytes.Equal(snapA.GetRecord().GetValue(), []byte("v1")) {
		t.Fatalf("first snapshot event = (%s, deleted=%t, %q), want (cut/a, false, v1)",
			snapA.GetRecord().GetKey(), snapA.GetRecord().GetDeleted(), snapA.GetRecord().GetValue())
	}
	if snapB.GetRecord().GetKey() != "cut/b" || snapB.GetRecord().GetDeleted() {
		t.Fatalf("second snapshot event = (%s, deleted=%t), want (cut/b, false)",
			snapB.GetRecord().GetKey(), snapB.GetRecord().GetDeleted())
	}
	if snapTomb.GetRecord().GetKey() != "tmp/x" || !snapTomb.GetRecord().GetDeleted() {
		t.Fatalf("third snapshot event = (%s, deleted=%t), want the tmp/x tombstone (tmp/x, true)",
			snapTomb.GetRecord().GetKey(), snapTomb.GetRecord().GetDeleted())
	}
	if len(snapTomb.GetRecord().GetVersion()) != 16 {
		t.Fatalf("tombstone event version size = %d, want the full 16-byte version (D-01)",
			len(snapTomb.GetRecord().GetVersion()))
	}

	// Live phase: an overwrite arrives as the next event — same one message
	// shape (D-03), strictly dominating version, no duplicate of the
	// snapshot row.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "cut/a", Value: []byte("v2")})
	if err != nil {
		t.Fatalf("live Set cut/a: want OK, got error: %v", err)
	}
	live := recvEvent(t, stream)
	rec := live.GetRecord()
	if rec.GetKey() != "cut/a" || rec.GetDeleted() || !bytes.Equal(rec.GetValue(), []byte("v2")) {
		t.Fatalf("live event = (%s, deleted=%t, %q), want (cut/a, false, v2)",
			rec.GetKey(), rec.GetDeleted(), rec.GetValue())
	}
	if bytes.Compare(rec.GetVersion(), snapA.GetRecord().GetVersion()) <= 0 {
		t.Fatalf("live version %x does not dominate the snapshot version %x",
			rec.GetVersion(), snapA.GetRecord().GetVersion())
	}
}

// receivedEvent is one delivered Watch event as the exact-cut test books it.
type receivedEvent struct {
	key     string
	version []byte
	value   []byte
	deleted bool
}

// lastByKeyEvent returns the latest delivered event per key — the final
// state the subscriber's stream reflects.
func lastByKeyEvent(events []receivedEvent) map[string]receivedEvent {
	out := make(map[string]receivedEvent, len(events))
	for _, ev := range events {
		out[ev.key] = ev
	}
	return out
}

// cutDrained reports whether every key's latest delivered event matches the
// live Get — once it holds everywhere the pipeline has fully drained and
// the no-duplicates/no-misses assertions are final.
func cutDrained(t *testing.T, ts *testHarness, keys []string, finals map[string]receivedEvent) bool {
	t.Helper()
	for _, key := range keys {
		get, err := ts.client.Get(t.Context(), &apiproto.GetRequest{Key: key})
		if err != nil {
			t.Fatalf("Get %s: want OK, got error: %v", key, err)
		}
		fin, ok := finals[key]
		if !ok || !bytes.Equal(fin.version, get.GetRecord().GetVersion()) {
			return false
		}
	}
	return true
}

// TestWatch_exactCut proves the boundary holds under concurrent writers
// (research Pattern 2, Pitfall 5): with Sets hammering the seeded keys
// while the subscriber is attached, every delivered (key, version) pair is
// unique — no duplicates across the snapshot/live boundary — each key's
// versions arrive strictly increasing — tap order is install order, D-05 —
// and the union of snapshot and live ends up reflecting the final state
// read back via Get — no misses, D-02.
func TestWatch_exactCut(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	keys := []string{"cut/a", "cut/b", "cut/c"}
	for _, key := range keys {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte("seed")})
		if err != nil {
			t.Fatalf("seed Set %s: want OK, got error: %v", key, err)
		}
	}

	streamCtx, cancelStream := context.WithCancel(ctx)
	t.Cleanup(cancelStream)
	stream, err := ts.client.Watch(streamCtx, &apiproto.WatchRequest{})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}

	// A collector drains the stream for the whole test — the client must
	// never stall, or the subscriber would be shed mid-test instead of
	// proving the cut.
	var (
		mu   sync.Mutex
		seen []receivedEvent
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		for {
			ev, err := stream.Recv()
			if err != nil {
				return
			}
			rec := ev.GetRecord()
			mu.Lock()
			seen = append(seen, receivedEvent{
				key:     rec.GetKey(),
				version: slices.Clone(rec.GetVersion()),
				value:   slices.Clone(rec.GetValue()),
				deleted: rec.GetDeleted(),
			})
			mu.Unlock()
		}
	}()

	// Concurrent writers hammer the seeded keys while the subscriber is
	// attached — the boundary the exact cut must hold across. One Set per
	// goroutine: versions are stamped fresh by the store's clock, so every
	// install is a distinct strictly-greater version.
	const rounds = 60
	var wg sync.WaitGroup
	for i := range rounds {
		wg.Go(func() {
			key := keys[i%len(keys)]
			_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte(fmt.Sprintf("w%03d", i))})
			if err != nil {
				t.Errorf("writer Set(%s): want OK, got error: %v", key, err)
			}
		})
	}
	wg.Wait()

	// The writers stopped: wait until every key's latest delivered event
	// equals the live Get, then the delivery set below is final.
	deadline := time.Now().Add(watchTestTimeout)
	for drained := false; !drained; {
		mu.Lock()
		finals := lastByKeyEvent(seen)
		mu.Unlock()
		drained = cutDrained(t, ts, keys, finals)
		if !drained {
			if time.Now().After(deadline) {
				t.Fatal("watch pipeline did not drain within the deadline")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancelStream()
	<-done

	mu.Lock()
	seenAll := slices.Clone(seen)
	mu.Unlock()
	assertExactCut(t, ts, keys, seenAll)
}

// assertExactCut proves the delivered set is the exact cut over the
// delivered events' slice: at least the snapshot row per key, no (key,
// version) pair delivered twice, each key's versions strictly increasing
// in arrival order, and the stream's final state per key exactly the final
// state read back via Get at test end — version, value and liveness.
func assertExactCut(t *testing.T, ts *testHarness, keys []string, seen []receivedEvent) {
	t.Helper()
	if len(seen) < len(keys) {
		t.Fatalf("delivered %d events, want at least the %d snapshot events", len(seen), len(keys))
	}
	lastByKey := make(map[string][]byte, len(keys))
	pairs := make(map[string]bool, len(seen))
	for i, ev := range seen {
		pair := ev.key + "/" + string(ev.version)
		if pairs[pair] {
			t.Fatalf("event %d: (key %s, version %x) delivered twice — a boundary duplicate",
				i, ev.key, ev.version)
		}
		pairs[pair] = true
		if prev, ok := lastByKey[ev.key]; ok && bytes.Compare(ev.version, prev) <= 0 {
			t.Fatalf("event %d: key %s version %x does not dominate the previously delivered %x — arrival-order violation",
				i, ev.key, ev.version, prev)
		}
		lastByKey[ev.key] = ev.version
	}
	finals := lastByKeyEvent(seen)
	for _, key := range keys {
		get, err := ts.client.Get(t.Context(), &apiproto.GetRequest{Key: key})
		if err != nil {
			t.Fatalf("final Get %s: want OK, got error: %v", key, err)
		}
		fin, ok := finals[key]
		if !ok {
			t.Fatalf("key %s never delivered, want at least its snapshot event", key)
		}
		if !bytes.Equal(fin.version, get.GetRecord().GetVersion()) ||
			!bytes.Equal(fin.value, get.GetRecord().GetValue()) || fin.deleted {
			t.Fatalf("key %s final delivered (%x, %q, deleted=%t) != final Get (%x, %q, deleted=%t)",
				key, fin.version, fin.value, fin.deleted,
				get.GetRecord().GetVersion(), get.GetRecord().GetValue(), get.GetRecord().GetDeleted())
		}
	}
}

// TestWatch_replicatedChange proves D-07 and D-06 end-to-end: a winning
// Engine.ApplyRemote install reaches the stream in exactly the shape a
// local Set produces — full record, no origin field — while a strictly
// older remote record loses silently and fires nothing.
func TestWatch_replicatedChange(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}

	// A local Set is the reference event shape.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "rep/local", Value: []byte("local")})
	if err != nil {
		t.Fatalf("Set rep/local: want OK, got error: %v", err)
	}
	localEv := recvEvent(t, stream)
	if localEv.GetRecord().GetKey() != "rep/local" || len(localEv.GetRecord().GetVersion()) != 16 {
		t.Fatalf("local event = (%s, version size %d), want (rep/local, 16)",
			localEv.GetRecord().GetKey(), len(localEv.GetRecord().GetVersion()))
	}

	// A winning remote merge arrives indistinguishably (D-07): the same
	// WatchResponse shape — full record, no origin field anywhere in it —
	// carrying exactly the applied record.
	nowMs := time.Now().UnixMilli()
	winner := storage.Record{Version: version.PackVersion(nowMs+2, 0, 0x0BBB), Data: []byte("remote")}
	conflicts, _, err := ts.engine.ApplyRemote("rep/remote", winner)
	if err != nil {
		t.Fatalf("ApplyRemote(winner): want nil, got error: %v", err)
	}
	if conflicts != nil {
		t.Fatalf("ApplyRemote(winner) conflicts = %v, want nil (fresh key)", conflicts)
	}
	remoteEv := recvEvent(t, stream)
	rec := remoteEv.GetRecord()
	if rec.GetKey() != "rep/remote" || rec.GetDeleted() ||
		!bytes.Equal(rec.GetValue(), []byte("remote")) || !bytes.Equal(rec.GetVersion(), winner.Version[:]) {
		t.Fatalf("remote event = (%s, deleted=%t, %q, version %x), want the applied record (rep/remote, false, remote, %x)",
			rec.GetKey(), rec.GetDeleted(), rec.GetValue(), rec.GetVersion(), winner.Version[:])
	}

	// A strictly older remote record loses (D-06 end-to-end): the store
	// rejects it, the tap never fires, the stream stays silent.
	loser := storage.Record{Version: version.PackVersion(nowMs+1, 0, 0x0AAA), Data: []byte("stale")}
	conflicts, _, err = ts.engine.ApplyRemote("rep/remote", loser)
	if err != nil {
		t.Fatalf("ApplyRemote(loser): want nil, got error: %v", err)
	}
	want := []storage.Conflict{{Key: "rep/remote", Winner: winner.Version, Loser: loser.Version}}
	if !slices.EqualFunc(conflicts, want, func(a, b storage.Conflict) bool {
		return a.Key == b.Key && a.Winner == b.Winner && a.Loser == b.Loser
	}) {
		t.Fatalf("ApplyRemote(loser) conflicts = %v, want %v", conflicts, want)
	}
	recvNone(t, stream)
}

// TestWatch_prefix proves D-04 on both phases: the snapshot of a prefixed
// subscription carries only its subset, the empty prefix sees all keys
// (tombstone included, the Q3 asymmetry with List), and the live phase is
// filtered the same way — on-prefix writes delivered, off-prefix ones
// never.
func TestWatch_prefix(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	for _, key := range []string{"svc/a", "svc/b", "cfg/a", "tmp/x"} {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte("v")})
		if err != nil {
			t.Fatalf("Set %s: want OK, got error: %v", key, err)
		}
	}
	_, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{Key: "tmp/x"})
	if err != nil {
		t.Fatalf("Delete tmp/x: want OK, got error: %v", err)
	}

	// The prefix stream's snapshot carries only the svc/ subset.
	svcStream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: "svc/"})
	if err != nil {
		t.Fatalf("Watch svc/: want OK, got error: %v", err)
	}
	svcA := recvEvent(t, svcStream)
	svcB := recvEvent(t, svcStream)
	if svcA.GetRecord().GetKey() != "svc/a" || svcB.GetRecord().GetKey() != "svc/b" {
		t.Fatalf("prefix snapshot keys = (%s, %s), want (svc/a, svc/b)",
			svcA.GetRecord().GetKey(), svcB.GetRecord().GetKey())
	}

	// The empty-prefix stream sees everything, tombstone included.
	allStream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{})
	if err != nil {
		t.Fatalf("Watch empty prefix: want OK, got error: %v", err)
	}
	for _, wantKey := range []string{"cfg/a", "svc/a", "svc/b"} {
		ev := recvEvent(t, allStream)
		if ev.GetRecord().GetKey() != wantKey {
			t.Fatalf("empty-prefix snapshot event = %s, want %s (sorted order)",
				ev.GetRecord().GetKey(), wantKey)
		}
	}
	allTomb := recvEvent(t, allStream)
	if allTomb.GetRecord().GetKey() != "tmp/x" || !allTomb.GetRecord().GetDeleted() {
		t.Fatalf("empty-prefix tombstone event = (%s, deleted=%t), want (tmp/x, true)",
			allTomb.GetRecord().GetKey(), allTomb.GetRecord().GetDeleted())
	}

	// Live phase filtered the same way: an on-prefix write is delivered to
	// the prefixed stream.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "svc/c", Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set svc/c: want OK, got error: %v", err)
	}
	svcC := recvEvent(t, svcStream)
	if svcC.GetRecord().GetKey() != "svc/c" {
		t.Fatalf("prefix live event = %s, want svc/c", svcC.GetRecord().GetKey())
	}

	// An off-prefix write never reaches the prefixed stream (the negative
	// proof is final for this stream).
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "cfg/other", Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set cfg/other: want OK, got error: %v", err)
	}
	recvNone(t, svcStream)
}

// TestWatch_slowConsumer proves the shed policy end-to-end (A3, T-05-05):
// a subscriber that stops reading is disconnected with ResourceExhausted
// once its bounded channel fills, while the store keeps accepting and
// confirming writes throughout — the tap never blocks the store write lock,
// and the disconnect (never a silent event drop) keeps D-02 honest because
// resubscription re-snapshots.
//
// The determinism trick: a subscriber that has stopped reading still has
// its events relayed into gRPC's own transport buffering by the handler,
// so real writes alone (paced by dump confirmation) cannot overflow the
// buffer-2 channel on any test-friendly timescale. The flood below
// exercises the exact entry point production uses — hub.Ingest, the
// storage.Config.OnChange target — and overpowers the handler's drain:
// the hub dispatches an event in nanoseconds (prefix check, map lookup,
// channel send) while the handler's per-event relay costs microseconds
// (proto marshal, transport write), so the subscriber's channel pins full
// within the flood's first events and the shed fires no matter how the
// schedulers interleave. Whether the per-subscriber channel or the hub
// ingress overflows first, both shed paths end the stream with the same
// resubscribe status.
func TestWatch_slowConsumer(t *testing.T) {
	t.Parallel()
	ts := newTestServerWithWatchBuffer(t, 2)
	ctx := t.Context()

	for _, key := range []string{"big/a", "big/b"} {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte("v")})
		if err != nil {
			t.Fatalf("seed Set %s: want OK, got error: %v", key, err)
		}
	}

	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: "big/"})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}
	snapA := recvEvent(t, stream)
	snapB := recvEvent(t, stream)
	if snapA.GetRecord().GetKey() != "big/a" || snapB.GetRecord().GetKey() != "big/b" {
		t.Fatalf("snapshot keys = (%s, %s), want (big/a, big/b)",
			snapA.GetRecord().GetKey(), snapB.GetRecord().GetKey())
	}

	// The subscriber stops reading here. Real writes keep completing on the
	// store — every one must succeed (store liveness under a wedged
	// subscriber, T-05-05) — while the hub's dispatch sits far enough ahead
	// of the stalled handler's relay to overflow the buffer-2 channel and
	// shed this subscriber.
	for i := range watchSlowWrites {
		_, err := ts.client.Set(ctx, &apiproto.SetRequest{
			Key:   fmt.Sprintf("big/w%03d", i),
			Value: []byte("w"),
		})
		if err != nil {
			t.Fatalf("Set big/w%03d with a wedged subscriber: want OK (store liveness), got error: %v", i, err)
		}
	}

	// The stalled reader resumes only after the writes, draining whatever
	// the handler relayed before the shed and collecting the verdict.
	got := make(chan error, 1)
	go func() {
		for {
			_, err := stream.Recv()
			if err != nil {
				got <- err
				return
			}
		}
	}()

	// The flood: the OnChange entry point at production rates. Generation 0
	// is already at or below the starting watermark, so these synthetic
	// events release immediately — the flood exercises the shed path, not the
	// durability gate (TestWatchHub_confirmGate and the held-dump tests own
	// that).
	nowMs := time.Now().UnixMilli()
	for i := range watchSlowFlood {
		ts.hub.Ingest(fmt.Sprintf("big/f%05d", i),
			storage.Record{Version: version.PackVersion(nowMs+int64(i), 0, 0x0FE9), Data: []byte("f")}, 0)
	}
	select {
	case err := <-got:
		assertCode(t, "Recv on the shed watch stream", err, codes.ResourceExhausted)
	case <-time.After(watchShedDeadline):
		t.Fatal("shed verdict did not arrive within the deadline")
	}
}

// TestWatch_unconfirmedNotAnnounced proves the durability gate end-to-end
// (D-04, APIS-04, T-06-17): an install the covering dump has not yet saved is
// not announced, and the successful dump that makes it durable releases it
// exactly once. The seed write proves the subscription is registered before
// the held write starts — the hub takes the snapshot only after registering
// the subscriber, so the snapshot event cannot arrive unregistered — and the
// single pending Recv serves both the silence proof and the later delivery: a
// timed-out select leaves that goroutine blocked in Recv, so it consumes the
// first released event instead of racing a second receiver.
func TestWatch_unconfirmedNotAnnounced(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/seed", Value: []byte("s")})
	if err != nil {
		t.Fatalf("seed Set gate/seed: want OK, got error: %v", err)
	}
	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: "gate/"})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}
	if snap := recvEvent(t, stream); snap.GetRecord().GetKey() != "gate/seed" {
		t.Fatalf("snapshot event = %s, want gate/seed", snap.GetRecord().GetKey())
	}

	pending := make(chan watchRecv, 1)
	go func() {
		ev, recvErr := stream.Recv()
		pending <- watchRecv{ev: ev, err: recvErr}
	}()

	// Park every dump: the next install lands in memory but is not durable.
	ts.dumps.holdDumps()
	setDone := make(chan error, 1)
	go func() {
		_, setErr := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gate/a", Value: []byte("held")})
		setDone <- setErr
	}()
	waitFor(t, gateWait, "the held write to install in memory", func() bool {
		_, getErr := ts.engine.Get("gate/a")
		return getErr == nil
	})

	select {
	case got := <-pending:
		t.Fatalf("Recv: announced an unconfirmed write before its dump (event %v, err %v)", got.ev, got.err)
	case <-time.After(watchQuietWindow):
	}
	select {
	case setErr := <-setDone:
		t.Fatalf("Set returned while its covering dump was held: %v", setErr)
	default:
	}

	ts.dumps.releaseDumps()
	select {
	case setErr := <-setDone:
		if setErr != nil {
			t.Fatalf("Set after the dump released: want OK, got error: %v", setErr)
		}
	case <-time.After(watchTestTimeout):
		t.Fatal("Set did not return within the deadline after the dump was released")
	}
	var live *apiproto.WatchResponse
	select {
	case got := <-pending:
		if got.err != nil {
			t.Fatalf("Recv: want the released event, got error: %v", got.err)
		}
		live = got.ev
	case <-time.After(watchTestTimeout):
		t.Fatal("the released durable event did not arrive within the deadline")
	}
	rec := live.GetRecord()
	if rec.GetKey() != "gate/a" || !bytes.Equal(rec.GetValue(), []byte("held")) {
		t.Fatalf("released event = (%s, %q), want (gate/a, held)", rec.GetKey(), rec.GetValue())
	}
	recvNone(t, stream) // exactly once: the final assertion.
}

// TestWatch_confirmedExactlyOnce pins the ordinary write's event count
// (APIS-04, D-04): one confirmed write yields exactly one live event — the
// debounce/dump cycle behind the write must not duplicate it. The seed
// snapshot proves the subscription is registered before the observed write.
func TestWatch_confirmedExactlyOnce(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "once/seed", Value: []byte("s")})
	if err != nil {
		t.Fatalf("seed Set once/seed: want OK, got error: %v", err)
	}
	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: "once/"})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}
	if snap := recvEvent(t, stream); snap.GetRecord().GetKey() != "once/seed" {
		t.Fatalf("snapshot event = %s, want once/seed", snap.GetRecord().GetKey())
	}

	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "once/a", Value: []byte("v1")})
	if err != nil {
		t.Fatalf("Set once/a: want OK, got error: %v", err)
	}
	live := recvEvent(t, stream)
	rec := live.GetRecord()
	if rec.GetKey() != "once/a" || rec.GetDeleted() || !bytes.Equal(rec.GetValue(), []byte("v1")) {
		t.Fatalf("live event = (%s, deleted=%t, %q), want (once/a, false, v1)",
			rec.GetKey(), rec.GetDeleted(), rec.GetValue())
	}
	recvNone(t, stream) // exactly once: the final assertion.
}

// TestWatch_dumpFailedThenConfirmed proves the gate's hardest case (D-04,
// T-06-17) through the API surface: a write whose dump failed answers
// Unavailable and stays unannounced, and a later successful dump — forced by
// a second write — releases both installs exactly once, in install order. A
// handler-side "notify after Set" would have announced the first write
// before the only dump that made it durable.
func TestWatch_dumpFailedThenConfirmed(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gatefail/seed", Value: []byte("s")})
	if err != nil {
		t.Fatalf("seed Set gatefail/seed: want OK, got error: %v", err)
	}
	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: "gatefail/"})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}
	if snap := recvEvent(t, stream); snap.GetRecord().GetKey() != "gatefail/seed" {
		t.Fatalf("snapshot event = %s, want gatefail/seed", snap.GetRecord().GetKey())
	}

	pending := make(chan watchRecv, 1)
	go func() {
		ev, recvErr := stream.Recv()
		pending <- watchRecv{ev: ev, err: recvErr}
	}()

	ts.dumps.mu.Lock()
	ts.dumps.failSave = true
	ts.dumps.mu.Unlock()
	_, setErr := ts.client.Set(ctx, &apiproto.SetRequest{Key: "gatefail/a", Value: []byte("unconfirmed")})
	if status.Code(setErr) != codes.Unavailable {
		t.Fatalf("Set during the failed dump = %v, want codes.Unavailable", setErr)
	}
	if got := status.Convert(setErr).Message(); got != persist.ErrDumpFailed.Error() {
		t.Fatalf("Set during the failed dump message = %q, want %q", got, persist.ErrDumpFailed.Error())
	}
	select {
	case got := <-pending:
		t.Fatalf("Recv: announced a write whose dump failed (event %v, err %v)", got.ev, got.err)
	case <-time.After(watchQuietWindow):
	}

	ts.dumps.mu.Lock()
	ts.dumps.failSave = false
	ts.dumps.mu.Unlock()
	// The second write forces a covering dump that includes the first key, so
	// the failed-then-confirmed install becomes durable and is released.
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "gatefail/b", Value: []byte("second")})
	if err != nil {
		t.Fatalf("Set gatefail/b after healing the dump: want OK, got error: %v", err)
	}

	var firstEvent *apiproto.WatchResponse
	select {
	case got := <-pending:
		if got.err != nil {
			t.Fatalf("Recv: want the first released event, got error: %v", got.err)
		}
		firstEvent = got.ev
	case <-time.After(watchTestTimeout):
		t.Fatal("the first released event did not arrive within the deadline after the covering dump")
	}
	first := firstEvent.GetRecord()
	if first.GetKey() != "gatefail/a" || !bytes.Equal(first.GetValue(), []byte("unconfirmed")) {
		t.Fatalf("first released event = (%s, %q), want (gatefail/a, unconfirmed) — install order",
			first.GetKey(), first.GetValue())
	}
	second := recvEvent(t, stream).GetRecord()
	if second.GetKey() != "gatefail/b" || !bytes.Equal(second.GetValue(), []byte("second")) {
		t.Fatalf("second released event = (%s, %q), want (gatefail/b, second)",
			second.GetKey(), second.GetValue())
	}
	recvNone(t, stream) // exactly once per key: the final assertion.
}

// TestWatchHub_confirmGate pins the hub's durability gate deterministically
// (APIS-04, D-04, T-06-17): an event whose generation is above the confirmed
// watermark is not delivered, Confirm(gen) releases it exactly once, and a
// second Confirm of the same generation delivers nothing. The seed write
// proves the subscription is registered; the synthetic event's generation is
// far above the watermark the seed's own dump left, so the gate — not timing
// — is what holds it.
func TestWatchHub_confirmGate(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "hubgate/seed", Value: []byte("s")})
	if err != nil {
		t.Fatalf("seed Set hubgate/seed: want OK, got error: %v", err)
	}
	stream, err := ts.client.Watch(ctx, &apiproto.WatchRequest{Prefix: "hubgate/"})
	if err != nil {
		t.Fatalf("Watch: want OK, got error: %v", err)
	}
	if snap := recvEvent(t, stream); snap.GetRecord().GetKey() != "hubgate/seed" {
		t.Fatalf("snapshot event = %s, want hubgate/seed", snap.GetRecord().GetKey())
	}

	pending := make(chan watchRecv, 1)
	go func() {
		ev, recvErr := stream.Recv()
		pending <- watchRecv{ev: ev, err: recvErr}
	}()

	const heldGen = 5
	held := storage.Record{
		Version: version.PackVersion(time.Now().UnixMilli(), 0, 0x0A11),
		Data:    []byte("held"),
	}
	ts.hub.Ingest("hubgate/a", held, heldGen)
	select {
	case got := <-pending:
		t.Fatalf("Recv: delivered generation %d before Confirm reached it (event %v, err %v)", heldGen, got.ev, got.err)
	case <-time.After(watchQuietWindow):
	}

	ts.hub.Confirm(heldGen)
	var live *apiproto.WatchResponse
	select {
	case got := <-pending:
		if got.err != nil {
			t.Fatalf("Recv: want the released event, got error: %v", got.err)
		}
		live = got.ev
	case <-time.After(watchTestTimeout):
		t.Fatalf("Confirm(%d) did not release the held event within the deadline", heldGen)
	}
	rec := live.GetRecord()
	if rec.GetKey() != "hubgate/a" || !bytes.Equal(rec.GetValue(), []byte("held")) {
		t.Fatalf("released event = (%s, %q), want (hubgate/a, held)", rec.GetKey(), rec.GetValue())
	}

	ts.hub.Confirm(heldGen) // the repeated confirmation delivers nothing.
	recvNone(t, stream)
}
