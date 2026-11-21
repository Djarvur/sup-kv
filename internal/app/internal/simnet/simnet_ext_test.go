package simnet_test

// The simulator extension tests: the executable acceptance gate for plan
// 02-03's additive API (D-05, TEST-06) and the machine-checked proof of the
// determinism contract every simnet addition documents. The real protobuf
// encoder is injected from this test layer through Policy.SizeOf, so the
// non-test harness keeps its stdlib + internal/app + internal/port import
// block (the sup-kv-app-no-io depguard boundary, RESEARCH Pitfall 7) while
// the size gate measures exactly what a real socket would send.

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/internal/simnet"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// The extension suite's fixed profile values: one virtual millisecond of
// one-way delay (so every scheduled instant is arithmetic, never a guess),
// a 20ms virtual dump latency for record announcements, and a datagram
// ceiling that sits between the small and oversize messages the size-gate
// test drives.
const (
	simExtOneWay      = time.Millisecond
	simExtAnnounce    = 20 * time.Millisecond
	simExtSizeCeiling = 1200
)

// simExtFixedDelay returns the extension tests' loss-free network profile:
// every datagram delivered after exactly simExtOneWay, no drops, no
// duplicates and nothing partitioned, so every timing assertion is exact.
func simExtFixedDelay() simnet.Policy {
	return simnet.Policy{Delay: func(*rand.Rand) time.Duration { return simExtOneWay }}
}

// simExtRunID returns a deterministic 16-byte run id derived from tag — a
// restart seam's fresh-process identity, distinct from every node's derived
// id and admitting no crypto/rand draw (the simnet determinism law).
func simExtRunID(tag string) [16]byte {
	var id [16]byte
	copy(id[:], tag)
	return id
}

// simWireSize measures msg's full datagram size through the real protobuf
// encoder: the adapter's port-message mapping and envelope shape
// (internal/udprepl's encode path) mirrored here at the test layer, so the
// size gate under test measures what a real socket would send while simnet
// itself never imports the encoder. A marshal fault is a test-harness
// mistake and panics.
func simWireSize(msg port.Message) int {
	body := simWireBody(msg)
	if body == nil {
		panic(fmt.Sprintf("simnet_ext_test: no standalone wire body for %T", msg))
	}
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		panic(fmt.Sprintf("simnet_ext_test: marshal body: %v", err))
	}
	env := &replproto.Envelope{
		ProtoVersion: repl.SupportedProtoVersion,
		NodeId:       1,
		RunId:        make([]byte, 16),
		Crc32:        crc32.ChecksumIEEE(bodyBytes),
		Body:         bodyBytes,
	}
	datagram, err := proto.Marshal(env)
	if err != nil {
		panic(fmt.Sprintf("simnet_ext_test: marshal envelope: %v", err))
	}
	return len(datagram)
}

// simWireBody mirrors the adapter's port-message mapping (udprepl.protoBody
// is unexported and its wire seam lives in that package's test binary), so
// this file can measure every message the simulator schedules. nil reports a
// message with no standalone datagram body.
func simWireBody(msg port.Message) *replproto.Body {
	switch m := msg.(type) {
	case port.Hello:
		return &replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}}
	case port.DataReplication:
		return &replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
			Key:      m.Key,
			Version:  m.Record.Version[:],
			Data:     m.Record.Data,
			Deleted:  m.Record.Deleted,
			Checksum: m.Record.Checksum,
		}}}
	case port.Ack:
		return &replproto.Body{Msg: &replproto.Body_Ack{Ack: &replproto.Ack{Key: m.Key, Version: m.Version[:]}}}
	case port.DataRequest:
		return &replproto.Body{Msg: &replproto.Body_DataRequest{DataRequest: &replproto.DataRequest{Key: m.Key}}}
	case port.DataSync:
		entries := make([]*replproto.SyncEntry, len(m.Entries))
		for i, entry := range m.Entries {
			entries[i] = &replproto.SyncEntry{Key: entry.Key, Version: entry.Version[:]}
		}
		return &replproto.Body{Msg: &replproto.Body_DataSync{DataSync: &replproto.DataSync{Entries: entries}}}
	default:
		return nil
	}
}

// TestSim_Delete_replicatesTombstone proves the simulator's deletion author
// sequence: Cluster.Delete writes a fresh-version tombstone that rides the
// same NotifyLocalWrite path as a write and replicates to the peer, while a
// delete of a missing or already-tombstoned key is storage.ErrNotFound and
// burns no version.
func TestSim_Delete_replicatesTombstone(t *testing.T) {
	t.Parallel()
	const seed = int64(11)
	rng := simRNG(seed)
	sched := simnet.NewScheduler(simAt(0))
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, simExtFixedDelay())
	sched.RunUntil(simAt(50 * time.Millisecond))

	live := cluster.Write(0, "delete/target", []byte("value"))
	tombstone, err := cluster.Delete(0, "delete/target")
	if err != nil {
		t.Fatalf("Delete(delete/target) error = %v, want nil after a live write", err)
	}
	if !tombstone.Deleted {
		t.Fatal("Delete returned a live record — the author sequence must write a tombstone")
	}
	if tombstone.Version.Compare(live.Version) <= 0 {
		t.Fatalf("tombstone version %x does not dominate the written version %x", tombstone.Version, live.Version)
	}

	// The ErrNotFound paths: a missing key and the just-tombstoned key both
	// change nothing and append no write-log entry (no version burned).
	for _, key := range []string{"delete/missing", "delete/target"} {
		logBefore := len(cluster.WriteLog())
		rec, err := cluster.Delete(0, key)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("Delete(%q) error = %v, want storage.ErrNotFound", key, err)
		}
		if rec.Version != [16]byte{} || len(rec.Data) != 0 || rec.Deleted || rec.Checksum != 0 {
			t.Fatalf("Delete(%q) returned %+v, want the zero record", key, rec)
		}
		if got := len(cluster.WriteLog()); got != logBefore {
			t.Fatalf("write log grew from %d to %d entries on the no-op Delete(%q)", logBefore, got, key)
		}
	}

	// No version is burned by a failed delete: two writes at the same virtual
	// instant stamp consecutive counter values with the failed deletes
	// interleaved (the clock increments its counter on every tick inside one
	// millisecond).
	before := cluster.Write(0, "delete/probe-before", []byte("a"))
	_, err = cluster.Delete(0, "delete/missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("interleaved Delete error = %v, want storage.ErrNotFound", err)
	}
	after := cluster.Write(0, "delete/probe-after", []byte("b"))
	_, beforeCounter, _ := before.Version.Unpack()
	_, afterCounter, _ := after.Version.Unpack()
	if afterCounter != beforeCounter+1 {
		t.Fatalf("counter after the failed delete = %d, want %d — the no-op delete burned a version", afterCounter, beforeCounter+1)
	}

	sched.RunUntil(simAt(200 * time.Millisecond))
	peer, ok := cluster.Store(1).Records()["delete/target"]
	if !ok {
		t.Fatal("the peer never received the tombstone for delete/target")
	}
	if !peer.Deleted || len(peer.Data) != 0 {
		t.Fatalf("peer record = %+v, want a deleted tombstone carrying no data", peer)
	}
	if peer.Version != tombstone.Version {
		t.Fatalf("peer tombstone version %x, want the replicated version %x", peer.Version, tombstone.Version)
	}
}

// TestSim_ReplaceNodeWithState_preservesState proves the restart-with-
// preserved-state seam: the replacement boots holding the seeded records and
// peers keep converging on them, while a plain ReplaceNode boots empty and
// re-derives the same state over the network — the documented contrast the
// seeded sibling exists for.
func TestSim_ReplaceNodeWithState_preservesState(t *testing.T) {
	t.Parallel()
	const seed = int64(17)

	// The seeded restart: two writes converge, then the writer restarts onto
	// its own store snapshot.
	rng := simRNG(seed)
	sched := simnet.NewScheduler(simAt(0))
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, simExtFixedDelay())
	sched.RunUntil(simAt(50 * time.Millisecond))
	one := cluster.Write(0, "restart/one", []byte("one"))
	two := cluster.Write(0, "restart/two", []byte("two"))
	sched.RunUntil(simAt(100 * time.Millisecond))
	if !recordsMapsEqual(cluster.Store(0).Records(), cluster.Store(1).Records()) {
		t.Fatal("the two nodes did not converge before the restart")
	}

	cluster.ReplaceNodeWithState(0, simExtRunID("restart-with-state"), 0, cluster.Store(0).Records())
	want := map[string]storage.Record{"restart/one": one, "restart/two": two}
	if got := cluster.Store(0).Records(); !recordsMapsEqual(got, want) {
		t.Fatalf("replacement store = %+v, want the seeded records %+v", got, want)
	}
	sched.RunUntil(simAt(time.Second))
	if !recordsMapsEqual(cluster.Store(0).Records(), cluster.Store(1).Records()) {
		t.Fatalf("stores diverged after the seeded restart: %+v vs %+v", cluster.Store(0).Records(), cluster.Store(1).Records())
	}

	// The control: an empty replacement does NOT preserve state — the records
	// only return through reconciliation, which is the difference the seeded
	// boot removes.
	rng2 := simRNG(seed)
	sched2 := simnet.NewScheduler(simAt(0))
	control := simnet.NewCluster(sched2, rng2, 2, repl.Config{}, simExtFixedDelay())
	sched2.RunUntil(simAt(50 * time.Millisecond))
	control.Write(0, "restart/one", []byte("one"))
	control.Write(0, "restart/two", []byte("two"))
	sched2.RunUntil(simAt(100 * time.Millisecond))
	control.ReplaceNode(0, simExtRunID("plain-restart"), 0)
	if got := control.Store(0).Records(); len(got) != 0 {
		t.Fatalf("plain ReplaceNode kept %d records, want an empty store — the control must not preserve state", len(got))
	}
	sched2.RunUntil(simAt(time.Second))
	if !recordsMapsEqual(control.Store(0).Records(), control.Store(1).Records()) {
		t.Fatalf("the plain replacement did not re-derive the peer's state: %+v vs %+v", control.Store(0).Records(), control.Store(1).Records())
	}
}

// TestSim_recvBudget_refusesAndCounts proves the per-node receive budget: the
// first delivery after the budget is set is accepted, every later one is
// refused at fire time and counted, and the sender's pending set keeps the
// refused writes unconfirmed.
func TestSim_recvBudget_refusesAndCounts(t *testing.T) {
	t.Parallel()
	const seed = int64(23)
	rng := simRNG(seed)
	sched := simnet.NewScheduler(simAt(0))
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, simExtFixedDelay())
	sched.RunUntil(simAt(50 * time.Millisecond))

	// Node id 2 (index 1) accepts exactly one more datagram after discovery.
	cluster.Network().SetRecvBudget(2, 1)
	keys := []string{"budget/one", "budget/two", "budget/three"}
	for _, key := range keys {
		cluster.Write(0, key, []byte("v"))
	}
	sched.RunUntil(simAt(60 * time.Millisecond))

	faults := cluster.Network().FaultStats()
	if faults.RecvBudgetRefused != 2 {
		t.Fatalf("RecvBudgetRefused = %d, want 2 — one delivery accepted, the next two refused", faults.RecvBudgetRefused)
	}
	peer := cluster.Store(1).Records()
	if _, ok := peer[keys[0]]; !ok {
		t.Fatalf("the first post-budget write %q never arrived", keys[0])
	}
	for _, key := range keys[1:] {
		if _, ok := peer[key]; ok {
			t.Fatalf("%q arrived despite an exhausted receive budget", key)
		}
	}

	// The sender's behavior is consistent: the two refused writes stay
	// unconfirmed (the accepted one drained on its Ack at 52ms).
	if got := cluster.Core(0).UnackedStats(simAt(60 * time.Millisecond)).Count; got != 2 {
		t.Fatalf("sender UnackedStats().Count = %d, want 2 — the refused writes must stay unconfirmed", got)
	}
}

// TestSim_sizeRefusal_byRealEncoder proves the size gate measures with the
// real encoder: a message the encoder measures over the configured ceiling is
// refused before it is scheduled and counted, while the under-limit message
// is delivered.
func TestSim_sizeRefusal_byRealEncoder(t *testing.T) {
	t.Parallel()
	const seed = int64(31)
	small := []byte("small-value")
	big := bytes.Repeat([]byte("x"), 2000)
	smallSize := simWireSize(port.DataReplication{Key: "size/small", Record: storage.Record{Data: small}})
	bigSize := simWireSize(port.DataReplication{Key: "size/big", Record: storage.Record{Data: big}})
	limit := (smallSize + bigSize) / 2
	if smallSize >= limit || bigSize <= limit {
		t.Fatalf("limit %d does not separate the message sizes (small %d, big %d)", limit, smallSize, bigSize)
	}

	rng := simRNG(seed)
	sched := simnet.NewScheduler(simAt(0))
	policy := simnet.Policy{
		Delay:            func(*rand.Rand) time.Duration { return simExtOneWay },
		SizeOf:           simWireSize,
		MaxDatagramBytes: limit,
	}
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, policy)
	// Discovery survives the gate: Hello and the startup DataSync measure far
	// below the ceiling.
	sched.RunUntil(simAt(50 * time.Millisecond))
	cluster.Write(0, "size/small", small)
	cluster.Write(0, "size/big", big)
	sched.RunUntil(simAt(60 * time.Millisecond))

	faults := cluster.Network().FaultStats()
	if faults.OversizeRefused != 2 {
		t.Fatalf("OversizeRefused = %d, want 2 — the oversize broadcast is refused once per recipient (self echo and peer)", faults.OversizeRefused)
	}
	peer := cluster.Store(1).Records()
	if _, ok := peer["size/small"]; !ok {
		t.Fatal("the under-limit record never arrived")
	}
	if _, ok := peer["size/big"]; ok {
		t.Fatal("the over-limit record arrived despite the size gate")
	}
}

// TestSim_announceDelay_virtualDumpLatency proves the virtual dump latency:
// a record announcement becomes visible to the peer exactly after the
// configured delay in scheduler time (never before, and the trace logs the
// delayed instant), while a non-announcement message waits only the drawn
// one-way delay.
func TestSim_announceDelay_virtualDumpLatency(t *testing.T) {
	t.Parallel()
	const seed = int64(37)
	rng := simRNG(seed)
	sched := simnet.NewScheduler(simAt(0))
	policy := simnet.Policy{
		Delay:         func(*rand.Rand) time.Duration { return simExtOneWay },
		AnnounceDelay: func(port.Message) time.Duration { return simExtAnnounce },
	}
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, policy)
	sched.RunUntil(simAt(50 * time.Millisecond))

	writeAt := sched.Now()
	cluster.Write(0, "announce/delayed", []byte("value"))
	arrival := writeAt.Add(simExtOneWay + simExtAnnounce)

	sched.RunUntil(arrival.Add(-time.Millisecond))
	if _, ok := cluster.Store(1).Records()["announce/delayed"]; ok {
		t.Fatal("the record became visible before the virtual dump latency elapsed")
	}
	if got := strings.Count(cluster.Network().TraceDigest(), "|n1|n2|port.DataReplication"); got != 0 {
		t.Fatalf("trace logged %d peer DataReplication deliveries before the latency elapsed", got)
	}

	sched.RunUntil(arrival)
	if _, ok := cluster.Store(1).Records()["announce/delayed"]; !ok {
		t.Fatal("the record was not visible exactly at the virtual dump latency")
	}
	wantArrival := fmt.Sprintf("%d|n1|n2|port.DataReplication", arrival.UnixMilli())
	if digest := cluster.Network().TraceDigest(); !strings.Contains(digest, wantArrival) {
		t.Fatalf("trace %q does not contain %q — the delayed delivery must be logged at its virtual time", digest, wantArrival)
	}

	// The Ack is not an announcement: AnnounceDelay applies to DataReplication
	// only, so the Ack waits just the drawn one-way delay.
	ackArrival := arrival.Add(simExtOneWay)
	sched.RunUntil(ackArrival)
	wantAck := fmt.Sprintf("%d|n2|n1|port.Ack", ackArrival.UnixMilli())
	if digest := cluster.Network().TraceDigest(); !strings.Contains(digest, wantAck) {
		t.Fatalf("trace does not contain %q — AnnounceDelay must not delay a non-announcement message", wantAck)
	}
	if got := cluster.Core(0).UnackedStats(ackArrival).Count; got != 0 {
		t.Fatalf("UnackedStats().Count = %d after the Ack, want 0", got)
	}
}

// runSimExtScenario runs one same-seed replay scenario with every extension
// active: drops, duplicates and drawn delays (the one injected rng), the
// real-encoder size gate, the virtual announcement latency, a receive budget,
// deletion replication and a restart-with-preserved-state. The traffic script
// and every offset are fixed — nothing outside simnet draws from the rng — so
// the whole run is a pure function of the seed.
func runSimExtScenario(seed int64) (string, []map[string]storage.Record, simnet.FaultStats) {
	rng := simRNG(seed)
	sched := simnet.NewScheduler(simAt(0))
	cluster := simnet.NewCluster(sched, rng, 2, repl.Config{}, simnet.Policy{
		DropProb:         0.10,
		DupProb:          0.05,
		Delay:            simDelay,
		SizeOf:           simWireSize,
		MaxDatagramBytes: simExtSizeCeiling,
		AnnounceDelay:    func(port.Message) time.Duration { return simExtAnnounce },
	})
	sched.RunUntil(simAt(50 * time.Millisecond))
	cluster.Write(0, "ext/alpha", []byte("alpha"))
	cluster.Write(1, "ext/beta", []byte("beta"))
	// A three-delivery budget: the scenario runs long enough to exhaust it, so
	// receive-side refusals are part of the replay.
	cluster.Network().SetRecvBudget(1, 3)
	sched.RunUntil(simAt(100 * time.Millisecond))
	cluster.Write(0, "ext/oversize", bytes.Repeat([]byte("x"), 4000))
	// The delete's target may be absent on node 0 under the lossy profile;
	// either outcome is deterministic and exercises the author path.
	_, _ = cluster.Delete(0, "ext/beta")
	cluster.ReplaceNodeWithState(0, simExtRunID("ext-restart"), 0, cluster.Store(0).Records())
	cluster.Write(0, "ext/gamma", []byte("gamma"))
	sched.RunUntil(simAt(1500 * time.Millisecond))

	stores := make([]map[string]storage.Record, cluster.Nodes())
	for i := range stores {
		stores[i] = cluster.Store(i).Records()
	}
	return cluster.Network().TraceDigest(), stores, cluster.Network().FaultStats()
}

// TestSim_determinism_withExtensions is the executable determinism-contract
// gate (T-2-05): the same seed replays byte-identically with every extension
// active — identical TraceDigest, identical fault counters and identical
// final stores — and the counters prove the extensions actually fired rather
// than being bypassed.
func TestSim_determinism_withExtensions(t *testing.T) {
	t.Parallel()
	const seed = int64(43)
	digest1, stores1, faults1 := runSimExtScenario(seed)
	digest2, stores2, faults2 := runSimExtScenario(seed)

	if digest1 == "" {
		t.Fatal("trace digest is empty — the determinism comparison would be vacuous")
	}
	if faults1.OversizeRefused == 0 {
		t.Fatal("OversizeRefused = 0 — the size gate never fired, so the replay does not exercise it")
	}
	if faults1.RecvBudgetRefused == 0 {
		t.Fatal("RecvBudgetRefused = 0 — the receive budget never fired, so the replay does not exercise it")
	}
	if digest1 != digest2 {
		t.Fatalf("seed %d produced two different traces (%d vs %d digest bytes) — a nondeterminism source leaked in with the extensions active", seed, len(digest1), len(digest2))
	}
	if faults1 != faults2 {
		t.Fatalf("seed %d produced different fault counters: %+v vs %+v", seed, faults1, faults2)
	}
	if len(stores1) != len(stores2) {
		t.Fatalf("two runs returned %d vs %d node stores", len(stores1), len(stores2))
	}
	for i := range stores1 {
		if !recordsMapsEqual(stores1[i], stores2[i]) {
			t.Fatalf("node %d ended with different stores across the two runs of seed %d", i, seed)
		}
		if len(stores1[i]) == 0 {
			t.Fatalf("node %d ended with an empty store — the scenario did not exercise the seeded restart", i)
		}
	}
}
