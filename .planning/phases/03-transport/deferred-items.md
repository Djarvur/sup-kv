# Deferred Items — Phase 3 (Transport)

Out-of-scope discoveries logged during execution and deliberately not fixed.

| Discovered | Item | Why out of scope | Suggested owner |
|------------|------|------------------|-----------------|
| 03-08 | `cmd/sup-kv`'s `freeUDPPort` closes its throwaway socket before the test binds the chosen port, so parallel smoke tests can steal it (`TestLog_jsonDefault`: "probe rx bind: address already in use"). Reproduced twice during 03-08, passed on immediate retry. | Pre-existing harness pattern in files this plan only touched for its own knob; the race is inherent to pick-then-bind, not to the validation work. 03-08's new test no longer picks ports. | Any later plan touching `cmd/sup-kv/main_test.go` harness helpers; a bind-with-SO_REUSEADDR retry loop or a single shared port allocator would close it. |
