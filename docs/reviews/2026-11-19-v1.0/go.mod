// Reviewer scratch module marker (2026-11-19 v1.0 review evidence).
// Purpose: keep `go test ./...` / `go build ./...` from the repository root from
// descending into the review's intentionally-RED repro tests. Nobody builds this
// module; the files are committed verbatim as review evidence.
module review-repro-scratch

go 1.27
