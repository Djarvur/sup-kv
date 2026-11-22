package udprepl_test

// The decoder negative table (TEST-08, D-08) and the deletion codec
// round-trip (TEST-02 level 1). The table drives decode() — the adapter's
// mechanical half — and asserts each corruption's verdict from the fixed
// vocabulary (Decoded/BadCRC/Undecodable/ProtoVersionMismatch), plus the
// core's discard counter delta for every class, observed through
// Adapter.Discards() after feeding the datagram over a real loopback socket.
// The wire-validation rows (empty key, tombstone-with-data, key length,
// record checksum, far-future version — XPORT-03/04/10) assert the classes
// the core's validation pipeline landed in 03-08: decode() still frames them
// (Decoded) and the core discards them before any store or clock call. The
// proto-version rows pin the pre-body verdict and its order: a foreign
// version classifies before the CRC gate, so a foreign version with a
// corrupted body is bad_proto_version, never bad_crc. No production code is
// touched — the negative table is the pin; fuzz targets with committed
// corpora are the sibling plan 02-17, wave 3.
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
// enveloped), that payload ceiling, the feedable one-byte-over-the-key-limit
// length, the record fixtures' vector payloads, and the version counters
// that keep every row's record distinct.
const (
	negOversizeKeyLen  = 70000
	negMaxKeyBytesLen  = storage.MaxKeyBytes + 1 // 1 025: the first rejected key length
	negMaxUDPPayload   = 65507                   // the largest payload one IPv4 UDP datagram carries
	negVectorBytes     = 32                      // the truncated-body fixture's data length
	negVersionMultiple = 3                       // one counter per row, so fixtures never share a version
	negChecksumLong    = 0xFFFFFF                // forces the trailing checksum varint to four bytes — the truncation point
)

// negFarFutureVersion returns the 16-byte all-ones version: its version
// nibble (0xF) and variant bits (0b11) violate the RFC 9562 layout and its
// reading is far beyond any clock lead — the crafted poison the version
// class rejects.
func negFarFutureVersion() version.RecordVersion {
	return version.RecordVersion{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
}

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

// TestDecode_negativeTable pins the decoder's reject behavior across D-08's
// corruption classes and the wire-validation classes the core landed in
// 03-08. Each row asserts (a) the mechanical verdict decode() returns and
// (b) the DiscardStats delta the adapter exposes after the same datagram is
// fed through its real event loop — so a row that silently reclassifies or
// stops counting fails here first. The decode-only oversize row stays
// Decoded with feed=false (the kernel refuses the send); its feedable
// one-byte-over-the-limit sibling asserts the core's key_too_long class.
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
				// a foreign-version envelope.
				return negVersionEncode(t, repl.SupportedProtoVersion+1)
			},
			wantStatus: port.ProtoVersionMismatch,
			wantClass:  repl.DiscardBadProtoVersion,
			feed:       true,
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// The pre-body verdict: no body was decoded, but the envelope
				// fields the core needs to attribute the discard survived.
				if in.Msg != nil {
					t.Fatalf("ProtoVersionMismatch verdict carries a decoded Msg %T — the body must never be decoded for a foreign version", in.Msg)
				}
				if in.ProtoVersion != repl.SupportedProtoVersion+1 {
					t.Fatalf("ProtoVersion = %d, want %d", in.ProtoVersion, repl.SupportedProtoVersion+1)
				}
				if in.NodeID != nodeIDB || in.RunID != runIDB {
					t.Fatalf("ProtoVersionMismatch verdict lost the envelope identity: node_id %d run_id %x", in.NodeID, in.RunID)
				}
			},
		},
		{
			name:       "proto_version_zero_is_a_mismatch",
			build:      func(t *testing.T) []byte { t.Helper(); return negVersionEncode(t, 0) },
			wantStatus: port.ProtoVersionMismatch,
			wantClass:  repl.DiscardBadProtoVersion,
			feed:       true,
			check:      negVersionMismatchCheck(0),
		},
		{
			name:       "proto_version_one_the_previous_wire_version_is_a_mismatch",
			build:      func(t *testing.T) []byte { t.Helper(); return negVersionEncode(t, 1) },
			wantStatus: port.ProtoVersionMismatch,
			wantClass:  repl.DiscardBadProtoVersion,
			feed:       true,
			check:      negVersionMismatchCheck(1),
		},
		{
			name:       "proto_version_max_uint32_is_a_mismatch",
			build:      func(t *testing.T) []byte { t.Helper(); return negVersionEncode(t, ^uint32(0)) },
			wantStatus: port.ProtoVersionMismatch,
			wantClass:  repl.DiscardBadProtoVersion,
			feed:       true,
			check:      negVersionMismatchCheck(^uint32(0)),
		},
		{
			// The ordering trap (D-07, Pitfall 12): a future version's body
			// need not follow this version's CRC contract, so the version
			// verdict must win over the CRC gate. The same corruption on a
			// same-version datagram stays BadCRC (the row above).
			name: "future_proto_version_wins_over_corrupted_body",
			build: func(t *testing.T) []byte {
				t.Helper()
				datagram := negVersionEncode(t, repl.SupportedProtoVersion+1)
				datagram[len(datagram)-1] ^= corruptByte
				return datagram
			},
			wantStatus: port.ProtoVersionMismatch,
			wantClass:  repl.DiscardBadProtoVersion,
			feed:       true,
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				if in.Msg != nil {
					t.Fatalf("combined foreign-version/corrupt-body row decoded a Msg %T, want nil", in.Msg)
				}
				if in.ProtoVersion != repl.SupportedProtoVersion+1 {
					t.Fatalf("ProtoVersion = %d, want %d — the pre-body verdict must preserve it", in.ProtoVersion, repl.SupportedProtoVersion+1)
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
			name: "oversize_key_decode_only_over_udp_ceiling",
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
				// The codec has no size bound, so an oversized slice still
				// decodes; the kernel's EMSGSIZE on the send path is what
				// keeps it off the wire (F2c, the rewritten stand scenario in
				// 03-09). The feedable one-byte-over-the-limit row below
				// asserts the core's key_too_long class over a real socket.
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
			name: "key_too_long_over_max_key_bytes",
			build: func(t *testing.T) []byte {
				t.Helper()
				key := strings.Repeat("k", negMaxKeyBytesLen)
				return negEncode(t, port.DataReplication{Key: key, Record: negRecord(t, key, 5, []byte("long-key-value"), false)})
			},
			wantStatus: port.Decoded,
			wantClass:  repl.DiscardKeyTooLong,
			feed:       true, // feedable: the key exceeds storage.MaxKeyBytes by one byte
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				msg, ok := in.Msg.(port.DataReplication)
				if !ok {
					t.Fatalf("Msg = %T, want port.DataReplication", in.Msg)
				}
				if len(msg.Key) != negMaxKeyBytesLen {
					t.Fatalf("decoded key length = %d, want %d", len(msg.Key), negMaxKeyBytesLen)
				}
			},
		},
		{
			name: "empty_key_discarded_by_core",
			build: func(t *testing.T) []byte {
				t.Helper()
				return negEncode(t, port.DataReplication{Key: "", Record: negRecord(t, "", 6, []byte("empty-key-value"), false)})
			},
			wantStatus: port.Decoded,
			wantClass:  repl.DiscardEmptyKey,
			feed:       true, // the core discards it before the store — XPORT-10
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// decode() frames the datagram; the core's validation
				// pipeline owns the empty-key verdict (landed in 03-08).
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
			name: "tombstone_with_data_discarded_by_core",
			build: func(t *testing.T) []byte {
				t.Helper()
				const key = "neg/tombstone-with-data"
				return negEncode(t, port.DataReplication{Key: key, Record: negRecord(t, key, 7, []byte("tombstone-payload"), true)})
			},
			wantStatus: port.Decoded,
			wantClass:  repl.DiscardTombstoneWithData,
			feed:       true, // the core discards it before the store — XPORT-10
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				// decode() stays a faithful codec: Deleted:true with non-empty
				// Data decodes intact, and the core's validation pipeline
				// rejects it (landed in 03-08).
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
		{
			name:       "bad_checksum_one_flipped_data_byte",
			build:      negFlippedDataByte,
			wantStatus: port.Decoded,
			wantClass:  repl.DiscardBadChecksum,
			feed:       true, // the envelope CRC is recomputed; only the record checksum is wrong
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				msg, ok := in.Msg.(port.DataReplication)
				if !ok {
					t.Fatalf("Msg = %T, want port.DataReplication", in.Msg)
				}
				if len(msg.Record.Data) == 0 || msg.Record.Data[0] != 'c'^corruptByte {
					t.Fatalf("decoded data = %x, want the one-byte-flipped payload", msg.Record.Data)
				}
				got := storage.RecordChecksum(msg.Key, msg.Record.Version, msg.Record.Data, msg.Record.Deleted)
				if got == msg.Record.Checksum {
					t.Fatal("fixture bug: the record checksum still matches the flipped payload")
				}
			},
		},
		{
			name: "bad_version_far_future",
			build: func(t *testing.T) []byte {
				t.Helper()
				const key = "neg/far-future"
				ver := negFarFutureVersion()
				return negEncode(t, port.DataReplication{Key: key, Record: storage.Record{
					Version:  ver,
					Data:     []byte("future"),
					Checksum: storage.RecordChecksum(key, ver, []byte("future"), false),
				}})
			},
			wantStatus: port.Decoded,
			wantClass:  repl.DiscardBadVersion,
			feed:       true, // the checksum is correct; the version layout/future bound rejects it
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				msg, ok := in.Msg.(port.DataReplication)
				if !ok {
					t.Fatalf("Msg = %T, want port.DataReplication", in.Msg)
				}
				if msg.Record.Version != negFarFutureVersion() {
					t.Fatalf("decoded version = %x, want the all-ones poison intact", msg.Record.Version[:])
				}
			},
		},
		{
			name: "bad_version_far_future_sync_entry",
			build: func(t *testing.T) []byte {
				t.Helper()
				return negEncode(t, port.DataSync{
					SyncID:     1,
					ChunkIndex: 1,
					ChunkCount: 1,
					Entries: []port.SyncEntry{
						{Key: "neg/sync-entry-ok", Version: version.PackVersion(testNowMs, 8, nodeIDB)},
						{Key: "neg/sync-entry-poison", Version: negFarFutureVersion()},
					},
				})
			},
			wantStatus: port.Decoded,
			wantClass:  repl.DiscardBadVersion,
			feed:       true, // one poisoned entry discards the whole listing
			check: func(t *testing.T, in port.InboundDatagram) {
				t.Helper()
				msg, ok := in.Msg.(port.DataSync)
				if !ok {
					t.Fatalf("Msg = %T, want port.DataSync", in.Msg)
				}
				if len(msg.Entries) != 2 {
					t.Fatalf("decoded %d sync entries, want 2", len(msg.Entries))
				}
				if msg.Entries[1].Version != negFarFutureVersion() {
					t.Fatalf("second entry version = %x, want the all-ones poison intact", msg.Entries[1].Version[:])
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
		dump, err := persist.Encode(records, storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("encode tombstone dump: %v", err)
		}
		// The greppable marker the stand's dump assertions match (the JSONL
		// record line always emits the deleted field).
		if !strings.Contains(string(dump), `"deleted":true`) {
			t.Fatalf("encoded dump does not carry the greppable \"deleted\":true marker:\n%s", dump)
		}
		decoded, discarded, err := persist.Decode(dump, storage.ValueMaxBytes)
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

	t.Run("dump_codec_tombstone_with_data_fidelity", func(t *testing.T) {
		t.Parallel()
		// The dump codec echoes Deleted:true alongside non-empty Data: the
		// codec's job is fidelity, and the inbound rejection of that shape is
		// the core's validation class (landed in 03-08), not the loader's.
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
		dump, err := persist.Encode(records, storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("encode tombstone-with-data dump: %v", err)
		}
		decoded, discarded, err := persist.Decode(dump, storage.ValueMaxBytes)
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
	case repl.DiscardEmptyKey:
		return stats.EmptyKey
	case repl.DiscardKeyTooLong:
		return stats.KeyTooLong
	case repl.DiscardTombstoneWithData:
		return stats.TombstoneWithData
	case repl.DiscardBadChecksum:
		return stats.BadChecksum
	case repl.DiscardBadVersion:
		return stats.BadVersion
	case repl.DiscardBadSyncHeader:
		return stats.BadSyncHeader
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
	case repl.DiscardEmptyKey:
		stats.EmptyKey++
	case repl.DiscardKeyTooLong:
		stats.KeyTooLong++
	case repl.DiscardTombstoneWithData:
		stats.TombstoneWithData++
	case repl.DiscardBadChecksum:
		stats.BadChecksum++
	case repl.DiscardBadVersion:
		stats.BadVersion++
	case repl.DiscardBadSyncHeader:
		stats.BadSyncHeader++
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

// negVersionEncode encodes one Hello through the explicit-version crafting
// seam — the only way to build a CRC-correct foreign-version datagram, since
// the production encoder always stamps SupportedProtoVersion.
func negVersionEncode(t *testing.T, protoVersion uint32) []byte {
	t.Helper()
	datagram, err := udprepl.EncodeDatagramForTestVersion(port.Hello{}, nodeIDB, runIDB, protoVersion)
	if err != nil {
		t.Fatalf("encode version-%d fixture: %v", protoVersion, err)
	}
	return datagram
}

// negVersionMismatchCheck returns the shared assertion of a foreign-version
// row: no body decoded, and the envelope identity plus the foreign version
// preserved for attribution.
func negVersionMismatchCheck(protoVersion uint32) func(t *testing.T, in port.InboundDatagram) {
	return func(t *testing.T, in port.InboundDatagram) {
		t.Helper()
		if in.Msg != nil {
			t.Fatalf("ProtoVersionMismatch verdict carries a decoded Msg %T, want nil", in.Msg)
		}
		if in.ProtoVersion != protoVersion {
			t.Fatalf("ProtoVersion = %d, want %d preserved", in.ProtoVersion, protoVersion)
		}
		if in.NodeID != nodeIDB || in.RunID != runIDB {
			t.Fatalf("ProtoVersionMismatch verdict lost the envelope identity: node_id %d run_id %x", in.NodeID, in.RunID)
		}
	}
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

// negFlippedDataByte builds the bad-checksum row: a real encoded
// DataReplication whose body is unmarshaled, one Data byte flipped and the
// body re-marshaled under a FRESHLY computed envelope CRC — so the CRC gate
// passes and only the record checksum no longer covers the payload.
func negFlippedDataByte(t *testing.T) []byte {
	t.Helper()
	const key = "neg/bad-checksum"
	rec := negRecord(t, key, 9, []byte("checksum-payload"), false)
	datagram := negEncode(t, port.DataReplication{Key: key, Record: rec})
	var env replproto.Envelope
	err := proto.Unmarshal(datagram, &env)
	if err != nil {
		t.Fatalf("unmarshal bad-checksum envelope: %v", err)
	}
	var body replproto.Body
	err = proto.Unmarshal(env.GetBody(), &body)
	if err != nil {
		t.Fatalf("unmarshal bad-checksum body: %v", err)
	}
	dr := body.GetDataReplication()
	if dr == nil || len(dr.GetData()) == 0 {
		t.Fatal("fixture bug: the encoded body carries no DataReplication payload")
	}
	dr.Data[0] ^= corruptByte
	bodyBytes, err := proto.Marshal(&body)
	if err != nil {
		t.Fatalf("marshal flipped body: %v", err)
	}
	env.Body = bodyBytes
	env.Crc32 = crc32.ChecksumIEEE(bodyBytes)
	flipped, err := proto.Marshal(&env)
	if err != nil {
		t.Fatalf("marshal flipped envelope: %v", err)
	}
	if storage.RecordChecksum(key, rec.Version, dr.GetData(), rec.Deleted) == rec.Checksum {
		t.Fatal("fixture bug: flipping a data byte left the record checksum valid")
	}
	return flipped
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
