package grpcapi

import (
	"context"
	"math"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/version"
)

const (
	// defaultWaitTimeout is the operator-tunable wait_for_replication
	// default (D-09, research A2): NewServer substitutes it for a zero or
	// negative ServerOptions.WaitTimeout so a misconfigured option can
	// never mean "wait forever".
	defaultWaitTimeout = time.Second

	// maxWaitTimeoutMs is the largest wait_timeout_ms whose millisecond→
	// nanosecond product cannot overflow int64 (WR-02): any per-request
	// value above it is clamped here BEFORE the conversion, because the
	// raw product of a hostile value wraps arbitrarily — negative (a
	// deadline already in the past, an immediate wrong report) or huge
	// positive (an effectively unbounded server-side wait holding the
	// handler goroutine).
	maxWaitTimeoutMs = math.MaxInt64 / int64(time.Millisecond)

	// waitTimeoutCeilingFactor is the per-request ceiling's multiple over
	// the operator default (WR-02): an unvalidated wait_timeout_ms may
	// hold a write handler at most an order of magnitude beyond the
	// operator's own configuration.
	waitTimeoutCeilingFactor = 10
)

// waitForReplication produces the wait_for_replication report (API-05,
// D-08..D-11) after a successful engine write: it reports only what the
// injected ServerOptions.WaitReplicated seam observed — the adapter's
// tracker, never the single-driver repl core and never a polled mirror.
// The report is the D-08 aggregate (acked plus counts, no per-peer
// breakdown); it may delay the response within timeout but never changes
// the write outcome, which the engine already confirmed before this runs.
// timeout is the resolved wait deadline (D-09: the positive per-request
// wait_timeout_ms, else the operator-tunable default).
//
// The pinned semantics (XPORT-12):
//
//  1. A nil WaitReplicated is the legal single-node wiring (D-11): no
//     tracker can exist through this seam, so the report is
//     (true, 0, 0) immediately.
//  2. total is captured exactly once, before the wait (D-10): peers
//     discovered during the wait never extend the awaited set. A nil
//     KnownPeers means zero peers — the single-node wiring.
//  3. total == 0 returns (true, 0, 0) after calling WaitReplicated with
//     total 0. That call is the RELEASE point for the tracker the adapter
//     created when the write's notification requested tracking: the wait
//     path itself enters no waiting loop, so without the call the
//     zero-peer path would leak the tracker (D-11).
//  4. Otherwise the adapter's tracker waits and reports; the count is
//     clamped to the write-time total (WR-01: a matching Ack from a node
//     outside the write-time peer snapshot joins the core's acked set, so
//     the raw count can exceed total) and success requires full write-time
//     coverage, as the tracker alone decides. Eviction is not an Ack and a
//     superseded version is an honest false — the tracker returns
//     acked=false for both; timeout and ctx cancellation return the last
//     observed count with acked=false. The wait never reports a success it
//     did not observe.
func (s *Server) waitForReplication(ctx context.Context, key string, ver version.RecordVersion, timeout time.Duration) (acked bool, ackedCount, total uint32) {
	if s.opts.WaitReplicated == nil {
		return true, 0, 0
	}
	var peerCount uint32
	if s.opts.KnownPeers != nil {
		peerCount = uint32(s.opts.KnownPeers()) //nolint:gosec // G115: the peer table is keyed by 16-bit node ids, so the count never exceeds 65535 — the conversion cannot truncate.
	}
	if peerCount == 0 {
		// Zero known peers at write time (D-11): immediate acked with
		// nothing awaited. The total==0 call releases the notify-time
		// tracker; its report is discarded because the single-node answer
		// is fixed.
		_, _ = s.opts.WaitReplicated(ctx, key, ver, 0, timeout)
		return true, 0, 0
	}
	acked, count := s.opts.WaitReplicated(ctx, key, ver, peerCount, timeout)
	return acked, min(count, peerCount), peerCount
}

// waitTimeout resolves the D-09 wait deadline for one write: a positive
// per-request wait_timeout_ms wins over the operator-tunable default,
// which NewServer already normalized to defaultWaitTimeout. The request
// field is unvalidated remote input (WR-02), so two bounds apply before a
// value becomes a duration: anything above maxWaitTimeoutMs clamps first —
// its raw conversion would wrap the int64 product — and the resolved
// duration then caps at ten times the operator default, so no client can
// hold a write handler an order of magnitude beyond the operator's own
// configuration (the cap clamps the request only; negative and zero keep
// falling back to the default, D-09, and a rejection is never an option:
// the write is already engine-confirmed when this resolves, and no path
// may turn a confirmed write into an error).
func (s *Server) waitTimeout(requestMs int64) time.Duration {
	if requestMs <= 0 {
		return s.opts.WaitTimeout
	}
	if requestMs > maxWaitTimeoutMs {
		requestMs = maxWaitTimeoutMs
	}
	timeout := time.Duration(requestMs) * time.Millisecond
	if ceiling := waitTimeoutCeilingFactor * s.opts.WaitTimeout; ceiling > 0 && timeout > ceiling {
		// The > 0 guard keeps an absurd operator default whose ×10
		// overflows from inverting the comparison; such a default then
		// stays bounded only by the conversion-safe ceiling.
		return ceiling
	}
	return timeout
}
