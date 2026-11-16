---
phase: 06-hardening-failure-scenarios-observability-benchmarks
reviewed: 2026-11-16T12:00:00Z
depth: deep
files_reviewed: 33
files_reviewed_list:
  - .github/workflows/ci.yml
  - Makefile
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - cmd/sup-kv/scenario_test.go
  - internal/app/internal/simnet/simnet.go
  - internal/app/persist/bench_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_internal_test.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/core.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/merge_stats_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_bound_test.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/grpcapi/bench_test.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait_integration_test.go
  - internal/metrics/metrics.go
  - internal/metrics/metrics_test.go
  - internal/persistfile/dumpfile.go
  - internal/persistfile/fsync_darwin.go
  - internal/persistfile/fsync_other.go
  - internal/udprepl/bench_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/scenario_test.go
  - internal/udprepl/smoke_test.go
findings:
  critical: 0
  warning: 3
  info: 1
  total: 4
  necessity: 0
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-11-16T12:00:00Z
**Depth:** deep
**Files Reviewed:** 33
**Status:** issues_found

## Summary

Deep review of the phase-6 hardening surface: signal ownership and the D-19 shutdown order in `cmd/sup-kv/main.go`, the slog seams (`OnDiscard`, `MergeStats`, `Retransmissions`) through `repl.Core`/`udprepl.Adapter`, the opt-in mTLS metrics/debug endpoint (`internal/metrics`), the conditional-dirty `ApplyRemote` and `DumpStats` in the persist engine, the simnet clock-offset/restart seams and nine simulator scenarios, the real-socket and two-process exec suites, three benchmarks, and the darwin `F_FULLFSYNC` flush. Cross-file analysis covered the full wiring chain (main → metrics/udprepl/grpcapi/persist/storage, the WatchHub tap ordering at startup, the engine flush-vs-Run-exit races, the adapter mirror/retention correlation) and the import graph (no cycles; the `internal/app` purity boundary holds — no slog/goroutines in the reviewed `internal/app` files, wall-clock only through the sanctioned engine/sim seams).

The production code held up under adversarial tracing. The concurrency-critical paths this phase touched — `Engine.flush` vs `Run`'s exit sweep (both stranding shapes), `NotifyLocalWrite`'s post-send re-check, `retainDrained`'s Ack correlation and dedup, `ApplyRemote`'s displacement decision (all four merge-outcome cases traced, including the tombstone-probe blind spot, which is documented and errs safe) — are each correct and backed by targeted regression tests. Verification evidence: `go build ./...` and `go vet ./...` clean; `go test -race` green for `internal/app/persist`, `internal/app/storage`, `internal/app/repl`, `internal/metrics`, `internal/grpcapi`, `internal/udprepl`.

What the review did surface: one test-reliability defect **observed firing during this review** (the ephemeral-port pick-to-bind race hits `cmd/sup-kv`'s log-pair probe, which hard-fails instead of retrying — the same race the phase's own exec suite explicitly hardened against), a cross-package contract-drift hazard (metrics re-stringifies the discard-class tokens instead of referencing the canonical constants), and a factually wrong mechanism in the metrics package's security-relevant comment about `DefaultServeMux` registration (verified against the Go 1.27 toolchain source). No Critical findings; no security vulnerabilities beyond the comment defect (the mTLS config is sound: `RequireAndVerifyClientCert`, client-CA pool, TLS 1.2 floor, default ciphers; plaintext mode is the documented opt-in loopback/debug shape).

Necessity assessment (#4100): performed per the rubric. Candidate belts (`confirmWaiters`' above-cover guard plus its new never-resolved assertion, the drain-path `Select` guards, `NotifyLocalWrite`'s re-check) were each traced; every one is a documented, deliberately-kept defense with written justification and a pinning test — the negative-space rules (defense in depth, test seams) exclude them. Zero necessity findings.

## Warnings

### WR-01: Ephemeral-port pick-to-bind race hard-fails the log-pair probe (observed flake during review)

**File:** `cmd/sup-kv/main_test.go:638-645` (probe), `cmd/sup-kv/main_test.go:127-148` (pickers); observed firing from `cmd/sup-kv/main_test.go:714`
**Issue:** `probeLoopbackBroadcast` binds the exact UDP port that `freeUDPPort` just picked and released, and treats a bind failure as `t.Fatalf`. Under the suite's parallel burst, a sibling test's throwaway `:0` bind can be handed the same recently-released ephemeral port and still hold it in the microsecond pick-to-bind window, so the probe's `SO_REUSEADDR` bind fails `EADDRINUSE` against the non-`REUSEADDR` holder and the test fails spuriously. This is not hypothetical: during this review's first `go test -race ./cmd/sup-kv/` run, `TestLog_jsonDefault` failed with `probe rx bind: bind udp4 0.0.0.0:56842: listen udp4: address already in use`; the identical rerun passed. The phase's own `cmd/sup-kv/scenario_test.go:96-147` names this exact race ("Pitfall 6's observed flake: a probe or node losing its just-picked port to a rival's `:0` assignment") and hardens against it twice — `scenarioUDPPort` picks from the private sub-ephemeral window with a cross-test registry, and `newExecPair` retries probe bind errors up to 5 times — but `main_test.go`'s probe/ports kept the vulnerable fatal pattern. The same unguarded `freeTCPPort` picks feed every parallel smoke node's gRPC/metrics binds, where a lost race surfaces as an `awaitServing` timeout instead.
**Fix:** Give `probeLoopbackBroadcast` the `newExecPair` treatment: on a probe rx/tx bind error, re-pick a fresh port and retry (bounded attempts), fataling only after the bound is exhausted; longer term, route the smoke suite's UDP ports through a shared `scenarioUDPPort`-style picker (private window below the ephemeral range plus the registry) so pick-to-bind is structurally race-free, as already done in `scenario_test.go`.

### WR-02: Metrics discard-class labels re-stringify the canonical tokens — silent contract-drift hazard

**File:** `internal/metrics/metrics.go:280-287`; canonical constants at `internal/app/repl/core.go:39-45`
**Issue:** `trafficFamilies` publishes the five discard-class label values as bare string literals (`"bad_crc"`, `"undecodable"`, `"bad_proto_version"`, `"own_broadcast"`, `"duplicate_node_id"`), while `repl` exports exactly these values as first-class constants (`DiscardBadCRC`, ...) whose doc states the observability surfaces "key on" them. A repo-wide grep confirms no non-test code outside `core.go` references the constants — the one consumer the constants were defined for duplicates them. The two sides are pinned by disjoint tests (`metrics_test.go` pins the literals; `repl`'s suites pin the tokens), so a future token rename on either side compiles green and passes its own tests while silently breaking the documented correspondence between `OnDiscard` events, the `Discards()` counters, and the `supkv_datagrams_discarded_total{class=...}` series — an operator correlating logs with metrics would see classes that no longer match.
**Fix:** Reference the constants as the label values:

```go
{Labels: []Label{{Name: labelClass, Value: repl.DiscardBadCRC}}, Value: float64(disc.BadCRC)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardUndecodable}}, Value: float64(disc.Undecodable)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardBadProtoVersion}}, Value: float64(disc.BadProtoVersion)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardOwnBroadcast}}, Value: float64(disc.OwnEcho)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardDuplicateNodeID}}, Value: float64(disc.DuplicateNodeID)},
```

(The existing `metrics_test.go` expectations keep passing unchanged; the compile-time link is what is gained.)

### WR-03: NewMux comment states a false mechanism — pprof/expvar DO register on DefaultServeMux on any import

**File:** `internal/metrics/metrics.go:388-391`
**Issue:** The comment claims "neither expvar nor net/http/pprof is ever blank-imported, so nothing registers on [http.DefaultServeMux] (Pitfall 4)". Verified against the Go 1.27 toolchain source: `net/http/pprof`'s `init()` calls `http.HandleFunc("/debug/pprof/", Index)` (and the four siblings) on `http.DefaultServeMux` unconditionally — a plain named import registers exactly the same as a blank one — and `expvar`'s `init()` registers `/debug/vars` the same way (godebug-gated variant aside). The actual reason the process is safe is different and stronger: nothing in the binary ever serves `http.DefaultServeMux` (main constructs `http.Server{Handler: mux}` over a fresh `ServeMux`; no `http.ListenAndServe`/`(nil)` handler exists anywhere). As written, the comment is the only record of the Pitfall 4 mitigation and it is wrong about the mechanism; a future contributor trusting it could add any `http.ListenAndServe(addr, nil)` convenience (or a library could) and silently expose unauthenticated pprof/expvar, believing "nothing registers on DefaultServeMux".
**Fix:** Reword to state the true invariant, e.g.: "net/http/pprof and expvar register their handlers on http.DefaultServeMux from their init() on any import; the exposure control is that this process never serves DefaultServeMux — the only HTTP server is built over this fresh mux — and the whole surface (metrics, vars, pprof) rides the one listener behind D-02's optional mTLS."

## Info

### IN-01: SentByType counts datagrams as sent before unicast resolution and before the socket write

**File:** `internal/udprepl/repl.go:515-533` (countSent called at 522; resolve at 527-531; write at 532/555-560); accessor docs at `internal/udprepl/repl.go:431-452`
**Issue:** `flush` calls `a.countSent(ob.Msg)` immediately after a successful encode, before `net.ResolveUDPAddr` for unicasts and before `WriteToUDP` for both paths. A unicast whose handle fails to resolve, or a write that returns an error, is still counted in `supkv_replication_datagrams_sent_total` — while the `SentByType` accessor doc says the counters cover datagrams "encoded and handed to its socket" and the flush doc says "the datagram is on its way to the socket". Today the two docs contradict each other (flush's own doc says "encode-successful") and the behavior matches flush's doc, not the accessor's. Practically unobservable in production (handles originate from `src.String()` of received datagrams, so resolution essentially cannot fail), hence Info.
**Fix:** Either move `a.countSent(ob.Msg)` to after the successful `write` (making the accessor doc true), or align the `SentByType`/flush doc comments to say "encode-successful outbound messages" so the counter's contract is stated accurately.

---

_Reviewed: 2026-11-16T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
