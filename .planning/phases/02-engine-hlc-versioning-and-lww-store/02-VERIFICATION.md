---
phase: 02-engine-hlc-versioning-and-lww-store
verified: 2026-11-02T14:57:21Z
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
covered_digest: "v4:sha256:71775348faa35b1e1f32cedc9a14bac0f6afa7a1691af0184f11a8ffa511ac26"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 40/40
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Report regeneration #3 — no gaps were open at the prior pass (2026-11-02T02:53:24Z). Three user-directed post-verification change sets landed on this branch (review fixes 329263e/03e3f9c/c00c618, package split 2bee4c9, Clock interface 81810f8), so every must-have was re-verified in full against the new layout with evidence recomputed."
deferred:
  - truth: "VERS-05's 'drift beyond drift_warn is logged as a warning' — the logging half (the engine-side signed drift signal is complete and property-proven)"
    addressed_in: "Phase 6"
    evidence: "REQUIREMENTS.md OBS-01: 'A metrics endpoint exposes ... clock-drift warnings'; OBS-03: structured logging. D-01 forbids logging inside internal/app by design, so the logging consumer necessarily lands in a later phase."
---

# Phase 2: Engine — HLC Versioning and LWW Store Verification Report

**Phase Goal:** The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Verified:** 2026-11-02T14:57:21Z
**Status:** passed
**Re-verification:** Yes — report regeneration #3 after structural refactor (prior passed report 2026-11-02T02:53:24Z at commit 0ad9dcd; full re-verification this pass, not a gap-closure pass)

## Structural Changes Since the First Passed Report

Three change sets landed on `feat/02-engine-hlc-lww-store` after 0ad9dcd, all re-verified this pass:

1. **Review fixes (329263e, 03e3f9c, c00c618)** — the dead `varPayloadMask` constant is gone (IN-01, confirmed absent from `version/version.go`); the HLC `l` now saturates at the 48-bit ceiling `maxL = 1<<48-1` in both Tick and Update instead of wrapping out of the version domain (IN-02, `clock/clock.go:77/85-90/110/121-129`, proven by the new `TestClock_ceilingSaturation`); `TestVersion_unpackTotal` now asserts the real canonicalization property `PackVersion(v.Unpack()) == canonicalRaw(v)` over 1000 seeded raw patterns (IN-03, `canonicalRaw` is an independent RFC 9562 bit-mask reference, not a re-run of PackVersion). Review disposition stands open:0/fixed:4 (02-REVIEW-FIX.md: IN-01..03 fixed this round, WR-01 prior-fixed by 02-04).
2. **Package split (2bee4c9)** — `internal/app` is now three production packages plus a shared test-support package: `internal/app/version` (RecordVersion + RFC 9562 pack/unpack/compare), `internal/app/clock` (Clock, NewClock, ceiling saturation), `internal/app/storage` (Store, Record, Config, Conflict, RecordChecksum, sentinels), `internal/app/internal/testutil` (seeded generators; printed-seed + `-seed` replay contract preserved in `rng.go`). Git rename detection scores the moves R060–R097 (move + adapt, not rewrite). Test inventory at HEAD is 31 functions vs 30 at 0ad9dcd — the only delta is the added `TestClock_ceilingSaturation`; zero deletions, zero renames of test functions.
3. **Clock interface (81810f8)** — `storage` declares the consumer-side `Clock` interface (Tick/Update, `storage/store.go:51-54`) and `NewStore(nodeID uint16, clk Clock, cfg Config)` injects the clock; the clock-tuning Config fields (`InitialL`/`InitialC`/`DriftWarn`) moved to `clock.NewClock` parameters. At 0ad9dcd the composition lived inside `NewStore(nodeID, cfg, now)` → `NewClock(now, cfg.InitialL, cfg.InitialC, cfg.DriftWarn)`; it now lives at the caller. Storage production code does not import `clock` (only doc comments mention it) — storage tests passing `*clock.Clock` into `NewStore` are the compile-time proof that the concrete clock satisfies the interface.

Consequence honored throughout this report: every plan-declared artifact path (`internal/app/*.go`) is stale; artifacts are verified at their new homes below. One warning-tier finding: REQUIREMENTS.md VERS-04 still says the watermark is seeded "via `Config.InitialL`/`InitialC`" — those fields no longer exist (see Anti-Patterns).

## Goal Achievement

The goal is achieved on the new layout. `internal/app` remains a pure, I/O-free library — now as three packages under the unchanged depguard glob `**/internal/app/**` (probe-verified this pass: a production-file `os` import in `internal/app/version` fails lint with `import 'os' is not allowed from list 'sup-kv-app-no-io'`), with zero `net`/`os`/`syscall`/`google.golang.org` imports, zero logging, zero `time.Now()` outside testutil's seed source. The HLC packs into RFC 9562 UUIDv7 (octet-exact vector plus the strengthened canonicalization property), the LWW-Element-Map store implements the full CORE semantics with exact loser accounting, and the correctness laws are property-proven by the seeded, replayable suite — green under the race detector at 100% statement coverage per production package. The previously falsified restart-dominance pair stays closed under the full-watermark model, and the one coincidental-reliance item from the prior report (the no-wrap clause resting on an undeclared `l < 2^48-1` precondition) is resolved by the ceiling clamp plus its adversarial-walk test. `go.mod`/`go.sum` are byte-identical to 0ad9dcd and `internal/port` is untouched.

### Observable Truths

Plan 02-01 (17 truths — all VERIFIED; re-verified against `internal/app/storage` + `internal/app/version`; full suite green under `go test -race -count=1 -cover ./...`, all named tests present, no test functions deleted since 0ad9dcd):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Local write flows end-to-end: Set stamps fresh HLC version in UUIDv7 carrying NodeID; Get returns exact bytes + version | ✓ VERIFIED | `TestStore_endToEnd` (storage/store_test.go); suite green |
| 2 | Get returns ErrNotFound for missing and tombstoned keys (CORE-01) | ✓ VERIFIED | `TestStore_Get` subtests; storage/errors.go sentinel |
| 3 | Set replaces prior value under LWW, greater version wins (CORE-02) | ✓ VERIFIED | `TestStore_Set` replacement subtest |
| 4 | SetIfAbsent returns ErrAlreadyExists for any existing record incl. tombstone (CORE-03) | ✓ VERIFIED | `TestStore_SetIfAbsent` subtests |
| 5 | Delete writes fresh-version tombstone that stays in map and merges like any write (CORE-04) | ✓ VERIFIED | `TestStore_Delete`; `Records()` keeps tombstones; `TestMerge_lawsTombstones` |
| 6 | Values over ValueMaxBytes rejected with ErrValueTooLarge before any copy (CORE-05) | ✓ VERIFIED | `TestStore_valueLimit` 61440/61441 on set, setifabsent and merge; store.go:133/165/196 check precedes every copy |
| 7 | Merge keeps remote version, displaces only on strictly greater full 16-byte version, returns every displaced/rejected version as Conflict (CORE-06/07) | ✓ VERIFIED | storage/store.go join() `rec.Version.Compare(stored.Version)` on the full 16 bytes (line 216); `TestMerge_losers` + `TestMerge_losersProperty` |
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

Plan 02-02 (11 truths — all VERIFIED against `internal/app/clock` + `internal/app/version` + `internal/app/internal/testutil`; truths 5 and 9 are the historically falsified pair, re-proven this pass with the named-test run):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Tick/Update follow Kulkarni rules, (l, counter) strictly increases on every call (VERS-03) | ✓ VERIFIED | `TestClock_tick`/`TestClock_update`/`TestClock_monotonic`; suite green under `-race` |
| 2 | Byte-wise compare equals (l, counter, NodeID) order vs field-wise reference model (VERS-01) | ✓ VERIFIED | `TestVersion_byteOrder` (seeded triples) + `TestVersion_layout` octet vector |
| 3 | 10,000 Ticks under stuck clock → 10,000 distinct strictly increasing versions (VERS-02) | ✓ VERIFIED | `TestVersion_uniquePerWrite` |
| 4 | Drift signal fires both directions with sign; always returned, never recorded (VERS-05, D-01) | ✓ VERIFIED | `TestClock_driftSignal`, `TestClock_driftUnderChaos`; zero logging in internal/app |
| 5 | Clock constructed with the dump's FULL watermark emits only versions strictly greater than EVERY dump version (VERS-04) | ✓ VERIFIED | Named run this pass: `TestClock_restartProperty` PASS — (dumpMax, watermarkC 1..4095 seeded, own testNodeID) watermark, 100-tick adversarial walk, every packed emission Compare-returns newer against all 9 dump versions (hlc_prop_test.go:238-274); plus `TestClock_restartFromDumpMax` 6 subtests PASS |
| 6 | Concurrent Tick/Update: parallel mix, zero race findings | ✓ VERIFIED | `TestClock_monotonic` concurrent subtest; full suite `-race` green |
| 7 | Same remote version twice never decreases (l, c) — second Update no-op | ✓ VERIFIED | `TestClock_update` equal-timestamp rows |
| 8 | InitialL ahead of wall clock → first versions at/above InitialL; zero starts from wall clock | ✓ VERIFIED | `TestClock_restartFromDumpMax` first two subtests |
| 9 | Restart exact to the ms: every subsequent version strictly greater than every dump version, even under backward-jumped clock (VERS-04 precision) | ✓ VERIFIED | Named run this pass: `TestStore_restartDominance` 4 subtests PASS — variant A (peer accepts first post-restart write, exact `Conflict{Winner: fresh}`) and variant B (stale watermark displaces with exactly one Conflict, never an equal-version no-op) both green under `-race` with the wall parked on the watermark ms |
| 10 | Stuck clock: Ticks 1-4096 fill counters 0-4095, Tick 4097 → (l+1, 0); never wrap/repeat (TEST-02 boundary) | ✓ VERIFIED (coincidental reliance RESOLVED) | `TestClock_counterOverflowBoundary` for the 4096/4097 rule; the no-wrap clause no longer rests on an undeclared precondition — clock.go clamps l at maxL (lines 77, 85-90, 110, 121-129) and `TestClock_ceilingSaturation` (3 subtests, incl. the adversarial beyond-ceiling walk asserting no packed version reads below its predecessor) proves it by test |
| 11 | HLC state is plain integer arithmetic — no time.Time held as state, no monotonic-read comparisons (Pitfall 8, backstop) | ✓ VERIFIED | Direct source observation: `Clock` fields are `l int64`, `c uint16`, `now func() time.Time`, `driftWarn time.Duration` (clock.go:34-40) — no time.Time state; Tick/Update reduce the seam to `UnixMilli()` int64 immediately; zero time-comparison methods, zero `time.Now()` in production code |

Plan 02-03 (11 truths — all VERIFIED against `internal/app/storage` + testutil; suite green under `-race`, no test deletions, merge path logic unchanged by the split per R073/R080 rename scores):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Merge commutative, associative, idempotent incl. tombstones over seeded random stores (TEST-01, CORE-06) | ✓ VERIFIED | `TestMerge_laws`/`TestMerge_lawsTombstones` — 500 seeded triples per mix |
| 2 | Every displaced/rejected version returned exactly once as Conflict paired with displacer (CORE-07) | ✓ VERIFIED | `TestMerge_losersProperty` both directions |
| 3 | Delivery order, duplicates, retransmissions cannot affect outcome | ✓ VERIFIED | Commutativity + `TestMerge_duplicateDelivery` |
| 4 | Parallel Set/SetIfAbsent/Delete/Merge/Get: zero race findings | ✓ VERIFIED | `TestStore_concurrentHammer`; full suite `-race` green |
| 5 | Concurrent SetIfAbsent on one key: exactly one succeeds | ✓ VERIFIED | `TestStore_concurrentSetIfAbsent` |
| 6 | Delete racing Set: survivor holds exactly the greater captured version | ✓ VERIFIED | `TestStore_concurrentDeleteVsSet` |
| 7 | merge(a,b) and merge(b,a): identical state AND identical conflict sets | ✓ VERIFIED | `assertMergeLaws` commutativity block |
| 8 | Every displaced version exactly once in conflicts, paired with displacer | ✓ VERIFIED | `TestMerge_losersProperty` exact expected set |
| 9 | merge(a,a) idempotent: zero conflicts, nothing changes | ✓ VERIFIED | `assertMergeLaws` idempotence block |
| 10 | Laws hold for empty stores | ✓ VERIFIED | empty-source clones + missing-key install cases |
| 11 | merge(merge(a,b),c) equals merge(a,merge(b,c)) incl. tombstoned keys | ✓ VERIFIED | `assertMergeLaws` associativity block |

Plan 02-04 (3 gap-closure truths — truths 1-2 restate and re-close the historically falsified 02-02 pair and are deduplicated into the count via those rows; truth 3 verified with its composition route restated for the Clock-interface refactor):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Clock constructed with the dump's full watermark emits only versions strictly greater than every dump version, incl. counter >= 1 at the watermark ms and the own NodeID (dedup of 02-02 #5) | ✓ VERIFIED | `TestClock_restartProperty` named run PASS this pass (watermarkC drawn 1..4095, never 0; own testNodeID; strict Compare dominance over all dump versions); `TestClock_restartFromDumpMax` seed rows PASS |
| 2 | First post-restart write never collides with or subordinates to the dump; peer accepts; stale watermark displaces with exactly one Conflict (dedup of 02-02 #9) | ✓ VERIFIED | `TestStore_restartDominance` named run PASS this pass — 4 subtests incl. variants A and B with exact `Unpack()` triple and `Compare(watermark) == cmpNewer` value-level assertions |
| 3 | The restart seam expresses a counter seed reaching the clock, so the dump's full watermark is carryable; the now seam remains the single wall-clock input — the seed is constructor state, not a clock read (D-02) | ✓ VERIFIED (route restated) | Operative clauses hold on the new layout: `clock.NewClock(now, initialL, initialC, driftWarn)` seeds and masks the counter (clock.go:63-64, `initialC & counterMask`); the seed is constructor state, never a clock read; `now` is still the single wall-clock seam. The literal route "Config.InitialC through NewStore into NewClock" was superseded by the Clock-interface refactor (81810f8): Config no longer carries InitialL/InitialC/DriftWarn and NewStore takes an injected `Clock` — composition now happens at the caller (`clock.NewClock(...)` → `storage.NewStore(nodeID, clk, cfg)`), exercised by `TestStore_restartDominance` and testutil.RandStore, with storage tests passing `*clock.Clock` as the compile-time interface proof. WR-01's seam deficiency stays closed: the seam carries the full watermark. See the REQUIREMENTS.md wording warning under Anti-Patterns |

**Score:** 40/40 truths verified (0 present, behavior-unverified). Denominator unchanged from the prior passed report: the 39 truths of plans 02-01..02-03 (17 + 11 + 11) plus 02-04's seam-expressiveness truth; 02-04's other two truths are deduplicated restatements of the 02-02 pair.

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | VERS-05's "logged as a warning" half — the engine returns the signed drift signal (complete, property-proven); actual logging/metrics wiring | Phase 6 | REQUIREMENTS.md OBS-01 (clock-drift warnings in the metrics endpoint), OBS-03 (structured logging); D-01 forbids logging inside internal/app by design |

### Advisory (New Scope, Unevidenced)

Re-verification ran (report regeneration); no new-scope unevidenced blocker-class findings. The one warning-tier finding this pass — REQUIREMENTS.md VERS-04 naming the removed `Config.InitialL`/`InitialC` fields — is recorded under Anti-Patterns with concrete evidence (it is a documentation-sync defect, not an unevidenced code concern).

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None — all Step 7 findings this pass are either evidence-backed warnings or info notes (below) | — | — |

### Required Artifacts

Every plan declared paths under the old single-package layout; `gsd-tools verify.artifacts` therefore reports 0/N "File not found" against the literal plan paths. Each artifact is verified below at its new home (exists, substantive, wired). Line counts exceed every plan minimum.

| Plan path (stale) | New home | Expected | Status | Details |
| -------- | -------- | -------- | ------ | ------- |
| internal/app/errors.go | internal/app/storage/errors.go | Sentinels matched with errors.Is, RPC mapping docs | ✓ VERIFIED | 19 lines (min 10); ErrNotFound/ErrAlreadyExists/ErrValueTooLarge exported with per-error docs; git rename R097 |
| internal/app/version.go | internal/app/version/version.go | RecordVersion: pack, total unpack, byte-wise compare | ✓ VERIFIED | 65 lines (min 40); all four exports present; dead varPayloadMask removed (IN-01) |
| internal/app/clock.go | internal/app/clock/clock.go | HLC with now seam, overflow rule, initial-l seam, drift signal | ✓ VERIFIED | 149 lines (min 50); Clock/NewClock/Tick/Update; 4-param NewClock with masked initialC; ceiling saturation (IN-02) |
| internal/app/store.go | internal/app/storage/store.go | Config, Record, Conflict, RecordChecksum, Store API | ✓ VERIFIED | 256 lines (min 90); all exports incl. NewStore (now clock-injecting) and full Store method set |
| internal/app/store_test.go | internal/app/storage/store_test.go | Tracer + CORE tables + merge unit tests | ✓ VERIFIED | `TestStore_endToEnd`, `TestStore_concurrentHammer`, `TestStore_restartDominance` all present |
| internal/app/doc.go | internal/app/storage/doc.go | Package contract: LWW laws, visibility, D-01 contracts | ✓ VERIFIED | 37-line package doc (old `internal/app/doc.go` deleted; contract lives with the store package it governs) |
| internal/app/clock_test.go | internal/app/clock/clock_test.go | Kulkarni tables, overflow boundary, restart, drift | ✓ VERIFIED | `TestClock_driftSignal`, `TestClock_restartFromDumpMax`, `TestClock_ceilingSaturation` (new) present |
| internal/app/props_test.go | internal/app/internal/testutil/rng.go + gen.go | Seeded generators, printed seed, -seed replay | ✓ VERIFIED | `NewRNG` prints "property seed: h,l (replay: ...)" and honors `-seed` (rng.go:41-51); RandVersion/RandRawVersion/RandRecord/RandKey/RandStore exported for cross-package use; exported renaming was required by the split |
| internal/app/version_test.go | internal/app/version/version_test.go | RFC 9562 vectors, round-trip, byte-order property | ✓ VERIFIED | `TestVersion_byteOrder` present; `TestVersion_unpackTotal` now the real canonicalization property (IN-03) |
| internal/app/hlc_prop_test.go | internal/app/clock/hlc_prop_test.go | TEST-02 property suite | ✓ VERIFIED | `TestClock_monotonic`, `TestClock_restartProperty` and the rest present; `nextWallTime` walk lives in clock_test.go (same test package) |
| internal/app/merge_test.go (02-03) | internal/app/storage/merge_test.go | TEST-01 law suite | ✓ VERIFIED | `TestMerge_laws`, `TestMerge_losersProperty`, `TestMerge_duplicateDelivery` present |

Wiring: `version` is imported by storage production (store.go) and all test packages; `storage` by its tests and testutil; `clock` by storage tests (the interface proof) and testutil — storage production imports only `bytes`, `hash/crc32`, `sync`, `time` and `version`. `testutil` is linked only into test binaries. No production binary composition exists yet (`cmd/sup-kv` remains the Phase 1 skeleton by design — engine wiring arrives with Phase 3/5), which is the expected state for a library phase.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/app/storage/store.go (02-01) | clock Tick/Update | write paths call `s.clock.Tick()` / `s.clock.Update()` through the consumer-side `storage.Clock` interface | ✓ WIRED | store.go:146 (`s.clock.Tick()` in write), store.go:203 (`s.clock.Update()` in Merge) |
| internal/app/storage/store.go (02-01) | version Pack/Compare | `version.PackVersion(l, counter, s.nodeID)`; join decides via `Compare` on the full 16 bytes | ✓ WIRED | store.go:147, store.go:216 |
| storage tests (02-01) | exported storage API | external `storage_test`-style packages drive only exported symbols with injected fakes | ✓ WIRED | imports confirmed in all five test files |
| hlc_prop_test → generators (02-02) | testutil + same-package helpers | `testutil.NewRNG`, `nextWallTime` (clock_test.go, same package) | ✓ WIRED | hlc_prop_test.go imports testutil; nextWallTime called at line 254 |
| version_test → version (02-02) | PackVersion/Unpack/Compare | layout vectors and canonicalization property | ✓ WIRED | version_test.go calls `version.PackVersion` throughout |
| clock_test → clock (02-02) | NewClock via fake now (D-02) | every Kulkarni branch | ✓ WIRED | `clock.NewClock(` call sites across clock_test.go |
| merge_test → testutil + storage (02-03) | RandStore/NewRNG; Merge/Records only | laws drive exported API; remote records keep their versions | ✓ WIRED | merge_test.go imports confirmed; join() never re-stamps |
| store_test → storage (02-03) | concurrent hammer on exported API | `sync.WaitGroup`/goroutines; race detector is the assertion | ✓ WIRED | TestStore_concurrentHammer + friends |
| store.go → clock.go (02-04) | ~~NewStore passes cfg.InitialC~~ → caller composes `clock.NewClock(now, initialL, initialC, driftWarn)` into `NewStore(nodeID, clk, cfg)` | ✓ WIRED (route restated) | The single composition point moved from inside NewStore to the caller with the Clock interface; `TestStore_restartDominance` (4 subtests) and `testutil.RandStore` exercise the composition; the tool's literal-pattern check reports the old route pending — the superseding route is source-verified |

### Data-Flow Trace (Level 4)

Not applicable in the render sense — a library with no UI/external data source. The equivalent holds on the new layout: every version flows from `clock.Tick/Update` through `version.PackVersion`; the restart seed flows from `NewClock`'s `initialL`/`initialC` parameters into clock constructor state (never a clock read — D-02 preserved); Merge decides via full 16-byte `Compare` on actual record versions; `RecordChecksum` recomputes from real key/version/data bytes on every install. No hardcoded data paths anywhere in `internal/app`.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full suite regression | `go test -race -count=1 -cover ./...` | exit 0; clock 100.0%, storage 100.0%, version 100.0% of statements | ✓ PASS |
| Previously-falsified pair + all three review fixes | `go test -race -count=1 -v ./internal/app/{clock,storage,version}/ -run '^TestClock_restartProperty$\|^TestStore_restartDominance$\|^TestClock_restartFromDumpMax$\|^TestClock_ceilingSaturation$\|^TestVersion_unpackTotal$'` | exit 0; 18 PASS lines, 0 FAIL — restart property, 6 restart-seed subtests, 3 ceiling-saturation subtests, 4 restart-dominance subtests, canonicalization property | ✓ PASS |
| Test-inventory continuity | `git grep '^func Test' 0ad9dcd` vs HEAD | 30 → 31 functions; only delta `+TestClock_ceilingSaturation`; no deletions/renames | ✓ PASS |
| Build + vet | `go build ./... && go vet ./...` | exit 0 | ✓ PASS |
| Strict lint | `go tool golangci-lint config verify && go tool golangci-lint run ./...` | exit 0, zero issues; `.golangci.yml` unchanged vs 0ad9dcd | ✓ PASS |
| Depguard boundary on subpackages (probe) | inject `import "os"` production file into `internal/app/version/`, run `go tool golangci-lint run ./internal/app/version/` | exit 1: `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)`; file removed, lint green again, tree clean | ✓ PASS (gate fires on the split layout; note `!$test` exempts `_test.go` files by design) |
| Zero-dependency policy (D-04) | `git diff --quiet 0ad9dcd..HEAD -- go.mod go.sum` | identical | ✓ PASS |
| internal/port untouched (D-03) | `git diff --quiet 0ad9dcd..HEAD -- internal/port/` + `ls internal/port` | untouched; doc.go only | ✓ PASS |
| Purity greps | forbidden imports / logging / production `time.Now()` in internal/app | none found (testutil's `time.Now().UnixNano()` is the test-seed source in a test-only-linked package) | ✓ PASS |
| Seed replay contract | testutil.NewRNG source + suite output | prints `property seed: h,l (replay: go test ./... -run '<Test>' -args -seed=N)`; `-seed` flag registered | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist or were declared (no scripts directory). The phase's probe obligations are discharged by the named Go tests above plus the depguard boundary probe documented in Behavioral Spot-Checks, executed in this verification's own process with the raw lint failure captured.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CORE-01 | 02-01 | Get: missing/tombstoned → NotFound | ✓ SATISFIED | TestStore_Get; storage/errors.go sentinel |
| CORE-02 | 02-01 | Set create/replace under LWW | ✓ SATISFIED | TestStore_Set, TestStore_endToEnd |
| CORE-03 | 02-01 | SetIfAbsent; existing (incl. tombstone) → AlreadyExists | ✓ SATISFIED | TestStore_SetIfAbsent, TestStore_concurrentSetIfAbsent |
| CORE-04 | 02-01 | Delete fresh-version tombstone, replicates like any write | ✓ SATISFIED | TestStore_Delete, TestMerge_lawsTombstones |
| CORE-05 | 02-01 | Values over value_max_bytes → InvalidArgument | ✓ SATISFIED | TestStore_valueLimit on all three write paths, check before any copy |
| CORE-06 | 02-01, 02-03 | LWW-Element-Map laws; order/duplicates/retransmissions cannot affect outcome | ✓ SATISFIED | TestMerge_laws/lawsTombstones/duplicateDelivery |
| CORE-07 | 02-01, 02-03 | Every displaced version recorded | ✓ SATISFIED | TestMerge_losers/losersProperty |
| VERS-01 | 02-02 | UUIDv7 byte order equals (l, c, NodeID) order | ✓ SATISFIED | TestVersion_layout/byteOrder |
| VERS-02 | 02-02 | Unique per write by construction | ✓ SATISFIED | TestVersion_uniquePerWrite; cross-restart uniqueness by seeded watermark dominance; no-wrap now ceiling-enforced and tested |
| VERS-03 | 02-02 | Monotonic under adversarial clock input | ✓ SATISFIED | TestClock_monotonic, counterOverflowBoundary, counterReset, ceilingSaturation |
| VERS-04 | 02-02, 02-04 | Restart seeds the dump's full watermark; every post-restart version strictly dominates every dumped version | ✓ SATISFIED (mechanism moved — see warning W-1) | clock.NewClock initialL/initialC seam + NewStore clock injection; TestClock_restartProperty full-watermark case; TestStore_restartDominance A/B. Substance fully satisfied; REQUIREMENTS.md still names the removed Config.InitialL/InitialC route |
| VERS-05 | 02-02 | Drift beyond drift_warn logged as warning | ✓ SATISFIED (engine side; logging deferred to Phase 6 OBS-01/03) | Signed drift signal returned by every clock-advancing op, both signs property-proven |
| TEST-01 | 02-03 | CRDT properties: commutative, associative, idempotent | ✓ SATISFIED | TestMerge_laws, TestMerge_lawsTombstones (500 seeded triples per mix) |
| TEST-02 | 02-02, 02-04 | HLC properties incl. counter-reset and restart-from-dump-max | ✓ SATISFIED | Six properties in clock/hlc_prop_test.go; restart case is the full-watermark model; ceiling case added |

Orphaned requirements: none — REQUIREMENTS.md maps exactly the 14 IDs to Phase 2, all claimed by plans (02-04 adds a second claim on VERS-02/VERS-04/TEST-02 for the gap closure). All 14 are marked Complete in the traceability table, consistent with this pass.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| storage/store_test.go | CORE-01..07, VERS-04 | 10 | 0 | 0 | Value/behavioral (exact Unpack triples, Compare results, error identity) | OK |
| storage/merge_test.go | TEST-01, CORE-06/07 | 3 | 0 | 0 | Value/behavioral (exact conflict sets, law equality) | OK |
| clock/clock_test.go | VERS-03/04/05 | 5 | 0 | 0 | Value (exact reading rows) | OK |
| clock/hlc_prop_test.go | TEST-02, VERS-02/03 | 6 | 0 | 0 | Behavioral (property walks, exact models) | OK |
| version/version_test.go | VERS-01 | 4 | 0 | 0 | Value (octet vectors) + behavioral (canonicalization property) | OK |

Disabled tests on requirements: 0. Circular patterns: 0 — the one candidate, `canonicalRaw`, is an independent RFC 9562 bit-mask reference (sets version/variant nibbles and zeroes filler per the spec's bit positions), not a re-run of the code under test; the generators in testutil produce inputs only. Expected-value provenance for the RFC 9562 layout vectors is the published RFC octet table (external standard), not system output. Assertion strength is value-level or behavioral on every requirement; no requirement is proven by existence-only assertions.

### Decision Coverage

All 4 trackable CONTEXT.md decisions honored by shipped artifacts (`check.decision-coverage-verify`: total 4, honored 4, not_honored []). Non-blocking by design.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| .planning/REQUIREMENTS.md | VERS-04 | Wording drift: names "via `Config.InitialL`/`InitialC`" — fields removed by the Clock-interface refactor; the watermark now seeds through `clock.NewClock(now, initialL, initialC, driftWarn)` composed into `NewStore` | ⚠️ Warning | None functional — the substance (full-watermark seeding, strict dominance) is implemented and tested. Phase 3 implementers reading VERS-04 literally will not find the Config fields; update the wording (or add a note) before Phase 3 wires dump loading |
| internal/app/internal/testutil/{rng,gen}.go | 35, 50, 36-37, 46, 59, 77 | Seven `//nolint` directives (gochecknoglobals x1, gosec x6) — prior state was zero; the split made testutil a normal package, so package-level-flag and integer-conversion linters now reach it | ℹ️ Info | All seven name their specific linter and carry a written reason, compliant with AGENTS.md deviation 3; no production suppressions |
| Prior open infos IN-01/02/03 | — | All three resolved this pass: dead constant deleted; ceiling saturation + test; real canonicalization property | ℹ️ Info (resolved) | Review disposition open:0/fixed:4 (02-REVIEW-DISPOSITION.md, 02-REVIEW-FIX.md) |

Zero debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) across internal/ — blocker-tier gate clean. Zero skipped tests. The `internal/port/doc.go` "Phase 1 skeleton placeholder" line predates this phase and is an honest documented skeleton, not a stub concealing behavior.

### Human Verification Required

None. This is an infrastructure/foundation phase (pure core library) with no user-facing elements, so manual UAT is N/a. Every behavior-dependent truth (restart dominance, ceiling no-wrap, merge laws, concurrency invariants) is exercised by a passing named test under the race detector — zero PRESENT_BEHAVIOR_UNVERIFIED truths — and the one backstop truth (integer-only HLC state) is a static structural claim fully settled by direct source inspection. The deferred VERS-05 logging half is Phase 6 scope by design (D-01), not a human-verification item.

### Gaps Summary

No gaps. The refactor and review fixes did not weaken any truth — every one of the 40 must-haves re-verified green against the new three-package layout: the full race suite passes at 100% per-package statement coverage, the test inventory grew by exactly one (the ceiling-saturation proof) with zero losses, the depguard boundary demonstrably fires on the subpackages, `go.mod`/`go.sum` are byte-identical to the last passed state, and `internal/port` is untouched. The previously falsified restart-dominance pair remains closed under the full-watermark model with value-level regression locks, and the prior pass's one coincidental-reliance item (no-wrap resting on an undeclared precondition) is resolved in the strongest form — a code-enforced ceiling with an adversarial-walk test. Two honest notes travel with the pass: plan 02-04's literal "Config.InitialC through NewStore" wiring was superseded by the clock-injection refactor (capability preserved, composition point moved to the caller, documented above), and REQUIREMENTS.md VERS-04 still carries the removed Config-field wording — a warning-tier documentation sync recommended before Phase 3 consumes the seam. The phase goal — pure I/O-free decision core, UUIDv7-packed HLC versions, LWW-Element-Map store, correctness laws property-proven by a seeded, replayable suite — is achieved in full.

---

_Verified: 2026-11-02T14:57:21Z_
_Verifier: Claude (gsd-verifier)_
