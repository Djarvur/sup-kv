---
phase: 04
review: 04-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "Drained-row retention can publish a duplicate node id in Acked"
  - id: IN-01
    severity: info
    disposition: open
    title: "NotifyLocalWrite pre-check leaves a narrow nil-return window during Run's exit"
  - id: WR-02
    severity: warning
    disposition: open
    title: "SyncResponseRate above 1e9 silently disables the egress rate limiter"
  - id: IN-02
    severity: info
    disposition: open
    title: "Restore marks the engine dirty unconditionally, forcing a redundant dump on a no-op replay"
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "applyRemote Acks — and supersedes pending tracking with — a record the store rejected"
open: 4
total: 5
recorded: 2026-11-14T22:07:37.406Z
---

# Phase 04: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| IN-01 | info | open | - |
| WR-02 | warning | open | - (not in the current review) |
| IN-02 | info | open | - (not in the current review) |
| CR-01 | critical | fixed | 04-REVIEW-FIX.md (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
