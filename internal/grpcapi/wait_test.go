package grpcapi_test

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi/apiproto"
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
