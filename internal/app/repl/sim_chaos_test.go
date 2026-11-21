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
	"strings"
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
	chaosDeleteProb  = 0.30
	chaosDrainProb   = 0.25
	chaosPickSpan    = 1 << 16

	// chaosFreshSeedsEnv parameterizes the fresh-seed batch each run appends
	// to the fixed corpus (Q1: the scheduled job sets larger values);
	// chaosFreshSeedsDef is the fast PR-CI default.
	chaosFreshSeedsEnv = "SUPKV_SIM_FRESH_SEEDS"
	chaosFreshSeedsDef = 1

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
// otherwise a fresh wall-clock draw — either way the caller logs the exact
// replay command so a failing run reproduces byte for byte. The chaos test's
// env branch takes the first path; its corpus and fresh-seed branches draw
// their own seeds, and the scenario suite shares this helper.
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

// chaosCorpusSeedsFromFile reads the committed corpus: one decimal seed per
// line, with blank and #-comment lines ignored for forward compatibility.
func chaosCorpusSeedsFromFile(t *testing.T) []int64 {
	t.Helper()
	data, err := os.ReadFile(chaosCorpusPath)
	if err != nil {
		t.Fatalf("reading the committed chaos seed corpus %s: %v", chaosCorpusPath, err)
	}
	var seeds []int64
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		seed, parseErr := strconv.ParseInt(line, 10, 64)
		if parseErr != nil {
			t.Fatalf("%s line %q is not an int64 seed: %v", chaosCorpusPath, line, parseErr)
		}
		seeds = append(seeds, seed)
	}
	if len(seeds) == 0 {
		t.Fatalf("the committed chaos seed corpus %s carries no seeds", chaosCorpusPath)
	}
	return seeds
}

// chaosFreshSeedCount returns the fresh-seed batch size:
// SUPKV_SIM_FRESH_SEEDS when set (the scheduled job scales it), else the
// fast PR-CI default.
func chaosFreshSeedCount(t *testing.T) int {
	t.Helper()
	raw := os.Getenv(chaosFreshSeedsEnv)
	if raw == "" {
		return chaosFreshSeedsDef
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s=%q is not an integer fresh-seed count: %v", chaosFreshSeedsEnv, raw, err)
	}
	if count < 0 {
		t.Fatalf("%s=%q is negative", chaosFreshSeedsEnv, raw)
	}
	return count
}

// chaosDelay is the chaos one-way delay draw: 1..10 virtual milliseconds.
func chaosDelay(rng *rand.Rand) time.Duration {
	return time.Duration(1+rng.IntN(10)) * time.Millisecond
}

// simHandle returns node index i's network handle (the cluster's n1..nN
// naming in node_id order).
func simHandle(i int) string { return fmt.Sprintf("n%d", i+1) }

// chaosWrite is one scenario operation: the virtual offset from the
// simulation base, the acting node, the write key and payload, and — for a
// deletion — the del flag plus pick, which selects the target among the
// acting node's live keys at execution time (pick modulo the sorted live-key
// count, the D-05 Cluster.Delete seam).
type chaosWrite struct {
	off  time.Duration
	node int
	key  string
	data []byte
	del  bool
	pick int
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
	drain      bool
}

// drawChaosScenario draws one scenario from rng (TEST-03's 2..N shape): 2 to
// chaosMaxNodes nodes, tens of testutil.RandKey-style operations on random
// nodes at random virtual times across ~200 virtual seconds — writes plus
// deletions against live keys (D-05, chaosDeleteProb) — and one to three
// random symmetric partition windows of tens of virtual seconds each; drop
// probability at most 0.25, duplicate at most 0.10 (docs/IDEA.md §Failure
// scenarios' chaos fault set). One in four scenarios is deletion-heavy
// (chaosDrainProb): a closing sweep deletes every key still live at the end,
// so its final state is empty or tombstone-only (the TEST-01 empty-state
// probe).
func drawChaosScenario(rng *rand.Rand) chaosScenario {
	sc := chaosScenario{
		nodes:    2 + rng.IntN(chaosMaxNodes-1),
		dropProb: 0.05 + rng.Float64()*0.20,
		dupProb:  0.02 + rng.Float64()*0.08,
		drain:    rng.Float64() < chaosDrainProb,
	}
	for i := range chaosWriteCount {
		op := chaosWrite{
			off:  time.Duration(rng.IntN(chaosLenMs)) * time.Millisecond,
			node: rng.IntN(sc.nodes),
			key:  testutil.RandKey(rng),
			data: []byte(fmt.Sprintf("chaos-payload-%d", i)),
		}
		if rng.Float64() < chaosDeleteProb {
			op.del = true
			op.pick = rng.IntN(chaosPickSpan)
		}
		sc.writes = append(sc.writes, op)
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
// quiescence and convergence verdicts against the hard cap (with the virtual
// instants of each), the successful deletion count, the cluster's write log
// (the model input) and the node count.
type chaosRun struct {
	digest     string
	stores     []map[string]storage.Record
	faults     simnet.FaultStats
	quiesced   bool
	converged  bool
	quiesceAt  time.Time
	convergeAt time.Time
	deletes    int
	log        []simnet.LoggedWrite
	nodes      int
}

// runChaosScenario runs one whole scenario under one seed: discovery, the
// scripted operations at their virtual times — writes and deletions against
// live keys (the D-05 Cluster.Delete path) — then the quiescence-and-
// convergence drive. Quiescence (no PendingDeliveries, every core's
// UnackedStats().Count zero, every core's QueuedResponses() zero) is
// necessary but not sufficient: while R-05 delays a returning peer's
// knowledge to its next periodic sync, the three seams can hold before every
// store equals the pointwise-supremum model. After quiescence the loop keeps
// stepping and polling until every store equals that model, all within the
// UNCHANGED hard cap of 10 × sync_interval past the last operation. Expiry
// leaves quiesced/converged false for the caller to fail on; the
// quiescence/convergence instants are recorded for the delta distribution.
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
	deletes := 0
	for _, w := range sc.writes {
		sched.RunUntil(at(w.off))
		if w.del {
			key, ok := liveChaosKey(cluster.Store(w.node), w.pick)
			if !ok {
				continue
			}
			_, err := cluster.Delete(w.node, key)
			if err == nil {
				deletes++
			}
			continue
		}
		cluster.Write(w.node, w.key, w.data)
	}
	if sc.drain {
		deletes += drainChaosLiveKeys(cluster)
	}

	capTime := at(sc.writes[len(sc.writes)-1].off + 10*syncInterval)
	model := chaosModelStore(cluster.WriteLog())
	quiesced, converged := false, false
	var quiesceAt, convergeAt time.Time
	for {
		next := sched.Now().Add(chaosQuiesceStep)
		if next.After(capTime) {
			next = capTime
		}
		sched.RunUntil(next)
		if !quiesced && chaosQuiescent(cluster, next) {
			quiesced = true
			quiesceAt = next
		}
		if quiesced && chaosConverged(cluster, model) {
			converged = true
			convergeAt = next
			break
		}
		if !next.Before(capTime) {
			break
		}
	}

	stores := make([]map[string]storage.Record, cluster.Nodes())
	for i := range stores {
		stores[i] = cluster.Store(i).Records()
	}
	return chaosRun{
		digest:     cluster.Network().TraceDigest(),
		stores:     stores,
		faults:     cluster.Network().FaultStats(),
		quiesced:   quiesced,
		converged:  converged,
		quiesceAt:  quiesceAt,
		convergeAt: convergeAt,
		deletes:    deletes,
		log:        cluster.WriteLog(),
		nodes:      cluster.Nodes(),
	}
}

// liveChaosKey returns the deletion target among store's live (non-tombstone)
// keys: sorted key order with pick modulo the live count — deterministic, with
// no map iteration order in the choice. ok is false when nothing is live.
func liveChaosKey(store *storage.Store, pick int) (string, bool) {
	records := store.Records()
	live := make([]string, 0, len(records))
	for key, rec := range records {
		if !rec.Deleted {
			live = append(live, key)
		}
	}
	if len(live) == 0 {
		return "", false
	}
	slices.Sort(live)
	return live[pick%len(live)], true
}

// drainChaosLiveKeys deletes every key the cluster still holds live — the
// deletion-heavy scenario's closing sweep. Nodes are walked in order and each
// takes its smallest live key, so the total live count strictly shrinks and
// the loop terminates; no rng draw or time advance is involved. Each delete
// installs a fresh local tombstone unconditionally (Store.Delete's install),
// so every authored live record is overwritten by a later tombstone from its
// author and the pointwise-supremum model ends empty or tombstone-only.
func drainChaosLiveKeys(cluster *simnet.Cluster) int {
	deletes := 0
	for {
		progressed := false
		for i := range cluster.Nodes() {
			key, ok := liveChaosKey(cluster.Store(i), 0)
			if !ok {
				continue
			}
			_, err := cluster.Delete(i, key)
			if err != nil {
				continue
			}
			deletes++
			progressed = true
		}
		if !progressed {
			return deletes
		}
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

// chaosConverged reports whether every node's store equals the pointwise-
// supremum model — the post-quiescence condition TEST-01 asserts (quiescence
// alone is necessary but not sufficient under the R-05 delayed heal).
func chaosConverged(cluster *simnet.Cluster, model *storage.Store) bool {
	want := model.Records()
	for i := range cluster.Nodes() {
		if !recordMapsEqual(cluster.Store(i).Records(), want) {
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
// duplicates, reorders, partitions and deletes converges — after quiescence
// the run polls until every node's store deep-equals the pointwise-supremum
// model of all operations (Version, Data, Deleted, Checksum; tombstones
// included in the comparison), and the fault coverage assertions hold (all
// four network classes plus deletions fired — a run that dropped nothing or
// deleted nothing fails). With SUPKV_SIM_SEED set the test runs that single
// seed first (the replay path); otherwise it runs every seed of the committed
// corpus as a named subtest plus SUPKV_SIM_FRESH_SEEDS fresh wall-clock runs.
// The seed prints on every run; a failing run replays exactly with the
// printed command.
func TestChaos_convergence(t *testing.T) {
	t.Parallel()
	if os.Getenv(simSeedEnv) != "" {
		chaosAssertConvergence(t, chaosSeed(t))
		return
	}
	for _, seed := range chaosCorpusSeedsFromFile(t) {
		t.Run(fmt.Sprintf("corpus/%d", seed), func(t *testing.T) {
			t.Parallel()
			chaosAssertConvergence(t, seed)
		})
	}
	for i := range chaosFreshSeedCount(t) {
		t.Run(fmt.Sprintf("fresh/%d", i), func(t *testing.T) {
			t.Parallel()
			chaosAssertConvergence(t, time.Now().UnixNano())
		})
	}
}

// chaosAssertConvergence runs one seed and asserts the full TEST-03 evidence:
// the cluster quiesced AND every store reached the pointwise-supremum model
// within the hard cap, the final stores deep-equal the model (Version, Data,
// Deleted, Checksum; tombstones included), and the fault coverage assertions
// hold (all four network classes plus deletions fired — a run that dropped
// nothing or deleted nothing fails). The replay command prints on every run;
// the per-seed quiescence/convergence delta is logged for the 10k proof's
// distribution evidence.
func chaosAssertConvergence(t *testing.T, seed int64) {
	t.Helper()
	t.Logf(chaosReplayFmt, seed)
	run := runChaosScenario(seed)
	if run.quiesced && run.converged {
		t.Logf("seed %d: quiesced at +%v, converged at +%v (delta %v)",
			seed, run.quiesceAt.Sub(at(0)), run.convergeAt.Sub(at(0)), run.convergeAt.Sub(run.quiesceAt))
	}
	if !run.quiesced {
		t.Fatalf("the cluster did not quiesce within 10*sync_interval past the last operation (faults %+v, nodes %d) "+chaosReplayFmt,
			run.faults, run.nodes, seed)
	}
	if !run.converged {
		t.Fatalf("the cluster quiesced but did not reach the pointwise-supremum model within 10*sync_interval past the last operation (faults %+v, nodes %d) "+chaosReplayFmt,
			run.faults, run.nodes, seed)
	}
	model := chaosModelStore(run.log)
	for i := range run.stores {
		if !recordMapsEqual(run.stores[i], model.Records()) {
			t.Fatalf("node %d holds %d records, the pointwise-supremum model holds %d — divergence after convergence "+chaosReplayFmt,
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
		{"deleted", uint64(run.deletes)},
	} {
		if tc.got == 0 {
			t.Fatalf("class %s observed 0 times — the chaos run did not exercise it "+chaosReplayFmt, tc.name, seed)
		}
	}
}

// TestChaos_corpusEmptyStateProbe proves the committed corpus carries the
// TEST-01 empty-state probe: at least one fixed seed's final pointwise-
// supremum model is empty or tombstone-only — a valid convergence point, so
// the convergence assertion is not quietly a live-data-only assertion.
func TestChaos_corpusEmptyStateProbe(t *testing.T) {
	t.Parallel()
	for _, seed := range chaosCorpusSeedsFromFile(t) {
		run := runChaosScenario(seed)
		if !run.quiesced || !run.converged {
			continue // the convergence test owns that failure
		}
		model := chaosModelStore(run.log).Records()
		if chaosRecordsTombstoneOnly(model) {
			t.Logf("seed %d: final model is empty or tombstone-only (%d records, %d deletes)", seed, len(model), run.deletes)
			return
		}
	}
	t.Fatal("no corpus seed ends empty or tombstone-only — the corpus lost its TEST-01 empty-state probe")
}

// chaosRecordsTombstoneOnly reports whether records is empty or every record
// is a tombstone.
func chaosRecordsTombstoneOnly(records map[string]storage.Record) bool {
	for _, rec := range records {
		if !rec.Deleted {
			return false
		}
	}
	return true
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
