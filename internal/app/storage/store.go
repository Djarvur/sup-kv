package storage

import (
	"bytes"
	"hash/crc32"
	"sync"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/version"
)

// Design default NewStore substitutes for a zero Config field, and the
// checksum flag bytes of the RecordChecksum domain, from docs/IDEA.md.
const (
	defaultValueMaxBytes = 61440
	notDeletedFlag       = 0
	deletedFlag          = 1
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
// NewStore. A zero or negative ValueMaxBytes is normalized to the design
// default of 61440, and a nil OnChange to a no-op: the store is total over
// any Config (the repl.Config.Log precedent). Clock tuning — the drift
// tolerance and the restart watermark (VERS-04) — is constructor state of
// the injected clock (clock.NewClock), not store configuration.
type Config struct {
	ValueMaxBytes int

	// OnChange, when non-nil, is the change tap the Watch stream consumes
	// (API-03): install fires it under the store write lock with the record
	// that just became the stored record — the exact record a subsequent
	// Records() snapshot holds, recomputed checksum included, and the record
	// a subsequent Get returns only when it is live. A tombstone install
	// fires the tap exactly like any other change (a delete IS a change),
	// yet a subsequent Get returns ErrNotFound for it — the tombstone is
	// visible only through Records(). Every change source
	// converges on install (local Set/SetIfAbsent/Delete writes, remote
	// merges, replicated tombstone deletes, snapshot-restore replay), so the
	// tap observes commit order for all of them, and the strictly-greater
	// join law keeps a merge loser and an equal-version redelivery from ever
	// reaching it (D-06: events fire only on actual state changes). The
	// callback is invoked synchronously from the write paths under the
	// store's locking discipline and must not block — fan-out belongs behind
	// a non-blocking channel send — and it must not call back into the same
	// store: the tap runs with the write lock held, Go's sync.RWMutex is
	// not reentrant, and even a read-only Get self-deadlocks, stalling every
	// store operation. The handed record is the
	// store's stored copy: records are replaced wholesale on the next
	// install, never mutated in place, so retaining it is safe.
	OnChange func(key string, rec Record)
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
// that won and the version it beat or rejected. It is the D-01 return-value
// contract behind the Phase 4/6 conflict journal.
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

// normalized returns cfg with a zero or negative ValueMaxBytes replaced by
// the design default and a nil OnChange replaced by a no-op.
func (c Config) normalized() Config {
	if c.ValueMaxBytes <= 0 {
		c.ValueMaxBytes = defaultValueMaxBytes
	}
	if c.OnChange == nil {
		c.OnChange = func(_ string, _ Record) {}
	}
	return c
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

// Merge joins an externally versioned record for key: the remote record
// keeps its own version, only a strictly greater full 16-byte version
// displaces the stored record, and an equal version is a full no-op. Every
// displaced or rejected version is returned exactly once as a Conflict.
// Values over Config.ValueMaxBytes are rejected with ErrValueTooLarge and
// nothing is stored.
func (s *Store) Merge(key string, rec Record) (conflicts []Conflict, drift time.Duration, err error) {
	if len(rec.Data) > s.cfg.ValueMaxBytes {
		return nil, 0, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	conflicts = s.join(key, rec)
	remoteL, remoteC, _ := rec.Version.Unpack()
	_, _, drift = s.clock.Update(remoteL, remoteC)
	return conflicts, drift, nil
}

// Observe absorbs one received version into the store's clock without
// touching any record: the per-DataSync-part HLC update (PROTO-01), so the
// next local write strictly beats that version. It reads and writes no
// record, so it takes no store lock; the clock owns its own synchronization
// (the same seam Merge's trailing Update uses). The returned drift is the
// clock's own signal, exactly like Merge's.
func (s *Store) Observe(ver version.RecordVersion) time.Duration {
	return 0
}

// Len returns the number of stored records — tombstones included — in
// constant time: one read-locked map length, cloning nothing. It is the
// startup empty-vs-data selection probe (PROTO-02, T-4-11); Records() would
// clone every value.
func (s *Store) Len() int {
	return 0
}

// Generation returns the store's monotone content generation: it moves by
// exactly one whenever a stored record actually changed, because every
// change source converges on install — local Set/SetIfAbsent/Delete writes,
// winning remote merges and snapshot-restore replay. A rejected
// SetIfAbsent, a Delete of a missing key, an equal-or-older Merge, the
// clock-only Observe and every read path therefore leave it unchanged: a
// content generation, never a call count (T-4-43).
func (s *Store) Generation() uint64 {
	return 0
}

// join compares rec against the stored record by the full 16-byte version
// and installs it when strictly greater; the caller must hold the write
// lock. It returns the displaced or rejected versions as conflicts.
func (s *Store) join(key string, rec Record) []Conflict {
	stored, ok := s.records[key]
	if !ok {
		s.install(key, rec)
		return nil
	}
	cmp := rec.Version.Compare(stored.Version)
	switch {
	case cmp > 0:
		s.install(key, rec)
		return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}
	case cmp < 0:
		return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}
	default:
		return nil
	}
}

// install defensively copies rec and stores it under key with a recomputed
// checksum, then fires the Config.OnChange tap with the stored record.
// install is the single convergence point of every change source — local
// writes reach it through write, remote merges through the strictly-greater
// join branch — so the tap fires exactly when a record became the stored
// record (D-06) and observes commit order; the caller must hold the write
// lock.
func (s *Store) install(key string, rec Record) {
	data := bytes.Clone(rec.Data)
	stored := Record{
		Version:  rec.Version,
		Data:     data,
		Deleted:  rec.Deleted,
		Checksum: RecordChecksum(key, rec.Version, data, rec.Deleted),
	}
	s.records[key] = stored
	s.cfg.OnChange(key, stored)
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
