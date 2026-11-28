---
phase: 02-stand-and-simulator-as-tests
plan: 07
subsystem: testing
tags: [decoder-negative-table, discard-counters, xport-10, f2c, deletion-replication, tombstone, stand-harness, test-02, test-08, d-08]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: "plan 02-03's simnet Cluster.Delete (fresh-version tombstone through NotifyLocalWrite) — the simulator level the stand levels mirror"
  - phase: 02-stand-and-simulator-as-tests
    provides: "plan 02-02/02-14's internal/stand harness (build-once binary, launch/execPair plumbing, awaitServing, dump-grep seams, derived bounds)"
provides:
  - "internal/udprepl/decode_negative_test.go — TestDecode_negativeTable (8 corruption rows: verdict + DiscardStats delta) and TestWire_deleteRoundTrip (DataReplication tombstone codec + persist dump codec)"
  - "internal/stand/stand_delete_test.go — TestStand_deleteReplication (two-process Delete -> NotFound on B, tombstone in B's dump, delete-twice exactly one tombstone) and TestStand_deleteWhileNodeDown (node stopped across the delete learns the tombstone after start)"
  - "the complete four-level TEST-02 proof across the phase: codec (this plan), simulator (02-15 + 02-06 chaos mix), two-process and node-down (this plan)"
  - "XPORT-10 forward-reference rows pinning the CURRENT Decoded behavior of empty-key and tombstone-with-data datagrams, ready to flip in Phase 3"
affects: [02-17 (fuzz corpora mirror this table's corruption classes), 02-15 (simulator deletion level), Phase 3 (XPORT-10/oversize row flips)]

# Actuals (#2632) — same estimateTokens scale as the plan estimate: chars/4 over the realized diff.
actuals:
  tokens: 8222
  tasks: 2
  commits: 2

plan_head_before: 7cbc0ef1ea16379237f83b890d09b459c36fcdbf
plan_head_after: 56da2fc9b22262fa37e15d6607bfcfd93acd93dd

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Exact discard-delta assertions ride an asymmetric adapter (bind P1, broadcast into a test-held sink socket at P2) so own-echo never contaminates the class counters"
    - "Fixture construction guards: the truncated-body row asserts proto.Unmarshal fails on the constructed bytes before decode() sees them"
    - "Processed-row evidence is the adapter's apply Ack where no discard class exists today (the XPORT-10 current-behavior rows)"
    - "Live dump-grep polling treats a missing or stale generation as not-yet; the final confirmed dump after SIGTERM proves durability"
    - "Returning-peer bounds derived from flags the test sets explicitly (pinned sync-response-rate + shrunken stand timers), never magic constants"

key-files:
  created:
    - internal/udprepl/decode_negative_test.go
    - internal/stand/stand_delete_test.go
  modified: []

key-decisions:
  - "The negative table asserts decode()'s CURRENT verdict separately from the core's counter: wrong proto_version is Decoded at the codec and BadProtoVersion at the core, exactly the split D-02 assigns"
  - "Empty-key and tombstone-with-data rows are Decoded today (XPORT-10, Phase 3) and use the adapter's Ack as processed-evidence; they flip to the rejection class with the fix, in place"
  - "Oversize is a decode-level row only: the kernel refuses the real send (F2c), and decode() has no size bound today; XPORT-03 adds it in Phase 3"
  - "The truncated-body fixture cuts the trailing checksum varint (forced four bytes via Checksum 0xFFFFFF) so the CRC gate passes and the body unmarshal is the failing branch, with a construction assertion guarding the fixture"
  - "Delete-twice asserts codes.NotFound from the second Delete (the store's no-op, no version burned) and exactly one dump record line for the key — the tombstone counts, not versions"
  - "The dump-codec round-trip also pins the Deleted:true-with-data echo (codec fidelity), documented as current behavior alongside the negative table's XPORT-10 row"

patterns-established:
  - "Counter-exactness without own-echo: a sink socket holds the broadcast port for the test's lifetime, so the only verdicts observed are the fed datagrams'"
  - "Adversarial hand-built envelopes for domain checks (bad run_id, truncated body) while production-encoder helpers cover the wire-real rows"
  - "Dump-grep assertion helper pair: standDumpHoldsTombstone (live waitFor condition) and standDumpSingleLine (exactly-one-record contract)"

requirements-completed: [TEST-02, TEST-08]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "8-row corrupted-datagram negative table asserting each row's CURRENT mechanical verdict (Decoded/BadCRC/Undecodable)"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "internal/udprepl/decode_negative_test.go#TestDecode_negativeTable"
        status: pass
    human_judgment: false
  - id: D2
    description: "Per-row DiscardStats delta through Adapter.Discards() for every class that exists today (BadCRC, Undecodable, BadProtoVersion) — only the driven class increments"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestDecode_negativeTable$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "XPORT-10 forward-reference rows: empty key and tombstone-with-data pin CURRENT Decoded+processed behavior ready to flip in Phase 3; oversize pins decode-has-no-size-bound (F2c/XPORT-03)"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "internal/udprepl/decode_negative_test.go#TestDecode_negativeTable (empty_key..., tombstone_with_data..., oversize...)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Deletion codec round-trip: DataReplication tombstone survives the wire codec for ASCII and multi-byte UTF-8 keys; Deleted:true records round-trip through the persist dump codec with the greppable marker"
    requirement: "TEST-02"
    verification:
      - kind: unit
        ref: "internal/udprepl/decode_negative_test.go#TestWire_deleteRoundTrip"
        status: pass
    human_judgment: false
  - id: D5
    description: "Two-process deletion: Delete on A -> NotFound on B within the configured-timers bound, tombstone line in B's dump, delete-twice leaves exactly one tombstone line, final confirmed dump keeps it"
    requirement: "TEST-02"
    verification:
      - kind: integration
        ref: "internal/stand/stand_delete_test.go#TestStand_deleteReplication"
        status: pass
    human_judgment: false
  - id: D6
    description: "Node-down-during-delete: B stops holding the pre-delete value, A deletes, B restarts and learns the tombstone within the returning-peer bound; B's dump shows the tombstone line"
    requirement: "TEST-02"
    verification:
      - kind: integration
        ref: "internal/stand/stand_delete_test.go#TestStand_deleteWhileNodeDown"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 7: Decoder Negative Table and Deletion Levels Summary

**The UDP decoder is pinned by an 8-row corrupted-datagram table asserting each corruption's current verdict and discard counter, and deletion replication is proven at the codec and both two-process levels — tombstone on the wire and on B's disk, plus a node that learns the delete after restart.**

## Performance

- **Duration:** 6 min (350 s)
- **Started:** 2026-11-21T11:05:27Z
- **Completed:** 2026-11-21T11:11:17Z
- **Tasks:** 2
- **Files created:** 2 (no production file touched)

## Accomplishments

- **Negative table (TEST-08, T-2-13):** `TestDecode_negativeTable` drives `decode()` with eight corruption classes — bad CRC, truncated envelope, truncated body, wrong proto_version with correct CRC, bad run_id, oversize, empty key, tombstone-with-data. Each row asserts its current verdict and, where a counter exists today, the exact `DiscardStats` delta through `Adapter.Discards()` after feeding the bytes over a real loopback socket. Counter rows prove only the driven class increments; the two XPORT-10 rows prove today's `Decoded`+Ack behavior with forward references; the oversize row proves `decode()` has no size bound (the kernel's EMSGSIZE is F2c's real gate, XPORT-03).
- **Deletion codec level (TEST-02 level 1):** `TestWire_deleteRoundTrip` encodes a `Deleted:true` DataReplication (ASCII and multi-byte UTF-8 keys), decodes it back with the flag, empty Data, version and checksum intact, and round-trips tombstone records through the persist dump codec — including the `"deleted":true` marker the stand assertions grep.
- **Two-process level (TEST-02 level 3):** `TestStand_deleteReplication` converges a live value, deletes on A, observes NotFound on B within the timer-derived bound, greps the tombstone line out of B's live dump, proves the delete-twice no-op (`codes.NotFound`, exactly one dump line), and confirms the tombstone survives B's final confirmed SIGTERM dump.
- **Node-down level (TEST-02 level 4):** `TestStand_deleteWhileNodeDown` stops B with the pre-delete value in its dump, deletes on A while B is down, restarts B and asserts it learns the tombstone within the returning-peer bound derived from the flags, with B's dump replacing the data line with the tombstone line.
- **Four-level TEST-02 linkage recorded in code comments:** codec (this plan), simulator (`TestSim_Delete_replicatesTombstone` from plan 02-15 plus plan 02-06's chaos deletions), two-process and node-down (this plan).

## Task Commits

Each task was committed atomically:

1. **Task 1: corrupted-datagram negative table + deletion codec round-trip** - `918078a` (test)
2. **Task 2: deletion replication — two-process levels on the stand** - `56da2fc` (test)

**Plan metadata:** pending docs commit (SUMMARY/STATE/ROADMAP/REQUIREMENTS)

## Files Created/Modified

- `internal/udprepl/decode_negative_test.go` — 8-row `negativeRow` table with per-row verdict + counter assertions, the asymmetric sink-broadcast adapter helper, hand-built envelope/truncated-body fixture builders, and `TestWire_deleteRoundTrip` (wire + persist codec halves)
- `internal/stand/stand_delete_test.go` — the two stand deletion scenarios plus gRPC Delete helpers and the JSONL dump-grep helpers (`standDumpHoldsTombstone`, `standDumpSingleLine`)

## Decisions Made

- **Verdict and counter are asserted separately:** `decode()` frames, the core classifies — the wrong-proto_version row shows `Decoded` at the codec and `BadProtoVersion` at the core, pinning exactly where each layer's responsibility ends.
- **Own-echo isolation for exact counters:** the fed adapter binds one port and broadcasts into a test-held sink port, so no own-broadcast verdict can move a counter between the before/after snapshots — the exactness assertion is structural, not probabilistic.
- **XPORT-10 rows are current-behavior pins, not red tests:** empty-key and tombstone-with-data rows assert `Decoded` today and use the apply Ack as proof they were processed; Phase 3 flips the expected verdict in the same row.
- **The dump-codec round-trip also covers Deleted:true-with-data** as a codec-fidelity echo (documented current behavior), matching the negative table's forward-reference row without asserting any rejection the codec does not perform.
- **The returning-peer bound is derived from flags this test sets** (`--sync-response-rate=500` passed explicitly, shrunken stand timers), so the bound never depends on a production default this package cannot read.

## Deviations from Plan

None - plan executed exactly as written. No production changes, no new files beyond the plan's two, no red gate (both scenarios are green pins of current behavior).

## Issues Encountered

- **Lint caught a redundant `//nolint:gosec`** in the dump-read helper (gosec is already excluded for `_test.go`); removed so `nolintlint` stays clean.
- **Lint caught inline error handling** in the truncated-body fixture guard; rewritten as a plain assignment per `noinlineerr`.
- Neither touched the assertions; both fixed before the task commits landed.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. Every assertion drives production code: `decode()`/`portMessage` for the table, the production encode path, the persist codec, the real `cmd/sup-kv` binary over real UDP/gRPC, and the real dump files on disk.

## Threat Flags

None — test-only additions. The negative table is the T-2-13 mitigation itself (decoder reject behavior + counter observability pinned); no new network endpoint, auth path, file-access pattern or schema change at a trust boundary.

## Next Phase Readiness

- **Plan 02-17 (fuzz targets):** its `FuzzDecode` seed corpus mirrors this table's corruption classes by definition (bad CRC, truncated envelope, truncated body, wrong proto_version, bad run_id) — the classes are now executable rows, so corpus drift is checkable against them.
- **Plan 02-15 (simulator deletion level):** `TestSim_Delete_replicatesTombstone` completes level 2; the stand levels here assume the same `Cluster.Delete` semantics (ErrNotFound no-op, fresh-version tombstone through `NotifyLocalWrite`).
- **Phase 3 (XPORT-10/XPORT-03):** the three current-behavior rows (`empty_key...`, `tombstone_with_data...`, `oversize...`) are the flip points — change the expected status and add the rejection assertion; no other row moves.
- **TEST-02 and TEST-08 stay open in REQUIREMENTS.md:** `requirements.ready-ids` reports both blocked because sibling plans 02-15/02-17 own the remaining halves; this plan's evidence is the codec + both two-process deletion levels and the full negative table, all green under `-race`.

## Self-Check

- [x] `internal/udprepl/decode_negative_test.go` exists (8 table rows + round-trip subtests)
- [x] `internal/stand/stand_delete_test.go` exists (both deletion scenarios)
- [x] Task commits `918078a` and `56da2fc` are ancestors of HEAD
- [x] `go test -race -count=1 ./internal/udprepl/ -run '^TestDecode_negativeTable$|^TestWire_deleteRoundTrip$'` passes
- [x] `go test -race -count=1 ./internal/stand/ -run '^TestStand_delete'` passes on Linux
- [x] `go test -race ./internal/udprepl/ ./internal/stand/` passes
- [x] `make lint` and `make test` pass
- [x] `check evaluation-scope --plan 02-07` returns both commits and both changed files, no missing/outside files

**Self-Check: PASSED**

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
