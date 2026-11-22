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
	"slices"
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

	// applyRemoteOversize is one byte over the design-default value limit
	// (docs/IDEA.md §Parameters): the wire-side oversize class whose
	// ErrValueTooLarge must pass through ApplyRemote unchanged (T-04-04).
	applyRemoteOversize = 61441
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
// defaults, D-04), the fake port's node identity and the flush-race width.
const (
	synctestNodeID   = uint16(0x0302)
	synctestDebounce = 50 * time.Millisecond
	synctestMaxWait  = time.Second
	burstWrites      = 1000
	fanoutWaiters    = 5
	flushRaceRounds  = 100
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
// not durably blocking and deadlock-panics the bubble (Pitfall 6). A
// non-nil gate parks every Save before it records anything — the hold
// point that keeps one dump in flight while the exit-race tests cancel.
// The snapshot methods mirror the adapter's semantics in memory:
// CreateSnapshot byte-copies the latest completed save under the name,
// missing snapshots read as an error.
type fakeDumpPort struct {
	mu        sync.Mutex
	attempts  int
	saves     []fakeSave
	failSaves int
	snapshots map[string][]byte
	gate      chan struct{}
}

// Save records the dump at the current virtual time, failing the leading
// configured saves; a configured gate parks the save before it records —
// the dump is in flight, not attempted, while parked.
func (f *fakeDumpPort) Save(dump []byte) error {
	f.mu.Lock()
	gate := f.gate
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}
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

// installSnapshot stores a hand-built snapshot generation under name — the
// seam the restore-report tests use to supply a persist.Encode fixture.
func (f *fakeDumpPort) installSnapshot(name string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshots == nil {
		f.snapshots = make(map[string][]byte)
	}
	f.snapshots[name] = bytes.Clone(data)
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
//
// Decision 14.3 (ADR 0003): the dump write policy stays as-is — the debounce
// window and the single save per quiet period are the accepted disk-protection
// bias, not an implementation accident this timing test may optimize away.
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
		records, discarded, err := persist.Decode(save.dump, storage.ValueMaxBytes)
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
		records, discarded, err := persist.Decode(dumps.save(dumps.saveCount()-1).dump, storage.ValueMaxBytes)
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

// TestEngine_flushRacesRunExit is CR-01's regression: the flush-path
// callers Snapshot and Restore race Run's exit across many interleavings,
// and every call must return promptly with nil or ErrClosed. A flush that
// passed its closed-check before the exit sweep and landed its flushCh
// send after the sweep's one-shot drain parks on the reply forever under
// the pre-fix code; inside the bubble that is an all-goroutines-asleep
// deadlock synctest fails loudly — the same post-close contract
// TestEngine_closeSurface pins, extended to the exit race itself (Pitfall
// 4).
func TestEngine_flushRacesRunExit(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		for i := range flushRaceRounds {
			eng, _ := newFakeEngine()
			ctx, cancel := context.WithCancel(t.Context())
			runDone := make(chan error, 1)
			go func() { runDone <- eng.Run(ctx) }()
			// One confirmed write and a copied snapshot: the raced calls
			// have real work — the restore replays, the snapshot copies.
			_, _, err := eng.Set("k", []byte("v"))
			if err != nil {
				t.Fatalf("round %d: seed Set(k) error = %v, want nil", i, err)
			}
			err = eng.Snapshot("seed")
			if err != nil {
				t.Fatalf("round %d: seed Snapshot(seed) error = %v, want nil", i, err)
			}
			raced := make(chan error, 1)
			go func() {
				var err error
				if i%2 == 0 {
					err = eng.Snapshot("raced")
				} else {
					_, err = eng.Restore("seed")
				}
				raced <- err
			}()
			if i%2 == 1 {
				synctest.Sleep(time.Millisecond) // vary the interleaving
			}
			cancel()
			// A flush stranded between its closed-check and its flushCh
			// send deadlocks the bubble at this receive, not just this
			// goroutine — that is the failure being guarded against.
			err = <-raced
			if err != nil && !errors.Is(err, persist.ErrClosed) {
				t.Errorf("round %d: raced call error = %v, want nil or ErrClosed", i, err)
			}
			err = <-runDone
			if !errors.Is(err, context.Canceled) {
				t.Errorf("round %d: Run() error = %v, want context.Canceled", i, err)
			}
		}
	})
}

// TestEngine_flushRacesExitMidDump walks the exit sweep through a dump held
// in flight with every flush shape stranded at once: a Set awaiting the
// held dump's confirmation, one snapshot whose flush request sits in the
// flushCh buffer, and two parked on the flushCh send itself (the buffer
// holds the first's request). Cancelling and releasing the gate together
// must resolve every call promptly — nil from the completed dump or
// ErrClosed from the sweep — and Run must return the context error: the
// sweep's drain answers the buffered and the parked senders, runDone
// releases what the drain cannot have seen, nothing hangs (CR-01, Pitfall
// 4).
func TestEngine_flushRacesExitMidDump(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		dumps.gate = make(chan struct{})
		ctx, cancel := context.WithCancel(t.Context())
		runErr := make(chan error, 1)
		go func() { runErr <- eng.Run(ctx) }()

		setErr := make(chan error, 1)
		go func() {
			_, _, err := eng.Set("k", []byte("v"))
			setErr <- err
		}()
		// Quiescence parks Run inside the gated Save of the debounced dump
		// — the dump is held in flight, the Set awaits its confirmation.
		synctest.Wait()
		// Flush shapes against the held dump: one request buffered in
		// flushCh, then senders parked on the full buffer.
		queued := make(chan error, 1)
		go func() { queued <- eng.Snapshot("queued") }()
		parkedA := make(chan error, 1)
		go func() { parkedA <- eng.Snapshot("parked-a") }()
		parkedB := make(chan error, 1)
		go func() { parkedB <- eng.Snapshot("parked-b") }()
		synctest.Wait()

		cancel()          // the exit sweep must not strand anyone mid-dump
		close(dumps.gate) // the held dump completes and the dumper exits
		results := []struct {
			name string
			err  error
		}{
			{"Set", <-setErr},
			{"queued", <-queued},
			{"parked-a", <-parkedA},
			{"parked-b", <-parkedB},
		}
		for _, res := range results {
			if res.err != nil && !errors.Is(res.err, persist.ErrClosed) {
				t.Errorf("%s error = %v, want nil or ErrClosed", res.name, res.err)
			}
		}
		err := <-runErr
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
		if got := dumps.saveCount(); got != 1 {
			t.Errorf("completed saves = %d, want exactly the one held dump", got)
		}
		synctest.Wait()
	})
}

// TestFlush_remoteDirtiness pins Flush's shutdown contract (D-19): a remote
// merge applied via ApplyRemote dirties the engine with NO waiter — only
// the debounced Run schedule would persist it — so shutdown must call Flush
// before cancelling the engine context. Run's exit sweep resolves stranded
// waiters with ErrClosed without performing a final dump, making Flush the
// only path that turns residual dirtiness into a confirmed dump: the saved
// bytes must carry the merged record — the write-confirmed-after-dump
// contract extended to the shutdown path.
func TestFlush_remoteDirtiness(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()

		remote := storage.Record{
			Version: version.PackVersion(1, 1, synctestNodeID+1),
			Data:    []byte("remote-value"),
		}
		conflicts, _, err := eng.ApplyRemote("remote", remote)
		if err != nil {
			t.Fatalf("ApplyRemote() error = %v, want nil", err)
		}
		if len(conflicts) != 0 {
			t.Fatalf("ApplyRemote() conflicts = %v, want none on a fresh key", conflicts)
		}
		// No synctest.Sleep: the debounce timer has not fired, so the flush
		// request is the first event Run services — the dump Flush forces is
		// the one covering the merge, exactly the shutdown-time dump D-19
		// orders.
		err = eng.Flush()
		if err != nil {
			t.Fatalf("Flush() error = %v, want nil", err)
		}
		if saves := dumps.saveCount(); saves != 1 {
			t.Fatalf("completed saves = %d, want exactly the one covering dump", saves)
		}
		records, discarded, err := persist.Decode(dumps.save(0).dump, storage.ValueMaxBytes)
		if err != nil || discarded != 0 {
			t.Fatalf("Decode(save) = (%d discarded, %v), want (0, nil)", discarded, err)
		}
		merged, ok := records["remote"]
		if !ok || merged.Version != remote.Version || string(merged.Data) != "remote-value" {
			t.Errorf("the Flush dump carries remote = (%t, %x, %q), want the merged record",
				ok, merged.Version[:], merged.Data)
		}
	})
}

// TestFlush_closed pins the closed half of the shutdown contract: once Run
// has returned — its exit sweep set the closed flag and closed runDone —
// Flush returns ErrClosed instead of hanging or re-dumping, the same
// post-close contract flush reports to its callers (Snapshot, Restore, and
// now the binary's signal path).
func TestFlush_closed(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, _ := newFakeEngine()
		stop := startDumper(t, eng)
		stop() // the exit sweep ran: closed is set, runDone is closed

		err := eng.Flush()
		if !errors.Is(err, persist.ErrClosed) {
			t.Errorf("Flush() error = %v, want ErrClosed after Run exited", err)
		}
	})
}

// TestFlush_clean pins the Phase-3 clean-engine decision on the shutdown
// path: every confirmed write is already on disk, so Flush on a clean
// engine returns nil WITHOUT issuing a new save — the redundant-dump
// prohibition the flushCh seam carries (D-10), extended to shutdown, so a
// clean node's final dump costs nothing.
func TestFlush_clean(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()

		// One confirmed write: its covering dump completed, the engine is
		// clean — the realistic shutdown state with data on disk.
		_, _, err := eng.Set("k", []byte("v"))
		if err != nil {
			t.Fatalf("Set(k) error = %v, want nil", err)
		}
		if saves := dumps.saveCount(); saves != 1 {
			t.Fatalf("completed saves after the confirmed Set = %d, want exactly one", saves)
		}
		err = eng.Flush()
		if err != nil {
			t.Fatalf("Flush() error = %v, want nil on a clean engine", err)
		}
		if saves := dumps.saveCount(); saves != 1 {
			t.Errorf("completed saves after Flush = %d, want the same one — a clean engine never re-dumps", saves)
		}
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
		records, _, err := persist.Decode(dumps.save(dumps.saveCount()-1).dump, storage.ValueMaxBytes)
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
		records, _, err = persist.Decode(dumps.save(dumps.saveCount()-1).dump, storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("Decode() error = %v, want nil", err)
		}
		if rec, ok := records["fresh"]; !ok || !rec.Deleted || rec.Version != ver {
			t.Errorf("covering dump carries the fresh tombstone = (present=%t, deleted=%t, match=%t)",
				ok, rec.Deleted, rec.Version == ver)
		}
	})
}

// TestEngine_lookup_tombstoneVisible proves the engine forwards the
// tombstone-visible view to its store while Get stays the live-only API
// read: a deleted key is ErrNotFound for Get, yet Lookup returns the
// tombstone with Deleted=true and the same version Version reports, and
// Versions carries it. The writes go through ApplyRemote — the non-blocking
// merge path — so no dumper goroutine is needed for a read-contract test.
func TestEngine_lookup_tombstoneVisible(t *testing.T) {
	t.Parallel()
	const probeNowMs = int64(1730000000000)
	eng, _ := newFakeEngine()
	live := version.PackVersion(probeNowMs, 0, synctestNodeID)
	_, _, err := eng.ApplyRemote("view/key", storage.Record{Version: live, Data: []byte("value")})
	if err != nil {
		t.Fatalf("ApplyRemote(live) error = %v, want nil", err)
	}
	tombstone := version.PackVersion(probeNowMs+1, 0, synctestNodeID)
	_, _, err = eng.ApplyRemote("view/key", storage.Record{Version: tombstone, Deleted: true})
	if err != nil {
		t.Fatalf("ApplyRemote(tombstone) error = %v, want nil", err)
	}

	_, err = eng.Get("view/key")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Get(tombstoned) error = %v, want storage.ErrNotFound", err)
	}
	rec, ok := eng.Lookup("view/key")
	if !ok {
		t.Fatal("Lookup(tombstoned) ok = false, want true")
	}
	if !rec.Deleted {
		t.Error("Lookup(tombstoned).Deleted = false, want true")
	}
	if rec.Version != tombstone {
		t.Errorf("Lookup(tombstoned) version = %x, want %x", rec.Version[:], tombstone[:])
	}
	ver, found := eng.Version("view/key")
	if !found || ver != tombstone {
		t.Errorf("Version(tombstoned) = (%x, %t), want (%x, true)", ver[:], found, tombstone[:])
	}
	versions := eng.Versions()
	if got, present := versions["view/key"]; !present || got != tombstone {
		t.Errorf("Versions()[view/key] = (%x, %t), want (%x, true)", got[:], present, tombstone[:])
	}
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
//
// Decision 14.1 (ADR 0001): the snapshot family rides the accepted rule that
// a snapshot load is a full rollback, so the named snapshot is an exact copy
// of the current confirmed state and Restore replays its keys under the same
// install discipline — never a partial merge into whatever the target holds.
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
	_, err = eng.Restore("pre")
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
	_, err = eng3.Restore("pre")
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
		records, discarded, err := persist.Decode(copied, storage.ValueMaxBytes)
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
	_, err = eng.Restore("nope")
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
	_, err = eng.Restore("pre")
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
	_, err = eng2.Restore("bad")
	if err == nil {
		t.Fatal("Restore(corrupt snapshot) = nil, want the generation-level decode error")
	}
	if !recordsEqual(eng2.Records(), before) {
		t.Error("records changed across the corrupt-snapshot restore")
	}
}

// snapshotFixture encodes records as a snapshot generation, filling each
// record's checksum from the production domain first: Decode discards a
// checksum-invalid record, so a hand-built fixture must carry the real
// checksum or a restore test could pass for the wrong reason (the 03-08
// hand-built-fixture rule).
func snapshotFixture(t *testing.T, records map[string]storage.Record) []byte {
	t.Helper()
	for key, rec := range records {
		rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
		records[key] = rec
	}
	dump, err := persist.Encode(records, storage.ValueMaxBytes)
	if err != nil {
		t.Fatalf("persist.Encode(snapshot fixture) error = %v, want nil", err)
	}
	return dump
}

// TestEngine_restoreReturnsReplayed proves the replay report PROTO-08's
// snapshot-load notify consumes: a restore that changes the store returns
// every snapshot record it replayed — sorted by key, each carrying the
// store's freshly stamped version, the snapshot's data and tombstone flag —
// while keys the snapshot never named keep their state untouched.
func TestEngine_restoreReturnsReplayed(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()

		for key, value := range map[string]string{
			"alpha": "old-alpha",
			"beta":  "old-beta",
			"gamma": "gamma-value",
		} {
			_, _, err := eng.Set(key, []byte(value))
			if err != nil {
				t.Fatalf("seed Set(%s) error = %v, want nil", key, err)
			}
		}

		snapVersion := version.PackVersion(1, 0, synctestNodeID+1)
		dumps.installSnapshot("snap", snapshotFixture(t, map[string]storage.Record{
			"alpha": {Version: snapVersion, Data: []byte("snap-alpha")},
			"beta":  {Version: snapVersion, Deleted: true},
			"delta": {Version: snapVersion, Data: []byte("snap-delta")},
		}))

		restored, err := eng.Restore("snap")
		if err != nil {
			t.Fatalf("Restore(snap) error = %v, want nil", err)
		}
		keys := make([]string, len(restored))
		for i, rec := range restored {
			keys[i] = rec.Key
		}
		if want := []string{"alpha", "beta", "delta"}; !slices.Equal(keys, want) {
			t.Fatalf("Restore(snap) keys = %v, want %v (sorted, skipped records absent)", keys, want)
		}
		for _, rec := range restored {
			if rec.Version == (version.RecordVersion{}) {
				t.Errorf("restored %s carries the zero version", rec.Key)
			}
			if rec.Version.Compare(snapVersion) != 1 {
				t.Errorf("restored %s version %x does not dominate the snapshot's %x",
					rec.Key, rec.Version[:], snapVersion[:])
			}
		}
		byKey := make(map[string]persist.RestoredRecord, len(restored))
		for _, rec := range restored {
			byKey[rec.Key] = rec
		}
		if rec := byKey["alpha"]; rec.Deleted || string(rec.Data) != "snap-alpha" {
			t.Errorf("restored alpha = (%q, deleted=%t), want the snapshot's live value", rec.Data, rec.Deleted)
		}
		if rec := byKey["beta"]; !rec.Deleted {
			t.Error("restored beta is live, want the snapshot's tombstone")
		}
		if rec := byKey["delta"]; rec.Deleted || string(rec.Data) != "snap-delta" {
			t.Errorf("restored delta = (%q, deleted=%t), want the snapshot's live value", rec.Data, rec.Deleted)
		}
		// gamma is absent from the snapshot: its current state stands.
		rec, getErr := eng.Get("gamma")
		if getErr != nil || string(rec.Data) != "gamma-value" {
			t.Errorf("post-restore Get(gamma) = (%q, %v), want the untouched value", rec.Data, getErr)
		}
	})
}

// TestEngine_restoreNoOpReturnsEmpty proves the no-op report: a snapshot
// equal to the store's current state returns an empty record list and nil,
// marks nothing dirty and performs no dump — the whole-restore no-op
// contract D-15 (03 IN-02 + 04 IN-02) with PROTO-08's empty answer.
func TestEngine_restoreNoOpReturnsEmpty(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()

		_, _, err := eng.Set("keep", []byte("keep-value"))
		if err != nil {
			t.Fatalf("seed Set(keep) error = %v, want nil", err)
		}
		err = dumps.CreateSnapshot("same")
		if err != nil {
			t.Fatalf("CreateSnapshot(same) error = %v, want nil", err)
		}
		saves := dumps.saveCount()

		restored, err := eng.Restore("same")
		if err != nil {
			t.Fatalf("Restore(same) error = %v, want nil", err)
		}
		if len(restored) != 0 {
			t.Fatalf("Restore(same) returned %d records, want an empty list", len(restored))
		}
		if got := dumps.saveCount(); got != saves {
			t.Errorf("completed saves after the no-op restore = %d, want the unchanged %d", got, saves)
		}
	})
}

// TestEngine_restoreFailedFlushReturnsReplayed proves a failed confirming
// dump still reports what the replay installed (PROTO-08): Restore returns
// the replayed records plus ErrDumpFailed, and the store keeps the replayed
// state readable in memory — the dump failure is reported honestly, never
// as a successful empty load.
func TestEngine_restoreFailedFlushReturnsReplayed(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()

		_, _, err := eng.Set("alpha", []byte("old-alpha"))
		if err != nil {
			t.Fatalf("seed Set(alpha) error = %v, want nil", err)
		}
		snapVersion := version.PackVersion(1, 0, synctestNodeID+1)
		dumps.installSnapshot("snap", snapshotFixture(t, map[string]storage.Record{
			"alpha": {Version: snapVersion, Data: []byte("new-alpha")},
		}))
		dumps.mu.Lock()
		dumps.failSaves = burstWrites
		dumps.mu.Unlock()

		restored, err := eng.Restore("snap")
		if !errors.Is(err, persist.ErrDumpFailed) {
			t.Fatalf("Restore(snap) error = %v, want ErrDumpFailed", err)
		}
		if len(restored) != 1 || restored[0].Key != "alpha" || string(restored[0].Data) != "new-alpha" {
			t.Fatalf("Restore(snap) records = %+v, want the replayed alpha", restored)
		}
		rec, getErr := eng.Get("alpha")
		if getErr != nil || string(rec.Data) != "new-alpha" {
			t.Errorf("post-restore Get(alpha) = (%q, %v), want the replayed in-memory state", rec.Data, getErr)
		}
	})
}

// TestEngine_restoreSkipsReplay proves skipped shapes are absent from the
// report (PROTO-08): a tombstone with no live record to beat and a value
// over the store's reduced limit are both skips — Restore returns an empty
// list and nil, installs nothing, and never notifies what it did not
// replay.
func TestEngine_restoreSkipsReplay(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		dumps := &fakeDumpPort{}
		clk := clock.NewClock(time.Now, 0, 0, time.Minute)
		store := storage.NewStore(synctestNodeID, clk, storage.Config{ValueMaxBytes: shrunkValueLimit})
		eng := persist.New(store, dumps, persist.Config{})
		defer startDumper(t, eng)()

		snapVersion := version.PackVersion(1, 0, synctestNodeID+1)
		dumps.installSnapshot("snap", snapshotFixture(t, map[string]storage.Record{
			"absent-tomb": {Version: snapVersion, Deleted: true},
			"oversize":    {Version: snapVersion, Data: bytes.Repeat([]byte("x"), shrunkValueLimit+1)},
		}))

		restored, err := eng.Restore("snap")
		if err != nil {
			t.Fatalf("Restore(snap) error = %v, want nil (both skips are not errors)", err)
		}
		if len(restored) != 0 {
			t.Fatalf("Restore(snap) records = %+v, want an empty list (both shapes skip)", restored)
		}
		if got := len(eng.Records()); got != 0 {
			t.Errorf("post-restore records = %d, want 0 (nothing was replayed)", got)
		}
	})
}

// TestEngine_applyRemoteTriggersDump proves the REPL-03 memory-apply contract
// at the persist seam plan 04-06 lands (the seam await's doc reserved in
// Phase 3: "only local writes register waiters, and Phase 4's remote merges
// will reuse the dirty path without waiters"): ApplyRemote merges the remote
// record into the store, marks the engine dirty with NO waiter — the call
// returns at the apply instant, strictly before the debounced dump — and the
// running Run completes exactly one save whose bytes decode to the merged
// record. An oversize record's ErrValueTooLarge passes through unchanged and
// the engine keeps serving (T-04-04).
func TestEngine_applyRemoteTriggersDump(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()

		remote := storage.Record{
			Version: version.PackVersion(1, 1, synctestNodeID+1),
			Data:    []byte("remote-value"),
		}
		returned := make(chan time.Time, 1)
		go func() {
			conflicts, _, err := eng.ApplyRemote("remote", remote)
			if err != nil {
				t.Errorf("ApplyRemote() error = %v, want nil", err)
			}
			if len(conflicts) != 0 {
				t.Errorf("ApplyRemote() conflicts = %v, want none on a fresh key", conflicts)
			}
			returned <- time.Now()
		}()
		returnedAt := <-returned
		// Advance the fake clock past the debounce fire: unlike the waiter
		// tests (where the blocked Set call itself holds the test goroutine
		// until the dump resolves), this no-waiter path needs an explicit
		// Sleep to move time to the timer (synctest.Wait alone does not
		// advance the bubble clock).
		synctest.Sleep(2 * synctestDebounce)

		got, err := eng.Get("remote")
		if err != nil || string(got.Data) != "remote-value" || got.Version != remote.Version {
			t.Errorf("post-ApplyRemote Get() = (%q, %x, %v), want the merged record", got.Data, got.Version[:], err)
		}
		if saves := dumps.saveCount(); saves != 1 {
			t.Fatalf("completed saves = %d, want exactly one dump for the remote merge", saves)
		}
		save := dumps.save(0)
		if !returnedAt.Before(save.at) {
			t.Errorf("ApplyRemote returned at %v, not before the dump at %v — it registered a waiter",
				returnedAt, save.at)
		}
		records, discarded, err := persist.Decode(save.dump, storage.ValueMaxBytes)
		if err != nil || discarded != 0 {
			t.Fatalf("Decode(save) = (%d discarded, %v), want (0, nil)", discarded, err)
		}
		if merged, ok := records["remote"]; !ok || merged.Version != remote.Version || string(merged.Data) != "remote-value" {
			t.Errorf("the save carries remote = (%t, %x, %q), want the merged record", ok, merged.Version[:], merged.Data)
		}

		// Oversize passthrough (T-04-04): nothing merges, nothing dirties,
		// the error surfaces unchanged and the next good merge still dumps.
		huge := storage.Record{
			Version: version.PackVersion(2, 1, synctestNodeID+1),
			Data:    bytes.Repeat([]byte("x"), applyRemoteOversize),
		}
		_, _, err = eng.ApplyRemote("huge", huge)
		if !errors.Is(err, storage.ErrValueTooLarge) {
			t.Errorf("oversize ApplyRemote() error = %v, want ErrValueTooLarge unchanged", err)
		}
		_, err = eng.Get("huge")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("oversize Get(huge) error = %v, want ErrNotFound", err)
		}
		good := storage.Record{
			Version: version.PackVersion(3, 1, synctestNodeID+1),
			Data:    []byte("still-serving"),
		}
		_, _, err = eng.ApplyRemote("second", good)
		if err != nil {
			t.Fatalf("post-error ApplyRemote() error = %v, want nil", err)
		}
		synctest.Sleep(2 * synctestDebounce)
		if saves := dumps.saveCount(); saves != 2 {
			t.Fatalf("completed saves after the oversize error = %d, want 2 (the engine kept serving)", saves)
		}
	})
}

// TestEngine_configShrunkValueLimits pins the config-shrink resilience
// paths: Open skips a dumped record the shrunk store limit rejects instead
// of aborting startup, Restore skips the same class instead of failing, and
// a direct Set surfaces storage.ErrValueTooLarge unchanged with no waiter
// registered (PERS-02/PERS-05 — the codec's 61440 envelope does not shrink
// with the config, so these are the belt-and-braces skips).
//
// Decision 14.11 (ADR 0011): the ~60 KiB single-value limit stays — this
// test narrows only the configurable envelope under it, so the accepted
// ceiling is what a shrunk config is measured against, never a probe that the
// decision would raise or remove.
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
	_, err = shrunk.Restore("cfg")
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

// redundantRemoteNode is the peer identity the redundant-dirty fixtures'
// remote versions carry — distinct from synctestNodeID so the versions
// never collide with locally stamped ones.
const redundantRemoteNode = uint16(0x0303)

// TestEngine_redundantDirty proves the conditional dirty (D-15, 03 IN-02 +
// 04 IN-02, threat T-6-01): case (a) — after one establishing remote merge
// has dumped, an equal-version redelivery of the same record, a stale
// (rejected) redelivery, and a Restore whose snapshot content already
// matches the store all leave the clean engine clean: no further dump
// runs. Case (b) — a genuinely greater remote version on a clean engine
// still arms the dump, and the one confirming dump covers a concurrent
// local write's waiter too, so PERS-04's confirmation semantics are
// unchanged.
func TestEngine_redundantDirty(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()

		base := version.PackVersion(1730000000000, 0, redundantRemoteNode)
		rec := storage.Record{Version: base, Data: []byte("remote value")}
		_, _, err := eng.ApplyRemote("cfg/remote", rec)
		if err != nil {
			t.Fatalf("establishing ApplyRemote() error = %v, want nil", err)
		}
		synctest.Sleep(synctestDebounce)
		synctest.Wait()
		if got := dumps.saveCount(); got != 1 {
			t.Fatalf("the establishing remote merge ran %d saves, want 1", got)
		}

		// Equal-version redelivery over the live record: a full no-op.
		_, _, err = eng.ApplyRemote("cfg/remote", rec)
		if err != nil {
			t.Fatalf("equal-redelivery ApplyRemote() error = %v, want nil", err)
		}
		// Stale redelivery: rejected with exactly one conflict naming the
		// stored record as the winner, nothing installed.
		stale := storage.Record{Version: version.PackVersion(1730000000000-1, 0, redundantRemoteNode), Data: []byte("stale value")}
		conflicts, _, err := eng.ApplyRemote("cfg/remote", stale)
		if err != nil {
			t.Fatalf("stale ApplyRemote() error = %v, want nil", err)
		}
		if len(conflicts) != 1 || conflicts[0].Winner != base {
			t.Fatalf("stale redelivery conflicts = %v, want exactly one with Winner = the stored version", conflicts)
		}
		// A snapshot of the current state replays entirely as skips: the
		// restore marks nothing dirty and its final flush replies nil
		// without a save.
		err = dumps.CreateSnapshot("matching")
		if err != nil {
			t.Fatalf("CreateSnapshot(matching) error = %v, want nil", err)
		}
		_, err = eng.Restore("matching")
		if err != nil {
			t.Fatalf("Restore(matching) error = %v, want nil", err)
		}

		// Any wrongly armed dirty epoch would fire its save inside this
		// window; a clean engine stays at exactly the one save.
		synctest.Sleep(synctestDebounce)
		synctest.Wait()
		if got := dumps.saveCount(); got != 1 {
			t.Fatalf("no-op redelivery + matching restore ran %d extra dumps, want 0 — redundant dirty (D-15)", got-1)
		}

		// Case (b): the greater remote version alone arms a dump on a
		// clean engine, and the same covering dump resolves the concurrent
		// local write's waiter — one save covers both.
		eng2, dumps2 := newFakeEngine()
		defer startDumper(t, eng2)()
		newer := version.PackVersion(1730000000001, 0, redundantRemoteNode)
		_, _, err = eng2.ApplyRemote("cfg/remote", storage.Record{Version: newer, Data: []byte("newer value")})
		if err != nil {
			t.Fatalf("greater-version ApplyRemote() error = %v, want nil", err)
		}
		setDone := make(chan error, 1)
		go func() {
			_, _, err := eng2.Set("cfg/local", []byte("local value"))
			setDone <- err
		}()
		synctest.Sleep(synctestDebounce)
		synctest.Wait()
		err = <-setDone
		if err != nil {
			t.Fatalf("local Set() through the remote-armed epoch error = %v, want nil (PERS-04 unchanged)", err)
		}
		if got := dumps2.saveCount(); got != 1 {
			t.Fatalf("the remote change plus the local waiter ran %d saves, want exactly 1 covering dump", got)
		}
		records, discarded, err := persist.Decode(dumps2.save(0).dump, storage.ValueMaxBytes)
		if err != nil || discarded != 0 {
			t.Fatalf("Decode(covering dump) = (%d discarded, %v), want (0, nil)", discarded, err)
		}
		if r, ok := records["cfg/remote"]; !ok || r.Version != newer || string(r.Data) != "newer value" {
			t.Fatalf("covering dump holds cfg/remote = (%v, %q), want (%v, %q)", r.Version, r.Data, newer, "newer value")
		}
		if r, ok := records["cfg/local"]; !ok || string(r.Data) != "local value" {
			t.Fatalf("covering dump holds cfg/local = %q, want the confirmed local write", r.Data)
		}
	})
}

// TestDumpStats_records proves the dump-cycle gauge (OBS-01): one forced
// successful dump records a strictly positive duration with total 1 and
// errors 0, and one failed Save through the fake port's injected failure
// records the error with total 2 — real time, not a synctest bubble,
// because the assertion is on a real monotonic span. The engine stops
// immediately after the failure, ahead of the re-armed debounce retry, so
// the totals stay exact.
func TestDumpStats_records(t *testing.T) {
	t.Parallel()
	eng, dumps := newFakeEngine()
	stop := runEngine(t, eng)

	mkRecord := func(data string) storage.Record {
		return storage.Record{
			Version: version.PackVersion(time.Now().UnixMilli(), 0, synctestNodeID),
			Data:    []byte(data),
		}
	}

	_, _, err := eng.ApplyRemote("dumpstats/one", mkRecord("one"))
	if err != nil {
		t.Fatalf("ApplyRemote(dumpstats/one): %v", err)
	}
	err = eng.Flush()
	if err != nil {
		t.Fatalf("Flush #1: %v", err)
	}
	if got := eng.DumpStats(); got.Total != 1 || got.Errors != 0 || got.DurationSeconds <= 0 {
		t.Fatalf("after the successful dump DumpStats() = %+v, want Total 1, Errors 0, a positive duration", got)
	}

	dumps.failSaves = 1
	_, _, err = eng.ApplyRemote("dumpstats/two", mkRecord("two"))
	if err != nil {
		t.Fatalf("ApplyRemote(dumpstats/two): %v", err)
	}
	err = eng.Flush()
	if !errors.Is(err, errFakeSave) {
		t.Fatalf("Flush #2 = %v, want the injected fake save failure", err)
	}
	stop()
	if got := eng.DumpStats(); got.Total != 2 || got.Errors != 1 || got.DurationSeconds <= 0 {
		t.Fatalf("after the failed dump DumpStats() = %+v, want Total 2, Errors 1, a positive duration", got)
	}
}

// TestEngine_keyCountAndObserveGeneration pins the three engine
// pass-throughs (PROTO-01, PROTO-02, T-4-43): KeyCount counts records
// without consulting dump state — 0 before any write, 1 once a confirmed
// Set has completed its dump — Generation moves only with the store's
// content and not for reads, and Observe feeds the store clock so the next
// Set strictly dominates the absorbed version.
func TestEngine_keyCountAndObserveGeneration(t *testing.T) {
	t.Parallel()
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, discarded := persist.Open(
		tracerNodeID, time.Now, tracerDriftWarn,
		persistfile.New(dumpPath), storage.Config{}, persist.Config{},
	)
	if discarded != 0 {
		t.Errorf("Open() discarded = %d, want 0", discarded)
	}
	runEngine(t, eng)

	if got := eng.KeyCount(); got != 0 {
		t.Fatalf("KeyCount() on an empty engine = %d, want 0", got)
	}
	if got := eng.Generation(); got != 0 {
		t.Fatalf("Generation() on an empty engine = %d, want 0", got)
	}

	// Set blocks until a covering dump completed, so the count is read after
	// the record is both in memory and on disk.
	_, _, err := eng.Set("alpha", []byte("alpha-value"))
	if err != nil {
		t.Fatalf("Set(alpha) error = %v, want nil", err)
	}
	if got := eng.KeyCount(); got != 1 {
		t.Fatalf("KeyCount() after a confirmed Set = %d, want 1", got)
	}
	if got := eng.Generation(); got != 1 {
		t.Fatalf("Generation() after a confirmed Set = %d, want 1", got)
	}

	// Read paths move neither counter, dump state or not.
	_, err = eng.Get("alpha")
	if err != nil {
		t.Fatalf("Get(alpha) error = %v, want nil", err)
	}
	_, _ = eng.Lookup("alpha")
	_, _ = eng.Version("alpha")
	_ = eng.Versions()
	_ = eng.Records()
	if got := eng.KeyCount(); got != 1 {
		t.Fatalf("KeyCount() after reads = %d, want the unchanged 1", got)
	}
	if got := eng.Generation(); got != 1 {
		t.Fatalf("Generation() after reads = %d, want the unchanged 1", got)
	}

	// Observe is clock-only: no record, no generation move, and the next
	// Set strictly dominates the absorbed version.
	absorbed := version.PackVersion(time.Now().Add(time.Hour).UnixMilli(), 5, tracerNodeID+1)
	eng.Observe(absorbed)
	if got := eng.KeyCount(); got != 1 {
		t.Fatalf("KeyCount() after Observe = %d, want the unchanged 1", got)
	}
	if got := eng.Generation(); got != 1 {
		t.Fatalf("Generation() after Observe = %d, want the unchanged 1", got)
	}
	ver, _, err := eng.Set("beta", []byte("beta-value"))
	if err != nil {
		t.Fatalf("Set(beta) error = %v, want nil", err)
	}
	if got := ver.Compare(absorbed); got <= 0 {
		t.Fatalf("next Set version %x does not strictly beat the absorbed %x", ver[:], absorbed[:])
	}
	if got := eng.KeyCount(); got != 2 {
		t.Fatalf("KeyCount() after the second Set = %d, want 2", got)
	}
	if got := eng.Generation(); got != 2 {
		t.Fatalf("Generation() after the second Set = %d, want 2", got)
	}
}
