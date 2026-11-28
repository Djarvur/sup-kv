package grpcapi

import (
	"bytes"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// replaySeamRecordBytes is the replayed value width the seam must not copy:
// the review's 20 000 x 61 440-byte load paid ~2.4 GB of transient garbage
// for the two post-report clones (04 New-3), so the gate measures the same
// record size.
const replaySeamRecordBytes = 60 * 1024

// TestReplaySeamCarriesVersionOnlyRecord pins the paced replay seam's record
// shape (PERF-05, 04 New-3): a snapshot load hands the seam a version-only
// record, because Core.NotifyReplayedWrite reads only rec.Version and the
// paced record-send drain re-reads the stored record through Lookup before
// emitting. Rebuilding Data/Deleted/Checksum here is pure garbage — and it
// silently promises a payload the core never consumes. A failure names the
// field, so the next edit that starts reading Data fails loudly (T-08-10).
func TestReplaySeamCarriesVersionOnlyRecord(t *testing.T) {
	t.Parallel()

	const key = "replay/seam-key"
	replayed := version.PackVersion(1_700_000_000_000, 7, 3)

	var (
		called      bool
		capturedKey string
		captured    storage.Record
	)
	store := storage.NewStore(7, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
	engine := persist.New(store, discardDumps{}, persist.Config{})
	srv := NewServer(engine, nil, nil, ServerOptions{
		ReplayNotify: func(k string, rec storage.Record) error {
			called = true
			capturedKey = k
			captured = rec
			return nil
		},
	})

	srv.notifyReplayedWrite(persist.RestoredRecord{
		Key:     key,
		Version: replayed,
		Data:    bytes.Repeat([]byte("d"), replaySeamRecordBytes),
		Deleted: true,
	})

	if !called {
		t.Fatal("notifyReplayedWrite did not call the paced ReplayNotify seam")
	}
	if capturedKey != key {
		t.Fatalf("replay seam key = %q, want %q", capturedKey, key)
	}
	if captured.Version != replayed {
		t.Fatalf("replay seam Version = %v, want %v", captured.Version, replayed)
	}
	if got := len(captured.Data); got != 0 {
		t.Fatalf("replay seam Data = %d bytes, want 0: the version-only seam must not rebuild the replayed record (the core reads only Version and the drain re-reads via Lookup)",
			got)
	}
	if captured.Deleted {
		t.Fatal("replay seam Deleted = true, want false: the version-only seam carries no state beyond the version")
	}
	if captured.Checksum != 0 {
		t.Fatalf("replay seam Checksum = %d, want 0: the version-only seam carries no checksum", captured.Checksum)
	}
}
