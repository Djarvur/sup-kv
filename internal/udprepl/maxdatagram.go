package udprepl

// This file owns the datagram-ceiling oracle (XPORT-03): the one honest
// measurement of how large the largest record the API accepts becomes on
// the wire. Hand-written varint arithmetic drifts from the generated schema
// the moment a field is added or its type widens (T-3-30), so the size is
// always produced by marshaling the real messages.

import (
	"math"
	"strings"

	"google.golang.org/protobuf/proto"

	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// MaxDatagramBytes is the maximum IPv4 UDP payload — 65 535 bytes less the
// 20-byte IP header and the 8-byte UDP header — and therefore the hard
// ceiling one replication datagram must fit. The socket receive buffer
// (datagramBufSize) has the same value: a larger buffer could never be
// filled by a valid datagram.
const MaxDatagramBytes = 65507

// MaxRecordDatagramBytes returns the worst-case encoded size of one
// DataReplication datagram carrying a keyBytes-byte key and a valueBytes-byte
// value: both messages are built with every variable-width field at its
// maximum width and marshaled through the generated schema, and the envelope
// length is returned.
//
// The result is deliberately conservative — a real record with a shorter
// version prefix or a smaller checksum encodes shorter — and it never
// undercounts a record the API can accept: the key and value lengths are
// exact, the version and run ids are the fixed 16-byte wire width, and
// proto_version, node_id, crc32 and checksum all carry [math.MaxUint32]
// (five-byte varints). It is the only honest oracle for the 65 507-byte
// ceiling (MaxDatagramBytes); a zero-filled probe would undercount by the
// varint overcount and let an unsendable record through.
//
// The tombstone flag follows the legal record shape: a record carrying a
// value is never a tombstone (tombstone-with-data is a receive-side discard
// class), while an empty value may be either a real empty Set or a
// tombstone, so the empty-value probe sets the flag and covers both. Setting
// it on a valued record would overcount two bytes and reject the documented
// 64 408-byte boundary — key 1 024 + value 64 408 encodes to exactly
// 65 507 bytes.
//
// Negative widths are clamped to zero: they are not record sizes, and the
// helper stays total over its inputs rather than panicking at a startup
// check.
func MaxRecordDatagramBytes(keyBytes, valueBytes int) int {
	keyBytes = max(keyBytes, 0)
	valueBytes = max(valueBytes, 0)
	body := &replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
		Key:      strings.Repeat("k", keyBytes),
		Version:  make([]byte, wireIDSize),
		Data:     make([]byte, valueBytes),
		Deleted:  valueBytes == 0,
		Checksum: math.MaxUint32,
	}}}
	// Marshal on these generated messages cannot fail: every string is valid
	// UTF-8 and every field carries its own type. The blank assignments are
	// the total-helper contract, not swallowed runtime errors.
	bodyBytes, _ := proto.Marshal(body)
	envelope, _ := proto.Marshal(&replproto.Envelope{
		ProtoVersion: math.MaxUint32,
		NodeId:       math.MaxUint32,
		RunId:        make([]byte, wireIDSize),
		Crc32:        math.MaxUint32,
		Body:         bodyBytes,
	})
	return len(envelope)
}
