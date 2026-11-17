# Phase 7: v1.0 — Docs, Examples and Release - Pattern Map

**Mapped:** 2026-11-17
**Files analyzed:** 13
**Analogs found:** 10 / 13 (3 are new doc/config artifact types with shape guidance from RESEARCH.md only)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/sup-kv/main.go` (+1 reflection line) | entrypoint wiring | request-response | `cmd/sup-kv/main.go:578-587` (listenGRPC itself) | exact |
| `cmd/sup-kv/readme_test.go` | test (external `main_test` pkg) | process-exec / file-read | `cmd/sup-kv/scenario_test.go` + `cmd/sup-kv/main_test.go` | exact |
| `cmd/sup-kv/examples_test.go` | test (external `main_test` pkg) | process-exec / file-read | `cmd/sup-kv/scenario_test.go` + `main_test.go` | exact |
| `README.md` (rewrite) | doc | — | `docs/IDEA.md` (tone/structure); no executable-doc precedent | partial |
| `docs/OPERATIONS.md` | doc | — | `docs/IDEA.md` §Trust model (content source, not shape) | partial |
| `docs/EXAMPLES.md` | doc | — | none (no grpcurl examples exist) | none |
| `CHANGELOG.md` | doc | — | none (KaC 2.0.0 template from RESEARCH Pattern 8) | none |
| `.goreleaser.yaml` | config (release) | batch/build | RESEARCH Pattern 7 verified shape; Makefile `build` target for ldflags parity | role-match |
| `.github/workflows/release.yml` | config (CI workflow) | event-driven (tag) | `.github/workflows/ci.yml` | exact |
| `Makefile` (+ `release` / `release-snapshot`) | config (build) | batch | existing Makefile targets (`bench`, `build`) | exact |
| `go.mod` / `go.sum` (+2 tool directives) | config | — | existing tool directives (`buf`, `golangci-lint`) | exact |
| `.gitignore` (`/dist/` anchor) | config | — | `.gitignore` line 4 — `dist/` ALREADY ignored (unanchored); only the `/dist/` anchor change is needed | exact |
| Branch-protection ruleset re-apply | ops task (no file) | — | `docs/setup/branch-protection.md` procedure | exact |

All analogs verified git-tracked (repo-root files read directly from the working tree).

## Pattern Assignments

### `cmd/sup-kv/main.go` — reflection registration (the ONLY product-code change)

**Analog:** the file itself, `listenGRPC` at lines 578-587.

```go
// cmd/sup-kv/main.go:578-587 (current)
func listenGRPC(grpcAddr string, srv *grpcapi.Server) (net.Listener, *grpc.Server, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen tcp %s: %w", grpcAddr, err)
	}
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	return listener, gs, nil
}
```

Change: add `import "google.golang.org/grpc/reflection"` and one line after line 585: `reflection.Register(gs)`. Update the function's doc comment (lines 575-577 mention what is registered). Zero new modules — the reflection package ships in the already-required grpc module.

### `cmd/sup-kv/readme_test.go` (test, exec/file-read)

**Analogs:** `cmd/sup-kv/scenario_test.go` (topology, ports, probe/loud-skip) and `cmd/sup-kv/main_test.go` (build-once harness, waitFor).

Join `package main_test`. Consume — do not rebuild — these existing helpers:

**Build-once harness** (`main_test.go:75-123`): `TestMain` builds the binary once via `buildSmokeBinary`; `smokeBinary(t)` loud-skips naming the failed probe:

```go
// main_test.go:113-123
func smokeBinary(t *testing.T) string {
	t.Helper()
	if errSmokeBinary != nil {
		t.Skipf("sup-kv binary smoke unavailable: %v — probed ...", errSmokeBinary, smokeDir)
	}
	return smokeBin
}
```

**Private port window** (`scenario_test.go:96-104`) — the README's fixed doc ports must sit in `[21000, 32678)` (below the ephemeral range on Linux AND darwin) so the parallel `-race` suite can never collide:

```go
// scenario_test.go:101-103
scenarioPortLo    = 21000
scenarioPortHi    = 32678
scenarioPortTries = 32
```

**Platform-capability probe → loud skip** (`scenario_test.go:163-193, 218-249`): `probeLoopbackSend(dstPort)` returns (reached, err); `newExecPair` skips loudly naming the probe when the platform lacks loopback broadcast. This is the exact shape for the macos-15 subnet-broadcast probe (Open Question 1). Note the copy rule: skip messages name what was probed and what still proves the behavior.

**Asymmetric-port topology** (`scenario_test.go:256-260`, `nodeArgs`): node binds P1, broadcasts to `127.255.255.255:P2`, mirror for B, distinct node IDs, explicit `--grpc-addr` per node (default is one shared value, `main.go:131`). The README quickstart's fixed-port variant of exactly this shape.

**New code this file adds** (no analog; from RESEARCH Pattern 2 code examples, already written): `extractMarkedBlocks(t, "../../README.md", "<!-- quickstart-verify -->")` (stdlib line scan, ~60 lines; zero marked blocks = `t.Fatal`) and `runBashVerbatim` (`bash -e`, `cmd.Dir` temp dir, augmented PATH carrying the binary dir + a `grpcurl` shim script that execs `go tool grpcurl "$@"`, combined stdout/stderr in the failure message). Test must read README.md itself — never a copied fixture.

**Discipline to copy:** every test opens with `t.Parallel()` and a multi-line package/file comment stating exactly what it proves (see both analogs' headers, `scenario_test.go:3-23`).

### `cmd/sup-kv/examples_test.go` (test, exec/file-read)

**Analogs:** same as above — `smokeBinary`, `startExecPair`-shaped pair launch on the doc's FIXED ports (contrast: scenarios pick random ports; the examples doc names its ports, so the test binds the same constants), asymmetric topology from `scenario_test.go`, `waitFor` polling from `main_test.go:153-160` with the `hardDeadlineMargin` discipline.

Structure: `TestExamplesDoc` extracts marked grpcurl blocks from `../../docs/EXAMPLES.md` via the same `extractMarkedBlocks`, runs each with `runBashVerbatim` against the already-running pair, and greps captured stdout for expected JSON substrings (`"key":"demo/hello"`, the base64 value, `"acked":true`). Assertion fields must be non-zero-valued (grpcurl omits zero fields — RESEARCH Pitfall 5). Depends on the reflection line landing first.

### `README.md`, `docs/OPERATIONS.md`, `docs/EXAMPLES.md`, `CHANGELOG.md` (docs)

No executable-doc precedent exists. Content sources (all read/trusted):

| Doc | Content source |
|-----|----------------|
| README.md | New quickstart in the RESEARCH "Code Examples → Quickstart block shape" (verbatim-runnable `bash`, background jobs with `NODE_A=$!`, `kill -9 $NODE_A`, cleanup line, fixed ports 279xx); links to the two docs (D-02) |
| docs/OPERATIONS.md | RESEARCH Pattern 6 content map — five mandated statements each tied to a verified source (`docs/IDEA.md` trust model; `main.go:134` `defaultValueMaxBytes = 61440`; REQUIREMENTS Out-of-Scope; `main.go:145` drift-warn default). Trust-model warning FIRST |
| docs/EXAMPLES.md | RESEARCH Pattern 4 — full method paths `supkvapi.v1.SupKVService/{Get,Set,Delete,List,Watch}` + `grpc.health.v1.Health/Check`; proto3 bytes as base64 in `-d` JSON; Watch backgrounded and killed |
| CHANGELOG.md | RESEARCH Pattern 8 Keep-a-Changelog 2.0.0 template verbatim; first release = one curated `Added` section; oldest version links to its tag, not a compare (no tags exist today) |

Docs-tone analog: `docs/IDEA.md` (English, D-03) and existing README headers.

### `.goreleaser.yaml` (config, batch/build)

No goreleaser config exists. Shape is fully specified in RESEARCH Pattern 7 (verified against goreleaser docs): `version: 2`, build `main: ./cmd/sup-kv`, `env: [CGO_ENABLED=0]`, `goos: [linux, darwin]`, `goarch: [amd64, arm64]`, `checksum`, `sboms` (artifacts: archive), `release.header.from_file: CHANGELOG.md`.

**ldflags parity analog** — Makefile lines 36-38:

```make
build:
	go build ./...
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/sup-kv ./cmd/sup-kv
```

goreleaser's ldflags `-X main.version={{ .Version }} -X main.commit={{ .ShortCommit }}` must be one conscious decision vs the Makefile's `git describe` form (RESEARCH Pitfall 6 — `{{ .Version }}` drops the `v`, planner picks deliberately). Verify with `make release-snapshot` before any tag.

### `.github/workflows/release.yml` (config, event-driven on tag)

**Analog:** `.github/workflows/ci.yml`. Copy its conventions:

```yaml
# ci.yml:11-12, 23-28
permissions:
  contents: read
...
- uses: actions/checkout@v7
  with:
    fetch-depth: 0
- uses: actions/setup-go@v7
  with:
    go-version-file: go.mod
```

Also copy: top comment block explaining check-name contracts, `timeout-minutes` on every job (ci.yml:21, 57 — P1-WR-01 rule), tool-directive invocation (`go tool ...`, never bare `go install`). Release workflow per RESEARCH Pattern 7: `on: push: tags: ["v*"]`, `permissions: contents: write`, `anchore/sbom-action/download-syft@v0.24.3`, `go tool goreleaser release --clean` with `GITHUB_TOKEN`. CRITICAL divergence from the analog: do NOT touch ci.yml's jobs — check-run names are a ruleset contract (RESEARCH Pitfall 4).

### `Makefile` (+ `release` / `release-snapshot`)

**Analog:** existing targets in `Makefile` — one `## name: rationale` comment block per target, `.PHONY` listing at line 1, `go tool <pinned-tool>` invocation (lines 11-13, 47-49). New targets:

```make
release:
	go tool goreleaser release --clean
release-snapshot:
	go tool goreleaser release --snapshot --clean
```

Add both names to the line-1 `.PHONY` list.

### `go.mod` (tool directives)

**Analog:** existing tool directives (buf, golangci-lint — invoked as `go tool buf` / `go tool golangci-lint` throughout Makefile and ci.yml). Add via `go get -tool github.com/goreleaser/goreleaser/v2@v2.20.2` and `go get -tool github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.4`; the tidy gate (`go mod tidy && git diff --exit-code go.mod go.sum`, ci.yml:84-85) must stay clean.

### `.gitignore`

**Already partially done:** line 4 is `dist/` (unanchored — also ignores any nested `dist/`). The plan task reduces to optionally anchoring it to `/dist/`; verify with `git check-ignore dist/` before claiming work.

### Branch-protection ruleset re-apply (ops task, no file)

**Analog:** `docs/setup/branch-protection.md` procedure (one `gh api` POST + read-side verify). Acceptance = honest attempt + verify read + marker update either way (may still 403 on org plan — RESEARCH Pitfall 10).

## Shared Patterns

### External `main_test` package discipline
**Source:** `cmd/sup-kv/main_test.go`, `cmd/sup-kv/scenario_test.go`
**Apply to:** both new test files. `package main_test`, `t.Parallel()` + `t.Helper()` in helpers, multi-line header comment stating what is proven and what stays proven elsewhere, loud skips naming the exact probe.

### Loud-skip / capability probe
**Source:** `cmd/sup-kv/scenario_test.go:163-193` (`probeLoopbackSend`) and `:218-249` (`newExecPair` skip)
**Apply to:** the macos-15 leg of both doc tests (Open Question 1 outcome decides run-for-real vs loud skip vs amended claim).

### Tool pinning
**Source:** `Makefile` + `.github/workflows/ci.yml` (every tool via `go tool`)
**Apply to:** goreleaser, grpcurl, syft-in-workflow. Never bare `go install`, no action-downloaded goreleaser binary.

### Fixed ports below the ephemeral range
**Source:** `scenario_test.go:96-104` comment
**Apply to:** README quickstart and EXAMPLES.md port constants (both UDP and gRPC TCP), e.g. 27901/27902/27911/27912.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `docs/EXAMPLES.md` | doc | — | No grpcurl examples exist; content fully specified by RESEARCH Pattern 4/5 (method paths, base64 semantics, zero-value omission) |
| `CHANGELOG.md` | doc | — | First changelog; use KaC 2.0.0 template from RESEARCH Pattern 8 verbatim |
| `README.md` quickstart (executable doc) | doc | — | No executable-doc precedent; shape specified by RESEARCH Pattern 2 + quickstart block example (marker `<!-- quickstart-verify -->`, verbatim bash, background jobs, cleanup) |

## Metadata

**Analog search scope:** `cmd/sup-kv/`, `Makefile`, `.github/workflows/`, `docs/`, repo root
**Files scanned:** 8 analog candidates, 6 read in full/partial this session (plus RESEARCH.md's verified in-repo citations for `main.go` flags/defaults)
**Pattern extraction date:** 2026-11-17
