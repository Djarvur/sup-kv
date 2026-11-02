package persist

import "github.com/Djarvur/sup-kv/internal/app/storage"

// dumpFormatVersion is the on-disk dump format this package writes and reads.
const dumpFormatVersion = 1

// maxRecordBytes is the design envelope from docs/IDEA.md — the codec's
// decode-side size bound for discarded records.
const maxRecordBytes = 61440

// Encode writes records as a byte-stable JSONL dump sorted by key (stub).
func Encode(_ map[string]storage.Record) ([]byte, error) { return nil, nil }

// Decode reads a JSONL dump, discarding per-record faults (stub).
func Decode(_ []byte) (records map[string]storage.Record, discarded int, err error) {
	return make(map[string]storage.Record), 0, nil
}

// Parse is the readability oracle Load consumes (stub).
func Parse(_ []byte) error { return nil }

// Watermark computes the full (max l, max counter at that l) reading (stub).
func Watermark(_ map[string]storage.Record) (initialL int64, initialC uint16) {
	return 0, 0
}
