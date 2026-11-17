package udprepl

// White-box on purpose: the wire encode/decode paths stay unexported — the
// adapter's mechanical halves (D-01/D-02) — so the external tracer test
// reaches them through these exported test-binary-only helpers (the
// dumpfile_internal_test.go precedent for the testpackage exemption).

import (
	"fmt"
	"hash/crc32"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/port"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
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

// EncodeDatagramForTestVersion encodes msg as a full wire datagram with an
// explicit envelope proto_version — the adversarial crafting seam for the
// version-mismatch scenario (plan 06-06): the production encode always
// stamps repl.SupportedProtoVersion, so a mismatched datagram cannot be
// built through it. The body rides the production protoBody mapping and the
// CRC is computed exactly as the production encode computes it, so the
// receiver's CRC gate passes and the core's version check is the branch
// under test.
func EncodeDatagramForTestVersion(msg port.Message, nodeID uint16, runID [16]byte, protoVersion uint32) ([]byte, error) {
	body, err := protoBody(msg)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("udprepl: marshal body: %w", err)
	}
	env := &replproto.Envelope{
		ProtoVersion: protoVersion,
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
