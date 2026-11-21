---
phase: "3"
slug: "transport"
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-21"
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test -race` + shell/test gates; red-gated suite via `scripts/red-runner` (expectation table now 5 green / 2 red) |
| **Config file** | Makefile (lint/test/red-run/bench-smoke/race-soak) |
| **Quick run command** | `bash scripts/red-runner` (7 rows + mutation stage) |
| **Full suite command** | `make lint && make test` |
| **Estimated runtime** | ~2 min quick, ~4 min full red gate at performance governor |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` gate commands
- **After every plan wave:** build gate + test gate
- **Before `/gsd:verify-work`:** full suite + red gate green
- **Max feedback latency:** per-task gates sub-second to ~2 min (real-socket wall-clock scenarios)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 03-01-1 | 03-01 | 1 | XPORT-05 | shell/test gate | `go test -race -count=1 ./internal/udprepl/ -run '^TestSo` | ✅ green |
| 03-01-2 | 03-01 | 1 | XPORT-05 | shell/test gate | `go test -race -count=1 ./internal/stand/ -run '^TestStan` | ✅ green |
| 03-01-3 | 03-01 | 1 | XPORT-05 | shell/test gate | `go test -race -count=1 ./internal/udprepl/ -run '^TestSo` | ✅ green |
| 03-02-1 | 03-02 | 1 | XPORT-11, XPORT-06 | shell/test gate | `go test -race -count=1 ./internal/app/storage/ -run '^Te` | ✅ green |
| 03-02-2 | 03-02 | 1 | XPORT-11, XPORT-06 | shell/test gate | `go test -race -count=1 ./internal/app/persist/ -run '^Te` | ✅ green |
| 03-02-3 | 03-02 | 1 | XPORT-11, XPORT-06 | shell/test gate | `go test -race -count=1 ./internal/app/internal/simnet/ -` | ✅ green |
| 03-03-1 | 03-03 | 1 | XPORT-04 | shell/test gate | `go test -race -count=1 ./internal/app/version/ -run '^Te` | ✅ green |
| 03-03-2 | 03-03 | 1 | XPORT-04 | shell/test gate | `go test -race -count=1 ./internal/app/version/ -run '^Te` | ✅ green |
| 03-04-1 | 03-04 | 2 | XPORT-06, XPORT-11 | shell/test gate | `go build ./... && go vet ./...` | ✅ green |
| 03-04-2 | 03-04 | 2 | XPORT-06, XPORT-11 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestC` | ✅ green |
| 03-05-1 | 03-05 | 3 | XPORT-02 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestR` | ✅ green |
| 03-05-2 | 03-05 | 3 | XPORT-02 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestR` | ✅ green |
| 03-05-3 | 03-05 | 3 | XPORT-02 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestC` | ✅ green |
| 03-06-1 | 03-06 | 4 | XPORT-12 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ ./internal/p` | ✅ green |
| 03-06-2 | 03-06 | 4 | XPORT-12 | shell/test gate | `go test -race -count=1 ./internal/udprepl/` | ✅ green |
| 03-06-3 | 03-06 | 4 | XPORT-12 | shell/test gate | `go test -race -count=1 ./internal/grpcapi/ ./cmd/sup-kv/` | ✅ green |
| 03-07-1 | 03-07 | 5 | XPORT-01, XPORT-02, XPORT-07, XPORT-09 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestS` | ✅ green |
| 03-07-2 | 03-07 | 5 | XPORT-01, XPORT-02, XPORT-07, XPORT-09 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestR` | ✅ green |
| 03-07-3 | 03-07 | 5 | XPORT-01, XPORT-02, XPORT-07, XPORT-09 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestE` | ✅ green |
| 03-08-1 | 03-08 | 6 | XPORT-04, XPORT-10, XPORT-03 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestD` | ✅ green |
| 03-08-2 | 03-08 | 6 | XPORT-04, XPORT-10, XPORT-03 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestD` | ✅ green |
| 03-08-3 | 03-08 | 6 | XPORT-04, XPORT-10, XPORT-03 | shell/test gate | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestMain_max` | ✅ green |
| 03-09-1 | 03-09 | 7 | XPORT-03 | shell/test gate | `go test -race -count=1 ./internal/udprepl/ -run '^TestMa` | ✅ green |
| 03-09-2 | 03-09 | 7 | XPORT-03 | shell/test gate | `go test -race -count=1 ./internal/grpcapi/ -run '^TestSe` | ✅ green |
| 03-09-3 | 03-09 | 7 | XPORT-03 | shell/test gate | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestMain_val` | ✅ green |
| 03-10-1 | 03-10 | 8 | XPORT-08, XPORT-09 | shell/test gate | `go test -race -count=1 ./internal/udprepl/ -run '^TestRe` | ✅ green |
| 03-10-2 | 03-10 | 8 | XPORT-08, XPORT-09 | shell/test gate | `go test -race -count=1 ./internal/udprepl/ -run '^TestSm` | ✅ green |
| 03-10-3 | 03-10 | 8 | XPORT-08, XPORT-09 | shell/test gate | `go test -race -count=1 ./internal/app/repl/ -run '^TestC` | ✅ green |
| 03-11-1 | 03-11 | 9 | XPORT-13 | shell/test gate | `go build ./... && go vet ./...` | ✅ green |
| 03-11-2 | 03-11 | 9 | XPORT-13 | shell/test gate | `go test -race -count=1 ./internal/app/persist/ ./interna` | ✅ green |
| 03-12-1 | 03-12 | 10 | XPORT-01, XPORT-02, XPORT-03, XPORT-04, XPORT-05, XPORT-06, XPORT-07, XPORT-08, XPORT-09, XPORT-10, XPORT-11, XPORT-12, XPORT-13 | shell/test gate | `go test -race -count=1 ./internal/metrics/ ./cmd/sup-kv/` | ✅ green |
| 03-12-2 | 03-12 | 10 | XPORT-01, XPORT-02, XPORT-03, XPORT-04, XPORT-05, XPORT-06, XPORT-07, XPORT-08, XPORT-09, XPORT-10, XPORT-11, XPORT-12, XPORT-13 | shell/test gate | `bash scripts/red-runner` | ✅ green |
| 03-12-3 | 03-12 | 10 | XPORT-01, XPORT-02, XPORT-03, XPORT-04, XPORT-05, XPORT-06, XPORT-07, XPORT-08, XPORT-09, XPORT-10, XPORT-11, XPORT-12, XPORT-13 | shell/test gate | `test "$(grep -v '^#' scripts/red-expectations.tsv \| awk ` | ✅ green |
| 03-13-1 | 03-13 | 11 | XPORT-01, XPORT-02, XPORT-05, XPORT-08, XPORT-09 | shell/test gate | `job=$(awk '/^ red-r01-x20:/{j=1;next} j&&/^ [A-Za-z0-9_-` | ✅ green |
| 03-13-2 | 03-13 | 11 | XPORT-01, XPORT-02, XPORT-05, XPORT-08, XPORT-09 | shell/test gate | `make lint && make test` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements (stand harness, runner, corpora landed in Phase 2).

---

## Manual-Only Verifications

None that are manual: the 20× R-01 evidence is the dispatchable `red-r01-x20` scheduled job; the nightly jobs are CI duties.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity holds
- [x] Wave 0 covers all MISSING references (none exist)
- [x] No watch-mode flags
- [x] Feedback latency within bounds
- [x] `nyquist_compliant: true`

**Approval:** approved 2026-11-22 (validate-phase §6 — 0 gaps; 35 task gates executed green during waves 1–11)

## Validation Audit 2026-11-21

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
