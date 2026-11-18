---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 06
subsystem: replication-testing
tags: [crdt, replication, udp, real-socket-tests, failure-scenarios, discard-classification, test-only]

requires:
  - phase: 04-replication-and-deterministic-simulator
    provides: udprepl adapter + repl core discard classifier, EncodeDatagramForTest test-binary helpers, smoke-test real-socket precedent
  - phase: 06-hardening-failure-scenarios-observability-benchmarks
    provides: "06-01 hardening fixes (the exact egress-pacing/drain/mirror contracts these scenarios assert through) and 06-03 counter seams (Discards classes, adapter mirrors)"
provides:
  - internal/udprepl/scenario_test.go — the real-socket in-process half of TEST-04: asymmetric-topology helper (free-port picker + cross-broadcast capability probe + loud darwin skip per D-12), TestScenario_convergence (bidirectional writes + unicast Ack drain), TestScenario_discardClasses (bad CRC / proto mismatch / duplicate node_id, per-class exact)
  - EncodeDatagramForTestVersion — the exported test-binary-only explicit-proto-version crafting seam (correct CRC, production encode untouched)
  - Zero production-code changes
affects: [06-07 exec scenarios, 06-08 benchmarks, TEST-04 verification, milestone close]

actuals:
  tokens: 5500   # chars/4 over the realized diff (22003 diff chars across 2 files)
  tasks: 2
  commits: 2      # MEASURED: git rev-list --count 04ab852..HEAD

tech-stack:
  added: []
  patterns:
    - "per-case isolated converged pairs for parallel subtests that assert counter exactness — a shared adapter's baseline cannot survive concurrent increments (scenarioConvergedPair is the shared prologue)"
    - "capability-probe loud skip naming the probed address AND the platform reason (D-12) — the gate is the darwin story, never a build tag"
    - "adversarial crafting seams as exported test-binary-only helpers with production-faithful framing (correct CRC over the real protoBody) so the classifier branch under test is the only thing that fires"

key-files:
  created:
    - internal/udprepl/scenario_test.go
  modified:
    - internal/udprepl/repl_internal_test.go

key-decisions:
  - "Per-case isolated converged pairs in TestScenario_discardClasses: the vendored paralleltest/tparallel gates demand parallel subtests, and per-class exactness must not share a mutable Discards baseline across concurrent subtests — each case converges its own pair"
  - "EncodeDatagramForTestVersion added to repl_internal_test.go (the plan-anticipated extension): explicit envelope proto_version with the CRC computed exactly as production does, so the receiver's CRC gate passes and the core's version check is the branch under test; the impostor pair needs no new helper — EncodeDatagramForTest already parameterizes nodeID and runID"
  - "The drain assertion reads PendingCopy() emptiness: the 250ms drained-row retention window is inside the timer-derived 10x-hello-interval converge bound, so an observably empty mirror is the honest full-Ack signal on the asymmetric topology"
  - "The scenario capability probe is scenario-local (shaped like the two existing probes) because its skip message must name the darwin lo0 platform reason per D-12 — the wait_integration/mirror probes name their own suites"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "Asymmetric-topology helper (free-port picker, cross-broadcast capability probe, loud D-12 darwin skip) in scenario_test.go"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "go test -race ./internal/udprepl/ -run '^TestScenario_' — both scenario tests run through startScenarioPair's probe (Linux side exercised); stable across -count=3"
        status: pass
      - kind: other
        ref: "grep checks: zero 'go:build' lines in scenario_test.go; t.Skipf names the probed address and the darwin lo0 no-BROADCAST reason"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestScenario_convergence — two real adapters on the asymmetric pairing: mutual discovery, one write each way, both stores byte-equal, both PendingCopy mirrors drained (the unicast Ack path)"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "go test -race ./internal/udprepl/ -run '^TestScenario_convergence$' -v — PASS (0.31s), stable -count=3; smoke/shared-port suite untouched and green"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestScenario_discardClasses — bad CRC, proto-version mismatch and duplicate node_id (remembered-runID + differing-handle variant, config-error log naming node_id and both handles) each classify exactly once, others at pre-shot values, convergence intact"
    requirement: TEST-04
    verification:
      - kind: integration
        ref: "go test -race ./internal/udprepl/ -run '^TestScenario_discardClasses$' -v — 3/3 subtests PASS, stable -count=3; core TestDiscard_classes/totality/onDiscardSeam green unmodified"
        status: pass
    human_judgment: false
  - id: D4
    description: "EncodeDatagramForTestVersion crafting seam — exported, test-binary-only, production New and encodeDatagram unchanged"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "bad_proto_version subtest PASS through the real socket path; git diff over internal/udprepl/repl.go and internal/app/repl/core.go empty"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 06: Real-Socket Scenario Suite Summary

**TEST-04's real-socket in-process half shipped: two-node convergence proven on the empirically-probed asymmetric-port topology (bidirectional writes + the unicast Ack drain the shared-port smoke topology cannot carry), plus the three adversarial input classes — bad CRC, proto-version mismatch, duplicate node_id — each classified into exactly its own Discards counter over real UDP, with the loud capability-gated darwin skip (D-12) and zero production-code changes.**

## Performance

- **Duration:** 12min
- **Started:** 2026-11-15T23:10:50Z
- **Completed:** 2026-11-15T23:22:24Z (UTC)
- **Tasks:** 2/2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- **Asymmetric-topology helper** (`startScenarioPair` + `scenarioProbeBroadcast`): node A binds P1 broadcasting to `127.255.255.255:P2`, node B mirrored (research Pattern 5) — the only same-host pairing whose cross-unicast Acks reach the peer deterministically. The one-port capability probe skips loudly naming the probed address and the darwin lo0 no-BROADCAST reason (D-12); no build tags anywhere.
- **TestScenario_convergence** (the plan's tracer): mutual discovery from broadcast Hellos, one write per node, both stores serving the peer's record byte-equal under the exact version, and both `PendingCopy` mirrors drained to empty inside the timer-derived 10x-hello-interval bound — the unicast Ack path, proven end-to-end on real sockets. Tracer feedback gate re-ran the verify under auto mode: PASS before expansion.
- **TestScenario_discardClasses**: three parallel subtests, each on its own converged pair — bad_crc (body byte flipped inside the CRC-covered region), bad_proto_version (explicit-version datagram with a CORRECT CRC, via the new helper), duplicate_node_id (phase-4 pinned second variant: same node_id, remembered run_id equal, different source handles — config-error log asserted to name node_id and both handles). Every case: only the driven class increments, other four at pre-shot values, both stores' snapshots unchanged, mutual discovery intact.
- **EncodeDatagramForTestVersion** in `repl_internal_test.go`: the exported test-binary-only explicit-proto-version crafting seam riding the production `protoBody` mapping — the plan-anticipated extension; production `New` and `encodeDatagram` byte-untouched.

## Task Commits

1. **Task 1 (tracer): asymmetric-topology convergence scenario** — `3457109` (test) — tracer feedback gate passed under auto mode
2. **Task 2: adversarial-input discard-class scenarios** — `cb06fb5` (test)

**Plan metadata:** (this commit) `docs(06-06): complete real-socket scenario suite plan`

## Files Created/Modified

- `internal/udprepl/scenario_test.go` — NEW: topology helper + capability probe (D-12 loud skip), TestScenario_convergence, TestScenario_discardClasses, logCapture, per-class exactness and convergence-intact assertions
- `internal/udprepl/repl_internal_test.go` — + EncodeDatagramForTestVersion (explicit-proto-version crafting seam)

## Decisions Made

See key-decisions frontmatter. Highlights: per-case isolated converged pairs (paralleltest gate + exactness semantics), the version-parameterized crafting helper with production-faithful CRC, PendingCopy-emptiness as the drain signal, and the scenario-local probe whose skip names the D-12 platform reason.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] Parallel subtests required per-case isolated pairs (plan sketched one shared topology)**
- **Found during:** Task 2, first full lint gate
- **Issue:** The vendored strict config enables paralleltest/tparallel: subtests of a `t.Parallel` test must call `t.Parallel` (the repo-wide TestDiscard_classes precedent). The plan's shape — one converged pair shared by sequential subtests — cannot run parallel safely: one case's increment would leak into another case's exactness snapshot read concurrently off the same adapter.
- **Fix:** Each subtest converges its own asymmetric pair (stronger isolation: zero baseline coupling between cases); the shared convergence prologue extracted into `scenarioConvergedPair`, which TestScenario_convergence now reuses — its assertions (discovery, cross-visibility, Ack drain) are unchanged and were re-run green after the refactor.
- **Files modified:** internal/udprepl/scenario_test.go
- **Verification:** go test -race ./internal/udprepl/ -run '^TestScenario_' green, stable -count=3; go tool golangci-lint run ./... exit 0
- **Committed in:** cb06fb5

**2. [Rule 3 - Lint shape] golines wanted the probe's Skipf args one per line**
- **Found during:** Task 1, package lint gate (pre-commit)
- **Issue:** golines (max-len 200) flagged the joined `dst, scenarioProbeWait` argument line.
- **Fix:** One argument per line, matching the smoke test's Skipf shape.
- **Files modified:** internal/udprepl/scenario_test.go
- **Verification:** go tool golangci-lint run ./internal/udprepl/... exit 0
- **Committed in:** 3457109

---

**Total deviations:** 2 auto-fixed (2 Rule 3 lint/blocker shapes; no Rule 4 — nothing architectural)
**Impact on plan:** None on scope — every must-have truth delivered; the discard topology is per-case rather than shared, which strengthens the exactness guarantee the plan asked for.

## Issues Encountered

None beyond the deviations above. No auth gates (pure in-repo work).

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - every scenario asserts through real sockets and the real classifier; no test is hidden behind build tags.

## Threat Flags

None. T-6-12 (accepted): the scenario sockets bind ephemeral wildcard/loopback UDP ports for the test's lifetime only, send fixed small datagrams, and clean up via t.Cleanup.

## Next Phase Readiness

- The TEST-04 real-socket in-process scenarios are in place for phase verification: 06-07 (exec two-process scenarios) reuses the same asymmetric pairing via `--bind-addr`/`--broadcast-addr` flags; 06-08's burst benchmark can reuse `startScenarioPair`'s shape.
- The shared-port smoke test remains untouched as the own-echo assertion home; the darwin cells loud-skip through the capability gate (D-12), to be observed on the macos-15 CI cell.

## Self-Check: PASSED

- key-files.created exists on disk: internal/udprepl/scenario_test.go FOUND
- Commits reachable from HEAD: 3457109, cb06fb5 FOUND (merge-base --is-ancestor ok)
- evaluation-scope --plan 06-06 --commits-only: resolved, 2 commits
- All task acceptance criteria re-run: PASS (no shared-port two-node construction; probe skip names address+reason, zero build tags; convergence asserts stores+drain; smoke green unmodified; per-class exactness on all three subtests; dup case log assertion present; crafting helper exported in repl_internal_test.go with production diff empty; TestDiscard_ suites green unmodified; full-repo race + lint exit 0)
- Plan-level verification: go test -race ./internal/udprepl/ -run '^TestScenario_' -v green; full udprepl package green; go test -race ./... exit 0; go tool golangci-lint run ./... exit 0
- Windows ledger: no stubs, skipped tests, or unrun verifies to record

---
*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
