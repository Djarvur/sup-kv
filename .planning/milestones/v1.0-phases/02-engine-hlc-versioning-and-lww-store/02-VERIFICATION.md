---
phase: 02-engine-hlc-versioning-and-lww-store
verified: 2026-11-18T10:30:02Z
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
covered_digest: "v4:sha256:04d8ebdcdeb263fbc623681a205e8730d2bc5f8a0c899cfa4935e32c87ce0973"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 40/40
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Report regeneration #7. The prior pass (2026-11-16T11:04:05Z, at 9b87620) was passed with no open gaps; this report was triggered because covered inputs changed after it: phases 3-7 (v1.0.0 shipped at HEAD bd8d70b) moved go.mod (+295/-x) and go.sum (+1278/-x) — the go directive rose to 1.27.4 and two later-phase tool directives (grpcurl, goreleaser) pulled their indirect graphs in. Every covered .go file and .golangci.yml is byte-identical to the prior verified state (git diff 9b87620..HEAD over the covered paths moves only go.mod/go.sum, proven this pass). The complete phase-2 suite re-ran green under -race in this verification's own process at 100.0% statement coverage per package on the new toolchain and dependency graph; strict lint clean; the depguard purity boundary re-probed in-process (denial captured verbatim). Direct requires remain exactly grpc + protobuf — no property-test or UUID library entered the build (pgregory.net/rapid absent from go.mod entirely, honoring the discussion-log Option A: stdlib property tests, zero new test deps). No regressions found."
deferred:
  - truth: "VERS-05's 'drift beyond drift_warn is logged as a warning' — the logging half (the engine-side signed drift signal is complete, property-proven, and overflow-safe since WR-01)"
    addressed_in: "Phase 6"
    evidence: "REQUIREMENTS.md OBS-01: 'A metrics endpoint exposes ... clock-drift warnings' and OBS-03: structured logging — both marked Complete in the traceability table (Phase 6 delivered the slog pipeline and the metrics endpoint with the drift-warning gauge); D-01 forbids logging inside internal/app by design, so the logging consumer necessarily lives outside the phase-2 scope. The deferral has been discharged by Phase 6's own verified work."
---

# Phase 2: Engine — HLC Versioning and LWW Store Verification Report

**Phase Goal:** The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Verified:** 2026-11-18T10:30:02Z (at HEAD bd8d70b, v1.0.0)
**Status:** passed
**Re-verification:** Yes — report regeneration #7 (prior passed report 2026-11-16T11:04:05Z at 9b87620; no gaps were open then; this pass refreshes the covered-input fingerprint after phases 3-7 moved go.mod/go.sum and re-verifies the build/test/lint behavior on the new graph)

## Change Delta Since the Prior Passed Report

Exactly two covered inputs moved after 9b87620 (`git diff --name-only 9b87620..HEAD` over the covered paths returns only these two; every covered `.go` file and `.golangci.yml` is byte-identical):

1. **`go.mod` (net +295 lines)** — three distinct movements, all later-phase development-tooling scope, none phase-2 code:
   - The `go` directive rose from `go 1.27.1` + `toolchain go1.27.4` to the single `go 1.27.4` line (toolchain folded into the directive).
   - Two new `tool` directives: `github.com/fullstorydev/grpcurl/cmd/grpcurl` (phase 4/5 ops debugging per the stack decision "Enable reflection so grpcurl works") and `github.com/goreleaser/goreleaser/v2` (phase 7 release tooling, the stack's own "adopt at the v1.0 release phase" recommendation). Both are dev tools invoked via `go tool`, never linked into the shipped binary.
   - The `// indirect` block grew by goreleaser's large transitive graph (cloud SDKs, AWS modules, sprig, etc.).
2. **`go.sum` (net +1278 lines)** — the checksums of that indirect graph. No direct dependency moved.

**D-04 substance re-checked at HEAD:** the direct `require` block is still exactly `google.golang.org/grpc v1.85.0` + `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e` — unchanged from the prior verified state. `pgregory.net/rapid` is absent from go.mod entirely (grep this pass), honoring the phase-2 discussion log's Option A decision: stdlib property tests, zero new test dependencies. `github.com/google/uuid` and `github.com/stretchr/testify` appear only as `// indirect` entries of the dev-tool graphs (uuid was already in grpc's graph per the stack decision; neither is imported by any phase-2 file — direct-import check below proves stdlib + own module only).

Because no covered `.go` file changed, every prior line-level proof carries over; what had to be re-proven on the new graph — compilation, the full behavioral suite, strict lint, the depguard boundary — was re-run in this verification's own process (all green, below).

## Goal Achievement

The goal holds at HEAD bd8d70b. `internal/app` remains a pure, I/O-free library: the depguard boundary `sup-kv-app-no-io` was re-probed in this verification's own process — a scratch file in `internal/app/storage` carrying `import "os"` fails lint with `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)` (captured verbatim this pass; file deleted, tree clean). Direct imports of the three phase-2 packages are stdlib + own module only (`go list -f '{{.Imports}}'`: version → bytes, encoding/binary; clock → math, sync, time; storage → bytes, errors, hash/crc32, sync, time, own version package). Purity greps find zero logging, zero `fmt.Print`, zero production `time.Now()` in the phase-2 production files, and zero blank imports. The HLC packs into RFC 9562 UUIDv7, the LWW-Element-Map store implements the full CORE semantics with exact loser accounting, and the correctness laws are property-proven by the seeded, replayable suite — **the complete phase-2 suite (every test in clock, version, storage, no filter) ran green under the race detector at 100.0% statement coverage per package on the go 1.27.4 toolchain and the new dependency graph, first-hand this pass.**

### Observable Truths

Plan 02-01 (17 truths — all VERIFIED; every covered `.go` file byte-identical to the prior verified state, and the complete `TestStore_*`/`TestMerge_*` surface re-ran green under `-race` this pass):

| # | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | Local write flows end-to-end: Set stamps fresh HLC version in UUIDv7 carrying NodeID; Get returns exact bytes + version | ✓ VERIFIED | `TestStore_endToEnd` green under `-race` this pass (round-trip, UUIDv7 marker bits, replace-under-greater-version subtests) |
| 2 | Get returns ErrNotFound for missing and tombstoned keys (CORE-01) | ✓ VERIFIED | `TestStore_Get` subtests; errors.go:10 sentinel |
| 3 | Set replaces prior value under LWW, greater version wins (CORE-02) | ✓ VERIFIED | `TestStore_endToEnd` "a second set replaces the value under a strictly greater version" (store_test.go:106, re-located this pass) — second Set, `Compare` strictly greater, counter advance, Get returns "two"; green under `-race` |
| 4 | SetIfAbsent returns ErrAlreadyExists for any existing record incl. tombstone (CORE-03) | ✓ VERIFIED | `TestStore_SetIfAbsent` subtests incl. tombstoned-no-recreation |
| 5 | Delete writes fresh-version tombstone that stays in map and merges like any write (CORE-04) | ✓ VERIFIED | `TestStore_Delete` (store_test.go:348 live-key subtest re-located); `Records()` keeps tombstones; `TestMerge_lawsTombstones` |
| 6 | Values over ValueMaxBytes rejected with ErrValueTooLarge before any copy (CORE-05) | ✓ VERIFIED | `TestStore_valueLimit` on set, setifabsent and merge; store.go checks precede every copy (defaultValueMaxBytes 61440 at store.go:15, doc contract at :157-158) |
| 7 | Merge keeps remote version, displaces only on strictly greater full 16-byte version, returns every displaced/rejected version as Conflict (CORE-06/07) | ✓ VERIFIED | store.go join() `rec.Version.Compare(stored.Version)` on the full 16 bytes (store.go:240, re-located this pass); `TestMerge_losers` + `TestMerge_losersProperty` green this pass |
| 8 | Two writes in same ms get distinct strictly increasing versions | ✓ VERIFIED | `TestStore_Set` same-ms subtest (store_test.go:229) |
| 9 | Zero-length value is legal | ✓ VERIFIED | `TestStore_Set` zero-length subtest (store_test.go:252) |
| 10 | Repeat writes never error, fresh version each time | ✓ VERIFIED | `TestStore_Set` repeat subtest (store_test.go:267) |
| 11 | SetIfAbsent twice: first creates, second ErrAlreadyExists, first value kept | ✓ VERIFIED | `TestStore_SetIfAbsent` |
| 12 | Delete on missing/tombstoned key: ErrNotFound, no new tombstone, no version burned | ✓ VERIFIED | `TestStore_Delete` no-burn subtests |
| 13 | Equal-version merge is a full no-op, no Conflict | ✓ VERIFIED | join() default branch returns nil; `TestMerge_losers` + `TestMerge_duplicateDelivery`; equal-version redelivery also locked tap-silent by `TestStore_onChange` |
| 14 | Merge into missing key installs, no loser reported | ✓ VERIFIED | join() install case; `TestMerge_losers` |
| 15 | No Conflict ever carries loser == winner | ✓ VERIFIED | `TestMerge_losersProperty` explicit assertion |
| 16 | Displacing-nothing merge returns empty conflict list, not error | ✓ VERIFIED | join() nil return; asserted across merge suite |
| 17 | Sentinel errors matched with errors.Is | ✓ VERIFIED | storage/errors.go 3 sentinels (re-read this pass); tests import and use errors.Is |

Plan 02-02 (11 truths — all VERIFIED; files byte-identical to the prior verified state; the complete unfiltered package suites re-ran green under `-race` this pass via the coverage run):

| # | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | Tick/Update follow Kulkarni rules, (l, counter) strictly increases on every call (VERS-03) | ✓ VERIFIED | `TestClock_tick`/`TestClock_update`/`TestClock_monotonic` — all in the unfiltered green run this pass |
| 2 | Byte-wise compare equals (l, counter, NodeID) order vs field-wise reference model (VERS-01) | ✓ VERIFIED | `TestVersion_byteOrder` + `TestVersion_layout`; full version package `-race` green this pass |
| 3 | 10,000 Ticks under stuck clock → 10,000 distinct strictly increasing versions (VERS-02) | ✓ VERIFIED | `TestVersion_uniquePerWrite`; ceiling-enforced and watermark-clamp-enforced end to end |
| 4 | Drift signal fires both directions with sign; always returned, never recorded (VERS-05, D-01) | ✓ VERIFIED | `TestClock_driftSignal` + `TestClock_driftUnderChaos` + `TestClock_driftSignalSaturates` — green this pass; zero logging in internal/app (grep re-run) |
| 5 | Clock constructed with the dump's FULL watermark emits only versions strictly greater than EVERY dump version (VERS-04) | ✓ VERIFIED | `TestClock_restartProperty` (hlc_prop_test.go:238) + `TestClock_restartFromDumpMax` (clock_test.go:355) — green this pass |
| 6 | Concurrent Tick/Update: parallel mix, zero race findings | ✓ VERIFIED | `TestClock_monotonic` concurrent subtest; full clock package `-race` green this pass |
| 7 | Same remote version twice never decreases (l, c) — second Update no-op | ✓ VERIFIED | `TestClock_update` equal-timestamp rows |
| 8 | InitialL ahead of wall clock → first versions at/above InitialL; zero starts from wall clock | ✓ VERIFIED | `TestClock_restartFromDumpMax` first two subtests |
| 9 | Restart exact to the ms: every subsequent version strictly greater than every dump version, even under backward-jumped clock (VERS-04 precision) | ✓ VERIFIED | `TestStore_restartDominance` (store_test.go:586) 4 subtests — green this pass in the full storage run |
| 10 | Stuck clock: Ticks 1-4096 fill counters 0-4095, Tick 4097 → (l+1, 0); never wrap/repeat (TEST-02 boundary) | ✓ VERIFIED | `TestClock_counterOverflowBoundary` + `TestClock_ceilingSaturation` + `TestClock_watermarkClampedIntoDomain` — green this pass |
| 11 | HLC state is plain integer arithmetic — no time.Time held as state, no monotonic-read comparisons (Pitfall 8, backstop) | ✓ VERIFIED | Re-observed first-hand this pass on the byte-stable source: `Clock` fields are `l int64`, `c uint16`, `now func() time.Time`, `driftWarn time.Duration`; Tick/Update reduce the seam to `UnixMilli()` int64 immediately |

Plan 02-03 (11 truths — all VERIFIED; files byte-identical; complete package surface green under `-race` this pass):

| # | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | Merge commutative, associative, idempotent incl. tombstones over seeded random stores (TEST-01, CORE-06) | ✓ VERIFIED | `TestMerge_laws`/`TestMerge_lawsTombstones` — seeded triples per mix, green this pass |
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

Plan 02-04 (3 gap-closure truths — truths 1-2 restate the historically falsified 02-02 pair and are deduplicated into the count via those rows; truth 3 re-verified on the byte-stable clock source this pass):

| # | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | Clock constructed with the dump's full watermark emits only versions strictly greater than every dump version, incl. counter >= 1 at the watermark ms and the own NodeID (dedup of 02-02 #5) | ✓ VERIFIED | `TestClock_restartProperty` + `TestClock_restartFromDumpMax` green this pass |
| 2 | First post-restart write never collides with or subordinates to the dump; peer accepts; stale watermark displaces with exactly one Conflict (dedup of 02-02 #9) | ✓ VERIFIED | `TestStore_restartDominance` green this pass in the full storage surface run — 4 subtests with value-level `Unpack()`/`Compare` assertions, under `-race` |
| 3 | The restart seam expresses a counter seed reaching the clock, so the dump's full watermark is carryable; the now seam remains the single wall-clock input — the seed is constructor state, not a clock read (D-02) | ✓ VERIFIED | `clock.NewClock(now, initialL, initialC, driftWarn)` (clock.go:79, signature re-observed this pass) — the seed is constructor state masked/clamped on entry, never a clock read; `now` is still the single wall-clock seam. Composition route exercised by `TestStore_restartDominance` and the `newStoreFor` helper, both green this pass. REQUIREMENTS.md VERS-04 documents exactly this route (the plan's literal `Config.InitialL`/`InitialC` wording is the stale pre-refactor spelling; the superseding route is source-verified) |

**Score:** 40/40 truths verified (0 present, behavior-unverified). Denominator unchanged from the prior passed report: the 39 truths of plans 02-01..02-03 (17 + 11 + 11) plus 02-04's seam-expressiveness truth; 02-04's other two truths are deduplicated restatements of the 02-02 pair.

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | VERS-05's "logged as a warning" half — the engine returns the signed, overflow-safe drift signal (complete, property-proven); actual logging/metrics wiring | Phase 6 | REQUIREMENTS.md OBS-01 (clock-drift warnings in the metrics endpoint) and OBS-03 (structured logging) — both marked Complete in the traceability table: Phase 6 delivered the slog pipeline and the metrics endpoint. The deferral is discharged; D-01 keeps the logging consumer outside internal/app by design |

### Advisory (New Scope, Unevidenced)

Re-verification ran (report regeneration #7). No new-scope unevidenced findings this pass: the only covered-input changes are go.mod/go.sum movements from later phases' dev-tooling (grpcurl, goreleaser tool directives + their indirect graphs), with zero phase-2 code movement (git diff over covered `.go` paths is empty) and the direct require block unchanged. The dependency-graph growth is fully evidenced as tool-only by the `tool` directive block and the direct-import purity check.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Required Artifacts

Every plan declares paths under the old single-package layout; the artifacts are verified at their post-split homes (the split predates the prior passed reports). Note: `gsd-tools verify.artifacts` reports 0 passed on all four plans because the PLANs' literal paths (`internal/app/store.go` etc.) no longer exist — the same stale-path adjudication as the prior passed reports; existence, substance and wiring below are verified manually at the current homes (all line counts and test-function counts re-taken this pass; every file byte-identical to the prior verified state).

| Plan path (stale) | New home | Expected | Status | Details |
| -------- | -------- | -------- | ------ | ------- |
| internal/app/errors.go | internal/app/storage/errors.go | Sentinels matched with errors.Is, RPC mapping docs | ✓ VERIFIED | 19 lines; 3 sentinels re-read this pass; byte-stable |
| internal/app/version.go | internal/app/version/version.go | RecordVersion: pack, total unpack, byte-wise compare | ✓ VERIFIED | 65 lines; byte-identical to prior verified state |
| internal/app/clock.go | internal/app/clock/clock.go | HLC with now seam, overflow rule, initial-l seam, drift signal | ✓ VERIFIED | 181 lines; struct + NewClock signature re-observed this pass; byte-identical |
| internal/app/store.go | internal/app/storage/store.go | Config, Record, Conflict, RecordChecksum, Store API | ✓ VERIFIED | 287 lines; byte-identical to prior verified state; LWW semantics unchanged, all exports intact |
| internal/app/store_test.go | internal/app/storage/store_test.go | Tracer + CORE tables + merge unit tests | ✓ VERIFIED | 1144 lines, 12 test functions (count re-taken this pass); `TestStore_endToEnd` retains the replace-under-LWW pinning at :106 |
| internal/app/doc.go | internal/app/storage/doc.go | Package contract: LWW laws, visibility, D-01 contracts | ✓ VERIFIED | 37 lines; byte-stable |
| internal/app/clock_test.go | internal/app/clock/clock_test.go | Kulkarni tables, overflow boundary, restart, drift | ✓ VERIFIED | 617 lines, 8 test functions (re-counted); byte-identical |
| internal/app/props_test.go | internal/app/internal/testutil/rng.go + gen.go | Seeded generators, printed seed, -seed replay | ✓ VERIFIED | 51 + 85 lines; `NewRNG` prints the seed and the `-seed` flag is registered (rng.go:35, re-checked this pass); byte-identical |
| internal/app/version_test.go | internal/app/version/version_test.go | RFC 9562 vectors, round-trip, byte-order property | ✓ VERIFIED | 211 lines, 4 test functions; byte-identical |
| internal/app/hlc_prop_test.go | internal/app/clock/hlc_prop_test.go | TEST-02 property suite | ✓ VERIFIED | 312 lines, 6 test functions incl. `TestClock_restartProperty` at :238; byte-identical |
| internal/app/merge_test.go (02-03) | internal/app/storage/merge_test.go | TEST-01 law suite | ✓ VERIFIED | 261 lines, 4 test functions; byte-identical |

Wiring re-located on the current source this pass (line numbers exact — code byte-identical since the prior verified state): `storage` production calls `s.clock.Tick()` (store.go:176) and `s.clock.Update(remoteL, remoteC)` (store.go:227) through the consumer-side `storage.Clock` interface (store.go:71-77) and stamps/decides via `version.PackVersion` (store.go:177) / full-16-byte `Compare` (store.go:240); `Merge` at store.go:219 returns `(conflicts, drift, err)` — losers and drift leave exclusively as return values. Storage production imports only `bytes`, `errors`, `hash/crc32`, `sync`, `time`, `version` (go list verified this pass). The store is consumed downstream (persist.Engine and the replication engine; `OnChange` wired end-to-end since phase 5) — not orphaned at HEAD v1.0.0.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/app/storage/store.go (02-01) | clock Tick/Update | `s.clock.Tick()` / `s.clock.Update()` through `storage.Clock` | ✓ WIRED | store.go:176, store.go:227 — re-located on the current source this pass |
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

All commands run in this verification's own process at HEAD bd8d70b on the go 1.27.4 toolchain and the new dependency graph (the staleness-triggered re-proof). Constraint honored: one unfiltered phase-2-package coverage run (executes every test in the three packages), no full-workspace re-run.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Complete phase-2 suite with coverage (unfiltered — every test in the three packages) | `go test -race -count=1 -cover ./internal/app/clock/ ./internal/app/version/ ./internal/app/storage/` | ok clock 1.074s 100.0%, version 1.017s 100.0%, storage 1.586s 100.0% of statements | ✓ PASS |
| Strict lint on phase-2 packages (new lint-graph from new go.mod) | `go tool golangci-lint run ./internal/app/clock/... ./internal/app/version/... ./internal/app/storage/... ./internal/app/internal/testutil/...` | exit 0, zero issues | ✓ PASS |
| Depguard boundary probe (own process) | scratch file with `import "os"` + usage in internal/app/storage, scoped lint, delete | lint fails: `import 'os' is not allowed from list 'sup-kv-app-no-io': internal/app is the pure decision core - no filesystem (depguard)` — captured verbatim; tree clean after removal | ✓ PASS |
| Purity greps | forbidden logging / `fmt.Print` / production `time.Now()` / blank imports in the phase-2 production files | none found | ✓ PASS |
| Direct-import purity | `go list -f '{{.Imports}}'` on version/clock/storage | version: bytes, encoding/binary; clock: math, sync, time; storage: bytes, errors, hash/crc32, sync, time, own version — stdlib + own module only | ✓ PASS |
| Byte-identity of covered code | `git diff --name-only 9b87620..HEAD -- internal/app/clock internal/app/version internal/app/storage internal/app/internal/testutil .golangci.yml` | empty — every covered `.go` file and the lint config identical to the prior verified state | ✓ PASS |
| Dependency policy (D-04 substance) | direct `require` block in go.mod + grep for rapid/uuid/testify as direct | exactly `google.golang.org/grpc` + `google.golang.org/protobuf`; `pgregory.net/rapid` absent from go.mod entirely (Option A honored); uuid/testify only `// indirect` in tool graphs | ✓ PASS |
| Test-inventory continuity | `grep -c '^func Test'` across the five test files | store_test 12, merge_test 4, clock_test 8, hlc_prop 6, version_test 4 — counts identical to the prior verified state | ✓ PASS |
| Seed replay contract | testutil/rng.go source | `NewRNG` prints the seed; `-seed` flag registered (rng.go:35, nolint with named linter + justification) | ✓ PASS |
| Backstop truth source check | clock.go struct + seam reads | `Clock{mu, l int64, c uint16, now func() time.Time, driftWarn time.Duration}`; Tick/Update call `now().UnixMilli()` immediately | ✓ PASS |
| Wiring re-location | grep Tick/Update/PackVersion/Compare/NewClock in store.go + clock.go | Tick store.go:176, Update store.go:227, PackVersion store.go:177, Compare store.go:240, NewClock clock.go:79 | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist or were declared (no scripts directory). The phase's probe obligations are discharged by the named Go tests above plus the depguard boundary probe, executed in this verification's own process with the raw lint failure captured.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CORE-01 | 02-01 | Get: missing/tombstoned → NotFound | ✓ SATISFIED | TestStore_Get; storage/errors.go sentinel — green this pass |
| CORE-02 | 02-01 | Set create/replace under LWW | ✓ SATISFIED | TestStore_endToEnd replace-under-greater-version subtest (store_test.go:106), TestStore_Set remaining subtests — green this pass |
| CORE-03 | 02-01 | SetIfAbsent; existing (incl. tombstone) → AlreadyExists | ✓ SATISFIED | TestStore_SetIfAbsent, TestStore_concurrentSetIfAbsent |
| CORE-04 | 02-01 | Delete fresh-version tombstone, replicates like any write | ✓ SATISFIED | TestStore_Delete, TestMerge_lawsTombstones |
| CORE-05 | 02-01 | Values over value_max_bytes → InvalidArgument | ✓ SATISFIED | TestStore_valueLimit on all three write paths, check before any copy |
| CORE-06 | 02-01, 02-03 | LWW-Element-Map laws; order/duplicates/retransmissions cannot affect outcome | ✓ SATISFIED | TestMerge_laws/lawsTombstones/duplicateDelivery — green this pass |
| CORE-07 | 02-01, 02-03 | Every displaced version recorded | ✓ SATISFIED | TestMerge_losers/losersProperty |
| VERS-01 | 02-02 | UUIDv7 byte order equals (l, c, NodeID) order | ✓ SATISFIED | TestVersion_layout/byteOrder — green this pass |
| VERS-02 | 02-02 | Unique per write by construction | ✓ SATISFIED | TestVersion_uniquePerWrite; ceiling + watermark-clamp enforcement |
| VERS-03 | 02-02 | Monotonic under adversarial clock input | ✓ SATISFIED | TestClock_monotonic, counterOverflowBoundary, ceilingSaturation — green this pass |
| VERS-04 | 02-02, 02-04 | Restart seeds the dump's full watermark; every post-restart version strictly dominates every dumped version | ✓ SATISFIED | clock.NewClock initialL/initialC seam + NewStore clock injection; TestClock_restartProperty, TestClock_restartFromDumpMax, TestStore_restartDominance — all green this pass; REQUIREMENTS.md wording matches the implemented route |
| VERS-05 | 02-02 | Drift beyond drift_warn logged as warning | ✓ SATISFIED (engine side; logging consumer delivered by Phase 6 — see Deferred) | Signed saturated drift signal returned by every clock-advancing op; TestClock_driftSignal + driftUnderChaos + driftSignalSaturates green this pass |
| TEST-01 | 02-03 | CRDT properties: commutative, associative, idempotent | ✓ SATISFIED | TestMerge_laws, TestMerge_lawsTombstones (seeded triples per mix) — green this pass |
| TEST-02 | 02-02, 02-04 | HLC properties incl. counter-reset and restart-from-dump-max | ✓ SATISFIED | Six properties in clock/hlc_prop_test.go; ceiling and watermark-clamp cases both tested |

Orphaned requirements: none — REQUIREMENTS.md maps exactly the 14 IDs to Phase 2, all claimed by plans (02-01: CORE-01..07; 02-02: VERS-01..05 + TEST-02; 02-03: TEST-01 + CORE-06/07; 02-04: VERS-02/04 + TEST-02), all marked Complete in the traceability table, consistent with this pass.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers, no empty implementations, no console logging in any phase-2 file (full scan this pass) | — | — |

Pre-existing (unchanged files, carried note): the `//nolint` directives in `internal/app/internal/testutil/{gen,rng}.go` all name their specific linter (gosec G115/G404, gochecknoglobals) with written justifications, per project deviation 3 — compliant, verified by the clean strict-lint run on the new graph. Zero nolint in phase-2 production files (grepped this pass).

### Prohibition Adjudication

- **02-01 "no hidden side effects… no logging, metrics, callbacks or accumulated events inside internal/app; losers and drift leave exclusively as return values (D-01)":** substance VERIFIED at HEAD — zero logging/metrics/accumulated events (greps clean this pass); Merge still returns every loser as `[]Conflict` and drift still leaves exclusively as a return value; the default Config is tap-free (`nil` OnChange normalized to a no-op). One clause — "no callbacks" — remains qualified by the documented later-phase decisions phase 5-03 (D-06, the explicit opt-in `Config.OnChange` tap) and phase 6-02 (D-15, the tap's doc-contract refinement, comment-only). This is a deliberate cross-phase architecture extension, not a phase-2 execution artifact and not a hidden side effect; it does not break the I/O-free guarantee the prohibition protects (depguard re-probed green in-process this pass). No override entry is created: the must-have's operative contract (return-value semantics, purity) holds, and the deviation belongs to the later phases' own verified scopes.
- **02-01 "go.mod keeps zero direct runtime requires" (D-04):** substance VERIFIED — phase 2 added no dependency and no property-test/UUID library entered the build (direct requires at HEAD are exactly grpc + protobuf, both later-phase entries under the AGENTS.md documented policy "gRPC and protobuf enter go.mod in the phases that first use them"; `pgregory.net/rapid` absent from go.mod entirely, honoring the discussion-log Option A — stdlib property tests, zero new test deps; the go.mod growth since the prior pass is tool directives (grpcurl, goreleaser) and their indirect graphs only, re-checked this pass). The zero-direct-requires letter described the phase-1/2 deliverable state and is superseded by the documented dependency policy, not by any phase-2 action.
- **02-01 re-stamping / truncation prohibitions, 02-02 and 02-03 seeded-randomness prohibitions:** VERIFIED — join() never re-stamps (code byte-identical this pass); oversized values rejected whole before any copy; every generator seeds via `testutil.NewRNG` with printed seed and `-seed` replay (files byte-stable since the prior verified state, source re-checked this pass).

### Human Verification Required

None. This is a pure library phase (infrastructure/foundation — no user-facing elements); every behavior-dependent truth is exercised by the seeded property/unit suite run green under `-race` in this verification's own process on the new toolchain and dependency graph — including the unfiltered per-package coverage run that executed every phase-2 test. No ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths.

### Gaps Summary

No gaps. The staleness trigger was fully diagnosed: only go.mod/go.sum moved since the prior passed report (go 1.27.4 directive + two later-phase dev-tool directives and their indirect graphs), with every covered `.go` file and the lint config byte-identical (git diff empty over those paths). What the new graph could have broken was re-proven in this verification's own process: the complete phase-2 suite green under the race detector at 100.0% statement coverage per package, strict lint clean, the depguard purity gate re-probed (denial captured verbatim), direct imports still stdlib + own module only, and the direct require block still exactly grpc + protobuf — with `pgregory.net/rapid` absent entirely per the discussion-log Option A decision. All 14 requirement IDs are satisfied; the one deferred item (VERS-05's logging consumer) has been discharged by Phase 6's own completed, verified work.

---

_Verified: 2026-11-18T10:30:02Z_
_Verifier: Claude (gsd-verifier)_
