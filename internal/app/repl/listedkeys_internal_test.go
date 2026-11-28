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

// Observe absorbs one received version into the store's clock.
func (s listedStateStore) Observe(ver version.RecordVersion) time.Duration {
	return s.store.Observe(ver)
}

// Generation returns the store's content generation.
func (s listedStateStore) Generation() uint64 { return s.store.Generation() }

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

// listedHello builds the Hello the listing binding requires (CR-01): the same
// node id and run_id listedDatagram stamps, so the sender is a known peer
// identity before its listing arrives.
func listedHello(peer uint16) port.InboundDatagram {
	return port.InboundDatagram{
		Source:       fmt.Sprintf("n%d", peer),
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       peer,
		RunID:        [16]byte{byte(peer)},
		Msg:          port.Hello{},
	}
}

// listedFeedHello teaches core peer's identity and drains the outbox — the
// first-seen branch enqueues a targeted listing, which these tests never
// drain further.
func listedFeedHello(core *Core, now time.Time, peer uint16) {
	var out port.Outbox
	core.HandleDatagram(now, listedHello(peer), &out)
	out.Drain()
}

// listedSpoofDatagram builds one complete one-part listing from nodeID, an
// arbitrary runID and an arbitrary source handle — the forged-envelope shape
// the CR-01 proofs drive.
func listedSpoofDatagram(nodeID uint16, runID [16]byte, source, key string) port.InboundDatagram {
	return port.InboundDatagram{
		Source:       source,
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       nodeID,
		RunID:        runID,
		Msg: port.DataSync{SyncID: 1, ChunkIndex: 1, ChunkCount: 1, Entries: []port.SyncEntry{{
			Key:     key,
			Version: version.PackVersion(listedTestNowMs, 1, nodeID),
		}}},
	}
}

// TestSync_unboundListingDiscarded is CR-01's first proof: the retained
// listed-key union is filled only from a peer-table identity, so a listing
// from a sender that never sent a Hello — the review's permanent-denial
// datagram — retains nothing, completes nothing and opens nothing, and no
// amount of elapsed time makes the forged keys appear.
func TestSync_unboundListingDiscarded(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(listedTestNowMs)
	core := newListedCore()
	var out port.Outbox

	const victimKey = "victim/key"
	core.HandleDatagram(now, listedSpoofDatagram(0x66, [16]byte{0xEE}, "spoof-102", victimKey), &out)
	out.Drain()

	if got := core.Discards().UnboundListing; got != 1 {
		t.Fatalf("UnboundListing discards after an unbound listing = %d, want 1 — the drop is the dedicated unbound_listing class (CR-01, New-2)", got)
	}
	if got := core.Discards().DuplicateNodeID; got != 0 {
		t.Fatalf("DuplicateNodeID discards after an unbound listing = %d, want 0 — an unbound listing is not an identity collision", got)
	}
	if got := len(core.ListedKeysSnapshot()); got != 0 {
		t.Fatalf("the unbound listing retained %d keys, want 0 — the union must never take an unknown identity's keys", got)
	}
	if got := len(core.listedByPeer); got != 0 {
		t.Fatalf("the unbound listing created %d listedByPeer rows, want 0", got)
	}
	if core.ListingComplete() || core.SyncArrived() {
		t.Fatal("an unbound complete listing opened the gate — no completeness may follow the drop (CR-01)")
	}
	if got := core.ListingGeneration(); got != 0 {
		t.Fatalf("ListingGeneration() = %d after an unbound listing, want 0 — nothing was published", got)
	}

	// The review's 24-hour proof: with no peer row the forged keys must never
	// surface, however long the driver runs.
	core.Advance(now.Add(24*time.Hour), &out)
	out.Drain()
	if got := len(core.ListedKeysSnapshot()); got != 0 {
		t.Fatalf("after 24h the unbound listing still retained %d keys, want 0 — retention is bound to a live peer identity (CR-01)", got)
	}
	if core.ListingComplete() || core.SyncArrived() {
		t.Fatal("the gate opened on the forged listing after 24h, want it never opened by an unbound sender")
	}
}

// TestSync_forgedRunIDCannotReplaceListing is CR-01's second proof: a listing
// from a KNOWN node_id whose envelope run_id differs from the row's is
// discarded, so it cannot flush the peer's protective set. A real restart —
// a Hello carrying the new run — binds the new identity, and only then does
// that run's listing replace the previous set.
func TestSync_forgedRunIDCannotReplaceListing(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(listedTestNowMs)
	core := newListedCore()
	var out port.Outbox

	const (
		legitKey = "legit/listed"
		nextKey  = "next/run-listed"
	)
	legitRun := [16]byte{byte(listedTestPeer)}
	forgedRun := [16]byte{0xEE, 0xEE}
	core.HandleDatagram(now, listedHello(listedTestPeer), &out)
	out.Drain()
	core.HandleDatagram(now, listedDatagram(listedTestPeer, 1, []port.SyncEntry{{
		Key:     legitKey,
		Version: version.PackVersion(listedTestNowMs, 1, listedTestPeer),
	}}), &out)
	out.Drain()
	if _, ok := core.ListedKeysSnapshot()[legitKey]; !ok {
		t.Fatalf("the bound listing did not retain %q, want it in the union", legitKey)
	}

	// Same node_id, different run_id: discarded, and the real set survives.
	core.HandleDatagram(now, listedSpoofDatagram(listedTestPeer, forgedRun, "spoof-2", "attacker/key"), &out)
	out.Drain()
	union := core.ListedKeysSnapshot()
	if _, ok := union[legitKey]; !ok {
		t.Fatalf("a forged-run listing dropped %q from the union, want the real run's set untouched (CR-01)", legitKey)
	}
	if _, ok := union["attacker/key"]; ok {
		t.Fatal("a forged-run listing installed attacker/key, want it discarded")
	}
	if got := core.Discards().UnboundListing; got != 1 {
		t.Fatalf("UnboundListing discards after the forged-run listing = %d, want 1 — the run-mismatched drop is the unbound_listing class", got)
	}
	if got := core.Discards().DuplicateNodeID; got != 0 {
		t.Fatalf("DuplicateNodeID discards after the forged-run listing = %d, want 0 — a forged run is not an identity collision", got)
	}
	if got := core.PeerSnapshot()[listedTestPeer].RunID; got != legitRun {
		t.Fatalf("the peer row's run became %x, want the authoritative %x", got[:], legitRun[:])
	}

	// A genuine restart Hello re-binds the identity; only then is the new
	// run's listing accepted and the old set replaced wholesale.
	restarted := port.InboundDatagram{
		Source:       fmt.Sprintf("n%d", listedTestPeer),
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       listedTestPeer,
		RunID:        forgedRun,
		Msg:          port.Hello{},
	}
	core.HandleDatagram(now, restarted, &out)
	out.Drain()
	core.HandleDatagram(now, listedSpoofDatagram(listedTestPeer, forgedRun, fmt.Sprintf("n%d", listedTestPeer), nextKey), &out)
	out.Drain()
	union = core.ListedKeysSnapshot()
	if _, ok := union[nextKey]; !ok {
		t.Fatalf("the restarted run's listing did not retain %q, want it in the union", nextKey)
	}
	if _, ok := union[legitKey]; ok {
		t.Fatalf("the previous run's %q survived the replacement, want the new complete listing to replace the set", legitKey)
	}
}

// TestSync_emptyListingCreatesNoRow pins the row-bound half of CR-01: a
// complete empty listing adds nothing to the union, so it must not create a
// listedByPeer row (or an LRU element) that the size cap can never reclaim —
// the review's un-evictable-row growth.
func TestSync_emptyListingCreatesNoRow(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(listedTestNowMs)
	core := newListedCore()
	var out port.Outbox

	listedFeedHello(core, now, listedTestPeer)
	core.HandleDatagram(now, listedDatagram(listedTestPeer, 1, nil), &out)
	out.Drain()

	if got := len(core.listedByPeer); got != 0 {
		t.Fatalf("an empty complete listing created %d listedByPeer rows, want 0", got)
	}
	if got := core.listedOrder.Len(); got != 0 {
		t.Fatalf("an empty complete listing left %d LRU elements, want 0", got)
	}
	if !core.ListingComplete() {
		t.Fatal("an empty complete listing did not complete the listing, want ListingComplete true")
	}
}

// TestSync_listedKeysExpire is CR-01's time bound: a set nobody republishes
// within two sync intervals leaves the union, while the peer row and the
// startup-gate completeness stay (the expiry is not an eviction and not a
// re-close). A fresh complete listing from the still-known peer re-retains
// the keys.
func TestSync_listedKeysExpire(t *testing.T) {
	t.Parallel()
	const syncInterval = time.Second
	fixedNow := func() time.Time { return time.UnixMilli(listedTestNowMs) }
	st := &listedStateStore{store: storage.NewStore(listedTestLocal, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})}
	core := New(listedTestLocal, [16]byte{1}, st, Config{SyncInterval: syncInterval})
	now := time.UnixMilli(listedTestNowMs)
	var out port.Outbox

	expiring := port.SyncEntry{Key: "expire/me", Version: version.PackVersion(listedTestNowMs, 1, listedTestPeer)}
	listedFeedHello(core, now, listedTestPeer)
	core.HandleDatagram(now, listedDatagram(listedTestPeer, 1, []port.SyncEntry{expiring}), &out)
	out.Drain()
	if _, ok := core.ListedKeysSnapshot()[expiring.Key]; !ok {
		t.Fatalf("the fresh listing did not retain %q, want it in the union", expiring.Key)
	}

	// Nothing republishes: past two sync intervals the set is released while
	// the peer row (peer_timeout is longer) and the completeness flag stay.
	core.Advance(now.Add(2*syncInterval+time.Millisecond), &out)
	out.Drain()
	if got := len(core.ListedKeysSnapshot()); got != 0 {
		t.Fatalf("the retained union still holds %d keys past two sync intervals, want 0 — retention is bounded in time (CR-01)", got)
	}
	if got := core.PeerSnapshot(); len(got) != 1 {
		t.Fatalf("the peer table holds %d rows after the expiry, want 1 — the expiry must not be an eviction", len(got))
	}
	if !core.ListingComplete() {
		t.Fatal("ListingComplete() = false after the expiry, want the startup-gate completeness to stay — expiry is not a re-close")
	}
	if got := core.ListedKeysEvicted(); got != 1 {
		t.Fatalf("ListedKeysEvicted() = %d after the expiry, want 1 — a released key is counted", got)
	}

	// A republished complete listing from the same peer restores the union.
	core.HandleDatagram(now.Add(3*syncInterval), listedDatagram(listedTestPeer, 2, []port.SyncEntry{expiring}), &out)
	out.Drain()
	if _, ok := core.ListedKeysSnapshot()[expiring.Key]; !ok {
		t.Fatalf("the republished listing did not restore %q, want it back in the union", expiring.Key)
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
		listedFeedHello(core, now, nodeID)
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
	listedFeedHello(core, now, listedTestPeer)
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

	listedFeedHello(core, now, listedTestPeer)
	listedFeedHello(core, now, listedTestPeer+1)
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

// observingStore wraps a real store and counts Observe calls: the real store
// proves the clock advance end to end, the counter proves once-per-part and
// zero-for-empty/discarded. Generation delegates untouched — nothing in these
// tests caches a build, so the pass-through is the contract.
type observingStore struct {
	store       *storage.Store
	observes    int
	maxObserved version.RecordVersion
}

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s *observingStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Version returns the stored version for key, tombstones included.
func (s *observingStore) Version(key string) (version.RecordVersion, bool) {
	return s.store.Version(key)
}

// Lookup returns the stored record for key, tombstones included.
func (s *observingStore) Lookup(key string) (storage.Record, bool) { return s.store.Lookup(key) }

// Versions returns the shallow key→version view, tombstones included.
func (s *observingStore) Versions() map[string]version.RecordVersion { return s.store.Versions() }

// Observe counts the call, records the greatest absorbed version and feeds
// the real store's clock.
func (s *observingStore) Observe(ver version.RecordVersion) time.Duration {
	s.observes++
	if s.observes == 1 || ver.Compare(s.maxObserved) > 0 {
		s.maxObserved = ver
	}
	return s.store.Observe(ver)
}

// Generation returns the store's content generation.
func (s *observingStore) Generation() uint64 { return s.store.Generation() }

// newObservingCore returns a core over a real store wrapped by the counting
// fake at the fixed test instant, with the production default MaxClockLead.
func newObservingCore() (*Core, *observingStore) {
	fixedNow := func() time.Time { return time.UnixMilli(listedTestNowMs) }
	st := &observingStore{store: storage.NewStore(listedTestLocal, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})}
	return New(listedTestLocal, [16]byte{1}, st, Config{}), st
}

// observeNow returns the virtual instant the observe rows drive from.
func observeNow() time.Time { return time.UnixMilli(listedTestNowMs) }

// TestSync_observePerPart pins PROTO-01's absorption point: one non-empty
// listing part feeds its entry version into the store clock, so the next
// local write strictly beats the peer's listed state.
func TestSync_observePerPart(t *testing.T) {
	t.Parallel()
	core, st := newObservingCore()
	absorbed := version.PackVersion(listedTestNowMs+10_000, 3, listedTestPeer)
	var out port.Outbox
	listedFeedHello(core, observeNow(), listedTestPeer)
	core.HandleDatagram(observeNow(), listedDatagram(listedTestPeer, 1, []port.SyncEntry{
		{Key: "peer/absorbed", Version: absorbed},
	}), &out)
	if got := st.observes; got != 1 {
		t.Fatalf("Observe calls after a one-entry part = %d, want 1", got)
	}
	ver, _, err := st.store.Set("local/after-listing", []byte("v"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got := ver.Compare(absorbed); got <= 0 {
		t.Fatalf("next local write %x does not strictly beat the listed %x", ver[:], absorbed[:])
	}
}

// TestSync_observePerPartEmptyAndDiscarded pins the pipeline position: an
// empty part and a datagram the validation pipeline discards perform no clock
// call at all, so a hostile datagram cannot advance the HLC before it passed
// validation (T-4-10).
func TestSync_observePerPartEmptyAndDiscarded(t *testing.T) {
	t.Parallel()

	t.Run("an empty part absorbs nothing", func(t *testing.T) {
		t.Parallel()
		core, st := newObservingCore()
		var out port.Outbox
		listedFeedHello(core, observeNow(), listedTestPeer)
		core.HandleDatagram(observeNow(), listedDatagram(listedTestPeer, 2, nil), &out)
		if got := st.observes; got != 0 {
			t.Fatalf("Observe calls after an empty part = %d, want 0", got)
		}
	})

	t.Run("a part discarded for an out-of-bound version absorbs nothing", func(t *testing.T) {
		t.Parallel()
		core, st := newObservingCore()
		var out port.Outbox
		listedFeedHello(core, observeNow(), listedTestPeer)
		ahead := version.PackVersion(
			listedTestNowMs+int64(defaultMaxClockLead/time.Millisecond)+1000, 0, listedTestPeer)
		core.HandleDatagram(observeNow(), listedDatagram(listedTestPeer, 3, []port.SyncEntry{
			{Key: "peer/future", Version: ahead},
		}), &out)
		if got := st.observes; got != 0 {
			t.Fatalf("Observe calls after a discarded part = %d, want 0 (T-4-10)", got)
		}
		if got := core.Discards().BadVersion; got != 1 {
			t.Fatalf("BadVersion discards = %d, want 1 — the part must be rejected before the clock", got)
		}
	})
}

// TestSync_observeOncePerPart pins the exactly-once contract: a part with
// several entries absorbs one version — the greatest by the full 16-byte
// comparison — and the next local write beats it.
func TestSync_observeOncePerPart(t *testing.T) {
	t.Parallel()
	core, st := newObservingCore()
	low := version.PackVersion(listedTestNowMs+1000, 0, listedTestPeer)
	high := version.PackVersion(listedTestNowMs+3000, 0, listedTestPeer)
	mid := version.PackVersion(listedTestNowMs+2000, 0, listedTestPeer)
	var out port.Outbox
	listedFeedHello(core, observeNow(), listedTestPeer)
	core.HandleDatagram(observeNow(), listedDatagram(listedTestPeer, 4, []port.SyncEntry{
		{Key: "peer/a", Version: low},
		{Key: "peer/b", Version: high},
		{Key: "peer/c", Version: mid},
	}), &out)
	if got := st.observes; got != 1 {
		t.Fatalf("Observe calls after a three-entry part = %d, want exactly 1", got)
	}
	if st.maxObserved != high {
		t.Fatalf("absorbed %x, want the part's maximum %x", st.maxObserved[:], high[:])
	}
	ver, _, err := st.store.Set("local/after-max", []byte("v"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got := ver.Compare(high); got <= 0 {
		t.Fatalf("next local write %x does not strictly beat the maximum %x", ver[:], high[:])
	}
}

// TestSync_generationPassThrough pins the generation seam a cached listing
// build trusts (T-4-43): the core's read paths leave the store's content
// generation unchanged, and a real store mutation moves it by exactly one.
func TestSync_generationPassThrough(t *testing.T) {
	t.Parallel()
	core, st := newObservingCore()
	if got := st.Generation(); got != 0 {
		t.Fatalf("Generation() on a fresh store = %d, want 0", got)
	}
	_, _, err := st.store.Set("local/older", []byte("local"))
	if err != nil {
		t.Fatalf("seed Set() error = %v, want nil", err)
	}
	if got := st.Generation(); got != 1 {
		t.Fatalf("Generation() after the seed Set = %d, want 1", got)
	}

	// A listing whose entry is older than the stored record drives the
	// Version + Lookup newer-response read path.
	var out port.Outbox
	listedFeedHello(core, observeNow(), listedTestPeer)
	core.HandleDatagram(observeNow(), listedDatagram(listedTestPeer, 5, []port.SyncEntry{
		{Key: "local/older", Version: version.PackVersion(listedTestNowMs-1000, 0, listedTestPeer)},
	}), &out)
	out.Drain()
	if got := st.Generation(); got != 1 {
		t.Fatalf("Generation() after the core's read paths = %d, want the unchanged 1", got)
	}

	// A real store mutation moves it exactly once more.
	_, _, err = st.store.Set("local/newer", []byte("v"))
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if got := st.Generation(); got != 2 {
		t.Fatalf("Generation() after a second Set = %d, want 2", got)
	}
}
