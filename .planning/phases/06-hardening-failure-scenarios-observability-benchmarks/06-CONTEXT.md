# Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks - Context

**Gathered:** 2026-11-15
**Status:** Ready for planning

<domain>
## Phase Boundary

The evidence phase: (1) every failure scenario from docs/IDEA.md becomes an automated passing test — a mix of in-process real-socket scenarios and 4 exec-level two-process runs in CI; (2) an opt-in metrics endpoint (in-repo Prometheus text format, no client library) exposing the OBS-01/OBS-02 families plus Go runtime and replication topology; (3) structured slog JSON logging with distinguishable dropped-datagram classes; (4) three benchmarks per TEST-05, run by hand via `make bench` (no CI gates). Additionally: ALL open code-review findings from phases 1–4 (4 Warning + 8 Info) are folded into this phase's plans — that is the hardening backlog.

Scope anchor: requirements OBS-01, OBS-02, OBS-03, TEST-04, TEST-05 plus the folded review findings. The engine, replication, persistence, API and binary already exist (Phases 2–5); this phase adds no new product capability — it makes the existing one provable and operable.

Explicit scope changes made by the maintainer during this discussion:
- The conflict journal (docs/IDEA.md §Observability "lost versions logged on merge conflicts") is CANCELLED — an LWW loser is an ordinary overwrite, not an incident. Merge losers stay silent (consistent with phase-5 D-06). The merge-conflicts COUNTER stays (OBS-01).
- TEST-04's "conflict-journal entries" evidence in partition scenarios becomes: deterministic winner asserted + merge-conflicts counter — the ROADMAP success-criterion wording is to be updated at planning time.

</domain>

<decisions>
## Implementation Decisions

### Metrics endpoint (OBS-01, OBS-02)
- **D-01:** Opt-in via `--metrics-addr` flag / `SUPKV_METRICS_ADDR` env, NO default — flag unset ⇒ no endpoint, no listener. — **Reversibility:** one-way — this is the published operational contract.
- **D-02:** mTLS is optional on the metrics listener: when `SUPKV_METRICS_CERT`/`SUPKV_METRICS_KEY`/`SUPKV_METRICS_CLIENT_CA` are set the listener requires client certificates; with only `--metrics-addr` set it serves plaintext (loopback/debug use). Incomplete cert sets (some flags set, some not) are a startup error. — **Reversibility:** costly — cert plumbing and operator runbooks bind to it.
- **D-03:** One listener serves the whole debug surface: `/metrics` (Prometheus text), `/debug/vars` (stdlib expvar), `/debug/pprof/*` (net/http/pprof) — all behind the same optional mTLS.
- **D-04:** Metric families: the OBS-01/OBS-02 minimum (unacked-records count and oldest age, dump duration and errors, merge conflicts, clock-drift warnings, discarded datagrams by class: bad CRC / own broadcast / bad proto version) PLUS `go_*`/`process_*` runtime families PLUS replication topology (known-peer count, datagrams sent/retransmitted by message type Hello/Data/Ack/Sync/DataRequest). In-repo text-format exporter (~100 lines, sorted families/labels, `text/plain; version=0.0.4`) — `prometheus/client_golang` is rejected (adds 3+ modules against the gRPC+protobuf-only bar; decision recorded in .claude/CLAUDE.md §rejected).
- **D-05:** Naming under the `supkv_` prefix following Prometheus conventions (e.g. `supkv_replication_unacked_records`, `supkv_dumps_duration_seconds`, `supkv_datagrams_discarded_total{class=...}`) — planner finalizes exact names.

### Structured logging (OBS-03)
- **D-06:** stdlib `log/slog` with the JSON handler as the default; a text-handler flag (`SUPKV_LOG_FORMAT=text|json`, default json) for local reading. Zero new dependencies. — **Reversibility:** one-way — log field names become the parsing contract for journald/log shippers.
- **D-07:** Output stays on **stderr** (current stdlib-log behavior preserved); level via `SUPKV_LOG_LEVEL` (`debug|info|warn|error`, default `info`).
- **D-08:** Every dropped-datagram class is a distinguishable structured attribute (slog attr like `class="bad_crc"|"own_broadcast"|"bad_proto_version"`), riding the existing `udprepl` Log tap and discard classifier — the classes already exist as counters and a total classification (TestDiscard_classes/totality).
- **D-09:** The pure core (`internal/app/**`) stays silent — all logging happens at adapters/main via existing seams (the `udprepl` Log callback, the OnChange-style taps), preserving the depguard I/O-free boundary. slog is stdlib and technically legal in internal/app, but the discipline is no logging in the decision core.

### Conflict journal — CANCELLED
- **D-10:** No conflict journal in any form (no file, no per-conflict log events). Maintainer's rationale, verbatim intent: "если запись была перезаписана — она была перезаписана, это нормальная работа, а не событие, требующее разбора". A merge loser in an LWW map is semantically an ordinary overwrite. Only the merge-conflicts COUNTER (OBS-01) remains. Partition-scenario tests assert the deterministic winner and the counter, not journal entries; update TEST-04 wording in ROADMAP/REQUIREMENTS at planning time to match. — Supersedes docs/IDEA.md §Observability "Conflict journal" line and the phase-5-era journal mentions.

### Real-socket tests in CI (TEST-04)
- **D-11:** Mixed strategy: the bulk of failure scenarios run in-process on real UDP sockets (the existing smoke-test pattern, `TestSmoke_twoNodesUDP`), and an exec harness covers exactly 4 two-process scenarios: (1) kill -9 one node then restart — convergence without operator action; (2) both-nodes-down restart; (3) SIGTERM graceful shutdown with final confirmed dump (D-19 contract, incl. exit codes); (4) empty-disk node start against a live peer (recovery at `sync_response_rate`). The harness builds the real binary (`go build`, TestMain or helper) and drives clients over real gRPC.
- **D-12:** On darwin the shared wildcard bind fails (EADDRINUSE, empirically resolved in D-08/PR #31) — exec and socket scenarios take the loud-skip precedent: run, print the reason, skip. No build-tag hiding of tests.

### Benchmarks (TEST-05)
- **D-13:** Three benchmarks, run by hand via a new `make bench` target (`go test -bench ./...` scoped); CI does NOT run benchmarks and there are NO performance thresholds — shared-runner noise makes them flaky. (1) `BenchmarkDump` at 10k keys; (2) replication burst throughput (~1000 writes/s, two nodes, convergence rate); (3) bounded confirmation latency — `wait_for_replication` latency under burst, reported as a benchmark metric.
- **D-14:** No benchmark artifacts/profiles stored in CI (cpuprofile et al. remain a local `go test -bench -cpuprofile` convenience, not a pipeline).

### Folded review findings (hardening backlog)
- **D-15:** ALL 12 open review findings from the 2026-11-14 re-reviews fold into this phase's plans (maintainer: "Влить всё"). The 4 Warnings: [P4-WR-01] drained-row `Acked` duplicate after discovery flush (fix: `slices.Compact` after sort in udprepl mirror); [P4-WR-02] `sync_response_rate` above 1e9 divides to zero and silently disables the egress limiter (fix: clamp out-of-domain rate to the default in `normalized()`); [P1-WR-01] no `timeout-minutes` on ci.yml jobs (15/20 min); [P1-IN-01→Warning-class fix] `serve()`'s `defer stop()` releases signal handling before shutdown completes (second signal during GracefulStop kills with exit 143, no final dump). The 8 Infos: [P4-IN-01] NotifyLocalWrite TOCTOU vs closed done; [P4-IN-02] Restore marks dirty unconditionally — SAME class as [P3-IN-02] ApplyRemote marks dirty on no-op merges (planner merges these two into one redundant-dump item); [P3-WR-01 is an Info-class scope decision] the provably-unreachable above-cover guard in `confirmWaiters` — resolve as: keep the documented belt AND pair it with an invariant assertion (do not silently delete); [P3-IN-01] dead Decode-error fallback in `Open`; [P2-IN-01/IN-02] OnChange doc-comment inaccuracies (tombstone wording; re-entrancy boundary); [P2-IN-03] duplicated subtest in store_test.go. Each fix updates the corresponding `0N-REVIEW-DISPOSITION.md` row to fixed.
- **D-16:** The 2026-11-14 buf zero-waiver refactor (7b55ad1) intentionally made VERIFICATION fingerprints of phases 1/4/5 stale; re-running `/gsd-execute-phase 1/4/5` to regenerate them is folded into this phase's execution flow (before milestone close), not new feature work.

### Claude's Discretion
- Exact Prometheus metric/label names and help strings (D-05 direction fixed).
- slog handler wiring details (where the handler is constructed in main, how the `udprepl` Log tap maps to slog attrs).
- Internal layout of the metrics exporter package and the exec-test harness (build helper, port allocation, process lifecycle) — follow the phase-4/5 adapter precedents.
- Which of the in-process socket scenarios beyond the IDEA list get dedicated tests vs. parameterized cases — planner decides with the researcher.
- macOS F_FULLFSYNC durability policy (open STATE blocker) — researcher investigates and proposes; if a code change is needed it lands here as a small plan item.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design (authoritative)
- `docs/IDEA.md` — §Observability (metrics list, structured logging; NOTE: the "Conflict journal" line is CANCELLED by D-10), §Failure scenarios (the TEST-04 list), §Parameters (defaults incl. `sync_response_rate`, `drift_warn`), §Trust model, §Out of scope for v1.
- `.claude/CLAUDE.md` — §Buf CLI (per-tree generate templates, zero-waiver lint), §rejected dependencies (in-repo Prometheus text exporter + expvar decision — the metrics-form basis), tech-stack and tool-pin conventions.

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` — OBS-01/02/03, TEST-04/05 rows (TEST-04 wording updated per D-10 at planning time).
- `.planning/ROADMAP.md` — Phase 6 section (goal + success criteria; same TEST-04 wording update).

### Prior-phase context
- `.planning/phases/05-grpc-api-and-binary/05-CONTEXT.md` — D-16..D-19 operational contract (SUPKV_ env, flag>env>default precedence, --version, graceful shutdown) that the metrics/log flags must follow; NOTE: its service/message name pins (SupKV/WatchEvent) were superseded 2026-11-14 by the buf zero-waiver rename (SupKVService/WatchResponse, STATE.md decision).
- `.planning/phases/04-replication-and-deterministic-simulator/04-CONTEXT.md` — SUPKV_SIM_SEED replay hook, discard classification, smoke-test precedent.
- `.planning/phases/01..04/*-REVIEW-DISPOSITION.md` — the open findings folded by D-15 (per-phase row state).

### Project conventions
- `AGENTS.md` — layout, dependency policy, lint-gate rules (fix direction satisfies the gate, never weakens the config).
- `.golangci.yml` — the depguard boundary the logging/metrics adapters must respect.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/udprepl/repl.go` — discard-classification counters (badCRC/undecodable/badProtoVersion/ownEcho/duplicateNodeID atomics), known-peer count, copy-on-write pending mirror (unacked per key with ages) — the metrics endpoint mostly PUBLISHES what already exists.
- `internal/app/repl/core.go` `UnackedStats(now)` — pure unacked/oldest-age read; the adapter seam for OBS-01 age metrics.
- `internal/app/clock/clock.go` — drift signal returned from Merge (driftWarn-gated); consumption point for drift-warning counters/logs is the adapter, not the core.
- `internal/grpcapi/gate.go` — gRPC health server bound 1:1 to the startup gate (the metrics listener is a separate, opt-in surface).
- `internal/udprepl/smoke_test.go` `TestSmoke_twoNodesUDP` — the real-socket in-process pattern (and the loud darwin skip).
- `cmd/sup-kv/main.go` — config plumbing (flags/env, `SUPKV_` prefix, exit codes); the `Log:` callback wired to stdlib log — the slog swap point.
- `internal/app/storage/store.go` `Merge` returning `[]Conflict` + drift — already the single convergence point; D-10 means these returns are only counted, never logged per-item.

### Established Patterns
- Consumer-side interfaces + adapter taps (OnChange tap, Log callback, ServerOptions seams) — all new observability rides existing seams; zero new core surface.
- depguard-enforced purity: `internal/app` may not import net/os/syscall/google.golang.org — the HTTP listener, slog handler and exporter live outside the core.
- Pinned tools via go.mod tool directives invoked with `go tool` — a `make bench` target follows the Makefile target convention.
- PR-only landing for code and docs; CI check-run names are a ruleset contract (any ci.yml job change — e.g. timeout-minutes — must not rename check contexts).

### Integration Points
- `cmd/sup-kv/main.go` — flags/env registration, listeners lifecycle (metrics listener joins the graceful-shutdown ordering), slog handler construction, exit codes.
- `.github/workflows/ci.yml` — exec-scenario cells run inside the existing gate matrix (no new jobs needed; timeout-minutes addition per P1-WR-01).
- `Makefile` — new `bench` target; `buf-*` targets unchanged.

</code_context>

<specifics>
## Specific Ideas

- Maintainer's explicit stance on conflicts (D-10), quoted for downstream agents: an overwritten record is normal operation, not an event requiring investigation — do not re-introduce per-conflict logging in any later phase without a new maintainer decision.
- The metrics/debug surface follows the "one listener, one mTLS" shape (D-03) — no per-endpoint auth divergence.
- `make bench` output should be copy-paste friendly for before/after comparisons (stable benchmark names).

</specifics>

<deferred>
## Deferred Ideas

- CI job with benchmark thresholds / stored profiles — rejected for now (shared-runner noise); revisit on dedicated hardware if ever needed.
- HMAC authentication of replication datagrams — docs/IDEA.md future-work item, out of scope for v1 entirely.
- Conflict journal in ANY form — cancelled by D-10, not merely deferred; re-introduction requires a new maintainer decision.

</deferred>

---

*Phase: 6-Hardening — Failure Scenarios, Observability, Benchmarks*
*Context gathered: 2026-11-15*
