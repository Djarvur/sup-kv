package persist_test

// AUDIT PROBE (scratch probe copy only): fuzz harness for the dump decoder.

import (
	"bytes"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

func FuzzZZDumpDecode(f *testing.F) {
	ver := version.PackVersion(1_700_000_000_000, 7, 3)
	recs := map[string]storage.Record{
		"a": {Version: ver, Data: []byte("v"), Checksum: storage.RecordChecksum("a", ver, []byte("v"), false)},
		"b": {Version: ver, Deleted: true, Checksum: storage.RecordChecksum("b", ver, nil, true)},
	}
	seed, err := persist.Encode(recs)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte("{\"format_version\":1}\n"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		got, discarded, err := persist.Decode(data)
		if err != nil {
			if got != nil || discarded != 0 {
				t.Fatalf("Decode error with non-zero results")
			}
			return
		}
		for key, rec := range got {
			if storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted) != rec.Checksum {
				t.Fatalf("Decode accepted a record whose checksum does not verify: key %q", key)
			}
		}
		again, err := persist.Encode(got)
		if err != nil {
			t.Fatalf("re-Encode failed: %v", err)
		}
		back, disc2, err := persist.Decode(again)
		if err != nil || disc2 != 0 || len(back) != len(got) {
			t.Fatalf("round trip of accepted records failed: %v discarded=%d %d/%d", err, disc2, len(back), len(got))
		}
		for key, rec := range got {
			b := back[key]
			if b.Version != rec.Version || b.Deleted != rec.Deleted || !bytes.Equal(b.Data, rec.Data) {
				t.Fatalf("round trip changed record %q", key)
			}
		}
		_, _ = persist.Watermark(got)
	})
}
