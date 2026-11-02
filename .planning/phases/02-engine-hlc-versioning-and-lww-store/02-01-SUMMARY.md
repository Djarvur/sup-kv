---
phase: 02-engine-hlc-versioning-and-lww-store
plan: 01
subsystem: database
tags: [go, hlc, hybrid-logical-clock, uuidv7, rfc-9562, crdt, lww-element-map, tombstone, stdlib-only]

requires:
  - phase: 01-ci-foundations
    provides: strict golangci-lint gate with the sup-kv-app-no-io depguard boundary, go.mod tool pins
provides:
  - RecordVersion UUIDv7 pack/unpack/byte-compare (RFC 9562) in internal/app
  - Clock: HLC Tick/Update with the injected now seam, counter-overflow l bump, signed drift signal
  - Store: LWW-Element-Map with Get/Set/SetIfAbsent/Delete/Merge/Records, tombstones, 60 KiB limit, CRC32 record checksum
  - Sentinel errors ErrNotFound, ErrAlreadyExists, ErrValueTooLarge for the Phase 5 RPC mapping
affects: [03-persistence, 04-replication, 05-api, 02-02, 02-03]

actuals:
  tokens: 10117
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - constructor-injected now seam as the single wall-clock entry point (D-02)
    - merge applies externally versioned records; losers returned as []Conflict values (D-01)
    - write lock held across each clock advance so version order equals commit order

key-files:
  created:
    - internal/app/errors.go
    - internal/app/version.go
    - internal/app/clock.go
    - internal/app/store_test.go
  modified:
    - internal/app/store.go
    - internal/app/doc.go
    - internal/app/app_test.go

key-decisions:
  - "Cold start: NewClock seeds l from initialL only (0 defers to the wall clock on the first Tick), so the first write stamps counter 0"
  - "Store write paths hold the write lock across Tick/Update, so version order equals commit order under concurrency"
  - "Clock.Update masks the remote counter to 12 bits, making Merge total over any record content"
  - "Merge installs remote records with a recomputed checksum instead of trusting the wire field (identical for honest records)"
  - "RecordChecksum flag byte via tombstoneFlag.checksumByte method — code-direction lint fix, zero nolint in internal/app"

patterns-established:
  - "External app_test package, t.Parallel on test and subtest, name-field tables, got/want errors, fixedNow(ms) fake clock with no monotonic reading"
  - "RED-GREEN per task with go test evidence classified by gsd-tools before each GREEN commit"

requirements-completed: [CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, CORE-07]

coverage:
  - id: D1
    description: "Local write path wired end-to-end: now seam, HLC Tick, UUIDv7 pack with own NodeID, Set/Get round-trip"
    requirement: CORE-02
    verification:
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_endToEnd"
        status: pass
    human_judgment: false
  - id: D2
    description: "Tombstone visibility matrix: Get NotFound on missing and tombstoned, SetIfAbsent AlreadyExists on any record including tombstones, Delete fresh-version tombstone with no no-op version burn"
    requirement: CORE-01
    verification:
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_Get"
        status: pass
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_SetIfAbsent"
        status: pass
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_Delete"
        status: pass
    human_judgment: false
  - id: D3
    description: "Value limit enforced at 61440/61441 on all three write paths before any copy; zero-length values legal"
    requirement: CORE-05
    verification:
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_valueLimit"
        status: pass
    human_judgment: false
  - id: D4
    description: "Merge keeps remote versions, displaces only on strictly greater full 16-byte versions, returns every displaced or rejected version exactly once, never on equal"
    requirement: CORE-07
    verification:
      - kind: unit
        ref: "internal/app/store_test.go#TestMerge_losers"
        status: pass
    human_judgment: false
  - id: D5
    description: "CORE-02 Set probes: strictly greater replace, same-millisecond counter disambiguation, zero-length value, fresh version per repeat write"
    requirement: CORE-02
    verification:
      - kind: unit
        ref: "internal/app/store_test.go#TestStore_Set"
        status: pass
    human_judgment: false
  - id: D6
    description: "Package contract doc stating LWW laws, tombstone rules, drift/conflict return contracts, ownership, concurrency; skeleton test removed; full gates green with go.mod untouched"
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... && go test -race ./... — exit 0"
        status: pass
      - kind: other
        ref: "go tool golangci-lint config verify && go tool golangci-lint run ./... — exit 0, zero nolint"
        status: pass
      - kind: other
        ref: "git diff --quiet go.mod go.sum — exit 0"
        status: pass
    human_judgment: false

duration: 31min
completed: 2026-11-02
status: complete
commits: 6
plan_head_before: 072b7822268c3a5703891536fd047dc265c04951
plan_head_after: 43e28fc56593901316b430b3478904b0bc741ae2
---

# Phase 02 Plan 01: HLC Versioning and LWW Store Summary

**Pure-Go engine core in internal/app: HLC clock with the injected now seam, RFC 9562 UUIDv7 record versions, and an LWW-Element-Map store with asymmetric tombstone visibility, 60 KiB limit, and merge losers returned by value (D-01, D-02).**

## Performance

- **Duration:** 31 min
- **Started:** 2026-11-02T00:17:26Z
- **Completed:** 2026-11-02T00:48:02Z
- **Tasks:** 3
- **Files modified:** 7 (4 created, 2 rewritten/modified, 1 deleted)

## Accomplishments

- Tracer slice first: one local write flows through now seam → HLC Tick → UUIDv7 pack → Set → Get, green under the race detector
- Full CORE semantics: the asymmetric tombstone-visibility matrix (Get vs SetIfAbsent vs Delete), LWW replace with strictly increasing versions, same-millisecond counter disambiguation, and the 61440/61441 boundary on all three write paths
- Merge joins externally versioned records by the full 16-byte compare, keeps remote versions (no re-stamping), returns every displaced or rejected version exactly once as a Conflict, and never reports a conflict on equal versions
- Package contract doc states every caller-facing law; the Phase 1 skeleton test is gone; strict lint gate green repo-wide with zero suppressions and go.mod/go.sum byte-identical

## Task Commits

Each task was committed atomically (TDD: RED then GREEN per task):

1. **Task 1 RED: failing tracer test** — `9008ea8` (test)
2. **Task 1 GREEN: local write engine slice** — `74f4b81` (feat)
3. **Task 2 RED: failing CORE table tests** — `a05d9ab` (test)
4. **Task 2 GREEN: tombstones, merge losers, setifabsent, records** — `6a19c5e` (feat)
5. **Task 3: package contract doc, skeleton test removal, full gate** — `148fba4` (chore)
6. **Refactor: drop the revive suppression on RecordChecksum** — `43e28fc` (refactor)

## TDD Gate Compliance

Both TDD tasks produced classified RED evidence before each GREEN commit
(`gsd_run check tdd-red-evidence` returned RED_EVIDENCE_OK for Task 1's
TestStore_endToEnd and Task 2's TestStore_Delete); each feat commit descends
from its test commit and the refactor commit keeps all tests green. No gate
violations.

## Files Created/Modified

- `internal/app/errors.go` — sentinel errors ErrNotFound, ErrAlreadyExists, ErrValueTooLarge with the Phase 5 RPC mapping documented
- `internal/app/version.go` — RecordVersion [16]byte, PackVersion/Unpack per RFC 9562 octet layout, byte-wise Compare
- `internal/app/clock.go` — Clock with Kulkarni Tick/Update, counter-overflow l bump, signed drift signal, total over any input
- `internal/app/store.go` — Config, Record, Conflict, RecordChecksum, Store with Get/Set/SetIfAbsent/Delete/Merge/Records
- `internal/app/store_test.go` — TestStore_endToEnd plus the CORE-01..07 table tests and TestMerge_losers (clock-free fixtures)
- `internal/app/doc.go` — rewritten package contract (LWW laws, visibility rules, return-value contracts, ownership, concurrency)
- `internal/app/app_test.go` — deleted; TestAppSkeleton superseded

## Decisions Made

- Cold start: NewClock seeds l from initialL only — 0 defers to the wall clock on the first Tick, which is what makes the first write's counter 0 (matches the plan's Test 1 expectation)
- The store holds its write lock across each Tick/Update so version order equals commit order; the clock keeps its own mutex so both types stay independently safe (Pitfall 7)
- Update masks the remote counter to the 12-bit domain so Merge is total over any record content (threat-model boundary: future replication feeds arbitrary versions)
- Merge recomputes RecordChecksum on install rather than trusting the wire field; identical bytes for honest records, self-validating otherwise
- RecordChecksum's flag byte comes from tombstoneFlag.checksumByte (a value-type method) — the code-direction answer to revive flag-parameter; zero nolint directives in internal/app

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The TDD evidence classifier initially rejected the first RED record
because exitCode was passed as a string rather than a number; re-recording
with the numeric field fixed it (process fix, no code impact).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Engine core complete for plans 02-02 (property tests: hand-rolled seeded generators drive Clock/RecordVersion/merge laws) and 02-03 (structural gates)
- Phase 3 can serialize Record (exported fields) and validate RecordChecksum on dump load; Records() is the dump source
- Phase 4 merges remote records and journals the returned Conflict losers; Phase 5 maps the three sentinels to NotFound/AlreadyExists/InvalidArgument

## Self-Check: PASSED

All six engine files verified on disk (and app_test.go verified absent); all
six task commits verified as ancestors of HEAD; full gates (build, vet,
-race tests, lint config verify + run) re-run green at plan close.

---
*Phase: 02-engine-hlc-versioning-and-lww-store*
*Completed: 2026-11-02*
