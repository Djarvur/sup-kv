// Package simnet is the deterministic simulator harness behind plan 04-05's
// convergence proofs: a discrete-event scheduler ordered by (Time, Seq), a
// fault-injecting virtual network, and an N-node cluster harness driving the
// same repl.Core handlers the production UDP adapter drives (no simulation
// mode anywhere — the determinism argument IS that the same handlers run;
// a sim-side reimplementation of any protocol decision would prove nothing
// about production). It is a normal package linked only into test binaries
// (the internal/app/internal/testutil precedent); unlike testutil its
// directory name does not match the no-test-libs-in-prod glob, so the
// non-test files here must not import Go's testing package — assertions live
// only in _test.go files. The package starts no goroutines, takes no locks
// and never reads a wall clock or sleeps: virtual time is the only time
// (D-06/D-07, RESEARCH Pitfall 9), every time value is built from
// [time.UnixMilli] arithmetic — no monotonic reading anywhere (the D-02 seam
// discipline) — and every random draw flows through the one rng injected at
// construction, drawn at send time, so the same seed reproduces the whole
// run byte for byte.
package simnet

import (
	"container/heap"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// clusterDriftWarn is the store clocks' drift tolerance inside the simulator
// (the docs/IDEA.md design default; the sim never reads the drift signal).
const clusterDriftWarn = time.Minute

// event is one scheduled callback at a virtual time; seq is the insertion
// counter that totally orders events sharing a time.
type event struct {
	at  time.Time
	seq uint64
	fn  func()
}

// eventHeap orders events by the (at, seq) total order — time first, then
// insertion order (container/heap is the RESEARCH "Don't Hand-Roll" pick for
// the deterministic priority queue).
type eventHeap []event

// Len implements [heap.Interface].
func (h *eventHeap) Len() int { return len(*h) }

// Less implements [heap.Interface]: strictly earlier times first; equal
// times break on the insertion counter, never on heap layout.
func (h *eventHeap) Less(i, j int) bool {
	if !(*h)[i].at.Equal((*h)[j].at) {
		return (*h)[i].at.Before((*h)[j].at)
	}
	return (*h)[i].seq < (*h)[j].seq
}

// Swap implements [heap.Interface].
func (h *eventHeap) Swap(i, j int) { (*h)[i], (*h)[j] = (*h)[j], (*h)[i] }

// Push implements [heap.Interface].
func (h *eventHeap) Push(x any) { *h = append(*h, x.(event)) }

// Pop implements [heap.Interface].
func (h *eventHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Scheduler is the discrete-event virtual clock (D-06/D-07, RESEARCH Pattern
// 4): events execute in the strict (Time, Seq) total order — every event
// with an earlier virtual time first, and events scheduled at the same
// virtual time in Schedule-call (insertion) order, the tie-break contract
// same-seed determinism is built on. Now advances only as RunUntil executes
// events; nothing here reads a wall clock, sleeps or launches a goroutine.
type Scheduler struct {
	now  time.Time
	seq  uint64
	pend eventHeap
}

// NewScheduler returns a scheduler whose virtual clock starts at start.
func NewScheduler(start time.Time) *Scheduler {
	return &Scheduler{now: start, pend: eventHeap{}}
}

// Schedule enqueues fn at virtual time t, assigning it the next monotonically
// increasing insertion counter. An event scheduled at or before the current
// virtual time is simply due: the next RunUntil covering its time executes
// it, ordered before any later-time event.
func (s *Scheduler) Schedule(t time.Time, fn func()) {
	s.seq++
	heap.Push(&s.pend, event{at: t, seq: s.seq, fn: fn})
}

// RunUntil executes every event with (Time, Seq) at or before t, in that
// total order, advancing the virtual clock to each event's time before it
// runs — so handlers observe exactly the event's scheduled time. Events
// scheduled by executed events (network deliveries at now+delay, timer
// pumps) join the same order and fire inside this same call whenever their
// time is at or before t. Afterwards the clock rests at t whether or not any
// event ran.
func (s *Scheduler) RunUntil(t time.Time) {
	for s.pend.Len() > 0 && !s.pend[0].at.After(t) {
		ev := heap.Pop(&s.pend).(event)
		s.now = ev.at
		ev.fn()
	}
	s.now = t
}

// Now returns the current virtual time.
func (s *Scheduler) Now() time.Time { return s.now }

// Policy is the fault-injection profile, pure data injected by each test:
// drop and duplicate probabilities, the one-way delay draw, the partition
// predicate, the optional datagram size gate (SizeOf over MaxDatagramBytes)
// and the optional record-announcement latency (AnnounceDelay). A nil Delay
// means zero delay and a nil Partition means nothing is ever blocked (the
// fixed-delay profile of the bound tests and the loss-free default); a nil
// SizeOf, a non-positive MaxDatagramBytes and a nil AnnounceDelay disable
// their extensions — the zero Policy reproduces the pre-extension behavior
// bit for bit. simnet draws randomness only through the injected rng, at
// send time, in a fixed draw order — drop, then duplicate, then each
// delivery's delay — so the same seed reproduces every fate decision
// (D-06/D-07); the size gate and the announcement latency are pure
// functions of the message (no rng, no wall clock) and the receive budget is
// plain state consulted at delivery time (SetRecvBudget).
type Policy struct {
	DropProb  float64
	DupProb   float64
	Delay     func(rng *rand.Rand) time.Duration
	Partition func(from, to string, now time.Time) bool

	// SizeOf, when non-nil, measures a message's simulated datagram size in
	// bytes — the test layer injects the real encoder's length here (the Q2
	// resolution: the replproto encode stays test-side, so this file's
	// import block remains stdlib + internal/app + internal/port). With a
	// positive MaxDatagramBytes it gates scheduleDelivery: a message
	// measuring over the ceiling is refused before it is ever scheduled —
	// counted in FaultStats.OversizeRefused and dropped, never delivered
	// (the message-too-long refusal of a real UDP path, F2c).
	SizeOf func(msg port.Message) int

	// MaxDatagramBytes is the size gate's ceiling in bytes; only a positive
	// value arms the gate (see SizeOf), so the zero disables it.
	MaxDatagramBytes int

	// AnnounceDelay, when non-nil, returns the extra virtual latency a
	// record announcement — a port.DataReplication, the write-path
	// broadcast and the DataRequest/sync answers alike — waits beyond the
	// drawn one-way delay, mirroring production's dump-then-broadcast order
	// (the DataReplication is broadcast after the dump is confirmed,
	// RESULTS.md §4). simnet consults it only for DataReplication messages
	// and adds the returned latency to the delivery delay. Pure function of
	// the message — no rng, no wall clock.
	AnnounceDelay func(msg port.Message) time.Duration
}

// delay draws one one-way delay (nil policy Delay: zero).
func (p Policy) delay(rng *rand.Rand) time.Duration {
	if p.Delay == nil {
		return 0
	}
	return p.Delay(rng)
}

// blocked consults the partition predicate (nil policy Partition: open).
func (p Policy) blocked(from, to string, now time.Time) bool {
	return p.Partition != nil && p.Partition(from, to, now)
}

// FaultStats counts the fault injector's observed classes (the coverage
// teeth the chaos test asserts on): sends dropped at send time, sends
// duplicated into two deliveries, deliveries arriving after a later-sent
// sibling of the same directed pair, sends blocked by the partition
// predicate, deliveries refused past the recipient's receive budget
// (SetRecvBudget), and messages refused by the size gate before scheduling
// (Policy.SizeOf over Policy.MaxDatagramBytes).
type FaultStats struct {
	Dropped     uint64
	Duplicated  uint64
	Reordered   uint64
	Partitioned uint64

	// RecvBudgetRefused counts deliveries dropped at their delivery time
	// because the recipient's receive budget (SetRecvBudget) was exhausted
	// — the receive-side drop behavior of a real socket.
	RecvBudgetRefused uint64

	// OversizeRefused counts messages the size gate refused before they
	// were ever scheduled (Policy.SizeOf over Policy.MaxDatagramBytes).
	OversizeRefused uint64
}

// netNode is one registered cluster member: the envelope identity (node id,
// run id) its datagrams carry, its handle, and the delivery function — the
// cluster wires it to the node's core handler followed by the drive loop.
type netNode struct {
	id      uint16
	runID   [16]byte
	handle  string
	deliver func(port.InboundDatagram)
}

// pairKey identifies one directed (from, to) handle pair — the granularity
// of the send-sequence tracking behind the reorder observation.
type pairKey struct{ from, to string }

// Network is the fault-injecting virtual datagram network (RESEARCH Pattern
// 4): every Send consults the partition policy, then draws the datagram's
// fate AT SEND TIME from the injected rng — drop (counted), duplicate (two
// deliveries at independently drawn delays), or a single delivery at
// now+delay — and every delivery is a scheduler event that hands the
// recipient's delivery function a Decoded InboundDatagram carrying the
// sender's envelope identity. Broadcast fans out to every registered node
// INCLUDING the sender (own echo is real; plan 01's OwnEcho class fires in
// simulation too), one independent fate draw per recipient, in registration
// order. Registered nodes iterate a slice, never a map, so fan-out order —
// and therefore rng draw order — is deterministic. A per-recipient receive
// budget (SetRecvBudget) models a real socket's receive-side drop behavior:
// a delivery due past its recipient's budget is refused at fire time and
// counted, never delivered.
type Network struct {
	sched      *Scheduler
	rng        *rand.Rand
	policy     Policy
	nodes      []netNode
	byHandle   map[string]int
	sendSeq    map[pairKey]uint64
	lastByPair map[pairKey]uint64
	recvBudget map[string]int
	faults     FaultStats
	pending    int
	trace      strings.Builder
}

// NewNetwork returns the network over sched, drawing every fault from rng
// under policy.
func NewNetwork(sched *Scheduler, rng *rand.Rand, policy Policy) *Network {
	return &Network{
		sched:      sched,
		rng:        rng,
		policy:     policy,
		byHandle:   make(map[string]int),
		sendSeq:    make(map[pairKey]uint64),
		lastByPair: make(map[pairKey]uint64),
		recvBudget: make(map[string]int),
	}
}

// Register makes (nodeID, runID, handle) a reachable delivery target whose
// inbound datagrams invoke deliver.
func (n *Network) Register(nodeID uint16, runID [16]byte, handle string, deliver func(port.InboundDatagram)) {
	n.byHandle[handle] = len(n.nodes)
	n.nodes = append(n.nodes, netNode{id: nodeID, runID: runID, handle: handle, deliver: deliver})
}

// Reregister atomically replaces the registration of an already-registered
// handle: the node id, run id and delivery function swap in place, keeping
// the entry's position in the registration slice — broadcast fan-out order
// and therefore rng draw order stay exactly as before (the in-place node
// replacement the scenario suite drives; a fresh handle still goes through
// Register). It reports false when handle is not registered.
func (n *Network) Reregister(nodeID uint16, runID [16]byte, handle string, deliver func(port.InboundDatagram)) bool {
	i, ok := n.byHandle[handle]
	if !ok {
		return false
	}
	n.nodes[i] = netNode{id: nodeID, runID: runID, handle: handle, deliver: deliver}
	return true
}

// SetRecvBudget caps the datagrams nodeID may accept: every delivered
// datagram decrements the remaining budget at its delivery time, and a
// delivery due once the budget is exhausted is refused — counted in
// FaultStats.RecvBudgetRefused and dropped, never delivered (the
// receive-side drop behavior of a real socket whose buffer or rate budget
// is exhausted, TEST-06). A zero or negative maxDeliveries refuses
// everything. The cap keys the node's handle, so it survives a ReplaceNode
// restart (the socket address the budget models outlives the process behind
// it) and is read live at each fire time — a budget set between the
// scheduling and the fire of a delivery still governs that delivery.
// Setting the budget is test-facing plain state; an unregistered node id is
// a driving test's mistake and panics.
//
// Determinism contract: as SetClockOffset — plain harness state, consumes
// no rng draws and schedules no events, so for any fixed sequence of calls
// the event stream stays a pure function of the seed.
func (n *Network) SetRecvBudget(nodeID uint16, maxDeliveries int) {
	for i := range n.nodes {
		if n.nodes[i].id == nodeID {
			n.recvBudget[n.nodes[i].handle] = maxDeliveries
			return
		}
	}
	panic(fmt.Sprintf("simnet: Network.SetRecvBudget: node id %d is not registered", nodeID))
}

// Send injects one datagram from fromHandle toward toHandle (D-06/D-07): a
// blocked pair counts partitioned and drops the send; otherwise the fate is
// drawn at send time — drop, duplicate or deliver — with the delays drawn
// from the policy. Sends addressed to unregistered handles are discarded
// silently (they model addresses that no longer exist).
func (n *Network) Send(fromHandle, toHandle string, msg port.Message) {
	from, ok := n.byHandle[fromHandle]
	if !ok {
		return
	}
	to, ok := n.byHandle[toHandle]
	if !ok {
		return
	}
	now := n.sched.Now()
	if n.policy.blocked(fromHandle, toHandle, now) {
		n.faults.Partitioned++
		return
	}
	pair := pairKey{from: fromHandle, to: toHandle}
	n.sendSeq[pair]++
	seq := n.sendSeq[pair]
	switch {
	case n.rng.Float64() < n.policy.DropProb:
		n.faults.Dropped++
	case n.rng.Float64() < n.policy.DupProb:
		n.faults.Duplicated++
		n.scheduleDelivery(n.nodes[from], n.nodes[to], msg, seq, n.policy.delay(n.rng))
		n.scheduleDelivery(n.nodes[from], n.nodes[to], msg, seq, n.policy.delay(n.rng))
	default:
		n.scheduleDelivery(n.nodes[from], n.nodes[to], msg, seq, n.policy.delay(n.rng))
	}
}

// Broadcast fans one datagram out to every registered node INCLUDING the
// sender — the sender's own broadcast really does loop back (plan 01's
// OwnEcho discard class), and every recipient's fate is drawn independently
// in registration order.
func (n *Network) Broadcast(fromHandle string, msg port.Message) {
	for i := range n.nodes {
		n.Send(fromHandle, n.nodes[i].handle, msg)
	}
}

// scheduleDelivery books one delivery event at now+delay: at fire time the
// pair's reorder observation updates (a delivery whose send sequence is older
// than the pair's highest delivered sequence arrives after a later-sent
// sibling — Pitfall 1's measurable), the tuple joins the ordered trace, and
// the recipient's delivery function receives the decoded datagram. Three
// policy extensions join the path additively, each off in the zero Policy:
// the size gate (SizeOf over MaxDatagramBytes) refuses an oversize message
// before it is ever scheduled — counted in FaultStats.OversizeRefused,
// dropped, never booked (the message-too-long refusal of a real UDP path);
// the announcement latency (AnnounceDelay on a port.DataReplication) joins
// the drawn one-way delay, mirroring production's dump-then-broadcast order;
// and at fire time the recipient's receive budget is consulted first — past
// it the delivery is refused (FaultStats.RecvBudgetRefused, the message
// dropped, never delivered) and a delivered one decrements the budget (the
// receive-side drop behavior of a real socket, SetRecvBudget).
func (n *Network) scheduleDelivery(from, to netNode, msg port.Message, seq uint64, delay time.Duration) {
	if n.policy.SizeOf != nil && n.policy.MaxDatagramBytes > 0 && n.policy.SizeOf(msg) > n.policy.MaxDatagramBytes {
		n.faults.OversizeRefused++
		return
	}
	pair := pairKey{from: from.handle, to: to.handle}
	total := delay
	if n.policy.AnnounceDelay != nil {
		if ann, ok := msg.(port.DataReplication); ok {
			total += n.policy.AnnounceDelay(ann)
		}
	}
	n.pending++
	n.sched.Schedule(n.sched.Now().Add(total), func() {
		n.pending--
		if budget, ok := n.recvBudget[to.handle]; ok {
			if budget <= 0 {
				n.faults.RecvBudgetRefused++
				return
			}
			n.recvBudget[to.handle] = budget - 1
		}
		if seq < n.lastByPair[pair] {
			n.faults.Reordered++
		} else {
			n.lastByPair[pair] = seq
		}
		fmt.Fprintf(&n.trace, "%d|%s|%s|%T\n", n.sched.Now().UnixMilli(), from.handle, to.handle, msg)
		to.deliver(port.InboundDatagram{
			Source:       from.handle,
			Status:       port.Decoded,
			ProtoVersion: repl.SupportedProtoVersion,
			NodeID:       from.id,
			RunID:        from.runID,
			Msg:          msg,
		})
	})
}

// FaultStats returns the fault counters — a value copy.
func (n *Network) FaultStats() FaultStats { return n.faults }

// PendingDeliveries returns the number of scheduled-but-undelivered network
// events — one of the simulator's three quiescence signals.
func (n *Network) PendingDeliveries() int { return n.pending }

// TraceDigest returns the ordered delivery trace as one string — every
// (virtual time, from, to, message type) tuple in execution order, the
// same-seed comparison artifact.
func (n *Network) TraceDigest() string { return n.trace.String() }

// LoggedWrite is one captured cluster write — the model input the caller's
// pointwise-supremum store merges over (merge order is irrelevant,
// CORE-06); a Cluster.Delete captures its tombstone here with Rec.Deleted
// true, no separate flag needed.
type LoggedWrite struct {
	Key string
	Rec storage.Record
}

// nodeStore adapts *storage.Store to repl.StateStore by delegating both
// methods — the plan 04-01 test precedent (the persist engine's ApplyRemote
// seam lands with plan 04-06).
type nodeStore struct{ store *storage.Store }

// ApplyRemote joins one replicated record through the store's LWW merge.
func (s nodeStore) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	return s.store.Merge(key, rec)
}

// Records returns the full snapshot, tombstones included.
func (s nodeStore) Records() map[string]storage.Record { return s.store.Records() }

// simRunID returns nodeID's run identifier: fresh per node (no two cluster
// members share one) and derived deterministically — the simulator admits no
// crypto/rand draw anywhere, keeping same-seed determinism total (T-04-10).
// The node id renders as its 16 decimal digits, so no bit-narrowing
// conversion is involved.
func simRunID(nodeID uint16) [16]byte {
	var id [16]byte
	copy(id[:], fmt.Sprintf("%016d", nodeID))
	return id
}

// clusterNode is one simulated node: a real storage.Store whose clock reads
// the scheduler's virtual time plus the node's cluster-owned offset slot
// (the D-02 seam), and a real repl.Core. The timerGen counter is the
// live-timer generation drive's scheduling discipline — and the restart
// seams' stale-timer supersession.
type clusterNode struct {
	id       uint16
	runID    [16]byte
	handle   string
	store    *storage.Store
	core     *repl.Core
	timerGen uint64
}

// Cluster is the N-node harness: n repl.Core instances over n real stores,
// wired to one virtual network, with the write log capturing the production
// author sequence (D-05). Each node's store clock reads the scheduler's
// virtual time plus the node's mutable offset slot (SetClockOffset — the
// wall-clock seam), and the remembered cfg rebuilds nodes for the restart
// seams (ReplaceNode, ReplaceNodeWithState, AddNode).
type Cluster struct {
	sched   *Scheduler
	net     *Network
	nodes   []clusterNode
	writes  []LoggedWrite
	offsets []time.Duration
	cfg     repl.Config
}

// NewCluster builds n nodes — node_id 1..n, handles n1..nN in node_id order
// — over sched, each a real storage.Store whose clock now func returns the
// scheduler's current virtual time plus the node's offset slot (the D-02
// fake-clock idiom; offsets start at zero, so the clock is sched.Now until
// test code moves it) and a repl.Core built with cfg; rng is the network's
// fault-draw source, policy its fault profile. Construction schedules each
// node's first timer pump at the scheduler's current time, so the first
// RunUntil fires every node's startup Hello and DataSync (the adapter's
// Run-startup mirror).
func NewCluster(sched *Scheduler, rng *rand.Rand, n int, cfg repl.Config, policy Policy) *Cluster {
	c := &Cluster{
		sched:   sched,
		net:     NewNetwork(sched, rng, policy),
		nodes:   make([]clusterNode, n),
		offsets: make([]time.Duration, n),
		cfg:     cfg,
	}
	for i := range c.nodes {
		nodeID := uint16(i + 1)
		c.nodes[i] = c.newClusterNode(i, nodeID, simRunID(nodeID))
	}
	for i := range c.nodes {
		c.net.Register(c.nodes[i].id, c.nodes[i].runID, c.nodes[i].handle, func(in port.InboundDatagram) {
			c.deliver(i, in)
		})
		c.sched.Schedule(sched.Now(), func() { c.drive(i, sched.Now()) })
	}
	return c
}

// newClusterNode builds node index i's harness node: a fresh store whose
// clock reads the scheduler's virtual time plus the node's offset slot,
// over a fresh core announcing runID under the node's derived handle. The
// shared construction of NewCluster, ReplaceNode and AddNode — one wiring,
// never three.
func (c *Cluster) newClusterNode(i int, nodeID uint16, runID [16]byte) clusterNode {
	handle := fmt.Sprintf("n%d", nodeID)
	now := func() time.Time { return c.sched.Now().Add(c.offsets[i]) }
	store := storage.NewStore(nodeID, clock.NewClock(now, 0, 0, clusterDriftWarn), storage.Config{})
	return clusterNode{
		id:     nodeID,
		runID:  runID,
		handle: handle,
		store:  store,
		core:   repl.New(nodeID, runID, handle, nodeStore{store: store}, c.cfg),
	}
}

// SetClockOffset re-aims node i's store clock at the scheduler's virtual
// time plus offset — the wall-clock seam the scenario suite drives
// (TEST-04: live jumps forward and backward, a far-ahead node). The offset
// is live: the next clock read observes it, so a jump between RunUntil
// calls re-times every subsequent stamp of that node while its HLC carries
// the pre-jump l (monotonicity is the clock's, never the wall's).
//
// Determinism contract: offsets are plain harness state mutated only by
// test code between RunUntil calls — the seam consumes no rng draws and
// schedules no events, so for any fixed sequence of SetClockOffset calls
// the event stream stays a pure function of the seed (06-RESEARCH Open
// Question 3: the Cluster-method shape).
func (c *Cluster) SetClockOffset(i int, offset time.Duration) {
	c.checkIndex(i, "SetClockOffset")
	c.offsets[i] = offset
}

// ReplaceNode restarts node i as a fresh process: the node keeps its id and
// handle but boots a new empty store and a new core announcing runID, with
// its clock offset set to offset — the REPL-07 replacement shape
// (restart_test.go's hand-built cores, wired into the virtual network).
// The network registration is replaced in place through Reregister;
// datagrams already in flight from the old run still carry the old run id
// (the envelope identity was copied at send time), while everything the new
// run sends carries runID. The old run's pending state and timers vanish
// with its core — the generation counter supersedes every timer the old
// run had scheduled — and the startup pump scheduled at the current virtual
// time fires the new run's Hello and DataSync on the next RunUntil.
// Determinism contract: as SetClockOffset — plain harness state, no rng.
func (c *Cluster) ReplaceNode(i int, runID [16]byte, offset time.Duration) {
	c.checkIndex(i, "ReplaceNode")
	gen := c.nodes[i].timerGen + 1
	c.offsets[i] = offset
	replacement := c.newClusterNode(i, c.nodes[i].id, runID)
	replacement.timerGen = gen
	c.nodes[i] = replacement
	if !c.net.Reregister(replacement.id, runID, replacement.handle, func(in port.InboundDatagram) {
		c.deliver(i, in)
	}) {
		panic(fmt.Sprintf("simnet: Cluster.ReplaceNode: handle %q of node %d is not registered", replacement.handle, replacement.id))
	}
	c.sched.Schedule(c.sched.Now(), func() { c.drive(i, c.sched.Now()) })
}

// ReplaceNodeWithState restarts node i onto preserved state: the node keeps
// its id and handle but boots a new store seeded with records and a new core
// announcing runID, with its clock offset set to offset — the
// restart-with-preserved-state shape (TEST-06: F14's replaced node carrying
// the dump's records, seeding its create-if-absent defaults). The records
// merge into the fresh store through the production load path's join
// (store.Merge, the persist.Open mergeRecords shape) in sorted-key order —
// the fixed order is the map-iteration guard the determinism contract needs,
// and the merges ride the clock's receive rule, so the replacement's HLC
// watermark dominates every seeded version before its first local write (the
// VERS-04 restart property through the merge path). Tombstones seed exactly
// like live records. A seed record over the store's design-default value
// limit is a driving test's mistake and panics (the checkIndex/Reregister
// convention). The wiring is ReplaceNode's verbatim: the in-place node swap,
// the network registration replaced through Reregister (datagrams already in
// flight from the old run still carry the old run id), the timer-generation
// supersession, and the startup drive pump scheduled at the current virtual
// time — the new run's Hello and DataSync fire on the next RunUntil.
//
// Determinism contract: plain harness state — the seeded merges consume no
// rng draws and schedule no events beyond the drive tick, so the event
// stream stays a pure function of the seed.
func (c *Cluster) ReplaceNodeWithState(i int, runID [16]byte, offset time.Duration, records map[string]storage.Record) {
	c.checkIndex(i, "ReplaceNodeWithState")
	gen := c.nodes[i].timerGen + 1
	c.offsets[i] = offset
	replacement := c.newClusterNode(i, c.nodes[i].id, runID)
	for _, key := range slices.Sorted(maps.Keys(records)) {
		_, _, err := replacement.store.Merge(key, records[key])
		if err != nil {
			panic(fmt.Sprintf("simnet: Cluster.ReplaceNodeWithState: seeding %q on node %d: %v", key, replacement.id, err))
		}
	}
	replacement.timerGen = gen
	c.nodes[i] = replacement
	if !c.net.Reregister(replacement.id, runID, replacement.handle, func(in port.InboundDatagram) {
		c.deliver(i, in)
	}) {
		panic(fmt.Sprintf("simnet: Cluster.ReplaceNodeWithState: handle %q of node %d is not registered", replacement.handle, replacement.id))
	}
	c.sched.Schedule(c.sched.Now(), func() { c.drive(i, c.sched.Now()) })
}

// AddNode boots one fresh cluster member: the next node id with its derived
// handle and a deterministically derived fresh run id, over an empty store
// whose clock starts at the scheduler's virtual time plus offset — the
// empty-disk join shape (a brand-new node reconciling through the periodic
// rounds, never a targeted listing). It returns the new node's index
// (nodes keep index = node_id-1). Determinism contract: as SetClockOffset —
// the derived identity draws no randomness, so the event stream stays a
// pure function of the seed.
func (c *Cluster) AddNode(offset time.Duration) int {
	i := len(c.nodes)
	if i >= math.MaxUint16 {
		panic(fmt.Sprintf("simnet: Cluster.AddNode: node id %d overflows uint16", i+1))
	}
	nodeID := uint16(i + 1)
	c.offsets = append(c.offsets, offset)
	c.nodes = append(c.nodes, clusterNode{})
	c.nodes[i] = c.newClusterNode(i, nodeID, simRunID(nodeID))
	c.net.Register(nodeID, c.nodes[i].runID, c.nodes[i].handle, func(in port.InboundDatagram) {
		c.deliver(i, in)
	})
	c.sched.Schedule(c.sched.Now(), func() { c.drive(i, c.sched.Now()) })
	return i
}

// checkIndex fails loudly on an out-of-range node index — always a
// driving test's indexing mistake, never a protocol condition.
func (c *Cluster) checkIndex(i int, op string) {
	if i < 0 || i >= len(c.nodes) {
		panic(fmt.Sprintf("simnet: Cluster.%s: node index %d out of range (0..%d)", op, i, len(c.nodes)-1))
	}
}

// deliver hands one inbound datagram to node i's core at the current virtual
// time and drives the node afterwards.
func (c *Cluster) deliver(i int, in port.InboundDatagram) {
	now := c.sched.Now()
	var out port.Outbox
	c.nodes[i].core.HandleDatagram(now, in, &out)
	c.route(i, out.Drain())
	c.drive(i, now)
}

// drive runs node i's Advance loop after any core handler call: every
// emission of the loop routes through the network, and the loop repeats
// Advance until the returned deadline is strictly future — only then is it
// scheduled as the node's next timer event. The loop is the trap: a deadline
// at or before now must fire inside this same call (the next Advance
// iteration does), never be re-scheduled as an event whose time already
// passed — mirroring the production adapter's event loop without its
// goroutines (D-04). The cores' strictly-future deadline contract makes the
// loop exit on its first iteration; the repetition is the harness refusing
// to depend on that invariant.
//
// The scheduled timer carries a generation counter (the node's timerGen at
// scheduling): every drive call supersedes the node's previously scheduled
// timer, so exactly one live timer exists per node and a superseded timer's
// fire is a no-op. Without it every event would arm another timer at the
// same deadline and the outstanding-timer count would grow linearly with
// virtual time — quadratically many no-op Advance calls over a long run.
// Semantics are unchanged: a superseded timer's deadline is never earlier
// than the live one's, so no due wakeup is ever missed.
func (c *Cluster) drive(i int, now time.Time) {
	for {
		var out port.Outbox
		deadline := c.nodes[i].core.Advance(now, &out)
		c.route(i, out.Drain())
		if !deadline.After(now) {
			continue
		}
		c.nodes[i].timerGen++
		gen := c.nodes[i].timerGen
		c.sched.Schedule(deadline, func() {
			if c.nodes[i].timerGen != gen {
				return
			}
			c.drive(i, c.sched.Now())
		})
		return
	}
}

// route sends every outbound emission of node i through the virtual network:
// broadcasts fan out, unicasts address their handle.
func (c *Cluster) route(i int, outbound []port.Outbound) {
	for _, ob := range outbound {
		if ob.Broadcast {
			c.net.Broadcast(c.nodes[i].handle, ob.Msg)
			continue
		}
		c.net.Send(c.nodes[i].handle, ob.To, ob.Msg)
	}
}

// Write performs the production author sequence on node i at the scheduler's
// current virtual time (D-05): the store Set, then the core NotifyLocalWrite
// carrying the written record, capturing the (key, record) pair into the
// write log. Call RunUntil first to place the write in virtual time.
//
// A rejected Set — an oversize value, the ErrValueTooLarge class — is a
// driving test's mistake, not a protocol condition: it panics naming the
// operation, key and node, because a silently dropped write would make every
// model comparison against the write log lie (02-REVIEW WR-02). An
// out-of-range node index is the same class of mistake and panics through
// checkIndex.
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
	c.checkIndex(i, "Write")
	now := c.sched.Now()
	_, _, err := c.nodes[i].store.Set(key, data)
	if err != nil {
		panic(fmt.Sprintf("simnet: Cluster.Write: Set %q on node %d: %v", key, i, err))
	}
	rec, ok := c.nodes[i].store.Lookup(key)
	if !ok {
		panic(fmt.Sprintf("simnet: Cluster.Write: Lookup %q on node %d after a successful Set: record missing", key, i))
	}
	var out port.Outbox
	c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
	c.route(i, out.Drain())
	c.drive(i, now)
	c.writes = append(c.writes, LoggedWrite{Key: key, Rec: rec})
	return rec
}

// Delete performs the production author sequence for a deletion on node i at
// the scheduler's current virtual time (D-05): the store Delete writes a
// fresh-version tombstone that merges like any write, the core
// NotifyLocalWrite carries the tombstone record through the same
// pending/wait-set path a write takes (Deleted=true crosses the wire per
// message), and the (key, tombstone) pair joins the write log — Rec.Deleted
// true, the pointwise-supremum model needs no separate flag. A missing or
// already-tombstoned key is storage.ErrNotFound and changes nothing: a
// no-op delete writes nothing and burns no version (the store.Delete
// contract). Call RunUntil first to place the delete in virtual time. The
// written tombstone is read back with the store's point Lookup — never a
// whole-store Records() copy — and an out-of-range node index panics through
// checkIndex, the same harness convention as Write.
//
// Determinism contract: as Write — the delete runs at the current virtual
// time and every emission routes through the network's seeded rng; the seam
// adds no randomness source, so the event stream stays a pure function of
// the seed.
func (c *Cluster) Delete(i int, key string) (storage.Record, error) {
	c.checkIndex(i, "Delete")
	now := c.sched.Now()
	_, _, err := c.nodes[i].store.Delete(key)
	if err != nil {
		return storage.Record{}, err
	}
	rec, ok := c.nodes[i].store.Lookup(key)
	if !ok {
		panic(fmt.Sprintf("simnet: Cluster.Delete: Lookup %q on node %d after a successful delete: record missing", key, i))
	}
	var out port.Outbox
	c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
	c.route(i, out.Drain())
	c.drive(i, now)
	c.writes = append(c.writes, LoggedWrite{Key: key, Rec: rec})
	return rec, nil
}

// Nodes returns the cluster size.
func (c *Cluster) Nodes() int { return len(c.nodes) }

// Store returns node i's storage store.
func (c *Cluster) Store(i int) *storage.Store { return c.nodes[i].store }

// Core returns node i's replication core — the quiescence and discard seams
// (UnackedStats, QueuedResponses, Discards) read through it.
func (c *Cluster) Core(i int) *repl.Core { return c.nodes[i].core }

// Network returns the virtual network — the FaultStats, PendingDeliveries
// and TraceDigest seams.
func (c *Cluster) Network() *Network { return c.net }

// Handle returns node i's network handle.
func (c *Cluster) Handle(i int) string { return c.nodes[i].handle }

// WriteLog returns every captured write in write order.
func (c *Cluster) WriteLog() []LoggedWrite { return c.writes }
