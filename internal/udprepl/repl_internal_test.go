package udprepl

// White-box on purpose: the wire encode/decode paths stay unexported — the
// adapter's mechanical halves (D-01/D-02) — so the external tracer test
// reaches them through these exported test-binary-only helpers (the
// dumpfile_internal_test.go precedent for the testpackage exemption).

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// EncodeDatagramForTest encodes msg as a full wire datagram from nodeID's
// runID through the adapter's encode path verbatim.
func EncodeDatagramForTest(msg port.Message, nodeID uint16, runID [16]byte) ([]byte, error) {
	return encodeDatagram(nodeID, runID, msg)
}

// DecodeDatagramForTest decodes a raw datagram through the adapter's decode
// path verbatim, with the sender's source handle attached the way the read
// goroutine attaches it.
func DecodeDatagramForTest(datagram []byte, source string) port.InboundDatagram {
	in := decode(datagram)
	in.Source = source
	return in
}

// EncodeDatagramForTestVersion encodes msg as a full wire datagram with an
// explicit envelope proto_version — the adversarial crafting seam for the
// version-mismatch scenario (plan 06-06): the production encode always
// stamps repl.SupportedProtoVersion, so a mismatched datagram cannot be
// built through it. The body rides the production protoBody mapping and the
// CRC is computed exactly as the production encode computes it, so the
// receiver's CRC gate passes and the core's version check is the branch
// under test.
func EncodeDatagramForTestVersion(msg port.Message, nodeID uint16, runID [16]byte, protoVersion uint32) ([]byte, error) {
	body, err := protoBody(msg)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("udprepl: marshal body: %w", err)
	}
	env := &replproto.Envelope{
		ProtoVersion: protoVersion,
		NodeId:       uint32(nodeID),
		RunId:        runID[:],
		Crc32:        crc32.ChecksumIEEE(bodyBytes),
		Body:         bodyBytes,
	}
	datagram, err := proto.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("udprepl: marshal envelope: %w", err)
	}
	return datagram, nil
}

// The tracker-map tests are white-box on purpose: trackersMu and trackers
// are unexported adapter state, and only a same-package test can observe the
// map directly — including that Run NEVER inserts into it (XPORT-12).

// The two wire identities the tracker tests exchange datagrams between.
const (
	trackerNodeA = uint16(21)
	trackerNodeB = uint16(22)
)

var trackerRunIDB = [16]byte{0x72, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}

// trackerStore adapts *storage.Store to repl.StateStore for these tests.
type trackerStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s trackerStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Version returns the stored version for key, tombstones included.
func (s trackerStore) Version(key string) (version.RecordVersion, bool) { return s.store.Version(key) }

// Lookup returns the stored record for key, tombstones included.
func (s trackerStore) Lookup(key string) (storage.Record, bool) { return s.store.Lookup(key) }

// Versions returns the shallow key→version view, tombstones included.
func (s trackerStore) Versions() map[string]version.RecordVersion { return s.store.Versions() }

// newTrackerTestAdapter returns an adapter bound to a loopback ephemeral
// port, with its socket closed by the test's cleanup. Run is started only by
// the tests that need it.
func newTrackerTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	fixedNow := func() time.Time { return time.UnixMilli(1730000000000) }
	store := storage.NewStore(trackerNodeA, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})
	adapter, err := New("127.0.0.1:0", "127.255.255.255:1", trackerNodeA, trackerStore{store: store}, repl.Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = adapter.conn.Close() })
	return adapter
}

// trackerCount reads the tracker-map length under its lock.
func trackerCount(a *Adapter) int {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	return len(a.trackers)
}

// trackerPresent reports whether k's tracker exists under its lock.
func trackerPresent(a *Adapter, k trackerKey) bool {
	a.trackersMu.Lock()
	defer a.trackersMu.Unlock()
	return a.trackers[k] != nil
}

// awaitCondition polls cond until it holds or the test fails after two
// seconds — the white-box twin of the external waitFor (this package cannot
// import its own external test helpers).
func awaitCondition(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// trackerSender dials the adapter's bound port once, so every datagram a
// test sends shares one source handle — the core's duplicate-node_id
// classifier treats a remembered run_id announcing from a NEW handle as a
// misconfiguration and drops the datagram.
func trackerSender(t *testing.T, adapter *Adapter) net.Conn {
	t.Helper()
	udpPort := adapter.conn.LocalAddr().(*net.UDPAddr).Port
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "udp4", fmt.Sprintf("127.0.0.1:%d", udpPort))
	if err != nil {
		t.Fatalf("dial adapter: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// sendTrackerDatagram encodes msg as nodeID's datagram and sends it over
// the test's persistent sender socket.
func sendTrackerDatagram(t *testing.T, conn net.Conn, msg port.Message, nodeID uint16, runID [16]byte) {
	t.Helper()
	datagram, err := encodeDatagram(nodeID, runID, msg)
	if err != nil {
		t.Fatalf("encode datagram: %v", err)
	}
	_, err = conn.Write(datagram)
	if err != nil {
		t.Fatalf("send datagram: %v", err)
	}
}

// TestNotify_createsTrackerBeforeCore pins the load-bearing ordering
// (XPORT-12): NotifyLocalWrite(track=true) creates the tracker synchronously
// BEFORE the notification is enqueued, so the tracker exists before Run can
// ever dequeue the write — a wait starting immediately after the call always
// finds it. track=false creates none; Run resolves the tracker through the
// result stream; and a tracked notify after Run exited reports
// ErrRunLoopExited and leaves the map empty.
func TestNotify_createsTrackerBeforeCore(t *testing.T) {
	t.Parallel()
	adapter := newTrackerTestAdapter(t)
	key := "tracker/before-core"
	ver := version.PackVersion(1730000000000, 0, trackerNodeA)
	notifyErr := adapter.NotifyLocalWrite(key, storage.Record{Version: ver, Data: []byte("v")}, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(track=true): %v", notifyErr)
	}
	// Run has not started: the tracker must already exist.
	if !trackerPresent(adapter, trackerKey{key: key, version: ver}) {
		t.Fatal("no tracker after NotifyLocalWrite(track=true) returned, want one created before the enqueue")
	}
	if got := trackerCount(adapter); got != 1 {
		t.Fatalf("tracker map holds %d entries after one tracked notify, want 1", got)
	}
	// track=false creates no tracker.
	untrackedKey := "tracker/untracked"
	untrackedVer := version.PackVersion(1730000000000, 1, trackerNodeA)
	notifyErr = adapter.NotifyLocalWrite(untrackedKey, storage.Record{Version: untrackedVer, Data: []byte("v")}, false)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(track=false): %v", notifyErr)
	}
	if trackerPresent(adapter, trackerKey{key: untrackedKey, version: untrackedVer}) {
		t.Fatal("track=false created a tracker, want none")
	}
	if got := trackerCount(adapter); got != 1 {
		t.Fatalf("tracker map holds %d entries after a track=false notify, want still 1", got)
	}

	ctx, cancel := context.WithCancel(t.Context())
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = adapter.Run(ctx)
	}()
	awaitCondition(t, "both queued writes to become tracked pending entries", func() bool {
		return adapter.UnackedStats().Count == 2
	})

	// Discovery makes B a waiter on the tracked write, then B's Ack resolves
	// the tracker through the core's result stream. One sender socket keeps
	// B's handle stable across both datagrams.
	sender := trackerSender(t, adapter)
	sendTrackerDatagram(t, sender, port.Hello{}, trackerNodeB, trackerRunIDB)
	awaitCondition(t, "discovery to mirror KnownPeers == 1", func() bool {
		return adapter.KnownPeers() == 1
	})

	reports := make(chan replicationResult, 1)
	go func() {
		acked, count := adapter.WaitReplicated(t.Context(), key, ver, 1, 2*time.Second)
		reports <- replicationResult{acked: acked, count: count}
	}()
	sendTrackerDatagram(t, sender, port.Ack{Key: key, Version: ver}, trackerNodeB, trackerRunIDB)
	select {
	case report := <-reports:
		if !report.acked || report.count != 1 {
			t.Fatalf("wait report = (%v, %d), want full coverage (true, 1)", report.acked, report.count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitReplicated did not return through the result stream")
	}
	if got := trackerCount(adapter); got != 0 {
		t.Fatalf("tracker map holds %d entries after the wait returned, want 0", got)
	}

	// A tracked notify after Run exited fails loudly and releases its
	// just-created tracker.
	cancel()
	<-runDone
	postVer := version.PackVersion(1730000000000, 2, trackerNodeA)
	err := adapter.NotifyLocalWrite("tracker/after-run", storage.Record{Version: postVer, Data: []byte("v")}, true)
	if !errors.Is(err, ErrRunLoopExited) {
		t.Fatalf("post-Run tracked notify = %v, want ErrRunLoopExited", err)
	}
	if got := trackerCount(adapter); got != 0 {
		t.Fatalf("a failed notify left %d trackers, want the map empty", got)
	}
}

// replicationResult is one WaitReplicated outcome.
type replicationResult struct {
	acked bool
	count uint32
}

// TestWaitReplicated_zeroPeersReleases pins the D-11 release point: a
// zero-peer wait reports (true, 0) immediately and deletes the tracker the
// write created at notify time — the zero-peer path never enters the
// waiting loop, so this call is its only release.
func TestWaitReplicated_zeroPeersReleases(t *testing.T) {
	t.Parallel()
	adapter := newTrackerTestAdapter(t)
	key := "tracker/zero-peers"
	ver := version.PackVersion(1730000000000, 0, trackerNodeA)
	notifyErr := adapter.NotifyLocalWrite(key, storage.Record{Version: ver, Data: []byte("v")}, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(track=true): %v", notifyErr)
	}
	acked, count := adapter.WaitReplicated(t.Context(), key, ver, 0, 0)
	if !acked || count != 0 {
		t.Fatalf("zero-peer wait report = (%v, %d), want (true, 0)", acked, count)
	}
	if got := trackerCount(adapter); got != 0 {
		t.Fatalf("zero-peer wait left %d trackers, want the map empty", got)
	}
}

// TestWaitReplicated_timeoutReleases pins the timeout abandonment path: an
// unresolved wait returns (false, 0) after the timeout and deletes its
// tracker, so the map is bounded by concurrent waits (T-3-16).
func TestWaitReplicated_timeoutReleases(t *testing.T) {
	t.Parallel()
	adapter := newTrackerTestAdapter(t)
	key := "tracker/timeout"
	ver := version.PackVersion(1730000000000, 0, trackerNodeA)
	notifyErr := adapter.NotifyLocalWrite(key, storage.Record{Version: ver, Data: []byte("v")}, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(track=true): %v", notifyErr)
	}
	acked, count := adapter.WaitReplicated(t.Context(), key, ver, 1, 20*time.Millisecond)
	if acked || count != 0 {
		t.Fatalf("timed-out wait report = (%v, %d), want the honest false (false, 0)", acked, count)
	}
	if got := trackerCount(adapter); got != 0 {
		t.Fatalf("timed-out wait left %d trackers, want the map empty", got)
	}
}

// TestWaitReplicated_ctxCancelReleases pins the cancellation abandonment
// path: a wait whose context is already canceled returns (false, 0) without
// polling and deletes its tracker.
func TestWaitReplicated_ctxCancelReleases(t *testing.T) {
	t.Parallel()
	adapter := newTrackerTestAdapter(t)
	key := "tracker/cancel"
	ver := version.PackVersion(1730000000000, 0, trackerNodeA)
	notifyErr := adapter.NotifyLocalWrite(key, storage.Record{Version: ver, Data: []byte("v")}, true)
	if notifyErr != nil {
		t.Fatalf("NotifyLocalWrite(track=true): %v", notifyErr)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	acked, count := adapter.WaitReplicated(ctx, key, ver, 1, time.Second)
	if acked || count != 0 {
		t.Fatalf("canceled wait report = (%v, %d), want the honest false (false, 0)", acked, count)
	}
	if got := trackerCount(adapter); got != 0 {
		t.Fatalf("canceled wait left %d trackers, want the map empty", got)
	}
}
