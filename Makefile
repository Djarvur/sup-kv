.PHONY: lint test generate build proto buf-lint buf-format buf-breaking

# Version stamping for the build target (D-18): version from git describe,
# commit from the short sha, "dev" when git is unavailable.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

## lint: strict golangci-lint gate (same pinned binary as CI via the go.mod tool directive).
## config verify first: `run` silently ignores unknown top-level config keys (T-01-01),
## so the vendored strict config is schema-checked on every gate invocation.
lint:
	go tool golangci-lint config verify
	go tool golangci-lint run ./...

## test: race-detector tests with coverage.
test:
	go test -race -cover ./...

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
	go tool buf generate internal/grpcapi/apiproto --template buf.gen.api.yaml

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
