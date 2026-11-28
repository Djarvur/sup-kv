---
phase: 09-documentation-and-release
plan: 02
subsystem: documentation
tags: [docs, formats, dump-format, jsonl, crc32, record-version, doc-anchor-tests, doc-is-test-input, m-24, tdd]

requires:
  - phase: 09-documentation-and-release (09-01)
    provides: the doc-is-test-input tracer shape (author from a machine source, verify by test, fail loudly on drift in both directions) and the sibling green cmd/sup-kv doc-anchor suite
  - phase: 05-persistence-and-api-foundations
    provides: the persist codec (EncodeTo/Decode/DecodeStats), storage.RecordChecksum's one checksum domain and the record_count census rules the doc documents
  - phase: 04-replication
    provides: the udprepl Envelope codec, the CRC-over-body coverage definition and the proto_version gate the envelope section documents
provides:
  - docs/FORMATS.md — the as-built format reference: JSONL dump grammar with the encoder's own example bytes (tombstone and live-empty shapes plus the zero-record generation), the checksum recipe, per-record rejection rules naming every DecodeStats class, the record_count integrity rules, the format-version policy, the datagram envelope with CRC coverage and verdict order, the RFC 9562 record-version layout, and manual decoding
  - internal/app/persist/formats_doc_test.go — the three-part byte-identical guard beside the codec (D-04's stated exception): TestFormatsDocEncodeMatchesExample, TestFormatsDocChecksumRecipe, TestFormatsDocRejectionRules; a codec format change without a doc regeneration fails make test
  - M-24's as-built truth (the dump is JSON Lines, not one JSON document) for the 09-04 registry sweep to cite
affects: [09-04 (registry sweep cites the M-24 as-built truth), 09-05 (OPERATIONS cross-links to the limits prose), DOCS-06 verification]

actuals:
  tokens: 7958
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "byte-identical doc fences: a marker line (<!-- formats-example -->) selects the next fenced ```dump block, whose body must equal persist.EncodeTo output over a shared fixed record set — the codec's own output committed verbatim, never hand-typed"
    - "reflection-enumerated class coverage: the doc's rejection rules are checked in both directions against reflect.TypeFor[persist.DecodeStats]() fields rendered snake_case, so a class the codec adds or removes without a doc edit fails"

key-files:
  created:
    - docs/FORMATS.md
    - internal/app/persist/formats_doc_test.go
    - .planning/phases/09-documentation-and-release/red-evidence/09-02-task1-red.json
  modified: []

key-decisions:
  - "the fixed record set uses node 0x5060 — the plan's 48-bit literal (0x102030405060) cannot fit the version layout's 16-bit NodeID domain (PackVersion takes uint16); doc and test share the corrected tuple, a Rule-1 plan-gloss correction"
  - "the doc documents the as-built data encodings the plan gloss missed: a tombstone is data:null and a live empty value is data:\"\" (a zero-length non-nil []byte renders as the empty base64 string), so the two edge shapes are distinguishable by the deleted flag alone"
  - "the dump loader does not validate the UUIDv7 layout or a future bound at load — class=version is hex/16-byte only (decodeRecord has no version.Validate call); the plan's 'version layout/future' gloss belongs to the replication receive path, and FORMATS.md documents the split with a cross-reference"
  - "the rejection-class enumeration rides reflect.TypeFor[persist.DecodeStats]() rather than parsing codec.go source text — the real class set is the type's own fields, immune to source reformatting, and a class the codec adds or removes without a doc edit fails"

patterns-established:
  - "formats-fence marker contract: the doc's example dumps are the codec's own output, regenerated after any codec change per the doc's stated maintenance contract; the failing test names the first differing line"
  - "honest-scope guard header: the test file states it is a byte-identity and recipe guard, not a truth claim about the format's wisdom"

requirements-completed: [DOCS-01, DOCS-06]

coverage:
  - id: D1
    description: "docs/FORMATS.md dump grammar: JSONL, header object (format_version 1, record_count), one sorted record line per record with the mandatory-sort/byte-stability contract, SetEscapeHTML(false), field table, tombstone and live-empty edge shapes, and both example fences byte-identical to EncodeTo over the fixed record set"
    requirement: DOCS-06
    verification:
      - kind: unit
        ref: "internal/app/persist/formats_doc_test.go#TestFormatsDocEncodeMatchesExample"
        status: pass
      - kind: other
        ref: "grep record_count/format_version/tombstone/CRC32/65-507 docs/FORMATS.md (all green)"
        status: pass
    human_judgment: false
  - id: D2
    description: "the checksum section states storage.RecordChecksum's exact domain (CRC32-IEEE over key bytes, 16 version bytes, data bytes, one live/tombstone flag byte) and the recipe recomputes every example record's decimal checksum"
    requirement: DOCS-06
    verification:
      - kind: unit
        ref: "internal/app/persist/formats_doc_test.go#TestFormatsDocChecksumRecipe"
        status: pass
    human_judgment: false
  - id: D3
    description: "the rejection-rules section names class=checksum/version/value_limit/malformed (every real DecodeStats class, no invented ones) plus the record_count integrity rules (incomplete / tolerated / legacy) and every generation-level fault the codec has"
    requirement: DOCS-06
    verification:
      - kind: unit
        ref: "internal/app/persist/formats_doc_test.go#TestFormatsDocRejectionRules"
        status: pass
    human_judgment: false
  - id: D4
    description: "format-version policy (v1 only, additive record_count rationale, unknown version unreadable, v1.0 legacy compat), datagram envelope and CRC coverage (fields and wire domains, CRC over body only, proto_version gated before CRC/body), record-version layout (RFC 9562 octet table, 32-hex rendering, (l, counter, nodeID) byte order, interop filler rule), manual decoding walk-through"
    requirement: DOCS-06
    verification:
      - kind: other
        ref: "authored from codec.go/udprepl codec.go/supkv.proto/version.go contracts read at execution; anchored by the three passing TestFormatsDoc tests and the doc greps"
        status: pass
    human_judgment: true
    rationale: "the prose sections are grounded in the sources read and pinned by the three machine tests where testable, but no automated check asserts every envelope/layout sentence — the anchor coverage is the three tests plus the greps recorded under D1-D3"
  - id: D5
    description: "M-24's as-built truth exists for the 09-04 registry sweep to cite: FORMATS.md states the dump is JSON Lines (not one JSON document) and supersedes the IDEA.md description"
    requirement: DOCS-01
    verification:
      - kind: unit
        ref: "internal/app/persist/formats_doc_test.go#TestFormatsDocEncodeMatchesExample (the JSONL shape is byte-identical to the encoder's output)"
        status: pass
    human_judgment: false

duration: 18min
completed: 2026-11-27
status: complete
---

# Phase 9 Plan 2: Format Reference Summary

**docs/FORMATS.md — the as-built JSONL dump grammar, checksum recipe, rejection rules and datagram-envelope reference whose example dumps ARE the encoder's bytes, pinned byte-identically by the three TestFormatsDoc guards beside the codec.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-11-27T21:34:00Z
- **Completed:** 2026-11-27T21:52:00Z
- **Tasks:** 2
- **Files modified:** 3 (2 production artifacts + 1 RED evidence record)

## Accomplishments

- The byte-identity guard was written FIRST and recorded RED: all three `TestFormatsDoc` tests failed exactly on the missing doc (`read ../../../docs/FORMATS.md: no such file or directory`), with `go vet` green first; the RED evidence record passed `gsd_run check tdd-red-evidence` with verdict RED_EVIDENCE_OK.
- `docs/FORMATS.md` ships complete in the plan's section order: `## Dump grammar` (header object, mandatory sort in the code's own terms — byte stability across generations of the same state —, field table, SetEscapeHTML(false), the tombstone `data:null/deleted:true` and live-empty `data:""` shapes, streaming one-write-per-line), `## Checksum` (the exact CRC32-IEEE domain over key ‖ version ‖ data ‖ one flag byte), `## Rejection rules` (per-record `class=malformed/version/value_limit/checksum` plus the generation-level faults and the record_count incomplete/tolerated/legacy rules), `## Format-version policy`, `## Datagram envelope and CRC coverage` (fields and wire domains, CRC over the marshaled body only, proto_version gated before CRC/body), `## Record-version layout` (RFC 9562 octet table, 32-hex rendering, `(l, counter, nodeID)` byte order, interop filler rule, the dump-vs-replication validation split) and `## Manual decoding` (a step-by-step read of the example with real values).
- Both example fences are `persist.EncodeTo`'s own bytes over the fixed record set (`alpha` live `hello-alpha`, `bravo` tombstone, `charlie` live empty value; node `0x5060`, counter 1, logical ms 1736937600000–002, minted one millisecond apart), generated by a scratch `go run` that was never committed; the zero-record fence is the empty-map output. The doc states the regeneration contract in its intro.
- The three-part guard is green under `-race`: byte identity for both fences, checksum recipe recomputation for every example line (via the shared `dumpLine`/`sealed` test helpers of the sibling codec suite), and the two-directional rejection-class coverage enumerated from `reflect.TypeFor[persist.DecodeStats]()` with a zero-token vacuity guard plus the record_count integrity anchors.
- The persist, storage and version suites are green under `-race`; the doc greps (record_count, format_version, tombstone, CRC32, 65 507) are green; the package lint is clean under the vendored strict config.
- M-24's as-built truth exists: the doc's intro states the dump is JSON Lines, not one JSON document, superseding the IDEA.md description — ready for the 09-04 registry sweep to cite.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the byte-identical encode test beside the codec** — `de58233` (test)
2. **Task 2: Author docs/FORMATS.md — grammar, checksum, rejection rules, policy, envelope, version layout** — `8a201cd` (feat)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `docs/FORMATS.md` — the as-built format reference: intro with the regeneration contract, dump grammar with both marked example fences, checksum recipe, rejection rules, format-version policy, datagram envelope and CRC coverage, record-version layout, manual decoding; cross-references OPERATIONS `## Record and datagram limits` and CONFIGURATION.
- `internal/app/persist/formats_doc_test.go` — the honest-scope guard: the fixed record set builder, the marker-fence and doc-section readers (fatal on missing/unclosed fences and headings), `TestFormatsDocEncodeMatchesExample` (both fences), `TestFormatsDocChecksumRecipe`, `TestFormatsDocRejectionRules` (reflection-enumerated classes both directions + record_count anchors).
- `.planning/phases/09-documentation-and-release/red-evidence/09-02-task1-red.json` — the RED evidence record (classifier RED_EVIDENCE_OK).

## Decisions Made

- The plan's node literal was corrected: `0x102030405060` is 48 bits and `version.PackVersion` takes a `uint16` node ID (the RFC 9562 layout carries a 16-bit NodeID in rand_b's high bits). The fixed record set uses the literal's low 16 bits, `0x5060`, and doc and test share the tuple. Recorded as a Rule-1 deviation below.
- The dump grammar documents the as-built data encodings the plan gloss missed: the scratch generator showed a tombstone encodes as `data:null` while a live empty value encodes as `data:""` — the doc presents both shapes explicitly instead of claiming one null shape for both.
- The `class=version` rule is documented honestly: the dump loader checks hex/16-byte shape only (`decodeRecord` has no `version.Validate` call), so the plan's "version layout/future" gloss was attributed to the replication receive path (where the layout and future-bound checks live) with a cross-reference, per the plan's own code-wins assumption.
- The rejection-class enumeration rides `reflect.TypeFor[persist.DecodeStats]()` over the exported type's fields rendered snake_case — the real class set without hand-copying, immune to codec.go source reformatting; a class added or removed without a doc edit fails in either direction.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The plan's 48-bit node literal cannot fit the 16-bit NodeID domain**
- **Found during:** Task 1 (fixed record set construction)
- **Issue:** the plan's `<behavior>` specifies `node=0x102030405060` for all three fixed versions, but the version layout carries a 16-bit NodeID (`version.PackVersion(l, counter uint16, nodeID uint16)`); the literal does not compile.
- **Fix:** the fixed record set uses the literal's low 16 bits (`0x5060`), documented in the test's comment; doc and test share the tuple exactly as the plan requires.
- **Files modified:** internal/app/persist/formats_doc_test.go
- **Verification:** TestFormatsDocEncodeMatchesExample green — the doc's example bytes match the encoder's output over the corrected set.
- **Committed in:** de58233 (Task 1 commit)

**2. [Rule 1 - Bug] The plan's "version layout/future → version class" gloss contradicts the dump codec**
- **Found during:** Task 2 (rejection-rules authoring)
- **Issue:** the plan's rejection-rule line assigns version-layout/future rejection to the dump's version class, but `decodeRecord` classifies `class=version` only when `record_version` is not 16 bytes of hex — no layout or future-bound validation happens at dump load.
- **Fix:** the doc states the dump's actual rule and cross-references the replication receive path where the layout nibble/variant checks and the `--max-clock-lead` future bound live; the plan's own assumption ("the code wins and the summary notes the delta") governs.
- **Files modified:** docs/FORMATS.md
- **Verification:** TestFormatsDocRejectionRules green (the class coverage is unaffected; the prose documents the split).
- **Committed in:** 8a201cd (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 plan-gloss-vs-code corrections). **Impact on plan:** none — both fixes follow the plan's own grounding rule that the code is authoritative over the gloss.

## Issues Encountered

- The first GREEN attempt failed the byte comparison: `EncodeTo`'s output ends with a trailing newline, so the fenced body needs a trailing empty line before the closing fence. The test's first-difference diagnostic named it immediately (`doc: ⟨no line⟩ / encode:` at the first differing line); adding the empty line before each closing fence resolved it. This is the guard working as designed, not a defect.

## TDD Gate Compliance

- Task 1 (`tdd="true"`) has its RED commit (`de58233`, `test(09-02)`) before the implementation commit (`8a201cd`, `feat(09-02)`, which descends from the RED); the RED evidence record passed `gsd_run check tdd-red-evidence` with verdict RED_EVIDENCE_OK. No REFACTOR commit was needed — the GREEN authoring needed no cleanup pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for 09-03 — the FORMATS reference is production quality and the M-24 as-built truth is in place for the 09-04 registry sweep; the doc's cross-reference targets (OPERATIONS `## Record and datagram limits`, CONFIGURATION) already exist and stay green.
- No blockers. DOCS-06 is marked Complete; DOCS-01 stays pending on the #2388 shared-ID gate (declared by nine sibling plans of this phase).

---
*Phase: 09-documentation-and-release*
*Completed: 2026-11-27*
