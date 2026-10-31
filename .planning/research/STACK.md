# Stack Research

**Domain:** Leaderless replicated in-memory KV store (LWW-Element-Map CRDT, HLC/UUIDv7 versions, UDP replication, JSON dumps, gRPC API) in Go — implementation-level stack around the decided design
**Researched:** 2026-10-31
**Confidence:** HIGH (all version facts machine-verified today against proxy.golang.org, the GitHub Releases API, the Go 1.27.1 GOROOT source, and the mandated go-ultimate v0.15.0 skill; practice recommendations cross-checked against official docs)

The design is decided (`docs/IDEA.md`). This document prescribes the implementation stack only. The governing constraint is the project principle: **runtime dependencies = gRPC + protobuf only**; HLC, UUIDv7 packing, the LWW map, and the UDP reliability layer are in-repo. Everything below respects that bar or explicitly flags the deviation.

## Recommended Stack

### Core Technologies (runtime dependencies — the entire shipped graph)

| Technology | Version | Purpose | Why Recommended | Confidence |
|------------|---------|---------|-----------------|------------|
| Go | 1.27 (`go 1.27` directive; current patch 1.27.1) | Language + stdlib | Latest stable (released 2026-08-19, verified locally). go-ultimate policy: applications pin **latest** stable Go. grpc-go v1.85 already requires `go >= 1.26.0`, so the floor is forced upward anyway. Bonus on 1.27: `encoding/json/v2` engine is now default under the v1 API, and `testing/synctest` (stable since 1.25) is available for the simulator. | HIGH |
| google.golang.org/grpc | v1.85.0 (2026-10-27, latest stable) | Client/server API, Watch server-streaming | The decided API layer. Release notes verified current; v1.85 fixed control-buffer throttling defaults and `GracefulStop`-during-`Stop`. Its `go.mod` requires Go 1.26 and already depends on `github.com/google/uuid v1.6.0` and `golang.org/x/sync v0.23.0` (matters below). Enable `google.golang.org/grpc/reflection` (same module, zero cost) so `grpcurl` works for ops/debugging. | HIGH |
| google.golang.org/protobuf | v1.36.12 (latest stable; `go >= 1.23`) | Message encoding for UDP envelopes and gRPC | The other half of the decided dependency bar. Also provides `cmd/protoc-gen-go` at the same version for codegen. | HIGH |
| github.com/google/uuid | v1.6.0 (latest, 2024-01-16) — **optional direct dep** | `RunID` generation, RFC 9562 test cross-checks | **Zero marginal cost**: grpc-go v1.85 already requires uuid v1.6.0 in its module graph, so promoting it to direct adds no new module to the build. v1.6.0 fixed `NewV7` monotonicity under concurrency (PR #150). RecordVersion packing stays in-repo regardless — `NewV7` does not implement the HLC counter-in-`rand_a` scheme sup-kv needs. Alternatively generate RunID from `crypto/rand` in ~10 in-repo lines; either is acceptable, do not vendor anything else for it. | HIGH |
| golang.org/x/sync | v0.23.0 (match grpc's requirement) — **optional direct dep** | `errgroup` for fail-fast goroutine fan-out | **Zero marginal cost**: already in grpc-go's graph. go-ultimate mandates errgroup for parallel fan-out; for plain tracked goroutines `sync.WaitGroup.Go` (Go 1.25+) suffices with no dep at all. Allowed as a conscious exception to the "gRPC + protobuf only" bar only because it adds nothing new to `go.sum`. | HIGH |

**Total shipped dependency graph: 2 required modules (grpc + protobuf) + at most 2 zero-marginal ones already inside grpc's graph. Nothing else.** Everything below is stdlib, test-only, or a pinned dev tool.

### Standard Library Components (where the decided design maps)

| Component | Version | Purpose | Why / Notes | Confidence |
|-----------|---------|---------|-------------|------------|
| `net` (UDP) | stdlib | Replication transport | Go sets `SO_BROADCAST=1` by default on every `SOCK_DGRAM` socket on all platforms — verified in the Go 1.27 GOROOT source (`net/sockopt_linux.go`, `sockopt_bsd.go`, `sockopt_windows.go`); no raw syscalls needed. Use one unconnected `net.ListenUDP` socket for both broadcast and unicast (`WriteToUDP`), which is what the design's peer table + own-broadcast-echo filtering assumes. Call `SetReadBuffer` early — kernel default rcvbuf is the main datagram-loss source at ~1000 writes/s bursts. Filter own broadcast echo by NodeID (design already requires this). | HIGH |
| `encoding/json` (v1 API) | stdlib (v2 engine under it since Go 1.27) | Human-readable JSON dumps | v1 API is stable and deterministic: map keys are sorted, `[]byte` is base64 automatically — exactly the dump format the design specifies. `encoding/json/v2` became default-on in Go 1.27 but the v1 API remains fully supported (reimplemented on the v2 engine). Keep the v1 API for on-disk format stability; do not chase v2 semantics mid-project (v2 is stricter: rejects duplicate keys/invalid UTF-8 — irrelevant for machine-written dumps). Use `json.MarshalIndent` for human readability. | HIGH |
| `log/slog` | stdlib (Go 1.21+) | Structured logging | go-ultimate observability floor mandates slog with structured key-value pairs; JSON handler for logs-as-artifact inspectability. Zero deps, satisfies "every dropped-datagram class distinguishable" via event/attrs naming. | HIGH |
| `expvar` + in-repo Prometheus text exporter | stdlib + ~100 in-repo lines | Metrics endpoint | See "Metrics decision" below. go-ultimate's floor is "one Prometheus-style registry; expose /metrics" — an in-repo text-format exporter satisfies it without `client_golang`. `expvar` provides runtime/debug vars at `/debug/vars` for free (GC stats, memstats). | MEDIUM |
| `testing/synctest` | stdlib (stable since Go 1.25) | Deterministic virtual-time bubbles for the simulator | The closest thing Go has to built-in deterministic simulation: `synctest.Test(t, func(t *testing.T){...})` runs goroutines in a bubble with a fake clock that advances only when every goroutine is durably blocked, plus deadlock detection. Combined with an in-repo seeded-chaos virtual network (below), this is the simulation harness for the failure-scenario suite. | HIGH |
| `testing` + `go test -race` | stdlib | Test framework | go-ultimate: vanilla `testing`, table-driven tests, no assertion library. `-race` mandatory in CI on every platform. | HIGH |
| `flag` + `os.Getenv` | stdlib | Flags/env config, `--version` | Least mechanism; the parameter list in IDEA.md maps directly to flags. No cobra/viper (see What NOT to Use). | HIGH |
| `runtime/debug.ReadBuildInfo` | stdlib (Go 1.24+ VCS stamping) | `--version` output | Prefer over `-ldflags -X` where possible; fall back to `-X` for version/date (go-ultimate build-stamping recipe). | HIGH |
| `crypto/rand`, `hash/crc32` | stdlib | RunID randomness (if not using uuid), record Checksum | Direct from the decided design; `hash/crc32` with Castagnoli or IEEE per design doc — tens of lines either way. | HIGH |

### Test-Only Dependencies

| Library | Version | Purpose | When to Use | Confidence |
|---------|---------|---------|-------------|------------|
| `pgregory.net/rapid` | v1.3.0 (2026-03-30; `go >= 1.18`) | Property-based testing: CRDT merge laws (commutative/associative/idempotent), HLC monotonicity under adversarial clocks, version uniqueness | **The one justified test-only dependency.** It is the de-facto standard Go PBT library with integrated shrinking (RapidCheck/Hypothesis style) — mandatory quality for shrinking a HLC clock-jump counterexample to a minimal failing sequence. stdlib `testing/quick` has no shrinking and no stateful generators; gopter is older with weaker ergonomics. Flag: it technically widens "gRPC + protobuf only" — the bar governs the *shipped runtime graph*; keep it in `_test.go` imports only. Note: module path is the vanity domain `pgregory.net/rapid` (repo `github.com/flyingmutt/rapid`; `github.com/pgale/rapid` does not exist — GitHub API 404, verified). | HIGH |
| `testing/synctest` (stdlib) | Go 1.25+ | Virtual time in simulator tests | Free; no dependency cost. | HIGH |
| In-repo simulator | — | Deterministic N-node simulation with lossy/duplicating/reordering/partitioning virtual network | **No Go DST framework exists to adopt** (searched 2026-10-31: articles and commercial Antithesis only; TigerBeetle's VOPR and marionette are Zig). Build in-repo per the "mostly DST" pattern (Polar Signals, 2024): nodes as goroutines inside a `synctest` bubble, a `Clock` interface for virtual time, and a virtual network whose chaos operations (drop/duplicate/reorder/partition) are driven by a seeded `math/rand/v2` — scenario seed printed on failure for reproducibility. Assert convergence invariants and bounds, not bit-identical traces: FoundationDB-style full determinism is not achievable in plain Go, and the CRDT's correctness guarantee (merge order-independence) makes trace determinism unnecessary. | MEDIUM |

### Development Tools (all pinned via `go tool` directives, Go 1.24+ — go-ultimate mandate; never `tools.go`, never bare `go install`)

| Tool | Version | Purpose | Notes | Confidence |
|------|---------|---------|-------|------------|
| Buf CLI | v1.75.0 (2026-10-22, GitHub Releases API) | Protobuf codegen driver + `.proto` lint + breaking-change detection | **Buf over protoc** (rationale below). Not a Go module dependency — a CI/dev binary pinned by release tag. `buf.gen.yaml` v2 format with local plugins invoking the pinned Go plugins: `local: go tool protoc-gen-go`. Buf's breaking-change detection additionally guards the wire protocol (`proto_version` evolution, field renumbering mistakes) — cheap insurance for a replication protocol. | HIGH |
| `protoc-gen-go` | v1.36.12 (= protobuf module version) | Go message codegen | Pinned as a tool directive: `go get -tool google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12`; invoked by buf, never by hand. `paths=source_relative`. | HIGH |
| `protoc-gen-go-grpc` | v1.6.2 (2026-05-07, latest) | gRPC service codegen | Same pinning pattern. v1.6.x generated code requires services to embed `Unimplemented<Service>Server` **by value** (opt-out exists but is not recommended). Watch streams, startup-gate errors, `wait_for_replication` are all plain generated-server work. | HIGH |
| golangci-lint | v2.15.0 (latest, proxy-verified) | Lint gate | Config: **powerman/golangci-lint-strict** — publishes enable-nearly-all-linters configs pinned per exact golangci-lint version; latest config is 2.15.0, an exact match. Rules: fetch the config for the exact version, never edit upstream, put project overrides in an extending config; no `//nolint` without a written justification (matches the project's stated quality gate). This delivers "all applicable linters enabled" without hand-assembling a list. | HIGH |
| golangci-lint-action | v9 (v9.3.0 latest) | CI lint job | v9 requires golangci-lint v2 — correct pairing with v2.15.0. Pin `version: v2.15` in the action inputs so local `go tool` and CI run the same linter. | HIGH |
| govulncheck (`golang.org/x/vuln/cmd/govulncheck`) | v1.9.0 (latest) | Scheduled vulnerability scanning | Run as a `go tool` directive entry (same-version guarantee) in a **scheduled cron workflow** — new CVEs land without code changes; also on dependency PRs. Low-noise: call-graph reachability, not every advisory. | HIGH |
| GitHub Actions | checkout@v7 (v7.0.1), setup-go@v7 (v7.0.0) | CI | Verified via Releases API today. `setup-go` with `go-version-file: go.mod` (go-ultimate: the Go version lives in exactly one place; module caching is built in). checkout v7 hardened `pull_request_target` defaults. | HIGH |
| Dependabot | gomod + github-actions ecosystems, weekly, grouped | Dependency updates | Config: `version: 2`, ecosystems `gomod` and `github-actions`, `interval: weekly`, group minor+patch so the tiny graph produces one PR, not five. With a two-module graph this stays near-silent. | MEDIUM |

## Installation

```bash
# Runtime dependencies (the entire shipped graph)
go get google.golang.org/grpc@v1.85.0 google.golang.org/protobuf@v1.36.12

# Zero-marginal optional direct deps (already inside grpc's graph)
go get github.com/google/uuid@v1.6.0 golang.org/x/sync@v0.23.0

# Test-only
go get pgregory.net/rapid@v1.3.0

# Pinned dev tools (Go 1.24+ tool directives; recorded in go.mod, run via `go tool`)
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
go get -tool google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
go get -tool google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
go get -tool golang.org/x/vuln/cmd/govulncheck@v1.9.0

# Buf CLI (dev/CI binary, not a module): install pinned from GitHub releases
# https://github.com/bufbuild/buf/releases/tag/v1.75.0 (checksums published on the release)

# Lint config: fetch the strict config for the EXACT golangci-lint version
# https://github.com/powerman/golangci-lint-strict → 2.15.0/.golangci.yml
```

`buf.gen.yaml` (v2) — local plugins invoking the tool-directive-pinned generators, no protoc, no PATH installs:

```yaml
version: v2
plugins:
  - local: go tool protoc-gen-go
    out: .
    opts: [paths=source_relative]
  - local: go tool protoc-gen-go-grpc
    out: .
    opts: [paths=source_relative]
```

CI matrix (all four cells free on a public repo):

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, ubuntu-24.04-arm, macos-13, macos-15]  # linux amd64/arm64, macOS amd64/arm64
steps:
  - uses: actions/checkout@v7
  - uses: actions/setup-go@v7
    with:
      go-version-file: go.mod
  - run: go vet ./... && go tool golangci-lint run ./...
  - run: go test -race ./...
  - run: go mod tidy -diff   # or: go mod tidy && git diff --exit-code go.mod go.sum
```

Single static binary (release target): `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=..." ./cmd/sup-kv`. systemd: `Type=exec`; if `Type=notify` is ever wanted, write `READY=1` to `$NOTIFY_SOCKET` with a ~15-line in-repo helper — do not add a go-systemd dependency.

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Buf CLI v1.75 | protoc + plugins | Only if the team refuses any non-Go-binary tool; protoc needs manual version-pinned downloads (distro packages are stale) and provides no lint/breaking checks for the wire protocol. Buf is the industry default and removes protoc entirely. |
| In-repo Prometheus text exporter + expvar | prometheus/client_golang v1.25.0 | Only if the dependency bar is formally relaxed by ADR (it drags prometheus/common, procfs, client_model into go.sum). Histograms/label-heavy metrics would be the trigger; sup-kv needs counters and gauges only. |
| In-repo simulator (synctest + seeded virtual network) | Antithesis (commercial), TigerBeetle-style VOPR | Never for v1 (commercial / different language). If bit-identical trace reproducibility ever becomes a requirement, that is a signal to reconsider the testing runtime, not the library list. |
| `pgregory.net/rapid` v1.3.0 | stdlib `testing/quick`, gopter | testing/quick only for throwaway checks (no shrinking, no stateful generators); gopter if rapid's API is rejected — neither is recommended. Note: the "pterodactyl" PBT library named in the research brief **does not exist** (verified: two searches + GitHub — no such Go library). |
| `slog` | zap / zerolog | Only for a service with pprof-proven serialization overhead in hot paths; sup-kv's write path is UDP + map updates, not log-bound. slog is the stdlib standard and the go-ultimate floor. |
| `flag` + env | cobra / viper | Only if the CLI grows subcommands beyond one binary with flags; even then cobra alone, never viper (config is flags+env by design). |
| Plain `go build` matrix in GH Actions | goreleaser | Fine to adopt at the v1.0 release phase for 4-target archives + checksums + SBOM in one shot; it is release tooling, not a runtime dep — but do not block Phase 0 on it. |
| golangci-lint-action v9 + powerman strict config | hand-rolled enable-list from go-ultimate engineering-policy | The hand-rolled 11-linter list is the go-ultimate *default*; the project's stated gate ("all applicable linters, no suppressions without written justification") is exactly what powerman/golangci-lint-strict encodes, so take the strict config. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `prometheus/client_golang` | Adds 3+ modules to `go.sum` — breaks the explicit "gRPC + protobuf only" bar for metrics that are all counters/gauges | In-repo text-format exporter (~100 lines, classic `text/plain; version=0.0.4`, sorted families/labels) at `/metrics` + stdlib `expvar` at `/debug/vars` |
| `protoc` (C++ binary) | Stale distro packages, manual pinned downloads, no lint/breaking detection for the replication wire format | Buf CLI pinned by release tag |
| `tools.go` blank-import hack | Replaced by Go 1.24+ `tool` directives; causes phantom imports and version drift | `go get -tool ...` + `go tool ...` |
| Assertion libraries (testify, go-testdeep) | go-ultimate: vanilla `testing` + table tests; assertion libs hide failure locality and add a public dep to test files | stdlib `testing` (+ `rapid` for properties) |
| zap/zerolog/logrus | Non-stdlib logging with no advantage at this scale; logrus is unmaintained | `log/slog` |
| cobra/viper | "flags/env config" is fully served by stdlib `flag`; viper's file/remote layers are out of scope | `flag` + `os.Getenv` |
| Raft/quorum libraries (hashicorp/raft, etcd) | The design is decided CRDT/AP; any consensus lib contradicts the product's reason to exist | The decided LWW-Element-Map in-repo |
| UUID libraries other than google/uuid (gofrs/uuid, etc.) | New modules in the graph for what is either in-repo (RecordVersion) or already free (RunID via grpc's graph) | In-repo RecordVersion packing; google/uuid v1.6.0 or crypto/rand for RunID |
| BSR remote plugins in `buf.gen.yaml` | Codegen becomes network-dependent and non-hermetic; CI reproducibility suffers | `local: go tool protoc-gen-go` plugins |
| Antithesis / DST frameworks | No Go OSS DST framework exists; commercial service is out of scope for v1 | In-repo synctest simulator |
| `encoding/json/v2` API migration mid-project | v2 engine is already default under the v1 API in Go 1.27; changing dump semantics risks the on-disk format contract for zero benefit | Keep v1 API; revisit post-1.0 with an ADR |

## Stack Patterns by Variant

**If the repository is private** (arm64 macOS/Linux hosted runners bill per minute):
- Keep the 4-arch *build* matrix but run `-race` tests on `ubuntu-latest` (amd64) + one arm64 runner only, and cross-compile (`GOOS`/`GOARCH`) for the remaining cells; gate on repo visibility when choosing.

**If metrics consumers are Prometheus-native:**
- The in-repo text exporter is scrape-compatible with Prometheus 3.x (classic format). Only introduce `client_golang` via an ADR if histograms with SLO bucketing are ever demanded.

**If a failure scenario test cannot reproduce from the printed seed:**
- The nondeterminism is in unvirtualized I/O (real filesystem in dump tests, real socket in non-simulator tests) — route those through the same virtual interfaces the simulator uses; do not reach for a heavier framework.

**If Dependabot PRs churn the two-module graph:**
- Expected to be near-zero (grpc + protobuf release monthly); group minor+patch into one weekly PR and review majors manually (grpc majors are rare; the `retract` note on their v1.74.0 shows why pinned review matters).

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| google.golang.org/grpc v1.85.0 | Go >= 1.26.0 (from its go.mod) | Forces the go directive to ≥1.26 anyway; use 1.27 per go-ultimate app policy |
| protoc-gen-go-grpc v1.6.2 | grpc-go current (v1.85) | Generated services must embed `Unimplemented<Service>Server` by value (default); pair with the same grpc-go version in tests |
| protoc-gen-go v1.36.12 | protobuf runtime v1.36.12 | Same module — keep plugin and runtime versions identical |
| powerman/golangci-lint-strict 2.15.0 | golangci-lint exactly v2.15.0 | Config schema changes per version; mismatch fails to parse or silently drops sections |
| golangci-lint-action v9 | golangci-lint v2.x only | v8+ dropped v1 support; pin `version: v2.15` in action inputs |
| pgregory.net/rapid v1.3.0 | Go >= 1.18 | No constraints against 1.27 |
| Buf v1.75.0 | buf.yaml/buf.gen.yaml `version: v2` configs | v2 config format is GA on all current buf 1.x |
| actions/checkout@v7, actions/setup-go@v7 | Current runner images | setup-go v7 requires runner v2.327.1+ (all current hosted runners qualify); go-version-file pins to go.mod |

## Sources

Confidence per the GSD source-hierarchy seam (webfetch/websearch verified = MEDIUM; unverified web claim = LOW). Versions additionally machine-verified 2026-10-31 against proxy.golang.org, api.github.com/releases, and the local Go 1.27.1 GOROOT — those facts are HIGH.

- proxy.golang.org (`go list -m -versions` / `@latest` / `.mod` files) — grpc v1.85.0 (go 1.26 floor, uuid+x/sync in graph), protobuf v1.36.12, golangci-lint v2.15.0, protoc-gen-go-grpc v1.6.2, x/vuln v1.9.0, client_golang v1.25.0, pgregory.net/rapid v1.3.0, google/uuid v1.6.0 — HIGH (machine-verified)
- GitHub Releases API — bufbuild/buf v1.75.0, actions/checkout v7.0.1, actions/setup-go v7.0.0, golangci-lint-action v9.3.0; google/uuid releases (v1.6.0 NewV7 monotonicity fix) — HIGH (machine-verified)
- Go 1.27.1 GOROOT source — `SO_BROADCAST` default-on for `SOCK_DGRAM` (`net/sockopt_linux.go` et al.) — HIGH (source-verified)
- github.com/grpc/grpc-go/releases — v1.85.0 release notes (2026-10-27) — MEDIUM
- pkg.go.dev / buf.build docs — protoc-gen-go-grpc v1.6.2 semantics (Unimplemented embedding); buf.gen.yaml v2 local-plugin syntax (`local: go tool ...`) — MEDIUM
- github.com/powerman/golangci-lint-strict — per-version strict configs, latest 2.15.0 — MEDIUM
- github.com/Djarvur/go-ultimate skill v0.15.0 (`references/engineering-policy.md`, `references/repo-and-ci.md`, `references/libraries.md`) — mandated conventions: tool directives, go-version-file, tidy gate, slog + Prometheus-style registry floor, build stamping — MEDIUM (curated, project-mandated)
- Go release history / jsonv2: Go 1.25 experiment → 1.27 default (`go.dev/doc/jsonv2-migration`, release coverage) — MEDIUM
- `testing/synctest` stable since Go 1.25 (proposal #67434, release coverage) — MEDIUM
- Polar Signals, "(Mostly) Deterministic Simulation Testing in Go" (2024-05-28) + Resonate HQ DST writeup + Antithesis docs — DST-in-Go pattern — MEDIUM
- GitHub community discussions (#73040) — Linux arm64 hosted runners free for public repos — MEDIUM
- Negative claims (no "pterodactyl" Go PBT library; no Go DST framework) — two independent searches each, no counter-evidence found — LOW (inherently unverifiable negatives)

---
*Stack research for: sup-kv (leaderless CRDT replicated in-memory KV store, Go)*
*Researched: 2026-10-31*
