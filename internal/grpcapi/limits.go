package grpcapi

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// The transport envelope's keepalive constants (D-05, APIS-05). The values
// are chosen for the trusted-LAN small-cluster envelope: a node has a handful
// of long-lived consumers, so a 2 m server ping with a 20 s answer window
// detects a dead peer quickly without generating background traffic, while a
// 30 s minimum client ping interval lets a consumer library use its own 30 s
// keepalive without tripping the enforcement policy. PermitWithoutStream stays
// true (see ServerOptions) because consumers legitimately hold an idle
// connection between writes.
const (
	// serverKeepaliveTime is the idle span after which the server pings a
	// silent client. gRPC's default is 2 h — far too slow to notice a dead
	// consumer on a LAN (T-06-22).
	serverKeepaliveTime = 2 * time.Minute

	// serverKeepaliveTimeout is how long the server waits for the ping's
	// answer before closing the connection: four round trips of slack on a
	// LAN, short enough that a stuck peer never holds a stream indefinitely.
	serverKeepaliveTimeout = 20 * time.Second

	// minClientPingInterval is the shortest client ping cadence the server
	// tolerates. gRPC's default is 5 min; 30 s is the documented consumer
	// contract and a lower bound, not a request to the client.
	minClientPingInterval = 30 * time.Second
)

// apiEnvelopeOverheadBytes bounds the bytes one API request or response
// carries besides its entries: the field framing and the encoded
// next_page_token at the dump format's line ceiling. It is plan 06-05's
// listEnvelopeOverheadBytes consumed here — the per-entry key, version and
// record framing already sit inside listPageByteBudget, and list.go owns both
// List constants, so this file never restates either (APIS-05, T-06-24). The
// receive side needs the token component too: the follow-up request that
// carries a maximum page token must fit the receive ceiling (WR-01).
const apiEnvelopeOverheadBytes = listEnvelopeOverheadBytes

// TransportLimits is the explicit gRPC transport envelope (D-05, APIS-05):
// nothing the server accepts or sends is governed by a gRPC default, and
// every value is derived from the configured protocol constants so a
// node-size change can never leave a limit behind.
type TransportLimits struct {
	// MaxRecvBytes bounds one inbound message. It covers the largest valid
	// request of either shape: a SetRequest with a key at storage.MaxKeyBytes
	// and a value at the EFFECTIVE configured limit, or a ListRequest carrying
	// a next_page_token at the dump format's line ceiling — a pre-upgrade key
	// above storage.MaxKeyBytes is a legal position, so the token domain is
	// the format's, not the write ceiling (WR-01). apiEnvelopeOverheadBytes
	// includes the token's encoded ceiling. A request above MaxRecvBytes is
	// rejected at the transport with ResourceExhausted — before any
	// allocation-heavy handling — which is a different verdict from the
	// handler's InvalidArgument value limit (T-06-21).
	MaxRecvBytes int

	// MaxSendBytes bounds one outbound message. It covers a maximum List
	// page: listPageByteBudget of serialized entries plus
	// listEnvelopeOverheadBytes of envelope (the token at the format ceiling,
	// the repeated-field tags and the gRPC prefix), as well as every
	// single-record response, which is far smaller. The ceiling is provable
	// only because the page budget counts full serialized entries — the key
	// twice, the version, the value and the fixed framing — so a page the
	// 06-05 walk returns cannot marshal past it.
	MaxSendBytes int

	// KeepaliveTime is the server's idle ping period.
	KeepaliveTime time.Duration

	// KeepaliveTimeout is how long the server waits for the ping's answer.
	KeepaliveTimeout time.Duration

	// MinPingInterval is the shortest client ping cadence the enforcement
	// policy permits.
	MinPingInterval time.Duration
}

// LimitsFor derives the transport envelope from the effective value limit
// (D-05, APIS-05). valueMaxBytes is the configured --value-max-bytes, never
// the storage default constant: a node running at the datagram ceiling
// (64 408) must accept its own maximum-size writes at the transport, while a
// non-positive input normalizes to storage.ValueMaxBytes exactly as the store
// does (RESEARCH Pitfall 11, T-06-24). The send ceiling does not depend on
// the value limit: it is the List page budget plus the envelope ceiling,
// whose token component is the dump format's line ceiling (WR-01).
func LimitsFor(valueMaxBytes int) TransportLimits {
	if valueMaxBytes <= 0 {
		valueMaxBytes = storage.ValueMaxBytes
	}
	return TransportLimits{
		MaxRecvBytes:     storage.MaxKeyBytes + valueMaxBytes + apiEnvelopeOverheadBytes,
		MaxSendBytes:     listPageByteBudget + listEnvelopeOverheadBytes,
		KeepaliveTime:    serverKeepaliveTime,
		KeepaliveTimeout: serverKeepaliveTimeout,
		MinPingInterval:  minClientPingInterval,
	}
}

// ServerOptions renders the limits as the grpc.Server options the production
// constructor passes to [grpc.NewServer] (D-05, APIS-05). PermitWithoutStream
// is always true: a consumer may hold an idle connection between writes and
// must not be disconnected for pinging without a stream. Every option is
// explicit so the server never silently inherits a gRPC default.
func (l TransportLimits) ServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.MaxRecvMsgSize(l.MaxRecvBytes),
		grpc.MaxSendMsgSize(l.MaxSendBytes),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    l.KeepaliveTime,
			Timeout: l.KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             l.MinPingInterval,
			PermitWithoutStream: true,
		}),
	}
}
