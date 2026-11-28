---
phase: 01
review: 01-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "ADR 0005 claims a registry linkage its own target rows do not contain"
  - id: WR-02
    severity: warning
    disposition: open
    title: "review-repl-api.md quotes the non-public Spec, contradicting REVIEW §0's publication-safety claim"
  - id: WR-03
    severity: warning
    disposition: open
    title: "Repro test comments use stale ENG/RPL finding IDs that contradict the published reports"
  - id: IN-01
    severity: info
    disposition: open
    title: "S-HLC-07 status violates the registry's declared enum and overstates the decision level"
  - id: IN-02
    severity: info
    disposition: open
    title: "REVIEW §0's guard-mechanism description is stale against the committed repo"
  - id: IN-03
    severity: info
    disposition: open
    title: "test-audit.md appendix inventories evidence files that are not committed"
  - id: IN-04
    severity: info
    disposition: open
    title: "Engine repro test count drift, and one committed \"test\" has no assertion"
  - id: IN-05
    severity: info
    disposition: open
    title: "REVIEW §16 correspondence table is partial and hides an ID crossing"
  - id: IN-06
    severity: info
    disposition: open
    title: "G-xx references resolve in no committed document"
open: 9
total: 9
recorded: 2026-11-24T21:55:22.659Z
---

# Phase 01: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |
| IN-06 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
