package persist

import (
	"slices"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// This file is the loader/composition-root role (D-10, PERSIST-11): the one
// startup composition point, the record merge and the post-load dirty epoch.

// Open is the one startup composition point: it loads the fallback chain
// through dumps, decodes the first generation that decodes (per-record
// faults discard and are counted by class, never abort), seeds a clock with
// the dump's FULL watermark (VERS-04), rebuilds the store key by key via
// Merge, and wraps it all in an engine. The returned count is
// DecodeStats.Total() — the four discard classes summed, for observability.
// No decodable generation at all is the empty start, never an error: the
// chain already exhausted .prev, and an empty node recovers from peers
// (PERS-02). Decoding rides Load's own oracle argument: the generation Load
// accepts is exactly the one whose Decode succeeded, so Open holds no
// Decode-error fallback at all — the former guard after Load was dead code
// for every contract-honoring DumpStore (Decode is the readability oracle
// over the same bytes, codec.go) and is gone (03 IN-01, D-15).
//
// The decode runs before any store exists — its clock is seeded from the
// decoded records — so it reads the configured limit through
// storeCfg.ValueLimit(), the same helper NewStore normalizes with: the bound
// the decoder applies and the limit the store enforces cannot diverge
// (PERSIST-01).
//
// The startup load fires no change notification: the caller's
// storage.Config.OnChange tap is wrapped in a synchronous loading guard
// around mergeRecords (Pitfall 8, APIS-04), so a loaded dump is never
// announced as a live change, and a nil tap remains a valid caller contract
// (the wrapper substitutes a no-op).
func Open(
	nodeID uint16,
	now func() time.Time,
	driftWarn time.Duration,
	dumps port.DumpStore,
	storeCfg storage.Config,
	engineCfg Config,
) (eng *Engine, discarded int) {
	limit := storeCfg.ValueLimit()
	var records map[string]storage.Record
	var stats DecodeStats
	decode := func(dump []byte) error {
		decoded, decodedStats, err := Decode(dump, limit)
		if err != nil {
			return err
		}
		records, stats = decoded, decodedStats
		return nil
	}
	result := dumps.Load(now(), decode)
	if records == nil {
		records = make(map[string]storage.Record)
	}
	initialL, initialC := Watermark(records)
	clk := clock.NewClock(now, initialL, initialC, driftWarn)
	// The startup load is not a change feed (Pitfall 8, T-06-20): mergeRecords
	// installs every loaded record, and an unguarded tap would announce the
	// whole dump as live changes — on a clean Main load no dump follows to
	// confirm them, so they would sit in the hub's bounded pending queue until
	// it shed. Capture the caller's tap, substitute a no-op for a nil one
	// (storage.Config{} is a documented valid caller shape; NewStore's
	// normalized makes the same substitution for its own path), and gate it
	// with a synchronous loading flag: the merge runs on this goroutine, no
	// tap consumer exists before Open returns, and afterwards the flag is
	// only read, so a plain bool is race-free.
	onChange := storeCfg.OnChange
	if onChange == nil {
		onChange = func(string, storage.Record, uint64) {}
	}
	loading := true
	storeCfg.OnChange = func(key string, rec storage.Record, gen uint64) {
		if loading {
			return
		}
		onChange(key, rec, gen)
	}
	store := storage.NewStore(nodeID, clk, storeCfg)
	mergeRecords(store, records)
	loading = false
	eng = New(store, dumps, engineCfg)
	eng.loadStats = LoadStats{
		Loaded:     true,
		Outcome:    result.Outcome,
		Generation: result.Generation,
		Failures:   slices.Clone(result.Failures),
		Decoded:    stats,
		Swept:      result.Swept,
		SweepErr:   result.SweepErr,
	}
	eng.startLoadDirty(result.Outcome, now())
	return eng, stats.Total()
}

// startLoadDirty arms the post-load dirty epoch for every outcome but Main
// (D-04, PERSIST-04): a Prompt start has no generation at all, a Backup
// start's memory came from .prev, and an Unusable start is empty — each must
// flush before a Snapshot bytes-copies the file, otherwise the snapshot could
// capture a corrupt main and the first rotation could displace the last good
// generation. A Main load — even one whose decode discarded records — starts
// clean: the loaded generation is on disk and the same decoder reproduces the
// same in-memory state, while rewriting it would shorten the PERSIST-10
// evidence window. Called once by Open before Run exists, so no nudge is
// needed here; the first write's markDirty wakes the dumper.
func (e *Engine) startLoadDirty(outcome port.LoadOutcome, at time.Time) {
	if outcome == port.LoadMain {
		return
	}
	e.startDirty(at)
}

// mergeRecords installs every record via Merge. The decoder applied the same
// configured limit — Open decodes with storeCfg.ValueLimit(), and NewStore
// normalizes ValueMaxBytes through that same helper — so every record handed
// here is within the limit and Merge's only error class,
// storage.ErrValueTooLarge, cannot fire; the store remains the sole
// authority on the bound. The merge error is therefore ignored explicitly
// rather than re-checked as a skip over a state the decoder already excluded
// (IN-05).
func mergeRecords(store *storage.Store, records map[string]storage.Record) {
	for key, rec := range records {
		_, _, _ = store.Merge(key, rec)
	}
}
