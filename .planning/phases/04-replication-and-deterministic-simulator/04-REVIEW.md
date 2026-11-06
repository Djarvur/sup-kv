---
phase: 04-replication-and-deterministic-simulator
reviewed: 2026-11-06T23:40:00Z
depth: standard
files_reviewed: 23
files_reviewed_list:
  - docs/IDEA.md
  - go.mod
  - internal/app/internal/simnet/simnet.go
  - internal/app/internal/simnet/simnet_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/core.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/peertable_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_bound_test.go
  - internal/app/repl/sim_chaos_test.go
  - internal/app/repl/sync_test.go
  - internal/port/doc.go
  - internal/port/repl.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/replproto/supkv.pb.go
  - internal/udprepl/replproto/supkv.proto
  - internal/udprepl/repl_test.go
  - internal/udprepl/smoke_test.go
  - internal/udprepl/syncsize_internal_test.go
  - Makefile
findings:
  critical: 1
  warning: 1
  info: 2
  total: 4
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-11-06T23:40:00Z
**Depth:** standard
**Files Reviewed:** 23
**Status:** issues_found

## Summary

Phase 4 adds the replication decision core (`internal/app/repl`), the port message
seam (`internal/port/repl.go`), the UDP/protobuf adapter (`internal/udprepl`), the
deterministic simulator (`internal/app/internal/simnet`), the persist `ApplyRemote`
seam, and the wire schema plus generated code. Verified during review: `go build
./...` clean, `go test -race ./internal/...` all pass, `make lint` (config verify +
golangci-lint 2.15.0 strict gate) clean; depguard boundaries hold (`internal/app`
imports no `net`/`os`/`syscall`/`google.golang.org/*`; `internal/port` is
stdlib+own-module); the generated `supkv.pb.go` provenance matches the pinned tooling
(protoc-gen-go v1.36.12 == the go.mod directive; protoc v3.21.12 == the Makefile's
3.21.x); the go.mod change (protobuf promoted from indirect to the single direct
runtime require, plus the protoc-gen-go tool directive) follows the AGENTS.md
dependency policy exactly.

The architecture is unusually disciplined (verified, not just claimed: the Advance
strictly-future deadline invariant holds across all six contributors; the chunk
estimate constants are pinned against the real codec by
`TestSync_chunkWireBudget`; the simulator drives the production `repl.Core`
handlers, so the determinism proof covers shipped code). One protocol-level
correctness defect and one adapter liveness defect survive that discipline:

1. The core Acks (and superseded tracking on) a `DataReplication` whose store apply
   FAILED — the confirmation layer can report "delivered" for a record that was
   never applied (CR-01).
2. `udprepl.Adapter.NotifyLocalWrite` has no exit-race guard and can block a writer
   forever after `Run` has returned (WR-01).

## Critical Issues

### CR-01: applyRemote Acks — and supersedes pending tracking with — a record the store rejected

**File:** `internal/app/repl/core.go:437-450`
**Issue:** `applyRemote` sends `out.Unicast(source, port.Ack{Key: msg.Key, Version: msg.Record.Version})`
unconditionally, even when `c.store.ApplyRemote` returned an error. It likewise
deletes the local pending entry when `msg.Record.Version.Compare(entry.version) > 0`
without consulting that error. Two spec violations follow from one failed apply:

- **False Ack.** docs/IDEA.md (§Replication protocol) and the port seam define the
  Ack as "sent after the record is applied to memory". The sender's `handleAck`
  (core.go:462-471) matches the exact tracked version, removes the rejecting peer
  from the wait-set, and can drain the entry entirely — retransmission stops and
  `UnackedStats` reports the write fully delivered while the rejecting node never
  stored it. Silent divergence is exactly the failure class the LWW design exists
  to exclude, and nothing reconciles it later: the periodic DataSync compare shows
  the key missing, the DataRequest reply re-delivers the record, the receiver
  rejects it again and Acks again — an infinite "confirmed but absent" loop.
- **Supersession by an unapplied record.** On the same error path, a strictly
  greater remote version deletes the local pending entry (core.go:446-448) even
  though neither the old nor the new record is now in flight toward this node's
  peers via retransmission.

The in-code justification ("merge idempotence makes the duplicate delivery of the
same record a no-op that still Acks") covers only the error-free equal-version
redelivery. The error path is reachable without any hostile actor: `storage.Merge`
returns `ErrValueTooLarge` whenever a datagram's `Data` exceeds the receiver's
`ValueMaxBytes` — and the codebase elsewhere explicitly contemplates heterogeneous
limits (persist `Open`/`Restore` skip oversize records; `ApplyRemote`'s own test
T-04-04 asserts `ErrValueTooLarge` passes through). The wire also permits it on an
otherwise homogeneous cluster: `udprepl` accepts datagrams up to 65507 bytes and
imposes no data-length cap in `decode`, so any 61441..65407-byte record (a node
with a larger limit, or a future version) is accepted by the adapter, rejected by
every store, and Acked by every receiver.

**Fix:** gate both the Ack and the supersession on a successful apply; keep the
error-free equal-version redelivery Acking (REPL-09's adjacency edge is preserved
because a no-op merge returns `nil`):

```go
func (c *Core) applyRemote(source string, msg port.DataReplication, out *port.Outbox) {
	_, _, err := c.store.ApplyRemote(msg.Key, msg.Record)
	if err != nil {
		c.cfg.Log(fmt.Sprintf("sup-kv: applying replicated record for key %q from handle %q: %v", msg.Key, source, err))
		return // not applied: no Ack, no supersession — never confirm a record the store refused
	}
	if entry, ok := c.pending[msg.Key]; ok && msg.Record.Version.Compare(entry.version) > 0 {
		delete(c.pending, msg.Key)
	}
	out.Unicast(source, port.Ack{Key: msg.Key, Version: msg.Record.Version})
}
```

The continued retransmission and climbing unacked gauge are the truthful signal
(the operator sees divergence instead of the protocol hiding it).

## Warnings

### WR-01: Adapter.NotifyLocalWrite blocks forever once Run has exited and the notify buffer fills

**File:** `internal/udprepl/repl.go:214-224`
**Issue:** `NotifyLocalWrite` is a plain send on the buffered `a.notify` channel
with no `a.done` select — unlike `readLoop` (repl.go:255-260), which selects on
`a.done` precisely to avoid this trap. After `Run` returns (context cancelled),
no receiver exists: up to 1024 notifications are silently swallowed, and the
1025th call blocks forever. This is the documented public write-path seam
("NotifyLocalWrite only enqueues and never blocks … the write path stays
available", D-05), so Phase 5's gRPC handlers will invoke it on user request
paths: a node whose replication context stops before its API server wedges every
subsequent write call. The silent drop of the first 1024 notifications is itself
a correctness gap — a confirmed local write quietly never replicates with no
error, no log, and no counter.
**Fix:** mirror `readLoop`'s exit guard and surface the drop:

```go
func (a *Adapter) NotifyLocalWrite(key string, rec storage.Record) {
	select {
	case a.notify <- localWrite{ /* … defensive copy as today … */ }:
	case <-a.done:
		a.log("udprepl: NotifyLocalWrite dropped: replication event loop has exited")
	}
}
```

(Consider returning an error instead of logging, so the Phase 5 caller can fail
the write visibly.)

## Info

### IN-01: Core.ownHandle is write-only dead state

**File:** `internal/app/repl/core.go:229` (field), `core.go:262-273` (assignment)
**Issue:** `ownHandle` is stored by `New` and never read anywhere in the package
(grep-verified). Its doc says it is "kept for the later own-echo and duplicate
node_id classification work", but that work landed in this very phase and uses
run_id equality and the peer table's remembered handle — never `ownHandle`. Every
caller (adapter, tests, simulator) threads a handle string through `New` for a
field that does nothing.
**Fix:** remove the field and the `New` parameter (mechanical: five call sites),
or, if the signature is considered frozen for Phase 5, correct the comment to say
it is currently unused rather than implying the classification work still needs it.

### IN-02: simnet Cluster.Write ignores the store Set error and logs a phantom write

**File:** `internal/app/internal/simnet/simnet.go:457-467`
**Issue:** `Write` discards the `store.Set` error (`_, _, _ =`), then reads
`Records()[key]` — on a rejected Set (data over the store's value limit) this is
the key's prior record or the zero Record, which is then broadcast via
`NotifyLocalWrite` and appended to `c.writes`, the write log the chaos test's
pointwise-supremum model is built from. A future scenario script with an oversized
payload would replicate and model a phantom record instead of failing loudly.
**Fix:** propagate the error — since the harness has no error channel, panic or
`panic(fmt.Sprintf(...))`-style loud failure is appropriate for a test harness;
alternatively return the error and let the scenario drivers `t.Fatalf`.

---

_Reviewed: 2026-11-06T23:40:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
