package grpcapi_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// waitAssertBound is the generous elapsed bound the wait tests assert:
// the waits under test run 30-500ms, so four times the largest bounded
// wait leaves wide scheduling margin under full-suite -race parallel load
// while still proving the response came back from the resolved wait — not
// from the 1s operator default running to term.
const (
	waitAssertBound = 400 * time.Millisecond

	// scriptedWaitDelay is the delay before a scripted wait resolves: long
	// enough that the handler is demonstrably blocked in the seam.
	scriptedWaitDelay = 30 * time.Millisecond
)

// waitCall is one WaitReplicated invocation the fake recorded.
type waitCall struct {
	key     string
	ver     version.RecordVersion
	total   uint32
	timeout time.Duration
}

// fakeWait is one scripted wait: the test resolves it with a report, or
// leaves it unresolved so the fake's own timeout/ctx path returns the
// honest false — the same contract the real adapter tracker honors.
type fakeWait struct {
	once  sync.Once
	done  chan struct{}
	acked bool
	count uint32
}

// newFakeWait returns an unresolved scripted wait.
func newFakeWait() *fakeWait {
	return &fakeWait{done: make(chan struct{})}
}

// resolve closes the wait with its report exactly once.
func (w *fakeWait) resolve(acked bool, count uint32) {
	w.once.Do(func() {
		w.acked = acked
		w.count = count
		close(w.done)
	})
}

// fakeTracker is the scripted stand-in for the adapter's tracker seam
// (XPORT-12): it records every call, answers the total==0 release call
// immediately, and otherwise waits on the scripted wait for the key, its own
// timeout or ctx cancellation. The real tracker's resolution rules are
// exercised by the udprepl tests; this fake pins the gRPC projection of the
// seam. peers is the write-time peer-id snapshot the fake publishes as the
// ServerOptions.WritePeers seam — n consecutive ids from 1, so a test's peer
// count and the per-id slice stay one value.
type fakeTracker struct {
	mu    sync.Mutex
	peers []uint16
	calls []waitCall
	waits map[string]*fakeWait
}

// newFakeTracker returns a tracker fake whose write-time peer snapshot holds
// n consecutive ids (1..n) and no scripted waits.
func newFakeTracker(n int) *fakeTracker {
	peers := make([]uint16, n)
	for i := range peers {
		peers[i] = uint16(i + 1)
	}
	return &fakeTracker{peers: peers, waits: make(map[string]*fakeWait)}
}

// WritePeers is the write-time peer snapshot seam (A-01).
func (f *fakeTracker) WritePeers() []uint16 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.peers)
}

// setWait scripts the wait outcome for key.
func (f *fakeTracker) setWait(key string, wait *fakeWait) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.waits[key] = wait
}

// waitCalls returns a copy of the recorded calls.
func (f *fakeTracker) waitCalls() []waitCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]waitCall(nil), f.calls...)
}

// WaitReplicated is the tracker seam: it records the call and honors the
// real adapter's contract — total==0 succeeds immediately (the release
// call), an absent scripted wait reports the honest false without waiting,
// and a scripted wait blocks until resolution, ctx cancellation or timeout.
func (f *fakeTracker) WaitReplicated(ctx context.Context, key string, ver version.RecordVersion, total uint32, timeout time.Duration) (bool, uint32) {
	f.mu.Lock()
	f.calls = append(f.calls, waitCall{key: key, ver: ver, total: total, timeout: timeout})
	if total == 0 {
		f.mu.Unlock()
		return true, 0
	}
	wait := f.waits[key]
	f.mu.Unlock()
	if wait == nil {
		return false, 0
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-wait.done:
		return wait.acked, wait.count
	case <-ctx.Done():
		return false, 0
	case <-timer.C:
		return false, 0
	}
}

// newWaitServer builds the open-gate harness with the tracker's closures as
// the wait seams and the given operator wait default (0 = the 1s design
// default NewServer substitutes).
func newWaitServer(t *testing.T, tracker *fakeTracker, waitTimeout time.Duration) *testHarness {
	t.Helper()
	cfg := testServerConfig{gate: openGateOptions(), waitTimeout: waitTimeout}
	cfg.writePeers = tracker.WritePeers
	cfg.waitReplicated = tracker.WaitReplicated
	h := buildTestServer(t, cfg)
	awaitGateOpen(t, h)
	return h
}

// scriptWait registers one scripted wait and resolves it after the scripted
// delay (long enough that the handler is blocked in the seam when it fires).
func scriptWait(t *testing.T, tracker *fakeTracker, key string, acked bool, count uint32) {
	t.Helper()
	wait := newFakeWait()
	tracker.setWait(key, wait)
	go func() {
		time.Sleep(scriptedWaitDelay)
		wait.resolve(acked, count)
	}()
}

// TestWait_zeroPeers pins A-01 (was D-11's overstatement): a write with zero
// captured peers is NOT acknowledged by anyone, so the report is the honest
// (false, 0, 0) immediately — and the tracker seam still receives the total==0
// release call, the only release point for the tracker the adapter created at
// notify time.
func TestWait_zeroPeers(t *testing.T) {
	t.Parallel()

	const key = "wait/zero"
	tracker := newFakeTracker(0)
	ts := newWaitServer(t, tracker, 0)
	ctx := t.Context()

	start := time.Now()
	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
		Key:                key,
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 0 {
		t.Fatalf("zero-peer report: want acked=false 0/0, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= waitAssertBound {
		t.Fatalf("zero-peer Set took %v: want well under %v — the wait must short-circuit", elapsed, waitAssertBound)
	}
	calls := tracker.waitCalls()
	if len(calls) != 1 || calls[0].total != 0 || calls[0].key != key {
		t.Fatalf("tracker calls = %+v, want exactly one total==0 release call for %q", calls, key)
	}
}

// TestWait_acksArrive: one known peer; the scripted tracker resolves as
// full coverage after ~30ms and the response reports acked=true 1/1.
func TestWait_acksArrive(t *testing.T) {
	t.Parallel()

	const key = "wait/acks"
	tracker := newFakeTracker(1)
	ts := newWaitServer(t, tracker, 0)
	scriptWait(t, tracker, key, true, 1)

	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                key,
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}
	if !resp.GetAcked() || resp.GetAckedCount() != 1 || resp.GetTotalPeers() != 1 {
		t.Fatalf("acks report: want acked=true 1/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
}

// TestWait_timeout: one known peer whose wait never resolves; the 100ms
// WaitTimeout option bounds the wait to an honest acked=false 0/1 — and the
// Set RPC itself returned OK, proving the wait never changes the write
// outcome.
func TestWait_timeout(t *testing.T) {
	t.Parallel()

	tracker := newFakeTracker(1)
	ts := newWaitServer(t, tracker, 100*time.Millisecond)

	start := time.Now()
	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                "wait/timeout",
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set wait/timeout: want OK (the wait never changes the outcome), got error: %v", err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 1 {
		t.Fatalf("timeout report: want acked=false 0/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= waitAssertBound {
		t.Fatalf("timed-out Set took %v: want under %v — the 100ms option must bound the wait", elapsed, waitAssertBound)
	}
}

// TestWait_deleteSymmetry: a delete is a write (API-05) — Delete with
// wait_for_replication under zero captured peers reports the honest
// acked=false 0/0 exactly like Set and sends the tracker the total==0 release
// call.
func TestWait_deleteSymmetry(t *testing.T) {
	t.Parallel()

	const key = "wait/del"
	tracker := newFakeTracker(0)
	ts := newWaitServer(t, tracker, 0)
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}

	resp, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{
		Key:                key,
		WaitForReplication: true,
	})
	if err != nil {
		t.Fatalf("Delete %s: want OK, got error: %v", key, err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 0 {
		t.Fatalf("delete zero-peer report: want acked=false 0/0, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	calls := tracker.waitCalls()
	if len(calls) != 1 || calls[0].total != 0 || calls[0].key != key {
		t.Fatalf("tracker calls = %+v, want exactly one total==0 release call for %q", calls, key)
	}
}

// TestWait_evictionNotAck locks RESEARCH Pitfall 3 (T-05-04): the tracked
// write resolves with no Acks — every waiter was evicted by peer_timeout.
// The tracker reports the honest false 0/1; the plain total-minus-waiting
// reading would have reported a success no peer granted.
func TestWait_evictionNotAck(t *testing.T) {
	t.Parallel()

	const key = "wait/evict"
	tracker := newFakeTracker(1)
	ts := newWaitServer(t, tracker, 0)
	scriptWait(t, tracker, key, false, 0)

	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                key,
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 1 {
		t.Fatalf("eviction report: want acked=false 0/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
}

// TestWait_superseded pins the Q2 interpretation: the tracked version is
// displaced by a greater remote version after one of two peers Acked. The
// report is acked=false with the counts frozen at the last observation
// (1/2): the tracked version's Acks did not arrive, and the superseding
// entry's coverage must not leak into the report.
func TestWait_superseded(t *testing.T) {
	t.Parallel()

	const key = "wait/superseded"
	tracker := newFakeTracker(2)
	ts := newWaitServer(t, tracker, 0)
	scriptWait(t, tracker, key, false, 1)

	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                key,
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 1 || resp.GetTotalPeers() != 2 {
		t.Fatalf("superseded report: want acked=false 1/2 frozen at the last observation, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
}

// TestWait_strangerAcksClampedToTotal locks the WR-01 report semantics: the
// tracker may observe more Acks than the write-time peer snapshot (a
// matching Ack from a node outside it joins the core's acked set), so the
// report clamps the count to total: the scripted (true, 3) against one peer
// reports acked=true 1/1.
func TestWait_strangerAcksClampedToTotal(t *testing.T) {
	t.Parallel()

	const key = "wait/strangers"
	tracker := newFakeTracker(1)
	ts := newWaitServer(t, tracker, 0)
	scriptWait(t, tracker, key, true, 3)

	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                key,
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}
	if !resp.GetAcked() || resp.GetAckedCount() != 1 || resp.GetTotalPeers() != 1 {
		t.Fatalf("stranger-ack report: want acked=true 1/1 with the count clamped to the write-time total, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
}

// overflowWrapMs is a wait_timeout_ms above the ms→ns conversion-safe
// ceiling whose raw int64 product wraps to a known huge POSITIVE duration:
// 18 446 744 173 910 ms × 1 000 000 ns/ms = 18 446 744 173 910 000 000 ns,
// which reduced mod 2^64 reads back as +100 200 448 384 ns — a 100.2s
// server-side wait from one hostile field value (WR-02). The negative-wrap
// sibling is conversionCeiling+1 ms: its product wraps to −292y, a deadline
// already in the past.
const (
	overflowWrapMs = int64(18446744173910)
	negativeWrapMs = int64(9223372036855)
)

// TestWait_overflowingTimeoutClamped locks the WR-02 bounds on the
// unvalidated remote wait_timeout_ms: a value whose ms→ns conversion wraps
// to a huge positive duration cannot hold the handler for ~100s (the 3s
// client deadline would sever the RPC first — the pre-fix failure), and a
// value wrapping negative cannot dodge the wait with an already-expired
// deadline (the pre-fix call returns immediately instead of running the
// bounded wait). The resolved deadline clamps to ten times the operator
// default (50ms here → 500ms), so both hostile values behave like an
// ordinary bounded timeout: honest acked=false 0/1, the wait never changes
// the write outcome.
func TestWait_overflowingTimeoutClamped(t *testing.T) {
	t.Parallel()

	const (
		keyWrap   = "wait/wrap-pos"
		keyNeg    = "wait/wrap-neg"
		opDefault = 50 * time.Millisecond
	)
	tracker := newFakeTracker(1)
	ts := newWaitServer(t, tracker, opDefault)
	// Both tracked waits stay unresolved, so both waits run to their
	// resolved deadline.
	tracker.setWait(keyWrap, newFakeWait())
	tracker.setWait(keyNeg, newFakeWait())
	ctx := t.Context()

	start := time.Now()
	wrapCtx, cancelWrap := context.WithTimeout(ctx, 3*time.Second)
	resp, err := ts.client.Set(wrapCtx, &apiproto.SetRequest{
		Key:                keyWrap,
		Value:              []byte("v"),
		WaitForReplication: true,
		WaitTimeoutMs:      overflowWrapMs,
	})
	cancelWrap()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set %s: want OK within the client deadline (the ceiling caps the hostile value), got error: %v", keyWrap, err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 1 {
		t.Fatalf("overflow-wrap report: want acked=false 0/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("overflow-wrap Set took %v: want under 2s — the positive wrap must clamp to the 10x ceiling (500ms)", elapsed)
	}

	start = time.Now()
	negCtx, cancelNeg := context.WithTimeout(ctx, 3*time.Second)
	resp, err = ts.client.Set(negCtx, &apiproto.SetRequest{
		Key:                keyNeg,
		Value:              []byte("v"),
		WaitForReplication: true,
		WaitTimeoutMs:      negativeWrapMs,
	})
	cancelNeg()
	elapsed = time.Since(start)
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", keyNeg, err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 1 {
		t.Fatalf("negative-wrap report: want acked=false 0/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed < 450*time.Millisecond {
		t.Fatalf("negative-wrap Set returned in %v: want the bounded ~500ms wait — the negative wrap must clamp, not yield an already-expired deadline", elapsed)
	}
}

// TestWait_perRequestTimeout pins D-09: the per-request wait_timeout_ms
// (120ms) wins over the 1s operator default in the options — the call
// returns in roughly 120ms, and an elapsed at or beyond the bound would
// expose the default running to term.
func TestWait_perRequestTimeout(t *testing.T) {
	t.Parallel()

	tracker := newFakeTracker(1)
	ts := newWaitServer(t, tracker, 0) // 0 -> NewServer's 1s default in opts

	start := time.Now()
	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                "wait/perreq",
		Value:              []byte("v"),
		WaitForReplication: true,
		WaitTimeoutMs:      120,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set wait/perreq: want OK, got error: %v", err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 1 {
		t.Fatalf("per-request timeout report: want acked=false 0/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= waitAssertBound {
		t.Fatalf("per-request-timeout Set took %v: want under %v — the 120ms request field must win over the 1s default", elapsed, waitAssertBound)
	}
}

// TestWait_fullCoverageOnDrained pins the success rule end to end: full
// write-time coverage (both peers) reports acked=true 2/2.
func TestWait_fullCoverageOnDrained(t *testing.T) {
	t.Parallel()

	const key = "wait/full"
	tracker := newFakeTracker(2)
	ts := newWaitServer(t, tracker, 0)
	scriptWait(t, tracker, key, true, 2)

	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                key,
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	if err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, err)
	}
	if !resp.GetAcked() || resp.GetAckedCount() != 2 || resp.GetTotalPeers() != 2 {
		t.Fatalf("full-coverage report: want acked=true 2/2, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
}

// TestWait_noTrackerReportsFalse replaces the old appearance-race test. The
// race is structurally removed (the adapter creates the tracker before the
// notification is enqueued, pinned by
// internal/udprepl.TestNotify_createsTrackerBeforeCore), so the honest pin
// here is that an absent tracker reports false immediately — no polling, no
// appearance phase, no 500ms wait for a phantom entry.
func TestWait_noTrackerReportsFalse(t *testing.T) {
	t.Parallel()

	tracker := newFakeTracker(1) // one peer, but no tracker for the write
	ts := newWaitServer(t, tracker, 0)

	start := time.Now()
	resp, err := ts.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                "wait/absent",
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set wait/absent: want OK, got error: %v", err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 1 {
		t.Fatalf("no-tracker report: want acked=false 0/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("no-tracker Set took %v: want an immediate false — no polling and no appearance phase remains", elapsed)
	}
}

// TestWait_trackerNilIsSingleNode pins the nil-seam contract (A-01): a nil
// WaitReplicated is the single-node wiring even when the peer seam reports
// peers — no tracker can exist through this seam, so nothing was awaited and
// nothing was acknowledged: the report is (false, 0, 0) immediately.
func TestWait_trackerNilIsSingleNode(t *testing.T) {
	t.Parallel()

	cfg := testServerConfig{gate: openGateOptions()}
	cfg.writePeers = func() []uint16 { return []uint16{1, 2, 3, 4, 5} }
	h := buildTestServer(t, cfg)
	awaitGateOpen(t, h)

	start := time.Now()
	resp, err := h.client.Set(t.Context(), &apiproto.SetRequest{
		Key:                "wait/nil-seam",
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set wait/nil-seam: want OK, got error: %v", err)
	}
	if resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 0 {
		t.Fatalf("nil-seam report: want acked=false 0/0, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= waitAssertBound {
		t.Fatalf("nil-seam Set took %v: want the immediate single-node answer", elapsed)
	}
}

// mutablePeers is a WritePeers seam whose result the test can replace after
// the handler captured it — the moving-target probe for A-01.
type mutablePeers struct {
	mu    sync.Mutex
	peers []uint16
	calls int
}

// WritePeers returns the current ids and counts the call.
func (m *mutablePeers) WritePeers() []uint16 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return m.peers
}

// set replaces the seam's result — a peer discovered (or lost) after the
// write was accepted.
func (m *mutablePeers) set(ids ...uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers = ids
}

// callCount returns how many times the handler consulted the seam.
func (m *mutablePeers) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// TestWait_writeTimeSetPinned pins A-01's capture point end to end: the peer
// set is read exactly once, immediately before the engine write, and that one
// slice feeds the replication notification and the wait report. Replacing the
// seam's answer while the handler is blocked in the wait cannot move the
// reported total, and a peer discovered after acceptance never joins it.
func TestWait_writeTimeSetPinned(t *testing.T) {
	t.Parallel()

	const key = "wait/pinned"
	peers := &mutablePeers{peers: []uint16{1, 2}}
	tracker := newFakeTracker(0)
	capture := &notifyCapture{}
	cfg := testServerConfig{
		gate:           openGateOptions(),
		notify:         capture.notify,
		writePeers:     peers.WritePeers,
		waitReplicated: tracker.WaitReplicated,
	}
	h := buildTestServer(t, cfg)
	awaitGateOpen(t, h)

	wait := newFakeWait()
	tracker.setWait(key, wait)

	type result struct {
		resp *apiproto.SetResponse
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := h.client.Set(t.Context(), &apiproto.SetRequest{
			Key:                key,
			Value:              []byte("v"),
			WaitForReplication: true,
		})
		done <- result{resp: resp, err: err}
	}()

	// Wait until the handler captured the set and enqueued the notification
	// (the handler is now blocked in the scripted wait).
	deadline := time.Now().Add(2 * time.Second)
	for len(capture.snapshot()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the Set handler never notified the replication seam")
		}
		time.Sleep(time.Millisecond)
	}

	// The seam now reports a DIFFERENT set: the captured one must stand.
	peers.set(3, 4, 5)
	wait.resolve(true, 2)

	res := <-done
	if res.err != nil {
		t.Fatalf("Set %s: want OK, got error: %v", key, res.err)
	}
	if !res.resp.GetAcked() || res.resp.GetAckedCount() != 2 || res.resp.GetTotalPeers() != 2 {
		t.Fatalf("write-time report: want acked=true 2/2, got acked=%v %d/%d",
			res.resp.GetAcked(), res.resp.GetAckedCount(), res.resp.GetTotalPeers())
	}
	if got := peers.callCount(); got != 1 {
		t.Fatalf("WritePeers consulted %d times for one write, want exactly 1 — the set is captured once", got)
	}
	calls := capture.snapshot()
	if len(calls) != 1 {
		t.Fatalf("notify calls = %d, want exactly 1", len(calls))
	}
	if !slices.Equal(calls[0].peers, []uint16{1, 2}) {
		t.Fatalf("notified peers = %v, want the write-time set [1 2]", calls[0].peers)
	}
	if !calls[0].track {
		t.Fatal("notify call carried track=false, want the request's wait_for_replication flag")
	}
	waits := tracker.waitCalls()
	if len(waits) != 1 || waits[0].total != 2 {
		t.Fatalf("wait calls = %+v, want exactly one with the write-time total 2", waits)
	}
}
