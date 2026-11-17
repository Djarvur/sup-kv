---
gsd_state_version: "1.0"
milestone: v1.0
current_phase: 7
current_phase_name: v1.0 — Docs, Examples and Release
current_plan: 3
status: executing
stopped_at: Completed 07-02-PLAN.md
last_updated: "2026-11-17T13:03:20.974Z"
last_activity: 2026-11-17
last_activity_desc: Phase 06 complete, transitioned to Phase 7
state_head: 0a0c25535a5e88df3a37ef5f64e5567d0f82d46b
progress:
  total_phases: 7
  completed_phases: 6
  total_plans: 38
  completed_plans: 35
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-11-11)

**Core value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.
**Current focus:** Phase 07 — v1.0 — Docs, Examples and Release

## Current Position

Phase: 7 (v1.0 — Docs, Examples and Release) — EXECUTING
Current Plan: 3
Total Plans in Phase: 5
Status: Executing — plan 07-01 complete
Last activity: 2026-11-17 — Plan 07-01 complete (option-b broadcast verdict recorded)

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 33
- Average duration: 10min
- Total execution time: ~1 hour (60 min of plan execution)

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 02 | 4 | - | - |
| 01 | 6 | - | - |
| 03 | 2 | - | - |
| 04 | 6 | - | - |
| 05 | 6 | - | - |
| 06 | 9 | - | - |

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
| Phase 05 P01 | 19 min | 3 tasks | 9 files |
| Phase 05 P02 | 16 min | 2 tasks | 5 files |
| Phase 05 P03 | 60min | 2 tasks | 6 files |
| Phase 05 P04 | 12min | 2 tasks | 4 files |
| Phase 05 P05 | 12min | 2 tasks | 7 files |
| Phase 05 P06 | 24min | 3 tasks | 5 files |
| Phase 06 P01 | 22min | 3 tasks | 9 files |
| Phase 06 P02 | 9min | 2 tasks | 7 files |
| Phase 06 P03 | 28min | 3 tasks | 11 files |
| Phase 06 P06 | 12min | 2 tasks | 2 files |
| Phase 06 P07 | 26min | 3 tasks | 2 files |
| Phase 06 P04 | 19min | 3 tasks | 4 files |
| Phase 06 P05 | 35min | 3 tasks | 3 files |
| Phase 06 P08 | 18min | 3 tasks | 8 files |
| Phase 06 P09 | 12min | 3 tasks | 3 files |
| Phase 07 P01 | 23min | 3 tasks | 8 files |
| Phase 07 P02 | 12min | 2 tasks | 3 files |

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
- [Phase 05]: Codegen rides buf (post-plan buf migration): buf.gen.yaml gained the go-tool protoc-gen-go-grpc plugin; Makefile proto target unchanged — one buf generate pass covers both proto dirs; per-file buf-lint waivers keep the plan-pinned wire names SupKV/WatchEvent (D-06 exception pattern)
- [Phase 05]: protobuf stays at the buf-migration pseudo-version v1.36.13-0.20261016172727-e9ee2a76ab8e (plan's v1.36.12 pin superseded by the buf baseline); grpc v1.85.0 MVS sits below it — zero version churn (plan 05-01)
- [Phase 05]: Delete notifies the tombstone through LocalNotify (API-05: a delete is a write; D-05 author contract) — Rule 2 addition beyond the task text, else deletes never replicate (plan 05-01)
- [Phase 05]: LocalNotify record rebuilt from the write's own inputs + RecordChecksum so the notified record equals the stored one; receivers recompute checksums on merge regardless (plan 05-01)
- [Phase 05]: mirrorState is one sibling call at the existing post-iteration stats site — sync flag and pending copy refresh together after every loop event, never torn (plan 05-02)
- [Phase 05]: handleAck writes the pending entry back to the map (value copy) after appending to acked; evictPeers deliberately never touches acked — eviction stays distinguishable from an Ack by construction (T-05-04, plan 05-02)
- [Phase 05]: udprepl pending mirror is copy-on-write: fresh map, every WaitingOn/Acked slice cloned, pointer replaced wholesale — published snapshots are immutable and stable across events (plan 05-02)
- [Phase 05]: write() routes through install so the storage OnChange tap truly has a single convergence point for every change source; the join law keeps D-06 decidable at one point (plan 05-03)
- [Phase 05]: WatchHub snapshot is built by the hub goroutine at subscribe and streamed by the handler from the subscribe reply — a snapshot larger than the subscriber buffer can never block the hub (plan 05-03)
- [Phase 05]: The disconnect status is written before the events channel close — the close happens-before the handler's zero-value receive carries the termination status with no locks (plan 05-03)
- [Phase 05]: TestWatch_slowConsumer floods hub.Ingest (the production OnChange entry) because gRPC flow control grants a stalled reader ~0.5-1 MiB of buffering — no test-scale write round can force the shed; the harness pins a static 64 KiB stream window (plan 05-03)
- [Phase 05]: Gate check is the first statement of Set/Delete — before empty-key validation (API-04 rejects writes during the gate, full stop); empty key surfaces Unavailable during the gate (plan 05-04)
- [Phase 05]: openGate is one CAS'd transition storing the gate flag and flipping health under the same branch — the resolver is the only opener, health bound 1:1 to the gate for the empty service name (T-05-08, plan 05-04)
- [Phase 05]: Snapshot RPCs gated exactly like writes (A5); names cross to persistfile untouched — its validateName is the single closed path-traversal control (T-05-07), the adapter only maps ErrInvalidSnapshotName to InvalidArgument (plan 05-04)
- [Phase 05]: Snapshot tests run against a real persistfile.FileStore (newFileServer) — the in-memory fake deliberately does not reproduce name validation, so a fake-backed test would prove nothing about T-05-07 (plan 05-04)
- [Phase 05]: Deterministic closed-gate observation: block the resolver's injected SyncArrived closure until the test observed the state — naive 'assert before the deadline' races the deadline under test under full-suite parallel load (plan 05-04)
- [Phase 05]: waitForReplication reads only the injected ServerOptions seams (KnownPeers/PendingCopy) — handler goroutines never touch the single-driver core; total captured once pre-poll (D-10) and zero peers short-circuit acked immediately (D-11), including the nil-seam single-node default (plan 05-05)
- [Phase 05]: The appearance phase (waitAppear 500ms cap, 5ms poll) closes the notify-queue race: absence is not drained until the (key,version) entry had its bounded chance to appear — Pitfall 4 locked by TestWait_appearanceRace (plan 05-05)
- [Phase 05]: Drain honesty reads counts only from the Acked seam: eviction (WaitingOn empty, Acked empty) reports false, supersession freezes counts at the last observation (Q2), deadline and ctx cancellation always report acked=false — the wait never changes the write outcome (plan 05-05)
- [Phase 05]: Proto wait-field comments made live and regenerated via buf: the stale accepted-but-deferred wording would misstate the now-live wire contract (plan 05-05)
- [Phase 05]: hub.Run(apiCtx) launched in wire(): the D-19 'ends Watch streams and the hub' step requires the hub goroutine running on apiCtx — without it Watch subscribe blocks forever (plan 05-06, Rule 3)
- [Phase 05]: Exit-code contract: exit 0 only after a confirmed final dump (D-19); failed Engine.Flush exits 1 (exitDumpFailed, Restart=on-failure reacts); missing/malformed config exits 2 (exitUsage) — malformed SUPKV_ env values are fatal, --version still works on a broken environment
- [Phase 05]: revive deep-exit reshaped main.go: every exit in main, registerConfig/serve/shutdown return errors (errors.Join for env mistakes); funlen split wire() into startEngine/wire/listenGRPC wiring helpers — semantics identical to the plan sketch
- [Phase 05]: Flush is export-by-delegation (unexported flush unchanged); gosec G115 nolints unnecessary on guarded uint64-to-uint16 conversions (directives removed as unused); sole remaining nolint is gochecknoglobals on commit (the -X ldflags mechanism, D-18)
- [Phase 06]: D-15 P4-WR-02: SyncResponseRate clamps to 500/s outside (0, 1e9] via maxSyncResponseRate — floor(1s/rate) stays non-zero, the egress limiter is never silently off (plan 06-01)
- [Phase 06]: D-15 P4-WR-01: the drained-row Acked union is sort-then-Compact — a stranger's Ack (counting behavior) plus a discovery-flush re-add legitimately puts one peer in both sets (plan 06-01)
- [Phase 06]: D-15 P4-IN-01: NotifyLocalWrite is one select over notify+done with a post-send re-check (the review's own alternative shape) — a bare single select loses ~50% of post-Run calls to the uniform choice and breaks the pinned 1100-call regression (plan 06-01)
- [Phase 06]: D-15 conditional dirty coded to Merge's Conflict contract: Winner=incoming is displacement, Winner=stored a rejection; the conflicts-nil ambiguity (new key vs equal redelivery) resolved by a Get pre-probe; equal-tombstone redelivery arms one redundant dump, the safe direction only (plan 06-01)
- [Phase 06]: D-15 Restore no-op decided WHOLE-snapshot, never per-record: the per-record content-skip broke the pinned fresh-version dominance contract (TestEngine_snapshotCreateLoadRestore) and was reworked before commit (plan 06-01)
- [Phase 06]: D-15 P3-WR-01: the above-cover guard is kept verbatim in intent plus an ADDED invariant assertion (kept waiters unresolved; resolved waiters never re-queued); white-box engine_internal_test.go follows the repl_internal_test.go precedent (plan 06-01)
- [Phase 06]: D-15 P3-IN-01: Open decodes through Load's own oracle argument — Parse IS Decode's error channel, so the accepted generation is exactly the decoded one; the dead branch is removed, not comment-tightened (plan 06-01)
- [Phase 06]: [Phase 06]: P1-IN-01: one armed signal channel (armSignals, buffer 2) spans serving and shutdown — serve consumes the first signal, shutdown selects the second as its force branch, signal.Stop runs in main only after shutdown returns (hurry-up semantics, Open Q4 recommendation b)
- [Phase 06]: [Phase 06]: P1-IN-01: the final Flush sits after GracefulStop in the D-19 sequence and therefore survives the force stop — the confirmed dump still lands and the node exits 0
- [Phase 06]: [Phase 06]: TestShutdown_secondSignal blocks the drain with a standard health Watch stream (not SupKV Watch, which apiCancel ends before GracefulStop by design) and observes NOT_SERVING on it as proof the D-19 sequence started before the second SIGTERM lands
- [Phase 06]: [Phase 06]: P1-IN-01 mutation check: re-moving signal.Stop to the pre-fix position reproduces the exact regression (signal kill, want exit 0) — the test binds on the fix
- [Phase 06]: [Phase 06]: P2-IN-03: the TestStore_Set copy of the duplicated subtest deleted, the TestStore_endToEnd copy kept byte-identical (endToEnd additionally asserts the unpacked l/counter/nodeID); cmpSame kept — still live in merge_test.go
- [Phase 06]: [Phase 06]: P1-WR-01: timeout-minutes 15/20 added to ci.yml jobs as additive keys only — zero check-run context diffs, the ruleset required_status_checks contract intact
- [Phase 06]: [Phase 06]: D-08/D-09: the OnDiscard-to-slog bridge lives in the ADAPTER (udprepl.New defaults a nil OnDiscard to logDiscard) — production discard events surface with zero main wiring and stay provable at the adapter seam; discard lines emit at Debug (T-6-06 flood bound) while the O(1) counters carry the signal (plan 06-03)
- [Phase 06]: [Phase 06]: setupLogging runs after finalizeConfig and before wire — slog.SetDefault's log-package redirect keeps every remaining log.Printf call in the same handler format, so every stderr line of a running node is one handler-formatted record; the exit-2 usage path stays plain text (plan 06-03)
- [Phase 06]: [Phase 06]: MergeStats.DriftWarnings counts clock.Update's own gated signal — the drift return is measured against the clock's pre-absorb l, so a far-ahead stamp warns on the NEXT apply, not its own arrival; TestMergeStats_counts pins the absorb-then-warn sequence (plan 06-03)
- [Phase 06]: 06-06 per-case isolated converged pairs in TestScenario_discardClasses — the vendored paralleltest/tparallel gates demand parallel subtests and per-class exactness must not share a mutable Discards baseline across concurrent subtests; scenarioConvergedPair is the shared convergence prologue TestScenario_convergence reuses (plan 06-06)
- [Phase 06]: 06-06 EncodeDatagramForTestVersion in repl_internal_test.go — the exported test-binary-only explicit-proto-version crafting seam with a production-faithful CRC (the receiver's CRC gate passes, the version check is the branch under test); the impostor pair rides plain EncodeDatagramForTest nodeID/runID parameters — production encode and New untouched (plan 06-06)
- [Phase 06]: 06-06 the Ack-drain assertion reads PendingCopy emptiness — the 250ms drained-row retention sits inside the timer-derived 10x-hello converge bound, so an observably empty mirror is the honest full-Ack signal on the asymmetric topology; the scenario-local capability probe names the darwin lo0 reason per D-12 (plan 06-06)
- [Phase 06]: 06-07 exec scenarios: the two-sided corrupted-record proof — A restarted ALONE first (absence/other-keys-present is provably its own disk load), then B back (heal is provably replication); corruption = one checksum bit (XOR 1) so the generation parses and PERS-02 discards exactly that record, evidenced by the restarted node's own discard stderr line
- [Phase 06]: 06-07 scenario UDP ports picked from a private window [21000, 32678) below ip_local_port_range with a claim-then-release registry held to test cleanup — no rival :0 bind can be issued there; structurally removes the pick-to-bind race that flaked the parallel suite 3/12 runs (zero after), and the scenarios leave the shared ephemeral pool so existing tests' exposure returns to its historical level
- [Phase 06]: 06-07 empty-disk bound derived in-test (N*floor(1s/rate)=400ms paced + gate/timer/CI margin), measured 116-195ms — flagged assumption A5 held, the documented fallback was not needed; awaitServing carries the sole new nolint (unparam, documented: the code-direction fix is barred by the no-harness-changes rule)
- [Phase 06]: 06-04: the D-01 guard lives in wire() (empty metricsAddr means no listenMetrics call at all — no listener, no goroutine) while listenMetrics keeps the fatal listen-then-serve bind split; the fatal bind is what makes TestMetrics_noDefaultListener's two-node collision shape discriminate against a default metrics port (plan 06-04)
- [Phase 06]: 06-04: metrics Shutdown sits immediately after the health NOT_SERVING flip in the D-19 sequence (documented in the shutdown doc comment, Pitfall 7) — scrapers see the close the moment the node stops serving; the mTLS tests' closing SIGTERM walks exercise the position with TLS active (plan 06-04)
- [Phase 06]: 06-04: the TLS builder landed in Task 2 as metrics.NewTLSConfig (the plan's sanctioned either-or) and Go's plaintext-to-TLS 400 record-header probe is asserted honestly — a plain GET must never yield the metrics surface, not must-error at the transport (plan 06-04)
- [Phase 06]: 06-05: simnet gained ReplaceNode/AddNode/Network.Reregister beyond the planned SetClockOffset (Rule 3) — in-place node replacement wired into the virtual network was required by the restart scenarios; Register appends and would leave a stale registration broadcasting, so the swap is in-place; test-harness package only, zero production code
- [Phase 06]: 06-05: partitionDays scales only the periodic cadence (hello+scan to one sync_interval, peer_timeout to three hellos) — multi-day runs at defaults cost ~20s wall in hello events plus no-op retransmit scans over retained suspended entries; every asserted semantic is cadence-independent (plan 06-05)
- [Phase 06]: 06-05: scenario drives are convergence-driven, not quiescence-driven — bare three-seam quiescence also holds in the pre-fill/pre-periodic-round gaps (emptyDisk's join wait, the restart stranded-key spread), so the loops bind on store equality with the pointwise-supremum model (plan 06-05)
- [Phase 06]: 06-05: the loss scenario's identical-supremum assertion compares winning DATA per key, never version bytes — a node's l watermark depends on absorption order before a local write, so versions are delivery-order-dependent by design; the join law makes the fixpoint data order-independent (CORE-06 at cluster scale) (plan 06-05)
- [Phase 06]: 06-08: benchmarks reuse the exact 06-06 asymmetric-topology helpers via *testing.T -> testing.TB generalization (thelper tb naming) — no second topology implementation; newScenarioNode cleanup waits for Run's return so -count reruns never leak ports
- [Phase 06]: 06-08: testing.TB carries neither Context() nor Deadline() (Go 1.24 added them to T and B only) — Deadline read through a structural type assertion in waitFor; two documented usetesting nolints where the linter's suggested tb.Context() cannot compile
- [Phase 06]: 06-08: correctness inside the benchmarks — burst run ends only when node B serves every written key (per-run namespaced keys so escalation residue cannot satisfy the check); confirmation-latency run fails on any unacked report (b.Fatalf naming the key); latency benchmark's background burst writes via the author-notification path so the 1ms cadence never serializes on dump confirmation
- [Phase 06]: [Phase 06]: 06-09: F_FULLFSYNC REPLACES File.Sync on darwin rather than wrapping it — fcntl(2) defines it as fsync-plus-drive-flush (strict superset), a preceding Sync would flush the same data twice; closes the Phase-3 macOS durability blocker as resolved-with-code (4f75a3b)
- [Phase 06]: [Phase 06]: 06-09: no darwin-only test file — no CI test can assert real power-loss behavior; both-arch cross-compile plus the unchanged linux race suite is the honest evidence (kill -9 durability unaffected on every platform)
- [Phase 06]: [Phase 06]: 06-09: Task 2 phase gate is evidence-only (no commit): lint 7.6s + race suite 7.7s + bench smoke green, go mod tidy byte-identical, zero slog under internal/app, zero blank pprof/expvar imports, all 12 D-15 rows fixed — recorded in 06-09-SUMMARY
- [Phase 06]: [Phase 06]: 06-09: D-16 fingerprint regeneration surfaced as pending-main-session work (checkpoint:human-action gate=blocking) — /gsd-execute-phase 1, 4, 5 before verify-work and milestone close; WINDOWS.md carries the open unrun-verify entry; no VERIFICATION.md hand-edited
- [Phase 07]: [Phase 07]: Option-b selected by maintainer (2026-11-17, plan 07-01 Task 3): README quickstart ships a CI-verified Linux block plus a marked darwin variant block; the probe's darwin leg is a named skip carrying the recorded macos-15 verdict (PR #32 run 44083351667: sendto 255.255.255.255 no route to host). D-06/D-07 PARTIALLY satisfied — macos-15 leg is a named skip, not a real run; WINDOWS.md deviation entry open, surfaces at phase verification (plan 07-01)
- [Phase 07]: [Phase 07]: Plan 07-01 requirements-completed empty — DOCS-01 (README quickstart) is 07-02's deliverable, DOCS-03 (full examples) is 07-03's; the tracer delivered prerequisites only (reflection, pinned tools, seed EXAMPLES.md, probe verdict). 01-05-SUMMARY precedent: declared-but-not-delivered requirements stay unchecked (plan 07-01)
- [Phase 07]: [Phase 07]: The probe's darwin loud-skip fires before any probing — determinism of the macos-15 gate cell beats re-gathering evidence the verdict already recorded; send-error exhaustion on darwin is moot post-decision (plan 07-01)
- [Phase 07]: Option-b README shape landed (plan 07-02): BOTH marked blocks (Linux literal 127.255.255.255 + darwin BCAST derive variant) carry the quickstart-verify marker so CI executes both on Linux — the variant runs via its || echo fallback arm there and cannot rot, while its macOS broadcast form stays documented-not-CI-verified
- [Phase 07]: TestReadmeQuickstart darwin leg skips deterministically before any probing (quickstartBroadcastOption constant, the 64a3cbb precedent) — determinism of the macos-15 gate cell beats re-gathering evidence the 07-01 verdict recorded
- [Phase 07]: Plan 07-02 requirements-completed empty: DOCS-01 stays unchecked because D-06/D-07 are partially satisfied (macos-15 leg is a named skip under option-b) — adjudication belongs to phase verification where option-c remains available (01-05/01-06/07-01 precedent)

### Pending Todos

None yet.

### Blockers/Concerns

- Phase 3 (Persistence): macOS F_FULLFSYNC durability policy RESOLVED WITH CODE (plan 06-09 Task 1, 4f75a3b): research resolved it — Go's darwin Fsync binds libc_fsync only, never F_FULLFSYNC; the build-tagged fcntl(fd, F_FULLFSYNC, 0) helper (internal/persistfile/fsync_darwin.go, replacing File.Sync per fcntl(2) superset semantics) makes the durability flush platform-uniform; kill -9 crash tests unaffected on every platform (only power loss distinguishes the two, no CI test simulates it).
- D-08 macOS smoke RESOLVED empirically (first macos-15 CI run 2026-11-08, PR #31): darwin refuses the shared wildcard bind (EADDRINUSE; SO_REUSEADDR does not share there) — fixed as a loud skip in 507228a; macos-15 cell expected green-with-skip on re-run.
- Phase 4 review (old, 2026-11-06 era): info-level findings IN-01 (write-only ownHandle dead state) and IN-02 (simnet Cluster.Write ignores store Set error) remain open by triage — see 04-REVIEW-DISPOSITION.md.
- Phase 1 re-review (2026-11-11): advisory findings open by triage — WR-01 (ci.yml jobs lack timeout-minutes) and IN-01 (serve()'s defer stop() releases signal handling before shutdown completes) — see 01-REVIEW-DISPOSITION.md.
- Phase 2 re-review (2026-11-14): 3 info-level findings open — IN-01/IN-02 (OnChange doc-comment inaccuracies), IN-03 (duplicated subtest) — see 02-REVIEW-DISPOSITION.md.
- Phase 3 re-review (2026-11-14): WR-01 (provably-unreachable above-cover guard in confirmWaiters — scope decision, do not auto-strip), IN-01 (dead Decode-error fallback in Open), IN-02 (ApplyRemote marks dirty on no-op merges) — see 03-REVIEW-DISPOSITION.md.
- Phase 4 re-review (2026-11-14): WR-01 (drained-row Acked set can contain a duplicate after discovery flush — missing dedup in clone(prev.Acked)+prev.WaitingOn, fix: slices.Compact after sort) and WR-02 (sync-response-rate above 1e9 divides to zero and silently disables the egress limiter — clamp in normalized()) open; IN-01 (NotifyLocalWrite TOCTOU vs closed done), IN-02 (Restore marks dirty unconditionally) — see 04-REVIEW-DISPOSITION.md.
- Phase 03 SECURITY.md was missing since the phase's original execution (2026-11-03) — created retroactively 2026-11-14 (7/7 threats closed, threats_open: 0).
- [2026-11-14] Maintainer decision: buf STANDARD satisfied with ZERO waivers (7b55ad1, pre-release window) — protos moved to package-derived trees (replproto/supkv/v1, apiproto/supkvapi/v1), service renamed SupKV→SupKVService, WatchEvent→WatchResponse (wire method paths changed; breaking-guard deferred until the rename lands on master). Supersedes the phase-5 CONTEXT pin of those names and the quick-task D-06 waivers.
- [2026-11-14] 7b55ad1 touched covered inputs of phases 1 (ci.yml, Makefile), 4 (udprepl/repl imports) and 5 (grpcapi surface) — their VERIFICATION.md fingerprints were stale. RESOLVED 2026-11-16: all five phases regenerated with fresh fingerprints in the main session per plan 06-09 Task 3's checkpoint — 1 (passed 25/25, 72497d7), 4 (passed 5/5, e47733c), 5 (passed 28/28, 9b87620), 2 (passed 40/40, 5e62bc7, go.mod toolchain staleness), 3 (passed 19/19, 8c548d8, staled by phase-6 covered-input changes to persist/engine.go and persistfile fsync). WINDOWS.md unrun-verify entry closable with phase-6 verification.
- [2026-11-15] Maintainer decision: Go toolchain directive bumped to go1.27.4 (go 1.27.1 language pin unchanged) — closes called-vulnerability GO-2026-6859 (CVE-2026-97054, net/http MaxBytesReader header race, published 2026-11-11, reachable via grpc init chains in generated code; fixed in go1.27.4). govulncheck exit 0, tidy stable, make lint/test green. Toolchain-patch bump recorded as maintainer decision per AGENTS.md version policy, no ADR.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 261108-vxx | Migrate proto codegen to buf — pinned tool, lint/format/breaking gates | 2026-11-09 | 9727c80 | [261108-vxx-migratsiya-proto-codegen-na-buf-lint-bre](./quick/261108-vxx-migratsiya-proto-codegen-na-buf-lint-bre/) |

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-11-17T13:03:20.251Z
Stopped at: Completed 07-02-PLAN.md
Resume file: None
