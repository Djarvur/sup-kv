---
phase: "6"
slug: "api-and-semantics"
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-22"
validated: "2026-11-23"
---

# Phase 6 — Validation Strategy

> Per-phase validation contract. Audited after execution: every task in the 13 plans
> carries an automated verification that ran green on the committed tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` + `go test -race` |
| **Config file** | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `buf-lint`, `buf-breaking`, `client-example`) |
| **Quick run command** | `go test -race -count=1 ./internal/grpcapi/ ./cmd/sup-kv/ ./api/supkvapi/v1/` |
| **Full suite command** | `make lint && make test && make buf-lint && make buf-breaking && make client-example` (+ `SUPKV_RED_TESTS=1 bash scripts/red-runner`) |
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
| 06-01-01 | 01 | 1 | APIS-02, APIS-06 | — | per plan threat model | automated | `go test -race -count=1 ./api/supkvapi/v1/ -run '^TestSchema_' -v` | ✅ | ✅ green |
| 06-01-02 | 01 | 1 | APIS-02, APIS-06 | — | per plan threat model | automated | `test "$(grep -rl 'internal/grpcapi/apiproto/supkvapi/v1' --include='*.go' cmd internal \| grep -v -e 'red_gate_test.go' -e 'stand_red_test.go' \| wc -l)" -eq 0 # [HEAD=18 EXPECT==0]` | ✅ | ✅ green |
| 06-01-03 | 01 | 1 | APIS-02, APIS-06 | — | per plan threat model | automated | `go tool buf lint && go tool buf format --diff --exit-code && make buf-breaking` | ✅ | ✅ green |
| 06-02-01 | 02 | 2 | APIS-01 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestWait_(zeroPeers\|trackerNilIsSingleNode\|writeTimeSetPinned\|deleteSymmetry)$' -v` | ✅ | ✅ green |
| 06-03-01 | 03 | 2 | APIS-06 | — | per plan threat model | automated | `cd examples/client && go mod tidy && go build ./... && go vet ./... && h1=$(git hash-object go.mod go.sum) && go mod tidy && h2=$(git hash-object go.mod go.sum) && test "$h1" = "$h2"` | ✅ | ✅ green |
| 06-03-02 | 03 | 2 | APIS-06 | — | per plan threat model | automated | `make client-example` | ✅ | ✅ green |
| 06-03-03 | 03 | 2 | APIS-06 | — | per plan threat model | automated | `test "$(grep -c 'github.com/Djarvur/sup-kv/internal' examples/client/README.md)" -eq 0` | ✅ | ✅ green |
| 06-04-01 | 04 | 3 | APIS-03 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_(missingSnapshotIsNotFound\|noDumpIsFailedPrecondition)$' -v` | ✅ | ✅ green |
| 06-04-02 | 04 | 3 | APIS-03 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_totalTable$' -v` | ✅ | ✅ green |
| 06-04-03 | 04 | 3 | APIS-03 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestRPC_statusMessagesPathFree$' -v` | ✅ | ✅ green |
| 06-05-01 | 05 | 4 | APIS-02 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list(LegacyAll\|FirstPage\|PageContinuation)$' -v` | ✅ | ✅ green |
| 06-05-02 | 05 | 4 | APIS-02 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list(Boundaries\|Empty\|Token\|Prefix\|Race)$' -v` | ✅ | ✅ green |
| 06-05-03 | 05 | 4 | APIS-02 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_listLegacyCostIndependent$' -v` | ✅ | ✅ green |
| 06-06-01 | 06 | 5 | APIS-04 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/storage/ -run '^TestStore_changeTapGeneration$' -v` | ✅ | ✅ green |
| 06-07-01 | 07 | 7 | APIS-05 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestLimitsFor$' -v` | ✅ | ✅ green |
| 06-07-02 | 07 | 7 | APIS-05 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestGRPC_maxRecv$' -v` | ✅ | ✅ green |
| 06-07-03 | 07 | 7 | APIS-05 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestHealth_shutdownIsOneWay$' -v` | ✅ | ✅ green |
| 06-08-01 | 08 | 7 | APIS-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_snapshotCreateLoadRestore$' -v` | ✅ | ✅ green |
| 06-08-02 | 08 | 7 | APIS-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_restore(ReturnsReplayed\|ExtraLiveKeyIsNotNoOp\|NoOpReturnsEmpty)$' -v` | ✅ | ✅ green |
| 06-08-03 | 08 | 7 | APIS-07 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_snapshotLoadRollback$' -v` | ✅ | ✅ green |
| 06-09-01 | 09 | 8 | APIS-01, APIS-02, APIS-03, APIS-04, APIS-05, APIS-06, APIS-07 | — | per plan threat model | automated | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v` | ✅ | ✅ green |
| 06-09-02 | 09 | 8 | APIS-01, APIS-02, APIS-03, APIS-04, APIS-05, APIS-06, APIS-07 | — | per plan threat model | automated | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v` | ✅ | ✅ green |
| 06-10-01 | 10 | 8 | APIS-01, APIS-02, APIS-06, APIS-07 | — | per plan threat model | automated | `test "$(grep -rn 'internal/grpcapi/apiproto' docs AGENTS.md --include='*.md' \| grep -v 'docs/reviews' \| wc -l)" -eq 0 # [HEAD=5 EXPECT==0]` | ✅ | ✅ green |
| 06-10-02 | 10 | 8 | APIS-01, APIS-02, APIS-06, APIS-07 | — | per plan threat model | automated | `grep -q 'nextPageToken' docs/EXAMPLES.md && grep -q 'records' docs/EXAMPLES.md && grep -qi 'zero known peers' docs/EXAMPLES.md` | ✅ | ✅ green |
| 06-11-01 | 11 | 3 | APIS-01 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/repl/ -run '^(TestRetransmit_\|TestPeerTable_\|TestNotify_)' -v` | ✅ | ✅ green |
| 06-11-02 | 11 | 3 | APIS-01 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/repl/ -run '^TestNotify_supersededResult$' -v` | ✅ | ✅ green |
| 06-12-01 | 12 | 6 | APIS-04 | — | per plan threat model | automated | `go test -race -count=1 ./internal/app/persist/ -run '^TestOpen_(loadDoesNotFireTap\|nilTapStaysWritable)$' -v` | ✅ | ✅ green |
| 06-12-02 | 12 | 6 | APIS-04 | — | per plan threat model | automated | `go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_shutdownUnavailable$' -v` | ✅ | ✅ green |
| 06-13-01 | 13 | 9 | APIS-01, APIS-02, APIS-03, APIS-04, APIS-05, APIS-06, APIS-07 | — | per plan threat model | automated | `a=$(git hash-object internal/grpcapi/red_gate_test.go) && b=$(git hash-object internal/stand/stand_red_test.go) && c=$(git hash-object scripts/red-expectations.tsv) && echo "red_gate=$a stand_red=$b tsv=$c" && test "$a" ` | ✅ | ✅ green |
| 06-13-02 | 13 | 9 | APIS-01, APIS-02, APIS-03, APIS-04, APIS-05, APIS-06, APIS-07 | — | per plan threat model | automated | `make lint && make test && make buf-lint && make buf-breaking && make client-example` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No Wave 0 scaffolding was needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification, including the executable docs (`TestExamplesDoc`,
`TestAPIDoc`) and the separate client module build (`make client-example`).

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 300 s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-11-23 (audited after 13/13 plans; every task automated, 0 gaps)

## Validation Audit 2026-11-23

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
