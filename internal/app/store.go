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

// RecordChecksum returns the CRC32 over the key bytes, the 16 version bytes,
// the data bytes and one flag byte for deleted. It is corruption detection
// for the Phase 3 dump, never tamper protection.
//
//nolint:revive // flag-parameter: deleted is checksum domain data (the flag byte 0/1 of the dump format), not a control flag; the signature is fixed by the Phase 3 dump contract.
func RecordChecksum(key string, ver RecordVersion, data []byte, deleted bool) uint32 {
	flag := byte(notDeletedFlag)
	if deleted {
		flag = deletedFlag
	}
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
