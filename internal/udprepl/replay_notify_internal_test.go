package udprepl

import (
	"bytes"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

const (
	// replayNotifyKey is the replayed key both gates use; the value is
	// arbitrary — only the key's identity and the record's shape matter.
	replayNotifyKey = "replay/version-only"

	// replayNotifyRecordBytes is the replayed value width the adapter hop
	// must not copy: the review's 20 000 x 61 440-byte load paid ~2.4 GB of
	// transient garbage for the discarded copies (04 New-3).
	replayNotifyRecordBytes = 60 * 1024

	// replayNotifyAllocCeiling is the allocation gate's bound: one replay
	// notification may allocate a few struct headers, never the record.
	replayNotifyAllocCeiling = 1024
)

// TestNotifyReplayedWriteCarriesVersionOnly proves the adapter's replay seam
// enqueues a version-only record (PERF-05, 04 New-3): Core.NotifyReplayedWrite
// reads only rec.Version and the paced drain re-reads the stored record
// through Lookup, so carrying Data/Deleted/Checksum through the notify queue
// is a discarded copy. The test reads the real notify channel, so it asserts
// the value the event loop will actually dequeue.
func TestNotifyReplayedWriteCarriesVersionOnly(t *testing.T) {
	t.Parallel()

	adapter := newTrackerTestAdapter(t)
	replayed := version.PackVersion(1_700_000_000_000, 5, 1)
	data := bytes.Repeat([]byte("d"), replayNotifyRecordBytes)
	rec := storage.Record{
		Version:  replayed,
		Data:     data,
		Deleted:  true,
		Checksum: storage.RecordChecksum(replayNotifyKey, replayed, data, true),
	}

	if err := adapter.NotifyReplayedWrite(replayNotifyKey, rec); err != nil {
		t.Fatalf("NotifyReplayedWrite: %v", err)
	}

	var lw localWrite
	select {
	case lw = <-adapter.notify:
	default:
		t.Fatal("NotifyReplayedWrite enqueued nothing on the notify channel")
	}

	if lw.key != replayNotifyKey {
		t.Fatalf("enqueued localWrite key = %q, want %q", lw.key, replayNotifyKey)
	}
	if lw.rec.Version != replayed {
		t.Fatalf("enqueued localWrite Version = %v, want %v", lw.rec.Version, replayed)
	}
	if !lw.replay {
		t.Fatal("enqueued localWrite replay = false, want true: a replay takes the paced record-send path")
	}
	if got := len(lw.rec.Data); got != 0 {
		t.Fatalf("enqueued localWrite Data = %d bytes, want 0: the version-only replay seam must not copy the record it will never read",
			got)
	}
	if lw.rec.Deleted {
		t.Fatal("enqueued localWrite Deleted = true, want false: the version-only replay seam carries no state beyond the version")
	}
	if lw.rec.Checksum != 0 {
		t.Fatalf("enqueued localWrite Checksum = %d, want 0: the version-only replay seam carries no checksum",
			lw.rec.Checksum)
	}
}

// TestNotifyReplayedWriteAllocationBound is the fail-first allocation gate for
// the replay hop (PERF-05, 04 New-3): one NotifyReplayedWrite of a 60 KiB
// record, followed by a drain of the enqueued value, must allocate well under
// the record size. Before the fix the adapter's cloneLocalRecord allocated the
// full value per call, so this gate failed at ~60 KiB/op; it fails by name and
// reports the measured bytes, never an allocation count.
func TestNotifyReplayedWriteAllocationBound(t *testing.T) {
	t.Parallel()

	adapter := newTrackerTestAdapter(t)
	rec := storage.Record{
		Version: version.PackVersion(1_700_000_000_000, 6, 2),
		Data:    bytes.Repeat([]byte("d"), replayNotifyRecordBytes),
	}

	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if err := adapter.NotifyReplayedWrite(replayNotifyKey, rec); err != nil {
				b.Errorf("NotifyReplayedWrite: %v", err)
				return
			}
			// Drain immediately: the 1024-deep channel must never fill, and
			// the enqueued value is exactly what the measurement covers.
			select {
			case <-adapter.notify:
			default:
				b.Error("drain: NotifyReplayedWrite enqueued nothing")
				return
			}
		}
	})

	if got := result.AllocedBytesPerOp(); got >= replayNotifyAllocCeiling {
		t.Fatalf("NotifyReplayedWrite allocates %d bytes/op for a %d-byte record, want under %d: the version-only replay seam must not copy rec.Data",
			got, replayNotifyRecordBytes, replayNotifyAllocCeiling)
	}
}
