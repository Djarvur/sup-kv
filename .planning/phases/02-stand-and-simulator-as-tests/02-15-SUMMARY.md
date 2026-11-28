---
phase: 02-stand-and-simulator-as-tests
plan: 15
subsystem: testing
tags: [simnet, extension-tests, determinism-replay, delete-replication, restart-with-state, receive-budget, size-gate, dump-latency, d05, test-06, test-02]

# Dependency graph
requires:
  - phase: 02-03 (wave 2 dependency, same phase)
    provides: the additive simnet API this plan proves — Cluster.Delete, Cluster.ReplaceNodeWithState, Network.SetRecvBudget + FaultStats.RecvBudgetRefused, Policy.SizeOf/MaxDatagramBytes + FaultStats.OversizeRefused, Policy.AnnounceDelay
  - phase: 04-replication (v1.0 shipped)
    provides: the simnet harness (Scheduler/Policy/Network/Cluster), the repl.Core handlers the simulator drives, and the FaultStats/TraceDigest seams the tests read
provides:
  - internal/app/internal/simnet/simnet_ext_test.go — the six-test executable acceptance gate for plan 02-03's simulator extensions (D-05)
  - the same-seed TraceDigest replay gate with every extension active (budget + size gate + announcement latency + deletion + restart-with-state): the machine-checked proof of the determinism contract every simnet API doc comment claims (T-2-05)
  - the simulator level of TEST-02's four-level deletion proof: Cluster.Delete replicating a tombstone end to end, with the ErrNotFound no-version-burn paths pinned
  - test-layer real-encoder size measurement (replproto protoBody + envelope mirror) injected through Policy.SizeOf, with simnet.go's import block still stdlib + internal/app + internal/port
affects: [02-16/02-17 (remaining phase-2 plans), the D-06 chaos corpus and scenario work riding deletions and restart-with-state, phase 3+ restart/delete scenarios, /gsd:verify-work's evaluation of TEST-06 and TEST-02]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 4884    # chars/4 over the realized diff (19537-byte file, 451 insertions)
  tasks: 1
  commits: 1      # MEASURED from the plan-start ledger at SUMMARY write (#3968): git rev-list --count acfc0d1..73f0e28
  plan_head_before: acfc0d1bfa99399c471cc6eee9e00681e51a4432
  plan_head_after: 73f0e28b7e238f4169381eb0a1347f965d19e92a

# Tech tracking
tech-stack:
  added: none — the test file imports stdlib, google.golang.org/protobuf/proto (already a module dependency of the adapter) and own-module packages; zero new module dependencies (zero-install phase per the Package Legitimacy Audit)
  patterns:
    - test-layer encoder injection: Policy.SizeOf is implemented in the _test.go file by mirroring the adapter's port-message mapping and envelope shape (protoBody + EncodeDatagramForTest's envelope), so the size gate measures the real wire size while simnet's non-test import block stays depguard-clean
    - fixed-instant timing assertions: zero-rng profiles (fixed one-way delay, fixed announcement latency) only where an exact scheduler instant is asserted; every other test keeps the seeded draw so replay semantics stay intact
    - boundary assertions with teeth: the same-seed replay test asserts the extension counters fired (OversizeRefused > 0, RecvBudgetRefused > 0) so a vacuous "extensions off" replay cannot pass
    - control-assertion contrast: the restart-with-state test runs a sibling cluster through the plain ReplaceNode to document empty-boot-then-reconcile against the seeded boot

key-files:
  created:
    - internal/app/internal/simnet/simnet_ext_test.go
  modified: []

key-decisions:
  - "The extension tests live in package simnet_test (external), not the plan's suggested internal package simnet: the internal form would need a //nolint:testpackage suppression (the lint config's skip-regexp covers only *_internal_test.go and the artifact path is fixed by the must_haves), while the external form imports internal/udprepl/replproto freely (the depguard $test exemption applies to external test files too) and reuses the existing simnet_test.go helpers (simAt, simRNG, simDelay, recordsMapsEqual) instead of duplicating them"
  - "The real encoder is mirrored, not imported: udprepl's EncodeDatagramForTest lives in that package's _test.go file and is therefore compiled only into udprepl's own test binary, so the test layer cannot call it; simWireBody reproduces the adapter's port-message mapping and simWireSize the full envelope (proto_version, node_id, run_id, crc32, body), giving the same byte length the adapter would send"
  - "The size-gate limit is derived in-test from two real measurements (small vs 2000-byte DataReplication, limit = midpoint) and asserted to separate them, so the boundary is a measured fact rather than a magic ceiling; the message length is fixed by the 16-byte version field and the data length, and the checksum-field delta is a few bytes against a ~1000-byte margin"
  - "The determinism replay uses a bounded receive budget (3 deliveries) deliberately: the scenario must exercise refusals, not converge, and both runs then replay the same refusal cascade byte for byte — the same-seed equality is the property under test, convergence is 02-06's"
  - "Cascade counts are asserted exactly where the fan-out is exact: one oversize broadcast in a 2-node cluster is two refused scheduleDelivery calls (self echo + peer), hence OversizeRefused == 2; a receive budget of 1 with three writes yields RecvBudgetRefused == 2 and UnackedStats().Count == 2"

patterns-established:
  - "Test-layer wire measurement: mirror the adapter's protoBody map and envelope in the _test.go file rather than exporting a production encoder seam or importing the adapter's test binary — the non-test harness keeps its import boundary by construction"
  - "Determinism-replay gate with activation teeth: assert the extension counters are non-zero before comparing digests, so the gate proves determinism WITH the extensions, never determinism of a zero-value policy"
  - "Exact-instant scheduling assertions: choose a fixed delay + fixed announcement latency, compute the arrival as an arithmetic expression, and assert absent at arrival-1ms, present at arrival, with the trace line carrying the exact UnixMilli — never a 'wait long enough' check"

requirements-completed: [TEST-06, TEST-02]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "TestSim_Delete_replicatesTombstone: 2-node cluster, Write then Delete on node 0 — the returned tombstone carries Deleted=true and a version dominating the write; the peer's store ends with the same tombstone version and no data; a missing key and the already-tombstoned key both return storage.ErrNotFound with a zero record and no write-log entry; interleaved failed deletes leave the clock's per-millisecond counter advancing by exactly one across two writes (no version burned). Simulator level of TEST-02's deletion proof."
    requirement: TEST-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_Delete_replicatesTombstone$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestSim_ReplaceNodeWithState_preservesState: two writes converge, then ReplaceNodeWithState boots the writer onto its own record snapshot — the replacement store immediately equals the seeded records with their original versions, and after the restart discovery cycle both nodes' stores are still equal. The control cluster runs the same writes through plain ReplaceNode: its store is empty immediately after the call (no preservation) and re-derives the peer's state only through reconciliation, documenting the difference the seeded seam exists for."
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_ReplaceNodeWithState_preservesState$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestSim_recvBudget_refusesAndCounts: SetRecvBudget(node 2, 1) after discovery, then three writes — exactly one delivery is accepted (its record present on the peer), the next two are refused at fire time and counted (RecvBudgetRefused == 2), the refused records are absent from the peer store, and the sender's pending set holds exactly the two unconfirmed entries (UnackedStats().Count == 2)."
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_recvBudget_refusesAndCounts$'"
        status: pass
    human_judgment: false
  - id: D4
    description: "TestSim_sizeRefusal_byRealEncoder: Policy.SizeOf is the real replproto encode measured at the test layer; the limit is the measured midpoint between an ~11-byte and a 2000-byte DataReplication and the test asserts it separates them. The under-limit write arrives; the over-limit write never arrives; the oversize broadcast is refused once per recipient (OversizeRefused == 2 for a 2-node fan-out including the self echo); startup Hello/DataSync survive the gate, so discovery still completes."
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_sizeRefusal_byRealEncoder$'"
        status: pass
    human_judgment: false
  - id: D5
    description: "TestSim_announceDelay_virtualDumpLatency: with a 1ms drawn delay and a 20ms announcement latency, a write is absent from the peer at write+20ms (and the trace logs no peer DataReplication delivery), present at write+21ms, and the trace line carries exactly the write+21ms UnixMilli; the peer's Ack then arrives after only the 1ms drawn delay (trace line n2→n1 port.Ack at 22ms) and drains the sender's pending entry — proving AnnounceDelay is scoped to DataReplication."
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_announceDelay_virtualDumpLatency$'"
        status: pass
    human_judgment: false
  - id: D6
    description: "TestSim_determinism_withExtensions: runSimExtScenario replays the same 2-node scenario twice under one seed with every extension active (drops/duplicates/drawn delays, size gate at 1200 bytes with a 4000-byte write, 20ms announcement latency, receive budget 3, a deletion, a restart-with-preserved-state, a post-restart write); digests are non-empty and byte-identical, fault counters identical, and every node's final store identical. Activation teeth: OversizeRefused > 0 and RecvBudgetRefused > 0 are asserted, so a zero-policy replay cannot pass. Also asserts every node store is non-empty (the seeded restart ran)."
    requirement: TEST-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_determinism_withExtensions$'"
        status: pass
      - kind: other
        ref: "go tool golangci-lint run ./internal/app/... → exit 0 (depguard boundary intact: simnet.go imports stdlib + internal/app/* + internal/port only)"
        status: pass
    human_judgment: false

# Metrics
duration: 10min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 15: Simulator Extension Tests and Determinism Replay Summary

**Six executable extension tests over plan 02-03's additive simnet API — tombstone replication through `Cluster.Delete`, restart-with-preserved-state against a plain-`ReplaceNode` control, receive-budget refusals, the real-encoder size gate, the virtual dump latency at exact scheduler instants — plus the same-seed `TraceDigest` replay gate that proves the determinism contract holds with all extensions active and their counters demonstrably firing.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-11-21T11:49:00Z (approx., first plan read)
- **Completed:** 2026-11-21T11:58:49Z
- **Tasks:** 1
- **Files changed:** 1 (`internal/app/internal/simnet/simnet_ext_test.go`, +451)

## Accomplishments

- **`TestSim_determinism_withExtensions` — the executable determinism-contract gate (T-2-05).** One fixed scenario runs twice under seed 43 with every extension active: drops/duplicates drawn from the single seeded rng, the 1200-byte size gate fed by the real encoder, a 20ms announcement latency, a 3-delivery receive budget, a deletion, a restart-with-preserved-state and a post-restart write. The two runs produce byte-identical `TraceDigest`, identical `FaultStats`, and identical final stores per node. Activation teeth (`OversizeRefused > 0`, `RecvBudgetRefused > 0`, non-empty stores) make a "extensions off" replay impossible to pass.
- **`TestSim_Delete_replicatesTombstone` — the simulator level of TEST-02's four-level deletion proof.** `Cluster.Delete` returns a dominating tombstone, the peer's store converges on exactly that version with `Deleted=true` and no data, and both no-op paths (missing key, already-tombstoned key) return `storage.ErrNotFound` with the zero record and no write-log entry. The no-version-burn property is pinned observably: two writes at one virtual instant with failed deletes interleaved stamp consecutive HLC counters.
- **`TestSim_ReplaceNodeWithState_preservesState` — preservation vs re-derivation.** The seeded replacement holds both records with their original versions immediately and the cluster stays converged afterwards; a sibling control cluster through plain `ReplaceNode` boots empty and re-derives the state over the network, documenting exactly the difference the seeded sibling exists for.
- **`TestSim_recvBudget_refusesAndCounts`** — budget 1 accepts exactly one post-discovery delivery, refuses and counts the next two, leaves the refused records absent from the peer store, and leaves exactly the two unconfirmed entries in the sender's pending set.
- **`TestSim_sizeRefusal_byRealEncoder`** — the size gate measures with the real protobuf encoder (adapter mapping mirrored at the test boundary), the limit is derived from two real measurements and asserted to separate them, the under-limit record arrives, the over-limit record is refused with `OversizeRefused == 2` (one per broadcast recipient), and startup discovery survives the gate.
- **`TestSim_announceDelay_virtualDumpLatency`** — the record is invisible at write+20ms, visible at write+21ms, the trace logs the peer `DataReplication` at exactly that virtual millisecond, and the subsequent `Ack` waits only the drawn 1ms delay — `AnnounceDelay` is scoped to announcements.
- **Depguard boundary kept by construction.** The encoder lives only in the `_test.go` file; `simnet.go` was not modified at all (its import block stays stdlib + `internal/app/*` + `internal/port`), and no `//nolint` suppressions were added.

## Task Commits

1. **Task 1: extension tests incl. determinism replay** - `73f0e28` (test)

_Measured from the plan-start ledger `acfc0d1` (`git rev-list --count acfc0d1..HEAD` at SUMMARY write): 1 task commit._

## Files Created/Modified

- `internal/app/internal/simnet/simnet_ext_test.go` — created (+451): the six named extension tests, the real-encoder measurement helpers (`simWireSize`, `simWireBody`), the deterministic run-id helper, the fixed-delay profile, and the replay scenario helper.

## Decisions Made

- **External test package, not the plan's internal package.** `package simnet_test` imports `internal/udprepl/replproto` freely (the depguard `$test` exemption applies to external test files too) and reuses the existing `simnet_test.go` helpers; the internal-package form would have required a `//nolint:testpackage` suppression because the lint config's skip-regexp covers only `*_internal_test.go` while the plan's must_haves fix the artifact filename. See Deviations 1.
- **The encoder is mirrored, not imported.** `udprepl.EncodeDatagramForTest` is defined in `repl_internal_test.go`, so it is compiled only into `udprepl`'s test binary and cannot be called from another package. `simWireBody`/`simWireSize` reproduce the adapter's `protoBody` mapping and full envelope (proto_version, node_id, run_id, crc32, body), yielding the same length the adapter would send — exactly the "the test layer mirrors the real encoder" shape the plan's read_first describes.
- **Measured, in-test size limit.** The gate's ceiling is the midpoint of two `simWireSize` measurements (small vs 2000-byte `DataReplication`), asserted to fall strictly between them — a measured boundary, not a magic number. The few-byte version/checksum delta between the synthetic and the stamped message is far inside the ~1000-byte margin.
- **Bounded budget in the replay scenario.** Budget 3 exhausts during the 1.45s run, so the replay includes a refusal cascade. Determinism, not convergence, is this plan's property; a converging scenario would have left the budget path untested.
- **Exact-instant assertions use zero-randomness profiles.** Tests 1–5 use a fixed 1ms one-way delay (and a fixed 20ms announcement latency in test 5) so the asserted instants are arithmetic; the rng-consuming `simDelay` is reserved for the replay scenario where only seed-reproducibility matters.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Test file uses the external `simnet_test` package**
- **Found during:** Task 1, before implementation (design decision driven by the lint gate)
- **Issue:** the plan's action specifies "package simnet, internal test". The repo's `testpackage` linter skips only `*_internal_test.go`, and the plan's must_haves fix the artifact name `simnet_ext_test.go`, so an internal-package file would need an added `//nolint:testpackage` suppression — against the project convention that every suppression carries a burden of proof when an unsuppressed layout exists.
- **Fix:** wrote the file in `package simnet_test` (the existing `simnet_test.go` package). The reason the plan preferred the internal package — free import of `internal/udprepl/replproto` — is satisfied by the external package too (depguard's `$test` exemption covers external test files), and the shared helpers (`simAt`, `simRNG`, `simDelay`, `recordsMapsEqual`) are reused rather than duplicated.
- **Files modified:** `internal/app/internal/simnet/simnet_ext_test.go`
- **Verification:** `go tool golangci-lint run ./internal/app/...` → exit 0 with no suppression; `go test -race -count=1 ./internal/app/internal/simnet/` → ok.
- **Commit:** `73f0e28`

**Total deviations:** 1
**Impact:** package-clause only; every test, assertion, and the injected-encoder shape landed as the plan specified, and `simnet.go` needed no API-surface correction.

## Issues Encountered

- One first-run lint finding in the new file itself (`noinlineerr` on an inline `if err := …; err != nil` in the version-burn probe) — rewritten to a plain assignment; no behavior change. Tests passed on the first execution.
- `make lint` and `make test` both green; simnet package coverage rose from 59.6% (02-03 baseline) to 86.5% with the extension paths exercised.
- Pre-existing untracked directory `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261119T193227Z/` was left untouched (not this plan's artifact).

## Known Stubs

None. Every test asserts real replicated state, real counters and real trace lines; nothing is a placeholder or a skipped branch. No test carries `t.Skip`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 02-03's extensions now have their executable acceptance gate, and the determinism contract's claims in every simnet doc comment are machine-checked with the extensions active.
- TEST-06 is complete from this plan's side (the determinism/ordering probes now have their executable proof); TEST-02's simulator level is landed here, with the codec level (02-07's `TestWire_deleteRoundTrip`) and the two-process levels (02-07's stand tests) completing the four-level deletion proof.
- Remaining phase-2 plans (02-16, 02-17) can consume `Cluster.Delete`, `ReplaceNodeWithState` and the fault hooks knowing they replay byte-identically under one seed.
- No blockers. `simnet.go`'s import block remains stdlib + `internal/app` + `internal/port`; no lint suppressions were added.

## Self-Check: PASSED

Re-run on the committed tree (`73f0e28`):

- Key file exists: `internal/app/internal/simnet/simnet_ext_test.go` (19537 bytes) — FOUND.
- Commit exists: `73f0e28` is an ancestor of HEAD; `git rev-list --count acfc0d1..HEAD` → 1.
- `gsd-tools check evaluation-scope --plan 02-15 --commits-only --raw` → resolved, no missing/outside/unreachable files, exit 0 (not 69).
- Plan verify 1: `go test -race -count=1 ./internal/app/internal/simnet/ -run '^TestSim_'` → ok (all six tests run, none skipped).
- Plan verify 2: `go tool golangci-lint run ./internal/app/...` → exit 0 (depguard boundary intact; `grep -n replproto simnet.go` → line 147, doc comment only).
- Plan verification: `go test -race -count=1 ./internal/app/internal/simnet/` → ok (existing suites unchanged).
- `make lint` → exit 0; `make test` → exit 0, all 14 packages ok (simnet 86.5%).
- Success criterion 1: six tests green under -race, none skipped — PASS.
- Success criterion 2: same seed → identical `TraceDigest`, identical `FaultStats`, identical final stores with all extensions active, and the counters prove the extensions fired — PASS.
- Success criterion 3: real-encoder `SizeOf` injected from the test layer; `simnet.go` never imports replproto — PASS.
- Success criterion 4: `simnet.go` was not modified (no API-surface fix needed), so the depguard boundary is untouched — PASS.

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
