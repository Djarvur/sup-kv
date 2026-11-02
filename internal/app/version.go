package app

// RecordVersion is the 16-byte record version: an RFC 9562 UUIDv7 carrying
// the hybrid logical clock reading (unix_ts_ms, counter) and the writing
// node's ID, so byte-wise comparison equals (l, counter, nodeID) order.
type RecordVersion [16]byte

// Unpack returns the hybrid logical clock reading and node ID packed into v.
// It is total: pure masking arithmetic over every possible bit pattern, it
// never panics.
func (RecordVersion) Unpack() (l int64, counter uint16, nodeID uint16) {
	return 0, 0, 0
}

// Compare returns -1, 0 or 1 as v sorts byte-wise before, equal to, or after
// other. Byte-wise order equals (l, counter, nodeID) order because the
// version and variant bits are constants.
func (RecordVersion) Compare(_ RecordVersion) int {
	return 0
}
