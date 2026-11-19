---
phase: 02-stand-and-simulator-as-tests
plan: 03
subsystem: testing
tags: [simnet, deterministic-simulation, delete, restart-with-state, receive-budget, size-gate, dump-delay, d05, test-06]

# Dependency graph
requires:
  - phase: 04-replication (v1.0 shipped)
    provides: the simnet package this plan extends (Scheduler/Policy/Network/Cluster seams), the repl.Core NotifyLocalWrite path the delete rides, the storage.Store.Delete contract (ErrNotFound for missing/already-tombstoned; fresh-version tombstone otherwise)
provides:
  - Cluster.Delete — the production author sequence for a deletion (store.Delete fresh-version tombstone → NotifyLocalWrite → route → drive → write log), propagating storage.ErrNotFound for missing/already-tombstoned keys (a no-op delete writes nothing and burns no version)
  - Cluster.ReplaceNodeWithState — ReplaceNode's state-seeding sibling: sorted-key store.Merge seeds the replacement store before the verbatim ReplaceNode wiring (in-place swap, Reregister, timer-gen supersession, drive tick)
  - Network.SetRecvBudget — per-node incoming-delivery budget (handle-keyed, survives ReplaceNode), consulted at fire time; deliveries past budget refused and counted in FaultStats.RecvBudgetRefused
  - Policy.SizeOf + Policy.MaxDatagramBytes — the test-injected encoder size gate refusing an oversize message before it is ever scheduled; counted in FaultStats.OversizeRefused (the real replproto encode stays test-side — simnet never imports it)
  - Policy.AnnounceDelay — extra virtual latency on port.DataReplication deliveries (write broadcasts and DataRequest/sync answers alike), mirroring production's dump-then-broadcast order
affects: [02-15 (extension tests incl. the same-seed TraceDigest replay gate are this API's executable proof), the D-06 chaos-corpus/scenario plans (deletions in the mix), phase 3+ plans consuming deletions and restart-with-state scenarios, 02-11 (scheduled chaos jobs ride these extensions)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 3787    # chars/4 over the realized diff (15148 chars: 188 insertions / 13 deletions in simnet.go)
  tasks: 2
  commits: 2      # MEASURED from the plan-start ledger at SUMMARY write (#3968)
  plan_head_before: c7b3d9521ed207e28c677a44f90654107a131b10
  plan_head_after: b4078ea922fd71c7bd193bfe95533c28c8549a77

# Tech tracking
tech-stack:
  added: none — stdlib only (maps/slices join the import block); zero new module dependencies (zero-install phase per the Package Legitimacy Audit)
  patterns:
    - additive simulator API: every extension is plain harness state or a policy hook consulted inside the existing deterministic paths; the zero Policy reproduces the pre-extension behavior bit for bit (existing sim_* suites green unchanged)
    - sorted-key merge seeding: the map-iteration guard the determinism contract needs whenever seeded state's HLC watermark is left behind by the merges themselves
    - refuse-and-count: budget and size refusals drop the message and count the class in FaultStats, never deliver; a size refusal is never even booked (PendingDeliveries stays a truthful quiescence signal)
    - determinism-contract doc comments on every new API, in the SetClockOffset style

key-files:
  modified:
    - internal/app/internal/simnet/simnet.go

key-decisions:
  - "Cluster.Delete returns (storage.Record, error): the plan's prose signature said a bare storage.Record while its own must_haves truth demands ErrNotFound propagation for missing/already-tombstoned keys — the error-carrying signature is the only shape that satisfies the truth; success returns the tombstone (Rec.Deleted true) that rode NotifyLocalWrite"
  - "ReplaceNodeWithState merges the seed records in sorted-key order (slices.Sorted(maps.Keys), the core.go listingChunks walk discipline): Go map iteration order would randomize the HLC watermark the merges leave behind — clock.Update's counter outcome depends on merge order whenever any record's l >= the merge-time virtual time — breaking same-seed replay; with a fixed order the post-seed clock state is a pure function of the seed, and the merges ride the clock's receive rule so the replacement's first local write dominates every seeded version (VERS-04 through the merge path, the persist.Open shape minus the redundant pre-seed)"
  - "The size gate refuses at schedule time (measures and refuses before booking — no pending increment, no event) while the receive budget is consulted at fire time — matching the plan's two wordings ('scheduleDelivery measures the message' vs 'consults it at delivery time') and keeping PendingDeliveries accurate for the three-seam quiescence"
  - "AnnounceDelay is consulted only for port.DataReplication messages — write-path broadcasts and unicast sync answers are the same wire type, which is exactly RESEARCH D-05's 'record announcements (DataReplication/sync answers)'; simnet adds the returned latency to the drawn one-way delay"
  - "A seed record over the store's design-default value limit panics (the checkIndex/Reregister convention — a driving test's mistake, never a protocol condition): simnet's store keeps storage.Config{} at 61440 and legitimate seed flows (another store's Records()) can never exceed it"
  - "Delete mirrors Write verbatim (no checkIndex — Write's precedent) and the error path uses the plain-assignment form per the non-test code convention; a no-op delete returns storage.Record{} and the store's ErrNotFound unchanged"

patterns-established:
  - "Determinism contract doc comment on every simnet API addition: plain harness state, consumes no rng draws and schedules no events (beyond the drive tick where the seam books one), so the event stream stays a pure function of the seed — plan 02-15's same-seed TraceDigest replay gate holds this line machine-checked"
  - "Seeding state through store.Merge in sorted-key order: any future simnet API that seeds a store must walk keys sorted, never a map — the HLC watermark the merges leave behind is order-sensitive"
  - "Refuse-and-count fault classes land in FaultStats appended after the four original fields (positions unchanged), one class per extension path"

patterns-established-note: See patterns-established above.

requirements-completed: [TEST-06, TEST-02]

coverage:
  - id: D1
    description: "Cluster.Delete — the deletion author sequence: store.Delete writes a fresh-version tombstone (ErrNotFound and zero side effects for missing/already-tombstoned — no version burned), NotifyLocalWrite carries the tombstone through the same pending/wait-set path, route/drive run, the (key, tombstone) pair joins the write log as LoggedWrite.Rec with Deleted=true (no LoggedWrite struct change — the model needs no separate flag)"
    requirement: TEST-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ ./internal/app/repl/ → both ok (existing suites green with the additive API present; the behavioral delete-in-simulator proof is plan 02-15's pinned scope)"
        status: pass
      - kind: unit
        ref: "grep 'Determinism contract: as Write' simnet.go → 1 (line 737); grep 'func (c \\*Cluster) Delete' → line 741"
        status: pass
    human_judgment: false
  - id: D2
    description: "Cluster.ReplaceNodeWithState — the restart-with-preserved-state seam: newClusterNode build, sorted-key store.Merge seeding (tombstones like live records; over-limit seeds panic per the harness convention), then ReplaceNode's wiring verbatim — timer-gen supersession captured before the swap, in-place Reregister, scheduled drive tick; determinism contract documented verbatim-style"
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ ./internal/app/repl/ → both ok; go vet ./internal/app/internal/simnet/ → clean"
        status: pass
      - kind: unit
        ref: "grep 'Determinism contract: plain harness state — the seeded merges' simnet.go → 1 (line 593); grep 'func (c \\*Cluster) ReplaceNodeWithState' → line 596"
        status: pass
    human_judgment: false
  - id: D3
    description: "Network.SetRecvBudget + FaultStats.RecvBudgetRefused — the per-node receive budget: plain-state setter keyed by the node's handle (survives ReplaceNode; unregistered node id panics), read live at each fire time inside scheduleDelivery, past budget the delivery is refused and counted, a delivered one decrements; zero/negative budget refuses everything; determinism contract documented"
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ → ok (nil-budget paths of every existing suite bit-identical)"
        status: pass
      - kind: unit
        ref: "grep 'func (n \\*Network) SetRecvBudget' → line 300; grep 'RecvBudgetRefused uint64' → line 199"
        status: pass
    human_judgment: false
  - id: D4
    description: "Policy.SizeOf + Policy.MaxDatagramBytes + FaultStats.OversizeRefused — the size gate: test-injected encoder measurement (the Q2 resolution — replproto never enters simnet's import block), gate armed only when SizeOf != nil AND MaxDatagramBytes > 0, refusal before booking, count per refusal (a duplicated send's two scheduleDelivery calls count independently)"
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "sed -n '/^import (/,/^)/p' simnet.go → stdlib + internal/app/* + internal/port only; grep -n replproto simnet.go → line 147 only (a doc-comment mention of the test-side encode)"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./internal/app/... → exit 0 (depguard boundary T-2-06 intact)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Policy.AnnounceDelay — the virtual dump delay: nil = none; simnet adds the returned latency to the drawn one-way delay of every port.DataReplication delivery (write broadcasts and sync answers), pure function of the message; documented with the RESULTS.md §4 dump-then-broadcast provenance"
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ ./internal/app/repl/ → both ok (nil AnnounceDelay → total == drawn delay, byte-identical scheduling)"
        status: pass
      - kind: unit
        ref: "grep 'AnnounceDelay func(msg port.Message) time.Duration' → line 167"
        status: pass
    human_judgment: false
  - id: D6
    description: "Nil-policy bit-identity — the determinism law preserved: zero Policy and no budgets reproduce the pre-extension behavior exactly (size gate skipped, total == drawn delay, budget lookup misses); existing sim_*_test.go suites run unchanged and green; repo-wide gates green"
    requirement: TEST-06
    verification:
      - kind: other
        ref: "make lint (config verify + run ./...) → exit 0; make test (go test -race -cover ./...) → exit 0, all 14 packages ok (simnet 59.6%, repl 97.8% coverage)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ ./internal/app/repl/ → ok 1.0s / ok 2.3s — chaos/bound/scenario suites with their fault teeth and convergence bounds all green unchanged"
        status: pass
    human_judgment: false

# Metrics
duration: 7min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 3: simnet API Extensions — Delete, Restart-with-State, Budget, Size Gate, Dump Delay Summary

**The simulator's five blind spots are closed as additive deterministic API: `Cluster.Delete` (tombstones through the production NotifyLocalWrite path with ErrNotFound propagation), `Cluster.ReplaceNodeWithState` (sorted-key Merge seeding before the verbatim ReplaceNode wiring), `Network.SetRecvBudget` (per-node receive-side refusals counted in FaultStats), the test-injected `Policy.SizeOf`/`MaxDatagramBytes` size gate, and `Policy.AnnounceDelay` (the dump-then-broadcast mirror) — the zero Policy reproduces today's behavior bit for bit, existing sim suites green unchanged, and simnet's import block stays stdlib + internal/app + internal/port.**

## Performance

- **Duration:** 7 min (441 s)
- **Started:** 2026-11-19T20:13:11Z
- **Completed:** 2026-11-19T20:20:32Z
- **Tasks:** 2
- **Files changed:** 1 (internal/app/internal/simnet/simnet.go, +188/−13)

## Accomplishments

- **`Cluster.Delete(i int, key string) (storage.Record, error)`** — the deletion author sequence mirroring `Cluster.Write` exactly: `store.Delete` writes a fresh-version tombstone that merges like any write, the tombstone record rides `core.NotifyLocalWrite` through the same pending/wait-set path (Deleted=true crosses the wire per message), the drained outbox routes through the virtual network, the drive loop runs, and the (key, tombstone) pair joins the write log — `LoggedWrite.Rec` carries Deleted=true, no struct extension needed (the plan's conditional: the model needs no explicit flag). A missing or already-tombstoned key returns `storage.ErrNotFound` and changes nothing — the no-op delete writes nothing and burns no version (the store.Delete contract, documented in the doc comment). Determinism contract: as Write — the delete runs at the current virtual time and every emission routes through the network's seeded rng; no new randomness source.
- **`Cluster.ReplaceNodeWithState(i int, runID [16]byte, offset time.Duration, records map[string]storage.Record)`** — ReplaceNode's state-seeding sibling (TEST-06/F14: a replaced node carrying the dump's records, seeding its create-if-absent defaults): the records merge into the fresh store through the production load path's join (`store.Merge`, the persist.Open mergeRecords shape) **in sorted-key order** — the fixed order is the map-iteration guard the determinism contract needs — then the wiring is ReplaceNode's verbatim: timer-generation supersession captured before the swap, in-place node swap, `Reregister` (in-flight datagrams from the old run keep the old run id), and the scheduled drive tick. The merges ride the clock's receive rule, so the replacement's HLC watermark dominates every seeded version before its first local write. Doc comment carries the plan's determinism contract verbatim-style: plain harness state, consumes no rng draws and schedules no events beyond the drive tick.
- **`Network.SetRecvBudget(nodeID uint16, maxDeliveries int)`** — the per-node incoming budget (TEST-06): a test-facing plain-state setter on Network, keyed by the target's handle so it survives a ReplaceNode restart (the socket address the budget models outlives the process behind it), read live at each delivery's fire time inside `scheduleDelivery` — past budget the delivery is refused (message dropped, never delivered) and counted in the new `FaultStats.RecvBudgetRefused`; a delivered datagram decrements the counter. Zero/negative budget refuses everything; an unregistered node id panics (the harness convention). Determinism contract: as SetClockOffset.
- **`Policy.SizeOf func(port.Message) int` + `Policy.MaxDatagramBytes`** — the size gate with the Q2 resolution's injected-encoder shape: the test layer injects the real encoder's length measurement (replproto stays test-side; simnet.go's import block is verified stdlib + internal/app + internal/port with one doc-comment mention only), and `scheduleDelivery` refuses a message measuring over the ceiling **before it is ever scheduled** — counted in `FaultStats.OversizeRefused`, never booked, so `PendingDeliveries` stays a truthful quiescence signal (the message-too-long refusal of a real UDP path, F2c). The gate arms only when SizeOf != nil AND MaxDatagramBytes > 0.
- **`Policy.AnnounceDelay func(port.Message) time.Duration`** — the virtual dump delay on record announcements: simnet consults it only for `port.DataReplication` messages (write-path broadcasts and DataRequest/sync answers are the same wire type — RESEARCH D-05's "record announcements (DataReplication/sync answers)") and adds the returned latency to the drawn one-way delay, mirroring production's dump-then-broadcast order (RESULTS.md §4). Pure function of the message — no rng, no wall clock.
- **`FaultStats` extension** — `RecvBudgetRefused` and `OversizeRefused` appended after the four original fields (existing fields/positions unchanged); the struct doc enumerates all six classes.
- All extensions document the determinism contract in the SetClockOffset style; every existing sim suite (chaos/bound/scenario with their fault teeth and convergence bounds) runs unchanged and green.

## Task Commits

1. **Task 1: Cluster.Delete + Cluster.ReplaceNodeWithState** - `b8ff560` (feat)
2. **Task 2: receive budget, SizeOf refusal, dump-delay announcements** - `b4078ea` (feat)

_Measured from the plan-start ledger `c7b3d95` (`git rev-list --count c7b3d95..HEAD` at SUMMARY write): 2 task commits._

## Files Created/Modified

- `internal/app/internal/simnet/simnet.go` — modified (+188/−13): the two Cluster methods, the Network budget setter + field, the Policy size gate and announcement-latency hooks, the scheduleDelivery extension, the two FaultStats counters, and the determinism-contract doc comments throughout.

## Decisions Made

- **Delete carries the error the must_haves demand.** The plan's prose signature (`storage.Record` bare) conflicts with its own must_haves truth ("propagates ErrNotFound for missing/already-tombstoned keys"); the implemented `(storage.Record, error)` shape is the only one satisfying the truth — see Deviations 1.
- **Sorted-key merge seeding.** `ReplaceNodeWithState` merges in `slices.Sorted(maps.Keys(...))` order (the core.go listingChunks walk discipline): Go map iteration order would randomize the HLC watermark the merges leave behind — `clock.Update`'s counter outcome depends on merge order whenever any record's l ≥ the merge-time virtual time — breaking same-seed replay. With a fixed order the post-seed clock state is a pure function of the seed. The plan's "merge each record into the replacement store" left the order open; this is the only determinism-preserving reading of it. The VERS-04 restart property arrives through the merge path (the clock's receive rule), so the production Open path's explicit Watermark pre-seed is redundant here and omitted.
- **Schedule-time size refusal, fire-time budget refusal.** The plan says the budget is consulted "at delivery time" and that scheduleDelivery "measures the message" for the size gate — implemented literally: the size gate refuses before booking (no pending increment, quiescence stays accurate), the budget is consulted in the fire-time closure against the live map (a budget set between schedule and fire still governs that delivery).
- **AnnounceDelay rides DataReplication only.** The plan's "for record-announcement messages (DataReplication)" qualifier is implemented as a type assertion in scheduleDelivery; sync answers are covered because they are DataReplication on the wire.
- **Over-limit seeds panic.** simnet's harness store keeps the design-default 61440 limit and legitimate seed flows (another store's Records()) can never exceed it, so an over-limit seed is a driving test's mistake — the checkIndex/Reregister panic convention applies, not production's skip (which exists for real dumps with shrunk configs simnet does not model).
- **Delete mirrors Write's exact shape** — no checkIndex (Write's precedent), plain-assignment error form (the non-test code convention), write-log append after drive.

## Deviations from Plan

**1. [Rule 3 - Blocker] Cluster.Delete's return carries the error**
- **Found during:** Task 1 implementation
- **Issue:** the plan's action names the signature `Cluster.Delete(i int, key string) storage.Record` while the same plan's must_haves truth requires "propagates ErrNotFound for missing/already-tombstoned keys" as "the return error path the test layer asserts" — a bare Record cannot carry it
- **Fix:** implemented `Delete(i int, key string) (storage.Record, error)`; success returns the tombstone, failure returns `storage.Record{}` plus the store's ErrNotFound
- **Files modified:** internal/app/internal/simnet/simnet.go (signature only — the action's body otherwise followed verbatim)
- **Verification:** the ErrNotFound path is the must_haves truth; suites green
- **Commit:** `b8ff560`

**Total deviations:** 1
**Impact:** the return-signature delta is forced by the plan's own must_haves; every other element of both tasks landed exactly as written (verified line-by-line against the plan action).

## Issues Encountered

- **The plan-start ledger carried a stale value.** The `gsd-plan-head-before-02-03` ledger file already existed on disk from a prior milestone's plan 02-03 execution (mtime 2026-11-02, pointing at an October commit `573abd9` unreachable from this branch), so the protocol's create-only guard kept it and the first commit measurement read the repo's entire 63-commit history. Caught at SUMMARY measurement time (the count contradicted the session's two commits), corrected to the true plan-start HEAD `c7b3d95` (the parent of this plan's first commit, verified `git rev-parse b8ff560^`), and re-measured: 2 commits. No code impact.
- No lint or test failures at any point: both task verifies passed on the first run.

## Known Stubs

None. Every API delivered is complete, documented, and consulted in the deterministic paths; nothing returns placeholder data. The behavioral proof of this API (extension tests including the same-seed TraceDigest determinism replay gate) is **not** a stub — it is dependent sibling plan 02-15's pinned scope (wave 2), per the plan's own revised-iteration header; this plan delivers the API, 02-15 proves it.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 02-15 (wave 2) lands `simnet_ext_test.go` against this API: the extension tests including the same-seed TraceDigest equality gate with all extensions active — the executable proof of the determinism contract every doc comment here claims.
- The D-06 chaos-corpus plan gains deletions in the mix through `Cluster.Delete`; restart-with-state scenarios (F14's simulator level) ride `Cluster.ReplaceNodeWithState`.
- The budget/size/dump-delay hooks are the teeth the nightly-scale scenario work (02-11's scheduled jobs) models receive-side drops, oversize datagrams and dump latency with — all off by default, so every existing scenario's meaning is unchanged.
- No blockers. simnet's non-test import block remains stdlib + internal/app + internal/port (T-2-06 depguard gate green); T-2-05's machine enforcement is 02-15's replay gate.

## Self-Check: PASSED

Re-run on the committed tree (`b4078ea`):

- Key file exists: `internal/app/internal/simnet/simnet.go` (33556 bytes) — FOUND.
- Commits exist: `b8ff560`, `b4078ea` — both ancestors of HEAD; `git rev-list --count c7b3d95..HEAD` → 2.
- `gsd-tools check evaluation-scope --plan 02-03 --commits-only --raw` → resolved, 2 commits, exit 0 (not 69).
- `go test -race -count=1 ./internal/app/internal/simnet/ ./internal/app/repl/` → ok 1.0s / ok 2.3s — existing suites unchanged and green (success criterion 4).
- `go tool golangci-lint run ./internal/app/...` → exit 0; `make lint` → exit 0 (config verify + run ./...).
- `make test` (`go test -race -cover ./...`) → exit 0, all 14 packages ok (simnet 59.6%, repl 97.8% coverage).
- `grep 'func (c \*Cluster) Delete'` → line 741; `grep 'func (c \*Cluster) ReplaceNodeWithState'` → line 596; both doc comments carry the determinism contract (lines 737, 593) — success criterion 1.
- `grep 'RecvBudgetRefused uint64'` → line 199; `grep 'OversizeRefused uint64'` → line 203; `Policy.SizeOf`/`MaxDatagramBytes`/`AnnounceDelay` at lines 153/157/167, consulted in scheduleDelivery (lines ~355-373) — success criterion 2.
- `sed -n '/^import (/,/^)/p' simnet.go` → stdlib (container/heap, fmt, maps, math, math/rand/v2, slices, strings, time) + internal/app/{clock,repl,storage} + internal/port only; `grep -n replproto simnet.go` → line 147 (doc comment only) — success criterion 3.

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-19*
