---
phase: 02-stand-and-simulator-as-tests
plan: 04
subsystem: persistence
tags: [fsops, seam, persistfile, atomic-write, durability, blocking-proof, d07, test-03]

# Dependency graph
requires:
  - phase: 03-persistence (v1.0 shipped)
    provides: the saveAtomically sequence this plan extracts behind the seam (MkdirAll → synced temp → rotate → rename → dir-fsync), the dumpfile suite that pins bit-identity through the refactor, the store mutex contract Save/CreateSnapshot serialize on, and the engine's synchronous dump path (Run serializes every Save; waiters block on the dump)
provides:
  - fsops interface (internal/persistfile, unexported) — the five TEST-03 order steps as five methods (write / fsync / rotate / rename / syncDir), one per durability step so each is individually order-observable and failure-injectable; per-save temp state held in the implementation value, safe under the store mutex
  - fileFSOps — the default implementation performing the exact real operations (same CreateTemp naming, same syscall order, same .prev rotation, same dir sync); the injection point is FileStore's unexported ops field, exported constructors unchanged
  - The TEST-03 fakes as executable proof: recordingFSOps (records + delegates to the real impl — the order assertion is taken over the production path's actual operations), failingFSOps (per-step injection through the production hygiene helpers), blockingDumpPort (Save parks until released — Set does not return while the dump stands)
affects: [02-16 (persist.Open clock-seeding proof lands on this same Save path — behavior-identity is what its composition proof rides), phase 5 persistence plans (P-01..P-10 fixes get their injection points here), TEST-03's stand-level process half (SIGKILL restart, internal/stand scenarios)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 7933    # chars/4 over the realized diff (31729 chars: 704 insertions / 76 deletions across 5 files)
  tasks: 2
  commits: 2      # MEASURED from the plan-start ledger at SUMMARY write (#3968)
  plan_head_before: 349a814afb2b91126af80696f680b25c38843f79
  plan_head_after: 26e935ee1e9b8eed36ec64bce20c34f22c03eb6f

# Tech tracking
tech-stack:
  added: none — stdlib only (os/errors/bytes/slices in the seam and test files); zero new module dependencies (zero-install phase per the Package Legitimacy Audit)
  patterns:
    - one-method-per-durability-step seam: the five TEST-03 operations are five interface methods, fsync its own step (individually order-observable and failure-injectable); per-save state (open temp handle + path) lives in the implementation value, safe because the store mutex serializes every save
    - failure hygiene as shared production helpers: closeRemoveTemp/removeTemp are fileFSOps methods the natural failure paths AND the failing fake's short-circuits invoke, so the failure matrix observes real dump hygiene, never fake-side bookkeeping
    - delegating recording fake: the order fake wraps the real fileFSOps, so the exact-sequence assertion is evidence about the production Save path, and the rotation-chain probe asserts real files (first main → .prev, second save replaces it)
    - synctest bubble for the blocking proof: virtual time makes the not-returned window deterministic — no real-time flake surface in the D-07 proof

key-files:
  created:
    - internal/persistfile/fsops.go
    - internal/persistfile/fsops_test.go
    - internal/app/persist/engine_blocking_test.go
  modified:
    - internal/persistfile/dumpfile.go
    - internal/persistfile/snapshot.go

key-decisions:
  - "Failure hygiene is impl-side in the seam, mirroring today's split exactly: write/fsync failures close + remove the temp inside fileFSOps (syncedTemp's old internal hygiene), rotate/rename failures remove it (saveAtomically's old caller-side remove, relocated), syncDir cleans nothing (the temp already became main) — saveAtomically is now a pure MkdirAll + five-call sequence with net observable behavior bit-identical"
  - "The recording fake delegates to the embedded real fileFSOps: the op-name order assertion (write → fsync → rotate → rename → dir-fsync, one ordered slice comparison) is taken over the production Save path's actual operations, and the two-consecutive-Saves rotation probe asserts the real files the delegation produced"
  - "The failing fake short-circuits BEFORE the wrapped real op and performs the step's own hygiene through the shared production helpers (closeRemoveTemp for fsync, removeTemp for rotate/rename): the matrix observes the seam contract's real effects — injected error surfaces, no temp litter, main never partial, recovery Save succeeds"
  - "Internal-test placement follows the internal/stand precedent (STATE-recorded): a justified //nolint:testpackage keeps the plan's artifact name fsops_test.go while the fakes reach the unexported FileStore.ops injection point — the plan mandates the internal test (package persistfile), and testpackage's external-package rule cannot express that pairing; injection is a direct field write (zero API surface, exported constructors unchanged)"
  - "The blocking proof rides a synctest bubble with engine_test.go's startDumper driving pattern and an in-memory blockingDumpPort (real I/O inside a bubble is not durably blocking, Pitfall 6): waitParked gives a deterministic dump-standing point, the not-returned window is virtual, and release → Set returns confirmed (nil)"

patterns-established:
  - "Seam steps carry their own failure hygiene in shared production helpers, so injection fakes exercise the same hygiene code the natural failure paths run — the failure matrix is evidence about production cleanup, not fake cleanup"
  - "Recording fakes for durability-order contracts delegate to the real implementation: the recorded sequence is then the production path's sequence, and file-level probes (rotation chain) ride the same fake"

requirements-completed: [TEST-03]

coverage:
  - id: D1
    description: "fsops interface + fileFSOps — the five TEST-03 order steps as five methods (write: CreateTemp + Write, handle held; fsync: syncDurable + Close; rotate: guarded main→.prev rename; rename: temp→main; syncDir: dir fsync through the platform flush), each doc comment naming its TEST-03 step; failure hygiene in closeRemoveTemp/removeTemp"
    requirement: TEST-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ → ok (the full existing suite — atomicRotation, fallbackChain, saveFailureHygiene, snapshotSmoke, snapshotConcurrentWithSave, snapshotWaitsOutSaveWindow — green through the extraction: bit-identity witness)"
        status: pass
      - kind: other
        ref: "grep 'TEST-03 order step' internal/persistfile/fsops.go → 5 interface-method doc comments; var _ fsops = (*fileFSOps)(nil) compiles the seam"
        status: pass
    human_judgment: false
  - id: D2
    description: "saveAtomically drives the seam — MkdirAll stays caller-side setup, then write → fsync → rotate → rename → syncDir in the exact production order; FileStore.Save and CreateSnapshot pass the shared ops value (serialized by the store mutex); exported constructors unchanged; nothing added to internal/port"
    requirement: TEST-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ ./internal/grpcapi/ ./cmd/sup-kv/ → all ok (every persistfile consumer green through the refactor)"
        status: pass
      - kind: other
        ref: "grep -c 'saveAtomically' internal/port/*.go → 0 (the seam is persistfile-local, D-07); .golangci.yml diff empty (sup-kv-app-no-io boundary untouched)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Order pin: TestFsops_saveOperationOrder — one Save through FileStore records EXACTLY write → fsync → rotate → rename → dir-fsync (single ordered slice comparison) with the argument paths (write/syncDir: dump dir; rotate: main+prev; rename: main) and the delegated steps placing the generation; TestFsops_saveOperationOrder_rotation_chain — two consecutive Saves each produce the five-step sequence and the real files show first main → .prev, second replacing main"
    requirement: TEST-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_' → ok (saveOperationOrder + rotation_chain pass)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Failure matrix: TestFsops_failureAtEachStep — one row per step (write, fsync, rotate, rename, syncDir); each row's injected error surfaces from Save, no temp litters (exact directory-entry oracle), main never partial (literal-bytes oracle: old gen, new gen, or the rotate-window absence the fallback chain absorbs), and a subsequent successful Save recovers"
    requirement: TEST-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_' → ok (all five rows pass, incl. the fsync row's handle-closed + temp-removed hygiene and the rename row's .prev residue)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Empty-dump round trip (probe empty case): TestFsops_emptyDumpRoundTrip — Save of an empty dump leaves a 0-byte main generation, Load with a permissive oracle returns zero records with no error"
    requirement: TEST-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_' → ok (emptyDumpRoundTrip passes)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Blocking proof (D-07): TestEngine_blockingSaveHoldsSet — blockingDumpPort parks Save; inside a synctest bubble, Set issued from a goroutine has NOT returned when the not-returned window fires (the expected timeout), then releasing the standing Save lets Set return confirmed (nil) with exactly 1 completed save — the dump path is synchronous end to end"
    requirement: TEST-03
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_blocking|^TestEngine_setBlocked' → ok (blockingSaveHoldsSet passes)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Gates: strict lint green repo-wide with the new files in tree; .golangci.yml byte-unmodified; fakes exist only in _test.go files (threat T-2-07); the default implementation stays the real file operations"
    requirement: TEST-03
    verification:
      - kind: other
        ref: "make lint → exit 0 (config verify + run ./...); make test → exit 0 (14 packages, persistfile 89.7% / app/persist 97.7% coverage); git diff .golangci.yml → empty; grep for the fakes → fsops_test.go / engine_blocking_test.go only"
        status: pass
    human_judgment: false

# Metrics
duration: 22min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 4: fsops Seam — Order, Failure Matrix, Blocking Proof Summary

**The atomic-dump write path is extracted behind a five-method `fsops` seam in `internal/persistfile` (write → fsync → rotate → rename → dir-fsync, one method per TEST-03 order step), behavior bit-identical under the existing suites, and TEST-03's seam half is now executable evidence: a delegating recording fake pins the exact operation order over the production path, a failing fake injects at each of the five steps with dump hygiene intact (no partial main, no temp litter, recovery), the empty-dump round trip is a named case, and a blocking `Save` fake proves at the engine boundary that `Set` does not return while the dump stands (D-07).**

## Performance

- **Duration:** 22 min (1191 s)
- **Started:** 2026-11-19T20:25:01Z
- **Completed:** 2026-11-19T20:44:52Z
- **Tasks:** 2
- **Files changed:** 5 (3 created, 2 modified; +704/−76)

## Accomplishments

- **`internal/persistfile/fsops.go` (new)** — the unexported `fsops` interface: the five ordered durability operations TEST-03 names as five methods, fsync deliberately its own step (not bundled inside write) so the durability-critical moment is individually order-observable and failure-injectable; every method's doc comment names its TEST-03 order step. `fileFSOps` is the default implementation performing the exact real operations the save path performed before the extraction — same `os.CreateTemp` naming (`dump-*`), same syscall order (open temp → write → fsync → close → rotate → rename → dir sync), same targetExists-guarded `.prev` rotation, same platform flush (`syncDurable`: F_FULLFSYNC on darwin) — with per-save state (the open temp handle and its path) held in the value, safe because FileStore's store mutex serializes Save against CreateSnapshot and concurrent Saves. Failure hygiene lives in the shared `closeRemoveTemp`/`removeTemp` helpers: write/fsync failures close + remove the temp (syncedTemp's old internal hygiene), rotate/rename failures remove it (saveAtomically's old caller-side remove, relocated), syncDir cleans nothing.
- **`saveAtomically` refactored to drive the seam** (dumpfile.go, snapshot.go call site): MkdirAll of the parent stays caller-side (setup, not one of the five), then the five ordered calls. `FileStore.Save` and `CreateSnapshot` pass the shared `f.ops` value; the exported `New(dumpPath)` signature is unchanged and callers are unaffected (verified: grpcapi, cmd/sup-kv, app/persist all green). Nothing was added to `internal/port` — the seam is persistfile-local per D-07 and the depguard boundary (T-2-07).
- **`internal/persistfile/fsops_test.go` (new, package persistfile)** — the TEST-03 order and failure proofs: `recordingFSOps` records every seam call (op + argument paths) and delegates to the embedded real `fileFSOps`, so `TestFsops_saveOperationOrder`'s single ordered slice comparison (`write → fsync → rotate → rename → dir-fsync`) is evidence about the production Save path's actual operations, with the argument paths asserted (dump dir for write/syncDir, main+prev for rotate, main for rename); `TestFsops_saveOperationOrder_rotation_chain` runs two consecutive Saves and asserts both the per-Save five-step sequences and the real rotation chain (first main becomes `.prev`, second Save rotates again and replaces main). `failingFSOps` injects at each of the five steps through the production hygiene helpers; `TestFsops_failureAtEachStep` asserts per row that the injected error surfaces from Save, the directory holds exactly the expected residue (no `dump-*` litter), the main path holds exactly the old generation, the new generation, or the rotate-window absence the fallback chain absorbs (never partial bytes), and a subsequent successful Save recovers. `TestFsops_emptyDumpRoundTrip` pins the empty probe: Save of an empty dump leaves a 0-byte main, Load returns zero records with no error.
- **`internal/app/persist/engine_blocking_test.go` (new, package persist_test)** — D-07's blocking proof at the engine boundary: `blockingDumpPort` parks Save on a release channel (`parked` closes once for a deterministic dump-standing point); `TestEngine_blockingSaveHoldsSet` composes `persist.New` with it, drives Run via the suite's `startDumper` pattern inside a synctest bubble, issues Set from a goroutine, and asserts — expecting the timeout — that Set has NOT returned while the dump stands, then releases Save and asserts Set returns with its confirmation (nil) and exactly 1 completed save. The dump path is synchronous end to end; the write-confirmation contract is machine-pinned.
- **Bit-identity held through the refactor**: the full existing persistfile suite (atomicRotation, fallbackChain, saveFailureHygiene, snapshotSmoke, snapshotConcurrentWithSave, snapshotWaitsOutSaveWindow) plus every consumer (app/persist, grpcapi, cmd/sup-kv) ran green unchanged; `make lint` and `make test` (14 packages) green.

## Task Commits

1. **Task 1: fsops seam — extract write/fsync/rotate/rename/dir-fsync** - `7af76ad` (feat)
2. **Task 2: recording/failing/blocking fakes — order, failure matrix, Set-does-not-return** - `26e935e` (test)

_Measured from the plan-start ledger `349a814` (`git rev-list --count 349a814..HEAD` at SUMMARY write): 2 task commits; cross-checked against `gsd-tools check evaluation-scope --plan 02-04 --commits-only` (resolved, exactly these 2)._

## Files Created/Modified

- `internal/persistfile/fsops.go` — created (+168): the fsops interface, fileFSOps default implementation, the shared hygiene helpers, targetExists (moved from dumpfile.go).
- `internal/persistfile/dumpfile.go` — modified (+52/−76): FileStore gains the ops field (default `&fileFSOps{}`), Save/CreateSnapshot drive the seam, saveAtomically becomes the MkdirAll + five-call sequence, syncedTemp/writeSynced/placeFile/syncDir fold into the seam.
- `internal/persistfile/snapshot.go` — modified (+1/−1): CreateSnapshot passes `f.ops` to saveAtomically.
- `internal/persistfile/fsops_test.go` — created (+369): recording/failing fakes, order + rotation-chain + failure-matrix + empty-round-trip tests.
- `internal/app/persist/engine_blocking_test.go` — created (+126): blockingDumpPort + TestEngine_blockingSaveHoldsSet.

## Decisions Made

- **Failure hygiene is impl-side in the seam, mirroring today's split exactly.** syncedTemp's internal close+remove (write/fsync failures) stays inside the implementation; saveAtomically's old caller-side temp remove (placeFile failures) relocates into rotate/rename; syncDir cleans nothing. saveAtomically becomes a pure five-call sequence with net observable behavior bit-identical — the existing hygiene suite (saveFailureHygiene) proves it.
- **The recording fake delegates to the real implementation.** The plan's "recording fake implementing the fsops interface records each call" is realized as a wrapper over the embedded real fileFSOps: the op-name order assertion is then evidence about the production Save path (a swapped default fails it), and the rotation-chain probe asserts real files — the strongest honest reading of "asserted by the recording fake sequence" for a probe about file movement.
- **The failing fake short-circuits before the real op and performs the step's own hygiene via the shared production helpers.** Injection at a step simulates that step failing; the cleanup the production step's failure path performs (closeRemoveTemp for fsync, removeTemp for rotate/rename) is invoked by the fake through the same production code, so the matrix observes the seam contract's real effects rather than fake-side bookkeeping.
- **Injection point: direct field write from the internal test.** The plan offered "a test-only setter or an internal constructor parameter"; neither is needed — the mandated internal test (package persistfile) assigns the unexported `f.ops` directly, keeping zero API surface and the exported constructors unchanged.
- **Internal-test placement follows the internal/stand precedent.** The strict config's testpackage linter exempts only `*_internal_test.go` filenames; keeping the plan's artifact name `fsops_test.go` with `package persistfile` uses the STATE-recorded justified `//nolint:testpackage` pattern (the plan mandates the internal test so the fakes reach the unexported injection point — testpackage's external-package rule cannot express that pairing).
- **The blocking proof rides a synctest bubble.** engine_test.go's established pattern (in-memory fake, `startDumper`, virtual time): the not-returned window is deterministic and instant, no real-time flake surface in a proof that will run at `-race -count=20` nightly.

## Deviations from Plan

None - plan executed exactly as written. (The final artifact set matches the plan's frontmatter and Task 2 text verbatim — including the `fsops_test.go` name, resolved mid-execution via the recorded internal/stand nolint precedent; see Issues Encountered. The Open clock-seeding proof is sibling plan 02-16 per the plan's own header, not a deviation.)

## Issues Encountered

- **The plan-start ledger carried a stale value (recurring, same as 02-03).** `gsd-plan-head-before-02-04` already existed on disk from a prior milestone's plan-02-era execution, pointing at `bedbd97c` (`docs(02): gap-closure plan...`, 66 commits back), so the first measurement read the repo's whole history (68). Caught at SUMMARY measurement time (the count contradicted the session's two commits and the evaluation-scope resolution), corrected to the true plan-start HEAD `349a814` (verified parent of this plan's first commit), re-measured: 2 commits. No code impact.
- **First failing-matrix run failed on a wrong TEST expectation, not a code bug.** The rename row's litter oracle expected `dump.json` still present, but rotate had already moved it — the directory correctly held only `dump.json.prev` (exactly the rotate-window residue the fallback chain absorbs). Fixed the test's `wantEntries` to derive from the residue table; production behavior was correct throughout.
- **Lint pass 1 on the new test file produced five findings** (embeddedstructfieldcheck ×2, gci, noinlineerr, an unused nolintlint directive) — all fixed in the same task; the second pass and `make lint` were clean.
- **An amend mishap, caught and corrected.** Renaming `fsops_internal_test.go` → `fsops_test.go` and amending Task 2's commit staged only the addition; the commit file-list check caught the renamed-out file still present, `git rm --cached` + re-amend produced the final commit containing exactly the two task files.

## Known Stubs

None. Every deliverable is complete and wired: the seam is the production Save path, the fakes are complete test values, and every test assertion ran green. (TEST-03's stand-level process half — single node Set → OK, SIGKILL, start, Get succeeds — is the internal/stand scenarios' scope per the phase's requirement map; this plan delivers the seam half its plan owns.)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Plan 02-16 (wave 2, depends_on this plan)** lands `open_seed_test.go` (TEST-07, persist.Open clock seeding) on this same Save path — the refactor's behavior-identity, pinned by the existing dumpfile suite plus the order test, is what its composition proof rides.
- **TEST-03 is now evidence-backed at the seam level**: exact order (recording fake over the production path), failure at each of the five steps with hygiene intact, empty round trip, and the blocking proof that Set does not return while the dump stands.
- **Phase 5 persistence plans (P-01..P-10) get their injection points**: any dump-path fix lands behind the seam, and the failing/blocking fakes are the harness for it.
- No blockers. The depguard architecture boundary is untouched (.golangci.yml diff empty); `make lint` and `make test` green repo-wide.

## Self-Check: PASSED

Re-run on the committed tree (`26e935e`):

- Key files exist: `internal/persistfile/fsops.go`, `internal/persistfile/fsops_test.go`, `internal/persistfile/dumpfile.go`, `internal/persistfile/snapshot.go`, `internal/app/persist/engine_blocking_test.go` — all FOUND.
- Commits exist: `7af76ad`, `26e935e` — both ancestors of HEAD; `git rev-list --count 349a814..HEAD` → 2 (ledger corrected from the stale `bedbd97c`, see Issues).
- `gsd-tools check evaluation-scope --plan 02-04 --commits-only --raw` → resolved, exactly 2 commits, exit 0 (not 69).
- Success criterion 1 (seam extracted, Save bit-identical): existing persistfile suite green through the refactor — `go test -race -count=1 ./internal/persistfile/` → ok; consumers `./internal/app/persist/ ./internal/grpcapi/ ./cmd/sup-kv/` → ok.
- Success criterion 2 (order + failure matrix): `go test -race -count=1 ./internal/persistfile/ -run '^TestFsops_'` → ok (saveOperationOrder, saveOperationOrder_rotation_chain, failureAtEachStep all five rows, emptyDumpRoundTrip).
- Success criterion 3 (blocking proof): `go test -race -count=1 ./internal/app/persist/ -run '^TestEngine_blocking|^TestEngine_setBlocked'` → ok (blockingSaveHoldsSet).
- Success criterion 4 (named cases): rotation chain + empty round trip green (criterion 2's run).
- Plan verification: `go test -race ./internal/persistfile/ ./internal/app/persist/` → ok/ok; `make lint` → exit 0; `git diff .golangci.yml` → empty; `make test` (`go test -race -cover ./...`) → exit 0, 14 packages ok (persistfile 89.7%, app/persist 97.7%).
- Threat T-2-07: the fakes live only in `fsops_test.go` / `engine_blocking_test.go` (`_test.go` files); the default implementation is the real file operations; `.golangci.yml` unmodified.

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-19*
