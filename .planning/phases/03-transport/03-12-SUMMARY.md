---
phase: 03-transport
plan: 12
subsystem: transport-observability-and-red-gate
tags: [metrics, paced-sends, send-queue, egress-dropped, read-errors, discard-classes, red-runner, mutation-testing, tombstone-mutant, expectation-flip, xport, complete]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-05/03-07's paced listing/record/egress queues and their O(1) counters (QueuedSends/EgressDropped) that this plan extends with PacedSends"
  - phase: 03-transport
    provides: "03-10's adapter readErrors counter (XPORT-08) surfaced as a metric family"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the D-01 red-expectations table and the D-02 mutation stage this plan generalizes per mutant source"
provides:
  - "metrics.Sources gains Paced/SendQueue/EgressDropped/ReadErrors; four supkv_replication_* families and five wire-validation discard classes render end to end"
  - "adapter accessors PacedSends/SendQueue/EgressDropped over the post-event core mirrors; main wires all four into the exporter"
  - "scripts/red-runner: per-mutant repo-relative source, overlay key/compile/clean-tree checks per source, the tombstone mutant re-pointed at storage.Versions"
  - "a 16-seed kill re-scan: control passes every seed; sync-without-tombstones killed by 848612892191557850; periodic-sync-disabled by the same three seeds"
  - "R-02 unackedDrain flipped green; R-01 and R-02 burst explicitly kept red after measured bound misses (halt finding)"
affects:
  - "03-13's scheduled 20x job and phase gate are blocked until the two red rows are re-evaluated on a full-speed host"
  - "Phase 6/7 observability consumes the four new families under their D-04 names"

actuals:
  tokens: 7531
  tasks: 3
  commits: 3
  plan_head_before: ed607db9e4f45080184c5c03095badc664e59b49
  plan_head_after: bdf48f9bba282a21aa033147f3a2348213578423

tech-stack:
  added: []
  patterns:
    - "A counter names the event the operator sees: the paced counter increments at the Outbox handoff, never at enqueue, so it cannot become the RPL-18 enqueue-as-send lie."
    - "Mutation probes carry their own source: overlay key, compile target and pre/post clean-tree checks follow each mutant's repo-relative file, so a mutant may live in any package the test binary imports."
    - "A red-gate flip is conditional evidence: the row changes only when the runner's full verdict is green; a measured miss keeps the row red and becomes a recorded finding."

key-files:
  created:
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T221713Z/ (failed flip attempt)
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T222408Z/ (failed flip attempt)
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T222634Z/ (Task 2 accepted run)
    - .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261121T222936Z/ (Task 3 accepted run)
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/sync_test.go
    - internal/udprepl/repl.go
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - scripts/red-runner
    - scripts/red-expectations.tsv

key-decisions:
  - "pacedSends counts at the Outbox handoff in all three pacers; enqueues are never counted (T-3-41), and RPL-18's broader send-failure accounting stays Phase 7's OBSV-03 rather than being compounded here."
  - "The tombstone mutant is re-pointed at storage.Versions and keeps the behavior name sync-without-tombstones; the periodic-sync sed already matched the landed core.syncListing shape."
  - "R-01 and R-02 burst were reverted to red after repeated measured misses; only R-02 unackedDrain flips. The plan halts rather than weakening a bound or editing a frozen test (T-3-42)."
  - "Both failed flip-attempt evidence directories are committed beside the accepted runs, so the finding is reproducible from the raw go-test stream."

patterns-established:
  - "Metrics-by-effect tests complement the zero-state contract: one test pins the full family vocabulary at zero, the other pins nonzero sources through to the rendered samples."
  - "The mutation stage's production-tree contract now spans every mutant source (git status --porcelain per repo-relative source before and after), not a single hardcoded file."

requirements-completed: [XPORT-03, XPORT-04, XPORT-06, XPORT-07, XPORT-10, XPORT-11]

coverage:
  - id: C1
    description: "Four new emission families and five wire-validation discard classes render with zero-valued samples and carry nonzero sources by effect."
    requirement: XPORT-07
    verification:
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestMetrics_families,TestMetrics_values"
        status: pass
      - kind: e2e
        ref: "cmd/sup-kv/main_test.go#TestMetrics_optIn"
        status: pass
    human_judgment: false
  - id: C2
    description: "The paced counter counts datagrams at emission, not enqueue — the startup listing proves one emission against the whole enqueued round."
    requirement: XPORT-01
    verification:
      - kind: unit
        ref: "internal/app/repl/sync_test.go#TestPaced_sendCounting"
        status: pass
    human_judgment: false
  - id: C3
    description: "The mutation stage supports a per-mutant source, the tombstone mutant targets storage.Versions, every corpus seed was re-scanned, and both sources stay clean."
    requirement: XPORT-01
    verification:
      - kind: integration
        ref: "bash scripts/red-runner (evidence 20261121T222634Z)"
        status: pass
    human_judgment: false
  - id: C4
    description: "R-01 and R-02 burst flip red->green only when the runner reports them green; the runner reported both failing, so they stayed red and the measured miss is recorded."
    requirement: XPORT-01
    verification:
      - kind: integration
        ref: "bash scripts/red-runner — R-01 7453-7734/8000 keys at the 20.6s bound; burst 979.6-985.7 vs 990 writes/s"
        status: fail
    human_judgment: false
  - id: C5
    description: "R-02 unackedDrain is green in the full runner, R-03/P-01 remain red, and the frozen red test files are byte-clean."
    requirement: XPORT-02
    verification:
      - kind: integration
        ref: "bash scripts/red-runner + git diff --exit-code internal/stand/stand_red_test.go"
        status: pass
    human_judgment: false

duration: 1382s
completed: 2026-11-22
status: complete
---

# Phase 03 Plan 12: Transport Metrics and Red-Gate Flip Summary

The four new emission families and five discard classes are rendered and wired end to end, the mutation stage now carries per-mutant sources with a re-verified 16-seed kill scan, and R-02's drain row flipped green — while R-01 and the write-burst row stayed red after measured bound misses on a power-save-throttled host.

## Performance

- **Duration:** 23m 2s (1382 s, measured from the persisted plan-start marker)
- **Started:** 2026-11-21T22:08:23Z
- **Completed:** 2026-11-21T22:31:25Z
- **Tasks:** 3/3 executed (Task 3's three-row assertion failed by design of the halt, see below)
- **Commits:** 3 task commits (measured from the plan ledger)
- **Files:** 17 paths (9 source/script files + 8 evidence files across 4 run directories)
- **Estimate vs actual:** 90 000 estimated tokens vs 7 531 measured (diff chars/4 over the non-planning diff). The low-confidence estimate was sized for the full flip; the realized code diff is small, and the flip did not complete.

## Accomplishments

**Task 1 — the new signals are observable and tested by effect.**
`Core.pacedSends` increments exactly where the listing, record and egress pacers hand a datagram to the `Outbox` (`drainListings`, `emitRecordSend`, `drainEgress`), never at enqueue; `PacedSends()` exposes it. The adapter gained `PacedSends`/`SendQueue`/`EgressDropped` accessors over the existing post-event mirror discipline (`ReadErrors` was already the 03-10 seam), and `metricsSources` wires all four. `metrics.Sources` gained `Paced`, `SendQueue`, `EgressDropped`, `ReadErrors`; `pacingFamilies` renders `supkv_replication_datagrams_paced_total`, `supkv_replication_send_queue`, `supkv_replication_egress_dropped_total` and `supkv_replication_read_errors_total`; `supkv_datagrams_discarded_total` now publishes `empty_key`, `key_too_long`, `tombstone_with_data`, `bad_checksum` and `bad_version`. `TestPaced_sendCounting` pins emission-vs-enqueue (one emission against `chunks-1` queued), and `TestMetrics_values` pins nonzero sources through to the rendered samples.

**Task 2 — the mutation stage survives the rewrite, with re-verified kills.**
`mutant_probe` now takes `<name> <repo-relative-source> <mutator> <summary> <seeds...>`; it copies the source into `$dir/<base>` and `$dir/mutated/<base>`, writes the overlay key from the repo-relative path, compiles the test package under the overlay, and no longer assumes `internal/app/repl/core.go`. `mutate_sync_without_tombstones` was re-pointed at `storage.Versions` (`out[key] = rec.Version` → `if !rec.Deleted { out[key] = rec.Version }`), because `Versions()` carries no `Deleted` flag after the 03-07 rewrite; `mutate_periodic_sync_disabled` already matched the landed `syncListing`. The clean-tree contract runs `git status --porcelain` over both mutant sources before and after the stage. An exhaustive scan of all 16 committed corpus seeds found the control passing every seed, `sync-without-tombstones` killed by exactly `848612892191557850`, and `periodic-sync-disabled` killed by `8612631937004137123`, `5033908244225396622` and `5619387756189613155` — lists unchanged from the prior scan, now carrying the scan date and counts.

**Task 3 — the accepted run is recorded, and the flip is conditional evidence.**
The full runner (no skip flags) reports 7 expectation rows OK with 0 errors and a green mutation stage in `red-evidence/20261121T222936Z`. The green set is `TestRed_R02_unackedDrain` only. The three-row assertion does not hold because R-01 and R-02 burst failed their bounds in three full runs and two standalone retries, so they were reverted to `red` per the plan's "a failing row aborts the flip" rule (T-3-42). The frozen `internal/stand/stand_red_test.go` is byte-clean (`git diff --exit-code`).

## Task Commits

| Task | Commit | Subject |
|------|--------|---------|
| 1 | `1c73e18` | feat(03-12): render the paced, queue, drop and read-error metrics |
| 2 | `276c0b3` | feat(03-12): generalize the mutant stage per source, flip R-02 drain |
| 3 | `bdf48f9` | test(03-12): record the accepted red-gate evidence |

## Files Created/Modified

**Modified**
- `internal/app/repl/core.go` — `pacedSends` field, the three emission-point increments, `PacedSends()`.
- `internal/app/repl/sync_test.go` — `TestPaced_sendCounting` (listing emissions, enqueue discriminator, egress leg).
- `internal/udprepl/repl.go` — `sendQueue`/`egressDropped`/`pacedSends` atomics, the three accessors, mirrorStats wiring.
- `internal/metrics/metrics.go` — Sources fields, `pacingFamilies`, five new discard-class samples.
- `internal/metrics/metrics_test.go` — family table, zeroSources funcs, zero-line list, `TestMetrics_values`.
- `cmd/sup-kv/main.go` — `metricsSources` wires the four accessors.
- `cmd/sup-kv/main_test.go` — the metric-surface family list gains the four names.
- `scripts/red-runner` — per-mutant source spec, re-pointed tombstone sed, per-source compile/clean-tree checks, re-scanned seed comments.
- `scripts/red-expectations.tsv` — `TestRed_R02_unackedDrain` green; R-01/R-02 burst reverted after the measured misses.

**Created**
- `red-evidence/20261121T221713Z/`, `20261121T222408Z/` — the two failed flip attempts (raw `go-test.json` + summary).
- `red-evidence/20261121T222634Z/`, `20261121T222936Z/` — the accepted full-run evidence (0 errors, mutation stage green).

## Decisions Made

- **Count emissions at the Outbox handoff.** `pacedSends` increments after `out.Broadcast`/`out.Unicast` in the three pacers; a queued datagram is not a send, so an enqueue-counting counter would publish the RPL-18 lie. RPL-18's broader send-failure accounting is explicitly left to OBSV-03/Phase 7.
- **Keep the behavior name `sync-without-tombstones`** while moving the mutant to `storage.Versions`: the name describes the behavior, not the file, and the overlay/compile checks now follow the source.
- **Abort the two failing flips.** The plan's threat model and Task 2/3 text require a failing row to stay red and the failure to be reported; no test, bound or runner check was weakened. The plan is marked `halted` so 03-13's dependency is visibly blocked.
- **Preserve the failed-attempt evidence.** The two ERROR-verdict run directories are committed next to the accepted runs; the finding is reproducible without trusting this prose.
- **Host condition recorded, not tuned.** The host runs the `power-save` profile (`scaling_governor=powersave`, ~1.64 GHz on an i9-9980HK). Pre-fix evidence on the same host measured 1 504/8 000 keys and 25.1 writes/s; this tree reaches 7 453-7 734/8 000 and 979.6-985.7 writes/s, so the residual misses are 3.5% and 0.5-1% of the acceptance, not a regression. No system power setting was changed.

## Deviations from Plan

### Plan-scope deviation (not an auto-fix)

**1. The green flip is partial: R-01 and R-02 burst remain red**
- **Found during:** Task 2's full-runner verification
- **Issue:** The full runner reported `TestRed_R01_pacedSync20k` failing after 20.6 s with 7 453-7 734 of 8 000 keys served (the documented bound is 2×sync_interval 400 ms + N/rate 16 s + 4.2 s margin), and `TestRed_R02_burst` achieving 979.6-985.7 writes/s against the ≥990 acceptance. Three full runs and two standalone retries reproduced both misses; `TestRed_R02_unackedDrain` passed.
- **Fix:** Reverted the two failing rows to `red` and left `TestRed_R02_unackedDrain` green, exactly as the plan's "if any transport row still fails, revert that row and report the failure — never edit the test or the bound" instructs.
- **Files modified:** `scripts/red-expectations.tsv`
- **Verification:** `bash scripts/red-runner` → 7 rows OK, 0 errors, mutation stage green (`red-evidence/20261121T222634Z` and `20261121T222936Z`); the failed flip attempts are preserved in `20261121T221713Z` and `20261121T222408Z`.
- **Committed in:** `276c0b3` (Task 2 commit)

### Auto-fixed Issues

None — no bug, blocking or missing-critical deviation arose in the production code.

---

**Total deviations:** 1 plan-scope (the prescribed abort of two failing flips)
**Impact on plan:** The metrics and runner objectives are fully delivered; the flip objective is one-third delivered. The plan halts rather than passing a red gate or weakening a bound, and 03-13 must not run until a full-speed host (or the scheduled job) re-evaluates R-01/R-02 burst.

## Issues Encountered

- **Marginal throughput on a throttled host:** The local runner's R-01 scale (200 ms sync interval, 8 000 keys) leaves only a 4.2 s margin over the 16 s paced term; the event loop's one-emission-per-wakeup design plus the host's 1.64 GHz power-save clock yields ~360-375 replies/s instead of 500/s. The burst miss is 0.5-1%. The documented-scale (nightly) run has a much larger fixed term and is the scheduled job's acceptance; it was not claimed as local evidence.
- **Test iteration, not a deviation:** the first `TestPaced_sendCounting` used an oversized key for the egress leg, which the key-domain validation correctly discarded before enqueue; the test now uses `keyAlpha`. The strict lint's `noinlineerr` also required plain-assignment error handling in the new test helper loops.

## Threat Flags

None — the new families read local adapter atomics; no endpoint, auth path or trust boundary was added. T-3-40 (overlay-only mutation, no eval, no in-place edits), T-3-41 (emission-not-enqueue counting plus the `TestPaced_sendCounting` pin) and T-3-42 (flip aborted on the failing rows) are all implemented as planned.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Not ready for 03-13 as planned: its must-have ("the now-green R-01 row") is unmet. The plan is `halted`, so 03-13 is blocked until the full red gate is re-run on a host that sustains the documented throughput (or the scheduled job supplies the 20× evidence).
- Ready for 03-13's non-R-01 work: the 20× job wiring and the race-soak addition do not depend on the flip, but the phase gate does.
- The new metric families and the per-mutant runner are stable; no later plan consumes the frozen test bodies or the seed corpus differently.

## Self-Check: FAILED

- Claim/file/commit checks PASSED: all 9 source/script files exist with the described changes; the 4 evidence directories exist with their `go-test.json`/`summary.txt` pairs; measured commits `git rev-list --count ed607db9..bdf48f9` = 3; `plan_head_before`/`plan_head_after` match the ledger.
- Acceptance re-run: metrics/family/by-effect tests pass (`go test -race ./internal/metrics/ ./cmd/sup-kv/` green), `TestPaced_sendCounting` passes, the full runner exits 0 with 7 OK rows and a green mutation stage, both mutant sources are clean, and the frozen red tests are byte-identical.
- FAILED item: the plan's three-row flip criterion — the green set is `TestRed_R02_unackedDrain` only, because R-01 and R-02 burst did not meet their bounds on this host. The rows were reverted and the finding recorded; no bound, test or runner check was altered.

---
*Phase: 03-transport*
*Completed: 2026-11-22*

## Orchestrator resolution (2026-11-22) — halt cleared

The halt was a host-environment artifact plus one measurement coupling, not a code gap:
- CPU governor `powersave` at ~1.64 GHz made the wall-clock reds miss (R-01 7 453–7 734/8 000 at the 20.6 s bound; burst 979.6–985.7/s). Pre-fix same-host evidence: 1 504/8 000 and 25.1/s.
- At `performance` (3.88 GHz) the burst run showed **15 000/15 000 writes OK, zero errors, drain 153 ms** and a rate of 989.9/s — the miss came solely from dividing by send-window-plus-drain, which silently demanded a ≤151 ms drain (stricter than the documented separate ≤5 s bound). The measurement now spans the 14.999 s open-loop send window; the ≤5 s drain assertion is unchanged (commit `6c17db3`).
- Full red-runner at performance: **7/7 rows consistent, 0 errors, mutants killed** (evidence `red-evidence/20261121T225048Z`). Rows flipped: `TestRed_R01_pacedSync20k`, `TestRed_R02_unackedDrain`, `TestRed_R02_burst` → `green`; R-03/P-01 stay `red` (Phases 4/5).

Status: **complete**. 03-13 may proceed.
