---
phase: 05-grpc-api-and-binary
reviewed: 2026-11-18T10:57:01Z
depth: deep
files_reviewed: 4
files_reviewed_list:
  - Makefile
  - cmd/sup-kv/main.go
  - go.mod
  - go.sum
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
  necessity: 1
status: issues_found
---

# Phase 5: Code Review Report (incremental re-review — delta since 69bb132)

**Reviewed:** 2026-11-18T10:57:01Z
**Depth:** deep
**Files Reviewed:** 4 (Makefile, cmd/sup-kv/main.go, go.mod, go.sum)
**Status:** issues_found

## Summary

Incremental deep re-review of the four phase-5-covered files that changed
since the phase's last full review (base 69bb132): the Makefile gained the
additive `bench`, `release` and `release-snapshot` targets; main.go gained
the gRPC reflection registration plus the structured-logging, metrics
listener, signal-ownership and shutdown-order work from later phases that
landed in this file; go.mod/go.sum gained the grpcurl and goreleaser tool
directives and the `go 1.27.4` bump.

**The gRPC surface wiring holds.** Traced end to end:
`grpcapi.NewServer(engine, adapter.NotifyLocalWrite, hub, ServerOptions{...})`
matches gate.go:46-83 field for field (SyncArrived/KnownPeers/PendingCopy
are real `*udprepl.Adapter` methods — repl.go:400-489); `hub.Ingest` is the
`storage.Config.OnChange` tap (watch.go:213); `srv.RunGate(apiCtx)` /
`hub.Run(apiCtx)` are launched on the API context exactly once;
`listenGRPC` registers the service, the gate-bound health server (verified
to start at NOT_SERVING for the empty service name — server.go:88) and
`reflection.Register` (grpcurl discovery works with zero proto files, per
the phase-7 D-09 intent). Wait/startup-gate semantics verified in gate.go
and server.go: gate checked before validation on all four write RPCs, wait
strictly after engine confirmation, `WaitTimeout <= 0` normalized. All
seven RPCs implemented over the `UnimplementedSupKVServiceServer` embed.

**Every flag knob main.go registers is defended at its consumer.** Zero or
negative durations/ints from flags cannot panic or wedge the node:
`repl.Config.normalized` (core.go:134-163, including the
SyncResponseRate (0, 1e9] clamp), `persist.Config.normalized`
(engine.go:37-44), `storage.Config.normalized` (store.go:104-109),
gate.go:123-126 / server.go:81-83 for startup-gate and wait-timeout, and
`driftWarn <= 0` is a documented disable in clock.go:75-77. This is the
deliberate "total over any Config" contract — main.go needs no duplicate
range validation, and flagging it for one would be an
already-defended-elsewhere anti-finding.

**Toolchain and Makefile verified mechanically.** `go build ./...`,
`go vet ./...`, `go mod tidy -diff` (exit 0 — go.mod/go.sum are exactly
tidy), `go tool golangci-lint config verify` (exit 0 against the vendored
powerman 2.15.0 strict config, matching the pinned golangci-lint v2.15.0
tool) and `go test -race ./cmd/sup-kv/` all pass. The runtime require
block is still exactly grpc + protobuf (everything else is tool-directive
indirect); the protoc-gen-go tool and the protobuf runtime resolve to the
one identical module version (v1.36.13-0.20261016172727), preserving the
plugin==runtime invariant. Both Makefile comment claims check out:
`.goreleaser.yaml` (main `./cmd/sup-kv`, linux/darwin × amd64/arm64,
checksums.txt, SPDX SBOMs) and the tag-triggered
`.github/workflows/release.yml` exist; all three benchmarks named by the
`bench` target exist (`bench_test.go` in each listed package); `.PHONY`
covers every target. go.sum is a generated lock file — consistency-checked
via tidy, not line-reviewed.

**Findings.** One new Warning: a shutdown-ordering race in main.go where
the still-live gate resolver can flip the health endpoint back to SERVING
between the drain's health flip and `apiCancel` — a window widened to
seconds by the intervening metrics-listener shutdown. One new Info: the
Serve-error exit path skips the final Flush/drain (recoverable, and the
code comments frame the fast exit as deliberate). The prior review's
single standing finding (IN-01, server.go) is re-verified still present
and carried forward unchanged; it lives in a file outside this review's
four-file scope. No critical issues: no injection surface, no secrets, no
unsafe defaults (metrics TLS is RequireAndVerifyClientCert + TLS 1.2
floor; plaintext mode is the documented loopback/debug shape; always-on
reflection is a documented decision consistent with the trusted-LAN
threat model and loopback default bind).

## Warnings

### WR-01: Gate resolver can flip health back to SERVING mid-drain — apiCancel sits after the metrics shutdown that widens the race window

**File:** `cmd/sup-kv/main.go:645-654` (race partner: `internal/grpcapi/gate.go:122-156`)
**Issue:** `shutdown` flips the whole-server health status to NOT_SERVING
(line 645) as D-19's traffic-shedding step, but `node.apiCancel()` — the
only thing that stops `RunGate` — does not run until line 654, *after* the
metrics-listener shutdown whose context timeout is `shutdownGrace` (10s).
If the SIGTERM arrives before the startup gate has opened (a supervisor
restart loop inside the default 1s window, or any operator-sized
`--startup-gate`), `RunGate` is still selecting during that window and can
call `openGate()` — via the backstop timer firing or a peer's sync
datagram arriving on the 20ms poll — whose CAS succeeds while `gateOpen`
is false and executes `SetServingStatus("", SERVING)`
(gate.go:152-156). The health endpoint then advertises SERVING while the
node is draining, so orchestrators gate on it and keep sending writes into
a node about to GracefulStop/Stop them — the exact traffic D-19's first
step exists to shed. A residual variant survives even after `apiCancel`:
with `ctx.Done` and `backstop.C` both ready, Go's random `select` pick can
run `openGate` after the cancel, flipping SERVING into the GracefulStop
window while health RPCs still answer. Not data loss (writes that land
commit durably per the per-RPC dump confirmation, and the post-Stop
`Flush` covers them), but a direct violation of the documented D-19
contract this same function's comment states.
**Fix:** Make RunGate's exit observable and serialize it before the flip:
have `RunGate` close a `gateDone` channel on every return path
(`defer close(s.gateDone)`), and reorder `shutdown` to
`node.apiCancel()` → `<-node.gateDone` → `SetServingStatus("",
NOT_SERVING)` → metrics shutdown → GracefulStop. Watch streams ending
before the health flip is harmless (clients reconnect; GracefulStop
still comes after). Alternatively, guard `openGate` with a shutdown flag
set before the health flip.

## Info

### IN-01: mapError's status.FromError passthrough branch is unreachable — no internal/app error can be a status error

**File:** `internal/grpcapi/server.go:306-308`
**Category:** necessity
**Issue:** (Carry-forward from the 2026-11-09 deep review and the
2026-11-14 incremental review; re-verified present at the same lines
during this pass. This file is OUTSIDE the current four-file scope —
listed for continuity of the phase's open-findings ledger only.)
Generality without instantiation: mapError's inputs come from the engine,
storage, persist and persistfile layers, none of which may import grpc
(the depguard boundary keeps `google.golang.org/*` out of internal/app),
so no error reaching this switch can be a status error; every real input
resolves through the sentinel table above or falls to `codes.Internal`.
The branch runs, but defends against nothing. It rests on judgment (a
future in-package caller could theoretically construct a status error),
hence Info rather than Warning.
**Fix:** Remove the passthrough branch and let the sentinel table plus
the Internal fallback be the whole contract — or keep it with a comment
naming the concrete future caller that would justify it.

### IN-02: Serve-error exit path skips the final Flush and the graceful drain

**File:** `cmd/sup-kv/main.go:239-242`
**Issue:** When `gs.Serve` fails before any signal, `main` goes straight
to `log.Fatalf` — none of `shutdown` runs: no final write-confirmed
`engine.Flush()`, no adapter drain, no health flip. Consequences are
recoverable rather than lossy: every client-confirmed write is already on
disk (the Set/Delete handlers return only after a dump containing the
record — PERS-04 inside the RPC), and remote-merged-only state re-converges
from peers via periodic sync after restart. But the node exits with its
dump file stale relative to RAM (slower restart convergence, and on a
degraded segment the stale tail persists until the next sync round), and
the D-19 "clean exit reserved for a confirmed dump" discipline has an
undocumented exception. The `serve` comment frames the fast fatal exit as
deliberate ("main decides the verdict"), which keeps this at Info.
**Fix:** Before the `Fatalf`, run the flush tail (or the whole `shutdown`)
so the exit code still signals the fault while the on-disk state stays
current — e.g. `if err := shutdown(node, sig); err != nil { log.Printf(...)
}` prior to `log.Fatalf("sup-kv: serve: %v", err)` — or extend the
`serve` doc comment to state explicitly why the dump tail is skipped on
this path.

---

_Reviewed: 2026-11-18T10:57:01Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
_Delta base: 69bb132 (phase-05 covered files only)_
