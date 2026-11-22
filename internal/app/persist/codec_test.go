package persist_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// Codec-test constants: the property step count (the merge-law precedent),
// the decode-side envelope the discard class probes one byte past, and the
// fixture versions' clock readings.
const (
	codecPropertySteps = 500
	oversizeBy         = 1
	// watermarkOrderScans re-scans the equal-counter fixture: map iteration
	// order is random per call, so the law is asserted over many orders.
	watermarkOrderScans = 100
	// tornTailCut bytes off the dump's tail, landing the cut mid-JSON of the
	// last record line — the torn-tail residue class.
	tornTailCut = 10
	// The TEST-15 small (l, c, node) space — the same 3x3x3 box the storage
	// merge property draws from (prop_small_space_test.go), so a counterexample
	// in either suite reduces to the same short adjacent-version shape. Half
	// the codec property's record versions come from here and the wide half
	// stays the unchanged testutil draw.
	smallSpaceLBase     = int64(1730000000000)
	smallSpaceLRange    = 3
	smallSpaceCRange    = 3
	smallSpaceNodeRange = 3
	smallSpaceKeyRange  = 3
	smallSpaceRecords   = 8
)

// dumpLine is the test-side mirror of the codec's record line — the shape
// tamper rewrites a field of.
type dumpLine struct {
	Key      string `json:"key"`
	Version  string `json:"record_version"`
	Data     []byte `json:"data"`
	Deleted  bool   `json:"deleted"`
	Checksum uint32 `json:"checksum"`
}

// rewriteLine unmarshals the record line carrying key, applies fix and
// marshals it back — a structurally valid JSON line whose only fault is the
// field fix set up.
func rewriteLine(t *testing.T, dump []byte, key string, fix func(*dumpLine)) []byte {
	t.Helper()
	lines := strings.Split(string(dump), "\n")
	for i, line := range lines {
		if !strings.Contains(line, `"key":"`+key+`"`) {
			continue
		}
		var rec dumpLine
		err := json.Unmarshal([]byte(line), &rec)
		if err != nil {
			t.Fatalf("unmarshal %s: %v", key, err)
		}
		fix(&rec)
		fixed, _ := json.Marshal(rec) // errchkjson: dumpLine is error-free; the blank is the config-sanctioned form.
		lines[i] = string(fixed)
		return []byte(strings.Join(lines, "\n"))
	}
	t.Fatalf("dump carries no line for key %s", key)
	return nil
}

// lineKey extracts a record line's key field — the first struct field, so
// the prefix cut suffices.
func lineKey(t *testing.T, line string) string {
	t.Helper()
	_, rest, found := strings.Cut(line, `"key":"`)
	if !found {
		t.Fatalf("line %q lacks a key field", line)
	}
	key, _, _ := strings.Cut(rest, `"`)
	return key
}

// sealed returns rec with its checksum computed over its own fields — the
// store's install discipline, hand-built for the fixtures.
func sealed(key string, rec storage.Record) storage.Record {
	rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
	return rec
}

// baseRecords returns the two surviving records every discard row keeps.
func baseRecords() map[string]storage.Record {
	return map[string]storage.Record{
		"keep-a": sealed("keep-a", storage.Record{Version: version.PackVersion(1000, 1, 1), Data: []byte("a-value")}),
		"keep-b": sealed("keep-b", storage.Record{Version: version.PackVersion(1001, 2, 1), Data: []byte("b-value")}),
	}
}

// encodeRecords merges base with extras and encodes it at the default value
// limit.
func encodeRecords(t *testing.T, extras map[string]storage.Record) []byte {
	t.Helper()
	return encodeRecordsAt(t, storage.ValueMaxBytes, extras)
}

// encodeRecordsAt merges base with extras and encodes it at limit. The
// oversize-discard fixture passes a deliberately larger limit than its
// decode so the encoder's tripwire cannot make the fixture unbuildable
// (RESEARCH Pitfall 11): the dump must exist to prove Decode discards the
// record.
func encodeRecordsAt(t *testing.T, limit int, extras map[string]storage.Record) []byte {
	t.Helper()
	records := baseRecords()
	maps.Copy(records, extras)
	dump, err := persist.Encode(records, limit)
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	return dump
}

// smallSpaceVersion returns a version drawn from TEST-15's tiny space: l in
// {base, base+1, base+2}, counter 0..2, node 1..3.
func smallSpaceVersion(rng *rand.Rand) version.RecordVersion {
	return version.PackVersion(
		smallSpaceLBase+int64(rng.IntN(smallSpaceLRange)),
		uint16(rng.IntN(smallSpaceCRange)),
		uint16(1+rng.IntN(smallSpaceNodeRange)),
	)
}

// mixedVersion draws half the versions from the small space and half from the
// unchanged wide testutil draw — the TEST-15 distribution.
func mixedVersion(rng *rand.Rand) version.RecordVersion {
	if rng.IntN(2) == 0 {
		return smallSpaceVersion(rng)
	}
	return testutil.RandVersion(rng)
}

// payloadForVersion derives the record payload from the version alone: the
// version bytes for a live record, a tombstone for every third first byte. Two
// records with equal versions are therefore byte-identical — the same-write
// premise the storage small-space property relies on.
func payloadForVersion(ver version.RecordVersion) storage.Record {
	if ver[0]%3 == 0 {
		return storage.Record{Version: ver, Deleted: true}
	}
	return storage.Record{Version: ver, Data: ver[:]}
}

// mixedSpaceRecords returns a map drawn with the TEST-15 distribution: keys
// from a three-key set and half the versions from the small (l, c, node)
// space, every record sealed for the codec's checksum check.
func mixedSpaceRecords(rng *rand.Rand) map[string]storage.Record {
	records := make(map[string]storage.Record, smallSpaceRecords)
	for range 1 + rng.IntN(smallSpaceRecords) {
		key := fmt.Sprintf("codec-%d", rng.IntN(smallSpaceKeyRange))
		records[key] = sealed(key, payloadForVersion(mixedVersion(rng)))
	}
	return records
}

// boundaryFixture seals one record per version under boundary-<i> keys, the
// shape every version-boundary subtest round-trips.
func boundaryFixture(versions []version.RecordVersion) map[string]storage.Record {
	records := make(map[string]storage.Record, len(versions))
	for i, ver := range versions {
		key := fmt.Sprintf("boundary-%d", i)
		records[key] = sealed(key, payloadForVersion(ver))
	}
	return records
}

// equalLVersions pins the equal-l class: one l, counters 0..2, one node.
func equalLVersions(rng *rand.Rand) []version.RecordVersion {
	base := smallSpaceLBase + int64(rng.IntN(smallSpaceLRange))
	node := uint16(1 + rng.IntN(smallSpaceNodeRange))
	versions := make([]version.RecordVersion, smallSpaceLRange)
	for i := range versions {
		versions[i] = version.PackVersion(base, uint16(i), node)
	}
	return versions
}

// equalLCounterVersions pins the equal-l-equal-c class: one l, one counter,
// nodes 1..3 — the class only the node ID separates.
func equalLCounterVersions(rng *rand.Rand) []version.RecordVersion {
	base := smallSpaceLBase + int64(rng.IntN(smallSpaceLRange))
	counter := uint16(rng.IntN(smallSpaceCRange))
	versions := make([]version.RecordVersion, smallSpaceNodeRange)
	for i := range versions {
		versions[i] = version.PackVersion(base, counter, uint16(1+i))
	}
	return versions
}

// adjacentLVersions pins the adjacent-l class: l, l+1, l+2 with one counter
// and one node.
func adjacentLVersions(rng *rand.Rand) []version.RecordVersion {
	base := smallSpaceLBase + int64(rng.IntN(smallSpaceLRange))
	counter := uint16(rng.IntN(smallSpaceCRange))
	node := uint16(1 + rng.IntN(smallSpaceNodeRange))
	versions := make([]version.RecordVersion, smallSpaceLRange)
	for i := range versions {
		versions[i] = version.PackVersion(base+int64(i), counter, node)
	}
	return versions
}

// assertCodecRoundTrip asserts the full-map round trip over one generated map:
// Encode then Decode returns the same records with nothing discarded, and
// re-encoding the decode output reproduces the first bytes — byte stability
// through the round trip (PERS-01, D-06).
func assertCodecRoundTrip(t *testing.T, step int, records map[string]storage.Record) {
	t.Helper()
	dump, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("step %d Encode() error = %v, want nil", step, err)
	}
	decoded, discarded, err := persist.Decode(dump, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("step %d Decode() error = %v, want nil (replay with the property seed above)",
			step, err)
	}
	if discarded != 0 {
		t.Fatalf("step %d Decode() discarded = %d, want 0 (replay with the property seed above)",
			step, discarded)
	}
	if !recordsEqual(decoded, records) {
		t.Fatalf("step %d round trip changed the records (replay with the property seed above)", step)
	}
	again, err := persist.Encode(decoded, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("step %d re-Encode() error = %v, want nil", step, err)
	}
	if !bytes.Equal(again, dump) {
		t.Fatalf("step %d re-encode is not byte-identical (replay with the property seed above)", step)
	}
}

// TestCodec_roundTripProperty proves the full-map round-trip over 500 seeded
// random maps per boundary class, tombstones included: Encode then Decode
// returns a deep-equal map with nothing discarded, and re-encoding the decode
// output yields bytes identical to the first encode — byte stability through
// the round trip (PERS-01, D-06). The first subtest draws with TEST-15's
// mixed distribution (half the versions from the small (l, c, node) space);
// the other three pin the version-boundary classes the small space puts in
// reach, so equal-l and adjacent-l collisions are explored deliberately.
func TestCodec_roundTripProperty(t *testing.T) {
	t.Parallel()
	t.Run("mixed small-space and wide draws round-trip byte-stably", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range codecPropertySteps {
			assertCodecRoundTrip(t, step, mixedSpaceRecords(rng))
		}
	})
	t.Run("equal-l boundaries round-trip byte-stably", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range codecPropertySteps {
			assertCodecRoundTrip(t, step, boundaryFixture(equalLVersions(rng)))
		}
	})
	t.Run("equal-l-equal-c-different-node boundaries round-trip byte-stably", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range codecPropertySteps {
			assertCodecRoundTrip(t, step, boundaryFixture(equalLCounterVersions(rng)))
		}
	})
	t.Run("adjacent-l boundaries round-trip byte-stably", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range codecPropertySteps {
			assertCodecRoundTrip(t, step, boundaryFixture(adjacentLVersions(rng)))
		}
	})
}

// TestCodec_byteStability proves the deterministic shape of the dump:
// encoding the same map twice yields identical bytes, and the record lines
// come out sorted by key (D-06).
func TestCodec_byteStability(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	records := testutil.RandStore(rng).Records()
	first, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	second, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("second Encode() error = %v, want nil", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("encoding the same map twice yielded different bytes")
	}
	lines := strings.Split(strings.TrimSuffix(string(first), "\n"), "\n")
	keys := make([]string, 0, len(lines))
	for _, line := range lines[1:] {
		keys = append(keys, lineKey(t, line))
	}
	if !slices.IsSorted(keys) {
		t.Errorf("record lines are not sorted by key: %v", keys)
	}
}

// TestCodec_discardClasses proves the class separation of Pitfall 7 and the
// hostile-input classes of Pitfall 8: checksum-faulty, malformed-version
// and oversized records discard alone while their generation keeps loading
// (Decode's error stays nil for a generation whose only fault is a bad
// record checksum), a duplicate key is last-wins, an empty input is a
// generation-level fault, and an unknown format_version makes the whole
// generation unreadable to Decode.
func TestCodec_discardClasses(t *testing.T) {
	t.Parallel()
	victim := sealed("victim", storage.Record{Version: version.PackVersion(1002, 3, 1), Data: []byte("c-value")})
	tampered := rewriteLine(t, encodeRecords(t, map[string]storage.Record{"victim": victim}),
		"victim", func(rec *dumpLine) { rec.Checksum++ })
	malformed := rewriteLine(t, encodeRecords(t, map[string]storage.Record{"victim": victim}),
		"victim", func(rec *dumpLine) { rec.Version = strings.Repeat("z", len(rec.Version)) })
	oversize := bytes.Repeat([]byte("x"), storage.ValueMaxBytes+oversizeBy)
	bigVer := version.PackVersion(1003, 4, 1)
	oversized := encodeRecordsAt(t, storage.ValueMaxBytes+oversizeBy, map[string]storage.Record{
		"big": sealed("big", storage.Record{Version: bigVer, Data: oversize}),
	})
	tests := []struct {
		name          string
		dump          []byte
		wantDiscarded int
		wantDecodeErr bool
	}{
		{
			name:          "a tampered checksum discards only its record",
			dump:          tampered,
			wantDiscarded: 1,
		},
		{
			name:          "a non-hex record_version discards only its record",
			dump:          malformed,
			wantDiscarded: 1,
		},
		{
			name:          "data over the envelope discards only its record",
			dump:          oversized,
			wantDiscarded: 1,
		},
		{
			name:          "an unknown format_version is a generation-level fault",
			dump:          []byte("{\"format_version\":999}\n"),
			wantDecodeErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			decoded, discarded, err := persist.Decode(tt.dump, storage.ValueMaxBytes)
			if (err != nil) != tt.wantDecodeErr {
				t.Errorf("Decode() error = %v, want error=%t", err, tt.wantDecodeErr)
			}
			if tt.wantDecodeErr {
				return
			}
			if discarded != tt.wantDiscarded {
				t.Errorf("Decode() discarded = %d, want %d", discarded, tt.wantDiscarded)
			}
			if !recordsEqual(decoded, baseRecords()) {
				t.Errorf("Decode() records = %v, want the surviving base records", decoded)
			}
		})
	}
	t.Run("a duplicate key in a hand-edited dump is last-wins", func(t *testing.T) {
		t.Parallel()
		first := encodeRecords(t, map[string]storage.Record{
			"dup": sealed("dup", storage.Record{Version: version.PackVersion(1004, 1, 1), Data: []byte("first")}),
		})
		second := encodeRecords(t, map[string]storage.Record{
			"dup": sealed("dup", storage.Record{Version: version.PackVersion(1005, 1, 1), Data: []byte("second")}),
		})
		firstLines := strings.Split(strings.TrimSuffix(string(first), "\n"), "\n")
		secondLines := strings.Split(strings.TrimSuffix(string(second), "\n"), "\n")
		joined := []byte(strings.Join([]string{firstLines[0], firstLines[1], secondLines[1]}, "\n") + "\n")
		decoded, discarded, err := persist.Decode(joined, storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("Decode(joined) error = %v, want nil", err)
		}
		if discarded != 0 {
			t.Errorf("Decode(joined) discarded = %d, want 0", discarded)
		}
		want := map[string]storage.Record{
			"dup": sealed("dup", storage.Record{Version: version.PackVersion(1005, 1, 1), Data: []byte("second")}),
		}
		if !recordsEqual(decoded, want) {
			t.Errorf("Decode(joined) records = %v, want the last-wins record", decoded)
		}
	})
	t.Run("an empty input is a generation-level fault, not a valid empty dump", func(t *testing.T) {
		t.Parallel()
		_, _, err := persist.Decode(nil, storage.ValueMaxBytes)
		if err == nil {
			t.Error("Decode(nil) error = nil, want a generation-level error")
		}
	})
	t.Run("a valid empty map encodes to a header-only dump that reloads empty", func(t *testing.T) {
		t.Parallel()
		dump, err := persist.Encode(make(map[string]storage.Record), storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("Encode(empty) error = %v, want nil", err)
		}
		decoded, discarded, err := persist.Decode(dump, storage.ValueMaxBytes)
		if err != nil || discarded != 0 || len(decoded) != 0 {
			t.Errorf("Decode(empty dump) = (%d records, %d discarded, %v), want empty, 0, nil",
				len(decoded), discarded, err)
		}
	})
}

// TestWatermark_fullProperty proves the FULL watermark law (VERS-04): over
// 500 seeded random maps the watermark equals an independent field-wise
// max scan — the maps drawn with TEST-15's mixed distribution, so half the
// versions come from the small (l, c, node) space and equal-l boundaries are
// explored deliberately; the WR-01 fixture — a watermark record at (dumpMax,
// watermarkC) against an older-l record carrying a higher counter — yields
// the exact (l, c) pair and not l alone or the global max counter; the
// empty map yields (0, 0).
func TestWatermark_fullProperty(t *testing.T) {
	t.Parallel()
	t.Run("equals an independent max scan over seeded records", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range codecPropertySteps {
			records := mixedSpaceRecords(rng)
			var wantL int64
			var wantC uint16
			for _, rec := range records {
				l, c, _ := rec.Version.Unpack()
				if l > wantL || (l == wantL && c > wantC) {
					wantL, wantC = l, c
				}
			}
			gotL, gotC := persist.Watermark(records)
			if gotL != wantL || gotC != wantC {
				t.Fatalf("step %d Watermark() = (%d, %d), want (%d, %d) (replay with the property seed above)",
					step, gotL, gotC, wantL, wantC)
			}
		}
	})
	t.Run("the WR-01 fixture keeps the full pair against a higher counter at a lower l", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		dumpMax := int64(1_730_000_000_000) + rng.Int64N(1_000_000)
		watermarkC := uint16(1 + rng.IntN(4095))
		olderHighC := uint16(1 + rng.IntN(4095))
		records := map[string]storage.Record{
			"watermark": {Version: version.PackVersion(dumpMax, watermarkC, 1)},
			"older":     {Version: version.PackVersion(dumpMax-1, olderHighC, 2)},
			"same-l":    {Version: version.PackVersion(dumpMax, watermarkC-1, 3)},
		}
		gotL, gotC := persist.Watermark(records)
		if gotL != dumpMax || gotC != watermarkC {
			t.Errorf("Watermark(WR-01) = (%d, %d), want (%d, %d)",
				gotL, gotC, dumpMax, watermarkC)
		}
	})
	t.Run("the empty map is the wall-clock start", func(t *testing.T) {
		t.Parallel()
		gotL, gotC := persist.Watermark(make(map[string]storage.Record))
		if gotL != 0 || gotC != 0 {
			t.Errorf("Watermark(empty) = (%d, %d), want (0, 0)", gotL, gotC)
		}
	})
}

// TestWatermark_equalCounterDomain pins the equal-l half of the full
// watermark law (VERS-04): at the maximum l the greatest counter wins, and a
// higher counter at a lower l never does — the (100,5)+(100,7)+(99,4095)
// acceptance tuple yielding (100,7). Map iteration order is random per call,
// so the fixture is scanned repeatedly: every order must keep the exact pair,
// which is what a restart-seeded clock needs to strictly dominate versions
// stamped at the same l with different counters.
func TestWatermark_equalCounterDomain(t *testing.T) {
	t.Parallel()
	records := map[string]storage.Record{
		"same-l-low-c":  sealed("same-l-low-c", storage.Record{Version: version.PackVersion(100, 5, 1), Data: []byte("low-c")}),
		"same-l-high-c": sealed("same-l-high-c", storage.Record{Version: version.PackVersion(100, 7, 1), Data: []byte("high-c")}),
		"lower-l-decoy": sealed("lower-l-decoy", storage.Record{Version: version.PackVersion(99, 4095, 1), Data: []byte("decoy")}),
	}
	for range watermarkOrderScans {
		gotL, gotC := persist.Watermark(records)
		if gotL != 100 || gotC != 7 {
			t.Fatalf("Watermark(equal-l fixture) = (%d, %d), want (100, 7) under every iteration order", gotL, gotC)
		}
	}
}

// TestCodec_truncatedGenerationUnreadable pins the generation-level fault
// class for a torn tail (PERS-02, D-02): a dump truncated mid-record is
// unreadable as a whole generation — Decode rejects it, so Load's fallback
// chain advances to .prev instead of serving the torn bytes — and the
// intact generation stays the control.
func TestCodec_truncatedGenerationUnreadable(t *testing.T) {
	t.Parallel()
	dump := encodeRecords(t, map[string]storage.Record{
		"tail": sealed("tail", storage.Record{Version: version.PackVersion(1006, 6, 1), Data: []byte("tail-value")}),
	})
	torn := dump[:len(dump)-tornTailCut]
	_, _, err := persist.Decode(torn, storage.ValueMaxBytes)
	if err == nil {
		t.Error("Decode(torn) error = nil, want a generation-level error, not the records before the tear")
	}
	_, _, err = persist.Decode(dump, storage.ValueMaxBytes)
	if err != nil {
		t.Errorf("Decode(intact) error = %v, want nil", err)
	}
}
