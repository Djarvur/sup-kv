package udprepl_test

// The datagram-ceiling oracle's proof (XPORT-03): the worst-case size the
// startup check uses must never undercount what the real codec produces,
// and every record the API accepts — key at most storage.MaxKeyBytes,
// value at most the largest value that still fits a maximal key — must
// encode into at most MaxDatagramBytes. The real datagrams are built
// through EncodeDatagramForTest, the adapter's production encode path.

import (
	"math"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// The sized facts the table pins: the IPv4 UDP ceiling, the single key
// constant, the design-default value limit, the largest value that fits a
// maximal key (1 024 + 64 408 + the envelope fills 65 507), one byte pair
// over it, and a key length the pre-upgrade review proposed and this phase
// rejected (it leaves no value headroom).
const (
	testDatagramCeiling                = 65507
	testMaxKeyBytes                    = 1024
	testDefaultValueBytes              = 61440
	testLargestValueWithMaxKey         = 64408
	testOverCeilingValueWithKey        = 64415
	testRejectedKeyBytes               = 4096
	testWireNodeID                     = uint16(1)
	testTombstoneChecksum       uint32 = math.MaxUint32
)

// testReplicationDatagram encodes one DataReplication carrying keyLen bytes
// of key and valueLen bytes of value through the production encode path.
// deleted and checksum are caller-controlled so the tests can compare the
// oracle against both the typical record shape and the maximal legal one.
func testReplicationDatagram(t *testing.T, keyLen, valueLen int, deleted bool, checksum uint32) []byte {
	t.Helper()
	datagram, err := udprepl.EncodeDatagramForTest(port.DataReplication{
		Key: strings.Repeat("k", keyLen),
		Record: storage.Record{
			Version:  version.RecordVersion{},
			Data:     make([]byte, valueLen),
			Deleted:  deleted,
			Checksum: checksum,
		},
	}, testWireNodeID, [16]byte{})
	if err != nil {
		t.Fatalf("encode key=%d value=%d: %v", keyLen, valueLen, err)
	}
	return datagram
}

// testMaximalLegalDeleted mirrors the oracle's record-shape rule: a valued
// record is never a tombstone (tombstone-with-data is a receive-side discard
// class); an empty value covers both an empty Set and a tombstone.
func testMaximalLegalDeleted(valueLen int) bool { return valueLen == 0 }

// TestMaxRecordDatagramBytes proves the oracle never undercounts and agrees
// with the research's measured ceiling: the boundary rows sit at or below
// 65 507, one byte of value past the largest fitting value crosses it, and
// the conservative size is never smaller than the real encoded datagram for
// either the typical or the maximal record shape.
func TestMaxRecordDatagramBytes(t *testing.T) {
	t.Parallel()

	rows := []struct {
		name       string
		keyBytes   int
		valueBytes int
		over       bool
	}{
		{"empty", 0, 0, false},
		{"max key, empty value", testMaxKeyBytes, 0, false},
		{"empty key, default value", 0, testDefaultValueBytes, false},
		{"max key, default value", testMaxKeyBytes, testDefaultValueBytes, false},
		{"max key, largest fitting value", testMaxKeyBytes, testLargestValueWithMaxKey, false},
		{"max key, one byte past the ceiling", testMaxKeyBytes, testOverCeilingValueWithKey, true},
		{"rejected key length, default value", testRejectedKeyBytes, testDefaultValueBytes, true},
	}
	for _, row := range rows {
		got := udprepl.MaxRecordDatagramBytes(row.keyBytes, row.valueBytes)
		over := got > testDatagramCeiling
		if over != row.over {
			t.Fatalf("%s: MaxRecordDatagramBytes(%d, %d) = %d, over ceiling = %t, want %t (ceiling %d)",
				row.name, row.keyBytes, row.valueBytes, got, over, row.over, testDatagramCeiling)
		}
		// The maximal legal shape (tombstone flag only on the empty-value
		// shape, maximum checksum) is the closest real encoding to the
		// oracle; the typical shape is what most records actually cost.
		// Neither may exceed the oracle.
		maximal := testReplicationDatagram(t, row.keyBytes, row.valueBytes, testMaximalLegalDeleted(row.valueBytes), testTombstoneChecksum)
		if got < len(maximal) {
			t.Fatalf("%s: oracle undercounts the maximal legal real datagram: %d < %d (key=%d value=%d)",
				row.name, got, len(maximal), row.keyBytes, row.valueBytes)
		}
		typical := testReplicationDatagram(t, row.keyBytes, row.valueBytes, false, 0)
		if got < len(typical) {
			t.Fatalf("%s: oracle undercounts the typical real datagram: %d < %d (key=%d value=%d)",
				row.name, got, len(typical), row.keyBytes, row.valueBytes)
		}
	}
}

// TestMaxRecordDatagramBytes_acceptedDomain walks the accepted domain — key
// lengths up to storage.MaxKeyBytes (including the multi-byte UTF-8
// boundary) and value lengths up to the largest value that still fits a
// maximal key — and asserts two properties for every combination: the real
// encoded datagram fits the 65 507-byte ceiling, and the conservative
// oracle both covers that real datagram and stays under the ceiling itself.
func TestMaxRecordDatagramBytes_acceptedDomain(t *testing.T) {
	t.Parallel()

	keyLens := []int{0, 1, 127, 128, 1023, testMaxKeyBytes}
	valueLens := []int{0, 1, 127, 128, 16383, 16384, 61439, testDefaultValueBytes, testLargestValueWithMaxKey - 1, testLargestValueWithMaxKey}
	for _, keyLen := range keyLens {
		for _, valueLen := range valueLens {
			encoded := testReplicationDatagram(t, keyLen, valueLen, testMaximalLegalDeleted(valueLen), testTombstoneChecksum)
			if len(encoded) > testDatagramCeiling {
				t.Fatalf("accepted record key=%d value=%d encodes to %d bytes, over the %d-byte ceiling",
					keyLen, valueLen, len(encoded), testDatagramCeiling)
			}
			oracle := udprepl.MaxRecordDatagramBytes(keyLen, valueLen)
			if oracle < len(encoded) {
				t.Fatalf("oracle undercounts key=%d value=%d: %d < %d", keyLen, valueLen, oracle, len(encoded))
			}
			if oracle > testDatagramCeiling {
				t.Fatalf("oracle rejects an accepted record key=%d value=%d: %d > %d",
					keyLen, valueLen, oracle, testDatagramCeiling)
			}
		}
	}

	// The boundary key is measured in bytes, not runes: 512 two-byte runes
	// are exactly storage.MaxKeyBytes bytes and must stay inside the domain.
	multiByteKey := strings.Repeat("é", testMaxKeyBytes/2)
	if len(multiByteKey) != testMaxKeyBytes {
		t.Fatalf("multi-byte boundary key is %d bytes, want %d", len(multiByteKey), testMaxKeyBytes)
	}
	datagram, err := udprepl.EncodeDatagramForTest(port.DataReplication{
		Key: multiByteKey,
		Record: storage.Record{
			Version:  version.RecordVersion{},
			Data:     make([]byte, testDefaultValueBytes),
			Deleted:  false,
			Checksum: testTombstoneChecksum,
		},
	}, testWireNodeID, [16]byte{})
	if err != nil {
		t.Fatalf("encode the multi-byte boundary record: %v", err)
	}
	if len(datagram) > testDatagramCeiling {
		t.Fatalf("the multi-byte boundary record encodes to %d bytes, over the %d-byte ceiling", len(datagram), testDatagramCeiling)
	}
	if oracle := udprepl.MaxRecordDatagramBytes(len(multiByteKey), testDefaultValueBytes); oracle < len(datagram) {
		t.Fatalf("oracle undercounts the multi-byte boundary record: %d < %d", oracle, len(datagram))
	}

	// A deterministic sample across the whole accepted rectangle: no
	// randomness needed to reach combinations the grid above skipped.
	for i := range 64 {
		keyLen := (i * 17) % (testMaxKeyBytes + 1)
		valueLen := (i * i * 331) % (testLargestValueWithMaxKey + 1)
		encoded := testReplicationDatagram(t, keyLen, valueLen, testMaximalLegalDeleted(valueLen), testTombstoneChecksum)
		if len(encoded) > testDatagramCeiling {
			t.Fatalf("sampled accepted record key=%d value=%d encodes to %d bytes, over the %d-byte ceiling",
				keyLen, valueLen, len(encoded), testDatagramCeiling)
		}
		if oracle := udprepl.MaxRecordDatagramBytes(keyLen, valueLen); oracle < len(encoded) {
			t.Fatalf("oracle undercounts sampled record key=%d value=%d: %d < %d", keyLen, valueLen, oracle, len(encoded))
		}
	}
}
