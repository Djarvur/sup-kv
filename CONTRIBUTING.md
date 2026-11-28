# Contributing to sup-kv

The real development loop: which gates run, how changes land, and which
tools are used. The project conventions — the recorded style deviations,
the package-layout boundary contract and the dependency policy — live in
[AGENTS.md](AGENTS.md), the canonical conventions file; this guide points at
it and never restates those contracts divergently.

## Dev loop

Two local gates, both before pushing:

```text
make lint    # go tool golangci-lint config verify && go tool golangci-lint run ./...
make test    # go test -race -cover ./...
```

`make lint` verifies the vendored strict config's schema first (config
verify: `run` silently ignores unknown top-level config keys) and then runs
the strict gate. `make test` runs every test under the race detector with
coverage. CI runs the same tools directly — `go tool golangci-lint config
verify`, the golangci-lint action, `go test -race ./...` — on every push and
PR; it does not call make for these gates. Proto gates (`buf lint`, `buf
format --diff --exit-code`, `buf breaking` against origin/master) run in CI
on every push too, so a proto change without regeneration is caught before
review.

`make red-run` (the expected-red gate and mutation probes) and `make
race-soak` (the flake soak) are maintainer/scheduled-job surfaces — see
[docs/SIMULATOR.md](docs/SIMULATOR.md) for what they run.

## Branches and PRs

Development is PR-only: direct pushes to `master` are forbidden, and every
change lands through a PR from a feature branch cut from an up-to-date
`master`. Branch protection is a server-side ruleset; its verify command
and payload are recorded in [docs/setup/branch-protection.md](docs/setup/branch-protection.md).
CI runs on every push and every PR.

## Proto changes

Edit the `.proto` source and regenerate — never hand-edit generated code:

```text
make proto    # go tool buf generate (both proto trees)
```

The committed `*.pb.go` files are the build input (CI never regenerates).
Buf's gates run in CI: lint, canonical format (`--diff --exit-code`), and
the wire-schema comparison against `origin/master`. Keep regeneration diffs
comment-only where the change is comment-only.

## Client example

`examples/client` is a separate Go module (`example.com/sup-kv-client`,
deliberately outside the root module's prefix, with a `replace` to this
repository) that imports only the public API package
`github.com/Djarvur/sup-kv/api/supkvapi/v1`. The root module's `go test
./...` never visits it by design — `make client-example` is its single gate
(build, vet, tidy-proof), and the CI `client-example` job invokes exactly
that target.

## Tools

Every tool runs through the `go.mod` tool block via `go tool <name>` —
never a bare `go install`, never a downloaded binary, never a `tools.go`
file. The pinned set: buf, grpcurl, golangci-lint, goreleaser, govulncheck,
protoc-gen-go, protoc-gen-go-grpc. Using the tool directives keeps the tool
versions and the module's dependency graph in one pinned place.

## Conventions

[AGENTS.md](AGENTS.md) is the canonical conventions file: the recorded
deviations from the strict lint setup (each with its written justification),
the layout contract, the depguard boundary rules and the dependency policy
all live there and are machine-audited against `go.mod` and `go list ./...`
— a doc change or a module change alone will fail `make test`. The as-built
architecture description is [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Release

Releases are tag-triggered: the pinned goreleaser tool runs through
`.github/workflows/release.yml` on a pushed `v*` tag, producing the Linux
archives, checksums and SBOMs, and the GitHub Release body from
`CHANGELOG.md`. The mandatory pre-tag gate is local: `make release-snapshot`
proves the whole artifact pipeline with no tag and no publishing — run it
and see it green before any tag is pushed. `make release` is the same
pipeline's local invocation; the tag on merged `master` is what publishes.
The step-by-step executed procedure — snapshot, archive inspection, the
one-way tag, and the release verification — is
[docs/setup/release-procedure.md](docs/setup/release-procedure.md).
