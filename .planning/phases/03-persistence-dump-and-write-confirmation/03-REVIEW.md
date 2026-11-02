---
phase: 03-persistence-dump-and-write-confirmation
reviewed: 2026-11-02T23:52:02Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - internal/app/persist/codec.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/doc.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/persist/errors.go
  - internal/persistfile/dumpfile.go
  - internal/persistfile/dumpfile_test.go
  - internal/persistfile/snapshot.go
  - internal/persistfile/snapshot_test.go
  - internal/port/doc.go
  - internal/port/dump.go
  - docs/IDEA.md
findings:
  critical: 1
  warning: 1
  info: 4
  total: 6
  blocker: 1
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-11-02T23:52:02Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

The phase-03 persistence stack was reviewed at standard depth against the on-disk state (which includes an uncommitted `codec_test.go` addition of two tests; both were reviewed and pass). Cross-referenced dependencies (`internal/app/storage`, `internal/app/version`, `internal/app/clock`, `internal/app/internal/testutil`) were read to verify contracts at the seams. `go build`, `go vet` and `go test -race` over `internal/app/persist`, `internal/persistfile` and `internal/port` all pass; the five declared `//nolint` suppressions each name their linter with a written reason (AGENTS.md deviation 3 compliant); `go.mod` keeps zero direct runtime requires; the `dump_max_wait` 1s entry is present in `docs/IDEA.md:148` between `dump_debounce` and `drift_warn` as decided (D-04).

Overall quality is high: the codec's failure-class separation, the cover discipline (lastLocal updated under the engine mutex after the store write, so confirmation means on-disk containment), the timer pair's Go 1.23+ no-drain usage, the atomic-write sequence, and the snapshot-name security boundary (charset + dot-specials + length, validated before any filesystem touch) were each traced and hold. Two concurrency defects were found: one provable permanent-goroutine-hang race in `flush()` against `Run`'s exit sweep (BLOCKER), and one spurious `ErrNoDump` race between `CreateSnapshot` and a concurrent `Save`'s rotate window (WARNING). Four lower-severity items follow.

Note: `internal/app/persist/codec_test.go` carries uncommitted modifications in the working tree (two added tests, `TestWatermark_equalCounterDomain` and `TestCodec_truncatedGenerationUnreadable`); both are correct and green, but they must be committed with the phase.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: `flush()` can block forever when Run exits between the closed-check and the flushCh send — permanent goroutine hang on shutdown

**File:** `internal/app/persist/engine.go:492-502` (with `sweepClosed` at `internal/app/persist/engine.go:223-243`)
**Issue:** `flush()` reads `e.closed` under the mutex, unlocks, sends its reply channel into `flushCh`, then blocks on `<-reply`:

```go
e.mu.Lock()
closed := e.closed
e.mu.Unlock()
if closed {
    return ErrClosed
}
e.flushCh <- reply
return <-reply
```

`Run`'s exit path (`sweepClosed`) sets `closed`, resolves stranded waiters, drains `flushCh` once with a non-blocking loop (lines 235-242), and returns — after which nothing ever reads `flushCh` again. The drain only answers requests that landed **before** it ran. Interleaving:

1. A `Snapshot`/`Restore` goroutine calls `flush()` and observes `closed == false` (mutex released).
2. The context is cancelled; `Run` executes `sweepClosed` to completion — the drain finds `flushCh` empty — and returns.
3. `flush()` now performs `e.flushCh <- reply`. The channel is buffered (cap 1), so the send succeeds into the buffer with no receiver alive.
4. `<-reply` blocks forever. The caller hangs permanently — no error, no timeout, no recovery.

This is supported usage, not caller error: `doc.go` states both operations require a running Run goroutine, and context cancellation (SIGTERM in Phase 5) may arrive at any instant concurrent with an operator snapshot/restore RPC; the hung goroutine is an API handler leak on every shutdown that races it. `await()` does not have this bug because its closed-check and waiter registration are a single atomic mutex hold; `flush()` cannot use that pattern (the send must happen outside `mu`), and the check-then-send replacement leaves the window. Related, lower-severity edge: `flush()` likewise blocks forever if Run was never launched — documented as a requirement, but the `closed` early-return creates the impression `flush` always returns.
**Fix:** Give Run an exit signal closed after the sweep, and race both the send and the receive against it, with a final non-blocking re-check that distinguishes "answered by the sweep" from "stranded after the sweep":

```go
// in Engine: runDone chan struct{} (created in New)
// at the very end of Run, after sweepClosed(): close(e.runDone)

func (e *Engine) flush() error {
	reply := make(chan error, 1)
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return ErrClosed
	}
	select {
	case e.flushCh <- reply:
	case <-e.runDone:
		return ErrClosed
	}
	select {
	case err := <-reply:
		return err
	case <-e.runDone:
		select {
		case err := <-reply: // the exit sweep answered us after all
			return err
		default:
			return ErrClosed
		}
	}
}
```

The inner non-blocking re-check is required because `runDone` closes only after the drain completes (program order in `Run`), so a reply present there is exactly one the sweep answered, and an absent one is exactly the stranded case.

## Warnings

### WR-01: `CreateSnapshot` can spuriously fail with `ErrNoDump` during a concurrent `Save`'s rotate window; the engine's concurrency-safety claim is incorrect

**File:** `internal/persistfile/dumpfile.go:133-141` (rotate window), `internal/persistfile/snapshot.go:45-49` and `snapshot.go:65-71` (`readMainDump` mapping `fs.ErrNotExist` to `ErrNoDump`); incorrect claim at `internal/app/persist/engine.go:423-425`
**Issue:** `placeFile` rotates the old main generation first (`os.Rename(path, prevPath)`) and only then renames the temp onto the main path (`os.Rename(tmpName, path)`). Between the two renames the main path does not exist. `Engine.Snapshot` runs `flush()` and then calls `CreateSnapshot` **on the caller's goroutine**, while `Run` remains free to start the next dump at any moment (a concurrent `Set` re-dirties the engine; the debounce fires ~50 ms later). If `readMainDump`'s `os.ReadFile(f.path)` lands inside that rotate window it sees `fs.ErrNotExist` and `Snapshot` returns `ErrNoDump` even though fully confirmed data exists on disk. The engine doc comment claims "Safe against a dump running concurrently with the copy — the adapter's atomic rename means it always reads one complete generation" — atomic rename guarantees no *torn* read, not no *absent* read; the rotate-first order (correct for the crash chain, D-02) creates precisely an absence window. Result: an operator-visible spurious failure of the snapshot operation exactly during write activity — a flaky RPC in Phase 5.
**Fix:** Serialize `CreateSnapshot` against `Save` inside `FileStore` — a `sync.Mutex` held across `Save`'s `placeFile` and across `CreateSnapshot`'s `readMainDump`+copy (Load/ReadSnapshot may stay lock-free since the adapter's rename keeps every observable generation complete). Alternatively, boundedly re-read on `ENOENT` (e.g. a short retry) before reporting `ErrNoDump`. Also correct the `Snapshot` doc comment at `engine.go:423-425` to state the real guarantee (no torn read; absence window handled by serialization).

## Info

### IN-01: `maxRecordBytes` literal duplicates the store's value limit — drift risk

**File:** `internal/app/persist/codec.go:25`
**Issue:** `maxRecordBytes = 61440` independently duplicates `defaultValueMaxBytes = 61440` in `internal/app/storage/store.go:15`. The codec's decode bound and the store's write bound encode the same design envelope (docs/IDEA.md `value_max_bytes`); a future change to one silently diverges the other.
**Fix:** Cross-reference in both constants' comments, or hoist a single shared constant (e.g. in `internal/port` or a shared app-level constants package).

### IN-02: Engine is silently single-shot — `closed` is never reset after one Run exit

**File:** `internal/app/persist/engine.go:225` (with `New` at 81-89)
**Issue:** `sweepClosed` sets `e.closed = true` permanently; nothing resets it. Launching `Run` again on the same engine (e.g. a Phase 5 supervisor restarting the dumper) resolves every waiter and flush with `ErrClosed` while the store keeps serving reads — a subtle half-alive state. `doc.go` documents Run's lifecycle but not that the engine is one-shot.
**Fix:** Document the single-shot contract on `Run`/`New`, or defensively panic/reset state if `Run` is entered on an already-closed engine.

### IN-03: Out-of-range `checksum` field collapses the whole generation instead of discarding one record

**File:** `internal/app/persist/codec.go:53` and `108-116`
**Issue:** `recordLine.Checksum` is `uint32`. A record line whose `checksum` JSON number is out of range (hand-edited `-1` or `4294967296`) fails inside `json.Decoder`, so `Decode` returns a generation-level error and the entire generation is discarded (fallback to `.prev` / empty start) — whereas an in-range wrong checksum discards only that record (PERS-02's class separation). Disk bit flips in the ASCII digits stay in range, so the exposure is limited to hand edits and corrupt writers, but the class asymmetry is real.
**Fix:** Decode `checksum` through `json.Number` (or `float64`) with an explicit range check in `decodeRecord`, returning the per-record discard class for out-of-range values.

### IN-04: `CreateSnapshot` creates the `.snapshots` directory even when it then fails with `ErrNoDump`

**File:** `internal/persistfile/snapshot.go:41-48`
**Issue:** `MkdirAll(snapshotDir, dirMode)` runs before `readMainDump`; a snapshot attempt with no main dump on disk leaves an empty `.snapshots` directory behind as a side effect of a failed call.
**Fix:** Move `readMainDump` before `MkdirAll` (the copy still creates the directory on the success path via `saveAtomically`).

---

_Reviewed: 2026-11-02T23:52:02Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
