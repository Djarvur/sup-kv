package grpcapi

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// datagramCeilingValueLimit is the largest --value-max-bytes the UDP
// datagram ceiling admits (the udprepl max-record probe's value width). The
// limits table runs it as the high end of the configured range: the recv
// ceiling must follow the EFFECTIVE value limit, never the storage default
// (RESEARCH Pitfall 11, T-06-24).
const datagramCeilingValueLimit = 64408

// TestLimitsFor pins the transport envelope as a pure function of the
// effective value limit and the List page budget (D-05, APIS-05):
//
//   - MaxRecvBytes covers the largest valid SetRequest — a key at
//     storage.MaxKeyBytes and a value at the EFFECTIVE configured limit —
//     plus apiEnvelopeOverheadBytes, so a legitimate maximally sized write is
//     never rejected at the transport.
//   - MaxSendBytes covers a maximum List page — listPageByteBudget plus
//     listEnvelopeOverheadBytes, the two constants plan 06-05 owns — and every
//     single-record response, which is far smaller.
//   - The keepalive triple is the documented trusted-LAN value: a 2 m server
//     ping with a 20 s timeout and a 30 s minimum client ping interval.
//
// The cases run the storage default, the datagram-ceiling limit and both
// non-positive normalizations, so the value limit is proven to be an input
// rather than a constant read inside LimitsFor.
func TestLimitsFor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		valueMaxBytes int
		wantRecv      int
	}{
		{
			name:          "storage default value limit",
			valueMaxBytes: storage.ValueMaxBytes,
			wantRecv:      storage.MaxKeyBytes + storage.ValueMaxBytes + apiEnvelopeOverheadBytes,
		},
		{
			name:          "datagram ceiling value limit",
			valueMaxBytes: datagramCeilingValueLimit,
			wantRecv:      storage.MaxKeyBytes + datagramCeilingValueLimit + apiEnvelopeOverheadBytes,
		},
		{
			name:          "zero normalizes to the storage default",
			valueMaxBytes: 0,
			wantRecv:      storage.MaxKeyBytes + storage.ValueMaxBytes + apiEnvelopeOverheadBytes,
		},
		{
			name:          "negative normalizes to the storage default",
			valueMaxBytes: -7,
			wantRecv:      storage.MaxKeyBytes + storage.ValueMaxBytes + apiEnvelopeOverheadBytes,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := LimitsFor(tc.valueMaxBytes)
			if got.MaxRecvBytes != tc.wantRecv {
				t.Fatalf("LimitsFor(%d).MaxRecvBytes = %d, want %d",
					tc.valueMaxBytes, got.MaxRecvBytes, tc.wantRecv)
			}
			if want := listPageByteBudget + listEnvelopeOverheadBytes; got.MaxSendBytes != want {
				t.Fatalf("LimitsFor(%d).MaxSendBytes = %d, want listPageByteBudget+listEnvelopeOverheadBytes = %d",
					tc.valueMaxBytes, got.MaxSendBytes, want)
			}
			if got.KeepaliveTime != 2*time.Minute {
				t.Fatalf("LimitsFor(%d).KeepaliveTime = %s, want 2m", tc.valueMaxBytes, got.KeepaliveTime)
			}
			if got.KeepaliveTimeout != 20*time.Second {
				t.Fatalf("LimitsFor(%d).KeepaliveTimeout = %s, want 20s", tc.valueMaxBytes, got.KeepaliveTimeout)
			}
			if got.MinPingInterval != 30*time.Second {
				t.Fatalf("LimitsFor(%d).MinPingInterval = %s, want 30s", tc.valueMaxBytes, got.MinPingInterval)
			}
		})
	}
}
