.PHONY: lint test generate build

## lint: strict golangci-lint gate (same pinned binary as CI via the go.mod tool directive).
lint:
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
