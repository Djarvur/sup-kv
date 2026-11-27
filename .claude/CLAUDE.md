<!-- GSD:project-start source:PROJECT.md -->

## Project

**sup-kv**

**sup-kv** ("supremum storage") is a leaderless replicated in-memory key-value store for small clusters — two to a handful of nodes on one trusted LAN broadcast segment. Every node accepts reads and writes locally and independently (AP); replicas exchange changes over UDP and converge to identical state — the pointwise supremum of all writes under the record-version order (strong eventual consistency via a Last-Write-Wins-Element-Map CRDT). It ships as one static Go binary with minimal dependencies and human-readable JSON persistence.

For service builders on small HA clusters (e.g. two management boards in one appliance) who need a shared, always-writable store for configuration and durable service state where quorum-based stores (etcd, Consul) cannot survive the loss of a single node.

**Core Value:** Every node stays writable at all times with no coordination, and a write confirmed to a client is never lost.

### Constraints

- **Tech stack**: Go; single static binary; minimal dependencies — gRPC + protobuf only; HLC, UUIDv7 packing, LWW map, UDP reliability layer all implemented in-repo (each is tens of lines against a known algorithm) — product principle: no invented consensus, correctness from proven components
- **Quality**: strict golangci-lint configuration (all applicable linters enabled; no suppressions without written justification) and the `go-ultimate` skill conventions (github.com/Djarvur/go-ultimate) govern all implementation from the first feature commit; enforced in CI, red gate blocks merge
- **Process**: PR-only development — branch protection on `master` (server-side ruleset ENFORCED since 2026-11-18, repo public; payload and audit trail in docs/setup/branch-protection.md), direct pushes forbidden, every change lands through a PR from a feature branch; CI on every push and PR
- **Scale envelope**: whole state in RAM (thousands to tens of thousands of keys); values ≤ 60 KiB; one UDP broadcast segment; trusted LAN
- **Correctness**: convergence never depends on wall-clock synchronization; determinism must be proven by simulation under a lossy/duplicating/reordering/partitioning virtual network
- **Performance**: bursts up to ~1000 writes/s on hot keys; disk flush latency within the dump debounce window is acceptable
- **Inspectability**: on-disk state human-readable; every artifact (dump, logs) inspectable with standard tools

<!-- GSD:project-end -->

<!-- GSD:stack-start source:go.mod + go list ./... -->

## Technology Stack

The as-built stack. `cmd/sup-kv/stack_audit_test.go` (TestClaudeStackAudit)
audits this section against `go.mod` and `go list ./...` at test time — the
section's sources are the module files themselves, so a dependency, package
or tool change without an edit here fails `make test`.

| Slot | As-built | Where the truth lives |
|------|----------|----------------------|
| Go | `go 1.27.4` — the `go.mod` directive's own value | `go.mod` |
| Direct runtime requires | exactly two: `google.golang.org/grpc v1.85.0` (the API layer) and `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e` (the wire encoding) | `go.mod` |
| `go.mod` tool directives | seven — `buf`, `grpcurl`, `golangci-lint`, `goreleaser`, `govulncheck`, `protoc-gen-go`, `protoc-gen-go-grpc` — run via `go tool`, never `tools.go` | `go.mod` tool block |
| Property testing | hand-rolled seeded `math/rand/v2` generators in `internal/app/internal/testutil`; the module carries no third-party PBT library | the generators |
| Simulation | `internal/app/internal/simnet` — a discrete-event scheduler ordered by (Time, Seq): no goroutines, no wall clock, virtual time is the only time | simnet's package doc |
| Dumps | `json.Encoder` one-line JSONL written by `internal/app/persist/codec.go` (one object per line, sorted by key; never indented documents) | `internal/app/persist/codec.go` |
| Version identity | ldflags-stamped (`-X main.version`, `-X main.commit`) at build time; `--version` reports it | the Makefile `build` target |
| Socket buffers | `internal/udprepl/socket.go` sets SO_RCVBUF/SO_SNDBUF at construction and reads the kernel values back (XPORT-05) — the "call SetReadBuffer early" advice is as-built | `internal/udprepl/socket.go` |
| golang.org/x/sync | v0.25.0, indirect — it rides grpc's module graph; the in-repo code uses plain `sync` discipline, not errgroup | `go.mod` |
| Lint gate | golangci-lint v2.15.0 exactly, with the vendored powerman strict config kept byte-unmodified behind its `# Origin:` header | `.golangci.yml` |

### Development Tools (pinned in the go.mod tool block; run via `go tool`)

| Tool | Version | Purpose |
|------|---------|---------|
| Buf CLI (`github.com/bufbuild/buf/cmd/buf`) | v1.75.0 | proto codegen driver, lint, format, breaking-change detection |
| protoc-gen-go (`google.golang.org/protobuf/cmd/protoc-gen-go`) | the runtime module's own plugin — one module, no separate pin | Go message codegen, invoked by buf |
| protoc-gen-go-grpc (`google.golang.org/grpc/cmd/protoc-gen-go-grpc`) | v1.6.2 | gRPC service codegen, invoked by buf |
| golangci-lint (`github.com/golangci/golangci-lint/v2/cmd/golangci-lint`) | v2.15.0 | the strict lint gate |
| goreleaser (`github.com/goreleaser/goreleaser/v2`) | v2.20.2 | tag-triggered release: archives, checksums, SBOMs |
| grpcurl (`github.com/fullstorydev/grpcurl/cmd/grpcurl`) | v1.9.4 | the gRPC client the executed doc blocks run (shimmed in CI) |
| govulncheck (`golang.org/x/vuln/cmd/govulncheck`) | v1.9.0 | scheduled vulnerability scanning |

Notes:

- protoc-gen-go: plugin and runtime are one module — the runtime require in `go.mod` pins both, so the plugin and runtime versions cannot drift; the v1.36.12 pairing named by the pre-implementation research is superseded by the module's version line.
- golangci-lint-action v9 wraps the lint job in CI (not a pinned tool); actions/checkout@v7 and actions/setup-go@v7 are the CI harness.
- Conventions are established: `AGENTS.md` is the canonical conventions file, and its dependency policy mirrors the tool inventory above.

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions are established: `AGENTS.md` is the canonical conventions file — the three recorded go-ultimate deviations, the package layout, the dependency policy and the tool inventory. The machine-checkable ones are pinned by the doc-anchor guards in `cmd/sup-kv` (the stack audit plus the README, EXAMPLES, OPERATIONS, API, CONFIGURATION and FORMATS guards).
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

The architecture is mapped in `AGENTS.md`'s Layout section: the package inventory (five `internal/app` packages plus the harness packages), the four adapters, the port boundary and the depguard rules that hold it. The write path, the event loops and the concurrency model are documented in the packages' own doc comments.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-fast` for a trivial task inline, with no subagents and no PLAN.md
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
