---
phase: 02-engine-hlc-versioning-and-lww-store
verified: 2026-11-02T02:53:24Z
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
  - internal/app/clock.go
  - internal/app/clock_test.go
  - internal/app/doc.go
  - internal/app/errors.go
  - internal/app/hlc_prop_test.go
  - internal/app/merge_test.go
  - internal/app/props_test.go
  - internal/app/store.go
  - internal/app/store_test.go
  - internal/app/version.go
  - internal/app/version_test.go
covered_digest: "v4:sha256:60e96819a878d8f5aff402fda44ce48cb8a2c3763a8822f1a88e0d6822753db2"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 37/39
  gaps_closed:
    - "G-02-VERS04-DOMINANCE — a clock constructed with the dump's full watermark (NewClock now, dumpMax, watermarkC, driftWarn) emits only versions strictly greater than every version in that dump, including watermark-millisecond versions with counter >= 1 and the store's own NodeID (02-02 truth 5, restated as 02-04 truth 1)"
    - "Restart-exact dominance — every subsequent version strictly greater than every dump version even under a backward-jumped clock, and the first post-restart write never collides with or subordinates to the dump: the peer-accept (variant A) and no-collision (variant B) probe failures are dead (02-02 truth 9, restated as 02-04 truth 2)"
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "VERS-05's 'drift beyond drift_warn is logged as a warning' — the logging half (the engine-side signed drift signal is complete and property-proven)"
    addressed_in: "Phase 6"
    evidence: "REQUIREMENTS.md OBS-01: 'A metrics endpoint exposes ... clock-drift warnings'; OBS-03: structured logging. D-01 forbids logging inside internal/app by design, so the logging consumer necessarily lands in a later phase."
coincidental_reliance_items:
  - truth: "Versions never wrap or repeat (TEST-02 4096/4097 overflow boundary)"
    reason: undeclared-precondition
    harden: "The no-wrap clause holds because l stays below the 48-bit ceiling (2^48 ms ~ year 8921), which nothing declares as a precondition on the injected now func. At l = 2^48-1 with counter 4095 the bump produces l = 2^48 and PackVersion's tsMask wraps the version to zero (review IN-02, still open info). Declare 'l < 2^48-1 is a caller precondition on now' in NewClock's doc, or clamp at the ceiling."
---

# Phase 2: Engine — HLC Versioning and LWW Store Verification Report

**Phase Goal:** The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Verified:** 2026-11-02T02:53:24Z
**Status:** passed
**Re-verification:** Yes — after gap closure (previous report 2026-11-02T02:04:55Z, status gaps_found, 37/39; gap G-02-VERS04-DOMINANCE closed by plan 02-04, commits a5a78a5..b4b5397, delta review 0c7ba3c)

## Goal Achievement

The goal is achieved. `internal/app` is a pure, I/O-free library (depguard-enforced; zero `net`/`os`/`syscall`/`google.golang.org` imports, zero logging), the HLC packs into RFC 9562 UUIDv7 (octet-exact vector proven), the LWW-Element-Map store implements the full CORE semantics with exact loser accounting, and the correctness laws are property-proven by a seeded, replayable suite — green under the race detector at 100% statement coverage. The single gap from the initial verification (G-02-VERS04-DOMINANCE / review WR-01: the restart seam seeded only `l`) is closed: `NewClock` gained the `initialC uint16` counter seed (masked to the 12-bit domain), `Config.InitialC` propagates through `NewStore` at the single composition point, Tick/Update stayed byte-identical (git diff confirms only the constructor signature, seeding body and docs changed), and the two falsified truths are now backed by named passing tests over the hazardous (dumpMax, counter >= 1, own NodeID) watermark — with a mutation check proving the new tests genuinely bind on the seed.

### Observable Truths

Plan 02-01 (17 truths — all VERIFIED; quick regression: full suite green under `go test -race -count=1` at 100% coverage, all named tests present, no test functions deleted since the prior verified state):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Local write flows end-to-end: Set stamps fresh HLC version in UUIDv7 carrying NodeID; Get returns exact bytes + version | ✓ VERIFIED | `TestStore_endToEnd`; suite green |
| 2 | Get returns ErrNotFound for missing and tombstoned keys (CORE-01) | ✓ VERIFIED | `TestStore_Get` subtests |
| 3 | Set replaces prior value under LWW, greater version wins (CORE-02) | ✓ VERIFIED | `TestStore_Set` replacement subtest |
| 4 | SetIfAbsent returns ErrAlreadyExists for any existing record incl. tombstone (CORE-03) | ✓ VERIFIED | `TestStore_SetIfAbsent` subtests |
| 5 | Delete writes fresh-version tombstone that stays in map and merges like any write (CORE-04) | ✓ VERIFIED | `TestStore_Delete`; `Records()` keeps tombstones; `TestMerge_lawsTombstones` |
| 6 | Values over ValueMaxBytes rejected with ErrValueTooLarge before any copy (CORE-05) | ✓ VERIFIED | `TestStore_valueLimit` 61440/61441 on set, setifabsent and merge |
| 7 | Merge keeps remote version, displaces only on strictly greater full 16-byte version, returns every displaced/rejected version as Conflict (CORE-06/07) | ✓ VERIFIED | store.go join() full-byte Compare; `TestMerge_losers` + `TestMerge_losersProperty` |
| 8 | Two writes in same ms get distinct strictly increasing versions | ✓ VERIFIED | `TestStore_Set` same-ms subtest |
| 9 | Zero-length value is legal | ✓ VERIFIED | `TestStore_valueLimit` zero-length rows |
| 10 | Repeat writes never error, fresh version each time | ✓ VERIFIED | `TestStore_Set` repeat subtest |
| 11 | SetIfAbsent twice: first creates, second ErrAlreadyExists, first value kept | ✓ VERIFIED | `TestStore_SetIfAbsent` |
| 12 | Delete on missing/tombstoned key: ErrNotFound, no new tombstone, no version burned | ✓ VERIFIED | `TestStore_Delete` no-burn subtests |
| 13 | Equal-version merge is a full no-op, no Conflict | ✓ VERIFIED | `TestMerge_losers` + `TestMerge_duplicateDelivery` |
| 14 | Merge into missing key installs, no loser reported | ✓ VERIFIED | `TestMerge_losers` install case |
| 15 | No Conflict ever carries loser == winner | ✓ VERIFIED | `TestMerge_losersProperty` explicit assertion |
| 16 | Displacing-nothing merge returns empty conflict list, not error | ✓ VERIFIED | join() nil return; asserted across merge suite |
| 17 | Sentinel errors matched with errors.Is | ✓ VERIFIED | errors.go 3 sentinels; tests use errors.Is |

Plan 02-02 (11 truths — all VERIFIED; truths 5 and 9 are the previously falsified pair, this pass fully re-verified with behavioral evidence):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Tick/Update follow Kulkarni rules, (l, counter) strictly increases on every call (VERS-03) | ✓ VERIFIED | `TestClock_tick`/`TestClock_update`/`TestClock_monotonic`; suite green under `-race` |
| 2 | Byte-wise compare equals (l, counter, NodeID) order vs field-wise reference model (VERS-01) | ✓ VERIFIED | `TestVersion_byteOrder` (1000 seeded triples) + `TestVersion_layout` octet vector |
| 3 | 10,000 Ticks under stuck clock → 10,000 distinct strictly increasing versions (VERS-02) | ✓ VERIFIED | `TestVersion_uniquePerWrite` |
| 4 | Drift signal fires both directions with sign; always returned, never recorded (VERS-05, D-01) | ✓ VERIFIED | `TestClock_driftSignal`, `TestClock_driftUnderChaos`; zero logging in internal/app |
| 5 | Clock constructed with the dump's max l — now the dump's FULL watermark — emits only versions strictly greater than EVERY version in that dump (VERS-04) | ✓ VERIFIED (was FAILED 2026-11-02T02:04:55Z) | CLOSED by 02-04: `NewClock(now, initialL, initialC, driftWarn)` (clock.go:39-40) seeds and masks the counter; `TestClock_restartProperty` proves across a 100-tick adversarial walk that every packed emission Compare-returns newer against every dump record including the (dumpMax, watermarkC 1..4095, testNodeID) watermark — run green under `-race`; mutation check below proves the test binds on the seed |
| 6 | Concurrent Tick/Update: parallel mix, zero race findings | ✓ VERIFIED | `TestClock_monotonic` concurrent subtest; suite `-race` green |
| 7 | Same remote version twice never decreases (l, c) — second Update no-op | ✓ VERIFIED | `TestClock_update` equal-timestamp branch rows |
| 8 | InitialL ahead of wall clock → first versions at/above InitialL; InitialL zero starts from wall clock | ✓ VERIFIED | `TestClock_restartFromDumpMax` first two subtests |
| 9 | Restart exact to the ms: every subsequent version strictly greater than every dump version, even under backward-jumped clock (VERS-04 precision) | ✓ VERIFIED (was FAILED 2026-11-02T02:04:55Z) | CLOSED by 02-04: `TestClock_restartProperty` starts the wall clock hours before dumpMax and walks adversarially — every emission beats every dump version; `TestStore_restartDominance` regressions prove the store-level consequences dead: variant A (peer holding the dump watermark accepts the first post-restart write, exactly one Conflict{Winner: fresh}, Get returns fresh data) and variant B (stale watermark record displaces with exactly one Conflict, never an equal-version no-op) — both green under `-race` with the wall clock parked exactly on the watermark millisecond |
| 10 | Stuck clock: Ticks 1-4096 fill counters 0-4095, Tick 4097 → (l+1, 0); never wrap/repeat (TEST-02 boundary) | ✓ VERIFIED (coincidental-reliance) | `TestClock_counterOverflowBoundary`; no-wrap clause still rests on the undeclared l < 2^48-1 now-precondition (IN-02 open info) — see coincidental_reliance_items |
| 11 | HLC state is plain integer arithmetic — no time.Time held as state, no monotonic-read comparisons (Pitfall 8, backstop) | ✓ VERIFIED | Re-confirmed on the current tree: Clock state is `l int64`, `c uint16`; the 02-04 delta touched only the constructor and docs (git diff); Tick/Update bodies byte-identical; zero time-comparison methods, zero `time.Now()` in production code |

Plan 02-03 (11 truths — all VERIFIED; quick regression: suite green under `-race`, no test deletions, no production-code changes to merge paths since the prior verified state):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Merge commutative, associative, idempotent incl. tombstones over seeded random stores (TEST-01, CORE-06) | ✓ VERIFIED | `TestMerge_laws`/`TestMerge_lawsTombstones` — 500 seeded triples per mix |
| 2 | Every displaced/rejected version returned exactly once as Conflict paired with displacer (CORE-07) | ✓ VERIFIED | `TestMerge_losersProperty` both directions |
| 3 | Delivery order, duplicates, retransmissions cannot affect outcome | ✓ VERIFIED | Commutativity + `TestMerge_duplicateDelivery` |
| 4 | Parallel Set/SetIfAbsent/Delete/Merge/Get: zero race findings | ✓ VERIFIED | `TestStore_concurrentHammer`; suite `-race` green |
| 5 | Concurrent SetIfAbsent on one key: exactly one succeeds | ✓ VERIFIED | `TestStore_concurrentSetIfAbsent` |
| 6 | Delete racing Set: survivor holds exactly the greater captured version | ✓ VERIFIED | `TestStore_concurrentDeleteVsSet` |
| 7 | merge(a,b) and merge(b,a): identical state AND identical conflict sets | ✓ VERIFIED | `assertMergeLaws` commutativity block |
| 8 | Every displaced version exactly once in conflicts, paired with displacer | ✓ VERIFIED | `TestMerge_losersProperty` exact expected set |
| 9 | merge(a,a) idempotent: zero conflicts, nothing changes | ✓ VERIFIED | `assertMergeLaws` idempotence block |
| 10 | Laws hold for empty stores | ✓ VERIFIED | empty-source clones + missing-key install cases |
| 11 | merge(merge(a,b),c) equals merge(a,merge(b,c)) incl. tombstoned keys | ✓ VERIFIED | `assertMergeLaws` associativity block |

Plan 02-04 (3 gap-closure truths — full verification this pass; truths 1-2 restate and close the previously falsified 02-02 pair and are deduplicated into the count via those rows, truth 3 is the new seam-expressiveness must-have):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Clock constructed with the dump's full watermark emits only versions strictly greater than every dump version, incl. counter >= 1 at the watermark ms and the own NodeID (dedup of 02-02 #5) | ✓ VERIFIED | `TestClock_restartProperty` (watermarkC 1..4095 seeded, own testNodeID, 100-tick walk, strict Compare dominance) + 4 new `TestClock_restartFromDumpMax` seed rows — all green under `-race`; mutation-verified binding |
| 2 | First post-restart write never collides with or subordinates to the dump; peer accepts; stale watermark displaces with exactly one Conflict (dedup of 02-02 #9) | ✓ VERIFIED | `TestStore_restartDominance` variants A and B — green under `-race`, exact Conflict{Key, Winner, Loser} assertions, fresh data survives on both sides |
| 3 | The restart seam expresses a counter seed: Config.InitialC uint16 reaches the clock through NewStore into NewClock; the now seam remains the single wall-clock input (D-02) | ✓ VERIFIED | clock.go:39-40 (`initialC uint16` param, `initialC & counterMask` masking); store.go:32 (`InitialC uint16` after InitialL), store.go:75 (`NewClock(now, cfg.InitialL, cfg.InitialC, cfg.DriftWarn)` — the single composition point); `verify.key-links` 1/1 verified; no new clock reads added — the seed is constructor state |

**Score:** 40/40 truths verified (0 present, behavior-unverified). Denominator: the 39 truths of plans 02-01..02-03 (17 + 11 + 11, continuity with the prior 37/39) plus 02-04's seam-expressiveness truth; 02-04's other two truths are deduplicated restatements of the previously falsified 02-02 pair.

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | VERS-05's "logged as a warning" half — the engine returns the signed drift signal (complete, property-proven); actual logging/metrics wiring | Phase 6 | REQUIREMENTS.md OBS-01 (clock-drift warnings in the metrics endpoint), OBS-03 (structured logging); D-01 forbids logging inside internal/app by design |

### Advisory (New Scope, Unevidenced)

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None — the 02-04 delta (5 files, +237/−33) was fully evidenced: seam grep gates, named tests, mutation check; no new-scope unevidenced findings | — | — |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| internal/app/clock.go | NewClock extended with initialC uint16 (masked); Tick/Update unchanged | ✓ VERIFIED | 4-param NewClock (line 39), `initialC & counterMask` (line 40); git diff vs eb5aa55 shows Tick/Update bodies untouched; doc contract states full-watermark restart and that initialL alone is insufficient |
| internal/app/store.go | Config.InitialC propagated by NewStore; full-watermark doc | ✓ VERIFIED | Config.InitialC (line 32, after InitialL); pass-through at line 75; doc rewritten |
| internal/app/clock_test.go | TestClock_restartFromDumpMax + 4 seed subtests | ✓ VERIFIED | 6 subtests total: (T,6)/(T,7) first ticks, counterMax bump (T+1,0), wall-ahead (pt,0), 4101 masked to 5 |
| internal/app/store_test.go | TestStore_restartDominance: Config rows + variant A/B regressions | ✓ VERIFIED | 4 subtests (587-698): two seed rows, peer-accept, no-collision |
| internal/app/hlc_prop_test.go | TestClock_restartProperty over (dumpMax, c>0, ownNodeID) watermark | ✓ VERIFIED | watermarkC 1..4095 from seeded rng, never 0; doc records the supersession of the counter-0 narrowing |
| internal/app/errors.go, version.go, doc.go, props_test.go, version_test.go, merge_test.go | unchanged since prior verified state | ✓ VERIFIED | git diff eb5aa55..HEAD touches exactly the 5 declared 02-04 files |

`gsd-tools verify.artifacts`: 02-02 (5/5), 02-03 (2/2), 02-04 (5/5) all passed; 02-01 reports "Missing export" only for methods (`Clock.Tick/Update`, `Store.Get/Set/SetIfAbsent/Delete/Merge/Records`, `RecordVersion.Unpack/Compare`) — the known tool limitation (package-level functions only); all methods confirmed present by direct inspection and by the compiling, passing suite.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/app/store.go | internal/app/clock.go | NewStore passes cfg.InitialC into NewClock's initialC — single composition point | ✓ WIRED | `verify.key-links` 1/1 verified (02-04); store.go:75 confirmed in source |
| store.go → clock.go (Tick/Update), store.go → version.go (Pack/Compare), test→API links (plans 02-01..02-03) | | | ✓ WIRED | `verify.key-links` 3/3 + 3/3 + 3/3 verified; unchanged since prior verified state |

### Data-Flow Trace (Level 4)

Not applicable in the render sense — a library with no UI/external data source. The equivalent check holds: every version flows from clock.Tick/Update through PackVersion; the restart seed flows from Config.InitialC through NewStore into clock state (constructor state, not a clock read — D-02 preserved); Merge decides via full 16-byte Compare on actual record versions. No hardcoded data paths.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Named restart tests (previously-failed truths) | `go test -race -count=1 ./internal/app/ -run '^TestClock_restartProperty$\|^TestStore_restartDominance$\|^TestClock_restartFromDumpMax$' -v` | all PASS (property + 4 store subtests + 6 unit subtests), exit 0 | ✓ PASS |
| Mutation check — tests bind on the seed | seed stripped (`c: initialC & counterMask` → `c: 0`), same three tests | all three FAIL as expected; source restored clean (`git diff --quiet` after restore) | ✓ PASS (binding proven — the regressions are not vacuous) |
| Full suite regression | `go test -race -count=1 -cover ./...` | ok, coverage 100.0% of statements, exit 0 | ✓ PASS |
| Seed replay contract | `TestClock_restartProperty -v` | prints `property seed: ...` with replay invocation | ✓ PASS |
| Build + vet | `go build ./... && go vet ./...` | exit 0 | ✓ PASS |
| Strict lint | `go tool golangci-lint run ./...` | exit 0, zero issues | ✓ PASS |
| Zero-dependency policy (D-04) | `git diff --quiet eb5aa55..HEAD -- go.mod go.sum` | identical | ✓ PASS |
| internal/port untouched (D-03) | `ls internal/port` | doc.go only | ✓ PASS |
| No test functions deleted | `git diff eb5aa55..HEAD -- internal/app/` grep deleted funcs | only the old 3-param NewClock replaced; test counts 5/6/4/11/4 (+1 = TestStore_restartDominance) | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist or were declared; the phase's probe obligations are discharged by the named Go tests above (the prior run's ad-hoc WR-01 probes are now permanent regression rows in TestStore_restartDominance). Not applicable.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CORE-01 | 02-01 | Get: missing/tombstoned → NotFound | ✓ SATISFIED | TestStore_Get; errors.go sentinel |
| CORE-02 | 02-01 | Set create/replace under LWW | ✓ SATISFIED | TestStore_Set, TestStore_endToEnd |
| CORE-03 | 02-01 | SetIfAbsent; existing (incl. tombstone) → AlreadyExists | ✓ SATISFIED | TestStore_SetIfAbsent, TestStore_concurrentSetIfAbsent |
| CORE-04 | 02-01 | Delete fresh-version tombstone, replicates like any write | ✓ SATISFIED | TestStore_Delete, TestMerge_lawsTombstones |
| CORE-05 | 02-01 | Values over value_max_bytes → InvalidArgument | ✓ SATISFIED | TestStore_valueLimit on all three write paths |
| CORE-06 | 02-01, 02-03 | LWW-Element-Map laws; order/duplicates/retransmissions cannot affect outcome | ✓ SATISFIED | TestMerge_laws/lawsTombstones/duplicateDelivery |
| CORE-07 | 02-01, 02-03 | Every displaced version recorded | ✓ SATISFIED | TestMerge_losers/losersProperty |
| VERS-01 | 02-02 | UUIDv7 byte order equals (l, c, NodeID) order | ✓ SATISFIED | TestVersion_layout/byteOrder |
| VERS-02 | 02-02 | Unique per write by construction | ✓ SATISFIED | TestVersion_uniquePerWrite; cross-restart uniqueness now covered by the seeded watermark dominance (no collision possible by construction: first Tick > watermark) |
| VERS-03 | 02-02 | Monotonic under adversarial clock input | ✓ SATISFIED | TestClock_monotonic, counterOverflowBoundary, counterReset |
| VERS-04 | 02-02, 02-04 | HLC initializes from dump's full watermark; every post-restart version strictly dominates every dumped version | ✓ SATISFIED | Config.InitialL/InitialC seam; TestClock_restartProperty full-watermark case; TestStore_restartDominance A/B; REQUIREMENTS.md wording updated to the strict-dominance extension — the previous "with open gap" qualifier is removed |
| VERS-05 | 02-02 | Drift beyond drift_warn logged as warning | ✓ SATISFIED (engine side; logging deferred to Phase 6 OBS-01/03) | Signed drift signal returned by every clock-advancing op, both signs property-proven |
| TEST-01 | 02-03 | CRDT properties: commutative, associative, idempotent | ✓ SATISFIED | TestMerge_laws, TestMerge_lawsTombstones |
| TEST-02 | 02-02, 02-04 | HLC properties incl. counter-reset and restart-from-dump-max | ✓ SATISFIED | Six properties in hlc_prop_test.go; restart case now the full-watermark model — the previous caveat is closed |

Orphaned requirements: none — REQUIREMENTS.md maps exactly the 14 IDs to Phase 2, all claimed by plans (02-04 adds a second claim on VERS-02/VERS-04/TEST-02 for the gap closure).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| internal/app/version.go | 21 | Unused constant `varPayloadMask` (dead code) — review IN-01, open info | ℹ️ Info | None functional; cleanup candidate |
| internal/app/version_test.go | 173-197 | Tautological range assertions in TestVersion_unpackTotal — review IN-03, open info | ℹ️ Info | Panic-freedom proof intact; dead checks overstate coverage |
| internal/app/clock.go | 48-54, 80-84 | No guard at 48-bit l ceiling — review IN-02, open info | ℹ️ Info | Unreachable via wall clock before ~year 8921; carried as coincidental-reliance (see frontmatter) |

Zero debt markers (TBD/FIXME/XXX/TODO/HACK), zero `//nolint`, zero logging, zero empty-return stubs across the five 02-04-modified files (scanned this pass). No new findings in the 02-04 delta (matches the incremental review 0c7ba3c).

### Human Verification Required

None. All 40 truths are verified with behavioral evidence (named tests green under the race detector; the two previously falsified truths upgraded by TestClock_restartProperty and TestStore_restartDominance, with the mutation check proving the tests bind). The deferred VERS-05 logging half is Phase 6 scope by design (D-01), not a human-verification item.

### Gaps Summary

No gaps. The single gap from the initial verification — G-02-VERS04-DOMINANCE (restart seam seeded only `l`; first post-restart Tick emitted (dumpMax, 1), which a dump version with counter >= 1 at the watermark millisecond could beat or collide with) — is closed with three layers of evidence: (1) the seam now carries the full watermark (`NewClock` initialC masked to 12 bits; `Config.InitialC` propagated by `NewStore`; Tick/Update byte-identical, confirmed by diff); (2) the property suite proves strict dominance over a (dumpMax, counter 1..4095, own NodeID) watermark across an adversarial walk, and the store-level regressions encode both demonstrated probe failures (silent revert, silent drop) as dead; (3) a mutation check run this pass (seed stripped → all three restart tests fail; source restored) proves the regression locks are not vacuous. Review disposition records WR-01 fixed; the three remaining open findings are info-level cleanups with no functional impact. The phase goal — pure I/O-free decision core, UUIDv7-packed HLC versions, LWW-Element-Map store, correctness laws property-proven — is achieved in full.

---

_Verified: 2026-11-02T02:53:24Z_
_Verifier: Claude (gsd-verifier)_
