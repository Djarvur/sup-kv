---
phase: 04-protocol
plan: 05
subsystem: replication-protocol
tags: [proto-03, cr-02, egress-identity, stale-answer, three-way-outcome, acked-false, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 03
    provides: "the DataSync per-entry compare/absorption handler whose egress producers this plan reduces to identities"
  - phase: 04-protocol
    plan: 01
    provides: "the completeness/listing path whose local-newer branch now enqueues no lookup"
  - phase: 03-transport
    provides: "the bounded deduplicated egress queue, its shared pacer, the pending ledger and the real-adapter waiter tracker"
provides:
  - "an identity-only egress queue: queuedUnicast is just egressKey, and the drain builds DataRequest/DataReplication from current store state"
  - "handleDataRequest's O(1) Version presence probe and handleDataSync's lookup-free local-newer enqueue (no Data copy before dedupe)"
  - "the strict three-way applyRemote outcome: strictly older answered with the held record and not Acked, equal Acked with no reply, newer/absent applied and Acked"
  - "the real-adapter proof that a sender waiting for an answered version reports acked=false"
affects:
  - "04-06/04-07 and later protocol plans reuse the same queue identity and dedupe; the stale producer is covered by CR-02's retention bound"
  - "Phase 5's wait_for_replication report consumes the acked=false behavior proven here"

actuals:
  tokens: 10710
  tasks: 2
  commits: 4
  plan_head_before: 1155ba4a9490e3dce1d4a6dfc54845c379e53446
  plan_head_after: 80492d90ab9e7af349c16ba9724cc5bc493e260a

tech-stack:
  added: []
  patterns:
    - "Identity-only queue entries: a producer passes (handle, kind, key) and the drain builds the message from current store state, so retention is bounded by key size and every answer carries the latest record."
    - "A three-way version probe before the merge: strictly older answers through the paced egress queue, equal Acks, newer/absent applies — the probe's position is the contract because the merge cannot change the verdict mid-decision."
    - "An equal-version short-circuit may skip the merge safely because the clock already absorbed that version at install (local write, winning merge, or the dump watermark), keeping retransmission ladders drainable without a redundant clock update."

key-files:
  created:
    - .planning/phases/04-protocol/red-evidence/04-05-task1.json
    - .planning/phases/04-protocol/red-evidence/04-05-task2.json
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/egress_internal_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/merge_stats_test.go
    - internal/grpcapi/wait_integration_test.go

key-decisions:
  - "The stale branch runs before ApplyRemote on one point Version probe: strictly older enqueues one identity-only DataReplication reply and returns without Ack, merge or ledger change; equal Acks and returns without a reply; newer/absent falls through to the existing apply + supersession + Ack path."
  - "Skipping ApplyRemote on an equal version is safe and intentional: the exact version was already absorbed by the clock when the record was installed (write, winning merge or dump watermark), so the merge would only repeat bookkeeping; equal must still Ack or the retransmission ladder never drains."
  - "handleDataRequest uses Version instead of Lookup, so a request for a present key copies no Data; handleDataSync's local-newer branch drops its second Lookup and lets the drain own the probe/read race — both are required to satisfy CR-02's no-pre-dedupe-lookup rule."
  - "The stale reply reuses the (handle, kind, key) egress identity path, so T-4-17's amplification bound and CR-02's retention bound cover the new producer with no extra queue machinery."
  - "TestEgress_noValueRetention measures TotalAlloc rather than HeapAlloc: CR-02 has a retention half and a pre-dedupe allocation half, and only the cumulative counter observes both; the test is deliberately non-parallel with a written nolint reason."
  - "merge_stats_test.go's post-absorption drift-warning row now triggers through a strictly-newer apply under an absent key, because an equal redelivery no longer calls Merge; the assertion's intent (clock.Update measures the pre-update divergence) is unchanged."

patterns-established:
  - "One identity, one drain-time constructor: no reactive unicast producer carries a message, and every queued answer re-reads the store at emission."
  - "Three outcomes at one seam: the replication apply path names stale/equal/newer explicitly instead of letting LWW merge silently decide what an Ack means."

requirements-completed: [PROTO-03]

coverage:
  - id: D1
    description: "Identity-only egress with drain-time construction: queuedUnicast is egressKey alone; DataRequest uses a Version presence probe; DataSync's local-newer branch enqueues without a second Lookup; drain builds both message kinds and skips vanished keys without consuming a pacer slot."
    requirement: XPORT-07
    verification:
      - kind: unit
        ref: "internal/app/repl/egress_internal_test.go#TestEgress_identityOnly"
        status: pass
      - kind: unit
        ref: "internal/app/repl/egress_internal_test.go#TestEgress_noValueRetention"
        status: pass
      - kind: unit
        ref: "internal/app/repl/egress_internal_test.go#TestEgress_boundedDeduped"
        status: pass
      - kind: unit
        ref: "internal/app/repl/egress_internal_test.go#TestEgress_capDropsOldest"
        status: pass
      - kind: unit
        ref: "internal/app/repl/egress_internal_test.go#TestEgress_drainSkipsGoneRecords"
        status: pass
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestDataRequest_replyAndSilence"
        status: pass
    human_judgment: false
  - id: D2
    description: "The strict three-way replication outcome: strictly older produces one unicast DataReplication carrying the held record and no Ack (store unmerged, tracking untouched); equal produces one Ack and no reply; newer and absent apply through ApplyRemote, count merge stats and Ack."
    requirement: PROTO-03
    verification:
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestReplication_staleAnsweredNotAcked"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestReplication_equalVersionAcked"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestReplication_newerApplied"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestReplication_absentApplies"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestRetransmit_ackDrainSemantics"
        status: pass
    human_judgment: false
  - id: D3
    description: "The sender-visible half through the real adapter pair: A writes while B holds a newer version, B answers and sends no Ack, and A's wait_for_replication reports acked=false while B keeps its newer record."
    requirement: PROTO-03
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_integration_test.go#TestWaitIntegration_staleRecordNotAcked"
        status: pass
      - kind: integration
        ref: "internal/grpcapi/wait_integration_test.go#TestWaitIntegration_ackedThroughRealAdapterTracker"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 05: Identity-Only Egress and the Stale-Record Answer Summary

**The egress queue now retains only (handle, kind, key) and builds answers at drain time, and a strictly older replicated record is answered with the newer held record instead of being falsely Acked.**

## Performance

- **Duration:** 13min (791s)
- **Started:** 2026-11-22T04:18:06Z
- **Completed:** 2026-11-22T04:31:17Z
- **Tasks:** 2
- **Files modified:** 5 code/test files (+2 RED evidence records)

## Accomplishments

- CR-02 is structurally fixed: `queuedUnicast` carries only its `egressKey`, `enqueueResponse` takes `(now, handle, kind, key)`, and `resolveQueuedResponse` builds the `DataRequest`/`DataReplication` from current store state. No producer looks a record up before the dedupe check, so the retention bound is key-sized instead of `value_max_bytes × 32 768`.
- `handleDataRequest` probes `Version` (presence only, no Data copy) and `handleDataSync`'s local-newer branch enqueues the identity directly; the drain owns the probe/read race and still skips a vanished key without consuming a pacer slot.
- PROTO-03's strict-three-way rule is implemented at `applyRemote`: strictly older is answered with exactly one unicast reply carrying the stored newer record and no Ack (no merge, no conflict count, no ledger change); equal is Acked with no reply, keeping retransmission ladders drainable; newer/absent applies and Acks exactly as before.
- The real-adapter harness proves the sender-visible consequence: `TestWaitIntegration_staleRecordNotAcked` seeds B ~10 s ahead of A's clock, and A's `wait_for_replication` reports `acked=false` while B's store still holds the newer record.
- Retention is pinned by measurement: 256 present-key DataRequests over 60 KiB values allocate under 1 MiB cumulatively, versus ~16.9 MB measured on the pre-fix queue.

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1 RED: failing retention test for identity-only egress** - `30d5240` (test)
2. **Task 1 GREEN: identity-only queue with drain-time construction** - `e42a20a` (feat)
3. **Task 2 RED: failing replication-outcome tests for the stale answer** - `96e7c69` (test)
4. **Task 2 GREEN: answer stale records instead of acknowledging them** - `80492d9` (feat)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `internal/app/repl/core.go` — identity-only `queuedUnicast`/`enqueueResponse`, `Version` presence probe, drain-time `resolveQueuedResponse`, and the strict three-way `applyRemote` with its rewritten contract comment.
- `internal/app/repl/egress_internal_test.go` — `egressStore.Lookup` clones like the production store; `TestEgress_identityOnly` and `TestEgress_noValueRetention` added; queue-construction call sites moved to the identity API.
- `internal/app/repl/retransmit_test.go` — `seedVersion` plus the stale/equal/newer/absent outcome rows.
- `internal/app/repl/merge_stats_test.go` — the post-absorption drift row now uses a strictly-newer apply under an absent key.
- `internal/grpcapi/wait_integration_test.go` — `TestWaitIntegration_staleRecordNotAcked` over the real adapter pair.
- `.planning/phases/04-protocol/red-evidence/04-05-task{1,2}.json` — persisted RED evidence, both `RED_EVIDENCE_OK`.

## Decisions Made

- **Strictly older is answered, never merged.** The branch runs before `ApplyRemote` on one point `Version` probe, so the merge cannot change the verdict mid-decision; no conflict or drift is counted because no merge happened, and the tracking ledger is untouched.
- **Equal is Acked without merging.** The exact version was already absorbed by the clock when the record was installed (local write, winning merge, or the dump watermark), so an equal merge would only repeat bookkeeping. Acking remains mandatory: treating equality as stale would leave retransmission ladders undrained (R-02).
- **The stale reply rides the existing egress identity path.** It is deduped by `(handle, kind, key)` and paced by the shared limiter, so a repeated stale flood collapses to one queued answer per key (T-4-17) and the CR-02 retention bound covers the new producer.
- **The retention probe uses `TotalAlloc`.** It observes both CR-02 halves — queued value retention and pre-dedupe allocation amplification — and the test is intentionally non-parallel with an explicit nolint justification, per AGENTS.md's suppression rule.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `TestMergeStats_counts`'s drift-warning row relied on the removed equal-version merge**
- **Found during:** Task 2 GREEN (first run of the full repl package)
- **Issue:** The old row drove its post-absorption drift warning through an equal-version redelivery. With the equal branch short-circuiting before `ApplyRemote` (the plan's explicit action), no `clock.Update` runs and `DriftWarnings` stayed 0, failing the row.
- **Fix:** Trigger the post-absorption warning with a strictly-newer record under the still-absent `keyGamma`: the first far-ahead apply absorbs the stamp with zero drift, and the next apply observes the clock one warning window ahead and counts one warning. Conflicts stay at 1 and the assertion's intent is unchanged.
- **Files modified:** `internal/app/repl/merge_stats_test.go`
- **Verification:** `go test -race -count=1 ./internal/app/repl/ -run '^TestMergeStats_counts$'` green; full repl package green.
- **Committed in:** `80492d9`

**2. [Rule 3 - Blocking] Lint adaptations in the new tests**
- **Found during:** Task 2 lint gate
- **Issue:** The strict config's `noinlineerr` rule rejected five new inline error checks, and `unparam` flagged `assertAck`'s `key` parameter once every call site passed the same constant.
- **Fix:** Split the assignments before checking the error, and moved the absent-key row to `keyGamma` so the helper's `key` parameter keeps two legitimate values instead of suppressing `unparam`.
- **Files modified:** `internal/app/repl/retransmit_test.go`, `internal/grpcapi/wait_integration_test.go`
- **Verification:** `make lint` green across the module.
- **Committed in:** `80492d9`

---

**Total deviations:** 2 auto-fixed (both Rule 3)
**Impact on plan:** Both are test adaptations forced by the planned production semantics; no assertion, bound or behavior was relaxed. The plan's production surface matches its action list exactly.

## TDD Gate Compliance

Both tasks followed RED → GREEN with persisted evidence; `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for each record:

- **Task 1:** RED `30d5240` — `04-05-task1.json` records `TestEgress_noValueRetention` failing on its planned bound: 256 present-key DataRequests over 60 KiB values allocated 16 880 056 bytes against a 1 MiB ceiling, because the queue retained a cloned value copy per answer. GREEN `e42a20a` made the same probe allocate only queue identities; `TestEgress_identityOnly` was added with the identity API it exercises.
- **Task 2:** RED `96e7c69` — `04-05-task2.json` records `TestReplication_staleAnsweredNotAcked` failing on its planned assertion: the stale datagram emitted one message directly (the false Ack) instead of queueing a reply. The sibling `TestWaitIntegration_staleRecordNotAcked` independently failed pre-fix with `acked=true 1/1`, confirming the same defect through the real adapter pair. GREEN `80492d9` made all four outcome rows and both integration expectations pass.
- **REFACTOR:** none separately — the lint and `merge_stats_test.go` adaptations are folded into `80492d9`, and no production simplification was warranted.

## Issues Encountered

None beyond the two documented test adaptations. The full `./internal/app/repl/`, `./internal/grpcapi/`, `./internal/udprepl/` and app dependency suites are green under `-race`, and `make lint` passes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CR-02's queue and PROTO-03's stale outcome are complete and covered; 04-06 can build the next protocol outcomes on the same identity-only queue and three-way seam.
- 8 plans remain in Phase 4. Ready for `04-06-PLAN.md`.

## Self-Check: PASSED

- SUMMARY.md exists at `.planning/phases/04-protocol/04-05-SUMMARY.md`.
- RED evidence records exist for both tasks.
- All four task commits (`30d5240`, `e42a20a`, `96e7c69`, `80492d9`) are ancestors of HEAD.
