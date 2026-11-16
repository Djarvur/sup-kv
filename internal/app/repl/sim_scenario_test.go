package repl_test

import (
	"bytes"
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// The TEST-04 simulator scenario suite (plan 06-05): the deterministic,
// virtual-time half of the failure-scenario coverage — wall-clock jumps
// live and across a restart, a far-ahead node, partitions of seconds and of
// days, one-way connectivity, datagram/Ack loss with duplicates and
// reordering, ~1000 writes/s bursts, empty-disk recovery at the
// sync_response_rate and restart-via-RunID. Every scenario follows the
// chaos skeleton (sim_chaos_test.go): all random inputs are drawn from the
// seeded rng before any event runs, the run is a pure function of the seed,
// quiescence is the three-seam condition (chaosQuiescent) and convergence
// is asserted against the pointwise-supremum model of the write log
// (chaosModelStore + recordMapsEqual) — never against a trace, never
// against wall-clock precision (the CRDT merge laws make delivery order
// irrelevant; the prohibitions of plan 06-05).

// scnRunID returns a scenario run id: nodeID's decimal rendering with the
// final digit forced to d, so replacement nodes announce run ids that are
// deterministic, distinct from every simRunID a cluster derives (their
// final digit is a node id 1..9) and distinct per generation.
func scnRunID(nodeID uint16, d byte) [16]byte {
	var id [16]byte
	copy(id[:], fmt.Sprintf("%016d", nodeID))
	id[15] = '0' + d
	return id
}

// scnReplay returns the copy-paste replay command every scenario run and
// failure names (D-06: SUPKV_SIM_SEED is the replay hook that works without
// -args; the chaos-suite convention).
func scnReplay(seed int64, test string) string {
	return fmt.Sprintf("replay: SUPKV_SIM_SEED=%d go test ./internal/app/repl/ -run '^TestScenario_%s$'", seed, test)
}

// scnQuiesce drives virtual time forward from the scheduler's current time
// until the three-seam quiescence condition holds (no pending deliveries,
// no unacked entries, no queued responses — chaosQuiescent) or the chaos
// suite's hard cap of 10 × sync_interval passes; it reports whether
// quiescence was observed, leaving the scheduler's clock AT the quiescence
// instant when it was (the caller measures elapsed virtual time off
// sched.Now()).
func scnQuiesce(sched *simnet.Scheduler, cluster *simnet.Cluster) bool {
	capTime := sched.Now().Add(10 * syncInterval)
	for {
		next := sched.Now().Add(chaosQuiesceStep)
		if next.After(capTime) {
			next = capTime
		}
		sched.RunUntil(next)
		if chaosQuiescent(cluster, next) {
			return true
		}
		if !next.Before(capTime) {
			return false
		}
	}
}

// scnDiscover drives mutual discovery under a lossy policy: the
// fault-free quiesceDiscovery's 100ms cap cannot hold when hello
// exchanges drop, but hello rounds retry every hello_interval, so
// discovery over a lossy network heals on a later round — drive up to
// capLen for it.
func scnDiscover(t *testing.T, sched *simnet.Scheduler, cluster *simnet.Cluster, capLen time.Duration) {
	t.Helper()
	capTime := sched.Now().Add(capLen)
	for !simDiscovered(cluster) {
		next := sched.Now().Add(chaosQuiesceStep)
		if !next.Before(capTime) {
			t.Fatalf("mutual discovery did not complete within %v of virtual time under the lossy policy", capLen)
		}
		sched.RunUntil(next)
	}
}

// scnAssertConverged fails t unless every node's store deep-equals model —
// the pointwise-supremum assertion (equality against the model IS the
// deterministic-winner assertion: the model joins every logged write by the
// LWW order, so a store equal to it holds exactly the winner per key).
func scnAssertConverged(t *testing.T, cluster *simnet.Cluster, model *storage.Store, seed int64, test string) {
	t.Helper()
	for i := range cluster.Nodes() {
		if !recordMapsEqual(cluster.Store(i).Records(), model.Records()) {
			t.Fatalf("node %d holds %d records, the pointwise-supremum model holds %d — divergence after quiescence %s",
				i, len(cluster.Store(i).Records()), len(model.Records()), scnReplay(seed, test))
		}
	}
}

// TestScenario_clockJumps proves TEST-04's live wall-clock-jump family: a
// node's clock jumps forward by hours mid-run and then backward below its
// peers, writes flow from every node at every phase, and the cluster still
// converges to the pointwise supremum with the jumping node's post-jump
// writes strictly dominating its pre-jump versions — the HLC's l/counter
// carry absorbs both jump directions, so version monotonicity never
// depends on wall-clock synchronization. The second half restarts node 0
// in place with a different clock offset (the replacement-node shape: new
// run id, empty store) and proves a restart-time jump re-converges the
// same way.
func TestScenario_clockJumps(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "clockJumps"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)

	// Scenario draws, fixed before any event runs: the jump magnitudes and
	// the per-phase payload stamps.
	fwdJump := 2*time.Hour + time.Duration(rng.IntN(60))*time.Minute
	backJump := -(time.Duration(10+rng.IntN(50)) * time.Minute)
	preTag, fwdTag, backTag := rng.IntN(1000), rng.IntN(1000), rng.IntN(1000)

	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{}, simFixedDelay(nil))
	quiesceDiscovery(t, sched, cluster)

	// Phase 1 — pre-jump writes from every node; node 0's baseline record.
	pre := cluster.Write(0, "jump/monotonic", []byte(fmt.Sprintf("pre-%d", preTag)))
	sched.RunUntil(sched.Now().Add(10 * time.Millisecond))
	cluster.Write(1, "jump/other/1", []byte("first from node 1"))
	cluster.Write(2, "jump/other/2", []byte("first from node 2"))
	sched.RunUntil(sched.Now().Add(10 * time.Millisecond))

	// Phase 2 — node 0's clock jumps forward by hours: its next write
	// stamps hours ahead and strictly dominates the pre-jump version.
	cluster.SetClockOffset(0, fwdJump)
	fwd := cluster.Write(0, "jump/monotonic", []byte(fmt.Sprintf("post-forward-%d", fwdTag)))
	if fwd.Version.Compare(pre.Version) <= 0 {
		t.Fatalf("the post-forward-jump version %x does not strictly dominate the pre-jump %x — monotonicity broke on the forward jump %s",
			fwd.Version[:], pre.Version[:], scnReplay(seed, test))
	}
	cluster.Write(1, "jump/other/1", []byte("second from node 1"))
	sched.RunUntil(sched.Now().Add(10 * time.Millisecond))

	// Phase 3 — node 0's clock jumps backward below both peers: the wall
	// reading falls behind, but the clock's l stays at the forward-jump
	// watermark and the counter carries — the next write still strictly
	// dominates the forward-jump version.
	cluster.SetClockOffset(0, backJump)
	back := cluster.Write(0, "jump/monotonic", []byte(fmt.Sprintf("post-backward-%d", backTag)))
	if back.Version.Compare(fwd.Version) <= 0 {
		t.Fatalf("the post-backward-jump version %x does not strictly dominate the post-forward %x — the counter carry broke on the backward jump %s",
			back.Version[:], fwd.Version[:], scnReplay(seed, test))
	}
	cluster.Write(2, "jump/other/2", []byte("second from node 2"))

	// Convergence: quiescence on the three seams, then the pointwise
	// supremum over every logged write.
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not quiesce after the jump phases %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, chaosModelStore(cluster.WriteLog()), seed, test)

	// Second half — the restart-time jump: node 0 restarts in place (same
	// node id and handle, fresh run id, empty store) with its clock two
	// hours behind the cluster; writes from every node afterwards still
	// converge to the supremum — a restart under clock skew is just another
	// divergence the merge heals.
	cluster.ReplaceNode(0, scnRunID(1, 9), -2*time.Hour)
	sched.RunUntil(sched.Now().Add(time.Millisecond)) // fire the replacement's startup pump
	cluster.Write(1, "jump/post-restart/1", []byte("after the restart"))
	cluster.Write(0, "jump/post-restart/0", []byte("written by the restarted node, two hours behind"))
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not re-converge after the restart-time jump %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, chaosModelStore(cluster.WriteLog()), seed, test)
	wantRun := scnRunID(1, 9)
	if got := cluster.Core(1).PeerSnapshot()[1].RunID; got != wantRun {
		t.Fatalf("node 1's peer row for the restarted node carries run %x, want the replacement's %x %s",
			got[:], wantRun[:], scnReplay(seed, test))
	}
}

// TestScenario_farAhead proves TEST-04's far-ahead-clock scenario: one
// node's clock sits 24 hours ahead for its whole run, everyone writes, and
// (a) the cluster converges to the pointwise supremum, (b) where writes to
// the same key are concurrent the far-ahead node's version wins by version
// order — not by arrival order — and (c) the peers left behind count drift
// warnings on the applies after absorbing the far-ahead stamps (the 06-03
// counter; drift beyond the store's drift-warn gate), while the far-ahead
// node itself counts none: its own wall matches its stamps.
func TestScenario_farAhead(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "farAhead"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)
	farTag := rng.IntN(1000)

	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{}, simFixedDelay(nil))
	cluster.SetClockOffset(1, 24*time.Hour) // before any event: node 1's whole run is a day ahead
	quiesceDiscovery(t, sched, cluster)

	// The concurrent same-key pair, written at the same virtual instant so
	// neither side has absorbed the other's stamp: the far-ahead write
	// carries l = now+24h and must dominate the peer's l = now.
	farShared := cluster.Write(1, "far/shared", []byte(fmt.Sprintf("from-the-future-%d", farTag)))
	localShared := cluster.Write(0, "far/shared", []byte("written now, 24h behind"))
	if farShared.Version.Compare(localShared.Version) <= 0 {
		t.Fatalf("the far-ahead write %x does not dominate the concurrent same-instant write %x %s",
			farShared.Version[:], localShared.Version[:], scnReplay(seed, test))
	}
	cluster.Write(1, "far/only/1", []byte("far-ahead 1"))
	cluster.Write(1, "far/only/2", []byte("far-ahead 2"))

	// A write from node 2 after the far-ahead stamps have been absorbed
	// everywhere: its delivery is the post-absorption apply that observes
	// the absorbed l a warning window past the local wall — the drift
	// counter's pinned absorb-then-warn sequence (TestMergeStats_counts).
	sched.RunUntil(sched.Now().Add(10 * time.Millisecond))
	cluster.Write(2, "far/after", []byte("after the absorption"))

	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not quiesce under the far-ahead skew %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, chaosModelStore(cluster.WriteLog()), seed, test)

	// The concurrent key resolves to the far-ahead write on a third party:
	// the deterministic winner by version order.
	served, err := cluster.Store(2).Get("far/shared")
	if err != nil {
		t.Fatalf("node 2 does not serve the concurrent key after convergence: %v", err)
	}
	if !recordsEqual(served, farShared) {
		t.Fatalf("node 2 serves %+v under the concurrent key, want the byte-equal far-ahead record %+v %s",
			served, farShared, scnReplay(seed, test))
	}

	// Drift observability: the behind peers counted warnings on their
	// post-absorption applies; the far-ahead node counted none.
	for _, i := range []int{0, 2} {
		if got := cluster.Core(i).MergeStats().DriftWarnings; got == 0 {
			t.Fatalf("node %d counted %d drift warnings under a 24h peer skew, want >= 1 %s", i, got, scnReplay(seed, test))
		}
	}
	if got := cluster.Core(1).MergeStats().DriftWarnings; got != 0 {
		t.Fatalf("the far-ahead node counted %d drift warnings, want 0 — its own wall matches its stamps %s", got, scnReplay(seed, test))
	}
}

// TestScenario_partitionSeconds proves TEST-04's short-partition scenario:
// a full bidirectional partition of a few virtual seconds (shorter than
// peer_timeout, so peer tables survive and the heal rides retransmission
// plus the sync exchange), both sides serving writes to distinct keys
// concurrently. At the heal instant the stores demonstrably diverge
// (teeth), and by the documented heal bound — 2 × sync_interval plus the
// response term for replaying the partition's keys, the boundPartitionHeal
// arithmetic, no re-derived numbers — every node holds the pointwise
// supremum with every write from both sides present.
func TestScenario_partitionSeconds(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "partitionSeconds"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)

	// Scenario draws, fixed before any event runs: the partition length
	// (kept under peer_timeout so this stays the retransmit-heal shape) and
	// the payload stamps.
	partLen := 2*time.Second + time.Duration(rng.IntN(1000))*time.Millisecond
	tagA, tagB := rng.IntN(1000), rng.IntN(1000)

	var partStart, partEnd time.Time
	policy := simFixedDelay(func(from, to string, now time.Time) bool {
		pair := (from == "n1" && to == "n2") || (from == "n2" && to == "n1")
		return pair && !now.Before(partStart) && now.Before(partEnd)
	})
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, policy)
	quiesceDiscovery(t, sched, cluster)

	// The partition, with both sides writing distinct keys inside it.
	partStart, partEnd = sched.Now(), sched.Now().Add(partLen)
	writeAt := func(off time.Duration, node int, key string, data []byte) storage.Record {
		t.Helper()
		sched.RunUntil(partStart.Add(off))
		return cluster.Write(node, key, data)
	}
	writeA := writeAt(500*time.Millisecond, 0, "part/sec/a", []byte(fmt.Sprintf("from-a-%d", tagA)))
	writeB := writeAt(1000*time.Millisecond, 1, "part/sec/b", []byte(fmt.Sprintf("from-b-%d", tagB)))
	writeC := writeAt(1500*time.Millisecond, 0, "part/sec/c", []byte(fmt.Sprintf("from-a-again-%d", tagA)))
	writeD := writeAt(2000*time.Millisecond, 1, "part/sec/d", []byte(fmt.Sprintf("from-b-again-%d", tagB)))

	// The heal instant: the divergence was real (teeth for the bound).
	sched.RunUntil(partEnd)
	if recordMapsEqual(cluster.Store(0).Records(), cluster.Store(1).Records()) {
		t.Fatalf("the stores already agree at the heal instant — the divergence probe is vacuous %s", scnReplay(seed, test))
	}

	// The documented heal bound, inclusive: every node converges to the
	// pointwise supremum of all four writes by partEnd + 2*sync_interval +
	// (K-1)*spacing with K = 4 distinct keys.
	sched.RunUntil(partEnd.Add(boundPartitionHeal(4)))
	model := chaosModelStore(cluster.WriteLog())
	for i := range cluster.Nodes() {
		if !recordMapsEqual(cluster.Store(i).Records(), model.Records()) {
			t.Fatalf("node %d has not converged to the supremum by the documented heal bound %v %s",
				i, boundPartitionHeal(4), scnReplay(seed, test))
		}
	}
	for _, w := range []struct {
		key string
		rec storage.Record
	}{
		{"part/sec/a", writeA},
		{"part/sec/b", writeB},
		{"part/sec/c", writeC},
		{"part/sec/d", writeD},
	} {
		for i := range cluster.Nodes() {
			served, err := cluster.Store(i).Get(w.key)
			if err != nil || !recordsEqual(served, w.rec) {
				t.Fatalf("node %d serves (%+v, %v) under %q after the heal bound, want the byte-equal written record %+v %s",
					i, served, err, w.key, w.rec, scnReplay(seed, test))
			}
		}
	}
}

// TestScenario_partitionDays proves TEST-04's multi-day partition in
// virtual time: both sides write the SAME keys concurrently for a
// days-long split — far past peer_timeout, so the heal rides rediscovery
// and the discovery flush — and the cluster converges to the deterministic
// pointwise-supremum winner per key with the MergeStats conflicts counter
// covering the displaced versions (D-10: the winner assertion plus the
// counter IS the evidence — no journal, none anywhere).
func TestScenario_partitionDays(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "partitionDays"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)

	// Scenario draws: the partition length in whole virtual days and the
	// payload stamps.
	partLen := time.Duration(2+rng.IntN(2)) * 24 * time.Hour
	tagA, tagB := rng.IntN(1000), rng.IntN(1000)
	const sharedKeys = 4

	// Virtual-time economics: at the defaults, a multi-day run costs ~20s
	// of wall time in periodic events alone — the 1s hello cadence is
	// linear in virtual time, and the 100ms retransmit-scan granularity
	// fires one no-op Advance per step for every entry retained in
	// suspension across the whole split. The days scenario therefore scales
	// ONLY the periodic cadence — hello_interval and the scan granularity
	// to one sync_interval, peer_timeout to three hello rounds (the
	// documented 5s/1s ratio kept) — while every semantic this scenario
	// asserts is cadence-independent: same-key divergence, the deterministic
	// winner, conflict counting, and the rediscovery-and-flush heal (the
	// flush at rediscovery is scan-independent; suspension makes every scan
	// inside the split a no-op by construction). The peer table still
	// evicts early inside the split (peer_timeout of minutes is still
	// nothing against days), exactly the default-shaped heal this test
	// exists to cover.
	daysCfg := repl.Config{
		HelloInterval:      syncInterval,
		PeerTimeout:        3 * syncInterval,
		RetransmitInterval: syncInterval,
	}
	var partStart, partEnd time.Time
	policy := simFixedDelay(func(from, to string, now time.Time) bool {
		pair := (from == "n1" && to == "n2") || (from == "n2" && to == "n1")
		return pair && !now.Before(partStart) && now.Before(partEnd)
	})
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 2, daysCfg, policy)
	quiesceDiscovery(t, sched, cluster)

	// The days-long split: node 0 writes every shared key first, node 1
	// writes every shared key later in virtual time — same clock offset, so
	// node 1's stamps strictly dominate and the winner per key is
	// deterministic by version order, independent of which side heals
	// first.
	partStart, partEnd = sched.Now(), sched.Now().Add(partLen)
	fromA := make([]storage.Record, sharedKeys)
	for i := range sharedKeys {
		sched.RunUntil(partStart.Add(time.Duration(100+i*100) * time.Millisecond))
		fromA[i] = cluster.Write(0, fmt.Sprintf("part/days/k%d", i), []byte(fmt.Sprintf("from-a-%d-%d", tagA, i)))
	}
	fromB := make([]storage.Record, sharedKeys)
	for i := range sharedKeys {
		sched.RunUntil(partStart.Add(time.Duration(1000+i*100) * time.Millisecond))
		fromB[i] = cluster.Write(1, fmt.Sprintf("part/days/k%d", i), []byte(fmt.Sprintf("from-b-%d-%d", tagB, i)))
	}

	// The heal instant: the same-key divergence was real (teeth).
	sched.RunUntil(partEnd)
	if recordMapsEqual(cluster.Store(0).Records(), cluster.Store(1).Records()) {
		t.Fatalf("the stores already agree at the heal instant — the same-key divergence probe is vacuous %s", scnReplay(seed, test))
	}

	// Heal and settle: rediscovery plus the flush plus the ladder all fit
	// far inside the quiescence cap; convergence is asserted against the
	// model — store equality IS the deterministic-winner-per-key assertion.
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not quiesce after the days-long partition healed %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, chaosModelStore(cluster.WriteLog()), seed, test)

	// The deterministic winner per shared key: node 1's later-stamped write
	// is what every node serves, byte-equal.
	for i := range sharedKeys {
		key := fmt.Sprintf("part/days/k%d", i)
		for node := range cluster.Nodes() {
			served, err := cluster.Store(node).Get(key)
			if err != nil || !recordsEqual(served, fromB[i]) {
				t.Fatalf("node %d serves (%+v, %v) under %q, want the byte-equal later-stamped winner %+v %s",
					node, served, err, key, fromB[i], scnReplay(seed, test))
			}
		}
	}

	// D-10 evidence: the conflicts counter covers the displaced versions.
	// Every shared key's loser is reported exactly once as a displacement
	// on its own node and once as a rejection on the winner's node (Merge's
	// Conflict contract); CORE-07 re-reports can only add. Counted only —
	// no journal, no per-conflict events.
	var conflicts uint64
	for i := range cluster.Nodes() {
		conflicts += cluster.Core(i).MergeStats().Conflicts
	}
	if conflicts < 2*sharedKeys {
		t.Fatalf("the cluster counted %d merge conflicts for %d displaced same-key versions, want >= %d %s",
			conflicts, sharedKeys, 2*sharedKeys, scnReplay(seed, test))
	}
}

// TestScenario_oneWay proves TEST-04's one-way-connectivity scenario: a
// directional partition blocks node 0 → node 1 only, so writes flow one
// way — the receiver converges to the union of both sides' writes while
// the far side holds only its own, and after the block lifts both reach
// the pointwise supremum.
func TestScenario_oneWay(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "oneWay"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)

	// Scenario draws: the block length (under peer_timeout — connectivity
	// asymmetry, not eviction) and the payload stamps.
	blockLen := 2500*time.Millisecond + time.Duration(rng.IntN(500))*time.Millisecond
	tagA, tagB := rng.IntN(1000), rng.IntN(1000)

	var blockStart, blockEnd time.Time
	policy := simFixedDelay(func(from, to string, now time.Time) bool {
		return from == "n1" && to == "n2" && !now.Before(blockStart) && now.Before(blockEnd)
	})
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, policy)
	quiesceDiscovery(t, sched, cluster)

	// Both sides write inside the block; only node 1's write crosses.
	blockStart, blockEnd = sched.Now(), sched.Now().Add(blockLen)
	sched.RunUntil(blockStart.Add(200 * time.Millisecond))
	writeB := cluster.Write(1, "oneway/from-b", []byte(fmt.Sprintf("from-b-%d", tagB)))
	sched.RunUntil(blockStart.Add(400 * time.Millisecond))
	writeA := cluster.Write(0, "oneway/from-a", []byte(fmt.Sprintf("from-a-%d", tagA)))

	// Mid-block: the receiver holds the union of both writes; the far side
	// holds only its own — the directed block has teeth.
	sched.RunUntil(blockStart.Add(1 * time.Second))
	served, err := cluster.Store(0).Get("oneway/from-b")
	if err != nil || !recordsEqual(served, writeB) {
		t.Fatalf("the receiver serves (%+v, %v) for the crossing write mid-block, want the byte-equal record %+v %s",
			served, err, writeB, scnReplay(seed, test))
	}
	served, err = cluster.Store(0).Get("oneway/from-a")
	if err != nil || !recordsEqual(served, writeA) {
		t.Fatalf("the receiver serves (%+v, %v) for its own write mid-block, want the byte-equal record %+v %s",
			served, err, writeA, scnReplay(seed, test))
	}
	_, err = cluster.Store(1).Get("oneway/from-a")
	if err == nil {
		t.Fatalf("the far side already serves the blocked write mid-block — the directional block is vacuous %s", scnReplay(seed, test))
	}

	// Lift the block: both endpoints reach the pointwise supremum.
	sched.RunUntil(blockEnd)
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not quiesce after the one-way block lifted %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, chaosModelStore(cluster.WriteLog()), seed, test)
}

// scnLossWrite is one scripted write of the loss scenario: the virtual
// offset from the run base, the writing node, the key, the payload and —
// for the windowed writes — the directed loss episode over
// [off, off+1s) that drops the initial send plus the +200/+600ms ladder
// attempts and delivers the +1400ms one (the REPL-10 window-partition
// reading; Ack traffic rides the same directed window, so the sender's
// confirms drop with the datagrams).
type scnLossWrite struct {
	off      time.Duration
	node     int
	key      string
	data     string
	windowed bool
}

// scnLossScript is the fixed write script both phases of the loss scenario
// run: four windowed unique-key writes split across both directions, plus
// one same-key pair written by both nodes — the early one clean (past the
// n1→n2 window tiling, so it heals at one RTT), the later one windowed on
// n2→n1 (its ladder delivers +1400ms), so the later stamp dominates by
// version order in every run regardless of delivery order. Sorted by
// offset.
func scnLossScript() []scnLossWrite {
	return []scnLossWrite{
		{100 * time.Millisecond, 0, "loss/a", "loss-a", true},
		{600 * time.Millisecond, 0, "loss/b", "loss-b", true},
		{800 * time.Millisecond, 1, "loss/d", "loss-d", true},
		{1100 * time.Millisecond, 0, "loss/c", "loss-c", true},
		{2200 * time.Millisecond, 0, "loss/shared", "loss-shared-early", false},
		{2600 * time.Millisecond, 1, "loss/shared", "loss-shared-late", true},
	}
}

// TestScenario_loss proves TEST-04's datagram/Ack-loss scenario in two
// phases over one fixed write script. Phase one drives deterministic
// directed loss episodes (time-window partitions dropping the initial send
// plus the +200/+600ms ladder attempts, delivering +1400ms) with writes
// ongoing, asserting per write the documented loss bound — healed within
// ack_timeout_max of the send, inclusively, with mid-window absence teeth
// (boundLossEpisode; the same window drops the returning Acks, so the
// ladder keeps climbing — the REPL-10 reading). Phase two replays the
// identical script on a fresh cluster under the duplicate and reorder
// fault draws (independent per-delivery delays, so duplicates land out of
// order), asserting idempotent convergence to the identical supremum —
// the same winning data per key regardless of delivery order (CORE-06
// observable at cluster scale).
func TestScenario_loss(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "loss"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)
	script := scnLossScript()

	// Phase one — deterministic loss episodes with the documented bound.
	// The windows live behind a pointer the partition closure dereferences
	// at send time: a plain slice variable would be captured by its old
	// header and later appends would be invisible to the network.
	type lossWindow struct {
		from, to   string
		start, end time.Time
	}
	windows := &struct{ list []lossWindow }{}
	schedA := simnet.NewScheduler(at(0))
	clusterA := simnet.NewCluster(schedA, rng, 2, repl.Config{}, simFixedDelay(func(from, to string, now time.Time) bool {
		for _, w := range windows.list {
			if from == w.from && to == w.to && !now.Before(w.start) && now.Before(w.end) {
				return true
			}
		}
		return false
	}))
	quiesceDiscovery(t, schedA, clusterA)
	base := schedA.Now()
	for _, w := range script {
		if w.windowed {
			windows.list = append(windows.list, lossWindow{
				from:  simHandle(w.node),
				to:    simHandle(1 - w.node),
				start: base.Add(w.off),
				end:   base.Add(w.off + time.Second),
			})
		}
	}

	// Drive writes and per-write checks on one merged, monotonic timeline.
	// Check kinds: the windowed UNIQUE keys are asserted absent on the far
	// node at +600ms (teeth — every attempt so far dropped) and served
	// byte-equal at +ack_timeout_max (the inclusive documented bound); the
	// windowed SHARED write's late record is asserted NOT to have landed on
	// the early writer's node at +600ms (it still serves its own early
	// record byte-equal) and to have won by +ack_timeout_max; the clean
	// shared write is served within one RTT.
	const (
		scLossAbsent = iota
		scLossRecord
	)
	type lossCheck struct {
		at     time.Duration
		key    string
		node   int
		kind   int
		recIdx int // for scLossRecord: the script index whose written record must be served
	}
	sharedEarly, sharedLate := 4, 5 // script indexes of the two shared-key writes
	checks := make([]lossCheck, 0, 2*len(script)+1)
	for i, w := range script {
		if !w.windowed {
			checks = append(checks, lossCheck{w.off + boundNormalOneRTT(simOneWay), w.key, 1 - w.node, scLossRecord, i})
			continue
		}
		absent := lossCheck{w.off + 600*time.Millisecond, w.key, 1 - w.node, scLossAbsent, 0}
		if i == sharedLate {
			// The far node wrote the shared key itself: absence is
			// unassertable there — the teeth is that the late record has
			// NOT landed, i.e. the early write is still what it serves.
			absent = lossCheck{w.off + 600*time.Millisecond, w.key, 1 - w.node, scLossRecord, sharedEarly}
		}
		checks = append(checks, absent,
			lossCheck{w.off + boundLossEpisode(ackTimeoutMax), w.key, 1 - w.node, scLossRecord, i})
	}
	slices.SortFunc(checks, func(a, b lossCheck) int { return cmp.Compare(a.at, b.at) })
	written := make([]storage.Record, len(script))
	lossCheckAt := func(c lossCheck) {
		served, err := clusterA.Store(c.node).Get(c.key)
		if c.kind == scLossAbsent {
			if err == nil {
				t.Fatalf("node %d already serves %q at send+600ms — every attempt inside the loss window must be dropped %s",
					c.node, c.key, scnReplay(seed, test))
			}
			return
		}
		if err != nil || !recordsEqual(served, written[c.recIdx]) {
			t.Fatalf("node %d serves (%+v, %v) under %q, want the byte-equal record %+v of script write %d %s",
				c.node, served, err, c.key, written[c.recIdx], c.recIdx, scnReplay(seed, test))
		}
	}
	nextWrite, nextCheck := 0, 0
	for nextWrite < len(script) || nextCheck < len(checks) {
		takeWrite := nextCheck >= len(checks) || (nextWrite < len(script) && script[nextWrite].off <= checks[nextCheck].at)
		if takeWrite {
			w := script[nextWrite]
			schedA.RunUntil(base.Add(w.off))
			written[nextWrite] = clusterA.Write(w.node, w.key, []byte(w.data))
			nextWrite++
			continue
		}
		c := checks[nextCheck]
		nextCheck++
		schedA.RunUntil(base.Add(c.at))
		lossCheckAt(c)
	}
	if ok := scnQuiesce(schedA, clusterA); !ok {
		t.Fatalf("the loss-phase cluster did not quiesce %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, clusterA, chaosModelStore(clusterA.WriteLog()), seed, test)

	// Phase two — the identical script under the duplicate/reorder fault
	// draws: per-delivery delays drawn independently (1..40ms), duplicates
	// at independent delays, drops live. Convergence must be idempotent to
	// all of it: every node reaches a store whose winning data per key is
	// identical to phase one's — delivery order cannot affect the outcome.
	dropProb := 0.15 + rng.Float64()*0.20
	dupProb := 0.15 + rng.Float64()*0.20
	schedB := simnet.NewScheduler(at(0))
	clusterB := simnet.NewCluster(schedB, rng, 2, repl.Config{}, simnet.Policy{
		DropProb: dropProb,
		DupProb:  dupProb,
		Delay: func(rng *rand.Rand) time.Duration {
			return time.Duration(1+rng.IntN(40)) * time.Millisecond
		},
	})
	scnDiscover(t, schedB, clusterB, 30*time.Second)
	baseB := schedB.Now()
	for _, w := range script {
		schedB.RunUntil(baseB.Add(w.off))
		clusterB.Write(w.node, w.key, []byte(w.data))
	}
	if ok := scnQuiesce(schedB, clusterB); !ok {
		t.Fatalf("the chaos-phase cluster did not quiesce %s", scnReplay(seed, test))
	}
	modelA, modelB := chaosModelStore(clusterA.WriteLog()), chaosModelStore(clusterB.WriteLog())
	scnAssertConverged(t, clusterB, modelB, seed, test)
	// The identical-supremum assertion is on the winning DATA per key (and
	// the key set), not on version bytes: a node's clock may absorb a
	// remote stamp before a later local write, so the exact l watermark a
	// stamp carries is delivery-order-dependent — exactly the dependency
	// the join law makes irrelevant to the fixpoint. Same keys, same
	// winning payload per key, every node holding it: that is the
	// order-independent outcome.
	supA, supB := modelA.Records(), modelB.Records()
	if len(supA) != len(supB) {
		t.Fatalf("the two phases' suprema hold %d vs %d keys over the identical write script %s",
			len(supA), len(supB), scnReplay(seed, test))
	}
	for key, ra := range supA {
		rb, ok := supB[key]
		if !ok || !bytes.Equal(ra.Data, rb.Data) || ra.Deleted != rb.Deleted {
			t.Fatalf("the winning data for %q differs across the two phases' suprema (%q/%v vs %q/%v) %s",
				key, ra.Data, ra.Deleted, rb.Data, rb.Deleted, scnReplay(seed, test))
		}
	}
	faults := clusterB.Network().FaultStats()
	if faults.Dropped == 0 || faults.Duplicated == 0 {
		t.Fatalf("the chaos phase observed dropped=%d duplicated=%d, want both live %s",
			faults.Dropped, faults.Duplicated, scnReplay(seed, test))
	}
}
