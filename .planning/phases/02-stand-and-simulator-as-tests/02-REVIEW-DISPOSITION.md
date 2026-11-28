---
phase: 02-stand-and-simulator-as-tests
updated: 2026-11-21T15:10:00Z
total: 14
---

# Code Review Disposition — Phase 02

One row per finding from `02-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**. This ledger is rebuilt
from REVIEW.md on each review run — decisions recorded here are preserved.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | red-runner drops package-level `fail` events (panics/timeouts/TestMain). Fix: count package-level fail events that carry no matching row as errors, or require a matching `fail` event per expected row AND zero unexpected package fails. Runner script fix (Phase 3 follow-up or quick task). |
| WR-02 | warning | open | `Cluster.Write` discards `store.Set`'s error then broadcasts a fabricated record — the exact P-01 oversize class. Fix: mirror `Delete`'s error handling, return the error, broadcast nothing on failure, plus a simnet test. |
| WR-03 | warning | open | TEST-13 meta-test walks the module root incl. the in-tree `.gomodcache`, can pass vacuously. Fix: scope the walk to the repo's own packages (skip dirs containing go.mod/go.sum or named .gomodcache/vendor). |
| WR-04 | warning | open | `scheduled.yml` seeds input check unanchored + raw echo into `::warning::` (newline injection). Fix: full-match `^[0-9]+$` with an upper bound, echo via a sanitized numeric variable only. |
| WR-05 | warning | open | Private port window mitigation is per-process; two packages and `freeTCPPort` (ephemeral pool) reintroduce pick-to-bind races. Fix target: bind-and-claim pattern shared across packages (Phase 3 flake work) or document the residual in TEST-11's audit. |
| WR-06 | warning | open | `race-soak` excludes `./internal/stand/` — TEST-11's ×3 bar never repeats the moved socket scenarios. Fix: include the package (with its reserved window this is safe) or record the exclusion rationale in the scheduled job. |
| IN-01 | info | open | Bench smoke prints p50/p99 but never asserts them. Phase 8 (PERF) owns thresholds; leave for the perf phase. |
| IN-02 | info | open | `scnRunID` collides with `simRunID` for `d == 9`. Cosmetic collision; fix in the simnet scenario ID generator. |
| IN-03 | info | open | `stand_nondarwin_test.go` name vs `!linux` tag mismatch. Rename or retag; harmless. |
| IN-04 | info | open | TEST-15 small-space constants duplicated across two test packages. Consider sharing via testutil in Phase 3. |
| IN-05 | info | open | Scheduled "latest evidence" step can print a previous run's committed summary. Fix: restrict to red-evidence dirs created in this run. |
| IN-06 | info | open | Nightly chaos job never verifies the fresh-seed count actually ran. Add a grep assertion on the seed count log line. |
| IN-07 | info | open | Fuzz corpora have no drift guard against their generators. Add a regenerate-and-compare test mirroring the chaos corpus guard. |
| IN-08 | info | open | Size-gate test measures a hand-copied encoder, not the production one. Use replproto's encoder directly or note the parity test. |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).
