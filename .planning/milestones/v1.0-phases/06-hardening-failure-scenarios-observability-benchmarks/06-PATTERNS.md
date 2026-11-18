# Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks - Pattern Map

**Mapped:** 2026-11-15
**Files analyzed:** 20 (9 new, 11 modified)
**Analogs found:** 17 / 20 (3 first-of-kind: HTTP listener/exporter, slog wiring, benchmarks)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/metrics/metrics.go` (NEW) | adapter (exporter + mux + TLS) | request-response | `internal/persistfile/dumpfile.go` | role-match (first HTTP code in repo) |
| `internal/metrics/metrics_test.go` (NEW) | test | request-response | `internal/persistfile/dumpfile_test.go` | role-match |
| `internal/udprepl/scenario_test.go` (NEW) | test (integration) | event-driven | `internal/grpcapi/wait_integration_test.go` | exact (asymmetric topology already exists) |
| `internal/app/repl/sim_scenario_test.go` (NEW) | test (simulator) | event-driven (virtual time) | `internal/app/repl/sim_chaos_test.go` | exact |
| `cmd/sup-kv/scenario_test.go` (NEW) | test (exec integration) | process lifecycle + request-response | `cmd/sup-kv/main_test.go` | exact (harness reused verbatim) |
| `internal/app/persist/bench_test.go` (NEW) | test (benchmark) | batch | none (first benchmark in repo) | none |
| `internal/udprepl/bench_test.go` (NEW) | test (benchmark) | streaming | `internal/grpcapi/wait_integration_test.go` | partial (setup shape only) |
| `internal/grpcapi/bench_test.go` (NEW) | test (benchmark) | request-response | `internal/grpcapi/wait_integration_test.go` | exact (latency measurement shape at lines 261-278) |
| `internal/persistfile/fsync_darwin.go` (NEW) | utility (platform syscall) | file-I/O | `internal/persistfile/dumpfile.go` | role-match |
| `cmd/sup-kv/main.go` (MOD) | config/wiring | request-response | itself — extend cited sites | exact |
| `internal/udprepl/repl.go` (MOD) | adapter | event-driven | itself — extend cited sites | exact |
| `internal/app/repl/core.go` (MOD) | model (decision core) | event-driven | itself — extend cited sites | exact |
| `internal/app/persist/engine.go` (MOD) | service | batch (serialized dumps) | itself — extend cited sites | exact |
| `internal/app/storage/store.go` (MOD) | model | CRUD | itself (doc-comment fix only) | exact |
| `internal/app/storage/store_test.go` (MOD) | test | CRUD | itself (subtest deletion) | exact |
| `internal/app/internal/simnet/simnet.go` (MOD) | test infrastructure | event-driven | itself — extend NewCluster clock wiring | exact |
| `Makefile` (MOD) | config | batch | itself — `test` target convention | exact |
| `.github/workflows/ci.yml` (MOD) | config | batch | itself — jobs `lint`/`gate` | exact |
| `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` (MOD) | docs | — | existing TEST-04 rows | exact |
| `.planning/phases/0N-*/0N-REVIEW-DISPOSITION.md` (MOD) | docs | — | existing disposition rows | exact |

**Verified:** all analog paths above are git-tracked (`git ls-files` non-empty for every path).

**Load-bearing discovery:** the asymmetric-port two-node topology (RESEARCH Pattern 5, required for ALL new two-node scenarios and the 4 exec scenarios) already exists, probed and documented, in `internal/grpcapi/wait_integration_test.go` — including the cross-broadcast construction AND a one-port capability probe. Copy that file's topology, NOT the shared-port smoke-test shape.

## Pattern Assignments

### `internal/metrics/metrics.go` (adapter, request-response) — NEW

**Analog:** `internal/persistfile/dumpfile.go` (adapter-over-stdlib-I/O conventions: package doc stating the adapter role, small exported surface, `var _ port.Seam = ...` conformance assertion); listener-shape analog `cmd/sup-kv/main.go` `listenGRPC`. No HTTP server exists in the repo — rendering/mux/TLS building blocks come from RESEARCH Pattern 2/3 (verified stdlib signatures).

**Adapter package conventions** (`internal/persistfile/dumpfile.go:1-19`):
```go
// Package persistfile is the sup-kv dump file adapter: the only
// os-importing persistence code, implementing port.DumpStore with atomic
// writes, .prev rotation and the fallback chain (D-01/D-02). ...
import (
	"os"
	"path/filepath"
	"sync"

	"github.com/Djarvur/sup-kv/internal/port"
)

// FileStore implements the whole-dump persistence seam.
var _ port.DumpStore = (*FileStore)(nil)
```
Copy this shape: a package doc that names the adapter boundary and the decision it do NOT own; a struct with a mutex only where genuinely needed; explicit conformance assertion if a port seam applies.

**Listener lifecycle analog** (`cmd/sup-kv/main.go:360-369`):
```go
func listenGRPC(grpcAddr string, srv *grpcapi.Server) (net.Listener, *grpc.Server, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen tcp %s: %w", grpcAddr, err)
	}
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	return listener, gs, nil
}
```
The metrics listener follows this error-wrapping/listen-then-serve split, plus D-01 (flag unset ⇒ NO listener, no goroutine) and Pitfall 5 (explicit `http.Server{ReadHeaderTimeout: ...}`, never `http.ListenAndServe` — gosec runs with `audit: true`).

**Mux/TLS assembly** — use RESEARCH Pattern 3's verified block (`tls.LoadX509KeyPair` + `tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs, MinVersion: tls.VersionTLS12}`; named imports of `expvar`/`net/http/pprof`, never blank imports; `pprof.Index` at the `/debug/pprof/` prefix covers named profiles).

**Exporter contract** (D-04/D-05): ~100-line pure formatter over a fixed family snapshot; sorted families and labels IN THE WRITER; `Content-Type: text/plain; version=0.0.4; charset=utf-8`; TYPE before first sample; escaping per RESEARCH Pattern 2. No registry framework (RESEARCH "Don't Hand-Roll").

**Test analog** for `metrics_test.go`: table-driven unit test pinning exact rendering with adversarial HELP text (Pitfall 3), plus `httptest`-over-the-mux integration for OBS-02 classes; in-test `crypto/x509.CreateCertificate` + `crypto/ecdsa` certs, no committed fixtures.

---

### `cmd/sup-kv/main.go` (config/wiring) — MODIFIED

**Analog:** itself. Four extension sites:

**1. New flags/env — copy the registerConfig pattern** (`cmd/sup-kv/main.go:194-253`, key excerpt 201-253):
```go
	envDur := func(env string, def time.Duration) time.Duration {
		v, err := envDuration(env, def)
		fail(env, err)
		return v
	}
	envNum := func(env string, def int) int {
		v, err := envInt(env, def)
		fail(env, err)
		return v
	}
	...
	flag.DurationVar(&cfg.dumpDebounce, "dump-debounce", envDur(envDumpDebounce, defaultDumpDebounce),
		"dump quiet window (env SUPKV_DUMP_DEBOUNCE)")
```
New knobs (`--metrics-addr`/`SUPKV_METRICS_ADDR` string via `envOrDefault`, `SUPKV_LOG_LEVEL`/`SUPKV_LOG_FORMAT` strings) follow exactly: env-name const block (lines 75-95), default const block (99-115), config struct fields (118-138), flag registration with the `(env SUPKV_X)` doc suffix. Malformed `SUPKV_LOG_LEVEL` = collected env error → exit 2 via `finalizeConfig` (lines 257-269). Incomplete cert triple validation belongs adjacent to `missingRequired` (432-447), BEFORE any bind (Pitfall 8).

**2. slog construction** — first slog code in repo (no analog; RESEARCH Pattern 4 verified signatures). Build once in main after `finalizeConfig`, `slog.SetDefault`, output `os.Stderr` (D-07).

**3. The Log-tap swap point** (`cmd/sup-kv/main.go:310`):
```go
		Log:                func(msg string) { log.Printf("%s", msg) },
```
becomes the slog bridge. sloglint constraint (`.golangci.yml:357-366`: `static-msg: true`, `msg-style: capitalized`, `context: scope`, forbidden keys time/level/msg/source, `key-naming-case: snake`): the tap's prebuilt strings ride as an attr (`slogger.Warn("Replication adapter", "detail", msg)`), NEVER as the msg.

**4. P1-IN-01 fix site — serve()'s early signal release** (`cmd/sup-kv/main.go:374-385`):
```go
func serve(node *wired, listener net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- node.gs.Serve(listener) }()
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop() // <-- releases signal handling before shutdown() runs: a second SIGTERM during GracefulStop kills with exit 143, no final dump
	select {
	case <-signalCtx.Done():
		return nil
	case err := <-serveErr:
		return err
	}
}
```
Fix: keep signal ownership alive across `shutdown(node)` (stop() moves after shutdown completes / second-signal triggers the existing `gs.Stop()` force path — Open Question 4 recommends (b)).

**Metrics listener lifecycle:** joins `wire()`/`shutdown()` (297-428) — `srv.Shutdown(ctx)` positioned in the D-19 sequence (recommended right after health NOT_SERVING, Pitfall 7); `wired` struct (142-151) gains the handles.

---

### `internal/udprepl/repl.go` (adapter, event-driven) — MODIFIED

**Analog:** itself. All new observability rides existing seams (D-09).

**Mirror pattern — THE precedent for every new counter** (`internal/udprepl/repl.go:449-457`, called from Run's loop at 300-301):
```go
func (a *Adapter) mirrorStats() {
	d := a.core.Discards()
	a.badCRC.Store(d.BadCRC)
	a.undecodable.Store(d.Undecodable)
	a.badProtoVersion.Store(d.BadProtoVersion)
	a.ownEcho.Store(d.OwnEcho)
	a.duplicateNodeID.Store(d.DuplicateNodeID)
	a.peerCount.Store(uint64(len(a.core.PeerSnapshot())))
}
```
The load-bearing shape: struct atomics (repl.go:147-152) + one post-event mirror call in Run's loop + a copying read accessor. New mirrors in the same sites:
- **UnackedStats mirror:** in `mirrorStats`/`mirrorState`, `a.core.UnackedStats(time.Now())` → new atomics; read accessor copies `Discards()` (343-351) exactly.
- **Sent-by-type counters:** count in `flush` (418-436) where the adapter already sees every `port.Message` (switch on `ob.Msg` type — Hello/Data/Ack/DataRequest/DataSync, mirroring the `protoBody` switch 636-661).
- **Retransmitted:** core-side only (`scanRetransmit`); mirror like DiscardStats.

**P4-WR-01 fix site — drained-row Acked duplicate** (`internal/udprepl/repl.go:513-514`):
```go
		acked := append(slices.Clone(prev.Acked), prev.WaitingOn...)
		slices.Sort(acked)
```
After a discovery flush re-adds an evicted peer to `waitSet` (core.go:433-440), the union can hold the peer twice. Fix: `slices.Compact(acked)` after the sort.

**P4-IN-01 fix site — NotifyLocalWrite TOCTOU** (`internal/udprepl/repl.go:328-339`): the pre-check select then enqueue select races loop exit; the fix keeps the closed-loop deterministic contract documented at 305-313 (never silently drop).

**Adapter doc comment** (repl.go:7-11) already documents the shared-port unicast limitation — new scenario tests cite it and use the asymmetric topology instead.

---

### `internal/app/repl/core.go` (model, event-driven) — MODIFIED

**Analog:** itself. The core gains counters and (optionally) one Config seam — never I/O (depguard `sup-kv-app-no-io` denies net/os/syscall/google.golang.org in `internal/app/**`).

**Counter precedent** (`internal/app/repl/core.go:166-172` + 315-322 + 884-888):
```go
type DiscardStats struct {
	BadCRC          uint64
	Undecodable     uint64
	BadProtoVersion uint64
	OwnEcho         uint64
	DuplicateNodeID uint64
}
...
	case port.BadCRC:
		c.discards.BadCRC++
		return
...
func (c *Core) Discards() DiscardStats {
	return c.discards
}
```
New `MergeStats` (conflicts + drift warnings) copies this triple: struct + increment site + copying accessor. Insertion sites:
- **Merge conflicts:** `applyRemote` line 455 — `_, _, err := c.store.ApplyRemote(msg.Key, msg.Record)` discards the conflicts return today; count `len(conflicts)`.
- **Drift warnings:** same site — the `drift` return gated by the drift-warn threshold (constructor knows it; mirror how `clock` gates the signal — see `storage.Merge` store.go:211-221 which returns the drift).

**Config seam precedent** (`internal/app/repl/core.go:95` + `normalized` 125-128) — for the OnDiscard decision (Open Question 1, recommended `OnDiscard func(class, detail string)`):
```go
type Config struct {
	...
	Log                func(msg string)
}
...
	if c.Log == nil {
		c.Log = func(_ string) {}
	}
```
The discard branches that go silent today: HandleDatagram 315-339 (BadCRC/Undecodable/BadProtoVersion/OwnEcho return without logging; DuplicateNodeID logs via `cfg.Log` with `fmt.Sprintf` at 334-337, 341-345).

**P4-WR-02 fix site — rate clamp** (`internal/app/repl/core.go:122-124` + 285):
```go
	if c.SyncResponseRate <= 0 {
		c.SyncResponseRate = defaultSyncResponseRate
	}
	...
		syncResponseInterval: time.Second / time.Duration(cfg.SyncResponseRate),
```
A rate above 1e9 makes the division truncate to 0 and silently disables the limiter. Fix in `normalized()`: clamp out-of-domain values (not just <= 0) to `defaultSyncRespRate`. The `floor(1s/rate)` arithmetic is pinned by ladder/spacing tests — regression-test the clamp (Pitfall 10).

---

### `internal/app/persist/engine.go` (service, batch) — MODIFIED

**Analog:** itself. Dump-stats atomics + accessors (lock-free for scrapers) are new state on Engine (58-77); RESEARCH recommends atomics since `runDump` is the single serialized dump site.

**Dump stats insertion site** (`internal/app/persist/engine.go:266-284`):
```go
func (e *Engine) runDump() error {
	e.mu.Lock()
	cover := e.lastLocal
	captured := e.waiters
	e.waiters = nil
	e.dirty = false
	e.mu.Unlock()

	block, err := Encode(e.store.Records())
	if err == nil {
		err = e.dumps.Save(block)
	}
	if err != nil {
		e.failWaiters(captured)
		return err
	}
	e.confirmWaiters(captured, cover)
	return nil
}
```
Time the Encode+Save span; count total/errors. NOTE: `internal/app` may not import `sync/atomic`? — it may (atomic is stdlib, not on the deny list net/os/syscall/google.golang.org); plain `sync` already imported. Counters stay here; the metrics adapter reads the accessor cross-goroutine — Engine is already documented safe for concurrent use.

**D-15 fix sites in this file:**
- **P3-WR-01** `confirmWaiters` 290-305: the above-cover guard (`if w.ver.Compare(cover) > 0` keeps the waiter) — KEEP the documented belt AND add an invariant assertion; do not delete.
- **P3-IN-01** `Open` 110-116: dead Decode-error fallback (`if err == nil { records, discarded = decoded, count }` — the else path leaves records nil = empty start; the finding calls the fallback dead).
- **P4-IN-02** `Restore` line 497 `e.markDirty()` unconditional — merge with:
- **P3-IN-02** `ApplyRemote` line 429 `e.markDirty()` on no-op merges — one redundant-dirty item: mark dirty only when the merge actually changed state (ApplyRemote's conflicts return is the signal; `markDirty` at 580-587). PERS-04 confirmation semantics must not regress.

---

### `internal/udprepl/scenario_test.go` (test, event-driven) — NEW

**Analog (primary):** `internal/grpcapi/wait_integration_test.go` — the asymmetric-port topology already exists, probed and documented. Package doc (lines 36-41):
```go
// The two nodes bind DIFFERENT loopback ports and target each other's port
// as their broadcast address: broadcast Hello discovery crosses over, while
// every unicast — the peer's Ack above all — is delivered to a port exactly
// one socket holds. The adapter's same-host caveat (unicast to a port
// shared by SO_REUSEADDR sockets is the kernel's choice) therefore never
// applies, and the Ack path this test asserts is deterministic.
```
Construction (lines 170-213): `portA, portB := integrationUDPPort(t), integrationUDPPort(t)`; adapter A `("0.0.0.0:P1", "127.255.255.255:P2")`, adapter B mirrored. Its `probeLoopbackBroadcast` (110-142) is the capability-probe shape for the D-12 darwin loud-skip (asymmetric topology removes the EADDRINUSE reason but not lo0-no-BROADCAST).

**Analog (secondary):** `internal/udprepl/smoke_test.go` — the loud-skip discipline and waitFor helper:
- `waitFor` + `hardDeadlineMargin` (smoke_test.go:190-205) — poll cadence 10ms, deadline clamped ahead of the test's own hard deadline. Copy verbatim.
- Loud skip shape (smoke_test.go:213-257 `probeBroadcastAddr`): `t.Skipf` naming exactly what was probed and why the platform lacks it; never `t.Skip()` bare.
- `smokeStore` StateStore wrapper (177-185) for the peer node.
- Hand-crafted bad-CRC datagrams: `encodeDatagram`/`decode` (repl.go:571-630) show the CRC field to corrupt.

Scenarios: duplicate node_id, proto mismatch, bad CRC, convergence — driven against `Discards()` classes and `KnownPeers()`/`PendingCopy()` mirrors.

---

### `internal/app/repl/sim_scenario_test.go` (test, virtual-time) — NEW

**Analog:** `internal/app/repl/sim_chaos_test.go` — the sim-scenario skeleton. Key shapes:
- Scenario struct drawn before any event runs (87-93, 101-124) — "every random input fixed before any event runs, so the whole run is a pure function of the rng sequence".
- Partition predicate as a `simnet.Policy.Partition func(from, to, handle, now)` closure (128-138) — timed and one-directional partitions expressible directly.
- Run driver: `sched.RunUntil(at(...))` stepping + `cluster.Write` (159-201).
- Quiescence via the three seams (206-216): `PendingDeliveries() == 0`, `UnackedStats(now).Count == 0`, `QueuedResponses() == 0`.
- Convergence model: pointwise-supremum store over the write log, joined by `Merge` (222-228); compare with `recordMapsEqual`.
- Replay hook: `SUPKV_SIM_SEED` + printed replay command (29-56).

**simnet.go modification (clock-offset seam):** `NewCluster` hardwires `clock.NewClock(sched.Now, 0, 0, clusterDriftWarn)` (simnet.go:374) — the wall-clock-jump scenarios need a per-node mutable offset behind the `now func() time.Time` seam (either a NewCluster variant or a Cluster method; offsets mutate only from test code — determinism preserved). `simRunID` (329-333) is construction-fixed: a sim restart = replacement node registered at the same handle (restart_test.go core-level precedent at `internal/app/repl/restart_test.go:97-190`).

---

### `cmd/sup-kv/scenario_test.go` (test, exec) — NEW

**Analog:** `cmd/sup-kv/main_test.go` — the harness exists verbatim; EXTEND, don't rebuild (RESEARCH "Don't Hand-Roll"). Reusable pieces (all in `package main_test`, same package as the new file — helpers are directly callable):
- `TestMain` build-once (63-95), `smokeBinary` loud-skip (101-111)
- `freeTCPPort`/`freeUDPPort` (115-136)
- `waitFor` (141-156)
- `startNode`/`reap`/`signalAndWaitExit`/`assertExitCode` (171-251) — kill -9 is `node.cmd.Process.Kill()` then `assertExitCode(t, exitErr, ...)` for the signal case
- `awaitServing` + `supKVConn` over real gRPC (256-280)
- `assertNoPanic` (284-290)
- Full lifecycle precedent with SIGTERM + exit 0 + dump grep (373-420)

The 4 scenarios (D-11) pass asymmetric-port flags (`--bind-addr=0.0.0.0:P1 --broadcast-addr=127.255.255.255:P2` to A, mirrored to B — copy the pattern from `TestMain_precedence`'s arg list 335-341) plus shrunken timers (`--startup-gate=10ms` precedent at 340; add `--hello-interval=100ms --sync-interval=200ms`). Exit-code constants to assert: `exitDumpFailed = 1`, `exitUsage = 2` (main.go:63-66). Darwin loud-skip: reuse the broadcast capability-probe shape (D-12).

---

### Benchmark files (NEW) — no codebase analog

| File | Benchmark | Closest existing shape |
|------|-----------|------------------------|
| `internal/app/persist/bench_test.go` | `BenchmarkDump10kKeys` | none — store seeding via `storage.NewStore` + clock fake (sim_chaos_test.go:222-228); dump through real `persist.Encode` + `persistfile.New(b.TempDir()).Save` (engine.go:274-276 path) |
| `internal/udprepl/bench_test.go` | `BenchmarkReplicationBurst` | two-adapter asymmetric setup: wait_integration_test.go:170-218; convergence poll via `KnownPeers()`/`PendingCopy()` |
| `internal/grpcapi/bench_test.go` | `BenchmarkConfirmationLatency` | the Set-with-WaitForReplication + elapsed measurement: wait_integration_test.go:261-278 (`start := time.Now(); ... elapsed := time.Since(start)` + acked-report assertion) |

Verified: zero `func Benchmark` exists in the repo — these are first-of-kind; follow RESEARCH Pattern 8 (`b.ReportMetric(n float64, unit string)`, no whitespace in unit strings, stable names, no thresholds — D-13/D-14).

### `internal/persistfile/fsync_darwin.go` (utility, file-I/O) — NEW

**Analog:** `internal/persistfile/dumpfile.go` fsync discipline (137-143 `writeSynced`, 167-174 `syncDir`):
```go
func writeSynced(tmp *os.File, data []byte) error {
	_, err := tmp.Write(data)
	if err != nil {
		return err
	}
	return tmp.Sync()
}
```
The darwin file (~15 lines, build-tagged `//go:build darwin`) wraps the fd in `syscall.Syscall(syscall.SYS_FCNTL, fd, F_FULLFSYNC, 0)`; syscall is depguard-legal here (adapter package — `internal/udprepl/repl.go:23` already imports syscall). Existing sanctioned-nolint precedent for this package: `//nolint:gosec // G304: candidate derives from the configured dump path, never caller input` (dumpfile.go:76) — every new nolint names its linter with a reason.

### `Makefile` / `.github/workflows/ci.yml` (config) — MODIFIED

**Makefile bench target** — copy the `test` target convention (Makefile:15-17), comment-header style included:
```make
## test: race-detector tests with coverage.
test:
	go test -race -cover ./...
```
`bench:` gets a `## bench:` doc comment, `.PHONY` addition (line 1), scoped `-run='^$' -bench=.` over the three packages; NOT in CI.

**ci.yml timeout-minutes (P1-WR-01)** — add `timeout-minutes: 15` to job `lint` (ci.yml:19) and `timeout-minutes: 20` to job `gate` (ci.yml:53). CONSTRAINT (file header lines 1-3): check-run names `lint` and `gate (<cell>)` are a ruleset contract — the matrix `name:` fields (60-67) MUST NOT change.

### Docs modifications

- `.planning/REQUIREMENTS.md` TEST-04 row + `.planning/ROADMAP.md` Phase 6 success criteria: replace conflict-journal wording with "deterministic winner asserted + merge-conflicts counter" (D-10).
- `.planning/phases/0{1..4}-*/0N-REVIEW-DISPOSITION.md`: each D-15 fix flips its row to fixed in the same PR.

## Shared Patterns

### Adapter atomic mirrors (counters cross goroutines)
**Source:** `internal/udprepl/repl.go:147-152` (fields), `449-457` (post-event mirror in Run's loop), `343-351` (copying accessor)
**Apply to:** every new OBS metric touching core state — UnackedStats mirror, sent-by-type, retransmitted, merge conflicts, drift warnings. Counter lives in core (incremented at the decision site), atomic mirror refreshed after every event-loop iteration, accessor safe from any goroutine.

### Core copying accessors (results, never callbacks)
**Source:** `internal/app/repl/core.go:884-948` (`Discards`, `SyncArrived`, `PeerSnapshot`, `PendingSnapshot`) and `969-983` (`UnackedStats(now)` — the now-parameter seam)
**Apply to:** new `MergeStats`-style counters in repl core and dump-stats accessors in the persist engine. Value copies, sorted keys where iteration is observable, no locks (single driver goroutine).

### Config seam normalization
**Source:** `internal/app/repl/core.go:95, 100-129` — `Log func(msg string)` field, nil-normalized to no-op in `normalized()`
**Apply to:** the OnDiscard seam decision (if adopted). Storage's OnChange (`storage/store.go:42`) is the same precedent with the re-entrancy doc the P2-IN-01/02 fix rewrites.

### Env/flag resolution (flag > env > default)
**Source:** `cmd/sup-kv/main.go:459-507` (`envOrDefault`, `envUint16`, `envInt`, `envDuration`) + `registerConfig` (194-253) + collected-error abort (196-200, 252)
**Apply to:** `--metrics-addr`/`SUPKV_METRICS_ADDR`, `SUPKV_METRICS_CERT/KEY/CLIENT_CA`, `SUPKV_LOG_LEVEL/FORMAT`. Malformed values are startup errors (exit 2), never silent defaults. Incomplete cert triple = startup error before any bind.

### Loud capability skip
**Source:** `internal/udprepl/smoke_test.go:213-257` (`probeBroadcastAddr` t.Skipf naming probed addresses + platform reason + where correctness stays proven); `cmd/sup-kv/main_test.go:101-111` (`smokeBinary`); `internal/grpcapi/wait_integration_test.go:110-142` (`probeLoopbackBroadcast` — the asymmetric one-port variant)
**Apply to:** every new real-socket scenario and all 4 exec scenarios on darwin (D-12). No build-tag hiding of tests.

### Poll-with-margin waiting
**Source:** `cmd/sup-kv/main_test.go:141-156` / `internal/udprepl/smoke_test.go:190-205` (`waitFor`: 10ms poll, deadline clamped to test deadline minus 250ms margin)
**Apply to:** all scenario tests and benchmark convergence polls. Assert bounds, never wall-clock precision.

### sloglint-compliant logging (NEW code only)
**Source:** `.golangci.yml:357-366` — `static-msg: true`, `msg-style: capitalized`, `context: scope`, forbidden keys time/level/msg/source, `key-naming-case: snake`
**Apply to:** all new slog calls (main wiring, udprepl tap bridge, discard-class events). Static capitalized msg + dynamics in snake-case attrs (`class`, `detail`). D-09: NO logging in `internal/app/**`.

### Depguard boundary
**Source:** `.golangci.yml:70-80` (`internal/app/**` denies net/os/syscall/google.golang.org prefix matches; `internal/port` stdlib+own-module only)
**Apply to:** exporter, HTTP listener, TLS, slog construction, F_FULLFSYNC — ALL outside internal/app. Core gets counters only.

### Error sentinels + errors.Is
**Source:** `cmd/sup-kv/main.go:51-57` (sentinel vars) + `internal/persistfile/dumpfile.go`'s error returns; exit codes `main.go:63-66`
**Apply to:** metrics/cert validation errors (exit 2 path), scenario assertions on exit codes.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/metrics/metrics.go` (HTTP listener + Prometheus rendering) | adapter | request-response | First HTTP server code in the repo (verified: no `net/http` import anywhere). Use RESEARCH Pattern 2/3 verified stdlib blocks; adapter conventions from persistfile. |
| slog handler construction in `cmd/sup-kv/main.go` | config | — | First slog usage in repo (verified). Use RESEARCH Pattern 4 verified signatures + sloglint constraints above. |
| `internal/app/persist/bench_test.go` | test | batch | First benchmark in repo (verified: no `func Benchmark` anywhere). Use RESEARCH Pattern 8 + `b.ReportMetric` API. |

## Metadata

**Analog search scope:** `cmd/sup-kv/`, `internal/` (app, grpcapi, persistfile, port, udprepl), `Makefile`, `.github/workflows/ci.yml`, `.golangci.yml` — full project source tree (23 non-generated .go files + configs); `.gomodcache/` excluded.
**Files scanned:** 16 analog candidates read (all git-tracked, verified via `git ls-files`); ~5,600 lines.
**Pattern extraction date:** 2026-11-15
