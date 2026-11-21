---
phase: 02-stand-and-simulator-as-tests
plan: 16
subsystem: testing
tags: [persist-open, clock-seeding, watermark, vers-04, d07, test-07, composition-path, restart]

# Dependency graph
requires:
  - phase: 02-04 (wave 2 dependency, same phase)
    provides: the fsops refactor of internal/persistfile — the Save path this composition's Set confirmation rides, verified behavior-identical to the pre-seam implementation
  - phase: 03-persistence (v1.0 shipped)
    provides: persist.Open's startup composition (Load → Decode → Watermark → clock.NewClock → mergeRecords), persist.Watermark's FULL (l, c) watermark, FileStore's fallback chain, and the storage.Store Merge/Update path
provides:
  - internal/persistfile/open_seed_test.go — TestOpen_clockSeededFromDumpWatermark, the five-row TEST-07 proof on the REAL composition path (FileStore on a temp dir + real codec + persist.Open; no fake dump port)
  - the D-07 injected-clock contract pinned end to end: an injected now earlier than the dump watermark still yields a first local write that strictly dominates every loaded version
  - executable boundary evidence for the seeding arithmetic: now exactly equal to the watermark l, now below it, empty start, equal-max-l records, and restart idempotence
affects: [02-17 (remaining phase-2 plan), phase 5 persistence plans (restart/clock fixes get this composition gate), /gsd:verify-work's evaluation of TEST-07]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 2177    # chars/4 over the realized diff (8707-byte file, 258 insertions)
  tasks: 1
  commits: 1      # MEASURED from the plan-start ledger at SUMMARY write (#3968): git rev-list --count e7bfb54..497b5df
  plan_head_before: e7bfb54cb1046fff0ba00324b7bc405ab21e448e
  plan_head_after: 497b5df68a69ca28c06e9cfca76ceaa0c5b5f36a

# Tech tracking
tech-stack:
  added: none — the test file imports stdlib (bytes/context/errors/path/filepath/testing/time) and own-module packages only; zero new module dependencies (zero-install phase per the Package Legitimacy Audit)
  patterns:
    - production-encoder fixtures: dumped records are built with version.PackVersion + storage.RecordChecksum and written through persist.Encode, so the real decoder's checksum oracle accepts them — never hand-written JSON that bypasses the codec
    - derived clock expectations: the expected first-write reading is computed from the full composition chain (NewClock seed → each load-time Merge's clock.Update → the first Tick), so the row fails if Open stops feeding the watermark through the store's merge path
    - full-version dominance: the local write is compared with Version.Compare against the records the returned engine actually loaded (Records()), not against the fixture, and not by l alone
    - real-dumper confirmation: a local Run goroutine with a 1 ms debounce drives Set to its documented confirmation; no synctest bubble, because the composition is real file I/O and real syscalls are not durably blocking in a bubble
    - external-package reuse: persistfile_test composes only exported API and reuses dumpfile_test.go's writeFile helper instead of duplicating it

key-files:
  created:
    - internal/persistfile/open_seed_test.go
  modified: []

key-decisions:
  - "The test file is package persistfile_test (external), not the plan's suggested package persistfile: the composition needs only exported API, and the lint config's testpackage skip-regexp covers only *_internal_test.go — the internal form would require a //nolint:testpackage suppression with no functional justification (AGENTS.md deviation 3). The external form also reuses dumpfile_test.go's writeFile helper. This mirrors 02-15's simnet_test decision."
  - "Expected counters pin the whole documented chain, not just 'strictly greater': NewClock seeds (L, C), every load-time mergeRecords → Store.Merge → clock.Update ticks the counter once per at-watermark record, and the first Tick increments again — yielding (aheadL,2), (now,9), (now,0) empty start, and (aheadL,8) for two equal-l records. The exact values are what would catch a seeding regression that dropped initialC while keeping initialL."
  - "Dominance is asserted on the full 16-byte version via Compare against eng.Records() (what the real codec + store loaded), so the proof covers nodeID ordering too and cannot pass vacuously on fixtures the decoder discarded."
  - "The five rows are named subtests of one table-driven test, each in its own t.TempDir() FileStore — the restart-idempotence row Opens the same directory twice and compares deep record equality without writing."
  - "No production code changed: the proof passes against current Open, so TEST-07 is green rather than a finding; Phase 2 writes tests only, and a failing row would have been recorded with its output instead of a fix."

patterns-established:
  - "Composition-path proofs craft their fixtures through the production encoder/checksum so a decoder regression surfaces as a fixture-load failure, never as a silently empty start the dominance assertion would still satisfy (the loaded-record equality assertion closes that hole)"
  - "Clock-seeding expectations are written as seed → merge-Update → first-Tick arithmetic in comments next to the expected pair, keeping the assertion and the documented mechanism one artifact"

requirements-completed: [TEST-07]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Dominates row — a crafted dump whose record l is one hour greater than the injected now; Open seeds the clock from that watermark and the first local write dominates the loaded version (strictly greater l, counter 2 after the seed→merge→tick chain)."
    requirement: TEST-07
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestOpen_clockSeededFromDumpWatermark$' (subtest: dump_watermark_ahead_of_the_injected_now_is_dominated)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Boundary-equal row — the dumped max l exactly equals the injected now; Open seeds at the watermark and the first local write is still strictly greater (counter+1: 8 after merge, 9 after the first Tick), never equal."
    requirement: TEST-07
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestOpen_clockSeededFromDumpWatermark$' (subtest: injected_now_exactly_at_the_watermark_l_still_yields_a_greater_write)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Empty-start row — no dump file at all: Open loads an empty store (records equality against the nil fixture) and the first local write is stamped from the injected now alone (now, 0)."
    requirement: TEST-07
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestOpen_clockSeededFromDumpWatermark$' (subtest: empty_start_seeds_from_the_injected_now_alone)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Equal-max-l row — two dumped records with different keys share the max l and differ in counter; Watermark takes that (l, c) pair, the first local write dominates both loaded versions (counter 8), proving the FULL watermark rather than max l alone."
    requirement: TEST-07
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestOpen_clockSeededFromDumpWatermark$' (subtest: two_records_sharing_the_max_l_take_that_counter_pair)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Restart-idempotence row — reopening the same dump directory a second time loads deep-equal records (versions, data, deleted flags, checksums), with zero discarded records on both Opens."
    requirement: TEST-07
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestOpen_clockSeededFromDumpWatermark$' (subtest: reopening_the_same_dump_loads_identical_records)"
        status: pass
    human_judgment: false

# Metrics
duration: 169s
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 16: persist.Open Clock Seeding on the Real Composition Path Summary

**Five-row table-driven proof that `persist.Open` seeds the HLC from the dump's FULL watermark — a real `FileStore` on a temp dir composed with the real codec, no fake dump port — so a first local write dominates every loaded version even when the injected `now` lies an hour behind the dump.**

## Performance

- **Duration:** 169s
- **Tasks:** 1
- **Files created:** 1 (258 lines)
- **Commits:** 1

## Accomplishments

- Added `TestOpen_clockSeededFromDumpWatermark` in `internal/persistfile/open_seed_test.go`, composing `persistfile.New(temp dir)` with `persist.Open(nodeID, now, driftWarn, realStore, ...)` and driving the first local write through the returned engine to its dump-confirmed return.
- Proved the five TEST-07 named rows green against current code: dominates (watermark ahead of injected now), boundary-equal (now exactly at watermark l), empty start (injected now alone), equal-max-l (watermark keeps the counter pair), restart idempotence (two Opens load identical records).
- Pinned the exact seeding arithmetic per row: seed → load-time `Merge`/`clock.Update` tick → first `Tick`, so a regression in either watermark half (l or c) or in the merge feeding fails the row with the reading in the message.
- Crafted dump fixtures through `persist.Encode` + `storage.RecordChecksum`, so the decoder's real checksum oracle is exercised and `Open` reports `discarded == 0`; loaded records are asserted equal to the fixture before the write, closing the "silently empty start" hole.

## Task Commits

1. **Task 1: persist.Open clock seeding on the real composition path** — `497b5df` (test)

## Files Created/Modified

- `internal/persistfile/open_seed_test.go` — created: the five-row TEST-07 composition test plus its fixture, dump-writer, dumper, `now` and record-equality helpers.

## Decisions Made

- External test package `persistfile_test` rather than the plan's `package persistfile`: no unexported access is needed, the internal form would need a `//nolint:testpackage` without functional justification, and the external form reuses `dumpfile_test.go`'s `writeFile` helper (same reasoning as 02-15's `simnet_test` decision).
- Expected clock readings are the full chain product (seed → one Update tick per at-watermark load-time merge → first Tick), not merely "strictly greater": the exact pairs are the regression probe for the counter half of the watermark.
- Dominance asserted with the full 16-byte `Version.Compare` against `eng.Records()` — the records the real codec and store loaded — rather than against the fixture or on `l` alone.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] External test package instead of `package persistfile`**
- **Found during:** Task 1 (file scaffold)
- **Issue:** The plan's task action names `package persistfile`, but the lint config's `testpackage` linter only exempts `*_internal_test.go` files; an internal `open_seed_test.go` would need a `//nolint:testpackage` with no functional justification (nothing unexported is touched), violating AGENTS.md's no-unjustified-suppression rule.
- **Fix:** Declared `package persistfile_test` and composed only exported API (`persistfile.New`, `persist.Open`, engine methods), reusing the package's existing `writeFile` helper.
- **Files modified:** `internal/persistfile/open_seed_test.go`
- **Verification:** `go tool golangci-lint run ./internal/persistfile/...` clean; `make lint` clean.
- **Commit:** `497b5df`

**Total deviations:** 1 auto-fixed (Rule 3)
**Impact:** None on behavior or coverage — the test still composes the real FileStore + codec + Open; only the test package clause differs.

## Issues Encountered

None. The proof passed against current code on the first full run; no production change was needed (Phase 2 writes tests only).

## User Setup Required

None.

## Next Phase Readiness

- TEST-07 is executable evidence, not just a plan claim: any future `Open`/`Watermark`/`NewClock` change that breaks restart dominance fails with the exact reading pair in the message.
- The file is ready for `/gsd:verify-work`'s TEST-07 row and for Phase 5's restart/clock fixes as a pre-existing green gate.
- Remaining phase-2 plan: 02-17.

## Self-Check

- FOUND: `internal/persistfile/open_seed_test.go`
- FOUND: commit `497b5df`
- `go test -race -count=1 ./internal/persistfile/ -run '^TestOpen_clockSeeded'` → PASS (all five subtests, none skipped)
- `go test -race -count=1 ./internal/persistfile/` → PASS
- `make lint` → PASS
- Ledger: `plan_head_before e7bfb54..HEAD` = 1 commit, matching the task commit

## Self-Check: PASSED
