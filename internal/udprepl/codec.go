package udprepl

import (
	"errors"
	"fmt"
	"hash/crc32"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// maxNodeIDWire is the highest node_id the 16-bit NodeID domain admits;
// an envelope beyond it is schema-malformed and classifies Undecodable.
const maxNodeIDWire = 0xFFFF

// wireIDSize is the wire size of both the run_id and the RFC 9562
// RecordVersion fields: exactly 16 bytes each.
const wireIDSize = 16

// errNoStandaloneBody rejects a port message that has no datagram body of
// its own on the wire schema.
var errNoStandaloneBody = errors.New("udprepl: message has no standalone datagram body")

// encodeDatagram frames msg as one full wire datagram from nodeID's runID
// (D-01): the port struct maps onto the proto Body oneof, whose marshaled
// bytes travel as the Envelope body under a CRC32 — corruption detection
// only, never tamper protection, per the accepted trust model.
func encodeDatagram(nodeID uint16, runID [16]byte, msg port.Message) ([]byte, error) {
	body, err := protoBody(msg)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("udprepl: marshal body: %w", err)
	}
	env := &replproto.Envelope{
		ProtoVersion: repl.SupportedProtoVersion,
		NodeId:       uint32(nodeID),
		RunId:        runID[:],
		Crc32:        crc32.ChecksumIEEE(bodyBytes),
		Body:         bodyBytes,
	}
	datagram, err := proto.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("udprepl: marshal envelope: %w", err)
	}
	return datagram, nil
}

// decode maps one raw datagram onto the port seam, mechanically per D-02:
// an empty datagram or an envelope that fails to unmarshal classifies
// Undecodable; envelope fields violating their wire domain — a node_id
// beyond the 16-bit range, a run_id that is not 16 bytes — classify
// Undecodable too; a proto_version other than the one this build speaks
// classifies ProtoVersionMismatch with the envelope fields carried over and
// NO body decode — the verdict precedes the CRC gate because a foreign
// version's body need not follow this version's framing contract (D-07); a
// CRC mismatch classifies BadCRC with the envelope fields carried over and
// the body NOT unmarshaled; only a passing CRC is the body decoded, and any
// body fault classifies Undecodable.
func decode(datagram []byte) port.InboundDatagram {
	var env replproto.Envelope
	if len(datagram) == 0 || proto.Unmarshal(datagram, &env) != nil {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	if env.GetNodeId() > maxNodeIDWire || len(env.GetRunId()) != wireIDSize {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	in := port.InboundDatagram{
		ProtoVersion: env.GetProtoVersion(),
		NodeID:       uint16(env.GetNodeId()), //nolint:gosec // G115: the maxNodeIDWire check above proved the value fits 16 bits.
	}
	copy(in.RunID[:], env.GetRunId())
	if in.ProtoVersion != repl.SupportedProtoVersion {
		in.Status = port.ProtoVersionMismatch
		return in
	}
	if crc32.ChecksumIEEE(env.GetBody()) != env.GetCrc32() {
		in.Status = port.BadCRC
		return in
	}
	var body replproto.Body
	err := proto.Unmarshal(env.GetBody(), &body)
	if err != nil {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	msg, ok := portMessage(&body)
	if !ok {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	in.Status = port.Decoded
	in.Msg = msg
	return in
}

// protoBody maps one port message onto its wire Body oneof — constructed
// via the generated wrapper types (RESEARCH Pitfall 6). A standalone
// SyncEntry has no datagram body of its own — it only travels inside
// DataSync — so it is rejected here.
func protoBody(msg port.Message) (*replproto.Body, error) {
	switch m := msg.(type) {
	case port.Hello:
		return &replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}}, nil
	case port.DataReplication:
		return &replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
			Key:      m.Key,
			Version:  m.Record.Version[:],
			Data:     m.Record.Data,
			Deleted:  m.Record.Deleted,
			Checksum: m.Record.Checksum,
		}}}, nil
	case port.Ack:
		return &replproto.Body{Msg: &replproto.Body_Ack{Ack: &replproto.Ack{Key: m.Key, Version: m.Version[:]}}}, nil
	case port.DataRequest:
		return &replproto.Body{Msg: &replproto.Body_DataRequest{DataRequest: &replproto.DataRequest{Key: m.Key}}}, nil
	case port.DataSync:
		entries := make([]*replproto.SyncEntry, len(m.Entries))
		for i, entry := range m.Entries {
			entries[i] = &replproto.SyncEntry{Key: entry.Key, Version: entry.Version[:]}
		}
		return &replproto.Body{Msg: &replproto.Body_DataSync{DataSync: &replproto.DataSync{
			SyncId:     m.SyncID,
			ChunkIndex: m.ChunkIndex,
			ChunkCount: m.ChunkCount,
			Entries:    entries,
		}}}, nil
	default:
		return nil, fmt.Errorf("%w: %T", errNoStandaloneBody, msg)
	}
}

// portMessage maps a decoded wire Body onto its port message struct (D-01),
// reporting false when a field violates its wire domain — a version that is
// not exactly 16 bytes — which the caller classifies Undecodable.
func portMessage(body *replproto.Body) (port.Message, bool) {
	switch msg := body.GetMsg().(type) {
	case *replproto.Body_Hello:
		return port.Hello{}, true
	case *replproto.Body_DataReplication:
		dr := msg.DataReplication
		ver, ok := recordVersion(dr.GetVersion())
		if !ok {
			return nil, false
		}
		return port.DataReplication{
			Key: dr.GetKey(),
			Record: storage.Record{
				Version:  ver,
				Data:     dr.GetData(),
				Deleted:  dr.GetDeleted(),
				Checksum: dr.GetChecksum(),
			},
		}, true
	case *replproto.Body_Ack:
		ver, ok := recordVersion(msg.Ack.GetVersion())
		if !ok {
			return nil, false
		}
		return port.Ack{Key: msg.Ack.GetKey(), Version: ver}, true
	case *replproto.Body_DataRequest:
		return port.DataRequest{Key: msg.DataRequest.GetKey()}, true
	case *replproto.Body_DataSync:
		wire := msg.DataSync.GetEntries()
		entries := make([]port.SyncEntry, len(wire))
		for i, entry := range wire {
			ver, ok := recordVersion(entry.GetVersion())
			if !ok {
				return nil, false
			}
			entries[i] = port.SyncEntry{Key: entry.GetKey(), Version: ver}
		}
		return port.DataSync{
			SyncID:     msg.DataSync.GetSyncId(),
			ChunkIndex: msg.DataSync.GetChunkIndex(),
			ChunkCount: msg.DataSync.GetChunkCount(),
			Entries:    entries,
		}, true
	default:
		return nil, false
	}
}

// recordVersion converts one wire version field into a RecordVersion,
// reporting false unless it is exactly 16 bytes — RFC 9562's UUID size, the
// full record-version domain.
func recordVersion(b []byte) (version.RecordVersion, bool) {
	var ver version.RecordVersion
	if len(b) != len(ver) {
		return version.RecordVersion{}, false
	}
	copy(ver[:], b)
	return ver, true
}
