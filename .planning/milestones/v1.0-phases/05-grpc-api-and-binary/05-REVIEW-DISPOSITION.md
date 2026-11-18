---
phase: 05
review: 05-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "Gate resolver can flip health back to SERVING mid-drain — apiCancel sits after the metrics shutdown that widens the race window"
  - id: IN-01
    severity: info
    disposition: open
    title: "mapError's status.FromError passthrough branch is unreachable — no internal/app error can be a status error"
  - id: IN-02
    severity: info
    disposition: open
    title: "Serve-error exit path skips the final Flush and the graceful drain"
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "wait_for_replication can never report acked=true against the real mirror"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "wait_timeout_ms converted without range validation — int64 overflow"
open: 3
total: 5
recorded: 2026-11-18T10:57:57.826Z
---

# Phase 05: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| CR-01 | critical | fixed | 05-REVIEW-FIX.md (not in the current review) |
| WR-02 | warning | fixed | 05-REVIEW-FIX.md (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
