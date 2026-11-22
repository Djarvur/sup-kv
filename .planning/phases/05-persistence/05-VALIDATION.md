---
phase: "5"
slug: "persistence"
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-22"
validated: "2026-11-22"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract. Audited after execution: every task in the 12 plans
> carries an automated verification that ran green on the committed tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` + `go test -race` |
| **Config file** | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `race-soak`) |
| **Quick run command** | `go test -race -count=1 ./internal/app/persist/ ./internal/persistfile/ ./internal/app/storage/` |
| **Full suite command** | `make lint && make test` (+ `SUPKV_RED_TESTS=1 bash scripts/red-runner`) |
| **Estimated runtime** | ~200–300 s full suite; per-task commands ≤ 60 s |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (the map below)
- **After every plan wave:** `go build ./... && make test`
- **Before phase verification:** full suite green on the committed tree
- **Max feedback latency:** ~300 s (full suite); per-task commands ≤ 60 s

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | PERSIST-01, PERSIST-10 | — | per plan threat model | automated | `go build ./... && go vet ./...` | ✅ | ✅ green |
| 05-01-02 | 01 | 1 | PERSIST-01, PERSIST-10 | — | per plan threat model | automated | `go vet ./...` | ✅ | ✅ green |
| 05-01-03 | 01 | 1 | PERSIST-01, PERSIST-10 | — | per plan threat model | automated | `go test -race -count=1 ./cmd/sup-kv/ -v -run '^TestMain_valueMaxNonPositive$' \| grep -q -- '--- PASS: TestMain_valueMaxNonPositive'` | ✅ | ✅ green |
| 05-02-01 | 02 | 2 | PERSIST-03, PERSIST-10 | — | per plan threat model | automated | `go build ./...` | ✅ | ✅ green |
| 05-02-02 | 02 | 2 | PERSIST-03, PERSIST-10 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -run '^TestCodec_'` | ✅ | ✅ green |
| 05-02-03 | 02 | 2 | PERSIST-03, PERSIST-10 | — | per plan threat model | automated | `go vet ./...` | ✅ | ✅ green |
| 05-03-01 | 03 | 3 | PERSIST-07 | — | per plan threat model | automated | `go build ./...` | ✅ | ✅ green |
| 05-03-02 | 03 | 3 | PERSIST-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/storage/ -run 'TestMerge_\|TestStore_\|TestProp_'` | ✅ | ✅ green |
| 05-03-03 | 03 | 3 | PERSIST-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/repl/ -v -run '^TestReplication_equalVersionSameSeedTrace$' \| grep -q -- '--- PASS: TestReplication_equalVersionSameSeedTrace'` | ✅ | ✅ green |
| 05-04-01 | 04 | 4 | PERSIST-06, PERSIST-11 | — | per plan threat model | automated | `go build ./... && go vet ./...` | ✅ | ✅ green |
| 05-04-02 | 04 | 4 | PERSIST-06, PERSIST-11 | — | per plan threat model | automated | `go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_(saveOperationOrder\|failureAtEachStep)$\|^TestFileStore_saveFailureHygiene$'` | ✅ | ✅ green |
| 05-04-03 | 04 | 4 | PERSIST-06, PERSIST-11 | — | per plan threat model | automated | `go build ./... && go vet ./...` | ✅ | ✅ green |
| 05-05-01 | 05 | 5 | PERSIST-02, PERSIST-09 | — | per plan threat model | automated | `go test -race -count=1 ./internal/persistfile/ -v -run '^TestFileStore_loadBackupPreservesMain$' \| grep -q -- '--- PASS: TestFileStore_loadBackupPreservesMain'` | ✅ | ✅ green |
| 05-05-02 | 05 | 5 | PERSIST-02, PERSIST-09 | — | per plan threat model | automated | `go test -race -count=1 ./internal/persistfile/ -v -run '^TestFileStore_load' \| grep -q -- '--- PASS: TestFileStore_load'` | ✅ | ✅ green |
| 05-05-03 | 05 | 5 | PERSIST-02, PERSIST-09 | — | per plan threat model | automated | `go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_saveOperationOrder$\|^TestEnsureDir_'` | ✅ | ✅ green |
| 05-06-01 | 06 | 6 | PERSIST-02, PERSIST-04, PERSIST-10 | — | per plan threat model | automated | `go build ./... && go vet ./...` | ✅ | ✅ green |
| 05-06-02 | 06 | 6 | PERSIST-02, PERSIST-04, PERSIST-10 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -v -run '^TestEngine_(loadStatsMapsResult\|startDirty\|markDirtyNudge\|skipEvidenceWindow)$' \| grep -q -- '--- PASS: TestEngine_'` | ✅ | ✅ green |
| 05-06-03 | 06 | 6 | PERSIST-02, PERSIST-04, PERSIST-10 | — | per plan threat model | automated | `go test -race -count=1 ./internal/metrics/ -v -run '^TestMetrics_' \| grep -q -- '--- PASS: TestMetrics_'` | ✅ | ✅ green |
| 05-07-01 | 07 | 7 | PERSIST-05, PERSIST-12 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -v -run '^TestEngine_(injectedClock\|confirmDebounceTiming)$' \| grep -q -- '--- PASS: TestEngine_'` | ✅ | ✅ green |
| 05-07-02 | 07 | 7 | PERSIST-05, PERSIST-12 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -v -run '^TestEngine_(backoffLadder\|dumpFailureClasses)$' \| grep -q -- '--- PASS: TestEngine_'` | ✅ | ✅ green |
| 05-07-03 | 07 | 7 | PERSIST-05, PERSIST-12 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/repl/ -v -run '^TestScenario_dumpNotify$' \| grep -q -- '--- PASS: TestScenario_dumpNotify'` | ✅ | ✅ green |
| 05-08-01 | 08 | 8 | PERSIST-08 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -v -run '^TestEngine_(confirmationTimeout\|timeoutThenDumpSuccess\|waiterRemoved)$' \| grep -q -- '--- PASS: TestEngine_'` | ✅ | ✅ green |
| 05-08-02 | 08 | 8 | PERSIST-08 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -v -run '^Test(Flush\|Snapshot\|Restore)_timeout' \| grep -q -- '--- PASS: Test'` | ✅ | ✅ green |
| 05-08-03 | 08 | 8 | PERSIST-08 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -v -run '^TestServer_unconfirmed' \| grep -q -- '--- PASS: TestServer_unconfirmed'` | ✅ | ✅ green |
| 05-09-01 | 09 | 9 | PERSIST-11 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/` | ✅ | ✅ green |
| 05-09-02 | 09 | 9 | PERSIST-11 | — | per plan threat model | automated | `go build ./... && go vet ./...` | ✅ | ✅ green |
| 05-09-03 | 09 | 9 | PERSIST-11 | — | per plan threat model | automated | `! grep -q 'func (e \*Engine) runDump' internal/app/persist/engine.go # [HEAD=1 EXPECT==0]` | ✅ | ✅ green |
| 05-10-01 | 10 | 10 | PERSIST-02, PERSIST-05, PERSIST-06 | — | per plan threat model | automated | `grep -q '^## Load outcomes' docs/OPERATIONS.md && grep -q '^## Memory model' docs/OPERATIONS.md # [HEAD=0 EXPECT>=1]` | ✅ | ✅ green |
| 05-10-02 | 10 | 10 | PERSIST-02, PERSIST-05, PERSIST-06 | — | per plan threat model | automated | `go test -race -count=1 ./cmd/sup-kv/ -v -run '^TestOperationsDoc$' \| grep -q -- '--- PASS: TestOperationsDoc'` | ✅ | ✅ green |
| 05-11-01 | 11 | 10 | PERSIST-01, PERSIST-02, PERSIST-07 | — | per plan threat model | automated | `grep -q '\| S-PAR-05 \|.*\| implemented \|' docs/SPEC-DEVIATIONS.md # [HEAD=0 EXPECT>=1]` | ✅ | ✅ green |
| 05-11-02 | 11 | 10 | PERSIST-01, PERSIST-02, PERSIST-07 | — | per plan threat model | automated | `base=$(git log --diff-filter=A --format=%H -1 -- .planning/phases/05-persistence/05-CONTEXT.md) \|\| exit 1; test -n "$base" \|\| exit 1; registry_diff=$(git diff -U0 "$base" -- docs/SPEC-DEVIATIONS.md) \|\| exit 1; test "$(pr` | ✅ | ✅ green |
| 05-12-01 | 12 | 11 | PERSIST-01, PERSIST-02, PERSIST-03, PERSIST-04, PERSIST-05, PERSIST-06, PERSIST-07, PERSIST-08, PERSIST-09, PERSIST-10, PERSIST-11, PERSIST-12 | — | per plan threat model | automated | `SUPKV_RED_TESTS=1 bash scripts/red-runner` | ✅ | ✅ green |
| 05-12-02 | 12 | 11 | PERSIST-01, PERSIST-02, PERSIST-03, PERSIST-04, PERSIST-05, PERSIST-06, PERSIST-07, PERSIST-08, PERSIST-09, PERSIST-10, PERSIST-11, PERSIST-12 | — | per plan threat model | automated | `make lint` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements — the red runner, stand, simulator and
strict lint gate predate Phase 5. No Wave 0 scaffolding was needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification. The two P-01 frozen acceptances (exact-limit
restart at 63 000/64 000 bytes) run in the red-runner rows and pass directly; the runbook anchors
are enforced by `TestOperationsDoc`; the darwin build/vet gate runs in the plan and phase gates.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 300 s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-11-22 (audited after 12/12 plans; every task automated, 0 gaps)

## Validation Audit 2026-11-22

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
