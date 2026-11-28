---
phase: 08-performance
plan: 10
subsystem: performance-documentation
tags: [performance, measurement-ledger, doc-anchor, drift-guard, perf-01, perf-02, perf-03, perf-04, perf-05, perf-06, amplification, memory-model, absent-peer, spec-deviations]
requires:
  - phase: 08-performance
    provides: the threshold table, hardware block and perf/race gates (08-01), the deterministic store-call and pending-visit gates plus the paired absent/live benchmark (08-05), the default-profile latency, amplification and memory perf rows (08-07), the two-profile confirmation benchmark and fast-profile gate (08-08), and the last cmd/sup-kv harness change (08-09)
provides:
  - docs/PERFORMANCE.md as the one measurement ledger, with a hardware block, commit and producing command in every measured section
  - cmd/sup-kv/performance_test.go TestPerformanceDoc/TestPerformanceDocFamilies/TestSpecDeviationsBurstRow as the presence and drift guards, with threshold literals formatted from internal/testutil
  - the closed S-FAIL-15 registry row citing TestRed_R02_burst, TestPerfBurstDistinctKeys and TestEngine_burstBoundedConfirmation with the 990 writes/s and 5 s bounds
  - docs/OPERATIONS.md reduced to a memory summary plus a ledger link, and .planning/PROJECT.md's latency claim carrying its fast-profile conditions
affects: [09-documentation-and-release, PERF-01, PERF-02, PERF-03, PERF-04, PERF-05, PERF-06, DOCS-05]
actuals:
  tokens: 10885
  tasks: 3
  commits: 3
plan_head_before: 9cdc0f1aedb18c86d6f6b44abaaab0bc5080ded0
plan_head_after: d89b1b48d98e36924f2adaf053c84fd2251b6604
tech-stack:
  added: []
  patterns:
    - "Derive doc threshold literals from the Go constants: durations via time.Duration.String(), the burst rate via %.0f, ratios via a fixed '≤ %.1f×' fragment, so a threshold change without a doc change fails make test"
    - "Assert one hardware block and one producing command per measured section, not a document-wide count, so a section cannot borrow a neighbour's block"
    - "Publish the second of two make perf runs and record its load; a number alone is never a measurement"
    - "Record the measured commit beside each hardware block because Go test binaries report vcs revision as unknown"
key-files:
  created:
    - docs/PERFORMANCE.md
    - cmd/sup-kv/performance_test.go
  modified:
    - docs/OPERATIONS.md
    - docs/SPEC-DEVIATIONS.md
    - .planning/PROJECT.md
key-decisions:
  - "Every published row is the second of two consecutive green make perf runs at commit 9cdc0f1; the document states that provenance and each block's load instead of gating on a calm host."
  - "Go test binaries report module=VCS unknown, so the measured commit is recorded beside each block rather than inside it; the block otherwise carries every mandated field."
  - "The fast-profile measured row shares the model section's single hardware block (Test 8 allows exactly one per section); the fast arm's own logged load is stated inline, since only that field differs."
  - "The amplification row is explicitly a workstation NVMe/ext4 result and the target-grade eMMC/SD gap is named against the Spec open question #2; no target number is claimed."
  - "The memory caveat was tightened after measurement: the 1 KiB rows' large RSS/logical ratio is the key-dominated baseline, while the 60 KiB row's dump is 1.34x value bytes, inside the model's 1.37x plus framing."
  - "The remaining '~10 ms' strings live only in ADR 0003's obligation sentence (now satisfied) and the historical review evidence; no standalone latency claim survives in the ledger or PROJECT.md."
patterns-established:
  - "Schema-plus-caveat doc guards: the anchor test pins presence, ordering, thresholds and per-section reproducibility; the perf run pins truth."
  - "One ledger plus links: OPERATIONS keeps the qualitative shape and the residual pause; PERFORMANCE.md owns the formula, the points and the bounds."
requirements-completed: [PERF-01, PERF-02, PERF-03, PERF-04, PERF-05, PERF-06]
coverage:
  - id: D1
    description: "docs/PERFORMANCE.md exists as the one ledger, opens with the write-latency model, and carries a hardware block, measured commit and producing command in each of the five measured sections."
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestPerformanceDoc$|^TestPerformanceDocFamilies$' (PASS 1.64 s)"
        status: pass
      - kind: integration
        ref: "make perf twice (both exit 0, second run published): 15.67 s burst, 36.43 s default profile, 37.43 s amplification, three memory points, 4.88 s repl, 2.47 s grpcapi, then the threshold-free stand suite"
        status: pass
    human_judgment: false
  - id: D2
    description: "The 1000 writes/s distinct-key burst is documented with its guarantee and non-guarantee lists, and the S-FAIL-15 registry row is closed against the shipped tests with the 990 writes/s and 5 s bounds."
    requirement: PERF-02
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run '^TestSpecDeviationsBurstRow$' (PASS 0.00 s)"
        status: pass
      - kind: integration
        ref: "make perf TestPerfBurstDistinctKeys: achieved=1000.1 writes/s (min 990), drain=305.13 ms (bound 5s), 10 000 distinct keys, 1 KiB values, fs ext4"
        status: pass
    human_judgment: false
  - id: D3
    description: "Disk-write amplification is measured with the five-step method, the measured NVMe row, the per-dump bound and an explicit target-grade eMMC/SD gap against the Spec open question #2."
    requirement: PERF-03
    verification:
      - kind: unit
        ref: "TestPerformanceDoc anchors bytes-per-dump, Δ dumps × dump size, write_bytes, Spec open question #2, eMMC and target-grade; grep gate: disk cost / 1 dump/s / 20 dumps/s / rewrites the whole state / in-memory dump / eMMC"
        status: pass
      - kind: integration
        ref: "make perf TestPerfAmplification: Δ dumps=15, dump=14 977 443 B, bytes-per-dump=14 979 072 B (bound ≤1.1× = 16 475 187 B), logical=15 645 000 B, amplification=14.36×, failures=0"
        status: pass
    human_judgment: false
  - id: D4
    description: "The memory model publishes the formula and three measured state-to-RSS points, ties the one-copy streamed path to PERSIST-06, and OPERATIONS keeps only a summary plus a link with no duplicated ratio."
    requirement: PERF-04
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./cmd/sup-kv/ -run 'TestOperationsDoc|TestPerformanceDoc|TestSpecDeviationsBurstRow|TestAPIDoc|TestExamplesDoc|TestReadme' (PASS 7.68 s); ! grep 'roughly 1.4 times' docs/OPERATIONS.md"
        status: pass
      - kind: integration
        ref: "make perf TestPerfMemoryPoints: 1k×1KiB RSS 28 639 232 B (27.59×), 10k×1KiB 67 739 648 B (6.53×), 1k×60KiB 229 527 552 B (3.73×, peak 0.54×)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The absent-peer claim is documented with the per-key index, the deterministic gates and the paired ratio, and PROJECT.md's latency claim carries the fast-profile conditions and the ledger link with no standalone ~10 ms wording."
    requirement: PERF-05
    verification:
      - kind: unit
        ref: "TestPerformanceDoc anchors O(1), per-key index and ≤ 2.0×; ! grep 'confirmation latency ~10ms under 1000 w/s load' .planning/PROJECT.md; grep 'in-memory dump' and 'docs/PERFORMANCE.md' in PROJECT.md"
        status: pass
      - kind: integration
        ref: "make perf TestPerfAbsentPeerWriteCost: absent 2 249 ns/op, live 8 598 ns/op, ratio 0.26 (bound ≤2.0×), unacked=10 000, both arms 0.00 pending-visits/op"
        status: pass
      - kind: integration
        ref: "make perf TestPerfFastProfileLatency: mean 2.82 ms (bound 25ms), p99 5.55 ms (bound 100ms), p50 2.60 ms, max 14.72 ms, 817 samples, in-memory dump / 1 ms debounce / 50 ms max-wait / two loopback / race off"
        status: pass
    human_judgment: false
duration: 17min
completed: 2026-11-25
status: complete
---

# Phase 08 Plan 10: Measurement Ledger, Doc Drift Guard and Registry Closure Summary

**docs/PERFORMANCE.md now publishes every phase-8 number from a real `make perf` run with its hardware block, commit and producing command, and `TestPerformanceDoc` formats the thresholds from `internal/testutil` so a code-only threshold change fails the suite; the S-FAIL-15 row is closed against the shipped burst tests and the old standalone latency claim is gone.**

## Performance

- **Duration:** 17 min
- **Started:** 2026-11-25T00:18:00+03:00
- **Completed:** 2026-11-25T00:35:26+03:00
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Measured numbers (second of two `make perf` runs, commit `9cdc0f1`)

Both runs exited 0. The ledger publishes the second run; loads below are from that run.

| Row | Measurement |
|-----|-------------|
| Default profile latency (`TestPerfDefaultProfileLatency`) | p50 598.87 ms, p99 1.095 s, 1 000.1 writes/s open loop, 10 000 × 1 KiB state, 100 hot keys, drain 638.96 ms, ext4, race off, load 4.00 5.75 6.89 |
| Fast profile latency (`TestPerfFastProfileLatency`) | mean 2.82 ms (bound 25ms), p50 2.60 ms, p99 5.55 ms (bound 100ms), max 14.72 ms, 817 samples, in-memory dump, 1 ms debounce / 50 ms max-wait, two loopback adapters, race off, load 3.84 5.26 6.59 |
| Distinct-key burst (`TestPerfBurstDistinctKeys`) | 1 000.1 writes/s achieved (min 990), drain 305.13 ms (bound 5s), 10 000 distinct keys, 1 KiB values, ext4, load 3.64 5.79 6.92 |
| Amplification (`TestPerfAmplification`) | Δ dumps 15, dump 14 977 443 B, `write_bytes` Δ 224 686 080 B, bytes-per-dump 14 979 072 B (bound ≤ 1.1× = 16 475 187 B), logical 15 645 000 B, amplification 14.36×, failures 0, load 4.67 5.77 6.85 |
| Memory point 1 000 × 1 KiB | logical 1 038 000 B, dump 1 492 779 B, RSS 28 639 232 B (27.59×), VmHWM 28 725 248 B, sampled peak +4 096 B (0.00×) |
| Memory point 10 000 × 1 KiB | logical 10 380 000 B, dump 14 927 449 B, RSS 67 739 648 B (6.53×), VmHWM 70 320 128 B, sampled peak +24 576 B (0.00×) |
| Memory point 1 000 × 60 KiB | logical 61 454 000 B, dump 82 044 792 B, RSS 229 527 552 B (3.73×), VmHWM 255 778 816 B, sampled peak +33 480 704 B (0.54×), load 4.12 5.52 6.73 |
| Absent vs live peer (`TestPerfAbsentPeerWriteCost`) | absent 2 249 ns/op, live 8 598 ns/op, ratio 0.26 (bound ≤ 2.0×), 10 000 pending 1 KiB records, both arms 0.00 pending-visits/op, load 3.80 5.31 6.62 |

Host: Intel Core i9-9980HK, 16 logical CPUs, 62.7 GiB RAM, Go 1.27.4 linux/amd64, dump on ext4 over LVM + dm-crypt on the NVMe SSD `nvme0n1` (ROTA 0). This is a workstation, not target-grade eMMC/SD media.

## Accomplishments

- `docs/PERFORMANCE.md` is the one measurement ledger: write-latency model for both profiles with their knobs and conditions, the sequential ceiling, the disk-cost trade linked to the measured amplification, the burst's guarantee/non-guarantee lists, the five-step amplification method with the measured NVMe row and the target-media gap, the memory formula with three measured points, the absent-peer O(1) claim with its deterministic gates, and the exact `make perf` invocation.
- `cmd/sup-kv/performance_test.go` guards it: `TestPerformanceDoc` pins the first-section order, every mandated statement, the forbidden standalone phrase, the per-section hardware block and command, and eight threshold literals formatted from `internal/testutil`; `TestPerformanceDocFamilies` cross-checks every named `supkv_*` family against `internal/metrics/metrics.go`; `TestSpecDeviationsBurstRow` pins the registry closure.
- The S-FAIL-15 row now states the documented distinct-key scale and cites `TestRed_R02_burst`, `TestPerfBurstDistinctKeys` and `TestEngine_burstBoundedConfirmation` with the 990 writes/s and 5 s bounds; the old "holds only on hot keys today" clause is gone.
- `docs/OPERATIONS.md` keeps a three-sentence memory summary, the residual-pause honesty paragraph and a link, with the duplicated "roughly 1.4 times" ratio and the peak-live derivation removed.
- `.planning/PROJECT.md`'s line-43 claim now carries the fast-profile conditions (1 ms debounce / 50 ms max-wait, in-memory dump, two loopback adapters, race off), names `BenchmarkConfirmationLatency`, and links `docs/PERFORMANCE.md`; no `~10 ms` wording survives there.

## Task Commits

Each task was committed atomically:

1. **Task 1: The doc-anchor guard, written before the doc exists** — `02ecad1` (test)
2. **Task 2: Write docs/PERFORMANCE.md from a real make perf run** — `9b23bb6` (docs)
3. **Task 3: Close the S-FAIL-15 row, collapse duplicated memory numbers, condition the standing claim** — `d89b1b4` (docs)

**Plan metadata:** the SUMMARY/docs commit follows.

## Files Created/Modified

- `docs/PERFORMANCE.md` — the ledger: six sections, per-section hardware block + commit + command, measured tables, thresholds named from their constants, and the target-media gap.
- `cmd/sup-kv/performance_test.go` — the honest-scope header, `performanceAnchors`/`performanceForbidden`, `thresholdAnchors()`, `docSections()`, `TestPerformanceDoc`, `TestPerformanceDocFamilies`, `TestSpecDeviationsBurstRow`.
- `docs/OPERATIONS.md` — `## Memory model` reduced to the as-built shape, the residual pause and the ledger link.
- `docs/SPEC-DEVIATIONS.md` — the S-FAIL-15 row rewritten in place, keeping its ID, status and four-column shape.
- `.planning/PROJECT.md` — the Phase-6 testing bullet's latency claim conditioned and linked.

## Decisions Made

- **Publish the second run, record the load.** Two consecutive `make perf` runs exited 0; every row is the second one, and each block's load is stated rather than gated (research A4).
- **Commit beside the block, not inside it.** Test binaries report `module=… (vcs unknown, unknown)`, so the measured commit `9cdc0f1` is recorded beside each block; every other mandated field is in the block itself.
- **One block per section, shared by that section's rows.** The fast row lives in the model section with the default arm's block (Test 8 permits exactly one) and its own logged load is stated inline; the two blocks differ only in that field.
- **Keep the gap a gap.** The amplification row says workstation NVMe/ext4, not target-grade; the eMMC/SD gap is named against the Spec open question #2 and the section says to replace, not annotate, the row if target media appears.
- **Anchor the disk-cost trade explicitly.** `disk cost`, `about 1 dump/s`, `about 20 dumps/s`, `rewrites the whole state` and a real `(#disk-write-amplification)` link are all pinned, so the fast profile cannot be presented as free.
- **Leave ADR 0003 and the review evidence untouched.** Their remaining "~10 ms" strings are an obligation sentence (now satisfied) and historical evidence, not published claims.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical information] Commit recorded beside the hardware block**
- **Found during:** Task 2 (first `make perf` run)
- **Issue:** The plan requires the block to carry the commit, but Go test binaries report `module=… (vcs unknown, unknown)`, so the field is unavailable in the block.
- **Fix:** Each measured section records `Commit: 9cdc0f1` beside its block and states why; the anchor test pins the `commit` mention and the block's `module=` field.
- **Files modified:** `docs/PERFORMANCE.md`, `cmd/sup-kv/performance_test.go`
- **Verification:** `TestPerformanceDoc` green; the block still carries every other mandated field (CPU, logical, RAM, load, disk, Go, race, GOMAXPROCS).
- **Committed in:** `02ecad1` (anchor) and `9b23bb6` (ledger)

**2. [Rule 1 - Bug] Two anchor/prose mismatches and one overstated memory ratio**
- **Found during:** Task 2 (first anchor-test runs and a self-review of the measured rows)
- **Issue:** The prose said "the Spec's open question #2" and "the per-key pending index", which the anchors (and the plan's wording) expect as "Spec open question #2" and "per-key index"; the memory caveat called the 60 KiB dump proportional to the model's 1.37 while the measured ratio is 1.34.
- **Fix:** Matched the anchored wording and restated the memory row with the measured 1.34×, keeping the model's 1.37× as the formula.
- **Files modified:** `docs/PERFORMANCE.md`
- **Verification:** `TestPerformanceDoc` and `TestPerformanceDocFamilies` PASS; the memory paragraph now separates measured ratio from model.
- **Committed in:** `9b23bb6`

---

**Total deviations:** 2 auto-fixed (1 missing critical information, 1 documentation-correctness bug). No threshold, bound or measurement changed; no frozen file touched.

## Issues Encountered

- `go tool golangci-lint fmt ./cmd/sup-kv/` reflowed one over-long `t.Fatalf` line (golines, max-len 200); `make lint` is clean afterwards.
- The first `make perf` run and the RED capture overlap in wall time only after the run finished; the RED tests were executed after run 1 to keep the measured host calm. RED evidence is at `.planning/phases/08-performance/red-evidence/` (untracked, like the other red-evidence directories).
- The pre-existing working-tree modifications (phases 01-06 verification files and the other red-evidence directories) were never staged, modified or reverted.

## User Setup Required

None — documentation and a doc test only; no external service, environment variable or package install (T-08-SC).

## Known Stubs

None. Every anchor is executable and green; no test was skipped and no `<verify>` was left unrun.

## Threat Flags

None. The change adds no endpoint, auth path, file-access pattern or dependency; it publishes measurements already produced by the shipped tests. T-08-25 is mitigated by the per-section block/command assertion and the forbidden standalone phrase; T-08-26 by the threshold literals formatted from `internal/testutil`; T-08-27 by the metric-family cross-check.

## Next Phase Readiness

Ready for 08-11's phase gate. Two follow-ups belong to Phase 9: `docs/API.md`'s write-latency section should link to `docs/PERFORMANCE.md` rather than duplicate it (research PERF-06/DOCS-05), and the amplification gap stays open until target-grade eMMC/SD media is attached — the documented method is ready to replace the workstation row. The thresholds remain operator-facing bounds for a calm Linux host; CI keeps `make bench-smoke` and never runs `make perf`.

## Self-Check: PASSED

- Files: `docs/PERFORMANCE.md`, `cmd/sup-kv/performance_test.go`, `docs/OPERATIONS.md`, `docs/SPEC-DEVIATIONS.md`, `.planning/PROJECT.md` all found.
- Commits: `02ecad1`, `9b23bb6`, `d89b1b4` all ancestors of HEAD.
