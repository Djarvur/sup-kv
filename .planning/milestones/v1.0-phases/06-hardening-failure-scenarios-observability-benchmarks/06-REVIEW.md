---
phase: 06-hardening-failure-scenarios-observability-benchmarks
reviewed: 2026-11-18T11:13:49Z
depth: deep
files_reviewed: 3
files_reviewed_list:
  - Makefile
  - cmd/sup-kv/main.go
  - cmd/sup-kv/scenario_test.go
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
  necessity: 0
status: issues_found
---

# Phase 6: Code Review Report (incremental re-review)

**Reviewed:** 2026-11-18T11:13:49Z
**Depth:** deep
**Files Reviewed:** 3 (incremental re-review of the phase-06 covered files that changed since 2aaadb7)
**Status:** issues_found

## Summary

Incremental deep re-review of phase 06 after phase 7's additions. Scope is strictly the three covered files that moved since the phase's last full review (33 files, 2026-11-16): `Makefile` (+`release`/`release-snapshot` goreleaser targets), `cmd/sup-kv/main.go` (+gRPC reflection registration, D-09), and `cmd/sup-kv/scenario_test.go` (`scenarioPortLo` 21000 → 28100, the reserved doc-port band). The prior review's four standing findings were re-verified against today's tree and are preserved below (all still valid, all disposition `open`, all in files outside this re-review's scope that have not changed since 2aaadb7).

The delta itself held up under adversarial tracing. Every claim the new Makefile targets make was verified against the repo and the pinned toolchain: goreleaser v2.20.2 is the go.mod tool directive; `.goreleaser.yaml` carries the 4-target matrix (linux/darwin × amd64/arm64), `checksums.txt`, per-archive SBOMs and the CHANGELOG.md release header; `.github/workflows/release.yml` is the tag-triggered counterpart; `dist/` is gitignored; `.PHONY` covers both new targets. The scenario port-window change is a genuine fix, not a cosmetic one: the phase-7 doc tests (`readme_test.go`, `examples_test.go`) are `t.Parallel()` and run the README/EXAMPLES blocks verbatim through `bash -e`, binding the fixed doc ports 27901/27902 (README) and 28001/28002 (EXAMPLES) — all inside the OLD window [21000, 32678). Lifting the floor to 28100 removes that collision class structurally; the reserved band [27900, 28100) exactly covers the documented 279xx/280xx UDP and TCP ports, the new floor keeps every pick strictly below the Linux ephemeral range (32768), and no stale `21000` reference remains anywhere in the repo. The reflection registration in `listenGRPC` is the documented D-09 decision (public schema, loopback default bind, trusted-LAN threat model, zero new modules — same grpc module) and is load-bearing for the doc tests' `grpcurl list`; it adds no data exposure beyond the already-public proto schema.

The observability wiring in `main.go` still holds: the seven metrics seams (`metricsSources`), the opt-in mTLS/plain listener with `ReadHeaderTimeout`, the single slog handler install before any goroutine, and the D-19 shutdown order are all unchanged since the full review and re-confirmed by build and the passing suites.

Verification evidence (this review, 2026-11-18): `go vet ./...` and `go build ./...` clean; `make lint` (config verify + strict golangci-lint run) exit 0; `go test -race -count=1 ./cmd/sup-kv/` green — all four TEST-04 exec scenarios pass (`TestExec_killNineRestart` 0.58s, `TestExec_sigtermFinalDump` 1.45s, `TestExec_bothNodesDown` 1.80s, `TestExec_emptyDiskRecovery` 3.86s with recovery in 274ms against the 4s derived bound), and they passed while running in parallel with the doc suites and the quickstart broadcast probe, exercising exactly the port-band separation the change builds. No `internal/` file changed since 2aaadb7 (verified by diff), so the prior review's green internal-package evidence carries forward.

Necessity assessment (#4100): performed at deep depth per the rubric. Candidates in the delta (the reserved-band floor itself, `reflection.Register`, the unchanged belt-and-suspenders guards re-read in `main.go`) were each traced; the port band is structural avoidance (not a guard), reflection is a feature (not a defense), and the pre-existing guards remain documented trust-boundary defenses with written justification. Zero necessity findings.

## Warnings

### WR-01: Ephemeral-port pick-to-bind race hard-fails the log-pair probe (observed flake during review) — carried over, still valid

**File:** `cmd/sup-kv/main_test.go:638-645` (probe, fatal bind path unchanged), `cmd/sup-kv/main_test.go:125-148` (pickers); observed firing from `cmd/sup-kv/main_test.go:714`
**Issue:** Carried over from the 2026-11-16 full review (disposition `open`); re-verified today — `probeLoopbackBroadcast` still treats a bind failure on the just-picked port as `t.Fatalf`, and `freeUDPPort`/`freeTCPPort` still pick from the ephemeral pool, so a parallel sibling's `:0` bind can still win the port in the pick-to-bind window and fail the test spuriously (observed once during the prior review; the identical rerun passed). File outside this re-review's scope; unchanged since 2aaadb7. Note: phase 7 added `cmd/sup-kv/broadcast_probe_test.go`, whose probe routes through the hardened `scenarioUDPPort` (private window + registry + bounded retries) — the fix pattern now exists in the same package, making the remaining fix mechanical.
**Fix:** Give `probeLoopbackBroadcast` the `newExecPair`/`probeBroadcastForm` treatment: on a probe rx/tx bind error, re-pick a fresh port and retry (bounded attempts), fataling only after the bound is exhausted; longer term, route the smoke suite's UDP ports through the shared `scenarioUDPPort` picker.

### WR-02: Metrics discard-class labels re-stringify the canonical tokens — silent contract-drift hazard — carried over, still valid

**File:** `internal/metrics/metrics.go:283-287`; canonical constants at `internal/app/repl/core.go:39-45`
**Issue:** Carried over from the 2026-11-16 full review (disposition `open`); re-verified today — the five `supkv_datagrams_discarded_total{class=...}` label values are still bare string literals while `repl` still exports the same values as first-class constants, pinned by disjoint tests, so a token rename on either side still passes its own tests while silently breaking the documented logs↔metrics class correspondence. File outside this re-review's scope; unchanged since 2aaadb7.
**Fix:** Reference the constants as the label values:

```go
{Labels: []Label{{Name: labelClass, Value: repl.DiscardBadCRC}}, Value: float64(disc.BadCRC)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardUndecodable}}, Value: float64(disc.Undecodable)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardBadProtoVersion}}, Value: float64(disc.BadProtoVersion)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardOwnBroadcast}}, Value: float64(disc.OwnEcho)},
{Labels: []Label{{Name: labelClass, Value: repl.DiscardDuplicateNodeID}}, Value: float64(disc.DuplicateNodeID)},
```

### WR-03: NewMux comment states a false mechanism — pprof/expvar DO register on DefaultServeMux on any import — carried over, still valid

**File:** `internal/metrics/metrics.go:388-391`
**Issue:** Carried over from the 2026-11-16 full review (disposition `open`); re-verified today — the comment still claims "neither expvar nor net/http/pprof is ever blank-imported, so nothing registers on [http.DefaultServeMux]", which is wrong about the mechanism (both packages register from `init()` on any import); the true invariant is that this process never serves `DefaultServeMux`. File outside this re-review's scope; unchanged since 2aaadb7.
**Fix:** Reword to state the true invariant: pprof/expvar register on `http.DefaultServeMux` from their `init()` on any import; the exposure control is that the only HTTP server in the process is built over the fresh mux, and the whole surface rides the one opt-in listener behind D-02's optional mTLS.

## Info

### IN-01: SentByType counts datagrams as sent before unicast resolution and before the socket write — carried over, still valid

**File:** `internal/udprepl/repl.go:515-533` (countSent at 522; resolve at 527-531; write at 532); accessor docs at `internal/udprepl/repl.go:431-452`
**Issue:** Carried over from the 2026-11-16 full review (disposition `open`); re-verified today — `flush` still counts on encode success, before `ResolveUDPAddr` and before `WriteToUDP`, contradicting the `SentByType` accessor doc. Practically unobservable (handles originate from received datagrams), hence Info. File outside this re-review's scope; unchanged since 2aaadb7.
**Fix:** Move `countSent` after the successful write, or align the accessor/flush doc comments to "encode-successful outbound messages".

### IN-02: `release-snapshot` — the mandated local pre-tag gate — has an undeclared `syft` prerequisite its comment does not name

**File:** `Makefile:82-87` (target + comment); prerequisite evidence in `.goreleaser.yaml` (sboms block) and goreleaser v2.20.2 source
**Issue:** The target's comment bills `release-snapshot` as "prove the whole artifact pipeline locally … The mandatory pre-tag gate (D-09/Pitfall 8): run this and see it green before any v* tag is pushed (07-05)". Verified against the pinned goreleaser v2.20.2 (module cache, `internal/pipe/sbom/sbom.go:37-38`): the sboms pipe skips only on an explicit `--skip=sbom` or an empty SBOMs config — snapshot mode does NOT skip it — so `--snapshot` shells out to `syft` for every archive. `syft` is not a go.mod tool directive and is not installed on this machine (`command -v syft` fails); the 07-04 proof required an ad-hoc `GOBIN=/tmp go install … syft@v1.57.2`, and `.goreleaser.yaml`'s comment documents syft as "installed by the … anchore/sbom-action/download-syft@v0.24.3 workflow step" — a CI-only framing. The maintainer who follows the Makefile's own instruction at v1.1.0 time on a stock dev machine hits an sbom-pipe failure and must dig into the phase summary to learn the prerequisite. Behavior itself is correct and the pipeline was proven (4 archives + 4 SBOMs + checksums.txt); this is a completeness gap in the gate target's one comment.
**Fix:** One comment line on the target, e.g.:

```make
## release-snapshot: prove the whole artifact pipeline locally with NO tag
## and NO publishing — --snapshot builds all 4 targets plus checksums into
## dist/. Requires syft on PATH (the sboms pipe runs in snapshot mode too;
## ad-hoc: GOBIN=/tmp/syft go install github.com/anchore/syft/cmd/syft@v1.57.2).
```

### IN-03: Empty-disk recovery bound is enforced from `waitFor` start, not from node B's process start — the window can exceed the documented bound

**File:** `cmd/sup-kv/scenario_test.go:651-670` (bound derivation 651-653, `start` 658, `awaitServing` 660, `waitFor` 663-670)
**Issue:** The scenario's stated contract is "from B's process start to B serving all N keys … must complete within the documented bound" (`deadline = paced + margin`, currently 400ms + 3.6s). But `waitFor` converts its timeout to an absolute deadline from its own start — after `startExecNode` and `awaitServing` — so the enforced window is `awaitServing-latency + 4s` from `start`, not `4s` from `start`. The leak is bounded by `servingDeadline` (10s) in the pathological case and is typically ~the startup-gate window (~1-1.5s), which the margin separately already counts; a marginal violation (all keys served at `start+5s` against the 4s bound) would pass the test, with only the `t.Logf` line (`time.Since(start)`) recording the true elapsed. The test still catches gross regressions (a broken limiter blows the bound by seconds), and today's run recovered in 274ms — far inside even the strict reading. Pre-existing in this in-scope file (only `scenarioPortLo` changed since the full review); marginal, hence Info.
**Fix:** Anchor the enforcement to `start`: capture `deadlineAt := start.Add(deadline)` and poll against it (e.g. a `waitFor`-shaped helper taking an absolute deadline, or subtract the elapsed `awaitServing` time from the passed timeout), so the asserted window matches the documented bound the comment derives.

---

_Reviewed: 2026-11-18T11:13:49Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
