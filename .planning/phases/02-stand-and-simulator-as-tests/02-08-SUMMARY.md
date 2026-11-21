---
phase: 02-stand-and-simulator-as-tests
plan: 08
subsystem: testing
tags: [watch, send-error, metrics, prometheus, metrics-by-effect, parameter-effects, stand, test-12, test-14]

# Dependency graph
requires:
  - phase: 02-stand-and-simulator-as-tests
    provides: internal/stand two-process harness (02-02), Watch cut/backpressure branches and harness (existing suite), metrics adapter seams (phase 06)
provides:
  - Watch handler Send-error branch covered deterministically (snapshot loop and live relay) plus real-transport cleanup/isolation proof
  - Exact-value /metrics scenario on the stand after conflict(==1) + discard(==1) + retransmit(>=1)
  - Per-parameter effect tests for startup-gate, hello-interval, sync-response-rate, value-max-bytes, peer-timeout
  - Complete parameter-to-test map for every sup-kv configuration flag
affects: [phase-3-fixes, phase-4-protocol, verify-work, TEST-12, TEST-14]

# Actuals (#2632) — chars/4 over the realized diff; pairs with the plan's estimate.
actuals:
  tokens: 9637
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fake Watch server stream (embedded grpc.ServerStream + stub Send/Context) to pin the handler's Send-error mapping without racing the stream-context branch"
    - "Exact-value metrics asserts from the rendered /metrics text of a live node, keyed by the exact sample line (family + sorted labels)"
    - "Seeded-node restart removes the discovery pending-flush so a request-side rate limit actually paces recovery"

key-files:
  created:
    - internal/grpcapi/watch_senderr_test.go
    - internal/stand/stand_metrics_test.go
  modified: []

key-decisions:
  - "The Send-error branch is pinned twice: fake streams assert the handler returns the transport error unchanged from both Send sites (snapshot loop and relayEvent), while a real bufconn stream pins the transport-break cleanup contract"
  - "A real gRPC cancel is not used to force the Send branch: cancel and event delivery are both ready select cases, so the handler would race its context branch; the fake stream makes the failing Send the only outcome (plan's deterministic requirement)"
  - "Conflict exactness rides ordered writes, not races: B serves A's version before writing, so B's HLC has absorbed it and B's write strictly dominates (store conflict == 1 on A)"
  - "The metrics scenario pins --sync-interval=10s on both nodes: a proactive listing generated before B's displacing write would arrive as a rejected older version and count a second conflict on A"
  - "sync-response-rate effect needs a restarted seeded node: a freshly discovered peer receives the sender's pending records as an immediate broadcast (handleHello discovery flush), which bypasses the egress limiter; loading the keys from A's final dump leaves the pending set empty so B's paced DataRequests are the only path"
  - "Retransmit is asserted as a named lower bound (>= 1) plus a pre-kill baseline climb, never equality: the cycle count depends on scheduler-dependent Ack timing on real sockets; conflict and discard are constructed to single events and assert == 1"
  - "Per-parameter effects are named subtests (one per parameter) rather than a uniform table literal: each needs a different topology (isolated node, restarted seed, two-node pair) and observes a different surface (health timing, sent_total, recovery pacing, Set code, peer gauge)"

patterns-established:
  - "Send-error coverage: deterministic fake stream for branch mapping + real transport for the surrounding cleanup/liveness contract"
  - "Metrics-by-effect: construct single events, assert rendered == values; use named lower bounds where real-socket scheduling makes exactness impossible"
  - "Parameter-effect tests state what the production default would produce, so the assertion discriminates the configured value rather than merely observing traffic"

requirements-completed: [TEST-14, TEST-12]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Watch handler Send-error branch: a failed snapshot Send and a failed live relay Send both unsubscribe the subscriber and return the transport error unchanged"
    requirement: TEST-14
    verification:
      - kind: unit
        ref: "internal/grpcapi/watch_senderr_test.go#TestWatch_sendErrorDisconnectsSubscriber (snapshot_send_fails, live_relay_send_fails)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Broken-transport contract: a subscriber whose stream breaks mid-flight is shed without affecting a second subscriber, store liveness, health, or a fresh subscriber's snapshot-then-live sequence"
    requirement: TEST-14
    verification:
      - kind: integration
        ref: "internal/grpcapi/watch_senderr_test.go#TestWatch_sendErrorDisconnectsSubscriber (broken_transport_leaves_peers_and_store_untouched)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Exact /metrics values on a live stand node after a constructed scenario: supkv_merges_conflicts_total == 1, supkv_datagrams_discarded_total{class=\"bad_crc\"} == 1, supkv_replication_datagrams_retransmitted_total >= 1"
    requirement: TEST-12
    verification:
      - kind: integration
        ref: "internal/stand/stand_metrics_test.go#TestStand_metricsByEffect"
        status: pass
    human_judgment: false
  - id: D4
    description: "Per-parameter observed effects: startup-gate open timing, hello cadence, sync-response-rate recovery pacing, value-max-bytes boundary, peer-timeout eviction"
    requirement: TEST-12
    verification:
      - kind: integration
        ref: "internal/stand/stand_metrics_test.go#TestConfig_parametersByEffect"
        status: pass
    human_judgment: false

duration: 16min
completed: 2026-11-21
status: complete
---

# Phase 02 Plan 08: Watch Send Errors and Metrics by Effect Summary

**Worked Watch's last uncovered branch (failed Send unsubscribes and returns the transport error) and pinned /metrics/configuration by observed effect: exactly one conflict, exactly one bad-CRC discard, at least one retransmit, plus one effect test per stand-observable parameter.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-11-21T11:18:10Z
- **Completed:** 2026-11-21T11:34:33Z
- **Tasks:** 2
- **Files created:** 2 (no production code changed)

## Accomplishments

- `internal/grpcapi/watch_senderr_test.go` covers both `Send` sites in `watch.go` (snapshot loop and `relayEvent`) with a fake `WatchServer` stream, proving the handler unsubscribes and returns the transport error unchanged — the one branch the existing Watch suite left uncovered.
- The real-transport subtest pins the contract the branch exists for: subscriber 1's stream breaks mid-flight, subscriber 2 keeps receiving, writes keep confirming, health stays SERVING, and a fresh subscriber re-snapshots the full state and receives a new live change (hub table not corrupted).
- The same test re-asserts the existing cut ordering and shed policy (ResourceExhausted via the hub-ingress overflow flavor), extending `TestWatch_slowConsumer` rather than duplicating it.
- `internal/stand/stand_metrics_test.go` drives conflict + discard + retransmit on a live two-process stand with `--metrics-addr` and asserts the exact rendered `/metrics` samples over HTTP.
- Five per-parameter effect subtests plus a complete parameter→test map (below): every configuration flag now has an observed effect or a named unit test.

## Task Commits

Each task was committed atomically:

1. **Task 1: Watch Send error branch** — `3e2696d` (test)
2. **Task 2: metrics-by-effect on the stand + per-parameter effect tables** — `8e9dae8` (test)

**Plan metadata:** docs commit created after this SUMMARY (Step: final metadata commit).

_No production code changed — both deliverables are test-only pins of current behavior._

## Files Created/Modified

- `internal/grpcapi/watch_senderr_test.go` — `TestWatch_sendErrorDisconnectsSubscriber` with four subtests: snapshot Send failure (fake stream), live relay Send failure (fake stream), broken transport (real bufconn streams: isolation, liveness, fresh resubscribe), and cut/shed retention.
- `internal/stand/stand_metrics_test.go` — `TestStand_metricsByEffect` (exact `/metrics` scenario, 14.8/14.4 reference links, retransmit lower-bound rationale) and `TestConfig_parametersByEffect` (startup-gate, hello-interval, sync-response-rate, value-max-bytes, peer-timeout), plus the scrape/sample/corrupt-datagram helpers.

## Decisions Made

- **Send-error determinism split.** Real gRPC cannot deterministically lose one `Send`: cancelling the client makes the handler's `ctx.Done` case ready alongside the event case, so a select race decides. The fake stream (embedded `grpc.ServerStream`, stub `Send`/`Context`) makes the failing `Send` the only outcome and pins the exact mapping; the real-transport subtest then pins the surrounding cleanup/isolation contract.
- **Conflict constructed by ordering, not racing.** Node B serves A's first value before writing, so B's HLC has absorbed A's version and B's write strictly dominates → exactly one displacement conflict on A. Both nodes run `--sync-interval=10s` so a stale proactive listing cannot arrive as a rejected older version and add a second conflict.
- **Retransmit asserted as a lower bound.** Conflict (`== 1`) and discard (`== 1`) are single constructed events; the retransmit cycle count depends on scheduler-dependent Ack timing on real sockets. The test asserts `>= 1` with the rationale in the doc comment, and additionally requires the counter to climb above the pre-kill baseline so the cycle is attributable to the stopped peer.
- **sync-response-rate effect requires a restarted seed.** A freshly discovered peer receives the sender's pending records as an immediate broadcast (`handleHello` discovery flush), which bypasses the egress limiter. The subtest seeds A, stops it cleanly so the dump holds the records, restarts it (empty pending set), then starts B with `--sync-response-rate=10`; B's paced `DataRequest` drain makes recovery take the paced bound.
- **Per-parameter cases are named subtests, not a table literal.** Each parameter needs a different topology and observes a different surface; the one-name-per-parameter contract of TEST-12 is preserved 1:1.

## Parameter Coverage Map (TEST-12)

Stand-observable parameters (this plan):

| Flag | Observed effect | Test |
|------|-----------------|------|
| `--metrics-addr` | opt-in listener serves `/metrics`; unset means none | `TestStand_metricsByEffect` (this plan); `TestMetrics_optIn`, `TestMetrics_noDefaultListener` (cmd/sup-kv) |
| `--startup-gate` | health stays NOT_SERVING until the configured backstop | `TestConfig_parametersByEffect/startup_gate_delays_gate_open`; `TestGate_opensOnTimeout` |
| `--hello-interval` | hello sends over a measured window (3+ at 50ms; the 1s default yields at most 1) | `TestConfig_parametersByEffect/hello_interval_sets_hello_cadence` |
| `--sync-response-rate` | empty-node recovery pacing bounded below by N/2/rate | `TestConfig_parametersByEffect/sync_response_rate_paces_recovery`; `TestStand_emptyDiskRecovery`; `TestSync_rateLimit`, `TestSync_responseRateClamp` |
| `--value-max-bytes` | Set accepted at the limit, InvalidArgument above it | `TestConfig_parametersByEffect/value_max_bytes_rejects_oversize`; `TestStore_valueLimit` |
| `--peer-timeout` | silent peer evicted from the peer gauge after the configured timeout | `TestConfig_parametersByEffect/peer_timeout_evicts_silent_peer`; `TestPeerTable_lifecycle` |

Parameters observed by existing unit/exec tests instead (no stand-level effect unique to the flag):

| Flag | Observing test |
|------|----------------|
| `--node-id` | `TestScenario_discardClasses` (`duplicate_node_id` subtest), internal/udprepl |
| `--bind-addr`, `--broadcast-addr` | every stand scenario's asymmetric topology (`newStandPair`/`nodeArgs`); `TestSmoke_twoNodesUDP`, internal/udprepl |
| `--dump-path` | `TestMain_lifecycle`; stand dump assertions (`readFinalDump`, `TestStand_deleteReplication`) |
| `--grpc-addr` | `TestMain_precedence` (flag beats env), `TestMain_lifecycle` |
| `--wait-timeout` | `TestWait_timeout`, `TestWait_perRequestTimeout` (internal/grpcapi) |
| `--sync-chunk-bytes` | `TestSync_chunking` (internal/app/repl), `TestSync_chunkWireBudget` (internal/udprepl) |
| `--ack-timeout`, `--ack-timeout-max`, `--retransmit-interval` | `TestRetransmit_counter`, `TestRetransmit_backoffLadder` (internal/app/repl); also exercised by this plan's retransmit scenario |
| `--sync-interval` | `TestSync_triggers` (internal/app/repl) |
| `--dump-debounce`, `--dump-max-wait` | `TestEngine_confirmDebounceTiming` (covers both: debounce and the max-wait bound) |
| `--drift-warn` | `TestMergeStats_counts` (internal/app/repl) |
| `SUPKV_METRICS_CERT/KEY/CLIENT_CA` (env-only) | `TestMetrics_incompleteCertSet`, `TestMetricsTLS_required` (cmd/sup-kv) |

## Deviations from Plan

None — plan executed exactly as written. No production code was changed; `watch.go`'s Send-error semantics were pinned as implemented (both sites return the transport error unchanged after unsubscribing).

## Issues Encountered

- **Fake-stream test race (fixed during Task 1).** The first draft set `sentOne = nil` inside `Send`; the test read the field in a select expression → race detector failure. Replaced with a buffered one-shot channel signal (non-blocking send), no field mutation.
- **Initial sync-response-rate subtest measured 5–17ms recovery.** Root cause: `handleHello`'s discovery flush broadcasts the seeded node's pending records immediately, and a live seeded node keeps them pending; the limiter never paces. Fixed by restarting the seeded node from its confirmed dump (empty pending set) before starting the empty node. This is documented in the test and the decisions above; the earlier focused measurement caught it before commit.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- TEST-14's Watch branch coverage is complete; no further Watch changes are needed before the Phase 3-5 fixes.
- TEST-12's metric definitions and configuration parameters are now pinned by observed effect; the parameter coverage map gives the verifier a single index for every flag.
- All verification commands passed on the committed tree: `go test -race ./internal/grpcapi/ -run '^TestWatch'`, `go test -race ./internal/stand/ -run 'TestStand_metricsByEffect|TestConfig_parametersByEffect'`, `make lint`.
- Note for future plan estimates: realized changed-code size was ~9.6k tokens (chars/4 over the two new files) against a 100k estimate — the confidence-low estimate was conservative.

---
*Phase: 02-stand-and-simulator-as-tests*
*Completed: 2026-11-21*

## Self-Check: PASSED

- `internal/grpcapi/watch_senderr_test.go`: FOUND
- `internal/stand/stand_metrics_test.go`: FOUND
- Commit `3e2696d` (Task 1): FOUND (ancestor of HEAD)
- Commit `8e9dae8` (Task 2): FOUND (ancestor of HEAD)
- Plan verification re-run on the committed tree: `go test -race ./internal/grpcapi/ -run '^TestWatch'` PASS; `go test -race ./internal/stand/ -run 'TestStand_metricsByEffect|TestConfig_parametersByEffect'` PASS; `make lint` PASS.
- Measured plan commits: 2 (`a4ca045..8e9dae8`).
