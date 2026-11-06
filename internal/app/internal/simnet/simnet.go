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
	"math/rand/v2"
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
// drop and duplicate probabilities, the one-way delay draw and the partition
// predicate. A nil Delay means zero delay and a nil Partition means nothing
// is ever blocked (the fixed-delay profile of the bound tests and the
// loss-free default). simnet draws randomness only through the injected rng,
// at send time, in a fixed draw order — drop, then duplicate, then each
// delivery's delay — so the same seed reproduces every fate decision
// (D-06/D-07).
type Policy struct {
	DropProb  float64
	DupProb   float64
	Delay     func(rng *rand.Rand) time.Duration
	Partition func(from, to string, now time.Time) bool
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
// sibling of the same directed pair, and sends blocked by the partition
// predicate.
type FaultStats struct {
	Dropped     uint64
	Duplicated  uint64
	Reordered   uint64
	Partitioned uint64
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
// and therefore rng draw order — is deterministic.
type Network struct {
	sched      *Scheduler
	rng        *rand.Rand
	policy     Policy
	nodes      []netNode
	byHandle   map[string]int
	sendSeq    map[pairKey]uint64
	lastByPair map[pairKey]uint64
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
	}
}

// Register makes (nodeID, runID, handle) a reachable delivery target whose
// inbound datagrams invoke deliver.
func (n *Network) Register(nodeID uint16, runID [16]byte, handle string, deliver func(port.InboundDatagram)) {
	n.byHandle[handle] = len(n.nodes)
	n.nodes = append(n.nodes, netNode{id: nodeID, runID: runID, handle: handle, deliver: deliver})
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
// the recipient's delivery function receives the decoded datagram.
func (n *Network) scheduleDelivery(from, to netNode, msg port.Message, seq uint64, delay time.Duration) {
	pair := pairKey{from: from.handle, to: to.handle}
	n.pending++
	n.sched.Schedule(n.sched.Now().Add(delay), func() {
		n.pending--
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
// CORE-06).
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
// the scheduler's virtual time (the D-02 seam), and a real repl.Core.
type clusterNode struct {
	id     uint16
	runID  [16]byte
	handle string
	store  *storage.Store
	core   *repl.Core
}

// Cluster is the N-node harness: n repl.Core instances over n real stores,
// wired to one virtual network, with the write log capturing the production
// author sequence (D-05).
type Cluster struct {
	sched  *Scheduler
	net    *Network
	nodes  []clusterNode
	writes []LoggedWrite
}

// NewCluster builds n nodes — node_id 1..n, handles n1..nN in node_id order
// — over sched, each a real storage.Store whose clock now func returns the
// scheduler's current virtual time (the D-02 fake-clock idiom) and a
// repl.Core built with cfg; rng is the network's fault-draw source, policy
// its fault profile. Construction schedules each node's first timer pump at
// the scheduler's current time, so the first RunUntil fires every node's
// startup Hello and DataSync (the adapter's Run-startup mirror).
func NewCluster(sched *Scheduler, rng *rand.Rand, n int, cfg repl.Config, policy Policy) *Cluster {
	c := &Cluster{
		sched: sched,
		net:   NewNetwork(sched, rng, policy),
		nodes: make([]clusterNode, n),
	}
	for i := range c.nodes {
		nodeID := uint16(i + 1)
		handle := fmt.Sprintf("n%d", nodeID)
		store := storage.NewStore(nodeID, clock.NewClock(sched.Now, 0, 0, clusterDriftWarn), storage.Config{})
		c.nodes[i] = clusterNode{
			id:     nodeID,
			runID:  simRunID(nodeID),
			handle: handle,
			store:  store,
			core:   repl.New(nodeID, simRunID(nodeID), handle, nodeStore{store: store}, cfg),
		}
	}
	for i := range c.nodes {
		c.net.Register(c.nodes[i].id, c.nodes[i].runID, c.nodes[i].handle, func(in port.InboundDatagram) {
			c.deliver(i, in)
		})
		c.sched.Schedule(sched.Now(), func() { c.drive(i, sched.Now()) })
	}
	return c
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
func (c *Cluster) drive(i int, now time.Time) {
	for {
		var out port.Outbox
		deadline := c.nodes[i].core.Advance(now, &out)
		c.route(i, out.Drain())
		if deadline.After(now) {
			c.sched.Schedule(deadline, func() { c.drive(i, c.sched.Now()) })
			return
		}
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
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
	now := c.sched.Now()
	_, _, _ = c.nodes[i].store.Set(key, data)
	rec := c.nodes[i].store.Records()[key]
	var out port.Outbox
	c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
	c.route(i, out.Drain())
	c.drive(i, now)
	c.writes = append(c.writes, LoggedWrite{Key: key, Rec: rec})
	return rec
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
