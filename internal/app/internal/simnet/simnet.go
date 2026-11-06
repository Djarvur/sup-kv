// Package simnet is the deterministic simulator harness behind plan 04-05's
// convergence proofs: a discrete-event scheduler ordered by (Time, Seq), a
// fault-injecting virtual network, and an N-node cluster harness driving the
// same repl.Core handlers the production UDP adapter drives (no simulation
// mode anywhere — the determinism argument IS that the same handlers run).
// It is a normal package linked only into test binaries (the testutil
// precedent); unlike testutil its directory name does not match the
// no-test-libs-in-prod glob, so the non-test files here must not import Go's
// testing package — assertions live only in _test.go files.
//
// RED-phase state: the Scheduler and Network below are compile stubs whose
// behavior is unimplemented — scheduled events are discarded and nothing is
// ever delivered — so the simnet_test.go suite fails on its assertions; the
// GREEN commit implements them. The Cluster harness is already the real
// shape (real storage.Store nodes, real repl.Core instances) because the
// failing tests build whole clusters as their fixtures.
package simnet

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// clusterDriftWarn is the store clocks' drift tolerance inside the simulator
// (the docs/IDEA.md design default; the sim never reads the drift signal).
const clusterDriftWarn = time.Minute

// Scheduler is the discrete-event virtual clock: events execute in
// (Time, Seq) total order. RED stub — Schedule discards its event and
// RunUntil only advances the clock, so equal-time ordering is unobservable
// until the GREEN implementation lands.
type Scheduler struct{ now time.Time }

// NewScheduler returns a scheduler whose virtual clock starts at start.
func NewScheduler(start time.Time) *Scheduler {
	return &Scheduler{now: start}
}

// Schedule enqueues fn at virtual time t. RED stub: the event is discarded.
func (*Scheduler) Schedule(time.Time, func()) {}

// RunUntil executes every event at or before t. RED stub: only Now advances.
func (s *Scheduler) RunUntil(t time.Time) { s.now = t }

// Now returns the current virtual time.
func (s *Scheduler) Now() time.Time { return s.now }

// Policy is the fault-injection profile, pure data injected by each test:
// drop and duplicate probabilities, the one-way delay draw and the partition
// predicate. simnet draws randomness only through the injected rng, so the
// same seed reproduces every fate decision (D-06/D-07).
type Policy struct {
	DropProb  float64
	DupProb   float64
	Delay     func(rng *rand.Rand) time.Duration
	Partition func(from, to string, now time.Time) bool
}

// FaultStats counts the fault injector's observed classes.
type FaultStats struct {
	Dropped     uint64
	Duplicated  uint64
	Reordered   uint64
	Partitioned uint64
}

// Network is the fault-injecting virtual datagram network. RED stub — every
// send is silently discarded, every counter stays zero, the trace stays
// empty.
type Network struct{}

// NewNetwork returns the network over sched, rng and policy. RED stub.
func NewNetwork(*Scheduler, *rand.Rand, Policy) *Network { return &Network{} }

// Register makes (nodeID, runID, handle) a reachable delivery target. RED
// stub: registrations are discarded.
func (*Network) Register(uint16, [16]byte, string, func(port.InboundDatagram)) {}

// Send injects one datagram between two handles. RED stub: never delivered.
func (*Network) Send(string, string, port.Message) {}

// Broadcast fans one datagram out to every registered node including the
// sender. RED stub: never delivered.
func (*Network) Broadcast(string, port.Message) {}

// FaultStats returns the fault counters. RED stub: all zero.
func (*Network) FaultStats() FaultStats { return FaultStats{} }

// PendingDeliveries returns scheduled-but-undelivered network events. RED
// stub: always zero.
func (*Network) PendingDeliveries() int { return 0 }

// TraceDigest returns the ordered delivery trace. RED stub: always empty.
func (*Network) TraceDigest() string { return "" }

// LoggedWrite is one captured cluster write — the model input the caller's
// pointwise-suprememum store merges over (merge order is irrelevant,
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
// its fault profile.
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
// goroutines (D-04).
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
