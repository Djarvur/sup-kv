package persist

// White-box tests for the streaming encoder: they live in package persist
// because the max-single-write bound is the unexported format property
// maxDumpLineBytes(), and the _internal_test.go suffix is the configured
// testpackage exemption (the same convention as
// engine_replay_internal_test.go).

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// encodeStreamRecords returns the seeded map both streaming tests encode: a
// tombstone, keys that exercise the HTML-escaping switch (ampersand, less-than
// and quote stay literal under SetEscapeHTML(false)), and a multi-kilobyte
// value, so the write-shape assertion covers more than the header plus small
// lines.
func encodeStreamRecords() map[string]storage.Record {
	return map[string]storage.Record{
		"alpha": {
			Version: version.PackVersion(1000, 1, 7),
			Data:    []byte("alpha-value"),
		},
		"a&b<c": {
			Version: version.PackVersion(1001, 1, 7),
			Data:    []byte(`html <tag> & "quote"`),
		},
		"tomb": {
			Version: version.PackVersion(1002, 1, 7),
			Deleted: true,
		},
		"z/" + strings.Repeat("k", 64): {
			Version: version.PackVersion(1003, 2, 7),
			Data:    bytes.Repeat([]byte("v"), 4096),
		},
	}
}

// recordingWriter records every Write call separately — the streamed shape
// oracle: one Write per format line, never one whole-dump write.
type recordingWriter struct {
	writes [][]byte
}

// Write records a copy of p and reports the full length.
func (r *recordingWriter) Write(p []byte) (int, error) {
	r.writes = append(r.writes, bytes.Clone(p))
	return len(p), nil
}

// TestEncodeTo_byteIdentical proves EncodeTo reproduces Encode's bytes
// exactly — header line, sorted record lines, tombstones and the
// SetEscapeHTML(false) spelling included — so the streaming path is a
// drop-in replacement for the buffered one (D-06, PERSIST-06).
func TestEncodeTo_byteIdentical(t *testing.T) {
	t.Parallel()
	records := encodeStreamRecords()
	want, err := Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	var got bytes.Buffer
	err = EncodeTo(&got, records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("EncodeTo() error = %v, want nil", err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Errorf("EncodeTo() bytes differ from Encode():\n got %q\nwant %q", got.Bytes(), want)
	}
}

// TestEncodeTo_maxSingleWrite proves the streamed shape (D-06, T-5-10): the
// encoder issues one Write per format line — a header plus one per record —
// and no single Write exceeds the format's longest legal line
// (maxDumpLineBytes), so a dump is never materialized as a second full
// buffer. The joined writes still equal Encode's bytes, so a writer that
// silently drops data cannot pass the bound vacuously.
func TestEncodeTo_maxSingleWrite(t *testing.T) {
	t.Parallel()
	records := encodeStreamRecords()
	var rec recordingWriter
	err := EncodeTo(&rec, records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("EncodeTo() error = %v, want nil", err)
	}
	if len(rec.writes) != len(records)+1 {
		t.Fatalf("EncodeTo() issued %d writes, want %d (one header plus one per record)",
			len(rec.writes), len(records)+1)
	}
	bound := maxDumpLineBytes()
	for i, write := range rec.writes {
		if len(write) > bound {
			t.Errorf("write %d is %d bytes, over the format maximum %d", i, len(write), bound)
		}
	}
	want, err := Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	if got := bytes.Join(rec.writes, nil); !bytes.Equal(got, want) {
		t.Errorf("joined streamed writes differ from Encode():\n got %q\nwant %q", got, want)
	}
}
