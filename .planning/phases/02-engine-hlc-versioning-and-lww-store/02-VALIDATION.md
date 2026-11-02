---
phase: "02"
slug: "engine-hlc-versioning-and-lww-store"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-02"
---

# Phase 02 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib testing, external `app_test` package, `t.Parallel()`, table-driven) |
| **Config file** | none — `go.mod` toolchain; gates in `Makefile` (lint/test/build) |
| **Quick run command** | `go test -race ./internal/app/...` |
| **Full suite command** | `make test` (`go test -race -cover ./...`) |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./internal/app/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | CORE-01, CORE-02 | — | N/A | unit + tracer | `go test -race ./internal/app/ -run '^TestStore_endToEnd$' -v` | ✅ | ✅ green |
| 02-01-02 | 01 | 1 | CORE-01..CORE-07 | — | N/A | unit (table) | `go test -race ./internal/app/ -run '^TestStore_Get$|^TestStore_Set$|^TestStore_SetIfAbsent$|^TestStore_Delete$|^TestStore_valueLimit$|^TestMerge_losers$' -v` | ✅ | ✅ green |
| 02-01-03 | 01 | 1 | CORE-01..CORE-07 (gate) | — | N/A | gate | `go build ./... && go vet ./... && go test -race ./...` + `go tool golangci-lint config verify && go tool golangci-lint run ./...` | ✅ | ✅ green |
| 02-02-01 | 02 | 2 | VERS-03, VERS-04, VERS-05 | — | N/A | unit (table) | `go test -race ./internal/app/ -run '^TestClock_tick$|^TestClock_update$|^TestClock_counterOverflowBoundary$|^TestClock_restartFromDumpMax$|^TestClock_driftSignal$' -v` | ✅ | ✅ green |
| 02-02-02 | 02 | 2 | VERS-01, VERS-02 | — | N/A | unit + property | `go test -race ./internal/app/ -run '^TestVersion_' -v` | ✅ | ✅ green |
| 02-02-03 | 02 | 2 | VERS-01..VERS-05, TEST-02 | — | N/A | property | `go test -race ./internal/app/ -run '^TestClock_monotonic$|^TestVersion_uniquePerWrite$|^TestClock_postReceive$|^TestClock_counterReset$|^TestClock_restartProperty$|^TestClock_driftUnderChaos$' -v` | ✅ | ✅ green |
| 02-03-01 | 03 | 3 | TEST-01, CORE-06, CORE-07 | — | N/A | property | `go test -race ./internal/app/ -run '^TestMerge_' -v` | ✅ | ✅ green |
| 02-03-02 | 03 | 3 | CORE-02..CORE-07 (gate) | — | N/A | gate + race hammer | `go test -race ./internal/app/ -run '^TestStore_concurrentHammer$|^TestStore_concurrentSetIfAbsent$|^TestStore_concurrentDeleteVsSet$' -v` + full lint/vet/race gate | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Coverage: `go test -race -cover ./...` reports **100.0% of statements in internal/app**. Property runs print their seed and replay deterministically (`-args -seed=N`).

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements — Go toolchain, Makefile gates and the seeded-generator conventions were installed by plans 02-01/02-02 themselves; no stubs were needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-11-02 (auto chain, gap count 0)

## Validation Audit 2026-11-02

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
