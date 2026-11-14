---
phase: 05-grpc-api-and-binary
reviewed: 2026-11-14T22:03:58Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto
  - internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go
  - internal/grpcapi/apiproto/supkvapi/v1/supkvapi_grpc.pb.go
  - internal/grpcapi/server.go
  - internal/grpcapi/watch.go
  - internal/grpcapi/gate_test.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait_test.go
  - internal/grpcapi/wait_integration_test.go
  - internal/grpcapi/watch_test.go
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
  necessity: 1
status: issues_found
---

# Phase 5: Code Review Report (incremental — delta 7b55ad1)

**Reviewed:** 2026-11-14T22:03:58Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Incremental review of commit 7b55ad1's phase-5 slice (diff base 9e754b5,
the phase's execution-era review): the client-facing schema's move into
its package-derived tree (`internal/grpcapi/apiproto/supkvapi/v1/`) plus
the service/message renames the buf STANDARD zero-waiver decision requires
(SupKV → SupKVService, WatchEvent → WatchResponse) and every consumer
update.

**The rename is complete and consistent.** Repo-wide greps over Go and
proto sources return ZERO hits for any stale symbol — `WatchEvent` (proto
type), `RegisterSupKVServer`, `NewSupKVClient`, `SupKVClient`,
`SupKVServer`, `UnimplementedSupKVServer`, `SupKV_WatchServer`, the
`SupKV_*` method-constant prefix, and the old wire path
`supkvapi.v1.SupKV/` (all now `SupKVService`-qualified). The remaining
"SupKV" mentions in source are prose comments only (gate.go:95,
server.go:2/64, server_test.go:170, main.go:358, main_test.go:271/369) —
product-name prose, accepted per this review's criterion that comments
are tolerated; the internal lower-case `watchEvent` domain type in
watch.go is deliberately unrenamed (it is not the wire symbol).

**The generated code is a clean mechanical regeneration, not a hand
edit.** Both new files carry intact DO-NOT-EDIT headers with the pinned
generator stamps (protoc-gen-go v1.36.12-devel, protoc-gen-go-grpc
v1.6.2), `source: supkvapi/v1/supkvapi.proto` (module-relative, matching
the move), `ServiceName: "supkvapi.v1.SupKVService"`, all seven
`SupKVService_*_FullMethodName` constants, and descriptor strings updated
down to the message-name length changes (`\xea\x03` → `\xf4\x03`,
`.supkvapi.v1.WatchEvent` → `.supkvapi.v1.WatchResponse`). The diff
against 9e754b5's generated files contains nothing but the move/rename
families plus the `file_internal_..._proto` → `file_supkvapi_v1_..._proto`
symbol-prefix renames. `make proto` re-run during this review reproduces
both files byte-identically; the `;apiproto` package alias preserves the
Go package name, so consumer diffs are import-path + renamed symbols
only.

**Consumers verified one by one:** server.go (embed
`UnimplementedSupKVServiceServer`, doc comments updated), watch.go
(`toProto` returns `*apiproto.WatchResponse`; `Watch`/`relayEvent` take
`SupKVService_WatchServer`), gate/server/wait/watch tests (import,
registration, client construction, `recvEvent` generic parameter), and
main.go — whose entire delta is the import line plus
`apiproto.RegisterSupKVServiceServer(gs, srv)`; `listenGRPC` and the rest
of the binary's behavior are otherwise unchanged (read in full at
main.go:356-369). `go build`/`go vet` clean; `go test -race` green on
`internal/grpcapi` and `cmd/sup-kv`.

**The wire-contract change is intentional and recorded.** Method paths
move to `/supkvapi.v1.SupKVService/*` and the streamed message type name
becomes `WatchResponse` (field numbers untouched — `Record record = 1`
keeps its slot, so framed payloads are unchanged; the names are visible
via descriptors/reflection). The maintainer decision (2026-11-14,
pre-release window, no external clients) is recorded at
`.planning/STATE.md:204` and acknowledged in
`06-CONTEXT.md` (D-16 and the note that 05-CONTEXT's SupKV/WatchEvent
name pins were superseded); buf breaking vs origin/master is correctly
guard-deferred (origin/master is at the phase-3 head — the proto tree has
never been compared against it).

No new findings in this delta. One still-open finding from the execution
-era review (IN-01, necessity) lives in the in-scope file server.go and
is re-reported unchanged below (same id, same finding — disposition row
preserved; CR-01/WR-01/WR-02 from that review are disposition: fixed and
stay closed). No structural-findings pre-pass and no external reviewer
evidence were provided.

## Narrative Findings (AI reviewer)

The single finding below is a carry-forward from the 2026-11-09 deep
review, re-verified present in the current server.go; the 7b55ad1 delta
touched server.go only in import/embed/comment lines.

## Info

### IN-01: mapError's status.FromError passthrough branch is unreachable — no internal/app error can be a status error

**File:** `internal/grpcapi/server.go:306-308`
**Category:** necessity
**Issue:** (Carry-forward, unchanged by 7b55ad1 — the
`if _, ok := status.FromError(err); ok { return err }` branch is still
present and still unreachable.) Generality without instantiation:
mapError's inputs come from the engine, storage, persist and persistfile
layers, none of which may import grpc (the depguard boundary
`sup-kv-app-no-io`/stdlib-only port rules keep `google.golang.org/*` out
of internal/app), so no error reaching this switch can be a status error;
every real input resolves through the sentinel table above or falls to
`codes.Internal`. The branch runs, but defends against nothing. It rests
on judgment (a future in-package caller could theoretically construct a
status error), hence Info rather than Warning.
**Fix:** Remove the passthrough branch and let the sentinel table plus
the Internal fallback be the whole contract — or keep it with a comment
naming the concrete future caller that would justify it.

---

_Reviewed: 2026-11-14T22:03:58Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
_Delta: 7b55ad1 (base 9e754b5)_
