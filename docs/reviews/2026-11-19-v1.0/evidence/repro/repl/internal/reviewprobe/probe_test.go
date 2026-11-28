package reviewprobe

// Scratch-copy probe: the production udprepl codec (repl.go copied verbatim
// under another package name so that no udprepl test — and no socket — runs).

import (
	"hash/crc32"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
	replproto "github.com/Djarvur/sup-kv/internal/udprepl/replproto/supkv/v1"
)

const udpMax = 65507

func enc(t *testing.T, msg port.Message) []byte {
	t.Helper()
	var run [16]byte
	run[0] = 1
	d, err := encodeDatagram(65535, run, msg)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestProbe_datagramSizes(t *testing.T) {
	ver := version.PackVersion(1730000000000, 4095, 65535)
	rec := func(n int) storage.Record {
		return storage.Record{Version: ver, Data: make([]byte, n), Checksum: 0xFFFFFFFF}
	}
	// 1. Overhead of DataReplication with a 1-byte key and a max value.
	base := len(enc(t, port.DataReplication{Key: "k", Record: rec(61440)}))
	t.Logf("DataReplication(value=61440, key=1B) = %d B => envelope+body overhead %d B", base, base-61440-1)
	// 2. Largest key that still fits with a max-size value.
	lo, hi := 1, 70000
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if len(enc(t, port.DataReplication{Key: strings.Repeat("k", mid), Record: rec(61440)})) <= udpMax {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	t.Logf("value=61440 B: largest key that still fits one UDP datagram = %d B", lo)
	over := len(enc(t, port.DataReplication{Key: strings.Repeat("k", 4096), Record: rec(61440)}))
	t.Logf("legal Set(key=4096 B, value=61440 B) => DataReplication datagram %d B (limit %d)", over, udpMax)
	// 3. Non-DataReplication messages vs sync_chunk_bytes=1400.
	k1400 := strings.Repeat("k", 1400)
	t.Logf("Ack(key=1400 B) = %d B; DataRequest(key=1400 B) = %d B; DataSync(single entry key=1400 B) = %d B (spec: every non-DataReplication datagram <= sync_chunk_bytes=1400)",
		len(enc(t, port.Ack{Key: k1400, Version: ver})),
		len(enc(t, port.DataRequest{Key: k1400})),
		len(enc(t, port.DataSync{Entries: []port.SyncEntry{{Key: k1400, Version: ver}}})))
	k66k := strings.Repeat("k", 66000)
	t.Logf("key=66000 B: Ack=%d B, DataRequest=%d B, DataSync(single)=%d B, DataReplication(empty value)=%d B — none can be sent (limit %d)",
		len(enc(t, port.Ack{Key: k66k, Version: ver})),
		len(enc(t, port.DataRequest{Key: k66k})),
		len(enc(t, port.DataSync{Entries: []port.SyncEntry{{Key: k66k, Version: ver}}})),
		len(enc(t, port.DataReplication{Key: k66k, Record: rec(0)})), udpMax)
	if over > udpMax {
		t.Errorf("a legal Set (key 4096 B, value at value_max_bytes) produces a %d B DataReplication > %d B UDP payload limit", over, udpMax)
	}
}

// A datagram from a FUTURE protocol version whose body this build cannot
// parse must be classified bad_proto_version (spec: logged), not undecodable.
func TestProbe_futureProtoVersionMisclassified(t *testing.T) {
	// Body with an unknown oneof member (field 6, length-delimited).
	var body []byte
	body = protowire.AppendTag(body, 6, protowire.BytesType)
	body = protowire.AppendBytes(body, []byte("v2-only message"))
	run := make([]byte, 16)
	run[0] = 9
	env := &replproto.Envelope{ProtoVersion: 2, NodeId: 7, RunId: run, Crc32: crc32.ChecksumIEEE(body), Body: body}
	raw, err := proto.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	in := decode(raw)
	t.Logf("decode(v2 datagram) => Status=%d ProtoVersion=%d NodeID=%d", in.Status, in.ProtoVersion, in.NodeID)

	var classes []string
	core := repl.New(1, [16]byte{1}, "self", nopStore{}, repl.Config{OnDiscard: func(class, _ string) { classes = append(classes, class) }})
	var out port.Outbox
	core.HandleDatagram(time.UnixMilli(1730000000000), in, &out)
	if len(classes) != 1 || classes[0] != repl.DiscardBadProtoVersion {
		t.Errorf("datagram with proto_version=2 was discarded as %v, want [%s]: version skew during a rolling upgrade is reported as garbage", classes, repl.DiscardBadProtoVersion)
	}
}

type nopStore struct{}

func (nopStore) ApplyRemote(string, storage.Record) ([]storage.Conflict, time.Duration, error) {
	return nil, 0, nil
}
func (nopStore) Records() map[string]storage.Record { return nil }

type memStore struct{ s *storage.Store }

func (m memStore) ApplyRemote(k string, r storage.Record) ([]storage.Conflict, time.Duration, error) {
	return m.s.Merge(k, r)
}
func (m memStore) Records() map[string]storage.Record { return m.s.Records() }

// Hostile-but-well-formed bodies the receive path accepts.
func TestProbe_wellFormedGarbageAccepted(t *testing.T) {
	now := func() time.Time { return time.UnixMilli(1730000000000) }
	st := storage.NewStore(1, clock.NewClock(now, 0, 0, time.Minute), storage.Config{})
	core := repl.New(1, [16]byte{1}, "self", memStore{st}, repl.Config{})
	feed := func(msg port.Message) (port.InboundDatagram, []port.Outbound) {
		var run [16]byte
		run[0] = 2
		raw, err := encodeDatagram(2, run, msg)
		if err != nil {
			t.Fatal(err)
		}
		in := decode(raw)
		in.Source = "peer"
		var out port.Outbox
		core.HandleDatagram(now(), in, &out)
		return in, out.Drain()
	}
	ver := version.PackVersion(1730000000001, 0, 2)

	// (a) empty key
	_, o := feed(port.DataReplication{Key: "", Record: storage.Record{Version: ver, Data: []byte("x")}})
	_, hasEmpty := st.Records()[""]
	t.Logf("(a) DataReplication with EMPTY key: stored=%v, emissions=%d (Ack)", hasEmpty, len(o))

	// (b) checksum that does not match the record fields
	_, o = feed(port.DataReplication{Key: "b", Record: storage.Record{Version: ver, Data: []byte("x"), Checksum: 0xDEADBEEF}})
	rb := st.Records()["b"]
	t.Logf("(b) record with wire checksum 0xDEADBEEF: stored=%v (stored checksum %#x), Acked=%v", rb.Data != nil, rb.Checksum, len(o) == 1)

	// (c) tombstone that still carries a payload
	feed(port.DataReplication{Key: "c", Record: storage.Record{Version: ver, Data: []byte("ghost"), Deleted: true}})
	rc := st.Records()["c"]
	t.Logf("(c) tombstone with payload: Deleted=%v Data=%q", rc.Deleted, rc.Data)

	// (d) a 'version' that is not a UUIDv7 at all (all 0xFF)
	var junk version.RecordVersion
	for i := range junk {
		junk[i] = 0xFF
	}
	feed(port.DataReplication{Key: "d", Record: storage.Record{Version: junk, Data: []byte("x")}})
	_, hasD := st.Records()["d"]
	nv, _, _ := st.Set("after", []byte("y"))
	l, c, _ := nv.Unpack()
	t.Logf("(d) version ff..ff accepted=%v; next LOCAL write is stamped l=%d (year ~%d) c=%d", hasD, l, 1970+l/1000/31556952, c)

	if hasEmpty || hasD {
		t.Errorf("receive path stored an empty-key record (%v) and a non-UUIDv7 far-future version (%v) without any validation", hasEmpty, hasD)
	}
}

// The envelope CRC covers only body: identity fields are unprotected.
func TestProbe_crcDoesNotCoverEnvelopeIdentity(t *testing.T) {
	var run [16]byte
	run[0] = 2
	raw := enc(t, port.Hello{})
	_ = run
	var env replproto.Envelope
	if err := proto.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	env.NodeId ^= 0x0100 // one flipped bit in node_id, CRC left as is
	env.RunId[3] ^= 0x01
	tampered, _ := proto.Marshal(&env)
	in := decode(tampered)
	if in.Status == port.Decoded {
		t.Errorf("a datagram with a corrupted node_id/run_id passes the CRC gate (Status=Decoded, NodeID=%d): crc32 covers body only", in.NodeID)
	}
}
