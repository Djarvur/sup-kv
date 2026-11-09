package grpcapi

import (
	"context"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/version"
)

const (
	// defaultWaitTimeout is the operator-tunable wait_for_replication
	// default (D-09, research A2): NewServer substitutes it for a zero or
	// negative ServerOptions.WaitTimeout so a misconfigured option can
	// never mean "wait forever".
	defaultWaitTimeout = time.Second

	// waitPoll is the pending-mirror poll period (the smoke waitFor
	// pattern, production-side): the udprepl adapter refreshes its
	// copy-on-write mirror after every event-loop iteration, so a 5ms poll
	// observes drain progress well inside the ack ladder's first 200ms
	// step while costing one map read per tick.
	waitPoll = 5 * time.Millisecond

	// waitAppear bounds the appearance phase (RESEARCH Pitfall 4 — the
	// load-bearing guard against the false acked): NotifyLocalWrite is
	// asynchronous through the adapter's 1024-deep notify queue, so an
	// entry absent from the mirror may merely mean "not yet dequeued".
	// Treating that absence as drained would report acked=true before a
	// single Ack was possible; the wait therefore polls for the (key,
	// version) entry's appearance for at most the earlier of timeout and
	// waitAppear before absence can mean drained (or never notified).
	waitAppear = 500 * time.Millisecond
)

// waitForReplication produces the wait_for_replication report (API-05,
// D-08..D-11) after a successful engine write: it reads only the injected
// ServerOptions seams — KnownPeers for the write-time peer snapshot,
// PendingCopy for the adapter's mirrored pending set — never the
// single-driver repl core. The report is the D-08 aggregate (acked plus
// counts, no per-peer breakdown) derived from what the mirror observed; it
// may delay the response within timeout but never changes the write
// outcome, which the engine already confirmed before this runs. timeout is
// the resolved wait deadline (D-09: the positive per-request
// wait_timeout_ms, else the operator-tunable default).
//
// The pinned algorithm:
//
//  1. total is captured exactly once from KnownPeers before any polling
//     (D-10): peers discovered during the wait never extend the awaited
//     set. A nil seam means zero peers — the single-node wiring.
//  2. total == 0 returns (true, 0, 0) immediately (D-11): a single-node
//     deployment is a valid configuration with nothing to replicate.
//  3. Appearance phase: the (key, ver) entry may lag the confirmed write
//     through the notify queue, so absence does not mean drained until the
//     entry had its bounded chance to appear (earlier of timeout and
//     waitAppear). Never appearing returns (false, 0, total).
//  4. Drain phase: lastAcked is the maximum len(Acked) observed. Entry
//     absent — drained via full Ack coverage. Entry with a different
//     version — a greater remote version superseded the tracked one and
//     the report freezes at the last observation (the Q2 pin: honest — the
//     tracked version's Acks did not all arrive). WaitingOn empty — the
//     wait set fully resolved; success only at full Acked coverage
//     (Pitfall 3: eviction is not an Ack).
//  5. Deadline — or ctx cancellation — returns the last observed state,
//     always with acked=false: the wait never reports a success it did not
//     observe.
func (s *Server) waitForReplication(ctx context.Context, key string, ver version.RecordVersion, timeout time.Duration) (acked bool, ackedCount, total uint32) {
	if s.opts.KnownPeers == nil || s.opts.PendingCopy == nil {
		// Nil seams are the legal single-node wiring (D-11): no
		// replication to wait for, success with nothing awaited.
		return true, 0, 0
	}
	peerCount := uint32(s.opts.KnownPeers()) //nolint:gosec // G115: the peer table is keyed by 16-bit node ids, so the count never exceeds 65535 — the conversion cannot truncate.
	if peerCount == 0 {
		// Zero known peers at write time (D-11): immediate acked with
		// nothing awaited — the appearance phase must not run either.
		return true, 0, 0
	}
	deadline := time.Now().Add(timeout)
	appear := deadline
	if bounded := time.Now().Add(waitAppear); bounded.Before(appear) {
		appear = bounded
	}
	poll := time.NewTimer(waitPoll)
	defer poll.Stop()
	for {
		if info, ok := s.opts.PendingCopy()[key]; ok && info.Version == ver {
			return s.waitForDrain(ctx, key, ver, peerCount, deadline)
		}
		if !time.Now().Before(appear) {
			// The entry never appeared within the appearance bound:
			// either the notify was dropped (shutdown) or the entry
			// appeared and drained entirely between two polls. Nothing
			// was observed, so the report is an honest false.
			return false, 0, peerCount
		}
		select {
		case <-ctx.Done():
			return false, 0, peerCount
		case <-poll.C:
			poll.Reset(waitPoll)
		}
	}
}

// waitForDrain is the drain phase of waitForReplication (step 4): poll the
// mirrored pending set until the tracked entry drains (absent — the core
// deletes it exactly when every write-time peer Acked), resolves
// (WaitingOn empty — every known peer Acked or was evicted, and eviction is
// never counted: T-05-04), is superseded (version mismatch — counts frozen
// at the last observation, the Q2 pin), or the deadline fires. lastAcked
// never decreases: it is the maximum len(Acked) observed across polls.
func (s *Server) waitForDrain(ctx context.Context, key string, ver version.RecordVersion, total uint32, deadline time.Time) (acked bool, ackedCount, totalPeers uint32) {
	lastAcked := uint32(0)
	poll := time.NewTimer(waitPoll)
	defer poll.Stop()
	for {
		info, ok := s.opts.PendingCopy()[key]
		// The Acked count narrows once per poll; the set is bounded by
		// the uint16-keyed peer table, so the conversion cannot truncate.
		ackedLen := uint32(len(info.Acked)) //nolint:gosec // G115: bounded by the uint16-keyed peer table (<= 65535), cannot truncate.
		switch {
		case !ok:
			// Drained: full Ack coverage deleted the entry — success
			// only if the observed Acked count reached the write-time
			// total.
			return lastAcked == total, lastAcked, total
		case info.Version != ver:
			// Superseded by a greater remote version: the core's
			// tracking promise was fulfilled by winning, not by Acks —
			// honest false with the counts frozen at the last
			// observation (the Q2 pin).
			return false, lastAcked, total
		case len(info.WaitingOn) == 0:
			// Fully resolved: only the Acked count is reportable
			// (evicted peers live in neither set — Pitfall 3), and
			// only full coverage is success.
			if ackedLen > lastAcked {
				lastAcked = ackedLen
			}
			return lastAcked == total, lastAcked, total
		}
		if ackedLen > lastAcked {
			lastAcked = ackedLen
		}
		if !time.Now().Before(deadline) {
			return false, lastAcked, total
		}
		select {
		case <-ctx.Done():
			return false, lastAcked, total
		case <-poll.C:
			poll.Reset(waitPoll)
		}
	}
}

// waitTimeout resolves the D-09 wait deadline for one write: a positive
// per-request wait_timeout_ms wins over the operator-tunable default,
// which NewServer already normalized to defaultWaitTimeout.
func (s *Server) waitTimeout(requestMs int64) time.Duration {
	if requestMs > 0 {
		return time.Duration(requestMs) * time.Millisecond
	}
	return s.opts.WaitTimeout
}
