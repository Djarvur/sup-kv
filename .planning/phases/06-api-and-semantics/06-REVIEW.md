---
phase: 06-api-and-semantics
reviewed: 2026-11-23T02:46:59Z
depth: deep
files_reviewed: 75
files_reviewed_list:
  - .github/workflows/ci.yml
  - .gitignore
  - .golangci.yml
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
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
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
findings:
  critical: 1
  warning: 3
  info: 4
  total: 8
  necessity: 1
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-11-23T02:46:59Z
**Depth:** deep
**Files Reviewed:** 75
**Status:** issues_found

## Summary

Reviewed the full Phase-6 scope (`2b36688..061fdb2`, 75 existing files) against locked
decisions D-01…D-07 and the frozen-R-01 reconciliation: the write-time peer capture and
the honest zero-peer report (APIS-01), the two-mode `List` with the byte-budgeted keyset
page (APIS-02), the total path-free mapper (APIS-03), the durability-gated Watch feed
(APIS-04), the explicit transport envelope plus one-way health (APIS-05), the public proto
move with the compatibility alias and the client-example module (APIS-06), and the
full-rollback snapshot load (APIS-07). I also re-checked the repository rules: the new
depguard rule is additive and effective, every new `//nolint` names its linter with a
reason, the vendored config is byte-unmodified behind its `Origin:` header, `go.mod`/
`go.sum` did not change, and `go test ./...` still excludes the nested client module.
The listed `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` was deleted by the move
(its generated Go package lives on as the alias package); it is omitted from the list
above because there is no content left to review.

**One blocker.** The full rollback reads the *decoded* record map as the snapshot's key
set. A record that a snapshot file still names but that the decoder discarded per record
(a checksum mismatch, a bad `record_version`, a malformed line, or the tail of a legacy
census-less generation) is therefore indistinguishable from "absent from the snapshot",
so `Restore` deletes the **live** key with a fresh tombstone and `SnapshotLoad`
broadcasts that tombstone to every peer. I reproduced it in a scratch copy of this commit:
corrupting one record's checksum in an otherwise valid snapshot makes `Restore` return
nil, delete the key, and include the rollback tombstone in its notify list. This is
silent data loss that replicates — the previous additive behavior left the key untouched.

The remaining findings are one contract gap in paged `List` (the page token cannot name a
pre-upgrade key longer than `storage.MaxKeyBytes`, which the store and `Get` explicitly
keep serving), one pre-existing exact-cut defect the phase's new API contract now claims
against (an equal-version tie-break install is suppressed for a subscriber whose snapshot
holds the same version), one false verification claim (the client example's module path
sits inside the root module's path prefix, so Go's internal rule does not stop it
importing `internal/` — verified: the committed module builds with an internal import,
while an out-of-tree module path rejects it), and four information items (one necessity
branch, one stale registry citation, one transport-doc overclaim, one missing
largest-value-limit round-trip).

**The phase's core claims otherwise check out.** The write-time capture is taken once and
feeds both the notification and the report (`server.go:212`, `:234`, `:238`, `:273`,
`:286`, `:290`); the tracker is scoped to the captured set both in the core
(`core.go:2268-2294`) and the adapter (`repl.go:783-795`, `:854-867`); the discovery flush
no longer grows the wait set or the ledger (`core.go:1413-1426`), and `peerGen` republishes
the id mirror on exactly the set changes (`core.go:1404-1408`, `:2762`, `repl.go:1188-1192`);
the zero-peer and nil-seam answers are the honest `(false,0,0)` (`wait.go:72-88`, `repl.go:830-846`),
with the tracker's `total == 0` branch (`repl.go:840-846`) closing the race where a
stranger's Ack could otherwise resolve an empty promise; the mapper is total, ordered as
documented, and every client-visible string is a constant or sentinel (`server.go:488-528`,
verified by grep over every `status.Error*` site); the durability watermark is published
after `confirmWaiters` and from the generation read *before* `Records()` (`scheduler.go:228-248`),
the pending queue is bounded with disconnect-and-resnapshot overflow (`watch.go:366-408`),
the startup load is suppressed by a synchronous guard (`loader.go:75-88`), and hub shutdown
ends streams `Unavailable` (`watch.go:475-481`, `:501-539`); the transport limits derive
from the configured value limit (`limits.go:86-117`, `main.go:735`), `health.Server.Shutdown`
gives the one-way transition (`gate.go:138-140`, `main.go:939`), and the worst-case List
page provably fits `MaxSendBytes` (94 max-size entries, ~41 KB of slack under
`listPageByteBudget+listEnvelopeOverheadBytes`). The public package is a real descriptor-pinned
contract (`schema_test.go`), the alias re-exports every top-level generated identifier and
cannot drift for the frozen tests, the frozen red-test files are byte-identical to
`2b36688`, and the client example's `go.mod`/`go.sum` are tidy (`GOFLAGS=-mod=readonly
go build ./... && go vet ./...` clean; a no-network `go mod tidy` produces no diff).

**Gates run for this review.** `go build ./...` and `go vet ./...` clean;
`go tool golangci-lint config verify` and `go tool golangci-lint run ./...` exit 0;
`go tool buf lint`, `go tool buf format --diff --exit-code` and `go tool buf breaking
--against '.git#ref=refs/remotes/origin/master'` exit 0 (the cross-layout `go_package`
waiver works); `go test -race -count=1 ./...` green for every package (no known-accepted
flake fired in this run); `examples/client` builds and vets standalone. Accepted residuals
were not re-raised (UDP port contention/lifecycle readiness, `keyCountCostFlat`, the
unbounded peer table, Phase-5 IN-04).

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Full rollback treats per-record decode discards as "absent from the snapshot" — deletes live keys and replicates the deletes

**File:** `internal/app/persist/snapshot.go:107` (the discarded `DecodeStats`), `:145-175` (`replaySnapshot`), `:182-193` (`replayKeys`), `:217-230` (`rollbackKey`), `:232-259` (`restoreChanges`); `internal/app/persist/codec.go:265-272`, `:337-361`, `:312-326`

**Issue:** `Decode` is deliberately tolerant per record: a line that is malformed but
newline-terminated, one whose `record_version` is not 16 hex bytes, one whose value
exceeds the configured limit, and one whose checksum mismatches are **counted and
discarded**, and `Decode` still returns `err == nil` whenever the `record_count` census is
satisfied (`codec.go:265-279`, `:312-326`, `:337-361`). `Restore` throws that information
away — `records, _, err := Decode(data, e.valueMaxBytes)` (`snapshot.go:107`) — and the new
rollback set is computed from the decoded map alone: `replayKeys` adds every current live
key that is not a key of `records` (`:182-193`), `rollbackKey` writes a fresh tombstone for
it (`:217-230`), and `restoreChanges` treats it as a change (`:243-259`). A snapshot entry
that failed to decode is therefore indistinguishable from a key the snapshot never named,
so `Restore` deletes the key the snapshot *does* name. `rollbackKey`'s own doc ("deletes
one current live key the snapshot does not name") and `docs/API.md`'s "Every current live
key absent from the snapshot is deleted" are both false for this case.

The result is silent, replicating data loss: `Restore` returns `nil`, the rollback
tombstone is part of the returned `[]RestoredRecord`, `SnapshotLoad` notifies exactly that
list (`server.go:356-366`), and the fresh tombstone version dominates the peers' good
copies by ordinary LWW — one damaged recovery artifact deletes the key cluster-wide. The
pre-phase additive `Restore` left such a key untouched. The same mechanism converts the
documented v1.0 limitation (a legacy census-less generation truncated at a line boundary,
`codec.go:230-234`) into deletes of the missing tail keys instead of skips.

Reproduced in a scratch copy of this commit: seed `keep` and `victim`, `Snapshot("snap")`,
flip only `victim`'s `"checksum"` in the snapshot file (JSON and census intact), then
`Restore("snap")`:

```
restored=[{Key:keep ...} {Key:victim Version:[...] Data:[] Deleted:true}]
Get("victim") = key not found          // the live key was rolled back
Restore error = nil                    // and the corruption was never surfaced
```

**Fix:** compute the rollback set from the keys the snapshot **names**, not from the keys
that decoded. Preferred shape: have `Decode` also return the set of keys whose lines were
rejected (it already knows them per class) and exclude those keys from `replayKeys` and
from `restoreChanges`' "extra live key" loop, leaving them untouched exactly as the
shrunk-limit skip is documented to do. A simpler, stricter alternative refuses the load:

```go
records, stats, err := Decode(data, e.valueMaxBytes)
if err != nil {
    return nil, err
}
if stats.Total() > 0 {
    // A full rollback over a partially decoded key set would delete keys the
    // snapshot names but could not decode; refuse instead of inventing deletes.
    return nil, fmt.Errorf("%w: snapshot %q has %d unreadable records", ErrSnapshotIncomplete, name, stats.Total())
}
```

(and map the new sentinel to `FailedPrecondition` in `mapError`; `docs/API.md` then needs
the row). Add two tests: (a) a snapshot with exactly one corrupt checksum over a live key
— `Restore` must not produce a delete for it (the shrunk-limit skip semantics), and (b) a
census-less truncated legacy snapshot — the missing tail keys must not be rolled back.

## Warnings

### WR-01: A page token cannot name a pre-upgrade key longer than `storage.MaxKeyBytes` — the paged walk stops with `InvalidArgument`, and the send-limit bound assumes 1 024-byte keys

**File:** `internal/grpcapi/list.go:64-77` (`listTokenEncodedBytes`, `listEnvelopeOverheadBytes`), `:110-142` (`listPage`), `:208-210` (token source), `:235-241` (`decodeListToken`); `internal/app/storage/store.go:26-33` (permissive reads), `:261-272` (`LiveVersions`); `internal/grpcapi/server.go:131-136` (`Get`'s permissive-read contract)

**Issue:** The store explicitly keeps serving pre-upgrade records under keys longer than
`storage.MaxKeyBytes` ("Validation is write-side only: the store keeps serving pre-upgrade
records under longer keys, so reads stay permissive", `store.go:26-33`; `Get` repeats it),
and `persist.Decode` applies no key-length bound (`codec.go:337-361`), so such a key loads
from a dump and is listed by `LiveVersions`. The paged walk then encodes that key as
`next_page_token` (`list.go:208-210`) and the next request is rejected, because
`decodeListToken` refuses any decoded key longer than `storage.MaxKeyBytes`
(`list.go:235-241`). The client can never fetch the keys that follow it with
`page_size > 0`; the loop the API documents as "send it back as page_token to resume
strictly after that position" ends in `InvalidArgument`. The same assumption underlies the
send-limit proof: `listTokenEncodedBytes` is sized from a 1 024-byte key
(`list.go:64-68`), so for a longer key the token can exceed
`listEnvelopeOverheadBytes`, and a nearly full page plus that token can marshal past
`MaxSendBytes` (`limits.go:86-97`, `list.go:218-220` counts the key twice — in `keys` and
`records[].key` — but the token is a third, base64-expanded copy).

Reproduced in a scratch copy through `listPage` directly: three live keys
`0first`, a `MaxKeyBytes+100`-byte key, `z`; `page_size=2` returns
`["0first", longKey]` with a 1 499-byte token, and the follow-up request fails with
`InvalidArgument: invalid page token` — `z` is unreachable in paged mode.

**Fix:** make the token domain consistent with what `List` admits. Either (a) skip keys
longer than `storage.MaxKeyBytes` in the paged walk (documented: the paged listing covers
API-writable keys; the legacy mode still lists every live key), or (b) widen the domain —
accept tokens up to the dump line's key capacity in `decodeListToken`, derive
`listTokenEncodedBytes` from that same bound, and let `MaxSendBytes` grow by the same
amount — so the walk is resumable and the envelope proof holds for every key the store can
hold. Either way, add a case to `TestServer_listToken`/`TestListPage_maxKeySerializedBound`
with a key above the ceiling.

### WR-02: The exact-cut floor suppresses a legitimate equal-version tie-break install — a subscriber can silently miss a content change or a remove-wins delete

**File:** `internal/grpcapi/watch.go:92-97` (the snapshot map stores only versions), `:340-360` (`handleSubscribe`), `:428-440` (`wants`); `internal/app/storage/store.go:57-63` (the tap contract); `docs/API.md:210-213` (the phase's new claim)

**Issue:** `wants` suppresses an event unless its version strictly exceeds the subscriber's
snapshot version for that key (`watch.go:438-439`), on the stated premise that "only
strictly greater versions ever install". That premise is false: `Store.join` installs an
equal-version content tie-break winner (remove-wins, or the greater `Data`), fires the
change tap, and the store's own `OnChange` doc says "a tie-break win at an equal version
IS a change" (`store.go:57-63`, `join` at `store.go:418-440`). When the subscriber's
snapshot already holds that version with the losing content, the winner's confirmed event
is dropped, so the subscriber's view of the key diverges from the store's (or, for
remove-wins, it never learns the key was deleted) until it resubscribes. This contradicts
docs/API.md's new contract text: "an equal-version tie-break winner can install once. An
event already contained in the subscriber's snapshot is suppressed" — the winner is not
contained in that snapshot. APIS-04's "each confirmed write yields exactly one event" is
not met for this install. (The `wants` code predates this phase; the contract sentence, and
the acceptance argument built on it, do not.)

Reproduced in a scratch copy of this commit: a live `k@V` with `aaa`, subscribe (snapshot
holds `k@V`/`aaa`), then install `k@V` with `bbb` through the store's tie-break, ingest the
tapped event and confirm its generation — the subscriber receives nothing.

**Fix:** keep the seconds of content that the floor needs. The store checksum is exactly
that identity (keyed + versioned + content-derived, recomputed on every install), so
`snapshot map[string]uint32` (checksum) plus the version is enough:

```go
// suppress only an event the snapshot already contains exactly
if ev.rec.Version.Compare(snapVer) > 0 {
    return true
}
return ev.rec.Version == snapVer && ev.rec.Checksum != snapChecksum
```

Add a Watch test that installs an equal-version content winner (and a remove-wins
tombstone) after a subscribe and requires one delivered event each.

### WR-03: The client example is not the "compile-time proof" it claims — its module path is inside the root module's prefix, so Go's internal rule still admits `internal/` imports

**File:** `examples/client/go.mod:1`; `examples/client/main.go:3-8`; `examples/client/README.md:3-12`; `.github/workflows/ci.yml` (`client-example` job), `AGENTS.md` Layout

**Issue:** The example's package path is `github.com/Djarvur/sup-kv/examples/client`, which
shares the `github.com/Djarvur/sup-kv/` prefix with the `internal` tree; Go's internal rule
is prefix-based, so the example may import `github.com/Djarvur/sup-kv/internal/...` and
still build. The README ("it fails to build if the public package ever grows an `internal/`
dependency"; "a successful build here is the compile-time proof of consumer importability")
and `main.go` ("doubles as the compile-time proof that the public API has no internal/
dependency") are therefore false, and the phase's acceptance argument leans on them.
Verified in a scratch copy: adding a blank import of
`github.com/Djarvur/sup-kv/internal/app/storage` to `examples/client` builds cleanly with
the committed module path; renaming the module to `example.com/supkv-client` makes the
same import fail with `use of internal package ... not allowed`. The actual enforcement is
the depguard rule: I verified `sup-kv-public-api-no-internal` fires on an internal import
in `api/supkvapi/v1` when only `depguard` runs (the rule is effective and correctly scoped;
the earlier full-config non-firing on my probe was golangci-lint's `uniq-by-line` merging a
same-line `revive` alias diagnostic — the same import with no redundant alias is reported).

**Fix:** name the example module outside the root prefix (e.g. `module supkv-example`),
keeping `require github.com/Djarvur/sup-kv v0.0.0` plus
`replace github.com/Djarvur/sup-kv => ../..`; the probe above shows the claimed build
failure then actually happens, and `make client-example` keeps working. Alternatively drop
the proof claim from both docs and cite the depguard rule as the enforcement.

## Info

### IN-01: `replayRecord`'s `ErrValueTooLarge` skip is a belt over a state the decoder already excluded

**File:** `internal/app/persist/snapshot.go:284-286` (the skip), `:107` (the decode call), `:200-211` (`replaySnapshotKey`); `internal/app/persist/codec.go:349-351`; `internal/app/persist/engine.go:183`; `internal/app/storage/store.go:136-158`, `:278-286`
**Category:** necessity
**Issue:** Subtype: already-defended-elsewhere. `Restore` decodes with `e.valueMaxBytes`,
which `New` captures from `store.ValueMaxBytes()` — the store's normalized limit, the same
value `Decode` applies — and `Decode` drops every record whose `Data` exceeds that limit as
`decodeValueLimit` (`codec.go:349-351`), so no record reaching `replayRecord` can make
`store.Set` return `ErrValueTooLarge` (`store.go:278-286`, the store's only rejection).
`replayRecord` is called only from `replaySnapshotKey`. The branch cannot fire for any
decoded record; it is the same belt Phase 5 removed from `mergeRecords` for the identical
reason (05 IN-05), and the doc's "mirroring Open, so a config-shrunk value limit never
aborts a restore" is true because of the decoder, not because of this skip.
**Fix:** drop the special case and make the totality explicit, e.g.
`ver, _, _ := store.Set(key, rec.Data)` with a comment naming the decoder and the store as
the bound's authorities, or move the invariant note to `Decode`'s contract.

### IN-02: The `S-ALG-CU-11` row the phase edited still cites a line range past the end of `wait.go`

**File:** `docs/SPEC-DEVIATIONS.md:116`
**Issue:** The row cites `internal/grpcapi/wait.go:94-225`; the file is 127 lines in this
tree (and was 113 lines at `2b36688`, so the range was already stale). This phase edited
that row (it updated the proto path to `api/supkvapi/v1/supkvapi.proto:63-80`), so the
one stale Go anchor it leaves behind is in the phase's own diff. The registry's value is
that its citations can be followed.
**Fix:** re-point the row at the current code, e.g. `internal/grpcapi/wait.go:72-88`
(`waitForReplication`) plus `:93-98` (`writePeers`), or cite the file without a range.

### IN-03: The transport documentation claims more explicitness than the phase configures

**File:** `docs/API.md:267-268`; `internal/grpcapi/limits.go:45-48`; `cmd/sup-kv/main.go:854-866`
**Issue:** "Every gRPC server setting is explicit; no message size or keepalive cadence is
inherited from a gRPC default" and "nothing the server accepts or sends is governed by a
gRPC default" overstate what `TransportLimits.ServerOptions` sets: `MaxRecvMsgSize`,
`MaxSendMsgSize`, the server keepalive and the enforcement policy (`limits.go:104-117`).
Other transport knobs (header-list size, connection timeout, stream/window sizes and the
concurrent-stream cap) remain gRPC defaults. D-05 asked for keepalive plus message-size
ceilings, and those are pinned correctly; only the absolute wording is wrong.
**Fix:** narrow both sentences to the pinned families, e.g. "the message-size ceilings and
the keepalive cadence are explicit; the remaining transport knobs keep gRPC defaults".

### IN-04: The largest-value-limit transport round-trip that the research requires is not tested

**File:** `internal/grpcapi/limits_test.go:25-45`; `internal/grpcapi/limits_internal_test.go:33-86`; `cmd/sup-kv/main_test.go:522-551`; `cmd/sup-kv/main.go:735`
**Issue:** Research Pitfall 11 asks for "a process-level round-trip at the largest accepted
value limit" so that computing the receive ceiling from the wrong constant (the storage
default instead of `cfg.valueMaxBytes`) is caught. `TestGRPC_maxRecv` exercises the
arithmetic only at `testValueLimit = 1024` (`server_test.go:37`); `TestLimitsFor` asserts
the 64 408-byte-table entry but not a request; and the boundary process test starts a node
at 64 408 and checks only that health reaches `SERVING` (`main_test.go:522-551`). A
regression such as `LimitsFor(storage.ValueMaxBytes)` at `main.go:735` would pass every
existing test and reject a legitimate maximum-size write at the transport.
**Fix:** extend `TestGRPC_maxRecv` (or the boundary process test) with a `Set` of a
`storage.MaxKeyBytes`-length key and a 64 408-byte value against a server built with
`LimitsFor(64408)`/`ValueMaxBytes: 64408`, asserting `OK`; the message is ~65.5 KB, well
under the 66 814-byte ceiling.

---

_Reviewed: 2026-11-23T02:46:59Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
