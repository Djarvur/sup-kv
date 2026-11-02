package app

import (
	"bytes"
	"encoding/binary"
)

// RecordVersion is the 16-byte record version: an RFC 9562 UUIDv7 carrying
// the hybrid logical clock reading (unix_ts_ms, counter) and the writing
// node's ID, so byte-wise comparison equals (l, counter, nodeID) order.
type RecordVersion [16]byte

// RFC 9562 layout constants of RecordVersion: a 48-bit unix_ts_ms in octets
// 0-5, the version nibble and a 12-bit counter in rand_a (octets 6-7), the
// variant bits and the node ID in the high 16 bits of rand_b (octets 8-10);
// the remaining 46 rand_b bits are zero filler.
const (
	uuidVerBits    = 0x70      // version 7 nibble in octet 6.
	uuidVarBits    = 0x80      // variant 0b10 in the top two bits of octet 8.
	counterMask    = 0xFFF     // the 12-bit counter kept from octets 6-7.
	varPayloadMask = 0x3F      // the nodeID bits kept from octet 8.
	nodeLsbMask    = 0x3       // the nodeID bits relocated into octet 10.
	lPadding       = 16        // l occupies the low 48 bits of the first octet pair.
	tsMask         = 1<<48 - 1 // the 48-bit timestamp domain, proving l&tsMask non-negative.
	nodeLsbDrop    = 2         // the nodeID's low 2 bits live in octet 10.
	fillerShift    = 6         // octet 10 carries the nodeID tail in its top bits.
)

// PackVersion packs a hybrid logical clock reading and node ID into the RFC
// 9562 UUIDv7 layout. The caller must guarantee the ranges: l from zero to
// 2^48-1, counter at most 4095 and nodeID at most 0xFFFF; out-of-range
// inputs are not validated.
func PackVersion(l int64, counter uint16, nodeID uint16) RecordVersion {
	var v RecordVersion
	binary.BigEndian.PutUint64(v[0:8], uint64(l&tsMask)<<lPadding)
	binary.BigEndian.PutUint16(v[6:8], counter)
	v[6] |= uuidVerBits
	binary.BigEndian.PutUint16(v[8:10], nodeID>>nodeLsbDrop)
	v[8] |= uuidVarBits
	v[10] = byte((nodeID & nodeLsbMask) << fillerShift)
	return v
}

// Unpack returns the hybrid logical clock reading and node ID packed into v.
// It is total: pure masking arithmetic over every possible bit pattern, it
// never panics.
func (v RecordVersion) Unpack() (l int64, counter uint16, nodeID uint16) {
	l = int64(binary.BigEndian.Uint64(v[0:8]) >> lPadding)
	counter = binary.BigEndian.Uint16(v[6:8]) & counterMask
	nodeID = binary.BigEndian.Uint16(v[8:10])<<nodeLsbDrop | uint16(v[10]>>fillerShift)
	return l, counter, nodeID
}

// Compare returns -1, 0 or 1 as v sorts byte-wise before, equal to, or after
// other. Byte-wise order equals (l, counter, nodeID) order because the
// version and variant bits are constants.
func (v RecordVersion) Compare(other RecordVersion) int {
	return bytes.Compare(v[:], other[:])
}
