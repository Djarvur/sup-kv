---
phase: 261108-vxx-migratsiya-proto-codegen-na-buf-lint-bre
plan: 01
subsystem: tooling
tags: [buf, protobuf, codegen, ci, lint, breaking-change-detection]
requires: [go 1.24+ tool directives, go.mod tool block, phase-4 replication proto]
provides:
  - "buf v1.75.0 pinned as go.mod tool directive github.com/bufbuild/buf/cmd/buf"
  - "buf.yaml v2 root module (STANDARD lint + one documented PACKAGE_DIRECTORY_MATCH exception, FILE breaking, .gomodcache excludes)"
  - "buf.gen.yaml v2 with local go-tool protoc-gen-go plugin, source-relative"
  - "Makefile targets: proto (buf generate), buf-lint, buf-format, buf-breaking (guarded)"
  - "CI lint job buf schema gates (lint + format + guarded breaking, fetch-depth: 0)"
affects: [phase-5 gRPC schema (internal/grpcapi/apiproto rides the same root module with zero config change)]
tech-stack:
  added:
    - "github.com/bufbuild/buf/cmd/buf v1.75.0 (tool directive)"
  patterns:
    - "protobuf runtime MVS-bumped to v1.36.13-0.20261016172727-e9ee2a76ab8e through buf's module graph — no replace, no downgrade; plugin and runtime stay one identical module (protoc-gen-go reports v1.36.12-devel)"
key-files:
  created: [buf.yaml, buf.gen.yaml]
  modified: [Makefile, go.mod, go.sum, internal/udprepl/replproto/supkv.pb.go, .github/workflows/ci.yml, .claude/CLAUDE.md]
decisions:
  - "D-01..D-09 implemented as planned; two probe-backed wordings recorded: STANDARD implements D-02/D-06's DEFAULT (identical rule lists; DEFAULT spelling deprecation-warns), and the origin/master guard implements D-08's clean pass (buf v1.75 errors exit 1 on a proto-less against-ref)"
  - "D-09 mapping recorded: the buf-generate goal is delivered by the ported proto target (recipe is exactly `go tool buf generate`); no duplicate buf-generate alias target — make proto is the single established regeneration entrypoint"
metrics:
  duration: 5min
  completed: 2026-11-08
status: complete
actuals:
  tokens: 16977 # chars/4 over the realized diff (67906 chars, 8 files, +354/-32)
  tasks: 2
  commits: 2
  plan_head_before: 23afa7d5e1b3afae36b837cd5350a18693cfe790
  plan_head_after: 9727c80a66c7228a605406a334b472ef391c9e1f
---

# Quick Task 261108-vxx Summary: proto codegen migrated to buf

Proto codegen now runs through buf v1.75.0 pinned as a go.mod tool directive — `go tool buf generate` with the go-tool protoc-gen-go plugin replaced the system protoc, and buf lint (STANDARD/DEFAULT rules), buf format and guarded FILE-policy buf breaking vs origin/master now gate the replication wire schema locally (Makefile) and in the CI lint job.

## Tasks Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Pin buf v1.75.0 as go.mod tool, land buf.yaml + buf.gen.yaml, port Makefile, regenerate supkv.pb.go, correct .claude/CLAUDE.md | ed880c7 | go.mod, go.sum, buf.yaml, buf.gen.yaml, Makefile, internal/udprepl/replproto/supkv.pb.go, .claude/CLAUDE.md |
| 2 | Wire buf lint + format + breaking into the CI lint job with the origin/master guard | 9727c80 | .github/workflows/ci.yml |

## Verification Results

All verify chains green; no gate weakened.

**Task 1 automated chain** (full chain passed):
- `go tool buf lint` exit 0 (STANDARD rules; the one PACKAGE_DIRECTORY_MATCH exception honored; .gomodcache excluded)
- `go tool buf format --diff --exit-code` exit 0 (committed proto format-clean)
- `go tool protoc-gen-go --version` = `protoc-gen-go v1.36.12-devel` (exact assert)
- go.mod carries `github.com/bufbuild/buf/cmd/buf` tool directive and `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e`
- `go build ./...` green; `go test -race ./internal/udprepl/...` green
- Regenerated-file diff gate = exactly 4 changed diff lines (the two header version lines: `protoc-gen-go v1.36.12-devel`, `protoc (unknown)`); second `make proto` byte-stable
- `go mod tidy` idempotent (sha256 re-check OK on go.mod and go.sum) — CI tidy gate stays green
- Stale-doc negative greps clean file-wide: "pinned by release tag", "install pinned from GitHub releases", "bufbuild/buf/releases" all absent from .claude/CLAUDE.md

**Task 2 automated chain** (full chain passed):
- Exact CI step block replayed locally against freshly fetched origin/master: buf lint exit 0, format exit 0, breaking took the loud defer branch (`buf breaking deferred: supkv.proto not on origin/master yet (D-08 guard)`)
- ci.yml grep gates: exactly 3 non-comment `go tool buf` occurrences, exactly 1 `fetch-depth: 0` (lint job checkout; gate job stays shallow)
- Makefile grep gate: 0 leftover `^[[:space:]]*protoc ` recipe lines (baseline 1 at HEAD before Task 1)
- Full parity gate: `make lint && make test` green (strict golangci-lint + race tests, whole tree)

**Makefile target behavior** spot-checked: `make buf-lint` exit 0; `make buf-format` no-op on the format-clean proto; `make buf-breaking` fetches origin master then defers loudly, exit 0.

## Decision Coverage (D-01..D-09)

- D-01: proto regeneration is `go tool buf generate` (Makefile proto target).
- D-02: DEFAULT rule set implemented as `use: [STANDARD]` (identical rule lists probe-verified; avoids the DEFAULT deprecation WARN) + the one documented file-scoped PACKAGE_DIRECTORY_MATCH exception.
- D-03: `breaking: use: [FILE]` vs origin/master, enforced in CI.
- D-04: `buf format --diff --exit-code` in CI; write form (`buf format -w`) local-only in make buf-format.
- D-05: buf pinned via `go get -tool github.com/bufbuild/buf/cmd/buf@v1.75.0`; buf.gen.yaml plugin `local: ["go", "tool", "protoc-gen-go"]` — no PATH install, no BSR remote plugins.
- D-06: root buf.yaml, one module (path "."), excludes .gomodcache; documented exception over moving the proto.
- D-07: regeneration byte-identical except exactly the two header version lines; build + race tests green.
- D-08: one against-ref form `.git#ref=refs/remotes/origin/master` in both Makefile and ci.yml (byte-identical); proven locally; guard documented in both places.
- D-09: buf-lint/buf-format/buf-breaking as new targets; lint/test/generate/build byte-unchanged; buf-generate delivered by the ported proto target per the recorded mapping (no alias).

## Deviations from Plan

None — plan executed exactly as written. The two deliberately re-worded locked-decision implementations (STANDARD for DEFAULT; the origin/master guard for D-08's "passes trivially") are in-plan items (verification item 4), both probe-evidence-backed and neither scope-reducing.

## Supply-Chain Notes (QT-SC)

Exact version pin `@v1.75.0` used in the single `go get -tool` invocation; go.sum covers the whole buf graph; no replace directives added (gomoddirectives gate still green via make lint). CI buf step runs the same go.mod-pinned binary as local (parity, no second install path). Threat model rows QT-01..QT-03 implemented as planned (object-path-precise guard with loud deferral; local write-only format target; .git-dir against-input with no token plumbing, permissions unchanged).

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: buf.yaml, buf.gen.yaml, Makefile (buf targets), regenerated supkv.pb.go, ci.yml buf step — all on disk
- FOUND: commit ed880c7 (ancestor of HEAD), commit 9727c80 (HEAD)
- Both task verify chains re-ran green after all edits; measured commits from ledger: 2 (23afa7d..9727c80)
- Working tree clean except the two foreign untracked planning files (.planning/phases/04-replication-and-deterministic-simulator/04-REVIEW-FIX.md, .planning/state.json) — left untouched per instructions
