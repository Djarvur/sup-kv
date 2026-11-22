---
phase: "6"
slug: "api-and-semantics"
status: draft
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-22"
---

# Phase 6 — Validation Strategy

> Per-phase validation contract. Audited after execution: every task in the phase plans
> carries an automated verification that ran green on the committed tree.
> Populated at plan revision time from the 13 plans' `<automated>` blocks; the
> post-execution validate-phase pass flips each status.

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

*File Exists legend: ✅ test file present at phase start · ➕ test file created by the task.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | APIS-06, APIS-02 | T-06-01 / T-06-02 | Public schema moved, page fields additive, descriptor pinned | unit | `go test -race -count=1 ./api/supkvapi/v1/ -run '^TestSchema_' -v`; `go build ./...`; `go tool buf breaking --against '.git#ref=HEAD'` | ➕ | ⬜ pending |
| 06-01-02 | 01 | 1 | APIS-06 | T-06-02 | Every real importer uses the public path; frozen tests keep the alias | build | `go build ./... && go vet ./...` (plus the legacy-path count gate, baseline [HEAD=18 EXPECT==0]) | ✅ | ⬜ pending |
| 06-01-03 | 01 | 1 | APIS-06, APIS-02 | T-06-01 / T-06-02 | buf lint/format/breaking green; public package cannot import internal/ | lint | `go tool buf lint && go tool buf format --diff --exit-code && make buf-breaking`; `go tool golangci-lint config verify && go tool golangci-lint run ./api/... ./internal/grpcapi/...` | ➕ | ⬜ pending |
| 06-02-01 | 02 | 2 | APIS-01 | T-06-05 / T-06-06 | Zero peers and nil seam answer honest false; one write-time capture | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestWait_(zeroPeers|trackerNilIsSingleNode|writeTimeSetPinned|deleteSymmetry)$' -v`; `go build ./... && go vet ./...` | ✅ | ⬜ pending |
| 06-03-01 | 03 | 2 | APIS-06 | T-06-08 / T-06-09 | Separate module builds only against the public package; tidy is idempotent | build | `cd examples/client && go mod tidy && go build ./... && go vet ./...` with the go.mod/go.sum hash-idempotence check | ➕ | ⬜ pending |
| 06-03-02 | 03 | 2 | APIS-06 | T-06-09 | One Makefile target gates the example; nested module stays out of the root graph | build | `make client-example`; `test -z "$(go list ./... | grep examples/client)"` | ✅ | ⬜ pending |
| 06-03-03 | 03 | 2 | APIS-06 | T-06-10 | README states the consumer-zone boundary and names no internal path | docs | `test "$(grep -c 'github.com/Djarvur/sup-kv/internal' examples/client/README.md)" -eq 0`; `grep -q 'next_page_token' ...; grep -q 'ADR 0007' ...` | ➕ | ⬜ pending |
| 06-04-01 | 04 | 3 | APIS-03 | T-06-11 / T-06-13 | Missing snapshot is NotFound, no dump is FailedPrecondition, no path leaks | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_(missingSnapshotIsNotFound|noDumpIsFailedPrecondition)$' -v`; `go test -race -count=1 ./internal/persistfile/ -run '^TestFileStore_snapshotErrorClasses$' -v` | ➕ | ⬜ pending |
| 06-04-02 | 04 | 3 | APIS-03 | T-06-11 | Total mapping table; constant Internal fallback logs the detail | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestMapError_(totalTable|unmappedIsConstantAndLogged)$' -v`; `go test -race -count=1 ./internal/persistfile/ -run '^TestCreateSnapshot_failureClasses$' -v` | ➕ | ⬜ pending |
| 06-04-03 | 04 | 3 | APIS-03 | T-06-11 / T-06-12 | Every RPC status message is path-free | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestRPC_statusMessagesPathFree$' -v` | ➕ | ⬜ pending |
| 06-05-01 | 05 | 4 | APIS-02 | T-06-14 / T-06-15 | Legacy all-keys keys-only listing; paged first page and continuation; live view | integration | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list(LegacyAll|FirstPage|PageContinuation)$' -v`; `go test -race -count=1 ./internal/app/storage/ -run '^TestStore_liveVersions' -v`; `go vet ./internal/stand/` | ➕ | ⬜ pending |
| 06-05-02 | 05 | 4 | APIS-02 | T-06-15 | Sizes, tokens, prefixes, empty input, tombstone races in both modes | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list(Boundaries|Empty|Token|Prefix|Race)$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list$' -v` | ✅ | ⬜ pending |
| 06-05-03 | 05 | 4 | APIS-02 | T-06-14 | Both paths' cost is independent of off-page value bytes; budget progress | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_listLegacyCostIndependent$' -v`; `... -run '^TestServer_listCostIndependent$' -v`; `... -run '^TestServer_listByteBudget$' -v`; `... -run '^TestListPage_maxKeySerializedBound$' -v` | ➕ | ⬜ pending |
| 06-06-01 | 06 | 5 | APIS-04 | T-06-17 | Generation-tagged installs; watermark after confirmWaiters; unconfirmed never announced | integration | `go test -race -count=1 ./internal/app/storage/ -run '^TestStore_changeTapGeneration$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch(Hub_confirmGate|_unconfirmedNotAnnounced)$' -v`; `go build ./... && go vet ./internal/grpcapi/` | ✅ | ⬜ pending |
| 06-07-01 | 07 | 7 | APIS-05 | T-06-21 / T-06-24 | Limits are a pure function of the effective value limit; production wiring | unit | `go test -race -count=1 ./internal/grpcapi/ -run '^TestLimitsFor$' -v`; `go build ./...` | ➕ | ⬜ pending |
| 06-07-02 | 07 | 7 | APIS-05 | T-06-21 | Over-limit input is ResourceExhausted; worst-case page fits MaxSendBytes | integration | `go test -race -count=1 ./internal/grpcapi/ -run '^TestGRPC_maxRecv$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestGRPC_worstCasePageFitsSendLimit$' -v` | ➕ | ⬜ pending |
| 06-07-03 | 07 | 7 | APIS-05 | T-06-23 | Health is one-way NOT_SERVING; binary shuts the API server down first | integration | `go test -race -count=1 ./internal/grpcapi/ -run '^TestHealth_shutdownIsOneWay$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestHealth_reflectsGate$' -v`; `go build ./... && go vet ./cmd/sup-kv` | ✅ | ⬜ pending |
| 06-08-01 | 08 | 7 | APIS-07 | T-06-25 / T-06-27 | Full rollback: absent live keys deleted with fresh tombstones, no version burn | unit | `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_snapshotCreateLoadRestore$' -v`; `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_restoreSkipsReplay$' -v` | ✅ | ⬜ pending |
| 06-08-02 | 08 | 7 | APIS-07 | T-06-25 | Extra live key is a change; no-op control stays empty and dump-free | unit | `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_restore(ReturnsReplayed|ExtraLiveKeyIsNotNoOp|NoOpReturnsEmpty)$' -v`; `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_restore' -v` | ✅ | ⬜ pending |
| 06-08-03 | 08 | 7 | APIS-07 | T-06-26 | Rollback tombstones broadcast, including on a failed final dump | integration | `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_snapshotLoadRollback$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_snapshotLoad' -v` | ➕ | ⬜ pending |
| 06-09-01 | 09 | 8 | APIS-01, APIS-02, APIS-06 | T-06-29 | Public proto, honest wait, and two-mode List pagination documented and anchored | docs | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v`; `grep -q '^## wait_for_replication' docs/API.md && grep -q '^## List pagination' docs/API.md` | ➕ | ⬜ pending |
| 06-09-02 | 09 | 8 | APIS-03, APIS-04, APIS-05, APIS-07 | T-06-28 / T-06-29 | Error table, Watch, snapshots, transport, consumer-zone documented and anchored | docs | `go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v`; section-heading greps for Errors/Watch/Snapshots/Transport | ✅ | ⬜ pending |
| 06-10-01 | 10 | 8 | APIS-06 | T-06-30 | No non-historical doc names the legacy internal path; public path present | docs | legacy-path count gate [HEAD=5 EXPECT==0]; `grep -q 'api/supkvapi/v1' docs/SPEC-DEVIATIONS.md && grep -q 'api/supkvapi/v1' AGENTS.md` | ✅ | ⬜ pending |
| 06-10-02 | 10 | 8 | APIS-01, APIS-02, APIS-07 | T-06-30 / T-06-31 | Examples state List modes, zero-peer wait, and the SnapshotLoad full rollback; IN-05 closed; S-PERS-12 flipped | docs | List/zero-peer greps; `grep -qi 'full rollback' docs/EXAMPLES.md && grep -qi 'absent from the snapshot' docs/EXAMPLES.md`; IN-05/WritePeers and S-PERS-12 greps; `go test -race -count=1 ./cmd/sup-kv/ -run '^TestExamplesDoc$' -v` | ✅ | ⬜ pending |
| 06-11-01 | 11 | 3 | APIS-01 | T-06-04 / T-06-07 | Tracker scoped to captured peers; discovery flush no longer grows the promise | unit | `go test -race -count=1 ./internal/app/repl/ -run '^(TestRetransmit_|TestPeerTable_|TestNotify_)' -v`; `go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated_' -v` | ✅ | ⬜ pending |
| 06-11-02 | 11 | 3 | APIS-01 | T-06-04 | Superseded fast path, inert stranger Ack, tracker release, captured-peer-only integration | integration | `go test -race -count=1 ./internal/app/repl/ -run '^TestNotify_supersededResult$' -v`; `go test -race -count=1 ./internal/udprepl/ -run '^TestWaitReplicated_(supersededFast|strangerAckDoesNotResolve|capturedPeerOnly)$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestWait_integration' -v` | ✅ | ⬜ pending |
| 06-12-01 | 12 | 6 | APIS-04 | T-06-20 | Startup load is silent and nil-safe; failed-then-confirmed announced once | integration | `go test -race -count=1 ./internal/app/persist/ -run '^TestOpen_(loadDoesNotFireTap|nilTapStaysWritable)$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_(dumpFailedThenConfirmed|confirmedExactlyOnce)$' -v` | ➕ | ⬜ pending |
| 06-12-02 | 12 | 6 | APIS-04 | T-06-18 / T-06-19 | Shutdown is Unavailable; pending is bounded; deletions and order pinned | integration | `go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch_shutdownUnavailable$' -v`; `go test -race -count=1 ./internal/grpcapi/ -run '^TestWatch(Hub_pendingBounded|_deletionsAndOrder)$' -v` | ✅ | ⬜ pending |
| 06-13-01 | 13 | 9 | APIS-01…07 | T-06-33 | Frozen red evidence still hashes to the planning-time pin; red-runner 7/7 | gate | three `git hash-object` literal comparisons; `bash scripts/red-runner` with the 7-row/0-error summary | ✅ | ⬜ pending |
| 06-13-02 | 13 | 9 | APIS-01…07 | T-06-34 | Standing lint, test, and buf-breaking gates green on the final tree | gate | `make lint && make test && make buf-breaking` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. Every task's automated command
targets a test that an existing file or the task itself creates; no stub plan is needed.
The only pre-existing, non-negotiable evidence is the frozen red suite, pinned by content
hash in plan 06-13.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Phase-9 documentation split (full failure table, latency model, key-prefix convention) | APIS-03, APIS-06 | The split is a scope statement, not a behavior | Read `docs/API.md` "What Phase 9 adds" and confirm it names DOCS-05 without claiming those sections complete |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none: existing infrastructure plus in-task files)
- [x] No watch-mode flags (`go test -count=1` everywhere)
- [x] Feedback latency < 300 s (full suite); per-task commands ≤ 60 s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
