.PHONY: lint test bench bench-smoke race-soak red-run generate build proto buf-lint buf-format buf-breaking release release-snapshot

# Version stamping for the build target (D-18): version from git describe,
# commit from the short sha, "dev" when git is unavailable.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

# Repetition count for race-soak: the TEST-11 command's default is the full
# 20 runs; `make race-soak COUNT=1` is the cheap local sanity pass. The
# ×3-green-across-runs bar is realized by plan 11's scheduled job, not by
# one local invocation.
COUNT ?= 20

## lint: strict golangci-lint gate (same pinned binary as CI via the go.mod tool directive).
## config verify first: `run` silently ignores unknown top-level config keys (T-01-01),
## so the vendored strict config is schema-checked on every gate invocation.
lint:
	go tool golangci-lint config verify
	go tool golangci-lint run ./...

## test: race-detector tests with coverage.
test:
	go test -race -cover ./...

## bench: the three TEST-05 benchmarks (D-13) — dump at 10k keys
## (BenchmarkDump10kKeys), replication burst throughput
## (BenchmarkReplicationBurst) and bounded confirmation latency
## (BenchmarkConfirmationLatency). Manual before/after comparison target
## ONLY: CI never runs it and there are no thresholds — shared-runner noise
## makes those flaky (D-13) — and no profiles are stored (D-14). Unit tests
## stay excluded (-run anchored empty); compare two runs by hand.
bench:
	go test -run '^$$' -bench . ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/

## bench-smoke: TEST-10's CI smoke — every benchmark exactly once
## (-benchtime=1x) on the real paths (real persistfile, default parameters)
## so CI exercises the measurement code and asserts the p50-ms/p99-ms lines
## appear; no thresholds, no self-pacing. Plan 11's scheduled bench job
## invokes this target, so the command has exactly one definition.
bench-smoke:
	go test -run '^$$' -bench . -benchtime=1x ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/

## race-soak: TEST-11's flake command — the affected set under the race
## detector for COUNT repetitions (default 20, the command D-09 names;
## `make race-soak COUNT=1` is the cheap sanity pass). The ×3-green bar
## spans scheduled runs: plan 11's nightly job runs this target three
## nights and requires green across runs.
## Flake classes audited after the internal/stand harness move:
##   1. UDP port pick-to-bind — structurally mitigated: internal/stand's
##      private window [28100, 32678) plus the claim registry moved intact.
##   2. SIGTERM before the handler is installed — a real product-side
##      binary window, carried per D-09's bar (needs a Phase 3-5 fix;
##      TestShutdown_secondSignal stays the existing post-arm pin).
##   3. README/EXAMPLES verbatim-block deadline — fixed test-side: each
##      doc block is bounded by docBlockBudget clamped inside t.Deadline()
##      (cmd/sup-kv/examples_test.go), so a wedged block fails its own run.
## Cost of the WR-06 addition (plan 03-13): `time go test -race -count=1
## ./internal/stand/` measures 6.53 s wall on the 3.88 GHz
## performance-governor host, so repeating it COUNT=20 projects ~2.2 min
## locally (~2-3 min on a 2-core hosted runner per review 02's estimate).
## The package's red tests skip without SUPKV_RED_TESTS, so the soak
## repeats only the ordinary stand suite and no red scenario inflates the
## count. The full target measures 81 s at COUNT=1 (the chaos package is
## the bulk), so COUNT=20 projects ~27 min — inside the 120-minute
## race-flake job budget unchanged.
race-soak:
	go test -race -count=$(COUNT) ./cmd/sup-kv/ ./internal/udprepl/ ./internal/app/repl/ ./internal/app/persist/ ./internal/persistfile/ ./internal/grpcapi/ ./internal/stand/

## red-run: the D-01 expected-red gate. Runs scripts/red-runner: the
## env-gated red tests with SUPKV_RED_TESTS=1 (the script exports it), every
## scripts/red-expectations.tsv row asserted from go test -json events — a
## red row must FAIL, a skip or an absence is an error — with the run's
## evidence recorded under .planning/phases/02-.../red-evidence/. Normal
## gates stay green by construction: red tests skip without the env var, so
## `make test` never runs their bodies. The Phase 3-5 flip-to-green changes
## only the expectation-table row (red -> green), never this Makefile.
red-run:
	bash scripts/red-runner

## generate: run go:generate directives.
generate:
	go generate ./...

## build: compile all packages, then produce the shippable artifact
## bin/sup-kv (gitignored): one static binary (CGO_ENABLED=0 — pure-Go deps
## only) with ldflags version/commit stamping (D-18).
build:
	go build ./...
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/sup-kv ./cmd/sup-kv

## proto: regenerate the committed wire code (dev machines only — CI never
## regenerates; the pb.go files are committed). One buf generate pass per
## proto tree (go.mod-pinned buf via the tool directive): the input is the
## tree itself so source-relative paths resolve against it, and each
## template's out pins the generated code back inside the tree — the
## per-tree buf.yaml modules make a single workspace-wide generate emit
## module-relative paths at the repo root instead.
proto:
	go tool buf generate internal/udprepl/replproto --template buf.gen.repl.yaml
	go tool buf generate api --template buf.gen.api.yaml

## buf-lint: lint the proto workspace with the STANDARD (DEFAULT) rule set —
## zero waivers (buf.yaml).
buf-lint:
	go tool buf lint

## buf-format: rewrite .proto files in place (local convenience only — CI
## checks the canonical format with `buf format --diff --exit-code`, D-04).
buf-format:
	go tool buf format -w

## buf-breaking: FILE-policy wire-schema comparison against origin/master.
## The guard is required because buf v1.75 errors (exit 1) on a proto-less
## against-ref instead of passing, while supkv.proto is not on origin/master
## yet (probe-verified); it self-destructs the moment the proto lands there
## and every later run does the real comparison (D-08).
buf-breaking:
	git fetch origin master
	@if git cat-file -e refs/remotes/origin/master:internal/udprepl/replproto/supkv/v1/supkv.proto; then \
		go tool buf breaking --against '.git#ref=refs/remotes/origin/master'; \
	else \
		echo "buf breaking deferred: supkv.proto not on origin/master yet (D-08 guard)"; \
	fi

## release: cut the release from the pushed v* tag via the go.mod-pinned
## goreleaser tool directive (never a downloaded action binary): 4-target
## archives + checksums.txt + per-archive SBOMs + a GitHub Release whose
## header pulls from CHANGELOG.md (.goreleaser.yaml,
## .github/workflows/release.yml is the tag-triggered counterpart).
release:
	go tool goreleaser release --clean

## release-snapshot: prove the whole artifact pipeline locally with NO tag
## and NO publishing — --snapshot builds all 4 targets plus checksums into
## dist/. The mandatory pre-tag gate (D-09/Pitfall 8): run this and see it
## green before any v* tag is pushed (07-05).
release-snapshot:
	go tool goreleaser release --snapshot --clean
