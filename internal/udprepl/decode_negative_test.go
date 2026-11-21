package udprepl_test

// The decoder negative table (TEST-08, D-08) and the deletion codec
// round-trip (TEST-02 level 1). The table drives decode() — the adapter's
// mechanical half — and asserts each corruption's CURRENT verdict from the
// fixed vocabulary (Decoded/BadCRC/Undecodable), plus the core's discard
// counter delta for every class that exists today, observed through
// Adapter.Discards() after feeding the datagram over a real loopback
// socket. Rows whose rejection is a Phase 3 fix (empty key,
// tombstone-with-data — XPORT-10) pin the CURRENT Decoded behavior with an
// explicit forward reference: they flip when the fix lands, exactly like
// the red tests' expectation table. No production code is touched — the
// negative table is the pin; fuzz targets with committed corpora are the
// sibling plan 02-17, wave 3.
//
// The deletion round-trip is level 1 of the phase's four-level TEST-02
// proof: a DataReplication tombstone survives the wire codec intact, and
// the dump codec round-trips Deleted:true records. Levels 2-4 live on the
// simulator (plan 02-15's TestSim_Delete_replicatesTombstone plus plan
// 02-06's chaos deletion mix) and on the stand (this plan's
// stand_delete_test.go).

import (
	"bytes"
	"hash/crc32"
	"net"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/udprepl"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

// The negative table's construction constants: the review's F2c
// 70 000-character key (over the maximal IPv4 UDP payload once
// enveloped), that payload ceiling, the record fixtures' vector payloads,
// and the version counters that keep every row's record distinct.
const (
	negOversizeKeyLen  = 70000
	negMaxUDPPayload   = 65507    // the largest payload one IPv4 UDP datagram carries
	negVectorBytes     = 32       // the truncated-body fixture's data length
	negVersionMultiple = 3        // one counter per row, so fixtures never share a version
	negChecksumLong    = 0xFFFFFF // forces the trailing checksum varint to four bytes — the truncation point
)

// negativeRow is one corrupted-datagram table row: build constructs the
// exact bytes, wantStatus is decode()'s CURRENT mechanical verdict,
// wantClass names the discard class the core counts today ("" = the row
// is processed with no counter class existing yet), and feed reports
// whether the bytes can travel a real socket (the kernel refuses a datagram
// over the UDP payload ceiling, so the oversize row is decode-level only).
type negativeRow struct {
	name       string
	build      func(t *testing.T) []byte
	wantStatus port.DecodeStatus
	wantClass  string
	feed       bool
	check      func(t *testing.T, in port.InboundDatagram)
}

// TestDecode_negativeTable pins the decoder's reject behavior against the
// eight corruption classes D-08 names. Each row asserts (a) the mechanical
// verdict decode() returns and (b) where a counter exists TODAY, the
// DiscardStats delta the adapter exposes after the same datagram is fed
// through its real event loop — so a row that silently reclassifies or
// stops counting fails here first. Rows 7 and 8 are current-behavior pins
// of the XPORT-10 defects: they assert Decoded today and flip in Phase 3.
func TestDecode_negativeTable(t *testing.T) {
	t.Parallel()

	rows := []negativeRow{
		{
			name: "bad_crc_one_corrupted_body_byte",
			build: func(t *testing.T) []byte {
				t.Helper()
				datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
				if err != nil {
					t.Fatalf("encode Hello fixture: %v", err)
				}
				// The body is the envelope's last field, so the final byte
				// lies inside the CRC-covered region: the envelope still
				// parses, the CRC no longer covers the body.
				datagram[len(datagram)-1] ^= corruptByte
				return datagram
			},
			wantStatus: port.BadCRC,
			wantClass:  repl.DiscardBadCRC,
			feed:       true,
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				if in.Msg != nil {
					t.Fatalf("BadCRC verdict carries a decoded Msg %T — the body must not be unmarshaled", in.Msg)
				}
				if in.NodeID != nodeIDB || in.RunID != runIDB || in.ProtoVersion != repl.SupportedProtoVersion {
					t.Fatalf("BadCRC verdict lost the parsed envelope fields: %+v", in)
				}
			},
		},
		{
			name: "truncated_envelope_first_half",
			build: func(t *testing.T) []byte {
				t.Helper()
				datagram := negEncode(t, port.DataReplication{
					Key:    "neg/truncated-envelope",
					Record: negRecord(t, "neg/truncated-envelope", 1, bytes.Repeat([]byte("e"), negVectorBytes), false),
				})
				return datagram[:len(datagram)/2]
			},
			wantStatus: port.Undecodable,
			wantClass:  repl.DiscardUndecodable,
			feed:       true,
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// A datagram that never unmarshals as an envelope carries no
				// parsed fields at all.
				if in.Msg != nil || in.NodeID != 0 || in.ProtoVersion != 0 {
					t.Fatalf("Undecodable verdict on a truncated envelope carries parsed fields: %+v", in)
				}
			},
		},
		{
			name:       "truncated_body_inside_length_delimited_field",
			build:      negTruncatedBody,
			wantStatus: port.Undecodable,
			wantClass:  repl.DiscardUndecodable,
			feed:       true,
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				if in.Msg != nil {
					t.Fatalf("Undecodable verdict on a truncated body carries a Msg %T", in.Msg)
				}
			},
		},
		{
			name: "wrong_proto_version_with_correct_crc",
			build: func(t *testing.T) []byte {
				t.Helper()
				// The adversarial crafting seam: production encoding always
				// stamps SupportedProtoVersion, so only this helper can build
				// a CRC-passing envelope the core's version gate rejects.
				datagram, err := udprepl.EncodeDatagramForTestVersion(port.Hello{}, nodeIDB, runIDB, repl.SupportedProtoVersion+1)
				if err != nil {
					t.Fatalf("encode version-mismatch fixture: %v", err)
				}
				return datagram
			},
			wantStatus: port.Decoded,
			wantClass:  repl.DiscardBadProtoVersion,
			feed:       true,
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// The split the table pins: decode() does not judge the
				// version — the core's gate does, and its counter is the
				// observable.
				if in.ProtoVersion != repl.SupportedProtoVersion+1 {
					t.Fatalf("ProtoVersion = %d, want %d", in.ProtoVersion, repl.SupportedProtoVersion+1)
				}
				if _, ok := in.Msg.(port.Hello); !ok {
					t.Fatalf("Msg = %T, want port.Hello (decode only frames, the core judges the version)", in.Msg)
				}
			},
		},
		{
			name: "bad_run_id_not_16_bytes",
			build: func(t *testing.T) []byte {
				t.Helper()
				body, err := proto.Marshal(&replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}})
				if err != nil {
					t.Fatalf("marshal Hello body: %v", err)
				}
				return negEnvelope(t, repl.SupportedProtoVersion, []byte{1, 2, 3}, body)
			},
			wantStatus: port.Undecodable,
			wantClass:  repl.DiscardUndecodable,
			feed:       true,
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				if in.Msg != nil {
					t.Fatalf("Undecodable verdict on a bad run_id carries a Msg %T", in.Msg)
				}
			},
		},
		{
			name: "oversize_datagram_over_udp_ceiling_decodes_today",
			build: func(t *testing.T) []byte {
				t.Helper()
				key := strings.Repeat("f2c-", negOversizeKeyLen/len("f2c-"))
				datagram := negEncode(t, port.DataReplication{Key: key, Record: negRecord(t, key, 4, []byte("f"), false)})
				if len(datagram) <= negMaxUDPPayload {
					t.Fatalf("oversize fixture is %d bytes, want more than the %d-byte UDP payload ceiling", len(datagram), negMaxUDPPayload)
				}
				return datagram
			},
			wantStatus: port.Decoded,
			feed:       false, // the kernel refuses a real send — F2c
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// CURRENT behavior: decode() has no size bound, so an
				// oversized slice still decodes; the kernel's EMSGSIZE on
				// the send path is what keeps it off the wire (F2c, stand
				// TestStand_oversizeKeyF2c). XPORT-03 adds the decoder-side
				// bound in Phase 3 — this row flips with that fix.
				msg, ok := in.Msg.(port.DataReplication)
				if !ok {
					t.Fatalf("Msg = %T, want port.DataReplication", in.Msg)
				}
				if len(msg.Key) != negOversizeKeyLen {
					t.Fatalf("decoded key length = %d, want the %d-character oversize key intact", len(msg.Key), negOversizeKeyLen)
				}
			},
		},
		{
			name: "empty_key_decoded_today_xport_10_forward_reference",
			build: func(t *testing.T) []byte {
				t.Helper()
				return negEncode(t, port.DataReplication{Key: "", Record: negRecord(t, "", 5, []byte("empty-key-value"), false)})
			},
			wantStatus: port.Decoded,
			feed:       true, // processed today: the apply Acks — no discard class exists yet
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// CURRENT behavior: an empty key decodes and reaches the
				// store. Rejection is XPORT-10 (Phase 3) — this row flips
				// with the fix and must then assert Undecodable (or the
				// replacement class the fix chooses).
				msg, ok := in.Msg.(port.DataReplication)
				if !ok {
					t.Fatalf("Msg = %T, want port.DataReplication", in.Msg)
				}
				if msg.Key != "" {
					t.Fatalf("decoded key = %q, want the empty key preserved", msg.Key)
				}
				if string(msg.Record.Data) != "empty-key-value" {
					t.Fatalf("decoded data = %q, want %q", msg.Record.Data, "empty-key-value")
				}
			},
		},
		{
			name: "tombstone_with_data_decoded_today_xport_10_forward_reference",
			build: func(t *testing.T) []byte {
				t.Helper()
				const key = "neg/tombstone-with-data"
				return negEncode(t, port.DataReplication{Key: key, Record: negRecord(t, key, 6, []byte("tombstone-payload"), true)})
			},
			wantStatus: port.Decoded,
			feed:       true, // processed today: the apply Acks — no discard class exists yet
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// CURRENT behavior: Deleted:true with non-empty Data decodes
				// and merges like any record. Rejection is XPORT-10 (Phase
				// 3) — this row flips with the fix.
				msg, ok := in.Msg.(port.DataReplication)
				if !ok {
					t.Fatalf("Msg = %T, want port.DataReplication", in.Msg)
				}
				if !msg.Record.Deleted {
					t.Fatal("tombstone-with-data row decoded Deleted=false")
				}
				if string(msg.Record.Data) != "tombstone-payload" {
					t.Fatalf("decoded data = %q, want the payload to survive the tombstone flag", msg.Record.Data)
				}
			},
		},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			datagram := row.build(t)
			in := udprepl.DecodeDatagramForTest(datagram, handleB)
			if in.Status != row.wantStatus {
				t.Fatalf("decode verdict = %d, want %d (source %q, %d bytes)", in.Status, row.wantStatus, in.Source, len(datagram))
			}
			row.check(t, in)
			if row.feed {
				assertFedRow(t, datagram, row.wantClass)
			}
		})
	}
}

// TestWire_deleteRoundTrip pins TEST-02 level 1: a Deleted:true record
// survives both codecs intact. DataReplication is the only message kind
// that carries a record (Ack carries key+version, a SyncEntry key+version),
// so the datagram half encodes the tombstone — plain ASCII and multi-byte
// UTF-8 keys — and decodes the Deleted flag, the empty Data and the full
// 16-byte version back; the dump half round-trips Deleted:true records
// through the persist codec, including the greppable "deleted":true marker
// the stand's dump assertions rely on.
func TestWire_deleteRoundTrip(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, key string }{
		{name: "ascii_key", key: "neg/delete-roundtrip"},
		{name: "multi_byte_utf8_key", key: "удалить/ключ/🚀"},
	} {
		t.Run("datagram_"+tc.name, func(t *testing.T) {
			t.Parallel()
			ver := version.PackVersion(testNowMs, negVersionMultiple, nodeIDB)
			tombstone := port.DataReplication{
				Key: tc.key,
				Record: storage.Record{
					Version:  ver,
					Deleted:  true,
					Checksum: storage.RecordChecksum(tc.key, ver, nil, true),
				},
			}
			datagram, err := udprepl.EncodeDatagramForTest(tombstone, nodeIDB, runIDB)
			if err != nil {
				t.Fatalf("encode tombstone %q: %v", tc.key, err)
			}
			in := udprepl.DecodeDatagramForTest(datagram, handleB)
			if in.Status != port.Decoded {
				t.Fatalf("decode tombstone %q: status %d, want port.Decoded", tc.key, in.Status)
			}
			msg, ok := in.Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("decoded message = %T, want port.DataReplication", in.Msg)
			}
			if msg.Key != tc.key {
				t.Fatalf("decoded key = %q, want %q (key bytes unchanged by the tombstone flag)", msg.Key, tc.key)
			}
			if !msg.Record.Deleted {
				t.Fatalf("decoded tombstone %q lost Deleted=true on the wire", tc.key)
			}
			if len(msg.Record.Data) != 0 {
				t.Fatalf("decoded tombstone %q carries %d data bytes, want none", tc.key, len(msg.Record.Data))
			}
			if msg.Record.Version != ver {
				t.Fatalf("decoded tombstone %q version = %v, want %v", tc.key, msg.Record.Version, ver)
			}
			if msg.Record.Checksum != tombstone.Record.Checksum {
				t.Fatalf("decoded tombstone %q checksum = %d, want %d", tc.key, msg.Record.Checksum, tombstone.Record.Checksum)
			}
		})
	}

	t.Run("dump_codec_tombstone", func(t *testing.T) {
		t.Parallel()
		ver := version.PackVersion(testNowMs, negVersionMultiple+1, nodeIDB)
		records := map[string]storage.Record{
			"neg/delete-roundtrip/dump": {
				Version:  ver,
				Deleted:  true,
				Checksum: storage.RecordChecksum("neg/delete-roundtrip/dump", ver, nil, true),
			},
		}
		dump, err := persist.Encode(records)
		if err != nil {
			t.Fatalf("encode tombstone dump: %v", err)
		}
		// The greppable marker the stand's dump assertions match (the JSONL
		// record line always emits the deleted field).
		if !strings.Contains(string(dump), `"deleted":true`) {
			t.Fatalf("encoded dump does not carry the greppable \"deleted\":true marker:\n%s", dump)
		}
		decoded, discarded, err := persist.Decode(dump)
		if err != nil {
			t.Fatalf("decode tombstone dump: %v", err)
		}
		if discarded != 0 {
			t.Fatalf("decode tombstone dump discarded %d records, want none", discarded)
		}
		rec, ok := decoded["neg/delete-roundtrip/dump"]
		if !ok {
			t.Fatalf("decoded dump lost the tombstone record: %v", decoded)
		}
		if !rec.Deleted {
			t.Fatal("dump round-trip lost Deleted=true")
		}
		if len(rec.Data) != 0 {
			t.Fatalf("dump round-trip tombstone carries %d data bytes, want none", len(rec.Data))
		}
		if rec.Version != ver {
			t.Fatalf("dump round-trip version = %v, want %v", rec.Version, ver)
		}
	})

	t.Run("dump_codec_tombstone_with_data_current_behavior", func(t *testing.T) {
		t.Parallel()
		// The codec echoes Deleted:true alongside non-empty Data — the same
		// XPORT-10 current behavior the negative table's row 8 pins; the
		// codec's job is fidelity, the rejection belongs to the decoder fix.
		const key = "neg/delete-roundtrip/dump-with-data"
		ver := version.PackVersion(testNowMs, negVersionMultiple+2, nodeIDB)
		data := []byte("tombstone-dump-payload")
		records := map[string]storage.Record{
			key: {
				Version:  ver,
				Data:     data,
				Deleted:  true,
				Checksum: storage.RecordChecksum(key, ver, data, true),
			},
		}
		dump, err := persist.Encode(records)
		if err != nil {
			t.Fatalf("encode tombstone-with-data dump: %v", err)
		}
		decoded, discarded, err := persist.Decode(dump)
		if err != nil {
			t.Fatalf("decode tombstone-with-data dump: %v", err)
		}
		if discarded != 0 {
			t.Fatalf("decode tombstone-with-data dump discarded %d records, want none", discarded)
		}
		rec := decoded[key]
		if !rec.Deleted || !bytes.Equal(rec.Data, data) {
			t.Fatalf("dump round-trip tombstone-with-data = (deleted %t, data %q), want (true, %q)", rec.Deleted, rec.Data, data)
		}
	})
}

// assertFedRow feeds datagram through a live adapter's real event loop and
// asserts the discard-class delta the core exposes today: with wantClass
// set, exactly that class increments by one and the other four stay at
// their pre-shot values; with wantClass empty the row is expected to be
// PROCESSED — the apply's Ack is the evidence — and no discard class moves.
// The adapter broadcasts into a sink socket it does not own, so no
// own-echo verdict can contaminate the exact counters.
func assertFedRow(t *testing.T, datagram []byte, wantClass string) {
	t.Helper()
	adapter, udpPort := newNegativeAdapter(t)
	tx := peerSocket(t)
	before := adapter.Discards()
	acksBefore := adapter.SentByType().Ack
	sendDatagram(t, tx, udpPort, datagram)

	if wantClass == "" {
		waitFor(t, mirrorWait, "the adapter's Ack for a processed row", func() bool {
			return adapter.SentByType().Ack == acksBefore+1
		})
		if got := adapter.Discards(); got != before {
			t.Fatalf("processed row moved discard counters: %+v, want unchanged %+v", got, before)
		}
		return
	}

	waitFor(t, mirrorWait, "the "+wantClass+" discard counter", func() bool {
		return discardCount(adapter.Discards(), wantClass) == discardCount(before, wantClass)+1
	})
	want := before
	bumpDiscard(&want, wantClass)
	if got := adapter.Discards(); got != want {
		t.Fatalf("discards = %+v, want exactly %+v — only %s increments for this row", got, want, wantClass)
	}
}

// newNegativeAdapter returns the adapter under test running its real event
// loop on a free bind port, broadcasting to a sink socket the test holds —
// the asymmetric shape (bind P1, broadcast to P2) whose own-broadcast
// never returns to the receiver, so the discard counters stay exactly the
// fed datagrams' deltas.
func newNegativeAdapter(t *testing.T) (*udprepl.Adapter, int) {
	t.Helper()
	sink, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp4", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("bind broadcast sink socket: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })
	sinkPort, ok := sink.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("sink local address = %T, want *net.UDPAddr", sink.LocalAddr())
	}
	udpPort := freeUDPPort(t)
	adapter, _ := newAsymAdapter(t, nodeIDA, udpPort, sinkPort.Port)
	return adapter, udpPort
}

// discardCount returns wantClass's counter value from stats.
func discardCount(stats repl.DiscardStats, class string) uint64 {
	switch class {
	case repl.DiscardBadCRC:
		return stats.BadCRC
	case repl.DiscardUndecodable:
		return stats.Undecodable
	case repl.DiscardBadProtoVersion:
		return stats.BadProtoVersion
	case repl.DiscardOwnBroadcast:
		return stats.OwnEcho
	case repl.DiscardDuplicateNodeID:
		return stats.DuplicateNodeID
	default:
		return 0
	}
}

// bumpDiscard adds one to wantClass's counter in stats.
func bumpDiscard(stats *repl.DiscardStats, class string) {
	switch class {
	case repl.DiscardBadCRC:
		stats.BadCRC++
	case repl.DiscardUndecodable:
		stats.Undecodable++
	case repl.DiscardBadProtoVersion:
		stats.BadProtoVersion++
	case repl.DiscardOwnBroadcast:
		stats.OwnEcho++
	case repl.DiscardDuplicateNodeID:
		stats.DuplicateNodeID++
	}
}

// negEncode encodes one port message through the adapter's production
// encode path from node B's wire identity.
func negEncode(t *testing.T, msg port.Message) []byte {
	t.Helper()
	datagram, err := udprepl.EncodeDatagramForTest(msg, nodeIDB, runIDB)
	if err != nil {
		t.Fatalf("encode %T fixture: %v", msg, err)
	}
	return datagram
}

// negEnvelope hand-builds one envelope whose CRC covers body exactly — the
// adversarial seam for the domain and truncation rows the production
// encoder cannot construct.
func negEnvelope(t *testing.T, protoVersion uint32, runID, body []byte) []byte {
	t.Helper()
	envelope := &replproto.Envelope{
		ProtoVersion: protoVersion,
		NodeId:       uint32(nodeIDB),
		RunId:        runID,
		Crc32:        crc32.ChecksumIEEE(body),
		Body:         body,
	}
	datagram, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal hand-built envelope: %v", err)
	}
	return datagram
}

// negTruncatedBody builds the truncated-body row: a real marshaled Body
// with its trailing checksum varint cut short, wrapped in an envelope whose
// CRC covers the truncated bytes — so the envelope parses and the CRC gate
// passes, leaving the body unmarshal as the failing branch.
func negTruncatedBody(t *testing.T) []byte {
	t.Helper()
	ver := version.PackVersion(testNowMs, 2, nodeIDB)
	body, err := proto.Marshal(&replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
		Key:      "neg/truncated-body",
		Version:  ver[:],
		Data:     bytes.Repeat([]byte("b"), negVectorBytes),
		Checksum: negChecksumLong,
	}}})
	if err != nil {
		t.Fatalf("marshal truncated-body fixture body: %v", err)
	}
	truncated := body[:len(body)-1]
	var probe replproto.Body
	err = proto.Unmarshal(truncated, &probe)
	if err == nil {
		t.Fatalf("fixture bug: the truncated body (%d of %d bytes) still unmarshals", len(truncated), len(body))
	}
	return negEnvelope(t, repl.SupportedProtoVersion, runIDB[:], truncated)
}

// negRecord builds one well-formed record fixture for key: a distinct
// counter per row keeps the versions unique, and the checksum is the
// production domain so the fixtures are wire-real.
func negRecord(t *testing.T, key string, counter uint16, data []byte, deleted bool) storage.Record {
	t.Helper()
	ver := version.PackVersion(testNowMs, counter, nodeIDB)
	return storage.Record{
		Version:  ver,
		Data:     data,
		Deleted:  deleted,
		Checksum: storage.RecordChecksum(key, ver, data, deleted),
	}
}
