package persist_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// memDump is an in-memory port.DumpStore that survives a simulated restart:
// Load returns the last successfully saved generation.
type memDump struct {
	mu   sync.Mutex
	last []byte
	fail bool
	n    int
}

var errDiskFull = errors.New("ENOSPC (injected)")

func (m *memDump) Save(dump []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.n++
	if m.fail {
		return errDiskFull
	}
	m.last = bytes.Clone(dump)
	return nil
}

func (m *memDump) Load(parse func([]byte) error) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.last == nil || parse(m.last) != nil {
		return nil, false
	}
	return m.last, true
}

func (*memDump) CreateSnapshot(string) error         { return nil }
func (*memDump) ReadSnapshot(string) ([]byte, error) { return nil, errDiskFull }
func (m *memDump) setFail(v bool)                    { m.mu.Lock(); m.fail = v; m.mu.Unlock() }
func (m *memDump) attempts() int                     { m.mu.Lock(); defer m.mu.Unlock(); return m.n }

func startRun(t *testing.T, eng *persist.Engine) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	return func() { cancel(); <-done }
}

// ENG-01: --value-max-bytes above the codec's hard-coded 61440 envelope: the
// write is accepted, dumped and CONFIRMED, and silently discarded at restart.
func TestReview_valueMaxAboveCodecEnvelopeLosesConfirmedWrite(t *testing.T) {
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	cfg := storage.Config{ValueMaxBytes: 65000} // operator raised the documented tunable
	eng, _ := persist.Open(1, time.Now, time.Minute, persistfile.New(dumpPath), cfg, persist.Config{})
	stop := startRun(t, eng)
	big := bytes.Repeat([]byte("x"), 62000) // legal under the configured limit, fits one UDP datagram
	if _, _, err := eng.Set("big", big); err != nil {
		t.Fatalf("Set(big) = %v, want nil (confirmed)", err)
	}
	if _, _, err := eng.Set("small", []byte("s")); err != nil {
		t.Fatalf("Set(small) = %v", err)
	}
	stop()

	eng2, discarded := persist.Open(1, time.Now, time.Minute, persistfile.New(dumpPath), cfg, persist.Config{})
	_, err := eng2.Get("big")
	t.Logf("after restart with the SAME config: discarded=%d, Get(big) err=%v", discarded, err)
	if err == nil {
		t.Fatal("expected the confirmed write to be lost (bug not reproduced)")
	}
	if _, err := eng2.Get("small"); err != nil {
		t.Fatalf("Get(small) = %v", err)
	}
	t.Errorf("CONFIRMED WRITE LOST: Set(big) returned nil, restart discarded it (discarded=%d, err=%v)", discarded, err)
}

// ENG-02: a version that reached the peer but never reached this node's dump
// is re-issued after restart while the HLC runs ahead of the wall clock. Equal
// versions with different payloads are a silent no-op in Merge: permanent
// divergence, with the second write CONFIRMED to its client.
func TestReview_equalVersionDivergenceAfterRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeA, nodeB = uint16(1), uint16(2)
		wall := time.Now()
		now := func() time.Time { return wall } // A's wall clock; advanced by hand below
		dumpsA := &memDump{}

		// Peer B: its wall clock runs 60 s ahead of A's (unsynchronized clocks are in scope).
		clkB := clock.NewClock(func() time.Time { return wall.Add(60 * time.Second) }, 0, 0, 0)
		storeB := storage.NewStore(nodeB, clkB, storage.Config{})

		engA, _ := persist.Open(nodeA, now, 0, dumpsA, storage.Config{}, persist.Config{})
		stop := startRun(t, engA)

		// B writes something; A absorbs it — A's HLC now runs 60 s ahead of A's wall clock.
		if _, _, err := storeB.Set("other", []byte("b")); err != nil {
			t.Fatal(err)
		}
		if _, _, err := engA.ApplyRemote("other", storeB.Records()["other"]); err != nil {
			t.Fatal(err)
		}
		v1, _, err := engA.Set("k", []byte("v1"))
		if err != nil {
			t.Fatalf("Set(k,v1) = %v", err)
		}

		// The disk fills up (or: the process is killed before the next dump).
		// A hot key keeps being written; none of these versions reaches the dump.
		dumpsA.setFail(true)
		var v2 version.RecordVersion
		for i := range 3 {
			v2, _, err = engA.Set("k", []byte(fmt.Sprintf("lost-%d", i)))
			if !errors.Is(err, persist.ErrDumpFailed) {
				t.Fatalf("Set(k,lost-%d) = %v, want ErrDumpFailed", i, err)
			}
		}
		// The record stays in memory and reaches B through the periodic DataSync
		// (A lists key->version, B answers DataRequest, A serves Records()[k]).
		if _, _, err := storeB.Merge("k", engA.Records()["k"]); err != nil {
			t.Fatal(err)
		}

		// A crashes; the operator frees the disk; A restarts 5 s later.
		stop()
		dumpsA.setFail(false)
		wall = wall.Add(5 * time.Second)
		engA2, _ := persist.Open(nodeA, now, 0, dumpsA, storage.Config{}, persist.Config{})
		stop2 := startRun(t, engA2)
		defer stop2()

		// Startup gate elapsed without a sync round (B briefly unreachable); a client writes k.
		v3, _, err := engA2.Set("k", []byte("v3"))
		if err != nil {
			t.Fatalf("Set(k,v3) = %v, want nil", err)
		}
		t.Logf("v1=%x (dumped)\nv2=%x (never dumped, held by B)\nv3=%x (confirmed after restart)", v1[:], v2[:], v3[:])
		if v3 != v2 {
			t.Skipf("no collision: v3 != v2")
		}

		// Full reconciliation both ways: DataSync sees equal versions => silence;
		// even a forced record exchange is a Merge no-op.
		cA, _, _ := engA2.ApplyRemote("k", storeB.Records()["k"])
		cB, _, _ := storeB.Merge("k", engA2.Records()["k"])
		a, b := engA2.Records()["k"], storeB.Records()["k"]
		t.Logf("after exchange: A=%q B=%q  conflicts A=%d B=%d", a.Data, b.Data, len(cA), len(cB))
		if a.Version == b.Version && !bytes.Equal(a.Data, b.Data) {
			t.Errorf("PERMANENT DIVERGENCE: same version %x, A holds %q, B holds %q; no conflict reported",
				a.Version[:], a.Data, b.Data)
		}
	})
}

// ENG-04: one flipped byte inside a base64 data field rejects the WHOLE
// generation (not just the record); a dump truncated at a line boundary is
// accepted as complete.
func TestReview_decodeGenerationFaults(t *testing.T) {
	records := make(map[string]storage.Record)
	for i := range 1000 {
		key := fmt.Sprintf("key-%04d", i)
		ver := version.PackVersion(int64(1000+i), 0, 7)
		data := bytes.Repeat([]byte{byte(i)}, 300)
		records[key] = storage.Record{Version: ver, Data: data, Checksum: storage.RecordChecksum(key, ver, data, false)}
	}
	dump, err := persist.Encode(records)
	if err != nil {
		t.Fatal(err)
	}

	// (a) flip ONE byte of one record's base64 payload to a non-alphabet byte.
	corrupt := bytes.Clone(dump)
	idx := bytes.Index(corrupt, []byte(`"key":"key-0500"`))
	dataIdx := idx + bytes.Index(corrupt[idx:], []byte(`"data":"`)) + len(`"data":"`) + 10
	corrupt[dataIdx] ^= 0x40 // single-bit flip
	got, discarded, err := persist.Decode(corrupt)
	t.Logf("(a) single bit flip in base64 (%q -> %q): records=%d discarded=%d err=%v",
		dump[dataIdx], corrupt[dataIdx], len(got), discarded, err)
	if err != nil {
		t.Errorf("one damaged record of 1000 rejected the whole generation: %v", err)
	}

	// (b) truncate at a line boundary: header + first 400 records.
	lines := bytes.SplitAfter(dump, []byte("\n"))
	truncated := bytes.Join(lines[:401], nil)
	got, discarded, err = persist.Decode(truncated)
	t.Logf("(b) truncated to 400 of 1000 records: records=%d discarded=%d err=%v", len(got), discarded, err)
	if err == nil && len(got) != len(records) {
		t.Errorf("truncated generation accepted as complete: %d of %d records, err=nil", len(got), len(records))
	}
}

// ENG-07: a persistently failing dump is retried every debounce interval
// forever, with no backoff, and the cause never reaches the caller.
func TestReview_failingDumpRetryCadence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dumps := &memDump{fail: true}
		clk := clock.NewClock(time.Now, 0, 0, time.Minute)
		eng := persist.New(storage.NewStore(1, clk, storage.Config{}), dumps, persist.Config{})
		stop := startRun(t, eng)
		defer stop()
		_, _, err := eng.Set("k", []byte("v"))
		t.Logf("Set error: %v; errors.Is(err, cause)=%t", err, errors.Is(err, errDiskFull))
		synctest.Sleep(10 * time.Second) // nobody writes any more
		synctest.Wait()
		st := eng.DumpStats()
		t.Logf("after 10 idle seconds on a broken disk: Save attempts=%d, DumpStats=%+v", dumps.attempts(), st)
		if dumps.attempts() > 20 {
			t.Errorf("no backoff: %d full encode+save attempts in 10 s after ONE write", dumps.attempts())
		}
		if !errors.Is(err, errDiskFull) {
			t.Errorf("the dump failure cause is not carried by the returned error: %v", err)
		}
	})
}

// ENG-06: confirmation latency under a sustained 1 ms write cadence.
func TestReview_sustainedLoadLatency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		eng, dumps := newFakeEngine()
		defer startDumper(t, eng)()
		const writes = 5000
		lat := make([]time.Duration, 0, writes)
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i := range writes {
			wg.Go(func() {
				start := time.Now()
				_, _, err := eng.Set(fmt.Sprintf("key-%d", i%10), []byte("v"))
				if err != nil {
					t.Errorf("Set: %v", err)
				}
				mu.Lock()
				lat = append(lat, time.Since(start))
				mu.Unlock()
			})
			synctest.Sleep(time.Millisecond)
		}
		wg.Wait()
		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		var sum time.Duration
		for _, d := range lat {
			sum += d
		}
		t.Logf("5 s @ 1000 writes/s, dump_debounce=50ms, instantaneous dump: saves=%d  p50=%v  p99=%v  max=%v  mean=%v",
			dumps.saveCount(), lat[len(lat)/2], lat[len(lat)*99/100], lat[len(lat)-1], sum/time.Duration(len(lat)))
		if lat[len(lat)/2] > 100*time.Millisecond {
			t.Errorf("median confirmation latency %v, spec bound is dump_debounce (50ms) + dump time", lat[len(lat)/2])
		}
	})
}

// ENG-05 (engine half): after a fallback start the engine is "clean", so
// Snapshot copies the corrupt main file and reports success.
func TestReview_snapshotAfterFallbackCopiesCorruptMain(t *testing.T) {
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, _ := persist.Open(1, time.Now, time.Minute, persistfile.New(dumpPath), storage.Config{}, persist.Config{})
	stop := startRun(t, eng)
	for _, k := range []string{"a", "b"} {
		if _, _, err := eng.Set(k, []byte("value-"+k)); err != nil {
			t.Fatal(err)
		}
	}
	stop()
	// main = generation 2 (a,b), .prev = generation 1 (a). Corrupt main structurally.
	main, _ := os.ReadFile(dumpPath)
	if err := os.WriteFile(dumpPath, main[:len(main)-7], 0o600); err != nil {
		t.Fatal(err)
	}

	eng2, discarded := persist.Open(1, time.Now, time.Minute, persistfile.New(dumpPath), storage.Config{}, persist.Config{})
	stop2 := startRun(t, eng2)
	defer stop2()
	_, errB := eng2.Get("b")
	t.Logf("fallback start: discarded=%d (no signal that .prev was used); Get(b) err=%v  records=%d",
		discarded, errB, len(eng2.Records()))

	err := eng2.Snapshot("after-fallback")
	snap, _ := os.ReadFile(filepath.Join(dumpPath+".snapshots", "after-fallback"))
	_, _, decErr := persist.Decode(snap)
	t.Logf("Snapshot() err=%v; snapshot decodes: %v", err, decErr)
	if err == nil && decErr != nil {
		t.Errorf("Snapshot reported success but copied the corrupt main: %v; Restore => %v",
			decErr, eng2.Restore("after-fallback"))
	}
}

// ENG-13: the restart watermark ignores records discarded for a checksum fault.
func TestReview_watermarkIgnoresDiscardedRecords(t *testing.T) {
	recs := map[string]storage.Record{}
	for i, k := range []string{"a", "hot"} {
		ver := version.PackVersion(5000, uint16(i+1), 1) // a=(5000,1) hot=(5000,2)
		recs[k] = storage.Record{Version: ver, Data: []byte(k), Checksum: storage.RecordChecksum(k, ver, []byte(k), false)}
	}
	dump, _ := persist.Encode(recs)
	// damage the newest record's payload so that its checksum fails ("hot" -> "hou")
	dump = bytes.Replace(dump, []byte(`"data":"aG90"`), []byte(`"data":"aG91"`), 1)
	got, discarded, err := persist.Decode(dump)
	l, c := persist.Watermark(got)
	t.Logf("discarded=%d err=%v watermark=(%d,%d); the discarded record carried (5000,2)", discarded, err, l, c)
	if c < 2 {
		t.Errorf("watermark (%d,%d) does not dominate the discarded record's version (5000,2): "+
			"the first post-restart Tick re-issues (5000,2,node 1)", l, c)
	}
}

// ENG-15: a key that is not valid UTF-8 is mangled by the JSON encoder, so the
// record fails its own checksum at the next load.
func TestReview_invalidUTF8KeyLostOnReload(t *testing.T) {
	key := "bad\xffkey"
	ver := version.PackVersion(1, 0, 1)
	recs := map[string]storage.Record{key: {Version: ver, Data: []byte("v"), Checksum: storage.RecordChecksum(key, ver, []byte("v"), false)}}
	dump, err := persist.Encode(recs)
	if err != nil {
		t.Logf("Encode rejected the key: %v", err)
		return
	}
	got, discarded, err := persist.Decode(dump)
	t.Logf("dump line: %s", strings.TrimSpace(strings.SplitN(string(dump), "\n", 2)[1]))
	if len(got) != 1 {
		t.Errorf("Encode accepted the key, Decode dropped it: records=%d discarded=%d err=%v", len(got), discarded, err)
	}
}

// ENG-09: Restore reports success for a snapshot whose records were partly discarded.
func TestReview_restoreIgnoresDiscardedSnapshotRecords(t *testing.T) {
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	eng, _ := persist.Open(1, time.Now, time.Minute, persistfile.New(dumpPath), storage.Config{}, persist.Config{})
	stop := startRun(t, eng)
	defer stop()
	for _, k := range []string{"a", "hot"} {
		if _, _, err := eng.Set(k, []byte(k)); err != nil {
			t.Fatal(err)
		}
	}
	if err := eng.Snapshot("s"); err != nil {
		t.Fatal(err)
	}
	snapPath := filepath.Join(dumpPath+".snapshots", "s")
	snap, _ := os.ReadFile(snapPath)
	snap = bytes.Replace(snap, []byte(`"data":"aG90"`), []byte(`"data":"aG91"`), 1)
	if err := os.WriteFile(snapPath, snap, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := eng.Set("hot", []byte("changed-after-snapshot")); err != nil {
		t.Fatal(err)
	}
	err := eng.Restore("s")
	rec, _ := eng.Get("hot")
	t.Logf("Restore err=%v; hot=%q", err, rec.Data)
	if err == nil && string(rec.Data) != "hot" {
		t.Errorf("Restore returned nil although the snapshot record for %q was discarded and not restored (value %q)", "hot", rec.Data)
	}
}

// ENG-04 census: every single-bit flip inside ONE record line of a dump,
// classified by what Decode does with the generation.
func TestReview_bitFlipCensus(t *testing.T) {
	records := make(map[string]storage.Record)
	for i := range 3 {
		key := fmt.Sprintf("svc/v1/key-%d", i)
		ver := version.PackVersion(int64(1_700_000_000_000+i), uint16(i), 7)
		data := bytes.Repeat([]byte{byte(0x41 + i), 0x00, 0xff, 0x7f}, 256) // 1 KiB value
		records[key] = storage.Record{Version: ver, Data: data, Checksum: storage.RecordChecksum(key, ver, data, false)}
	}
	dump, err := persist.Encode(records)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.SplitAfter(dump, []byte("\n"))
	start := len(lines[0]) + len(lines[1]) // flip only inside the 2nd record line
	end := start + len(lines[2])
	var generationLost, recordDiscarded, accepted, silentWrong int
	for pos := start; pos < end; pos++ {
		for bit := range 8 {
			c := bytes.Clone(dump)
			c[pos] ^= 1 << bit
			got, _, err := persist.Decode(c)
			switch {
			case err != nil:
				generationLost++
			case len(got) < len(records):
				recordDiscarded++
			default:
				accepted++
				for k, want := range records {
					if g, ok := got[k]; !ok || g.Version != want.Version || !bytes.Equal(g.Data, want.Data) || g.Deleted != want.Deleted {
						silentWrong++
					}
				}
			}
		}
	}
	total := generationLost + recordDiscarded + accepted
	t.Logf("single-bit flips in one %d-byte record line: total=%d  whole generation rejected=%d (%.0f%%)  record discarded=%d (%.0f%%)  harmless=%d  silently wrong=%d",
		end-start, total, generationLost, 100*float64(generationLost)/float64(total),
		recordDiscarded, 100*float64(recordDiscarded)/float64(total), accepted, silentWrong)
	if generationLost > 0 {
		t.Errorf("%d of %d single-bit faults confined to one record collapse the whole generation", generationLost, total)
	}
}
