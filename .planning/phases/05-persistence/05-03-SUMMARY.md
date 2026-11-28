---
phase: 05-persistence
plan: 03
subsystem: replication-and-storage
tags: [equal-version-tiebreak, deterministic-merge, remove-wins, bytes-compare, merge-changed, wire-decision, adr-0012, same-seed-replay]
requires:
  - phase: 05-persistence
    provides: the single-source value limit and the four-class dump decode from 05-01/05-02
  - phase: 04-protocol
    provides: the StateStore seam, the identity-only egress queue and the PROTO-03 stale-record reply path
  - phase: 03-transport
    provides: the retransmission ladders and the equal-version Ack that keeps them draining
affects: [persistence, storage-merge, replication-core, persist-engine, p07, p12, spec-deviations]
actuals:
  tokens: 14074
  tasks: 3
  commits: 4
tech-stack:
  added: []
  patterns:
    - "Deterministic content tie-break: (version, content) order with remove-wins first, then bytes.Compare over Data — no randomness, no map order"
    - "One comparison, two consumers: storage.RecordsEqual is the single content test shared by the merge law and the replication wire rule"
    - "Changed-result authority: MergeChanged reports whether the incoming record became the stored record; the wrapper Merge delegates"
    - "Wire answer from the stored record after the merge, never from the version bytes (which are equal on both sides)"
key-files:
  created:
    - docs/adr/0012-equal-version-tiebreak.md
  modified:
    - internal/app/storage/store.go
    - internal/app/storage/store_test.go
    - internal/app/storage/merge_test.go
    - internal/app/storage/prop_small_space_test.go
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
key-decisions:
  - "The join order is (version, content): equal versions with identical Deleted and Data are a full no-op; differing content resolves remove-wins first, then the lexicographically greater Data by bytes.Compare"
  - "A tie-break win or loss reports exactly one Conflict whose Winner and Loser are the same 16 version bytes — the stored record is the only observable evidence"
  - "storage.RecordsEqual (version, deleted, Data; checksum excluded as derived) is the one comparison the merge law and the core's equal-version branch share"
  - "MergeChanged returns (conflicts, changed, drift, err); Merge stays the three-result wrapper so no caller signature changes"
  - "repl.Core.applyRemote Acks identical content early, applies differing content through the store, then Acks on an incoming win or queues exactly one unicast DataReplication of the stored record with no Ack"
  - "Engine.ApplyRemote dirties on MergeChanged's changed result; the old Get probe and Winner == rec.Version heuristic are removed (research Pitfall 5)"
  - "ADR 0012 records a deviation from the Spec's random version tail: zero filler (S-HLC-07) plus a deterministic content law, with PFXD-01 and the version layout as explicit non-changes"
patterns-established:
  - "Content law tested from an independent test-side expression (contentWinner) rather than from the production join"
  - "Wire determinism pinned by replaying one fixed equal-version sequence through two identically seeded cores and comparing the decision traces"
  - "Same-seed trace asserts the expected decision sequence too, so equality cannot pass vacuously"
requirements-completed: [PERSIST-07]
coverage:
  - id: D1
    description: "The store joins equal versions by the deterministic (version, content) law and reports installation through MergeChanged; Merge delegates"
    requirement: "PERSIST-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestMerge_equalVersionBranchTable$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestMerge_(equalVersionContentLaw|losersProperty|laws|lawsTombstones)$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestProp_mergeSmallSpace$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Engine.ApplyRemote dirties only on a real content change: a tie-break win arms one dump, an identical redelivery arms none"
    requirement: "PERSIST-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_applyRemoteEqualVersionContent$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_(redundantDirty|applyRemoteTriggersDump)$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The wire rule reaches production: identical content Acks, an incoming tie-break win Acks with no reply, a stored tie-break win answers exactly one unicast DataReplication and no Ack, and same-fixture replay decisions are identical"
    requirement: "PERSIST-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestReplication_equalVersion(ContentLaw|Acked|SameSeedTrace)$'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestReplication_(staleAnsweredNotAcked|newerApplied)$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "ADR 0012 records the decision, context, consequences and links in the slim-MADR shape"
    requirement: "PERSIST-07"
    verification:
      - kind: command
        ref: "test -f docs/adr/0012-equal-version-tiebreak.md && grep -q '^## Context' && grep -q '^## Decision' && grep -q '^## Consequences' && grep -q '^## Links'"
        status: pass
      - kind: command
        ref: "grep -q 'informs: \\[PERSIST-07\\]' docs/adr/0012-equal-version-tiebreak.md"
        status: pass
    human_judgment: true
    rationale: "The presence and frontmatter gates are automated, but whether the Context, Decision and Consequences fully and honestly record the deviation is a reading judgment for the verifier."
  - id: D5
    description: "The old no-op assertions are replaced, not weakened, and the identical-content idempotence path is untouched"
    requirement: "PERSIST-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -v -run '^TestMerge_duplicateDelivery$'"
        status: pass
      - kind: command
        ref: "diff of TestMerge_duplicateDelivery against the phase base 4f81fd7 is empty"
        status: pass
    human_judgment: false
duration: 20min
completed: 2026-11-22
status: complete
plan_head_before: c964cceeed99a0f07e6d1602a1154da0fa5ee41b
plan_head_after: 06691b8484f010df7dfb46048a0f23936c074e36
---

# Phase 05 Plan 03: Deterministic equal-version tie-break Summary

**Equal versions with different content now resolve by a pure function of the records — remove-wins, then greater `Data` — at the store merge, in the engine's dump decision and on the wire, with the deviation recorded in ADR 0012.**

## Performance

- **Duration:** 20min (1224s)
- **Started:** 2026-11-22T13:24:53Z
- **Completed:** 2026-11-22T13:45:17Z
- **Tasks:** 3
- **Files modified:** 9 (8 modified, 1 created)

## Accomplishments

- `storage.RecordsEqual` is the one content comparison; `storage.Store.MergeChanged` returns `(conflicts, changed, drift, err)` and `Merge` delegates, so no existing caller signature changed.
- `join` now resolves equal versions: identical content is a full no-op, an incoming or stored tombstone wins remove-wins, and otherwise the greater `Data` by `bytes.Compare` wins. Every tie-break direction reports exactly one `Conflict` pairing the equal version bytes.
- `persist.Engine.ApplyRemote` switches from the `Get` probe plus `Winner == rec.Version` heuristic to `MergeChanged`'s `changed`: a tie-break win arms exactly one dump, an identical redelivery arms none, and the stored-tombstone blind spot is gone.
- `repl.Core.applyRemote` stops short-circuiting every equal version: identical content keeps the old Ack-and-no-reply path, differing content falls through to the store, and the answer follows the stored record — incoming win Acks, stored win queues exactly one unicast `DataReplication` of the stored record and sends no Ack.
- The exhaustive branch table pins both directions of every branch (live/live, tombstone/live both ways, tombstone/tombstone with different data, identical live and tombstone content), including `changed` and the `Merge`/`MergeChanged` agreement.
- The small-space property suite gains a collision generator whose payload and tombstone flag are independent of the version, so commutativity, associativity and idempotence are proven where equal versions carry different content.
- `TestReplication_equalVersionSameSeedTrace` replays one fixed ordered equal-version sequence through two identically seeded cores and compares both the Ack/reply decision traces and the resulting store state, with the expected sequence asserted so equality cannot pass vacuously.
- ADR 0012 records the deviation, its lost-operation consequence and its explicit non-changes (version layout, zero filler, PFXD-01).
- `make lint` and `make test` are green; `go build ./...`/`go vet ./...` green; `internal/app/version/` is byte-identical to the phase base 4f81fd7; both red-runner mutant anchors (`storage.Versions`, `nextSync` gate) are intact.

## Task Commits

1. **Task 1 RED: failing equal-version content-law tests** - `c56caf5` (test)
2. **Task 1 GREEN: store law, engine changed decision, wire branch** - `90bf1e4` (feat)
3. **Task 2: exhaustive branch table and collision property** - `c3d54c4` (test)
4. **Task 3: same-seed wire trace and ADR 0012** - `06691b8` (test)

**Plan metadata:** the `docs(05-03)` completion commit (this commit).

## Files Created/Modified

- `internal/app/storage/store.go` - `RecordsEqual`, `MergeChanged`, content-aware `join`, `Merge` delegating wrapper, contract comments updated.
- `internal/app/storage/store_test.go` - content-law merge subtests, split tap subtests (content win fires, identical redelivery silent), generation expectations moved to the law.
- `internal/app/storage/merge_test.go` - failing content-law test, exhaustive branch table, `Merge`/`MergeChanged` agreement, `contentWinner`/`equalVersionConflicts`/`assertStoredRecord` helpers, rewritten `TestMerge_losersProperty`.
- `internal/app/storage/prop_small_space_test.go` - collision subtest rewritten to the law, `randCollisionStore`, new three-laws property over independently drawn payloads.
- `internal/app/repl/core.go` - content-aware equal-version branch in `applyRemote`, post-merge Ack-versus-reply decision, rewritten handler contract.
- `internal/app/repl/retransmit_test.go` - failing wire-law test, same-seed decision trace.
- `internal/app/persist/engine.go` - `ApplyRemote` uses `MergeChanged`'s `changed`; contract comment rewritten.
- `internal/app/persist/engine_test.go` - failing dirty-decision test (tie-break win dumps, identical redelivery does not).
- `docs/adr/0012-equal-version-tiebreak.md` - the recorded decision, context, consequences and links.

## Decisions Made

- The order is `(version, content)`; remove-wins precedes `bytes.Compare` over `Data`. This preserves deterministic replay and byte-identical dumps, which a random tail would break.
- `RecordsEqual` is exported from `storage` so the core and the store share exactly one content test; the checksum is excluded because it is derived.
- `MergeChanged` carries the installation decision instead of inferring it from `Conflict` fields, which cannot distinguish a tie-break win from a loss (both pair the same version bytes).
- The wire answers from the stored record after the merge rather than from either side's version; this also makes the strictly-newer path self-verifying.
- The identical-content early Ack is preserved unchanged so retransmission ladders still drain (R-02, S-PROTO-16).
- Cross-node convergence is deliberately not claimed here: a merge-only seed emits no replication message, so that proof belongs to plan 05-07's `Cluster.NotifyInstalled` seam.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Branch-table `changed` expectation mislabelled a losing tie-break**
- **Found during:** Task 2 first verify run
- **Issue:** The table derived `changed` from "content differs", which is true for a tie-break loss too; the losing rows expected `changed == true` while the store correctly reported `false`.
- **Fix:** `changed` is now derived from "the incoming record is the content winner and the content was not already identical" — the same definition `MergeChanged` implements.
- **Files modified:** `internal/app/storage/merge_test.go`
- **Verification:** the whole branch table passes, including both losing rows.
- **Commit:** `c3d54c4`

**2. [Rule 1 - Bug] `TestStore_generationContentOnly` encoded the old no-op law**
- **Found during:** Task 2 first verify run
- **Issue:** The pre-existing test asserted that an equal-version merge with `Data` `"x"` over `"a"` leaves the generation at 4; under the content law that merge installs and moves it to 5, so the test failed at the implementation's correct behavior. The plan's rewrite list named the merge/tap/collision assertions but not this generation row.
- **Fix:** The equal-version step now expects the tie-break install (generation 5), an identical redelivery stays at 5, and the following strictly-greater merge moves to 6; the doc comment names the content tie-break.
- **Files modified:** `internal/app/storage/store_test.go`
- **Verification:** `go test -race -count=1 ./internal/app/storage/ -run '^TestStore_'` green.
- **Commit:** `c3d54c4`

### Intentional plan-order adjustments

- Task 2's planned rewrites of the store equal-version subtest, the tap subtest, `TestMerge_losersProperty`'s `cmpSame` expectation and the small-space collision subtest landed with Task 1's GREEN commit (`90bf1e4`): the storage suite could not stay green in one commit with the law implemented and the old no-op assertions still in place (Rule 3, blocking). Task 2 then added the exhaustive branch table, the `Merge`/`MergeChanged` agreement rows and the collision property generator.
- Lint findings in the new tests (`noinlineerr` on two seed merges, one unused `//nolint:gosec`) were fixed before the Task 2 commit; no suppression was added.

## TDD Gate Compliance

- **Task 1 RED:** `TestMerge_equalVersionContentLaw` (plus `TestReplication_equalVersionContentLaw` and `TestEngine_applyRemoteEqualVersionContent`) failed on the planned assertions before the implementation: the equal-version branch returned no conflict and installed nothing, the wire emitted one message where a stored-win reply had to queue, and the engine reported no conflicts. Evidence verdict `RED_EVIDENCE_OK` (`target_test_failed`), record `.planning/phases/05-persistence/red-05-03-task1.json`. GREEN: `90bf1e4`.
- **Task 1 RED extension:** the `changed`-result assertions were added with the GREEN commit because `MergeChanged` does not exist before the implementation; the RED test exercised the same law through the existing `Merge` wrapper.
- **Task 2:** unexpected GREEN by design — the tracer (Task 1) already implemented the law, so the exhaustive table, wrapper-agreement rows and collision property are recorded as characterization pins rather than a fabricated RED. Commit: `c3d54c4`.
- **Task 3:** unexpected GREEN by design — the deterministic law was already in place, so the same-seed decision trace is a characterization pin for the wire rule. Commit: `06691b8`.

## Issues Encountered

- The plan-commit ledger file predated this execution and carried a stale, non-ancestor base (`e2dc8a3`, an unrelated history line from an earlier attempt). It was corrected to the actual pre-plan HEAD `c964cce`, and the measured commit count is 4 task commits.

## Threat Model Disposition

- **T-5-06 (tampering, rogue same-version record):** mitigated — every replica applies the same deterministic remove-wins/content order and the wire returns the stored record when it wins; no randomness exists for a peer to exploit.
- **T-5-07 (repudiation, nondeterministic winner):** mitigated — `bytes.Compare` over explicit fields, no map iteration and no random tail decide a winner; the same-seed trace compares two independent runs.
- **T-5-08 (doS, amplified replies):** mitigated — a stored win emits exactly one unicast reply and no Ack; the core test asserts the count and the existing egress identity/dedupe path bounds retention.
- **T-5-SC (malicious installs):** no dependency was added.

## Known Stubs

None — no placeholder, stub or skipped test was introduced by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 05-04 can build on `MergeChanged` and `RecordsEqual`: the dump decision is now driven by an authoritative changed result rather than a conflict heuristic.
- 05-07 owns the cross-node convergence proof on the `Cluster.NotifyInstalled` notification seam; this plan deliberately claims wire determinism only.
- The SPEC-DEVIATIONS rows (S-HLC-07, S-ALG-REP-03) are unchanged here by plan instruction; the registry flips in a later plan with ADR 0012 now linkable.
- No stubs, skipped tests or unrun verifies were introduced; the broken-windows ledger needs no entries from this plan.

## Self-Check: PASSED

All nine files exist and all four task commits are ancestors of HEAD; the measured `plan_head_before`/`plan_head_after` values were verified against the corrected ledger.
