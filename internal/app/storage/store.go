package storage

import (
	"bytes"
	"hash/crc32"
	"sync"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/version"
)

// ValueMaxBytes is the one authoritative value-size limit (PERSIST-01,
// ADR 0011): the design default from docs/IDEA.md, exported so every
// consumer — the launch flag's default, the pre-store dump decode and the
// codec — derives the limit from this single definition. The store default,
// the codec's encode/decode bound and the binary's ceiling check all read it
// through Config.ValueLimit() or Store.ValueMaxBytes(); no second copy of the
// number exists in production. The checksum flag bytes of the
// RecordChecksum domain sit beside it.
const (
	ValueMaxBytes  = 61440
	notDeletedFlag = 0
	deletedFlag    = 1
)

// MaxKeyBytes is the documented key-length ceiling (XPORT-03): a longer key
// is rejected at the API boundary and on inbound datagrams. The binding
// constraint is the 65 507-byte datagram ceiling together with the
// 61 440-byte default value limit — 1 024 key bytes + 64 408 value bytes +
// the envelope overhead fill exactly 65 507. Validation is write-side only:
// the store keeps serving pre-upgrade records under longer keys, so reads
// stay permissive; Set/Delete and the receive path enforce this constant.
const MaxKeyBytes = 1024

// Config holds the store tuning parameters; it is immutable once handed to
// NewStore. A zero or negative ValueMaxBytes is normalized to
// ValueMaxBytes, and a nil OnChange to a no-op: the store is total over
// any Config (the repl.Config.Log precedent). Clock tuning — the drift
// tolerance and the restart watermark (VERS-04) — is constructor state of
// the injected clock (clock.NewClock), not store configuration.
type Config struct {
	ValueMaxBytes int

	// OnChange, when non-nil, is the change tap the Watch stream consumes
	// (API-03): install fires it under the store write lock with the record
	// that just became the stored record — the exact record a subsequent
	// Records() snapshot holds, recomputed checksum included, and the record
	// a subsequent Get returns only when it is live — and with the content
	// generation that install produced. The generation is the durability
	// watermark's unit (APIS-04, D-04): a tapped event is an install
	// announcement, never a durability claim; the persist engine publishes
	// the generation covered by a successful dump, and the Watch hub releases
	// the event only once that watermark reaches it. A tombstone install
	// fires the tap exactly like any other change (a delete IS a change),
	// yet a subsequent Get returns ErrNotFound for it — the tombstone is
	// visible only through Records(). Every change source
	// converges on install (local Set/SetIfAbsent/Delete writes, remote
	// merges, replicated tombstone deletes, snapshot-restore replay), so the
	// tap observes commit order for all of them, and the (version, content)
	// join law keeps a merge loser and an identical equal-version redelivery
	// from ever reaching it (D-06: events fire only on actual state changes;
	// a tie-break win at an equal version IS a change). The
	// callback is invoked synchronously from the write paths under the
	// store's locking discipline and must not block — fan-out belongs behind
	// a non-blocking channel send — and it must not call back into the same
	// store: the tap runs with the write lock held, Go's sync.RWMutex is
	// not reentrant, and even a read-only Get self-deadlocks, stalling every
	// store operation. The handed record is the
	// store's stored copy: records are replaced wholesale on the next
	// install, never mutated in place, so retaining it is safe.
	OnChange func(key string, rec Record, generation uint64)
}

// Record is one stored key's state. Data is defensively copied on the way
// into and out of the store; callers keep ownership of their slices.
type Record struct {
	Version  version.RecordVersion
	Data     []byte
	Deleted  bool
	Checksum uint32
}

// Conflict reports one displaced version from a Merge: the greater version
// that won and the version it beat or rejected, the equal-version content
// tie-break included. It is [Store.MergeChanged]/[Store.Merge]'s per-outcome
// return contract. It is NOT the source of the conflict counter
// (supkv_merges_conflicts_total): that counter measures pending (unacked)
// local writes displaced by an applied replicated record, computed from the
// replication core's pending set — an ordinary overwrite, a stale rejection
// and an identical redelivery are no conflicts (ADR 0004, OBSV-03).
type Conflict struct {
	Key    string
	Winner version.RecordVersion
	Loser  version.RecordVersion
}

// Clock is the version source the store consumes: Tick stamps local writes,
// Update absorbs received readings. The concrete clock (internal/app/clock)
// satisfies it; tests may substitute fakes. Declared consumer-side here —
// per D-03, internal/port holds no interfaces.
type Clock interface {
	Tick() (l int64, counter uint16, drift time.Duration)
	Update(remoteL int64, remoteC uint16) (l int64, counter uint16, drift time.Duration)
}

// Store is the in-memory LWW-Element-Map: per key the record with the
// greater full version wins on merge. Store is safe for concurrent use
// by multiple goroutines.
type Store struct {
	mu      sync.RWMutex
	nodeID  uint16
	cfg     Config
	records map[string]Record
	clock   Clock

	// generation is the content generation: it moves by exactly one per
	// install (every stored-record change converges there) and never for a
	// no-op or read path. Guarded by mu, like records.
	generation uint64
}

// NewStore returns a store for nodeID whose writes are stamped by clk, with
// cfg normalized to the design defaults. clk must be non-nil; the restart
// watermark and the drift tolerance are clk's constructor state (D-02: the
// single wall-clock seam lives in clock.NewClock), not store configuration.
func NewStore(nodeID uint16, clk Clock, cfg Config) *Store {
	return &Store{
		nodeID:  nodeID,
		cfg:     cfg.normalized(),
		records: make(map[string]Record),
		clock:   clk,
	}
}

// ValueLimit returns the effective value-size limit this Config describes:
// the configured ValueMaxBytes when positive, ValueMaxBytes otherwise.
// NewStore normalizes through this same helper, so a caller that must know
// the limit before a store exists — persist.Open's pre-store dump decode
// runs before the clock seed and the store construction — reads exactly the
// value the store will enforce.
func (c Config) ValueLimit() int {
	if c.ValueMaxBytes <= 0 {
		return ValueMaxBytes
	}
	return c.ValueMaxBytes
}

// normalized returns cfg with a zero or negative ValueMaxBytes replaced by
// ValueMaxBytes through ValueLimit and a nil OnChange replaced by a no-op.
func (c Config) normalized() Config {
	c.ValueMaxBytes = c.ValueLimit()
	if c.OnChange == nil {
		c.OnChange = func(_ string, _ Record, _ uint64) {}
	}
	return c
}

// ValueMaxBytes returns the effective value-size limit this store enforces.
// It is fixed by NewStore (Config is immutable afterwards) and reads without
// the store lock.
func (s *Store) ValueMaxBytes() int {
	return s.cfg.ValueMaxBytes
}

// tombstoneFlag carries the Record deleted flag into the checksum domain.
type tombstoneFlag bool

// checksumByte returns the checksum domain byte for the flag: deletedFlag
// for a tombstone, notDeletedFlag for a live record.
func (f tombstoneFlag) checksumByte() byte {
	if f {
		return deletedFlag
	}
	return notDeletedFlag
}

// RecordChecksum returns the CRC32 over the key bytes, the 16 version bytes,
// the data bytes and one flag byte for deleted. It is corruption detection
// for the Phase 3 dump, never tamper protection.
func RecordChecksum(key string, ver version.RecordVersion, data []byte, deleted bool) uint32 {
	flag := tombstoneFlag(deleted).checksumByte()
	sum := crc32.Update(0, crc32.IEEETable, []byte(key))
	sum = crc32.Update(sum, crc32.IEEETable, ver[:])
	sum = crc32.Update(sum, crc32.IEEETable, data)
	return crc32.Update(sum, crc32.IEEETable, []byte{flag})
}

// RecordsEqual reports whether two records hold identical content: the same
// full 16-byte version, the same tombstone flag and identical Data bytes.
// Checksum is excluded because it is derived from the key and those fields.
// It is the one content comparison the (version, content) tie-break and the
// replication core's equal-version branch share (PERSIST-07, ADR 0012), so the
// merge law and the wire rule cannot drift.
func RecordsEqual(a, b Record) bool {
	return a.Version == b.Version && a.Deleted == b.Deleted && bytes.Equal(a.Data, b.Data)
}

// Get returns a copy of the live record for key. A missing key and a
// tombstoned key are both ErrNotFound; the tombstone is invisible to Get.
func (s *Store) Get(key string) (Record, error) {
	s.mu.RLock()
	rec, ok := s.records[key]
	s.mu.RUnlock()
	if !ok || rec.Deleted {
		return Record{}, ErrNotFound
	}
	return Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Checksum: rec.Checksum,
	}, nil
}

// Version returns the stored version for key — tombstones included — and
// false when the key holds no record. It holds the read lock for one map
// lookup and copies no Data: the cheap per-entry probe a sync comparison
// takes instead of a whole-store snapshot.
func (s *Store) Version(key string) (version.RecordVersion, bool) {
	s.mu.RLock()
	rec, ok := s.records[key]
	s.mu.RUnlock()
	return rec.Version, ok
}

// Lookup returns a copy of the stored record for key, tombstones included:
// unlike Get, a deleted record is returned with Deleted true instead of
// ErrNotFound. This is the tombstone-visible point read reconciliation
// needs; the returned Data is a caller-owned clone, exactly like Get's.
func (s *Store) Lookup(key string) (Record, bool) {
	s.mu.RLock()
	rec, ok := s.records[key]
	s.mu.RUnlock()
	if !ok {
		return Record{}, false
	}
	return Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Deleted:  rec.Deleted,
		Checksum: rec.Checksum,
	}, true
}

// Versions returns a fresh map of every key — tombstones included — to its
// stored version, copying no Data. The map is caller-owned: mutating it
// never affects the store, and a consumer that needs the record itself
// calls Lookup for that key.
func (s *Store) Versions() map[string]version.RecordVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]version.RecordVersion, len(s.records))
	for key, rec := range s.records {
		out[key] = rec.Version
	}
	return out
}

// LiveVersions returns a fresh map of every live — non-tombstoned — key to
// its stored version, copying no Data. The map is caller-owned: mutating it
// never affects the store, and a consumer that needs the record itself calls
// Lookup for that key. Liveness is the stored Deleted flag and nothing else:
// len(Data)==0 is not a tombstone test, so a live record with an empty value
// is included. This is the List pagination view (D-02, APIS-02) — a
// tombstone-free key→version snapshot whose cost is the key count, never the
// store's value bytes.
func (s *Store) LiveVersions() map[string]version.RecordVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]version.RecordVersion, len(s.records))
	for key, rec := range s.records {
		if rec.Deleted {
			continue
		}
		out[key] = rec.Version
	}
	return out
}

// Set stores data under key, replacing any prior record under LWW, and
// returns the freshly stamped version and the drift signal observed at the
// clock Tick. Values over Config.ValueMaxBytes are rejected with
// ErrValueTooLarge before any copy is made.
func (s *Store) Set(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	if len(data) > s.cfg.ValueMaxBytes {
		return version.RecordVersion{}, 0, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ver, drift := s.write(key, data, false)
	return ver, drift, nil
}

// write stamps a fresh version via the clock and installs the record
// through install — the same convergence point a winning merge uses — so
// the OnChange tap observes local writes exactly like remote ones; the
// caller must hold the write lock. install clones the data defensively and
// recomputes the checksum. It returns the new version and the drift signal
// from the Tick.
func (s *Store) write(key string, data []byte, deleted bool) (version.RecordVersion, time.Duration) {
	l, counter, drift := s.clock.Tick()
	ver := version.PackVersion(l, counter, s.nodeID)
	s.install(key, Record{Version: ver, Data: data, Deleted: deleted})
	return ver, drift
}

// SetIfAbsent stores data under key only when the key holds no record,
// including no tombstone: any existing record is ErrAlreadyExists, so a
// Delete-then-SetIfAbsent must not recreate the key. Values over
// Config.ValueMaxBytes are rejected with ErrValueTooLarge before any copy
// is made.
func (s *Store) SetIfAbsent(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	if len(data) > s.cfg.ValueMaxBytes {
		return version.RecordVersion{}, 0, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[key]; ok {
		return version.RecordVersion{}, 0, ErrAlreadyExists
	}
	ver, drift := s.write(key, data, false)
	return ver, drift, nil
}

// Delete writes a fresh-version tombstone for key that merges like any
// write; a missing key and an already-tombstoned key are both ErrNotFound
// and write nothing, so no version is burned on a no-op.
func (s *Store) Delete(key string) (version.RecordVersion, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.records[key]; !ok || rec.Deleted {
		return version.RecordVersion{}, 0, ErrNotFound
	}
	ver, drift := s.write(key, nil, true)
	return ver, drift, nil
}

// MergeChanged joins an externally versioned record for key by the
// (version, content) order and reports whether the incoming record became the
// stored record. The version order is unchanged: only a strictly greater full
// 16-byte version displaces the stored record. At an equal version the
// deterministic content tie-break decides (PERSIST-07, ADR 0012): identical
// content is a full no-op, a tombstone beats a live record whichever side
// carries it, and otherwise the lexicographically greater Data bytes win.
// Every displaced or rejected version is returned exactly once as a Conflict —
// the equal-version tie-break included, with Winner and Loser the same version
// bytes. Values over Config.ValueMaxBytes are rejected with ErrValueTooLarge
// and nothing is stored. The changed result is authoritative for the dump
// decision: it is true exactly when the incoming record was installed.
func (s *Store) MergeChanged(key string, rec Record) (conflicts []Conflict, changed bool, drift time.Duration, err error) {
	if len(rec.Data) > s.cfg.ValueMaxBytes {
		return nil, false, 0, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	conflicts, changed = s.join(key, rec)
	remoteL, remoteC, _ := rec.Version.Unpack()
	_, _, drift = s.clock.Update(remoteL, remoteC)
	return conflicts, changed, drift, nil
}

// Merge joins an externally versioned record for key exactly like
// [Store.MergeChanged] and returns the conflicts and the clock drift: the
// three-result wrapper every pre-tie-break caller uses. It discards the
// changed result; callers that decide a dump or a reply from the installation
// outcome call MergeChanged directly.
func (s *Store) Merge(key string, rec Record) (conflicts []Conflict, drift time.Duration, err error) {
	conflicts, _, drift, err = s.MergeChanged(key, rec)
	return conflicts, drift, err
}

// Observe absorbs one received version into the store's clock without
// touching any record: the per-DataSync-part HLC update (PROTO-01), so the
// next local write strictly beats that version. It reads and writes no
// record, so it takes no store lock; the clock owns its own synchronization
// (the same seam Merge's trailing Update uses). The returned drift is the
// clock's own signal, exactly like Merge's.
func (s *Store) Observe(ver version.RecordVersion) time.Duration {
	l, c, _ := ver.Unpack()
	_, _, drift := s.clock.Update(l, c)
	return drift
}

// Len returns the number of stored records — tombstones included — in
// constant time: one read-locked map length, cloning nothing. It is the
// startup empty-vs-data selection probe (PROTO-02, T-4-11); Records() would
// clone every value.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// Generation returns the store's monotone content generation: it moves by
// exactly one whenever a stored record actually changed, because every
// change source converges on install — local Set/SetIfAbsent/Delete writes,
// winning remote merges (a content tie-break win included) and
// snapshot-restore replay. A rejected
// SetIfAbsent, a Delete of a missing key, a losing or identical-content
// Merge, the clock-only Observe and every read path therefore leave it
// unchanged: a content generation, never a call count (T-4-43).
func (s *Store) Generation() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.generation
}

// join compares rec against the stored record by the full 16-byte version and,
// at an equal version, by the deterministic content tie-break (PERSIST-07,
// ADR 0012): identical content is a full no-op, remove-wins settles a
// differing tombstone flag, and otherwise the greater Data bytes win by
// [bytes.Compare] over explicit fields (no map order, no randomness). It
// installs only when the incoming record wins and reports whether it did; the
// caller must hold the write lock. Every displaced or rejected version is
// returned exactly once as a Conflict — the tie-break included, where Winner
// and Loser are the same version bytes and the stored record is the only
// observable evidence of the outcome.
func (s *Store) join(key string, rec Record) ([]Conflict, bool) {
	stored, ok := s.records[key]
	if !ok {
		s.install(key, rec)
		return nil, true
	}
	switch cmp := rec.Version.Compare(stored.Version); {
	case cmp > 0:
		s.install(key, rec)
		return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
	case cmp < 0:
		return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
	default:
		switch {
		case RecordsEqual(stored, rec):
			return nil, false
		case rec.Deleted && !stored.Deleted:
			// Remove-wins: a delete at the same version beats a live value.
			s.install(key, rec)
			return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
		case stored.Deleted && !rec.Deleted:
			return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
		case bytes.Compare(rec.Data, stored.Data) > 0:
			s.install(key, rec)
			return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
		default:
			return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
		}
	}
}

// install defensively copies rec and stores it under key with a recomputed
// checksum, bumps the content generation and fires the Config.OnChange tap
// with the stored record and that generation. install is the single
// convergence point of every change source — local writes reach it through
// write, remote merges through a winning join branch — so the tap fires
// exactly when a record became the stored record (D-06) and observes commit
// order, and the generation moves exactly once per actual change and never
// for a no-op; the caller must hold the write lock.
func (s *Store) install(key string, rec Record) {
	data := bytes.Clone(rec.Data)
	stored := Record{
		Version:  rec.Version,
		Data:     data,
		Deleted:  rec.Deleted,
		Checksum: RecordChecksum(key, rec.Version, data, rec.Deleted),
	}
	s.records[key] = stored
	s.generation++
	s.cfg.OnChange(key, stored, s.generation)
}

// Records returns a full snapshot of the map, tombstones included, with Data
// defensively copied. This is the Phase 3 dump source; Get cannot see
// tombstones by design.
func (s *Store) Records() map[string]Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]Record, len(s.records))
	for key, rec := range s.records {
		out[key] = Record{
			Version:  rec.Version,
			Data:     bytes.Clone(rec.Data),
			Deleted:  rec.Deleted,
			Checksum: rec.Checksum,
		}
	}
	return out
}
