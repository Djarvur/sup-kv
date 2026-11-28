---
phase: 05-persistence
fixed_at: 2026-11-22T18:43:34Z
review_path: .planning/phases/05-persistence/05-REVIEW.md
iteration: 2
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 5: Code Review Fix Report — Iteration 2

**Fixed at:** 2026-11-22T18:43:34Z
**Source review:** `.planning/phases/05-persistence/05-REVIEW.md` (re-verification of the
iteration-1 fix pass)
**Iteration:** 2

**Summary:**
- Defects in scope: 5 (N-1, N-2, N-3, IN-01 residual, WR-01 nit)
- Fixed: 5
- Skipped: 0

The iteration-1 fixes (WR-02 ladder refusal, IN-03 explicitly counted `other` class, IN-05
merge totality, and the docs rows) remain in force. This pass corrects the three defects the
re-verification found in and around them and finishes the two documentation residuals. The
iteration-1 report is preserved in commit `0ac1bfb`.

## Fixed Issues

### N-1: A pending retry rung made the shutdown flush skip the mandated final dump

**Files modified:** `internal/app/persist/flush.go` (new),
`internal/app/persist/confirm.go`, `internal/app/persist/scheduler.go`,
`internal/app/persist/snapshot.go`, `internal/app/persist/engine.go`,
`internal/app/persist/doc.go`, `internal/app/persist/backoff_test.go`
**Commit:** 691f3e4
**Applied fix:**
- A flush request now carries its origin: `flushRequest{snapshot bool}` (new shared file
  `internal/app/persist/flush.go`). `Flush`/`FlushTimeout` send an explicit request;
  `Snapshot`/`Restore` send a snapshot-originated one. The new file keeps the iteration-1
  cited declaration anchors in `confirm.go`/`scheduler.go`/`engine.go` unchanged.
- `scheduler.flushDump` refuses only snapshot-originated requests while a rung is pending
  (`scheduler.pendingFailure`) and wraps that refusal as
  `fmt.Errorf("%w: %w", ErrDumpFailed, lastErr)`, so `SnapshotCreate` maps to
  `codes.Unavailable` instead of `codes.Internal` with the adapter's path in the message.
  Explicit flushes always call `runDump`, even with the ladder armed: a transient failure
  followed by a recovered disk now still ends with the dirty state on disk.
- WR-02's client-flood bound is intact: `TestSnapshot_forcedFlushRespectsRetryLadder` still
  sends five snapshots inside each of two rungs and asserts the attempt count stays 1 then 2
  (no client-driven copies, no extra `OnDumpFailure` lines); its refusal assertions now also
  require `errors.Is(err, persist.ErrDumpFailed)`.
- New regression test `TestFlush_forcesAttemptPastPendingRung`: (a) one save fails —
  `Set` returns `ErrDumpFailed`, attempts 1, completed saves 0; (b) the failure is cleared;
  (c) `FlushTimeout(5s)` (the shutdown call) is issued; (d) attempts is 2, completed saves
  is 1, the decoded dump carries the dirty record, and a following `Flush` on the now-clean
  engine adds no save.
- The four now-false contract statements were rewritten: `engine.go`'s `Flush`,
  `FlushTimeout` and `flush` docs now state that explicit flushes force one attempt past a
  pending rung and that snapshot flushes defer with a wrapped `ErrDumpFailed`; `doc.go`'s
  ladder paragraph qualifies the refusal to snapshot requests and names the explicit-flush
  exception, and its snapshot paragraph states that a pending rung aborts the snapshot with
  `ErrDumpFailed` and nothing copied.
**Status:** fixed: requires human verification (shutdown/flush state machine; see the
logic-bug limitation).

### N-2: `TestMetrics_dumpOtherMonotone` could not fail on the old derived form

**Files modified:** `internal/metrics/metrics_test.go`
**Commit:** a128e64
**Applied fix:** Replaced the two-scrape window (which passed under the old derived
implementation) with the faithful three-scrape one:
`{Total:1, Errors:1}` → `{Total:1, Errors:1, DiskFull:1}` → `{Other:1}`. The first renders
0 under the counted series, the second stays 0 (the derived form drops 1→0), and the third
settles `other` at 1 (asserted). Verified by an overlay probe that compiles the test against
`d098bd0^:internal/metrics/metrics.go`: the old derived form FAILS at scrape 1 with
`rendered class="other" = 0 after 1`, while the committed implementation PASSES.
**Status:** fixed.

### N-3: The runbook claimed a snapshot always produces a censused generation

**Files modified:** `docs/OPERATIONS.md`
**Commit:** 0fcb80b
**Applied fix:** Corrected the legacy-census paragraph: a snapshot taken right after a clean
`Main` load copies the legacy generation byte-for-byte and therefore carries no census;
only a confirmed write (or a flush of a dirty engine, which rewrites the main generation)
produces a censused main. All DOCS-02 anchors remain green (`TestOperationsDoc` in
`cmd/sup-kv` passed in the full suite).
**Status:** fixed.

### IN-01 residual: two remaining stale SPEC-DEVIATIONS citations

**Files modified:** `docs/SPEC-DEVIATIONS.md`
**Commit:** f617366
**Applied fix:** Re-pointed exactly the two citations the re-verification named:
- `S-ALG-CU-07`: `server.go:299-300` → `internal/grpcapi/server.go:433-454` (`mapError`
  maps `ErrDumpFailed`/`ErrUnconfirmed` to `Unavailable`).
- `S-ALG-REP-02`: `core.go:539-542` → `internal/app/repl/core.go:1489-1539` (`applyRemote`);
  `store.go:237,243,268` → `internal/app/storage/store.go:318-338`
  (`MergeChanged`/`Merge`) and `:386-415,425-436` (`join`/`install`).
No other row changed (the diff is two rows, four lines).

**Changed-row set across both fix passes (10 row edits over 8 distinct rows):**
- Pass 1 (eight rows): `S-ALG-CU-04`, `S-ALG-CU-07`, `S-ALG-CU-08`, `S-ALG-REP-02`,
  `S-PERS-05`, `S-PERS-10`, `S-PERS-11`, `S-PERS-12`.
- Pass 2 (two row edits, both re-edits of rows already in the set): `S-ALG-CU-07`,
  `S-ALG-REP-02`.
- Union (eight distinct rows): the pass-1 list above. Every other row is byte-identical to
  its pre-pass-1 content. The pass-1 line anchors in `scheduler.go` (`109`, `149`, `176`,
  `221`, `248`, `296`), `engine.go` (`flush` at `268`) and `confirm.go` (`await` at `98`,
  `confirmWaiters` at `207`, `failWaiters` at `229`) are unchanged by this pass.
**Status:** fixed.

### WR-01 nit: the package contract overstated the flush bound

**Files modified:** `internal/app/persist/doc.go`
**Commit:** 57155c1
**Applied fix:** "Flush, FlushTimeout, Snapshot and Restore are bounded by the same
confirmation deadline" became "FlushTimeout is bounded by its own `d`; Flush, Snapshot and
Restore are bounded by `Config.ConfirmTimeout`".
**Status:** fixed.

## Skipped Issues

None.

## Verification

- `workflow.use_worktrees` is `false` in `.planning/config.json`, so every edit, commit and
  gate run happened in the main checkout (no isolated worktree), exactly as in iteration 1.
- Per-fix Tier 1 (re-read of the touched symbols) and Tier 2 (`gofmt`, `go build ./...`,
  `go vet` on the touched packages) before each commit.
- Targeted runs on the committed tree:
  `go test -race -count=1 ./internal/app/persist -run 'TestFlush|TestSnapshot|TestEngine_backoff|TestEngine_dumpFailureClasses|TestDumpStats'`
  and `go test -race -count=1 ./internal/metrics -run 'TestMetrics_dumpOtherMonotone|TestMetrics_values|TestMetrics_families'`
  — PASS.
- N-2 overlay probe (proves the test has teeth): with
  `go test -count=1 -overlay=<tmp>/overlay.json ./internal/metrics -run '^TestMetrics_dumpOtherMonotone$'`
  where the overlay replaces `internal/metrics/metrics.go` with `d098bd0^:internal/metrics/metrics.go`,
  the run FAILS at scrape 1 (`0 after 1`); the same test without the overlay PASSES.
- `make lint` (`go tool golangci-lint config verify` +
  `go tool golangci-lint run ./...`): exit 0 on the committed tree.
- `make test` (`go test -race -cover ./...`): exit 0 on the committed tree; every package
  `ok`, including `internal/app/persist` (95.5%), `internal/metrics` (69.4%) and
  `internal/app/repl` (93.1%, 179.7 s). No known-accepted flake fired, so no re-run was
  needed.
- Frozen artifacts untouched: `scripts/red-expectations.tsv`,
  `internal/stand/stand_red_test.go`, `internal/persistfile/red_value_limit_test.go`,
  `.golangci.yml`, `go.mod`, `go.sum`.
- New file created and documented: `internal/app/persist/flush.go` (the shared
  `flushRequest` type; no behavior lives there).

---

_Fixed: 2026-11-22T18:43:34Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
