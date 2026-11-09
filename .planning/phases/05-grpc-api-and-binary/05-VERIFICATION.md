---
phase: 05-grpc-api-and-binary
verified: 2026-11-09T15:02:03Z
status: passed
score: 28/28 must-haves verified
covered_files: [".planning/phases/05-grpc-api-and-binary/05-01-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-01-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-02-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-02-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-03-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-03-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-04-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-04-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-05-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-05-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-06-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-06-SUMMARY.md", "Makefile", "buf.gen.yaml", "buf.yaml", "cmd/sup-kv/main.go", "cmd/sup-kv/main_test.go", "go.mod", "go.sum", "internal/app/persist/engine.go", "internal/app/persist/engine_test.go", "internal/app/repl/core.go", "internal/app/repl/retransmit_test.go", "internal/app/repl/sync_test.go", "internal/app/storage/store.go", "internal/app/storage/store_test.go", "internal/grpcapi/apiproto/supkvapi.pb.go", "internal/grpcapi/apiproto/supkvapi.proto", "internal/grpcapi/apiproto/supkvapi_grpc.pb.go", "internal/grpcapi/gate.go", "internal/grpcapi/gate_test.go", "internal/grpcapi/server.go", "internal/grpcapi/server_test.go", "internal/grpcapi/wait.go", "internal/grpcapi/wait_integration_test.go", "internal/grpcapi/wait_test.go", "internal/grpcapi/watch.go", "internal/grpcapi/watch_test.go", "internal/udprepl/repl.go", "internal/udprepl/repl_test.go"]
covered_digest: "v4:sha256:3ac1add124a6b015085d8c839e447ca9b45ad4a313114ca589fc9e52829d3770"
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "systemd unit behavior end-to-end (TOOL-01: Type=simple unit + SIGTERM lifecycle observed on a real host)"
    addressed_in: "Phase 7"
    evidence: "Phase 7 goal: 'A stranger can deploy and operate a two-node sup-kv cluster from the documentation alone'; 05-06-PLAN defers systemd documentation to Phase 7 ('Type=simple with the health endpoint as readiness — documentation lands in Phase 7'); 05-VALIDATION.md records the unit e2e as the single manual-only item, deployment-environment state. The binary-side observable contract it depends on (SIGTERM -> exit 0 after final dump, health readiness, exit codes) is behaviorally proven by TestMain_lifecycle and the ./bin/sup-kv probes in this verification."
decision_coverage:
  honored: 19
  total: 19
  not_honored: []
---

# Phase 5: gRPC API and Binary — Verification Report

**Phase Goal:** sup-kv becomes a usable product — a single static binary exposing the typed gRPC surface with Watch, startup gate, `wait_for_replication` and standard health checking
**Verified:** 2026-11-09T15:02:03Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Must-haves merged from ROADMAP success criteria 1-5 (the contract) and PLAN frontmatter truths (plans 05-01..05-06). Every truth below has behavioral evidence — a named passing test or a directly executed command on current HEAD (9e754b5), not a SUMMARY claim.

| # | Truth | Source | Status | Evidence |
|---|-------|--------|--------|----------|
| 1 | gRPC Set then Get returns identical value and 16-byte version (D-01) | SC1, 05-01 | ✓ VERIFIED | TestServer_setGetRoundTrip PASS (named run, -race); wire contract supkvapi.proto declares Record{key,version,value,deleted} |
| 2 | Every sentinel maps to its pinned RPC code via errors.Is (NotFound/AlreadyExists/InvalidArgument/Unavailable×2/InvalidArgument) | SC1, 05-01 | ✓ VERIFIED | TestServer_errorMapping PASS; one shared mapError helper in server.go:291-310 with the six-sentinel table |
| 3 | create_if_absent=true returns AlreadyExists for any existing record including a tombstoned one | SC1, 05-01 | ✓ VERIFIED | TestServer_errorMapping rows (live + tombstoned variants) PASS; Set handler routes engine.SetIfAbsent |
| 4 | List returns sorted live keys under prefix; empty prefix = all keys; no pagination (D-12..D-15) | SC1, 05-01 | ✓ VERIFIED | TestServer_list PASS; server.go List filters Deleted, HasPrefix, slices.Sort, single response |
| 5 | go.mod/go.sum are tidy-exact; grpc v1.85.0 direct require; protoc-gen-go-grpc v1.6.2 tool directive | 05-01 | ✓ VERIFIED | `go mod tidy && git diff --exit-code go.mod go.sum` exit 0; go.mod lines 14/9/295 |
| 6 | Watch subscriber first receives current state as ordinary events (tombstones as deleted=true), then live changes — no duplicates, no misses (D-02/D-03) | SC2, 05-03 | ✓ VERIFIED | TestWatch_snapshotThenLive PASS (named run); TestWatch_exactCut PASS (concurrent writers: strictly increasing per-key versions, no repeated (key,version)) |
| 7 | Events fire only on actual installs; losing/equal-version merge produces no event (D-06) | SC2, 05-03 | ✓ VERIFIED | TestStore_onChange PASS (8 subtests incl. loser/equal silence locks); TestWatch_replicatedChange PASS |
| 8 | Local and replicated changes arrive on one stream in arrival order, indistinguishable by content (D-05/D-07) | SC2, 05-03 | ✓ VERIFIED | TestWatch_replicatedChange PASS; watch.go toProto carries no origin field |
| 9 | Slow consumer disconnected (ResourceExhausted) while the store keeps accepting writes — the tap never blocks the store write lock | SC2, 05-03 | ✓ VERIFIED | TestWatch_slowConsumer PASS (ResourceExhausted termination + store Set liveness throughout); Ingest is select-with-default into bounded 4096 ingress (watch.go:213-219). Noted deviation: the shed is triggered by flooding hub.Ingest (the identical entry point the storage tap calls) because gRPC flow control absorbs test-scale real-write bursts — documented in 05-03-SUMMARY |
| 10 | Prefix filter applies to snapshot and live events; empty prefix = all keys (D-04) | SC2, 05-03 | ✓ VERIFIED | TestWatch_prefix PASS |
| 11 | Before the gate opens, Set/Delete return Unavailable while Get/List/Watch succeed (API-04) | SC3, 05-04 | ✓ VERIFIED | TestGate_writesUnavailableReadsOK PASS (named run); gate check is first statement of Set/Delete (server.go:132-134, 172-174) before validation; reads contain no gate check |
| 12 | Gate opens on the earlier of the injected sync flip or startup_gate elapsing (A8) | SC3, 05-04 | ✓ VERIFIED | TestGate_opensOnSync + TestGate_opensOnTimeout PASS (full-suite run, exit 0); RunGate 20ms poll + timer backstop, gate.go:122-145 |
| 13 | Standard health endpoint reports NOT_SERVING for empty service name before the gate, SERVING after — bound 1:1 (API-06) | SC3, 05-04 | ✓ VERIFIED | TestHealth_reflectsGate PASS (named run, both sides); single CAS'd openGate stores flag and flips health (gate.go:152-156); registered alongside SupKV in main.go:367 |
| 14 | SnapshotCreate/SnapshotLoad gated like writes; persistfile.ErrInvalidSnapshotName -> InvalidArgument (A5, API-07) | SC3/SC4, 05-04 | ✓ VERIFIED | TestSnapshot_gated + TestSnapshot_invalidName (empty, "..", "a/b", 65 chars) + TestSnapshot_createLoadRoundTrip PASS (named run); adapter maps only, never validates |
| 15 | wait_for_replication reports the D-08 aggregate and never changes the write outcome (API-05) | SC4, 05-05 | ✓ VERIFIED | TestWait_timeout PASS (Set RPC still OK with acked=false 0/1); TestWait_acksArrive PASS (1/1); wiring strictly after engine confirmation (server.go:150-156) |
| 16 | Zero known peers reports acked=true immediately, total_peers 0 (D-11), including nil seams | SC4, 05-05 | ✓ VERIFIED | TestWait_zeroPeers PASS (named run, elapsed well under bound); wait.go:95-105 nil-seam and zero-count short-circuits |
| 17 | Evicted peer never counted as an Ack; superseded entry reports acked=false with counts frozen (Q2) | SC4, 05-05 | ✓ VERIFIED | TestWait_evictionNotAck PASS; TestWait_superseded PASS (named run) |
| 18 | Wait waits for the (key, version) entry to appear before absence means drained — no false acked from the notify-queue race (Pitfall 4) | SC4, 05-05 | ✓ VERIFIED | TestWait_appearanceRace PASS (named run); plus the CR-01 fix integration test TestWaitIntegration_ackedThroughRealAdapterMirror PASS (named run, -race): two real adapters, Set with wait_for_replication through the real mirror, acked=true 1/1 |
| 19 | Timeout is per-request wait_timeout_ms when positive, else operator default (D-09), overflow-safe (WR-02) | SC4, 05-05 | ✓ VERIFIED | TestWait_perRequestTimeout + TestWait_overflowingTimeoutClamped PASS (named runs); waitTimeout clamps at MaxInt64/ms then 10x operator ceiling (wait.go:210-225) |
| 20 | SIGTERM produces clean exit 0 after a final write-confirmed dump; committed data never memory-only across shutdown (D-19) | SC5, 05-06 | ✓ VERIFIED | TestMain_lifecycle PASS (real process, real TCP: health SERVING -> Set/Get -> SIGTERM -> exit 0 -> dump file contains the key -> no panic); shutdown() in main.go:397-428 follows the six pinned steps; final-dump failure exits 1 |
| 21 | Every knob configurable as flag and SUPKV_ env with precedence flag > env > default (D-16/D-17) | SC5, 05-06 | ✓ VERIFIED | TestMain_precedence PASS (env-chosen port then flag-chosen port, env port unbound in run 2); ./bin/sup-kv with no knobs exits 2 with usage naming all 19 flag/env knobs (executed in this verification) |
| 22 | --version prints version + commit + Go runtime, ldflags-injected, dev fallback (D-18) | SC5, 05-06 | ✓ VERIFIED | `./bin/sup-kv --version` -> `sup-kv 9e754b5 9e754b5 go1.27.3` (executed); TestMain_version PASS |
| 23 | Engine.Flush() turns residual dirtiness (remote merges, no waiter) into a confirmed dump before engine ctx cancel | SC5, 05-06 | ✓ VERIFIED | TestFlush_ family (remoteDirtiness / closed / clean) PASS (named run); Flush exported at engine.go delegating to flush() with the D-19 exit-sweep doc |
| 24 | Shipped artifact builds as one static CGO_ENABLED=0 binary | SC5, 05-06 | ✓ VERIFIED | `make build` + `file bin/sup-kv` -> "statically linked" (executed); Makefile:26-28 |
| 25 | PendingSnapshot distinguishes a peer that Acked (in Acked) from one evicted by peer_timeout (absent from both sets) | 05-02 | ✓ VERIFIED | TestRetransmit_ackDrainSemantics PASS (named run: sorted insertion, eviction-regression lock, duplicate idempotence); evictPeers never touches acked (core.go:768) |
| 26 | Core.SyncArrived() false until first DataSync handled, true forever after (A1) | 05-02 | ✓ VERIFIED | TestSync_syncArrived PASS (named run); flag set in handleDataSync (core.go:559), accessor core.go:901 |
| 27 | gRPC-side readers obtain both replication states from any goroutine via adapter atomics, never the single-driver core | 05-02 | ✓ VERIFIED | TestMirror_pendingAckVisible + TestMirror_syncArrivedFlip PASS (named run, real loop); copy-on-write pendingCopy atomic + SyncArrived atomic, refreshed at the post-iteration mirrorState site (repl.go:301, 469-487) |
| 28 | No callbacks or channels added to the core — results, not callbacks | 05-02 | ✓ VERIFIED | grep of core.go: only pre-existing Config.Log func field; phase diff added bool field, acked slice, accessors per 05-02 acceptance criteria |

**Score:** 28/28 truths verified (0 present, behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | systemd unit behavior end-to-end (real host, Type=simple unit) | Phase 7 | Phase 7 goal covers deploy/operate from documentation; 05-06-PLAN explicitly defers systemd documentation to Phase 7; 05-VALIDATION.md lists it as the only manual-only item. The binary-side contract it supervises (SIGTERM -> exit 0 + final dump, health readiness, exit codes 0/1/2) is fully behaviorally proven at the binary level. |

### Required Artifacts

The gsd-tools `verify artifacts` verb parsed zero artifacts from the plans' string-form `must_haves.artifacts` lists (total 0, exit 1 via `query`, exit 0 with empty result via the space verb) — a plan-side format variance, not a codebase finding. All artifacts were verified manually at the three levels (exists / substantive / wired):

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/grpcapi/apiproto/supkvapi.proto` | whole-phase wire contract, 7 RPCs, D-01..D-15 shapes | ✓ VERIFIED | 5.2 KB; all 7 RPCs declared; create_if_absent/wait_for_replication/wait_timeout_ms/D-08 report fields/prefix present; buf lint + format exit 0 |
| `internal/grpcapi/apiproto/supkvapi.pb.go`, `supkvapi_grpc.pb.go` | committed generated code | ✓ VERIFIED | 32 KB / 15 KB; protoc-gen-go-grpc marker count 1; buf-generated (no hand edits — 05-REVIEW verified regeneration-identical) |
| `internal/grpcapi/server.go` | unary handlers + mapError + seams | ✓ VERIFIED | 14.7 KB; Get/Set/Delete/List/SnapshotCreate/SnapshotLoad wired to persist.Engine; mapError six-sentinel table |
| `internal/grpcapi/watch.go` | WatchHub + Watch handler | ✓ VERIFIED | 16.4 KB; hub goroutine, exact cut at line 314 (`Version.Compare(snapVer) > 0`), disconnect-never-drop, nil-hub defensive path (production always wires the hub via main.go:324) |
| `internal/grpcapi/gate.go` | ServerOptions + resolver + health | ✓ VERIFIED | 6.8 KB; RunGate poll loop, CAS'd openGate, wait seams documented |
| `internal/grpcapi/wait.go` | waitForReplication | ✓ VERIFIED | 10.8 KB; two-phase wait, Acked-only counting, WR-01 clamp, WR-02 bounds |
| `internal/app/storage/store.go` | Config.OnChange tap from install | ✓ VERIFIED | OnChange field + nil normalization (97-103), write routed through install (163-167), tap fired with stored record (251-260) |
| `internal/app/repl/core.go` | SyncArrived + PendingInfo.Acked | ✓ VERIFIED | syncArrived flag (262-268), acked slice + ackedInsert sorted/idempotent (508-517), WR-01 doc alignment (484-486) |
| `internal/udprepl/repl.go` | adapter mirrors | ✓ VERIFIED | syncArrived/pendingCopy atomics (154-162), mirrorState at post-iteration site (469-487), CR-01 drainedRetention rows (63-76, 494-522) |
| `internal/app/persist/engine.go` | exported Flush | ✓ VERIFIED | Flush() delegating to flush(), doc names D-19 and the exit-sweep gap |
| `cmd/sup-kv/main.go` | flags/env/signals/wiring only | ✓ VERIFIED | 18.9 KB; 19 knobs + SUPKV_ twins, D-19 shutdown, full ServerOptions wiring, health registration, ldflags vars |
| `cmd/sup-kv/main_test.go` | binary smoke family | ✓ VERIFIED | 14.2 KB; real-process tests, loud capability-skip only, value/behavioral assertions |
| `Makefile`, `go.mod`, `go.sum`, `buf.gen.yaml`, `buf.yaml` | build/codegen/deps | ✓ VERIFIED | build target CGO_ENABLED=0 + ldflags; proto target = `go tool buf generate` with both go-tool plugins; grpc v1.85.0 direct require; tidy-exact |

Wiring (import + use) confirmed for every artifact: all are referenced by main.go, the harness, or the packages' own consumers; no orphaned files.

### Key Link Verification

All key links from the six plans' `must_haves.key_links` traced in code (manual — the verb is not applicable to the plans' prose-form links):

| From | To | Via | Status |
| ---- | -- | --- | ------ |
| Server handlers (Get/Set/SetIfAbsent/Delete/List/Snapshot/Restore) | persist.Engine | direct engine calls, server.go:106/143/145/178/202/227/247 | ✓ WIRED |
| Server notify seam | udprepl.Adapter.NotifyLocalWrite | main.go:325 injects adapter.NotifyLocalWrite as LocalNotify; Set/Delete call notifyLocalWrite | ✓ WIRED |
| storage.Store.install | Config.OnChange -> WatchHub.Ingest | store.go:260 fires; main.go:279 threads `OnChange: hub.Ingest` through persist.Open | ✓ WIRED |
| Exact-cut suppression | RecordVersion.Compare | watch.go:314 `ev.rec.Version.Compare(snapVer) > 0` | ✓ WIRED |
| ServerOptions.SyncArrived -> gate resolver -> gated() | adapter mirror | main.go:326 `SyncArrived: adapter.SyncArrived`; RunGate polls it; Set/Delete read gated() | ✓ WIRED |
| Gate resolution -> health.SetServingStatus | 1:1 binding | gate.go:152-156 single CAS branch; health registered main.go:367 | ✓ WIRED |
| waitForReplication | KnownPeers/PendingCopy seams | wait.go:95/114/188 read only injected closures; main.go:327-328 injects adapter mirrors | ✓ WIRED |
| Snapshot handlers | persistfile name validation | server.go:227/247 call engine.Snapshot/Restore; adapter only maps ErrInvalidSnapshotName | ✓ WIRED |
| handleAck/handleDataSync -> core state -> adapter mirrors | post-iteration mirrorState | repl.go:301 mirrorState after every loop event; core.go:500 acked insert, 559 syncArrived | ✓ WIRED |
| main wiring graph | hub -> engine -> adapter -> server -> gate/health -> serve | main.go:297-354 (wire), exactly the plan's key_links graph incl. `go hub.Run(apiCtx)` (334) | ✓ WIRED |
| SIGTERM -> shutdown order | D-19 six steps | main.go:397-428: health NOT_SERVING -> apiCancel -> bounded GracefulStop/Stop -> replCancel+join -> Flush -> engineCancel | ✓ WIRED |
| Makefile build | static stamped bin/sup-kv | Makefile:26-28 CGO_ENABLED=0 + -X main.version/main.commit | ✓ WIRED |
| go.mod tool block | protoc-gen-go-grpc via go build, never go install | go.mod:9/295; buf.gen.yaml local plugins `go tool protoc-gen-go-grpc`; make proto = `go tool buf generate` | ✓ WIRED |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Get handler | record | engine.Get (persist engine over real store) | Yes | ✓ FLOWING |
| List handler | keys | engine.Records() snapshot, filtered/sorted | Yes | ✓ FLOWING |
| Watch stream | events | storage install tap -> hub ingress -> subscribers | Yes | ✓ FLOWING |
| Wait report | acked/counts | injected adapter mirrors (core-acknowledged state) | Yes | ✓ FLOWING |
| Health status | gate atomic | resolver openGate / shutdown | Yes | ✓ FLOWING |
| Set response | version | engine-stamped RecordVersion | Yes | ✓ FLOWING |

No static returns, no hardcoded literals, no mocks in any production response path. (Test fakes exist only in _test.go files, per the sanctioned in-memory fake discipline.)

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Set/Get round trip | `go test -race ./internal/grpcapi/ -run '^TestServer_setGetRoundTrip$' -count=1` | ok 1.073s | ✓ PASS |
| Watch exact cut / snapshot-then-live / slow consumer | named -run batch (14 tests incl. gate/health/wait/snapshot) | 14/14 PASS under -race | ✓ PASS |
| Gate writes-unavailable reads-ok; health reflects gate | same batch | PASS | ✓ PASS |
| CR-01 fix: acked=true through real adapter mirror | `go test -race ./internal/grpcapi/ -run '^TestWaitIntegration_ackedThroughRealAdapterMirror$' -count=1 -v` | PASS 0.05s | ✓ PASS |
| Binary SIGTERM lifecycle | `go test -race ./cmd/sup-kv/ -run '^TestMain_lifecycle$' -count=1 -timeout 180s` | PASS | ✓ PASS |
| repl seams (Acked/syncArrived) + mirrors | named runs across repl/udprepl packages | ok | ✓ PASS |
| storage OnChange tap | `go test -race ./internal/app/storage/ -run '^TestStore_onChange$' -count=1` | ok | ✓ PASS |
| Flush semantics | `go test -race ./internal/app/persist/ -run '^TestFlush_' -count=1` | ok | ✓ PASS |
| Static artifact + version + exit codes | `make build && file bin/sup-kv && ./bin/sup-kv --version && ./bin/sup-kv` | statically linked; `sup-kv 9e754b5 9e754b5 go1.27.3`; exit 2 with usage | ✓ PASS |
| Full suite (single run) | `make test` (go test -race -cover ./...) | exit 0, 10/10 packages ok, no FAIL/panic/DATA RACE; grpcapi 91.1% cover, repl 97.7%, storage 100% | ✓ PASS |
| Full lint gate | `make lint` (config verify + run ./...) | exit 0, no findings | ✓ PASS |
| Tidy gate | `go mod tidy && git diff --exit-code go.mod go.sum` | exit 0 | ✓ PASS |
| buf gates | `go tool buf lint` + `go tool buf format --diff --exit-code` | exit 0 both | ✓ PASS |

### Probe Execution

No phase-declared probe scripts (`scripts/*/tests/probe-*`) exist; the phase's runnable-check contract is the Go test suite itself, covered above.

### Requirements Coverage

Requirement IDs claimed across the phase's plans: 05-01 [API-01, API-02], 05-02 [API-04, API-05], 05-03 [API-03], 05-04 [API-04, API-06, API-07], 05-05 [API-05], 05-06 [TOOL-01]. Union = exactly the phase's 8 IDs. REQUIREMENTS.md maps exactly API-01..API-07 + TOOL-01 to Phase 5 — no orphaned requirements, no unclaimed IDs.

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| API-01 | 05-01 | gRPC service with protobuf; Get/Set (incl. create-if-absent)/Delete with OK/NotFound/AlreadyExists/InvalidArgument/Unavailable | ✓ SATISFIED | Truths 1-3; proto + handlers + TestServer_errorMapping |
| API-02 | 05-01 | List keys under prefix, sorted order | ✓ SATISFIED | Truth 4; TestServer_list |
| API-03 | 05-03 | Watch stream covering local and replicated changes alike | ✓ SATISFIED | Truths 6-10; TestWatch_* family |
| API-04 | 05-02, 05-04 | Writes Unavailable during startup gate; reads immediate | ✓ SATISFIED | Truths 11-12, 25-26; TestGate_* + TestSync_syncArrived |
| API-05 | 05-02, 05-05 | wait_for_replication: waits for Acks within timeout, reports, never changes outcome | ✓ SATISFIED | Truths 15-19, 27; TestWait_* family + CR-01 integration test |
| API-06 | 05-04 | Standard health endpoint reflects gate (NOT_SERVING until open) | ✓ SATISFIED | Truth 13; TestHealth_reflectsGate |
| API-07 | 05-04 | Snapshot create/load through the API | ✓ SATISFIED | Truth 14; TestSnapshot_* |
| TOOL-01 | 05-06 | Single static binary, flags/env config, --version, systemd-friendly | ✓ SATISFIED | Truths 20-24; TestMain_* + make build + executed binary probes |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/grpcapi/server_test.go | API-01/02 | 3 | 0 | none | value/behavioral (codes, bytes, ordering) | adequate |
| internal/grpcapi/watch_test.go | API-03 | 5 | 0 | none | behavioral (exact cut, ResourceExhausted, store liveness) | adequate |
| internal/grpcapi/gate_test.go | API-04/06/07 | 7 | 0 | none | behavioral (gate both sides, hostile names, round trip over real persistfile) | adequate |
| internal/grpcapi/wait_test.go | API-05 | 10 | 0 | none | behavioral (adversarial mirror states, timing bounds, clamps) | adequate |
| internal/grpcapi/wait_integration_test.go | API-05 (CR-01) | 1 | 0 (loud capability skip only) | none | end-to-end behavioral (two real adapters, acked=true 1/1) | adequate |
| internal/app/repl/retransmit_test.go, sync_test.go | API-04/05 seams | extended families | 0 | none | value/behavioral | adequate |
| internal/udprepl/repl_test.go | API-05 mirrors | TestMirror_* 2 | 0 | none | behavioral over the real event loop | adequate |
| internal/app/storage/store_test.go | API-03 tap | TestStore_onChange (8 subtests) | 0 | none | value/behavioral (loser silence, stored-record equality) | adequate |
| internal/app/persist/engine_test.go | TOOL-01 Flush | TestFlush_* 3 | 0 | none | value (dump bytes contain record) | adequate |
| cmd/sup-kv/main_test.go | TOOL-01 | 3 | 0 (loud skip path only) | none | end-to-end behavioral (real process/TCP, exit codes, dump content) | adequate |

**Disabled tests on requirements:** 0 (only documented loud capability-probe skips: loopback broadcast probe in the integration test, go-tool probe in the smoke TestMain — both name what was probed)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts (gate `check.decision-coverage-verify`): 19/19 honored, 0 not honored — including D-01..D-07 (watch contract), D-08..D-11 (wait report), D-12..D-15 (list), D-16..D-19 (binary/config/shutdown).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | No TBD/FIXME/XXX debt markers in any phase-modified file | — | — |
| (none) | — | No stubs: the only `Unimplemented` returns are the mandatory protoc-gen-go-grpc v1.6.2 embed and watch.go:366's nil-hub defensive path (production wiring always constructs the hub, main.go:298/324; documented as a programming-error guard, not a placeholder) | — | — |

### Review Findings Disposition (context)

The deep review (05-REVIEW.md) found CR-01 (wait_for_replication could never report acked=true against the real mirror), WR-01 (outside-wait-set Acks inflating acked_count), WR-02 (wait_timeout_ms int64 overflow). All three are fixed on HEAD (05f84de, af3d5c4, 6c171bf) and re-verified here: the new integration test passes against two real adapters (truth 18), TestWait_strangerAcksClampedToTotal and TestWait_overflowingTimeoutClamped pass (truths 17/19). IN-01 (unreachable mapError status-passthrough branch) is open by triage at info level — harmless forward compatibility, documented, non-blocking.

### Human Verification Required

N/A — infrastructure/foundation phase (internal gRPC API + binary; no UI or user-facing visual surface). All acceptance criteria are verifiable programmatically and were verified above. The single deployment-environment item (systemd unit end-to-end) is deferred to Phase 7 with rationale recorded under Deferred Items — the binary-side contract it supervises is fully behaviorally proven.

### Gaps Summary

None. All 28 merged must-have truths verified with behavioral evidence on current HEAD; all artifacts substantive and wired; all key links traced in code; all 8 requirement IDs satisfied with no orphans; full quality gates (make test, make lint, tidy, buf lint/format) green in this verification's own runs; the phase goal — a single static binary exposing the typed gRPC surface with Watch, startup gate, wait_for_replication and standard health checking — is observably achieved.

---

_Verified: 2026-11-09T15:02:03Z_
_Verifier: Claude (gsd-verifier)_
