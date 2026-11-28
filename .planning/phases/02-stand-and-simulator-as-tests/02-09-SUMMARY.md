---
phase: 02-stand-and-simulator-as-tests
plan: 09
subsystem: testing
tags: [test-13, test-15, decision-coverage, adr, property-shrinking, small-space-draws, replay-command, testutil, depguard-boundary]

# Dependency graph
requires:
  - phase: 02-01..02-08 (same phase, waves 1-2)
    provides: the anchor tests the citations attach to (red_gate_test.go from 02-01, the stand suites from 02-05/02-06, the chaos corpus from 02-07) and the property suites (storage merge_test.go, persist codec_test.go) the small-space draws extend
  - phase: 01 (Phase 1 deliverables)
    provides: docs/adr/0001..0011 — one ADR per owner decision 14.1-14.11 — whose filenames the coverage meta-test enumerates and stats
provides:
  - internal/app/repl/decision_coverage_test.go — TestDecisions_referencedByTests, the 11-row index (decision, ADR file, anchor test) and the meta-test that fails naming any decision without a citation
  - name citations for all 11 owner decisions across 10 test files, so every 14.x decision is traceable to executable evidence
  - TEST-15's dependency-free shrinking mechanism — half the property versions drawn from a shared 3x3x3 (l, c, node) space — in the storage merge laws (small-space file) and the persist round-trip/watermark properties
  - testutil.ReplayCommand plus a fixed NewRNG hint: package-scoped `go test` with the test pinned by -run and the seed through -args, proven by a test that executes the printed command
affects: [/gsd:verify-work's TEST-13/TEST-15 evaluation, Phase 4 PROTO-09 (3+-node real-socket tests take over 14.9's anchor), any future testutil user (the -seed hint format is now the working package-scoped form), Phase 12 P-01 red tests (they carry their own 14.11 citations)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 9031    # chars/4 over the realized diff (36125 bytes of `git diff b518f06..HEAD`)
  tasks: 3
  commits: 4      # MEASURED from the plan-start ledger at SUMMARY write (#3968): git rev-list --count b518f06..HEAD; the 4th is the lint-fixup commit
  plan_head_before: b518f06e89733eed09812debd41bc8140ece42ad
  plan_head_after: 0e79b61

# Tech tracking
tech-stack:
  added: none — zero new module requires (go.mod/go.sum unchanged; the rapid import grep stays 0, D-08's dependency-free mechanism holds). All new code is stdlib (io/fs, os, path/filepath, regexp, runtime, os/exec, context) plus own-module packages
  patterns:
    - word-bounded citation matching: decision 14.1 is matched as `\bdecision 14\.1\b`, so 14.10 cannot satisfy it; the accepted forms are the ADR token and the decision token
    - self-excluding meta-test: the coverage index file is skipped by base name, because its own table spells every ADR token and would satisfy all 11 rows vacuously
    - version-derived payloads: small-space records derive data/tombstone from the version bytes, so equal versions are byte-identical — the same-write premise the equal-version no-op law needs for order independence
    - os-free package-path derivation: the replay hint resolves the caller's import path from the internal/app anchor via runtime.Caller, keeping testutil inside the depguard boundary (no os import in internal/app)
    - printed-command-as-contract: NewRNG and the replay-executor test share one format helper, so the executed command is the printed one, never a log-parsed reconstruction (T-2-18)
    - boundary-class subtests: equal-l, equal-l-equal-c-different-node and adjacent-l get named subtests over the same round-trip assertions, making the distribution change auditable

key-files:
  created:
    - internal/app/repl/decision_coverage_test.go
    - internal/app/storage/prop_small_space_test.go
  modified:
    - internal/app/internal/testutil/rng.go
    - internal/app/persist/codec_test.go
    - internal/app/persist/engine_test.go
    - internal/grpcapi/gate_test.go
    - internal/grpcapi/server_test.go
    - internal/stand/stand_metrics_test.go
    - internal/app/repl/sim_chaos_test.go
    - cmd/sup-kv/main_test.go

key-decisions:
  - "The coverage meta-test excludes its own file and the docs/ subtree from the scan: the index table spells all 11 ADR tokens (would self-satisfy), and docs/reviews/.../evidence/repro is the reviewer's separate module, not this module's suite."
  - "A decision counts as referenced when a test file carries `ADR 000N` or `decision 14.N`, both word-bounded so 14.1 cannot be satisfied by 14.10 or ADR 0001 by ADR 0010. The citations added here use both tokens for readability."
  - "The replay hint is package-scoped and adds -v: the old `./...` form aborted in every package without the -seed flag (verified: cmd/sup-kv rejects -args -seed), and -v is what keeps the seed line visible on a human replay and lets the executor test assert the same test re-ran and passed."
  - "The caller's package path is derived from the internal/app anchor with runtime.Caller rather than os.Getwd/os.Stat: testutil is a non-test file under internal/app, and the depguard rule forbids os there; the anchor is always present because Go's internal rule confines testutil importers to internal/app/... The hardcoded modulePath const closes the derivation."
  - "Small-space records derive payloads from the version (data = version bytes, tombstone when ver[0]%3 == 0): two records with equal versions must be identical, or equal-version draws would make the CRDT laws order-dependent over a state two production writes cannot produce. The equal-version collision subtest separately pins that an incoming equal-version record with different bytes is ignored (full no-op)."
  - "The three boundary classes are named subtests in TestCodec_roundTripProperty with the assertions extracted into assertCodecRoundTrip — the plan's 'generator distribution only, no assertion changes' holds by construction; TestWatermark_fullProperty's max-scan subtest draws the same mixed distribution."
  - "red_gate_test.go (14.5/ADR 0005) and stand_nondarwin_test.go (14.10/ADR 0010) already carried complete citations, so they were not edited; the meta-test confirms them. Six files received the ten citation insertions the mapping required."

patterns-established:
  - "Coverage meta-tests scan source text only (never execute it), word-bound their tokens, and exclude their own index file — the three properties that keep a self-documenting table from passing itself"
  - "A printed replay command is part of the test contract: one exported format helper builds it, a test executes it, and the printed seed reproduces the exact generator state (verified: the replayed command reprints the same seed pair)"
  - "Property generators that deliberately collide versions must tie every record field to the version, or the property would assert laws over states the production write path cannot create"

requirements-completed: [TEST-13, TEST-15]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "The TEST-13 decision coverage meta-test: all 11 owner decisions 14.1-14.11 are name-referenced in test files and the meta-test fails naming any gap (decision, ADR file, expected anchor)."
    requirement: TEST-13
    verification:
      - kind: unit
        ref: "internal/app/repl/decision_coverage_test.go#TestDecisions_referencedByTests (go test ./internal/app/repl/ -run '^TestDecisions_referencedByTests$')"
        status: pass
      - kind: unit
        ref: "grep -rl 'ADR 000' internal/ cmd/ --include='*_test.go' | wc -l => 9 (gate: >= 8)"
        status: pass
    human_judgment: false
  - id: D2
    description: "TEST-15 small-space property in the storage merge suite: half the record versions drawn from a 3x3x3 (l, c, node) space, the three CRDT laws, the pointwise-supremum law and the equal-version full-no-op pin."
    requirement: TEST-15
    verification:
      - kind: unit
        ref: "internal/app/storage/prop_small_space_test.go#TestProp_mergeSmallSpace (go test -race ./internal/app/storage/ -run '^TestProp_')"
        status: pass
    human_judgment: false
  - id: D3
    description: "Working printed replay command: testutil.NewRNG prints a package-scoped `go test <pkg> -run '<test>' -v -args -seed=N` hint and a test executes exactly that string through the shell, asserting exit 0 plus the same test's re-run and pass."
    requirement: TEST-15
    verification:
      - kind: unit
        ref: "internal/app/storage/prop_small_space_test.go#TestProp_replayCommandExecutes (go test -race ./internal/app/storage/ -run '^TestProp_')"
        status: pass
      - kind: other
        ref: "manual replay from the repo root reproduced the same property seed pair (417992386,3807866397)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The codec round-trip and watermark properties explore the same small (l, c, node) space in half their draws, with equal-l, equal-l-equal-c-different-node and adjacent-l as named boundary subtests; assertions unchanged."
    requirement: TEST-15
    verification:
      - kind: unit
        ref: "internal/app/persist/codec_test.go#TestCodec_roundTripProperty and #TestWatermark_fullProperty (go test -race ./internal/app/persist/ -run '^TestCodec_roundTripProperty$|^TestWatermark')"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 09: Decision Coverage and Dependency-Free Property Shrinking Summary

**All 11 owner decisions 14.1-14.11 are name-referenced from tests behind a self-documenting meta-test that fails naming gaps, and TEST-15 shrinking lands as shared 3x3x3 (l, c, node) draws plus a package-scoped replay command that a test actually executes.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-11-21T12:10:00Z
- **Completed:** 2026-11-21T12:25:00Z
- **Tasks:** 3
- **Files modified:** 10 (2 created, 8 modified)
- **Commits:** 4 measured (`git rev-list --count b518f06..HEAD`)
- **Realized diff:** 36125 bytes across 10 files (4 created/modified test files carry the substance; six files are doc-comment citation edits)

## Execution composition (from the plan's footprint note)

The plan predicted 12 files as "8 one-line citation edits + 2 new files + rng fix + generator extension". What actually landed, measured from the diff:

| Component | Planned | Actual |
|-----------|---------|--------|
| One-line citation edits | 8 files listed (engine_test, gate_test, server_test, red_gate_test, stand_nondarwin_test, stand_metrics_test, sim_chaos_test, main_test) | **6 files edited, 10 citation insertions** (engine_test ×3, server_test ×2, gate_test, stand_metrics_test, sim_chaos_test, main_test). red_gate_test.go and stand_nondarwin_test.go already carried complete citations (14.5/ADR 0005, 14.10/ADR 0010) — verified by the meta-test, deliberately not re-edited |
| New test files | 2 (decision_coverage_test.go, prop_small_space_test.go) | 2 — 153 and 187 lines |
| Helper fix | 1 (testutil/rng.go replay hint) | 1 — +73/−2 lines (hint + exported format helper + os-free package-path derivation) |
| Generator extension | 1 (persist/codec_test.go small-space draws) | 1 — +170/−31 lines (mixed generator, three boundary fixtures, extracted assertion helper) |

The token footprint (9031 on the estimate scale) is an order of magnitude under the plan's 90k estimate; the composition is mechanical breadth, not deep implementation.

## Accomplishments

- **TEST-13 closed with an executable index.** `TestDecisions_referencedByTests` enumerates all 11 decisions (decision, ADR file, anchor test), stat-checks every ADR file, and fails naming decision + ADR file + expected anchor for any gap. It found 7 missing before the citations and passes now.
- **All 11 decisions cited by name.** 14.1/14.3/14.11 on the persist anchors, 14.2/14.7 on the gRPC API surface, 14.6 on the startup-gate backstop, 14.4/14.8 on the metrics pins, 14.9 on the 2..5-node chaos convergence, with 14.5 and 14.10 already on the red-gate and non-darwin files. The ADR-citing test-file count rose from 2 to 9 (gate ≥ 8).
- **The broken replay hint is fixed.** The old `go test ./... -run ... -args -seed=N` aborted in every package without the flag (verified: `cmd/sup-kv` fails with "flag provided but not defined: -seed"). NewRNG now prints the package-scoped form, derived os-free from the `internal/app` anchor via `runtime.Caller` so the depguard boundary stays intact.
- **The printed command is proven by execution.** `TestProp_replayCommandExecutes` asks `testutil.ReplayCommand` for the exact string NewRNG prints, checks it is package-scoped, runs it through the shell and asserts exit 0 plus the same test's re-run and pass. A manual replay of a printed subtest command reprinted the identical seed pair.
- **TEST-15 shrinking is live, dependency-free.** A shared 3x3x3 `(l, c, node)` space feeds half the versions in the storage merge laws and the persist round-trip/watermark properties; equal-l, equal-l-equal-c-different-node and adjacent-l are named boundary subtests. `go.mod`/`go.sum` unchanged; the `pgregory.net/rapid` import grep stays 0.

## Task Commits

Each task was committed atomically:

1. **Task 1: decision-to-test mapping + name citations** - `cb3eba5` (test)
2. **Task 2: small-space property draws + working replay command** - `ec9eedd` (test)
3. **Task 3: extend small-space draws to codec/watermark properties** - `9bbe9b1` (test)
4. **Lint fixup for the new helpers** - `0e79b61` (test)

**Plan metadata:** the SUMMARY/STATE/ROADMAP commit that follows this file (docs).

## Files Created/Modified

- `internal/app/repl/decision_coverage_test.go` - created: the 11-row TEST-13 index + `TestDecisions_referencedByTests` (word-bounded citation match, self-exclusion, ADR existence check)
- `internal/app/storage/prop_small_space_test.go` - created: `TestProp_mergeSmallSpace` (laws, pointwise supremum, equal-version no-op) and `TestProp_replayCommandExecutes`
- `internal/app/internal/testutil/rng.go` - modified: package-scoped replay hint, `ReplayCommand`, `callerPackagePath`/`runPattern` helpers
- `internal/app/persist/codec_test.go` - modified: mixed small-space generator, three boundary fixtures, `assertCodecRoundTrip`, boundary subtests; watermark max-scan subtest draws the mixed distribution
- `internal/app/persist/engine_test.go` - modified: 14.1, 14.3, 14.11 doc citations on the three anchors
- `internal/grpcapi/gate_test.go` - modified: 14.6 citation on `TestGate_opensOnTimeout`
- `internal/grpcapi/server_test.go` - modified: 14.2 citation on the tombstoned create-if-absent block, 14.7 on `TestServer_setGetRoundTrip`
- `internal/stand/stand_metrics_test.go` - modified: ADR 0004/0008 tokens added to the existing 14.4/14.8 citations
- `internal/app/repl/sim_chaos_test.go` - modified: 14.9 citation on `TestChaos_convergence`
- `cmd/sup-kv/main_test.go` - modified: 14.8 citation on `TestMetrics_noDefaultListener`

## Decisions Made

- The meta-test excludes its own file and the `docs/` subtree; citations match `ADR 000N` or `decision 14.N` word-bounded (so 14.10 never satisfies 14.1).
- The replay hint is package-scoped with `-v`; the package path comes from the `internal/app` anchor via `runtime.Caller` (no `os` import — depguard), and `ReplayCommand` shares the one format helper so the executed command is the printed command.
- Small-space records derive every field from the version, keeping equal-version records byte-identical; the equal-version collision subtest separately pins the full-no-op behavior for differing payloads.
- The three codec boundary classes are named subtests over one extracted assertion helper, so the distribution change provably left assertions untouched.
- 14.5 and 14.10 needed no edits — their files already carried complete citations (confirmed by the meta-test rather than churned for symmetry).

## Deviations from Plan

### Plan-vs-reality adjustments

**1. [Composition] Two of the eight enumerated citation files needed no edit**
- **Found during:** Task 1
- **Issue:** The plan's composition note enumerates red_gate_test.go and stand_nondarwin_test.go among the eight one-line citation edits, but both already carried complete citations (14.5/ADR 0005 and 14.10/ADR 0010) from earlier plans.
- **Fix:** Left them untouched; the meta-test confirms both references. The other six files received ten citation insertions (engine_test.go covers three decisions, server_test.go two).
- **Files modified:** none for these two (deliberate)
- **Verification:** `TestDecisions_referencedByTests` passes with all 11 rows; ADR-citing file count = 9
- **Committed in:** cb3eba5 (Task 1)

**2. [Form] The printed replay command carries `-v`**
- **Found during:** Task 2
- **Issue:** The plan's suggested form was `go test <pkg> -run '<TestName>' -args -seed=<N>`. Without `-v`, a replay hides the `property seed:` log line (Go suppresses logs for passing tests) and the executor test cannot assert that the same test re-ran.
- **Fix:** The printed form adds `-v` before `-args`. The command still targets exactly one package and one test.
- **Files modified:** internal/app/internal/testutil/rng.go
- **Verification:** `TestProp_replayCommandExecutes` passes; manual replay reprints the same seed pair
- **Committed in:** ec9eedd (Task 2)

**3. [Lint] Strict-gate fixups split into a fourth commit**
- **Found during:** the post-task `make lint` run
- **Issue:** Five findings on the new code: mnd (magic frame count), modernize (strings.CutLast), godoclint (`[runtime.Caller]` link), noinlineerr (stat check), paralleltest (defer + t.Parallel).
- **Fix:** Named the frame-distance const, used `strings.CutLast`, linked the stdlib call, plain assignment, and `t.Cleanup(cancel)`. Behavior unchanged.
- **Files modified:** internal/app/internal/testutil/rng.go, internal/app/repl/decision_coverage_test.go, internal/app/storage/prop_small_space_test.go
- **Verification:** `make lint` green; the targeted suites re-ran green afterwards
- **Committed in:** 0e79b61

---

**Total deviations:** 3 plan-vs-reality adjustments (1 composition, 1 command form, 1 lint fixup)
**Impact on plan:** None on scope or semantics. The composition table records the honest footprint; the `-v` addition strengthens the replay evidence; the lint fixup is AGENTS.md deviation-3 compliance.

## Issues Encountered

- The `./...` replay form was confirmed broken before fixing it: `go test ./cmd/sup-kv/ -run '^TestNothingMatches$' -args -seed=1` fails with "flag provided but not defined: -seed", while the package-scoped storage/version forms pass. This is the evidence behind the hint change.
- The coverage meta-test's first run named exactly the 7 missing decisions (14.1, 14.2, 14.3, 14.6, 14.7, 14.9, 14.11) — the failure message format was exercised for real, not only asserted.
- `internal/app/repl` is slow to compile (~4-8 s per run) because of the simulator suite; the meta-test's own runtime is milliseconds. No timeout risk.
- The untracked `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261119T193227Z/` directory predates this plan (left by the red-runner) and was deliberately not staged.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TEST-13 and TEST-15 are green and enforced; `/gsd:verify-work` can evaluate both from the coverage block.
- The small-space draw constants are duplicated (not shared) between storage and persist test packages because both are external test packages and the shared home for a generator would be testutil; if a third suite needs the space, promote the constants there.
- Phase 4 (PROTO-09) should take over 14.9's anchor when real-socket 3+-node tests land: update `decisionIndex`'s 14.9 anchor row together with the citation, or the index drifts from the evidence.
- The remaining wave-3 plans (02-10..02-13) are untouched by this plan's files.

## Self-Check: PASSED

- Created files exist: decision_coverage_test.go, prop_small_space_test.go, rng.go, codec_test.go — all FOUND
- Commits exist: cb3eba5, ec9eedd, 9bbe9b1, 0e79b61 — all ancestors of HEAD
- Task 1 acceptance: `go test ./internal/app/repl/ -run '^TestDecisions_referencedByTests$'` ok; ADR-citing test files = 9 (≥ 8)
- Task 2 acceptance: `go test -race ./internal/app/storage/ -run '^TestProp_'` ok; rapid import grep = 0
- Task 3 acceptance: `go test -race ./internal/app/persist/ -run '^TestCodec_roundTripProperty$|^TestWatermark'` ok
- Plan verification: go.mod/go.sum unchanged across b518f06..HEAD; `make lint` green; `go test -race ./internal/app/... ./internal/grpcapi/` green

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
