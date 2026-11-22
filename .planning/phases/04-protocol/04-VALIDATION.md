---
phase: "4"
slug: "protocol"
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-22"
validated: "2026-11-22"
---

# Phase 4 — Validation Strategy

> Per-phase validation contract. Audited after execution: every task in the 13 plans
> carries an automated verification that ran green on the committed tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` + `go test -race` |
| **Config file** | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `race-soak`) |
| **Quick run command** | `go test -race -count=1 ./internal/app/repl/ ./internal/udprepl/ ./internal/grpcapi/` |
| **Full suite command** | `make lint && make test` (+ `SUPKV_RED_TESTS=1 bash scripts/red-runner`) |
| **Estimated runtime** | ~120–200 s full suite; ~5–30 s per task command |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (the map below)
- **After every plan wave:** `go build ./... && make test`
- **Before phase verification:** full suite green on the committed tree
- **Max feedback latency:** ~200 s (full suite); per-task commands ≤ 30 s

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | PROTO-01, PROTO-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/udprepl/ -v -run '^TestSync_chunkWireBudget$' \| grep -q -- '--- PASS: TestSync_chunkWireBudget'` | ✅ | ✅ green |
| 04-01-02 | 01 | 1 | PROTO-01, PROTO-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/repl/ -v -run '^TestDiscard_badSyncHeader$' \| grep -q -- '--- PASS: TestDiscard_badSyncHeader'` | ✅ | ✅ green |
| 04-01-03 | 01 | 1 | PROTO-01, PROTO-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/udprepl/ -v -run '^TestDecode_negativeTable$' \| grep -q -- '--- PASS: TestDecode_negativeTable'` | ✅ | ✅ green |
| 04-02-01 | 02 | 2 | PROTO-01 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/grpcapi/ -v -run '^TestGate_readsAndWritesUnavailable$\|^TestGate_emptyKeyStillUnavailable$\|^TestGate_openRestoresReads$' \| awk '/--- PASS: TestGate_/{n++} END{exit !(n>=3)}'` | ✅ | ✅ green |
| 04-02-02 | 02 | 2 | PROTO-01 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/grpcapi/ -v -run '^TestServer_(getListedNotArrived\|getListedTombstoneAndAbsent\|getListedLive\|createIfAbsentListed\|gateOpenListedRace)$' \| awk '/--- PASS: TestServer_/{n++} END{exit !(n>=5)}'` | ✅ | ✅ green |
| 04-03-01 | 03 | 2 | PROTO-01, PROTO-02 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/storage/ -v -run '^TestStore_(observeAdvancesClock\|observeMonotone\|observeReturnsDrift\|len\|generationContentOnly)$' \| awk '/--- PASS: TestStore_/{n++} END{exit !(n>=5)}'` | ✅ | ✅ green |
| 04-03-02 | 03 | 2 | PROTO-01, PROTO-02 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestSync_observe' \| awk '/--- PASS: TestSync_observe/{n++} END{exit !(n>=3)}'` | ✅ | ✅ green |
| 04-04-01 | 04 | 3 | PROTO-02 | — | per plan threat model | automated | `go build ./... && go vet ./...` | ✅ | ✅ green |
| 04-04-02 | 04 | 3 | PROTO-02 | — | per plan threat model | automated | `go test -race -count=1 ./cmd/sup-kv/ -v -run '^TestMain_startupGateEmptySelection$' \| grep -q -- '--- PASS: TestMain_startupGateEmptySelection'` | ✅ | ✅ green |
| 04-05-01 | 05 | 3 | PROTO-03 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestEgress_identityOnly$\|^TestEgress_noValueRetention$' \| awk '/--- PASS: TestEgress_/{n++} END{exit !(n>=2)}'` | ✅ | ✅ green |
| 04-05-02 | 05 | 3 | PROTO-03 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestReplication_(staleAnsweredNotAcked\|equalVersionAcked\|newerApplied\|absentApplies)$' \| awk '/--- PASS: TestReplication_/{n++} END{exit !(n>=4)}'` | ✅ | ✅ green |
| 04-06-01 | 06 | 3 | PROTO-08 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/persist/ -v -run '^TestEngine_restore(ReturnsReplayed\|NoOpReturnsEmpty\|FailedFlushReturnsReplayed\|SkipsReplay)$' \| awk '/--- PASS: TestEngine_restore/{n++} END{exit !(n>=4)}'` | ✅ | ✅ green |
| 04-06-02 | 06 | 3 | PROTO-08 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/grpcapi/ -v -run '^TestServer_(dumpFailed(Write\|Delete\|CreateIfAbsent)Notifies\|closedWriteDoesNotNotify)$' \| awk '/--- PASS: TestServer_(dumpFailed\|closedWrite)/{n++} END{exit !(n>=4)}'` | ✅ | ✅ green |
| 04-07-01 | 07 | 3 | PROTO-09 | — | per plan threat model | automated | `go test -race -count=1 ./internal/udprepl/ -v -run '^TestSmoke_threeNodeBroadcastConvergence$' \| grep -Eq -- '--- (PASS\|SKIP): TestSmoke_threeNodeBroadcastConvergence'` | ✅ | ✅ green |
| 04-08-01 | 08 | 4 | PROTO-04 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestListing_(hostileBounded\|sharedBuild\|generationRebuild\|rotationPreservesIndices\|restartDropsOldRunFIFO)$' \| awk '/--- PASS: TestListing_/{n++} END{exit !(n>=5)}'` | ✅ | ✅ green |
| 04-08-02 | 08 | 4 | PROTO-04 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestRestart_(firstHelloExactlyOneListing\|repeatHelloNoListing\|runIDChangeTriggersFullSync\|returningAfterEvictionGetsListing\|firstHelloStillFlushesPending\|helloDoesNotLeakListingsToStranger\|generationInvalidatesCachedListing)$' \| awk '/--- PASS: TestRestart_/{n++} END{exit !(n>=7)}'` | ✅ | ✅ green |
| 04-08-03 | 08 | 4 | PROTO-04 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/repl/` | ✅ | ✅ green |
| 04-09-01 | 09 | 5 | PROTO-05, PROTO-06 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestNotify_(firstNotificationProceeds\|equalTrackedIgnored\|regressedIgnored\|newerBroadcasts\|absentKeyBroadcasts)$' \| awk '/--- PASS: TestNotify_/{n++} END{exit !(n>=5)}'` | ✅ | ✅ green |
| 04-09-02 | 09 | 5 | PROTO-05, PROTO-06 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestPeer_(alternatingRunIDsIsConfigError\|alternatingWarnRateLimitedPerPeer\|seenRingBounded\|repeatSameRunStillRefresh)$' \| awk '/--- PASS: TestPeer_/{n++} END{exit !(n>=4)}'` | ✅ | ✅ green |
| 04-10-01 | 10 | 6 | PROTO-09, PROTO-01 | — | per plan threat model | automated | `set -o pipefail; go test -race -count=1 ./internal/app/repl/ -v -run '^TestScenario_(threeNode\|joinerWaitsForCompleteListing\|f14ReplacedNode)$' \| awk '/--- PASS: TestScenario_/{n++} END{exit !(n>=3)}'` | ✅ | ✅ green |
| 04-10-02 | 10 | 6 | PROTO-09, PROTO-01 | — | per plan threat model | automated | `grep -c '"14\.9".*"TestScenario_threeNode"' internal/app/repl/decision_coverage_test.go && ! grep -q '"14\.9".*"TestChaos_convergence"' internal/app/repl/decision_coverage_test.go # [HEAD=0 EXPECT>=1]` | ✅ | ✅ green |
| 04-11-01 | 11 | 7 | PROTO-01, PROTO-03, PROTO-04, PROTO-07 | — | per plan threat model | automated | `go build ./... && go vet ./...` | ✅ | ✅ green |
| 04-11-02 | 11 | 7 | PROTO-01, PROTO-03, PROTO-04, PROTO-07 | — | per plan threat model | automated | `go test -race -count=1 ./...` | ✅ | ✅ green |
| 04-12-01 | 12 | 7 | PROTO-09 | — | per plan threat model | automated | `for s in '## Traffic shape' '## Timing' '## wait_for_replication at N > 2' '## Protocol bounds' '## Three-node topology in tests' '## Residual windows' '## Handoffs'; do grep -qF -- "$s" docs/04-MULTI-NODE.md \|\| exit 1; done` | ✅ | ✅ green |
| 04-12-02 | 12 | 7 | PROTO-09 | — | per plan threat model | automated | `grep -q 'bad_sync_header' docs/OPERATIONS.md && grep -q 'startup-gate-empty' docs/OPERATIONS.md` | ✅ | ✅ green |
| 04-13-01 | 13 | 8 | PROTO-01, PROTO-02, PROTO-03, PROTO-04, PROTO-05, PROTO-06, PROTO-07, PROTO-08, PROTO-09 | — | per plan threat model | automated | `grep -qE '^TestRed_R03_gateClosedUntilFullList[[:space:]]+green$' scripts/red-expectations.tsv && grep -qE '^TestRed_R03_f14ReplacedNode[[:space:]]+green$' scripts/red-expectations.tsv && grep -qE '^TestRed_P01_valueLimit[[:space:]]+red$' scripts/red-expectations.tsv && grep -qE '^TestRed_P01_valueLimitEngine[[:space:]]+red$' scripts/red-expectations.tsv` | ✅ | ✅ green |
| 04-13-02 | 13 | 8 | PROTO-01, PROTO-02, PROTO-03, PROTO-04, PROTO-05, PROTO-06, PROTO-07, PROTO-08, PROTO-09 | — | per plan threat model | automated | `grep -qE '^\\| CR-01 \\| critical \\| fixed \\|.*[0-9a-f]{7,}' .planning/phases/03-transport/03-REVIEW-DISPOSITION.md && grep -qE '^\\| CR-02 \\| critical \\| fixed \\|.*[0-9a-f]{7,}' .planning/phases/03-transport/03-REVIEW-DISPOSITION.md` | ✅ | ✅ green |
| 04-13-03 | 13 | 8 | PROTO-01, PROTO-02, PROTO-03, PROTO-04, PROTO-05, PROTO-06, PROTO-07, PROTO-08, PROTO-09 | — | per plan threat model | automated | `make lint && make test` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements — the stand, simulator, red runner and
strict lint gate all predate Phase 4 (Phase 2 delivered them). No Wave 0 scaffolding was needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification. The three-node topology and F14 value-preservation
consequences are asserted by real-socket tests (`internal/udprepl/three_node_test.go`) and simulator
scenarios (`TestScenario_threeNode`, `TestScenario_f14ReplacedNode`); the docs walkthrough runs its
marked blocks verbatim in `TestExamplesDoc`.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 200 s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-11-22 (audited after 13/13 plans; 28/28 tasks automated, 0 gaps)

## Validation Audit 2026-11-22

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
