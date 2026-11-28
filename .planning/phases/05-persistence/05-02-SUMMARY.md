---
phase: 05-persistence
plan: 02
subsystem: persistence
tags: [dump-codec, jsonl, census, line-isolation, decode-stats, fuzz, legacy-format]
requires:
  - phase: 05-persistence
    provides: the configured value limit threaded through Encode/Decode and Open
  - phase: 03-transport
    provides: the persistfile dump adapter and the fallback-chain Load oracle
affects: [persistence, dump-codec, load-stats, p03, p10, truncation-detection]
actuals:
  tokens: 12662
  tasks: 3
  commits: 6
tech-stack:
  added: []
  patterns:
    - "Line-oriented decode: bounded bufio.Reader + ReadSlice + per-line json.Unmarshal, never a streaming json.Decoder"
    - "Format-bound line limit computed from named constants, independent of the configured value limit"
    - "Additive header census: record_count present enforces completeness, absent is the documented v1.0 legacy read"
    - "Per-class decode outcome (accepted plus four fault classes) feeding one DecodeStats census"
key-files:
  created: []
  modified:
    - internal/app/persist/codec.go
    - internal/app/persist/codec_test.go
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - internal/app/persist/fuzz_dump_test.go
    - internal/udprepl/decode_negative_test.go
key-decisions:
  - "Decode's second result is decayed DecodeStats (Checksum, Version, ValueLimit, Malformed) with Total(); Open keeps its frozen (*Engine, int) shape by returning stats.Total()"
  - "A record line is parsed with json.Unmarshal so a syntax fault loses only that line; the streaming decoder was entirely removed"
  - "The reader bound is maxDumpLineBytes() — datagram base64 expansion plus sixfold escaping of a MaxKeyBytes key plus fixed overhead — never valueMaxBytes, so a default-limit dump still reads at limit 16"
  - "The header keeps format_version 1 and adds optional record_count: fewer lines than the count is a generation fault, more is tolerated, absent is the legacy path with no census"
  - "An unterminated final fragment is a generation fault when it fails to parse and a normal record line when it parses"
patterns-established:
  - "Config-independent reader bound: the format's maximum legal line is a named-constant formula, so configuration shrinkage can never turn a legal record into ErrBufferFull"
  - "Census-aware fuzz oracle: fuzzDumpScan walks every post-header line independently and the invariant is len(records)+stats.Total() <= record lines"
requirements-completed: [PERSIST-03, PERSIST-10]
coverage:
  - id: D1
    description: "DecodeStats keeps checksum, version, value-limit and malformed classes separate with Total(); Open returns the total as its frozen second value"
    requirement: "PERSIST-10"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestCodec_(corruptLineIsolated|discardClasses)$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Line-oriented decode isolates one syntax-corrupt record line; generation faults return nil records and the zero DecodeStats"
    requirement: "PERSIST-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestCodec_corruptLineIsolated$'"
        status: pass
      - kind: command
        ref: "test \"$(grep -c 'json.NewDecoder' internal/app/persist/codec.go)\" -eq 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "Encode writes record_count; a v1.1 generation with fewer lines than the count faults, more lines load, and a v1.0 header without the field reads with no census"
    requirement: "PERSIST-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestCodec_headerCensus$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "The format-bound reader rejects overlong lines, holds a 1 024-byte quote/control key with a 61 440-byte value, and classifies a default-limit record as ValueLimit under a 16-byte limit"
    requirement: "PERSIST-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestCodec_(escapedMaxKeyRoundTrip|shrunkLimitKeepsOtherRecords)$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "The fuzz target's seeds and invariants are census-aware, and every remaining persist.Decode call site speaks the stats return"
    requirement: "PERSIST-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^FuzzDumpDecode$'"
        status: pass
      - kind: command
        ref: "go test -count=1 ./internal/app/persist/ -run '^$' -fuzz '^FuzzDumpDecode$' -fuzztime 3s"
        status: pass
      - kind: command
        ref: "go test -race -count=1 ./internal/app/persist/ ./internal/udprepl/ ./internal/persistfile/"
        status: pass
    human_judgment: false
duration: 14min
completed: 2026-11-22
status: complete
plan_head_before: 246dbb8f000b6d7190c683016a341561d5d47248
plan_head_after: 4b88551e16de98113135c4a8dc2f2e00e9380bea
---

# Phase 05 Plan 02: Line-isolated decode and completeness census Summary

**A dump now declares its record count and the reader walks bounded lines with json.Unmarshal, so a line-boundary truncation is a generation fault, a syntax-corrupt line loses only itself, and the four discard classes are counted separately.**

## Performance

- **Duration:** 14min (868s)
- **Started:** 2026-11-22T13:02:39Z
- **Completed:** 2026-11-22T13:17:07Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- `Decode` now returns `DecodeStats{Checksum, Version, ValueLimit, Malformed}` with `Total()`; `Open` keeps its frozen `(*Engine, int)` return by reporting `stats.Total()`, and `Restore` discards the stats.
- The `json.Decoder` record path is gone: a bounded `bufio.Reader` + `ReadSlice('\n')` + per-line `json.Unmarshal` isolates one syntax fault to one `Malformed` discard, while `grep -c 'json.NewDecoder' internal/app/persist/codec.go` is 0.
- `Encode` writes `{"format_version":1,"record_count":N}`; a v1.1 generation with fewer record lines than the count is `errIncompleteDump`, more lines are tolerated, and a v1.0 header without the field takes the documented legacy path with no census.
- `maxDumpLineBytes()` is the named-constant format bound (65 507-byte datagram base64 expansion + 6× a 1 024-byte key + 256 bytes overhead); overlong lines are generation faults, while a record over the configured limit inside a readable line is a `ValueLimit` discard.
- `TestCodec_escapedMaxKeyRoundTrip` proves the near-worst-case legal record the old `(limit+2)/3*4 + MaxKeyBytes + 256` bound could not hold: a 1 024-byte quote/control key plus a 61 440-byte value round-trips at the default limit.
- `TestCodec_shrunkLimitKeepsOtherRecords` proves a default-limit dump still reads under `Decode(dump, 16)`: both small records load and the one over-limit record is counted as `ValueLimit`, never `ErrBufferFull`.
- The fuzz corpus is census-aware: `fuzzDumpScan` walks every post-header line independently and the invariant `len(records)+stats.Total() <= record lines` holds, with new count-too-large, injected-newline and overlong-line seeds.
- The duplicate-key fixture now carries a matching header count; the last-wins assertion stays intact.
- `make lint`, `make test`, `go vet ./...`, the three affected package suites and the 3-second fuzz smoke are green; the two frozen P-01 test files are byte-identical to the phase-start base (4f81fd7).

## Task Commits

1. **Task 1 RED: corrupt-line isolation test** - `f4d5d58` (test)
2. **Task 1 GREEN: line-isolated decode and four-class stats** - `500423e` (feat)
3. **Task 2: census and format-bound pins** - `2062baa` (test)
4. **Task 3 RED: census-aware fuzz seeds and invariants** - `c22eeb2` (test)
5. **Task 3 GREEN: line-walking fuzz scan and call sites** - `4ececed` (test)
6. **Task 3 REFACTOR: DecodeStats receiver/doc-link shape** - `4b88551` (refactor)

**Plan metadata:** the `docs(05-02)` completion commit (this commit).

## Files Created/Modified

- `internal/app/persist/codec.go` - `DecodeStats`/`Total`, `decodeClass`, bounded line reader, format constants, header census, per-class `decodeRecord`.
- `internal/app/persist/engine.go` - `Open` stores the accepted generation's stats and returns `stats.Total()`; doc contract updated.
- `internal/app/persist/codec_test.go` - corrupt-line test, census/boundary table, escaped max-key and shrunk-limit tests, exact-class discard table, matching duplicate-key count.
- `internal/app/persist/engine_test.go` - decode call sites speak `DecodeStats`; no assertion weakened.
- `internal/app/persist/fuzz_dump_test.go` - census header/seed builder, three new seed shapes, line-walking scan, stats-aware echo.
- `internal/udprepl/decode_negative_test.go` - tombstone dump fixtures assert `stats.Total() == 0`.

## Decisions Made

- The stats return is `DecodeStats`, not a bare int: the four classes are the contract PERSIST-10 and the later load report need, and `Open` preserves its frozen two-value shape through `Total()`.
- The reader bound is derived only from the format (datagram ceiling, key ceiling, escaping) and never from `valueMaxBytes`, which is what makes a config-shrunk load classify rather than fault.
- `format_version` stays 1 with an optional `record_count`, honoring D-03's downgrade-safe additive field rather than a version bump or a trailer.
- The census tolerates extra lines and rejects only missing ones: hand edits and duplicate keys keep working, while a cut at a line boundary is the detectable fault.
- A malformed unterminated tail is a generation fault; an unterminated tail that parses is an ordinary record line, so a writer that omits the final newline is not punished.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 2's census and format-bound reader landed with Task 1**
- **Found during:** Task 1 GREEN
- **Issue:** The header `record_count` census, the bounded line reader and the writer's count share the same `Decode` loop and `header` struct as the line-isolation rewrite. Splitting them would have committed a decoder with a half-built reader or a header whose field nothing enforces.
- **Fix:** Implemented the census and the format-bound reader together in `500423e`; Task 2 then pinned every planned behavior with `TestCodec_headerCensus`, `TestCodec_escapedMaxKeyRoundTrip` and `TestCodec_shrunkLimitKeepsOtherRecords`, and upgraded the discard table to exact per-class assertions.
- **Files modified:** `internal/app/persist/codec.go`, `internal/app/persist/codec_test.go`
- **Verification:** all Task 2 verifies green; the TDD consequence is disclosed below.
- **Commit:** `500423e` (implementation), `2062baa` (Task 2 pins)

**2. [Rule 1 - Bug] Duplicate-key fixture rebuilt in Task 1, not Task 2**
- **Found during:** Task 1 GREEN
- **Issue:** The hand-joined duplicate-key dump inherited a header count larger than its joined record lines, so the new census rejected it as incomplete.
- **Fix:** The joined dump now carries `record_count: 2`, equal to its two record lines, keeping the last-wins assertion. (The first rebuild inserted a blank line because `censusHeader` already ends in a newline; the join now trims it, and the pre-commit test run caught the extra `Malformed`.)
- **Files modified:** `internal/app/persist/codec_test.go`
- **Verification:** `TestCodec_discardClasses/a_duplicate_key...` passes with `stats.Total() == 0`.
- **Commit:** `500423e`

**3. [Rule 1 - Lint] DecodeStats receiver consistency and doc links**
- **Found during:** Task 3 lint gate
- **Issue:** `recvcheck` rejected `Total`'s value receiver against `add`'s pointer receiver, and `godoclint` required bracketed stdlib links in the new doc comments.
- **Fix:** `add` is now a value receiver returning the counted stats; `[bufio.Reader]`, `[json.Decoder]`, `[json.Unmarshal]`, `[io.EOF]` and `[bufio.ErrBufferFull]` are linked in the comments. Behavior unchanged.
- **Files modified:** `internal/app/persist/codec.go`
- **Verification:** `make lint` green; the three package suites and the fuzz smoke re-run green.
- **Commit:** `4b88551`

### Intentional plan-order adjustment

`internal/app/persist/bench_test.go` was listed in Task 1's file set but contains no `Decode` call (it encodes only), so it needed no edit. The plan's "replace the test-local `maxRecordBytes` fixture" step was already satisfied by 05-01, which deleted that constant.

## TDD Gate Compliance

- **Task 1 RED:** `TestCodec_corruptLineIsolated` was added against the pre-change return and failed on its planned assertion (verdict `RED_EVIDENCE_OK` / `target_test_failed`: the streaming decoder returned `persist: dump record unreadable: invalid character '!'` instead of the two intact siblings). GREEN: `500423e`.
- **Task 2 RED:** none — its production behavior legitimately existed after Task 1 (deviation 1), so the planned tests passed on add. They are recorded as characterization pins rather than a fabricated RED: `TestCodec_headerCensus`, `TestCodec_escapedMaxKeyRoundTrip`, `TestCodec_shrunkLimitKeepsOtherRecords` and the exact-class discard assertion. Commit: `2062baa`.
- **Task 3 RED:** after adding the census-aware seeds and invariant but before fixing the oracle, `FuzzDumpDecode` failed on seed#10 with `discarded=2 outside 0..0 parsed record lines` (verdict `RED_EVIDENCE_OK` / `target_test_failed`). GREEN: `4ececed`; REFACTOR: `4b88551`.
- **RED evidence records:** `.planning/phases/05-persistence/red-05-02-task1.json` and `.planning/phases/05-persistence/red-05-02-task3.json` (added by this plan's metadata commit).

## Issues Encountered

- The plan-commit ledger file predated this execution and carried a stale base (`ce4a28e`) from an interrupted earlier attempt, which would have counted 313 commits for this plan. It was corrected to the actual pre-plan HEAD `246dbb8`, and the measured commit count is 6.
- No other issues; the lint findings are recorded as deviation 3.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 05-03 can build on the four-class `DecodeStats` and the census: the load report has a stable per-class census to expose, and `Open`'s frozen shape is intact.
- The legacy v1.0 no-census limitation is documented in `Decode`'s contract and pinned by an explicit test row, so later format work has the compatibility baseline.
- No stubs, skipped tests or unrun verifies were introduced; the broken-windows ledger needs no entries from this plan.

## Self-Check: PASSED

All six modified files and all six task commits were found; the SUMMARY frontmatter's measured `plan_head_before`/`plan_head_after` and commit count were verified against the corrected ledger.
