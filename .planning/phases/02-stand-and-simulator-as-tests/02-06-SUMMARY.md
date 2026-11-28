---
phase: 02-stand-and-simulator-as-tests
plan: 06
subsystem: testing
tags: [chaos-testing, deterministic-simulation, seed-corpus, convergence-polling, deletions, lww-crdt, test-01, d-06]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: "plan 02-03's simnet extensions — Cluster.Delete (fresh-version tombstone through NotifyLocalWrite into the write log) and the determinism contract the deletion mix rides"
provides:
  - "committed chaos seed corpus internal/app/repl/testdata/chaos-seeds.txt (16 seeds, one decimal int64 per line, no header)"
  - "TestChaos_seedCorpusGenerator: always regenerate-and-byte-compare drift check; the write is gated on a missing file or SUPKV_SIM_CORPUS_REGEN=1"
  - "TestChaos_convergence corpus iteration (corpus/<seed> subtests) plus SUPKV_SIM_FRESH_SEEDS fresh batches (default 1); SUPKV_SIM_SEED replay path first and the chaosReplayFmt contract verbatim"
  - "quiescence-necessary-not-sufficient stabilization: after the three-seam condition the drive keeps polling until every store equals the pointwise-supremum model, within the unchanged 10×sync_interval hard cap"
  - "deletions in the chaos mix via Cluster.Delete against live keys, with a deletion-count fault tooth and deterministic drain scenarios whose final state is empty/tombstone-only"
  - "per-seed quiescence/convergence/delta logs — the distribution evidence plan 02-11's 10k proof interprets"
affects: [02-11 (10k stability proof), 02-13 (mutation sensitivity + TEST-09 bounds)]

# Actuals (#2632) — same estimateTokens scale as the plan estimate: chars/4 over the realized diff.
actuals:
  tokens: 5455
  tasks: 2
  commits: 2

plan_head_before: 242513f4619c5ae2076628f016f85b4732df22a1
plan_head_after: b35060d75cde03790a0009063a5e184fd933df42

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Drift-checked seed corpus: the generator test always regenerates in memory (twice, proving determinism) and byte-compares against the committed file; writes gated on missing/SUPKV_SIM_CORPUS_REGEN=1"
    - "Quiescence necessary-not-sufficient: post-quiescence poll to pointwise-supremum equality via recordMapsEqual, bounded by the unchanged hard cap"
    - "Deletion-heavy drain sweep: deterministic node-order walk over sorted live keys guarantees empty/tombstone-only final states without rng or time advance"

key-files:
  created:
    - internal/app/repl/testdata/chaos-seeds.txt
  modified:
    - internal/app/repl/sim_chaos_test.go

key-decisions:
  - "Corpus format is bare decimal seed lines with no header comment, so `head -1` of the committed file is always a valid SUPKV_SIM_SEED replay argument for the plan's verify command and CI scripts"
  - "Deletion-heavy scenarios end with a deterministic closing drain sweep (each node's smallest live key, no rng draws, no time advance) instead of tuning the delete probability until a seed happens to end tombstone-only — the sweep guarantees the TEST-01 empty-state probe without weakening any assertion"
  - "Random deletes target the acting node's sorted live keys with a pre-drawn pick index: 'against live keys' holds at execution time while every random input stays fixed before any event runs"
  - "chaosSeed keeps its env-or-fresh semantics because the scenario suite shares the helper; the chaos test checks SUPKV_SIM_SEED first, so the single-seed replay path stays first and unchanged"
  - "The drain flag is drawn before the operation loop, so its draw changes the scenario sequence once at implementation time but regeneration of the corpus (seeds only) remains independent of scenario semantics"
  - "Deletions count as a fifth fault tooth alongside dropped/duplicated/reordered/partitioned: a run that never deleted fails, mirroring the existing 'a run that dropped nothing fails' bar"

patterns-established:
  - "Generator-owned testdata format: the generator test is the format spec, drift fails the build, and the write gate keeps green runs read-only"
  - "Test-side stabilization without product changes: assert the same full model and the same cap, poll longer (the R-05 delayed heal becomes a delta measurement, not a failure)"
  - "Deterministic deletion mix: live-key targeting by sorted order + modulo pick, and drain sweeps as the deletion-heavy scenario shape"

requirements-completed: [TEST-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Committed fixed seed corpus with a always-regenerate-and-compare generator test that fails on byte drift and never rewrites the file on green runs"
    requirement: "TEST-01"
    verification:
      - kind: unit
        ref: "internal/app/repl/sim_chaos_test.go#TestChaos_seedCorpusGenerator"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestChaos_convergence iterates every corpus seed as a named subtest plus SUPKV_SIM_FRESH_SEEDS fresh runs (default 1), with the SUPKV_SIM_SEED single-seed replay path first"
    requirement: "TEST-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run '^TestChaos'"
        status: pass
    human_judgment: false
  - id: D3
    description: "Post-quiescence poll-to-pointwise-supremum stabilization within the unchanged 10×sync_interval hard cap, with per-seed quiescence/convergence/delta logs"
    requirement: "TEST-01"
    verification:
      - kind: unit
        ref: "internal/app/repl/sim_chaos_test.go#TestChaos_convergence (chaosAssertConvergence + chaosConverged)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Deletions in the chaos mix via Cluster.Delete against live keys with a deletion-count fault tooth; the model covers tombstones through the write log"
    requirement: "TEST-01"
    verification:
      - kind: unit
        ref: "internal/app/repl/sim_chaos_test.go#TestChaos_convergence"
        status: pass
    human_judgment: false
  - id: D5
    description: "Corpus keeps an executable empty-state probe: at least one fixed seed converges to an empty or tombstone-only model"
    requirement: "TEST-01"
    verification:
      - kind: unit
        ref: "internal/app/repl/sim_chaos_test.go#TestChaos_corpusEmptyStateProbe"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 6: Chaos Stabilization — Fixed Corpus, Deletions, Poll-to-Model Summary

**Chaos is stable test-side: a committed 16-seed PCG corpus with a byte-exact drift-checked generator, deletions in the mix via `Cluster.Delete`, and quiescence treated as necessary-not-sufficient with a post-quiescence poll to full pointwise-supremum equality under the unchanged 10×sync_interval cap.**

## Performance

- **Duration:** 10 min (574 s)
- **Started:** 2026-11-21T10:47:01Z
- **Completed:** 2026-11-21T10:56:35Z
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- **Fixed corpus + generator drift gate (D-06, T-2-12):** `testdata/chaos-seeds.txt` carries 16 seeds drawn from one PCG stream seeded with `0x5eedc0de5eed` through `simRNG`; `TestChaos_seedCorpusGenerator` regenerates the bytes in memory twice and byte-compares against the committed file, failing on drift. The write is gated on a missing file or `SUPKV_SIM_CORPUS_REGEN=1`, so a green run never touches the committed corpus.
- **Corpus + fresh-seed iteration:** with `SUPKV_SIM_SEED` unset, `TestChaos_convergence` runs every corpus seed as a `corpus/<seed>` subtest plus `SUPKV_SIM_FRESH_SEEDS` (default 1) fresh wall-clock runs. The `SUPKV_SIM_SEED` single-seed replay stays first and `chaosReplayFmt` is byte-identical to baseline.
- **Stabilization without weakening:** quiescence (no pending deliveries, no unacked, no queued responses) is now necessary-not-sufficient. The drive keeps stepping in `chaosQuiesceStep` increments until every store equals the pointwise-supremum model via the existing `recordMapsEqual`, bounded by the UNCHANGED hard cap `10×sync_interval` past the last operation. The model equality, the cap, and the fault teeth are unmodified.
- **Deletions in the mix:** operation draws gain `Cluster.Delete` against live keys (pre-drawn pick modulo sorted live keys), one-in-four scenarios add a deterministic closing drain sweep, and a fifth fault tooth (`deleted`) fails a run that never deleted. Seven of the 16 corpus seeds end empty/tombstone-only, proving the TEST-01 empty-state probe remains executable.
- **Evidence:** `go test -race -count=1 ./internal/app/repl/ -run '^TestChaos'` green on the corpus + 50 fresh seeds; the review's failing seed `1795020540436451818` now converges; per-seed delta logs expose the quiescence→convergence gap for the 10k proof (one pre-drain run measured a 44.6 s delta, confirming the old first-quiet-moment failure mode).

## Task Commits

Each task was committed atomically:

1. **Task 1: seed corpus + drift-checked generator test** - `110d5f6` (test)
2. **Task 2: corpus iteration + stabilization semantics + deletions in the mix** - `b35060d` (test)

**Plan metadata:** pending docs commit (SUMMARY/STATE/ROADMAP/REQUIREMENTS)

## Files Created/Modified

- `internal/app/repl/testdata/chaos-seeds.txt` — committed 16-seed corpus, one decimal int64 per line, no header (so `head -1` is a valid replay seed)
- `internal/app/repl/sim_chaos_test.go` — corpus generator test + reader, fresh-seed parameterization, deletion op draw with live-key targeting, drain sweep, corpus iteration, poll-to-model convergence loop, delta logging, deletion fault tooth, empty-state probe

## Decisions Made

- **Bare seed lines, no header:** the plan's replay verify uses `head -1 testdata/chaos-seeds.txt`; a comment header would break it. Comment-line tolerance exists in the reader for forward compatibility, but the committed format is bare seeds.
- **Drain sweep instead of probability tuning:** with 30% random deletes, no corpus seed ended tombstone-only (roughly 22 distinct keys, each needing its supremum to be a tombstone). A deterministic closing sweep over each node's sorted live keys guarantees the empty/tombstone-only scenario without weakening the model, the cap, or the fault teeth.
- **Live-key targeting without execution-time randomness:** the delete target is `sorted(liveKeys)[pick % len]` with `pick` pre-drawn at scenario-draw time. Selection is deterministic and consumes no rng during execution, preserving the "every random input fixed before any event runs" property.
- **Drain flag drawn before operations:** this intentionally changes scenario sequences once (new semantics); the corpus itself is seeds only, so its regeneration stays independent of scenario semantics.
- **Shared `chaosSeed` helper preserved:** the scenario suite also calls it; the chaos test checks `SUPKV_SIM_SEED` first, so the replay branch is first and unchanged while scenario tests keep their fresh-seed fallback.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Deterministic drain sweep makes deletion-heavy corpus seeds real**
- **Found during:** Task 2 (stabilization semantics + deletions)
- **Issue:** The plan's must-have requires the corpus to include deletion-heavy seeds whose final stores are empty or tombstone-only. With the specified "op draw gains `Cluster.Delete` against live keys" and a plausible 30% delete probability, all 16 corpus seeds still ended with live records — the TEST-01 empty-state probe had no executable instance.
- **Fix:** Added a `drain` scenario branch (1 in 4, drawn before the op loop): after the scripted operations, a deterministic sweep deletes every key the cluster still holds live, walking nodes in order and taking each node's smallest live key. Because `Store.Delete` installs unconditionally and the sweep covers every node, every authored live record is overwritten by a later tombstone from its author; the final pointwise-supremum model is empty/tombstone-only. No rng draws, no time advance, no change to the convergence assertion, cap, or fault teeth.
- **Files modified:** `internal/app/repl/sim_chaos_test.go`
- **Verification:** `TestChaos_corpusEmptyStateProbe` passes and logs a tombstone-only seed; a diagnostic scan showed 7 of 16 corpus seeds end tombstone-only; all 16 corpus seeds plus 50 fresh seeds pass the full convergence test under `-race`.
- **Committed in:** `b35060d` (Task 2 commit)

**2. [Rule 2 - Missing Critical] Executable empty-state probe test**
- **Found during:** Task 2 (stabilization semantics + deletions)
- **Issue:** The must-have "deletions participate … and the model covers tombstones" is only indirectly observable from `TestChaos_convergence`; without a dedicated assertion, a future edit could silently drop the corpus's empty/tombstone-only seeds.
- **Fix:** Added `TestChaos_corpusEmptyStateProbe`, which runs the corpus, skips seeds whose convergence failed (the convergence test owns that failure), and asserts at least one fixed seed's final model is empty or tombstone-only.
- **Files modified:** `internal/app/repl/sim_chaos_test.go`
- **Verification:** passes locally (one logged seed with 18 records, all tombstones).
- **Committed in:** `b35060d` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 missing critical)
**Impact on plan:** Both are test-side additions inside the plan's two listed files; no production code, no new files beyond the planned corpus, no weakening of any assertion. The drain sweep is the mechanism that makes the plan's own must-have ("deletion-heavy seeds whose final stores are empty or tombstone-only") executable rather than aspirational.

## Issues Encountered

- **First deletion mix produced no empty/tombstone-only corpus seed.** The 30% random delete mix left every seed with live records; the pure-probability reading of the must-have is effectively unreachable at 40 ops over a 32-key space. Resolved with the deterministic drain branch (deviation 1).
- **Interim `chaosSeed` refactor broke the scenario suite.** Making the helper env-only caused `TestScenario_partitionDays` to fail on `ParseInt("")`. Restored the env-or-fresh helper and kept the chaos env branch first (decision 5). No committed state ever contained the broken helper: the fix landed before the Task 2 commit.
- **Lint caught inline error handling and a nested branch in the generator test.** Rewrote the load/regenerate path as `chaosCorpusLoad` with plain assignments and a `SplitSeq` reader; `golangci-lint run ./internal/app/repl/...` is clean.

## User Setup Required

None - no external service configuration required. `SUPKV_SIM_FRESH_SEEDS` and `SUPKV_SIM_CORPUS_REGEN` are test-only env knobs, no product configuration.

## Known Stubs

None. The corpus is real data, the generator is deterministic, and every assertion runs against production `storage.Store`/`repl.Core` code paths through `simnet`.

## Threat Flags

None — the only trust-boundary-relevant change (corpus file → test semantics, T-2-12) is mitigated by the drift-checked generator: a tampered corpus line fails `TestChaos_seedCorpusGenerator`, and fresh seeds still run every execution so a trimmed corpus cannot hide regressions.

## Next Phase Readiness

- **Plan 02-11 (10k proof):** the corpus + `SUPKV_SIM_FRESH_SEEDS` batch interface is in place; `-v`/`-json` runs expose the per-seed quiescence/convergence/delta lines for the distribution measurement.
- **Plan 02-13 (mutation sensitivity + TEST-09):** the stabilized semantics still assert the full pointwise-supremum model and the unchanged hard cap, so the overlay mutants have real teeth; the corpus is the fixed-seed substrate they run against.
- **TEST-01 remains open in REQUIREMENTS.md:** this plan and plan 02-13 both own TEST-01; the requirement's 10k-scale "0 fails" statement is plan 02-11's proof. This plan's evidence is 16 corpus seeds + 50 fresh seeds under `-race`, all green, with no residual R-05 red evidence observed.
- **No blockers carried:** the review's failing seed `1795020540436451818` converges under the new semantics (its scenario is intentionally different — the op draw changed — so it is not a byte-identical reproduction of the old run).

## Self-Check

- [x] `internal/app/repl/testdata/chaos-seeds.txt` exists (16 seeds)
- [x] `internal/app/repl/sim_chaos_test.go` exists and compiles
- [x] Task commits `110d5f6` and `b35060d` are ancestors of HEAD
- [x] `go test -race -count=1 ./internal/app/repl/ -run '^TestChaos'` passes
- [x] `SUPKV_SIM_SEED=$(head -1 internal/app/repl/testdata/chaos-seeds.txt) go test -count=1 ./internal/app/repl/ -run '^TestChaos_convergence$' -v | grep -c 'replay: SUPKV_SIM_SEED='` returns 1
- [x] `golangci-lint run ./internal/app/repl/...` clean
- [x] `check evaluation-scope --plan 02-06` returns both commits and both changed files, no missing/outside files

**Self-Check: PASSED**

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
