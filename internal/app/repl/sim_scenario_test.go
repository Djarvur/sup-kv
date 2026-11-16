package repl_test

import (
	"fmt"
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
// no unacked entries, no queued responses — chaosQuiescent) or capLen
// passes; it reports whether quiescence was observed, leaving the
// scheduler's clock AT the quiescence instant when it was (the caller
// measures elapsed virtual time off sched.Now()).
func scnQuiesce(sched *simnet.Scheduler, cluster *simnet.Cluster, step, capLen time.Duration) bool {
	capTime := sched.Now().Add(capLen)
	for {
		next := sched.Now().Add(step)
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
	if ok := scnQuiesce(sched, cluster, chaosQuiesceStep, 10*syncInterval); !ok {
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
	if ok := scnQuiesce(sched, cluster, chaosQuiesceStep, 10*syncInterval); !ok {
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

	if ok := scnQuiesce(sched, cluster, chaosQuiesceStep, 10*syncInterval); !ok {
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
