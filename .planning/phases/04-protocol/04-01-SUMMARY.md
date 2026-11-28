---
phase: 04-protocol
plan: 01
subsystem: replication-protocol
tags: [proto-01, proto-07, datasync-header, listing-completeness, atomic-mirror, lru-eviction, pre-body-version-verdict, bad-sync-header, status-complete]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "the pure repl core, the port message seam, the paced listing/record/egress queues, the wire-validation pipeline, the adapter atomic-mirror pattern and the discard class/counter surface this plan extends"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the R-03 red rows that the complete-listing gate turns green (their production signal is what this plan makes mean 'complete')"
provides:
  - "the v2 wire schema: DataSync carries sync_id (uint64), chunk_index and chunk_count (uint32) on field numbers 2-4, with regenerated committed code"
  - "core-side per-(node_id, run_id, sync_id) accumulation with bounded streams/entries and a completeness predicate, so a partial listing never opens the gate"
  - "one atomic {complete, listedKeys} listing mirror, published only on a content-generation move; Adapter.ListedKey reads the same pointer"
  - "retained listed-key state bounded by maxListedKeysTotal with whole-peer-set LRU eviction, refcounted shared keys and a counted release"
  - "proto_version judged in the adapter after the envelope and before CRC/body, exposed as DecodeStatus ProtoVersionMismatch"
  - "the bad_sync_header discard class, counted and WARNed per (node_id, class) on event time, published in both metrics directions"
  - "re-derived chunk-budget fixtures on both sides (96-byte overhead, sum(len)=1112 lenses)"
affects:
  - "04-02..04-13 build on the header, the accumulation and the listing mirror; the gate/RPC plans consume SyncArrived/ListedKey and the per-peer WARN key"
  - "Phase 5's startup gate reads the mirror's complete flag; the listed-key set is the 14.5(6) seam the RPC plans consume"

actuals:
  tokens: 29355
  tasks: 3
  commits: 6
  plan_head_before: d9042cb4995083038867c11a4668019c15ed56c5
  plan_head_after: d1760dba239548bfef1d08ae444f5c27ae060154

tech-stack:
  added: []
  patterns:
    - "Completeness is wire state, not arrival: a monotone sync_id + 1-based chunk_index + chunk_count accumulated as an index set; only every index 1..count opens the gate."
    - "One immutable {complete, keys} value published by one atomic pointer, with a content-gated generation: identical repeated listings republish nothing and clone nothing."
    - "A remotely fillable retained set is bounded by whole-peer-set LRU eviction through refcounts, so shared keys survive until the last lister drops them and every released key is counted."
    - "The protocol version verdict precedes the CRC/body: a foreign version's body is never decoded, and a combined future-version/corrupt-body datagram pins the order."
    - "Rate limits key on (node_id, class) over event time, so one peer's flood cannot suppress another peer's first WARN."

key-files:
  created:
    - internal/app/repl/listedkeys_internal_test.go
    - .planning/phases/04-protocol/red-evidence/04-01-task1.json
    - .planning/phases/04-protocol/red-evidence/04-01-task2.json
    - .planning/phases/04-protocol/red-evidence/04-01-task3.json
  modified:
    - internal/udprepl/replproto/supkv/v1/supkv.proto
    - internal/udprepl/replproto/supkv/v1/supkv.pb.go
    - internal/port/repl.go
    - internal/udprepl/repl.go
    - internal/app/repl/core.go
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
    - internal/udprepl/decode_negative_test.go
    - internal/udprepl/fuzz_decode_test.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/repl_internal_test.go
    - internal/udprepl/syncsize_internal_test.go
    - internal/app/repl/sync_test.go
    - internal/app/repl/discard_test.go
    - internal/app/repl/egress_internal_test.go
    - internal/app/repl/bench_test.go

key-decisions:
  - "sync_id is a monotone per-core counter (nextSyncID starting at 1), never crypto/rand, and the accumulation key includes the peer run_id: same-seed simulator replay stays byte-identical and a restart's fresh counter cannot merge into its old run's partial state."
  - "Header fields are stamped at emission from each chunk's ORIGINAL position; rotation reorders transmission only, because completeness compares indices 1..count, never emission position."
  - "The listing generation is a union-content version: it moves only when a key enters/leaves the union or the complete flag first flips. A second peer listing already-listed keys, or a peer dropping keys another still lists, moves nothing and republishes nothing."
  - "Listed-key eviction is whole-peer-set LRU (container/list) and releases through refcounts; ListedKeysEvicted counts only keys that actually left the union, so a shared key is counted once, by its last lister."
  - "Stream eviction is deterministic: fewest received chunks first, ties by (peer, run bytes, sync_id) ascending; per-stream overflow drops that stream, total overflow drops the weakest until the total fits."
  - "The header check runs before per-entry validation (chunk_count>=1, 1<=index<=count, count<=maxSyncChunks, and consistency with a live round), and a disagreeing part leaves the live round's stored count untouched."
  - "The discard WARN limiter key is (node_id, class) on the handler's event time; ProtoVersionMismatch and bad_sync_header share the helper, while the existing Decoded-verdict field check stays as silent defence-in-depth."
  - "SupportedProtoVersion is 2; a version-1 listing cannot express completeness, so accepting it would restore the first-chunk reading (D-07's version bump is what makes the header trustworthy)."
  - "chunkOverheadBytes is 96 (envelope + DataSync framing + header) and both graduated lenses were re-derived to sum 1112; the oversized wire proof compares two datagrams with identical header values."
  - "Task 3 is test hardening: the production decode verdict shipped with task 1 per the plan's acceptance, so its resolving commit is a test commit and the pre-update failure is the RED evidence (see TDD Gate Compliance)."

patterns-established:
  - "A cross-goroutine completeness signal publishes one immutable struct pointer, never a bool plus a map; the reader loads the same pointer for both answers."
  - "A remotely fillable retention set pairs a total cap with whole-peer-set eviction and refcounts, so the union, the per-peer sets and the counters shrink together."
  - "A pre-body framing verdict carries the envelope fields it parsed, so the core can attribute and rate-limit a discard that has no message."
  - "Budget fixtures are re-derived, never relaxed: both sides' lens arithmetic and the 96-byte constant move in the same commit as the header."

requirements-completed: [PROTO-01, PROTO-07]

coverage:
  - id: D1
    description: "DataSync carries sync_id/chunk_index/chunk_count through the real protobuf codec on the v2 schema, with regenerated committed wire code."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/udprepl/repl_test.go#TestWire_dataSyncHeaderRoundTrip"
        status: pass
      - kind: other
        ref: "make proto && make buf-lint && make buf-breaking (clean; only replproto supkv.pb.go changed)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Chunks accumulate per (node_id, run_id, sync_id) into an index set; a partial or reordered listing never opens the gate, and the closing chunk flips SyncArrived/ListingComplete exactly once."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_partialListingKeepsGateClosed"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_headerOutOfOrderAndDuplicate"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_emptyListingCompletes"
        status: pass
    human_judgment: false
  - id: D3
    description: "One atomic {complete, keys} pointer answers SyncArrived and ListedKey from the same snapshot; publication happens only on a content-generation move, so an identical listing flood republishes nothing."
    requirement: PROTO-01
    verification:
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestMirror_listingComplete"
        status: pass
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestMirror_syncArrivedFlip"
        status: pass
      - kind: unit
        ref: "internal/udprepl/repl_internal_test.go#TestMirror_identicalListingFloodDoesNotRepublish"
        status: pass
    human_judgment: false
  - id: D4
    description: "Retained listed keys stay at or under maxListedKeysTotal under a 200-peer flood, with whole-peer-set eviction counted; identical listings move no generation, shared-key churn moves none until the last lister drops."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/listedkeys_internal_test.go#TestSync_listedKeysBounded"
        status: pass
      - kind: unit
        ref: "internal/app/repl/listedkeys_internal_test.go#TestSync_listingGenerationContentOnly"
        status: pass
      - kind: unit
        ref: "internal/app/repl/listedkeys_internal_test.go#TestSync_listingGenerationSharedKeys"
        status: pass
    human_judgment: false
  - id: D5
    description: "A foreign proto_version is classified ProtoVersionMismatch after the envelope and before CRC/body, with the envelope fields preserved; the core counts it once as bad_proto_version, fires OnDiscard once and WARNs per peer."
    requirement: PROTO-07
    verification:
      - kind: unit
        ref: "internal/udprepl/decode_negative_test.go#TestDecode_negativeTable"
        status: pass
      - kind: unit
        ref: "internal/udprepl/fuzz_decode_test.go#FuzzDecode"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_onDiscardSeam"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_warnRateLimitedPerPeer"
        status: pass
    human_judgment: false
  - id: D6
    description: "Malformed listing headers are the bad_sync_header class: header-first validation, exact totals, per-(node_id,class) WARN and a zero-state plus by-value metrics sample."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_badSyncHeader"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_classes"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_totality"
        status: pass
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestMetrics_families"
        status: pass
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestMetrics_values"
        status: pass
    human_judgment: false
  - id: D7
    description: "The chunk estimate stays honest against the real codec: overhead 96 and both graduated lenses sum 1112; a budgeted chunk encodes at or under 1400 bytes and the oversized proof compares same-header datagrams."
    requirement: PROTO-01
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestSync_chunking"
        status: pass
      - kind: unit
        ref: "internal/udprepl/syncsize_internal_test.go#TestSync_chunkWireBudget"
        status: pass
    human_judgment: false
  - id: D8
    description: "The full quality gates pass on the plan's tree: strict lint over every package and race-detector tests over the three touched packages (plus a full-repo non-race run)."
    verification:
      - kind: other
        ref: "make lint (config verify + golangci-lint run ./...)"
        status: pass
      - kind: other
        ref: "go test -race -count=1 ./internal/udprepl/ ./internal/app/repl/ ./internal/metrics/"
        status: pass
    human_judgment: false

duration: 23min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 01: Complete Listing Travels the Wire and Bounds Its Retained Keys

**A listing is now complete on the wire and in the core: DataSync v2 carries sync_id/chunk_index/chunk_count, parts accumulate into an index set, only the closing chunk opens a one-pointer {complete, keys} mirror whose retained key payload is LRU-bounded, and a foreign proto_version is judged before the body is ever touched.**

## Performance

- **Duration:** 23 min
- **Started:** 2026-11-22T03:01:47Z
- **Completed:** 2026-11-22T03:25:16Z
- **Tasks:** 3
- **Files modified:** 17 code/test files plus 3 red-evidence records

## Accomplishments

- The R-03 defect's root is gone: the first DataSync chunk is a PART, never the whole listing. `TestSync_partialListingKeepsGateClosed`, the out-of-order/duplicate test and the empty-listing test pin the completeness predicate.
- The gate's two answers are structurally consistent: `SyncArrived` and `ListedKey` load ONE immutable `{complete, keys}` pointer, published only when the core's content generation moves. An identical complete-listing flood republishes no pointer and clones no map.
- The remotely fillable retained key set is bounded: 200 spoofed peers × 1 000 keys end with `listedKeyCount <= 131 072`, positive `ListedKeysEvicted`, and the gate still open; shared keys survive until their last lister.
- PROTO-07's verdict moved ahead of the body: `decode` classifies a foreign version after the envelope and before the CRC, so a future version with a corrupted body is `bad_proto_version`, never `bad_crc`. The core counts it once, attributes it, and WARNs per peer.
- Listing-header violations are a first-class `bad_sync_header` class, counted, WARNed per `(node_id, class)` on event time, and published in both metrics directions.
- Both chunk-budget fixtures were re-derived (96-byte overhead, sum(len)=1112) and the wire proof is green — no byte-budget assertion was weakened.

## Task Commits

Each task was committed atomically (TDD RED → GREEN where production changed):

1. **Task 1 RED: failing completeness-header tests** - `14d6c60` (test)
2. **Task 1 GREEN: header through wire and core** - `e3690e0` (feat)
3. **Task 2 RED: failing bad_sync_header/per-peer WARN tests** - `8287f07` (test)
4. **Task 2 GREEN: header class, per-peer WARN, metrics** - `53683ee` (feat)
5. **Task 3: pin the pre-body verdict from both ends** - `abb7967` (test)
6. **Generation refinement: union membership only** - `d1760db` (feat)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `internal/udprepl/replproto/supkv/v1/supkv.proto` / `supkv.pb.go` — v2 schema with the DataSync header; generated, never hand-edited
- `internal/port/repl.go` — `DataSync{SyncID, ChunkIndex, ChunkCount}` and `ProtoVersionMismatch`
- `internal/udprepl/repl.go` — header mapping both ways, pre-CRC version verdict, one-pointer listing mirror, `ListedKey`, `bad_sync_header` mirror
- `internal/app/repl/core.go` — SupportedProtoVersion 2, accumulation with three caps, completeness/gate, listed-key retention + LRU, `bad_sync_header`, `(node_id,class)` WARN key, header-stamped emission, 96-byte overhead
- `internal/metrics/metrics.go` / `metrics_test.go` — `bad_sync_header` in the zero-state and by-value surfaces
- `internal/app/repl/sync_test.go`, `listedkeys_internal_test.go`, `discard_test.go`, `egress_internal_test.go`, `bench_test.go` — new behavior tests plus fixture adaptation
- `internal/udprepl/decode_negative_test.go`, `fuzz_decode_test.go`, `repl_test.go`, `repl_internal_test.go`, `syncsize_internal_test.go` — negative table/ordering row, fuzz vocabulary, mirror tests, re-derived budget fixtures
- `.planning/phases/04-protocol/red-evidence/04-01-task{1,2,3}.json` — persisted RED evidence records

## Decisions Made

See frontmatter `key-decisions` — the substantive ones: monotone `sync_id` with run_id in the key; original-position indices under rotation; union-content generation; whole-peer-set LRU with refcounts and last-lister counting; fewest-chunks deterministic stream eviction; header-first validation; `(node_id, class)` event-time WARN; v2 as the completeness-making version bump; 96-byte overhead with both lenses re-derived.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The DataSync benchmark fixture would have benchmarked the discard branch**
- **Found during:** Task 2 (header validation)
- **Issue:** `internal/app/repl/bench_test.go` builds `port.DataSync{Entries: head}` with a zero header. Once `validateSync` checks the header first, the benchmark's datagram would be discarded as `bad_sync_header`, so `BenchmarkHandleDataSync` and the flatness gate would measure a rejected datagram instead of the compare path.
- **Fix:** Stamped a valid 1-of-1 header on the benchmark fixture with a comment naming why.
- **Files modified:** `internal/app/repl/bench_test.go`
- **Verification:** `go test -run '^$' -bench BenchmarkHandleDataSync -benchtime=1x ./internal/app/repl/` and `TestCore_keyCountCostFlat` remain green (full package run).
- **Committed in:** `8287f07` (part of the task 2 test commit)

**2. [Rule 3 - Blocking] The white-box egress tests called the old `handleDataSync` signature**
- **Found during:** Task 1 GREEN compile
- **Issue:** `egress_internal_test.go` drives `core.handleDataSync(now, source, entries)` directly; the planned handler now takes the datagram, message and outbox.
- **Fix:** Added `egressSync` and routed both calls through a header-valid one-part listing; the egress dedupe assertion is unchanged.
- **Files modified:** `internal/app/repl/egress_internal_test.go`
- **Verification:** `TestEgress_boundedDeduped` and the full `internal/app/repl` suite green.
- **Committed in:** `e3690e0`

**3. [Rule 1 - Bug] `TestDiscard_classes` treated the new class WARN as a configuration-error log**
- **Found during:** Task 2 verification
- **Issue:** Its harness errors on any log line unless the row expects a configuration error; the new `bad_sync_header` row correctly warns once, so the row failed for the wrong reason.
- **Fix:** Added a `wantWarn` row field asserting exactly one WARN naming the class, keeping the existing configuration-error and silence assertions exact.
- **Files modified:** `internal/app/repl/discard_test.go`
- **Verification:** `TestDiscard_classes` green; the row still asserts the exact counter set.
- **Committed in:** `53683ee`

---

**Total deviations:** 3 auto-fixed (2 Rule 1, 1 Rule 3)
**Impact on plan:** All three keep existing fixtures/tests honest under the new contract; no behavior, bound or assertion was relaxed, and no scope was added.

## TDD Gate Compliance

RED evidence records are committed under `.planning/phases/04-protocol/red-evidence/`:

- Task 1: `04-01-task1.json` — nine target tests failed, led by `TestSync_partialListingKeepsGateClosed` opening the gate on chunk 1. RED `14d6c60` → GREEN `e3690e0`.
- Task 2: `04-01-task2.json` — `TestDiscard_badSyncHeader` moved no counter. RED `8287f07` → GREEN `53683ee`.
- Task 3: `04-01-task3.json` — `FuzzDecode` rejected the now-live `ProtoVersionMismatch` verdict and the negative table still expected `Decoded`. The plan's own acceptance placed the decode verdict in task 1 ("the pre-body verdict exists"), so this task is test hardening: the resolving commit `abb7967` is a `test(...)` commit and no separate `feat` exists for it. The plan-level RED/GREEN pairs are complete (task 1 and task 2 each have both), and the behavior the tests pin is verified green.
- `d1760db` is a post-task refinement tightening the generation to union membership only, with its own pinning test.

## Issues Encountered

- `make test` (full `-race -cover ./...`) hit one transient `cmd/sup-kv` failure: `TestLog_formatText` could not bind its probe port (`address already in use`). Re-running the package under race passed; this is the documented UDP pick-to-bind flake class, unrelated to the plan's changes. Every touched package passed under race on the same tree.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. No placeholder values, TODOs or unwired data sources were introduced; the deferred HLC absorption (`StateStore.Observe`) is explicitly plan 04-03's, per the plan text, not a stub in this code.

## Next Phase Readiness

Ready for 04-02. The wire header, the accumulation core, the listing mirror and `ListedKey` are in place; the gate/RPC plans can consume `SyncArrived`/`ListedKey`, and the per-peer WARN key is the helper D-06/D-07 ride. The four remaining red rows (R-03 x2, P-01 x2) are later plans' to flip.

---
*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Created files exist: `internal/app/repl/listedkeys_internal_test.go`, the three RED evidence records and this SUMMARY.
- Commits exist and are ancestors of HEAD: `14d6c60`, `e3690e0`, `8287f07`, `53683ee`, `abb7967`, `d1760db`.
