package persist_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// Tracer-test constants: the shared node ID and drift tolerance of the
// real-filesystem engines, and the JSONL field tag the file-shape assertions
// anchor on.
const (
	tracerNodeID    = uint16(0x0301)
	tracerDriftWarn = time.Minute
	versionTag      = `"record_version":"`
	hexVersionLen   = 32
)

// runEngine launches eng's single dumper goroutine (Run's lifecycle is the
// documented API contract) and returns an idempotent stop func that cancels
// the context and waits for Run to return context.Canceled; stop is also
// registered as a test cleanup so a failing assertion path still drains the
// goroutine.
func runEngine(t *testing.T, eng *persist.Engine) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	var once sync.Once
	stop = func() {
		once.Do(func() {
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Errorf("Run() error = %v, want context.Canceled", err)
			}
		})
	}
	t.Cleanup(stop)
	return stop
}

// recordsEqual reports whether two record maps hold the same keys with
// deep-equal records: version, deleted flag, checksum and data bytes.
func recordsEqual(got, want map[string]storage.Record) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantRec := range want {
		gotRec, ok := got[key]
		if !ok || gotRec.Version != wantRec.Version || gotRec.Deleted != wantRec.Deleted ||
			gotRec.Checksum != wantRec.Checksum || string(gotRec.Data) != string(wantRec.Data) {
			return false
		}
	}
	return true
}

// recordLine returns the dump line carrying key, or nil when absent.
func recordLine(t *testing.T, dump []byte, key string) string {
	t.Helper()
	for _, line := range strings.Split(string(dump), "\n") {
		if strings.Contains(line, `"key":"`+key+`"`) {
			return line
		}
	}
	return ""
}

// TestEngine_endToEndRestart is the Phase 3 tracer: a confirmed local write
// flows end-to-end — engine.Set commits to the store, the debounced dumper
// encodes the full map, the file adapter writes it atomically, and Set
// returns nil only after that completed dump — then persist.Open on the same
// path reloads the records, seeds the clock with the full watermark, and a
// post-restart write strictly dominates every dumped version (PERS-01,
// PERS-04, VERS-04 persistence half).
func TestEngine_endToEndRestart(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	if discarded != 0 {
		t.Errorf("Open() discarded = %d, want 0", discarded)
	}
	stop := runEngine(t, eng)

	if _, _, err := eng.Set("alpha", []byte("alpha-value")); err != nil {
		t.Fatalf("Set(alpha) error = %v, want nil", err)
	}
	if _, _, err := eng.Set("beta", []byte("beta-value")); err != nil {
		t.Fatalf("Set(beta) error = %v, want nil", err)
	}
	if _, _, err := eng.Set("gamma", []byte("gamma-value")); err != nil {
		t.Fatalf("Set(gamma) error = %v, want nil", err)
	}
	if _, _, err := eng.Delete("gamma"); err != nil {
		t.Fatalf("Delete(gamma) error = %v, want nil", err)
	}

	// A confirmed write implies the dump exists on disk at this moment.
	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("confirmed write left no dump on disk: %v", err)
	}
	lines := strings.Split(string(dump), "\n")
	if !strings.Contains(lines[0], "format_version") {
		t.Errorf("dump header %q lacks format_version", lines[0])
	}
	for _, line := range lines {
		if strings.Contains(line, versionTag[:len(versionTag)-1]+"[") {
			t.Errorf("dump line %q carries the record_version as a JSON number array", line)
		}
	}
	alphaLine := recordLine(t, dump, "alpha")
	if alphaLine == "" {
		t.Fatalf("dump carries no line for key alpha: %q", dump)
	}
	if !strings.Contains(alphaLine, base64.StdEncoding.EncodeToString([]byte("alpha-value"))) {
		t.Errorf("alpha line %q lacks the base64 value", alphaLine)
	}
	if !hexVersion(alphaLine) {
		t.Errorf("alpha line %q lacks a 32-char hex record_version", alphaLine)
	}

	// Simulated restart: engine 1 dies after the confirmed dump, engine 2
	// Opens the same path and must reload the full map, tombstone included.
	before := eng.Records()
	stop()
	eng2, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	if discarded != 0 {
		t.Errorf("restart Open() discarded = %d, want 0", discarded)
	}
	reloaded := eng2.Records()
	if !recordsEqual(reloaded, before) {
		t.Fatalf("reloaded records differ from engine 1's Records()")
	}
	if rec, err := eng2.Get("alpha"); err != nil || string(rec.Data) != "alpha-value" {
		t.Errorf("reloaded Get(alpha) = (%q, %v), want (alpha-value, nil)", rec.Data, err)
	}
	if _, err := eng2.Get("gamma"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("reloaded Get(gamma) error = %v, want ErrNotFound (tombstone)", err)
	}
	if rec, ok := reloaded["gamma"]; !ok || !rec.Deleted {
		t.Errorf("reloaded records lack the gamma tombstone: (%v, deleted=%t)", ok, rec.Deleted)
	}

	// VERS-04: a fresh post-restart write strictly dominates every reloaded
	// version, so it wins any LWW comparison against the dump's records.
	runEngine(t, eng2)
	ver, _, err := eng2.Set("delta", []byte("delta-value"))
	if err != nil {
		t.Fatalf("post-restart Set(delta) error = %v, want nil", err)
	}
	for key, rec := range reloaded {
		if ver.Compare(rec.Version) != 1 {
			t.Errorf("fresh version %x does not dominate reloaded key %q version %x",
				ver[:], key, rec.Version[:])
		}
	}
}

// hexVersion reports whether line carries a record_version whose value is a
// 32-character hex string — the UUID-shaped, greppable spelling of the 16
// version bytes.
func hexVersion(line string) bool {
	i := strings.Index(line, versionTag)
	if i < 0 {
		return false
	}
	value := line[i+len(versionTag):]
	if len(value) < hexVersionLen {
		return false
	}
	_, err := hex.DecodeString(value[:hexVersionLen])
	return err == nil
}

// TestEngine_openEmptyStart proves the empty start: with no dump files
// present, Open returns an engine over an empty store and the first Set
// works without error.
func TestEngine_openEmptyStart(t *testing.T) {
	t.Parallel()
	eng, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(filepath.Join(t.TempDir(), "dump.json")), storage.Config{}, persist.Config{},
	)
	if discarded != 0 {
		t.Errorf("Open() discarded = %d, want 0", discarded)
	}
	if _, err := eng.Get("missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get() on the empty start error = %v, want ErrNotFound", err)
	}
	runEngine(t, eng)
	if _, _, err := eng.Set("first", []byte("value")); err != nil {
		t.Fatalf("first Set() error = %v, want nil", err)
	}
}
