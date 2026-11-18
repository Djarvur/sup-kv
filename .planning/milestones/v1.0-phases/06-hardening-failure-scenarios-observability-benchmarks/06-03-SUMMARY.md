---
phase: 06-hardening-failure-scenarios-observability-benchmarks
plan: 03
subsystem: observability-logging
tags: [slog, structured-logging, observability, discard-classification, metrics-seams, udprepl, repl-core, persist-engine]

requires:
  - phase: 05-grpc-api-and-binary
    provides: cmd/sup-kv config/exit-code contract (SUPKV_ env, flag>env>default, exitUsage 2) and the exec smoke harness
  - phase: 04-replication-and-deterministic-simulator
    provides: repl core discard classification + DiscardStats, udprepl adapter atomic-mirror pattern, EncodeDatagramForTest seam
provides:
  - OBS-03 complete — slog JSON-on-stderr pipeline with env-controlled level/format (SUPKV_LOG_LEVEL/SUPKV_LOG_FORMAT, malformed values exit 2), the udprepl Log tap as a static-msg slog bridge, and every discard class a distinguishable structured event
  - OBS-01's full counter layer for 06-04's exporter — repl.Core MergeStats/Retransmissions, persist.Engine DumpStats, udprepl Adapter UnackedStats/SentByType/MergeStats/Retransmissions mirrors
  - repl.Config.OnDiscard seam with the five stable class tokens (bad_crc, undecodable, bad_proto_version, own_broadcast, duplicate_node_id)
affects: [06-04 metrics exporter, OBS-01/OBS-02 families, phase verification]

actuals:
  tokens: 16900   # chars/4 over the realized diff (67715 diff chars across 11 files)
  tasks: 3
  commits: 3      # MEASURED: git rev-list --count eaf88c0..HEAD
plan_head_before: eaf88c00be5f2316accd48bbb8daad8a08562f99
plan_head_after: 3ff81db163b6370a86f2e90d2a49645185425a06

tech-stack:
  added: []        # zero new dependencies — stdlib log/slog only (D-06); go mod tidy byte-identical
  patterns:
    - "one handler install in main before any service goroutine; slog.SetDefault's log-package redirect keeps the remaining log.Printf calls in the same format — every stderr line of a running node is one handler-formatted record"
    - "adapter-owned default seam: udprepl.New replaces a nil OnDiscard with its own slog bridge, so discard events surface with zero main wiring and stay testable at the adapter seam"
    - "Debug-level discard events as the flood bound (T-6-06): default info suppresses a bad-CRC flood from stderr while the O(1) counters carry the reliable signal"
    - "source-handle-filtered record capture for parallel-safe slog.SetDefault assertions — other adapters' events ride the same default logger and are excluded by the detail attr"

key-files:
  created:
    - internal/app/repl/merge_stats_test.go
  modified:
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - internal/app/repl/core.go
    - internal/app/repl/discard_test.go
    - internal/app/repl/restart_test.go
    - internal/app/repl/retransmit_test.go
    - internal/app/persist/engine.go
    - internal/app/persist/engine_test.go
    - internal/udprepl/repl.go
    - internal/udprepl/repl_test.go

key-decisions:
  - "The OnDiscard-to-slog bridge lives in the ADAPTER (udprepl.New defaults a nil OnDiscard to logDiscard), not main — the plan allowed either; the adapter default makes production discard events reachable with zero wiring and provable at the adapter seam (TestDiscardLog_classes)"
  - "setupLogging runs after finalizeConfig and before wire, so the tap, the adapter's discard bridge and the redirected log-package calls all flow through the one handler; the exit-2 usage path stays plain text (CLI UX, not log output)"
  - "Discard-class slog events emit at Debug (T-6-06): the default info level suppresses a datagram-flood amplification into stderr; the adapter's O(1) Discards counters carry the reliable signal — documented at logDiscard"
  - "DriftWarnings counts clock.Update's own gated signal: the drift return is measured against the clock's pre-absorb l, so a far-ahead stamp warns on the NEXT apply, not its own arrival — TestMergeStats_counts pins the absorb-then-warn sequence"
  - "DumpStats records the Encode+Save span in real nanoseconds internally, exposed as float seconds — atomics keep it scrape-safe beside the serialized dumper"
  - "remoteRecord dropped its always-zero counter parameter (unparam, code-direction) — two pre-existing call sites updated with it"

requirements-completed: [OBS-03]   # OBS-01 declared by 06-03 AND 06-04 — shared-ID gate keeps it open until 06-04 finishes

coverage:
  - id: D1
    description: "slog pipeline end-to-end — SUPKV_LOG_LEVEL/SUPKV_LOG_FORMAT env knobs (malformed values exit 2 through the collected env-error path), JSON handler default on stderr, text via SUPKV_LOG_FORMAT=text, udprepl Log tap as a static-msg + detail-attr slog bridge"
    requirement: OBS-03
    verification:
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestLog_jsonDefault (every stderr line of a real two-node run parses as one JSON object with level+msg; the driven DuplicateNodeID event rides the detail attr under msg=Replication adapter)"
        status: pass
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestLog_formatText (key=value lines, no JSON objects, same detail-attr shape)"
        status: pass
      - kind: integration
        ref: "cmd/sup-kv/main_test.go#TestLog_envValidation (SUPKV_LOG_LEVEL=bogus and SUPKV_LOG_FORMAT=bogus both exit 2)"
        status: pass
    human_judgment: false
  - id: D2
    description: "OnDiscard seam (D-08) — repl.Config.OnDiscard nil-normalized, fired at every discard branch with the five stable class tokens and metadata details; adapter default bridges to distinguishable Debug-level slog events; core stays silent (D-09)"
    requirement: OBS-03
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_onDiscardSeam (seven-row table: token, detail, unchanged total classification, healthy datagram fires nothing)"
        status: pass
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestDiscardLog_classes (real-socket bad_crc and bad_proto_version events captured through slog.SetDefault at Debug level; zero records before any send)"
        status: pass
      - kind: unit
        ref: "TestDiscard_classes + TestDiscard_totality green unmodified (additions-only diff to discard_test.go)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Core counter seams — repl.Core MergeStats{Conflicts, DriftWarnings} fed from the previously discarded ApplyRemote returns, plus the retransmissions counter in scanRetransmit, behind copying accessors"
    requirement: OBS-01
    verification:
      - kind: unit
        ref: "internal/app/repl/merge_stats_test.go#TestMergeStats_counts (displacement counts 1; equal redelivery counts 0; post-absorption drift warning counts 1 with no conflict on the absent key)"
        status: pass
      - kind: unit
        ref: "internal/app/repl/merge_stats_test.go#TestRetransmit_counter (0 before the first due step, exactly 1 and 2 at the pinned 200ms/600ms ladder steps)"
        status: pass
    human_judgment: false
  - id: D4
    description: "persist.Engine DumpStats — last Encode+Save span (seconds), total dumps and errors, atomics-backed, recorded on success and failure"
    requirement: OBS-01
    verification:
      - kind: unit
        ref: "internal/app/persist/engine_test.go#TestDumpStats_records (success: duration>0, total 1, errors 0; injected fake Save failure: total 2, errors 1, engine stopped ahead of the re-armed retry)"
        status: pass
    human_judgment: false
  - id: D5
    description: "udprepl mirror seams — UnackedStats (count+oldest age), MergeStats, Retransmissions refreshed at the mirrorStats site, and SentByType per-kind counters in flush — everything 06-04's exporter reads"
    requirement: OBS-01
    verification:
      - kind: integration
        ref: "internal/udprepl/repl_test.go#TestMirror_unackedSent (asymmetric-socket write lifecycle: unacked observably 1 in suspension, drains to 0 on B's discovery+Ack; sent Hello/Data >= 1 on A, Ack >= 1 on B; B serves the record byte-equal)"
        status: pass
    human_judgment: false

duration: 28min
completed: 2026-11-16
status: complete
---

# Phase 06 Plan 03: Observability Seams and Structured Logging Summary

**slog JSON pipeline on the real binary (SUPKV_LOG_LEVEL/SUPKV_LOG_FORMAT, malformed env exits 2, udprepl tap as a static-msg slog bridge) plus the complete OBS-01 counter layer 06-04's exporter reads — core MergeStats/retransmissions, engine DumpStats, adapter UnackedStats/SentByType mirrors — with every discard class a distinguishable OnDiscard token bridged to Debug-level structured events, all through tested accessors and zero new dependencies.**

## Performance

- **Duration:** 28min
- **Started:** 2026-11-15T22:37:47Z
- **Completed:** 2026-11-15T23:06Z (UTC)
- **Tasks:** 3/3
- **Files modified:** 11 (1 created)

## Accomplishments

- **OBS-03 (Task 1, tracer)**: `setupLogging` installs one slog handler on stderr before any service goroutine — `slog.NewJSONHandler` by default, `slog.NewTextHandler` when `SUPKV_LOG_FORMAT=text`, at the `SUPKV_LOG_LEVEL`-resolved level; `slog.SetDefault`'s log-package redirect pulls the remaining `log.Printf` calls into the same format, so every stderr line of a running node is one handler-formatted record. The udprepl Log tap became `slog.Warn("Replication adapter", "detail", msg)` — sloglint-clean static msg, dynamics in attrs. Tracer feedback gate re-ran verify end-to-end under auto mode: PASS.
- **D-08/D-09 (Task 2)**: `repl.Config.OnDiscard func(class, detail string)` (nil-normalized beside Log) fires at every HandleDatagram discard site with exported tokens `bad_crc`/`undecodable`/`bad_proto_version`/`own_broadcast`/`duplicate_node_id` and a metadata detail; the duplicate-node_id Log calls stay (additive seam). `udprepl.New` defaults a nil OnDiscard to its own `logDiscard` slog bridge at Debug level (T-6-06 flood bound). No slog import anywhere under `internal/app`.
- **OBS-01 counter layer (Task 3)**: `repl.Core.MergeStats()` (Conflicts from the previously-discarded ApplyRemote return, DriftWarnings from the clock's gated nonzero drift signal) and `Retransmissions()`; `persist.Engine.DumpStats()` (last span seconds, total, errors — recorded on both runDump paths); `udprepl.Adapter` mirrors `UnackedStats()`/`MergeStats()`/`Retransmissions()` at the mirrorStats site and counts `SentByType()` in flush. All copy-on-read, refreshed post-event like `Discards()`.

## Task Commits

1. **Task 1 (tracer): slog pipeline end-to-end** — `d9a0a43` (feat) — tracer feedback gate: verify re-ran end-to-end (3/3 PASS, no "no tests to run")
2. **Task 2: OnDiscard seam + discard-class log events** — `34b089f` (feat)
3. **Task 3: counter seams — core/engine/adapter** — `3ff81db` (feat)

**Plan metadata:** (this commit) `docs(06-03): complete observability seams plan`

## Files Created/Modified

- `cmd/sup-kv/main.go` — env consts, config fields, resolveLogKnobs, setupLogging, the slog tap bridge, envSlogLevel/envLogValue + errLogFormatEnv sentinel
- `cmd/sup-kv/main_test.go` — outputCapture/logBuffer live stderr seam, launchNode refactor, logProbeSocket/probeLoopbackBroadcast/startLogPair, TestLog_jsonDefault/formatText/envValidation
- `internal/app/repl/core.go` — Discard* token constants, Config.OnDiscard + normalization, OnDiscard at all six discard sites, MergeStats type + counters at applyRemote, retransmitted in scanRetransmit, three new accessors
- `internal/app/repl/discard_test.go` — TestDiscard_onDiscardSeam (additions only; existing suites untouched)
- `internal/app/repl/merge_stats_test.go` — NEW: TestMergeStats_counts, TestRetransmit_counter
- `internal/app/repl/restart_test.go`, `internal/app/repl/retransmit_test.go` — remoteRecord signature (unparam lint-direction fix, two call sites)
- `internal/app/persist/engine.go` — DumpStats type + atomics, runDump span measurement, recordDump, DumpStats accessor
- `internal/app/persist/engine_test.go` — TestDumpStats_records
- `internal/udprepl/repl.go` — logDiscard default bridge in New, OBS-01 mirror atomics + accessors, countSent in flush
- `internal/udprepl/repl_test.go` — discardCapture handler, TestDiscardLog_classes, probeLoopbackBroadcast, newAsymAdapter, TestMirror_unackedSent

## Decisions Made

See key-decisions frontmatter; the load-bearing one is the adapter-owned OnDiscard default (the plan's "adapter or main" latitude resolved toward the adapter so the seam is provable without main in the loop).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Lint] registerConfig funlen + first-slog lint shapes**
- **Found during:** Task 1 lint gate
- **Issue:** The two env resolutions inline pushed registerConfig past funlen's 40-statement limit; err113 flagged the dynamic format error; godoclint/noinlineerr/godox fired on the new code shapes.
- **Fix:** Extracted `resolveLogKnobs` (the registerConfig fail contract, stated in its doc); `errLogFormatEnv` static sentinel wrapped with %w; `[log.Printf]`/`[slog.SetDefault]` link forms; plain error assignments; formatter run.
- **Files modified:** cmd/sup-kv/main.go, cmd/sup-kv/main_test.go
- **Verification:** go tool golangci-lint run ./cmd/sup-kv/... exit 0
- **Committed in:** d9a0a43

**2. [Rule 1 - Bug] The plan's drift-warning prose pinned the wrong event**
- **Found during:** Task 3 first TestMergeStats_counts run
- **Issue:** "a crafted record stamped far ahead of local time (beyond driftWarn) drives DriftWarnings" fails as written: clock.Update gates the drift signal against the clock's own pre-absorb l, and the earlier greater-apply had already raised l — the far-ahead record's OWN arrival can never warn; the warning lands on the NEXT apply after absorption.
- **Fix:** Test restructured to the honest absorb-then-warn sequence (far-ahead install: 0 warnings; equal redelivery of it: 1 warning), with the semantics documented in the test and MergeStats docs.
- **Files modified:** internal/app/repl/merge_stats_test.go, internal/app/repl/core.go (docs)
- **Verification:** TestMergeStats_counts green; full repl suite green
- **Committed in:** 3ff81db

**3. [Rule 3 - Blocker] EncodeDatagramForTest cannot stamp a foreign proto_version**
- **Found during:** Task 2 test construction
- **Issue:** The plan's "proto-version-mismatch datagram via EncodeDatagramForTest/DecodeDatagramForTest-shaped bytes" is not expressible — the helper always stamps SupportedProtoVersion, and the core's version check only runs past a CRC-passing envelope.
- **Fix:** TestDiscardLog_classes hand-builds the v2 envelope through the generated replproto types with a correct CRC (the classification-order dependency made explicit); the bad-CRC datagram stays the corrupted-byte EncodeDatagramForTest shape.
- **Files modified:** internal/udprepl/repl_test.go
- **Verification:** TestDiscardLog_classes green (bad_crc and bad_proto_version both captured)
- **Committed in:** 34b089f

**4. [Rule 1 - Lint] remoteRecord's always-zero counter parameter (unparam)**
- **Found during:** Task 3 full-repo lint gate
- **Issue:** Every call site (including the two pre-existing ones) passes counter 0; my new calls tipped unparam over.
- **Fix:** Parameter dropped (code direction, the 06-01/02 precedent); two pre-existing call sites updated in the same commit.
- **Files modified:** internal/app/repl/retransmit_test.go, internal/app/repl/restart_test.go, internal/app/repl/merge_stats_test.go
- **Verification:** full lint exit 0; full race suite green
- **Committed in:** 3ff81db

---

**Total deviations:** 4 auto-fixed (2 Rule 1 lint/bug, 1 Rule 1 plan-prose bug, 1 Rule 3 blocker; none architectural — no Rule 4 checkpoints)
**Impact on plan:** None on scope — every artifact, test name and API the plan pinned landed as specified; one test's internal sequence was corrected to the clock's actual (documented) drift semantics.

## Issues Encountered

None beyond the deviations above. No auth gates (pure in-repo work).

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — every seam ships with a binding test; no pinned suite was modified (discard_test.go diff is additions-only).

## Threat Flags

None beyond the plan's own register — T-6-05's attr surface is asserted by TestLog_jsonDefault/TestDiscardLog_classes (details carry handles and node ids only, never record values or keys), and T-6-06's flood bound is the Debug-level default documented at logDiscard and enforced by the capturing handler's Debug filter.

## Next Phase Readiness

- 06-04's exporter reads exactly the accessors this plan pinned: `Adapter.Discards/KnownPeers/UnackedStats/MergeStats/Retransmissions/SentByType` and `Engine.DumpStats` — nothing left to discover.
- The asymmetric-topology probe + `newAsymAdapter` helper (repl_test.go) and the cmd-level `probeLoopbackBroadcast`/`startLogPair` are reusable precedents for 06-05/06-07's socket and exec scenarios.

## Self-Check: PASSED

- key-files on disk: 11/11 FOUND (merge_stats_test.go created; ten modified)
- Commits reachable from HEAD: d9a0a43, 34b089f, 3ff81db FOUND (merge-base --is-ancestor ok)
- evaluation-scope --plan 06-03 --commits-only: resolved, 3 commits
- Task acceptance criteria re-run: PASS — Task 1 (tidy byte-identical, env contract via TestLog_envValidation, sloglint-clean lint scope exit 0, jsonDefault parses every line as JSON with the detail event); Task 2 (OnDiscard nil-normalized, six firing sites + five tokens in a seven-row table, real-socket class attrs at Debug, TestDiscard_classes/totality green on an additions-only diff, no slog/net/os/syscall imports added to internal/app); Task 3 (accessors + captured returns, both DumpStats paths, copy-on-read mirrors, TestMirror_unackedSent lifecycle, full go test -race ./... and lint exit 0, direct requires unchanged at 2)
- Plan-level verification re-run: V1 3/3, V2 4/4, V3 1/1, V4 5/5 test PASS counts; full race suite 0 failures; full lint OK; go mod tidy byte-identical
- Regression gate: this repo's single test command (go test -race ./...) covers every prior phase's suites — run green twice this plan (post-Task-1 and final)
- Windows ledger: no stubs, skipped tests, or unrun verifies to record

---

*Phase: 06-hardening-failure-scenarios-observability-benchmarks*
*Completed: 2026-11-16*
