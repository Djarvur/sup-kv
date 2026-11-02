package app

import "time"

// Config holds the store tuning parameters. Zero fields are normalized by
// NewStore to the design defaults: ValueMaxBytes 61440 and DriftWarn 60s.
// InitialL seeds the hybrid logical clock from a dump's maximum l (VERS-04);
// zero means start from the wall clock.
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
type Store struct{}

// NewStore returns a store for nodeID with cfg normalized to the design
// defaults and a clock driven by now (D-02: the single wall-clock seam). now
// must be non-nil.
func NewStore(_ uint16, _ Config, _ func() time.Time) *Store {
	return &Store{}
}

// Get returns a copy of the live record for key. A missing key and a
// tombstoned key are both ErrNotFound; the tombstone is invisible to Get.
func (*Store) Get(_ string) (Record, error) {
	return Record{}, nil
}

// Set stores data under key, replacing any prior record under LWW, and
// returns the freshly stamped version and the drift signal observed at the
// clock Tick. Values over Config.ValueMaxBytes are rejected with
// ErrValueTooLarge before any copy is made.
func (*Store) Set(_ string, _ []byte) (RecordVersion, time.Duration, error) {
	return RecordVersion{}, 0, nil
}
