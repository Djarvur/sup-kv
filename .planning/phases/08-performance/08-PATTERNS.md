---
phase: 08-performance
mapped: 2026-11-23
head: bf6966f
---

# Phase 8: Performance - Pattern Map

**Files analyzed:** 31 (9 new, 21 modified, 1 reference-only)
**Analogs found:** 25 / 31 (6 with no close analog; see the last section)
**Search scope:** `internal/`, `cmd/`, `docs/`, `Makefile`, `.golangci.yml`, `.github/workflows/`
(all analogs below are git-tracked source; no gitignored mirror paths were used)

## File Classification

| File | Action | Role | Data flow | Closest analog (file:line) | Match |
|---|---|---|---|---|---|
| `internal/testutil/perf.go` | create | test utility / config | transform (consts) | `internal/testutil/bench.go:1-58` | role-match |
| `internal/testutil/perf_linux.go` | create | test utility | file-I/O (`/proc`) | `internal/metrics/rss_linux.go:1-39` | partial |
| `internal/testutil/perf_other.go` | create | test utility | transform | `internal/metrics/rss_other.go:1-20` | partial |
| `internal/testutil/race_on.go`, `race_off.go` | create | test utility | build-tag split | `internal/metrics/rss_linux.go` + `rss_other.go` | partial |
| `internal/testutil/ports.go` | create | test utility | socket bind | `internal/stand/stand.go:175-231` | role-match |
| `internal/stand/perf_test.go` | create | acceptance test | request-response / batch | `internal/stand/stand_red_test.go:381-588`; `stand_metrics_test.go:150-251` | role-match |
| `cmd/sup-kv/performance_test.go` | create | doc test | file-I/O | `cmd/sup-kv/operations_test.go:48-208` | exact |
| `docs/PERFORMANCE.md` | create | docs | prose ledger | `docs/OPERATIONS.md:1-215` | role-match |
| `internal/app/repl/core.go` | modify | model / state machine | event-driven | itself: `peerUnacked`, `retryHeap.index`, `recordQueued` | exact (self) |
| `internal/app/repl/bench_test.go` | modify | benchmark + gate | batch | itself `:25-309`; `internal/app/persist/bench_test.go:86-141` | exact (self) |
| `internal/app/repl/retransmit_test.go` | modify | test fixture | transform | `:66-104` `stateStore` wrapper | exact (self) |
| `internal/app/repl/sim_dump_test.go` | modify | simulator harness | event-driven | `internal/app/persist/engine_test.go:644-692`; `backoff_test.go:1-14` | exact |
| `internal/app/repl/config_internal_test.go` | modify | unit test | transform | itself `:57-81` | exact (self) |
| `internal/app/repl/events_test.go` | reference only | unit test | event-driven | `events_test.go:162-260` | exact |
| `internal/grpcapi/bench_test.go` | modify | benchmark + gate | request-response | itself `:36-244` | exact (self) |
| `internal/grpcapi/server.go` | modify | adapter / service | request-response | itself `:403-423` | exact (self) |
| `internal/udprepl/loop.go` | modify | adapter / event loop | event-driven | itself `:248-281` | exact (self) |
| `internal/udprepl/smoke_test.go` | modify | test utility use | socket bind | `:45-66` `freeUDPPort` | exact (self) |
| `internal/grpcapi/wait_integration_test.go` | modify | test utility use | socket bind | `:61-78` `integrationUDPPort` | exact (self) |
| `cmd/sup-kv/main_test.go` | modify | test harness | process lifecycle | itself `:55-67,127-150,314-327,1081-1099,1715-1750` | exact (self) |
| `cmd/sup-kv/broadcast_probe_test.go` | modify | test utility use | socket bind | duplicate registry of `stand.go:175-219` | role-match |
| `cmd/sup-kv/examples_test.go` | modify | doc test | batch | itself `:335-337` | exact (self) |
| `cmd/sup-kv/readme_test.go` | modify | doc test | batch | itself `:75-77` | exact (self) |
| `internal/stand/stand.go` | modify | harness | process lifecycle | itself `:409-439`, `:485-501`, `:570-597` | exact (self) |
| `internal/stand/stand_red_test.go` | modify | acceptance test | batch | itself `:452-588` | exact (self) |
| `internal/stand/persist_test.go` | modify (likely none) | acceptance test | process lifecycle | `:70-90` kill call site | exact |
| `Makefile` | modify | build config | — | itself `:1,25-41,43-78` | exact (self) |
| `docs/OPERATIONS.md` | modify | docs | prose | itself `:190-214` | exact (self) |
| `docs/SPEC-DEVIATIONS.md` | modify | docs | table row | itself `:210` (S-FAIL-15) | exact (self) |
| `.planning/PROJECT.md` | modify | plan docs | prose | itself `:43,80,82,97` ("~10 ms") | exact (self) |
| `.github/workflows/ci.yml` | no change | — | — | — | — (D-06 keeps CI threshold-free) |

---

## Pattern Assignments (what to copy)

### `internal/testutil/perf.go` — the one threshold place

**Analog:** `internal/testutil/bench.go:1-17` (package doc), `:19-25` (const block), and `internal/app/internal/testutil/bench.go:1-25` (re-export wrapper).

Copy:
- The package doc's explanation of *why this package may import `testing`* (`bench.go:1-11`): `no-test-libs-in-prod` exempts `**/test*/**` (`.golangci.yml:52-64`). Extend that doc; do not create a second test-helper home.
- Const block style with a short prose comment per group (`bench.go:19-25`). Thresholds must be `const`, never `var`, so `gochecknoglobals` has nothing to flag (research Code Example §4; the `_test.go` exclusion at `.golangci.yml:517-532` does not cover a non-test file).
- Naming: exported (`PerfDefaultP50`, …) because `internal/stand`, `internal/grpcapi` and `internal/app/repl` all consume them.

Pitfalls already solved by the analog: a single module-wide home exists precisely because Go's internal rule forbids `internal/udprepl`/`internal/grpcapi` from importing `internal/app/internal/testutil` (`bench.go:5-10`) — put the new helper in `internal/testutil`, not the app-internal one.

Add: `PerfEnabled()` (`os.Getenv("SUPKV_PERF") == "1"`) and `RaceEnabled()` as a build-tag pair. Gate idiom must mirror `standNightly` (`internal/stand/stand_test.go:49-51`) and the red tests' `if os.Getenv("SUPKV_RED_TESTS") != "1" { t.Skip(...) }` (`stand_red_test.go:277-279,382-384,538-540`): a named skip, never a build tag (research Pattern 4).

### `internal/testutil/perf_linux.go` / `perf_other.go` / `race_on.go` / `race_off.go`

**Analogs:** `internal/metrics/rss_linux.go:1-39` (`//go:build linux` + `/proc/self/statm` parse) and `internal/metrics/rss_other.go:1-20` (documented non-Linux fallback).

Copy the tag-pair discipline: Linux file carries the real `/proc` reads (`/proc/cpuinfo`, `/proc/meminfo`, `/proc/loadavg`, `/proc/self/mountinfo`), the other file returns a documented "unavailable" block. Race pair: `//go:build race` sets `RaceEnabled() bool { return true }`; `//go:build !race` returns false. No new dependency, no `runtime` probes in test bodies.

Pitfall: `HardwareBlock()` runs from tests in Linux packages only; do **not** let a missing `/proc` file fail a perf test — degrade to `unknown` and log, except the amplification test's tmpfs check, which must fail naming `SUPKV_PERF_DUMP_DIR` (research PERF-06).

### `internal/testutil/ports.go` — one allocator with package bands

**Analog:** `internal/stand/stand.go:175-231` (registry, claim-while-held, bounded retry) — the only correct picker in the repo; the three `:0` pickers are the bug (`cmd/sup-kv/main_test.go:127-150`, `internal/udprepl/smoke_test.go:45-66`, `internal/grpcapi/wait_integration_test.go:61-78`).

Copy verbatim semantics:
- `portRegistry{mu sync.Mutex; taken map[string]bool}` keyed by `network:port` (`stand.go:180-185`).
- Random candidate in band → bind probe → claim **while the bind is held** → close → `t.Cleanup` releases (`stand.go:192-219`).
- Bounded retry then `t.Fatalf` naming the band (`stand.go:194,217`).
- `rand.IntN` with the `//nolint:gosec // G404` precedent (`stand.go:195`).
- API must take `testing.TB` (benchmarks reuse it: `smoke_test.go:48-49` shows the reason), and cannot call `tb.Context()` — use `context.Background()` with the documented `//nolint:usetesting // suggestion does not compile on TB` precedent (`smoke_test.go:55`; `wait_integration_test.go:70`).
- Bands: stand `[28100,29100)`, `cmd/sup-kv` `[29100,30100)`, `udprepl` `[30100,31100)`, `grpcapi` `[31100,32100)`; reserve `[27900,28100)` for the fixed doc ports (`stand.go:76-91`). Assert `/proc/sys/net/ipv4/ip_local_port_range` low bound > window on Linux; skip with a comment elsewhere.
- TCP shares the registry (`network` in the key); the stand's `freeTCPPort` (`stand.go:221-231`) is the call to replace.

Pitfalls: the registry is per process — bands are the cross-process cure; keep package-local wrappers thin so call sites keep their names (research F-4).

### `internal/stand/perf_test.go` — the five perf rows

**Analogs:** `stand_red_test.go:381-450` (unacked-drain: env gate, precondition wait, derived bound), `:452-588` (load generator + burst), `stand_metrics_test.go:150-251` and helpers `:474-548` (`standMetricsReady/Scrape/Sample`, `standSetWith`).

Copy:
- Test layout: env gate first statement → `newStandPair(t)` → nodes with explicit flags → `awaitServing` → seed → wait → assert → `signalAndWaitExit` + `assertNoPanic` on both nodes (`stand_red_test.go:537-588`).
- Open-loop load: reuse `redRunBurst` (`:470-515`) rather than a second generator; extract the scenario body (from `p := newStandPair(t)` through assertions) into a shared helper so `TestRed_R02_burst` stays byte-for-byte the red contract and `TestPerfBurstDistinctKeys` is a thin caller (research PERF-02).
- Metrics: `redMetricsBody`/`redMetricValue` (`:194-232`) or the richer `standMetricsScrape`/`standMetricsSample` (`stand_metrics_test.go:510-547`); `redThrottled` (`:237-246`) for hot polls.
- Derived bounds only: every timeout is an expression over the flags the nodes run with, with an explicit margin (`stand_red_test.go:57-66,426-434`) — never a magic wall-clock constant.
- Seeding: `redSeedClients`/`redSeedKeys` (`:140-187`), 1 KiB `redSeedValue` (`:135-137`).
- Dump path: the pair currently uses `t.TempDir()` (`stand.go:570-578`); perf rows must route `p.dumpA`/`p.dumpB` through `testutil.PerfDumpDir(t)` before `startStandNode`. Do not bypass `newStandPair` (ports, build-once and probe all live there).
- First-seen ordering: start both nodes before preloading, and pin `--sync-interval` long, so the measured window excludes the startup listing (research PERF-01/03).

Pitfalls already solved: `waitFor` clamps to `t.Deadline()` minus the margin (`stand.go:316-335`); concurrent port claims; SIGKILL diagnostics (below); `t.Parallel` is deliberately absent in wall-clock stand scenarios (`stand_red_test.go:25-31`) — follow it, and add `//nolint:paralleltest // <reason>` when `paralleltest` fires (precedent `bench_test.go:279`).

### `cmd/sup-kv/performance_test.go` — the published-number drift guard

**Analog:** `cmd/sup-kv/operations_test.go` in full:
- Honest-scope header (`:1-12`): the test proves presence/drift, never truth.
- Doc path const relative to the package (`:21-23`).
- Anchor table `{literal, pins}` (`:48-119`) with one literal per mandated claim and a `pins` sentence explaining what breaks.
- Forbidden table `{phrase, why}` (`:127-134`) matched against a **whitespace-normalized** document (`:161-170`) so line wrapping cannot keep a stale sentence alive.
- First-section ordering rule (`:147-155`) if PERFORMANCE.md must lead with the model.
- Optional exporter cross-check (`:176-208`) if the doc names metric families.
- `t.Parallel()` at the top (`:141`).

Phase-8 addition: derive anchors from `internal/testutil` constants (`fmt.Sprintf("%v", testutil.PerfDefaultP99)` or `time.Duration.String()`) instead of hand-typed strings where the doc states a threshold; otherwise a threshold change leaves the doc green. Add the command that produced each number as an anchor too (the docs rule from 08-CONTEXT "Specific Ideas").

### `docs/PERFORMANCE.md`

**Analog:** `docs/OPERATIONS.md:1-6` (intro + links) and `:190-214` (the memory model: qualitative statement, then the honest residual).

Copy the register: claim → derivation from named knobs → what is *not* claimed. Every table row carries profile, keys, value size, rate, media, race state, and the command. Keep OPERATIONS' `## Memory model` as a short summary linking here (`docs/OPERATIONS.md:190-214`), and do not duplicate the model text.

### `internal/app/repl/core.go` — pending index + `PendingVisits`

See "Core index pattern" below; field/accessor idioms:
- Counter fields documented as driver-goroutine-only behind a copying accessor (`:894-901`, `:921-933`), accessor beside `PacedSends` (`:3311-3320`) / `Retransmissions` (`:3142`).
- `New` initializes every map (`:989-1014`) — a new index must be in that literal or it panics on first write.
- The deletion chokepoint is `removePending` (`:3404-3422`); every path (drain, supersession, stale record) already funnels through it, so one delete there is sufficient.

### `internal/app/repl/bench_test.go` — deterministic gates + paired benchmark

**Analog:** itself. Copy:
- Top-of-file doc comment stating the claim, the acceptance pair and "never a hand measurement" (`:3-11`).
- Named constants with basis (`:25-47`, `:210-217`) — `mnd` is happy and the failure message can name the bound.
- Fixture builds once outside the loop, learns the peer identity, drains every startup/listing emission and returns the stop instant (`:58-87`); parameterize it with a `repl.StateStore` so the counting double can wrap the real store.
- Shared body used by both the benchmark and the gate (`:115-131`, `:145-164`) — do not fork the measured path.
- Failure messages name both values and the ratio (`:196-201`, `:303-307`); `t.Logf` on success.
- Sibling gate stays sequential with `//nolint:paralleltest // Wall-clock ratio gate: …` (`:279`) — the reason is mandatory (`nolintlint.require-explanation`).

Changes: replace `testing.Benchmark` in `TestCore_keyCountCostFlat` with a fixed-K loop and a counting `StateStore` (expected: DataRequest `Version` 1 + `Lookup` 1; DataSync `Observe` 1 + `Version` 16); snapshot counters **after** `benchFixture`'s drain (the startup Hello legitimately calls `Versions`/`Generation`, `:78-85`). Replace `TestCore_unackedCostFlat` with `PendingVisits/op == 0` at both sizes. Fix the fixture version domain: `version.PackVersion(testNowMs+int64(i), 0, localID)` (`:65` currently packs `uint16(i)`, outside the 12-bit counter domain; `benchUnackedFixture:231-235` is the correct shape). Add `BenchmarkNotifyWriteCost/{live,absent}` and `TestPerfAbsentPeerWriteCost` (paired ratio, `make perf` only).

### `internal/app/repl/retransmit_test.go` — counting double

**Analog:** `:66-88` `stateStore` is already a delegating `repl.StateStore` (the seam is consumer-side, `core.go:402-445`); `:97-104` `newTestCore`.

Implement `countingStore` embedding `*stateStore` and incrementing plain `int` fields (single-goroutine fixture, no atomics — research Code Example §1). Keep it in `repl_test`; do not add exported surface to the core. Follow the helper conventions: `t.Helper()` at function top, no `t.Parallel` inside helpers, fail messages carry `%s` name plus got/want.

### `internal/app/repl/sim_dump_test.go` — synctest bubble

**Analog:** `internal/app/persist/engine_test.go:650-654` and `backoff_test.go:1-14` (import `testing/synctest`; `synctest.Test(t, func(t *testing.T){…})`; `synctest.Wait()` as the idle barrier; `synctest.Sleep` for virtual pacing).

Delete the wall-clock machinery at `sim_dump_test.go:248-312`: `settle`, `awaitActivity`, `simSettlePause`, `simQuietRounds`, `simSettleWallCap` (`:49-51`) and the `simActivity` counter (`:75-98`) are the flake. Drive with the existing `sched.RunUntil`; call `synctest.Test` **from each subtest body**, and drop `t.Parallel`/`t.Run` inside the bubble (forbidden by `synctest.Test`; research F-3). One bubble per `runDumpConfirm` — the engine goroutine must not outlive its bubble.

### `internal/app/repl/config_internal_test.go` — capacity teeth

**Analog:** itself `:57-81`. It is `package repl` (internal) because unexported `cfg`/`chunkSyncEntries` are asserted (`testpackage` skips `*_internal_test.go`, `.golangci.yml:491-492`). Keep the existing shape/order assertions; add `cap` assertions (3-entry/max-budget case `cap(chunks[0]) == len(entries)`; multi-chunk case: each `cap ≤ entries remaining`, adjacent chunks do not share a backing array). Failure messages already name the chunk shape (`:73-74,77-78`) — extend that style.

### `internal/app/repl/events_test.go` — reference only

**Analog for any new counter assertion.** `TestDriftEvent` (`:162-260`) shows the counter-vs-event split: assert the raw counter through the copying accessor (`core.MergeStats().DriftWarnings`, `:182-186`) and the rate-limited event separately. Place a `PendingVisits` assertion beside the counter tests, not in the scenario files. No edit is required by the research unless the planner chooses to group all counter gates here.

### `internal/grpcapi/bench_test.go` — profile table + replay allocation gate

**Analog:** itself. Copy:
- Header states conditions and "an unacked report FAILS the benchmark — never noise" (`:3-11`).
- Constants for the operator default and the burst pace with rationale (`:36-45`).
- Full real-stack fixture: bufconn client, two real loopback UDP adapters, production wait seam, `b.Cleanup` ordering so repeated runs never leak ports (`:68-170`).
- Metric reporting at the end with `testutil.ReportLatencyPercentiles` (`:241-243`).
- Fast-profile knobs `Debounce: time.Millisecond`, `MaxWait: 50ms` (`:84-88`) — the exact conditions the "~10 ms" claim must cite.

Changes: turn the body into a profile table (`default`, `fast`) sharing one `benchConfirmationLatency(b, cfg)` body; the default arm uses a real-filesystem dump store (not `fakeDumpStore`, `internal/grpcapi/server_test.go:80-117`) where dump time matters. Add `TestPerfFastProfileLatency` wrapping `testing.Benchmark` and asserting `Extra["p50-ms"]`/`Extra["p99-ms"]` (gate only under `SUPKV_PERF=1`), and `BenchmarkReplayNotify` + `TestReplayNotify_noDataCopy` (`AllocedBytesPerOp() < 1024` for a 60 KiB record). Pitfall: the gate must not run `t.Parallel` and must skip under `-race` (detector measures the detector).

### `internal/grpcapi/server.go` — drop the replay clone

**Analog:** itself `:403-423`. Change `notifyReplayedWrite` to pass `storage.Record{Version: rec.Version}` on the `ReplayNotify` branch (no `bytes.Clone`, no `RecordChecksum`; both dead — `core.go:2350-2360` reads only `rec.Version`, and the drain re-reads via `Lookup`, `core.go:3032-3044`). Update the `LocalReplayNotify` contract doc (`:53-60`) and the method comment (`:403-411`) to say the record carries only the version. Keep the `ReplayNotify == nil` fallback path untouched (it rebuilds a full record for the live seam).

### `internal/udprepl/loop.go` — drop the second clone

**Analog:** itself `:248-267`. `NotifyReplayedWrite` should enqueue `localWrite{key: key, rec: storage.Record{Version: rec.Version}, replay: true}` — no `cloneLocalRecord`. Keep `cloneLocalRecord` for the live `NotifyLocalWrite` path (the author may reuse its buffer). Update the seam doc (`:248-253`). Do not touch `enqueueLocalWrite`'s closed-loop select (`:269-296`); that behavior is settled.

### Port call-site migrations

| File | Analog to replace | Note |
|---|---|---|
| `internal/udprepl/smoke_test.go:45-66` | itself | keep the `testing.TB` signature; delegate to `testutil.UDPPort(tb, testutil.UDPreplBand)` |
| `internal/grpcapi/wait_integration_test.go:61-78` | itself | keep the TB signature; grpcapi band |
| `cmd/sup-kv/main_test.go:127-150` | itself | UDP + TCP wrappers; keep names so ~40 call sites stay untouched |
| `internal/stand/stand.go:175-231` | itself | replace registry internals with the shared allocator, keep `scenarioUDPPort` signature; `newStandPair` probe-retry (`:548-568`) stays |
| `cmd/sup-kv/broadcast_probe_test.go:50-100` | duplicate of `stand.go:175-219` | delete the private registry, use the cmd band |

### `cmd/sup-kv/main_test.go` — readiness budgets and early-exit diagnosis

**Analog:** itself. Keep the constants-with-basis block (`:51-67`) and add distinct budgets rather than raising the shared one: `processServingDeadline = 30s` for a freshly spawned child's first wait (used at `:1099`, `:429`, `:444`, …), leaving `servingDeadline = 10s` for polls of a running node; `logEventDeadline = 30s` for the duplicate-node log wait (`:1842-1852` context). Make `awaitServing` (`:314-327`) and `startNode`/`launchNode` (`:213-299`) check `node.exited` and fail immediately with captured stderr/exit status, so a crashed child never looks like a slow one. The existing `waitFor` already clamps to `t.Deadline()` (`:152-175`) — do not reimplement.

### `cmd/sup-kv/examples_test.go:335-337`, `readme_test.go:75-77`

Drop `t.Parallel()`; keep the block budget and process-group reap. Add the `//nolint:paralleltest` justification only if the linter still asks (the file no longer satisfies the linter's parallel requirement). The reason must be human-readable (`nolintlint`).

### `internal/stand/stand.go` — `killAndReap` diagnosis

**Analog:** itself `:409-439` (`reap`/`killAndReap`) plus `:466-483` `assertExitCode` for the message shape. On `errors.Is(err, os.ErrProcessDone)`: wait for `node.exited`, read the buffered `node.exit` status, and fail naming pid, exit code/Wait status, stderr tail and stdout. A genuine kill error stays fatal with the same captured output. Optional: store the Wait result in a mutex-guarded field (the `exit` channel is 1-buffered, `:397-403`) so diagnostics can read it without consuming the channel. Call sites `internal/stand/persist_test.go:76-86,294-300` assert behavior, not message text, so no test edit should be needed.

### `internal/stand/stand_red_test.go` — extract, do not duplicate

Move the burst scenario body (`:541-588`) into a helper returning the achieved rate/elapsed, keep `TestRed_R02_burst`'s assertions and expectation-table contract identical (`:517-540`), and let `TestPerfBurstDistinctKeys` call the helper with `SUPKV_PERF=1`. Constants (`:91-125`) stay in one block.

### `Makefile`

See "Makefile target pattern" below.

### Docs rows

- `docs/OPERATIONS.md:190-214`: keep the short model, replace the duplicated numbers with a link to `docs/PERFORMANCE.md`; keep the residual-pause honesty paragraph.
- `docs/SPEC-DEVIATIONS.md:210` (S-FAIL-15): update the description to drop "holds only on hot keys today", cite `TestPerfBurstDistinctKeys`/`TestRed_R02_burst` and the documented bounds, keep the verification reference format (`fixed-in-v1.1(PERF-02)` is already there).
- `.planning/PROJECT.md:43,80,82,97`: tie "~10 ms under 1000 writes/s" to `BenchmarkConfirmationLatency`'s conditions (fast profile 1 ms/50 ms, in-memory dump, two loopback adapters, race off) — never standalone.

---

## Makefile target pattern

Established shape, from `bench` (`Makefile:25-33`), `bench-smoke` (`:35-41`), `race-soak` (`:43-67`), `red-run` (`:69-78`):

1. Target name in the `.PHONY` list on line 1.
2. A `## name: …` prose header directly above the target; continuation lines also start `##`. The header states: what runs, why it is manual/CI, what it does **not** do (no thresholds / no profiles / no `-race`), and who invokes it.
3. Recipe: one command per line, tab-indented, env prefix inline (`SUPKV_RED_TESTS` lives in the runner script for `red-run`; put `SUPKV_PERF=1` inline like the research proposal).
4. One definition only where CI invokes it (`bench-smoke:35-39` states this explicitly).

For `perf`:
- Add `perf` to `.PHONY`; header must state Linux-only, no `-race`, `-p 1` serialization, thresholds in `internal/testutil/perf.go`, hardware block logged, and that CI keeps `bench-smoke` (D-13/D-14).
- `mkdir -p .tmp/perf` is safe: `.tmp/` is gitignored (`.gitignore`), and `SUPKV_PERF_DUMP_DIR=$(CURDIR)/.tmp/perf` keeps the dump off `/tmp` (often tmpfs).
- Update `bench`/`bench-smoke` package lists to add `./internal/app/repl/` and extend their prose, so the core benchmarks stop being orphaned; `scheduled.yml:66-78` invokes `make bench-smoke`, so it stays target-driven with no workflow edit.
- Do **not** add `perf` to CI (`.github/workflows/ci.yml:84-95` stays `lint` + `go test -race ./...`).

## Doc-anchor pattern

From `cmd/sup-kv/operations_test.go`:
- Honest-scope comment first (`:1-12`): a presence/drift guard, not a truth claim.
- `const …DocPath = "../../docs/…"` (`:21-23`).
- `[]struct{ literal, pins string }` (`:48-119`): literal from the authored doc, `pins` says what breaks when it disappears; one per mandated statement, including headings (`## …`).
- `[]struct{ phrase, why string }` (`:127-134`) for removed claims; compare against `strings.Join(strings.Fields(doc), " ")` on both sides (`:161-170`) — this is the whitespace-normalized forbidden-phrase idiom.
- Optional first-section ordering check (`:147-155`).
- Optional source cross-check (`:176-208`): regex the doc's family names, assert each exists in the declaring source.
- `TestPerformanceDoc` should additionally compute threshold literals from `internal/testutil` consts where possible; a hand-typed threshold anchor drifts silently.

## Deterministic-counter-test pattern

From `TestMergeStats_counts` (`internal/app/repl/merge_stats_test.go:20-60`), `TestStand_metricsByEffect` (`internal/stand/stand_metrics_test.go:150-251`) and the ratio gates (`bench_test.go:196-201,303-307`):

- One fresh fixture per row so no counter leaks between rows (`merge_stats_test.go:38-40`).
- Assertion message names the claim and the counter: `"after the displacing apply MergeStats() = %+v, want Conflicts 1, …"` (`:57-58`).
- Stand exact-value assertions dump the whole scrape on failure (`stand_metrics_test.go:235-246`); baseline-then-climb when the quantity is a monotone total (`:223-228`).
- For the new gates: snapshot counters **after** fixture construction, then assert `(after-before)/K` exactly; both sizes are measured, both values are logged, and the failure names both plus the bound. For `PendingVisits`, the assertions are exact zeros (`idle Advance`, fresh-key write) — a fresh-key write may not inspect any index entry; a repeated-key write may inspect ≤2 (research PERF-05).
- No `testing.Benchmark` inside a `make test` gate; `testing.B` stays in `make bench`/`make perf` (research Pattern 1). Ratio-based paired benchmark is a `TestPerf*` row under `SUPKV_PERF=1`, sequential, with `//nolint:paralleltest` + reason.

## Core index pattern

**Recommendation:** add `pendingByKey map[string]pendingKey` beside `c.pending` (`core.go:815-823`).

Why a plain map, not `map[string]map[pendingKey]struct{}`:
- The one-pair-per-key invariant is already documented and enforced: `trackedVersion` states it (`core.go:3424-3436`), the guard rejects not-greater versions (`:2371-2387`), and `removeSupersededLocal` removes smaller pairs before insert (`:3444-3451`). A single-slot map encodes exactly that invariant; a set map only pays extra allocation to be resilient to a state the guard forbids.
- Both helper rewrites become direct reads: `trackedVersion` → `pk, ok := c.pendingByKey[key]`; `removeSupersededLocal` → inspect the single entry and compare; `applyRemote`'s displacement loop (`:1659-1670`) → inspect only `msg.Key`'s entry.

Closest existing side-table idioms to imitate (all in `core.go`):
- `retryHeap.index map[pendingKey]int` (`:745-753`): declared beside its owner, doc comment states the maintenance contract, updated in `Push`/`Pop`/`Swap` (`:767-791`).
- `peerUnacked` + `ledgerAdd`/`ledgerRemove` (`:815-822`, `:3453-3470`): per-index insertion helpers on the one insertion path, deletion via `removePending`.
- `recordQueued` dedupe set (`:868-869`) maintained in `enqueueRecordSend`/`drainRecordSends` (`:2987-3009`).

Maintenance rules:
- Insert in `trackLocalWrite` at the `c.pending[pk] = entry` site (`:2416`); delete in `removePending` (`:3409-3422`) — the single deletion point for drain, supersession and stale-record paths. Guard the delete with `if cur, ok := c.pendingByKey[pk.key]; ok && cur == pk` so a stale call can never evict a newer pair.
- Initialize in `New` (`:989-1014`), beside `pending` at `:997`.
- `evictPeers` (`:2912-2926`) does not remove pending entries (suspension), so it needs no index maintenance.
- `handleHello`'s discovery flush (`:1503-1516`) stays O(n) — it must emit every pending pair once to a new peer. It uses `sortedPendingKeys` (`:3368-3375`); keep the deterministic order.

`PendingVisits`:
- Plain `uint64` field beside `mergeStats`/`retransmitted`/`notificationsRegressed` (`:921-933`), driver-goroutine-only, no atomics; accessor beside `PacedSends` (`:3311-3320`).
- Increment once per index entry/pending entry inspected (not per map probe): `trackedVersion`, `removeSupersededLocal`, `applyRemote`'s displacement, `scanRetransmit`'s due loop (`:2970`), and the discovery flush. With the index, the hot paths inspect 0 entries for a fresh key, and the discovery flush reports its honest n.
- The accessor doc should say what the counter is *for* (the deterministic flatness seam) and that it is not exported as a metric.

## No-analog list

| File / capability | Role | Nearest-by-shape choice | Why no real analog |
|---|---|---|---|
| `HardwareBlock()` (`/proc` facts + build info) | test utility | `internal/metrics/rss_linux.go:1-39` | Repo parses only `/proc/self/statm`; no CPU/RAM/load/mountinfo reader exists |
| `PerfDumpDir` filesystem-type guard | test utility | `rss_linux.go`/`rss_other.go` tag pair | No mountinfo/`statfs` use anywhere |
| `TestPerfAmplification` (`/proc/<pid>/io` `write_bytes`) | acceptance test | `stand_metrics_test.go:150-251` + `scheduler.go:201-249` | No process-IO accounting exists; the metric sources are HTTP `/metrics` only |
| `TestPerfMemoryPoints` (RSS/VmHWM sampling) | acceptance test | `stand_metrics_test.go` scrape + `persist/bench_test.go:77-84` | No stand-level RSS sampling; only the exporter gauge and benchmark heap read |
| `BenchmarkReplayNotify` / `TestReplayNotify_noDataCopy` | benchmark + gate | `persist/bench_test.go:86-141` (`peak-live-B`) | No allocation-ceiling gate exists; `AllocsPerRun` has no precedent and its GOMAXPROCS side effect is a known hazard |
| `TestPerfAbsentPeerWriteCost` (paired live/absent ratio) | acceptance test | `benchUnackedFixture` (`bench_test.go:223-238`) + ratio gates | No paired-arm benchmark exists; nearest shape is the two-size ratio gate |
| `docs/PERFORMANCE.md` | docs | `docs/OPERATIONS.md:1-215` | First measurement-ledger doc in the repo; only the operations runbook has the model+table shape |

**No change expected (do not create work):** `.github/workflows/ci.yml` and `scheduled.yml` (D-06), `internal/config/config.go` (knobs already exist, `:107-108`), `internal/metrics/metrics.go` (the `supkv_dumps_*` counters and current-RSS gauge already exist at `:467-474,539-540`), `docs/adr/0003-dump-policy-unchanged.md` (decision stands), `internal/app/repl/events.go` and `sentbytype.go`.

## Weak-analog summary for the planner

- `internal/testutil/perf.go`, `perf_linux.go`, `race_*`, and especially `ports.go` are **new helper surfaces** with only shape analogs; plan an explicit unit test for the allocator band logic and for `PerfEnabled`/`RaceEnabled` before the perf rows depend on them.
- `TestPerfAmplification` and `TestPerfMemoryPoints` are the only rows with no measurement precedent; they must reuse `redMetricsBody`/`standMetricsScrape` for metrics and `benchSeedStore`-style deterministic preload, and must log the raw inputs so a failure is diagnosable without rerunning.
- `docs/PERFORMANCE.md` + `TestPerformanceDoc` form a new pair: copy the operations anchor-test structure exactly, but derive threshold anchors from the Go constants or the guard will not catch threshold drift.
- All source analogs are tracked; no phase artifact should cite `.gsd/` mirrors or untracked research paths.
