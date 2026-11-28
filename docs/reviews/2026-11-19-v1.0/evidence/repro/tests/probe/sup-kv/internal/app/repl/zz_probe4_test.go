package repl_test

// AUDIT PROBE (scratch probe copy only): the chaos scenario with DELETES and
// per-node clock skew mixed in — two fault dimensions the shipped chaos suite
// never draws (simnet.Cluster.Write only ever calls Set; offsets stay zero).
// Convergence is checked against the pointwise-supremum model after driving
// well past the periodic sync rounds (not at the first quiescent instant).

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/port"
)

func TestZZProbe_chaosWithDeletesAndSkew(t *testing.T) {
	rawN := os.Getenv("SUPKV_PROBE_N")
	if rawN == "" {
		t.Skip("probe only")
	}
	n, _ := strconv.Atoi(rawN)
	bad, deletes, tombsInModel := 0, 0, 0
	for seed := int64(1); seed <= int64(n); seed++ {
		rng := simRNG(seed)
		sc := drawChaosScenario(rng)
		sched := simnet.NewScheduler(at(0))
		cluster := simnet.NewCluster(sched, rng, sc.nodes, repl.Config{}, simnet.Policy{
			DropProb: sc.dropProb, DupProb: sc.dupProb, Delay: chaosDelay, Partition: sc.blocked,
		})
		// per-node clock skew: -30s .. +30s, one node occasionally hours ahead
		skewRng := simRNG(seed ^ 0x5eed)
		for i := range sc.nodes {
			off := time.Duration(skewRng.IntN(60_000)-30_000) * time.Millisecond
			if skewRng.IntN(10) == 0 {
				off = time.Duration(1+skewRng.IntN(5)) * time.Hour
			}
			cluster.SetClockOffset(i, off)
		}
		model := chaosModelStore(nil)
		sched.RunUntil(at(50 * time.Millisecond))
		for k, w := range sc.writes {
			sched.RunUntil(at(w.off))
			if k%3 == 2 { // every third op is a delete attempt on the same node
				if _, _, err := cluster.Store(w.node).Delete(w.key); err == nil {
					rec := cluster.Store(w.node).Records()[w.key]
					var out port.Outbox
					cluster.Core(w.node).NotifyLocalWrite(sched.Now(), w.key, rec, &out)
					for _, ob := range out.Drain() {
						if ob.Broadcast {
							cluster.Network().Broadcast(cluster.Handle(w.node), ob.Msg)
						} else {
							cluster.Network().Send(cluster.Handle(w.node), ob.To, ob.Msg)
						}
					}
					_, _, _ = model.Merge(w.key, rec)
					deletes++
					continue
				}
			}
			rec := cluster.Write(w.node, w.key, w.data)
			_, _, _ = model.Merge(w.key, rec)
		}
		last := sc.writes[len(sc.writes)-1].off
		sched.RunUntil(at(last + 20*syncInterval))
		ok := true
		for i := range cluster.Nodes() {
			if !recordMapsEqual(cluster.Store(i).Records(), model.Records()) {
				ok = false
			}
		}
		for _, r := range model.Records() {
			if r.Deleted {
				tombsInModel++
			}
		}
		if !ok {
			bad++
			if bad <= 5 {
				t.Logf("PROBE4 seed %d: NOT converged 20*sync_interval after the last write (nodes=%d)", seed, sc.nodes)
			}
		}
	}
	t.Logf("PROBE4 seeds=%d deletes_issued=%d tombstones_in_final_models=%d not_converged=%d", n, deletes, tombsInModel, bad)
}
