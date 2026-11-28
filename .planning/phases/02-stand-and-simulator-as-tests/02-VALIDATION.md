---
phase: "2"
slug: "stand-and-simulator-as-tests"
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-19"
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test -race` + shell/test gates; red-gated suite under `SUPKV_RED_TESTS=1` via `scripts/red-runner` |
| **Config file** | Makefile (lint/test/red-run/bench-smoke/race-soak targets) |
| **Quick run command** | `bash scripts/red-runner` (7 expected-red rows + 4 controls + 4 mutant kills) |
| **Full suite command** | `make lint && make test` |
| **Estimated runtime** | ~2.5 min (red-runner ~6 min at documented scale on schedule; local default scale ~2 min) |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` gate commands
- **After every plan wave:** build gate (`go build ./...`) + test gate (`make test`)
- **Before `/gsd:verify-work`:** full suite must be green
- **Max feedback latency:** per-task gates sub-second to ~2 min (real-socket reds by design); heavy runs on the nightly schedule

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 02-01-1 | 02-01 | 1 | TEST-04 | shell/test gate | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/grpc` | ✅ green |
| 02-01-2 | 02-01 | 1 | TEST-04 | shell/test gate | `bash scripts/red-runner` | ✅ green |
| 02-01-3 | 02-01 | 1 | TEST-04 | shell/test gate | `make red-run` | ✅ green |
| 02-02-1 | 02-02 | 1 | TEST-05, TEST-03 | shell/test gate | `go build ./internal/stand/ && go vet ./internal/stand/` | ✅ green |
| 02-02-2 | 02-02 | 1 | TEST-05, TEST-03 | shell/test gate | `go test -race -count=1 ./internal/stand/ -run '^TestStan` | ✅ green |
| 02-03-1 | 02-03 | 1 | TEST-06, TEST-02 | shell/test gate | `go test -race -count=1 ./internal/app/internal/simnet/ .` | ✅ green |
| 02-03-2 | 02-03 | 1 | TEST-06, TEST-02 | shell/test gate | `go test -race -count=1 ./internal/app/internal/simnet/ .` | ✅ green |
| 02-04-1 | 02-04 | 1 | TEST-03 | shell/test gate | `go test -race -count=1 ./internal/persistfile/` | ✅ green |
| 02-04-2 | 02-04 | 1 | TEST-03 | shell/test gate | `go test -race -count=1 ./internal/persistfile/ -run '^Te` | ✅ green |
| 02-05-1 | 02-05 | 2 | TEST-04 | shell/test gate | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stan` | ✅ green |
| 02-05-2 | 02-05 | 2 | TEST-04 | shell/test gate | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stan` | ✅ green |
| 02-06-1 | 02-06 | 2 | TEST-01 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestC` | ✅ green |
| 02-06-2 | 02-06 | 2 | TEST-01 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestC` | ✅ green |
| 02-07-1 | 02-07 | 2 | TEST-02, TEST-08 | shell/test gate | `go test -race -count=1 ./internal/udprepl/ -run '^TestDe` | ✅ green |
| 02-07-2 | 02-07 | 2 | TEST-02, TEST-08 | shell/test gate | `go test -race -count=1 ./internal/stand/ -run '^TestStan` | ✅ green |
| 02-08-1 | 02-08 | 2 | TEST-14, TEST-12 | shell/test gate | `go test -race -count=1 ./internal/grpcapi/ -run '^TestWa` | ✅ green |
| 02-08-2 | 02-08 | 2 | TEST-14, TEST-12 | shell/test gate | `go test -race -count=1 ./internal/stand/ -run '^TestStan` | ✅ green |
| 02-09-1 | 02-09 | 3 | TEST-13, TEST-15 | shell/test gate | `go test -count=1 ./internal/app/repl/ -run '^TestDecisio` | ✅ green |
| 02-09-2 | 02-09 | 3 | TEST-13, TEST-15 | shell/test gate | `go test -race -count=1 ./internal/app/storage/ -run '^Te` | ✅ green |
| 02-09-3 | 02-09 | 3 | TEST-13, TEST-15 | shell/test gate | `go test -race -count=1 ./internal/app/persist/ -run '^Te` | ✅ green |
| 02-10-1 | 02-10 | 2 | TEST-10, TEST-11 | shell/test gate | `go test -run '^$' -bench . -benchtime=1x ./internal/app/` | ✅ green |
| 02-10-2 | 02-10 | 2 | TEST-10, TEST-11 | shell/test gate | `make race-soak COUNT=1` | ✅ green |
| 02-11-1 | 02-11 | 5 | TEST-16, TEST-06, TEST-08, TEST-11 | shell/test gate | `grep -c 'timeout-minutes:' .github/workflows/scheduled.y` | ✅ green |
| 02-11-2 | 02-11 | 5 | TEST-16, TEST-06, TEST-08, TEST-11 | shell/test gate | `grep -c 'go test -race ./internal/stand/' .github/workfl` | ✅ green |
| 02-11-3 | 02-11 | 5 | TEST-16, TEST-06, TEST-08, TEST-11 | shell/test gate | `grep -c 'shuffle=on\\|count=3\\|benchtime=1x\\|SUPKV_STAND_` | ✅ green |
| 02-12-1 | 02-12 | 3 | TEST-04 | shell/test gate | `SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stan` | ✅ green |
| 02-12-2 | 02-12 | 3 | TEST-04 | shell/test gate | `bash scripts/red-runner && grep -v '^#' scripts/red-expe` | ✅ green |
| 02-13-1 | 02-13 | 4 | TEST-01, TEST-09 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestS` | ✅ green |
| 02-13-2 | 02-13 | 4 | TEST-01, TEST-09 | shell/test gate | `bash scripts/red-runner` | ✅ green |
| 02-14-1 | 02-14 | 1 | TEST-05 | shell/test gate | `GOOS=darwin go vet ./internal/stand/` | ✅ green |
| 02-15-1 | 02-15 | 2 | TEST-06, TEST-02 | shell/test gate | `go test -race -count=1 ./internal/app/internal/simnet/ -` | ✅ green |
| 02-16-1 | 02-16 | 2 | TEST-07 | shell/test gate | `go test -race -count=1 ./internal/persistfile/ -run '^Te` | ✅ green |
| 02-17-1 | 02-17 | 3 | TEST-08 | shell/test gate | `go test -race -count=1 ./internal/udprepl/ -run '^FuzzDe` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements (harness, corpora, runner, workflows all landed inside the phase).

---

## Manual-Only Verifications

All phase behaviors have automated verification. The 10 000-seed TEST-01 proof is a dispatchable scheduled run (`gh workflow run scheduled.yml -f seeds=10000`) — not manual, nightly-exercised.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none exist)
- [x] No watch-mode flags
- [x] Feedback latency within schedule bounds
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-11-21 (validate-phase §6 — 0 gaps found; all 33 task gates executed green during waves 1–5)

## Validation Audit 2026-11-21

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
