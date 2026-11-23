---
phase: 06-api-and-semantics
plan: 10
subsystem: documentation-ripple
tags:
  - apis-01
  - apis-02
  - apis-06
  - apis-07
  - examples-doc
  - multi-node-guide
  - spec-registry
  - repository-layout
  - public-api-path
  - in-05-closure
  - full-rollback
  - keyset-pagination
requires:
  - phase: 06-api-and-semantics (06-01)
    provides: the public api/supkvapi/v1 schema and the sup-kv-public-api-no-internal depguard rule
  - phase: 06-api-and-semantics (06-05)
    provides: the two List modes, the page token and the page-size resolution
  - phase: 06-api-and-semantics (06-08)
    provides: the snapshot-load full rollback and its broadcast report
  - phase: 06-api-and-semantics (06-09)
    provides: docs/API.md, the client-visible contract these surrounding docs point at
provides:
  - "docs/EXAMPLES.md: the paginated List walk (keys + records + nextPageToken), the legacy unset-page_size keys-only mode, the zero-known-peers wait answer and the SnapshotLoad full-rollback statement"
  - "docs/04-MULTI-NODE.md: the WritePeers write-time set, the zero-peer answer and IN-05 closed by APIS-01"
  - "docs/SPEC-DEVIATIONS.md: S-PERS-12 implemented with rollback code and test evidence; all five API-proto evidence links on the public path"
  - "AGENTS.md: api/supkvapi/v1 and examples/client in the Layout, the public-schema dependency-policy sentence and the three-rule depguard deviation 1"
affects:
  - 06-13-phase-gate
  - phase-9-docs
actuals:
  tokens: 4627
  tasks: 2
  commits: 2
  plan_head_before: bd85af0375541451cbd2fd07cb3eb6e8e86358b0
  plan_head_after: 570a231dfa2c3350c9491e48c52846aca282b070
tech-stack:
  added: []
  patterns:
    - "A documentation sweep runs as a tracer: the legacy-path grep gate is taken from its current count to zero before any content edit, so the path move is proven end-to-end first."
    - "The executable doc demonstrates a new API mode with a real request/response walk (two pages plus the legacy call), not prose alone — the CI-run block is the proof."
key-files:
  created: []
  modified:
    - docs/EXAMPLES.md
    - docs/04-MULTI-NODE.md
    - docs/SPEC-DEVIATIONS.md
    - AGENTS.md
key-decisions:
  - "The path swap carried the evidence line numbers with it: three of the five links pointed at the wrong declaration after 06-01 moved and rewrote the proto, so each was re-derived to its actual line (S-ALG-CU-11 :63-80, S-SEM-04 :63-69, S-IMP-01 :66, S-IMP-02 :67-68,86-87, S-IMP-03 :39)."
  - "EXAMPLES demonstrates both List modes with a real two-page walk over the two live demo keys: page 1 (page_size 1) returns demo/hello with nextPageToken ZGVtby9oZWxsbw, page 2 resumes with that token and returns demo/wait with the empty token omitted, then the delete and the unset-page_size legacy listing prove the keys-only mode. The token is the raw-URL base64 of demo/hello, so the literal stays valid top to bottom."
  - "The zero-known-peers answer is documented as an honest false with zeroed counts (both fields omitted by grpcurl), never as a success-shaped response; the wait paragraph also states the set is captured once at write acceptance."
  - "docs/04-MULTI-NODE.md names the WritePeers seam, states that an Ack outside the captured set is inert, gives the zero-peer answer, and closes IN-05: the discovery flush broadcasts and re-arms the ladder but never inserts the discovered peer into a wait set or per-peer ledger. The stale Phase 6 handoff bullet became a statement that docs/API.md ships the contract."
  - "S-PERS-12 flips to implemented with the sorted-union replay, rollbackKey and restoreChanges links plus the engine and RPC rollback tests; S-PERS-13 and every PROTO-08 row are untouched."
  - "AGENTS.md names api/supkvapi/v1 and the nested examples/client module, states the public-schema/internal-replication split in the dependency policy, and deviation 1 names all three depguard rules with 'These rules live in .golangci.yml'; deviations 2-3 and the Go version policy are byte-identical."
  - "requirements-completed stays empty: requirements.ready-ids reports APIS-01/APIS-02/APIS-06/APIS-07 all blocked by the #2388 shared-ID gate, so 06-13 closes the checkboxes (06-05/06-06/06-09 precedent)."
patterns-established:
  - "A docs ripple plan proves its path move first with a zero-count grep gate over the non-historical corpus, then proves each behavioral statement with its own literal gate plus the executable doc test."
  - "Historical evidence is excluded structurally (docs/reviews/**) and pinned by git diff --exit-code, so the sweep can never rewrite the record it cites."
requirements-completed: []
coverage:
  - id: D1
    description: "Every non-historical doc names the public api/supkvapi/v1 schema (five SPEC-DEVIATIONS evidence links, including re-derived line numbers); docs/reviews stays untouched"
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "grep -rn 'internal/grpcapi/apiproto' docs AGENTS.md --include='*.md' | grep -v 'docs/reviews' | wc -l -> 0 (was 5)"
        status: pass
      - kind: process
        ref: "grep -q 'api/supkvapi/v1' docs/SPEC-DEVIATIONS.md && grep -q 'api/supkvapi/v1' AGENTS.md -> exit 0"
        status: pass
      - kind: process
        ref: "git diff HEAD --exit-code -- docs/reviews -> exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/EXAMPLES.md shows the page_size > 0 List response (keys, records with values, nextPageToken), states the unset/0 page_size legacy keys-only listing, and states the zero-known-peers acked=false/total_peers=0 answer; the edited block runs verbatim"
    requirement: "APIS-02, APIS-01"
    verification:
      - kind: process
        ref: "grep -q nextPageToken docs/EXAMPLES.md && grep -q records docs/EXAMPLES.md && grep -qi 'zero known peers' docs/EXAMPLES.md -> exit 0"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestExamplesDoc$' -v -> PASS (4.14s); re-run on the committed tree -> ok github.com/Djarvur/sup-kv/cmd/sup-kv 5.870s"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/EXAMPLES.md's Snapshot round-trip section states SnapshotLoad is a full rollback: live keys absent from the snapshot are deleted with fresh tombstones that replicate to peers, not an additive replay"
    requirement: "APIS-07"
    verification:
      - kind: process
        ref: "grep -qi 'full rollback' docs/EXAMPLES.md && grep -qi 'absent from the snapshot' docs/EXAMPLES.md -> exit 0"
        status: pass
      - kind: process
        ref: "the paragraph names fresh versions, rollback tombstones, peer replication and last-write-wins in one statement (read-through against internal/app/persist/snapshot.go:130-180)"
        status: pass
    human_judgment: false
  - id: D4
    description: "docs/04-MULTI-NODE.md describes the WritePeers write-time peer set and the zero-peer answer, and states IN-05 is closed by APIS-01 (the discovery flush no longer grows the awaited set or ledger); the old 'Known limitation IN-05' paragraph is gone"
    requirement: "APIS-01"
    verification:
      - kind: process
        ref: "grep -q 'closed in v1.1' docs/04-MULTI-NODE.md && ! grep -q 'Known limitation IN-05' docs/04-MULTI-NODE.md && grep -q 'WritePeers' docs/04-MULTI-NODE.md -> exit 0"
        status: pass
      - kind: process
        ref: "internal/grpcapi/gate.go:83 declares WritePeers func() []uint16; internal/grpcapi/wait.go:91-97 captures it once per write"
        status: pass
    human_judgment: false
  - id: D5
    description: "docs/SPEC-DEVIATIONS.md's S-PERS-12 row is implemented with links to the rollback code (replaySnapshot/replayKeys, rollbackKey, restoreChanges) and the rewritten tests; S-PERS-13 and every PROTO-08 row are unchanged"
    requirement: "APIS-07, APIS-06"
    verification:
      - kind: process
        ref: "grep -q '| S-PERS-12 .*implemented' docs/SPEC-DEVIATIONS.md && ! grep -q '| S-PERS-13 .*implemented' docs/SPEC-DEVIATIONS.md -> exit 0"
        status: pass
      - kind: process
        ref: "cited code read: internal/app/persist/snapshot.go:130-180 (replaySnapshot, replayKeys), :217-230 (rollbackKey), :232-260 (restoreChanges); tests exist at internal/app/persist/engine_test.go:1641 (#TestEngine_restoreReturnsReplayed), :1764 (#TestEngine_restoreExtraLiveKeyIsNotNoOp) and internal/grpcapi/rollback_test.go:64 (#TestServer_snapshotLoadRollback)"
        status: pass
    human_judgment: false
  - id: D6
    description: "AGENTS.md Layout names api/supkvapi/v1 and examples/client; the dependency policy states the API schema is public while the replication proto stays internal; deviation 1 names sup-kv-public-api-no-internal and says 'These rules live in .golangci.yml'; deviations 2-3 and the Go version policy are byte-identical"
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "grep -q 'sup-kv-public-api-no-internal' AGENTS.md && ! grep -q 'Both rules live' AGENTS.md -> exit 0"
        status: pass
      - kind: process
        ref: "grep -q 'api/supkvapi/v1' AGENTS.md && grep -q 'examples/client' AGENTS.md -> exit 0; git diff -U0 AGENTS.md shows no changed line in deviations 2-3 or the Go version policy"
        status: pass
      - kind: process
        ref: ".golangci.yml:89 declares sup-kv-public-api-no-internal denying github.com/Djarvur/sup-kv/internal under '**/api/**'"
        status: pass
    human_judgment: false
duration: 3min
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 10: Documentation Ripple Summary

**The surrounding docs now state the shipped phase behavior — the public `api/supkvapi/v1` path everywhere non-historical, an executable two-page List walk with `records`/`nextPageToken` beside the legacy keys-only mode, the honest zero-peer wait answer, the `WritePeers` write-time set with IN-05 closed, the SnapshotLoad full rollback, and the three-rule depguard boundary in the repository layout — while `docs/reviews/**`, S-PERS-13 and every PROTO-08 row stay untouched.**

## Performance

- **Duration:** 3 min (200 s)
- **Started:** 2026-11-23T01:44:07Z
- **Completed:** 2026-11-23T01:47:27Z
- **Tasks:** 2
- **Files modified:** 4
- **Commits:** 2 measured from ledger `bd85af0..570a231`

## Accomplishments

- **Public-path sweep (Task 1, tracer).** The legacy-path gate went from 5 occurrences to 0 over `docs` and `AGENTS.md` excluding `docs/reviews`. The five `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` links became `api/supkvapi/v1/supkvapi.proto`, with the line numbers re-derived to the actual declarations after 06-01's move and rewrite. `AGENTS.md` gained the `api/supkvapi/v1/` and `examples/client/` Layout entries.
- **List and wait in `docs/EXAMPLES.md`.** The Delete-and-list block now walks both modes against real nodes: page 1 (`page_size 1`) returns `keys` + `records` (with the base64 value) + `nextPageToken`, page 2 resumes with that token and returns the last key with the empty token omitted, then the delete and the unset-`page_size` listing prove the legacy keys-only response. The wait paragraph states the write-time peer set and the zero-known-peers `acked=false`/zero-counts answer.
- **Snapshot full rollback in `docs/EXAMPLES.md`.** The Snapshot round-trip section names `SnapshotLoad` a full rollback: fresh-version replay, deletion of live keys absent from the snapshot with fresh tombstones, and those tombstones replicating to peers like any write.
- **IN-05 closed in `docs/04-MULTI-NODE.md`.** The wait section now describes the `ServerOptions.WritePeers` seam, the write-time captured set, the inert outside Ack, and the zero-peer answer; the limitation paragraph became "IN-05 is closed in v1.1 (APIS-01)" explaining that the discovery flush broadcasts and re-arms but no longer grows the awaited set or ledger. The stale Phase 6 handoff bullet now records the shipped `docs/API.md` contract.
- **Registry row and layout.** `S-PERS-12` is `implemented` with the union/rollback code and both engine/RPC test families as evidence; `S-PERS-13` and the PROTO-08 rows are untouched. `AGENTS.md` states the public-schema/internal-replication split and names the third depguard rule in deviation 1.

## TDD Evidence

The plan is a documentation ripple, so no artificial RED/GREEN pair was manufactured (workflow note: keep the doc-anchor tests green). `TestExamplesDoc` — the executable contract for every `examples-verify` block — passes against the edited document, both on the working tree and on the committed tree.

## Task Commits

1. **Task 1: public-path sweep across the non-historical docs** — `ae8af71` (docs)
2. **Task 2: List/wait/rollback/IN-05 content ripple** — `570a231` (docs)

**Plan metadata:** the final `docs(06-10)` commit.

## Files Created/Modified

- `docs/EXAMPLES.md` — the paginated List walk plus the legacy mode, the zero-known-peers wait paragraph and the SnapshotLoad full-rollback statement.
- `docs/04-MULTI-NODE.md` — the `WritePeers` write-time set, the zero-peer answer and the IN-05 closure; the shipped `docs/API.md` handoff.
- `docs/SPEC-DEVIATIONS.md` — the five public-path evidence links and the `S-PERS-12` implemented row.
- `AGENTS.md` — the public API and example-module Layout entries, the dependency-policy sentence and the three-rule deviation 1.

## Decisions Made

- The evidence links were re-derived line by line, not path-swapped alone: after 06-01's move, `S-SEM-04`, `S-IMP-01`, `S-IMP-02` and `S-IMP-03` pointed at neighbouring declarations, which would have been a false citation even with the right path.
- The List example is a real two-page walk rather than a static shape: the token literal `ZGVtby9oZWxsbw` is the raw-URL base64 of `demo/hello`, deterministic in the doc's own flow, so copy-paste works top to bottom and the empty token is proven as the end signal.
- The legacy mode is demonstrated after the paginated pages on the same block, so a reader sees both response shapes from one running pair instead of two disconnected snippets.
- IN-05's closure is stated as a behavior change (the discovered peer never joins the wait set or ledger) with `docs/API.md` named as the client contract, and the old handoff bullet was retired so the guide cannot advertise pending work that shipped.
- Only `S-PERS-12` flips; `S-PERS-13` and every PROTO-08 row stay for Phase 9's registry audit, gated in both directions.
- `requirements-completed` stays empty: `requirements.ready-ids` reports all four plan requirements blocked by the #2388 shared-ID gate, so plan 06-13 closes the checkboxes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stale proto line numbers re-derived with the path swap**
- **Found during:** Task 1
- **Issue:** The five evidence links still carried pre-move line numbers; after the public proto rewrite, `S-IMP-01 :67` pointed at `wait_for_replication` instead of `create_if_absent`, `S-IMP-02 :68-69,87-88` missed both `wait_for_replication` fields, `S-IMP-03 :40` pointed at the `SnapshotCreate` comment instead of `rpc Watch`, and `S-SEM-04 :30-45` did not contain `create_if_absent`.
- **Fix:** Re-derived each range against the current `api/supkvapi/v1/supkvapi.proto` (`:63-80`, `:63-69`, `:66`, `:67-68,86-87`, `:39`).
- **Files modified:** `docs/SPEC-DEVIATIONS.md`
- **Verification:** the path gate is 0 and each cited line was read back with `awk`; `TestExamplesDoc` and the registry grep gates pass.
- **Commit:** `ae8af71`

**2. [Rule 1 - Bug] Retired the stale Phase 6 handoff bullet after closing IN-05**
- **Found during:** Task 2
- **Issue:** The Handoffs section still said Phase 6 owns the "final `docs/API.md` wording … including the IN-05 limitation described above", contradicting the closure statement added in the same edit.
- **Fix:** Replaced the bullet with a Phase 9 handoff recording that `docs/API.md` now ships the wait contract and IN-05 is closed.
- **Files modified:** `docs/04-MULTI-NODE.md`
- **Verification:** the IN-05 grep gate passes and the section no longer references a limitation.
- **Commit:** `570a231`

## Issues Encountered

The rollback grep gate initially failed because the mandated phrase straddled a line break (`absent from the` / `snapshot`); the paragraph was reflowed so the literal sits on one line. No other issues.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None.

## Threat Flags

None — documentation only. The plan's T-06-30 (stale paths) and T-06-31 (overclaimed registry rows) mitigations are the grep gates recorded above; no new trust surface was introduced.

## Next Phase Readiness

The surrounding docs now match the shipped phase behavior and the public schema path. One plan remains in Phase 6: **06-13** (the phase gate). No blockers; the registry audit's remaining rows (`S-PERS-13`, PROTO-08 family) are Phase 9's by design.

## Self-Check: PASSED

- Both task commits are ancestors of HEAD: `ae8af71` (Task 1), `570a231` (Task 2).
- All four modified files exist: `docs/EXAMPLES.md`, `docs/04-MULTI-NODE.md`, `docs/SPEC-DEVIATIONS.md`, `AGENTS.md`.
- `docs/reviews` is byte-identical to HEAD (`git diff HEAD --exit-code` exit 0).
- `TestExamplesDoc` passes on the committed tree.
