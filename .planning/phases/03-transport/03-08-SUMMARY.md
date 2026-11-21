---
phase: 03-transport
plan: 08
subsystem: repl-wire-validation
tags: [wire-validation, discard-classes, max-clock-lead, hlc-poisoning, rate-limited-warn, checksum, key-length, xport-03, xport-04, xport-10]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-03's version.RecordVersion.Validate (layout + now+maxLead) and the distinguishable ErrVersionLayout/ErrVersionFuture sentinels"
  - phase: 03-transport
    provides: "03-02's storage.MaxKeyBytes and storage.RecordChecksum, the single key ceiling and the record checksum domain"
  - phase: 03-transport
    provides: "03-07's paced listing/record/egress queues, which the validation step sits in front of without touching"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the negative table whose four XPORT-10 rows pinned the pre-fix behavior and flip here"
provides:
  - "a wire-validation step in HandleDatagram between classification and handleMessage: no invalid datagram reaches a handler, the store (ApplyRemote is the only Clock.Update path) or a timer"
  - "five discard classes and counters — empty_key, key_too_long, tombstone_with_data, bad_checksum, bad_version — with the pinned intra-message order and source-only details"
  - "repl.Config.MaxClockLead (default 60 s, the documented drift tolerance) and the same bound applied to DataSync entry versions"
  - "one WARN per class per 10 s of event-loop time, driven by the handler's now and naming only the class"
  - "the adapter mirror of the five new counters through Discards()"
  - "--max-clock-lead / SUPKV_MAX_CLOCK_LEAD at the documented 60 s default, passed into repl.Config.MaxClockLead"

actuals:
  tokens: 19400
  tasks: 3
  commits: 3
plan_head_before: 9c3b99affbed580248108c808b9530146bb2b629
plan_head_after: f4ac44b43a18cfcc291c0be80e353acf38d83aa1

tech-stack:
  added: []
  patterns:
    - "Validation as its own pipeline step returning (class, detail, valid), called after framing/identity classification and before dispatch: the position is the correctness contract, not the handler's first line."
    - "Per-message validation helpers (validateReplication, validateSync, keyVerdict) so the pinned intra-message order lives in one readable place per shape and cognitive-complexity stays under the lint gate."
    - "Rate limiting as per-class last-warn timestamp keyed on the event-loop's now: no wall clock, no global throttle, no dependency, deterministic under the simulator."
    - "Wire-shaped model generators: RandNearVersion straddles now+60 s so a property model exercises accepted and rejected readings, and RandRecordForKey stamps the production checksum so a record's only fault is the one a test injects."
    - "A counting storage.Clock fake turns 'validated before the clock' from an ordering claim into an observable zero-Tick/zero-Update assertion."

key-files:
  created: []
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/discard_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/merge_stats_test.go
    - internal/app/repl/sim_scenario_test.go
    - internal/app/internal/testutil/gen.go
    - internal/udprepl/repl.go
    - internal/udprepl/decode_negative_test.go
    - internal/udprepl/repl_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go

key-decisions:
  - "The validation detail is exactly the source handle — not the class token, not a node id, never a key: the class is already the OnDiscard first argument, and the detail carries no record metadata that could name user data (T-3-27)."
  - "The pinned order is implemented per shape: DataReplication — empty key, key over storage.MaxKeyBytes, tombstone-with-data, checksum, version; DataRequest/Ack — key only; DataSync — each entry's key then version in received order, first failure discarding the whole datagram; Hello/nil/standalone SyncEntry — nothing."
  - "An Ack carries a version but no record, so its version is not validated: a crafted future Ack is a harmless tracking no-op, and inventing a class for it would reject protocol traffic the pool never asked about. A table control row pins that asymmetry."
  - "Config.MaxClockLead normalizes <= 0 to 60 s inside the core, so main's flag default only makes the documented value visible in --help; the core remains total over any Config."
  - "The drift-warning scenario legitimately needs a lead wider than the 60 s default (its stamp is 61 s ahead while drift_warn is 60 s), so that one test constructs its core with an explicit 2-minute lead instead of weakening the production default."
  - "The two far-ahead simulation scenarios keep their intent with explicit leads (clockJumps 4 h, farAhead 25 h); the default never widens for tests."
  - "The model's version half calls version.RecordVersion.Validate directly while the key/order halves are hand-mirrored: the version domain is 03-03's tested contract, and re-deriving UUID bit masks in the discard model would duplicate a definition, not check one."
  - "The negative table's 70 000-character row stays decode-only (the kernel refuses the datagram) and a feedable 1 025-byte row now asserts key_too_long through a real socket; no decoder-side size bound exists or is promised."
  - "Metrics keeps its five-class label vocabulary: 03-12 owns the five new class samples plus the pacing families, so this plan stops at the adapter's Discards() mirror."
  - "The five class token strings are the shared vocabulary 03-12 will render; they are defined once in core.go and consumed by value, never re-spelled."

patterns-established:
  - "Every claim that a check happens before a side effect gets a counting fake: a zero count is the proof, and the fake sits behind the production seam (storage.Clock)."
  - "Wire fixtures are built wire-valid by default (production checksum recomputation in the shared datagram helper) so an accidental fixture fault surfaces as a test failure instead of silently changing what the test measures."
  - "A property model that must track a rejection pipeline mirrors the pipeline's order and classes independently while delegating the domain predicate (version layout) to its owning package."
  - "Tests that need a wider normalization than production construct their own Config explicitly and say why in a comment; the shared helper keeps the default."

requirements-completed: [XPORT-04, XPORT-10, XPORT-03]

coverage:
  - id: D1
    description: "The five validation classes exist with their own counters and tokens; the pinned intra-message order holds, including the mixed DataSync case where one invalid entry discards the whole listing"
    requirement: XPORT-10
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestDiscard_'"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_validationClasses (20 rows: class, order, mixed listing, two accepted controls)"
        status: pass
    human_judgment: false
  - id: D2
    description: "No invalid datagram reaches the store or the clock: a counting clock records zero Tick and zero Update for every invalid class and DataSync shape, the store stays empty and the sync flag false"
    requirement: XPORT-04
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_beforeClock"
        status: pass
      - kind: other
        ref: "local mutation check: disabling the validate call made TestDiscard_beforeClock/validationClasses/warnRateLimited/totality fail, then the mutation was reverted"
        status: pass
    human_judgment: false
  - id: D3
    description: "Key domain, tombstone shape and record checksum are rejected before the store: empty keys, keys over storage.MaxKeyBytes and tombstones carrying Data are counted; a checksum not matching storage.RecordChecksum is bad_checksum"
    requirement: XPORT-10
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_validationClasses rows empty_key / key_too_long / tombstone_with_data / bad_checksum"
        status: pass
      - kind: integration
        ref: "internal/udprepl/decode_negative_test.go#TestDecode_negativeTable rows empty_key_discarded_by_core, tombstone_with_data_discarded_by_core, key_too_long_over_max_key_bytes, bad_checksum_one_flipped_data_byte fed over a real loopback socket"
        status: pass
    human_judgment: false
  - id: D4
    description: "MaxClockLead is configurable end to end: the core normalizes to 60 s, the DataSync entry versions use the same bound, and --max-clock-lead / SUPKV_MAX_CLOCK_LEAD reach repl.Config with the documented default"
    requirement: XPORT-04
    verification:
      - kind: unit
        ref: "cmd/sup-kv/main_test.go#TestMain_maxClockLead (help listing + env twin, env-derived default, malformed env exitUsage, explicit flag parsed over a valid env twin)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMain_maxClockLead$|^TestMain_precedence$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "The validation WARN is rate-limited per class on the event-loop's now (one per 10 s, boundary inclusive) while the per-datagram OnDiscard seam and counters keep counting; neither the detail nor the WARN carries a key"
    requirement: XPORT-10
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_warnRateLimited"
        status: pass
    human_judgment: false
  - id: D6
    description: "The discard model conserves all ten counters over 500 adversarial datagrams, with near versions, wire checksums and adversarial key/record mutations; the negative table asserts the landed classes and keeps the scenarios' HLC-absorption intent"
    requirement: XPORT-10
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestDiscard_'"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestScenario_farAhead$|^TestScenario_clockJumps$'"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestDecode_negativeTable$'"
        status: pass
    human_judgment: false

duration: 21min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 8: Wire Validation Before the Clock Summary

**Every inbound datagram is now judged before any handler runs: five new discard classes count empty keys, over-limit keys, tombstones carrying data, forged checksums and poisoned versions, and a far-future version can no longer reach the store's Clock.Update, proven with a counting clock.**

## Performance

- **Duration:** 21 min (1269 s)
- **Started:** 2026-11-21T20:38:19Z
- **Completed:** 2026-11-21T20:59:28Z
- **Tasks:** 3
- **Files modified:** 11 (none created)

## Accomplishments

- `Core.validate` sits in `HandleDatagram` after the framing/identity classification and before `handleMessage`, returning `(class, detail, valid)`. Its position is the contract: `handleMessage` is the only path to the store, `applyRemote` is the only `ApplyRemote` call, and `storage.Merge` is the only `clock.Update` call — a rejected datagram can reach none of them.
- The pinned order is implemented per shape: `DataReplication` — empty key, key over `storage.MaxKeyBytes`, `Deleted` with non-empty `Data`, checksum mismatch against `storage.RecordChecksum(key, version, data, deleted)`, then `version.Validate(now, cfg.MaxClockLead)`; `DataRequest`/`Ack` — empty key then key length (an Ack's version is deliberately not validated); `DataSync` — each entry's key then version in received order, with the first failure discarding the whole datagram; `Hello`, a nil `Msg` and a standalone `SyncEntry` — nothing.
- The five class tokens and counters are added (`DiscardEmptyKey`, `DiscardKeyTooLong`, `DiscardTombstoneWithData`, `DiscardBadChecksum`, `DiscardBadVersion`; `EmptyKey`, `KeyTooLong`, `TombstoneWithData`, `BadChecksum`, `BadVersion`), each fired through `OnDiscard` with the source handle as the detail and logged as one WARN per class per 10 s of event time.
- `repl.Config.MaxClockLead` normalizes `<= 0` to `defaultMaxClockLead` (60 s, the drift tolerance). The far-ahead scenarios set explicit leads (clockJumps 4 h, farAhead 25 h) and the drift-warning unit test 2 min, so the production default never widens for a test.
- The adapter mirrors the five counters into atomics in `mirrorStats` and returns them from `Discards()`; no metrics change is made here because 03-12 owns the class samples and the pacing families.
- `--max-clock-lead` / `SUPKV_MAX_CLOCK_LEAD` defaults to `time.Minute` and is passed as `repl.Config.MaxClockLead`; a malformed env value aborts through the collected env-error path with exit 2.
- `testutil.RandNearVersion` (readings in `[now−1h, now+90s]`, straddling the 60 s bound) and `testutil.RandRecordForKey` (RandRecord's shape over a near version with the production checksum) let the totality model exercise both verdicts; `RandVersion`/`RandRecord` stay untouched for the storage property tests.
- Proofs: `TestDiscard_validationClasses` (20 rows including order rows, DataSync-entry rows, a mixed listing, and two accepted controls), `TestDiscard_beforeClock` (one datagram per invalid shape over a counting clock: zero Tick/Update, empty store, false sync flag), `TestDiscard_warnRateLimited` (per-class boundary, per-datagram seam still counting, no key material).
- The negative table flipped its empty-key and tombstone-with-data rows to their landed classes, split the oversize case into a decode-only 70 000-character row plus a feedable 1 025-byte `key_too_long` row, and gained `bad_checksum_one_flipped_data_byte`, `bad_version_far_future` (DataReplication) and `bad_version_far_future_sync_entry`, all asserted through `assertFedRow` over a real socket.

## Task Commits

Each task was committed atomically, with one documented boundary move:

1. **Task 1: Validation pipeline, five classes, MaxClockLead and the rate-limited WARN** — `5d71a1c` (feat)
2. **Task 2: Counting-clock proof, totality model, scenario leads and negative-table flips** — `dbf5229` (test)
3. **Task 3: max_clock_lead flag/env twin** — `f4ac44b` (feat)

**Plan metadata:** see the `docs(03-08)` commit that follows this summary.

_Note: no TDD tasks in this plan; the boundary move is documented under Deviations._

## Files Created/Modified

- `internal/app/repl/core.go` — the five class tokens and `DiscardStats` fields; `Config.MaxClockLead` + `defaultMaxClockLead`/`discardWarnInterval`; `Core.discardWarnAt`; `validate`/`validateReplication`/`validateSync`/`keyVerdict`/`keyClass`/`versionAccepted`/`discardValidated`; the `HandleDatagram` insertion and its updated doc comment.
- `internal/app/repl/discard_test.go` — the model's `validateClass`/`keyViolation`/`futureVersion`, the extended `bump`, adversarial `randBody`/`randWireKey`/`corruptRecord`, and the three new tests plus their helpers (`wireRecord`, `validWireVersion`, `futureWireVersion`, `badLayoutWireVersion`, `onlyClass`, `sumDiscards`, `validationRow`, `assertAcceptedDatagram`, `assertDiscardedDatagram`, `countingClock`).
- `internal/app/internal/testutil/gen.go` — `RandNearVersion`, `RandRecordForKey` and their window constants.
- `internal/app/repl/retransmit_test.go` — `newTestCoreWithConfig`; `replFrom` recomputes the production checksum so every hand-built datagram is wire-valid.
- `internal/app/repl/merge_stats_test.go` — `driftLead` (2 min) so the drift-warning scenario is not rejected by the version bound.
- `internal/app/repl/sim_scenario_test.go` — explicit `MaxClockLead` for clockJumps (4 h) and farAhead (25 h) with the intent documented.
- `internal/udprepl/repl.go` — five atomics, their `mirrorStats` stores and the `Discards()` return.
- `internal/udprepl/decode_negative_test.go` — flipped rows, the decode-only/feedable oversize split, three new rows, `negFarFutureVersion`, `negFlippedDataByte`, the extended class helpers and the fidelity-renamed dump subtest.
- `internal/udprepl/repl_test.go` — the supersession fixture carries its checksum (it previously passed for the wrong reason once validation landed).
- `cmd/sup-kv/main.go` — `envMaxClockLead`, `defaultMaxClockLead`, `config.maxClockLead`, the flag registration and the `repl.Config` pass-through.
- `cmd/sup-kv/main_test.go` — `TestMain_maxClockLead` and the `runToCompletion` helper.

## Decisions Made

- The detail is exactly the source handle: the class already rides `OnDiscard`'s first argument, and the T-6-05 contract forbids key names in either the detail or the WARN.
- `Ack` validates its key but not its version: it carries no record, a future-version Ack is a tracking no-op, and an accepted-control row pins the asymmetry.
- `Config.MaxClockLead <= 0` normalizes inside the core, so the launch flag is convenience and documentation, never the authority on the default.
- The drift-warning test needs a lead above 60 s because its intent (a stamp beyond `drift_warn`) collides with the new default bound; it sets 2 min explicitly rather than changing either default.
- The scenarios keep their HLC-absorption intent with explicit leads; the production 60 s default stays untouched.
- The model delegates the version verdict to `version.Validate` while hand-mirroring the key/order halves: the UUID domain is 03-03's tested contract, and the pipeline order is what this plan's model owns.
- The negative table's oversize row stays decode-only forever (no decoder-side bound exists); the feedable 1 025-byte row is the core-class evidence.
- Metrics is untouched: 03-12 explicitly owns the five new class samples, so this plan stops at the adapter mirror to avoid two plans landing the same label vocabulary.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Commit boundary] The Task 1 commit carried the test-model and negative-table adaptations forward from Task 2**
- **Found during:** Task 1 verification
- **Issue:** Task 1's verify (`-run '^TestDiscard_'`) includes `TestDiscard_totality`, whose model must know the new pipeline, and the udprepl negative table pins the pre-fix `Decoded`/processed behavior — so neither can stay red at a Task 1 commit without leaving a broken tree.
- **Fix:** Task 1's commit includes the model update (`gen.go`, `discard_test.go` model half), the wire-fixture checksums, the explicit scenario/drift leads and the negative-table flips/rows; Task 2's commit then adds the three new proof tests (and the dump-subtest rename). Each commit builds and its filtered verifies pass.
- **Files modified:** `internal/app/internal/testutil/gen.go`, `internal/app/repl/discard_test.go`, `internal/app/repl/retransmit_test.go`, `internal/app/repl/merge_stats_test.go`, `internal/app/repl/sim_scenario_test.go`, `internal/udprepl/decode_negative_test.go`, `internal/udprepl/repl_test.go`
- **Verification:** `go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./cmd/sup-kv/` green at HEAD; each commit's filtered verifies re-run green.
- **Committed in:** `5d71a1c`, `dbf5229`

**2. [Rule 1 - Bug] A udprepl test passed for the wrong reason after validation landed**
- **Found during:** the first udprepl run
- **Issue:** `TestWaitReplicated_superseded` sends a `DataReplication` whose fixture record had a zero checksum. With validation in place the datagram was rejected as `bad_checksum`, the tracked write was never superseded, and the wait timed out — which happened to match the test's `(false, 0)` expectation. The test no longer proved supersession.
- **Fix:** the fixture now carries `storage.RecordChecksum(...)`, so the supersession path is the one exercised.
- **Files modified:** `internal/udprepl/repl_test.go`
- **Verification:** the test passes with the supersession path live (a log-free apply), and the earlier pre-fix run's rejection is no longer possible.
- **Committed in:** `5d71a1c`

**3. [Rule 1 - Bug] Every hand-built inbound record fixture needed its production checksum**
- **Found during:** full repl suite after the pipeline landed
- **Issue:** `replFrom`-based tests (`TestMergeStats_counts`, `TestRetransmit_ackDrainSemantics`, `TestOutboxResults_resolutionEvents`, `TestRetransmit_heapInvariant`, `TestRestart_staleRunIdAndInFlight`) built records without a checksum, so the new checksum step discarded them and the tests failed on their real assertions.
- **Fix:** the shared `replFrom` helper recomputes the checksum (the only place inbound DataReplication fixtures are built), leaving each test's intent intact; a test that wants `bad_checksum` mutates its own record.
- **Files modified:** `internal/app/repl/retransmit_test.go`
- **Verification:** the six failing tests and the full repl suite green.
- **Committed in:** `5d71a1c`

**4. [Rule 3 - Blocking] The drift-warning test's stamp is outside the new version bound**
- **Found during:** full repl suite
- **Issue:** `TestMergeStats_counts` stamps a record `drift_warn + 1 s` (= 61 s) ahead to trigger the drift counter, while the new default rejects anything beyond 60 s. The two 60 s defaults collide.
- **Fix:** the test constructs its core with `MaxClockLead: 2 * time.Minute` (new `newTestCoreWithConfig` helper) and documents why; neither default changes.
- **Files modified:** `internal/app/repl/merge_stats_test.go`, `internal/app/repl/retransmit_test.go`
- **Verification:** `TestMergeStats_counts` green; the default path stays 60 s in every other test.
- **Committed in:** `5d71a1c`

**5. [Rule 3 - Blocking] The two far-ahead scenarios needed explicit larger leads**
- **Found during:** full repl suite
- **Issue:** `TestScenario_clockJumps` jumps up to ~3 h forward and `TestScenario_farAhead` runs a node 24 h ahead; with the 60 s default every such record was rejected and the clusters never quiesced.
- **Fix:** explicit `repl.Config{MaxClockLead: 4 * time.Hour}` and `{25 * time.Hour}` with comments stating the scenario's intent is HLC absorption, not rejection — exactly the plan's Task 2 item.
- **Files modified:** `internal/app/repl/sim_scenario_test.go`
- **Verification:** both scenarios green across fresh seeds.
- **Committed in:** `5d71a1c`

**6. [Rule 2 - Test truth] Stale forward-reference comments and names**
- **Found during:** Task 2 acceptance ("no forward-reference comment remains for a landed fix")
- **Issue:** the flipped rows' names still said `decoded_today_xport_10_forward_reference`, and the dump-codec subtest still described the inbound rejection as future work.
- **Fix:** renamed the rows to their landed behavior (`*_discarded_by_core`, `oversize_key_decode_only_over_udp_ceiling`) and renamed the dump subtest to `dump_codec_tombstone_with_data_fidelity` with a comment separating codec fidelity from inbound rejection.
- **Files modified:** `internal/udprepl/decode_negative_test.go`
- **Verification:** the table's comments and names now match the assertions; no forward reference to a landed fix remains.
- **Committed in:** `5d71a1c`, `dbf5229`

**7. [Rule 3 - Verification hygiene] The new launch test initially picked free ports**
- **Found during:** a cmd/sup-kv run that hit the harness's pre-existing port race twice
- **Issue:** the two error-path subtests called `freeUDPPort`, adding parallel port-race windows to a harness whose pick-then-bind pattern is already racy; the failures surfaced in an unrelated logging test.
- **Fix:** the configuration-error subtests use placeholder addresses (no bind happens before the error) and pick no ports at all.
- **Files modified:** `cmd/sup-kv/main_test.go`
- **Verification:** `TestMain_maxClockLead` green repeatedly; the remaining flake is the harness's own pre-existing race (see Issues).
- **Committed in:** `f4ac44b`

---

**Total deviations:** 7 auto-fixed (1 commit-boundary move, 2 test-truth/bug fixes, 3 blocking test adaptations, 1 verification hygiene)
**Impact on plan:** No scope change. Every must-have and acceptance criterion is met; the five tokens are defined once for 03-12 to reuse, and metrics is deliberately left to 03-12.

## Issues Encountered

- **Pre-existing harness flake:** `cmd/sup-kv`'s `freeUDPPort` closes its throwaway socket before the test binds the port, so parallel tests can steal it (`TestLog_jsonDefault`: "probe rx bind: address already in use"). It reproduced twice and passed on immediate retry; the new test no longer contributes port picks. Not fixed here (out of scope; the pattern predates this plan).
- **Drift bound collision:** the 60 s `drift_warn` default and the 60 s `max_clock_lead` default meet exactly, so the one test that must observe drift beyond the threshold needs an explicit wider lead. This is inherent to the design decision (A1) and is now pinned in the test comment.
- **Ack asymmetry:** the plan's pinned order lists key checks for `Ack` but no version check; the implementation follows it and a control row documents that a future-version Ack is processed. If a later plan wants Acks bounded, that is a visible new row, not a silent change.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 03-09 can read `storage.MaxKeyBytes` at the API and startup seams without touching the inbound check: the core already enforces the same constant (its "inbound check landed in 03-08" premise holds).
- 03-12 has the five class tokens, the five `DiscardStats` fields and the adapter mirror it needs; it still owns the metrics label samples (`empty_key`, `key_too_long`, `tombstone_with_data`, `bad_checksum`, `bad_version`) and the pacing families.
- The simulator's models are adapted, so 03-10/03-11 scenario work inherits a green baseline; any new scenario with a deliberately skewed clock must set its own `MaxClockLead` and say why.
- `--max-clock-lead` is available to the stand for an end-to-end rejection scenario if a later plan wants one; the core proof already exists.

## Self-Check: PASSED

- All named files exist on disk; no file was created (created list empty), 11 modified.
- Measured commits from the persisted plan ledger: `git rev-list --count 9c3b99a..f4ac44b` = 3; `plan_head_before` = `9c3b99affbed580248108c808b9530146bb2b629`, `plan_head_after` = `f4ac44b43a18cfcc291c0be80e353acf38d83aa1`.
- The `evaluation-scope` query verb is unavailable in this runtime (`gsd_run` is not installed), so the close-out uses the persisted plan ledger and `git rev-list --count` as the documented equivalent measurement (the 03-06/03-07 precedent).
- Plan verification re-run at close-out: `go build ./...` green; `go tool golangci-lint run ./...` green; `go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./cmd/sup-kv/` green; the focused `TestScenario_farAhead|TestScenario_clockJumps|TestDiscard_` and `TestDecode_negativeTable` runs green; adjacent suites (`simnet`, `metrics`, `grpcapi`, `storage`, `version`) green.
- Acceptance re-check: five classes with counters and source-only details; every invalid datagram rejected before `handleMessage` (zero Tick/Update, empty store, false sync flag); WARN per class per 10 s on event time with the per-datagram seam still counting; `MaxClockLead` normalizes to 60 s and reaches the core from flag/env; negative table asserts the landed classes and no landed-fix forward reference remains.
- Mutation evidence: temporarily disabling the `validate` call failed `TestDiscard_beforeClock`, `TestDiscard_validationClasses`, `TestDiscard_warnRateLimited` and `TestDiscard_totality`; the mutation was reverted and the green re-run recorded.
- REQUIREMENTS.md boxes for XPORT-03/XPORT-04/XPORT-10 are left for the phase gate (the 03-07/03-12 precedent; 03-09 still owns XPORT-03's API and startup halves); the SUMMARY records the plan's three requirement IDs verbatim.

---
*Phase: 03-transport*
*Completed: 2026-11-21*
