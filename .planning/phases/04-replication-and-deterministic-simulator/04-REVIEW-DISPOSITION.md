---
phase: 04
review: 04-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "applyRemote Acks — and supersedes pending tracking with — a record the store rejected"
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "Adapter.NotifyLocalWrite blocks forever once Run has exited and the notify buffer fills"
  - id: IN-01
    severity: info
    disposition: open
    title: "Core.ownHandle is write-only dead state"
  - id: IN-02
    severity: info
    disposition: open
    title: "simnet Cluster.Write ignores the store Set error and logs a phantom write"
open: 2
total: 4
recorded: 2026-11-06T20:39:38.629Z
---

# Phase 04: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | fixed | 04-REVIEW-FIX.md |
| WR-01 | warning | fixed | 04-REVIEW-FIX.md |
| IN-01 | info | open | - |
| IN-02 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
