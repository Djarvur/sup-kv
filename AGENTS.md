# AGENTS

## Conventions

- Go version: as pinned in `go.mod` (`go 1.27`). Do not raise it without an ADR.
- Run `make lint` and `make test` (`go tool golangci-lint run ./...` and
  `go test -race ./...`) before pushing; CI runs the same targets.
- Deviations from go-ultimate: three, recorded below with reasons.

### Project deviations

1. **Depguard architecture boundary.** The lint gate machine-enforces the
   architecture: `internal/app` (the pure decision core) may not import
   `net`, `os`, `syscall` or `google.golang.org/*` (rule `sup-kv-app-no-io`);
   `internal/port` may import standard library and the own module path only
   (rule `sup-kv-port-stdlib-only`); and `api/**` may not import
   `github.com/Djarvur/sup-kv/internal` (rule
   `sup-kv-public-api-no-internal`), so the published client package stays
   importable from another module. Three more additive rules hold the
   adapter direction: `sup-kv-adapters-not-in-core` (neither `internal/app`
   nor `internal/port` imports an adapter), `sup-kv-metrics-no-adapters`
   (the exporter imports value seams, never a socket, dump or API adapter)
   and `sup-kv-grpcapi-no-repl-metrics` (the API adapter imports neither the
   replication adapter nor the exporter). One composition edge is allowed
   with a written justification in `.golangci.yml`: `grpcapi` imports
   `internal/persistfile` for its two snapshot sentinels and
   `internal/app/persist` for the `*persist.Engine` store facade. These
   rules live in `.golangci.yml`. Any change to the boundary is a visible
   `.golangci.yml` diff, never a quiet code-side workaround.
2. **Vendored lint config provenance.** `.golangci.yml` is the
   powerman/golangci-lint-strict config for golangci-lint **exactly** 2.15.0,
   fetched from
   `https://raw.githubusercontent.com/powerman/golangci-lint-strict/main/2.15.0/.golangci.yml`
   and kept byte-unmodified behind its `# Origin:` header (first line). Project
   rules are additive keys inside `linters.settings.depguard.rules` only. A
   golangci-lint version bump means re-fetching the matching strict config —
   never editing upstream lines in place.
3. **No lint suppressions without a written justification.** Every `//nolint`
   must name its specific linter and carry a human-readable reason (enforced
   by `nolintlint.require-explanation` and `require-specific` in the vendored
   config).

## Layout

- `api/supkvapi/v1/` — the public API schema: `supkvapi.proto` and its
  generated Go package `supkvapi`. It is the client-facing surface and stays
  importable from another module (`github.com/Djarvur/sup-kv/api/supkvapi/v1`);
  the replication proto under `internal/udprepl/replproto/` remains internal.
- `cmd/sup-kv/` — the single binary; `main()` parses (via `internal/config`),
  handles help/version, installs logging and calls `run`, the tested
  lifecycle seam; signals and the shutdown order live in `run`/`shutdown`.
- `internal/config/` — composition-level configuration: `Parse` (flag > env >
  default), `Validate` (every startup range and relation in one pass) and
  `Usage` (the rendered help). It imports the standard library and the own
  module's constants (`internal/app/repl`, `internal/app/storage`,
  `internal/udprepl`) and deliberately lives outside the `internal/app`
  no-I/O rule: it reads the environment and validates the adapters' datagram
  ceiling, which the lint gate's `sup-kv-app-no-io` must not cover.
- `examples/client/` — a separate Go module (`examples/client/go.mod`, module
  path `example.com/sup-kv-client`, deliberately outside the root module's
  prefix, with a `replace` to the root) that imports only the public API
  package and builds in CI via `make client-example`. Its out-of-prefix path
  makes Go's internal rule reject an `internal/` import from the example; the
  depguard rule `sup-kv-public-api-no-internal` remains the enforced boundary
  inside the repository.
- `internal/app/` — pure decision core, split into three packages:
  `version` (RFC 9562 record versions), `clock` (hybrid logical clock) and
  `storage` (LWW-Element-Map store, consuming the consumer-side `Clock`
  interface declared in `storage`); no I/O, enforced by the lint gate.
  `internal/app/internal/testutil` holds the shared property-test
  generators.
- `internal/port/` — port/interface definitions; stdlib-only imports,
  enforced by the lint gate.
- Adapter packages under `internal/` appear with the phases that need them;
  do not create empty adapter directories.

## Dependency policy

- The shipped runtime graph is intentionally minimal: Phase 1 carries **zero
  direct runtime requires**. gRPC and protobuf enter `go.mod` in the phases
  that first use them. The client-facing API schema under `api/` is public —
  its generated Go package is the supported client entry point — while the
  replication proto under `internal/udprepl/replproto/` stays internal.
- The `tool` directives (golangci-lint v2.15.0, govulncheck v1.9.0) drag
  ~200 `// indirect` requires into `go.mod`/`go.sum`. This is the accepted
  reading of "zero-dependency" (D-03): it binds **direct runtime requires**
  only — the shipped build graph stays empty. Tools are always invoked via
  `go tool <name>`, never `tools.go`, never bare `go install`.
