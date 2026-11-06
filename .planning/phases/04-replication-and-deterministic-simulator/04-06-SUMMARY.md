---
phase: 04-replication-and-deterministic-simulator
plan: "06"
subsystem: replication
tags: [udp, udp-broadcast, replication, discard-classification, duplicate-node-id, crdt, synctest, smoke-test, repl-01, repl-08, repl-09, go]

# Dependency graph
requires:
  - phase: 04-replication-and-deterministic-simulator/plan 01
    provides: the classification skeleton with the pinned order and counters, PeerSnapshot/PeerInfo, the UDP adapter and its atomic mirroring this plan's smoke test runs against
  - phase: 04-replication-and-deterministic-simulator/plan 04
    provides: the Hello-handler restart reaction (targeted listing) the classifier must never pre-empt, and the handleHello doc surface the discrimination pin extends
  - phase: 03-persistence-dump-and-write-confirmation
    provides: the reserved dirty-path seam ("remote merges will reuse the dirty path without waiters") and the synctest suite shape ApplyRemote's test reuses
provides:
  - the completed D-02 discard classification: all five classes distinguishable and total — the second duplicate-node_id variant (a known node_id whose remembered run_id repeats verbatim from a DIFFERENT handle) is dropped, counted and logged as a configuration error naming the node_id and both conflicting handles; a changed run_id is never a duplicate (the classifier never pre-empts the plan-04-04 restart branch, doc-pinned in HandleDatagram and handleHello)
  - REPL-09 totality proof: 500 seeded adversarial datagrams (random framing verdicts, any run_id bit pattern, every message shape) with zero panics and exact counter conservation via a model function (per-class equality, sum == discarded, peer-table agreement)
  - persist.Engine.ApplyRemote — REPL-03's memory-apply contract: Merge + markDirty + the non-blocking changed nudge, no waiter, no flush, store errors passed through unchanged
  - the D-08 real-socket two-node smoke test (TestSmoke_twoNodesUDP): pre-flight capability probe (127.255.255.255, then first broadcast-capable non-loopback IPv4 subnet broadcast, then a loud skip naming both addresses), mutual discovery from broadcast Hellos alone, write visibility byte-equal on node B, and node A's own broadcast echo counted in OwnEcho on real Linux sockets
affects: [phase 05 gRPC write handler (NotifyLocalWrite + ApplyRemote wiring over the real engine), phase 06 OBS observability counters (DiscardStats is the OBS-02/03 seam), phase 06 failure scenarios (duplicate node_id misconfiguration), DOCS-01 quickstart (same-host broadcast topology)]

# Actuals (#2632) — pairs with the plan's `estimate` (estimateTokens scale: chars/4 over the realized diff).
actuals:
  tokens: 9300
  tasks: 2
  commits: 3

plan_head_before: d92550cb7ccfb3daf71e0834b4b6260efa9f7b60
plan_head_after: 6e3bb655ff7f18c96dff681a128c521069e3286a

# Tech tracking
tech-stack:
  added: []
  patterns: [classifier-level identity checks ahead of message dispatch (duplicate node_id never reaches protocol processing), model-function conservation proof for a classifier (expectedDrift precedent applied to discard counting), no-waiter dirty paths need an explicit dumper wake (markDirty alone parks Run), synctest.Sleep for no-waiter dump timing (Wait does not advance the bubble clock), capability-probed smoke tests (probe, subnet fallback, loud skip)]

key-files:
  created:
    - internal/app/repl/discard_test.go
    - internal/udprepl/smoke_test.go
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/retransmit_test.go
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go

key-decisions:
  - "The second duplicate-node_id variant keys on run_id EQUALITY with the remembered peer row plus a DIFFERING source handle: run_id inequality is the restart path, so the classifier never pre-empts the plan-04-04 restart branch — pinned in HandleDatagram's doc comment and echoed in handleHello's"
  - "Engine.ApplyRemote = store.Merge + markDirty + the non-blocking changed nudge: markDirty alone leaves an idle dumper parked in its select with no timer armed (nothing would ever dump), so the seam sends await's own wake signal — no waiter, no flush, errors unchanged, not gated on Run's exit"
  - "The no-waiter dump path advances the synctest bubble clock with synctest.Sleep: synctest.Wait returns without firing timers once every other goroutine is blocked, and the existing waiter tests never hit this because their blocked Set calls hold the test goroutine until the dump resolves"
  - "The smoke test's node stores are storage.Store behind a Merge-delegating wrapper (the retransmit-test precedent): D-08 keeps the smoke about real protobuf datagrams, real sockets and real addressing — dumps stay with the ApplyRemote seam's own synctest proof"
  - "The persist RED carried a merge-only ApplyRemote stub inside the test commit (tdd.md's make-it-compile instruction): the stub made TestEngine_applyRemoteTriggersDump compile and fail on the planned no-dump assertion, both RED runs machine-classified RED_EVIDENCE_OK"

patterns-established:
  - "Identity-claim classification before dispatch: any datagram claiming a known identity from a second handle is discarded at the classifier level — Hello, data and control traffic alike never see it"
  - "Adversarial totality via model function: the test mirrors only the classifier's decision inputs (peer-table handle/run_id upserts), never the protocol handlers, so counter conservation is provable without duplicating the implementation"

requirements-completed: [REPL-01, REPL-08, REPL-09]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "All five discard classes distinguishable and total (REPL-09): counter-per-class isolation over nine rows (zero-length, garbage, CRC-failure-with-nil-body, proto-version+1, own-echo, both duplicate variants, valid Hello, restart) plus 500 seeded adversarial datagrams with zero panics and exact counter conservation"
    requirement: REPL-09
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_classes (go test -race ./internal/app/repl/ -run '^TestDiscard_' -v)"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_totality (500 seeded rounds, model-function conservation)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both duplicate-node_id variants dropped AND surfaced (REPL-08): own node_id with a foreign run_id, and a known node_id repeating its remembered run_id from a second handle — each logs a configuration-error line through Config.Log naming the node_id and the conflicting handles, and nothing is silently resolved"
    requirement: REPL-08
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_classes rows 'own node_id with a foreign run_id…' and 'known node_id repeating its run_id from a second handle…' via assertConfigErrorLog"
        status: pass
    human_judgment: false
  - id: D3
    description: "Restart-vs-duplicate discrimination pin: a known node_id whose announced run_id differs from the remembered one is processed as a restart (entry update plus the plan-04-04 targeted listing) and never counted DuplicateNodeID — doc-pinned at the classifier"
    requirement: REPL-08
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_classes row 'a known node_id with a changed run_id is a restart, never a duplicate' (zero counters, updated PeerInfo, one listing emission)"
        status: pass
    human_judgment: false
  - id: D4
    description: "persist.Engine.ApplyRemote — REPL-03's memory-apply contract: remote merge dirties the engine with no waiter (returns at the apply instant, strictly before the dump), exactly one debounced dump whose bytes decode to the record, ErrValueTooLarge passed through unchanged with the engine still serving"
    verification:
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestEngine_applyRemoteTriggersDump (synctest bubble, fakeDumpPort)"
        status: pass
    human_judgment: false
  - id: D5
    description: "D-08 real-socket two-node smoke test: capability probe (loopback broadcast, subnet-broadcast fallback, loud skip naming both addresses), mutual discovery from broadcast Hellos alone (REPL-01 zero-config), one write on A visible on B byte-equal, node A's own echo counted in OwnEcho — no Ack/unicast assertions (same-host unicast reaches exactly one socket, kernel's choice)"
    requirement: REPL-01
    verification:
      - kind: integration
        ref: "internal/udprepl/smoke_test.go#TestSmoke_twoNodesUDP (go test -race ./internal/udprepl/ -run '^TestSmoke_twoNodesUDP$' -v — PASS in 0.02s on this Linux box, also green x5 repeated)"
        status: pass
    human_judgment: true
    rationale: "The probe-fallback and loud-skip paths are platform-conditional: they only execute where loopback broadcast fan-out is absent (macOS lo0 carries no BROADCAST flag), which this Linux machine cannot exercise. The first macOS CI gate run is the empirical proof research open question 1 defers — the outcome is to be recorded for Phase 6."

# Metrics
duration: 15 min
completed: 2026-11-06
status: complete
---

# Phase 4 Plan 06: Discard Classification Completion, ApplyRemote Seam, Real-Socket Smoke Test Summary

**The full D-02 discard contract (both duplicate-node_id variants dropped, logged as configuration errors and counted; restarts never pre-empted; 500 adversarial datagrams with exact counter conservation), the persist.Engine.ApplyRemote memory-apply seam (merge dirties, no waiter), and the D-08 two-node real-UDP smoke test with capability probe — own echo observed on a real Linux socket.**

## Performance

- **Duration:** 15 min (19:53–20:09 UTC)
- **Started:** 2026-11-06T19:53:55Z
- **Completed:** 2026-11-06T20:09:22Z
- **Tasks:** 2 (Task 1 TDD red-green; Task 2 smoke test)
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments
- The five discard classes proven distinguishable and total (REPL-09): TestDiscard_classes pins counter isolation over nine rows — zero-length and garbage datagrams classify Undecodable, a CRC failure with Msg nil classifies BadCRC, proto-version+1 classifies BadProtoVersion, own node_id + own run_id classifies OwnEcho — and TestDiscard_totality feeds 500 seeded adversarial datagrams (RandRecord/RandKey/RandRawVersion bit patterns, random verdicts and every message shape) proving zero panics and exact counter conservation against a model function
- Both duplicate-node_id variants dropped AND surfaced (REPL-08): own id under a foreign run id, and the new second variant — a known node_id whose remembered run_id repeats verbatim from a second handle — each counting DuplicateNodeID exactly once and logging a configuration-error line through Config.Log that names the node_id and the conflicting handles; the restart branch is doc-pinned as never pre-empted (run_id inequality is the restart path and fires the plan-04-04 targeted listing, proven by the restart row's zero counters and updated PeerInfo)
- persist.Engine.ApplyRemote landed (REPL-03's memory-apply contract, Pitfall 7 closed): Merge + markDirty + the non-blocking changed nudge — the nudge is load-bearing, an idle dumper parked in its select would otherwise never arm its debounce timers — with no waiter (the call returns strictly before the dump), exactly one debounced save decoding to the merged record, ErrValueTooLarge passed through unchanged and the engine still serving
- TestSmoke_twoNodesUDP (D-08): two Adapters over genuine UDP on one host discover each other purely via broadcast Hello (both peer counts == 1, REPL-01 zero-config), one local Set on A becomes byte-equal visible on B, and A's own broadcast echo is counted in OwnEcho on a real Linux socket; the pre-flight capability probe tries 127.255.255.255, then the first broadcast-capable non-loopback IPv4 subnet broadcast, then skips loudly naming both addresses — and the test never asserts Ack/DataRequest/unicast delivery (same-host unicast reaches exactly one REUSEADDR socket, the kernel's choice)
- Full gates green: make test (whole repo, -race -cover) and go tool golangci-lint config verify && go tool golangci-lint run ./... both exit 0; go.mod/go.sum untouched (T-04-SC)

## Task Commits

Each task was committed atomically (TDD: RED test commit then GREEN feat commit; Task 2 is a pure test deliverable):

1. **Task 1 RED: failing TestDiscard_classes/TestDiscard_totality + TestEngine_applyRemoteTriggersDump with a merge-only ApplyRemote compile stub** - `f839c0d` (test)
2. **Task 1 GREEN: second duplicate-node_id class in HandleDatagram + Engine.ApplyRemote seam** - `4042699` (feat)
3. **Task 2: TestSmoke_twoNodesUDP with the capability probe** - `6e3bb65` (test)

**Plan metadata:** (this SUMMARY commit)

Both RED runs were machine-classified `RED_EVIDENCE_OK` (go-test adapter): TestDiscard_classes failed on exactly the planned second-handle row (DuplicateNodeID stayed 0, the datagram reached protocol processing and upserted the entry; totality's conservation broke 16-vs-27 with sum 344 != 355), TestEngine_applyRemoteTriggersDump failed on "completed saves = 0, want exactly one dump" — the missing dirty-marking. The GREEN commit directly descends from the RED commit (merge-base verified). No REFACTOR commit: GREEN landed in minimal shape; the post-GREEN edits (lint fixes and the synctest harness note below) rode the same commits.

## Files Created/Modified
- `internal/app/repl/core.go` - HandleDatagram's second DuplicateNodeID variant (known node_id, remembered run_id, second handle) with the configuration-error log naming node_id and both handles; the never-pre-empt-the-restart-branch doc pin in HandleDatagram and handleHello; StateStore doc refreshed to the shipped seam
- `internal/app/repl/discard_test.go` - TestDiscard_classes (nine rows, full-counter equality, assertConfigErrorLog, restart pin), TestDiscard_totality (500 seeded rounds, modelPeer/classify/bump conservation model, randDatagram/randBody generators)
- `internal/app/repl/retransmit_test.go` - stateStore doc comment refreshed (the seam this plan shipped; one stale line outside the plan's files_modified list)
- `internal/app/persist/engine.go` - Engine.ApplyRemote: Merge + markDirty + changed nudge, no waiter, errors unchanged, REPL-03 doc citing the reserved seam
- `internal/app/persist/engine_test.go` - TestEngine_applyRemoteTriggersDump (synctest bubble: one dump, return-before-dump, oversize passthrough, still serving) + applyRemoteOversize constant
- `internal/udprepl/smoke_test.go` - TestSmoke_twoNodesUDP + capability probe (freeUDPPort, smokeSocket, broadcastReachable, subnetBroadcast, waitFor with t.Deadline protection)

## Decisions Made
- Second duplicate variant = remembered run_id equality + differing handle; the restart branch stays reachable by construction (doc pin at both ends)
- ApplyRemote wakes the dumper with await's own non-blocking changed signal — without it a clean engine's remote merge would never dump (markDirty mutates state nobody polls)
- synctest harness shape for no-waiter paths: synctest.Sleep advances the bubble clock to the debounce fire; synctest.Wait does not (it returns once the other goroutines are blocked)
- Smoke scope: storage.Store behind the Merge-delegating wrapper, dumps excluded (D-08 minimal; the persist seam has its own proof), adapter's existing same-host doc comment already documents the unicast limitation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Stale-doc hygiene] retransmit_test.go seam comment refreshed**
- **Found during:** Task 1 (GREEN)
- **Issue:** The stateStore adapter's comment still said "the persist engine's ApplyRemote seam lands with plan 04-06" — stale the moment this plan shipped it; retransmit_test.go is not in the plan's files_modified list
- **Fix:** One-line comment update naming the shipped production applier
- **Files modified:** internal/app/repl/retransmit_test.go
- **Verification:** go tool golangci-lint run ./internal/app/repl/... exits 0
- **Committed in:** 4042699

---

**Total deviations:** 1 auto-fixed (1 Rule 1 doc hygiene, one line outside the plan's file list)
**Impact on plan:** None — no behavioral or scope change; the plan's five artifacts all landed as specified.

## Issues Encountered
- The persist test's first GREEN run still failed ("completed saves = 0"): synctest.Wait returns without advancing the bubble clock once the other goroutines are blocked — the existing waiter tests never hit this because their blocked Set calls hold the test goroutine until the dump resolves. Fixed by advancing the clock with synctest.Sleep(2×debounce); the RED assertion semantics are unchanged (the merge-only stub fails identically under both harness shapes), so the recorded RED evidence remains accurate for the final test
- Strict-lint findings fixed in code direction, zero suppressions added: gofmt alignment and nestif (log assertion extracted to assertConfigErrorLog) in discard_test.go, noinlineerr split in engine_test.go, and noctx/usetesting/golines/noinlineerr in smoke_test.go (net.ListenConfig method with t.Context, t.Context for the run context, reformatted skip messages, plain error assignment)
- net.ListenConfig{} is not addressable for its pointer-receiver ListenPacket — freeUDPPort uses a var lc net.ListenConfig

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 4 complete: all six plans have summaries — REPL-01..10 and TEST-03 delivered; the replication core, adapter, simulator, persist seam and smoke test form the closed loop the gRPC phase consumes
- Ready for Phase 5 (API): the gRPC write handler wires store write -> NotifyLocalWrite (D-05 author call, udprepl.Adapter.NotifyLocalWrite with its 1024-deep buffer) and the inbound DataReplication path already lands through the engine's ApplyRemote — Engine now satisfies repl.StateStore directly
- Flagged assumptions #17 (crossing first Hellos converge by construction; simultaneous start exercised live by the smoke test and by 04-05's seeded chaos) and #20 (duplicate-node_id split: own-id-foreign-run and second-handle-same-run both DuplicateNodeID config errors; different run_id is a restart) are pinned in code and tests for phase verification to ratify
- D-08's macOS outcome stays empirical until the first macOS CI gate run (research open question 1's deferred empirical outcome — record it for Phase 6)
- Full make lint + make test green (CI parity)

## Self-Check: PASSED

All five plan artifacts exist on disk; the three task commits f839c0d/4042699/6e3bb65 are ancestors of HEAD (evaluation-scope resolved 3 commits for plan 04-06); every task's acceptance criteria re-verified (AC greps and test runs logged above); all plan <verification> commands re-run green under -race (TestDiscard_classes, TestDiscard_totality, TestEngine_applyRemoteTriggersDump, TestSmoke_twoNodesUDP, make test, full lint); TDD gates present with RED-before-GREEN ancestry and both RED runs machine-classified RED_EVIDENCE_OK.

---
*Phase: 04-replication-and-deterministic-simulator*
*Completed: 2026-11-06*
