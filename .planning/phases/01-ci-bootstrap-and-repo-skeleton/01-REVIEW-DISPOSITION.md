---
phase: 01
review: 01-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "Neither CI job sets a timeout-minutes bound — a wedged job holds its runner for GitHub's 6-hour default"
  - id: IN-07
    severity: info
    disposition: open
    title: "Three stale `buf.gen.yaml` references remain in .claude/CLAUDE.md after the file was deleted"
  - id: IN-08
    severity: info
    disposition: open
    title: "buf.yaml comment claims a single workspace-wide generate pass that no longer exists"
  - id: IN-01
    severity: info
    disposition: fixed
    title: "Signal handling is released before the D-19 shutdown runs — a second SIGTERM/SIGINT kills the process and forfeits the final write-confirmed dump"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "NewClock never clamps initialL to the 48-bit domain — the one unclamped input silently defeats the documented wrap protection"
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
open: 6
total: 10
recorded: 2026-11-14T22:07:37.347Z
---

# Phase 01: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | fixed | fixed — timeout-minutes 15 on job lint and 20 on job gate, additive keys only: job ids and every check-run context (the gate (${{ matrix.name }}) names, the ruleset required_status_checks contract) byte-identical (plan 06-02, D-15) |
| IN-07 | info | fixed | fixed in 4f76e26 — stale buf.gen.yaml references rewritten to buf.gen.*.yaml (review-day fix) |
| IN-08 | info | fixed | fixed in 4f76e26 — buf.yaml comment corrected: bare workspace generate is gone, make proto is the path (review-day fix) |
| IN-01 | info | fixed | fixed — serve() no longer releases signal ownership before shutdown: the armed SIGTERM/SIGINT channel spans serving and the D-19 sequence, a second signal during the GracefulStop drain takes the force path (Stop) and the final Flush still lands, exit 0; regression test TestShutdown_secondSignal in cmd/sup-kv/main_test.go (plan 06-02, D-15) |
| WR-02 | warning | fixed | fix 64f2d24 (branch fix/clock-review-warnings): NewClock clamps initialL into [0, maxL]; regression test TestClock_watermarkClampedIntoDomain (not in the current review) |
| IN-02 | info | fixed | PR #20 — .claude/CLAUDE.md deferral parenthetical; confirmed by incremental review 85fa4dc (not in the current review) |
| IN-03 | info | open | - (not in the current review) |
| IN-04 | info | open | - (not in the current review) |
| IN-05 | info | open | - (not in the current review) |
| IN-06 | info | open | Increment 2 (PR #26 follow-up review, 2026-11-01) — new finding (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
