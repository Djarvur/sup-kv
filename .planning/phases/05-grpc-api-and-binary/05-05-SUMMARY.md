---
phase: 05-grpc-api-and-binary
plan: 05
subsystem: api
tags: [wait-for-replication, replication-report, acked-seam, timeouts, bufconn, honesty]

# Dependency graph
requires:
  - phase: 05-grpc-api-and-binary
    provides: 05-01's bufconn harness and the SetRequest/DeleteRequest wait fields committed in the wire contract; 05-04's gated Set/Delete and the ServerOptions options-struct pattern
  - phase: 04-replication-and-deterministic-simulator
    provides: repl.PendingInfo.Acked (05-02's honesty half), udprepl Adapter.KnownPeers/PendingCopy copy-on-write mirrors (05-02) — the exact seams the injected closures carry in production
provides:
  - grpcapi.waitForReplication (wait.go): the single wait algorithm — write-time peer snapshot (D-10), zero-peer short-circuit (D-11, nil seams included), appearance phase (waitAppear 500ms cap) closing the notify-queue race (Pitfall 4), drain phase reading only the Acked seam with frozen supersession counts (Q2), deadline/ctx always honest-false
  - ServerOptions.KnownPeers func() uint64 / PendingCopy func() map[string]repl.PendingInfo / WaitTimeout time.Duration (NewServer normalizes zero/negative to the 1s A2 default) — the seams 05-06 main fills with the adapter mirrors
  - Set/Delete report wiring: wait_for_replication decorates the D-08 aggregate after engine confirmation; per-request wait_timeout_ms overrides the default (D-09); the report fields keep zero values without the flag
  - TestWait_* family (8 tests): zero peers (nil seams + explicit zero), acks arrive, honest timeout with OK outcome, delete symmetry, appearance race, eviction-not-ack, supersession freeze, per-request timeout
affects: [05-06, phase-6-hardening]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 8831    # chars/4 over the realized diff (35324 chars)
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns: [two-phase wait (appearance-before-drain) against a copy-on-write mirror, write-time peer snapshot captured once before polling, counts-from-Acked-only honesty (eviction and supersession distinguishable), mutex-guarded deep-copying fake mirror under -race]

key-files:
  created:
    - internal/grpcapi/wait.go
    - internal/grpcapi/wait_test.go
  modified:
    - internal/grpcapi/server.go
    - internal/grpcapi/gate.go
    - internal/grpcapi/server_test.go
    - internal/grpcapi/apiproto/supkvapi.proto
    - internal/grpcapi/apiproto/supkvapi.pb.go

key-decisions:
  - "waitForReplication reads only the injected ServerOptions seams — handler goroutines never touch the single-driver repl core; total is captured exactly once before any polling (D-10), and both the nil-seam wiring and an explicit zero count short-circuit to (true,0,0) (D-11)"
  - "The appearance phase (waitPoll 5ms, waitAppear 500ms cap, bounded also by the wait deadline) closes the notify-queue race: absence is not drained until the (key, version) entry had its bounded chance to appear; never appearing is an honest (false,0,total)"
  - "Drain honesty counts only from Acked: WaitingOn empty with Acked empty is eviction, never an Ack (T-05-04/Pitfall 3); a version-mismatch entry freezes the counts at the last observation (Q2); deadline and ctx cancellation always report acked=false — the wait never changes the write outcome, which the engine fixed before the wait started"
  - "Proto wait-field comments made live and regenerated via buf — the stale accepted-but-deferred wording would misstate the now-live wire contract (Rule 2); generated .pb.go rides make proto per the 05-01 adaptation"

patterns-established:
  - "Two-phase wait against a mirror: appearance (bounded) then drain (until absent/resolved/superseded/deadline), one 5ms poll tick per iteration with the Go 1.23+ Reset discipline — no drain selects, one live timer per phase"
  - "Fake-mirror test pattern: mutex-guarded mutable map whose reader returns fresh deep copies (successive immutable snapshots), mutations driven from test goroutines; engine.Get polling supplies the engine-stamped version a fake entry must carry"

requirements-completed: [API-05]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "wait_for_replication on Set and Delete reports the D-08 aggregate within the per-request or default timeout and never changes the write outcome — the timed-out Set returns OK with the honest false report"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_timeout (OK outcome + acked=false 0/1 under a 100ms option)"
        status: pass
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_deleteSymmetry (delete-as-write, acked=true 0/0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Zero known peers reports acked=true immediately with total_peers 0 (D-11) — both the nil-seam single-node default and an explicit zero count, without the appearance wait"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_zeroPeers (nil seams, elapsed well under 400ms)"
        status: pass
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_deleteSymmetry (explicit KnownPeers=0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The wait first waits for the (key, version) entry to appear before absence means drained — no false acked from the notify-queue race (Pitfall 4); full Ack coverage reports acked=true with correct counts"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_appearanceRace (60ms empty phase, elapsed floor pinned)"
        status: pass
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_acksArrive (absent -> Acked=[1] resolved: acked=true 1/1)"
        status: pass
    human_judgment: false
  - id: D4
    description: "An evicted peer is never counted as an Ack — WaitingOn empty with Acked empty reports honest acked=false 0/total (Pitfall 3, T-05-04)"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_evictionNotAck"
        status: pass
    human_judgment: false
  - id: D5
    description: "A superseded entry reports acked=false with counts frozen at the last observation — the superseding version's Acked set never leaks (Q2 pin)"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_superseded (false, 1/2 frozen)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The timeout is the per-request wait_timeout_ms when positive, else the operator-tunable 1s default (D-09, A2); total is the write-time peer snapshot (D-10)"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_perRequestTimeout (120ms request field wins over the 1s default, elapsed under 400ms)"
        status: pass
      - kind: integration
        ref: "internal/grpcapi/wait_test.go#TestWait_timeout (100ms via opts)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Plan-level gates: full race+cover suite and the full-repo strict lint green; go.mod/go.sum untouched (T-05-SC — no package installs)"
    requirement: API-05
    verification:
      - kind: other
        ref: "command:make test && make lint (both exit 0); TestWait_ family green under -race -count=2"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-11-09
status: complete
commits: 2
plan_head_before: f776e289356953ff9b5fb10cf027b0d6d20d73ca
plan_head_after: 9a59ba65da13ede626ce21bb34e622c85d115b9a
---

# Phase 05 Plan 05: wait_for_replication Summary

**wait_for_replication delivered as one honest report: the two-phase wait (appearance-before-drain) over the injected KnownPeers/PendingCopy seams, counts read only from the Acked seam, wired into Set and Delete behind the per-request timeout — eight bufconn tests lock every D-08..D-11 reading and all four research pitfalls under -race**

## Performance

- **Duration:** 12 min
- **Started:** 2026-11-09T12:52:51Z
- **Completed:** 2026-11-09T13:05:32Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- `grpcapi.ServerOptions` gains the wait seams: `KnownPeers func() uint64`, `PendingCopy func() map[string]repl.PendingInfo`, `WaitTimeout time.Duration` — NewServer normalizes zero/negative WaitTimeout to the 1s design default (D-09/A2); the nil seams are the legal single-node wiring (D-11)
- `grpcapi.waitForReplication` (wait.go): total captured exactly once from KnownPeers before any polling (D-10); zero peers short-circuits `(true, 0, 0)` immediately — no appearance wait; the appearance phase polls the mirror every waitPoll=5ms bounded by the earlier of the deadline and waitAppear=500ms so pre-notify absence is never misread as drained (Pitfall 4); the drain phase tracks lastAcked as the maximum len(Acked) observed and resolves on entry-absent (drained), version-mismatch (superseded — counts frozen, Q2), WaitingOn-empty (resolved — only Acked counts, eviction is never an Ack, T-05-04), or deadline/ctx cancellation (always honest false)
- Set and Delete wired: strictly after the engine confirmed the write and the notification is queued, a `wait_for_replication` request decorates the D-08 aggregate with the timeout resolved per-request-first (wait_timeout_ms > 0) else the default; no code path after engine confirmation can turn the write into an error; without the flag the report fields keep their zero values
- Eight-test TestWait_* family over the bufconn harness with a mutex-guarded deep-copying fake mirror: zero-peer immediacy (nil seams, elapsed well under 400ms), acks arrive 1/1 through a 30ms absent phase, honest 100ms timeout with the Set RPC still OK, delete symmetry, the 60ms appearance race (elapsed floor pinned), eviction-not-ack, supersession freeze at 1/2, and the 120ms per-request timeout beating the 1s default — all green under `-race` and `-count=2`
- Proto wait-field comments updated to the live semantics and regenerated via `make proto` (buf); full `make test` (race+cover) and the full-repo strict lint gate green

## Task Commits

Each task was committed atomically:

1. **Task 1: waitForReplication — aggregate report, zero-peer short-circuit, timeout, wiring** - `6b746d5` (feat)
2. **Task 2: Adversarial wait edges — appearance race, eviction, supersession, per-request timeout** - `9a59ba6` (test)

## Files Created/Modified
- `internal/grpcapi/wait.go` - defaultWaitTimeout/waitPoll/waitAppear consts, waitForReplication (appearance + zero-peer short-circuits), waitForDrain (absent/superseded/resolved/deadline readings), waitTimeout resolver; two sanctioned //nolint:gosec directives with written justifications (the peer table's uint16 key space bounds both conversions)
- `internal/grpcapi/wait_test.go` - fakeMirror (mutex + deep-copying PendingCopy), newWaitServer/awaitWrittenVersion helpers, the eight TestWait_* tests, supersededVersion (0xAA fill cannot collide with an RFC 9562 stamp)
- `internal/grpcapi/server.go` - NewServer normalizes WaitTimeout; Set/Delete take ctx, run the wait strictly after engine confirmation + notify, decorate the D-08 report fields; handler docs updated to the live semantics
- `internal/grpcapi/gate.go` - ServerOptions gains KnownPeers/PendingCopy/WaitTimeout with the seam contracts documented (the struct lives here, not server.go)
- `internal/grpcapi/server_test.go` - testServerConfig gains waitTimeout/knownPeers/pendingCopy, merged over the gate wiring in buildTestServer
- `internal/grpcapi/apiproto/supkvapi.proto` + regenerated `supkvapi.pb.go` - SetRequest/SetResponse comments now describe the live wait behavior (wire contract unchanged — comments only)

## Decisions Made
- The drain's entry-absent reading returns `(lastAcked == total, lastAcked, total)` exactly as pinned: success only at observed full Acked coverage, so a mirror that skips from pending to absent between polls reports an honest false rather than a success it never saw
- The version-mismatch drain branch models the mirror tracking a *different* (greater) version for the key after supersession — the state a wait can genuinely observe when a newer write replaces the pending entry; the Q2 freeze keeps the superseding entry's Acked set out of the report
- Timer discipline per the repo convention: one live `time.Timer` per phase, Reset after receive, Stop deferred — no drain selects (Go 1.23+ semantics)
- Test goroutines never call `t` — `awaitWrittenVersion` fails silently and the caller's assertions catch the miss (goroutine-safe failure reporting)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Harness extension in server_test.go (outside the plan's files_modified)**
- **Found during:** Task 1
- **Issue:** The plan's tests require "the 05-01 harness with ServerOptions carrying fake closures", but testServerConfig had no knobs for KnownPeers/PendingCopy/WaitTimeout — the wait tests cannot compile without them
- **Fix:** testServerConfig gained waitTimeout/knownPeers/pendingCopy; buildTestServer merges them over the gate wiring before NewServer; zero/nil values keep the documented defaults so no existing test changes behavior
- **Files modified:** internal/grpcapi/server_test.go
- **Verification:** the entire pre-existing TestServer_*/TestWatch_*/TestGate_*/TestSnapshot_* suites pass unmodified under -race
- **Committed in:** 6b746d5

**2. [Rule 3 - Blocking] ServerOptions edit landed in gate.go, not server.go**
- **Found during:** Task 1
- **Issue:** The plan lists server.go for the ServerOptions extension, but the struct lives in gate.go (the 05-04 layout); server.go only needed the NewServer normalization
- **Fix:** Fields + seam docs added in gate.go; NewServer's normalization and doc update in server.go — same package, purely mechanical placement
- **Files modified:** internal/grpcapi/gate.go, internal/grpcapi/server.go
- **Verification:** compile + full package suite green; the acceptance criteria read identically (fields declared, default applied)
- **Committed in:** 6b746d5

**3. [Rule 2 - Missing critical] Proto wait-field comments made live**
- **Found during:** Task 1
- **Issue:** The committed wire contract still said the wait fields are "accepted but not acted on until plan 05-05" — a doc lie now that the fields are live; this plan makes them live, so the contract text must match
- **Fix:** SetRequest/SetResponse comments updated to the live semantics (DeleteRequest/DeleteResponse wording was already correct); regenerated via `make proto` — buf-pinned toolchain, comment-only diff, wire shape untouched
- **Files modified:** internal/grpcapi/apiproto/supkvapi.proto, internal/grpcapi/apiproto/supkvapi.pb.go
- **Verification:** buf lint + `buf format --diff --exit-code` green; generated diff is comments only
- **Committed in:** 6b746d5

---

**Total deviations:** 3 auto-fixed (2 blocking-environment/harness adaptations, 1 missing-critical doc correction)
**Impact on plan:** All three are mechanical placement/infrastructure/doc adaptations; the pinned algorithm, wire contract and every acceptance criterion are unchanged. No scope creep.

## Issues Encountered
- Four strict-lint findings on first gate run fixed in code direction: revive confusing-results (drain results named), enforce-map-style (make() in the fake), and two gosec G115 conversions narrowed to one hoisted conversion site per poll with written justifications (the uint16-keyed peer table bounds both — the 04-01 sanctioned-directive precedent); golines forced the hoist when the inline-comment form exceeded its 200-char budget. No suppressions without justification; every gate re-run green after.

## Known Stubs
None — the wait is fully implemented end-to-end; no placeholder, TODO or unwired surface was introduced. The ServerOptions seams await 05-06 main to inject the production adapter mirrors (`adapter.KnownPeers`, `adapter.PendingCopy`, `cfg.WaitTimeout`).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- API-05 delivered end-to-end at the API layer: 05-06 main completes the wiring in three lines — `ServerOptions{SyncArrived: adapter.SyncArrived, KnownPeers: adapter.KnownPeers, PendingCopy: adapter.PendingCopy}` plus the operator's wait_timeout default — and the report goes live over the real mirrors
- The harness (buildTestServer) remains the working reference implementation of the full ServerOptions contract (gate + wait seams together)
- No blockers; go.mod/go.sum untouched (T-05-SC holds — no package installs)

---
*Phase: 05-grpc-api-and-binary*
*Completed: 2026-11-09*

## Self-Check: PASSED

All six created/modified source files exist on disk; commits 6b746d5 and 9a59ba6 are ancestors of HEAD; `commits: 2` is measured from the persisted ledger (`git rev-list --count f776e289..HEAD`); plan-level gates green at SUMMARY time: `make test` (full race+cover suite) and `make lint` (config verify + full-repo run) both exit 0; the TestWait_ family is stable under `-race -count=2`; go.mod/go.sum untouched (T-05-SC: no package installs).
