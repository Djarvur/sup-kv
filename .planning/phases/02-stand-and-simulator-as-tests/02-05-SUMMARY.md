---
phase: 02-stand-and-simulator-as-tests
plan: 05
subsystem: testing
tags: [red-tests, stand, real-socket, r01, r02, r03, proto-01, xport-01, xport-02, d01, linux-only]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: 02-01's D-01 red-gating mechanism (SUPKV_RED_TESTS=1 gate, red-runner, expectation table) and 02-02's internal/stand harness (build-once binary, asymmetric-port pair, SIGKILL/SIGTERM lifecycle, deadline-clamped waits, /metrics and dump seams)
  - phase: 04-replication (v1.0 shipped)
    provides: the defect-bearing production code the four scenarios pin (gate opens on the first DataSync chunk, reads never consult the gate, per-event unacked-set cost, unpaced egress)
provides:
  - The four stand-level Critical red tests — R-01 paced sync, R-02 unacked drain, R-02 burst, R-03 F14 replaced-node — all FAIL against v1.0 with SUPKV_RED_TESTS=1 and SKIP without it
  - SUPKV_STAND_NIGHTLY scale selection proven at the documented 20 000-key R-01 scale and the larger R-03 listing
  - The stand-level flip contract: Phases 3-5 change only the expectation-table rows; no test edits
affects: [02-12 (the complete P-01 table, the runner's package list and the recorded full run), 02-11 (the scheduled red-run job sets SUPKV_STAND_NIGHTLY=1), phases 3-5 (XPORT-01/XPORT-02 and PROTO-01 flips)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 9521     # chars/4 over the realized diff: one 38 085-byte file (the estimate assumed writing it from scratch; the interrupted prior attempt's draft was reviewed, verified and completed rather than rewritten)
  tasks: 2
  commits: 2       # MEASURED from the plan-start ledger at SUMMARY write (#3968): git rev-list --count 265536f..HEAD
  plan_head_before: 265536f450e26c04378cd5ec9e869b9717d7be52
  plan_head_after: 2f274f1b902c1900dd6f6ed883c91a11d0f3cddb

# Tech tracking
tech-stack:
  added: none — stdlib plus the already-required grpc/apiproto; the R-03 test imports internal/app/version for the typed version comparison
  patterns:
    - env-gated expected-red stand tests (D-01): the body's first statement skips unless SUPKV_RED_TESTS=1
    - shaped closed window for a red acceptance: assert closure while the peer is absent, never race a live multi-chunk listing
    - timer-derived bounds only: paced = N × (1s/sync_response_rate), margin = startup gate + sync/hello intervals + process slack; the reviewer's absolute seconds are cited in comments, never asserted
    - sequential red scenarios: no t.Parallel on wall-clock throughput/pacing tests

key-files:
  created:
    - internal/stand/stand_red_test.go
  modified: []

key-decisions:
  - "Every test asserts the DOCUMENTED acceptance expression, never the reviewer's absolute seconds: R-01 2×sync_interval + N/sync_response_rate; R-02 N/rate + 10 s with ≤3N retransmissions; R-02 burst ≥990 writes/s with the unacked set empty 5 s after the send window; R-03 the full observable PROTO-01 five-point acceptance. The Phase 3-5 flip is therefore the expectation-table row only."
  - "R-03's closed window is SHAPED, not raced: A is seeded and stopped, the empty-disk replaced node starts alone (no peer can deliver any listing) with a 30 s startup-gate backstop so v1.0's 1 s timer cannot open it, and only then does A restart. This makes the closed-side read assertion deterministic — v1.0's Get never consults the gate and answers NotFound, never Unavailable."
  - "R-01 uses the pure-resync shape: A is seeded and then RESTARTED so nothing is pending, leaving B's only path to the state the periodic DataSync anti-entropy. That is the review's F9b evidence, kept distinct from R-02's F9 retransmit-storm shape."
  - "Default scales breach v1.0 empirically (R-01 8 000 keys — the ~320 KiB listing overflows the default receive buffer; R-03 512 keys — ~16 DataSync chunks at the chunker's ~32 entries per 1400 B chunk), so a local red run stays viable; SUPKV_STAND_NIGHTLY=1 selects the documented 20 000-key R-01 scale and the 4 096-key R-03 listing."
  - "R-02 drain and burst run at the documented scale in both modes (10 000 unacked records; 1000 writes/s × 15 s): the verdict fires at the documented bound long before the reviewer's 14.5-minute storm, so a nightly variant would reproduce the same breach at more cost. The burst keeps a live peer (B) because the unacked-set acceptance needs a replication target; the load is applied to one node, the review's 'against one node' shape."
  - "The control-level helpers (redAssertCode, redGetRecord, redHealthStatus, awaitHealthAnswer) were added because standGet fatals on any non-NotFound error — exactly the Unavailable the closed-side assertion expects — and awaitServing waits for SERVING, exactly what the closed side denies."

patterns-established:
  - "A red test that must observe a closed state defines that state structurally (peer absent, gate backstop out of reach) instead of racing a live protocol window"
  - "When a harness success helper would fatal on the error a red assertion expects, read the raw RPC and assert the code explicitly"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "TestRed_R01_pacedSync20k — R-01/XPORT-01: the pure-resync F9b shape (A seeded then restarted; empty-disk B must serve every key within 2×sync_interval + N/sync_response_rate + margin); 8 000 keys locally, 20 000 under SUPKV_STAND_NIGHTLY=1"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_R01_pacedSync20k$' → exit 1: timed out after 20.6 s with 2 691/8 000 keys listed on B (bound 2×200 ms + 16 s + 4.2 s); test 56.0 s"
        status: pass
      - kind: unit
        ref: "SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 ... -run '^TestRed_R01_pacedSync20k$' → exit 1: timed out after 1 m 46 s with 2 976/20 000 keys (bound 2×30 s + 40 s + 6 s); test 388 s — the documented scale is RED and the nightly selection is wired"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestRed_' (no env var) → SKIP, exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestRed_R02_unackedDrain — R-02 part 1/XPORT-02(2): 10 000 unacked 1 KiB records held for a returning peer must drain within N/sync_response_rate + 10 s with at most 3N retransmissions read from A's /metrics (supkv_replication_unacked_records, supkv_replication_datagrams_retransmitted_total)"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "SUPKV_RED_TESTS=1 ... -run '^TestRed_R02_' → TestRed_R02_unackedDrain exit 1: the precondition reached 10 000/10 000 unacked, then the drain timed out after 35 s (bound N/rate 20 s + 10 s + 5 s); test 106.5 s"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestRed_R02_burst — R-02 part 2/XPORT-02(3): the reviewer's open-loop schedule (1000 writes/s × 15 s, 10 000 distinct keys, 1024 workers over 8 connections, 10 s per-RPC deadline) must achieve ≥990 writes/s and leave the unacked set empty 5 s after the send window's end"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "SUPKV_RED_TESTS=1 ... -run '^TestRed_R02_' → TestRed_R02_burst exit 1: 51.8 writes/s (5 750/15 000 OK, 9 250 DeadlineExceeded), want ≥990; test 111.2 s"
        status: pass
    human_judgment: false
  - id: D4
    description: "TestRed_R03_f14ReplacedNode — R-03/PROTO-01 (ADR 0005, decision 14.5): the full observable five-point acceptance with a multi-datagram listing — (a) Unavailable for reads and writes before the full listing, (b) the F14 seeding attempt does not change A's user value, (c) health NOT_SERVING until the full listing, (d) create_if_absent for a peer-known key returns AlreadyExists after it, (e) the next local write after the max-version part takes a greater version"
    requirement: TEST-04
    verification:
      - kind: unit
        ref: "SUPKV_RED_TESTS=1 go test -race -count=1 ./internal/stand/ -run '^TestRed_R03_f14ReplacedNode$' → exit 1: 'Get … before any listing: want code Unavailable, got NotFound' in 2.0 s — the R-03 defect (reads never consult the gate)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestRed_' (no env var) → all four TestRed_ SKIP, exit 0"
        status: pass
    human_judgment: false
  - id: D5
    description: "Normal gates stay green on the committed tree; the plan's phase boundary holds — no production non-test file changed"
    requirement: TEST-04
    verification:
      - kind: other
        ref: "make lint → exit 0 (config verify + run ./...)"
        status: pass
      - kind: other
        ref: "make test (go test -race -cover ./...) → exit 0; internal/stand ok 6.18 s, coverage 81.8%"
        status: pass
      - kind: other
        ref: "git diff --name-only 265536f..HEAD -- internal/app internal/udprepl internal/grpcapi internal/persistfile → empty"
        status: pass
    human_judgment: false

# Metrics
duration: 17min
completed: 2026-11-21
status: complete
---

# Phase 2 Plan 5: Stand-Level R-01/R-02/R-03 Red Tests Summary

**The four stand-level Criticals are pinned as machine-reproducible red tests: R-01 paced sync, R-02 unacked drain and burst throughput, and R-03's F14 replaced-node acceptance all fail against v1.0 on real sockets under `SUPKV_RED_TESTS=1` and skip without it, with bounds written as the documented expressions so Phases 3-5 flip only the expectation table.**

## Performance

- **Duration:** 17 min (~1 044 s), including two full red-suite runs (~277 s each), one documented-scale nightly run (388 s), and the repo-wide gates
- **Started:** 2026-11-21T10:22Z
- **Completed:** 2026-11-21T10:40Z
- **Tasks:** 2
- **Files modified:** 1 created

## Accomplishments

- `internal/stand/stand_red_test.go` (new, 870 lines, `//go:build linux`, package `stand`) — the four stand-level red scenarios, each env-gated on `SUPKV_RED_TESTS=1` as its first statement (D-01) and written as the full documented acceptance:
  - `TestRed_R01_pacedSync20k` — the F9b pure-resync shape: A seeds N × 1 KiB keys, is restarted so nothing is pending, and the empty-disk B must serve every key within `2×sync_interval + N/sync_response_rate + margin`. Local scale 8 000 keys (~16-chunk listing overflow); `SUPKV_STAND_NIGHTLY=1` runs the documented 20 000 keys with the documented 30 s cadence and 1 s hello interval.
  - `TestRed_R02_unackedDrain` — 10 000 unacked 1 KiB records held by A while B is down must drain within `N/sync_response_rate + 10 s` with at most 3N retransmissions, both read from A's opt-in `/metrics` (`supkv_replication_unacked_records`, `supkv_replication_datagrams_retransmitted_total`).
  - `TestRed_R02_burst` — the reviewer's open-loop schedule (1000 writes/s × 15 s, 10 000 distinct keys, 1024 workers over 8 connections, 10 s RPC deadline) against one node must achieve ≥990 writes/s and leave the unacked set empty 5 s after the send window's end. Achieved is measured the reviewer's way: successful ops over elapsed, elapsed covering the full in-flight drain.
  - `TestRed_R03_f14ReplacedNode` — the full observable PROTO-01 acceptance (ADR 0005 / decision 14.5) across a multi-datagram listing: closed reads and writes plus NOT_SERVING before the complete listing, AlreadyExists for a peer-known key after it, the user value unchanged on both nodes, and the next local write above the listing's max version.
- The R-03 closed window is deterministic by construction, not raced: node A is seeded and stopped, the replaced node B starts alone with a 30 s startup-gate backstop (so v1.0's 1 s timer cannot open it), and only then does A restart and stream its listing.
- Empirical red evidence on the committed tree (v1.0 production code):
  - R-01 local: 2 691/8 000 keys after the 20.6 s bound (test 56.0 s).
  - R-01 nightly (documented scale): 2 976/20 000 keys after the 1 m 46 s bound (test 388 s).
  - R-02 drain: the unacked set never drains within the 35 s bound (test 106.5 s).
  - R-02 burst: 51.8 writes/s, 5 750/15 000 OK, 9 250 `DeadlineExceeded` (test 111.2 s).
  - R-03: `Get` answers `NotFound`, never `Unavailable`, in 2.0 s.
- `make lint` and `make test` stay green on the committed tree; no production non-test file changed (the plan's phase boundary).

## Task Commits

Each task was committed atomically:

1. **Task 1: R-01 and R-02 red scenarios on the stand** - `cbc8090` (test)
2. **Task 2: R-03 stand-level F14 acceptance red test** - `2f274f1` (test)

**Plan metadata:** the `docs(02-05)` commit following this SUMMARY (carries SUMMARY + STATE/ROADMAP/REQUIREMENTS updates).

_Measured from the plan-start ledger `265536f` (`git rev-list --count 265536f..HEAD` at SUMMARY write): 2 task commits._

## Files Created/Modified

- `internal/stand/stand_red_test.go` — the four stand-level red tests, their timer-derived bounds, the `/metrics` scrape/parse helpers, and the control-level RPC/health helpers.

## Decisions Made

- The documented acceptance expressions are asserted, never the reviewer's absolute seconds (cited in comments): R-01 `2×sync_interval + N/rate`; R-02 `N/rate + 10 s` and `≤3N`; burst `≥990/s` and a 5 s drain; R-03 the five-point PROTO-01 acceptance. The Phase 3-5 flip is the expectation-table row only.
- R-03's closed window is shaped (peer absent, backstop out of reach) rather than raced against a live multi-chunk listing.
- R-01 uses the F9b pure-resync shape, kept distinct from R-02's F9 storm shape.
- Default scales breach v1.0 empirically; `SUPKV_STAND_NIGHTLY=1` selects the documented R-01 scale and the larger R-03 listing.
- R-02 drain and burst run at the documented scale in both modes; the verdict fires at the documented bound long before the reviewer's 14.5-minute storm.
- New control-level helpers were needed because `standGet` fatals on the `Unavailable` a closed read must return, and `awaitServing` waits for the `SERVING` the closed side denies.

## Deviations from Plan

None - plan executed exactly as written.

The interrupted prior attempt's draft (Task 1's three tests) was reviewed, empirically re-verified RED (not merely trusted), kept where it matched the plan, and completed with Task 2's R-03 test. Its header comment's note that the completed expectation table and the runner's package list belong to plan 02-12 was preserved.

## Issues Encountered

- The host was heavily loaded (load average 7-9 on 16 cores) during the runs: measured throughput is far below the review's VM figures (burst 51.8/s vs the review's 465/s). This only strengthens the red verdict; the assertions remain the documented bounds.
- The nightly R-01 seeds 20 000 × 1 KiB keys (~280 s on this host), so the scheduled red-run job's timeout must cover seeding plus the ~106 s bound (plan 02-11's 900 s-class budget is sufficient).
- A pre-existing untracked directory `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261119T193227Z/` (from the interrupted prior attempt; it contains only the grpcapi R-03 gate-test run, because `./internal/stand/` is still commented out of the runner's package list) was left untouched — it is not this plan's deliverable, and plan 02-12 owns the recorded red-run evidence.

## Known Stubs

None. The intentionally failing tests are the plan's deliverable, not stubs: their failure is machine-asserted by `scripts/red-runner` against the expectation table once plan 02-12 adds the rows, and their resolution path is the documented expectation-table flip (`red` → `green`) as Phases 3-5 land. No WINDOWS.md entries are added for red rows — the expectation table is their tracking ledger (the 02-01 precedent).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 02-12 must uncomment `./internal/stand/` in `scripts/red-runner`'s `PACKAGES` list and append the four rows to `scripts/red-expectations.tsv` (`red`), then record the full red run.
- Plan 02-11's scheduled job sets `SUPKV_STAND_NIGHTLY=1` alongside the runner's own `SUPKV_RED_TESTS=1` export; that selection now runs the documented R-01 scale and the larger R-03 listing.
- Phase 3 (XPORT-01/XPORT-02) flips the R-01 and R-02 rows; Phase 4 (PROTO-01) flips the R-03 row. No test edits are expected.
- No blockers: the R-01/R-02/R-03 defects remain unfixed by design (phase boundary: tests only).

## Self-Check: PASSED

- Key file exists: `internal/stand/stand_red_test.go` — FOUND.
- Commits exist and are ancestors of HEAD: `cbc8090` and `2f274f1` — FOUND.
- `gsd-tools check evaluation-scope --plan 02-05 --commits-only --raw` → `status: resolved`, two commits, `missingOnDisk: []`, `outsideUnion: []` — exit 0.
- Plan `<verification>` re-run on the committed tree: `SUPKV_RED_TESTS=1 go test -race ./internal/stand/ -run '^TestRed_R01|^TestRed_R02|^TestRed_R03_f14'` → exit 1 with all four tests FAILing (277 s) — the red verdict.
- `go test -race ./internal/stand/ -run '^TestRed_'` without the env var → exit 0, all four SKIP.
- No production non-test file modified from `265536f` to HEAD.
- `make lint` → exit 0; `make test` → exit 0 (stand package 6.18 s, 81.8% coverage).

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*
