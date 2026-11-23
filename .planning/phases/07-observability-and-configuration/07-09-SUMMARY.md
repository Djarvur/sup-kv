---
phase: 07-observability-and-configuration
plan: 09
subsystem: observability
tags: [metrics, prometheus, rss, send-accounting, definitions, help-drift]
requires:
  - phase: 07-observability-and-configuration
    provides: the adapter/mirror/exporter surface (07-01), the pending-only conflict counter (07-06) and the four routed core counters' accessors
provides:
  - honest sent accounting — a datagram counts only after WriteToUDP accepts it
  - the four routed Phase-4 counters published as unlabeled families and mirrored through the adapter
  - current Linux RSS from /proc/self/statm, with the documented darwin peak fallback
  - the definition guards TestMetrics_definitions and TestParseStatm/TestCurrentRSS
affects: [07-10, docs/OPERATIONS.md]
actuals:
  tokens: 11094
  tasks: 3
  commits: 4
plan_head_before: 706d8b5803dc463607bd6a82a47e312162026af4
plan_head_after: 812896951f428cba2530cf11134e5196705c5089
tech-stack:
  added: []
  patterns:
    - "Write-path accounting: a counter increments only after the socket accepts the write, and each failure branch has its own fake-socket case"
    - "Platform split by build tag: one current-value probe on Linux, one documented-peak fallback elsewhere"
    - "Definition drift guard: lowercase the rendered # HELP line and assert one semantic phrase per metric"
key-files:
  created:
    - internal/udprepl/mirror_internal_test.go
    - internal/metrics/rss_linux.go
    - internal/metrics/rss_other.go
    - internal/metrics/rss_linux_internal_test.go
    - internal/metrics/metrics_definitions_test.go
    - .planning/phases/07-observability-and-configuration/red-07-09-task1-sent.json
  modified:
    - internal/udprepl/loop.go
    - internal/udprepl/adapter.go
    - internal/udprepl/mirror.go
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
    - cmd/sup-kv/main.go
key-decisions:
  - "sent means successfully handed to the socket: write returns the socket's verdict and flush counts only on nil; encode, resolve and WriteToUDP failures count nothing"
  - "The four Phase-4 counters are unlabeled counters with definition Helps, mirrored post-event like the discard counters and published at zero when unseen"
  - "Linux RSS is /proc/self/statm field 2 x os.Getpagesize(); the !linux file keeps ru_maxrss and the Help names it the development-only peak"
  - "The fake send socket is named flushPacketConn because fakePacketConn is already the read-loop fake in this package's test binary"
  - "rss_linux_test.go became rss_linux_internal_test.go: the vendored testpackage config exempts only *_internal_test.go and the parser is unexported"
patterns-established:
  - "Pattern: a counter's definition lives in its Help and TestMetrics_definitions fails when a definition phrase disappears"
  - "Pattern: one fake-socket case per non-counting failure branch (refused write, resolve failure, encode failure) beside the counting success case"
requirements-completed: []
coverage:
  - id: D1
    description: "Send accounting: write returns an error and flush counts into SentByType only after the socket accepted the datagram; a refused write, an unresolvable unicast handle and an encode failure count nothing"
    requirement: "OBSV-03"
    verification:
      - kind: unit
        ref: "internal/udprepl/mirror_internal_test.go#TestFlush_countsOnlySuccessfulWrites"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/"
        status: pass
    human_judgment: false
  - id: D2
    description: "The four routed Phase-4 counters (listings dropped, sync streams dropped, listed keys evicted, notifications regressed) have adapter mirrors, metrics families rendered at zero and exact values, and main wiring"
    requirement: "OBSV-03"
    verification:
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestMetrics_families / #TestMetrics_values"
        status: pass
      - kind: unit
        ref: "internal/udprepl/mirror_internal_test.go#TestMirror_listingCounters"
        status: pass
      - kind: process
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run TestMetrics_optIn"
        status: pass
    human_judgment: false
  - id: D3
    description: "Current Linux RSS from statm field 2 x page size, the documented ru_maxrss peak fallback off Linux, and the darwin cross-build"
    requirement: "OBSV-03"
    verification:
      - kind: unit
        ref: "internal/metrics/rss_linux_internal_test.go#TestParseStatm / #TestCurrentRSS"
        status: pass
      - kind: build
        ref: "GOOS=darwin go build ./internal/metrics/"
        status: pass
    human_judgment: false
  - id: D4
    description: "One definition guard per metric: the rewritten conflict/sent/duration/drift/RSS Helps and the four counter Helps each carry their semantic phrase"
    requirement: "OBSV-03"
    verification:
      - kind: unit
        ref: "internal/metrics/metrics_definitions_test.go#TestMetrics_definitions"
        status: pass
    human_judgment: false
duration: 15min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 09: Honest metric definitions and the four counters Summary

Sent now means the socket accepted the datagram, memory is current on Linux, the four routed Phase-4 counters render at zero and at exact values, and `TestMetrics_definitions` fails if any published definition phrase disappears.

## Performance

- **Duration:** 15min (started 2026-11-23T11:02:51Z, completed 2026-11-23T11:18:12Z)
- **Tasks:** 3 of 3
- **Commits:** 4 (1 RED test, 3 implementation; plus the metadata commit)
- **Files modified:** 12 (6 created, 6 modified)
- **Realized diff:** 723 insertions, 45 deletions

## Accomplishments

- **Sent means handed to the socket.** `Adapter.write` returns the socket's verdict and `flush` calls `countSent` only on a nil return; a refused `WriteToUDP`, an unresolvable unicast handle and an encode failure all count nothing. `TestFlush_countsOnlySuccessfulWrites` drives all five cases through one switchable fake socket, and the full `internal/udprepl` suite stayed green — no scenario depended on the old count-before-write order.
- **The four Phase-4 counters are visible in production.** The adapter mirrors `ListingDropped`, `SyncStreamDropped`, `ListedKeysEvicted` and `NotificationsRegressed` into atomics after every loop event and exposes four accessors; `metrics.Sources` gained the matching fields, `listingFamilies` renders `supkv_replication_{listings_dropped,sync_streams_dropped,listed_keys_evicted,notifications_regressed}_total` as unlabeled counters, and `main.metricsSources` wires all four. Every `metrics.Sources` literal, the family table, the zero-state sample list and the by-effect value rows were updated in the same commit, so a scrape can never meet a nil accessor.
- **`TestMirror_listingCounters` proves the mirror tracks the core**, driving the regression counter with an equal duplicate notification and the eviction counter with a listing whose two-sync-interval retention expires, then comparing all four accessors with the core after one `mirrorStats` pass.
- **Memory is current on Linux.** `rss_linux.go` parses `/proc/self/statm` field 2 (resident pages) times `os.Getpagesize()`; `rss_other.go` keeps `ru_maxrss` and says in the Help that the development-only darwin build reports the peak. `processUsage` split into `processCPU` (user+system seconds) plus `currentRSS()`. `TestParseStatm` covers the valid, whitespace, truncated, empty, non-numeric and negative shapes; `TestCurrentRSS` asserts the live probe is positive; the darwin cross-build compiles.
- **Every definition is stated and guarded.** Five Helps were rewritten (pending-only conflicts, socket-accepted sends, the last dump *attempt*, wall-time-leading drift, current RSS) and the four counter Helps state their definitions. `TestMetrics_definitions` renders a zero-state snapshot, reads each `# HELP` line and asserts its phrase against the lowercased text, so sentence case can neither manufacture nor mask a match.

## Task Commits

1. **Task 1 (RED): the failing send-accounting test** — `d84ca19` (test)
2. **Task 1 (GREEN): count a datagram as sent only after the socket accepts it** — `bf8ccbb` (feat)
3. **Task 2: publish the four routed Phase-4 counters** — `4b51e27` (feat)
4. **Task 3: current RSS and one test per definition** — `8128969` (feat)
5. **Plan metadata** — the docs commit that adds this SUMMARY (see `git log`)

## Files Created/Modified

- `internal/udprepl/loop.go` — `write` returns the socket error; `flush` counts after the write guard; `countSent`/`flush` doc comments state "successfully handed to the socket"
- `internal/udprepl/adapter.go` — four `atomic.Uint64` mirror fields for the routed Phase-4 counters
- `internal/udprepl/mirror.go` — four accessors and their `mirrorStats` stores; the mirror doc lists them
- `internal/udprepl/mirror_internal_test.go` — `flushPacketConn` fake, `TestFlush_countsOnlySuccessfulWrites` (five cases), `TestMirror_listingCounters`
- `internal/metrics/metrics.go` — `Sources` gains the four fields; `listingFamilies`; `Snapshot` appends them; five Helps rewritten; `processUsage` → `processCPU`; `runtimeFamilies` reads `currentRSS()`
- `internal/metrics/rss_linux.go` — `statmResidentField`, `parseStatm`, `currentRSS`
- `internal/metrics/rss_other.go` — `ru_maxrss` peak fallback, documented
- `internal/metrics/rss_linux_internal_test.go` — `TestParseStatm` table, `TestCurrentRSS` integration
- `internal/metrics/metrics_definitions_test.go` — `helpTexts` and the per-definition `TestMetrics_definitions`
- `internal/metrics/metrics_test.go` — four family rows, four `zeroSources` accessors, four zero-state lines, four by-effect value rows
- `cmd/sup-kv/main.go` — `metricsSources` wires the four accessors
- `.planning/.../red-07-09-task1-sent.json` — the RED evidence record (`RED_EVIDENCE_OK`)

## Decisions Made

- **Counting point:** the counter increments after `WriteToUDP` returns nil, and the error is simply returned from `write` (already logged there). Encode and resolve failures keep their existing `continue` with no count.
- **Counter names and shape:** the four families use the research A11 names with the `_total` suffix, are unlabeled (one dimension each), and render at zero when unseen — a zero count is information.
- **RSS fallback:** the non-Linux probe reports the peak rather than omitting the family; the Help names it the development-only peak, so no scrape mistakes it for a current value.
- **Test placement:** all new udprepl tests live in the plan's `mirror_internal_test.go`; the metrics definition guard is its own file so the drift guard is obvious.
- **Requirements:** `OBSV-03` stays unchecked — `requirements.ready-ids` reports it blocked because a sibling plan also declares it (shared-ID gate).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking compile/name collision] The fake send socket is `flushPacketConn`, not a second `fakePacketConn`**
- **Found during:** Task 1, writing `mirror_internal_test.go`
- **Issue:** `fakePacketConn` already exists in `readloop_internal_test.go`; both files compile into the same `udprepl` test binary, so the plan's literal type name would not compile.
- **Fix:** the write-side fake is `flushPacketConn` — same `packetConn` seam, switchable `WriteToUDP` fault and a blocking read, with a comment naming the collision.
- **Files modified:** `internal/udprepl/mirror_internal_test.go`
- **Verification:** the five-case suite and the full udprepl package pass.
- **Commit:** `d84ca19`

**2. [Rule 1 - Lint gate] `rss_linux_test.go` lives as `rss_linux_internal_test.go`**
- **Found during:** Task 3, `golangci-lint run ./internal/metrics/`
- **Issue:** the vendored `testpackage` config exempts only `*_internal_test.go`; the plan's filename triggered `testpackage`, and `parseStatm`/`currentRSS` are unexported, so an external test package cannot drive the parser table.
- **Fix:** renamed the file to the exempt suffix. This keeps the lint gate clean without a `//nolint` suppression (AGENTS.md deviation 3) and follows the repo's existing white-box naming.
- **Files modified:** `internal/metrics/rss_linux_internal_test.go`
- **Verification:** `make lint` exits 0; the parser and integration tests pass.
- **Commit:** `8128969`

## Issues Encountered

- `noinlineerr` flagged the first `TestMetrics_definitions` draft's inline `if err := Render(...)`; rewritten to the assigned form the vendored config requires. No other lint findings.
- Full gates after Task 3: `make lint` 0, `make test` (`go test -race -cover ./...`) 0, including `internal/app/repl` at 202 s.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Ready for 07-10. Two plans remain in Phase 7 (07-10, 07-11).
- 07-10's OPERATIONS metric catalog should now carry the definitions this plan made true: pending-only conflicts, socket-accepted sends ("UDP sent is not delivery"), the last dump *attempt* duration, wall-time-leading drift, current Linux RSS with the darwin peak caveat, and the four routed counters.
- `OBSV-03` remains pending until the last sibling plan that declares it completes.

## Self-Check

- Created files exist: `internal/udprepl/mirror_internal_test.go`, `internal/metrics/rss_linux.go`, `internal/metrics/rss_other.go`, `internal/metrics/rss_linux_internal_test.go`, `internal/metrics/metrics_definitions_test.go` — FOUND
- Commits exist on HEAD's ancestry: `d84ca19`, `bf8ccbb`, `4b51e27`, `8128969` — FOUND
- Working tree clean at SUMMARY time — confirmed
