---
phase: 06-api-and-semantics
plan: 01
subsystem: public-api
tags: [apis-02, apis-06, public-proto, api-supkvapi-v1, keyset-pagination, page-size, page-token, next-page-token, alias-package, deprecated-alias, buf-module, buf-breaking, file-same-go-package, depguard-boundary, tdd-red-evidence, status-complete]
requires:
  - phase: 05-persistence
    provides: the frozen red evidence (R-03/P-01 stand tests) whose imports must keep compiling byte-unchanged
  - phase: 04-protocol
    provides: the generated-package shape (`apiproto` alias discipline) and the buf module-per-tree workspace
provides:
  - "the client-facing wire schema at github.com/Djarvur/sup-kv/api/supkvapi/v1 (package supkvapi), outside internal/, regenerated through `go tool buf generate api`"
  - "the keyset-pagination wire fields: ListRequest.page_size=2 / page_token=3 and ListResponse.next_page_token=2 / records=3, with keys=1 retained"
  - "a deprecated alias package at the legacy generated-package path that re-exports every generated identifier, so the frozen tests compile with zero edits"
  - "a machine-enforced public-package boundary: sup-kv-public-api-no-internal denies github.com/Djarvur/sup-kv/internal under api/"
  - "a one-file, commented FILE_SAME_GO_PACKAGE breaking waiver proven load-bearing against the pre-move tree"
affects: [06-05-list-handler, 06-03-client-example, 06-13-phase-gate, phase-9-docs]
actuals:
  tokens: 16396
  tasks: 3
  commits: 2
  plan_head_before: 2b366888c0cae3eb110fc85f091111ddcce8c665
  plan_head_after: 4cb5d4b2396602c76bcd8258561bada70a2dae07
tech-stack:
  added: []
  patterns:
    - "A public-code move keeps a pure alias package at the deprecated path: type aliases for generated types, `var` aliases for generated funcs/vars, so the old import path exposes the identical types and no importer identifier changes."
    - "Generated wire code is pinned by a descriptor test in the package, not by a byte diff: the move necessarily rewrote `go_package`, the package clause and the source comment, while the wire schema stayed identical."
    - "The alias-dir package/doc names are historical (`apiproto`, generated underscore identifiers), so each lint objection carries a scoped `//nolint:revive` with a written reason; the package deliberately omits the machine-readable `Deprecated:` marker because it would make staticcheck fail the frozen imports."
key-files:
  created:
    - api/supkvapi/v1/supkvapi.proto
    - api/supkvapi/v1/supkvapi.pb.go
    - api/supkvapi/v1/supkvapi_grpc.pb.go
    - api/supkvapi/v1/doc.go
    - api/supkvapi/v1/schema_test.go
    - internal/grpcapi/apiproto/supkvapi/v1/doc.go
    - internal/grpcapi/apiproto/supkvapi/v1/alias.go
    - .planning/phases/06-api-and-semantics/red-evidence/06-01-task1.json
  modified:
    - buf.yaml
    - buf.gen.api.yaml
    - Makefile
    - .golangci.yml
    - 17 importer files (cmd/sup-kv/main.go, cmd/sup-kv/main_test.go, 11 internal/grpcapi/*.go, internal/stand/{stand.go,persist_test.go,stand_buffers_test.go,stand_delete_test.go,stand_metrics_test.go,stand_test.go})
  deleted:
    - internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto (moved to the public module root)
key-decisions:
  - "The RED evidence is the compile-clean assertion failure, not the package-missing build error the plan anticipated: the go-test RED classifier refuses `[build failed]` runs, so the moved proto was generated first without the page fields and TestSchema_listPageFields was recorded failing on the six missing-field assertions (RED_EVIDENCE_OK, 4 tests, 1 target failure)."
  - "Tasks 1 and 2 landed as one atomic commit, per the plan: the tree never commits a state that imports the deprecated alias everywhere, and the earlier first commit was amended (`--amend`) to carry the lint-safe alias files, so each commit is green on its own."
  - "The alias package keeps `package apiproto` (its historical name) rather than renaming to match the directory: four scoped `//nolint:revive` directives with written reasons cover package-directory-mismatch and the generated underscore identifiers — never an unnamed or blanket suppression."
  - "The alias package's doc comment states the deprecation in prose and omits the Go `Deprecated:` marker: the marker makes staticcheck report SA1019 at the frozen imports, and those files may not be edited. The pointer to the public path and the removal intent stay in the doc."
  - "The FILE_SAME_GO_PACKAGE waiver was proven load-bearing: the same comparison with a waiver-free copy of buf.yaml names exactly api/supkvapi/v1/supkvapi.proto as the go_package change, and the committed narrow waiver makes it exit 0."
  - "The stale `.git/gsd-plan-head-before-06-01` ledger (dated 2026-11-16, commit 45208b8) was reset to the current HEAD before the first commit so the SUMMARY's `commits:` count measures this plan and not three weeks of planning commits."
patterns-established:
  - "A public schema move is proven by identity of the exported identifier set: the alias package must expose every type, func and var the generated package did, or the frozen importers stop compiling."
  - "A depguard boundary is verified by probe, not by config presence: a temporary internal import under api/ must be rejected by rule name before the rule is called proven."
requirements-completed: [APIS-02, APIS-06]
coverage:
  - id: D1
    description: "The client-facing proto and generated Go live at api/supkvapi/v1 (package supkvapi) with the old schema tree gone, and `go build ./...` is green."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "test -f api/supkvapi/v1/{supkvapi.proto,supkvapi.pb.go,supkvapi_grpc.pb.go,doc.go} && test ! -e internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto -> exit 0"
        status: pass
      - kind: process
        ref: "grep -q 'option go_package = \"github.com/Djarvur/sup-kv/api/supkvapi/v1;supkvapi\";' api/supkvapi/v1/supkvapi.proto -> exit 0"
        status: pass
      - kind: process
        ref: "go build ./... -> exit 0; GOOS=darwin GOARCH=amd64 go build ./... -> exit 0; GOOS=darwin GOARCH=arm64 go build ./... -> exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "The wire descriptor is pinned: package supkvapi.v1, service SupKVService with exactly Get, Set, Delete, List, Watch (the only server-streaming method), SnapshotCreate, SnapshotLoad; Record and every non-List message keep their field numbers, kinds and cardinalities."
    requirement: "APIS-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./api/supkvapi/v1/ -run '^TestSchema_' -> ok (TestSchema_serviceDescriptor, TestSchema_recordFields, TestSchema_otherMessages pass)"
        status: pass
    human_judgment: false
  - id: D3
    description: "ListRequest carries prefix=1, page_size=2 (int32), page_token=3 (string); ListResponse carries keys=1 (repeated string), next_page_token=2 (string), records=3 (repeated supkvapi.v1.Record); the proto comments document the dual mode, the 256 cap and the token-needs-page-size rule."
    requirement: "APIS-02"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./api/supkvapi/v1/ -run '^TestSchema_listPageFields' -> PASS (the same test failed before the GREEN step; evidence in red-evidence/06-01-task1.json, RED_EVIDENCE_OK)"
        status: pass
      - kind: process
        ref: "grep -q 'int32 page_size = 2;' && grep -q 'string page_token = 3;' && grep -q 'string next_page_token = 2;' && grep -q 'repeated Record records = 3;' api/supkvapi/v1/supkvapi.proto -> exit 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "The legacy path is a pure alias package exposing every generated identifier, so both frozen red tests compile and their bytes are unchanged."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "git diff HEAD~2 --exit-code -- internal/grpcapi/red_gate_test.go internal/stand/stand_red_test.go -> exit 0 (identical to the pre-plan base)"
        status: pass
      - kind: process
        ref: "go vet ./... -> exit 0 (type-checks both frozen test packages through the alias)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every real importer (17 files) uses the public path with the apiproto alias kept; only the two frozen tests still name the legacy package."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "grep -rl 'internal/grpcapi/apiproto/supkvapi/v1' --include='*.go' cmd internal | grep -v -e 'red_gate_test.go' -e 'stand_red_test.go' | wc -l -> 0"
        status: pass
      - kind: process
        ref: "grep -q 'apiproto \"github.com/Djarvur/sup-kv/api/supkvapi/v1\"' internal/grpcapi/server.go -> exit 0"
        status: pass
    human_judgment: false
  - id: D6
    description: "buf lint, buf format and the CI-equivalent make buf-breaking are green with exactly one commented FILE_SAME_GO_PACKAGE waiver, and the waiver is load-bearing."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "go tool buf lint && go tool buf format --diff --exit-code && make buf-breaking -> exit 0 (origin/master comparison)"
        status: pass
      - kind: process
        ref: "go tool buf breaking --against '.git#ref=HEAD' --config <waiver-free copy> -> names exactly api/supkvapi/v1/supkvapi.proto: File option \"go_package\" changed"
        status: pass
      - kind: process
        ref: "go tool buf breaking --against '.git#ref=HEAD' -> exit 0 (pre-commit, with the committed waiver)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The public package is machine-barred from importing internal/: config verify and the focused lint are green, and a probe import is rejected by rule name."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "go tool golangci-lint config verify && go tool golangci-lint run ./api/... ./internal/grpcapi/... -> exit 0"
        status: pass
      - kind: process
        ref: "temporary api/supkvapi/v1/probe_boundary.go importing internal/app/version -> golangci-lint reports \"not allowed from list 'sup-kv-public-api-no-internal'\"; probe deleted, tree clean"
        status: pass
    human_judgment: false
  - id: D8
    description: "The project gates are green on the committed tree: full strict lint and the full race suite."
    verification:
      - kind: process
        ref: "go tool golangci-lint run (repo-wide, the make lint command) -> exit 0; go tool golangci-lint fmt --diff ./api/... ./internal/grpcapi/... -> exit 0"
        status: pass
      - kind: integration
        ref: "make test -> exit 0 (api/supkvapi/v1 ok 1.0 s; grpcapi ok 2.2 s; stand ok 7.3 s; app/repl ok 235.7 s; every package ok)"
        status: pass
    human_judgment: false
duration: 12min
completed: 2026-11-22
status: complete
---

# Phase 06 Plan 01: Public API Move and List Page Fields Summary

**The client wire schema now lives at the public `api/supkvapi/v1` package with `List` pagination fields on the wire, a deprecated alias package keeps both frozen red tests compiling byte-unchanged, and buf (lint/format/breaking) plus the strict lint gate — including a machine-enforced no-internal boundary — are green.**

## Performance

- **Duration:** 12min
- **Started:** 2026-11-22T22:00:08Z
- **Completed:** 2026-11-22T22:13:00Z
- **Tasks:** 3 (Tasks 1 and 2 share one atomic commit, as the plan requires)
- **Files modified:** 30 paths in the plan diff (7 created, 22 modified, 1 moved away)

## Accomplishments

- **The public move is complete and generated, not hand-copied.** `api/supkvapi/v1/{supkvapi.proto,supkvapi.pb.go,supkvapi_grpc.pb.go,doc.go,schema_test.go}` carry the schema and its codegen output; `internal/grpcapi/apiproto/supkvapi/v1/` no longer holds a proto or generated code, only the alias package. Codegen ran through the pinned toolchain: `go tool buf generate api --template buf.gen.api.yaml` (buf 1.75.0, `protoc-gen-go v1.36.12-devel`, `protoc-gen-go-grpc v1.6.2`).
- **The `List` page contract is on the wire.** `ListRequest` gained `page_size=2` (`int32`) and `page_token=3` (`string`); `ListResponse` gained `next_page_token=2` (`string`) and `records=3` (`repeated Record`), with `keys=1` retained. The proto comments state the dual mode (unset/0 = legacy all-keys keys-only listing, positive = bounded records page), the 256 cap, the opaque token and the page-token-requires-page-size rule.
- **The frozen red tests did not change by one byte.** `git diff HEAD~2 --exit-code -- internal/grpcapi/red_gate_test.go internal/stand/stand_red_test.go` exits 0 against the pre-plan base (2b36688), while `go vet ./...` proves both still type-check through the alias package.
- **The descriptor is pinned, not byte-compared.** The move necessarily rewrote `go_package`, the generated package clause and the leading source comment, so `TestSchema_*` pins what a client actually sees: package `supkvapi.v1`, the seven RPCs with `Watch` as the only server-streaming method, `Record` = key/version/value/deleted, the four `List` page fields, and every other message's field numbers, kinds and cardinalities.
- **All 17 real importers point at the public path** with the `apiproto` alias retained, so no identifier changed anywhere; the legacy-path grep over `cmd` + `internal` (excluding the two frozen files) returns 0.
- **buf is green with one narrow, commented waiver.** `buf lint`, `buf format --diff --exit-code` and `make buf-breaking` (the `refs/remotes/origin/master` comparison the CI job runs) all exit 0. The `FILE_SAME_GO_PACKAGE` waiver is scoped to exactly `api/supkvapi/v1/supkvapi.proto` and was proven load-bearing: a waiver-free copy reports the `go_package` change on that file.
- **The public package cannot grow an internal dependency.** `sup-kv-public-api-no-internal` denies `github.com/Djarvur/sup-kv/internal` for non-test files under `**/api/**`; a temporary probe import was rejected by rule name and then removed.
- **Project gates green on the committed tree:** repo-wide `golangci-lint run` exit 0, `make test` exit 0 (`api/supkvapi/v1` ok 1.0 s), and both darwin cross-builds exit 0.

## Task Commits

1. **Task 1 + Task 2: public schema move, page fields, alias package and import sweep** — `6f0d1a9` (feat): the proto move, regenerated code, `doc.go`, the descriptor test, the alias package, the buf/Makefile rewiring, the RED evidence and all 17 importers, in one atomic commit. (Amended once to fold in the lint-safe alias-doc/nolint adjustments; the plan's "if the harness already committed Task 1 separately, squash it" applies.)
2. **Task 3: public-package import boundary and the final buf gate** — `4cb5d4b` (chore): the `sup-kv-public-api-no-internal` depguard rule.

**Plan metadata:** the `docs(06-01)` completion commit (this commit).

## Files Created/Modified

- `api/supkvapi/v1/supkvapi.proto` (created) — the moved schema with the public `go_package`, consumer-facing comments and the four `List` page fields.
- `api/supkvapi/v1/supkvapi.pb.go`, `api/supkvapi/v1/supkvapi_grpc.pb.go` (created, generated) — `package supkvapi`, `source: supkvapi/v1/supkvapi.proto`.
- `api/supkvapi/v1/doc.go` (created) — the public contract: seven RPCs, full-record shape, reproducible codegen via `make proto`, and the never-import-internal rule.
- `api/supkvapi/v1/schema_test.go` (created) — `package supkvapi_test`; four `TestSchema_*` descriptor pins, all identifiers qualified through the import.
- `internal/grpcapi/apiproto/supkvapi/v1/doc.go`, `alias.go` (created) — the deprecated alias package: prose deprecation pointing at the public path, plus type aliases for the 15 messages and 6 service types and `var` aliases for `File_supkvapi_v1_supkvapi_proto`, `NewSupKVServiceClient`, `RegisterSupKVServiceServer`, `SupKVService_ServiceDesc`.
- `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` (deleted) — moved to the public tree; its two generated files moved with it.
- `buf.yaml` (modified) — the second module is now rooted at `api`; the comment explains the public/internal split; `breaking.ignore_only.FILE_SAME_GO_PACKAGE` waives exactly the public file.
- `buf.gen.api.yaml` (modified) — both plugins emit into `api`.
- `Makefile` (modified) — `proto:` runs `go tool buf generate api --template buf.gen.api.yaml`.
- `.golangci.yml` (modified) — additive `sup-kv-public-api-no-internal` depguard rule after the existing project rules; the upstream block is untouched.
- 17 importer files (modified) — one import line each: `apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"`.
- `.planning/phases/06-api-and-semantics/red-evidence/06-01-task1.json` (created) — the immutable RED record (command, exit 1, full console output, target test, expected/actual, semantic assessment).

## Decisions Made

- **The RED record is the assertion failure, not the missing-package build error.** The plan expected a build error while `api/supkvapi/v1` did not exist; the `go test` RED classifier refuses a `[build failed]` run outright (`build_failure`). Following the TDD reference ("make the test compile and fail on its assertion, then record the run again"), the test was written first, the moved proto was generated without the page fields, and `TestSchema_listPageFields` was recorded failing on the six missing-field assertions — `RED_EVIDENCE_OK`, 4 tests, 3 pass, 1 target failure — before the GREEN step added the fields. The build-failure attempt is preserved in the record's `semanticAssessment`, not offered as evidence.
- **One atomic commit for Tasks 1 and 2 (then amended).** The plan forbids committing the move in isolation because every importer would otherwise commit a tree that names the deprecated alias. The first commit was later amended to carry the lint-safe alias files, so the commit that introduces them is green on its own rather than depending on Task 3.
- **The alias package keeps `package apiproto` and carries scoped nolints.** Renaming the package to match its directory would change the historical package name at the legacy path; instead four `//nolint:revive` directives with written reasons cover `package-directory-mismatch` and the generated underscore identifiers (`SupKVService_WatchClient/Server`, `File_supkvapi_v1_supkvapi_proto`, `SupKVService_ServiceDesc`), which cannot be renamed without breaking the re-export.
- **No machine-readable `Deprecated:` marker on the alias package.** The marker makes staticcheck report SA1019 at the frozen tests' import lines, and those files are byte-frozen. The deprecation is stated in prose in `doc.go`, with the public path and the removal intent; the `//nolint` reason names the tradeoff.
- **`var` aliases for generated funcs, `type` aliases for generated types.** This is the plan's shape and it keeps the alias package signature-free: `apiproto.NewSupKVServiceClient` is the very same function value, and `SupKVServiceClient` is the very same interface type, so the frozen tests' `[]apiproto.SupKVServiceClient` is identical to the public package's slice element type.
- **The stale plan-head ledger was corrected.** `.git/gsd-plan-head-before-06-01` predated this run (2026-11-16) and pointed at a planning commit; measuring `commits:` from it would have counted weeks of unrelated history. It was reset to the pre-plan HEAD (2b36688) before the first commit, and the SUMMARY's `commits: 2` is the measured `git rev-list --count`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The alias/doc files tripped the vendored strict lint config**
- **Found during:** Task 3's gate (`go tool golangci-lint run ./api/... ./internal/grpcapi/...`)
- **Issue:** `revive` reported `package-directory-mismatch` on both hand-written files in the alias directory (package `apiproto` in dir `v1`/`supkvapi`) and `var-naming` on the four generated identifiers containing underscores. The generated `.pb.go` files were exempt only because the config excludes `*.pb.go` by path.
- **Fix:** four line-scoped `//nolint:revive` directives, each with a written reason naming the generated/historical name being preserved; no blanket or unnamed suppression. AGENTS.md deviation 3 permits exactly this shape.
- **Files modified:** `internal/grpcapi/apiproto/supkvapi/v1/alias.go`, `internal/grpcapi/apiproto/supkvapi/v1/doc.go`
- **Verification:** focused lint exit 0; repo-wide `golangci-lint run` exit 0.

**2. [Rule 3 - Blocking] The `Deprecated:` marker made the frozen tests fail staticcheck**
- **Found during:** the same lint run
- **Issue:** `SA1019` fired at `internal/grpcapi/red_gate_test.go:16` — importing the deprecation-marked alias package. The frozen files are byte-frozen by plan prohibition R-03/P-01, so neither a `//nolint` nor an import edit is available; a config-level exclusion for the frozen files would have violated Task 3's "no other waiver" instruction.
- **Fix:** the package doc keeps the deprecation in prose (with the public path and the removal intent) but omits the Go `Deprecated:` marker, and says why. The deprecation is still visible to humans and to `godoclint`; only the staticcheck trigger is removed.
- **Files modified:** `internal/grpcapi/apiproto/supkvapi/v1/doc.go`
- **Verification:** repo-wide lint exit 0 with both frozen files still importing the legacy path; `git diff HEAD~2 --exit-code` on the frozen files exits 0.

**3. [Rule 3 - Blocking] The plan's RED assumption is rejected by the RED classifier**
- **Found during:** Task 1's RED step
- **Issue:** the plan says the test "fails with a build error while `api/supkvapi/v1` does not exist", but a `[build failed]` run is `INVALID_RED` (`build_failure`) — the run asserted nothing.
- **Fix:** per the TDD reference, the minimal moved schema was generated first (no page fields) so the test compiles, then the assertion failure was recorded as the RED evidence and validated as `RED_EVIDENCE_OK`; the page fields were the GREEN step.
- **Files modified:** `api/supkvapi/v1/supkvapi.proto` (generated twice: pre-fields, then with fields), `.planning/phases/06-api-and-semantics/red-evidence/06-01-task1.json`
- **Verification:** `gsd_run check tdd-red-evidence ... --raw` → `RED_EVIDENCE_OK`, reason `target_test_failed`.

**4. [Rule 1 - Bug] Commit-amend to keep the first commit lint-clean**
- **Found during:** Task 3
- **Issue:** the alias nolint/doc adjustments were made after the Tasks 1+2 commit; leaving them in the Task 3 commit would make the public-move commit fail lint if checked out alone.
- **Fix:** `git commit --amend --no-edit` on the unpushed move commit (the plan explicitly endorses squashing Task 1 into the shared commit), then the Task 3 commit carries only `.golangci.yml`.
- **Verification:** `git log --oneline` shows two commits over 2b36688; the tree at `6f0d1a9` lints clean (the nolints and prose deprecation are in it).

### Notes

- No plan task was skipped, reordered, or expanded; no file outside the plan's list was authored. `docs/SPEC-DEVIATIONS.md`, `docs/EXAMPLES.md` and `AGENTS.md` still name the old proto import path in prose — the plan's file list excludes them, so they are left for the docs-owning plan (Phase 9 / 06-13's ripple), not silently edited here.

## Issues Encountered

- **Transient port collision, not a regression.** Running a hand-picked subset (`go test -race ./api/supkvapi/v1/ ./internal/grpcapi/ ./internal/stand/ ./cmd/...`) failed once in `cmd/sup-kv`'s `TestLog_formatText` with `bind: address already in use` (UDP probe port), because `internal/stand` binds fixed test ports while packages run in parallel. The package passes standalone (`ok 5.868s`) and the project gate `make test` — which runs the same packages — exits 0. Recorded here rather than fixed: it is a pre-existing cross-package port contention flake in an untouched test, outside this plan's scope.
- **buf 1.75 prints nothing on success and exit 0** for `buf breaking --against '.git#ref=HEAD'`; the waiver's load-bearing property was therefore verified with a deliberate waiver-free copy rather than inferred from the silent pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 06-02.** The public package compiles, is descriptor-pinned, and is the only schema location; the alias package is in the tree and stays until a later plan removes it.
- **The wire contract is additive and the server ignores it for now.** `page_size`/`page_token` are accepted on the wire but the current `List` handler ignores them and keeps the legacy all-keys response; plan 06-05 owns the handler, the constants (256 max, 6 MiB page budget) and the ordering/token semantics. Clients built against this schema see the legacy behavior until then — that is the plan's explicit split, not a stub.
- **Frozen evidence untouched:** `internal/grpcapi/red_gate_test.go`, `internal/stand/stand_red_test.go` and `scripts/red-expectations.tsv` are byte-identical to the pre-plan base; plan 06-13 owns the hash gate.

## Self-Check: PASSED

- Created files exist: `api/supkvapi/v1/{supkvapi.proto,supkvapi.pb.go,supkvapi_grpc.pb.go,doc.go,schema_test.go}`, `internal/grpcapi/apiproto/supkvapi/v1/{doc.go,alias.go}`, `.planning/phases/06-api-and-semantics/red-evidence/06-01-task1.json`, this SUMMARY — all FOUND.
- Task commits are ancestors of HEAD: `6f0d1a9`, `4cb5d4b` — both FOUND.
