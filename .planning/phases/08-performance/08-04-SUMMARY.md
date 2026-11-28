---
phase: 08-performance
plan: 04
subsystem: replication-replay-seam
tags: [performance, allocation, replay, snapshot-load, version-only, perf-05, tdd]
requires:
  - phase: 04-protocol
    provides: the paced replay seam (LocalReplayNotify, Adapter.NotifyReplayedWrite) and the record-send pacer
  - phase: 05-persistence
    provides: persist.RestoredRecord and SnapshotLoad's replay loop
provides:
  - a version-only record on both replay notify hops (the gRPC seam and the adapter enqueue)
  - two fail-first gates pinning the record shape, plus an allocation gate at the adapter hop
  - seam documentation that states the version-only contract at both hops
affects: [08-05, PERF-05, snapshot-load, internal/udprepl, internal/grpcapi]
actuals:
  tokens: 3602
  tasks: 2
  commits: 2
plan_head_before: 817d1dd455c4c699f44e035cf6215e9c30ef00e0
plan_head_after: bde702c1ced3b4e84e5c60972cc07303398f5c36
tech-stack:
  added: []
  patterns:
    - "Fail-first allocation gate: testing.Benchmark(...).AllocedBytesPerOp() as an assertion (RED 65536 -> GREEN 0 bytes/op), logging the passing measurement"
    - "Version-only seam record: a notification that carries exactly the field the consumer reads, with the test naming every zeroed field"
key-files:
  created:
    - internal/grpcapi/replay_seam_internal_test.go
    - internal/udprepl/replay_notify_internal_test.go
  modified:
    - internal/grpcapi/server.go
    - internal/udprepl/loop.go
key-decisions:
  - "The paced replay seam carries storage.Record{Version} and nothing else at both hops; the nil-ReplayNotify fallback is untouched because the live LocalNotify contract still carries the full record."
  - "cloneLocalRecord stays on the live NotifyLocalWrite path only, and its doc now says so, so a future editor cannot quietly reuse it for the replay hop."
  - "The third clone — persist.replaySnapshotKey's RestoredRecord.Data — is left untouched: it is the report's own payload, not one of the two discard-path copies 04 New-3 names."
  - "The allocation gate is an assertion, not a report: 60 KiB record, ceiling 1 KiB/op, failure text names the seam and the byte figure; the passing run logs the measured 0 bytes/op."
patterns-established:
  - "When a seam's consumer reads one field, the seam's test asserts the other fields are zero by name, so the next consumer of the payload fails loudly instead of silently receiving garbage."
requirements-completed: [PERF-05]
coverage:
  - id: D1
    description: "notifyReplayedWrite hands the paced ReplayNotify seam a version-only record: Version preserved, Data empty, Deleted false, Checksum zero"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestReplaySeam' (TestReplaySeamCarriesVersionOnlyRecord, driven through NewServer and the real notifyReplayedWrite)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Adapter.NotifyReplayedWrite enqueues a version-only localWrite with replay=true on the real notify channel"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ -run '^TestNotifyReplayedWrite' (TestNotifyReplayedWriteCarriesVersionOnly reads the enqueued localWrite)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Allocation gate at the adapter hop: one replay notification for a 60 KiB record allocates under 1 KiB/op (RED 65536, GREEN 0 bytes/op)"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/udprepl/ -run '^TestNotifyReplayedWriteAllocationBound$' (logs '0 bytes/op for a 61440-byte record'; RED recorded 65536)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live write path unchanged: cloneLocalRecord still serves NotifyLocalWrite and the nil-ReplayNotify fallback still rebuilds the full record; both seam docs state the version-only replay contract"
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ ./internal/udprepl/ (green, including TestServer_snapshotLoadReplayFallsBackToLive and TestReplayNotify_paced)"
        status: pass
      - kind: command
        ref: "grep -q 'version only' internal/grpcapi/server.go && grep -q 'version only' internal/udprepl/loop.go"
        status: pass
    human_judgment: false
duration: 7min
completed: 2026-11-23
status: complete
---

# Phase 08 Plan 04: Version-Only Replay Seam Summary

**[Substantive one-liner] The paced replay path now moves a 16-byte version instead of a 60 KiB record: both post-report `Data` clones are gone, an allocation gate pins the adapter hop at 0 bytes/op, and both seam contracts say why.**

## Performance
- **Duration:** 7min (408 s)
- **Started:** 2026-11-23T18:23:43Z
- **Completed:** 2026-11-23T18:30:31Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- `grpcapi.notifyReplayedWrite` now passes `storage.Record{Version: rec.Version}` to the paced `LocalReplayNotify` seam; the `bytes.Clone` and `storage.RecordChecksum` rebuild are deleted. The nil-seam fallback is byte-for-byte unchanged: it still routes through `notifyLocalWrite`, which rebuilds and checksums the full record for the live seam.
- `udprepl.Adapter.NotifyReplayedWrite` enqueues `localWrite{key: key, rec: storage.Record{Version: rec.Version}, replay: true}` with no `cloneLocalRecord`. The live `NotifyLocalWrite` path keeps its defensive clone, and `cloneLocalRecord`'s doc now scopes it to that path.
- The `LocalReplayNotify` type doc, `notifyReplayedWrite`'s method doc, `NotifyReplayedWrite`'s doc, `cloneLocalRecord`'s doc and `localWrite`'s doc all state the record shape and the reason: the core reads only `rec.Version`, and the paced record-send drain re-reads the stored record through `Lookup` before emitting.
- Two internal gates pin the contract: the gRPC gate drives the real `notifyReplayedWrite` through `NewServer` with a 60 KiB `persist.RestoredRecord` and asserts the exact version-only shape; the adapter gate reads the real notify channel and additionally asserts `replay == true`. Both failed on the pre-fix tree by field name (`Data = 61440 bytes`), and the allocation gate failed at `65536 bytes/op` for a 61440-byte record.
- The transient the review measured cannot recur: the only remaining `Data` allocation on a snapshot load is `persist.replaySnapshotKey`'s report payload, which the plan deliberately leaves untouched.

## Task Commits

1. **Task 1 RED: failing version-only replay gates** — `0ccbf04` (test)
2. **Task 2 GREEN: drop the discard-path replay clones** — `bde702c` (feat)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `internal/grpcapi/replay_seam_internal_test.go` — the in-package gate over the real `notifyReplayedWrite` and the paced seam: key, version, empty Data, false Deleted, zero Checksum, with failure messages naming the field and the seam.
- `internal/udprepl/replay_notify_internal_test.go` — the in-package enqueue-shape gate (real notify channel, `replay == true`) and the allocation gate (`testing.Benchmark`, 60 KiB record, ceiling 1024 bytes/op, logs the passing measurement).
- `internal/grpcapi/server.go` — `LocalReplayNotify` doc and `notifyReplayedWrite` rewritten for the version-only contract; the clone/checksum rebuild deleted from the paced branch only.
- `internal/udprepl/loop.go` — `NotifyReplayedWrite` enqueues a version-only record; `cloneLocalRecord` and `localWrite` docs scoped to the live path.

## Decisions Made

- **Version-only means version-only.** The record is `storage.Record{Version: rec.Version}` at both hops; `Data`, `Deleted` and `Checksum` stay zero. The gates assert each zeroed field separately so a future change that starts reading one fails with the field name.
- **The live contract is unchanged.** `notifyLocalWrite` still clones the author's buffer and recomputes the checksum; the nil-`ReplayNotify` fallback calls it exactly as before. Removing `cloneLocalRecord` would have broken the live seam's ownership rule, so it stays with a narrowed doc.
- **The third clone is out of scope by design.** `persist.replaySnapshotKey`'s clone into `RestoredRecord.Data` is the report's own payload; New-3 counts only the two copies made after the report and then discarded.
- **The gate asserts, the log reports.** The allocation gate fails on `AllocedBytesPerOp() >= 1024` and also logs the passing measurement (0 bytes/op), so the phase summary can quote the before/after without a separate benchmark run.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Lint gate] The new tests were reshaped for `noinlineerr` and `thelper`, and the gate logs its measurement**
- **Found during:** Task 2 verification (lint gate)
- **Issue:** `make lint` flagged two inline `if err := ...; err != nil` forms (`noinlineerr`) and the `testing.Benchmark` closure (`thelper` wants `b.Helper()` first), and the passing allocation gate printed no byte figure for the summary.
- **Fix:** Plain `err := ...` assignments, `b.Helper()` at the closure top, and a `t.Logf` that prints the measured bytes/op on success. No behavior or bound changed.
- **Files modified:** `internal/udprepl/replay_notify_internal_test.go`
- **Verification:** `make lint` exit 0 on the whole repo; both gates green under `-race`; the log shows `0 bytes/op for a 61440-byte record (ceiling 1024)`.
- **Commit:** `bde702c`

## TDD Gate Compliance

| Gate | Commit | Result |
|------|--------|--------|
| RED (Task 1) | `0ccbf04` test(08-04): add failing version-only replay seam gates | `RED_EVIDENCE_OK` on both records (`target_test_failed`): gRPC `Data = 61440 bytes`; adapter `Data = 61440 bytes` and `65536 bytes/op` |
| GREEN (Task 2) | `bde702c` feat(08-04): drop the discard-path replay clones | Version-only shapes pass; `0 bytes/op`; both packages green under `-race` |

The two RED evidence records, their console reports and exit files are under `.planning/phases/08-performance/red-evidence/20261123T182505Z/` (untracked, matching the phase's red-evidence convention).

## Issues Encountered

- None blocking. `make test` (`go test -race -cover ./...`) was green on the first run, including the two wall-clock ratio gates in `internal/app/repl/bench_test.go` that 08-05 will replace — no 08-05-class failure surfaced here.
- The frozen files were verified byte-identical after the work: `internal/grpcapi/red_gate_test.go`, `internal/stand/stand_red_test.go`, `scripts/red-expectations.tsv` (md5s unchanged from before Task 1). The red-runner sed targets (`core.go`'s pacing condition, `store.go`'s `out[key] = rec.Version`) were not touched.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. Both hops pass the version they read and the gates assert the full record shape; no placeholder, TODO or skipped test was left behind.

## Next Phase Readiness

Ready for 08-05. The replay path moves a version and two integers; the live path keeps its defensive copy; the seam contracts are stated where the next editor reads them. PERF-05 remains open in `REQUIREMENTS.md` because 08-05 still owns the paired absent/live wall-clock gates and the perf-target wiring.

## Self-Check: PASSED

- Files: `internal/grpcapi/replay_seam_internal_test.go`, `internal/udprepl/replay_notify_internal_test.go`, `.planning/phases/08-performance/08-04-SUMMARY.md` all found.
- Commits: `0ccbf04` and `bde702c` both ancestors of HEAD.
