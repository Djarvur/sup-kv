package app

import (
	"bytes"
	"hash/crc32"
	"sync"
	"time"
)

// Design defaults NewStore substitutes for zero Config fields, and the
// checksum flag bytes of the RecordChecksum domain, from docs/IDEA.md.
const (
	defaultValueMaxBytes = 61440
	defaultDriftWarn     = 60 * time.Second
	notDeletedFlag       = 0
	deletedFlag          = 1
)

// Config holds the store tuning parameters; it is immutable once handed to
// NewStore. Zero fields are normalized to the design defaults:
// ValueMaxBytes 61440 and DriftWarn 60s. InitialL seeds the hybrid logical
// clock from a dump's maximum l (VERS-04); zero means start from the wall
// clock. A negative DriftWarn is passed through and disables the signal.
type Config struct {
	ValueMaxBytes int
	DriftWarn     time.Duration
	InitialL      int64
}

// Record is one stored key's state. Data is defensively copied on the way
// into and out of the store; callers keep ownership of their slices.
type Record struct {
	Version  RecordVersion
	Data     []byte
	Deleted  bool
	Checksum uint32
}

// Conflict reports one displaced version from a Merge: the greater version
// that won and the version it beat or rejected. It is the D-01 return-value
// contract behind the Phase 4/6 conflict journal.
type Conflict struct {
	Key    string
	Winner RecordVersion
	Loser  RecordVersion
}

// Store is the in-memory LWW-Element-Map: per key the record with the
// greater full RecordVersion wins on merge. Store is safe for concurrent use
// by multiple goroutines.
type Store struct {
	mu      sync.RWMutex
	nodeID  uint16
	cfg     Config
	records map[string]Record
	clock   *Clock
}

// NewStore returns a store for nodeID with cfg normalized to the design
// defaults and a clock built from cfg.InitialL, cfg.DriftWarn and now
// (D-02: the single wall-clock seam). now must be non-nil.
func NewStore(nodeID uint16, cfg Config, now func() time.Time) *Store {
	cfg = cfg.normalized()
	return &Store{
		nodeID:  nodeID,
		cfg:     cfg,
		records: make(map[string]Record),
		clock:   NewClock(now, cfg.InitialL, cfg.DriftWarn),
	}
}

// normalized returns cfg with zero fields replaced by the design defaults.
// A negative ValueMaxBytes is normalized too; a negative DriftWarn is kept
// and disables the drift signal (see NewClock).
func (c Config) normalized() Config {
	if c.ValueMaxBytes <= 0 {
		c.ValueMaxBytes = defaultValueMaxBytes
	}
	if c.DriftWarn == 0 {
		c.DriftWarn = defaultDriftWarn
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
func RecordChecksum(key string, ver RecordVersion, data []byte, deleted bool) uint32 {
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

// Set stores data under key, replacing any prior record under LWW, and
// returns the freshly stamped version and the drift signal observed at the
// clock Tick. Values over Config.ValueMaxBytes are rejected with
// ErrValueTooLarge before any copy is made.
func (s *Store) Set(key string, data []byte) (RecordVersion, time.Duration, error) {
	if len(data) > s.cfg.ValueMaxBytes {
		return RecordVersion{}, 0, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ver, drift := s.write(key, data, false)
	return ver, drift, nil
}

// write stamps a fresh version via the clock, defensively copies data and
// stores the record; the caller must hold the write lock. It returns the new
// version and the drift signal from the Tick.
func (s *Store) write(key string, data []byte, deleted bool) (RecordVersion, time.Duration) {
	l, counter, drift := s.clock.Tick()
	ver := PackVersion(l, counter, s.nodeID)
	stored := bytes.Clone(data)
	s.records[key] = Record{
		Version:  ver,
		Data:     stored,
		Deleted:  deleted,
		Checksum: RecordChecksum(key, ver, stored, deleted),
	}
	return ver, drift
}

// SetIfAbsent stores data under key only when the key holds no record,
// including no tombstone: any existing record is ErrAlreadyExists, so a
// Delete-then-SetIfAbsent must not recreate the key. Values over
// Config.ValueMaxBytes are rejected with ErrValueTooLarge before any copy
// is made.
func (s *Store) SetIfAbsent(key string, data []byte) (RecordVersion, time.Duration, error) {
	if len(data) > s.cfg.ValueMaxBytes {
		return RecordVersion{}, 0, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[key]; ok {
		return RecordVersion{}, 0, ErrAlreadyExists
	}
	ver, drift := s.write(key, data, false)
	return ver, drift, nil
}

// Delete writes a fresh-version tombstone for key that merges like any
// write; a missing key and an already-tombstoned key are both ErrNotFound
// and write nothing, so no version is burned on a no-op.
func (s *Store) Delete(key string) (RecordVersion, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.records[key]; !ok || rec.Deleted {
		return RecordVersion{}, 0, ErrNotFound
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
// checksum; the caller must hold the write lock.
func (s *Store) install(key string, rec Record) {
	data := bytes.Clone(rec.Data)
	s.records[key] = Record{
		Version:  rec.Version,
		Data:     data,
		Deleted:  rec.Deleted,
		Checksum: RecordChecksum(key, rec.Version, data, rec.Deleted),
	}
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
