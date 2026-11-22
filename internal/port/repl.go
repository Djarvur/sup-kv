package port

import (
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// DecodeStatus is the UDP adapter's mechanical verdict over one inbound
// datagram's framing, computed before any protocol decision (D-02): the
// core classifies and counts every discard class itself, consuming this
// verdict as an input, never re-deriving it.
type DecodeStatus int8

// The four decode outcomes of one inbound datagram.
const (
	// Decoded means the envelope and body both unmarshaled and passed the
	// CRC gate; Msg carries the decoded message.
	Decoded DecodeStatus = iota
	// BadCRC means the envelope unmarshaled but its CRC32 does not cover
	// the body bytes; the adapter did NOT decode the body (D-02).
	BadCRC
	// Undecodable means the datagram is not a well-formed envelope (or
	// body) at all — including schema-domain violations such as a run_id
	// that is not 16 bytes.
	Undecodable
	// ProtoVersionMismatch means the envelope unmarshaled but its
	// proto_version differs from the protocol version this build speaks.
	// The verdict is decided after the envelope fields are read and BEFORE
	// the CRC gate and body decode: a foreign version's body need not
	// follow this version's framing contract, so it is never decoded. Msg
	// is nil; ProtoVersion, NodeID and RunID carry the parsed envelope
	// fields so the core can attribute and rate-limit the discard (D-07).
	ProtoVersionMismatch
)

// Message is the sealed interface of the replication message structs. The
// port seam carries these decoded structs, not bytes (D-01): the UDP
// adapter (internal/udprepl) decodes the protobuf datagram into them for
// the core (internal/app/repl) and encodes them back on the send path,
// because protobuf cannot cross into internal/app (the
// sup-kv-app-no-io depguard rule). The unexported method seals the set —
// exactly the five protocol messages plus the DataSync entry below.
type Message interface {
	isMessage()
}

// Hello is the discovery announcement: broadcast every hello_interval, with
// node_id and run_id traveling as envelope fields, so the body is empty
// (REPL-01). A new run_id from a known node means it restarted (REPL-07).
type Hello struct{}

// DataReplication carries one full record for Key: broadcast by the write's
// author (D-05) and on retransmission, unicast as a DataRequest reply
// (REPL-02). The record crosses as the storage struct — the core and the
// store share record shape, the adapter only re-encodes it (D-01).
type DataReplication struct {
	Key    string
	Record storage.Record
}

// Ack confirms one (Key, Version) pair: unicast by the receiver immediately
// after the record is applied to memory — nothing in the Ack path waits for
// a dump (REPL-03).
type Ack struct {
	Key     string
	Version version.RecordVersion
}

// DataRequest asks the addressed peer for the record under Key; a peer
// without the key does not reply (REPL-04).
type DataRequest struct {
	Key string
}

// SyncEntry is one key→version pair inside a DataSync chunk (REPL-05). It
// implements Message for uniform seam typing but never travels alone: a
// datagram body is always one of the five protocol messages.
type SyncEntry struct {
	Key     string
	Version version.RecordVersion
}

// DataSync is one chunk of the sorted key→version listing exchanged for
// full reconciliation (REPL-05). The header makes completeness a wire fact:
// every chunk of one listing round carries the same SyncID and ChunkCount,
// and its own 1-based ChunkIndex, so the receiver accumulates parts and
// knows the listing is complete exactly when every index 1..ChunkCount has
// arrived — the first chunk alone is never the whole listing (PROTO-01).
type DataSync struct {
	// SyncID identifies the listing round. A fresh value is allocated per
	// emission round, so a reordered or retransmitted chunk from one round
	// can never be accumulated into another.
	SyncID uint64
	// ChunkIndex is this chunk's 1-based position within the round.
	ChunkIndex uint32
	// ChunkCount is the number of chunks in the round, at least 1: an empty
	// listing travels as one empty chunk with count 1.
	ChunkCount uint32
	// Entries is this chunk's share of the sorted listing.
	Entries []SyncEntry
}

// The sealed Message set.
func (Hello) isMessage()           {}
func (DataReplication) isMessage() {}
func (Ack) isMessage()             {}
func (DataRequest) isMessage()     {}
func (SyncEntry) isMessage()       {}
func (DataSync) isMessage()        {}

// InboundDatagram is one datagram handed to the core with the adapter's
// mechanical decode verdict attached (D-01/D-02): Source is the sender's
// opaque address handle (the adapter's src.String(); the core never touches
// a [net.Addr], D-03), Status carries the framing verdict, and the envelope
// fields are filled whenever the envelope itself unmarshaled — including on
// BadCRC, so the counters can still attribute the discard. Msg is non-nil
// only when Status is Decoded.
type InboundDatagram struct {
	Source       string
	Status       DecodeStatus
	ProtoVersion uint32
	NodeID       uint16
	RunID        [16]byte
	Msg          Message
}

// Outbound is one message the core emitted: broadcast to the segment, or
// unicast to one peer's opaque address handle (D-03 — only the adapter can
// resolve a handle to a real address).
type Outbound struct {
	Broadcast bool
	To        string
	Msg       Message
}

// ResultKind classifies one replication result event — the core telling its
// driver what became of a tracked local write, without any callback
// crossing back into the pure core (D-04's results-not-callbacks rule).
type ResultKind int8

// The four ways a tracked write resolves.
const (
	// ResultAck reports that Peer acknowledged the tracked (Key, Version)
	// — including a peer that was never in the write's wait-set (the WR-01
	// counting behavior). The event is informational; the entry resolves on
	// ResultDrained, never on an Ack alone.
	ResultAck ResultKind = iota
	// ResultDrained reports that the tracked wait-set emptied while at
	// least one peer remains: every write-time waiter acknowledged the
	// tracked version, so the write is fully replicated.
	ResultDrained
	// ResultSuperseded reports that Version under Key was displaced by a
	// strictly greater version (a remote apply or a newer local write).
	// The tracked version's Acks did not all arrive: the resolution is an
	// honest false.
	ResultSuperseded
	// ResultWaitSetEmpty reports that peer eviction removed the tracked
	// entry's last waiter while the entry stays suspended. Eviction is not
	// an Ack: the resolution is an honest false.
	ResultWaitSetEmpty
)

// Result is one result event the core appended to an Outbox: which kind,
// the tracked write identity (Key, Version — always the tracked version,
// never a displacing one) and, for ResultAck only, the acknowledging peer.
// Plain data, like the rest of the port seam.
type Result struct {
	Kind    ResultKind
	Key     string
	Version version.RecordVersion
	Peer    uint16
}

// Outbox accumulates the messages a core handler emits during one call: the
// driver hands the same box to the handler, then drains and sends (or, in
// the simulator, fault-injects and schedules) every entry — results, not
// callbacks (D-04/D-05). Two independent streams share the box: the message
// stream behind [Outbox.Drain] and the result stream behind
// [Outbox.DrainResults] (the wait_for_replication seam, XPORT-12). A driver
// must drain BOTH after every handler, or the undrained stream grows for
// the process lifetime. Not safe for concurrent use: exactly one event-loop
// goroutine drives the core.
type Outbox struct {
	outbound []Outbound
	results  []Result
}

// Broadcast enqueues msg for broadcast to the whole segment.
func (o *Outbox) Broadcast(msg Message) {
	o.outbound = append(o.outbound, Outbound{Broadcast: true, Msg: msg})
}

// Unicast enqueues msg for handle — an opaque address handle resolved by
// the adapter alone (D-03).
func (o *Outbox) Unicast(handle string, msg Message) {
	o.outbound = append(o.outbound, Outbound{To: handle, Msg: msg})
}

// Drain returns every enqueued outbound message in emission order and
// resets the message stream, so the driver starts each event with an empty
// outbox. The result stream is untouched — the two streams are independent.
func (o *Outbox) Drain() []Outbound {
	out := o.outbound
	o.outbound = nil
	return out
}

// Ack records that peer acknowledged the tracked (key, ver) pair — the
// per-peer counting half of the wait_for_replication seam (XPORT-12). Keys
// and versions only: the adapter owns every waiter, the core stays free of
// channels and goroutines.
func (o *Outbox) Ack(key string, ver version.RecordVersion, peer uint16) {
	o.results = append(o.results, Result{Kind: ResultAck, Key: key, Version: ver, Peer: peer})
}

// Drained records that the tracked (key, ver) write's wait-set emptied with
// peers present: full write-time Ack coverage, the only success resolution.
func (o *Outbox) Drained(key string, ver version.RecordVersion) {
	o.results = append(o.results, Result{Kind: ResultDrained, Key: key, Version: ver})
}

// Superseded records that the tracked (key, ver) write was displaced by a
// strictly greater version — an honest false resolution.
func (o *Outbox) Superseded(key string, ver version.RecordVersion) {
	o.results = append(o.results, Result{Kind: ResultSuperseded, Key: key, Version: ver})
}

// WaitSetEmpty records that peer eviction removed the tracked (key, ver)
// write's last waiter while the entry stays suspended — another honest
// false: an eviction is never an Ack.
func (o *Outbox) WaitSetEmpty(key string, ver version.RecordVersion) {
	o.results = append(o.results, Result{Kind: ResultWaitSetEmpty, Key: key, Version: ver})
}

// DrainResults returns every enqueued result event in emission order and
// resets the result stream, so a driver can consume each event's resolution
// exactly once. The message stream is untouched — the two streams are
// independent.
func (o *Outbox) DrainResults() []Result {
	out := o.results
	o.results = nil
	return out
}
