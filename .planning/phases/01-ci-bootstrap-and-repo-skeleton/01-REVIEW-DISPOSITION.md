---
phase: 01
review: 01-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "Dependabot will silently break the golangci-lint local/CI version-parity invariant"
  - id: IN-01
    severity: info
    disposition: open
    title: "AGENTS.md quotes an outdated go directive value"
  - id: IN-02
    severity: info
    disposition: fixed
    title: ".claude/CLAUDE.md asserts branch protection that is not yet applied"
  - id: IN-03
    severity: info
    disposition: open
    title: "Third-party actions pinned by mutable major tags, not commit SHAs"
  - id: IN-04
    severity: info
    disposition: open
    title: "The no-I/O boundary exempts test files and does not cover x/sys"
  - id: IN-05
    severity: info
    disposition: open
    title: "\"Apply command above\" points the wrong direction — the Apply command is below"
  - id: IN-06
    severity: info
    disposition: open
    title: "CI's schema gate and CI's lint run use two separately-pinned binaries"
open: 5
total: 7
recorded: 2026-11-01T00:00:00.000Z
---

# Phase 01: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | fixed | PR #12 |
| IN-01 | info | open | - |
| IN-02 | info | fixed | PR #20 — .claude/CLAUDE.md deferral parenthetical; confirmed by incremental review 85fa4dc |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |
| IN-06 | info | open | Increment 2 (PR #26 follow-up review, 2026-11-01) — new finding |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
