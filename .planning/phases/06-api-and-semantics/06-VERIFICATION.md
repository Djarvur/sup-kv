---
phase: 06-api-and-semantics
verified: 2026-11-23T03:46:24Z
status: passed
score: 79/79 must-haves verified
covered_files:
  - .github/workflows/ci.yml
  - .gitignore
  - .golangci.yml
  - .planning/phases/06-api-and-semantics/06-01-PLAN.md
  - .planning/phases/06-api-and-semantics/06-01-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-02-PLAN.md
  - .planning/phases/06-api-and-semantics/06-02-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-03-PLAN.md
  - .planning/phases/06-api-and-semantics/06-03-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-04-PLAN.md
  - .planning/phases/06-api-and-semantics/06-04-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-05-PLAN.md
  - .planning/phases/06-api-and-semantics/06-05-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-06-PLAN.md
  - .planning/phases/06-api-and-semantics/06-06-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-07-PLAN.md
  - .planning/phases/06-api-and-semantics/06-07-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-08-PLAN.md
  - .planning/phases/06-api-and-semantics/06-08-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-09-PLAN.md
  - .planning/phases/06-api-and-semantics/06-09-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-10-PLAN.md
  - .planning/phases/06-api-and-semantics/06-10-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-11-PLAN.md
  - .planning/phases/06-api-and-semantics/06-11-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-12-PLAN.md
  - .planning/phases/06-api-and-semantics/06-12-SUMMARY.md
  - .planning/phases/06-api-and-semantics/06-13-PLAN.md
  - .planning/phases/06-api-and-semantics/06-13-SUMMARY.md
  - AGENTS.md
  - Makefile
  - api/supkvapi/v1/doc.go
  - api/supkvapi/v1/schema_test.go
  - api/supkvapi/v1/supkvapi.pb.go
  - api/supkvapi/v1/supkvapi.proto
  - api/supkvapi/v1/supkvapi_grpc.pb.go
  - buf.gen.api.yaml
  - buf.yaml
  - cmd/sup-kv/api_docs_test.go
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - docs/04-MULTI-NODE.md
  - docs/API.md
  - docs/EXAMPLES.md
  - docs/SPEC-DEVIATIONS.md
  - examples/client/README.md
  - examples/client/go.mod
  - examples/client/go.sum
  - examples/client/main.go
  - internal/app/internal/simnet/simnet.go
  - internal/app/persist/codec.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_replay_internal_test.go
  - internal/app/persist/engine_test.go
  - internal/app/persist/errors.go
  - internal/app/persist/facade.go
  - internal/app/persist/loader.go
  - internal/app/persist/loader_internal_test.go
  - internal/app/persist/scheduler.go
  - internal/app/persist/snapshot.go
  - internal/app/repl/bench_test.go
  - internal/app/repl/core.go
  - internal/app/repl/peertable_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/wait_internal_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/app/storage/view_test.go
  - internal/grpcapi/apiproto/supkvapi/v1/alias.go
  - internal/grpcapi/apiproto/supkvapi/v1/doc.go
  - internal/grpcapi/bench_test.go
  - internal/grpcapi/errors_internal_test.go
  - internal/grpcapi/errors_test.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/gate_test.go
  - internal/grpcapi/limits.go
  - internal/grpcapi/limits_internal_test.go
  - internal/grpcapi/limits_test.go
  - internal/grpcapi/list.go
  - internal/grpcapi/list_internal_test.go
  - internal/grpcapi/rollback_test.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait.go
  - internal/grpcapi/wait_integration_test.go
  - internal/grpcapi/wait_test.go
  - internal/grpcapi/watch.go
  - internal/grpcapi/watch_senderr_test.go
  - internal/grpcapi/watch_test.go
  - internal/persistfile/snapshot.go
  - internal/persistfile/snapshot_class_internal_test.go
  - internal/stand/persist_test.go
  - internal/stand/stand.go
  - internal/stand/stand_buffers_test.go
  - internal/stand/stand_delete_test.go
  - internal/stand/stand_metrics_test.go
  - internal/stand/stand_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/scenario_test.go
  - internal/udprepl/smoke_test.go
  - internal/udprepl/three_node_test.go
covered_digest: "v4:sha256:fdd2555c585bec02ff521aa3d77592328f5f6d6688d5307f4c740cc98a59f050"
behavior_unverified: 0
overrides_applied: 0
decision_coverage:
  honored: 7
  total: 7
  not_honored: []
---

# Phase 6: API and Semantics Verification Report

**Phase Goal:** The gRPC API tells the truth — `wait_for_replication` reports what actually happened, errors are total and path-free, Watch and health behave as documented, and the proto is public so consumers can build against it.
**Verified:** 2026-11-23T03:46:24Z
**Status:** passed
**Re-verification:** No — initial verification (no prior `*-VERIFICATION.md` existed)

## Goal Achievement

### Roadmap Success Criteria

| # | Criterion | Status | Evidence |
| - | --------- | ------ | -------- |
| 1 | `wait_for_replication` reports honestly: zero known peers returns `(false, 0, 0)`; an `Ack` from a node unknown at write time does not yield `true`; a displaced version answers within 20 ms instead of idling 500 ms | ✓ VERIFIED | Capture-before-write and the honest false: `server.go:206-238`, `:270-291`; `wait.go:72-98`. `TestWait_zeroPeers` (exactly one `total=0` release call, elapsed under the short bound), `TestWait_trackerNilIsSingleNode`, `TestWait_writeTimeSetPinned` (seam consulted once; captured `[1 2]` stands after the seam changes), `TestWait_deleteSymmetry`. Stranger-Ack inertness: `tracker.wants` filter in `applyResults`/`WaitReplicated` (`repl.go:777-868`), `TestWait_strangerAcksClampedToTotal`, `TestWaitReplicated_strangerAckDoesNotResolve`. Superseded: `ResultSuperseded` in the displacing handler call (`core.go:2243-2260`), `TestNotify_supersededResult`; real-socket `TestWaitReplicated_supersededFast` passes in 0.02 s (asserts the 250 ms CI bound; the 20 ms target is the structural same-call resolution, as the phase research resolved). |
| 2 | Errors and lifecycle are total: missing snapshot → `NotFound`, no dump → `FailedPrecondition`, no server paths in any message; Watch ends `Unavailable` on shutdown (not `OK`); health never returns to `SERVING` after shutdown begins; local changes announced on Watch only after dump confirmation — all stated in the API docs | ✓ VERIFIED | Ordered total mapper `server.go:493-542`; `TestMapError_totalTable` (17 rows incl. the fallback), `TestMapError_missingSnapshotIsNotFound`, `TestMapError_noDumpIsFailedPrecondition`, `TestMapError_missingSnapshotRealAdapter`, `TestMapError_unmappedIsConstantAndLogged`; all `status.Error*` sites audited (no path/errno/backslash; `server.go:142,156,198,204,265,431`, `list.go:164-166,253`, `gate.go:117`, `watch.go:45,53,537`). `TestRPC_statusMessagesPathFree` walks the six failing RPCs in both errno variants with a self-checking predicate. Watch shutdown: `watch.go:44-46,506-512`, `TestWatch_shutdownUnavailable`. One-way health: `gate.go:130-140`, `TestHealth_shutdownIsOneWay` (a late backstop provably opens the gate, health stays `NOT_SERVING`). Durability gate: `TestWatchHub_confirmGate`, `TestWatch_unconfirmedNotAnnounced`, `TestWatch_dumpFailedThenConfirmed`. Contract text: `docs/API.md` §§ Errors/Watch/Snapshots/Transport and health; `TestAPIDoc` anchors. |
| 3 | Snapshot load is a full rollback (14.1): keys absent from the snapshot are deleted, the result broadcasts to peers, and the rewritten `TestEngine_snapshotCreateLoadRestore` proves the new behavior | ✓ VERIFIED | `snapshot.go:146-191` (`replaySnapshot`/`replayKeys`), `:229-246` (`rollbackKey`), `:248-275` (`restoreChanges`); `TestEngine_snapshotCreateLoadRestore` (post-key deleted with a dominating fresh tombstone, `gone` keeps its version, re-tomb/resurrect semantics, recovery-from-scratch), `TestEngine_restoreReturnsReplayed`, `TestEngine_restoreExtraLiveKeyIsNotNoOp`, `TestEngine_restoreNoOpReturnsEmpty` (no extra dump), `TestServer_snapshotLoadRollback` (sorted union `snap/alpha, snap/mike(tombstone), snap/zulu` crosses the notify seam, survives a failed final dump with `Unavailable`, no-op notifies nothing). |
| 4 | `List` does not copy the world: pagination or streaming over the key→version view, with cost independent of value sizes | ✓ VERIFIED | `Store.LiveVersions` copies keys/versions only (`store.go:253-272`); legacy mode returns keys only and never calls `Lookup` (`list.go:122-154`); paged mode looks up only page entries with a serialized-byte budget (`list.go:188-232`). `TestServer_listLegacyAll` (one page_size-0 call returns all 512 live keys, zero records/token), `TestServer_listFirstPage`/`PageContinuation`, `TestServer_listLegacyCostIndependent`/`CostIndependent` (100 off-page max-size values do not grow the response), `TestServer_listByteBudget`, `TestListPage_maxKeySerializedBound`, `TestGRPC_worstCasePageFitsSendLimit`; source-shape gate over `listPage` (LiveVersions used, no `.Records()`). |
| 5 | The proto is public: proto and generated Go code live outside `internal/`; a Go client example builds in CI as a separate module; the last-read-value file pattern is documented as consumer-library territory (14.7) | ✓ VERIFIED | `api/supkvapi/v1/{supkvapi.proto,supkvapi.pb.go,supkvapi_grpc.pb.go,doc.go}`; no schema under `internal/` (only the two-file alias package); old generated import path gone from all non-frozen importers (grep count 0). Descriptor pins: `TestSchema_*`. Frozen red tests byte-identical (`519789bd…`, `573fed68…`, TSV `add45b810…`) and compile through the alias. `examples/client` is `example.com/sup-kv-client` (outside the root prefix) with `replace ../..`; `make client-example` green; dedicated CI job. `TestAPIDoc`/README pin the consumer-zone boundary (ADR 0007 / 14.7). |

### Observable Truths (plan must-haves)

**Score:** 79/79 truths verified, 0 present-behaviour-unverified.

| # | Plan | Truth | Status | Evidence |
| - | ---- | ----- | ------ | -------- |
| T-01.1 | 06-01 | Public proto and generated Go live at `api/supkvapi/v1` outside `internal/`; `go build ./...` green | ✓ VERIFIED | Files exist; no proto/generated code remains under `internal/grpcapi/apiproto`; `go build ./...` exit 0 |
| T-01.2 | 06-01 | Legacy generated-package path compiles as a thin deprecated alias; frozen red tests need no byte change | ✓ VERIFIED | `alias.go` type/var aliases; `doc.go` prose deprecation; `go test ./internal/grpcapi ./internal/stand` compile/run; frozen hashes unchanged |
| T-01.3 | 06-01 | Descriptor: `supkvapi.v1`, `SupKVService` with exactly Get/Set/Delete/List/Watch(server-streaming)/SnapshotCreate/SnapshotLoad; Record and existing messages keep field numbers | ✓ VERIFIED | `TestSchema_serviceDescriptor`, `TestSchema_recordFields`, `TestSchema_otherMessages` PASS; proto read |
| T-01.4 | 06-01 | `ListRequest` page_size=2/page_token=3; `ListResponse` next_page_token=2/records=3 plus keys=1 | ✓ VERIFIED | `TestSchema_listPageFields` PASS; proto lines 110-126 |
| T-01.5 | 06-01 | `buf lint`, `buf format --diff --exit-code`, `make buf-breaking` exit 0 with the one-file FILE_SAME_GO_PACKAGE waiver | ✓ VERIFIED | All three re-run exit 0; `buf.yaml` waiver scoped to `api/supkvapi/v1/supkvapi.proto` only |
| T-02.1 | 06-02 | Zero captured peers and a nil wait seam report `(false, 0, 0)` immediately | ✓ VERIFIED | `wait.go:72-88`; `TestWait_zeroPeers`, `TestWait_trackerNilIsSingleNode` |
| T-02.2 | 06-02 | Peer total is the set captured immediately before the engine write; same slice feeds notify and wait; a later seam answer does not move the report | ✓ VERIFIED | `server.go:212,234,238,273,286,290`; `TestWait_writeTimeSetPinned` (one seam call, captured peers notified, report unchanged after seam mutation) |
| T-02.3 | 06-02 | Zero-peer path still calls the seam with total 0 (release point) | ✓ VERIFIED | `wait.go:83`; `TestWait_zeroPeers` asserts exactly one `total==0` call |
| T-02.4 | 06-02 | Every changed notification call site traced: simulator Write/Delete/NotifyInstalled pass captured peers; `notifyReplayedWrite` passes nil | ✓ VERIFIED | `simnet.go:789,828,858`; `server.go:412-423`; `go build ./... && go vet ./...` exit 0 |
| T-02.5 | 06-02 | One atomic peer-id mirror feeds `WritePeers`; `KnownPeers` stays for metrics only | ✓ VERIFIED | `core.go` `PeerIDs`/`PeerGeneration`; `repl.go:348-358,916-923,1188-1191`; `main.go:724` (`WritePeers: adapter.KnownPeerIDs`), `:801` (`KnownPeers` metrics) |
| T-03.1 | 06-03 | `examples/client` is a separate module requiring the root via `replace ../..`, building only against the public package | ✓ VERIFIED | `examples/client/go.mod:3`, `replace` line; `go build`/`go vet` green |
| T-03.2 | 06-03 | Example demonstrates Set-with-wait, Get, a List page loop following the token, one bounded Watch event | ✓ VERIFIED | `examples/client/main.go` `setWithWait`, `get`, `listPages` (`NextPageToken` loop), `watchOne` (3 s context) |
| T-03.3 | 06-03 | `make client-example` builds, vets, tidies and proves no go.mod/go.sum diff | ✓ VERIFIED | Target re-run exit 0 |
| T-03.4 | 06-03 | CI has a dedicated client-example job running that target | ✓ VERIFIED | `.github/workflows/ci.yml:102-110` (`run: make client-example`) |
| T-03.5 | 06-03 | Root `go test`/`go list` never include the nested module | ✓ VERIFIED | `go list ./...` has 18 packages, zero `examples/client`; `make test` root suite unaffected |
| T-03.6 | 06-03 | No last-read-value cache; README states consumer-library territory | ✓ VERIFIED | README "last-read-value cache is consumer-library territory", ADR 0007/14.7; no cache code in `main.go` |
| T-04.1 | 06-04 | Every internal sentinel/class maps to a pinned code; unmapped error → `Internal` with exactly `internal error` | ✓ VERIFIED | `server.go:493-542`; `TestMapError_totalTable` (all rows + fallback), `TestMapError_unmappedIsConstantAndLogged` |
| T-04.2 | 06-04 | Missing snapshot → `NotFound` `snapshot not found`; no dump → `FailedPrecondition` | ✓ VERIFIED | `server.go:521-524`; `TestMapError_missingSnapshotIsNotFound`, `TestMapError_noDumpIsFailedPrecondition`, real-adapter sibling |
| T-04.3 | 06-04 | No status message carries an absolute/configured path or backslash; detail is logged | ✓ VERIFIED | All `status.Error*` sites audited; `TestRPC_statusMessagesPathFree` (six RPCs × two errno variants, predicate self-check) |
| T-04.4 | 06-04 | Pre-built status passes through; cancellation/deadline map to Canceled/DeadlineExceeded | ✓ VERIFIED | `server.go:527-536`; table rows in `TestMapError_totalTable` |
| T-04.5 | 06-04 | `CreateSnapshot` classifies ensureDir/read/save failures; `ErrNoDump` passes through | ✓ VERIFIED | `internal/persistfile/snapshot.go:48-75`; `TestCreateSnapshot_failureClasses` |
| T-04.6 | 06-04 | Fallback logs one static message with a snake_case detail attribute, never the detail in the response | ✓ VERIFIED | `server.go:537-541`; `TestMapError_unmappedIsConstantAndLogged` asserts exact response + captured log |
| T-05.1 | 06-05 | page_size unset/0 keeps the legacy all-live-keys, keys-only listing, ascending, no records/token | ✓ VERIFIED | `list.go:150-153`; `TestServer_listLegacyAll` (incl. 512-key single call) |
| T-05.2 | 06-05 | page_size > 0 returns a bounded page with values, aligned keys, non-empty token unless exhausted | ✓ VERIFIED | `list.go:188-224`; `TestServer_listFirstPage`, `TestServer_listPageContinuation` |
| T-05.3 | 06-05 | Token legal only with positive size; negative size and token-with-size-0 are `InvalidArgument` `invalid page size`; undecodable/over-domain token → `invalid page token` | ✓ VERIFIED (reviewed WR-01 supersession) | `list.go:161-173,250-256`; `TestServer_listBoundaries`, `TestServer_listToken`. The reviewed WR-01 fix deliberately widened the token domain from `storage.MaxKeyBytes` to `persist.MaxDumpLineBytes` so a pre-upgrade longer key stays walkable; `TestServer_listTokenAboveWriteCeiling` proves the 1 124-byte-key walk resumes (`1 499`-char token → `["z"]`). |
| T-05.4 | 06-05 | Empty token starts at the first key; token is an opaque position, empty means end; not a snapshot | ✓ VERIFIED | `list.go:122-148,220-222`; `TestServer_listPageContinuation`, `TestServer_listToken`, `TestListPage_tokenAboveWriteCeiling` |
| T-05.5 | 06-05 | page_size above max clamps to 256; page always ≥1 entry; budget may stop early with guaranteed progress | ✓ VERIFIED | `list.go:169-170,198-218`; `TestServer_listBoundaries`, `TestServer_listByteBudget` |
| T-05.6 | 06-05 | Byte budget is serialized: key twice, 16-byte version, value, per-record overhead; max-key page fits `listPageByteBudget+listEnvelopeOverheadBytes` | ✓ VERIFIED | `list.go:39-43,226-232`; `TestListPage_maxKeySerializedBound`; `TestGRPC_worstCasePageFitsSendLimit` |
| T-05.7 | 06-05 | `list.go` owns the four page/token constants; 06-07 consumes them and does not redefine | ✓ VERIFIED | `list.go:20-87`; `limits.go:45,101` references `listEnvelopeOverheadBytes`/`listPageByteBudget` |
| T-05.8 | 06-05 | `LiveVersions` returns live-only key→version, copies no values, uses the Deleted flag not empty data | ✓ VERIFIED | `store.go:253-272`; `TestStore_liveVersions` |
| T-05.9 | 06-05 | No whole-store record snapshot on any List path; legacy copies no value bytes; page cost independent of off-page values | ✓ VERIFIED | `LiveVersions` + `Lookup` only for page entries; `TestServer_listLegacyCostIndependent`, `TestServer_listCostIndependent`; source-shape gate (0 `.Records()` in `list.go`) |
| T-05.10 | 06-05 | Pagination semantics documented in the public proto comments | ✓ VERIFIED | `supkvapi.proto:99-126` (dual mode, ordering, position token, live view, bound, empty token) |
| T-06.1 | 06-06 | `OnChange` receives the just-incremented generation exactly once per real install, never for rejected/no-op | ✓ VERIFIED | `store.go:451-462`; `TestStore_changeTapGeneration`, `TestStore_onChange` |
| T-06.2 | 06-06 | `OnConfirmed` fires once per successful dump, only after `confirmWaiters`; watermark read immediately before `Records()` | ✓ VERIFIED | `engine.go:50-58`; `scheduler.go:232-247`; `TestWatch_dumpFailedThenConfirmed` |
| T-06.3 | 06-06 | Local change announced only after a covering dump; gen>confirmed waits; `Confirm(gen)` releases exactly once | ✓ VERIFIED | `watch.go:281-312,399-415`; `TestWatchHub_confirmGate`, `TestWatch_unconfirmedNotAnnounced` |
| T-06.4 | 06-06 | A second `Confirm` delivers nothing; a later lower confirmation never moves the watermark back | ✓ VERIFIED | Atomic-max `Confirm` (`watch.go:298-312`); `TestWatchHub_confirmGate` (repeat delivers nothing) |
| T-06.5 | 06-06 | Production wiring complete (`OnChange`→`Ingest`, `OnConfirmed`→`Confirm`); held-dump test: no event before release, exactly one after | ✓ VERIFIED | `main.go:580,587`; `TestWatch_unconfirmedNotAnnounced` (held dump, no event, Set blocked, one event after release) |
| T-07.1 | 06-07 | Server built with explicit keepalive and explicit max recv/send sizes | ✓ VERIFIED | `limits.go:20-35,114-127`; `main.go:868`; `TestLimitsFor` |
| T-07.2 | 06-07 | `MaxRecvBytes` from the effective configured value limit; `MaxSendBytes` = page budget + envelope | ✓ VERIFIED | `limits.go:95-106`; `main.go:735` passes `cfg.valueMaxBytes`; `TestLimitsFor` (default, 64 408, 0, negative) |
| T-07.3 | 06-07 | Budget is serialized-entry based; any page fits `MaxSendBytes`; legacy keys-only is key-count sized | ✓ VERIFIED | `list.go:226-232`; `TestGRPC_worstCasePageFitsSendLimit`; `TestServer_listLegacyAll` |
| T-07.4 | 06-07 | A request over `MaxRecvBytes` is `ResourceExhausted` at the transport, distinct from handler `InvalidArgument` | ✓ VERIFIED | `TestGRPC_maxRecv`; `TestGRPC_maxValueRoundTrip` at the 64 408-byte limit |
| T-07.5 | 06-07 | Worst-case List page from max-length keys/max values marshals within `MaxSendBytes` | ✓ VERIFIED | `TestGRPC_worstCasePageFitsSendLimit` (257 records, page stops on budget, marshaled ≤ ceiling) |
| T-07.6 | 06-07 | `Server.Shutdown` uses the standard health server's one-way `Shutdown`; late backstop cannot return SERVING | ✓ VERIFIED | `gate.go:130-140`; `TestHealth_shutdownIsOneWay` (backstop opens the gate, health stays `NOT_SERVING`, double Shutdown safe) |
| T-07.7 | 06-07 | Binary calls `api.Shutdown()` before `apiCancel`; never `SetServingStatus` directly; never `Resume` | ✓ VERIFIED | `main.go:939-948`; grep finds no `node.health.SetServingStatus`/`health.Resume` |
| T-08.1 | 06-08 | Full rollback: absent live key gets a fresh tombstone that joins the returned list | ✓ VERIFIED | `snapshot.go:161-191,233-246`; `TestEngine_snapshotCreateLoadRestore`, `TestEngine_restoreReturnsReplayed` |
| T-08.2 | 06-08 | Rollback result broadcasts through the existing `SnapshotLoad` notify path, including when the final dump fails | ✓ VERIFIED | `server.go:356-375`; `TestServer_snapshotLoadRollback` (normal + `failSave` subtests) |
| T-08.3 | 06-08 | An already-tombstoned absent key is not deleted again; no version burned; no record produced | ✓ VERIFIED | `snapshot.go:198-208,259-275`; `TestEngine_snapshotCreateLoadRestore` `gone` assertions |
| T-08.4 | 06-08 | An extra live key absent from the snapshot is a change; an equal-state load is a no-op with empty list and no dump | ✓ VERIFIED | `restoreChanges`; `TestEngine_restoreExtraLiveKeyIsNotNoOp`, `TestEngine_restoreNoOpReturnsEmpty` |
| T-08.5 | 06-08 | Replay and rollback are one deterministic sorted union; a mid-pass error returns the partial installed list with the error | ✓ VERIFIED | `replayKeys` sorted union; `replaySnapshot` returns `restored, err`; code read |
| T-08.6 | 06-08 | `Restore`/replay docs state full-rollback semantics, not additive merge | ✓ VERIFIED | `snapshot.go:100-109,146-160,248-258` |
| T-09.1 | 06-09 | `docs/API.md` exists with the phase-owned contract sections | ✓ VERIFIED | `docs/API.md` (326 lines): Public proto, wait, List, Errors, Watch, Snapshots, Transport and health, Consumer library territory, Phase 9 split |
| T-09.2 | 06-09 | Wait section states write-time set, zero-peer false, Superseded, caller-bounded, unknown-Ack exclusion | ✓ VERIFIED | `docs/API.md:52-92`; anchors in `TestAPIDoc` |
| T-09.3 | 06-09 | List section states both modes, ordering, position token, 256 max, 6 MiB budget, live-only, empty-token end | ✓ VERIFIED | `docs/API.md:94-140`; anchors |
| T-09.4 | 06-09 | Error contract names every mapper code + List constant message, detail logged, no server path | ✓ VERIFIED (see Info finding) | `docs/API.md:142-194`: all 15 mapper rows are in the table; `invalid page size`/`invalid page token` are stated in the List section and pinned by `TestAPIDoc`; logging/path-free rules stated. The exhaustive per-RPC table is Phase 9/DOCS-05. The Errors intro's "covers every error" sentence is broader than the table — recorded as an Info anti-pattern below, not a goal gap. |
| T-09.5 | 06-09 | Watch section states after-dump, exactly once, deletions, version order, asymmetry, Unavailable, resubscribe | ✓ VERIFIED | `docs/API.md:196-236`; anchors |
| T-09.6 | 06-09 | Snapshots section states full rollback, fresh versions, no version burn, broadcast, non-linearizable caveat | ✓ VERIFIED | `docs/API.md:238-279` (plus the CR-01 partial-snapshot refusal) |
| T-09.7 | 06-09 | Transport section states keepalive, ceilings, one-way NOT_SERVING; consumer zone states ADR 0007/14.7 | ✓ VERIFIED | `docs/API.md:281-316`; values match `LimitsFor` (187 472 / 190 440 / 6 MiB + 125 008) |
| T-09.8 | 06-09 | A doc-anchor test pins the sections, so deletion/heading drift fails the normal test gate | ✓ VERIFIED | `cmd/sup-kv/api_docs_test.go` (anchor table read from `../../docs/API.md`); `TestAPIDoc` PASS |
| T-10.1 | 06-10 | No non-historical doc names the legacy internal generated-package path; `docs/reviews/**` untouched | ✓ VERIFIED | grep over `docs`+`AGENTS.md` excluding `docs/reviews` = 0; `git diff` on `docs/reviews` empty in phase diff |
| T-10.2 | 06-10 | `docs/EXAMPLES.md` shows the paged List shape, the legacy unset/0 mode, and the zero-peer wait; `TestExamplesDoc` passes | ✓ VERIFIED | `docs/EXAMPLES.md:216-276`; `TestExamplesDoc` PASS (4.14 s) |
| T-10.3 | 06-10 | `docs/EXAMPLES.md` states SnapshotLoad is a full rollback that replicates | ✓ VERIFIED | "full rollback", "absent from the snapshot" present; paragraph names fresh versions/tombstones/LWW |
| T-10.4 | 06-10 | `docs/04-MULTI-NODE.md` describes the `WritePeers` set and states IN-05 is closed | ✓ VERIFIED | `04-MULTI-NODE.md:96-118`; no "Known limitation IN-05" text remains |
| T-10.5 | 06-10 | `docs/SPEC-DEVIATIONS.md` S-PERS-12 `implemented` with rollback code/test links; API-proto links public; S-PERS-13 untouched | ✓ VERIFIED | Row 170 cites `Restore`/`replaySnapshot`/`replayKeys`/`rollbackKey`/`restoreChanges` + tests; rows 180/249-251 use `api/supkvapi/v1`; S-PERS-13 still `fixed-in-v1.1(PROTO-08)` |
| T-10.6 | 06-10 | `AGENTS.md` Layout lists `api/supkvapi/v1` and `examples/client`; deviation 1 names the third depguard rule; deviations 2-3 and Go policy unchanged | ✓ VERIFIED | `AGENTS.md:18-19,37-48`; `sup-kv-public-api-no-internal` present; "These rules live in `.golangci.yml`" |
| T-11.1 | 06-11 | Tracker counts only captured write-time peers; a post-write Ack never increases/flips | ✓ VERIFIED | `repl.go:167-193,777-796,830-868`; `TestWaitReplicated_trackerCapturedPeersOnly`, `TestWaitReplicated_strangerAckDoesNotResolve`, `TestWait_strangerAcksClampedToTotal` |
| T-11.2 | 06-11 | Captured-but-gone peer emits `WaitSetEmpty`; adapter reports `(false, 0)` immediately | ✓ VERIFIED | `core.go:2268-2280`; `TestWaitReplicated_waitSetEmptyImmediate` (0.31 s against a 2 s timeout) |
| T-11.3 | 06-11 | Discovery flush still broadcasts/re-arms but never inserts into a wait-set or ledger (IN-05) | ✓ VERIFIED | `core.go:1413-1426`; `TestPeerTable_discoveryFlushKeepsPromise` (wait-set/ledger unchanged, record send queued, ladder reset) |
| T-11.4 | 06-11 | A displaced version resolves via `ResultSuperseded` in the same handler call (structural 20 ms proof); real socket under 250 ms | ✓ VERIFIED | `TestNotify_supersededResult`; `TestWaitReplicated_supersededFast` (0.02 s, asserts `< 250 ms`) |
| T-11.5 | 06-11 | Zero-peer path releases the tracker: one `total=0` seam call and an empty adapter tracker map | ✓ VERIFIED | `repl.go:830-835`; `TestWaitReplicated_zeroPeersReleases`, `TestWait_zeroPeers` |
| T-11.6 | 06-11 | Two-node real-adapter wait with post-write discovery returns true from the captured peer alone | ✓ VERIFIED | `TestWait_integrationCapturedPeerOnly` PASS (report `1/1`) |
| T-12.1 | 06-12 | A startup load fires no Watch event; the nil-tap wrapper is safe for `storage.Config{}` callers | ✓ VERIFIED | `loader.go:65-88`; `TestOpen_loadDoesNotFireTap` (zero taps, all keys loaded), `TestOpen_nilTapStaysWritable` |
| T-12.2 | 06-12 | A dump-failed write is announced exactly once when a later dump succeeds; never before | ✓ VERIFIED | `TestWatch_dumpFailedThenConfirmed`, `TestWatch_confirmedExactlyOnce` |
| T-12.3 | 06-12 | Deletions are events; live events follow version order; the snapshot stays current memory | ✓ VERIFIED | `TestWatch_deletionsAndOrder`; `handleSubscribe` snapshot at `watch.go:359-379` |
| T-12.4 | 06-12 | Shutdown ends every stream with `Unavailable` `watch hub is shutting down`, never OK | ✓ VERIFIED | `watch.go:44-46,506-512`; `TestWatch_shutdownUnavailable` |
| T-12.5 | 06-12 | Pending is bounded by `watchIngress`; overflow disconnects/clears; resubscribe re-snapshots | ✓ VERIFIED | `watch.go:27-36,385-427`; `TestWatchHub_pendingBounded` |
| T-13.1 | 06-13 | Frozen red evidence hashes equal the planning-time literals | ✓ VERIFIED | `red_gate=519789bd…`, `stand_red=573fed68…`, `tsv=add45b810…` recomputed on the current tree |
| T-13.2 | 06-13 | Red runner exits 0, 7 rows, 0 errors, mutation stage passes | ✓ VERIFIED | `bash scripts/red-runner` exit 0: "7 expectation row(s) checked, 0 error(s)"; both mutants killed; evidence `20261123T033923Z` |
| T-13.3 | 06-13 | `make lint` and `make test` green on the final tree | ✓ VERIFIED | Both re-run exit 0; `grpcapi` 98.4%, `persist` 94.7%, `stand` 83.9% coverage |
| T-13.4 | 06-13 | `make buf-breaking` green on the final tree | ✓ VERIFIED | Exit 0 against `refs/remotes/origin/master` |

### Deferred Items

No must-have gap was deferred. Two observations are informational only:

| Item | Owner | Note |
| ---- | ----- | ---- |
| `docs/API.md` Errors intro says "The table below covers every error the RPC handlers produce", but the two List validation messages (`invalid page size`, `invalid page token`) are documented in the List section rather than as table rows | Phase 9 / DOCS-05 (full per-RPC failure table) | Both messages are present, client-visible and anchor-pinned; the code is total. See the Info anti-pattern. |
| Pre-existing cross-package UDP port-contention / lifecycle-readiness flakes recorded during execution | Phase 8 flake owner (per `deferred-items.md`) | Did not fire in this verification; the full suite and red runner were green. |

## Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `api/supkvapi/v1/supkvapi.proto` | Public schema, 7 RPCs, page fields | ✓ VERIFIED | 159 lines; `go_package` public; page fields present; comments document both List modes |
| `api/supkvapi/v1/{supkvapi.pb.go,supkvapi_grpc.pb.go}` | Generated public Go code | ✓ VERIFIED | Generated through the pinned toolchain; no `internal/` import |
| `api/supkvapi/v1/doc.go`, `schema_test.go` | Package contract + descriptor pin | ✓ VERIFIED | `TestSchema_*` 4/4 PASS |
| `internal/grpcapi/apiproto/supkvapi/v1/{alias.go,doc.go}` | Thin deprecated alias; frozen tests compile | ✓ VERIFIED | 25 re-exports; both frozen packages compile/run |
| `internal/grpcapi/wait.go` | Write-time capture, honest zero-peer/nil-seam, timeout bounds | ✓ VERIFIED | `waitForReplication`, `writePeers`, `waitTimeout` |
| `internal/grpcapi/server.go` | Total path-free mapper; gated handlers; notify paths | ✓ VERIFIED | `mapError` 15 rows + pass-through + constant fallback |
| `internal/grpcapi/list.go` | Two-mode List, token codec, serialized budget | ✓ VERIFIED | 257 lines; constants owned here |
| `internal/grpcapi/limits.go` | TransportLimits, LimitsFor, ServerOptions | ✓ VERIFIED | Effective-limit derivation, keepalive triple |
| `internal/grpcapi/watch.go` | Durable gate, bounded pending, Unavailable shutdown | ✓ VERIFIED | pending/confirmed/confirmCh; content-aware suppression |
| `internal/grpcapi/gate.go` | `WritePeers`, `ServerOptions`, one-way `Shutdown` | ✓ VERIFIED | `s.healthSrv.Shutdown()` |
| `internal/app/storage/store.go` | Generation-tagged OnChange tap; `LiveVersions` | ✓ VERIFIED | `install` is the single convergence point |
| `internal/app/persist/{scheduler.go,engine.go,loader.go}` | Watermark publication; tap suppression | ✓ VERIFIED | `OnConfirmed` after `confirmWaiters`; startup guard |
| `internal/app/persist/snapshot.go` | Full rollback + refusal-before-change | ✓ VERIFIED | `Restore` checks `DecodeStats.Total()` before any store touch |
| `internal/app/repl/{core.go,wait_internal_test.go}` | Captured-set promise, no-growth flush | ✓ VERIFIED | `trackLocalWrite` intersection; flush split |
| `internal/udprepl/repl.go` | Captured-set tracker, `KnownPeerIDs` mirror | ✓ VERIFIED | `wants` + atomic pointer |
| `internal/persistfile/snapshot.go` | Classified `CreateSnapshot` | ✓ VERIFIED | Three failure steps classified |
| `cmd/sup-kv/main.go` | Production wiring: taps, peers, limits, shutdown order | ✓ VERIFIED | Lines 580, 587, 724-735, 939-948 |
| `docs/API.md`, `cmd/sup-kv/api_docs_test.go` | Phase-owned contract + drift guard | ✓ VERIFIED | 326 lines; anchors PASS |
| `docs/{EXAMPLES.md,04-MULTI-NODE.md,SPEC-DEVIATIONS.md}`, `AGENTS.md` | Documentation ripple | ✓ VERIFIED | Claims checked literally |
| `examples/client/{go.mod,go.sum,main.go,README.md}` | Out-of-prefix consumer module | ✓ VERIFIED | `module example.com/sup-kv-client`; build/vet/tidy green |
| `Makefile`, `.github/workflows/ci.yml` | `client-example` target + job | ✓ VERIFIED | Job invokes exactly the target |
| `buf.yaml`, `buf.gen.api.yaml`, `.golangci.yml` | Buf module move + public-package boundary | ✓ VERIFIED | One-file waiver; `sup-kv-public-api-no-internal` |
| `scripts/red-runner`, `scripts/red-expectations.tsv` | Frozen gate | ✓ VERIFIED | Runner executed; TSV unchanged |

The `gsd_run query verify.artifacts` verb reports `total: 0` for these plans because their `must_haves.artifacts` entries are plain path strings rather than the object shape the heuristic expects; the artifact lists were therefore checked directly above.

## Key Link Verification

All plan-declared links were checked by reading the source/import and by the green build, test and lint gates. The `verify.key-links` heuristic reports `Target not referenced in source` for same-package calls and build/config relationships (the target filename is not textually named in the source); that is a matcher limitation, not a broken link. Cross-package/doc links it could match are green (`06-04`, `06-09`, `06-10`, `06-13`).

| # | From | To | Via (abridged) | Status | Details |
| - | ---- | -- | -------------- | ------ | ------- |
| L-01.1 | `api/supkvapi/v1/supkvapi.proto` | `buf.yaml` | second buf module rooted at `api` | WIRED | `buf.yaml` module `api`; `make buf-lint` green |
| L-01.2 | proto | generated `.pb.go` | `make proto` (buf.gen.api.yaml, source_relative) | WIRED | Files carry `source: supkvapi/v1/supkvapi.proto` |
| L-01.3 | alias package | public generated code | type/var aliases | WIRED | `alias.go` imports the public package |
| L-01.4 | `cmd/sup-kv/main.go` | public generated service | `apiproto` alias import; `RegisterSupKVServiceServer` | WIRED | `main.go:868`; build green |
| L-02.1 | `server.go` | `wait.go` | Set/Delete capture and call `waitForReplication` | WIRED | `server.go:212-238,273-290` |
| L-02.2 | `server.go` | `udprepl/repl.go` | `LocalNotify` gains peers; adapter stores them | WIRED | `LocalNotify` type; `NotifyLocalWrite` signature; main wiring |
| L-02.3 | `server.go` | `udprepl/repl.go` | replay notify passes nil; installed notify passes `PeerIDs()` | WIRED | `server.go:412-423`; `simnet.go:789,828,858` |
| L-02.4 | `core.go` | `udprepl/repl.go` | `PeerIDs`/`PeerGeneration` published via `mirrorStats` | WIRED | `repl.go:1188-1191` |
| L-03.1 | `examples/client/go.mod` | root `go.mod` | require + `replace ../..` | WIRED | `go.mod:3-12` |
| L-03.2 | `examples/client/main.go` | public generated client | `supkvapi.NewSupKVServiceClient` | WIRED | Build green |
| L-03.3 | `Makefile` | `ci.yml` | `run: make client-example` | WIRED | `ci.yml:102-110` |
| L-04.1 | `server.go` | `port/dump.go` | `port.FailureClassOf` after `ErrDumpFailed` | WIRED | `server.go:503-506` |
| L-04.2 | `server.go` | `persistfile` | `ErrNoDump`/`ErrInvalidSnapshotName`/fs rows | WIRED | `server.go:519-526` |
| L-04.3 | `persistfile/snapshot.go` | `saveerror.go` | `classifySaveError` on ensure/read/save | WIRED | `snapshot.go:56-71` |
| L-05.1 | `server.go` | `list.go` | `s.listPage(req)` after the gate | WIRED | `server.go:312-317` |
| L-05.2 | `list.go` | `persist/facade.go` | `Engine.LiveVersions` | WIRED | `list.go:137`; facade delegate |
| L-05.3 | `facade.go` | `storage/store.go` | `Store.LiveVersions` under RLock | WIRED | `store.go:261-272` |
| L-05.4 | `list.go` | `server.go` | `recordToProto` for page entries | WIRED | `list.go:218` |
| L-05.5 | `list.go` | `limits.go` | page budget/envelope consumed by `MaxSendBytes` | WIRED | `limits.go:101` |
| L-05.6 | `stand_test.go` | `list.go` | nightly poll follows `NextPageToken` | WIRED | `stand_test.go:552-562` |
| L-06.1 | `store.go` | `watch.go` | `OnChange` → `hub.Ingest` | WIRED | `main.go:580`; harness wiring |
| L-06.2 | `scheduler.go` | `watch.go` | `OnConfirmed` → `hub.Confirm` after `confirmWaiters` | WIRED | `main.go:587`; `scheduler.go:244-247` |
| L-06.3 | `watch.go` | `watch.go` | pending release in arrival order | WIRED | `watch.go:399-415` |
| L-07.1 | `limits.go` | `storage/store.go` | `MaxKeyBytes`/effective limit constants | WIRED | `limits.go:100` |
| L-07.2 | `limits.go` | `list.go` | `listPageByteBudget` + envelope → `MaxSendBytes` | WIRED | `limits.go:101` |
| L-07.3 | `main.go` | `limits.go` | `grpc.NewServer(limits.ServerOptions()...)` from `LimitsFor(cfg.valueMaxBytes)` | WIRED | `main.go:735,868` |
| L-07.4 | `main.go` | `gate.go` | `node.api.Shutdown()` before `apiCancel` | WIRED | `main.go:939,948` |
| L-08.1 | `snapshot.go` | `engine_test.go` | rewritten rollback test | WIRED | `TestEngine_snapshotCreateLoadRestore` |
| L-08.2 | `snapshot.go` | `grpcapi/rollback_test.go` | rollback tombstones cross `SnapshotLoad` notify | WIRED | `TestServer_snapshotLoadRollback` |
| L-08.3 | `snapshot.go` | `store.go` | `Delete` absent live key; ErrNotFound skip | WIRED | `snapshot.go:233-246` |
| L-09.1 | `docs/API.md` | public proto | import path and RPC names | WIRED | `docs/API.md:19-50`; `TestAPIDoc` |
| L-09.2 | `docs/API.md` | `server.go` | error rows match mapper constants | WIRED | Read-through of every row; `TestMapError_totalTable` |
| L-09.3 | `api_docs_test.go` | `docs/API.md` | anchors read from the repo file | WIRED | `apiDocPath = "../../docs/API.md"` |
| L-10.1 | `SPEC-DEVIATIONS.md` | public proto | five evidence links | WIRED | `grep` shows `api/supkvapi/v1`; line ranges resolve |
| L-10.2 | `SPEC-DEVIATIONS.md` | `persist/snapshot.go` | S-PERS-12 implemented row | WIRED | Cited functions present |
| L-10.3 | `04-MULTI-NODE.md` | `gate.go` | `WritePeers` seam described | WIRED | `gate.go:83` |
| L-10.4 | `AGENTS.md` | `examples/client/go.mod` | Layout entry | WIRED | `AGENTS.md:43-48` |
| L-11.1 | `repl.go` | `repl.go` | `ensureTracker` stores total+wants; `applyResults` filters | WIRED | `repl.go:186-193,777-796` |
| L-11.2 | `core.go` | `core.go` | seeded wait-set; flush does not grow it | WIRED | `core.go:1413-1426,2263-2293` |
| L-11.3 | `core.go` | `repl.go` | Superseded/WaitSetEmpty resolve the tracker | WIRED | `repl.go:789-793` |
| L-12.1 | `loader.go` | `store.go` | nil-safe synchronous loading guard | WIRED | `loader.go:75-88` |
| L-12.2 | `watch.go` | `watch.go` | shutdown writes Unavailable; overflow clears pending | WIRED | `watch.go:385-390,506-512` |
| L-13.1 | `scripts/red-runner` | `scripts/red-expectations.tsv` | parses the unchanged 7-row table | WIRED | Runner exit 0, 7/7 |
| L-13.2 | pinned blob hash | frozen tests | `git hash-object` comparison | WIRED | Three literals matched |

## Data-Flow Trace (Level 4)

This phase's rendered values are gRPC responses and streams. Every traced production value ends in a real store/file/clock; no static or mock source reaches a response.

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `Get` response | `rec` | `engine.Get` → store map under RLock | Yes | ✓ FLOWING |
| `Set`/`Delete` version + report | `ver`, `peers`, tracker | clock `Tick`, captured `KnownPeerIDs`, adapter tracker | Yes — real UDP Ack tests | ✓ FLOWING |
| Legacy `List` | `keys` | `engine.LiveVersions` (live key→version) | Yes — 512-key test | ✓ FLOWING |
| Paged `List` | `records` | `engine.Lookup` for page entries only | Yes — values asserted | ✓ FLOWING |
| `Watch` snapshot | `snapshot` | `engine.Records` on the hub goroutine | Yes — seeded records | ✓ FLOWING |
| `Watch` live events | `pending` → `events` | store tap generation → dump watermark | Yes — held-dump test | ✓ FLOWING |
| `SnapshotLoad` notify | `records` | `Restore` report (replay + rollback) | Yes — engine + RPC tests | ✓ FLOWING |
| Health | status | live gate/`health.Server` | Yes — bufconn health checks | ✓ FLOWING |
| Transport limits | `TransportLimits` | `cfg.valueMaxBytes` + page/format constants | Yes — `LimitsFor` table | ✓ FLOWING |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Whole-tree build | `go build ./...` | exit 0 | ✓ PASS |
| Vet (host and darwin) | `go vet ./...`; `GOOS=darwin GOARCH={amd64,arm64} go build ./... && go vet ./...` | all exit 0 | ✓ PASS |
| Strict lint | `make lint` | config verify + `golangci-lint run ./...` exit 0 | ✓ PASS |
| Full race suite | `make test` (`go test -race -cover ./...`) | exit 0; app/repl 175.1 s; every package `ok`; no flake fired | ✓ PASS |
| Proto lint/format/breaking | `make buf-lint`, `go tool buf format --diff --exit-code`, `make buf-breaking` | all exit 0 (breaking against fetched `origin/master`) | ✓ PASS |
| Client example | `make client-example` | exit 0 (build + vet + tidy no-diff) | ✓ PASS |
| Root graph excludes nested module | `go list ./...` | 18 packages, zero `examples/client` | ✓ PASS |
| Red gate | `bash scripts/red-runner` | exit 0; "7 expectation row(s) checked, 0 error(s)"; both mutants killed; evidence `20261123T033923Z` | ✓ PASS |
| Frozen content pin | `git hash-object` on the two red tests + TSV | `519789bd…`, `573fed68…`, `add45b810…` — exact match | ✓ PASS |
| Focused phase packages | `go test -race -count=1 ./internal/grpcapi ./internal/app/persist ./internal/app/storage ./internal/udprepl ./api/supkvapi/v1 ./cmd/sup-kv` | all `ok` | ✓ PASS |
| Wait matrix | grpcapi `TestWait_*` (13 tests) | 13/13 PASS | ✓ PASS |
| Watch matrix | grpcapi `TestWatch_*`/`TestWatchHub_*` (9 tests) | 9/9 PASS | ✓ PASS |
| List matrix | grpcapi `TestServer_list*` + internal bound tests (15 tests) | 15/15 PASS | ✓ PASS |
| Transport/health matrix | `TestLimitsFor`, `TestGRPC_*`, `TestHealth_*` | 8/8 PASS | ✓ PASS |
| Mapper/path matrix | `TestMapError_*`, `TestRPC_statusMessagesPathFree` | 6/6 PASS | ✓ PASS |
| Rollback matrix | persist `TestEngine_restore*`/`TestOpen_*`, grpcapi `TestServer_snapshotLoadRollback` | all PASS | ✓ PASS |
| Adapter/core wait matrix | `TestWaitReplicated_*` (12), repl core 4 named tests | all PASS | ✓ PASS |
| Doc anchors/executable docs | `TestAPIDoc`, `TestExamplesDoc` | both PASS | ✓ PASS |
| RED evidence records | `check tdd-red-evidence` over 17 JSON records | all `RED_EVIDENCE_OK` (`target_test_failed`) | ✓ PASS |

## Probe Execution

No `scripts/*/tests/probe-*.sh` exists and no phase plan declares a probe script. The phase's executable acceptance runner is the frozen red gate, re-executed on the current tree (7/7 rows, both mutants killed, evidence `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261123T033923Z/`). The earlier committed evidence directories were left untouched.

## Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
| --------- | ---------- | ------ | ------- | -------- | --------------- | ------- |
| `internal/grpcapi/red_gate_test.go` (frozen) | APIS-01…07 regression | red-gated; ran | 0 in the runner | No | Behavioral | PASS |
| `internal/stand/stand_red_test.go` (frozen) | APIS-01…07 regression | red-gated; ran | 0 in the runner | No | Behavioral (real process/sockets) | PASS |
| `internal/grpcapi/wait_test.go` | APIS-01 | all | 0 | No | Value + behavioral | PASS |
| `internal/grpcapi/wait_integration_test.go` | APIS-01 | all | 0 | No | Behavioral (real UDP pair) | PASS |
| `internal/udprepl/repl_test.go`, `repl_internal_test.go` | APIS-01 | all | 0 | No | Behavioral (real sockets + tracker state) | PASS |
| `internal/app/repl/retransmit_test.go`, `wait_internal_test.go` | APIS-01 | all | 0 | No | Behavioral (deterministic Outbox) | PASS |
| `internal/grpcapi/server_test.go` (list tests) | APIS-02 | all | 0 | No | Value (exact keys/tokens/sizes) | PASS |
| `internal/grpcapi/list_internal_test.go` | APIS-02 | all | 0 | No | Value (serialized bound, race seam) | PASS |
| `internal/grpcapi/errors_test.go`, `errors_internal_test.go` | APIS-03 | all | 0 | No | Value (exact code/message) + behavioral matrix | PASS |
| `internal/persistfile/snapshot_class_internal_test.go` | APIS-03 | all | 0 | No | Value (`FailureClassOf`) | PASS |
| `internal/grpcapi/watch_test.go` | APIS-04 | all | 0 | No | Behavioral (confirmation gate, shutdown, order) | PASS |
| `internal/app/persist/engine_test.go`, `loader_internal_test.go` | APIS-04/07 | all | 0 | No | Behavioral (startup silence, rollback) | PASS |
| `internal/grpcapi/limits_test.go`, `limits_internal_test.go` | APIS-05 | all | 0 | No | Value (exact ceilings) + behavioral | PASS |
| `internal/grpcapi/gate_test.go` | APIS-05 | all | 0 | No | Behavioral (one-way health) | PASS |
| `internal/grpcapi/rollback_test.go` | APIS-07 | all | 0 | No | Behavioral (notify seam, failed dump) | PASS |
| `api/supkvapi/v1/schema_test.go` | APIS-06/02 | 4 | 0 | No | Value (descriptor) | PASS |
| `cmd/sup-kv/api_docs_test.go`, `examples_test.go` | APIS-01…07 | all | 0 | No | Presence anchors + executable doc | PASS |

**Disabled tests on requirements:** 0 (the env-gated frozen red tests are the designed mechanism and ran via the red runner; environment-conditional skips in `smoke_test.go`/`main_test.go`/`persist_test.go` are unrelated to APIS). **Circular patterns:** 0 — no test generates expected values by running the system under test; fixtures carry literal records/checksums and expected outputs are asserted constants. **Assertion strength:** the critical tests use value/behavioral assertions (exact code/message, exact key lists, exact count/version, real-socket outcome), not existence-only. **Insufficient assertions:** 0.

## Requirements Coverage

| Requirement | Source Plan(s) | Status | Evidence |
| ----------- | -------------- | ------ | -------- |
| APIS-01 | 02, 11, 09, 10, 13 | ✓ SATISFIED | Write-time capture, honest `(false,0,0)`, captured-set tracker, unknown-Ack inertness, no-growth flush, `Superseded` structural + real-socket proof, zero-peer release, two-node integration |
| APIS-02 | 01, 05, 09, 10, 13 | ✓ SATISFIED | Dual-mode List over `LiveVersions`, page fields on the wire, boundaries/token/prefix/race/empty tests, serialized budget and max-key page fit, cost independence, proto comments |
| APIS-03 | 04, 09, 13 | ✓ SATISFIED | Total ordered mapper, `NotFound`/`FailedPrecondition` rows, constant fallback with logged detail, classified snapshot create, six-RPC path-free matrix |
| APIS-04 | 06, 12, 09, 13 | ✓ SATISFIED | Generation-tagged tap, confirmed watermark, pending gate, startup suppression, exactly-once, deletions/order, bounded pending, `Unavailable` shutdown, contract in `docs/API.md` |
| APIS-05 | 07, 09, 13 | ✓ SATISFIED | Effective-limit-derived ceilings, page-budget send ceiling, explicit keepalive, `ResourceExhausted` vs `InvalidArgument`, worst-case page fit, one-way health, binary shutdown order |
| APIS-06 | 01, 03, 09, 10, 13 | ✓ SATISFIED | Public proto + generated code, alias keeps frozen tests compiling, out-of-prefix client module built in CI, replication proto stays internal, consumer-zone docs, depguard boundary |
| APIS-07 | 08, 09, 10, 13 | ✓ SATISFIED | Full rollback with fresh tombstones, already-tombstoned no-op, broadcast incl. failed dump, CR-01 refusal of partially readable snapshots before any change, rewritten engine test, docs |

No ORPHANED requirements: `.planning/REQUIREMENTS.md` maps exactly APIS-01…07 to Phase 6, all seven are marked `[x]` and `Complete`, and every ID is declared by at least one phase plan (06-13 declares all seven).

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX` debt markers in phase-modified files. The only `FIXME` match is `.golangci.yml:162`, the `todo-comments` linter's keyword documentation — not a debt marker | — | None |
| — | — | No `TODO`/`HACK`/`PLACEHOLDER` or "coming soon"/"not yet implemented" markers in production files | — | None |
| — | — | No empty implementations or placeholder returns on any List/Watch/wait/rollback path | — | None |
| — | — | No hardcoded empty props/static data on a production path; every response is store- or tracker-derived | — | None |
| `docs/API.md` | 144-147 | The Errors section says "The table below covers every error the RPC handlers produce", but the two List validation messages are documented in the List section, not as table rows | ℹ️ Info | Both messages are present and client-visible in the same document and anchor-pinned; the code mapping is total. The exhaustive per-RPC table is Phase 9/DOCS-05. Not a goal gap. |
| `docs/API.md` | 294 | The `MaxSendBytes` row states the server can send a 6 MiB + 125 008-byte page, but does not note that a consumer requesting a maximum page must raise its own gRPC receive limit (the gRPC default is 4 MiB, below this server ceiling) | ℹ️ Info | The server ceiling is honest and pinned; the plan's own harness sets an 8 MiB test client receive limit for this reason (`server_test.go:55-60`). A consumer following only `docs/API.md` could see the client reject a legitimate large page with `ResourceExhausted`; a one-line client-side note would close it. Not a goal gap. |
| — | — | Nine new `//nolint` directives in the phase diff, all naming a linter and a written reason; no unnamed or blanket suppression | — | None |

## Decision Coverage

All 7 trackable CONTEXT.md decisions (D-01…D-07) are honored by shipped artifacts (`check.decision-coverage-verify`: total 7, honored 7, not_honored []).

## Human Verification Required

None. This is an infrastructure/backend phase (gRPC API, persistence, docs) with no UI or user-facing interaction; every acceptance criterion is exercised programmatically, including the behaviour-dependent truths (wait resolution, durability-gated Watch, one-way health, startup tap suppression, rollback broadcast, partial-snapshot refusal) by passing deterministic or real-socket tests. There are 0 present-behaviour-unverified truths and no abstained non-inferable truth.

## Gaps Summary

No gaps. All seven requirements are shipped and wired end to end, all five roadmap success criteria are met, and the four review-fix areas were independently re-verified:

- **CR-01** — `Restore` now reads `DecodeStats` and refuses a partially decoded snapshot with `ErrSnapshotIncomplete` → `DataLoss` before any store mutation, replay, delete, flush or notify (`persist/snapshot.go:110-127`); `TestEngine_restorePartialSnapshotRefused` and `TestServer_snapshotLoadRollback/partially_readable_snapshot_is_refused` prove keys/versions survive with zero extra saves and zero notifications. The disclosed semantics change (the shrink-limit restore now refuses while `Open` keeps count-and-skip resilience) is documented in `docs/API.md` and the review disposition.
- **WR-01** — The page-token domain is `persist.MaxDumpLineBytes` (the readable dump's line bound), re-derived ceilings are 187 472/190 440 receive and 6 MiB + 125 008 send, and `TestServer_listTokenAboveWriteCeiling` resumes past a 1 124-byte key.
- **WR-02** — The Watch snapshot stores version + checksum and delivers equal-version content/remove-wins changes (`watch.go:63-121,459-471`; `TestWatch_equalVersionTieBreakDelivered`).
- **WR-03** — `examples/client` is `example.com/sup-kv-client` outside the root prefix; the internal-import compile rejection is documented and the depguard rule remains the enforced boundary.
- **IN-01…IN-04 and N-1** — the unreachable skip is gone, the `S-ALG-CU-11`/`S-PERS-11`/`S-PERS-12` citations resolve to the current functions, the transport doc now claims only the pinned settings, the largest-value round trip exists, and all citations were re-pointed after the CR-01 insertion.

The frozen contracts are intact: both red-test files and `scripts/red-expectations.tsv` hash exactly to the phase-start pin, the red runner is 7/7 with both mutants killed, and the working tree is clean apart from the runner's new untracked evidence directory. No gate was narrowed or skipped in this verification.
