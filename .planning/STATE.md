---
gsd_state_version: "1.0"
milestone: v1.1
milestone_name: Spec Compliance and Resilience
current_phase: 9
current_phase_name: Documentation and Release
current_plan: 11
status: executing
stopped_at: Completed 09-10-PLAN.md
last_updated: "2026-11-28T01:24:23.328Z"
last_activity: 2026-11-28
last_activity_desc: "Plan 09-10 executed — the release mechanics landed: the hand-written CHANGELOG 1.1.0 entry with the seven (Phase N) suffixes scrubbed and the compare footers updated (TestProvenClaims green); both protos comment-scrubbed (DOC-21) with supkv.pb.go regenerated as a comment-only diff and buf gates green; .goreleaser.yaml demoted to goos:[linux] per ADR 0010/D-06 with the archive files: contract shipping docs/DEPLOYMENT.md + deploy/sup-kv.service and the sliced release header verified offline against the real CHANGELOG.md; syft v1.54.0 provisioned checksum-verified before the first green snapshot (2 Linux archives, no darwin, checksums + SBOMs); the seven-step release-procedure doc with THE ONE-WAY tag step reserved for 09-11; Q-09 dead-code verified at zero survivors"
state_head: 10e2ade7705e6e05ff7b497dca8a8ddcddbe6edc
progress:
  total_phases: 9
  completed_phases: 8
  total_plans: 105
  completed_plans: 104
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-11-19)

**Core value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.
**Current focus:** Phase 9 — Documentation and Release

## Current Position

Phase: 9 (Documentation and Release) — EXECUTING
Current Plan: 11
Total Plans in Phase: 11
Status: Ready to execute
Last activity: 2026-11-28 — Plan 09-10 executed (the hand-written CHANGELOG 1.1.0 entry with the (Phase N) scrub, TestProvenClaims green; both protos comment-scrubbed with the pb.go regenerated comment-only; .goreleaser.yaml demoted to Linux-only archives per ADR 0010/D-06 carrying DEPLOYMENT.md + the systemd unit, the sliced release header verified offline; the snapshot gate green with checksum-verified syft v1.54.0; the release-procedure doc with the one-way tag step reserved for 09-11; Q-09 dead-code verified at zero survivors)

Progress: [█████████▉] 99%

## Performance Metrics

**Velocity:**
- v1.0 (archived): 38 plans, 7 phases — avg ~10 min/plan; full per-plan history in [milestones/v1.0-ROADMAP.md](milestones/v1.0-ROADMAP.md)
- v1.1: no plans executed yet

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| v1.1-1 | 0 | - | - |
| 1 | 4 | - | - |
| 2 | 17 | - | - |
| 3 | 13 | - | - |
| 4 | 13 | - | - |
| 5 | 12 | - | - |
| 6 | 13 | - | - |
| 7 | 11 | - | - |
| 8 | 11 | - | - |

*Updated after each plan completion*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P01 | 349s | 3 tasks | 51 files |
| Phase 01 P02 | 787s | 3 tasks | 1 files |
| Phase 01 P03 | 524s | 3 tasks | 11 files |
| Phase 01 P04 | 299 | 2 tasks | 0 files |
| Phase 02 P01 | 520s | 3 tasks | 8 files |
| Phase 02 P02 | 2040s | 2 tasks | 4 files |
| Phase 02 P03 | 441 | 2 tasks | 1 files |
| Phase 02 P04 | 1191s | 2 tasks | 5 files |
| Phase 02 P14 | 178s | 1 tasks | 1 files |
| Phase 02 P05 | 1044 | 2 tasks | 1 files |
| Phase 02 P06 | 574 | 2 tasks | 2 files |
| Phase 02 P07 | 350 | 2 tasks | 2 files |
| Phase 02 P08 | 983s | 2 tasks | 2 files |
| Phase 02 P10 | 269s | 2 tasks | 7 files |
| Phase 02 P15 | 10min | 1 tasks | 1 files |
| Phase 02 P16 | 169s | 1 tasks | 1 files |
| Phase 02 P09 | 900 | 3 tasks | 10 files |
| Phase 02 P12 | 720s | 2 tasks | 8 files |
| Phase 02 P17 | 8min | 1 tasks | 22 files |
| Phase 02 P13 | 40min | 2 tasks | 4 files |
| Phase 02 P11 | 694s | 3 tasks | 3 files |
| Phase 3 P01 | 427s | 3 tasks | 5 files |
| Phase 03 P02 | 297s | 3 tasks | 6 files |
| Phase 03 P03 | 148 | 2 tasks | 2 files |
| Phase 03 P04 | 10min | 2 tasks | 9 files |
| Phase 03 P05 | 15min | 3 tasks | 4 files |
| Phase 03 P06 | 1252s | 3 tasks | 17 files |
| Phase 03 P07 | 20min | 3 tasks | 11 files |
| Phase 03 P08 | 21min | 3 tasks | 11 files |
| Phase 03 P09 | 602s | 3 tasks | 9 files |
| Phase 03 P10 | 925s | 3 tasks | 8 files |
| Phase 03 P11 | 649s | 2 tasks | 19 files |
| Phase 03 P12 | 1382s | 3 tasks | 17 files |
| Phase 03 P13 | 1048s | 2 tasks | 10 files |
| Phase 04 P01 | 23min | 3 tasks | 17 files |
| Phase 04 P02 | 5min | 2 tasks | 8 files |
| Phase 04 P03 | 587s | 2 tasks | 16 files |
| Phase 04 P04 | 10min | 2 tasks | 9 files |
| Phase 04 P05 | 791s | 2 tasks | 5 files |
| Phase 04 P06 | 294 | 2 tasks | 5 files |
| Phase 04 P07 | 287s | 1 tasks | 1 files |
| Phase 04 P08 | 1800s | 3 tasks | 9 files |
| Phase 04 P09 | 847 | 2 tasks | 5 files |
| Phase 04 P10 | 397s | 2 tasks | 3 files |
| Phase 04 P11 | 780 | 2 tasks | 2 files |
| Phase 04 P12 | 272s | 2 tasks | 3 files |
| Phase 04 P13 | 1178 | 3 tasks | 10 files |
| Phase 05 P01 | 761s | 3 tasks | 14 files |
| Phase 05 P02 | 868 | 3 tasks | 6 files |
| Phase 05 P03 | 1224 | 3 tasks | 9 files |
| Phase 05 P04 | 13min | 3 tasks | 13 files |
| Phase 05 P05 | 829s | 3 tasks | 10 files |
| Phase 05 P06 | 881 | 3 tasks | 9 files |
| Phase 05 P07 | 1114 | 3 tasks | 10 files |
| Phase 05 P08 | 924 | 3 tasks | 10 files |
| Phase 05 P09 | 12min | 3 tasks | 7 files |
| Phase 05 P10 | 3min | 2 tasks | 2 files |
| Phase 05 P11 | 134 | 2 tasks | 1 files |
| Phase 05 P12 | 30min | 2 tasks | 2 files |
| Phase 06 P01 | 721s | 3 tasks | 30 files |
| Phase 06 P02 | 8min | 1 tasks | 19 files |
| Phase 06 P03 | 7min | 3 tasks | 7 files |
| Phase 06 P04 | 7min | 3 tasks | 6 files |
| Phase 06 P11 | 30min | 2 tasks | 11 files |
| Phase 06 P05 | 817s | 3 tasks | 14 files |
| Phase 06 P06 | 580s | 1 tasks | 12 files |
| Phase 06 P12 | 361 | 2 tasks | 7 files |
| Phase 06 P07 | 522s | 3 tasks | 11 files |
| Phase 06 P08 | 705s | 3 tasks | 4 files |
| Phase 06 P09 | 284s | 2 tasks | 4 files |
| Phase 06 P10 | 200s | 2 tasks | 4 files |
| Phase 06 P13 | 20min | 2 tasks | 2 files |
| Phase 07 P01 | 567s | 2 tasks | 9 files |
| Phase 07 P02 | 1324s | 3 tasks | 14 files |
| Phase 07 P03 | 261 | 2 tasks | 1 files |
| Phase 07 P04 | 819s | 2 tasks | 3 files |
| Phase 07 P05 | 1154 | 2 tasks | 10 files |
| Phase 07 P06 | 27min | 2 tasks | 8 files |
| Phase 07 P07 | 24min | 3 tasks | 9 files |
| Phase 07 P08 | 1214s | 2 tasks | 13 files |
| Phase 07 P09 | 921s | 3 tasks | 12 files |
| Phase 07 P10 | 281s | 2 tasks | 4 files |
| Phase 07 P11 | 1105 | 2 tasks | 2 files |
| Phase 08 P01 | 746s | 3 tasks | 9 files |
| Phase 08 P02 | 954 | 3 tasks | 2 files |
| Phase 08 P04 | 408 | 2 tasks | 4 files |
| Phase 08 P05 | 15min | 3 tasks | 2 files |
| Phase 08 P06 | 9min | 3 tasks | 9 files |
| Phase 08 P03 | 7min | 2 tasks | 2 files |
| Phase 8 P7 | 15min | 3 tasks | 3 files |
| Phase 08 P08 | 6min | 3 tasks | 2 files |
| Phase 08 P09 | 13min | 3 tasks | 5 files |
| Phase 8 P10 | 17min | 3 tasks | 5 files |
| Phase 8 P11 | 20min | 2 tasks | 0 files |
| Phase 09 P1 | 10min | 3 tasks | 3 files |
| Phase 9 P2 | 18min | 2 tasks | 3 files |
| Phase 9 P3 | 30min | 3 tasks | 11 files |
| Phase 09 P04 | 35 min | 3 tasks | 3 files |
| Phase 09 P05 | 10 min | 3 tasks | 4 files |
| Phase 09 P06 | 23 min | 2 tasks | 3 files |
| Phase 09 P07 | 35 min | 4 tasks | 5 files |
| Phase 09 P08 | 40min | 3 tasks | 5 files |
| Phase 09 P09 | 13min | 3 tasks | 8 files |
| Phase 09 P10 | 18 min | 4 tasks | 6 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: v1.1 phase structure follows v1.0 review section 15, maintainer-confirmed 2026-11-19 — red tests (Phase 2) precede fixes (Phases 3-5); no deviation from the confirmed composition
- Roadmap: phase numbers reset to 1 for v1.1 (`--reset-phase-numbers`); v1.0 phases 1-7 are archived history under milestones/v1.0-phases/
- Roadmap: owner decisions 14.1-14.11 are settled (2026-11-19) and are not re-litigated; 14.5(6) and 14.7 accepted by default and recorded in SPEC-02
- Roadmap: Phase 2's Critical-scenario tests (R-01, R-02, R-03, P-01) are expected RED against v1.0 code by design — planning must choose the gating mechanism so red tests can merge without breaking CI (they turn green in Phases 3-5)
- [Phase 01]: Publication-guard verify gates scoped to the non-public prefix docs/reviews/2026-11-19-v1.0/internal/; evidence/repro/**/internal/** repro paths are legitimate D-01 payload (Rule 1 fix during 01-01 Task 1)
- [Phase 01]: Registry Links cells expand the trace's shorthand code refs to full repo paths per the trace's own file legend, so public readers resolve links without the local-only legend
- [Phase 01]: S-PROTO-08 closed as implemented with recorded-intent nuance in the summary cell (04-CONTEXT D-02); accepted-deviation(14.x) would have been an invented reference
- [Phase 01]: S-HLC-07 accepted-deviation references recorded v1.0 research assumption 02-RESEARCH A2 — no ADR/14.x covers the zero filler; amend-check confirmed the amendment left the random-filler requirement standing
- [Phase 01]: Unnamed rows resolved by the priority rule with REQ IDs verified in REQUIREMENTS.md: S-PROTO-02/03 to XPORT-03, S-PAR-05 to PERSIST-01, S-ALG-RC-09 to XPORT-07, S-PAR-16 to OBSV-05, S-FAIL-17 to OBSV-06, S-FAIL-03 to PROTO-03, S-HLC-10 to PROTO-01, S-IMP-08 accepted-deviation(ADR 0008)
- [Phase 01]: ADR set authored slim-MADR with rule-level detail for 14.5 (six numbered sub-rules) so Phase 4 can address accepted-by-default rule (6) individually; honest gaps kept (0002 informs [], 0011 loss-share has no dedicated v1.1 REQ) instead of invented mappings
- [Phase 01]: Phase gate closed with zero task commits: every gate passed on the committed tree as plans 02/03 left it; fixups were gated on gate failure and none failed — Plan 01-04's <output> creates no new files and commits fixups only when a gate fails; committing a no-op would fabricate history - the green gate record lives in 01-04-SUMMARY.md
- [Phase 01]: Scoped-prefix publication guard (docs/reviews/2026-11-19-v1.0/internal/) is the standing phase-gate assertion, never any-path-containing-internal substring matching — 22 legitimate evidence/repro/**/internal/** repro-test module paths are committed D-01 payload; literal substring matching contradicts the 46-tracked-evidence-files acceptance (wave-1 Rule 1 scoping, re-affirmed at the gate)
- [Phase 02]: R-03 red test's SyncArrived fake bakes v1.0 mirror semantics (any-chunk->true) with closed-side assertions parked behind a release channel (TestGate_opensOnTimeout pattern) — the Phase 4 flip needs zero test edits, only the expectation-table row
- [Phase 02]: red-runner errors on unexpected failing tests (names not in the expectation table; subtests count as their parent's) — T-2-01 'unknown names are errors' applied to the event stream
- [Phase 02]: red-runner package list hardcoded with an extend comment (./internal/grpcapi/ first) — the TSV carries no package column
- [Phase 02]: internal/stand harness + scenarios landed as one commit (91765fd): the strict lint gate's transitive unused analysis cannot reach the t-coupled harness helpers without their callers — stand.go alone is un-committable; the plan's Task 1/2 file split survives in content, only the commit boundary moved
- [Phase 02]: internal/stand keeps the plan's file names AND internal tests via justified //nolint:testpackage (stand_test.go package clause) and //nolint:depguard (testing import in stand.go) — both with written reasons per AGENTS.md; the package doc states the test-infrastructure status the path-based rules cannot see; sibling plans 02-05/02-14 follow the same pattern
- [Phase 02]: The cmd/sup-kv port registry was NOT dead after the scenario move — broadcast_probe_test.go (Phase 7 DOCS-01 probe) still uses scenarioUDPPort, so it moved to its remaining consumer and main_test.go needed zero removals
- [Phase 02]: F2c pin rides standAbsent (NotFound-based absence) + standSetWith (review's 1-byte value): a value-mismatched record must fail the never-replicated window, and the derived 70KB scenario value would trip the store's value limit before the datagram ceiling matters
- [Phase 02]: Cluster.Delete returns (storage.Record, error): the plan's must_haves ErrNotFound propagation beats the prose's bare-Record signature
- [Phase 02]: ReplaceNodeWithState seeds via store.Merge in sorted-key order: map iteration would randomize the HLC watermark the merges leave behind and break same-seed replay; the VERS-04 property rides the merge path, no Watermark pre-seed
- [Phase 02]: simnet size gate refuses at schedule time (before booking — PendingDeliveries stays truthful); receive budget consults at fire time against the live map, keyed by handle so it survives ReplaceNode
- [Phase 02]: AnnounceDelay rides port.DataReplication only (write broadcasts + sync answers); over-limit seed records panic per the checkIndex/Reregister harness convention
- [Phase 02]: fsops seam: failure hygiene is impl-side in shared production helpers (closeRemoveTemp/removeTemp) — natural failure paths and the injection fake run the same hygiene code, so the failure matrix observes real dump hygiene; saveAtomically is a pure MkdirAll + five-call sequence, behavior bit-identical (existing suites green)
- [Phase 02]: fsops fakes ride the embedded real fileFSOps: the order assertion (write→fsync→rotate→rename→dir-fsync) is taken over the production Save path's actual operations and the rotation chain over real files; the failing fake short-circuits before the real op and performs the step's own hygiene via the production helpers
- [Phase 02]: fsops internal-test placement: justified //nolint:testpackage keeps the plan's artifact name fsops_test.go with package persistfile (the internal/stand precedent) so the fakes reach the unexported FileStore.ops injection point; exported constructors unchanged, nothing added to internal/port
- [Phase 02]: D-07 blocking proof rides a synctest bubble with an in-memory blockingDumpPort (startDumper pattern): deterministic not-returned window, no real-time flake surface for the nightly -race -count=20 soak
- [Phase 02]: internal/stand nondarwin half: the darwin-untested leak assertion is a real runtime check — go/build MatchFile against an explicit darwin context over every sibling .go file (constraint-form-agnostic), so a leaked scenario file fails the darwin cell with a named error; the loud statement is a wrapped const keeping the plan's grep gate at exactly one phrase line
- [Phase 02]: R-03's stand closed window is shaped structurally (A seeded+stopped, replaced node alone with a 30 s backstop, then A restarts) rather than raced against a live multi-chunk listing; v1.0's Get answers NotFound, never Unavailable
- [Phase 02]: All four stand red tests assert the documented acceptance expressions (R-01 2×sync_interval + N/rate; R-02 N/rate + 10 s and ≤3N; burst ≥990/s and 5 s drain; R-03 five-point PROTO-01), so Phases 3-5 flip only the expectation-table rows
- [Phase 02]: Chaos corpus format is bare decimal seed lines (no header): head -1 of testdata/chaos-seeds.txt stays a valid SUPKV_SIM_SEED replay argument for the plan verify and CI scripts
- [Phase 02]: Deletion-heavy chaos scenarios use a deterministic closing drain sweep (each node's sorted live keys, no rng or time advance) so the corpus guarantees empty/tombstone-only final states without weakening model equality, the 10x hard cap, or the fault teeth
- [Phase 02]: Chaos deletes target the acting node's sorted live keys via a pre-drawn pick index: 'against live keys' holds at execution time while every random input stays fixed before any event runs
- [Phase 02]: Negative-table counter exactness rides an asymmetric adapter whose broadcast lands in a test-held sink socket, so own-echo can never contaminate the per-class DiscardStats deltas
- [Phase 02]: Wrong-proto_version is Decoded at decode() and BadProtoVersion at the core: the negative table asserts the codec verdict and the counter separately, pinning D-02's split
- [Phase 02]: Empty-key, tombstone-with-data and oversize rows pin CURRENT behavior with explicit XPORT-10/XPORT-03 forward references (processed evidence = the apply Ack where no counter exists) and flip in Phase 3
- [Phase 02]: DELETE-twice is codes.NotFound (the store no-op, no version burned) plus exactly one dump record line; node-down-during-delete bound derives from explicitly pinned flags, never production defaults
- [Phase 02]: Watch Send-error branch pinned twice (02-08): fake server streams assert the handler returns the transport error unchanged from both Send sites (snapshot loop and relayEvent) while a real bufconn transport pins the cleanup/isolation contract — a real client cancel races the handler's ctx.Done branch, so a fake is the only deterministic Send failure
- [Phase 02]: Conflict exactness rides ordered writes (02-08): B serves A's version before writing so its HLC has absorbed it and its write strictly dominates; both nodes run --sync-interval=10s so no stale proactive listing adds a rejected-older-version conflict on A
- [Phase 02]: sync-response-rate effect needs a restarted seeded node (02-08): handleHello's discovery flush broadcasts the sender's pending records immediately and bypasses the egress limiter; a dump-loaded restart leaves pending empty, so the empty node's paced DataRequests are the only recovery path
- [Phase 02]: Retransmit asserted as a named lower bound (02-08): >=1 plus a pre-kill baseline climb because cycle count depends on scheduler-dependent Ack timing on real sockets; conflict and discard are constructed single events and assert ==1
- [Phase 02]: Per-parameter effects are named subtests (02-08): one per stand-observable parameter (startup-gate, hello-interval, sync-response-rate, value-max-bytes, peer-timeout), each with its own topology/surface; remaining flags map to existing unit/exec tests in the SUMMARY parameter table
- [Phase 02]: Shared benchmark percentile reporter lives in the module-wide internal/testutil package: Go's internal rule blocks internal/udprepl and internal/grpcapi from internal/app/internal/testutil, and all three benchmarks share one implementation; the plan's app-internal path is a documented wrapper
- [Phase 02]: Benchmarks report p50-ms/p99-ms from their own measured per-op spans with no self-pacing; the -benchtime=1x smoke reports p50==p99==the single sample
- [Phase 02]: make race-soak is the single TEST-11 command definition (-race -count=20 over the affected set, COUNT override); T-11 dispositions: port window intact (mitigated), pre-arm SIGTERM carried per D-09 (TestShutdown_secondSignal remains the pin), README deadline fixed test-side per doc block
- [Phase 02]: Extension tests live in the external simnet_test package: the internal form would need a //nolint:testpackage suppression (skip-regexp covers only *_internal_test.go) while the external form imports replproto under the depguard $test exemption and reuses the existing simnet_test.go helpers
- [Phase 02]: The determinism replay gate asserts extension activation (OversizeRefused > 0, RecvBudgetRefused > 0, non-empty stores) before comparing same-seed digests, so a zero-policy replay cannot pass vacuously
- [Phase 02]: 02-16: the TEST-07 proof lives in package persistfile_test (external) composing only exported API — the internal package would need an unjustified //nolint:testpackage, and the external form reuses dumpfile_test.go's writeFile helper (same reasoning as 02-15's simnet_test)
- [Phase 02]: TEST-13 coverage meta-test lives in internal/app/repl: an 11-row index (decision, ADR file, anchor test) with word-bounded citation matching and self-exclusion; a decision counts when a test file carries ADR 000N or decision 14.N, so 14.10 cannot satisfy 14.1
- [Phase 02]: The printed property replay hint is package-scoped: testutil resolves the caller's import path from the internal/app anchor via runtime.Caller (os-free, depguard-safe) and ReplayCommand shares the one format helper; a test executes the printed string and asserts exit 0 plus the same test's re-run
- [Phase 02]: TEST-15 small-space records derive every field from the version so equal versions are byte-identical — the premise the equal-version no-op law needs; a separate subtest pins that an incoming equal-version record with different bytes is ignored
- [Phase 02]: P-01 engine test lives in package persistfile_test (external) not the plan's literal internal package: it composes only exported API and reuses open_seed_test.go's seedDumper/seedFixedNow helpers, while the internal form would need an unjustified //nolint:testpackage (02-15/02-16 precedent)
- [Phase 02]: scripts/red-runner serializes packages with -p 1: the stand reds assert wall-clock pacing and throughput bounds, so sibling packages' race builds and load would corrupt the verdict's measurement
- [Phase 02]: The P-01 process test asserts the dump still holds the record between Set (which confirms the covering dump, PERS-04) and SIGKILL/restart, so a red verdict names the loader's 61 440-byte decode bound rather than leaving the writer or the kill suspect
- [Phase 02]: Fuzz targets choose external test packages over the existing test-binary decode seam: FuzzDecode drives decode() through DecodeDatagramForTest, FuzzDumpDecode through persist.Decode, adding no testpackage exemption and no production change
- [Phase 02]: Fuzz seed builders are plain error-returning functions so the fuzz targets and the committed corpus files share one payload list; corruption fixtures carry construction guards (truncation still fails to unmarshal, base64 flip changes decoded bytes) to prevent class decay
- [Phase 02]: long-partition subtests straddle the REPL-06 at-or-after eviction boundary at peer_timeout ± one hello round: a partition of exactly peer_timeout evicts at its heal instant, so the plan's 'exactly = not yet evicted' assumption is inverted by the code
- [Phase 02]: periodic-sync-disabled mutant advances nextSync then returns on the timer path: a bare return leaves the deadline in the past and livelocks the scheduler, so the runner also rejects go-test timeouts as non-kills
- [Phase 02]: mutant kill seeds are per-mutant verified corpus lists (sync-without-tombstones: 1 seed; periodic-sync-disabled: 3 seeds) because an exhaustive 16-seed scan showed different sensitivity, and every probed seed must fail
- [Phase 02]: TEST-09 no-fresh-loss qualifier is structural: all writes and fault windows end at the heal (loss via deterministic directed windows, only duplication continues), and every pre-heal record is asserted present byte-equal on every node
- [Phase 02]: Nightly F9 runs the reviewer's documented default flags (hello 1s, sync 30s, rate 500) and both the recovery bound and the post-recovery write wait derive from the selected flags — the shrunken-timer wait was replaced after it timed out at the 30s cadence.
- [Phase 02]: scheduled.yml's two stand jobs share SUPKV_STAND_NIGHTLY=1; red-run owns assert-red (the runner exports SUPKV_RED_TESTS itself) while nightly-stand leaves red tests skipped by construction.
- [Phase 03]: Effective socket buffer sizes come only from the kernel getsockopt read-back; the requested constants never reach the startup log or SocketBuffers (XPORT-05).
- [Phase 03]: validateBufferRequest accepts exactly MaxInt32/2 and rejects anything above it: the kernel doubles the request, so the read-back must fit a 32-bit int (T-3-01); this resolves the plan Task 3 table contradiction in favor of the primary guard contract.
- [Phase 03]: Every buffer-configuration failure (set, read-back, send floor) is a construction error that closes the socket before New returns; no degraded node starts.
- [Phase 03]: The stand delivery test passes --sync-response-rate explicitly so its paced bound term derives from a flag the nodes actually run with; this also clears unparam on startStandLoggedNode.
- [Phase 03]: Store read view is tombstone-visible by construction: Version, Lookup and Versions include deleted records while Get stays live-only.
- [Phase 03]: Every view read returns caller-owned state: Lookup clones Data and Versions builds a fresh map, never a view of store internals.
- [Phase 03]: storage.MaxKeyBytes = 1024 documents write-side-only validation; reads stay permissive for pre-upgrade long keys.
- [Phase 03]: Cluster.Write panics naming operation, key and node on a rejected store write, and Write/Delete read through the point Lookup, not Records().
- [Phase 03]: Version layout is validated before the future bound, keeping ErrVersionLayout and ErrVersionFuture distinguishable for the bad_version discard class (XPORT-04, D-06).
- [Phase 03]: The 46 RFC 9562 rand_b filler bits are accepted so a conforming third-party UUIDv7 passes; rejecting them would break interoperability.
- [Phase 03]: Validate's future bound is inclusive at now+maxLead, overflow-safe by subtraction, with a negative lead clamped to zero.
- [Phase 03]: StateStore's tombstone-visible view is the interface contract: Records was removed so the O(store) sync path cannot return without a visible interface change (XPORT-06, T-3-12).
- [Phase 03]: DataSync compares per entry through Version(key) and reads a reply record with Lookup only when the local version is strictly newer; Versions() is never called on the inbound path (XPORT-06).
- [Phase 03]: Benchmarks pin SyncResponseRate at 1e9/s so one nanosecond of virtual time drains one egress slot; TestCore_keyCountCostFlat enforces the 2x bound and names both ns/op values (XPORT-06).
- [Phase 03]: 03-05: Discovery flush always records the new peer as a waiter; a Lookup miss skips only the broadcast, and the retransmit scan deletes the stale entry at its next due time.
- [Phase 03]: 03-05: Due retransmissions emit by (key, version) to preserve the sorted-emission pin while the heap (nextRetry, seq) keeps a deterministic pop order; the invariant test inserts ascending keys.
- [Phase 03]: 03-05: Local supersession deletes every strictly smaller tracked version under the key; an equal-version notification is a tracking no-op that still broadcasts.
- [Phase 03]: 03-05: Remote supersession deletes every tracked version under the key strictly smaller than the applied remote version.
- [Phase 03]: 03-05: PendingSnapshot reports the greatest version under a key when two coexist transiently, matching the store's LWW choice.
- [Phase 03]: 03-05: mirrorState is untouched here (03-06 removes it); only the gauge half of mirrorStats changed to PeerCount().
- [Phase 03]: 03-05: The flatness gate stays sequential with a justified //nolint:paralleltest because it is a wall-clock ratio gate.
- [Phase 03]: 03-06: The Outbox result stream is the only waiter seam — Ack/Drained/Superseded/WaitSetEmpty cross as plain data; an Ack alone never resolves, and success requires resolved full coverage.
- [Phase 03]: 03-06: Tracker creation is confined to Adapter.NotifyLocalWrite before the channel send (the sole production creation site); Run only looks trackers up and drops results for absent identities, never inserting.
- [Phase 03]: 03-06: Every WaitReplicated return path deletes its tracker (resolution, zero-peer release, ctx cancel, timeout) and a failed notify deletes the one it created, bounding the map by concurrent waits (T-3-16).
- [Phase 03]: 03-06: The zero-peer path releases through the total==0 WaitReplicated call, so D-11's immediate success cannot leak the notify-time tracker; wait.go discards that call's report.
- [Phase 03]: 03-06: Eviction and supersession resolve an honest false; WaitSetEmpty is emitted only on the last-waiter transition during eviction, so an already-suspended entry is not re-announced.
- [Phase 03]: 03-06: Tasks 2 and 3 share commit c1da11e because the LocalNotify/WaitReplicated signature change and PendingCopy removal are one compile unit; splitting would leave go build ./... broken.
- [Phase 03]: 03-06: Tracker tests write the store record before notifying (the production author sequence): a notification for an unstored record is removed by the retransmit scan and would make the eviction wait time out.
- [Phase 03]: 03-07: Three pacers share the normalized syncResponseInterval with independent slots; the egress spacing stays exact while the listing and record spacing carry the ±10% deterministic jitter.
- [Phase 03]: 03-07: A destination is in flight exactly while its listing FIFO exists; a busy trigger advances nextSync and enqueues nothing, so the short-interval stand can never livelock.
- [Phase 03]: 03-07: The discovery flush and retransmits share one (key, version)-deduplicated queue; the author's first broadcast stays immediate, and the ladder is re-armed only at emission.
- [Phase 03]: 03-07: A flush send whose record is absent leaves the entry tracked (only the retry path deletes stale entries), preserving the 03-05/03-06 synthetic-notify contract.
- [Phase 03]: 03-07: Jitter is FNV-1a(runID, counter, salt): positive-only on Hello/sync deadlines, ±10% on listing/record spacing, and never on a ladder deadline.
- [Phase 03]: 03-07: The egress identity is (handle, kind, key); the version stays out and DataReplication answers re-read the store at drain, so the queue never pins Data.
- [Phase 03]: 03-07: The egress cap and its dedupe map mutate in one function (dropOldestEgress); QueuedResponses keeps its egress meaning while QueuedSends is the total-queue seam.
- [Phase 03]: 03-07: The empty-disk scenario bound adds the sync-jitter envelope, and partitionDays asserts conflicts ≥ sharedKeys because a paced flush can supersede the loser before it crosses the wire.
- [Phase 03]: 03-08: Validation is a pipeline step between classification and handleMessage, returning (class, detail, valid); its position is the contract because handleMessage is the only path to ApplyRemote, hence to Clock.Update.
- [Phase 03]: 03-08: The validation detail is exactly the source handle; the class is OnDiscard's first argument, so no key name or node id rides the detail or the per-class WARN (T-3-27).
- [Phase 03]: 03-08: An Ack validates its key but not its version — it carries no record, so a crafted future Ack is a harmless tracking no-op; a control row pins the asymmetry.
- [Phase 03]: 03-08: Config.MaxClockLead normalizes ≤ 0 to 60 s inside the core; the launch flag's default is documentation only, and scenarios that legitimately need more set an explicit lead (clockJumps 4 h, farAhead 25 h, drift test 2 min).
- [Phase 03]: 03-08: The validation WARN is one per class per 10 s of event-loop time via a per-class timestamp map; counters and the OnDiscard seam stay per-datagram, and no wall clock is read.
- [Phase 03]: 03-08: Every hand-built inbound record fixture must carry the production checksum (replFrom recomputes it); a zero-checksum fixture would otherwise be rejected and a test could pass for the wrong reason.
- [Phase 03]: 03-08: The metrics label vocabulary stays at five classes; 03-12 owns the five new class samples, so this plan stops at the adapter's Discards() mirror.
- [Phase 03]: 03-08: Task 1's commit carried the test-model, fixture and negative-table adaptations because the Task 1 verify runs the whole TestDiscard_ filter and the table pins pre-fix behavior; Task 2's commit added the three proof tests.
- [Phase 03]: 03-09: The datagram oracle marshals the real schema and sets the tombstone flag only for the empty-value shape — a valued record is never a tombstone (receive-side discard class) — so 1 024 + 64 408 measures exactly 65 507 instead of an overcounted 65 509.
- [Phase 03]: 03-09: The startup probe width is clamped to the 65 507-byte ceiling before allocation, so a misconfigured multi-gigabyte value exits 2 instead of OOMing during measurement; that clamped case reports the size as a lower bound.
- [Phase 03]: 03-09: API key enforcement is write-side only (Set/SetIfAbsent/Delete before the engine); Get/List stay permissive for pre-upgrade long keys per RESEARCH OQ2, with Phase 5 owning load-time accounting.
- [Phase 03]: 03-09: The startup comparison is `>` and both sides are pinned executably: --value-max-bytes=64 408 (exactly 65 507 bytes) starts, 64 409 exits 2, and 64 000 remains the P-01 executable guard distinct from the rows-unchanged TSV grep.
- [Phase 03]: 03-09: F2c keeps its name but now pins the fix — the 70 000-character key is rejected locally and never replicates, while a 1 024-byte key with a 61 440-byte value replicates and lands in node B's final dump.
- [Phase 03]: 03-10: The read loop exits only on `net.ErrClosed`; every other receive error increments `Adapter.ReadErrors`, logs at most once per second and sleeps the 100 ms backoff. The fake-`packetConn` test pins continuation (datagram processed, one backoff observed) and the no-spin bound, so 03-12 can publish the counter as-is.
- [Phase 03]: 03-10: The `packetConn` seam carries `LocalAddr` because construction derives the core's own handle from the bound address (D-03) and the white-box test reads it; production binds set no `SO_REUSEADDR` (`WithAddressReuse` is test-only), and both sockets of a shared port must opt in — Linux only grants the double-bind when every socket set the option.
- [Phase 03]: 03-10: The no-Hello WARN counts from the first `Advance` (not construction) and latches for the process lifetime; `helloSeen` ignores the node's own broadcast echo, so a lone node that hears only itself still warns. The non-wildcard bind WARN is emitted in `main` between `setupLogging` and `wire`, never a rejection.
- [Phase 03]: 03-11: Dump confirmation is proven by lock/capture order, not a watermark — `await` appends after the store write under the engine mutex, and `runDump` captures the waiter list before `Encode(e.store.Records())`, so every captured waiter's record is in the saved snapshot; the proof is documented in all three sites.
- [Phase 03]: 03-11: `waiter.ver` and `await`'s version parameter were deleted with the cover branch (nothing read them); the write paths call `e.await()` and `confirmWaiters` keeps a blank `(*Engine)` receiver to satisfy revive's unused-receiver check.
- [Phase 03]: 03-11: `persist.Parse` was literally `Decode`'s error channel, so the codec table's Parse/Decode columns were merged into one `Decode` assertion per row — no expectation lost; `Open` already carried its own Decode closure.
- [Phase 03]: 03-11: `fsops` now calls `os.File.Sync` directly; the darwin F_FULLFSYNC/EINTR/ENOTSUP path is the standard library's internal/poll `Fsync`, and S-PERS-03 records that as-built path. ADR 0010 was checked and left untouched (no fsync reference).
- [Phase 03]: 03-11: The ownHandle inventory was file-counted, not call-counted — five listed files construct through `udprepl.New` and needed no edit, while `internal/app/repl/egress_internal_test.go`'s unqualified `New(...)` was found by `go vet`. Realized footprint: 16 modified + 3 deleted = 19 files.
- [Phase 03]: 03-12: `Core.pacedSends` increments at the Outbox handoff in `drainListings`/`emitRecordSend`/`drainEgress`, never at enqueue (T-3-41); RPL-18's send-failure accounting stays Phase 7's OBSV-03, and `TestPaced_sendCounting` pins one emission against the whole enqueued listing round.
- [Phase 03]: 03-12: The red-runner's tombstone mutant moved from `core.go`'s old `listingChunks` shape to `storage.Versions` (the 03-07 rewrite left `Versions` with no `Deleted` flag); across all 16 corpus seeds the control passes, `sync-without-tombstones` is killed only by 848612892191557850, and `periodic-sync-disabled` by the same three seeds.
- [Phase 03]: 03-12: The flip halted at one of three rows — `TestRed_R02_unackedDrain` is green, while R-01 (7 453-7 734/8 000 keys at the 20.6 s bound) and R-02 burst (979.6-985.7 vs 990 writes/s) stayed red after three full runs and two standalone retries; the rows were reverted per the plan, and the failed-attempt evidence is committed beside the accepted runs.
- [Phase 03]: 03-12: The bound misses are host-speed findings, not regressions — pre-fix evidence on this host measured 1 504/8 000 keys and 25.1 writes/s, the host runs `powersave` at ~1.64 GHz, and no system power setting was changed; 03-13 is blocked until a full-speed re-run or the scheduled job re-evaluates the two rows.
- [Phase 03]: 03-13: The 20/20 acceptance lives in a dedicated `red-r01-x20` scheduled job — `-count=20` at the documented 20 000-key scale, `-timeout 110m`, `timeout-minutes: 120`, both env gates set, no skip/tolerance flag; a failure after the flip is a transport verdict and fails CI.
- [Phase 03]: 03-13: The job's `always()` evidence step tests its glob with `compgen -G` instead of red-run's `ls … || true`, because the plan's prohibition bans `|| true` in this job; the no-evidence branch states the direct-`go test` contract instead of fabricating a directory.
- [Phase 03]: 03-13: `make race-soak` gained `./internal/stand/` only after measuring it — 6.53 s single run, ~2.2 min projected at COUNT=20; the whole target is 81.22 s at COUNT=1, so ~27 min at the default 20 and the 120-minute race-flake budget stands.
- [Phase 03]: 03-13: The documented-scale R-01 run measured 110.26 s, projecting 20 runs to ~37 min — inside the 120-minute job budget and consistent with RESEARCH A5; the first scheduled run is the observation, and a budget miss is recorded rather than paying with the count.
- [Phase 03]: 03-13: The full gate ran with real exit statuses (tidy, lint, test, red-runner with mutants, darwin cross build/vet, bench smoke, one nightly-scale R-01 run, clean tree); the four remaining red rows (R-03 x2, P-01 x2) are Phase 4/5 work and no bound, row or test was weakened.
- [Phase 04]: 04-01: DataSync v2 carries sync_id/chunk_index/chunk_count; chunks accumulate as an index set per (node_id, run_id, sync_id), and only all indices 1..count open the gate.
- [Phase 04]: 04-01: One atomic {complete, keys} mirror publishes only on a union-content generation move; an identical listing flood clones and republishes nothing.
- [Phase 04]: 04-01: Retained listed keys are capped at 131072 by whole-peer-set LRU eviction through refcounts; ListedKeysEvicted counts keys that actually left the union.
- [Phase 04]: 04-01: proto_version is judged pre-CRC/body as ProtoVersionMismatch; bad_sync_header is a header-first class; both WARN per (node_id, class) on event time.
- [Phase 04]: 04-01: chunkOverheadBytes 96 with both graduated lenses re-derived to sum 1112; sync_id is a monotone build-time counter and run_id is part of the accumulation key, keeping same-seed replay byte-identical.
- [Phase 04]: 04-02: Get distinguishes a local tombstone from a listed-but-unarrived key with Engine.Version, never Engine.Get — Get hides tombstones, so probing it would answer Unavailable forever for a deleted listed key (RESEARCH Pitfall 10).
- [Phase 04]: 04-02: ServerOptions.ListedKey is a nil-able seam (nil = no listing knowledge), preserving the single-node configuration and every pre-existing harness; production wiring is 04-06's.
- [Phase 04]: 04-02: The gate is every data handler's first statement, before validation, listed-key knowledge and the nil-hub check; a closed node answers Unavailable for reads and writes alike.
- [Phase 04]: 04-02: create_if_absent on any listed key is AlreadyExists before the engine call; the post-open race test models completeness+membership as one atomic.Pointer publication.
- [Phase 04]: 04-02: Delete keeps local NotFound for a listed-but-unarrived key (rule 14.5(6) names reads; A7); the Watch send-error fixture now opens its gate through the resolver (Rule 3).
- [Phase 04]: 04-03: Observe takes no store lock (it touches no record; the clock's mutex is the only synchronization) and Generation bumps exactly once inside install, the single convergence point of local writes, merge winners and restore replay.
- [Phase 04]: 04-03: handleDataSync absorbs each non-empty validated part's exact maximum entry version once, after header validation and independent of stream completion; empty and discarded parts absorb nothing (T-4-10).
- [Phase 04]: 04-03: StateStore gained Observe and Generation; every implementer delegates (engine, simnet nodeStore, seven pre-existing fakes, 04-01's white-box fake), while the map-holding egressStore returns a clock-less no-op Observe and a fixed 0 generation.
- [Phase 04]: 04-04: The empty-vs-data gate is selected once in wire from Engine.KeyCount() after the synchronous load; a node with any record (tombstone included) never consults startup_gate_empty, and production never derives one gate from the other (ADR 0006, T-4-14).
- [Phase 04]: 04-04: The stand harness derives --startup-gate-empty from an explicit --startup-gate in the extras, yields to an explicit empty gate, and appends the derived flag last so later-flag-wins parsing gives it precedence; the frozen R-03 30 s governs both timers without editing stand_red_test.go (T-4-13).
- [Phase 04]: 04-04: checkStartupGates validates positivity only and names the offending flag — a zero/negative gate exits 2 through the usage path; cross-parameter relations stay Phase 7's one-point validation (T-4-12).
- [Phase 04]: 04-04: The executed docs demo pair pins --startup-gate-empty=2s so a lost first empty listing cannot hold an empty demo node closed for the 60 s default and flake the verbatim README/EXAMPLES runs (Rule 3).
- [Phase 04]: 04-05: The egress queue is identity-only (handle, kind, key): enqueueResponse takes the identity directly and resolveQueuedResponse builds DataRequest/DataReplication from current store state at drain time, so a queued answer pins no record bytes and no producer looks a record up before the dedupe check (CR-02).
- [Phase 04]: 04-05: handleDataRequest probes StateStore.Version (presence only, no Data copy) and handleDataSync's local-newer branch enqueues the identity without a second Lookup; the drain owns the probe/read race and a vanished key still skips without consuming a pacer slot.
- [Phase 04]: 04-05: applyRemote branches three ways before ApplyRemote on one point Version probe — strictly older queues exactly one unicast reply carrying the held record and sends no Ack (no merge, no conflict, ledger untouched); equal Acks with no reply (the version was absorbed at install, and the retransmission drain needs it); newer/absent applies and Acks as before (PROTO-03).
- [Phase 04]: 04-05: The stale reply reuses the (handle, kind, key) egress identity path, so T-4-17's dedupe/amplification bound and CR-02's key-sized retention bound cover the new producer with no extra queue machinery; TestWaitIntegration_staleRecordNotAcked proves the real adapter tracker reports acked=false.
- [Phase 04]: 04-06: Restore reports the replayed records even when the final flush failed, and converts a non-ErrClosed flush failure into ErrDumpFailed wrapping the cause so the RPC maps to Unavailable while errors.Is still reaches the port error; ErrClosed passes through unchanged (shutdown is never a dump failure).
- [Phase 04]: 04-06: replayRecord's bool is the explicit install/skip signal, separate from the tombstone flag: ErrNotFound on a tombstone replay and ErrValueTooLarge on a live replay return the zero version, replayed=false and nil; skipped records are absent from RestoredRecord and never notified.
- [Phase 04]: 04-06: Dump-failed Set/SetIfAbsent/Delete notify the stamped record with track=false before mapError answers Unavailable; SnapshotLoad notifies Restore's list before mapping the error and skips the loop entirely on ErrClosed, so abandoned shutdown work is never announced.
- [Phase 04]: 04-06: The snapshot-load notify order is the replay's sorted-key order and the tests assert the exact key sequence, not a count, so a map-order regression fails; the no-op load notifies nothing and returns OK.
- [Phase 04]: 04-07: The test-only task's tdd=true produced unexpected GREEN because the asserted production behavior already existed; the test commit is the deliverable and the gate outcome is disclosed instead of fabricating a RED record or an unnecessary production change.
- [Phase 04]: 04-07: waitServing mirrors waitFor's visibilityDeadline, pollEvery cadence and hard-deadline clamp but names the last observed record on timeout, so an N=3 convergence failure reports per-store state rather than only the expectation.
- [Phase 04]: 04-07: The same-key LWW row waits for all three stores to hold A's version before B writes, so B's clock has observed it and B's store-stamped version strictly dominates; convergence on B's version therefore proves LWW, not wall-clock luck.
- [Phase 04]: Listing destinations are (node_id, run_id) identities, never datagram source handles; the zero identity is the broadcast round and the drain resolves the handle from the peer row (CR-01/T-4-25).
- [Phase 04]: One immutable listingBuild is shared by every FIFO while it is younger than sync_interval and StateStore.Generation is unchanged; any store mutation forces exactly one rebuild before the next listing (T-4-24).
- [Phase 04]: Only a restart advances the per-peer listing rotation; the first-seen round starts at chunk 0, preserving the pinned rotation contracts while restarts keep covering the key space.
- [Phase 04]: The first-seen Hello enqueues one targeted listing and falls through to the discovery flush instead of returning early, so R-02's pending-write drain bound survives PROTO-04.
- [Phase 04]: The notification guard compares against the tracked pending version, never store.Version: the store already holds the first notified version when NotifyLocalWrite runs, so a <= store comparison would swallow the normal path; absent tracking always proceeds and the first notification still broadcasts (PROTO-05).
- [Phase 04]: peer.seenRuns is a fixed 8-entry ring of previously current run_ids: an unseen run is a restart with one listing, while a seen non-current run is a counted, rate-limited configuration error dropped with no listing and no row/liveness change; nine runs keep the first, the tenth evicts it (PROTO-06).
- [Phase 04]: All three duplicate-node_id paths route through one per-(node_id, class) rate-limited WARN helper, and the discard totality model now mirrors the bounded seen-run history so counter conservation stays exact.
- [Phase 04]: 04-10: The N=3 simulator scenario asserts same-seed TraceDigest equality in addition to final convergence, so scenario-local determinism is pinned where listing traffic grows.
- [Phase 04]: 04-10: runThreeNodeScenario threads *testing.T so the mid-partition teeth (node 3 must not have converged) are asserted in the factored body; the returned cluster+digest contract is as planned.
- [Phase 04]: 04-10: The joiner's partial-listing window is observed through TraceDigest DataSync deliveries and the exported SyncArrived/ListingComplete reads — no test-only production counter.
- [Phase 04]: 04-10: ReplaceNodeWithState onto a stale snapshot opens no gate — both completeness reads stay false at the restart instant and node A's newer value survives (F14 at simulator level).
- [Phase 04]: 04-10: 14.9's decision index names TestScenario_threeNode; the real-socket three-node test is recorded as the wiring proof.
- [Phase 04]: 04-10: First-seen listing traffic moved no simnet extension bound or determinism expectation; the audit is recorded in comments (same-seed equality, budget armed after discovery listings).
- [Phase 04]: 04-11: listing drift is pinned as clock-only — MergeStats stays zero while the next local write strictly beats the listed max version, so an Observe-skipping implementation fails (OQ6).
- [Phase 04]: 04-11: the WARN limiter is pinned per (node_id, class) by counting class+node pairs in the captured Log — two classes from one peer and a second peer's first occurrence each warn once.
- [Phase 04]: 04-11: the 12 declared residual files needed no edit — every one was already adapted by its owning semantic plan or compiles unchanged through shared fakes; task 2's deliverable is the recorded all-green run and the adaptation table.
- [Phase 04]: 04-12: every multi-node claim carries a code or test reference; residual windows are labelled accepted/residual with Phase 6/7/9 owners.
- [Phase 04]: 04-12: OPERATIONS' stale 'reads are never gated by startup' sentence was corrected while documenting --startup-gate-empty, matching ADR 0005/D-01's closed reads-and-writes interface.
- [Phase 04]: 04-12: exactly the five declared deviation rows (S-HLC-04, S-HLC-10, S-PROTO-18, S-ALG-ST-06, S-ALG-RC-05) were flipped to implemented with current evidence; PROTO-09 remains for 04-13 to close.
- [Phase 04]: 04-13: the two R-03 rows flipped green by a TSV data change only; the frozen tests are byte-identical and both P-01 rows stay red (D-01).
- [Phase 04]: 04-13: the periodic-sync-disabled kill list was re-selected after a 16-seed scan — first-seen listings now heal 15 seeds, leaving 848612892191557850 as the verified kill; the four-seed control arm is unchanged.
- [Phase 04]: 04-13: CR-01 is closed naming plan 04-08/commit 1e06c58 and CR-02 naming plan 04-05/commit e42a20a, each with its named regression tests in the Phase-3 disposition ledger.
- [Phase 04]: 04-13: the Phase-4 gate ran green on the final tree (tidy, lint, tests, red-runner with mutants, buf lint/breaking, determinism, bench probes, darwin build/vet); the one IN-03 wall-clock ratio flake was proven standalone/rerun and deferred to Phase 8 without touching any bound.
- [Phase 05]: 05-01: The value limit is one exported storage.ValueMaxBytes; Config.ValueLimit serves the pre-store decode and NewStore normalizes through it, so the decode bound and the enforcement point cannot diverge (PERSIST-01, D-01).
- [Phase 05]: 05-01: Open keeps its frozen (*Engine, int) signature — the decode uses storeCfg.ValueLimit() before the clock seed and NewStore; the engine captures Store.ValueMaxBytes() for runDump and Restore.
- [Phase 05]: 05-01: An over-limit encode is a wrapped storage.ErrValueTooLarge naming the key (a store-invariant tripwire), never a silent skip; a non-positive --value-max-bytes exits 2 through the usage path instead of falling back to the default.
- [Phase 05]: Decode returns DecodeStats (Checksum, Version, ValueLimit, Malformed) with Total(); Open keeps its frozen (*Engine, int) shape by returning the total.
- [Phase 05]: The dump reader bound is maxDumpLineBytes(), derived from the format's datagram and key ceilings and independent of valueMaxBytes, so a config-shrunk load classifies records instead of faulting.
- [Phase 05]: format_version stays 1 with an optional record_count: fewer record lines than the count is an incomplete generation, more is tolerated, and an absent field is the documented v1.0 legacy path.
- [Phase 05]: 05-03: The join order is (version, content): identical Deleted+Data is a full no-op, differing content resolves remove-wins first, then the greater Data by bytes.Compare; a tie-break win or loss reports exactly one Conflict pairing the equal version bytes
- [Phase 05]: 05-03: storage.RecordsEqual (version, deleted, Data; checksum derived) is the one content comparison shared by MergeChanged's join and repl.applyRemote, so the merge law and the wire rule cannot drift
- [Phase 05]: 05-03: MergeChanged returns (conflicts, changed, drift, err) and Merge delegates; Engine.ApplyRemote dirties on changed, replacing the Get probe and the Winner==rec.Version heuristic (research Pitfall 5)
- [Phase 05]: 05-03: applyRemote Acks identical content early, applies differing content through the store, then answers from the stored record: incoming win Acks, stored win queues exactly one unicast DataReplication and no Ack
- [Phase 05]: 05-03: ADR 0012 records the deviation from the Spec's random tail with the version layout, zero filler (S-HLC-07) and PFXD-01 as explicit non-changes; cross-node convergence stays 05-07's proof on the NotifyInstalled seam
- [Phase 05]: EncodeTo is the single dump encoder and Encode is its bytes.Buffer wrapper, so the streaming and buffered paths cannot drift
- [Phase 05]: SaveStream's fill runs exactly once on the caller's goroutine and must not retain the writer; the adapter owns the temp lifetime and removes it on any fill error
- [Phase 05]: The adapter buffers the streamed dump through a named 64 KiB bufio writer, coalescing the codec's per-line writes without holding the dump
- [Phase 05]: Save leaves port.DumpStore; FileStore.Save stays as a byte-fixture convenience over SaveStream and the engine has no byte-slice save path
- [Phase 05]: Load classifies only in 05-04: absent candidates are non-failures, present-but-unreadable ones carry path and cause, and preservation/sweep stay for 05-05
- [Phase 05]: peak-live-B samples runtime.MemStats HeapAlloc at the fill's end, the peak moment when the one Records() copy and the adapter's bufio are both live
- [Phase 05]: 05-05: Preservation is load-time: every rejected regular generation is renamed to .corrupt-<UTC>[-N] before any save, so the first post-backup rotation finds no main and .prev stays byte-identical (PERSIST-02, T-5-12).
- [Phase 05]: 05-05: The temp marker is .tmp~ (outside the snapshot charset), superseding the research dump-* prefix; the sweep selects by marker on regular files, age-bounds at one hour, and excludes the configured main/.prev/.corrupt-* exactly in the dump directory (PERSIST-09, T-5-14).
- [Phase 05]: 05-05: A Backup start always carries exactly one main failure — fs.ErrNotExist when the main was absent — so the engine's single WARN needs no synthesised cause.
- [Phase 05]: 05-05: ensureDir syncs a created directory's parent before the temp write; existing directories keep the five-step TEST-03 order, and CreateSnapshot uses the same helper (D-08).
- [Phase 05]: 05-05: Only the directory fsync tolerates ENOTSUP/EINVAL; file-data fsync keeps every error, and classifySaveError wraps every SaveStream error with port.FailureClass while preserving errors.Is (T-5-15/T-5-16).
- [Phase 05]: 05-06: LoadStats is the engine's own report mapped once from port.LoadResult in Open; Open keeps its frozen (*Engine, int) signature and New-built engines report the zero value
- [Phase 05]: 05-06: Dirty exactly when the load outcome is not Main — Prompt/Backup/Unusable flush before a Snapshot; a Main load with discarded records stays clean to preserve the PERSIST-10 evidence window
- [Phase 05]: 05-06: markDirty owns the non-blocking changed nudge; ApplyRemote's duplicate is gone and the restore replay's mid-error path is woken (Phase-4 New-1)
- [Phase 05]: 05-06: Load log attributes are candidate-keyed — Backup scans Failures for GenMain (absent main names fs.ErrNotExist) and Unusable iterates Failures, so zero/one/two entries never panic or produce a phantom entry
- [Phase 05]: 05-06: A value-limit skip is counted, logged once with count and value_max_bytes, published as supkv_dump_records_discarded_total{class=value_limit}, and left on disk until normal rotation displaces it
- [Phase 05]: Config.Now/After default to the wall clock only inside normalized(); every other engine read and timer uses the injected pair, so the simulator owns the dump loop's time (PERSIST-12)
- [Phase 05]: A failed dump doubles 100 ms to the 5 s cap with the retry instant fixed at the failure; a pending retry suppresses the debounce/deadline channels so the 1 s MaxWait cannot defeat the cap, and a success resets the ladder
- [Phase 05]: DumpStats.Errors stays the failed-attempt total while DiskFull/Permission are class subsets; the exporter renders other as the clamped residual, so every scrape carries all three labels
- [Phase 05]: Health stays SERVING under a persistent dump failure: the failure surfaces as Unavailable writes, one static ERROR per attempt with the class, and the class counter; the SERVING transition remains gate-only
- [Phase 05]: The simulator Scheduler takes a lock (Schedule/ScheduleAfter/Now) because the engine arms timers from its own goroutine; RunUntil remains the single driver and releases the lock around callbacks
- [Phase 05]: Cluster.NotifyInstalled runs the production author notification without re-Setting, so the engine's stamp and the confirmed record survive; exact ladder instants stay in synctest while the simulator pins the interaction
- [Phase 05]: A waiter is claimed at resolution, never at capture, so a dump hanging after capture cannot block the caller's timeout; the timeout removes its waiter and counts the outcome
- [Phase 05]: Flush() is FlushTimeout(0); one deadline bounds both the flush-channel send and the reply, and Restore keeps ErrUnconfirmed distinguishable from ErrDumpFailed
- [Phase 05]: The binary bounds shutdown with FlushTimeout(shutdownGrace) plus a grace-bounded engineDone wait that abandons a stuck dumper with one WARN; ErrUnconfirmed is total on the RPC path and published as supkv_dumps_unconfirmed_total
- [Phase 05]: 05-09: Engine embeds *confirmer instead of a named conf field, because the frozen white-box tests read Engine.mu/Engine.waiters and call Engine.markDirty and no test edit is allowed; promotion keeps the state in the role and the call sites byte-identical
- [Phase 05]: 05-09: Each state-owning role keeps one mutex, never nested; rearm reads the confirmer epoch and the scheduler retry state under two sequential locks
- [Phase 05]: 05-09: New constructs one dumpStats holder and hands the same pointer to the scheduler and the confirmer, so the timeout counter and the dump counters are one published value
- [Phase 05]: 05-09: The split adds no exported name and no interface; the five role files are internal-only, Engine stays the composition root
- [Phase 05]: 05-10: The runbook's persistence additions are pinned one anchor literal per statement; the count gate is line-based (grep -c), so the anchor block's doc comment names the mandated phrases in prose while each literal still carries its own row
- [Phase 05]: 05-10: TestOperationsDocFamilies extracts every supkv_ name from docs/OPERATIONS.md and requires it in internal/metrics/metrics.go, so a documented family with no exporter counterpart fails without a hardcoded duplicate list
- [Phase 05]: 05-10: The memory model states peak live (one Records() copy + one record line + the 64 KiB writer) and names the copy's read-lock writer pause as residual; B/op churn is explicitly a different, larger number
- [Phase 05]: 05-11: S-PAR-05 and S-PERS-09 flipped to implemented and S-PERS-08 kept implemented with the port.LoadBackup outcome and one-WARN evidence; S-HLC-07 kept accepted-deviation with the ADR 0012 link; S-DM-04/05/08 cite ValueMaxBytes, MergeChanged/join/RecordsEqual and the tie-break tests
- [Phase 05]: 05-11: the preamble names ADR 0012 separately from the 0001–0011 / 14.1–14.11 mapping because 0012 records P-07, not an owner decision; the diff-scope gate proved exactly the eight rows changed and task 2 made no no-op commit
- [Phase 05]: Both frozen P-01 tests ran directly before the flip (engine PASS 1.06 s; stand PASS 4.18 s with both 63000- and 64000-byte subtests), so the green rows rest on a genuine pass; the runner then recorded observed=pass,run for both.
- [Phase 05]: The flip is exactly the two red->green data fields in scripts/red-expectations.tsv; the header and the other five rows are untouched, and the runner plus both frozen tests are byte-identical to the phase-start base 4f81fd7 (diff exit 0).
- [Phase 05]: The Phase-5 gate ran in order on the committed flip: lint, darwin and bench-smoke were green on first invocation; the first make test hung 600 s in TestScenario_dumpNotify and the package-alone run hit the known ratio flake, while the full rerun was green (repl 276 s) with no test or bound changed.
- [Phase 05]: Both load-sensitive flakes (TestScenario_dumpNotify virtual-time harness hang, TestCore_keyCountCostFlat ratio repeat) are deferred to the Phase-8 flake owner with their standalone/rerun evidence; the TSV-only diff cannot have caused either.
- [Phase 06]: The legacy-path alias package keeps package apiproto and omits the machine-readable Deprecated: marker: the marker made staticcheck SA1019 fail the byte-frozen red tests, so deprecation is stated in prose plus a scoped, reasoned //nolint:revive set instead
- [Phase 06]: 06-01 Tasks 1+2 landed as one atomic commit (amended once to be lint-clean), so no commit ever has every importer naming the deprecated alias; the RED record is the compile-clean TestSchema_listPageFields assertion failure because the go-test RED classifier refuses [build failed] runs
- [Phase 06]: The public API schema lives at api/supkvapi/v1 (module root api) with a one-file FILE_SAME_GO_PACKAGE waiver proven load-bearing; sup-kv-public-api-no-internal denies github.com/Djarvur/sup-kv/internal under api/ and was probe-verified
- [Phase 06]: Write-time capture: Set/Delete call the WritePeers seam exactly once immediately before the engine write and hand the same cloned slice to the notification and the wait report, so a peer discovered mid-wait can never move the total (A-01).
- [Phase 06]: Zero captured peers and a nil WaitReplicated seam both report the honest (false, 0, 0); the zero-peer path still calls the seam with total 0 as the tracker release point (T-06-05).
- [Phase 06]: The core seeds the tracked wait set from the captured ids intersected with the live peer table and emits WaitSetEmpty for a stale non-empty capture; the discovery-flush split and tracker promise-set enforcement stay with plan 06-11.
- [Phase 06]: Core.PeerIDs/PeerGeneration feed a generation-gated Adapter.KnownPeerIDs atomic mirror; Adapter.KnownPeers stays for metrics only and ServerOptions.KnownPeers is removed.
- [Phase 06]: 06-03: examples/client is a nested Go module (require github.com/Djarvur/sup-kv v0.0.0 + replace ../.. ) importing only the public api/supkvapi/v1 package plus gRPC; go mod tidy reclassified protobuf as indirect, and the committed tidy form is the canonical one. It is gated by one make client-example target and a dedicated client-example CI job (APIS-06).
- [Phase 06]: 06-03: The example README states the last-read-value cache is consumer-library territory per ADR 0007 (14.7) and lists the deliberate omissions (no auth, retries, cache or snapshot RPCs). APIS-06's REQUIREMENTS.md checkbox stays pending because sibling plans 06-09/06-10/06-13 also declare it (#2388 shared-ID gate).
- [Phase 06]: 06-03: /examples/client/client is gitignored because the plan's literal recipe (go build ./... inside the nested module) writes the executable beside main.go; the target itself is unchanged. The stale gsd-plan-head-before-06-03 ledger was reset to fc4ab65 before measuring (06-01 precedent).
- [Phase 06]: 06-04: mapError is one ordered total table — persist.ErrDumpFailed before the port-class rows (a write answers Unavailable with the pinned sentinel, while CreateSnapshot surfaces its class as ResourceExhausted/PermissionDenied), persistfile.ErrNoDump before the fs rows, context through the matched sentinel — and the fallback logs the detail via slog then answers exactly "internal error".
- [Phase 06]: 06-04: CreateSnapshot classifies its ensureDir, non-ErrNoDump main-dump read and atomic-save failures with classifySaveError while passing ErrNoDump through unchanged; FailureClassOf(ErrNoDump) stays FailureOther with no port.ClassedError, so the mapper can tell no-dump from a filesystem fault.
- [Phase 06]: 06-04: The RPC path-shape matrix builds one DumpStore fake per errno variant whose save/create/read seams all return a path-bearing *fs.PathError; SnapshotCreate's forced flush fails on the save seam before CreateSnapshot, so that row proves the ErrDumpFailed constant while the class mapping is proven by the adapter test and the total table.
- [Phase 06]: 06-04: Task 2's test-only tdd=true task was an investigated unexpected GREEN — Task 1's files implement exactly the rows it pins, so the test commit is the deliverable (04-07 precedent) and no RED was fabricated.
- [Phase 06]: 06-04: APIS-03's checkbox stays pending because requirements ready-ids reports it blocked (the #2388 shared-ID gate); the stale plan-head ledger was reset to dabd55e before measuring, so actuals.commits: 4 is measured.
- [Phase 06]: APIS-01: the adapter tracker's captured total and wants set are authoritative; WaitReplicated reports the acked-set intersection with that membership and requires count == captured total, so a stranger Ack is inert even on a Drained result.
- [Phase 06]: APIS-01: the adapter's zero-peer release answers (false, 0) (retiring 06-02's (true, 0) benign verdict) while still deleting the notify-time tracker; the handler's public (false, 0, 0) is unchanged.
- [Phase 06]: APIS-01: the discovery flush is broadcast-only — it re-arms the ladder and queues the record send but never inserts the discovered peer into a wait-set or the per-peer ledger (IN-05 closes).
- [Phase 06]: APIS-01: the 20 ms displacement target is proven structurally by the displacing handler call's ResultSuperseded Outbox event; the real-socket regression keeps a generous 250 ms CI-safe bound (RESEARCH Pitfall 2).
- [Phase 06]: APIS-01: plan 06-11 Task 2 is regression pinning — its tests pass on the Task 1 tree, so no second RED commit exists; the behavior change and its RED evidence live in Task 1 (plan frontmatter is type: execute).
- [Phase 06]: 06-05: the two List modes share one LiveVersions call; Lookup is called only for entries the page returns, so the page_size-0 legacy listing copies no value bytes at all
- [Phase 06]: 06-05: the page token is the RawURL base64 of the last examined key — a strict-greater position, never an index or snapshot; a token over storage.MaxKeyBytes or undecodable is InvalidArgument, a token without a positive page_size is InvalidArgument
- [Phase 06]: 06-05: listEnvelopeOverheadBytes 1382 is derived term by term (1366-char token ceiling + tag/length 3 + gRPC prefix 5 + repeated-field framing 8); plan 06-07 adds it to listPageByteBudget 6 MiB for MaxSendBytes and must not redefine either
- [Phase 06]: 06-05: the byte budget stops before an over-budget entry and leaves it as the next page's first, while the first entry is always included, so a page is never empty non-terminal and progress is structural
- [Phase 06]: 06-05: the page walk takes a lookup function so the view-to-lookup race (tombstone or vanished key) is pinned deterministically in list_internal_test.go instead of raced through the RPC surface
- [Phase 06]: 06-05: APIS-02's checkbox stays pending because requirements ready-ids reports it blocked (the #2388 shared-ID gate among 06-01/06-05/06-07)
- [Phase 06]: 06-06: storage.Config.OnChange carries the just-incremented install generation; persist.Config.OnConfirmed publishes the generation read immediately before Records() after confirmWaiters and never on a failed dump, so the watermark always describes the saved snapshot.
- [Phase 06]: 06-06: WatchHub holds ingress installs in a pending queue bounded at watchIngress and releases only the leading gen <= confirmed prefix, dropping what it dispatches, so delivery is exactly-once in install order; Confirm is a CAS maximum with a buffered(1) wake, so a repeated or lower confirmation delivers nothing.
- [Phase 06]: 06-06: either overflow flavor (full ingress or full pending) sheds every subscriber with ResourceExhausted and clears pending; resubscription re-snapshots (D-02), and the bounded-policy pin plus startup suppression and Unavailable shutdown remain 06-12's expansion slice.
- [Phase 06]: 06-06: the subscribe snapshot stays the current in-memory state (what Get sees) while the live feed is the durable change feed; the asymmetry is the documented D-04 contract, not a defect.
- [Phase 06]: 06-06: RED is the two compile-clean integration tests (held dump; dump-failed-then-confirmed); the tap-generation and hub-gate unit tests need symbols this plan introduces, so they landed with GREEN (06-05 store-view precedent). APIS-04's checkbox stays pending on the #2388 shared-ID gate among 06-06/06-09/06-13.
- [Phase 06]: 06-12: Open captures the caller's OnChange, substitutes a no-op for a nil tap and gates it with a synchronous loading bool around mergeRecords, so a startup dump is never announced; the nil-tap regression pins the storage.Config{} contract.
- [Phase 06]: 06-12: shutdownSubscribers writes a fresh codes.Unavailable status ("watch hub is shutting down") before closing each channel, and the late-subscribe refusal uses the same watchShuttingDown helper, so a dying node never looks like a clean OK.
- [Phase 06]: 06-12: the pending bound (watchIngress) is pinned by an observable shed+resubscribe test; the unexported pending slice is unobservable from the external test package, and a snapshot-keyed zero-version flood keeps post-shed residue suppressed.
- [Phase 06]: 06-12: RED targets are the two missing behaviors (startup suppression, shutdown status); the normal-write, pending-bound and deletion/order tests are regression pins of the 06-06 gate and pass pre-change by design. APIS-04 stays pending on the #2388 shared-ID gate.
- [Phase 06]: 06-12: no post-load confirmed-watermark initialization was added — with the tap suppressed no loaded event is ever pending, and the first successful dump advances the hub watermark past the loaded generation.
- [Phase 06]: APIS-05: MaxRecvBytes = storage.MaxKeyBytes + the effective configured value limit + apiEnvelopeOverheadBytes (63 846 at the default, 66 814 at the 64 408 datagram limit), never the default constant; MaxSendBytes = listPageByteBudget + listEnvelopeOverheadBytes, and apiEnvelopeOverheadBytes is an alias of 06-05's listEnvelopeOverheadBytes, not a second definition.
- [Phase 06]: APIS-05: Server.Shutdown calls the standard health server's one-way Shutdown (never Resume); the binary calls node.api.Shutdown() before apiCancel and no longer flips health directly, so a gate-resolver backstop firing after shutdown began cannot return SERVING (T-06-23, RESEARCH Pitfall 9).
- [Phase 06]: APIS-05: the transport-limits harness knob is opt-in (nil keeps the plain grpc.NewServer()), the receive-ceiling test proves ResourceExhausted at the transport versus InvalidArgument at the handler, and the one-way test waits for the backstop through a successful Set rather than a sleep; all three tasks carry machine-valid RED records despite their new symbols needing compiling skeletons.
- [Phase 06]: Full rollback is one deterministic sorted union of snapshot keys and current live keys absent from the snapshot; a snapshot key replays through replayRecord and an absent live key is deleted with a fresh tombstone that joins the returned report (D-07, ADR 0001).
- [Phase 06]: restoreChanges counts an extra current live key as a change but never an extra current tombstone: an already-rolled-back key burns no version and produces no record (RESEARCH Pitfalls 13-14).
- [Phase 06]: Rollback tombstones join RestoredRecord (nil Data, Deleted=true), so SnapshotLoad's existing notify loop broadcasts them with zero handler wiring, including when the final dump fails (PROTO-08 / T-06-26).
- [Phase 06]: A rollback delete rides storage.Store.Delete, whose ErrNotFound is a benign concurrent skip that burns no version; concurrent writes after the rollback set is computed are ordinary LWW, not snapshot isolation.
- [Phase 06]: 06-08 tasks 2 and 3 were investigated unexpected GREENs (behaviour landed in Task 1's GREEN commit and the handler needed no change), so their test commits are the deliverables; gocognit/nestif/revive/noinlineerr forced the replay-pass helper split (04-07 / 06-04 precedent).
- [Phase 06]: 06-09: docs/API.md and TestAPIDoc land as one contract pair — the anchor table reads the repo file and pins one literal per mandated statement, so section deletion or heading drift fails the normal test gate (T-06-29); the doc comment states the honest scope (presence and drift, not sentence truth).
- [Phase 06]: 06-09: the error table follows the mapper's first-match order and adds the handler validation rows beside it; the ordering rationale (pre-built status pass-through, ErrDumpFailed before the port classes, ErrNoDump before the fs rows, context through the matched sentinel, logged constant fallback) is part of the documented contract.
- [Phase 06]: 06-09: every limit is documented as implemented (caller-bounded wait, non-linearizable rollback, current-memory snapshot) and the Phase 9/DOCS-05 remainder is named pending; requirements-completed stays empty on the #2388 shared-ID gate, with 06-13 closing the checkboxes.
- [Phase 06]: 06-10: the public-path sweep re-derived the five SPEC-DEVIATIONS proto links line by line (S-ALG-CU-11 :63-80, S-SEM-04 :63-69, S-IMP-01 :66, S-IMP-02 :67-68,86-87, S-IMP-03 :39) - a pure path swap would have cited neighbouring declarations.
- [Phase 06]: 06-10: EXAMPLES demonstrates both List modes in one executed block - a page_size-1 page carrying records plus nextPageToken ZGVtby9oZWxsbw, a resumed page returning the last key with the empty token omitted, then the legacy unset-page_size keys-only listing.
- [Phase 06]: 06-10: 04-MULTI-NODE names the WritePeers write-time set, the zero-peer (false,0,0) answer and the inert outside Ack, and states IN-05 is closed by APIS-01 (the discovery flush broadcasts and re-arms but never grows the awaited set or ledger).
- [Phase 06]: 06-10: S-PERS-12 flips to implemented with the sorted-union replay/rollbackKey/restoreChanges links and the engine plus RPC rollback tests; S-PERS-13 and every PROTO-08 row stay for Phase 9.
- [Phase 06]: 06-10: AGENTS.md gains api/supkvapi/v1 and examples/client in the Layout, the public-schema/internal-replication dependency-policy sentence and the three-rule depguard deviation 1 ('These rules live in .golangci.yml'); deviations 2-3 and the Go version policy are byte-identical.
- [Phase 06]: Phase gate: the frozen evidence hashes to its phase-start literals and the red-runner is 7/7 with both mutants killed; the gate found a stale F2c assertion (pre-06-07 InvalidArgument vs the documented transport ResourceExhausted) which was fixed with both layers pinned
- [Phase 06]: TestStand_oversizeKeyF2c reads the rejected key's absence through List, not Get: a Get of the 70 000-byte key is itself over MaxRecvBytes and would prove nothing about the store
- [Phase 07]: 07-01: udprepl split into socket/codec/loop/mirror/adapter in one package; SentByType moved to internal/app/repl with the udprepl alias and accessor retained, so metrics no longer imports the socket adapter.
- [Phase 07]: 07-02: internal/config owns Parse/Validate/Usage and stays outside internal/app because it reads the environment and imports the adapter ceilings; the no-I/O depguard rule must not cover it.
- [Phase 07]: 07-02: Validate walks one 12-rule catalog and errors.Join's every failure under its rule name; the pre-existing sentinel substrings (`datagram ceiling`, `value-max-bytes must be positive`, `startup gate must be positive`, `sync-chunk-bytes exceeds the datagram ceiling`) stay matchable.
- [Phase 07]: 07-02: the max-clock-lead default moved to 2 m in the binary and the core in the same commit as the drift-vs-lead rule, because the 1 m default would fail its own validation and violate the 'every default stays accepted' prohibition.
- [Phase 07]: 07-02: Usage resolves defaults from the process environment (preserving SUPKV_MAX_CLOCK_LEAD=90s --help showing 1m30s) while Parse stays pure over its getenv seam; repl.MaxSyncResponseRate is exported as an int so the cap is never a literal in config.
- [Phase 07]: 07-02: persist.EncodeTo rejects an out-of-domain value limit before the header write wrapping storage.ErrValueTooLarge, closing IN-04; version.Validate and the core docs state the 2 m default and the max_clock_lead >= drift_warn + 1 ms relation.
- [Phase 07]: 07-03: Three named depguard rules (adapters-not-in-core, metrics-no-adapters, grpcapi-no-repl-metrics) enforce cmd -> adapters -> core/ports; adapters are named individually with !$test so port -> app/storage/version and internal/testutil stay legal
- [Phase 07]: 07-03: The grpcapi -> persistfile / *persist.Engine composition edge is explicitly allowed in .golangci.yml with its written justification (two snapshot sentinels, 12 engine methods, Watch hub Records/Set) rather than hidden behind a //nolint
- [Phase 07]: 07-03: A depguard rule is proven by a used-import probe under the full config; a blank-import probe is masked by golangci-lint uniq-by-line (revive blank-imports shares the import position) — the Phase 6 WR-03 lesson generalized
- [Phase 07]: 07-04: run(cfg, rt) int is the injectable lifecycle seam over a runtime of function values; productionRuntime assigns armSignals/signal.Stop/wire/serve/shutdown directly (matching directions), and one switch gives a failed final dump (3) precedence over a Serve error (1) over 0.
- [Phase 07]: 07-04: startupError carries the replication_bind/grpc_listen/metrics_listen stage and run logs one static Startup-failed ERROR with stage+detail, so a fatal startup survives SUPKV_LOG_LEVEL=error; the stdlib runtime import is aliased goruntime because a package-block type cannot share its name.
- [Phase 07]: 07-04: the final-dump process test writes against the blocked dump path (Set answers Unavailable, engine stays dirty) because a fresh node's Prompt-load dump succeeds inside the 50 ms debounce and flushDump skips a clean engine; a regular file in place of the directory makes os.CreateTemp fail with ENOTDIR for every uid.
- [Phase 07]: 07-05: the drift signal is the received reading's lead — Clock.Update compares remoteL with the wall clock, Tick keeps the local-versus-wall diagnostic, and the core counts only drift > 0.
- [Phase 07]: 07-05: a DataSync part counts one drift warning from maxEntryVersion, never per entry, so a listing cannot inflate the counter (T-07-16).
- [Phase 07]: 07-05: the drift rows are seeded through localWrite and the tie-break row asserts only drift, leaving 07-06's pending-only conflict definition unconstrained.
- [Phase 07]: 07-06: supkv_merges_conflicts_total counts only the displacement of a pending (unacked) local write, walked from c.pending after a successful apply — the store's per-outcome Conflict slice is no longer read (ADR 0004, OBSV-03).
- [Phase 07]: 07-06: an equal-version tie-break win that displaces a pending write counts 1 and emits Superseded; a loss leaves the entry tracked — the old loop handled only strictly greater versions.
- [Phase 07]: 07-06: one post-apply stored-record read yields incomingWon and serves both the reply verdict and the displacement count, so they cannot disagree.
- [Phase 07]: 07-06: the apply-failure WARN rides the shared (node_id, apply_failure) limiter and names only the quoted handle; the key name is gone (T-07-17, T-07-18).
- [Phase 07]: 07-06: TestStand_metricsByEffect stops node B before A's write and drives the displacement with one crafted wire-valid higher-version DataReplication, keeping an exact conflicts == 1 assertion.
- [Phase 07]: 07-06: the chaos scenario logs the aggregate conflict count instead of asserting it — a definition is proven by constructed unit and stand rows, not a scheduler-dependent total.
- [Phase 07]: Gate opening is narrated by a typed reason fired inside the openGate CAS after the SERVING flip, so the event and the writable transition are one single-shot fact.
- [Phase 07]: Peer lifecycle and drift cross the core boundary as typed events (PeerEvent/DriftEvent); only cmd renders them, so internal/app stays slog-free and the transition matrix is unit-testable.
- [Phase 07]: Drift rides the shared allowWarn slot keyed (node_id, clock_drift) while MergeStats.DriftWarnings stays per record; nil OnDrift falls back to the prebuilt Log string. The log key is handle because sloglint forbids source.
- [Phase 07]: A start logs exactly one load-outcome record: INFO Dump load for prompt/main and the existing Backup WARN/Unusable ERROR with the same outcome/generation/records/discarded attributes; generation none is derived from the outcome, never the zero enum.
- [Phase 07]: 07-08: unbound_listing is its own discard class (counter, rate-limited WARN without 'configuration error', metric label); the own-node-id, second-handle and alternation paths keep duplicate_node_id
- [Phase 07]: 07-08: the decoded ProtoVersion-mismatch branch now WARNs through the shared limiter like the adapter verdict; the catalog level matrix calls the named renderers (logAdapterWarn, logDumpFailure extracted for the Log and OnDumpFailure seams) and asserts level+msg
- [Phase 07]: 07-08: docs/04-MULTI-NODE.md's CR-01 paragraph was corrected to unbound_listing (no phase-7 plan owned the file); docs/OPERATIONS.md's label catalog stays with 07-10's explicit action item
- [Phase 07]: Sent accounting counts a datagram only after WriteToUDP accepts it; encode, resolve and write failures count nothing
- [Phase 07]: The four routed Phase-4 counters are unlabeled, mirrored post-event and rendered at zero when unseen
- [Phase 07]: Linux RSS reads /proc/self/statm field 2 x page size; the !linux file reports ru_maxrss as the documented development-only peak
- [Phase 07]: 07-10: the drift runbook states only the shipped received-lead direction; the anchor plus the 2 m max_clock_lead default and the drift_warn < max_clock_lead rule keep the WARN reachable
- [Phase 07]: 07-10: operationsForbidden matches whitespace-normalized text on both sides, so a reflowed stale claim (the Main load no-log sentence) cannot survive
- [Phase 07]: 07-10: S-SEM-07 keeps superseded-by-amendment and S-IMP-08 keeps accepted-deviation(ADR 0008); only their notes/evidence changed (pending-local displacement; honest conflict and drift counters)
- [Phase 07]: 07-10: AGENTS names internal/config as composition-level outside the app no-I/O rule and lists the three additive depguard rules plus the written grpcapi composition exception
- [Phase 07]: 07-11: The first full red-runner attempt failed only on an ordinary stand test (TestStand_persistLoadUnusableSingle, 'SIGKILL <pid>: os: process already finished') while all seven expectation rows were green; the full gate re-ran unchanged and passed, so the transient was recorded in deferred-items for Phase 8 instead of being fixed in a no-source gate plan.
- [Phase 07]: 07-11: buf-breaking ran the real FILE-policy comparison against refs/remotes/origin/master (d6b2a75) because supkv.proto is now published there, not the D-08 deferral.
- [Phase 07]: 07-11: The phase gate produced zero task commits — it changes no source and every gate passed on the committed tree, so the record lives in 07-11-SUMMARY.md (the 01-04 precedent).
- [Phase 08]: Phase 8 thresholds are one compiled const block in internal/testutil/perf.go; no env parsing and no threshold file (D-06).
- [Phase 08]: The perf burst row reuses the frozen red generator read-only; TestPerfBurstBoundsMatchRedContract pins the perf thresholds to the frozen red constants.
- [Phase 08]: make perf is manual and Linux-only: SUPKV_PERF=1, .tmp/perf dump base, -p 1, no -race, then the functional stand line; CI keeps the threshold-free bench-smoke.
- [Phase 08]: pendingByKey is a single-slot map[string]pendingKey, not a set-valued index: the one-pair-per-key invariant is total, so a hidden second pair is a bug the fixed-seed differential names rather than a state the index accommodates
- [Phase 08]: PendingVisits counts pending entries INSPECTED, never handler calls or emissions: that reading makes the pre-index 2N/N counts and the post-index zeros the same measurement, and the field comment obliges any new pending-set loop to increment it
- [Phase 08]: 08-02 RED evidence is the counter-instrumented split (write 2N, apply N red; idle-Advance tripwire green): referencing pendingByKey in a RED commit would have been a package build failure, which #3770 refuses as INVALID_RED, so the invariant/differential proofs landed in GREEN
- [Phase 08]: PERF-05 remains unmarked in REQUIREMENTS.md: 08-02 lands the core fix and deterministic proofs, while 08-04 (docs) and 08-05 (paired wall-clock gates) still cover the requirement; requirements ready-ids still reports it blocked
- [Phase 08]: The paced replay seam carries storage.Record{Version} at both hops (gRPC LocalReplayNotify and the adapter enqueue); the nil-ReplayNotify fallback and the live NotifyLocalWrite clone stay unchanged, because the live seam's contract still carries the full record.
- [Phase 08]: The replay allocation gate is an assertion, not a report: 60 KiB record, AllocedBytesPerOp() ceiling 1 KiB; RED measured 65536 bytes/op, GREEN 0, and the passing run logs the measurement.
- [Phase 08]: The key-count gate asserts exact consumer-seam call vectors at both sizes: DataRequest = 1 Version + 1 Lookup, DataSync = 1 Observe + 16 Version, with Versions/Records/ApplyRemote at zero.
- [Phase 08]: The counting double embeds the external test package's delegating stateStore and adds no production surface; a StateStore method addition becomes a compile error in the double.
- [Phase 08]: Task 2's three PendingVisits gates are tripwires that pass on arrival; 08-02's Tests 1-2 remain the RED evidence for the write/apply flatness.
- [Phase 08]: The paired PERF-05 test SKIPs outside make perf, logs the hardware block and both arms, and reads its bound from testutil.PerfAbsentPeerRatio.
- [Phase 08]: The store-keys and unacked metrics are reported after the measured loop because ResetTimer clears b.extra.
- [Phase 08]: The port allocator splits claim (returns an error) from pick (turns it into tb.Fatalf): testing.TB cannot be implemented outside the testing package, so this is the only way to assert the empty/full band contract in-process and pick can never return zero
- [Phase 08]: TCP probes use net.Listen and UDP probes ListenPacket: ListenPacket rejects the tcp network, which the first GREEN run exposed; one probeBind helper branches per transport
- [Phase 08]: Four disjoint per-package bands below the kernel's ephemeral range plus a network:port claim held across the bind probe replace every private picker; both duplicate registries (scenarioPorts, probePorts) are deleted
- [Phase 08]: Each dumpNotify scenario run owns one testing/synctest bubble created in the subtest body; assertions run on the captured result after bubble exit, and two same-seed runs use two bubbles so the first engine cannot outlive its run.
- [Phase 08]: The chunker test pins cap(chunks[0]) == len(entries), per-chunk capacity bounded by entries remaining and adjacent chunks never sharing a backing array; teeth proved by temporary budget-hint (2729 vs 3) and cur[:0] mutations reverted byte-identically, since the WR-06 fix predates the plan.
- [Phase 8]: LatencyPercentiles is the one nearest-rank primitive (pinned by a 1..100 ms sample); the default-profile row measured p50 584 ms / p99 1.074 s at 1000.1 writes/s on ext4 against the 750/1500 ms bounds.
- [Phase 8]: Disk-write amplification is published as 14.36x (15 dumps x 14.98 MB vs 15.65 MB logical), bytes-per-dump 1.0001x the dump file; a tmpfs/ramfs base fails by name before any measurement.
- [Phase 8]: The memory points gate only the value-dominated 60 KiB row: RSS 280 MB against the 1.006 GB tripwire and dump-cycle peak growth 19.6 MB against the 122.9 MB PerfDumpPeakLiveRatio bound.
- [Phase 08]: Confirmation-latency profiles are data over one body: only the knobs and dump implementation differ, so the fast arm's cited 1 ms/50 ms + in-memory-dump conditions cannot drift.
- [Phase 08]: TestPerfFastProfileLatency wraps the shared body, not the benchmark table, because testing.Benchmark's Extra map is per-B: sub-benchmark metrics never reach the parent result.
- [Phase 08]: All four benchmark metrics are required by key name in the gate; the 25 ms/100 ms fast-profile bounds stay the documented values and are not retuned to the measured 2.64 ms/4.82 ms.
- [Phase 08]: The 30 s first-wait budget is a harness start budget, not a product bound: the eight-slot spawn serializer lands first, servingDeadline still governs running-node polls, and exceeding 30 s is a finding rather than a reason to raise it.
- [Phase 08]: exitStatus returns (bool, error) (revive's error-last rule) and the stand launcher dropped its proven-unused env parameter; no behavior changed.
- [Phase 08]: A dead child is diagnosed through the error-returning awaitNodeServing core with a fatal requireNodeServing wrapper; TestMain_readinessDiagnosis asserts the first-poll diagnosis directly.
- [Phase 8]: The shared port allocator claims a candidate before its bind probe; the owner's immediate-bind contract is pinned by a test, and the TEST-11 acceptance is green three runs in a row after the 08-06/08-09 follow-ups
- [Phase 8]: Startup-gate bounds measure from the instant cmd.Start returns (spawnSem queue time is harness wait, not product startup), and the loopback-broadcast probe retries a bind/send failure on a fresh private-band port per the probeBroadcastForm policy
- [Phase 8]: Measured rows publish the second of two green make perf runs at commit 9cdc0f1; each section carries the hardware block, the commit beside it (test binaries report vcs unknown) and the producing command.
- [Phase 8]: Threshold anchors are formatted from internal/testutil constants (durations via .String(), rate via %.0f, ratios as '≤ %.1f×'), so a threshold change without a ledger change fails make test.
- [Phase 8]: The amplification row is a workstation NVMe/ext4 result; the target-grade eMMC/SD gap stays named against the Spec open question #2 and is replaced, not annotated, if target media appears.
- [Phase 8]: S-FAIL-15 is closed against TestRed_R02_burst, TestPerfBurstDistinctKeys and TestEngine_burstBoundedConfirmation; OPERATIONS keeps a memory summary plus a link, and PROJECT.md's latency claim carries its fast-profile conditions.
- [Phase 8]: Phase-8 gate made no task commit: files_modified is empty, no source changed and every gate passed on the committed tree (the 01-04/07-11 precedent); the record is 08-11-SUMMARY.md
- [Phase 8]: The plan's deterministic-gate package list omitted ./internal/stand/, where TestPerfBurstBoundsMatchRedContract lives (internal/stand/perf_test.go:247); it was run explicitly in its own package in addition to the verbatim command - more coverage, never a narrower run
- [Phase 8]: Two external planning-only commits (c0a5266, a27d62b) landed during the gate; the source tree stayed byte-identical to the dispatched HEAD d64d189 (git diff --stat excluding .planning empty), so the gate results stand and this plan's own source commits are zero
- [Phase 09]: the --help capture is authoritative over the research prose — research section 4 rounded the registered surface to 22 flags; the clean-env --help capture shows 23 registered flags plus the five env-only knobs, and the plan's own assumption makes the live capture the source — the >=22 vacuity guard holds and M-07's doc half covers all 23
- [Phase 09]: dump_debounce <= dump_max_wait is documented as the dump cycle's intended relation the validator does not enforce — internal/config/validate.go's 12-rule catalog contains no debounce-vs-max-wait rule, so CONFIGURATION.md states the honest semantics (a smaller max-wait truncates every quiet window at the deadline) instead of inventing a startup rule
- [Phase 09]: M-07's doc half is closed by docs/CONFIGURATION.md with the OPERATIONS cross-link deferred to plan 09-05 — all registered flags and defaults are documented and machine-checked in both directions by TestConfigurationDoc; the OPERATIONS-side link edit belongs to the plan that owns OPERATIONS.md, and the two-half disposition is recorded here and in 09-01-SUMMARY.md
- [Phase 9]: 09-02: the fixed record set uses node 0x5060 - the plan 48-bit literal 0x102030405060 cannot fit the version layout 16-bit NodeID domain, and doc and test share the tuple — PackVersion takes a uint16 node id (RFC 9562 rand_b high 16 bits); the plan gloss is a Rule-1 correction recorded in the summary and in the test comment
- [Phase 9]: 09-02: the example dumps show the as-built data encodings the plan gloss missed - a tombstone is data:null and a live empty value is data:"" (a zero-length non-nil []byte renders as the empty base64 string), so the two edge shapes are distinguishable by the deleted flag alone — The scratch generator output exposed the distinction; the doc documents both shapes explicitly because the dump grammar must not claim one null shape for both
- [Phase 9]: 09-02: the dump loader does not validate the UUIDv7 layout or a future bound at load (class=version is hex/16-byte only) - the plan "version layout/future" gloss belongs to the replication receive path, and FORMATS.md documents the split — decodeRecord has no version.Validate call; the plan own grounding rule (the code wins over the research gloss) applied, and the doc cross-references the replication-side layout/future checks
- [Phase 9]: 09-02: the rejection-class enumeration rides reflect.TypeFor[persist.DecodeStats]() instead of parsing codec.go source - the real class set is the type own fields, immune to source reformatting — The plan acceptance asks for the real classes not a hand list; the exported type is the codec contract surface, and a class the codec adds or removes without a doc edit fails the test
- [Phase 9]: 09-03: the stack audit is bidirectional and derived at test time — expected values are parsed from go.mod (go directive, tool block) and `go list ./...`, so a dependency or package change without a doc edit fails make test in both directions (doc-cited-but-absent and present-but-undocumented)
- [Phase 9]: 09-03: the tool-directive inventory is line-scoped — the audit parses only lines naming the tool directives, so AGENTS.md's dependency policy and the CLAUDE.md stack table each keep the complete seven-name list on one source line and prose backticks never count
- [Phase 9]: 09-03: M-40's SetReadBuffer advice is verified-then-kept — internal/udprepl/socket.go sets SO_RCVBUF/SO_SNDBUF and reads the kernel values back (XPORT-05), so the research advice became accurate when Phase 3 shipped
- [Phase 9]: 09-03: M-51 is an executed claim — the by-id ruleset command ran 2026-11-28 and printed exactly the documented shape, recorded verbatim in branch-protection.md; M-23's Go half is verify-then-delete (zero journal survivors in storage/repl production comments)
- [Phase 9]: 09-03: M-49's split — the Makefile buf-breaking recipe lost its dead pre-Phase-6 probe branch (real comparison proven green) while ci.yml got comment-only edits per the plan's 09-07 ownership constraint; the leftover probe carries an honest comment
- [Phase 09]: the registry gate excludes the preamble by construction: rows are lines whose normalized form starts "| S-" and the status check parses the fourth cell, so the vocabulary line's identical fixed-in-v1.1( token can never count as a row — T-09-11 (a gate that counts the preamble can never pass)
- [Phase 09]: the milestone exit criterion is zero unclosed rows, not zero non-implemented rows: deferred-v2(SNAP-01) and the two accepted-deviation rows stay permanent, and S-FAIL-15 still carried its unclosed status cell so it flipped with the other 21 — research pitfall 2; the grep-count verify (167 rows, one fixed-in-v1.1 token) is the proof
- [Phase 09]: TestSpecDeviationsBurstRow's status anchor was re-pointed from fixed-in-v1.1(PERF-02) to implemented - the Phase-8 anchor froze the pre-sweep status this plan's own exit criterion removes; its purpose (the row cites the shipped tests and bounds) is unchanged — Rule-1 deviation: the plan assumed S-FAIL-15 was already status-flipped; only its summary gloss was
- [Phase 09]: citation re-derivation was symbol-extent based across all 167 rows: ~70 drifted citations beyond the New-3 seeds corrected (core.go growth, the persist scheduler/engine split, the watch hub, loader.go Open), and S-PAR-01..04 re-cited to the registered flag names - the 06-10 lesson is now a permanent gate (TestSpecDeviationsLinksResolve, 720 checks) — research pitfall 3; the gate walks every Links cell under a 250-check vacuity floor
- [Phase 09]: [Phase 09]: 09-05 retained TestOperationsDocFamilies beside the new two-directional check: the structured check is stronger per family (type + label values against Snapshot) but section-scoped, while the whole-doc exporter grep still covers family mentions in runbooks, Monitoring and the failure table — the deletion bar (equal or stronger coverage) is not met — [Phase 09]: 09-05 retained TestOperationsDocFamilies beside the new two-directional check: the structured check is stronger per family (type + label values against Snapshot) but section-scoped, while the whole-doc exporter grep still covers family mentions in runbooks, Monitoring and the failure table — the deletion bar (equal or stronger coverage) is not met
- [Phase 09]: [Phase 09]: 09-05's vocabulary source builds zero-returning source closures mirroring the exporter's own zeroSources() helper because metrics.Snapshot(metrics.Sources{}) would panic on the nil accessors; the complete-catalog rule (all label values render at zero state) is the check's real foundation, and the supkv_ vacuity floor is 23 — the exporter's true family count, not the plan's miscounted 24 — [Phase 09]: 09-05's vocabulary source builds zero-returning source closures mirroring the exporter's own zeroSources() helper because metrics.Snapshot(metrics.Sources{}) would panic on the nil accessors; the complete-catalog rule (all label values render at zero state) is the check's real foundation, and the supkv_ vacuity floor is 23 — the exporter's true family count, not the plan's miscounted 24
- [Phase 09]: 09-06's per-RPC reference mirrors the proto at test time — the schema declares seven RPCs, not the plan's nine (SetIfAbsent is Set's create_if_absent variant, WaitForReplication is Set/Delete's wait fields), so the vacuity floor is 7; the order check is exact-equality over parsed rpc declarations, so the reference and the schema cannot silently diverge
- [Phase 09]: 09-06's error tables factor the every-RPC facts into the RPC reference intro (gate-closed Unavailable before any validation; transport ResourceExhausted at MaxRecvBytes) and carry honest absences — List and Watch have no Internal row because no engine or watch-failure path exists, while the snapshot tables carry classifySaveError's classified faults (ResourceExhausted/PermissionDenied) plus the mapper's totality fallback
- [Phase 09]: 09-06's pending flip moves the five pending-section literals into the whitespace-normalized apiDocForbidden table and rewrites the Errors section's pending-pointer sentence (it carried two of them) — "keep every existing section intact" yields to the flip prohibition; raw anchor literals must sit on one doc line, so mandated literals are reflowed contiguous while whitespace normalization stays with the forbidden table
- [Phase 09]: 09-07's unit consistency is doc-anchored, not help-anchored — TestDeploymentUnitConsistency compares every ExecStart flag against docs/CONFIGURATION.md's parsed table rows and requires each non-default value stated in DEPLOYMENT.md via a boundary-exact `--flag value` regex, so the demo's `--node-id 11` can never satisfy a check for the unit's `--node-id 1`, and a usage error (exit 2) is structurally excluded from the restart loop by RestartPreventExitStatus=2 (M-16)
- [Phase 09]: 09-07's node-id is the 16-bit integer the binary parses, not the plan's 16-hex gloss (a base-0 uint64 rejects the literal with exit 2) — the unit carries --node-id 1 and the CI smoke runs the unit's value; the unit's metrics listener stays loopback plaintext per OPERATIONS' own guidance with the SUPKV_METRICS_* mTLS triple documented as the beyond-loopback path
- [Phase 09]: 09-07's CI leg joins the gate job's Linux cells under `!matrix.cross && runner.os == 'Linux'` with a documented override delta (four addresses to loopback doc-band ports, dump path to /tmp, --startup-gate-empty 2s pinned for the fresh empty-disk node) — the job-key count vs HEAD and the additive-diff guard keep the five check-run contexts and the matrix byte-identical; the unit's ExecStart is one line so the plan's single-line grep guards hold while the consistency parser joins continuations either way
- [Phase 09]: The proven-claim checker's paragraph-scoped scan is more complete than the plan's two-site RED gloss — it names five bare sites (README.md:21,30,134; CHANGELOG.md:18,28), and the plan's 'every current bare spot' must-have governs, so all five get fixed by naming their bound tests — The docs-audit evidence section (audit line 260) had already listed README.md:21,30 as proven/proving sites; the word-boundary paragraph scan found them exactly as designed, so the RED state is the DOCS-09 rule made executable, not a checker bug
- [Phase 09]: M-31 is closed by 09-08 although the plan task text omitted it — the master M-assignment (09-04-SUMMARY, the table 09-11 greps) assigns it to 09-08, and the README edit set already owned the quickstart prose (crossed ports named as the same-host trick; the CI darwin cross-compile cell stated) — Skipping it would leave a master-table row unowned-closed at the milestone completeness check; the fix is a two-sentence README edit inside the plan's declared file scope
- [Phase 09]: Both SERVING assertion rows flip to the quoted '"status": "SERVING"' shape, not just the Start row — audit M-17 flagged both examples_test.go rows as NOT_SERVING-satisfiable — Fixing one row would leave exactly the weakness threat T-09-23 mitigates (a health wait satisfied by a not-yet-serving node); the quoted shape is strictly stronger and TestExamplesDoc stays green
- [Phase 09]: requirements.ready-ids reported DOCS-01 and DOCS-10 ready despite unfinished declaring siblings (09-09/09-10/09-11, no summaries) — the #2388 documented rule won and the premature Complete flips were reverted by hand; DOCS-09 stayed Complete (only sibling declarer 09-05 has a summary, all its clauses landed) — Later plans (09-09/09-10/09-11) must re-evaluate DOCS-01/DOCS-10 at their own close-out; the REQUIREMENTS.md checkboxes and phase table are the authoritative state, currently Pending for both
- [Phase 09]: Plan 09-09 executed — the maintainer docs landed machine-guarded: TestArchitectureDocPackages (every go list ./... package named in docs/ARCHITECTURE.md, six depguard rule IDs, 19-package floor), TestADRIndex (bidirectional frontmatter-derived file↔row correspondence over 0001–0012, numeric order, rows link files) and TestArchitectureDocDivisionOfLabor (the AGENTS pointer, provenance never restated) went RED-then-GREEN; the five new docs carry M-29's parameter-surface half and M-30's architecture half with CONFIGURATION.md/API.md pointers; CONTRIBUTING points at AGENTS.md and defers the release pointer to 09-10; SECURITY.md mirrors OPERATIONS' three absences verbatim with the private-advisory disclosure path; the DOCS-10 benchmark-results coverage check found no gap (5 hardware blocks, every section with its producing command); the 09-03 deferred second architecture pointer landed in .claude/CLAUDE.md — the maintainer docs are machine-bound to the live repo so doc drift fails make test — guards derived at test time (go list ./..., ADR frontmatter), never hand-typed inventories
- [Phase 09]: 09-10 archives files: is an explicit list (README.md, LICENSE, CHANGELOG.md, docs/DEPLOYMENT.md, deploy/sup-kv.service) because defining files: replaces goreleaser v2's default license*/readme*/changelog* globs — verified in the pinned v2.20.2 archive pipe source (archive.go:83-89, defaults apply only when files is empty); explicit entries keep the archive paths predictable — the DOC-05 archive contract needs the deployment artifacts beside the defaults and relying on default-glob coexistence would have been unverified behavior
- [Phase 09]: 09-10 release header slices the [1.1.0] section via the planning-proven template (split binds to strings.Split, content first — verified in the pinned tmpl.go) with the two version markers documented in the file as the next release's bump points; the offline text/template render over the real CHANGELOG.md produced exactly the 1.1.0 section, so the whole-file fallback was never needed — the old whole-file header's own comment promised the slice from the second release on; DOC-24 forbids [Unreleased] in the body and the recorded render output is the shape 09-11's release-body check must match
- [Phase 09]: 09-10 syft v1.54.0 provisioned before the first snapshot as a checksum-verified GitHub-release download into $HOME/.local/bin (mirrors release.yml's anchore/sbom-action/download-syft@v0.24.3 pin); --skip=sbom never used — checksums.txt hashes the SBOM files — the sboms pipe shells out to syft and the machine had none; a package-manager install would violate the T-09-SC no-npm/pip/cargo disposition, and the checksum-verified download is the same provisioning shape CI gets from the pinned action
- [Phase 09]: 09-10 DOCS-01/DOCS-10 REQUIREMENTS flips stay Pending despite requirements.ready-ids reporting both ready — the #2388 shared-ID gate is applied per the recorded premature-flip precedent because 09-11 also declares both and has no summary; the last declaring plan's close-out re-evaluates — the maintainer hand-reverted the 09-08 close-out's premature Complete flips and recorded that later plans must re-evaluate at their own close-out; flipping now would repeat the exact mistake while 09-11's release clauses are still open
- [Phase 09]: 09-10 CONTRIBUTING.md gains the release-procedure pointer although the plan's files_modified omitted it — 09-09's recorded decision states the pointer is deferred to this plan; verified first that no Go test pins CONTRIBUTING.md content (Rule-2 classed deviation, recorded in 09-10-SUMMARY) — skipping it would leave the cross-plan contract unfulfilled and the procedure doc undiscoverable from the dev-loop doc
- [Phase 09]: 09-10 Q-09 dead-code verification recorded (D-09/XPORT-13 closure evidence) — five probes, zero survivors: lastLocal 0, ownHandle 0, func Parse in internal/app/persist 0, the cover identifier 0 (five surviving grep lines are comment prose using the verb per research section 8's identifier-vs-prose distinction), fsync_darwin.go gone; verify-then-record, no deletions — Phase 3's rewrites already removed the targets, so the expected zero-survivor state is the evidence; inventing deletions would fabricate history

### Pending Todos

None yet.

### Blockers/Concerns

- Carried from v1.0: server-side branch-protection ruleset still deferred (org-plan 403); re-apply path recorded in docs/setup/branch-protection.md — revisit at the v1.0.1+/ruleset window noted in PROJECT.md
- Carried from v1.0: macOS quickstart is option-b (Linux CI-verified + marked darwin loud-skip); v1.1 DOCS-09 must keep the pair working and DOCS-10 demotes darwin to development-only (14.10)
- **03-12 halt resolved (2026-11-22):** the `powersave`-governor miss was cleared at `performance` — the burst measurement window was aligned with the documented acceptance (commit `6c17db3`), the full runner is 7/7 OK with mutants killed (`red-evidence/20261121T225048Z`), and R-01/R-02 burst/drain are green. 03-13 executed and closed the phase; no blocker remains.
- **Nightly 20/20 observation:** XPORT-01's 20x evidence is the first scheduled `red-r01-x20` run. If it exceeds the 120-minute job budget, the finding is recorded and the budget raised — the count stays 20.

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| Feature | AUTH-01 HMAC datagram auth; GARB-01 tombstone GC; PFXD-01 per-prefix delete-wins; PEER-01 explicit peer lists; SNAP-01 consumer proto mapping | Tracked in REQUIREMENTS.md v2 section | 2026-11-19 | v1.1 |

## Session Continuity

Last session: 2026-11-28T01:22:06.085Z
Stopped at: Completed 09-10-PLAN.md
Resume file: None
Next action: Execute 09-11-PLAN.md (the final release plan: gates on master, the snapshot re-run, the one-way v1.1.0 tag behind its publication checkpoint, release-body verification)
