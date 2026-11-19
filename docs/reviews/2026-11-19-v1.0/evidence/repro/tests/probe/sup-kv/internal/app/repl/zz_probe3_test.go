package repl_test

// AUDIT PROBE (scratch probe copy only): does receiving a DataSync listing
// advance the receiver's HLC (upstream spec: "Update on the max version of
// the chunk")? If not, a write accepted right after the startup gate opens on
// sync arrival can be stamped BELOW a version the peer already listed.

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/repl"
)

func TestZZProbe_gateOpensBeforeClockAbsorbsListing(t *testing.T) {
	// Egress limiter slowed to 1 response/s so the DataRequest reply (the only
	// path that feeds the HLC) is observably later than the listing arrival.
	sched := simnet.NewScheduler(at(0))
	cluster := simnet.NewCluster(sched, simRNG(11), 2, repl.Config{SyncResponseRate: 1}, simFixedDelay(nil))
	cluster.SetClockOffset(1, time.Hour) // node B's wall clock runs one hour ahead
	quiesceDiscovery(t, sched, cluster)
	// B writes two keys; both replicate to A normally.
	cluster.Write(1, "gate/k0", []byte("b-0"))
	recB := cluster.Write(1, "gate/k1", []byte("written by B, clock +1h"))
	if !scnQuiesce(sched, cluster) {
		t.Fatal("no quiescence")
	}
	// A restarts in place with an EMPTY disk and a correct clock.
	cluster.ReplaceNode(0, scnRunID(1, 9), 0)
	tRestart := sched.Now()
	// Drive until A's core reports the first DataSync arrival = the moment the
	// production startup gate opens (grpcapi.RunGate polls SyncArrived).
	for !cluster.Core(0).SyncArrived() {
		sched.RunUntil(sched.Now().Add(time.Millisecond))
		if sched.Now().Sub(tRestart) > 5*time.Second {
			t.Fatal("no DataSync arrived")
		}
	}
	_, errGet := cluster.Store(0).Get("gate/k1")
	// The gate is open: a client write on A for the key B already holds.
	recA := cluster.Write(0, "gate/k1", []byte("written by A after the gate opened"))
	t.Logf("PROBE3 gate opened %v after restart; A holds k1 at that instant: %v; A's new version %x vs B's listed version %x; A's confirmed write dominates: %v",
		sched.Now().Sub(tRestart), errGet == nil, recA.Version[:8], recB.Version[:8], recA.Version.Compare(recB.Version) > 0)
	if !scnQuiesce(sched, cluster) {
		t.Fatal("no quiescence after the write")
	}
	sched.RunUntil(sched.Now().Add(2 * syncInterval))
	final, _ := cluster.Store(0).Get("gate/k1")
	t.Logf("PROBE3 final value on A: %q (A's post-gate write survived: %v)", final.Data, string(final.Data) == "written by A after the gate opened")
}
