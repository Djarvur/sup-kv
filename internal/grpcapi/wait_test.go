package grpcapi_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/version"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

// waitAssertBound is the generous elapsed bound the wait tests assert:
// the waits under test run 100-500ms, so four times the largest bounded
// wait leaves wide scheduling margin under full-suite -race parallel load
// while still proving the response came back from the bounded wait — not
// from the 1s default (or the 500ms appearance bound) running to term.
const waitAssertBound = 400 * time.Millisecond

// fakeMirror is the mutex-guarded fake of the udprepl adapter's replication
// mirrors (05-02's copy-on-write discipline, modeled as a fake): tests
// mutate the pending map on their own goroutines while the server's wait
// loop polls it, so every read takes the lock and returns a fresh deep
// copy — successive immutable snapshots, never a shared mutable map, exactly
// the contract the real mirror pins on its readers.
type fakeMirror struct {
	mu      sync.Mutex
	peers   uint64
	pending map[string]repl.PendingInfo
}

// newFakeMirror returns a mirror with the given write-time peer count and
// an empty pending set.
func newFakeMirror(peers uint64) *fakeMirror {
	return &fakeMirror{peers: peers, pending: make(map[string]repl.PendingInfo)}
}

// KnownPeers mirrors the adapter's peer count.
func (f *fakeMirror) KnownPeers() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.peers
}

// PendingCopy mirrors the adapter's pending snapshot: a fresh map whose
// entries clone every WaitingOn/Acked slice, sharing no backing array with
// anything the fake holds.
func (f *fakeMirror) PendingCopy() map[string]repl.PendingInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]repl.PendingInfo, len(f.pending))
	for key, info := range f.pending {
		out[key] = repl.PendingInfo{
			Version:   info.Version,
			WaitingOn: slices.Clone(info.WaitingOn),
			Acked:     slices.Clone(info.Acked),
			NextRetry: info.NextRetry,
		}
	}
	return out
}

// setPending replaces one pending entry — the test-side stand-in for the
// core's handleAck, evictPeers and supersession transitions.
func (f *fakeMirror) setPending(key string, info repl.PendingInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending[key] = info
}

// newWaitServer builds the open-gate harness with the mirror's closures as
// the wait seams and the given operator wait default (0 = the 1s design
// default NewServer substitutes). A nil mirror wires the nil seams — the
// single-node configuration (D-11).
func newWaitServer(t *testing.T, mirror *fakeMirror, waitTimeout time.Duration) *testHarness {
	t.Helper()
	cfg := testServerConfig{gate: openGateOptions(), waitTimeout: waitTimeout}
	if mirror != nil {
		cfg.knownPeers = mirror.KnownPeers
		cfg.pendingCopy = mirror.PendingCopy
	}
	h := buildTestServer(t, cfg)
	awaitGateOpen(t, h)
	return h
}

// awaitWrittenVersion polls the engine until key carries a record and
// returns its version — the test-side stand-in for the notify-queue lag:
// the tracked version is engine-stamped at write time, so only the stored
// record knows it, and the fake's pending entry must carry exactly that
// version for the wait to track it. It never calls t (it runs on a test
// goroutine); on miss it returns ok=false and the caller's assertions fail
// the test.
func awaitWrittenVersion(h *testHarness, key string) (version.RecordVersion, bool) {
	for range 500 {
		rec, err := h.engine.Get(key)
		if err == nil {
			return rec.Version, true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return version.RecordVersion{}, false
}

// TestWait_zeroPeers pins D-11 on the nil-seam wiring: the single-node
// configuration reports acked=true immediately with total_peers 0, and the
// response arrives without the appearance phase running — the whole round
// trip stays far under the appearance bound.
func TestWait_zeroPeers(t *testing.T) {
	t.Parallel()

	ts := newWaitServer(t, nil, 0)
	ctx := t.Context()

	start := time.Now()
	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
		Key:                "wait/zero",
		Value:              []byte("v"),
		WaitForReplication: true,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Set wait/zero: want OK, got error: %v", err)
	}
	if !resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 0 {
		t.Fatalf("zero-peer report: want acked=true 0/0, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed >= waitAssertBound {
		t.Fatalf("zero-peer Set took %v: want well under %v — the wait must short-circuit, not run the appearance phase", elapsed, waitAssertBound)
	}
}

// TestWait_acksArrive: one known peer, the mirror transitions absent ->
// entry{Acked: [1], WaitingOn: empty} after ~30ms; the response reports
// acked=true 1/1 — the appearance phase waited through the absent phase and
// the drain branch read the fully-resolved wait set.
func TestWait_acksArrive(t *testing.T) {
	t.Parallel()

	const key = "wait/acks"
	mirror := newFakeMirror(1)
	ts := newWaitServer(t, mirror, 0)
	ctx := t.Context()

	go func() {
		ver, ok := awaitWrittenVersion(ts, key)
		if !ok {
			return
		}
		time.Sleep(30 * time.Millisecond)
		mirror.setPending(key, repl.PendingInfo{Version: ver, Acked: []uint16{1}})
	}()

	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
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

// TestWait_timeout: one known peer whose Ack never arrives; the 100ms
// WaitTimeout option bounds the wait to an honest acked=false 0/1 — and the
// Set RPC itself returned OK, proving the wait never changes the write
// outcome.
func TestWait_timeout(t *testing.T) {
	t.Parallel()

	mirror := newFakeMirror(1)
	ts := newWaitServer(t, mirror, 100*time.Millisecond)
	ctx := t.Context()

	start := time.Now()
	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
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
// wait_for_replication under zero known peers reports acked=true 0/0
// exactly like Set, short-circuiting before the appearance phase.
func TestWait_deleteSymmetry(t *testing.T) {
	t.Parallel()

	mirror := newFakeMirror(0)
	ts := newWaitServer(t, mirror, 0)
	ctx := t.Context()

	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "wait/del", Value: []byte("v")})
	if err != nil {
		t.Fatalf("Set wait/del: want OK, got error: %v", err)
	}

	resp, err := ts.client.Delete(ctx, &apiproto.DeleteRequest{
		Key:                "wait/del",
		WaitForReplication: true,
	})
	if err != nil {
		t.Fatalf("Delete wait/del: want OK, got error: %v", err)
	}
	if !resp.GetAcked() || resp.GetAckedCount() != 0 || resp.GetTotalPeers() != 0 {
		t.Fatalf("delete zero-peer report: want acked=true 0/0, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
}

// supersededVersion returns a RecordVersion guaranteed different from any
// engine-stamped one: RFC 9562 versions carry the 0x7 version nibble in
// byte 6 and a variant encoding in byte 8, so the 0xAA fill cannot collide.
func supersededVersion() version.RecordVersion {
	var sup version.RecordVersion
	for i := range sup {
		sup[i] = 0xAA
	}
	return sup
}

// TestWait_appearanceRace locks RESEARCH Pitfall 4: the mirror stays empty
// for the first ~60ms, and only then carries the entry with Acked=[1],
// WaitingOn empty — pre-appearance absence must not be misread as drained.
// The response reports acked=true 1/1 only after the poll waited through
// the empty phase, which the elapsed floor also pins.
func TestWait_appearanceRace(t *testing.T) {
	t.Parallel()

	const key = "wait/race"
	mirror := newFakeMirror(1)
	ts := newWaitServer(t, mirror, 0)
	ctx := t.Context()

	go func() {
		ver, ok := awaitWrittenVersion(ts, key)
		if !ok {
			return
		}
		time.Sleep(60 * time.Millisecond)
		mirror.setPending(key, repl.PendingInfo{Version: ver, Acked: []uint16{1}})
	}()

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
	if !resp.GetAcked() || resp.GetAckedCount() != 1 || resp.GetTotalPeers() != 1 {
		t.Fatalf("appearance-race report: want acked=true 1/1, got acked=%v %d/%d",
			resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	}
	if elapsed < 60*time.Millisecond {
		t.Fatalf("appearance-race Set returned in %v — before the entry could appear: the poll did not wait through the empty phase", elapsed)
	}
}

// TestWait_evictionNotAck locks RESEARCH Pitfall 3 (T-05-04): the mirror
// shows the entry resolved with WaitingOn empty but Acked empty — every
// waiter was evicted by peer_timeout, none Acked. The report is honest
// acked=false 0/1; the plain total-minus-WaitingOn reading would have
// reported a success no peer granted.
func TestWait_evictionNotAck(t *testing.T) {
	t.Parallel()

	const key = "wait/evict"
	mirror := newFakeMirror(1)
	ts := newWaitServer(t, mirror, 0)
	ctx := t.Context()

	go func() {
		ver, ok := awaitWrittenVersion(ts, key)
		if !ok {
			return
		}
		mirror.setPending(key, repl.PendingInfo{Version: ver})
	}()

	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
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

// TestWait_superseded pins the Q2 interpretation: the drain phase watches
// the tracked version's entry — whose prior snapshot showed Acked=[1]
// against 2 known peers — until a greater remote version replaces it. The
// report is acked=false with the counts frozen at the last observation
// (1/2): the tracked version's Acks did not arrive, and the superseding
// entry's Acked set must not leak into the report.
func TestWait_superseded(t *testing.T) {
	t.Parallel()

	const key = "wait/superseded"
	mirror := newFakeMirror(2)
	ts := newWaitServer(t, mirror, 0)
	ctx := t.Context()

	go func() {
		ver, ok := awaitWrittenVersion(ts, key)
		if !ok {
			return
		}
		mirror.setPending(key, repl.PendingInfo{Version: ver, Acked: []uint16{1}, WaitingOn: []uint16{2}})
		time.Sleep(30 * time.Millisecond)
		mirror.setPending(key, repl.PendingInfo{Version: supersededVersion(), WaitingOn: []uint16{2}})
	}()

	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
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

// TestWait_strangerAcksClampedToTotal locks the WR-01 report semantics:
// Acks from node ids outside the write-time peer snapshot join the core's
// acked set (the counting behavior — no wait-set gate), so the raw count
// can exceed the write-time total. The D-08 report reads that count
// clamped to total (D-10 — the report's peer set is the write-time
// snapshot): the timeout report stays honest acked=false with the clamped
// count, and once the one real peer's Ack resolves the wait set the report
// is acked=true 1/1 — not the raw 3/1, and not the inflated-count failure
// the unclamped criterion would produce.
func TestWait_strangerAcksClampedToTotal(t *testing.T) {
	t.Parallel()

	const key = "wait/strangers"
	mirror := newFakeMirror(1)
	ts := newWaitServer(t, mirror, 0)
	ctx := t.Context()

	go func() {
		ver, ok := awaitWrittenVersion(ts, key)
		if !ok {
			return
		}
		// Two strangers (node ids never in the write-time snapshot) Ack
		// the tracked version while the one real waiter has not.
		mirror.setPending(key, repl.PendingInfo{Version: ver, Acked: []uint16{7, 9}, WaitingOn: []uint16{1}})
		time.Sleep(30 * time.Millisecond)
		// The real peer's Ack arrives and empties the wait set: resolved.
		mirror.setPending(key, repl.PendingInfo{Version: ver, Acked: []uint16{1, 7, 9}})
	}()

	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
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
	mirror := newFakeMirror(1)
	ts := newWaitServer(t, mirror, opDefault)
	ctx := t.Context()

	// Both tracked entries stay unresolved (WaitingOn non-empty), so both
	// waits run their drain phase to the resolved deadline.
	go func() {
		for _, key := range []string{keyWrap, keyNeg} {
			if ver, ok := awaitWrittenVersion(ts, key); ok {
				mirror.setPending(key, repl.PendingInfo{Version: ver, WaitingOn: []uint16{1}})
			}
		}
	}()

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
// expose the appearance phase running on the default.
func TestWait_perRequestTimeout(t *testing.T) {
	t.Parallel()

	mirror := newFakeMirror(1)
	ts := newWaitServer(t, mirror, 0) // 0 -> NewServer's 1s default in opts
	ctx := t.Context()

	start := time.Now()
	resp, err := ts.client.Set(ctx, &apiproto.SetRequest{
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
