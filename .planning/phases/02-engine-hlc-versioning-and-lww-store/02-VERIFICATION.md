---
phase: 02-engine-hlc-versioning-and-lww-store
verified: 2026-11-14T19:03:16Z
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
covered_digest: "v4:sha256:95117661311f84b849a62dcb83c3f9b815fb79d7212d0c0b203aef678793000d"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 40/40
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Report regeneration #5. The prior pass (2026-11-03T14:16:17Z, at be149ce) was passed with no open gaps; this report was triggered because two covered files changed after it: commit c2746cb (phase 5-03, D-06) added the storage OnChange tap to internal/app/storage/store.go (+55/-16) and TestStore_onChange to store_test.go (+262/-0, purely additive). go.mod/go.sum also moved (phases 3-5 added the documented gRPC/protobuf tool/runtime requires and buf; .golangci.yml unchanged). The clock, version and testutil packages are byte-identical to the prior verified state (git diff be149ce..HEAD over the covered paths moves only the four files named). Every storage-package truth was re-verified in full against the changed code (source read line-by-line; the complete TestStore_*/TestMerge_* surface re-run green under -race; 100.0% statement coverage); clock/version truths were regression-checked with named-test runs. Depguard purity re-probed in this verification's own process. No regressions found."
deferred:
  - truth: "VERS-05's 'drift beyond drift_warn is logged as a warning' — the logging half (the engine-side signed drift signal is complete, property-proven, and overflow-safe since WR-01)"
    addressed_in: "Phase 6"
    evidence: "REQUIREMENTS.md OBS-01: 'A metrics endpoint exposes ... clock-drift warnings' (Phase 6, Pending); OBS-03: structured logging. D-01 forbids logging inside internal/app by design, so the logging consumer necessarily lands in a later phase."
---

# Phase 2: Engine — HLC Versioning and LWW Store Verification Report

**Phase Goal:** The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Verified:** 2026-11-14T19:03:16Z (at HEAD f5f9dac)
**Status:** passed
**Re-verification:** Yes — report regeneration #5 after the phase-5-03 storage OnChange tap (prior passed report 2026-11-03T14:16:17Z at be149ce; no gaps were open then)

## Change Delta Since the Prior Passed Report

Exactly one covered-code change set landed after the prior verification: **c2746cb (phase 5-03, decision D-06)** on `internal/app/storage/store.go` (+55/−16) and `store_test.go` (+262/−0):

1. **`Config.OnChange func(key string, rec Record)` change tap.** Nil-normalized to a no-op in `Config.normalized()` (store.go:102-104), fired from `install()` under the store write lock with the stored record (store.go:260). `write()` now routes local Set/SetIfAbsent/Delete through `install()` so every change source converges on the single tap point; remote merges already reached `install()` via the strictly-greater `join()` branch. Locked by `TestStore_onChange` (8 subtests: fires for every source; silent on a strictly-older loser and an equal-version redelivery — the D-06 regression locks; handed record asserted field-by-field; nil OnChange no-op).
2. **Phase-2 semantics unchanged by construction.** The LWW join law, the value-limit checks preceding every copy (store.go:152, 180, 212), the defensive copies, the tombstone rules and the Conflict return-value contract are byte-for-byte the same logic — the diff only extracts the store+checksum sequence into `install()` and adds the explicit, opt-in tap. The full storage named-test surface re-ran green under `-race` at 100.0% statement coverage.

`go.mod`/`go.sum` also changed since be149ce — the documented later-phase policy at work (AGENTS.md: gRPC/protobuf enter `go.mod` in the phases that first use them; `buf` joined the tool directives). Direct requires at HEAD are exactly `google.golang.org/grpc` and `google.golang.org/protobuf` — **no property-test or UUID library entered the build** (the phase-2 D-04 substance; see Prohibition Adjudication). `.golangci.yml` is unchanged, so the depguard boundary rules are identical to the prior verified state. The clock, version and testutil packages are byte-identical to the prior verified state (`git diff be149ce..HEAD` over the covered paths moves only the four files named above).

A fresh incremental code review ran today (02-REVIEW.md, depth standard, both changed files read in full): 0 critical, 0 warning, 3 info — all recorded open in 02-REVIEW-DISPOSITION.md, advisory only. These inform but do not replace this goal-backward verification.

## Goal Achievement

The goal holds at HEAD f5f9dac. `internal/app` remains a pure, I/O-free library (now six packages; the three phase-2 ones — `version`, `clock`, `storage` — plus test-only `internal/testutil` and the later-phase `persist`, `repl` additions): the depguard boundary `sup-kv-app-no-io` was re-probed in this verification's own process (a production blank `import "os"` in `internal/app/storage` fails lint with `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)`; lint green after removal, tree clean). Direct imports of the three phase-2 packages are stdlib + own module only (`go list -f '{{.Imports}}'`: version → bytes, encoding/binary; clock → math, sync, time; storage → bytes, errors, hash/crc32, sync, time, own version package). Purity greps find zero logging, zero `fmt.Print`, zero production `time.Now()` in the phase-2 production files. The HLC packs into RFC 9562 UUIDv7, the LWW-Element-Map store implements the full CORE semantics with exact loser accounting, and the correctness laws are property-proven by the seeded, replayable suite — green under the race detector at 100.0% statement coverage per production package, first-hand this pass.

### Observable Truths

Plan 02-01 (17 truths — all VERIFIED; this is the changed package: every truth re-verified in full against the OnChange-tap code with the complete named-test surface green under `-race` this pass):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Local write flows end-to-end: Set stamps fresh HLC version in UUIDv7 carrying NodeID; Get returns exact bytes + version | ✓ VERIFIED | `TestStore_endToEnd` green under `-race` this pass against the install-routed write path |
| 2 | Get returns ErrNotFound for missing and tombstoned keys (CORE-01) | ✓ VERIFIED | `TestStore_Get` subtests; store.go:137 sentinel return |
| 3 | Set replaces prior value under LWW, greater version wins (CORE-02) | ✓ VERIFIED | `TestStore_Set` replacement subtest |
| 4 | SetIfAbsent returns ErrAlreadyExists for any existing record incl. tombstone (CORE-03) | ✓ VERIFIED | `TestStore_SetIfAbsent` subtests incl. tombstoned-no-recreation |
| 5 | Delete writes fresh-version tombstone that stays in map and merges like any write (CORE-04) | ✓ VERIFIED | `TestStore_Delete`; `Records()` keeps tombstones; `TestMerge_lawsTombstones` |
| 6 | Values over ValueMaxBytes rejected with ErrValueTooLarge before any copy (CORE-05) | ✓ VERIFIED | `TestStore_valueLimit` 61440/61441 on set, setifabsent and merge; store.go:152/180/212 checks precede every copy — re-read on the changed source |
| 7 | Merge keeps remote version, displaces only on strictly greater full 16-byte version, returns every displaced/rejected version as Conflict (CORE-06/07) | ✓ VERIFIED | store.go join() `rec.Version.Compare(stored.Version)` on the full 16 bytes; `TestMerge_losers` + `TestMerge_losersProperty` green this pass |
| 8 | Two writes in same ms get distinct strictly increasing versions | ✓ VERIFIED | `TestStore_Set` same-ms subtest |
| 9 | Zero-length value is legal | ✓ VERIFIED | `TestStore_valueLimit` zero-length rows |
| 10 | Repeat writes never error, fresh version each time | ✓ VERIFIED | `TestStore_Set` repeat subtest |
| 11 | SetIfAbsent twice: first creates, second ErrAlreadyExists, first value kept | ✓ VERIFIED | `TestStore_SetIfAbsent` |
| 12 | Delete on missing/tombstoned key: ErrNotFound, no new tombstone, no version burned | ✓ VERIFIED | `TestStore_Delete` no-burn subtests; store.go:198 guards before write |
| 13 | Equal-version merge is a full no-op, no Conflict | ✓ VERIFIED | join() default branch returns nil; `TestMerge_losers` + `TestMerge_duplicateDelivery`; equal-version redelivery also locked tap-silent by `TestStore_onChange` |
| 14 | Merge into missing key installs, no loser reported | ✓ VERIFIED | join() install case; `TestMerge_losers` |
| 15 | No Conflict ever carries loser == winner | ✓ VERIFIED | `TestMerge_losersProperty` explicit assertion |
| 16 | Displacing-nothing merge returns empty conflict list, not error | ✓ VERIFIED | join() nil return; asserted across merge suite |
| 17 | Sentinel errors matched with errors.Is | ✓ VERIFIED | storage/errors.go 3 sentinels; tests import and use errors.Is |

Plan 02-02 (11 truths — all VERIFIED; clock/version/testutil files byte-identical to the prior verified state, regression-checked with named-test runs green under `-race` this pass):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Tick/Update follow Kulkarni rules, (l, counter) strictly increases on every call (VERS-03) | ✓ VERIFIED | `TestClock_tick`/`TestClock_update`/`TestClock_monotonic` — named regression run PASS this pass |
| 2 | Byte-wise compare equals (l, counter, NodeID) order vs field-wise reference model (VERS-01) | ✓ VERIFIED | `TestVersion_byteOrder` + `TestVersion_layout`; all `TestVersion_*` green this pass |
| 3 | 10,000 Ticks under stuck clock → 10,000 distinct strictly increasing versions (VERS-02) | ✓ VERIFIED | `TestVersion_uniquePerWrite`; ceiling-enforced and watermark-clamp-enforced end to end |
| 4 | Drift signal fires both directions with sign; always returned, never recorded (VERS-05, D-01) | ✓ VERIFIED | `TestClock_driftSignal` + `TestClock_driftUnderChaos` + `TestClock_driftSignalSaturates` — named regression run PASS this pass; zero logging in internal/app |
| 5 | Clock constructed with the dump's FULL watermark emits only versions strictly greater than EVERY dump version (VERS-04) | ✓ VERIFIED | `TestClock_restartProperty` + `TestClock_restartFromDumpMax` — named regression run PASS this pass |
| 6 | Concurrent Tick/Update: parallel mix, zero race findings | ✓ VERIFIED | `TestClock_monotonic` concurrent subtest; full clock package `-race` green this pass |
| 7 | Same remote version twice never decreases (l, c) — second Update no-op | ✓ VERIFIED | `TestClock_update` equal-timestamp rows |
| 8 | InitialL ahead of wall clock → first versions at/above InitialL; zero starts from wall clock | ✓ VERIFIED | `TestClock_restartFromDumpMax` first two subtests |
| 9 | Restart exact to the ms: every subsequent version strictly greater than every dump version, even under backward-jumped clock (VERS-04 precision) | ✓ VERIFIED | `TestStore_restartDominance` 4 subtests — re-run green this pass against the CHANGED store (the composition route `clock.NewClock` → `storage.NewStore` now also carries the install-routed write path) |
| 10 | Stuck clock: Ticks 1-4096 fill counters 0-4095, Tick 4097 → (l+1, 0); never wrap/repeat (TEST-02 boundary) | ✓ VERIFIED | `TestClock_counterOverflowBoundary` + `TestClock_ceilingSaturation` + `TestClock_watermarkClampedIntoDomain` — named regression run PASS this pass |
| 11 | HLC state is plain integer arithmetic — no time.Time held as state, no monotonic-read comparisons (Pitfall 8, backstop) | ✓ VERIFIED | Re-observed (file unchanged since prior verified state, where it was read in full): `Clock` fields `l int64`, `c uint16`, `now func() time.Time`, `driftWarn time.Duration`; Tick/Update reduce the seam to `UnixMilli()` int64 immediately |

Plan 02-03 (11 truths — all VERIFIED; storage is the changed package: full named-test re-verification green under `-race` this pass):

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

Plan 02-04 (3 gap-closure truths — truths 1-2 restate the historically falsified 02-02 pair and are deduplicated into the count via those rows; truth 3 re-verified against the changed store):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Clock constructed with the dump's full watermark emits only versions strictly greater than every dump version, incl. counter >= 1 at the watermark ms and the own NodeID (dedup of 02-02 #5) | ✓ VERIFIED | `TestClock_restartProperty` + `TestClock_restartFromDumpMax` named runs PASS this pass |
| 2 | First post-restart write never collides with or subordinates to the dump; peer accepts; stale watermark displaces with exactly one Conflict (dedup of 02-02 #9) | ✓ VERIFIED | `TestStore_restartDominance` re-run PASS this pass against the CHANGED store — 4 subtests with value-level `Unpack()`/`Compare` assertions, under `-race` |
| 3 | The restart seam expresses a counter seed reaching the clock, so the dump's full watermark is carryable; the now seam remains the single wall-clock input — the seed is constructor state, not a clock read (D-02) | ✓ VERIFIED | `clock.NewClock(now, initialL, initialC, driftWarn)` — the seed is constructor state masked/clamped on entry, never a clock read; `now` is still the single wall-clock seam. Composition route (caller builds `clock.NewClock(...)` → `storage.NewStore(nodeID, clk, cfg)`) exercised by `TestStore_restartDominance` and `testutil.RandStore` — both green this pass. REQUIREMENTS.md VERS-04 documents exactly this route (the plan's literal `Config.InitialL`/`InitialC` wording is the stale pre-refactor spelling; the superseding route is source-verified) |

**Score:** 40/40 truths verified (0 present, behavior-unverified). Denominator unchanged from the prior passed report: the 39 truths of plans 02-01..02-03 (17 + 11 + 11) plus 02-04's seam-expressiveness truth; 02-04's other two truths are deduplicated restatements of the 02-02 pair.

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | VERS-05's "logged as a warning" half — the engine returns the signed, overflow-safe drift signal (complete, property-proven); actual logging/metrics wiring | Phase 6 | REQUIREMENTS.md OBS-01 (clock-drift warnings in the metrics endpoint, Phase 6 Pending), OBS-03 (structured logging); D-01 forbids logging inside internal/app by design |

### Advisory (New Scope, Unevidenced)

Re-verification ran (report regeneration #5). No new-scope unevidenced findings this pass: the only covered-code change (the D-06 OnChange tap) is a documented phase-5 decision (05-CONTEXT.md D-06) fully evidenced by the passing `TestStore_onChange` regression locks run in this verification's own process; the go.mod/go.sum movement is the documented dependency policy (see Prohibition Adjudication). The three info-level review findings (IN-01/IN-02/IN-03) are evidenced doc-comment/test-hygiene observations, recorded open in 02-REVIEW-DISPOSITION.md — listed under Anti-Patterns below.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Required Artifacts

Every plan declares paths under the old single-package layout; the artifacts are verified at their post-split homes (the split predates the prior passed reports). Note: `gsd-tools verify.artifacts` reports 0/6 on 02-01 because the PLAN's literal paths (`internal/app/store.go` etc.) no longer exist — the same stale-path adjudication as the prior two passed reports; existence, substance and wiring below are verified manually at the current homes, with the two changed files read in full this pass.

| Plan path (stale) | New home | Expected | Status | Details |
| -------- | -------- | -------- | ------ | ------- |
| internal/app/errors.go | internal/app/storage/errors.go | Sentinels matched with errors.Is, RPC mapping docs | ✓ VERIFIED | 19 lines; ErrNotFound/ErrAlreadyExists/ErrValueTooLarge exported; unchanged since prior verified state |
| internal/app/version.go | internal/app/version/version.go | RecordVersion: pack, total unpack, byte-wise compare | ✓ VERIFIED | 65 lines; all four exports; byte-identical to prior verified state |
| internal/app/clock.go | internal/app/clock/clock.go | HLC with now seam, overflow rule, initial-l seam, drift signal | ✓ VERIFIED | 181 lines (WR-01/WR-02 state); byte-identical to prior verified state |
| internal/app/store.go | internal/app/storage/store.go | Config, Record, Conflict, RecordChecksum, Store API | ✓ VERIFIED | 279 lines (was 256; +23 net from the D-06 OnChange tap); all exports intact; read in full this pass — LWW semantics unchanged, write now converges on install() |
| internal/app/store_test.go | internal/app/storage/store_test.go | Tracer + CORE tables + merge unit tests | ✓ VERIFIED | 1163 lines (was 901); prior tests intact (diff is +262/−0) + `TestStore_onChange` with 8 subtests |
| internal/app/doc.go | internal/app/storage/doc.go | Package contract: LWW laws, visibility, D-01 contracts | ✓ VERIFIED | 37-line package doc; unchanged |
| internal/app/clock_test.go | internal/app/clock/clock_test.go | Kulkarni tables, overflow boundary, restart, drift | ✓ VERIFIED | 617 lines; byte-identical to prior verified state |
| internal/app/props_test.go | internal/app/internal/testutil/rng.go + gen.go | Seeded generators, printed seed, -seed replay | ✓ VERIFIED | 51 + 85 lines; `NewRNG` prints the seed and honors `-seed`; byte-identical to prior verified state |
| internal/app/version_test.go | internal/app/version/version_test.go | RFC 9562 vectors, round-trip, byte-order property | ✓ VERIFIED | 211 lines; byte-identical to prior verified state |
| internal/app/hlc_prop_test.go | internal/app/clock/hlc_prop_test.go | TEST-02 property suite | ✓ VERIFIED | 312 lines; six properties incl. `TestClock_restartProperty`; byte-identical to prior verified state |
| internal/app/merge_test.go (02-03) | internal/app/storage/merge_test.go | TEST-01 law suite | ✓ VERIFIED | 261 lines; byte-identical to prior verified state |

Wiring re-confirmed on the changed source: `storage` production calls `s.clock.Tick()` (store.go:168) and `s.clock.Update()` (store.go:219) through the consumer-side `storage.Clock` interface (store.go:67-70) and stamps/decides via `version.PackVersion` (store.go:169) / full-16-byte `Compare` (store.go:232); storage production imports only `bytes`, `errors`, `hash/crc32`, `sync`, `time`, `version` (go list verified). The store is consumed downstream by `persist.Engine` and, since phase 5, the OnChange tap is wired end-to-end (`cmd/sup-kv/main.go:279` passes `OnChange: hub.Ingest` into the store config through `persist.Open`) — the artifact is not orphaned at HEAD.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/app/storage/store.go (02-01) | clock Tick/Update | `s.clock.Tick()` / `s.clock.Update()` through `storage.Clock` | ✓ WIRED | store.go:168, store.go:219 — re-read on the changed source this pass |
| internal/app/storage/store.go (02-01) | version Pack/Compare | `version.PackVersion(l, counter, s.nodeID)`; join decides via full-16-byte `Compare` | ✓ WIRED | store.go:169, store.go:232 |
| storage tests (02-01) | exported storage API | external test packages drive only exported symbols with injected fakes | ✓ WIRED | imports confirmed in all test files; `TestStore_onChange` likewise drives only `Config.OnChange` + exported API |
| hlc_prop_test → generators (02-02) | testutil + same-package helpers | `testutil.NewRNG`, `nextWallTime` | ✓ WIRED | unchanged |
| version_test → version (02-02) | PackVersion/Unpack/Compare | layout vectors and canonicalization property | ✓ WIRED | unchanged |
| clock_test → clock (02-02) | NewClock via fake now (D-02) | every Kulkarni branch + WR regression paths | ✓ WIRED | unchanged |
| merge_test → testutil + storage (02-03) | RandStore/NewRNG; Merge/Records only | laws drive exported API; remote records keep their versions | ✓ WIRED | join() never re-stamps — re-read this pass |
| store_test → storage (02-03) | concurrent hammer on exported API | `sync.WaitGroup`/goroutines; race detector is the assertion | ✓ WIRED | `TestStore_concurrentHammer` + friends green this pass |
| store.go → clock.go (02-04) | caller composes `clock.NewClock(now, initialL, initialC, driftWarn)` into `NewStore(nodeID, clk, cfg)` | ✓ WIRED | Composition exercised by `TestStore_restartDominance` (PASS this pass against the changed store) and `testutil.RandStore`; the tool's literal `InitialC` pattern check reports the old pre-refactor route — the superseding constructor route is source-verified and documented in REQUIREMENTS.md VERS-04 |

### Data-Flow Trace (Level 4)

Not applicable in the render sense — a library with no UI/external data source. The equivalent holds on the changed source: every version flows from `clock.Tick/Update` through `version.PackVersion`; the restart seed flows from `NewClock`'s `initialL`/`initialC` into constructor state (never a clock read — D-02 preserved); the drift signal flows out of every clock-advancing op as a `time.Duration` return value (never recorded, never fed back); Merge decides via full 16-byte `Compare` on actual record versions; the OnChange tap hands the stored record computed by `install()` (defensive clone + recomputed checksum — real data, not a hardcoded payload). No hardcoded data paths anywhere in `internal/app`.

### Behavioral Spot-Checks

All commands run in this verification's own process at HEAD f5f9dac. Constraint honored: no full-workspace re-run; scoped named-test runs plus one phase-2-package coverage run.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full storage named-test surface (the changed package) | `go test -race -count=1 -run '^(TestStore_\|TestMerge_)' ./internal/app/storage` | `ok` 1.912s — all 16 test functions incl. `TestStore_onChange` (8 subtests), `TestMerge_laws*`, `TestStore_restartDominance`, concurrent hammer | ✓ PASS |
| D-06 regression locks on the new tap | included above: `TestStore_onChange` | PASS — fires per install for every source; silent on loser merge and equal-version redelivery; handed record equals Get/Records; nil OnChange no-op | ✓ PASS |
| Clock regression on unchanged package | `go test -race -count=1 -run '^(TestClock_restartProperty\|TestClock_monotonic\|TestClock_tick\|TestClock_update\|TestClock_driftSignal\|TestClock_watermarkClampedIntoDomain\|TestClock_driftSignalSaturates\|TestClock_counterOverflowBoundary)$' ./internal/app/clock` | `ok` 1.025s | ✓ PASS |
| Version regression on unchanged package | `go test -race -count=1 -run '^TestVersion_' ./internal/app/version` | `ok` 1.012s | ✓ PASS |
| Phase-2 package suite with coverage | `go test -race -count=1 -cover ./internal/app/clock/ ./internal/app/version/ ./internal/app/storage/` | ok clock 100.0%, version 100.0%, storage 100.0% of statements — the OnChange code included | ✓ PASS |
| Strict lint on phase-2 packages | `go tool golangci-lint run ./internal/app/clock/... ./internal/app/version/... ./internal/app/storage/... ./internal/app/internal/testutil/...` | exit 0, zero issues | ✓ PASS |
| Depguard boundary probe (own process) | inject blank `import "os"` into `internal/app/storage/store.go`, run scoped lint, remove | lint fails: `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)`; lint green after removal; tree clean | ✓ PASS |
| Purity greps | forbidden logging / `fmt.Print` / production `time.Now()` in the five phase-2 production files | none found (testutil's `time.Now().UnixNano()` is the test-seed source in a test-only-linked package) | ✓ PASS |
| Direct-import purity | `go list -f '{{.Imports}}'` on version/clock/storage | version: bytes, encoding/binary; clock: math, sync, time; storage: bytes, errors, hash/crc32, sync, time, own version — stdlib + own module only | ✓ PASS |
| Test-inventory continuity | `git show c2746cb -- store_test.go` + `grep -c '^func Test'` | store_test.go 11 → 12 functions; only delta `+TestStore_onChange`; zero deletions (diff +262/−0) | ✓ PASS |
| Dependency policy (D-04 substance) | direct `require` block in go.mod | exactly `google.golang.org/grpc` + `google.golang.org/protobuf` (phase-5 entries, documented policy); **no property-test or UUID library in the build** | ✓ PASS |
| Seed replay contract | testutil/rng.go source (byte-identical to prior verified state) | `NewRNG` prints `property seed: ...`; `-seed` flag registered | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist or were declared (no scripts directory). The phase's probe obligations are discharged by the named Go tests above plus the depguard boundary probe, executed in this verification's own process with the raw lint failure captured.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CORE-01 | 02-01 | Get: missing/tombstoned → NotFound | ✓ SATISFIED | TestStore_Get; storage/errors.go sentinel — green this pass |
| CORE-02 | 02-01 | Set create/replace under LWW | ✓ SATISFIED | TestStore_Set, TestStore_endToEnd — green this pass |
| CORE-03 | 02-01 | SetIfAbsent; existing (incl. tombstone) → AlreadyExists | ✓ SATISFIED | TestStore_SetIfAbsent, TestStore_concurrentSetIfAbsent |
| CORE-04 | 02-01 | Delete fresh-version tombstone, replicates like any write | ✓ SATISFIED | TestStore_Delete, TestMerge_lawsTombstones |
| CORE-05 | 02-01 | Values over value_max_bytes → InvalidArgument | ✓ SATISFIED | TestStore_valueLimit on all three write paths, check before any copy (store.go:152/180/212 re-read) |
| CORE-06 | 02-01, 02-03 | LWW-Element-Map laws; order/duplicates/retransmissions cannot affect outcome | ✓ SATISFIED | TestMerge_laws/lawsTombstones/duplicateDelivery — green this pass on the changed store |
| CORE-07 | 02-01, 02-03 | Every displaced version recorded | ✓ SATISFIED | TestMerge_losers/losersProperty |
| VERS-01 | 02-02 | UUIDv7 byte order equals (l, c, NodeID) order | ✓ SATISFIED | TestVersion_layout/byteOrder — green this pass |
| VERS-02 | 02-02 | Unique per write by construction | ✓ SATISFIED | TestVersion_uniquePerWrite; ceiling + watermark-clamp enforcement |
| VERS-03 | 02-02 | Monotonic under adversarial clock input | ✓ SATISFIED | TestClock_monotonic, counterOverflowBoundary, ceilingSaturation — green this pass |
| VERS-04 | 02-02, 02-04 | Restart seeds the dump's full watermark; every post-restart version strictly dominates every dumped version | ✓ SATISFIED | clock.NewClock initialL/initialC seam + NewStore clock injection; TestClock_restartProperty, TestClock_restartFromDumpMax, TestStore_restartDominance — all green this pass; REQUIREMENTS.md wording matches the implemented route |
| VERS-05 | 02-02 | Drift beyond drift_warn logged as warning | ✓ SATISFIED (engine side; logging deferred to Phase 6 OBS-01/03) | Signed saturated drift signal returned by every clock-advancing op; TestClock_driftSignal + driftUnderChaos + driftSignalSaturates green this pass |
| TEST-01 | 02-03 | CRDT properties: commutative, associative, idempotent | ✓ SATISFIED | TestMerge_laws, TestMerge_lawsTombstones (500 seeded triples per mix) — green this pass |
| TEST-02 | 02-02, 02-04 | HLC properties incl. counter-reset and restart-from-dump-max | ✓ SATISFIED | Six properties in clock/hlc_prop_test.go; ceiling and watermark-clamp cases both tested |

Orphaned requirements: none — REQUIREMENTS.md maps exactly the 14 IDs to Phase 2, all claimed by plans (02-01: CORE-01..07; 02-02: VERS-01..05 + TEST-02; 02-03: TEST-01 + CORE-06/07; 02-04: VERS-02/04 + TEST-02), all marked Complete in the traceability table, consistent with this pass.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers, no empty implementations, no console logging in any phase-2 file (changed files scanned in full this pass) | — | — |

From today's incremental code review (02-REVIEW.md → 02-REVIEW-DISPOSITION.md, all `open`, advisory only — none block the goal):

| File | Finding | Severity | Impact |
| ---- | ------- | -------- | ------ |
| internal/app/storage/store.go | IN-01: OnChange doc misstates the record a "subsequent Get returns" for tombstone installs (Get never returns tombstones) | ℹ️ Info | doc-comment accuracy only |
| internal/app/storage/store.go | IN-02: OnChange contract does not state the re-entrancy deadlock boundary | ℹ️ Info | doc-comment completeness only |
| internal/app/storage/store_test.go | IN-03: duplicated subtest body between TestStore_endToEnd and TestStore_Set | ℹ️ Info | test hygiene only |

Pre-existing (unchanged files, carried note): the six `//nolint` directives in `internal/app/internal/testutil/{gen,rng}.go` all name their specific linter (gosec G115/G404, gochecknoglobals) with written justifications, per project deviation 3 — compliant, verified by the clean strict-lint run.

### Prohibition Adjudication

- **02-01 "no hidden side effects… no logging, metrics, callbacks or accumulated events; losers and drift leave exclusively as return values (D-01)":** substance VERIFIED at HEAD — zero logging/metrics/accumulated events (greps clean); Merge still returns every loser as `[]Conflict` and drift still leaves exclusively as a return value; the default Config is tap-free (`nil` OnChange normalized to a no-op, store.go:102-104). One clause — "no callbacks" — is now qualified by a **documented later-phase decision**: phase 5-03 (c2746cb) added the explicit, opt-in `Config.OnChange` change tap under decision D-06 (05-CONTEXT.md: "Events fire only on actual state changes"), with the silence regression locks in `TestStore_onChange` green this pass. This is a deliberate cross-phase architecture extension (the `repl.Config.Log` precedent), not a phase-2 execution artifact and not a hidden side effect; it does not break the I/O-free guarantee the prohibition protects (depguard re-probed green). No override entry is created: the must-have's operative contract (return-value semantics, purity) holds, and the deviation belongs to phase 5's own verified scope.
- **02-01 "go.mod keeps zero direct runtime requires" (D-04):** substance VERIFIED — phase 2 added no dependency and no property-test/UUID library entered the build (direct requires at HEAD are exactly grpc + protobuf, both phase-5 entries under the AGENTS.md documented policy "gRPC and protobuf enter go.mod in the phases that first use them"). The zero-direct-requires letter described the phase-1/2 deliverable state and is superseded by the documented dependency policy, not by any phase-2 action.
- **02-01 re-stamping / truncation prohibitions, 02-02 and 02-03 seeded-randomness prohibitions:** VERIFIED — join() never re-stamps (re-read this pass); oversized values rejected whole before any copy; every generator seeds via `testutil.NewRNG` with printed seed and `-seed` replay (files unchanged since prior verified state).

### Human Verification Required

None. This is a pure library phase (infrastructure/foundation — no user-facing elements); every behavior-dependent truth is exercised by the seeded property/unit suite run green under `-race` in this verification's own process, including the changed install-routed write path and the D-06 tap locks. No ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths.

### Decision Coverage

Gate `check.decision-coverage-verify` ran (non-blocking): 4/4 trackable CONTEXT.md decisions honored by shipped artifacts; `not_honored` empty.

### Gaps Summary

No gaps. The single post-verification change to covered code (c2746cb, the D-06 OnChange tap) was re-verified in full: both changed files read line-by-line, the complete storage named-test surface re-run green under the race detector at 100.0% statement coverage, strict lint clean, the depguard purity gate re-probed in-process (denial captured), direct imports confirmed stdlib + own module only, and the test inventory confirmed purely additive (+262/−0). The go.mod/go.sum movement is the documented later-phase dependency policy — no property-test or UUID library entered the build. The "no callbacks" clause of one prohibition is qualified by the documented phase-5 D-06 decision (adjudicated above, not a phase-2 gap). All 14 requirement IDs are satisfied; the one deferred item (VERS-05's logging consumer) remains scheduled for Phase 6 by design.

---

_Verified: 2026-11-14T19:03:16Z_
_Verifier: Claude (gsd-verifier)_
