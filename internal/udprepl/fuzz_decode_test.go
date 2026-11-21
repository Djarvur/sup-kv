package udprepl_test

// The decoder fuzz target (TEST-08, D-08): FuzzDecode feeds arbitrary bytes
// through the adapter's decode() path — the same verdict machinery the
// corrupted-datagram negative table (decode_negative_test.go, plan 02-07)
// pins row by row — and asserts invariants only:
//
//   - decode() never panics on any input (a panic is a fuzz failure);
//   - a Decoded verdict never carries a nil Msg;
//   - BadCRC and Undecodable verdicts carry no Msg at all;
//   - the verdict always lies inside the fixed vocabulary
//     Decoded/BadCRC/Undecodable.
//
// It deliberately does NOT assert that arbitrary garbage is always rejected:
// a body can parse and decode (the negative table's XPORT-10 rows prove two
// such classes today), so "never Decoded for garbage" is not an invariant.
//
// The committed seed corpus under testdata/fuzz/FuzzDecode/ mirrors the
// negative table's corruption classes by definition — bad CRC, truncated
// envelope, truncated body, wrong proto_version with a correct CRC, bad
// run_id — never by importing that test file. Plain `go test` loads and runs
// the corpus (the PR-CI smoke, where a fuzz target runs as an ordinary test
// over its seeds); deep coverage-guided runs happen on the nightly schedule
// (plan 02-11) and always bound the run with -fuzztime (RESEARCH Pitfall 4).
// One fuzz target per package: FuzzDumpDecode lives in internal/app/persist.

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/udprepl"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// The fuzz seed identity and fixture constants: a wire node id, one run id,
// one version base, and the integer that keeps hand-added seeds from sharing
// a version.
const (
	fuzzDecodeNodeID  = uint16(9)
	fuzzDecodeNowMs   = int64(1_730_000_000_000)
	fuzzDecodeCounter = uint16(1)
	fuzzDecodeKey     = "fuzz/decode"
	fuzzDecodeLongCRC = 0xFFFFFF // forces the trailing checksum varint to four bytes — the truncation point
)

// fuzzDecodeRunID is the seed corpus' wire run id.
var fuzzDecodeRunID = [16]byte{0xF9, 0x02, 0x17, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}

// errFuzzTruncatedBodyParses is the truncated-body construction guard: the
// fixture is only a valid class representative while the cut bytes fail to
// unmarshal (a wrapped static error, so the guard reads as one sentinel).
var errFuzzTruncatedBodyParses = errors.New("fuzz fixture bug: the truncated body still unmarshals")

// FuzzDecode is the decoder's native fuzz target: the seed corpus covers
// every valid message type plus one representative of each corruption class
// the negative table pins, and the body asserts the verdict invariants only.
func FuzzDecode(f *testing.F) {
	seeds, err := fuzzDecodeSeeds(fuzzDecodeRunID)
	if err != nil {
		f.Fatalf("build the decoder seed corpus: %v", err)
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, datagram []byte) {
		in := udprepl.DecodeDatagramForTest(datagram, "fuzz-source")
		switch in.Status {
		case port.Decoded:
			if in.Msg == nil {
				t.Fatalf("Decoded verdict carries a nil Msg for %d bytes", len(datagram))
			}
		case port.BadCRC:
			if in.Msg != nil {
				t.Fatalf("BadCRC verdict carries a decoded Msg %T — the body must not be unmarshaled", in.Msg)
			}
		case port.Undecodable:
			if in.Msg != nil {
				t.Fatalf("Undecodable verdict carries a decoded Msg %T", in.Msg)
			}
		default:
			t.Fatalf("decode verdict %d outside the fixed vocabulary Decoded/BadCRC/Undecodable", in.Status)
		}
	})
}

// fuzzDecodeSeeds builds the seed payloads: each valid message type encoded
// through the production encoder, the empty datagram, and one representative
// of every corruption class from the negative table's definitions. The
// construction guards fail seed time loudly, so a fixture that stopped
// representing its class cannot silently weaken the corpus. It is a plain
// function rather than a *testing.F method so the same payloads drive both
// the fuzz target and the committed testdata corpus files.
func fuzzDecodeSeeds(runID [16]byte) ([][]byte, error) {
	ver := version.PackVersion(fuzzDecodeNowMs, fuzzDecodeCounter, fuzzDecodeNodeID)
	valid := []port.Message{
		port.Hello{},
		port.DataReplication{
			Key: fuzzDecodeKey,
			Record: storage.Record{
				Version:  ver,
				Data:     []byte("value"),
				Checksum: storage.RecordChecksum(fuzzDecodeKey, ver, []byte("value"), false),
			},
		},
		port.DataReplication{
			Key: fuzzDecodeKey,
			Record: storage.Record{
				Version:  ver,
				Deleted:  true,
				Checksum: storage.RecordChecksum(fuzzDecodeKey, ver, nil, true),
			},
		},
		port.Ack{Key: fuzzDecodeKey, Version: ver},
		port.DataRequest{Key: fuzzDecodeKey},
		port.DataSync{Entries: []port.SyncEntry{{Key: "fuzz/a", Version: ver}, {Key: "fuzz/b", Version: ver}}},
	}
	seeds := make([][]byte, 0, len(valid)+6)
	for _, msg := range valid {
		datagram, err := udprepl.EncodeDatagramForTest(msg, fuzzDecodeNodeID, runID)
		if err != nil {
			return nil, fmt.Errorf("encode %T seed: %w", msg, err)
		}
		seeds = append(seeds, datagram)
	}

	// The empty datagram: the Undecodable floor.
	seeds = append(seeds, []byte{})

	// Bad CRC: one corrupted body byte. The body is the envelope's last
	// field, so the final byte lies inside the CRC-covered region — the
	// envelope still parses and the CRC no longer covers the body.
	badCRC := bytes.Clone(seeds[1])
	badCRC[len(badCRC)-1] ^= 0xFF
	seeds = append(seeds, badCRC)

	// Truncated envelope: the first half of a well-formed datagram.
	truncatedEnvelope := bytes.Clone(seeds[1][:len(seeds[1])/2])
	seeds = append(seeds, truncatedEnvelope)

	// Truncated body: a CRC-passing envelope whose body is cut inside a
	// length-delimited field, so the envelope parses and the body unmarshal
	// is the failing branch.
	truncatedBody, err := fuzzDecodeTruncatedBody(runID)
	if err != nil {
		return nil, err
	}
	seeds = append(seeds, truncatedBody)

	// Wrong proto_version with a correct CRC: the production encoder always
	// stamps SupportedProtoVersion, so only the crafting seam can build it.
	versionMismatch, err := udprepl.EncodeDatagramForTestVersion(
		port.Hello{}, fuzzDecodeNodeID, runID, repl.SupportedProtoVersion+1)
	if err != nil {
		return nil, fmt.Errorf("encode version-mismatch seed: %w", err)
	}
	seeds = append(seeds, versionMismatch)

	// Bad run_id: not the 16-byte RFC 9562 domain, wrapped in a CRC-correct
	// envelope (the production encoder refuses a short run id by type).
	helloBody, err := fuzzDecodeHelloBody()
	if err != nil {
		return nil, err
	}
	badRunID, err := fuzzDecodeEnvelope(repl.SupportedProtoVersion, []byte{1, 2, 3}, helloBody)
	if err != nil {
		return nil, err
	}
	seeds = append(seeds, badRunID)

	return seeds, nil
}

// fuzzDecodeTruncatedBody builds a real marshaled Body with its trailing
// checksum varint cut short — the checksum 0xFFFFFF forces a four-byte
// varint, so one byte removed truncates the field — wrapped in an envelope
// whose CRC covers the truncated bytes. A construction guard proves the
// truncated bytes still fail to unmarshal, so the fixture cannot decay into
// a merely-valid datagram.
func fuzzDecodeTruncatedBody(runID [16]byte) ([]byte, error) {
	ver := version.PackVersion(fuzzDecodeNowMs, fuzzDecodeCounter+1, fuzzDecodeNodeID)
	key := "fuzz/truncated-body"
	body, err := proto.Marshal(&replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
		Key:      key,
		Version:  ver[:],
		Data:     bytes.Repeat([]byte("b"), 32),
		Checksum: fuzzDecodeLongCRC,
	}}})
	if err != nil {
		return nil, fmt.Errorf("marshal truncated-body fixture: %w", err)
	}
	truncated := body[:len(body)-1]
	var probe replproto.Body
	err = proto.Unmarshal(truncated, &probe)
	if err == nil {
		return nil, fmt.Errorf("%w (%d of %d bytes)", errFuzzTruncatedBodyParses, len(truncated), len(body))
	}
	return fuzzDecodeEnvelope(repl.SupportedProtoVersion, runID[:], truncated)
}

// fuzzDecodeHelloBody marshals the Hello body for the hand-built envelope.
func fuzzDecodeHelloBody() ([]byte, error) {
	body, err := proto.Marshal(&replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}})
	if err != nil {
		return nil, fmt.Errorf("marshal Hello body: %w", err)
	}
	return body, nil
}

// fuzzDecodeEnvelope hand-builds one envelope whose CRC covers body exactly
// — the adversarial seam for the domain rows the production encoder cannot
// construct.
func fuzzDecodeEnvelope(protoVersion uint32, runID, body []byte) ([]byte, error) {
	envelope := &replproto.Envelope{
		ProtoVersion: protoVersion,
		NodeId:       uint32(fuzzDecodeNodeID),
		RunId:        runID,
		Crc32:        crc32.ChecksumIEEE(body),
		Body:         body,
	}
	datagram, err := proto.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal hand-built envelope: %w", err)
	}
	return datagram, nil
}
