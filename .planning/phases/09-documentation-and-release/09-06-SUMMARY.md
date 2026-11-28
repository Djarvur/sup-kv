---
phase: 09-documentation-and-release
plan: 06
subsystem: documentation
tags: [api-reference, grpc, error-tables, watch-contract, key-prefix, doc-anchor-tests, tdd, mapper-derived]

requires:
  - phase: 09-documentation-and-release (09-01…09-05)
    provides: the green cmd/sup-kv doc-anchor suite this plan extends, the as-built PERFORMANCE.md ledger the latency-model section links, and the OPERATIONS.md reference the caveats point at
  - phase: 06-api-and-semantics
    provides: the grpcapi mapper (mapError's total table), the Watch hub contract, the List pagination modes and the public proto the reference mirrors
provides:
  - docs/API.md completed as the full integrator reference (D-05): the per-RPC reference in the proto's own declaration order (every RPC with semantics and a mapper-derived error table), the API-level Watch contract, the write-latency model as summary + the one-ledger link, the key-prefix convention, the consistency caveats, and the client patterns with proto obtainment
  - cmd/sup-kv/api_docs_test.go extended: TestAPIDocPerRPCSections (parses the public proto's rpc declarations and requires the RPC reference to mirror them in order, each section carrying its error-table marker), the DOCS-05 anchors, and the apiDocForbidden table that flips the pending-section literals into forbidden phrases
  - the M-19 API half and M-30 close: ResourceExhausted and Unavailable are named and explained in the API error tables, and the cache pattern + key-prefix convention are documented outside IDEA.md for the first time
affects: [09-07…09-10 (their doc edits must keep TestAPIDoc* green), 09-08 (TestProvenClaims scans the API text), 09-11 (milestone verification)]

actuals:
  tokens: 7400
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "schema-mirror doc check: the test parses rpc declarations out of the public proto at test time and requires the reference's ### headings to equal them in order — a missing RPC, an extra section or a reordered one all fail, so reference and schema cannot silently diverge"
    - "anchor flip as a forbidden table: the literals that pinned the pending section move to a whitespace-normalized forbidden table, so deleting the section without the flip fails the old gate and re-adding the section fails the new one"

key-files:
  created:
    - .planning/phases/09-documentation-and-release/red-evidence/09-06-task1-red.json
  modified:
    - docs/API.md
    - cmd/sup-kv/api_docs_test.go

key-decisions:
  - "The proto wins over the plan's nine-RPC gloss: the schema declares seven RPCs (SetIfAbsent is Set's create_if_absent variant and WaitForReplication is Set/Delete's wait fields, never separate RPCs), so the vacuity floor is 7, not the plan's ≥9 — the plan's own assumption names the proto as the source"
  - "the per-RPC error tables factor the two every-RPC facts into the RPC reference intro (gate-closed Unavailable before any validation; transport ResourceExhausted at MaxRecvBytes) and list only each RPC's input-dependent codes, so Set's table does not repeat the gate row seven times and the transport verdict is documented once where the ceiling lives"
  - "SnapshotCreate/SnapshotLoad tables carry PermissionDenied and Internal rows derived from persistfile's classifySaveError plus the mapper's totality fallback — beyond the plan's sketch (NotFound/FailedPrecondition/Unavailable) — because the tables must list exactly what the mapper produces for the RPC's inputs, including the classified filesystem faults and the unclassified-read fallback"
  - "List's table carries no Internal row and says so: the listing has no engine error path past the gate, so the absence is deliberate and stated — the same honest-absence idiom the Watch table uses for its single terminal error"
  - "the '## Watch contract' section is additive to the existing '## Watch' mechanics section (the plan keeps every existing section intact); the rename-free coexistence keeps the existing {'## Watch'} anchor green by substring while the contract section states the integrator-level guarantee (exact cut, dump-confirmed announcements, bounded pending, Unavailable shutdown, one-way health)"
  - "the forbidden set is exactly four phrases ('What Phase 9 adds', 'Treat those sections as pending', 'per-RPC failure table', 'DOCS-05'); 'write-latency model' and 'key-prefix convention' were NOT forbidden because the shipped sections carry those names as headings — their flip is the new capitalized heading anchors"

patterns-established:
  - "single-line anchor discipline: raw strings.Contains anchors must sit on one doc line — the authored doc is reflowed so each mandated literal is contiguous (three reflows during GREEN); whitespace normalization is reserved for the forbidden table per the operations idiom"

requirements-completed: [DOCS-01, DOCS-05]

coverage:
  - id: rpc-reference-structure
    description: "The RPC reference covers every schema-declared RPC in the proto's declaration order, each section carrying its error-table marker; the order is parsed from the proto at test time"
    requirement: DOCS-05
    verification:
      - kind: tests
        ref: cmd/sup-kv/api_docs_test.go#TestAPIDocPerRPCSections
        status: pass
      - kind: tests
        ref: ".planning/phases/09-documentation-and-release/red-evidence/09-06-task1-red.json (RED evidence, classifier RED_EVIDENCE_OK)"
        status: pass
    human_judgment: false
  - id: anchor-flip
    description: "The DOCS-05 anchors pin the new sections and the pending-section literals are forbidden — deleting the pending section without the flip fails, and its return fails harder"
    requirement: DOCS-05
    verification:
      - kind: tests
        ref: cmd/sup-kv/api_docs_test.go#TestAPIDoc (forbidden check first, then the anchor table)
        status: pass
    human_judgment: false
  - id: integrator-sections
    description: "Watch contract (exact cut, dump-confirmed announcements, bounded pending, Unavailable shutdown, one-way health), write-latency model linking docs/PERFORMANCE.md, key-prefix convention (enforced by documentation), consistency caveats (read-your-writes, the 14.5(6) Unavailable window) and client patterns (last-read-value cache, proto obtainment, examples/client) present with their mandated statements"
    requirement: DOCS-05
    verification:
      - kind: tests
        ref: cmd/sup-kv/api_docs_test.go#TestAPIDoc (the DOCS-05 anchor rows)
        status: pass
      - kind: command
        ref: "grep gate: no 'What Phase 9 adds'/'treat those sections as pending'; docs/PERFORMANCE.md, supkvapi.proto, ResourceExhausted, key-prefix present"
        status: pass
    human_judgment: false
  - id: one-ledger-rule
    description: "The write-latency model section is summary + link; the ledger's figures and conditions are never duplicated"
    requirement: DOCS-05
    verification:
      - kind: command
        ref: "grep -cE '[0-9]' over the ## Write-latency model section = 0; the docs/PERFORMANCE.md link present"
        status: pass
    human_judgment: false
  - id: error-table-truth
    description: "Every per-RPC error table lists exactly the codes the grpcapi mapper produces for that RPC's inputs (Watch's single terminal error; List's honest Internal absence; snapshot RPCs' classified faults)"
    requirement: DOCS-05
    verification: []
    human_judgment: true
    rationale: "Truth-of-rows against the mapper is a manual read-through — the tables were derived line-by-line from internal/grpcapi/server.go (mapError + handler gates), list.go, watch.go and internal/persistfile/snapshot.go (classifySaveError), and the anchor tests deliberately pin section presence and the error-table marker, not the matrix contents. This is the doc-anchor tests' own honest-scope boundary and the phase verification's manual UAT territory."

# Metrics
duration: 23min
completed: 2026-11-27
status: complete
---

# Phase 9 Plan 6: The Complete API Reference Summary

docs/API.md is now the full integrator reference (D-05): seven per-RPC sections in the proto's own declaration order with mapper-derived error tables, the API-level Watch contract, the write-latency model as summary plus the one-ledger link, the key-prefix convention, the consistency caveats and the client patterns — and the pending section it replaces is now a forbidden-phrase set, machine-checked by the flipped anchors.

## Performance

- **Duration:** 23 min
- **Started:** 2026-11-27T23:27Z
- **Completed:** 2026-11-27T23:50Z
- **Tasks:** 2
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- **Task 1 (RED, TDD).** `cmd/sup-kv/api_docs_test.go` extended: `TestAPIDocPerRPCSections` reads `../../api/supkvapi/v1/supkvapi.proto` by relative path, parses the `service SupKVService` block's rpc declarations in source order (structural parse, vacuity floor 7), and requires the `## RPC reference` section's `### ` headings to equal the parsed declarations in order — a missing RPC, an extra section or a reordered one all fail — with each section carrying its `#### Errors` marker; the DOCS-05 anchor rows join `apiDocAnchors` (16 new rows), and the five literals that pinned `## What Phase 9 adds` move into the new whitespace-normalized `apiDocForbidden` table, which `TestAPIDoc` runs before the anchor loop so a returned pending section is named even while sections are missing. RED recorded (red-evidence/09-06-task1-red.json, classifier RED_EVIDENCE_OK): `TestAPIDocPerRPCSections` failed naming the missing section with the schema's declaration order, and `TestAPIDoc` failed on the stale claim "What Phase 9 adds" — the pending literals are now forbidden and currently present; vet and lint clean, so the failures were assertion failures, not build faults.
- **Task 2 (GREEN).** docs/API.md's pending tail replaced, in order: `## RPC reference` (intro factoring the two every-RPC facts — gate-closed `Unavailable` before any validation, transport `ResourceExhausted` at `MaxRecvBytes` — then `### Get`, `### Set`, `### Delete`, `### List`, `### Watch`, `### SnapshotCreate`, `### SnapshotLoad`, each with semantics in the proto's own field names and an error table derived from the mapper); `## Watch contract` (exact snapshot-then-live cut, dump-confirmed announcements, bounded pending with loud shed, shutdown `Unavailable` never OK, one-way health); `## Write-latency model` (three sentences + the docs/PERFORMANCE.md link, zero digits — the grep proves it); `## Key-prefix convention` ('/'-prefixed per-service namespaces, enforced by documentation, the store stays prefix-agnostic); `## Consistency caveats` (LWW last-writer-wins, node-local read-your-writes, cross-node bounds linking OPERATIONS, the listed-but-unarrived window answering Unavailable, not NotFound per 14.5(6)); `## Client patterns` (the last-read-value cache as consumer-library territory per ADR 0007/14.7, and obtaining the proto: the source path, the committed Go package, reflection, and the `examples/client` module `example.com/sup-kv-client`). The Errors section's pending-pointer sentence — which carried two now-forbidden literals — was rewritten to point at the RPC reference. M-19's API half and M-30 close.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the per-RPC section anchors and the pending-section flip** - `a38e9f8` (test)
2. **Task 2: the API.md extension (GREEN)** - `49cc7ab` (feat)

_Note: TDD task 1 produced the RED commit; Task 2 is its GREEN._

## Files Created/Modified

- `cmd/sup-kv/api_docs_test.go` - the schema-mirror order test, the DOCS-05 anchor rows and the forbidden pending-literal table
- `docs/API.md` - the six new sections replacing the pending tail; the Errors intro's pending pointer rewritten
- `.planning/phases/09-documentation-and-release/red-evidence/09-06-task1-red.json` - the classifier-validated RED record

## Decisions Made

- **The proto wins over the plan's nine-RPC gloss.** The schema declares seven RPCs — the plan's must_haves listed SetIfAbsent and WaitForReplication as RPCs, but the proto (the plan's own declared source) expresses them as `create_if_absent` and the wait fields. The vacuity floor is 7 with a comment naming the count; the order check fails on any real drift, so nothing is weakened.
- **Gate and transport factored into the reference intro.** The per-RPC tables list input-dependent codes; the gate-closed `Unavailable` and the transport-ceiling `ResourceExhausted` are stated once for every RPC instead of being repeated per table, with the scoping said explicitly ("the tables below state what each RPC's inputs can produce once the gate is open").
- **Snapshot RPC tables exceed the plan's sketch to stay total.** `classifySaveError`'s classified filesystem faults (`ResourceExhausted`/`PermissionDenied`) and the mapper's totality fallback (`Internal`) are rows on SnapshotCreate/SnapshotLoad because the mapper produces them for those inputs; a table that omitted them would promise less than the mapping does.
- **Honest absences are stated, not silent.** List's table carries no `Internal` row (no engine error path past the gate) and Watch's table carries no `Internal` row (there is no watch-failure path) — both sections say the absence is deliberate.
- **Watch contract is additive, not a rename.** The existing `## Watch` mechanics section stays intact (the plan mandates it); the new `## Watch contract` states the integrator-level guarantee without repeating the mechanics' numbers, and the existing `{"## Watch"}` anchor stays green by substring.
- **Forbidden set scoped to the four pending artifacts.** "write-latency model" and "key-prefix convention" become real section names, so forbidding them would fight the shipped headings; their pending-section anchor rows are replaced by the capitalized heading anchors.
- **Anchor literals are single-line checked** (`strings.Contains` on the raw doc), so the doc was reflowed to keep each mandated literal contiguous — recorded as a pattern; whitespace normalization stays with the forbidden table per the operations idiom.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan fact] The schema declares seven RPCs, not nine**
- **Found during:** Task 1 (vacuity floor selection)
- **Issue:** the plan's behavior bullet sets "vacuity: ≥9 RPC sections", but the proto declares seven (SetIfAbsent/WaitForReplication are not RPCs); the plan's own assumptions clause makes the proto the source
- **Fix:** floor set to 7 with a comment naming the seven and why the two are not RPCs
- **Files modified:** cmd/sup-kv/api_docs_test.go
- **Verification:** the parse finds 7 at RED and GREEN; the order check is exact-equality, so any schema/doc drift still fails
- **Committed in:** a38e9f8

**2. [Rule 1 - Plan fact] The Errors section's pending-pointer sentence had to be rewritten**
- **Found during:** Task 2 (GREEN)
- **Issue:** the plan says "keep every existing section intact above the extension", but the Errors section's intro carried "the exhaustive per-RPC failure table is Phase 9/DOCS-05" — two literals the same plan's flip makes forbidden; keeping it verbatim fails the new gate
- **Fix:** that one sentence rewritten to point at the RPC reference; every other existing section byte-untouched
- **Files modified:** docs/API.md
- **Verification:** the forbidden greps are negative; the whole TestAPIDoc* suite green
- **Committed in:** 49cc7ab

---

**Total deviations:** 2 auto-fixed (2 × Rule 1 plan-fact corrections). **Impact on plan:** none on scope — both make the plan's own flip semantics executable.

## Issues Encountered

Three anchor literals were initially split across doc line breaks (raw `strings.Contains` cannot see them); the doc was reflowed so each mandated literal sits on one line. Cosmetic, resolved inside Task 2 before its verify ran.

## Verification

- Task 1 verify: `go test -race -count=1 ./cmd/sup-kv/ -run 'TestAPIDoc' -v` — both target tests FAIL on their planned assertions (missing `## RPC reference`; stale claim "What Phase 9 adds"); the log grep gate passes; `go vet ./cmd/sup-kv/` clean — PASS.
- Task 2 verify 1: the same filter — PASS (both tests green).
- Task 2 verify 2: `! grep 'What Phase 9 adds'` + `! grep -i 'treat those sections as pending'` + `docs/PERFORMANCE.md`/`supkvapi.proto`/`ResourceExhausted`/`key-prefix` present — PASS.
- Task 2 verify 3: `go test -race -count=1 ./cmd/sup-kv/` — the whole package green (21–22 s) — PASS.
- Latency-model no-figures: digits in the `## Write-latency model` section = 0 — PASS.
- `go tool golangci-lint run ./cmd/sup-kv/` — clean (one map-literal style fix landed inside the RED commit before its evidence was captured).

## TDD Gate Compliance

- RED: `test(09-06)` commit a38e9f8 precedes GREEN; both target tests failed on their planned assertions; classifier verdict RED_EVIDENCE_OK.
- GREEN: `feat(09-06)` commit 49cc7ab makes both pass under -race.
- REFACTOR: none needed (the one lint style fix landed inside the RED commit before its evidence was recorded).

## Self-Check: PASSED

- All tasks executed and committed individually (a38e9f8, 49cc7ab).
- All acceptance criteria re-run after the final edit: the TestAPIDoc* suite under -race, the pending-literal greps, the content greps, the whole cmd/sup-kv package, vet and lint — all green.
- Success criteria: the per-RPC sections match the proto's declaration order (test-enforced); the error tables are grounded in the mapper with Watch's honest single terminal error; the latency model links the ledger without duplicating a single figure; the key-prefix convention and the cache pattern are documented outside IDEA.md; the pending section, its anchors and every pending wording are gone/flipped.

Next: Ready for 09-07.
