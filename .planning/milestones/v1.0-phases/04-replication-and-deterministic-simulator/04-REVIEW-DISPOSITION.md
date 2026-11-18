---
phase: 04
review: 04-REVIEW.md
titles: json
findings:
  - id: WR-03
    severity: warning
    disposition: open
    title: "`release-snapshot` — the \"mandatory pre-tag gate\" — fails on any machine without the external `syft` binary"
  - id: WR-04
    severity: warning
    disposition: open
    title: "go.mod tool-directive additions sit permanently outside the vulnerability gate's reach"
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "Adapter.NotifyLocalWrite blocks forever once Run has exited and the notify buffer fills"
  - id: IN-01
    severity: info
    disposition: fixed
    title: "NotifyLocalWrite pre-check leaves a narrow nil-return window during Run's exit"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "SyncResponseRate above 1e9 silently disables the egress rate limiter"
  - id: IN-02
    severity: info
    disposition: fixed
    title: "Restore marks the engine dirty unconditionally, forcing a redundant dump on a no-op replay"
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "applyRemote Acks — and supersedes pending tracking with — a record the store rejected"
open: 2
total: 7
recorded: 2026-11-18T10:42:09.510Z
---

# Phase 04: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-03 | warning | open | - |
| WR-04 | warning | open | - |
| WR-01 | warning | fixed | 04-REVIEW-FIX.md (not in the current review) |
| IN-01 | info | fixed | fix: NotifyLocalWrite is one select over notify+done with a re-check after a won send (the review's own suggested shape) — the pre-check/enqueue TOCTOU nil-return window is closed, the post-Run error is deterministic; regressions TestNotifyLocalWrite_afterDone + TestNotifyLocalWrite_afterRunReturns (not in the current review) |
| WR-02 | warning | fixed | fix: normalized() clamps SyncResponseRate to the 500/s default outside the domain (0, 1e9] (maxSyncResponseRate cap) — the floor(1s/rate) spacing stays non-zero, the limiter never silently off; regression TestSync_responseRateClamp pins both boundary sides (not in the current review) |
| IN-02 | info | fixed | fix (merged with 03 IN-02): a snapshot equal to the store's current state is a whole-restore no-op — no replay write, no dirty, the final flush replies nil without a dump; any difference replays in full with fresh-version dominance intact; regression TestEngine_redundantDirty (not in the current review) |
| CR-01 | critical | fixed | 04-REVIEW-FIX.md (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
