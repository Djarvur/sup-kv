---
phase: 02
review: 02-REVIEW.md
titles: json
findings:
  - id: IN-01
    severity: info
    disposition: fixed
    title: "OnChange doc misstates the record a \"subsequent Get returns\" for tombstone installs"
  - id: IN-02
    severity: info
    disposition: fixed
    title: "OnChange contract does not state the re-entrancy deadlock boundary"
  - id: IN-03
    severity: info
    disposition: fixed
    title: "Duplicated subtest body between TestStore_endToEnd and TestStore_Set"
open: 0
total: 3
recorded: 2026-11-14T18:56:15.670Z
---

# Phase 02: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| IN-01 | info | fixed | fixed — OnChange doc names both surfaces: the record a subsequent Records() snapshot holds, and that a subsequent Get returns only when the record is live; a tombstone install fires the tap exactly like any other change (a delete IS a change) while Get answers ErrNotFound (plan 06-02, D-15) |
| IN-02 | info | fixed | fixed — OnChange doc states the re-entrancy boundary: the tap is invoked synchronously from the write paths under the store's locking discipline, the write lock is held, Go's sync.RWMutex is not reentrant, and even a read-only Get self-deadlocks (plan 06-02, D-15) |
| IN-03 | info | fixed | fixed — the TestStore_Set copy of "a second set replaces the value under a strictly greater version" removed (plus its now-unused cmpSame constant); the TestStore_endToEnd copy kept byte-identical as the layer-wiring probe; suite green under -race (plan 06-02, D-15) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
