package grpcapi_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"google.golang.org/grpc/codes"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// rollbackSnapshotFixture encodes a snapshot generation holding the given
// live records with the production checksums (the 03-08 hand-built-fixture
// rule): a zero-checksum fixture would be discarded at decode and the load
// could pass for the wrong reason.
func rollbackSnapshotFixture(t *testing.T, records map[string]storage.Record) []byte {
	t.Helper()
	for key, rec := range records {
		rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
		records[key] = rec
	}
	dump, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("persist.Encode(snapshot fixture) error = %v, want nil", err)
	}
	return dump
}

// seedRollbackKeys writes values straight through the engine — never the RPC
// handler — so the notify capture observes only the snapshot load.
func seedRollbackKeys(t *testing.T, eng *persist.Engine, values map[string]string) {
	t.Helper()
	for key, value := range values {
		_, _, err := eng.Set(key, []byte(value))
		if err != nil {
			t.Fatalf("seed Set(%q) error = %v, want nil", key, err)
		}
	}
}

// installRollbackSnapshot hands the harness's DumpStore the snapshot bytes
// the load will read, leaving the failure knob to the caller.
func installRollbackSnapshot(t *testing.T, dumps *fakeDumpStore, dump []byte, failSave bool) {
	t.Helper()
	dumps.mu.Lock()
	defer dumps.mu.Unlock()
	dumps.snapshot = bytes.Clone(dump)
	dumps.failSave = failSave
}

// TestServer_snapshotLoadRollback proves the RPC half of APIS-07 / decision
// 14.1 (ADR 0001): a snapshot load is a full rollback, and the rollback
// tombstones it installs ride the SAME notify path PROTO-08 already drives —
// the handler broadcasts exactly the record list Restore returns, so a key
// absent from the snapshot is announced as deleted without any handler
// change. The notification order is the replay's deterministic sorted union
// of snapshot keys and rollback keys, and a load that changes nothing
// notifies nothing.
func TestServer_snapshotLoadRollback(t *testing.T) {
	t.Parallel()

	snapVersion := version.PackVersion(1_700_000_000_000, 3, 7)
	// snap/mike is the post-snapshot live key: absent from the fixture, so
	// the rollback deletes it. It sorts between the two replayed keys, which
	// makes the notification order distinguish a sorted union from either
	// "snapshot keys first" or map iteration.
	seeds := map[string]string{
		"snap/alpha": "alpha-value",
		"snap/mike":  "mike-value",
		"snap/zulu":  "zulu-value",
	}
	fixture := func() map[string]storage.Record {
		return map[string]storage.Record{
			"snap/alpha": {Version: snapVersion, Data: []byte("snap-alpha")},
			"snap/zulu":  {Version: snapVersion, Data: []byte("snap-zulu")},
		}
	}
	wantKeys := []string{"snap/alpha", "snap/mike", "snap/zulu"}

	t.Run("notifies the rollback tombstone", func(t *testing.T) {
		t.Parallel()

		ts, capture := newNotifyServer(t)
		seedRollbackKeys(t, ts.engine, seeds)
		preLoad := ts.engine.Records()
		snap := fixture()
		installRollbackSnapshot(t, ts.dumps, rollbackSnapshotFixture(t, snap), false)

		_, err := ts.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "rollback"})
		if err != nil {
			t.Fatalf("SnapshotLoad error = %v, want nil", err)
		}

		calls := capture.snapshot()
		gotKeys := make([]string, len(calls))
		for i, call := range calls {
			gotKeys[i] = call.key
		}
		if !slices.Equal(gotKeys, wantKeys) {
			t.Fatalf("notify keys = %v, want %v (the sorted union of replay and rollback keys)", gotKeys, wantKeys)
		}
		for _, call := range calls {
			if call.track {
				t.Errorf("snapshot-load notify for %q has track=true, want false", call.key)
			}
			if call.key == "snap/mike" {
				if !call.rec.Deleted {
					t.Error("notified snap/mike is live, want Deleted=true (the rollback tombstone)")
				}
				if len(call.rec.Data) != 0 {
					t.Errorf("notified snap/mike carries %d data bytes, want none", len(call.rec.Data))
				}
				if prior := preLoad["snap/mike"].Version; call.rec.Version.Compare(prior) != 1 {
					t.Errorf("notified snap/mike version %x does not dominate its pre-load version %x",
						call.rec.Version[:], prior[:])
				}
				continue
			}
			if call.rec.Deleted {
				t.Errorf("notified %q is a tombstone, want the snapshot's live record", call.key)
			}
			if !bytes.Equal(call.rec.Data, snap[call.key].Data) {
				t.Errorf("notified %q data = %q, want the snapshot's value", call.key, call.rec.Data)
			}
		}

		// Store outcome: the extra key is gone for Get, the replayed keys are
		// live with the snapshot's values.
		_, getErr := ts.engine.Get("snap/mike")
		if !errors.Is(getErr, storage.ErrNotFound) {
			t.Errorf("post-load Get(snap/mike) error = %v, want ErrNotFound (the rollback tombstone wins)", getErr)
		}
		rec, getErr := ts.engine.Get("snap/alpha")
		if getErr != nil || string(rec.Data) != "snap-alpha" {
			t.Errorf("post-load Get(snap/alpha) = (%q, %v), want the replayed snapshot value", rec.Data, getErr)
		}
	})

	t.Run("failed final dump still notifies the rollback", func(t *testing.T) {
		t.Parallel()

		ts, capture := newNotifyServer(t)
		seedRollbackKeys(t, ts.engine, seeds)
		installRollbackSnapshot(t, ts.dumps, rollbackSnapshotFixture(t, fixture()), true)

		_, err := ts.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "rollback"})
		assertCode(t, "SnapshotLoad with a failed final dump", err, codes.Unavailable)

		calls := capture.snapshot()
		gotKeys := make([]string, len(calls))
		for i, call := range calls {
			gotKeys[i] = call.key
		}
		if !slices.Equal(gotKeys, wantKeys) {
			t.Fatalf("notify keys after the failed dump = %v, want %v (PROTO-08 totality survives the dump failure)",
				gotKeys, wantKeys)
		}
		if !calls[1].rec.Deleted {
			t.Error("notified snap/mike is live after the failed dump, want Deleted=true")
		}
	})

	t.Run("no-op load notifies nothing", func(t *testing.T) {
		t.Parallel()

		ts, capture := newNotifyServer(t)
		seedRollbackKeys(t, ts.engine, map[string]string{
			"same/alpha": "alpha-value",
			"same/beta":  "beta-value",
		})
		// The snapshot IS the current state, extra live keys included, so the
		// rollback has nothing to delete.
		dump, err := persist.Encode(ts.engine.Records(), storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("persist.Encode(current state) error = %v, want nil", err)
		}
		installRollbackSnapshot(t, ts.dumps, dump, false)

		_, loadErr := ts.client.SnapshotLoad(t.Context(), &apiproto.SnapshotLoadRequest{Name: "same"})
		if loadErr != nil {
			t.Fatalf("no-op SnapshotLoad error = %v, want nil", loadErr)
		}
		if calls := capture.snapshot(); len(calls) != 0 {
			t.Fatalf("the no-op load notified %d records, want 0", len(calls))
		}
	})
}
