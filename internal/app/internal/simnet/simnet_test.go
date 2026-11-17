package simnet_test

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// Shared simulator-test constants: the simulation epoch (every virtual time
// derives from time.UnixMilli(simBaseMs), so no monotonic reading is ever
// involved — the D-02 seam) and the seed-split width mirroring
// internal/app/internal/testutil/rng.go so a replayed seed reproduces the
// exact PCG draw sequence (D-06/D-07).
const (
	simBaseMs    = int64(1730000100000)
	seedHalfBits = 32
)

// simAt returns the virtual time at offset from the simulation epoch.
func simAt(off time.Duration) time.Time {
	return time.UnixMilli(simBaseMs).Add(off)
}

// simRNG builds the seeded generator with testutil.NewRNG's exact seed split
// into the two 32-bit PCG halves, so a replayed seed reproduces the draw
// sequence (D-06/D-07, rng.go's discipline).
func simRNG(seed int64) *rand.Rand {
	hi, lo := uint64(seed>>seedHalfBits), uint64(seed)&(1<<seedHalfBits-1)
	return rand.New(rand.NewPCG(hi, lo))
}

// simDelay is the chaos-style one-way delay draw: 1..10 virtual milliseconds.
func simDelay(rng *rand.Rand) time.Duration {
	return time.Duration(1+rng.IntN(10)) * time.Millisecond
}

// recordsMapsEqual reports whether two record maps are equal in every record
// field — Version, Data, Deleted and Checksum — over the same key set.
func recordsMapsEqual(a, b map[string]storage.Record) bool {
	if len(a) != len(b) {
		return false
	}
	for key, ra := range a {
		rb, ok := b[key]
		if !ok || ra.Version != rb.Version || ra.Deleted != rb.Deleted ||
			ra.Checksum != rb.Checksum || !bytes.Equal(ra.Data, rb.Data) {
			return false
		}
	}
	return true
}

// TestSimnet_tieBreak proves the scheduler's total order: events scheduled at
// the same virtual time execute in Schedule-call (insertion, seq) order, and
// an earlier-time event always executes first regardless of insertion order
// (RESEARCH Pattern 4, Pitfall 1 — the tie-break contract same-seed
// determinism is built on).
func TestSimnet_tieBreak(t *testing.T) {
	t.Parallel()
	sched := simnet.NewScheduler(simAt(0))
	var order []int
	sched.Schedule(simAt(time.Millisecond), func() { order = append(order, 1) })
	sched.Schedule(simAt(time.Millisecond), func() { order = append(order, 2) })
	sched.Schedule(simAt(time.Millisecond), func() { order = append(order, 3) })
	sched.Schedule(simAt(0), func() { order = append(order, 0) })
	sched.RunUntil(simAt(2 * time.Millisecond))
	if !slices.Equal(order, []int{0, 1, 2, 3}) {
		t.Fatalf("events ran in order %v, want [0 1 2 3] — the earlier time first, then equal-time events in insertion (seq) order", order)
	}
}

// simnetScenario is one fixed 3-node scenario shape: a chaos-style policy
// (drops, duplicates, 1..10ms delays, one full-cluster partition window drawn
// from the rng) and a fixed write script crossing that window.
type simnetScenario struct {
	policy  simnet.Policy
	writes  []simnetWrite
	cluster int
}

// simnetWrite is one scripted cluster write: the virtual offset, the node
// index, the key.
type simnetWrite struct {
	off  time.Duration
	node int
	key  string
}

// drawSimnetScenario builds the scenario from rng — every random input (the
// partition window) is drawn here, before any event runs, so the whole run
// is a pure function of the rng sequence.
func drawSimnetScenario(rng *rand.Rand) simnetScenario {
	partStart := time.Duration(100+rng.IntN(100)) * time.Millisecond
	partLen := time.Duration(200+rng.IntN(200)) * time.Millisecond
	return simnetScenario{
		policy: simnet.Policy{
			DropProb: 0.20,
			DupProb:  0.10,
			Delay:    simDelay,
			Partition: func(_, _ string, now time.Time) bool {
				off := now.Sub(simAt(0))
				return off >= partStart && off < partStart+partLen
			},
		},
		writes: []simnetWrite{
			{60 * time.Millisecond, 0, "sim/one"},
			{75 * time.Millisecond, 1, "sim/two"},
			{90 * time.Millisecond, 2, "sim/three"},
			{120 * time.Millisecond, 0, "sim/four"},
			{150 * time.Millisecond, 1, "sim/five"},
		},
		cluster: 3,
	}
}

// runSimnetScenario runs one scenario end to end over a fresh scheduler and
// cluster, seeding the network's fault draws from the same seed, and returns
// the delivery trace digest plus every node's final records (TEST-03's
// same-seed comparison artifacts).
func runSimnetScenario(seed int64) (string, []map[string]storage.Record) {
	rng := simRNG(seed)
	sc := drawSimnetScenario(rng)
	sched := simnet.NewScheduler(simAt(0))
	cluster := simnet.NewCluster(sched, rng, sc.cluster, repl.Config{}, sc.policy)
	sched.RunUntil(simAt(50 * time.Millisecond))
	for _, w := range sc.writes {
		sched.RunUntil(simAt(w.off))
		cluster.Write(w.node, w.key, []byte("value "+w.key))
	}
	sched.RunUntil(simAt(2 * time.Second))
	stores := make([]map[string]storage.Record, cluster.Nodes())
	for i := range stores {
		stores[i] = cluster.Store(i).Records()
	}
	return cluster.Network().TraceDigest(), stores
}

// TestSimnet_determinism proves TEST-03's core property: the same seed
// produces the byte-identical trace digest and identical final stores across
// two fresh scheduler/cluster runs of the identical scenario. The non-empty
// digest assertion gives the proof teeth — an empty trace would compare
// equal vacuously.
func TestSimnet_determinism(t *testing.T) {
	t.Parallel()
	const seed = int64(42)
	digest1, stores1 := runSimnetScenario(seed)
	digest2, stores2 := runSimnetScenario(seed)
	if digest1 == "" {
		t.Fatal("trace digest is empty — no deliveries were logged, so the determinism comparison would be vacuous")
	}
	if digest1 != digest2 {
		t.Fatalf("seed %d produced two different traces (%d vs %d digest bytes) — a nondeterminism source leaked into the simulator", seed, len(digest1), len(digest2))
	}
	if len(stores1) != len(stores2) {
		t.Fatalf("two runs returned %d vs %d node stores", len(stores1), len(stores2))
	}
	for i := range stores1 {
		if !recordsMapsEqual(stores1[i], stores2[i]) {
			t.Fatalf("node %d ended with different stores across the two runs of seed %d", i, seed)
		}
	}
}

// TestSimnet_faultsObserved proves the fault injector has coverage teeth
// (the TestClock_driftUnderChaos both-signs pattern): under the lossy policy
// with a directed partition window, a scripted workload observes drops,
// duplicates, reorders, partitioned sends and — because Broadcast fans out to
// every registered node including the sender — non-zero OwnEcho discards in
// the cores themselves (plan 01's own-echo class firing in simulation).
func TestSimnet_faultsObserved(t *testing.T) {
	t.Parallel()
	const seed = int64(7)
	rng := simRNG(seed)
	sched := simnet.NewScheduler(simAt(0))
	policy := simnet.Policy{
		DropProb: 0.30,
		DupProb:  0.30,
		Delay:    simDelay,
		Partition: func(from, to string, now time.Time) bool {
			return from == "n1" && to == "n2" &&
				!now.Before(simAt(100*time.Millisecond)) && now.Before(simAt(700*time.Millisecond))
		},
	}
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{}, policy)
	sched.RunUntil(simAt(50 * time.Millisecond))
	for i := range 12 {
		sched.RunUntil(simAt(time.Duration(60+i*40) * time.Millisecond))
		cluster.Write(i%3, fmt.Sprintf("sim/faults/%d", i), []byte("payload"))
	}
	sched.RunUntil(simAt(3 * time.Second))

	faults := cluster.Network().FaultStats()
	for _, tc := range []struct {
		name string
		got  uint64
	}{
		{"dropped", faults.Dropped},
		{"duplicated", faults.Duplicated},
		{"reordered", faults.Reordered},
		{"partitioned", faults.Partitioned},
	} {
		if tc.got == 0 {
			t.Fatalf("fault class %s observed 0 times — the injector demonstrably fired nothing in this class (seed %d)", tc.name, seed)
		}
	}
	for i := range cluster.Nodes() {
		if got := cluster.Core(i).Discards().OwnEcho; got == 0 {
			t.Fatalf("node %d observed %d own-echo discards, want > 0 — broadcast must fan out to the sender itself", i, got)
		}
	}
}
