package repl_test

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// Chaos-test constants: the seed environment variable (D-06's locked-text
// replay hook — the only hook that works when the test binary runs without
// -args, while testutil's -seed flag remains the property tests' hook; the
// flag-vs-env reconciliation is pinned in the plan's flagged assumptions),
// the printed replay command every run and every failure names, the scenario
// shape (2..5 nodes over ~200 virtual seconds, tens of writes) and the
// quiescence drive's virtual step. The hard quiescence cap is 10 ×
// sync_interval past the last write (the shared normalized-default
// constant).
const (
	simSeedEnv       = "SUPKV_SIM_SEED"
	chaosReplayFmt   = "replay: SUPKV_SIM_SEED=%d go test ./internal/app/repl/ -run '^TestChaos_convergence$'"
	chaosMaxNodes    = 5
	chaosLenMs       = 200_000
	chaosWriteCount  = 40
	chaosPartMax     = 3
	chaosPartMinMs   = 5_000
	chaosPartSpanMs  = 150_000
	chaosDelayMinMs  = 10_000
	chaosDelaySpanMs = 30_000
	chaosQuiesceStep = 100 * time.Millisecond

	// The committed fixed seed corpus (D-06): one decimal int64 seed per
	// line and nothing else, so `head -1` of the file is always a valid
	// SUPKV_SIM_SEED replay argument. The seeds are drawn from one PCG
	// stream whose fixed constant is chaosCorpusGenSeed; regeneration is
	// byte-exact on every machine (the generator test owns the format).
	chaosCorpusPath     = "testdata/chaos-seeds.txt"
	chaosCorpusGenSeed  = int64(0x5eedc0de5eed)
	chaosCorpusCount    = 16
	chaosCorpusRegenEnv = "SUPKV_SIM_CORPUS_REGEN"
)

// chaosSeed returns this run's seed (D-06): SUPKV_SIM_SEED when set,
// otherwise a fresh wall-clock draw — either way the test logs the exact
// replay command on every run so a failing run reproduces byte for byte.
func chaosSeed(t *testing.T) int64 {
	t.Helper()
	if raw := os.Getenv(simSeedEnv); raw != "" {
		seed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			t.Fatalf("%s=%q is not an int64 seed: %v", simSeedEnv, raw, err)
		}
		return seed
	}
	return time.Now().UnixNano()
}

// chaosCorpusSeeds draws the committed corpus's seeds from one PCG stream:
// simRNG splits the fixed constant into the generator's two 32-bit halves
// exactly as a replayed run seed is split, and math/rand/v2's PCG is
// specified, so every machine regenerates the identical sequence (D-06).
func chaosCorpusSeeds() []int64 {
	rng := simRNG(chaosCorpusGenSeed)
	seeds := make([]int64, 0, chaosCorpusCount)
	for range chaosCorpusCount {
		seeds = append(seeds, rng.Int64())
	}
	return seeds
}

// chaosCorpusFile renders the corpus in its committed format: one decimal
// seed per line and a trailing newline, nothing else — no header comment, so
// `head -1` of the file stays a valid replay seed.
func chaosCorpusFile(seeds []int64) []byte {
	var out []byte
	for _, seed := range seeds {
		out = strconv.AppendInt(out, seed, 10)
		out = append(out, '\n')
	}
	return out
}

// chaosCorpusLoad returns the committed corpus bytes, or want when the file
// is missing or SUPKV_SIM_CORPUS_REGEN=1 asked for a rewrite — the write gate
// that keeps a green run from silently changing the committed corpus.
func chaosCorpusLoad(t *testing.T, want []byte) []byte {
	t.Helper()
	got, err := os.ReadFile(chaosCorpusPath)
	missing := errors.Is(err, os.ErrNotExist)
	if err != nil && !missing {
		t.Fatalf("reading %s: %v", chaosCorpusPath, err)
	}
	if !missing && os.Getenv(chaosCorpusRegenEnv) != "1" {
		return got
	}
	err = os.MkdirAll(filepath.Dir(chaosCorpusPath), 0o750)
	if err != nil {
		t.Fatalf("creating the corpus directory: %v", err)
	}
	err = os.WriteFile(chaosCorpusPath, want, 0o600)
	if err != nil {
		t.Fatalf("writing regenerated %s: %v", chaosCorpusPath, err)
	}
	t.Logf("regenerated %s", chaosCorpusPath)
	return want
}

// TestChaos_seedCorpusGenerator proves the committed corpus is exactly the
// generator's output (D-06, T-2-12): every run regenerates the bytes in
// memory twice — the second run proving determinism — and compares them with
// the committed file, failing on any drift. The write itself is gated: the
// file is rewritten only when it is missing or SUPKV_SIM_CORPUS_REGEN=1, so a
// green run never silently changes the committed corpus.
func TestChaos_seedCorpusGenerator(t *testing.T) {
	t.Parallel()
	want := chaosCorpusFile(chaosCorpusSeeds())
	again := chaosCorpusFile(chaosCorpusSeeds())
	if !bytes.Equal(want, again) {
		t.Fatal("corpus regeneration is nondeterministic: two in-process runs differ")
	}
	got := chaosCorpusLoad(t, want)
	if !bytes.Equal(got, want) {
		t.Fatalf("%s drifts from the generator (%d committed bytes vs %d generated); rerun with %s=1 to regenerate",
			chaosCorpusPath, len(got), len(want), chaosCorpusRegenEnv)
	}
}

// chaosDelay is the chaos one-way delay draw: 1..10 virtual milliseconds.
func chaosDelay(rng *rand.Rand) time.Duration {
	return time.Duration(1+rng.IntN(10)) * time.Millisecond
}

// simHandle returns node index i's network handle (the cluster's n1..nN
// naming in node_id order).
func simHandle(i int) string { return fmt.Sprintf("n%d", i+1) }

// chaosWrite is one scenario write: the virtual offset from the simulation
// base, the writing node, the key and the payload.
type chaosWrite struct {
	off  time.Duration
	node int
	key  string
	data []byte
}

// chaosPartition is one symmetric partition window: the directed pair is
// blocked in both directions over [start, end).
type chaosPartition struct {
	a, b  int
	end   time.Duration
	start time.Duration
}

// chaosScenario is the drawn scenario data — every random input fixed
// before any event runs, so the whole run is a pure function of the rng
// sequence.
type chaosScenario struct {
	nodes      int
	writes     []chaosWrite
	partitions []chaosPartition
	dropProb   float64
	dupProb    float64
}

// drawChaosScenario draws one scenario from rng (TEST-03's 2..N shape): 2 to
// chaosMaxNodes nodes, tens of testutil.RandKey-style writes on random nodes
// at random virtual times across ~200 virtual seconds, and one to three
// random symmetric partition windows of tens of virtual seconds each; drop
// probability at most 0.25, duplicate at most 0.10 (docs/IDEA.md §Failure
// scenarios' chaos fault set).
func drawChaosScenario(rng *rand.Rand) chaosScenario {
	sc := chaosScenario{
		nodes:    2 + rng.IntN(chaosMaxNodes-1),
		dropProb: 0.05 + rng.Float64()*0.20,
		dupProb:  0.02 + rng.Float64()*0.08,
	}
	for i := range chaosWriteCount {
		sc.writes = append(sc.writes, chaosWrite{
			off:  time.Duration(rng.IntN(chaosLenMs)) * time.Millisecond,
			node: rng.IntN(sc.nodes),
			key:  testutil.RandKey(rng),
			data: []byte(fmt.Sprintf("chaos-payload-%d", i)),
		})
	}
	slices.SortFunc(sc.writes, func(a, b chaosWrite) int { return cmp.Compare(a.off, b.off) })
	for range 1 + rng.IntN(chaosPartMax) {
		a := rng.IntN(sc.nodes - 1)
		b := a + 1 + rng.IntN(sc.nodes-1-a)
		start := time.Duration(chaosPartMinMs+rng.IntN(chaosPartSpanMs)) * time.Millisecond
		end := start + time.Duration(chaosDelayMinMs+rng.IntN(chaosDelaySpanMs))*time.Millisecond
		sc.partitions = append(sc.partitions, chaosPartition{a: a, b: b, start: start, end: end})
	}
	return sc
}

// blocked reports whether the directed pair (from, to) is inside a partition
// window at now — the scenario's partition predicate, symmetric per window.
func (sc chaosScenario) blocked(from, to string, now time.Time) bool {
	off := now.Sub(at(0))
	for _, p := range sc.partitions {
		ha, hb := simHandle(p.a), simHandle(p.b)
		pair := (from == ha && to == hb) || (from == hb && to == ha)
		if pair && off >= p.start && off < p.end {
			return true
		}
	}
	return false
}

// chaosRun is one completed scenario run's observable outcome: the delivery
// trace digest, every node's final records, the fault counters, the
// quiescence verdict against the hard cap, the cluster's write log (the
// model input) and the node count.
type chaosRun struct {
	digest   string
	stores   []map[string]storage.Record
	faults   simnet.FaultStats
	quiesced bool
	log      []simnet.LoggedWrite
	nodes    int
}

// runChaosScenario runs one whole scenario under one seed: discovery, the
// scripted writes at their virtual times, then the quiescence drive — no
// PendingDeliveries, every core's UnackedStats().Count zero and every
// core's QueuedResponses() zero (the plan 04-02/04-03 seams), with a hard
// virtual cap of 10 × sync_interval past the last write whose expiry leaves
// quiesced false for the caller to fail on.
func runChaosScenario(seed int64) chaosRun {
	rng := simRNG(seed)
	sc := drawChaosScenario(rng)
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, sc.nodes, repl.Config{}, simnet.Policy{
		DropProb:  sc.dropProb,
		DupProb:   sc.dupProb,
		Delay:     chaosDelay,
		Partition: sc.blocked,
	})
	sched.RunUntil(at(50 * time.Millisecond))
	for _, w := range sc.writes {
		sched.RunUntil(at(w.off))
		cluster.Write(w.node, w.key, w.data)
	}

	capTime := at(sc.writes[len(sc.writes)-1].off + 10*syncInterval)
	quiesced := false
	for !quiesced {
		next := sched.Now().Add(chaosQuiesceStep)
		if next.After(capTime) {
			next = capTime
		}
		sched.RunUntil(next)
		quiesced = chaosQuiescent(cluster, next)
		if !next.Before(capTime) {
			break
		}
	}

	stores := make([]map[string]storage.Record, cluster.Nodes())
	for i := range stores {
		stores[i] = cluster.Store(i).Records()
	}
	return chaosRun{
		digest:   cluster.Network().TraceDigest(),
		stores:   stores,
		faults:   cluster.Network().FaultStats(),
		quiesced: quiesced,
		log:      cluster.WriteLog(),
		nodes:    cluster.Nodes(),
	}
}

// chaosQuiescent reports the three-seam quiescence condition: no
// scheduled-but-undelivered network events, no unacked entries on any core,
// no queued rate-limited responses on any core.
func chaosQuiescent(cluster *simnet.Cluster, now time.Time) bool {
	if cluster.Network().PendingDeliveries() != 0 {
		return false
	}
	for i := range cluster.Nodes() {
		if cluster.Core(i).UnackedStats(now).Count != 0 || cluster.Core(i).QueuedResponses() != 0 {
			return false
		}
	}
	return true
}

// chaosModelStore builds the pointwise-supremum model over the cluster's
// write log: every captured (key, record) joined by Merge — merge order is
// irrelevant, the LWW join is commutative (CORE-06) — over a plain store
// parked at the simulation base.
func chaosModelStore(log []simnet.LoggedWrite) *storage.Store {
	model := storage.NewStore(0, clock.NewClock(func() time.Time { return at(0) }, 0, 0, time.Minute), storage.Config{})
	for _, w := range log {
		_, _, _ = model.Merge(w.Key, w.Rec)
	}
	return model
}

// TestChaos_convergence proves the phase's own acceptance criterion (TEST-03,
// D-06/D-07): a 2..N-node cluster over a seeded virtual network that drops,
// duplicates, reorders and partitions datagrams converges — after quiescence
// every node's store deep-equals the pointwise-supremum model of all writes
// (Version, Data, Deleted, Checksum; tombstones included in the comparison),
// and the fault coverage assertions hold (all four classes fired — a run
// that dropped nothing fails). The seed prints on every run; a failing run
// replays exactly with the printed command.
func TestChaos_convergence(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	t.Logf(chaosReplayFmt, seed)
	run := runChaosScenario(seed)
	if !run.quiesced {
		t.Fatalf("the cluster did not quiesce within 10*sync_interval past the last write (faults %+v, nodes %d) "+chaosReplayFmt,
			run.faults, run.nodes, seed)
	}
	model := chaosModelStore(run.log)
	for i := range run.stores {
		if !recordMapsEqual(run.stores[i], model.Records()) {
			t.Fatalf("node %d holds %d records, the pointwise-supremum model holds %d — divergence after quiescence "+chaosReplayFmt,
				i, len(run.stores[i]), len(model.Records()), seed)
		}
	}
	for _, tc := range []struct {
		name string
		got  uint64
	}{
		{"dropped", run.faults.Dropped},
		{"duplicated", run.faults.Duplicated},
		{"reordered", run.faults.Reordered},
		{"partitioned", run.faults.Partitioned},
	} {
		if tc.got == 0 {
			t.Fatalf("fault class %s observed 0 times — the chaos run did not exercise this class "+chaosReplayFmt, tc.name, seed)
		}
	}
}

// TestChaos_determinismReplay proves the fixed-seed determinism the phase
// gate runs on (D-06/D-07, TEST-03): the identical scenario definition run
// twice in-process under one fixed seed, through two fresh schedulers and
// clusters, produces the byte-identical trace digest and identical final
// stores — deterministic by construction, no wall clock anywhere in the
// decision path.
func TestChaos_determinismReplay(t *testing.T) {
	t.Parallel()
	const seed = int64(4242)
	first := runChaosScenario(seed)
	second := runChaosScenario(seed)
	if first.digest != second.digest {
		t.Fatalf("seed %d produced two different traces (%d vs %d digest bytes)", seed, len(first.digest), len(second.digest))
	}
	if len(first.stores) != len(second.stores) {
		t.Fatalf("two runs of seed %d returned %d vs %d node stores", seed, len(first.stores), len(second.stores))
	}
	for i := range first.stores {
		if !recordMapsEqual(first.stores[i], second.stores[i]) {
			t.Fatalf("node %d ended with different stores across the two runs of seed %d", i, seed)
		}
	}
}
