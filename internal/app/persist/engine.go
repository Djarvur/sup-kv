package persist

import (
	"context"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Design defaults a zero or negative Config field is normalized to (D-04).
const (
	defaultDebounce = 50 * time.Millisecond
	defaultMaxWait  = time.Second
)

// Config holds the dump timing parameters; it is immutable once handed to
// New. A zero or negative field is normalized to the design default.
type Config struct {
	Debounce time.Duration
	MaxWait  time.Duration
}

// normalized returns cfg with zero or negative fields replaced by the design
// defaults.
func (c Config) normalized() Config {
	if c.Debounce <= 0 {
		c.Debounce = defaultDebounce
	}
	if c.MaxWait <= 0 {
		c.MaxWait = defaultMaxWait
	}
	return c
}

// Engine is the write-confirmation engine over one store (stub).
type Engine struct {
	store *storage.Store
	dumps port.DumpStore
	cfg   Config
}

// New returns an engine persisting store's state through dumps.
func New(store *storage.Store, dumps port.DumpStore, cfg Config) *Engine {
	return &Engine{store: store, dumps: dumps, cfg: cfg.normalized()}
}

// Open builds an engine from the dump on disk (stub: always the empty start).
func Open(
	nodeID uint16,
	now func() time.Time,
	driftWarn time.Duration,
	dumps port.DumpStore,
	storeCfg storage.Config,
	engineCfg Config,
) (*Engine, int) {
	clk := clock.NewClock(now, 0, 0, driftWarn)
	return New(storage.NewStore(nodeID, clk, storeCfg), dumps, engineCfg), 0
}

// Run is the single dumper goroutine (stub).
func (e *Engine) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// Set stores data and returns after a covering dump (stub: no confirmation).
func (e *Engine) Set(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	return e.store.Set(key, data)
}

// Delete writes a fresh-version tombstone and returns after a covering dump
// (stub: no confirmation).
func (e *Engine) Delete(key string) (version.RecordVersion, time.Duration, error) {
	return e.store.Delete(key)
}

// Get returns a copy of the live record for key.
func (e *Engine) Get(key string) (storage.Record, error) {
	return e.store.Get(key)
}

// Records returns a full snapshot of the map, tombstones included.
func (e *Engine) Records() map[string]storage.Record {
	return e.store.Records()
}
