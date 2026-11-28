package repl

// The WR-05 white-box probe: the partial-listing stream's chunk-index map must
// be sized by the chunks that actually arrive, never by the attacker-supplied
// chunk_count header. The probe measures the retained heap of a full
// maxSyncStreams-wide set of one-chunk rounds under two claims — the clamp's
// own hint and the protocol maximum — and fails when the larger claim retains
// materially more. It lives in package repl because the map is unexported
// state, and it is the _internal_test.go testpackage exemption.

import (
	"fmt"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

const (
	// streamProbeNowMs is the fixed virtual base (the D-02 now-seam idiom).
	streamProbeNowMs = int64(1730000000000)
	// streamProbeLocal and streamProbePeer are the node identities the probe
	// builds its core and sender from.
	streamProbeLocal = uint16(1)
	streamProbePeer  = uint16(2)
	// streamProbeAllowance is the accepted retained-heap difference between
	// the two claims. The pre-fix hint cost ~1.2 MiB per stream (~19 MiB for
	// the 16-stream bound); the clamped hint costs a few KiB, so this bound
	// sits orders of magnitude from both sides and only absorbs allocator and
	// ambient-suite noise.
	streamProbeAllowance = 1 << 20
	// streamProbeSamples is the sample count; each claim's measurement is the
	// minimum over its samples, which suppresses upward noise from tests
	// running in parallel.
	streamProbeSamples = 3
)

// newStreamProbeCore builds a probe core over a real store at the fixed
// virtual instant — the listedStateStore wrapper the union tests share.
func newStreamProbeCore() *Core {
	fixedNow := func() time.Time { return time.UnixMilli(streamProbeNowMs) }
	st := &listedStateStore{store: storage.NewStore(streamProbeLocal, clock.NewClock(fixedNow, 0, 0, 0), storage.Config{})}
	return New(streamProbeLocal, [16]byte{1}, st, Config{})
}

// streamProbeClaim sends the identity Hello and then maxSyncStreams
// one-chunk listing parts that all claim claimChunks chunks, and returns the
// retained heap growth those rounds hold at the end. The streams stay live in
// the returned core; callers keep it alive through the after-reading.
func streamProbeClaim(t *testing.T, claimChunks uint32) (*Core, int64) {
	t.Helper()
	var before, after runtime.MemStats
	//nolint:revive // call-to-gc: a heap-growth measurement needs the garbage from the previous sample collected first.
	runtime.GC()
	runtime.ReadMemStats(&before)

	core := newStreamProbeCore()
	now := time.UnixMilli(streamProbeNowMs)
	listedFeedHello(core, now, streamProbePeer)
	var out port.Outbox
	for i := range maxSyncStreams {
		core.HandleDatagram(now, port.InboundDatagram{
			Source:       fmt.Sprintf("n%d", streamProbePeer),
			Status:       port.Decoded,
			ProtoVersion: SupportedProtoVersion,
			NodeID:       streamProbePeer,
			RunID:        [16]byte{byte(streamProbePeer)},
			Msg: port.DataSync{
				SyncID:     uint64(i + 1),
				ChunkIndex: 1,
				ChunkCount: claimChunks,
				Entries: []port.SyncEntry{{
					Key:     fmt.Sprintf("hint/%02d", i),
					Version: version.PackVersion(streamProbeNowMs, uint16(i), streamProbePeer),
				}},
			},
		}, &out)
		out.Drain()
	}
	if got := len(core.syncs); got != maxSyncStreams {
		t.Fatalf("the probe retained %d partial streams, want %d — the fixture must fill the stream bound", got, maxSyncStreams)
	}

	//nolint:revive // call-to-gc: collect the fixture's garbage so the retained growth is what REMAINS live, not what was allocated.
	runtime.GC()
	runtime.ReadMemStats(&after)
	return core, int64(after.HeapAlloc) - int64(before.HeapAlloc)
}

// TestSync_streamMapHintIndependentOfClaim is WR-05's probe. A datagram is
// only required to carry one valid chunk index, so a claim of 65 536 chunks
// must cost the same retained heap as a claim of the clamp's own hint; only
// the chunks that actually arrive may grow the map.
//
//nolint:paralleltest // the probe reads process-wide heap statistics; parallel neighbours would only add noise to compare against.
func TestSync_streamMapHintIndependentOfClaim(t *testing.T) {
	bestClamped, bestClaimed := int64(math.MaxInt64), int64(math.MaxInt64)
	for range streamProbeSamples {
		clampedCore, clamped := streamProbeClaim(t, syncStreamGotHint)
		claimedCore, claimed := streamProbeClaim(t, maxSyncChunks)
		bestClamped = min(bestClamped, clamped)
		bestClaimed = min(bestClaimed, claimed)
		runtime.KeepAlive(clampedCore)
		runtime.KeepAlive(claimedCore)
	}
	if diff := bestClaimed - bestClamped; diff >= streamProbeAllowance {
		t.Fatalf(
			"16 one-chunk streams claiming %d chunks retained %d more bytes than the same set claiming %d (%d vs %d)"+
				": the chunk-index map hint must never scale with the claimed chunk_count (WR-05)",
			maxSyncChunks, diff, syncStreamGotHint, bestClaimed, bestClamped,
		)
	}
}
