---
phase: 05-persistence
plan: 01
subsystem: persistence
tags: [value-limit, codec, dump, restart, jsonl, validation]
requires:
  - phase: 03-transport
    provides: persist.Engine write confirmation, the JSONL dump codec and the persistfile adapter
  - phase: 04-protocol
    provides: the dump/replication composition this plan leaves intact
provides:
  - storage.ValueMaxBytes as the only production definition of 61 440
  - storage.Config.ValueLimit and storage.Store.ValueMaxBytes
  - a configured limit on persist.Encode/persist.Decode with an encoder tripwire
  - engine wiring that decodes before the store exists and encodes/restores at the store's effective limit
  - binary rejection of a non-positive --value-max-bytes with the flag named
affects: [persistence, dump-codec, value-limit, load-stats, value-max-bytes]
actuals:
  tokens: 10669
  tasks: 3
  commits: 4
tech-stack:
  added: []
  patterns:
    - "Single exported constant plus one normalization helper, read before the object that normalizes it exists"
    - "Over-limit encode is a storage.ErrValueTooLarge tripwire naming the key, never a silent skip"
key-files:
  created: []
  modified:
    - internal/app/storage/store.go
    - internal/app/storage/store_test.go
    - internal/app/persist/codec.go
    - internal/app/persist/engine.go
    - internal/app/persist/codec_test.go
    - internal/app/persist/engine_test.go
    - internal/app/persist/bench_test.go
    - internal/app/persist/fuzz_dump_test.go
    - internal/persistfile/open_seed_test.go
    - internal/grpcapi/server_test.go
    - internal/udprepl/decode_negative_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - cmd/sup-kv/operations_test.go
key-decisions:
  - "The one definition is an exported storage.ValueMaxBytes; Config.ValueLimit and Store.ValueMaxBytes both derive from it, so no wiring path restates the number"
  - "persist.Open computes storeCfg.ValueLimit() before decoding: the decode seeds the dump watermark and therefore runs before NewStore exists"
  - "The encoder treats an over-limit record as the store-invariant violation — a wrapped storage.ErrValueTooLarge naming the key — because a confirmed write must never fail to encode"
  - "A non-positive --value-max-bytes is a usage error (exit 2) naming the flag, not the store's silent default"
patterns-established:
  - "Single-source config: Config.ValueLimit() is the shared pre-store/normalization expression; Store.ValueMaxBytes() is the post-store capture"
  - "Oversize-decode fixtures encode at limit+oversizeBy and decode at limit (RESEARCH Pitfall 11)"
requirements-completed: [PERSIST-01, PERSIST-10]
coverage:
  - id: D1
    description: "storage.ValueMaxBytes is the only production definition of 61 440, with Config.ValueLimit and Store.ValueMaxBytes deriving from it"
    requirement: "PERSIST-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestStorage_valueLimit$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "persist.Encode/Decode take the configured limit; the encoder errors on an over-limit record and the decoder discards one, deleting maxRecordBytes"
    requirement: "PERSIST-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ (TestCodec_discardClasses, round-trip property, fuzz corpus)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Engine decodes at the pre-store limit and captures the store's limit for dump and snapshot restore"
    requirement: "PERSIST-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_exactValueLimitRestart$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_configShrunkValueLimits$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "A 63 000-byte record accepted at ValueMaxBytes=64 000 survives a real engine restart (frozen P-01 engine acceptance)"
    requirement: "PERSIST-01"
    verification:
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/persistfile/ -run '^TestRed_P01_valueLimitEngine$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "A real binary confirms the record, takes SIGKILL and returns the full value after restart at --value-max-bytes=64000 (frozen process acceptance)"
    requirement: "PERSIST-01"
    verification:
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_P01_valueLimit$'"
        status: pass
    human_judgment: false
  - id: D6
    description: "--value-max-bytes=0 and -1 exit 2 naming the flag; 64 000/64 408 still start and 64 409 still exits 2 with the unchanged ceiling text"
    requirement: "PERSIST-01"
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestMain_valueMaxNonPositive$|^TestMain_valueMaxDatagram$'"
        status: pass
    human_judgment: false
  - id: D7
    description: "The two P-01 expectation rows stay red and the two frozen test files are byte-identical to the phase-start base"
    requirement: "PERSIST-10"
    verification:
      - kind: command
        ref: "grep -c -P '^TestRed_P01_valueLimit(Engine)?\\tred$' scripts/red-expectations.tsv; git diff --quiet 4f81fd7 -- internal/persistfile/red_value_limit_test.go internal/stand/stand_red_test.go"
        status: pass
    human_judgment: false
duration: 13min
completed: 2026-11-22
status: complete
plan_head_before: b1204b025d96ec1cf11d3de00b73f7b28dcf6b0c
plan_head_after: 84376e29cf82681b58e647a8dcdd224ece1f481a
---

# Phase 05 Plan 01: Single-source value limit from store through restart Summary

**The 61 440-byte limit now exists once in production and flows through the store, codec, engine and binary, so a write the store accepted at any accepted flag value survives the next restart.**

## Performance

- **Duration:** 13min (761s)
- **Started:** 2026-11-22T12:37:19Z
- **Completed:** 2026-11-22T12:50:00Z
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments

- `storage.ValueMaxBytes = 61440` is now the only production definition; `Config.ValueLimit()` and `Store.ValueMaxBytes()` derive from it, and `normalized()` uses the same helper. `defaultValueMaxBytes` is gone from the store and the binary.
- `persist.Encode`/`persist.Decode` take the configured limit as a parameter. `maxRecordBytes` is deleted; an over-limit encode is a `storage.ErrValueTooLarge` tripwire naming the key, and decode discards an over-limit record as the value-limit class.
- `persist.Open` decodes at `storeCfg.ValueLimit()` before the store exists (the decode seeds the clock), and the engine captures `Store.ValueMaxBytes()` for every dump and snapshot restore. `Open` still returns `(*Engine, int)`.
- Both frozen P-01 acceptances pass unedited: the engine-level 63 000-byte restart and the process-level SIGKILL/restart at `--value-max-bytes=64000` (63 000- and 64 000-byte subtests).
- `--value-max-bytes=0` and `-1` exit 2 through the usage path naming the flag; 64 000 and the exact ceiling value 64 408 still start, and 64 409 still exits 2 with the unchanged datagram-ceiling text.
- Pins added: `TestStorage_valueLimit`, `TestEngine_exactValueLimitRestart` (inclusive 8-byte boundary plus limit+1 rejection), and `TestMain_valueMaxNonPositive`.
- `make lint`, `make test` and the plan-level package suite are green; the two P-01 TSV rows remain `red`.

## Task Commits

1. **Task 1: end-to-end single-source limit** - `a13af5f` (feat)
2. **Task 2: shrunken-limit expectation and oversize-fixture limits** - `6f2a9b0` (test)
3. **Task 3 tests: the three pins** - `0c14800` (test)
4. **Task 3 implementation: non-positive validation** - `84376e2` (feat)

**Plan metadata:** `docs(05-01)` completion commit (this commit).

## Files Created/Modified

- `internal/app/storage/store.go` - exported `ValueMaxBytes`, `Config.ValueLimit`, `Store.ValueMaxBytes`, shared normalization; deleted `defaultValueMaxBytes`.
- `internal/app/storage/store_test.go` - `TestStorage_valueLimit` pins the constant and both helpers.
- `internal/app/persist/codec.go` - configured limit parameters on `Encode`/`Decode`, encode tripwire, deleted `maxRecordBytes`.
- `internal/app/persist/engine.go` - pre-store `ValueLimit()` decode, captured `valueMaxBytes`, dump/restore limits.
- `internal/app/persist/codec_test.go` - explicit limits at every fixture; oversize fixture encoded above its decode limit.
- `internal/app/persist/engine_test.go` - explicit decode limits, `discarded == 1` shrunk expectation, `TestEngine_exactValueLimitRestart`.
- `internal/app/persist/bench_test.go` - dump benchmark encodes at `storage.ValueMaxBytes`.
- `internal/app/persist/fuzz_dump_test.go` - fuzz decode/encode calls carry the explicit limit.
- `internal/persistfile/open_seed_test.go` - seed dump encodes at the default limit.
- `internal/grpcapi/server_test.go` - snapshot fixtures encode at the default limit.
- `internal/udprepl/decode_negative_test.go` - dump codec fixtures carry explicit limits.
- `cmd/sup-kv/main.go` - flag default and ceiling fallback from the store constant; `errValueMaxBytesRange` and `checkValueMaxBytes` in `finalizeConfig`.
- `cmd/sup-kv/main_test.go` - `TestMain_valueMaxNonPositive` plus its probe bound.
- `cmd/sup-kv/operations_test.go` - the 61 440 anchor names `storage.ValueMaxBytes`.

## Decisions Made

- The single source is an exported constant, not a config default duplicated in each layer: every consumer derives through `Config.ValueLimit()` (pre-store) or `Store.ValueMaxBytes()` (post-store), which is why the decode bound and the store's enforcement point cannot diverge.
- `Open` keeps its frozen `(*Engine, int)` shape; the richer load report stays a later plan's `LoadStats()`.
- The encoder is a tripwire, not a filter: a record the store accepted must encode, so an over-limit record is an error wrapping the store's sentinel, never a skipped line.
- Non-positive `--value-max-bytes` is rejected before the datagram-ceiling check; the ceiling check's fallback remains only as a total-function guard.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Lint gate] Dumper cleanup form in the new engine test**
- **Found during:** Task 3 (lint gate after the first test commit)
- **Issue:** `defer startDumper(t, eng)()` tripped `revive`'s defer-chain rule and `paralleltest`'s `t.Parallel` cleanup rule.
- **Fix:** `stop := startDumper(t, eng); t.Cleanup(stop)` — behavior identical, cleanup still runs after the test.
- **Files modified:** `internal/app/persist/engine_test.go`
- **Verification:** `make lint` green; `TestEngine_exactValueLimitRestart` re-run green.
- **Commit:** `84376e2`

**2. [Rule 2 - Doc accuracy] Restore-skip comment names the decode-time discard**
- **Found during:** Task 2
- **Issue:** `TestEngine_restoreSkipsReplay`'s comment described the oversize record as a replay skip, but the configured-limit decode now discards it before replay.
- **Fix:** Comment corrected; the test's assertions and outcome are unchanged.
- **Files modified:** `internal/app/persist/engine_test.go`
- **Verification:** `go test -race -count=1 ./internal/app/persist/` green.
- **Commit:** `6f2a9b0`

**3. [Rule 3 - Blocking] Stale `Current Plan` counter corrected before advancing**
- **Found during:** State close-out
- **Issue:** STATE.md carried `current_plan: 6` from the previous phase transition while this phase had zero summaries, so `state advance-plan` would have recorded plan 7.
- **Fix:** Set `Current Plan` to 1 (the phase's true start position), then ran `state advance-plan`, yielding 2 (next plan). ROADMAP progress was updated separately; `state update-progress` declined because the milestone phase directory is unscoped, leaving the pre-existing progress bar unchanged.
- **Files modified:** `.planning/STATE.md`
- **Verification:** frontmatter now reads `current_plan: 2`, `completed_plans: 48`; ROADMAP reads `1/12`.
- **Commit:** plan metadata commit (this commit)

Otherwise the plan executed as written; Task 2 legitimately required only test changes because Task 1's production change had already moved the behavior.

## TDD Gate Compliance

- **Task 1 RED:** the pre-existing frozen `TestRed_P01_valueLimitEngine` ran at HEAD and failed on its planned assertion (record verdict `RED_EVIDENCE_OK` / `target_test_failed`; expected the full 63 000-byte value, actual "key not found" with `Open discarded 1`). GREEN: `a13af5f`. The frozen test is not editable by contract, so no separate RED commit was possible.
- **Task 2 RED:** after Task 1, `TestEngine_configShrunkValueLimits` failed on the planned expectation (actual `discarded = 1`, old want 0; `RED_EVIDENCE_OK`). GREEN: `6f2a9b0` (test-only).
- **Task 3 RED:** `TestMain_valueMaxNonPositive` ran both subtests against the real binary; both started and served until the 3 s probe killed them, so the exit-2 assertion failed (`RED_EVIDENCE_OK`). GREEN: `84376e2`. `TestStorage_valueLimit` and `TestEngine_exactValueLimitRestart` are characterization pins over Task 1's production and passed on add — disclosed rather than fabricating a red.
- **REFACTOR:** none beyond the lint-mandated cleanup form folded into `84376e2`.

## Issues Encountered

None. The only iteration was the two lint findings in the new test (documented above).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 05-02 can build on the single-source limit: the codec now takes the value, and later plans can add the decode-class split and load reporting without revisiting the constant.
- `requirements.ready-ids` reports PERSIST-01 and PERSIST-10 blocked because sibling plans in this phase still lack summaries; they were deliberately not marked complete.
- No stubs, skipped tests or unrun verifies were introduced; the broken-windows ledger needs no entries from this plan.

## Self-Check: PASSED
