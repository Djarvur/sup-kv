package persist

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// dumpFormatVersion is the on-disk dump format this package writes and
// reads; a generation announcing any other version is unreadable and the
// fallback chain advances (research A4).
const dumpFormatVersion = 1

// maxRecordBytes is the design envelope from docs/IDEA.md — the codec's
// decode-side size bound: a record whose Data exceeds it is discarded, the
// config-shrink class (research Open Question 5).
const maxRecordBytes = 61440

// versionBytes is the fixed size of a version.RecordVersion — the byte
// length a dump line's record_version hex string must decode to exactly.
const versionBytes = 16

// errUnsupportedFormat is the generation-level fault for a dump announcing
// a format version this codec does not read: the fallback chain advances
// (research A4).
var errUnsupportedFormat = errors.New("persist: dump format version unsupported")

// header is the first JSON line of a dump: the format version alone, since
// anything volatile like a timestamp would break D-06's byte-stability
// across generations of the same state.
type header struct {
	FormatVersion int `json:"format_version"`
}

// recordLine is one record per JSON line (PERS-01): the version spelled as
// a 32-char hex string — the [16]byte pitfall, since encoding/json would
// emit a JSON number array, never a string — the data as base64 ([]byte's
// automatic encoding) and the record's storage.RecordChecksum, the one and
// only checksum domain (D-07).
type recordLine struct {
	Key      string `json:"key"`
	Version  string `json:"record_version"`
	Data     []byte `json:"data"`
	Deleted  bool   `json:"deleted"`
	Checksum uint32 `json:"checksum"`
}

// Encode writes records as the JSONL dump (PERS-01, D-06): the header
// object on line 1, then one record object per line sorted by key — the
// sort is mandatory because Go map iteration order is random and byte
// stability across generations of the same state is the contract.
// SetEscapeHTML(false) keeps keys containing ampersand, less-than and
// greater-than literal and greppable.
func Encode(records map[string]storage.Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(header{FormatVersion: dumpFormatVersion})
	if err != nil {
		return nil, err
	}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		rec := records[key]
		line := recordLine{
			Key:      key,
			Version:  hex.EncodeToString(rec.Version[:]),
			Data:     rec.Data,
			Deleted:  rec.Deleted,
			Checksum: rec.Checksum,
		}
		err = enc.Encode(line)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// Decode reads a JSONL dump produced by Encode and keeps the three failure
// classes distinct (Pitfall 7): a truncated, structurally unparseable or
// empty generation, or one announcing an unknown format_version, is a
// generation-level error — the fallback chain advances (D-02); a per-record
// fault — a version that is not exactly 16 bytes of hex, Data over
// maxRecordBytes, or a checksum mismatch against storage.RecordChecksum —
// discards only that record and counts it in discarded, so one torn record
// never collapses its generation to the empty start (PERS-02). A duplicate
// key in a hand-edited file is last-wins. The returned records own their
// Data slices.
func Decode(dump []byte) (records map[string]storage.Record, discarded int, err error) {
	dec := json.NewDecoder(bytes.NewReader(dump))
	var h header
	err = dec.Decode(&h)
	if err != nil {
		return nil, 0, fmt.Errorf("persist: dump header unreadable: %w", err)
	}
	if h.FormatVersion != dumpFormatVersion {
		return nil, 0, errUnsupportedFormat
	}
	records = make(map[string]storage.Record)
	for {
		var line recordLine
		err = dec.Decode(&line)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return records, discarded, nil
			}
			return nil, 0, fmt.Errorf("persist: dump record unreadable: %w", err)
		}
		rec, ok := decodeRecord(line)
		if !ok {
			discarded++
			continue
		}
		records[line.Key] = rec
	}
}

// decodeRecord validates one decoded line against the record domain and
// returns the storage record it carries with defensively copied Data; a
// hex-undecodable or wrong-length version, Data over maxRecordBytes or a
// checksum mismatch discards the record (ok false). It never panics on any
// bit pattern: [hex.DecodeString] is total, Unpack is total and the
// checksum comparison is arithmetic (T-3-01).
func decodeRecord(line recordLine) (storage.Record, bool) {
	raw, err := hex.DecodeString(line.Version)
	if err != nil || len(raw) != versionBytes {
		return storage.Record{}, false
	}
	var ver version.RecordVersion
	copy(ver[:], raw)
	if len(line.Data) > maxRecordBytes {
		return storage.Record{}, false
	}
	if storage.RecordChecksum(line.Key, ver, line.Data, line.Deleted) != line.Checksum {
		return storage.Record{}, false
	}
	return storage.Record{
		Version:  ver,
		Data:     bytes.Clone(line.Data),
		Deleted:  line.Deleted,
		Checksum: line.Checksum,
	}, true
}

// Parse is the readability oracle the engine hands Load: it accepts a
// generation whose structure reads, reporting only generation-level faults.
// A checksum-faulty record does NOT fail Parse — it discards silently in
// Decode — so a torn record never collapses its whole generation
// (Pitfall 7's class separation, D-02).
func Parse(dump []byte) error {
	_, _, err := Decode(dump)
	return err
}

// Watermark returns the FULL dump watermark (VERS-04): the maximum l over
// the records plus the maximum counter at that l — not max l alone — so a
// post-restart clock seeded with it emits versions strictly dominating
// every version in the records. The empty map returns (0, 0) and NewClock
// starts from the wall clock.
func Watermark(records map[string]storage.Record) (initialL int64, initialC uint16) {
	for _, rec := range records {
		l, c, _ := rec.Version.Unpack()
		switch {
		case l > initialL:
			initialL, initialC = l, c
		case l == initialL && c > initialC:
			initialC = c
		}
	}
	return initialL, initialC
}
