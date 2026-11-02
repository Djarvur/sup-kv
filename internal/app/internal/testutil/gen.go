package testutil

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// Generator tuning and domain constants: the RFC 9562 bounds RandVersion
// draws from, the byte alphabet of raw draws, and the fixture shapes.
const (
	lDomainBits = 48  // the 48-bit unix_ts_ms domain.
	byteValues  = 256 // one byte's worth of random values.
	tombChance  = 3   // one in three generated records is a tombstone.
	dataMax     = 64  // random data length ceiling, bytes.
	keySpace    = 32  // small key space so records collide.
	recsMax     = 16  // records merged into one RandStore.
)

// fixedNow returns a now func parked at the fixed test wall-clock
// millisecond, built with [time.UnixMilli] so no monotonic reading is
// involved (D-02).
func fixedNow() func() time.Time {
	return func() time.Time { return time.UnixMilli(testNowMs) }
}

// RandVersion packs a valid version from a random in-range HLC reading:
// l anywhere in the 48-bit domain, counter 0..4095, nodeID 0..0xFFFF.
func RandVersion(rng *rand.Rand) version.RecordVersion {
	return version.PackVersion(
		rng.Int64N(1<<lDomainBits),
		uint16(rng.IntN(testCounterMax+1)), //nolint:gosec // G115: IntN(4096) <= 4095, the conversion cannot overflow
		uint16(rng.IntN(maxNodeID+1)),      //nolint:gosec // G115: IntN(0x10000) <= 0xFFFF, the conversion cannot overflow
	)
}

// RandRawVersion fills all 16 bytes with random bits: an arbitrary bit
// pattern off the wire, the trust-boundary input of Unpack and Compare.
func RandRawVersion(rng *rand.Rand) version.RecordVersion {
	var v version.RecordVersion
	for i := range v {
		v[i] = byte(rng.IntN(byteValues)) //nolint:gosec // G115: IntN(256) < 256, the conversion cannot overflow
	}
	return v
}

// RandRecord returns a record carrying a random version: roughly one third
// tombstones with nil Data, the rest up to 63 random data bytes.
func RandRecord(rng *rand.Rand) storage.Record {
	if rng.IntN(tombChance) == 0 {
		return storage.Record{Version: RandVersion(rng), Deleted: true}
	}
	data := make([]byte, rng.IntN(dataMax))
	for i := range data {
		data[i] = byte(rng.IntN(byteValues)) //nolint:gosec // G115: IntN(256) < 256, the conversion cannot overflow
	}
	return storage.Record{Version: RandVersion(rng), Data: data}
}

// RandKey returns a random key from a small space so records collide.
func RandKey(rng *rand.Rand) string {
	return fmt.Sprintf("key-%d", rng.IntN(keySpace))
}

// RandStore returns a store pre-populated by Merge with RandRecord records
// over random keys: Merge installs externally versioned records, which is
// how tests construct stores with arbitrary versions (research Pattern 2).
func RandStore(rng *rand.Rand) *storage.Store {
	// The one-minute drift tolerance mirrors the design default from
	// docs/IDEA.md; generated stores never read the drift signal.
	clk := clock.NewClock(fixedNow(), 0, 0, time.Minute)
	store := storage.NewStore(
		uint16(rng.IntN(maxNodeID+1)), //nolint:gosec // G115: IntN(0x10000) <= 0xFFFF, the conversion cannot overflow
		clk,
		storage.Config{},
	)
	for range 1 + rng.IntN(recsMax) {
		_, _, _ = store.Merge(RandKey(rng), RandRecord(rng))
	}
	return store
}
