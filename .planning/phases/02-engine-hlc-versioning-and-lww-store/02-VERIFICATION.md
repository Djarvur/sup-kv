---
phase: 02-engine-hlc-versioning-and-lww-store
verified: 2026-11-02T02:04:55Z
status: gaps_found
score: 37/39 must-haves verified
covered_files:
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-01-PLAN.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-01-SUMMARY.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-02-PLAN.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-02-SUMMARY.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-03-PLAN.md
  - .planning/phases/02-engine-hlc-versioning-and-lww-store/02-03-SUMMARY.md
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
covered_digest: "v4:sha256:d05f7aef320bf8d93975f3c531094fdee103330785e9c3b0e709384ddaeacbbc"
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "A clock constructed with InitialL equal to the dump's max l emits only versions strictly greater than every version in that dump, whatever the wall clock does afterward (VERS-04) — falsifies two 02-02 truths: the strictly-greater claim and the restart-precision probe"
    status: failed
    reason: >-
      NewClock seeds l only (no counter seed exists in the API). The first post-restart Tick
      emits (dumpMax, 1). A dump version at the watermark l with counter >= 1 — reachable
      whenever 2+ writes (own or merged remote) landed in the dump's final millisecond —
      beats or collides with that emission. Verified by behavioral probe against the shipped
      package: (variant A) a local write landing before dump-load stamps (T,1,7) and a peer
      holding the dump's (T,5,99) REJECTS it — the acknowledged new write silently reverts
      to the pre-restart value; (variant B) a dump record of exactly (T,1,7) with different
      data merges as an equal-version no-op — fresh data silently dropped, permanent
      per-key divergence. Mitigation exists on the natural ordering (Merge-load the dump,
      then write: Merge's clock.Update advances the counter past the dump watermark, and
      the probe confirms the new write then WINS), but nothing in internal/app enforces
      that ordering, the plan's own test models the watermark only as (dumpMax, 0,
      maxNodeID), and Phase 3 will wire InitialL directly. Matches open review finding
      WR-01 (02-REVIEW.md, warning, untriaged in 02-REVIEW-DISPOSITION.md). Requirement
      VERS-04 as worded in REQUIREMENTS.md ("initializes at startup from the dump's max l,
      not from the wall clock") IS satisfied — the failed claim is the plan truth's
      strictly-greater-every-version extension.
    artifacts:
      - path: internal/app/clock.go
        issue: "NewClock(now, initialL, driftWarn) has no initialC parameter; the doc contract (lines 24-29) states 'pass the dump's maximum l', which is insufficient information to preserve uniqueness and post-restart dominance"
      - path: internal/app/store.go
        issue: "Config.InitialL int64 cannot express a counter seed; nothing prevents a local write between NewStore and dump-load"
      - path: internal/app/hlc_prop_test.go
        issue: "TestClock_restartProperty models the dump watermark only as (dumpMax, 0, maxNodeID) — the hazardous (dumpMax, c>0) cases are untested"
    missing:
      - "Extend the restart seam per the WR-01 fix: add an initialC uint16 parameter to NewClock (first Tick returns at least (initialL, initialC+1) with the existing overflow rule), or keep the signature and document + enforce dumpMax+1 seeding"
      - "Propagate the seam through Config and NewStore"
      - "Extend TestClock_restartProperty with a dump whose watermark is (dumpMax, c>0, ownNodeID) and assert no collision/subordination"
deferred:
  - truth: "VERS-05's 'drift beyond drift_warn is logged as a warning' — the logging half (the engine-side signed drift signal is complete and property-proven)"
    addressed_in: "Phase 6"
    evidence: "REQUIREMENTS.md OBS-01: 'A metrics endpoint exposes ... clock-drift warnings'; OBS-03: structured logging. D-01 forbids logging inside internal/app by design, so the logging consumer necessarily lands in a later phase."
coincidental_reliance_items:
  - truth: "Versions never wrap or repeat (TEST-02 4096/4097 overflow boundary)"
    reason: undeclared-precondition
    harden: "The no-wrap clause holds because l stays below the 48-bit ceiling (2^48 ms ~ year 8921), which nothing declares as a precondition on the injected now func. At l = 2^48-1 with counter 4095 the bump produces l = 2^48 and PackVersion's tsMask wraps the version to zero (review IN-02). Declare 'l < 2^48-1 is a caller precondition on now' in NewClock's doc, or clamp at the ceiling."
---

# Phase 2: Engine — HLC Versioning and LWW Store Verification Report

**Phase Goal:** The decision core exists as a pure, I/O-free library — HLC versions packed into UUIDv7 and the LWW-Element-Map store — with its correctness laws proven by property tests that run forever after
**Verified:** 2026-11-02T02:04:55Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

The goal is substantially achieved: `internal/app` is a pure, I/O-free library (depguard-enforced, verified: zero `net`/`os`/`syscall`/`google.golang.org` imports, zero logging), the HLC packs into RFC 9562 UUIDv7 (octet-exact vector proven), the LWW-Element-Map store implements the full CORE semantics, and the correctness laws are property-proven by a seeded, replayable suite that is green under the race detector at 100% statement coverage. One plan-level truth — the strictly-greater-than-every-dump-version restart guarantee (VERS-04 extension) — is falsified by a demonstrated counterexample (open review finding WR-01), leaving 2 of 39 truths failed from that single root cause.

### Observable Truths

Plan 02-01 (17 truths — all VERIFIED; full suite green under `go test -race -count=1`, case tables confirmed in source):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Local write flows end-to-end: Set stamps fresh HLC version in UUIDv7 carrying NodeID; Get returns exact bytes + version | ✓ VERIFIED | `TestStore_endToEnd` — asserts Unpack = (testNowMs, 0, testNodeID), byte round-trip, UUIDv7 marker bits |
| 2 | Get returns ErrNotFound for missing and tombstoned keys (CORE-01) | ✓ VERIFIED | `TestStore_Get` subtests "missing key returns ErrNotFound", "tombstoned key returns ErrNotFound" |
| 3 | Set replaces prior value under LWW, greater version wins (CORE-02) | ✓ VERIFIED | `TestStore_Set` "a second set replaces the value under a strictly greater version" |
| 4 | SetIfAbsent returns ErrAlreadyExists for any existing record incl. tombstone (CORE-03) | ✓ VERIFIED | `TestStore_SetIfAbsent` "existing live key…", "existing tombstoned key returns ErrAlreadyExists without recreation" |
| 5 | Delete writes fresh-version tombstone that stays in map and merges like any write (CORE-04) | ✓ VERIFIED | `TestStore_Delete` "live key gets a strictly greater fresh tombstone"; `Records()` keeps tombstones; `TestMerge_lawsTombstones` no-resurrection case |
| 6 | Values over ValueMaxBytes rejected with ErrValueTooLarge before any copy (CORE-05) | ✓ VERIFIED | `TestStore_valueLimit` — 61440 accepted / 61441 rejected on set, setifabsent AND merge; "set over the size limit is rejected and stores nothing" asserts key stays absent |
| 7 | Merge keeps remote version, displaces only on strictly greater full 16-byte version, returns every displaced/rejected version as Conflict (CORE-06/07) | ✓ VERIFIED | `store.go` join() lines 206-222 (full-byte Compare, no re-stamp); `TestMerge_losers` + `TestMerge_losersProperty` (500 seeded pairs) |
| 8 | Two writes in same ms get distinct strictly increasing versions (counter disambiguates) | ✓ VERIFIED | `TestStore_Set` "two sets inside one millisecond get distinct strictly increasing versions" |
| 9 | Zero-length value is legal | ✓ VERIFIED | `TestStore_valueLimit` zero-length rows on all three write paths |
| 10 | Repeat writes never error, fresh version each time | ✓ VERIFIED | `TestStore_Set` "writing the same value twice stamps a fresh version each time" |
| 11 | SetIfAbsent twice: first creates, second ErrAlreadyExists, first write's value kept | ✓ VERIFIED | `TestStore_SetIfAbsent` "existing live key returns ErrAlreadyExists and keeps the stored value" (Get asserts "one") |
| 12 | Delete on missing/tombstoned key: ErrNotFound, no new tombstone, no version burned | ✓ VERIFIED | `TestStore_Delete` "missing key returns ErrNotFound and burns no version", "already tombstoned key returns ErrNotFound and writes nothing" |
| 13 | Equal-version merge is a full no-op, no Conflict | ✓ VERIFIED | `TestMerge_losers` "equal version is a no-op with no conflicts"; `TestMerge_duplicateDelivery` (200 iters) |
| 14 | Merge into missing key installs, no loser reported | ✓ VERIFIED | `TestMerge_losers` "merging into a missing key installs with no conflicts" |
| 15 | No Conflict ever carries loser == winner | ✓ VERIFIED | `TestMerge_losersProperty` explicit loop assertion (lines 253-258) |
| 16 | Displacing-nothing merge returns empty conflict list, not error | ✓ VERIFIED | join() returns nil on install/equal paths; asserted across the merge suite |
| 17 | Sentinel errors matched with errors.Is (tracer + tables) | ✓ VERIFIED | errors.go 3 sentinels via errors.New; all tests use errors.Is |

Plan 02-02 (11 truths — 9 VERIFIED, 2 FAILED from one root cause):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Tick/Update follow Kulkarni rules, (l, counter) strictly increases on every call (VERS-03) | ✓ VERIFIED | `TestClock_tick`/`TestClock_update` (all four named branches), `TestClock_monotonic` 2000-step adversarial walk + 8-goroutine concurrent subtest, zero race findings |
| 2 | Byte-wise compare equals (l, counter, NodeID) order vs field-wise reference model (VERS-01) | ✓ VERIFIED | `TestVersion_byteOrder` — 1000 seeded triples against in-test reference model; `TestVersion_layout` octet-exact vector `0000000000017123aaf3400000000000` |
| 3 | 10,000 Ticks under stuck clock → 10,000 distinct strictly increasing versions (VERS-02) | ✓ VERIFIED | `TestVersion_uniquePerWrite` — pairwise-distinct map + strict Compare increase through overflow bumps |
| 4 | Drift signal fires both directions with sign; always returned, never recorded (VERS-05, D-01) | ✓ VERIFIED | `TestClock_driftSignal` (both signs + disabled rows), `TestClock_driftUnderChaos` (both signs observed, exact magnitudes); zero logging in internal/app |
| 5 | Clock with InitialL = dump's max l emits only versions strictly greater than EVERY version in that dump (VERS-04) | ✗ FAILED | Behavioral probe (see Gaps Summary): first Tick emits (dumpMax, 1); a dump version at the watermark l with counter ≥ 1 beats or collides with it. WR-01 open |
| 6 | Concurrent Tick/Update: parallel mix, zero race findings (VERS-03 concurrency) | ✓ VERIFIED | `TestClock_monotonic` concurrent subtest (8 × 500 ops, per-caller monotonicity); full suite `-race` green |
| 7 | Same remote version twice never decreases (l, c) — second Update no-op (VERS-03 idempotency) | ✓ VERIFIED | Kulkarni branch `newL==l==remoteL → c = max(c,cm)+1` (clock.go:71-72); `TestClock_update` "equal timestamps" branch rows |
| 8 | InitialL ahead of wall clock → first versions at/above InitialL; InitialL zero starts from wall clock (VERS-04 boundary) | ✓ VERIFIED | `TestClock_restartFromDumpMax` both directions |
| 9 | Restart exact to the ms: every subsequent version strictly greater than every dump version, even under backward-jumped clock (VERS-04 precision) | ✗ FAILED | `TestClock_restartProperty` passes but models the watermark only as (dumpMax, 0, maxNodeID); same root cause as truth 5 — the (dumpMax, c≥1) watermark is unbeatable from a cold start (02-02 SUMMARY documents this narrowing) |
| 10 | Stuck clock: Ticks 1-4096 fill counters 0-4095, Tick 4097 → (l+1, 0); never wrap/repeat (TEST-02 boundary) | ✓ VERIFIED (coincidental-reliance) | `TestClock_counterOverflowBoundary` asserts the 4097th tick = (l+1, 0). The "never wrap" clause relies on the undeclared precondition l < 2^48-1 (see coincidental_reliance_items) |
| 11 | HLC state is plain integer arithmetic — no time.Time held as state, no monotonic-read comparisons (Pitfall 8, `verification: backstop`) | ✓ VERIFIED | Exhaustive static enumeration + exact-model property tests: clock.go state is `l int64`, `c uint16` (lines 18-19; `now func()` is the D-02 seam, not state); zero time-comparison methods (`.After/.Before/.Sub/.Equal`) and zero `time.Now()` in production code; `TestClock_counterReset` and `TestClock_monotonic` pin exact integer-ms readings via `time.UnixMilli` fakes — behavior would diverge if any time.Time state or monotonic comparison existed |

Plan 02-03 (11 truths — all VERIFIED):

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Merge commutative, associative, idempotent incl. tombstones over seeded random stores (TEST-01, CORE-06) | ✓ VERIFIED | `TestMerge_laws` — 500 seeded triples, state + key-sorted conflict sets; `assertMergeLaws` shared helper |
| 2 | Every displaced/rejected version returned exactly once as Conflict paired with displacer (CORE-07) | ✓ VERIFIED | `TestMerge_losersProperty` — 500 seeded pairs, both directions, Winner/Loser pairing exact |
| 3 | Delivery order, duplicates, retransmissions cannot affect outcome (CORE-06) | ✓ VERIFIED | Commutativity (identical state AND conflicts both directions) + `TestMerge_duplicateDelivery` |
| 4 | Parallel Set/SetIfAbsent/Delete/Merge/Get: zero race findings | ✓ VERIFIED | `TestStore_concurrentHammer` — 8 goroutines × 200 ops, atomic fake now, per-goroutine PCG; full suite `-race -count=1` green |
| 5 | Concurrent SetIfAbsent on one key: exactly one succeeds, others ErrAlreadyExists | ✓ VERIFIED | `TestStore_concurrentSetIfAbsent` — 16 racers, exactly-one-winner + payload assertion |
| 6 | Delete racing Set: survivor holds exactly the greater captured version | ✓ VERIFIED | `TestStore_concurrentDeleteVsSet` — 50 duels, outcome-independent greater-version assertion |
| 7 | merge(a,b) and merge(b,a): identical state AND identical conflict sets | ✓ VERIFIED | `assertMergeLaws` commutativity block with `sortedConflicts` order-insensitive comparison |
| 8 | Every displaced version exactly once in conflicts, paired with displacer | ✓ VERIFIED | `TestMerge_losersProperty` switch on Compare derives exact expected conflict set |
| 9 | merge(a,a) idempotent: zero conflicts, nothing changes | ✓ VERIFIED | `assertMergeLaws` idempotence block (500 × 2 mixes) |
| 10 | Laws hold for empty stores | ✓ VERIFIED | `randStore`/`randTombStore` can yield empty maps (1 + rng.IntN(16) min 1 merge... min 1 record; empty-store case covered by cloneStore on empty src and the missing-key install cases in `TestMerge_losers`) |
| 11 | merge(merge(a,b),c) equals merge(a,merge(b,c)) incl. tombstoned keys | ✓ VERIFIED | `assertMergeLaws` associativity block over both record mixes |

**Score:** 37/39 truths verified (0 present, behavior-unverified; 2 failed from one root cause)

### Deferred Items

| # | Item | Addressed In | Evidence |
| --- | --- | --- | --- |
| 1 | VERS-05's "logged as a warning" half — the engine returns the signed drift signal (complete, property-proven); actual logging/metrics wiring | Phase 6 | REQUIREMENTS.md OBS-01 ("clock-drift warnings" in the metrics endpoint), OBS-03 (structured logging); D-01 forbids logging inside internal/app by design |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| internal/app/errors.go | 3 sentinels via errors.New | ✓ VERIFIED | ErrNotFound/ErrAlreadyExists/ErrValueTooLarge, lowercase, RPC mapping documented (19 lines) |
| internal/app/version.go | RecordVersion pack/unpack/compare | ✓ VERIFIED | 59 lines; PackVersion/Unpack total/Compare wrapping bytes.Compare; RFC 9562 constants |
| internal/app/clock.go | HLC with now seam, overflow rule, initialL seam, drift | ✓ VERIFIED | 103 lines; Kulkarni Tick/Update, counter-overflow l bump, signed driftBetween |
| internal/app/store.go | Config/Record/Conflict/RecordChecksum/Store full API | ✓ VERIFIED | 252 lines; all six Store methods + defensive copies + checksum |
| internal/app/store_test.go | Tracer + CORE tables + concurrency hammer | ✓ VERIFIED | 761 lines, 10 test functions, case tables confirmed |
| internal/app/doc.go | Package contract | ✓ VERIFIED | 34 lines; LWW laws, visibility asymmetry, D-01 return contracts, ownership, concurrency |
| internal/app/clock_test.go | 5 clock unit tables | ✓ VERIFIED | 378 lines; all five named tests incl. both drift signs |
| internal/app/props_test.go | Seeded generators | ✓ VERIFIED | 98 lines; newRNG (printed "property seed:" + -seed replay), nextWallTime, randVersion/RawVersion/Record/Key/Store |
| internal/app/version_test.go | Layout/roundtrip/byteOrder/unpackTotal | ✓ VERIFIED | 197 lines; octet-exact vector, 1000-iteration properties |
| internal/app/hlc_prop_test.go | 6 TEST-02 properties | ✓ VERIFIED | 303 lines; incl. concurrent subtest and 10,000-tick stress |
| internal/app/merge_test.go | 4 merge-law property tests | ✓ VERIFIED | 260 lines; laws/tombstones/duplicates/losers with order-insensitive conflict comparison |

Note: `gsd-tools verify.artifacts` reported "Missing export" for `Clock.Tick`, `Clock.Update`, `Store.Get/Set/SetIfAbsent/Delete/Merge/Records`, `RecordVersion.Unpack/Compare` — a tool limitation (it matches only package-level functions, not methods). All ten methods confirmed present by direct inspection: version.go:47,57; clock.go:38,63; store.go:110,128,159,175,191,239.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| store.go | clock.go | Tick/Update calls | ✓ WIRED | write() → clock.Tick (store.go:142); Merge → clock.Update (store.go:199) |
| store.go | version.go | PackVersion/Compare | ✓ WIRED | write() → PackVersion (store.go:143); join() → Compare (store.go:212) |
| store_test.go | store.go | exported API only, fake now | ✓ WIRED | external app_test package, fixedNow/atomicNow seams |
| hlc_prop_test.go | props_test.go | shared generators | ✓ WIRED | newRNG/nextWallTime/randRecord/randStore consumed |
| version_test.go | version.go | PackVersion/Unpack/Compare | ✓ WIRED | layout vectors + ordering property |
| clock_test.go | clock.go | NewClock fake-now seam | ✓ WIRED | all Kulkarni branches driven via closures |
| merge_test.go | props_test.go | randStore/randRecord/newRNG | ✓ WIRED | all three consumed |
| merge_test.go | store.go | exported Merge/Records only | ✓ WIRED | cloneStore/mergeAll use exclusively Merge+Records |
| store_test.go | store.go | concurrent hammer via exported API | ✓ WIRED | sync.WaitGroup.Go goroutines over Set/SetIfAbsent/Delete/Merge/Get |

All 9 links verified by `gsd-tools verify.key-links` (exit 0 per plan).

### Data-Flow Trace (Level 4)

Not applicable in the render sense — this phase is a library with no UI/external data source. The equivalent check (versions and records derive from real clock/state, not static values) holds: every version flows from clock.Tick/Update through PackVersion; every stored Record derives from caller data + RecordChecksum; Merge decides via the full 16-byte Compare on actual record versions. No hardcoded data paths found.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full package suite, race detector, fresh run | `go test -race -count=1 -cover ./...` | ok, 3.229s, coverage 100.0% of statements, exit 0 | ✓ PASS |
| Build + vet | `go build ./... && go vet ./...` | exit 0 | ✓ PASS |
| Strict lint repo-wide | `go tool golangci-lint config verify && go tool golangci-lint run ./...` | exit 0, zero issues | ✓ PASS |
| Seed replay hook | `go test ./internal/app/ -run '^TestVersion_byteOrder$' -args -seed=12345` | ok, exit 0 | ✓ PASS |
| WR-01 probe: restart, Merge-load dump, then write | temporary in-package test against shipped NewStore(Config{InitialL: T}) | write stamped (T,7,7), peer accepts NEW data — Merge-load advances the clock past the watermark | ✓ PASS (natural ordering defended) |
| WR-01 probe variant A: write BEFORE dump-load | same method, write first, then dump record (T,5,99) to peer | write stamped (T,1,7); peer REJECTS it, keeps "old remote value" — acknowledged write silently reverts | ✗ FAIL (gap evidence) |
| WR-01 probe variant B: equal-version collision | write-before-load (T,1,7) vs dump (T,1,7) different data | peer merge is a silent no-op; fresh data dropped — permanent divergence | ✗ FAIL (gap evidence) |
| go.mod/go.sum unchanged vs origin/master | `git diff --stat origin/master -- go.mod go.sum` | empty diff, exit 0 | ✓ PASS |
| internal/port untouched (D-03) | `ls internal/port` | doc.go only | ✓ PASS |

Probe files were created inside internal/app only for the duration of each run and removed afterward (working tree verified clean of them).

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist or were declared by this phase; the phase's probe obligations ("probe:" truths) are discharged by the named Go tests above. Not applicable.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| CORE-01 | 02-01 | Get: missing/tombstoned → NotFound | ✓ SATISFIED | TestStore_Get; errors.go sentinel |
| CORE-02 | 02-01 | Set create/replace under LWW | ✓ SATISFIED | TestStore_Set (4 probe cases), TestStore_endToEnd |
| CORE-03 | 02-01 | SetIfAbsent; existing (incl. tombstone) → AlreadyExists | ✓ SATISFIED | TestStore_SetIfAbsent, TestStore_concurrentSetIfAbsent |
| CORE-04 | 02-01 | Delete fresh-version tombstone, replicates like any write | ✓ SATISFIED | TestStore_Delete, TestMerge_lawsTombstones no-resurrection |
| CORE-05 | 02-01 | Values over value_max_bytes → InvalidArgument | ✓ SATISFIED | TestStore_valueLimit 61440/61441 on all three write paths (Phase 5 maps ErrValueTooLarge → InvalidArgument) |
| CORE-06 | 02-01, 02-03 | LWW-Element-Map laws; order/duplicates/retransmissions cannot affect outcome | ✓ SATISFIED | TestMerge_laws/lawsTombstones/duplicateDelivery (500 seeded iterations each) |
| CORE-07 | 02-01, 02-03 | Every displaced version recorded (merge exposes loser) | ✓ SATISFIED | TestMerge_losers/losersProperty; Conflict returned by value (D-01) |
| VERS-01 | 02-02 | UUIDv7 byte order equals (l, c, NodeID) order | ✓ SATISFIED | TestVersion_layout/byteOrder (reference model, 1000 triples) |
| VERS-02 | 02-02 | Unique per write by construction | ✓ SATISFIED | TestVersion_uniquePerWrite (10,000 ticks). Cross-restart uniqueness has the WR-01 corner (see gap) — within a run, proven |
| VERS-03 | 02-02 | Monotonic under adversarial clock input | ✓ SATISFIED | TestClock_monotonic (2000-step + concurrent), counterOverflowBoundary, counterReset |
| VERS-04 | 02-02 | HLC initializes from dump's max l, not wall clock | ✓ SATISFIED (with open gap on the stricter dominance guarantee) | NewClock/NewStore InitialL seam; TestClock_restartFromDumpMax/restartProperty. The initialization source requirement is met; the plan-truth's strictly-greater-than-every-dump-version extension is the failed gap (WR-01) |
| VERS-05 | 02-02 | Drift beyond drift_warn logged as warning | ✓ SATISFIED (engine side; logging deferred to Phase 6 OBS-01/03) | Signed drift signal returned by every clock-advancing op; both signs property-proven (TestClock_driftUnderChaos) |
| TEST-01 | 02-03 | CRDT properties: commutative, associative, idempotent | ✓ SATISFIED | TestMerge_laws, TestMerge_lawsTombstones — 500 seeded triples per mix |
| TEST-02 | 02-02 | HLC properties: adversarial monotonicity, post-receive ordering, uniqueness, counter-reset, restart | ✓ SATISFIED | Six named properties in hlc_prop_test.go (restart case covers the l-level guarantee; the full-version dominance corner is the gap) |

Orphaned requirements: none — REQUIREMENTS.md maps exactly the 14 IDs above to Phase 2, all claimed by plans.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| internal/app/version.go | 21 | Unused constant `varPayloadMask` (dead code; linter does not flag consts) — review IN-01 | ℹ️ Info | None functional; cleanup candidate |
| internal/app/version_test.go | 173-197 | Tautological range assertions in TestVersion_unpackTotal (structurally unfailable) — review IN-03 | ℹ️ Info | Test still genuinely proves panic-freedom over raw bit patterns (its T-02-03 purpose); the dead checks overstate coverage |
| internal/app/clock.go | 48-54, 80-84 | No guard at 48-bit l ceiling — version stream wraps at l = 2^48 (review IN-02) | ℹ️ Info | Unreachable via wall clock before ~year 8921; recorded as coincidental-reliance on the undeclared now-precondition |

Zero debt markers (TBD/FIXME/XXX/TODO/HACK), zero `//nolint`, zero logging, zero empty-return stubs across all phase files. The temporary nolint:unused directives documented in 02-02-SUMMARY were verifiably removed.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| store_test.go | CORE-01..07 | 10 | 0 | 0 | Value/Behavioral (exact versions, bytes, errors.Is) | OK |
| clock_test.go | VERS-03/04/05 | 5 | 0 | 0 | Value (exact (l,c,drift) triples, hand-computed table) | OK |
| version_test.go | VERS-01 | 4 | 0 | 0 | Value (octet-exact vector, reference model) | OK (IN-03 note above) |
| hlc_prop_test.go | VERS-02..05, TEST-02 | 6 | 0 | 0 | Value/Property (exact-model expectedDrift, derived expectations) | OK |
| merge_test.go | TEST-01, CORE-06/07 | 4 | 0 | 0 | Behavioral (deep state equality + conflict-set equality) | OK |

No disabled tests, no circular expected-value generation (reference models are independent field-wise implementations written inside the tests), assertion strength matches the requirement level. Seeds print and replay (`-seed=12345` verified).

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts (4/4, gate exit 0).

### Human Verification Required

Not applicable — status is gaps_found (rule 1). The WR-01 triage decision (fix the restart seam vs accept via override) is exactly the gap-closure decision routed to `/gsd:plan-phase --gaps`; 02-REVIEW-DISPOSITION.md already tracks it as open.

### Gaps Summary

One root cause — the restart seam seeds only `l` (open review finding WR-01) — falsifies two of the 39 plan truths: the clock constructed with `InitialL = dumpMax` emits `(dumpMax, 1)` first, which does not strictly dominate dump versions carrying counter ≥ 1 at the watermark millisecond. Behavioral probes against the shipped package demonstrated both consequences: an acknowledged post-restart write silently reverting on peer merge (variant A) and an equal-version collision silently dropping fresh data (variant B). The natural restart ordering — Merge-load the dump before serving writes — is defended (probe confirmed the clock's Update advances past the watermark), but nothing in `internal/app` enforces that ordering, `Config`/`NewClock` cannot express a counter seed, and Phase 3 will wire `InitialL` directly onto this seam. The requirement VERS-04 as worded (initialization source) is satisfied; the plan truth's stricter dominance guarantee is not. Fix options are concrete in 02-REVIEW.md WR-01 (add `initialC`, or document/enforce `dumpMax+1`), plus a watermark-`(dumpMax, c>0, ownNodeID)` extension of `TestClock_restartProperty`.

Everything else the goal demands is demonstrably true: the pure I/O-free library, the RFC 9562 packing, the LWW store with asymmetric tombstone visibility and exact loser accounting, the property suites (merge laws over 500 seeded triples × 2 mixes, 10,000-tick uniqueness, adversarial monotonicity, both drift signs), the concurrency proofs, 100% statement coverage under the race detector, zero dependencies added, and the strict lint gate green with zero suppressions.

---

_Verified: 2026-11-02T02:04:55Z_
_Verifier: Claude (gsd-verifier)_
