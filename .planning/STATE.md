---
gsd_state_version: "1.0"
milestone: v1.1
milestone_name: Spec Compliance and Resilience
current_phase: 3
current_phase_name: Transport
status: executing
stopped_at: Completed 03-11-PLAN.md
last_updated: "2026-11-21T22:05:57.000Z"
last_activity: 2026-11-22
last_activity_desc: Phase 3 plan 11 complete — dead-code removal
state_head: 14c0cc6f740f0fd3671b25855cb54e54626227d5
progress:
  total_phases: 9
  completed_phases: 2
  total_plans: 34
  completed_plans: 32
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-11-19)

**Core value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.
**Current focus:** Phase 3 — Transport

## Current Position

Phase: 3 (Transport) — EXECUTING
Plan: 12 of 13
Status: Ready to execute
Last activity: 2026-11-22 — Completed 03-11 (dead-code removal)

Progress: [█████████░] 94%

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

### Pending Todos

None yet.

### Blockers/Concerns

- Carried from v1.0: server-side branch-protection ruleset still deferred (org-plan 403); re-apply path recorded in docs/setup/branch-protection.md — revisit at the v1.0.1+/ruleset window noted in PROJECT.md
- Carried from v1.0: macOS quickstart is option-b (Linux CI-verified + marked darwin loud-skip); v1.1 DOCS-09 must keep the pair working and DOCS-10 demotes darwin to development-only (14.10)

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| Feature | AUTH-01 HMAC datagram auth; GARB-01 tombstone GC; PFXD-01 per-prefix delete-wins; PEER-01 explicit peer lists; SNAP-01 consumer proto mapping | Tracked in REQUIREMENTS.md v2 section | 2026-11-19 | v1.1 |

## Session Continuity

Last session: 2026-11-21T22:05:57.000Z
Stopped at: Completed 03-11-PLAN.md
Resume file: None
Next action: `/gsd:execute-phase 3` (03-12-PLAN.md)
