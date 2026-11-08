---
gsd_state_version: "1.0"
milestone: v1.0
current_phase: 05
current_phase_name: gRPC API and Binary
status: executing
stopped_at: Phase 5 context gathered
last_updated: "2026-11-08T19:03:38.504Z"
last_activity: 2026-11-07
last_activity_desc: Phase 4 complete, transitioned to Phase 5
state_head: 7d609542820d043de2eaae3c52ce6165df5bd242
progress:
  total_phases: 7
  completed_phases: 4
  total_plans: 24
  completed_plans: 18
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-11-07)

**Core value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.
**Current focus:** Phase 5 — gRPC API and Binary

## Current Position

Phase: 05 (gRPC API and Binary) — READY TO EXECUTE
Plan: Not started
Status: Ready to execute
Last activity: 2026-11-07 — Phase 4 complete, transitioned to Phase 5

Progress: [██████░░░░] 57%

## Performance Metrics

**Velocity:**
- Total plans completed: 18
- Average duration: 10min
- Total execution time: ~1 hour (60 min of plan execution)

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 02 | 4 | - | - |
| 01 | 6 | - | - |
| 3 | 2 | - | - |
| 4 | 6 | - | - |

**Recent Trend:**
- Last 5 plans: -
- Trend: -

*Updated after each plan completion*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P01 | 10min | 3 tasks | 10 files |
| Phase 01 P02 | 12min | 3 tasks | 2 files |
| Phase 01 P03 | 6min | 3 tasks | 2 files |
| Phase 01 P04 | 6min | 3 tasks | 2 files |
| Phase 01 P05 | 18min | 2 tasks | 2 files |
| Phase 01 P06 | 8min | 2 tasks | 5 files |
| Phase 02 P01 | 31min | 3 tasks | 7 files |
| Phase 02 P02 | 17min | 3 tasks | 4 files |
| Phase 02 P03 | 14min | 2 tasks | 2 files |
| Phase 02 P04 | 8min | 2 tasks | 5 files |
| Phase 03 P01 | 35min | 3 tasks | 12 files |
| Phase 03 P02 | 16min | 2 tasks | 4 files |
| Phase 4 P01 | 21 min | 1 tasks | 11 files |
| Phase 4 P02 | 19 min | 3 tasks | 3 files |
| Phase 04 P03 | 19 min | 3 tasks | 5 files |
| Phase 04 P04 | 9 min | 2 tasks | 2 files |
| Phase 04 P05 | 24 min | 3 tasks | 4 files |
| Phase 4 P06 | 15 min | 2 tasks | 6 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: phase order is dependency-forced (engine → persistence → replication → API); simulator lands in Phase 4 with replication, not in hardening (research-validated refinement)
- Roadmap: property tests (TEST-01/02) live in Phase 2 with the engine; failure suite (TEST-04) and benchmarks (TEST-05) in Phase 6
- Roadmap: strict lint config + depguard boundary rules land in Phase 1 before any feature code (QUAL-01)
- [Phase 01]: D-03 read as zero direct runtime requires; tool-directive graph's ~213 indirect lines documented in AGENTS.md (plan 01-01)
- [Phase 01]: Lint config vendored single-file: extends: silently no-ops in golangci-lint 2.15.0, so sup-kv depguard rules are additive keys inside linters.settings.depguard.rules with upstream byte-unmodified (plan 01-01)
- [Phase 01]: sup-kv-port-stdlib-only allows $gostd plus own module path github.com/Djarvur/sup-kv so port-to-app type references stay legal (plan 01-01)
- [Phase 01]: Strict-gate findings fixed in code direction: app_test.go external app_test package with t.Parallel(); never weaken the vendored config (plan 01-01)
- [Phase 01]: PR-only landing for code and docs per CI-03: plan 01-01 code via PR #1, SUMMARY via docs PR; no direct master commits (plan 01-01)
- [Phase 01]: CI check-run contexts fixed verbatim as lint, gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15), gate (darwin-amd64-cross) — plan 04 ruleset required_status_checks must copy these exactly (plan 01-02)
- [Phase 01]: go.sum completed with 54 tidy-required entries after the CI tidy gate caught the gap; go.mod unchanged, tidy idempotent under GOTOOLCHAIN=go1.27.1 — fix direction: complete module files, never weaken the tidy gate (plan 01-02)
- [Phase 01]: vulncheck triggers fixed: weekly cron 0 6 * * 1 (Mon 06:00 UTC) + pull_request paths go.mod/go.sum + workflow_dispatch; scan runs go tool govulncheck (pinned v1.9.0, never a latest-tag run) (plan 01-03)
- [Phase 01]: Dependabot groups minor+patch weekly (gomod-safe-updates, actions-safe-updates) so the small tool graph yields one PR per week; alerts are GHAS-gated on this private repo and stay off until the deferred public flip (plan 01-03)
- [Phase 01]: Dispatched vulncheck run 41216077968 green on master at a893d9c — 'Your code is affected by 0 vulnerabilities'; CI-02's assertable half proven live (plan 01-03)
- [Phase 01]: Plan 01-04 Task 2 decision auto-selected 'defer' (gate=blocking, auto_select honored): no org plan upgrade now — rulesets API is 403-plan-gated on this private repo; payload + one-command re-apply recorded in docs/setup/branch-protection.md; CI-03 protection sub-item flagged plan-gated, not hidden
- [Phase 01]: Ruleset required_approving_review_count=0 (A5 resolved): GitHub forbids PR authors approving their own PRs, so any count >= 1 deadlocks the solo maintainer; require_last_push_approval and required_review_thread_resolution false for the same reason; PR-only contract still enforced by the pull_request rule (plan 01-04)
- [Phase 01]: delete_branch_on_merge=true applied via gh api PATCH and verified live (ordinary repo setting, not plan-gated); CI-03 deliberately left Pending in REQUIREMENTS.md with Partial annotation — PR-only is practiced (PRs #1-#10) but server-side protection is deferred (plan 01-04)
- [Phase 01]: Gap deferrals recorded as accepted overrides with must_have/reason verbatim from the verifier's own suggestion block, ratified 2026-10-31, attributed accepted_by "onokonem" — the verifier re-run reads the overrides: mechanism and passes gaps 1-2 as accepted deviations (plan 01-05)
- [Phase 01]: CI-02 traceability honestly Partial (unchecked, alerts half deferred GHAS-gated, ratified) mirroring CI-03's pattern; CI-02/CI-03 deliberately NOT marked complete — plan 01-05 records accepted deviations, it does not deliver the deferred halves (plan 01-05)
- [Phase 01]: Rule 3 deviation — orphan STATE.md commit 3ac4b3f (planner's, outside PR flow) routed through PR #16 before task branches so each task PR diff stayed single-file per acceptance criteria (plan 01-05)
- [Phase 01]: Verification gap 1 ratified 2026-10-31 (maintainer, accepted override): CI-03 server-side branch protection stays deferred — org plan 403 gates rulesets; branch-protection.md marker now DEFERRED (accepted override) with the re-apply path kept; WINDOWS.md item 2 waived (gap plan 01-06)
- [Phase 01]: Verification gap 2 ratified 2026-10-31 (maintainer, accepted override): CI-02 alerts half stays GHAS-gated off until GHAS license or the deferred public flip; updates + scheduled govulncheck delivered and green; REQUIREMENTS.md CI-02 annotated Partial mirroring CI-03 (gap plan 01-05)
- [Phase 01]: Plan 01-06 records ratified deferral markers only — CI-02/CI-03 deliberately not marked complete in REQUIREMENTS.md (Partial stands); requirements-completed empty (plan 01-06)
- [Phase 02]: Cold start: NewClock seeds l from initialL only (0 defers to the wall clock on the first Tick), so the first write stamps counter 0 (plan 02-01)
- [Phase 02]: Store write paths hold the write lock across Tick/Update so version order equals commit order; Clock keeps its own mutex (Pitfall 7) (plan 02-01)
- [Phase 02]: Clock.Update masks the remote counter to 12 bits, making Merge total over any record content (plan 02-01)
- [Phase 02]: Merge installs remote records with a recomputed checksum instead of trusting the wire field (plan 02-01)
- [Phase 02]: RecordChecksum flag byte via tombstoneFlag.checksumByte method: code-direction lint fix, zero nolint in internal/app (plan 02-01)
- [Phase 02]: Plan 02-02: zero clock.go fixes — the 02-01 implementation held on every Kulkarni branch, the 4096/4097 boundary, both drift signs and all six TEST-02 properties
- [Phase 02]: Plan 02-02: fake clocks build time with time.UnixMilli; the plan prose's time.Unix(ms, 0) reads ms as seconds (Pitfall 8 applied to the plan text)
- [Phase 02]: Plan 02-02: restart property reads strictly-greater-than-every-dump-version as older-l or (dumpMax, 0); counters >= 1 at the watermark l are unreachable from a cold start given the locked (dumpMax, 1) first tick
- [Phase 02]: Plan 02-02: temporary written-reason nolint:unused bridged task 2's lint gate for generators consumed in task 3 / plan 02-03; all removed once call sites landed, zero nolint remains
- [Phase 02]: Plan 02-02: concurrent Tick/Update hammer added as a TestClock_monotonic subtest (t.Run adds no test function), satisfying the VERS-03 concurrency probe
- [Phase 02]: Plan 02-03: zero store.go fixes — the 02-01 Merge held every law (commutativity, associativity, idempotence, exact loser pairing) over 1000 seeded triples plus 500 loser pairs
- [Phase 02]: Plan 02-03: atomicNow uses time.UnixMilli over an atomic Int64 ms counter — the plan prose's time.Unix(ms, 0) repeated the Pitfall 8 seconds trap, locked 02-02 correction applied
- [Phase 02]: Plan 02-03: hammer pre-seeds every key so the every-key-present coherence assertion is reachable (Delete/SetIfAbsent never create); duplicate-delivery no-op asserted for the equal-version case, a re-delivered loser legitimately re-reports (CORE-07)
- [Phase 02]: Plan 02-04: initialC parameter over dumpMax+1 — the restart seam expresses an exact restart (constructor state equals the dump's full watermark); a dumpMax+1 convention stays unenforceable at NewClock level and discards the counter (WR-01 mechanism)
- [Phase 02]: Plan 02-04: Tick/Update stayed byte-identical — the seeded state routes the first Tick through the existing increment, counterMax overflow and wall-ahead paths, so the 02-02 Kulkarni proofs carry over unmodified
- [Phase 02]: Plan 02-04: WR-01 dominance proofs are mutation-checked regression locks — stripping the seed reproduces the verifier's probe failures (tick 1 subordinate, peer revert, equal-version drop), proving the tests bind on the initialC seam
- [Phase 03]: Parse-oracle Load settled per D-02-literal recommendation (A6/Open Q1): engine hands codec.Parse to the adapter, which stays record-blind (plan 03-01)
- [Phase 03]: Dump/snapshot files keep CreateTemp 0600, directories 0700 — values privacy over non-owner readability; flagged for phase-verification ratification (plan 03-01)
- [Phase 03]: Dump failure retry cadence per A8: dirty re-arms from a fresh epoch, debounce timer starts the next attempt one interval later, never a tight loop (plan 03-01)
- [Phase 03]: Engine cover discipline (PERS-04): lastLocal captured under mu before Records(); a waiter resolves only from a dump whose snapshot contains its record — confirmation means on-disk containment (plan 03-01)
- [Phase 03]: Five documented nolint suppressions where no code-direction fix exists: gosec G304 on the adapter's variable-path reads, iface on the port seam consumed cross-package (AGENTS.md sanctioned form) (plan 03-01)
- [Phase 03]: Snapshot flush semantics (plan 03-02): Run services flushCh with an immediate dump only when dirty — a clean engine replies nil without a redundant save; the exit sweep resolves stranded flush requests with ErrClosed
- [Phase 03]: Restore is semantics-level D-09 (plan 03-02): per-key ordinary store writes with fresh versions, one batched confirming flush — flagged assumption 1 for phase verification
- [Phase 03]: Snapshot overwrite atomically in place with no .prev of its own (plan 03-02) — flagged assumption 2 for phase verification
- [Phase 4]: Own-echo classification keys on run_id equality (impostor cannot guess 16 random bytes); same nodeID+different runID is the logged DuplicateNodeID config error (plan 04-01)
- [Phase 4]: Envelope-domain validation (node_id 16-bit, run_id exactly 16 bytes) classifies Undecodable before the CRC gate — research ASVS V5 field validation (plan 04-01)
- [Phase 4]: go mod tidy (go1.27) extracts a sole direct require into a single-line directive; CI tidy gate makes that the only shippable go.mod form, plan's tab-indented grep verified via tidy-stable equivalent (plan 04-01)
- [Phase 4]: Plan-prescribed nolint:iface dropped: iface does not fire on Message/StateStore and nolintlint forbids unused directives; one sanctioned gosec G115 nolint guards the range-checked uint16 node_id conversion (plan 04-01)
- [Phase 4]: Adapter NotifyLocalWrite enqueues into a 1024-deep buffer: never blocks on replication progress (D-05), bounded backpressure on a wedged loop, never silent loss (plan 04-01)
- [Phase 4]: Encode/decode stay unexported in udprepl; the external tracer test reaches them through exported test-binary-only helpers in repl_internal_test.go (dumpfile_internal_test.go precedent) (plan 04-01)
- [Phase 4]: A5 scan-granularity reading implemented and documented at scanRetransmit: retransmit_interval bounds Advance's returned sleep while any entry is pending; the per-key ladder doubles from ack_timeout and saturates at ack_timeout_max with every step (200/400/800/1600/3200ms then 5000ms repeated) spelled out in the scan's doc comment (plan 04-02) — The research open question resolved by this plan; bounds tests (04-05) phrase on ack_timeout_max either way, and the doc comment is the single documented home of the ladder
- [Phase 4]: Suspension is total: an empty peer table freezes nextRetry/backoff (never advanced); during suspension Advance's retransmit deadline keeps only the now+retransmit_interval ceiling — a stale past retry time never leaks into the deadline (plan 04-02) — Keeps the strictly-future deadline invariant while entries are suspended; caught live by TestRetransmit_suspendAndFlush's futurity assertion when the peer-timeout scan landed
- [Phase 4]: Flush-on-discovery fires only on the empty-to-non-empty table transition: the Hello ending an empty table broadcasts every pending entry, adds the peer to every wait-set and resets ladders; a Hello into a non-empty table (including a re-learned evicted peer) adds no waiters (plan 04-02) — Research A6 reading: brand-new peers need prompt catch-up, but re-learned peers regain coverage through the ladder's next fire — avoids wait-set churn on every periodic Hello
- [Phase 4]: UnackedStats takes now as a parameter (the D-02 seam) rather than reading a clock — a pure read inside the core; the earliest-peer-expiry zero time is guarded so an empty table never mins into the deadline (plan 04-02) — No wall-clock reads inside internal/app/repl (depguard-plus-discipline); the zero-guard is a regression lock from the Task-3 integration
- [Phase 04]: Egress-limiter scope pin implemented as ONE FIFO over all reactive unicasts (exactly three enqueue sites: DataRequest emission, DataRequest reply, DataSync-newer response; Hello/retransmits/DataSync never queue), spacing floor(1s/rate) integer nanoseconds — 2ms at 500/s, 3333333ns at 300/s — first response immediate, drain order equals enqueue order (plan 04-03, probe #16, T-04-07)
- [Phase 04]: Chunk budget is enforced by conservative core-side estimate constants (entryOverheadBytes 24, chunkOverheadBytes 64) because protobuf is adapter-side (D-01); the real-codec guarantee is proven cross-seam by udprepl's white-box TestSync_chunkWireBudget — a codec change that shrinks the slack breaks the test, not production (plan 04-03)
- [Phase 04]: Startup emission order Hello-then-DataSync pinned by TestSync_triggers (IDEA §Node lifecycle); plan 04-01's tracer startup assertion updated for the designed startup DataSync — Rule 1 prior-plan test adjustment, the 04-02 eviction/ladder precedent (plan 04-03)
- [Phase 04]: The egress queue machinery landed in Task 1's GREEN (the plan sketched it under Task 3) because Task 1's reply-drain assertions require enqueue+drain; Task 3 then added the two DataSync-compare enqueue sites and pinned the spacing arithmetic (plan 04-03)
- [Phase 04]: [Phase 04]: REPL-07 trigger pinned as pure inequality with the remembered run_id: random 16-byte run_ids carry no order, so a stale return is just another change (entry flip plus one idempotent re-sync listing, single table row); in-flight Ack/DataReplication/DataSync from the old run process run_id-agnostically on (key, version) merits (plan 04-04, probe #19)
- [Phase 04]: [Phase 04]: The restart-targeted listing bypasses the egress limiter (proactive, once per detected change — the plan-04-03 egress-scope pin) while the replay traffic it solicits stays limiter-gated; a detected restart never re-adds wait-sets nor touches the ladder — reconciliation heals a restarted peer, retransmission does not (plan 04-04)
- [Phase 04]: [Phase 04]: The restart reaction reuses the plan-04-03 listing verbatim through the extracted listingChunks build (one snapshot, sorted, chunked) shared with syncListing — no second listing implementation, no new exported symbols (plan 04-04)
- [Phase 04]: One live timer per node in simnet's drive() via a generation counter: every drive supersedes the node's previously scheduled timer, else outstanding timers grow linearly with virtual time (quadratic no-op Advances over a long run) — 66 virtual seconds cost 9ms of CPU instead of 73ms, semantics and delivery trace unchanged (plan 04-05) — One live timer per node in simnet's drive() via a generation counter: every drive supersedes the node's previously scheduled timer, else outstanding timers grow linearly with virtual time (quadratic no-op Advances over a long run) — 66 virtual seconds cost 9ms of CPU instead of 73ms, semantics and delivery trace unchanged (plan 04-05)
- [Phase 04]: Simulator determinism kept total: deterministic per-node run ids (decimal-digit rendering, no crypto/rand anywhere), send-time fault draws through the one injected rng in a fixed draw order, (Time, Seq) insertion tie-breaks, registered-node slice iteration for broadcast fan-out (plan 04-05) — Simulator determinism kept total: deterministic per-node run ids (decimal-digit rendering, no crypto/rand anywhere), send-time fault draws through the one injected rng in a fixed draw order, (Time, Seq) insertion tie-breaks, registered-node slice iteration for broadcast fan-out (plan 04-05)
- [Phase 04]: D-06 flag-vs-env reconciliation delivered as pinned: SUPKV_SIM_SEED is the chaos replay hook (the only one that works when the test binary runs without -args), every run prints the copy-paste replay command, testutil's -seed flag remains the property-test hook (plan 04-05) — D-06 flag-vs-env reconciliation delivered as pinned: SUPKV_SIM_SEED is the chaos replay hook (the only one that works when the test binary runs without -args), every run prints the copy-paste replay command, testutil's -seed flag remains the property-test hook (plan 04-05)
- [Phase 04]: REPL-10 bounds asserted inclusively with negative probes; the loss bound models its loss episode as a directed time-window partition dropping the initial send plus the +200/+600ms ladder attempts and delivering +1400ms — flagged assumption #21's pinned window reading (plan 04-05) — REPL-10 bounds asserted inclusively with negative probes; the loss bound models its loss episode as a directed time-window partition dropping the initial send plus the +200/+600ms ladder attempts and delivering +1400ms — flagged assumption #21's pinned window reading (plan 04-05)
- [Phase 4]: [Phase 04]: The duplicate-node_id second variant keys on remembered-run_id equality plus a differing source handle, so run_id inequality is always the restart path — the classifier never pre-empts the plan-04-04 restart branch; both variants log a configuration error naming node_id and both handles (plan 04-06) — Two sources claiming one identity is a misconfiguration (REPL-08); a changed run_id is a restart (REPL-07) — the equality test keeps the branches disjoint by construction
- [Phase 4]: [Phase 04]: Engine.ApplyRemote = Merge + markDirty + the non-blocking changed nudge (an idle dumper parked in its select would never arm timers without the wake); no waiter, no flush, errors unchanged, REPL-03's memory-apply contract (plan 04-06) — markDirty alone mutates state nobody polls — remote merges on a clean engine would never reach disk, the Phase-3 reserved seam's exact gap
- [Phase 4]: [Phase 04]: No-waiter dump paths in synctest advance the bubble clock with synctest.Sleep — synctest.Wait returns without firing timers; waiter-based tests never hit this because their blocked calls hold the test goroutine until the dump resolves (plan 04-06) — The first persist GREEN run failed only on harness timing; pinned so future no-waiter timing tests use Sleep from the start

### Pending Todos

None yet.

### Blockers/Concerns

- Phase 3 (Persistence): open planning decision — macOS F_FULLFSYNC durability policy (verify in Phase 6 crash scenarios).
- Phase 5: D-08 macOS smoke outcome is empirical until the first macOS CI run (Linux green; macOS loud-skip path code-verified; UAT item 2 passed 2026-11-07).
- Phase 4 review: info-level findings IN-01 (write-only ownHandle dead state) and IN-02 (simnet Cluster.Write ignores store Set error) remain open by triage — see 04-REVIEW-DISPOSITION.md.

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-11-08T17:46:45.753Z
Stopped at: Phase 5 context gathered
Resume file: .planning/phases/05-grpc-api-and-binary/05-CONTEXT.md
