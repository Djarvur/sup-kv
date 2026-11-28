---
phase: 05-persistence
plan: 04
subsystem: persistence
tags: [streaming-dump, save-stream, encode-to, load-outcome, load-result, fill-hygiene, peak-live, benchmark, bufio]
requires:
  - phase: 05-persistence
    provides: the single-source value limit and the four-class line decoder from 05-01/05-02
  - phase: 03-transport
    provides: the canonical five-step atomic save seam (TEST-03) and the FileStore fallback chain
  - phase: 04-protocol
    provides: the engine's freeze on the exported API and the confirmation contract PERS-04
affects: [persistence, dump-path, load-outcome, memory-model, benchmark, p07, p12, p15]
actuals:
  tokens: 12851
  tasks: 3
  commits: 6
plan_head_before: 98d17932caea2615ad95df75cbe2d67782fcecc5
plan_head_after: 0e7e7225f8d8e00c30cf0288308fbd27266284fe
tech-stack:
  added: []
  patterns:
    - "Streaming save: the port hands the adapter a fill callback, the adapter hands the fill its buffered temp writer, and the codec writes one format line per Write"
    - "One records copy: runDump takes exactly one Records() snapshot and never materializes the encoded dump as a second buffer"
    - "Fill-error hygiene: any fill or flush error closes and removes the temp before the error surfaces"
    - "Load report through the port: LoadOutcome + accepted Generation + per-candidate GenerationFailure, with absent candidates as non-failures"
    - "Measured memory model: B/op for allocation churn plus a sampled peak-live-B taken at the fill's end"
key-files:
  created:
    - internal/app/persist/encode_stream_internal_test.go
  modified:
    - internal/port/dump.go
    - internal/app/persist/codec.go
    - internal/app/persist/engine.go
    - internal/app/persist/bench_test.go
    - internal/app/persist/engine_test.go
    - internal/app/persist/engine_blocking_test.go
    - internal/persistfile/dumpfile.go
    - internal/persistfile/fsops.go
    - internal/persistfile/snapshot.go
    - internal/persistfile/dumpfile_test.go
    - internal/persistfile/fsops_test.go
    - internal/grpcapi/server_test.go
key-decisions:
  - "EncodeTo is the single encoder; Encode is its bytes.Buffer wrapper, so the streaming and buffered paths cannot drift"
  - "SaveStream's fill runs exactly once on the caller's goroutine and must not retain the writer; its error aborts the generation and the adapter has already removed the temp"
  - "The adapter buffers the fill through a named 64 KiB bufio writer, so the codec's per-line Writes coalesce into few syscalls without holding the dump"
  - "Save leaves port.DumpStore; FileStore.Save stays as a byte-fixture convenience over SaveStream"
  - "Load classifies only in this plan: absent candidates are not failures, present-but-unreadable candidates are recorded with path and cause, and preservation/sweep stay for 05-05"
  - "The benchmark samples runtime.MemStats HeapAlloc at the end of the fill — the peak moment when the one Records() copy and the adapter's bufio are both live — rather than after release"
patterns-established:
  - "Test-double ripple in one commit: every DumpStore implementation moves with the interface so each task commit compiles"
  - "RED evidence for compile-coupled interface tasks: the new symbol lands as a declaration-only stub in the test commit, so the target test fails on its assertion rather than a build error"
  - "Failure-matrix row per fault source: the five step rows keep their exact order and the fill-error row reuses the same no-temp/no-partial-main assertions"
requirements-completed: [PERSIST-06, PERSIST-11]
coverage:
  - id: D1
    description: "EncodeTo reproduces Encode byte-for-byte and issues one Write per format line, none above maxDumpLineBytes()"
    requirement: "PERSIST-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEncodeTo_'"
        status: pass
    human_judgment: false
  - id: D2
    description: "A dump cycle takes one Records() copy and streams it through SaveStream/EncodeTo; a standing dump still holds Set"
    requirement: "PERSIST-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_blockingSaveHoldsSet$'"
        status: pass
      - kind: build
        ref: "go build ./... && go vet ./..."
        status: pass
    human_judgment: false
  - id: D3
    description: "The adapter's fill-based write keeps the five-step order and removes the temp on a fill error; Save remains a byte-fixture wrapper"
    requirement: "PERSIST-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_(saveOperationOrder|failureAtEachStep|injectedWriteStillRunsFill)$|^TestFileStore_saveFailureHygiene$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/"
        status: pass
    human_judgment: false
  - id: D4
    description: "The port's v2 load contract classifies LoadPrompt/LoadMain/LoadBackup/LoadUnusable with accepted generation and per-candidate failures, and all four implementations satisfy it"
    requirement: "PERSIST-11"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFileStore_(fallbackChain|atomicRotation)$|^TestFsops_emptyDumpRoundTrip$'"
        status: pass
      - kind: build
        ref: "go build ./... && go vet ./..."
        status: pass
    human_judgment: false
  - id: D5
    description: "The memory model is measured: the benchmark reports B/op and a sampled peak-live-B, with the one Records() copy stated as the residual"
    requirement: "PERSIST-06"
    verification:
      - kind: benchmark
        ref: "go test -run '^$' -bench 'BenchmarkDump' -benchmem ./internal/app/persist/ — peak-live-B 17496904, B/op 6422059, bytes/op 5147571"
        status: pass
    human_judgment: false
duration: 13min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 04: Streamed Dump Cycle and the Port Load Outcome Summary

**Dump cycles now stream one `Records()` copy through `SaveStream`/`EncodeTo` (B/op down from ~18.9 MB to 6.4 MB), and the port carries the four-outcome `LoadResult` in the same interface ripple.**

## Performance

- **Duration:** 13min
- **Started:** 2026-11-22T13:58:37Z
- **Completed:** 2026-11-22T14:11:15Z
- **Tasks:** 3
- **Files modified:** 13

## Accomplishments

- `EncodeTo(w, records, limit)` writes the JSONL dump one format line per `Write`, and `Encode` is now its `bytes.Buffer` wrapper — the two encoders cannot drift.
- `runDump` takes exactly one `Records()` copy and hands it to `SaveStream`; the adapter supplies a 64 KiB buffered temp writer and the five TEST-03 steps are unchanged (write → fsync → rotate → rename → dir-fsync).
- A fill or flush error closes and removes the temp before surfacing; the failure matrix gained the fill-error row with the same no-temp/no-partial-main assertions, and the injected-write row still exercises the caller's fill.
- `port.DumpStore` v2: `Load(now, parse) LoadResult` with `LoadOutcome` (`Prompt`/`Main`/`Backup`/`Unusable`), accepted `Generation`, and `GenerationFailure` entries carrying path and cause; `FileStore` and all three test doubles moved in the same task.
- The benchmark now drives the streamed path and reports `peak-live-B` alongside `B/op`, `bytes/op` and `dump-bytes`.

## Memory model (measured)

`go test -run '^$' -bench 'BenchmarkDump' -benchmem ./internal/app/persist/` on the 10 000-key fixture (5 147 571-byte dump):

| Metric | Before (plan baseline, docs evidence) | After (this plan) |
|--------|---------------------------------------|-------------------|
| `B/op` (allocation churn) | 18 937 008–18 938 008 | 6 422 059 |
| `allocs/op` | 40 104 | 40 096 |
| `bytes/op` / `dump-bytes` | 5 147 550 | 5 147 571 |
| `peak-live-B` | not reported | 17 496 904 |
| `ns/op` | ~17.6–20.0 ms | ~54.6 ms |

`peak-live-B` is `runtime.MemStats.HeapAlloc` sampled at the end of the fill — the cycle's peak moment, while the one unavoidable `Records()` copy and the adapter's 64 KiB `bufio` are both live. It is an absolute live-heap reading that includes the seeded store itself; the accepted residual is the one `Records()` copy plus the writer pause, and a second encoded state would show up as a roughly doubled delta. `B/op` is the allocation churn and dropped ~66 % because the encoded dump no longer exists as a second buffer; `ns/op` is the real durable cycle (temp, fsync, rotate, rename, dir-fsync) and varies with the filesystem, not with the encoding change.

## Task Commits

1. **Task 1 RED: failing streaming-encode equivalence test** - `b0d7782` (test)
2. **Task 1 GREEN: stream dump cycles through SaveStream** - `1e1e6f8` (feat)
3. **Task 2 RED: fill-error hygiene and injected-write callback** - `f4ffc17` (test)
4. **Task 2 GREEN: clean the temp on a fill error, run the injected fill** - `6f9f801` (feat)
5. **Task 3 RED: v2 load outcome contract** - `bc43705` (test)
6. **Task 3 GREEN: classify the load outcome and measure the streamed peak** - `0e7e722` (feat)

**Plan metadata:** the `docs(05-04)` completion commit (this commit).

## Files Created/Modified

- `internal/app/persist/encode_stream_internal_test.go` — byte-identical and max-single-write streaming proofs (white-box for `maxDumpLineBytes`).
- `internal/app/persist/codec.go` — `EncodeTo` streams one line per `Write`; `Encode` delegates to it.
- `internal/app/persist/engine.go` — `runDump` snapshots once and streams through `SaveStream`; `Open` passes its clock to `Load`.
- `internal/app/persist/bench_test.go` — streamed cycle, counting writer, `peak-live-B`, `bytes/op`, `dump-bytes`.
- `internal/app/persist/engine_test.go`, `engine_blocking_test.go` — the two persist doubles implement `SaveStream`/`LoadResult`; the blocking fake parks the streamed save.
- `internal/port/dump.go` — `SaveStream`; `Load(now, parse) LoadResult`; `LoadOutcome`, `Generation`, `GenerationFailure`, `LoadResult`; `Save` leaves the interface.
- `internal/persistfile/fsops.go` — `write(dir, fill)` through `bufio` (`maxDumpBufioBytes` = 64 KiB), with fill/flush temp hygiene.
- `internal/persistfile/dumpfile.go` — `SaveStream`, `Save` as its wrapper, and the four-outcome `Load` classification.
- `internal/persistfile/snapshot.go` — `CreateSnapshot` wraps its bytes in a one-`Write` fill; behavior unchanged.
- `internal/persistfile/dumpfile_test.go`, `fsops_test.go` — outcome table, failure-matrix fill row, injected-write callback, new `Load` shapes.
- `internal/grpcapi/server_test.go` — the API harness fake implements `SaveStream`/`LoadResult`.

## Decisions Made

- **`EncodeTo` is the one encoder.** `Encode` is a `bytes.Buffer` wrapper, so the byte-identity test pins the only production path.
- **The fill is caller-goroutine and non-retaining.** `SaveStream` documents that the fill runs once and must not retain the writer; the adapter owns the temp lifetime.
- **Load only classifies here.** Absent candidates are not failures; preservation and the orphan sweep are 05-05's additive work, so `Swept`/`SweepErr` stay zero and `Preserved` stays empty.
- **`Save` leaves the port, stays on `FileStore`.** Tests and snapshot fixtures keep the convenience; the engine has no byte-slice save path.
- **Peak sampling point.** `peak-live-B` is sampled inside the fill rather than after release, because that is the moment the one copy and the adapter buffer are both live.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Test/lint convention] Streaming test file named `encode_stream_internal_test.go`**
- **Found during:** Task 1
- **Issue:** The plan names `internal/app/persist/encode_stream_test.go`, but `TestEncodeTo_maxSingleWrite` asserts against the unexported `maxDumpLineBytes()`, and the vendored `testpackage` config exempts only `*_internal_test.go` files.
- **Fix:** Created `internal/app/persist/encode_stream_internal_test.go` with the same two tests.
- **Files modified:** `internal/app/persist/encode_stream_internal_test.go`
- **Verification:** `go tool golangci-lint run ./internal/...` is clean; both tests pass.
- **Commit:** `b0d7782`

**2. [Rule 1 - TDD sequencing] Fill-error hygiene landed in Task 2, not Task 1**
- **Found during:** Task 1
- **Issue:** Task 1's action text includes the fill-error temp cleanup, but Task 2's named behavior and acceptance criteria own it. Implementing it in Task 1 would have made Task 2's RED pass immediately.
- **Fix:** Task 1 streamed through the temp without the fill/flush cleanup; Task 2 added the failing hygiene row and then the `closeRemoveTemp` path. No behavior was dropped within the plan — the final state matches the plan's text.
- **Files modified:** `internal/persistfile/fsops.go`, `internal/persistfile/fsops_test.go`
- **Verification:** `TestFsops_failureAtEachStep/fill_failure_cleans_the_temp,_main_untouched` red on `f4ffc17`, green on `6f9f801`; full `internal/persistfile` suite green.
- **Commit:** `6f9f801`

**3. [Rule 3 - Blocking issue] Stale plan commit ledger from a prior milestone**
- **Found during:** Before the first task commit
- **Issue:** `.git/gsd-plan-head-before-05-04` already existed from a 2025-11-09 run and pointed at `2d41087`, which would have made `commits:` measure months of history.
- **Fix:** Reset the ledger to this plan's base (`98d1793`) so the measured commit count is this plan's own.
- **Files modified:** `.git/gsd-plan-head-before-05-04` (local git state, not tracked)
- **Verification:** `git rev-list --count 98d1793..HEAD` = 6, matching the six task commits.
- **Commit:** n/a (pre-commit bookkeeping)

## Issues Encountered

- **Full-suite flake, not this plan's code.** One `go test -race ./...` run failed in `cmd/sup-kv` `TestLog_formatText` with `probe rx bind: ... address already in use`; the test passed twice on retry. This is the documented UDP pick-to-bind flake class and is unrelated to the persistence changes. Recorded in `deferred-items.md`.
- **RED evidence for compile-coupled interface changes.** A test referencing a not-yet-existing symbol is a build failure, which the RED classifier refuses. Both interface tasks therefore landed the new symbol as a declaration-only stub in the test commit so the target test failed on its assertion (empty bytes / zero writes; zero-result `Load`). The GREEN commit replaced the stub.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The port now carries both the streaming save and the load outcome; plan 05-05 can add the orphan sweep, preservation and the engine report on top without another interface break.
- All four `DumpStore` implementations compile and pass; `port.Save` removal is already done, so 05-05 only adds fields/methods on the result side.
- Ready for 05-05.

## Self-Check: PASSED
