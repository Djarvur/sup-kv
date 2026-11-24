package repl_test

// The PERSIST-12 dump↔replication simulator suite: a real persist.Engine over
// a cluster node's shared store, its clock and timers bound to the
// simulator's scheduler (Config.Now/After), so the write-confirmation loop
// runs on virtual time next to the real repl.Core handlers. The scenario
// proves a confirmed write is announced and converges the peer, that a dump
// failure still notifies and retries on the bounded ladder, that the
// shrunk-limit skip is reported at load, and that two same-seed runs are
// identical. Exact ladder instants live in the synctest suite
// (internal/app/persist/backoff_test.go); this file pins the interaction —
// the documented D-11 fallback split.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// The dump-notify scenario constants: the engine timing driven on virtual
// time (the design defaults, D-04), the network's one-way and announcement
// latencies, the drive slack past an announcement, the seed, and the
// shrunk-limit fixture sizes. The engine goroutine is a real goroutine inside
// a testing/synctest bubble (the engine's time is virtual), so synctest.Wait
// is the only idle barrier and no wall-clock constant is needed.
const (
	simDumpDebounce = 50 * time.Millisecond
	simDumpMaxWait  = time.Second
	simDumpOneWay   = time.Millisecond
	simDumpAnnounce = time.Millisecond
	simDumpSlack    = 4 * time.Millisecond
	simDumpSeed     = int64(50707)

	simSkipLimit    = 8
	simSkipValueLen = simSkipLimit + 1
	simSkipVersion  = int64(1730000000000)
)

// errSimDump is the simulator port's injected save failure.
var errSimDump = errors.New("sim: dump failed")

// simDumpFailWindow is the number of leading attempts the failure scenario
// fails; the attempt offsets after the first one follow the ladder's
// 100/200/400 ms rungs, so the fourth attempt succeeds at +700 ms.
const simDumpFailWindow = 3

// simDumpLadder holds the failure scenario's expected attempt offsets from
// the first attempt: one failure retries at 100 ms, the next at +300, the
// last at +700, where the fourth attempt succeeds.
var simDumpLadder = []time.Duration{
	100 * time.Millisecond,
	300 * time.Millisecond,
	700 * time.Millisecond,
}

// virtualAfter adapts the engine's Config.After seam to the simulator: each
// armed duration books one scheduler event that fires the channel at the due
// instant. The scheduler's one-call ScheduleAfter computes the instant and
// enqueues it under one lock, so the engine goroutine cannot split the two
// across another goroutine's clock advance.
func virtualAfter(sched *simnet.Scheduler) func(time.Duration) <-chan time.Time {
	return func(d time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		sched.ScheduleAfter(d, func() { ch <- sched.Now() })
		return ch
	}
}

// simDumpPort is the simulator's in-memory DumpStore: SaveStream encodes the
// snapshot and completes at the virtual instant it runs (the engine already
// waited its debounce or retry delay), records every attempt's instant and
// fails the leading failRemaining attempts with an unclassified error. Its
// snapshot methods are the in-memory stand-ins the engine's interface needs.
type simDumpPort struct {
	sched    *simnet.Scheduler
	mu       sync.Mutex
	fail     int
	attempts []time.Time
	saves    [][]byte
	snaps    map[string][]byte
}

// SaveStream encodes fill into memory, records the attempt on the virtual
// clock and fails while the configured window lasts.
func (p *simDumpPort) SaveStream(fill func(io.Writer) error) error {
	var buf bytes.Buffer
	err := fill(&buf)
	at := p.sched.Now()
	p.mu.Lock()
	p.attempts = append(p.attempts, at)
	failed := p.fail > 0
	if failed {
		p.fail--
	} else if err == nil {
		p.saves = append(p.saves, bytes.Clone(buf.Bytes()))
	}
	p.mu.Unlock()
	if err != nil {
		return err
	}
	if failed {
		return errSimDump
	}
	return nil
}

// Load reports the empty start: the dump-notify scenarios never restart.
func (*simDumpPort) Load(_ time.Time, _ func([]byte) error) port.LoadResult {
	return port.LoadResult{}
}

// CreateSnapshot stores an in-memory copy of the latest completed save.
func (p *simDumpPort) CreateSnapshot(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.saves) == 0 {
		return errSimDump
	}
	if p.snaps == nil {
		p.snaps = make(map[string][]byte)
	}
	p.snaps[name] = bytes.Clone(p.saves[len(p.saves)-1])
	return nil
}

// ReadSnapshot returns an in-memory copy.
func (p *simDumpPort) ReadSnapshot(name string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	data, ok := p.snaps[name]
	if !ok {
		return nil, errSimDump
	}
	return bytes.Clone(data), nil
}

// attemptInstants returns the recorded attempt instants in attempt order.
func (p *simDumpPort) attemptInstants() []time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.attempts)
}

// simLoadPort is the load-only DumpStore of the shrunk-limit report: Load
// hands the fixture generation to the engine's parse oracle and returns the
// configured result.
type simLoadPort struct{ result port.LoadResult }

// SaveStream is unreachable in the load-only scenario.
func (*simLoadPort) SaveStream(func(io.Writer) error) error { return nil }

// Load runs the engine's parse oracle over the fixture generation.
func (p *simLoadPort) Load(_ time.Time, parse func([]byte) error) port.LoadResult {
	if p.result.Dump != nil {
		_ = parse(p.result.Dump) // the fixture parses by construction; the report is what this port proves.
	}
	return p.result
}

// CreateSnapshot is unreachable in the load-only scenario.
func (*simLoadPort) CreateSnapshot(string) error { return errSimDump }

// ReadSnapshot is unreachable in the load-only scenario.
func (*simLoadPort) ReadSnapshot(string) ([]byte, error) { return nil, errSimDump }

// simEngine is the engine-over-cluster harness: a real persist.Engine over
// node 0's store, its clock and timers bound to the scheduler.
type simEngine struct {
	sched   *simnet.Scheduler
	cluster *simnet.Cluster
	port    *simDumpPort
	eng     *persist.Engine
	cancel  context.CancelFunc
	done    chan error
	once    sync.Once
}

// startSimEngine builds and launches the engine over node 0's store, failing
// the leading failAttempts dumps. It must run inside the scenario's synctest
// bubble: the engine goroutine joins the bubble and t.Cleanup's stop runs at
// the bubble's exit.
func startSimEngine(t *testing.T, sched *simnet.Scheduler, cluster *simnet.Cluster, failAttempts int) *simEngine {
	t.Helper()
	dumps := &simDumpPort{sched: sched, fail: failAttempts}
	eng := persist.New(cluster.Store(0), dumps, persist.Config{
		Debounce: simDumpDebounce,
		MaxWait:  simDumpMaxWait,
		Now:      sched.Now,
		After:    virtualAfter(sched),
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	h := &simEngine{sched: sched, cluster: cluster, port: dumps, eng: eng, cancel: cancel, done: done}
	t.Cleanup(func() { h.stop(t) })
	return h
}

// stop cancels the engine and reads Run's result after the runtime's idle
// barrier: synctest.Wait returns only once the engine goroutine has exited, so
// the buffered result must already be there.
func (h *simEngine) stop(t *testing.T) {
	t.Helper()
	h.once.Do(func() {
		h.cancel()
		synctest.Wait()
		select {
		case err := <-h.done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Run() error = %v, want context.Canceled", err)
			}
		default:
			t.Error("Run() did not return after cancel — the engine goroutine outlived its bubble")
		}
	})
}

// advanceTo drives virtual time to target and then waits for the engine
// goroutine to reach its next durable block: RunUntil executed every event due
// at or before target, and synctest.Wait proves the engine consumed them and
// re-armed before the caller advances again.
func (h *simEngine) advanceTo(t *testing.T, target time.Time) {
	t.Helper()
	h.sched.RunUntil(target)
	synctest.Wait()
}

// writeAndConfirm starts a Set, waits for its debounce dump at the exact
// virtual instant and returns the write's start instant and result.
func (h *simEngine) writeAndConfirm(t *testing.T, key, value string) (time.Time, error) {
	t.Helper()
	synctest.Wait() // the engine is idle at the write's epoch start
	start := h.sched.Now()
	setRet := make(chan error, 1)
	go func() {
		_, _, err := h.eng.Set(key, []byte(value))
		setRet <- err
	}()
	synctest.Wait() // the Set blocks on its waiter; the engine consumed the nudge and armed the debounce
	h.sched.RunUntil(start.Add(simDumpDebounce))
	synctest.Wait() // the dump completed and the engine re-armed clean
	var err error
	select {
	case err = <-setRet:
	default:
		t.Fatalf("no Set result at the dump instant %v after the idle barrier — the engine did not confirm the write",
			start.Add(simDumpDebounce))
	}
	return start, err
}

// notify announces node 0's stored record and drives the announcement and
// its reply to quiescence.
func (h *simEngine) notify(t *testing.T, key string) time.Time {
	t.Helper()
	at := h.sched.Now()
	h.cluster.NotifyInstalled(0, key)
	h.advanceTo(t, at.Add(simDumpSlack))
	return at
}

// records returns both nodes' full record maps — the replay comparison
// artifact.
func records(cluster *simnet.Cluster) [2]map[string]storage.Record {
	var out [2]map[string]storage.Record
	for i := range out {
		out[i] = cluster.Store(i).Records()
	}
	return out
}

// offsetsFrom first returns every instant's offset from first.
func offsetsFrom(first time.Time, instants []time.Time) []time.Duration {
	out := make([]time.Duration, len(instants))
	for i, at := range instants {
		out[i] = at.Sub(first)
	}
	return out
}

// simDumpRun is one scenario run's observable outcome.
type simDumpRun struct {
	firstAttempt time.Duration // the first dump attempt, relative to the write's start
	attempts     []time.Duration
	confirmAt    time.Duration
	notifyAt     time.Duration
	setErr       error
	nodes        [2]map[string]storage.Record
}

// newDumpCluster builds the two-node cluster, quiesces discovery and starts
// the engine over node 0 with the requested failure window.
func newDumpCluster(t *testing.T, failAttempts int) (*simnet.Scheduler, *simnet.Cluster, *simEngine) {
	t.Helper()
	sched := simnet.NewScheduler(at(0))
	policy := simnet.Policy{
		Delay:         func(*rand.Rand) time.Duration { return simDumpOneWay },
		AnnounceDelay: func(port.Message) time.Duration { return simDumpAnnounce },
	}
	cluster := simnet.NewCluster(sched, simRNG(simDumpSeed), 2, repl.Config{}, policy)
	quiesceDiscovery(t, sched, cluster)
	eng := startSimEngine(t, sched, cluster, failAttempts)
	return sched, cluster, eng
}

// runDumpConfirm runs the confirmation-then-notification scenario: one write
// confirms at the debounce dump instant, its announcement converges the peer,
// and the announcement follows the confirmation.
func runDumpConfirm(t *testing.T) simDumpRun {
	t.Helper()
	const key = "dump/confirm"
	sched, cluster, eng := newDumpCluster(t, 0)
	start, err := eng.writeAndConfirm(t, key, "confirmed-value")
	if err != nil {
		t.Fatalf("Set(%q) error = %v, want nil from the confirming dump", key, err)
	}
	confirmAt := sched.Now().Sub(start)
	notifyAt := eng.notify(t, key).Sub(start)
	instants := eng.port.attemptInstants()
	return simDumpRun{
		firstAttempt: instants[0].Sub(start),
		attempts:     offsetsFrom(instants[0], instants),
		confirmAt:    confirmAt,
		notifyAt:     notifyAt,
		setErr:       err,
		nodes:        records(cluster),
	}
}

// runDumpFailure runs the failure-window scenario: the first three attempts
// fail, Set returns ErrDumpFailed, the announcement still converges the peer,
// and the retries walk the ladder to the recovering fourth attempt.
func runDumpFailure(t *testing.T) simDumpRun {
	t.Helper()
	const key = "dump/failure"
	sched, cluster, eng := newDumpCluster(t, simDumpFailWindow)
	start, err := eng.writeAndConfirm(t, key, "failed-window-value")
	if !errors.Is(err, persist.ErrDumpFailed) {
		t.Fatalf("Set(%q) error = %v, want ErrDumpFailed", key, err)
	}
	confirmAt := sched.Now().Sub(start)
	notifyAt := eng.notify(t, key).Sub(start)
	first := eng.port.attemptInstants()[0]
	for i, offset := range simDumpLadder {
		eng.advanceTo(t, first.Add(offset))
		if got := len(eng.port.attemptInstants()); got != i+2 {
			t.Fatalf("attempts by %v = %d, want %d — the retry ladder is not signalling at %v",
				offset, got, i+2, offset)
		}
	}
	// The success cleared the ladder: nothing fires past the recovering
	// attempt.
	eng.advanceTo(t, first.Add(simDumpLadder[len(simDumpLadder)-1]+simDumpDebounce))
	if got := len(eng.port.attemptInstants()); got != len(simDumpLadder)+1 {
		t.Fatalf("attempts after the recovery = %d, want %d — a clean engine must stop retrying",
			got, len(simDumpLadder)+1)
	}
	return simDumpRun{
		firstAttempt: first.Sub(start),
		attempts:     offsetsFrom(first, eng.port.attemptInstants()),
		confirmAt:    confirmAt,
		notifyAt:     notifyAt,
		setErr:       err,
		nodes:        records(cluster),
	}
}

// TestScenario_dumpNotify proves the dump↔replication interaction on virtual
// time: a confirmed write is announced and the peer converges, a failure
// window returns ErrDumpFailed but still notifies while the retries follow the
// bounded ladder, the shrunk-limit skip is reported on the Open-built engine,
// and two same-seed runs are identical. Every scenario run owns its synctest
// bubble (created here, in the subtest body) and its assertions run on the
// captured result after the bubble returned, where the engine goroutine has
// already been stopped by the bubble's cleanup.
//
//nolint:paralleltest // a bubble forbids t.Parallel; the four scenario runs are independent bubbles, not shared-state subtests.
func TestScenario_dumpNotify(t *testing.T) {
	t.Run("a confirmed write is announced and converges the peer", func(t *testing.T) {
		var run simDumpRun
		synctest.Test(t, func(t *testing.T) { run = runDumpConfirm(t) })
		if run.firstAttempt != simDumpDebounce {
			t.Errorf("the first attempt landed %v after the write, want the debounce instant %v",
				run.firstAttempt, simDumpDebounce)
		}
		if len(run.attempts) != 1 {
			t.Fatalf("attempts = %v, want exactly the one confirming dump", run.attempts)
		}
		if run.confirmAt != simDumpDebounce {
			t.Errorf("Set returned %v after the write, want the dump instant %v", run.confirmAt, simDumpDebounce)
		}
		if run.notifyAt < run.confirmAt {
			t.Errorf("the announcement at %v preceded the confirmation at %v", run.notifyAt, run.confirmAt)
		}
		for i, recs := range run.nodes {
			rec, ok := recs["dump/confirm"]
			if !ok || string(rec.Data) != "confirmed-value" {
				t.Errorf("node %d holds dump/confirm = (%q, %t), want the confirmed value", i, rec.Data, ok)
			}
		}
	})
	t.Run("a failure window still notifies and retries on the ladder", func(t *testing.T) {
		var run simDumpRun
		synctest.Test(t, func(t *testing.T) { run = runDumpFailure(t) })
		if run.confirmAt != simDumpDebounce {
			t.Errorf("Set returned %v after the write, want the first attempt's instant %v", run.confirmAt, simDumpDebounce)
		}
		want := append([]time.Duration{0}, simDumpLadder...)
		if !slices.Equal(run.attempts, want) {
			t.Errorf("attempt offsets = %v, want %v — the ladder did not decelerate", run.attempts, want)
		}
		for i, recs := range run.nodes {
			rec, ok := recs["dump/failure"]
			if !ok || string(rec.Data) != "failed-window-value" {
				t.Errorf("node %d holds dump/failure = (%q, %t), want the announced in-memory value — a failure must still notify",
					i, rec.Data, ok)
			}
		}
	})
	t.Run("a shrunk limit is reported at load", func(t *testing.T) {
		records := map[string]storage.Record{
			"skip/fat":  {Version: version.PackVersion(simSkipVersion, 0, 1), Data: bytes.Repeat([]byte("f"), simSkipValueLen)},
			"skip/slim": {Version: version.PackVersion(simSkipVersion, 1, 1), Data: []byte("slim")},
		}
		for key, rec := range records {
			rec.Checksum = storage.RecordChecksum(key, rec.Version, rec.Data, rec.Deleted)
			records[key] = rec
		}
		dump, err := persist.Encode(records, storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("Encode(fixture) error = %v, want nil", err)
		}
		loader := &simLoadPort{result: port.LoadResult{
			Outcome:    port.LoadMain,
			Generation: port.GenMain,
			Dump:       dump,
		}}
		eng, discarded := persist.Open(1, func() time.Time { return at(0) }, time.Minute, loader,
			storage.Config{ValueMaxBytes: simSkipLimit}, persist.Config{})
		if discarded != 1 {
			t.Fatalf("Open() discarded = %d, want 1 (the over-limit record)", discarded)
		}
		stats := eng.LoadStats()
		if !stats.Loaded || stats.Outcome != port.LoadMain || stats.Decoded.ValueLimit != 1 {
			t.Fatalf("LoadStats() = %+v, want a loaded Main with one value_limit discard", stats)
		}
		_, err = eng.Get("skip/fat")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get(skip/fat) error = %v, want ErrNotFound (skipped at load)", err)
		}
		rec, err := eng.Get("skip/slim")
		if err != nil || string(rec.Data) != "slim" {
			t.Errorf("Get(skip/slim) = (%q, %v), want the in-limit record", rec.Data, err)
		}
	})
	t.Run("two same-seed runs are identical", func(t *testing.T) {
		// One bubble per run: the engine goroutine's cleanup runs at bubble
		// exit, so two runs in one bubble would leave the first engine alive.
		var first, second simDumpRun
		synctest.Test(t, func(t *testing.T) { first = runDumpConfirm(t) })
		synctest.Test(t, func(t *testing.T) { second = runDumpConfirm(t) })
		if !slices.Equal(first.attempts, second.attempts) ||
			first.confirmAt != second.confirmAt ||
			first.notifyAt != second.notifyAt {
			t.Fatalf("same-seed replay diverged: attempts %v/%v, confirm %v/%v, notify %v/%v",
				first.attempts, second.attempts, first.confirmAt, second.confirmAt, first.notifyAt, second.notifyAt)
		}
		for i := range first.nodes {
			if !recordMapsEqual(first.nodes[i], second.nodes[i]) {
				t.Fatalf("same-seed replay diverged on node %d's final records", i)
			}
		}
	})
}

// TestScenario_tiebreakConverges proves the cross-node equal-version law plan
// 05-03 defers here: two nodes seeded with the same key and version but
// different Data converge to the law's winner once Node 0's
// Cluster.NotifyInstalled announces its record. The announcement is a real
// DataReplication on the wire; the merge-only seed emits none. Both
// directions run — the announcing node holds the winning content, and the
// receiving node holds it and answers with its stored record.
func TestScenario_tiebreakConverges(t *testing.T) {
	t.Parallel()
	const key = "tiebreak/dump"
	base := version.PackVersion(simSkipVersion, 7, 1)
	winner := []byte("zzz-tiebreak-winner")
	loser := []byte("aaa-tiebreak-loser")
	cases := []struct {
		name         string
		node0, node1 []byte
	}{
		{"the announcing node holds the winner", winner, loser},
		{"the receiving node holds the winner", loser, winner},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sched := simnet.NewScheduler(at(0))
			cluster := simnet.NewCluster(sched, simRNG(simDumpSeed), 2, repl.Config{}, simFixedDelay(nil))
			quiesceDiscovery(t, sched, cluster)
			cluster.ReplaceNodeWithState(0, scnRunID(1, 5), 0,
				map[string]storage.Record{key: {Version: base, Data: tc.node0}})
			cluster.ReplaceNodeWithState(1, scnRunID(2, 6), 0,
				map[string]storage.Record{key: {Version: base, Data: tc.node1}})
			if trace := cluster.Network().TraceDigest(); strings.Contains(trace, "port.DataReplication") {
				t.Fatalf("the merge-only seed emitted a DataReplication:\n%s", trace)
			}
			cluster.NotifyInstalled(0, key)
			if !scnQuiesce(sched, cluster) {
				t.Fatalf("the cluster did not quiesce after the announcement %s",
					scnReplay(simDumpSeed, "tiebreakConverges"))
			}
			want := storage.Record{Version: base, Data: winner}
			for i := range cluster.Nodes() {
				got, ok := cluster.Store(i).Lookup(key)
				if !ok || !storage.RecordsEqual(got, want) {
					t.Fatalf("node %d holds %q at version %x, want the law's winner %q",
						i, got.Data, got.Version[:], want.Data)
				}
			}
			if trace := cluster.Network().TraceDigest(); !strings.Contains(trace, "port.DataReplication") {
				t.Fatalf("the notification path emitted no DataReplication:\n%s", trace)
			}
			if log := cluster.WriteLog(); len(log) != 1 || log[0].Key != key {
				t.Fatalf("WriteLog() = %v, want the one announced record", log)
			}
		})
	}
}
