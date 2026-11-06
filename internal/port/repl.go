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

// The three decode outcomes of one inbound datagram.
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

// DataSync is one self-sufficient chunk of the sorted key→version listing
// exchanged for full reconciliation (REPL-05).
type DataSync struct {
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

// Outbox accumulates the messages a core handler emits during one call: the
// driver hands the same box to the handler, then drains and sends (or, in
// the simulator, fault-injects and schedules) every entry — results, not
// callbacks (D-04/D-05). Not safe for concurrent use: exactly one
// event-loop goroutine drives the core.
type Outbox struct {
	outbound []Outbound
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
// resets the box, so the driver starts each event with an empty outbox.
func (o *Outbox) Drain() []Outbound {
	out := o.outbound
	o.outbound = nil
	return out
}
