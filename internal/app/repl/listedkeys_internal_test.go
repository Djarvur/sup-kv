package repl

// White-box tests for the retained listed-key payload and its publication
// generation (PROTO-01, T-4-41). They live in package repl because the
// retained total (listedKeyCount), the union and the generation are
// unexported state no external test can assert directly, and because the
// hostile multi-peer flood must prove the cap from the inside. The
// _internal_test.go suffix is the documented testpackage exemption.

import (
	"fmt"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The internal-test fixtures: a fixed virtual base, the local node id and the
// peer id stamped into the fixtures' versions. Kept local because package
// repl cannot import the external repl_test package's constants.
const (
	listedTestNowMs = int64(1730000000000)
	listedTestLocal = uint16(1)
	listedTestPeer  = uint16(2)

	// listedFloodPeers and listedFloodKeys size the hostile retention flood:
	// 200 spoofed node ids times 1 000 keys each is 200 000 keys offered, so
	// whole-peer-set eviction must keep the retained total at or under
	// maxListedKeysTotal.
	listedFloodPeers = 200
	listedFloodKeys  = 1000
)

// listedStateStore adapts *storage.Store to the core's StateStore seam — the
// same four delegations the external tests' stateStore makes; package repl
// cannot import that type, so the wrapper is local (the egressStore
// precedent).
type listedStateStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s listedStateStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Version returns the stored version for key, tombstones included.
func (s listedStateStore) Version(key string) (version.RecordVersion, bool) {
	return s.store.Version(key)
}

// Lookup returns the stored record for key, tombstones included.
func (s listedStateStore) Lookup(key string) (storage.Record, bool) { return s.store.Lookup(key) }

// Versions returns the shallow key→version view, tombstones included.
func (s listedStateStore) Versions() map[string]version.RecordVersion { return s.store.Versions() }

// newListedCore returns a core over a real empty store at the fixed test
// instant — the generation test's shape; the bounded flood seeds its fixture
// versions without storing anything.
func newListedCore() *Core {
	fixedNow := func() time.Time { return time.UnixMilli(listedTestNowMs) }
	st := &listedStateStore{store: storage.NewStore(listedTestLocal, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})}
	return New(listedTestLocal, [16]byte{1}, st, Config{})
}

// listedDatagram wraps one header-valid one-part listing from peer — the
// 1-of-1 shape whose single part completes the round.
func listedDatagram(peer uint16, syncID uint64, entries []port.SyncEntry) port.InboundDatagram {
	return port.InboundDatagram{
		Source:       fmt.Sprintf("n%d", peer),
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       peer,
		RunID:        [16]byte{byte(peer)},
		Msg:          port.DataSync{SyncID: syncID, ChunkIndex: 1, ChunkCount: 1, Entries: entries},
	}
}

// TestSync_listedKeysBounded is the T-4-41 acceptance: a hostile segment
// floods complete one-part listings from 200 distinct node_ids, each listing
// 1 000 distinct keys built on one shared version set. After the flood the
// retained total never exceeds maxListedKeysTotal, ListedKeysEvicted is
// positive, the published union is no larger, and ListingComplete stays true
// — eviction bounds the remotely fillable payload without closing the gate.
func TestSync_listedKeysBounded(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(listedTestNowMs)
	core := newListedCore()
	versions := make([]version.RecordVersion, listedFloodKeys)
	for j := range versions {
		versions[j] = version.PackVersion(listedTestNowMs, uint16(j), listedTestPeer)
	}
	var out port.Outbox
	for peer := range listedFloodPeers {
		nodeID := uint16(100 + peer)
		entries := make([]port.SyncEntry, listedFloodKeys)
		for j := range entries {
			entries[j] = port.SyncEntry{Key: fmt.Sprintf("p%03d/k%04d", peer, j), Version: versions[j]}
		}
		core.HandleDatagram(now, listedDatagram(nodeID, uint64(peer+1), entries), &out)
		out.Drain()
		if got := core.listedKeyCount; got > maxListedKeysTotal {
			t.Fatalf("after peer %d the retained listed total is %d, want at or under %d — the cap must hold on every listing, not only at the end",
				peer, got, maxListedKeysTotal)
		}
	}
	if got := core.listedKeyCount; got > maxListedKeysTotal {
		t.Fatalf("retained listed total = %d, want at or under maxListedKeysTotal = %d", got, maxListedKeysTotal)
	}
	if got := core.ListedKeysEvicted(); got == 0 {
		t.Fatal("ListedKeysEvicted() = 0 after a 200 000-key flood, want a positive count — whole-peer-set eviction must release keys")
	}
	if got := len(core.ListedKeysSnapshot()); got > maxListedKeysTotal {
		t.Fatalf("listed union holds %d keys, want at or under %d", got, maxListedKeysTotal)
	}
	if got := len(core.listedKeys); got != core.listedKeyCount {
		t.Fatalf("union length %d != retained total %d — the incrementally maintained total drifted from the union", got, core.listedKeyCount)
	}
	if !core.ListingComplete() || !core.SyncArrived() {
		t.Fatalf("ListingComplete/SyncArrived = %v/%v after the flood, want true/true — eviction bounds the payload without closing the gate",
			core.ListingComplete(), core.SyncArrived())
	}
}

// TestSync_listingGenerationContentOnly pins the generation's meaning
// (PROTO-01, T-4-05): it is a CONTENT version. The first complete listing —
// empty or not — moves it exactly once and sets ListingComplete; an identical
// repeat moves nothing; a listing whose key set changes moves it again.
func TestSync_listingGenerationContentOnly(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(listedTestNowMs)
	core := newListedCore()
	var out port.Outbox
	send := func(syncID uint64, entries ...port.SyncEntry) {
		t.Helper()
		core.HandleDatagram(now, listedDatagram(listedTestPeer, syncID, entries), &out)
		out.Drain()
	}

	send(1)
	first := core.ListingGeneration()
	if first != 1 || !core.ListingComplete() {
		t.Fatalf("after the first complete empty listing generation = %d, complete = %v, want 1/true — the completion itself is one content change",
			first, core.ListingComplete())
	}
	send(2)
	if got := core.ListingGeneration(); got != first {
		t.Fatalf("an identical repeated complete listing moved the generation %d -> %d, want no move", first, got)
	}

	keyA := port.SyncEntry{Key: "gen/a", Version: version.PackVersion(listedTestNowMs, 1, listedTestPeer)}
	keyB := port.SyncEntry{Key: "gen/b", Version: version.PackVersion(listedTestNowMs, 2, listedTestPeer)}
	send(3, keyA, keyB)
	changed := core.ListingGeneration()
	if changed == first {
		t.Fatalf("a changed key set left generation at %d, want a move", changed)
	}
	send(4, keyA, keyB)
	if got := core.ListingGeneration(); got != changed {
		t.Fatalf("an identical repeated complete listing moved the generation %d -> %d, want no move", changed, got)
	}

	send(5, keyB)
	shrunk := core.ListingGeneration()
	if shrunk == changed {
		t.Fatalf("a listing whose key set lost %q left generation at %d, want a move", keyA.Key, shrunk)
	}
	if got := core.ListedKeysSnapshot(); len(got) != 1 {
		t.Fatalf("listed union holds %d keys after the shrink, want exactly 1 (%q)", len(got), keyB.Key)
	} else if _, ok := got[keyB.Key]; !ok {
		t.Fatalf("listed union %v lost %q after the shrink", got, keyB.Key)
	}
}

// TestSync_listingGenerationSharedKeys pins the union-membership half of the
// content contract: the generation moves when a key enters or leaves the
// union, never merely because one peer's own set was rewritten. A second peer
// listing keys the first already lists changes no membership; a peer dropping
// keys another still lists changes none; the last lister's release does.
func TestSync_listingGenerationSharedKeys(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(listedTestNowMs)
	core := newListedCore()
	var out port.Outbox
	keyA := port.SyncEntry{Key: "shared/a", Version: version.PackVersion(listedTestNowMs, 1, listedTestPeer)}
	keyB := port.SyncEntry{Key: "shared/b", Version: version.PackVersion(listedTestNowMs, 2, listedTestPeer)}
	send := func(peer uint16, syncID uint64, entries ...port.SyncEntry) {
		t.Helper()
		core.HandleDatagram(now, listedDatagram(peer, syncID, entries), &out)
		out.Drain()
	}

	send(listedTestPeer, 1, keyA, keyB)
	first := core.ListingGeneration()
	if first == 0 {
		t.Fatal("the first complete listing moved no generation, want a move")
	}

	// A second peer lists the identical keys: the union does not change.
	send(listedTestPeer+1, 2, keyA, keyB)
	if got := core.ListingGeneration(); got != first {
		t.Fatalf("a second peer's identical listing moved the generation %d -> %d, want no move — the union did not change", first, got)
	}
	// The second peer drops to an empty listing: the first peer still lists
	// both keys, so the union still does not change.
	send(listedTestPeer+1, 3)
	if got := core.ListingGeneration(); got != first {
		t.Fatalf("a peer dropping shared keys moved the generation %d -> %d, want no move — another peer still lists them", first, got)
	}
	// The last lister drops both: now the union empties.
	send(listedTestPeer, 4)
	if got := core.ListingGeneration(); got == first {
		t.Fatalf("the last lister's release left the generation at %d, want a move", got)
	}
	if got := len(core.ListedKeysSnapshot()); got != 0 {
		t.Fatalf("listed union holds %d keys after the last lister released them, want 0", got)
	}
}
