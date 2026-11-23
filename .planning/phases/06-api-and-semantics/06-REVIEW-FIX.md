---
phase: 06-api-and-semantics
fixed_at: 2026-11-23T03:16:10Z
review_path: .planning/phases/06-api-and-semantics/06-REVIEW.md
iteration: 1
findings_in_scope: 8
fixed: 8
skipped: 0
status: all_fixed
---

# Phase 06: Code Review Fix Report

**Fixed at:** 2026-11-23T03:16:10Z
**Source review:** `.planning/phases/06-api-and-semantics/06-REVIEW.md`
**Iteration:** 1
**Base commit:** `fd54a3a` (the review commit); fixes land in `0c6dbf3..a1e8bfd`
on `gsd/v1.1-milestone`.

**Summary:**
- Findings in scope: 8 (1 critical, 3 warning, 4 info; the necessity-marked
  IN-01 is in scope under `fix_scope: all`)
- Fixed: 8
- Skipped: 0

**Isolation:** `workflow.use_worktrees` is `false` in `.planning/config.json`,
so all edits and commits were made in the main checkout, per the documented
opt-out. No worktree, temp branch or recovery sentinel was created.

## Fixed Issues

### CR-01: Full rollback treats per-record decode discards as "absent from the snapshot"

**Files modified:** `internal/app/persist/errors.go`,
`internal/app/persist/snapshot.go`, `internal/app/persist/engine_test.go`,
`internal/grpcapi/server.go`, `internal/grpcapi/errors_internal_test.go`,
`internal/grpcapi/rollback_test.go`, `docs/API.md`
**Commit:** `0c6dbf3` (test lint follow-up in `a1e8bfd`)
**Applied fix:** `Restore` now reads the `DecodeStats` it previously discarded
and refuses the whole load — `ErrSnapshotIncomplete`, wrapping the snapshot
name and the discarded-line count — whenever `stats.Total() > 0`, before any
store mutation, replay, rollback delete, flush or notify. The new sentinel is
mapped to `codes.DataLoss` with the constant message `snapshot does not decode
cleanly` (chosen over `Unavailable` because the artifact exists and was read
but part of its content is unrecoverable), documented in the `mapError` table
and in the `docs/API.md` error table. Full-rollback deletion therefore runs
only on a cleanly decoded snapshot; the previous additive failure mode (a
replicated tombstone for a live key the snapshot names) is impossible.

Regression tests:
- `TestEngine_restorePartialSnapshotRefused` (persist): seeds `keep` and
  `victim`, snapshots, flips only `victim`'s checksum in the encoded
  snapshot, restores. Asserts `ErrSnapshotIncomplete`, a nil record list, both
  live values and versions untouched, and zero extra dump saves.
- `TestServer_snapshotLoadRollback/partially_readable_snapshot_is_refused`
  (grpcapi): the same fixture through `SnapshotLoad`; asserts `DataLoss` with
  the constant message, zero notify calls, and both live keys unchanged.
- `TestMapError_totalTable` gains the `ErrSnapshotIncomplete` row.
- `TestEngine_configShrunkValueLimits` was updated to the mandated semantics:
  a value-limit discard is a discard too, so the shrunk-limit restore now
  refuses the snapshot instead of silently skipping the record. The `Open`
  resilience assertions (discard count, `Get`, `Set`) are unchanged and the
  test now also proves the refusal applied nothing.
- `TestEngine_restoreSkipsReplay` keeps only the tombstone-with-nothing-to-beat
  skip (the value-limit shape is no longer a skip).

**Documented residual:** a legacy v1.0 snapshot has no `record_count` census,
so a truncation exactly at a line boundary is indistinguishable from a
shorter complete dump — the codec's documented compatibility limitation
(`codec.go`: legacy path). Every per-record discard class the decoder *can*
detect is now refused. Snapshots created by this version always carry the
census, so the residual applies only to a hand-carried pre-upgrade artifact.

**Docs:** `docs/API.md` gains the `DataLoss` row and a snapshot bullet stating
the clean-decode precondition; the rollback wording no longer implies partial
snapshots are tolerated.

### WR-01: A page token could not name a pre-upgrade key longer than `storage.MaxKeyBytes`

**Files modified:** `internal/app/persist/codec.go`,
`internal/grpcapi/list.go`, `internal/grpcapi/limits.go`,
`internal/grpcapi/server_test.go`, `internal/grpcapi/list_internal_test.go`,
`internal/stand/stand_test.go`, `cmd/sup-kv/api_docs_test.go`, `docs/API.md`
**Commit:** `acb4620` (test lint follow-up in `a1e8bfd`)
**Applied fix:** the dump format's maximum record line is now exported as
`persist.MaxDumpLineBytes` (still the compile-time constant the reader buffer
uses) and is the page-token cursor domain: `listTokenKeyBytes` derives from
it, `listTokenEncodedBytes` from that (124 992 characters at the format
maximum, versus 1 366 at the write ceiling), and `decodeListToken` rejects
only an undecodable token or one above the format bound. `storage.MaxKeyBytes`
no longer appears in the token path. A key rides inside one record line, so
no readable dump can name a longer key — which is exactly the pre-upgrade key
set the store keeps serving.

The send-ceiling proof follows the same envelope: `listEnvelopeOverheadBytes`
is now 125 008 bytes, and `MaxSendBytes` = 6 MiB page budget + that envelope.
Because gRPC's receive ceiling is per-message and must admit the follow-up
`ListRequest` carrying that token, `MaxRecvBytes` grows to the same envelope
plus key + effective value limit (187 472 at the default limit, 190 440 at
the 64 408-byte datagram ceiling); the docs table and the doc-anchor literal
were updated to the new numbers.

Tests:
- `TestServer_listTokenAboveWriteCeiling` reproduces the reviewer's boundary
  end-to-end: three live keys (`0first`, a 1 124-byte key, `z`), `page_size=2`
  returns the first two and a **1 499-character** token, and the second page
  succeeds with `["z"]`.
- `TestListPage_tokenAboveWriteCeiling` proves the internal half: the token
  round-trips through `decodeListToken`, the marshaled page stays within
  `listPageByteBudget+listEnvelopeOverheadBytes`, and the next page resumes.
- `TestServer_listToken`'s rejection case now uses a key above
  `persist.MaxDumpLineBytes` (an actually out-of-format position) and its doc
  states that a key above the write ceiling is a legal position.
- `TestStand_oversizeKeyF2c` was updated to the new envelope: the review's
  70 000-byte key now reaches the handler's `InvalidArgument` key bound, and a
  key derived from `grpcapi.LimitsFor(0).MaxRecvBytes` (so the test cannot
  drift) proves the transport's `ResourceExhausted` verdict. Both rejected
  keys are asserted absent from the peer endpoint and from the final dump.

### WR-02: The exact-cut floor suppressed legitimate equal-version tie-break installs

**Files modified:** `internal/grpcapi/watch.go`,
`internal/grpcapi/watch_test.go`, `docs/API.md`
**Commit:** `73d3549`
**Applied fix:** each subscriber's snapshot now stores a content identity per
key (`watchSnapshotContent`: the version plus the store checksum, 20 bytes
per key, never the value bytes) instead of the version alone. `wants`
suppresses an event only when it is already contained exactly — same version
AND same checksum — and delivers an equal-version event whose content differs,
which is precisely the store's tie-break law (`storage.join`; the `OnChange`
doc: "a tie-break win at an equal version IS a change"). A key absent from the
snapshot is still always delivered; a strictly greater version is still
delivered.

Test: `TestWatch_equalVersionTieBreakDelivered` subscribes over a seeded
`k@V/aaa`, then installs `k@V/bbb` (greater-Data winner) and `k@V` tombstone
(remove-wins) through `Engine.ApplyRemote`; each must arrive once with the
install's exact version and content. Both were silently dropped before the
fix.

**Docs:** the Watch ordering bullet now says an event is suppressed only when
the snapshot already contains the same version **and** content.

### WR-03: The client example was not the compile-time proof it claimed

**Files modified:** `examples/client/go.mod`, `examples/client/main.go`,
`examples/client/README.md`, `AGENTS.md`, `.gitignore`
**Commit:** `73eb0b8`
**Applied fix:** the example module is now `example.com/sup-kv-client` —
outside the root module's `github.com/Djarvur/sup-kv/` prefix — with the same
`replace github.com/Djarvur/sup-kv => ../..`, so Go's prefix-based internal
rule rejects an import of the root module's `internal/` tree from the example.
Verified by a throwaway copy: adding a blank internal import to the new module
fails with `use of internal package ... not allowed` (the same import built
cleanly under the old in-prefix path). The README and `main.go` now name the
depguard rule `sup-kv-public-api-no-internal` as the enforced boundary and the
out-of-prefix module path as the compile-time second proof. `AGENTS.md`'s
Layout names the module path; `.gitignore` follows the renamed build artifact
(`sup-kv-client`, with the old `client` entry kept for stale trees).

**Gates:** `make client-example` is green (build + vet + `go mod tidy` with a
no-diff proof), and `go list ./...` at the root still lists no example package,
so the nested-module exclusion holds.

### IN-01: `replayRecord`'s `ErrValueTooLarge` skip was unreachable

**Files modified:** `internal/app/persist/snapshot.go`,
`internal/app/persist/engine_replay_internal_test.go`
**Commit:** `1635a03`
**Applied fix:** the `ErrValueTooLarge`-as-skip branch is gone. `Restore`
decodes with `e.valueMaxBytes`, captured from `store.ValueMaxBytes()`, so the
decoder already excluded every over-limit record (and CR-01 now refuses the
load over one); the store remains the bound's authority and `Set`'s error path
stays for totality. `TestEngine_replayRecordSignal` now proves the store's
`ErrValueTooLarge` surfaces from a direct call instead of reading as a skip,
and its doc records why the decoder makes the state unreachable.

### IN-02: The `S-ALG-CU-11` row cited a line range past the end of `wait.go`

**Files modified:** `docs/SPEC-DEVIATIONS.md`
**Commit:** `d213044`
**Applied fix:** the row now cites `internal/grpcapi/wait.go:72-88`
(`waitForReplication`) and `:93-98` (`writePeers`), the two functions the row
describes, instead of the stale `:94-225`.

### IN-03: The transport documentation overclaimed explicitness

**Files modified:** `docs/API.md`, `internal/grpcapi/limits.go`
**Commit:** `a9090e7`
**Applied fix:** both the doc and the `TransportLimits`/`ServerOptions`
comments now claim only what is pinned — the message-size ceilings and the
keepalive cadence — and say explicitly that the remaining knobs (header-list
size, connection timeout, stream/window sizes, concurrent-stream cap) keep
gRPC's defaults.

### IN-04: No largest-value transport round trip

**Files modified:** `internal/grpcapi/limits_test.go`
**Commit:** `c55c2b8`
**Applied fix:** `TestGRPC_maxValueRoundTrip` writes a
`storage.MaxKeyBytes`-length key with a 64 408-byte value against a server
built with `ValueMaxBytes: 64408` and `LimitsFor(64408)`, asserts `OK`, and
reads the value back through `Get` — the largest record the wire ceiling
admits round-trips end to end.

**Honest scope note (recorded in the test):** the receive envelope now also
admits a maximum List page token (WR-01), so `MaxRecvBytes` is much larger
than one `SetRequest` and a single maximum-size write can no longer
distinguish the effective value limit from the storage default by message
size. That derivation remains pinned by `TestLimitsFor`'s exact arithmetic
(`MaxKeyBytes + effective limit + envelope` per case); the new test keeps the
end-to-end half of research Pitfall 11.

## Verification

**Where the gates ran:** the main checkout (`/home/nil/DiskD/W/Djarvur/sup-kv`),
because `workflow.use_worktrees` is `false`; the numbers are reproducible from
the tree at `a1e8bfd`.

| Gate | Command | Result |
|------|---------|--------|
| Build | `go build ./...` | pass |
| Vet | `go vet ./...` | pass |
| Lint | `make lint` (`config verify` + `run ./...`) | pass |
| Tests | `make test` (`go test -race -cover ./...`) | pass, every package `ok` |
| Proto compatibility | `make buf-breaking` | pass |
| Client example | `make client-example` | pass (tidy no-diff) |
| Red gate | `make red-run` | pass: 7/7 rows, mutation stage green, 0 errors |
| Internal-import probe | throwaway copy of `examples/client` + blank internal import | rejected: `use of internal package ... not allowed` |

No known-accepted flake fired, so no re-run was needed. `go.mod`/`go.sum`,
`.golangci.yml`, `scripts/red-expectations.tsv`, the frozen red tests and the
`.planning/` tree are byte-identical to `fd54a3a`; the CPU governor was not
touched. The only untracked paths are the red-runner's generated evidence
directories.

## Notes for the orchestrator

- **Semantics changed by CR-01:** `SnapshotLoad` of a snapshot with any
  discarded record line now fails `DataLoss` instead of applying a partial
  rollback. This also changes the shrunk-value-limit restore path
  (`TestEngine_configShrunkValueLimits`): `Open` still skips and counts the
  over-limit record, but a `Restore` of that same snapshot refuses. That is
  the mandated "full-rollback deletion only on a cleanly decoded snapshot"
  reading.
- **Transport ceilings changed by WR-01:** `MaxRecvBytes` 63 846 → 187 472 at
  the default limit (66 814 → 190 440 at the datagram ceiling);
  `MaxSendBytes` 6 MiB + 1 382 → 6 MiB + 125 008. Both changes are documented
  in `docs/API.md` and derived from the format bound.
- The `06-REVIEW-DISPOSITION.md` ledger is untouched; it is the human triage
  surface and the orchestrator owns its status column.

---

_Fixed: 2026-11-23T03:16:10Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
