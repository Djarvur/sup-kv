package grpcapi

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const (
	// gatePollInterval is the resolver's poll period. A8 pins the design:
	// one small goroutine polling every 20ms is acceptable — worst case it
	// wakes once per interval until the gate opens, then RunGate returns
	// and the goroutine is gone.
	gatePollInterval = 20 * time.Millisecond

	// defaultStartupGate is the design default (IDEA.md §Parameters,
	// startup_gate 1s), substituted for a zero or negative StartupGate so
	// a misconfigured option can never mean "never open".
	defaultStartupGate = time.Second

	// gateMessage is the status message every gated write RPC carries. It
	// names both opening paths so an operator reading client logs can tell
	// a cold node (no sync yet) from one whose backstop elapsed.
	gateMessage = "startup gate: waiting for first sync round or startup_gate"
)

// ServerOptions carries the startup-gate wiring (API-04, API-06): the
// sync-arrived signal source and the backstop deadline. The zero value is
// legal — a nil SyncArrived means the sync path can never fire (the
// resolver only polls it when non-nil) and a zero StartupGate normalizes
// to the 1s design default — so a bare ServerOptions{} is a single-node
// server whose gate opens after one second.
//
// Exported fields on an exported struct on purpose: main (plan 05-06)
// injects the udprepl adapter's SyncArrived mirror, tests inject fakes,
// and future gate options grow without another NewServer signature change.
type ServerOptions struct {
	// SyncArrived reports whether the first replication sync round has
	// arrived — the API-04 sync half. Production injects the udprepl
	// adapter's atomic mirror (A1 semantics: any received DataSync
	// datagram counts); tests inject fakes. Only the gate resolver's poll
	// loop calls it, never the write handlers: the handlers see only the
	// resolved gate flag.
	SyncArrived func() bool

	// StartupGate is the API-04 backstop: the gate opens this long after
	// RunGate starts even when no sync round arrived — a partitioned or
	// single node must still become writable. Zero or negative means the
	// defaultStartupGate.
	StartupGate time.Duration
}

// gateUnavailable returns the API-04 status every gated write carries
// while the startup gate is closed. A fresh value per rejection: status
// errors are single-use response payloads.
func gateUnavailable() error {
	return status.Error(codes.Unavailable, gateMessage)
}

// Health returns the standard gRPC health server bound 1:1 to the startup
// gate (API-06): it reports NOT_SERVING for the empty service name — the
// whole-server status per the health protocol — from construction until
// RunGate opens the gate, SERVING after. Register it alongside SupKV on
// the same grpc.Server — healthpb.RegisterHealthServer(gs, srv.Health()) —
// so orchestrators gate client traffic on the standard endpoint.
func (s *Server) Health() *health.Server {
	return s.healthSrv
}

// gated reports whether the startup gate is still closed, so writes are
// rejected with Unavailable (API-04). Reads never consult it: Get, List
// and Watch are served from the first moment — serving stale-but-locally-
// confirmed state beats rejecting reads on a node whose peers have not
// synced yet, and the gate's whole purpose is to keep a cold node from
// *writing* versions its peers' newer state would have to beat.
func (s *Server) gated() bool {
	return !s.gateOpen.Load()
}

// RunGate runs the startup-gate resolver: a poll loop (the A8 pin — one
// small goroutine, one wake per interval until the gate opens) that opens
// the gate on the earlier of the injected SyncArrived signal flipping true
// or StartupGate elapsing since this call. Opening stores the gate flag
// and flips the health endpoint's empty service name to SERVING — the two
// transitions are one event (T-05-08's 1:1 binding, and the resolver is
// the only opener). RunGate returns once the gate has opened, or on ctx
// cancel with the gate left closed: shutdown sheds traffic through the
// health endpoint instead (plan 05-06's D-19 order). Launch exactly once,
// before serving: `go srv.RunGate(ctx)`.
func (s *Server) RunGate(ctx context.Context) {
	deadline := s.opts.StartupGate
	if deadline <= 0 {
		deadline = defaultStartupGate
	}
	backstop := time.NewTimer(deadline)
	defer backstop.Stop()
	poll := time.NewTicker(gatePollInterval)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-backstop.C:
			s.openGate()
			return
		case <-poll.C:
			if s.opts.SyncArrived != nil && s.opts.SyncArrived() {
				s.openGate()
				return
			}
		}
	}
}

// openGate opens the gate exactly once and mirrors the transition into the
// health endpoint under the same branch. The resolver is the only opener,
// but its two opening paths (sync flip, backstop) can both fire in one
// select iteration — the CAS keeps the health flip single-shot with the
// flag store.
func (s *Server) openGate() {
	if s.gateOpen.CompareAndSwap(false, true) {
		s.healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	}
}
