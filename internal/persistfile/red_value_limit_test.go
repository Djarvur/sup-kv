package persistfile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// P-01's (decision 14.11 / ADR 0011) engine-level constants: the review's
// F15 flag value — an ACCEPTED --value-max-bytes above the codec's frozen
// 61 440-byte decode bound — the reviewer's 63 000-byte record, and the
// scenario's key.
const (
	redP01ValueMaxBytes = 64000
	redP01ValueBytes    = 63000
	redP01EngineKey     = "red/p01/engine-value"
)

// TestRed_P01_valueLimitEngine pins review finding P-01 at engine level as
// an expected-red test (D-01): on the REAL composition path — persist.Open
// over a persistfile.FileStore and the real codec, no fake dump port
// anywhere — a 63 000-byte value is accepted by the store
// (storage.Config.ValueMaxBytes=64000), written to a dump and confirmed to
// the caller (PERS-04). Reopening the same directory must return the full
// value, but the codec's own maxRecordBytes = 61440 rejects it at dump
// decode and the key silently disappears. This is the engine-level core of
// the review's F15 evidence (Set OK; after restart the key is gone) and of
// decision 14.11 / ADR 0011's acceptance: a record of exactly
// value_max_bytes survives a restart at any accepted flag value. The
// process-level twin is internal/stand's TestRed_P01_valueLimit.
//
// Gating (D-01): the body skips unless SUPKV_RED_TESTS=1, so `make test`
// stays green while the defect the test pins is unfixed; scripts/red-runner
// opens the gate, asserts the FAIL event from `go test -json` and records
// the evidence. Phase 5's fix (the limit defined in one place and passed by
// configuration) turns this green with zero edits here — only the
// expectation-table row changes.
func TestRed_P01_valueLimitEngine(t *testing.T) {
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	t.Parallel()

	dumpPath := filepath.Join(t.TempDir(), "dump.jsonl")
	cfg := storage.Config{ValueMaxBytes: redP01ValueMaxBytes}
	eng, discarded := persist.Open(
		seedNodeID, seedFixedNow(seedNowMs), seedDriftWarn,
		persistfile.New(dumpPath), cfg,
		persist.Config{Debounce: seedDebounce},
	)
	if discarded != 0 {
		t.Fatalf("Open() on an empty directory discarded = %d, want 0", discarded)
	}
	seedDumper(t, eng)

	value := bytes.Repeat([]byte("v"), redP01ValueBytes)
	_, _, err := eng.Set(redP01EngineKey, value)
	if err != nil {
		t.Fatalf("Set(%q) of a %d-byte value with ValueMaxBytes=%d: %v — the store must accept a value under the configured limit",
			redP01EngineKey, redP01ValueBytes, redP01ValueMaxBytes, err)
	}

	// Set returned, so the dump covering the record completed (PERS-04):
	// the record is on disk before the reopen. Asserting it separates the
	// writer's success from the loader's loss — the defect's own premise.
	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read the dump %s after Set returned: %v", dumpPath, err)
	}
	if !bytes.Contains(dump, []byte(`"key":"`+redP01EngineKey+`"`)) {
		t.Fatalf("the confirmed %d-byte record is not in the dump %s after Set returned", redP01ValueBytes, dumpPath)
	}

	reopened, discardedAfter := persist.Open(
		seedNodeID, seedFixedNow(seedNowMs), seedDriftWarn,
		persistfile.New(dumpPath), cfg,
		persist.Config{Debounce: seedDebounce},
	)
	rec, err := reopened.Get(redP01EngineKey)
	if err != nil {
		t.Fatalf("the confirmed %d-byte record is gone after reopening the dump "+
			"(Open discarded %d record(s)): %v — the codec's 61 440-byte bound discards "+
			"a record the configured ValueMaxBytes=%d accepted (P-01, decision 14.11 / ADR 0011)",
			redP01ValueBytes, discardedAfter, err, redP01ValueMaxBytes)
	}
	if !bytes.Equal(rec.Data, value) {
		t.Fatalf("the %d-byte value after reopening the dump = %d bytes, want the full value (P-01)",
			redP01ValueBytes, len(rec.Data))
	}
}
