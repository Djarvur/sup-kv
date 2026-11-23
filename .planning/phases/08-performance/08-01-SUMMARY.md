---
phase: 08-performance
plan: 01
subsystem: test-harness
tags: [performance, thresholds, hardware-block, makefile, stand, tdd]
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: the real-binary stand harness, redRunBurst and the frozen 1000 writes/s red row
  - phase: 07-observability-and-configuration
    provides: the metrics endpoint and the stabilized stand suite
provides:
  - one compiled perf threshold table (internal/testutil/perf.go)
  - the perf/env and race gates plus the build-tag race predicate
  - the hardware block and the dump-directory filesystem/device guard
  - the operator-facing burst perf row with the frozen-contract drift guard
  - the `make perf` target
affects: [08-02, 08-03, 08-04, 08-05, 08-06, 08-10, 08-11]
actuals:
  tokens: 8734
  tasks: 3
  commits: 5
plan_head_before: 38fc1f87eec01ebe5164cffc9095ee1135c4625a
plan_head_after: 67535a2f7dd0dec6c210a477db9e9ef70503e1d1
tech-stack:
  added: []
  patterns:
    - one compiled const threshold table (no env parsing, no threshold file)
    - named-skip perf gate naming `make perf`
    - build-tag race predicate answered by which file the compiler selects
    - hardware block assembled from best-effort /proc and build-info reads
    - per-test dump subdirectory under an operator base with the media facts logged
key-files:
  created:
    - internal/testutil/perf.go
    - internal/testutil/perf_test.go
    - internal/testutil/race_on.go
    - internal/testutil/race_off.go
    - internal/testutil/perf_linux.go
    - internal/testutil/perf_other.go
    - internal/testutil/perf_linux_test.go
    - internal/stand/perf_test.go
  modified:
    - Makefile
key-decisions:
  - "Phase 8's thresholds are one compiled const block in internal/testutil/perf.go; no consumer retypes a literal and no threshold is parsed from the environment."
  - "RED evidence used a compiling skeleton, not the plan's missing-file failure: a build failure is INVALID_RED and cannot authorize GREEN."
  - "The perf burst row reuses the frozen red generator (redRunBurst) and constants read-only; TestPerfBurstBoundsMatchRedContract is the always-on drift guard."
  - "PerfDumpDir gives every call its own subdirectory under SUPKV_PERF_DUMP_DIR and logs path, filesystem type, device and rotational flag; make perf points the base at $(CURDIR)/.tmp/perf."
  - "The device-mapper root path falls back to rotational=unknown as the plan documents, rather than resolving the dm device through the nvme backing disk."
patterns-established:
  - "Perf gate: a perf test's first statement is testutil.RequirePerf(t), which SKIPs (never fails) naming `make perf` and SKIPs under -race."
  - "Verdict line: `PERF-02 burst: achieved=... writes/s (min ...) drain=... (bound ...) keys=... value-bytes=... fs=...` — one machine-greppable line per perf row."
  - "Hardware block: one `hardware: ...` line carrying CPU, logical CPUs, RAM, load, root-disk(device/fs/rotational), GOOS/GOARCH, Go version, module/VCS, race state and GOMAXPROCS."
requirements-completed: [PERF-02, PERF-06]
coverage:
  - id: D1
    description: "One compiled threshold table with the PERF-01..PERF-06 starting bounds, PerfEnabled/RequirePerf gates and the race build-tag predicate"
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/testutil/ -run 'TestPerfEnabled|TestRequirePerfSkips|TestPerfThresholdRelations|TestRaceEnabledStable'"
        status: pass
      - kind: command
        ref: "go tool golangci-lint run ./internal/testutil/"
        status: pass
    human_judgment: false
  - id: D2
    description: "The hardware block plus the dump-directory filesystem/device/rotational guard, on both platforms"
    requirement: PERF-06
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/testutil/ -run 'TestHardwareBlock|TestPerfDumpDirResolvesUnderBase|TestPerfDumpFSType|TestPerfDumpDevice'"
        status: pass
      - kind: command
        ref: "GOOS=darwin go vet ./internal/testutil/ && go vet ./internal/testutil/"
        status: pass
    human_judgment: false
  - id: D3
    description: "The 1000 writes/s distinct-key burst as an operator-runnable perf row with the frozen-contract drift guard"
    requirement: PERF-02
    verification:
      - kind: command
        ref: "make perf (exit 0): achieved=1000.1 writes/s, drain=96.5ms, unacked set empty"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 -run '^TestPerfBurst' -v ./internal/stand/ (row SKIPs naming make perf; drift guard PASSes)"
        status: pass
    human_judgment: false
  - id: D4
    description: "`make perf`: Linux-only guard, .tmp/perf dump base, -p 1, no -race, perf rows then the one functional stand-suite line"
    requirement: PERF-06
    verification:
      - kind: command
        ref: "make perf > /tmp/sup-kv-perf-08-01.log 2>&1 (exit 0; hardware block present; internal/stand appears twice)"
        status: pass
    human_judgment: false
duration: 12min
completed: 2026-11-23
status: complete
---

# Phase 08 Plan 01: Perf Harness Summary

**One compiled threshold table, a named-skip perf gate, a hardware block and dump-media guard, and `make perf` proving the frozen 1000 writes/s burst at 1000.1 writes/s on the real binary.**

## Performance

- **Duration:** 12min (746s)
- **Started:** 2026-11-23T17:41:03Z
- **Completed:** 2026-11-23T17:53:29Z
- **Tasks:** 3 (two TDD: RED + GREEN each)
- **Files modified:** 9 (8 created, 1 modified)

The measured `make perf` run on this host (calm: load 1.85 at start, performance governor):

```
hardware: CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=1.85 1.36 1.52; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16
perf dump dir: /home/nil/DiskD/W/Djarvur/sup-kv/.tmp/perf/perf-3544544080 fs=ext4 dev=/dev/mapper/ubuntu--vg-ubuntu--lv,rotational=unknown
PERF-02 burst: achieved=1000.1 writes/s (min 990) drain=96.537607ms (bound 5s) keys=10000 value-bytes=1024 fs=ext4
```

The functional stand-suite line ran green (`ok .../internal/stand 6.154s`), and `go test -race ./...` (make test) was green with no perf row measured.

## Accomplishments

- `internal/testutil/perf.go` is the ONE threshold place (D-06): one compiled const block with the research's PERF-01…PERF-06 starting bounds and a basis comment per group; `PerfEnabled` opens only on the exact `SUPKV_PERF=1` spelling; `RequirePerf` SKIPs (never fails) naming `make perf`, and SKIPs under `-race`.
- `race_on.go`/`race_off.go` answer `RaceEnabled()` from the build tag the compiler selected — no runtime probe in a test body.
- `internal/testutil/perf_linux.go` assembles the one-line hardware block from best-effort `/proc` reads (cpu model, logical CPUs, RAM, load), `debug.ReadBuildInfo` (module/VCS when present), the race state, GOMAXPROCS and the root mount's device/filesystem/rotational flag from `/proc/self/mountinfo` + `/sys/block/*/queue/rotational`; a missing file degrades one field to `unknown`.
- `PerfDumpDir` gives every call a fresh per-test subdirectory under `SUPKV_PERF_DUMP_DIR` (else `t.TempDir()`) and logs path, filesystem type, device and rotational flag; `PerfDumpFSType` names the backing filesystem by statfs magic so a tmpfs/ramfs dump is visible; `PerfDumpDevice` renders `<device>,rotational=<0|1|unknown>` and `unknown` for unresolvable paths. The non-Linux pair keeps all four signatures with documented `unknown` facts.
- `internal/stand/perf_test.go` adds `TestPerfBurstDistinctKeys`: the frozen row's `redRunBurst`/constants reused read-only, the hardware block logged, the dumps under the operator base, the achieved rate against `testutil.PerfBurstMinRate`, the unacked gauge against `testutil.PerfBurstDrain + redBurstDrainMargin`, and exactly one `PERF-02 burst: ...` verdict line. `TestPerfBurstBoundsMatchRedContract` runs in the ordinary suite and fails by name if the perf threshold and the frozen red constants diverge.
- `make perf` (Makefile) refuses a non-Linux host by name, creates `$(CURDIR)/.tmp/perf`, runs the perf rows with `SUPKV_PERF=1`/`SUPKV_PERF_DUMP_DIR` under `-p 1` without `-race` and a 30m timeout, then runs `internal/stand` once more with `env -u SUPKV_PERF -u SUPKV_PERF_DUMP_DIR` (the functional stand-suite line, PERF-06's literal reading). CI workflows are untouched; `make bench-smoke` remains the threshold-free CI smoke target.

## Task Commits

1. **Task 1 RED: perf gate/threshold tests** — `480367b` (test)
2. **Task 1 GREEN: perf gate/threshold table** — `646d26e` (feat)
3. **Task 2 RED: hardware block/dump-guard tests** — `05498a5` (test)
4. **Task 2 GREEN: hardware block/dump-dir guard** — `2a3b71d` (feat)
5. **Task 3: burst perf row + make perf** — `67535a2` (feat)

**Plan metadata:** (docs commit follows this SUMMARY).

## Files Created/Modified

- `internal/testutil/perf.go` — the one compiled threshold table, `PerfEnabled`, `RequirePerf`, the shared dump-directory rule and `raceLabel`.
- `internal/testutil/race_on.go` / `race_off.go` — the `race`/`!race` build-tag pair for `RaceEnabled`.
- `internal/testutil/perf_test.go` — the gate, threshold-relation and race-stability tests.
- `internal/testutil/perf_linux.go` — `HardwareBlock`, `PerfDumpDir`, `PerfDumpFSType`, `PerfDumpDevice` over `/proc`, statfs and sysfs.
- `internal/testutil/perf_other.go` — the documented non-Linux four-signature pair.
- `internal/testutil/perf_linux_test.go` — the hardware-block and dump-guard tests.
- `internal/stand/perf_test.go` — `TestPerfBurstDistinctKeys` plus the frozen-contract drift guard.
- `Makefile` — the `perf` target and its `.PHONY` entry.

## Decisions Made

- Thresholds are compiled constants with a basis comment per group; the environment carries only the gate, never a bound.
- RED evidence was produced with a compiling skeleton (zero values / empty returns) so each target test failed on its planned assertion; a missing-file build failure would have been INVALID_RED under the TDD gate.
- The perf row is a thin caller over the frozen red generator; the plan's "extract a shared helper" is not executable because `stand_red_test.go` is byte-pinned, so the wrapper and a constant-drift guard cover the same ground without touching the pin.
- The device-mapper root keeps `rotational=unknown` per the plan's documented fallback (the host's root is `/dev/mapper/ubuntu--vg-ubuntu--lv`, whose basename has no `/sys/block` entry).

## Deviations from Plan

### Process / TDD evidence

**1. [Process - TDD evidence] RED used a compiling skeleton instead of the planned missing-file failure**
- **Found during:** Task 1 RED
- **Issue:** The task text says the tests fail because `internal/testutil/perf.go` does not exist; with `tdd_mode` enabled, a package that does not compile is `build_failure`/INVALID_RED and cannot authorize GREEN.
- **Fix:** The RED commits carry a minimal compiling skeleton (wrong gate value, empty helpers, zero thresholds); each named test failed on its assertion, and the GREEN commits replaced the skeleton with the real implementation. `gsd_run check tdd-red-evidence` returned `RED_EVIDENCE_OK` for both RED records.
- **Files modified:** `internal/testutil/perf.go`, `internal/testutil/perf_linux.go`, `internal/testutil/perf_other.go`
- **Verification:** `RED_EVIDENCE_OK` for `TestPerfEnabled` (task 1) and `TestHardwareBlock` (task 2).
- **Commit:** `480367b`, `05498a5`

**2. [Process - Commit granularity] Each TDD task produced a RED and a GREEN commit**
- **Found during:** All tasks
- **Issue:** The plan asks for each task to be committed atomically; the TDD gate additionally requires a `test(...)` commit before the `feat(...)` commit.
- **Fix:** Five commits: RED+GREEN for Tasks 1 and 2, one commit for Task 3 (non-TDD). Both RED commits precede their GREEN commits.
- **Verification:** `git log` order and TDD Gate Compliance below.

**3. [Implementation] `PerfDumpDir` is exported from both tag files but delegates to one shared helper**
- **Found during:** Task 2 GREEN
- **Issue:** The plan assigns `PerfDumpDir` to both `perf_linux.go` and `perf_other.go`; duplicating the 20-line env/temp-dir rule invites drift and `dupl`.
- **Fix:** Both tag files keep the exact exported signature, delegating to `perfDumpDir` in `perf.go` with the platform's `PerfDumpFSType`/`PerfDumpDevice` as arguments.
- **Verification:** `GOOS=darwin go vet ./internal/testutil/` and `go vet ./internal/testutil/` both clean.

## TDD Gate Compliance

| Gate | Commit | Result |
|------|--------|--------|
| RED (Task 1) | `480367b` test(08-01): add failing test for the perf gate and threshold table | `RED_EVIDENCE_OK` (TestPerfEnabled failed on the gate values) |
| GREEN (Task 1) | `646d26e` feat(08-01): implement the perf gate and threshold table | All four target tests pass |
| RED (Task 2) | `05498a5` test(08-01): add failing test for the hardware block and dump-dir guard | `RED_EVIDENCE_OK` (TestHardwareBlock failed on the empty line) |
| GREEN (Task 2) | `2a3b71d` feat(08-01): implement the hardware block and dump-dir guard | All four target tests pass; darwin vet clean |
| Task 3 (not TDD) | `67535a2` feat(08-01): add the burst perf row and make perf target | `make perf` green, drift guard green |

The RED evidence records and full console reports are under `.planning/phases/08-performance/red-evidence/` (untracked, matching the phase-2 red-evidence convention).

## Issues Encountered

- The burst row and the functional stand-suite line both ran green; the known F-6 `TestStand_persist*` SIGKILL/kill-diagnosis class did not fire in either invocation.
- The hardware block's module VCS fields report `vcs unknown, unknown` because the `go test` test binary carries no VCS build settings; the module path is present and the helper is documented best-effort (the plan says "when present"). The stand's spawned product binary is built separately and is unaffected.
- No threshold was edited; the measured rate (1000.1 writes/s) cleared the 990 writes/s bound on the first run at load 1.85.

## User Setup Required

None — no external service configuration required.

## Known Stubs

None. All helpers and the perf row are wired to real sources; no placeholder values or TODOs remain.

## Threat Flags

None. The plan's threat register is mitigated: the threshold/verdict messages name the PERF ID and constants (T-08-01), the frozen red file still hashes to `573fed68d2f1a039b8ac023a342446103064273b` and GitHub workflows are untouched (T-08-02), and every run logs CPU/RAM/load/dump-fs/race state (T-08-03). No package was installed; the helper package uses the standard library only (T-08-SC).

## Next Phase Readiness

- Ready for 08-02: the threshold table, gate, hardware block, dump guard and `make perf` exist, so the later stand rows (latency, amplification, memory) add measurement code to a proven pipeline.
- The perf row is reachable only through `make perf`; `make test` stays threshold-free and green (verified with `make test`, exit 0).
- The 08-11 gate should re-run `make perf` after 08-09 on the final tree, as the plan's assumptions require.

## Self-Check: PASSED

- `internal/testutil/perf.go` — FOUND
- `internal/testutil/race_on.go` / `race_off.go` — FOUND
- `internal/testutil/perf_linux.go` / `perf_other.go` — FOUND
- `internal/stand/perf_test.go` — FOUND
- `Makefile` perf target — FOUND
- Commits `480367b`, `646d26e`, `05498a5`, `2a3b71d`, `67535a2` — ancestors of HEAD
- `make perf` — PASSED (1000.1 writes/s, hardware block, functional line)
