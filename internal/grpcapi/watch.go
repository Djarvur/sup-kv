package grpcapi

import (
	"context"
	"slices"
	"strings"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

const (
	// watchBuffer is the per-subscriber event channel depth. A subscriber
	// whose stream stops draining it is disconnected once this many
	// undelivered events have queued up — bounded memory per subscriber
	// (T-05-06's operational envelope), and the disconnect (never a silent
	// event drop, A3) is what keeps the D-02 no-holes contract honest: the
	// shed client resubscribes and re-snapshots. 256 absorbs a write burst
	// far beyond the design envelope while a healthy reader catches up.
	watchBuffer = 256

	// watchIngress bounds the hub's single event queue fed by the store's
	// OnChange tap from under the store write lock (T-05-05: the store
	// never waits on any subscriber). Beyond it Ingest only raises the
	// overflow flag — the hub answers by disconnecting every subscriber
	// with the resubscribe status, never by silently dropping events,
	// because shed subscribers re-snapshot on resubscribe (D-02).
	watchIngress = 4096
)

// watchOverwhelmed returns the status a shed subscriber's stream ends with:
// the slow-consumer policy is disconnect, never drop events (A3) — the
// client resubscribes and re-snapshots, losing nothing (D-02). A fresh
// value per disconnect: status errors are single-use response payloads.
func watchOverwhelmed() error {
	return status.Error(codes.ResourceExhausted, "watch subscriber overwhelmed; resubscribe")
}

// watchEvent is one change notification inside the hub: the key and the
// stored record the tap handed over. The store replaces records wholesale
// on the next install and never mutates one in place, so a single install
// fans out to every subscriber without copying.
type watchEvent struct {
	key string
	rec storage.Record
}

// toProto maps one event onto the wire message (D-01): snapshot and live
// events share the one WatchResponse shape, tombstones included with
// deleted=true (D-02), and carry no origin field — local and replicated
// changes are indistinguishable (D-07).
func (e watchEvent) toProto() *apiproto.WatchResponse {
	return &apiproto.WatchResponse{Record: recordToProto(e.key, e.rec)}
}

// watchSubscriber is one Watch RPC attached to the hub. Every field is
// hub-owned — only the hub goroutine reads or writes it after registration,
// so no synchronization is needed beyond the events channel itself.
type watchSubscriber struct {
	// events buffers the live events awaiting the stream handler. The hub
	// closes it exactly once — err written first — on disconnect or hub
	// shutdown, ending the handler's receive loop.
	events chan watchEvent

	// prefix filters both the snapshot and every live event (D-04); the
	// empty prefix matches all keys.
	prefix string

	// snapshot holds the per-key version each live event must strictly
	// dominate to be delivered (the exact-cut rule): only strictly greater
	// versions ever install, so an event not dominating the snapshot
	// version is necessarily already contained in the snapshot state, and
	// a key absent from the snapshot has nothing to dominate.
	snapshot map[string]version.RecordVersion

	// err is the stream's termination status for a closed events channel:
	// the resubscribe status on a disconnect, nil on hub shutdown. The hub
	// writes it before closing the channel, and a receive observing the
	// close happens after that write (Go memory model), so the handler
	// reads it without further synchronization.
	err error
}

// watchSubscribeRequest is one subscription request on the hub's subscribe
// channel: the prefix to filter by (D-04) and the buffered reply the hub
// answers once the subscriber is registered and its snapshot taken —
// buffered, so the hub never blocks on a handler that moved on.
type watchSubscribeRequest struct {
	prefix string
	ready  chan watchSubscription
}

// watchSubscription is the subscribe reply: the registered subscriber (the
// handler holds it for receives, unsubscription and the disconnect status)
// and the snapshot events to stream before entering the live loop (D-03).
type watchSubscription struct {
	sub      *watchSubscriber
	snapshot []watchEvent
}

// WatchHub fans the storage install tap out to Watch subscribers. One
// goroutine — the caller's Run — owns all subscriber state (the udprepl
// event-loop discipline): the tap enqueues into the bounded ingress without
// blocking (it runs under the store write lock, T-05-05), subscribe and
// unsubscribe requests arrive on channels, and the hub serializes snapshot
// taking and fan-out — which is what makes the snapshot-then-live cut exact
// by construction (research Pattern 2). On either overflow flavor (a full
// subscriber channel or a full ingress) the hub disconnects subscribers
// with the resubscribe status instead of dropping events silently; shed
// clients re-snapshot on resubscribe, so the D-02 no-holes contract
// survives overload. Each subscriber costs one bounded channel and v1 has
// no per-subscriber cap — the trusted-LAN operational envelope recorded as
// accepted (T-05-06), to revisit with the Phase 6 observability work.
type WatchHub struct {
	// engine is attached by NewServer — the hub is constructed first so
	// persist.Open can thread the tap through storage.Config, and the
	// snapshot at subscribe time must be taken on the hub goroutine (the
	// exact cut), hence the engine reference rather than a caller-supplied
	// Records snapshot.
	engine *persist.Engine

	ingress       chan watchEvent
	subscribeCh   chan watchSubscribeRequest
	unsubscribeCh chan *watchSubscriber
	subs          map[*watchSubscriber]struct{}
	overflow      atomic.Bool
	done          chan struct{}
	buffer        int
}

// NewWatchHub returns a hub with the default per-subscriber buffer
// (watchBuffer). The hub attaches to an engine in NewServer and the caller
// launches Run exactly once, before the gRPC server starts serving Watch.
func NewWatchHub() *WatchHub {
	return newWatchHub(watchBuffer)
}

// NewWatchHubWithBuffer is NewWatchHub with the per-subscriber buffer set
// explicitly (watchBuffer when not positive) — the knob that shrinks the
// buffer to force the disconnect path deterministically in tests.
func NewWatchHubWithBuffer(subscriberBuffer int) *WatchHub {
	if subscriberBuffer <= 0 {
		subscriberBuffer = watchBuffer
	}
	return newWatchHub(subscriberBuffer)
}

// newWatchHub builds the hub with an explicit subscriber buffer.
func newWatchHub(buffer int) *WatchHub {
	return &WatchHub{
		ingress:       make(chan watchEvent, watchIngress),
		subscribeCh:   make(chan watchSubscribeRequest),
		unsubscribeCh: make(chan *watchSubscriber),
		subs:          make(map[*watchSubscriber]struct{}),
		done:          make(chan struct{}),
		buffer:        buffer,
	}
}

// attach wires the engine the hub snapshots from at subscribe time; the
// single call site is NewServer, so the harness's and main's construction
// order (hub first, engine second — persist.Open threads the tap through
// storage.Config) keeps the exact-cut property: the snapshot is taken on
// the hub goroutine, never handed in pre-built.
func (h *WatchHub) attach(engine *persist.Engine) {
	h.engine = engine
}

// Run is the single hub goroutine the caller launches. It services the
// subscribe and unsubscribe requests, fans every ingested install event
// out to the matching subscribers in arrival order, and — on either
// overflow flavor — sheds subscribers with the resubscribe status instead
// of dropping events (D-02). Run returns ctx's error once the context is
// cancelled, closing every subscriber's channel with a nil termination
// status and the done channel that guards the late subscribe and
// unsubscribe sends. Plain channel selects, no timers (the persist engine
// timer discipline).
func (h *WatchHub) Run(ctx context.Context) error {
	defer close(h.done)
	defer h.shutdownSubscribers()
	for {
		if h.overflow.CompareAndSwap(true, false) {
			h.disconnectAll()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev := <-h.ingress:
			h.dispatch(ev)
		case req := <-h.subscribeCh:
			h.handleSubscribe(req)
		case sub := <-h.unsubscribeCh:
			h.remove(sub)
		}
	}
}

// Ingest is the storage.Config.OnChange target (the change tap): it hands
// one install event to the hub through a non-blocking send into the
// bounded ingress, so the store write lock is never held waiting on the
// Watch surface (T-05-05). A full ingress is never a silent drop: the
// overflow flag makes the hub disconnect every subscriber on its next
// iteration, and they re-snapshot on resubscribe (D-02). The handed record
// is the store's stored copy — never mutated in place — so the hub may fan
// it out to every subscriber without copying. Safe from any goroutine;
// never blocks.
func (h *WatchHub) Ingest(key string, rec storage.Record) {
	select {
	case h.ingress <- watchEvent{key: key, rec: rec}:
	default:
		h.overflow.Store(true)
	}
}

// subscribe registers a new subscriber from the Watch handler goroutine
// and returns it with its snapshot events, or ok false once the hub has
// shut down. The hub goroutine takes the engine snapshot while servicing
// the request — the exact-cut point.
func (h *WatchHub) subscribe(prefix string) (watchSubscription, bool) {
	req := watchSubscribeRequest{prefix: prefix, ready: make(chan watchSubscription, 1)}
	select {
	case h.subscribeCh <- req:
	case <-h.done:
		return watchSubscription{}, false
	}
	select {
	case got := <-req.ready:
		return got, true
	case <-h.done:
		// The hub may answer and shut down between the two selects; a
		// buffered reply here belongs to a subscriber the shutdown sweep
		// already closed and removed, so refusing is the correct verdict.
		return watchSubscription{}, false
	}
}

// unsubscribe removes a subscriber whose handler is returning — client
// disconnect or a failed stream send — so the hub stops fanning out to it.
// A no-op once the hub has shut down: the shutdown sweep already removed
// every subscriber.
func (h *WatchHub) unsubscribe(sub *watchSubscriber) {
	select {
	case h.unsubscribeCh <- sub:
	case <-h.done:
	}
}

// handleSubscribe registers the subscriber and takes the exact-cut
// snapshot on the hub goroutine: every event the hub relays from this
// point on either strictly dominates the snapshot version for its key
// (delivered) or is already contained in the snapshot state (suppressed),
// with no lock over the store beyond Records' read snapshot. Tombstones
// are part of the state, so the snapshot delivers them as ordinary
// deleted=true events (D-02, the Q3 pin — List excluding them is the
// deliberate asymmetry). The snapshot is emitted in sorted-key order —
// arrival order (D-05) governs live changes, not the initial state.
func (h *WatchHub) handleSubscribe(req watchSubscribeRequest) {
	records := h.engine.Records()
	sub := &watchSubscriber{
		events:   make(chan watchEvent, h.buffer),
		prefix:   req.prefix,
		snapshot: make(map[string]version.RecordVersion, len(records)),
	}
	snapshot := make([]watchEvent, 0, len(records))
	for key, rec := range records {
		if !strings.HasPrefix(key, sub.prefix) {
			continue
		}
		sub.snapshot[key] = rec.Version
		snapshot = append(snapshot, watchEvent{key: key, rec: rec})
	}
	slices.SortFunc(snapshot, func(a, b watchEvent) int {
		return strings.Compare(a.key, b.key)
	})
	h.subs[sub] = struct{}{}
	req.ready <- watchSubscription{sub: sub, snapshot: snapshot}
}

// dispatch fans one install event out to the matching subscribers in
// arrival order — tap order is install order under the store write lock
// (D-05), so no re-sorting anywhere — with the exact-cut suppression rule
// and the non-blocking per-subscriber send: a full channel disconnects
// that subscriber only (A3), never blocks the hub or the store.
func (h *WatchHub) dispatch(ev watchEvent) {
	for sub := range h.subs {
		if !h.wants(sub, ev) {
			continue
		}
		select {
		case sub.events <- ev:
		default:
			h.disconnect(sub)
		}
	}
}

// wants reports whether sub should receive ev: the prefix filter (D-04)
// and the exact-cut suppression — a live event whose version does not
// strictly dominate the snapshot version for its key is necessarily
// already contained in the snapshot state (only strictly greater versions
// ever install), so it is suppressed; a key absent from the snapshot is
// new information and always delivered.
func (*WatchHub) wants(sub *watchSubscriber, ev watchEvent) bool {
	if !strings.HasPrefix(ev.key, sub.prefix) {
		return false
	}
	snapVer, seen := sub.snapshot[ev.key]
	return !seen || ev.rec.Version.Compare(snapVer) > 0
}

// disconnect ends one overwhelmed subscriber: the resubscribe status goes
// into sub.err before the channel close (the close happens-before the
// handler's zero-value receive), and the subscriber leaves the table —
// resubscription re-snapshots, which is what keeps the stream hole-free
// under overload (D-02, A3).
func (h *WatchHub) disconnect(sub *watchSubscriber) {
	sub.err = watchOverwhelmed()
	delete(h.subs, sub)
	close(sub.events)
}

// disconnectAll sheds every current subscriber on a hub-ingress overflow:
// events past the ingress bound were dropped — silently losing them to
// connected subscribers would violate D-02 — so the honest verdict is to
// end every stream with the resubscribe status; the re-snapshot on
// resubscribe heals the loss.
func (h *WatchHub) disconnectAll() {
	for sub := range h.subs {
		h.disconnect(sub)
	}
}

// remove drops a subscriber whose handler is leaving on its own (client
// disconnect, failed send): nothing is closed — the handler owns the only
// remaining reference and is returning, so the channel is garbage after
// that.
func (h *WatchHub) remove(sub *watchSubscriber) {
	delete(h.subs, sub)
}

// shutdownSubscribers closes every subscriber's channel with a nil
// termination status — the ctx-cancel path: the streams end cleanly while
// the process is shutting down around them.
func (h *WatchHub) shutdownSubscribers() {
	for sub := range h.subs {
		delete(h.subs, sub)
		close(sub.events)
	}
}

// Watch streams the subscriber's snapshot as ordinary WatchResponse events (D-03)
// followed by every live change under the prefix in arrival order (D-05) —
// tombstones included as deleted=true (D-02), no origin marking (D-07).
// The hub owns the exact cut; the handler streams the snapshot, then
// relays the hub's buffered events until the stream context ends or the
// hub sheds this subscriber with ResourceExhausted — the slow-consumer
// disconnect whose resubscribe-and-resnapshot heals the shed (D-02, A3).
func (s *Server) Watch(req *apiproto.WatchRequest, stream apiproto.SupKVService_WatchServer) error {
	if s.hub == nil {
		return status.Error(codes.Unimplemented, "watch requires a hub; construct the server through NewServer")
	}
	got, ok := s.hub.subscribe(req.GetPrefix())
	if !ok {
		return status.Error(codes.Unavailable, "watch hub is shutting down")
	}
	for _, ev := range got.snapshot {
		err := stream.Send(ev.toProto())
		if err != nil {
			s.hub.unsubscribe(got.sub)
			return err
		}
	}
	for {
		select {
		case <-stream.Context().Done():
			s.hub.unsubscribe(got.sub)
			return stream.Context().Err()
		case ev, open := <-got.sub.events:
			if !open {
				// The hub closed the channel: sub.err was written before
				// the close, and the zero-value receive happens after it,
				// so the read is synchronized by the close itself — the
				// resubscribe status on a disconnect, nil on hub shutdown.
				return got.sub.err
			}
			err := s.relayEvent(stream, got.sub, ev)
			if err != nil {
				return err
			}
		}
	}
}

// relayEvent streams one relayed live event; a failed send means the
// client is gone, so the subscriber is deregistered. Any non-nil error
// ends the RPC.
func (s *Server) relayEvent(stream apiproto.SupKVService_WatchServer, sub *watchSubscriber, ev watchEvent) error {
	err := stream.Send(ev.toProto())
	if err != nil {
		s.hub.unsubscribe(sub)
		return err
	}
	return nil
}
