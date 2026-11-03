---
phase: 03
review: 03-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: open
    title: "`flush()` can block forever when Run exits between the closed-check and the flushCh send — permanent goroutine hang on shutdown"
  - id: WR-01
    severity: warning
    disposition: open
    title: "`CreateSnapshot` can spuriously fail with `ErrNoDump` during a concurrent `Save`'s rotate window; the engine's concurrency-safety claim is incorrect"
  - id: IN-01
    severity: info
    disposition: open
    title: "`maxRecordBytes` literal duplicates the store's value limit — drift risk"
  - id: IN-02
    severity: info
    disposition: open
    title: "Engine is silently single-shot — `closed` is never reset after one Run exit"
  - id: IN-03
    severity: info
    disposition: open
    title: "Out-of-range `checksum` field collapses the whole generation instead of discarding one record"
  - id: IN-04
    severity: info
    disposition: open
    title: "`CreateSnapshot` creates the `.snapshots` directory even when it then fails with `ErrNoDump`"
open: 6
total: 6
recorded: 2026-11-03T00:17:22.234Z
---

# Phase 03: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | - |
| WR-01 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
