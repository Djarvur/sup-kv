---
phase: 02-stand-and-simulator-as-tests
plan: 17
subsystem: testing
tags: [fuzzing, native-fuzz, seed-corpus, decoder-invariants, dump-codec, test-08, d-08, asvs-v5]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: "plan 02-07's corrupted-datagram negative table (bad CRC, truncated envelope, truncated body, wrong proto_version, bad run_id) — the corruption classes the seed corpus mirrors by definition"
provides:
  - "internal/udprepl/fuzz_decode_test.go — FuzzDecode over decode() with invariant-only assertions (no panic, Decoded never nil Msg, BadCRC/Undecodable carry no Msg, verdict always in the fixed vocabulary)"
  - "internal/udprepl/testdata/fuzz/FuzzDecode/ — 12 committed seed entries covering every valid message type plus each negative-table corruption class"
  - "internal/app/persist/fuzz_dump_test.go — FuzzDumpDecode over persist.Decode with the codec's documented contract (generation faults error with empty results, per-record faults only increment discarded, checksums verify, echo round-trip, tombstones never gain invented payload)"
  - "internal/app/persist/testdata/fuzz/FuzzDumpDecode/ — 8 committed seed entries covering the valid JSONL shape and every failure class the codec distinguishes"
affects: [02-11 (scheduled.yml deep-fuzz jobs invoke these targets with -fuzztime), Phase 3 (XPORT-03/XPORT-10 decoder fixes must keep these invariants true)]

# Actuals (#2632) — same estimateTokens scale as the plan estimate: chars/4 over the realized diff.
actuals:
  tokens: 7800
  tasks: 1
  commits: 1

plan_head_before: 1b8034c4ee08a9d649fd0bf91175c21ee867ac65
plan_head_after: efe7d3f599b8dec5c218441a7ec5368a3f37ea20

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One fuzz target per package (RESEARCH Pitfall 4): FuzzDecode in udprepl, FuzzDumpDecode in app/persist — separate packages keep `-fuzz` regexps single-match"
    - "Seed builders are plain functions returning (payloads, error) rather than *testing.F methods, so one source of truth drives both f.Add and the committed testdata corpus files"
    - "Construction guards on every corruption fixture (e.g. truncation still fails to unmarshal, base64 flip changes decoded bytes) so a class representative cannot silently decay into a valid input"
    - "Fuzz bodies assert the codec contract only — generation faults return nil/0/err, per-record faults only increment discarded, accepted records verify checksums and re-encode identically — never a stronger 'garbage is rejected' claim"
    - "Committed corpus entries named seed_NN_<class> in the go test fuzz v1 two-line format; plain go test loads them as ordinary fuzz seeds (PR-CI smoke), deep runs stay on the nightly schedule with -fuzztime"

key-files:
  created:
    - internal/udprepl/fuzz_decode_test.go
    - internal/udprepl/testdata/fuzz/FuzzDecode/ (12 seed entries)
    - internal/app/persist/fuzz_dump_test.go
    - internal/app/persist/testdata/fuzz/FuzzDumpDecode/ (8 seed entries)
  modified: []

key-decisions:
  - "FuzzDecode lives in package udprepl_test and drives decode() through the test-binary-only DecodeDatagramForTest seam — the same production decode path, without a new testpackage exemption"
  - "Seed builders return errors instead of taking *testing.F: the temporary corpus generator and the fuzz target consume the identical payload list, so committed corpora cannot drift from f.Add"
  - "The truncated-body seed cuts one byte off a four-byte checksum varint (0xFFFFFF) so the CRC gate passes and only the body unmarshal fails — the same construction the negative table pins, re-derived here with zero test-file coupling"
  - "The flipped-byte dump seed flips the base64 token's first character; flipping the padded quantum's last character feeds only a dropped output byte and would leave the payload (and checksum) intact — caught by the corpus classification, now guarded by a base64 decode comparison"
  - "FuzzDumpDecode's tombstone-with-data check scans the input lines and allows payload whenever any input line for that key carried it — the union direction is sound across production's last-wins duplicates and per-record discards, so the assertion can only fail on invented data"
  - "The echo invariant runs persist.Encode(Decode(x)) and re-decodes: every accepted record must round-trip with checksum and flags intact and nothing discarded, the strongest property the codec guarantees today"
  - "Corpora were written from the f.Add payloads by a temporary generator (deleted before commit) rather than copying fuzz-cache mutations: provenance is one deterministic construction per class; both targets were additionally fuzz-smoked for 15s each with zero crashers"

patterns-established:
  - "Per-class committed fuzz corpus with filename-encoded class and an origin table in the summary — the T-2-14 mitigation (no external data, no secrets, every file's derivation reviewable)"
  - "Error-returning seed builders: fuzz target and corpus generator share one payload list; corpus regeneration is a five-line temporary test, never hand-edited byte literals"

requirements-completed: [TEST-08]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "FuzzDecode fuzz target over decode() asserting invariants only: no panic, Decoded never carries a nil Msg, BadCRC/Undecodable carry no Msg, verdict always in Decoded/BadCRC/Undecodable"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "internal/udprepl/fuzz_decode_test.go#FuzzDecode"
        status: pass
    human_judgment: false
  - id: D2
    description: "12-entry udprepl seed corpus (every valid message type, empty datagram, bad CRC, truncated envelope, truncated body, wrong proto_version with correct CRC, bad run_id) loading and passing under plain go test"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^FuzzDecode$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "FuzzDumpDecode target asserting the dump codec contract: generation faults return nil/0/err, per-record faults only increment discarded, accepted records' checksums verify and account for parsed lines, accepted records echo through re-encode/re-decode, tombstones carry Data only when the input did"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "internal/app/persist/fuzz_dump_test.go#FuzzDumpDecode"
        status: pass
    human_judgment: false
  - id: D4
    description: "8-entry persist seed corpus (valid dump with tombstone, header-only, empty, unsupported format version, bad-checksum line, torn final record, dump cut mid-line, flipped payload byte) loading and passing under plain go test"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^FuzzDumpDecode$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "Bounded fuzz smoke on both targets (-fuzztime=15s each): no crashers, no invariant violations, no crasher files written; one fuzz target per package verified by grep"
    requirement: "TEST-08"
    verification:
      - kind: unit
        ref: "go test -fuzz='^FuzzDecode$' -fuzztime=15s ./internal/udprepl/ ; go test -fuzz='^FuzzDumpDecode$' -fuzztime=15s ./internal/app/persist/"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 17: Fuzz Targets with Committed Corpora Summary

**Native fuzzing now pins both decoders: FuzzDecode feeds arbitrary bytes through the UDP verdict machinery and FuzzDumpDecode through the dump codec, each backed by a committed in-repo seed corpus that plain `go test` loads — every corruption class from the negative table is an executable seed, and a 15-second fuzz smoke per target found no crashers.**

## Performance

- **Duration:** 8 min (493 s)
- **Started:** 2026-11-21T12:50:45Z
- **Completed:** 2026-11-21T12:58:58Z
- **Tasks:** 1
- **Files created:** 22 (2 test files + 20 corpus entries; no production file touched, no dependency added)

## Accomplishments

- **FuzzDecode (TEST-08, D-08, ASVS V5):** seeds every valid wire message through the production encoder — Hello, DataReplication with data, DataReplication tombstone, Ack, DataRequest, DataSync with entries — plus the empty datagram and one representative of each negative-table corruption class (bad CRC, truncated envelope, truncated body, wrong proto_version with a correct CRC, bad run_id). The body asserts invariants only: no panic, a `Decoded` verdict never carries a nil `Msg`, `BadCRC`/`Undecodable` verdicts carry no `Msg`, and the verdict always lies inside the fixed vocabulary. It deliberately does not claim arbitrary garbage is rejected — the negative table's XPORT-10 rows prove two such classes decode today.
- **FuzzDumpDecode (TEST-08):** seeds valid JSONL dumps built by the production encoder (data record plus tombstone), a header-only dump, an empty input, an unsupported format version, a bad-checksum line, a torn final record, a dump cut mid-line and a flipped payload byte. The body asserts the codec contract: generation-level faults return `nil, 0, err`; success accounts for parsed lines (`len(records)+discarded <= lines`); every accepted record's `storage.RecordChecksum` verifies; accepted records re-encode and re-decode identically with nothing discarded; and a returned tombstone carries Data only when some input line for that key carried it (echo-only, never invented).
- **Committed corpora:** 12 udprepl and 8 persist entries, each in the RESEARCH-verified two-line `go test fuzz v1` + `[]byte("...")` format, named `seed_NN_<class>`. Plain `go test` loads them as ordinary fuzz seeds (verified: each appears as a `FuzzDecode/seed_...` subtest), which is the PR-CI smoke; deep coverage-guided runs stay on the nightly schedule (plan 02-11) and always carry `-fuzztime`.
- **Bounded fuzz smoke:** `-fuzztime=15s` per target — udprepl 700 421 executions with 105 new interesting inputs, persist 1 791 755 executions with 362 new interesting inputs — both PASS, no crasher files written to `testdata`, so there is no FINDING against production decode paths and nothing was committed unreviewed.
- **One fuzz target per package (Pitfall 4):** `grep '^func Fuzz'` returns exactly `FuzzDecode` in `internal/udprepl` and `FuzzDumpDecode` in `internal/app/persist`, so a `-fuzz` regexp always matches exactly one target in one package.

## Task Commits

Each task was committed atomically:

1. **Task 1: FuzzDecode + FuzzDumpDecode with committed corpora** - `efe7d3f` (test)

**Plan metadata:** this docs commit (SUMMARY/STATE/ROADMAP/REQUIREMENTS)

## Files Created/Modified

- `internal/udprepl/fuzz_decode_test.go` — `FuzzDecode`, the plain-function seed builder, truncated-body/Hello-body/hand-built-envelope helpers, and the truncated-body construction guard
- `internal/udprepl/testdata/fuzz/FuzzDecode/` — 12 seed entries (see origin table below)
- `internal/app/persist/fuzz_dump_test.go` — `FuzzDumpDecode`, the seed builder, JSONL line renderer, base64 flip helper with payload-change guard, echo assertion helper, and the input-line scanner
- `internal/app/persist/testdata/fuzz/FuzzDumpDecode/` — 8 seed entries (see origin table below)

## Committed Corpus Origins (T-2-14)

Every entry derives from an in-repo encoded message or its defined corruption — no external data, no secrets, no credential-like material. Corpus generation method: the `f.Add` payload list, rendered by a temporary generator (deleted before the commit) in the `go test fuzz v1` format; `-fuzztime=15s` was the bounded smoke run per target and produced no committed crashers.

| File | Origin |
|------|--------|
| `udprepl/.../seed_00_valid_hello` | production encoding of `port.Hello{}` |
| `udprepl/.../seed_01_valid_data_replication` | production encoding of `DataReplication{Key:"fuzz/decode", Data:"value"}` |
| `udprepl/.../seed_02_valid_tombstone` | production encoding of `DataReplication{Deleted:true}` (empty Data, tombstone checksum) |
| `udprepl/.../seed_03_valid_ack` | production encoding of `Ack{Key:"fuzz/decode"}` |
| `udprepl/.../seed_04_valid_data_request` | production encoding of `DataRequest{Key:"fuzz/decode"}` |
| `udprepl/.../seed_05_valid_data_sync` | production encoding of `DataSync` with two entries |
| `udprepl/.../seed_06_corruption_undecodable` | empty datagram (the Undecodable floor) |
| `udprepl/.../seed_07_corruption_bad_crc` | seed_01 with its final body byte XOR 0xFF (envelope parses, CRC fails) |
| `udprepl/.../seed_08_corruption_undecodable` | first half of seed_01 (truncated envelope) |
| `udprepl/.../seed_09_corruption_undecodable` | hand-built envelope, correct CRC, body cut one byte inside the 4-byte checksum varint (truncated body) |
| `udprepl/.../seed_10_corruption_wrong_proto_version` | production body mapping with envelope `proto_version = SupportedProtoVersion+1`, correct CRC |
| `udprepl/.../seed_11_corruption_undecodable` | hand-built Hello envelope with a 3-byte run_id, correct CRC (schema-domain violation) |
| `persist/.../seed_00_accepted_with_tombstone` | `persist.Encode` of two records (one data, one tombstone) |
| `persist/.../seed_01_accepted` | header-only dump `{"format_version":1}` |
| `persist/.../seed_02_generation_fault` | empty input |
| `persist/.../seed_03_generation_fault` | `{"format_version":2}` — unsupported generation |
| `persist/.../seed_04_per_record_discard` | header plus one record line whose checksum (0xDEADBEEF) does not cover it |
| `persist/.../seed_05_generation_fault` | header plus a valid record line plus a torn final line `{"key":"fuzz/torn"` |
| `persist/.../seed_06_generation_fault` | the valid dump cut 10 bytes off its final line |
| `persist/.../seed_07_per_record_discard` | the valid dump with its first payload base64 byte flipped (valid JSON/base64, checksum mismatch) |

## Decisions Made

- **External test package, production seam:** `FuzzDecode` sits in `package udprepl_test` and calls `DecodeDatagramForTest`, the existing test-binary-only wrapper around `decode()` — the production path without adding a `testpackage` exemption for a new filename.
- **Error-returning seed builders:** both seed builders return `([][]byte, error)` instead of taking `*testing.F`. The one-off corpus generator and the fuzz target then share the identical payload list, making committed-corpus drift structurally impossible.
- **Zero test-file coupling to plan 02-07:** the corruption classes are re-derived from their definitions (production encoder + hand-built envelopes) rather than importing `decode_negative_test.go` helpers, exactly as the plan's assumption requires.
- **Contract-only dump assertions:** the fuzz body asserts generation-vs-per-record fault shapes, checksum verification, line accounting, echo round-trip and tombstone-data honesty — nothing stronger than the codec guarantees, so it does not encode a Phase 3 fix as a fuzzer failure.
- **Union scan for tombstone-with-data:** the input scanner allows payload if any line for the key carried it; with last-wins duplicates and per-record discards, this is the only sound direction and can only fire on genuinely invented data.
- **Corpus from f.Add payloads, not cache mutants:** deterministic one-construction-per-class provenance, reviewable file by file, with the fuzz-cache mutations deliberately not committed.

## Deviations from Plan

None - plan executed exactly as written. The plan explicitly allowed either committing fuzz-written corpus entries or hand-writing them from the `f.Add` payloads; the payload path was chosen, and both targets were still fuzz-smoked with bounded `-fuzztime` as the plan requires. No production code was changed and no new dependency was added.

## Issues Encountered

- **Base64 padding trap in the flipped-byte fixture:** the first flip targeted the final padded quantum's last character, which only feeds a dropped output byte — the decoded payload, and therefore the checksum, stayed valid, and the corpus generator classified the seed as `accepted_with_tombstone`. Fixed by flipping the token's first character and adding a `base64` decode comparison guard (`fuzzDumpPayloadChanged`) so the fixture cannot silently stop representing its class.
- **Lint fixes on the new files:** `err113` demanded wrapped static errors for the two construction-guard failures (package-level sentinels added), `errchkjson` proved the fixture `json.Marshal` cannot fail (error deliberately dropped with a comment), and `noinlineerr` required plain assignments before the error checks. All fixed before the task commit; no assertion changed.

## User Setup Required

None - no external service configuration required. No new environment variables; the fuzz targets read nothing but their seeds.

## Known Stubs

None. Both targets exercise production code (`decode()` and `persist.Decode`) over real committed inputs; the corpus entries are data, not mocks.

## Threat Flags

None — test-only additions. They implement the T-2-14 mitigation (corpora derive only from in-repo encoded messages and their corruptions; every committed file's origin is listed above) and extend TEST-08's ASVS V5 input-validation pin; no new network endpoint, auth path, file-access pattern or schema change at a trust boundary.

## Next Phase Readiness

- **Plan 02-11 (scheduled.yml):** the deep-fuzz jobs can invoke exactly these targets — `go test -fuzz='^FuzzDecode$' -fuzztime=<d> ./internal/udprepl/` and `go test -fuzz='^FuzzDumpDecode$' -fuzztime=<d> ./internal/app/persist/` — with the one-target-per-package property already satisfied and a fast PR-CI path (plain `go test` runs the 20 committed seeds).
- **Phase 3 (XPORT-03/XPORT-10):** the decoder-side size bound and empty-key/tombstone-with-data rejection must land without violating these invariants; the seeds that currently decode (oversize is not seeded — it exercises no new verdict; empty-key and tombstone-with-data live in the negative table) stay valid inputs either way.
- **TEST-08 stays open in REQUIREMENTS.md:** this plan supplies the fuzz half (targets + corpora) on top of plan 02-07's negative table, but `requirements.ready-ids` reports TEST-08 blocked because plan 02-11 (the scheduled workflow that runs the deep-fuzz jobs) still declares it; `mark-complete` fires when that last declaring plan lands.
- **TEST-02 stays with plan 02-15:** its simulator deletion level is the remaining half; this plan touches neither simulator nor stand.

## Self-Check

- [x] `internal/udprepl/fuzz_decode_test.go` exists (`FuzzDecode` + seed builder + construction guards)
- [x] `internal/app/persist/fuzz_dump_test.go` exists (`FuzzDumpDecode` + seed builder + echo/scan helpers)
- [x] `internal/udprepl/testdata/fuzz/FuzzDecode/` holds 12 entries, all starting with `go test fuzz v1`
- [x] `internal/app/persist/testdata/fuzz/FuzzDumpDecode/` holds 8 entries, all starting with `go test fuzz v1`
- [x] Task commit `efe7d3f` is an ancestor of HEAD (no deletions in it)
- [x] `go test -race -count=1 ./internal/udprepl/ -run '^FuzzDecode$'` passes (plan verify 1a)
- [x] `go test -race -count=1 ./internal/app/persist/ -run '^FuzzDumpDecode$'` passes (plan verify 1b)
- [x] Corpus check: `ls` over both directories lists 23 lines; the first entry's first line is `go test fuzz v1` (plan verify 2)
- [x] `go test -race ./internal/udprepl/ ./internal/app/persist/` green (plan verification)
- [x] `make lint` and `make test` pass repo-wide
- [x] `check evaluation-scope --plan 02-17` returns the commit and all 22 changed files, `missingOnDisk: []`, `outsideUnion: []`

**Self-Check: PASSED**

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
