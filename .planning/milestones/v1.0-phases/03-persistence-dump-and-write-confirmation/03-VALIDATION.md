---
phase: "3"
slug: "persistence-dump-and-write-confirmation"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-03"
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go stdlib `testing` + `testing/synctest`), `-race` mandatory |
| **Config file** | none — existing Makefile targets + CI (`.github/workflows`) |
| **Quick run command** | `go test -race ./internal/app/persist/... ./internal/persistfile/...` |
| **Full suite command** | `make lint && make test` |
| **Estimated runtime** | ~4 s quick (uncached), ~25 s full gate (repo-wide, `-race -cover`) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./internal/app/persist/... ./internal/persistfile/...`
- **After every plan wave:** Run `make lint && make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~5 s (phase packages; synctest keeps timing tests instant and exact)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 3-01-01 | 01 | 1 | PERS-01, PERS-02, PERS-04, VERS-04 | T-3-01 / T-3-02 | Confirmed write on disk in greppable JSONL (hex version, base64 data, no number array); restart reloads tombstones and strictly dominates | integration | `go test -race ./internal/app/persist/ -run '^TestEngine_endToEndRestart$\|^TestEngine_openEmptyStart$' -v` | ✅ | ✅ green |
| 3-01-02 | 01 | 1 | PERS-03, PERS-04 | T-3-03 | Exact virtual-time confirmation (50 ms debounce), 1000-write burst bounded at oldest+1 s in ≤3 dumps, ErrDumpFailed fan-out at one instant, ErrClosed exit sweep, full write surface | unit (synctest) | `go test -race ./internal/app/persist/ -run '^TestEngine_confirmDebounceTiming$\|^TestEngine_burstBoundedConfirmation$\|^TestEngine_dumpFailureFansOut$\|^TestEngine_runExitClosesWaiters$\|^TestEngine_writeSurface$' -v` | ✅ | ✅ green |
| 3-01-03 | 01 | 1 | PERS-01, PERS-02, VERS-04 | T-3-01 / T-3-02 | Codec laws over 500 seeded steps (round-trip, byte-stability, discard-class separation); crash-residue fallback chain; torn-record restart; full-watermark law | unit + integration | `go test -race ./internal/app/persist/ -run 'TestCodec\|TestWatermark\|TestEngine_restartDiscardsTornRecord' -v && go test -race ./internal/persistfile/ -v` | ✅ | ✅ green |
| 3-02-01 | 02 | 2 | PERS-05 | T-3-06 | Snapshot is a byte-exact copy of the confirmed dump; restore replays only snapshot keys with fresh dominating versions, absent keys untouched, both tombstone directions | integration | `go test -race ./internal/app/persist/ -run '^TestEngine_snapshotCreateLoadRestore$' -v` | ✅ | ✅ green |
| 3-02-02 | 02 | 2 | PERS-05 | T-3-04 / T-3-06 | Name-validation security table (traversal/separator/dot/length rejected before any filesystem touch, nothing escapes `.snapshots`); atomic overwrite without `.prev`; error classes; flush-forces-dump | unit + integration | `go test -race ./internal/persistfile/ -run 'TestFileStore_nameValidation\|TestFileStore_snapshotOverwriteAtomic\|TestFileStore_snapshotErrorClasses' -v && go test -race ./internal/app/persist/ -run 'TestEngine_snapshotFlushForcesDump\|TestEngine_restoreMissingSnapshot' -v` | ✅ | ✅ green |
| 3-V-01 | audit | — | VERS-04 | — | Equal-l half of the full watermark: at max l the greatest counter wins (the (100,5)+(100,7)+(99,4095) → (100,7) acceptance tuple) under every map iteration order | unit | `go test -race ./internal/app/persist/ -run '^TestWatermark_equalCounterDomain$' -v` | ✅ | ✅ green |
| 3-V-02 | audit | — | PERS-02 | T-3-01 | Torn-tail generation fault: a dump truncated mid-record is unreadable as a generation — Parse rejects (chain advances to `.prev`), Decode errors instead of returning the pre-tear records | unit | `go test -race ./internal/app/persist/ -run '^TestCodec_truncatedGenerationUnreadable$' -v` | ✅ | ✅ green |
| 3-V-03 | audit | — | PERS-04 | T-3-03 | Post-close surface: Set/SetIfAbsent/Delete/Snapshot/Restore return ErrClosed immediately after Run exits (no writer registers after the sweep and blocks), Get keeps serving, nothing copied | integration | `go test -race ./internal/app/persist/ -run '^TestEngine_closeSurface$' -v` | ✅ | ✅ green |
| 3-V-04 | audit | — | PERS-05 | T-3-06 | D-10 safety half: a failed forced flush aborts Snapshot with the flush error and copies nothing — unconfirmed state never becomes a recovery point | unit (synctest) | `go test -race ./internal/app/persist/ -run '^TestEngine_snapshotFlushErrorAborts$' -v` | ✅ | ✅ green |
| 3-V-05 | audit | — | PERS-05 | T-3-06 / T-3-01 | Corrupt-snapshot class: a generation-faulty snapshot surfaces its decode error from Restore — never a silent empty success — leaving the map unchanged | integration | `go test -race ./internal/app/persist/ -run '^TestEngine_restoreCorruptSnapshot$' -v` | ✅ | ✅ green |
| 3-V-06 | audit | — | PERS-02, PERS-05 | T-3-01 | Config-shrink resilience: Open and Restore skip records the shrunk value limit rejects instead of aborting; Set surfaces ErrValueTooLarge unchanged with no waiter | integration | `go test -race ./internal/app/persist/ -run '^TestEngine_configShrunkValueLimits$' -v` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements — Makefile `lint`/`test` targets, CI, `testutil` seeded generators, `-race`. New test files landed with their packages (plan 03-01/03-02); the validation audit added six tests inside the existing test files. No framework work needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

Residual uncovered lines (91.6% → 97.4% `internal/app/persist`, 87.7% `internal/persistfile` after the audit), none requirement-bearing:
- `codec.go` Encode error returns — `json.Encoder` into a `bytes.Buffer` cannot fail; handling is errchkjson-mandated.
- `engine.go:238` sweep drain and `engine.go:288` above-cover re-queue — defensive paths the implementers document as unreachable/near-unreachable (select arbitration makes a deterministic pin infeasible; the user-visible post-close contract is pinned by 3-V-03).
- `engine.go:398` `default:` arm of the non-blocking change signal; `engine.go:459` replay error return with no reachable trigger under the current store API.
- `internal/persistfile` fsync/rename/dir-sync failure plumbing — requires fault injection or a crash harness; durability under real crashes is Phase 6 (TEST-04) scope per the phase research.
- macOS F_FULLFSYNC durability — Phase 6 verification item (phase boundary).

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none missing)
- [x] No watch-mode flags
- [x] Feedback latency < 5 s (phase packages)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-11-03 (Nyquist audit: full gate `make lint` + `make test` green under `-race`; six audit tests added and mutation-verified — a neutered equal-counter watermark branch and a torn-tail-tolerant decode both fail the new tests while the pre-existing suite passes them, proving the gaps were real)

## Validation Audit

| Metric | Value |
|--------|-------|
| Gaps found | 6 (VERS-04 equal-counter domain; PERS-02 torn-tail fault; PERS-04 post-close surface; PERS-05 flush-abort / corrupt-snapshot / config-shrink classes) |
| Resolved | 6 (tests added to `internal/app/persist/codec_test.go` and `engine_test.go`; all green; 2 mutation-verified) |
| Escalated | 0 |
| Coverage | `internal/app/persist` 93.0% → 97.4%, `internal/persistfile` 87.7% (residuals classified above) |
