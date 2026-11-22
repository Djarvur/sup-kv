---
phase: 05-persistence
updated: 2026-11-22T18:05:00Z
total: 9
---

# Code Review Disposition — Phase 05

One row per finding from `05-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**. This ledger is rebuilt
from REVIEW.md on each review run — decisions recorded here are preserved.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | `internal/app/persist/doc.go` package contract contradicts shipped behavior (retry cadence; write contract omits `ErrUnconfirmed`/`ConfirmTimeout`). **Fix in phase:** doc correction. |
| WR-02 | warning | open | Flush-forced dump attempts bypass the retry ladder: under a persistent failure each `SnapshotCreate`/`SnapshotLoad` RPC drives one forced attempt, one full `Records()` copy and one ERROR line (client-controlled rate); a failed attempt leaves the stale armed rung. **Fix in phase:** route forced attempts through the ladder, bound attempts/ERRORs per RPC, add a regression test. |
| IN-01 | info | open | `docs/SPEC-DEVIATIONS.md` — eight citations point past the post-split `engine.go` line ranges. **Fix in phase:** repoint to the role files. |
| IN-02 | info | open | `internal/app/repl/core.go:1475-1476` comment still claims the pre-merge `Version` probe fixes the reply verdict. **Fix in phase:** comment correction. |
| IN-03 | info | open | Derived `{class="other"}` metric can decrease between scrapes (increment-order race), reading as a reset. **Fix in phase:** make the derived series monotone; add a metrics test. |
| IN-04 | info | deferred | `EncodeTo` can write a line above `maxDumpLineBytes` for a configured limit above the datagram ceiling; only `main`'s ceiling check prevents it in the shipped binary. **Phase 7 (config validation / value-limit catalog)** owns the library-level guard; the package doc names the precondition meanwhile. |
| IN-05 | info | open | Unreachable `ErrValueTooLarge` skip in `mergeRecords` (decoder applies the same limit first). **Fix in phase:** remove or justify. |
| IN-06 | info | open | Legacy no-census truncation limitation documented only in a Go comment, not the operator runbook. **Fix in phase:** runbook sentence; anchors stay green. |
| IN-07 (necessity) | info | open | Same root as IN-05 — necessity item: duplicated limit enforcement. **Fix in phase** together with IN-05. |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).
