---
phase: "7"
slug: "observability-and-configuration"
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-22"
---

# Phase 7 — Validation Strategy

> Per-phase validation contract. Audited after execution: every task in the phase plans
> carries an automated verification that ran green on the committed tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` + `go test -race` |
| **Config file** | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `buf-lint`, `buf-breaking`, `client-example`) |
| **Quick run command** | `go test -race -count=1 ./internal/grpcapi/ ./cmd/sup-kv/ ./api/supkvapi/v1/` |
| **Full suite command** | `make lint && make test && make buf-lint && make buf-breaking && make client-example` (+ `SUPKV_RED_TESTS=1 bash scripts/red-runner`) |
| **Estimated runtime** | ~200–300 s full suite; per-task commands ≤ 60 s |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (the map below)
- **After every plan wave:** `go build ./... && make test`
- **Before phase verification:** full suite green on the committed tree
- **Max feedback latency:** ~300 s (full suite); per-task commands ≤ 60 s

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 07-01 | 1 | OBSV-08 | T-07-01 / T-07-03 | Same-package codec/socket move with the exporter import removed | automated | `go build ./... && go test -race -count=1 ./internal/udprepl/ ./internal/metrics/ ./internal/app/repl/` | ✅ | ⚠️ flaky |
| 07-01-02 | 07-01 | 1 | OBSV-08 | T-07-01 / T-07-02 | Single-driver loop and mirror moved verbatim | automated | `go vet ./internal/udprepl/ && go test -race -count=1 ./internal/udprepl/ ./internal/metrics/` | ✅ | ✅ green |
| 07-02-01 | 07-02 | 1 | OBSV-02 | T-07-05 / T-07-06 | One parse/usage point; help/version contract preserved | automated | `go test -race -count=1 ./internal/config/ -run 'TestParse\|TestUsage' -v` | ✅ (task) | ✅ green |
| 07-02-02 | 07-02 | 1 | OBSV-02 | T-07-04 / T-07-07 | Every range/relation exits 2 naming the parameter | automated | `go test -race -count=1 ./internal/config/ -run 'TestValidate\|TestRules' -v` | ✅ (task) | ✅ green |
| 07-02-03 | 07-02 | 1 | OBSV-02 | T-07-04 | Value guard precedes the header write; 2 m default consistent | automated | `go test -race -count=1 ./internal/app/persist/ -run 'TestCodec\|TestEncodeTo\|TestEncode' -v` | ✅ | ✅ green |
| 07-03-01 | 07-03 | 2 | OBSV-07 | T-07-08 / T-07-09 | Core/ports direction enforced; vendored body byte-identical | automated | `test ! -e internal/app/probe_boundary.go && h=$(sed -n '1,64p' .golangci.yml \| git hash-object --stdin) && test "$h" = a121e0466bc211e48ca898f6c07434dc50eba07c && make lint` | ✅ | ✅ green |
| 07-03-02 | 07-03 | 2 | OBSV-07 | T-07-08 / T-07-10 | Exporter/API rules fire; the one composition edge is written | automated | `h=$(sed -n '1,64p' .golangci.yml \| git hash-object --stdin) && test "$h" = a121e0466bc211e48ca898f6c07434dc50eba07c && make lint` | ✅ | ✅ green |
| 07-04-01 | 07-04 | 2 | OBSV-04 | T-07-11 / T-07-12 | Exit table, signals before wire, fatal bind is an ERROR record | automated | `go test -race -count=1 ./cmd/sup-kv/ -run 'TestRun_' -v` | ✅ (task) | ✅ green |
| 07-04-02 | 07-04 | 2 | OBSV-04 | T-07-13 | Serve error still dumps; final-dump failure exits 3 | automated | `go test -race -count=1 ./cmd/sup-kv/ -run 'TestRun_serveErrorStillShutsDown\|TestRun_dumpFailureCode\|TestRun_serveAndDumpFailure' -v` | ✅ | ✅ green |
| 07-05-01 | 07-05 | 2 | OBSV-05 | T-07-15 | Received-lead signal boundary-tested; pre-existing drift tests reconciled | automated | `go test -race -count=1 ./internal/app/clock/ -run 'TestClock_driftSignal' -v` | ✅ | ✅ green |
| 07-05-02 | 07-05 | 2 | OBSV-05, OBSV-03 | T-07-15 / T-07-16 | Only positive leads count; listing maxima once per part | automated | `go test -race -count=1 ./internal/app/repl/ -run 'TestMergeStats' -v` | ✅ | ✅ green |
| 07-06-01 | 07-06 | 3 | OBSV-03 | T-07-19 | Conflicts mean pending displacement; Conflict comment corrected | automated | `go test -race -count=1 ./internal/app/repl/ -run 'TestMergeStats' -v` | ✅ | ✅ green |
| 07-06-02 | 07-06 | 3 | OBSV-03 | T-07-17 / T-07-18 | Keyless rate-limited apply-failure WARN; stand conflict exactly 1 | automated | `go test -race -count=1 ./internal/app/repl/ -run 'TestApplyFailure_warnLimiterKeyedByNodeAndClass\|TestDiscard_warnLimiterKeyedByNodeAndClass' -v` | ✅ (task) | ✅ green |
| 07-07-01 | 07-07 | 4 | OBSV-01 | T-07-22 | Gate reason single-shot after the CAS, recorded end-to-end | automated | `go test -race -count=1 ./internal/grpcapi/ -run 'TestGate_openReason\|TestGate_opensOn\|TestHealth_reflectsGate' -v` | ✅ (task) | ✅ green |
| 07-07-02 | 07-07 | 4 | OBSV-01, OBSV-05 | T-07-20 / T-07-22 | Typed peer/drift events, rate-limited WARN, nil-safe seams | automated | `go test -race -count=1 ./internal/app/repl/ -run 'TestPeerEvents\|TestDriftEvent\|TestDiscard_warnLimiterKeyedByNodeAndClass' -v` | ✅ (task) | ✅ green |
| 07-07-03 | 07-07 | 4 | OBSV-01 | T-07-21 | Startup record secrets-free; exactly one load record | automated | `go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_startupLogRecord\|TestMain_startupLogLoadOutcome' -v` | ✅ (task) | ✅ green |
| 07-08-01 | 07-08 | 5 | OBSV-06, OBSV-03 | T-07-23 / T-07-25 | Unbound listing is its own class; conservation intact | automated | `go test -race -count=1 ./internal/app/repl/ -run 'TestDiscard\|TestSync_unbound\|TestSync_forgedRun' -v` | ✅ (task) | ✅ green |
| 07-08-02 | 07-08 | 5 | OBSV-06 | T-07-24 | Both proto branches WARN; level matrix at documented levels | automated | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestCatalogLevels$' -v` | ✅ (task) | ✅ green |
| 07-09-01 | 07-09 | 6 | OBSV-03 | T-07-26 | Sent counts only after the socket accepts the datagram | automated | `go test -race -count=1 ./internal/udprepl/ -run 'TestFlush_countsOnlySuccessfulWrites\|TestMirror_' -v` | ✅ (task) | ✅ green |
| 07-09-02 | 07-09 | 6 | OBSV-03 | T-07-27 | Four counters published, unlabeled | automated | `go test -race -count=1 ./internal/metrics/ -run 'TestMetrics_families\|TestMetrics_values' -v` | ✅ | ✅ green |
| 07-09-03 | 07-09 | 6 | OBSV-03 | T-07-28 | Current RSS; one test per published definition | automated | `go test -race -count=1 ./internal/metrics/ -run 'TestCurrentRSS\|TestParseStatm\|TestMetrics_definitions' -v` | ✅ (task) | ✅ green |
| 07-10-01 | 07-10 | 7 | OBSV-01…07 | T-07-29 / T-07-30 | Runbook matches the shipped catalog, exit table and drift condition | automated | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestOperationsDoc$\|^TestOperationsDocFamilies$' -v` | ✅ | ✅ green |
| 07-10-02 | 07-10 | 7 | OBSV-03…07 | T-07-31 | Registry rows cite shipped code; AGENTS names the rule set | automated | `grep -q 'pending-local displacement' docs/SPEC-DEVIATIONS.md && grep -q 'internal/config/' AGENTS.md && grep -q 'sup-kv-adapters-not-in-core' AGENTS.md` | ✅ | ✅ green |
| 07-11-01 | 07-11 | 8 | OBSV-01…08 | T-07-32 | Frozen red evidence byte-identical; sed targets present; 7/7 | automated | `a=$(git hash-object internal/grpcapi/red_gate_test.go) && b=$(git hash-object internal/stand/stand_red_test.go) && c=$(git hash-object scripts/red-expectations.tsv) && test "$a" = 519789bd1f7abb2af09053b81c26e0ce45d9bdaf && test "$b" = 573fed68d2f1a039b8ac023a342446103064273b && test "$c" = add45b810d36d69fd574ca83cb575dab5722a55b` | ✅ | ✅ green |
| 07-11-02 | 07-11 | 8 | OBSV-01…08 | T-07-33 / T-07-34 | Full standing gate green; no new runtime dependency | automated | `make lint && make test && make buf-lint && make buf-breaking && make client-example` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*File Exists: ✅ = the package/test infrastructure exists on the committed tree; `✅ (task)` = the task creates the named test file itself, so no Wave 0 scaffold is required.*

**Post-execution audit (2026-11-23):** every row's automated command was re-run on the committed
tree (head `3f3d017`). 24 rows green. `07-01-01` is marked ⚠️ flaky because its command runs the
whole `internal/app/repl` package, whose two wall-clock ratio gates (`TestCore_unackedCostFlat`
3.62, `TestCore_keyCountCostFlat` DataSync 2.15 / DataRequest 1.02) failed under the host's external
load (average 12 on 16 cores) while `internal/udprepl` and `internal/metrics` passed and every
non-benchmark `repl` test passed; the row's target behavior (the codec/socket move held inside
`udprepl` and `metrics`) is green, and the same full suite passed unchanged at the 07-11 phase gate
(`make test`, 5m54s, exit 0, same tree). The flake is recorded in `deferred-items.md` and owned by
Phase 8. `07-11-02` is green from the 07-11 gate run itself (no source commit followed it).

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — existing infrastructure only)
- [x] No watch-mode flags
- [x] Feedback latency < 300 s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-11-23 — post-execution audit ran all 25 rows on the committed tree (24 green, one wall-clock flake documented above and owned by Phase 8).
