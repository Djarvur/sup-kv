package grpcapi

import (
	"time"

	"google.golang.org/grpc"
)

// apiEnvelopeOverheadBytes bounds the bytes one API request or response
// carries besides its entries: the key/value field framing and the base64
// next_page_token. It is plan 06-05's listEnvelopeOverheadBytes, consumed
// here — the per-entry key, version and record framing already sit inside
// listPageByteBudget, and list.go owns both List constants.
const apiEnvelopeOverheadBytes = listEnvelopeOverheadBytes

// TransportLimits is the explicit gRPC transport envelope (D-05, APIS-05).
type TransportLimits struct {
	MaxRecvBytes     int
	MaxSendBytes     int
	KeepaliveTime    time.Duration
	KeepaliveTimeout time.Duration
	MinPingInterval  time.Duration
}

// LimitsFor derives the transport envelope from the effective value limit.
func LimitsFor(valueMaxBytes int) TransportLimits {
	return TransportLimits{}
}

// ServerOptions renders the limits as gRPC server options.
func (l TransportLimits) ServerOptions() []grpc.ServerOption {
	return nil
}
