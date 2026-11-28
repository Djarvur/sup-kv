package repl

// Config normalization's datagram-ceiling bound (WR-06): the chunk-budget
// normalizer must never keep a SyncChunkBytes above the UDP payload ceiling,
// in-domain values must pass through untouched, and the chunker must size each
// chunk from the entries it can actually append, never from byte budget alone.
// The tests live in package repl because cfg is unexported state.

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// newConfigProbeCore builds a core over a real store at the fixed virtual
// instant with the caller's Config — the normalized Config is what the tests
// read back.
func newConfigProbeCore(cfg Config) *Core {
	fixedNow := func() time.Time { return time.UnixMilli(listedTestNowMs) }
	st := &listedStateStore{store: storage.NewStore(listedTestLocal, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})}
	return New(listedTestLocal, [16]byte{1}, st, cfg)
}

// TestConfig_syncChunkBytesClamped pins the WR-06 normalizer: an
// out-of-range budget clamps to the datagram ceiling (never left describing an
// unsendable chunk the cached listing build would pin), an in-domain value
// passes through, and a non-positive value keeps the design default.
func TestConfig_syncChunkBytesClamped(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name string
		in   int
		want int
	}{
		{"above the ceiling clamps", MaxSyncChunkBytes + 1, MaxSyncChunkBytes},
		{"a three-digit-millions budget clamps", 1_000_000_000, MaxSyncChunkBytes},
		{"exactly the ceiling passes", MaxSyncChunkBytes, MaxSyncChunkBytes},
		{"an in-domain budget passes", 1400, 1400},
		{"zero keeps the design default", 0, defaultSyncChunkBytes},
		{"negative keeps the design default", -1, defaultSyncChunkBytes},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			core := newConfigProbeCore(Config{SyncChunkBytes: row.in})
			if got := core.cfg.SyncChunkBytes; got != row.want {
				t.Fatalf("Config.SyncChunkBytes = %d normalized to %d, want %d (WR-06)", row.in, got, row.want)
			}
		})
	}
}

// TestChunkSyncEntries_capacityFollowsEntries proves the WR-06 chunker half:
// with a budget far larger than the listing, the chunk capacity hint follows
// the entries actually present rather than the byte budget, so a huge (now
// clamped) budget can never pre-allocate an unfillable slice. The observable
// contract is the chunk shape and capacity: every entry still lands in one
// chunk, in order, each chunk's capacity is bounded by the entries remaining
// at its creation, and a closed chunk's backing array is never reused.
func TestChunkSyncEntries_capacityFollowsEntries(t *testing.T) {
	t.Parallel()
	t.Run("the capacity hint follows the listing under a maximal budget", func(t *testing.T) {
		t.Parallel()
		entries := make([]port.SyncEntry, 0, 3)
		for i := range 3 {
			entries = append(entries, port.SyncEntry{
				Key:     string(rune('a' + i)),
				Version: version.PackVersion(listedTestNowMs, uint16(i), listedTestPeer),
			})
		}
		chunks := chunkSyncEntries(entries, MaxSyncChunkBytes)
		if len(chunks) != 1 || len(chunks[0]) != len(entries) {
			t.Fatalf("a %d-entry listing under a maximal budget chunked as %d chunks of %d entries, want one chunk holding every entry",
				len(entries), len(chunks), len(chunks[0]))
		}
		if got, want := cap(chunks[0]), len(entries); got != want {
			t.Fatalf("cap(chunks[0]) = %d, want %d (the entry count) — the byte-budget sizing would report %d instead",
				got, want, MaxSyncChunkBytes/entryOverheadBytes)
		}
		for i, entry := range chunks[0] {
			if entry.Key != entries[i].Key {
				t.Fatalf("chunk entry %d = %q, want %q — the walk order must be preserved", i, entry.Key, entries[i].Key)
			}
		}
	})
	t.Run("a multi-chunk listing never pre-allocates past the entries remaining", func(t *testing.T) {
		t.Parallel()
		const entryCount = 6
		// One single-character key's conservative estimate; the budget covers
		// the fixed chunk overhead plus exactly two such entries, so every
		// chunk holds two and closes at the boundary.
		entrySize := len("a") + entryOverheadBytes
		budget := chunkOverheadBytes + entrySize + entrySize
		perChunk := (budget - chunkOverheadBytes) / entrySize
		entries := make([]port.SyncEntry, 0, entryCount)
		for i := range entryCount {
			entries = append(entries, port.SyncEntry{
				Key:     string(rune('a' + i)),
				Version: version.PackVersion(listedTestNowMs, uint16(i), listedTestPeer),
			})
		}
		chunks := chunkSyncEntries(entries, budget)
		if wantChunks := len(entries) / perChunk; len(chunks) != wantChunks {
			t.Fatalf("a %d-entry listing with a two-entry budget chunked as %d chunks of %d entries, want %d chunks",
				len(entries), len(chunks), len(chunks[0]), wantChunks)
		}
		// A closed chunk's slice must never be reused through cur[:0], or every
		// later chunk aliases its predecessor's memory and the earlier chunk's
		// records mutate with it.
		for i := 0; i+1 < len(chunks); i++ {
			if len(chunks[i]) == 0 || len(chunks[i+1]) == 0 {
				continue
			}
			if &chunks[i][0] == &chunks[i+1][0] {
				t.Fatalf("chunks %d and %d share a backing array at %p — a closed chunk must reset to nil, not cur[:0]",
					i, i+1, &chunks[i][0])
			}
		}
		start := 0
		for i, chunk := range chunks {
			if len(chunk) != perChunk {
				t.Fatalf("chunk %d holds %d entries, want %d — the budget closes at two single-character entries",
					i, len(chunk), perChunk)
			}
			remaining := len(entries) - start
			if got := cap(chunk); got > remaining {
				t.Fatalf("cap(chunks[%d]) = %d, want at most the %d entries remaining at its creation — a byte-budget-sized hint would pre-allocate %d",
					i, got, remaining, budget/entryOverheadBytes)
			}
			if got := cap(chunk); got < len(chunk) {
				t.Fatalf("cap(chunks[%d]) = %d, want at least its %d entries", i, got, len(chunk))
			}
			start += len(chunk)
		}
	})
	t.Run("an empty listing yields one empty chunk", func(t *testing.T) {
		t.Parallel()
		chunks := chunkSyncEntries(nil, MaxSyncChunkBytes)
		if len(chunks) != 1 {
			t.Fatalf("an empty listing chunked as %d chunks, want one empty chunk", len(chunks))
		}
		if len(chunks[0]) != 0 {
			t.Fatalf("the empty listing's chunk holds %d entries, want 0", len(chunks[0]))
		}
	})
}
