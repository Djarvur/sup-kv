package persist_test

// The dump-codec fuzz target (TEST-08, D-08): FuzzDumpDecode feeds arbitrary
// bytes to persist.Decode and asserts the codec's documented contract only —
// never anything stronger than the code guarantees today:
//
//   - Decode never panics on any input (a panic is a fuzz failure);
//   - a generation-level fault returns an error with no records and no
//     discarded count (Decode returns nil, 0, err — the fallback-chain
//     signal), so a torn record can never collapse into empty success;
//   - a per-record fault only increments discarded: success still returns
//     checksum-verified records, accepted lines are accounted for
//     (len(records)+discarded <= parsed record lines), and every returned
//     record re-encodes and round-trips unchanged;
//   - a returned tombstone (Deleted=true) carries Data only when the input
//     itself carried Data under that key — the codec echoes, it never
//     invents payloads for deletions (XPORT-10's current tombstone-with-data
//     behavior included, but not extended).
//
// The committed seed corpus under testdata/fuzz/FuzzDumpDecode/ derives only
// from persist.Encode over in-repo fixtures and their corruptions: a valid
// multi-record dump with a tombstone, a header-only dump, an empty input, an
// unsupported format version, a bad-checksum line, a torn final record, a
// dump cut mid-line, and a flipped payload byte. Plain `go test` loads and
// runs the corpus (the PR-CI smoke); deep coverage-guided runs happen on the
// nightly schedule (plan 02-11) and always bound the run with -fuzztime
// (RESEARCH Pitfall 4). One fuzz target per package: FuzzDecode lives in
// internal/udprepl.

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// The fuzz seed identity and fixture constants: the version base, the dump
// header literal, and the fixture records' keys and payload.
const (
	fuzzDumpNowMs        = int64(1_730_000_000_000)
	fuzzDumpNodeID       = uint16(12)
	fuzzDumpHeader       = "{\"format_version\":1}\n"
	fuzzDumpDataKey      = "fuzz/data"
	fuzzDumpTombstoneKey = "fuzz/tombstone"
	fuzzDumpPayload      = "value"
)

// errFuzzDumpFixtureNoPayload is the flipped-byte construction guard: the
// valid seed dump must carry the fixture payload for the flip to represent
// its corruption class.
var errFuzzDumpFixtureNoPayload = errors.New("fuzz fixture bug: the valid seed dump does not carry the base64 payload to flip")

// FuzzDumpDecode is the dump codec's native fuzz target: the seed corpus
// covers the valid JSONL shape plus every failure class the codec
// distinguishes, and the body asserts the contract invariants only.
func FuzzDumpDecode(f *testing.F) {
	seeds, err := fuzzDumpSeeds()
	if err != nil {
		f.Fatalf("build the dump seed corpus: %v", err)
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, dump []byte) {
		records, stats, err := persist.Decode(dump, storage.ValueMaxBytes)
		if err != nil {
			if records != nil {
				t.Fatalf("Decode returned %d records alongside the generation-level fault: %v", len(records), err)
			}
			if stats.Total() != 0 {
				t.Fatalf("Decode returned discarded=%d alongside the generation-level fault: %v", stats.Total(), err)
			}
			return
		}
		lines, deletionsWithData := fuzzDumpScan(dump)
		if stats.Total() < 0 || stats.Total() > lines {
			t.Fatalf("discarded=%d outside 0..%d parsed record lines", stats.Total(), lines)
		}
		if len(records)+stats.Total() > lines {
			t.Fatalf("accepted %d records + %d discarded > %d parsed record lines", len(records), stats.Total(), lines)
		}
		for key, rec := range records {
			if storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted) != rec.Checksum {
				t.Fatalf("Decode accepted record %q whose checksum does not verify", key)
			}
			if rec.Deleted && len(rec.Data) > 0 && !deletionsWithData[key] {
				t.Fatalf("Decode invented %d data bytes on tombstone %q: no input line carried Data under that key", len(rec.Data), key)
			}
		}
		fuzzDumpAssertEcho(t, records)
	})
}

// fuzzDumpAssertEcho proves the codec's echo contract: re-encoding accepted
// records and decoding the result yields the same records, checksums and
// flags, with nothing discarded — the one round-trip property the codec
// guarantees for every record it accepted.
func fuzzDumpAssertEcho(t *testing.T, records map[string]storage.Record) {
	t.Helper()
	again, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("re-Encode of %d accepted records: %v", len(records), err)
	}
	back, backStats, err := persist.Decode(again, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("re-Decode of the re-encoded dump: %v", err)
	}
	if backStats.Total() != 0 {
		t.Fatalf("re-Decode of the re-encoded dump discarded %d records, want none", backStats.Total())
	}
	if len(back) != len(records) {
		t.Fatalf("round trip kept %d of %d accepted records", len(back), len(records))
	}
	for key, rec := range records {
		got, ok := back[key]
		if !ok {
			t.Fatalf("round trip lost record %q", key)
		}
		if got.Version != rec.Version || got.Deleted != rec.Deleted || got.Checksum != rec.Checksum || !bytes.Equal(got.Data, rec.Data) {
			t.Fatalf("round trip changed record %q: (version %v, deleted %t, checksum %d, %d data bytes) became (version %v, deleted %t, checksum %d, %d data bytes)",
				key, rec.Version, rec.Deleted, rec.Checksum, len(rec.Data), got.Version, got.Deleted, got.Checksum, len(got.Data))
		}
	}
}

// fuzzDumpSeeds builds the seed payloads from the production encoder over
// in-repo fixtures and from the corruptions the codec distinguishes, with a
// construction guard on every corrupted shape so a fixture that stopped
// representing its class cannot silently weaken the corpus. It is a plain
// function rather than a *testing.F method so the same payloads drive both
// the fuzz target and the committed testdata corpus files.
func fuzzDumpSeeds() ([][]byte, error) {
	ver := version.PackVersion(fuzzDumpNowMs, 1, fuzzDumpNodeID)
	valid, err := persist.Encode(map[string]storage.Record{
		fuzzDumpDataKey: {
			Version:  ver,
			Data:     []byte(fuzzDumpPayload),
			Checksum: storage.RecordChecksum(fuzzDumpDataKey, ver, []byte(fuzzDumpPayload), false),
		},
		fuzzDumpTombstoneKey: {
			Version:  ver,
			Deleted:  true,
			Checksum: storage.RecordChecksum(fuzzDumpTombstoneKey, ver, nil, true),
		},
	}, storage.ValueMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("encode valid seed dump: %w", err)
	}

	dataChecksum := storage.RecordChecksum(fuzzDumpDataKey, ver, []byte(fuzzDumpPayload), false)
	validLine := fuzzDumpRecord(fuzzDumpDataKey, ver, []byte(fuzzDumpPayload), false, dataChecksum)
	badChecksumLine := fuzzDumpRecord(fuzzDumpDataKey, ver, []byte(fuzzDumpPayload), false, 0xDEADBEEF)

	flipped, ok := fuzzDumpFlipPayloadByte(valid)
	if !ok {
		return nil, errFuzzDumpFixtureNoPayload
	}

	return [][]byte{
		valid,
		[]byte(fuzzDumpHeader),
		{},
		[]byte("{\"format_version\":2}\n"),
		fuzzDumpJoin([]byte(fuzzDumpHeader), badChecksumLine),
		fuzzDumpJoin([]byte(fuzzDumpHeader), validLine, []byte(`{"key":"fuzz/torn"`)),
		bytes.Clone(valid[:len(valid)-10]),
		flipped,
	}, nil
}

// fuzzDumpRecord renders one JSONL record line in the on-disk format
// (record_version as 32 hex chars, data base64 via encoding/json) with an
// explicit checksum — the corruption rows supply a checksum that does not
// cover the line.
func fuzzDumpRecord(key string, ver version.RecordVersion, data []byte, deleted bool, checksum uint32) []byte {
	// errchkjson proves this Marshal cannot fail: the struct holds only a
	// string, a string, base64 []byte, a bool and a uint32.
	line, _ := json.Marshal(fuzzDumpLine{
		Key:      key,
		Version:  hex.EncodeToString(ver[:]),
		Data:     data,
		Deleted:  deleted,
		Checksum: checksum,
	})
	return append(line, '\n')
}

// fuzzDumpLine mirrors the on-disk record line (codec.go's recordLine) for
// fixture construction; the fuzz body never decodes through it.
type fuzzDumpLine struct {
	Key      string `json:"key"`
	Version  string `json:"record_version"`
	Data     []byte `json:"data"`
	Deleted  bool   `json:"deleted"`
	Checksum uint32 `json:"checksum"`
}

// fuzzDumpJoin concatenates dump fragments into one fresh slice.
func fuzzDumpJoin(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

// fuzzDumpFlipPayloadByte flips one byte inside the valid dump's base64
// payload — still valid JSON and valid base64, no longer the bytes the
// checksum covers — so the record faults per-record (discard, not a
// generation error). The base64 spelling of the fixture payload locates the
// spot; the guard proves the flip changes the decoded payload (a flip in the
// final padded quantum's last character only feeds a dropped byte), and ok
// is false when the fixture is already corrupt.
func fuzzDumpFlipPayloadByte(dump []byte) (flipped []byte, ok bool) {
	target := []byte("dmFsdWU=") // base64 of fuzzDumpPayload
	at := bytes.Index(dump, target)
	if at < 0 {
		return dump, false
	}
	flipped = bytes.Clone(dump)
	flipped[at] ^= 0x01 // 'd' -> 'e': the first payload byte's bit source
	changed, ok := fuzzDumpPayloadChanged(target, flipped[at:at+len(target)])
	if !ok || !changed {
		return dump, false
	}
	return flipped, true
}

// fuzzDumpPayloadChanged reports whether the two base64 tokens decode to
// different bytes (ok is false when either token is not valid base64).
func fuzzDumpPayloadChanged(before, after []byte) (changed, ok bool) {
	was, err := base64.StdEncoding.DecodeString(string(before))
	if err != nil {
		return false, false
	}
	now, err := base64.StdEncoding.DecodeString(string(after))
	if err != nil {
		return false, false
	}
	return !bytes.Equal(was, now), true
}

// fuzzDumpScan mirrors the decoder's JSON stream one value at a time: it
// counts parsed record lines and records which keys ever carried
// Deleted=true alongside non-empty Data. Production's last-wins duplicate
// handling and per-record discards make the union over all lines the sound
// direction — an accepted tombstone may carry Data if any input line for
// that key did. Lines the production decoder would reject end the scan,
// exactly where it stops.
func fuzzDumpScan(dump []byte) (lines int, deletionsWithData map[string]bool) {
	deletionsWithData = make(map[string]bool)
	dec := json.NewDecoder(bytes.NewReader(dump))
	var h struct {
		FormatVersion int `json:"format_version"`
	}
	err := dec.Decode(&h)
	if err != nil {
		return 0, deletionsWithData
	}
	for {
		var line struct {
			Key     string `json:"key"`
			Data    []byte `json:"data"`
			Deleted bool   `json:"deleted"`
		}
		err = dec.Decode(&line)
		if err != nil {
			return lines, deletionsWithData
		}
		lines++
		if line.Deleted && len(line.Data) > 0 {
			deletionsWithData[line.Key] = true
		}
	}
}
