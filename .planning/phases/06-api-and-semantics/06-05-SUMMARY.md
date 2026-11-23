---
phase: 06-api-and-semantics
plan: 05
subsystem: api-list-pagination
tags:
  - apis-02
  - list
  - keyset-pagination
  - page-size
  - page-token
  - next-page-token
  - live-view
  - byte-budget
  - cost-independence
  - proto-comments
  - stand-poll
requires:
  - phase: 06-api-and-semantics (06-01)
    provides: the public proto page fields — page_size=2, page_token=3, next_page_token=2, repeated Record records=3
  - phase: 06-api-and-semantics (06-04)
    provides: the total, path-free error mapper the new InvalidArgument message constants sit beside
  - phase: 03-protocol
    provides: Store.Version/Lookup/Versions, the caller-owned read-view shape LiveVersions mirrors
provides:
  - "Store.LiveVersions and Engine.LiveVersions: tombstone-free key to version under RLock, copying no Data"
  - "internal/grpcapi/list.go: maxListPageSize 256, listPageByteBudget 6 MiB, listRecordOverheadBytes 128, listEnvelopeOverheadBytes 1382, listEntrySerializedBytes, the RawURL base64 token codec, the page-size resolver and the two-mode page walk"
  - "Server.List: the legacy all-keys keys-only listing at page_size 0 and a bounded keyset page of aligned records at page_size > 0"
  - "public proto comments documenting both modes, byte-wise ordering, the position-not-snapshot token, the live-only view, the serialized-byte budget and the empty-token end signal"
  - "a paged nightly recovery poll in internal/stand/stand_test.go that follows next_page_token"
  - "plan 06-07's input: listPageByteBudget + listEnvelopeOverheadBytes for MaxSendBytes"
affects:
  - 06-07-max-message-sizes
  - 06-13-phase-gate
  - phase-9-api-docs
actuals:
  tokens: 16257
  tasks: 3
  commits: 7
  plan_head_before: b2a1a88cfa39bea9fc7ad257e7aa583687665888
  plan_head_after: 14b8a9179edff1de6d2670f393876e916434f29b
tech-stack:
  added: []
  patterns:
    - "One handler body, two truthful modes: page_size 0 keeps the frozen legacy contract over the live key→version view; page_size > 0 adds keyset pagination without changing the wire shape."
    - "A serialized-page budget costs each entry as its key twice (keys and records[].key), its value once and a fixed framing allowance, stops below the page size, and keeps a one-entry floor so progress is guaranteed."
    - "The page walk takes a lookup function, so the view→lookup race window is pinned deterministically in an internal test instead of raced through the RPC surface."
key-files:
  created:
    - internal/grpcapi/list.go
    - internal/grpcapi/list_internal_test.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-05-task1.json
    - .planning/phases/06-api-and-semantics/red-evidence/06-05-task2.json
    - .planning/phases/06-api-and-semantics/red-evidence/06-05-task3.json
  modified:
    - internal/grpcapi/server.go
    - internal/grpcapi/server_test.go
    - internal/app/storage/store.go
    - internal/app/storage/view_test.go
    - internal/app/persist/facade.go
    - internal/stand/stand_test.go
    - api/supkvapi/v1/supkvapi.proto
    - api/supkvapi/v1/supkvapi.pb.go
    - api/supkvapi/v1/supkvapi_grpc.pb.go
key-decisions:
  - "The two modes share one view call: LiveVersions is taken once per request, the prefix filter and the token seek run over it, and Lookup is called only for entries the page returns — the legacy path never calls Lookup, so no value bytes cross it."
  - "The page token is the base64.RawURLEncoding of the last examined key, not an index or a snapshot: a token names a strict-greater position, so keys added or deleted between pages may appear or disappear, and an out-of-domain token is InvalidArgument, never a seek."
  - "listEnvelopeOverheadBytes is derived term by term — the ceil(4n/3) token expansion of a storage.MaxKeyBytes key (1 366), the token field tag and two-byte length (3), the gRPC message prefix (5) and the two repeated-field tags plus length varints (8) — for 1 382 bytes; plan 06-07 adds it to the budget and must not redefine it."
  - "The one-entry progress floor is a condition on len(page)>0 rather than a post-check: the over-budget entry is left unexamined and becomes the next page's first, so the token never skips it."
  - "The external cost tests restate the unexported envelope bound as listEnvelopeCeiling (2 048) and raise only the harness client's receive limit to 8 MiB; plan 06-07 raises the production limits from the same constants."
  - "A tombstone race is skipped by the Deleted-flag-aware Lookup result and the token advances past the examined key; LiveVersions never infers a tombstone from empty data, so an empty live value stays listed."
patterns-established:
  - "A frozen single-call contract and a new paginated mode coexist behind one request field: page_size 0 is the documented legacy shape, page_size > 0 opts into pages, and a token without a page size is a client error."
  - "Cost independence is proven twice: a source-shape gate over the handler's call graph (LiveVersions present, no whole-store Records call) and marshaled-response bounds with production-size values."
requirements-completed: []
coverage:
  - id: D1
    description: "With page_size unset/0, List returns every live key under the prefix, byte-wise ascending, with zero records, no token and no value bytes; the 512-key single call proves the mode is unbounded by a page size."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list(LegacyAll|)$' -v -> PASS"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D2
    description: "With page_size > 0, List returns at most the resolved page size records with values, keys aligned to records, and a non-empty next_page_token unless the page ends the listing; following the token yields the remaining live keys exactly once."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list(LegacyAll|FirstPage|PageContinuation)$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "Boundaries are executable: negative page_size and a token without a positive page_size are InvalidArgument 'invalid page size'; page_size above 256 clamps; undecodable and over-MaxKeyBytes tokens are InvalidArgument 'invalid page token'; a token past the end is an empty terminal page; empty store and single key behave in both modes; a prefix filters before pagination."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_list(Boundaries|Empty|Token|Prefix|Race)$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D4
    description: "The live key→version view is tombstone-free by the stored Deleted flag (never by empty data), includes an empty live value, and returns a caller-owned map."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/storage/ -run '^TestStore_liveVersions' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D5
    description: "Both List paths cost only what they return: on a store with 100 off-page maximum-size values, the legacy response is bounded by the one key plus envelope, and a page_size-1 response by one value plus envelope."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_listLegacyCostIndependent$' -v -> PASS"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_listCostIndependent$' -v -> PASS"
        status: pass
      - kind: process
        ref: "awk '/func .*listPage/,/^}/' internal/grpcapi/list.go | grep -c LiveVersions >= 1; grep -v '^\\s*//' internal/grpcapi/list.go | grep -c '\\.Records()' == 0 -> both exit 0"
        status: pass
    human_judgment: false
  - id: D6
    description: "The 6 MiB serialized-byte budget stops a page below the page size while guaranteeing progress, and the worst-case page of maximum-length keys and maximum-size values marshals within listPageByteBudget + listEnvelopeOverheadBytes (the input to plan 06-07's MaxSendBytes)."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_listByteBudget$' -v -> PASS (a first page below the page size with a non-empty token, 120 distinct keys over the walk)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestListPage_maxKeySerializedBound$' -v -> PASS"
        status: pass
    human_judgment: false
  - id: D7
    description: "A key that races into a tombstone or disappears between the view and its Lookup is skipped, keys and records stay aligned, and the token advances past the examined key so the next page progresses."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/grpcapi/ -run '^TestServer_listRace$' -v -> PASS (tombstoned and vanished variants)"
        status: pass
    human_judgment: false
  - id: D8
    description: "The public proto documents both modes, ordering, the position-not-snapshot token, the live-only view, the byte-budget upper bound and the empty-token end signal; the generated Go comments match and buf lint/format stay green."
    requirement: "APIS-02"
    verification:
      - kind: process
        ref: "go tool buf lint && go tool buf format --diff --exit-code -> exit 0; go tool buf generate api refreshed the comment-only pb.go diff"
        status: pass
    human_judgment: false
  - id: D9
    description: "The nightly recovery poll follows next_page_token on a page_size 256 request instead of one unpaged listing; the frozen R-01 test and the expectation table are byte-identical."
    requirement: "APIS-02"
    verification:
      - kind: process
        ref: "go vet ./internal/stand/ && grep -q NextPageToken internal/stand/stand_test.go -> exit 0"
        status: pass
      - kind: process
        ref: "git diff --exit-code b2a1a88 -- internal/stand/stand_red_test.go scripts/red-expectations.tsv -> exit 0 (frozen files untouched)"
        status: pass
    human_judgment: false
duration: 14min
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 05: Two-Mode List Pagination Summary

**`List` now serves two truthful modes over one value-free live view: the frozen all-keys, keys-only listing at `page_size` 0 and a 256-capped, 6 MiB-budgeted keyset page of records with an opaque position token at `page_size` > 0.**

## Performance

- **Duration:** 14 min (817 s)
- **Started:** 2026-11-23T00:00:23Z
- **Completed:** 2026-11-23T00:14:01Z
- **Tasks:** 3
- **Files modified:** 14 (11 code/proto, 3 RED-evidence records)
- **Commits:** 7 measured from ledger `b2a1a88..14b8a91`

## Accomplishments

- `Store.LiveVersions` / `Engine.LiveVersions`: the tombstone-free key→version view that copies no value bytes — the A-02 fix, replacing the whole-store `Records()` snapshot on the List path. Liveness is the stored `Deleted` flag, never `len(Data)`, so an empty live value stays listed.
- `internal/grpcapi/list.go`: both modes, the size resolver, the RawURL base64 token codec, the entry-cost function and the four size constants — `maxListPageSize` 256, `listPageByteBudget` 6 MiB, `listRecordOverheadBytes` 128, `listEnvelopeOverheadBytes` 1382 (derived: 1366-char token ceiling + 3 tag/length + 5 gRPC prefix + 8 repeated-field framing).
- `Server.List` is now a gate check plus the page helper. `page_size` 0 keeps the legacy contract the frozen R-01 test makes; `page_size` > 0 returns aligned records, values included, and a position token.
- The page walk stops on the serialized-byte budget below the page size, always includes its first entry, and skips a key that raced into a tombstone or vanished while the token advances past it.
- The public proto comments document both modes, ordering, the position-not-snapshot token, the live-only view, the budget upper bound and the empty-token end signal; the generated Go comments were regenerated (comment-only diff).
- The nightly stand recovery poll now follows `next_page_token` on a `page_size` 256 request, with a page-count guard so a stuck token fails the poll instead of hanging it.

## Task Commits

1. **Task 1 RED: failing page-listing and continuation tests** — `419728c` (test)
2. **Task 1 GREEN: live key view and two-mode handler** — `d45a7b7` (feat)
3. **Task 2 RED: boundary, token, prefix and empty-input tests** — `7641ebc` (test)
4. **Task 2 GREEN: page-size and token boundaries** — `54622b8` (feat)
5. **Task 3 RED: cost-independence and byte-budget tests** — `f65fe13` (test)
6. **Task 3 GREEN: serialized-byte page budget** — `2f751d1` (feat)
7. **Lint cleanup of the new tests** — `14b8a91` (refactor)

**Plan metadata:** see the final `docs(06-05)` commit.

## Files Created/Modified

- `internal/grpcapi/list.go` — the two modes, the four constants, the token codec, the size resolver, the entry cost and the page walk.
- `internal/grpcapi/list_internal_test.go` — the deterministic view→lookup race test and the worst-case serialized-bound test (package `grpcapi`, so the unexported walk and constants are visible).
- `internal/grpcapi/server.go` — `List` delegates past the gate; the old `Records()` snapshot is gone from the request path.
- `internal/grpcapi/server_test.go` — the table-driven List contract, legacy-all (incl. 512 keys), first page/continuation, boundaries, tokens, empty/single key, prefix and the production-limit cost tests; harness knobs for the production value limit and a discarding dump store.
- `internal/app/storage/store.go` — `LiveVersions`.
- `internal/app/storage/view_test.go` — the live-view property test.
- `internal/app/persist/facade.go` — the one-line `Engine.LiveVersions` delegate.
- `internal/stand/stand_test.go` — the paged nightly poll (`emptyDiskPageSize` 256).
- `api/supkvapi/v1/supkvapi.proto` (+ generated `.pb.go`) — the public pagination documentation.

## Decisions Made

- **One view call per request, lookups only for page entries.** `LiveVersions` is taken once; the prefix filter and the token seek run over it; `Lookup` is called only for entries the page returns. The legacy path never calls `Lookup`, so it copies no value bytes at all.
- **The token is a position, never a snapshot or an index.** It is the RawURL base64 of the last examined key; the next page seeks strictly greater. A malformed token or one over `storage.MaxKeyBytes` is `InvalidArgument` with a constant message; a token past the end is a legal empty terminal page.
- **The envelope bound is derived, not guessed.** `listEnvelopeOverheadBytes` = ceil(4n/3) token expansion for `storage.MaxKeyBytes` (1 366) + token field tag/length (3) + gRPC message prefix (5) + two repeated-field tags and length varints (8) = 1 382. Plan 06-07 consumes it with the budget and must not redefine it.
- **The one-entry floor is structural.** The over-budget entry is left unexamined and becomes the next page's first entry, so the token can never skip it and an empty non-terminal page is impossible.
- **Test-side restatement of the envelope.** The external cost tests use `listEnvelopeCeiling` (2 048) and raise only the harness client's receive limit to 8 MiB, because a budgeted page legitimately exceeds gRPC's 4 MiB default; production limits are plan 06-07's.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The store-view test could not precede its symbol**
- **Found during:** Task 1
- **Issue:** `TestStore_liveVersions` calls `Store.LiveVersions`, which the Task-1 tree does not define, so the storage test package would not compile — and the go-test RED classifier refuses `[build failed]` runs as evidence.
- **Fix:** The compile-clean RED record is the gRPC-level first-page/continuation failure (`06-05-task1.json`); the store-view test landed in the GREEN commit and pins the implementation. The plan's "tests first" intent is preserved where machine-verifiable RED exists.
- **Files modified:** `internal/app/storage/view_test.go`
- **Commit:** `d45a7b7`

**2. [Rule 3 - Blocking] The deterministic race test required the lookup seam**
- **Found during:** Task 2
- **Issue:** A tombstone racing between `LiveVersions` and `Lookup` inside one call cannot be scheduled through the RPC surface without flakiness.
- **Fix:** `pagedResponse` now takes a `lookup func(key string) (storage.Record, bool)` (production passes `engine.Lookup`), and `TestServer_listRace` lives in `list_internal_test.go` where a fake lookup answers tombstoned/vanished for one key deterministically.
- **Files modified:** `internal/grpcapi/list.go`, `internal/grpcapi/list_internal_test.go`
- **Commit:** `54622b8`

**3. [Rule 1 - Bug] The harness client rejected a legitimately budgeted page**
- **Found during:** Task 3 RED
- **Issue:** The 6 MiB budget exceeds gRPC's 4 MiB default client receive limit, so the byte-budget test failed with `ResourceExhausted` before reaching its assertion.
- **Fix:** The harness client opts into an 8 MiB receive limit (`maxTestRecvBytes`) with a comment naming plan 06-07 as the production owner of the real limits; no production setting changed.
- **Files modified:** `internal/grpcapi/server_test.go`
- **Commit:** `f65fe13`

### Plan-text clarifications

- **Task 3's legacy cost test** said "one small key that sorts first" and "returns exactly one key" with an empty prefix, which is arithmetically impossible with 100 live bulk keys. The test uses `Prefix: "a"` so the response is exactly the one small key, while the hundred maximum-size values stay in the store off-page — the property the test exists to prove.
- **The committed `.pb.go` files** were regenerated after the proto comment edit (comment-only diff). The plan's `files_modified` did not list them; leaving the committed generated documentation stale would contradict `make proto`'s contract.

## Issues Encountered

- The strict lint gate flagged the new tests for inline error handling (`noinlineerr`), proto field access (`protogetter`), a `max` identifier (`revive`), and a stdlib doc link (`godoclint`). All fixed in `14b8a91`; `make lint` is green.
- `roadmap update-plan-progress 06` matched the v1.0 Hardening row (phase 6) instead of the v1.1 API row: it rewrote that archived row to `6/13 In Progress` and left the v1.1 row at `5/13`. Repaired by hand (the `919a471` 06-01 precedent): v1.0 back to `9/9 Complete`, v1.1 forward to `6/13 In Progress`.
- `state update-progress` declined with "phase scope is unscoped, not complete"; the phase-level progress bar therefore stays at 33% until the phase closes. Position, plan counter, metric row and session fields all advanced.
- No authentication gates and no user setup were needed.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. Every path is wired: the legacy mode serves the frozen contract, the paged mode serves records and tokens, and the constants feed plan 06-07.

## Next Phase Readiness

- APIS-02's REQUIREMENTS checkbox stays pending: `requirements ready-ids` reports it blocked (the #2388 shared-ID gate among 06-01/06-05/06-07), so this plan marks no requirement complete.
- Plan 06-07 can now size `MaxSendBytes` as `listPageByteBudget + listEnvelopeOverheadBytes` and must consume, not redefine, both constants.
- The frozen `internal/stand/stand_red_test.go` and `scripts/red-expectations.tsv` are byte-identical to the plan's base commit.
- Ready for 06-06.

## Self-Check: PASSED

All listed key files exist and all seven task commits are ancestors of HEAD.
