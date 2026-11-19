# Repro material from the 2026-11-19 review

Every file here is meant to be laid over a scratch copy of the repository (same relative paths);
nothing here belongs in the product tree as is.

- `engine/` — failing tests and benchmarks for the ENG-xx findings (`review_*_test.go`).
  Run: `go test ./internal/app/... ./internal/persistfile/ ./internal/metrics/ -run 'TestReview' -count=1`.
- `repl/` — failing tests for the RPL-xx / API-xx / BIN-xx findings
  (`internal/app/repl/review_repro_test.go`) and adapter probes (`internal/reviewprobe/`).
  The probes need a verbatim copy of `internal/udprepl/repl.go` saved as
  `internal/reviewprobe/repl.go` with the package clause changed to `reviewprobe`.
  They use ephemeral loopback ports only.
- `tests/` — the test audit's probes (`probe/sup-kv/**/zz_*_test.go`, including two fuzz harnesses),
  the mutation tooling (`scripts/`) and its raw result (`out/mutations.jsonl`: 73 evaluated mutants).

A test "fails" while the defect it describes is present. Convert each into a real regression test
when the corresponding finding is fixed.
