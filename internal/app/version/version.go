// Package version implements the sup-kv record version: a 16-byte RFC 9562
// UUIDv7 carrying the hybrid logical clock reading (unix_ts_ms, counter) and
// the writing node's ID. The big-endian layout guarantees that byte-wise
// comparison equals (l, counter, nodeID) order — the total order the sibling
// storage package's LWW join sorts by. The package depends on the standard
// library only.
package version

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"
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
	uuidVerBits = 0x70      // version 7 nibble in octet 6.
	uuidVarBits = 0x80      // variant 0b10 in the top two bits of octet 8.
	counterMask = 0xFFF     // the 12-bit counter kept from octets 6-7.
	nodeLsbMask = 0x3       // the nodeID bits relocated into octet 10.
	lPadding    = 16        // l occupies the low 48 bits of the first octet pair.
	tsMask      = 1<<48 - 1 // the 48-bit timestamp domain, proving l&tsMask non-negative.
	nodeLsbDrop = 2         // the nodeID's low 2 bits live in octet 10.
	fillerShift = 6         // octet 10 carries the nodeID tail in its top bits.
)

// Sentinel errors returned by [RecordVersion.Validate]. Callers match them
// with [errors.Is], never identity comparison; the transport core maps them
// onto its own wire discard classes (XPORT-04).
var (
	// ErrVersionLayout is returned when the 16 bytes are not an RFC 9562
	// UUIDv7: the version nibble in octet 6 is not 7, or the variant bits in
	// the top of octet 8 are not 0b10.
	ErrVersionLayout = errors.New("version is not an RFC 9562 UUIDv7")

	// ErrVersionFuture is returned when the unix_ts_ms reading lies beyond
	// now plus the caller's max_clock_lead: the clock-poisoning vector
	// Validate rejects before any clock call.
	ErrVersionFuture = errors.New("version reading is beyond now plus the maximum clock lead")
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
// never panics. The variant bits of octet 8 carry no explicit mask here: the
// nodeID's left shift truncates them out of the uint16.
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

// Validate reports whether v is an acceptable inbound record version at the
// wall-clock instant now: an RFC 9562 UUIDv7 whose unix_ts_ms reading lies at
// or before now+maxLead (XPORT-04; the inbound pipeline runs it before any
// store or clock touch, so a crafted reading can never poison the hybrid
// logical clock).
//
// The layout check accepts the version 7 nibble and the variant 0b10 bits and
// nothing else. The 46 rand_b filler bits RFC 9562 leaves to the
// implementation are deliberately NOT rejected: a conforming UUIDv7 minted by
// another vendor may set them, and rejecting them would make the protocol
// needlessly non-interoperable.
//
// A reading in the past is always accepted — the hybrid logical clock absorbs
// it on merge. The future bound is exclusive only beyond now+maxLead: a
// reading of exactly now+maxLead is accepted, one millisecond beyond is
// rejected with ErrVersionFuture. The design default for maxLead is 60 s, the
// documented drift tolerance, so peers inside it stay accepted; a negative
// maxLead normalizes to zero. Milliseconds are the only precision the layout
// carries, so a fractional now is truncated to its unix_ts_ms.
//
// Validate reads no clock (now is the caller's, the D-02 seam), allocates
// nothing and is total: every one of the 2^128 bit patterns yields a verdict,
// never a panic. The subtraction form of the future bound cannot overflow for
// a 48-bit reading and an absurd maxLead.
func (v RecordVersion) Validate(now time.Time, maxLead time.Duration) error {
	if v[6]&0xF0 != uuidVerBits || v[8]&0xC0 != uuidVarBits {
		return ErrVersionLayout
	}
	l, _, _ := v.Unpack()
	nowMs := now.UnixMilli()
	lead := max(maxLead.Milliseconds(), 0)
	if l >= nowMs && l-nowMs > lead {
		return ErrVersionFuture
	}
	return nil
}
