---
phase: 05
review: 05-REVIEW.md
titles: json
findings:
  - id: IN-01
    severity: info
    disposition: open
    title: "mapError's status.FromError passthrough branch is unreachable — no internal/app error can be a status error"
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "wait_for_replication can never report acked=true against the real mirror"
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "handleAck doc contradicts the code — outside-wait-set Acks are counted"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "wait_timeout_ms converted without range validation — int64 overflow"
open: 1
total: 4
recorded: 2026-11-14T22:07:37.544Z
---

# Phase 05: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| IN-01 | info | open | - |
| CR-01 | critical | fixed | 05-REVIEW-FIX.md (not in the current review) |
| WR-01 | warning | fixed | 05-REVIEW-FIX.md (not in the current review) |
| WR-02 | warning | fixed | 05-REVIEW-FIX.md (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
