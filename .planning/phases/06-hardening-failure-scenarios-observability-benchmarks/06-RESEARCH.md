# Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks - Research

**Researched:** 2026-11-15
**Domain:** Go observability (in-repo Prometheus text exporter, slog, mTLS debug listener), failure-scenario testing (simulator + real-socket + exec harness), benchmarks, folded code-review fixes
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Metrics endpoint (OBS-01, OBS-02)**
- **D-01:** Opt-in via `--metrics-addr` flag / `SUPKV_METRICS_ADDR` env, NO default — flag unset ⇒ no endpoint, no listener.
- **D-02:** mTLS is optional on the metrics listener: when `SUPKV_METRICS_CERT`/`SUPKV_METRICS_KEY`/`SUPKV_METRICS_CLIENT_CA` are set the listener requires client certificates; with only `--metrics-addr` set it serves plaintext (loopback/debug use). Incomplete cert sets (some flags set, some not) are a startup error.
- **D-03:** One listener serves the whole debug surface: `/metrics` (Prometheus text), `/debug/vars` (stdlib expvar), `/debug/pprof/*` (net/http/pprof) — all behind the same optional mTLS.
- **D-04:** Metric families: the OBS-01/OBS-02 minimum (unacked-records count and oldest age, dump duration and errors, merge conflicts, clock-drift warnings, discarded datagrams by class: bad CRC / own broadcast / bad proto version) PLUS `go_*`/`process_*` runtime families PLUS replication topology (known-peer count, datagrams sent/retransmitted by message type Hello/Data/Ack/Sync/DataRequest). In-repo text-format exporter (~100 lines, sorted families/labels, `text/plain; version=0.0.4`) — `prometheus/client_golang` is rejected (adds 3+ modules against the gRPC+protobuf-only bar; decision recorded in .claude/CLAUDE.md §rejected).
- **D-05:** Naming under the `supkv_` prefix following Prometheus conventions (e.g. `supkv_replication_unacked_records`, `supkv_dumps_duration_seconds`, `supkv_datagrams_discarded_total{class=...}`) — planner finalizes exact names.

**Structured logging (OBS-03)**
- **D-06:** stdlib `log/slog` with the JSON handler as the default; a text-handler flag (`SUPKV_LOG_FORMAT=text|json`, default json) for local reading. Zero new dependencies.
- **D-07:** Output stays on **stderr** (current stdlib-log behavior preserved); level via `SUPKV_LOG_LEVEL` (`debug|info|warn|error`, default `info`).
- **D-08:** Every dropped-datagram class is a distinguishable structured attribute (slog attr like `class="bad_crc"|"own_broadcast"|"bad_proto_version"`), riding the existing `udprepl` Log tap and discard classifier — the classes already exist as counters and a total classification (TestDiscard_classes/totality).
- **D-09:** The pure core (`internal/app/**`) stays silent — all logging happens at adapters/main via existing seams (the `udprepl` Log callback, the OnChange-style taps), preserving the depguard I/O-free boundary. slog is stdlib and technically legal in internal/app, but the discipline is no logging in the decision core.

**Conflict journal — CANCELLED**
- **D-10:** No conflict journal in any form (no file, no per-conflict log events). Maintainer's rationale, verbatim intent: "если запись была перезаписана — она была перезаписана, это нормальная работа, а не событие, требующее разбора". A merge loser in an LWW map is semantically an ordinary overwrite. Only the merge-conflicts COUNTER (OBS-01) remains. Partition-scenario tests assert the deterministic winner and the counter, not journal entries; update TEST-04 wording in ROADMAP/REQUIREMENTS at planning time to match.

**Real-socket tests in CI (TEST-04)**
- **D-11:** Mixed strategy: the bulk of failure scenarios run in-process on real UDP sockets (the existing smoke-test pattern, `TestSmoke_twoNodesUDP`), and an exec harness covers exactly 4 two-process scenarios: (1) kill -9 one node then restart — convergence without operator action; (2) both-nodes-down restart; (3) SIGTERM graceful shutdown with final confirmed dump (D-19 contract, incl. exit codes); (4) empty-disk node start against a live peer (recovery at `sync_response_rate`). The harness builds the real binary (`go build`, TestMain or helper) and drives clients over real gRPC.
- **D-12:** On darwin the shared wildcard bind fails (EADDRINUSE, empirically resolved in D-08/PR #31) — exec and socket scenarios take the loud-skip precedent: run, print the reason, skip. No build-tag hiding of tests.

**Benchmarks (TEST-05)**
- **D-13:** Three benchmarks, run by hand via a new `make bench` target (`go test -bench ./...` scoped); CI does NOT run benchmarks and there are NO performance thresholds — shared-runner noise makes them flaky. (1) `BenchmarkDump` at 10k keys; (2) replication burst throughput (~1000 writes/s, two nodes, convergence rate); (3) bounded confirmation latency — `wait_for_replication` latency under burst, reported as a benchmark metric.
- **D-14:** No benchmark artifacts/profiles stored in CI (cpuprofile et al. remain a local `go test -bench -cpuprofile` convenience, not a pipeline).

**Folded review findings (hardening backlog)**
- **D-15:** ALL 12 open review findings from the 2026-11-14 re-reviews fold into this phase's plans (maintainer: "Влить всё"). The 4 Warnings: [P4-WR-01] drained-row `Acked` duplicate after discovery flush (fix: `slices.Compact` after sort in udprepl mirror); [P4-WR-02] `sync_response_rate` above 1e9 divides to zero and silently disables the egress limiter (fix: clamp out-of-domain rate to the default in `normalized()`); [P1-WR-01] no `timeout-minutes` on ci.yml jobs (15/20 min); [P1-IN-01→Warning-class fix] `serve()`'s `defer stop()` releases signal handling before shutdown completes (second signal during GracefulStop kills with exit 143, no final dump). The 8 Infos: [P4-IN-01] NotifyLocalWrite TOCTOU vs closed done; [P4-IN-02] Restore marks dirty unconditionally — SAME class as [P3-IN-02] ApplyRemote marks dirty on no-op merges (planner merges these two into one redundant-dump item); [P3-WR-01 is an Info-class scope decision] the provably-unreachable above-cover guard in `confirmWaiters` — resolve as: keep the documented belt AND pair it with an invariant assertion (do not silently delete); [P3-IN-01] dead Decode-error fallback in `Open`; [P2-IN-01/IN-02] OnChange doc-comment inaccuracies (tombstone wording; re-entrancy boundary); [P2-IN-03] duplicated subtest in store_test.go. Each fix updates the corresponding `0N-REVIEW-DISPOSITION.md` row to fixed.
- **D-16:** The 2026-11-14 buf zero-waiver refactor (7b55ad1) intentionally made VERIFICATION fingerprints of phases 1/4/5 stale; re-running `/gsd-execute-phase 1/4/5` to regenerate them is folded into this phase's execution flow (before milestone close), not new feature work.

### Claude's Discretion
- Exact Prometheus metric/label names and help strings (D-05 direction fixed).
- slog handler wiring details (where the handler is constructed in main, how the `udprepl` Log tap maps to slog attrs).
- Internal layout of the metrics exporter package and the exec-test harness (build helper, port allocation, process lifecycle) — follow the phase-4/5 adapter precedents.
- Which of the in-process socket scenarios beyond the IDEA list get dedicated tests vs. parameterized cases — planner decides with the researcher.
- macOS F_FULLFSYNC durability policy (open STATE blocker) — researcher investigates and proposes; if a code change is needed it lands here as a small plan item.

### Deferred Ideas (OUT OF SCOPE)
- CI job with benchmark thresholds / stored profiles — rejected for now (shared-runner noise); revisit on dedicated hardware if ever needed.
- HMAC authentication of replication datagrams — docs/IDEA.md future-work item, out of scope for v1 entirely.
- Conflict journal in ANY form — cancelled by D-10, not merely deferred; re-introduction requires a new maintainer decision.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| OBS-01 | Metrics endpoint: unacked-records count and oldest age, dump duration and errors, merge conflicts, clock-drift warnings | Seam inventory below: `Discards()`/`KnownPeers()`/`PendingCopy()` exist; `UnackedStats(now)` exists on Core but is NOT yet mirrored in the adapter; merge-conflicts and drift counters need a new core counter + mirror (ApplyRemote's returns are discarded today, repl core.go:455); dump duration/errors need a new persist engine seam (runDump has no tap). In-repo text exporter pattern verified against the format spec. |
| OBS-02 | Discarded datagrams counted by class (bad CRC / own broadcast / bad proto version) | `Adapter.Discards()` returns `repl.DiscardStats` with all 5 classes already mirrored to atomics (udprepl repl.go:343-351) — the exporter only publishes. Superset (undecodable, duplicate_node_id) recommended. |
| OBS-03 | Structured logging, every dropped-datagram class distinguishable | slog JSON handler wiring verified (signatures from pkg.go.dev); sloglint vendored-config constraints (static-msg, capitalized, context:scope) verified at .golangci.yml:357-365 — the Log tap's `fmt.Sprintf` messages must ride as attrs, not as the slog msg; discard-class logging needs a small seam decision (3 options documented). |
| TEST-04 | Every documented failure scenario an automated test demonstrating convergence within documented bounds | Existing coverage mapped (sim bounds/chaos/restart/discard tests + smoke); gap list per scenario below; exec harness ALREADY EXISTS in cmd/sup-kv/main_test.go (TestMain build, startNode/reap/signalAndWaitExit/assertExitCode, awaitServing, port pickers); empirical probe establishes the asymmetric-port topology required for two-node convergence (shared-port unicast self-delivers — cannot converge). |
| TEST-05 | Benchmarks: dump at 10k keys, replication burst throughput, bounded confirmation latency | `b.ReportMetric(n float64, unit string)` verified; three benchmark designs sketched; `make bench` target convention follows existing Makefile; wait-drain measurement can reuse the grpcapi wait seam (wait_integration_test.go precedent) or adapter PendingCopy polling. |
</phase_requirements>

## Summary

Phase 6 adds zero product capability; it makes the existing one provable and operable. Three workstreams: (1) an opt-in metrics/debug HTTP listener (one listener, optional mTLS) fed by an in-repo Prometheus text exporter — almost everything it publishes already exists as adapter atomics (`Discards()`, `KnownPeers()`, `PendingCopy()`), with three genuine seam gaps: unacked-stats mirror, merge-conflicts/drift counters (the repl core discards `ApplyRemote`'s conflict/drift returns today), and dump duration/error counters in the persist engine; (2) slog JSON logging on stderr with distinguishable discard classes, wired at main and the udprepl Log tap, constrained by the vendored sloglint config (static messages, attrs for dynamics); (3) the failure-scenario suite and benchmarks. The exec harness for D-11's four two-process scenarios already exists in `cmd/sup-kv/main_test.go` — the new scenarios extend it.

**The load-bearing empirical finding:** the shared-wildcard-port topology used by the existing in-process smoke test CANNOT support two-node convergence assertions. A probe run on this Linux machine shows unicast to the shared port is delivered to exactly one socket — the last-bound one — so node B's unicast replies to node A self-deliver to B (own-echo discard) and A never receives them. Convergence between two same-host nodes requires the **asymmetric-port pairing**: node A binds port P1 and broadcasts to `127.255.255.255:P2`, node B binds P2 and broadcasts to `127.255.255.255:P1`. Probed: each node's broadcast reaches exactly the peer, cross-unicast (including to the `0.0.0.0:P` LocalAddr handle shape the adapter actually emits) reaches exactly the peer, and no self-echo occurs. Both the in-process real-socket scenarios and the 4 exec scenarios must use this topology. This also sidesteps darwin's shared-wildcard-bind EADDRINUSE entirely (no shared bind), though loopback broadcast fan-out remains Linux-only, so the D-12 loud-skip on darwin stays correct for a different reason (lo0 carries no BROADCAST flag).

**macOS F_FULLFSYNC (open STATE blocker, resolved):** Go's `syscall.Fsync` on darwin calls plain libc `fsync()` — verified in the Go 1.27.4 toolchain source (zsyscall_darwin_*.go binds `libc_fsync`); the stdlib never issues `fcntl(F_FULLFSYNC)` (constant `F_FULLFSYNC = 0x33` is defined but unused by Fsync). Apple's fcntl(2) documents F_FULLFSYNC as "Does the same thing as fsync(2) then asks the drive to flush all buffered data to the permanent storage device". Impact on Phase 6 tests: **none** — kill -9 (process death) does not defeat kernel-cache fsync; only power loss does, and no CI test simulates power loss. The crash scenarios therefore legitimately assert process-kill durability on all platforms. Recommendation: land the small build-tagged darwin `fcntl(fd, F_FULLFSYNC, 0)` in persistfile (~15 lines, one syscall in an adapter package where syscall is depguard-legal — udprepl already imports it), keeping the durability claim platform-uniform; tests unchanged. Document-only is the acceptable fallback.

**Primary recommendation:** Plan around the seam inventory (below) — publish what exists, add the three small counter/mirror seams in the exact shape of existing patterns (core counters + adapter atomic mirrors like DiscardStats; engine counters like nothing-yet — use atomics + accessor), use the asymmetric-port topology for every two-node real-socket and exec test, extend the existing exec harness in cmd/sup-kv/main_test.go rather than building a new one, and fold the 12 review findings as one early "hardening backlog" plan before the feature plans compound on top of them.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Metrics HTTP listener + mTLS | cmd/sup-kv (main wiring) | new internal adapter package (exporter + mux) | Listener lifecycle joins main's D-19 shutdown ordering; net/http/crypto/tls are I/O — depguard-forbidden in internal/app, so the exporter lives in a new internal/ adapter |
| Prometheus text rendering | new internal adapter pkg (~100 lines) | — | Pure formatting of supplied gauges/counters; sorted families, verified escaping rules |
| go_*/process_* families | new internal adapter pkg | — | runtime.ReadMemStats / runtime.NumGoroutine / getrusage — I/O-free reads but belong with the exporter, not the decision core |
| slog handler construction | cmd/sup-kv | — | Flag/env resolution + handler selection is main's existing job (D-16/D-17 pattern) |
| Discard-class slog events | internal/udprepl adapter (tap mapping) | internal/app/repl core (classification) | D-09: core stays silent; the class decision lives in core (HandleDatagram), the logging at the adapter seam |
| Unacked/merge/drift/sent counters | internal/app/repl core (counters) | internal/udprepl adapter (atomic mirrors) | The DiscardStats precedent: counters live in core, mirrored post-event, read cross-goroutine |
| Dump duration/error counters | internal/app/persist engine (atomics + accessor) | metrics adapter (reads accessor) | runDump is the single serialized dump site; atomics keep it lock-free for scrapers |
| In-process real-socket scenarios | internal/udprepl tests | internal/app/repl sim tests | Real sockets express discovery/echo/gRPC; partitions/wall-clock/one-way need the simulator |
| Exec two-process scenarios | cmd/sup-kv main_test.go | — | Harness exists (TestMain/startNode/reap); extend |
| Benchmarks | internal/app/persist, internal/udprepl, internal/grpcapi test files | Makefile `bench` target | Each benchmark lives with the seam it measures; make target is the D-13 door |
| CI timeout-minutes | .github/workflows/ci.yml | — | P1-WR-01; must not rename check contexts (ruleset contract) |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` | go 1.27.1 pin (toolchain 1.27.4) | Metrics/debug listener, ServeMux | Already the API-layer transport family; zero new deps [VERIFIED: go.mod — 2 direct requires only] |
| Go stdlib `crypto/tls` + `crypto/x509` | stdlib | Optional mTLS on metrics listener | `tls.Config{Certificates, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs}` — verified signatures [CITED: pkg.go.dev/crypto/tls] |
| Go stdlib `log/slog` | stdlib | Structured JSON logging (OBS-03) | D-06 locked; `slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: ...}))` [CITED: pkg.go.dev/log/slog] |
| Go stdlib `expvar` | stdlib | `/debug/vars` | `expvar.Handler()` mounts on a custom mux [CITED: pkg.go.dev/expvar] |
| Go stdlib `net/http/pprof` | stdlib | `/debug/pprof/*` | Manual mounting on custom mux verified: `pprof.Handler(name)`, `pprof.Index` at prefix [CITED: pkg.go.dev/net/http/pprof] |
| In-repo Prometheus text exporter | ~100 new lines | `/metrics` | D-04 locked (client_golang rejected); format rules verified [CITED: prometheus.io/docs/instrumenting/exposition_formats/] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `runtime` (+`runtime.ReadMemStats`, `runtime.NumGoroutine`) | stdlib | go_* families | Scrape-time reads in the exporter |
| `syscall.Getrusage` / `/proc/self` reads | stdlib | process_* families | Linux+darwin both support getrusage; keep the family set minimal (planner discretion under D-04) |
| `syscall.Syscall(SYS_FCNTL, fd, F_FULLFSYNC, 0)` | stdlib (darwin-only file) | True disk flush on macOS | Recommended small fix; build-tagged `fsync_darwin.go` in persistfile |
| `testing` + `b.ReportMetric` | stdlib | Benchmarks | `func (b *B) ReportMetric(n float64, unit string)` — per-op metrics divide by b.N with "/op" unit [CITED: pkg.go.dev/testing] |
| `pgregory.net/rapid` | v1.3.0 (existing test dep) | Any new property-style assertions | Only if a scenario wants randomized input; scenarios are mostly deterministic |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-repo text exporter | prometheus/client_golang v1.25.0 | REJECTED by D-04/.claude/CLAUDE.md — drags prometheus/common, procfs, client_model into go.sum against the gRPC+protobuf-only bar |
| slog | zap/zerolog | REJECTED — non-stdlib, no advantage at this scale (.claude/CLAUDE.md stack table) |
| Exec harness from scratch | reuse cmd/sup-kv/main_test.go harness | Reuse — TestMain build + startNode/reap/signalAndWaitExit/assertExitCode/awaitServing already exist [VERIFIED: cmd/sup-kv/main_test.go:63-290] |

**Installation:** none — `go.mod` stays at exactly 2 direct requires (`google.golang.org/grpc v1.85.0`, `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e`) [VERIFIED: go.mod require blocks]. Everything Phase 6 touches is stdlib. This is worth stating in every plan: **zero new dependencies, direct or test**.

## Package Legitimacy Audit

> No external packages are installed in this phase. All code is Go stdlib + in-repo. (`prometheus/client_golang` explicitly rejected per D-04; `pgregory.net/rapid` v1.3.0 already a test dependency from Phase 2 — unchanged.)

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| (none new) | — | — | — | — | — | N/A |

## Architecture Patterns

### System Architecture Diagram

```
                      ┌────────────────────────── cmd/sup-kv main ──────────────────────────┐
                      │  registerConfig: +--metrics-addr/SUPKV_METRICS_ADDR, SUPKV_METRICS_  │
                      │  CERT/KEY/CLIENT_CA, SUPKV_LOG_LEVEL, SUPKV_LOG_FORMAT               │
                      │  slog.New(NewJSONHandler|NewTextHandler(os.Stderr, level)).SetDefault│
                      └───────┬──────────────────────┬───────────────────────┬──────────────┘
                              │                      │                       │
              D-19 shutdown   │                      │                       │
              ordering gains  ▼                      ▼                       ▼
   ┌──────── metrics/debug listener (opt-in) ─┐   gRPC listener          udprepl Adapter
   │  mux: /metrics      → in-repo exporter   │   (existing)            (existing event loop)
   │        /debug/vars   → expvar.Handler()  │                            │ mirrors per event:
   │        /debug/pprof/ → pprof.Index +     │                            │  Discards(), peerCount,
   │          pprof.Handler(name)...           │                            │  syncArrived, pendingCopy
   │  tls.NewListener(inner, cfg) when        │                            │  + NEW: UnackedStats mirror,
   │  CERT/KEY/CLIENT_CA all set              │                            │    sent-by-type, merge
   └──────────────┬───────────────────────────┘                            │    conflicts, drift
                  │ scrape (pull)                                          │
   ┌──────────────▼──────────────────────────────────────────────────────┐ │
   │ exporter reads (all pull-safe, atomic or copy-on-write):            │ │
   │  adapter.Discards()       → supkv_datagrams_discarded_total{class} ◄─┘
   │  adapter.KnownPeers()     → supkv_replication_peers
   │  adapter.UnackedStats()   → supkv_replication_unacked_records{,_oldest_age_seconds}  [NEW mirror]
   │  adapter.SentByType()     → supkv_replication_datagrams_{sent,retransmitted}_total{type}  [NEW]
   │  adapter.MergeStats()     → supkv_merges_conflicts_total, supkv_clock_drift_warnings_total  [NEW]
   │  engine.DumpStats()       → supkv_dumps_duration_seconds, supkv_dumps_{total,errors_total}  [NEW]
   │  runtime/expvar           → go_*, process_* families
   └────────────────────────────────────────────────────────────────────┘

   Failure-scenario tests (TEST-04):
   sim (internal/app/repl sim tests, virtual time)     real-socket (internal/udprepl, in-process
   ├─ partition seconds/days, deterministic winner     ├─ two adapters, ASYMMETRIC ports:
   ├─ one-way connectivity (Policy.blocked)            │   A: bind P1, bcast→P2; B: bind P2, bcast→P1
   ├─ wall-clock jumps/far-ahead (clock-offset seam)   ├─ duplicate node_id, proto mismatch,
   ├─ ack loss, 1000 writes/s bursts                     hand-crafted bad CRC datagrams
   └─ empty-disk recovery rate-bounded                  └─ (existing shared-port smoke stays for OwnEcho)
   exec (cmd/sup-kv main_test.go — 4 scenarios, real binary, real gRPC clients)
   ├─ kill -9 → restart convergence      ├─ both-nodes-down restart
   ├─ SIGTERM → exit 0 + final dump      └─ empty-disk node vs live peer
```

### Recommended Project Structure
```
internal/
├── metrics/            # NEW adapter: in-repo Prometheus text exporter + registry
│   ├── metrics.go      #   family model, sorted rendering, escaping
│   └── metrics_test.go
├── obshttp/ (name: planner)  # NEW or fold into metrics/: mux assembly (/metrics,
│                             #   /debug/vars, /debug/pprof/*), TLS config builder
├── udprepl/
│   ├── repl.go         # + UnackedStats mirror, sent-by-type counters, discard-class logging
│   ├── scenario_test.go # NEW: asymmetric-topology real-socket failure scenarios
│   └── smoke_test.go   # unchanged (OwnEcho assertion stays here)
├── app/repl/
│   ├── core.go         # + merge-conflicts/drift counters, rate clamp (P4-WR-02)
│   └── sim_scenario_test.go # NEW: partition-days, one-way, clock-jump scenarios
├── app/persist/
│   ├── engine.go       # + dump stats atomics (duration/errors), redundant-dirty fixes
│   └── bench_test.go   # NEW: BenchmarkDump10kKeys
├── persistfile/
│   ├── dumpfile.go     # (unchanged unless F_FULLFSYNC adopted)
│   └── fsync_darwin.go # NEW (recommended): F_FULLFSYNC fcntl
├── grpcapi/
│   └── bench_test.go   # NEW: confirmation-latency benchmark (wait seam)
└── storage/store.go   # OnChange doc fixes (P2-IN-01/02)
cmd/sup-kv/
├── main.go             # + metrics listener wiring, slog handler, serve() signal fix (P1-IN-01)
└── scenario_test.go    # NEW: the 4 exec scenarios (extends existing harness)
Makefile                # + bench target
.github/workflows/ci.yml # + timeout-minutes (P1-WR-01)
```

### Pattern 1: Adapter atomic mirrors (extend, don't invent)
**What:** Cross-goroutine reads of core state go through atomics the event loop refreshes after every iteration. `mirrorStats`/`mirrorState` in udprepl repl.go:449-488 are the precedent; `Discards()` (repl.go:343-351) is the read shape.
**When to use:** Every new OBS metric touching core state: UnackedStats (call `core.UnackedStats(time.Now())` at the mirror site), sent-by-type (count at `flush` where the adapter sees every `port.Message`), retransmitted (core-side counter — only `scanRetransmit` knows — mirrored like DiscardStats), merge conflicts + drift warnings (core counters incremented at the `HandleDatagram` ApplyRemote site where the returns are discarded today: `_, _, err := c.store.ApplyRemote(msg.Key, msg.Record)` [VERIFIED: internal/app/repl/core.go:455]).
**Example:**
```go
// Source: internal/udprepl/repl.go:449-457 (existing precedent to extend)
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

### Pattern 2: In-repo Prometheus text exporter
**What:** ~100-line formatter: families sorted by name, labels sorted by label name, HELP/TYPE emitted once before a family's samples, `# TYPE name counter|gauge` must precede the first sample of that name.
**Verified format rules** [CITED: prometheus.io/docs/instrumenting/exposition_formats/]:
- Content-Type: `text/plain` with parameter `version=0.0.4` (charset=utf-8 conventional).
- HELP docstring: escape backslash as `\\` and line feed as `\n` only.
- Label values: escape backslash `\\`, double-quote `\"`, line feed `\n`.
- All lines of one metric form one single group, HELP/TYPE first; each line a unique name+labels combination; last line ends `\n`; values are Go `ParseFloat` floats (`NaN`, `+Inf`, `-Inf` valid).
- Reproducible sorting across expositions is "preferred but not required" — for sup-kv, sorting IS required by D-04 (sorted families/labels) and is what makes scrape diffs stable.
**Naming** [CITED: prometheus.io/docs/practices/naming/]: base units (seconds, bytes), unit suffix plural (`_seconds`), counters carry `_total`, single-word app prefix (`supkv_`), labels for class/type not embedded in names, bounded label cardinality (all label values here are small fixed enums).
**Example skeleton (D-05 direction, planner finalizes names):**
```go
// supkv_* families, per D-04/D-05 (names finalized by planner):
//   supkv_replication_unacked_records (gauge)
//   supkv_replication_unacked_oldest_age_seconds (gauge)
//   supkv_dumps_duration_seconds (gauge, last dump)
//   supkv_dumps_total / supkv_dumps_errors_total (counters)  — or {result} label
//   supkv_merges_conflicts_total (counter)
//   supkv_clock_drift_warnings_total (counter)
//   supkv_datagrams_discarded_total{class="bad_crc"|"own_broadcast"|"bad_proto_version"|"undecodable"|"duplicate_node_id"}
//   supkv_replication_peers (gauge)
//   supkv_replication_datagrams_sent_total{type="hello"|"data"|"ack"|"sync"|"data_request"}
//   supkv_replication_datagrams_retransmitted_total
//   go_* / process_* minimal families (planner picks set under D-04)
```

### Pattern 3: One listener, optional mTLS, custom mux
**What:** D-03's debug surface on one opt-in listener. Verified building blocks:
```go
// Source: pkg.go.dev/crypto/tls (verified semantics)
cert, err := tls.LoadX509KeyPair(certFile, keyFile)
clientCAs := x509.NewCertPool()
clientCAs.AppendCertsFromPEM(caPEM)
cfg := &tls.Config{
	Certificates: []tls.Certificate{cert},
	ClientAuth:   tls.RequireAndVerifyClientCert,
	ClientCAs:    clientCAs,
	MinVersion:   tls.VersionTLS12, // explicit floor — security-section recommendation
}
inner, err := net.Listen("tcp", metricsAddr)
listener := tls.NewListener(inner, cfg)

// Source: pkg.go.dev/net/http/pprof + pkg.go.dev/expvar (custom-mux mounting)
mux := http.NewServeMux()
mux.Handle("/metrics", exporter)
mux.Handle("/debug/vars", expvar.Handler())
mux.HandleFunc("/debug/pprof/", pprof.Index)
mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
mux.Handle("/debug/pprof/heap", pprof.Handler("heap")) // named profiles via Handler(name)
srv := &http.Server{Handler: mux, ReadHeaderTimeout: ...} // explicit Server: see Pitfall 5
```
**Notes:** blank-importing `net/http/pprof` or `expvar` registers on `http.DefaultServeMux` — do NOT blank-import; reference the symbols explicitly (which is required anyway for custom mounting). Do not register `pprof.Handler(name)` per named profile AND rely on Index — the `pprof.Index` prefix route covers named profiles generically [CITED: pkg.go.dev/net/http/pprof]. mTLS test certificates: generate in-test with stdlib `crypto/x509.CreateCertificate` + `crypto/ecdsa` (~60-line helper, no openssl dependency, no committed fixtures).

### Pattern 4: slog wiring at main + the Log tap
**What:** main builds the handler once from env (`SUPKV_LOG_LEVEL` via `slog.Level.UnmarshalText` — accepts "debug"/"info"/... case-insensitively; `SUPKV_LOG_FORMAT` picks NewJSONHandler default vs NewTextHandler), `slog.SetDefault`, and the udprepl `Log: func(msg string)` callback (main.go:310 today: `Log: func(msg string) { log.Printf("%s", msg) }`) becomes the slog bridge.
**Constraints from the vendored strict config** [VERIFIED: .golangci.yml:357-365 — sloglint `context: scope`, `static-msg: true`, `msg-style: capitalized`, `forbidden-keys: time/level/msg/source`]:
- The slog message must be a **static string** — the tap's `fmt.Sprintf`-built messages (e.g. DuplicateNodeID config errors at core.go:334-344) must ride as an **attr** (`slogger.Warn("Replication adapter", "detail", msg)` shape), never as the msg.
- Capitalized msg style; attrs must not use reserved keys.
- `context: scope`: when a `ctx` is in lexical scope, log via the context-aware call shape — the adapter tap closures don't hold ctx, which keeps them clean; prefer that.
**Discard-class distinguishability (D-08) — three shapes, planner picks one:**
1. Extend `repl.Config.Log` semantics: core's discard branches (HandleDatagram, core.go:315-339 — BadCRC/Undecodable/BadProtoVersion/OwnEcho return silently today) log class-prefixed messages; the adapter tap maps to `slog` with a fixed msg + `class` attr parsed from the prefix. Downside: stringly contract.
2. New `repl.Config` seam alongside Log, e.g. `OnDiscard func(class, detail string)` — core calls it at each discard branch with the class as a first-class value (the DiscardStats precedent is "counters, not callbacks", but `Log func(msg string)` in Config is the existing callback precedent [VERIFIED: core.go:95 — `Log func(msg string)`]). Recommended: cleanest attr mapping, no parsing.
3. Adapter-side classification only — possible for bad_crc/undecodable/bad_proto_version (the adapter's `decode()` already computes them before the core sees the datagram) but NOT for own_broadcast/duplicate_node_id (those need core state). Incomplete — reject.
Rate-limiting consideration: discard classes can be hot (bad-CRC floods); the slog Warn level + a simple per-class suppression window (or logging at Debug with the counter carrying the signal at Warn) is planner discretion — the COUNTER is the reliable signal, the log line is for humans.

### Pattern 5: Asymmetric-port two-node topology (the convergence-test topology)
**What:** For any same-host test needing bidirectional delivery between two nodes (in-process scenarios AND the 4 exec scenarios): node A binds `0.0.0.0:P1`, broadcast-addr `127.255.255.255:P2`; node B binds `0.0.0.0:P2`, broadcast-addr `127.255.255.255:P1`.
**Empirically probed on this Linux machine (2026-11-15):**
```
--- probe1: shared port, A bound first, B second (the smoke topology) ---
B got "thirdparty-unicast" from 127.0.0.1:35180      # unicast -> last-bound only
B got "b-unicast-to-shared" from 127.0.0.1:60125     # B's unicast to A SELF-DELIVERS to B; A gets nothing
A got "broadcast" ...  B got "broadcast" ...          # broadcast fan-out reaches both
--- probe2: asymmetric ports, cross-broadcast ---
B got "a-bcast-to-p2" ...  B got "a-unicast-to-p2" ...  # A's broadcast AND unicast reach only B
A got "b-bcast-to-p1" ...  A got "b-unicast-to-p1" ...  # B's broadcast AND unicast reach only A
```
Follow-up probe: unicast to the `0.0.0.0:P1` destination (the exact `conn.LocalAddr().String()` handle shape the adapter resolves in `flush`, repl.go:429) delivers to the sole socket on P1 — `write to 0.0.0.0:48977: n=12 err=<nil>` / `A read: n=12 src=127.0.0.1:36018 payload="to-zero-addr"`.
**Why it matters:** convergence after divergence depends on unicast replies (DataRequest answers, DataSync-newer DataReplication, Acks). In the shared-port topology those self-deliver to the sender (own-echo discard) on Linux — the kernel's choice was consistently the last-bound socket — so a kill -9/empty-disk convergence assertion would hang ~always for the last-bound node. The adapter package doc already records the shared-port unicast limitation [VERIFIED: internal/udprepl/repl.go:5-10 "unicast to the shared port is delivered to exactly one socket, the kernel's choice"].
**Trade-off:** no self-echo in the asymmetric topology (a node never receives its own broadcast) — the OwnEcho discard assertion stays in the existing shared-port `TestSmoke_twoNodesUDP` (Linux-only), unchanged. Darwin: no shared bind is needed at all (EADDRINUSE vanishes) but loopback broadcast fan-out remains Linux-only (macOS lo0 has no BROADCAST flag) — the loud-skip stays, for the broadcast-unavailable reason; reuse the `probeBroadcastAddr` capability-probe shape with a cross-delivery check.
**When to use:** every new two-node real-socket scenario; the 4 exec scenarios (pass `--bind-addr=0.0.0.0:P1 --broadcast-addr=127.255.255.255:P2` to node A and the mirror to node B).

### Pattern 6: Exec scenarios on the existing harness
**What:** cmd/sup-kv/main_test.go already contains the whole D-11 harness: `TestMain` builds the binary once into a temp dir (main_test.go:63-95), `smokeBinary(t)` loud-skips on toolchain probe failure (101-111), `freeTCPPort`/`freeUDPPort` (115-136), `waitFor` (141-156), `startNode`/`reap`/`signalAndWaitExit`/`assertExitCode` (171-251), `awaitServing` + `supKVConn` over real gRPC (256-280), `assertNoPanic` (284-290). Existing tests already drive `--version`, flag>env precedence, and a full lifecycle with SIGTERM + exit 0 + dump grep (296-420).
**The 4 scenarios are extensions:** (1) kill -9 = `node.cmd.Process.Kill()` (not SIGTERM), restart with the same `--dump-path`, assert convergence via Get on both gRPC endpoints within REPL-10-ish bounds; (2) both-nodes-down = SIGTERM both, await exit 0 both, restart both, assert convergence + dump contents survived; (3) SIGTERM contract = largely TestMain_lifecycle already — extend to assert the D-19 final-dump semantics against a second live peer (replicated writes in the final dump) and the exit codes (`exitDumpFailed = 1`, `exitUsage = 2` [VERIFIED: main.go:63-66 — `exitDumpFailed = 1`, `exitUsage = 2`]); (4) empty-disk = start node B with a fresh `--dump-path` against a live seeded node A, assert recovery bounded by the `sync_response_rate` drain (measure wall time, assert `< keys/rate + margin`).
**Timers:** pass explicit `--hello-interval=100ms --sync-interval=200ms --startup-gate=1s` style flags to keep scenarios inside CI deadlines (the smoke tests already shrink `--startup-gate`).

### Pattern 7: Simulator scenarios (partition-days, one-way, clock jumps)
**What:** simnet already provides `Policy.blocked(from, to string, now time.Time) bool` (a test-supplied closure — timed and one-directional partitions are directly expressible) [VERIFIED: internal/app/internal/simnet/simnet.go:146], `Scheduler.RunUntil` virtual time (days of partition cost microseconds), `Cluster.Write/Store/Core/Network/FaultStats`.
**Gap:** each node's clock is hardwired to `sched.Now` with a fixed drift gate at construction — `clock.NewClock(sched.Now, 0, 0, clusterDriftWarn)` [VERIFIED: simnet.go:374, `clusterDriftWarn = time.Minute` at simnet.go:35]. Wall-clock-jump/far-ahead scenarios need a per-node clock wrapper with a mutable offset (a `func() time.Time` closure over a settable skew). Small harness extension: either a NewCluster variant taking per-node now funcs, or an exported per-node `Skew(d time.Duration)` helper on Cluster. Deterministic (offsets change only from test code).
**Restart scenario in sim:** runIDs are construction-fixed (`simRunID(nodeID)`, simnet.go:329-333); a restart scenario builds a replacement node (new core, new runID, empty store) registered at the same handle — the TestRestart_* core tests (restart_test.go:97-190) already pin the protocol reaction; the sim-cluster version proves the whole-cluster convergence shape.

### Pattern 8: Benchmarks (TEST-05 / D-13)
**What:** three benchmarks, stable names, `b.ReportMetric` for derived quantities. Verified API: `func (b *B) ReportMetric(n float64, unit string)` — per-iteration metrics divide by `b.N` with a `/op` unit suffix; setting ns/op to 0 suppresses the built-in; report once after any parallel section [CITED: pkg.go.dev/testing].
1. `BenchmarkDump10kKeys` (internal/app/persist): store seeded with 10k keys (varied value sizes), dump through the real path (Encode + persistfile Save into `b.TempDir()`); report `bytes/op` and dump size.
2. `BenchmarkReplicationBurst` (internal/udprepl): two adapters on the asymmetric topology, shrunken timers; write at ~1ms cadence for b.N writes, then wait for convergence (KnownPeers/UnackedStats==0 poll); report converged writes/s (`b.ReportMetric(writesPerSec, "writes/s")` — note unit strings must not contain whitespace).
3. `BenchmarkConfirmationLatency` (internal/grpcapi or udprepl): per-write drain latency — the wait_for_replication shape (waitForReplication reads only the injected seams; grpcapi has integration-test precedent in wait_integration_test.go). Under a background burst writer, measure per-write appearance→drain latency; report `latency-ns/op` and max observed.
**make bench:** follow the Makefile convention; scope to the three packages with `-run='^$' -bench=.` so unit tests don't run; NOT in CI (D-13/D-14).

### Anti-Patterns to Avoid
- **Blank-importing `net/http/pprof` or `expvar`** — registers on `http.DefaultServeMux`, which nothing serves (and would silently couple to future DefaultServeMux users). Mount explicitly on the metrics mux.
- **Two-node convergence tests on the shared-port topology** — unicast self-delivers; the test hangs or flakes (see Pattern 5 probe).
- **Dynamic slog messages** — sloglint `static-msg: true` fails the gate; put dynamics in attrs.
- **New callbacks into internal/app for metrics** — where a counter suffices, mirror the DiscardStats pattern; the core stays results-only except the existing Log-style Config seams.
- **CI benchmark gates** — rejected by D-13; `make bench` is a manual target only.
- **Renaming CI jobs while adding timeout-minutes** — check-run names are a ruleset contract [VERIFIED: STATE.md decision "CI check-run contexts fixed verbatim"].

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Metrics exposition | A "registry" framework with registration lifecycles | A ~100-line pure formatter over a fixed family snapshot function | The family set is closed and small; D-04 explicitly scopes the exporter to ~100 lines |
| TLS server/client plumbing | Custom handshake code | `tls.NewListener` + `tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs}` | stdlib, verified semantics |
| pprof/expvar handlers | Reimplementing profile endpoints | `pprof.Index/Handler(name)/Cmdline/Profile/Symbol/Trace`, `expvar.Handler()` | stdlib, verified custom-mux mounting |
| Structured logging | Custom JSON log formatter | `slog.NewJSONHandler` / `NewTextHandler` | stdlib, D-06 locked |
| Test certificates | Committed PEM fixtures (rot/audit burden) | In-test `crypto/x509.CreateCertificate` self-signed CA + leaf pair | Deterministic, no fixtures, stdlib only |
| Exec process management | New harness | Extend `cmd/sup-kv/main_test.go` helpers | Everything exists: build-once, start/reap/signal/exit-code asserts |
| Two-node UDP test topology | New socket tricks (SO_REUSEPORT etc.) | Asymmetric port pairing (Pattern 5) | Probed reliable; zero kernel-behavior gambling |

**Key insight:** every observable quantity this phase publishes already exists or is one small counter away from existing — the phase's risk is concentrated in test topology and wiring, not in new mechanisms.

## Common Pitfalls

### Pitfall 1: Shared-port unicast self-delivery breaks convergence tests
**What goes wrong:** A two-node test on one shared UDP port asserts convergence; the last-bound node's unicast replies (Ack, DataRequest answers, sync-newer DataReplication) return to itself (own-echo discard); convergence never completes or times out.
**Why it happens:** Linux delivers unicast to a port shared by SO_REUSEADDR sockets to exactly one socket — empirically the last bound (probe, Pattern 5). Recorded in the adapter doc but easy to forget when copying the smoke-test shape.
**How to avoid:** Asymmetric port pairing for every two-node scenario (Pattern 5).
**Warning signs:** scenario hangs waiting for visibility; OwnEcho counter climbing on the node that should be receiving.

### Pitfall 2: sloglint static-msg gate failure
**What goes wrong:** Wiring the existing `fmt.Sprintf`-style Log tap messages straight into `slog.Warn(msg)` fails `make lint` (`static-msg`).
**Why:** vendored strict config enables sloglint with `static-msg: true`, `msg-style: capitalized`, `context: scope`, forbidden keys time/level/msg/source [VERIFIED: .golangci.yml:357-365].
**How to avoid:** fixed capitalized msg + `detail`/`class` attrs (Pattern 4).
**Warning signs:** lint failures on first slog code; attr key collisions with reserved keys.

### Pitfall 3: Prometheus exposition ordering/escaping bugs
**What goes wrong:** A scrape parses but pollutes: duplicate HELP lines, TYPE after samples, unescaped `\` or `"` in label values (help strings contain `"`), unsorted families making diffs noisy.
**Why:** The format requires TYPE before the first sample of a name, exactly one HELP/TYPE per name, single-group families; label values escape `\\`, `\"`, `\n`; HELP escapes `\\`, `\n` [CITED: exposition formats doc]. sup-kv label values are fixed enums (safe), but HELP text and any future key-carrying labels are not.
**How to avoid:** exporter unit test pinning the exact rendering of a family table with adversarial help text; sort families and labels in the writer, not at call sites.
**Warning signs:** promtool-free CI won't catch this — only the unit test will.

### Pitfall 4: DefaultServeMux leakage
**What goes wrong:** `import _ "net/http/pprof"` or `_ "expvar"` in main silently registers handlers on the DefaultServeMux that nothing serves — or worse, some future DefaultServeMux listener exposes them unauthenticated.
**Why:** Both packages register on import [CITED: pkg.go.dev docs].
**How to avoid:** Named imports, explicit mounting (Pattern 3).
**Warning signs:** any blank import of pprof/expvar in diff review.

### Pitfall 5: gosec on the HTTP server
**What goes wrong:** `http.ListenAndServe`-style serving without timeouts draws a gosec finding under the vendored config (gosec enabled, `audit: true`, only G104 excluded [VERIFIED: .golangci.yml:162-167]).
**Why:** unbounded-timeout servers are a DoS surface; the strict config doesn't suppress it.
**How to avoid:** construct `http.Server{ReadHeaderTimeout, ...}` explicitly and Serve on the listener — the code-direction fix, never a nolint. (Exact rule ID for this shape not re-verified this session [ASSUMED] — the practice is right regardless.)
**Warning signs:** first lint run on the listener code.

### Pitfall 6: Exec-scenario flakiness in CI
**What goes wrong:** Two-process scenarios time out on loaded shared runners; port pickers race; darwin cells fail on broadcast.
**Why:** Real-time waits, real processes, real sockets; darwin has no loopback broadcast fan-out.
**How to avoid:** generous `waitFor` deadlines with poll cadence (existing constants: servingDeadline 10s, exitDeadline 15s); asymmetric ports (no shared bind); capability probe + loud darwin skip (D-12 locked); `timeout-minutes` (P1-WR-01) bounds the whole job; never assert wall-clock precision — assert bounds (e.g. empty-disk recovery `< keys/rate + slack`).
**Warning signs:** repeated `timed out after ... waiting for` failures on one matrix cell.

### Pitfall 7: Metrics listener lifecycle vs D-19 ordering
**What goes wrong:** The metrics listener is forgotten in the graceful-shutdown sequence, or shutting it down early removes observability mid-shutdown.
**Why:** main's shutdown order is the contract (D-19: health NOT_SERVING → apiCancel → GracefulStop bounded → replCancel → Flush → engineCancel).
**How to avoid:** explicit position in the sequence — recommend `srv.Shutdown(ctx)` for the metrics server right after health flips NOT_SERVING (scrapers see the close) or concurrently with GracefulStop; document the choice in the shutdown doc comment. Also: `--metrics-addr` unset ⇒ NO listener at all (D-01) — no goroutine, no default port.
**Warning signs:** a metrics scrape succeeding after exit began; missing-listener flag handling creating a listener anyway.

### Pitfall 8: Incomplete cert-set validation order
**What goes wrong:** One of the three mTLS envs unset — listener starts plaintext with certs half-loaded, or exits with a confusing bind error.
**Why:** D-02: incomplete sets are a startup error (exit 2, the existing `exitUsage` path [VERIFIED: main.go:63-66]).
**How to avoid:** validate the cert triple in `finalizeConfig`-adjacent code BEFORE binding anything; error message names all three envs.
**Warning signs:** any code path that can reach tls.LoadX509KeyPair with an empty string.

### Pitfall 9: Benchmark noise and unit-string mistakes
**What goes wrong:** `make bench` numbers jump run-to-run; `b.ReportMetric` panics.
**Why:** shared-runner noise (D-13's rationale for no CI gates); ReportMetric panics on empty or whitespace-containing unit strings [CITED: pkg.go.dev/testing].
**How to avoid:** stable names, no thresholds, `-benchtime` left default, unit strings like `writes/s`/`latency-ns/op` (no spaces); run-to-run comparison is the human's job.
**Warning signs:** any `if nsPerOp > X { b.Fatal }` creeping in.

### Pitfall 10: The 12 folded fixes regressing pinned behavior
**What goes wrong:** P3-WR-01's guard gets "simplified" away; the redundant-dirty fix changes PERS-04 confirmation semantics; the rate clamp changes the pinned spacing arithmetic (200/400/800ms ladder tests).
**Why:** Each finding touches behavior other tests pin (e.g. `floor(1s/rate)` spacing documented at core.go:83-85; the ladder doc comment is "the single documented home").
**How to avoid:** D-15's exact fix shapes (slices.Compact after sort; clamp in normalized(); keep-guard-plus-assertion); every fix lands with a regression test; update the corresponding 0N-REVIEW-DISPOSITION.md row in the same PR.
**Warning signs:** a fix PR whose diff deletes a guard comment without adding an assertion.

## Code Examples

### Exporter core (illustrative shape; planner finalizes)
```go
// Source: format rules verified from prometheus.io exposition docs (classic text format 0.0.4)
// Content-Type: text/plain; version=0.0.4; charset=utf-8
// TYPE must precede the family's first sample; one HELP/TYPE per name;
// label values escape \\ \" \n; HELP escapes \\ \n; families sorted by name.
w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
// render families in sorted order:
//   # HELP supkv_datagrams_discarded_total Inbound datagrams discarded by class.
//   # TYPE supkv_datagrams_discarded_total counter
//   supkv_datagrams_discarded_total{class="bad_crc"} 3
//   ...
```

### slog handler construction (verified signatures)
```go
// Source: pkg.go.dev/log/slog
var lvl slog.Level // "debug|info|warn|error", default info; malformed = startup error (env pattern)
_ = lvl.UnmarshalText([]byte(raw))
h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
if format == "text" {
	h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
}
slog.SetDefault(slog.New(h))
```

### Confirmation-latency benchmark reporting (verified API)
```go
// Source: pkg.go.dev/testing — ReportMetric(n float64, unit string)
b.ReportMetric(float64(totalDrainNanos)/float64(b.N), "latency-ns/op")
b.ReportMetric(maxDrainNanos, "max-latency-ns") // no whitespace in unit strings
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| stdlib `log` everywhere | `log/slog` structured JSON (Go 1.21+) | stdlib since 1.21; project adopts in Phase 6 (D-06) | log field names become the parsing contract for shippers (D-06 one-way) |
| client_golang as default metrics lib | Zero-dep text exporter viable for counter/gauge sets | project decision 2026-11 (D-04, .claude/CLAUDE.md §rejected) | ~100 lines vs 3+ modules; histograms would be the revisit trigger |
| `b.N`-only benchmarks | `b.ReportMetric` custom metrics (Go 1.13+), `b.Loop` (recent) | — | D-13's latency metric uses ReportMetric |

**Deprecated/outdated:** none project-relevant beyond the above.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | gosec flags timeout-less HTTP serving under the vendored config; explicit `http.Server` + `ReadHeaderTimeout` is the fix shape (exact rule ID unverified) | Pitfall 5 | Low — practice correct regardless; if the rule doesn't fire, nothing is lost |
| A2 | The minimal `go_*`/`process_*` family set (goroutines, memstats subset, CPU/RSS via getrusage) satisfies D-04's "runtime families" | Standard Stack / Pattern 2 | Low — D-04 fixes the direction, planner finalizes enumeration; getrusage is linux+darwin (matrix has no windows) |
| A3 | `syscall.Syscall(syscall.SYS_FCNTL, fd, F_FULLFSYNC, 0)` works on both darwin/arm64 and amd64 for file+dir fds | F_FULLFSYNC recommendation | Low — if the small fix is dropped, the document-only fallback stands; tests are unaffected either way |
| A4 | Cert envs stay env-only (no flag twins), matching D-02's wording | Pattern 3 / Open Questions | Low — planner may mirror the flag>env pattern instead; operational contract nuance |
| A5 | Exec empty-disk bound "keys/rate + slack" is assertable with shrunken `--sync-interval`/rate flags without flaking | Pattern 6 | Medium — if CI timing proves too noisy, fall back to asserting convergence-within-deadline only (still satisfies the scenario) |
| A6 | darwin CI cells loud-skip the new socket/exec scenarios via the broadcast-unavailable probe (asymmetric topology removes the EADDRINUSE reason but not the lo0-no-BROADCAST reason) | Pattern 5 / D-12 | Low — if macos-15 subnet broadcast actually works, the probe passes and the scenario runs; D-12's skip is capability-gated either way |

## Open Questions

1. **Discard-class logging seam shape** (D-08): extend `repl.Config.Log` conventions vs a new `OnDiscard func(class, detail string)` Config field (recommended, Pattern 4). — What we know: sloglint forces static msgs; core classification sites are the HandleDatagram branches. Unclear: maintainer's preference for seam surface growth in repl.Config. Recommendation: OnDiscard-style field — class is a first-class value, no parsing, mirrors the Log precedent.
2. **Dump-duration metric shape**: last-duration gauge vs count+errors counters vs both (recommended: `supkv_dumps_duration_seconds` gauge + `supkv_dumps_total`/`supkv_dumps_errors_total` counters). Prometheus purists would use a histogram — rejected as exporter overkill for D-04's ~100-line scope.
3. **simnet clock-offset seam**: exact shape (NewCluster variant vs Cluster method) for wall-clock-jump/far-ahead scenarios. Recommendation: small `Cluster` method or per-node now-func injection; keep determinism (offsets mutate only from test code).
4. **Second-signal semantics for P1-IN-01**: after fixing signal release (stop() moves past shutdown), a second SIGTERM during GracefulStop could (a) be ignored until shutdown completes, or (b) trigger the force-stop path (`gs.Stop()` — the existing 10s grace already bounds this). Recommendation: (b) — an operator's second signal means "hurry"; the final Flush still runs because it is after GracefulStop in the sequence.
5. **F_FULLFSYNC adoption**: implement the darwin fcntl (recommended, ~15 lines) vs document-only. Planner picks; either resolves the STATE blocker with the finding that process-kill tests are unaffected.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | go1.27.4 linux/amd64 [VERIFIED: `go version`] | — |
| `make test` (race suite) | baseline + new tests | ✓ | full suite green, ~7s wall [VERIFIED: run this session] | — |
| `make lint` (golangci-lint 2.15.0 via tool directive) | gate | ✓ (toolchain-pinned, `go tool`) | v2.15.0 | — |
| Loopback broadcast fan-out (Linux) | real-socket + exec scenarios | ✓ [VERIFIED: probes this session] | — | loud skip (darwin, D-12) |
| Free UDP/TCP ports on test hosts | scenarios | ✓ | — | port pickers exist |
| GitHub Actions runners (linux amd64/arm64, macos-15, darwin-cross) | CI | ✓ (existing matrix) | — | darwin cells loud-skip |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | stdlib `testing` (+ `pgregory.net/rapid` v1.3.0 test-only, unchanged); `-race` mandatory |
| Config file | none (per-package _test.go files; vendored .golangci.yml governs test-file lint relaxations [VERIFIED: .golangci.yml:457-468 excludes funlen/gochecknoglobals et al. for `_test.go`]) |
| Quick run command | `go test -race ./...` (≈7s baseline [VERIFIED: this session]) |
| Full suite command | `make test` (= `go test -race -cover ./...`); `make lint` gate alongside |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| OBS-01 | metric families rendered, values correct from seams | unit + integration | `go test -race ./internal/metrics/ ./cmd/sup-kv/ -run 'TestMetrics'` | ❌ Wave 0 |
| OBS-01 | unacked count/oldest age published (needs new mirror) | unit (adapter) | `go test -race ./internal/udprepl/ -run 'TestMirror'` | ❌ Wave 0 |
| OBS-01 | dump duration/errors counted (needs engine seam) | unit | `go test -race ./internal/app/persist/ -run 'TestDumpStats'` | ❌ Wave 0 |
| OBS-01 | merge conflicts + drift warnings counted (needs core counters) | unit | `go test -race ./internal/app/repl/ -run 'TestMergeStats'` | ❌ Wave 0 |
| OBS-02 | discarded-by-class exposed on /metrics | integration (httptest over the mux) | `go test -race ./internal/metrics/ -run 'TestDiscardMetrics'` | ❌ Wave 0 |
| OBS-03 | slog JSON on stderr, level env, format flag; discard classes distinguishable | unit (tap capture) | `go test -race ./cmd/sup-kv/ ./internal/udprepl/ -run 'TestLog'` | ❌ Wave 0 |
| OBS mTLS | plaintext mode; mTLS required-cert mode; incomplete set = exit 2 | integration | `go test -race ./cmd/sup-kv/ -run 'TestMetricsTLS'` | ❌ Wave 0 |
| TEST-04 | sim scenarios (partition-days, one-way, ack loss, clock jumps, far-ahead, bursts, empty-disk bound) | sim | `go test -race ./internal/app/repl/ -run 'TestScenario'` | ❌ Wave 0 (simnet exists; scenario tests + clock seam new) |
| TEST-04 | real-socket scenarios (dup node_id, proto mismatch, bad CRC, convergence) | integration | `go test -race ./internal/udprepl/ -run 'TestScenario'` | ❌ Wave 0 |
| TEST-04 | exec scenarios ×4 (kill-9, both-down, SIGTERM, empty-disk) | exec integration | `go test -race ./cmd/sup-kv/ -run 'TestExec'` | ❌ Wave 0 (harness exists; scenarios new) |
| TEST-05 | three benchmarks | benchmark | `make bench` | ❌ Wave 0 |
| D-15 fixes | 12 review fixes with regression tests | unit per fix | per-package `-run` on new regression tests | partially (existing suites pin current behavior) |

### Sampling Rate
- **Per task commit:** `go test -race ./...` + `go tool golangci-lint run ./...` (the project's standing rule: `make lint && make test` before push)
- **Per wave merge:** same full suite on all 4 CI cells (push to PR)
- **Phase gate:** full suite green before `/gsd:verify-work`; benchmarks by hand (`make bench`) — no CI thresholds (D-13)

### Wave 0 Gaps
- [ ] `internal/metrics/metrics_test.go` — exporter rendering (ordering, escaping, HELP/TYPE) + mux/TLS assembly (REQ OBS-01/02)
- [ ] `internal/udprepl/scenario_test.go` — asymmetric-topology helper + real-socket scenarios (REQ TEST-04)
- [ ] `cmd/sup-kv/scenario_test.go` — the 4 exec scenarios on the existing harness (REQ TEST-04)
- [ ] `internal/app/repl/sim_scenario_test.go` + simnet clock-offset seam (REQ TEST-04)
- [ ] `internal/app/persist/bench_test.go`, `internal/grpcapi` (or udprepl) latency bench, `internal/udprepl` burst bench (REQ TEST-05)
- [ ] Makefile `bench` target
- Framework install: none needed.

## Security Domain

`security_enforcement: true`, ASVS level 1, block on high [VERIFIED: .planning/config.json].

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes (metrics surface only) | Optional client-cert mTLS: `tls.RequireAndVerifyClientCert` + ClientCAs pool; opt-in listener (D-01/D-02); replication segment stays trust-model-documented (no auth in v1 — out of scope, SEC-01 deferred) |
| V3 Session Management | no | Stateless metrics endpoints; no sessions |
| V4 Access Control | yes | /debug surface exposure control: default OFF; plaintext mode documented for loopback/debug; mTLS for non-loopback |
| V5 Input Validation | yes | No user input reaches the exporter (label values are fixed enums); metrics-addr parsed by net.Listen; cert files by tls.LoadX509KeyPair (file errors = startup exit 2); /debug/pprof is GET-only (Go 1.22+) |
| V6 Cryptography | yes | stdlib crypto/tls only — never hand-roll; explicit `MinVersion: tls.VersionTLS12` floor recommended; test certs via crypto/x509+ecdsa in-test generation |
| V14 Config | yes | SUPKV_METRICS_* validation: incomplete cert triple = startup error (D-02); malformed SUPKV_LOG_LEVEL/FORMAT = exit 2 (existing env-error pattern) |

### Known Threat Patterns for {Go UDP service + HTTP debug surface}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Debug endpoint exposure (pprof heap dumps = information disclosure; profile = DoS) | Information Disclosure / DoS | Opt-in listener (D-01), optional mTLS (D-02), single listener (D-03); docs advise loopback bind for plaintext |
| Datagram flood → log spam amplification | DoS | Counters (O(1)) carry the signal; discard-class log lines rate-limited or Debug-level (planner detail) |
| Bad-CRC / malformed datagram flood | DoS | Existing: bounded buffer, classification counters; no per-datagram allocation growth |
| TLS downgrade / weak ciphers | Spoofing | crypto/tls defaults + explicit MinVersion floor; no custom cipher config |
| Cert/key file path traversal or world-readable keys | Information Disclosure | Paths from operator config only (not user input); LoadX509KeyPair errors are startup-fatal; file perms left to deployment (document in Phase 7 ops docs) |
| kill -9 during dump (torn file) | Tampering | Existing: temp+fsync+rename+.prev fallback chain (PERS-02) — exec scenario 1 re-proves it end-to-end |

## Sources

### Primary (HIGH confidence)
- Codebase, read this session: internal/udprepl/repl.go, internal/app/repl/core.go (Config/DiscardStats/UnackedStats/egress), internal/app/storage/store.go (Merge/Conflict/OnChange), internal/app/persist/engine.go (runDump/ApplyRemote/Flush), internal/app/clock/clock.go (drift semantics), internal/app/internal/simnet/simnet.go (Policy/Cluster/clock wiring), internal/udprepl/smoke_test.go (probe + loud-skip), cmd/sup-kv/main.go + main_test.go (config contract, exec harness), internal/persistfile/dumpfile.go (atomic write), .golangci.yml (depguard/sloglint/gosec), .github/workflows/ci.yml, Makefile, go.mod, all five 0N-REVIEW-DISPOSITION.md files
- Empirical probes (this session, this Linux host): shared-port unicast self-delivery; asymmetric cross-broadcast full bidirectional delivery; 0.0.0.0-destination unicast delivery
- Go 1.27.4 toolchain source: zsyscall_darwin_{amd64,arm64}.go (Fsync binds libc_fsync only), zerrors_darwin_*.go (F_FULLFSYNC=0x33 defined, unused by Fsync)
- pkg.go.dev: crypto/tls (ClientAuthType values verbatim), net/http/pprof (custom-mux mounting), expvar (Handler), log/slog (NewJSONHandler/HandlerOptions/Level.UnmarshalText), testing (B.ReportMetric)
- prometheus.io/docs: exposition formats (0.0.4 escaping/ordering/Content-Type), naming practices (units, _total, labels)

### Secondary (MEDIUM confidence)
- Apple fcntl(2) man page (keith.github.io/xcode-man-pages mirror of the Apple man page): F_FULLFSYNC semantics quote
- .planning/phases/05-grpc-api-and-binary/05-CONTEXT.md (D-16..D-19 operational contract)

### Tertiary (LOW confidence)
- gosec rule ID for timeout-less HTTP serving [ASSUMED, A1]

## Project Constraints (from CLAUDE.md / AGENTS.md)

- Go version pinned by go.mod (`go 1.27`, toolchain go1.27.4); no raise without an ADR.
- `make lint` + `make test` before pushing; CI runs the same. Red gate blocks merge.
- Depguard boundary machine-enforced: `internal/app` may not import net/os/syscall/google.golang.org; `internal/port` stdlib+own-module only. The metrics listener, slog handler construction, exporter, and F_FULLFSYNC code all live OUTSIDE internal/app (adapters/main). slog itself stays out of internal/app by discipline (D-09) even though stdlib-legal.
- .golangci.yml is the vendored powerman strict config — byte-unmodified upstream; project rules additive in depguard.rules only; never weaken it (fix direction only).
- Every `//nolint` names its linter with a written reason (nolintlint). Existing sanctioned nolints: gosec G304 (persistfile path reads), gosec G115 (guarded uint16), gochecknoglobals (ldflags commit var).
- Zero new direct runtime requires; tools via `go tool` only. Phase 6 requires ZERO dependency changes.
- PR-only landing; CI check-run names are a ruleset contract — the P1-WR-01 timeout-minutes change must not rename `lint` or `gate (<cell>)`.
- Layout: adapters appear with the phases that need them; `cmd/sup-kv/main.go` holds flags/env/signals/wiring only.
- sloglint (vendored): static-msg, capitalized, context:scope, forbidden keys — all new slog code must comply.
- User global instruction: fresh feature branch from up-to-date master for new work (GSD phase branching applies).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — everything stdlib; signatures verified against pkg.go.dev this session
- Architecture: HIGH — seam inventory read from source with line citations; exec harness exists; topology empirically probed
- Pitfalls: HIGH — sloglint/gosec/depguard constraints read from the vendored config; UDP topology probed; two MEDIUM items (gosec rule ID, darwin runner behavior) flagged as assumptions

**Research date:** 2026-11-15
**Valid until:** 2027-01-15 (stable domain; stdlib + repo facts do not rot)
