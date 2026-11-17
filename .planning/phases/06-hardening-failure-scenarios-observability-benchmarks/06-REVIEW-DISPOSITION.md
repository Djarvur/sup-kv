---
phase: 06
review: 06-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "Ephemeral-port pick-to-bind race hard-fails the log-pair probe (observed flake during review)"
  - id: WR-02
    severity: warning
    disposition: open
    title: "Metrics discard-class labels re-stringify the canonical tokens — silent contract-drift hazard"
  - id: WR-03
    severity: warning
    disposition: open
    title: "NewMux comment states a false mechanism — pprof/expvar DO register on DefaultServeMux on any import"
  - id: IN-01
    severity: info
    disposition: open
    title: "SentByType counts datagrams as sent before unicast resolution and before the socket write"
open: 4
total: 4
recorded: 2026-11-16T11:33:23.263Z
---

# Phase 06: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| IN-01 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
