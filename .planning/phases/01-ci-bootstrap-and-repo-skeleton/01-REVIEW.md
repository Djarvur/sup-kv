---
phase: 01-ci-bootstrap-and-repo-skeleton
reviewed: 2026-11-17T20:12:43Z
depth: deep
files_reviewed: 6
files_reviewed_list:
  - .gitignore
  - Makefile
  - cmd/sup-kv/main.go
  - docs/setup/branch-protection.md
  - go.mod
  - go.sum
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
  necessity: 0
status: issues_found
---

# Phase 01: Code Review Report (incremental — delta since b68dbd4)

**Reviewed:** 2026-11-17T20:12:43Z
**Depth:** deep
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Incremental deep re-review of the phase-1 owned files as changed since
b68dbd4 ("fix(security): bump toolchain to go1.27.4"). The six files
carried the weight of phases 5–7 landing on top of the phase-1 skeleton:
`cmd/sup-kv/main.go` grew the full service wiring (slog pipeline, metrics
listener with mTLS, reflection, the P1-IN-01 signal-ownership fix),
`go.mod`/`go.sum` absorbed the grpcurl + goreleaser tool directives (the
goreleaser graph is why the indirect block roughly tripled), `Makefile`
gained `bench`/`release`/`release-snapshot`, `.gitignore` anchored
`/dist/` for goreleaser, and `branch-protection.md` recorded the 2026-11-17
D-11 re-apply attempt.

**Deep verification performed (all hold):**

- **Build/module integrity, live-verified:** `go build ./...` exit 0,
  `go vet ./...` (compiles every test file) exit 0, `go mod tidy -diff`
  empty — go.mod/go.sum are tidy-consistent with the source tree, so the
  CI tidy gate is not compensating for drift.
- **go.mod policy compliance:** all seven tool directives are genuinely
  consumed — buf (Makefile `proto`/`buf-*`), golangci-lint (`lint`),
  goreleaser (`release`/`release-snapshot`), govulncheck
  (vulncheck.yml:30), protoc-gen-go/-go-grpc (both buf.gen templates),
  grpcurl (docs/EXAMPLES.md:22-24 "go tool grpcurl"). Direct requires
  match the CLAUDE.md pins exactly (grpc v1.85.0, protobuf
  v1.36.13-0.20261016172727-e9ee2a76ab8e). The `go 1.27.4` bump is the
  GO-2026-6859 fix commit. The `pgregory.net/rapid v1.2.0` go.sum entry is
  graph-required (tidy keeps it), and the "rapid" hits in
  internal/app/repl/restart_test.go are English words, not imports.
- **main.go cross-file traces (every callee contract checked):**
  `persist.Open`'s 6-arg signature matches; `Engine.Run` returns `ctx.Err()`
  exactly as shutdown's `errors.Is(runErr, context.Canceled)` check assumes;
  `Engine.Flush` runs before `engineCancel` so the flushCh request is served
  by the still-live Run loop; `metrics.Sources` field types match every
  adapter method value in `metricsSources` one-for-one;
  `metrics.NewTLSConfig` enforces `RequireAndVerifyClientCert` + TLS 1.2
  floor (real mTLS, not optional client cert); the D-19 chain
  `apiCancel → hub.Run defer shutdownSubscribers() → Watch handlers return
  → GracefulStop drains` is real in watch.go:184-201, so the
  "GracefulStop blocks forever while any stream is open" premise holds.
- **Knob-range defense chain complete:** main passes int/duration knobs
  through unvalidated, and every consumer normalizes out-of-domain values —
  `repl.Config.normalized()` (repl/core.go:132+, incl. the
  maxSyncResponseRate truncation bound), `persist.Config.normalized()`
  (engine.go:37-43), `storage.Config` ValueMaxBytes<=0→61440
  (store.go:22-24), `ServerOptions` StartupGate/WaitTimeout<=0→defaults
  (gate.go:52,74). No division-by-zero or negative-timer path is reachable
  from main's config surface.
- **Error/exit paths:** `--version` works on a broken environment
  (env error deferred past Parse, as documented); flag parse errors exit 2
  via ExitOnError, matching `exitUsage`; `errors.Join` preserves
  `errors.Is` matching for the sentinels; `checkMetricsCerts` closes the
  half-set path before any bind; `listenMetrics` closes the inner listener
  on TLS-config failure; `abortWire` compensates the listener on the
  post-listenGRPC failure path.
- **Makefile:** all 10 targets in `.PHONY`; the three `bench` packages and
  all three named benchmarks exist (verified by grep);
  `-run '^$$'` is the correct empty-anchor exclusion; `.goreleaser.yaml`
  and `.github/workflows/release.yml` referenced by the release comments
  both exist.
- **.gitignore `/dist/` anchoring:** verified safe — the only nested `dist`
  directory in the tree lives inside the separately-ignored `.gomodcache/`,
  so nothing previously ignored becomes visible.
- **branch-protection.md:** the five required-check contexts the doc names
  are byte-identical to `docs/setup/ruleset.json` (lint, gate
  (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15),
  gate (darwin-amd64-cross)) and to ci.yml's matrix; ruleset name
  "protect master" matches; the new D-11 block is internally consistent,
  and its refusal to run the enforcement probe on the 403 path is correct
  reasoning (it would land a real commit on master).

**Standing findings from the prior review — all three RESOLVED, verified
against the current tree:** WR-01 (no CI job timeout-minutes) is fixed —
ci.yml:21 (15m lint), ci.yml:57 (20m gate), release.yml:24 (20m), each
tagged `P1-WR-01`; IN-07 (stale `buf.gen.yaml` references in
.claude/CLAUDE.md) — grep returns zero hits; IN-08 (buf.yaml's false
"one pass, both trees" comment) — the comment now states a bare
`buf generate` fails and names the per-tree `make proto` flow.

New findings: one Warning (a signal-handling window in main.go — the same
class as the P1-IN-01 regression the file itself documents) and one Info
(exit-code collision). No Critical issues. No necessity findings: every
guard in the delta guards a reachable state (env/flag trust boundaries and
cross-goroutine races), and main duplicates no validation its callees own.
No structural-findings pre-pass and no external reviewer evidence were
provided for this run; all findings are narrative (direct review).

## Narrative Findings (AI reviewer)

## Warnings

### WR-02: SIGTERM between process start and armSignals() bypasses the graceful-shutdown path entirely

**File:** `cmd/sup-kv/main.go:229-238`
**Issue:** `armSignals()` (main.go:238) runs only after `wire(cfg)`
(main.go:229) returns. During wire — dump-chain load (`persist.Open` reads
the fallback `.prev` generations, potentially seconds on a large state),
the UDP bind, and the adapter's `Run` goroutine already receiving peer
merges — the process still holds the default SIGTERM/SIGINT disposition,
so a signal in that window kills the process with no final dump and no
shutdown sequence. The file's own comments establish this exact class as a
real regression risk: the P1-IN-01 fix (69d1e47) closed the equivalent
hole at the serve→shutdown boundary because "a second signal during the
graceful drain … instead of killing the process with the default
disposition and forfeiting the final dump" — yet the startup side of the
same boundary is open. Impact is bounded (no client-confirmed writes can
exist before `serve`, and peer-merged state is recoverable from peers by
CRDT re-merge), but a supervisor doing fast stop/start loops — exactly the
deployment the D-19 exit-code contract targets — deterministically hits
the ungraceful path, and the fix is free: `signal.Notify` is
level-triggered with a buffer, so arming before `wire()` costs nothing and
`serve`'s select simply consumes the buffered signal later.
**Fix:**
```go
setupLogging(cfg)

sig := armSignals() // arm before wire: a startup-window SIGTERM must take the graceful path too
node, listener, err := wire(cfg)
if err != nil {
	log.Fatalf("sup-kv: %v", err)
}
err = serve(node, listener, sig)
```
(signal ownership then genuinely spans the process's whole life, matching
the armSignals doc comment; extend that comment's "serving and the D-19
shutdown alike" to include wiring.)

## Info

### IN-09: log.Fatalf paths exit 1 — indistinguishable from exitDumpFailed in the documented exit-code contract

**File:** `cmd/sup-kv/main.go:66-73, 231, 241`
**Issue:** The const block frames a three-way exit contract: 0 = clean
shutdown with confirmed final dump, 1 (`exitDumpFailed`) = failed final
dump, 2 (`exitUsage`) = configuration error. But the two `log.Fatalf`
calls on the wire (main.go:231) and serve (main.go:241) failure paths also
exit 1 via the log package's default. A supervisor reading the documented
contract cannot distinguish "final dump failed — data at risk, investigate
the dump file" from "node never came up (bind conflict)". The durability
verdict the exit code exists to carry gets blurred exactly on the path
where it matters most.
**Fix:** Give fatal startup/serve failures their own code, e.g.
`exitFatal = 3`, replacing the two `log.Fatalf` calls with
`log.Printf` + `os.Exit(exitFatal)` (which also makes main's
"main owns every exit" comment literally true), or amend the const-block
doc to state that 1 covers every non-usage fatality.

---

_Reviewed: 2026-11-17T20:12:43Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
_Delta: b68dbd4..HEAD (incremental re-review of phase-1 owned files)_
