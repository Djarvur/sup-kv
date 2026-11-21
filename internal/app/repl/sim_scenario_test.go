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
	"github.com/Djarvur/sup-kv/internal/port"
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
// irrelevant; the prohibitions of plan 06-05). Plan 02-13 adds TEST-09's
// combined-failure and long-partition scenarios here: the documented
// convergence bounds carry the D-06 no-fresh-loss qualifier, and the
// long-partition pair straddles the REPL-06 eviction boundary.

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
	// The scenario's intent is HLC absorption of a live jump, not
	// rejection: the forward jump is up to ~3 h, so the inbound version
	// bound is widened to 4 h explicitly. The production 60 s default is
	// not widened for this test's convenience anywhere else.
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{MaxClockLead: 4 * time.Hour}, simFixedDelay(nil))
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
	// The scenario's intent is HLC absorption of a 24 h far-ahead clock,
	// not rejection: the inbound version bound is widened to 25 h (the
	// offset plus margin) explicitly. The production 60 s default stays
	// untouched.
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{MaxClockLead: 25 * time.Hour}, simFixedDelay(nil))
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
	// A displaced version is reported as a displacement on the side that
	// applied the winner (Merge's Conflict contract). Under the paced
	// discovery flush the winner can land before the loser's flush send
	// leaves — the loser is then superseded out of tracking and never
	// crosses the wire — so the rejection half on the winner's node is not
	// guaranteed; the counted-at-least-once half always is. Counted only —
	// no journal, no per-conflict events.
	var conflicts uint64
	for i := range cluster.Nodes() {
		conflicts += cluster.Core(i).MergeStats().Conflicts
	}
	if conflicts < sharedKeys {
		t.Fatalf("the cluster counted %d merge conflicts for %d displaced same-key versions, want >= %d %s",
			conflicts, sharedKeys, sharedKeys, scnReplay(seed, test))
	}
}

// scnWritten is one recorded write the no-fresh-loss qualifier tracks: the
// key and the exact record the author installed.
type scnWritten struct {
	key string
	rec storage.Record
}

// scnConvergeBy drives virtual time from the scheduler's current instant in
// chaosQuiesceStep increments until every node's store deep-equals model —
// the pointwise supremum (chaosConverged) — or capTime passes, returning the
// instant convergence was observed (the zero time and false when it never
// was).
func scnConvergeBy(
	sched *simnet.Scheduler,
	cluster *simnet.Cluster,
	model *storage.Store,
	capTime time.Time,
) (time.Time, bool) {
	for {
		next := sched.Now().Add(chaosQuiesceStep)
		if next.After(capTime) {
			next = capTime
		}
		sched.RunUntil(next)
		if chaosConverged(cluster, model) {
			return next, true
		}
		if !next.Before(capTime) {
			return time.Time{}, false
		}
	}
}

// scnAssertRecordsPresent fails t unless every recorded write is served
// byte-equal by every node — the D-06 no-fresh-loss qualifier's executable
// form: a record written on a live node before the heal instant is present
// in the converged state (no fresh loss).
func scnAssertRecordsPresent(t *testing.T, cluster *simnet.Cluster, writes []scnWritten, seed int64, test string) {
	t.Helper()
	for _, w := range writes {
		for i := range cluster.Nodes() {
			served, err := cluster.Store(i).Get(w.key)
			if err != nil || !recordsEqual(served, w.rec) {
				t.Fatalf("node %d serves (%+v, %v) under %q after the bound, want the byte-equal written record %+v (no fresh loss) %s",
					i, served, err, w.key, w.rec, scnReplay(seed, test))
			}
		}
	}
}

// scnHelloInterval is the design-default hello cadence (docs/IDEA.md
// §Parameters), restated here because the long-partition subtests straddle
// the REPL-06 eviction deadline (lastHello + peer_timeout) in whole hello
// rounds: a returning peer's first unblocked Hello can only arrive on the
// hello cadence, so the partition-length boundary is quantized by it.
const scnHelloInterval = time.Second

// TestScenario_longPartition proves TEST-09's long-partition bound (the
// review's T-09 remedy) on three nodes: after a partition longer than
// peer_timeout, convergence is NOT BEFORE the first post-heal sync and NOT
// LATER than the documented partition-heal bound (boundPartitionHeal) — the
// D-06 no-fresh-loss qualifier holding because no write follows the heal, so
// every record written on a live node before the heal instant is present in
// the converged state. Two named subtests straddle the REPL-06 eviction
// boundary. The eviction rule is at-or-after lastHello+peer_timeout
// (TestPeerTable_lifecycle, probe #18), so a partition ending exactly at the
// deadline already evicts: the peer-table-survives side ends one hello round
// before the deadline (the repeating ladder heals it without a sync round)
// and the evicted side runs one hello round past it (only the first
// post-heal sync round replays the partition's writes). The heal instant
// itself must show divergence — the partition was real.
func TestScenario_longPartition(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "longPartition"
	t.Logf("%s", scnReplay(seed, test))

	for _, tc := range []struct {
		name     string
		partLen  time.Duration
		writesAt []time.Duration
		evicted  bool
	}{
		{
			// Ends one hello round before lastHello+peer_timeout: the peer
			// tables survive, so the ladder alone heals the split.
			name:     "peerTableSurvives",
			partLen:  peerTimeout - 2*scnHelloInterval,
			writesAt: []time.Duration{time.Second, 1500 * time.Millisecond, 2 * time.Second},
		},
		{
			// Past the eviction deadline: the peers are gone, the partition's
			// records drain from the ladders, and only the first post-heal
			// sync round carries them back.
			name:     "peerEvicted",
			partLen:  peerTimeout + scnHelloInterval,
			writesAt: []time.Duration{5200 * time.Millisecond, 5600 * time.Millisecond, 5900 * time.Millisecond},
			evicted:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tag := simRNG(seed).IntN(1000)
			var partStart, partEnd time.Time
			policy := simFixedDelay(func(from, to string, now time.Time) bool {
				isolated := (from == "n3") != (to == "n3")
				return isolated && !now.Before(partStart) && now.Before(partEnd)
			})
			sched := simnet.NewScheduler(at(0))
			cluster := simnet.NewCluster(sched, simRNG(seed), 3, repl.Config{}, policy)
			quiesceDiscovery(t, sched, cluster)

			// One record per node before the partition, fully replicated
			// (quiesced) before it starts, then pin the last Hello: advance to
			// the 2s tick's delivery so every pair's lastHello is 2s+one delay
			// and the eviction deadline sits exactly peer_timeout later.
			writes := make([]scnWritten, 0, 6)
			for i := range 3 {
				key := fmt.Sprintf("longpart/pre/%c", 'a'+i)
				writes = append(writes, scnWritten{
					key: key,
					rec: cluster.Write(i, key, []byte(fmt.Sprintf("pre-%c-%d", 'a'+i, tag))),
				})
			}
			if ok := scnQuiesce(sched, cluster); !ok {
				t.Fatalf("the pre-partition writes did not quiesce %s", scnReplay(seed, test))
			}
			sched.RunUntil(at(2*time.Second + simOneWay))
			partStart = sched.Now()
			partEnd = partStart.Add(tc.partLen)

			// The partition's writes: one per node inside the split.
			for i := range 3 {
				sched.RunUntil(partStart.Add(tc.writesAt[i]))
				key := fmt.Sprintf("longpart/mid/%c", 'a'+i)
				writes = append(writes, scnWritten{
					key: key,
					rec: cluster.Write(i, key, []byte(fmt.Sprintf("mid-%c-%d", 'a'+i, tag))),
				})
			}

			// The heal instant: divergence teeth (the split was real) and the
			// eviction verdict the subtest names.
			sched.RunUntil(partEnd)
			if recordMapsEqual(cluster.Store(0).Records(), cluster.Store(2).Records()) {
				t.Fatalf("the stores already agree at the heal instant — the partition probe is vacuous %s", scnReplay(seed, test))
			}
			wantPeers := 2
			if tc.evicted {
				wantPeers = 1
			}
			if got := len(cluster.Core(0).PeerSnapshot()); got != wantPeers {
				t.Fatalf("node 0's peer table holds %d rows at the heal instant, want %d — the eviction probe disagrees with the declared subtest shape %s",
					got, wantPeers, scnReplay(seed, test))
			}

			// The cluster's sync ticks are at 0, sync_interval, 2×sync_interval
			// from the simulation base; both subtests heal inside the first
			// interval, so the first post-heal sync is the first tick.
			firstPostHealSync := at(syncInterval)
			model := chaosModelStore(cluster.WriteLog())
			bound := partEnd.Add(boundPartitionHeal(len(writes)))
			if tc.evicted {
				// Not before the first post-heal sync: a returning node learns
				// the partition's writes only from a listing, and no listing is
				// emitted between the heal and the tick.
				sched.RunUntil(firstPostHealSync.Add(-time.Nanosecond))
				if chaosConverged(cluster, model) {
					t.Fatalf("the cluster converged before the first post-heal sync — the heal-term lower bound is vacuous %s", scnReplay(seed, test))
				}
			}
			convergedAt, withinBound := scnConvergeBy(sched, cluster, model, bound)
			if !withinBound {
				t.Fatalf("the cluster did not converge within the documented heal bound %v %s",
					boundPartitionHeal(len(writes)), scnReplay(seed, test))
			}
			// The subtest shape decides which side of the first post-heal sync
			// convergence must land on: the evicted peers need the sync's
			// reconciliation term, the surviving peers must heal without it.
			if usedSyncTerm := !convergedAt.Before(firstPostHealSync); usedSyncTerm != tc.evicted {
				t.Fatalf("convergence at +%v against the first post-heal sync (+%v) contradicts the %q subtest shape (evicted=%v) %s",
					convergedAt.Sub(partStart), firstPostHealSync.Sub(partStart), tc.name, tc.evicted, scnReplay(seed, test))
			}
			scnAssertRecordsPresent(t, cluster, writes, seed, test)
			if ok := scnQuiesce(sched, cluster); !ok {
				t.Fatalf("the cluster did not quiesce after the healed partition %s", scnReplay(seed, test))
			}
			scnAssertConverged(t, cluster, model, seed, test)
		})
	}
}

// combinedFailuresMargin is the documented allowance the combined-failure
// bound carries beyond boundPartitionHeal: the D-06 qualifier limits the
// bound to records written before the heal with no fresh loss, and the
// restarted node's reconciliation waits for the hello cadence, so a few
// hello rounds of scheduler-phase slack sit inside the qualifier rather than
// being hidden in a re-derived constant.
const combinedFailuresMargin = 5 * time.Second

// TestScenario_combinedFailures proves TEST-09's combined-failure scenario:
// three nodes under loss, duplication, a timed partition and one
// ReplaceNode restart at once, converging no later than the
// boundPartitionHeal expression for the key count used plus the documented
// margin, with the D-06 no-fresh-loss qualifier — every record written on a
// live node before the heal instant is present in the converged state (all
// fault windows and writes end at the heal; only duplication continues, and
// a duplicate can never lose a record). The heal instant itself must show
// divergence: the partition and the restart were real.
func TestScenario_combinedFailures(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "combinedFailures"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)
	tag := rng.IntN(1000)

	var lossStart, lossEnd, partStart, partEnd time.Time
	policy := simnet.Policy{
		DupProb: 0.25,
		Delay: func(rng *rand.Rand) time.Duration {
			return time.Duration(1+rng.IntN(10)) * time.Millisecond
		},
		Partition: func(from, to string, now time.Time) bool {
			// The loss episode: node 0's datagrams are lost for 600ms (the
			// TestScenario_loss directed-window form, not a network-wide
			// partition). The timed partition isolates node 2 from both peers
			// for 8s — past peer_timeout, so eviction fires on both sides.
			lost := (from == "n1" || to == "n1") && !now.Before(lossStart) && now.Before(lossEnd)
			isolated := (from == "n3") != (to == "n3") && !now.Before(partStart) && now.Before(partEnd)
			return lost || isolated
		},
	}
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{}, policy)
	quiesceDiscovery(t, sched, cluster)

	// Pre-fault writes, one per node, fully replicated before any fault: the
	// record the restart wipes locally must already live on both peers, or
	// the no-fresh-loss qualifier could not hold.
	writes := make([]scnWritten, 0, 8)
	for i := range 3 {
		key := fmt.Sprintf("comb/pre/%c", 'a'+i)
		writes = append(writes, scnWritten{
			key: key,
			rec: cluster.Write(i, key, []byte(fmt.Sprintf("pre-%c-%d", 'a'+i, tag))),
		})
	}
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the pre-fault writes did not quiesce %s", scnReplay(seed, test))
	}

	t0 := sched.Now()
	lossStart, lossEnd = t0, t0.Add(600*time.Millisecond)
	partStart, partEnd = t0.Add(2*time.Second), t0.Add(10*time.Second)

	// The loss episode, with writes on both sides inside it: node 0's
	// broadcast and its first ladder rungs drop, the later rung heals it.
	sched.RunUntil(t0.Add(300 * time.Millisecond))
	writes = append(writes,
		scnWritten{key: "comb/loss/a", rec: cluster.Write(0, "comb/loss/a", []byte(fmt.Sprintf("lost-a-%d", tag)))},
		scnWritten{key: "comb/loss/b", rec: cluster.Write(1, "comb/loss/b", []byte(fmt.Sprintf("lost-b-%d", tag)))},
	)

	// The timed partition, with the restart inside it: the replacement boots
	// empty while isolated, so it heals only through the post-heal listing.
	sched.RunUntil(t0.Add(5 * time.Second))
	newRun := scnRunID(3, 9)
	cluster.ReplaceNode(2, newRun, 0)
	sched.RunUntil(sched.Now().Add(time.Millisecond)) // fire the replacement's startup pump

	// Writes after the eviction deadline (lastHello+peer_timeout): their
	// authors' pending entries drain, so only the post-heal reconciliation
	// can carry them across the split.
	for _, w := range []struct {
		at   time.Duration
		node int
		key  string
	}{
		{7500 * time.Millisecond, 0, "comb/part/a"},
		{8 * time.Second, 1, "comb/part/b"},
		{8500 * time.Millisecond, 2, "comb/part/c"},
	} {
		sched.RunUntil(t0.Add(w.at))
		writes = append(writes, scnWritten{
			key: w.key,
			rec: cluster.Write(w.node, w.key, []byte(fmt.Sprintf("%s-%d", w.key, tag))),
		})
	}

	// The heal instant: divergence teeth — the partition and the restart were
	// not vacuous — and the fault classes the scenario claims actually fired.
	sched.RunUntil(partEnd)
	if recordMapsEqual(cluster.Store(0).Records(), cluster.Store(2).Records()) {
		t.Fatalf("node 0 and the replacement already agree at the heal instant — the combined faults are vacuous %s", scnReplay(seed, test))
	}
	faults := cluster.Network().FaultStats()
	if faults.Partitioned == 0 || faults.Duplicated == 0 {
		t.Fatalf("the combined run observed partitioned=%d duplicated=%d, want both live %s",
			faults.Partitioned, faults.Duplicated, scnReplay(seed, test))
	}

	// The documented bound: boundPartitionHeal over the distinct keys plus
	// the named margin, asserted together with the no-fresh-loss qualifier.
	model := chaosModelStore(cluster.WriteLog())
	bound := partEnd.Add(boundPartitionHeal(len(writes)) + combinedFailuresMargin)
	if _, ok := scnConvergeBy(sched, cluster, model, bound); !ok {
		t.Fatalf("the cluster did not converge within the documented combined-failure bound %v %s",
			boundPartitionHeal(len(writes))+combinedFailuresMargin, scnReplay(seed, test))
	}
	scnAssertRecordsPresent(t, cluster, writes, seed, test)
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not quiesce after the combined failures healed %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, model, seed, test)

	// REPL-07's row update survived the combined faults: the peers' row for
	// the restarted node carries the replacement's run id.
	if got := cluster.Core(0).PeerSnapshot()[3].RunID; got != newRun {
		t.Fatalf("node 0's peer row for the restarted node carries run %x, want the replacement's %x %s",
			got[:], newRun[:], scnReplay(seed, test))
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

// TestScenario_burst proves TEST-04's ~1000 writes/s burst scenario: a
// write every virtual millisecond over a multi-second window — one hot key
// hammered by every node plus a spread of distinct keys — sustains without
// loss and converges to the pointwise supremum within the documented
// bounds once the burst ends. This is the correctness half only: the 06-08
// benchmark measures throughput separately, so no performance is asserted
// here — bounded convergence and zero lost writes are.
func TestScenario_burst(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "burst"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)

	// Scenario draws, fixed before any event runs: the writing node per
	// millisecond. Every tenth write goes to a fresh spread key; the other
	// nine hammer the one hot key — 3000 writes, 301 distinct keys.
	const burstWrites = 3000
	type scnBurstWrite struct {
		off  time.Duration
		node int
		key  string
		data string
	}
	script := make([]scnBurstWrite, 0, burstWrites)
	for i := range burstWrites {
		key := "burst/hot"
		if i%10 == 9 {
			key = fmt.Sprintf("burst/spread/%04d", i/10)
		}
		script = append(script, scnBurstWrite{
			off:  time.Duration(i) * time.Millisecond,
			node: rng.IntN(3),
			key:  key,
			data: fmt.Sprintf("burst-%d", i),
		})
	}
	const burstKeys = 1 + burstWrites/10

	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{}, simFixedDelay(nil))
	quiesceDiscovery(t, sched, cluster)

	t0 := sched.Now()
	for _, w := range script {
		sched.RunUntil(t0.Add(w.off))
		cluster.Write(w.node, w.key, []byte(w.data))
	}
	if got := len(cluster.WriteLog()); got != burstWrites {
		t.Fatalf("the write log captured %d of %d writes %s", got, burstWrites, scnReplay(seed, test))
	}

	// The documented bounds once the burst ends: the largest documented
	// convergence window — the partition-heal bound over the burst's
	// distinct keys — covers the post-burst divergence (every write was
	// broadcast; stragglers ride the ladder far inside it).
	lastOff := script[len(script)-1].off
	sched.RunUntil(t0.Add(lastOff + boundPartitionHeal(burstKeys)))
	model := chaosModelStore(cluster.WriteLog())
	for i := range cluster.Nodes() {
		if !recordMapsEqual(cluster.Store(i).Records(), model.Records()) {
			t.Fatalf("node %d has not converged to the supremum by %v after the burst ended %s",
				i, boundPartitionHeal(burstKeys), scnReplay(seed, test))
		}
	}
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not quiesce after the burst %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, model, seed, test)
}

// TestScenario_emptyDisk proves TEST-04's empty-disk recovery bound: a
// converged two-node cluster gains a brand-new member with an empty store
// (fresh node id and handle — a join, never a restart listing), and the
// joiner converges to the full state within N/sync_response_rate plus the
// documented pacing margin. The bound's arithmetic, from the mechanism:
// the joiner learns state only from the peers' periodic DataSync listings
// (a new node id is never targeted), so the fill starts at their next
// sync_interval tick — at most one sync_interval plus the positive-only sync
// jitter (a tenth of the interval) after the join — and then
// rides the egress limiter: both peers' listings (each node's periodic timer
// carries its own deterministic jitter, so the two may fire up to the jitter
// envelope apart) enqueue up to 2N DataRequests on the joiner's single FIFO,
// spaced
// floor(1s/rate) with the first immediate, so the last departs at
// (2N-1)*spacing and its reply lands one spacing later; the measurement
// step is one spacing wide, adding at most one more. Bound:
// sync_interval + sync_interval/10 + 2N*spacing + 2*spacing, all stated
// terms.
func TestScenario_emptyDisk(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "emptyDisk"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)

	const nKeys = 50
	pace := time.Second / time.Duration(simDefaultResponseRate)
	bound := syncInterval + syncInterval/10 + time.Duration(2*nKeys+2)*pace

	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, simFixedDelay(nil))
	quiesceDiscovery(t, sched, cluster)
	for i := range nKeys {
		sched.RunUntil(sched.Now().Add(time.Millisecond))
		cluster.Write(i%2, fmt.Sprintf("emptydisk/key/%02d", i), []byte(fmt.Sprintf("value-%d-%d", i, rng.IntN(1000))))
	}
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the seed cluster did not quiesce before the join %s", scnReplay(seed, test))
	}
	model := chaosModelStore(cluster.WriteLog())

	// The join: fresh node id, derived fresh run id, empty store, clock on
	// cluster time. The measurement starts at the joiner's first Hello.
	tJoin := sched.Now()
	joiner := cluster.AddNode(0)
	if got := cluster.Nodes(); got != 3 {
		t.Fatalf("the cluster holds %d nodes after the join, want 3", got)
	}
	sched.RunUntil(tJoin) // fire the joiner's startup pump at tJoin exactly

	// Drive at the pacing granularity until the recovery COMPLETES — the
	// joiner holding the full state AND the three-seam quiescence (bare
	// quiescence is not recovery: the cluster is also quiescent in the
	// gap before the peers' next periodic listing starts the fill) — or
	// the bound passes; then assert the elapsed virtual time inside the
	// bound.
	capTime := tJoin.Add(bound)
	recovered := false
	for {
		next := sched.Now().Add(pace)
		if !next.Before(capTime) {
			next = capTime
		}
		sched.RunUntil(next)
		if chaosQuiescent(cluster, next) && recordMapsEqual(cluster.Store(joiner).Records(), model.Records()) {
			recovered = true
			break
		}
		if !next.Before(capTime) {
			break
		}
	}
	if !recovered {
		t.Fatalf("the joiner did not recover the full state within the empty-disk bound %v %s", bound, scnReplay(seed, test))
	}
	if elapsed := sched.Now().Sub(tJoin); elapsed > bound {
		t.Fatalf("the empty-disk recovery took %v of virtual time, want <= %v (N/rate plus the documented pacing margin) %s",
			elapsed, bound, scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, model, seed, test)
}

// TestScenario_restart proves TEST-04's restart-via-RunID scenario in
// cluster (REPL-07): a converged node restarts in place — same node id and
// handle, fresh run id, empty store — and (a) the peers detect the run_id
// change and answer with the targeted full listing, so the replacement
// re-converges in milliseconds, far inside one sync_interval (the listing
// is immediate; only a brand-new node id waits for the periodic rounds —
// the empty-disk scenario's path); (b) the replay traffic the listing
// solicits rides the egress limiter on both sides (the replacement's
// DataRequests and the peers' DataReplication replies — the plan-04-04
// reconciliation contract observed live), while the listing itself never
// queues; and (c) an in-flight write from the old run — its broadcast
// stranded by a directed block at send time, so it reached nobody before
// the disk went empty — still merges by version merit when it finally
// arrives under the old run id: run ids gate the restart reaction, never
// the record join.
func TestScenario_restart(t *testing.T) {
	t.Parallel()
	seed := chaosSeed(t)
	const test = "restart"
	t.Logf("%s", scnReplay(seed, test))
	rng := simRNG(seed)

	// The stranded-write block: on while node 1's outbound broadcast is
	// absorbed by the restart, off afterwards.
	strandOn := false
	policy := simFixedDelay(func(from, to string, _ time.Time) bool {
		return strandOn && from == "n2" && (to == "n1" || to == "n3")
	})
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, rng, 3, repl.Config{}, policy)
	quiesceDiscovery(t, sched, cluster)
	for i := range 6 {
		sched.RunUntil(sched.Now().Add(time.Millisecond))
		cluster.Write(i%3, fmt.Sprintf("restart/key/%d", i), []byte(fmt.Sprintf("value-%d", i)))
	}
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the seed cluster did not quiesce before the restart %s", scnReplay(seed, test))
	}

	// The stranded write: broadcast with node 1's outbound blocked, so it
	// reaches nobody (own echo aside) and dies with the old run's disk.
	strandOn = true
	stranded := cluster.Write(1, "restart/stranded", []byte("from the old run"))
	oldRun := scnRunID(2, 2) // node id 2's derived run id, exactly as NewCluster built it
	newRun := scnRunID(2, 9)

	// The restart, in place: the peers' rows for node id 2 still carry
	// oldRun, so the replacement's first Hello fires the targeted listing.
	tReplace := sched.Now()
	strandOn = false
	cluster.ReplaceNode(1, newRun, 0)
	sched.RunUntil(tReplace)

	// Re-convergence is immediate (the targeted listing bypasses the
	// periodic rounds), so drive with a fine step and a cap far under one
	// sync_interval, polling the replacement's egress queue for the
	// limiter-paced replay.
	maxQueued := 0
	capTime := tReplace.Add(2 * time.Second)
	quiesced := false
	for {
		next := sched.Now().Add(5 * time.Millisecond)
		if !next.Before(capTime) {
			next = capTime
		}
		sched.RunUntil(next)
		if q := cluster.Core(1).QueuedResponses(); q > maxQueued {
			maxQueued = q
		}
		if chaosQuiescent(cluster, next) {
			quiesced = true
			break
		}
		if !next.Before(capTime) {
			break
		}
	}
	if !quiesced {
		t.Fatalf("the cluster did not re-converge within one sync_interval of the restart — the targeted listing is immediate %s", scnReplay(seed, test))
	}

	// The stranded write is the one logged write nobody holds: convergence
	// is asserted against the model of everything else.
	model := chaosModelStore(nil)
	for _, w := range cluster.WriteLog() {
		if w.Key != "restart/stranded" {
			_, _, _ = model.Merge(w.Key, w.Rec)
		}
	}
	scnAssertConverged(t, cluster, model, seed, test)

	// REPL-07's row update: exactly one row per node id, carrying the new
	// run id, on a peer that saw the replacement's Hello.
	snap := cluster.Core(0).PeerSnapshot()
	if len(snap) != 2 {
		t.Fatalf("node 0's peer table holds %d rows after the restart, want exactly 2 — a run_id change never forks the table %s", len(snap), scnReplay(seed, test))
	}
	if got := snap[2].RunID; got != newRun {
		t.Fatalf("node 0's row for node id 2 carries run %x after the restart, want the replacement's %x %s",
			got[:], newRun[:], scnReplay(seed, test))
	}

	// The limiter-paced replay, observed live in the cluster: the targeted
	// listing itself never queues (plan-04-04's scope pin, pinned at the
	// seam by TestRestart_runIDChangeTriggersFullSync), but the replay it
	// solicits does — the replacement's DataRequests for the listed keys
	// pile onto its single egress FIFO when both peers' listings land
	// together, so its queue observably holds a backlog while the limiter
	// drains it at the response spacing. The peers' reply queues are
	// structurally invisible to boundary polling by the same contract: the
	// replacement's own limiter spaces its requests one spacing apart, so
	// each reply's egress slot is already due when its request arrives and
	// the peer's queue empties inside the same delivery cascade — the
	// pacing governs the emission times either way, which is what the
	// queue-depth observation of the replacement's side witnesses.
	if maxQueued == 0 {
		t.Fatalf("the replacement's egress queue never held a response — the replay the targeted listing solicits must be limiter-gated %s", scnReplay(seed, test))
	}

	// The stale-runID in-flight write: the stranded record finally arrives
	// at node 0 under the old run's id. It merges by version merit —
	// nobody else ever wrote the key, so it installs — the Ack routes to
	// the replacement as a stale no-op, and the periodic rounds spread the
	// key until the whole cluster holds it.
	var out port.Outbox
	cluster.Core(0).HandleDatagram(sched.Now(), port.InboundDatagram{
		Source:       cluster.Handle(1),
		Status:       port.Decoded,
		ProtoVersion: repl.SupportedProtoVersion,
		NodeID:       2,
		RunID:        oldRun,
		Msg:          port.DataReplication{Key: "restart/stranded", Record: stranded},
	}, &out)
	for _, ob := range out.Drain() {
		if ob.Broadcast {
			cluster.Network().Broadcast(cluster.Handle(0), ob.Msg)
			continue
		}
		cluster.Network().Send(cluster.Handle(0), ob.To, ob.Msg)
	}
	// The spread rides the periodic rounds (node 0 wrote nothing — a
	// remote apply retransmits nothing), so drive until every node holds
	// the model-with-stranded-key, not merely until quiescence: the
	// cluster is quiescent while the key sits on node 0 alone.
	_, _, _ = model.Merge("restart/stranded", stranded)
	capTime = sched.Now().Add(10 * syncInterval)
	converged := false
	for {
		next := sched.Now().Add(chaosQuiesceStep)
		if !next.Before(capTime) {
			next = capTime
		}
		sched.RunUntil(next)
		equal := true
		for i := range cluster.Nodes() {
			equal = equal && recordMapsEqual(cluster.Store(i).Records(), model.Records())
		}
		if equal {
			converged = true
			break
		}
		if !next.Before(capTime) {
			break
		}
	}
	if !converged {
		t.Fatalf("the stranded key did not spread to the whole cluster within the periodic rounds %s", scnReplay(seed, test))
	}
	if ok := scnQuiesce(sched, cluster); !ok {
		t.Fatalf("the cluster did not settle after the stale-run spread %s", scnReplay(seed, test))
	}
	scnAssertConverged(t, cluster, model, seed, test)
	if got := cluster.Core(0).PeerSnapshot()[2].RunID; got != newRun {
		t.Fatalf("node 0's row for node id 2 carries run %x after the stale-run delivery, want the replacement's %x — an old-run datagram must not flip the table %s",
			got[:], newRun[:], scnReplay(seed, test))
	}
}
