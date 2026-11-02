package persist_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// Tracer-test constants: the shared node ID and drift tolerance of the
// real-filesystem engines, the JSONL field tag the file-shape assertions
// anchor on, and the hex length of a record_version value.
const (
	tracerNodeID    = uint16(0x0301)
	tracerDriftWarn = time.Minute
	versionTag      = `"record_version":"`
	hexVersionLen   = 32
	dumpFileMode    = 0o600
	snapshotsDir    = ".snapshots"
	// shrunkValueLimit is the config-shrunk store limit the resilience test
	// installs: values over it stay codec-legal (envelope 61440) but the
	// store rejects them.
	shrunkValueLimit = 16
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
			err := <-done
			if !errors.Is(err, context.Canceled) {
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
			gotRec.Checksum != wantRec.Checksum || !bytes.Equal(gotRec.Data, wantRec.Data) {
			return false
		}
	}
	return true
}

// recordLine returns the dump line carrying key, or the empty string when
// absent.
func recordLine(dump []byte, key string) string {
	for line := range strings.SplitSeq(string(dump), "\n") {
		if strings.Contains(line, `"key":"`+key+`"`) {
			return line
		}
	}
	return ""
}

// hexVersion reports whether line carries a record_version whose value is
// a 32-character hex string — the UUID-shaped, greppable spelling of the 16
// version bytes.
func hexVersion(line string) bool {
	_, value, found := strings.Cut(line, versionTag)
	if !found || len(value) < hexVersionLen {
		return false
	}
	_, err := hex.DecodeString(value[:hexVersionLen])
	return err == nil
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

	_, _, err := eng.Set("alpha", []byte("alpha-value"))
	if err != nil {
		t.Fatalf("Set(alpha) error = %v, want nil", err)
	}
	_, _, err = eng.Set("beta", []byte("beta-value"))
	if err != nil {
		t.Fatalf("Set(beta) error = %v, want nil", err)
	}
	_, _, err = eng.Set("gamma", []byte("gamma-value"))
	if err != nil {
		t.Fatalf("Set(gamma) error = %v, want nil", err)
	}
	_, _, err = eng.Delete("gamma")
	if err != nil {
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
	alphaLine := recordLine(dump, "alpha")
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
		t.Fatal("reloaded records differ from engine 1's Records()")
	}
	rec, err := eng2.Get("alpha")
	if err != nil || string(rec.Data) != "alpha-value" {
		t.Errorf("reloaded Get(alpha) = (%q, %v), want (alpha-value, nil)", rec.Data, err)
	}
	_, err = eng2.Get("gamma")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("reloaded Get(gamma) error = %v, want ErrNotFound (tombstone)", err)
	}
	if tomb, ok := reloaded["gamma"]; !ok || !tomb.Deleted {
		t.Errorf("reloaded records lack the gamma tombstone: (present=%t, deleted=%t)", ok, tomb.Deleted)
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
	_, err := eng.Get("missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get() on the empty start error = %v, want ErrNotFound", err)
	}
	runEngine(t, eng)
	_, _, err = eng.Set("first", []byte("value"))
	if err != nil {
		t.Fatalf("first Set() error = %v, want nil", err)
	}
}

// Synctest-suite constants: the engine timing under test (the design
// defaults, D-04) and the fake port's node identity.
const (
	synctestNodeID   = uint16(0x0302)
	synctestDebounce = 50 * time.Millisecond
	synctestMaxWait  = time.Second
	burstWrites      = 1000
	fanoutWaiters    = 5
)

// errFakeSave is the fake port's injected Save failure.
var errFakeSave = errors.New("fake dump save failure")

// fakeSave is one recorded dump: its virtual timestamp and bytes.
type fakeSave struct {
	at   time.Time
	dump []byte
}

// fakeDumpPort is the in-memory port.DumpStore of the synctest suite: it
// records every save's virtual timestamp and bytes, counts attempts
// separately from completed saves, and fails the leading failSaves saves.
// It never touches the filesystem — real I/O inside a synctest bubble is
// not durably blocking and deadlock-panics the bubble (Pitfall 6). The
// snapshot methods mirror the adapter's semantics in memory: CreateSnapshot
// byte-copies the latest completed save under the name, missing snapshots
// read as an error.
type fakeDumpPort struct {
	mu        sync.Mutex
	attempts  int
	saves     []fakeSave
	failSaves int
	snapshots map[string][]byte
}

// Save records the dump at the current virtual time, failing the leading
// configured saves.
func (f *fakeDumpPort) Save(dump []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.failSaves > 0 {
		f.failSaves--
		return errFakeSave
	}
	f.saves = append(f.saves, fakeSave{at: time.Now(), dump: bytes.Clone(dump)})
	return nil
}

// Load is the empty start; the synctest suite never restarts.
func (*fakeDumpPort) Load(_ func([]byte) error) ([]byte, bool) { return nil, false }

// CreateSnapshot stores an in-memory copy of the latest completed save
// under the name.
func (f *fakeDumpPort) CreateSnapshot(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshots == nil {
		f.snapshots = make(map[string][]byte)
	}
	if len(f.saves) == 0 {
		return persist.ErrDumpFailed // nothing on disk to copy — ErrNoDump's in-memory stand-in.
	}
	f.snapshots[name] = bytes.Clone(f.saves[len(f.saves)-1].dump)
	return nil
}

// ReadSnapshot returns the in-memory copy, or a plain missing error.
func (f *fakeDumpPort) ReadSnapshot(name string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.snapshots[name]
	if !ok {
		return nil, errFakeSave
	}
	return bytes.Clone(data), nil
}

// snapshot returns the in-memory copy recorded under name.
func (f *fakeDumpPort) snapshot(name string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snapshots[name]
}

// saveCount returns the number of completed saves.
func (f *fakeDumpPort) saveCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.saves)
}

// saveAttempts returns the number of Save calls, failed ones included.
func (f *fakeDumpPort) saveAttempts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attempts
}

// save returns the recorded save i.
func (f *fakeDumpPort) save(i int) fakeSave {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.saves[i]
}

// newFakeEngine builds an engine over an empty store and a fresh fake port,
// running the design-default timing (the D-04 defaults the suite asserts);
// inside a synctest bubble time.Now is the virtual clock, so the store's
// clock rides it directly.
func newFakeEngine() (*persist.Engine, *fakeDumpPort) {
	dumps := &fakeDumpPort{}
	clk := clock.NewClock(time.Now, 0, 0, time.Minute)
	store := storage.NewStore(synctestNodeID, clk, storage.Config{})
	return persist.New(store, dumps, persist.Config{}), dumps
}

// startDumper launches eng.Run inside the bubble and returns a stop func
// that cancels and drains it; every synctest test defers stop inside the
// bubble so no dumper goroutine survives it (Pitfall 4).
func startDumper(t *testing.T, eng *persist.Engine) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	return func() {
		cancel()
		err := <-done
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
	}
}

// TestEngine_confirmDebounceTiming proves the write-confirmation timing law
// exactly in virtual time (PERS-04, D-04): a Set at t=0 registers its
// waiter, the debounced dumper waits out the full 50 ms quiet window, the
// single save lands at virtual t=50 ms, and Set returns only then — with
// the saved bytes decoding to the written record.
func TestEngine_confirmDebounceTiming(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()
		start := time.Now()
		setDone := make(chan time.Time, 1)
		go func() {
			ver, _, err := eng.Set("k", []byte("v"))
			if err != nil {
				t.Errorf("Set() error = %v, want nil", err)
			}
			if ver == (version.RecordVersion{}) {
				t.Error("Set() version is the zero version")
			}
			setDone <- time.Now()
		}()
		returnedAt := <-setDone
		synctest.Wait()
		if got := dumps.saveCount(); got != 1 {
			t.Fatalf("save count = %d, want 1", got)
		}
		save := dumps.save(0)
		if !save.at.Equal(start.Add(synctestDebounce)) {
			t.Errorf("save landed at %v, want exactly %v", save.at, start.Add(synctestDebounce))
		}
		if !returnedAt.Equal(start.Add(synctestDebounce)) {
			t.Errorf("Set returned at %v, want exactly the dump instant %v",
				returnedAt, start.Add(synctestDebounce))
		}
		records, discarded, err := persist.Decode(save.dump)
		if err != nil || discarded != 0 {
			t.Fatalf("Decode() = (%d discarded, %v), want (0, nil)", discarded, err)
		}
		if rec, ok := records["k"]; !ok || string(rec.Data) != "v" {
			t.Errorf("saved record for k = (%q, %t), want the written value", rec.Data, ok)
		}
	})
}

// TestEngine_burstBoundedConfirmation proves the D-04 bound under a
// sustained 1000 writes/s burst (PERS-03, success criterion 3): writes at a
// 1 ms virtual cadence keep resetting the debounce, so the first dump is
// forced by the oldest-write deadline at exactly virtual 1 s, every one of
// the 1000 writes is confirmed by then, and the burst coalesces into at
// most 3 dumps — never one dump per write.
func TestEngine_burstBoundedConfirmation(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()
		oldest := time.Now()
		returns := make(chan time.Time, burstWrites)
		var wg sync.WaitGroup
		for i := range burstWrites {
			wg.Go(func() {
				_, _, err := eng.Set(fmt.Sprintf("key-%03d", i), []byte("burst-value"))
				if err != nil {
					t.Errorf("burst Set(key-%03d) error = %v, want nil", i, err)
				}
				returns <- time.Now()
			})
			synctest.Sleep(time.Millisecond)
		}
		wg.Wait()
		close(returns)
		synctest.Wait()
		bound := oldest.Add(synctestMaxWait)
		confirmed := 0
		for returnedAt := range returns {
			confirmed++
			if returnedAt.After(bound) {
				t.Errorf("a burst write returned at %v, past the bound %v", returnedAt, bound)
			}
		}
		if confirmed != burstWrites {
			t.Fatalf("confirmed writes = %d, want %d", confirmed, burstWrites)
		}
		if got := dumps.saveAttempts(); got > 3 {
			t.Errorf("save attempts = %d, want at most 3 for %d writes", got, burstWrites)
		}
		if got := dumps.saveAttempts(); got == 0 {
			t.Fatal("no dump ever started")
		}
		if first := dumps.save(0).at; first.After(bound) {
			t.Errorf("first save at %v, past the oldest-write bound %v", first, bound)
		}
	})
}

// TestEngine_dumpFailureFansOut proves the D-05 failure contract: the
// failed save hands ErrDumpFailed to every current waiter at the same
// virtual instant while every record stays readable through Get, and after
// the port recovers a further Set is confirmed by the second save, which
// also carries the failed-era records — exactly 2 saves, no retry loop in
// between beyond the re-armed debounce timer.
func TestEngine_dumpFailureFansOut(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		dumps.failSaves = 1
		defer startDumper(t, eng)()
		failedAt := make(chan time.Time, fanoutWaiters)
		var wg sync.WaitGroup
		for i := range fanoutWaiters {
			wg.Go(func() {
				_, _, err := eng.Set(fmt.Sprintf("fan-%d", i), []byte("fan-value"))
				failedAt <- time.Now()
				if !errors.Is(err, persist.ErrDumpFailed) {
					t.Errorf("Set(fan-%d) error = %v, want ErrDumpFailed", i, err)
				}
			})
		}
		wg.Wait()
		close(failedAt)
		synctest.Wait()
		var instant time.Time
		first := true
		for at := range failedAt {
			if first {
				instant, first = at, false
			} else if !at.Equal(instant) {
				t.Errorf("waiters resolved at %v and %v, want the same virtual instant", instant, at)
			}
		}
		for i := range fanoutWaiters {
			rec, err := eng.Get(fmt.Sprintf("fan-%d", i))
			if err != nil || string(rec.Data) != "fan-value" {
				t.Errorf("Get(fan-%d) after the failed dump = (%q, %v), want the in-memory record",
					i, rec.Data, err)
			}
		}
		recoveredAt := make(chan time.Time, 1)
		go func() {
			_, _, err := eng.Set("recovered", []byte("recovered-value"))
			if err != nil {
				t.Errorf("recovery Set() error = %v, want nil", err)
			}
			recoveredAt <- time.Now()
		}()
		<-recoveredAt
		synctest.Wait()
		if got := dumps.saveAttempts(); got != 2 {
			t.Fatalf("save attempts = %d, want exactly 2 (one failed, one recovered — no retry loop)", got)
		}
		if got := dumps.saveCount(); got != 1 {
			t.Fatalf("completed saves = %d, want 1", got)
		}
		records, discarded, err := persist.Decode(dumps.save(dumps.saveCount() - 1).dump)
		if err != nil || discarded != 0 {
			t.Fatalf("Decode(second dump) = (%d discarded, %v), want (0, nil)", discarded, err)
		}
		for i := range fanoutWaiters {
			if _, ok := records[fmt.Sprintf("fan-%d", i)]; !ok {
				t.Errorf("second dump lacks the failed-era record fan-%d", i)
			}
		}
		if _, ok := records["recovered"]; !ok {
			t.Error("second dump lacks the recovered record")
		}
	})
}

// TestEngine_runExitClosesWaiters proves Run's exit sweep: a Set blocked
// mid-debounce resolves with ErrClosed the moment the context is
// cancelled, Run returns the context error, and synctest quiescence proves
// no goroutine survives the bubble.
func TestEngine_runExitClosesWaiters(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, _ := newFakeEngine()
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- eng.Run(ctx) }()
		setDone := make(chan error, 1)
		go func() {
			_, _, err := eng.Set("k", []byte("v"))
			setDone <- err
		}()
		synctest.Wait()
		cancel()
		err := <-setDone
		if !errors.Is(err, persist.ErrClosed) {
			t.Errorf("blocked Set() error = %v, want ErrClosed", err)
		}
		err = <-done
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
		synctest.Wait()
	})
}

// TestEngine_writeSurface proves the full write surface through the engine:
// SetIfAbsent on a tombstoned key returns the store's ErrAlreadyExists and
// Delete on a missing key the store's ErrNotFound — both unchanged, with no
// waiter registered — while SetIfAbsent on a fresh key and Delete of a live
// key confirm like Set: nil only after a covering dump that carries the
// record (PERS-04).
func TestEngine_writeSurface(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()
		_, _, err := eng.Set("tomb", []byte("doomed"))
		if err != nil {
			t.Fatalf("seed Set(tomb) error = %v, want nil", err)
		}
		_, _, err = eng.Delete("tomb")
		if err != nil {
			t.Fatalf("seed Delete(tomb) error = %v, want nil", err)
		}
		_, _, err = eng.SetIfAbsent("tomb", []byte("resurrect"))
		if !errors.Is(err, storage.ErrAlreadyExists) {
			t.Errorf("SetIfAbsent(tombstoned) error = %v, want ErrAlreadyExists", err)
		}
		_, _, err = eng.Delete("missing")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Delete(missing) error = %v, want ErrNotFound", err)
		}
		synctest.Wait()
		savesBefore := dumps.saveCount()

		ver, _, err := eng.SetIfAbsent("fresh", []byte("fresh-value"))
		if err != nil {
			t.Fatalf("SetIfAbsent(fresh) error = %v, want nil", err)
		}
		synctest.Wait()
		if got := dumps.saveCount(); got <= savesBefore {
			t.Fatalf("SetIfAbsent confirmed without a covering dump: saves %d, before %d",
				got, savesBefore)
		}
		records, _, err := persist.Decode(dumps.save(dumps.saveCount() - 1).dump)
		if err != nil {
			t.Fatalf("Decode() error = %v, want nil", err)
		}
		if rec, ok := records["fresh"]; !ok || rec.Version != ver {
			t.Errorf("covering dump carries fresh = (%t, version match %t), want the SetIfAbsent record",
				ok, rec.Version == ver)
		}

		ver, _, err = eng.Delete("fresh")
		if err != nil {
			t.Fatalf("Delete(live) error = %v, want nil", err)
		}
		synctest.Wait()
		records, _, err = persist.Decode(dumps.save(dumps.saveCount() - 1).dump)
		if err != nil {
			t.Fatalf("Decode() error = %v, want nil", err)
		}
		if rec, ok := records["fresh"]; !ok || !rec.Deleted || rec.Version != ver {
			t.Errorf("covering dump carries the fresh tombstone = (present=%t, deleted=%t, match=%t)",
				ok, rec.Deleted, rec.Version == ver)
		}
	})
}

// TestEngine_restartDiscardsTornRecord proves the restart class separation
// on the real filesystem (Pitfall 7): a torn record — a structurally valid
// line whose checksum was corrupted on disk — is discarded and counted at
// startup while its two siblings load byte-equal, and startup completes
// instead of collapsing to the empty start.
func TestEngine_restartDiscardsTornRecord(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, _ := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	stop := runEngine(t, eng)
	_, _, err := eng.Set("torn", []byte("torn-value"))
	if err != nil {
		t.Fatalf("Set(torn) error = %v, want nil", err)
	}
	_, _, err = eng.Set("sib-a", []byte("a-value"))
	if err != nil {
		t.Fatalf("Set(sib-a) error = %v, want nil", err)
	}
	_, _, err = eng.Set("sib-b", []byte("b-value"))
	if err != nil {
		t.Fatalf("Set(sib-b) error = %v, want nil", err)
	}
	stop()

	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read dump: %v", err)
	}
	tampered := rewriteLine(t, dump, "torn", func(rec *dumpLine) { rec.Checksum++ })
	err = os.WriteFile(dumpPath, tampered, dumpFileMode)
	if err != nil {
		t.Fatalf("rewrite the torn dump: %v", err)
	}

	eng2, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	if discarded != 1 {
		t.Errorf("Open() discarded = %d, want 1 (the torn record)", discarded)
	}
	_, err = eng2.Get("torn")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(torn) after the torn restart = %v, want ErrNotFound", err)
	}
	for key, want := range map[string]string{"sib-a": "a-value", "sib-b": "b-value"} {
		rec, err := eng2.Get(key)
		if err != nil || string(rec.Data) != want {
			t.Errorf("Get(%s) after the torn restart = (%q, %v), want (%q, nil)",
				key, rec.Data, err, want)
		}
	}
	if got := len(eng2.Records()); got != 2 {
		t.Errorf("reloaded records = %d, want the two siblings only", got)
	}
}

// TestEngine_snapshotCreateLoadRestore is plan 03-02's tracer (PERS-05): a
// named snapshot is a byte-exact copy of the confirmed on-disk main dump
// (D-10), and Restore replays exactly the snapshot's keys through the
// store's ordinary write path with fresh dominating versions — a
// snapshot-live value beats a current tombstone, a snapshot tombstone beats
// a current live value, and a key absent from the snapshot keeps its state
// untouched (D-09) — while a fresh engine with only the carried-over
// snapshot directory restores every snapshot key (recovery from scratch).
func TestEngine_snapshotCreateLoadRestore(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	if discarded != 0 {
		t.Fatalf("Open() discarded = %d, want 0", discarded)
	}
	runEngine(t, eng)

	// The snapshot's state: three live keys plus two that go in tombstoned.
	seeds := []struct{ key, value string }{
		{"keep-a", "keep-a-value"},
		{"keep-b", "keep-b-value"},
		{"resurrect", "resurrect-value"},
		{"re-tomb", "doomed-in-snapshot"},
		{"gone", "gone-in-snapshot"},
	}
	for _, s := range seeds {
		_, _, err := eng.Set(s.key, []byte(s.value))
		if err != nil {
			t.Fatalf("seed Set(%s) error = %v, want nil", s.key, err)
		}
	}
	for _, key := range []string{"re-tomb", "gone"} {
		_, _, err := eng.Delete(key)
		if err != nil {
			t.Fatalf("seed Delete(%s) error = %v, want nil", key, err)
		}
	}

	err := eng.Snapshot("pre")
	if err != nil {
		t.Fatalf("Snapshot(pre) error = %v, want nil", err)
	}
	// D-10: the snapshot file is byte-identical to the main dump at
	// creation time — read both before any further write lands.
	mainDump, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read the main dump: %v", err)
	}
	snapshotPath := filepath.Join(dumpPath+snapshotsDir, "pre")
	snapshotDump, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read the snapshot: %v", err)
	}
	if !bytes.Equal(snapshotDump, mainDump) {
		t.Error("snapshot bytes differ from the main dump at creation time")
	}

	// Post-snapshot drift: tombstone a snapshot-live key, resurrect a
	// snapshot-tombstoned one, create a key the snapshot never saw.
	_, _, err = eng.Delete("resurrect")
	if err != nil {
		t.Fatalf("post-snapshot Delete(resurrect) error = %v, want nil", err)
	}
	_, _, err = eng.Set("re-tomb", []byte("live-after-snapshot"))
	if err != nil {
		t.Fatalf("post-snapshot Set(re-tomb) error = %v, want nil", err)
	}
	_, _, err = eng.Set("post-key", []byte("post-value"))
	if err != nil {
		t.Fatalf("post-snapshot Set(post-key) error = %v, want nil", err)
	}

	preRestore := eng.Records()
	err = eng.Restore("pre")
	if err != nil {
		t.Fatalf("Restore(pre) error = %v, want nil", err)
	}
	restored := eng.Records()

	// Fresh-version dominance: every replayed record beats every
	// pre-restore version, so the restore replicates and wins by normal
	// LWW like any write (D-09).
	var maxPre version.RecordVersion
	for _, rec := range preRestore {
		if rec.Version.Compare(maxPre) > 0 {
			maxPre = rec.Version
		}
	}
	for _, key := range []string{"keep-a", "keep-b", "resurrect", "re-tomb"} {
		rec, ok := restored[key]
		if !ok {
			t.Fatalf("restored records lack %s", key)
		}
		if rec.Version.Compare(maxPre) != 1 {
			t.Errorf("restored %s version %x does not dominate every pre-restore version (max %x)",
				key, rec.Version[:], maxPre[:])
		}
	}
	// The snapshot's live values are back, including the resurrected key
	// whose fresh Set beats the post-snapshot tombstone.
	for key, want := range map[string]string{
		"keep-a":    "keep-a-value",
		"keep-b":    "keep-b-value",
		"resurrect": "resurrect-value",
	} {
		rec, err := eng.Get(key)
		if err != nil || string(rec.Data) != want {
			t.Errorf("post-restore Get(%s) = (%q, %v), want (%q, nil)", key, rec.Data, err, want)
		}
	}
	// re-tomb: the snapshot's tombstone beats the post-snapshot live value.
	_, err = eng.Get("re-tomb")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("post-restore Get(re-tomb) error = %v, want ErrNotFound (the snapshot tombstone wins)", err)
	}
	if rec := restored["re-tomb"]; !rec.Deleted {
		t.Error("restored re-tomb is live, want the snapshot's tombstone")
	}
	// gone: the snapshot holds it deleted and it stayed tombstoned — the
	// replay Delete is the store's ErrNotFound no-op: no version burned.
	if got, want := restored["gone"].Version, preRestore["gone"].Version; got != want {
		t.Errorf("restored gone version %x differs from pre-restore %x, want identical (no version burned on the no-op Delete)",
			got[:], want[:])
	}
	// post-key: absent from the snapshot — its post-snapshot state stands.
	rec, err := eng.Get("post-key")
	if err != nil || string(rec.Data) != "post-value" {
		t.Errorf("post-restore Get(post-key) = (%q, %v), want the untouched post-snapshot value",
			rec.Data, err)
	}
	if got, want := restored["post-key"].Version, preRestore["post-key"].Version; got != want {
		t.Errorf("post-key version moved across the restore: %x then %x, want identical (absent keys untouched)",
			want[:], got[:])
	}

	// Recovery from scratch: a fresh engine — empty store, fresh disk, only
	// the snapshot directory carried over — loads every snapshot key.
	freshPath := filepath.Join(t.TempDir(), "dump.json")
	err = os.MkdirAll(freshPath+snapshotsDir, 0o700)
	if err != nil {
		t.Fatalf("create the fresh snapshots dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(freshPath+snapshotsDir, "pre"), snapshotDump, dumpFileMode)
	if err != nil {
		t.Fatalf("carry the snapshot over: %v", err)
	}
	eng3, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(freshPath), storage.Config{}, persist.Config{},
	)
	if discarded != 0 {
		t.Errorf("fresh Open() discarded = %d, want 0", discarded)
	}
	runEngine(t, eng3)
	err = eng3.Restore("pre")
	if err != nil {
		t.Fatalf("fresh Restore(pre) error = %v, want nil", err)
	}
	for key, want := range map[string]string{
		"keep-a":    "keep-a-value",
		"keep-b":    "keep-b-value",
		"resurrect": "resurrect-value",
	} {
		freshRec, freshErr := eng3.Get(key)
		if freshErr != nil || string(freshRec.Data) != want {
			t.Errorf("fresh-restored Get(%s) = (%q, %v), want (%q, nil)", key, freshRec.Data, freshErr, want)
		}
	}
	for _, key := range []string{"re-tomb", "gone", "post-key"} {
		_, err := eng3.Get(key)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("fresh-restored Get(%s) error = %v, want ErrNotFound", key, err)
		}
	}
	// The snapshot's tombstones replay as ErrNotFound no-ops on the fresh
	// engine — nothing to beat, nothing written — so exactly the three
	// live keys exist.
	if got := len(eng3.Records()); got != 3 {
		t.Errorf("fresh-restored records = %d, want the 3 live snapshot keys (tombstones are no-ops here)", got)
	}
}

// TestEngine_snapshotFlushForcesDump pins D-10's "forces the current dump
// to complete" against the real dumper loop under exact virtual time with
// no real I/O inside the bubble (PERS-05): a dirty engine — its one save
// failed, the write returned ErrDumpFailed — must complete a NEW dump
// before the snapshot copies, observable as the fake's attempt count (the
// forced save is attempt two) and the copied bytes carrying the record;
// a Snapshot on the fully-confirmed engine that follows adds no save at
// all (not dirty: no dump, straight copy).
func TestEngine_snapshotFlushForcesDump(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		dumps.failSaves = 1
		defer startDumper(t, eng)()
		setErr := make(chan error, 1)
		go func() {
			_, _, err := eng.Set("forced", []byte("forced-value"))
			setErr <- err
		}()
		err := <-setErr
		if !errors.Is(err, persist.ErrDumpFailed) {
			t.Fatalf("Set() error = %v, want ErrDumpFailed (the dirty state)", err)
		}
		if got := dumps.saveCount(); got != 0 {
			t.Fatalf("completed saves before the snapshot = %d, want 0", got)
		}
		err = eng.Snapshot("s")
		if err != nil {
			t.Fatalf("Snapshot(s) error = %v, want nil", err)
		}
		if got := dumps.saveAttempts(); got != 2 {
			t.Fatalf("save attempts = %d, want exactly 2 (one failed, one forced by the snapshot)", got)
		}
		if got := dumps.saveCount(); got != 1 {
			t.Fatalf("completed saves = %d, want 1", got)
		}
		copied := dumps.snapshot("s")
		if !bytes.Equal(copied, dumps.save(0).dump) {
			t.Error("snapshot copy differs from the forced save's bytes")
		}
		records, discarded, err := persist.Decode(copied)
		if err != nil || discarded != 0 {
			t.Fatalf("Decode(snapshot) = (%d discarded, %v), want (0, nil)", discarded, err)
		}
		if rec, ok := records["forced"]; !ok || string(rec.Data) != "forced-value" {
			t.Errorf("snapshot lacks the forced record: (%q, %t), want the written value", rec.Data, ok)
		}

		// Fully confirmed now: the second Snapshot is a straight copy —
		// no save beyond what the dirty state required.
		err = eng.Snapshot("s2")
		if err != nil {
			t.Fatalf("Snapshot(s2) error = %v, want nil", err)
		}
		if got := dumps.saveAttempts(); got != 2 {
			t.Errorf("save attempts after the confirmed snapshot = %d, want still 2 (no extra save)", got)
		}
		if !bytes.Equal(dumps.snapshot("s2"), dumps.save(0).dump) {
			t.Error("second snapshot copy differs from the confirmed save's bytes")
		}
	})
}

// TestEngine_restoreMissingSnapshot proves the missing-snapshot error class
// on the real adapter (PERS-05): Restore of a name with no snapshot file
// surfaces the wrapped fs.ErrNotExist error — never a silent empty success
// — and leaves the engine's records untouched.
func TestEngine_restoreMissingSnapshot(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, _ := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	runEngine(t, eng)
	_, _, err := eng.Set("stable", []byte("stable-value"))
	if err != nil {
		t.Fatalf("seed Set(stable) error = %v, want nil", err)
	}
	before := eng.Records()
	err = eng.Restore("nope")
	if err == nil {
		t.Fatal("Restore(nope) = nil, want the missing-snapshot error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Restore(nope) error = %v, want one wrapping fs.ErrNotExist", err)
	}
	if !recordsEqual(eng.Records(), before) {
		t.Error("records changed across the failed restore")
	}
}

// TestEngine_closeSurface pins the post-close contract of every blocking
// engine call (PERS-04, Pitfall 4): once Run has exited, Set, SetIfAbsent,
// Delete, Snapshot and Restore return ErrClosed immediately — no writer can
// register after the exit sweep and block forever — while Get keeps serving
// the in-memory records (reads are never gated on dumps or the dumper) and a
// rejected Snapshot copies nothing to disk.
func TestEngine_closeSurface(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	if discarded != 0 {
		t.Fatalf("Open() discarded = %d, want 0", discarded)
	}
	stop := runEngine(t, eng)
	_, _, err := eng.Set("k", []byte("v"))
	if err != nil {
		t.Fatalf("pre-close Set(k) error = %v, want nil", err)
	}
	err = eng.Snapshot("pre")
	if err != nil {
		t.Fatalf("pre-close Snapshot(pre) error = %v, want nil", err)
	}
	stop()

	_, _, err = eng.Set("set", []byte("v"))
	if !errors.Is(err, persist.ErrClosed) {
		t.Errorf("post-close Set() error = %v, want ErrClosed", err)
	}
	_, _, err = eng.SetIfAbsent("sia", []byte("v"))
	if !errors.Is(err, persist.ErrClosed) {
		t.Errorf("post-close SetIfAbsent() error = %v, want ErrClosed", err)
	}
	err = eng.Snapshot("post")
	if !errors.Is(err, persist.ErrClosed) {
		t.Errorf("post-close Snapshot() error = %v, want ErrClosed", err)
	}
	_, err = os.Stat(filepath.Join(dumpPath+snapshotsDir, "post"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("post-close Snapshot() left a file behind: %v", err)
	}
	rec, getErr := eng.Get("k")
	if getErr != nil || string(rec.Data) != "v" {
		t.Errorf("post-close Get(k) = (%q, %v), want (v, nil) — reads are never gated", rec.Data, getErr)
	}
	_, _, err = eng.Delete("k")
	if !errors.Is(err, persist.ErrClosed) {
		t.Errorf("post-close Delete() error = %v, want ErrClosed", err)
	}
	err = eng.Restore("pre")
	if !errors.Is(err, persist.ErrClosed) {
		t.Errorf("post-close Restore() error = %v, want ErrClosed", err)
	}
}

// TestEngine_snapshotFlushErrorAborts pins D-10's safety half under synctest:
// when the forced flush's dump fails, Snapshot aborts with that error and
// copies nothing — state that is not confirmed on disk must never become a
// recovery point — while the record stays readable in memory (D-05).
func TestEngine_snapshotFlushErrorAborts(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		dumps.failSaves = burstWrites // every save this bubble attempts fails
		defer startDumper(t, eng)()
		_, _, err := eng.Set("k", []byte("v"))
		if !errors.Is(err, persist.ErrDumpFailed) {
			t.Fatalf("Set(k) error = %v, want ErrDumpFailed", err)
		}
		err = eng.Snapshot("never")
		if !errors.Is(err, errFakeSave) {
			t.Fatalf("Snapshot(never) error = %v, want the forced flush's save error", err)
		}
		if got := dumps.snapshot("never"); got != nil {
			t.Error("Snapshot() copied bytes despite the failed forced flush")
		}
		rec, getErr := eng.Get("k")
		if getErr != nil || string(rec.Data) != "v" {
			t.Errorf("Get(k) after the failed flush = (%q, %v), want the in-memory record", rec.Data, getErr)
		}
	})
}

// TestEngine_restoreCorruptSnapshot pins the corrupt-snapshot error class
// (PERS-05): a snapshot whose bytes carry a generation-level decode fault —
// a torn record tail — surfaces that error from Restore, never a silent
// empty success, and leaves the map unchanged.
func TestEngine_restoreCorruptSnapshot(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, _ := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	stop := runEngine(t, eng)
	_, _, err := eng.Set("k", []byte("v"))
	if err != nil {
		t.Fatalf("seed Set(k) error = %v, want nil", err)
	}
	err = eng.Snapshot("bad")
	if err != nil {
		t.Fatalf("seed Snapshot(bad) error = %v, want nil", err)
	}
	stop()
	snapPath := filepath.Join(dumpPath+snapshotsDir, "bad")
	torn := []byte("{\"format_version\":1}\n{\"key\":\"torn\"")
	err = os.WriteFile(snapPath, torn, dumpFileMode)
	if err != nil {
		t.Fatalf("corrupt the snapshot bytes: %v", err)
	}
	eng2, _ := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	runEngine(t, eng2)
	before := eng2.Records()
	err = eng2.Restore("bad")
	if err == nil {
		t.Fatal("Restore(corrupt snapshot) = nil, want the generation-level decode error")
	}
	if !recordsEqual(eng2.Records(), before) {
		t.Error("records changed across the corrupt-snapshot restore")
	}
}

// TestEngine_configShrunkValueLimits pins the config-shrink resilience
// paths: Open skips a dumped record the shrunk store limit rejects instead
// of aborting startup, Restore skips the same class instead of failing, and
// a direct Set surfaces storage.ErrValueTooLarge unchanged with no waiter
// registered (PERS-02/PERS-05 — the codec's 61440 envelope does not shrink
// with the config, so these are the belt-and-braces skips).
func TestEngine_configShrunkValueLimits(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	fat := bytes.Repeat([]byte("f"), shrunkValueLimit*4)
	eng, _ := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	stop := runEngine(t, eng)
	_, _, err := eng.Set("fat", fat)
	if err != nil {
		t.Fatalf("seed Set(fat) error = %v, want nil", err)
	}
	_, _, err = eng.Set("slim", []byte("slim"))
	if err != nil {
		t.Fatalf("seed Set(slim) error = %v, want nil", err)
	}
	err = eng.Snapshot("cfg")
	if err != nil {
		t.Fatalf("seed Snapshot(cfg) error = %v, want nil", err)
	}
	stop()

	shrunk, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath),
		storage.Config{ValueMaxBytes: shrunkValueLimit}, persist.Config{},
	)
	if discarded != 0 {
		t.Errorf("shrunk Open() discarded = %d, want 0 (the skip is at merge, not codec)", discarded)
	}
	_, err = shrunk.Get("fat")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("shrunk Get(fat) error = %v, want ErrNotFound (skipped at startup)", err)
	}
	slimRec, slimErr := shrunk.Get("slim")
	if slimErr != nil || string(slimRec.Data) != "slim" {
		t.Errorf("shrunk Get(slim) = (%q, %v), want (slim, nil)", slimRec.Data, slimErr)
	}
	_, _, err = shrunk.Set("huge", fat)
	if !errors.Is(err, storage.ErrValueTooLarge) {
		t.Errorf("shrunk Set(huge) error = %v, want ErrValueTooLarge unchanged", err)
	}
	runEngine(t, shrunk)
	err = shrunk.Restore("cfg")
	if err != nil {
		t.Fatalf("shrunk Restore(cfg) error = %v, want nil (oversized record skipped)", err)
	}
	_, err = shrunk.Get("fat")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("post-restore Get(fat) error = %v, want ErrNotFound (replay skipped)", err)
	}
	slimRec, slimErr = shrunk.Get("slim")
	if slimErr != nil || string(slimRec.Data) != "slim" {
		t.Errorf("post-restore Get(slim) = (%q, %v), want (slim, nil)", slimRec.Data, slimErr)
	}
}
