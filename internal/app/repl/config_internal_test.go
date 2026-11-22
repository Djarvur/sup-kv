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
// contract is the chunk shape: every entry still lands in one chunk, in order.
func TestChunkSyncEntries_capacityFollowsEntries(t *testing.T) {
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
		t.Fatalf("a %d-entry listing under a maximal budget chunked as %d chunks of %d/%d/%d entries, want one chunk holding every entry",
			len(entries), len(chunks), len(chunks[0]), len(chunks[len(chunks)-1]), len(entries))
	}
	for i, entry := range chunks[0] {
		if entry.Key != entries[i].Key {
			t.Fatalf("chunk entry %d = %q, want %q — the walk order must be preserved", i, entry.Key, entries[i].Key)
		}
	}
}
