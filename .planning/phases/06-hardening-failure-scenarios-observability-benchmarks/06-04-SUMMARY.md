---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 04
subsystem: observability-metrics
tags: [prometheus-text-format, metrics-endpoint, mtls, pprof, expvar, opt-in-listener, http, observability]

requires:
  - phase: 06-hardening-failure-scenarios-observability-benchmarks
    provides: the 06-03 counter seams (Adapter.Discards/KnownPeers/UnackedStats/MergeStats/Retransmissions/SentByType, Engine.DumpStats) this exporter publishes
  - phase: 05-grpc-api-and-binary
    provides: the cmd/sup-kv config contract (SUPKV_ env, flag>env>default, exitUsage 2), listen-then-serve precedent and the exec smoke harness
provides:
  - OBS-01 and OBS-02 delivered — an opt-in, optionally mTLS-secured one-listener metrics/debug endpoint exposing the full supkv_ family set plus go_*/process_* runtime families
  - internal/metrics package — in-repo Prometheus 0.0.4 text exporter (family model, writer-side sorting, exposition escaping), Sources/Snapshot closed seam set, NewMux debug-surface assembly, NewTLSConfig mTLS builder
  - --metrics-addr flag / SUPKV_METRICS_ADDR env with NO default (D-01) and the env-only all-or-nothing SUPKV_METRICS_CERT/KEY/CLIENT_CA triple validated before any bind (D-02, exit 2)
  - /debug/vars and the whole /debug/pprof/* surface (index, cmdline/profile/symbol/trace, six named profile routes) behind the same optional mTLS (D-03), named imports only
affects: [06-05 through 06-09 (green suite dependency), phase verification, ops documentation]

actuals:
  tokens: 14600   # chars/4 over the realized diff (58361 diff chars across 4 files)
  tasks: 3
  commits: 3      # MEASURED: git rev-list --count d4dbd76..HEAD
plan_head_before: d4dbd767ba90f004abe8fb78c1f98a8856f26310
plan_head_after: b29600697039aeee1c9df355b3182923791dc645

tech-stack:
  added: []        # stdlib only — net/http, crypto/tls, crypto/x509, expvar, net/http/pprof, syscall; go.mod/go.sum byte-identical (tidy gate)
  patterns:
    - "pure formatter over a closed seam set: Sources is a fixed struct of accessor closures, Snapshot reads each once per scrape, no registry lifecycle ever"
    - "writer-side sorting and escaping: families sorted by name, labels by label name, HELP escapes backslash+LF while label values also escape double quotes — byte-pinned by the exact-render test"
    - "one-listener one-auth-story debug surface: every debug route mounted by name on a fresh mux, never a blank import (Pitfall 4)"
    - "discriminating no-default-listener proof: two concurrent flag-less nodes must both serve — a fixed default port collapses the second node's bind, so the test fails under the exact D-01 regression"

key-files:
  created:
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
  modified:
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go

key-decisions:
  - "TLS builder landed in Task 2 as the metrics.NewTLSConfig helper (the plan's sanctioned 'Task 3 builder helper or its direct construction' choice), so listenMetrics is complete from its first commit; Task 3 adds only the enforcement tests and the debug mux"
  - "The cert triple resolves env-only in registerConfig and validates all-or-nothing in finalizeConfig (exit 2 naming all three envs) — before wire() could ever bind (Pitfall 8); the cert envs have no flags, so the usage dump cannot carry their names: only the error line does, which is what the test greps"
  - "wire() guards the D-01 opt-in (empty metricsAddr means no listenMetrics call at all — no listener, no goroutine) while listenMetrics keeps the listen-then-serve fatal-bind split; a bind failure aborts startup, which is what makes TestMetrics_noDefaultListener's collision shape discriminate"
  - "Metrics Shutdown sits immediately after the health NOT_SERVING flip in the D-19 sequence (documented in the shutdown doc comment) — scrapers see the close the moment the node stops serving; both mTLS tests' closing SIGTERM walks exercise it with TLS active"
  - "processUsage normalizes ru_maxrss to bytes at runtime (KiB on Linux, already bytes on darwin) — one GOOS switch instead of build-tagged files, keeping the plan's artifact list intact"
  - "Go's plaintext-to-TLS behavior is asserted honestly: net/http answers a plain GET with its 400 'Client sent an HTTP request to an HTTPS server' record-header probe, so the test pins 'never a served metrics surface' rather than a transport error the platform does not produce"

requirements-completed: [OBS-01, OBS-02]

coverage:
  - id: D1
    description: "In-repo Prometheus 0.0.4 text exporter — family model, writer-side sorting (families by name, labels by label name), one HELP/TYPE per name before the first sample, exposition escaping (HELP: backslash+LF; label values: backslash, double quote, LF), ParseFloat values, exact Content-Type over HTTP"
    requirement: OBS-01
    verification:
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestRender_exact (byte-exact adversarial HELP/label-value table)"
        status: pass
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestRender_sorted (construction order never leaks into the rendering)"
        status: pass
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestMetricsHandler_serves (httptest: 200, exact Content-Type, HELP/TYPE strictly before samples)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The closed supkv_ family snapshot — 11 families over the 06-03 seams (unacked count+oldest age, peers, sent by type, retransmitted, discards by class, merge conflicts, drift warnings, dump duration/total/errors) plus go_*/process_* runtime families; zero-state publishes every family with zero-valued samples"
    requirement: OBS-02
    verification:
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestMetrics_families (fake zero-state snapshot: every family exactly once with its TYPE, full class/type label vocabulary, zero-valued samples)"
        status: pass
    human_judgment: true
    rationale: "Rendering and completeness are machine-verified, but the plan's flagged assumption A2 (06-RESEARCH Standard Stack A2) marks the minimal go_*/process_* enumeration — go_goroutines, go_memstats_alloc_bytes, go_memstats_sys_bytes, process_cpu_seconds_total, process_resident_memory_bytes — as a judgment call for human confirmation at verify: a human may judge the runtime enumeration insufficient."
  - id: D3
    description: "Opt-in listener config contract — D-01 (unset means no listener at all, no default port, no goroutine) and D-02's all-or-nothing cert triple exiting 2 before any bind with all three env names"
    requirement: OBS-01
    verification:
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestMetrics_optIn (real binary: --metrics-addr scrape carries the full supkv_ surface)"
        status: pass
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestMetrics_noDefaultListener (discriminating two-node collision shape — a fixed default metrics port breaks the second node's bind)"
        status: pass
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestMetrics_incompleteCertSet (exit 2, stderr names SUPKV_METRICS_CERT/KEY/CLIENT_CA)"
        status: pass
    human_judgment: false
  - id: D4
    description: "mTLS enforcement + one-listener debug surface — RequireAndVerifyClientCert with the client CA pool and a TLS 1.2 floor (no cipher overrides); plaintext loopback mode when no triple is set; /debug/vars and the whole /debug/pprof/* surface behind the same mTLS, named imports only"
    requirement: OBS-02
    verification:
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestMetricsTLS_required (in-test x509+ecdsa PKI: plain GET never yields the surface, no-cert handshake fails, authorized pair scrapes /metrics and /debug/vars)"
        status: pass
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestDebugSurface (pprof index and the heap named-profile route answer behind the same mTLS listener)"
        status: pass
    human_judgment: false

duration: 19min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 04: Metrics/Debug Endpoint Summary

**The opt-in metrics/debug endpoint (OBS-01/OBS-02): an in-repo Prometheus 0.0.4 text exporter over the 06-03 counter seams publishing the full supkv_ family set plus go_*/process_* runtime families, wired into main behind --metrics-addr with no default (D-01), an env-only all-or-nothing mTLS cert triple validated before any bind (D-02), and one listener carrying /metrics with /debug/vars and the whole /debug/pprof surface (D-03) — zero new dependencies, go.mod byte-identical.**

## Performance

- **Duration:** 19min
- **Started:** 2026-11-16T00:00:36Z
- **Completed:** 2026-11-16T00:19Z (UTC)
- **Tasks:** 3/3
- **Files modified:** 4 (2 created)

## Accomplishments

- **Task 1 (tracer):** `internal/metrics` — the repo's first HTTP code. Family/Sample/Label model; `Render` with writer-side sorting, one HELP/TYPE per name before the first sample, and the exposition's asymmetric escaping (HELP: backslash+LF; label values: backslash, double quote, LF); `NewHandler` as a pure formatter over a fixed snapshot function. Tracer feedback gate re-ran verify end-to-end under auto mode: 3/3 PASS — expanded.
- **Task 2:** the `Sources` closed seam set + `Snapshot` (11 supkv_ families with the fixed class/type label enums + 5 runtime families read at scrape time, Getrusage with the KiB-to-bytes normalization), `NewTLSConfig` (RequireAndVerifyClientCert, CA pool, TLS 1.2 floor, stdlib ciphers), and the main wiring: `--metrics-addr`/`SUPKV_METRICS_ADDR` with empty default, env-only cert triple all-or-nothing in `finalizeConfig` (exit 2 naming all three envs), `listenMetrics` with the fatal-bind split and explicit `http.Server{ReadHeaderTimeout}`, and the metrics `Shutdown` right after the health NOT_SERVING flip in the D-19 order.
- **Task 3:** `NewMux` — the one-listener debug surface (expvar + full pprof route set, six named profile endpoints via `pprof.Handler`, named imports only); `TestMetricsTLS_required` with the in-test crypto/x509+ecdsa PKI and `TestDebugSurface` over the heap named-profile route.

## Task Commits

1. **Task 1 (tracer): exporter — family model, sorted 0.0.4 rendering, /metrics handler** — `3714924` (feat)
2. **Task 2: supkv_* families + opt-in listener with cert-triple validation** — `c59db3c` (feat)
3. **Task 3: mTLS enforcement + /debug surface (expvar, pprof)** — `b296006` (feat)

**Plan metadata:** (this commit) `docs(06-04): complete metrics endpoint plan`

## Files Created/Modified

- `internal/metrics/metrics.go` — NEW: package doc naming the adapter boundary; Family model; Render + escapes; NewHandler; Sources/Snapshot (replication/traffic/dump/runtime family builders); processUsage; NewMux; NewTLSConfig
- `internal/metrics/metrics_test.go` — NEW: TestRender_exact, TestRender_sorted, TestMetricsHandler_serves, TestMetrics_families
- `cmd/sup-kv/main.go` — metrics env consts/fields/flag, resolveMetricsCerts + checkMetricsCerts, wired.metricsSrv, listenMetrics, metricsSources, abortWire, newReplicationAdapter, registerDurationFlags (funlen splits), shutdown position + doc
- `cmd/sup-kv/main_test.go` — httpGetBody, TestMetrics_optIn, TestMetrics_noDefaultListener, TestMetrics_incompleteCertSet, metricsCertSuite/generateMetricsCerts/startMTLSNode/tlsGet, TestMetricsTLS_required, TestDebugSurface

## Decisions Made

See key-decisions frontmatter. The load-bearing ones: the D-01 guard lives in wire() (so "no listener, no goroutine" is structural, not a nil check inside the listener path), and the metrics Shutdown position is pinned in the D-19 documentation the plan required (Pitfall 7).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Lint] First-cut metrics_test.go shapes vs the strict config**
- **Found during:** Task 1 lint gate
- **Issue:** testpackage (external test package), noctx (http.Get), noinlineerr (inline error handling), paralleltest (defer with t.Parallel), staticcheck QF1001, gocritic httpNoBody all fired on the first version.
- **Fix:** external `metrics_test` package, `http.NewRequestWithContext` + client.Do with `http.NoBody`, plain error assignments, no defer in parallel tests, De-Morgan rewrite — all in code direction.
- **Files modified:** internal/metrics/metrics_test.go
- **Verification:** `go tool golangci-lint run ./internal/metrics/...` exit 0; all three tracer tests PASS
- **Committed in:** 3714924

**2. [Rule 1 - Lint] metrics.go strict-gate shapes**
- **Found during:** Task 2 lint gate
- **Issue:** funlen on Snapshot (132 > 60 lines), goconst on the repeated `type`/`class` label names, err113 on a dynamic error in NewTLSConfig.
- **Fix:** Snapshot split into replicationFamilies/trafficFamilies/dumpFamilies/runtimeFamilies; labelType/labelClass constants; errNoClientCertPEM sentinel wrapped with %w.
- **Files modified:** internal/metrics/metrics.go
- **Verification:** lint exit 0; TestMetrics_families + the Task 1 suite green
- **Committed in:** c59db3c

**3. [Rule 1 - Lint] main.go strict-gate shapes**
- **Found during:** Task 2 lint gate
- **Issue:** err113+errorlint on the cert-triple error, funlen on registerConfig (63) and wire (74), godoclint link forms, nilnil on listenMetrics' (nil, nil) return, revive if-return.
- **Fix:** %w wrap on errIncompleteCertSet; registerDurationFlags/newReplicationAdapter/metricsSources/abortWire extractions (zero behavior change — the adapter construction and flags are byte-identical moves); `[http.Server]`/`[http.ListenAndServe]` links; the D-01 empty-addr guard moved to wire() so listenMetrics never returns nil-nil; `return checkMetricsCerts(*cfg)`.
- **Files modified:** cmd/sup-kv/main.go
- **Verification:** lint exit 0; full race suite green; TestMetrics_incompleteCertSet still pins exit 2 with all three env names
- **Committed in:** c59db3c

**4. [Rule 1 - Bug] The plan's "plain HTTP GET fails" pinned a transport error Go does not produce**
- **Found during:** Task 3 first TestMetricsTLS_required run
- **Issue:** the plan expected a plain GET against the TLS listener to fail at the transport; Go's net/http answers plaintext-to-TLS with a 400 "Client sent an HTTP request to an HTTPS server" (the record-header probe), so `Do` returns a real response and the first assertion failed.
- **Fix:** the test pins the honest contract — a plain GET must never yield the metrics surface (no 200, no supkv_ body), accepting either the 400 rejection or a transport error; the probe's behavior is documented in the test.
- **Files modified:** cmd/sup-kv/main_test.go
- **Verification:** TestMetricsTLS_required green (all three client classes asserted)
- **Committed in:** b296006

---

**Total deviations:** 4 auto-fixed (3 Rule 1 lint-shape, 1 Rule 1 plan-prose expectation; none architectural — no Rule 4 checkpoints)
**Impact on plan:** None on scope — every artifact, test name and acceptance criterion the plan pinned landed as specified.

## Issues Encountered

None beyond the deviations above. No auth gates (pure in-repo work).

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — every surface ships with a binding test; no pinned suite was modified.

## Threat Flags

None beyond the plan's own register. The register's mitigations are live: T-6-07 (opt-in D-01, one mTLS listener D-02/D-03 — TestMetrics_noDefaultListener/TestMetricsTLS_required), T-6-08 (MinVersion TLS 1.2, stdlib ciphers, handshake enforcement test), T-6-09 (accept-with-documentation per the plan; Phase 7 ops docs restate the plaintext-loopback caveat), T-6-10 (validation before any bind — TestMetrics_incompleteCertSet).

## Next Phase Readiness

- The exporter reads exactly the 06-03 accessors; nothing new is needed from later plans. 06-05+ inherit a green full suite (race + lint) with the metrics endpoint in the tree.
- The plan's flagged assumption A2 (minimal go_*/process_* enumeration) surfaces as coverage D2's human-judgment flag for the phase verifier.

## Self-Check: PASSED

- key-files on disk: 4/4 FOUND (metrics.go + metrics_test.go created; main.go + main_test.go modified)
- Commits reachable from HEAD: 3714924, c59db3c, b296006 FOUND (merge-base --is-ancestor ok)
- evaluation-scope --plan 06-04 --commits-only: resolved, 3 commits
- Task acceptance criteria re-run: PASS — Task 1 (named imports only, adversarial escaping pinned, TYPE-before-sample, exact Content-Type, package lint 0); Task 2 (all 16 families with correct TYPE exactly once, no-default collision proof, exit 2 naming three envs, explicit http.Server ReadHeaderTimeout, go.mod unchanged, full race + lint 0); Task 3 (TLS 1.2 floor + RequireAndVerifyClientCert with no cipher overrides, no-cert handshake fails, named imports only repo-wide, pprof index + heap route behind mTLS, full suite green)
- Plan-level verification re-run: internal/metrics 4/4 PASS; cmd/sup-kv TestMetrics|TestDebug 5/5 PASS; go mod tidy byte-identical (git diff --exit-code); go tool golangci-lint run ./... exit 0
- Windows ledger: no stubs, skipped tests, or unrun verifies to record

---

*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
