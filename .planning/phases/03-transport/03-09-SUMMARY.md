---
phase: 03-transport
plan: 09
subsystem: transport-limits
tags: [xport-03, key-limit, value-limit, datagram-ceiling, maxdatagram, property-test, f2c, startup-exit-2, docs]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-02's storage.MaxKeyBytes (1 024) and the documented write-side-only key ceiling"
  - phase: 03-transport
    provides: "03-08's receive-side key_too_long validation and the MaxKeyBytes single-constant discipline"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the stand harness and the F2c scenario whose pin this plan rewrites to the fixed behavior"
provides:
  - "udprepl.MaxRecordDatagramBytes(keyBytes, valueBytes int) int and udprepl.MaxDatagramBytes (65 507): the worst-case datagram oracle through the real generated schema"
  - "API key-length enforcement on Set/SetIfAbsent/Delete (InvalidArgument over 1 024 bytes, before the engine) with Get/List deliberately permissive"
  - "startup datagram-ceiling check in finalizeConfig: exit 2 naming --value-max-bytes, the key limit and the computed size when the worst case exceeds 65 507"
  - "F2c rewritten to the fixed behavior: local rejection, no replication, and a 1 024-byte-key / 61 440-byte-value positive boundary"
  - "docs/OPERATIONS.md record-and-datagram limits section and the closed S-DM-03 registry row"
affects:
  - "03-10/03-11 inherit a node whose startup refuses an unsendable configuration before any bind"
  - "Phase 5's load-time accounting owns the permissive read path for pre-upgrade long keys"

actuals:
  tokens: 10411
  tasks: 3
  commits: 3
plan_head_before: 0fd2f44d2cba9a9612389cf27448bbe6c7004230
plan_head_after: 34e9a8fe031b679eb0d7d6874d619f26d9c3f44a

tech-stack:
  added: []
  patterns:
    - "A size oracle marshals the real generated schema instead of hand-summing varint widths: field additions and type widenings cannot silently drift the ceiling."
    - "The oracle models the legal record shape (tombstone flag only on empty values) so the documented exact boundary measures exactly 65 507 bytes while tombstones stay bounded."
    - "A property test walks the accepted domain (key bytes <= 1 024, value bytes <= 64 408) and asserts both the real encoded size and the conservative oracle stay under the ceiling."
    - "Startup probes are bounded by the ceiling they check, so a misconfigured multi-gigabyte value is a config error, never an out-of-memory measurement."
    - "The startup comparison is pinned on both sides executably: exactly 65 507 passes, one byte more exits 2."

key-files:
  created:
    - internal/udprepl/maxdatagram.go
    - internal/udprepl/maxdatagram_test.go
  modified:
    - internal/grpcapi/server.go
    - internal/grpcapi/server_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - internal/stand/stand_test.go
    - docs/OPERATIONS.md
    - docs/SPEC-DEVIATIONS.md

key-decisions:
  - "The oracle encodes the legal record shape: the tombstone flag is set only for the empty-value probe. A valued record is never a tombstone (tombstone-with-data is a receive-side discard class), and setting the flag on a valued record overcounts two bytes, which would reject the documented 64 408-byte boundary that sizes to exactly 65 507."
  - "The startup probe width is clamped to the 65 507-byte ceiling before allocation. A multi-gigabyte --value-max-bytes is over the ceiling before any key or envelope byte, so measuring it literally would turn a config typo into an OOM; the clamped case reports the size as a lower bound."
  - "Key enforcement is write-side only: Set (both create paths) and Delete check len(key) > storage.MaxKeyBytes before the engine; Get and List stay permissive so pre-upgrade long-key records remain readable (RESEARCH Open Question 2), with Phase 5 owning load-time accounting."
  - "The API feature adds no second key literal: server.go, the 03-08 receive path in internal/app/repl/core.go and the startup check all read storage.MaxKeyBytes. The datagram ceiling is exported once as udprepl.MaxDatagramBytes and reused by the startup error."
  - "The F2c scenario keeps its name for traceability but now asserts the fixed behavior; its positive half (1 024-byte key, 61 440-byte value) doubles as the end-to-end default-envelope proof."

patterns-established:
  - "Server-side byte ceilings get an executable equality/adjacency pair: the exact fitting value starts, one byte more exits, and the error names the parameter and both sizes."
  - "A conservative size oracle is tested in both directions: it must cover every real encoding of a legal record, and it must not reject any record the accepted domain admits."
  - "Documentation rows in SPEC-DEVIATIONS close with concrete file:line references to the API, receive and startup checks, not prose."

requirements-completed: [XPORT-03]

coverage:
  - id: D1
    description: "MaxRecordDatagramBytes marshals the real DataReplication/Envelope with maximum-width varints; every table row's conservative size covers both the maximal legal and the typical real encoding; key 1 024 + value 64 408 sizes to exactly 65 507, 64 415 and a 4 096-byte key cross it"
    requirement: XPORT-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestMaxRecordDatagramBytes$'"
        status: pass
      - kind: other
        ref: "local probe: the Body oneof wrapper costs 4 bytes and the tombstone flag 2; the research's 62 539/65 507 numbers match the Deleted=false encoding exactly"
        status: pass
    human_judgment: false
  - id: D2
    description: "The accepted domain property holds: for key lengths 0..1 024 (including a 1 024-byte multi-byte UTF-8 key) and value lengths 0..64 408 plus a deterministic 64-sample sweep, the real encoded datagram fits 65 507 bytes and the conservative oracle covers it without exceeding the ceiling itself"
    requirement: XPORT-03
    verification:
      - kind: unit
        ref: "internal/udprepl/maxdatagram_test.go#TestMaxRecordDatagramBytes_acceptedDomain"
        status: pass
    human_judgment: false
  - id: D3
    description: "Set, SetIfAbsent and Delete reject a key longer than storage.MaxKeyBytes with InvalidArgument before the engine; exactly 1 024 bytes is accepted (Set and Delete), the empty key keeps its own message, and the limit is measured in bytes not runes; Get stays permissive (NotFound for long keys)"
    requirement: XPORT-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_keyLimit$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "Startup exits 2 with an error naming --value-max-bytes, the 1 024-byte key limit and the computed size when the worst case exceeds 65 507; --value-max-bytes=64 000 and the exact boundary 64 408 start and serve; 64 409 and 65 500 exit 2"
    requirement: XPORT-03
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMain_valueMaxDatagram$'"
        status: pass
      - kind: unit
        ref: "cmd/sup-kv/main_test.go subtests: 64000 / exact boundary serve; one byte over / 65500 exit 2"
        status: pass
    human_judgment: false
  - id: D5
    description: "F2c reflects the fix end to end: the 70 000-character key is rejected with InvalidArgument, node A does not serve it, node B never serves it across the full window and its final dump lacks it; a 1 024-byte key with a 61 440-byte value replicates to B and appears in B's final dump"
    requirement: XPORT-03
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_oversizeKeyF2c$'"
        status: pass
    human_judgment: false
  - id: D6
    description: "docs/OPERATIONS.md documents the key ceiling, the value limit, the 65 507-byte datagram ceiling, the 64 408-byte largest value with a maximal key and --max-clock-lead; the S-DM-03 row is closed as implemented with API/receive/startup references"
    requirement: XPORT-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestOperationsDoc$'"
        status: pass
      - kind: other
        ref: "grep -n 'S-DM-03' docs/SPEC-DEVIATIONS.md shows the closed row with internal/app/storage/store.go, server.go, core.go, main.go and maxdatagram.go references"
        status: pass
    human_judgment: false
  - id: D7
    description: "The P-01 expectation rows stay unchanged and red, and --value-max-bytes=64000 remains a valid startup: the rows-unchanged guard is the TSV grep, while the accepted-flag claim is the executable TestMain_valueMaxDatagram"
    requirement: XPORT-03
    verification:
      - kind: other
        ref: "grep -n 'TestRed_P01' scripts/red-expectations.tsv -> both rows end in red"
        status: pass
      - kind: other
        ref: "grep -n 'value-max-bytes=64000' internal/stand/stand_red_test.go -> still present"
        status: pass
      - kind: integration
        ref: "TestMain_valueMaxDatagram/64000 starts and serves"
        status: pass
    human_judgment: false

duration: 602s
completed: 2026-11-22
status: complete
---

# Phase 3 Plan 9: Key and Datagram Ceilings Summary

**One key constant now governs the API, the receive path and startup; the worst-case record is measured through the real codec, so a 64 408-byte value with a 1 024-byte key fits exactly and one byte more exits 2 before any socket opens.**

## Performance

- **Duration:** 602 s (~10 min)
- **Started:** 2026-11-21T21:05:18Z
- **Completed:** 2026-11-21T21:16:18Z
- **Tasks:** 3
- **Files:** 9 (2 created, 7 modified)

## Accomplishments

- `udprepl.MaxRecordDatagramBytes(keyBytes, valueBytes int) int` builds a real `replproto.Body`/`Envelope` with `math.MaxUint32` in every variable-width field (proto_version, node_id, crc32, checksum), a 16-byte version and run id, and marshals both messages. `udprepl.MaxDatagramBytes` (65 507) names the IPv4 UDP payload ceiling once for the startup path.
- The oracle models the legal record shape: the tombstone flag is set only when the value is empty. A valued record can never be a tombstone (tombstone-with-data is a receive-side discard class), and this is what makes key 1 024 + value 64 408 measure exactly 65 507 instead of an overcounted 65 509. Empty-value probes still cover both empty Sets and tombstones.
- `MaxRecordDatagramBytes` is proven in both directions: the table checks `(0,0)`, `(1024,0)`, `(0,61440)`, `(1024,61440)`, `(1024,64408)`, `(1024,64415)` and `(4096,61440)` against the real encoding; `TestMaxRecordDatagramBytes_acceptedDomain` walks key lengths 0..1 024 and value lengths 0..64 408 (plus a 64-sample deterministic sweep and a 1 024-byte multibyte UTF-8 key) asserting both the real datagram and the oracle stay under 65 507.
- `Server.Set` and `Server.Delete` reject `len(key) > storage.MaxKeyBytes` with `InvalidArgument` immediately after the empty-key check and before any engine call; `SetIfAbsent` rides the same path. The error names the length, the 1 024-byte limit and that it counts bytes. `Get` and `List` stay permissive for pre-upgrade long keys, documented with the OQ2 rationale.
- `finalizeConfig` now ends with `checkDatagramCeiling`: the worst case is `MaxRecordDatagramBytes(storage.MaxKeyBytes, value_max_bytes)` and a size over `udprepl.MaxDatagramBytes` exits through `exitUsage` (2) with `value-max-bytes exceeds the datagram ceiling: --value-max-bytes=..., the 1024-byte key limit, the computed size, the 65507-byte ceiling`. Non-positive values use the store's own default; the probe is clamped at the ceiling before allocation.
- `TestMain_valueMaxDatagram` pins four startup cases: 64 000 and the exact 64 408 boundary start and serve; 64 409 and 65 500 exit 2 with the naming message. The equality case is the executable proof that the comparison is `>`, and the 64 000 case is the executable guard for the P-01 claim (the TSV grep remains only the rows-unchanged guard).
- `TestStand_oversizeKeyF2c` now proves the fix on the real binary: the 70 000-character Set returns `InvalidArgument`, node A does not serve the key, node B does not serve it across the full derived window, and B's final dump lacks it. Its positive half writes a 1 024-byte key with a 61 440-byte value; the record replicates to B, is served there, and appears in B's final dump.
- `docs/OPERATIONS.md` gained a "Record and datagram limits" section (key 1 024 bytes measured in bytes, value `--value-max-bytes` default 61 440, the 65 507-byte combined ceiling, the 64 408-byte largest value with a maximal key, startup exit 2, and `--max-clock-lead`). The `S-DM-03` registry row is closed as `implemented` with references to `storage.MaxKeyBytes`, the API check, the receive check, the startup check and the oracle.

## Task Commits

Each task was committed atomically (local lint fixes folded in before the SUMMARY):

1. **Task 1: worst-case datagram oracle + property** — `0b2797e` (feat)
2. **Task 2: API key-length enforcement** — `0458bee` (feat)
3. **Task 3: startup ceiling, F2c rewrite, documented limits** — `34e9a8f` (feat)

**Plan metadata:** the `docs(03-09)` commit that follows this summary.

## Files Created/Modified

- `internal/udprepl/maxdatagram.go` (created) — `MaxDatagramBytes`, `MaxRecordDatagramBytes`, the legal-record-shape rule and the conservative-oracle contract.
- `internal/udprepl/maxdatagram_test.go` (created) — the boundary table, the real-codec cross-check (maximal legal and typical shapes), and the accepted-domain property/sweep including the multibyte boundary.
- `internal/grpcapi/server.go` — length checks in `Set`/`Delete`, `keyTooLongError`, the Get-permissive and XPORT-03 doc comments.
- `internal/grpcapi/server_test.go` — `TestServer_keyLimit` (inclusive/exclusive boundary, empty-key message, UTF-8 byte counting, Get permissiveness, Delete at the limit).
- `cmd/sup-kv/main.go` — `errDatagramCeiling`, the `finalizeConfig` hook and `checkDatagramCeiling` with the bounded probe.
- `cmd/sup-kv/main_test.go` — `TestMain_valueMaxDatagram` (64000 and 64 408 start; 64 409 and 65 500 exit 2) and the boundary constants.
- `internal/stand/stand_test.go` — the F2c rewrite (local rejection, absence window, positive boundary replication and dump assertions), imports, and the removed EMSGSIZE constants.
- `docs/OPERATIONS.md` — the record-and-datagram limits section.
- `docs/SPEC-DEVIATIONS.md` — the closed S-DM-03 row.

## Decisions Made

- **Tombstone shape in the oracle:** the flag rides only the empty-value probe. The alternative (always set) overcounted by two bytes and rejected 64 408, contradicting the research's exact-65 507 boundary; always omitting it would undercount a maximal tombstone. The chosen shape is the honest worst case over records the receive path accepts.
- **Bounded startup probe:** the probe clamps at 65 507 bytes so a pathological `--value-max-bytes` exits 2 instead of attempting a matching allocation. The clamped case reports "at least" the probe size (T-3-31).
- **Write-side-only key enforcement:** matches `storage.MaxKeyBytes`'s documented contract and RESEARCH OQ2; Phase 5 accounts for pre-upgrade long keys at load time.
- **Boundary pinning:** the startup test asserts both sides of `> 65 507` (64 408 starts, 64 409 exits), and 64 000 remains the P-01 executable guard. The TSV grep is deliberately only the rows-unchanged guard.
- **F2c naming:** the scenario keeps its name for traceability but asserts the fixed behavior; the old EMSGSIZE polling and its constants are gone.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Correctness] The oracle's tombstone flag follows the legal record shape**
- **Found during:** Task 1's first run (the 1 024/64 408 row measured 65 509)
- **Issue:** The plan's action did not mention the `Deleted` field. Setting it unconditionally (the first implementation) overcounted two bytes and rejected the documented exact boundary; omitting it unconditionally would undercount a maximal tombstone. A local probe also showed the `Body` oneof wrapper costs 4 bytes, explaining why the research's 65 507 matches the `Deleted=false` encoding when every other varint is maximal.
- **Fix:** `Deleted: valueBytes == 0`. A valued record is never a tombstone (tombstone-with-data is discarded on receive), while the empty-value probe covers both an empty Set and a tombstone.
- **Files modified:** `internal/udprepl/maxdatagram.go`, `internal/udprepl/maxdatagram_test.go`
- **Verification:** `TestMaxRecordDatagramBytes` now reports exactly 65 507 at the boundary; the property test also asserts the oracle never exceeds the ceiling for any accepted record.
- **Commit:** `0b2797e`

**2. [Rule 2 - Correctness] The startup probe is bounded by the ceiling**
- **Found during:** Task 3 implementation review
- **Issue:** `MaxRecordDatagramBytes` allocates the value buffer. A malformed `--value-max-bytes` of multiple gigabytes would OOM during the measurement, violating T-3-31's "exit 2 naming the parameter" mitigation.
- **Fix:** `checkDatagramCeiling` probes `min(valueBytes, udprepl.MaxDatagramBytes)`; a value above the ceiling is over it regardless of key/envelope overhead, and the error reports the probe as a lower bound ("at least N-byte datagrams").
- **Files modified:** `cmd/sup-kv/main.go`
- **Verification:** the ordinary over-ceiling rows still report the exact computed size; the bounded path is structural (no allocation proportional to the configured value).
- **Commit:** `34e9a8f`

**3. [Rule 3 - Verification hygiene] Lint fixes landed after the task commits and were folded back**
- **Found during:** the first `go tool golangci-lint run ./...` after all tasks
- **Issue:** godoclint wanted `[math.MaxUint32]` linked in the oracle's doc, noinlineerr rejected four inline error checks in the key-limit test, and revive rejected the test variable name `real` (a builtin). The fixes touched Task 1 and Task 2 files after their commits.
- **Fix:** the branch was soft-reset to the plan base (local, unpushed) and the three task commits were rebuilt with the lint fixes folded in, so each task commit is lint-clean at HEAD. No content changed beyond the fixes.
- **Files modified:** `internal/udprepl/maxdatagram.go`, `internal/udprepl/maxdatagram_test.go`, `internal/grpcapi/server_test.go`
- **Verification:** `go tool golangci-lint run ./...` green at HEAD.
- **Commit:** `0b2797e`, `0458bee`

---

**Total deviations:** 3 auto-fixed (2 correctness, 1 verification hygiene)
**Impact on plan:** No scope change. Every must-have and acceptance criterion is met; the oracle's record-shape rule is recorded here and in its doc comment so the 65 507 boundary cannot silently drift.

## Issues Encountered

- **Parallel-load test flakes (pre-existing):** in one combined four-package run, the untouched `internal/stand` pacing test failed with `Unavailable: EOF`, and in a serialized rerun the untouched `cmd/sup-kv` `TestDebugSurface` timed out waiting for SERVING. Each package passed immediately when run alone (stand 6.5 s, cmd 9.1 s, udprepl 1.3 s, grpcapi 2.9 s), and the final full verification is green. The harness's pick-then-bind free-port pattern and real-process timing under concurrent load are the likely cause; neither failure touches this plan's code paths.
- **Research arithmetic nuance:** the research's 65 507 figure silently assumed a valued record (tombstone flag absent) and the 4-byte `Body` oneof wrapper. The implementation matches the measured boundary rather than a hand sum, and the probe output is recorded in the deviation above.
- **No auth or setup gates:** none encountered; no `user_setup` entries.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- XPORT-03 is complete at all three seams: the API (`Set`/`SetIfAbsent`/`Delete`), the receive path (03-08's `keyVerdict`) and startup (`checkDatagramCeiling`). All read `storage.MaxKeyBytes`; no second key literal exists.
- 03-10 is unblocked (it depends on 03-09): the node now refuses an unsendable configuration before any bind, so the read-loop/bind-warning work starts from a bounded-envelope baseline.
- 03-12 owns the metrics labels; this plan adds no metric family, only the startup validation path.
- Phase 5 should account for pre-upgrade long-key records at load time: `Get`/`List` remain permissive on purpose, and `docs/OPERATIONS.md` states that such records stay readable but cannot be written again.

## Self-Check: PASSED

- All 9 key files exist on disk (2 created, 7 modified).
- Measured commits from the persisted plan ledger: `git rev-list --count 0fd2f44..34e9a8f` = 3; `plan_head_before` = `0fd2f44d2cba9a9612389cf27448bbe6c7004230`, `plan_head_after` = `34e9a8fe031b679eb0d7d6874d619f26d9c3f44a`.
- `gsd_run` is unavailable in this executor runtime; the close-out uses the persisted plan ledger, direct STATE/ROADMAP edits and `git rev-list --count` as the documented fallback.
- Task verifies re-run at close-out: `TestMaxRecordDatagramBytes` (plus the accepted-domain property), `TestServer_keyLimit`, `TestMain_valueMaxDatagram` (all four boundary cases) and `TestStand_oversizeKeyF2c` green.
- Plan verification re-run: `go test -race -count=1 ./internal/udprepl/ ./internal/grpcapi/ ./cmd/sup-kv/ ./internal/stand/` green when each package is run (the only combined-run failures were the two untouched parallel-load flakes documented above); `go tool golangci-lint run ./...` green.
- Acceptance re-check: API rejects >1 024 with InvalidArgument and accepts 1 024; startup exits 2 above 65 507 and starts at 64 000 and 64 408; the accepted-domain property holds through the real codec; F2c's long key never replicates and the boundary record does; OPERATIONS documents the limits; S-DM-03 is closed; both P-01 TSV rows remain `red` and the 64 000 flag usage remains in `stand_red_test.go`.

---
*Phase: 03-transport*
*Completed: 2026-11-22*
