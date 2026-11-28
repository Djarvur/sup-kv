package grpcapi_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
)

// TestGRPC_maxRecv proves the receive ceiling is enforced at the transport
// (D-05, APIS-05, T-06-21) and not confused with the handler's value limit:
// the server runs under LimitsFor(testValueLimit), so a request that carries
// a value of MaxRecvBytes marshals above the ceiling and is rejected with
// ResourceExhausted at the transport — the store would have answered
// InvalidArgument for the same value, so the code proves which layer spoke —
// while a value at the configured limit reaches the handler and succeeds.
func TestGRPC_maxRecv(t *testing.T) {
	t.Parallel()

	limits := grpcapi.LimitsFor(testValueLimit)
	ts := buildTestServer(t, testServerConfig{
		gate:   openGateOptions(),
		limits: &limits,
	})
	awaitGateOpen(t, ts)
	ctx := t.Context()

	oversized := bytes.Repeat([]byte("x"), limits.MaxRecvBytes)
	_, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: "limit/recv", Value: oversized})
	assertCode(t, "Set whose value alone reaches MaxRecvBytes", err, codes.ResourceExhausted)

	legitimate := bytes.Repeat([]byte("x"), testValueLimit)
	_, err = ts.client.Set(ctx, &apiproto.SetRequest{Key: "limit/ok", Value: legitimate})
	if err != nil {
		t.Fatalf("Set at the configured value limit: want OK, got error: %v", err)
	}
}

// datagramCeilingValueLimit is the largest --value-max-bytes the UDP
// datagram ceiling admits (the udprepl max-record probe's value width), the
// high end of the configured range the transport envelope must follow.
const datagramCeilingValueLimit = 64408

// TestGRPC_maxValueRoundTrip is research Pitfall 11's largest-value transport
// round trip (IN-04): a node configured at the datagram-ceiling value limit
// accepts a Set carrying a storage.MaxKeyBytes-length key and a 64 408-byte
// value — the largest record the wire ceiling allows — through the production
// transport options, and the value reads back. The transport ceiling is
// derived from the EFFECTIVE value limit (LimitsFor(64 408)), so a node at the
// documented maximum cannot reject its own maximum-size write at the
// transport.
//
// The receive envelope now also has to admit a maximum List page token
// (WR-01), so MaxRecvBytes is much larger than one SetRequest and a single
// maximum-size write can no longer distinguish the effective limit from the
// storage default by message size alone. That derivation stays pinned by
// TestLimitsFor's exact arithmetic (MaxKeyBytes + effective limit + envelope
// for each case); this test keeps the end-to-end half — the documented
// maximum really round-trips.
func TestGRPC_maxValueRoundTrip(t *testing.T) {
	t.Parallel()

	limits := grpcapi.LimitsFor(datagramCeilingValueLimit)
	ts := buildTestServer(t, testServerConfig{
		gate:          openGateOptions(),
		valueMaxBytes: datagramCeilingValueLimit,
		discardDumps:  true,
		limits:        &limits,
	})
	awaitGateOpen(t, ts)
	ctx := t.Context()

	key := strings.Repeat("k", storage.MaxKeyBytes)
	value := bytes.Repeat([]byte("v"), datagramCeilingValueLimit)
	if payload := len(key) + len(value); payload > limits.MaxRecvBytes {
		t.Fatalf("largest accepted record payload = %d bytes, exceeds MaxRecvBytes (%d)", payload, limits.MaxRecvBytes)
	}

	setResp, err := ts.client.Set(ctx, &apiproto.SetRequest{Key: key, Value: value})
	if err != nil {
		t.Fatalf("Set of a %d-byte key and a %d-byte value at the datagram-ceiling limit: want OK, got %v",
			len(key), len(value), err)
	}
	if got := len(setResp.GetVersion()); got != 16 {
		t.Fatalf("Set response version size = %d, want the full 16-byte version", got)
	}

	got, err := ts.client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err != nil {
		t.Fatalf("Get of the largest accepted record: want OK, got %v", err)
	}
	if rec := got.GetRecord(); rec.GetKey() != key || !bytes.Equal(rec.GetValue(), value) {
		t.Errorf("Get returned (%d-byte key, %d-byte value), want the %d-byte key and %d-byte value written",
			len(rec.GetKey()), len(rec.GetValue()), len(key), len(value))
	}
}

// TestGRPC_worstCasePageFitsSendLimit is the cross-plan proof that plan
// 06-05's serialized-entry accounting and this plan's send ceiling agree at
// maximum key length (D-05, APIS-05): 257 maximum-length keys holding
// maximum-size values are seeded at the production value limit, the real
// List handler answers a page_size 256 request, and the byte budget stops
// the page below the page size. The marshaled response must fit
// LimitsFor(storage.ValueMaxBytes).MaxSendBytes, so a page the handler
// returns can never be rejected by the production send ceiling.
func TestGRPC_worstCasePageFitsSendLimit(t *testing.T) {
	t.Parallel()

	limits := grpcapi.LimitsFor(storage.ValueMaxBytes)
	ts := buildTestServer(t, testServerConfig{
		gate:          openGateOptions(),
		valueMaxBytes: storage.ValueMaxBytes,
		discardDumps:  true,
		limits:        &limits,
	})
	awaitGateOpen(t, ts)

	// One more than the documented maximum page size: if the budget never
	// stopped the walk, the page-size cap alone would return 256 entries.
	value := bytes.Repeat([]byte("v"), storage.ValueMaxBytes)
	keyPrefix := strings.Repeat("k", storage.MaxKeyBytes-4)
	for i := range 257 {
		key := fmt.Sprintf("%s%04d", keyPrefix, i)
		if got := len(key); got != storage.MaxKeyBytes {
			t.Fatalf("seed key length = %d, want exactly storage.MaxKeyBytes (%d)", got, storage.MaxKeyBytes)
		}
		rec := storage.Record{Version: version.PackVersion(int64(i)+1, 0, 7), Data: value}
		_, _, seedErr := ts.engine.ApplyRemote(key, rec)
		if seedErr != nil {
			t.Fatalf("seed ApplyRemote(%q) error = %v, want nil", key, seedErr)
		}
	}

	resp, err := ts.client.List(t.Context(), &apiproto.ListRequest{PageSize: 256})
	if err != nil {
		t.Fatalf("List page_size 256: want OK, got error: %v", err)
	}
	if got := len(resp.GetRecords()); got == 0 {
		t.Fatal("List page_size 256: want a non-empty page, got 0 records")
	}
	if got := len(resp.GetRecords()); got >= 256 {
		t.Fatalf("List page_size 256: want the byte budget to stop the page below 256, got %d records", got)
	}
	for _, key := range resp.GetKeys() {
		if got := len(key); got != storage.MaxKeyBytes {
			t.Fatalf("page key length = %d, want the maximum %d", got, storage.MaxKeyBytes)
		}
	}
	raw, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("proto.Marshal(ListResponse): want nil error, got %v", err)
	}
	if got, ceiling := len(raw), limits.MaxSendBytes; got > ceiling {
		t.Fatalf("marshaled page = %d bytes, want at most LimitsFor(storage.ValueMaxBytes).MaxSendBytes (%d)",
			got, ceiling)
	}
}
