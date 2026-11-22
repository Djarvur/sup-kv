# Deferred Items — Phase 05

Out-of-scope findings from plan execution. Not fixed here by the deviation
scope boundary; carried for the phase verifier and later plans.

| Item | File | Class | Description | Evidence |
|------|------|-------|-------------|----------|
| UDP pick-to-bind flake | `cmd/sup-kv/main_test.go` (`freeUDPPort`, `TestLog_formatText`) | pre-existing flake, unrelated to 05-04 | A full `go test -race ./...` run failed once with `probe rx bind: bind udp4 0.0.0.0:<ephemeral>: address already in use` while other UDP-using packages ran in parallel. Re-running the test alone passed twice. This is the documented pick-to-bind flake class (Makefile `race-soak` notes, class 1). | 05-04 run: `TestLog_formatText` failed on two different ports in two full/package runs; `go test -race -count=1 ./cmd/sup-kv/ -run '^TestLog_formatText$'` passed on two consecutive retries. |
| DuplicateNodeID event timeout under full-suite load | `cmd/sup-kv/main_test.go` (`TestLog_formatText`) | pre-existing flake, unrelated to 05-06 | A full `make test` run failed once with `timed out after 10s waiting for the DuplicateNodeID event in node A's stderr` while the long `internal/app/repl` race suite ran in parallel. The test passes alone (0.11 s) and the `cmd/sup-kv` package passed its own targeted gate in the same plan. No load/logging code in this plan's diff touches the duplicate-node path. | 05-06 run: `go test -race -count=1 ./cmd/sup-kv/ -run '^TestLog_formatText$'` PASS; `go test -race -count=1 ./cmd/sup-kv/` (whole package) PASS earlier in the same plan. |
