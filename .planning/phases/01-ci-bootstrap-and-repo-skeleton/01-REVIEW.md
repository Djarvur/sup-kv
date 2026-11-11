---
phase: 01-ci-bootstrap-and-repo-skeleton
reviewed: 2026-11-11T10:55:35Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - .claude/CLAUDE.md
  - cmd/sup-kv/main.go
  - .github/workflows/ci.yml
  - .gitignore
  - go.mod
  - go.sum
  - internal/port/doc.go
  - Makefile
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
  necessity: 0
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-11-11T10:55:35Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Incremental re-review of phase 01's own key files as they stand after later
phases evolved them (commit adddd0b → HEAD: ci.yml +20 lines, Makefile +43,
main.go +506, go.mod/go.sum tool-graph growth, doc.go reworded). The lens:
phase 01's contract — zero unsanctioned direct runtime requires, the vendored
strict `.golangci.yml` untouched upstream, Makefile lint/test/build intact,
ci.yml matrix and hygiene intact, doc.go/package comments in place — as
amended by the phases that legitimately grew these files.

**Phase 01 contract verification (all hold):**

- `go.mod` — direct runtime requires are exactly the sanctioned pair
  `google.golang.org/grpc v1.85.0` and
  `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e`
  (AGENTS.md documents gRPC/protobuf entering in their first-use phases;
  `google/uuid` and `golang.org/x/sync` remain indirect). Five tool
  directives, all sanctioned (buf, golangci-lint, govulncheck,
  protoc-gen-go, protoc-gen-go-grpc); `go 1.27.1` matches the AGENTS.md pin.
- `.golangci.yml` (checked as contract background, outside the formal file
  list) — the `# Origin: powerman/golangci-lint-strict version 2.15.0`
  header is intact and project additions are confined to the additive
  depguard rules (`sup-kv-app-no-io`, `sup-kv-port-stdlib-only`).
- `Makefile` — lint/test/build targets intact, recipes tab-indented, all
  targets `.PHONY`-declared; the D-18 ldflags stamping matches main.go's
  `version`/`commit` vars exactly.
- `ci.yml` — the 4-cell matrix, check-run names ("lint", "gate (<cell>)"),
  `permissions: contents: read`, and the concurrency group are intact. The
  D-08 buf-breaking guard was verified live against this tree: the guarded
  path `internal/udprepl/replproto/supkv.proto` matches the real file on
  disk and is absent from `origin/master`, so the deferred branch is the
  correct behavior today and the gate self-activates when the proto lands.
- `internal/port/doc.go` — package comment intact and now accurate
  (`repl.go` and `dump.go` exist; the described seams match the code).
- `go.sum` — 1154 lines, no duplicate module@version entries; both direct
  requires carry h1:/go.mod pairs.
- Anti-pattern sweep across all 8 files (secrets, dangerous functions,
  debug artifacts, empty catch): clean.

**main.go cross-referencing (standard depth):** every cross-package call was
verified against the real signatures — `persist.Open` argument order,
`Engine.Run`/`Flush`, `WatchHub.Run`/`Ingest`, `Adapter.Run`/
`NotifyLocalWrite`/`SyncArrived`/`KnownPeers`/`PendingCopy`,
`NewServer`/`ServerOptions`/`LocalNotify`. All four service goroutines
(engine, adapter, hub, gate) have context-driven exit conditions and
buffered/single-use completion channels — no leaks on any wire() error path
or the shutdown path. `Engine.Run`, `Adapter.Run` and `WatchHub.Run` all
return `ctx.Err()` on cancellation, so main's `_ =` discards and the
`errors.Is(runErr, context.Canceled)` classification are correct, not silent
error swallowing. The health endpoint starts NOT_SERVING at construction
(server.go) and only RunGate opens it, so the 1:1 gate binding holds;
shutdown's shed-traffic-first order is correct. The D-19 shutdown ordering
(health → apiCancel → GracefulStop/grace → repl → Flush → engineCancel) is
deadlock-free: Flush runs while the Run goroutine is still alive. Config
logic in package main is covered by `main_test.go`. The startup-gate
lifecycle, zero-knob downstream normalization (persist/storage/repl all
normalize zero/negative knobs to design defaults), and the
`--version`-works-on-broken-env behavior all check out.

**CI/CD rubric** (`cicd-review-patterns.md`): actions are tag-pinned
(`@v7`/`@v9` — a pin per the rubric's negative space), concurrency guard
present, no deploy steps (deploy checks N/A), no `continue-on-error`, no
static credentials. One hit: missing job timeouts (WR-01).

**Necessity rubric** (`necessity-review.md`): no necessity findings. The env/
flag inputs main.go validates cross a trust boundary (never
impossible-state guards), and the buf-breaking guard defends a state that is
real today (proto absent from origin/master, probe-confirmed on this tree).

No structural-findings pre-pass and no external reviewer evidence were
provided for this run. Findings below are all narrative (direct review); no
Critical issues were found.

## Narrative Findings (AI reviewer)

All findings in this report originate from the direct adversarial review
above. Warnings and Info follow.

## Warnings

### WR-01: Neither CI job sets a timeout-minutes bound — a wedged job holds its runner for GitHub's 6-hour default

**File:** `.github/workflows/ci.yml:19-20` (job `lint`), `.github/workflows/ci.yml:53-55` (job `gate`)
**Issue:** Both jobs declare `runs-on:` but neither sets a job-level
`timeout-minutes:`. A hung `golangci-lint` download, a wedged `-race` test
suite, or a stuck `go tool buf` invocation holds the runner until GitHub's
6-hour default. The matrix multiplies the exposure: `fail-fast: false` keeps
all four gate cells alive precisely so a red cell cannot cancel the others'
evidence, so one wedged cell runs to the 6-hour limit alongside its three
siblings — up to five held runners per pushed commit. Branch-protection
rulesets imply a private/org-plan repo, so that is paid runner minutes.
**Fix:** Bound each job to a generous-but-finite budget:

```yaml
jobs:
  lint:
    runs-on: ubuntu-latest
    timeout-minutes: 15
  gate:
    name: gate (${{ matrix.name }})
    runs-on: ${{ matrix.runs-on || matrix.name }}
    timeout-minutes: 20
```

## Info

### IN-01: Signal handling is released before the D-19 shutdown runs — a second SIGTERM/SIGINT kills the process and forfeits the final write-confirmed dump

**File:** `cmd/sup-kv/main.go:377-378` (serve's `defer stop()`), affected window `cmd/sup-kv/main.go:397-428` (shutdown)
**Issue:** `serve` defers `stop()`, so the moment the first SIGTERM/SIGINT
returns from `serve`, `signal.NotifyContext` is unregistered and the
disposition reverts to default — before `shutdown` starts. During the
shutdown window (up to the 10 s `shutdownGrace` GracefulStop plus the final
`engine.Flush()`), a second SIGTERM/SIGINT — the standard operator move when
a service is slow to stop — terminates the process immediately with exit
143: no final dump, and from the D-19 contract's perspective ("a clean exit
0 is reserved for a shutdown whose final write-confirmed dump completed")
an unconfirmed shutdown that no log line explains. Exposure is bounded —
write-confirmed writes are already on disk via the engine's per-write
`await`, and remote-merged-only dirt recovers from peers on restart
(PERS-02), while the T-05-10 comment shows supervisors SIGKILL through hangs
anyway — which is why this is Info, not a Warning. But the window is cheap
to close, and the project treats the final dump as the shutdown contract.
**Fix:** Keep SIGTERM/SIGINT registered (or ignored) until shutdown
completes — e.g., create the NotifyContext in `main` and release it only
after `shutdown` returns, or re-register the signals as no-ops for the
shutdown duration so only SIGKILL can interrupt the final dump:

```go
signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
defer stop()
// ... serve ...
err = shutdown(node)
// stop() now runs after the final dump, so a second SIGTERM during
// GracefulStop/Flush is swallowed (handled, no action) instead of killing
// the process with the dump incomplete.
```

---

_Reviewed: 2026-11-11T10:55:35Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
