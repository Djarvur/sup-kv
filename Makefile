.PHONY: lint test generate build proto

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

## build: compile all packages.
build:
	go build ./...

## proto: regenerate the committed replication wire code (dev machines only —
## CI never regenerates; internal/udprepl/replproto/supkv.pb.go is committed).
## Builds the go.mod-pinned plugin, then runs system protoc (3.21.x) over the
## schema with source-relative output paths.
proto:
	go build -o .tmp/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	protoc --plugin=.tmp/protoc-gen-go \
	       --go_out=. --go_opt=paths=source_relative internal/udprepl/replproto/supkv.proto
