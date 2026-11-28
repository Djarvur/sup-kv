---
phase: 06-api-and-semantics
plan: 09
subsystem: api-contract-docs
tags:
  - apis-01
  - apis-02
  - apis-03
  - apis-04
  - apis-05
  - apis-06
  - apis-07
  - docs-api
  - doc-anchor-guard
  - wait-for-replication
  - keyset-pagination
  - error-table
  - watch-contract
  - snapshot-full-rollback
  - transport-envelope
  - consumer-zone
requires:
  - phase: 06-api-and-semantics (06-01)
    provides: the public api/supkvapi/v1 schema, import path and package names
  - phase: 06-api-and-semantics (06-05)
    provides: the two List modes, the page token, maxListPageSize and the 6 MiB budget with listEnvelopeOverheadBytes
  - phase: 06-api-and-semantics (06-07)
    provides: the explicit transport envelope (63 846-byte receive ceiling, 6 MiB + 1 382 send ceiling, 2 m/20 s/30 s keepalive) and the one-way health rule
  - phase: 06-api-and-semantics (06-08)
    provides: the snapshot-load full rollback and its broadcast report
  - phase: 06-api-and-semantics (06-04) / (06-06) / (06-12)
    provides: the ordered total mapper and the durability-gated Watch hub with its shutdown status
provides:
  - "docs/API.md: the phase-owned API contract — public proto, honest wait, List pagination, total path-free error table, Watch durability contract, snapshot full rollback, transport/health envelope, consumer-zone boundary, explicit Phase 9 split"
  - "cmd/sup-kv/api_docs_test.go: TestAPIDoc, an (anchor, pins) table read from the repo file, so section deletion or heading drift fails the normal test gate"
affects:
  - 06-13-phase-gate
  - phase-9-api-docs (DOCS-05 completes the surrounding document)
actuals:
  tokens: 7664
  tasks: 2
  commits: 2
  plan_head_before: 2a87fc98f6513f8ac078a6d20b2478b9bb75c040
  plan_head_after: 6ddcc07a880f0cb826fe5c7cbd64e3d4ac60d71c
tech-stack:
  added: []
  patterns:
    - "A contract document ships with its own anchor guard: one (literal, pins) row per mandated statement, read from the repo file, so a deleted section or a renamed heading fails the normal test gate rather than drifting silently."
    - "Every documented guarantee names its limit as implemented — caller-bounded wait, non-linearizable rollback, current-memory snapshot — and the Phase 9 remainder is named explicitly instead of half-written."
key-files:
  created:
    - docs/API.md
    - cmd/sup-kv/api_docs_test.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-09-task1.json
    - .planning/phases/06-api-and-semantics/red-evidence/06-09-task2.json
  modified: []
key-decisions:
  - "TestAPIDoc reads docs/API.md from the repo (../../docs/API.md) and asserts one literal per mandated statement; its doc comment states the honest scope — presence and heading drift, not the truth of each sentence, which is the phase read-through's job."
  - "The error table follows the mapper's first-match order and states why: pre-built statuses pass through, ErrDumpFailed precedes the port classes, ErrNoDump precedes the fs rows, and context is answered through the matched sentinel via status.FromContextError."
  - "The error section documents the handler validation rows (empty key, key length, startup gate, listed-but-unarrived, watch shed, watch unimplemented) beside the mapper rows, so a client sees every constant message the API can return."
  - "The Watch section states the deliberate snapshot/live asymmetry (current memory vs durable change feed) and the 256-event subscriber buffer / 4 096-event hub bound behind ResourceExhausted resubscription."
  - "The snapshot section names the full rollback's limits rather than overclaiming: fresh versions, rollback tombstones, no version burned for an already-tombstoned absent key, result broadcast, and not linearizable against concurrent writes."
  - "The transport section pins the receive ceiling at the effective value limit (63 846 at the default, 66 814 at the 64 408 datagram ceiling), the 6 MiB + 1 382 send ceiling, the 2 m/20 s/30 s keepalive triple and the one-way NOT_SERVING rule."
  - "The last-read-value cache is documented as consumer-library territory (ADR 0007 / decision 14.7), explicitly not a server feature, and the Phase 9/DOCS-05 remainder (full per-RPC failure table, write-latency model, key-prefix convention) is named as pending rather than claimed."
  - "All seven APIS IDs stay blocked in requirements ready-ids (the #2388 shared-ID gate: 06-13 also declares them), so requirements-completed stays empty and the phase gate closes the checkboxes (06-05/06-07/06-08 precedent)."
patterns-established:
  - "A doc-anchor guard for a contract document mirrors internal/metrics cross-checks where possible but stays an anchor table when the statements are prose: one literal per mandated fact, each row naming what its loss would mean."
  - "Documenting a deliberately limited guarantee follows the same shape every time: the promise, the bound, and the name of the thing the implementation does not provide."
requirements-completed: []
coverage:
  - id: D1
    description: "docs/API.md exists, opens with '# sup-kv API', and the Public proto section names the proto file, package supkvapi.v1, the Go import path github.com/Djarvur/sup-kv/api/supkvapi/v1, the Go package name supkvapi, the SupKVService service, the seven RPCs, make proto and the reflection/grpcurl path"
    requirement: "APIS-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (anchors '# sup-kv API', '## Public proto', import path, 'supkvapi.v1', 'SupKVService', 'make proto')"
        status: pass
      - kind: process
        ref: "grep -n '^## ' docs/API.md -> Public proto first; grep -n 'internal/' docs/API.md -> no match"
        status: pass
    human_judgment: false
  - id: D2
    description: "The wait_for_replication section states the write-time peer set, the zero-known-peers (false, 0, 0) answer, that an outside Ack cannot yield true, the Superseded fast path (20 ms design target), the caller-bounded timeout with the wait_timeout_ms clamp and the aggregate acked_count/total_peers meaning"
    requirement: "APIS-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (anchors '## wait_for_replication', 'write-time peer set', 'zero known peers', 'acked=false', 'total_peers=0', 'Superseded', 'wait_timeout_ms', 'caller-bounded')"
        status: pass
      - kind: process
        ref: "grep -q '^## wait_for_replication' docs/API.md -> exit 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "The List pagination section states the two modes (unset/0 legacy all-keys keys-only listing; page_size > 0 bounded records page), page_size/page_token/next_page_token, byte-wise ordering, the token as a position not a snapshot, the 256 maximum, the 6 MiB budget as an upper bound, the empty-token end signal, and that a key space above the transport send ceiling requires page_size > 0"
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (anchors '## List pagination', page_size/page_token/next_page_token, 'byte-wise', 'position', 'unset', 'legacy', 'keys only', '256', '6 MiB', 'empty token')"
        status: pass
      - kind: process
        ref: "grep -q '^## List pagination' docs/API.md -> exit 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "The Errors section carries one row per mapper constant and List constant message (key not found, snapshot not found, key already exists, value exceeds the maximum allowed size, invalid snapshot name, invalid page size, invalid page token, no dump on disk to snapshot, dump failed; write not confirmed, write not confirmed before the confirmation timeout, persist engine closed, watch hub is shutting down, disk full, permission denied, Canceled, DeadlineExceeded, internal error), states the ordering rule, that internal detail is logged and never returned, and that no message carries a server path"
    requirement: "APIS-03"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (one anchor literal per constant plus 'detail is logged' and 'no server path')"
        status: pass
      - kind: process
        ref: "grep -q '^## Errors' docs/API.md -> exit 0; the table rows were derived from every status.Error/status.Errorf branch in internal/grpcapi/server.go and internal/grpcapi/list.go"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Watch section states the current-in-memory snapshot, after-dump-confirmation release of live events, exactly-once delivery, deletions as events, version order, the snapshot/live asymmetry, the 256-event buffer and 4 096-event hub bound with ResourceExhausted resubscribe, and Unavailable 'watch hub is shutting down' on shutdown"
    requirement: "APIS-04"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (anchors '## Watch', 'after the dump confirmation', 'exactly once', 'Deletions are events', 'version order', 'resubscribe', 'current in-memory state', 'watch hub is shutting down')"
        status: pass
      - kind: process
        ref: "grep -q '^## Watch' docs/API.md -> exit 0"
        status: pass
    human_judgment: false
  - id: D6
    description: "The Snapshots section states full rollback: snapshot keys replay with fresh versions, live keys absent from the snapshot are deleted with fresh tombstones, an already-tombstoned absent key burns no version and produces no record, the result is broadcast (even on a failed final dump, never on a closed engine), and the rollback is not linearizable against concurrent writes"
    requirement: "APIS-07"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (anchors '## Snapshots', 'full rollback', 'absent from the snapshot', 'fresh version', 'no version is burned', 'broadcast', 'not linearizable')"
        status: pass
      - kind: process
        ref: "grep -q '^## Snapshots' docs/API.md -> exit 0"
        status: pass
    human_judgment: false
  - id: D7
    description: "The Transport and health section pins the keepalive triple (2 m ping, 20 s answer timeout, 30 s minimum client interval, PermitWithoutStream), MaxRecvBytes (1 024 + effective value limit + 1 382 = 63 846 at the default, 66 814 at the datagram ceiling) and MaxSendBytes (6 MiB + 1 382), the transport ResourceExhausted verdict distinct from the handler InvalidArgument, and the one-way NOT_SERVING rule that never returns to SERVING"
    requirement: "APIS-05"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (anchors '## Transport and health', '2 m', '20 s', '30 s', 'MaxRecvBytes', 'MaxSendBytes', '63 846', 'ResourceExhausted', 'NOT_SERVING', 'never returns to SERVING')"
        status: pass
      - kind: process
        ref: "grep -q '^## Transport and health' docs/API.md -> exit 0"
        status: pass
    human_judgment: false
  - id: D8
    description: "The consumer-zone section states the last-read-value file/cache pattern is consumer-library territory, not a server feature this binary ships, citing ADR 0007 and decision 14.7; the What Phase 9 adds section names the DOCS-05 remainder (full per-RPC failure table, write-latency model, key-prefix convention) as pending so nothing is claimed complete"
    requirement: "APIS-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS (anchors '## Consumer library territory', 'consumer-library territory', 'ADR 0007', '14.7', 'not a server feature', '## What Phase 9 adds', 'DOCS-05', 'per-RPC failure table', 'write-latency model', 'key-prefix convention')"
        status: pass
    human_judgment: false
  - id: D9
    description: "TestAPIDoc is the drift guard: it reads the repo file (not a fixture) and fails when a mandated section or heading literal is missing; both tasks produced machine-valid RED records before their GREEN"
    requirement: "APIS-01, APIS-02, APIS-03, APIS-04, APIS-05, APIS-06, APIS-07"
    verification:
      - kind: unit
        ref: "gsd_run check tdd-red-evidence red-evidence/06-09-task1.json -> RED_EVIDENCE_OK (missing 'read ../../docs/API.md'); 06-09-task2.json -> RED_EVIDENCE_OK (missing '## Errors')"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestAPIDoc$' -v -> PASS on the final tree"
        status: pass
      - kind: build
        ref: "gofmt -l cmd/sup-kv/api_docs_test.go -> no output; go vet ./cmd/sup-kv/ -> exit 0"
        status: pass
    human_judgment: false
duration: 5min
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 09: API Contract Document Summary

**`docs/API.md` now states the phase-owned client contract — the public `api/supkvapi/v1` proto, the honest write-time-set wait, the two List modes with the position token, the total path-free error table, the durability-gated Watch feed, the full-rollback snapshot load, the explicit transport/health envelope and the consumer-library last-read-value boundary — and `TestAPIDoc` pins every mandated statement so a deleted section or renamed heading fails the normal test gate.**

## Performance

- **Duration:** 5 min (284 s)
- **Started:** 2026-11-23T01:34:08Z
- **Completed:** 2026-11-23T01:38:52Z
- **Tasks:** 2
- **Files modified:** 4 created (1 doc, 1 test, 2 RED records)
- **Commits:** 2 measured from ledger `2a87fc9..6ddcc07`

## Accomplishments

- `docs/API.md` (308 lines) opens with `# sup-kv API` and the trusted-LAN pointer to `docs/OPERATIONS.md`, then carries the phase-owned sections in one house voice:
  - **Public proto** — proto file, `supkvapi.v1`, the Go import path `github.com/Djarvur/sup-kv/api/supkvapi/v1`, Go package `supkvapi`, the seven RPCs, `make proto`, and the always-on reflection path.
  - **wait_for_replication** — the write-time peer set as the promise set, the zero-known-peers `(false, 0, 0)` answer, the outside-Ack exclusion, the `Superseded` fast path (20 ms design target), the caller-bounded timeout with the ten-times clamp, and the aggregate report.
  - **List pagination** — the frozen legacy all-keys keys-only listing versus the bounded records page, the RawURL-base64 position token (not a snapshot), the 256 maximum, the 6 MiB serialized-byte budget as an upper bound, and the empty-token end signal.
  - **Errors** — one row per mapper constant and List constant in first-match order, the handler validation rows, the ordering rationale, the logged-never-returned detail rule and the no-server-path rule.
  - **Watch** — current-in-memory snapshot, durable live feed released after the dump confirmation, exactly once, deletions as events, version order, the documented asymmetry, ResourceExhausted resubscription and the Unavailable shutdown status.
  - **Snapshots** — full rollback with fresh versions, rollback tombstones for absent live keys, no version burn for already-tombstoned keys, the broadcast, and the non-linearizable caveat.
  - **Transport and health** — the 2 m/20 s/30 s keepalive triple, the derived `MaxRecvBytes`/`MaxSendBytes` ceilings with their two anchor values, and the one-way `NOT_SERVING` rule.
  - **Consumer library territory** — last-read-value as consumer-library work (ADR 0007 / 14.7), not a server feature.
  - **What Phase 9 adds** — the DOCS-05 remainder named as pending.
- `cmd/sup-kv/api_docs_test.go` — `TestAPIDoc` with an anchor table of one literal per mandated statement, read from `../../docs/API.md`; both tasks produced compile-clean RED records (`gsd_run check tdd-red-evidence` → `RED_EVIDENCE_OK` ×2) before their GREEN.
- Plan-level manual read-through: every section was checked against the acceptance criteria of plans 06-02 through 06-08 (the zero-peer release, the List mode resolution and budget floor, the mapper order, the hub confirmation gate and shed policy, the rollback union and skip rules, and the limits/health values) before the SUMMARY was written.

## TDD Evidence

| Task | RED record | RED observation | Target |
|------|------------|-----------------|--------|
| 1 | `red-evidence/06-09-task1.json` | `TestAPIDoc` failed reading `../../docs/API.md: no such file or directory` | `TestAPIDoc` |
| 2 | `red-evidence/06-09-task2.json` | the extended table failed on the first new anchor, `## Errors` (missing) | `TestAPIDoc` |

Both records passed `gsd_run check tdd-red-evidence` (`RED_EVIDENCE_OK`, reason `target_test_failed`).

## Task Commits

1. **Task 1: public proto, honest wait and List pagination contract** — `fe93c75` (docs)
2. **Task 2: errors, Watch, snapshots, transport and the consumer zone** — `6ddcc07` (docs)

**Plan metadata:** the final `docs(06-09)` commit.

## Files Created/Modified

- `docs/API.md` — the phase-owned API contract, nine sections, every statement bounded by what the code ships.
- `cmd/sup-kv/api_docs_test.go` — `TestAPIDoc`: the anchor table (README-style presence guard with the honest-scope comment) reading the repo file.
- `.planning/phases/06-api-and-semantics/red-evidence/06-09-task1.json` — the missing-document RED record.
- `.planning/phases/06-api-and-semantics/red-evidence/06-09-task2.json` — the missing-section RED record.

## Decisions Made

- The guard reads the repo file rather than a fixture, and its doc comment states the honest scope: anchors prove presence and heading stability, not the truth of each sentence (that is the phase read-through's job).
- The error table is presented in mapper order with the rationale written down, so a future reorder is a visible contract change; the handler validation rows sit beside it so a client sees every constant message.
- The doc names every limit as implemented (caller-bounded wait, non-linearizable rollback, current-memory snapshot) instead of claiming a stronger guarantee — the T-06-28 mitigation.
- The Phase 9/DOCS-05 remainder is an explicit section, not a silent gap; the document claims only what it states.
- All seven APIS checkboxes stay pending: `requirements.ready-ids` reports every one blocked by the #2388 shared-ID gate (06-13 also declares them), so the phase gate closes them.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The phase-owned contract is documented and drift-guarded. Two plans remain in Phase 6: **06-10** (the surrounding docs/EXAMPLES, docs/04-MULTI-NODE, docs/SPEC-DEVIATIONS and AGENTS updates) and **06-13** (the phase gate). No blockers; the only residual is the deliberate Phase 9/DOCS-05 remainder named in `docs/API.md`.

## Self-Check: PASSED

- Every created file is present: `docs/API.md`, `cmd/sup-kv/api_docs_test.go`, both RED records, this SUMMARY.
- Both task commits are ancestors of HEAD: `fe93c75` (Task 1), `6ddcc07` (Task 2).
