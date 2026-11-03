---
phase: 02-engine-hlc-versioning-and-lww-store
verified: 2026-11-03T14:16:17Z
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
covered_digest: "v4:sha256:40e07388416f7c2199b9071ab99546c44717411612bcbfbf43bcf70477d6e608"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 40/40
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Report regeneration #4. The prior pass (2026-11-02T14:57:21Z, merged as 307f252) was passed with no open gaps; this report was triggered because two covered files changed after it: commit 22575e3 (PR #29, WR-01 drift saturation + WR-02 restart-watermark clamp) touched internal/app/clock/clock.go and clock_test.go. Phase 3 (617951e) added new packages but touched none of the covered set (git diff over the covered paths confirms only the two clock files moved). Every clock-package truth was re-verified in full against the changed code with named-test runs; all other truths were regression-checked (files byte-identical to the prior verified state, suite green under -race at 100% per-package coverage, scoped strict lint clean). No regressions found."
deferred:
  - truth: "VERS-05's 'drift beyond drift_warn is logged as a warning' — the logging half (the engine-side signed drift signal is complete, property-proven, and now overflow-safe)"
    addressed_in: "Phase 6"
    evidence: "REQUIREMENTS.md OBS-01: 'A metrics endpoint exposes ... clock-drift warnings'; OBS-03: structured logging. D-01 forbids logging inside internal/app by design, so the logging consumer necessarily lands in a later phase."
---

# Phase 2: Engine — HLC Versioning and LWW Store Verification Report

**Phase Goal:** The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Verified:** 2026-11-03T14:16:17Z (at HEAD be149ce)
**Status:** passed
**Re-verification:** Yes — report regeneration #4 after the WR-01/WR-02 clock fix (prior passed report 2026-11-02T14:57:21Z; no gaps were open then)

## Change Delta Since the Prior Passed Report

Exactly one covered-file change set landed after the prior verification: **22575e3 (PR #29)** on `internal/app/clock/clock.go` (+39/−7) and `clock_test.go` (+88/−0):

1. **WR-02 — restart watermark clamped into the version domain.** `NewClock` now applies `l: min(max(initialL, 0), maxL)` (clock.go:80): a corrupted dump watermark beyond the 48-bit ceiling saturates at `maxL` instead of bypassing Tick's ceiling saturation (where `PackVersion`'s `tsMask` would silently wrap every subsequent write to the bottom of the version stream), and a negative watermark reads as zero. The constructor is now total over any int64 input, mirroring the existing `initialC & counterMask`. Regression test: `TestClock_watermarkClampedIntoDomain` (clock_test.go:456, 3 subtests incl. a packed-version `Compare` assertion against the domain top).
2. **WR-01 — drift signal saturates instead of wrapping int64.** `driftBetween` clamps both operands and their difference into the ±`driftMsMax` band (`math.MaxInt64 / Millisecond`) before the nanosecond conversion (clock.go:159-181); the old raw `(l - ptMs) * Millisecond` product wrapped past ~292 years of divergence, flipping sign/magnitude on exactly the worst drift the signal exists to report. Drift stays diagnostic-only, returned never recorded. Regression test: `TestClock_driftSignalSaturates` (clock_test.go:586, 3 subtests with a derived 200-year floor).

Everything else in the covered set is byte-identical to the prior verified state (`git diff 307f252..be149ce` over the covered paths moves only the two clock files). Phase 3 (617951e) added `internal/app/persist`, `internal/persistfile` and extended `internal/port` — outside the covered set; `go.mod`/`go.sum`/`.golangci.yml` are unchanged, so the zero-direct-runtime-requires and depguard-boundary prohibitions still hold at HEAD.

One prior warning resolved without phase-2 action: REQUIREMENTS.md VERS-04 was reworded by the phase-3 docs update to name the current route (`clock.NewClock`'s `initialL`/`initialC` injected into `storage.NewStore` through the `storage.Clock` interface) — the stale `Config.InitialL`/`InitialC` wording flagged in the prior report is gone.

## Goal Achievement

The goal holds at HEAD be149ce. `internal/app` remains a pure, I/O-free three-package library (`version`, `clock`, `storage` plus test-only `internal/testutil`): the depguard boundary `sup-kv-app-no-io` was re-probed in this verification's own process (a production-file `import "os"` in `internal/app/clock` fails lint with `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)`), scoped strict lint over all four packages exits 0 with the new `math` import, and purity greps find zero logging, zero `fmt.Print`, zero production `time.Now()`. The HLC packs into RFC 9562 UUIDv7, the LWW-Element-Map store implements the full CORE semantics with exact loser accounting, and the correctness laws are property-proven by the seeded, replayable suite — green under the race detector at 100.0% statement coverage per production package (clock, version, storage; first-hand this pass). The WR-01/WR-02 fix strengthens two previously unproven input domains (out-of-range restart watermark, unrepresentable drift divergence) with the constructor/signal now total and both regression tests passing under `-race`.

### Observable Truths

Plan 02-01 (17 truths — all VERIFIED; storage/version/testutil files byte-identical to the prior verified state, regression-checked: all `TestStore_*`/`TestMerge_*`/`TestVersion_*` named runs green under `-race` this pass; per-package coverage 100.0%):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Local write flows end-to-end: Set stamps fresh HLC version in UUIDv7 carrying NodeID; Get returns exact bytes + version | ✓ VERIFIED | `TestStore_endToEnd` (storage/store_test.go); green under `-race` this pass |
| 2 | Get returns ErrNotFound for missing and tombstoned keys (CORE-01) | ✓ VERIFIED | `TestStore_Get` subtests; storage/errors.go sentinel |
| 3 | Set replaces prior value under LWW, greater version wins (CORE-02) | ✓ VERIFIED | `TestStore_Set` replacement subtest |
| 4 | SetIfAbsent returns ErrAlreadyExists for any existing record incl. tombstone (CORE-03) | ✓ VERIFIED | `TestStore_SetIfAbsent` subtests |
| 5 | Delete writes fresh-version tombstone that stays in map and merges like any write (CORE-04) | ✓ VERIFIED | `TestStore_Delete`; `Records()` keeps tombstones; `TestMerge_lawsTombstones` |
| 6 | Values over ValueMaxBytes rejected with ErrValueTooLarge before any copy (CORE-05) | ✓ VERIFIED | `TestStore_valueLimit` 61440/61441 on set, setifabsent and merge; store.go:133/165/199 check precedes every copy (paths re-confirmed in 02-SECURITY.md sync) |
| 7 | Merge keeps remote version, displaces only on strictly greater full 16-byte version, returns every displaced/rejected version as Conflict (CORE-06/07) | ✓ VERIFIED | storage/store.go join() `rec.Version.Compare(stored.Version)` on the full 16 bytes; `TestMerge_losers` + `TestMerge_losersProperty` |
| 8 | Two writes in same ms get distinct strictly increasing versions | ✓ VERIFIED | `TestStore_Set` same-ms subtest |
| 9 | Zero-length value is legal | ✓ VERIFIED | `TestStore_valueLimit` zero-length rows |
| 10 | Repeat writes never error, fresh version each time | ✓ VERIFIED | `TestStore_Set` repeat subtest |
| 11 | SetIfAbsent twice: first creates, second ErrAlreadyExists, first value kept | ✓ VERIFIED | `TestStore_SetIfAbsent` |
| 12 | Delete on missing/tombstoned key: ErrNotFound, no new tombstone, no version burned | ✓ VERIFIED | `TestStore_Delete` no-burn subtests; store.go:182 guards before write |
| 13 | Equal-version merge is a full no-op, no Conflict | ✓ VERIFIED | join() default branch returns nil; `TestMerge_losers` + `TestMerge_duplicateDelivery` |
| 14 | Merge into missing key installs, no loser reported | ✓ VERIFIED | join() install case; `TestMerge_losers` |
| 15 | No Conflict ever carries loser == winner | ✓ VERIFIED | `TestMerge_losersProperty` explicit assertion |
| 16 | Displacing-nothing merge returns empty conflict list, not error | ✓ VERIFIED | join() nil return; asserted across merge suite |
| 17 | Sentinel errors matched with errors.Is | ✓ VERIFIED | storage/errors.go 3 sentinels; tests import and use errors.Is |

Plan 02-02 (11 truths — all VERIFIED; this is the plan whose package changed: truths 4, 5, 9, 10 were re-verified in full with named-test runs against the WR-01/WR-02 code; every other clock test green under `-race` this pass):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Tick/Update follow Kulkarni rules, (l, counter) strictly increases on every call (VERS-03) | ✓ VERIFIED | `TestClock_tick`/`TestClock_update`/`TestClock_monotonic` — named run PASS this pass (both monotonic subtests incl. concurrent) |
| 2 | Byte-wise compare equals (l, counter, NodeID) order vs field-wise reference model (VERS-01) | ✓ VERIFIED | `TestVersion_byteOrder` (seeded triples) + `TestVersion_layout` octet vector; version package untouched |
| 3 | 10,000 Ticks under stuck clock → 10,000 distinct strictly increasing versions (VERS-02) | ✓ VERIFIED | `TestVersion_uniquePerWrite`; no-wrap now ceiling-enforced AND watermark-clamp-enforced end to end |
| 4 | Drift signal fires both directions with sign; always returned, never recorded (VERS-05, D-01) | ✓ VERIFIED | Named run PASS this pass: `TestClock_driftSignal` (9 rows: both signs, both ops, boundary-exactly-warn, disabled states) + `TestClock_driftUnderChaos`; NEW `TestClock_driftSignalSaturates` (3 subtests) proves the WR-01 saturation keeps the correct sign at divergences a Duration cannot represent; zero logging in internal/app |
| 5 | Clock constructed with the dump's FULL watermark emits only versions strictly greater than EVERY dump version (VERS-04) | ✓ VERIFIED | Named run PASS this pass against the changed `NewClock`: `TestClock_restartProperty` (watermarkC 1..4095 seeded, 100-tick adversarial walk, strict Compare dominance over all dump versions) + `TestClock_restartFromDumpMax` 6 subtests; the new WR-02 clamp only narrows the accepted input domain (out-of-range watermarks now saturate in-domain instead of wrapping) — in-domain watermarks behave identically |
| 6 | Concurrent Tick/Update: parallel mix, zero race findings | ✓ VERIFIED | `TestClock_monotonic` concurrent subtest PASS this pass; full clock package `-race` green |
| 7 | Same remote version twice never decreases (l, c) — second Update no-op | ✓ VERIFIED | `TestClock_update` equal-timestamp rows |
| 8 | InitialL ahead of wall clock → first versions at/above InitialL; zero starts from wall clock | ✓ VERIFIED | `TestClock_restartFromDumpMax` first two subtests PASS this pass |
| 9 | Restart exact to the ms: every subsequent version strictly greater than every dump version, even under backward-jumped clock (VERS-04 precision) | ✓ VERIFIED | Named run PASS this pass: `TestStore_restartDominance` 4 subtests (variant A peer-accepts with exact `Conflict{Winner: fresh}`, variant B stale-watermark displaces with exactly one Conflict) — exercises the changed `NewClock` through `storage.NewStore` composition |
| 10 | Stuck clock: Ticks 1-4096 fill counters 0-4095, Tick 4097 → (l+1, 0); never wrap/repeat (TEST-02 boundary) | ✓ VERIFIED | `TestClock_counterOverflowBoundary` PASS this pass for the 4096/4097 rule; no-wrap holds on both ends — ceiling clamp in Tick/Update (`TestClock_ceilingSaturation`, 3 subtests PASS) and, since WR-02, the constructor clamp (`TestClock_watermarkClampedIntoDomain` subtest 1 asserts the packed version never reads below the domain top); the prior coincidental-reliance item stays resolved |
| 11 | HLC state is plain integer arithmetic — no time.Time held as state, no monotonic-read comparisons (Pitfall 8, backstop) | ✓ VERIFIED | Re-observed on the changed source: `Clock` fields are `l int64`, `c uint16`, `now func() time.Time`, `driftWarn time.Duration` (clock.go:43-49); Tick/Update reduce the seam to `UnixMilli()` int64 immediately; the WR-01/WR-02 changes are pure `min`/`max` integer arithmetic over the existing constants — zero time-comparison methods, zero `time.Now()` in production code |

Plan 02-03 (11 truths — all VERIFIED; storage/testutil files byte-identical to the prior verified state, regression-checked: all `TestMerge_*`/`TestStore_*` named runs green under `-race` this pass):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Merge commutative, associative, idempotent incl. tombstones over seeded random stores (TEST-01, CORE-06) | ✓ VERIFIED | `TestMerge_laws`/`TestMerge_lawsTombstones` — 500 seeded triples per mix |
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

Plan 02-04 (3 gap-closure truths — truths 1-2 restate the historically falsified 02-02 pair and are deduplicated into the count via those rows; truth 3 re-verified against the changed constructor):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Clock constructed with the dump's full watermark emits only versions strictly greater than every dump version, incl. counter >= 1 at the watermark ms and the own NodeID (dedup of 02-02 #5) | ✓ VERIFIED | `TestClock_restartProperty` + `TestClock_restartFromDumpMax` named runs PASS this pass against the WR-02-clamped constructor |
| 2 | First post-restart write never collides with or subordinates to the dump; peer accepts; stale watermark displaces with exactly one Conflict (dedup of 02-02 #9) | ✓ VERIFIED | `TestStore_restartDominance` named run PASS this pass — 4 subtests with value-level `Unpack()`/`Compare` assertions, under `-race` |
| 3 | The restart seam expresses a counter seed reaching the clock, so the dump's full watermark is carryable; the now seam remains the single wall-clock input — the seed is constructor state, not a clock read (D-02) | ✓ VERIFIED (strengthened by WR-02) | `clock.NewClock(now, initialL, initialC, driftWarn)` — signature unchanged by the fix; the seed is constructor state masked/clamped on entry (`initialC & counterMask`, `min(max(initialL, 0), maxL)`, clock.go:80), never a clock read; `now` is still the single wall-clock seam. WR-02 makes the seam total over any int64 watermark, so no caller-side pre-clamping is needed. Composition route (caller builds `clock.NewClock(...)` → `storage.NewStore(nodeID, clk, cfg)`) exercised by `TestStore_restartDominance` and `testutil.RandStore`; REQUIREMENTS.md VERS-04 now documents exactly this route |

**Score:** 40/40 truths verified (0 present, behavior-unverified). Denominator unchanged from the prior passed report: the 39 truths of plans 02-01..02-03 (17 + 11 + 11) plus 02-04's seam-expressiveness truth; 02-04's other two truths are deduplicated restatements of the 02-02 pair.

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | VERS-05's "logged as a warning" half — the engine returns the signed, now overflow-safe drift signal (complete, property-proven); actual logging/metrics wiring | Phase 6 | REQUIREMENTS.md OBS-01 (clock-drift warnings in the metrics endpoint), OBS-03 (structured logging); D-01 forbids logging inside internal/app by design |

### Advisory (New Scope, Unevidenced)

Re-verification ran (report regeneration #4). No new-scope unevidenced findings this pass: the only covered-file changes (clock.go/clock_test.go) are fully evidenced by the passing WR-01/WR-02 regression tests run in this verification's own process, and the out-of-covered-scope phase-3 additions are covered by phase 3's own verification.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Required Artifacts

Every plan declares paths under the old single-package layout; the artifacts are verified at their post-split homes (the split predates the prior passed report). Existence, substance and wiring re-confirmed this pass; only the two clock files changed content, both growing.

| Plan path (stale) | New home | Expected | Status | Details |
| -------- | -------- | -------- | ------ | ------- |
| internal/app/errors.go | internal/app/storage/errors.go | Sentinels matched with errors.Is, RPC mapping docs | ✓ VERIFIED | 19 lines; ErrNotFound/ErrAlreadyExists/ErrValueTooLarge exported with per-error docs |
| internal/app/version.go | internal/app/version/version.go | RecordVersion: pack, total unpack, byte-wise compare | ✓ VERIFIED | 65 lines; all four exports present; byte-identical to prior verified state |
| internal/app/clock.go | internal/app/clock/clock.go | HLC with now seam, overflow rule, initial-l seam, drift signal | ✓ VERIFIED | 181 lines (was 149; +32 net from WR-01/WR-02); Clock/NewClock/Tick/Update; 4-param NewClock, initialC masked, initialL clamped (WR-02); ceiling saturation; drift saturation (WR-01); read in full this pass |
| internal/app/store.go | internal/app/storage/store.go | Config, Record, Conflict, RecordChecksum, Store API | ✓ VERIFIED | 256 lines; all exports incl. clock-injecting NewStore; byte-identical to prior verified state |
| internal/app/store_test.go | internal/app/storage/store_test.go | Tracer + CORE tables + merge unit tests | ✓ VERIFIED | 901 lines; `TestStore_endToEnd`, `TestStore_concurrentHammer`, `TestStore_restartDominance` present |
| internal/app/doc.go | internal/app/storage/doc.go | Package contract: LWW laws, visibility, D-01 contracts | ✓ VERIFIED | 37-line package doc |
| internal/app/clock_test.go | internal/app/clock/clock_test.go | Kulkarni tables, overflow boundary, restart, drift | ✓ VERIFIED | 617 lines (was 529); prior tests intact + `TestClock_watermarkClampedIntoDomain` (line 456) + `TestClock_driftSignalSaturates` (line 586) |
| internal/app/props_test.go | internal/app/internal/testutil/rng.go + gen.go | Seeded generators, printed seed, -seed replay | ✓ VERIFIED | 51 + 85 lines; `NewRNG` prints the seed and honors `-seed`; byte-identical to prior verified state |
| internal/app/version_test.go | internal/app/version/version_test.go | RFC 9562 vectors, round-trip, byte-order property | ✓ VERIFIED | 211 lines; `TestVersion_byteOrder`, `TestVersion_unpackTotal` canonicalization property |
| internal/app/hlc_prop_test.go | internal/app/clock/hlc_prop_test.go | TEST-02 property suite | ✓ VERIFIED | 312 lines; six properties incl. `TestClock_restartProperty`; byte-identical to prior verified state |
| internal/app/merge_test.go (02-03) | internal/app/storage/merge_test.go | TEST-01 law suite | ✓ VERIFIED | 261 lines; `TestMerge_laws`, `TestMerge_losersProperty`, `TestMerge_duplicateDelivery` |

Wiring re-confirmed on current source: `storage` production calls `s.clock.Tick()` (store.go:146) and `s.clock.Update()` (store.go:203) through the consumer-side `storage.Clock` interface (store.go:51) and stamps/decides via `version.PackVersion`/`Compare`; storage production imports only `bytes`, `hash/crc32`, `sync`, `time`, `version`; `clock` is imported by storage tests (the interface proof) and testutil. No production binary composition exists yet (`cmd/sup-kv` remains the Phase 1 skeleton) — the expected state for a library phase; engine wiring is later-phase scope.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/app/storage/store.go (02-01) | clock Tick/Update | `s.clock.Tick()` / `s.clock.Update()` through `storage.Clock` | ✓ WIRED | store.go:146, store.go:203 — unchanged, re-grepped this pass |
| internal/app/storage/store.go (02-01) | version Pack/Compare | `version.PackVersion(l, counter, s.nodeID)`; join decides via full-16-byte `Compare` | ✓ WIRED | store.go:147, store.go:216 |
| storage tests (02-01) | exported storage API | external test packages drive only exported symbols with injected fakes | ✓ WIRED | imports confirmed in all five test files |
| hlc_prop_test → generators (02-02) | testutil + same-package helpers | `testutil.NewRNG`, `nextWallTime` | ✓ WIRED | hlc_prop_test.go imports testutil; unchanged |
| version_test → version (02-02) | PackVersion/Unpack/Compare | layout vectors and canonicalization property | ✓ WIRED | unchanged |
| clock_test → clock (02-02) | NewClock via fake now (D-02) | every Kulkarni branch + the two new WR regression paths | ✓ WIRED | `clock.NewClock(` call sites across clock_test.go incl. the maxL+5 / MaxInt64 / negative-watermark and pre-epoch-wall rows |
| merge_test → testutil + storage (02-03) | RandStore/NewRNG; Merge/Records only | laws drive exported API; remote records keep their versions | ✓ WIRED | join() never re-stamps; unchanged |
| store_test → storage (02-03) | concurrent hammer on exported API | `sync.WaitGroup`/goroutines; race detector is the assertion | ✓ WIRED | `TestStore_concurrentHammer` + friends |
| store.go → clock.go (02-04) | caller composes `clock.NewClock(now, initialL, initialC, driftWarn)` into `NewStore(nodeID, clk, cfg)` | ✓ WIRED | Composition exercised by `TestStore_restartDominance` (named run PASS this pass) and `testutil.RandStore`; the tool's literal-pattern check still reports the old pre-refactor route pending — the superseding route is source-verified |

### Data-Flow Trace (Level 4)

Not applicable in the render sense — a library with no UI/external data source. The equivalent holds on current source: every version flows from `clock.Tick/Update` through `version.PackVersion`; the restart seed flows from `NewClock`'s `initialL`/`initialC` into constructor state (clamped in-domain since WR-02, never a clock read — D-02 preserved); the drift signal flows out of every clock-advancing op as a saturated `time.Duration` return value (never recorded, never fed back); Merge decides via full 16-byte `Compare` on actual record versions. No hardcoded data paths anywhere in `internal/app`.

### Behavioral Spot-Checks

All commands run in this verification's own process at HEAD be149ce. Constraint honored: no full-workspace re-run (the orchestrator ran `make test` green at be149ce; the runs below are scoped named-test runs plus one phase-2-package coverage run).

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| WR-02 regression: watermark clamp | `go test -race -count=1 -v -run '^TestClock_watermarkClampedIntoDomain$' ./internal/app/clock` | PASS — 3 subtests (ceiling saturation with packed Compare check, MaxInt64 input, negative watermark adopts wall clock) | ✓ PASS |
| WR-01 regression: drift saturation | `go test -race -count=1 -v -run '^TestClock_driftSignalSaturates$' ./internal/app/clock` | PASS — 3 subtests (post-maxL-merge positive, saturated-clock Update positive, pre-epoch wall positive — sign never flips) | ✓ PASS |
| Restart dominance vs changed NewClock (VERS-04 pair) | `go test -race -count=1 -v -run '^(TestClock_restartProperty|TestClock_restartFromDumpMax|TestClock_ceilingSaturation|TestClock_counterOverflowBoundary)$' ./internal/app/clock` + `^TestStore_restartDominance$ ./internal/app/storage` | PASS — restart property, 6 restart-seed subtests, 3 ceiling subtests, overflow boundary, 4 restart-dominance subtests | ✓ PASS |
| Drift contract unchanged in-band (VERS-05) | `go test -race -count=1 -v -run '^(TestClock_driftSignal|TestClock_driftUnderChaos)$' ./internal/app/clock` | PASS — 9 table rows + chaos property | ✓ PASS |
| Clock monotonicity/concurrency (VERS-03) | included in the named clock runs above (`TestClock_monotonic` both subtests) | PASS under `-race` | ✓ PASS |
| Remaining unit/property surface regression | `go test -race -count=1 -run '^TestClock_tick$|^TestClock_update$|^TestVersion_|^TestMerge_|^TestStore_' ./internal/app/clock/ ./internal/app/version/ ./internal/app/storage/` | `ok` for all 3 packages | ✓ PASS |
| Phase-2 package suite with coverage | `go test -race -count=1 -cover ./internal/app/clock/ ./internal/app/version/ ./internal/app/storage/` | ok clock 100.0%, version 100.0%, storage 100.0% of statements | ✓ PASS |
| Strict lint on phase-2 packages | `go tool golangci-lint run ./internal/app/clock/... ./internal/app/version/... ./internal/app/storage/... ./internal/app/internal/testutil/...` | exit 0, zero issues (new `math` import compliant) | ✓ PASS |
| Depguard boundary probe (own process) | inject production `import "os"` into `internal/app/clock/`, run scoped lint, remove | exit 1: `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)`; lint green after removal; tree clean | ✓ PASS |
| Zero-dependency policy (D-04) | `git diff 81810f8..HEAD -- go.mod go.sum .golangci.yml` | empty — byte-identical; only the two tool directives, zero direct runtime requires | ✓ PASS |
| Purity greps | forbidden imports / logging / `fmt.Print` / production `time.Now()` in the five production files | none found (testutil's `time.Now().UnixNano()` is the test-seed source in a test-only-linked package) | ✓ PASS |
| Test-inventory continuity | `git show 22575e3 -- clock_test.go` func lines + `grep -rh '^func Test'` over the three packages | 31 → 33 functions; only delta `+TestClock_watermarkClampedIntoDomain`, `+TestClock_driftSignalSaturates`; zero deletions/renames | ✓ PASS |
| Seed replay contract | testutil/rng.go source (byte-identical to prior verified state) | `NewRNG` prints `property seed: h,l (replay: ...)`; `-seed` flag registered | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist or were declared (no scripts directory). The phase's probe obligations are discharged by the named Go tests above plus the depguard boundary probe, executed in this verification's own process with the raw lint failure captured.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CORE-01 | 02-01 | Get: missing/tombstoned → NotFound | ✓ SATISFIED | TestStore_Get; storage/errors.go sentinel |
| CORE-02 | 02-01 | Set create/replace under LWW | ✓ SATISFIED | TestStore_Set, TestStore_endToEnd |
| CORE-03 | 02-01 | SetIfAbsent; existing (incl. tombstone) → AlreadyExists | ✓ SATISFIED | TestStore_SetIfAbsent, TestStore_concurrentSetIfAbsent |
| CORE-04 | 02-01 | Delete fresh-version tombstone, replicates like any write | ✓ SATISFIED | TestStore_Delete, TestMerge_lawsTombstones |
| CORE-05 | 02-01 | Values over value_max_bytes → InvalidArgument | ✓ SATISFIED | TestStore_valueLimit on all three write paths, check before any copy |
| CORE-06 | 02-01, 02-03 | LWW-Element-Map laws; order/duplicates/retransmissions cannot affect outcome | ✓ SATISFIED | TestMerge_laws/lawsTombstones/duplicateDelivery — green this pass |
| CORE-07 | 02-01, 02-03 | Every displaced version recorded | ✓ SATISFIED | TestMerge_losers/losersProperty |
| VERS-01 | 02-02 | UUIDv7 byte order equals (l, c, NodeID) order | ✓ SATISFIED | TestVersion_layout/byteOrder |
| VERS-02 | 02-02 | Unique per write by construction | ✓ SATISFIED | TestVersion_uniquePerWrite; no-wrap enforced and tested at both the ceiling (Tick/Update saturation) and, since WR-02, the constructor watermark clamp |
| VERS-03 | 02-02 | Monotonic under adversarial clock input | ✓ SATISFIED | TestClock_monotonic (named run PASS this pass), counterOverflowBoundary, counterReset, ceilingSaturation |
| VERS-04 | 02-02, 02-04 | Restart seeds the dump's full watermark; every post-restart version strictly dominates every dumped version | ✓ SATISFIED | clock.NewClock initialL/initialC seam (now total over any int64 per WR-02) + NewStore clock injection; TestClock_restartProperty, TestClock_restartFromDumpMax, TestStore_restartDominance — all named-run PASS this pass; REQUIREMENTS.md wording now matches the implemented route |
| VERS-05 | 02-02 | Drift beyond drift_warn logged as warning | ✓ SATISFIED (engine side, now overflow-safe; logging deferred to Phase 6 OBS-01/03) | Signed saturated drift signal returned by every clock-advancing op; TestClock_driftSignal + TestClock_driftUnderChaos + TestClock_driftSignalSaturates |
| TEST-01 | 02-03 | CRDT properties: commutative, associative, idempotent | ✓ SATISFIED | TestMerge_laws, TestMerge_lawsTombstones (500 seeded triples per mix) |
| TEST-02 | 02-02, 02-04 | HLC properties incl. counter-reset and restart-from-dump-max | ✓ SATISFIED | Six properties in clock/hlc_prop_test.go; restart case is the full-watermark model; ceiling and watermark-clamp cases both tested |

Orphaned requirements: none — REQUIREMENTS.md maps exactly the 14 IDs to Phase 2, all claimed by plans, all marked Complete in the traceability table, consistent with this pass.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | None in the changed files: no TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers, no empty implementations, no console logging | — | — |

Pre-existing (unchanged files, carried note): the six `//nolint` directives in `internal/app/internal/testutil/{gen,rng}.go` all name their specific linter (gosec G115/G404, gochecknoglobals) with written justifications, per project deviation 3 — compliant, verified by the clean strict-lint run.

### Human Verification Required

None. Every behavior-dependent truth in this phase is exercised by the seeded property/unit suite (the phase's deliverable is exactly that suite); the changed code paths carry the two new WR regression tests, run green under `-race` in this verification's own process.

### Gaps Summary

No gaps. The single post-verification change set to covered files (22575e3, WR-01 drift saturation + WR-02 watermark clamp) was re-verified in full: source read line-by-line, both regression tests plus the full restart-dominance and drift contracts re-run green under the race detector, strict lint and the depguard purity gate re-probed in-process, go.mod/go.sum confirmed byte-identical (zero direct runtime requires), and the test inventory confirmed additive (31 → 33, no deletions). All 14 requirement IDs are satisfied; the one deferred item (VERS-05's logging consumer) remains scheduled for Phase 6 by design.

---

_Verified: 2026-11-03T14:16:17Z_
_Verifier: Claude (gsd-verifier)_
