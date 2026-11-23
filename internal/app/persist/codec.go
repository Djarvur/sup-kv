package persist

import (
	"bufio"
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
// fallback chain advances (research A4). It stays 1 while record_count is
// added: a version bump would make a downgrade reject new dumps, whereas an
// old reader ignores the unknown additive field (D-03).
const dumpFormatVersion = 1

// versionBytes is the fixed size of a version.RecordVersion — the byte
// length a dump line's record_version hex string must decode to exactly.
const versionBytes = 16

// Format-bound reader constants (T-5-03). maxDumpValueBytes is the
// documented datagram ceiling (XPORT-03): the transport adapter enforces it,
// and persist names the number here rather than importing the transport
// (the app core may not). base64PaddingBytes, base64GroupBytes and
// base64SymbolBytes are []byte's JSON encoding — ceil(n/3)*4 base64
// characters. jsonEscapeExpansion is the worst-case JSON escaping of a legal
// storage.MaxKeyBytes key: every byte can become the six-byte \u00XX form (a
// quote doubles; control and invalid-UTF-8 bytes expand sixfold).
// dumpLineOverheadBytes is the fixed field and separator overhead, and
// dumpLineDelimiterBytes is the newline the reader reserves room for. Every
// literal is named, so the strict lint config's mnd has nothing to flag.
const (
	maxDumpValueBytes      = 65507
	base64PaddingBytes     = 2
	base64GroupBytes       = 3
	base64SymbolBytes      = 4
	jsonEscapeExpansion    = 6
	dumpLineOverheadBytes  = 256
	dumpLineDelimiterBytes = 1
)

// MaxDumpLineBytes is the longest record line the dump format can carry — a
// property of the format, never of the current valueMaxBytes configuration.
// It is the datagram ceiling's base64 expansion plus the worst-case JSON
// escaping of a storage.MaxKeyBytes key plus fixed overhead: no supported
// writer can emit a longer line at any accepted configuration, so a longer
// one is a generation-level fault, while a record over the configured limit
// inside a readable line is a ValueLimit discard. Because the bound takes no
// configuration input, a dump written at the default limit still reads under
// a tiny new limit (T-5-03).
//
// It is also the page-token cursor domain (WR-01): a key rides inside one
// record line, so no readable dump can name a longer key, and the store
// keeps serving pre-upgrade keys above storage.MaxKeyBytes (validation is
// write-side only). A transport that validates a page token against a key
// bound must therefore use this format bound, not the write ceiling.
const MaxDumpLineBytes = (maxDumpValueBytes+base64PaddingBytes)/base64GroupBytes*base64SymbolBytes +
	jsonEscapeExpansion*storage.MaxKeyBytes + dumpLineOverheadBytes

// maxDumpLineBytes returns MaxDumpLineBytes for the reader's buffer size.
func maxDumpLineBytes() int { return MaxDumpLineBytes }

// errUnsupportedFormat is the generation-level fault for a dump announcing
// a format version this codec does not read: the fallback chain advances
// (research A4).
var errUnsupportedFormat = errors.New("persist: dump format version unsupported")

// errIncompleteDump is the generation-level fault for a v1.1 dump whose
// record lines are fewer than the record_count its header announces: the
// generation was cut at a line boundary, which is indistinguishable from a
// shorter complete file without the census (PERSIST-03, D-03).
var errIncompleteDump = errors.New("persist: dump generation incomplete")

// errDumpLineTooLong is the generation-level fault for a record line longer
// than maxDumpLineBytes: no supported writer can emit one, so it is not a
// legal record to isolate (T-5-03).
var errDumpLineTooLong = errors.New("persist: dump line exceeds the format maximum")

// errTornTail is the generation-level fault for an unterminated final
// fragment that does not parse: a mid-line cut must not collapse into a
// shorter successful load (T-5-04).
var errTornTail = errors.New("persist: dump tail is an unreadable fragment")

// header is the first JSON line of a dump: the format version and the
// additive record_count census (PERSIST-03, D-03). record_count is the
// number of record lines the writer emitted; a v1.1 reader treats fewer
// lines than the count as a generation-level fault (line-boundary
// truncation) and more lines as a tolerated hand edit. The field is
// optional and written only by this format generation: a v1.0 dump without
// record_count takes the documented legacy path, which has no completeness
// census. The header carries no timestamp or other volatile field, so
// D-06's byte stability across generations of the same state holds.
type header struct {
	FormatVersion int  `json:"format_version"`
	RecordCount   *int `json:"record_count,omitempty"`
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

// EncodeTo writes records as the JSONL dump to w, streaming one format line
// at a time: the header object first, then one record object per line sorted
// by key — the sort is mandatory because Go map iteration order is random and
// byte stability across generations of the same state is the contract.
// SetEscapeHTML(false) keeps keys containing ampersand, less-than and
// greater-than literal and greppable. Each line is written in one Write call,
// so a caller-supplied writer never has to hold the whole dump (D-06).
//
// valueMaxBytes is the configured value limit the store enforces
// (PERSIST-01): a record whose Data exceeds it is an invariant violation,
// never a silent skip — the encoder returns an error wrapping
// storage.ErrValueTooLarge naming the key. The tripwire sits over the
// store's own write check, so a confirmed write can never fail to encode for
// size.
func EncodeTo(w io.Writer, records map[string]storage.Record, valueMaxBytes int) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	count := len(records)
	err := enc.Encode(header{FormatVersion: dumpFormatVersion, RecordCount: &count})
	if err != nil {
		return err
	}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		rec := records[key]
		if len(rec.Data) > valueMaxBytes {
			return fmt.Errorf("persist: record %q is %d bytes, over the configured value limit %d: %w",
				key, len(rec.Data), valueMaxBytes, storage.ErrValueTooLarge)
		}
		line := recordLine{
			Key:      key,
			Version:  hex.EncodeToString(rec.Version[:]),
			Data:     rec.Data,
			Deleted:  rec.Deleted,
			Checksum: rec.Checksum,
		}
		err = enc.Encode(line)
		if err != nil {
			return err
		}
	}
	return nil
}

// Encode writes records as one JSONL dump buffer — EncodeTo's bytes wrapper
// for callers that need the whole generation in memory (the snapshot
// fixtures and tests). Production dump cycles stream through EncodeTo
// instead (D-06).
func Encode(records map[string]storage.Record, valueMaxBytes int) ([]byte, error) {
	var buf bytes.Buffer
	err := EncodeTo(&buf, records, valueMaxBytes)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeStats is the per-record discard census (PERSIST-10, PERS-02): one
// counter for each fault class a record line can take, kept separate so the
// load report and metrics can name the cause instead of one opaque total. A
// generation-level fault returns the zero value; the counters describe only
// lines the decoder read and discarded.
type DecodeStats struct {
	Checksum   int
	Version    int
	ValueLimit int
	Malformed  int
}

// Total returns the sum of the four discard classes.
func (s DecodeStats) Total() int {
	return s.Checksum + s.Version + s.ValueLimit + s.Malformed
}

// decodeClass is one record line's decode outcome: decodeAccepted (the zero
// value) or one of the four DecodeStats fault classes.
type decodeClass int

const (
	decodeAccepted decodeClass = iota
	decodeChecksum
	decodeVersion
	decodeValueLimit
	decodeMalformed
)

// add returns s with one line's outcome counted; an accepted line counts
// nothing.
func (s DecodeStats) add(class decodeClass) DecodeStats {
	switch class {
	case decodeChecksum:
		s.Checksum++
	case decodeVersion:
		s.Version++
	case decodeValueLimit:
		s.ValueLimit++
	case decodeMalformed:
		s.Malformed++
	case decodeAccepted:
	}
	return s
}

// Decode reads a JSONL dump produced by Encode and keeps the failure classes
// separate (Pitfall 7, PERSIST-03). A generation-level fault — an empty or
// header-unreadable input, an unknown format_version, a line longer than the
// format's maximum, an unterminated tail that does not parse, or a v1.1
// generation with fewer record lines than its record_count census — returns
// nil records, the zero DecodeStats and the error, so the fallback chain
// advances (D-02). A per-record fault discards only that record and counts
// it in the matching DecodeStats class: Malformed (the line is not a JSON
// object), Version (record_version is not 16 bytes of hex), ValueLimit (Data
// over the configured valueMaxBytes) or Checksum (the checksum does not
// cover the line). One torn record never collapses its generation to the
// empty start (PERS-02, PERSIST-03).
//
// Lines are read through a bounded [bufio.Reader] and parsed one at a time
// with [json.Unmarshal]: a streaming [json.Decoder] is permanently poisoned
// by a syntax error and loses every line after it. A duplicate key in a
// hand-edited file is last-wins. A legacy v1.0 dump with no record_count
// carries no completeness census — a truncation exactly at a line boundary
// of such a dump is the documented compatibility limitation — and its final
// unterminated tail still has to parse. The returned records own their Data
// slices.
//
// valueMaxBytes is the configured value limit (PERSIST-01) — the same
// Config.ValueLimit() value the store normalizes and enforces — and it is
// applied after the whole line is read, so the reader bound never shrinks
// with the configuration.
func Decode(dump []byte, valueMaxBytes int) (records map[string]storage.Record, stats DecodeStats, err error) {
	reader := bufio.NewReaderSize(bytes.NewReader(dump), maxDumpLineBytes()+dumpLineDelimiterBytes)
	headerLine, _, err := readDumpLine(reader)
	if err != nil {
		return nil, DecodeStats{}, fmt.Errorf("persist: dump header unreadable: %w", err)
	}
	var h header
	err = json.Unmarshal(headerLine, &h)
	if err != nil {
		return nil, DecodeStats{}, fmt.Errorf("persist: dump header unreadable: %w", err)
	}
	if h.FormatVersion != dumpFormatVersion {
		return nil, DecodeStats{}, errUnsupportedFormat
	}
	records = make(map[string]storage.Record)
	lines := 0
	for {
		line, terminated, readErr := readDumpLine(reader)
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, DecodeStats{}, fmt.Errorf("persist: dump record unreadable: %w", readErr)
		}
		lines++
		key, rec, class := decodeRecord(line, valueMaxBytes)
		if class == decodeMalformed && !terminated {
			return nil, DecodeStats{}, fmt.Errorf("persist: dump record unreadable: %w", errTornTail)
		}
		stats = stats.add(class)
		if class == decodeAccepted {
			records[key] = rec
		}
	}
	err = checkRecordCount(h.RecordCount, lines)
	if err != nil {
		return nil, DecodeStats{}, err
	}
	return records, stats, nil
}

// readDumpLine reads one line, bounded by the reader's format-sized buffer.
// terminated reports whether the line ended with a newline: an unterminated
// remainder is the dump's final fragment, which the caller accepts only when
// it parses as a record. An overlong line is a generation-level fault
// ([bufio.ErrBufferFull] — a line no supported writer can emit), and [io.EOF]
// with an empty remainder is the clean end of the stream.
func readDumpLine(reader *bufio.Reader) (line []byte, terminated bool, err error) {
	segment, err := reader.ReadSlice('\n')
	switch {
	case err == nil:
		return trimLineEnding(segment), true, nil
	case errors.Is(err, io.EOF):
		if len(segment) == 0 {
			return nil, false, io.EOF
		}
		return trimLineEnding(segment), false, nil
	case errors.Is(err, bufio.ErrBufferFull):
		return nil, false, fmt.Errorf("persist: dump line exceeds %d bytes: %w",
			maxDumpLineBytes(), errDumpLineTooLong)
	default:
		return nil, false, err
	}
}

// trimLineEnding strips one trailing newline and one preceding carriage
// return, so a CRLF-terminated dump reads the same as an LF one.
func trimLineEnding(segment []byte) []byte {
	segment = bytes.TrimSuffix(segment, []byte("\n"))
	return bytes.TrimSuffix(segment, []byte("\r"))
}

// checkRecordCount enforces the v1.1 completeness census (D-03): when the
// header carries record_count and the generation has fewer record lines than
// it announces, the generation is incomplete — otherwise a cut exactly at a
// line boundary is indistinguishable from a shorter complete file — so it is
// a generation-level fault carrying both numbers and the fallback chain
// advances. A nil count is the legacy v1.0 path with no census, and more
// lines than the count is tolerated because hand edits and duplicate keys
// parse normally.
func checkRecordCount(count *int, lines int) error {
	if count == nil || lines >= *count {
		return nil
	}
	return fmt.Errorf("persist: dump generation incomplete: %d of %d record lines: %w",
		lines, *count, errIncompleteDump)
}

// decodeRecord parses and validates one line and returns the key it carries,
// the storage record it declares with defensively copied Data, and the class
// the line fell into. The classes are the four DecodeStats counters:
// decodeMalformed for a line that is not a JSON object, decodeVersion for a
// record_version that is not exactly 16 bytes of hex, decodeValueLimit for
// Data over valueMaxBytes and decodeChecksum for a checksum mismatch.
// decodeAccepted is the zero value. It never panics on any bit pattern:
// [json.Unmarshal] is total, [hex.DecodeString] is total, Unpack is total
// and the checksum comparison is arithmetic (T-3-01).
func decodeRecord(line []byte, valueMaxBytes int) (key string, rec storage.Record, class decodeClass) {
	var parsed recordLine
	err := json.Unmarshal(line, &parsed)
	if err != nil {
		return "", storage.Record{}, decodeMalformed
	}
	raw, err := hex.DecodeString(parsed.Version)
	if err != nil || len(raw) != versionBytes {
		return parsed.Key, storage.Record{}, decodeVersion
	}
	var ver version.RecordVersion
	copy(ver[:], raw)
	if len(parsed.Data) > valueMaxBytes {
		return parsed.Key, storage.Record{}, decodeValueLimit
	}
	if storage.RecordChecksum(parsed.Key, ver, parsed.Data, parsed.Deleted) != parsed.Checksum {
		return parsed.Key, storage.Record{}, decodeChecksum
	}
	return parsed.Key, storage.Record{
		Version:  ver,
		Data:     bytes.Clone(parsed.Data),
		Deleted:  parsed.Deleted,
		Checksum: parsed.Checksum,
	}, decodeAccepted
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
