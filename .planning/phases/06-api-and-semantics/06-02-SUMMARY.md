---
phase: 06-api-and-semantics
plan: 02
subsystem: api-and-replication
tags: [apis-01, write-time-peer-capture, wait-for-replication, honest-false, zero-peers, nil-seam, local-notify-peers, wait-set-seeding, wait-set-empty, peer-id-mirror, peer-generation, atomic-pointer, tracer-slice, tdd-red-evidence, status-complete]
requires:
  - phase: 05-persistence
    provides: the Outbox result stream (Ack/Superseded/WaitSetEmpty/Drained) and the confirmation seam the wait report consumes
  - phase: 04-protocol
    provides: the core peer table, the discovery flush and the retransmission ladder the tracked entry rides
  - phase: 06-api-and-semantics
    provides: the public api/supkvapi/v1 import path (06-01) every touched file already used
provides:
  - "ServerOptions.WritePeers plus one write-time capture per Set/Delete feeding BOTH the replication notification and the wait report"
  - "a peer-carrying LocalNotify -> adapter localWrite.peers -> Core wait-set seeded from exactly the captured ids"
  - "the honest (false, 0, 0) for zero captured peers and for a nil WaitReplicated seam, with the zero-peer release call preserved"
  - "WaitSetEmpty when a non-empty capture intersects an empty live peer table (a captured peer already gone cannot Ack)"
  - "Core.PeerIDs/PeerGeneration and the generation-gated Adapter.KnownPeerIDs mirror that feeds WritePeers in main"
affects:
  - 06-11-wait-expansion
  - 06-13-phase-gate
  - 06-03
  - phase-7-metrics
actuals:
  tokens: 17681
  tasks: 1
  commits: 2
  plan_head_before: 919a471243368cb90b099f6b58802ede449c8f56
  plan_head_after: fdc7b60bd8eeb15962bee9f0536d1350a464053c
tech-stack:
  added: []
  patterns:
    - "Write-acceptance capture: read the seam exactly once immediately before the engine call, clone it, and hand the SAME slice to the notification and the report — a seam that answers differently afterwards can no longer move the report."
    - "Peer-id mirror: one atomic.Pointer[[]uint16] published from the event loop only when the core's peer-id generation moved, so a handler reads a complete sorted set and an unchanged table pays no clone (the listing-mirror precedent)."
    - "Promise-set seeding: intersect the captured ids with the live peer table at tracking time and emit WaitSetEmpty when a non-empty capture has an empty intersection, instead of idling to the caller's timeout."
    - "TDD under a signature ripple: the mechanical call-site sweep compiles the new assertions while keeping the old behaviour, so the RED run is a real assertion failure rather than a refused build failure."
key-files:
  created:
    - .planning/phases/06-api-and-semantics/red-evidence/06-02-task1.json
  modified:
    - internal/grpcapi/gate.go
    - internal/grpcapi/wait.go
    - internal/grpcapi/server.go
    - internal/grpcapi/wait_test.go
    - internal/grpcapi/server_test.go
    - internal/grpcapi/bench_test.go
    - internal/grpcapi/wait_integration_test.go
    - internal/udprepl/repl.go
    - internal/udprepl/repl_test.go
    - internal/udprepl/repl_internal_test.go
    - internal/udprepl/smoke_test.go
    - internal/udprepl/three_node_test.go
    - internal/udprepl/scenario_test.go
    - internal/app/repl/core.go
    - internal/app/repl/bench_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/internal/simnet/simnet.go
    - cmd/sup-kv/main.go
key-decisions:
  - "The 18-file mechanical signature ripple landed IN the RED commit: the four new assertions cannot compile without the peer-carrying notification shape, so the sweep was written to keep the previous zero-peer/nil-seam answers while exposing the new seams. The recorded RED run then failed all four targets on their planned assertions (target TestWait_zeroPeers), which is a real assertion RED, not a build failure."
  - "The adapter's total==0 release keeps its (true, 0) verdict. The public report belongs to the handler, which now answers the honest (false, 0, 0); the call exists solely to delete the tracker the adapter created at notify time, so no tracker leaks on the zero-peer path (T-06-05). The seam's verdict is documented as a benign release, not a report."
  - "Core.NotifyReplayedWrite keeps its signature and seeds its wait-set from c.PeerIDs(). That is byte-for-byte the previous peer-table seeding, so a snapshot replay's tracking behaviour did not change: a replay has no waiter and is deliberately outside D-01's promise."
  - "The discovery flush is deliberately untouched. A peer discovered after acceptance still joins every pending wait-set (the IN-05 hole) until plan 06-11 splits promise-set enforcement; this plan's honesty comes from the captured total, not from rebuilding the wait set. The plan's scope rationale assigns that split to the expansion plan."
  - "PeerGeneration bumps on the two set-changing events only: a first-seen node id in handleHello, and an eviction in evictPeers. A run_id change under a known id is not a set change, so the mirror does not republish for it."
  - "WritePeers is cloned twice on the live path (Server.writePeers and the adapter's queue copy) and once more by KnownPeerIDs, so no caller can mutate a captured set; the cost is a handful of uint16 per write."
  - "The A-01 mechanism the previous SUMMARY flagged: the adapter's KnownPeers count stays for the metrics surface only; the write path uses KnownPeerIDs, and ServerOptions.KnownPeers is gone (the plan's prohibition)."
  - "The stale plan-head ledger (.git/gsd-plan-head-before-06-02, dated 2026-11-16, commit 03f2173) was reset to the pre-plan HEAD (919a471) before measuring, exactly as 06-01 had to do: it pointed 405 commits back, so the SUMMARY's commits: 2 is the measured git rev-list --count for this plan."
patterns-established:
  - "An honest wait report is proven by three separable assertions: the zero-peer/nil-seam answer, the single capture (seam call count == 1), and the capture's immutability against a seam that changes its answer mid-wait."
  - "A cross-goroutine peer-id read publishes ONE immutable slice pointer and a generation, never a live map: the handler's capture sees a complete sorted set or nothing."
requirements-completed: [APIS-01]
coverage:
  - id: D1
    description: "The handler captures the peer-id set exactly once immediately before the engine write and the same slice feeds the notification and the wait; zero captured peers and a nil WaitReplicated seam both answer the honest (false, 0, 0) immediately, and the zero-peer path still makes the total==0 release call."
    requirement: "APIS-01"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWait_(zeroPeers|trackerNilIsSingleNode|writeTimeSetPinned|deleteSymmetry)$' -> PASS (4/4; TestWait_zeroPeers and TestWait_deleteSymmetry assert false 0/0 plus exactly one total==0 release call, TestWait_writeTimeSetPinned asserts one capture, notification peers [1 2], wait total 2 and an unchanged report after the seam's answer changes, TestWait_trackerNilIsSingleNode asserts false 0/0 with five peers offered)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The write-time slice crosses notification -> adapter queue -> core wait-set: LocalNotify carries peers, localWrite clones them, trackLocalWrite seeds the wait set from the captured ids intersected with the live peer table and emits WaitSetEmpty when a non-empty capture is already gone, notifyReplayedWrite passes nil, and the simulator's Write/Delete/NotifyInstalled pass PeerIDs()."
    requirement: "APIS-01"
    verification:
      - kind: process
        ref: "go build ./... && go vet ./... -> exit 0 (go vet type-checks every test package, so a missed call site would fail here)"
        status: pass
      - kind: process
        ref: "go tool golangci-lint run ./... -> exit 0 (repo-wide strict lint)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./... -> every package ok (api 1.0s, cmd/sup-kv 11.7s, app/repl 131.3s, grpcapi 3.4s, udprepl 1.4s, stand 10.0s)"
        status: pass
    human_judgment: false
  - id: D3
    description: "One atomic peer-id mirror feeds production: Core.PeerIDs returns the sorted ids and Core.PeerGeneration moves only on an id-set change; Adapter.KnownPeerIDs publishes a cloned slice behind atomic.Pointer and republishes only when the generation moves; Adapter.KnownPeers stays for metrics; main wires WritePeers to KnownPeerIDs."
    requirement: "APIS-01"
    verification:
      - kind: process
        ref: "grep -q 'WritePeers:     adapter.KnownPeerIDs' cmd/sup-kv/main.go -> exit 0"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestWaitIntegration_' -> PASS (both real-UDP-pair tests run with opts.WritePeers = adapterA.KnownPeerIDs; ackedThroughRealAdapterTracker reports 1/1 and staleRecordNotAcked reports the honest false)"
        status: pass
    human_judgment: false
  - id: D4
    description: "TDD gate: a real RED record for the changed semantics precedes the GREEN that makes it pass."
    requirement: "APIS-01"
    verification:
      - kind: process
        ref: "gsd_run check tdd-red-evidence .planning/phases/06-api-and-semantics/red-evidence/06-02-task1.json --raw -> RED_EVIDENCE_OK (reason target_test_failed, 4 tests, 4 failing, target TestWait_zeroPeers)"
        status: pass
      - kind: process
        ref: "git log --oneline 919a471..HEAD -> b93d35a test(06-02), fdc7b60 feat(06-02) (RED precedes GREEN; no REFACTOR commit needed)"
        status: pass
    human_judgment: false
duration: 8min
completed: 2026-11-22
status: complete
---

# Phase 06 Plan 02: Write-Time Peer Capture and the Honest Zero-Peer Report Summary

**`wait_for_replication` now reports the peer set captured immediately before the engine write: zero peers and a nil seam answer the honest `(false, 0, 0)` instead of claiming an acknowledgement, and one cloned slice crosses handler → notification → core wait-set → report so a peer discovered mid-wait can never move the total.**

## Performance

- **Duration:** 8min (started 2026-11-22T22:23:46Z, completed 2026-11-22T22:31:38Z)
- **Tasks:** 1 tracer task (RED + GREEN, as the TDD gate requires)
- **Commits:** 2 (`b93d35a` test, `fdc7b60` feat) — measured `git rev-list --count 919a471..HEAD`
- **Files modified:** 19 paths (18 modified, 1 created: the RED evidence)

## Accomplishments

- **The zero-peer overstatement is gone.** `waitForReplication` no longer treats a zero peer count as an acknowledgement: with `WritePeers` returning nil (or the seam missing entirely) the report is `acked=false, acked_count=0, total_peers=0` immediately. This is the A-01 defect the phase exists to close.
- **The total no longer moves.** `Set` and `Delete` call `s.writePeers()` once, immediately before `engine.Set`/`SetIfAbsent`/`Delete`, and that same slice is passed to `notifyLocalWrite` and to `waitForReplication`. The helper clones the seam's answer, so a seam that reports different peers a moment later cannot alter the report — proven by `TestWait_writeTimeSetPinned`, which mutates the seam mid-wait and asserts the seam was consulted exactly once, the notification carried `[1 2]`, the wait carried total 2, and the response stayed `2/2`.
- **The zero-peer release point survived.** A zero-peer write still calls `WaitReplicated(ctx, key, ver, 0, timeout)`, which deletes the tracker the adapter created at notify time — the T-06-05 leak guard — while the handler answers the honest false.
- **The slice crosses the whole write half.** `LocalNotify` gained `peers []uint16`; the adapter clones them into its `localWrite` queue entry and the event loop hands `lw.peers` to `Core.NotifyLocalWrite`; `trackLocalWrite` seeds the pending entry's wait set from exactly those ids, intersected with the live peer table. A captured peer that has already been evicted can never Ack, so a non-empty capture with an empty intersection emits `out.WaitSetEmpty` immediately — the waiter learns the honest false instead of burning its timeout. `Server.notifyReplayedWrite` passes nil (a replay has no waiter) and `Core.NotifyReplayedWrite` keeps its signature, seeding from `PeerIDs()` exactly as it seeded from the peer table before.
- **Everything that names the notification shape was traced.** The simulator's `Write`, `Delete` and `NotifyInstalled` pass `c.nodes[i].core.PeerIDs()` (sorted, so same-seed replay stays deterministic); every adapter and core test call site passes `KnownPeerIDs()`/`PeerIDs()` for tracked writes and nil for untracked ones. `go build ./... && go vet ./...` is green, and vet compiles every test package — that is the completeness proof.
- **Production wiring is one atomic mirror.** `Core.PeerIDs()` returns `slices.Sorted(maps.Keys(c.peers))`; `Core.PeerGeneration()` bumps only when an id enters (first-seen Hello) or leaves (eviction); `Adapter.KnownPeerIDs()` returns a clone behind `atomic.Pointer[[]uint16]`, republished from `mirrorStats` only when the generation moved. `Adapter.KnownPeers()` stays for the metrics surface, `ServerOptions.KnownPeers` is gone, and `main` wires `WritePeers: adapter.KnownPeerIDs`.
- **Full gates green on the committed tree:** repo-wide strict `golangci-lint` exit 0, and the whole `go test -race -count=1 ./...` suite exits 0 (app/repl 131.3s, grpcapi 3.4s, udprepl 1.4s, stand 10.0s, cmd 11.7s).

## Task Commits

1. **Task 1 (RED): the four failing write-time tests plus the mechanical notification-shape ripple** — `b93d35a` (test). The tests, the harness (`notifyCapture`, `testServerConfig.writePeers`, `fakeTracker.WritePeers`), every test call site in four packages, and the production scanning of the new seams (LocalNotify peers, WritePeers, adapter queue field, core parameter, PeerIDs/PeerGeneration, simulator and main wiring) that the new assertions need to compile. The ripple deliberately kept the old zero-peer/nil-seam answers, which the tests then rejected.
2. **Task 1 (GREEN): the honest report and the exact promise set** — `fdc7b60` (feat). `waitForReplication` consumes the captured slice and answers `(false, 0, 0)` for nil seam / zero peers; Set and Delete pass the captured slice to the wait; the core intersects the capture with the live table and emits `WaitSetEmpty` when the capture is already stale; docs updated (gate.go's `WaitReplicated` contract, the adapter's release-point reading, the integration test's degenerate-path note).

**Plan metadata:** the `docs(06-02)` completion commit (this commit).

## Files Created/Modified

- `internal/grpcapi/gate.go` — `ServerOptions.KnownPeers func() uint64` replaced by `WritePeers func() []uint16`; the `WaitReplicated` contract now states the nil-seam `(false, 0, 0)`.
- `internal/grpcapi/wait.go` — `waitForReplication` takes the captured `peers`, derives `total` from it, and answers the honest false for nil seam / zero peers while keeping the release call; new `writePeers` clone helper.
- `internal/grpcapi/server.go` — `LocalNotify` carries `peers`; `Set`/`Delete` capture once before the engine call and pass the slice to both notify and wait; `notifyLocalWrite` gained the parameter; `notifyReplayedWrite` passes nil.
- `internal/udprepl/repl.go` — `localWrite.peers`, cloned at enqueue; event loop passes them to the core; `peerIDs atomic.Pointer[[]uint16]` + `peerGen` mirror published from `mirrorStats`; new `KnownPeerIDs`; `WaitReplicated`'s total==0 doc explains the release-vs-report split.
- `internal/app/repl/core.go` — `peerGen` field and bumps; `PeerIDs()`/`PeerGeneration()` accessors; `NotifyLocalWrite`/`trackLocalWrite` take the captured slice and seed the wait set by intersection; `WaitSetEmpty` for a stale non-empty capture; `NotifyReplayedWrite` documents its unchanged peer-table seeding.
- `internal/app/internal/simnet/simnet.go` — the three direct core notifications pass the node's sorted `PeerIDs()`.
- `cmd/sup-kv/main.go` — `WritePeers: adapter.KnownPeerIDs` (the metrics source keeps `KnownPeers`).
- Test files — `internal/grpcapi/{wait_test.go,server_test.go,bench_test.go,wait_integration_test.go}`, `internal/udprepl/{repl_test.go,repl_internal_test.go,smoke_test.go,three_node_test.go,scenario_test.go}`, `internal/app/repl/{bench_test.go,retransmit_test.go}`.
- `.planning/phases/06-api-and-semantics/red-evidence/06-02-task1.json` — the immutable RED record (command, exit 1, full console output, target test, expected/actual, semantic assessment, `RED_EVIDENCE_OK`).

## Decisions Made

- **The signature ripple lives in the RED commit, by necessity.** The four assertions reference `WritePeers`, the peer-carrying `LocalNotify` and the recorded peers; without the sweep the package does not compile and the RED classifier refuses a `[build failed]` run. The sweep was therefore written to preserve the old behaviour exactly, and the RED record shows all four targets failing on their assertions (`RED_EVIDENCE_OK`, `target_test_failed`), not on build output.
- **The adapter's zero-peer release keeps its `(true, 0)` verdict.** Ownership of the public report sits in the handler; the seam call exists to delete the notify-time tracker (T-06-05). Its doc now says so explicitly, so a future reader cannot mistake the release for an acknowledgement.
- **`Core.NotifyReplayedWrite` is deliberately outside the promise.** It keeps its signature and seeds from `c.PeerIDs()` — identical to the previous peer-table seeding — so snapshot replay tracking is unchanged; a replay has no waiter to promise anything to.
- **The discovery flush stays as it is.** A peer discovered after acceptance still joins every pending wait set (the IN-05 hole) until plan 06-11 splits promise-set enforcement from broadcast fan-out. This plan makes the *reported total* honest; 06-11 makes the *wait set* exact.
- **The peer mirror is generation-gated, not per-event.** `PeerGeneration` moves only on an id entering or leaving the table, so an unchanged peer table costs no `slices.Sorted` clone in the event loop — the same trade the listing mirror already makes.
- **The stale plan-head ledger was corrected.** `.git/gsd-plan-head-before-06-02` (dated 2026-11-16, commit `03f2173`) pointed 405 commits back into the v1.0 phase-6 history; it was reset to the pre-plan HEAD `919a471` before measuring, so `actuals.commits: 2` is the measurement for this plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The new assertions could not compile without the notification-shape ripple**
- **Found during:** Task 1's RED step
- **Issue:** the plan's own scope note calls the 19-file count "the mechanical ripple of one atomic signature change", but a TDD RED run cannot include a build failure — the classifier refuses `[build failed]` runs outright. The four tests had to compile against a tree that already exposed the new seams.
- **Fix:** the ripple landed in the RED commit, written so the old zero-peer/nil-seam answers stayed in place (the handler read `WritePeers` at wait time; the wait answered `acked=true`). The RED run then failed all four targets on their planned assertions.
- **Files modified:** the 18 modified paths listed above.
- **Verification:** `gsd_run check tdd-red-evidence ... --raw` → `RED_EVIDENCE_OK`, reason `target_test_failed`, 4 tests / 4 failing; semantic assessment recorded in the RED JSON.
- **Commit:** `b93d35a`

**2. [Rule 1 - Bug] An unused `//nolint:gosec` in the new test tripped `nolintlint`**
- **Found during:** the repo-wide lint run after the GREEN edits
- **Issue:** `uint16(i + 1)` in the test's peer generator does not trigger G115 in this configuration, so the named suppression was flagged as unused (`nolintlint`), failing `make lint`'s strict contract on the RED commit's tree.
- **Fix:** the directive was removed; the correction was amended into the RED commit (`git commit --amend --no-edit`) so each plan commit lints clean on its own.
- **Files modified:** `internal/grpcapi/wait_test.go`
- **Verification:** `go tool golangci-lint run ./...` exit 0.
- **Commit:** `b93d35a` (amended)

**3. [Rule 1 - Bug] `roadmap update-plan-progress 06` rewrote the v1.0 phase-6 row again**
- **Found during:** the metadata close-out, when reading the resulting `ROADMAP.md`
- **Issue:** the tooling defect 06-01 logged recurred exactly as predicted: the command matched the **v1.0** phase-6 row (`6. Hardening …`) and rewrote it as `2/13 | In Progress`, while the v1.1 `6. API and Semantics` row stayed at `1/13`. The v1.1 section's own "Plans: 2/13" line and the 06-02 checklist entry were updated correctly.
- **Fix:** the v1.0 row was restored byte-for-byte (`9/9 | Complete | 2026-11-16`) and the v1.1 row set to `2/13 | In Progress`; the defect stays recorded in the phase's `deferred-items.md` from 06-01.
- **Files modified:** `.planning/ROADMAP.md`
- **Verification:** `git diff .planning/ROADMAP.md` shows only the v1.1 "Plans: 2/13", the 06-02 checkbox and the v1.1 phase-6 row.
- **Commit:** the `docs(06-02)` metadata commit

## TDD Gate Compliance

- **RED:** `b93d35a` — `test(06-02): add failing write-time capture tests`. Evidence `.planning/phases/06-api-and-semantics/red-evidence/06-02-task1.json`, verdict `RED_EVIDENCE_OK` (target `TestWait_zeroPeers`, 4 tests, 4 failing).
- **GREEN:** `fdc7b60` — `feat(06-02): report the write-time peer set honestly`. The four target tests pass, and the whole existing suite stays green.
- **REFACTOR:** none needed — no cleanup commit was made, and the GREEN tree is the committed tree.

## Issues Encountered

- **One anomalous `timeout` kill, not reproduced.** The first attempt to run `go test -race ./internal/app/repl/ -run '^TestScenario_'` hit the 570 s outer timeout once; the same command then completed in 1.9 s, the chaos suite in 1.6 s, and the full package in 129 s. No failing or hanging test was observed; the full `go test -race ./...` run is green. Recorded here rather than silently dropped.
- **The plan's `files_modified` lists `internal/udprepl/bench_test.go`, which needs no edit.** A grep for `NotifyLocalWrite`/`NotifyReplayedWrite` finds no call site there; the package's benchmark path reaches the adapter through the shared helpers. No change was invented to match the list.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 06-03.** The write half of `wait_for_replication` is honest end to end and the tree is fully green (`go test -race ./...`, strict lint, build and vet).
- **Plan 06-11 owns the rest of A-01.** Tracker promise-set enforcement (counting Acks only for captured ids), the discovery-flush split, and the regression matrix are explicitly out of scope here; the IN-05 hole in the wait set remains until then, while the reported total is already pinned.
- **Plan 06-03 (client example) and Phase 7 (metrics)** can rely on `Adapter.KnownPeerIDs` being the write-path seam and `Adapter.KnownPeers` the metrics seam.

## Self-Check: PASSED

- Created file exists: `.planning/phases/06-api-and-semantics/red-evidence/06-02-task1.json` — FOUND.
- Task commits are ancestors of HEAD: `b93d35a`, `fdc7b60` — both FOUND.
