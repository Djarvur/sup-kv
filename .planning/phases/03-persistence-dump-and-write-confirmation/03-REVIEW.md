---
phase: 03-persistence-dump-and-write-confirmation
reviewed: 2026-11-14T19:19:18Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
  necessity: 2
status: issues_found
---

# Phase 3: Code Review Report (incremental re-review)

**Reviewed:** 2026-11-14T19:19:18Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Incremental re-review of the two phase-03 files that changed since the last
review (base 617951e, 2026-11-03): the delta is `Engine.ApplyRemote` (Merge +
markDirty + non-blocking `changed` nudge, no waiter), the `Engine.Flush`
shutdown seam, and three new tests plus the explicit `synctest.Sleep` advance
on the no-waiter dump paths (`TestFlush_remoteDirtiness`, `TestFlush_closed`,
`TestFlush_clean`, `TestEngine_applyRemoteTriggersDump`). The CURRENT state of
both files was re-audited against phase 03's contract; cross-references were
read in `internal/app/storage/store.go`, `internal/app/version/version.go`,
`internal/app/persist/codec.go`, `internal/port/dump.go`,
`internal/persistfile/dumpfile.go` and `internal/persistfile/snapshot.go` to
verify every seam claim. `go test -race -count=1 ./internal/app/persist/`
passes.

Contract verification results (all hold):

- **Write confirmation.** The cover discipline is sound: `await`
  (engine.go:394-401) raises `lastLocal` to ≥ `ver` under the same mutex hold
  that appends the waiter; `runDump` (engine.go:267-274) captures
  `cover = lastLocal` and the waiter list under one hold, then takes
  `Records()` after — so every captured waiter's record is provably inside the
  dumped snapshot (program order writer: store write → await; `version.Compare`
  is `bytes.Compare`, a total order). `ApplyRemote` and `Restore`'s replays
  register no waiter and cannot perturb the cover.
- **Retry cadence.** `failWaiters` re-dirties from a fresh epoch
  (engine.go:311-326); `rearm` restarts the debounce timer one interval later.
  The stale-epoch deadline timer surviving a failed dump (rearm never resets a
  non-nil deadline) forces at most one extra back-to-back attempt per MaxWait
  window — earlier, never later than the bound, never a tight loop.
- **Flush semantics.** Clean engine replies nil with no save (flushDump,
  engine.go:187-195); the exit sweep resolves stranded waiters and buffered
  flush requests with `ErrClosed`, `close(runDone)` releases flush senders
  parked on the full `flushCh` buffer and senders whose request landed after
  the sweep's drain (engine.go:550-574) — the prior CR-01 fix holds, and the
  two new exit-race tests exercise it. `Flush` inherits the identical
  contract.
- **Exactly-once resolution.** `sweepClosed`/`failWaiters`/`confirmWaiters`
  take disjoint waiter lists under `mu`, write `err` before `close(done)`;
  `w.err` reads happen-after the close. No path resolves a waiter twice.
- **New-code integration.** `ApplyRemote` returns before the debounced dump
  (no waiter — pinned by `TestEngine_applyRemoteTriggersDump`), passes
  `ErrValueTooLarge` through without dirtying, and its `changed` nudge wakes an
  idle dumper whose timers are all nil — the lost-nudge hazard is closed
  because the nudge is sent after the state change and `rearm` re-reads state
  under `mu` after consuming it (condition-variable discipline, correct).
  Snapshot names forwarded by `Snapshot`/`Restore` are validated at the
  adapter (charset + dot-specials + length, persistfile/snapshot.go:96-106) —
  no traversal via the engine seam.

No Critical findings this run. One provable necessity finding (an
impossible-state guard the code's own comment concedes is unreachable), one
judgment-call necessity finding, and one quality item on no-op dirty-marking.
The necessity rubric (`$HOME/.zcode/gsd-core/references/necessity-review.md`)
was loaded and applied; the go-ultimate skill index was consulted (project
AGENTS.md deviations — depguard boundary, vendored lint config, justified
`//nolint` — are all respected by the diff: no new imports outside the module,
no new suppressions).

## Warnings

### WR-01: Impossible-state guard — confirmWaiters' above-cover keep branch cannot fire

**File:** `internal/app/persist/engine.go:293-296`
**Category:** necessity
**Issue:** The `if w.ver.Compare(cover) > 0 { e.waiters = append(e.waiters, w); continue }` branch defends against a waiter above the captured cover, a state provably unreachable in this repo: (1) the only registration site is `await` (engine.go:394-401), which raises `e.lastLocal` to ≥ `ver` under the same `mu` hold that appends the waiter; (2) `lastLocal` is monotone — the sole write site only ever raises it, never resets (verified by grep: lines 394-396 are the only assignment); (3) `runDump` captures `cover = e.lastLocal` and the waiter list atomically under one `mu` hold (engine.go:267-272); (4) `version.Compare` is `bytes.Compare` over a fixed `[16]byte` — a total, transitive order (internal/app/version/version.go:63-65). Therefore every captured waiter satisfies `ver ≤ cover` and the branch never executes — the code's own comment concedes "unreachable in practice". It runs on every confirmation but defends against nothing in the current codebase.
**Fix:** Removal is a scope decision, not a mechanical fix — if the invariant ever broke, removing the branch would turn the belt into a *false* nil confirmation (an above-cover waiter claimed on-disk when it is not), strictly worse than the current re-queue. Either remove the branch together with a written invariant assertion (comment or test asserting `ver.Compare(cover) <= 0` for all captured waiters), or keep it as-is: the existing "unreachable in practice, but stranded never" comment already documents the justification. Do not strip it silently (the `necessity` category marker exists so the fixer does not).

## Info

### IN-01: Already-defended-elsewhere — Open's Decode-error fallback is dead in the shipped stack

**File:** `internal/app/persist/engine.go:111-116`
**Category:** necessity
**Issue:** `Open` guards a `Decode(dump)` error after `dumps.Load(Parse)` returned ok, falling to the empty start. That state cannot occur with the shipped components: `Load` returns only generations whose bytes passed the caller-supplied oracle (port/dump.go:23-30; persistfile/dumpfile.go:74-86 skips any candidate with `parse(data) != nil`), and the oracle `Parse` is exactly `Decode`'s error channel (`func Parse(dump []byte) error { _, _, err := Decode(dump); return err }`, codec.go:158-161) — a pure function over the same bytes, so `Parse(dump) == nil` implies `Decode(dump)` returns a nil error. The guard is live only against a `port.DumpStore` implementation that violates its documented contract (e.g., a contract-breaking test fake). Rated Info rather than Warning because the bytes originate from files — a trust boundary — and the port is an explicit substitution seam, so the belt is defensible judgment, not a proven lapse.
**Fix:** Acceptable to keep as seam insurance; alternatively tighten the comment to say the branch is unreachable for contract-honoring adapters rather than implying a real "generation-level decode error" path (the current doc comment at engine.go:98-101 presents it as a live case).

### IN-02: ApplyRemote dirties the engine on no-op merges — redundant dump on stale/equal redelivery

**File:** `internal/app/persist/engine.go:429`
**Issue:** `ApplyRemote` calls `markDirty()` unconditionally whenever `Merge` returns a nil error, but `storage.Merge` also returns nil when nothing was installed: an equal-version redelivery (full no-op, store.go:239-240) and a losing LWW merge (conflict reported, nothing stored, store.go:237-238). A stale or duplicate replicated record therefore schedules a full dump of unchanged state, defeating the "clean engine does not re-dump" economy for redelivery-heavy streams. Correctness is unaffected — dumps are idempotent full snapshots and the debounce amortizes bursts — so this is a quality item, not a bug (performance as such is out of scope; flagged because it contradicts the clean-engine discipline the phase documents). The engine cannot distinguish apply from no-op today: `conflicts` is empty both for a fresh-key install and for an equal redelivery. `Restore`'s unconditional `markDirty` (engine.go:497) shares the class but is forced there (replay outcomes are indistinguishable), while `await` only dirties on real writes.
**Fix:** If the redundant saves ever matter, extend the store seam in its owning phase to report whether a `Merge` installed (e.g., an `applied bool` return), and gate `markDirty` on it; otherwise record the behavior as accepted in the ApplyRemote doc comment so future readers do not assume dirty implies state change.

---

_Reviewed: 2026-11-14T19:19:18Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
