package grpcapi

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/version"
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
// sync-arrived signal source and the backstop deadline — plus the
// wait_for_replication seams (API-05, wait.go): the write-time peer-id
// source, the tracker wait seam and the operator-tunable wait default. The
// zero value is legal — a nil SyncArrived means the sync path can never fire
// (the resolver only polls it when non-nil), a zero StartupGate and a zero
// WaitTimeout normalize to their 1s design defaults, and nil wait seams mean
// the single-node configuration — so a bare ServerOptions{} is a single-node
// server whose gate opens after one second and whose writes report the
// honest zero-peer (acked=false, 0, 0) (A-01).
//
// Exported fields on an exported struct on purpose: main (plan 05-06)
// injects the udprepl adapter's SyncArrived mirror, tests inject fakes,
// and future gate options grow without another NewServer signature change.
type ServerOptions struct {
	// SyncArrived reports whether a COMPLETE listing round has been processed
	// — every chunk of some peer's DataSync round received, not merely the
	// first arrival — the API-04 sync half. Production injects the udprepl
	// adapter's atomic mirror (the adapter reports `mirrorStats`' completeness
	// bit, D-01 semantics); tests inject fakes. Only the gate resolver's poll
	// loop calls it, never the write handlers: the handlers see only the
	// resolved gate flag.
	SyncArrived func() bool

	// StartupGate is the API-04 backstop: the gate opens this long after
	// RunGate starts even when no sync round arrived — a partitioned or
	// single node must still become writable. Zero or negative means the
	// defaultStartupGate.
	StartupGate time.Duration

	// ListedKey reports whether a peer's complete listing named key — the
	// adapter's listing-mirror membership (D-01, rule 14.5(6)). Nil means
	// no listing knowledge: the single-node configuration and tests. The
	// production adapter publishes completeness and membership in one
	// atomic snapshot, so a gate open implies the listed keys are visible;
	// the handlers consult this seam only AFTER the gate, because a closed
	// node answers Unavailable for every read whatever the listing says.
	ListedKey func(key string) bool

	// WritePeers reports the sorted peer ids known when the write was
	// accepted — the write-time peer snapshot the wait_for_replication
	// report is pinned to (D-01, A-01). The handler calls it exactly once,
	// immediately before the engine write, and the SAME slice feeds the
	// replication notification and the wait report, so a peer discovered
	// after acceptance can neither extend the awaited set nor move the
	// reported total. Production injects the udprepl adapter's
	// KnownPeerIDs atomic mirror; nil — like an empty slice — means zero
	// peers, the single-node configuration, which answers the honest
	// (acked=false, 0, 0) rather than overstating a write no peer
	// acknowledged (A-01).
	WritePeers func() []uint16

	// WaitReplicated is the wait_for_replication tracker seam (XPORT-12,
	// D-04): it waits for the (key, version) write's tracker, bounded by
	// timeout and ctx, and reports the observed acked flag and count
	// against the write-time peer total. Production injects the udprepl
	// adapter's method value; a nil seam is the single-node wiring (D-11)
	// — no tracker can exist through this seam, so nothing was awaited and
	// the wait reports the honest (false, 0, 0) immediately (A-01).
	// Handler goroutines wait only here, never by calling the single-driver
	// core.
	WaitReplicated func(ctx context.Context, key string, ver version.RecordVersion, total uint32, timeout time.Duration) (acked bool, ackedCount uint32)

	// WaitTimeout is the operator-tunable wait_for_replication default
	// (D-09, A2): NewServer substitutes the 1s design default when zero
	// or negative. A positive per-request wait_timeout_ms overrides it
	// for that write.
	WaitTimeout time.Duration

	// ReplayNotify is the paced replication seam for a snapshot load's
	// replayed records (PROTO-08, T-4-20, WR-04): production injects the
	// udprepl adapter's NotifyReplayedWrite, which queues each record on the
	// shared record-send pacer instead of broadcasting it immediately, so a
	// large load drains one datagram per pacing slot rather than flooding the
	// segment. A nil seam is the single-node or unpaced configuration: the
	// handler then notifies each replayed record through the live LocalNotify
	// seam with track=false. Only SnapshotLoad calls it.
	ReplayNotify LocalReplayNotify
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

// gated reports whether the startup gate is still closed, so every data
// RPC — reads and writes alike — is rejected with Unavailable (API-04,
// D-01). The purpose is that a cold node must not answer from a picture it
// does not yet have: reads of keys it has never heard of would lie as
// NotFound, and writes would stamp versions its peers' newer state would
// have to beat. Get, List and Watch consult the gate exactly like Set and
// Delete.
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
