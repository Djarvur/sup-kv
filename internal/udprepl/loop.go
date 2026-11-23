package udprepl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

const (
	// inboundQueue bounds the datagrams waiting for the event loop; beyond
	// it the read goroutine blocks and the socket receive buffer
	// backpressences the segment.
	inboundQueue = 64

	// notifyQueue bounds the local-write notifications waiting for the
	// event loop. NotifyLocalWrite only enqueues and never blocks on
	// replication progress (D-05 — the write path stays available); a
	// wedged loop eventually fills the queue and blocks authors, a
	// bounded backpressure that never silently drops a write.
	notifyQueue = 1024

	// readErrorBackoff is the pause after a non-terminal receive error
	// (XPORT-08): a persistent kernel error must never turn the read loop
	// into a busy spin — the counter and the WARN are the signals, and the
	// loop keeps reading the next datagram after the pause.
	readErrorBackoff = 100 * time.Millisecond

	// readErrorLogInterval is the read-error rate limit: at most one log
	// line per interval keeps a flood of receive errors from flooding the
	// log (T-3-36) while the counter carries the reliable signal. Adapter
	// side, so the wall clock is the sanctioned time source here.
	readErrorLogInterval = time.Second
)

// ErrRunLoopExited is the NotifyLocalWrite verdict once the adapter's event
// loop has returned: nothing drains the notify queue any more, so the
// notification is dropped rather than enqueued — the caller (Phase 5's
// gRPC write path) fails the write visibly instead of the call blocking
// forever on a queue nobody serves.
var ErrRunLoopExited = errors.New("udprepl: replication event loop has exited")

// inboundEvent is one received datagram on the event loop's inbound queue,
// with the bytes defensively copied off the read buffer (which is reused on
// the next read) and the sender's address as the opaque handle string
// (src.String(), D-03).
type inboundEvent struct {
	datagram []byte
	source   string
}

// localWrite is one author's NotifyLocalWrite notification on the event
// loop's queue, with the record defensively copied at enqueue — the author
// may reuse its buffer once the call returns. The queue carries no track
// flag: the tracker is created synchronously in NotifyLocalWrite before the
// enqueue, so no later code can mistake a notification for a tracker
// creation site. peers is the write-time peer-id snapshot the author captured
// before its engine write, cloned at enqueue for the same reuse reason; the
// event loop hands it to the core so the promise set is the acceptance-time
// set (A-01). The replay flag marks a notification for already-installed
// state (a snapshot load) rather than a live write; a replay takes the core's
// paced record-send path instead of the immediate broadcast (WR-04).
type localWrite struct {
	key    string
	rec    storage.Record
	peers  []uint16
	replay bool
}

// trackerKey identifies one wait-requesting local write: the key and the
// exact record version the wait observes. The version is part of the
// identity, never a field compared later, so a superseded version can never
// be confused with the write that replaced it.
type trackerKey struct {
	key     string
	version version.RecordVersion
}

// tracker is one write's replication observation state — the adapter-side
// waiter the core's result stream resolves. The mutex guards acked,
// resolved and ok; done is closed exactly once, on the first resolution,
// while the mutex is held, so a waiter waking on it observes the final
// state. The acked set is sorted ascending and idempotent (a re-Ack adds no
// second entry), and it only ever holds peers from the captured write-time
// set: total is that set's length and wants its membership, both stored when
// the tracker is created in NotifyLocalWrite and immutable afterwards, so
// the report can count only the promise the write actually made (A-01,
// T-06-04).
type tracker struct {
	mu sync.Mutex

	// wants is the captured write-time membership set, stored when the
	// tracker is created and immutable afterwards; an Ack from a peer outside
	// it is inert (T-06-04).
	wants map[uint16]struct{}

	total    uint32
	acked    []uint16
	resolved bool
	ok       bool
	done     chan struct{}
}

// newTracker returns a tracker scoped to the captured write-time peers: the
// total is the slice length and wants its membership set, so an Ack from any
// peer outside the capture is inert. The done channel is armed with no
// resolution yet.
func newTracker(peers []uint16) *tracker {
	wants := make(map[uint16]struct{}, len(peers))
	for _, id := range peers {
		wants[id] = struct{}{}
	}
	//nolint:gosec // G115: peers holds 16-bit node ids, so at most 65536 entries — the conversion cannot truncate.
	return &tracker{total: uint32(len(peers)), wants: wants, done: make(chan struct{})}
}

// resolveLocked records the first resolution and closes done; later results
// are ignored, so done is closed exactly once. The caller holds t.mu.
func (t *tracker) resolveLocked(ok bool) {
	if t.resolved {
		return
	}
	t.resolved = true
	t.ok = ok
	close(t.done)
}

// insertPeer returns peers with id inserted at its sorted position, or
// unchanged when already present — the sorted, idempotent Acked discipline
// the core's ackedInsert applies.
func insertPeer(peers []uint16, id uint16) []uint16 {
	pos, found := slices.BinarySearch(peers, id)
	if found {
		return peers
	}
	return slices.Insert(peers, pos, id)
}

// Run is the single event-loop goroutine the caller launches (D-04). It
// starts the read goroutine, emits the startup Hello through the first
// Advance (the core's zero initial hello deadline), then selects over
// context cancellation, inbound datagrams, local-write notifications and
// the timer channel re-armed from Advance's returned deadline. After every
// event it calls the core handler with [time.Now] (the production now
// seam), drains the shared Outbox's message stream through encode+send,
// applies the result stream to the waiter trackers, and mirrors the core's
// counters and replication read states into atomics. Run returns ctx's
// error; closing the context unblocks both the loop and the read goroutine.
func (a *Adapter) Run(ctx context.Context) error {
	go a.readLoop()
	defer func() { _ = a.conn.Close() }() // unblocks the read goroutine's ReadFromUDP.
	defer close(a.done)                   // unblocks the read goroutine's queued send.
	var out port.Outbox
	next := a.core.Advance(time.Now(), &out)
	a.flush(&out)
	a.applyResults(&out)
	var timer *time.Timer
	for {
		timer = armTimer(timer, next)
		select {
		case <-ctx.Done():
			stopTimer(timer)
			return ctx.Err()
		case ev := <-a.inbound:
			in := decode(ev.datagram)
			in.Source = ev.source
			a.core.HandleDatagram(time.Now(), in, &out)
		case lw := <-a.notify:
			// The tracker, when the write requested one, was created in
			// NotifyLocalWrite BEFORE this enqueue. Neither this case nor
			// the core handler creates one: a result for an absent trackers
			// entry is dropped (applyResults), never an insert. A replayed
			// record (a snapshot load) rides the paced record-send queue,
			// never an immediate broadcast (WR-04).
			if lw.replay {
				a.core.NotifyReplayedWrite(time.Now(), lw.key, lw.rec, &out)
			} else {
				a.core.NotifyLocalWrite(time.Now(), lw.key, lw.rec, lw.peers, &out)
			}
		case <-timerChan(timer):
			// A due deadline — the Advance below fires the due timers.
		}
		next = a.core.Advance(time.Now(), &out)
		a.flush(&out)
		a.applyResults(&out)
		a.mirrorStats()
	}
}

// NotifyLocalWrite enqueues the explicit post-write replication call
// (D-05) for the event loop: it defensively copies the record and returns
// without doing any protocol work, so the write path stays available. The
// event loop's core handler broadcasts the DataReplication. When track is
// true the write asked for a wait_for_replication report, and the tracker
// for its exact (key, version) identity is created HERE — synchronously,
// under trackersMu, BEFORE the channel send (XPORT-12): the gRPC handler
// calls [Adapter.WaitReplicated] immediately after this method returns, so
// a tracker created later (say, when the loop dequeues the notification)
// would be missed, and the old appearance race the polling mirror worked
// around could return. This is the sole production creation site — the
// queue struct carries no track flag, so no later code can create a
// tracker from a notification either.
//
// peers is the write-time peer-id snapshot the author captured immediately
// before its engine write (A-01): it is cloned into the queue and handed to
// the core, which seeds the tracked entry's promise set from exactly those
// ids. A nil slice is the replay or untracked shape — no promise set.
//
// The enqueue and the closed-loop decision are one select over the notify
// channel and the loop's done channel, with a re-check after a won send
// (D-15 P4-IN-01 — the former pre-check-then-enqueue pair raced Run's exit
// and could report nil for a notification nobody will ever drain): while
// Run is active the send enqueues into the bounded 1024-deep buffer (D-05)
// and reports nil; once Run has returned, done is closed, so either the
// select's done branch or — when a buffered send still won the uniform
// choice against the already closed done — the post-send re-check reports
// [ErrRunLoopExited]. A failed notify deletes the just-created tracker
// before returning, so the zero-value path (Run already gone, no waiter
// will ever resolve it) cannot leak a map entry. A writer never blocks
// forever on a loop that no longer receives, never panics, and the drop is
// surfaced to the author, never silent.
//
//nolint:revive // flag-parameter: track is the wait_for_replication request crossing the D-05 notification seam (XPORT-12); a second method would split the producer contract.
func (a *Adapter) NotifyLocalWrite(key string, rec storage.Record, track bool, peers []uint16) error {
	keyID := trackerKey{key: key, version: rec.Version}
	if track {
		a.ensureTracker(keyID, peers)
	}
	err := a.enqueueLocalWrite(localWrite{key: key, rec: cloneLocalRecord(rec), peers: slices.Clone(peers)})
	if err != nil {
		// A failed notify deletes the just-created tracker before returning, so
		// the zero-value path (Run already gone, no waiter will ever resolve it)
		// cannot leak a map entry. An absent tracker makes this a no-op.
		a.deleteTracker(keyID)
		return err
	}
	return nil
}

// NotifyReplayedWrite enqueues one snapshot-load (or recovery-replay) record
// for the event loop, where the core puts it on the PACED record-send queue
// instead of broadcasting immediately (WR-04, T-4-20): a large load drains one
// datagram per pacing slot rather than emitting one broadcast per key
// back-to-back. No tracker is created — a load has no wait_for_replication
// report — and the copy rules are NotifyLocalWrite's.
func (a *Adapter) NotifyReplayedWrite(key string, rec storage.Record) error {
	return a.enqueueLocalWrite(localWrite{key: key, rec: cloneLocalRecord(rec), replay: true})
}

// cloneLocalRecord defensively copies one record for the notification
// channel: the author may reuse its buffer once the call returns.
func cloneLocalRecord(rec storage.Record) storage.Record {
	return storage.Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Deleted:  rec.Deleted,
		Checksum: rec.Checksum,
	}
}

// enqueueLocalWrite is the notification channel handoff shared by the live and
// replay seams. The enqueue and the closed-loop decision are one select over
// the notify channel and the loop's done channel, with a re-check after a won
// send (D-15 P4-IN-01 — the former pre-check-then-enqueue pair raced Run's
// exit and could report nil for a notification nobody will ever drain): while
// Run is active the send enqueues into the bounded 1024-deep buffer (D-05) and
// reports nil; once Run has returned, done is closed, so either the select's
// done branch or — when a buffered send still won the uniform choice against
// the already closed done — the post-send re-check reports
// [ErrRunLoopExited]. A writer never blocks forever on a loop that no longer
// receives, never panics, and the drop is surfaced to the author, never
// silent.
func (a *Adapter) enqueueLocalWrite(lw localWrite) error {
	select {
	case a.notify <- lw:
		// The send won, but a send into a buffer nobody drains proves
		// nothing on its own: when the select ran with done already
		// closed, the uniform choice may still have picked the buffered
		// send. The re-check turns that overtaken enqueue into the same
		// deterministic error — never a nil report for a dropped write.
		select {
		case <-a.done:
			return ErrRunLoopExited
		default:
		}
		return nil
	case <-a.done:
		return ErrRunLoopExited
	}
}

// ensureTracker returns the tracker for k, creating it when absent — scoped
// to the captured write-time peers (A-01). Only NotifyLocalWrite calls this
// — the sole production creation site; a tracker already present for the
// same identity is reused (a repeated tracked notification of one write
// shares its resolution and its original promise set).
func (a *Adapter) ensureTracker(k trackerKey, peers []uint16) *tracker {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	if tr := a.trackers[k]; tr != nil {
		return tr
	}
	tr := newTracker(peers)
	a.trackers[k] = tr
	return tr
}

// lookupTracker returns the tracker for k, or nil when the write was not
// tracked. Every caller drops a nil result — no path inserts through a
// lookup.
func (a *Adapter) lookupTracker(k trackerKey) *tracker {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	return a.trackers[k]
}

// deleteTracker removes k's tracker; a no-op when absent. Every
// WaitReplicated return path and a failed notify call it, so the map stays
// bounded by concurrent waits (T-3-16).
func (a *Adapter) deleteTracker(k trackerKey) {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	delete(a.trackers, k)
}

// applyResults consumes the core's result stream after one handler
// (XPORT-12): every result resolves or accumulates on the tracker for its
// exact (key, version) identity. ResultAck inserts the peer into acked only
// when it is a member of the tracker's captured write-time set (A-01,
// T-06-04: an Ack from a node outside the promise is inert), without
// resolving — an Ack alone is not full coverage; ResultDrained resolves
// ok=true (full write-time coverage); ResultSuperseded and
// ResultWaitSetEmpty resolve ok=false (honest false — the tracked version
// lost, or its last waiter was evicted). A result for an absent tracker is
// DROPPED, never a map insert: NotifyLocalWrite is the sole creation site,
// so an absent identity is an untracked write (track=false) or a
// directly-driven core in a white-box test, and neither has a waiter.
// Event-loop goroutine only; each tracker's own mutex is the
// cross-goroutine edge.
func (a *Adapter) applyResults(out *port.Outbox) {
	for _, res := range out.DrainResults() {
		tr := a.lookupTracker(trackerKey{key: res.Key, version: res.Version})
		if tr == nil {
			continue
		}
		tr.mu.Lock()
		switch res.Kind {
		case port.ResultAck:
			if _, promised := tr.wants[res.Peer]; promised {
				tr.acked = insertPeer(tr.acked, res.Peer)
			}
		case port.ResultDrained:
			tr.resolveLocked(true)
		case port.ResultSuperseded, port.ResultWaitSetEmpty:
			tr.resolveLocked(false)
		}
		tr.mu.Unlock()
	}
}

// WaitReplicated waits for one wait-requesting local write's replication
// outcome — the adapter half of the wait_for_replication seam (XPORT-12,
// D-04). total is the write-time peer snapshot the caller captured; the
// tracker created at notify time owns the same set as its total and
// membership, and that captured copy is authoritative here: acked is true
// only on full captured-set coverage, and ackedCount is the number of
// captured peers observed in the acked set — never the raw acked length, so
// an Ack from a node outside the capture can neither increment the count nor
// flip acked (A-01, T-06-04).
//
// The success rule is resolved && ok && count == tr.total: full write-time
// coverage is the only success. Eviction (ResultWaitSetEmpty) and
// supersession (ResultSuperseded) are honest false reports, and a timeout
// or ctx cancellation returns the last observed count with acked=false —
// the wait never reports a success it did not observe.
//
// total == 0 is the D-11 single-node release case: the adapter creates the
// tracker at notify time even for a zero-peer write, so this call is its
// RELEASE point. It reports (false, 0) — no peer was captured, so nothing
// was acknowledged — and deletes the tracker. A tracker whose captured total
// is zero is released the same way even when the caller passes a non-zero
// total, because the captured set is authoritative. An absent tracker
// reports an immediate (false, 0) without polling: two paths reach it — a
// write notified with track=false, and a white-box test driving the core
// directly — and neither has a waiter to resolve.
//
// One wait per write identity is expected (the gRPC handler issues exactly
// one). A second concurrent wait for the same identity shares the same
// tracker's resolution; the first return deletes the map entry, and the
// second reads the already-final tracker state before its own delete is a
// no-op. Every return path deletes the tracker, so the map is bounded by
// concurrent waits (T-3-16).
func (a *Adapter) WaitReplicated(ctx context.Context, key string, ver version.RecordVersion, total uint32, timeout time.Duration) (acked bool, ackedCount uint32) {
	k := trackerKey{key: key, version: ver}
	if total == 0 {
		a.deleteTracker(k)
		return false, 0
	}
	tr := a.lookupTracker(k)
	if tr == nil {
		return false, 0
	}
	if tr.total == 0 {
		// The tracker's captured set is empty: nobody can ever complete this
		// promise, so the wait is the release point regardless of the caller's
		// view (the handler normally passes total 0 for this same reason).
		a.deleteTracker(k)
		return false, 0
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-tr.done:
	case <-ctx.Done():
	case <-timer.C:
	}
	tr.mu.Lock()
	// Count only Acks the tracker's captured set contains: an outside peer's
	// matching Ack is inert by construction (T-06-04), and the count can never
	// exceed the captured total.
	var count uint32
	for _, id := range tr.acked {
		if _, promised := tr.wants[id]; promised {
			count++
		}
	}
	ok := tr.resolved && tr.ok && count == tr.total
	tr.mu.Unlock()
	a.deleteTracker(k)
	return ok, count
}

// newAdapterWithConn returns the bare adapter the receive-path white-box
// tests drive: the socket seam, the log sink and the read loop's two
// channels, with no core and no socket configuration. Production
// construction is [New]; this constructor exists only so a fake [packetConn]
// can prove a non-closed read error is survivable (XPORT-08).
func newAdapterWithConn(conn packetConn) *Adapter {
	return &Adapter{
		conn:    conn,
		log:     func(_ string) {},
		inbound: make(chan inboundEvent, inboundQueue),
		done:    make(chan struct{}),
	}
}

// readLoop is the one read goroutine feeding the event loop's inbound queue
// (D-04): it owns the socket's single 65 507-byte receive buffer for the
// socket's lifetime and defensively copies every read's bytes before
// handing them over, because the buffer is reused on the next read.
//
// XPORT-08's contract is the error branch: ONLY a read error matching
// [net.ErrClosed] exits — the socket closed, which is Run's own exit path.
// Every other error is counted, logged at most once per
// [readErrorLogInterval], paused for [readErrorBackoff] so a persistent
// kernel error cannot spin, and the loop reads the next datagram. A dead
// receive path while Hello broadcasts continue would be a silently deaf
// node; the counter and the log are what make that state visible instead.
func (a *Adapter) readLoop() {
	buf := make([]byte, datagramBufSize)
	for {
		n, src, err := a.conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return // the socket closed — Run's exit path, the only exit.
			}
			a.readErrors.Add(1)
			a.logReadError(err)
			time.Sleep(readErrorBackoff)
			continue
		}
		ev := inboundEvent{datagram: bytes.Clone(buf[:n]), source: src.String()}
		select {
		case a.inbound <- ev:
		case <-a.done:
			return
		}
	}
}

// logReadError surfaces one survived receive error, rate-limited to at most
// one line per [readErrorLogInterval] so an error flood cannot flood the log
// (T-3-36). The wall clock is the sanctioned time source on the adapter
// side; the core's event-time discipline does not apply to this goroutine.
func (a *Adapter) logReadError(err error) {
	a.readErrMu.Lock()
	defer a.readErrMu.Unlock()
	now := time.Now()
	if !a.readErrLogAt.IsZero() && now.Sub(a.readErrLogAt) < readErrorLogInterval {
		return
	}
	a.readErrLogAt = now
	a.log(fmt.Sprintf("udprepl: receive error (read loop continues): %v", err))
}

// flush encodes and sends every outbound message the core emitted for the
// just-handled event: broadcast to the configured broadcast address,
// unicast to the handle resolved through [net.ResolveUDPAddr] — handles are
// opaque to the core; only here do they become addresses (D-03). Each
// encode-successful message counts into its SentByType kind (OBS-01) — the
// datagram is on its way to the socket; an encode failure counts nowhere.
// Failures are logged, never panics: a transient send error must not take
// the node down.
func (a *Adapter) flush(out *port.Outbox) {
	for _, ob := range out.Drain() {
		datagram, err := encodeDatagram(a.nodeID, a.runID, ob.Msg)
		if err != nil {
			a.log(err.Error())
			continue
		}
		a.countSent(ob.Msg)
		if ob.Broadcast {
			a.write(datagram, a.broadcastAddr)
			continue
		}
		addr, err := net.ResolveUDPAddr("udp4", ob.To)
		if err != nil {
			a.log(fmt.Sprintf("udprepl: resolve unicast handle %q: %v", ob.To, err))
			continue
		}
		a.write(datagram, addr)
	}
}

// countSent adds one to the message kind's send counter (OBS-01) — the
// five-kind vocabulary of the wire Body oneof, counted in flush where every
// outbound message crosses the encode step on the event-loop goroutine.
func (a *Adapter) countSent(msg port.Message) {
	switch msg.(type) {
	case port.Hello:
		a.sentHello.Add(1)
	case port.DataReplication:
		a.sentData.Add(1)
	case port.Ack:
		a.sentAck.Add(1)
	case port.DataRequest:
		a.sentDataRequest.Add(1)
	case port.DataSync:
		a.sentSync.Add(1)
	}
}

// write sends one datagram to addr, logging (never panicking on) a failure.
func (a *Adapter) write(datagram []byte, addr *net.UDPAddr) {
	_, err := a.conn.WriteToUDP(datagram, addr)
	if err != nil {
		a.log(fmt.Sprintf("udprepl: send to %s: %v", addr, err))
	}
}

// armTimer re-arms (or first creates) t for the next deadline and returns
// the timer to hold. Go 1.23+ timer semantics make Reset without draining
// safe (the persist engine's timerChan/stopTimer precedent).
func armTimer(t *time.Timer, next time.Time) *time.Timer {
	d := time.Until(next)
	if t == nil {
		return time.NewTimer(d)
	}
	t.Reset(d)
	return t
}

// timerChan returns t's channel, or nil for a nil timer — a nil channel
// receives never, disabling that select case (the persist engine
// precedent).
func timerChan(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

// stopTimer stops t when non-nil. Go 1.23+ timer semantics make Stop safe
// without draining.
func stopTimer(t *time.Timer) {
	if t != nil {
		t.Stop()
	}
}
