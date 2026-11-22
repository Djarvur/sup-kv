package repl

// White-box tests for the bounded, deduplicated egress queue (XPORT-07,
// RESEARCH Pattern 9). They live in package repl because the queue's
// identity (egressKey/egressKind), its cap (maxEgressEntries) and its
// producer (enqueueResponse) are unexported, and because the gone-record
// drain case needs a store that can forget a key outright — storage.Store
// keeps tombstones, so a lookup would still succeed. The _internal_test.go
// suffix is the documented testpackage exemption.

import (
	"bytes"
	"fmt"
	"maps"
	"runtime"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The internal-test fixtures: a fixed virtual base and the local node id,
// kept local because package repl cannot import the external repl_test
// package's constants.
const (
	egressTestNowMs = int64(1730000000000)
	egressTestLocal = uint16(1)
	egressTestPeer  = uint16(2)
)

// egressStore is a minimal StateStore whose record set the test mutates
// directly: the drain-skip case needs a key that is truly gone (not a
// tombstone), which storage.Store cannot express.
type egressStore struct {
	versions map[string]version.RecordVersion
	records  map[string]storage.Record
}

// ApplyRemote joins one record into the fake store.
func (s *egressStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	s.versions[key] = rec.Version
	s.records[key] = rec
	return nil, 0, nil
}

// Version returns the fake store's version for key.
func (s *egressStore) Version(key string) (version.RecordVersion, bool) {
	v, ok := s.versions[key]
	return v, ok
}

// Lookup returns a caller-owned copy of the fake store's record for key —
// the production storage.Store contract, whose clone is load-bearing for
// TestEgress_noValueRetention: an enqueue path that looks a record up
// allocates the full value there, exactly as the real store's clone would.
func (s *egressStore) Lookup(key string) (storage.Record, bool) {
	rec, ok := s.records[key]
	if !ok {
		return storage.Record{}, false
	}
	rec.Data = bytes.Clone(rec.Data)
	return rec, true
}

// Versions returns a fresh copy of the fake store's version view.
func (s *egressStore) Versions() map[string]version.RecordVersion {
	out := make(map[string]version.RecordVersion, len(s.versions))
	maps.Copy(out, s.versions)
	return out
}

// Observe is a clock-less no-op: this fake holds no clock.
func (*egressStore) Observe(_ version.RecordVersion) time.Duration { return 0 }

// Generation returns a fixed 0: this fake holds its own map and nothing
// caches a listing build over it.
func (*egressStore) Generation() uint64 { return 0 }

// newEgressCore returns a core over the mutable fake store.
func newEgressCore() *Core {
	return New(egressTestLocal, [16]byte{1, 2, 3}, &egressStore{
		versions: make(map[string]version.RecordVersion),
		records:  make(map[string]storage.Record),
	}, Config{})
}

// egressNow returns the fixed virtual test instant.
func egressNow() time.Time {
	return time.UnixMilli(egressTestNowMs)
}

// missingEntries builds n distinct listing entries for keys the fake store
// does not hold.
func missingEntries(n int) []port.SyncEntry {
	entries := make([]port.SyncEntry, 0, n)
	for i := range n {
		entries = append(entries, port.SyncEntry{
			Key:     fmt.Sprintf("missing/%06d", i),
			Version: version.PackVersion(egressTestNowMs, uint16(i), egressTestPeer),
		})
	}
	return entries
}

// egressSync feeds one complete one-part listing, from source under syncID,
// through the DataSync handler — the white-box shape the egress tests drive.
func egressSync(core *Core, now time.Time, source string, syncID uint64, entries []port.SyncEntry) {
	msg := port.DataSync{SyncID: syncID, ChunkIndex: 1, ChunkCount: 1, Entries: entries}
	in := port.InboundDatagram{
		Source:       source,
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       egressTestPeer,
		RunID:        [16]byte{1},
		Msg:          msg,
	}
	var out port.Outbox
	core.handleDataSync(now, in, msg, &out)
}

// TestEgress_boundedDeduped proves the XPORT-07 acceptance: two DataSync
// listings carrying the same 1 000 missing keys leave exactly 1 000
// DataRequest answers queued, not 2 000 — the identity is
// (handle, kind, key) and the second listing's answers are all duplicates.
func TestEgress_boundedDeduped(t *testing.T) {
	t.Parallel()
	core := newEgressCore()
	now := egressNow()
	const missing = 1000
	entries := missingEntries(missing)

	egressSync(core, now, "peer", 1, entries)
	egressSync(core, now, "peer", 2, entries)
	if got := core.QueuedResponses(); got != missing {
		t.Fatalf("two listings of %d missing keys queued %d answers, want %d — the dedupe identity collapsed the duplicates", missing, got, missing)
	}
	for _, q := range core.egress {
		if q.id.kind != egressDataRequest || q.id.handle != "peer" {
			t.Fatalf("queued identity = %+v, want a DataRequest toward the listing sender", q.id)
		}
	}
}

// TestEgress_identitySeparates proves the identity includes both the
// destination handle and the message kind: the same key for two handles, and
// the same (handle, key) as a DataReplication reply, are three distinct
// entries.
func TestEgress_identitySeparates(t *testing.T) {
	t.Parallel()
	core := newEgressCore()
	now := egressNow()
	core.enqueueResponse(now, "peer-a", port.DataRequest{Key: "cfg/k"})
	core.enqueueResponse(now, "peer-a", port.DataRequest{Key: "cfg/k"})
	core.enqueueResponse(now, "peer-b", port.DataRequest{Key: "cfg/k"})
	rec := storage.Record{Version: version.PackVersion(egressTestNowMs, 0, egressTestPeer), Data: []byte("v")}
	core.enqueueResponse(now, "peer-a", port.DataReplication{Key: "cfg/k", Record: rec})

	if got := core.QueuedResponses(); got != 3 {
		t.Fatalf("QueuedResponses() = %d, want 3 (one duplicate collapsed, two handles and two kinds kept apart)", got)
	}
}

// TestEgress_capDropsOldest proves the cap's drop-oldest policy and its
// lockstep dedupe-map deletion (T-3-21, RESEARCH Pitfall 10): cap+1 enqueues
// drop exactly one entry, the oldest is gone, and — the load-bearing half —
// the evicted identity re-enqueues successfully instead of being treated as
// already queued.
func TestEgress_capDropsOldest(t *testing.T) {
	t.Parallel()
	core := newEgressCore()
	now := egressNow()
	evicted := "cap/000000"
	core.enqueueResponse(now, "peer", port.DataRequest{Key: evicted})
	for i := 1; i <= maxEgressEntries; i++ {
		core.enqueueResponse(now, "peer", port.DataRequest{Key: fmt.Sprintf("cap/%06d", i)})
	}

	if got := core.QueuedResponses(); got != maxEgressEntries {
		t.Fatalf("QueuedResponses() = %d after cap+1 enqueues, want the cap %d", got, maxEgressEntries)
	}
	if got := core.EgressDropped(); got != 1 {
		t.Fatalf("EgressDropped() = %d after one overflow, want 1", got)
	}
	if head := core.egress[0].id.key; head != "cap/000001" {
		t.Fatalf("the queue head after the overflow = %q, want the new oldest cap/000001", head)
	}
	if _, queued := core.egressIDs[egressKey{handle: "peer", kind: egressDataRequest, key: evicted}]; queued {
		t.Fatal("the evicted identity is still in the dedupe map — a later identical request would be silently starved")
	}

	core.enqueueResponse(now, "peer", port.DataRequest{Key: evicted})
	if got := core.EgressDropped(); got != 2 {
		t.Fatalf("re-enqueueing the evicted identity left EgressDropped() = %d, want 2 — it was treated as already queued", got)
	}
	if tail := core.egress[len(core.egress)-1].id.key; tail != evicted {
		t.Fatalf("the re-enqueued identity sits at %q, want the queue tail %q", tail, evicted)
	}
	if got := core.QueuedResponses(); got != maxEgressEntries {
		t.Fatalf("QueuedResponses() = %d after the re-enqueue, want the cap %d", got, maxEgressEntries)
	}
}

// TestEgress_noValueRetention proves the CR-02 retention bound: enqueuing one
// DataReplication answer per stored key retains no record Data in the queue.
// The store holds answers values of valueLen bytes each — 256 × 60 KiB ≈
// 15 MiB — so an enqueue path that looked a record up (the production Lookup
// clones the value) would allocate that whole volume. TotalAlloc is the
// probe rather than HeapAlloc because CR-02 has two halves: the queue pinning
// up to maxEgressEntries value copies (retention) and the pre-dedupe Lookup
// allocating and discarding a full copy per duplicate request (allocation
// amplification) — only the cumulative counter observes both. The bound is
// deliberately generous, a small fraction of a single value's volume and far
// below the ≈15 MiB the defect would allocate.
func TestEgress_noValueRetention(t *testing.T) {
	// Not parallel: the allocation probe needs a quiet package — a parallel
	// sibling's allocations would contaminate TotalAlloc.
	core := newEgressCore()
	st, ok := core.store.(*egressStore)
	if !ok {
		t.Fatalf("newEgressCore store is a %T, want the white-box *egressStore", core.store)
	}
	now := egressNow()
	var out port.Outbox
	core.Advance(now, &out) // startup Hello + empty listing, outside the probe
	out.Drain()

	const (
		answers   = 256
		valueLen  = 60 * 1024
		maxGrowth = 1 << 20 // 1 MiB, versus the ≈15 MiB value volume
	)
	keys := make([]string, answers)
	for i := range answers {
		key := fmt.Sprintf("bulk/%06d", i)
		keys[i] = key
		ver := version.PackVersion(egressTestNowMs, uint16(i), egressTestPeer)
		st.versions[key] = ver
		st.records[key] = storage.Record{Version: ver, Data: bytes.Repeat([]byte("x"), valueLen)}
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for _, key := range keys {
		core.handleDataRequest(now, "peer", key)
	}
	runtime.ReadMemStats(&after)

	if got := core.QueuedResponses(); got != answers {
		t.Fatalf("QueuedResponses() = %d, want %d — every present key queues one answer", got, answers)
	}
	if growth := after.TotalAlloc - before.TotalAlloc; growth > maxGrowth {
		t.Fatalf("enqueuing %d answers over %d-byte values allocated %d bytes, want <= %d — the queue pinned value copies (CR-02)",
			answers, valueLen, growth, maxGrowth)
	}
}

// TestEgress_drainSkipsGoneRecords proves the drain-time truthfulness
// (XPORT-07): a queued DataReplication whose key vanished from the store
// emits nothing — the queue pins no Data, and a record that no longer exists
// must not ship a stale copy.
func TestEgress_drainSkipsGoneRecords(t *testing.T) {
	t.Parallel()
	core := newEgressCore()
	st, _ := core.store.(*egressStore)
	now := egressNow()
	var out port.Outbox
	core.Advance(now, &out) // startup Hello + empty listing, outside the assertion
	out.Drain()

	rec := storage.Record{Version: version.PackVersion(egressTestNowMs, 0, egressTestPeer), Data: []byte("v")}
	st.records["gone/key"] = rec
	st.versions["gone/key"] = rec.Version
	core.enqueueResponse(now, "peer", port.DataReplication{Key: "gone/key", Record: rec})
	delete(st.records, "gone/key")
	delete(st.versions, "gone/key")

	core.Advance(now, &out)
	if got := len(out.Drain()); got != 0 {
		t.Fatalf("the drain emitted %d datagrams for a vanished record, want 0", got)
	}
	if got := core.QueuedResponses(); got != 0 {
		t.Fatalf("QueuedResponses() = %d after the drain, want 0", got)
	}
}
