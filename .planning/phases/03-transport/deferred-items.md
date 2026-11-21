# Deferred Items — Phase 3 (Transport)

Out-of-scope discoveries logged during execution and deliberately not fixed.

| Discovered | Item | Why out of scope | Suggested owner |
|------------|------|------------------|-----------------|
| 03-08 | `cmd/sup-kv`'s `freeUDPPort` closes its throwaway socket before the test binds the chosen port, so parallel smoke tests can steal it (`TestLog_jsonDefault`: "probe rx bind: address already in use"). Reproduced twice during 03-08, passed on immediate retry. | Pre-existing harness pattern in files this plan only touched for its own knob; the race is inherent to pick-then-bind, not to the validation work. 03-08's new test no longer picks ports. | Any later plan touching `cmd/sup-kv/main_test.go` harness helpers; a bind-with-SO_REUSEADDR retry loop or a single shared port allocator would close it. |
| 03-10 | The same pick-then-bind race recurred once in a combined `go test -race ./internal/udprepl/ ./internal/app/repl/ ./cmd/sup-kv/` run (`TestLog_formatText`: "probe rx bind: 0.0.0.0:48099: address already in use"); three immediate `cmd/sup-kv` reruns and the final full `go test -race ./...` were green. 03-10 adds two tests that hold `:0`-picked UDP ports while running (`TestMain_bindAddrWarn`, `TestMain_duplicateBind`), slightly widening the existing race window without introducing a new class. | The harness fix is outside this plan's read-loop/socket-policy scope and the race predates it (03-08 row above). | The same owner as the 03-08 row: a shared port allocator or bind-with-retry in the `cmd/sup-kv` harness. |
