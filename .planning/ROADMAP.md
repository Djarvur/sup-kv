# Roadmap: sup-kv

## Overview

Milestone v1.1 closes the findings of the 2026-11-19 v1.0 review (`docs/reviews/2026-11-19-v1.0/`): recovery and burst scenarios miss their stated bounds on real sockets, and sup-kv deviates from its Spec in ways that are neither fixed nor recorded. The phase order is red-tests-first: the deviation registry and decision ADRs land first (Phase 1), then the review's scenarios — F9, F9b, F14, the 1000-writes/s burst, F2c — are automated on real Linux sockets and the simulator's blind spots are closed (Phase 2), with the four Critical findings (R-01, R-02, R-03, P-01) pinned by tests that are expected to be RED against v1.0 code. Fixes then flow transport → protocol → persistence → API → observability → performance, turning the red tests green one subsystem at a time, and the milestone closes with test-anchored documentation and the v1.1 release. Exit criteria: F9, F9b, F14 and the 1000-writes/s × 15 s burst over 10 000 keys pass on real sockets in CI; the deviation registry has no unclosed rows; every row of the Spec's failure table has a test at real-socket level or a recorded reason why the simulator suffices.

## Milestones

- ✅ **v1.0 sup-kv v1.0** — Phases 1-7 (shipped 2026-11-18, tag v1.0.0)
- 🚧 **v1.1 Spec Compliance and Resilience** — Phases 1-9 (in progress; phase numbers reset for v1.1, `--reset-phase-numbers`)

## Phases

<details>
<summary>✅ v1.0 sup-kv v1.0 (Phases 1-7) — SHIPPED 2026-11-18</summary>

- [x] Phase 1: CI Bootstrap and Repo Skeleton (6/6 plans) — completed 2026-11-02
- [x] Phase 2: Engine — HLC Versioning and LWW Store (4/4 plans) — completed 2026-11-02
- [x] Phase 3: Persistence — Dump and Write Confirmation (2/2 plans) — completed 2026-11-03
- [x] Phase 4: Replication and Deterministic Simulator (6/6 plans) — completed 2026-11-07
- [x] Phase 5: gRPC API and Binary (6/6 plans) — completed 2026-11-09
- [x] Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks (9/9 plans) — completed 2026-11-16
- [x] Phase 7: v1.0 — Docs, Examples and Release (5/5 plans) — completed 2026-11-18

Full phase details, decisions and evidence: [milestones/v1.0-ROADMAP.md](milestones/v1.0-ROADMAP.md)

</details>

**🚧 v1.1 Spec Compliance and Resilience** — close every v1.0-review finding by code fix or recorded deviation; scenarios pass their stated bounds on real sockets.

- [x] **Phase 1: Spec Reference and Decisions** - Deviation registry mapping every Spec statement to a decision and link; 11 ADRs for owner decisions 14.1-14.11; docs/IDEA.md marked historical (completed 2026-11-19)
- [x] **Phase 2: Stand and Simulator as Tests** - F9/F9b/F14/burst/F2c automated on real sockets; simulator blind spots closed; chaos stabilized; red tests for the four Criticals (R-01, R-02, R-03, P-01) (completed 2026-11-21)
- [x] **Phase 3: Transport** - Paced sync/retransmit sending, socket buffer setup and reporting, datagram validation, bounded queues; key→version view and Outbox seams (completed 2026-11-22)
- [x] **Phase 4: Protocol** - Interface closed until the full key list arrives, `startup_gate_empty`, DataSync headers, stale-record answer, immediate sync for new/returning peers, 3+-node tests (completed 2026-11-22)
- [x] **Phase 5: Persistence** - Single value limit, visible dump-load outcomes, generation integrity, decaying retries, streaming encode, blocking deadlines, engine split (completed 2026-11-22)
- [x] **Phase 6: API and Semantics** - Honest `wait_for_replication`, snapshot load as full rollback, total error mapping, paginated `List`, public proto + Go client example in CI (completed 2026-11-23)
- [ ] **Phase 7: Observability and Configuration** - Lifecycle logging, one-point config validation, honest metric definitions, distinct exit codes, machine-enforced dependency boundary
- [ ] **Phase 8: Performance** - Documented write-latency model for both profiles, standing burst test, amplification and memory measured, `make perf` with thresholds
- [ ] **Phase 9: Documentation and Release** - All 52 doc mismatches closed, test-anchored reference docs, failure table + runbooks, registry closed, v1.1 release

## Phase Details

### Phase 1: Spec Reference and Decisions
**Goal**: Every deviation between sup-kv and its Spec is visible and every settled owner decision is recorded — a reader can always answer "does sup-kv implement Spec statement X, and if not, who decided that and where is the code?"
**Depends on**: Nothing (first phase of v1.1)
**Requirements**: SPEC-01, SPEC-02
**Success Criteria** (what must be TRUE):
  1. A deviation registry exists in the repository mapping every Spec statement ID to a decision and a code/ADR link — for any Spec statement, a reader can find its status in one lookup
  2. Each review section-7 deviation (S-01, S-02, S-06, S-07, S-08, S-12) has a registry row closed one of two ways: by code (with a forward reference to the v1.1 requirement that fixes it) or by an accepted-deviation entry — no row is empty or TBD
  3. Eleven ADRs record owner decisions 14.1-14.11, including the two accepted-by-default confirmations of 2026-11-19 — 14.5(6) read of a listed-but-not-yet-arrived key returns `Unavailable`, and 14.7 last-read-value file lives outside sup-kv
  4. docs/IDEA.md is marked historical (or replaced by an as-built pointer), so no reader mistakes the Spec for a description of current behavior
**Plans**: 4/4 plans complete

Plans:
**Wave 1**
- [x] 01-01-PLAN.md — Relocate review artifacts to docs/reviews/2026-11-19-v1.0/ with the .gitignore publication guard, sweep stale refs, mark IDEA.md historical (D-01..D-04, D-12/D-13)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 01-02-PLAN.md — Seed docs/SPEC-DEVIATIONS.md: all 167 statement IDs, D-08 statuses, section-7 closures (D-05..D-08)
- [x] 01-03-PLAN.md — Author 11 slim-MADR ADRs for decisions 14.1-14.11 with the D-11 reopenable lines (D-09..D-11)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 01-04-PLAN.md — Phase gate: registry↔ADR cross-audit, publication guard re-assertion, make lint/test

### Phase 2: Stand and Simulator as Tests
**Goal**: The review's verdict is reproducible by machine — every claimed miss (recovery, burst) and every Critical finding is pinned by an automated test on real sockets or a hardened simulator, RED where the code is wrong. The tests are expected to fail against v1.0 code; that is the point — Critical findings get failing tests before fixes, and Phases 3-5 turn them green.
**Depends on**: Phase 1 (decision names and ADR references that TEST-13 tests cite)
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06, TEST-07, TEST-08, TEST-09, TEST-10, TEST-11, TEST-12, TEST-13, TEST-14, TEST-15, TEST-16
**Success Criteria** (what must be TRUE):
  1. A real-socket stand exists in CI: review scenarios F9, F9b, F14, the 1000-writes/s × 15 s burst over 10 000 keys and F2c run as automated tests over real Linux sockets built with `-race`, and no socket test is ever silently skipped (a Linux CI skip is an error; the macOS cell either runs a two-node test or states darwin replication is untested)
  2. The tests demonstrably bite: run against v1.0 code, the Critical-scenario tests — R-01 paced sync over 20 000 keys, R-02 unacked-set cost and burst drain, R-03 closed-until-list interface, P-01 single value limit — fail, with the red runs recorded as phase evidence; `TestChaos_convergence` is mutation-sensitive (fails under "sync without tombstones" and "periodic sync disabled" mutants)
  3. The simulator's blind spots are closed: per-node incoming-datagram budgets, size checks by the real encoder, record announcement after a virtual dump delay, restart with state, and deletions; `TestChaos_convergence` is stable across 10 000 seeds (fixed corpus plus fresh seeds), and convergence bounds hold under combined failures and partitions longer than `peer_timeout`
  4. The seams are pinned by tests, not conventions: a filesystem seam under `persistfile` with recording and failing fakes asserting write → fsync → rotate → rename → dir-fsync and a failure at each step; `persist.Open` clock seeding proven on the real composition path; a corrupted-datagram table plus `FuzzDecode`/`FuzzDumpDecode` with an in-repo corpus; deletion replication proven at codec, simulator, two-process and dump level; Watch cut/backpressure branches, metrics-by-observed-effect, per-decision tests referencing 14.x by name, and property shrinking over a small `(l, c, node)` space
  5. The heavy machinery runs on a schedule: a scheduled CI workflow (1000 seeds, `-shuffle=on -count=3`, `-bench . -benchtime=1x`), a nightly real-socket run at 10 000 keys × 1 KiB with the review's section-15 scenarios; benchmarks report p50/p99 without self-pacing on the real `persistfile`; `go test -race -count=20` over `cmd/sup-kv`, `udprepl` and affected packages green three times in a row (flakiness classes that require Phase 3-5 fixes carry their red test forward)
**Plans**: 17/17 plans complete

Plans:
**Wave 1**
- [x] 02-01-PLAN.md — Tracer: red-gating mechanism end to end — first env-gated R-03 red test + scripts/red-runner expectation table + evidence capture (D-01, TEST-04)
- [x] 02-02-PLAN.md — internal/stand package: harness move, four migrated scenarios, F2c pin, no-silent-skip policy (D-03, TEST-05 Linux branch, TEST-03 process half)
- [x] 02-03-PLAN.md — simnet additive extensions: Delete, restart-with-state, receive budget, SizeOf by real encoder, dump-delay (D-05, TEST-06, TEST-02 sim level API)
- [x] 02-04-PLAN.md — fsops seam under persistfile (five ops: write/fsync/rotate/rename/dir-fsync) + blocking-Save proof (D-07, TEST-03 seam half)
- [x] 02-14-PLAN.md — Loud non-Linux darwin report test in internal/stand (D-03, TEST-05 darwin branch; moved from 02-02)

**Wave 2** *(blocked on Wave 1)*
- [x] 02-05-PLAN.md — Stand-level red tests R-01/R-02/R-03-F14 with documented bounds and nightly-scale selection (D-01, TEST-04)
- [x] 02-06-PLAN.md — Chaos stabilization: fixed seed corpus + drift-checked generator, poll-to-model semantics, corpus iteration with SUPKV_SIM_FRESH_SEEDS, deletions in mix (D-06, TEST-01)
- [x] 02-07-PLAN.md — Deletion replication at four levels + corrupted-datagram negative table (D-08, TEST-02, TEST-08)
- [x] 02-08-PLAN.md — Watch Send-error branch + metrics-by-effect exact values + per-parameter effect tests (D-08, TEST-14, TEST-12)
- [x] 02-10-PLAN.md — Benchmarks report p50/p99, no self-pacing, smoke target; flake-class audit + parameterized race-soak target (TEST-10, TEST-11)
- [x] 02-15-PLAN.md — simnet extension tests incl. same-seed determinism replay gate (D-05, TEST-06, TEST-02 sim level pin; moved from 02-03)
- [x] 02-16-PLAN.md — persist.Open clock seeding on the real composition path (D-07, TEST-07; moved from 02-04)

**Wave 3** *(blocked on Wave 2)*
- [x] 02-09-PLAN.md — Per-decision 14.1-14.11 name citations + coverage meta-test; property shrinking over small (l, c, node) space + executable replay command (D-08, TEST-13, TEST-15)
- [x] 02-12-PLAN.md — P-01 value-limit red tests (process + engine), complete 7-row expectation table, full red-run evidence (D-01, TEST-04)
- [x] 02-17-PLAN.md — FuzzDecode/FuzzDumpDecode with committed in-repo corpora (D-08, TEST-08; moved from 02-07)

**Wave 4** *(blocked on Wave 3)*
- [x] 02-13-PLAN.md — TEST-09 combined/long-partition scenario tests + red-runner overlay mutant stage (D-02, TEST-01, TEST-09)

**Wave 5** *(blocked on Wave 4)*
- [x] 02-11-PLAN.md — Nightly documented-scale stand wiring (F9 10k × 1 KiB selection) + scheduled.yml six-job nightly set with parameterized 10k-seed dispatch + ci.yml tightening (D-09, TEST-16, TEST-06 nightly 10k × 1 KiB, TEST-08 deep fuzz, TEST-11 soak)

### Phase 3: Transport
**Goal**: The UDP transport meets the review's stated bounds on real sockets — sending is paced and bounded, buffers are configured and reported, input is validated before it can touch the clock — so the recovery and burst scenarios converge within their documented limits.
**Depends on**: Phase 2 (red tests exist to turn green)
**Requirements**: XPORT-01, XPORT-02, XPORT-03, XPORT-04, XPORT-05, XPORT-06, XPORT-07, XPORT-08, XPORT-09, XPORT-10, XPORT-11, XPORT-12, XPORT-13
**Success Criteria** (what must be TRUE):
  1. Sync listing is paced and robust to buffer loss: two nodes on real Linux sockets with default buffers, 20 000 keys × 1 KiB into an empty receiver converge no later than 2 × `sync_interval` + N / `sync_response_rate` in 20 runs out of 20, in CI, with the sending start chunk rotating across rounds (R-01 red test turns green)
  2. The unacked set is cheap at any size: iteration time at 10 vs 10 000 unacked records differs by at most 2×; a peer returning to a node holding 10 000 unacked records drains within N / `sync_response_rate` + 10 s with at most 3N retransmits; open-loop 1000 writes/s × 15 s over 10 000 distinct keys achieves at least 990 writes/s with the set empty 5 s after the end (R-02 red test turns green; milestone burst exit criterion)
  3. Datagrams are bounded and validated end to end: key-length and `value_max_bytes` ceilings enforced (`InvalidArgument` on oversize, exit 2 when the largest record cannot fit a datagram, any accepted record encodes into ≤ 65 507 bytes); received versions validated before any clock call and counted as their own class; invalid combinations (empty key, tombstone with data, bad checksum) discarded with own counters; the read loop exits only on `net.ErrClosed` — a node never keeps saying `Hello` while deaf
  4. Socket reality is visible and misuse is loud: startup logs actual `SO_RCVBUF`/`SO_SNDBUF` and fails if the send buffer cannot hold a maximal datagram; 1, 20 and 60 KiB values deliver on Linux (darwin ~9 KiB limit documented, not fixed); a non-wildcard `--bind-addr` warns or is rejected; a second instance fails with "address already in use"; a silent node WARNs after `peer_timeout`
  5. The seams later phases build on exist: a cheap key→version view with tombstone visibility instead of store copies (also making `DataRequest`/sync-chunk handling independent of foreign records); "record acknowledged by peer X" traveling as an `Outbox` result to waiters (removing snapshot diffing and the 5 ms polling); a bounded, deduplicated sync-answer queue (1000 missing keys twice → 1000 answers, not 2000); dead code (`lastLocal`/`cover`, exported `persist.Parse`, `ownHandle`, `fsync_darwin.go`) gone or justified
**Plans**: 13/13 plans complete

Plans:
**Wave 1**
- [x] 03-01-PLAN.md — Tracer: socket buffers set, read back, logged, send floor enforced; 1/20/60 KiB delivery (D-02, XPORT-05)
- [x] 03-02-PLAN.md — Store key→version view + tombstone-visible Lookup + MaxKeyBytes + WR-02 simnet fix (D-08, XPORT-11/XPORT-06)
- [x] 03-03-PLAN.md — version.Validate layout/future predicate with boundary and property tests (D-06, XPORT-04)

**Wave 2** *(blocked on 03-02)*
- [x] 03-04-PLAN.md — Core consumes the view; StateStore shrinks; key-count flatness gate at 1 000 vs 50 000 (D-08, XPORT-06/XPORT-11)

**Wave 3** *(blocked on 03-04)*
- [x] 03-05-PLAN.md — Record-free (key, version) ledger, per-peer unacked index, due-heap, O(1) gauges, 10 vs 10 000 gate (D-03, XPORT-02)

**Wave 4** *(blocked on 03-05)*
- [x] 03-06-PLAN.md — Outbox result events, adapter tracker, thin honest wait; mirror/poll removal (D-04, XPORT-12)

**Wave 5** *(blocked on 03-06)*
- [x] 03-07-PLAN.md — Paced rotating listings, record pacer with ladder re-arm, deterministic jitter, bounded deduped queue (D-01/D-05/D-08, XPORT-01/02/07/09)

**Wave 6** *(blocked on 03-07 + 03-03)*
- [x] 03-08-PLAN.md — Wire validation before the clock, five discard classes, max_clock_lead, scenario/test-model adaptation (D-06, XPORT-04/10/03)

**Wave 7** *(blocked on 03-08 + 03-02)*
- [x] 03-09-PLAN.md — Key limit at the API, startup datagram ceiling exit 2, property through the real codec, F2c rewrite, documented limits (D-06, XPORT-03)

**Wave 8** *(blocked on 03-09)*
- [x] 03-10-PLAN.md — Read loop exits only on ErrClosed; SO_REUSEADDR removed; bind and no-Hello WARNs (D-07, XPORT-08/09)

**Wave 9** *(blocked on 03-10)*
- [x] 03-11-PLAN.md — Delete lastLocal/cover, persist.Parse, ownHandle, fsync_darwin.go/fsync_other.go; S-PERS-03 updated (D-08, XPORT-13)

**Wave 10** *(blocked on 03-11)*
- [x] 03-12-PLAN.md — New metric families + discard classes; red-runner per-mutant source and re-pointed tombstone mutant; executed 2026-11-22 — HALTED on the flip: R-02 drain green, R-01/R-02 burst red (see 03-12-SUMMARY.md) (XPORT-01..13 gate)

**Wave 11** *(blocked on 03-12; halt resolved 2026-11-22)*
- [x] 03-13-PLAN.md — Scheduled 20× R-01 evidence job, measured race-soak WR-06, full phase gate (XPORT-01/02/05/08/09)

### Phase 4: Protocol
**Goal**: The replication protocol implements the Spec's reconciliation semantics — a joining or restarting node's interface stays closed until it has the complete picture, and every peer interaction (stale records, new peers, duplicate identities, version mismatch) has defined, tested behavior.
**Depends on**: Phase 3 (paced sending, key→version view, Outbox seam)
**Requirements**: PROTO-01, PROTO-02, PROTO-03, PROTO-04, PROTO-05, PROTO-06, PROTO-07, PROTO-08, PROTO-09
**Success Criteria** (what must be TRUE):
  1. The interface is closed until the full key list arrives (R-03 red test turns green): F14 passes on the stand and in the simulator — the value on the receiving node A does not change; before the final `DataSync` part B answers `Unavailable` for reads and writes (14.5(6)); health stays `NOT_SERVING`; after the list, `create_if_absent` for a peer-known key returns `AlreadyExists`; after the part carrying max version `(now + 10 s)` the next local write gets a greater version; `DataSync` carries `SyncID`/`ChunkIndex`/`ChunkCount` and the HLC updates on each part's max version
  2. Empty-disk and returning peers are handled deliberately: `startup_gate_empty` (default 60 s) holds an empty node longer than `startup_gate` holds a node with data; two empty nodes exchange empty lists and open without the new parameter; the first `Hello` from an unknown `node_id` triggers exactly one full listing (a repeat, nothing); a peer returning after `peer_timeout` + 1 s receives the neighbor's state within N / `sync_response_rate` plus margin, without the `sync_interval` term
  3. Staleness never lies: a node holding `k@v5` that receives `k@v3` sends back exactly one unicast `DataReplication k@v5` and no `Ack` (the sender's `wait_for_replication` for `v3` reports `acked: false`); a local-write notification with a version not greater than the tracked one is ignored
  4. Misconfiguration and version mismatch are named errors: a third process under a known `node_id` with alternating `run_id`s is counted and logged as a configuration error and receives no full listing; an incompatible `proto_version` is checked before body decode, classified `bad_proto_version`, and WARN-logged rate-limited per peer per interval
  5. Replication is total and N-node behavior is specified: a write whose dump failed still broadcasts (counted as unconfirmed) and a snapshot load broadcasts its result; 3+-node behavior and timing is documented with real-socket three-node tests, and `wait_for_replication` semantics at N > 2 are in the API docs
**Plans**: 13/13 plans complete
**Wave 1**
- [x] 04-01-PLAN.md — Complete-listing wire header, per-stream accumulation, one atomic mirror, pre-body proto-version verdict (PROTO-01, PROTO-07)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 04-02-PLAN.md — Gated reads and listed-key RPC semantics: Unavailable/NotFound/AlreadyExists (PROTO-01)
- [x] 04-03-PLAN.md — Store/engine Observe and KeyCount: per-part HLC absorption and O(1) empty selection (PROTO-01, PROTO-02)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 04-04-PLAN.md — `startup_gate_empty` end to end, including the stand-harness derivation (PROTO-02)
- [x] 04-05-PLAN.md — Identity-only egress (CR-02) and the stale-record answered-not-Acked outcome (PROTO-03)
- [x] 04-06-PLAN.md — Totality: dump-failed writes and snapshot-load replay broadcast (PROTO-08)
- [x] 04-07-PLAN.md — Three-node real-socket broadcast convergence (PROTO-09)

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 04-08-PLAN.md — First-seen/restart listings and the bounded shared listing build (PROTO-04, CR-01)

**Wave 5** *(blocked on Wave 4 completion)*
- [x] 04-09-PLAN.md — Notification regression guard and alternating-identity configuration error (PROTO-05, PROTO-06)

**Wave 6** *(blocked on Wave 5 completion)*
- [x] 04-10-PLAN.md — Simulator N=3, complete-listing window, 14.9 evidence anchor (PROTO-09, PROTO-01)

**Wave 7** *(blocked on Wave 6 completion)*
- [x] 04-11-PLAN.md — Test-adaptation sweep and whole-suite green (adaptation wave)
- [x] 04-12-PLAN.md — `docs/04-MULTI-NODE.md`, operations and deviation-registry handoff (PROTO-09)

**Wave 8** *(blocked on Wave 7 completion)*
- [x] 04-13-PLAN.md — R-03 TSV flip, Phase-3 critical ledger closure, full phase gate (PROTO-01…09)

### Phase 5: Persistence
**Goal**: The dump path is honest about its limits and failures — one enforced value limit, visible load outcomes, generation integrity, bounded memory and blocking — so "a confirmed write is on disk" is a tested guarantee rather than a convention.
**Depends on**: Phase 4 (DumpStore load-outcome port backs S-02; simulator seam exercises dump↔replication)
**Requirements**: PERSIST-01, PERSIST-02, PERSIST-03, PERSIST-04, PERSIST-05, PERSIST-06, PERSIST-07, PERSIST-08, PERSIST-09, PERSIST-10, PERSIST-11, PERSIST-12
**Success Criteria** (what must be TRUE):
  1. The value limit is defined once and enforced everywhere (P-01 red test turns green): a record of exactly `value_max_bytes` survives a restart at any accepted flag value; an invalid value exits with code 2; the 61 440 constant exists exactly once, passed by configuration
  2. Dump-load outcomes are visible and never destroy evidence: starting from `.prev` produces one WARN with path and cause; both generations unusable → ERROR and a nonzero metric; absent files → neither; unusable files are preserved under a separate name, never overwritten by rotation; the format carries a record count or terminator so a line-boundary truncation is recognized as incomplete and one corrupted record discards only that record
  3. Failure handling is bounded and honest: after a backup or empty start the engine is marked dirty so the first rotation never displaces the last good generation (and `SnapshotCreate` right after such a start decodes back to the in-memory state); each dump failure logs one ERROR with the underlying error (ENOSPC, EACCES) and retries with a growing pause up to a cap, with health behavior documented; records skipped due to a reduced value limit are counted and logged, not silently dropped
  4. Resources are bounded: streaming encode keeps a dump cycle at most two state sizes with the memory model documented; a hung fsync cannot block writes, snapshots or shutdown indefinitely; orphan temp files are collected, new directories are synced at the parent, and `fsync_darwin.go` is removed or handles EINTR/ENOTSUP
  5. Structure serves the seams: `persist.Engine` is split along its responsibilities (loader/composition root, store facade, dump scheduler, confirmation accounting, snapshot orchestration) with the `DumpStore` port returning the load outcome and failure cause; `internal/app/persist` timers and `time.Now()` sit behind the same seam as the repl core so the simulator exercises the dump↔replication interaction; equal-version-different-content is resolved by a recorded ADR decision (random tail or deterministic tie-break), implemented and tested
**Plans**: 12/12 plans complete

Plans:
**Wave 1**
- [x] 05-01-PLAN.md — Single-source value limit end to end (`storage.ValueMaxBytes` + `Config.ValueLimit` + `Store.ValueMaxBytes`, codec limit, cmd validation) (D-01/PERSIST-01, PERSIST-10)

**Wave 2** *(blocked on Wave 1)*
- [x] 05-02-PLAN.md — Dump census (`record_count`) and line-oriented decode with four-class stats (D-03/PERSIST-03, PERSIST-10)

**Wave 3** *(blocked on Wave 2)*
- [x] 05-03-PLAN.md — Deterministic equal-version tie-break in the store and on the wire, ADR 0012 (D-12/PERSIST-07; ordered after 05-02 on the shared engine/codec files)

**Wave 4** *(blocked on Wave 3)*
- [x] 05-04-PLAN.md — Streaming save (`SaveStream`/`EncodeTo`) and the port's load-result contract (D-06, D-10/PERSIST-06, PERSIST-11)

**Wave 5** *(blocked on Wave 4)*
- [x] 05-05-PLAN.md — Load outcomes, evidence preservation, orphan sweep, parent sync, dir-fsync tolerance, errno classes (D-02, D-08, D-09/PERSIST-02, PERSIST-09)

**Wave 6** *(blocked on Wave 5)*
- [x] 05-06-PLAN.md — Engine load report, dirty after a backup/empty start, markDirty nudge, skip accounting, binary log/metrics (D-04, D-02, D-09/PERSIST-04, PERSIST-02, PERSIST-10)

**Wave 7** *(blocked on Wave 6)*
- [x] 05-07-PLAN.md — Injected now/after seam, bounded retry ladder with classes, simulator dump↔notify and tie-break convergence scenario (D-11, D-05/PERSIST-12, PERSIST-05)

**Wave 8** *(blocked on Wave 7)*
- [x] 05-08-PLAN.md — Confirmation/flush/shutdown deadlines, `ErrUnconfirmed`, RPC totality (D-07/PERSIST-08)

**Wave 9** *(blocked on Wave 8)*
- [x] 05-09-PLAN.md — Engine split into five internal roles, public API frozen (D-10/PERSIST-11)

**Wave 10** *(blocked on Wave 9)*
- [x] 05-10-PLAN.md — OPERATIONS runbook: load outcomes, retry/health, evidence files, memory model, new metrics (PERSIST-02, PERSIST-05, PERSIST-06)
- [x] 05-11-PLAN.md — SPEC-DEVIATIONS: exactly eight rows amended with current evidence (PERSIST-01, PERSIST-02, PERSIST-07)

**Wave 11** *(blocked on Wave 10)*
- [x] 05-12-PLAN.md — Phase gate: flip the two P-01 rows, 7/7 red-runner with mutants, lint/test/darwin/bench


### Phase 6: API and Semantics
**Goal**: The gRPC API tells the truth — `wait_for_replication` reports what actually happened, errors are total and path-free, Watch and health behave as documented, and the proto is public so consumers can build against it.
**Depends on**: Phase 5 (persistence outcomes; Outbox seam from Phase 3 feeds the honest wait)
**Requirements**: APIS-01, APIS-02, APIS-03, APIS-04, APIS-05, APIS-06, APIS-07
**Success Criteria** (what must be TRUE):
  1. `wait_for_replication` reports honestly: zero known peers returns `(false, 0, 0)`; an `Ack` from a node unknown at write time does not yield `true`; a displaced version answers within 20 ms instead of idling 500 ms
  2. Errors and lifecycle are total: missing snapshot → `NotFound`, no dump → `FailedPrecondition`, no server paths in any message; server shutdown ends Watch streams with `Unavailable` (not `OK`) and health never returns to `SERVING` after shutdown begins; local changes are announced on Watch only after dump confirmation — all stated in the API docs as the contract
  3. Snapshot load is a full rollback (14.1): keys absent from the snapshot are deleted, the result broadcasts to peers, and the rewritten `TestEngine_snapshotCreateLoadRestore` proves the new behavior
  4. `List` does not copy the world: pagination or streaming over the key→version view, with cost independent of value sizes
  5. The proto is public: proto and generated Go code live outside `internal/`; a Go client example builds in CI as a separate module; the last-read-value file pattern is documented as consumer-library territory (14.7)
**Plans**: 13/13 plans complete

Plans:
**Wave 1**
- [x] 06-01-PLAN.md — Public API schema: move to api/supkvapi/v1, add List page fields, compatibility alias, import sweep, buf gates

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 06-02-PLAN.md — Honest wait tracer: write-time peer capture, zero-peer false, signature ripple (incl. NotifyInstalled and notifyReplayedWrite)
- [x] 06-03-PLAN.md — Standalone Go client example module and CI gate

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 06-04-PLAN.md — Total, path-free error mapping with adapter-side failure classes
- [x] 06-11-PLAN.md — Wait expansion: tracker promise set, no-growth discovery flush, regression matrix

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 06-05-PLAN.md — Keyset-paginated List over the live key→version view (owns the serialized page-budget constants)

**Wave 5** *(blocked on Wave 4 completion)*
- [x] 06-06-PLAN.md — Durability-gate tracer: generation-tagged installs and the confirmed watermark

**Wave 6** *(blocked on Wave 5 completion)*
- [x] 06-12-PLAN.md — Watch expansion: nil-safe startup suppression, bounded pending, Unavailable shutdown

**Wave 7** *(blocked on Wave 6 completion)*
- [x] 06-07-PLAN.md — Explicit gRPC transport limits and one-way health
- [x] 06-08-PLAN.md — Snapshot load as a full rollback with broadcast

**Wave 8** *(blocked on Wave 7 completion)*
- [x] 06-09-PLAN.md — docs/API.md phase-owned contract sections and doc-anchor guard
- [x] 06-10-PLAN.md — Documentation and registry ripple (EXAMPLES, MULTI-NODE, SPEC-DEVIATIONS, AGENTS)

**Wave 9** *(blocked on Wave 8 completion)*
- [x] 06-13-PLAN.md — Phase gate: frozen-file base diff, red-runner 7/7, make lint/test/buf-breaking

### Phase 7: Observability and Configuration
**Goal**: An operator can see what the node is doing and misconfiguration fails fast — lifecycle logs, one validation point, metric definitions that match the binary, distinct exit codes, and a machine-enforced dependency boundary.
**Depends on**: Phase 6
**Requirements**: OBSV-01, OBSV-02, OBSV-03, OBSV-04, OBSV-05, OBSV-06, OBSV-07, OBSV-08
**Success Criteria** (what must be TRUE):
  1. The node narrates its lifecycle: a startup entry with version and effective config; one entry each for loaded dump generation, dump failures, gate opening, peer appear/disappear/restart and drift warnings; a fatal bind error at `SUPKV_LOG_LEVEL=error` produces an ERROR entry, not a silent exit 1; the Spec-mandated entries (dump failure, start without usable dumps, foreign proto_version, duplicate node_id) exist at the right levels, and the drift runbook describes a condition that exists in the binary
  2. Configuration is validated in one place: a single validation function covers ranges and relations (negative/zero values, `sync-chunk-bytes` overflow panic, `retransmit-interval=1ns` busy loop, `peer_timeout ≤ hello_interval`, `ack_timeout > ack_timeout_max`, positional args ignored), exits with code 2 naming the parameter, is table-tested, with flag parsing extracted from `main.go`
  3. Metrics mean what they say (14.4): the conflict counter counts only the displacement of a local unconfirmed record by another value; a failed send is not counted as sent; `process_resident_memory_bytes` reports current RSS; dump-duration and drift-counter semantics are defined — and every definition is confirmed by a test
  4. Exit codes and shutdown are distinct and ordered: usage error, startup failure and final-dump failure have distinct codes; a `Serve` error does not skip the final dump; the final dump is time-bounded; the signal handler is installed before dump load and bind; `--help` lists the five env-only variables
  5. Boundaries are machine-enforced and units decomposed: a depguard rule forbids adapter imports from `app`/`port` and the observed cross-imports (`metrics`→`udprepl`, `grpcapi`→`persistfile`/`*persist.Engine`) — any boundary change is a visible `.golangci.yml` diff; socket, loop, mirror and codec are separate units in `udprepl` and `main.go` is wiring only
**Plans**: 3/11 plans executed

Plans:
**Wave 1**
- [x] 07-01-PLAN.md — Tracer: split udprepl into socket/codec/loop/mirror units and relocate SentByType to the core package (OBSV-08, OBSV-07; D-08, D-07)
- [x] 07-02-PLAN.md — Tracer: internal/config with one Validate rule table, wiring-only main, sync-chunk/value bounds, codec value guard and the 2 m max-clock-lead default (OBSV-02, OBSV-08; D-03, D-04)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 07-03-PLAN.md — Tracer: three additive depguard rules with per-rule import probes and the written grpcapi exception (OBSV-07; D-07)
- [ ] 07-04-PLAN.md — Tracer: exit codes 0/1/2/3, injectable run, signals before wire, Serve errors still dumping (OBSV-04; D-06)
- [ ] 07-05-PLAN.md — Tracer: received-lead drift signal in Clock.Update plus positive-lead counting for records and listing maxima (OBSV-05, OBSV-03; D-02)

**Wave 3** *(blocked on Wave 2 completion)*
- [ ] 07-06-PLAN.md — Tracer: pending-displacement conflict counting, keyless rate-limited apply-failure WARN, rebuilt stand conflict proof (OBSV-03; D-05)

**Wave 4** *(blocked on Wave 3 completion)*
- [ ] 07-07-PLAN.md — Tracer: gate-open reason, typed peer/drift seams, startup and load-outcome records (OBSV-01, OBSV-06, OBSV-05; D-01)

**Wave 5** *(blocked on Wave 4 completion)*
- [ ] 07-08-PLAN.md — Tracer: unbound_listing discard class end-to-end, proto-version WARN on both branches, catalog level matrix (OBSV-06, OBSV-03; D-04, D-01)

**Wave 6** *(blocked on Wave 5 completion)*
- [ ] 07-09-PLAN.md — Tracer: send-only-on-success accounting, four Phase-4 counter families, current-RSS split, definition guard (OBSV-03; D-05)

**Wave 7** *(blocked on Wave 6 completion)*
- [ ] 07-10-PLAN.md — Tracer: OPERATIONS lifecycle/exit/drift/metrics contract with anchors, five registry rows, AGENTS architecture note (OBSV-01…07; D-01, D-02, D-06, D-07)

**Wave 8** *(blocked on Wave 7 completion)*
- [ ] 07-11-PLAN.md — Phase gate: frozen-hash and mutation-target pins, 7/7 red-runner, lint/test/buf/darwin/client-example (OBSV-01…08)

### Phase 8: Performance
**Goal**: Performance claims are measurements with documented conditions — the write-latency model, the burst bound, disk amplification, the memory model and absent-peer latency are all documented and standing-tested, not folklore.
**Depends on**: Phase 7 (all fixes in; `make perf` builds on the stabilized stand)
**Requirements**: PERF-01, PERF-02, PERF-03, PERF-04, PERF-05, PERF-06
**Success Criteria** (what must be TRUE):
  1. The write-latency model is documented for both profiles (14.3): docs state that under sustained write flow latency reaches `dump_max_wait` + dump time by design; the fast-response profile (Debounce/MaxWait tuning) is documented with its disk cost; the "~10 ms under 1000 writes/s" claim is tied to its measurement conditions
  2. The 1000 writes/s burst over distinct keys is a tested bound: the XPORT-02(3) scenario runs as a standing test, and the docs describe what is and is not guaranteed at that rate
  3. Amplification and memory are measured, or the gap is recorded: disk-write amplification on target-grade media (Spec open question #2) and the memory model from stand measurements (with PERSIST-06)
  4. Write latency with an absent peer and a growing unconfirmed set stays near live-peer latency, proven by a benchmark (same fix as XPORT-02)
  5. `make perf` runs the stand suite with thresholds, and every published number carries a hardware description
**Plans**: TBD

### Phase 9: Documentation and Release
**Goal**: The documentation is the as-built contract and the milestone ships — every documented mismatch closed, every reference anchored to a test that exists, the deviation registry fully closed, and v1.1 released.
**Depends on**: Phase 8
**Requirements**: DOCS-01, DOCS-02, DOCS-03, DOCS-04, DOCS-05, DOCS-06, DOCS-07, DOCS-08, DOCS-09, DOCS-10
**Success Criteria** (what must be TRUE):
  1. All 52 documented mismatches (M-01…M-52) are closed by a doc or code fix; `docs/CONFIGURATION.md` covers every flag and env var (defaults, units, ranges, relations, "must match on all nodes") with a test iterating registered flags against the table; `docs/DEPLOYMENT.md` + the systemd unit cover the two-board topology end to end, run in CI and ship in the release archives
  2. Operational references are complete and anchored to the binary: the Spec's 18-row failure-behavior table (symptom → behavior → recovery → metric → log line) and runbooks for dump failure/full disk, missing peer, growing unconfirmed set, write conflict, incompatible versions, corrupted dump and duplicate `node_id` — every row references a metric or log line that exists in the binary; a metrics reference and log-message catalog are cross-checked by a test; `docs/FORMATS.md` defines the dump grammar, checksum and envelope with a byte-identical encode test
  3. API and as-built documentation land: `docs/API.md` with per-RPC semantics, code tables, Watch contract, write-latency model and client patterns (with APIS-06); the as-built architecture description and spec-deviation table close the Phase 1 registry with no unclosed rows (milestone exit criterion); `AGENTS.md`/`.claude/CLAUDE.md` stack sections match `go.mod` and `go list ./...`
  4. Every "proven" claim names its test and doc drift breaks CI (`TestOperationsDoc` verifies against the binary); the macOS quickstart pair works; darwin is demoted to development-only in README and release archives (14.10); CONTRIBUTING, SECURITY, ADR index, simulator guide, release procedure and benchmark results land; phase/decision numbers are scrubbed from proto comments and CHANGELOG; the branch-protection ruleset check command works as written; the v1.1 release ships
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → … → 9 (v1.1; v1.0's phases 1-7 are archived history under [milestones/v1.0-phases/](milestones/v1.0-phases/))

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. CI Bootstrap and Repo Skeleton | v1.0 | 4/4 | Complete    | 2026-11-19 |
| 2. Engine — HLC Versioning and LWW Store | v1.0 | 17/17 | Complete    | 2026-11-21 |
| 3. Persistence — Dump and Write Confirmation | v1.0 | 13/13 | Complete | 2026-11-03 |
| 4. Replication and Deterministic Simulator | v1.0 | 13/13 | Complete    | 2026-11-22 |
| 5. gRPC API and Binary | v1.0 | 12/12 | Complete    | 2026-11-22 |
| 6. Hardening — Failure Scenarios, Observability, Benchmarks | v1.0 | 13/13 | Complete | 2026-11-16 |
| 7. v1.0 — Docs, Examples and Release | v1.0 | 3/11 | In Progress |  |
| 1. Spec Reference and Decisions | v1.1 | 4/4 | Complete | 2026-11-19 |
| 2. Stand and Simulator as Tests | v1.1 | 17/17 | Complete | 2026-11-21 |
| 3. Transport | v1.1 | 13/13 | Complete | 2026-11-22 |
| 4. Protocol | v1.1 | 0/? | Not started | - |
| 5. Persistence | v1.1 | 0/? | Not started | - |
| 6. API and Semantics | v1.1 | 13/13 | Complete | 2026-11-23 |
| 7. Observability and Configuration | v1.1 | 2/11 | In Progress | - |
| 8. Performance | v1.1 | 0/? | Not started | - |
| 9. Documentation and Release | v1.1 | 0/? | Not started | - |
