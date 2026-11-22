---
phase: 05-persistence
fixed_at: 2026-11-22T18:15:55Z
review_path: .planning/phases/05-persistence/05-REVIEW.md
iteration: 1
findings_in_scope: 8
fixed: 7
skipped: 1
status: partial
---

# Phase 5: Code Review Fix Report

**Fixed at:** 2026-11-22T18:15:55Z
**Source review:** `.planning/phases/05-persistence/05-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 8 (WR-01, WR-02, IN-01…IN-06; `fix_scope: all`)
- Fixed: 7
- Skipped/deferred: 1 (IN-04, deferred to Phase 7 by the orchestrator; the
  permitted docs-only precondition note was applied)

## Fixed Issues

### WR-01: Package contract contradicted the shipped retry cadence and omitted the third write outcome

**Files modified:** `internal/app/persist/doc.go`
**Commit:** 99b72f3
**Applied fix:** Rewrote the two stale paragraphs of the package contract. The
timing paragraph now states the shipped bounded ladder (armed at the failure
instant, 100 ms doubling to a 5 s cap, reset by a success) instead of "one
debounce interval later". The write contract now names all three resolutions —
nil (on disk), `ErrDumpFailed` (failed dump, record readable in memory) and
`ErrUnconfirmed` at `Config.ConfirmTimeout` (record in memory, not yet
durable) — and states that `Flush`, `FlushTimeout`, `Snapshot` and `Restore`
are bounded by the same confirmation deadline.

### WR-02: Flush-forced dumps bypassed the retry ladder and kept a stale armed rung

**Files modified:** `internal/app/persist/scheduler.go`,
`internal/app/persist/snapshot.go`, `internal/app/persist/backoff_test.go`,
`internal/app/persist/engine_test.go`, `docs/OPERATIONS.md`
**Commits:** 58d022f (fix), 4e08869 (strict-lint follow-up on the same tests)
**Applied fix:** Chose the review's first mechanism — a flush-forced attempt
is refused while a ladder rung is pending — rather than the alternative
rate-limited-log variant:
- `scheduler.flushDump` keeps the forced dump for a dirty engine whose
  ladder is idle (the D-10 snapshot seam), but when a rung is pending and a
  classified failure exists it returns that last failure instead of calling
  `runDump`. No extra `store.Records()` copy, no extra
  `supkv_dumps_failures_total` increment and no extra `OnDumpFailure` ERROR
  line per client request; the node rate is the ladder's rate.
- `scheduler.handleDumpFailure` records the classified failure under the
  scheduler mutex; `resetRetry` clears it on success.
- `scheduler.rearm` now also receives the instant the armed retry channel was
  armed for and replaces the channel when the ladder's `retryAt` advanced, so
  a forced failure can no longer leave the next attempt firing at the old
  (stale) instant. This is defense-in-depth: with the refusal above a forced
  attempt cannot advance a pending ladder, but the mismatch is now
  impossible by construction.
- `docs/OPERATIONS.md` states beside the ladder paragraph that a snapshot's
  forced flush does not bypass the rung.
- Regression test `TestSnapshot_forcedFlushRespectsRetryLadder` drives five
  `Snapshot` calls inside each of two armed rungs against a perpetually
  failing port and asserts: attempts stay at 1 then 2 (no client-driven
  copies), `OnDumpFailure` fires exactly once per attempt (not per request),
  and the retries land at failure+100 ms and failure+300 ms, proving the
  ladder's next attempt time still advances.
- `TestEngine_snapshotFlushForcesDump` was adapted to the deliberate
  rung-aware contract: it now proves the forced dump with a dirty engine that
  has no rung pending (a remote merge), with the failure-path coverage in the
  new regression test.
**Status:** fixed: requires human verification (scheduler state-machine and
rate behavior; see the logic-bug limitation).

### IN-01: SPEC-DEVIATIONS rows cited pre-split `engine.go` line ranges

**Files modified:** `docs/SPEC-DEVIATIONS.md`
**Commit:** 23fa460
**Applied fix:** Re-pointed exactly the eight named rows — S-ALG-CU-04,
S-ALG-CU-07, S-ALG-CU-08, S-ALG-REP-02, S-PERS-05, S-PERS-10, S-PERS-11,
S-PERS-12 — at the post-split role files (`scheduler.go`, `confirm.go`,
`snapshot.go`, `facade.go`) with the current line ranges, and refreshed the
stale `metrics.go` citation inside S-ALG-CU-08. No other row changed (the
diff is eight rows, eight replacements).

### IN-02: `applyRemote` doc claimed the pre-merge version probe fixed the reply verdict

**Files modified:** `internal/app/repl/core.go`
**Commit:** 24ecf99
**Applied fix:** Replaced the sentence with the actual mechanism: the
`Version` probe selects the stale/identical/merge path, and the reply verdict
is re-derived from the stored record read after the merge because an
equal-version tie-break win or loss cannot be named by the version bytes.

### IN-03: Derived `{class="other"}` counter could decrease between scrapes

**Files modified:** `internal/app/persist/engine.go`,
`internal/app/persist/scheduler.go`, `internal/app/persist/backoff_test.go`,
`internal/metrics/metrics.go`, `internal/metrics/metrics_test.go`
**Commit:** d098bd0
**Applied fix:** Counted the unclassified class explicitly instead of deriving
it: `dumpStats.other` is incremented in `handleDumpFailure`'s
`FailureOther` branch, surfaced as `persist.DumpStats.Other`, and rendered
directly by `dumpFamilies` (the `Errors - DiskFull - Permission` clamp is
gone). Every published class series is now monotone by construction even for
a scrape landing between the error-total increment and the class increment.
`TestEngine_dumpFailureClasses` now asserts `Other == 1`; the new
`TestMetrics_dumpOtherMonotone` replays the exact two-scrape window
(`Errors: 1, Other: 0` then `Errors: 1, Other: 1`) and asserts the rendered
`{class="other"}` series never decreases — the old derived form would have
rendered 1 then 0.
**Status:** fixed: requires human verification (counter derivation/order;
see the logic-bug limitation).

### IN-05: `mergeRecords`'s `ErrValueTooLarge` skip guarded an impossible state

**Files modified:** `internal/app/persist/loader.go`
**Commit:** dcc546a
**Applied fix:** Removed the special case and the now-unused `errors` import.
The loop ignores the merge error explicitly (`_, _, _ = store.Merge(...)`)
with the totality documented: `Open` decodes with
`storeCfg.ValueLimit()` and `NewStore` normalizes through the same helper, so
every decoded record is within the limit and the store's only merge error
cannot fire. The store remains the sole authority on the bound.

### IN-06: Legacy no-census truncation limitation was absent from the runbook

**Files modified:** `docs/OPERATIONS.md`
**Commit:** fa454e3
**Applied fix:** Added a sentence to the "Load outcomes" section: a
pre-`record_count` generation has no completeness census, so a line-boundary
truncation of it loads as authoritative; taking a snapshot (or letting one
confirmed write rotate the main generation) after upgrading produces a
generation that carries the census. All DOCS-02 anchors remain present.

## Skipped Issues

### IN-04: `EncodeTo` accepts a value limit above the codec's own line bound

**File:** `internal/app/persist/codec.go:41,59-62,126-153`
**Reason:** Deferred to Phase 7 by the orchestrator: config validation owns
the value-limit ceiling and the shipped binary already rejects the value via
`checkDatagramCeiling` in `main`. No codec bound or behavior was changed. The
permitted docs-only step was applied in 642742a: the `persist` package doc now
names the precondition (`valueMaxBytes` must stay at or below
`maxDumpValueBytes`, the datagram ceiling the line bound is sized from) and
records that the library-level guard is Phase 7's.
**Original issue:** `EncodeTo` can emit a line longer than `maxDumpLineBytes`
for a caller-supplied `valueMaxBytes` above the datagram ceiling, which the
same package's `Decode` rejects as a generation-level fault; only the
binary's startup ceiling check prevents it today.

## Verification

- `workflow.use_worktrees` is `false` in `.planning/config.json`, so all edits,
  commits and gate runs happened in the main checkout (no isolated worktree).
- Per-finding: affected code re-read (Tier 1) and `gofmt`/`go vet` on the
  touched packages (Tier 2) before each commit.
- `make lint` (`go tool golangci-lint config verify` +
  `go tool golangci-lint run ./...`): green after the 4e08869 lint follow-up.
- `make test` (`go test -race -cover ./...`): green — all packages `ok`,
  including `internal/app/persist` (95.2%) and `internal/metrics`. The known
  `TestCore_keyCountCostFlat` and `TestScenario_dumpNotify` flakes did not
  fire in this run, so no re-run was needed.
- Frozen artifacts untouched: `scripts/red-expectations.tsv`,
  `internal/stand/stand_red_test.go`,
  `internal/persistfile/red_value_limit_test.go`, `.golangci.yml`,
  `go.mod`, `go.sum`.

---

_Fixed: 2026-11-22T18:15:55Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
