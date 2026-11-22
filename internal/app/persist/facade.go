package persist

import (
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// This file is the store-facade role (D-10, PERSIST-11): the thin forwards
// to the embedded store plus ApplyRemote's merge-and-dirty shape. Reads are
// never gated on dumps, and no method here consults the confirmation or
// scheduling state except ApplyRemote's one dirty mark.

// ApplyRemote joins one remotely replicated record into the store and,
// only when the merge actually changed the stored record, marks the engine
// dirty with NO waiter — REPL-03's memory-apply contract, the seam await's
// doc reserved in Phase 3 ("only local writes register waiters, and Phase
// 4's remote merges will reuse the dirty path without waiters"): the
// replication core Acks a DataReplication as soon as the record is
// applied to memory, and the dump that persists it is asynchronous to that
// confirmation — the running Run picks the dirty flag up and dumps on the
// debounce schedule, so markDirty's non-blocking e.changed nudge is what wakes
// an idle dumper to re-arm its timers.
// The dirty decision is exactly storage.Store.MergeChanged's changed result
// (D-15, 03 IN-02 + 04 IN-02, threat T-6-01; PERSIST-07, ADR 0012): the
// store alone knows whether the incoming record became the stored record —
// a strictly newer version, the first record under the key, or an
// equal-version content tie-break win — while a stale rejection, a losing
// tie-break and an identical redelivery install nothing. The old
// Conflict/version heuristic cannot tell a tie-break win from a loss,
// because both pair the same version bytes, and the live-read probe could
// not see a stored tombstone; the changed result removes both ambiguities,
// so a no-op redelivery always leaves a clean engine clean (no redundant
// dump) and a content change never skips its dump.
// Store errors surface unchanged: an oversize record's ErrValueTooLarge
// merges nothing, dirties nothing and never wedges the engine (T-04-04).
// Not gated on Run's exit: the merge keeps the store converging even while
// the dumper is gone, exactly like reads keep serving (Get).
func (e *Engine) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	conflicts, changed, drift, err := e.store.MergeChanged(key, rec)
	if err != nil {
		return conflicts, drift, err
	}
	if !changed {
		return conflicts, drift, nil
	}
	e.markDirty()
	return conflicts, drift, nil
}

// Get returns a copy of the live record for key — reads are never gated on
// dumps, and a failed dump leaves every record readable in memory (D-05).
func (e *Engine) Get(key string) (storage.Record, error) {
	return e.store.Get(key)
}

// Lookup returns a copy of the stored record for key, tombstones included:
// the tombstone-visible replication view, unlike Get's live-only API read.
// It consults no dump or confirmation state — a failed dump leaves every
// record readable in memory (D-05).
func (e *Engine) Lookup(key string) (storage.Record, bool) {
	return e.store.Lookup(key)
}

// Version returns the stored version for key, tombstones included, and
// false when the key holds no record — the cheap per-entry replication
// probe, copying no Data and consulting no dump state.
func (e *Engine) Version(key string) (version.RecordVersion, bool) {
	return e.store.Version(key)
}

// Versions returns the caller-owned key→version snapshot, tombstones
// included — the listing view that copies no record data and consults no
// dump state.
func (e *Engine) Versions() map[string]version.RecordVersion {
	return e.store.Versions()
}

// Observe absorbs one received version into the store's clock without
// touching any record: the per-DataSync-part HLC update (PROTO-01). It
// consults no dump or confirmation state.
func (e *Engine) Observe(ver version.RecordVersion) time.Duration {
	return e.store.Observe(ver)
}

// KeyCount returns the number of stored records — tombstones included — in
// constant time: the startup empty-vs-data selection (PROTO-02). It clones
// no record data and consults no dump or confirmation state.
func (e *Engine) KeyCount() int {
	return e.store.Len()
}

// Generation returns the store's monotone content generation — moved only
// when a stored record actually changed — which the listing build records
// and reuses only while unchanged (04-08, T-4-43). It consults no dump or
// confirmation state.
func (e *Engine) Generation() uint64 {
	return e.store.Generation()
}

// Records returns a full snapshot of the map, tombstones included, with
// Data defensively copied — the same pass-through contract as Get.
func (e *Engine) Records() map[string]storage.Record {
	return e.store.Records()
}
