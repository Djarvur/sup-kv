---
phase: 05-persistence
updated: 2026-11-22T19:05:00Z
total: 13
---

# Code Review Disposition — Phase 05

One row per finding from `05-REVIEW.md` plus the re-verification rounds. Edit the **Disposition**
column by hand when triaging (`open` / `fixed` / `skipped` / `deferred`); put the reason in
**Source**. This ledger is rebuilt from REVIEW.md on each review run — decisions recorded here are
preserved.

Re-verification verdicts (same reviewer, read-only, overlay probes): WR-01/WR-02/IN-01/IN-02/IN-03/
IN-05/IN-06 FIXED; IN-04 DEFERRED to Phase 7 (docs-only precondition note applied); the three
defects the first fix pass introduced (N-1/N-2/N-3) FIXED in a second pass and re-verified; no new
material issue remains (two cosmetic notes, no fix warranted).

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | fixed | `99b72f3` (+ nit `57155c1`): `doc.go` states the three write resolutions, the 100 ms→5 s ladder, and `FlushTimeout` bounded by its own `d`. Re-verified against the code. |
| WR-02 | warning | fixed | `58d022f` (+ lint `4e08869`): flush-forced dumps answer the last classified failure while a rung is pending (no attempt/copy/class/ERROR), `rearm` replaces a stale rung; `TestSnapshot_forcedFlushRespectsRetryLadder` proves five snapshots per rung add zero attempts. Re-verified. |
| IN-01 | info | fixed | `23fa460` + `f617366`: the eight named rows and the two residual citations (`S-ALG-CU-07`, `S-ALG-REP-02`) now point at the current role files/symbols; union diff is exactly 8 rows with no other row touched. Re-verified. |
| IN-02 | info | fixed | `24ecf99`: comment describes the post-merge `Lookup` verdict derivation. Re-verified. |
| IN-03 | info | fixed | `d098bd0` + `a128e64`: explicit `DumpStats.Other` counter (monotone by construction); the faithful three-scrape test fails on the old derived form and passes now (proved with an overlay). Re-verified. |
| IN-04 | info | deferred | `642742a` (docs only): package doc names the `valueMaxBytes <= maxDumpValueBytes` precondition; **Phase 7's config validation** owns the library-level guard. No codec behavior changed. |
| IN-05 | info | fixed | `dcc546a`: unreachable `ErrValueTooLarge` skip removed with the totality documented. Re-verified (behavior unchanged). |
| IN-06 | info | fixed | `fa454e3` + `0fcb80b`: runbook states the legacy no-census truncation limitation and correctly says a snapshot of a clean legacy `Main` load stays uncensused. Re-verified against the code. |
| IN-07 (necessity) | info | fixed | Same root as IN-05 — duplicated limit enforcement removed. |
| N-1 | warning | fixed | `691f3e4`: flush requests are origin-aware — `Flush`/`FlushTimeout` (shutdown seam) always force one attempt past a pending rung, `Snapshot`/`Restore` defer as `ErrDumpFailed` (RPC maps Unavailable); `TestFlush_forcesAttemptPastPendingRung` proves a recovered disk lands the dirty state; the four contract statements match. Re-verified. |
| N-2 | warning | fixed | `a128e64`: `TestMetrics_dumpOtherMonotone` uses the faithful `{Errors:1}` → `{DiskFull:1}` → `{Other:1}` window and fails on the old derived form under an overlay. Re-verified. |
| N-3 | warning | fixed | `0fcb80b`: runbook corrected — a snapshot of a clean legacy `Main` load copies the generation byte-for-byte with no census; only a confirmed write or dirty-engine flush produces a censused main. Re-verified. |
| N-4 | info | skipped | Cosmetic only, re-verification explicitly says no fix warranted: `Restore` re-wraps the already-`ErrDumpFailed` refusal (errors.Is and RPC mapping unaffected) and one doc sentence is contorted. Recorded, not changed. |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).

**Post-fix gates (2026-11-22):** `make lint` clean; `go vet ./...` clean; `go test -race -cover ./...`
green on the committed tree (no accepted flake fired); `SUPKV_RED_TESTS=1 bash scripts/red-runner`
green — 7/7 rows, 0 errors, both mutants killed (evidence `20261122T181914Z`); frozen tests, TSV,
`.golangci.yml`, `go.mod`/`go.sum` byte-unchanged.
