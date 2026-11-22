---
phase: 05-persistence
reviewed: 2026-11-22T17:56:27Z
depth: deep
files_reviewed: 52
files_reviewed_list:
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - cmd/sup-kv/operations_test.go
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - docs/adr/0012-equal-version-tiebreak.md
  - internal/app/internal/simnet/simnet.go
  - internal/app/persist/backoff_test.go
  - internal/app/persist/bench_test.go
  - internal/app/persist/codec.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/confirm.go
  - internal/app/persist/doc.go
  - internal/app/persist/encode_stream_internal_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_blocking_test.go
  - internal/app/persist/engine_test.go
  - internal/app/persist/errors.go
  - internal/app/persist/facade.go
  - internal/app/persist/fuzz_dump_test.go
  - internal/app/persist/load_stats_internal_test.go
  - internal/app/persist/load_stats_test.go
  - internal/app/persist/loader.go
  - internal/app/persist/scheduler.go
  - internal/app/persist/snapshot.go
  - internal/app/persist/timeout_internal_test.go
  - internal/app/persist/timeout_test.go
  - internal/app/repl/core.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_dump_test.go
  - internal/app/storage/merge_test.go
  - internal/app/storage/prop_small_space_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/metrics/metrics.go
  - internal/metrics/metrics_test.go
  - internal/persistfile/dumpfile.go
  - internal/persistfile/dumpfile_test.go
  - internal/persistfile/fsops.go
  - internal/persistfile/fsops_test.go
  - internal/persistfile/load_outcome_test.go
  - internal/persistfile/open_seed_test.go
  - internal/persistfile/saveerror.go
  - internal/persistfile/saveerror_test.go
  - internal/persistfile/snapshot.go
  - internal/persistfile/sweep_test.go
  - internal/port/dump.go
  - internal/stand/persist_test.go
  - internal/udprepl/decode_negative_test.go
  - scripts/red-expectations.tsv
findings:
  critical: 0
  warning: 2
  info: 6
  total: 8
  necessity: 1
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-11-22T17:56:27Z
**Depth:** deep
**Files Reviewed:** 52
**Status:** issues_found

## Summary

Reviewed the full Phase-5 scope (commit range `4f81fd7..ea75d77`) against locked decisions
D-01…D-12 and the phase's honesty surfaces: the single-source value limit, the census codec,
the four load outcomes with evidence preservation and the orphan sweep, the dirty start plus the
Phase-4 New-1 nudge, the bounded retry ladder with classification, the streaming encode, the
confirmation/flush/shutdown deadlines with the single-claim waiter protocol, the deterministic
equal-version tie-break in both the store and the replication core, the five-role engine split and
the injected clock/timer seam. The necessity rubric
(`$HOME/.zcode/gsd-core/references/necessity-review.md`) was read before classifying findings.

**The phase's core claims check out.** No critical defect was found.

- **Single-source limit.** `storage.ValueMaxBytes` is the only production literal (grep for
  `61440` in non-test Go yields `store.go:21` alone); `Config.ValueLimit()` (`store.go:131-136`)
  is used by `persist.Open` before the store exists (`loader.go:43`) and by `NewStore`
  (`store.go:140-146`), and `Store.ValueMaxBytes()` feeds the encoder and the snapshot decoder —
  the three sites cannot disagree. Exactly-limit records encode and decode; a shrunken limit is a
  per-record `ValueLimit` discard, never a generation fault (`maxDumpLineBytes` is
  configuration-independent, `codec.go:59-62`).
- **Load outcomes / evidence.** `persistfile.Load` (`dumpfile.go:127-175`) classifies Prompt/
  Main/Backup/Unusable, records a synthetic `fs.ErrNotExist` main failure for an absent-main
  backup, and preserves every rejected candidate via a collision-probed `.corrupt-<UTC>` rename
  (`dumpfile.go:206-233`) before any save; `.prev` survives the first post-backup rotation because
  the corrupt main has left the rotation path. The sweep selects only by the `.tmp~` marker
  (outside the snapshot-name charset), is age-bounded, and excludes the configured main, `.prev`
  and `.corrupt-*` names (`dumpfile.go:252-310`).
- **Honesty of resolution.** The single-claim protocol (`confirm.go:98-250`) is race-free: capture
  never claims, every resolver and the timeout claim under the mutex before writing `err`/closing,
  a timed-out write is always `ErrUnconfirmed` (or the resolver's genuine answer when it won the
  race), and a hung `SaveStream` cannot block writes (await deadline), snapshots (`Snapshot`'s
  bounded flush), or shutdown (`FlushTimeout` plus the bounded `engineDone` select,
  `main.go:955-972`). A timed-out write is still notified to replication before `Unavailable`
  (`server.go:214-221`, `:261-270`, `:442-443`).
- **Tie-break.** `store.join` is the total order `(version, deleted-desc, data)`
  (`store.go:386-415`), shared with the core through `storage.RecordsEqual`
  (`core.go:1494-1501`, `:1527-1536`); an incoming win Acks, a stored win answers with the stored
  record and no Ack, and the engine's dirty decision follows `MergeChanged`'s `changed`
  (`facade.go:40-50`). The commutativity/associativity/idempotence properties hold under
  different-content equal versions (`prop_small_space_test.go`'s collision store).
- **Repository rules.** No `.golangci.yml`, `go.mod` or `go.sum` drift; the depguard boundaries
  hold (no `os`/`syscall`/`net`/`google.golang.org` imports in `internal/app`, port stdlib-only);
  every production `//nolint` names its linter with a reason; `scripts/red-expectations.tsv` flips
  exactly the two P-01 rows and both frozen tests (`stand_red_test.go`,
  `red_value_limit_test.go`) are byte-unchanged against `4f81fd7`.

**Gates run for this review.** `go build ./...` and `go vet ./...` clean; `go tool
golangci-lint run ./...` exit 0; `SUPKV_RED_TESTS=1 go test -race ./internal/persistfile/ -run
'^TestRed_P01_valueLimitEngine$'` PASS; `go test -race -count=1` green for `cmd/sup-kv`,
`internal/app/{clock,persist,storage}`, `internal/app/repl` sub-packages other than the one
failure below, `internal/grpcapi`, `internal/metrics`, `internal/persistfile`, `internal/stand`,
`internal/udprepl`. The one failure in the full-suite run is the known-accepted
`TestCore_keyCountCostFlat` wall-clock ratio flake (`internal/app/repl`, Phase 8 owner; measured
2.16 > 2.0 under the parallel suite) and is not re-raised, nor are the accepted
`TestScenario_dumpNotify` virtual-time harness hang, the `TestLog_*` port-probe flake, the
unbounded peer table, or the Phase-4 deferrals (`New-1` is verified fixed: `markDirty` itself
nudges, `confirm.go:139-147`).

The eight findings below are documentation, observability and defense-in-depth defects; none
changes a durability or convergence verdict.

## Narrative Findings (AI reviewer)

### Warnings

#### WR-01: The package contract in `persist/doc.go` now contradicts the shipped behavior — the retry cadence is a 100 ms→5 s ladder, and the write contract omits the third resolution outcome

**File:** `internal/app/persist/doc.go:27-34` and `:11-15`
**Issue:** `doc.go` is the package's primary API contract and was edited by this phase (the
five-role paragraph was added) while two statements the phase invalidated were left in place:

1. "After a failed dump the next attempt starts one debounce interval later — the re-armed timer,
   never a tight retry loop". The shipped behavior is the bounded exponential ladder
   (`scheduler.go:17-24`, `:225-241`, `:246-255`): the next attempt is armed at the failure
   instant plus 100 ms, doubling to a 5 s cap, and `rearm` deliberately does not arm the
   debounce/deadline while a rung is pending (`scheduler.go:158-168`). One debounce interval is
   50 ms — the doc describes the pre-phase behavior.
2. "A write blocks inside Set, SetIfAbsent or Delete until a dump containing it completed or
   failed — nil confirms … ErrDumpFailed reports the failed dump". Since this phase there is a
   third, distinct resolution: `ErrUnconfirmed` at `Config.ConfirmTimeout`
   (`engine.go:26-34`, `confirm.go:112-129`, `errors.go:14-20`), and the package doc never names
   `ConfirmTimeout`, `FlushTimeout` or the bounded-flush contract — the phase's headline
   PERSIST-08 guarantee. A reader of the package doc would conclude a hung dumper parks writes
   forever, which is exactly what this phase fixed.

**Fix:** rewrite the two paragraphs, e.g. "After a failed dump the next attempt is armed at the
failure instant plus a bounded ladder (100 ms doubling to a 5 s cap), reset by a success" and add
the third outcome: "… completed (nil), failed (ErrDumpFailed), or did not complete within
Config.ConfirmTimeout (ErrUnconfirmed: the record is in memory and not yet durable); Flush,
FlushTimeout, Snapshot and Restore are bounded by the same deadline".

#### WR-02: Flush-forced dump attempts bypass the bounded retry ladder — a snapshot RPC loop under a persistent dump failure drives one attempt, one full state copy and one ERROR line per request, and can shorten an armed rung

**File:** `internal/app/persist/scheduler.go:126-128` (`flushCh` → `flushDump`), `:140-145`
(`flushDump` ⇒ `runDump` whenever dirty), `:158-177` (`rearm`); `internal/app/persist/confirm.go:229-250`
(`failWaiters` re-dirties every failed attempt); `internal/app/persist/snapshot.go:32-38`, `:98`;
`internal/grpcapi/server.go:317-322`, `:341-346`; `cmd/sup-kv/main.go:586-588`
**Issue:** D-05 promises "the retry ladder is bounded and never busy-loops", and the runbook
states the ladder governs the retry rate. That holds for engine-driven attempts only. Because
`failWaiters` re-dirties the engine after every failure, a persistently failing dump path leaves
the engine permanently dirty, and each `SnapshotCreate`/`SnapshotLoad` RPC calls `flush` →
`flushDump`, which — deliberately — calls `runDump` immediately, ignoring `retryDelay`/`retryAt`.
One client request therefore produces: one forced dump attempt, one `store.Records()` full state
copy (`scheduler.go:203`), one `supkv_dumps_failures_total` increment, and one `slog.Error` via
`OnDumpFailure` (`main.go:586-588`). The attempt/error rate is client-controlled (bounded only by
the failed attempt's own latency, e.g. a fast `CreateTemp` EACCES), which is the log-flood surface
on the path this phase owns: the operator gets ERROR lines at the client's request rate precisely
when the disk is broken. The trust model (`docs/OPERATIONS.md:18-21`) makes this a trusted-client
amplification rather than an unauthenticated attack, and a second, smaller defect shares the same
root: when a flush-forced attempt fails while a rung is already armed, `rearm` receives a non-nil
`retry` and keeps it (`scheduler.go:163-166`), so the next attempt fires at the previously armed
(old) instant rather than the newly computed `retryAt` — the ladder's recorded growth is not the
ladder actually followed.

**Fix:** keep the forced dump but take the rate out of the client's hands, e.g. refuse a
flush-forced attempt while a rung is pending and the last failure was for the same dump
(`flushDump` returns the last classified error instead of re-attempting), or rate-limit
`OnDumpFailure` (one line per ladder rung plus a suppressed-count attribute) while still counting
every attempt; and in `rearm`, when the flush path re-armed the ladder (`retryAt` changed after
`retry` was armed), replace the channel instead of keeping the stale one. Document the chosen
behavior in OPERATIONS beside the ladder paragraph.

### Info

#### IN-01: The engine split invalidated a large set of `docs/SPEC-DEVIATIONS.md` line citations; several now point past the end of `engine.go`

**File:** `docs/SPEC-DEVIATIONS.md:109`, `:112`, `:113`, `:126`, `:163`, `:168`, `:169`, `:170`, `:171`
(eight rows cite `internal/app/persist/engine.go` at lines beyond its current 290), plus `:94`,
`:95`, `:114`, `:162`, `:165`, `:202`, `:205`, `:208`, `:210`, `:242` (in-range lines whose code
moved to `loader.go`/`scheduler.go`/`facade.go`/`snapshot.go`)
**Issue:** `engine.go` is now 290 lines after the five-role split. `S-PERS-05` cites
`engine.go:292-368`, `S-PERS-10` `:544-550`, `S-PERS-11` `:572-630`, `S-PERS-12` `:552-587`,
`S-ALG-REP-02` `:513-517`, `S-ALG-CU-04` `:411-472`, `S-ALG-CU-07` `:374-389`, `S-ALG-CU-08`
`:317-323` — every one of those ranges lies past EOF and cannot be followed. The other rows cite
in-range lines whose code is now in another file (e.g. `S-PERS-04`/`S-PERS-07`'s scheduling
policy → `scheduler.go:107-177`; `S-ALG-ST-01`/`S-FAIL-10`'s composition root →
`loader.go:35-74`; `S-PERS-11`/`S-PERS-12`'s replay → `snapshot.go:121-196`). The registry exists
to be verifiable; the phase updated only its own eight rows.
**Fix:** re-point the affected rows at the post-split files/lines (a mechanical pass), or add a
one-line note to the registry preamble that Phase 5's split moved the `persist` citations and
that rows are file-anchored, not line-frozen.

#### IN-02: The `applyRemote` doc still claims the pre-merge `Version` probe fixes the reply verdict, but the verdict is now re-derived from post-merge `Lookup`s

**File:** `internal/app/repl/core.go:1475-1476`
**Issue:** "The probe is one point Version read before ApplyRemote, so the merge can never change
the verdict mid-decision." That was true when the only decision was the version comparison; after
this phase the reply is decided by a second read after the merge (`core.go:1530-1536`), so the
verdict *is* re-derived from state the probe did not see (and must be — that is the tie-break
fix). A future maintainer reading the sentence could believe the tail's `Lookup` is redundant and
remove it.
**Fix:** replace the sentence with the actual mechanism: the `Version` probe selects the
stale/identical/merge path, and the reply follows the stored record read after the merge because
an equal-version tie-break win or loss cannot be named by the version bytes.

#### IN-03: The derived `{class="other"}` failure counter can decrease between scrapes, which reads as a counter reset to `rate()`

**File:** `internal/metrics/metrics.go:402-410`
**Issue:** `other` is computed as `Errors - (DiskFull + Permission)` from four independent atomic
loads. `recordDump` increments `Errors` (`scheduler.go:269-275`) and only afterwards does
`handleDumpFailure` increment the class counter (`scheduler.go:225-241`), so a scrape landing in
that window renders `other=1` and the next scrape renders `other=0`: a Prometheus counter going
down, which `rate()`/`increase()` interpret as a reset. The comment already reasons about the race
("clamped so a scrape racing a dump's counter updates never renders a negative sample") but the
decrease itself is not addressed. The same window inflates the rendered class split for one
scrape.
**Fix:** count the unclassified class explicitly — `dumpStats.unclassified` incremented in
`handleDumpFailure`'s `FailureOther` branch — and render it directly (monotonic by construction),
keeping the clamp unnecessary; or derive `other` only from `Total`/`Errors` consistently and
accept the one-scrape skew in the documented comment.

#### IN-04: `EncodeTo` accepts a `valueMaxBytes` the codec's own reader rejects, producing an unreadable generation outside `main`'s ceiling check

**File:** `internal/app/persist/codec.go:41`, `:59-62`, `:126-153`; `:240-241`, `:349-351`
**Issue:** `maxDumpLineBytes()` is a fixed format property sized from the 65 507-byte datagram
ceiling (`maxDumpValueBytes`), while `EncodeTo` writes any record whose `Data` is within the
*caller-supplied* `valueMaxBytes`. A `valueMaxBytes` above roughly 65.6 KB (max-escaped
`storage.MaxKeyBytes` key plus overhead) makes `EncodeTo` emit a line longer than
`maxDumpLineBytes()`, which the same package's `Decode` then rejects as `errDumpLineTooLong` — a
generation-level fault, so the next start falls back to `.prev` and the rotation discipline
eventually displaces the good generation. The shipped binary cannot reach this because
`checkDatagramCeiling` (`main.go:541-563`) caps the flag below the ceiling, and the codec's
comment says "the transport adapter enforces it"; but `persist` is the package that both writes
and reads the format, and its encode side does not enforce the bound its decode side relies on.
**Fix:** reject the impossible configuration at the encode boundary — in `EncodeTo`, `if
valueMaxBytes > maxDumpValueBytes { return fmt.Errorf("persist: configured value limit %d exceeds
the dump format maximum %d", …) }` (and/or add it to `Config.normalized()`'s contract), so the
guarantee does not depend on the binary's argument validation.

#### IN-05: `mergeRecords`'s `ErrValueTooLarge` skip is a belt over an impossible state

**File:** `internal/app/persist/loader.go:97-104` (skip at `:100-102`)
**Category:** necessity
**Issue:** Subtype: already-defended-elsewhere. `Open` decodes with `limit :=
storeCfg.ValueLimit()` (`loader.go:43`) and `Decode` discards every record whose `Data` exceeds
that limit as `decodeValueLimit` (`codec.go:349-351`), so every record handed to `mergeRecords`
has `len(Data) <= limit`. `NewStore` normalizes `ValueMaxBytes` through the same `ValueLimit()`
(`store.go:131-146`), so `MergeChanged`'s own check (`store.go:319-321`) is false for every one
of them. The branch therefore cannot fire for any decoded record; the doc calls it "the belt over
that bound", i.e. a second defense for a state the decoder has already excluded. (The encode-side
tripwire at `codec.go:136-139` is deliberately a fail-loud invariant assertion, not this pattern;
it is not flagged.)
**Fix:** drop the special case and make the loop's totality explicit — either ignore the merge
error with a comment naming the store as the sole authority (`_, _, _ = store.Merge(key, rec)`,
with the invariant documented), or assert it loudly (`if err != nil { panic(...) }` is not
desirable in startup code, so a plain ignore plus one line of doc is the honest form).

#### IN-06: The legacy no-census limitation on PERSIST-03 is documented only in the codec comment, not where operators read about load outcomes

**File:** `internal/app/persist/codec.go:230-234` (`Decode`'s doc: "A legacy v1.0 dump with no
`record_count` carries no completeness census — a truncation exactly at a line boundary of such a
dump is the documented compatibility limitation"); `docs/OPERATIONS.md:94-116` (the "Load
outcomes" section)
**Issue:** The phase's PERSIST-03 acceptance is "a file truncated at a line boundary is
recognized as incomplete". That is true for dumps this binary writes, but the documented
compatibility path (a pre-upgrade generation with no `record_count`) accepts a line-boundary
truncation silently, and such a `LoadMain` starts clean — so the truncated state is authoritative
and the first write rotates the incomplete main over a possibly good `.prev`. The only place this
caveat is written down is a Go doc comment inside `internal/app/persist`; the operator-facing
runbook's load-outcome section never mentions it (grep for `legacy`/`census` in `docs/` returns
nothing).
**Fix:** add one sentence to the "Load outcomes" section: a generation written before this
format's `record_count` field has no completeness census, so a line-boundary truncation of it is
undetectable — take a snapshot (or let one confirmed write rotate it) after upgrading, and the
next generation carries the census.

---

_Reviewed: 2026-11-22T17:56:27Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
