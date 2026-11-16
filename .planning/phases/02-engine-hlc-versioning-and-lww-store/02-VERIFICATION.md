---
phase: 02-engine-hlc-versioning-and-lww-store
verified: 2026-11-16T11:04:05Z
status: passed
score: 40/40 must-haves verified
covered_files:
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-01-PLAN.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-01-SUMMARY.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-02-PLAN.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-02-SUMMARY.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-03-PLAN.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-03-SUMMARY.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-04-PLAN.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-04-SUMMARY.md
  - go.mod
  - go.sum
  - internal/app/clock/clock.go
  - internal/app/clock/clock_test.go
  - internal/app/clock/hlc_prop_test.go
  - internal/app/internal/testutil/gen.go
  - internal/app/internal/testutil/rng.go
  - internal/app/storage/doc.go
  - internal/app/storage/errors.go
  - internal/app/storage/merge_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/app/version/version.go
  - internal/app/version/version_test.go
covered_digest: "v4:sha256:ce0e0a4f77a21870af1823db8fafe9b72fac292fd6f603cb0a9d7aa433fcced0"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 40/40
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Report regeneration #6. The prior pass (2026-11-14T19:03:16Z, at f5f9dac) was passed with no open gaps; this report was triggered because covered inputs changed after it: go.mod gained the toolchain go1.27.4 directive (+2 lines, the staleness trigger, 2026-11-15), and commit ba0d984 (phase 6-02, D-15) landed the phase-2 review-disposition closure — IN-01/IN-02 OnChange doc-comment fixes in internal/app/storage/store.go (+11/-3, comment-only: stripping // comments leaves the code byte-identical, proven by strip-diff this pass) and the IN-03 test dedup in store_test.go (the TestStore_Set 'second set replaces' subtest removed; the observable stays pinned in TestStore_endToEnd, re-read this pass). go.sum, .golangci.yml, and every clock/version/testutil file are byte-identical to the prior verified state (git diff f5f9dac..HEAD over the covered paths moves only the three files named). The complete phase-2 suite re-ran green under -race in this verification's own process at 100.0% statement coverage per package; the depguard purity boundary re-probed in-process (denial captured). No regressions found."
deferred:
  - truth: "VERS-05's 'drift beyond drift_warn is logged as a warning' — the logging half (the engine-side signed drift signal is complete, property-proven, and overflow-safe since WR-01)"
    addressed_in: "Phase 6"
    evidence: "REQUIREMENTS.md OBS-01: 'A metrics endpoint exposes ... clock-drift warnings' and OBS-03: structured logging — both now marked Complete in the traceability table (Phase 6 delivered the slog pipeline and the metrics endpoint with the drift-warning gauge); D-01 forbids logging inside internal/app by design, so the logging consumer necessarily lives outside the phase-2 scope. The deferral has been discharged by Phase 6's own verified work."
---

# Phase 2: Engine — HLC Versioning and LWW Store Verification Report

**Phase Goal:** The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Verified:** 2026-11-16T11:04:05Z (at HEAD 9b87620)
**Status:** passed
**Re-verification:** Yes — report regeneration #6 (prior passed report 2026-11-14T19:03:16Z at f5f9dac; no gaps were open then; this pass refreshes the covered-input fingerprint and re-verifies the changed sites)

## Change Delta Since the Prior Passed Report

Exactly three covered inputs moved after f5f9dac (`git diff --stat f5f9dac..HEAD` over the covered paths):

1. **`go.mod` (+2 lines)** — the `toolchain go1.27.4` directive (the staleness trigger). Zero dependency movement: `go.sum` is byte-identical to f5f9dac, and the direct `require` block is still exactly `google.golang.org/grpc v1.85.0` + `google.golang.org/protobuf v1.36.13-…` — **no property-test or UUID library entered the build** (the phase-2 D-04 substance; see Prohibition Adjudication).
2. **`internal/app/storage/store.go` (+11/−3) — comment-only, proven.** Commit ba0d984 (phase 6-02, decision D-15) closed review findings IN-01/IN-02 inside the `Config.OnChange` doc comment: the tap's handed record is now documented as "the exact record a subsequent Records() snapshot holds … and the record a subsequent Get returns only when it is live" with the tombstone clarification (a delete IS a change; Get answers ErrNotFound; tombstones visible only through Records()), and the re-entrancy boundary is stated (the tap runs with the write lock held, Go's sync.RWMutex is not reentrant, even a read-only Get self-deadlocks). **Strip-diff proof:** removing `//`-comment text from both revisions leaves the code byte-identical (the only residual delta is 8 blank lines where the doc block grew) — no executable line changed, so every prior behavioral proof of the store carries over unmodified.
3. **`internal/app/storage/store_test.go` (net −19 lines) — the IN-03 dedup.** The `TestStore_Set` "a second set replaces the value under a strictly greater version" subtest was removed as a duplicate; the `TestStore_endToEnd` copy (store_test.go:106-130) is retained and re-read this pass: it performs a first and second Set, asserts `second.Compare(first) == cmpNewer` and the counter advance, and reads back "two" via Get — the CORE-02 replace observable remains pinned exactly once. Test-function inventory is unchanged at 12 (a subtest was deduplicated, not a test function removed).

The phase-2 review dispositions are now all `fixed` in 02-REVIEW-DISPOSITION.md (IN-01/IN-02/IN-03, plan 06-02 / D-15; open: 0). One letter-level imprecision in that disposition record, noted for accuracy under Anti-Patterns: the IN-03 row says the "now-unused cmpSame constant" was removed with the subtest — in fact `cmpSame` (store_test.go:39) remains and must remain, because merge_test.go:156,247 (same package) still uses it; the code state is correct and compiles green.

`.golangci.yml`, `go.sum`, and every clock/version/testutil covered file are byte-identical to the prior verified state.

## Goal Achievement

The goal holds at HEAD 9b87620. `internal/app` remains a pure, I/O-free library: the depguard boundary `sup-kv-app-no-io` was re-probed in this verification's own process — a scratch file in `internal/app/storage` carrying `import "os"` fails lint with `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)` (captured verbatim this pass; file deleted, tree clean). Direct imports of the three phase-2 packages are stdlib + own module only (`go list -f '{{.Imports}}'`: version → bytes, encoding/binary; clock → math, sync, time; storage → bytes, errors, hash/crc32, sync, time, own version package). Purity greps find zero logging, zero `fmt.Print`, zero production `time.Now()` in the phase-2 production files, and zero blank imports. The HLC packs into RFC 9562 UUIDv7, the LWW-Element-Map store implements the full CORE semantics with exact loser accounting, and the correctness laws are property-proven by the seeded, replayable suite — **the complete phase-2 suite (every test in clock, version, storage, no filter) ran green under the race detector at 100.0% statement coverage per package, first-hand this pass.**

### Observable Truths

Plan 02-01 (17 truths — all VERIFIED; storage is a changed covered package this pass: the complete `TestStore_*`/`TestMerge_*` surface re-ran green under `-race`, and the two changed files were read in full):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Local write flows end-to-end: Set stamps fresh HLC version in UUIDv7 carrying NodeID; Get returns exact bytes + version | ✓ VERIFIED | `TestStore_endToEnd` green under `-race` this pass (subtests re-read: round-trip, UUIDv7 marker bits, replace-under-greater-version) |
| 2 | Get returns ErrNotFound for missing and tombstoned keys (CORE-01) | ✓ VERIFIED | `TestStore_Get` subtests; store.go:141 sentinel return |
| 3 | Set replaces prior value under LWW, greater version wins (CORE-02) | ✓ VERIFIED | **Evidence home moved by the IN-03 dedup:** `TestStore_endToEnd` "a second set replaces the value under a strictly greater version" (store_test.go:106) — second Set, `Compare` strictly greater, counter advance, Get returns "two"; re-read this pass, green under `-race`. The duplicate copy in `TestStore_Set` was removed precisely because this pinning exists |
| 4 | SetIfAbsent returns ErrAlreadyExists for any existing record incl. tombstone (CORE-03) | ✓ VERIFIED | `TestStore_SetIfAbsent` subtests incl. tombstoned-no-recreation |
| 5 | Delete writes fresh-version tombstone that stays in map and merges like any write (CORE-04) | ✓ VERIFIED | `TestStore_Delete`; `Records()` keeps tombstones; `TestMerge_lawsTombstones` |
| 6 | Values over ValueMaxBytes rejected with ErrValueTooLarge before any copy (CORE-05) | ✓ VERIFIED | `TestStore_valueLimit` 61440/61441 on set, setifabsent and merge; store.go:160-161/188-189/220-221 checks precede every copy — re-located on the current source this pass (code byte-identical, only line numbers shifted by the doc growth) |
| 7 | Merge keeps remote version, displaces only on strictly greater full 16-byte version, returns every displaced/rejected version as Conflict (CORE-06/07) | ✓ VERIFIED | store.go join() `rec.Version.Compare(stored.Version)` on the full 16 bytes (store.go:240); `TestMerge_losers` + `TestMerge_losersProperty` green this pass |
| 8 | Two writes in same ms get distinct strictly increasing versions | ✓ VERIFIED | `TestStore_Set` same-ms subtest (store_test.go:229) — retained by the dedup |
| 9 | Zero-length value is legal | ✓ VERIFIED | `TestStore_Set` zero-length subtest (store_test.go:252) — retained |
| 10 | Repeat writes never error, fresh version each time | ✓ VERIFIED | `TestStore_Set` repeat subtest (store_test.go:267) — retained |
| 11 | SetIfAbsent twice: first creates, second ErrAlreadyExists, first value kept | ✓ VERIFIED | `TestStore_SetIfAbsent` |
| 12 | Delete on missing/tombstoned key: ErrNotFound, no new tombstone, no version burned | ✓ VERIFIED | `TestStore_Delete` no-burn subtests |
| 13 | Equal-version merge is a full no-op, no Conflict | ✓ VERIFIED | join() default branch returns nil; `TestMerge_losers` + `TestMerge_duplicateDelivery`; equal-version redelivery also locked tap-silent by `TestStore_onChange` |
| 14 | Merge into missing key installs, no loser reported | ✓ VERIFIED | join() install case (store.go:237); `TestMerge_losers` |
| 15 | No Conflict ever carries loser == winner | ✓ VERIFIED | `TestMerge_losersProperty` explicit assertion |
| 16 | Displacing-nothing merge returns empty conflict list, not error | ✓ VERIFIED | join() nil return; asserted across merge suite |
| 17 | Sentinel errors matched with errors.Is | ✓ VERIFIED | storage/errors.go 3 sentinels; tests import and use errors.Is |

Plan 02-02 (11 truths — all VERIFIED; clock/version/testutil files byte-identical to the prior verified state; the complete unfiltered package suites re-ran green under `-race` this pass via the coverage run, plus named regression runs):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Tick/Update follow Kulkarni rules, (l, counter) strictly increases on every call (VERS-03) | ✓ VERIFIED | `TestClock_tick`/`TestClock_update`/`TestClock_monotonic` — named regression run PASS this pass |
| 2 | Byte-wise compare equals (l, counter, NodeID) order vs field-wise reference model (VERS-01) | ✓ VERIFIED | `TestVersion_byteOrder` + `TestVersion_layout`; full version package `-race` green this pass |
| 3 | 10,000 Ticks under stuck clock → 10,000 distinct strictly increasing versions (VERS-02) | ✓ VERIFIED | `TestVersion_uniquePerWrite`; ceiling-enforced and watermark-clamp-enforced end to end |
| 4 | Drift signal fires both directions with sign; always returned, never recorded (VERS-05, D-01) | ✓ VERIFIED | `TestClock_driftSignal` + `TestClock_driftUnderChaos` + `TestClock_driftSignalSaturates` — named regression run PASS this pass; zero logging in internal/app |
| 5 | Clock constructed with the dump's FULL watermark emits only versions strictly greater than EVERY dump version (VERS-04) | ✓ VERIFIED | `TestClock_restartProperty` + `TestClock_restartFromDumpMax` — both in the named regression run PASS this pass |
| 6 | Concurrent Tick/Update: parallel mix, zero race findings | ✓ VERIFIED | `TestClock_monotonic` concurrent subtest; full clock package `-race` green this pass |
| 7 | Same remote version twice never decreases (l, c) — second Update no-op | ✓ VERIFIED | `TestClock_update` equal-timestamp rows |
| 8 | InitialL ahead of wall clock → first versions at/above InitialL; zero starts from wall clock | ✓ VERIFIED | `TestClock_restartFromDumpMax` first two subtests |
| 9 | Restart exact to the ms: every subsequent version strictly greater than every dump version, even under backward-jumped clock (VERS-04 precision) | ✓ VERIFIED | `TestStore_restartDominance` 4 subtests — green this pass in the full storage surface run |
| 10 | Stuck clock: Ticks 1-4096 fill counters 0-4095, Tick 4097 → (l+1, 0); never wrap/repeat (TEST-02 boundary) | ✓ VERIFIED | `TestClock_counterOverflowBoundary` + `TestClock_ceilingSaturation` + `TestClock_watermarkClampedIntoDomain` — package suite green this pass |
| 11 | HLC state is plain integer arithmetic — no time.Time held as state, no monotonic-read comparisons (Pitfall 8, backstop) | ✓ VERIFIED | Re-observed first-hand this pass on the byte-stable source: `Clock` fields are `l int64`, `c uint16`, `now func() time.Time`, `driftWarn time.Duration` (clock.go:43-49); Tick/Update reduce the seam to `UnixMilli()` int64 immediately (clock.go:90, 123) |

Plan 02-03 (11 truths — all VERIFIED; storage is a changed covered package: complete named-test surface green under `-race` this pass):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Merge commutative, associative, idempotent incl. tombstones over seeded random stores (TEST-01, CORE-06) | ✓ VERIFIED | `TestMerge_laws`/`TestMerge_lawsTombstones` — 500 seeded triples per mix, green this pass |
| 2 | Every displaced/rejected version returned exactly once as Conflict paired with displacer (CORE-07) | ✓ VERIFIED | `TestMerge_losersProperty` both directions |
| 3 | Delivery order, duplicates, retransmissions cannot affect outcome | ✓ VERIFIED | Commutativity + `TestMerge_duplicateDelivery` |
| 4 | Parallel Set/SetIfAbsent/Delete/Merge/Get: zero race findings | ✓ VERIFIED | `TestStore_concurrentHammer`; full storage package `-race` green this pass |
| 5 | Concurrent SetIfAbsent on one key: exactly one succeeds | ✓ VERIFIED | `TestStore_concurrentSetIfAbsent` |
| 6 | Delete racing Set: survivor holds exactly the greater captured version | ✓ VERIFIED | `TestStore_concurrentDeleteVsSet` |
| 7 | merge(a,b) and merge(b,a): identical state AND identical conflict sets | ✓ VERIFIED | `assertMergeLaws` commutativity block |
| 8 | Every displaced version exactly once in conflicts, paired with displacer | ✓ VERIFIED | `TestMerge_losersProperty` exact expected set |
| 9 | merge(a,a) idempotent: zero conflicts, nothing changes | ✓ VERIFIED | `assertMergeLaws` idempotence block |
| 10 | Laws hold for empty stores | ✓ VERIFIED | empty-source clones + missing-key install cases |
| 11 | merge(merge(a,b),c) equals merge(a,merge(b,c)) incl. tombstoned keys | ✓ VERIFIED | `assertMergeLaws` associativity block |

Plan 02-04 (3 gap-closure truths — truths 1-2 restate the historically falsified 02-02 pair and are deduplicated into the count via those rows; truth 3 re-verified on the byte-stable clock source and the read test file):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Clock constructed with the dump's full watermark emits only versions strictly greater than every dump version, incl. counter >= 1 at the watermark ms and the own NodeID (dedup of 02-02 #5) | ✓ VERIFIED | `TestClock_restartProperty` + `TestClock_restartFromDumpMax` named runs PASS this pass |
| 2 | First post-restart write never collides with or subordinates to the dump; peer accepts; stale watermark displaces with exactly one Conflict (dedup of 02-02 #9) | ✓ VERIFIED | `TestStore_restartDominance` green this pass in the full storage surface run — 4 subtests with value-level `Unpack()`/`Compare` assertions, under `-race` |
| 3 | The restart seam expresses a counter seed reaching the clock, so the dump's full watermark is carryable; the now seam remains the single wall-clock input — the seed is constructor state, not a clock read (D-02) | ✓ VERIFIED | `clock.NewClock(now, initialL, initialC, driftWarn)` (clock.go:79, signature re-observed this pass) — the seed is constructor state masked/clamped on entry, never a clock read; `now` is still the single wall-clock seam. Composition route exercised by `TestStore_restartDominance` and the test helper `newStoreFor` (store_test.go: `clock.NewClock(fixedNow(), 0, 0, testDriftWarn)` → `storage.NewStore`), both green this pass. REQUIREMENTS.md VERS-04 documents exactly this route (the plan's literal `Config.InitialL`/`InitialC` wording is the stale pre-refactor spelling; the superseding route is source-verified) |

**Score:** 40/40 truths verified (0 present, behavior-unverified). Denominator unchanged from the prior passed report: the 39 truths of plans 02-01..02-03 (17 + 11 + 11) plus 02-04's seam-expressiveness truth; 02-04's other two truths are deduplicated restatements of the 02-02 pair.

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | VERS-05's "logged as a warning" half — the engine returns the signed, overflow-safe drift signal (complete, property-proven); actual logging/metrics wiring | Phase 6 | REQUIREMENTS.md OBS-01 (clock-drift warnings in the metrics endpoint) and OBS-03 (structured logging) — both now marked Complete in the traceability table: Phase 6 delivered the slog pipeline and the metrics endpoint. The deferral is discharged; D-01 keeps the logging consumer outside internal/app by design |

### Advisory (New Scope, Unevidenced)

Re-verification ran (report regeneration #6). No new-scope unevidenced findings this pass: the only covered-code changes are the comment-only IN-01/IN-02 doc fixes and the IN-03 dedup (a documented phase-6 decision, 06-02/D-15) fully evidenced by the green suite, plus the go.mod toolchain line (no dependency movement — direct requires re-checked). The one observation raised (disposition-record prose imprecision re `cmpSame`) is recorded under Anti-Patterns as info; it alleges no code defect and has none — the code state is correct and compiling.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Required Artifacts

Every plan declares paths under the old single-package layout; the artifacts are verified at their post-split homes (the split predates the prior passed reports). Note: `gsd-tools verify.artifacts` reports 0/6 on 02-01 because the PLAN's literal paths (`internal/app/store.go` etc.) no longer exist — the same stale-path adjudication as the prior passed reports; existence, substance and wiring below are verified manually at the current homes, with the two changed files read in full this pass.

| Plan path (stale) | New home | Expected | Status | Details |
| -------- | -------- | -------- | ------ | ------- |
| internal/app/errors.go | internal/app/storage/errors.go | Sentinels matched with errors.Is, RPC mapping docs | ✓ VERIFIED | Unchanged since prior verified state (byte-stable) |
| internal/app/version.go | internal/app/version/version.go | RecordVersion: pack, total unpack, byte-wise compare | ✓ VERIFIED | Byte-identical to prior verified state |
| internal/app/clock.go | internal/app/clock/clock.go | HLC with now seam, overflow rule, initial-l seam, drift signal | ✓ VERIFIED | Byte-identical to prior verified state; struct + NewClock signature re-observed this pass |
| internal/app/store.go | internal/app/storage/store.go | Config, Record, Conflict, RecordChecksum, Store API | ✓ VERIFIED | 287 lines (was 279; +8 net, all doc-comment); read in full this pass — code byte-identical after comment-strip, LWW semantics unchanged, all exports intact |
| internal/app/store_test.go | internal/app/storage/store_test.go | Tracer + CORE tables + merge unit tests | ✓ VERIFIED | 1144 lines (was 1163; −19 net from the IN-03 dedup); still 12 test functions; `TestStore_endToEnd` retains the replace-under-LWW pinning; prior coverage intact |
| internal/app/doc.go | internal/app/storage/doc.go | Package contract: LWW laws, visibility, D-01 contracts | ✓ VERIFIED | Unchanged since prior verified state |
| internal/app/clock_test.go | internal/app/clock/clock_test.go | Kulkarni tables, overflow boundary, restart, drift | ✓ VERIFIED | Byte-identical to prior verified state; 8 test functions |
| internal/app/props_test.go | internal/app/internal/testutil/rng.go + gen.go | Seeded generators, printed seed, -seed replay | ✓ VERIFIED | Byte-identical to prior verified state; `NewRNG` prints the seed and honors `-seed` (source re-checked this pass) |
| internal/app/version_test.go | internal/app/version/version_test.go | RFC 9562 vectors, round-trip, byte-order property | ✓ VERIFIED | Byte-identical to prior verified state; 4 test functions |
| internal/app/hlc_prop_test.go | internal/app/clock/hlc_prop_test.go | TEST-02 property suite | ✓ VERIFIED | Byte-identical to prior verified state; 6 test functions incl. `TestClock_restartProperty` |
| internal/app/merge_test.go (02-03) | internal/app/storage/merge_test.go | TEST-01 law suite | ✓ VERIFIED | Byte-identical to prior verified state; 4 test functions |

Wiring re-located on the current source (line numbers shifted by the doc growth, code unchanged): `storage` production calls `s.clock.Tick()` (store.go:176) and `s.clock.Update()` (store.go:227) through the consumer-side `storage.Clock` interface and stamps/decides via `version.PackVersion` (store.go:177) / full-16-byte `Compare` (store.go:240); the OnChange tap fires from `install()` (store.go:268). Storage production imports only `bytes`, `errors`, `hash/crc32`, `sync`, `time`, `version` (go list verified this pass). The store is consumed downstream (persist.Engine; `OnChange` wired end-to-end since phase 5) — not orphaned at HEAD.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/app/storage/store.go (02-01) | clock Tick/Update | `s.clock.Tick()` / `s.clock.Update()` through `storage.Clock` | ✓ WIRED | store.go:176, store.go:227 — re-located on the current source this pass (code byte-identical) |
| internal/app/storage/store.go (02-01) | version Pack/Compare | `version.PackVersion(l, counter, s.nodeID)`; join decides via full-16-byte `Compare` | ✓ WIRED | store.go:177, store.go:240 |
| storage tests (02-01) | exported storage API | external test packages drive only exported symbols with injected fakes | ✓ WIRED | imports confirmed in all test files; `TestStore_onChange` likewise drives only `Config.OnChange` + exported API |
| hlc_prop_test → generators (02-02) | testutil + same-package helpers | `testutil.NewRNG`, `nextWallTime` | ✓ WIRED | unchanged (files byte-stable) |
| version_test → version (02-02) | PackVersion/Unpack/Compare | layout vectors and canonicalization property | ✓ WIRED | unchanged |
| clock_test → clock (02-02) | NewClock via fake now (D-02) | every Kulkarni branch + WR regression paths | ✓ WIRED | unchanged |
| merge_test → testutil + storage (02-03) | RandStore/NewRNG; Merge/Records only | laws drive exported API; remote records keep their versions | ✓ WIRED | join() never re-stamps — code byte-identical this pass |
| store_test → storage (02-03) | concurrent hammer on exported API | `sync.WaitGroup`/goroutines; race detector is the assertion | ✓ WIRED | `TestStore_concurrentHammer` + friends green this pass |
| store.go → clock.go (02-04) | caller composes `clock.NewClock(now, initialL, initialC, driftWarn)` into `NewStore(nodeID, clk, cfg)` | ✓ WIRED | Composition exercised by `TestStore_restartDominance` (green this pass) and the `newStoreFor` helper; the tool's literal `InitialC`-in-Config pattern check reports the old pre-refactor route — the superseding constructor route is source-verified and documented in REQUIREMENTS.md VERS-04 |

### Data-Flow Trace (Level 4)

Not applicable in the render sense — a library with no UI/external data source. The equivalent holds on the current source: every version flows from `clock.Tick/Update` through `version.PackVersion`; the restart seed flows from `NewClock`'s `initialL`/`initialC` into constructor state (never a clock read — D-02 preserved); the drift signal flows out of every clock-advancing op as a `time.Duration` return value (never recorded, never fed back); Merge decides via full 16-byte `Compare` on actual record versions; the OnChange tap hands the stored record computed by `install()` (defensive clone + recomputed checksum — real data, not a hardcoded payload). No hardcoded data paths anywhere in `internal/app`.

### Behavioral Spot-Checks

All commands run in this verification's own process at HEAD 9b87620. Constraint honored: no full-workspace re-run; scoped named-test runs plus one unfiltered phase-2-package coverage run (which itself executes every test in the three packages).

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full storage named-test surface (changed package) | `go test -race -count=1 -run '^(TestStore_\|TestMerge_)' ./internal/app/storage` | `ok` 2.255s — all 12 TestStore_* (incl. `TestStore_onChange` 8 subtests, `TestStore_restartDominance` 4 subtests) + 4 TestMerge_* functions | ✓ PASS |
| Clock named regressions (byte-stable package) | `go test -race -count=1 -run '^(TestClock_restartProperty\|TestClock_monotonic\|TestClock_tick\|TestClock_update\|TestClock_driftSignal\|TestClock_watermarkClampedIntoDomain\|TestClock_driftSignalSaturates\|TestClock_counterOverflowBoundary\|TestClock_restartFromDumpMax\|TestClock_driftUnderChaos)$' ./internal/app/clock` | `ok` 1.023s (superset of the prior pass's named set) | ✓ PASS |
| Version named regressions (byte-stable package) | `go test -race -count=1 -run '^TestVersion_' ./internal/app/version` | `ok` 1.019s | ✓ PASS |
| Complete phase-2 suite with coverage (unfiltered — every test in the three packages) | `go test -race -count=1 -cover ./internal/app/clock/ ./internal/app/version/ ./internal/app/storage/` | ok clock 100.0%, version 100.0%, storage 100.0% of statements — the doc-touched store and deduped test file included | ✓ PASS |
| Strict lint on phase-2 packages | `go tool golangci-lint run ./internal/app/clock/... ./internal/app/version/... ./internal/app/storage/... ./internal/app/internal/testutil/...` | exit 0, zero issues (config `.golangci.yml` byte-identical to the prior verified state) | ✓ PASS |
| Depguard boundary probe (own process) | scratch file with `import "os"` + usage in internal/app/storage, scoped lint, delete | lint fails: `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)` — captured verbatim; tree clean after removal | ✓ PASS |
| Purity greps | forbidden logging / `fmt.Print` / production `time.Now()` / blank imports in the phase-2 production files | none found | ✓ PASS |
| Direct-import purity | `go list -f '{{.Imports}}'` on version/clock/storage | version: bytes, encoding/binary; clock: math, sync, time; storage: bytes, errors, hash/crc32, sync, time, own version — stdlib + own module only | ✓ PASS |
| Comment-only proof for store.go | strip `//` comments from both revisions, diff | only residual: 8 blank lines where the doc block grew — code byte-identical | ✓ PASS |
| Test-inventory continuity | `grep -c '^func Test'` across the five test files | store_test 12, merge_test 4, clock_test 8, hlc_prop 6, version_test 4 — function counts unchanged from the prior verified state (the dedup removed a subtest, not a function) | ✓ PASS |
| Dependency policy (D-04 substance) | direct `require` block in go.mod + `git diff f5f9dac..HEAD -- go.sum` | exactly `google.golang.org/grpc` + `google.golang.org/protobuf`; go.sum byte-identical — the +2 go.mod lines are the toolchain directive only; **no property-test or UUID library in the build** | ✓ PASS |
| Seed replay contract | testutil/rng.go source (byte-stable) | `NewRNG` prints the seed; `-seed` flag registered | ✓ PASS |
| Backstop truth source check | clock.go struct + seam reads | `Clock{mu, l int64, c uint16, now func() time.Time, driftWarn time.Duration}`; Tick/Update call `now().UnixMilli()` immediately | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist or were declared (no scripts directory). The phase's probe obligations are discharged by the named Go tests above plus the depguard boundary probe, executed in this verification's own process with the raw lint failure captured.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CORE-01 | 02-01 | Get: missing/tombstoned → NotFound | ✓ SATISFIED | TestStore_Get; storage/errors.go sentinel — green this pass |
| CORE-02 | 02-01 | Set create/replace under LWW | ✓ SATISFIED | TestStore_endToEnd (incl. the replace-under-greater-version subtest that now solely pins this observable post-dedup), TestStore_Set remaining subtests — green this pass |
| CORE-03 | 02-01 | SetIfAbsent; existing (incl. tombstone) → AlreadyExists | ✓ SATISFIED | TestStore_SetIfAbsent, TestStore_concurrentSetIfAbsent |
| CORE-04 | 02-01 | Delete fresh-version tombstone, replicates like any write | ✓ SATISFIED | TestStore_Delete, TestMerge_lawsTombstones |
| CORE-05 | 02-01 | Values over value_max_bytes → InvalidArgument | ✓ SATISFIED | TestStore_valueLimit on all three write paths, check before any copy (store.go:160/188/220 re-located this pass) |
| CORE-06 | 02-01, 02-03 | LWW-Element-Map laws; order/duplicates/retransmissions cannot affect outcome | ✓ SATISFIED | TestMerge_laws/lawsTombstones/duplicateDelivery — green this pass |
| CORE-07 | 02-01, 02-03 | Every displaced version recorded | ✓ SATISFIED | TestMerge_losers/losersProperty |
| VERS-01 | 02-02 | UUIDv7 byte order equals (l, c, NodeID) order | ✓ SATISFIED | TestVersion_layout/byteOrder — green this pass |
| VERS-02 | 02-02 | Unique per write by construction | ✓ SATISFIED | TestVersion_uniquePerWrite; ceiling + watermark-clamp enforcement |
| VERS-03 | 02-02 | Monotonic under adversarial clock input | ✓ SATISFIED | TestClock_monotonic, counterOverflowBoundary, ceilingSaturation — green this pass |
| VERS-04 | 02-02, 02-04 | Restart seeds the dump's full watermark; every post-restart version strictly dominates every dumped version | ✓ SATISFIED | clock.NewClock initialL/initialC seam + NewStore clock injection; TestClock_restartProperty, TestClock_restartFromDumpMax, TestStore_restartDominance — all green this pass; REQUIREMENTS.md wording matches the implemented route |
| VERS-05 | 02-02 | Drift beyond drift_warn logged as warning | ✓ SATISFIED (engine side; logging consumer delivered by Phase 6 — see Deferred) | Signed saturated drift signal returned by every clock-advancing op; TestClock_driftSignal + driftUnderChaos + driftSignalSaturates green this pass |
| TEST-01 | 02-03 | CRDT properties: commutative, associative, idempotent | ✓ SATISFIED | TestMerge_laws, TestMerge_lawsTombstones (500 seeded triples per mix) — green this pass |
| TEST-02 | 02-02, 02-04 | HLC properties incl. counter-reset and restart-from-dump-max | ✓ SATISFIED | Six properties in clock/hlc_prop_test.go; ceiling and watermark-clamp cases both tested |

Orphaned requirements: none — REQUIREMENTS.md maps exactly the 14 IDs to Phase 2, all claimed by plans (02-01: CORE-01..07; 02-02: VERS-01..05 + TEST-02; 02-03: TEST-01 + CORE-06/07; 02-04: VERS-02/04 + TEST-02), all marked Complete in the traceability table, consistent with this pass.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers, no empty implementations, no console logging in any phase-2 file (changed files scanned in full this pass) | — | — |

Review-disposition status: all three 02-REVIEW.md findings (IN-01/IN-02/IN-03) are dispositioned `fixed` by plan 06-02 (D-15); open count 0. One letter-level note on the record: the IN-03 disposition row states the "now-unused cmpSame constant" was removed with the subtest — `cmpSame` (store_test.go:39) in fact remains and is still used by merge_test.go:156,247, so it was never solely owned by the removed subtest (had it been removed, the package would not compile). ℹ️ Info — disposition-prose accuracy only; the code state is correct and the suite compiles and runs green.

Pre-existing (unchanged files, carried note): the `//nolint` directives in `internal/app/internal/testutil/{gen,rng}.go` all name their specific linter (gosec G115/G404, gochecknoglobals) with written justifications, per project deviation 3 — compliant, verified by the clean strict-lint run. Zero nolint in phase-2 production files (grepped this pass).

### Prohibition Adjudication

- **02-01 "no hidden side effects… no logging, metrics, callbacks or accumulated events inside internal/app; losers and drift leave exclusively as return values (D-01)":** substance VERIFIED at HEAD — zero logging/metrics/accumulated events (greps clean this pass); Merge still returns every loser as `[]Conflict` and drift still leaves exclusively as a return value; the default Config is tap-free (`nil` OnChange normalized to a no-op). One clause — "no callbacks" — remains qualified by the **documented later-phase decision** phase 5-03 (D-06, the explicit opt-in `Config.OnChange` tap) and now also by phase 6-02 (D-15), which refined only the tap's doc contract (comment-only change, proven this pass). This is a deliberate cross-phase architecture extension, not a phase-2 execution artifact and not a hidden side effect; it does not break the I/O-free guarantee the prohibition protects (depguard re-probed green in-process this pass). No override entry is created: the must-have's operative contract (return-value semantics, purity) holds, and the deviation belongs to the later phases' own verified scopes.
- **02-01 "go.mod keeps zero direct runtime requires" (D-04):** substance VERIFIED — phase 2 added no dependency and no property-test/UUID library entered the build (direct requires at HEAD are exactly grpc + protobuf, both later-phase entries under the AGENTS.md documented policy "gRPC and protobuf enter go.mod in the phases that first use them"; go.sum byte-identical since the prior verified state; the only go.mod movement is the `toolchain go1.27.4` directive). The zero-direct-requires letter described the phase-1/2 deliverable state and is superseded by the documented dependency policy, not by any phase-2 action.
- **02-01 re-stamping / truncation prohibitions, 02-02 and 02-03 seeded-randomness prohibitions:** VERIFIED — join() never re-stamps (code byte-identical this pass); oversized values rejected whole before any copy; every generator seeds via `testutil.NewRNG` with printed seed and `-seed` replay (files byte-stable since the prior verified state, source re-checked).

### Human Verification Required

None. This is a pure library phase (infrastructure/foundation — no user-facing elements); every behavior-dependent truth is exercised by the seeded property/unit suite run green under `-race` in this verification's own process — including the unfiltered per-package coverage run that executed every phase-2 test. No ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths.

### Gaps Summary

No gaps. The three post-verification movements of covered inputs were each verified at depth: the go.mod change is the toolchain directive only (dependency graph byte-stable, D-04 substance re-checked); the store.go change is comment-only by strip-diff proof (code byte-identical, all prior behavioral proofs carry) with both IN-01/IN-02 doc fixes confirmed in the source; the store_test.go change is the IN-03 dedup whose removed subtest left the CORE-02 replace observable pinned exactly once in `TestStore_endToEnd` (re-read, green). The complete phase-2 suite re-ran green under the race detector at 100.0% statement coverage per package, strict lint is clean on the unchanged config, the depguard purity gate was re-probed in-process (denial captured), direct imports remain stdlib + own module only, and the test-function inventory is unchanged. All 14 requirement IDs are satisfied; the one deferred item (VERS-05's logging consumer) has been discharged by Phase 6's own completed, verified work.

---

_Verified: 2026-11-16T11:04:05Z_
_Verifier: Claude (gsd-verifier)_
