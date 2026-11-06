package repl_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// Shared bound-test constants: the bound suite's network profile pins the
// one-way delay at exactly 1 virtual millisecond (RTT 2ms, a known constant —
// RESEARCH Pattern 7's "size the stores so the rate-limited term is
// assertable"), and the normalized-default response rate from docs/IDEA.md
// §Parameters (500/s, spacing 2ms) feeds the heal bound's response term.
// syncInterval and ackTimeoutMax come from the shared retransmit-test
// constants (the same normalized Config defaults).
const (
	simOneWay              = time.Millisecond
	simDefaultResponseRate = 500
	simDiscoveryStep       = 5 * time.Millisecond
	simDiscoveryCap        = 100 * time.Millisecond
)

// simRNG builds the seeded generator with testutil.NewRNG's exact seed split
// into the two 32-bit PCG halves, so a replayed seed reproduces the draw
// sequence (D-06/D-07; the bound tests draw no faults, but the network still
// consumes its rng for every send — determinism by construction).
func simRNG(seed int64) *rand.Rand {
	const seedHalfBits = 32
	hi, lo := uint64(seed>>seedHalfBits), uint64(seed)&(1<<seedHalfBits-1)
	return rand.New(rand.NewPCG(hi, lo))
}

// simFixedDelay returns the bound-test network profile: every datagram
// delivered after exactly simOneWay, no drops, no duplicates; the partition
// predicate (nil: nothing blocked) is each test's deterministic fault.
func simFixedDelay(partition func(from, to string, now time.Time) bool) simnet.Policy {
	return simnet.Policy{
		Delay:     func(*rand.Rand) time.Duration { return simOneWay },
		Partition: partition,
	}
}

// simDiscovered reports whether mutual discovery completed: every node's
// peer table holds every other node.
func simDiscovered(cluster *simnet.Cluster) bool {
	for i := range cluster.Nodes() {
		if len(cluster.Core(i).PeerSnapshot()) != cluster.Nodes()-1 {
			return false
		}
	}
	return true
}

// quiesceDiscovery drives the scheduler until every node knows every other
// node — discovery quiesced before each measurement, so the measured event
// (a write, a heal) starts from steady state, never from discovery noise.
func quiesceDiscovery(t *testing.T, sched *simnet.Scheduler, cluster *simnet.Cluster) {
	t.Helper()
	base := sched.Now()
	for step := simDiscoveryStep; step <= simDiscoveryCap; step += simDiscoveryStep {
		now := base.Add(step)
		sched.RunUntil(now)
		if simDiscovered(cluster) {
			return
		}
	}
	t.Fatalf("mutual discovery did not complete within %v of virtual time", simDiscoveryCap)
}

// recordMapsEqual reports whether two record maps agree on the full record
// (Version, Data, Deleted, Checksum) under every key — the deep-equality the
// convergence assertions use, tombstones included.
func recordMapsEqual(a, b map[string]storage.Record) bool {
	if len(a) != len(b) {
		return false
	}
	for key, ra := range a {
		rb, ok := b[key]
		if !ok || !recordsEqual(ra, rb) {
			return false
		}
	}
	return true
}

// The three documented convergence bounds, each as the named model function
// its test asserts (the hlc_prop_test.go expectedDrift style — the assertion
// and the documented expression stay one artifact). Source, docs/IDEA.md
// lines 99-100: "Convergence bounds: normal operation one RTT; datagram
// loss ≤ `ack_timeout_max`; after a partition heals ≤ 2 × `sync_interval` +
// response time." Bounds are inclusive (flagged assumption #21): convergence
// exactly at the bound passes, one virtual step past it fails.

// boundNormalOneRTT returns the normal-operation convergence bound: one
// round trip over the given one-way delay (docs/IDEA.md:99).
func boundNormalOneRTT(oneWay time.Duration) time.Duration { return 2 * oneWay }

// boundLossEpisode returns the datagram-loss bound: a loss episode heals
// within ack_timeout_max of the send (docs/IDEA.md:100). The pinned reading
// (flagged assumption #21): the bound covers loss episodes whose delivery
// opportunity falls inside the window — the plan 04-02 ladder guarantees
// attempts at 200/600/1400/3000ms from the write, all inside the 5s default
// window; sustained loss past the window is the partition-heal bound's
// territory.
func boundLossEpisode(ackTimeoutMax time.Duration) time.Duration { return ackTimeoutMax }

// boundPartitionHeal returns the partition-heal bound: 2 × sync_interval
// plus the response time, where the response term is the exact limiter
// arithmetic for replaying the given key count — (keys-1) × floor(1s/rate)
// integer nanoseconds, the first response immediate (docs/IDEA.md:100;
// RESEARCH Pattern 7's response-time math, the plan 04-03 spacing pin).
func boundPartitionHeal(syncInterval time.Duration, keys, responseRate int) time.Duration {
	return 2*syncInterval + time.Duration(keys-1)*(time.Second/time.Duration(responseRate))
}

// TestBound_normalOneRTT proves the first documented convergence bound
// (docs/IDEA.md:99-100, REPL-10): in normal operation a write on one node is
// visible on every other node within one round-trip time — asserted
// inclusively at exactly t0+RTT through boundNormalOneRTT, with the negative
// probe at t0 proving the assertion has teeth. All timing is virtual
// (RESEARCH Pitfall 9): the test's wall time is sub-millisecond.
func TestBound_normalOneRTT(t *testing.T) {
	t.Parallel()
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, simRNG(101), 2, repl.Config{}, simFixedDelay(nil))
	quiesceDiscovery(t, sched, cluster)

	t0 := sched.Now()
	written := cluster.Write(0, keyAlpha, []byte("visible within one RTT"))
	_, err := cluster.Store(1).Get(keyAlpha)
	if err == nil {
		t.Fatalf("node B already serves %q at t0 — the bound assertion would be vacuous (teeth probe)", keyAlpha)
	}

	sched.RunUntil(t0.Add(boundNormalOneRTT(simOneWay)))
	served, err := cluster.Store(1).Get(keyAlpha)
	if err != nil {
		t.Fatalf("node B does not serve %q at t0+RTT: %v", keyAlpha, err)
	}
	if !recordsEqual(served, written) {
		t.Fatalf("node B serves %+v at t0+RTT, want the byte-equal written record %+v", served, written)
	}
}

// TestBound_lossWithinAckTimeoutMax proves the second documented convergence
// bound (docs/IDEA.md:99-100, REPL-10): a datagram-loss episode heals within
// ack_timeout_max of the send. The loss episode is deterministic: a directed
// time-window partition of A→B over [t0, t0+1s) drops exactly the first
// three attempts — the initial send at t0 plus the retransmits at +200ms and
// +600ms, the plan 04-02 ladder steps — and delivers the +1400ms attempt one
// delay later; the pinned window reading is boundLossEpisode's doc comment
// (flagged assumption #21). Negative probes at +600ms (still inside the
// window) and at +1400ms (the delivered attempt in flight) give the bound
// teeth; the inclusive assertion runs at exactly send+ack_timeout_max.
func TestBound_lossWithinAckTimeoutMax(t *testing.T) {
	t.Parallel()
	var lossStart, lossEnd time.Time
	policy := simFixedDelay(func(from, to string, now time.Time) bool {
		return from == "n1" && to == "n2" && !now.Before(lossStart) && now.Before(lossEnd)
	})
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, simRNG(102), 2, repl.Config{}, policy)
	quiesceDiscovery(t, sched, cluster)

	t0 := sched.Now()
	lossStart, lossEnd = t0, t0.Add(time.Second)
	written := cluster.Write(0, keyBeta, []byte("heals within ack_timeout_max"))

	sched.RunUntil(t0.Add(600 * time.Millisecond))
	_, err := cluster.Store(1).Get(keyBeta)
	if err == nil {
		t.Fatalf("node B serves %q at t0+600ms — the third dropped attempt must not have reached it", keyBeta)
	}
	sched.RunUntil(t0.Add(1400 * time.Millisecond))
	_, err = cluster.Store(1).Get(keyBeta)
	if err == nil {
		t.Fatalf("node B serves %q at t0+1400ms — the delivered attempt lands one delay later, at +1401ms", keyBeta)
	}

	sched.RunUntil(t0.Add(boundLossEpisode(ackTimeoutMax)))
	served, err := cluster.Store(1).Get(keyBeta)
	if err != nil {
		t.Fatalf("node B does not serve %q by t0+ack_timeout_max (%v): %v", keyBeta, boundLossEpisode(ackTimeoutMax), err)
	}
	if !recordsEqual(served, written) {
		t.Fatalf("node B serves %+v, want the byte-equal written record %+v", served, written)
	}
}

// TestBound_partitionHeal proves the third documented convergence bound
// (docs/IDEA.md:99-100, REPL-10): after a partition heals, every node holds
// the pointwise supremum within 2 × sync_interval + response time — the
// response term computed exactly from the rate limiter's spacing through
// boundPartitionHeal (K=3 distinct keys replayed, (K-1) × floor(1s/500) =
// 4ms — RESEARCH Pattern 7). Three nodes run (N>2 coverage): C is fully
// partitioned from A and B in both directions for a few virtual seconds —
// shorter than peer_timeout, so tables stay and the heal exercises
// retransmission plus the sync exchange — while both sides serve writes (A,
// B and C each write one key). The divergence-at-heal probe proves the
// convergence assertion has teeth; the bound is asserted inclusively.
func TestBound_partitionHeal(t *testing.T) {
	t.Parallel()
	const (
		healKeys     = 3
		partitionLen = 3 * time.Second
	)
	var partStart, partEnd time.Time
	policy := simFixedDelay(func(from, to string, now time.Time) bool {
		blocked := func(a, b string) bool { return from == a && to == b }
		inside := !now.Before(partStart) && now.Before(partEnd)
		return inside && (blocked("n3", "n1") || blocked("n3", "n2") || blocked("n1", "n3") || blocked("n2", "n3"))
	})
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, simRNG(103), 3, repl.Config{}, policy)
	quiesceDiscovery(t, sched, cluster)

	t1 := sched.Now()
	partStart, partEnd = t1, t1.Add(partitionLen)
	sched.RunUntil(t1.Add(500 * time.Millisecond))
	writeA := cluster.Write(0, "bound/heal/a", []byte("written on A during the partition"))
	sched.RunUntil(t1.Add(1000 * time.Millisecond))
	writeB := cluster.Write(1, "bound/heal/b", []byte("written on B during the partition"))
	sched.RunUntil(t1.Add(1500 * time.Millisecond))
	writeC := cluster.Write(2, "bound/heal/c", []byte("written on C during the partition"))

	// The heal instant: the divergence was real — both sides served, and the
	// stores differ (teeth for the convergence assertion).
	sched.RunUntil(partEnd)
	if recordMapsEqual(cluster.Store(0).Records(), cluster.Store(2).Records()) {
		t.Fatal("nodes A and C already hold equal stores at the heal instant — the divergence probe is vacuous")
	}

	// The documented bound, inclusive: every node converges to the pointwise
	// supremum of all three writes by t2 + 2*sync_interval + (K-1)*spacing.
	sched.RunUntil(partEnd.Add(boundPartitionHeal(syncInterval, healKeys, simDefaultResponseRate)))
	for i := range cluster.Nodes() {
		for j := i + 1; j < cluster.Nodes(); j++ {
			if !recordMapsEqual(cluster.Store(i).Records(), cluster.Store(j).Records()) {
				t.Fatalf("nodes %d and %d hold different stores by t2+%v: %d vs %d records",
					i, j, boundPartitionHeal(syncInterval, healKeys, simDefaultResponseRate),
					len(cluster.Store(i).Records()), len(cluster.Store(j).Records()))
			}
		}
	}
	for _, w := range []struct {
		key string
		rec storage.Record
	}{
		{"bound/heal/a", writeA},
		{"bound/heal/b", writeB},
		{"bound/heal/c", writeC},
	} {
		for i := range cluster.Nodes() {
			served, err := cluster.Store(i).Get(w.key)
			if err != nil {
				t.Fatalf("node %d does not serve %q after the heal bound: %v", i, w.key, err)
			}
			if !recordsEqual(served, w.rec) {
				t.Fatalf("node %d serves %+v under %q, want the byte-equal written record %+v", i, served, w.key, w.rec)
			}
		}
	}
}
