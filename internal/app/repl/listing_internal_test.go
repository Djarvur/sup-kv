package repl

// White-box tests for the bounded, shared, generation-checked listing queue
// (CR-01, T-4-23/T-4-24, PROTO-04): destinations are peer identities rather
// than datagram source handles, the queue is capped with a counted drop, one
// immutable build serves every destination while its sync_interval age and the
// store generation still hold, rotation preserves the original chunk indices,
// and a restart releases the previous run's queued work. They live in package
// repl because the destination key, the caps and the drop counter are
// unexported, and because the generation check needs a store whose Versions()
// calls are counted. The _internal_test.go suffix is the documented
// testpackage exemption.

import (
	"encoding/binary"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

const (
	// listingTestNowMs is the fixed virtual base (the D-02 now-seam idiom).
	listingTestNowMs = int64(1730000000000)
	// listingTestLocal is the local node id of every core these tests build.
	listingTestLocal = uint16(1)
	// listingTestStep is the drain loop's virtual step: comfortably above the
	// jittered listing spacing at the 500/s default (2ms + 10% = 2.2ms), so
	// each Advance crosses exactly one pacer slot, and far below every timer
	// (Hello 1s, sync 30s), so the loops never trip an unrelated firing.
	listingTestStep = 3 * time.Millisecond
	// hostileTriggers is the CR-01 probe's hostile Hello count.
	hostileTriggers = 200
	// listingTestChunks is the multi-chunk fixture's chunk count.
	listingTestChunks = 4
	// listingTestBudget is the tiny chunk budget that makes each ~100-byte
	// fixture key its own chunk: 96 bytes of per-chunk overhead plus 124 bytes
	// of entry estimate exceed it on the second entry, so the walk closes a
	// chunk after every entry.
	listingTestBudget = 128
	// listingTestKeyLen sizes the multi-chunk fixture keys.
	listingTestKeyLen = 100
)

// listingStore is the white-box StateStore for the listing tests: a key→version
// map whose Versions() calls are counted and whose Generation() moves only when
// a seeding helper actually changes the map — the 04-03 content-generation
// contract the shared build checks.
type listingStore struct {
	versions      map[string]version.RecordVersion
	records       map[string]storage.Record
	versionsCalls int
	generation    uint64
}

// newListingStore returns an empty fake store.
func newListingStore() *listingStore {
	return &listingStore{
		versions: make(map[string]version.RecordVersion),
		records:  make(map[string]storage.Record),
	}
}

// set installs one record and bumps the generation, mirroring the production
// store's install-point bump.
func (s *listingStore) set(key string, ver version.RecordVersion) {
	s.versions[key] = ver
	s.records[key] = storage.Record{Version: ver}
	s.generation++
}

// ApplyRemote joins one record into the fake store.
func (s *listingStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	s.set(key, rec.Version)
	return nil, 0, nil
}

// Version returns the fake store's version for key.
func (s *listingStore) Version(key string) (version.RecordVersion, bool) {
	v, ok := s.versions[key]
	return v, ok
}

// Lookup returns the fake store's record for key.
func (s *listingStore) Lookup(key string) (storage.Record, bool) {
	rec, ok := s.records[key]
	return rec, ok
}

// Versions returns a fresh copy of the version view and counts the call — the
// shared-build probe.
func (s *listingStore) Versions() map[string]version.RecordVersion {
	s.versionsCalls++
	out := make(map[string]version.RecordVersion, len(s.versions))
	maps.Copy(out, s.versions)
	return out
}

// Observe is a clock-less no-op: this fake holds no clock.
func (*listingStore) Observe(_ version.RecordVersion) time.Duration { return 0 }

// Generation returns the fake's mutation counter.
func (s *listingStore) Generation() uint64 { return s.generation }

// listingTime returns the virtual instant off after the fixed test base.
func listingTime(off time.Duration) time.Time {
	return time.UnixMilli(listingTestNowMs).Add(off)
}

// listingRunID returns a deterministic 16-byte run id distinct per (identity,
// generation) pair.
func listingRunID(identity, generation int) [16]byte {
	var id [16]byte
	binary.BigEndian.PutUint32(id[:4], uint32(identity))
	id[15] = byte(generation)
	return id
}

// listingHello builds one decoded Hello datagram from nodeID announcing runID
// from source.
func listingHello(nodeID uint16, runID [16]byte, source string) port.InboundDatagram {
	return port.InboundDatagram{
		Source:       source,
		Status:       port.Decoded,
		ProtoVersion: SupportedProtoVersion,
		NodeID:       nodeID,
		RunID:        runID,
		Msg:          port.Hello{},
	}
}

// listingKey returns the i-th multi-chunk fixture key: listingTestKeyLen bytes,
// so one entry's estimate alone exceeds listingTestBudget.
func listingKey(i int) string {
	prefix := fmt.Sprintf("k%d/", i)
	return prefix + strings.Repeat("x", listingTestKeyLen-len(prefix))
}

// newListingCore returns a core over a fresh listingStore seeded with chunks
// multi-chunk fixture keys, its startup timers fired and every queue drained,
// so a test's own triggers are the only traffic it observes. The returned time
// is the virtual instant the drain stopped at.
func newListingCore(t *testing.T, cfg Config, chunks int) (*Core, *listingStore, time.Time) {
	t.Helper()
	st := newListingStore()
	for i := range chunks {
		st.set(listingKey(i), version.PackVersion(listingTestNowMs, uint16(i), listingTestLocal))
	}
	core := New(listingTestLocal, [16]byte{1, 2, 3}, st, cfg)
	_, now := drainListingQueues(core, listingTime(0))
	return core, st, now
}

// drainListingQueues advances core until every core queue is empty, returning
// every drained outbound and the virtual instant it stopped at.
func drainListingQueues(core *Core, now time.Time) ([]port.Outbound, time.Time) {
	var out port.Outbox
	core.Advance(now, &out)
	drained := out.Drain()
	for core.QueuedSends() > 0 {
		now = now.Add(listingTestStep)
		core.Advance(now, &out)
		drained = append(drained, out.Drain()...)
	}
	return drained, now
}

// targetedListingSyncs returns every unicast DataSync chunk of drained
// addressed to handle, failing the test when one is addressed elsewhere — the
// targeted-listing probe (package repl cannot use repl_test's helper).
func targetedListingSyncs(t *testing.T, drained []port.Outbound, handle string) []port.DataSync {
	t.Helper()
	var syncs []port.DataSync
	for _, ob := range drained {
		if ob.Broadcast {
			continue
		}
		msg, ok := ob.Msg.(port.DataSync)
		if !ok {
			continue
		}
		if ob.To != handle {
			t.Fatalf("a targeted DataSync chunk is addressed to %q, want %q", ob.To, handle)
		}
		syncs = append(syncs, msg)
	}
	return syncs
}

// listingChunkIndices returns the wire chunk_index of every chunk in emission
// order — the rotation probe.
func listingChunkIndices(syncs []port.DataSync) []uint32 {
	out := make([]uint32, 0, len(syncs))
	for _, sync := range syncs {
		out = append(out, sync.ChunkIndex)
	}
	return out
}

// listingKeys returns every entry key across the chunks in emission order.
func listingKeys(syncs []port.DataSync) []string {
	var keys []string
	for _, sync := range syncs {
		for _, entry := range sync.Entries {
			keys = append(keys, entry.Key)
		}
	}
	return keys
}

// TestListing_hostileBounded proves CR-01's bounds: 200 hostile Hello pairs
// stop at maxListingDestinations in-flight destinations and maxListingChunks
// queued chunks with every refused trigger counted, and a source handle is
// never a destination — one node identity announcing 200 fresh run ids from
// 200 distinct spoofed source handles still owns exactly one FIFO.
func TestListing_hostileBounded(t *testing.T) {
	t.Parallel()
	now := listingTime(0)
	t.Run("200 identities and source handles stop at the destination cap with counted drops", func(t *testing.T) {
		t.Parallel()
		core, _, _ := newListingCore(t, Config{}, 0)
		var out port.Outbox
		for i := range hostileTriggers {
			id := uint16(i + 2) // node ids 2.., clear of the local id 1
			source := fmt.Sprintf("hostile-%03d", i)
			core.HandleDatagram(now, listingHello(id, listingRunID(i, 0), source), &out)
			core.HandleDatagram(now, listingHello(id, listingRunID(i, 1), source), &out)
		}
		out.Drain()
		if got := len(core.listingByDst); got > maxListingDestinations {
			t.Fatalf("200 hostile identities left %d in-flight listing destinations, want at most %d (CR-01, T-4-23)",
				got, maxListingDestinations)
		}
		if got := core.listingCount; got > maxListingChunks {
			t.Fatalf("the hostile burst queued %d listing chunks, want at most %d", got, maxListingChunks)
		}
		if minimum := uint64(hostileTriggers - maxListingDestinations); core.ListingDropped() < minimum {
			t.Fatalf("ListingDropped() = %d after %d triggers past the cap, want at least %d — every refused trigger is counted",
				core.ListingDropped(), hostileTriggers, minimum)
		}
	})
	t.Run("a spoofed source handle cannot mint a destination for one identity", func(t *testing.T) {
		t.Parallel()
		core, _, _ := newListingCore(t, Config{}, 0)
		const identity = uint16(7)
		var out port.Outbox
		core.HandleDatagram(now, listingHello(identity, listingRunID(0, 0), "learn"), &out)
		for i := range hostileTriggers {
			core.HandleDatagram(now, listingHello(identity, listingRunID(i, 1), fmt.Sprintf("spoofed-%03d", i)), &out)
		}
		if got := len(core.listingByDst); got != 1 {
			t.Fatalf(
				"one node identity announcing %d run ids from as many source handles left %d destinations, want exactly 1 — a source handle must never key a listing FIFO (CR-01, T-4-25)",
				hostileTriggers,
				got,
			)
		}
		if got := core.ListingDropped(); got != 0 {
			t.Fatalf("ListingDropped() = %d under the destination cap, want 0", got)
		}
	})
}

// TestListing_sharedBuild proves the shared-build bound (T-4-24): two listing
// triggers inside one sync_interval with an unchanged store generation copy
// Versions() exactly once, and a trigger past the age rebuilds exactly once —
// never one O(store) copy per trigger.
func TestListing_sharedBuild(t *testing.T) {
	t.Parallel()
	core, st, now := newListingCore(t, Config{}, 0)
	const identity = uint16(9)
	var out port.Outbox
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 0), "learn"), &out)
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 1), "second-run"), &out)
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 2), "third-run"), &out)
	if st.versionsCalls != 1 {
		t.Fatalf("two listing triggers inside one sync_interval and one store generation copied the store %d times, want exactly 1 (CR-01)",
			st.versionsCalls)
	}
	core.HandleDatagram(now.Add(defaultSyncInterval), listingHello(identity, listingRunID(0, 3), "aged-run"), &out)
	if st.versionsCalls != 2 {
		t.Fatalf("a trigger at the sync_interval boundary copied the store %d times in total, want 2 — the age bound rebuilds exactly once",
			st.versionsCalls)
	}
}

// TestListing_generationRebuild proves the freshness half of the cache
// (T-4-24, PROTO-04): after a trigger cached the build, a store mutation bumps
// the generation, and the next trigger rebuilds exactly once and emits chunks
// carrying the seeded key — a pre-mutation build is never served.
func TestListing_generationRebuild(t *testing.T) {
	t.Parallel()
	core, st, now := newListingCore(t, Config{}, 0)
	const identity = uint16(11)
	var out port.Outbox
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 0), "learn"), &out)
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 1), "before-seed"), &out)
	if st.versionsCalls != 1 {
		t.Fatalf("the first trigger copied the store %d times, want 1", st.versionsCalls)
	}
	const seeded = "seeded/key"
	st.set(seeded, version.PackVersion(listingTestNowMs, 1, listingTestLocal))
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 2), "after-seed"), &out)
	if st.versionsCalls != 2 {
		t.Fatalf("the trigger after a store mutation copied the store %d times in total, want 2 — the generation change forces exactly one rebuild",
			st.versionsCalls)
	}
	drained, _ := drainListingQueues(core, now)
	if got := listingKeys(targetedListingSyncs(t, drained, "after-seed")); !slices.Contains(got, seeded) {
		t.Fatalf("the post-mutation listing carries keys %v, want the seeded key %q — a cached pre-mutation build was served",
			got, seeded)
	}
}

// TestListing_rotationPreservesIndices proves the rotation contract: the next
// round's start chunk rotates by one while each chunk keeps its ORIGINAL
// 1-based index and the round's chunk_count, so a receiver can still recognize
// completeness and the two rounds cover the same chunk set.
func TestListing_rotationPreservesIndices(t *testing.T) {
	t.Parallel()
	core, _, now := newListingCore(t, Config{SyncChunkBytes: listingTestBudget}, listingTestChunks)
	const identity = uint16(13)
	var out port.Outbox
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 0), "learn"), &out)
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 1), "round-0"), &out)
	firstDrained, now := drainListingQueues(core, now)
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 2), "round-1"), &out)
	secondDrained, _ := drainListingQueues(core, now)

	first := targetedListingSyncs(t, firstDrained, "round-0")
	second := targetedListingSyncs(t, secondDrained, "round-1")
	if got := listingChunkIndices(first); !slices.Equal(got, []uint32{1, 2, 3, 4}) {
		t.Fatalf("round 0 emitted chunk indices %v, want the unrotated 1..4", got)
	}
	if got := listingChunkIndices(second); !slices.Equal(got, []uint32{2, 3, 4, 1}) {
		t.Fatalf("round 1 emitted chunk indices %v, want the one-chunk rotation 2,3,4,1", got)
	}
	byIndex := make(map[uint32][]port.SyncEntry, len(first))
	for _, sync := range first {
		byIndex[sync.ChunkIndex] = sync.Entries
	}
	for _, sync := range second {
		if sync.ChunkCount != uint32(listingTestChunks) {
			t.Fatalf("a rotated chunk carries chunk_count %d, want %d", sync.ChunkCount, listingTestChunks)
		}
		if !slices.Equal(byIndex[sync.ChunkIndex], sync.Entries) {
			t.Fatalf("rotated chunk %d carries different entries than the unrotated chunk of the same index — rotation must reorder, never relabel",
				sync.ChunkIndex)
		}
	}
}

// TestListing_restartDropsOldRunFIFO proves the run-change cleanup: a restart
// releases the previous run's remaining queued chunks before enqueueing the new
// round, so stale work cannot outlive the peer row (CR-01).
func TestListing_restartDropsOldRunFIFO(t *testing.T) {
	t.Parallel()
	core, _, now := newListingCore(t, Config{SyncChunkBytes: listingTestBudget}, listingTestChunks)
	const identity = uint16(15)
	var out port.Outbox
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 0), "learn"), &out)
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 1), "old-run"), &out)
	core.Advance(now, &out) // one paced chunk leaves
	out.Drain()
	if got := core.listingCount; got != listingTestChunks-1 {
		t.Fatalf("after one paced chunk %d chunks remain queued, want %d", got, listingTestChunks-1)
	}
	core.HandleDatagram(now, listingHello(identity, listingRunID(0, 2), "new-run"), &out)
	if got := core.listingCount; got != listingTestChunks {
		t.Fatalf("after the restart %d chunks are queued, want %d — the old run's remaining %d chunks must be released",
			got, listingTestChunks, listingTestChunks-1)
	}
}

// TestListing_chunkCapCoversEveryDestination pins the WR-01 arithmetic: the
// queued-chunk bound must cover one full receiver-acceptable round for every
// targeted destination PLUS the reserved broadcast round. Lowering it below
// that product makes the chunk cap, not the destination cap, refuse
// documented-scale rounds as soon as a handful of destinations are draining —
// the defect this bound exists to remove.
func TestListing_chunkCapCoversEveryDestination(t *testing.T) {
	t.Parallel()
	want := (maxListingDestinations + 1) * maxSyncChunks
	if maxListingChunks < want {
		t.Fatalf(
			"maxListingChunks = %d, want at least %d = (%d destinations + 1 broadcast) × %d chunks"+
				" — the destination cap must be the binding bound for any round a receiver accepts (WR-01)",
			maxListingChunks, want, maxListingDestinations, maxSyncChunks,
		)
	}
}

// TestListing_broadcastNeverStarved proves the WR-01 reservation: with every
// targeted destination slot already in flight, the periodic broadcast round is
// still admitted (the destination cap exempts it), and it drains AHEAD of the
// targeted FIFOs instead of waiting behind their entire queued work.
func TestListing_broadcastNeverStarved(t *testing.T) {
	t.Parallel()
	// A long peer_timeout keeps all 32 targeted rows (and their FIFOs) alive
	// across the periodic-trigger jump this test makes.
	core, _, now := newListingCore(t,
		Config{SyncChunkBytes: listingTestBudget, PeerTimeout: time.Hour}, listingTestChunks)
	var out port.Outbox
	for i := range maxListingDestinations {
		id := uint16(i + 2) // clear of the local node id
		core.HandleDatagram(now, listingHello(id, listingRunID(i, 0), fmt.Sprintf("fill-%02d", i)), &out)
	}
	out.Drain()
	if got := len(core.listingByDst); got != maxListingDestinations {
		t.Fatalf("%d targeted triggers left %d in-flight FIFOs, want %d", maxListingDestinations, got, maxListingDestinations)
	}

	// The next periodic tick must admit the broadcast despite the full
	// destination table. The jump clears the sync interval plus the timer's
	// positive-only jitter (at most a tenth of the interval).
	broadcastAt := now.Add(defaultSyncInterval + defaultSyncInterval/jitterDivisor)
	core.Advance(broadcastAt, &out)
	if _, admitted := core.listingByDst[broadcastDst()]; !admitted {
		t.Fatalf("the periodic broadcast round was refused with %d targeted FIFOs in flight (ListingDropped %d) — a listing flood must never starve the periodic round (WR-01)",
			len(core.listingByDst), core.ListingDropped())
	}
	if got := core.ListingDropped(); got != 0 {
		t.Fatalf("ListingDropped() = %d after the periodic trigger under the destination cap, want 0 — the broadcast is exempt", got)
	}

	// Drain priority: the pacer's next slot emits the broadcast round, never a
	// targeted unicast queued ahead of it.
	first := dataSyncEmission(t, out.Drain())
	if !first.Broadcast {
		t.Fatalf("the first listing emission after the periodic trigger is a targeted unicast to %q, want the broadcast round to drain first (WR-01)", first.To)
	}
}

// dataSyncEmission returns the first DataSync message among drained,
// failing the test when the batch carries none.
func dataSyncEmission(t *testing.T, drained []port.Outbound) port.Outbound {
	t.Helper()
	for _, ob := range drained {
		if _, ok := ob.Msg.(port.DataSync); ok {
			return ob
		}
	}
	t.Fatal("the drained batch carries no DataSync emission")
	return port.Outbound{}
}

// TestListing_refusedFirstSeenIsRearmed proves the WR-01 re-arm: a first-seen
// Hello refused by the destination cap is kept (not lost for the run) and the
// refused peer still receives its complete targeted listing once the drain
// frees a slot — even though its next Hello would take the repeat branch and
// never ask again.
func TestListing_refusedFirstSeenIsRearmed(t *testing.T) {
	t.Parallel()
	core, _, now := newListingCore(t, Config{SyncChunkBytes: listingTestBudget}, listingTestChunks)
	var out port.Outbox
	for i := range maxListingDestinations {
		id := uint16(i + 2)
		core.HandleDatagram(now, listingHello(id, listingRunID(i, 0), fmt.Sprintf("fill-%02d", i)), &out)
	}
	out.Drain()

	const refusedID = uint16(maxListingDestinations + 2)
	const refusedHandle = "late-joiner"
	core.HandleDatagram(now, listingHello(refusedID, listingRunID(99, 0), refusedHandle), &out)
	out.Drain()
	if got := core.ListingDropped(); got != 1 {
		t.Fatalf("ListingDropped() = %d after the cap-refused first-seen Hello, want 1", got)
	}
	if got := len(core.listingOwed); got != 1 {
		t.Fatalf("the refused first-seen listing left %d owed entries, want 1 — a refusal must be re-armed, not lost (WR-01)", got)
	}
	if core.listingByDst[listingDst{nodeID: refusedID, runID: listingRunID(99, 0)}] != nil {
		t.Fatal("the refused first-seen listing was admitted anyway, want it waiting for capacity")
	}

	drained, _ := drainListingQueues(core, now)
	got := listingSyncsTo(drained, refusedHandle)
	if len(got) != listingTestChunks {
		t.Fatalf("the re-armed first-seen listing delivered %d chunks to %q, want the full %d-chunk round (WR-01)",
			len(got), refusedHandle, listingTestChunks)
	}
	if indices := listingChunkIndices(got); !slices.Equal(indices, []uint32{1, 2, 3, 4}) {
		t.Fatalf("the re-armed round emitted chunk indices %v, want the complete 1..%d sequence", indices, listingTestChunks)
	}
	if got := len(core.listingOwed); got != 0 {
		t.Fatalf("%d owed listings survived the drain, want 0", got)
	}
}

// listingSyncsTo returns every unicast DataSync chunk of drained addressed to
// handle, skipping the other destinations' chunks — the per-destination
// filter the multi-destination re-arm test needs.
func listingSyncsTo(drained []port.Outbound, handle string) []port.DataSync {
	var syncs []port.DataSync
	for _, ob := range drained {
		if ob.Broadcast || ob.To != handle {
			continue
		}
		if msg, ok := ob.Msg.(port.DataSync); ok {
			syncs = append(syncs, msg)
		}
	}
	return syncs
}
