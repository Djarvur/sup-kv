package repl_test

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Shared sync-test constants (the retransmit-test precedent, same test
// package): the egress limiter's per-response spacing at the normalized
// default rate — ⌊1s/500⌋ = 2ms, docs/IDEA.md §Parameters — the key no
// test store ever holds (REPL-04's silence row), the design-default sync
// interval (IDEA §Parameters), and the oversized-key length whose own
// estimate (len+24+64) exceeds the 1400-byte chunk budget.
const (
	responseSpacing = 2 * time.Millisecond
	keyUnknown      = "cfg/never-written"
	syncInterval    = 30 * time.Second
	oversizedKeyLen = 1400
)

// graduatedKeyLens are the eight graduated key lengths whose per-entry
// estimates (len+24) sum to exactly 1336: with the 64-byte chunk overhead
// the first chunk closes at exactly the 1400-byte budget (probe #15's
// at-boundary case).
var graduatedKeyLens = [8]int{100, 130, 160, 140, 120, 150, 170, 174}

// requestFrom builds a decoded DataRequest datagram from peerID (REPL-04:
// unicast "send me this key").
func requestFrom(peerID uint16, key string) port.InboundDatagram {
	return datagramFrom(peerID, port.DataRequest{Key: key})
}

// unicasts returns every unicast outbound message in drained in emission
// order — broadcast emissions (Hello, DataSync, retransmit
// DataReplication) are filtered out, leaving exactly the reactive unicast
// traffic the egress queue drains.
func unicasts(drained []port.Outbound) []port.Outbound {
	out := make([]port.Outbound, 0, len(drained))
	for _, ob := range drained {
		if !ob.Broadcast {
			out = append(out, ob)
		}
	}
	return out
}

// recordsEqual reports whether a and b are byte-equal in every record field
// — Version, Data, Deleted and Checksum — the REPL-04 exact-record reply
// contract.
func recordsEqual(a, b storage.Record) bool {
	return a.Version == b.Version && a.Deleted == b.Deleted &&
		a.Checksum == b.Checksum && bytes.Equal(a.Data, b.Data)
}

// TestDataRequest_replyAndSilence proves REPL-04 through the egress queue:
// a unicast DataRequest for a key the store holds — live or tombstoned, the
// delete-healing pin — is answered by one unicast DataReplication carrying
// the exact snapshot record; a DataRequest for an absent key enqueues and
// emits nothing, ever. The requester is never Hello'd first: a stranger's
// DataRequest is still protocol traffic from a live node, so the reply goes
// to the datagram's source handle regardless of peer-table membership. One
// Records() snapshot serves each handler call — the discipline the Task 2/3
// sync paths reuse.
func TestDataRequest_replyAndSilence(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"a known live key is answered with one unicast exact-record DataReplication", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.Advance(now, &out) // startup Hello (plan 04-01)
			out.Drain()
			_, _, err := st.store.Set(keyAlpha, []byte("live value"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyAlpha, err)
			}
			want := st.Records()[keyAlpha]

			core.HandleDatagram(now, requestFrom(peerID1, keyAlpha), &out)
			if got := len(out.Drain()); got != 0 {
				t.Fatalf("the DataRequest handler emitted %d messages directly, want 0 — replies leave only through the egress queue", got)
			}
			if got := core.QueuedResponses(); got != 1 {
				t.Fatalf("after the DataRequest QueuedResponses() = %d, want 1", got)
			}
			core.Advance(now, &out)
			drained := unicasts(out.Drain())
			if len(drained) != 1 || drained[0].To != handleFor(peerID1) {
				t.Fatalf("the reply Advance drained %v, want exactly one unicast to the requester handle %q", drained, handleFor(peerID1))
			}
			reply, ok := drained[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the reply is a %T, want port.DataReplication", drained[0].Msg)
			}
			if reply.Key != keyAlpha {
				t.Fatalf("reply key = %q, want %q", reply.Key, keyAlpha)
			}
			if !recordsEqual(reply.Record, want) {
				t.Fatalf("reply record = %+v, want the exact snapshot record %+v", reply.Record, want)
			}
			if got := core.QueuedResponses(); got != 0 {
				t.Fatalf("after the drain QueuedResponses() = %d, want 0", got)
			}
		}},
		{"a tombstoned key replies with the tombstone — deletes heal through reconciliation", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.Advance(now, &out)
			out.Drain()
			_, _, err := st.store.Set(keyBeta, []byte("doomed"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyBeta, err)
			}
			_, _, err = st.store.Delete(keyBeta)
			if err != nil {
				t.Fatalf("store Delete(%q): %v", keyBeta, err)
			}
			want := st.Records()[keyBeta]
			if !want.Deleted {
				t.Fatalf("test setup: key %q is not tombstoned in the snapshot", keyBeta)
			}

			core.HandleDatagram(now, requestFrom(peerID1, keyBeta), &out)
			out.Drain()
			if got := core.QueuedResponses(); got != 1 {
				t.Fatalf("after the DataRequest for the tombstoned key QueuedResponses() = %d, want 1", got)
			}
			core.Advance(now, &out)
			drained := unicasts(out.Drain())
			if len(drained) != 1 || drained[0].To != handleFor(peerID1) {
				t.Fatalf("the tombstone reply Advance drained %v, want exactly one unicast to %q", drained, handleFor(peerID1))
			}
			reply, ok := drained[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the tombstone reply is a %T, want port.DataReplication", drained[0].Msg)
			}
			if reply.Key != keyBeta || !recordsEqual(reply.Record, want) {
				t.Fatalf("tombstone reply = (%q, %+v), want the exact tombstone record (%q, %+v)", reply.Key, reply.Record, keyBeta, want)
			}
			if !reply.Record.Deleted {
				t.Fatal("reply record is live, want the tombstone (Deleted true)")
			}
		}},
		{"an unknown key stays silent — no enqueue, no emission, ever", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, _ := newTestCore()
			var out port.Outbox
			core.Advance(now, &out)
			out.Drain()

			// A second requester peer exercises the parameter both ways:
			// silence holds regardless of who asks.
			core.HandleDatagram(now, requestFrom(peerID2, keyUnknown), &out)
			if got := len(out.Drain()); got != 0 {
				t.Fatalf("the unknown-key DataRequest emitted %d messages directly, want 0", got)
			}
			if got := core.QueuedResponses(); got != 0 {
				t.Fatalf("a DataRequest for the unknown key %q enqueued %d responses, want 0", keyUnknown, got)
			}
			// Drive Advance well past a full rate window — five times the
			// 2ms per-response spacing: nothing may ever emit, no deferred
			// surprise (REPL-04's silence contract).
			for off := responseSpacing; off <= 5*responseSpacing; off += responseSpacing {
				core.Advance(now.Add(off), &out)
				if got := len(out.Drain()); got != 0 {
					t.Fatalf("Advance at t0+%v emitted %d messages after an unknown-key DataRequest, want 0", off, got)
				}
				if got := core.QueuedResponses(); got != 0 {
					t.Fatalf("QueuedResponses() = %d at t0+%v, want 0", got, off)
				}
			}
		}},
		{"two DataRequests in one event share the queue and drain in request order", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.Advance(now, &out)
			out.Drain()
			_, _, err := st.store.Set(keyAlpha, []byte("a"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyAlpha, err)
			}
			_, _, err = st.store.Set(keyBeta, []byte("b"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyBeta, err)
			}
			snap := st.Records()

			core.HandleDatagram(now, requestFrom(peerID1, keyBeta), &out)
			core.HandleDatagram(now, requestFrom(peerID1, keyAlpha), &out)
			out.Drain()
			if got := core.QueuedResponses(); got != 2 {
				t.Fatalf("after two DataRequests QueuedResponses() = %d, want 2", got)
			}

			// The first Advance drains the first request's reply immediately;
			// the second follows one spacing later. The limiter's exact
			// cadence is Task 3's assertion — here both replies must simply
			// emit, in request order, nothing more.
			core.Advance(now, &out)
			first := unicasts(out.Drain())
			core.Advance(now.Add(responseSpacing), &out)
			second := unicasts(out.Drain())
			if len(first) != 1 || len(second) != 1 {
				t.Fatalf("the two requests drained as %v then %v, want one unicast reply per Advance", first, second)
			}
			r1, ok := first[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the first reply is a %T, want port.DataReplication", first[0].Msg)
			}
			r2, ok := second[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the second reply is a %T, want port.DataReplication", second[0].Msg)
			}
			if r1.Key != keyBeta || !recordsEqual(r1.Record, snap[keyBeta]) {
				t.Fatalf("the first reply = (%q, %+v), want the exact record under %q", r1.Key, r1.Record, keyBeta)
			}
			if r2.Key != keyAlpha || !recordsEqual(r2.Record, snap[keyAlpha]) {
				t.Fatalf("the second reply = (%q, %+v), want the exact record under %q", r2.Key, r2.Record, keyAlpha)
			}
			if first[0].To != handleFor(peerID1) || second[0].To != handleFor(peerID1) {
				t.Fatalf("reply handles = %q, %q, want both unicast to %q", first[0].To, second[0].To, handleFor(peerID1))
			}
			if got := core.QueuedResponses(); got != 0 {
				t.Fatalf("after both drains QueuedResponses() = %d, want 0", got)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			t.Parallel()
			tt.run(t)
		})
	}
}

// graduatedKey returns the i-th sorted graduated key of keyLen bytes: the
// fixed numeric prefix keeps byte order equal to index order, the x filler
// reaches the length the estimate math needs.
func graduatedKey(i, keyLen int) string {
	prefix := fmt.Sprintf("k%d/", i)
	return prefix + strings.Repeat("x", keyLen-len(prefix))
}

// oversizedKey returns the sorted-last key whose own estimate
// (len+24+64 = 1488) already exceeds the 1400-byte budget — it must ship
// alone as one oversized chunk, never dropped, never split.
func oversizedKey() string {
	return "z" + strings.Repeat("y", oversizedKeyLen-1)
}

// dataSyncs returns every broadcast DataSync in drained, in emission order
// — the chunk-sequence probe (Hello and retransmit broadcasts filter out).
func dataSyncs(drained []port.Outbound) []port.DataSync {
	var syncs []port.DataSync
	for _, ob := range drained {
		if !ob.Broadcast {
			continue
		}
		if msg, ok := ob.Msg.(port.DataSync); ok {
			syncs = append(syncs, msg)
		}
	}
	return syncs
}

// allSyncEntries concatenates the chunks' entries in chunk order.
func allSyncEntries(syncs []port.DataSync) []port.SyncEntry {
	var entries []port.SyncEntry
	for _, sync := range syncs {
		entries = append(entries, sync.Entries...)
	}
	return entries
}

// syncEntryKeys returns the chunk's keys in order.
func syncEntryKeys(sync port.DataSync) []string {
	keys := make([]string, 0, len(sync.Entries))
	for _, entry := range sync.Entries {
		keys = append(keys, entry.Key)
	}
	return keys
}

// dataSyncsEqual reports whether two chunk sequences carry identical
// entries in identical chunk order — the determinism probe.
func dataSyncsEqual(a, b []port.DataSync) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !slices.Equal(a[i].Entries, b[i].Entries) {
			return false
		}
	}
	return true
}

// listingFrom builds the expected sorted listing from a snapshot: every
// record including tombstones as {Key, Version}, sorted by key in byte
// order — the construction the core implements (probe #11).
func listingFrom(snap map[string]storage.Record) []port.SyncEntry {
	entries := make([]port.SyncEntry, 0, len(snap))
	for _, key := range slices.Sorted(maps.Keys(snap)) {
		entries = append(entries, port.SyncEntry{Key: key, Version: snap[key].Version})
	}
	return entries
}

// TestSync_chunking proves the DataSync listing and chunking determinism
// (REPL-05, probes #11 and #15): the listing is every record sorted by key
// in byte order, packed into chunks whose conservative estimates stay
// within sync_chunk_bytes — the chunk filled to exactly the budget closes,
// the entry that would cross it opens the next chunk, and a single entry
// whose own estimate exceeds the budget ships alone as one oversized
// chunk, never dropped, never split (A4: chunks are self-sufficient, no
// continuation state). An empty store still emits one zero-entry listing
// (probe #7: an empty listing is a valid sync round).
func TestSync_chunking(t *testing.T) {
	t.Parallel()
	t.Run("exact-budget close, boundary crossing and the oversized single entry", func(t *testing.T) {
		t.Helper()
		t.Parallel()
		now0 := at(0)
		core, st := newTestCore()
		var out port.Outbox
		for i, keyLen := range graduatedKeyLens {
			_, _, err := st.store.Set(graduatedKey(i, keyLen), []byte("v"))
			if err != nil {
				t.Fatalf("store Set(graduated key %d): %v", i, err)
			}
		}
		_, _, err := st.store.Set(graduatedKey(8, 50), []byte("v"))
		if err != nil {
			t.Fatalf("store Set(graduated key 8): %v", err)
		}
		_, _, err = st.store.Set(graduatedKey(9, 50), []byte("v"))
		if err != nil {
			t.Fatalf("store Set(graduated key 9): %v", err)
		}
		_, _, err = st.store.Set(oversizedKey(), []byte("v"))
		if err != nil {
			t.Fatalf("store Set(oversized key): %v", err)
		}
		want := listingFrom(st.Records())

		core.Advance(now0, &out)
		syncs := dataSyncs(out.Drain())
		if len(syncs) != 3 {
			t.Fatalf("startup sync emitted %d chunks, want 3 (at-budget, follow-up, oversized)", len(syncs))
		}
		wantFirst := make([]string, 0, len(graduatedKeyLens))
		for i := range graduatedKeyLens {
			wantFirst = append(wantFirst, graduatedKey(i, graduatedKeyLens[i]))
		}
		if got := syncEntryKeys(syncs[0]); !slices.Equal(got, wantFirst) {
			t.Fatalf("the at-budget chunk holds %d keys, want exactly the first %d graduated keys (64 + sum(len+24) = 1400)", len(got), len(wantFirst))
		}
		if got := syncEntryKeys(syncs[1]); !slices.Equal(got, []string{graduatedKey(8, 50), graduatedKey(9, 50)}) {
			t.Fatalf("the follow-up chunk holds %v, want the two keys that crossed the budget boundary", got)
		}
		if got := syncEntryKeys(syncs[2]); !slices.Equal(got, []string{oversizedKey()}) {
			t.Fatalf("the oversized chunk holds %v, want exactly the one oversized key shipping alone", got)
		}
		if got := allSyncEntries(syncs); !slices.Equal(got, want) {
			t.Fatalf("concatenated chunks hold %d entries, want the full sorted listing of %d — every key in exactly one chunk", len(got), len(want))
		}
	})
	t.Run("two independent builds over the same store produce identical chunk sequences", func(t *testing.T) {
		t.Helper()
		t.Parallel()
		now0 := at(0)
		core1, st := newTestCore()
		core2 := repl.New(localID, testRunID(localID), handleFor(localID), st, repl.Config{})
		var out port.Outbox
		for i, keyLen := range graduatedKeyLens {
			_, _, err := st.store.Set(graduatedKey(i, keyLen), []byte("v"))
			if err != nil {
				t.Fatalf("store Set(graduated key %d): %v", i, err)
			}
		}

		core1.Advance(now0, &out)
		first := dataSyncs(out.Drain())
		if len(first) != 1 {
			t.Fatalf("the first build emitted %d chunks, want 1 over the eight graduated keys", len(first))
		}
		core2.Advance(now0, &out)
		second := dataSyncs(out.Drain())
		if !dataSyncsEqual(first, second) {
			t.Fatalf("two fresh-snapshot builds diverged: %v vs %v — a map-iteration order leaked into the chunk sequence", first, second)
		}
		// The timer rebuild over the same core takes another fresh snapshot
		// sync_interval later and must reproduce the same sequence.
		core1.Advance(now0.Add(syncInterval), &out)
		third := dataSyncs(out.Drain())
		if !dataSyncsEqual(first, third) {
			t.Fatalf("the timer rebuild diverged from the startup build: %v vs %v", first, third)
		}
	})
	t.Run("an empty store still announces itself with one zero-entry DataSync", func(t *testing.T) {
		t.Helper()
		t.Parallel()
		core, _ := newTestCore()
		var out port.Outbox
		core.Advance(at(0), &out)
		syncs := dataSyncs(out.Drain())
		if len(syncs) != 1 || len(syncs[0].Entries) != 0 {
			t.Fatalf("an empty store emitted %d DataSync chunks with entry counts %v, want exactly one zero-entry listing (probe #7)", len(syncs), syncs)
		}
	})
}

// TestSync_triggers proves the DataSync emission triggers (REPL-05): the
// first Advance fires the startup Hello immediately followed by the
// broadcast DataSync chunk sequence — the IDEA §Node lifecycle order,
// pinned here — and the sync timer re-fires on every subsequent Advance at
// or after lastSync+sync_interval, never between ticks.
func TestSync_triggers(t *testing.T) {
	t.Parallel()
	now0 := at(0)
	core, st := newTestCore()
	_, _, err := st.store.Set(keyAlpha, []byte("v"))
	if err != nil {
		t.Fatalf("store Set(%q): %v", keyAlpha, err)
	}
	var out port.Outbox

	core.Advance(now0, &out)
	drained := out.Drain()
	if len(drained) != 2 || !drained[0].Broadcast || !drained[1].Broadcast {
		t.Fatalf("startup Advance drained %v, want exactly two broadcasts: Hello then the DataSync chunk", drained)
	}
	if _, ok := drained[0].Msg.(port.Hello); !ok {
		t.Fatalf("the startup emission order broke: first message is %T, want port.Hello (IDEA §Node lifecycle)", drained[0].Msg)
	}
	if _, ok := drained[1].Msg.(port.DataSync); !ok {
		t.Fatalf("the startup emission order broke: second message is %T, want port.DataSync right after the Hello", drained[1].Msg)
	}

	// Off-tick Advances across the whole interval emit no DataSync (their
	// periodic Hellos are filtered out).
	for _, off := range []time.Duration{time.Second, 10 * time.Second, syncInterval - time.Second} {
		core.Advance(now0.Add(off), &out)
		if syncs := dataSyncs(out.Drain()); len(syncs) != 0 {
			t.Fatalf("Advance at t0+%v emitted %d DataSync messages before the sync interval elapsed, want 0", off, len(syncs))
		}
	}
	// The at-boundary tick — lastSync+sync_interval exactly — fires it.
	core.Advance(now0.Add(syncInterval), &out)
	if syncs := dataSyncs(out.Drain()); len(syncs) != 1 {
		t.Fatalf("the at-boundary Advance at t0+%v emitted %d DataSync messages, want exactly 1", syncInterval, len(syncs))
	}
}
