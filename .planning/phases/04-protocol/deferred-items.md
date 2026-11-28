# Phase 4 — Deferred Items

Items observed during execution that are outside the owning plan's scope. Each
entry names what happened, the evidence, and where it is owned.

## Deferred Items

- Load-sensitive docs walkthrough flake (`TestExamplesDoc`)
  status: open
  **What:** One full-gate run (`go test -race -cover ./...`, wave-3 post-merge gate) failed in the
  "Read a missing key" block with an empty `grpcurl` capture (exit 1, no `NotFound`). The same
  package passes 3/3 in isolation (`go test -race -count=1 ./cmd/sup-kv/`), so the failure is
  load-induced, not a behavior change. The walkthrough uses fixed ports 28001/28002/28011/28012
  and shells out through `go tool grpcurl`; under full-suite compile load the tool invocation can
  stall or race the pair's startup. Owner: Phase 8 (performance/CI hardening) — verify with a
  serialized docs job or ephemeral-port rewriting if it recurs.

- `freeUDPPort` pick-then-bind race (inherited WR-04)
  status: open
  **What:** One wave-3 gate run failed with `probe rx bind: bind udp4 0.0.0.0:<port>: address
  already in use` — the known pick-then-bind race recorded in
  `.planning/phases/03-transport/deferred-items.md` (WR-04). Not caused by Phase 4; the new
  parallel stand tests increase the collision window. Owner: Phase 8.

- Startup-gate probe transient-answer race
  status: resolved
  **What:** `TestMain_startupGateEmptySelection` treated one transient health-RPC failure at the
  300 ms probe as a definitive status. Fixed in-phase by retrying the probe within a bound that
  stays inside the empty backstop (commit "test(04-04): retry a transient health-answer failure
  at the empty-gate probe"). Kept here as the record of the class: a health poll's transport
  error is "not yet", never a status.

- `TestCore_keyCountCostFlat` wall-clock ratio flake under full-suite load (inherited IN-03)
  status: open
  **What:** The 04-13 phase-gate `make test` run failed once with `HandleDatagram(DataRequest):
  1000 keys = 13165.0 ns/op, 50000 keys = 36879.0 ns/op, ratio 2.80` (> the 2.0 bound). The same
  test passed standalone (ratio 0.96 for DataRequest, 1.00 for DataSync) and the full gate
  passed on a clean rerun (`internal/app/repl` ok, 186.037 s, 93.0% coverage). The test carries
  `t.Parallel` (03-REVIEW IN-03), so the chaos corpus running concurrently contaminates its
  wall-clock measurement. Not caused by plan 04-13 (no Go source changed by this plan); no
  bound, test or expectation was touched. Owner: Phase 8 — drop `t.Parallel` or replace the
  wall-clock ratio with a deterministic operation-count assertion, per IN-03.
