---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 09
subsystem: persistence
tags: [darwin, fsync, F_FULLFSYNC, durability, persistfile, phase-gate, benchmarks]

# Dependency graph
requires:
  - phase: 06-hardening-failure-scenarios-observability-benchmarks
    provides: "waves 1-4: D-15 fixes, metrics endpoint, scenario suites, exec harness, D-13 benchmarks — everything the full gate runs over"
provides:
  - "Platform-uniform durability flush: darwin F_FULLFSYNC fcntl through the shared syncDurable helper at both persistfile sync sites"
  - "Full-repo phase-gate evidence (lint, race suite, bench smoke, boundary checks, D-15 closure) recorded for verify-work"
  - "TEST-05 benchmark evidence snapshot from make bench"
affects: [07-release-milestone, milestone-close, verify-work]

actuals:
  tokens: 3800
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns: ["build-tagged platform sync adapter (fsync_darwin.go / fsync_other.go) — first platform-split file in the repo, syscall legal in adapter packages per the udprepl precedent"]

key-files:
  created:
    - internal/persistfile/fsync_darwin.go
    - internal/persistfile/fsync_other.go
  modified:
    - internal/persistfile/dumpfile.go

key-decisions:
  - "F_FULLFSYNC REPLACES File.Sync on darwin rather than wrapping it — Apple's fcntl(2) defines it as fsync-plus-drive-flush (a strict superset), so a preceding Sync would flush the same data twice; choice documented in the helper's comment (plan 06-09 Task 1)"
  - "No darwin-only test file: no CI test can assert real power-loss behavior and an fcntl-succeeds-on-a-temp-file check proves nothing durable; both-arch cross-compile plus the unchanged linux suite is the honest evidence (06-RESEARCH finding, plan 06-09 Task 1)"
  - "Task 2 is evidence-only — zero file changes, so no per-task commit exists; all gate output lives in this SUMMARY (plan 06-09 Task 2)"
  - "D-16 fingerprint regeneration surfaced as pending-main-session work (checkpoint:human-action, gate=blocking): three /gsd-execute-phase runs (phases 1, 4, 5) before /gsd:verify-work and milestone close; no VERIFICATION.md hand-edited; WINDOWS.md carries the unrun-verify entry (plan 06-09 Task 3)"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "Platform-uniform durability flush — build-tagged darwin F_FULLFSYNC helper wired into both persistfile sync sites (file data + directory entry); byte-identical behavior on every other platform"
    verification:
      - kind: automated
        ref: "GOOS=darwin GOARCH=arm64 go build ./internal/persistfile/ && GOOS=darwin GOARCH=amd64 go build ./internal/persistfile/ — both green (assumption A3 arch coverage)"
        status: pass
      - kind: unit
        ref: "go test -race ./internal/persistfile/ — green unmodified (fsync_other is a pass-through)"
        status: pass
      - kind: automated
        ref: "go tool golangci-lint run ./internal/persistfile/... — exit 0, zero new nolints"
        status: pass
    human_judgment: false
  - id: D2
    description: "Full phase gate green with captured evidence: make lint, make test (race+cover), make bench three-name smoke, dependency/core-silence boundary checks, all 12 D-15 disposition rows fixed — the TEST-04/TEST-05 evidence snapshot for verify-work"
    requirement: TEST-04
    verification:
      - kind: automated
        ref: "make lint && make test && make bench >/tmp/phase6-bench.log 2>&1 && grep -q '^BenchmarkDump10kKeys' && grep -q '^BenchmarkReplicationBurst' && grep -q '^BenchmarkConfirmationLatency' && go mod tidy && git diff --exit-code go.mod go.sum — full chain exit 0"
        status: pass
      - kind: automated
        ref: "boundary greps: zero slog imports under internal/app; zero blank/underscore net/http/pprof or expvar imports repo-wide; all 12 D-15 rows show fixed in 0N-REVIEW-DISPOSITION.md"
        status: pass
    human_judgment: false
  - id: D3
    description: "D-16 — regenerated VERIFICATION fingerprints for phases 1, 4 and 5 (staled by the 2026-11-14 buf zero-waiver refactor 7b55ad1)"
    verification: []
    human_judgment: true
    rationale: "Requires interactive main-session slash-command runs (/gsd-execute-phase 1, 4, 5) — not subagent-executable by design (checkpoint:human-action, gate=blocking). Surfaced here as pending-main-session work and recorded in WINDOWS.md (kind: unrun-verify); must complete before /gsd:verify-work for Phase 6 and milestone close."

# Metrics
duration: 12min
completed: 2026-11-16
status: halted
---

# Phase 06 Plan 09: darwin F_FULLFSYNC + Full Phase Gate Summary

**Build-tagged darwin F_FULLFSYNC fcntl makes the persistfile durability flush platform-uniform (the open Phase-3 STATE blocker closes with code), the full-repo gate is green with the TEST-05 evidence snapshot captured, and the D-16 fingerprint regeneration is surfaced as pending main-session work.**

## Performance

- **Duration:** ~12min
- **Started:** 2026-11-16T01:27:08Z
- **Completed:** 2026-11-16T01:38:00Z (executor portion; D-16 pending)
- **Tasks:** 2 of 3 executed (Task 3 is a checkpoint:human-action pending in the main session)
- **Files modified:** 3

## Accomplishments

- Task 1 (tracer): `internal/persistfile/fsync_darwin.go` issues `fcntl(fd, F_FULLFSYNC, 0)` via `syscall.Syscall`, REPLACING `File.Sync` on darwin (fcntl(2): "does the same thing as fsync(2) then asks the drive to flush to the permanent storage device" — a strict superset; Go's darwin Fsync binds libc_fsync only, verified in the Go 1.27 toolchain source). `fsync_other.go` (!darwin) delegates to `File.Sync` — byte-identical behavior on every other platform. `dumpfile.go` routes both sync sites (`writeSynced`, `syncDir`) through the shared `syncDurable`; the snapshot path inherits via `saveAtomically`. Zero new nolints; kill -9 crash tests unaffected on every platform (only power loss distinguishes the two flushes, and no CI test simulates power loss).
- Task 2 (auto): full phase gate green with evidence captured below — lint, race suite, bench smoke, dependency and core-silence boundaries, D-15 closure. This task produces evidence, not code: no commit exists for it by design.
- Tracer feedback gate: auto mode active; the tracer's `<verify>` re-ran end-to-end green after Task 1's commit before expansion continued.
- Task 3 (checkpoint:human-action, gate=blocking): D-16 surfaced as pending main-session work — see "Pending Main-Session Work". No VERIFICATION.md was hand-edited.

## Task Commits

Each task was committed atomically:

1. **Task 1: darwin F_FULLFSYNC — build-tagged sync helper wired into the dump path** - `4f75a3b` (feat)
2. **Task 2: Full phase gate** - no commit (evidence-only task, zero file changes; output recorded here)
3. **Task 3: D-16 fingerprint regeneration** - no commit (human-action checkpoint; not subagent work)

**Plan metadata:** see the docs(06-09) commit following this summary.

## Gate Evidence (Task 2)

- **make lint** — exit 0 (config verify + strict run over ./...), 7.6s wall. Zero findings.
- **make test** (`go test -race -cover ./...`) — exit 0, 7.7s wall (same order as the ~7s pre-phase baseline plus the phase's scenario/exec additions; no surprising growth). Coverage: clock/storage/version 100%, persist 97.7%, repl 97.8%, grpcapi 91.1%, udprepl 86.2%, metrics 68.1%, persistfile 88.4%, simnet 74.0%.
- **make bench** — exit 0; all three D-13 benchmarks ran (two full runs, both green; numbers informational, no thresholds per D-13). First-run snapshot (Intel i9-9980HK, linux/amd64):
  - `BenchmarkDump10kKeys-16      10   123240972 ns/op   5147550 bytes/op   5147550 dump-bytes`
  - `BenchmarkReplicationBurst-16    1024   1020560 ns/op   34.00 max-unacked   979.9 writes/s`
  - `BenchmarkConfirmationLatency-16    124   10211499 ns/op   10204284 latency-ns/op   16498423 max-latency-ns`
- **Dependency boundary** — `go mod tidy` leaves go.mod/go.sum byte-identical (`git diff --exit-code` clean): zero new dependencies, the D-04 rejection held.
- **Core-silence discipline** — zero `log/slog` imports anywhere under `internal/app` (production and test files); zero blank/underscore imports of `net/http/pprof` or `expvar` repo-wide.
- **D-15 closure** — all twelve folded findings show `fixed` in their disposition files: 01-WR-01/IN-01 (P1-WR-01 timeout-minutes, P1-IN-01 serve signal), 02-IN-01/02/03, 03-WR-01/IN-01/IN-02, 04-WR-01/WR-02/IN-01/IN-02.

## Files Created/Modified

- `internal/persistfile/fsync_darwin.go` - NEW: build-tagged darwin `syncDurable` — the fcntl F_FULLFSYNC flush with the research finding documented (libc_fsync binding, fcntl superset semantics, kill -9 neutrality)
- `internal/persistfile/fsync_other.go` - NEW: !darwin `syncDurable` — plain `File.Sync` pass-through, behavior byte-identical to the pre-helper call sites
- `internal/persistfile/dumpfile.go` - `writeSynced` and `syncDir` route through `syncDurable` (doc comments updated; no logic changes beyond the routing)

## Decisions Made

Recorded in frontmatter key-decisions (also pushed to STATE.md): F_FULLFSYNC replaces rather than wraps File.Sync on darwin; no darwin-only test file (nothing real to assert in CI); Task 2 evidence-only with no commit; D-16 surfaced as pending main-session work.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. No auth gates (pure in-repo work).

## User Setup Required

None for the executor's portion. One main-session action is pending — see below.

## Pending Main-Session Work (Task 3 — D-16, gate=blocking human-action)

The 2026-11-14 buf zero-waiver refactor (7b55ad1) staled the VERIFICATION.md fingerprints of phases 1, 4 and 5. Regeneration is interactive workflow work (slash-command driven), not subagent work. Before `/gsd:verify-work` for Phase 6 and before milestone close, run in the main session, in order:

1. `/gsd-execute-phase 1`
2. `/gsd-execute-phase 4`
3. `/gsd-execute-phase 5`

Each run re-executes that phase's verification against the current tree and regenerates its VERIFICATION.md fingerprints. Done-state: all three phases show fresh fingerprints and the STATE.md 2026-11-14 staleness entry is updated to record the regeneration. This summary intentionally does NOT claim that evidence (must_haves prohibition: fingerprints are evidence, not a checkbox) — the WINDOWS.md ledger carries the open `unrun-verify` entry until the runs complete.

## Next Phase Readiness

- Phase 6's code waves are complete: scenario suites (simulator + real-socket + exec), observability (metrics + structured logs), benchmarks, D-15 hardening backlog, and now platform-uniform durability. The full-repo gate is green for `/gsd:verify-work` — after the three D-16 fingerprint re-runs above.
- The Phase-3 STATE blocker "macOS F_FULLFSYNC durability policy" closes as resolved-with-code (`4f75a3b`); process-kill crash tests remain valid on every platform either way.
- Phase 7 (docs, examples, release) inherits a green gate, byte-identical module files, and the bench door (`make bench`) for any release-notes performance snapshot.

## Self-Check: PASSED

- Key files exist on disk: fsync_darwin.go, fsync_other.go, dumpfile.go — FOUND
- Task commit `4f75a3b` is an ancestor of HEAD — FOUND
- `check evaluation-scope --plan 06-09 --commits-only` returned 1 commit — FOUND
- Task 1 acceptance criteria re-run: build tags, fcntl call, both-site routing, both-arch cross-compile, race suite, lint, zero new nolints — ALL PASS
- Task 2 acceptance criteria re-run: lint/test exit 0, three bench names printed, tidy byte-identical, slog/pprof greps clean, 12/12 D-15 rows fixed — ALL PASS
- Task 3 verification: intentionally NOT run (human-action; see Pending Main-Session Work) — recorded in coverage D3 + WINDOWS.md

---
*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
