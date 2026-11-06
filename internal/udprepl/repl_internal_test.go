package udprepl

// White-box on purpose: the wire encode/decode paths stay unexported — the
// adapter's mechanical halves (D-01/D-02) — so the external tracer test
// reaches them through these exported test-binary-only helpers (the
// dumpfile_internal_test.go precedent for the testpackage exemption).

import (
	"github.com/Djarvur/sup-kv/internal/port"
)

// EncodeDatagramForTest encodes msg as a full wire datagram from nodeID's
// runID through the adapter's encode path verbatim.
func EncodeDatagramForTest(msg port.Message, nodeID uint16, runID [16]byte) ([]byte, error) {
	return encodeDatagram(nodeID, runID, msg)
}

// DecodeDatagramForTest decodes a raw datagram through the adapter's decode
// path verbatim, with the sender's source handle attached the way the read
// goroutine attaches it.
func DecodeDatagramForTest(datagram []byte, source string) port.InboundDatagram {
	in := decode(datagram)
	in.Source = source
	return in
}
