---
phase: 05-persistence
plan: 11
subsystem: persistence
tags: [deviation-registry, spec-deviations, adr-0012, tie-break, value-limit, load-outcomes, docs, persist-01, persist-02, persist-07]
requires:
  - phase: 05-persistence
    provides: the one value limit, the configured-limit decoder and the P-01 repair from 05-01
  - phase: 05-persistence
    provides: the main/.prev fallback chain with LoadBackup/LoadUnusable, preservation and the load metric from 05-04..05-06
  - phase: 05-persistence
    provides: the (version, content) tie-break, RecordsEqual/MergeChanged and the content-aware applyRemote rule from 05-03
  - phase: 05-persistence
    provides: ADR 0012 recording the equal-version decision, and the role split that moved the engine seams into facade/loader
affects: [deviation-registry, docs, phase-9, future-review]
actuals:
  tokens: 3527
  tasks: 2
  commits: 1
plan_head_before: f4d63b82c1e9042b1257599643bb5daeaf34f56b
plan_head_after: 1f79eef40f6b0519781e163bb7a02d37a750038e
tech-stack:
  added: []
  patterns:
    - "Registry amendments cite a current symbol or test anchor, never a bare line span that the next refactor silently invalidates"
    - "A proof-only task runs its gate on the tree the authoring task committed and makes no no-op commit"
key-files:
  created: []
  modified:
    - docs/SPEC-DEVIATIONS.md
key-decisions:
  - "S-PAR-05 and S-PERS-09 flipped from fixed-in-v1.1 to implemented; S-PERS-08 keeps implemented but its summary drops the now-false 'silent in the log' clause and names port.LoadBackup plus the one-WARN policy"
  - "S-HLC-07's summary is untouched — the zero filler is still the shipped deviation — and the row only gains the docs/adr/0012-equal-version-tiebreak.md link, so the accepted deviation and the tie-break decision stay visibly separate"
  - "The preamble keeps the exact 14.1–14.11 mapping beside ADR 0001–0011 and names ADR 0012 separately, because 0012 records P-07 rather than an owner decision 14.x"
  - "The engine link in S-ALG-REP-03 moved from the pre-split engine.go to internal/app/persist/facade.go, where ApplyRemote now dirties only on a changed merge"
  - "Task 2 made no commit: its diff-scope proof passed on the tree task 1 committed, following the 01-04/05-09 precedent that a no-op commit fabricates history"
patterns-established:
  - "A deviation row's status flip is gated by a status grep, and its evidence by the path/test anchors the row actually names"
  - "The changed-row set is extracted from the diff against the phase-start base with a multi-segment id pattern, so a collateral edit cannot hide behind S-ALG-* ids"
requirements-completed: [PERSIST-01, PERSIST-02, PERSIST-07]
coverage:
  - id: D1
    description: "S-PAR-05 is implemented and states the one enforced limit source: storage.ValueMaxBytes as default, Config.ValueLimit as the effective limit, Store.ValueMaxBytes as the enforcement point, the configured-limit decoder, and the P-01 engine test"
    requirement: "PERSIST-01"
    verification:
      - kind: process
        ref: "grep -q '| S-PAR-05 |.*| implemented |' docs/SPEC-DEVIATIONS.md"
        status: pass
      - kind: process
        ref: "grep -q '| S-DM-05 |.*ValueMaxBytes.*Config.ValueLimit' docs/SPEC-DEVIATIONS.md; every cited path exists on disk"
        status: pass
    human_judgment: false
  - id: D2
    description: "S-PERS-09 is implemented with the Unusable outcome, candidate preservation, the one-ERROR log and the supkv_dump_load_total sample; S-PERS-08 stays implemented and gains the port.LoadBackup outcome path, the Backup WARN and the stand test"
    requirement: "PERSIST-02"
    verification:
      - kind: process
        ref: "grep -q '| S-PERS-09 |.*| implemented |' docs/SPEC-DEVIATIONS.md && grep -q '| S-PERS-08 |.*LoadBackup' docs/SPEC-DEVIATIONS.md"
        status: pass
      - kind: process
        ref: "grep -q 'func TestStand_persistLoadBackup' internal/stand/persist_test.go; internal/persistfile/dumpfile.go:127-175,206-233; cmd/sup-kv/main.go:613-644,651-664 exist"
        status: pass
    human_judgment: false
  - id: D3
    description: "S-ALG-REP-03's summary states the content-aware rule (identical content Acks with no dump; differing content resolved by the deterministic tie-break, stored win replies with the stored record and no Ack) and the row links ADR 0012, applyRemote, facade.ApplyRemote and the core test"
    requirement: "PERSIST-07"
    verification:
      - kind: process
        ref: "grep -q '| S-ALG-REP-03 |.*deterministic (version, content) tie-break.*docs/adr/0012-equal-version-tiebreak.md' docs/SPEC-DEVIATIONS.md"
        status: pass
      - kind: process
        ref: "grep -q 'func TestReplication_equalVersionContentLaw' internal/app/repl/retransmit_test.go"
        status: pass
    human_judgment: false
  - id: D4
    description: "S-HLC-07 keeps accepted-deviation(02-RESEARCH A2) with its zero-filler statement unchanged and gains the ADR 0012 link"
    requirement: "PERSIST-07"
    verification:
      - kind: process
        ref: "grep -q '| S-HLC-07 |.*| accepted-deviation' docs/SPEC-DEVIATIONS.md && grep -q 'docs/adr/0012-equal-version-tiebreak.md' docs/SPEC-DEVIATIONS.md"
        status: pass
    human_judgment: false
  - id: D5
    description: "S-DM-04, S-DM-05 and S-DM-08 keep implemented and now cite ValueMaxBytes/Config.ValueLimit/Store.ValueMaxBytes, MergeChanged/join/RecordsEqual and the merge-law, tie-break and small-space property tests"
    requirement: "PERSIST-07"
    verification:
      - kind: process
        ref: "grep -q '| S-DM-04 |.*MergeChanged.*TestMerge_equalVersionContentLaw' docs/SPEC-DEVIATIONS.md; grep -q '| S-DM-08 |.*RecordsEqual.*TestProp_mergeSmallSpace' docs/SPEC-DEVIATIONS.md"
        status: pass
    human_judgment: false
  - id: D6
    description: "The preamble names ADR 0012 and the P-07 tie-break while keeping the exact 14.1–14.11 mapping beside ADR 0001–0011"
    requirement: "PERSIST-07"
    verification:
      - kind: process
        ref: "sed -n '22,29p' docs/SPEC-DEVIATIONS.md | tr '\\n' ' ' | grep -q 'ADR numbers 0001–0011.*14.1–14.11.*ADR 0012'"
        status: pass
    human_judgment: false
  - id: D7
    description: "The registry diff against the phase-start base changes exactly the eight rows plus the preamble, all eight rows remain present, and the table shape is unchanged"
    requirement: "PERSIST-07"
    verification:
      - kind: process
        ref: "base=$(git log --diff-filter=A --format=%H -1 -- .planning/phases/05-persistence/05-CONTEXT.md); git diff -U0 \"$base\" -- docs/SPEC-DEVIATIONS.md | grep -E '^[+-]\\| S-' | grep -oE 'S-[A-Z]+(-[A-Z]+)*-[0-9]+' | sort -u → exactly S-PAR-05, S-PERS-08, S-PERS-09, S-ALG-REP-03, S-HLC-07, S-DM-04, S-DM-05, S-DM-08"
        status: pass
      - kind: process
        ref: "grep -q '| ID | Summary (own words) | Status | Links |' docs/SPEC-DEVIATIONS.md; for id in S-PAR-05 S-PERS-08 S-PERS-09 S-ALG-REP-03 S-HLC-07 S-DM-04 S-DM-05 S-DM-08; do grep -q \"| $id |\" docs/SPEC-DEVIATIONS.md || exit 1; done"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestOperationsDoc$'"
        status: pass
    human_judgment: false
duration: 2min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 11: Deviation Registry Closure Summary

**The deviation registry now settles this phase's eight rows with shipped evidence: `S-PAR-05` and `S-PERS-09` become implemented, the Backup/Unusable load outcomes are cited by name, `S-ALG-REP-03` states the content-aware equal-version rule against ADR 0012, `S-HLC-07` keeps its accepted deviation with the ADR link, the three data-model rows point at the one limit and the merge helpers, and the diff proves no other decision moved.**

## Performance

- **Duration:** 2min
- **Started:** 2026-11-22T16:47:33Z
- **Completed:** 2026-11-22T16:49:47Z
- **Tasks:** 2
- **Files modified:** 1 (`docs/SPEC-DEVIATIONS.md`)

## Accomplishments

- `S-PAR-05` moved from `fixed-in-v1.1(PERSIST-01)` to `implemented`. Its summary keeps the v1.0 history (the hard-wired 61440 discarded confirmed writes) and states the v1.1 single source: `storage.ValueMaxBytes` is the default, `Config.ValueLimit` the effective limit, `Store.ValueMaxBytes` the enforcement point, with the P-01 engine test linked.
- `S-PERS-09` moved from `fixed-in-v1.1(OBSV-06)` to `implemented`: an all-generations-failed load now logs one ERROR with one attribute group per failed candidate and publishes the `unusable` sample on `supkv_dump_load_total`.
- `S-PERS-08` kept `implemented` and lost its now-false "silent in the log" clause: the row names `port.LoadBackup`, the one-WARN Backup line (path, cause, preservation error), `Load`'s main-then-`.prev` chain, `preserve`, and `TestStand_persistLoadBackup`.
- `S-ALG-REP-03` now states the content-aware rule: an equal version is Acked with no dump only when the content is identical; differing content reaches the store's deterministic `(version, content)` tie-break, where an incoming win Acks and a stored win is answered with the stored record and no Ack. It links ADR 0012, `applyRemote`, `facade.ApplyRemote` and `TestReplication_equalVersionContentLaw`.
- `S-HLC-07` kept `accepted-deviation(02-RESEARCH A2)` with its zero-filler statement byte-identical and gained the `docs/adr/0012-equal-version-tiebreak.md` link.
- `S-DM-04`, `S-DM-05` and `S-DM-08` kept `implemented` and now cite `ValueMaxBytes`/`Config.ValueLimit`/`Store.ValueMaxBytes`, `MergeChanged`/`join`/`RecordsEqual`, `TestMerge_laws`, `TestMerge_equalVersionContentLaw` and `TestProp_mergeSmallSpace`.
- The preamble now reads "ADR numbers 0001–0011 … record owner decisions 14.1–14.11, and ADR 0012 … records the P-07 equal-version tie-break", keeping the 14.x mapping intact instead of folding a non-14.x decision into it.
- The diff-scope gate confirmed the changed-row set is exactly the eight — no collateral registry edit — and all eight rows remain present in the unchanged `| ID | Summary (own words) | Status | Links |` shape.

## Task Commits

1. **Task 1: Amend the eight rows and the preamble** — `1f79eef` (docs)

**Task 2 made no commit:** the proof-only task ran its diff-scope, presence and preamble gates on the tree task 1 committed and found nothing to fix, so no working-tree change existed to commit (the 01-04 and 05-09 precedent: a no-op commit fabricates history). Its evidence is the gate output recorded in D7.

**Plan metadata:** the `docs(05-11)` completion commit (this commit).

## Files Created/Modified

- `docs/SPEC-DEVIATIONS.md` (modified) — the preamble's ADR sentence, the eight amended rows (`S-PAR-05`, `S-PERS-08`, `S-PERS-09`, `S-ALG-REP-03`, `S-HLC-07`, `S-DM-04`, `S-DM-05`, `S-DM-08`); no other row or section touched.

## Decisions Made

- **Status flips and evidence live in the same row.** `S-PAR-05` and `S-PERS-09` carry the shipped symbol or test that proves the statement now holds, so the status cannot outrun the code; neither row says TBD, future or unchecked.
- **`S-PERS-08`'s summary was corrected, its status was not.** The old text asserted the fallback was silent, which 05-05/05-06 made false; the row stays `implemented` and now names the `LoadBackup` outcome and the one-WARN policy.
- **`S-HLC-07`'s statement is untouched.** The zero filler is still the shipped deviation; the ADR 0012 link records that the tie-break decision was made *without* reintroducing the Spec's random tail, so the deviation and the decision stay visibly distinct.
- **The preamble keeps 14.1–14.11 mapping exact.** ADR 0012 records P-07, not an owner decision 14.x, so the sentence names it separately rather than widening the "0001–0011 … 14.1–14.11" pairing into a false equivalence.
- **Pre-split engine references were re-homed.** `S-ALG-REP-03`'s `internal/app/persist/engine.go:507-511` became `internal/app/persist/facade.go:15-49` after 05-09's role split, keeping the link on the code that actually decides the dirty mark.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Line-number citations were corrected against the post-05-09 tree while authoring (e.g. `preserve` ends at dumpfile.go:233, `MergeChanged` at store.go:328, `join` at store.go:415); every cited path and test anchor was grep-verified before the commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The registry answers "is Spec statement X implemented?" for this phase's decisions with current code and test anchors; no row outside the eight changed.
- `requirements.mark-complete` was not run: `requirements.ready-ids` reports PERSIST-01/PERSIST-02/PERSIST-07 still blocked by a remaining owning plan (05-12), so the checkboxes stay for that plan's completion.
- One plan remains in the phase (05-12). Ready for 05-12.

## Self-Check: PASSED

- Files verified on disk: `docs/SPEC-DEVIATIONS.md`, `.planning/phases/05-persistence/05-11-SUMMARY.md`.
- Task commit verified as an ancestor of HEAD: `1f79eef`.
