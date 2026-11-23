package persist

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// This file is the snapshot-orchestration role (D-10, PERSIST-11): Snapshot
// forces the covering dump through the public flush seam, Restore reuses the
// facade's store write path, and the replay helpers stay pure functions over
// the store.

// Snapshot forces the current dump to complete — every confirmed write is
// on disk — then byte-copies the finished main dump under name through the
// adapter (D-10, PERS-05): the snapshot file is byte-identical to the main
// dump at creation, and a flush triggered while state is dirty forces a new
// dump before the copy. A flush error aborts with nothing copied: state
// that is not confirmed on disk must never become a recovery point. A flush
// requested while the dump-failure ladder has a rung pending answers that
// ladder's last classified failure immediately instead of forcing an extra
// attempt (WR-02), so a snapshot loop under a persistent failure cannot
// bypass the bounded retry rate. The
// flush is bounded by the confirmation timeout (PERSIST-08), so a hung
// fsync answers ErrUnconfirmed rather than parking the call. Safe
// against a dump running concurrently with the copy: the adapter's atomic
// renames mean no torn read, and its save-snapshot serialization closes
// the rotate window in which the main path is briefly absent — the copy
// always reads one complete generation. Requires a running Run goroutine
// like every blocking engine call.
func (e *Engine) Snapshot(name string) error {
	err := e.flush(e.cfg.ConfirmTimeout, flushRequest{snapshot: true})
	if err != nil {
		return err
	}
	return e.dumps.CreateSnapshot(name)
}

// RestoredRecord is one record a Restore actually installed: the key, the
// fresh version the store stamped for it, the record's data and whether the
// install wrote a tombstone. A report covers the whole rollback — both the
// snapshot's own records (re-stamped fresh) and the ROLLBACK tombstones of
// current live keys the snapshot does not name (nil Data, Deleted true) —
// so the notify sequence a snapshot load drives replicates the restored key
// set. Records a replay skips — a tombstone with nothing to beat, a value
// the store's shrunk limit rejects, a live key a concurrent writer already
// deleted — are absent, and the order is the replay's deterministic
// sorted-key order (the sorted union of snapshot keys and rollback keys),
// so the notify sequence a snapshot load drives is stable across runs.
type RestoredRecord struct {
	Key     string
	Version version.RecordVersion
	Data    []byte
	Deleted bool
}

// Restore is a FULL ROLLBACK of the named snapshot (decision 14.1, ADR
// 0001): it replays the snapshot's records as ordinary writes and deletes
// with fresh versions (D-09, PERS-05) — a snapshot-live record is a fresh
// Set that beats any current tombstone, a snapshot tombstone a fresh Delete
// that beats any current live value — and it DELETES every current live key
// the snapshot does not name, stamping a fresh tombstone of its own, so the
// snapshot's key set becomes the store's key set. A key already tombstoned
// locally and absent from the snapshot is already rolled back and stays
// untouched: no version is burned and no record is produced. The snapshot's
// own versions are never re-installed (no Merge); fresh versions are what
// make the restore replicate and win by normal LWW. The replay and the
// rollback walk one deterministic sorted union of snapshot keys and
// rollback keys, so a load's notify sequence is stable. Concurrent writes
// after the rollback set is computed are ordinary LWW, not snapshot
// isolation: a write that lands mid-load is neither suppressed nor
// rolled back beyond the key set the pass computed. A snapshot that already
// equals the store's current state — every record, version and tombstone —
// and leaves no extra live key replays as a whole-restore no-op: the engine
// stays clean, the returned record list is empty and the final flush replies
// nil without a redundant dump (D-15, 03 IN-02 + 04 IN-02); ANY difference
// replays in full, every snapshot record re-stamped fresh, preserving the
// fresh-version dominance the restore replicates and wins by. The replay runs in deterministic
// sorted-key order through the store's write path, and the call returns
// only after one final flush whose dump covers every replayed write — the
// same write-confirmation contract as Set, batched into one flush because
// the per-key semantics are identical while sequential per-write blocking
// would serialize N writes at one debounce each. The returned list names
// exactly the records the replay installed — sorted by key, skipped records
// absent — and is returned even when the confirming dump failed, so a
// snapshot load can replicate what is already in memory (PROTO-08) while
// the error still reports the dump honestly: a failed final dump surfaces
// as ErrDumpFailed carrying its cause, a dump that outlived the confirmation
// timeout as ErrUnconfirmed (the two stay distinguishable), and a closed
// engine as ErrClosed.
// A mid-replay store error aborts deterministically instead: the records
// installed before it stay marked dirty (the dumper persists them) and come
// back with the error, so no installed-but-unreturned record can be lost
// (IN-04) — the caller's snapshot-load notify covers exactly that list.
// Load and decode errors surface unchanged — a missing snapshot or a
// generation-level decode fault — never a silent empty success.
func (e *Engine) Restore(name string) ([]RestoredRecord, error) {
	data, err := e.dumps.ReadSnapshot(name)
	if err != nil {
		return nil, err
	}
	records, _, err := Decode(data, e.valueMaxBytes)
	if err != nil {
		return nil, err
	}
	restored, err := e.replaySnapshot(records)
	if err != nil {
		return restored, err
	}
	err = e.flush(e.cfg.ConfirmTimeout, flushRequest{snapshot: true})
	switch {
	case err == nil:
		return restored, nil
	case errors.Is(err, ErrClosed):
		return restored, err
	case errors.Is(err, ErrUnconfirmed):
		// Already the honest durability answer (PERSIST-08): keep it
		// distinguishable from a failed dump.
		return restored, err
	default:
		return restored, fmt.Errorf("%w: %w", ErrDumpFailed, err)
	}
}

// replaySnapshot walks the sorted union of the snapshot's keys and the
// current live keys the snapshot does not name, in one deterministic pass
// (decision 14.1, ADR 0001). A snapshot key replays through replayRecord
// exactly as before; a current-live-only key is the ROLLBACK delete — the
// store's ordinary Delete stamps a fresh tombstone that beats the live
// value — and a concurrent ErrNotFound (another writer deleted the key
// between the Records read and this delete) is a benign skip that burned no
// version. The pass marks the engine dirty whenever anything was installed
// and returns exactly the records it installed: the replayed snapshot
// records and the rollback tombstones, which is what a snapshot load
// broadcasts. A mid-pass store error aborts the replay deterministically and
// returns the PARTIAL list with the error (IN-04): the records installed
// before the abort are already in memory, so they are marked dirty for the
// next dump and returned so the caller's totality notify covers them — an
// installed-but-unreturned record would be neither dumped nor replicated.
func (e *Engine) replaySnapshot(records map[string]storage.Record) ([]RestoredRecord, error) {
	current := e.store.Records()
	restored := make([]RestoredRecord, 0, len(records))
	if !restoreChanges(current, records) {
		return restored, nil
	}
	for _, key := range replayKeys(records, current) {
		snapRec, inSnapshot := records[key]
		var (
			rec       RestoredRecord
			installed bool
			err       error
		)
		if inSnapshot {
			rec, installed, err = e.replaySnapshotKey(key, snapRec)
		} else {
			rec, installed, err = e.rollbackKey(key)
		}
		if err != nil {
			if len(restored) > 0 {
				e.markDirty()
			}
			return restored, err
		}
		if installed {
			restored = append(restored, rec)
		}
	}
	e.markDirty()
	return restored, nil
}

// replayKeys returns the deterministic sorted union of the snapshot's keys
// and the current LIVE keys the snapshot does not name: the replay set and
// the rollback set of one full-restore pass. An extra current tombstone is
// excluded — it is already rolled back and the pass must not burn a version
// on it.
func replayKeys(snap, current map[string]storage.Record) []string {
	union := make(map[string]struct{}, len(snap)+len(current))
	for key := range snap {
		union[key] = struct{}{}
	}
	for key, rec := range current {
		if !rec.Deleted {
			union[key] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(union))
}

// replaySnapshotKey installs one snapshot record through replayRecord's
// ordinary Set/Delete rules and reports the record the store installed: a
// fresh Set carrying the snapshot's data, or a fresh tombstone. Skipped
// records (a tombstone with nothing to beat, an over-limit value) install
// nothing and are absent from the report.
func (e *Engine) replaySnapshotKey(key string, snapRec storage.Record) (RestoredRecord, bool, error) {
	ver, replayed, err := replayRecord(e.store, key, snapRec)
	if err != nil || !replayed {
		return RestoredRecord{}, false, err
	}
	return RestoredRecord{
		Key:     key,
		Version: ver,
		Data:    bytes.Clone(snapRec.Data),
		Deleted: snapRec.Deleted,
	}, true, nil
}

// rollbackKey deletes one current live key the snapshot does not name and
// reports the fresh tombstone it installed. A concurrent ErrNotFound means
// another writer tombstoned the key first: the rollback outcome already
// holds, Delete burned no version, and nothing is installed.
func (e *Engine) rollbackKey(key string) (RestoredRecord, bool, error) {
	ver, _, err := e.store.Delete(key)
	if errors.Is(err, storage.ErrNotFound) {
		return RestoredRecord{}, false, nil
	}
	if err != nil {
		return RestoredRecord{}, false, err
	}
	return RestoredRecord{
		Key:     key,
		Version: ver,
		Deleted: true,
	}, true, nil
}

// restoreChanges reports whether a full rollback of the snapshot would
// change the store's current state: true when any of the snapshot's keys
// holds a different record — version, data or tombstone flag — than the
// store does now, OR when any current LIVE key is absent from the snapshot,
// because the rollback deletes it (decision 14.1; missing the second half
// would make a load that only deletes look like a no-op). An extra current
// TOMBSTONE absent from the snapshot is not a change: the key is already
// deleted, and the rollback touches neither its version nor the dump. A
// snapshot equal to the store's current state with no extra live key is a
// whole-restore no-op (D-15), while any difference replays in full with
// every snapshot record re-stamped fresh (D-09's dominance contract).
func restoreChanges(current, snap map[string]storage.Record) bool {
	for key, rec := range snap {
		stored, ok := current[key]
		if !ok || stored.Version != rec.Version || stored.Deleted != rec.Deleted || !bytes.Equal(stored.Data, rec.Data) {
			return true
		}
	}
	for key, rec := range current {
		if rec.Deleted {
			continue
		}
		if _, inSnapshot := snap[key]; !inSnapshot {
			return true
		}
	}
	return false
}

// replayRecord installs one snapshot record through the store's ordinary
// write path (D-09): a tombstone replays as a fresh-version Delete whose
// ErrNotFound is a skip — a missing or already-tombstoned key has nothing
// to beat, and Delete writes nothing there anyway — and a live record as a
// fresh-version Set whose ErrValueTooLarge is a skip, mirroring Open, so a
// config-shrunk value limit never aborts a restore. The returned version is
// the store-stamped fresh version and replayed reports whether the store
// installed the record; a skip returns the zero version, replayed=false and
// a nil error, so a real replay can never read as a skip. The tombstone flag
// is not part of the signal — the caller already knows it from the snapshot
// entry.
func replayRecord(store *storage.Store, key string, rec storage.Record) (version.RecordVersion, bool, error) {
	if rec.Deleted {
		ver, _, err := store.Delete(key)
		if errors.Is(err, storage.ErrNotFound) {
			return version.RecordVersion{}, false, nil
		}
		if err != nil {
			return version.RecordVersion{}, false, err
		}
		return ver, true, nil
	}
	ver, _, err := store.Set(key, rec.Data)
	if errors.Is(err, storage.ErrValueTooLarge) {
		return version.RecordVersion{}, false, nil
	}
	if err != nil {
		return version.RecordVersion{}, false, err
	}
	return ver, true, nil
}
