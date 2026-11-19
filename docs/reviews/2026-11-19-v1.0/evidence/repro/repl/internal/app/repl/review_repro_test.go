package repl_test

// Review repro tests (scratch copy only). Every test asserts the behaviour
// the accepted spec / the documented contract requires, so a FAIL here is the
// reproduction of the finding named in the test doc.

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// RPL-01: spec "Replication", third outcome — a DataReplication whose
// version is LOWER than the local one must be answered with a unicast
// DataReplication carrying the local record and must NOT be Acked.
func TestReview_staleDataReplicationIsAckedInsteadOfAnswered(t *testing.T) {
	core, st := newTestCore()
	now := at(0)
	var out port.Outbox
	core.Advance(now, &out)
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	out.Drain()

	// Local node holds the NEWER record (l = base+10s).
	newer := remoteRecord(testNowMs+10_000, peerID2, "newer")
	if _, _, err := st.store.Merge(keyAlpha, newer); err != nil {
		t.Fatal(err)
	}
	// Peer 1 broadcasts an OLDER version of the same key.
	older := remoteRecord(testNowMs+1_000, peerID1, "older")
	core.HandleDatagram(now, replFrom(peerID1, keyAlpha, older), &out)
	drained := out.Drain()

	var acks, replies int
	for _, ob := range drained {
		switch m := ob.Msg.(type) {
		case port.Ack:
			acks++
			t.Logf("emitted Ack(%q, %x) to %q", m.Key, m.Version[:6], ob.To)
		case port.DataReplication:
			if !ob.Broadcast && ob.To == handleFor(peerID1) && m.Record.Version == newer.Version {
				replies++
			}
		}
	}
	if acks != 0 || replies != 1 {
		t.Fatalf("stale DataReplication: got %d Ack(s) and %d unicast newer-record replies; spec requires 0 Acks and exactly 1 unicast DataReplication with the local newer record", acks, replies)
	}
}

// RPL-01 consequence: the stale writer's pending entry drains on the Ack of
// a peer that REJECTED the record, so the writer keeps the losing value and
// reports the write as replicated.
func TestReview_staleWriterDrainsOnRejectingPeersAck(t *testing.T) {
	sched := simnet.NewScheduler(time.UnixMilli(testNowMs))
	cluster := simnet.NewCluster(sched, simRNG(1), 2, repl.Config{}, simFixedDelay(nil))
	quiesceDiscovery(t, sched, cluster)

	// Node 1 (index 1) holds a newer record for the key that node 0 never
	// saw and that node 1 no longer tracks as pending (the state after a
	// restart of node 1, a dump-failed write, a snapshot load or an
	// out-of-order notify): install it straight into the store.
	newer := storage.Record{Version: version.PackVersion(sched.Now().UnixMilli()+20_000, 0, 2), Data: []byte("B-newer")}
	if _, _, err := cluster.Store(1).Merge("cfg/x", newer); err != nil {
		t.Fatal(err)
	}
	// Node 0 writes the same key with its (lower) clock.
	rec := cluster.Write(0, "cfg/x", []byte("A-older"))
	if rec.Version.Compare(newer.Version) >= 0 {
		t.Fatal("test setup: node 0 version must be lower")
	}
	sched.RunUntil(sched.Now().Add(10 * time.Millisecond)) // 5 RTTs

	if n := cluster.Core(0).UnackedStats(sched.Now()).Count; n != 0 {
		t.Logf("node 0 still tracks %d pending entries", n)
	} else {
		t.Logf("node 0 pending set drained: the write is reported replicated")
	}
	got, err := cluster.Store(0).Get("cfg/x")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Data) != "B-newer" {
		t.Fatalf("5 RTTs after node 0's stale write: node 0 reads %q, node 1 holds %q; spec bound is one RTT (peer answers the stale record with its newer one), the code leaves it to the next sync_interval round", got.Data, "B-newer")
	}
}

// RPL-02: spec "Hello" — a peer seen for the FIRST time (or returning after
// peer_timeout eviction) must be sent the full listing immediately. The code
// answers only a run_id change of a still-remembered peer.
func TestReview_restartAfterEvictionGetsNoListing(t *testing.T) {
	core, st := newTestCore()
	seedGraduatedKeys(t, st)
	var out port.Outbox
	now := at(0)
	core.Advance(now, &out)
	core.HandleDatagram(now, helloFromRun(runIDFor(1)), &out)
	out.Drain()

	// The peer goes down for longer than peer_timeout (5s): evicted.
	now = at(6 * time.Second)
	core.Advance(now, &out)
	out.Drain()
	if len(core.PeerSnapshot()) != 0 {
		t.Fatal("test setup: peer must be evicted")
	}
	// It comes back as a fresh process (new run_id).
	core.HandleDatagram(now, helloFromRun(runIDFor(2)), &out)
	syncs := targetedSyncs(t, out.Drain(), handleFor(peerID1))
	if len(syncs) == 0 {
		t.Fatalf("a peer restarted after %v of downtime (> peer_timeout) received no targeted DataSync listing; it stays blind to keys it lacks until the next periodic round (up to sync_interval=30s) and its startup gate opens on the 1s backstop without any sync", 6*time.Second)
	}
}

// RPL-03 / API gate: the startup gate's sync half (Core.SyncArrived) flips on
// the first received DataSync chunk, long before the records it announces
// have been fetched. A create-if-absent issued right after the gate opened
// then overwrites the peer's existing user value cluster-wide.
func TestReview_gateOpensBeforeSyncIsProcessed(t *testing.T) {
	sched := simnet.NewScheduler(time.UnixMilli(testNowMs))
	cluster := simnet.NewCluster(sched, simRNG(1), 2, repl.Config{}, simFixedDelay(nil))
	quiesceDiscovery(t, sched, cluster)

	const keys = 300 // 0.6s of DataRequest replies at sync_response_rate=500/s: inside startup_gate=1s
	for i := range keys {
		cluster.Write(1, fmt.Sprintf("cfg/%04d", i), []byte("user-value"))
	}
	if !scnQuiesce(sched, cluster) {
		t.Fatal("setup did not quiesce")
	}
	// Node 0 loses its disk and restarts empty within peer_timeout.
	cluster.ReplaceNode(0, scnRunID(1, 7), 0)
	restartAt := sched.Now()
	for !cluster.Core(0).SyncArrived() {
		sched.RunUntil(sched.Now().Add(time.Millisecond))
		if sched.Now().Sub(restartAt) > time.Second {
			t.Fatal("SyncArrived never flipped")
		}
	}
	// grpcapi.RunGate polls every 20ms: the gate is open at most 20ms later.
	sched.RunUntil(sched.Now().Add(20 * time.Millisecond))
	have := len(cluster.Store(0).Records())
	t.Logf("gate open %v after restart; restarted node holds %d of %d peer records", sched.Now().Sub(restartAt), have, keys)

	lastKey := fmt.Sprintf("cfg/%04d", keys-1)
	_, _, err := cluster.Store(0).SetIfAbsent(lastKey, []byte("build-default"))
	if err == nil {
		// The author sequence of grpcapi.Set: notify after the write.
		rec := cluster.Store(0).Records()[lastKey]
		var out port.Outbox
		cluster.Core(0).NotifyLocalWrite(sched.Now(), lastKey, rec, &out)
		for _, ob := range out.Drain() {
			if ob.Broadcast {
				cluster.Network().Broadcast(cluster.Handle(0), ob.Msg)
			}
		}
	}
	if !scnQuiesce(sched, cluster) {
		t.Fatal("did not quiesce")
	}
	peerVal, _ := cluster.Store(1).Get(lastKey)
	if err == nil || string(peerVal.Data) != "user-value" {
		t.Fatalf("create-if-absent right after the gate opened: err=%v, peer now holds %q for %s — the user's confirmed value was replaced by the build default (gate opened with %d/%d records synced)", err, peerVal.Data, lastKey, have, keys)
	}
}

// RPL-03b: spec "Recovery" — on every received DataSync chunk the node must
// call HLC Update with the chunk's maximum version. The code never does, so
// a write accepted right after the gate opened can be stamped BELOW a version
// the node has already been told about, and is then silently overwritten.
func TestReview_dataSyncDoesNotAdvanceHLC(t *testing.T) {
	core, st := newTestCore() // local wall clock parked at testNowMs
	var out port.Outbox
	now := at(0)
	core.Advance(now, &out)
	out.Drain()

	// The peer's clock runs 30s ahead (inside drift_warn=60s): its listing
	// announces a version 30s in our future.
	peerVer := version.PackVersion(testNowMs+30_000, 0, peerID1)
	core.HandleDatagram(now, syncFrom(peerID1, port.SyncEntry{Key: keyAlpha, Version: peerVer}), &out)
	if !core.SyncArrived() {
		t.Fatal("test setup: gate signal expected")
	}
	// Gate is open: a client write is accepted and (in production) confirmed.
	localVer, _, err := st.store.Set(keyAlpha, []byte("confirmed-after-gate"))
	if err != nil {
		t.Fatal(err)
	}
	if localVer.Compare(peerVer) <= 0 {
		t.Fatalf("write accepted after the first sync chunk got version %x, BELOW the version %x that chunk announced: the confirmed write will be discarded by LWW as soon as the requested record arrives (spec: Update(max version of the chunk) on receipt)", localVer[:8], peerVer[:8])
	}
}

// RPL-04: NotifyLocalWrite replaces the pending entry unconditionally. Two
// concurrent Set RPCs on one key are released by the same dump and race to
// notify; when the older version's notification arrives second it REPLACES
// the newer one, and the newer version is never retransmitted.
func TestReview_outOfOrderNotifyReplacesNewerPending(t *testing.T) {
	core, st := newTestCore()
	var out port.Outbox
	now := at(0)
	core.Advance(now, &out)
	core.HandleDatagram(now, helloFrom(peerID1), &out)
	out.Drain()

	v1, _, _ := st.store.Set(keyAlpha, []byte("first"))
	rec1 := st.store.Records()[keyAlpha]
	v2, _, _ := st.store.Set(keyAlpha, []byte("second"))
	rec2 := st.store.Records()[keyAlpha]
	if v2.Compare(v1) <= 0 {
		t.Fatal("test setup")
	}
	// Handler goroutines notify in the opposite order.
	core.NotifyLocalWrite(now, keyAlpha, rec2, &out)
	core.NotifyLocalWrite(now, keyAlpha, rec1, &out)
	out.Drain()

	// The v2 broadcast is lost; the peer applies v1 and Acks it.
	core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, v1), &out)
	out.Drain()

	// Drive the whole retransmission ladder: v2 must still be offered.
	sawV2 := false
	for off := time.Duration(0); off <= 20*time.Second; off += 100 * time.Millisecond {
		core.Advance(at(off), &out)
		for _, m := range broadcastRepls(out.Drain()) {
			if m.Record.Version == v2 {
				sawV2 = true
			}
		}
		// keep the peer alive
		if off%time.Second == 0 {
			core.HandleDatagram(at(off), helloFrom(peerID1), &out)
			out.Drain()
		}
	}
	if _, tracked := core.PendingSnapshot()[keyAlpha]; !tracked && !sawV2 {
		t.Fatalf("store holds v2 (%x) but tracking was replaced by the older v1 (%x) and drained on v1's Ack: v2 was never retransmitted in 20s — the lost-datagram bound (ack_timeout_max) silently degrades to sync_interval", v2[:8], v1[:8])
	}
}

// RPL-05: a single CRC-valid DataReplication whose version carries the
// 48-bit timestamp ceiling is applied without any sanity bound. It pins the
// HLC of every node that merges it at the ceiling; after the 12-bit counter
// is exhausted every later write gets the SAME version, which peers treat as
// "equal — no-op": writes stop replicating and anti-entropy cannot see it.
func TestReview_farFutureVersionExhaustsVersionDomain(t *testing.T) {
	wall := func() time.Time { return time.UnixMilli(testNowMs) }
	mk := func(id uint16) (*repl.Core, *stateStore) {
		st := &stateStore{store: storage.NewStore(id, clock.NewClock(wall, 0, 0, testDriftWarn), storage.Config{})}
		return repl.New(id, testRunID(id), handleFor(id), st, repl.Config{}), st
	}
	coreA, stA := mk(localID)
	_, stB := mk(peerID1)
	var out port.Outbox
	now := at(0)
	coreA.Advance(now, &out)
	out.Drain()

	const maxL = int64(1)<<48 - 1
	poison := storage.Record{Version: version.PackVersion(maxL, 0, strangerID), Data: []byte("x")}
	coreA.HandleDatagram(now, replFrom(strangerID, "poison", poison), &out)
	if d := out.Drain(); len(d) != 1 {
		t.Fatalf("poison record: %d emissions", len(d))
	}
	if _, err := stA.store.Get("poison"); err != nil {
		t.Skip("poison record was rejected — defect not present")
	}
	// Ordinary traffic exhausts the 4096-step counter.
	for i := range 5000 {
		if _, _, err := stA.store.Set(fmt.Sprintf("burn/%d", i%7), []byte("b")); err != nil {
			t.Fatal(err)
		}
	}
	vOld, _, _ := stA.store.Set(keyAlpha, []byte("old"))
	recOld := stA.store.Records()[keyAlpha]
	vNew, _, _ := stA.store.Set(keyAlpha, []byte("new"))
	recNew := stA.store.Records()[keyAlpha]
	// Replicate both to B in order.
	_, _, _ = stB.store.Merge(keyAlpha, recOld)
	_, _, _ = stB.store.Merge(keyAlpha, recNew)
	gotB, _ := stB.store.Get(keyAlpha)
	if vOld == vNew || string(gotB.Data) != "new" {
		t.Fatalf("after one far-future datagram: two successive writes share version %x (equal=%v); node A holds %q, node B holds %q with the SAME version — permanent divergence invisible to DataSync", vNew[:8], vOld == vNew, "new", gotB.Data)
	}
}

// RPL-06: every DataRequest and every DataSync chunk deep-copies the whole
// store (StateStore.Records) to look at one record / ~30 versions.
func TestReview_recordsDeepCopyPerDatagram(t *testing.T) {
	const (
		keys      = 10_000
		valueSize = 1024
	)
	core, st := newTestCore()
	val := make([]byte, valueSize)
	for i := range keys {
		if _, _, err := st.store.Set(fmt.Sprintf("svc/v1/key-%05d", i), val); err != nil {
			t.Fatal(err)
		}
	}
	var out port.Outbox
	now := at(0)
	core.Advance(now, &out)
	out.Drain()

	measure := func(name string, n int, fn func(i int)) (perOpBytes uint64, perOp time.Duration) {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		start := time.Now()
		for i := range n {
			fn(i)
		}
		perOp = time.Since(start) / time.Duration(n)
		runtime.ReadMemStats(&after)
		perOpBytes = (after.TotalAlloc - before.TotalAlloc) / uint64(n)
		t.Logf("%-28s %8.2f MiB allocated, %v per datagram", name, float64(perOpBytes)/(1<<20), perOp)
		return perOpBytes, perOp
	}
	reqBytes, _ := measure("DataRequest (1 record)", 50, func(i int) {
		core.HandleDatagram(now, requestFrom(peerID1, fmt.Sprintf("svc/v1/key-%05d", i)), &out)
	})
	chunk := make([]port.SyncEntry, 0, 33)
	snap := st.store.Records()
	for i := range 33 {
		k := fmt.Sprintf("svc/v1/key-%05d", i)
		chunk = append(chunk, port.SyncEntry{Key: k, Version: snap[k].Version})
	}
	syncBytes, _ := measure("DataSync chunk (33 entries)", 50, func(int) {
		core.HandleDatagram(now, syncFrom(peerID1, chunk...), &out)
	})
	const budget = 64 << 10 // a generous bound for serving one <=60KiB record
	if reqBytes > budget || syncBytes > budget {
		t.Fatalf("store of %d keys x %d B: one DataRequest allocates %d B and one DataSync chunk %d B — a full-store deep copy per datagram (empty-node recovery = %d requests => %.0f GiB of copying; one sync round = ~%d chunks)", keys, valueSize, reqBytes, syncBytes, keys, float64(reqBytes)*keys/(1<<30), keys/33)
	}
}

// RPL-07: the discovery flush (and every later retransmission round) emits
// the WHOLE pending set in one handler call, bypassing sync_response_rate.
func TestReview_discoveryFlushIsOneUnpacedBurst(t *testing.T) {
	const keys = 5000
	core, st := newTestCore()
	var out port.Outbox
	now := at(0)
	core.Advance(now, &out)
	out.Drain()
	val := make([]byte, 8<<10)
	for i := range keys { // the node serves writes alone (peer down for hours)
		k := fmt.Sprintf("k/%05d", i)
		if _, _, err := st.store.Set(k, val); err != nil {
			t.Fatal(err)
		}
		core.NotifyLocalWrite(now, k, st.store.Records()[k], &out)
		out.Drain()
	}
	core.HandleDatagram(at(time.Hour), helloFrom(peerID1), &out)
	burst := broadcastRepls(out.Drain())
	var bytesOut int
	for _, m := range burst {
		bytesOut += len(m.Record.Data)
	}
	// The ladder keeps all entries on one shared deadline.
	core.Advance(at(time.Hour+ackTimeout), &out)
	retry := broadcastRepls(out.Drain())
	if len(burst) > 500 || len(retry) > 500 {
		t.Fatalf("peer discovery emitted %d DataReplication broadcasts (%.1f MiB) in ONE event-loop iteration and %d more 200ms later; sync_response_rate (500/s) exists precisely so recovery does not overrun socket buffers, yet this path bypasses it", len(burst), float64(bytesOut)/(1<<20), len(retry))
	}
}

// RPL-08: with a duplicate node_id elsewhere on the segment, a THIRD node
// sees the two runs alternate and treats every Hello as a restart: one full
// targeted listing per Hello, forever, and no duplicate-node_id discard or
// log line on this node.
func TestReview_duplicateNodeIDFlapsRestartOnThirdNode(t *testing.T) {
	var logs []string
	core := newDiscardCore(&logs)
	var out port.Outbox
	core.Advance(at(0), &out)
	out.Drain()
	listings := 0
	for sec := range 10 {
		for i, clone := range []struct {
			run    [16]byte
			handle string
		}{{runIDFor(1), "10.0.0.2:7900"}, {runIDFor(2), "10.0.0.3:7900"}} {
			in := helloFromRun(clone.run)
			in.Source = clone.handle
			now := at(time.Duration(sec)*time.Second + time.Duration(i)*500*time.Millisecond)
			core.HandleDatagram(now, in, &out)
			for _, ob := range out.Drain() {
				if _, ok := ob.Msg.(port.DataSync); ok && !ob.Broadcast {
					listings++
				}
			}
		}
	}
	if listings > 2 || core.Discards().DuplicateNodeID == 0 || len(logs) == 0 {
		t.Fatalf("two live nodes sharing node_id %d for 10s: third node sent %d targeted full listings (one per Hello), counted %d duplicate_node_id discards and logged %d configuration errors — the misconfiguration is invisible here and turns into a perpetual listing storm", peerID1, listings, core.Discards().DuplicateNodeID, len(logs))
	}
}

// RPL-09: the reactive egress queue has no bound and no de-duplication:
// re-delivered listings (duplicates, several peers, the storm above) enqueue
// the same responses again while the drain stays at sync_response_rate.
func TestReview_egressQueueUnboundedNoDedup(t *testing.T) {
	core, _ := newTestCore()
	var out port.Outbox
	now := at(0)
	core.Advance(now, &out)
	out.Drain()
	entries := make([]port.SyncEntry, 0, 30)
	for i := range 30 {
		entries = append(entries, port.SyncEntry{Key: fmt.Sprintf("missing/%02d", i), Version: version.PackVersion(testNowMs, 0, peerID1)})
	}
	for range 1000 { // the same chunk from the same peer, 1000 times in 1s
		core.HandleDatagram(now, syncFrom(peerID1, entries...), &out)
	}
	if q := core.QueuedResponses(); q > len(entries) {
		t.Fatalf("1000 deliveries of one 30-entry chunk queued %d DataRequests for 30 distinct keys (%.0fs of backlog at 500/s, head-of-line for every other peer's replies)", q, float64(q)/500)
	}
}

// BIN/RPL: an over-large sync_chunk_bytes (accepted by main without any
// upper bound) crashes the process at the first listing build.
func TestReview_hugeSyncChunkBytesPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("--sync-chunk-bytes=%d: first Advance panicked: %v", 1<<50, r)
		}
	}()
	fixedNow := func() time.Time { return time.UnixMilli(testNowMs) }
	st := &stateStore{store: storage.NewStore(localID, clock.NewClock(fixedNow, 0, 0, testDriftWarn), storage.Config{})}
	core := repl.New(localID, testRunID(localID), handleFor(localID), st, repl.Config{SyncChunkBytes: 1 << 50})
	var out port.Outbox
	core.Advance(at(0), &out)
}

// RPL (perf): the per-event mirror cost. udprepl mirrors PendingSnapshot,
// PeerSnapshot and UnackedStats after EVERY loop iteration; with the peer
// down the pending set holds one entry per written key.
func TestReview_perEventMirrorCostWithLargePending(t *testing.T) {
	const keys = 10_000
	core, st := newTestCore()
	var out port.Outbox
	now := at(0)
	core.Advance(now, &out)
	out.Drain()
	for i := range keys {
		k := fmt.Sprintf("svc/v1/key-%05d", i)
		_, _, _ = st.store.Set(k, []byte("v"))
		core.NotifyLocalWrite(now, k, st.store.Records()[k], &out)
		out.Drain()
	}
	const rounds = 200
	start := time.Now()
	for range rounds {
		snap := core.PendingSnapshot() // mirrorState
		_ = len(core.PeerSnapshot())   // mirrorStats
		_ = core.UnackedStats(now)     // mirrorStats
		copied := make(map[string]repl.PendingInfo, len(snap))
		for k, v := range snap { // mirrorState's copy-on-write clone
			copied[k] = v
		}
	}
	per := time.Since(start) / rounds
	t.Logf("pending=%d: %v of mirror work per event-loop iteration => at most %.0f events/s on one core", keys, per, float64(time.Second)/float64(per))
	if per > 200*time.Microsecond {
		t.Fatalf("per-event mirror cost %v with %d pending entries: the single event loop cannot sustain the documented ~1000 writes/s while a peer is down", per, keys)
	}
}
