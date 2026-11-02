package persist_test

import (
	"bytes"
	"encoding/json"
	"maps"
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
	maxRecordBytes     = 61440
	oversizeBy         = 1
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

// encodeRecords merges base with extras and encodes it.
func encodeRecords(t *testing.T, extras map[string]storage.Record) []byte {
	t.Helper()
	records := baseRecords()
	maps.Copy(records, extras)
	dump, err := persist.Encode(records)
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	return dump
}

// TestCodec_roundTripProperty proves the full-map round-trip over 500
// seeded random stores, tombstones included: Encode then Decode returns a
// deep-equal map with nothing discarded, and re-encoding the decode output
// yields bytes identical to the first encode — byte stability through the
// round trip (PERS-01, D-06).
func TestCodec_roundTripProperty(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	for step := range codecPropertySteps {
		records := testutil.RandStore(rng).Records()
		dump, err := persist.Encode(records)
		if err != nil {
			t.Fatalf("step %d Encode() error = %v, want nil", step, err)
		}
		decoded, discarded, err := persist.Decode(dump)
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
		again, err := persist.Encode(decoded)
		if err != nil {
			t.Fatalf("step %d re-Encode() error = %v, want nil", step, err)
		}
		if !bytes.Equal(again, dump) {
			t.Fatalf("step %d re-encode is not byte-identical (replay with the property seed above)", step)
		}
	}
}

// TestCodec_byteStability proves the deterministic shape of the dump:
// encoding the same map twice yields identical bytes, and the record lines
// come out sorted by key (D-06).
func TestCodec_byteStability(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	records := testutil.RandStore(rng).Records()
	first, err := persist.Encode(records)
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	second, err := persist.Encode(records)
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
// and oversized records discard alone while their generation keeps loading,
// Parse accepts a generation whose only fault is a bad record checksum, a
// duplicate key is last-wins, an empty input is a generation-level fault,
// and an unknown format_version makes the whole generation unreadable to
// both Decode and Parse.
func TestCodec_discardClasses(t *testing.T) {
	t.Parallel()
	victim := sealed("victim", storage.Record{Version: version.PackVersion(1002, 3, 1), Data: []byte("c-value")})
	tampered := rewriteLine(t, encodeRecords(t, map[string]storage.Record{"victim": victim}),
		"victim", func(rec *dumpLine) { rec.Checksum++ })
	malformed := rewriteLine(t, encodeRecords(t, map[string]storage.Record{"victim": victim}),
		"victim", func(rec *dumpLine) { rec.Version = strings.Repeat("z", len(rec.Version)) })
	oversize := bytes.Repeat([]byte("x"), maxRecordBytes+oversizeBy)
	bigVer := version.PackVersion(1003, 4, 1)
	oversized := encodeRecords(t, map[string]storage.Record{
		"big": sealed("big", storage.Record{Version: bigVer, Data: oversize}),
	})
	tests := []struct {
		name          string
		dump          []byte
		wantDiscarded int
		wantParseErr  bool
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
			wantParseErr:  true,
			wantDecodeErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := persist.Parse(tt.dump)
			if (err != nil) != tt.wantParseErr {
				t.Errorf("Parse() error = %v, want error=%t", err, tt.wantParseErr)
			}
			decoded, discarded, err := persist.Decode(tt.dump)
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
		decoded, discarded, err := persist.Decode(joined)
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
		err := persist.Parse(nil)
		if err == nil {
			t.Error("Parse(nil) = nil, want a generation-level error")
		}
	})
	t.Run("a valid empty map encodes to a header-only dump that reloads empty", func(t *testing.T) {
		t.Parallel()
		dump, err := persist.Encode(make(map[string]storage.Record))
		if err != nil {
			t.Fatalf("Encode(empty) error = %v, want nil", err)
		}
		decoded, discarded, err := persist.Decode(dump)
		if err != nil || discarded != 0 || len(decoded) != 0 {
			t.Errorf("Decode(empty dump) = (%d records, %d discarded, %v), want empty, 0, nil",
				len(decoded), discarded, err)
		}
	})
}

// TestWatermark_fullProperty proves the FULL watermark law (VERS-04): over
// 500 seeded random stores the watermark equals an independent field-wise
// max scan; the WR-01 fixture — a watermark record at (dumpMax,
// watermarkC) against an older-l record carrying a higher counter — yields
// the exact (l, c) pair and not l alone or the global max counter; the
// empty map yields (0, 0).
func TestWatermark_fullProperty(t *testing.T) {
	t.Parallel()
	t.Run("equals an independent max scan over seeded records", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		for step := range codecPropertySteps {
			records := testutil.RandStore(rng).Records()
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
