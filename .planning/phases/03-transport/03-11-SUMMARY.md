---
phase: 03-transport
plan: 11
subsystem: transport-dead-code-removal
tags: [xport-13, q-09, dead-code, confirm-waiters, ownhandle, persist-parse, fsync-darwin, file-sync, darwin-cross-build, spec-deviations]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-10's adapter construction (listenDatagram, packetConn, the bound-address derivation) whose repl.New call this plan simplifies"
  - phase: 03-transport
    provides: "03-04's repl bench fixture, one of the compile-enforced ownHandle call sites"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the fsops five-step durability seam whose File.Sync steps this plan re-homes onto the standard library"
provides:
  - "Q-09 dead code gone: Engine.lastLocal, the cover parameter, the above-cover branch and waiter.ver; the capture proof is the confirmation argument"
  - "Exported persist.Parse deleted; Decode is the single readability oracle, and its tests assert the error directly"
  - "Core.ownHandle and the repl.New parameter deleted across every direct call site (16 sites in 9 files), keeping the four-argument constructor"
  - "internal/persistfile/fsync_darwin.go and fsync_other.go deleted; fsops calls os.File.Sync, whose darwin path is F_FULLFSYNC with EINTR retry and ENOTSUP fallback"
  - "S-PERS-03 updated to the as-built durability path; ADR 0010 checked and left untouched (no fsync reference)"
affects:
  - "Phase 5's P-09 inherits a clean fsops: the remaining work is orphan-temp collection and parent-directory sync policy, not the file flush"
  - "Every future repl.New caller uses the four-argument form; a stale five-argument call is now a compile error"
  - "03-12 and later read a persist facade with no Parse oracle; Decode is the only decode entry point"

actuals:
  tokens: 10437
  tasks: 2
  commits: 2
  plan_head_before: e9da2f890245d83e4fe23af04739ebf0340ce6a4
  plan_head_after: 14c0cc6f740f0fd3671b25855cb54e54626227d5

tech-stack:
  added: []
  patterns:
    - "Dead-code removal is compile-enforced: deleting a constructor parameter turns every missed call site into a build error, so the inventory is a checklist and the compiler is the gate."
    - "Dump confirmation is proven by lock-and-capture order, not by a version watermark: await appends after the store write under the engine mutex, and runDump captures the list before Encode takes the Records snapshot."
    - "Platform durability lives in the standard library: os.File.Sync already reaches internal/poll's darwin Fsync (F_FULLFSYNC, EINTR retry, ENOTSUP fallback), so no hand-rolled platform file is warranted."
    - "Deletion completeness is asserted three ways: go build/vet over the module, a GOOS=darwin cross-build and vet, and a word-bounded grep gate over internal/, cmd/ and the updated spec row."

key-files:
  created: []
  modified:
    - internal/app/persist/engine.go
    - internal/app/persist/codec.go
    - internal/app/persist/codec_test.go
    - internal/app/repl/core.go
    - internal/app/repl/discard_test.go
    - internal/app/repl/sync_test.go
    - internal/app/internal/simnet/simnet.go
    - internal/udprepl/repl.go
    - internal/persistfile/fsops.go
    - docs/SPEC-DEVIATIONS.md
  deleted:
    - internal/app/persist/engine_internal_test.go
    - internal/persistfile/fsync_darwin.go
    - internal/persistfile/fsync_other.go

key-decisions:
  - "waiter.ver and await's version parameter are deleted with the cover discipline, not retained: after the above-cover branch goes, nothing reads the waiter's version, and keeping it would violate the plan's own no-dead-code prohibition (and unparam's unused-parameter check)."
  - "confirmWaiters keeps the method shape with a blank receiver ((*Engine)); revive's unused-receiver check rejects the named receiver, and the house already uses blank receivers for receiver-free methods (fileFSOps.syncDir)."
  - "The codec table's wantParseErr column is merged into wantDecodeErr: Parse was literally Decode's error channel over the same bytes, so every row's two expectations were identical and no assertion is lost by asserting Decode once."
  - "The plan's five remaining inventory files (udprepl/scenario_test.go, smoke_test.go, grpcapi/bench_test.go, wait_integration_test.go, cmd/sup-kv/main.go) need no edit: they construct through udprepl.New, whose signature is unchanged. The inventory counted files, not direct core-constructor calls."
  - "docs/adr/0010-darwin-development-only.md was checked and not edited: it covers the ~9 KiB darwin send limit and carries no fsync/syncDurable reference, exactly as the plan records."

patterns-established:
  - "A deletion-only plan still proves its safety: every deleted branch's case for absence is written into the surviving code comment (runDump, confirmWaiters, await), so a future reader sees why the removed guard was unreachable."
  - "The durable-step seam now composes the stdlib: fsops names [os.File.Sync] and cites internal/poll's darwin Fsync, so a future platform claim can be checked against the toolchain rather than a local shim."

requirements-completed: [XPORT-13]

coverage:
  - id: C1
    description: "lastLocal, the cover parameter and the unreachable above-cover branch are deleted; confirmWaiters(captured) resolves every captured waiter, with the mutex/capture proof recorded in runDump, confirmWaiters and await."
    requirement: XPORT-13
    tests: [TestEngine_confirmDebounceTiming, TestEngine_burstBoundedConfirmation, TestEngine_dumpFailureFansOut, TestEngine_runExitClosesWaiters]
  - id: C2
    description: "The invariant test that pinned the unreachable branch is deleted with it (the file held no other test), and no above-cover symbol survives."
    requirement: XPORT-13
    tests: ["grep -rn 'lastLocal|ownHandle|aboveCover|AboveCover' internal/ cmd/ (empty)", "go build ./... && go vet ./..."]
  - id: C3
    description: "ownHandle (core field and repl.New parameter) is gone; every direct call site compiles without it, including 03-04's bench fixture and the package-internal egress fixture the inventory missed."
    requirement: XPORT-13
    tests: ["go build ./... && go vet ./...", TestWire_twoNodeHandoff, TestChaos_convergence, TestSmoke_twoNodesUDP]
  - id: C4
    description: "persist.Parse is deleted; the codec tests assert Decode's generation-level error directly, and the stale engine comment no longer names Parse."
    requirement: XPORT-13
    tests: [TestCodec_discardClasses, TestCodec_truncatedGenerationUnreadable, TestEngine_openEmptyStart]
  - id: C5
    description: "fsync_darwin.go and fsync_other.go are deleted; fsops calls File.Sync at both durability steps, and the fsync-order and failure-injection seam tests stay green."
    requirement: XPORT-13
    tests: [TestFsops_saveOperationOrder, TestFsops_saveOperationOrder_rotation_chain, TestFsops_failureAtEachStep, TestFileStore_atomicRotation]
  - id: C6
    description: "Darwin durability is unchanged: os.File.Sync reaches F_FULLFSYNC with EINTR retry and ENOTSUP fallback via internal/poll, and a darwin cross-build and vet pass."
    requirement: XPORT-13
    tests: ["GOOS=darwin GOARCH=arm64 go build ./... && GOOS=darwin GOARCH=arm64 go vet ./...", "internal/poll/fd_fsync_darwin.go (GOROOT source cited in fsops)"]
  - id: C7
    description: "S-PERS-03 reflects the as-built durability path (fsops.go's direct File.Sync, stdlib darwin behavior) and the grep gate over internal/ cmd/ docs/SPEC-DEVIATIONS.md is empty."
    requirement: XPORT-13
    tests: ["grep -rn 'persist.Parse|syncDurable|fsync_darwin|fsync_other' internal/ cmd/ docs/SPEC-DEVIATIONS.md (empty)"]

duration: 649s
completed: 2026-11-22
status: complete
---

# Phase 03 Plan 11: Dead-Code Removal Summary

All four Q-09 findings are deleted rather than retained: the watermark/cover discipline, `persist.Parse`, `Core.ownHandle` and the hand-rolled darwin fsync, with the confirmation proof written into the surviving engine comments and the module green under a darwin cross-build.

## Performance

- **Duration:** 10m 49s (649 s, measured from the persisted plan-start marker)
- **Tasks:** 2/2
- **Commits:** 2 (one per task)
- **Files:** 19 unique (16 modified, 3 deleted)
- **Estimate vs actual:** 60 000 estimated tokens vs 10 437 measured (diff chars/4). The estimate was low-confidence and sized for the 23-file footprint; the realized diff is smaller because five inventory files construct through `udprepl.New` and needed no edit. Task count was exact.

## Accomplishments

**Task 1 — watermark/cover and ownHandle deleted with every call site.**
`Engine.lastLocal` is gone, along with the `cover` capture in `runDump`, the above-cover branch and its invariant assertion in `confirmWaiters`, and `waiter.ver`/`await`'s version parameter (dead once the branch went). `confirmWaiters(captured)` now resolves every captured waiter with nil. The proof of soundness is recorded at all three sites: `await` appends a waiter only after the caller's store write, under the same mutex the dump capture holds; `runDump` captures the list before `Encode(e.store.Records())`; therefore every captured waiter's record is inside the snapshot that dump saved, and a write that loses the capture race re-dirties and is covered by the next dump. `TestConfirmWaiters_aboveCoverInvariant` was deleted with the branch (it was the only test in `engine_internal_test.go`). Separately, `Core.ownHandle` and the `repl.New` parameter are gone across all 16 direct call sites in 9 files, including 03-04's bench fixture and the package-internal egress fixture the inventory missed.

**Task 2 — persist.Parse and the darwin fsync shim deleted; S-PERS-03 updated.**
`persist.Parse` is removed; its four test call sites assert `Decode`'s generation-level error directly (the table's Parse/Decode columns were identical by construction, so they were merged without losing an expectation), and the stale `Open` comment no longer names Parse. `fsops.go` now calls `o.tmp.Sync()` and `d.Sync()` directly; its doc block states that `os.File.Sync` on darwin reaches `internal/poll`'s darwin `Fsync` — F_FULLFSYNC with EINTR retry and an ENOTSUP fallback — so the hand-rolled file was strictly worse and is deleted. `docs/SPEC-DEVIATIONS.md` S-PERS-03 now points at `fsops.go:85-163` and names the standard-library durability path; ADR 0010 was checked and correctly left alone.

## Task Commits

| Task | Commit | Subject |
|------|--------|---------|
| 1 | `01c8f9c` | refactor(03-11): delete lastLocal/cover and ownHandle dead state |
| 2 | `14c0cc6` | refactor(03-11): delete persist.Parse and the hand-rolled darwin fsync |

## Files Created/Modified

**Modified**
- `internal/app/persist/engine.go` — lastLocal/cover/waiter.ver/await-parameter removal; capture proof in runDump/confirmWaiters/await; comment no longer names Parse.
- `internal/app/persist/codec.go` — exported `Parse` deleted.
- `internal/app/persist/codec_test.go` — Parse call sites assert `Decode` directly; columns merged.
- `internal/app/repl/core.go` — ownHandle field/parameter/assignment and "kept for later" doc removed.
- `internal/app/repl/bench_test.go`, `discard_test.go`, `sync_test.go`, `restart_test.go`, `retransmit_test.go`, `egress_internal_test.go` — `repl.New` call sites updated (egress fixture found by the compile gate).
- `internal/app/internal/simnet/simnet.go` — cluster-node core construction updated.
- `internal/udprepl/repl.go`, `repl_test.go`, `syncsize_internal_test.go` — adapter and test constructions updated.
- `internal/persistfile/fsops.go` — `File.Sync` at both durability steps; stdlib darwin behavior documented.
- `docs/SPEC-DEVIATIONS.md` — S-PERS-03 as-built row.

**Deleted**
- `internal/app/persist/engine_internal_test.go` — held only the above-cover invariant test and its helpers.
- `internal/persistfile/fsync_darwin.go`, `internal/persistfile/fsync_other.go` — the false-premise platform shim.

**Untouched plan-listed files (documented deviation):** `internal/udprepl/scenario_test.go`, `internal/udprepl/smoke_test.go`, `internal/grpcapi/bench_test.go`, `internal/grpcapi/wait_integration_test.go`, `cmd/sup-kv/main.go` — they construct through `udprepl.New`, whose signature did not change.

## Decisions Made

- **Delete `waiter.ver` and `await`'s parameter with the branch.** After the cover comparison went, nothing read the waiter's version; retaining it would be exactly the dead state the plan prohibits. `Set`/`Delete`/`SetIfAbsent` now call `e.await()`.
- **Keep `confirmWaiters` as a blank-receiver method** rather than inlining it: the plan names the helper, the call-site shape stays stable, and revive's unused-receiver check is satisfied the same way `fileFSOps.syncDir` already does.
- **Merge the codec table's two error columns.** `Parse` was `Decode`'s error channel over the same bytes; asserting `Decode` once preserves every row's expectation without a redundant second decode.
- **Do not edit the five inventory files that only call `udprepl.New`.** The adapter's signature is unchanged; forcing edits there would be churn, and the module-wide compile gate is the real missed-site check.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fifth call site missed by the inventory: `internal/app/repl/egress_internal_test.go`**
- **Found during:** Task 1's first `go vet ./...`
- **Issue:** The plan's inventory listed 13 non-evidence files by name, but the package-internal egress fixture calls the unqualified `New(..., "self", ...)` — invisible to a `repl.New(` grep. `go vet` failed with "too many arguments in call to New".
- **Fix:** Dropped the handle argument; the fixture now uses the four-argument form.
- **Files modified:** `internal/app/repl/egress_internal_test.go`
- **Verification:** `go build ./... && go vet ./...` green; full suite green.
- **Committed in:** `01c8f9c` (Task 1 commit)

**2. [Rule 2 - Missing critical functionality] `waiter.ver` and `await`'s parameter were themselves dead**
- **Found during:** Task 1 implementation
- **Issue:** Deleting the above-cover branch leaves the waiter's version field and `await`'s parameter with no reader. Keeping them would contradict the plan's prohibition ("no dead code may be retained") and trip the strict lint's unused/unparam checks.
- **Fix:** Removed both; `await()` takes no parameter and the write paths call `e.await()`.
- **Files modified:** `internal/app/persist/engine.go`
- **Verification:** `go build`, `go vet`, `go test -race ./internal/app/persist/` and the strict lint green.
- **Committed in:** `01c8f9c` (Task 1 commit)

**3. [Rule 3 - Blocking] `engine_internal_test.go` became a deletion, not an adaptation**
- **Found during:** Task 1 implementation
- **Issue:** The plan listed the file under `files_modified` and said to "keep/adapt the remaining waiter tests", but the file contained only `TestConfirmWaiters_aboveCoverInvariant` and its dedicated fakes — nothing remained to keep.
- **Fix:** Deleted the file with `git rm`; the surviving waiter behavior is covered by the engine suite (`TestEngine_dumpFailureFansOut`, `TestEngine_runExitClosesWaiters`, `TestEngine_burstBoundedConfirmation`, and the confirmation-timing tests).
- **Files deleted:** `internal/app/persist/engine_internal_test.go`
- **Verification:** `go test -race ./internal/app/persist/` green; grep gate clean.
- **Committed in:** `01c8f9c` (Task 1 commit)

**4. [Rule 3 - Blocking] Strict-lint shape on the deleted-symbol sites**
- **Found during:** the plan-wide `go tool golangci-lint run ./...`
- **Issue:** revive flagged `confirmWaiters`'s now-unused named receiver, and godoclint required doc links for `os.File.Sync` and `syscall.Fsync` in the new fsops comments.
- **Fix:** Blank receiver `(*Engine)`; doc links `[os.File.Sync]` / `[syscall.Fsync]`. No behavior change.
- **Files modified:** `internal/app/persist/engine.go`, `internal/persistfile/fsops.go`
- **Verification:** `go tool golangci-lint run ./...` green with no new `//nolint`.
- **Committed in:** `14c0cc6` (Task 2 commit)

**Footprint composition (accepted scope note).** The plan accepted 21 modified + 2 deleted files across the four independent Q-09 findings — `lastLocal`/`cover` (engine plus its invariant test), `persist.Parse` (codec, tests, one engine comment), `ownHandle` and its 13-file inventory plus 03-04's bench fixture, and the darwin fsync pair. The realized diff is 16 modified + 3 deleted = 19 unique files: five inventory files construct through `udprepl.New` and needed no edit, `engine_internal_test.go` became a deletion rather than a modification, and `egress_internal_test.go` joined from the compile gate. The four-finding composition and the accepted scope spread are unchanged.

---

**Total deviations:** 4 auto-fixed (3 blocking, 1 complete-deletion correctness)
**Impact on plan:** No scope change. Every must-have truth and acceptance criterion is met; the extra fixture and the deletion of `waiter.ver` are the compile/lint consequences of deleting the two symbols completely.

## Issues Encountered

- **Two lint rounds, no behavior change:** the first strict-lint run flagged the unused receiver and two doc-link forms; both fixes are comment/receiver shape only. The second run flagged `syscall.Fsync`; after linking it the gate is clean.
- **No auth or setup gates:** none encountered; `user_setup` is empty.

## Threat Flags

None — this plan removes surface (an unused constructor parameter, an unused exported oracle, a platform shim, and an unreachable branch). No new endpoints, auth paths, file-access patterns or trust boundaries; T-3-37's durability requirement is met by the standard library's darwin `File.Sync`.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- XPORT-13 is complete: all four Q-09 findings are gone, proven by the module-wide build/vet, the darwin cross-build/vet, the grep gates and the full `-race` suite.
- Phase 5's P-09 is the remaining fsync-adjacent work (orphan temp collection and parent-directory sync policy); the file-flush path itself is now the standard library's.
- 03-12 can proceed against the same tree; no API it consumes changed in this plan.

## Self-Check: PASSED

- All key files exist on disk (16 modified present, 3 deleted absent).
- Measured commits from the persisted plan ledger: `git rev-list --count e9da2f8..14c0cc6` = 2; `plan_head_before` = `e9da2f890245d83e4fe23af04739ebf0340ce6a4`, `plan_head_after` = `14c0cc6f740f0fd3671b25855cb54e54626227d5`.
- `gsd_run` is unavailable in this executor runtime (exit 1, no output); the close-out uses the persisted plan ledger, `git rev-list --count` and direct STATE/ROADMAP/REQUIREMENTS edits as the documented fallback.
- Acceptance criteria re-run at close-out: `go build ./... && go vet ./...` green; `go test -race -count=1 ./...` green across every package; `GOOS=darwin GOARCH=arm64 go build ./... && … go vet ./...` green; `grep -rn 'lastLocal|ownHandle|func Parse|syncDurable' internal/ cmd/` empty; `grep -rn 'persist.Parse|syncDurable|fsync_darwin|fsync_other' internal/ cmd/ docs/SPEC-DEVIATIONS.md` empty; `go tool golangci-lint run ./...` green with no new suppressions.
- S-PERS-03 points at `internal/persistfile/fsops.go:85-163`; ADR 0010 contains no fsync reference and was not edited.

---
*Phase: 03-transport*
*Completed: 2026-11-22*
