# Deferred Items — Phase 05

Out-of-scope findings from plan execution. Not fixed here by the deviation
scope boundary; carried for the phase verifier and later plans.

| Item | File | Class | Description | Evidence |
|------|------|-------|-------------|----------|
| UDP pick-to-bind flake | `cmd/sup-kv/main_test.go` (`freeUDPPort`, `TestLog_formatText`) | pre-existing flake, unrelated to 05-04 | A full `go test -race ./...` run failed once with `probe rx bind: bind udp4 0.0.0.0:<ephemeral>: address already in use` while other UDP-using packages ran in parallel. Re-running the test alone passed twice. This is the documented pick-to-bind flake class (Makefile `race-soak` notes, class 1). | 05-04 run: `TestLog_formatText` failed on two different ports in two full/package runs; `go test -race -count=1 ./cmd/sup-kv/ -run '^TestLog_formatText$'` passed on two consecutive retries. |
