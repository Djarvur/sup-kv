---
phase: 04-protocol
plan: 12
subsystem: replication-protocol
tags: [proto-09, multi-node, documentation, operations-handoff, spec-deviations, wait-for-replication, residual-windows, status-complete]

# Dependency graph
requires:
  - phase: 04-protocol
    plan: 08
    provides: "the first-seen targeted listing and the bounded listing caps the traffic-shape and protocol-bounds sections record"
  - phase: 04-protocol
    plan: 10
    provides: "the N=3 simulator scenario and the complete-listing/join evidence the timing section cites"
  - phase: 04-protocol
    plan: 11
    provides: "the whole-suite green record and the listing-drift/WARN pins the artifact relies on"
  - phase: 04-protocol
    plan: 07
    provides: "the real-socket three-node test whose shared-port topology and unicast limitation the document names"
  - phase: 04-protocol
    plan: 01
    provides: "the DataSync completeness header, per-part accumulation and bad_sync_header class the deviation rows describe"
provides:
  - "docs/04-MULTI-NODE.md — the N-node traffic, timing, wait_for_replication, bounds, topology and residual-window artifact PROTO-09 requires"
  - "the OPERATIONS handoff lines: bad_sync_header in the discard-class list and --startup-gate-empty / SUPKV_STARTUP_GATE_EMPTY in the gate description"
  - "five SPEC-DEVIATIONS rows (S-HLC-04, S-HLC-10, S-PROTO-18, S-ALG-ST-06, S-ALG-RC-05) flipped to implemented with current code and test evidence"
affects:
  - "Phase 6 (APIS-01/DOCS-05) folds the wait_for_replication wording and IN-05 into docs/API.md"
  - "Phase 7 owns the peer-table hardening, drift semantics and the one-point startup_gate_empty relation validation"
  - "Phase 9 owns the node-replacement runbook wording and the final OPERATIONS catalog"

actuals:
  tokens: 5718
  tasks: 2
  commits: 2
  plan_head_before: 379eddf67f885a9e4fb59307095dc29ff4586570
  plan_head_after: 47ecdccfd30a7596ec3d0a9128ba5d5dcfa7b218

tech-stack:
  added: []
  patterns:
    - "Operator-facing claims carry a code or test reference on the same line: a bound names its constant, a timing statement names the scenario, and every residual window names its owner or acceptance."
    - "A deviation row is flipped only with current evidence: implemented rows describe the shipped behavior in own words and cite the exact handler, accumulation point and test that prove it."
    - "A handoff edit touches only the lines the later phase will fold in: the operations knobs are documented where the gate semantics already live, without a new catalog."

key-files:
  created:
    - docs/04-MULTI-NODE.md
  modified:
    - docs/OPERATIONS.md
    - docs/SPEC-DEVIATIONS.md

key-decisions:
  - "Every multi-node claim is tied to a code symbol or test name; the residual section labels each window 'accepted' or 'residual' with Phase 6/7/9 ownership, so the artifact never states a guarantee the code does not provide."
  - "The traffic-shape section names the startup listing alongside the periodic one and the oversized-entry chunk exception, because 'one listing per sync_interval' alone would understate the startup round and overstate the chunk budget's universality."
  - "The five deviation summaries were rewritten from deviation wording to shipped behavior (the plan's option 3), keeping each ID and its REVIEW §7/G reference; no other row was touched."
  - "OPERATIONS' stale 'reads are never gated by startup' sentence was corrected while documenting the new knob: ADR 0005/D-01 closes reads and writes alike, so leaving it would contradict the same paragraph's new gate description."
  - "PROTO-09 stays unchecked in REQUIREMENTS.md: requirements.ready-ids reports it blocked because 04-13's plan also owns it; 04-13 closes it."

patterns-established:
  - "An N-node artifact is organized by operator symptom (normal write, partition heal, evicted-peer heal, join), not by protocol mechanism."
  - "Residual risk is documented as an explicit multiplier: the bounded retained listed-key payload is named beside the unbounded peer-row table so the peer × key exposure is legible."
  - "Deviation-registry evidence uses stable code symbols plus current line ranges and test names, so the row remains checkable after later refactors."

requirements-completed: [PROTO-09]

coverage:
  - id: D1
    description: "docs/04-MULTI-NODE.md exists with all seven required sections: traffic shape, timing, wait_for_replication at N > 2, protocol bounds, three-node test topology, accepted residual windows and handoffs."
    requirement: PROTO-09
    verification:
      - kind: command
        ref: "for s in '## Traffic shape' '## Timing' '## wait_for_replication at N > 2' '## Protocol bounds' '## Three-node topology in tests' '## Residual windows' '## Handoffs'; do grep -qF -- \"$s\" docs/04-MULTI-NODE.md || exit 1; done"
        status: pass
      - kind: command
        ref: "grep -q 'Phase 6' docs/04-MULTI-NODE.md && grep -q 'Phase 9' docs/04-MULTI-NODE.md && grep -qE 'maxListingDestinations|listing destinations' docs/04-MULTI-NODE.md && grep -qE 'maxListedKeysTotal|retained listed keys' docs/04-MULTI-NODE.md && grep -q 'kernel-arbitrary' docs/04-MULTI-NODE.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "The wait_for_replication section states the write-time total, all-peer Ack requirement, the WR-01 clamp and the IN-05 safe-direction limitation, and names Phase 6 APIS-01/DOCS-05 as the owner of tightening and final docs/API.md wording."
    requirement: PROTO-09
    verification:
      - kind: static
        ref: "docs/04-MULTI-NODE.md ## wait_for_replication at N > 2 vs internal/grpcapi/wait.go:66-84 and the ServerOptions seams in internal/grpcapi/gate.go:47-94"
        status: pass
    human_judgment: false
  - id: D3
    description: "The protocol-bounds section lists the eight shipped constants (16 streams, 65 536 entries/stream, 131 072 total entries, 65 536 chunk cap, 131 072 retained listed keys, 32 destinations, 65 536 queued chunks, 8 seen runs) with rationale, and names SyncStreamDropped, ListedKeysEvicted, ListingDropped and NotificationsRegressed as counted, never silent."
    requirement: PROTO-09
    verification:
      - kind: static
        ref: "docs/04-MULTI-NODE.md ## Protocol bounds vs internal/app/repl/core.go:113-169 and the accessors at core.go:2570,2621,2628,2722"
        status: pass
    human_judgment: false
  - id: D4
    description: "The topology section cites the in-process shared-port three-adapter test as the real-socket proof, states the kernel-arbitrary unicast limitation, names the simulator and asymmetric-port two-node harness as the unicast owners, and records that the production binary has no reuse flag."
    requirement: PROTO-09
    verification:
      - kind: static
        ref: "docs/04-MULTI-NODE.md ## Three-node topology in tests vs internal/udprepl/three_node_test.go:53-73 and internal/udprepl/scenario_test.go:1-20"
        status: pass
    human_judgment: false
  - id: D5
    description: "OPERATIONS lists bad_sync_header in the discard-class list and documents --startup-gate-empty / SUPKV_STARTUP_GATE_EMPTY (default 60 s, empty-disk nodes only) without restructuring or adding a catalog."
    requirement: PROTO-09
    verification:
      - kind: command
        ref: "grep -q 'bad_sync_header' docs/OPERATIONS.md && grep -q 'startup-gate-empty' docs/OPERATIONS.md"
        status: pass
    human_judgment: false
  - id: D6
    description: "All five rows this phase closes (S-HLC-04, S-HLC-10, S-PROTO-18, S-ALG-ST-06, S-ALG-RC-05) read implemented with current code and test evidence; exactly five rows changed and no unrelated row was touched."
    requirement: PROTO-09
    verification:
      - kind: command
        ref: "set -o pipefail; grep -E '^\\| S-(HLC-04|HLC-10|PROTO-18|ALG-ST-06|ALG-RC-05) \\|' docs/SPEC-DEVIATIONS.md | awk '/implemented/{n++} END{exit !(n>=5)}'"
        status: pass
      - kind: command
        ref: "git diff -U0 docs/SPEC-DEVIATIONS.md | grep '^[-+]|' (five removed and five added rows, all the five declared IDs)"
        status: pass
    human_judgment: false

duration: 272s
completed: 2026-11-22
status: complete
---

# Phase 04 Plan 12: Multi-Node Artifact, Operations Handoff and Deviation Evidence Summary

**The phase's N-node contract is now an operator-facing document — traffic shape, timing, `wait_for_replication` at N > 2, the eight shipped bounds and four accepted residual windows, each tied to code or tests — with the operations handoff entries and the five closed deviation rows carrying current evidence.**

## Performance

- **Duration:** 272s (~4.5 min)
- **Started:** 2026-11-22T06:32:55Z
- **Completed:** 2026-11-22T06:37:27Z
- **Tasks:** 2
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- `docs/04-MULTI-NODE.md` now carries PROTO-09's artifact. The traffic-shape section records the periodic broadcast listing, the startup round, the per-chunk pacing and the one-shared-build property, then the first-seen/restart targeted listings and the N−1 + N−1 shape a joiner sees. The timing section states the one-RTT normal bound, the `2 × sync_interval + response` partition-heal bound, the no-`sync_interval`-term evicted-peer heal (`Hello` plus N / `sync_response_rate`) and the first-Hello join, each naming its test.
- The `wait_for_replication` section records the exact N > 2 contract from `wait.go`: `total` captured once before the wait, `acked=true` requiring every write-time peer, the WR-01 count clamp, and IN-05 (a mid-wait discovery joins the wait set but not `total`, so the report understates coverage — the safe direction), with Phase 6 APIS-01/DOCS-05 named as the owner of tightening and the final API wording.
- The protocol-bounds table records all eight shipped constants with rationale — `maxSyncStreams` 16, `maxSyncEntriesPerStream` 65 536, `maxSyncEntriesTotal` 131 072, `maxSyncChunks` 65 536, `maxListedKeysTotal` 131 072 (whole-peer-set LRU eviction), `maxListingDestinations` 32, `maxListingChunks` 65 536, `seenRunsCap` 8 — and names the four counters (`SyncStreamDropped`, `ListedKeysEvicted`, `ListingDropped`, `NotificationsRegressed`) plus the `bad_sync_header` class.
- The topology section cites `TestSmoke_threeNodeBroadcastConvergence` as the real-socket proof, states the kernel-arbitrary unicast limitation of a shared `SO_REUSEADDR` port, names the simulator and the asymmetric-port two-node harness as the unicast owners, and records that `WithAddressReuse` is test-only.
- The residual section labels four accepted/residual windows with owners: the mixed-cluster empty-listing gate window (D-01 literal reading), the listed-but-unarrived Delete (`NotFound`, 14.5(6) covers reads), the unbounded peer-row table (Phase 7) explicitly paired with the capped retained listed-key payload, and listing drift staying out of `MergeStats` (Phase 7 OBSV-05).
- `docs/OPERATIONS.md` gained `bad_sync_header` in the discard-class list and the `--startup-gate` / `--startup-gate-empty` gate description in `## Read semantics`; the stale claim that reads are never gated at startup was corrected to ADR 0005/D-01's closed-interface contract while documenting the empty-disk knob.
- `docs/SPEC-DEVIATIONS.md` flipped exactly S-HLC-04, S-HLC-10, S-PROTO-18, S-ALG-ST-06 and S-ALG-RC-05 to `implemented`, with rewritten shipped-behavior summaries and current evidence: `Store.Observe`, `handleDataSync`/`maxEntryVersion`, the accumulation/completeness path, the proto header fields, the gRPC gate in `server.go`/`watch.go`, and the tests `TestSync_observePerPart`, `TestSync_partialListingKeepsGateClosed`, `TestSync_chunkWireBudget`, `TestMirror_listingComplete`, `TestGate_readsAndWritesUnavailable`.

## Task Commits

1. **Task 1: Write docs/04-MULTI-NODE.md** - `fabdc94` (docs)
2. **Task 2: OPERATIONS handoff entries and deviation-registry evidence flips** - `47ecdcc` (docs)

**Plan metadata:** this SUMMARY/STATE/ROADMAP commit follows this file.

## Files Created/Modified

- `docs/04-MULTI-NODE.md` — new; the N-node traffic/timing/wait/bounds/topology/residual/handoff artifact (182 lines).
- `docs/OPERATIONS.md` — `bad_sync_header` added to the discard-class list; the `## Read semantics` gate paragraph now documents `--startup-gate` and `--startup-gate-empty`/`SUPKV_STARTUP_GATE_EMPTY` (60 s, empty-disk only) and the closed reads-and-writes contract.
- `docs/SPEC-DEVIATIONS.md` — five declared rows rewritten and flipped to `implemented` with current code/test evidence; all other rows byte-identical.

## Decisions Made

- **Claims carry evidence on the same line.** Every bound names its constant, every timing statement names the scenario, and every residual window names its owner or its acceptance — the artifact cannot quietly drift into an unbacked guarantee.
- **The startup round is named.** "One broadcast listing per `sync_interval`" alone would omit the startup listing right after the first Hello (S-ALG-RC-03), so the document states both and keeps the periodic claim exact.
- **The chunk budget's exception is named.** An entry that alone reaches `sync_chunk_bytes` ships as its own chunk; stating only "chunks under the budget" would overstate the guarantee.
- **The OPERATIONS correction rides the handoff edit.** The old sentence said reads are never gated by startup, contradicting ADR 0005/D-01 and the new gate paragraph; correcting it is the minimum needed for the knob's description to be true. Phase 9 still owns the final catalog wording.
- **Five rows only.** `S-HLC-04`, `S-HLC-10`, `S-PROTO-18`, `S-ALG-ST-06` and `S-ALG-RC-05` are this phase's closures; the surrounding rows were left byte-identical, and the line-number evidence uses the current code positions.
- **PROTO-09 remains unchecked in REQUIREMENTS.md.** `requirements.ready-ids` blocks it because 04-13's plan also owns it; the phase gate closes it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical correctness] OPERATIONS asserted reads are never gated at startup**
- **Found during:** Task 2, while locating the gate paragraph for the new knob
- **Issue:** `docs/OPERATIONS.md` said "A restarted node serves reads immediately (reads are never gated by startup)". ADR 0005/D-01 and `internal/grpcapi/server.go` gate reads and writes alike (`Get`, `List`, `Watch` all answer `Unavailable` while the gate is closed), so the sentence contradicted both the shipped code and the handoff text the task adds.
- **Fix:** Replaced the sentence with the closed-interface paragraph: reads and writes answer `Unavailable` until a complete listing arrives or the applicable gate elapses, health stays `NOT_SERVING`, and the data-node (`--startup-gate`, 1 s) versus empty-disk (`--startup-gate-empty`, 60 s) split is stated.
- **Files modified:** `docs/OPERATIONS.md`
- **Verification:** `grep -q 'bad_sync_header' docs/OPERATIONS.md && grep -q 'startup-gate-empty' docs/OPERATIONS.md`; the document now matches `internal/grpcapi/gate.go` and `server.go`.
- **Committed in:** `47ecdcc`

No other deviation — the plan executed as written.

## Issues Encountered

None. Both tasks' automated verifications passed on the first run, and no unrelated document line moved.

## TDD Gate Compliance

No task in this plan carries `tdd="true"`; both are documentation tasks with file-content verifications, so no RED/GREEN cycle applies.

## Known Stubs

None. No placeholder text, TODO/FIXME marker or unwired statement was introduced.

## Threat Flags

None — no new trust-boundary surface. T-4-35 is mitigated: every multi-node claim carries a code or test reference and each residual window is explicitly labelled with its owner. T-4-36 is mitigated: exactly the five declared deviation rows changed, each with current evidence paths, and no unrelated row moved. T-4-SC: no dependency was added or installed.

## Next Phase Readiness

- PROTO-09's documentation artifact, operations handoff and deviation evidence are in place; the real-socket, simulator and scenario tests they cite are green from 04-07/04-10/04-11.
- PROTO-09 remains unchecked in REQUIREMENTS.md until 04-13 (the phase gate) closes it; the coverage entries here are ready for that gate.
- 1 plan remains in Phase 4. Ready for `04-13-PLAN.md`.

---

*Phase: 04-protocol*
*Completed: 2026-11-22*

## Self-Check: PASSED

- Created file exists: `docs/04-MULTI-NODE.md` — FOUND.
- Commits exist and are ancestors of HEAD: `fabdc94`, `47ecdcc`.
- Measured plan commits: 2 from the ledger base `379eddf6`, matching the frontmatter `commits: 2`.
- No tracked file was deleted by either commit.
