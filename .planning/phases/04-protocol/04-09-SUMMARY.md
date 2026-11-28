---
phase: 04-protocol
plan: 09
subsystem: replication-protocol
tags: [proto-05, proto-06, r-10, r-12, notification-guard, seen-run-ring, alternation, config-error, shared-warn-limiter, tdd, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 08
    provides: "the first-seen/restart Hello classification and the targeted-listing path the alternation rule discriminates within"
  - phase: 04-protocol
    plan: 05
    provides: "the identity-only egress queue and stale-record outcome that keep in-flight old-run datagrams harmless"
  - phase: 04-protocol
    plan: 01
    provides: "the per-(node_id, class) discard-WARN limiter key and the D-08 OnDiscard seam the duplicate paths now share"
provides:
  - "the not-greater-than-tracked notification guard: regressive and equal duplicate notifications are silent, counted, and create no tracking"
  - "NotificationsRegressed() uint64 with the Phase 7 publication handoff documented"
  - "peer.seenRuns: a fixed 8-entry ring of previously current run_ids, pushed on every accepted run change"
  - "the alternating-identity configuration error in handleHello: DuplicateNodeID counted, one rate-limited configuration-error WARN, dropped with no listing and no row/liveness update"
  - "one shared per-(node_id, class) rate-limited WARN helper behind all three duplicate-node_id paths and the validation classes"
affects:
  - "04-13's full phase gate re-runs the PROTO-05/06 acceptance rows and closes their REQUIREMENTS.md checkboxes"
  - "Phase 5's wait_for_replication consumes the untouched first-notification path: an accepted write still creates tracking before any wire traffic"
  - "Phase 7's OBSV surface publishes NotificationsRegressed; no adapter mirror exists this phase"

actuals:
  tokens: 12700
  tasks: 2
  commits: 4
  plan_head_before: cc13ce3ee1f2282112cd84c059621de851817319
  plan_head_after: 4fd41d5da4edac0523ec58bb0c87df81007376d6

tech-stack:
  added: []
  patterns:
    - "Identity history as a fixed ring: the previous N current identities live in a fixed array with a write position and a valid count, so a hostile flapping peer cannot grow per-peer state; the bound is proven behaviorally on both sides (9 runs keep the first, 10 evict it)."
    - "Guard against the tracked state, not the source of truth: the notification guard reads the pending set, never the store, because the store necessarily already holds the notified version by the time the post-write notification runs."
    - "One limiter, many classes: a message-taking rate-limited WARN primitive lets every counted discard class keep its own operator wording while sharing the per-(node_id, class) slot, so no call path can bypass the bound."

key-files:
  created:
    - .planning/phases/04-protocol/red-evidence/04-09-task1.json
    - .planning/phases/04-protocol/red-evidence/04-09-task2.json
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/repl/restart_test.go
    - internal/app/repl/peertable_test.go
    - internal/app/repl/discard_test.go

key-decisions:
  - "The notification guard compares rec.Version against the tracked pending version for the key, never store.Version: the store already holds exactly the first notified version when NotifyLocalWrite runs, so a <= store comparison would swallow the normal first-notification path. Absent tracking means proceed."
  - "The guard covers equal duplicates as well as regressive writes (not-greater-than), because both would re-broadcast a version the node has already superseded or is already tracking; one counter (NotificationsRegressed) covers both and only Phase 7's OBSV surface publishes it."
  - "The seen-run ring stores only PREVIOUS current run_ids; the current one remains in peer.runID. The cap is 8, so after nine distinct runs the first is still inside (re-announcing it is a configuration error) and the tenth evicts it (the same announcement becomes a restart) — both sides are pinned."
  - "The alternation branch returns before ANY mutation: no listing FIFO, no retained-key drop, no row update and no liveness refresh, so the currently remembered run stays authoritative and a flapping host gains nothing per datagram."
  - "The alternation branch also fires OnDiscard with the duplicate_node_id class, matching the D-08 seam contract every other discard site follows; the plan's counted-plus-WARN requirement is a superset of the existing duplicate evidence shape."
  - "The shared per-(node_id, class) limiter means all three duplicate-node_id paths contend for one slot per node. That is the T-4-28 intent: a host cannot flood the log by cycling among the own-node-id, migrated-handle and alternation paths."
  - "The stale-return subtest of TestRestart_staleRunIdAndInFlight is rewritten as the configuration-error expectation while its in-flight subtest keeps proving that old-run Ack/DataReplication traffic processes on (key, version) merits."
  - "The discard totality model became a mutable mirror (map[uint16]*modelPeer) that applies the Hello identity transition inside classify, because counter conservation now depends on the bounded seen-run history, not only on the current row."

patterns-established:
  - "Bounded identity history with a fixed array: seenRuns plus seenLen/seenPos, no slice growth, oldest-entry determinism."
  - "Rate-limited WARN with caller-supplied wording: warnRateLimited(now, nodeID, class, msg) is the single primitive; warnDiscard delegates with the generic validation wording."
  - "Discrimination by unseen, not by unequal: a changed run_id is a restart only when the row has never carried it."

requirements-completed: [PROTO-05, PROTO-06]

coverage:
  - id: D1
    description: "PROTO-05's notification guard: a first notification of a store-held record broadcasts and tracks; an equal duplicate and a regressive notification are silent, increment NotificationsRegressed and leave the tracked version and entry count untouched; a newer notification supersedes and broadcasts; an untracked key proceeds."
    requirement: PROTO-05
    verification:
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestNotify_firstNotificationProceeds"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestNotify_equalTrackedIgnored"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestNotify_regressedIgnored"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestNotify_newerBroadcasts"
        status: pass
      - kind: unit
        ref: "internal/app/repl/retransmit_test.go#TestNotify_absentKeyBroadcasts"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestRetransmit_|^TestPending_'"
        status: pass
    human_judgment: false
  - id: D2
    description: "PROTO-06's alternating-identity configuration error: a previously seen, non-current run_id is counted as DuplicateNodeID, WARNed as a configuration error, dropped with no listing and no row/liveness change; the current run still refreshes; an unseen run is still a restart; the history is a fixed 8-entry ring."
    requirement: PROTO-06
    verification:
      - kind: unit
        ref: "internal/app/repl/peertable_test.go#TestPeer_alternatingRunIDsIsConfigError"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_staleRunIdAndInFlight"
        status: pass
      - kind: unit
        ref: "internal/app/repl/peertable_test.go#TestPeer_seenRingBounded"
        status: pass
      - kind: unit
        ref: "internal/app/repl/peertable_test.go#TestPeer_repeatSameRunStillRefresh"
        status: pass
      - kind: unit
        ref: "internal/app/repl/restart_test.go#TestRestart_runIDChangeTriggersFullSync"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestRestart_|^TestPeer_|^TestDiscard_'"
        status: pass
    human_judgment: false
  - id: D3
    description: "One shared per-(node_id, class) rate limiter covers the alternation path and both pre-existing duplicate-node_id paths: two peers alternating inside one interval each WARN once, a repeated alternation from one peer is suppressed, and the boundary fires at the interval on event time."
    requirement: PROTO-06
    verification:
      - kind: unit
        ref: "internal/app/repl/peertable_test.go#TestPeer_alternatingWarnRateLimitedPerPeer"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_warnRateLimited"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_warnRateLimitedPerPeer"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_totality"
        status: pass
    human_judgment: false
  - id: D4
    description: "The whole package and the whole module stay green under the race detector and the strict lint config: simulator scenarios, discard totality, the three-node/stand suites and the depguard architecture boundary."
    requirement: PROTO-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/"
        status: pass
      - kind: other
        ref: "make lint (go tool golangci-lint config verify && go tool golangci-lint run ./...)"
        status: pass
      - kind: other
        ref: "make test (go test -race -cover ./...)"
        status: pass
    human_judgment: false

duration: 14min
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 09: Notification Regression Guard and Alternating-Identity Configuration Error Summary

**A local notification that cannot beat the tracked version now dies before the wire and is counted, and a host flapping between run_ids of one node_id is a named, rate-limited configuration error with a fixed 8-entry memory instead of a full listing per datagram.**

## Performance

- **Duration:** 14min (847s)
- **Started:** 2026-11-22T05:39:42Z
- **Completed:** 2026-11-22T05:53:49Z
- **Tasks:** 2
- **Files modified:** 5 code/test files (+2 RED evidence records)

## Accomplishments

- PROTO-05 is enforced at the notification seam. `NotifyLocalWrite` first resolves the version the core already tracks for the key and returns immediately — no broadcast, no ledger or tracking change — when the incoming version is not greater, incrementing `notificationsRegressed` (published by `NotificationsRegressed()`). The first notification of a key has no tracked entry and always proceeds, and the guard deliberately never reads `store.Version`: by the time the post-write notification runs, the store already holds exactly that version, so a `<=` store comparison would swallow the normal path (T-4-29).
- PROTO-06 is a named configuration error. `peer.seenRuns` is a fixed `[8][16]byte` ring with a valid count and write position; every accepted run change pushes the outgoing current run. `handleHello` now discriminates: same run refreshes, a run in the ring is `DuplicateNodeID` — counted, WARNed with `sup-kv: configuration error: node_id %d ...`, dropped with no listing and no row or liveness change — and an unseen run is still a restart with exactly one targeted listing (T-4-27).
- The bound is proven on both sides: after nine distinct runs the first run is still inside the ring (its re-announcement is a configuration error), while the tenth run evicts it and the same re-announcement becomes a legitimate restart. Nine runs × 16 bytes per peer is the entire worst case, so a flapping identity cannot grow per-peer memory (T-4-30).
- All three duplicate-node_id paths — own node_id with a foreign run, a remembered run migrating handles, and alternation — share one per-`(node_id, class)` rate-limited WARN helper on event time, so a hostile host cannot flood the log by cycling paths (T-4-28). The existing validation classes use the same primitive and keep their own wording.
- In-flight old-run traffic is untouched by design: the in-flight subtest of `TestRestart_staleRunIdAndInFlight` still proves an old run's Ack drains and a strictly-greater DataReplication applies and Acks.

## Task Commits

Each TDD task produced its RED and GREEN commits:

1. **Task 1 RED: failing notification-guard tests + compile scaffolding** - `f057c1e` (test)
2. **Task 1 GREEN: ignore notifications that cannot advance the tracked version** - `05bf894` (feat)
3. **Task 2 RED: failing alternating-identity and seen-ring tests** - `8d206b3` (test)
4. **Task 2 GREEN: name the alternating identity and bound its history** - `4fd41d5` (feat)

**Plan metadata:** this SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `internal/app/repl/core.go` — `notificationsRegressed` + `NotificationsRegressed()`, `trackedVersion`, the not-greater guard at the top of `NotifyLocalWrite`, its rewritten contract comment, `seenRunsCap = 8`, `peer.seenRuns/seenLen/seenPos`, `peer.pushSeenRun`/`peer.seen`, the three-way `handleHello` identity discrimination, and `warnRateLimited` with `warnDiscard` delegating; both existing duplicate branches now warn through it.
- `internal/app/repl/retransmit_test.go` — the five `TestNotify_*` rows: first notification, equal duplicate, regressive write, newer supersession with its `Superseded` result, and the untracked-key empty case.
- `internal/app/repl/restart_test.go` — `TestRestart_staleRunIdAndInFlight`'s first subtest rewritten to the alternation expectation (no listing, counted, WARN, row unchanged); the in-flight and double-restart subtests kept.
- `internal/app/repl/peertable_test.go` — `helloRun` and `configErrorWarns` helpers plus `TestPeer_alternatingRunIDsIsConfigError`, `TestPeer_alternatingWarnRateLimitedPerPeer`, `TestPeer_seenRingBounded`, `TestPeer_repeatSameRunStillRefresh`.
- `internal/app/repl/discard_test.go` — the totality model now carries the bounded seen-run history and applies the Hello identity transition, mirroring the core rule.
- `.planning/phases/04-protocol/red-evidence/04-09-task{1,2}.json` — both `RED_EVIDENCE_OK`.

## Decisions Made

- **The guard reads the pending set, never the store.** The store is always ahead at notification time; only the tracked version distinguishes "first notification" from "stale re-notification".
- **Equal duplicates are counted with regressions.** Both are not greater than the tracked version and both would re-broadcast superseded state; one counter keeps the operator signal simple.
- **The ring holds previous runs only.** The current run stays in `runID`; nine distinct runs still leave the first in the 8-entry ring, the tenth evicts it. The ring test pins both outcomes rather than assuming one.
- **Alternation returns before any mutation.** Dropping the old run's listing/keys would be dead work for a dropped datagram and would let a flapper churn retention state.
- **`OnDiscard` fires on the alternation branch.** The D-08 seam is per discard site; the plan's counted-plus-WARN requirement does not replace it.
- **The totality model mirrors state, not just classification.** The new rule depends on history, so `classify` owns the Hello transition through a pointer-valued model row.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestDiscard_totality`'s classifier model diverged from the new alternation rule**
- **Found during:** Task 2 verify (`go test ... -run '^TestRestart_|^TestPeer_|^TestDiscard_'`)
- **Issue:** The model function treated every differing run_id for a known node as a restart, so a random Hello replaying a previously seen run_id was expected to be processed while the core counted `DuplicateNodeID` — 41 actual vs 39 expected, breaking counter conservation.
- **Fix:** `modelPeer` gained the same bounded 8-entry seen-run history; `classify` now applies the core's transition (learn on unknown, refresh on equal, push-and-update on unseen, `DuplicateNodeID` on a seen non-current run) through a `map[uint16]*modelPeer`. No assertion or counter was relaxed — the model was made to mirror the pinned decision.
- **Files modified:** `internal/app/repl/discard_test.go`
- **Verification:** `TestDiscard_totality` green; the full `TestRestart_|TestPeer_|TestDiscard_` set and the whole package green.
- **Committed in:** `4fd41d5`

**2. [Rule 3 - Blocking] Lint adaptations forced by the larger peer row and the new ring loops**
- **Found during:** Task 2 lint gate
- **Issue:** Growing `peer` to 200 bytes made `PeerSnapshot`'s `for id, p := range c.peers` trip `gocritic`'s `rangeValCopy`; the two ring scans tripped `intrange` (Go 1.22 integer ranges).
- **Fix:** `PeerSnapshot` indexes the map (`p := c.peers[id]`) and both ring scans use `for i := range p.seenLen`. No behavior change.
- **Files modified:** `internal/app/repl/core.go`, `internal/app/repl/discard_test.go`
- **Verification:** `go tool golangci-lint run ./internal/app/repl/` and `make lint` green; full package re-run green.
- **Committed in:** `4fd41d5`

---

**Total deviations:** 2 auto-fixed (1 Rule 1, 1 Rule 3)
**Impact on plan:** The production surface matches the plan's action list; the only extra production edit is the lint-shaped `PeerSnapshot` iteration. The extra test file (`discard_test.go`) exists because the totality model is a correctness oracle for the new decision and had to learn the new rule.

## TDD Gate Compliance

Both TDD tasks followed RED → GREEN with persisted, machine-validated evidence; `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for each record:

- **Task 1:** RED `f057c1e` — `04-09-task1.json` records `TestNotify_equalTrackedIgnored` and `TestNotify_regressedIgnored` failing on their planned silence assertion: the pre-guard core emitted one broadcast for each, where the guard promises zero and one counted regression. The other three rows passed as preserved-contract pins (first notification, newer supersession, untracked key). The RED commit carried only the counter field and its copying accessor as compile scaffolding. GREEN `05bf894` made all five rows pass and left `TestRetransmit_`/`TestPending_` green.
- **Task 2:** RED `8d206b3` — `04-09-task2.json` records three failures on planned assertions: the alternation emitted a listing instead of none, the rate-limit row captured zero configuration-error WARNs, and the ring row still listed run 1 after nine runs. `TestPeer_repeatSameRunStillRefresh` passed as the preserved repeat contract. GREEN `4fd41d5` made all four rows and the rewritten stale-return subtest pass, with the full `TestRestart_|TestPeer_|TestDiscard_` set green.
- **REFACTOR:** none separately — the two Rule 3 lint adaptations are folded into `4fd41d5`; no production simplification was warranted.

## Issues Encountered

The discard-totality model divergence (deviation 1) was the only substantive issue; it surfaced immediately in the task-2 verify because the counter-conservation assertion is exact. No flake, no timeout, no environmental failure.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. No placeholder values, TODOs, skipped tests or unwired data sources were introduced. `NotificationsRegressed` is a live counter; its OBSV publication is explicitly Phase 7's (documented on the accessor), not a stub.

## Threat Flags

None — no new trust-boundary surface. T-4-27 (alternation spoofing: bounded ring + named config error, no listing), T-4-28 (log flood: the shared per-`(node_id, class)` limiter covers all three duplicate paths), T-4-29 (stale notification broadcast: tracked-set guard with the first-notification path pinned) and T-4-30 (seen-ring growth: fixed 8-entry array) are mitigated as planned. No new dependency (T-4-SC).

## Next Phase Readiness

- PROTO-05 and PROTO-06 are implemented and covered. Their REQUIREMENTS.md checkboxes stay deliberately Pending: `requirements.ready-ids` blocks them because 04-13 (the full phase gate) also owns both IDs and has no SUMMARY yet; 04-13 will close them.
- 4 plans remain in Phase 4. Ready for `04-10-PLAN.md` (simulator N=3, complete-listing window, 14.9 evidence anchor).

---

*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Created files exist: this SUMMARY and both RED evidence records.
- Commits exist and are ancestors of HEAD: `f057c1e`, `05bf894`, `8d206b3`, `4fd41d5`.
- The measured plan commit count is 4 from the ledger base `cc13ce3`, matching the frontmatter `commits: 4`.
- No tracked file from the plan's diff was deleted.
