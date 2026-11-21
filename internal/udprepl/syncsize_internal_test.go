package udprepl

// White-box on purpose (the *_internal_test.go filename is the configured
// testpackage exemption, the dumpfile_internal_test.go precedent): the
// wire-budget proof drives the repl core's startup DataSync and encodes
// every emitted chunk through this package's unexported encodeDatagram —
// the cross-seam check that the core's conservative chunk-size estimate
// stays honest against the real protobuf codec (D-01: the core never
// imports protobuf, so it budgets chunks by estimate constants; a size
// regression here means the estimate constants, never the codec, give).

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Sync-budget test constants: the fixed wall-clock millisecond (the D-02
// seam), the wired node, the design-default chunk budget (docs/IDEA.md
// §Parameters), and the oversized-key length whose own core-side estimate
// (len+24+64) exceeds the budget.
const (
	wireNowMs       = int64(1730000000000)
	wireNodeID      = uint16(7)
	wireBudget      = 1400
	wireOversizeLen = 1400

	// wireListingStep is the multi-chunk drain's virtual step: above the
	// jittered listing spacing at the 500/s default (2ms + 10%), so each
	// Advance crosses exactly one listing pacer slot.
	wireListingStep = 3 * time.Millisecond
)

// wireGraduatedLens are the eight graduated key lengths whose per-entry
// estimates (len+24) sum to exactly 1336: with the 64-byte chunk overhead
// the first emitted chunk closes at exactly the 1400-byte budget — the
// tightest chunk the estimator can build, and therefore the one whose real
// encoded size proves the estimate's slack.
var wireGraduatedLens = [8]int{100, 130, 160, 140, 120, 150, 170, 174}

// wireStateStore adapts *storage.Store to repl.StateStore — the Phase 4
// consumer seam — by delegating the core's four methods.
type wireStateStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s wireStateStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Version returns the stored version for key, tombstones included.
func (s wireStateStore) Version(key string) (version.RecordVersion, bool) {
	return s.store.Version(key)
}

// Lookup returns the stored record for key, tombstones included.
func (s wireStateStore) Lookup(key string) (storage.Record, bool) { return s.store.Lookup(key) }

// Versions returns the shallow key→version view, tombstones included.
func (s wireStateStore) Versions() map[string]version.RecordVersion { return s.store.Versions() }

// wireGraduatedKey returns the i-th sorted graduated key of keyLen bytes
// (the repl-side chunking test's construction).
func wireGraduatedKey(i, keyLen int) string {
	prefix := fmt.Sprintf("k%d/", i)
	return prefix + strings.Repeat("x", keyLen-len(prefix))
}

// wireOversizedKey returns the sorted-last key whose own estimate already
// exceeds the budget — it must ship alone as one oversized datagram.
func wireOversizedKey() string {
	return "z" + strings.Repeat("y", wireOversizeLen-1)
}

// TestSync_chunkWireBudget proves REPL-05's size guarantee cross-seam: the
// core chunks its listing by conservative estimate constants because
// protobuf is adapter-side (D-01), so the proof that every budgeted
// chunk's REAL encoded Envelope stays within sync_chunk_bytes lives here,
// against encodeDatagram. The store straddles the budget exactly the way
// the repl-side chunking test arranges: an at-budget chunk (estimates sum
// to exactly 1400), a follow-up chunk, and one oversized single-entry
// chunk — which exceeds the budget by its own unavoidable minimum, one
// datagram, never split, never dropped (T-04-08).
func TestSync_chunkWireBudget(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(wireNowMs)
	fixedNow := func() time.Time { return now }
	st := &wireStateStore{store: storage.NewStore(wireNodeID, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})}
	for i, keyLen := range wireGraduatedLens {
		_, _, err := st.store.Set(wireGraduatedKey(i, keyLen), []byte("v"))
		if err != nil {
			t.Fatalf("store Set(graduated key %d): %v", i, err)
		}
	}
	_, _, err := st.store.Set(wireGraduatedKey(8, 50), []byte("v"))
	if err != nil {
		t.Fatalf("store Set(graduated key 8): %v", err)
	}
	_, _, err = st.store.Set(wireGraduatedKey(9, 50), []byte("v"))
	if err != nil {
		t.Fatalf("store Set(graduated key 9): %v", err)
	}
	_, _, err = st.store.Set(wireOversizedKey(), []byte("v"))
	if err != nil {
		t.Fatalf("store Set(oversized key): %v", err)
	}

	core := repl.New(wireNodeID, repl.NewRunID(), st, repl.Config{})
	var out port.Outbox
	var syncs []port.DataSync
	collect := func(drained []port.Outbound) {
		for _, ob := range drained {
			if !ob.Broadcast {
				continue
			}
			if msg, ok := ob.Msg.(port.DataSync); ok {
				syncs = append(syncs, msg)
			}
		}
	}
	// The listing pacer emits one datagram per interval (D-01), so the
	// three-chunk startup round completes across successive Advances.
	advance := now
	core.Advance(advance, &out)
	collect(out.Drain())
	for core.QueuedSends() > 0 {
		advance = advance.Add(wireListingStep)
		core.Advance(advance, &out)
		collect(out.Drain())
	}
	if len(syncs) != 3 {
		t.Fatalf("the startup sync emitted %d chunks, want 3 (at-budget, follow-up, oversized)", len(syncs))
	}

	runID := repl.NewRunID()
	sawOversized := false
	for i, sync := range syncs {
		datagram, err := encodeDatagram(wireNodeID, runID, sync)
		if err != nil {
			t.Fatalf("encode chunk %d: %v", i, err)
		}
		if len(sync.Entries) == 1 && sync.Entries[0].Key == wireOversizedKey() {
			sawOversized = true
			// One datagram carrying exactly the one oversized entry — its
			// own unavoidable minimum: re-encoding that single-entry
			// DataSync directly must yield the identical size (nothing
			// piggybacked, nothing split off), and the budget is exceeded
			// by exactly the entry's own excess.
			minimum, err := encodeDatagram(wireNodeID, runID, port.DataSync{Entries: sync.Entries})
			if err != nil {
				t.Fatalf("encode the oversized single-entry chunk: %v", err)
			}
			if len(datagram) != len(minimum) {
				t.Fatalf("the oversized chunk encoded to %d bytes, want its unavoidable minimum %d", len(datagram), len(minimum))
			}
			if len(datagram) <= wireBudget {
				t.Fatalf("the %d-byte oversized test key produced a %d-byte chunk, want one exceeding the %d-byte budget", wireOversizeLen, len(datagram), wireBudget)
			}
			continue
		}
		if len(datagram) > wireBudget {
			t.Fatalf("chunk %d encoded to %d bytes, want at most the %d-byte sync_chunk_bytes — the core's estimate constants are no longer conservative", i, len(datagram), wireBudget)
		}
	}
	if !sawOversized {
		t.Fatalf("no single-entry oversized chunk among %v — the oversized key was dropped or split", syncs)
	}
}
